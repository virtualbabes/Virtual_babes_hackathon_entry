//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"
)

// AchievementService manages the unlocking and tracking of player milestones and trophies.
// PILLAR 5: Stateless Service Design.
type AchievementService struct{}

// unlockAchievement checks if a player already has an achievement; if not, it unlocks it and notifies them.
func (s *AchievementService) UnlockAchievement(l *Lobby, wallet, id string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	s.UnlockAchievementLocked(l, wallet, id)
}

// unlockAchievementLocked checks if a player already has an achievement, assuming the lock is held.
func (s *AchievementService) UnlockAchievementLocked(l *Lobby, wallet, id string) {
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

	// PILLAR 3: Identity Hardening. 
	// Ensure notifications target the canonical normalized wallet.
	for cid, w := range l.wallets {
		if strings.EqualFold(w, targetWallet) {
			l.sendToClientLocked(cid, Envelope{
				Type:    "admin_notification",
				FromID:  "SERVER",
				Payload: msg,
			})
		}
	}

	// PILLAR 4: Snapshot Pattern.
	// Capture a point-in-time state of the lobby to avoid blocking the loop during I/O.
	msg = l.getLobbyUpdateMsgLocked()
	go func() { l.broadcast <- msg }()

	// PILLAR 3: Milestone Persistence. Ensure trophies survive server restarts immediately.
	go l.saveLeaderboard()
}

// checkTreasuryRecoveryAchievementLocked evaluates if a club owner qualifies for the recovery trophy.
// PILLAR 1: Organizational Achievement.
func (s *AchievementService) CheckTreasuryRecoveryAchievementLocked(l *Lobby, clubID string, currentTreasury, avg float64) {
	if l.treasuryCrashed[clubID] && currentTreasury > (avg*0.50) {
		l.treasuryCrashed[clubID] = false
		if club, ok := l.clubs[clubID]; ok && club.OwnerWallet != "" {
			s.UnlockAchievementLocked(l, strings.ToLower(club.OwnerWallet), "TREASURY_RECOVERY")
		}
	}
}

// checkMojoSurgeAchievementLocked evaluates if a club owner qualifies for the Mojo Surge trophy.
// This achievement unlocks when a club's Mojo increases by 100 points within a 24-hour period.
// PILLAR 1: Organizational Achievement.
func (s *AchievementService) CheckMojoSurgeAchievementLocked(l *Lobby, clubID string) {
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
		s.UnlockAchievementLocked(l, strings.ToLower(club.OwnerWallet), "MOJO_SURGE")
		// Reset the window immediately after achievement unlock to start tracking for the next surge.
		club.MojoStartOf24hWindow = club.Mojo
		club.MojoWindowStartTime = time.Now()
		l.clubs[clubID] = club // Update the club in the map
	}
}

// checkHeistSaboteurAchievementLocked evaluates if a player qualifies for the Heist Saboteur trophy.
func (s *AchievementService) CheckHeistSaboteurAchievementLocked(l *Lobby, wallet string) {
	targetWallet := strings.ToLower(wallet)
	l.ensurePlayerStatsMapsInitialized(targetWallet)
	stats := l.leaderboard[targetWallet]
	if stats.HeistAlarmsJammerCount >= 3 {
		s.UnlockAchievementLocked(l, wallet, "HEIST_SABOTEUR")
	}
}

// checkCorporateEspionageAchievementLocked evaluates if a player qualifies for the espionage trophy.
// PILLAR 3: Criminality & Intelligence.
func (s *AchievementService) CheckCorporateEspionageAchievementLocked(l *Lobby, wallet string) {
	targetWallet := strings.ToLower(wallet)
	l.ensurePlayerStatsMapsInitialized(targetWallet)
	stats := l.leaderboard[targetWallet]

	rivalAudits := 0
	for clubID := range stats.AuditedClubs {
		auditedClub, exists := l.clubs[clubID]
		if !exists {
			continue
		}

		// PILLAR 3: Espionage Integrity.
		if l.clubService.IsPlayerAffiliatedWithClubLocked(l, targetWallet, auditedClub) {
			continue
		}
		rivalAudits++
	}

	if rivalAudits >= 5 {
		s.UnlockAchievementLocked(l, wallet, "CORPORATE_ESPIONAGE")
	}
}

