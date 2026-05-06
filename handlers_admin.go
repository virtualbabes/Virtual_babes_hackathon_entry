package main

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/types"
)

// logAdminAudit records an administrative action to a separate file for permanent record keeping.
func (l *Lobby) logAdminAudit(action, target, details string) {
	l.mutex.RLock()
	load := len(l.matches) / 2
	l.mutex.RUnlock()

	entry := struct {
		Timestamp  string `json:"timestamp"`
		Action     string `json:"action"`
		Target     string `json:"target"`
		Details    string `json:"details"`
		ServerLoad int    `json:"server_load"`
	}{
		Timestamp:  time.Now().Format(time.RFC3339),
		Action:     action,
		Target:     target,
		Details:    details,
		ServerLoad: load,
	}

	b, _ := json.Marshal(entry)
	f, err := os.OpenFile("admin_audit.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("[AUDIT ERROR] Failed to write to admin log: %v\n", err)
		return
	}
	defer f.Close()
	f.Write(append(b, '\n'))
}

// broadcastToAdmins sends a high-priority system message to all connected administrators.
func (l *Lobby) broadcastToAdmins(text string) {
	payload, _ := json.Marshal(map[string]string{"text": text})
	env := Envelope{
		Type:    "chat",
		FromID:  "SERVER",
		Payload: payload,
	}
	msg, _ := json.Marshal(env)

	l.mutex.RLock()
	defer l.mutex.RUnlock()
	for _, client := range l.clients {
		if client.isAdmin {
			select {
			case client.send <- msg:
			default:
			}
		}
	}
}

// escapeHTML escapes HTML special characters in a string to prevent XSS.
func escapeHTML(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&#x27;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// broadcastHealthReport constructs and sends a real-time status update of the arena to all clients.
func (l *Lobby) broadcastHealthReport() {
	l.mutex.RLock()
	activeMatches := len(l.matches) / 2
	balance := l.faucetBalance
	l.mutex.RUnlock()

	healthText := fmt.Sprintf("[SERVER HEALTH] Active Arena Matches: %d | Vault Balance: %.2f $VBV", activeMatches, balance)
	payload, _ := json.Marshal(map[string]string{"text": healthText})

	update := Envelope{
		Type:    "chat",
		FromID:  "SERVER",
		Payload: payload,
	}
	msg, _ := json.Marshal(update)
	l.broadcast <- msg
	log.Printf("[SERVER] Automated health report broadcasted: %s\n", healthText)
}

func (l *Lobby) handleRefillVault(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		l.logAdminAudit("REFILL_VAULT_AUTH_FAIL", r.RemoteAddr, "Unauthorized attempt to refill vault")
		return
	}
	var req struct {
		Amount float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount <= 0 {
		http.Error(w, "Invalid refill amount", http.StatusBadRequest)
		return
	}
	l.mutex.Lock()
	l.faucetBalance = req.Amount
	l.applyDynamicScaling()
	l.mutex.Unlock()
	update := Envelope{Type: "vault_update", FromID: "SERVER", Payload: json.RawMessage(fmt.Sprintf(`{"balance": %f}`, req.Amount))}
	msg, _ := json.Marshal(update)
	l.broadcast <- msg
	l.logAdminAudit("REFILL_VAULT", "GLOBAL", fmt.Sprintf("Amount: %.2f", req.Amount))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "new_balance": req.Amount})
}

func (l *Lobby) handleUpdateRules(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}
	var req struct{ Open, Power_copy, Power_up bool }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	payload, _ := json.Marshal(req)
	l.broadcast <- jsonListEnvelope("rules_update", payload)
	l.logAdminAudit("UPDATE_RULES", "GLOBAL", fmt.Sprintf("Open: %v, Power_copy: %v, Power_up: %v", req.Open, req.Power_copy, req.Power_up))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "rules": req})
}

