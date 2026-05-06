package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// handleCourthouseReset allows players to pay a $VBV fine to reset their Wanted Level.
// The fine is calculated as 100 $VBV per Wanted Level point.
func (l *Lobby) handleCourthouseReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Wallet  string `json:"wallet"`
		TxID    string `json:"txid"`
		Network string `json:"network"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Wallet == "" || req.TxID == "" {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	l.mutex.RLock()
	stats, exists := l.leaderboard[req.Wallet]
	voiConfig, voiOk := l.availableNetworks["Voi Mainnet"]
	avoiAssetID := l.avoiAssetID
	vaultAddr := l.vaultAddress
	l.mutex.RUnlock()

	if !voiOk {
		http.Error(w, "Voi network configuration missing", http.StatusInternalServerError)
		return
	}

	if !exists || stats.WantedLevel <= 0 {
		http.Error(w, "No active wanted level to reset", http.StatusBadRequest)
		return
	}

	// Cost calculation: 100 $VBV per Wanted Level point
	costBase := float64(stats.WantedLevel * 100)
	costMicro := uint64(costBase * 1000000)

	assetID := voiConfig.AssetID
	verifyNet := "Voi"
	if req.Network == "ALGO" {
		assetID = avoiAssetID
		verifyNet = "Algorand"
	}

	// Verify the fine payment transaction via blockchain indexer
	verified, _, err := l.verifyBuyInTransaction(verifyNet, req.TxID, costMicro, assetID, req.Wallet, vaultAddr)
	if err != nil || !verified {
		log.Printf("[COURTHOUSE] Verification failed for %s. Error: %v\n", req.Wallet, err)
		http.Error(w, "Fine payment verification failed or insufficient amount", http.StatusPaymentRequired)
		return
	}

	// Update Player Stats and Vault balance
	l.mutex.Lock()
	stats.WantedLevel = 0
	l.leaderboard[req.Wallet] = stats
	l.faucetBalance += (costBase / 2.0) // Half of the fine returns to the global faucet pool
	l.distributeCourthouseFineToClubsLocked(costBase / 2.0) // The other half is distributed to clubs
	l.mutex.Unlock()

	l.logAdminAudit("COURTHOUSE_RESET", req.Wallet, fmt.Sprintf("Paid %.2f $VBV fine to reset Wanted Level", costBase))
	go l.unlockAchievement(req.Wallet, "REHABILITATED")

	// Update all clients with the new social standing
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":           "success",
		"message":          "Wanted level cleared. The Arena recognizes your clean slate.",
		"new_wanted_level": 0,
		"fine_paid":        costBase,
	})
}
