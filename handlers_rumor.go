//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"
)

// SpreadRumorData defines the payload for spreading a rumor.
type SpreadRumorData struct {
	TargetWallet string  `json:"target_wallet"`
	Type         string  `json:"type"`     // "positive", "negative"
	Strength     float64 `json:"strength"` // Multiplier (e.g., 1.1 for +10%, 0.9 for -10%)
	Duration     int     `json:"duration_minutes"`
}

// handleSpreadRumor processes a request from a player to spread a rumor about another entity.
func (l *Lobby) handleSpreadRumor(env *Envelope) {
	var data SpreadRumorData
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		log.Printf("[RUMOR] Invalid spread_rumor payload from %s: %v\n", env.FromID, err)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	spreaderWallet, ok := l.wallets[env.FromID]
	if !ok {
		log.Printf("[RUMOR] Rumor failed: Spreader %s not connected.\n", env.FromID)
		return
	}

	l.ensurePlayerStatsMapsInitialized(spreaderWallet)
	spreaderStats := l.leaderboard[spreaderWallet]

	// PILLAR 3: Career Path influence. 'Gossips' receive tiered discount on manipulation fees.
	// Tier 3+ (Journeyman+) = 20% discount, lower tiers = no discount.
	// Base cost: 500 $VBV.
	const standardRumorCost = 500 * 1000000
	rumorCost := standardRumorCost

	if spreaderStats.JobRole == "Gossip" && spreaderStats.CareerXP != nil {
		discount := spreaderStats.CareerXP.GetRumorFeeDiscount()
		rumorCost = int(float64(standardRumorCost) * discount)

		// PILLAR 13: Task 4201-1C — Add visible buff tag when Gossip ≥ Tier 3 (discount active)
		if discount < 1.0 {
			if spreaderStats.ActiveBuffs == nil {
				spreaderStats.ActiveBuffs = make(map[string]string)
			}
			spreaderStats.ActiveBuffs["Gossip_Discount"] = "active"
		}
	}

	if l.playerBalances[spreaderWallet] < rumorCost {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Rumor Failed: Insufficient $VBV to spread rumors."}`)})
		return
	}

	// Hardening: Sanity check for rumor metrics
	if data.Strength < 0.1 || data.Strength > 2.0 {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Rumor Failed: Strength must be between 0.1 and 2.0."}`)})
		return
	}
	if data.Duration <= 0 || data.Duration > 1440 { // Max 24 hours
		data.Duration = 60 // Default to 1 hour if invalid
	}

	targetWallet := strings.ToLower(data.TargetWallet)

	// Validate target existence
	if _, exists := l.leaderboard[targetWallet]; !exists {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Rumor Failed: Target not found in Arena."}`)})
		return
	}

	// INDUSTRIAL LOOP: Fee Redistribution & Spreader Standing
	l.playerBalances[spreaderWallet] -= rumorCost

	feeBase := float64(rumorCost) / 1000000.0
	var governors []*Club
	for _, club := range l.clubs {
		if len(club.Territories) >= 2 {
			governors = append(governors, club)
		}
	}

	if len(governors) > 0 {
		govTaxBase := feeBase * 0.20 // 20% Regional Governor Tax
		faucetShareBase := feeBase - govTaxBase
		l.faucetBalance += faucetShareBase
		taxPerGov := govTaxBase / float64(len(governors))
		for _, gov := range governors {
			gov.Treasury += taxPerGov
			gov.LastActivity = time.Now()
		}
		l.logAdminAuditLocked("RUMOR_FEE_REDISTRIBUTION", spreaderWallet, fmt.Sprintf("Governors: %d, Tax: %.2f", len(governors), govTaxBase))
	} else {
		l.faucetBalance += feeBase
	}

	l.applyDynamicScalingLocked()

	spreaderStats.Reputation = l.CalculateReputation(spreaderStats)

	// PILLAR 3: Underworld Contract Completion (CONTRACT-006).
	// Objective: Successfully spread a Negative Rumor about a Regional Governor.
	if spreaderStats.ActiveUnderworldContractID == "CONTRACT-006" && data.Type == "negative" {
		// Check if the target is a Regional Governor (Owner of a club with 2+ districts)
		isGov := false
		for _, club := range l.clubs {
			if strings.EqualFold(club.OwnerWallet, targetWallet) && l.clubService.IsClubRegionalLocked(l, club) {
				isGov = true
				break
			}
		}

		if isGov {
			const rewardMicro = 1500 * 1000000
			l.playerBalances[spreaderWallet] += rewardMicro
			spreaderStats.ActiveUnderworldContractID = ""
			l.logAdminAuditLocked("CONTRACT_COMPLETED", spreaderWallet, "ID: CONTRACT-006, Payout: 1500.00")
			l.sendToClientLocked(env.FromID, Envelope{
				Type:    "admin_notification",
				Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> Regional Governor defamed. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0)),
			})
			l.applyDynamicScalingLocked()
		}
	}

	// PILLAR 3: Justice Mission Completion (MISSION-006).
	// Objective: Successfully spread a Positive Rumor about a Justice-aligned player.
	if spreaderStats.ActiveJusticeMissionID == "MISSION-006" && data.Type == "positive" {
		// Check if the target is a Justice-aligned player
		targetStats, exists := l.leaderboard[targetWallet]
		if exists && l.playerService.GetHegemonyPath(targetStats.JobRole) == "JUSTICE" {
			const rewardMicro = 1500 * 1000000
			l.playerBalances[spreaderWallet] += rewardMicro
			spreaderStats.ActiveJusticeMissionID = ""
			l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", spreaderWallet, "ID: MISSION-006, Payout: 1500.00")
			l.sendToClientLocked(env.FromID, Envelope{
				Type:    "admin_notification",
				Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> Justice reputation amplified. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0)),
			})
			l.applyDynamicScalingLocked()
		}
	}

	// PILLAR 13: Underworld Career #1 — Gossip (Rumor Manipulator).
	// XP tracking: rumors amplify reputation manipulation; +50 XP per successful rumor.
	// Phase 4: Scale XP by loyalty/fame multipliers from career state (Task 4301).
	baseGossipXP := uint64(50)
	loyaltyBonus := 0.0
	if spreaderStats.CareerXP != nil {
		lessonsCompleted := int32(0)
		for _, tier := range spreaderStats.CareerXP.Tiers {
			lessonsCompleted += tier.LessonsCompleted
		}
		// Loyalty: +5% per 10 lessons, cap at +50% (10 tiers)
		if lessonsCompleted > 0 {
			loyaltyBonus = math.Min(0.50, float64(lessonsCompleted)/200.0)
		}
		// Fame: +10% per standing tier, cap at +60% (6 tiers)
		fameBonus := 0.0
		for _, tier := range spreaderStats.CareerXP.Tiers {
			if tier.LessonsCompleted > 0 && tier.StandingTier > 0 {
				tierFame := float64(tier.StandingTier) * 0.10
				if tierFame > fameBonus {
					fameBonus = tierFame
				}
			}
		}
		fameBonus = math.Min(0.60, fameBonus)

		// $VBV-gated multiplier: scale XP by player's sustained liquidity tier (PILLAR 13).
		vbvMultiplier := spreaderStats.CareerXP.GetVBVGatingMultiplier()
		scaledXP := uint64(float64(spreaderStats.CareerXP.computeScaledXP(baseGossipXP, loyaltyBonus, fameBonus)) * vbvMultiplier)

		l.logAdminAuditLocked("CAREER_GOSSIP_XP", spreaderWallet, fmt.Sprintf("+%d XP (base: %d, loyalty: %.2f, fame: %.2f, $VBV-gate: ×%.0f)", scaledXP, baseGossipXP, loyaltyBonus, fameBonus, vbvMultiplier))
		l.TrackCareerXP(spreaderWallet, "Gossip", scaledXP)

		if vbvMultiplier > 1.0 {
			l.logAdminAuditLocked("CAREER_GOSSIP_VBV_GATE", spreaderWallet, fmt.Sprintf("$VBV-gate active: ×%.0f (AvgSustainedMicro: %d μVBV)", vbvMultiplier, spreaderStats.CareerXP.AvgSustainedMicro))
		}
	} else {
		l.logAdminAuditLocked("CAREER_GOSSIP_XP_NO_CAREER", spreaderWallet, fmt.Sprintf("+%d XP (no career state)", baseGossipXP))
		l.TrackCareerXP(spreaderWallet, "Gossip", baseGossipXP)
	}

	l.leaderboard[spreaderWallet] = spreaderStats

	// Refresh target Standing to reflect market volatility
	l.ensurePlayerStatsMapsInitialized(targetWallet)
	targetStats := l.leaderboard[targetWallet]
	targetStats.Reputation = l.CalculateReputation(targetStats)
	l.leaderboard[targetWallet] = targetStats

	// Create and add rumor
	rumorID := fmt.Sprintf("RUMOR-%d", time.Now().UnixNano())
	rumor := &Rumor{ // Define rumor here so rumorJSON can use it
		ID:             rumorID,
		SpreaderWallet: spreaderWallet,
		TargetWallet:   targetWallet,
		Type:           data.Type,
		Strength:       data.Strength,
		ExpiresAt:      time.Now().Add(time.Duration(data.Duration) * time.Minute),
	}
	l.rumors[rumorID] = rumor // Add to lobby's rumors map

	l.logAdminAuditLocked("RUMOR_SPREAD", spreaderWallet, fmt.Sprintf("Target: %s, Type: %s, Strength: %.2f, Duration: %dmin", targetWallet, data.Type, data.Strength, data.Duration))
	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"📢 Rumor about %s spread successfully!"}`, targetWallet))})

	// Notify target (if connected)
	targetClientID := l.getClientIDFromWalletLocked(targetWallet)
	if targetClientID != "" {
		l.sendToClientLocked(targetClientID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"👀 A %s rumor is circulating about you!"}`, data.Type))})
	}

	// Broadcast rumor update to all clients for lobby visibility, including the full rumor object
	rumorJSON, _ := json.Marshal(rumor) // Marshal the created rumor object
	envelope := Envelope{
		Type:    "rumor_update",
		Payload: json.RawMessage(fmt.Sprintf(`{"rumor":%s}`, string(rumorJSON))),
	}
	envelopeBytes, _ := json.Marshal(envelope)
	l.broadcast <- envelopeBytes

	msg := l.getLobbyUpdateMsgLocked()
	go func() { l.broadcast <- msg }()
}
