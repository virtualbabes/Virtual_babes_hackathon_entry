package main

import (
	"bytes"
	"context"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/v2/types"
)

const cardCacheName = "card_cache.json"

// indexerRequest executes an HTTP GET request across multiple indexer endpoints with retries.
// PILLAR 4: RPC Failover. Automatically cycles configured endpoints on 429 or 5xx errors.
func (l *Lobby) indexerRequest(cfg NetworkConfig, path string) (*http.Response, error) {
	var lastErr error
	for _, baseURL := range cfg.IndexerURLs {
		url := baseURL + path
		for i := 0; i < 3; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), indexerTimeout)
			req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
			resp, err := http.DefaultClient.Do(req)
			cancel()
			if err != nil {
				lastErr = err
				time.Sleep(500 * time.Millisecond)
				continue
			}
			if resp.StatusCode == http.StatusTooManyRequests {
				resp.Body.Close()
				lastErr = fmt.Errorf("rate-limited (429) at %s", baseURL)
				time.Sleep(time.Duration(i+1) * 1 * time.Second)
				continue
			}
			if resp.StatusCode >= 500 {
				resp.Body.Close()
				lastErr = fmt.Errorf("server error %d at %s", resp.StatusCode, baseURL)
				continue
			}
			return resp, nil
		}
	}
	return nil, fmt.Errorf("indexer request failed after cycling endpoints: %w", lastErr)
}

