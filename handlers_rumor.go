package main

import (
	"encoding/json"
	"fmt"
	"log"
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

	// Cost to spread a rumor: 500 $VBV (in micro-units)
	const rumorCost = 500 * 1000000
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

	l.ensurePlayerStatsMapsInitialized(spreaderWallet)
	spreaderStats := l.leaderboard[spreaderWallet]
	spreaderStats.RumorCount++
	spreaderStats.Reputation = l.CalculateReputation(spreaderStats)
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
