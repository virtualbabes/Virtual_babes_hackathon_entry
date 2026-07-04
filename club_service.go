//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

// ClubService encapsulates logic for organization management, heists, and territory expansion.
// PILLAR 5: Stateless Service Design.
type ClubService struct{}

// HandleHeist processes a criminal attempt to loot a Club Treasury.
func (s *ClubService) HandleHeist(l *Lobby, env *Envelope) {
	var data struct {
		TargetClubID string `json:"target_club_id"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}
	l.mutex.Lock()
	defer l.mutex.Unlock()

	wallet, ok := l.wallets[env.FromID]
	if !ok {
		return
	}

	// PILLAR 3: Identity Hardening.
	l.ensurePlayerStatsMapsInitialized(wallet)

	playerStats := l.leaderboard[wallet]
	targetClub, exists := l.clubs[data.TargetClubID]
	if !exists {
		return
	}

	// PILLAR 1: Alliance Integration.
	// Helper to check if perpetrator is allied with the target.
	// You cannot heist your own alliance.
	if s.IsPlayerAffiliatedWithClubLocked(l, wallet, targetClub) {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ <b>HEIST BLOCKED:</b> Internal infiltration of alliance treasuries is strictly forbidden."}`)})
		return
	}

	// SUCCESS CHANCE CALCULATION: Base 50% chance + (Effective Cunning - Security Level) / 100
	successChance := 0.5

	// PILLAR 3: Career Path Influence. 'Heist Planners' in the operative's org provide a +5% buff.
	var heistPlanner string
	if playerStats.EmployerClubID != "" {
		if heisterClub, ok := l.clubs[playerStats.EmployerClubID]; ok {
			for w, r := range heisterClub.Staff {
				if strings.EqualFold(r, "Heist Planner") {
					heistPlanner = w
					successChance += 0.05
					break
				}
			}
		}
	}

	// PILLAR 3: Sabotage Check.
	// If the club is under a state of Sabotage, hardware-based traps are ignored.
	sabotaged := false
	now := time.Now()
	if expiry, exists := targetClub.BuffExpirations["SABOTAGE"]; exists {
		if now.Before(expiry) {
			sabotaged = true
		} else {
			delete(targetClub.BuffExpirations, "SABOTAGE")
		}
	}

	// PILLAR 3: Intelligence Enforcement.
	// A Cyber-Lock prevents heist initiation by encrypting the treasury coordinates.
	// This block is only bypassed if the club is currently sabotaged.
	if !sabotaged {
		for trapID, itemID := range targetClub.ActiveBuffs {
			if itemID == "cyber_lock" {
				if expiry, exists := targetClub.BuffExpirations[trapID]; exists && now.Before(expiry) {
					l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"❌ <b>HEIST BLOCKED:</b> %s treasury is protected by a Cyber-Lock. Deploy a Sabotage protocol to proceed."}`, escapeHTML(targetClub.Name)))})
					return
				}
			}
		}
	}

	// PILLAR 1: Infrastructure Security - Territory Invasion Alert
	// Trigger high-priority warning if the treasury is highly capitalized (>= 5,000 $VBV).
	// This alerts members and triggers the 'Warning_long.mp3' alarm on the client side.
	if targetClub.Treasury >= 5000.0 {
		invasionAlert := "🚨 <b>TERRITORY INVASION:</b> A rival cell is striking your high-value assets!"
		warningPayload, _ := json.Marshal(map[string]string{
			"text": invasionAlert,
			"type": "critical",
		})
		warningEnv := Envelope{Type: "admin_notification", FromID: "SERVER", Payload: warningPayload}
		warningMsg, _ := json.Marshal(warningEnv)

		for cid, client := range l.clients {
			cWallet, ok := l.wallets[cid]
			if !ok {
				continue
			}
			stats := l.leaderboard[strings.ToLower(cWallet)]
			if stats.EmployerClubID == targetClub.ID || strings.EqualFold(cWallet, targetClub.OwnerWallet) {
				select {
				case client.send <- warningMsg:
				default:
				}
			}
		}
		l.logAdminAuditLocked("TERRITORY_INVASION", targetClub.OwnerWallet, fmt.Sprintf("High-value heist detected on club %s", targetClub.Name))
	}

	// PILLAR 1: Regional Security Integration.
	isRegion := s.IsClubRegionalLocked(l, targetClub)
	securityLevel := float64(targetClub.Mojo) / 10.0
	if isRegion {
		// PILLAR 1: Regional Governor Security Scaling.
		// The administrative bonus (+15) is bolstered by the organizational footprint.
		// Every active staff member adds 2.0 to the administrative oversight layer.
		securityLevel += 15.0 + (float64(len(targetClub.Staff)) * 2.0)
	}

	for _, role := range targetClub.Staff {
		// PILLAR 1: Hardened Role Validation. Ensure case-insensitive security bonuses.
		if strings.EqualFold(role, "Security") {
			securityLevel += 15.0
		}
	}

	// Apply Attribute Modifier: Players compete against the club's total security profile
	successChance += (float64(l.playerService.GetEffectiveCunning(playerStats)) - securityLevel) / 100.0

	// Apply Trap Penalties from the Club's active buffs with lazy pruning
	for trapID, itemID := range targetClub.ActiveBuffs {
		// Check for trap expiration before applying modifiers
		if expiry, exists := targetClub.BuffExpirations[trapID]; exists && now.After(expiry) {
			delete(targetClub.ActiveBuffs, trapID)
			delete(targetClub.BuffExpirations, trapID)
			log.Printf("[INDUSTRIAL] Defense trap %s expired for club %s\n", trapID, targetClub.Name)
			continue
		}

		if item, ok := GlobalShopRegistry[itemID]; ok && item.HeistSuccessModifier != 0 {
			modifier := item.HeistSuccessModifier

			// PILLAR 3: Sabotage Bypass.
			// Hardware traps are disabled if the club has been successfully sabotaged.
			if sabotaged && item.ClubType == "Hardware" {
				log.Printf("[CRIMINALITY] Hardware trap %s bypassed due to active sabotage on %s\n", itemID, targetClub.Name)
				continue
			}

			// PILLAR 1: Regional Governor Synergy.
			// Defensive hardware is more effective when integrated into a district-wide regional network.
			if isRegion {
				modifier *= 1.5
			}
			// Master Tier items provide an additional scaling bonus to their specialized effects.
			if item.IsMasterTier {
				modifier *= 1.25
			}
			successChance += modifier
		}
	}

	if successChance < 0.05 {
		successChance = 0.05
	}
	if successChance > 0.95 {
		successChance = 0.95
	}

	roll := rand.Float64()
	var status string
	var netLoot, fenceFee float64
	canKidnap := false

	if roll < successChance {
		// SUCCESSFUL HEIST
		status = "success"

		// PILLAR 3: Career Path influence. 'Kidnappers' have double the baseline capture rate.
		kidnapChance := 0.25
		if playerStats.JobRole == "Kidnapper" {
			kidnapChance = 0.50
		}

		if l.playerService.GetEffectiveCunning(playerStats) >= 3 && rand.Float64() < kidnapChance {
			canKidnap = true
		}

		// Calculate Loot: 10% of target club's treasury, capped at 500 VBV
		maxLootMicro := uint64(500 * 1000000)
		// PILLAR 2: Precision Hardening. Convert treasury float to micro-units first (clamp dust via rounding).
		// Avoid float fractional rounding constants that can trigger uint64 conversion drift at compile-time.
		clubTreasuryMicro := uint64(targetClub.Treasury * 1000000) // floor to micro-unit
		lootMicro := (clubTreasuryMicro*10 + 50) / 100            // 10% of treasury, rounded
		if lootMicro > maxLootMicro {
			lootMicro = maxLootMicro
		}

		// INDUSTRIAL LOOP: 10% "Fence Fee" returns to the Faucet Pool
		// Integer math with rounding to nearest micro-unit
		fenceFeeMicro := (lootMicro*10 + 50) / 100
		netLootMicro := lootMicro - fenceFeeMicro
		fenceFee = float64(fenceFeeMicro) / 1000000.0

		// PILLAR 3: Heist Planner Commission.
		// Planners claim a 5% cut of the net loot for their tactical oversight.
		if heistPlanner != "" {
			plannerCutMicro := (netLootMicro * 5) / 100
			netLootMicro -= plannerCutMicro
			l.playerBalances[heistPlanner] += plannerCutMicro

			l.logAdminAuditLocked("HEIST_PLANNER_CUT", heistPlanner, fmt.Sprintf("Amount: %.2f (Source: %s)", float64(plannerCutMicro)/1000000.0, wallet))
			if cid := l.getClientIDFromWalletLocked(heistPlanner); cid != "" {
				l.sendToClientLocked(cid, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💼 <b>PLANNER DIVIDEND:</b> You earned %.2f $VBV from %s's successful heist."}`, float64(plannerCutMicro)/1000000.0, l.oracleService.ResolveEnvoiName(l, wallet)))})
			}
		}

		netLoot = float64(netLootMicro) / 1000000.0

		sectorID := "neutral_zone"
		if len(targetClub.Territories) > 0 {
			sectorID = targetClub.Territories[0]
		}

		// PILLAR 3: Intelligence Tracking. Crime leaves a trail in the sector.
		l.lastSeenDistricts[wallet] = sectorID

		// PILLAR 3: Financial Proof.
		// Record successful heist on-chain for the immutable audit trail.
		heistDetails := map[string]interface{}{
			"target_id": targetClub.ID,
			"perp":      wallet,
			"fence_fee": fenceFee,
			"net_loot":  netLoot,
			"sector_id": sectorID, // PILLAR 3: Localized Economic Auditing.
			"ts":        now.Unix(),
		}

		// INDUSTRIAL LOOP: Stolen tokens return from the Club Reserve to the general Faucet pool.
		// PILLAR 2: Ledger Integrity.
		playerStats.Playstyle.RiskTolerance += 0.05
		playerStats.HeistAttempts++

		// Execute internal reallocation
		// PILLAR 2: Authoritative Treasury Sync.
		// Ensure the organizational liability is correctly decremented in the accounting kernel.
		numericID, _ := strconv.ParseUint(strings.TrimPrefix(targetClub.ID, "CLUB-"), 10, 64)
		if l.tokenSinkRouter != nil {
			l.tokenSinkRouter.Mu.Lock()
			if node, ok := l.tokenSinkRouter.ActiveClubs[numericID]; ok {
				if node.TreasuryBalance >= lootMicro {
					node.TreasuryBalance -= lootMicro
				} else {
					node.TreasuryBalance = 0
				}
				// Keep authoritative micro-unit ledger.
				targetClub.TreasuryMicro = node.TreasuryBalance
			} else {
				// Fallback for clubs created before the router node migration:
				// Update via micro to preserve uint64 ledger integrity.
				targetClub.TreasuryMicro = uint64(targetClub.Treasury * 1000000.0) // floor to micro-unit
				if targetClub.TreasuryMicro >= lootMicro {
					targetClub.TreasuryMicro -= lootMicro
				} else {
					targetClub.TreasuryMicro = 0
				}
			}
			l.tokenSinkRouter.Mu.Unlock()
		} else {
			// Non-router: update via micro-unit.
			targetClub.TreasuryMicro = uint64(targetClub.Treasury * 1000000.0) // floor to micro-unit
			if targetClub.TreasuryMicro >= lootMicro {
				targetClub.TreasuryMicro -= lootMicro
			} else {
				targetClub.TreasuryMicro = 0
			}
		}
		targetClub.LastActivity = now // Consistent activity tracking

		// Add net loot to player's virtual rewards
		l.playerBalances[wallet] += netLootMicro

		// PILLAR 2: Industrial Loop (Token-Sink Router migration).
		// Route the 10% fence fee to the Faucet via the reconciliation kernel.
		if l.tokenSinkRouter != nil {
			matrix := RevenueSplitMatrix{FaucetShare: 1.0, ClubShare: 0.0, GovernanceShare: 0.0}
			_ = l.tokenSinkRouter.RouteCriminalTax(sectorID, fenceFeeMicro, matrix, 0, "")

			// Sync float balance with authoritative micro-unit total
			l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0
		}

		playerStats.WantedLevel += 5
		playerStats.Cunning += 1 // Successful heists improve Cunning

		// PILLAR 2: Ledger Integrity. Usable liquidity increases as club reserves decrease.
		// Recalculate scaling AFTER the liability shift is committed.
		if lootMicro > 0 {
			l.applyDynamicScalingLocked()
		}

		// Update local reputation and leaderboard before achievement to ensure accuracy
		playerStats.Reputation = l.CalculateReputation(playerStats)
		l.leaderboard[wallet] = playerStats

		// Dispatch on-chain log for financial verification
		go func(hd interface{}) {
			jsonPayload, _ := json.Marshal(hd)
			l.sendNoteTx(fmt.Sprintf("VBT_HEIST_LOG:%s", string(jsonPayload)))
		}(heistDetails)

		// Achievement unlock uses the Locked variant since we already hold the lobby mutex.
		l.achievementService.UnlockAchievementLocked(l, wallet, "FIRST_HEIST")
		playerStats = l.leaderboard[wallet] // Re-fetch to avoid clobbering achievement

		// PILLAR 3: Underworld Contract Completion (CONTRACT-009).
		// Objective: Execute a successful Heist against a club controlled by a Regional Governor.
		if playerStats.ActiveUnderworldContractID == "CONTRACT-009" {
			if s.IsClubRegionalLocked(l, targetClub) {
				const rewardMicro = 4000 * 1000000
				l.playerBalances[wallet] += rewardMicro
				playerStats.ActiveUnderworldContractID = ""
				l.logAdminAuditLocked("CONTRACT_COMPLETED", wallet, "ID: CONTRACT-009, Payout: 4000.00")
				l.sendToClientLocked(env.FromID, Envelope{
					Type:    "admin_notification",
					Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> Regional Governor treasury breached. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0)),
				})
				l.applyDynamicScalingLocked()
				l.leaderboard[wallet] = playerStats
			}
		}

		// PILLAR 3: Underworld Contract Completion (CONTRACT-012).
		// Objective: Successfully execute a Heist against a club with an active 'MOJO_STABILIZER'.
		if playerStats.ActiveUnderworldContractID == "CONTRACT-012" {
			if expiry, active := targetClub.BuffExpirations["MOJO_STABILIZER"]; active && time.Now().Before(expiry) {
				const rewardMicro = 7500 * 1000000
				l.playerBalances[wallet] += rewardMicro
				playerStats.ActiveUnderworldContractID = ""
				l.logAdminAuditLocked("CONTRACT_COMPLETED", wallet, "ID: CONTRACT-012, Payout: 7500.00")
				l.sendToClientLocked(env.FromID, Envelope{
					Type:    "admin_notification",
					Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> Mojo field neutralized. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0)),
				})
				l.applyDynamicScalingLocked()
				l.leaderboard[wallet] = playerStats
			}
		}

		// PILLAR 3: Underworld Contract Completion (CONTRACT-013).
		// Objective: Successful Heist against the Arena Center controller while in an Alliance.
		if playerStats.ActiveUnderworldContractID == "CONTRACT-013" {
			isArenaCenterOwner := false
			for _, t := range targetClub.Territories {
				if t == "arena_center" {
					isArenaCenterOwner = true
					break
				}
			}
			if isArenaCenterOwner && targetClub.AlliedClubID != "" {
				const rewardMicro = 10000 * 1000000
				l.playerBalances[wallet] += rewardMicro
				playerStats.ActiveUnderworldContractID = ""
				l.logAdminAuditLocked("CONTRACT_COMPLETED", wallet, "ID: CONTRACT-013, Payout: 10000.00")
				l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> Arena Center alliance breached. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
				l.applyDynamicScalingLocked()
				l.leaderboard[wallet] = playerStats
			}
		}

		// PILLAR 3: Underworld Contract Completion (CONTRACT-015).
		// Objective: Successful Heist against Arena Center controller with Wanted Level 25+.
		if playerStats.ActiveUnderworldContractID == "CONTRACT-015" {
			isArenaCenterOwner := false
			for _, t := range targetClub.Territories {
				if t == "arena_center" {
					isArenaCenterOwner = true
					break
				}
			}
			if isArenaCenterOwner && playerStats.WantedLevel >= 25 {
				const rewardMicro = 12000 * 1000000
				l.playerBalances[wallet] += rewardMicro
				playerStats.ActiveUnderworldContractID = ""
				l.logAdminAuditLocked("CONTRACT_COMPLETED", wallet, "ID: CONTRACT-015, Payout: 12000.00")
				l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> Arena Center hegemony breached. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
				l.applyDynamicScalingLocked()
				l.leaderboard[wallet] = playerStats
			}
		}

		// PILLAR 3: Underworld Contract Completion (CONTRACT-017).
		// Objective: Successful Heist against a Regional Governor holding 3+ hostages.
		if playerStats.ActiveUnderworldContractID == "CONTRACT-017" {
			victimWallet := strings.ToLower(targetClub.OwnerWallet)
			if victimStats, vExists := l.leaderboard[victimWallet]; vExists && s.IsClubRegionalLocked(l, targetClub) && len(victimStats.HeldHostageCards) >= 3 {
				const rewardMicro = 20000 * 1000000
				l.playerBalances[wallet] += rewardMicro
				playerStats.ActiveUnderworldContractID = ""
				l.logAdminAuditLocked("CONTRACT_COMPLETED", wallet, "ID: CONTRACT-017, Payout: 20000.00")
				l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> Governor's hostage ring breached. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
				l.applyDynamicScalingLocked()
				l.leaderboard[wallet] = playerStats
			}
		}

		// PILLAR 3: Underworld Contract Completion (CONTRACT-019).
		// Objective: Successful Heist against Arena Center while Ghosted vs 3+ Allied Traps.
		if playerStats.ActiveUnderworldContractID == "CONTRACT-019" {
			if time.Now().Before(playerStats.GhostProtocolExpiresAt) {
				isArenaCenter := false
				for _, t := range targetClub.Territories {
					if t == "arena_center" {
						isArenaCenter = true
						break
					}
				}
				if isArenaCenter {
					trapCount := 0
					for bID := range targetClub.ActiveBuffs {
						if strings.HasPrefix(bID, "TRAP_") {
							if expiry, ok := targetClub.BuffExpirations[bID]; ok && time.Now().Before(expiry) {
								trapCount++
							}
						}
					}
					if targetClub.AlliedClubID != "" {
						if ally, ok := l.clubs[targetClub.AlliedClubID]; ok {
							for bID := range ally.ActiveBuffs {
								if strings.HasPrefix(bID, "TRAP_") {
									if expiry, ok := ally.BuffExpirations[bID]; ok && time.Now().Before(expiry) {
										trapCount++
									}
								}
							}
						}
					}
					if trapCount >= 3 {
						const rewardMicro = 30000 * 1000000
						l.playerBalances[wallet] += rewardMicro
						playerStats.ActiveUnderworldContractID = ""
						l.logAdminAuditLocked("CONTRACT_COMPLETED", wallet, "ID: CONTRACT-019, Payout: 30000.00")
						l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> The Invisible Hand of Chaos has struck. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
						l.applyDynamicScalingLocked()
						l.leaderboard[wallet] = playerStats
					}
				}
			}
		}

		// PILLAR 3: Underworld Contract Completion (CONTRACT-020).
		// Objective: Successful Heist vs Org with 5+ hostages while Wanted 40+.
		if playerStats.ActiveUnderworldContractID == "CONTRACT-020" && playerStats.WantedLevel >= 40 {
			victimWallet := strings.ToLower(targetClub.OwnerWallet)
			if victimStats, vExists := l.leaderboard[victimWallet]; vExists && len(victimStats.HeldHostageCards) >= 5 {
				const rewardMicro = 20000 * 1000000
				l.playerBalances[wallet] += rewardMicro
				playerStats.ActiveUnderworldContractID = ""
				l.logAdminAuditLocked("CONTRACT_COMPLETED", wallet, "ID: CONTRACT-020, Payout: 40000.00")
				l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> The Apex Syndicate has been fleeced. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
				l.applyDynamicScalingLocked()
				l.leaderboard[wallet] = playerStats
			}
		}

		// PILLAR 3: Underworld Contract Completion (CONTRACT-022).
		// Objective: Successful Heist vs Org with 10+ hostages while Wanted 60+.
		if playerStats.ActiveUnderworldContractID == "CONTRACT-022" && playerStats.WantedLevel >= 60 {
			victimWallet := strings.ToLower(targetClub.OwnerWallet)
			if victimStats, vExists := l.leaderboard[victimWallet]; vExists && len(victimStats.HeldHostageCards) >= 10 {
				const rewardMicro = 35000 * 1000000
				l.playerBalances[wallet] += rewardMicro
				playerStats.ActiveUnderworldContractID = ""
				l.logAdminAuditLocked("CONTRACT_COMPLETED", wallet, "ID: CONTRACT-022, Payout: 60000.00")
				l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> The Sovereign Hostage Heist was successful. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
				l.applyDynamicScalingLocked()
				l.leaderboard[wallet] = playerStats
			}
		}

		// PILLAR 3: Underworld Contract Completion (CONTRACT-023).
		// Objective: Successful Heist vs 'Arena Center' @ Wanted 100+.
		if playerStats.ActiveUnderworldContractID == "CONTRACT-023" && playerStats.WantedLevel >= 100 {
			isArenaCenter := false
			for _, t := range targetClub.Territories {
				if t == "arena_center" {
					isArenaCenter = true
					break
				}
			}

			if isArenaCenter {
				const rewardMicro = 50000 * 1000000
				l.playerBalances[wallet] += rewardMicro
				playerStats.ActiveUnderworldContractID = ""
				l.logAdminAuditLocked("CONTRACT_COMPLETED", wallet, "ID: CONTRACT-023, Payout: 100000.00")
				l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> The Arena Center Apex Heist was successful. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
				l.applyDynamicScalingLocked()
				l.leaderboard[wallet] = playerStats
			}
		}

		// PILLAR 3: Underworld Contract Completion (CONTRACT-027).
		// Objective: Successful Heist while holding 'Smuggler' role and Wanted 30+.
		if playerStats.ActiveUnderworldContractID == "CONTRACT-027" && playerStats.JobRole == "Smuggler" && playerStats.WantedLevel >= 30 {
			const rewardMicro = 20000 * 1000000
			l.playerBalances[wallet] += rewardMicro
			playerStats.ActiveUnderworldContractID = ""
			l.logAdminAuditLocked("CONTRACT_COMPLETED", wallet, "ID: CONTRACT-027, Payout: 20000.00")
			l.sendToClientLocked(env.FromID, Envelope{
				Type:    "admin_notification",
				Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> Smuggling strike successful. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0)),
			})
			l.applyDynamicScalingLocked()
			l.leaderboard[wallet] = playerStats
		}
	} else {
		status = "failure"
		playerStats.WantedLevel += 15
		playerStats.Playstyle.RiskTolerance += 0.10
		playerStats.HeistAttempts++

		// MOJO GAIN: Reward the club for successful defense
		mojoGain := s.CalculateMojoGain(l, targetClub, "DEFENSE", 0) // This calls s.CalculateMojoGain, not PlayerStats.GetEffectiveMojo
		targetClub.Mojo += mojoGain
		l.achievementService.CheckMojoSurgeAchievementLocked(l, targetClub.ID)

		// PILLAR 1: Reputation Ripple.
		// Update standings for all club employees to reflect the increased Mojo multiplier from defense.
		for w, s := range l.leaderboard {
			if s.EmployerClubID == targetClub.ID {
				s.Reputation = l.CalculateReputation(s)
				l.leaderboard[w] = s
			}
		}

		targetClub.LastActivity = now // Defense engagement counts as activity

		hasGuardDog := false
		for _, trapItemID := range targetClub.ActiveBuffs {
			if trapItemID == "guard_dog" {
				hasGuardDog = true
				break
			}
		}

		if hasGuardDog {
			rarestCard, found := l.findRarestCardInInventory(wallet)
			if found {
				if targetClub.Jail == nil {
					targetClub.Jail = make(map[int]ServerCard)
				}
				targetClub.Jail[rarestCard.ID] = rarestCard

				// Decrement instead of absolute deletion to handle duplicate card instances
				cardKey := fmt.Sprintf("CARD-%d", rarestCard.ID)
				playerStats.Inventory[cardKey]--
				if playerStats.Inventory[cardKey] <= 0 {
					delete(playerStats.Inventory, cardKey)
				}

				if playerStats.JailedCards == nil {
					playerStats.JailedCards = make(map[int]string)
				}
				playerStats.JailedCards[rarestCard.ID] = targetClub.ID
				l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🚨 <b>GUARD DOG BUST:</b> You were caught by a Bio-Guard Dog! Your rarest card (%s) has been jailed by %s."}`, rarestCard.Name, targetClub.Name))})
			}
		}
	}

	// PILLAR 3: Intelligence Stealth.
	// Heists trigger a security alarm and map alert unless suppressed by a Cyber-Jammer.
	// This applies regardless of whether the perpetrator is an external agent or the owner.
	if playerStats.HasCyberJammer {
		playerStats.HasCyberJammer = false
		playerStats.HeistAlarmsJammerCount++ // Increment successful jammer uses
		l.logAdminAuditLocked("HEIST_JAMMED", wallet, fmt.Sprintf("Security alarm suppressed for %s", targetClub.Name))
	} else {
		// 1. Map Alert (Visual "Under Attack" pulse on failure)
		if status == "failure" {
			targetClub.LastHeistAt = now
		}

		// 2. Chat Alarm (Warning to members and owner)
		warningText := fmt.Sprintf("🚨 <b>SECURITY ALERT:</b> Sensors indicate an unauthorized intrusion attempt at %s treasury vault!", escapeHTML(targetClub.Name))
		warningPayload, _ := json.Marshal(map[string]string{"text": warningText})
		warningEnv := Envelope{Type: "chat", FromID: "SERVER", Payload: warningPayload}
		warningMsg, _ := json.Marshal(warningEnv)

		for cid, client := range l.clients {
			cWallet, ok := l.wallets[cid]
			if !ok {
				continue
			}

			stats := l.leaderboard[strings.ToLower(cWallet)]
			if stats.EmployerClubID == targetClub.ID || strings.EqualFold(cWallet, targetClub.OwnerWallet) {
				select {
				case client.send <- warningMsg:
				default:
				}
			}
		}
	}

	// FINAL REPUTATION SYNC: Reflect all status changes, including potential jailing and jammer usage.
	playerStats.Reputation = l.CalculateReputation(playerStats)
	l.leaderboard[wallet] = playerStats

	// PILLAR 3: Underworld Contract Completion (CONTRACT-018).
	// Objective: Sabotage the Arena Center while bolstered by 3+ allied traps.
	if playerStats.ActiveUnderworldContractID == "CONTRACT-018" {
		isArenaCenter := false
		for _, t := range targetClub.Territories {
			if t == "arena_center" {
				isArenaCenter = true
				break
			}
		}

		if isArenaCenter {
			trapCount := 0
			for bID := range targetClub.ActiveBuffs {
				if strings.HasPrefix(bID, "TRAP_") {
					if expiry, ok := targetClub.BuffExpirations[bID]; ok && time.Now().Before(expiry) {
						trapCount++
					}
				}
			}
			if targetClub.AlliedClubID != "" {
				if ally, ok := l.clubs[targetClub.AlliedClubID]; ok {
					for bID := range ally.ActiveBuffs {
						if strings.HasPrefix(bID, "TRAP_") {
							if expiry, ok := ally.BuffExpirations[bID]; ok && time.Now().Before(expiry) {
								trapCount++
							}
						}
					}
				}
			}
			if trapCount >= 3 {
				const rewardMicro = 25000 * 1000000
				l.playerBalances[wallet] += rewardMicro
				playerStats.ActiveUnderworldContractID = ""
				l.logAdminAuditLocked("CONTRACT_COMPLETED", wallet, "ID: CONTRACT-018, Payout: 25000.00")
				l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> Arena Center fortress breached. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
				l.applyDynamicScalingLocked()
			}
		}
	}

	// PILLAR 3: Underworld Contract Completion (CONTRACT-021).
	// Objective: Successful Sabotage vs Org with 7+ hostages while Wanted 50+.
	if playerStats.ActiveUnderworldContractID == "CONTRACT-021" && playerStats.WantedLevel >= 50 {
		victimWallet := strings.ToLower(targetClub.OwnerWallet)
		if victimStats, vExists := l.leaderboard[victimWallet]; vExists && len(victimStats.HeldHostageCards) >= 7 {
			const rewardMicro = 25000 * 1000000
			l.playerBalances[wallet] += rewardMicro
			playerStats.ActiveUnderworldContractID = ""
			l.logAdminAuditLocked("CONTRACT_COMPLETED", wallet, "ID: CONTRACT-021, Payout: 50000.00")
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> The Hostage Hegemony has collapsed. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
			l.applyDynamicScalingLocked()
		}
	}

	// PILLAR 3: Underworld Contract Completion (CONTRACT-004).
	// Check if the current sabotage attempt fulfills an active criminal mission.
	if playerStats.ActiveUnderworldContractID == "CONTRACT-004" {
		// Objective: Sabotage a district controlled by a Regional Governor.
		// This check uses the ClubService's IsClubRegionalLocked helper.
		if s.IsClubRegionalLocked(l, targetClub) {
			const rewardMicro = 2000 * 1000000
			l.playerBalances[wallet] += rewardMicro
			playerStats.ActiveUnderworldContractID = ""
			l.logAdminAuditLocked("CONTRACT_COMPLETED", wallet, "ID: CONTRACT-004, Payout: 2000.00")
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> Regional Governor disrupted. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
			// No explicit l.leaderboard[wallet] = playerStats needed here, as there's a final one at the end of the function.
		}
	}

	// PILLAR 3: Underworld Contract Completion (CONTRACT-007).
	// Objective: Sabotage the club controlling the 'arena_center' district.
	if playerStats.ActiveUnderworldContractID == "CONTRACT-007" {
		isTargetDistrict := false
		for _, t := range targetClub.Territories {
			if t == "arena_center" {
				isTargetDistrict = true
				break
			}
		}
		if isTargetDistrict {
			const rewardMicro = 2500 * 1000000
			l.playerBalances[wallet] += rewardMicro
			playerStats.ActiveUnderworldContractID = ""
			l.logAdminAuditLocked("CONTRACT_COMPLETED", wallet, "ID: CONTRACT-007, Payout: 2500.00")
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> Arena Center neutralized. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
			l.applyDynamicScalingLocked()
		}
	}

	// PILLAR 3: Underworld Contract Completion (CONTRACT-008).
	// Objective: Successfully sabotage a club that has active 'Cyber-Counter' or 'Cyber-Lock' defenses.
	if playerStats.ActiveUnderworldContractID == "CONTRACT-008" {
		hasTargetDefenses := false
		for _, itemID := range targetClub.ActiveBuffs {
			if itemID == "cyber_counter" || itemID == "cyber_lock" {
				hasTargetDefenses = true
				break
			}
		}
		if hasTargetDefenses {
			const rewardMicro = 3500 * 1000000
			l.playerBalances[wallet] += rewardMicro
			playerStats.ActiveUnderworldContractID = ""
			l.logAdminAuditLocked("CONTRACT_COMPLETED", wallet, "ID: CONTRACT-008, Payout: 3500.00")
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> Defensive infrastructure compromised. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
			l.applyDynamicScalingLocked()
		}
	}

	// PILLAR 3: Underworld Contract Completion (CONTRACT-016).
	// Objective: Successfully sabotage the Arena Center with Wanted Level 30+.
	if playerStats.ActiveUnderworldContractID == "CONTRACT-016" {
		isArenaCenterOwner := false
		for _, t := range targetClub.Territories {
			if t == "arena_center" {
				isArenaCenterOwner = true
				break
			}
		}
		if isArenaCenterOwner && playerStats.WantedLevel >= 30 {
			const rewardMicro = 15000 * 1000000
			l.playerBalances[wallet] += rewardMicro
			playerStats.ActiveUnderworldContractID = ""
			l.logAdminAuditLocked("CONTRACT_COMPLETED", wallet, "ID: CONTRACT-016, Payout: 15000.00")
			l.sendToClientLocked(env.FromID, Envelope{
				Type:    "admin_notification",
				Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> Arena Center hegemony breached. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0)),
			})
			l.applyDynamicScalingLocked()
		}
	}

	// PILLAR 3: Underworld Contract Completion (CONTRACT-014).
	// Objective: Sabotage a Regional Governor's district with Wanted Level 20+.
	if playerStats.ActiveUnderworldContractID == "CONTRACT-014" {
		if s.IsClubRegionalLocked(l, targetClub) && playerStats.WantedLevel >= 20 {
			const rewardMicro = 8000 * 1000000
			l.playerBalances[wallet] += rewardMicro
			playerStats.ActiveUnderworldContractID = ""
			l.logAdminAuditLocked("CONTRACT_COMPLETED", wallet, "ID: CONTRACT-014, Payout: 8000.00")
			l.sendToClientLocked(env.FromID, Envelope{
				Type:    "admin_notification",
				Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> Sovereign infrastructure crippled. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0)),
			})
			l.applyDynamicScalingLocked()
		}
	}

	// PILLAR 3: Underworld Contract Completion (CONTRACT-011).
	// Objective: Sabotage a titan club (controlling 3+ territories).
	if playerStats.ActiveUnderworldContractID == "CONTRACT-011" {
		if len(targetClub.Territories) >= 3 {
			const rewardMicro = 6000 * 1000000
			l.playerBalances[wallet] += rewardMicro
			playerStats.ActiveUnderworldContractID = ""
			l.logAdminAuditLocked("CONTRACT_COMPLETED", wallet, "ID: CONTRACT-011, Payout: 6000.00")
			l.sendToClientLocked(env.FromID, Envelope{
				Type:    "admin_notification",
				Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> Titan infrastructure crippled. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0)),
			})
			l.applyDynamicScalingLocked()
		}
	}

	// PILLAR 3: Underworld Contract Completion.
	// Check if the current sabotage attempt fulfills an active criminal mission.
	if playerStats.ActiveUnderworldContractID == "CONTRACT-001" {
		// Objective: Sabotage a club controlling the East Gate district.
		isTargetDistrict := false
		for _, t := range targetClub.Territories {
			if t == "east_gate" {
				isTargetDistrict = true
				break
			}
		}

		if isTargetDistrict {
			const rewardMicro = 1500 * 1000000
			l.playerBalances[wallet] += rewardMicro
			playerStats.ActiveUnderworldContractID = ""
			l.logAdminAuditLocked("CONTRACT_COMPLETED", wallet, "ID: CONTRACT-001, Payout: 1500.00")
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> High-infamy sabotage successful. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
		}
	}

	// PILLAR 3: Underworld Contract Completion (CONTRACT-025).
	// Objective: Successful Sabotage while holding 'Kidnapper' role.
	if playerStats.ActiveUnderworldContractID == "CONTRACT-025" && playerStats.JobRole == "Kidnapper" {
		const rewardMicro = 15000 * 1000000
		l.playerBalances[wallet] += rewardMicro
		playerStats.ActiveUnderworldContractID = ""
		l.logAdminAuditLocked("CONTRACT_COMPLETED", wallet, "ID: CONTRACT-025, Payout: 15000.00")
		l.sendToClientLocked(env.FromID, Envelope{
			Type:    "admin_notification",
			Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> Disruptive strike successful. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0)),
		})
		l.applyDynamicScalingLocked()
	}

	// PILLAR 3: Underworld Contract Completion (CONTRACT-026).
	// Objective: Successful Sabotage vs Org whose owner has received 5+ reparations.
	if playerStats.ActiveUnderworldContractID == "CONTRACT-026" && playerStats.JobRole == "Saboteur" {
		targetOwnerWallet := strings.ToLower(targetClub.OwnerWallet)
		if targetOwnerStats, exists := l.leaderboard[targetOwnerWallet]; exists {
			if targetOwnerStats.ReparationsReceivedCount >= 5 {
				const rewardMicro = 18000 * 1000000
				l.playerBalances[wallet] += rewardMicro
				playerStats.ActiveUnderworldContractID = ""
				l.logAdminAuditLocked("CONTRACT_COMPLETED", wallet, "ID: CONTRACT-026, Payout: 18000.00")
				l.sendToClientLocked(env.FromID, Envelope{
					Type:    "admin_notification",
					Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> Reparation sabotage successful. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0)),
				})
				l.applyDynamicScalingLocked()
			}
		}
	}
	l.leaderboard[wallet] = playerStats // Final write-back for contract state

	// PILLAR 1: Achievement Integration.
	// Evaluate Heist Saboteur progress after saving state to the leaderboard.
	l.achievementService.CheckHeistSaboteurAchievementLocked(l, wallet)

	l.logAdminAuditLocked("HEIST_ATTEMPT", wallet, fmt.Sprintf("Target: %s, Result: %s, Loot: %.2f, FenceFee: %.2f", data.TargetClubID, status, netLoot, fenceFee))
	if status == "success" {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>HEIST SUCCESS:</b> You looted %.2f $VBV from %s (Net after Fence Fee)."}`, netLoot, targetClub.Name))})
	}

	response, _ := json.Marshal(map[string]interface{}{
		"status":            status,
		"wanted_level":      playerStats.WantedLevel,
		"cunning":           playerStats.Cunning,
		"effective_cunning": playerStats.GetEffectiveCunning(),
		"playstyle":         playerStats.Playstyle,
		"heist_attempts":    playerStats.HeistAttempts,
		"kidnap_eligible":   canKidnap,
		"target_club_id":    data.TargetClubID,
	})
	l.sendToClientLocked(env.FromID, Envelope{Type: "heist_result", Payload: response})

	// Trigger Global Sync so others see the treasury loot and the player's new Wanted Level
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// HandleMutationVectorRealignment allows players to re-allocate card power points at specialized facilities.
// PILLAR 6: Specialized Gene-Editing.
func (s *ClubService) HandleMutationVectorRealignment(l *Lobby, env *Envelope) {
	var data struct {
		CardID   int    `json:"card_id"`
		ClubID   string `json:"club_id"`
		NewPower [4]int `json:"new_power"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	wallet, ok := l.wallets[env.FromID]
	if !ok {
		return
	}

	// 1. Facility Verification: Mutation protocols require Vitality or Elemental labs.
	club, exists := l.clubs[data.ClubID]
	if !exists || (club.Type != "Vitality" && club.Type != "Elemental") {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Mutation Failed: Specialized lab facility required."}`)})
		return
	}

	// 2. Ownership Verification: Player must possess the asset.
	stats := l.leaderboard[wallet]
	cardKey := fmt.Sprintf("CARD-%d", data.CardID)
	if count, has := stats.Inventory[cardKey]; !has || count <= 0 {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Mutation Failed: You do not possess this card."}`)})
		return
	}

	// 3. Invariant Verification: Sum of power must remain constant (No Power Creep).
	card := l.inventory[data.CardID]
	currentSum, newSum := 0, 0
	for i := 0; i < 4; i++ {
		currentSum += card.Power[i]
		newSum += data.NewPower[i]
		if data.NewPower[i] < 5 { // Minimum operational floor for any side.
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Mutation Failed: Vector instability detected. Minimum side power is 5."}`)})
			return
		}
	}
	if currentSum != newSum {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Mutation Failed: Energy conservation breach. Power sum must be identical."}`)})
		return
	}

	// 4. Industrial Fee Processing: 500 VBV per re-allocation.
	const mutationCostMicro = 500 * 1000000
	if l.playerBalances[wallet] < mutationCostMicro {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Mutation Failed: Insufficient VBV for protocol."}`)})
		return
	}

	// Execute Virtual Liability Shift
	l.playerBalances[wallet] -= mutationCostMicro

	// PILLAR 6: Probability Modifier.
	// Success is determined by facility quality (Mojo + Staff).
	// If Mutation Insurance is active, success is guaranteed and the buff is consumed.
	successChance := s.CalculateMutationSuccessChance(l, club)
	usedInsurance := false
	if stats.HasMutationInsurance {
		successChance = 1.1 // Force 100% success
		usedInsurance = true
		stats.HasMutationInsurance = false
	}

	if rand.Float64() > successChance {
		// FAILURE: Apply Mutation Scar (Permanent Artifact reduction)
		l.applyMutationScars(data.CardID, 50)
		club.MutationFailures++
		l.logAdminAuditLocked("MUTATION_FAILURE", wallet, fmt.Sprintf("Vector realignment failed for card %d at club %s", data.CardID, club.Name))

		failedCard := l.inventory[data.CardID]
		l.sendToClientLocked(env.FromID, Envelope{
			Type:    "admin_notification",
			Payload: json.RawMessage(fmt.Sprintf(`{"text":"🚨 <b>MUTATION FAILURE:</b> The procedure failed. %s has suffered permanent mutation scars (-50 Artifact)."}`, failedCard.Name)),
		})
		l.applyDynamicScalingLocked()
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		return
	}

	// 5. Atomic Routing: Commission Split.
	var govCutMicro uint64 = 0
	context := "VECTOR_REALIGNMENT" // Context for telemetry

	if club.AlliedClubID != "" && s.IsClubRegionalLocked(l, club) {
		if ally, ok := l.clubs[club.AlliedClubID]; ok {
			govCutMicro = (mutationCostMicro * 5) / 100 // 5% of total cost (integer micro math)


			// Route the governor's cut separately
			if l.tokenSinkRouter != nil {
				// PILLAR 2: Instant Settlement. 
				// Route to the ally as a ClubShare to update their treasury instantly.
				matrix := RevenueSplitMatrix{FaucetShare: 0.0, ClubShare: 1.0, GovernanceShare: 0.0}
				numericID, _ := strconv.ParseUint(strings.TrimPrefix(ally.ID, "CLUB-"), 10, 64)
				_ = l.tokenSinkRouter.RouteCriminalTax(context+"_GOV_SPLIT", govCutMicro, matrix, numericID, "")

				// PILLAR 2: UI Parity Sync.
				if node, ok := l.tokenSinkRouter.ActiveClubs[numericID]; ok {
					ally.TreasuryMicro = node.TreasuryBalance
				}

				// PILLAR 1: Organizational Progression.
				// Allied governors earn Mojo for procedures performed within their coalition.
				allyMojo := s.CalculateMojoGain(l, ally, "REVENUE", float64(govCutMicro)/1000000.0)
				ally.Mojo += allyMojo
				l.achievementService.CheckMojoSurgeAchievementLocked(l, ally.ID)
			} else {
				ally.TreasuryMicro += govCutMicro
			}
			ally.LastActivity = time.Now()

			// Record event in alliance history
			ally.CommissionHistory = append(ally.CommissionHistory, CommissionEvent{
				Timestamp: time.Now().Unix(), SourceClub: club.Name, Type: "VECTOR", Amount: float64(govCutMicro) / 1000000.0,
			})
			if len(ally.CommissionHistory) > 50 {
				ally.CommissionHistory = ally.CommissionHistory[len(ally.CommissionHistory)-50:]
			}
			l.logAdminAuditLocked("MUTATION_GOV_SPLIT", wallet, fmt.Sprintf("Recipient: %s, Fee: %.2f", ally.Name, float64(govCutMicro)/1000000.0))
		}
	}

	// Route the remaining portion of the mutation cost
	remainingCostMicro := mutationCostMicro - govCutMicro
	if l.tokenSinkRouter != nil {
		matrix := RevenueSplitMatrix{FaucetShare: 0.90, ClubShare: 0.10, GovernanceShare: 0.0} // Default split for remaining
		// Map string ID ("CLUB-123") to numeric router target
		numericID, _ := strconv.ParseUint(strings.TrimPrefix(club.ID, "CLUB-"), 10, 64)
		_ = l.tokenSinkRouter.RouteCriminalTax(context, remainingCostMicro, matrix, numericID, "GLOBAL")
	}

	if usedInsurance {
		l.logAdminAuditLocked("INSURANCE_CONSUMED", wallet, fmt.Sprintf("Used on card %d realignment", data.CardID))
	}

	// 6. Commit State: Update both active inventory and persistent archival cache.
	card.Power = data.NewPower
	l.inventory[data.CardID] = card
	club.MutationSuccesses++
	l.persistentCardCache[data.CardID] = card

	// PILLAR 1: Organizational Progression.
	// Clubs earn Mojo for performing successful gene-editing procedures.
	procMojo := s.CalculateMojoGain(l, club, "REVENUE", float64(mutationCostMicro)/1000000.0)
	club.Mojo += procMojo

	// 6.1 Record Forensic Mutation Event
	stats.MutationHistory = append(stats.MutationHistory, MutationEvent{
		Timestamp: time.Now().Unix(),
		Type:      "VECTOR",
		CardID:    data.CardID,
		Details:   fmt.Sprintf("Power Realigned: %v", data.NewPower),
	})

	// Update reputation immediately to reflect the new tactical standing
	stats.Reputation = l.CalculateReputation(stats)
	l.leaderboard[wallet] = stats

	l.logAdminAuditLocked("VECTOR_REALIGNMENT", wallet, fmt.Sprintf("Vector realignment successful for card %d at club %s", data.CardID, club.Name))

	// Immersion: Notify the player of successful gene-editing
	l.sendToClientLocked(env.FromID, Envelope{
		Type:    "admin_notification",
		Payload: json.RawMessage(fmt.Sprintf(`{"text":"🧬 <b>MUTATION SUCCESS:</b> %s vectors have been realigned."}`, card.Name)),
	})

	// 7. Global Synchronization
	l.applyDynamicScalingLocked()
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// HandleMutationMoodRecalibration allows players to permanently change a card's native element.
// PILLAR 6: Specialized Gene-Editing.
func (s *ClubService) HandleMutationMoodRecalibration(l *Lobby, env *Envelope) {
	var data struct {
		CardID  int    `json:"card_id"`
		ClubID  string `json:"club_id"`
		NewMood string `json:"new_mood"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	wallet, ok := l.wallets[env.FromID]
	if !ok {
		return
	}

	// 1. Facility Verification: Vitality or Elemental labs required.
	club, exists := l.clubs[data.ClubID]
	if !exists || (club.Type != "Vitality" && club.Type != "Elemental") {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Mutation Failed: Specialized lab facility required."}`)})
		return
	}

	// 2. Ownership Verification: Player must possess the asset and a Catalyst.
	stats := l.leaderboard[wallet]
	cardKey := fmt.Sprintf("CARD-%d", data.CardID)
	if count, has := stats.Inventory[cardKey]; !has || count <= 0 {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Mutation Failed: You do not possess this card."}`)})
		return
	}

	if count, has := stats.Inventory["mood_catalyst"]; !has || count <= 0 {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Mutation Failed: Mood Catalyst required in inventory."}`)})
		return
	}

	// 3. Element Validation
	validMoods := map[string]bool{"Volatile": true, "Serene": true, "Spirited": true, "Grounded": true, "Neutral": true}
	if !validMoods[data.NewMood] {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Mutation Failed: Invalid mood element target."}`)})
		return
	}

	// 4. Industrial Fee Processing: 250 VBV per recalibration.
	const recalCostMicro = 250 * 1000000
	if l.playerBalances[wallet] < recalCostMicro {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Mutation Failed: Insufficient VBV for protocol."}`)})
		return
	}

	// Execute Deductions
	l.playerBalances[wallet] -= recalCostMicro
	stats.Inventory["mood_catalyst"]--
	if stats.Inventory["mood_catalyst"] <= 0 {
		delete(stats.Inventory, "mood_catalyst")
	}

	// PILLAR 6: Probability Modifier.
	// Procedure requires Catalyst consumption regardless of success.
	// Mutation Insurance guarantees success for elemental realignment.
	successChance := s.CalculateMutationSuccessChance(l, club)
	usedInsurance := false
	if stats.HasMutationInsurance {
		successChance = 1.1
		usedInsurance = true
		stats.HasMutationInsurance = false
	}

	if rand.Float64() > successChance {
		// FAILURE: Apply Mutation Scar
		l.applyMutationScars(data.CardID, 50)
		club.MutationFailures++
		l.logAdminAuditLocked("MUTATION_FAILURE", wallet, fmt.Sprintf("Mood recalibration failed for card %d at club %s", data.CardID, club.Name))

		failedCard := l.inventory[data.CardID]
		l.sendToClientLocked(env.FromID, Envelope{
			Type:    "admin_notification",
			Payload: json.RawMessage(fmt.Sprintf(`{"text":"🚨 <b>MUTATION FAILURE:</b> Elemental alignment failed. %s has suffered permanent mutation scars."}`, failedCard.Name)),
		})
		l.applyDynamicScalingLocked()
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		return
	}

	// 5. Atomic Routing: Commission Split.
	var govCutMicro uint64 = 0
	context := "MOOD_RECALIBRATION" // Context for telemetry

	if club.AlliedClubID != "" && s.IsClubRegionalLocked(l, club) {
		if ally, ok := l.clubs[club.AlliedClubID]; ok {
			govCutMicro = (recalCostMicro * 5) / 100 // 5% of total cost (integer micro math)


			// Route the governor's cut separately
			if l.tokenSinkRouter != nil {
				// PILLAR 2: Instant Settlement.
				matrix := RevenueSplitMatrix{FaucetShare: 0.0, ClubShare: 1.0, GovernanceShare: 0.0}
				numericID, _ := strconv.ParseUint(strings.TrimPrefix(ally.ID, "CLUB-"), 10, 64)
				_ = l.tokenSinkRouter.RouteCriminalTax(context+"_GOV_SPLIT", govCutMicro, matrix, numericID, "")

				// PILLAR 2: UI Parity Sync.
				if node, ok := l.tokenSinkRouter.ActiveClubs[numericID]; ok {
					ally.TreasuryMicro = node.TreasuryBalance
				}

				// PILLAR 1: Allied Mojo Gain.
				allyMojo := s.CalculateMojoGain(l, ally, "REVENUE", float64(govCutMicro)/1000000.0)
				ally.Mojo += allyMojo
				l.achievementService.CheckMojoSurgeAchievementLocked(l, ally.ID)
			} else {
				ally.TreasuryMicro += govCutMicro
			}
			ally.LastActivity = time.Now()

			// Record event in alliance history
			ally.CommissionHistory = append(ally.CommissionHistory, CommissionEvent{
				Timestamp: time.Now().Unix(), SourceClub: club.Name, Type: "MOOD", Amount: float64(govCutMicro) / 1000000.0,
			})
			if len(ally.CommissionHistory) > 50 {
				ally.CommissionHistory = ally.CommissionHistory[len(ally.CommissionHistory)-50:]
			}
			l.logAdminAuditLocked("MUTATION_GOV_SPLIT", wallet, fmt.Sprintf("Recipient: %s, Fee: %.2f", ally.Name, float64(govCutMicro)/1000000.0))
		}
	}

	// Route the remaining portion of the mutation cost
	remainingCostMicro := recalCostMicro - govCutMicro
	if l.tokenSinkRouter != nil {
		matrix := RevenueSplitMatrix{FaucetShare: 0.90, ClubShare: 0.10, GovernanceShare: 0.0} // Default split for remaining
		numericID, _ := strconv.ParseUint(strings.TrimPrefix(club.ID, "CLUB-"), 10, 64)
		_ = l.tokenSinkRouter.RouteCriminalTax(context, remainingCostMicro, matrix, numericID, "GLOBAL")
	}

	if usedInsurance {
		l.logAdminAuditLocked("INSURANCE_CONSUMED", wallet, fmt.Sprintf("Used on card %d element shift", data.CardID))
	}

	// 6. Commit State: Update both active inventory and persistent archival cache.
	card := l.inventory[data.CardID]
	card.Mood = data.NewMood
	club.MutationSuccesses++
	l.inventory[data.CardID] = card
	l.persistentCardCache[data.CardID] = card

	// PILLAR 1: Organizational Progression.
	procMojo := s.CalculateMojoGain(l, club, "REVENUE", float64(recalCostMicro)/1000000.0)
	club.Mojo += procMojo
	l.achievementService.CheckMojoSurgeAchievementLocked(l, club.ID)

	// 6.1 Record Forensic Mutation Event
	stats.MutationHistory = append(stats.MutationHistory, MutationEvent{
		Timestamp: time.Now().Unix(),
		Type:      "MOOD",
		CardID:    data.CardID,
		Details:   fmt.Sprintf("Element recalibrated to %s", data.NewMood),
	})

	// Update reputation and leaderboard
	stats.Reputation = l.CalculateReputation(stats)
	l.leaderboard[wallet] = stats

	l.logAdminAuditLocked("MOOD_RECALIBRATION", wallet, fmt.Sprintf("Mood recalibration successful for card %d at club %s", data.CardID, club.Name))

	// Immersion: Notify the player of successful element shift
	l.sendToClientLocked(env.FromID, Envelope{
		Type:    "admin_notification",
		Payload: json.RawMessage(fmt.Sprintf(`{"text":"🧬 <b>MUTATION SUCCESS:</b> %s is now aligned with the %s element."}`, card.Name, data.NewMood)),
	})

	// 7. Global Synchronization
	l.applyDynamicScalingLocked()
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// HandleMutationLoyaltySynthesis allows players to instantly soul-bond a card for a high fee.
// PILLAR 6: Specialized Gene-Editing.
func (s *ClubService) HandleMutationLoyaltySynthesis(l *Lobby, env *Envelope) {
	var data struct {
		CardID int    `json:"card_id"`
		ClubID string `json:"club_id"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	wallet, ok := l.wallets[env.FromID]
	if !ok {
		return
	}

	// 1. Facility Verification: Vitality or Elemental labs required.
	club, exists := l.clubs[data.ClubID]
	if !exists || (club.Type != "Vitality" && club.Type != "Elemental") {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Synthesis Failed: Specialized lab facility required."}`)})
		return
	}

	// 2. Ownership & Invariant Verification: Player must possess card and it must not be maxed.
	stats := l.leaderboard[wallet]
	cardKey := fmt.Sprintf("CARD-%d", data.CardID)
	if count, has := stats.Inventory[cardKey]; !has || count <= 0 {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Synthesis Failed: You do not possess this card."}`)})
		return
	}

	card := l.inventory[data.CardID]
	if card.Loyalty >= 100 {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Synthesis Failed: This asset is already Soul-Bonded."}`)})
		return
	}

	// 3. Industrial Fee Processing: 1,000 VBV per synthesis.
	const synthesisCostMicro = 1000 * 1000000
	if l.playerBalances[wallet] < synthesisCostMicro {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Synthesis Failed: Insufficient VBV for soul-bonding protocol."}`)})
		return
	}

	// Execute Virtual Liability Shift
	l.playerBalances[wallet] -= synthesisCostMicro

	// PILLAR 2: Industrial Seal. 
	// The fee is routed regardless of the procedure outcome to ensure ledger circularity.	
	var govCutMicro uint64 = 0
	var targetGovDistrict string = "GLOBAL"
	context := "LOYALTY_SYNTHESIS" // Context for telemetry

	if club.AlliedClubID != "" && s.IsClubRegionalLocked(l, club) {
		if ally, ok := l.clubs[club.AlliedClubID]; ok {
			govCutMicro = (synthesisCostMicro * 5) / 100 // 5% of total cost (integer micro math)


			// Route the governor's cut separately
			if l.tokenSinkRouter != nil {
				// PILLAR 2: Instant Settlement.
				matrix := RevenueSplitMatrix{FaucetShare: 0.0, ClubShare: 1.0, GovernanceShare: 0.0}
				numericID, _ := strconv.ParseUint(strings.TrimPrefix(ally.ID, "CLUB-"), 10, 64)
				_ = l.tokenSinkRouter.RouteCriminalTax(context+"_GOV_SPLIT", govCutMicro, matrix, numericID, "")

				// PILLAR 2: UI Parity Sync.
				if node, ok := l.tokenSinkRouter.ActiveClubs[numericID]; ok {
					ally.TreasuryMicro = node.TreasuryBalance
				}

				// PILLAR 1: Allied Mojo Gain.
				allyMojo := s.CalculateMojoGain(l, ally, "REVENUE", float64(govCutMicro)/1000000.0)
				ally.Mojo += allyMojo
				l.achievementService.CheckMojoSurgeAchievementLocked(l, ally.ID)
			} else {
				ally.TreasuryMicro += govCutMicro
			}
			ally.LastActivity = time.Now()

			// Record event in alliance history
			ally.CommissionHistory = append(ally.CommissionHistory, CommissionEvent{
				Timestamp: time.Now().Unix(), SourceClub: club.Name, Type: "LOYALTY", Amount: float64(govCutMicro) / 1000000.0,
			})
			if len(ally.CommissionHistory) > 50 {
				ally.CommissionHistory = ally.CommissionHistory[len(ally.CommissionHistory)-50:]
			}
			l.logAdminAuditLocked("MUTATION_GOV_SPLIT", wallet, fmt.Sprintf("Recipient: %s, Fee: %.2f", ally.Name, float64(govCutMicro)/1000000.0))
		}
	}

	// Route the remaining portion of the mutation cost
	remainingCostMicro := synthesisCostMicro - govCutMicro
	if l.tokenSinkRouter != nil {
		matrix := RevenueSplitMatrix{FaucetShare: 0.90, ClubShare: 0.10, GovernanceShare: 0.0} // Default split for remaining
		numericID, _ := strconv.ParseUint(strings.TrimPrefix(club.ID, "CLUB-"), 10, 64)
		_ = l.tokenSinkRouter.RouteCriminalTax(context, remainingCostMicro, matrix, numericID, "GLOBAL")
	}

	// PILLAR 6: Probability Modifier.
	// Loyalty synthesis is a high-stakes roll; insurance is highly recommended.
	successChance := s.CalculateMutationSuccessChance(l, club)
	usedInsurance := false
	if stats.HasMutationInsurance {
		successChance = 1.1
		usedInsurance = true
		stats.HasMutationInsurance = false
	}

	if rand.Float64() > successChance {
		// FAILURE: Apply Mutation Scar (Permanent Artifact reduction)
		l.applyMutationScars(data.CardID, 50)
		club.MutationFailures++
		l.logAdminAuditLocked("MUTATION_FAILURE", wallet, fmt.Sprintf("Loyalty synthesis failed for card %d at club %s", data.CardID, club.Name))

		l.sendToClientLocked(env.FromID, Envelope{
			Type:    "admin_notification",
			Payload: json.RawMessage(fmt.Sprintf(`{"text":"🚨 <b>MUTATION FAILURE:</b> Soul-bonding failed. %s has suffered permanent mutation scars."}`, card.Name)),
		})
		
		// PILLAR 1: Career Service Update.
		// Recalculate reputation even on failure to reflect procedure attempts.
		stats.Reputation = l.CalculateReputation(stats)
		l.leaderboard[wallet] = stats
		
		l.applyDynamicScalingLocked()
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		return
	}

	if usedInsurance {
		l.logAdminAuditLocked("INSURANCE_CONSUMED", wallet, fmt.Sprintf("Used on card %d soul-bonding", data.CardID))
	}

	// 5. Commit State: Update both active inventory and persistent archival cache.
	card.Loyalty = 100
	l.inventory[data.CardID] = card

	// PILLAR 1: Organizational Progression.
	procMojo := s.CalculateMojoGain(l, club, "REVENUE", float64(synthesisCostMicro)/1000000.0)
	club.Mojo += procMojo

	club.MutationSuccesses++
	l.persistentCardCache[data.CardID] = card

	// 5.1 Record Forensic Mutation Event
	stats.MutationHistory = append(stats.MutationHistory, MutationEvent{
		Timestamp: time.Now().Unix(),
		Type:      "LOYALTY",
		CardID:    data.CardID,
		Details:   "Instant Loyalty Synthesis (Soul-Bonded)",
	})

	// Update reputation immediately to reflect the new soul-bonded standing
	stats.Reputation = l.CalculateReputation(stats)
	l.leaderboard[wallet] = stats

	l.logAdminAuditLocked("LOYALTY_SYNTHESIS", wallet, fmt.Sprintf("Loyalty synthesis successful for card %d at club %s", data.CardID, club.Name))

	// Immersion: Notify the player of successful synthesis
	l.sendToClientLocked(env.FromID, Envelope{
		Type:    "admin_notification",
		Payload: json.RawMessage(fmt.Sprintf(`{"text":"🧬 <b>SYNTHESIS SUCCESS:</b> %s is now soul-bonded to your profile."}`, card.Name)),
	})

	// 6. Global Synchronization
	l.applyDynamicScalingLocked()
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// HandleSabotage allows a player to pay $VBV to disable a target club's hardware defenses for 1 hour.
func (s *ClubService) HandleSabotage(l *Lobby, env *Envelope) {
	var data struct {
		TargetClubID string `json:"target_club_id"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	wallet, ok := l.wallets[env.FromID]
	if !ok {
		return
	}

	targetClub, exists := l.clubs[data.TargetClubID]
	if !exists {
		return
	}

	// 1. Calculate Total Sabotage Cost (Industrial Loop)
	// PILLAR 1: Political Influence. Use dynamic cost based on target resilience.
	totalCostMicro := l.CalculateSabotageCostLocked(targetClub)
	
	if l.playerBalances[wallet] < totalCostMicro {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"❌ Sabotage Failed: Insufficient $VBV rewards. Total required: %.0f $VBV."}`, float64(totalCostMicro)/1000000.0))})
		return
	}
	l.playerBalances[wallet] -= totalCostMicro

	// Determine territory context for telemetry
	telemetryContext := "neutral_zone"
	if len(targetClub.Territories) > 0 {
		telemetryContext = targetClub.Territories[0]
	}

	// PILLAR 2: Unified Organizational Accounting (Token-Sink Router migration).
	if l.tokenSinkRouter != nil {
		// Proportional split (2/3 to Faucet, 1/3 to Club) maintained for dynamic costs.
		dynamicBaseMicro := (totalCostMicro * 1000) / 1500
		dynamicSurchargeMicro := totalCostMicro - dynamicBaseMicro

		// A. Route Dynamic Base Cost to Faucet
		matrixBase := RevenueSplitMatrix{FaucetShare: 1.0, ClubShare: 0.0, GovernanceShare: 0.0}
		_ = l.tokenSinkRouter.RouteCriminalTax("SABOTAGE_BASE", dynamicBaseMicro, matrixBase, 0, "")

		// B. Route Infiltration Surcharge
		if s.IsClubRegionalLocked(l, targetClub) {
			matrixSurcharge := RevenueSplitMatrix{FaucetShare: 0.0, ClubShare: 1.0, GovernanceShare: 0.0}
			numericID, _ := strconv.ParseUint(strings.TrimPrefix(targetClub.ID, "CLUB-"), 10, 64)
			_ = l.tokenSinkRouter.RouteCriminalTax(telemetryContext, dynamicSurchargeMicro, matrixSurcharge, numericID, "")

			// Sync treasury from authoritative router node
			if node, ok := l.tokenSinkRouter.ActiveClubs[numericID]; ok {
				targetClub.TreasuryMicro = node.TreasuryBalance
				targetClub.Treasury = float64(node.TreasuryBalance) / 1000000.0
			}
			l.SabotageSurchargeTotal += dynamicSurchargeMicro
			l.logAdminAuditLocked("SABOTAGE_SURCHARGE_PAID", wallet, fmt.Sprintf("Recipient: %s (Target), Amount: 500.00", targetClub.Name))

			// Trigger Security Telemetry logic (Reparations)
			s.handleSabotageReparationsLocked(l, targetClub)
		} else {
			matrixSurcharge := RevenueSplitMatrix{FaucetShare: 1.0, ClubShare: 0.0, GovernanceShare: 0.0}
			_ = l.tokenSinkRouter.RouteCriminalTax("SABOTAGE_SURCHARGE_FAUCET", dynamicSurchargeMicro, matrixSurcharge, 0, "")
			l.logAdminAuditLocked("SABOTAGE_SURCHARGE_TO_FAUCET", wallet, fmt.Sprintf("Target %s is not a Regional Governor", targetClub.Name))
		}

		// Sync physical balance float
		l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0
	} else {
	}

	l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0
	l.applyDynamicScalingLocked()

	// Apply Sabotage: Set a special buff expiration key
	expiry := time.Now().Add(1 * time.Hour)
	if targetClub.BuffExpirations == nil {
		targetClub.BuffExpirations = make(map[string]time.Time)
	}
	targetClub.BuffExpirations["SABOTAGE"] = expiry
	targetClub.LastActivity = time.Now()

	// PILLAR 3: Financial Proof.
	// Record sabotage event on-chain for the audit trail.
	sabotageDetails := map[string]interface{}{
		"target_id": targetClub.ID,
		"perp":      wallet,
		"cost":      float64(totalCostMicro) / 1000000.0,
		"expiry":    expiry.Unix(),
		"ts":        time.Now().Unix(),
	}

	l.logAdminAuditLocked("CLUB_SABOTAGE", wallet, fmt.Sprintf("Target: %s, Expiry: %s", targetClub.Name, expiry.Format(time.RFC822)))
	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🥷 <b>SABOTAGE ACTIVE:</b> %s's hardware defenses are disabled for 1 hour."}`, targetClub.Name))})

	// PILLAR 1: Reputation Ripple.
	// Update standings for all club employees to reflect the Sabotage penalty.
	for w, s := range l.leaderboard {
		if s.EmployerClubID == targetClub.ID || strings.EqualFold(w, targetClub.OwnerWallet) {
			s.Reputation = l.CalculateReputation(s)
			l.leaderboard[w] = s
		}
	}

	// PILLAR 3: Intelligence Stealth.
	// Check for active Cyber-Jammer to suppress the defense alert.
	perpStats := l.leaderboard[wallet]
	if !perpStats.HasCyberJammer {
		// PILLAR 3: Sabotage Warning System.
		// Notify all connected club members and the owner that their defenses have been disrupted.
		warningText := fmt.Sprintf("🚨 <b>DEFENSE ALERT:</b> Sensors indicate %s hardware defenses have been disrupted by a Sabotage protocol!", escapeHTML(targetClub.Name))
		warningPayload, _ := json.Marshal(map[string]string{"text": warningText})
		warningEnv := Envelope{Type: "chat", FromID: "SERVER", Payload: warningPayload}
		warningMsg, _ := json.Marshal(warningEnv)

		for cid, client := range l.clients {
			cWallet, ok := l.wallets[cid]
			if !ok {
				continue
			}

			stats := l.leaderboard[cWallet]
			if stats.EmployerClubID == targetClub.ID || strings.EqualFold(cWallet, targetClub.OwnerWallet) {
				select {
				case client.send <- warningMsg:
				default:
				}
			}
		}
	} else {
		perpStats.HasCyberJammer = false
		l.leaderboard[wallet] = perpStats
		l.logAdminAuditLocked("SABOTAGE_JAMMED", wallet, fmt.Sprintf("Warning suppressed for %s", targetClub.Name))
	}

	// Dispatch on-chain archival
	go func(sd interface{}) {
		jsonPayload, _ := json.Marshal(sd)
		l.sendNoteTx(fmt.Sprintf("VBT_SABOTAGE_LOG:%s", string(jsonPayload)))
	}(sabotageDetails)

	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// handleSabotageReparationsLocked processes the social and telemetry consequences of a sabotage surcharge payout.