func (l *Lobby) getVerifiedCards(wallet string, tokenIDs []int, networkName string) (map[int]ServerCard, error) {
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
				accResp, err := l.indexerRequest(cfg, fmt.Sprintf("/v2/accounts/%s", t.addr))
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
							meta, std, err := l.MetadataDispatcher(t.network, int(as.AssetID))
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
				resp, err := l.indexerRequest(cfg, fmt.Sprintf("/tokens?owner=%s", t.addr))
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
							m, s, err := l.MetadataDispatcher(t.network, tok.TokenID)
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
					cfg.IndexerURL, cfg.AssetID, t.addr)

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
		return l.getVerifiedCardsCrossChain(tokenIDs, netConfig)
	}

	// PILLAR 3: Multi-Standard Discovery.
	// We iterate through missing tokens and utilize the MetadataDispatcher to identify 
	// and fetch standard-compliant metadata (ARC-72, ARC-19, or ARC-69).
	for _, id := range toFetch {
		meta, standard, err := l.MetadataDispatcher(networkName, id)
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

// getVerifiedCardsCrossChain handles metadata retrieval for non-Algorand networks (EVM, Solana, etc).
func (l *Lobby) getVerifiedCardsCrossChain(tokenIDs []int, cfg NetworkConfig) (map[int]ServerCard, error) {
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
			url := fmt.Sprintf("%s/api?module=token&action=tokenid_metadata&contractaddress=%s&tokenid=%d",
				cfg.IndexerURL, cfg.AssetID, id)

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

			url := cfg.NodeURL // DAS API is typically accessed via the NodeURL (RPC endpoint)

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
							Metadata struct {
								Name        string `json:"name"`
								Image       string `json:"image"`
								Description string `json:"description"`
							} `json:"metadata"`
						} `json:"content"`
					} `json:"result"`
				}
				if json.NewDecoder(resp.Body).Decode(&dasRes) == nil && dasRes.Result.Content.Metadata.Name != "" {
					newCard = ServerCard{
						ID: id, Name: dasRes.Result.Content.Metadata.Name, Image: dasRes.Result.Content.Metadata.Image,
						Power:         [4]int{cfg.PowerBase, 20, cfg.PowerBase, 20}, // Apply network's base power
						LastUpdated:   time.Now(),
						MetadataValid: true,
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

func (l *Lobby) syncStatsFromBlockchain(clientID, wallet string) {
	l.mutex.RLock()
	voiConfig, ok := l.availableNetworks["Voi Mainnet"]
	vaultAddr := l.vaultAddress
	l.mutex.RUnlock()
	if !ok {
		return
	}

	baseURL := voiConfig.IndexerURL

	// PASS 1: Wins/DNFs (Vault -> Wallet)
	url := fmt.Sprintf("%s/arc200/transfers?contractId=%s&from=%s&to=%s&limit=500",
		baseURL, voiConfig.AssetID, vaultAddr, wallet)

	var resp *http.Response
	var err error
	for i := 0; i < 3; i++ { // Retry up to 3 times
		ctx, cancel := context.WithTimeout(context.Background(), indexerTimeout)
		req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
		resp, err = http.DefaultClient.Do(req)
		cancel() // Ensure context is cancelled after each attempt
		if err != nil {
			if i < 2 { // If not the last attempt, wait and retry
				time.Sleep(500 * time.Millisecond)
				continue
			}
			log.Printf("[ORACLE ERROR] Failed to connect to indexer for stats sync after retries: %v\n", err)
			return // Return on persistent network error
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			if i < 2 { // If not the last attempt, wait with backoff and retry
				time.Sleep(time.Duration(i+1) * 1 * time.Second)
				continue
			}
			log.Printf("[ORACLE ERROR] Indexer rate-limited (429) for stats sync after retries.\n")
			return // Return on persistent rate-limiting
		}
		break // Break loop on successful response
	}
	if err != nil {
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
			if tx.Timestamp < l.seasonStart.Unix() {
				continue
			}
			if strings.HasPrefix(tx.Metadata, "VBT_WIN:") {
				wins++

				// PILLAR 4: Historical Reconstruction. Parse note metadata to rebuild match context.
				var data struct {
					Opp    string `json:"opp"`
					Scores [2]int `json:"scores"`
					TID    string `json:"tid"` // Tournament ID
					MID    string `json:"mid"` // Match ID
				}
				if err := json.Unmarshal([]byte(strings.TrimPrefix(tx.Metadata, "VBT_WIN:")), &data); err == nil {
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
			if strings.HasPrefix(tx.Metadata, "VBT_DNF:") {
				dnfs++
			}
		}
	}

	// PILLAR 4: Mirrored Immersion.
	// Pass 3: Global Result Recovery. Scan the Vault's output to find matches where I was the Loser.
	// This allows reconstructing persistent "Loss" records without extra blockchain fees.
	globalURL := fmt.Sprintf("%s/arc200/transfers?contractId=%s&from=%s&limit=200",
		baseURL, voiConfig.AssetID, vaultAddr)

	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), indexerTimeout)
		req, _ := http.NewRequestWithContext(ctx, "GET", globalURL, nil)
		resp, err = http.DefaultClient.Do(req)
		cancel()
		if err == nil && resp.StatusCode == http.StatusOK {
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
					if tx.Timestamp < l.seasonStart.Unix() {
						continue
					}

					if strings.HasPrefix(tx.Metadata, "VBT_WIN:") {
						var data struct {
							Opp    string `json:"opp"`
							Scores [2]int `json:"scores"`
							TID    string `json:"tid"` // Tournament ID
							MID    string `json:"mid"` // Match ID
						}
						// If I am the 'Opponent', I lost this match. Add to history as Loss (WinnerIndex: 1).
						if err := json.Unmarshal([]byte(strings.TrimPrefix(tx.Metadata, "VBT_WIN:")), &data); err == nil {
							if strings.EqualFold(data.Opp, wallet) {
								matchID := data.MID
								if matchID == "" {
									matchID = data.TID
								} // Legacy fallback
								tournID := data.TID
								if data.MID == "" {
									tournID = ""
								} // If MID is empty, TID was MatchID

								matchHistory = append(matchHistory, MatchHistory{
									Opponent:          tx.To, // The person who received the win transaction
									Scores:            data.Scores,
									TournamentID:      tournID,
									TournamentMatchID: matchID,
									ReceiptTxID:       tx.TransactionID,
									Timestamp:         time.Unix(tx.Timestamp, 0),
									WinnerIndex:       1, // Mirror record: relative Loss
								})
							}
						}
					} else if strings.HasPrefix(tx.Metadata, "VBT_DNF:") {
						var data struct {
							Leaver string `json:"leaver"`
							Opp    string `json:"opp"`
							TID    string `json:"tid"`
						}
						if err := json.Unmarshal([]byte(strings.TrimPrefix(tx.Metadata, "VBT_DNF:")), &data); err == nil {
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
	resp, err = l.indexerRequest(voiConfig, fmt.Sprintf("/arc200/transfers?contractId=%s&from=%s&to=%s&limit=500",
		voiConfig.AssetID, wallet, vaultAddr))

	if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
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
					if strings.HasPrefix(tx.Metadata, "VBT_TOURN_BUYIN:") || strings.HasPrefix(tx.Metadata, "ARENA_TOURN_BUYIN:") {
						txTime := time.Unix(tx.Timestamp, 0)
						l.registeredTxIDs[tx.TransactionID] = txTime

						// PILLAR 3: Registration Reconstruction.
						// Use TournamentID for precise reconstruction if available in note
						parts := strings.Split(tx.Metadata, ":")
						matchesCurrent := false
						if len(parts) >= 2 && parts[1] == l.tournament.ID {
							matchesCurrent = true
						} else if txTime.After(l.tournament.OpenTime) {
							matchesCurrent = true // Legacy fallback
						}

						if l.tournament.Active && l.tournament.CurrentRound == 0 && matchesCurrent {
							if !l.isWalletRegistered(wallet) {
								l.paidParticipants = append(l.paidParticipants, wallet)
								log.Printf("[ORACLE] Reconstructed tournament entry for %s (Tx: %s)\n", wallet, tx.TransactionID)
							}
						}
					}
				}
				l.mutex.Unlock()
			}
		}
	}
}

