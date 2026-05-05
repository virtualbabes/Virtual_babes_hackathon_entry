package main

import (
	"encoding/json"
	"fmt"
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
	if !ok { return }

	targetClub, exists := l.clubs[data.TargetClubID]
	if !exists { return }

	victimWallet := strings.ToLower(targetClub.OwnerWallet)
	victimStats, victimExists := l.leaderboard[victimWallet]
	if !victimExists { return }

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
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Kidnap Failed: Target has no valuable assets."}`)})
			return
		}
		cardToKidnap = rarest
		cardFound = true
	}

	// Final check: If no card was found after all attempts (should ideally not happen if findRarestCardInInventory is robust)
	if !cardFound {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Kidnap Failed: No suitable card found for target."}`)})
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
	if victimStats.HeldHostageCards == nil { victimStats.HeldHostageCards = make(map[int]string) }
	victimStats.HeldHostageCards[targetCardID] = perpWallet
	l.leaderboard[victimWallet] = victimStats

	// Perp View: They have Kidnapped a card
	perpStats := l.leaderboard[perpWallet]
	if perpStats.KidnappedCards == nil { perpStats.KidnappedCards = make(map[int]string) }
	perpStats.KidnappedCards[targetCardID] = victimWallet
	l.leaderboard[perpWallet] = perpStats

	// Track Expiration for Insurance Recovery (48 Hours)
	l.activeKidnappings[targetCardID] = KidnapState{
		VictimWallet: victimWallet,
		PerpWallet:   perpWallet,
		ExpiresAt:    time.Now().Add(48 * time.Hour),
	}

	l.logAdminAudit("KIDNAP_GAMBIT", perpWallet, fmt.Sprintf("Victim: %s, CardID: %d, Ransom: %d", victimWallet, targetCardID, data.RansomAmount))

	// Notify Victim
	victimClientID := l.getClientIDFromWallet(victimWallet)
	if victimClientID != "" {
		msg := fmt.Sprintf(`{"text":"🚨 <b>KIDNAP GAMBIT:</b> %s has kidnapped your card #%d! Ransom demanded: %.2f $VBV.", "card_id": %d, "perp_wallet": "%s", "ransom": %d}`, 
			perpWallet, targetCardID, float64(data.RansomAmount)/1000000.0, targetCardID, perpWallet, data.RansomAmount)
		l.sendToClient(victimClientID, Envelope{Type: "ransom_demand", Payload: json.RawMessage(msg)})
	}

	l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"💼 <b>HOSTAGE SECURED:</b> The target card is now in your custody."}`)})
	
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
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
	if !ok { return }

	victimStats := l.leaderboard[victimWallet]
	if victimStats.HeldHostageCards == nil || victimStats.HeldHostageCards[data.CardID] != data.PerpWallet {
		return
	}

	// Financial Transaction
	if l.rewards[victimWallet] < data.RansomAmount {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Payment Failed: Insufficient reward balance."}`)})
		return
	}

	l.rewards[victimWallet] -= data.RansomAmount
	l.rewards[data.PerpWallet] += data.RansomAmount

	// Release Card
	delete(victimStats.HeldHostageCards, data.CardID)
	l.leaderboard[victimWallet] = victimStats

	perpStats := l.leaderboard[data.PerpWallet]
	delete(perpStats.KidnappedCards, data.CardID)
	l.leaderboard[data.PerpWallet] = perpStats

	// Remove from tracking
	delete(l.activeKidnappings, data.CardID)

	l.logAdminAudit("RANSOM_PAID", victimWallet, fmt.Sprintf("Paid %d to %s for Card #%d", data.RansomAmount, data.PerpWallet, data.CardID))

	// Notify Perp
	perpClientID := l.getClientIDFromWallet(data.PerpWallet)
	if perpClientID != "" {
		l.sendToClient(perpClientID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>RANSOM RECEIVED:</b> %s paid %.2f $VBV for the release of their card."}`, victimWallet, float64(data.RansomAmount)/1000000.0))})
	}

	l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"✅ <b>CARD RECLAIMED:</b> Your asset has been returned to your inventory."}`)})
	
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
	if !ok { return }

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
	victimStats := l.leaderboard[victimWallet]
	if victimStats.HeldHostageCards == nil || victimStats.HeldHostageCards[data.CardID] != perpWallet {
		return
	}

	delete(perpStats.KidnappedCards, data.CardID)
	l.leaderboard[perpWallet] = perpStats
	delete(victimStats.HeldHostageCards, data.CardID)
	l.leaderboard[victimWallet] = victimStats
	delete(l.activeKidnappings, data.CardID)

	l.logAdminAudit("HOSTAGE_RELEASED", perpWallet, fmt.Sprintf("Card #%d voluntarily released to %s", data.CardID, victimWallet))

	if vCID := l.getClientIDFromWallet(victimWallet); vCID != "" {
		l.sendToClient(vCID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"✅ <b>HOSTAGE RELEASED:</b> Card #%d has been returned by the kidnapper."}`, data.CardID))})
	}
	l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"✅ <b>HOSTAGE RELEASED:</b> Card #%d has been returned to the victim."}`, data.CardID))})

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
		
		l.leaderboard[state.VictimWallet] = victimStats

		perpStats := l.leaderboard[state.PerpWallet]
		delete(perpStats.KidnappedCards, cardID)
		l.leaderboard[state.PerpWallet] = perpStats

		delete(l.activeKidnappings, cardID)

		l.logAdminAudit("INSURANCE_RECOVERY", state.VictimWallet, fmt.Sprintf("Card #%d automatically returned from %s", cardID, state.PerpWallet))

		// Notify Players
		if vCID := l.getClientIDFromWallet(state.VictimWallet); vCID != "" {
			l.sendToClient(vCID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"✅ <b>INSURANCE RECOVERY:</b> Your kidnapped card #%d has been returned."}`, cardID))})
		}
		if pCID := l.getClientIDFromWallet(state.PerpWallet); pCID != "" {
			l.sendToClient(pCID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🚨 <b>HOSTAGE ESCAPED:</b> Card #%d has returned to its owner via Insurance Recovery."}`, cardID))})
		}
	}

	// Broadcast update to refresh UI lists
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}