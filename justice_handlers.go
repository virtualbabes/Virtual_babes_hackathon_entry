//go:build !js && !wasm

// justice_handlers.go - Justice Hegemony Dashboard HTTP Handlers (Pillar 7)
// Implements the presentation layer for the Justice Dashboard API endpoints.
// Business logic remains in justice_service.go; this file owns only HTTP I/O.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ============================================================================
// REQUEST / RESPONSE TYPES
// ============================================================================

// JusticeDashboardAPI is the JSON response for GET /api/justice/dashboard.
type JusticeDashboardAPI struct {
	PowerBonus       int              `json:"powerBonus"`        // percentage bonus (e.g., +10%)
	Tier             int              `json:"tier"`              // BountyRank level
	Bounties         []BountyItemJSON `json:"bounties"`          // active bounty targets
	TruthSerumActive bool             `json:"truthSerumActive"`  // boolean flag
	ShieldRemaining  int              `json:"shieldRemaining"`   // remaining shield capacity
}

// BountyItemJSON represents a single bounty target in the dashboard response.
type BountyItemJSON struct {
	TargetWallet string `json:"targetWallet"` // wallet address (canonical identifier)
	TargetName   string `json:"targetName"`   // resolved player name or alias
	WantedLevel  int    `json:"wantedLevel"`
	Reward       uint64 `json:"reward"` // in micro-units
}

// UseTruthSerumRequest is the JSON request body for POST /api/justice/use-truth-serum.
type UseTruthSerumRequest struct {
	TargetWallet string `json:"targetWallet"`
}

// CaptureBountyRequest is the JSON request body for POST /api/justice/capture-bounty.
type CaptureBountyRequest struct {
	TargetWallet string `json:"targetWallet"`
}

// APIResponseJSON is a generic success/error response wrapper.
type APIResponseJSON struct {
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
	Reward   uint64 `json:"reward,omitempty"`    // for bounty capture responses
	Duration int    `json:"duration,omitempty"`  // for truth serum responses
}

// ============================================================================
// HANDLER STRUCTS
// ============================================================================

// JusticeHandlers holds the http.Handler implementations for the Justice Dashboard.
type JusticeHandlers struct {
	lobby *Lobby // access to shared state (justiceService, playerService, broadcast)
}

// ============================================================================
// HELPER: Wallet-to-Player Resolution
// ============================================================================

func (h *JusticeHandlers) resolveWalletToPlayerID(wallet string) string {
	wallet = strings.ToLower(strings.TrimSpace(wallet))
	if wallet == "" {
		return ""
	}
	h.lobby.mutex.RLock()
	defer h.lobby.mutex.RUnlock()
	if pid, ok := h.lobby.wallets[wallet]; ok && pid != "" {
		return pid
	}
	if _, ok := h.lobby.leaderboard[wallet]; ok {
		return wallet
	}
	return ""
}

// ============================================================================
// HELPER: Broadcast Justice WebSocket Event
// ============================================================================

func (h *JusticeHandlers) broadcastJusticeEvent(eventType string, payload map[string]interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Justice] Failed to serialize event %s: %v", eventType, err)
		return
	}
	msg := Envelope{
		Type:    "justice_event",
		FromID:  "SERVER",
		Payload: json.RawMessage(fmt.Sprintf(`{"type":"%s","data":%s}`, eventType, string(data))),
	}
	out, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[Justice] Failed to serialize envelope: %v", err)
		return
	}
	h.lobby.mutex.RLock()
	defer h.lobby.mutex.RUnlock()
	for _, client := range h.lobby.clients {
		select {
		case client.send <- out:
		default:
			// Skip slow clients to avoid blocking broadcast
		}
	}
}

// ============================================================================
// HELPER: Get Dashboard for Player (internal)
// ============================================================================

