//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// applyItemEffect centralizes the logic for applying item effects.
// It is called by lobby_manager.go's handleUseItem.
// This function assumes the main lobby mutex is already held by the caller.
// It modifies the passed playerStats pointer and returns a notification message or an error.
func (l *Lobby) applyItemEffect(env *Envelope, data UseItemData, wallet string, playerStats *PlayerStats, item ShopItem) (string, error) {
	var notificationText string
	match, inMatch := l.matches[env.FromID] // Check if player is in a match

	switch item.ClubType {
	case "Vitality": // Stamina Stim, Loyalty Pledge (affect PlayerStats or specific cards in global inventory)
		if data.TargetCardID == 0 { // Must target a specific card
			return "", fmt.Errorf("vitality items require a target card")
		}
		targetCard, cardExists := l.inventory[data.TargetCardID]
		if !cardExists {
			return "", fmt.Errorf("target card not found")
		}

		switch data.ItemID {
		case "stamina_stim":
			targetCard.Fatigue -= 20
			if targetCard.Fatigue < 0 {
				targetCard.Fatigue = 0
			}
			notificationText = fmt.Sprintf("⚡ %s's Fatigue reduced by 20!", targetCard.Name)
		case "loyalty_pledge":
			targetCard.Loyalty += 10
			if targetCard.Loyalty > 100 {
				targetCard.Loyalty = 100
			}
			notificationText = fmt.Sprintf("💖 %s's Loyalty increased by 10!", targetCard.Name)
		}
		l.inventory[data.TargetCardID] = targetCard           // Update global card cache
		l.persistentCardCache[data.TargetCardID] = targetCard // Update persistent cache

		isBounty := false
		isTourney := false
		if inMatch {
			isBounty, isTourney = match.IsBountyMatch, match.TournamentMatchID != ""
		}
		l.updatePlayerPlaystyleTendenciesLocked(wallet, false, [2]int{}, []int{}, isBounty, isTourney)

		playerStats.Playstyle = l.leaderboard[wallet].Playstyle
		playerStats.Reputation = l.CalculateReputation(*playerStats)

	case "Elemental", "Tactical":
		// PILLAR 1: Infrastructure Prestige.
		// The District Stabilizer is a tactical upgrade for the club, not a match modifier.
		if data.ItemID == "district_stabilizer" {
			if playerStats.JobRole != "Manager" && playerStats.JobRole != "CEO" {
				return "", fmt.Errorf("manager role required to activate stabilizer")
			}
			targetClub, clubExists := l.clubs[playerStats.EmployerClubID]
			if !clubExists {
				return "", fmt.Errorf("club records corrupted")
			}

			if targetClub.ActiveBuffs == nil {
				targetClub.ActiveBuffs = make(map[string]string)
			}
			if targetClub.BuffExpirations == nil {
				targetClub.BuffExpirations = make(map[string]time.Time)
			}

			// Set the suppression field with a 48-hour expiration
			targetClub.ActiveBuffs["MOJO_STABILIZER"] = data.ItemID
			targetClub.BuffExpirations["MOJO_STABILIZER"] = time.Now().Add(48 * time.Hour)
			targetClub.LastActivity = time.Now()
			targetClub.Mojo += item.MojoBonus

			// Ripple Effect: members Standing reflects the new Mojo immediately
			for w, s := range l.leaderboard {
				if s.EmployerClubID == targetClub.ID {
					s.Reputation = l.CalculateReputation(s)
					l.leaderboard[w] = s
				}
			}
			l.clubs[playerStats.EmployerClubID] = targetClub
			l.updatePlayerPlaystyleTendenciesLocked(wallet, false, [2]int{}, []int{}, false, false)
			l.checkMojoSurgeAchievementLocked(targetClub.ID)
			return fmt.Sprintf("📡 <b>%s ACTIVATED:</b> Mojo decay for %s is now suppressed for 48 hours.", item.Name, targetClub.Name), nil
		}

		if !inMatch {
			return "", fmt.Errorf("this item can only be used during a match")
		}
		// Delegate to battle_service for in-match effects
		l.applyItemEffectToMatch(match, env.FromID, data.ItemID, data.TargetCardID, data.TargetGridIndex)
		notificationText = fmt.Sprintf("✨ %s activated!", item.Name)

		isBounty := false
		isTourney := false
		if inMatch {
			isBounty, isTourney = match.IsBountyMatch, match.TournamentMatchID != ""
		}
		l.updatePlayerPlaystyleTendenciesLocked(wallet, true, [2]int{}, []int{}, isBounty, isTourney)

		playerStats.Playstyle = l.leaderboard[wallet].Playstyle
		playerStats.Reputation = l.CalculateReputation(*playerStats)

	case "Hardware": // Traps: tripwire, sentry_turret, guard_dog
		if playerStats.JobRole != "Security" || playerStats.EmployerClubID == "" {
			return "", fmt.Errorf("security role required to deploy hardware")
		}

		targetClub, clubExists := l.clubs[playerStats.EmployerClubID]
		if !clubExists {
			return "", fmt.Errorf("club data corrupted")
		}

		// Guardrail: Max 3 Active Traps per Club
		activeTraps := 0
		for key := range targetClub.ActiveBuffs {
			if strings.HasPrefix(key, "TRAP_") {
				activeTraps++
			}
		}
		if activeTraps >= 3 {
			return "", fmt.Errorf("maximum defense capacity (3/3) reached")
		}

		// Initialize maps if nil
		if targetClub.ActiveBuffs == nil {
			targetClub.ActiveBuffs = make(map[string]string)
		}
		if targetClub.BuffExpirations == nil {
			targetClub.BuffExpirations = make(map[string]time.Time)
		}

		// Deploy Trap with 24-hour expiration
		trapID := fmt.Sprintf("TRAP_%d", time.Now().UnixNano())
		targetClub.ActiveBuffs[trapID] = data.ItemID
		targetClub.BuffExpirations[trapID] = time.Now().Add(24 * time.Hour)
		targetClub.LastActivity = time.Now() // Mark club as active

		// PILLAR 1: Infrastructure Prestige.
		// Deployment of hardware traps rewards the club with Mojo, increasing its social standing and unlocking premium tiers.
		targetClub.Mojo += item.MojoBonus
		l.checkMojoSurgeAchievementLocked(targetClub.ID)

		// Ripple Effect: Update Reputation for all club employees to reflect the increased Mojo multiplier.
		for w, s := range l.leaderboard {
			if s.EmployerClubID == targetClub.ID {
				s.Reputation = l.CalculateReputation(s)
				l.leaderboard[w] = s
				if strings.EqualFold(w, wallet) {
					*playerStats = s // Keep the local reference in sync
				}
			}
		}

		l.clubs[playerStats.EmployerClubID] = targetClub
		notificationText = fmt.Sprintf("🛰️ %s deployed in %s's territory!", item.Name, targetClub.Name)

	case "Intelligence": // Cyber-Audit, District Scanner
		switch data.ItemID {
		case "cyber_audit":
			if data.TargetClubID == "" {
				return "", fmt.Errorf("cyber-audit requires a target club")
			}
			l.applyCyberAudit(env.FromID, data.TargetClubID, playerStats)
			notificationText = fmt.Sprintf("🔍 Cyber-Audit initiated on %s.", data.TargetClubID)
		case "district_scanner":
			l.applyDistrictScanner(env.FromID, playerStats)
			notificationText = "📡 <b>DISTRICT SCANNER ACTIVE:</b> Hardware traps are now visible on the world map for 10 minutes."
		case "cyber_jammer":
			playerStats.HasCyberJammer = true
			notificationText = "📡 <b>CYBER-JAMMER ACTIVE:</b> Your next sabotage attempt will be silent."
		}

	default:
		return fmt.Sprintf("❓ Used %s. Effect unknown or not yet implemented.", item.Name), nil
	}

	// Update preferred items after successful use
	// This line is here because it applies to all successful item uses.
	playerStats.Playstyle.PreferredItems[data.ItemID] = playerStats.Playstyle.PreferredItems[data.ItemID]*0.9 + 1.0

	return notificationText, nil
}

