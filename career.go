package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// startSalaryDispenser runs a daily ticker to pay salaries from Club Treasuries.
func (l *Lobby) startSalaryDispenser() {
	salaryTicker := time.NewTicker(24 * time.Hour) // Daily payment cycle
	defer salaryTicker.Stop()

	for range salaryTicker.C {
		log.Println("[CAREER] Running daily salary dispenser...")
		l.mutex.Lock()
		for wallet, stats := range l.leaderboard {
			if stats.JobRole != "" && stats.EmployerClubID != "" && stats.Salary > 0 {
				if time.Since(stats.LastSalaryPayment) >= 24*time.Hour {
					club, exists := l.clubs[stats.EmployerClubID]
					if exists && club.Treasury >= float64(stats.Salary)/1000000.0 {
						// Industrial Loop: Gross salary deducted from Club Treasury
						club.Treasury -= float64(stats.Salary) / 1000000.0
						club.LastActivity = time.Now()

						// Outlaw Tax Logic: garnish earnings based on infamy
						taxRate := 0.0
						if stats.WantedLevel >= 5 {
							taxRate = float64(stats.WantedLevel) * 0.02
							if taxRate > 0.40 {
								taxRate = 0.40
							}
						}

						// PILLAR 1: Precision Rounding for the Industrial Loop.
						taxAmountMicro := uint64(float64(stats.Salary)*taxRate + 0.5)
						netSalaryMicro := stats.Salary - taxAmountMicro

						l.rewards[wallet] += netSalaryMicro
						if taxAmountMicro > 0 {
							l.faucetBalance += float64(taxAmountMicro) / 1000000.0
							l.applyDynamicScalingLocked()
						}

						stats.LastSalaryPayment = time.Now()

						// PILLAR 1: Career Service Update.
						// Re-sync reputation to reflect the ongoing service and potential club mojo shifts.
						stats.Reputation = l.CalculateReputation(stats)
						l.leaderboard[wallet] = stats

						l.logAdminAuditLocked("SALARY_PAID", wallet, fmt.Sprintf("Club: %s, Net: %.2f, Tax: %.2f", club.Name, float64(netSalaryMicro)/1000000.0, float64(taxAmountMicro)/1000000.0))
						notification := fmt.Sprintf(`{"text":"💰 <b>SALARY PAID:</b> You received %.2f $VBV from %s! (Outlaw Tax: %.2f $VBV)"}`, float64(netSalaryMicro)/1000000.0, club.Name, float64(taxAmountMicro)/1000000.0)
						l.sendToClientLocked(l.getClientIDFromWalletLocked(wallet), Envelope{Type: "admin_notification", Payload: json.RawMessage(notification)})
					} else {
						log.Printf("[CAREER] Club %s has insufficient funds to pay %s's salary or club not found.\n", stats.EmployerClubID, wallet)
					}
				}
			}
		}
		l.mutex.Unlock()
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	}
}