func (s *ClubService) handleSabotageReparationsLocked(l *Lobby, targetClub *Club) {
	// PILLAR 1: Security Telemetry. Increment the owner's reparation counter (authoritative).
	targetOwner := strings.ToLower(targetClub.OwnerWallet)
	if ownerStats, exists := l.leaderboard[targetOwner]; exists {
		ownerStats.ReparationsReceivedCount++
		ownerStats.Reputation = l.CalculateReputation(ownerStats)
		l.leaderboard[targetOwner] = ownerStats

		// PILLAR 1: Resilience Alert. Broadcast milestone to the sector.
		if ownerStats.ReparationsReceivedCount == 5 {
			name := l.ResolveEnvoiName(targetClub.OwnerWallet)
			alert := fmt.Sprintf("🛡️ <b>RESILIENCE ALERT:</b> Governor %s has been designated as a <b>Hardened Sector Leader</b> after securing their 5th reparation!", escapeHTML(name))
			payload, _ := json.Marshal(map[string]string{"text": alert})
			l.broadcast <- jsonListEnvelope("chat", payload)
		}
	}

	// PILLAR 1: Reparation Notification. Inform the victim of the payout.
	ownerCID := l.getClientIDFromWalletLocked(targetClub.OwnerWallet)
	if ownerCID != "" {
		msg := fmt.Sprintf(`{"text":"🛡️ <b>REPARATION RECEIVED:</b> Your organization received a 500 $VBV surcharge from an infiltration attempt on %s."}`, escapeHTML(targetClub.Name))
		l.sendToClientLocked(ownerCID, Envelope{Type: "admin_notification", FromID: "SERVER", Payload: json.RawMessage(msg)})
	}
}

