package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// handleGetBlackMarket returns liquidated items available for purchase.
// Gated by Cunning and Wanted Level.
func (l *Lobby) handleGetBlackMarket(w http.ResponseWriter, r *http.Request) {
	wallet := strings.ToLower(r.URL.Query().Get("wallet"))

	l.mutex.RLock()
	stats, exists := l.leaderboard[wallet]
	l.mutex.RUnlock()

	if !exists || stats.Cunning < 10 || stats.WantedLevel < 5 {
		http.Error(w, "The Black Market is hidden from you. Increase your Cunning and Infamy.", http.StatusForbidden)
		return
	}

	l.mutex.RLock()
	defer l.mutex.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(l.blackMarket)
}

// handleSellMarketTokens allows players to liquidate equity back into spendable $VBV.
func (l *Lobby) handleSellMarketTokens(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Wallet string `json:"wallet"`
		Amount uint64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	wallet := strings.ToLower(req.Wallet)
	l.mutex.Lock()
	defer l.mutex.Unlock()

	stats, exists := l.leaderboard[wallet]
	if !exists || stats.MarketTokens < req.Amount {
		http.Error(w, "Insufficient Market Tokens", http.StatusBadRequest)
		return
	}

	// Exchange rate: 1 Market Token = 0.8 VBV (Scavenger tax)
	vbvGainMicro := uint64(float64(req.Amount) * 0.8)
	vbvGainBase := float64(vbvGainMicro) / 1000000.0

	// Industrial Loop: Check Faucet Liquidity for payout
	if l.faucetBalance < vbvGainBase {
		l.sendToClientLocked(l.getClientIDFromWalletLocked(wallet), Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Market Error: The Arena Faucet has insufficient liquidity for this liquidation."}`)})
		return
	}

	stats.MarketTokens -= req.Amount
	l.rewards[wallet] += vbvGainMicro
	l.leaderboard[wallet] = stats

	// Industrial Loop: Deduct payout from Faucet and trigger scaling
	l.faucetBalance -= vbvGainBase
	l.applyDynamicScalingLocked()

	l.logAdminAudit("TOKEN_LIQUIDATION", wallet, fmt.Sprintf("Sold %d tokens for %.2f $VBV", req.Amount, float64(vbvGainMicro)/1000000.0))
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "vbv_gained": vbvGainMicro})
}

// handleBuyBlackMarket allows high-infamy players to buy liquidated bundles at a discount.
func (l *Lobby) handleBuyBlackMarket(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Wallet  string `json:"wallet"`
		LoanID  string `json:"loan_id"`
		Network string `json:"network"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	wallet := strings.ToLower(req.Wallet)
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// 1. Find the liquidated loan in blackMarket
	idx := -1
	for i, bm := range l.blackMarket {
		if bm.ID == req.LoanID {
			idx = i
			break
		}
	}
	if idx == -1 {
		http.Error(w, "Item no longer available", http.StatusNotFound)
		return
	}

	loan := l.blackMarket[idx]

	stats, exists := l.leaderboard[wallet]
	if !exists {
		http.Error(w, "Player records not found", http.StatusInternalServerError)
		return
	}

	// Scavenger Price: 75% of the original repayment amount (Rounded to nearest micro-unit)
	scavengePrice := (loan.RepaymentAmount*75 + 50) / 100

	if l.rewards[wallet] < scavengePrice {
		http.Error(w, "Insufficient reward balance to scavenge this bundle", http.StatusPaymentRequired)
		return
	}

	// Execute scavenge
	l.rewards[wallet] -= scavengePrice
	
	// Recovery: Add scavenge proceeds back to faucet and trigger scaling
	l.faucetBalance += float64(scavengePrice) / 1000000.0
	l.applyDynamicScalingLocked() // Prevent deadlock since lock is already held

	if stats.Inventory == nil {
		stats.Inventory = make(map[string]int)
	}

	// Add items with a Wanted penalty (Scavenging stolen goods)
	if loan.CollateralBundle.CardID != 0 {
		stats.Inventory[fmt.Sprintf("CARD-%d", loan.CollateralBundle.CardID)]++
	}
	if loan.CollateralBundle.WeaponID != "" {
		stats.Inventory[loan.CollateralBundle.WeaponID]++
	}
	if loan.CollateralBundle.FaceplateID != "" {
		stats.Inventory[loan.CollateralBundle.FaceplateID]++
	}

	stats.WantedLevel += 5 // Scavenging stolen goods increases infamy
	stats.Reputation = l.CalculateReputation(stats) // Recalculate after Wanted Level increase
	l.leaderboard[wallet] = stats

	l.blackMarket = append(l.blackMarket[:idx], l.blackMarket[idx+1:]...)
	l.logAdminAudit("BLACK_MARKET_BUY", wallet, fmt.Sprintf("Scavenged %s for %.2f $VBV", loan.ID, float64(scavengePrice)/1000000.0))
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Stolen goods acquired. Watch your back."})
}