func (l *Lobby) handleAdminAddReward(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}
	var req struct {
		AssetID string  `json:"asset_id"`
		Amount  float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.AssetID != "" && req.AssetID != "0" {
		if optedIn, _, err := l.checkAssetOptIn("VOI", l.vaultAddress, req.AssetID); err != nil || !optedIn {
			http.Error(w, "Vault not opted-in to asset", http.StatusBadRequest)
			return
		}
	}
	l.mutex.Lock()
	l.rewards[req.AssetID] = uint64(req.Amount * 1000000)
	l.initialRewards[req.AssetID] = l.rewards[req.AssetID]
	l.saveSeasonMetadataLocked() // Ensure persistence
	l.mutex.Unlock()
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	l.logAdminAudit("ADD_REWARD", fmt.Sprintf("Asset %s", req.AssetID), fmt.Sprintf("Base Amount: %.2f", req.Amount))
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success"})
}

func (l *Lobby) handleAdminRemoveReward(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}
	var req struct {
		AssetID string `json:"asset_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	l.mutex.Lock()
	delete(l.rewards, req.AssetID)
	delete(l.initialRewards, req.AssetID)
	l.saveSeasonMetadataLocked() // Ensure persistence
	l.mutex.Unlock()
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	l.logAdminAudit("REMOVE_REWARD", fmt.Sprintf("Asset %s", req.AssetID), "Removed from stack")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success"})
}

func (l *Lobby) handleSetActiveNetwork(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}
	var req struct {
		NetworkName string `json:"network_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	l.mutex.Lock()
	if _, ok := l.availableNetworks[req.NetworkName]; !ok {
		l.mutex.Unlock()
		http.Error(w, "Network not found", http.StatusNotFound)
		return
	}
	l.adminFocusNetwork = req.NetworkName
	l.mutex.Unlock()
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	l.logAdminAudit("SET_ADMIN_FOCUS_NETWORK", "GLOBAL", req.NetworkName)
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success"})
}

func (l *Lobby) handleAddNetwork(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}
	var newConfig NetworkConfig
	if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if newConfig.NetworkName == "" || newConfig.NodeURL == "" || newConfig.ChainID == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}
	l.mutex.Lock()
	l.availableNetworks[newConfig.NetworkName] = newConfig
	l.mutex.Unlock()
	l.saveNetworkConfigs()
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	l.logAdminAudit("ADD_NETWORK", newConfig.NetworkName, "New network added")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success"})
}

func (l *Lobby) handleUpdatePowerScaling(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}
	var req struct {
		Divisor float64 `json:"divisor"`
		Base    int     `json:"base"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	l.mutex.Lock()
	if config, ok := l.availableNetworks[l.adminFocusNetwork]; ok {
		config.PowerDivisor, config.PowerBase = req.Divisor, req.Base
		l.availableNetworks[l.adminFocusNetwork] = config
	}
	l.mutex.Unlock()
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	l.logAdminAudit("UPDATE_POWER_SCALING", l.adminFocusNetwork, fmt.Sprintf("Divisor: %.2f, Base: %d", req.Divisor, req.Base))
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success"})
}

func (l *Lobby) handleSystemMessage(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Text == "" {
		http.Error(w, "Invalid message", http.StatusBadRequest)
		return
	}
	if req.Text == "@health" {
		go l.broadcastHealthReport()
		l.logAdminAudit("SYSTEM_MESSAGE", "GLOBAL", "Manual Health Report Triggered")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "success"})
		return
	}
	payload, _ := json.Marshal(map[string]string{"text": "[ADMIN] " + req.Text})
	l.broadcast <- jsonListEnvelope("chat", payload)
	l.logAdminAudit("SYSTEM_MESSAGE", "GLOBAL", req.Text)
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success"})
}

func (l *Lobby) handleBanPlayer(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}
	var req struct {
		Wallet string `json:"wallet"`
		Hours  int    `json:"hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Wallet == "" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if req.Hours <= 0 {
		req.Hours = 24
	}
	l.mutex.Lock()
	stats := l.leaderboard[req.Wallet]
	stats.BanExpires = time.Now().Add(time.Duration(req.Hours) * time.Hour)
	stats.DNFs++
	stats.DisconnectStreak++
	stats.Reputation = l.CalculateReputation(stats)
	l.leaderboard[req.Wallet] = stats
	for clientID, wallet := range l.wallets {
		if wallet == req.Wallet {
			if match, ok := l.matches[clientID]; ok {
				opp := match.P1ID
				if clientID == match.P1ID {
					opp = match.P2ID
				}
				l.sendToClient(opp, Envelope{Type: "chat", FromID: "SERVER", Payload: json.RawMessage(`{"text":"Match terminated: Opponent restricted."}`)})
				delete(l.matches, opp)
				delete(l.matches, clientID)
			}
		}
	}
	msg := l.getLobbyUpdateMsgLocked()
	l.mutex.Unlock()
	l.broadcast <- msg
	l.logAdminAudit("BAN_PLAYER", req.Wallet, fmt.Sprintf("%d hours", req.Hours))
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success"})
}

