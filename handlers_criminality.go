//go:build !js || !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// KidnapData represents the parameters for initiating a Kidnap Gambit.
type KidnapData struct {
	TargetClubID string `json:"target_club_id"`
	RansomAmount uint64 `json:"ransom_amount"` // In micro-VBV
}

// handleKidnapRequest processes the perpetrator's decision to take a card hostage.
func (l *Lobby) handleKidnapRequest(env *Envelope) {
	var data KidnapData
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	perpWallet, ok := l.wallets[env.FromID]
	if !ok {
		return
	}

	targetClub, exists := l.clubs[data.TargetClubID]
	if !exists {
		return
	}

	victimWallet := strings.ToLower(targetClub.OwnerWallet)
	victimStats, victimExists := l.leaderboard[victimWallet]
	if !victimExists {
		return
	}

	// Selection Logic: Target the CEO's Favorite Card or their Rarest Card
	var cardToKidnap ServerCard
	cardFound := false

	// 1. Attempt to use FavoriteCardID if set and present in victim's inventory
	if victimStats.FavoriteCardID != 0 {
		cardKey := fmt.Sprintf("CARD-%d", victimStats.FavoriteCardID)
		if count, hasCard := victimStats.Inventory[cardKey]; hasCard && count > 0 {
			if c, exists := l.inventory[victimStats.FavoriteCardID]; exists { // Also ensure it exists in global inventory
				cardToKidnap = c
				cardFound = true
			}
		}
	}

	// 2. If favorite card not found or not in inventory, fall back to rarest card
	if !cardFound {
		rarest, found := l.findRarestCardInInventory(victimWallet)
		if !found {
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Kidnap Failed: Target has no valuable assets."}`)})
			return
		}
		cardToKidnap = rarest
		cardFound = true
	}

	// Final check: If no card was found after all attempts (should ideally not happen if findRarestCardInInventory is robust)
	if !cardFound {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Kidnap Failed: No suitable card found for target."}`)})
		return
	}

	targetCardID := cardToKidnap.ID

	// Remove the card from the victim's inventory
	cardKey := fmt.Sprintf("CARD-%d", targetCardID)
	victimStats.Inventory[cardKey]--
	if victimStats.Inventory[cardKey] == 0 {
		delete(victimStats.Inventory, cardKey)
	}

	// Move the card to Hostage state
	// Victim View: Their card is being Held Hostage
	if victimStats.HeldHostageCards == nil {
		victimStats.HeldHostageCards = make(map[int]string)
	}
	victimStats.HeldHostageCards[targetCardID] = perpWallet
	l.leaderboard[victimWallet] = victimStats

	// Perp View: They have Kidnapped a card
	perpStats := l.leaderboard[perpWallet]
	if perpStats.KidnappedCards == nil {
		perpStats.KidnappedCards = make(map[int]string)
	}
	perpStats.KidnappedCards[targetCardID] = victimWallet
	l.leaderboard[perpWallet] = perpStats

	// Track Expiration for Insurance Recovery (48 Hours)
	l.activeKidnappings[targetCardID] = KidnapState{
		VictimWallet: victimWallet,
		PerpWallet:   perpWallet,
		ExpiresAt:    time.Now().Add(48 * time.Hour),
	}

	// Update reputation for both parties immediately
	victimStats.Reputation = l.CalculateReputation(victimStats)
	l.leaderboard[victimWallet] = victimStats
	perpStats.Reputation = l.CalculateReputation(perpStats)
	l.leaderboard[perpWallet] = perpStats

	l.logAdminAuditLocked("KIDNAP_GAMBIT", perpWallet, fmt.Sprintf("Victim: %s, CardID: %d, Ransom: %d", victimWallet, targetCardID, data.RansomAmount))

	// Notify Victim
	victimClientID := l.getClientIDFromWalletLocked(victimWallet)
	if victimClientID != "" {
		msg := fmt.Sprintf(`{"text":"🚨 <b>KIDNAP GAMBIT:</b> %s has kidnapped your card #%d! Ransom demanded: %.2f $VBV.", "card_id": %d, "perp_wallet": "%s", "ransom": %d}`,
			perpWallet, targetCardID, float64(data.RansomAmount)/1000000.0, targetCardID, perpWallet, data.RansomAmount)
		l.sendToClientLocked(victimClientID, Envelope{Type: "ransom_demand", Payload: json.RawMessage(msg)})
	}

	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"💼 <b>HOSTAGE SECURED:</b> The target card is now in your custody."}`)})

	msg := l.getLobbyUpdateMsgLocked()
	go func() { l.broadcast <- msg }()
}

// handlePayRansom allows a victim to reclaim their card by paying the demanded VBV.
func (l *Lobby) handlePayRansom(env *Envelope) {
	var data struct {
		CardID       int    `json:"card_id"`
		PerpWallet   string `json:"perp_wallet"`
		RansomAmount uint64 `json:"ransom_amount"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	victimWallet, ok := l.wallets[env.FromID]
	if !ok {
		return
	}

	victimStats := l.leaderboard[victimWallet]
	if victimStats.HeldHostageCards == nil || victimStats.HeldHostageCards[data.CardID] != data.PerpWallet {
		return
	}

	// Financial Transaction
	if l.rewards[victimWallet] < data.RansomAmount {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Payment Failed: Insufficient reward balance."}`)})
		return
	}

	perpStats, perpExists := l.leaderboard[data.PerpWallet] // Check if perp stats exist
	if !perpExists {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Ransom Failed: Perpetrator stats not found."}`)})
		return
	}

	l.rewards[victimWallet] -= data.RansomAmount

	// INDUSTRIAL LOOP: Gross ransom returns to the general Faucet pool.
	// Use integer math with rounding to the nearest micro-unit to prevent dust leaks.
	arenaFeeMicro := (data.RansomAmount*20 + 50) / 100
	perpShareMicro := data.RansomAmount - arenaFeeMicro
	l.rewards[data.PerpWallet] += perpShareMicro

	// Add gross amount to cover future virtual reward liability and capture tax.
	l.faucetBalance += float64(data.RansomAmount) / 1000000.0
	l.applyDynamicScalingLocked()

	// Release Card
	delete(victimStats.HeldHostageCards, data.CardID)
	// Hardening: Restore card instance to victim's inventory
	cardKey := fmt.Sprintf("CARD-%d", data.CardID)
	if victimStats.Inventory == nil {
		victimStats.Inventory = make(map[string]int)
	}
	victimStats.Inventory[cardKey]++

	victimStats.Reputation = l.CalculateReputation(victimStats)
	l.leaderboard[victimWallet] = victimStats

	delete(perpStats.KidnappedCards, data.CardID)
	perpStats.Reputation = l.CalculateReputation(perpStats)
	l.leaderboard[data.PerpWallet] = perpStats

	// Remove from tracking
	delete(l.activeKidnappings, data.CardID)

	l.logAdminAuditLocked("RANSOM_PAID", victimWallet, fmt.Sprintf("Paid %d to %s for Card #%d (Fee: %d)", data.RansomAmount, data.PerpWallet, data.CardID, arenaFeeMicro))

	// Notify Perp
	perpClientID := l.getClientIDFromWalletLocked(data.PerpWallet)
	if perpClientID != "" {
		l.sendToClientLocked(perpClientID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>RANSOM RECEIVED:</b> %s paid %.2f $VBV for card release (Net: %.2f $VBV after Arena fees)."}`, victimWallet, float64(data.RansomAmount)/1000000.0, float64(perpShareMicro)/1000000.0))})
	}

	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"✅ <b>CARD RECLAIMED:</b> Your asset has been returned to your inventory."}`)})

	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// handleReleaseHostage allows a kidnapper to release a hostage card back to the victim voluntarily.
func (l *Lobby) handleReleaseHostage(env *Envelope) {
	var data struct {
		CardID int `json:"card_id"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	perpWallet, ok := l.wallets[env.FromID]
	if !ok {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Release Failed: Your wallet is not registered."}`)})
		return
	}

	perpStats := l.leaderboard[perpWallet]
	if perpStats.KidnappedCards == nil {
		return
	}

	if _, found := perpStats.KidnappedCards[data.CardID]; !found {
		return
	}

	kidnapState, active := l.activeKidnappings[data.CardID]
	if !active || kidnapState.PerpWallet != perpWallet {
		return
	}

	victimWallet := kidnapState.VictimWallet
	victimStats, victimExists := l.leaderboard[victimWallet] // Check if victim stats exist
	if !victimExists {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Release Failed: Victim player stats not found."}`)})
		return
	}
	if victimStats.HeldHostageCards == nil || victimStats.HeldHostageCards[data.CardID] != perpWallet {
		return
	}

	delete(perpStats.KidnappedCards, data.CardID)
	perpStats.Reputation = l.CalculateReputation(perpStats)
	l.leaderboard[perpWallet] = perpStats

	delete(victimStats.HeldHostageCards, data.CardID)
	// Hardening: Restore card instance to victim's inventory
	cardKey := fmt.Sprintf("CARD-%d", data.CardID)
	if victimStats.Inventory == nil {
		victimStats.Inventory = make(map[string]int)
	}
	victimStats.Inventory[cardKey]++

	victimStats.Reputation = l.CalculateReputation(victimStats)
	l.leaderboard[victimWallet] = victimStats

	delete(l.activeKidnappings, data.CardID)

	l.logAdminAuditLocked("HOSTAGE_RELEASED", perpWallet, fmt.Sprintf("Card #%d voluntarily released to %s", data.CardID, victimWallet))

	if vCID := l.getClientIDFromWalletLocked(victimWallet); vCID != "" {
		l.sendToClientLocked(vCID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"✅ <b>HOSTAGE RELEASED:</b> Card #%d has been returned by the kidnapper."}`, data.CardID))})
	}
	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"✅ <b>HOSTAGE RELEASED:</b> Card #%d has been returned to the victim."}`, data.CardID))})

	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// handleBailCard allows a player to pay a fine to release a jailed card.
