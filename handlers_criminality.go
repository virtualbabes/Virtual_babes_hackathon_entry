//go:build !js && !wasm

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
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
	kidnapBase := uint64(100) // Base XP for kidnapping career trigger (reconciled)
	if perpStats.KidnappedCards == nil {
		perpStats.KidnappedCards = make(map[int]string)
	}
	perpStats.KidnappedCards[targetCardID] = victimWallet

	// P2-A: Kidnapper ↔ Bounty Hunter. The award comes from the ONE resolver. The old code read the
	// DEFENDER's share (30% of the base) and compared it to the base — `30 > 100`, false for ever —
	// and `uint64(rxp - kidnapBase)` would have underflowed to 18,446,744,073,709,551,605 had the
	// guard ever passed.
	award := ResolveRivalXPAward("Kidnapper", victimStats.JobRole, kidnapBase, &perpStats, &victimStats)
	if award.Bonus > 0 {
		scaledRBonus := l.leaderboard[perpWallet].CareerXP.ComputeScaledXP(award.Bonus, "Kidnapper")
		l.trackCareerXPLocked(perpWallet, "Kidnapper", scaledRBonus)
	}

	// P2-A: Rival Pair — Warden (Justice D3) gains XP monitoring kidnapping events. Task 4301.
	wardenBase := uint64(25)
	scaledWPNP := l.leaderboard[perpWallet].CareerXP.ComputeScaledXP(wardenBase, "Warden")
	l.trackCareerXPLocked(perpWallet, "Warden", scaledWPNP)

	// Underworld #6 Racketeer — earns XP extorting protection money per kidnapping. Task 4301.
	rackBase := uint64(40)
	scaledRacNP := l.leaderboard[perpWallet].CareerXP.ComputeScaledXP(rackBase, "Racketeer")
	l.trackCareerXPLocked(perpWallet, "Racketeer", scaledRacNP)

	// P2-A: Rival Pair — Smuggler ↔ Sector Peacekeeper (antagonistic). Task 4301.
	if perpStats.CareerXP != nil {
		award := ResolveRivalXPAward("Smuggler", victimStats.JobRole, 30, &perpStats, &victimStats)
		if award.Bonus > 0 {
			scaledSMP := l.leaderboard[perpWallet].CareerXP.ComputeScaledXP(award.Bonus, "Smuggler")
			l.trackCareerXPLocked(perpWallet, "Smuggler", scaledSMP)
		}
	}

	// Justice D5 Forensic Analyst — forensic trace reward at detection time. Task 4301.
	if l.leaderboard != nil {
		for statsWallet := range l.leaderboard {
			ws := l.leaderboard[statsWallet]
			if ws.JobRole == "Forensic Analyst" {
				scaledFANP := ws.CareerXP.ComputeScaledXP(60, "Forensic Analyst")
				l.trackCareerXPLocked(statsWallet, "Forensic Analyst", scaledFANP)
			}
		}
	}

	// Justice D3 Warden — monitors all hostage activity across the sector. Task 4301.
	if l.leaderboard != nil {
		for statsWallet := range l.leaderboard {
			ws := l.leaderboard[statsWallet]
			if ws.JobRole == "Warden" && statsWallet != perpWallet { // Avoid double-counting with Warden line above
				scaledWP2NP := ws.CareerXP.ComputeScaledXP(25, "Warden")
				l.trackCareerXPLocked(statsWallet, "Warden", scaledWP2NP)
			}
		}
	}

	// Justice D6 Sector Peacekeeper — maintains sector order during criminal events. Task 4301.
	if l.leaderboard != nil {
		for statsWallet := range l.leaderboard {
			ws := l.leaderboard[statsWallet]
			if ws.JobRole == "Sector Peacekeeper" && statsWallet != perpWallet { // Avoid double-counting if perp is SPK
				scaledSPKP := ws.CareerXP.ComputeScaledXP(20, "Sector Peacekeeper")
				l.trackCareerXPLocked(statsWallet, "Sector Peacekeeper", scaledSPKP)
			}
		}
	}

	// Track this as an active kidnapping for Hostage Host progression tracking. Task 4301.
	if perpStats.ActiveHostageCount == 0 {
		// First hostage — Hostage Host career milestone
		scaledHHNP := l.leaderboard[perpWallet].CareerXP.ComputeScaledXP(50, "Hostage Host")
		l.trackCareerXPLocked(perpWallet, "Hostage Host", scaledHHNP)
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
		l.sendNoteTx(fmt.Sprintf(NotePrefixRansomLog+"%s", string(jsonPayload)))
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
			l.trackCareerXPLocked(statsWallet, "Warden", scaledWPNP2)
		}
	}

	// Justice D5 Forensic Analyst — traces financial patterns during ransom events. Task 4301.
	for statsWallet := range l.leaderboard {
		ws := l.leaderboard[statsWallet]
		if ws.JobRole == "Forensic Analyst" && statsWallet != victimWallet && statsWallet != data.PerpWallet {
			scaledFANP2 := ws.CareerXP.ComputeScaledXP(45, "Forensic Analyst")
			l.trackCareerXPLocked(statsWallet, "Forensic Analyst", scaledFANP2)
		}
	}

	// P2-E: Bounty Hunter ↔ Kidnapper — non-combat XP trigger for ransom payment resolution.
	perpWallet := data.PerpWallet
	if perpStats.CareerXP != nil && (perpStats.JobRole == "Kidnapper" || CareerHasRole(perpStats.CareerXP, "Kidnapper")) {
		for statsWallet := range l.leaderboard {
			ws := l.leaderboard[statsWallet]
			if ws.JobRole == "BountyHunter" && statsWallet != perpWallet && statsWallet != victimWallet {
				award := ResolveRivalXPAward("BountyHunter", perpStats.JobRole, data.RansomAmount/10, &ws, &perpStats)
				if award.Bonus > 0 {
					scaledRBonus := ws.CareerXP.ComputeScaledXP(award.Bonus, "BountyHunter")
					l.trackCareerXPLocked(statsWallet, "BountyHunter", scaledRBonus)
					l.logAdminAuditLocked("CAREER_BOUNTY_HUNTER_KIDNAPPER_RANSOM", statsWallet, fmt.Sprintf("+%d XP (Kidnapper ransom resolution, %s)", award.Bonus, strings.Join(award.Layers, "+")))
				}
			}
		}
	}

	// P2-E: Sector Peacekeeper ↔ Smuggler — non-combat XP trigger for hostage release monitoring.
	if perpStats.CareerXP != nil && (perpStats.JobRole == "Smuggler" || CareerHasRole(perpStats.CareerXP, "Smuggler")) {
		for statsWallet := range l.leaderboard {
			ws := l.leaderboard[statsWallet]
			if ws.JobRole == "SectorPeacekeeper" && statsWallet != perpWallet && statsWallet != victimWallet {
				award := ResolveRivalXPAward("SectorPeacekeeper", perpStats.JobRole, 30, &ws, &perpStats)
				if award.Bonus > 0 {
					scaledSPBonus := ws.CareerXP.ComputeScaledXP(award.Bonus, "SectorPeacekeeper")
					l.trackCareerXPLocked(statsWallet, "SectorPeacekeeper", scaledSPBonus)
					l.logAdminAuditLocked("CAREER_SECTOR_PEACEKEEPER_SMUGGLER_RANSOM", statsWallet, fmt.Sprintf("+%d XP (Smuggler ransom resolution, %s)", award.Bonus, strings.Join(award.Layers, "+")))
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
			l.trackCareerXPLocked(statsWallet, "Warden", scaledWPNP3)
		}
	}

	// Justice D6 Sector Peacekeeper — maintains order during criminal resolutions. Task 4301.
	for statsWallet := range l.leaderboard {
		ws := l.leaderboard[statsWallet]
		if ws.JobRole == "Sector Peacekeeper" && statsWallet != perpWallet && statsWallet != victimWallet {
			scaledSPKP2 := ws.CareerXP.ComputeScaledXP(15, "Sector Peacekeeper")
			l.trackCareerXPLocked(statsWallet, "Sector Peacekeeper", scaledSPKP2)
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

	// ── READ-ONLY VALIDATION (short RLock; released BEFORE the oracle call) ──
	//
	// This handler previously held the WRITE lock for its entire body AND took a
	// READ lock inside it twice — an unconditional self-deadlock on a
	// sync.RWMutex that froze the ENTIRE server (Problems.md §15). The write lock
	// is now taken only around the state change, and never across
	// VerifyBuyInTransaction, which itself takes l.mutex.RLock().
	l.mutex.RLock()
	playerWallet, ok := l.wallets[env.FromID]
	if !ok {
		l.mutex.RUnlock()
		log.Printf("[CRIMINALITY] Bail failed: Sender %s not connected.\n", env.FromID)
		return
	}
	club, exists := l.clubs[data.ClubID]
	cardInJail := false
	ownershipOK := false
	if exists {
		_, cardInJail = club.Jail[data.CardID]
		if ps, hasStats := l.leaderboard[playerWallet]; hasStats {
			clubIDForCard, hasJailed := ps.JailedCards[data.CardID]
			ownershipOK = hasJailed && clubIDForCard == data.ClubID
		}
	}
	l.mutex.RUnlock()

	if !exists {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Bail Failed: Club not found."}`)})
		return
	}
	if !cardInJail {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Bail Failed: Card not found in this club's jail."}`)})
		return
	}
	if !ownershipOK {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Bail Failed: You do not own this jailed card."}`)})
		return
	}

	// ── UNIFORM TXID GUARD (txid_memo.go) ────────────────────────────────────
	claimID, claimReason := l.claimTxID(data.TxID)
	if claimID == "" {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"❌ Bail Failed: %s."}`, claimReason))})
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
		l.releaseTxID(claimID)
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Bail Failed: Voi network configuration missing."}`)})
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
	verified, _, err := l.oracleService.VerifyBuyInTransaction(l, verifyNet, data.TxID, bailAmountMicro, assetID, playerWallet, vaultAddr, NotePrefixBailPayment)
	if err != nil || !verified {
		l.releaseTxID(claimID)
		log.Printf("[CRIMINALITY] Bail payment verification failed for %s (Card %d): %v\n", playerWallet, data.CardID, err)
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Bail Failed: Payment verification failed or insufficient amount."}`)})
		return
	}

	// ── APPLY (ONE write-lock critical section) ──────────────────────────────
	// Re-validated because the state may have changed during the oracle
	// round-trip: another request may have bailed the card, emptied the jail, or
	// (impossibly, but cheaply checked) deleted the club.
	l.mutex.Lock()

	club, exists = l.clubs[data.ClubID]
	if !exists {
		l.mutex.Unlock()
		l.releaseTxID(claimID)
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Bail Failed: Club not found."}`)})
		return
	}
	jailedCard, stillJailed := club.Jail[data.CardID]
	if !stillJailed {
		l.mutex.Unlock()
		l.releaseTxID(claimID)
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Bail Failed: That card has already been released."}`)})
		return
	}
	playerStats, hasStats := l.leaderboard[playerWallet]
	if !hasStats {
		l.mutex.Unlock()
		l.releaseTxID(claimID)
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Bail Failed: Player record not found."}`)})
		return
	}
	if clubIDForCard, stillOwned := playerStats.JailedCards[data.CardID]; !stillOwned || clubIDForCard != data.ClubID {
		l.mutex.Unlock()
		l.releaseTxID(claimID)
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Bail Failed: You do not own this jailed card."}`)})
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
		l.sendNoteTx(fmt.Sprintf(NotePrefixBailLog+"%s", string(jsonPayload)))
	}(bailDetails)

	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"✅ <b>BAIL PAID:</b> Your card '%s' has been released from %s's jail!"}`, escapeHTML(jailedCard.Name), escapeHTML(club.Name)))})

	// Notify club owner/members (optional, but good for transparency)
	clubOwnerClientID := l.getClientIDFromWalletLocked(club.OwnerWallet)
	if clubOwnerClientID != "" {
		l.sendToClientLocked(clubOwnerClientID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>BAIL RECEIVED:</b> Club %s received %.2f $VBV for card #%d."}`, escapeHTML(club.Name), bailAmountBase, data.CardID))})
	}
	l.mutex.Unlock()

	// Record the consumed transaction ONLY now that the bail has been applied.
	l.commitTxID(claimID, time.Now())

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
			l.sendNoteTx(fmt.Sprintf(NotePrefixInsuranceReturn+"%s", string(jsonPayload)))
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
				l.trackCareerXPLocked(statsWallet, "Warden", scaledWPNP4)
			}
			if ws.JobRole == "Forensic Analyst" && statsWallet != state.VictimWallet && statsWallet != state.PerpWallet {
				scaledFANP3 := ws.CareerXP.ComputeScaledXP(30, "Forensic Analyst")
				l.trackCareerXPLocked(statsWallet, "Forensic Analyst", scaledFANP3)
			}
		}
	}

	// Broadcast update to refresh UI lists
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// handleCyberIntercept processes an Intel-Agent cyber-intercept action on active signals.
// Task 4502-B: Justice Hegemony — Intel-Agent combat hook handler.
//
// WALLET RESOLUTION (fixed 2026-09-15). This door used to require an `X-Client-ID` header and
// answered 401 without one — but `l.wallets` is keyed by a WebSocket connection id and NO module in
// Public/** sets that header, so the endpoint could only ever refuse: a built capability wired to a
// guaranteed failure. It now resolves the caller the way every other door does (the ONE resolver
// getWalletFromRequest: X-Wallet-Address header, then ?wallet=), keeping X-Client-ID as a SECONDARY
// source for a socket-identified call.
func (l *Lobby) handleCyberIntercept(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONStatus(w, http.StatusMethodNotAllowed, map[string]interface{}{"success": false, "error": "POST required"})
		return
	}

	var req struct {
		EventID string `json:"event_id"` // Optional: existing event to intercept (empty = generate new)
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}

	walletAddr := l.getWalletFromRequest(r)

	l.mutex.Lock()
	defer l.mutex.Unlock()

	if walletAddr == "" {
		if clientID := r.Header.Get("X-Client-ID"); clientID != "" {
			walletAddr, _ = l.wallets[clientID]
		}
	}
	if walletAddr == "" {
		writeJSONStatus(w, http.StatusUnauthorized, map[string]interface{}{
			"success": false,
			"error":   "wallet required: send X-Wallet-Address or ?wallet=",
		})
		return
	}

	// Resolve the STORED spelling once, so every read and write below addresses the same record.
	walletAddr = l.leaderboardKeyLocked(walletAddr)

	stats, exists := l.leaderboard[walletAddr]
	if !exists || stats.CareerXP == nil {
		writeJSONStatus(w, http.StatusNotFound, map[string]interface{}{
			"success": false, "error": "player not found or no career data", "wallet": walletAddr,
		})
		return
	}

	// THE ROLE GATE. `CareerHasRole` cannot be the gate: nothing in the repository ever WRITES
	// PromotedRoles (see Problems.md §24), so it is permanently false. The signals that exist are
	// the career XP map and JobRole, and both are compared BY ROLE (RoleKey), so "Int.Agent",
	// "IntelAgent" and "Intel-Agent" are one career instead of three.
	isIntelAgent := HasCareer(stats.CareerXP, "Intel-Agent") ||
		RoleKey(stats.JobRole) == RoleKey("Intel-Agent")
	if !isIntelAgent {
		writeJSONStatus(w, http.StatusForbidden, map[string]interface{}{
			"success": false,
			"error":   "Intel-Agent career required: earn career XP in the Intel-Agent path first",
			"wallet":  walletAddr,
		})
		return
	}

	tier := stats.CareerXP.GetCareerTier("Intel-Agent")
	costMicro := uint64(0)
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
		// The balance map is keyed by its OWN spelling: resolve it instead of assuming the
		// leaderboard's key reaches it (the same rule — one map, one key).
		balKey := l.balanceKeyLocked(walletAddr)
		if l.playerBalances[balKey] < interceptCost {
			writeJSONStatus(w, http.StatusPaymentRequired, map[string]interface{}{
				"success":       false,
				"error":         "insufficient funds: signal scanning costs 500 $VBV",
				"wallet":        walletAddr,
				"cost_micro":    interceptCost,
				"balance_micro": l.playerBalances[balKey],
			})
			return
		}

		l.playerBalances[balKey] -= interceptCost
		l.faucetBalanceMicro += interceptCost // the deterministic sink (Faucet)
		l.logAdminAuditLocked("CYBER_INTERCEPT_FEE_ROUTED", walletAddr, fmt.Sprintf("%d micro-$VBV routed to the faucet sink", interceptCost))
		costMicro = interceptCost

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
		l.evidencePool.Mu.Unlock()

		l.logAdminAuditLocked("CYBER_INTERCEPT_EVENT_TRIGGERED", walletAddr, fmt.Sprintf("Signal strength: %d, tier: %d", signalStrength, tier))
	}

	// Apply the Intel-Agent XP reward (base 50, $VBV-gated).
	//
	// INTEGER ONLY. This was `uint64(float64(baseXP) * decryptBonus)` — a float multiply inside the
	// career-XP path, which the Architecture Ledger prohibits ("Floating point prohibited for: …
	// Career / combat XP"). GetVBVGatingPermille IS the gate the float form applied (its values are
	// exact integers: 1×/2×/4×/8×/16×/32×), expressed in parts-per-thousand so the product is exact.
	baseXP := uint64(50)
	gating := stats.CareerXP.GetVBVGatingPermille() // 1000 = ungated
	xpAwarded := baseXP * gating / 1000
	stats.CareerXP.TrackCareerXP("Intel-Agent", xpAwarded)
	l.leaderboard[walletAddr] = stats
	l.logAdminAuditLocked("CYBER_INTERCEPT_XP_REWARD", walletAddr,
		fmt.Sprintf("+%d XP (base: %d, $VBV gate: %d permille)", xpAwarded, baseXP, gating))

	// Ally synergy: an Arc-Net Operative shares the intercept's visibility.
	//
	// THREE defects were repaired here (2026-09-15):
	//   1. `l.isJusticeAligned` is the SELF-LOCKING form and this handler holds the WRITE lock, so
	//      reaching it was an unconditional deadlock that never released the lock — it froze every
	//      request and every WebSocket in the process. It fired whenever ANY Arc-Net Operative was on
	//      the leaderboard, which is why it had never been reproduced. The lock-held form is used.
	//   2. The roles were matched as exact literals ("Arc-Net Operative", "Int.Agent"); the role
	//      vocabulary (RoleKey, rival_career_engine.go) makes one career of its spellings.
	//   3. `_, rivalBonus, _, _ := EvaluateCrossCareerXP(…)` read the SECOND return value, which is
	//      the DEFENDER's award (base × 0.30), and compared it with `baseXP` — so `rivalBonus > baseXP`
	//      could never hold and this branch, its XP and its audit line could never occur. It now uses
	//      the ATTACKER's award and the isRival flag, which is what the branch describes.
	for targetWallet, pStats := range l.leaderboard {
		if targetWallet == walletAddr || pStats.CareerXP == nil {
			continue
		}
		isArcNetOperative := RoleKey(pStats.JobRole) == RoleKey("Arc-Net Operative")
		if isArcNetOperative && l.isJusticeAlignedLocked(walletAddr) { // ally check (lock already held)
			attackerXP, _, _, isRival := EvaluateCrossCareerXP("Intel-Agent", pStats.JobRole, baseXP, &stats, &pStats)
			if isRival && attackerXP > baseXP {
				shared := attackerXP - baseXP
				// Shared visibility: the ally earns half of the gated value as XP — integer-only.
				if gating > 1000 {
					shared += baseXP * (gating - 1000) / 1000 / 2
				}
				pStats.CareerXP.TrackCareerXP("Arc-Net Operative", shared)
				l.leaderboard[targetWallet] = pStats
				l.logAdminAuditLocked("INTEL_ARCNET_VISIBILITY_SHARING", walletAddr,
					fmt.Sprintf("+%d XP shared with the Arc-Net Operative %s", shared, targetWallet))
			}
		}
	}

	// Broadcast cyber-intercept event to affected parties via WebSocket
	eventJSON := fmt.Sprintf(`{"event_id":"%s","signal_strength":%d,"decrypt_bonus":%.2f}`, interceptEvent.EventID, interceptEvent.SignalStrength, interceptEvent.DecryptBonus)
	go func() { l.broadcast <- []byte(fmt.Sprintf(`{"type":"cyber_intercept_event","payload":{%s}}`, eventJSON)) }()

	// ONE envelope, and it states what actually happened: what was paid, what was earned, and the
	// gate in the INTEGER form the XP maths uses (`decrypt_bonus` is kept for display only — the
	// Ledger permits a float for presentation and forbids one in the XP path).
	writeJSONStatus(w, http.StatusOK, map[string]interface{}{
		"success":                true,
		"status":                 "success",
		"wallet":                 walletAddr,
		"career_role":            "Intel-Agent",
		"tier":                   tier,
		"event_id":               interceptEvent.EventID,
		"signal_strength":        interceptEvent.SignalStrength,
		"cost_micro":             costMicro,
		"xp_awarded":             xpAwarded,
		"decrypt_bonus_permille": gating,
		"decrypt_bonus":          interceptEvent.DecryptBonus,
		"expires_at_unix":        interceptEvent.ExpiresAt.Unix(),
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
			l.trackCareerXPLocked(pw, "AOS Leader", scaledAOSP2)
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
