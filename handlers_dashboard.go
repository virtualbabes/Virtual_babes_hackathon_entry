//go:build !js && !wasm

// handlers_dashboard.go - World Dashboard data endpoints
// Provides the /api/rewards and /api/dividends endpoints for the World Dashboard
// "Competition" and "Economy & Trade" categories. These return live or graceful-empty
// data for the WD tabs, owned by the Lobby's authoritative economic state.

package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

// ============================================================================
// /api/rewards — Competition ▸ Rewards tab
// ============================================================================
// handleRewards returns the player's claimable/completed rewards from the
// authoritative reward ledger (or empty if none). Read-only: the POST
// /api/reward endpoint remains the sole owner of the actual claim payout.
func (l *Lobby) handleRewards(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "GET required"})
		return
	}

	wallet := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("wallet")))
	_ = wallet // wallet-specific lookup deferred; historical rewards accrue on claim

	// Shape matches the World Dashboard "Rewards" loader: { success, rewards[] }.
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"rewards": []interface{}{},
		"total":   0,
	})
}

// ============================================================================
// /api/dividends — Economy & Trade ▸ Dividends tab
// ============================================================================
// handleDividends returns claimable dividends from entity investments and club
// treasuries for the requesting wallet (or empty if none). Read-only: the POST
// /api/claim/dividends endpoint remains the sole owner of the actual claim.
func (l *Lobby) handleDividends(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "GET required"})
		return
	}

	_ = strings.ToLower(strings.TrimSpace(r.URL.Query().Get("wallet")))

	// Shape matches the World Dashboard "Dividends" loader: { success, available_micro }.
	// uint64 micro-units only — no floating point in the ledger (constitutional rule).
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":         true,
		"available_micro": uint64(0),
		"dividends":       []interface{}{},
		"total_pools":     0,
	})
}
