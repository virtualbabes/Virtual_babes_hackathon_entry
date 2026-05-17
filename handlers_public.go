//go:build !js || !wasm

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/client/v2/algod"
)

func (l *Lobby) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	type entry struct {
		Wins             int       `json:"wins"`
		DNFs             int       `json:"dnfs"`
		DisconnectStreak int       `json:"disconnect_streak"`
		Reputation       int       `json:"reputation"`
		BestRating       string    `json:"best_rating"`
		BanExpires       time.Time `json:"ban_expires"`
		Wallet           string    `json:"wallet"`
	}
	var list []entry
	l.mutex.RLock()
	for w, stats := range l.leaderboard {
		list = append(list, entry{
			Wins: stats.Wins, DNFs: stats.DNFs, DisconnectStreak: stats.DisconnectStreak,
			Reputation: stats.Reputation, BestRating: stats.BestRating,
			BanExpires: stats.BanExpires, Wallet: w,
		})
	}
	l.mutex.RUnlock()

	sort.Slice(list, func(i, j int) bool { return list[i].Wins > list[j].Wins })
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// handlePublicStatus provides public-facing statistics for external sites (e.g., Carrd.co).
func (l *Lobby) handlePublicStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	l.mutex.RLock()
	defer l.mutex.RUnlock()

	status := struct {
		FaucetBalance float64   `json:"faucet_balance"`
		Maintenance   bool      `json:"maintenance_mode"`
		ActiveMatches int       `json:"active_matches"`
		TotalPlayers  int       `json:"total_players"`
		Timestamp     time.Time `json:"timestamp"`
	}{
		FaucetBalance: l.faucetBalance,
		Maintenance:   l.maintenanceMode,
		ActiveMatches: l.countUniqueMatchesLocked(), // Fixed: Uses unique match counting
		TotalPlayers:  len(l.clients),
		Timestamp:     time.Now(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (l *Lobby) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	// PILLAR 4: High-Fidelity Health Monitoring.
	// This endpoint allows Render's load balancer to verify that the server
	// is not just responsive, but has active connectivity and liquidity.
	l.mutex.RLock()
	voiConfig, ok := l.availableNetworks["Voi Mainnet"]
	balance := l.faucetBalance
	clientsCount := len(l.clients)
	l.mutex.RUnlock()

	isHealthy := true
	var errs []string

	// 1. Verify RPC Connectivity (Ping the primary Voi node)
	if ok && len(voiConfig.NodeURLs) > 0 {
		client, _ := algod.MakeClient(voiConfig.NodeURLs[0], "")
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := client.HealthCheck().Do(ctx); err != nil {
			isHealthy = false
			errs = append(errs, "rpc_unreachable")
		}
	} else {
		isHealthy = false
		errs = append(errs, "config_missing")
	}

	// 2. Verify Faucet Liquidity (Gas Check)
	// The Arena requires at least 1.0 VOI/VBV in the vault to function correctly.
	if balance < 1.0 {
		isHealthy = false
		errs = append(errs, "low_liquidity")
	}

	status := struct {
		Status      string   `json:"status"`
		Connections int      `json:"connections"`
		Vault       float64  `json:"vault_balance"`
		Errors      []string `json:"errors,omitempty"`
	}{Status: "ok", Connections: clientsCount, Vault: balance}

	if !isHealthy {
		status.Status = "unhealthy"
		status.Errors = errs
		w.WriteHeader(http.StatusServiceUnavailable) // Return 503 so Render triggers a restart
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (l *Lobby) handleCardStats(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	card, err := l.getVerifiedCard("", id, "Voi Mainnet")
	if err != nil {
		http.Error(w, "Card verification failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(card)
}

func (l *Lobby) handleGetCardDetails(w http.ResponseWriter, r *http.Request) {
	idsStr := r.URL.Query().Get("ids")
	network := r.URL.Query().Get("network")
	wallet := r.URL.Query().Get("wallet")
	if network == "" {
		network = "Voi Mainnet"
	}

	var ids []int
	for _, s := range strings.Split(idsStr, ",") {
		if id, err := strconv.Atoi(s); err == nil {
			ids = append(ids, id)
		}
	}

	cards, err := l.getVerifiedCards(wallet, ids, network)
	if err != nil {
		http.Error(w, "Metadata retrieval failed", http.StatusInternalServerError)
		return
	}
	var results []ServerCard
	for _, id := range ids {
		if c, ok := cards[id]; ok {
			results = append(results, c)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (l *Lobby) getVerifiedCard(wallet string, tokenID int, networkName string) (ServerCard, error) {
	cards, err := l.getVerifiedCards(wallet, []int{tokenID}, networkName)
	if err != nil || len(cards) == 0 {
		return ServerCard{}, err
	}
	return cards[tokenID], nil
}

// handleActiveMatches returns a list of ongoing matches for the spectator portals.
func (l *Lobby) handleActiveMatches(w http.ResponseWriter, r *http.Request) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	type matchSummary struct {
		ID        string   `json:"id"`
		P1        string   `json:"p1_id"`
		P2        string   `json:"p2_id"`
		Rating    string   `json:"rating"`
		Territory string   `json:"territory"`
		Spectators int     `json:"spectator_count"`
		StartTime  time.Time `json:"start_time"`
	}

	var active []matchSummary
	seen := make(map[*MatchState]bool)

	for _, m := range l.matches {
		if seen[m] || m.IsFinished {
			continue
		}
		
		// Use P1's ID as the primary Match ID for routing
		active = append(active, matchSummary{
			ID:        m.P1ID,
			P1:        m.P1ID,
			P2:        m.P2ID,
			Rating:    l.calculateDeckRating(m.P1Deck),
			Territory: m.TerritoryID,
			Spectators: len(m.Spectators),
		})
		seen[m] = true
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":   len(active),
		"matches": active,
	})
}
}
