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

	// SUCCESS CHANCE CALCULATION
	successChance := 0.50
	securityLevel := float64(targetClub.Mojo) / 10.0

	for _, role := range targetClub.Staff {
		if role == "Security" {
			securityLevel += 15.0
		}
	}

	for buffID, itemID := range targetClub.ActiveBuffs {
		if strings.HasPrefix(buffID, "TRAP_") {
			if item, exists := GlobalShopRegistry[itemID]; exists {
				successChance += item.HeistSuccessModifier
			}
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
	canKidnap := false

	if roll < successChance {
		status = "success"
		if playerStats.Cunning >= 3 && rand.Float64() < 0.25 {
			canKidnap = true
		}

		loot := targetClub.Treasury * 0.10
		if loot > 500 {
			loot = 500
		}
		playerStats.Playstyle.RiskTolerance += 0.05
		playerStats.HeistAttempts++
		targetClub.Treasury -= loot
		targetClub.LastActivity = time.Now()
		playerStats.WantedLevel += 5
		playerStats.Cunning += 1

		go l.unlockAchievement(wallet, "FIRST_HEIST")
	} else {
		status = "failure"
		playerStats.WantedLevel += 15
		playerStats.Playstyle.RiskTolerance += 0.10
		playerStats.HeistAttempts++

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
				delete(playerStats.Inventory, fmt.Sprintf("CARD-%d", rarestCard.ID))
				if playerStats.JailedCards == nil {
					playerStats.JailedCards = make(map[int]string)
				}
				playerStats.JailedCards[rarestCard.ID] = targetClub.ID
				l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🚨 <b>GUARD DOG BUST:</b> You were caught by a Bio-Guard Dog! Your rarest card (%s) has been jailed by %s."}`, rarestCard.Name, targetClub.Name))})
			}
		}
	}

	l.leaderboard[wallet] = playerStats
	l.logAdminAudit("HEIST_ATTEMPT", wallet, fmt.Sprintf("Target: %s, Result: %s", data.TargetClubID, status))

	response, _ := json.Marshal(map[string]interface{}{
		"status":          status,
		"wanted_level":    playerStats.WantedLevel,
		"cunning":         playerStats.Cunning,
		"playstyle":       playerStats.Playstyle,
		"heist_attempts":  playerStats.HeistAttempts,
		"kidnap_eligible": canKidnap,
		"target_club_id":  data.TargetClubID,
	})
	l.sendToClient(env.FromID, Envelope{Type: "heist_result", Payload: response})
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
		BuffExpirations: make(map[string]time.Time),
		CreatedAt:       time.Now(), Jail: make(map[int]ServerCard), LastActivity: time.Now(),
	}
	newClub.Staff[strings.ToLower(wallet)] = "CEO"
	l.clubs[clubID] = newClub
	l.mutex.Unlock()

	l.logAdminAudit("CLUB_CREATED", wallet, fmt.Sprintf("Name: %s, ID: %s", data.Name, clubID))
	l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🏛️ Club '%s' successfully founded!"}`, data.Name))})
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
	if data.Network == "ALGO" {
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
		club.Members[strings.ToLower(wallet)] = time.Now()
		club.Treasury += 250.0
		club.LastActivity = time.Now()
		l.mutex.Unlock()
		l.logAdminAudit("CLUB_JOIN", wallet, fmt.Sprintf("Club: %s", data.ClubID))
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🤝 Welcome to %s!"}`, club.Name))})
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

	l.mutex.Lock()
	defer l.mutex.Unlock()

	ownerWallet, ok := l.wallets[env.FromID]
	if !ok {
		return
	}

	club, exists := l.clubs[data.ClubID]
	if !exists || strings.ToLower(club.OwnerWallet) != strings.ToLower(ownerWallet) {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Territory Purchase Failed: Unauthorized or Club not found."}`)})
		return
	}

	for _, existingClub := range l.clubs {
		for _, t := range existingClub.Territories {
			if t == data.TerritoryID {
				l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Territory Purchase Failed: District already claimed."}`)})
				return
			}
		}
	}

	purchaseCost := 2500.0
	voiConfig := l.availableNetworks["Voi Mainnet"]
	assetID := voiConfig.AssetID
	verifyNet := "Voi"
	if data.Network == "ALGO" {
		assetID = l.avoiAssetID
		verifyNet = "Algorand"
	}

	verified, _, err := l.verifyBuyInTransaction(verifyNet, data.TxID, uint64(purchaseCost*1000000), assetID, ownerWallet, l.vaultAddress)
	if err != nil || !verified {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Territory Purchase Failed: Payment verification failed."}`)})
		return
	}

	club.Territories = append(club.Territories, data.TerritoryID)
	l.clubs[data.ClubID] = club
	club.LastActivity = time.Now()

	l.logAdminAudit("TERRITORY_PURCHASED", ownerWallet, fmt.Sprintf("Club: %s (%s), Territory: %s", club.Name, club.ID, data.TerritoryID))
	l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🗺️ Club '%s' has acquired %s!"}`, club.Name, data.TerritoryID))})

	go l.refreshRegionalRoles()
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
	club, exists := l.clubs[data.ClubID]
	if !ok || !exists {
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

	totalCost := item.Price * float64(data.Quantity)
	if club.Treasury < totalCost {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"❌ Restock Failed: Insufficient Treasury funds."}`))})
		return
	}

	club.Treasury -= totalCost

	// Ensure map is initialized even if old data exists
	if club.Inventory == nil {
		club.Inventory = make(map[string]int)
	}

	club.LastActivity = time.Now()
	club.Inventory[data.ItemID] += data.Quantity

	l.logAdminAudit("CLUB_RESTOCK", ownerWallet, fmt.Sprintf("Club: %s, Item: %s, Qty: %d, Cost: %.2f", club.Name, data.ItemID, data.Quantity, totalCost))
	l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"📦 <b>RESTOCK COMPLETE:</b> Added %d units of %s to inventory."}`, data.Quantity, item.Name))})
}

// distributeCourthouseFineToClubs distributes a portion of the fine among clubs and governors.
func (l *Lobby) distributeCourthouseFineToClubs(amount float64) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

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
		}
	}

	sharePerClub := remainingPool / float64(len(l.clubs))
	for _, club := range l.clubs {
		club.Treasury += sharePerClub
	}
}
