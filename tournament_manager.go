package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const regCacheFileName = "registrations.json"

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

	l.mutex.RLock()
	if !l.tournament.Active || l.tournament.CurrentRound != 0 {
		l.mutex.RUnlock()
		http.Error(w, "Registration is currently closed", http.StatusForbidden)
		return
	}
	if l.isWalletRegistered(req.Wallet) {
		l.mutex.RUnlock()
		http.Error(w, "Wallet already registered", http.StatusForbidden)
		return
	}
	openTime := l.tournament.OpenTime
	buyInAmt := l.tournament.BuyInAmount
	l.mutex.RUnlock()

	// Identify Elite Status
	l.mutex.RLock()
	type entry struct { wallet string; wins int }
	var hof []entry
	for wallet, stats := range l.leaderboard { hof = append(hof, entry{wallet: wallet, wins: stats.Wins}) }
	l.mutex.RUnlock()
	sort.Slice(hof, func(i, j int) bool { return hof[i].wins > hof[j].wins })
	isElite := false
	for i := 0; i < len(hof) && i < 10; i++ {
		if hof[i].wallet == req.Wallet { isElite = true; break }
	}

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
		verifyNetwork := "Voi"

		if req.Network == "ALGO" {
			buyInAsset = l.avoiAssetID
			verifyNetwork = "Algorand"
		}

		// verifyBuyInTransaction expects a prefix that matches "Network Mainnet" keys
		verified, txTime, err := l.verifyBuyInTransaction(verifyNetwork, req.TxID, uint64(buyInAmt*1000000), buyInAsset, req.Wallet, l.vaultAddress)
		if err != nil || !verified || txTime < openTime.Unix() {
			log.Printf("[TOURNAMENT] Verification failed for %s on %s. Error: %v\n", req.Wallet, verifyNetwork, err)
			http.Error(w, "Payment verification failed or transaction too old", http.StatusPaymentRequired)
			return
		}
	}

	l.mutex.Lock()
	l.paidParticipants = append(l.paidParticipants, req.Wallet)
	if !isElite {
		l.registeredTxIDs[req.TxID] = time.Now()
		l.faucetBalance += (buyInAmt / 2.0)
		l.tournamentPotBonus += (buyInAmt / 2.0)
	}
	l.mutex.Unlock()

	// Process Club Kickback (Tournament Revenue Loop)
	if !isElite {
		l.distributeTournamentKickback(req.Wallet, uint64(buyInAmt*1000000), time.Now())
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "is_elite": isElite})
}

func (l *Lobby) handleTournamentHistory(w http.ResponseWriter, r *http.Request) {
	l.mutex.RLock()
	voiConfig, _ := l.availableNetworks["Voi Mainnet"]
	l.mutex.RUnlock()

	url := fmt.Sprintf("%s/arc200/transfers?contractId=%s&from=%s&to=%s&limit=500", 
		voiConfig.IndexerURL, voiConfig.AssetID, l.vaultAddress, l.vaultAddress)

	ctx, cancel := context.WithTimeout(context.Background(), indexerTimeout)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil { return }
	defer resp.Body.Close()

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
				json.Unmarshal([]byte(strings.TrimPrefix(tx.Metadata, "VBT_TOURN_SUMM:")), &s)
				uniqueSummaries[s.ID] = s
			} else if strings.HasPrefix(tx.Metadata, "VBT_TOURN_DATA:") {
				var chunk struct { ID string; Matches []TournamentMatch `json:"m"` }
				json.Unmarshal([]byte(strings.TrimPrefix(tx.Metadata, "VBT_TOURN_DATA:")), &chunk)
				chunkMap[tx.TransactionID] = chunk.Matches
			}
		}
	}

	var history []TournamentSummary
	for _, s := range uniqueSummaries {
		for _, link := range s.Links {
			if m, ok := chunkMap[link]; ok { s.Matches = append(s.Matches, m...) }
		}
		if s.Checksum != "" {
			b, _ := json.Marshal(s.Matches)
			h := sha256.Sum256(b)
			if hex.EncodeToString(h[:]) == s.Checksum { s.IsVerified = true }
		}
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
	if !found { return }

	roundComplete := true
	for _, m := range l.tournament.Matches {
		if m.Round == l.tournament.CurrentRound && m.Winner == "" {
			roundComplete = false; break
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
			if l.isWalletConnected(m.Winner) { roundWinners = append(roundWinners, m.Winner) }
		}
	}

	if len(roundWinners) <= 1 {
		l.finalizeTournament(roundWinners)
		return
	}

	l.tournament.CurrentRound++
	newRound := l.tournament.CurrentRound
	for i := 0; i < len(roundWinners); i += 2 {
		if i+1 >= len(roundWinners) {
			l.tournament.Matches = append(l.tournament.Matches, TournamentMatch{
				ID: fmt.Sprintf("R%d-M%d-BYE", newRound, i/2), P1: roundWinners[i], P2: "BYE", Round: newRound, Winner: roundWinners[i],
			})
			continue
		}
		l.tournament.Matches = append(l.tournament.Matches, TournamentMatch{
			ID: fmt.Sprintf("R%d-M%d", newRound, i/2), P1: roundWinners[i], P2: roundWinners[i+1], Round: newRound,
		})
	}
	l.broadcastTournamentState()
}

func (l *Lobby) finalizeTournament(winners []string) {
	winner := ""
	if len(winners) > 0 { winner = winners[0] }
	
	if l.tournament.Pot > 0 && winner != "" {
		payouts := []float64{0.40, 0.25, 0.15, 0.10, 0.10}
		// Heuristic to fill Top 5 based on bracket progress
		top5 := []string{winner}
		l.mockARC200Dispenser(winner, l.tournament.Pot*payouts[0], 1)
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
			end := i + 4; if end > len(summary.Matches) { end = len(summary.Matches) }
			chunk := struct { ID string `json:"id"`; Matches []TournamentMatch `json:"m"` }{ID: summary.ID, Matches: summary.Matches[i:end]}
			chunkJSON, _ := json.Marshal(chunk)
			txid, err := l.sendNoteTx(fmt.Sprintf("VBT_TOURN_DATA:%s", string(chunkJSON)))
			if err == nil { childLinks = append(childLinks, txid) }
		}
		summary.Matches = nil
	}

	summary.Links = childLinks
	jsonData, _ := json.Marshal(summary)
	l.sendNoteTx(fmt.Sprintf("VBT_TOURN_SUMM:%s", string(jsonData)))
}

func (l *Lobby) broadcastTournamentState() {
	payload, _ := json.Marshal(l.tournament)
	l.broadcast <- jsonListEnvelope("tournament_update", payload)
}

func (l *Lobby) isWalletRegistered(wallet string) bool {
	for _, p := range l.paidParticipants {
		if p == wallet { return true }
	}
	return false
}
