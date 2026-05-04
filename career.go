package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// handleHirePlayer allows Club owners to assign roles to other players.
func (l *Lobby) handleHirePlayer(env *Envelope) {
	var data struct {
		ClubID   string `json:"club_id"`
		TargetID string `json:"target_id"`
		Role     string `json:"role"` // Manager, Security, Clerk
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		log.Printf("[CAREER] Invalid hire_player payload from %s: %v\n", env.FromID, err)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	ownerWallet, ok := l.wallets[env.FromID]
	if !ok {
		log.Printf("[CAREER] Hire failed: Sender %s not connected.\n", env.FromID)
		return
	}

	club, exists := l.clubs[data.ClubID]
	if !exists || strings.ToLower(club.OwnerWallet) != strings.ToLower(ownerWallet) {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Hiring Failed: Unauthorized or Club not found."}`)})
		log.Printf("[CAREER] Hire failed for %s: Unauthorized or Club %s not found.\n", ownerWallet, data.ClubID)
		return
	}

	targetWallet, targetConnected := l.wallets[data.TargetID]
	if !targetConnected {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Hiring Failed: Player not found in lobby."}`)})
		log.Printf("[CAREER] Hire failed for %s: Target player %s not in lobby.\n", ownerWallet, data.TargetID)
		return
	}

	// Update target player stats (Employment Record)
	stats := l.leaderboard[targetWallet]
	stats.JobRole = data.Role
	stats.EmployerClubID = data.ClubID
	l.leaderboard[targetWallet] = stats

	// Update club staffing map
	if club.Staff == nil {
		club.Staff = make(map[string]string)
	}
	club.Staff[strings.ToLower(targetWallet)] = data.Role
	l.clubs[data.ClubID] = club

	l.logAdminAudit("PLAYER_HIRED", targetWallet, fmt.Sprintf("Club: %s (%s), Role: %s", club.Name, club.ID, data.Role))

	// Notify the employee of their new position
	notification, _ := json.Marshal(map[string]string{
		"text": fmt.Sprintf("💼 <b>JOB ASSIGNMENT:</b> You are now the %s for %s!", data.Role, club.Name),
	})
	l.sendToClient(data.TargetID, Envelope{Type: "admin_notification", Payload: notification})

	// Trigger global sync to update UI roles and badges
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	log.Printf("[CAREER] Player %s hired by %s as %s.\n", targetWallet, club.Name, data.Role)
}

// handleSetSalary allows Club owners to set salaries for their employees.
func (l *Lobby) handleSetSalary(env *Envelope) {
	var data struct {
		ClubID      string  `json:"club_id"`
		TargetWallet string `json:"target_wallet"`
		SalaryAmount float64 `json:"salary_amount"` // In whole $VBV units
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		log.Printf("[CAREER] Invalid set_salary payload from %s: %v\n", env.FromID, err)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	ownerWallet, ok := l.wallets[env.FromID]
	if !ok {
		log.Printf("[CAREER] Set salary failed: Sender %s not connected.\n", env.FromID)
		return
	}

	club, exists := l.clubs[data.ClubID]
	if !exists || strings.ToLower(club.OwnerWallet) != strings.ToLower(ownerWallet) {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Salary Failed: Unauthorized or Club not found."}`)})
		log.Printf("[CAREER] Set salary failed for %s: Unauthorized or Club %s not found.\n", ownerWallet, data.ClubID)
		return
	}

	// Ensure target is actually employed by this club
	if club.Staff == nil || strings.ToLower(club.Staff[strings.ToLower(data.TargetWallet)]) == "" {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Salary Failed: Player not employed by this club."}`)})
		log.Printf("[CAREER] Set salary failed for %s: Player %s not employed by Club %s.\n", ownerWallet, data.TargetWallet, data.ClubID)
		return
	}

	// Update player's salary
	stats := l.leaderboard[data.TargetWallet]
	stats.Salary = uint64(data.SalaryAmount * 1000000) // Store in micro-units
	l.leaderboard[data.TargetWallet] = stats

	l.logAdminAudit("SET_SALARY", data.TargetWallet, fmt.Sprintf("Club: %s (%s), Amount: %.2f $VBV", club.Name, club.ID, data.SalaryAmount))

	// Notify the employee of their new salary
	notification, _ := json.Marshal(map[string]string{
		"text": fmt.Sprintf("💰 <b>SALARY UPDATE:</b> Your new salary at %s is %.2f $VBV per day!", club.Name, data.SalaryAmount),
	})
	l.sendToClient(l.getClientIDFromWallet(data.TargetWallet), Envelope{Type: "admin_notification", Payload: notification})

	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	log.Printf("[CAREER] Salary for %s set to %.2f $VBV by %s.\n", data.TargetWallet, data.SalaryAmount, ownerWallet)
}

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