func (l *Lobby) handleGloatBan(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}
	var req struct {
		Wallet string `json:"wallet"`
		Hours  int    `json:"hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Wallet == "" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	l.mutex.Lock()
	stats := l.leaderboard[req.Wallet]
	stats.GloatBannedUntil = time.Now().Add(time.Duration(req.Hours) * time.Hour)
	l.leaderboard[req.Wallet] = stats
	l.mutex.Unlock()
	l.logAdminAudit("GLOAT_BAN", req.Wallet, fmt.Sprintf("%d hours", req.Hours))
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success"})
}

func (l *Lobby) handleAvatarBan(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}
	var req struct {
		URL   string `json:"url"`
		Hours int    `json:"hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		http.Error(w, "Invalid request body or missing URL", http.StatusBadRequest)
		return
	}

	targetURL := strings.TrimSpace(req.URL)
	if req.Hours <= 0 {
		req.Hours = 720 // Default 30 days
	}

	l.mutex.Lock()
	expiry := time.Now().Add(time.Duration(req.Hours) * time.Hour)
	l.bannedAvatars[targetURL] = expiry

	// Immediate Enforcement: Boot anyone currently using this avatar
	affectedCount := 0
	for _, client := range l.clients {
		if client.avatarURL == targetURL {
			client.avatarURL = safeAvatarPool[rand.Intn(len(safeAvatarPool))]
			client.avatarBanNotice = "Your avatar was restricted by an administrator."
			l.sendToClient(client.id, Envelope{
				Type:    "admin_notification",
				FromID:  "SERVER",
				Payload: json.RawMessage(`{"text":"🚨 <b>MODERATION:</b> Your profile image has been restricted globally."}`),
			})
			affectedCount++
		}
	}
	msg := l.getLobbyUpdateMsgLocked()
	l.mutex.Unlock()

	l.broadcast <- msg
	l.logAdminAudit("AVATAR_BAN", targetURL, fmt.Sprintf("Duration: %d hours. Affected users: %d", req.Hours, affectedCount))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "success",
		"url":            targetURL,
		"affected_count": affectedCount,
	})
}

