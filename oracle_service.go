//go:build !js && !wasm

package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/types"
)

type OracleService struct{}

var (
	ErrInvalidSignature = errors.New("security exception: cryptographic verification failed")
	ErrExpiredNonce     = errors.New("security exception: payload nonce lifetime exceeded")
	ErrReplayedNonce    = errors.New("security exception: nonce vector has already been consumed")
)

const (
	cardCacheName = "card_cache.json"

	// envoiAPIBase is the public Envoi naming-service (Voi human-readable addresses, like ENS).
	// Overridable via ENVOI_API_URL for custom gateways / testing.
	envoiAPIBase = "https://api.envoi.sh"

	// SecureCommandType enums for signed payloads
	CommandMarketBuy  uint8 = 1
	CommandMarketSell uint8 = 2
)

// MarketActionPayload represents the structured data signed by the user's wallet.
type MarketActionPayload struct {
	CommandType uint8  `json:"command_type"`
	EntityID    uint64 `json:"entity_id"`
	ShareCount  uint64 `json:"share_count"`
	MaxCost     uint64 `json:"max_cost"`
	Nonce       uint64 `json:"nonce"`
	Timestamp   int64  `json:"timestamp"`
}

// NewVerificationHook initializes the security challenge registry.
func NewVerificationHook() *VerificationHook {
	return &VerificationHook{
		ActiveNonces:   make(map[uint64]time.Time),
		ConsumedNonces: make(map[uint64]bool),
	}
}

// GenerateNextNonce reserves a valid transactional entry block with a 2-minute TTL.
func (vh *VerificationHook) GenerateNextNonce(nonceID uint64) {
	vh.Mu.Lock()
	defer vh.Mu.Unlock()
	vh.ActiveNonces[nonceID] = time.Now().Add(2 * time.Minute)
}

// CompilePayloadDigest hashes the structural payload into a deterministic 32-byte array.
// PILLAR 3: Deterministic Data Integrity.
func CompilePayloadDigest(payload MarketActionPayload) []byte {
	hasher := sha256.New()

	// Write raw continuous binary fields to prevent padding manipulation
	hasher.Write([]byte{payload.CommandType})

	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, payload.EntityID)
	hasher.Write(buf)

	binary.BigEndian.PutUint64(buf, payload.ShareCount)
	hasher.Write(buf)

	binary.BigEndian.PutUint64(buf, payload.MaxCost)
	hasher.Write(buf)

	binary.BigEndian.PutUint64(buf, payload.Nonce)
	hasher.Write(buf)

	binary.BigEndian.PutUint64(buf, uint64(payload.Timestamp))
	hasher.Write(buf)

	return hasher.Sum(nil)
}

// VerifyIncomingCommand validates wallet authenticity before exposing engine mutations.
func (s *OracleService) VerifyIncomingCommand(l *Lobby, publicKey ed25519.PublicKey, payload MarketActionPayload, signature []byte) error {
	l.verificationHook.Mu.Lock()
	defer l.verificationHook.Mu.Unlock()

	// 1. Replay Protection Guardrail
	if l.verificationHook.ConsumedNonces[payload.Nonce] {
		return ErrReplayedNonce
	}

	// 2. Nonce Validity & Expiration Checks
	expiration, exists := l.verificationHook.ActiveNonces[payload.Nonce]
	if !exists {
		return ErrExpiredNonce
	}
	if time.Now().After(expiration) {
		delete(l.verificationHook.ActiveNonces, payload.Nonce)
		return ErrExpiredNonce
	}

	// 3. Compile and Evaluate Cryptographic Verification Vector
	digest := CompilePayloadDigest(payload)
	if !ed25519.Verify(publicKey, digest, signature) {
		return ErrInvalidSignature
	}

	// 4. Commit state changes to eliminate reuse vectors
	l.verificationHook.ConsumedNonces[payload.Nonce] = true
	delete(l.verificationHook.ActiveNonces, payload.Nonce)

	return nil
}

// indexerGet is THE ONE INDEXER TRANSPORT for the process.
//
// The project reads the chain through several base URLs, and this is the only
// place that decides which one answers a path and how a failure is retried.
// Two methods used to hold private copies of this loop (`OracleService.
// IndexerRequest` and `Lobby.indexerRequest`); two copies of a transport is two
// chances for the oracle readers and the checkpoint readers to disagree about
// retries, deadlines or status handling, so both now delegate here.
//
// PILLAR 4: RPC Failover. Cycles configured endpoints on 429 and 5xx, and —
// see below — on a 404 that is really a ROUTING answer.
//
// A 404 IS A ROUTING ANSWER, NOT A RESULT. A base that does not serve a path
// answers 404 for EVERY request to it, so returning that 404 immediately ended
// the chain at the first base and the remaining bases were never asked. A 404
// therefore advances to the NEXT base and, because a routing answer cannot
// change between two identical requests, the same URL is not retried. If EVERY
// base answered 404 the most recent of those responses is returned unchanged,
// which preserves the two callers that read a 404 as "not found yet"
// (VerifyBuyInTransaction's Algorand branch, the account-existence probe) on a
// single-base configuration.
// indexerMaxResponseBytes caps how much of an indexer response is held in memory.
// It is generous — a record read returns many chunk notes — while still bounded.
const indexerMaxResponseBytes = 32 << 20

// indexerBufferedResponse reads a response's body INTO MEMORY and hands the response
// back carrying those bytes.
//
// WHY THIS EXISTS (measured 2026-09-19, found by the record reader's end-to-end
// test): the attempt context is cancelled as soon as `Do` returns, but the RESPONSE
// is handed to the caller, which decodes it AFTERWARD. A Body still bound to a
// cancelled attempt context answers `context canceled` for anything the socket had
// not already buffered — so this transport worked for tiny bodies and 404s and FAILED
// for a real payload. The record reader returns tens of KB, so reading the body HERE,
// while the attempt context is still alive, is what makes a real read possible.
func indexerBufferedResponse(resp *http.Response) (*http.Response, error) {
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, indexerMaxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("reading the %d response body failed: %w", resp.StatusCode, err)
	}
	if len(body) > indexerMaxResponseBytes {
		return nil, fmt.Errorf("the response body is above the %d byte read cap", indexerMaxResponseBytes)
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return resp, nil
}

func indexerGet(bases []string, path string) (*http.Response, error) {
	// PILLAR 4: Multi-failover timeout hardening. An outer context bounds the
	// total duration of the whole failover/retry sequence.
	outerCtx, outerCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer outerCancel()

	var lastErr error
	var notFound *http.Response // the most recent "this base does not serve the path" answer
	for _, baseURL := range bases {
		url := baseURL + path
		for i := 0; i < 3; i++ {
			// Derive attempt context from outer context to respect total deadline
			attemptCtx, attemptCancel := context.WithTimeout(outerCtx, indexerTimeout)
			req, reqErr := http.NewRequestWithContext(attemptCtx, "GET", url, nil)
			if reqErr != nil {
				attemptCancel()
				lastErr = reqErr
				break // a malformed URL cannot be fixed by retrying it
			}
			resp, err := http.DefaultClient.Do(req)

			if err != nil {
				attemptCancel()
				lastErr = err
				time.Sleep(500 * time.Millisecond)
				continue
			}
			if resp.StatusCode == http.StatusTooManyRequests {
				attemptCancel()
				resp.Body.Close()
				lastErr = fmt.Errorf("rate-limited (429) at %s", baseURL)
				time.Sleep(time.Duration(i+1) * 1 * time.Second)
				continue
			}
			if resp.StatusCode >= 500 {
				attemptCancel()
				resp.Body.Close()
				lastErr = fmt.Errorf("server error %d at %s", resp.StatusCode, baseURL)
				continue
			}
			if resp.StatusCode == http.StatusNotFound {
				// A 404 is RETURNED to its caller, so its body is buffered as well: the
				// attempt context dies with this frame.
				buffered, bufErr := indexerBufferedResponse(resp)
				attemptCancel()
				if bufErr != nil {
					lastErr = bufErr
					continue
				}
				if notFound != nil {
					notFound.Body.Close()
				}
				notFound = buffered
				break // routing answer: ask the next base, do not repeat this one
			}

			// THE BODY IS READ BEFORE THE CANCELLATION (indexerBufferedResponse). The
			// response goes back to a caller that decodes it after this function
			// returns, and a Body bound to a cancelled attempt context answers
			// `context canceled` for anything the socket had not already buffered.
			buffered, bufErr := indexerBufferedResponse(resp)
			attemptCancel()
			if bufErr != nil {
				lastErr = bufErr
				continue
			}
			return buffered, nil
		}
	}
	if notFound != nil {
		return notFound, nil
	}
	if lastErr == nil {
		return nil, fmt.Errorf("indexer request failed: no indexer base is configured for this network")
	}
	return nil, fmt.Errorf("indexer request failed after cycling endpoints: %w", lastErr)
}

// IndexerRequest executes an HTTP GET for an indexer path through the ONE
// transport (indexerGet). The *Lobby parameter is retained because every call
// site already passes it; the transport itself needs no lobby state.
func (s *OracleService) IndexerRequest(l *Lobby, cfg NetworkConfig, path string) (*http.Response, error) {
	return indexerGet(cfg.IndexerURLs, path)
}