// CalculateMutationSuccessChance evaluates the risk of specialized gene-editing.
// PILLAR 6: Specialized Gene-Editing Scaling.
func (s *ClubService) CalculateMutationSuccessChance(l *Lobby, club *Club) float64 {
	// Base success rate for specialized gene-editing is 70%.
	chance := 0.70

	// PILLAR 1: Infrastructure Quality.
	// Mojo represents the club's prestige and technical investment.
	// Bonus: +1% per 50 Mojo (Max +20% at 1000 Mojo).
	mojoBonus := float64(club.Mojo) / 5000.0
	if mojoBonus > 0.20 {
		mojoBonus = 0.20
	}
	chance += mojoBonus

	// PILLAR 1: Organizational Capacity.
	// Each active staff member adds +2% (Max +10% for 5 members).
	staffBonus := float64(len(club.Staff)) * 0.02
	if staffBonus > 0.10 {
		staffBonus = 0.10
	}
	chance += staffBonus

	// PILLAR 3: Sabotage Impact.
	// Disrupted organization networks reduce procedure stability by 15%.
	if expiry, active := club.BuffExpirations["SABOTAGE"]; active && time.Now().Before(expiry) {
		chance -= 0.15
	}

	// PILLAR 1: Regional Governor Stability.
	// Regional Governors (2+ districts) receive a flat +5% stability bonus.
	if s.IsClubRegionalLocked(l, club) {
		chance += 0.05
	}

	// PILLAR 6: Tactical Infrastructure Buffs.
	// Staff Training provides a specialized boost to procedure stability.
	if expiry, active := club.BuffExpirations["STAFF_TRAINING"]; active && time.Now().Before(expiry) {
		// Use the value from the registry to ensure mathematical parity
		if item, ok := GlobalShopRegistry["staff_training"]; ok {
			chance += item.MutationSuccessModifier
		}
	}

	// Safety Clamping: No procedure is risk-free without insurance.
	if chance > 0.98 {
		chance = 0.98
	}
	if chance < 0.50 {
		chance = 0.50
	}

	return chance
}

