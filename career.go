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
						club.Treasury -= float64(stats.Salary) / 1000000.0
						l.rewards[wallet] += stats.Salary
						stats.LastSalaryPayment = time.Now()
						l.leaderboard[wallet] = stats
						l.logAdminAudit("SALARY_PAID", wallet, fmt.Sprintf("Club: %s, Amount: %.2f $VBV", club.Name, float64(stats.Salary)/1000000.0))
						l.sendToClient(l.getClientIDFromWallet(wallet), Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>SALARY PAID:</b> You received %.2f $VBV from %s!"}`, float64(stats.Salary)/1000000.0, club.Name))})
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