func (s *OracleService) GetVerifiedCards(l *Lobby, wallet string, tokenIDs []int, networkName string) (map[int]ServerCard, error) {
	results := make(map[int]ServerCard)
	var toFetch []int

	l.mutex.RLock()
	for _, id := range tokenIDs {
		if card, exists := l.inventory[id]; exists && time.Since(card.LastUpdated) < 1*time.Hour {
			results[id] = card
		} else {
			toFetch = append(toFetch, id)
		}
	}
	l.mutex.RUnlock()

	// DISCOVERY MODE: If no IDs provided, iterate through linked wallets to find all owned cards
	if len(tokenIDs) == 0 && wallet != "" {
		log.Printf("[ORACLE] Discovery started for %s across linked chains.\n", wallet)

		// 1. Compile list of wallets and networks
		type target struct{ addr, network string }
		targets := []target{{wallet, networkName}}

		l.mutex.RLock()
		if linkInfo, ok := l.linkedWallets[wallet]; ok {
			for _, lw := range linkInfo.Linked {
				netKey := l.mapChainToNetworkName(lw.Chain)
				if netKey != "" && lw.Verified {
					targets = append(targets, target{lw.Address, netKey})
				}
			}
		}
		l.mutex.RUnlock()

		// 2. Query each target network (Multi-Chain Discovery)
		for _, t := range targets {
			l.mutex.RLock()
			cfg, ok := l.availableNetworks[t.network]
			l.mutex.RUnlock()
			if !ok {
				continue
			}

			if strings.Contains(cfg.ChainID, "algorand") {
				// PATH 1: Standard ASA Scan (ARC-19/ARC-69)
				// Query account info to find all held assets, regardless of contract status.
				accResp, err := s.IndexerRequest(l, cfg, fmt.Sprintf("/v2/accounts/%s", t.addr))
				if err == nil && accResp.StatusCode == http.StatusOK {
					var accRes struct {
						Account struct {
							Assets []struct {
								AssetID uint64 `json:"asset-id"`
								Deleted bool   `json:"deleted"`
								Amount  uint64 `json:"amount"`
							} `json:"assets"`
						} `json:"account"`
					}
					if json.NewDecoder(accResp.Body).Decode(&accRes) == nil {
						for _, as := range accRes.Account.Assets {
							if as.Deleted || as.Amount == 0 {
								continue
							}
							// Check cache first to avoid re-dispatching known assets
							l.mutex.RLock()
							_, exists := l.inventory[int(as.AssetID)]
							l.mutex.RUnlock()
							if exists {
								continue
							}

							// Use the Dispatcher to resolve ARC-19 or ARC-69 metadata
							meta, std, err := s.MetadataDispatcher(l, t.network, int(as.AssetID))
							if err == nil && meta != nil {
								newCard := ServerCard{
									ID:            int(as.AssetID),
									Name:          meta.Name,
									Image:         meta.Image,
									Power:         [4]int{cfg.PowerBase, 10, cfg.PowerBase, 10},
									LastUpdated:   time.Now(),
									MetadataValid: true,
								}
								l.mutex.Lock()
								l.inventory[int(as.AssetID)] = newCard
								l.mutex.Unlock()
								results[int(as.AssetID)] = newCard
								log.Printf("[ORACLE] Discovered %s asset via account scan: %d\n", std, as.AssetID)
							}
						}
					}
					accResp.Body.Close()
				}

				// PATH 2: ARC-72 Collection Scan
				// Keep existing logic to find tokens within a specific smart contract collection.
				log.Printf("[ORACLE] Syncing tokens for %s on %s...\n", t.addr, t.network)
				resp, err := s.IndexerRequest(l, cfg, fmt.Sprintf("/tokens?owner=%s", t.addr))
				if err == nil && resp.StatusCode == http.StatusOK {
					var res struct {
						Tokens []struct {
							TokenID  int    `json:"tokenId"`
							Metadata string `json:"metadata"`
						} `json:"tokens"`
					}
					if json.NewDecoder(resp.Body).Decode(&res) == nil {
						for _, tok := range res.Tokens {
							var meta *ARC72Metadata
							var std string

							// Optimization: Try parsing the bulk metadata first (common for ARC-72 indexers)
							if tok.Metadata != "" {
								var m ARC72Metadata
								if json.Unmarshal([]byte(tok.Metadata), &m) == nil {
									meta = &m
									std = "ARC-72"
								}
							}

							// If bulk metadata is missing, use the Dispatcher for deep discovery (ARC-19/69)
							if meta == nil {
								m, s, err := s.MetadataDispatcher(l, t.network, tok.TokenID)
								if err == nil {
									meta = m
									std = s
								}
							}

							if meta != nil {
								newCard := ServerCard{
									ID:            tok.TokenID,
									Name:          meta.Name,
									Image:         meta.Image,
									Power:         [4]int{cfg.PowerBase, 10, cfg.PowerBase, 10},
									LastUpdated:   time.Now(),
									MetadataValid: true,
								}
								l.mutex.Lock()
								l.inventory[tok.TokenID] = newCard
								l.mutex.Unlock()
								results[tok.TokenID] = newCard
								log.Printf("[ORACLE] Discovered %s token during owner scan: %d\n", std, tok.TokenID)
							}
						}
					}
					resp.Body.Close()
				}
			} else if strings.HasPrefix(cfg.ChainID, "eip155") {
				// EVM Discovery logic: Query Etherscan NFT transfer history for ownership patterns
				log.Printf("[ORACLE] Syncing EVM tokens for %s on %s...\n", t.addr, t.network)
				url := fmt.Sprintf("%s/api?module=account&action=tokennfttx&contractaddress=%s&address=%s&sort=desc",
					cfg.IndexerURLs[0], cfg.AssetID, t.addr)

				var resp *http.Response
				var err error
				for i := 0; i < 3; i++ {
					ctx, cancel := context.WithTimeout(context.Background(), indexerTimeout)
					req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
					resp, err = http.DefaultClient.Do(req)
					cancel()
					if err != nil {
						if i < 2 {
							time.Sleep(500 * time.Millisecond)
							continue
						}
						break
					}
					if resp.StatusCode == http.StatusTooManyRequests {
						resp.Body.Close()
						if i < 2 {
							time.Sleep(time.Duration(i+1) * 1 * time.Second)
							continue
						}
						break
					}
					break
				}

				if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
					var evmRes struct {
						Status string `json:"status"`
						Result []struct {
							TokenID string `json:"tokenID"`
						} `json:"result"`
					}
					if json.NewDecoder(resp.Body).Decode(&evmRes) == nil && evmRes.Status == "1" {
						for _, tok := range evmRes.Result {
							id, err := strconv.Atoi(tok.TokenID)
							if err != nil {
								continue
							}

							// Basic discovery entry: Metadata will be verified during specific card refresh
							newCard := ServerCard{
								ID:            id,
								Name:          fmt.Sprintf("%s Artifact #%d", cfg.NetworkName, id),
								Image:         "Cards/placeholder.webp",
								Power:         [4]int{cfg.PowerBase, 20, cfg.PowerBase, 20},
								LastUpdated:   time.Now(),
								MetadataValid: false,
							}
							l.mutex.Lock()
							l.inventory[id] = newCard
							l.mutex.Unlock()
							results[id] = newCard
						}
					}
					resp.Body.Close()
				}
			}
		}

		return results, nil
	}

	if len(toFetch) == 0 && len(tokenIDs) > 0 {
		return results, nil
	}

	l.mutex.RLock() // Use RLock for reading availableNetworks
	netConfig, ok := l.availableNetworks[networkName]
	l.mutex.RUnlock()
	if !ok {
		return nil, fmt.Errorf("network not found: %s", networkName)
	}

	// Cross-Chain Safety Guard: Ensure we only hit Algorand indexers for Algorand-based chains
	if !strings.Contains(netConfig.ChainID, "algorand") {
		return s.GetVerifiedCardsCrossChain(l, tokenIDs, netConfig)
	}

	// PILLAR 3: Multi-Standard Discovery.
	// We iterate through missing tokens and utilize the MetadataDispatcher to identify
	// and fetch standard-compliant metadata (ARC-72, ARC-19, or ARC-69).
	for _, id := range toFetch {
		meta, standard, err := s.MetadataDispatcher(l, networkName, id)
		if err != nil {
			log.Printf("[ORACLE] Metadata resolution failed for %s #%d: %v\n", networkName, id, err)
			// Cache a placeholder to prevent repeated hits for invalid assets
			l.mutex.Lock()
			l.inventory[id] = ServerCard{ID: id, Name: "Unknown Artifact", LastUpdated: time.Now()}
			l.mutex.Unlock()
			continue
		}

		newCard := ServerCard{
			ID: id, Name: meta.Name, Image: meta.Image,
			Power:         [4]int{netConfig.PowerBase, 10, netConfig.PowerBase, 10},
			LastUpdated:   time.Now(),
			MetadataValid: true,
		}

		l.mutex.Lock()
		l.inventory[id] = newCard
		l.mutex.Unlock()
		results[id] = newCard

		log.Printf("[ORACLE] Ingested %s card: %s (#%d)\n", standard, meta.Name, id)
	}
	return results, nil
}