// HandleRegionalSabotage allows elite players to disrupt coordinated alliance boosts in a target district.
// PILLAR 1: Regional Warfare.
func (s *ClubService) HandleRegionalSabotage(l *Lobby, env *Envelope) {
	var data struct {
		TargetClubID      string `json:"target_club_id"`
		TargetTerritoryID string `json:"target_territory_id"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	wallet, ok := l.wallets[env.FromID]
	if !ok {
		return
	}

	targetClub, exists := l.clubs[data.TargetClubID]
	if !exists {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Sabotage Failed: Target organization not found."}`)})
		return
	}

	// 1. Calculate Aggregated Warfare Cost (Integer Supremacy)
	// Base: 5,000 $VBV + 1,000 $VBV per allied district.
	alliedDistricts := len(targetClub.Territories)
	if targetClub.AlliedClubID != "" {
		if ally, ok := l.clubs[targetClub.AlliedClubID]; ok {
			alliedDistricts += len(ally.Territories)
		}
	}

	warfareFeeMicro := uint64(5000+1000*alliedDistricts) * 1000000

	// PILLAR 1: Reparation Multiplier.
	// Apply the same security resilience scaling to regional warfare protocols.
	multiplier := uint64(100)
	targetOwner := strings.ToLower(targetClub.OwnerWallet)
	if ownerStats, exists := l.leaderboard[targetOwner]; exists {
		multiplier += uint64((ownerStats.ReparationsReceivedCount / 5) * 10)
	}
	warfareFeeMicro = (warfareFeeMicro * multiplier) / 100

	// PILLAR 3: Career Path Influence. 'Saboteurs' receive a 1,500 $VBV discount on warfare protocols.
	stats := l.leaderboard[wallet]
	if stats.JobRole == "Saboteur" {
		const discountMicro = 1500 * 1000000
		if warfareFeeMicro > discountMicro {
			warfareFeeMicro -= discountMicro
		} else {
			warfareFeeMicro = 0
		}
	}

	if l.playerBalances[wallet] < warfareFeeMicro {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"❌ Sabotage Failed: Insufficient funds. Protocol requires %.0f $VBV."}`, float64(warfareFeeMicro)/1000000.0))})
		return
	}

	// 2. Execute Virtual Liability Shift
	l.playerBalances[wallet] -= warfareFeeMicro

	// 3. Routing: 70% to Faucet, 30% to Competitor Pool (Industrial Loop).
	// PILLAR 2: Unified Organizational Accounting (Token-Sink Router migration).
	if l.tokenSinkRouter != nil {
		matrix := RevenueSplitMatrix{FaucetShare: 0.70, ClubShare: 0.30, GovernanceShare: 0.0}
		_ = l.tokenSinkRouter.RouteCriminalTax("REGIONAL_WARFARE", warfareFeeMicro, matrix, 0, "GLOBAL")

			// PILLAR 2: UI Parity Sync.
			// Sync treasuries for all potential recipients from authoritative router nodes.
			for id, club := range l.clubs {
				numericID, _ := strconv.ParseUint(strings.TrimPrefix(id, "CLUB-"), 10, 64)
				if node, ok := l.tokenSinkRouter.ActiveClubs[numericID]; ok {
					club.TreasuryMicro = node.TreasuryBalance
				}
			}
	} else {
		// Fallback for non-router environments (Pillar 6)
		faucetCutMicro := (warfareFeeMicro * 70) / 100
		competitorCutMicro := warfareFeeMicro - faucetCutMicro
		l.faucetBalanceMicro += faucetCutMicro

		var otherGovs []*Club
		for _, c := range l.clubs {
			if c.ID != targetClub.ID && c.ID != targetClub.AlliedClubID && s.IsClubRegionalLocked(l, c) {
				otherGovs = append(otherGovs, c)
			}
		}
		if len(otherGovs) > 0 {
			shareMicro := competitorCutMicro / uint64(len(otherGovs))
			for _, g := range otherGovs {
				g.Treasury += float64(shareMicro) / 1000000.0
			}
		} else {
			l.faucetBalanceMicro += competitorCutMicro
		}
	}

	l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0
	l.applyDynamicScalingLocked()

	// 4. Apply Disruption: Disables Regional Boost and Coalition Defense
	expiry := time.Now().Add(2 * time.Hour)
	if targetClub.BuffExpirations == nil {
		targetClub.BuffExpirations = make(map[string]time.Time)
	}
	// Key format: REGIONAL_DISRUPTION_<TerritoryID>
	targetClub.BuffExpirations["DISRUPTION_"+data.TargetTerritoryID] = expiry
	targetClub.LastActivity = time.Now()

	l.logAdminAuditLocked("REGIONAL_SABOTAGE", wallet, fmt.Sprintf("Target: %s, District: %s, Cost: %d", targetClub.Name, data.TargetTerritoryID, warfareFeeMicro))

	// 5. Broadcast Blackout Alert
	blackoutAlert := fmt.Sprintf("📡 <b>NETWORK BLACKOUT:</b> Defensive coordination in %s has been disrupted! All Regional and Coalition boosts are OFFLINE for 2 hours.", data.TargetTerritoryID)
	alertPayload, _ := json.Marshal(map[string]string{"text": blackoutAlert})
	l.broadcast <- jsonListEnvelope("chat", alertPayload)

	// PILLAR 5: Reactive Atmosphere.
	// Dispatch high-intensity critical notification to trigger 'Warning_long.mp3' for affected governors.
	criticalPayload, _ := json.Marshal(map[string]string{
		"text": fmt.Sprintf("🔥 <b>CRITICAL ALERT:</b> A Regional Sabotage protocol in %s has severed your defensive coordination!", strings.ReplaceAll(strings.ToUpper(data.TargetTerritoryID), "_", " ")),
		"type": "critical",
	})
	criticalEnv := Envelope{Type: "admin_notification", FromID: "SERVER", Payload: criticalPayload}

	// Identify all affected governors (Target Owner and Alliance partner Owner)
	affectedOwners := make(map[string]bool)
	if targetClub.OwnerWallet != "" { affectedOwners[strings.ToLower(targetClub.OwnerWallet)] = true }
	if targetClub.AlliedClubID != "" {
		if ally, ok := l.clubs[targetClub.AlliedClubID]; ok && ally.OwnerWallet != "" {
			affectedOwners[strings.ToLower(ally.OwnerWallet)] = true
		}
	}

	for ownerWallet := range affectedOwners {
		if cid := l.getClientIDFromWalletLocked(ownerWallet); cid != "" {
			l.sendToClientLocked(cid, criticalEnv)
		}
	}

	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"🥷 <b>SABOTAGE SUCCESS:</b> Sector coordination disrupted. Defensive boosts are disabled."}`)})

	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// HandleCreateClub allows a player to found a new organization.