func (l *Lobby) handleResetStats(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}
	var req struct {
		Wallet string `json:"wallet"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Wallet == "" {
		http.Error(w, "Invalid wallet", http.StatusBadRequest)
		return
	}
	l.mutex.Lock()
	if _, exists := l.leaderboard[req.Wallet]; exists {
		l.leaderboard[req.Wallet] = PlayerStats{
			Reputation: 100,
			Inventory: make(map[string]int),
			JailedCards: make(map[int]string),
			// Initialize new maps for Kidnap Gambit
			FavoriteCardID: 0,
			KidnappedCards: make(map[int]string),
			HeldHostageCards: make(map[int]string),
			RumorCount: 0,
			Playstyle: PlaystyleTendencies{
				PreferredRules: make(map[string]float64),
				PreferredCardMoods: make(map[string]float64),
				PreferredItems: make(map[string]float64),
			},
		}
		msg := l.getLobbyUpdateMsgLocked()
		l.mutex.Unlock()
		l.broadcast <- msg
		l.logAdminAudit("RESET_STATS", req.Wallet, "Metrics cleared")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	} else {
		l.mutex.Unlock()
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

func (l *Lobby) handleUpdateBaseReward(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}
	var req struct {
		Amount float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount < 0 {
		http.Error(w, "Invalid reward", http.StatusBadRequest)
		return
	}
	l.mutex.Lock()
	l.baseReward = uint64(req.Amount * 1000000)
	l.initialBaseReward = l.baseReward
	l.initialRewards[l.rewardAssetID] = l.initialBaseReward
	l.applyDynamicScaling()
	l.saveSeasonMetadataLocked() // Ensure persistence
	l.mutex.Unlock()
	l.broadcast <- jsonListEnvelope("reward_update", json.RawMessage(fmt.Sprintf(`{"amount": %f}`, req.Amount)))
	l.logAdminAudit("UPDATE_REWARD", "GLOBAL", fmt.Sprintf("%.2f", req.Amount))
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success"})
}

func (l *Lobby) handleMaintenanceMode(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}
	var req struct {
		Active  bool `json:"active"`
		Minutes int  `json:"minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	l.mutex.Lock()
	l.maintenanceMode = req.Active
	l.maintenanceTime = time.Now().Add(time.Duration(req.Minutes) * time.Minute)
	msg := jsonListEnvelope("maintenance_update", json.RawMessage(fmt.Sprintf(`{"active":%v,"timestamp":"%s"}`, req.Active, l.maintenanceTime.Format(time.RFC3339))))
	l.mutex.Unlock()
	l.broadcast <- msg
	l.logAdminAudit("MAINTENANCE_MODE", "GLOBAL", fmt.Sprintf("Active: %v", req.Active))
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success"})
}

func (l *Lobby) handleUpdateRewardAsset(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}
	var req struct {
		AssetID string `json:"asset_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid asset", http.StatusBadRequest)
		return
	}
	l.mutex.Lock()
	l.rewardAssetID = req.AssetID
	// Migrating the unscaled base value to the new primary asset
	l.initialRewards[l.rewardAssetID] = l.initialBaseReward
	l.saveSeasonMetadataLocked() // Ensure persistence
	l.mutex.Unlock()
	l.broadcast <- jsonListEnvelope("asset_update", json.RawMessage(fmt.Sprintf(`{"asset_id": "%s"}`, req.AssetID)))
	l.logAdminAudit("UPDATE_ASSET", "GLOBAL", req.AssetID)
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success"})
}

func (l *Lobby) handleStartTournament(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}
	var req struct {
		Size    int  `json:"size"`
		IsBuyIn bool `json:"is_buy_in"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || (req.Size != 8 && req.Size != 16) {
		http.Error(w, "Invalid size", http.StatusBadRequest)
		return
	}
	l.mutex.Lock()
	defer l.mutex.Unlock()
	type entry struct {
		wallet string
		wins   int
	}
	var hof []entry
	for w, s := range l.leaderboard {
		hof = append(hof, entry{wallet: w, wins: s.Wins})
	}
	sort.Slice(hof, func(i, j int) bool { return hof[i].wins > hof[j].wins })
	participants := []string{}
	pot := 500.0
	if req.IsBuyIn {
		elite := make(map[string]bool)
		for i := 0; i < len(hof) && i < 10; i++ {
			elite[hof[i].wallet] = true
			participants = append(participants, hof[i].wallet)
			if len(participants) >= req.Size {
				break
			}
		}
		for _, p := range l.paidParticipants {
			if len(participants) >= req.Size {
				break
			}
			if !elite[p] {
				participants = append(participants, p)
			}
		}
		if len(participants) < req.Size {
			http.Error(w, "Need more players", http.StatusBadRequest)
			return
		}
		pot += l.tournamentPotBonus
		l.tournamentPotBonus = 0
		l.paidParticipants = []string{}
		rand.Shuffle(len(participants), func(i, j int) { participants[i], participants[j] = participants[j], participants[i] })
	} else {
		if len(hof) < req.Size {
			http.Error(w, "Need more Hall of Fame players", http.StatusBadRequest)
			return
		}
		for i := 0; i < req.Size; i++ {
			participants = append(participants, hof[i].wallet)
		}
	}
	matches := []TournamentMatch{}
	seedMap := map[int][]int{8: {0, 7, 3, 4, 1, 6, 2, 5}, 16: {0, 15, 7, 8, 4, 11, 3, 12, 1, 14, 6, 9, 5, 10, 2, 13}}[req.Size]
	for i := 0; i < len(seedMap); i += 2 {
		matches = append(matches, TournamentMatch{ID: fmt.Sprintf("R1-M%d", (i/2)+1), P1: participants[seedMap[i]], P2: participants[seedMap[i+1]], Round: 1})
	}
	l.tournament = TournamentState{Active: true, Participants: participants, Matches: matches, CurrentRound: 1, Pot: pot, BuyInAmount: 50.0, IsBuyInMode: req.IsBuyIn}
	l.logAdminAudit("START_TOURNAMENT", "GLOBAL", fmt.Sprintf("Size: %d", req.Size))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(l.tournament)
}