func (l *Lobby) refreshGlobalLeaderboard() {
	l.mutex.RLock()
	voiConfig, ok := l.availableNetworks["Voi Mainnet"]
	l.mutex.RUnlock()
	if !ok {
		return
	}

	url := fmt.Sprintf("%s/arc200/transfers?contractId=%s&from=%s&limit=1000",
		voiConfig.IndexerURL, voiConfig.AssetID, l.vaultAddress)

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
			log.Printf("[ORACLE ERROR] Failed to connect to indexer for leaderboard refresh after retries: %v\n", err)
			return
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			if i < 2 {
				time.Sleep(time.Duration(i+1) * 1 * time.Second)
				continue
			}
			log.Printf("[ORACLE ERROR] Indexer rate-limited (429) for leaderboard refresh after retries.\n")
			return
		}
		break
	}
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
			if strings.HasPrefix(tx.Metadata, "VBT_WIN:") {
				data[tx.To].wins++
			}
			if strings.HasPrefix(tx.Metadata, "VBT_DNF:") {
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

// loadOnboardedWalletsFromIndexer reconstructs the historical Sybil protection state.
// It scans the indexer for past onboarding transactions from the vault address.
func (l *Lobby) loadOnboardedWalletsFromIndexer() {
	l.mutex.RLock()
	voiConfig, ok := l.availableNetworks["Voi Mainnet"]
	vaultAddr := l.vaultAddress
	rewardAsset := l.rewardAssetID
	l.mutex.RUnlock()

	if !ok || vaultAddr == "" {
		log.Println("[ORACLE ERROR] Cannot load onboarded wallets: Network config or Vault address missing.")
		return
	}

	log.Printf("[ORACLE] Reconstructing onboarding history from %s...\n", voiConfig.IndexerURL)

	// Implementation: Paged scan to ensure completeness
	limit := 1000
	offset := 0
	totalRestored := 0

	for {
		url := fmt.Sprintf("%s/arc200/transfers?contractId=%s&from=%s&limit=%d&offset=%d",
			voiConfig.IndexerURL, rewardAsset, vaultAddr, limit, offset)

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
				log.Printf("[ORACLE ERROR] Indexer connection failed during onboarding sync after retries: %v\n", err)
				return
			}
			if resp.StatusCode == http.StatusTooManyRequests {
				resp.Body.Close()
				if i < 2 {
					time.Sleep(time.Duration(i+1) * 1 * time.Second)
					continue
				}
				log.Printf("[ORACLE ERROR] Indexer rate-limited (429) during onboarding sync after retries.\n")
				return
			}
			break
		}

		if err != nil {
			return // Keep SybilSyncComplete as false
		}
		if resp.StatusCode != http.StatusOK {
			log.Printf("[ORACLE ERROR] Indexer returned non-200 status during onboarding sync: %d %s\n", resp.StatusCode, resp.Status)
			resp.Body.Close()
			return // Critical failure: stop and keep SybilSyncComplete as false
		}

		var res struct {
			Transfers []struct {
				To       string `json:"to"`
				Metadata string `json:"metadata"`
			} `json:"transfers"`
		}

		decodeErr := json.NewDecoder(resp.Body).Decode(&res)
		resp.Body.Close()

		if decodeErr != nil {
			log.Printf("[ORACLE ERROR] Failed to decode indexer response during onboarding sync: %v\n", decodeErr)
			return
		}

		if len(res.Transfers) == 0 {
			break
		}

		l.mutex.Lock()
		for _, tx := range res.Transfers {
			if strings.HasPrefix(tx.Metadata, "VBT_ONBOARD:TOKEN") {
				l.onboardedWallets[strings.ToLower(tx.To)] = true
				totalRestored++
			}
		}
		l.mutex.Unlock()

		if len(res.Transfers) < limit {
			break
		}
		offset += limit
	}

	l.mutex.Lock()
	l.SybilSyncComplete = true
	l.mutex.Unlock()
	log.Printf("[ORACLE] Successfully restored %d historical onboarding records.\n", totalRestored)
}