func (s *ClubService) HandleCreateClub(l *Lobby, env *Envelope) {
	var data struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		TerritoryID string `json:"territory_id"`
		TxID        string `json:"txid"`
		Network     string `json:"network"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}

	l.mutex.RLock()
	wallet, ok := l.wallets[env.FromID]
	vaultAddr := l.vaultAddress
	voiConfig := l.availableNetworks["Voi Mainnet"]
	l.mutex.RUnlock()

	if !ok {
		return
	}

	assetID := voiConfig.AssetID
	verifyNet := "Voi"
	if strings.ToUpper(data.Network) == "ALGO" || strings.HasPrefix(strings.ToUpper(data.Network), "ALGO") {
		assetID = l.avoiAssetID
		verifyNet = "Algorand"
	}

	// PILLAR 3: Dynamic Precision & Bound Verification.
	l.mutex.RLock()
	netCfg, hasCfg := l.availableNetworks[verifyNet+" Mainnet"]
	l.mutex.RUnlock()

	divisor := 1000000.0
	if hasCfg && netCfg.PowerDivisor > 0 {
		divisor = netCfg.PowerDivisor
	}

	// Include the target name in the prefix to bind payment to this specific organization.
	prefix := "FOUND_CLUB:" + data.Name + ":"

	l.mutex.Lock()
	if _, isUsed := l.registeredTxIDs[data.TxID]; isUsed && data.TxID != "" {
		l.mutex.Unlock()
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Club Foundry Error: Transaction already utilized."}`)})
		return
	}
	l.mutex.Unlock()

	verified, txTime, err := l.verifyBuyInTransaction(verifyNet, data.TxID, uint64(5000*divisor), assetID, wallet, vaultAddr, prefix)
	if err != nil || !verified {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Club Foundry Error: Payment verification failed."}`)})
		return
	}

	l.mutex.Lock()
	// Prevent TxID recycling
	l.registeredTxIDs[data.TxID] = time.Unix(txTime, 0)

	// INDUSTRIAL LOOP: Integer Supremacy.
	// Physically increment the vault total to reflect confirmed on-chain inflow.
	feeMicro := uint64(5000 * 1000000)
	l.faucetBalanceMicro += feeMicro
	l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0

	// PILLAR 2: Real-time Reconciliation.
	if l.tokenSinkRouter != nil && l.tokenSinkRouter.Audit != nil {
		_ = l.tokenSinkRouter.Audit.InterceptAndAudit("CLUB_FOUNDRY_FEE", feeMicro, feeMicro, 0, 0)
	}

	l.applyDynamicScalingLocked()

	clubID := fmt.Sprintf("CLUB-%d", time.Now().Unix())
	newClub := &Club{
		ID: clubID, Name: data.Name, OwnerWallet: wallet, Type: data.Type,
		Territories: []string{data.TerritoryID}, Commission: 0.05,
		Staff: make(map[string]string), Members: map[string]time.Time{strings.ToLower(wallet): time.Now()},
		Inventory:       make(map[string]int),
		ActiveBuffs:     make(map[string]string),
		Leases:          make(map[string]*Lease),
		Mojo:            0, // PILLAR 1: Start with neutral social standing
		BuffExpirations: make(map[string]time.Time),
		CreatedAt:       time.Now(), Jail: make(map[int]ServerCard), LastActivity: time.Now(),
	}
	newClub.Staff[strings.ToLower(wallet)] = "CEO"
	l.clubs[clubID] = newClub
	l.mutex.Unlock()

	l.logAdminAudit("CLUB_CREATED", wallet, fmt.Sprintf("Name: %s, ID: %s", data.Name, clubID))
	l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🏛️ Club '%s' successfully founded!"}`, data.Name))})

	// Trigger Global Sync to show the new club on the world map
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// HandleJoinClub allows a player to become a member of an existing club.
func (s *ClubService) HandleJoinClub(l *Lobby, env *Envelope) {
	var data struct {
		ClubID  string `json:"club_id"`
		TxID    string `json:"txid"`
		Network string `json:"network"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}

	l.mutex.RLock()
	wallet, ok := l.wallets[env.FromID]
	vaultAddr := l.vaultAddress
	voiConfig := l.availableNetworks["Voi Mainnet"]
	l.mutex.RUnlock()

	if !ok {
		return
	}

	joinFee := 500.0
	assetID := voiConfig.AssetID
	verifyNet := "Voi"

	if strings.EqualFold(data.Network, "ALGO") || strings.HasPrefix(strings.ToUpper(data.Network), "ALGO") {
		assetID = l.avoiAssetID
		verifyNet = "Algorand"
	}

	// PILLAR 1: Capital Presence (Task 921).
	// Apply a 10% 'Capital Entry Surcharge' if joining the organization controlling the Arena Center.
	l.mutex.RLock()
	targetClub, clubExists := l.clubs[data.ClubID]
	isCapitalOwner := false
	if clubExists {
		for _, t := range targetClub.Territories {
			if t == "arena_center" {
				isCapitalOwner = true
				break
			}
		}
	}
	l.mutex.RUnlock()

	if isCapitalOwner {
		joinFee = 550.0
	}

	// PILLAR 3: Dynamic Precision.
	l.mutex.RLock()
	netCfg, hasCfg := l.availableNetworks[verifyNet+" Mainnet"]
	l.mutex.RUnlock()

	divisor := 1000000.0 // Fallback to standard 6 decimals
	if hasCfg && netCfg.PowerDivisor > 0 {
		divisor = netCfg.PowerDivisor
	}

	// PILLAR 3: Bound Verification for Club Entry.
	// Include the Club ID in the prefix to prevent transaction note reuse across different clubs.
	prefix := "JOIN_CLUB:" + data.ClubID + ":"

	l.mutex.Lock()
	if _, isUsed := l.registeredTxIDs[data.TxID]; isUsed && data.TxID != "" {
		l.mutex.Unlock()
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Club Entry Error: Transaction already utilized."}`)})
		return
	}
	l.mutex.Unlock()

	verified, txTime, err := l.verifyBuyInTransaction(verifyNet, data.TxID, uint64(joinFee*divisor), assetID, wallet, vaultAddr, prefix)
	if err != nil || !verified {
		log.Printf("[CLUB] Join verification failed for %s on %s. Error: %v\n", wallet, verifyNet, err)
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Club Entry Error: Payment verification failed."}`)})
		return
	}

	l.mutex.Lock()
	// PILLAR 3: Identity Persistence.
	l.ensurePlayerStatsMapsInitialized(wallet)

	// Prevent TxID recycling
	l.registeredTxIDs[data.TxID] = time.Unix(txTime, 0)

	// INDUSTRIAL LOOP: Update physical vault total and trigger reactive scaling.
	// 500 $VBV entered the vault; 250 is allocated to Club Treasury liability.
	l.faucetBalance += joinFee

	if club, exists := l.clubs[data.ClubID]; exists {
		if club.Members == nil {
			club.Members = make(map[string]time.Time)
		}

		// Prevent double-joining exploit
		if _, isMember := club.Members[strings.ToLower(wallet)]; isMember {
			l.applyDynamicScalingLocked()
			l.mutex.Unlock()
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"⚠️ You are already a member of this club."}`)})
			return
		}

		// PILLAR 1: Reputation & Mojo Gate. Elite clubs can set minimum requirements for joining.
		// These requirements scale with the club's Mojo, reflecting its elite status.
		minReputationRequired := 0
		minMojoRequired := 0
		if club.Mojo >= 500 { // Elite status starts at 500 Mojo
			minReputationRequired = club.Mojo / 5 // e.g., 500 Mojo -> 100 Rep required
			minMojoRequired = club.Mojo / 10      // e.g., 500 Mojo -> 50 Mojo required
		}

		playerStats := l.leaderboard[wallet]
		// PILLAR 1: Real-time Standing Verification. Ensure we check the most current Reputation.
		playerStats.Reputation = l.CalculateReputation(playerStats) // This calls CalculateReputation, not PlayerStats.GetEffectiveMojo
		l.leaderboard[wallet] = playerStats

		if minReputationRequired > 0 && (playerStats.Reputation < minReputationRequired || l.playerService.GetEffectiveMojo(playerStats) < minMojoRequired) {
			l.applyDynamicScalingLocked()
			l.mutex.Unlock()
			msg := fmt.Sprintf(`{"text":"❌ Club Entry Failed: Elite club %s requires %d Reputation and %d Mojo social standing to join."}`, club.Name, minReputationRequired, minMojoRequired)
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(msg)})
			return
		}

		club.Members[strings.ToLower(wallet)] = time.Now()
		club.LastActivity = time.Now()

		// PILLAR 2: Unified Organizational Accounting.
		// Update the club's treasury in the authoritative router node to ensure 
		// accurate systemic liability reporting.
		if l.tokenSinkRouter != nil {
			l.tokenSinkRouter.Mu.Lock()
			numericID, _ := strconv.ParseUint(strings.TrimPrefix(club.ID, "CLUB-"), 10, 64)
			if node, ok := l.tokenSinkRouter.ActiveClubs[numericID]; ok {
				node.TreasuryBalance += 250 * 1000000
				club.TreasuryMicro = node.TreasuryBalance
				club.Treasury = float64(node.TreasuryBalance) / 1000000.0
			}
			l.tokenSinkRouter.Mu.Unlock()
		} else {
			club.Treasury += 250.0
		}

		// PILLAR 1: Mojo Gain for Club Entry revenue.
		mojoGain := s.CalculateMojoGain(l, club, "REVENUE", 250.0)
		club.Mojo += mojoGain
		l.achievementService.CheckMojoSurgeAchievementLocked(l, club.ID)

		// PILLAR 2: Ledger Integrity. Update scaling to reflect the new treasury liability.
		l.applyDynamicScalingLocked()

		l.mutex.Unlock()
		l.logAdminAudit("CLUB_JOIN", wallet, fmt.Sprintf("Club: %s, CapitalSurcharge: %v", data.ClubID, isCapitalOwner))
		
		welcomeMsg := fmt.Sprintf(`{"text":"🤝 Welcome to %s!"}`, club.Name)
		if isCapitalOwner {
			welcomeMsg = fmt.Sprintf(`{"text":"🏛️ <b>CAPITAL AFFILIATION:</b> Welcome to %s! (Included 10%% Surcharge)"}`, club.Name)
		}
		
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(welcomeMsg)})

		// Sync UI to update membership lists and treasury balances
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	} else {
		// Fallback for organizational lookup failure: update scaling for the added faucet entry.
		l.applyDynamicScalingLocked()
		l.mutex.Unlock()
	}
}

// HandlePurchaseTerritory allows a club to expand its influence.
func (s *ClubService) HandlePurchaseTerritory(l *Lobby, env *Envelope) {
	var data struct {
		ClubID      string `json:"club_id"`
		TerritoryID string `json:"territory_id"`
		TxID        string `json:"txid"`
		Network     string `json:"network"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}

	l.mutex.RLock()
	ownerWallet, ok := l.wallets[env.FromID]
	vaultAddr := l.vaultAddress
	voiConfig, voiOk := l.availableNetworks["Voi Mainnet"]
	l.mutex.RUnlock()

	if !voiOk {
		return
	}

	if !ok {
		return
	}

	l.mutex.RLock()
	club, exists := l.clubs[data.ClubID]
	if !exists || !strings.EqualFold(club.OwnerWallet, ownerWallet) {
		l.mutex.RUnlock()
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Territory Purchase Failed: Unauthorized or Club not found."}`)})
		return
	}

	for _, existingClub := range l.clubs {
		for _, t := range existingClub.Territories {
			if strings.EqualFold(t, data.TerritoryID) {
				l.mutex.RUnlock()
				l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Territory Purchase Failed: District already claimed."}`)})
				return
			}
		}
	}
	l.mutex.RUnlock()

	purchaseCostMicro := uint64(2500 * 1000000)
	assetID := voiConfig.AssetID
	verifyNet := "Voi"

	if strings.EqualFold(data.Network, "ALGO") || strings.HasPrefix(strings.ToUpper(data.Network), "ALGO") {
		assetID = l.avoiAssetID
		verifyNet = "Algorand"
	}

	// PILLAR 3: Dynamic Precision.
	// Fetch specific network config to get the correct micro-unit divisor for the purchase asset.
	l.mutex.RLock()
	netCfg, hasCfg := l.availableNetworks[verifyNet+" Mainnet"]
	l.mutex.RUnlock()

	divisor := 1000000.0 // Fallback to standard 6 decimals (VBV/AVoi)
	if hasCfg && netCfg.PowerDivisor > 0 {
		divisor = netCfg.PowerDivisor
	}

	// PILLAR 3: Bound Verification. Bind to specific Club and District to prevent note reuse.
	prefix := "CLAIM_DISTRICT:" + data.ClubID + ":" + data.TerritoryID + ":"

	l.mutex.Lock()
	if _, isUsed := l.registeredTxIDs[data.TxID]; isUsed && data.TxID != "" {
		l.mutex.Unlock()
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Territory Purchase Failed: Transaction already utilized."}`)})
		return
	}
	l.mutex.Unlock()

	verified, txTime, err := l.verifyBuyInTransaction(verifyNet, data.TxID, uint64(2500.0*divisor), assetID, ownerWallet, vaultAddr, prefix)
	if err != nil || !verified {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Territory Purchase Failed: Payment verification failed."}`)})
		return
	}

	l.mutex.Lock()
	// RE-VERIFY: Ensure territory was not claimed while we were verifying the transaction
	for _, existingClub := range l.clubs {
		for _, t := range existingClub.Territories {
			if strings.EqualFold(t, data.TerritoryID) {
				l.applyDynamicScalingLocked()
				l.mutex.Unlock()
				l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Territory Purchase Failed: District claimed by another entity during verification."}`)})
				return
			}
		}
	}

	// Prevent TxID recycling
	l.registeredTxIDs[data.TxID] = time.Unix(txTime, 0)

	// PILLAR 2: Industrial Loop (Token-Sink Router migration).
	// Atomic redistribution of the 2,500 $VBV fee: 95% Faucet, 5% Regional Governors.
	if l.tokenSinkRouter != nil {
		matrix := RevenueSplitMatrix{FaucetShare: 0.95, ClubShare: 0.0, GovernanceShare: 0.05}
		_ = l.tokenSinkRouter.RouteCriminalTax("TERRITORY_PURCHASE", purchaseCostMicro, matrix, 0, data.TerritoryID)

		// Sync float balance with authoritative micro-unit total
		l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0
	} else {
		l.faucetBalance += 2500.0
	}

	club, exists = l.clubs[data.ClubID]
	if !exists || !strings.EqualFold(club.OwnerWallet, ownerWallet) {
		l.applyDynamicScalingLocked()
		l.mutex.Unlock()
		return
	}

	club.Territories = append(club.Territories, data.TerritoryID)

	// PILLAR 1: Immediate Regional Role & Achievement Integration.
	// If this is the second district, trigger Governor status immediately to ensure atomic UI sync.
	if s.IsClubRegionalLocked(l, club) {
		if club.RegionName == "" {
			club.RegionName = "Governor"
		}
		l.achievementService.UnlockAchievementLocked(l, strings.ToLower(club.OwnerWallet), "GOVERNOR")
	}

	// PILLAR 2: Ledger Integrity. Re-calculate scaling for faucet entry and reserve shifts.
	l.applyDynamicScalingLocked()

	l.clubs[data.ClubID] = club
	club.LastActivity = time.Now()
	l.mutex.Unlock()

	l.logAdminAudit("TERRITORY_PURCHASED", ownerWallet, fmt.Sprintf("Club: %s (%s), Territory: %s", club.Name, club.ID, data.TerritoryID))
	l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🗺️ Club '%s' has acquired %s!"}`, club.Name, data.TerritoryID))})

	// Trigger Global Sync to update territory ownership visuals
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// HandleRestockInventory allows authorized staff to restock items in the club shop.
func (s *ClubService) HandleRestockInventory(l *Lobby, env *Envelope) {
	var data struct {
		ClubID   string `json:"club_id"`
		ItemID   string `json:"item_id"`
		Quantity int    `json:"quantity"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	ownerWallet, ok := l.wallets[env.FromID]
	if !ok {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Restock Failed: Sender not connected."}`)})
		return
	}
	club, exists := l.clubs[data.ClubID]
	if !exists {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Restock Failed: Club not found."}`)})
		return
	}

	// PILLAR 1: Career Role Validation.
	// Ensure the 'Manager' role check is robust and accounts for normalization.
	isOwner := strings.EqualFold(club.OwnerWallet, ownerWallet)
	role := club.Staff[strings.ToLower(ownerWallet)]

	// CEO/Owner or Manager can restock.
	canRestock := isOwner || strings.EqualFold(role, "Manager") || strings.EqualFold(role, "CEO")
	if !canRestock {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Restock Failed: Unauthorized."}`)})
		return
	}

	item, itemExists := GlobalShopRegistry[data.ItemID]
	if !itemExists {
		return
	}

	// CAPACITY GUARD: Limit total items per club to prevent state bloat (Max 1000 items)
	const maxClubInventory = 1000
	currentStock := 0
	for _, qty := range club.Inventory {
		currentStock += qty
	}

	if currentStock+data.Quantity > maxClubInventory {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"❌ Restock Failed: Inventory capacity reached (%d/%d)."}`, currentStock, maxClubInventory))})
		return
	}

	// Units: Both item.Price and club.Treasury are in base $VBV units.
	// Hardening: We use micro-unit math for the cost calculation to ensure absolute precision.
	// PILLAR 2: Precision Hardening. Round to nearest micro-unit.
	itemPriceMicro := uint64(item.Price*1000000 + 0.5)
	totalCostMicro := itemPriceMicro * uint64(data.Quantity)
	totalCostBase := float64(totalCostMicro) / 1000000.0

	// Convert Treasury to micro-units for comparison and subtraction to prevent dust leaks.
	treasuryMicro := uint64(club.Treasury*1000000 + 0.5)

	if treasuryMicro < totalCostMicro {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"❌ Restock Failed: Insufficient Treasury funds. Need %.2f $VBV."}`, totalCostBase))})
		return
	}

	// Subtract in micro-units and convert back to ensure mathematical integrity.
	newTreasuryMicro := treasuryMicro - totalCostMicro
	club.Treasury = float64(newTreasuryMicro) / 1000000.0

	// Ensure map is initialized even if old data exists
	if club.Inventory == nil {
		club.Inventory = make(map[string]int)
	}

	club.LastActivity = time.Now()
	club.Inventory[data.ItemID] += data.Quantity

	// PILLAR 2: Industrial Loop.
	// Restocking reduces club reserves, satisfy organizational debt via integer math.
	if l.tokenSinkRouter != nil {
		l.tokenSinkRouter.Mu.Lock()
		numericID, _ := strconv.ParseUint(strings.TrimPrefix(club.ID, "CLUB-"), 10, 64)
		if node, ok := l.tokenSinkRouter.ActiveClubs[numericID]; ok {
			node.TreasuryBalance = newTreasuryMicro
		}
		l.tokenSinkRouter.Mu.Unlock()
	}
	l.applyDynamicScalingLocked()

	l.logAdminAuditLocked("CLUB_RESTOCK", ownerWallet, fmt.Sprintf("Club: %s, Item: %s, Qty: %d, Cost: %.2f", club.Name, data.ItemID, data.Quantity, totalCostBase))
	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"📦 <b>RESTOCK COMPLETE:</b> Added %d units of %s to inventory."}`, data.Quantity, item.Name))})

	// Trigger Global Sync to update UI (faucet/treasury/inventory)
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// HandlePurchaseItem processes a player's request to buy an item from a district shop.
// PILLAR 2: Ledger Integrity. Moves the logic from lobby_manager.go to enforce modular authority.
func (s *ClubService) HandlePurchaseItem(l *Lobby, env *Envelope) {
	var data struct {
		ItemID      string `json:"item_id"`
		TerritoryID string `json:"territory_id"`
		Price       uint64 `json:"price"` // Micro-VBV
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	wallet, ok := l.wallets[env.FromID]
	if !ok {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Purchase Failed: Wallet not registered."}`)})
		return
	}

	l.ensurePlayerStatsMapsInitialized(wallet)
	stats := l.leaderboard[wallet]

	// FINANCIAL VERIFICATION: Ensure player has enough virtual balance to cover the price
	if l.playerBalances[wallet] < data.Price {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Purchase Failed: Insufficient reward balance."}`)})
		return
	}

	// 1. Find the Club managing this territory
	var targetClub *Club
	for _, club := range l.clubs {
		for _, t := range club.Territories {
			if strings.EqualFold(t, data.TerritoryID) {
				targetClub = club
				break
			}
		}
		if targetClub != nil {
			break
		}
	}

	// Authorization & Stock Check
	if targetClub == nil || targetClub.Inventory[data.ItemID] <= 0 {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Purchase Failed: Item out of stock or district unclaimed."}`)})
		return
	}

	// PILLAR 1: Professional Prestige & Industrial Unlocks.
	item, itemExists := GlobalShopRegistry[data.ItemID]
	if !itemExists {
		return
	}

	// Role Check: Career role must match item requirement (e.g. Sentry Turrets require Security)
	if item.RequiredRole != "" && stats.JobRole != item.RequiredRole {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"❌ Purchase Failed: Career role '%s' required."}`, item.RequiredRole))})
		return
	}

	// Mojo Check: Club influence must meet item threshold
	if targetClub.Mojo < item.RequiredMojo {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"❌ Purchase Failed: Club Mojo too low (%d/%d)."}`, targetClub.Mojo, item.RequiredMojo))})
		return
	}

	// Regional Governance Check: Master Tier items require 2+ districts
	if item.IsMasterTier && !s.IsClubRegionalLocked(l, targetClub) {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Purchase Failed: This is a Master Tier item. Requires Regional Governor status."}`)})
		return
	}

	// 2. Fulfillment: Deduct from Player, Deduct from Club Stock
	l.playerBalances[wallet] -= data.Price
	targetClub.Inventory[data.ItemID]--
	targetClub.LastActivity = time.Now()

	// Apply item-specific Mojo bonus to the club
	targetClub.Mojo += item.MojoBonus
	l.checkMojoSurgeAchievementLocked(targetClub.ID)

	if stats.Inventory == nil {
		stats.Inventory = make(map[string]int)
	}
	stats.Inventory[data.ItemID]++

	// Update reputation immediately to reflect the shift in virtual liabilities
	stats.Reputation = l.CalculateReputation(stats)
	l.leaderboard[wallet] = stats

	// 3. Process Revenue (Industrial Loop)
	revenueToDistribute := data.Price

	// PILLAR 3: Smuggler Facilitation Fee (Section 13.A).
	// If the organization owner is a Smuggler, 5% is siphoned to Faucet.
	if l.playerService.GetHegemonyPath(targetClub.Staff[targetClub.OwnerWallet]) == "UNDERWORLD" && 
	   targetClub.Staff[targetClub.OwnerWallet] == "Smuggler" {
		smuggleFeeMicro := (data.Price * 5) / 100
		revenueToDistribute -= smuggleFeeMicro
		if l.tokenSinkRouter != nil {
			matrix := RevenueSplitMatrix{FaucetShare: 1.0, ClubShare: 0.0, GovernanceShare: 0.0}
			_ = l.tokenSinkRouter.RouteCriminalTax("SMUGGLER_FEE", smuggleFeeMicro, matrix, 0, "")
			l.logAdminAuditLocked("SMUGGLER_FEE_COLLECTED", targetClub.ID, fmt.Sprintf("Amt: %d", smuggleFeeMicro))
		}
	}

	if item.IsMasterTier {
		// PILLAR 2: Luxury Tax Logic.
		// Divert 1% of the gross price from Master Tier items to fund the Global Faucet.
		luxuryTaxMicro := uint64(float64(data.Price)*0.01 + 0.5)
		l.faucetBalanceMicro += luxuryTaxMicro
		l.LuxuryTaxTotal += luxuryTaxMicro
		l.LuxuryTaxCount++
		l.achievementService.CheckTaxMilestoneAchievementLocked(l)
		l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0
		
		revenueToDistribute -= luxuryTaxMicro
		l.logAdminAuditLocked("LUXURY_TAX_COLLECTED", wallet, fmt.Sprintf("Item: %s, Tax: %.2f", data.ItemID, float64(luxuryTaxMicro)/1000000.0))
	}

	s.DistributeShopRevenueLocked(l, data.TerritoryID, revenueToDistribute, data.ItemID)

	l.logAdminAuditLocked("ITEM_PURCHASE", wallet, fmt.Sprintf("Item: %s, Territory: %s, Cost: %.2f", data.ItemID, data.TerritoryID, float64(data.Price)/1000000.0))

	notification := fmt.Sprintf(`{"text":"📦 <b>PURCHASE COMPLETE:</b> %s added to inventory."}`, item.Name)
	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(notification)})

	// Sync back to client to update local UI inventory and balances
	msg := l.getLobbyUpdateMsgLocked()
	go func() { l.broadcast <- msg }()
}

