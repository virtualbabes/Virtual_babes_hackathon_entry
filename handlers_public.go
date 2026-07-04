//go:build !js && !wasm

package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/client/v2/algod"
)

// handleSpectatorWager processes spectator bets on ongoing matches.
// PILLAR 2: Industrial Loop (Spectator Siphon).
func (l *Lobby) handleSpectatorWager(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SpectatorWallet string `json:"spectator_wallet"`
		MatchID         string `json:"match_id"` // This should be the P1ID of the match
		BetOnWallet     string `json:"bet_on_wallet"`
		WagerMicro      uint64 `json:"wager_micro"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SpectatorWallet == "" || req.MatchID == "" || req.BetOnWallet == "" || req.WagerMicro == 0 {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	spectatorWallet := strings.ToLower(req.SpectatorWallet)
	betOnWallet := strings.ToLower(req.BetOnWallet)

	l.mutex.Lock()
	defer l.mutex.Unlock()

	// 1. Validate Spectator's Balance
	if l.playerBalances[spectatorWallet] < req.WagerMicro {
		http.Error(w, "Insufficient rewards for wager", http.StatusPaymentRequired)
		return
	}

	// 2. Find the Match and Validate Participants
	match, ok := l.matches[req.MatchID] // MatchState is keyed by P1ID
	if !ok {
		http.Error(w, "Match not found or no longer active", http.StatusNotFound)
		return
	}

	// Ensure the match is still active and not finished
	if match.IsFinished {
		http.Error(w, "Cannot place wager on a finished match", http.StatusForbidden)
		return
	}

	// Ensure the spectator is actually spectating this match
	isSpectator := false
	for _, sID := range match.Spectators {
		if sWallet, ok := l.wallets[sID]; ok && strings.EqualFold(sWallet, spectatorWallet) {
			isSpectator = true
			break
		}
	}
	if !isSpectator {
		http.Error(w, "You are not spectating this match", http.StatusForbidden)
		return
	}
	// PILLAR 2: Fair Play Enforcement. Prevent players from betting on their own matches.
	if strings.EqualFold(match.P1Wallet, spectatorWallet) || strings.EqualFold(match.P2Wallet, spectatorWallet) {
		http.Error(w, "You cannot place a wager on your own match.", http.StatusForbidden)
		return
	}

	// Ensure betOnWallet is a valid participant in the match
	if !strings.EqualFold(match.P1Wallet, betOnWallet) && !strings.EqualFold(match.P2Wallet, betOnWallet) {
		http.Error(w, "Invalid player to bet on", http.StatusBadRequest)
		return
	}

	// 3. Deduct Wager from Spectator and Add to Match Pool
	l.playerBalances[spectatorWallet] -= req.WagerMicro
	match.WagersMicro += req.WagerMicro

	// 4. Log the event
	l.logAdminAuditLocked("SPECTATOR_WAGER", spectatorWallet, fmt.Sprintf("Match: %s, BetOn: %s, Amount: %d micro-VBV", req.MatchID, betOnWallet, req.WagerMicro))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "message": "Wager placed successfully"})
}

func (l *Lobby) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	type entry struct {
		Wins             int       `json:"wins"`
		DNFs             int       `json:"dnfs"`
		DisconnectStreak int       `json:"disconnect_streak"`
		Reputation       int       `json:"reputation"`
		BestRating       string    `json:"best_rating"`
		BanExpires       time.Time `json:"ban_expires"`
		Wallet           string    `json:"wallet"`
		TotalDonated     uint64    `json:"total_donated"`
		ReparationsReceivedCount int `json:"reparations_received_count"`
	}
	var list []entry
	l.mutex.RLock()
	for w, stats := range l.leaderboard {
		list = append(list, entry{
			Wins: stats.Wins, DNFs: stats.DNFs, DisconnectStreak: stats.DisconnectStreak,
			Reputation: stats.Reputation, BestRating: stats.BestRating,
			BanExpires: stats.BanExpires, Wallet: w,
			TotalDonated: stats.TotalDonated,
			ReparationsReceivedCount: stats.ReparationsReceivedCount,
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
		ActiveMatches: l.countUniqueMatchesLocked(), // Consistent load telemetry
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
	nodeReport := l.getHealthReportLocked()
	l.mutex.RUnlock()

	isHealthy := true
	var errs []string

	// 1. Verify RPC Connectivity (Cycle through all available nodes with LlamaRPC failover)
	rpcResponded := false
	if ok && len(voiConfig.NodeURLs) > 0 {
		for _, url := range voiConfig.NodeURLs {
			client, _ := algod.MakeClient(url, "")
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			err := client.HealthCheck().Do(ctx)
			cancel()
			if err == nil {
				rpcResponded = true
				break
			}
			log.Printf("[HEALTH] Node unreachable: %s - %v\n", url, err)
		}
		if !rpcResponded {
			isHealthy = false
			errs = append(errs, "rpc_unreachable")
		}
	} else {
		isHealthy = false
		errs = append(errs, "config_missing")
	}

	// 2. Verify Faucet Liquidity (Gas Check)
	if balance < 1.0 {
		isHealthy = false
		errs = append(errs, "low_liquidity")
	}

	status := struct {
		Status      string            `json:"status"`
		Connections int               `json:"connections"`
		Vault       float64           `json:"vault_balance"`
		Nodes       []NodeHealthReport `json:"nodes,omitempty"`
		Errors      []string          `json:"errors,omitempty"`
	}{Status: "ok", Connections: clientsCount, Vault: balance, Nodes: nodeReport}

	if !isHealthy {
		status.Status = "unhealthy"
		status.Errors = errs
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// getHealthReportLocked returns the node health report. Must be called under l.mutex.RLock.
func (l *Lobby) getHealthReportLocked() []NodeHealthReport {
	if l.ledgerClient == nil {
		return nil
	}
	return l.ledgerClient.GetHealthReport()
}

// handleLiveEndpoint provides the /live Kubernetes-compatible liveliness probe.
// This endpoint always returns 200 as long as the process is running, allowing
// Kubernetes/Render to detect if the server process is alive regardless of external dependencies.
func (l *Lobby) handleLiveEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now().Format(time.RFC3339),
		"uptime":    time.Since(l.seasonStart).Round(time.Second).String(),
	})
}

// handleReadyEndpoint provides the /ready Kubernetes-compatible readiness probe.
// This endpoint returns 200 only when all critical dependencies are operational,
// including RPC connectivity and faucet liquidity.
func (l *Lobby) handleReadyEndpoint(w http.ResponseWriter, r *http.Request) {
	l.mutex.RLock()
	voiConfig, ok := l.availableNetworks["Voi Mainnet"]
	balance := l.faucetBalance
	nodeReport := l.getHealthReportLocked()
	l.mutex.RUnlock()

	isReady := true
	var notReady []string

	// Check RPC connectivity
	if !ok || len(voiConfig.NodeURLs) == 0 {
		isReady = false
		notReady = append(notReady, "voi_config_missing")
	} else {
		for _, url := range voiConfig.NodeURLs {
			client, _ := algod.MakeClient(url, "")
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			err := client.HealthCheck().Do(ctx)
			cancel()
			if err == nil {
				break
			}
			notReady = append(notReady, fmt.Sprintf("node_down:%s", url))
			isReady = false
		}
	}

	// Check faucet liquidity
	if balance < 1.0 {
		isReady = false
		notReady = append(notReady, "low_liquidity")
	}

	statusCode := http.StatusOK
	if !isReady {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      map[string]bool{"ready": isReady},
		"nodes":       nodeReport,
		"vault_balance": balance,
		"not_ready":   notReady,
	})
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
		ID         string    `json:"id"`
		P1         string    `json:"p1_id"`
		P2         string    `json:"p2_id"`
		Rating     string    `json:"rating"`
		Territory  string    `json:"territory"`
		Spectators int       `json:"spectator_count"`
		StartTime  time.Time `json:"start_time"`
	}

	active := []matchSummary{} // Ensure empty array instead of null in JSON
	seen := make(map[*MatchState]bool)

	for _, m := range l.matches {
		// PILLAR 4: Broadcasting Accuracy.
		// Only show matches that have been paired (P2ID present) and are actively in combat.
		if seen[m] || m.IsFinished || m.P2ID == "" {
			continue
		}

		// Use P1's ID as the primary Match ID for routing
		active = append(active, matchSummary{
			ID:         m.P1ID,
			P1:         m.P1ID,
			P2:         m.P2ID,
			Rating:     m.MatchRating, // Correctly uses snapshot to survive Sudden Death
			StartTime:  m.StartTime,
			Territory:  m.TerritoryID,
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