// applyCyberAudit reveals a target club's current treasury average and crash status.
// PILLAR 3: Criminality & Intelligence.
func (l *Lobby) applyCyberAudit(clientID string, targetClubID string, stats *PlayerStats) {
	wallet, ok := l.wallets[clientID]
	if !ok {
		return
	}

	targetClub, exists := l.clubs[targetClubID]
	if !exists {
		l.sendToClientLocked(clientID, Envelope{
			Type:    "admin_notification",
			Payload: json.RawMessage(`{"text":"❌ Audit Failed: Target club records are encrypted or missing."}`),
		})
		return
	}

	now := time.Now()

	// PILLAR 3: Sabotage Check.
	// Sabotage disrupts the encryption field, allowing audits to bypass Cyber-Locks.
	sabotaged := false
	if expiry, exists := targetClub.BuffExpirations["SABOTAGE"]; exists {
		if now.Before(expiry) {
			sabotaged = true
		} else {
			delete(targetClub.BuffExpirations, "SABOTAGE")
		}
	}

	// PILLAR 3: Cyber-Lock Check.
	// Identify if the target club has deployed an active encryption field.
	isCyberLockActive := false
	for trapID, itemID := range targetClub.ActiveBuffs {
		if itemID == "cyber_lock" {
			if expiry, exists := targetClub.BuffExpirations[trapID]; exists && now.Before(expiry) {
				isCyberLockActive = true
				break
			}
		}
	}

	if isCyberLockActive && !sabotaged {
		l.sendToClientLocked(clientID, Envelope{
			Type:    "admin_notification",
			Payload: json.RawMessage(fmt.Sprintf(`{"text":"❌ <b>AUDIT BLOCKED:</b> %s has an active Cyber-Lock. Treasury records are encrypted."}`, escapeHTML(targetClub.Name))),
		})
		l.logAdminAuditLocked("CYBER_AUDIT_BLOCKED", wallet, fmt.Sprintf("Target: %s (%s)", targetClub.Name, targetClubID))
		return
	}

	// PILLAR 3: Counter-Intelligence Check.
	isCyberCounterActive := false
	var counterTrapID string
	for trapID, itemID := range targetClub.ActiveBuffs {
		if itemID == "cyber_counter" {
			if expiry, exists := targetClub.BuffExpirations[trapID]; exists && now.Before(expiry) {
				isCyberCounterActive = true
				counterTrapID = trapID
				break
			}
		}
	}

	// Retrieve rolling economic metrics from the Lobby's intelligence engine.
	// These are calculated in processTreasuryAnalytics via Exponential Moving Average.
	avg, hasAvg := l.treasuryAverages[targetClubID]
	if !hasAvg {
		avg = targetClub.Treasury // Fallback for new organizations
	}
	crashed := l.treasuryCrashed[targetClubID]

	status := "STABLE"
	if crashed {
		status = "CRITICAL / VOLATILE"
	}

	sabotagePrefix := ""
	if sabotaged && isCyberLockActive {
		sabotagePrefix = "🔓 <b>ENCRYPTION BREACHED via Sabotage:</b><br/>"
	}

	// PILLAR 3: Intelligence Intel.
	// Provide the economic vulnerability snapshot to the infiltrator.
	msg := fmt.Sprintf("%s🔍 <b>CYBER-AUDIT REPORT: %s</b><br/>Current Treasury: %.2f $VBV<br/>Weekly Average: %.2f $VBV<br/>Security Status: %s",
		sabotagePrefix, escapeHTML(targetClub.Name), targetClub.Treasury, avg, status)

	l.sendToClientLocked(clientID, Envelope{
		Type:    "admin_notification",
		Payload: json.RawMessage(fmt.Sprintf(`{"text":"%s"}`, msg)),
	})

	l.logAdminAuditLocked("CYBER_AUDIT_USED", wallet, fmt.Sprintf("Target: %s (%s)", targetClub.Name, targetClubID))

	// PILLAR 3: Counter-Intelligence Revelation.
	if isCyberCounterActive {
		auditorName := l.ResolveEnvoiName(wallet)                              // ResolveEnvoiName uses its own mutex, so safe.
		ownerClientID := l.getClientIDFromWalletLocked(targetClub.OwnerWallet) // Safe under Lobby lock.
		if ownerClientID != "" {
			counterMsg := fmt.Sprintf(`🚨 <b>CYBER-COUNTER ALERT:</b> %s attempted a Cyber-Audit on your club! (Auditor: %s)`,
				escapeHTML(auditorName), escapeHTML(auditorName))
			l.sendToClientLocked(ownerClientID, Envelope{ // Safe under Lobby lock.
				Type:    "admin_notification",
				Payload: json.RawMessage(fmt.Sprintf(`{"text":"%s"}`, counterMsg)),
			})
			l.logAdminAuditLocked("CYBER_COUNTER_TRIGGERED", targetClub.OwnerWallet, fmt.Sprintf("Auditor: %s (%s)", auditorName, wallet)) // Safe under Lobby lock.
		}
		// Remove the Cyber-Counter after it's triggered once
		delete(targetClub.ActiveBuffs, counterTrapID)
		delete(targetClub.BuffExpirations, counterTrapID)
		log.Printf("[INTELLIGENCE] Cyber-Counter %s triggered and removed for club %s.\n", counterTrapID, targetClub.Name)
	}

	// PILLAR 3: Intelligence Tracking.
	// Record the unique audit and check for achievement qualification.
	if stats.AuditedClubs == nil {
		stats.AuditedClubs = make(map[string]bool)
	}
	stats.AuditedClubs[targetClubID] = true

	// PILLAR 3: State Sync.
	// Explicitly update the leaderboard entry so the achievement check
	// sees the latest unique audit record immediately.
	l.leaderboard[wallet] = *stats
	l.checkCorporateEspionageAchievementLocked(wallet)
}