// HandleAllianceInvite allows a club owner to propose a partnership to another club.
func (s *ClubService) HandleAllianceInvite(l *Lobby, env *Envelope) {
	var data struct {
		MyClubID     string `json:"my_club_id"`
		TargetClubID string `json:"target_club_id"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}
	l.mutex.Lock()
	defer l.mutex.Unlock()

	wallet, ok := l.wallets[env.FromID]
	if !ok {
		return
	}

	clubA, existsA := l.clubs[data.MyClubID]

	// PILLAR 1: Alliance Decline Logic.
	// If TargetClubID is empty, it signifies a decline of an incoming invitation.
	if data.TargetClubID == "" {
		if !existsA || !strings.EqualFold(clubA.OwnerWallet, wallet) {
			return
		} // Unauthorized decline attempt

		// PILLAR 1: Alliance Decline Notification.
		// Notify the sender that their proposal was rejected before clearing the marker.
		requesterID := clubA.AllianceInviteID
		if requesterID != "" {
			if requesterClub, exists := l.clubs[requesterID]; exists {
				ownerCID := l.getClientIDFromWalletLocked(requesterClub.OwnerWallet)
				if ownerCID != "" {
					l.sendToClientLocked(ownerCID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🤝 <b>ALLIANCE DECLINED:</b> %s has declined your partnership proposal."}`, escapeHTML(clubA.Name)))})
				}
			}
		}

		clubA.AllianceInviteID = "" // Clear the pending invitation
		l.logAdminAuditLocked("ALLIANCE_DECLINED", wallet, fmt.Sprintf("Club: %s declined invite.", clubA.Name))
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"🤝 Alliance proposal declined."}`)})
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		return
	}

	// Proceed with alliance invitation logic if TargetClubID is not empty
	clubB, existsB := l.clubs[data.TargetClubID]

	if !existsA || !existsB || !strings.EqualFold(clubA.OwnerWallet, wallet) {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Alliance Failed: Unauthorized or Club not found."}`)})
		return
	}

	if clubA.AlliedClubID != "" || clubB.AlliedClubID != "" {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Alliance Failed: One of these clubs is already in an alliance."}`)})
		return
	}

	clubB.AllianceInviteID = clubA.ID
	clubB.AllianceInviteExpiresAt = time.Now().Add(24 * time.Hour) // PILLAR 1: 24h Proposal Window
	l.logAdminAuditLocked("ALLIANCE_INVITE", wallet, fmt.Sprintf("From: %s, To: %s", clubA.Name, clubB.Name))

	// Notify Target Owner
	targetOwnerCID := l.getClientIDFromWalletLocked(clubB.OwnerWallet)
	if targetOwnerCID != "" {
		l.sendToClientLocked(targetOwnerCID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🤝 <b>ALLIANCE PROPOSAL:</b> %s has proposed a regional alliance!"}`, escapeHTML(clubA.Name)))})
	}
	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"📡 <b>INVITATION SENT:</b> Awaiting response from alliance target."}`)})
}

// HandleAllianceAccept allows a club owner to finalize a partnership.
func (s *ClubService) HandleAllianceAccept(l *Lobby, env *Envelope) {
	var data struct {
		MyClubID string `json:"my_club_id"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	wallet, _ := l.wallets[env.FromID]
	clubB, existsB := l.clubs[data.MyClubID]
	if !existsB || !strings.EqualFold(clubB.OwnerWallet, wallet) || clubB.AllianceInviteID == "" {
		return
	}

	clubA, existsA := l.clubs[clubB.AllianceInviteID]
	if !existsA {
		clubB.AllianceInviteID = ""
		return
	}

	// Execute Alliance
	clubA.AlliedClubID = clubB.ID
	clubB.AlliedClubID = clubA.ID
	clubB.AllianceInviteID = ""

	// PILLAR 1: Region Synergy check.
	// If the combined territories reach 2, both owners get Governor status.
	if s.IsClubRegionalLocked(l, clubA) {
		if clubA.RegionName == "" {
			clubA.RegionName = "Governor"
		}
		if clubB.RegionName == "" {
			clubB.RegionName = "Governor"
		}
		l.achievementService.UnlockAchievementLocked(l, strings.ToLower(clubA.OwnerWallet), "GOVERNOR")
		l.achievementService.UnlockAchievementLocked(l, strings.ToLower(clubB.OwnerWallet), "GOVERNOR")
	}

	l.logAdminAuditLocked("ALLIANCE_FORMED", wallet, fmt.Sprintf("Clubs: %s & %s", clubA.Name, clubB.Name))
	l.broadcastToAdmins(fmt.Sprintf("🤝 <b>REGIONAL ALLIANCE:</b> %s and %s have combined their influence.", clubA.Name, clubB.Name))

	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// HandleAllianceDissolve allows a club owner to terminate their partnership.