func (l *Lobby) handleOpenRegistration(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}
	var req struct {
		BuyIn float64 `json:"buy_in"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	l.mutex.Lock()
	l.tournament = TournamentState{Active: true, CurrentRound: 0, BuyInAmount: req.BuyIn, Matches: []TournamentMatch{}, Participants: []string{}, OpenTime: time.Now()}
	l.paidParticipants = []string{}
	l.tournamentPotBonus = 0
	l.mutex.Unlock()
	l.broadcastTournamentState()
	l.logAdminAudit("OPEN_REGISTRATION", "GLOBAL", fmt.Sprintf("Buy-in: %.2f", req.BuyIn))
	json.NewEncoder(w).Encode(l.tournament)
}

func (l *Lobby) handleSimulateTournament(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}
	var req struct {
		Size    int  `json:"size"`
		IsBuyIn bool `json:"is_buy_in"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || (req.Size != 8 && req.Size != 16) {
		http.Error(w, "Invalid size (must be 8 or 16)", http.StatusBadRequest)
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[SIMULATION CRITICAL] Panic in tournament simulation: %v\n", r)
			}
		}()
		l.simulateTournament(req.Size, req.IsBuyIn)
		l.logAdminAudit("SIMULATE_TOURNAMENT", "GLOBAL", fmt.Sprintf("Size: %d, Buy-in: %v", req.Size, req.IsBuyIn))
		l.broadcastToAdmins(fmt.Sprintf("🏆 Tournament simulation (%d players) completed!", req.Size))
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "message": fmt.Sprintf("Simulating %d-player tournament...", req.Size)})
}

func (l *Lobby) handleGetAdminLogs(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}
	l.mutex.RLock()
	currentBalance := l.faucetBalance
	l.mutex.RUnlock()
	query := r.URL.Query()
	filter := strings.ToUpper(query.Get("filter"))
	startStr, endStr := query.Get("start_date"), query.Get("end_date")
	var start, end time.Time
	if startStr != "" {
		start, _ = time.Parse(time.RFC3339, startStr)
	}
	if endStr != "" {
		end, _ = time.Parse(time.RFC3339, endStr)
	}
	f, err := os.Open("admin_audit.log")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "error", "message": "No logs"})
		return
	}
	defer f.Close()
	stat, _ := f.Stat()
	size := stat.Size()
	readSize := int64(512 * 1024)
	if readSize > size {
		readSize = size
	}
	buffer := make([]byte, readSize)
	f.ReadAt(buffer, size-readSize)
	content := string(buffer)
	if size > readSize {
		if idx := strings.Index(content, "\n"); idx != -1 {
			content = content[idx+1:]
		}
	}
	lines := strings.Split(strings.TrimSpace(content), "\n")
	var results []json.RawMessage
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i] == "" {
			continue
		}
		if filter != "" && !strings.Contains(strings.ToUpper(lines[i]), filter) {
			continue
		}
		if !start.IsZero() || !end.IsZero() {
			var logData struct {
				Timestamp string `json:"timestamp"`
			}
			if json.Unmarshal([]byte(lines[i]), &logData) == nil {
				ts, _ := time.Parse(time.RFC3339, logData.Timestamp)
				if (!start.IsZero() && ts.Before(start)) || (!end.IsZero() && ts.After(end)) {
					continue
				}
			}
		}
		results = append(results, json.RawMessage(lines[i]))
		if len(results) >= 100 {
			break
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "balance": currentBalance, "logs": results})
}

