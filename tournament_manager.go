package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/mnemonic"
	"github.com/algorand/go-algorand-sdk/v2/transaction"
	"github.com/algorand/go-algorand-sdk/v2/types"
)

const regCacheName = "registrations.json"

// tournamentCacheEntry stores the serialized JSON and its associated query key.
type tournamentCacheEntry struct {
	key  string
	data []byte
}

func (l *Lobby) handleTournamentRegister(w http.ResponseWriter, r *http.Request) {
	l.mutex.RLock()
	voiConfig, ok := l.availableNetworks["Voi Mainnet"]
	l.mutex.RUnlock()
	if !ok {
		http.Error(w, "Voi Mainnet configuration not found", http.StatusInternalServerError)
		return
	}
	var req struct {
		Wallet  string `json:"wallet"`
		TxID    string `json:"txid,omitempty"`
		Network string `json:"network,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Wallet == "" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	targetWallet := strings.ToLower(req.Wallet)

	l.mutex.RLock()
	if !l.tournament.Active || l.tournament.CurrentRound != 0 {
		l.mutex.RUnlock()
		http.Error(w, "Registration is currently closed", http.StatusForbidden)
		return
	}
	if l.isWalletRegistered(targetWallet) {
		l.mutex.RUnlock()
		http.Error(w, "Wallet already registered", http.StatusForbidden)
		return
	}

	// 0. Verification Throttling: Check if already processing or TxID recycled
	if _, isProcessing := l.processingRegistrations[targetWallet]; isProcessing {
		l.mutex.RUnlock()
		http.Error(w, "Registration already in progress for this wallet", http.StatusConflict)
		return
	}
	if _, isUsed := l.registeredTxIDs[req.TxID]; isUsed && req.TxID != "" {
		l.mutex.RUnlock()
		http.Error(w, "Transaction ID already utilized for another entry", http.StatusConflict)
		return
	}
	l.mutex.RUnlock()

	// Lock to mark as processing to block rapid repeat clicks
	l.mutex.Lock()
	l.processingRegistrations[targetWallet] = time.Now()
	l.mutex.Unlock()

	// Ensure processing status is cleared after attempt
	defer func() {
		l.mutex.Lock()
		delete(l.processingRegistrations, targetWallet)
		l.mutex.Unlock()
	}()

	openTime := l.tournament.OpenTime
	buyInAmt := l.tournament.BuyInAmount

	// Identify Elite Status
	l.mutex.RLock()
	type entry struct {
		wallet string
		wins   int
	}
	var hof []entry
	for wallet, stats := range l.leaderboard {
		hof = append(hof, entry{wallet: wallet, wins: stats.Wins})
	}
	l.mutex.RUnlock()
	sort.Slice(hof, func(i, j int) bool { return hof[i].wins > hof[j].wins })
	isElite := false
	for i := 0; i < len(hof) && i < 10; i++ {
		if hof[i].wallet == req.Wallet {
			isElite = true
			break
		}
	}

	var actualRegistrationTime time.Time

	if !isElite {
		if req.TxID == "" {
			http.Error(w, "TxID required", http.StatusBadRequest)
			return
		}

		// STRICT ENFORCEMENT: Only VOI and ALGO networks support tournament buy-ins
		if req.Network != "VOI" && req.Network != "ALGO" {
			http.Error(w, "Tournament registration payments are only accepted on Voi or Algorand", http.StatusBadRequest)
			return
		}

		buyInAsset := voiConfig.AssetID
		if buyInAsset == "" {
			buyInAsset = voiConfig.AppID
		}
		verifyNetwork := "Voi"

		if req.Network == "ALGO" {
			l.mutex.RLock()
			algoCfg, hasAlgo := l.availableNetworks["Algorand Mainnet"]
			l.mutex.RUnlock()
			if hasAlgo && algoCfg.AssetID != "" {
				buyInAsset = algoCfg.AssetID
			} else {
				buyInAsset = l.avoiAssetID
			}
			verifyNetwork = "Algorand"
		}

		// PILLAR 3: Dynamic Precision.
		// Fetch specific network config to get the correct micro-unit divisor for the buy-in asset.
		l.mutex.RLock()
		netCfg, hasCfg := l.availableNetworks[verifyNetwork+" Mainnet"]
		l.mutex.RUnlock()

		divisor := 1000000.0 // Fallback to standard 6 decimals (VBV/AVoi)
		if hasCfg && netCfg.PowerDivisor > 0 {
			divisor = netCfg.PowerDivisor
		}

		// PILLAR 3: Concurrency Throttling.
		// Limit simultaneous indexer requests to prevent rate-limiting during burst registration.
		select {
		case l.oracleSemaphore <- struct{}{}:
			// Acquired slot, proceed to oracle
		case <-time.After(15 * time.Second):
			http.Error(w, "Arena Indexer busy. Please try again in a few moments.", http.StatusServiceUnavailable)
			return
		}
		defer func() { <-l.oracleSemaphore }()

		// verifyBuyInTransaction expects a prefix that matches "Network Mainnet" keys
		verified, txUnixTime, err := l.verifyBuyInTransaction(verifyNetwork, req.TxID, uint64(buyInAmt*divisor), buyInAsset, targetWallet, l.vaultAddress)
		if err != nil || !verified || txUnixTime < openTime.Unix() {
			log.Printf("[TOURNAMENT] Verification failed for %s on %s. Error: %v\n", targetWallet, verifyNetwork, err)
			msg := "Payment verification failed or transaction too old"
			if err != nil && strings.Contains(err.Error(), "429") {
				msg = "External Indexer rate-limited. Please retry."
			}
			http.Error(w, msg, http.StatusPaymentRequired)
			return
		}
		actualRegistrationTime = time.Unix(txUnixTime, 0)

		// Process Club Kickback (Tournament Revenue Loop) using the correct network precision
		l.distributeTournamentKickback(req.Wallet, uint64(buyInAmt*divisor), actualRegistrationTime, verifyNetwork)
	}

	l.mutex.Lock()
	l.paidParticipants = append(l.paidParticipants, targetWallet)
	if !isElite {
		l.registeredTxIDs[req.TxID] = time.Now()
		l.faucetBalance += (buyInAmt / 2.0)
		l.tournamentPotBonus += (buyInAmt / 2.0)
	}
	l.mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "is_elite": isElite})
}

func (l *Lobby) handleTournamentHistory(w http.ResponseWriter, r *http.Request) {
	l.mutex.RLock()
	voiConfig, _ := l.availableNetworks["Voi Mainnet"]
	l.mutex.RUnlock()

	url := fmt.Sprintf("%s/arc200/transfers?contractId=%s&from=%s&to=%s&limit=500",
		voiConfig.IndexerURL, voiConfig.AssetID, l.vaultAddress, l.vaultAddress)

	// PILLAR 3: Concurrency Throttling.
	// Protect the indexer from redundant history requests during peak traffic.
	select {
	case l.oracleSemaphore <- struct{}{}:
		// Acquired slot
	case <-time.After(5 * time.Second):
		http.Error(w, "Arena Indexer is busy. Please try again later.", http.StatusServiceUnavailable)
		return
	}
	defer func() { <-l.oracleSemaphore }()

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
			http.Error(w, "Failed to connect to indexer", http.StatusInternalServerError)
			return
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			if i < 2 {
				time.Sleep(time.Duration(i+1) * 1 * time.Second)
				continue
			}
			http.Error(w, "Indexer rate-limited (429)", http.StatusTooManyRequests)
			return
		}
		break
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("Indexer returned non-200 status: %d", resp.StatusCode), http.StatusInternalServerError)
		return
	}
	var res struct {
		Transfers []struct {
			TransactionID string `json:"transactionId"`
			Metadata      string `json:"metadata"`
		} `json:"transfers"`
	}

	uniqueSummaries := make(map[string]TournamentSummary)
	chunkMap := make(map[string][]TournamentMatch)
	if json.NewDecoder(resp.Body).Decode(&res) == nil {
		for _, tx := range res.Transfers {
			if strings.HasPrefix(tx.Metadata, "VBT_TOURN_SUMM:") {
				var s TournamentSummary
				// Defensive check: ensure the summary has a valid ID after unmarshaling
				if err := json.Unmarshal([]byte(strings.TrimPrefix(tx.Metadata, "VBT_TOURN_SUMM:")), &s); err == nil && s.ID != "" {
					uniqueSummaries[s.ID] = s
				}
			} else if strings.HasPrefix(tx.Metadata, "VBT_TOURN_DATA:") {
				var chunk struct {
					ID      string
					Matches []TournamentMatch `json:"m"`
				}
				json.Unmarshal([]byte(strings.TrimPrefix(tx.Metadata, "VBT_TOURN_DATA:")), &chunk)
				chunkMap[tx.TransactionID] = chunk.Matches
			}
		}
	}

	// Check for deep_verify parameter from the request
	deepVerify := r.URL.Query().Get("deep_verify") == "true"

	var history []TournamentSummary
	for _, s := range uniqueSummaries {
		// Only perform deep reconstruction and checksum validation if deep_verify is requested.
		// Otherwise, s.Matches will remain nil (if chunked) and IsVerified will be false.
		if deepVerify {
			for _, link := range s.Links {
				if m, ok := chunkMap[link]; ok {
					s.Matches = append(s.Matches, m...)
				}
			}
			if s.Checksum != "" {
				b, _ := json.Marshal(s.Matches)
				h := sha256.Sum256(b)
				if hex.EncodeToString(h[:]) == s.Checksum {
					s.IsVerified = true
				}
			}
		}
		// If not deepVerify, s.Matches will only contain what was directly in the summary (potentially nil if chunked)
		// and s.IsVerified will remain false (its default value).
		history = append(history, s)
	}
	sort.Slice(history, func(i, j int) bool { return history[i].Timestamp.After(history[j].Timestamp) })
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"history": history})
}

func (l *Lobby) processTournamentResult(matchID, winnerWallet string) {
	found := false
	for i, m := range l.tournament.Matches {
		if m.ID == matchID && m.Winner == "" {
			l.tournament.Matches[i].Winner = winnerWallet
			found = true
			break
		}
	}
	if !found {
		return
	}

	roundComplete := true
	for _, m := range l.tournament.Matches {
		if m.Round == l.tournament.CurrentRound && m.Winner == "" {
			roundComplete = false
			break
		}
	}
	if roundComplete {
		l.advanceTournamentRound()
	} else {
		l.broadcastTournamentState()
	}
}

func (l *Lobby) advanceTournamentRound() {
	var roundWinners []string
	for _, m := range l.tournament.Matches {
		if m.Round == l.tournament.CurrentRound && m.Winner != "" {
			roundWinners = append(roundWinners, m.Winner)
		}
	}

	if len(roundWinners) <= 1 {
		l.finalizeTournament(roundWinners)
		return
	}

	l.tournament.CurrentRound++
	newRound := l.tournament.CurrentRound
	for i := 0; i < len(roundWinners); i += 2 {
		matchNum := (i / 2) + 1
		if i+1 >= len(roundWinners) {
			// PILLAR 3: Bracket Integrity.
			// Handle odd numbers of winners by granting a BYE.
			// The winner string is enforced to be the advancing player.
			l.tournament.Matches = append(l.tournament.Matches, TournamentMatch{
				ID: fmt.Sprintf("R%d-M%d-BYE", newRound, matchNum), P1: roundWinners[i], P2: "BYE", Round: newRound, Winner: roundWinners[i],
			})
			continue
		}
		l.tournament.Matches = append(l.tournament.Matches, TournamentMatch{
			ID: fmt.Sprintf("R%d-M%d", newRound, matchNum), P1: roundWinners[i], P2: roundWinners[i+1], Round: newRound,
		})
	}
	l.broadcastTournamentState()
}

// determineTop5 identifies the tournament rankings based on bracket progression.
func (l *Lobby) determineTop5(matches []TournamentMatch, winner string) []string {
	top5 := []string{}
	if winner == "" {
		return top5
	}
	top5 = append(top5, winner)

	maxRound := 0
	for _, m := range matches {
		if m.Round > maxRound {
			maxRound = m.Round
		}
	}

	// 2nd Place: Loser of the final
	for _, m := range matches {
		if m.Round == maxRound && m.Winner != "" {
			runnerUp := m.P1
			if strings.EqualFold(m.P1, winner) {
				runnerUp = m.P2
			}
			if runnerUp != "" && !strings.EqualFold(runnerUp, "BYE") {
				top5 = append(top5, runnerUp)
			}
			break
		}
	}

	// 3rd & 4th: Losers of semi-finals (Sorted by Reputation)
	semiLosers := []string{}
	for _, m := range matches {
		if m.Round == maxRound-1 && maxRound > 1 && m.Winner != "" {
			var loser string
			if strings.EqualFold(m.Winner, m.P1) {
				loser = m.P2
			} else {
				loser = m.P1
			}
			if loser != "" && !strings.EqualFold(loser, "BYE") {
				semiLosers = append(semiLosers, loser)
			}
		}
	}
	// PILLAR 1: Performance Tie-breaker
	sort.Slice(semiLosers, func(i, j int) bool {
		return l.leaderboard[semiLosers[i]].Reputation > l.leaderboard[semiLosers[j]].Reputation
	})
	top5 = append(top5, semiLosers...)

	// 5th: Losers of quarter-finals (Sorted by Reputation)
	if len(top5) < 5 {
		quartLosers := []string{}
		for _, m := range matches {
			if m.Round == maxRound-2 && maxRound > 2 && m.Winner != "" {
				var loser string
				if strings.EqualFold(m.Winner, m.P1) {
					loser = m.P2
				} else {
					loser = m.P1
				}
				if loser != "" && !strings.EqualFold(loser, "BYE") {
					quartLosers = append(quartLosers, loser)
				}
			}
		}
		sort.Slice(quartLosers, func(i, j int) bool {
			return l.leaderboard[quartLosers[i]].Reputation > l.leaderboard[quartLosers[j]].Reputation
		})
		for _, lsr := range quartLosers {
			top5 = append(top5, lsr)
			if len(top5) >= 5 {
				break
			}
		}
	}

	return top5
}

func (l *Lobby) finalizeTournament(winners []string) {
	winner := ""
	if len(winners) > 0 {
		winner = winners[0]
	}

	// PILLAR 1: Governor's Tax Integration.
	// 5% of the total tournament pot is routed to the club controlling the 'arena_center' territory.
	var govTax float64
	centerClub := l.getClubByTerritoryID("arena_center")
	if centerClub != nil {
		govTax = l.tournament.Pot * 0.05
		centerClub.Treasury += govTax
		centerClub.LastActivity = time.Now()
		l.logAdminAuditLocked("GOVERNOR_TAX_PAID", centerClub.ID, fmt.Sprintf("Tournament Pot Tax: %.2f $VBV", govTax))
	}

	// Calculate effective pot available for player distribution
	effectivePot := l.tournament.Pot - govTax

	// Placement Identification & Multi-Asset Reward Loop
	top5 := l.determineTop5(l.tournament.Matches, winner)
	payoutPercentages := []float64{0.40, 0.25, 0.15, 0.10, 0.10}

	if effectivePot > 0 && len(top5) > 0 {
		// PILLAR 3: Economic Precision.
		// The loop iterates only over the actual number of players in top5.
		// If top5 is shorter than 5, only the corresponding payout percentages are distributed.
		// The remaining portion of the effectivePot is retained in the faucet.
		log.Printf("[TOURNAMENT] Finalizing Event. Pot: %.2f $VBV (Tax: %.2f). Payout Ranks: %v\n", effectivePot, govTax, top5)

		for i, player := range top5 {
			if i >= len(payoutPercentages) {
				break
			}

			// Calculate Pot Share (Primary Asset)
			shareMicro := uint64(effectivePot * payoutPercentages[i] * 1000000)

			// Dispatch grouped rewards in background goroutine
			go func(p string, rank int, amt uint64) {
				txid, skipped, err := l.dispatchTournamentRewards(p, rank+1, amt)
				if err != nil {
					log.Printf("[TOURNAMENT ERROR] Payout failed for rank %d (%s): %v\n", rank+1, p, err)
				} else {
					log.Printf("[TOURNAMENT] Payout successful for rank %d (%s). Tx: %s. Skipped: %v\n", rank+1, p, txid, skipped)
					l.broadcastToAdmins(fmt.Sprintf("🏆 <b>TOURNAMENT PAYOUT:</b> Rank %d (%s) received rewards. Tx: %s. (Skipped: %v)", rank+1, p, txid, strings.Join(skipped, ", ")))
				}
			}(player, i, shareMicro)
		}
	}

	summary := TournamentSummary{
		ID: fmt.Sprintf("ARENA-T-%d", time.Now().Unix()), Timestamp: time.Now(),
		Pot: l.tournament.Pot, Winner: winner, Matches: l.tournament.Matches,
	}

	go l.recordTournamentOnChain(summary)

	l.tournament.Active = false
	l.broadcastTournamentState()
}

func (l *Lobby) recordTournamentOnChain(summary TournamentSummary) {
	var childLinks []string
	matchBytes, _ := json.Marshal(summary.Matches)
	hash := sha256.Sum256(matchBytes)
	summary.Checksum = hex.EncodeToString(hash[:])

	if len(matchBytes) > 800 {
		for i := 0; i < len(summary.Matches); i += 4 {
			end := i + 4
			if end > len(summary.Matches) {
				end = len(summary.Matches)
			}
			chunk := struct {
				ID      string            `json:"id"`
				Matches []TournamentMatch `json:"m"`
			}{ID: summary.ID, Matches: summary.Matches[i:end]}
			chunkJSON, _ := json.Marshal(chunk)
			txid, err := l.sendNoteTx(fmt.Sprintf("VBT_TOURN_DATA:%s", string(chunkJSON)))
			if err == nil {
				childLinks = append(childLinks, txid)
			}
		}
		summary.Matches = nil
	}

	summary.Links = childLinks
	jsonData, _ := json.Marshal(summary)
	l.sendNoteTx(fmt.Sprintf("VBT_TOURN_SUMM:%s", string(jsonData)))
}

// dispatchTournamentRewards handles multi-asset distribution for tournament finishers.
func (l *Lobby) dispatchTournamentRewards(recipient string, rank int, potShareMicro uint64) (string, []string, error) {
	l.mutex.RLock()
	voiConfig, _ := l.availableNetworks["Voi Mainnet"]
	activeRewards := l.rewards
	rewardAsset := l.rewardAssetID
	stats, hasStats := l.leaderboard[recipient]
	l.mutex.RUnlock()

	// PILLAR 4: Reputation Bonus Integration.
	// Apply 1.1x multiplier for high-standing players (Diamond Tier / Rep 500+).
	multiplier := 1.0
	if hasStats && stats.Reputation >= 500 {
		multiplier = 1.1
	}

	client, _ := algod.MakeClient(voiConfig.NodeURL, "")
	pk, _ := mnemonic.ToPrivateKey(os.Getenv("FAUCET_MNEMONIC"))
	faucetAccount, _ := crypto.AccountFromPrivateKey(pk)
	sp, _ := client.SuggestedParams().Do(context.Background())

	var txns []types.Transaction
	var skippedAssets []string
	vaultAddrObj, _ := types.DecodeAddress(l.vaultAddress)
	note := []byte(fmt.Sprintf("VBT_TOURN_PAYOUT:{\"rank\":%d,\"pot_share\":%d}", rank, potShareMicro))
	var totalUnits float64

	// Build Atomic Group for all active reward assets
	for appIDStr, baseAmt := range activeRewards {
		appID, err := strconv.ParseUint(appIDStr, 10, 64)
		if err != nil {
			continue
		}

		amt := uint64(float64(baseAmt) * multiplier)
		// Add the tournament pot share if this is the primary asset
		if appIDStr == rewardAsset {
			amt += uint64(float64(potShareMicro) * multiplier)
		}

		// NEW: Granular Opt-in Verification to prevent group failure
		optedIn, _, err := l.checkAssetOptIn("VOI", recipient, appIDStr)
		if err != nil || !optedIn {
			log.Printf("[TOURNAMENT] Skipping asset %s for %s: Opt-in missing or error: %v", appIDStr, recipient, err)
			skippedAssets = append(skippedAssets, appIDStr)
			continue
		}

		// Check vault's balance for this specific asset
		boxResponse, err := client.GetApplicationBoxByName(appID, vaultAddrObj[:]).Do(context.Background())
		if err != nil {
			log.Printf("[TOURNAMENT] Reward app %s box fetch failed: %v", appIDStr, err)
			skippedAssets = append(skippedAssets, appIDStr)
			continue
		}

		if len(boxResponse.Value) >= 32 {
			bal := new(big.Int).SetBytes(boxResponse.Value[:32]).Uint64()
			if bal < amt {
				log.Printf("[TOURNAMENT] Insufficient vault balance for asset %s. Needed: %d", appIDStr, amt)
				skippedAssets = append(skippedAssets, appIDStr)
				continue
			}
		}

		totalUnits += float64(amt) / 1000000.0
		// Build NoOp call for ARC-200 with winner as account argument and placement note
		txn, _ := transaction.MakeApplicationNoOpTx(appID, nil, []string{recipient}, nil, nil, sp, vaultAddrObj, note, types.Digest{}, [32]byte{}, types.Address{})
		txns = append(txns, txn)
	}

	if len(txns) == 0 {
		if len(skippedAssets) > 0 {
			return "", skippedAssets, fmt.Errorf("all attempted assets skipped due to opt-in or balance failures")
		}
		return "", nil, fmt.Errorf("no reward assets configured for payout")
	}

	gid, _ := crypto.ComputeGroupID(txns)
	var signedGroup []byte
	var firstTxID string
	for i := range txns {
		txns[i].Group = gid
		txid, stxn, _ := crypto.SignTransaction(faucetAccount.PrivateKey, txns[i])
		signedGroup = append(signedGroup, stxn...)
		if i == 0 {
			firstTxID = txid
		}
	}

	if _, err := client.SendRawTransaction(signedGroup).Do(context.Background()); err != nil {
		return "", skippedAssets, err
	}

	// INDUSTRIAL LOOP: Deduct payout from liquid faucet balance and trigger scaling.
	l.mutex.Lock()
	l.faucetBalance -= totalUnits
	l.applyDynamicScalingLocked()
	l.mutex.Unlock()

	return firstTxID, skippedAssets, nil
}

func (l *Lobby) broadcastTournamentState() {
	payload, _ := json.Marshal(l.tournament)
	go func() {
		l.broadcast <- jsonListEnvelope("tournament_update", payload)
	}()
}

func (l *Lobby) isWalletRegistered(wallet string) bool {
	for _, p := range l.paidParticipants {
		if strings.EqualFold(p, wallet) {
			return true
		}
	}
	return false
}