func (h *JusticeHandlers) getDashboardForPlayer(playerID string) JusticeDashboardAPI {
	dash := APIResponseJSON{}
	result := JusticeDashboardAPI{PowerBonus: 0, Tier: 0, TruthSerumActive: false, ShieldRemaining: 0}

	h.lobby.mutex.RLock()
	defer h.lobby.mutex.RUnlock()

	faction := h.lobby.justiceService.GetJusticeFaction(playerID)
	if faction == nil {
		return result
	}

	result.Tier = faction.BountyRank

	// Compute power bonus from Justice cards (sum of PowerBonus as percentage points)
	var totalPower int64
	for _, card := range faction.JusticeCards {
		totalPower += int64(card.PowerBonus)
	}
	if totalPower > 0 {
		result.PowerBonus = int(totalPower / 10000) // Convert micro-units to percentage points (simplified)
	}

	// Check truth serum active status
	revealed := h.lobby.justiceService.GetRevealedBuffs(playerID)
	if len(revealed) > 0 {
		result.TruthSerumActive = true
	}

	// Get shield remaining
	result.ShieldRemaining = h.lobby.justiceService.GetShieldRemaining(playerID)

	return result
}

// ============================================================================
// ENDPOINT 1: GET /api/justice/dashboard (JSON)
// ============================================================================

func (h *JusticeHandlers) handleGetDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract player wallet from query param or session
	wallet := strings.TrimSpace(r.URL.Query().Get("wallet"))
	if wallet == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponseJSON{Success: false, Message: "missing wallet parameter"})
		return
	}

	playerID := h.resolveWalletToPlayerID(wallet)
	if playerID == "" {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIResponseJSON{Success: false, Message: "player not found"})
		return
	}

	dash := h.getDashboardForPlayer(playerID)

	// Fetch bounty targets from wanted players in the system
	h.lobby.mutex.RLock()
	for wAddr, stats := range h.lobby.leaderboard {
		if wAddr == wallet {
			continue // Skip self
		}
		wantedLevel := 0
		if stats.Wanted != nil {
			wantedLevel = *stats.Wanted
		}
		if wantedLevel >= JusticeOutlawBonusThreshold {
			dash.Bounties = append(dash.Bounties, BountyItemJSON{
				TargetWallet: wAddr,
				TargetName:   stats.Name,
				WantedLevel:  wantedLevel,
				Reward:       uint64(wantedLevel) * 100_000, // Reward scales with wanted level in micro-units
			})
		}
	}
	h.lobby.mutex.RUnlock()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dash)

	// Broadcast dashboard_refresh to all connected clients
	h.broadcastJusticeEvent("dashboard_refresh", map[string]interface{}{
		"playerID": playerID,
	})
}

// ============================================================================
// ENDPOINT 2: POST /api/justice/use-truth-serum (JSON)
// ============================================================================

func (h *JusticeHandlers) handleUseTruthSerum(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req UseTruthSerumRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponseJSON{Success: false, Message: "invalid request body"})
		return
	}

	targetWallet := strings.TrimSpace(req.TargetWallet)
	if targetWallet == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponseJSON{Success: false, Message: "missing targetWallet"})
		return
	}

	targetPlayerID := h.resolveWalletToPlayerID(targetWallet)
	if targetPlayerID == "" {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIResponseJSON{Success: false, Message: "target player not found"})
		return
	}

	h.lobby.justiceService.ApplyTruthSerum(
		targetPlayerID,
		nil, // Revealed buffs resolved by service internally
		TruthSerumDefaultDuration,
	)

	// Broadcast WebSocket event
	h.broadcastJusticeEvent("truth_serum_applied", map[string]interface{}{
		"targetWallet": targetWallet,
		"duration":     TruthSerumDefaultDuration.Seconds(),
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(APIResponseJSON{
		Success:  true,
		Duration: int(TruthSerumDefaultDuration.Seconds()),
	})
}

// ============================================================================
// ENDPOINT 3: POST /api/justice/capture-bounty (JSON)
// ============================================================================

func (h *JusticeHandlers) handleCaptureBounty(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req CaptureBountyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponseJSON{Success: false, Message: "invalid request body"})
		return
	}

	targetWallet := strings.TrimSpace(req.TargetWallet)
	if targetWallet == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponseJSON{Success: false, Message: "missing targetWallet"})
		return
	}

	targetPlayerID := h.resolveWalletToPlayerID(targetWallet)
	if targetPlayerID == "" {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIResponseJSON{Success: false, Message: "target player not found"})
		return
	}

	h.lobby.mutex.RLock()
	faction := h.lobby.justiceService.GetJusticeFaction(targetPlayerID)
	if faction == nil {
		h.lobby.mutex.RUnlock()
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIResponseJSON{Success: false, Message: "target has no bounty data"})
		return
	}

	var reward uint64 = 1_500_000 // Default base reward in micro-units (1.5 $VBV)
	if faction.BountyRank > 0 {
		reward = uint64(faction.BountyRank) * 500_000 // Scale by bounty rank
	}

	h.lobby.mutex.RUnlock()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(APIResponseJSON{Success: true, Reward: reward})

	// Broadcast WebSocket event
	h.broadcastJusticeEvent("bounty_updated", map[string]interface{}{
		"targetWallet": targetWallet,
		"wantedLevel":  faction.BountyRank * 10, // Convert rank to wanted level scale
		"reward":       reward,
	})

	log.Printf("[Justice] Bounty captured: %s → %s (reward: %d micro-VBV)", targetPlayerID, targetWallet, reward)
}

