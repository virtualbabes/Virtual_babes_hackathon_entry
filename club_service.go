package main

import (
	"encoding/json"
	"fmt"
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

	playerStats := l.leaderboard[wallet]
	targetClub, exists := l.clubs[data.TargetClubID]
	if !exists {
		return
	}

	// SUCCESS CHANCE CALCULATION: Base 50% chance + (Effective Cunning - Security Level) / 100
	successChance := 0.50
	securityLevel := float64(targetClub.Mojo) / 10.0

	for _, role := range targetClub.Staff {
		if role == "Security" {
			securityLevel += 15.0
		}
	}

	// Apply Attribute Modifier: Players compete against the club's Mojo and Security staff
	successChance += (float64(playerStats.GetEffectiveCunning()) - securityLevel) / 100.0

	// Apply Trap Penalties from the Club's active buffs with lazy pruning
	now := time.Now()
	for trapID, itemID := range targetClub.ActiveBuffs {
		// Check for trap expiration before applying modifiers
		if expiry, exists := targetClub.BuffExpirations[trapID]; exists && now.After(expiry) {
			delete(targetClub.ActiveBuffs, trapID)
			delete(targetClub.BuffExpirations, trapID)
			continue
		}

		if item, ok := GlobalShopRegistry[itemID]; ok && item.HeistSuccessModifier != 0 {
			successChance += item.HeistSuccessModifier
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
		// Use micro-units for internal precision logic
		maxLootMicro := uint64(500 * 1000000)
		lootMicro := uint64(targetClub.Treasury * 0.10 * 1000000)
		if lootMicro > maxLootMicro {
			lootMicro = maxLootMicro
		}

		// INDUSTRIAL LOOP: 10% "Fence Fee" returns to the Faucet Pool
		// Integer math with rounding to nearest micro-unit
		fenceFeeMicro := (lootMicro*10 + 50) / 100
		netLootMicro := lootMicro - fenceFeeMicro
		fenceFee = float64(fenceFeeMicro) / 1000000.0
		netLoot = float64(netLootMicro) / 1000000.0

		if fenceFeeMicro > 0 {
			l.faucetBalance += fenceFee
			l.applyDynamicScalingLocked()
		}
		playerStats.Playstyle.RiskTolerance += 0.05
		playerStats.HeistAttempts++
		targetClub.Treasury -= float64(lootMicro) / 1000000.0
		targetClub.LastActivity = now // Consistent activity tracking
		playerStats.WantedLevel += 5
		playerStats.Reputation = l.CalculateReputation(playerStats) // Update social standing
		playerStats.Cunning += 1                                    // Successful heists improve Cunning

		// Add net loot to player's rewards
		l.rewards[wallet] += netLootMicro

		// Achievement unlock uses the Locked variant since we already hold the lobby mutex.
		l.unlockAchievementLocked(wallet, "FIRST_HEIST")

	} else {
		status = "failure"
		playerStats.WantedLevel += 15
		playerStats.Playstyle.RiskTolerance += 0.10
		playerStats.Reputation = l.CalculateReputation(playerStats) // Update social standing
		playerStats.HeistAttempts++

		// MOJO GAIN: Reward the club for successful defense
		mojoGain := l.calculateMojoGain(targetClub, "DEFENSE", 0)
		targetClub.Mojo += mojoGain

		targetClub.LastHeistAt = now  // Trigger visual "Under Attack" state
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
				l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🚨 <b>GUARD DOG BUST:</b> You were caught by a Bio-Guard Dog! Your rarest card (%s) has been jailed by %s."}`, rarestCard.Name, targetClub.Name))})
			}
		}
	}

	l.leaderboard[wallet] = playerStats
	l.logAdminAudit("HEIST_ATTEMPT", wallet, fmt.Sprintf("Target: %s, Result: %s, Loot: %.2f, FenceFee: %.2f", data.TargetClubID, status, netLoot, fenceFee))
	if status == "success" {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>HEIST SUCCESS:</b> You looted %.2f $VBV from %s (Net after Fence Fee)."}`, netLoot, targetClub.Name))})
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
	l.sendToClient(env.FromID, Envelope{Type: "heist_result", Payload: response})

	// Trigger Global Sync so others see the treasury loot and the player's new Wanted Level
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

	verified, _, err := l.verifyBuyInTransaction(verifyNet, data.TxID, 5000*1000000, assetID, wallet, vaultAddr)
	if err != nil || !verified {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Club Foundry Error: Payment verification failed."}`)})
		return
	}

	l.mutex.Lock()
	clubID := fmt.Sprintf("CLUB-%d", time.Now().Unix())
	newClub := &Club{
		ID: clubID, Name: data.Name, OwnerWallet: wallet, Type: data.Type,
		Territories: []string{data.TerritoryID}, Commission: 0.05,
		Staff: make(map[string]string), Members: map[string]time.Time{strings.ToLower(wallet): time.Now()},
		Inventory:       make(map[string]int),
		ActiveBuffs:     make(map[string]string),
		Leases:          make(map[string]*Lease),
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

	assetID := voiConfig.AssetID
	verifyNet := "Voi"
	if strings.ToUpper(data.Network) == "ALGO" || strings.HasPrefix(strings.ToUpper(data.Network), "ALGO") {
		assetID = l.avoiAssetID
		verifyNet = "Algorand"
	}

	verified, _, err := l.verifyBuyInTransaction(verifyNet, data.TxID, 500*1000000, assetID, wallet, vaultAddr)
	if err != nil || !verified {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Club Entry Error: Payment verification failed."}`)})
		return
	}

	l.mutex.Lock()
	if club, exists := l.clubs[data.ClubID]; exists {
		if club.Members == nil {
			club.Members = make(map[string]time.Time)
		}

		// Prevent double-joining exploit
		if _, isMember := club.Members[strings.ToLower(wallet)]; isMember {
			l.mutex.Unlock()
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"⚠️ You are already a member of this club."}`)})
			return
		}

		club.Members[strings.ToLower(wallet)] = time.Now()
		club.Treasury += 250.0
		club.LastActivity = time.Now()
		l.mutex.Unlock()
		l.logAdminAudit("CLUB_JOIN", wallet, fmt.Sprintf("Club: %s", data.ClubID))
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🤝 Welcome to %s!"}`, club.Name))})

		// Sync UI to update membership lists and treasury balances
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	} else {
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
	if !exists || strings.ToLower(club.OwnerWallet) != strings.ToLower(ownerWallet) {
		l.mutex.RUnlock()
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Territory Purchase Failed: Unauthorized or Club not found."}`)})
		return
	}

	for _, existingClub := range l.clubs {
		for _, t := range existingClub.Territories {
			if t == data.TerritoryID {
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

	verified, _, err := l.verifyBuyInTransaction(verifyNet, data.TxID, uint64(purchaseCost*divisor), assetID, ownerWallet, vaultAddr)
	if err != nil || !verified {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Territory Purchase Failed: Payment verification failed."}`)})
		return
	}

	l.mutex.Lock()
	// RE-VERIFY: Ensure territory was not claimed while we were verifying the transaction
	for _, existingClub := range l.clubs {
		for _, t := range existingClub.Territories {
			if strings.EqualFold(t, data.TerritoryID) {
				l.mutex.Unlock()
				l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Territory Purchase Failed: District claimed by another entity during verification."}`)})
				return
			}
		}
	}

	club, exists = l.clubs[data.ClubID]
	if !exists || strings.ToLower(club.OwnerWallet) != strings.ToLower(ownerWallet) {
		l.mutex.Unlock()
		return
	}

	club.Territories = append(club.Territories, data.TerritoryID)
	l.clubs[data.ClubID] = club
	club.LastActivity = time.Now()
	l.mutex.Unlock()

	l.logAdminAudit("TERRITORY_PURCHASED", ownerWallet, fmt.Sprintf("Club: %s (%s), Territory: %s", club.Name, club.ID, data.TerritoryID))
	l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🗺️ Club '%s' has acquired %s!"}`, club.Name, data.TerritoryID))})

	go l.refreshRegionalRoles()

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

	isOwner := strings.EqualFold(club.OwnerWallet, ownerWallet)
	isManager := club.Staff[strings.ToLower(ownerWallet)] == "Manager"
	if !isOwner && !isManager {
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
	totalCostMicro := uint64(item.Price*1000000) * uint64(data.Quantity)
	totalCostBase := float64(totalCostMicro) / 1000000.0

	if club.Treasury < totalCostBase {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"❌ Restock Failed: Insufficient Treasury funds. Need %.2f $VBV."}`, totalCostBase))})
		return
	}

	club.Treasury -= totalCostBase
	// Ensure map is initialized even if old data exists
	if club.Inventory == nil {
		club.Inventory = make(map[string]int)
	}

	club.LastActivity = time.Now()
	club.Inventory[data.ItemID] += data.Quantity

	l.logAdminAudit("CLUB_RESTOCK", ownerWallet, fmt.Sprintf("Club: %s, Item: %s, Qty: %d, Cost: %.2f", club.Name, data.ItemID, data.Quantity, totalCostBase))
	l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"📦 <b>RESTOCK COMPLETE:</b> Added %d units of %s to inventory."}`, data.Quantity, item.Name))})
}

