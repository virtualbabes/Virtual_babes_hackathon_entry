//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"html/template"
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
		if data.ItemID == "staff_training" {
			if playerStats.JobRole != "Manager" && playerStats.JobRole != "CEO" {
				return "", fmt.Errorf("manager role required to activate staff training")
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

			targetClub.ActiveBuffs["STAFF_TRAINING"] = data.ItemID
			targetClub.BuffExpirations["STAFF_TRAINING"] = time.Now().Add(24 * time.Hour)
			targetClub.LastActivity = time.Now()
			targetClub.Mojo += item.MojoBonus

			telemetryContext := "global"
			if len(targetClub.Territories) > 0 {
				telemetryContext = targetClub.Territories[0]
			}
			l.logAdminAuditLocked("MOJO_GAIN", telemetryContext, fmt.Sprintf("Item: %s, Mojo: +%d (Club: %s)", item.Name, item.MojoBonus, targetClub.ID))

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
			l.achievementService.CheckMojoSurgeAchievementLocked(l, targetClub.ID)
			return fmt.Sprintf("🧬 <b>STAFF TRAINING ACTIVE:</b> Mutation stability for %s increased by +5%% for 24 hours.", targetClub.Name), nil
		}

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

			telemetryContext := "global"
			if len(targetClub.Territories) > 0 {
				telemetryContext = targetClub.Territories[0]
			}
			l.logAdminAuditLocked("MOJO_GAIN", telemetryContext, fmt.Sprintf("Item: %s, Mojo: +%d (Club: %s)", item.Name, item.MojoBonus, targetClub.ID))

			// Ripple Effect: members Standing reflects the new Mojo immediately
			for w, s := range l.leaderboard {
				if s.EmployerClubID == targetClub.ID {
					s.Reputation = l.CalculateReputation(s)
					l.leaderboard[w] = s
				}
			}
			l.clubs[playerStats.EmployerClubID] = targetClub
			l.updatePlayerPlaystyleTendenciesLocked(wallet, false, [2]int{}, []int{}, false, false)
			l.achievementService.CheckMojoSurgeAchievementLocked(l, targetClub.ID)
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
		duration := 24 * time.Hour
		if playerStats.JobRole == "Arc-Net Operative" && (data.ItemID == "cyber_lock" || data.ItemID == "cyber_counter") {
			duration = 36 * time.Hour // PILLAR 3: 50% Duration Bonus
		}
		targetClub.BuffExpirations[trapID] = time.Now().Add(duration)
		targetClub.LastActivity = time.Now() // Mark club as active

		// PILLAR 1: Infrastructure Prestige.
		// Deployment of hardware traps rewards the club with Mojo, increasing its social standing and unlocking premium tiers.
		targetClub.Mojo += item.MojoBonus

		telemetryContext := "global"
		if len(targetClub.Territories) > 0 {
			telemetryContext = targetClub.Territories[0]
		}
		l.logAdminAuditLocked("MOJO_GAIN", telemetryContext, fmt.Sprintf("Item: %s, Mojo: +%d (Club: %s)", item.Name, item.MojoBonus, targetClub.ID))
		l.achievementService.CheckMojoSurgeAchievementLocked(l, targetClub.ID)

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
		case "deep_scan_decryptor":
			// PILLAR 3: Justice Layer - Intelligence.
			if playerStats.JobRole != "Intel-Agent" {
				return "", fmt.Errorf("Deep-Scan Decryptor restricted to Intel-Agents")
			}
			if data.TargetWallet == "" {
				return "", fmt.Errorf("Deep-Scan requires a target wallet address")
			}
			tWallet := strings.ToLower(data.TargetWallet)
			tStats, exists := l.leaderboard[tWallet]
			if !exists {
				return "", fmt.Errorf("target signature not found in sector")
			}

			vBal := float64(l.playerBalances[tWallet]) / 1000000.0
			targetName := l.oracleService.ResolveEnvoiName(l, tWallet)
			activeContract := tStats.ActiveUnderworldContractID
			if activeContract == "" {
				activeContract = "None"
			}

			// PILLAR 1: Infrastructure Prestige. Gain Mojo for the organization.
			if playerStats.EmployerClubID != "" {
				if targetClub, ok := l.clubs[playerStats.EmployerClubID]; ok {
					targetClub.Mojo += item.MojoBonus
					targetClub.LastActivity = time.Now()
					l.achievementService.CheckMojoSurgeAchievementLocked(l, targetClub.ID)
				}
			}

			l.logAdminAuditLocked("DEEP_SCAN_USED", wallet, fmt.Sprintf("Target: %s (Wanted: %d, Bal: %.2f)", tWallet, tStats.WantedLevel, vBal))
			notificationText = fmt.Sprintf("⚖️ <b>DEEP-SCAN: %s</b><br/>Wanted Level: %d<br/>Rewards: %.2f $VBV<br/>Active Contract: %s",
				template.HTMLEscapeString(targetName), tStats.WantedLevel, vBal, activeContract)
		case "raid_jammer":
			if playerStats.JobRole != "Hostage Host" && playerStats.JobRole != "Kidnapper" {
				return "", fmt.Errorf("Raid Jammer restricted to Underworld safehouse personnel")
			}
			targetClub, clubExists := l.clubs[playerStats.EmployerClubID]
			if !clubExists {
				return "", fmt.Errorf("club records missing")
			}

			if targetClub.ActiveBuffs == nil {
				targetClub.ActiveBuffs = make(map[string]string)
			}
			if targetClub.BuffExpirations == nil {
				targetClub.BuffExpirations = make(map[string]time.Time)
			}

			targetClub.ActiveBuffs["RAID_JAMMER"] = data.ItemID
			targetClub.BuffExpirations["RAID_JAMMER"] = time.Now().Add(12 * time.Hour)
			targetClub.LastActivity = time.Now()
			targetClub.Mojo += item.MojoBonus

			l.logAdminAuditLocked("RAID_JAMMER_DEPLOYED", wallet, fmt.Sprintf("Club: %s", targetClub.ID))
			l.achievementService.CheckMojoSurgeAchievementLocked(l, targetClub.ID)
			notificationText = "📡 <b>RAID JAMMER ONLINE:</b> AOS tactical teams will find it 15% harder to breach your safehouse for 12 hours."
		case "signal_dampener":
			if playerStats.JobRole != "Hostage Host" {
				return "", fmt.Errorf("Signal Dampener restricted to Hostage Hosts")
			}
			targetClub, clubExists := l.clubs[playerStats.EmployerClubID]
			if !clubExists {
				return "", fmt.Errorf("club records missing")
			}

			if targetClub.ActiveBuffs == nil {
				targetClub.ActiveBuffs = make(map[string]string)
			}
			if targetClub.BuffExpirations == nil {
				targetClub.BuffExpirations = make(map[string]time.Time)
			}

			targetClub.ActiveBuffs["SIGNAL_DAMPENER"] = data.ItemID
			targetClub.BuffExpirations["SIGNAL_DAMPENER"] = time.Now().Add(24 * time.Hour)
			targetClub.LastActivity = time.Now()
			targetClub.Mojo += item.MojoBonus

			l.logAdminAuditLocked("SIGNAL_DAMPENER_ACTIVE", wallet, fmt.Sprintf("Club: %s", targetClub.ID))
			l.achievementService.CheckMojoSurgeAchievementLocked(l, targetClub.ID)
			notificationText = "📡 <b>SIGNAL DAMPENER ACTIVE:</b> Organization signatures are now scrambled on the Bounty Board for 24 hours."
		case "legal_pardon":
			// PILLAR 3: Justice Layer - Judicial Authority.
			if playerStats.JobRole != "Judge" {
				return "", fmt.Errorf("Legal Pardon restricted to Judges")
			}
			if data.TargetWallet == "" {
				return "", fmt.Errorf("Legal Pardon requires a target wallet address")
			}
			tWallet := strings.ToLower(data.TargetWallet)
			err := l.courthouseService.ApplyLegalPardonLocked(l, wallet, tWallet, item)
			if err != nil {
				return "", err
			}
			targetName := l.oracleService.ResolveEnvoiName(l, tWallet)
			notificationText = fmt.Sprintf("⚖️ <b>LEGAL PARDON:</b> %s infamy has been reduced by 50%% by order of the Judge.", template.HTMLEscapeString(targetName))
		case "tax_haven_license":
			// PILLAR 1: Political Influence.
			if playerStats.JobRole != "Manager" && playerStats.JobRole != "CEO" {
				return "", fmt.Errorf("Tax Haven License restricted to organizational management")
			}
			targetClub, clubExists := l.clubs[playerStats.EmployerClubID]
			if !clubExists {
				return "", fmt.Errorf("club records missing")
			}

			if targetClub.Treasury < 5000.0 {
				return "", fmt.Errorf("Treasury insufficient (5,000 $VBV required for Tax Haven status)")
			}

			targetClub.TaxHavenExpiresAt = time.Now().Add(48 * time.Hour)
			targetClub.Mojo += item.MojoBonus
			targetClub.LastActivity = time.Now()
			l.achievementService.CheckMojoSurgeAchievementLocked(l, targetClub.ID)

			l.logAdminAuditLocked("TAX_HAVEN_ACTIVATED", wallet, fmt.Sprintf("Club: %s", targetClub.ID))
			notificationText = fmt.Sprintf("🏦 <b>TAX HAVEN ACTIVE:</b> Members of %s are exempt from Exchange Fees for 48 hours.", escapeHTML(targetClub.Name))
		case "audit_shield":
			// PILLAR 1: Political Influence.
			if playerStats.JobRole != "Manager" && playerStats.JobRole != "CEO" {
				return "", fmt.Errorf("Audit Shield restricted to organizational management")
			}
			targetClub, clubExists := l.clubs[playerStats.EmployerClubID]
			if !clubExists {
				return "", fmt.Errorf("club records missing")
			}

			if targetClub.ActiveBuffs == nil {
				targetClub.ActiveBuffs = make(map[string]string)
			}
			if targetClub.BuffExpirations == nil {
				targetClub.BuffExpirations = make(map[string]time.Time)
			}

			targetClub.ActiveBuffs["AUDIT_SHIELD"] = data.ItemID
			targetClub.BuffExpirations["AUDIT_SHIELD"] = time.Now().Add(24 * time.Hour)
			targetClub.Mojo += item.MojoBonus
			targetClub.LastActivity = time.Now()
			l.achievementService.CheckMojoSurgeAchievementLocked(l, targetClub.ID)

			l.logAdminAuditLocked("AUDIT_SHIELD_ACTIVATED", wallet, fmt.Sprintf("Club: %s", targetClub.ID))
			notificationText = fmt.Sprintf("🛡️ <b>AUDIT SHIELD ONLINE:</b> %s is now protected from Justice flagging for 24 hours.", escapeHTML(targetClub.Name))
		case "market_freeze":
			// PILLAR 3: Justice Layer - Economic Enforcement.
			if playerStats.JobRole != "Tax Auditor" {
				return "", fmt.Errorf("Market Freeze restricted to Tax Auditors")
			}
			if data.TargetWallet == "" {
				return "", fmt.Errorf("Market Freeze requires a target wallet address")
			}
			tWallet := strings.ToLower(data.TargetWallet)
			tStats, exists := l.leaderboard[tWallet]
			if !exists {
				return "", fmt.Errorf("target signature not found in sector")
			}

			if tStats.WantedLevel < 20 {
				return "", fmt.Errorf("target infamy too low (Wanted Level 20+ required)")
			}

			tStats.MarketFrozenUntil = time.Now().Add(24 * time.Hour)
			l.leaderboard[tWallet] = tStats

			if playerStats.EmployerClubID != "" {
				if targetClub, ok := l.clubs[playerStats.EmployerClubID]; ok {
					targetClub.Mojo += item.MojoBonus
					l.achievementService.CheckMojoSurgeAchievementLocked(l, targetClub.ID)
				}
			}

			l.logAdminAuditLocked("MARKET_FREEZE_USED", wallet, fmt.Sprintf("Target: %s (Wanted: %d)", tWallet, tStats.WantedLevel))
			notificationText = fmt.Sprintf("⚖️ <b>MARKET FREEZE:</b> Share trading for %s has been suspended for 24 hours.",
				template.HTMLEscapeString(l.oracleService.ResolveEnvoiName(l, tWallet)))
		case "truth_serum":
			// PILLAR 3: Justice Layer - Information Disclosure.
			// Revealed all active item buffs and debuffs on an opponent's cards.
			if !inMatch {
				return "", fmt.Errorf("Truth Serum requires an active match to analyze an opponent")
			}
			oppID := match.P1ID
			if env.FromID == match.P1ID {
				oppID = match.P2ID
			}
			oppWallet, ok := l.wallets[oppID]
			if !ok {
				return "", fmt.Errorf("opponent identification failed")
			}
			buffs := match.ActiveItemBuffs[oppID]
			var report []string
			if len(buffs) == 0 {
				report = append(report, "No active combat modifiers detected.")
			} else {
				for bID, val := range buffs {
					report = append(report, fmt.Sprintf("- %s (%d matches remaining)", strings.ReplaceAll(bID, "_", " "), val))
				}
			}
			oppName := l.ResolveEnvoiName(oppWallet)
			notificationText = fmt.Sprintf("⚖️ <b>TRUTH SERUM: %s</b><br/>%s", template.HTMLEscapeString(oppName), strings.Join(report, "<br/>"))
		case "arc_net_spy":
			// PILLAR 3: Underworld Layer - Espionage.
			// Discloses the full inventory of a target player.
			if data.TargetWallet == "" {
				return "", fmt.Errorf("Arc-Net-Spy requires a target wallet address")
			}
			tWallet := strings.ToLower(data.TargetWallet)
			targetStats, exists := l.leaderboard[tWallet]
			if !exists {
				return "", fmt.Errorf("target profile not found in sector databases")
			}
			var items []string
			if len(targetStats.Inventory) == 0 {
				items = append(items, "No items in inventory.")
			} else {
				for itemID, qty := range targetStats.Inventory {
					items = append(items, fmt.Sprintf("- %s (x%d)", strings.ReplaceAll(itemID, "_", " "), qty))
				}
			}
			targetName := l.ResolveEnvoiName(tWallet)
			notificationText = fmt.Sprintf("💀 <b>ARC-NET SPY: %s</b><br/>%s", template.HTMLEscapeString(targetName), strings.Join(items, "<br/>"))
		case "district_scanner":
			l.applyDistrictScanner(env.FromID, playerStats)
			notificationText = "📡 <b>DISTRICT SCANNER ACTIVE:</b> Hardware traps are now visible on the world map for 10 minutes."
		case "cloak_disruptor":
			// PILLAR 3: Criminality & Intelligence.
			// This item allows a Bounty Hunter to temporarily reveal an outlaw's signal.
			if playerStats.WantedLevel > 2 {
				return "", fmt.Errorf("this item is restricted to clean players (Bounty Hunters)")
			}

			// PILLAR 3: Anti-Spam Cooldown.
			// Verify if the hunter is still on cooldown from a previous disruption attempt.
			if !playerStats.DisruptorCooldownAt.IsZero() && time.Now().Before(playerStats.DisruptorCooldownAt) {
				remaining := time.Until(playerStats.DisruptorCooldownAt).Seconds()
				return "", fmt.Errorf("Cloak Disruptor recalibrating. Ready in %.0f seconds", remaining)
			}

			if data.TargetWallet == "" {
				return "", fmt.Errorf("cloak disruptor requires a target wallet")
			}
			targetWallet := strings.ToLower(data.TargetWallet)

			// PILLAR 1: Infrastructure Prestige.
			// Cloak disruption rewards the auditor's organization with Mojo.
			if playerStats.EmployerClubID != "" {
				if targetClub, exists := l.clubs[playerStats.EmployerClubID]; exists {
					targetClub.Mojo += item.MojoBonus
					targetClub.LastActivity = time.Now()

					telemetryContext := "global"
					if len(targetClub.Territories) > 0 {
						telemetryContext = targetClub.Territories[0]
					}
					l.logAdminAuditLocked("MOJO_GAIN", telemetryContext, fmt.Sprintf("Item: %s, Mojo: +%d (Club: %s)", item.Name, item.MojoBonus, targetClub.ID))
					l.achievementService.CheckMojoSurgeAchievementLocked(l, targetClub.ID)

					for w, s := range l.leaderboard {
						if s.EmployerClubID == targetClub.ID {
							s.Reputation = l.CalculateReputation(s)
							l.leaderboard[w] = s
						}
					}
					*playerStats = l.leaderboard[wallet]
					l.clubs[playerStats.EmployerClubID] = targetClub
				}
			}
			targetStats, exists := l.leaderboard[targetWallet]
			if !exists {
				return "", fmt.Errorf("target records not found in sector")
			}

			// Verify if target is currently ghosted
			if time.Now().After(targetStats.GhostProtocolExpiresAt) {
				return "", fmt.Errorf("target signal is already visible on the Bounty Board")
			}

			// Set 60-second cooldown for the hunter (attacker)
			playerStats.DisruptorCooldownAt = time.Now().Add(60 * time.Second)

			targetStats.CloakDisruptedUntil = time.Now().Add(5 * time.Minute)
			l.leaderboard[targetWallet] = targetStats
			l.logAdminAuditLocked("CLOAK_DISRUPTED", "sector_all", fmt.Sprintf("Hunter: %s, Target: %s", wallet, targetWallet))
			notificationText = fmt.Sprintf("📡 <b>CLOAK DISRUPTED:</b> %s is now visible on the Bounty Board for 5 minutes.", l.ResolveEnvoiName(targetWallet))

		case "forensic_audit_kit":
			// PILLAR 3: Justice Layer - Forensic Auditing.
			if playerStats.JobRole != "Forensic Analyst" {
				return "", fmt.Errorf("this item is restricted to Forensic Analysts")
			}
			if data.TargetCardID == 0 {
				return "", fmt.Errorf("forensic Audit Kit requires a target card ID")
			}
			targetCard, cardExists := l.inventory[data.TargetCardID]
			if !cardExists {
				return "", fmt.Errorf("target card not found in inventory")
			}

			// PILLAR 1: Infrastructure Prestige (Club Mojo).
			if playerStats.EmployerClubID != "" {
				if targetClub, exists := l.clubs[playerStats.EmployerClubID]; exists {
					targetClub.Mojo += item.MojoBonus
					targetClub.LastActivity = time.Now()
					telemetryContext := "global"
					if len(targetClub.Territories) > 0 {
						telemetryContext = targetClub.Territories[0]
					}
					l.logAdminAuditLocked("MOJO_GAIN", telemetryContext, fmt.Sprintf("Item: %s, Mojo: +%d (Club: %s)", item.Name, item.MojoBonus, targetClub.ID))
					l.achievementService.CheckMojoSurgeAchievementLocked(l, targetClub.ID)
					for w, s := range l.leaderboard {
						if s.EmployerClubID == targetClub.ID {
							s.Reputation = l.CalculateReputation(s)
							l.leaderboard[w] = s
						}
					}
					*playerStats = l.leaderboard[wallet]
					l.clubs[playerStats.EmployerClubID] = targetClub
				}
			}

			// Apply Artifact reduction (Permanent Battle Scars)
			l.applyMutationScarsLocked(data.TargetCardID, 50)
			notificationText = fmt.Sprintf("⚖️ <b>FORENSIC AUDIT:</b> Card #%d's forensic grade reduced. Artifact: %d.", data.TargetCardID, targetCard.Artifact-50)

			// PILLAR 3: Mission Completion Logic.
			var ownerWallet string
			for w, s := range l.leaderboard {
				if _, has := s.Inventory[fmt.Sprintf("CARD-%d", data.TargetCardID)]; has {
					ownerWallet = w
					break
				}
			}

			if ownerWallet != "" {
				ownerStats := l.leaderboard[ownerWallet]
				isFGrade := (targetCard.Artifact - 50) <= -100

				// Self-Audit Loophole Prevention
				if isFGrade && strings.EqualFold(ownerWallet, wallet) {
					if playerStats.ActiveJusticeMissionID == "MISSION-007" || playerStats.ActiveJusticeMissionID == "MISSION-008" {
						l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Mission Failed: Cannot audit your own assets for this objective."}`)})
						return notificationText, fmt.Errorf("cannot audit own assets for high-tier missions")
					}
				}

				if isFGrade {
					// MISSION-008: Regional Governor's favorite card.
					if playerStats.ActiveJusticeMissionID == "MISSION-008" {
						isGov := false
						if club, exists := l.clubs[ownerStats.EmployerClubID]; exists && strings.EqualFold(club.OwnerWallet, ownerWallet) {
							if l.clubService.IsClubRegionalLocked(l, club) {
								isGov = true
							}
						}
						if isGov && ownerStats.FavoriteCardID == data.TargetCardID {
							const rewardMicro = 3000 * 1000000
							l.playerBalances[wallet] += rewardMicro
							playerStats.ActiveJusticeMissionID = ""
							l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", wallet, "ID: MISSION-008, Payout: 3000.00")
							l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> Governor's high-value asset devalued. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
							l.applyDynamicScalingLocked()
						}
					}
					// MISSION-007: High-infamy outlaw (Wanted 15+).
					if playerStats.ActiveJusticeMissionID == "MISSION-007" && ownerStats.WantedLevel >= 15 {
						const rewardMicro = 2500 * 1000000
						l.playerBalances[wallet] += rewardMicro
						playerStats.ActiveJusticeMissionID = ""
						l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", wallet, "ID: MISSION-007, Payout: 2500.00")
						l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> Outlaw asset compromised. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
						l.applyDynamicScalingLocked()
					}
					// MISSION-005: Forensic Audit successful.
					if playerStats.ActiveJusticeMissionID == "MISSION-005" {
						const rewardMicro = 2000 * 1000000
						l.playerBalances[wallet] += rewardMicro
						playerStats.ActiveJusticeMissionID = ""
						l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", wallet, "ID: MISSION-005, Payout: 2000.00")
						l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> Forensic Audit successful. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
						l.applyDynamicScalingLocked()
					}
				}
			}

		case "cyber_jammer":
			playerStats.HasCyberJammer = true
			notificationText = "📡 <b>CYBER-JAMMER ACTIVE:</b> Your next sabotage attempt will be silent."
		case "security_override":
			// PILLAR 7: Asset Redemption.
			// Bribe underworld security to force the return of a card from the Black Market.
			if data.TargetCardID == 0 {
				return "", fmt.Errorf("Security Override requires a target card ID")
			}

			// 1. Locate the card in the Black Market (liquidated loans)
			foundIdx := -1
			for i, loan := range l.blackMarket {
				if loan.CollateralBundle.CardID == data.TargetCardID {
					foundIdx = i
					break
				}
			}

			if foundIdx == -1 {
				return "", fmt.Errorf("Target BABE #%d not found in Black Market datastreams", data.TargetCardID)
			}

			// 2. Remove from Black Market and restore to user
			loan := l.blackMarket[foundIdx]
			l.blackMarket = append(l.blackMarket[:foundIdx], l.blackMarket[foundIdx+1:]...)

			cardKey := fmt.Sprintf("CARD-%d", data.TargetCardID)
			playerStats.Inventory[cardKey]++

			l.logAdminAuditLocked("SECURITY_OVERRIDE", wallet, fmt.Sprintf("Bribed return of Card #%d from original owner %s", data.TargetCardID, loan.BorrowerWallet))

			// PILLAR 2: Structural Audit. Override fees are already captured via item purchase.
			// We simply update the player's reputation to reflect the reclamation.
			playerStats.Reputation = l.CalculateReputation(*playerStats)
			notificationText = fmt.Sprintf("🔓 <b>SECURITY OVERRIDE:</b> Underworld ledgers corrupted. BABE #%d restored to inventory.", data.TargetCardID)
		case "mutation_insurance":
			playerStats.HasMutationInsurance = true
			notificationText = "🛡️ <b>MUTATION INSURANCE ACTIVE:</b> Your next gene-editing procedure is guaranteed to succeed."
		}

	case "Underworld_Admin":
		switch data.ItemID {
		case "illicit_commission_permit":
			// PILLAR 3: Lawyer-Commissioner Influence.
			if playerStats.JobRole != "Lawyer-Commissioner" {
				return "", fmt.Errorf("Illicit Commission Permit restricted to Lawyer-Commissioners")
			}
			if data.TargetClubID == "" {
				return "", fmt.Errorf("Illicit Commission Permit requires a target club ID")
			}
			targetClub, clubExists := l.clubs[data.TargetClubID]
			if !clubExists {
				return "", fmt.Errorf("target club records missing")
			}

			// PILLAR 1: Illicit Alignment. Only Tactical clubs can receive this buff.
			if targetClub.Type != "Tactical" {
				return "", fmt.Errorf("Illicit Commission Permit can only be applied to Tactical organizations")
			}

			if targetClub.BuffExpirations == nil {
				targetClub.BuffExpirations = make(map[string]time.Time)
			}

			// Apply the buff: The actual commission rate override will be handled in DistributeShopRevenueLocked.
			targetClub.BuffExpirations["ILLICIT_COMMISSION"] = time.Now().Add(24 * time.Hour)
			targetClub.LastActivity = time.Now()
			targetClub.Mojo += item.MojoBonus

			l.logAdminAuditLocked("ILLICIT_COMMISSION_APPLIED", wallet, fmt.Sprintf("Club: %s, Duration: 24h", targetClub.ID))
			l.achievementService.CheckMojoSurgeAchievementLocked(l, targetClub.ID)

			// Route the item's cost to the Faucet (100% Faucet share)
			if l.tokenSinkRouter != nil {
				matrix := RevenueSplitMatrix{FaucetShare: 1.0, ClubShare: 0.0, GovernanceShare: 0.0}
				_ = l.tokenSinkRouter.RouteCriminalTax("ILLICIT_LICENSE_FEE", uint64(item.Price*1000000), matrix, 0, "")
			}
			notificationText = fmt.Sprintf("💀 <b>ILLICIT LICENSE:</b> Commission rate for %s set to 1%% for 24 hours.", template.HTMLEscapeString(targetClub.Name))

		case "illicit_commission_permit":
			// PILLAR 3: Lawyer-Commissioner Influence.
			if playerStats.JobRole != "Lawyer-Commissioner" {
				return "", fmt.Errorf("Illicit Commission Permit restricted to Lawyer-Commissioners")
			}
			if data.TargetClubID == "" {
				return "", fmt.Errorf("Illicit Commission Permit requires a target club ID")
			}
			targetClub, clubExists := l.clubs[data.TargetClubID]
			if !clubExists {
				return "", fmt.Errorf("target club records missing")
			}

			// PILLAR 1: Illicit Alignment. Only Tactical clubs can receive this buff.
			if targetClub.Type != "Tactical" {
				return "", fmt.Errorf("Illicit Commission Permit can only be applied to Tactical organizations")
			}

			if targetClub.BuffExpirations == nil {
				targetClub.BuffExpirations = make(map[string]time.Time)
			}

			// Apply the buff: The actual commission rate override will be handled in DistributeShopRevenueLocked.
			targetClub.BuffExpirations["ILLICIT_COMMISSION"] = time.Now().Add(24 * time.Hour)
			targetClub.LastActivity = time.Now()
			targetClub.Mojo += item.MojoBonus

			l.logAdminAuditLocked("ILLICIT_COMMISSION_APPLIED", wallet, fmt.Sprintf("Club: %s, Duration: 24h", targetClub.ID))
			l.achievementService.CheckMojoSurgeAchievementLocked(l, targetClub.ID)

			// Route the item's cost to the Faucet (100% Faucet share)
			if l.tokenSinkRouter != nil {
				matrix := RevenueSplitMatrix{FaucetShare: 1.0, ClubShare: 0.0, GovernanceShare: 0.0}
				_ = l.tokenSinkRouter.RouteCriminalTax("ILLICIT_LICENSE_FEE", uint64(item.Price*1000000), matrix, 0, "")
			}
			notificationText = fmt.Sprintf("💀 <b>ILLICIT LICENSE:</b> Commission rate for %s set to 1%% for 24 hours.", template.HTMLEscapeString(targetClub.Name))

		case "regulatory_bypass_permit":
			// PILLAR 3: Lawyer-Commissioner Influence.
			if playerStats.JobRole != "Lawyer-Commissioner" {
				return "", fmt.Errorf("Regulatory Bypass Permit restricted to Lawyer-Commissioners")
			}
			if data.TargetClubID == "" {
				return "", fmt.Errorf("Regulatory Bypass Permit requires a target club ID")
			}
			targetClub, clubExists := l.clubs[data.TargetClubID]
			if !clubExists {
				return "", fmt.Errorf("target club records missing")
			}

			if targetClub.ActiveBuffs == nil {
				targetClub.ActiveBuffs = make(map[string]string)
			}
			if targetClub.BuffExpirations == nil {
				targetClub.BuffExpirations = make(map[string]time.Time)
			}

			targetClub.ActiveBuffs["REGULATORY_BYPASS"] = data.ItemID
			targetClub.BuffExpirations["REGULATORY_BYPASS"] = time.Now().Add(24 * time.Hour)
			targetClub.LastActivity = time.Now()
			targetClub.Mojo += item.MojoBonus

			l.logAdminAuditLocked("REGULATORY_BYPASS_ACTIVATED", wallet, fmt.Sprintf("Club: %s", targetClub.ID))
			l.achievementService.CheckMojoSurgeAchievementLocked(l, targetClub.ID)

			// Route the item's cost to the Faucet (100% Faucet share)
			if l.tokenSinkRouter != nil {
				matrix := RevenueSplitMatrix{FaucetShare: 1.0, ClubShare: 0.0, GovernanceShare: 0.0}
				_ = l.tokenSinkRouter.RouteCriminalTax("REGULATORY_BYPASS_FEE", uint64(item.Price*1000000), matrix, 0, "")
			}
			notificationText = fmt.Sprintf("⚖️ <b>REGULATORY BYPASS:</b> Corporate Tax for %s reduced by 50%% for 24 hours.", template.HTMLEscapeString(targetClub.Name))
		}

	case "Justice_Admin":
		switch data.ItemID {
		case "pro_social_commission_license":
			// PILLAR 3: Justice Commissioner Influence.
			if playerStats.JobRole != "Justice Commissioner" {
				return "", fmt.Errorf("Pro-Social Commission License restricted to Justice Commissioners")
			}
			if data.TargetClubID == "" {
				return "", fmt.Errorf("Pro-Social Commission License requires a target club ID")
			}
			targetClub, clubExists := l.clubs[data.TargetClubID]
			if !clubExists {
				return "", fmt.Errorf("target club records missing")
			}

			// PILLAR 1: Pro-Social Alignment. Only Vitality or Elemental clubs can receive this buff.
			if targetClub.Type != "Vitality" && targetClub.Type != "Elemental" {
				return "", fmt.Errorf("Pro-Social Commission License can only be applied to Vitality or Elemental clubs")
			}

			if targetClub.BuffExpirations == nil {
				targetClub.BuffExpirations = make(map[string]time.Time)
			}

			// Apply the buff: The actual commission rate override will be handled in DistributeShopRevenueLocked.
			targetClub.BuffExpirations["PRO_SOCIAL_COMMISSION"] = time.Now().Add(24 * time.Hour)
			targetClub.LastActivity = time.Now()
			targetClub.Mojo += item.MojoBonus

			l.logAdminAuditLocked("PRO_SOCIAL_COMMISSION_APPLIED", wallet, fmt.Sprintf("Club: %s, Duration: 24h", targetClub.ID))
			l.achievementService.CheckMojoSurgeAchievementLocked(l, targetClub.ID)

			// Route the item's cost to the Faucet (100% Faucet share)
			if l.tokenSinkRouter != nil {
				matrix := RevenueSplitMatrix{FaucetShare: 1.0, ClubShare: 0.0, GovernanceShare: 0.0}
				_ = l.tokenSinkRouter.RouteCriminalTax("PRO_SOCIAL_LICENSE_FEE", uint64(item.Price*1000000), matrix, 0, "")
			}
			notificationText = fmt.Sprintf("⚖️ <b>PRO-SOCIAL LICENSE:</b> Commission rate for %s set to 1%% for 24 hours.", template.HTMLEscapeString(targetClub.Name))
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

	// PILLAR 4: Regional Telemetry. Use target territory context for audit logs.
	telemetryContext := "neutral_zone"
	if len(targetClub.Territories) > 0 {
		telemetryContext = targetClub.Territories[0]
	}

	// PILLAR 3: Intelligence Intel.
	// Provide the economic vulnerability snapshot to the infiltrator.
	msg := fmt.Sprintf("%s🔍 <b>CYBER-AUDIT REPORT: %s</b><br/>Current Treasury: %.2f $VBV<br/>Weekly Average: %.2f $VBV<br/>Security Status: %s",
		sabotagePrefix, escapeHTML(targetClub.Name), targetClub.Treasury, avg, status)

	l.sendToClientLocked(clientID, Envelope{
		Type:    "admin_notification",
		Payload: json.RawMessage(fmt.Sprintf(`{"text":"%s"}`, msg)),
	})

	l.logAdminAuditLocked("CYBER_AUDIT_USED", telemetryContext, fmt.Sprintf("Auditor: %s, Target: %s (%s)", wallet, targetClub.Name, targetClubID))

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
			l.logAdminAuditLocked("CYBER_COUNTER_TRIGGERED", telemetryContext, fmt.Sprintf("Owner: %s, Auditor: %s (%s)", targetClub.OwnerWallet, auditorName, wallet)) // Safe under Lobby lock.
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

	// PILLAR 3: Underworld Contract Completion.
	// Check if the current Cyber-Audit fulfills an active criminal mission.
	if stats.ActiveUnderworldContractID == "CONTRACT-002" {
		// Objective: Audit the 'Data Haven' club.
		if targetClubID == "data_haven" {
			const rewardMicro = 750 * 1000000
			l.playerBalances[wallet] += rewardMicro
			stats.ActiveUnderworldContractID = ""
			l.logAdminAuditLocked("CONTRACT_COMPLETED", wallet, "ID: CONTRACT-002, Payout: 750.00")
			l.sendToClientLocked(clientID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> Cyber-Audit on Data Haven successful. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
		}
	}

	// PILLAR 3: Justice Mission Completion (MISSION-003)
	// Check if the current Cyber-Audit fulfills an active Justice mission.
	if stats.ActiveJusticeMissionID == "MISSION-003" {
		// Objective: Audit the 'Elemental Forge' club.
		if targetClubID == "elemental_forge" {
			const rewardMicro = 800 * 1000000
			l.playerBalances[wallet] += rewardMicro
			stats.ActiveJusticeMissionID = ""
			l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", wallet, "ID: MISSION-003, Payout: 800.00")
			l.sendToClientLocked(clientID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> Regulatory Audit on Elemental Forge successful. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
		}
	}

	// PILLAR 3: State Sync.
	// Explicitly update the leaderboard entry so the achievement check
	// sees the latest unique audit record immediately.
	l.leaderboard[wallet] = *stats
	l.achievementService.CheckCorporateEspionageAchievementLocked(l, wallet)
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
	l.logAdminAuditLocked("DISTRICT_SCAN_USED", "sector_all", fmt.Sprintf("User: %s", wallet))
}
