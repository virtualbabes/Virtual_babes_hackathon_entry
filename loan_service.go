//go:build !js && !wasm

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/encoding/msgpack"
	"github.com/algorand/go-algorand-sdk/v2/mnemonic"
	"github.com/algorand/go-algorand-sdk/v2/transaction"
	"github.com/algorand/go-algorand-sdk/v2/types"
)

// LoanService encapsulates logic for collateralized debt and credit markets.
// PILLAR 5: Stateless Service Design.
type LoanService struct{}

// HandleGetLoans returns all active loans or loans specific to a player.
func (s *LoanService) HandleGetLoans(l *Lobby, w http.ResponseWriter, r *http.Request) {
	l.mutex.RLock()
	var loans []*Loan
	// PILLAR 3: Identity Normalization.
	borrowerWallet := strings.ToLower(r.URL.Query().Get("wallet"))

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

// HandleTakeLoan allows a player to take a $VBV loan using a CardBundle as collateral.
func (s *LoanService) HandleTakeLoan(l *Lobby, w http.ResponseWriter, r *http.Request) {
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

	// PILLAR 3: Identity Normalization.
	targetWallet := strings.ToLower(req.Wallet)
	borrowerName := l.ResolveEnvoiName(targetWallet)

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
	if err := msgpack.Decode(req.SignedTx, &stx); err != nil || !strings.EqualFold(stx.Txn.Sender.String(), targetWallet) || string(stx.Txn.Note) != nonceData.Value {
		http.Error(w, "Unauthorized: Signature mismatch", http.StatusUnauthorized)
		return
	}

	// Check collateral in player's inventory
	l.ensurePlayerStatsMapsInitialized(targetWallet)
	stats := l.leaderboard[targetWallet]
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

	// PILLAR 5: Modular Orchestration.
	s.TransferBundleItems(l, targetWallet, req.CollateralBundle, false)

	// Calculate repayment amount (e.g., 10% interest)
	// PILLAR 2: Integer Supremacy. Apply 10% interest with nearest-micro-unit rounding.
	loanAmountMicro := uint64(req.LoanAmount*1000000 + 0.5)
	repaymentAmountMicro := (loanAmountMicro*110 + 50) / 100

	loanID := fmt.Sprintf("LOAN-%d", time.Now().UnixNano())
	l.loans[loanID] = &Loan{
		ID:               loanID,
		BorrowerName:     borrowerName,
		BorrowerWallet:   targetWallet,
		CollateralBundle: req.CollateralBundle,
		LoanAmount:       loanAmountMicro,
		RepaymentAmount:  repaymentAmountMicro,
		DueAt:            time.Now().Add(time.Duration(req.DurationHours) * time.Hour),
		Status:           "active",
		TerritoryID:      "south_slums", // Fixed territory for Second-Hand Store (Loan Office)
	}

	// PILLAR 2: Industrial Loop Accounting.
	l.playerBalances[targetWallet] += loanAmountMicro

	l.applyDynamicScalingLocked() // PILLAR 2: Recalculate rewards based on new liability

	// PILLAR 2: Industrial Loop Accounting.
	// Principal is NOT subtracted from faucetBalance here. FaucetBalance represents
	// the authoritative on-chain vault total; the deduction occurs in faucet_service.go
	// during reward dispatch to maintain ledger consistency across the Industrial Loop.

	l.logAdminAuditLocked("LOAN_TAKEN", targetWallet, fmt.Sprintf("Loan ID: %s, Amount: %.2f, Repay: %.2f", loanID, req.LoanAmount, float64(repaymentAmountMicro)/1000000.0))
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(l.loans[loanID])

	// Trigger global sync update
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// HandleRepayLoan allows a player to repay a loan and retrieve their collateral.
func (s *LoanService) HandleRepayLoan(l *Lobby, w http.ResponseWriter, r *http.Request) {
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

	targetWallet := strings.ToLower(req.Wallet)
	l.mutex.Lock()
	defer l.mutex.Unlock()

	loan, exists := l.loans[req.LoanID]
	if !exists || loan.Status != "active" || !strings.EqualFold(loan.BorrowerWallet, targetWallet) {
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

	// PILLAR 3: Specific Purpose Verification for loan repayments
	verified, _, err := l.verifyBuyInTransaction(verifyNet, req.TxID, loan.RepaymentAmount, assetID, targetWallet, l.vaultAddress, "REPAY_LOAN:")
	if err != nil || !verified {
		http.Error(w, "Repayment verification failed", http.StatusPaymentRequired)
		return
	}

	// PILLAR 3: Financial Proof.
	// Record loan repayment on-chain for the audit trail.
	interestMicro := loan.RepaymentAmount - loan.LoanAmount
	paybackDetails := map[string]interface{}{
		"id":         loan.ID,
		"wallet":     targetWallet,
		"principal":  float64(loan.LoanAmount) / 1000000.0,
		"interest":   float64(interestMicro) / 1000000.0,
		"collateral": loan.CollateralBundle,
		"ts":         time.Now().Unix(),
	}

	// Add the full repayment amount (principal + interest) to the faucet balance
	l.faucetBalance += float64(loan.RepaymentAmount) / 1000000.0
	l.applyDynamicScalingLocked() // Recalculate rewards based on new faucet balance

	// PILLAR 5: Modular Orchestration.
	s.TransferBundleItems(l, targetWallet, loan.CollateralBundle, true)

	delete(l.loans, req.LoanID)
	// No need to update BorrowerName here, as the loan is being deleted.

	l.logAdminAuditLocked("LOAN_REPAID", targetWallet, fmt.Sprintf("Loan ID: %s, Amount: %.2f", loan.ID, float64(loan.RepaymentAmount)/1000000.0))

	// Dispatch on-chain log for financial verification
	go func(pd interface{}) {
		jsonPayload, _ := json.Marshal(pd)
		l.sendNoteTx(fmt.Sprintf("VBT_LOAN_PAYBACK:%s", string(jsonPayload)))
	}(paybackDetails)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "message": "Collateral returned."})

	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// ProcessLoans checks for defaulted loans and handles collateral liquidation.
// PILLAR 2: Economic Intelligence.
func (s *LoanService) ProcessLoans(l *Lobby) {
	l.mutex.RLock()
	now := time.Now()
	var candidates []*Loan
	for _, loan := range l.loans {
		if loan.Status == "active" && now.After(loan.DueAt) {
			candidates = append(candidates, loan)
		}
	}
	l.mutex.RUnlock()

	if len(candidates) == 0 {
		return
	}

	anyProcessed := false
	for _, loan := range candidates {
		// PILLAR 2: Non-Custodial Automated Repayment.
		// Before defaulting, verify if the borrower has pre-approved the vault to pull 
		// the repayment amount. This provides a "safety net" for forgetful borrowers.
		approvedAmt, _ := l.oracleService.CheckAssetApproval(l, "VOI", loan.BorrowerWallet, l.vaultAddress, l.rewardAssetID)

		if approvedAmt >= loan.RepaymentAmount {
			// Execute automated on-chain pull (Network I/O)
			if err := s.pullApprovedTokens(l, loan.BorrowerWallet, loan.RepaymentAmount); err == nil {
				l.mutex.Lock()
				// RE-VERIFY: Ensure status hasn't changed while we were processing the pull
				if target, exists := l.loans[loan.ID]; exists && target.Status == "active" {
					// Settle as Repaid
					s.TransferBundleItems(l, loan.BorrowerWallet, loan.CollateralBundle, true)
					l.logAdminAuditLocked("LOAN_AUTO_REPAID", loan.BorrowerWallet, fmt.Sprintf("Loan ID: %s, Amount: %.2f", loan.ID, float64(loan.RepaymentAmount)/1000000.0))
					
					// Track high-value auto-repayment on-chain
					interestMicro := loan.RepaymentAmount - loan.LoanAmount
					paybackDetails := map[string]interface{}{
						"id":         loan.ID,
						"wallet":     loan.BorrowerWallet,
						"principal":  float64(loan.LoanAmount) / 1000000.0,
						"interest":   float64(interestMicro) / 1000000.0,
						"collateral": loan.CollateralBundle,
						"type":       "AUTOMATED",
						"ts":         time.Now().Unix(),
					}
					go func(pd interface{}) {
						jsonPayload, _ := json.Marshal(pd)
						l.sendNoteTx(fmt.Sprintf("VBT_LOAN_PAYBACK:%s", string(jsonPayload)))
					}(paybackDetails)

					delete(l.loans, loan.ID)
					anyProcessed = true
				}
				l.mutex.Unlock()
				continue
			} else {
				log.Printf("[LOAN ERROR] Automated pull failed for %s: %v. Proceeding to default.\n", loan.BorrowerWallet, err)
			}
		}

		// Proceed to Standard Liquidation (Locked)
		l.mutex.Lock()
		// Re-verify status hasn't changed during the approval check
		if target, exists := l.loans[loan.ID]; exists && target.Status == "active" && now.After(target.DueAt) {
			id := target.ID
			loan := target

			loan.Status = "defaulted"

			// Residual Value: 15% of the loan amount is returned as Market Tokens
			tokenReward := (loan.LoanAmount*15 + 50) / 100 // Round to nearest micro-unit to prevent fractional dust

			borrowerWallet := loan.BorrowerWallet
			borrowerStats, exists := l.leaderboard[borrowerWallet]
			if exists {
				borrowerStats.MarketTokens += tokenReward
				// PILLAR 3: Behavioral Consequence.
				// Defaulting on a loan increases infamy (Wanted Level), which
				// is then naturally reflected in the Reputation calculation.
				borrowerStats.WantedLevel += 5
				borrowerStats.Reputation = l.CalculateReputation(borrowerStats)
				l.leaderboard[borrowerWallet] = borrowerStats

				l.sendToClientLocked(l.getClientIDFromWalletLocked(borrowerWallet), Envelope{
					Type:    "admin_notification",
					Payload: json.RawMessage(fmt.Sprintf(`{"text":"🚨 <b>LOAN DEFAULTED:</b> Collateral moved to Black Market. You received %.2f Market Tokens as equity."}`, float64(tokenReward)/1000000.0)),
				})
			}

			// INDUSTRIAL LOOP: 5% Liquidation Fee to the Second-Hand Store district owner
			liquidationFeeMicro := (loan.LoanAmount*5 + 50) / 100
			liquidationFeeBase := float64(liquidationFeeMicro) / 1000000.0
			owningClub := l.getClubByTerritoryID(loan.TerritoryID)
			feeRecipient := "FAUCET"

			if owningClub != nil {
				owningClub.Treasury += liquidationFeeBase
				owningClub.LastActivity = now
				feeRecipient = owningClub.ID
				l.logAdminAuditLocked("LOAN_LIQUIDATION_FEE", loan.TerritoryID, fmt.Sprintf("Club %s earned %.2f $VBV liquidation fee", owningClub.Name, liquidationFeeBase))
			}

			// PILLAR 3: Financial Proof.
			// Record loan liquidation (default) on-chain for the audit trail.
			liquidateDetails := map[string]interface{}{
				"id":             loan.ID,
				"wallet":         borrowerWallet,
				"collateral":     loan.CollateralBundle,
				"original_owner": loan.BorrowerWallet, // PILLAR 3: Forensic Consistency.
				"territory_id":   loan.TerritoryID,
				"fee_recipient":  feeRecipient,
				"fee_amount":     liquidationFeeBase,
				"ts":             now.Unix(),
			}

			// Update playstyle on loan default (Internal call to avoid deadlock)
			l.updatePlayerPlaystyleTendenciesLocked(borrowerWallet, false, [2]int{}, []int{}, false, false)
			l.logAdminAuditLocked("LOAN_LIQUIDATED", borrowerWallet, fmt.Sprintf("ID: %s, Tokens: %d", loan.ID, tokenReward))

			// Dispatch on-chain log for forensic verification
			go func(ld interface{}) {
				jsonPayload, _ := json.Marshal(ld)
				l.sendNoteTx(fmt.Sprintf("VBT_LOAN_LIQUIDATE:%s", string(jsonPayload)))
			}(liquidateDetails)

			// Add the defaulted loan to the black market with a size cap to prevent memory bloat.
			// We maintain a FIFO buffer of 50 items to keep the Underworld market fresh.
			l.blackMarket = append(l.blackMarket, *loan)
			if len(l.blackMarket) > 50 {
				l.blackMarket = l.blackMarket[1:] // Prune oldest entry
			}

			delete(l.loans, id)
			anyProcessed = true
		}
		l.mutex.Unlock()
	}

	if anyProcessed {
		// Trigger global scaling recalculation to reflect the shift in liquid reserves
		msg := l.getLobbyUpdateMsg()
		go func() { l.broadcast <- msg }()
	}
}

// pullApprovedTokens executes an on-chain transferFrom to move tokens from the borrower to the vault.
// PILLAR 2: High-Finance.
func (s *LoanService) pullApprovedTokens(l *Lobby, from string, amount uint64) error {
	l.mutex.RLock()
	voiConfig, _ := l.availableNetworks["Voi Mainnet"]
	vaultAddr := l.vaultAddress
	rewardAsset := l.rewardAssetID
	l.mutex.RUnlock()

	if len(voiConfig.NodeURLs) == 0 { return fmt.Errorf("no nodes available") }

	client, _ := algod.MakeClient(voiConfig.NodeURLs[0], "")
	pk, _ := mnemonic.ToPrivateKey(os.Getenv("FAUCET_MNEMONIC"))
	faucetAccount, _ := crypto.AccountFromPrivateKey(pk)
	sp, _ := client.SuggestedParams().Do(context.Background())

	methodSelector := []byte{0x23, 0xb8, 0x72, 0xdd}
	fromAddr, _ := types.DecodeAddress(from)
	toAddr, _ := types.DecodeAddress(vaultAddr)
	amountBytes := make([]byte, 32)
	new(big.Int).SetUint64(amount).FillBytes(amountBytes)

	appArgs := [][]byte{methodSelector, fromAddr[:], toAddr[:], amountBytes}
	appID, _ := strconv.ParseUint(rewardAsset, 10, 64)
	senderAddr, _ := types.DecodeAddress(vaultAddr)
	note := []byte(fmt.Sprintf("VBT_LOAN_AUTO_PULL:{\"from\":\"%s\",\"amt\":%d}", from, amount))

	txn, _ := transaction.MakeApplicationNoOpTx(appID, appArgs, nil, nil, nil, sp, senderAddr, note, types.Digest{}, [32]byte{}, types.Address{})
	_, stxn, err := crypto.SignTransaction(faucetAccount.PrivateKey, txn)
	if err != nil { return err }
	
	if _, err := client.SendRawTransaction(stxn).Do(context.Background()); err != nil { return err }

	l.mutex.Lock()
	l.faucetBalanceMicro += amount
	l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0

	// PILLAR 2: Real-time Reconciliation.
	// Ensure the automated debt recovery is captured in the financial health report.
	if l.tokenSinkRouter != nil && l.tokenSinkRouter.Audit != nil {
		// PILLAR 2: Structural Audit. Report zero siphon for standard on-chain pulls.
		_ = l.tokenSinkRouter.Audit.InterceptAndAudit("LOAN_AUTO_PULL", amount, amount, 0, 0, 0)
	}

	l.applyDynamicScalingLocked()
	l.mutex.Unlock()

	return nil
}

// TransferBundleItems handles adding or removing items from a player's inventory during loan cycles.
// If add is true, items are added. If add is false, items are removed (escrowed).
// It assumes the lobby mutex is held.
func (s *LoanService) TransferBundleItems(l *Lobby, wallet string, bundle CardBundle, add bool) {
	l.ensurePlayerStatsMapsInitialized(wallet)
	stats := l.leaderboard[wallet]

	// PILLAR 5: Defensive Map Handling.
	if stats.Inventory == nil {
		stats.Inventory = make(map[string]int)
	}

	if bundle.CardID != 0 {
		cardKey := fmt.Sprintf("CARD-%d", bundle.CardID)
		if add {
			stats.Inventory[cardKey]++
		} else {
			stats.Inventory[cardKey]--
			if stats.Inventory[cardKey] <= 0 {
				delete(stats.Inventory, cardKey)
			}
		}
	}
	if bundle.WeaponID != "" {
		if add {
			stats.Inventory[bundle.WeaponID]++
		} else {
			stats.Inventory[bundle.WeaponID]--
			if stats.Inventory[bundle.WeaponID] <= 0 {
				delete(stats.Inventory, bundle.WeaponID)
			}
		}
	}
	if bundle.FaceplateID != "" {
		if add {
			stats.Inventory[bundle.FaceplateID]++
		} else {
			stats.Inventory[bundle.FaceplateID]--
			if stats.Inventory[bundle.FaceplateID] <= 0 {
				delete(stats.Inventory, bundle.FaceplateID)
			}
		}
	}

	// PILLAR 3: Identity Hardening. 
	// Recalculate reputation to ensure the borrower's social standing is valid.
	stats.Reputation = l.CalculateReputation(stats)
	l.leaderboard[wallet] = stats
}