// ============================================================================
// ENDPOINT 4: GET /api/justice/bounty-board (JSON — alias to dashboard)
// ============================================================================

func (h *JusticeHandlers) handleBountyBoard(w http.ResponseWriter, r *http.Request) {
	// Convenience alias for frontend compatibility with bounty board access.
	h.handleGetDashboard(w, r)
}

// ============================================================================
// ENDPOINT 5: POST /api/justice/award-card (JSON — award justice card to player)
// ============================================================================

type AwardJusticeCardRequest struct {
	PlayerID   string                  `json:"player_id"`    // Player ID of recipient
	CardType   JusticeCardType         `json:"card_type"`    // Card archetype: ENFORCER, MEDIATOR, WARDEN, COMMISSIONER
}

func (h *JusticeHandlers) handleAwardJusticeCard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req AwardJusticeCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponseJSON{Success: false, Message: "invalid request body"})
		return
	}

	playerID := strings.TrimSpace(req.PlayerID)
	cardType := JusticeCardType(strings.ToUpper(string(req.CardType)))
	if playerID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponseJSON{Success: false, Message: "missing player_id"})
		return
	}
	if cardType == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponseJSON{Success: false, Message: "missing card_type"})
		return
	}

	err := h.lobby.justiceService.AwardJusticeCard(playerID, cardType)
	if err != nil {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(APIResponseJSON{Success: false, Message: err.Error()})
		return
	}

	// Broadcast WebSocket event
	h.broadcastJusticeEvent("justice_card_awarded", map[string]interface{}{
		"playerID": playerID,
		"cardType": string(cardType),
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(APIResponseJSON{Success: true, CardType: string(cardType)})
}

// ============================================================================
// ENDPOINT 6: POST /api/justice/use-rep-shield (JSON — apply reputation shield)
// ============================================================================

type ApplyRepShieldRequest struct {
	PlayerID   string `json:"player_id"`    // Player ID of recipient
	Protection int    `json:"protection"`   // Protection amount (default 50 if omitted)
	DurationSec int   `json:"duration_sec"` // Duration in seconds (default 3600 = 1 hour if omitted)
}

func (h *JusticeHandlers) handleApplyRepShield(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req ApplyRepShieldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponseJSON{Success: false, Message: "invalid request body"})
		return
	}

	playerID := strings.TrimSpace(req.PlayerID)
	if playerID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponseJSON{Success: false, Message: "missing player_id"})
		return
	}

	protection := req.Protection
	if protection <= 0 {
		protection = ReputationShieldDefaultProtection // Use constant from service
	}

	durationSec := req.DurationSec
	if durationSec <= 0 {
		durationSec = 3600 // Default: 1 hour
	}

	h.lobby.justiceService.ApplyReputationShield(playerID, protection, time.Duration(durationSec)*time.Second)

	// Broadcast WebSocket event
	shieldRemaining := h.lobby.justiceService.GetShieldRemaining(playerID)
	h.broadcastJusticeEvent("shield_active", map[string]interface{}{
		"playerID":   playerID,
		"remaining":  shieldRemaining,
		"capacity":   protection,
		"durationSec": durationSec,
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(APIResponseJSON{Success: true, Remaining: shieldRemaining, Capacity: protection})
}
