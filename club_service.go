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

// handleHeist processes a criminal attempt to loot a Club Treasury.
func (l *Lobby) handleHeist(env *Envelope) {
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
	if l.isPlayerAffiliatedWithClubLocked(wallet, targetClub) {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ <b>HEIST BLOCKED:</b> Internal infiltration of alliance treasuries is strictly forbidden."}`)})
		return
	}

	// SUCCESS CHANCE CALCULATION: Base 50% chance + (Effective Cunning - Security Level) / 100
	successChance := 0.50

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

	// PILLAR 1: Regional Security Integration.
	isRegion := l.isClubRegionalLocked(targetClub)
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
	successChance += (float64(playerStats.GetEffectiveCunning()) - securityLevel) / 100.0

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
		if playerStats.GetEffectiveCunning() >= 3 && rand.Float64() < 0.25 {
			canKidnap = true
		}

		// Calculate Loot: 10% of target club's treasury, capped at 500 VBV
		maxLootMicro := uint64(500 * 1000000)
		// PILLAR 2: Precision Hardening. Convert treasury to micro-units first for accurate calculation.
		clubTreasuryMicro := uint64(targetClub.Treasury*1000000 + 0.5) // Round to nearest micro-unit
		lootMicro := (clubTreasuryMicro*10 + 50) / 100                  // 10% of treasury, rounded
		if lootMicro > maxLootMicro {
			lootMicro = maxLootMicro
		}

		// INDUSTRIAL LOOP: 10% "Fence Fee" returns to the Faucet Pool
		// Integer math with rounding to nearest micro-unit
		fenceFeeMicro := (lootMicro*10 + 50) / 100
		netLootMicro := lootMicro - fenceFeeMicro
		fenceFee = float64(fenceFeeMicro) / 1000000.0
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
		targetClub.Treasury -= float64(lootMicro) / 1000000.0
		targetClub.LastActivity = now // Consistent activity tracking

		// Add net loot to player's virtual rewards
		l.playerBalances[wallet] += netLootMicro

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
		l.unlockAchievementLocked(wallet, "FIRST_HEIST")
		playerStats = l.leaderboard[wallet] // Re-fetch to avoid clobbering achievement

	} else {
		status = "failure"
		playerStats.WantedLevel += 15
		playerStats.Playstyle.RiskTolerance += 0.10
		playerStats.HeistAttempts++

		// MOJO GAIN: Reward the club for successful defense
		mojoGain := l.calculateMojoGain(targetClub, "DEFENSE", 0)
		targetClub.Mojo += mojoGain
		l.checkMojoSurgeAchievementLocked(targetClub.ID)

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

	// PILLAR 1: Achievement Integration.
	// Evaluate Heist Saboteur progress after saving state to the leaderboard.
	l.checkHeistSaboteurAchievementLocked(wallet)

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

// handleSabotage allows a player to pay $VBV to disable a target club's hardware defenses for 1 hour.
func (l *Lobby) handleSabotage(env *Envelope) {
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

	// COST: 1000 $VBV (micro-units)
	const sabotageCostMicro = 1000 * 1000000
	if l.playerBalances[wallet] < sabotageCostMicro {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Sabotage Failed: Insufficient $VBV rewards."}`)})
		return
	}

	// Deduct payment and trigger industrial loop scaling
	l.playerBalances[wallet] -= sabotageCostMicro
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
		"cost":      1000.0,
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

// handleCreateClub allows a player to found a new organization.
func (l *Lobby) handleCreateClub(env *Envelope) {
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

	// INDUSTRIAL LOOP: Update physical vault total and trigger reactive scaling.
	l.faucetBalance += 5000.0
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

// handleJoinClub allows a player to become a member of an existing club.
func (l *Lobby) handleJoinClub(env *Envelope) {
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
		playerStats.Reputation = l.CalculateReputation(playerStats)
		l.leaderboard[wallet] = playerStats

		if minReputationRequired > 0 && (playerStats.Reputation < minReputationRequired || playerStats.GetEffectiveMojo() < minMojoRequired) {
			l.applyDynamicScalingLocked()
			l.mutex.Unlock()
			msg := fmt.Sprintf(`{"text":"❌ Club Entry Failed: Elite club %s requires %d Reputation and %d Mojo social standing to join."}`, club.Name, minReputationRequired, minMojoRequired)
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(msg)})
			return
		}

		club.Members[strings.ToLower(wallet)] = time.Now()
		club.Treasury += 250.0
		club.LastActivity = time.Now()

		// PILLAR 1: Mojo Gain for Club Entry revenue.
		mojoGain := l.calculateMojoGain(club, "REVENUE", 250.0)
		club.Mojo += mojoGain
		l.checkMojoSurgeAchievementLocked(club.ID)

		// PILLAR 2: Ledger Integrity. Update scaling to reflect the new treasury liability.
		l.applyDynamicScalingLocked()

		l.mutex.Unlock()
		l.logAdminAudit("CLUB_JOIN", wallet, fmt.Sprintf("Club: %s", data.ClubID))
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🤝 Welcome to %s!"}`, club.Name))})

		// Sync UI to update membership lists and treasury balances
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	} else {
		// Fallback for organizational lookup failure: update scaling for the added faucet entry.
		l.applyDynamicScalingLocked()
		l.mutex.Unlock()
	}
}

