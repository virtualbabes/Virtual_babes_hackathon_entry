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

	for id, loan := range l.loans {
		if loan.Status == "active" && now.After(loan.DueAt) {
			loan.Status = "defaulted"

			// Residual Value: 15% of the loan amount is returned as Market Tokens
			tokenReward := (loan.LoanAmount*15 + 50) / 100 // Round to nearest micro-unit to prevent fractional dust

			borrowerWallet := loan.BorrowerWallet
			borrowerStats, exists := l.leaderboard[borrowerWallet]
			if exists {
				borrowerStats.MarketTokens += tokenReward
				borrowerStats.Reputation -= 50
				if borrowerStats.Reputation < 0 {
					borrowerStats.Reputation = 0
				}
				// RECONCILE: Ensure calculated stats are in sync
				borrowerStats.Reputation = l.CalculateReputation(borrowerStats)
				l.leaderboard[borrowerWallet] = borrowerStats

				l.sendToClient(l.getClientIDFromWallet(borrowerWallet), Envelope{
					Type:    "admin_notification",
					Payload: json.RawMessage(fmt.Sprintf(`{"text":"🚨 <b>LOAN DEFAULTED:</b> Collateral moved to Black Market. You received %.2f Market Tokens as equity."}`, float64(tokenReward)/1000000.0)),
				})
			}

			// Update playstyle on loan default (Internal call to avoid deadlock)
			l.updatePlayerPlaystyleTendenciesLocked(borrowerWallet, false, [2]int{}, []int{}, false)
			l.logAdminAudit("LOAN_LIQUIDATED", borrowerWallet, fmt.Sprintf("ID: %s, Tokens: %d", loan.ID, tokenReward))

			// Add the defaulted loan to the black market
			l.blackMarket = append(l.blackMarket, *loan)

			delete(l.loans, id)
			// Trigger UI sync to reflect changes in the black market and player stats
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		}
	}
}