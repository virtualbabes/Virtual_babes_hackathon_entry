//go:build !js && !wasm

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

var ErrAttackerAlreadyHoldsHostage = errors.New("you already have an active kidnapping operation against this target")

// RegisterKidnap validates and records a new hostage event in the registry.
// PILLAR 3: Multi-Slot Attacker Isolation.
func (vr *VictimRegistry) RegisterKidnap(victim string, attacker string, assetID uint64, ransom uint64) error {
	if vr == nil {
		return errors.New("victim registry not initialized")
	}

	vr.Mu.Lock()
	defer vr.Mu.Unlock()

	// Ensure the victim entry exists in the top-level map
	if vr.ActiveKidnaps == nil {
		vr.ActiveKidnaps = make(map[string]map[string]HostageSituation)
	}
	if _, exists := vr.ActiveKidnaps[victim]; !exists {
		vr.ActiveKidnaps[victim] = make(map[string]HostageSituation)
	}

	// Check if this SPECIFIC attacker already occupies a slot against this victim
	now := time.Now().Unix()
	if existingSituation, exists := vr.ActiveKidnaps[victim][attacker]; exists {
		if now < existingSituation.ExpirationTime {
			return ErrAttackerAlreadyHoldsHostage
		}
		// Clean up expired situation
		delete(vr.ActiveKidnaps[victim], attacker)
	}

	// Register the new high-stakes capture
	vr.ActiveKidnaps[victim][attacker] = HostageSituation{
		AttackerAddress: attacker,
		AssetID:         assetID,
		RansomAmount:    ransom,
		ExpirationTime:  time.Now().Add(48 * time.Hour).Unix(),
	}

	return nil
}

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
	l.ensurePlayerStatsMapsInitialized(victimWallet)
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

	// PILLAR 3: Multi-Slot Attacker Isolation.
	// Register the kidnap attempt. This will fail if the attacker already holds a hostage from this victim.
	// This logic replaces the previous "one active crisis per victim" rule which was vulnerable to self-kidnapping.
	if err := l.victimRegistry.RegisterKidnap(victimWallet, perpWallet, uint64(targetCardID), data.RansomAmount); err != nil {
		msg := fmt.Sprintf(`{"text":"❌ Kidnap Failed: %v"}`, err)
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(msg)})
		return
	}

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

		rxp, pair, isRival := EvaluateCrossCareerXP("Kidnapper", victimStats.JobRole, kidnapBase, perpStats, victimStats)
		if isRival && rxp > int(kidnapBase) {
			scaledRBonus := l.leaderboard[perpWallet].CareerXP.ComputeScaledXP(uint64(rxp-int(kidnapBase)), "Kidnapper")
			l.TrackCareerXP(perpWallet, "Kidnapper", scaledRBonus)
			_ = pair // Suppress unused warning if needed
		}
	}

	// P2-A: Rival Pair — Warden (Justice D3) gains XP monitoring kidnapping events. Task 4301.
	wardenBase := uint64(25)
	scaledWPNP := l.leaderboard[perpWallet].CareerXP.ComputeScaledXP(wardenBase, "Warden")
	l.TrackCareerXP(perpWallet, "Warden", scaledWPNP)

	// Underworld #6 Racketeer — earns XP extorting protection money per kidnapping. Task 4301.
	rackBase := uint64(40)
	scaledRacNP := l.leaderboard[perpWallet].CareerXP.ComputeScaledXP(rackBase, "Racketeer")
	l.TrackCareerXP(perpWallet, "Racketeer", scaledRacNP)

	// P2-A: Rival Pair — Smuggler ↔ Sector Peacekeeper (antagonistic). Task 4301.
	if perpStats.CareerXP != nil {
		rxp3, _, _ := EvaluateCrossCareerXP("Smuggler", victimStats.JobRole, 30, perpStats, victimStats)
		if rxp3 > 30 {
			scaledSMP := l.leaderboard[perpWallet].CareerXP.ComputeScaledXP(uint64(rxp3-30), "Smuggler")
			l.TrackCareerXP(perpWallet, "Smuggler", scaledSMP)
		}
	}

	// Justice D5 Forensic Analyst — forensic trace reward at detection time. Task 4301.
	if l.leaderboard != nil {
		for statsWallet := range l.leaderboard {
			ws := l.leaderboard[statsWallet]
			if ws.JobRole == "Forensic Analyst" {
				scaledFANP := ws.CareerXP.ComputeScaledXP(60, "Forensic Analyst")
				l.TrackCareerXP(statsWallet, "Forensic Analyst", scaledFANP)
			}
		}
	}

	// Justice D3 Warden — monitors all hostage activity across the sector. Task 4301.
	if l.leaderboard != nil {
		for statsWallet := range l.leaderboard {
			ws := l.leaderboard[statsWallet]
			if ws.JobRole == "Warden" && statsWallet != perpWallet { // Avoid double-counting with Warden line above
				scaledWP2NP := ws.CareerXP.ComputeScaledXP(25, "Warden")
				l.TrackCareerXP(statsWallet, "Warden", scaledWP2NP)
			}
		}
	}

	// Justice D6 Sector Peacekeeper — maintains sector order during criminal events. Task 4301.
	if l.leaderboard != nil {
		for statsWallet := range l.leaderboard {
			ws := l.leaderboard[statsWallet]
			if ws.JobRole == "Sector Peacekeeper" && statsWallet != perpWallet { // Avoid double-counting if perp is SPK
				scaledSPKP := ws.CareerXP.ComputeScaledXP(20, "Sector Peacekeeper")
				l.TrackCareerXP(statsWallet, "Sector Peacekeeper", scaledSPKP)
			}
		}
	}

	// Track this as an active kidnapping for Hostage Host progression tracking. Task 4301.
	if perpStats.ActiveHostageCount == 0 {
		// First hostage — Hostage Host career milestone
		scaledHHNP := l.leaderboard[perpWallet].CareerXP.ComputeScaledXP(50, "Hostage Host")
		l.TrackCareerXP(perpWallet, "Hostage Host", scaledHHNP)
	}
	perpStats.ActiveHostageCount++

	// PILLAR 3: Underworld Contract Completion (CONTRACT-005).
	// Objective: Successfully execute a Kidnap Gambit.
	if perpStats.ActiveUnderworldContractID == "CONTRACT-005" {
		const rewardMicro = 3000 * 1000000
		l.playerBalances[perpWallet] += rewardMicro
		perpStats.ActiveUnderworldContractID = ""
		l.logAdminAuditLocked("CONTRACT_COMPLETED", perpWallet, "ID: CONTRACT-005, Payout: 3000.00")
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> High-value ransom secured. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
		l.applyDynamicScalingLocked()
	}

	// PILLAR 3: Underworld Contract Completion (CONTRACT-010).
	// Objective: Successfully execute a Kidnap Gambit against a Regional Governor's favorite card.
	if perpStats.ActiveUnderworldContractID == "CONTRACT-010" &&
		l.clubService.IsClubRegionalLocked(l, targetClub) &&
		victimStats.FavoriteCardID == targetCardID {
		const rewardMicro = 5000 * 1000000
		l.playerBalances[perpWallet] += rewardMicro
		perpStats.ActiveUnderworldContractID = ""
		l.logAdminAuditLocked("CONTRACT_COMPLETED", perpWallet, "ID: CONTRACT-010, Payout: 5000.00")
		l.sendToClientLocked(env.FromID, Envelope{
			Type:    "admin_notification",
			Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> Governor's favorite card secured. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0)),
		})
		l.applyDynamicScalingLocked()
	}

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
	if l.playerBalances[victimWallet] < data.RansomAmount {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Payment Failed: Insufficient reward balance."}`)})
		return
	}

	perpStats, perpExists := l.leaderboard[data.PerpWallet] // Check if perp stats exist
	if !perpExists {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Ransom Failed: Perpetrator stats not found."}`)})
		return
	}

	l.playerBalances[victimWallet] -= data.RansomAmount

	// PILLAR 3: Identity Hardening. Ensure records are fully initialized before state modification.
	l.ensurePlayerStatsMapsInitialized(victimWallet)
	l.ensurePlayerStatsMapsInitialized(data.PerpWallet)

	// PILLAR 2: Industrial Loop (Token-Sink Router migration).
	// Shift liabilities: Victim -> Perp (80%) + Faucet (20% Laundering Tax).
	arenaFeeMicro := (data.RansomAmount*20 + 50) / 100
	perpShareMicro := data.RansomAmount - arenaFeeMicro
	l.playerBalances[data.PerpWallet] += perpShareMicro

	if l.tokenSinkRouter != nil {
		matrix := RevenueSplitMatrix{FaucetShare: 1.0, ClubShare: 0.0, GovernanceShare: 0.0}
		// PILLAR 2: Forensic Visibility. Use global sector context for ransom taxes.
		_ = l.tokenSinkRouter.RouteCriminalTax("sector_all", arenaFeeMicro, matrix, 0, "")

		// Sync float balance with authoritative micro-unit total
		l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0
	}

	// PILLAR 3: Financial Proof.
	// Record successful ransom on-chain for the immutable audit trail.
	ransomDetails := map[string]interface{}{
		"card_id":     data.CardID,
		"victim":      victimWallet,
		"perp":        data.PerpWallet,
		"laundry_tax": float64(arenaFeeMicro) / 1000000.0,
		"net_payout":  float64(perpShareMicro) / 1000000.0,
		"ts":          time.Now().Unix(),
	}

	l.applyDynamicScalingLocked()

	// Release Card
	delete(victimStats.HeldHostageCards, data.CardID)
	// Hardening: Restore card instance to victim's inventory
	cardKey := fmt.Sprintf("CARD-%d", data.CardID)
	victimStats.Inventory[cardKey]++

	victimStats.Reputation = l.CalculateReputation(victimStats)
	l.leaderboard[victimWallet] = victimStats

	delete(perpStats.KidnappedCards, data.CardID)
	perpStats.Reputation = l.CalculateReputation(perpStats)
	l.leaderboard[data.PerpWallet] = perpStats

	// Remove from tracking
	delete(l.activeKidnappings, data.CardID)

	// PILLAR 3: Modular Authority. Clear the record from the multi-slot registry.
	l.victimRegistry.Mu.Lock()
	if attackers, exists := l.victimRegistry.ActiveKidnaps[victimWallet]; exists {
		delete(attackers, data.PerpWallet)
		if len(attackers) == 0 {
			delete(l.victimRegistry.ActiveKidnaps, victimWallet)
		}
	}
	l.victimRegistry.Mu.Unlock()

	l.logAdminAuditLocked("RANSOM_PAID", victimWallet, fmt.Sprintf("Paid %d to %s for Card #%d (Fee: %d)", data.RansomAmount, data.PerpWallet, data.CardID, arenaFeeMicro))

	// Dispatch on-chain log for financial verification
	go func(rd interface{}) {
		jsonPayload, _ := json.Marshal(rd)
		l.sendNoteTx(fmt.Sprintf("VBT_RANSOM_LOG:%s", string(jsonPayload)))
	}(ransomDetails)

	// Notify Perp
	perpClientID := l.getClientIDFromWalletLocked(data.PerpWallet)
	if perpClientID != "" {
		l.sendToClientLocked(perpClientID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>RANSOM RECEIVED:</b> %s paid %.2f $VBV for card release (Net: %.2f $VBV after Arena fees)."}`, victimWallet, float64(data.RansomAmount)/1000000.0, float64(perpShareMicro)/1000000.0))})
	}

	// Justice D3 Warden — earns XP monitoring financial flows during releases. Task 4301.
	for statsWallet := range l.leaderboard {
		ws := l.leaderboard[statsWallet]
		if ws.JobRole == "Warden" && statsWallet != victimWallet && statsWallet != data.PerpWallet {
			scaledWPNP2 := ws.CareerXP.ComputeScaledXP(15, "Warden")
			l.TrackCareerXP(statsWallet, "Warden", scaledWPNP2)
		}
	}

	// Justice D5 Forensic Analyst — traces financial patterns during ransom events. Task 4301.
	for statsWallet := range l.leaderboard {
		ws := l.leaderboard[statsWallet]
		if ws.JobRole == "Forensic Analyst" && statsWallet != victimWallet && statsWallet != data.PerpWallet {
			scaledFANP2 := ws.CareerXP.ComputeScaledXP(45, "Forensic Analyst")
			l.TrackCareerXP(statsWallet, "Forensic Analyst", scaledFANP2)
		}
	}

	// P2-E: Bounty Hunter ↔ Kidnapper — non-combat XP trigger for ransom payment resolution.
	if perpStats.CareerXP != nil && (perpStats.JobRole == "Kidnapper" || CareerHasRole(perpStats.CareerXP, "Kidnapper")) {
		for statsWallet := range l.leaderboard {
			ws := l.leaderboard[statsWallet]
			if ws.JobRole == "BountyHunter" && statsWallet != perpWallet && statsWallet != victimWallet {
				rivalXP, _, isRival := EvaluateCrossCareerXP("BountyHunter", perpStats.JobRole, data.RansomAmount/10, &ws, perpStats)
				if isRival && rivalXP > 0 {
					scaledRBonus := ws.CareerXP.ComputeScaledXP(uint64(rivalXP), "BountyHunter")
					l.TrackCareerXP(statsWallet, "BountyHunter", scaledRBonus)
					l.logAdminAuditLocked("CAREER_BOUNTY_HUNTER_KIDNAPPER_RANSOM", statsWallet, fmt.Sprintf("+%d XP (Kidnapper ransom resolution)", rivalXP))
				}
			}
		}
	}

	// P2-E: Sector Peacekeeper ↔ Smuggler — non-combat XP trigger for hostage release monitoring.
	if perpStats.CareerXP != nil && (perpStats.JobRole == "Smuggler" || CareerHasRole(perpStats.CareerXP, "Smuggler")) {
		for statsWallet := range l.leaderboard {
			ws := l.leaderboard[statsWallet]
			if ws.JobRole == "SectorPeacekeeper" && statsWallet != perpWallet && statsWallet != victimWallet {
				rivalXP2, _, isRival2 := EvaluateCrossCareerXP("SectorPeacekeeper", perpStats.JobRole, 30, &ws, perpStats)
				if isRival2 && rivalXP2 > 30 {
					scaledSPBonus := ws.CareerXP.ComputeScaledXP(uint64(rivalXP2-30), "SectorPeacekeeper")
					l.TrackCareerXP(statsWallet, "SectorPeacekeeper", scaledSPBonus)
					l.logAdminAuditLocked("CAREER_SECTOR_PEACEKEEPER_SMUGGLER_RANSOM", statsWallet, fmt.Sprintf("+%d XP (Smuggler ransom resolution)", rivalXP2-30))
				}
			}
		}
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

	// PILLAR 3: Identity Hardening.
	l.ensurePlayerStatsMapsInitialized(perpWallet)
	l.ensurePlayerStatsMapsInitialized(victimWallet)

	if !victimExists || victimStats.HeldHostageCards[data.CardID] != perpWallet {
		return
	}

	delete(perpStats.KidnappedCards, data.CardID)
	perpStats.Reputation = l.CalculateReputation(perpStats)
	l.leaderboard[perpWallet] = perpStats

	delete(victimStats.HeldHostageCards, data.CardID)
	// Hardening: Restore card instance to victim's inventory
	cardKey := fmt.Sprintf("CARD-%d", data.CardID)
	victimStats.Inventory[cardKey]++

	victimStats.Reputation = l.CalculateReputation(victimStats)
	l.leaderboard[victimWallet] = victimStats

	delete(l.activeKidnappings, data.CardID)

	// PILLAR 3: Modular Authority. Clear the record from the multi-slot registry.
	l.victimRegistry.Mu.Lock()
	if attackers, exists := l.victimRegistry.ActiveKidnaps[victimWallet]; exists {
		delete(attackers, perpWallet)
		if len(attackers) == 0 {
			delete(l.victimRegistry.ActiveKidnaps, victimWallet)
		}
	}
	l.victimRegistry.Mu.Unlock()

	l.logAdminAuditLocked("HOSTAGE_RELEASED", perpWallet, fmt.Sprintf("Card #%d voluntarily released to %s", data.CardID, victimWallet))

	// Justice D3 Warden — monitors all hostage releases across the sector. Task 4301.
	for statsWallet := range l.leaderboard {
		ws := l.leaderboard[statsWallet]
		if ws.JobRole == "Warden" && statsWallet != perpWallet && statsWallet != victimWallet {
			scaledWPNP3 := ws.CareerXP.ComputeScaledXP(20, "Warden")
			l.TrackCareerXP(statsWallet, "Warden", scaledWPNP3)
		}
	}

	// Justice D6 Sector Peacekeeper — maintains order during criminal resolutions. Task 4301.
	for statsWallet := range l.leaderboard {
		ws := l.leaderboard[statsWallet]
		if ws.JobRole == "Sector Peacekeeper" && statsWallet != perpWallet && statsWallet != victimWallet {
			scaledSPKP2 := ws.CareerXP.ComputeScaledXP(15, "Sector Peacekeeper")
			l.TrackCareerXP(statsWallet, "Sector Peacekeeper", scaledSPKP2)
		}
	}

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

	// PILLAR 2: Industrial Loop (Token-Sink Router migration).
	// 1. Physically increment the vault total to reflect the confirmed on-chain inflow.
	l.faucetBalanceMicro += bailAmountMicro
	l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0
	l.applyDynamicScalingLocked() // PILLAR 2: Synchronize scaling with physical inflow

	// 2. Redistribute the proceeds: 100% to the jailing club's treasury liability.
	if l.tokenSinkRouter != nil {
		matrix := RevenueSplitMatrix{FaucetShare: 0.0, ClubShare: 1.0, GovernanceShare: 0.0}
		numericID, _ := strconv.ParseUint(strings.TrimPrefix(club.ID, "CLUB-"), 10, 64)
		_ = l.tokenSinkRouter.RouteCriminalTax("BAIL_PAYMENT", bailAmountMicro, matrix, numericID, "")

		// PILLAR 2: UI Parity Sync. 
		if node, ok := l.tokenSinkRouter.ActiveClubs[numericID]; ok {
			club.Treasury = float64(node.TreasuryBalance) / 1000000.0
		}
	}

	club.LastActivity = time.Now() // Revenue counts as activity

	// Release card from jail
	delete(club.Jail, data.CardID)

	// Remove from player's JailedCards and add back to Inventory
	delete(playerStats.JailedCards, data.CardID)
	if playerStats.Inventory == nil {
		playerStats.Inventory = make(map[string]int)
	}
	playerStats.Inventory[fmt.Sprintf("CARD-%d", data.CardID)]++

	// PILLAR 3: Justice Mission Completion (MISSION-002)
	// Facilitate Legal Rehabilitation: Process a Bail payment for any card currently held in custody.
	if playerStats.ActiveJusticeMissionID == "MISSION-002" {
		const rewardMicro = 500 * 1000000
		l.playerBalances[playerWallet] += rewardMicro
		playerStats.ActiveJusticeMissionID = ""
		l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", playerWallet, "ID: MISSION-002, Payout: 500.00")
		l.sendToClientLocked(env.FromID, Envelope{
			Type:    "admin_notification",
			Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> Legal rehabilitation facilitated. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0)),
		})
		l.applyDynamicScalingLocked() // PILLAR 2: Synchronize scaling with new virtual liability
	}

	playerStats.Reputation = l.CalculateReputation(playerStats)
	l.leaderboard[playerWallet] = playerStats

	l.logAdminAuditLocked("CARD_BAILED", playerWallet, fmt.Sprintf("Card #%d bailed from Club %s for %.2f $VBV", data.CardID, club.Name, bailAmountBase))

	// PILLAR 3: Financial Proof.
	// Record bail settlement on-chain for the immutable audit trail.
	bailDetails := map[string]interface{}{
		"card_id": data.CardID,
		"bailer":  playerWallet,
		"club_id": data.ClubID,
		"amount":  bailAmountBase,
		"ts":      time.Now().Unix(),
	}

	// Dispatch on-chain log for financial verification
	go func(bd interface{}) {
		jsonPayload, _ := json.Marshal(bd)
		l.sendNoteTx(fmt.Sprintf("VBT_BAIL_LOG:%s", string(jsonPayload)))
	}(bailDetails)

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

		// PILLAR 3: Identity Hardening.
		// Ensure player records are initialized to prevent nil map panics or
		// orphaned wallets without reputation multipliers.
		l.ensurePlayerStatsMapsInitialized(state.VictimWallet)
		l.ensurePlayerStatsMapsInitialized(state.PerpWallet)

		// Automatic Return: No VBV exchange
		victimStats := l.leaderboard[state.VictimWallet]
		delete(victimStats.HeldHostageCards, cardID)

		cardKey := fmt.Sprintf("CARD-%d", cardID)
		victimStats.Inventory[cardKey]++ // Increment count, assuming it was decremented by 1

		victimStats.Reputation = l.CalculateReputation(victimStats)
		l.leaderboard[state.VictimWallet] = victimStats

		perpStats := l.leaderboard[state.PerpWallet]
		delete(perpStats.KidnappedCards, cardID)
		perpStats.Reputation = l.CalculateReputation(perpStats)
		l.leaderboard[state.PerpWallet] = perpStats

		// PILLAR 3: Financial Proof.
		// Record automated recovery on-chain for the immutable audit trail.
		recoveryDetails := map[string]interface{}{
			"card_id": cardID,
			"victim":  state.VictimWallet,
			"perp":    state.PerpWallet,
			"ts":      now.Unix(),
		}

		delete(l.activeKidnappings, cardID)

		// PILLAR 3: Multi-Slot Cleanup. Ensure the registry is purged on automated recovery.
		l.victimRegistry.Mu.Lock()
		if attackers, exists := l.victimRegistry.ActiveKidnaps[state.VictimWallet]; exists {
			delete(attackers, state.PerpWallet)
			if len(attackers) == 0 {
				delete(l.victimRegistry.ActiveKidnaps, state.VictimWallet)
			}
		}
		l.victimRegistry.Mu.Unlock()

		l.logAdminAuditLocked("INSURANCE_RECOVERY", state.VictimWallet, fmt.Sprintf("Card #%d automatically returned from %s", cardID, state.PerpWallet))

		// Dispatch on-chain log for immutable verification
		go func(rd interface{}) {
			jsonPayload, _ := json.Marshal(rd)
			l.sendNoteTx(fmt.Sprintf("VBT_INSURANCE_RETURN:%s", string(jsonPayload)))
		}(recoveryDetails)

		// Notify Players
		if vCID := l.getClientIDFromWalletLocked(state.VictimWallet); vCID != "" {
			l.sendToClientLocked(vCID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"✅ <b>INSURANCE RECOVERY:</b> Your kidnapped card #%d has been returned."}`, cardID))})
		}
		if pCID := l.getClientIDFromWalletLocked(state.PerpWallet); pCID != "" {
			l.sendToClientLocked(pCID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🚨 <b>HOSTAGE ESCAPED:</b> Card #%d has returned to its owner via Insurance Recovery."}`, cardID))})
		}

		// Justice careers earn XP monitoring insurance recovery events. Task 4301.
		for statsWallet := range l.leaderboard {
			ws := l.leaderboard[statsWallet]
			if ws.JobRole == "Warden" && statsWallet != state.VictimWallet && statsWallet != state.PerpWallet {
				scaledWPNP4 := ws.CareerXP.ComputeScaledXP(10, "Warden")
				l.TrackCareerXP(statsWallet, "Warden", scaledWPNP4)
			}
			if ws.JobRole == "Forensic Analyst" && statsWallet != state.VictimWallet && statsWallet != state.PerpWallet {
				scaledFANP3 := ws.CareerXP.ComputeScaledXP(30, "Forensic Analyst")
				l.TrackCareerXP(statsWallet, "Forensic Analyst", scaledFANP3)
			}
		}
	}

	// Broadcast update to refresh UI lists
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// handleCyberIntercept processes an Intel-Agent cyber-intercept action on active signals.
// Task 4502-B: Justice Hegemony — Intel-Agent combat hook handler.
func (l *Lobby) handleCyberIntercept(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		EventID string `json:"event_id"` // Optional: existing event to intercept (empty = generate new)
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	walletAddr := ""
	clientIDFromWallet := func(wallet string) string {
		for id, c := range l.clients {
			if cw, ok := l.wallets[id]; ok && cw == wallet {
				return id
			}
		}
		return ""
	}

	// Find the player's wallet from their client ID (passed via session or auth)
	clientID := r.Header.Get("X-Client-ID")
	if clientID != "" {
		walletAddr, _ = l.wallets[clientID]
	} else {
		http.Error(w, "Missing X-Client-ID header", http.StatusUnauthorized)
		return
	}

	stats, exists := l.leaderboard[walletAddr]
	if !exists || stats.CareerXP == nil {
		http.Error(w, "Player not found or no career data", http.StatusNotFound)
		return
	}

	isIntelAgent := stats.JobRole == "Int.Agent" || CareerHasRole(stats.CareerXP, "Int.Agent")
	if !isIntelAgent {
		http.Error(w, "Access denied: Intel-Agent role required", http.StatusForbidden)
		return
	}

	tier := stats.CareerXP.GetCareerTier("Int.Agent")
	signalStrength := 30 + tier*10 // Base signal strength scales with tier (max 90 at Boss tier)
	if signalStrength > 100 {
		signalStrength = 100
	}

	var interceptEvent *CyberInterceptEvent

	if req.EventID != "" && l.evidencePool != nil {
		// Re-intercept an existing event
		l.evidencePool.Mu.Lock()
		existing, ok := l.evidencePool.ActiveRecords[req.EventID]
		l.evidencePool.Mu.Unlock()
		if !ok || existing.Intercepted {
			http.Error(w, "Event not found or already intercepted", http.StatusBadRequest)
			return
		}
		interceptEvent = &CyberInterceptEvent{
			EventID:        req.EventID,
			SourceWallet:   existing.SourceWallet,
			TargetWallet:   walletAddr,
			SignalStrength: signalStrength,
			DecryptBonus:   stats.CareerXP.GetVBVGatingMultiplier(), // $VBV-gated decrypt bonus
			CreatedAt:      existing.CreatedAt,
			ExpiresAt:      time.Now().Add(30 * time.Minute), // Extend TTL on intercept attempt
			Intercepted:    true,
		}

		l.evidencePool.Mu.Lock()
		existing.Intercepted = true
		l.evidencePool.Mu.Unlock()
	} else {
		// Generate a new cyber-intercept event (costs resources)
		const interceptCost = 500 * 1000000 // 500 $VBV cost to scan for signals
		if l.playerBalances[walletAddr] < interceptCost {
			http.Error(w, "Insufficient funds: signal scanning costs 500 $VBV", http.StatusPaymentRequired)
			return
		}

		l.playerBalances[walletAddr] -= interceptCost
		l.faucetBalanceMicro += interceptCost // Cost goes to system faucet (deterministic sink)

		newEventID := fmt.Sprintf("CYBER-%s-%d", walletAddr, time.Now().UnixNano())
		interceptEvent = &CyberInterceptEvent{
			EventID:        newEventID,
			SourceWallet:   "", // Unknown suspect (signal detected but source obscured)
			TargetWallet:   walletAddr,
			SignalStrength: signalStrength + rand.Intn(15), // Add variance (+0-14 bonus)
			DecryptBonus:   stats.CareerXP.GetVBVGatingMultiplier(),
			CreatedAt:      time.Now(),
			ExpiresAt:      time.Now().Add(30 * time.Minute),
			Intercepted:    false,
		}

		l.evidencePool.Mu.Lock()
		if l.evidencePool.ActiveRecords == nil {
			l.evidencePool.ActiveRecords = make(map[string]*RaidEvidence)
		}
		l.evidencePool.ActiveRecords[newEventID] = &RaidEvidence{
			EvidenceID:   newEventID,
			SourceWallet: walletAddr, // Collector is the Intel-Agent who scanned
			CrimeType:    "CYBER_INTERCEPT",
			Confidence:   float64(signalStrength) / 100.0,
			CollectedAt:  time.Now(),
		}
		if l.evidencePool.CollectorMap == nil {
			l.evidencePool.CollectorMap = make(map[string][]string)
		}
		l.evidencePool.CollectorMap[walletAddr] = append(l.evidencePool.CollectorMap[walletAddr], newEventID)
		l.evidencePool.Mu.Unlock()

		l.logAdminAuditLocked("CYBER_INTERCEPT_EVENT_TRIGGERED", walletAddr, fmt.Sprintf("Signal strength: %d, tier: %d", signalStrength, tier))
	}

	// Apply Intel-Agent XP reward (base 50 + decrypt bonus)
	baseXP := uint64(50)
	if interceptEvent.DecryptBonus > 1.0 {
		scaledBaseXP := uint64(float64(baseXP) * interceptEvent.DecryptBonus)
		stats.CareerXP.TrackCareerXP("Int.Agent", scaledBaseXP)

		l.logAdminAuditLocked("CYBER_INTERCEPT_XP_REWARD", walletAddr, fmt.Sprintf("+%d XP (base: %d, decrypt bonus: %.2f)", scaledBaseXP, baseXP, interceptEvent.DecryptBonus))
	} else {
		stats.CareerXP.TrackCareerXP("Int.Agent", baseXP)
		l.logAdminAuditLocked("CYBER_INTERCEPT_XP_REWARD", walletAddr, fmt.Sprintf("+%d XP (base reward)", baseXP))
	}

	l.leaderboard[walletAddr] = stats

	// Ally synergy: Intel-Agent + Arc-Net Operante share visibility bonus
	for targetWallet, pStats := range l.leaderboard {
		if targetWallet == walletAddr || pStats.CareerXP == nil {
			continue
		}
		isArcNetOperative := pStats.JobRole == "Arc-Net Operative" || CareerHasRole(pStats.CareerXP, "Arc-Net Operative")
		if isArcNetOperative && l.isJusticeAligned(walletAddr) { // Ally synergy check
			rivalBonus, _, _ := EvaluateCrossCareerXP("Int.Agent", pStats.JobRole, baseXP, &stats, &pStats)
			if rivalBonus > int(baseXP) {
				pStats.CareerXP.TrackCareerXP("Arc-Net Operative", uint64(rivalBonus-int(baseXP)))
				l.leaderboard[targetWallet] = pStats

				// Shared visibility: Arc-Net gets half the decrypt bonus as XP
				if interceptEvent.DecryptBonus > 1.0 {
					sharingBonus := uint64(float64(baseXP) * interceptEvent.DecryptBonus * 0.5)
					pStats.CareerXP.TrackCareerXP("Arc-Net Operative", sharingBonus)
				}

				l.logAdminAuditLocked("INTEL_ARCNET_VISIBILITY_SHARING", walletAddr, fmt.Sprintf("+%d XP shared with Arc-Net Operante: %s", rivalBonus-int(baseXP), targetWallet))
			}
		}
	}

	// Broadcast cyber-intercept event to affected parties via WebSocket
	eventJSON := fmt.Sprintf(`{"event_id":"%s","signal_strength":%d,"decrypt_bonus":%.2f}`, interceptEvent.EventID, interceptEvent.SignalStrength, interceptEvent.DecryptBonus)
	go func() { l.broadcast <- []byte(fmt.Sprintf(`{"type":"cyber_intercept_event","payload":{%s}}`, eventJSON)) }()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "success",
		"event_id":      interceptEvent.EventID,
		"signal_strength": interceptEvent.SignalStrength,
		"decrypt_bonus":  interceptEvent.DecryptBonus,
	})

	l.logAdminAuditLocked("CYBER_INTERCEPT_COMPLETE", walletAddr, fmt.Sprintf("Event: %s, Signal: %d", interceptEvent.EventID, interceptEvent.SignalStrength))
}