// GetVerifiedCardsCrossChain handles metadata retrieval for non-Algorand networks (EVM, Solana, etc).
func (s *OracleService) GetVerifiedCardsCrossChain(l *Lobby, tokenIDs []int, cfg NetworkConfig) (map[int]ServerCard, error) {
	results := make(map[int]ServerCard)

	// Identify Network Type
	isEVM := strings.HasPrefix(cfg.ChainID, "eip155")
	isSolana := strings.HasPrefix(cfg.ChainID, "solana")

	for _, id := range tokenIDs {
		var newCard ServerCard
		foundOnChain := false

		if isEVM {
			// EVM Metadata Fetch (Ethereum / Polygon)
			// Expected IndexerURL format for EVM: https://api.etherscan.io or similar
			// Using a generic ERC721 metadata discovery pattern
			path := fmt.Sprintf("/api?module=token&action=tokenid_metadata&contractaddress=%s&tokenid=%d",
				cfg.AssetID, id)

			resp, err := s.IndexerRequest(l, cfg, path)

			if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
				var evmRes struct {
					Status  string `json:"status"`
					Result  string `json:"result"` // Usually JSON string of metadata
					Message string `json:"message"`
				}
				if json.NewDecoder(resp.Body).Decode(&evmRes) == nil && evmRes.Status == "1" {
					var meta ARC72Metadata // Reuse AVM structure as it maps closely to OpenSea/EVM standards
					if json.Unmarshal([]byte(evmRes.Result), &meta) == nil {
						newCard = ServerCard{
							ID: id, Name: meta.Name, Image: meta.Image,
							Power:         [4]int{cfg.PowerBase, 20, cfg.PowerBase, 20}, // EVM artifacts get a baseline power boost
							LastUpdated:   time.Now(),
							MetadataValid: true,
						}
						foundOnChain = true
					}
				}
				resp.Body.Close()
			} else if resp != nil {
				log.Printf("[ORACLE] EVM Metadata fetch for %s #%d returned status %d", cfg.NetworkName, id, resp.StatusCode)
				resp.Body.Close()
			}
		} else if isSolana {
			// Solana Metadata Fetch (Metaplex Digital Asset Standard - DAS) via NodeURL

			// NOTE: Solana uses Mint Addresses (strings). If the game passes an int ID,
			// we assume it is a CRC32 or similar hash of the mint address, or we use the
			// cfg.AssetID as the target if 'id' is a specific token index in a collection.
			targetMint := cfg.AssetID

			// Construct RPC request body for DAS getAsset
			rpcRequestBody := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1, // Arbitrary ID
				"method":  "getAsset",
				"params": map[string]interface{}{
					"id": targetMint,
				},
			}
			jsonBody, _ := json.Marshal(rpcRequestBody)

			url := cfg.NodeURLs[0] // DAS API is typically accessed via the NodeURL (RPC endpoint)

			var resp *http.Response
			var err error
			for i := 0; i < 3; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), indexerTimeout)
				req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				resp, err = http.DefaultClient.Do(req)
				cancel()
				if err != nil {
					if i < 2 {
						time.Sleep(500 * time.Millisecond)
						continue
					}
					break
				}
				if resp.StatusCode == http.StatusTooManyRequests {
					resp.Body.Close()
					if i < 2 {
						time.Sleep(time.Duration(i+1) * 1 * time.Second)
						continue
					}
					break
				}
				break
			}

			if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
				var dasRes struct {
					Result struct {
						Content struct {
							Links struct {
								Image string `json:"image"`
							} `json:"links"`
							Metadata struct {
								Name        string              `json:"name"`
								Image       string              `json:"image"`
								Description string              `json:"description"`
								Attributes  []MetadataAttribute `json:"attributes"`
							} `json:"metadata"`
							Files []struct {
								URI string `json:"uri"`
							} `json:"files"`
						} `json:"content"`
					} `json:"result"`
				}

				if json.NewDecoder(resp.Body).Decode(&dasRes) == nil && dasRes.Result.Content.Metadata.Name != "" {
					imgURL := dasRes.Result.Content.Links.Image
					if imgURL == "" && len(dasRes.Result.Content.Files) > 0 {
						imgURL = dasRes.Result.Content.Files[0].URI
					}

					newCard = ServerCard{
						ID:            id,
						Name:          dasRes.Result.Content.Metadata.Name,
						Image:         imgURL,
						Power:         [4]int{cfg.PowerBase, 20, cfg.PowerBase, 20}, // Apply network's base power
						LastUpdated:   time.Now(),
						MetadataValid: true,
					}

					// PILLAR 4: RPG Attribute Ingestion.
					// Scan Solana traits for keywords to populate Arena mechanics.
					for _, attr := range dasRes.Result.Content.Metadata.Attributes {
						valStr := fmt.Sprintf("%v", attr.Value)
						switch strings.ToLower(attr.TraitType) {
						case "mood", "element":
							newCard.Mood = valStr
						case "power", "strength":
							if p, err := strconv.Atoi(valStr); err == nil {
								newCard.Power[0] += p // Add trait bonus to base
							}
						}
					}
					foundOnChain = true
				} else {
					log.Printf("[ORACLE] Solana DAS 'getAsset' response parsing failed or no metadata for %s #%d. Error: %v", cfg.NetworkName, id, err)
				}
				resp.Body.Close()
			} else if resp != nil {
				log.Printf("[ORACLE] Solana DAS 'getAsset' request failed for %s #%d. Status: %d", cfg.NetworkName, id, resp.StatusCode)
				resp.Body.Close()
			} else {
				log.Printf("[ORACLE] Solana DAS 'getAsset' connection failed for %s #%d: %v", cfg.NetworkName, id, err)
			}

			if !foundOnChain {
				log.Printf("[ORACLE] Solana DAS fetch for %s #%d failed to retrieve valid metadata. Using placeholder card.", cfg.NetworkName, id)
				newCard = ServerCard{
					ID: id, Name: fmt.Sprintf("%s NFT #%d (DAS Failed)", cfg.NetworkName, id), Image: "Cards/solana_placeholder.webp",
					Power:         [4]int{cfg.PowerBase, cfg.PowerBase, cfg.PowerBase, cfg.PowerBase},
					LastUpdated:   time.Now(),
					MetadataValid: false,
				}
			}
		}

		// Fallback/Default for unhandled or failed fetches
		if !foundOnChain {
			log.Printf("[ORACLE] Metadata fetch failed or unhandled for %s #%d. Using placeholders.\n", cfg.NetworkName, id)
			newCard = ServerCard{
				ID: id, Name: fmt.Sprintf("%s Artifact #%d", cfg.NetworkName, id), Image: "Cards/placeholder.webp",
				Power:       [4]int{cfg.PowerBase, cfg.PowerBase, cfg.PowerBase, cfg.PowerBase},
				LastUpdated: time.Now(),
			}
		}

		l.mutex.Lock()
		l.inventory[id] = newCard
		l.mutex.Unlock()
		results[id] = newCard
	}
	return results, nil
}