// distributeShopRevenue handles payout to club treasuries based on shop turnover.
func (l *Lobby) distributeShopRevenue(territoryID string, amountMicro uint64, itemID string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.distributeShopRevenueLocked(territoryID, amountMicro, itemID)
}

// distributeShopRevenueLocked handles payout to club treasuries with Regional Taxation.
func (l *Lobby) distributeShopRevenueLocked(territoryID string, amountMicro uint64, itemID string) {
	now := time.Now()
	isPerishable := strings.Contains(itemID, "stim") || strings.Contains(itemID, "catalyst") || strings.Contains(itemID, "pledge")

	// 1. Identify the specific club owning this territory
	var owningClub *Club
	for _, club := range l.clubs {
		for _, t := range club.Territories {
			if t == territoryID {
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
		if len(club.Territories) >= 2 {
			governors = append(governors, club)
		}
	}

	// 3. Calculate total commission based on item type and club rate
	rate := owningClub.Commission
	if isPerishable {
		if rate < 0.05 {
			rate = 0.05
		}
		if rate > 0.50 {
			rate = 0.50
		}
	}

	// Use micro-unit precision for all distribution logic to prevent dust leaks
	totalCommissionMicro := uint64(float64(amountMicro)*rate + 0.5)
	totalCommissionBase := float64(totalCommissionMicro) / 1000000.0

	// MOJO GAIN: Progress the club's social standing based on shop turnover
	mojoGain := l.calculateMojoGain(owningClub, "REVENUE", float64(amountMicro)/1000000.0)
	owningClub.Mojo += mojoGain

	// 4. Regional Governor Tax: 5% is distributed to all Governors.
	var totalDistributedToGovsMicro uint64
	if len(governors) > 0 {
		regionalTaxPoolMicro := (totalCommissionMicro*5 + 50) / 100
		taxPerGovernorMicro := regionalTaxPoolMicro / uint64(len(governors))

		for _, govClub := range governors {
			govClub.Treasury += float64(taxPerGovernorMicro) / 1000000.0
			govClub.LastActivity = now
		}
		totalDistributedToGovsMicro = taxPerGovernorMicro * uint64(len(governors))
	}

	// 5. Final Payout to the Territory Owner (Net after Regional Tax)
	netCommissionMicro := totalCommissionMicro - totalDistributedToGovsMicro
	owningClub.Treasury += float64(netCommissionMicro) / 1000000.0
	owningClub.LastActivity = now

	// INDUSTRIAL LOOP: Deduct commission from Faucet liquidity and re-scale
	l.faucetBalance -= totalCommissionBase
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
	case "DEFENSE":
		// Successful heist defense yields a flat Mojo boost
		gain = 15
	}

	if gain <= 0 && reason == "REVENUE" {
		// Minimum gain for any revenue event to ensure progress
		gain = 1
	}

	// PILLAR 1: Territory Weighting.
	// Ownership signifies infrastructure. Each additional territory increases Mojo efficiency by 25%.
	efficiencyMult := 1.0 + (float64(len(club.Territories)-1) * 0.25)

	// PILLAR 1: Regional Governor Bonus.
	// Governors (2+ territories) receive a flat +50% synergy bonus to all Mojo gains.
	if len(club.Territories) >= 2 {
		efficiencyMult *= 1.5
	}

	return int(float64(gain) * efficiencyMult)
}

// distributeCourthouseFineToClubsLocked distributes a portion of the fine among clubs and governors.
// This function assumes the main lobby mutex is already held by the caller.
func (l *Lobby) distributeCourthouseFineToClubsLocked(amount float64) {
	now := time.Now()
	if len(l.clubs) == 0 {
		return
	}

	var governors []*Club
	for _, club := range l.clubs {
		if len(club.Territories) >= 2 {
			governors = append(governors, club)
		}
	}

	regionalTaxPool := 0.0
	remainingPool := amount

	if len(governors) > 0 {
		regionalTaxPool = amount * 0.15
		remainingPool = amount - regionalTaxPool
		taxPerGovernor := regionalTaxPool / float64(len(governors))
		for _, govClub := range governors {
			govClub.Treasury += taxPerGovernor
			govClub.LastActivity = now // Revenue counts as activity
		}
	}

	sharePerClub := remainingPool / float64(len(l.clubs))
	for _, club := range l.clubs {
		club.Treasury += sharePerClub
		club.LastActivity = now // Revenue counts as activity
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

	l.logAdminAudit("LEASE_CREATED", wallet, fmt.Sprintf("Club: %s, Card: %d, Price: %.2f", data.ClubID, data.CardID, data.Price))
	l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"📜 <b>LEASE ADVERTISED:</b> %s is now available for rent in %s."}`, card.Name, club.Name))})
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
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Lease Error: Invalid borrower or already active."}`)})
		return
	}

	priceMicro := uint64(lease.Price * 1000000)
	if l.rewards[borrowerWallet] < priceMicro {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Lease Error: Insufficient funds."}`)})
		return
	}

	// Payment distribution: 50% creator, 20% faucet, 20% club, 10% members
	l.rewards[borrowerWallet] -= priceMicro
	l.faucetBalance += lease.Price * 0.20
	club.Treasury += lease.Price * 0.20
	club.LastActivity = time.Now() // Club is active when a lease is taken
	l.applyDynamicScalingLocked()

	memberShareTotalMicro := uint64(lease.Price * 0.10 * 1000000)
	numMembers := uint64(len(club.Members))
	if numMembers > 0 {
		perMemberMicro := memberShareTotalMicro / numMembers
		for m := range club.Members {
			l.rewards[strings.ToLower(m)] += perMemberMicro
		}

		// Precision Recovery: Redirect rounding remainder to Club Treasury to ensure no micro-units are lost
		remainderMicro := memberShareTotalMicro - (perMemberMicro * numMembers)
		if remainderMicro > 0 {
			club.Treasury += float64(remainderMicro) / 1000000.0
		}
	} else {
		// Fallback: If no members exist, the entire 10% share defaults to the Club Treasury
		club.Treasury += lease.Price * 0.10
	}
	l.rewards[strings.ToLower(lease.LenderWallet)] += uint64(float64(priceMicro) * 0.50)

	// Execute lease
	lease.Borrower = borrowerWallet
	lease.ExpiresAt = time.Now().Add(time.Duration(lease.DurationHours) * time.Hour)

	borrowerStats := l.leaderboard[borrowerWallet]
	if borrowerStats.Inventory == nil {
		borrowerStats.Inventory = make(map[string]int)
	}
	borrowerStats.Inventory[fmt.Sprintf("CARD-%d", lease.CardID)]++
	l.leaderboard[borrowerWallet] = borrowerStats

	l.logAdminAudit("LEASE_TAKEN", borrowerWallet, fmt.Sprintf("ID: %s, Price: %.2f", lease.ID, lease.Price))
	l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🤝 <b>LEASE SECURED:</b> You have rented %s."}`, lease.CardName))})
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
				bStats := l.leaderboard[lease.Borrower]
				cardKey := fmt.Sprintf("CARD-%d", lease.CardID)
				if bStats.Inventory[cardKey] > 0 {
					bStats.Inventory[cardKey]--
				}
				l.leaderboard[lease.Borrower] = bStats

				lStats := l.leaderboard[lease.LenderWallet]
				if lStats.Inventory == nil {
					lStats.Inventory = make(map[string]int)
				}
				lStats.Inventory[cardKey]++
				l.leaderboard[lease.LenderWallet] = lStats

				delete(club.Leases, id)
				l.logAdminAudit("LEASE_EXPIRED", lease.LenderWallet, fmt.Sprintf("Card %d returned from %s", lease.CardID, lease.Borrower))
				leasesExpired = true
			}
		}
	}

	if leasesExpired {
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	}
}