func (l *Lobby) handleBailCard(env *Envelope) {
	var data BailCardData
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		log.Printf("[CRIMINALITY] Invalid bail_card payload from %s: %v\n", env.FromID, err)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	playerWallet, ok := l.wallets[env.FromID]
	if !ok {
		log.Printf("[CRIMINALITY] Bail failed: Sender %s not connected.\n", env.FromID)
		return
	}

	club, exists := l.clubs[data.ClubID]
	if !exists {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Bail Failed: Club not found."}`)})
		return
	}

	jailedCard, isJailed := club.Jail[data.CardID]
	if !isJailed {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Bail Failed: Card not found in this club's jail."}`)})
		return
	}

	// Ensure the card belongs to the player attempting to bail it
	playerStats := l.leaderboard[playerWallet]
	if clubIDForCard, ok := playerStats.JailedCards[data.CardID]; !ok || clubIDForCard != data.ClubID {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Bail Failed: You do not own this jailed card."}`)})
		return
	}

	// Bail amount: 200 $VBV (micro-units)
	const bailAmountMicro = 200 * 1000000
	bailAmountBase := float64(bailAmountMicro) / 1000000.0

	// Verify payment transaction
	l.mutex.RLock()
	voiConfig, voiOk := l.availableNetworks["Voi Mainnet"]
	avoiAssetID := l.avoiAssetID
	vaultAddr := l.vaultAddress
	l.mutex.RUnlock()

	if !voiOk {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Bail Failed: Voi network configuration missing."}`)})
		return
	}

	assetID := voiConfig.AssetID
	if assetID == "" {
		assetID = voiConfig.AppID
	}
	verifyNet := "Voi"
	if strings.EqualFold(data.Network, "ALGO") {
		l.mutex.RLock()
		algoCfg, hasAlgo := l.availableNetworks["Algorand Mainnet"]
		l.mutex.RUnlock()
		if hasAlgo && algoCfg.AssetID != "" {
			assetID = algoCfg.AssetID
		} else {
			assetID = avoiAssetID
		}
		verifyNet = "Algorand"
	}

	// PILLAR 3: Specific Purpose Verification for underworld bail
	verified, _, err := l.verifyBuyInTransaction(verifyNet, data.TxID, bailAmountMicro, assetID, playerWallet, vaultAddr, "BAIL_PAYMENT:")
	if err != nil || !verified {
		log.Printf("[CRIMINALITY] Bail payment verification failed for %s (Card %d): %v\n", playerWallet, data.CardID, err)
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Bail Failed: Payment verification failed or insufficient amount."}`)})
		return
	}

	// Distribute bail proceeds to the jailing club's treasury
	club.Treasury += bailAmountBase
	club.LastActivity = time.Now() // Revenue counts as activity

	// Release card from jail
	delete(club.Jail, data.CardID)

	// Remove from player's JailedCards and add back to Inventory
	delete(playerStats.JailedCards, data.CardID)
	if playerStats.Inventory == nil {
		playerStats.Inventory = make(map[string]int)
	}
	playerStats.Inventory[fmt.Sprintf("CARD-%d", data.CardID)]++
	playerStats.Reputation = l.CalculateReputation(playerStats)
	l.leaderboard[playerWallet] = playerStats

	l.logAdminAuditLocked("CARD_BAILED", playerWallet, fmt.Sprintf("Card #%d bailed from Club %s for %.2f $VBV", data.CardID, club.Name, bailAmountBase))
	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"✅ <b>BAIL PAID:</b> Your card '%s' has been released from %s's jail!"}`, escapeHTML(jailedCard.Name), escapeHTML(club.Name)))})

	// Notify club owner/members (optional, but good for transparency)
	clubOwnerClientID := l.getClientIDFromWalletLocked(club.OwnerWallet)
	if clubOwnerClientID != "" {
		l.sendToClientLocked(clubOwnerClientID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>BAIL RECEIVED:</b> Club %s received %.2f $VBV for card #%d."}`, escapeHTML(club.Name), bailAmountBase, data.CardID))})
	}

	// Broadcast update to refresh UI lists
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// processInsuranceRecovery checks for cards that have been held hostage for too long.
func (l *Lobby) processInsuranceRecovery() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	now := time.Now()
	var toRelease []int
	for cardID, state := range l.activeKidnappings {
		if now.After(state.ExpiresAt) {
			toRelease = append(toRelease, cardID)
		}
	}

	if len(toRelease) == 0 {
		return
	}

	for _, cardID := range toRelease {
		state := l.activeKidnappings[cardID]

		// Automatic Return: No VBV exchange
		victimStats := l.leaderboard[state.VictimWallet]
		delete(victimStats.HeldHostageCards, cardID)
		// CRITICAL FIX: Add the card back to the victim's inventory
		cardKey := fmt.Sprintf("CARD-%d", cardID)
		if victimStats.Inventory == nil {
			victimStats.Inventory = make(map[string]int)
		}
		victimStats.Inventory[cardKey]++ // Increment count, assuming it was decremented by 1

		victimStats.Reputation = l.CalculateReputation(victimStats)
		l.leaderboard[state.VictimWallet] = victimStats

		perpStats := l.leaderboard[state.PerpWallet]
		delete(perpStats.KidnappedCards, cardID)
		perpStats.Reputation = l.CalculateReputation(perpStats)
		l.leaderboard[state.PerpWallet] = perpStats

		delete(l.activeKidnappings, cardID)

		l.logAdminAuditLocked("INSURANCE_RECOVERY", state.VictimWallet, fmt.Sprintf("Card #%d automatically returned from %s", cardID, state.PerpWallet))

		// Notify Players
		if vCID := l.getClientIDFromWalletLocked(state.VictimWallet); vCID != "" {
			l.sendToClientLocked(vCID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"✅ <b>INSURANCE RECOVERY:</b> Your kidnapped card #%d has been returned."}`, cardID))})
		}
		if pCID := l.getClientIDFromWalletLocked(state.PerpWallet); pCID != "" {
			l.sendToClientLocked(pCID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🚨 <b>HOSTAGE ESCAPED:</b> Card #%d has returned to its owner via Insurance Recovery."}`, cardID))})
		}
	}

	// Broadcast update to refresh UI lists
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}