func (s *OracleService) SyncStatsFromBlockchain(l *Lobby, _, wallet string) {
	// PILLAR 7-E: Resolve to primary wallet if a linked identity was provided
	if l.identityBridge != nil {
		primary := l.identityBridge.ResolvePrimaryWallet(l, wallet, "Voi")
		if primary != "" {
			wallet = primary
		}
	}

	l.mutex.RLock()
	voiConfig, ok := l.availableNetworks["Voi Mainnet"]
	vaultAddr := l.vaultAddress

	// PILLAR 3: Boundary Snapshot.
	// Snapshot temporal boundaries to ensure consistency if rollover occurs during scan.
	seasonStartUnix := l.seasonStart.Unix()
	activeTournID := l.tournament.ID
	tournOpenTime := l.tournament.OpenTime
	isTournActive := l.tournament.Active
	tournRound := l.tournament.CurrentRound
	l.mutex.RUnlock()

	if !ok || vaultAddr == "" {
		return
	}

	// PASS 1: Wins/DNFs (Vault -> Wallet)
	resp, err := s.IndexerRequest(l, voiConfig, fmt.Sprintf("/arc200/transfers?contractId=%s&from=%s&to=%s&limit=500",
		voiConfig.AssetID, vaultAddr, wallet))

	if err != nil {
		log.Printf("[ORACLE ERROR] Multi-failover indexer scan failed for %s: %v\n", wallet, err)
		return // Should be caught by the loop, but as a final safeguard
	}
	defer resp.Body.Close() // Ensure body is closed

	if resp.StatusCode != http.StatusOK {
		log.Printf("[ORACLE ERROR] Indexer returned non-200 status for stats sync: %d %s\n", resp.StatusCode, resp.Status)
		return // Return on non-OK status
	}

	var res struct {
		Transfers []struct {
			TransactionID string `json:"transactionId"`
			Metadata      string `json:"metadata"`
			Timestamp     int64  `json:"timestamp"`
		} `json:"transfers"`
	}

	wins, dnfs := 0, 0
	var matchHistory []MatchHistory // Unique list of matches for immersion

	// Pass 1: Scan transactions RECEIVED by the wallet (My Wins)
	if json.NewDecoder(resp.Body).Decode(&res) == nil {
		for _, tx := range res.Transfers {
			if tx.Timestamp < seasonStartUnix {
				continue
			}
			if strings.HasPrefix(tx.Metadata, NotePrefixWin) {
				wins++

				// PILLAR 4: Historical Reconstruction. Parse note metadata to rebuild match context.
				var data struct {
					Opp    string `json:"opp"`
					Scores [2]int `json:"scores"`
					TID    string `json:"tid"` // Tournament ID
					MID    string `json:"mid"` // Match ID
				}
				if err := json.Unmarshal([]byte(strings.TrimPrefix(tx.Metadata, NotePrefixWin)), &data); err == nil {
					matchID := data.MID
					if matchID == "" {
						matchID = data.TID
					} // Legacy fallback
					tournID := data.TID
					if data.MID == "" {
						tournID = ""
					} // If MID is empty, TID was MatchID

					matchHistory = append(matchHistory, MatchHistory{
						Opponent:          data.Opp,
						Scores:            data.Scores,
						TournamentID:      tournID,
						TournamentMatchID: matchID,
						ReceiptTxID:       tx.TransactionID,
						Timestamp:         time.Unix(tx.Timestamp, 0),
						WinnerIndex:       0, // Recipient of VBT_WIN is the winner
					})
				} else {
					matchHistory = append(matchHistory, MatchHistory{Opponent: "Legacy Victory", Timestamp: time.Unix(tx.Timestamp, 0)})
				}
			}
			if strings.HasPrefix(tx.Metadata, NotePrefixDNF) {
				dnfs++
			}
		}
	}

	// PILLAR 4: Mirrored Immersion.
	// Pass 3: Global Result Recovery. Scan the Vault's output to find matches where I was the Loser.
	// This allows reconstructing persistent "Loss" records without extra blockchain fees.
	resp, err = s.IndexerRequest(l, voiConfig, fmt.Sprintf("/arc200/transfers?contractId=%s&from=%s&limit=200",
		voiConfig.AssetID, vaultAddr))

	if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		var gRes struct {
			Transfers []struct {
				TransactionID string `json:"transactionId"`
				To            string `json:"to"`
				Metadata      string `json:"metadata"`
				Timestamp     int64  `json:"timestamp"`
			} `json:"transfers"`
		}
		if json.NewDecoder(resp.Body).Decode(&gRes) == nil {
			for _, tx := range gRes.Transfers {
				if tx.Timestamp < seasonStartUnix {
					continue
				}

				if strings.HasPrefix(tx.Metadata, NotePrefixWin) {
					var data struct {
						Opp    string `json:"opp"`
						Scores [2]int `json:"scores"`
						TID    string `json:"tid"`
						MID    string `json:"mid"`
					}
					if err := json.Unmarshal([]byte(strings.TrimPrefix(tx.Metadata, NotePrefixWin)), &data); err == nil {
						if strings.EqualFold(data.Opp, wallet) {
							matchHistory = append(matchHistory, MatchHistory{
								Opponent:          tx.To,
								Scores:            data.Scores,
								TournamentID:      data.TID,
								TournamentMatchID: data.MID,
								ReceiptTxID:       tx.TransactionID,
								Timestamp:         time.Unix(tx.Timestamp, 0),
								WinnerIndex:       1,
							})
						}
					}
				} else if strings.HasPrefix(tx.Metadata, NotePrefixDNF) {
					var data struct {
						Leaver string `json:"leaver"`
						Opp    string `json:"opp"`
						TID    string `json:"tid"`
					}
					if err := json.Unmarshal([]byte(strings.TrimPrefix(tx.Metadata, NotePrefixDNF)), &data); err == nil {
						if strings.EqualFold(data.Leaver, wallet) {
							matchHistory = append(matchHistory, MatchHistory{
								Opponent: data.Opp, TournamentMatchID: data.TID,
								ReceiptTxID: tx.TransactionID,
								Timestamp:   time.Unix(tx.Timestamp, 0), WinnerIndex: 1, // I left
							})
						} else if strings.EqualFold(data.Opp, wallet) {
							matchHistory = append(matchHistory, MatchHistory{
								Opponent: data.Leaver, TournamentMatchID: data.TID,
								ReceiptTxID: tx.TransactionID,
								Timestamp:   time.Unix(tx.Timestamp, 0), WinnerIndex: 0, // They left
							})
						}
					}
				}
			}
		}
	}

	sort.Slice(matchHistory, func(i, j int) bool { return matchHistory[i].Timestamp.After(matchHistory[j].Timestamp) })

	l.mutex.Lock()
	l.ensurePlayerStatsMapsInitialized(wallet) // Ensure maps are initialized
	stats := l.leaderboard[wallet]
	stats.Wins, stats.DNFs = wins, dnfs // Update raw wins/dnfs
	stats.History = matchHistory
	stats.Reputation = l.CalculateReputation(stats)
	l.leaderboard[wallet] = stats
	l.mutex.Unlock()

	// PASS 2: Buy-ins/Registrations (Wallet -> Vault)
	// This allows the server to discover used TxIDs for the specific player joining.
	resp, err = s.IndexerRequest(l, voiConfig, fmt.Sprintf("/arc200/transfers?contractId=%s&from=%s&to=%s&limit=500",
		voiConfig.AssetID, wallet, vaultAddr))

	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var regRes struct {
				Transfers []struct {
					TransactionID string `json:"transactionId"`
					From          string `json:"from"`
					Metadata      string `json:"metadata"`
					Timestamp     int64  `json:"timestamp"`
				} `json:"transfers"`
			}
			if json.NewDecoder(resp.Body).Decode(&regRes) == nil {
				l.mutex.Lock()
				for _, tx := range regRes.Transfers {
					if strings.HasPrefix(tx.Metadata, NotePrefixTournamentBuyIn) || strings.HasPrefix(tx.Metadata, NotePrefixArenaTournamentBuyIn) {
						txTime := time.Unix(tx.Timestamp, 0)
						l.registeredTxIDs[tx.TransactionID] = txTime

						// PILLAR 3: Registration Reconstruction.
						// Use TournamentID for precise reconstruction if available in note
						parts := strings.Split(tx.Metadata, ":")
						matchesCurrent := false
						if len(parts) >= 2 && parts[1] == activeTournID {
							matchesCurrent = true
						} else if txTime.After(tournOpenTime) {
							matchesCurrent = true // Legacy fallback
						}

						if isTournActive && tournRound == 0 && matchesCurrent {
							if !l.isWalletRegisteredLocked(wallet) {
								l.paidParticipants = append(l.paidParticipants, wallet)
								log.Printf("[ORACLE] Reconstructed tournament entry for %s (Tx: %s)\n", wallet, tx.TransactionID)
							}
						}
					}
				} // Fixed: Missing closing brace for for loop
				l.mutex.Unlock()
			}
		}
	}
}

func (s *OracleService) RefreshGlobalLeaderboard(l *Lobby) {
	l.mutex.RLock()
	voiConfig, ok := l.availableNetworks["Voi Mainnet"]
	l.mutex.RUnlock()
	if !ok {
		return
	}

	resp, err := s.IndexerRequest(l, voiConfig, fmt.Sprintf("/arc200/transfers?contractId=%s&from=%s&limit=1000",
		voiConfig.AssetID, l.vaultAddress))

	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[ORACLE ERROR] Indexer returned non-200 status for leaderboard refresh: %d %s\n", resp.StatusCode, resp.Status)
		return
	}

	var res struct {
		Transfers []struct {
			To        string `json:"to"`
			Metadata  string `json:"metadata"`
			Timestamp int64  `json:"timestamp"`
		} `json:"transfers"`
	}

	type tStats struct{ wins, dnfs int }
	data := make(map[string]*tStats)
	if json.NewDecoder(resp.Body).Decode(&res) == nil {
		for _, tx := range res.Transfers {
			if tx.Timestamp < l.seasonStart.Unix() {
				continue
			}
			if _, ok := data[tx.To]; !ok {
				data[tx.To] = &tStats{}
			}
			if strings.HasPrefix(tx.Metadata, NotePrefixWin) {
				data[tx.To].wins++
			}
			if strings.HasPrefix(tx.Metadata, NotePrefixDNF) {
				data[tx.To].dnfs++
			}
		}
	}

	l.mutex.Lock()
	for w, s := range data {
		st := l.leaderboard[w]
		l.ensurePlayerStatsMapsInitialized(w) // Ensure maps are initialized
		st.Wins, st.DNFs = s.wins, s.dnfs     // Update raw wins/dnfs
		st.Reputation = l.CalculateReputation(st)
		l.leaderboard[w] = st
	}
	msg := l.getLobbyUpdateMsgLocked()
	l.mutex.Unlock()
	l.broadcast <- msg
}

// loadOnboardedWalletsFromIndexer reconstructs the historical Sybil protection state from blockchain snapshots.
// PILLAR 6: Blockchain Persistence.
func (s *OracleService) LoadOnboardedWalletsFromIndexer(l *Lobby) {
	// PILLAR 6: Bootstrap Optimization.
	// If the state was already hydrated from a blockchain snapshot (VBT_ECONOMY_SNAPSHOT),
	// we bypass the expensive paged indexer scan to minimize startup latency.
	l.mutex.RLock()
	if len(l.onboardedWallets) > 0 {
		log.Printf("[ORACLE] Onboarding state hydrated from snapshot (%d records). Bypassing scan.\n", len(l.onboardedWallets))
		l.mutex.RUnlock()
		return
	}
	l.mutex.RUnlock()

	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.onboardedWallets = make(map[string]bool)
	if l.loadBlockchainStateSnapshotLocked(NotePrefixOnboardSnapshot, &l.onboardedWallets) {
		l.SybilSyncComplete = true
	} else {
		log.Println("[ORACLE] No onboarding snapshot found. Sybil protection starting fresh.")
		l.SybilSyncComplete = true // Allow operations even if history is empty
	}
}

// saveOnboardedWallets persists the Sybil protection map to the blockchain.
// PILLAR 6: Blockchain Persistence.
func (s *OracleService) SaveOnboardedWallets(l *Lobby) {
	l.mutex.RLock()
	// Create a point-in-time snapshot of the map
	snapshot := make(map[string]bool, len(l.onboardedWallets))
	for k, v := range l.onboardedWallets {
		snapshot[k] = v
	}
	l.mutex.RUnlock()

	// Marshal the snapshot outside the lock
	data, err := json.Marshal(snapshot)
	if err != nil {
		log.Printf("[CACHE ERROR] Failed to marshal onboarded wallets snapshot: %v\n", err)
		return
	}
	l.dispatchBlockchainSnapshot(NotePrefixOnboardSnapshot, data)
}

