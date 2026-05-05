package main

import (
	"encoding/json"
	"fmt"
	"log"
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
	if l.rewards[spreaderWallet] < rumorCost {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Rumor Failed: Insufficient $VBV to spread rumors."}`)})
		return
	}

	// Validate target
	if _, exists := l.leaderboard[data.TargetWallet]; !exists {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Rumor Failed: Target not found in Arena."}`)})
		return
	}

	// Deduct cost
	l.rewards[spreaderWallet] -= rumorCost

	// Update spreader's RumorCount
	spreaderStats := l.leaderboard[spreaderWallet]
	spreaderStats.RumorCount++
	l.leaderboard[spreaderWallet] = spreaderStats

	// Create and add rumor
	rumorID := fmt.Sprintf("RUMOR-%d", time.Now().UnixNano())
	rumor := &Rumor{ // Define rumor here so rumorJSON can use it
		ID:            rumorID,
		SpreaderWallet: spreaderWallet,
		TargetWallet:  data.TargetWallet,
		Type:          data.Type,
		Strength:      data.Strength,
		ExpiresAt:     time.Now().Add(time.Duration(data.Duration) * time.Minute),
	}
	l.rumors[rumorID] = rumor // Add to lobby's rumors map

	l.logAdminAudit("RUMOR_SPREAD", spreaderWallet, fmt.Sprintf("Target: %s, Type: %s, Strength: %.2f, Duration: %dmin", data.TargetWallet, data.Type, data.Strength, data.Duration))
	l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"📢 Rumor about %s spread successfully!"}`, data.TargetWallet))})

	// Notify target (if connected)
	targetClientID := l.getClientIDFromWallet(data.TargetWallet)
	if targetClientID != "" {
		l.sendToClient(targetClientID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"👀 A %s rumor is circulating about you!"}`, data.Type))})
	}

	// Broadcast rumor update to all clients for lobby visibility, including the full rumor object
	rumorJSON, _ := json.Marshal(rumor) // Marshal the created rumor object
	envelope := Envelope{
		Type:    "rumor_update",
		Payload: json.RawMessage(fmt.Sprintf(`{"rumor":%s}`, string(rumorJSON))),
	}
	envelopeBytes, _ := json.Marshal(envelope)
	l.broadcast <- envelopeBytes

	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}