// checkBountyHunterAchievementLocked monitors progress towards the Bounty Hunter trophy.
// Unlocks after 3 unique outlaws (Wanted 15+) are successfully captured.
// PILLAR 3: Criminality & Intelligence.
func (s *AchievementService) CheckBountyHunterAchievementLocked(l *Lobby, hunterWallet string, victimWallet string, victimWantedAtMatch int) {
	// 1. Condition Verification: Victim must have been a high-infamy outlaw at the time of match conclusion.
	if victimWantedAtMatch < 15 {
		return
	}

	hWallet := strings.ToLower(hunterWallet)
	vWallet := strings.ToLower(victimWallet)

	// 2. State Prep
	l.ensurePlayerStatsMapsInitialized(hWallet)
	stats := l.leaderboard[hWallet]

	if stats.CapturedOutlaws == nil {
		stats.CapturedOutlaws = make(map[string]bool)
	}

	// 3. Unique Tracking
	if !stats.CapturedOutlaws[vWallet] {
		stats.CapturedOutlaws[vWallet] = true
		l.leaderboard[hWallet] = stats

		log.Printf("[CRIMINALITY] %s tracked unique capture of %s (Wanted: %d). Progress: %d/3\n", 
			hWallet, vWallet, victimWantedAtMatch, len(stats.CapturedOutlaws))

		if len(stats.CapturedOutlaws) >= 3 {
			s.UnlockAchievementLocked(l, hWallet, "BOUNTY_HUNTER")
		}
	}
}

// checkTaxMilestoneAchievementLocked evaluates if the systemic tax revenue has reached the milestone.
// PILLAR 1: Industrial Loop.
func (s *AchievementService) CheckTaxMilestoneAchievementLocked(l *Lobby) {
	totalTax := l.CorporateTaxTotal + l.LuxuryTaxTotal
	// Milestone: 10,000 VBV (10,000,000,000 micro-units)
	target := uint64(10000 * 1000000)

	if totalTax >= target && l.vaultAddress != "" {
		// Verify if already unlocked to prevent broadcast spam
		stats, exists := l.leaderboard[l.vaultAddress]
		alreadyUnlocked := false
		if exists {
			for _, a := range stats.Achievements {
				if a == "ECOSYSTEM_GUARDIAN" { alreadyUnlocked = true; break }
			}
		}

		if !alreadyUnlocked {
			s.UnlockAchievementLocked(l, l.vaultAddress, "ECOSYSTEM_GUARDIAN")
			
			// PILLAR 1: Global Milestone Broadcast.
			payload, _ := json.Marshal(map[string]string{
				"text": "🛡️ <b>SYSTEM MILESTONE REACHED:</b> ECOSYSTEM GUARDIAN. The Arena's Industrial Loop has recovered 10,000 $VBV!",
				"type": "critical",
			})
			l.broadcast <- jsonListEnvelope("admin_notification", payload)
		}
	}
}

// checkPhilanthropistAchievementLocked evaluates major donations for the Philanthropist trophy.
// PILLAR 1: Social Economic Simulation.
func (s *AchievementService) CheckPhilanthropistAchievementLocked(l *Lobby, wallet string, amountMicro uint64) {
	// Criteria: Single donation of 1,000 $VBV or more.
	if amountMicro < 1000*1000000 {
		return
	}

	// 25% chance per major donation to unlock the rare trophy
	if rand.Float64() < 0.25 {
		s.UnlockAchievementLocked(l, wallet, "PHILANTHROPIST")
	} else {
		log.Printf("[SOCIAL] %s attempted major donation, but luck failed for Philanthropist trophy.\n", wallet)
	}
}

// checkGenerosityMilestoneLocked evaluates cumulative donations for the Patron of the Arts trophy.
// PILLAR 1: Industrial Loop & Social Standing.
func (s *AchievementService) CheckGenerosityMilestoneLocked(l *Lobby, wallet string) {
	targetWallet := strings.ToLower(wallet)
	l.ensurePlayerStatsMapsInitialized(targetWallet)
	stats := l.leaderboard[targetWallet]
	// Milestone: 5,000 VBV (5,000,000,000 micro-units)
	if stats.TotalDonated >= 5000*1000000 {
		s.UnlockAchievementLocked(l, wallet, "PATRON_OF_THE_ARTS")
	}
}

// TransferBundleItems handles adding or removing items from a player's inventory via achievement rewards.
// If add is true, items are added. If add is false, items are removed.
// It assumes the lobby mutex is held.
// PILLAR 5: Modular Orchestration.
func (s *AchievementService) TransferBundleItems(l *Lobby, wallet string, bundle CardBundle, add bool) {
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
	l.leaderboard[wallet] = stats
}