// checkAdminAuth validates the administrator using either an Algorand signature (Preferred)
// or the legacy X-Admin-Key (Fallback).
func (l *Lobby) checkAdminAuth(w http.ResponseWriter, r *http.Request) bool {
	// 1. Try Signature Authentication (Modern/Secure)
	wallet := r.Header.Get("X-Admin-Wallet")
	nonce := r.Header.Get("X-Admin-Nonce")
	signature := r.Header.Get("X-Admin-Signature")

	if wallet != "" && nonce != "" && signature != "" {
		if l.verifyAdminSignature(wallet, nonce, signature) {
			return true
		}
		l.logAdminAudit("AUTH_FAILURE", wallet, "Invalid signature provided for nonce: "+nonce)
		log.Printf("[SECURITY ALERT] Invalid Admin Signature Attempt from Wallet: %s", wallet)
	} else {
		log.Printf("[SECURITY ALERT] Unauthorized Admin Access Attempt (Missing Headers) from IP: %s", r.RemoteAddr)
	}

	http.Error(w, "Unauthorized", http.StatusUnauthorized)
	return false
}

// verifyAdminSignature confirms the wallet is an admin and the signature for the nonce is valid.
func (l *Lobby) verifyAdminSignature(wallet, nonce, signatureStr string) bool {
	if !l.isAdminWallet(wallet) {
		return false
	}

	// 1. Verify that the nonce exists and is active globally
	l.mutex.RLock()
	found := false
	for _, nd := range l.nonces {
		if nd.Value == nonce {
			// Check expiration (5 minutes)
			if time.Since(nd.CreatedAt) < 5*time.Minute {
				found = true
			}
			break
		}
	}
	l.mutex.RUnlock()

	if !found {
		return false
	}

	// 2. Multi-Chain Verification Logic
	if strings.HasPrefix(wallet, "0x") {
		// EVM signature verification (personal_sign)
		message := fmt.Sprintf("\x19Ethereum Signed Message:\n%dVirtualbabes Arena Admin Auth:%s", 
			len("Virtualbabes Arena Admin Auth:")+len(nonce), nonce)
		messageHash := ethcrypto.Keccak256([]byte(message))

		signatureBytes, err := hex.DecodeString(strings.TrimPrefix(signatureStr, "0x"))
		if err != nil || len(signatureBytes) != 65 {
			return false
		}
		if signatureBytes[64] == 27 || signatureBytes[64] == 28 {
			signatureBytes[64] -= 27
		}

		pubKey, err := ethcrypto.SigToPub(messageHash, signatureBytes)
		if err != nil {
			return false
		}
		recoveredAddress := ethcrypto.PubkeyToAddress(*pubKey).Hex()
		return strings.EqualFold(recoveredAddress, wallet)
	} else {
		// AVM signature verification (ARC-14)
		addr, err := types.DecodeAddress(wallet)
		if err != nil {
			return false
		}

		sigBytes, err := base64.StdEncoding.DecodeString(signatureStr)
		if err != nil {
			return false
		}

		// Hardened message string to prevent signature confusion/replay attacks
		msg := fmt.Sprintf("Algorand Signed Message:\nVirtualbabes Arena Admin Auth:%s", nonce)
		return crypto.VerifyBytes(addr[:], []byte(msg), sigBytes)
	}
}

// isAdminWallet checks if a given wallet address is present in the ADMIN_WALLETS env variable.
func (l *Lobby) isAdminWallet(wallet string) bool {
	if wallet == "" {
		return false
	}
	admins := os.Getenv("ADMIN_WALLETS")
	if admins == "" {
		return false
	}
	for _, addr := range strings.Split(admins, ",") {
		if strings.TrimSpace(addr) == wallet {
			return true
		}
	}
	return false
}