// ResolveEnvoiName attempts to find a .voi or .algo name for a wallet address.
// It utilizes a dedicated lock and local cache to minimize indexer traffic and avoid deadlocks.
func (l *Lobby) ResolveEnvoiName(address string) string {
	if address == "" || address == "TBD" || address == "BYE" {
		return address
	}

	l.envoiMutex.RLock()
	if name, ok := l.envoiCache[address]; ok {
		l.envoiMutex.RUnlock()
		return name
	}
	l.envoiMutex.RUnlock()

	// Basic Truncation fallback
	truncated := address[:6] + "..." + address[len(address)-4:]

	// Optimization: Fetch Indexer URL once under a brief lock to prevent recursive deadlock
	var baseURL string
	l.mutex.RLock()
	if cfg, ok := l.availableNetworks["Voi Mainnet"]; ok {
		baseURL = cfg.IndexerURL
	}
	l.mutex.RUnlock()

	if baseURL == "" {
		return truncated
	}

	url := fmt.Sprintf("%s/tokens?owner=%s", baseURL, address)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
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

func (l *Lobby) verifyBuyInTransaction(network, txid string, expectedAmt uint64, expectedAsset, sender, vaultAddr, expectedNotePrefix string) (bool, int64, error) {
	// 1. Authoritative Network Key Resolution (Deterministic Case Sync)
	netKey := l.mapChainToNetworkName(network)
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
		// VOI Logic: Custom ARC-200 Indexer
		url := fmt.Sprintf("%s/arc200/transfers?transactionId=%s", netConfig.IndexerURL, txid)
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
				return false, 0, err
			}
			if resp.StatusCode == http.StatusTooManyRequests {
				resp.Body.Close()
				if i < 2 {
					time.Sleep(time.Duration(i+1) * 1 * time.Second)
					continue
				}
				return false, 0, fmt.Errorf("voi indexer rate-limited (429)")
			}
			break
		}
		if err != nil {
			return false, 0, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return false, 0, fmt.Errorf("voi indexer returned non-200 status: %d", resp.StatusCode)
		}

		var res struct {
			Transfers []struct {
				From, To, Amount, Metadata string
				ContractID                 uint64
				Timestamp                  int64
			} `json:"transfers"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err == nil {
			for _, tx := range res.Transfers {
				amt, _ := strconv.ParseUint(tx.Amount, 10, 64)
				// SECURITY: Verify exact note prefix to prevent cross-purpose payment replays
				if strings.EqualFold(tx.From, sender) && strings.EqualFold(tx.To, vaultAddr) && amt >= expectedAmt && strconv.FormatUint(tx.ContractID, 10) == targetAsset && strings.HasPrefix(tx.Metadata, expectedNotePrefix) {
					return true, tx.Timestamp, nil
				}
			}
		}
	} else {
		// ALGORAND Logic: Standard Indexer Transaction Endpoint
		url := fmt.Sprintf("%s/v2/transactions/%s", netConfig.IndexerURL, txid)
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
				return false, 0, err
			}
			if resp.StatusCode == http.StatusTooManyRequests {
				resp.Body.Close()
				if i < 2 {
					time.Sleep(time.Duration(i+1) * 1 * time.Second)
					continue
				}
				return false, 0, fmt.Errorf("algorand indexer rate-limited (429)")
			}
			break
		}
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

// fetchARC69Metadata retrieves metadata from the latest configuration transaction note.
func (l *Lobby) fetchARC69Metadata(cfg NetworkConfig, assetID int) (*ARC72Metadata, error) {
	resp, err := l.indexerRequest(cfg, fmt.Sprintf("/v2/assets/%d/transactions?tx-type=acfg&limit=1", assetID))
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
func (l *Lobby) fetchARC19Metadata(cfg NetworkConfig, assetID int) (*ARC72Metadata, error) {
	// 1. Fetch Asset Information from Indexer to retrieve the Reserve Address
	resp, err := l.indexerRequest(cfg, fmt.Sprintf("/v2/assets/%d", assetID))
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
	ipfsURL := "https://ipfs.io/ipfs/" + cid
	var ipfsResp *http.Response
	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), indexerTimeout)
		req, _ := http.NewRequestWithContext(ctx, "GET", ipfsURL, nil)
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
func (l *Lobby) MetadataDispatcher(networkName string, assetID int) (*ARC72Metadata, string, error) {
	l.mutex.RLock()
	cfg, ok := l.availableNetworks[networkName]
	l.mutex.RUnlock()
	if !ok {
		return nil, "", fmt.Errorf("unsupported network for metadata dispatch: %s", networkName)
	}

	// 1. ARC-19 Detection: Fetch Asset parameters from Indexer to check for template URL.
	// This is the most efficient first check for dynamic ASAs.
	url := fmt.Sprintf("%s/v2/assets/%d", cfg.IndexerURL, assetID)
	
	ctx, cancel := context.WithTimeout(context.Background(), indexerTimeout)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	cancel()

	if err == nil && resp.StatusCode == http.StatusOK {
		var res struct {
			Asset struct {
				Params struct {
					URL string `json:"url"`
				} `json:"params"`
			} `json:"asset"`
		}
		if json.NewDecoder(resp.Body).Decode(&res) == nil {
			resp.Body.Close()
			if strings.Contains(res.Asset.Params.URL, "template-ipfs") {
				meta, err := l.fetchARC19Metadata(cfg.IndexerURL, assetID)
				return meta, "ARC-19", err
			}
		} else {
			resp.Body.Close()
		}
	} else if resp != nil {
		resp.Body.Close()
	}

	// 2. ARC-72 Check: If network has a configured AppID, check if this ID exists as a token.
	if cfg.AppID != "" && cfg.AppID != "0" {
		checkURL := fmt.Sprintf("%s/tokens?contractId=%s&tokenId=%d", cfg.IndexerURL, cfg.AppID, assetID)
		ctx72, cancel72 := context.WithTimeout(context.Background(), indexerTimeout)
		req72, _ := http.NewRequestWithContext(ctx72, "GET", checkURL, nil)
		resp72, err := http.DefaultClient.Do(req72)
		cancel72()
		if err == nil && resp72.StatusCode == http.StatusOK {
			var res72 struct {
				Tokens []struct { Metadata string `json:"metadata"` } `json:"tokens"`
			}
			if json.NewDecoder(resp72.Body).Decode(&res72) == nil && len(res72.Tokens) > 0 {
				resp72.Body.Close()
				var meta ARC72Metadata
				json.Unmarshal([]byte(res72.Tokens[0].Metadata), &meta)
				return &meta, "ARC-72", nil
			}
			resp72.Body.Close()
		}
	}

	// 3. Fallback to ARC-69: Scan configuration history for JSON notes.
	meta, err := l.fetchARC69Metadata(cfg.IndexerURL, assetID)
	return meta, "ARC-69", err
}

// checkVaultBalanceOnChain synchronizes the internal faucetBalance with the on-chain $VBV pool.
func (l *Lobby) checkVaultBalanceOnChain() {
	l.mutex.RLock()
	voiConfig, ok := l.availableNetworks["Voi Mainnet"]
	rewardAppIDStr := l.rewardAssetID
	vaultAddr := l.vaultAddress
	l.mutex.RUnlock()

	if !ok || rewardAppIDStr == "" || vaultAddr == "" {
		return
	}

	rewardAppID, err := strconv.ParseUint(rewardAppIDStr, 10, 64)
	if err != nil {
		return
	}

	client, _ := algod.MakeClient(voiConfig.NodeURL, "")
	addrObj, _ := types.DecodeAddress(vaultAddr)

	// ARC-200 Balance is stored in an application box named by the account's public key bytes.
	var boxValue []byte
	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), indexerTimeout)
		boxResp, err := client.GetApplicationBoxByName(rewardAppID, addrObj[:]).Do(ctx)
		cancel()
		if err != nil {
			// If not found, vault is empty or not initialized
			if strings.Contains(err.Error(), "404") || strings.Contains(strings.ToLower(err.Error()), "not found") {
				log.Printf("[ORACLE] Note: Vault has no $VBV balance box yet (Asset: %s).\n", rewardAppIDStr)
				return
			}
			// Handle Node rate-limiting (429)
			if strings.Contains(err.Error(), "429") {
				if i < 2 {
					time.Sleep(time.Duration(i+1) * 1 * time.Second)
					continue
				}
			}
			// Transient network errors
			if i < 2 {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			log.Printf("[ORACLE ERROR] Failed to fetch vault box balance after retries: %v\n", err)
			return
		}
		boxValue = boxResp.Value
		break
	}

	// ARC-200 balances are 32-byte uint256 values
	if len(boxValue) >= 32 {
		bal := new(big.Int).SetBytes(boxValue[:32]).Uint64()
		l.mutex.Lock()
		l.faucetBalance = float64(bal) / 1000000.0
		l.applyDynamicScalingLocked() // Adjust reward amounts based on new liquidity level
		l.mutex.Unlock()
		log.Printf("[ORACLE] Vault $VBV Pool Synced: %.2f units.\n", l.faucetBalance)
	}
}

func (l *Lobby) checkNativeVaultBalanceOnChain() {
	l.mutex.RLock()
	voiConfig, _ := l.availableNetworks["Voi Mainnet"]
	l.mutex.RUnlock()

	for _, nodeURL := range voiConfig.NodeURLs {
		client, _ := algod.MakeClient(nodeURL, "")
		var amount uint64
		success := false

		for i := 0; i < 3; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), indexerTimeout)
			info, err := client.AccountInformation(l.vaultAddress).Do(ctx)
			cancel()
			if err != nil {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			amount = info.Amount
			success = true
			break
		}

		if success {
			l.mutex.Lock()
			if amount < 1000000 {
				log.Printf("[CRITICAL] Vault low on gas at %s! Balance: %d", nodeURL, amount)
				l.broadcastToAdmins("⚠️ <b>CRITICAL:</b> Vault gas is nearly depleted.")
			}
			l.faucetBalance = float64(amount) / 1000000.0
			l.mutex.Unlock()
			return // Success
		}
	}
}

// savePersistentCardCache persists the current card inventory to a JSON file.
func (l *Lobby) savePersistentCardCache() {
	l.mutex.RLock()
	cache := make(map[int]ServerCard, len(l.persistentCardCache))
	for id, card := range l.persistentCardCache {
		cache[id] = card
	}
	l.mutex.RUnlock()

	data, err := json.Marshal(cache)
	if err != nil {
		log.Printf("[CACHE] Failed to marshal card cache: %v\n", err)
		return
	}
	if err := os.WriteFile(l.getDataPath(cardCacheName), data, 0644); err != nil {
		log.Printf("[CACHE] Failed to write card cache file: %v\n", err)
	}
}

// handleSeasonHistory fetches archived seasonal standings from the blockchain.
func (l *Lobby) handleSeasonHistory(w http.ResponseWriter, r *http.Request) {
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
	baseURL := voiConfig.IndexerURL // Use the authoritative Indexer URL from config

	// HARDENING: Added limit to prevent massive indexer payloads
	// Query transfers where the vault sent 0 to itself to record season history
	url := fmt.Sprintf("%s/arc200/transfers?contractId=%s&from=%s&to=%s&limit=100", baseURL, rewardAssetID, faucetAddr, faucetAddr)

	ctx, cancel := context.WithTimeout(context.Background(), indexerTimeout)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "Failed to connect to indexer", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var res struct {
		Transfers []struct {
			Metadata string `json:"metadata"`
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
			if strings.HasPrefix(tx.Metadata, "VBT_SEASON_ARCHIVE:") {
				jsonStr := strings.TrimPrefix(tx.Metadata, "VBT_SEASON_ARCHIVE:")
				var archive SeasonArchive
				if err := json.Unmarshal([]byte(jsonStr), &archive); err == nil {
					// Only include if no specific season was requested, or if it matches the target
					if targetSeason == -1 || archive.Season == targetSeason {
						uniqueSeasons[archive.Season] = archive
					}
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
func (l *Lobby) handleReSyncStats(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Wallet string `json:"wallet"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Wallet == "" {
		http.Error(w, "Invalid wallet", http.StatusBadRequest)
		return
	}
	go l.syncStatsFromBlockchain("UI_TRIGGER", req.Wallet)
	json.NewEncoder(w).Encode(map[string]string{"status": "sync_initiated"})
}

