//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// unlockAchievement checks if a player already has an achievement; if not, it unlocks it and notifies them.
func (l *Lobby) unlockAchievement(wallet, id string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.unlockAchievementLocked(wallet, id)
}

// unlockAchievementLocked checks if a player already has an achievement, assuming the lock is held.
func (l *Lobby) unlockAchievementLocked(wallet, id string) {
	targetWallet := strings.ToLower(wallet)
	l.ensurePlayerStatsMapsInitialized(targetWallet)
	stats, exists := l.leaderboard[targetWallet]

	// PILLAR 3: Identity Defense.
	// If the player record is somehow missing after initialization attempt, abort.
	if !exists {
		log.Printf("[ACHIEVEMENT ERROR] Attempted to grant trophy %s to missing wallet %s\n", id, targetWallet)
		return
	}

	// Prevention: Ensure achievement is only granted once
	for _, a := range stats.Achievements {
		if a == id {
			return
		}
	}

	// Unlock trophy
	stats.Achievements = append(stats.Achievements, id)
	// Update reputation to reflect the social impact of the new achievement
	stats.Reputation = l.CalculateReputation(stats)
	l.leaderboard[targetWallet] = stats

	l.logAdminAuditLocked("ACHIEVEMENT_UNLOCKED", targetWallet, id)

	// Notify all client sessions associated with this wallet
	msg, _ := json.Marshal(map[string]string{
		"text": fmt.Sprintf("🏆 <b>TROPHY UNLOCKED:</b> %s", strings.ReplaceAll(id, "_", " ")),
	})

	for cid, w := range l.wallets {
		if strings.ToLower(w) == wallet {
			l.sendToClientLocked(cid, Envelope{
				Type:    "admin_notification",
				FromID:  "SERVER",
				Payload: msg,
			})
		}
	}

	// Broadcast lobby update using the Locked snapshot pattern
	msg = l.getLobbyUpdateMsgLocked()
	go func() { l.broadcast <- msg }()

	// PILLAR 3: Milestone Persistence. Ensure trophies survive server restarts immediately.
	go l.saveLeaderboard()
}

// checkTreasuryRecoveryAchievementLocked evaluates if a club owner qualifies for the recovery trophy.
// PILLAR 1: Organizational Achievement.
func (l *Lobby) checkTreasuryRecoveryAchievementLocked(clubID string, currentTreasury, avg float64) {
	if l.treasuryCrashed[clubID] && currentTreasury > (avg*0.50) {
		l.treasuryCrashed[clubID] = false
		if club, ok := l.clubs[clubID]; ok && club.OwnerWallet != "" {
			l.unlockAchievementLocked(strings.ToLower(club.OwnerWallet), "TREASURY_RECOVERY")
		}
	}
}

// checkMojoSurgeAchievementLocked evaluates if a club owner qualifies for the Mojo Surge trophy.
// This achievement unlocks when a club's Mojo increases by 100 points within a 24-hour period.
// PILLAR 1: Organizational Achievement.
func (l *Lobby) checkMojoSurgeAchievementLocked(clubID string) {
	club, ok := l.clubs[clubID]
	if !ok || club.OwnerWallet == "" {
		return
	}

	// This logic assumes the Club struct (in common_types.go) has the following fields:
	// MojoStartOf24hWindow int       // Mojo value at the start of the current 24h window
	// MojoWindowStartTime  time.Time // Timestamp when the current 24h window began

	// PILLAR 1: Temporal Integrity.
	// If the window hasn't been initialized, has expired, or the system clock was shifted backward, reset it.
	elapsed := time.Since(club.MojoWindowStartTime)
	if club.MojoWindowStartTime.IsZero() || elapsed > 24*time.Hour || elapsed < 0 {
		club.MojoStartOf24hWindow = club.Mojo
		club.MojoWindowStartTime = time.Now()
		l.clubs[clubID] = club // Update the club in the map
		return
	}

	// Check for Mojo surge within the current 24-hour window
	if club.Mojo-club.MojoStartOf24hWindow >= 100 {
		l.unlockAchievementLocked(strings.ToLower(club.OwnerWallet), "MOJO_SURGE")
		// Reset the window immediately after achievement unlock to start tracking for the next surge.
		club.MojoStartOf24hWindow = club.Mojo
		club.MojoWindowStartTime = time.Now()
		l.clubs[clubID] = club // Update the club in the map
	}
}

// checkHeistSaboteurAchievementLocked evaluates if a player qualifies for the Heist Saboteur trophy.
func (l *Lobby) checkHeistSaboteurAchievementLocked(wallet string) {
	stats := l.leaderboard[strings.ToLower(wallet)]
	if stats.HeistAlarmsJammerCount >= 3 {
		l.unlockAchievementLocked(wallet, "HEIST_SABOTEUR")
	}
}

// checkCorporateEspionageAchievementLocked evaluates if a player qualifies for the espionage trophy.
// PILLAR 3: Criminality & Intelligence.
func (l *Lobby) checkCorporateEspionageAchievementLocked(wallet string) {
	targetWallet := strings.ToLower(wallet)
	stats := l.leaderboard[targetWallet]

	rivalAudits := 0
	for clubID := range stats.AuditedClubs {
		auditedClub, exists := l.clubs[clubID]
		if !exists {
			continue
		}

		// PILLAR 3: Espionage Integrity.
		// Only unique RIVAL audits (no affiliation) count towards the trophy.
		if l.isPlayerAffiliatedWithClubLocked(targetWallet, auditedClub) {
			continue
		}
		rivalAudits++
	}

	if rivalAudits >= 5 {
		l.unlockAchievementLocked(wallet, "CORPORATE_ESPIONAGE")
	}
}
