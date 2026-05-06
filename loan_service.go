package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/encoding/msgpack"
	"github.com/algorand/go-algorand-sdk/v2/types"
)

// handleGetLoans returns all active loans or loans specific to a player.
func (l *Lobby) handleGetLoans(w http.ResponseWriter, r *http.Request) {
	l.mutex.RLock()
	var loans []*Loan
	borrowerWallet := r.URL.Query().Get("wallet")

	for _, loan := range l.loans {
		if borrowerWallet == "" || strings.EqualFold(loan.BorrowerWallet, borrowerWallet) {
			loans = append(loans, loan)
		}
	}
	l.mutex.RUnlock()

	// Lazy Resolution outside the global lock to prevent display latency and deadlocks.
	for _, loan := range loans {
		if loan.BorrowerName == "" {
			loan.BorrowerName = l.ResolveEnvoiName(loan.BorrowerWallet)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loans)
}

// handleTakeLoan allows a player to take a $VBV loan using a CardBundle as collateral.
func (l *Lobby) handleTakeLoan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Wallet           string     `json:"wallet"`
		CollateralBundle CardBundle `json:"collateral_bundle"`
		LoanAmount       float64    `json:"loan_amount"` // In whole $VBV units
		DurationHours    int        `json:"duration_hours"`
		ClientID         string     `json:"client_id"`
		SignedTx         []byte     `json:"signed_tx"` // For nonce verification
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.LoanAmount <= 0 || req.DurationHours <= 0 {
		http.Error(w, "Invalid loan amount or duration", http.StatusBadRequest)
		return
	}

	borrowerName := l.ResolveEnvoiName(req.Wallet)

	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.faucetBalance < req.LoanAmount { // Check if faucet has enough to fund the loan
		http.Error(w, "Faucet has insufficient funds for this loan", http.StatusServiceUnavailable)
		return
	}

	// Nonce verification
	nonceData, nonceExists := l.nonces[req.ClientID]
	if !nonceExists || time.Since(nonceData.CreatedAt) > 5*time.Minute {
		http.Error(w, "Unauthorized: Nonce expired", http.StatusUnauthorized)
		return
	}
	var stx types.SignedTxn
	if err := msgpack.Decode(req.SignedTx, &stx); err != nil || stx.Txn.Sender.String() != req.Wallet || string(stx.Txn.Note) != nonceData.Value {
		http.Error(w, "Unauthorized: Signature mismatch", http.StatusUnauthorized)
		return
	}

	// Check collateral in player's inventory
	stats := l.leaderboard[req.Wallet]
	if req.CollateralBundle.CardID != 0 && stats.Inventory[fmt.Sprintf("CARD-%d", req.CollateralBundle.CardID)] <= 0 {
		http.Error(w, "Collateral card not in inventory", http.StatusBadRequest)
		return
	}
	if req.CollateralBundle.WeaponID != "" && stats.Inventory[req.CollateralBundle.WeaponID] <= 0 {
		http.Error(w, "Collateral weapon not in inventory", http.StatusBadRequest)
		return
	}
	if req.CollateralBundle.FaceplateID != "" && stats.Inventory[req.CollateralBundle.FaceplateID] <= 0 {
		http.Error(w, "Collateral faceplate not in inventory", http.StatusBadRequest)
		return
	}

	// Escrow collateral: Deduct from player's inventory
	if req.CollateralBundle.CardID != 0 {
		stats.Inventory[fmt.Sprintf("CARD-%d", req.CollateralBundle.CardID)]--
	}
	if req.CollateralBundle.WeaponID != "" {
		stats.Inventory[req.CollateralBundle.WeaponID]--
	}
	if req.CollateralBundle.FaceplateID != "" {
		stats.Inventory[req.CollateralBundle.FaceplateID]--
	}
	l.leaderboard[req.Wallet] = stats

	// Calculate repayment amount (e.g., 10% interest)
	loanAmountMicro := uint64(req.LoanAmount * 1000000)
	repaymentAmountMicro := uint64(float64(loanAmountMicro) * 1.10) // 10% interest

	loanID := fmt.Sprintf("LOAN-%d", time.Now().UnixNano())
	l.loans[loanID] = &Loan{
		ID:               loanID,
		BorrowerName:     borrowerName,
		BorrowerWallet:   req.Wallet,
		CollateralBundle: req.CollateralBundle,
		LoanAmount:       loanAmountMicro,
		RepaymentAmount:  repaymentAmountMicro,
		DueAt:            time.Now().Add(time.Duration(req.DurationHours) * time.Hour),
		Status:           "active",
		TerritoryID:      "the_second_hand_store", // Fixed territory for Second-Hand Store
	}

	// Dispense loan amount to player's rewards
	// CRITICAL FIX: Deduct principal from Faucet Pool to maintain economic balance
	l.faucetBalance -= req.LoanAmount
	l.rewards[req.Wallet] += loanAmountMicro
	l.applyDynamicScalingLocked()

	l.logAdminAudit("LOAN_TAKEN", req.Wallet, fmt.Sprintf("Loan ID: %s, Amount: %.2f, Repay: %.2f", loanID, req.LoanAmount, float64(repaymentAmountMicro)/1000000.0))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(l.loans[loanID])

	// Trigger global sync update
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// handleRepayLoan allows a player to repay a loan and retrieve their collateral.
func (l *Lobby) handleRepayLoan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LoanID  string `json:"loan_id"`
		Wallet  string `json:"wallet"`
		TxID    string `json:"txid"`
		Network string `json:"network"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Wallet == "" || req.TxID == "" {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	loan, exists := l.loans[req.LoanID]
	if !exists || loan.Status != "active" || !strings.EqualFold(loan.BorrowerWallet, req.Wallet) {
		http.Error(w, "Loan not found or not active for this wallet", http.StatusBadRequest)
		return
	}

	// Verify repayment...
	voiConfig, _ := l.availableNetworks["Voi Mainnet"]
	assetID := voiConfig.AssetID
	verifyNet := "Voi"
	if strings.EqualFold(req.Network, "ALGO") {
		assetID = l.avoiAssetID
		verifyNet = "Algorand"
	}

	verified, _, err := l.verifyBuyInTransaction(verifyNet, req.TxID, loan.RepaymentAmount, assetID, req.Wallet, l.vaultAddress)
	if err != nil || !verified {
		http.Error(w, "Repayment verification failed", http.StatusPaymentRequired)
		return
	}

	// Add the full repayment amount (principal + interest) to the faucet balance
	l.faucetBalance += float64(loan.RepaymentAmount) / 1000000.0
	l.applyDynamicScalingLocked() // Recalculate rewards based on new faucet balance
	// Fulfillment
	stats := l.leaderboard[req.Wallet]
	if stats.Inventory == nil {
		stats.Inventory = make(map[string]int)
	}
	if loan.CollateralBundle.CardID != 0 {
		stats.Inventory[fmt.Sprintf("CARD-%d", loan.CollateralBundle.CardID)]++
	}
	if loan.CollateralBundle.WeaponID != "" {
		stats.Inventory[loan.CollateralBundle.WeaponID]++
	}
	if loan.CollateralBundle.FaceplateID != "" {
		stats.Inventory[loan.CollateralBundle.FaceplateID]++
	}
	l.leaderboard[req.Wallet] = stats

	delete(l.loans, req.LoanID)
	// No need to update BorrowerName here, as the loan is being deleted.

	l.logAdminAudit("LOAN_REPAID", req.Wallet, fmt.Sprintf("Loan ID: %s, Amount: %.2f", loan.ID, float64(loan.RepaymentAmount)/1000000.0))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "message": "Collateral returned."})

	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}
