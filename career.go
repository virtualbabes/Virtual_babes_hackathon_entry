//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"
	"time"
)

// CareerService manages the automated distribution of salaries to club employees.
// PILLAR 5: Stateless Service Design.
type CareerService struct{}

// StartSalaryDispenser runs a daily ticker to pay salaries from Club Treasuries.
func (s *CareerService) StartSalaryDispenser(l *Lobby) {
	salaryTicker := time.NewTicker(24 * time.Hour) // Daily payment cycle
	defer salaryTicker.Stop()

	for range salaryTicker.C {
		log.Println("[CAREER] Running daily salary dispenser...")
		l.mutex.Lock()
		anyPaid := false
		anyPruned := false
		for wallet, stats := range l.leaderboard {
			if stats.JobRole != "" && stats.EmployerClubID != "" && stats.Salary > 0 {
				if time.Since(stats.LastSalaryPayment) >= 24*time.Hour {
					club, exists := l.clubs[stats.EmployerClubID]
					// PILLAR 2: Integer Supremacy. Use TreasuryMicro for authoritative checks.
					if exists && club.TreasuryMicro >= stats.Salary {
						// Industrial Loop: Gross salary deducted from Club Treasury micro-unit reservoir.
						club.TreasuryMicro -= stats.Salary
						club.LastActivity = time.Now()

						// PILLAR 1: Corporate Tax Logic.
						// Deduct 2% from high-salary contracts (>= 500 $VBV) to fund the Global Faucet.
						corpTaxMicro := uint64(0)
						if stats.Salary >= 500*1000000 {
							corpTaxMicro = uint64(float64(stats.Salary)*0.02 + 0.5)
							l.CorporateTaxTotal += corpTaxMicro
							l.CorporateTaxCount++

							// PILLAR 3: Lawyer-Commissioner Influence.
							// If a 'Regulatory Bypass Permit' is active for the club, reduce corporate tax by 50%.
							if expiry, exists := club.BuffExpirations["REGULATORY_BYPASS"]; exists && time.Now().Before(expiry) {
								corpTaxMicro = (corpTaxMicro * 50) / 100
								l.logAdminAuditLocked("REGULATORY_BYPASS_APPLIED", club.ID, fmt.Sprintf("Corporate tax reduced by 50%% for %s", club.Name))
							}

							l.achievementService.CheckTaxMilestoneAchievementLocked(l)
						}

						// Outlaw Tax Logic: garnish earnings based on infamy
						outlawTaxRate := 0.0
						if stats.WantedLevel >= 5 {
							outlawTaxRate = float64(stats.WantedLevel) * 0.02
							if outlawTaxRate > 0.40 {
								outlawTaxRate = 0.40
							}
						}

						// PILLAR 1: Precision Rounding for the Industrial Loop.
						outlawTaxMicro := uint64(float64(stats.Salary)*outlawTaxRate + 0.5)
						totalTaxMicro := corpTaxMicro + outlawTaxMicro
						netSalaryMicro := stats.Salary - totalTaxMicro

						l.playerBalances[wallet] += netSalaryMicro

						// PILLAR 2: Ledger Integrity (Industrial Loop).
						// Reroute taxes to the Faucet pool to fund dynamic rewards.
						l.faucetBalanceMicro += totalTaxMicro
						l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0

						stats.LastSalaryPayment = time.Now()

						// PILLAR 1: Career Service Update.
						// Re-sync reputation to reflect the ongoing service and potential club mojo shifts.
						stats.Reputation = l.CalculateReputation(stats)
						l.leaderboard[wallet] = stats

						l.logAdminAuditLocked("SALARY_PAID", wallet, fmt.Sprintf("Club: %s, Net: %.2f, CorpTax: %.2f, OutlawTax: %.2f",
							club.Name, float64(netSalaryMicro)/1000000.0, float64(corpTaxMicro)/1000000.0, float64(outlawTaxMicro)/1000000.0))

						notification := fmt.Sprintf(`{"text":"💰 <b>SALARY PAID:</b> You received %.2f $VBV from %s! (Taxes: %.2f $VBV)"}`,
							float64(netSalaryMicro)/1000000.0, club.Name, float64(totalTaxMicro)/1000000.0)
						l.sendToClientLocked(l.getClientIDFromWalletLocked(wallet), Envelope{Type: "admin_notification", Payload: json.RawMessage(notification)})
						anyPaid = true
					} else {
						// PILLAR 1: Default Pruning.
						// If the club is gone or insolvent, the contract is terminated.
						clubID := stats.EmployerClubID
						reason := "Insolvency"
						sourceName := clubID
						if !exists {
							reason = "Dissolution"
						} else {
							sourceName = club.Name
							delete(club.Staff, strings.ToLower(wallet))
						}
						log.Printf("[CAREER] Club %s defaulted (%s). Terminating %s's contract.\n", clubID, reason, wallet)
						stats.JobRole = "Freelancer"
						stats.EmployerClubID = ""
						stats.Salary = 0
						stats.LastSalaryPayment = time.Time{} // Clear clock to prevent immediate payment on re-hire
						stats.Reputation = l.CalculateReputation(stats)
						l.leaderboard[wallet] = stats
						l.logAdminAuditLocked("CAREER_DEFAULT", wallet, fmt.Sprintf("Club: %s, Reason: %s", clubID, reason))
						notification := fmt.Sprintf(`{"text":"⚠️ <b>CONTRACT TERMINATED:</b> %s failed to pay your salary (%s). You are now a Freelancer."}`, escapeHTML(sourceName), reason)
						l.sendToClientLocked(l.getClientIDFromWalletLocked(wallet), Envelope{Type: "admin_notification", Payload: json.RawMessage(notification)})
						anyPruned = true
					}
				}
			}
		}
		if anyPaid {
			l.applyDynamicScalingLocked()
		}
		l.mutex.Unlock()

		if anyPaid || anyPruned {
			go l.saveLeaderboard()
		}
		if anyPaid {
			go l.saveEconomyState()
		}
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	}
}