func (s *ClubService) HandleAllianceDissolve(l *Lobby, env *Envelope) {
	var data struct {
		MyClubID string `json:"my_club_id"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	wallet, _ := l.wallets[env.FromID]
	clubA, existsA := l.clubs[data.MyClubID]
	if !existsA || !strings.EqualFold(clubA.OwnerWallet, wallet) || clubA.AlliedClubID == "" {
		return
	}

	clubB, existsB := l.clubs[clubA.AlliedClubID]
	clubA.AlliedClubID = ""
	if existsB {
		clubB.AlliedClubID = ""
	}

	// PILLAR 1: Regional Governor Update.
	// After dissolution, re-evaluate Governor status for both clubs independently.
	// If an organization no longer controls 2+ districts alone, the title is stripped.
	if !s.IsClubRegionalLocked(l, clubA) {
		clubA.RegionName = ""
	}
	if existsB && !s.IsClubRegionalLocked(l, clubB) {
		clubB.RegionName = ""
	}

	// PILLAR 1: Conflict Cleanup.
	// Clear any active coordinated-defense disruptions from both organizations.
	// Since the alliance is severed, coordinated blackouts are no longer valid.
	for key := range clubA.BuffExpirations {
		if strings.HasPrefix(key, "DISRUPTION_") {
			delete(clubA.BuffExpirations, key)
		}
	}

	if existsB {
		for key := range clubB.BuffExpirations {
			if strings.HasPrefix(key, "DISRUPTION_") {
				delete(clubB.BuffExpirations, key)
			}
		}
	}

	// PILLAR 1: Social Notification.
	// Notify both owners of the dissolution to ensure transparency in the Trust Layer.
	ownerACID := l.getClientIDFromWalletLocked(clubA.OwnerWallet)
	if ownerACID != "" {
		l.sendToClientLocked(ownerACID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"🤝 <b>ALLIANCE DISSOLVED:</b> You have terminated your partnership."}`)})
	}
	if existsB {
		ownerBCID := l.getClientIDFromWalletLocked(clubB.OwnerWallet)
		if ownerBCID != "" {
			l.sendToClientLocked(ownerBCID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🤝 <b>ALLIANCE DISSOLVED:</b> %s has terminated the regional partnership."}`, escapeHTML(clubA.Name)))})
		}
	}

	l.logAdminAuditLocked("ALLIANCE_DISSOLVED", wallet, fmt.Sprintf("Triggered by %s", clubA.Name))
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// IsClubRegionalLocked checks if a club (or its alliance) owns 2 or more territories.
func (s *ClubService) IsClubRegionalLocked(l *Lobby, club *Club) bool {
	count := len(club.Territories)
	if club.AlliedClubID != "" {
		if allied, ok := l.clubs[club.AlliedClubID]; ok {
			count += len(allied.Territories)
		}
	}
	return count >= 2
}

// IsPlayerAffiliatedWithClubLocked checks if a player is a member or owner of a club or its allied organization.
func (s *ClubService) IsPlayerAffiliatedWithClubLocked(l *Lobby, wallet string, club *Club) bool {
	lowerW := strings.ToLower(wallet)

	// Direct affiliation: Owner, Staff, or Member
	if strings.EqualFold(club.OwnerWallet, wallet) {
		return true
	}
	if _, isStaff := club.Staff[lowerW]; isStaff {
		return true
	}
	if _, isMember := club.Members[lowerW]; isMember {
		return true
	}

	// Allied affiliation
	if club.AlliedClubID != "" {
		if allied, ok := l.clubs[club.AlliedClubID]; ok {
			if strings.EqualFold(allied.OwnerWallet, wallet) {
				return true
			}
			if _, isStaff := allied.Staff[lowerW]; isStaff {
				return true
			}
			if _, isMember := allied.Members[lowerW]; isMember {
				return true
			}
		}
	}

	return false
}

// DistributeShopRevenue handles payout to club treasuries based on shop turnover.
func (s *ClubService) DistributeShopRevenue(l *Lobby, territoryID string, amountMicro uint64, itemID string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	s.DistributeShopRevenueLocked(l, territoryID, amountMicro, itemID)
}

// DistributeShopRevenueLocked handles payout to club treasuries with Regional Taxation.
// PILLAR 2: Industrial Seal. Ensures any virtual balance not allocated as commission
// returns to the Faucet pool to maintain mathematical circularity.
func (s *ClubService) DistributeShopRevenueLocked(l *Lobby, territoryID string, amountMicro uint64, itemID string) {
	now := time.Now()

	// 1. Identify the specific club owning this territory
	var owningClub *Club
	for _, club := range l.clubs {
		for _, t := range club.Territories {
			if strings.EqualFold(t, territoryID) {
				owningClub = club
				break
			}
		}
		if owningClub != nil {
			break
		}
	}
	if owningClub == nil {
		return
	}

	// 2. Identify all Regional Governors (Clubs owning 2+ territories)
	var governors []*Club
	for _, club := range l.clubs {
		if s.IsClubRegionalLocked(l, club) {
			governors = append(governors, club)
		}
	}

	// 3. Calculate proportions for the Token-Sink Router
	rate := owningClub.Commission
	// PILLAR 3: Justice Commissioner Influence.
	// If a 'PRO_SOCIAL_COMMISSION' buff is active, override the club's commission rate.
	if expiry, exists := owningClub.BuffExpirations["PRO_SOCIAL_COMMISSION"]; exists && now.Before(expiry) {
		rate = 0.01 // Fixed 1% commission rate
		l.logAdminAuditLocked("PRO_SOCIAL_COMMISSION_ACTIVE", owningClub.ID, fmt.Sprintf("Commission rate overridden to 1%% for %s", owningClub.Name))
	}
	if rate < 0.05 { rate = 0.05 }
	if rate > 0.50 { rate = 0.50 }

	taxRate := 0.05 // Default 5% Governor Tax on the commission
	if l.tokenSinkRouter != nil {
		l.tokenSinkRouter.Mu.RLock()
		if metric, exists := l.tokenSinkRouter.RegionalDistricts[territoryID]; exists && metric.CustomTaxRate > 0 {
			taxRate = metric.CustomTaxRate
		}
		l.tokenSinkRouter.Mu.RUnlock()
	}

	// PILLAR 2: Unified Organizational Accounting.
	// Migrate to the TokenSinkRouter to enforce the Industrial Seal and ensure
	// organizational revenue is correctly vetted and audited.
	if l.tokenSinkRouter != nil {
		matrix := RevenueSplitMatrix{
			FaucetShare:     1.0 - rate,
			ClubShare:       rate * (1.0 - taxRate),
			GovernanceShare: rate * taxRate,
		}

		// PILLAR 1 & 2: Dividend Diversion.
		// Deduct 0.5% of the gross price from the ClubShare allocation.
		dividendAmt := (amountMicro * 5) / 1000
		ownerWallet := strings.ToLower(owningClub.OwnerWallet)
		node := l.getOrCreateMarketNodeLocked(ownerWallet)

		// PILLAR 3: Dividend Freeze Check.
		// If frozen by a Tax Auditor, dividends are seized as a Regulatory Fine.
		if node.IsDividendFrozen {
			if l.tokenSinkRouter != nil && l.tokenSinkRouter.GlobalFaucetPool != nil {
				*l.tokenSinkRouter.GlobalFaucetPool += dividendAmt
				l.logAdminAuditLocked("DIVIDEND_SEIZED", ownerWallet, fmt.Sprintf("Reg. Fine: %d micro-VBV", dividendAmt))
			}
		} else if node.TotalSharesIssued > 0 {
			node.DividendPoolMicro += dividendAmt
			// Update Cumulative Yield using fixed-point math (precision scaling by 1e12)
			// to ensure tiny dividends are tracked correctly across high share counts.
			node.CumulativeYieldPerShare += (dividendAmt * 1000000000000) / node.TotalSharesIssued
		} else {
			// Fallback: If no shares issued, return to Faucet.
			if l.tokenSinkRouter != nil && l.tokenSinkRouter.GlobalFaucetPool != nil {
				*l.tokenSinkRouter.GlobalFaucetPool += dividendAmt
			}
		}

		numericID, _ := strconv.ParseUint(strings.TrimPrefix(owningClub.ID, "CLUB-"), 10, 64)
		_ = l.tokenSinkRouter.RouteCriminalTax("SHOP_REVENUE", amountMicro, matrix, numericID, territoryID) // Governance share to territory owner

		// PILLAR 2: UI Parity Sync. 
		if node, ok := l.tokenSinkRouter.ActiveClubs[numericID]; ok {
			owningClub.TreasuryMicro = node.TreasuryBalance
			owningClub.Treasury = float64(node.TreasuryBalance) / 1000000.0
		}

		// Sync float balance with authoritative micro-unit total
		l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0
		l.applyDynamicScalingLocked()
	}

	// 4. Organizational Progression (Mojo Gain)
	mojoGain := s.CalculateMojoGain(l, owningClub, "REVENUE", float64(amountMicro)/1000000.0)
	owningClub.Mojo += mojoGain
	owningClub.LastActivity = now
	l.achievementService.CheckMojoSurgeAchievementLocked(l, owningClub.ID)

	// PILLAR 1: Localized Governor Mojo Gain.
	// Mojo should only be awarded to the Regional Governor who actually received the tax.
	if taxRate > 0 {
		govClub := l.getClubByTerritoryID(territoryID)
		if govClub != nil && s.IsClubRegionalLocked(l, govClub) {
			govTaxMicro := uint64(float64(amountMicro)*rate*taxRate + 0.5)
			govMojo := s.CalculateMojoGain(l, govClub, "REVENUE", float64(govTaxMicro)/1000000.0)
			govClub.Mojo += govMojo
			govClub.LastActivity = now
			l.achievementService.CheckMojoSurgeAchievementLocked(l, govClub.ID)
		}
	}
}

// DistributeTournamentKickback handles the 1-5% payout to clubs based on member tournament fees.
func (s *ClubService) DistributeTournamentKickback(l *Lobby, playerWallet string, feeMicro uint64, registrationTime time.Time) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	s.DistributeTournamentKickbackLocked(l, playerWallet, feeMicro, registrationTime)
}