// mapChainToNetworkName translates frontend chain codes to internal NetworkConfig keys.
func (l *Lobby) mapChainToNetworkName(chain string) string {
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
func (l *Lobby) checkAssetOptIn(network, wallet string, assetIDStr string) (bool, int64, error) {
	if assetIDStr == "" || assetIDStr == "0" {
		return true, 0, nil
	}

	// 1. Authoritative Network Key Resolution (Deterministic Case Sync)
	netKey := l.mapChainToNetworkName(network)
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
		var lastErr error

		for _, nodeURL := range netConfig.NodeURLs {
			client, _ := algod.MakeClient(nodeURL, "")
			for i := 0; i < 3; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_, err := client.GetApplicationBoxByName(assetID, addr[:]).Do(ctx)
				cancel()
				if err != nil {
					if strings.Contains(err.Error(), "404") || strings.Contains(strings.ToLower(err.Error()), "not found") {
						return false, 0, nil // Definitively not opted in
					}
					lastErr = err
					time.Sleep(500 * time.Millisecond)
					continue
				}
				return true, 0, nil
			}
		}
		return false, 0, fmt.Errorf("voi node failover failed: %w", lastErr)
	}

	// 3. ALGORAND / ASA Pattern: Indexer Account Asset Scan
	resp, err := l.indexerRequest(netConfig, fmt.Sprintf("/v2/accounts/%s", wallet))
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
