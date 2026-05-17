//go:build !js || !wasm

package main

import (
	"encoding/json"
	"fmt"
	"time"
)

// processLoans checks for defaulted loans and handles collateral liquidation.
func (l *Lobby) processLoans() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	now := time.Now()
	anyProcessed := false

	for id, loan := range l.loans {
		if loan.Status == "active" && now.After(loan.DueAt) {
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
				// INDUSTRIAL LOOP: Deduct distributed fee from liquid faucet balance
				l.faucetBalance -= liquidationFeeBase
				owningClub.LastActivity = now
				feeRecipient = owningClub.ID
				l.logAdminAuditLocked("LOAN_LIQUIDATION_FEE", loan.TerritoryID, fmt.Sprintf("Club %s earned %.2f $VBV liquidation fee", owningClub.Name, liquidationFeeBase))
			}

			// PILLAR 3: Financial Proof.
			// Record loan liquidation (default) on-chain for the audit trail.
			liquidateDetails := map[string]interface{}{
				"id":            loan.ID,
				"wallet":        borrowerWallet,
				"collateral":    loan.CollateralBundle,
				"territory_id":  loan.TerritoryID,
				"fee_recipient": feeRecipient,
				"fee_amount":    liquidationFeeBase,
				"ts":            now.Unix(),
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
	}

	if anyProcessed {
		// Trigger global scaling recalculation to reflect the shift in liquid reserves
		l.applyDynamicScalingLocked()
		msg := l.getLobbyUpdateMsgLocked()
		go func() { l.broadcast <- msg }()
	}
}
