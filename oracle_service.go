package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/v2/types"
)

const cardCacheFileName = "card_cache.json"

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
				log.Printf("[ORACLE] Syncing tokens for %s on %s...\n", t.addr, t.network)
				url := fmt.Sprintf("%s/tokens?owner=%s", cfg.IndexerURL, t.addr)

				ctx, cancel := context.WithTimeout(context.Background(), indexerTimeout)
				req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
				resp, err := http.DefaultClient.Do(req)

				if err == nil && resp.StatusCode == http.StatusOK {
					var res struct {
						Tokens []struct {
							TokenID  int    `json:"tokenId"`
							Metadata string `json:"metadata"`
						} `json:"tokens"`
					}
					if json.NewDecoder(resp.Body).Decode(&res) == nil {
						for _, tok := range res.Tokens {
							var meta ARC72Metadata
							if json.Unmarshal([]byte(tok.Metadata), &meta) == nil {
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
							}
						}
					}
					resp.Body.Close()
				}
				cancel()
			} else if strings.HasPrefix(cfg.ChainID, "eip155") {
				// EVM Discovery logic: Query Etherscan NFT transfer history for ownership patterns
				log.Printf("[ORACLE] Syncing EVM tokens for %s on %s...\n", t.addr, t.network)
				url := fmt.Sprintf("%s/api?module=account&action=tokennfttx&contractaddress=%s&address=%s&sort=desc",
					cfg.IndexerURL, cfg.AssetID, t.addr)

				ctx, cancel := context.WithTimeout(context.Background(), indexerTimeout)
				req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
				resp, err := http.DefaultClient.Do(req)

				if err == nil && resp.StatusCode == http.StatusOK {
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
				cancel()
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

	baseURL := netConfig.IndexerURL
	contractID := netConfig.AppID

	type metaResult struct {
		mintRound   int
		name, image string
		exists      bool
	}
	tokenMeta := make(map[int]metaResult)

	ids := make([]string, len(toFetch))
	for i, id := range toFetch {
		ids[i] = strconv.Itoa(id)
	}
	url := fmt.Sprintf("%s/tokens?contractId=%s&tokenId=%s", baseURL, contractID, strings.Join(ids, ","))

	ctx, cancel := context.WithTimeout(context.Background(), indexerTimeout)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		var res struct {
			Tokens []struct {
				TokenID   int    `json:"tokenId"`
				MintRound int    `json:"mintRound"`
				Metadata  string `json:"metadata"`
			} `json:"tokens"`
		}
		if json.NewDecoder(resp.Body).Decode(&res) == nil {
			for _, t := range res.Tokens {
				var meta ARC72Metadata
				json.Unmarshal([]byte(t.Metadata), &meta)
				tokenMeta[t.TokenID] = metaResult{mintRound: t.MintRound, name: meta.Name, image: meta.Image, exists: true}
			}
		}
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()
	for _, id := range toFetch {
		meta := tokenMeta[id]
		newCard := ServerCard{
			ID: id, Name: meta.name, Image: meta.image,
			Power:       [4]int{netConfig.PowerBase, 10, netConfig.PowerBase, 10},
			LastUpdated: time.Now(),
		}
		l.inventory[id] = newCard
		results[id] = newCard
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

			ctx, cancel := context.WithTimeout(context.Background(), indexerTimeout)
			req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
			resp, err := http.DefaultClient.Do(req)

			if err == nil && resp.StatusCode == http.StatusOK {
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
			}
			cancel()
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

			ctx, cancel := context.WithTimeout(context.Background(), indexerTimeout)
			req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)

			if err == nil && resp.StatusCode == http.StatusOK {
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
			} else {
				log.Printf("[ORACLE] Solana DAS 'getAsset' request failed for %s #%d. Error: %v, Status: %d", cfg.NetworkName, id, err, resp.StatusCode)
			}
			cancel()

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
	l.mutex.RUnlock()
	if !ok {
		return
	}

	baseURL := voiConfig.IndexerURL
	url := fmt.Sprintf("%s/arc200/transfers?contractId=%s&from=%s&to=%s&limit=500",
		baseURL, voiConfig.AssetID, l.vaultAddress, wallet)

	ctx, cancel := context.WithTimeout(context.Background(), indexerTimeout)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var res struct {
		Transfers []struct {
			Metadata  string `json:"metadata"`
			Timestamp int64  `json:"timestamp"`
		} `json:"transfers"`
	}

	wins, dnfs := 0, 0
	if json.NewDecoder(resp.Body).Decode(&res) == nil {
		for _, tx := range res.Transfers {
			if tx.Timestamp < l.seasonStart.Unix() {
				continue
			}
			if strings.HasPrefix(tx.Metadata, "VBT_WIN:") {
				wins++
			}
			if strings.HasPrefix(tx.Metadata, "VBT_DNF:") {
				dnfs++
			}
		}
	}

	l.mutex.Lock()
	l.ensurePlayerStatsMapsInitialized(wallet) // Ensure maps are initialized
	stats := l.leaderboard[wallet]
	stats.Wins, stats.DNFs = wins, dnfs // Update raw wins/dnfs
	stats.Reputation = l.CalculateReputation(stats)
	l.leaderboard[wallet] = stats
	l.mutex.Unlock()
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

	ctx, cancel := context.WithTimeout(context.Background(), indexerTimeout)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

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

	// Query ARC-200 transfers sent FROM the vault
	url := fmt.Sprintf("%s/arc200/transfers?contractId=%s&from=%s&limit=1000",
		voiConfig.IndexerURL, rewardAsset, vaultAddr)

	ctx, cancel := context.WithTimeout(context.Background(), indexerTimeout)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[ORACLE ERROR] Indexer connection failed during onboarding sync: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var res struct {
		Transfers []struct {
			To       string `json:"to"`
			Metadata string `json:"metadata"`
		} `json:"transfers"`
	}

	if json.NewDecoder(resp.Body).Decode(&res) == nil {
		l.mutex.Lock()
		for _, tx := range res.Transfers {
			if strings.HasPrefix(tx.Metadata, "VBT_ONBOARD:TOKEN") {
				l.onboardedWallets[tx.To] = true
			}
		}
		l.mutex.Unlock()
		log.Printf("[ORACLE] Successfully restored %d historical onboarding records.\n", len(l.onboardedWallets))
	}
}

func (l *Lobby) verifyBuyInTransaction(network, txid string, expectedAmt uint64, expectedAsset string, sender, vaultAddr string) (bool, int64, error) {
	l.mutex.RLock()
	netConfig, ok := l.availableNetworks[network+" Mainnet"]
	l.mutex.RUnlock()
	if !ok {
		return false, 0, fmt.Errorf("network error")
	}

	// Branch logic based on Network Type
	if strings.Contains(strings.ToLower(network), "voi") {
		// VOI Logic: Custom ARC-200 Indexer
		url := fmt.Sprintf("%s/arc200/transfers?transactionId=%s", netConfig.IndexerURL, txid)
		ctx, cancel := context.WithTimeout(context.Background(), indexerTimeout)
		defer cancel()
		req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return false, 0, err
		}
		defer resp.Body.Close()

		var res struct {
			Transfers []struct {
				From, To, Amount string
				ContractID       uint64
				Timestamp        int64
			} `json:"transfers"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err == nil {
			for _, tx := range res.Transfers {
				amt, _ := strconv.ParseUint(tx.Amount, 10, 64)
				if strings.EqualFold(tx.From, sender) && strings.EqualFold(tx.To, vaultAddr) && amt >= expectedAmt && strconv.FormatUint(tx.ContractID, 10) == expectedAsset {
					return true, tx.Timestamp, nil
				}
			}
		}
	} else {
		// ALGORAND Logic: Standard Indexer Transaction Endpoint
		url := fmt.Sprintf("%s/v2/transactions/%s", netConfig.IndexerURL, txid)
		ctx, cancel := context.WithTimeout(context.Background(), indexerTimeout)
		defer cancel()
		req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return false, 0, err
		}
		defer resp.Body.Close()

		var res struct {
			Transaction struct {
				AssetTransfer struct {
					Receiver string `json:"receiver"`
					Amount   uint64 `json:"amount"`
					AssetID  uint64 `json:"asset-id"`
				} `json:"asset-transfer-transaction"`
				Sender    string `json:"sender"`
				RoundTime int64  `json:"round-time"`
			} `json:"transaction"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err == nil {
			t := res.Transaction
			if strings.EqualFold(t.Sender, sender) && strings.EqualFold(t.AssetTransfer.Receiver, vaultAddr) && t.AssetTransfer.Amount >= expectedAmt && strconv.FormatUint(t.AssetTransfer.AssetID, 10) == expectedAsset {
				return true, t.RoundTime, nil
			}
		}
	}
	return false, 0, nil
}

func (l *Lobby) checkNativeVaultBalanceOnChain() {
	l.mutex.RLock()
	voiConfig, _ := l.availableNetworks["Voi Mainnet"]
	l.mutex.RUnlock()
	client, _ := algod.MakeClient(voiConfig.NodeURL, "")
	info, _ := client.AccountInformation(l.vaultAddress).Do(context.Background())
	l.mutex.Lock()
	l.faucetBalance = float64(info.Amount) / 1000000.0
	l.mutex.Unlock()
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
	if err := os.WriteFile(cardCacheFileName, data, 0644); err != nil {
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
		Season int       `json:"season"`
		Start  time.Time `json:"start"`
		End    time.Time `json:"end"`
		Top    []struct {
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
	l.mutex.RLock()
	netConfig, _ := l.availableNetworks[network+" Mainnet"]
	l.mutex.RUnlock()
	client, _ := algod.MakeClient(netConfig.NodeURL, "")
	if network == "VOI" {
		assetID, _ := strconv.ParseUint(assetIDStr, 10, 64)
		addr, _ := types.DecodeAddress(wallet)
		_, err := client.GetApplicationBoxByName(assetID, addr[:]).Do(context.Background())
		return err == nil, 0, nil
	}
	url := fmt.Sprintf("%s/v2/accounts/%s", netConfig.IndexerURL, wallet)
	ctx, cancel := context.WithTimeout(context.Background(), indexerTimeout)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()
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