// DistributeTournamentKickbackLocked handles the kickback distribution assuming the mutex is already held.
// PILLAR 2: Unified Organizational Accounting.
func (s *ClubService) DistributeTournamentKickbackLocked(l *Lobby, playerWallet string, feeMicro uint64, registrationTime time.Time) {
	lowerWallet := strings.ToLower(playerWallet)

	for _, club := range l.clubs {
		joinedAt, isMember := club.Members[lowerWallet]

		if isMember && joinedAt.Before(registrationTime) {
			// Base 1% kickback, scales with Club Mojo up to 5%
			rate := 0.01 + (float64(club.Mojo)/1000.0)*0.04
			if rate > 0.05 { rate = 0.05 }

			kickbackMicro := uint64(float64(feeMicro)*rate + 0.5)
			if kickbackMicro == 0 { return }

			// PILLAR 2: Industrial Loop (Liability Shift).
			// Move funds from the unreserved Faucet pool to the organizational treasury.
			if l.faucetBalanceMicro >= kickbackMicro && l.tokenSinkRouter != nil {
				l.faucetBalanceMicro -= kickbackMicro
				l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0

				matrix := RevenueSplitMatrix{FaucetShare: 0.0, ClubShare: 1.0, GovernanceShare: 0.0}
				numericID, _ := strconv.ParseUint(strings.TrimPrefix(club.ID, "CLUB-"), 10, 64)
				_ = l.tokenSinkRouter.RouteCriminalTax("TOURN_KICKBACK", kickbackMicro, matrix, numericID, "arena_center")

				if node, ok := l.tokenSinkRouter.ActiveClubs[numericID]; ok {
					club.TreasuryMicro = node.TreasuryBalance
				}
				l.applyDynamicScalingLocked()
			} else if l.tokenSinkRouter == nil {
				club.Treasury += float64(kickbackMicro) / 1000000.0
			}

			club.LastActivity = time.Now()
			log.Printf("[REVENUE] Club %s received %.2f $VBV kickback from %s registration.\n",
				club.Name, float64(kickbackMicro)/1000000.0, playerWallet)

			return
		}
	}
}

// CalculateMojoGain computes the Mojo increase for a club based on economic or defensive events.
// It weights the gain based on territory ownership and Regional Governor status.
func (s *ClubService) CalculateMojoGain(l *Lobby, club *Club, reason string, value float64) int {
	gain := 0
	switch reason {
	case "REVENUE":
		// Earn 1 Mojo for every 50 $VBV in turnover (Value is in base units)
		gain = int(value / 50.0)
		// PILLAR 1: Anti-Whale Guard.
		if gain > 20 {
			gain = 20 // Cap base revenue gain per transaction
		}
	case "DEFENSE":
		// Successful heist defense yields a flat Mojo boost
		gain = 15
	case "JAIL_CAPTURE":
		// Jailing an opponent's card yields a flat Mojo boost for the club
		gain = 5

		// PILLAR 1: Regional Security Synergy.
		// Governors (2+ territories) have interlocked security grids that yield
		// more prestige upon successful defense.
		if s.IsClubRegionalLocked(l, club) {
			gain += 10
		}

		// Tech Synergy: Bonus for every active hardware trap successfully defended.
		for key := range club.ActiveBuffs {
			if strings.HasPrefix(key, "TRAP_") {
				gain += 5
			}
		}
	}

	if gain <= 0 && reason == "REVENUE" {
		// Minimum gain for any revenue event to ensure progress
		gain = 1
	}

	// PILLAR 1: Hardened Infrastructure Scaling.
	// Each additional territory increases Mojo efficiency by 20% (Capped at 2.0x).
	efficiencyMult := 1.0 + (float64(len(club.Territories)-1) * 0.20)
	if efficiencyMult > 2.0 {
		efficiencyMult = 2.0
	}

	// Regional Governor Synergy is now an additive 30% bonus rather than a multiplicative 50%.
	if len(club.Territories) >= 2 {
		efficiencyMult += 0.30
	}

	finalGain := int(float64(gain) * efficiencyMult)
	// Global Event Cap: Prevent runaway inflation during peak tournament activity
	if finalGain > 60 {
		finalGain = 60
	}
	return finalGain
}

/**
 * HandlePurifyCard allows a Vitality Lab to remove the 'Fallen' debuff from an asset.
 * PILLAR 7: Underworld Recovery.
 */
func (s *ClubService) HandlePurifyCard(l *Lobby, env *Envelope) {
	var data struct {
		CardID int `json:"card_id"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()
	wallet, ok := l.wallets[env.FromID]
	if !ok { return }
	stats := l.leaderboard[wallet]

	club := l.clubs[stats.EmployerClubID]
	if club == nil || club.Type != "Vitality" {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Access Denied: Purification Rituals require a Vitality Lab."}`)})
		return
	}

	const purifyCost = 750 * 1000000
	if l.playerBalances[wallet] < purifyCost {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Insufficient rewards for purification."}`)})
		return
	}

	if card, exists := l.inventory[data.CardID]; exists && card.Fallen {
		l.playerBalances[wallet] -= purifyCost
		card.Fallen = false
		l.inventory[data.CardID] = card
		l.persistentCardCache[data.CardID] = card
		l.logAdminAuditLocked("ASSET_PURIFIED", wallet, fmt.Sprintf("Card: %d", data.CardID))
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"✨ <b>PURIFICATION SUCCESS:</b> Fallen debuff removed. Genetic stability restored."}`)})
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	}
}
func (s *ClubService) DistributeCourthouseFineMicroToClubsLocked(l *Lobby, amountMicro uint64) {
	if l.tokenSinkRouter == nil { return }

	// PILLAR 2: Industrial Loop (Token-Sink Router migration).
	// Atomic redistribution: 50% Faucet, 35% Global Clubs, 15% Regional Governors.
	// Passing targetID=0 and targetDistrict="GLOBAL" triggers sector-wide distribution.
	matrix := RevenueSplitMatrix{FaucetShare: 0.50, ClubShare: 0.35, GovernanceShare: 0.15}
	_ = l.tokenSinkRouter.RouteCriminalTax("COURTHOUSE_FINE", amountMicro, matrix, 0, "sector_all")

	amountBase := float64(amountMicro) / 1000000.0

	// PILLAR 1: Organizational Mojo.
	// Organizations act as security guilds; processing global fines awards Mojo.
	if len(l.clubs) > 0 {
		shareBase := (amountBase * 0.35) / float64(len(l.clubs))
		for _, club := range l.clubs {
			mojo := s.CalculateMojoGain(l, club, "REVENUE", shareBase)
			club.Mojo += mojo
			club.LastActivity = time.Now()

			// UI Parity Sync: Update the treasury float from the router node
			numericID, _ := strconv.ParseUint(strings.TrimPrefix(club.ID, "CLUB-"), 10, 64)
			if node, ok := l.tokenSinkRouter.ActiveClubs[numericID]; ok {
				club.TreasuryMicro = node.TreasuryBalance
				club.Treasury = float64(node.TreasuryBalance) / 1000000.0
			}

			l.achievementService.CheckMojoSurgeAchievementLocked(l, club.ID)
		}
	}

	// PILLAR 1: Regional Governor Mojo.
	// Governors are rewarded with social prestige for maintaining the sector's legal framework.
	var governors []*Club
	for _, club := range l.clubs {
		if s.IsClubRegionalLocked(l, club) {
			governors = append(governors, club)
		}
	}
	if len(governors) > 0 {
		govShareBase := (amountBase * 0.15) / float64(len(governors))
		for _, gov := range governors {
			mojo := s.CalculateMojoGain(l, gov, "REVENUE", govShareBase)
			gov.Mojo += mojo
			gov.LastActivity = time.Now()
			l.achievementService.CheckMojoSurgeAchievementLocked(l, gov.ID)
		}
	}

	// PILLAR 2: Ledger Integrity.
	l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0
	l.applyDynamicScalingLocked()
}

// HandleCreateLease allows a player to put a card up for lease in their club.
func (s *ClubService) HandleCreateLease(l *Lobby, env *Envelope) {
	var data struct {
		ClubID        string  `json:"club_id"`
		CardID        int     `json:"card_id"`
		Price         float64 `json:"price"` // Base VBV
		DurationHours int     `json:"duration_hours"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	wallet, ok := l.wallets[env.FromID]
	if !ok {
		return
	}

	club, exists := l.clubs[data.ClubID]
	if !exists {
		return
	}

	// Verify membership
	if _, isMember := club.Members[strings.ToLower(wallet)]; !isMember {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Lease Failed: Club membership required."}`)})
		return
	}

	stats := l.leaderboard[wallet]
	cardKey := fmt.Sprintf("CARD-%d", data.CardID)
	if stats.Inventory[cardKey] <= 0 {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Lease Failed: Card not found in inventory."}`)})
		return
	}

	// Escrow: Remove from lender
	stats.Inventory[cardKey]--
	l.leaderboard[wallet] = stats

	if club.Leases == nil {
		club.Leases = make(map[string]*Lease)
	}
	leaseID := fmt.Sprintf("LEASE-%d", time.Now().UnixNano())
	card, _ := l.inventory[data.CardID]

	club.Leases[leaseID] = &Lease{
		ID: leaseID, LenderWallet: wallet, CardID: data.CardID,
		CardName: card.Name, Price: data.Price, DurationHours: data.DurationHours,
		ClubID: data.ClubID,
	}
	club.LastActivity = time.Now() // Club is active when a lease is created

	l.logAdminAuditLocked("LEASE_CREATED", wallet, fmt.Sprintf("Club: %s, Card: %d, Price: %.2f", data.ClubID, data.CardID, data.Price))
	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"📜 <b>LEASE ADVERTISED:</b> %s is now available for rent in %s."}`, card.Name, club.Name))})
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// HandleTakeLease allows a player to rent a card from a club.
func (s *ClubService) HandleTakeLease(l *Lobby, env *Envelope) {
	var data struct {
		ClubID  string `json:"club_id"`
		LeaseID string `json:"lease_id"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	borrowerWallet, ok := l.wallets[env.FromID]
	if !ok {
		return
	}

	club, exists := l.clubs[data.ClubID]
	if !exists || club.Leases[data.LeaseID] == nil {
		return
	}

	lease := club.Leases[data.LeaseID]
	if lease.Borrower != "" || strings.EqualFold(lease.LenderWallet, borrowerWallet) {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Lease Error: Invalid borrower or already active."}`)})
		return
	}

	priceMicro := uint64(lease.Price * 1000000)
	if l.playerBalances[borrowerWallet] < priceMicro {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Lease Error: Insufficient funds."}`)})
		return
	}

	// PILLAR 1: Industrial Lease Revenue Distribution.
	// Split: 50% Lender, 20% Faucet (Tax), 20% Club Treasury, 10% Members.
	// We use micro-unit math to ensure absolute ledger integrity.
	l.playerBalances[borrowerWallet] -= priceMicro

	faucetShareMicro := (priceMicro * 20) / 100
	clubShareMicro := (priceMicro * 20) / 100
	lenderShareMicro := (priceMicro * 50) / 100
	memberShareTotalMicro := priceMicro - faucetShareMicro - clubShareMicro - lenderShareMicro

	// PILLAR 3: Financial Proof.
	// Record lease initiation on-chain to archive the expected revenue distribution.
	takeDetails := map[string]interface{}{
		"id":       lease.ID,
		"lender":   lease.LenderWallet,
		"borrower": borrowerWallet,
		"card_id":  lease.CardID,
		"price":    lease.Price,
		"duration": lease.DurationHours,
		"split":    map[string]float64{"lender": float64(lenderShareMicro) / 1000000.0, "faucet": float64(faucetShareMicro) / 1000000.0, "club": float64(clubShareMicro) / 1000000.0, "members": float64(memberShareTotalMicro) / 1000000.0},
		"ts":       time.Now().Unix(),
	}

	l.faucetBalance += float64(faucetShareMicro) / 1000000.0
	club.LastActivity = time.Now() // Club is active when a lease is taken
	l.applyDynamicScalingLocked()

	numMembers := uint64(len(club.Members))
	if numMembers > 0 {
		perMemberMicro := memberShareTotalMicro / numMembers
		for m := range club.Members {
			l.playerBalances[strings.ToLower(m)] += perMemberMicro
		}

		// Precision Recovery: Redirect division remainder to Club Treasury.
		// This handles cases where the 10% share isn't perfectly divisible by member count.
		memberRemainderMicro := memberShareTotalMicro - (perMemberMicro * numMembers)
		clubShareMicro += memberRemainderMicro
	} else {
		// Fallback: If no members, the 10% share defaults to the Club Treasury
		clubShareMicro += memberShareTotalMicro
	}

	l.playerBalances[strings.ToLower(lease.LenderWallet)] += lenderShareMicro

	// PILLAR 2: Unified Organizational Accounting.
	if l.tokenSinkRouter != nil {
		l.tokenSinkRouter.Mu.Lock()
		numericID, _ := strconv.ParseUint(strings.TrimPrefix(club.ID, "CLUB-"), 10, 64)
		if node, ok := l.tokenSinkRouter.ActiveClubs[numericID]; ok {
			node.TreasuryBalance += clubShareMicro
			club.TreasuryMicro = node.TreasuryBalance
			club.Treasury = float64(node.TreasuryBalance) / 1000000.0
		}
		l.tokenSinkRouter.Mu.Unlock()
	} else {
		club.Treasury += float64(clubShareMicro) / 1000000.0
	}

	// Execute lease
	lease.Borrower = borrowerWallet
	lease.ExpiresAt = time.Now().Add(time.Duration(lease.DurationHours) * time.Hour)

	borrowerStats := l.leaderboard[borrowerWallet]
	if borrowerStats.Inventory == nil {
		borrowerStats.Inventory = make(map[string]int)
	}
	borrowerStats.Inventory[fmt.Sprintf("CARD-%d", lease.CardID)]++
	l.leaderboard[borrowerWallet] = borrowerStats

	l.logAdminAuditLocked("LEASE_TAKEN", borrowerWallet, fmt.Sprintf("ID: %s, Price: %.2f", lease.ID, lease.Price))
	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🤝 <b>LEASE SECURED:</b> You have rented %s."}`, lease.CardName))})

	// Dispatch on-chain log for financial verification
	go func(td interface{}) {
		jsonPayload, _ := json.Marshal(td)
		l.sendNoteTx(fmt.Sprintf("VBT_LEASE_TAKE:%s", string(jsonPayload)))
	}(takeDetails)

	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// ProcessLeaseExpirations handles the return of leased cards to their owners.
func (s *ClubService) ProcessLeaseExpirations(l *Lobby) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	now := time.Now()
	leasesExpired := false
	for _, club := range l.clubs {
		for id, lease := range club.Leases {
			if lease.Borrower != "" && now.After(lease.ExpiresAt) {
				// PILLAR 3: Financial Proof.
				// Reconstruct the revenue split for the on-chain audit trail.
				priceMicro := uint64(lease.Price * 1000000)
				faucetShareMicro := (priceMicro * 20) / 100
				clubShareMicro := (priceMicro * 20) / 100
				lenderShareMicro := (priceMicro * 50) / 100
				memberShareTotalMicro := priceMicro - faucetShareMicro - clubShareMicro - lenderShareMicro

				returnDetails := map[string]interface{}{
					"id":       lease.ID,
					"lender":   lease.LenderWallet,
					"borrower": lease.Borrower,
					"card_id":  lease.CardID,
					"price":    lease.Price,
					"split": map[string]float64{
						"lender":  float64(lenderShareMicro) / 1000000.0,
						"faucet":  float64(faucetShareMicro) / 1000000.0,
						"club":    float64(clubShareMicro) / 1000000.0,
						"members": float64(memberShareTotalMicro) / 1000000.0,
					},
					"ts": now.Unix(),
				}

				// PILLAR 3: Identity Hardening.
				l.ensurePlayerStatsMapsInitialized(lease.Borrower)
				l.ensurePlayerStatsMapsInitialized(lease.LenderWallet)

				if bStats, bExists := l.leaderboard[lease.Borrower]; bExists {
					cardKey := fmt.Sprintf("CARD-%d", lease.CardID)
					if bStats.Inventory[cardKey] > 0 {
						bStats.Inventory[cardKey]--
					}
					bStats.Reputation = l.CalculateReputation(bStats)
					l.leaderboard[lease.Borrower] = bStats
				}

				if lStats, lExists := l.leaderboard[lease.LenderWallet]; lExists {
					cardKey := fmt.Sprintf("CARD-%d", lease.CardID)
					lStats.Inventory[cardKey]++
					lStats.Reputation = l.CalculateReputation(lStats)
					l.leaderboard[lease.LenderWallet] = lStats
				}

				// Dispatch on-chain log for financial verification
				go func(rd interface{}) {
					jsonPayload, _ := json.Marshal(rd)
					l.sendNoteTx(fmt.Sprintf("VBT_LEASE_RETURN:%s", string(jsonPayload)))
				}(returnDetails)

				delete(club.Leases, id)
				// PILLAR 5: Deadlock Resolution. Use Locked variant while holding the mutex.
				l.logAdminAuditLocked("LEASE_EXPIRED", lease.LenderWallet, fmt.Sprintf("Card %d returned from %s", lease.CardID, lease.Borrower))
				leasesExpired = true
			}
		}
	}

	if leasesExpired {
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	}
}
