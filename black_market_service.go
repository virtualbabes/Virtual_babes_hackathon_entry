package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// handleGetBlackMarket returns liquidated items available for purchase.
// Gated by Cunning and Wanted Level.
func (l *Lobby) handleGetBlackMarket(w http.ResponseWriter, r *http.Request) {
	wallet := r.URL.Query().Get("wallet")

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

	l.mutex.Lock()
	defer l.mutex.Unlock()

	stats, exists := l.leaderboard[req.Wallet]
	if !exists || stats.MarketTokens < req.Amount {
		http.Error(w, "Insufficient Market Tokens", http.StatusBadRequest)
		return
	}

	// Exchange rate: 1 Market Token = 0.8 VBV (Scavenger tax)
	vbvGainMicro := uint64(float64(req.Amount) * 0.8)
	stats.MarketTokens -= req.Amount
	l.rewards[req.Wallet] += vbvGainMicro
	l.leaderboard[req.Wallet] = stats

	l.logAdminAudit("TOKEN_LIQUIDATION", req.Wallet, fmt.Sprintf("Sold %d tokens for %.2f $VBV", req.Amount, float64(vbvGainMicro)/1000000.0))
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
	// Scavenger Price: 75% of the original repayment amount
	scavengePrice := uint64(float64(loan.RepaymentAmount) * 0.75)

	if l.rewards[req.Wallet] < scavengePrice {
		http.Error(w, "Insufficient reward balance to scavenge this bundle", http.StatusPaymentRequired)
		return
	}

	// Execute scavenge
	l.rewards[req.Wallet] -= scavengePrice
	stats := l.leaderboard[req.Wallet]
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
	l.leaderboard[req.Wallet] = stats

	l.blackMarket = append(l.blackMarket[:idx], l.blackMarket[idx+1:]...)
	l.logAdminAudit("BLACK_MARKET_BUY", req.Wallet, fmt.Sprintf("Scavenged %s for %.2f $VBV", loan.ID, float64(scavengePrice)/1000000.0))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Stolen goods acquired. Watch your back."})
}
