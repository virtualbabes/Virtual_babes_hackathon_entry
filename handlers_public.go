package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
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
		ActiveMatches: len(l.matches) / 2,
		TotalPlayers:  len(l.clients),
		Timestamp:     time.Now(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (l *Lobby) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	status := struct {
		Status      string  `json:"status"`
		Connections int     `json:"connections"`
		Vault       float64 `json:"vault_balance"`
	}{Status: "ok", Connections: len(l.clients), Vault: l.faucetBalance}
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
