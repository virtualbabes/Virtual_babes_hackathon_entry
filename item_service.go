package main

import (
	"fmt"
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
		l.inventory[data.TargetCardID] = targetCard                                // Update global card cache
		l.persistentCardCache[data.TargetCardID] = targetCard                      // Update persistent cache
		l.updatePlayerPlaystyleTendencies(wallet, false, [2]int{}, []int{}, false) // Update playstyle on item use

	case "Elemental", "Tactical": // Mood Catalyst, Grounded Shield, Rule Breaker, Intel Report (affect MatchState)
		if !inMatch {
			return "", fmt.Errorf("this item can only be used during a match")
		}
		// Delegate to battle_service for in-match effects
		l.applyItemEffectToMatch(match, env.FromID, data.ItemID, data.TargetCardID, data.TargetGridIndex)
		notificationText = fmt.Sprintf("✨ %s activated!", item.Name)
		l.updatePlayerPlaystyleTendencies(wallet, true, [2]int{}, []int{}, false) // Update playstyle on item use in match

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

		l.clubs[playerStats.EmployerClubID] = targetClub
		notificationText = fmt.Sprintf("🛰️ %s deployed in %s's territory!", item.Name, targetClub.Name)

	default:
		return fmt.Sprintf("❓ Used %s. Effect unknown or not yet implemented.", item.Name), nil
	}

	// Update preferred items after successful use
	// This line is here because it applies to all successful item uses.
	playerStats.Playstyle.PreferredItems[data.ItemID] = playerStats.Playstyle.PreferredItems[data.ItemID]*0.9 + 1.0

	return notificationText, nil
}