// applyDistrictScanner activates the 10-minute scan window for a player.
// PILLAR 3: Criminality & Intelligence.
func (l *Lobby) applyDistrictScanner(clientID string, stats *PlayerStats) {
	wallet, ok := l.wallets[clientID]
	if !ok {
		return
	}

	// Set the expiration for 10 minutes
	stats.DistrictScannerExpiresAt = time.Now().Add(10 * time.Minute)

	// Collect current trap data for an initial chat report
	var reports []string
	for _, club := range l.clubs {
		traps := 0
		for key := range club.ActiveBuffs {
			if strings.HasPrefix(key, "TRAP_") {
				traps++
			}
		}
		if traps > 0 {
			reports = append(reports, fmt.Sprintf("%s: %d traps active", escapeHTML(club.Name), traps))
		}
	}

	reportMsg := "📡 <b>INITIAL SCAN REPORT:</b><br/>" + strings.Join(reports, "<br/>")
	if len(reports) == 0 {
		reportMsg = "📡 <b>SCAN COMPLETE:</b> No active hardware traps detected in the sector."
	}

	l.sendToClientLocked(clientID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"%s"}`, reportMsg))})
	l.logAdminAuditLocked("DISTRICT_SCAN_USED", wallet, "Sector-wide scan active")
}