// loadRegistrationsFromIndexer reconstructs the tournament registration state from the blockchain.
// It identifies paid participants by scanning the indexer for transactions with the buy-in prefix.
func (s *OracleService) LoadRegistrationsFromIndexer(l *Lobby) {
	l.mutex.RLock()
	voiConfig, ok := l.availableNetworks["Voi Mainnet"]
	vaultAddr := l.vaultAddress
	rewardAsset := l.rewardAssetID
	activeTournID := l.tournament.ID

	// PILLAR 6: Bootstrap Optimization.
	// If the participant list was already hydrated from a blockchain snapshot,
	// we bypass the full sweep to minimize startup latency and RPC overhead.
	if len(l.paidParticipants) > 0 {
		log.Printf("[ORACLE] Tournament state hydrated from snapshot (%d entries). Bypassing indexer scan for event %s.\n",
			len(l.paidParticipants), activeTournID)
		l.mutex.RUnlock()
		return
	}

	// PILLAR 3: State Reconstruction.
	// Fetch the buy-in amount to reconstruct the prize pool commitment.
	buyInAmt := l.tournament.BuyInAmount
	l.mutex.RUnlock()

	if !ok || vaultAddr == "" || activeTournID == "" {
		return
	}

	log.Printf("[ORACLE] Syncing tournament registrations from indexer for event %s...\n", activeTournID)

	// Query all transfers received by the vault for the current primary reward asset
	url := fmt.Sprintf("/arc200/transfers?contractId=%s&to=%s&limit=500", rewardAsset, vaultAddr)
	resp, err := s.IndexerRequest(l, voiConfig, url)
	if err != nil {
		log.Printf("[ORACLE ERROR] Failed to fetch registrations from indexer: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var res struct {
		Transfers []struct {
			TransactionID string `json:"transactionId"`
			From          string `json:"from"`
			Metadata      string `json:"metadata"`
			Timestamp     int64  `json:"timestamp"`
		} `json:"transfers"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return
	}

	// PILLAR 4: Mutex Contention Minimization.
	// Process indexer results into local buffers outside the write lock.
	// This ensures the lobby loop remains responsive during heavy state reconstruction.
	newParticipants := []string{}
	newTxIDs := make(map[string]time.Time)
	var reconstructedBonus float64 = 0
	seen := make(map[string]bool)

	for _, tx := range res.Transfers {
		// PILLAR 3: Bound Verification. Use trailing colon for absolute ID isolation.
		if strings.HasPrefix(tx.Metadata, NotePrefixTournamentBuyIn+activeTournID+":") {
			newTxIDs[tx.TransactionID] = time.Unix(tx.Timestamp, 0)

			lowerFrom := strings.ToLower(tx.From)
			if !seen[lowerFrom] {
				seen[lowerFrom] = true
				newParticipants = append(newParticipants, tx.From)
				reconstructedBonus += float64(buyInAmt / 2)
			}
		}
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Reset bonus before reconstruction to prevent double-counting if the scan is re-triggered.
	l.tournamentPotBonus = reconstructedBonus

	for txid, ts := range newTxIDs {
		l.registeredTxIDs[txid] = ts
	}

	// Optimization: Use a map for O(1) lookups to minimize lock duration
	existing := make(map[string]bool)
	for _, p := range l.paidParticipants {
		existing[strings.ToLower(p)] = true
	}

	for _, p := range newParticipants {
		// PILLAR 5: Scalability.
		// Deduplicate against existing memory state in O(M) time.
		if !existing[strings.ToLower(p)] {
			l.paidParticipants = append(l.paidParticipants, p)
			log.Printf("[ORACLE] Reconstructed participant entry: %s\n", p)
		}
	}

	log.Println("[ORACLE] Tournament registration reconstruction complete.")
}

// ResolveEnvoiName attempts to find a .voi or .algo name for a wallet address.
// It utilizes a dedicated lock and local cache to minimize indexer traffic and avoid deadlocks.
func (s *OracleService) ResolveEnvoiName(l *Lobby, address string) string {
	if address == "" || address == "TBD" || address == "BYE" {
		return address
	}

	l.envoiMutex.RLock()
	if name, ok := l.envoiCache[address]; ok {
		l.envoiMutex.RUnlock()
		return name
	}
	l.envoiMutex.RUnlock()

	// Primary: canonical Envoi naming-service API (Voi address naming, like ENS).
	// Falls back to the indexer token-scan below if the service is unreachable/unresolved.
	if name, ok := s.resolveEnvoiViaAPI(address); ok && name != "" {
		l.envoiMutex.Lock()
		l.envoiCache[address] = name
		l.envoiMutex.Unlock()
		return name
	}

	// Basic Truncation fallback
	truncated := address[:6] + "..." + address[len(address)-4:]

	l.mutex.RLock()
	voiConfig, ok := l.availableNetworks["Voi Mainnet"]
	l.mutex.RUnlock()

	if !ok || len(voiConfig.IndexerURLs) == 0 {
		return truncated
	}

	// PILLAR 4: RPC Failover.
	// Use the resilient dispatcher to resolve .voi names.
	resp, err := s.IndexerRequest(l, voiConfig, fmt.Sprintf("/tokens?owner=%s", address))
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		var res struct {
			Tokens []struct {
				Metadata string `json:"metadata"`
			} `json:"tokens"`
		}
		if json.NewDecoder(resp.Body).Decode(&res) == nil {
			for _, t := range res.Tokens {
				var meta struct {
					Name string `json:"name"`
				}
				if json.Unmarshal([]byte(t.Metadata), &meta) == nil && strings.HasSuffix(strings.ToLower(meta.Name), ".voi") {
					l.envoiMutex.Lock()
					l.envoiCache[address] = meta.Name
					l.envoiMutex.Unlock()
					return meta.Name
				}
			}
		}
	}

	// Negative Cache: Store the truncated fallback to prevent repeated indexer hits for non-.voi wallets
	l.envoiMutex.Lock()
	l.envoiCache[address] = truncated
	l.envoiMutex.Unlock()

	return truncated
}

// resolveEnvoiViaAPI queries the public Envoi naming service for the .voi name bound to address.
// Returns ("", false) when unresolved or on any transport/parse error so the caller can fall back
// to the indexer token-scan or truncation. Honors ENVOI_API_URL for custom gateways.
func (s *OracleService) resolveEnvoiViaAPI(address string) (string, bool) {
	base := os.Getenv("ENVOI_API_URL")
	if base == "" {
		base = envoiAPIBase
	}
	base = strings.TrimRight(base, "/")
	url := fmt.Sprintf("%s/api/name/%s", base, address)

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", false
	}
	var out struct {
		Results []struct {
			Name string `json:"name"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", false
	}
	for _, r := range out.Results {
		if r.Name != "" {
			return r.Name, true
		}
	}
	return "", false
}

func (s *OracleService) VerifyBuyInTransaction(l *Lobby, network, txid string, expectedAmt uint64, expectedAsset, sender, vaultAddr, expectedNotePrefix string) (bool, int64, error) {
	// ── FAIL CLOSED ON THE BINDING INPUTS (see note_vocabulary.go / txid_memo.go)
	//
	// An EMPTY purpose prefix must be refused, never tolerated: strings.HasPrefix
	// (note, "") is ALWAYS true, so an empty prefix would drop the purpose
	// binding entirely and let a payment made for one purpose satisfy another.
	// The vocabulary owner is the only legitimate source of a prefix and its
	// builders cannot produce a non-empty prefix without a base.
	//
	// An EMPTY transaction id is refused for the same class of reason: it cannot
	// be memoised, so a door that accepted it would be replayable forever.
	// Callers perform the same checks earlier so they can answer with their own
	// message; this is the boundary that makes the rule unbypassable.
	if expectedNotePrefix == "" {
		return false, 0, fmt.Errorf("empty note purpose prefix refused: a purpose prefix is required to bind a payment to its purpose")
	}
	if strings.TrimSpace(txid) == "" {
		return false, 0, fmt.Errorf("empty transaction id refused: an unmemoisable payment cannot be verified")
	}

	// 1. Authoritative Network Key Resolution (Deterministic Case Sync)
	netKey := s.MapChainToNetworkName(network)
	if netKey == "" {
		netKey = network // Fallback for direct usage
	}

	l.mutex.RLock()
	netConfig, ok := l.availableNetworks[netKey]
	l.mutex.RUnlock()

	if !ok {
		return false, 0, fmt.Errorf("network configuration not found for: %s", netKey)
	}

	// Authoritative ID Resolution: Pull from networks.json if the provided parameter is generic
	// PILLAR 3: Robust economic validation.
	targetAsset := expectedAsset
	if targetAsset == "" || targetAsset == "0" {
		if netConfig.AssetID != "" && netConfig.AssetID != "0" {
			targetAsset = netConfig.AssetID
		} else if netConfig.AppID != "" && netConfig.AppID != "0" {
			targetAsset = netConfig.AppID
		}
	}

	// 2. Branch logic based on Network Type
	if strings.Contains(strings.ToLower(netKey), "voi") {
		// ── ARC-200 ON VOI: TWO FACTS, TWO ENDPOINTS ───────────────────────────
		//
		// MEASURED 2026-09-17 (Problems.md §36):
		//   * `/arc200/transfers` returns the transfer registry — `sender`,
		//     `receiver`, `amount`, `contractId`, `timestamp` — and NO note. The
		//     decode here spelled those fields `From`/`To`/`Metadata`, which Go
		//     matches against the json keys `from`/`to`/`metadata`: keys this API
		//     never sends. Every field except Amount/ContractID/Timestamp was
		//     therefore ALWAYS empty, so the condition below could not hold and NO
		//     Voi payment could ever verify. (It failed CLOSED: nothing was
		//     falsely accepted — it simply never worked.)
		//   * `?transactionId=` is IGNORED by this API (it answers with the newest
		//     transfers regardless), so the id is matched HERE, in code.
		//   * the NOTE lives on the TRANSACTION, not on the transfer record, and is
		//     read from the standard indexer's `/v2/transactions/{txid}`.
		//
		// Both facts are required: a transfer without a matching note prefix is not
		// bound to a purpose, and a note without a matching transfer is not a
		// payment. A note that cannot be READ is a refusal, never a silent pass.
		resp, err := s.IndexerRequest(l, netConfig, fmt.Sprintf("/arc200/transfers?contractId=%s&from=%s&limit=1000",
			targetAsset, sender))
		if err != nil {
			return false, 0, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return false, 0, fmt.Errorf("voi indexer returned non-200 status: %d", resp.StatusCode)
		}

		var res struct {
			Transfers []struct {
				TransactionID string `json:"transactionId"`
				Sender        string `json:"sender"`
				Receiver      string `json:"receiver"`
				Amount        string `json:"amount"`
				ContractID    uint64 `json:"contractId"`
				Timestamp     int64  `json:"timestamp"`
			} `json:"transfers"`
		}

		found := false
		var foundTS int64
		if err := json.NewDecoder(resp.Body).Decode(&res); err == nil {
			for _, tx := range res.Transfers {
				if tx.TransactionID != txid {
					continue
				}
				amt, amtErr := parseARC200Amount(tx.Amount)
				// SECURITY: the transaction id, the sender, the receiver, the
				// minimum amount and the token contract must ALL agree.
				if amtErr != nil || !strings.EqualFold(tx.Sender, sender) || !strings.EqualFold(tx.Receiver, vaultAddr) {
					continue
				}
				if amt < expectedAmt || strconv.FormatUint(tx.ContractID, 10) != targetAsset {
					continue
				}
				found = true
				foundTS = tx.Timestamp
				break
			}
		}
		if !found {
			// An unseen or mismatched transfer is simply "not verified yet".
			return false, 0, nil
		}

		note, noteFound, noteErr := s.fetchTransactionNote(l, netConfig, txid)
		if noteErr != nil {
			return false, 0, fmt.Errorf("voi note lookup failed: %w", noteErr)
		}
		if !noteFound {
			return false, 0, fmt.Errorf("transaction %s is not indexed by the standard indexer yet, so its purpose note could not be read", txid)
		}
		if !strings.HasPrefix(note, expectedNotePrefix) {
			return false, 0, nil
		}
		return true, foundTS, nil
	} else {
		// ALGORAND Logic: Standard Indexer Transaction Endpoint
		resp, err := s.IndexerRequest(l, netConfig, fmt.Sprintf("/v2/transactions/%s", txid))
		if err != nil {
			return false, 0, err
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			return false, 0, nil
		}
		if resp.StatusCode != http.StatusOK {
			return false, 0, fmt.Errorf("algorand indexer returned non-200 status: %d", resp.StatusCode)
		}

		var res struct {
			Transaction struct {
				AssetTransfer *struct {
					Receiver string `json:"receiver"`
					Amount   uint64 `json:"amount"`
					AssetID  uint64 `json:"asset-id"`
				} `json:"asset-transfer-transaction,omitempty"`
				Payment *struct {
					Receiver string `json:"receiver"`
					Amount   uint64 `json:"amount"`
				} `json:"payment-transaction,omitempty"`
				Sender    string `json:"sender"`
				Note      []byte `json:"note"`
				RoundTime int64  `json:"round-time"`
			} `json:"transaction"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err == nil {
			t := res.Transaction
			noteStr := string(t.Note)
			// Handle ASA Transfers
			if t.AssetTransfer != nil && strings.EqualFold(t.Sender, sender) && strings.EqualFold(t.AssetTransfer.Receiver, vaultAddr) && t.AssetTransfer.Amount >= expectedAmt && strconv.FormatUint(t.AssetTransfer.AssetID, 10) == targetAsset && strings.HasPrefix(noteStr, expectedNotePrefix) {
				return true, t.RoundTime, nil
			}
			// Handle Native Payments (Asset ID "0" or empty)
			if (targetAsset == "" || targetAsset == "0") && t.Payment != nil && strings.EqualFold(t.Sender, sender) && strings.EqualFold(t.Payment.Receiver, vaultAddr) && t.Payment.Amount >= expectedAmt && strings.HasPrefix(noteStr, expectedNotePrefix) {
				return true, t.RoundTime, nil
			}
		}
	}
	return false, 0, nil
}

// fetchTransactionNote reads the NOTE of one transaction from the standard
// indexer (`/v2/transactions/{txid}`).
//
// The ARC-200 transfer registry does not carry notes at all (measured
// 2026-09-17: its records are transactionId/contractId/timestamp/round/sender/
// receiver/amount), so a payment's PURPOSE binding can only be checked against
// the transaction itself. The standard indexer is the host that serves /v2/*;
// indexerGet advances past a base that does not route the path.
//
// Returns (note, true, nil) when the transaction was read — the note may be
// empty, which the caller treats as a prefix mismatch — and ("", false, nil) for
// a 404 (not indexed yet).
func (s *OracleService) fetchTransactionNote(l *Lobby, cfg NetworkConfig, txid string) (string, bool, error) {
	resp, err := s.IndexerRequest(l, cfg, fmt.Sprintf("/v2/transactions/%s", txid))
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("standard indexer returned HTTP %d for transaction %s", resp.StatusCode, txid)
	}

	var res struct {
		Transaction struct {
			Note []byte `json:"note"`
		} `json:"transaction"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", false, fmt.Errorf("transaction response could not be decoded: %w", err)
	}
	return string(res.Transaction.Note), true, nil
}

// CheckAssetApproval verifies if an owner has approved a spender (e.g. the Vault)
// for a specific amount of an ARC-200 token.
// PILLAR 2: High-Finance. Enables non-custodial bidding and automated settlement.
func (s *OracleService) CheckAssetApproval(l *Lobby, network, owner, spender, assetIDStr string) (uint64, error) {
	netKey := s.MapChainToNetworkName(network)
	if netKey == "" {
		netKey = network
	}

	l.mutex.RLock()
	netConfig, ok := l.availableNetworks[netKey]
	l.mutex.RUnlock()

	if !ok {
		return 0, fmt.Errorf("network config not found: %s", netKey)
	}

	// Approvals are currently specific to ARC-200 on Voi
	if !strings.Contains(strings.ToLower(netKey), "voi") {
		return 0, fmt.Errorf("approval verification only supported on Voi ARC-200 assets")
	}

	url := fmt.Sprintf("/arc200/approvals?contractId=%s&owner=%s&spender=%s", assetIDStr, owner, spender)
	resp, err := s.IndexerRequest(l, netConfig, url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("indexer returned status: %d", resp.StatusCode)
	}

	var res struct {
		Approvals []struct {
			Amount string `json:"amount"`
		} `json:"approvals"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err == nil && len(res.Approvals) > 0 {
		// PILLAR 2: Integer Supremacy.
		amt, _ := strconv.ParseUint(res.Approvals[0].Amount, 10, 64)
		return amt, nil
	}

	return 0, nil
}

// fetchARC69Metadata retrieves metadata from the latest configuration transaction note.
func (s *OracleService) FetchARC69Metadata(l *Lobby, cfg NetworkConfig, assetID int) (*ARC72Metadata, error) {
	resp, err := s.IndexerRequest(l, cfg, fmt.Sprintf("/v2/assets/%d/transactions?tx-type=acfg&limit=1", assetID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("indexer returned non-200 status: %d", resp.StatusCode)
	}

	var res struct {
		Transactions []struct {
			Note []byte `json:"note"`
		} `json:"transactions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode transaction history: %w", err)
	}

	if len(res.Transactions) == 0 || len(res.Transactions[0].Note) == 0 {
		return nil, fmt.Errorf("no metadata found in asset configuration history")
	}

	var meta ARC72Metadata
	if err := json.Unmarshal(res.Transactions[0].Note, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse ARC-69 JSON note: %w", err)
	}

	return &meta, nil
}

// fetchARC19Metadata resolves a dynamic IPFS CID from the asset's reserve address.
func (s *OracleService) FetchARC19Metadata(l *Lobby, cfg NetworkConfig, assetID int) (*ARC72Metadata, error) {
	// 1. Fetch Asset Information from Indexer to retrieve the Reserve Address
	resp, err := s.IndexerRequest(l, cfg, fmt.Sprintf("/v2/assets/%d", assetID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res struct {
		Asset struct {
			Params struct {
				Reserve string `json:"reserve"`
			} `json:"params"`
		} `json:"asset"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode asset info: %w", err)
	}

	if res.Asset.Params.Reserve == "" {
		return nil, fmt.Errorf("no reserve address found for ARC-19 resolution")
	}

	// 2. Convert Reserve Address to CIDv1 (ARC-19 Standard)
	addr, err := types.DecodeAddress(res.Asset.Params.Reserve)
	if err != nil {
		return nil, fmt.Errorf("failed to decode reserve address: %w", err)
	}

	// PILLAR 4: ARC-19 CIDv1 Conversion.
	// binary: [0x01 (v1), 0x55 (raw), 0x12 (sha2-256), 0x20 (len), <32_byte_pubkey>]
	header := []byte{0x01, 0x55, 0x12, 0x20}
	full := append(header, addr[:]...)
	cid := "b" + strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(full))

	// 3. Fetch Metadata JSON from IPFS Gateway
	ipfsGateway := cfg.IPFSGatewayURL
	if ipfsGateway == "" {
		ipfsGateway = "https://ipfs.io/ipfs/" // Fallback to public gateway
	}
	ipfsURL := ipfsGateway + cid

	// PILLAR 4: Authenticated IPFS Access.
	// Add API key or custom headers if configured.
	var ipfsResp *http.Response
	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), indexerTimeout)
		req, _ := http.NewRequestWithContext(ctx, "GET", ipfsURL, nil)
		if cfg.IPFSAPIKey != "" {
			req.Header.Set("Authorization", "Bearer "+cfg.IPFSAPIKey)
		}
		for k, v := range cfg.IPFSHeaders {
			req.Header.Set(k, v)
		}
		ipfsResp, err = http.DefaultClient.Do(req)
		cancel() // Release resources early

		if err == nil && ipfsResp != nil {
			if ipfsResp.StatusCode == http.StatusOK {
				break
			}
			if ipfsResp.StatusCode == http.StatusTooManyRequests {
				ipfsResp.Body.Close()
				time.Sleep(time.Duration(i+1) * 1 * time.Second) // Exponential backoff
				continue
			}
			ipfsResp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}

	if err != nil || ipfsResp == nil || ipfsResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch ARC-19 metadata from IPFS")
	}
	defer ipfsResp.Body.Close()

	var meta ARC72Metadata
	if err := json.NewDecoder(ipfsResp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}
	return &meta, nil
}

// MetadataDispatcher identifies the NFT standard (ARC-72, ARC-69, or ARC-19)
// and routes the metadata retrieval request to the appropriate service.
func (s *OracleService) MetadataDispatcher(l *Lobby, networkName string, assetID int) (*ARC72Metadata, string, error) {
	l.mutex.RLock()
	cfg, ok := l.availableNetworks[networkName]
	l.mutex.RUnlock()
	if !ok {
		return nil, "", fmt.Errorf("unsupported network for metadata dispatch: %s", networkName)
	}

	// 1. ARC-19 Detection: Fetch Asset parameters from Indexer to check for template URL.
	// This is the most efficient first check for dynamic ASAs.
	resp, err := s.IndexerRequest(l, cfg, fmt.Sprintf("/v2/assets/%d", assetID))

	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		var res struct {
			Asset struct {
				Params struct {
					URL string `json:"url"`
				} `json:"params"`
			} `json:"asset"`
		}
		if json.NewDecoder(resp.Body).Decode(&res) == nil && strings.Contains(res.Asset.Params.URL, "template-ipfs") {
			meta, err := s.FetchARC19Metadata(l, cfg, assetID)
			return meta, "ARC-19", err
		}
	} else if resp != nil {
		resp.Body.Close()
	}

	// 2. ARC-72 Check: If network has a configured AppID, check if this ID exists as a token.
	if cfg.AppID != "" && cfg.AppID != "0" {
		resp72, err := s.IndexerRequest(l, cfg, fmt.Sprintf("/tokens?contractId=%s&tokenId=%d", cfg.AppID, assetID))
		if err == nil && resp72.StatusCode == http.StatusOK {
			defer resp72.Body.Close()
			var res72 struct {
				Tokens []struct {
					Metadata string `json:"metadata"`
				} `json:"tokens"`
			}
			if json.NewDecoder(resp72.Body).Decode(&res72) == nil && len(res72.Tokens) > 0 {
				var meta ARC72Metadata
				json.Unmarshal([]byte(res72.Tokens[0].Metadata), &meta)
				return &meta, "ARC-72", nil
			}
		}
	}

	// 3. Fallback to ARC-69: Scan configuration history for JSON notes.
	meta, err := s.FetchARC69Metadata(l, cfg, assetID)
	return meta, "ARC-69", err
}

// ════════════════════════════════════════════════════════════════════════════
// ARC-200 BALANCES — the ONE way a holder's token balance is read.
//
// Measured 2026-09-17 (Problems.md §36): the vault's $VBV was read from an
// ALGOD APPLICATION BOX (`GetApplicationBoxByName(rewardAssetID, vault)`), which
// an ARC-200 token application does not have — so the read failed with "box not
// found" on every boot and every faucet refresh, and the server published a ZERO
// pool while the chain showed the vault holding 7,000 $VBV. A false zero is the
// worst kind of wrong: it reads as a funding state.
//
// The ARC-200 API answers the question directly:
//
//	GET /arc200/balances?accountId=<holder>&contractId=<token>
//	-> {"balances":[{"balance":"<decimal string>","contractId":N,...}]}
//
// The amount is a DECIMAL STRING on the wire and is parsed with an explicit
// range check, because ARC-200 amounts are uint256 and this indexer does emit
// the 2^256-1 mint/burn sentinel in transfer records: a value above uint64 must
// be REFUSED, never truncated into the ledger.
// ════════════════════════════════════════════════════════════════════════════

// arc200BalanceEntry is the projection of one /arc200/balances row.
type arc200BalanceEntry struct {
	Name       string `json:"name"`
	Symbol     string `json:"symbol"`
	Balance    string `json:"balance"`
	Decimals   int    `json:"decimals"`
	AccountID  string `json:"accountId"`
	ContractID uint64 `json:"contractId"`
}

// FetchARC200Balance reads one holder's balance of one ARC-200 contract.
//
// It takes the network config directly and touches no lobby state, so a caller
// may use it from inside a locked section without extending that lock across the
// network round-trip.
func (s *OracleService) FetchARC200Balance(cfg NetworkConfig, accountID, contractID string) (uint64, string, error) {
	if strings.TrimSpace(accountID) == "" {
		return 0, "", fmt.Errorf("arc200 balance: an account id is required")
	}
	if strings.TrimSpace(contractID) == "" || contractID == "0" {
		return 0, "", fmt.Errorf("arc200 balance: a non-empty contract id is required (\"0\" can never resolve a token)")
	}
	if len(cfg.IndexerURLs) == 0 {
		return 0, "", fmt.Errorf("arc200 balance: no indexer base is configured for this network")
	}

	path := fmt.Sprintf("/arc200/balances?accountId=%s&contractId=%s",
		url.QueryEscape(accountID), url.QueryEscape(contractID))
	resp, err := indexerGet(cfg.IndexerURLs, path)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return 0, "", fmt.Errorf("no configured indexer base serves the ARC-200 balances path %s; the Voi registry must include the ARC-200 API host", path)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("arc200 balance read for contract %s returned HTTP %d", contractID, resp.StatusCode)
	}

	var res struct {
		Balances []arc200BalanceEntry `json:"balances"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return 0, "", fmt.Errorf("arc200 balance response could not be decoded: %w", err)
	}

	want, err := strconv.ParseUint(contractID, 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("arc200 balance: contract id %q is not a decimal id", contractID)
	}
	for _, b := range res.Balances {
		if b.ContractID != want {
			continue
		}
		if b.AccountID != "" && !strings.EqualFold(b.AccountID, accountID) {
			continue
		}
		bal, err := parseARC200Amount(b.Balance)
		if err != nil {
			return 0, b.Symbol, fmt.Errorf("arc200 balance for account %s: %w", accountID, err)
		}
		return bal, b.Symbol, nil
	}

	// No matching row is an ANSWER (not opted in, or zero holdings) — returned as
	// an error so a caller must decide, rather than a bare 0 being read as
	// "the vault is empty".
	return 0, "", fmt.Errorf("account %s holds no balance row for contract %s (not opted in, or zero holdings)", accountID, contractID)
}

// parseARC200Amount converts a decimal amount string with an explicit uint64
// range check. ARC-200 amounts are uint256 on the wire; a value above
// math.MaxUint64 is refused rather than truncated.
func parseARC200Amount(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("the amount field is empty")
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("amount %q is not a decimal integer", s)
		}
	}
	if len(s) > 20 { // 20 digits is already beyond MaxUint64 (18446744073709551615)
		return 0, fmt.Errorf("amount %s exceeds the uint64 ledger range (the ARC-200 mint/burn sentinel is not a balance)", s)
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("amount %q is outside the uint64 ledger range: %w", s, err)
	}
	return v, nil
}

// checkVaultBalanceOnChain synchronizes the internal faucetBalance with the on-chain $VBV pool.
func (s *OracleService) CheckVaultBalanceOnChain(l *Lobby) {
	l.mutex.RLock()
	rewardAssetIDStr := l.rewardAssetID
	vaultAddr := l.vaultAddress
	voiCfg, hasVoi := l.availableNetworks["Voi Mainnet"]
	l.mutex.RUnlock()

	if rewardAssetIDStr == "" || vaultAddr == "" {
		return
	}
	if !hasVoi || len(voiCfg.IndexerURLs) == 0 {
		log.Println("[ORACLE] Vault pool check skipped: the Voi Mainnet registry entry carries no indexer base (see networks.json).")
		return
	}

	// The $VBV pool is an ARC-200 BALANCE. The read that stood here asked ALGOD for
	// an application BOX of the token app — which an ARC-200 token does not have —
	// so it answered "box not found" for ever and the server published a ZERO pool
	// while the chain showed the vault holding 7,000 $VBV (measured 2026-09-17,
	// Problems.md §36). A read that FAILS is now reported as "not live" with the
	// last known reservoir intact, instead of asserting a balance nobody measured.
	// It also no longer probes/blacklists the RPC node cluster for a box that can
	// never exist.
	bal, symbol, err := s.FetchARC200Balance(voiCfg, vaultAddr, rewardAssetIDStr)
	if err != nil {
		log.Printf("[ORACLE] Vault $VBV pool read failed: %v. Reporting the pool as NOT live rather than as zero.\n", err)
		l.mutex.Lock()
		l.vaultBalanceLive = false
		l.mutex.Unlock()
		return
	}
	if symbol == "" {
		symbol = "$VBV"
	}

	l.mutex.Lock()
	l.faucetBalanceMicro = bal
	l.faucetBalance = float64(bal) / 1000000.0
	l.vaultBalanceLive = true
	l.applyDynamicScalingLocked()
	l.mutex.Unlock()
	log.Printf("[ORACLE] Vault %s pool synced from the ARC-200 balances endpoint: %.2f units (contract %s).\n", symbol, l.faucetBalance, rewardAssetIDStr)
}

func (s *OracleService) CheckNativeVaultBalanceOnChain(l *Lobby) {
	l.mutex.RLock()
	vaultAddr := l.vaultAddress
	lb := l.ledgerClient
	l.mutex.RUnlock()

	// PILLAR 4: Multi-Node Cluster Failover.
	if lb == nil {
		return
	}

	lb.Mu.RLock()
	nodeCount := len(lb.Nodes)
	lb.Mu.RUnlock()

	success := false
	for i := 0; i < nodeCount; i++ {
		client, url, err := lb.GetBestClient()
		if err != nil {
			break
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		info, err := client.AccountInformation(vaultAddr).Do(ctx)
		cancel()

		if err != nil {
			// Authoritative HTTP 404: the vault account has no on-chain state yet
			// (unfunded). The node answered correctly — do NOT blacklist it and do NOT
			// raise a gas alert for a fresh account; the real balance is read once funded.
			if IsSemantic404Error(err) {
				log.Printf("[ORACLE] Vault gas state not found on-chain (unfunded); node %s healthy, not blacklisted.\n", url)
				success = true
				break
			}
			log.Printf("[ORACLE WARNING] Node %s failed gas check: %v. Blacklisting.\n", url, err)
			lb.MarkNodeFailure(url)
			continue
		}

		// Evaluate liquidity threshold for automated gas alerts
		if info.Amount < 1000000 { // 1.0 VOI Threshold
			log.Printf("[CRITICAL] Vault gas low! Balance: %d\n", info.Amount)
			l.broadcastToAdmins("⚠️ <b>CRITICAL:</b> Vault gas is nearly depleted.")
		}
		success = true
		break
	}

	if !success {
		log.Printf("[ORACLE ERROR] Gas check failed: all cluster nodes degraded.\n")
	}
}

// handleSeasonHistory fetches archived seasonal standings from the blockchain.
func (s *OracleService) HandleSeasonHistory(l *Lobby, w http.ResponseWriter, r *http.Request) {
	// Ensure Voi Mainnet config is available for transactional operations
	l.mutex.RLock()
	voiConfig, ok := l.availableNetworks["Voi Mainnet"]
	l.mutex.RUnlock()
	if !ok {
		http.Error(w, "Voi Mainnet configuration not found. Cannot fetch season history.", http.StatusInternalServerError)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Optional filter for a specific season number
	targetSeason := -1
	if sStr := r.URL.Query().Get("season"); sStr != "" {
		if val, err := strconv.Atoi(sStr); err == nil {
			targetSeason = val
		}
	}

	faucetAddr := l.vaultAddress
	rewardAssetID := l.rewardAssetID

	// PILLAR 4: RPC Failover.
	// Utilizing the dispatcher to fetch season history receipts.
	resp, err := s.IndexerRequest(l, voiConfig, fmt.Sprintf("/arc200/transfers?contractId=%s&from=%s&to=%s&limit=100",
		rewardAssetID, faucetAddr, faucetAddr))

	if err != nil {
		http.Error(w, "Failed to connect to indexer", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var res struct {
		Transfers []struct {
			TransactionID string `json:"transactionId"`
			Metadata      string `json:"metadata"`
		} `json:"transfers"`
	}

	type SeasonArchive struct {
		Season     int       `json:"season"`
		Start      time.Time `json:"start"`
		End        time.Time `json:"end"`
		Highlights []struct {
			W string `json:"w"` // Wallet
			A string `json:"a"` // Award/Placement Title
			M string `json:"m"` // Meta/Detail (e.g. Tournament ID)
		} `json:"highlights,omitempty"`
		Top []struct {
			W string `json:"w"` // Wallet
			V int    `json:"v"` // Wins
			R string `json:"r"` // Rating
		} `json:"top"`
	}

	// Deduplication Map: Season Number -> Archive Data
	uniqueSeasons := make(map[int]SeasonArchive)

	if err := json.NewDecoder(resp.Body).Decode(&res); err == nil {
		for _, tx := range res.Transfers {
			if strings.HasPrefix(tx.Metadata, NotePrefixSeasonArchive) {
				jsonStr := strings.TrimPrefix(tx.Metadata, NotePrefixSeasonArchive)
				var archive SeasonArchive
				if err := json.Unmarshal([]byte(jsonStr), &archive); err == nil && archive.Season > 0 {
					// PILLAR 4: Replay Resilience. Ensure record is valid before deduplication.
					// A Season ID of 0 indicates a malformed or empty unmarshal result.
					if targetSeason == -1 || archive.Season == targetSeason {
						uniqueSeasons[archive.Season] = archive
					}
				} else {
					log.Printf("[SEASON HISTORY] Skipping malformed record in TxID %s: %v\n", tx.TransactionID, err)
				}
			}
		}
	} else {
		log.Printf("[SEASON HISTORY ERROR] Failed to decode indexer response: %v\n", err)
	}

	history := []SeasonArchive{} // Initialize as empty slice to ensure JSON returns [] instead of null
	for _, s := range uniqueSeasons {
		history = append(history, s)
	}

	// Sort newest first
	sort.Slice(history, func(i, j int) bool { return history[i].Season > history[j].Season })

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// handleReSyncStats triggers a manual sync for a specific wallet address.
func (s *OracleService) HandleReSyncStats(l *Lobby, w http.ResponseWriter, r *http.Request) {
	var req struct {
		Wallet string `json:"wallet"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Wallet == "" {
		http.Error(w, "Invalid wallet", http.StatusBadRequest)
		return
	}
	go s.SyncStatsFromBlockchain(l, "UI_TRIGGER", req.Wallet)
	json.NewEncoder(w).Encode(map[string]string{"status": "sync_initiated"})
}

// mapChainToNetworkName translates frontend chain codes to internal NetworkConfig keys.
func (s *OracleService) MapChainToNetworkName(chain string) string {
	switch strings.ToUpper(chain) {
	case "ETH":
		return "Ethereum"
	case "SOL":
		return "Solana"
	case "POLY":
		return "Polygon"
	case "ALGO":
		return "Algorand Mainnet"
	case "VOI":
		return "Voi Mainnet"
	default:
		return ""
	}
}

// checkAssetOptIn verifies if a wallet is opted into a specific asset (ASA or ARC-200 balance box).
func (s *OracleService) CheckAssetOptIn(l *Lobby, network, wallet string, assetIDStr string) (bool, int64, error) {
	if assetIDStr == "" || assetIDStr == "0" {
		return true, 0, nil
	}

	// 1. Authoritative Network Key Resolution (Deterministic Case Sync)
	netKey := s.MapChainToNetworkName(network)
	if netKey == "" {
		netKey = network // Fallback for direct full-name usage
	}

	l.mutex.RLock()
	netConfig, ok := l.availableNetworks[netKey]
	l.mutex.RUnlock()

	if !ok {
		return false, 0, fmt.Errorf("network configuration not found: %s", netKey)
	}

	// 2. VOI / ARC-200 Pattern: Verify Balance Box existence
	if strings.Contains(strings.ToLower(netKey), "voi") {
		assetID, _ := strconv.ParseUint(assetIDStr, 10, 64)
		addr, _ := types.DecodeAddress(wallet)

		// PILLAR 4: Cluster integration for Opt-In verification.
		if l.ledgerClient != nil {
			client, url, err := l.ledgerClient.GetBestClient()
			if err == nil {
				for i := 0; i < 3; i++ {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					_, err := client.GetApplicationBoxByName(assetID, addr[:]).Do(ctx)
					cancel()
					if err == nil {
						return true, 0, nil
					}
					if strings.Contains(err.Error(), "404") || strings.Contains(strings.ToLower(err.Error()), "not found") {
						return false, 0, nil
					}
					time.Sleep(500 * time.Millisecond)
				}
				l.ledgerClient.MarkNodeFailure(url)
			}
		}
		return false, 0, fmt.Errorf("voi cluster failover: all healthy nodes exhausted during opt-in check")
	}

	// 3. ALGORAND / ASA Pattern: Indexer Account Asset Scan
	resp, err := s.IndexerRequest(l, netConfig, fmt.Sprintf("/v2/accounts/%s", wallet))
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, 0, nil
	} else if resp.StatusCode != http.StatusOK {
		return false, 0, fmt.Errorf("indexer returned error status: %d", resp.StatusCode)
	}

	var res struct {
		Account struct {
			Assets []struct {
				AssetID uint64 `json:"asset-id"`
			} `json:"assets"`
		} `json:"account"`
	}
	if json.NewDecoder(resp.Body).Decode(&res) == nil {
		for _, a := range res.Account.Assets {
			if strconv.FormatUint(a.AssetID, 10) == assetIDStr {
				return true, 0, nil
			}
		}
	}
	return false, 0, nil
}

// SavePersistentCardCache persists the current card inventory to a blockchain snapshot.
// PILLAR 6: Blockchain Persistence.
func (s *OracleService) SavePersistentCardCache(l *Lobby) {
	l.mutex.RLock()
	state := l.persistentCardCache
	l.mutex.RUnlock()
	l.saveBlockchainStateSnapshotLocked(NotePrefixCardCacheSnapshot, state)
}

// LoadPersistentCardCache reconstructs the card cache from blockchain snapshots.
// PILLAR 6: Blockchain Persistence.
func (s *OracleService) LoadPersistentCardCache(l *Lobby) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.persistentCardCache = make(map[int]ServerCard)
	if l.loadBlockchainStateSnapshotLocked(NotePrefixCardCacheSnapshot, &l.persistentCardCache) {
		for k, v := range l.persistentCardCache {
			l.inventory[k] = v
		}
	}
}