/**
 * HandleAOSRaid allows a tactical Justice team to recover cards from a Hostage Host.
 * PILLAR 3: Team-based Rivalry Mechanics.
 */
func (l *Lobby) HandleAOSRaid(env *Envelope) {
	var data struct {
		TargetHostWallet string `json:"target_wallet"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()
	wallet, ok := l.wallets[env.FromID]
	if !ok { return }
	stats := l.leaderboard[wallet]

	if stats.JobRole != "AOS Leader" && !CareerHasRole(stats.CareerXP, "AOS Leader") {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Access Denied: AOS Raid protocol requires the 'AOS Leader' role."}`)})
		return
	}

	// Deployment Fee: 1,500 $VBV (Paid to System)
	const deploymentFee = 1500 * 1000000
	if l.playerBalances[wallet] < deploymentFee {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Raid Failed: Insufficient rewards for tactical deployment (1,500 $VBV required)."}`)})
		return
	}

	targetWallet := strings.ToLower(data.TargetHostWallet)
	targetStats, exists := l.leaderboard[targetWallet]
	if !exists || len(targetStats.KidnappedCards) == 0 {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Raid Failed: No active hostage signals detected at target location."}`)})
		return
	}

	// PILLAR 1: Team Synergy. Success scales with connected club members.
	teamSize := 1
	var participants []string
	participants = append(participants, wallet)
	for _, client := range l.clients {
		cw, ok := l.wallets[client.id]
		if ok && cw != wallet && l.leaderboard[cw].EmployerClubID == stats.EmployerClubID {
			teamSize++
			participants = append(participants, cw)
			if teamSize >= 4 { break }
		}
	}

	l.playerBalances[wallet] -= deploymentFee
	l.faucetBalanceMicro += deploymentFee

	// Success Chance: 40% base + 10% per teammate (Max 70%)
	successChance := 0.40 + (float64(teamSize-1) * 0.10)
	if rand.Float64() < successChance {
		// PILLAR 3: Raid Insurance Check.
		// If target has active insurance, block the success and consume the claim.
		if targetStats.RaidInsuranceClaimsRemaining > 0 && time.Now().Before(targetStats.RaidInsuranceExpiresAt) {
			targetStats.RaidInsuranceClaimsRemaining--
			l.leaderboard[targetWallet] = targetStats

			l.logAdminAuditLocked("RAID_INSURANCE_CONSUMED", wallet, fmt.Sprintf("Target: %s", targetWallet))
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"🎯 <b>RAID INTERCEPTED:</b> Tactical strike blocked by active Raid Insurance protocol."}`)})

			if tCID := l.getClientIDFromWalletLocked(targetWallet); tCID != "" {
				l.sendToClientLocked(tCID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"🛡️ <b>INSURANCE TRIGGERED:</b> A successful AOS Raid was blocked. One claim consumed."}`)})
			}
			return
		}

		// SUCCESS: Recover a random card
		var targetCardID int
		for cid := range targetStats.KidnappedCards { targetCardID = cid; break } // Select first entry

		victimWallet := targetStats.KidnappedCards[targetCardID]
		delete(targetStats.KidnappedCards, targetCardID)
		l.leaderboard[targetWallet] = targetStats

		vStats := l.leaderboard[victimWallet]
		delete(vStats.HeldHostageCards, targetCardID)
		vStats.Inventory[fmt.Sprintf("CARD-%d", targetCardID)]++
		l.leaderboard[victimWallet] = vStats
		delete(l.activeKidnappings, targetCardID)

		// Reward: 5,000 $VBV split among team
		rewardSplit := (5000 * 1000000) / uint64(len(participants))
		for _, pw := range participants {
			l.playerBalances[pw] += rewardSplit
			if cid := l.getClientIDFromWalletLocked(pw); cid != "" {
				l.sendToClientLocked(cid, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🎯 <b>RAID SUCCESS:</b> Card #%d recovered! Your share: %.2f $VBV."}`, targetCardID, float64(rewardSplit)/1000000.0))})
			}
			// Career XP: AOS Leader gains XP per successful raid (scaled by team size). Task 4301.
			aosBase := uint64(60 + teamSize*10)
			scaledAOSP2 := l.leaderboard[pw].CareerXP.ComputeScaledXP(aosBase, "AOS Leader")
			l.TrackCareerXP(pw, "AOS Leader", scaledAOSP2)
		}
		l.logAdminAuditLocked("AOS_RAID_SUCCESS", wallet, fmt.Sprintf("Host: %s, Card: %d, Team: %d", targetWallet, targetCardID, teamSize))
	} else {
		// FAILURE: Penalty
		stats.WantedLevel += 10
		stats.Reputation = l.CalculateReputation(stats)
		l.leaderboard[wallet] = stats
		l.logAdminAuditLocked("AOS_RAID_FAILED", wallet, fmt.Sprintf("Initiator: %s, Target: %s", wallet, targetWallet))
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"💀 <b>RAID FAILED:</b> Tactical team intercepted. Your signature has been flagged."}`)})
	}
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}