// handlePurchaseTerritory allows a club to expand its influence.
func (l *Lobby) handlePurchaseTerritory(env *Envelope) {
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

	purchaseCost := 2500.0
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

	verified, txTime, err := l.verifyBuyInTransaction(verifyNet, data.TxID, uint64(purchaseCost*divisor), assetID, ownerWallet, vaultAddr, prefix)
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

	// INDUSTRIAL LOOP: Add cost to vault before processing distributions
	l.faucetBalance += purchaseCost

	club, exists = l.clubs[data.ClubID]
	if !exists || !strings.EqualFold(club.OwnerWallet, ownerWallet) {
		l.applyDynamicScalingLocked()
		l.mutex.Unlock()
		return
	}

	club.Territories = append(club.Territories, data.TerritoryID)

	// PILLAR 1: Immediate Regional Role & Achievement Integration.
	// If this is the second district, trigger Governor status immediately to ensure atomic UI sync.
	if l.isClubRegionalLocked(club) {
		if club.RegionName == "" {
			club.RegionName = "Governor"
		}
		l.unlockAchievementLocked(strings.ToLower(club.OwnerWallet), "GOVERNOR")
	}

	// PILLAR 1: Regional Governor Protocol Fee.
	// A portion (5%) of the territory purchase cost is distributed to existing Regional Governors.
	var governorProtocolFee float64
	var governors []*Club
	for _, c := range l.clubs {
		// Check if it's a Regional Governor (club with 2+ territories)
		if l.isClubRegionalLocked(c) {
			governors = append(governors, c)
		}
	}

	if len(governors) > 0 {
		// Calculate 5% of the purchase cost as a protocol fee.
		// Use micro-unit math for precision.
		protocolFeeMicro := uint64(purchaseCost * 0.05 * divisor)
		governorProtocolFee = float64(protocolFeeMicro) / divisor

		feePerGovernor := governorProtocolFee / float64(len(governors))
		for _, govClub := range governors {
			govClub.Treasury += feePerGovernor
			govClub.LastActivity = time.Now()
		}
		l.logAdminAuditLocked("TERRITORY_PROTOCOL_FEE", data.TerritoryID, fmt.Sprintf("Distributed %.2f $VBV to %d Governors.", governorProtocolFee, len(governors)))
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

// handleRestockInventory allows authorized staff to restock items in the club shop.
func (l *Lobby) handleRestockInventory(env *Envelope) {
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
	// Restocking reduces club reserves, returning funds to the unreserved pool.
	l.applyDynamicScalingLocked()

	l.logAdminAuditLocked("CLUB_RESTOCK", ownerWallet, fmt.Sprintf("Club: %s, Item: %s, Qty: %d, Cost: %.2f", club.Name, data.ItemID, data.Quantity, totalCostBase))
	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"📦 <b>RESTOCK COMPLETE:</b> Added %d units of %s to inventory."}`, data.Quantity, item.Name))})

	// Trigger Global Sync to update UI (faucet/treasury/inventory)
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// handlePurchaseItem processes a player's request to buy an item from a district shop.
// PILLAR 2: Ledger Integrity. Moves the logic from lobby_manager.go to enforce modular authority.
func (l *Lobby) handlePurchaseItem(env *Envelope) {
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
	if item.IsMasterTier && !l.isClubRegionalLocked(targetClub) {
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
	l.distributeShopRevenueLocked(data.TerritoryID, data.Price, data.ItemID)

	l.logAdminAuditLocked("ITEM_PURCHASE", wallet, fmt.Sprintf("Item: %s, Territory: %s, Cost: %.2f", data.ItemID, data.TerritoryID, float64(data.Price)/1000000.0))

	notification := fmt.Sprintf(`{"text":"📦 <b>PURCHASE COMPLETE:</b> %s added to inventory."}`, item.Name)
	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(notification)})

	// Sync back to client to update local UI inventory and balances
	msg := l.getLobbyUpdateMsgLocked()
	go func() { l.broadcast <- msg }()
}

// handleAllianceInvite allows a club owner to propose a partnership to another club.
func (l *Lobby) handleAllianceInvite(env *Envelope) {
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
	l.logAdminAuditLocked("ALLIANCE_INVITE", wallet, fmt.Sprintf("From: %s, To: %s", clubA.Name, clubB.Name))

	// Notify Target Owner
	targetOwnerCID := l.getClientIDFromWalletLocked(clubB.OwnerWallet)
	if targetOwnerCID != "" {
		l.sendToClientLocked(targetOwnerCID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🤝 <b>ALLIANCE PROPOSAL:</b> %s has proposed a regional alliance!"}`, escapeHTML(clubA.Name)))})
	}
	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"📡 <b>INVITATION SENT:</b> Awaiting response from alliance target."}`)})
}

// handleAllianceAccept allows a club owner to finalize a partnership.
func (l *Lobby) handleAllianceAccept(env *Envelope) {
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
	if l.isClubRegionalLocked(clubA) {
		if clubA.RegionName == "" {
			clubA.RegionName = "Governor"
		}
		if clubB.RegionName == "" {
			clubB.RegionName = "Governor"
		}
		l.unlockAchievementLocked(strings.ToLower(clubA.OwnerWallet), "GOVERNOR")
		l.unlockAchievementLocked(strings.ToLower(clubB.OwnerWallet), "GOVERNOR")
	}

	l.logAdminAuditLocked("ALLIANCE_FORMED", wallet, fmt.Sprintf("Clubs: %s & %s", clubA.Name, clubB.Name))
	l.broadcastToAdmins(fmt.Sprintf("🤝 <b>REGIONAL ALLIANCE:</b> %s and %s have combined their influence.", clubA.Name, clubB.Name))

	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// handleAllianceDissolve allows a club owner to terminate their partnership.
func (l *Lobby) handleAllianceDissolve(env *Envelope) {
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
	if !l.isClubRegionalLocked(clubA) {
		clubA.RegionName = ""
	}
	if existsB && !l.isClubRegionalLocked(clubB) {
		clubB.RegionName = ""
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

// isClubRegionalLocked checks if a club (or its alliance) owns 2 or more territories.
func (l *Lobby) isClubRegionalLocked(club *Club) bool {
	count := len(club.Territories)
	if club.AlliedClubID != "" {
		if allied, ok := l.clubs[club.AlliedClubID]; ok {
			count += len(allied.Territories)
		}
	}
	return count >= 2
}

// isPlayerAffiliatedWithClubLocked checks if a player is a member or owner of a club or its allied organization.
func (l *Lobby) isPlayerAffiliatedWithClubLocked(wallet string, club *Club) bool {
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

// distributeShopRevenue handles payout to club treasuries based on shop turnover.
func (l *Lobby) distributeShopRevenue(territoryID string, amountMicro uint64, itemID string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.distributeShopRevenueLocked(territoryID, amountMicro, itemID)
}

// distributeShopRevenueLocked handles payout to club treasuries with Regional Taxation.
// PILLAR 2: Industrial Seal. Ensures any virtual balance not allocated as commission
// returns to the Faucet pool to maintain mathematical circularity.
func (l *Lobby) distributeShopRevenueLocked(territoryID string, amountMicro uint64, itemID string) {
	now := time.Now()
	divisor := 1000000.0

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
		if l.isClubRegionalLocked(club) {
			governors = append(governors, club)
		}
	}

	// 3. Calculate total commission based on item type and club rate
	rate := owningClub.Commission

	// PILLAR 1: Context Preservation. isPerishable is a placeholder for future
	// logic where certain items (food/meds) have different tax implications.
	isPerishable := false
	if _, ok := GlobalShopRegistry[itemID]; ok {
		// Future: item-based perishability checks
	}

	if isPerishable {
		if rate < 0.05 {
			rate = 0.05
		}
		if rate > 0.50 {
			rate = 0.50
		}
	}

	if rate < 0.05 {
		rate = 0.05
	}
	if rate > 0.50 {
		rate = 0.50
	}

	// Use micro-unit precision for all distribution logic to prevent dust leaks
	totalCommissionMicro := uint64(float64(amountMicro)*rate + 0.5)

	// MOJO GAIN: Progress the club's social standing based on shop turnover
	mojoGain := l.calculateMojoGain(owningClub, "REVENUE", float64(amountMicro)/divisor)
	owningClub.Mojo += mojoGain
	l.checkMojoSurgeAchievementLocked(owningClub.ID)

	// 4. Regional Governor Tax: 5% is distributed to all Governors.
	var totalDistributedToGovsMicro uint64 = 0
	var regionalTaxPoolMicro uint64 = 0
	if len(governors) > 0 {
		regionalTaxPoolMicro = (totalCommissionMicro*5 + 50) / 100
		taxPerGovernorMicro := regionalTaxPoolMicro / uint64(len(governors))

		for _, govClub := range governors {
			taxBase := float64(taxPerGovernorMicro) / divisor
			govClub.Treasury += taxBase
			govClub.LastActivity = now

			// PILLAR 1: Mojo Gain for Governor Tax revenue.
			// Governors are rewarded with social prestige for maintaining the sector's industrial loop.
			mojoGain := l.calculateMojoGain(govClub, "REVENUE", taxBase)
			govClub.Mojo += mojoGain
			l.checkMojoSurgeAchievementLocked(govClub.ID)
		}
		totalDistributedToGovsMicro = taxPerGovernorMicro * uint64(len(governors))

		// PILLAR 2: Industrial Seal (Remainder Recovery for Governor Tax).
		// Any micro-unit dust from the regional tax pool that couldn't be perfectly
		// distributed to governors is returned to the Faucet.
		regionalTaxDustMicro := regionalTaxPoolMicro - totalDistributedToGovsMicro
		if regionalTaxDustMicro > 0 {
			l.faucetBalance += float64(regionalTaxDustMicro) / divisor
			l.logAdminAuditLocked("SHOP_GOV_TAX_DUST_TO_FAUCET", owningClub.ID, fmt.Sprintf("Regional tax remainder: %.2f", float64(regionalTaxDustMicro)/divisor))
		}
	}

	// 5. Final Payout to the Territory Owner (Net after Regional Tax)
	netCommissionToOwningClubMicro := totalCommissionMicro - regionalTaxPoolMicro
	owningClub.Treasury += float64(netCommissionToOwningClubMicro) / divisor
	owningClub.LastActivity = now

	// PILLAR 2: Ledger Integrity.
	// Re-calculate scaling to reflect tokens moving into the club treasury.
	l.applyDynamicScalingLocked()
}

// calculateMojoGain computes the Mojo increase for a club based on economic or defensive events.
// It weights the gain based on territory ownership and Regional Governor status.
func (l *Lobby) calculateMojoGain(club *Club, reason string, value float64) int {
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
		if l.isClubRegionalLocked(club) {
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

// distributeCourthouseFineMicroToClubsLocked distributes a portion of the fine among clubs and governors,
// with the Regional Governor tax calculated as 15% of the TOTAL fine (30% of the club share).
// This function assumes the main lobby mutex is already held by the caller.
// Any micro-unit remainders are returned to the Faucet pool to maintain the Industrial Seal.
func (l *Lobby) distributeCourthouseFineMicroToClubsLocked(amountMicro uint64) {
	now := time.Now()
	if len(l.clubs) == 0 {
		return
	}

	divisor := 1000000.0
	var governors []*Club
	for _, club := range l.clubs {
		if l.isClubRegionalLocked(club) {
			governors = append(governors, club)
		}
	}

	// PILLAR 1: Regional Governor Tax Compliance (15% of Total).
	// Since amountMicro is half of the total fine, we take 30% of it.
	var totalDistributedToGovsMicro uint64
	if len(governors) > 0 {
		regionalTaxPoolMicro := (amountMicro*30 + 50) / 100
		taxPerGovernorMicro := regionalTaxPoolMicro / uint64(len(governors))
		for _, govClub := range governors {
			govClub.Treasury += float64(taxPerGovernorMicro) / divisor
			govClub.LastActivity = now
		}
		totalDistributedToGovsMicro = taxPerGovernorMicro * uint64(len(governors))
	}

	remainingPoolMicro := amountMicro - totalDistributedToGovsMicro
	sharePerClubMicro := remainingPoolMicro / uint64(len(l.clubs))
	shareBase := float64(sharePerClubMicro) / divisor
	for _, club := range l.clubs {
		club.Treasury += shareBase
		club.LastActivity = now

		// PILLAR 1: Mojo Gain for Fine redistribution revenue.
		// Organizations act as "Security Guilds"; processing fines builds organizational Mojo.
		mojoGain := l.calculateMojoGain(club, "REVENUE", shareBase)
		club.Mojo += mojoGain
		l.checkMojoSurgeAchievementLocked(club.ID)
	}

	// PILLAR 2: Industrial Seal (Remainder Recovery).
	// Return unallocated micro-unit dust to the Faucet balance.
	distributedToClubsMicro := sharePerClubMicro * uint64(len(l.clubs))
	leftoverMicro := amountMicro - totalDistributedToGovsMicro - distributedToClubsMicro
	if leftoverMicro > 0 {
		l.faucetBalance += float64(leftoverMicro) / divisor
	}
}

// handleCreateLease allows a player to put a card up for lease in their club.
func (l *Lobby) handleCreateLease(env *Envelope) {
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

// handleTakeLease allows a player to rent a card from a club.
func (l *Lobby) handleTakeLease(env *Envelope) {
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
	club.Treasury += float64(clubShareMicro) / 1000000.0

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

// processLeaseExpirations handles the return of leased cards to their owners.
func (l *Lobby) processLeaseExpirations() {
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
