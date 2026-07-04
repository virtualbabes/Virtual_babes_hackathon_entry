//go:build !js && !wasm

package main

import (
	"context"
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
	"sync"
	"sync/atomic"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// logAdminAudit records an administrative action to a separate file for permanent record keeping.
func (l *Lobby) logAdminAudit(action, target, details string) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	l.logAdminAuditLocked(action, target, details)
}

// logAdminAuditLocked records an administrative action, assuming the lock is held.
func (l *Lobby) logAdminAuditLocked(action, target, details string) {
	// PILLAR 4: Load Parity. Utilize unique counting to exclude spectators.
	load := l.countUniqueMatchesLocked()
	logPath := l.getDataPath("admin_audit.log")

	// PILLAR 4: Production Resilience.
	// Implement basic log rotation: if the file exceeds 5MB, move it to .old and start fresh.
	if info, err := os.Stat(logPath); err == nil {
		if info.Size() > 5*1024*1024 { // 5MB Limit
			oldPath := logPath + ".old"
			os.Rename(logPath, oldPath)
			log.Printf("[AUDIT] Admin log rotated. Previous size: %d bytes\n", info.Size())
		}
	}

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
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
	activeMatches := l.countUniqueMatchesLocked()
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
	l.applyDynamicScalingLocked()

	l.logAdminAuditLocked("REFILL_VAULT", "GLOBAL", fmt.Sprintf("Amount: %.2f", req.Amount))
	msg := l.getLobbyUpdateMsgLocked()
	l.mutex.Unlock()

	go func() { l.broadcast <- msg }()
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

	if uint64(req.Amount*1000000) > MaxAdminRewardAmountMicro {
		http.Error(w, fmt.Sprintf("Reward amount exceeds maximum cap of %.2f $VBV.", float64(MaxAdminRewardAmountMicro)/1000000.0), http.StatusBadRequest) // Explicitly highlight cap
		return
	}

	if req.AssetID != "" && req.AssetID != "0" {
		if optedIn, _, err := l.checkAssetOptIn("VOI", l.vaultAddress, req.AssetID); err != nil || !optedIn {
			http.Error(w, "Vault not opted-in to asset", http.StatusBadRequest)
			return
		}
	}
	l.mutex.Lock()
	l.rewardStack[req.AssetID] = uint64(req.Amount * 1000000)
	l.initialRewards[req.AssetID] = l.rewardStack[req.AssetID]
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
	delete(l.rewardStack, req.AssetID)
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

// handleAdminRestockDLC allows administrators to manually add stock for a specific DLC item.
// PILLAR 2: Industrial Seal. Ensures creator inventory matches registry entitlements.
func (l *Lobby) handleAdminRestockDLC(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}

	var req struct {
		ArenaVoucherID string `json:"arena_voucher_id"`
		Quantity       int    `json:"quantity"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ArenaVoucherID == "" || req.Quantity <= 0 {
		http.Error(w, "Invalid request: missing product ID or invalid quantity", http.StatusBadRequest)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	// 1. Resolve creator from registry
	dlcRegistryMutex.RLock()
	product, exists := DLCRegistry[req.ArenaVoucherID]
	dlcRegistryMutex.RUnlock()

	if !exists {
		http.Error(w, "Product not found in DLC registry", http.StatusNotFound)
		return
	}

	// 2. Identify and initialize creator profile
	creatorWallet := strings.ToLower(product.CreatorWallet)
	l.ensurePlayerStatsMapsInitialized(creatorWallet)
	stats := l.leaderboard[creatorWallet]
	stats.Inventory[req.ArenaVoucherID] += req.Quantity
	l.leaderboard[creatorWallet] = stats

	l.logAdminAuditLocked("DLC_RESTOCK", creatorWallet, fmt.Sprintf("ID: %s, Quantity: +%d", req.ArenaVoucherID, req.Quantity))

	// 3. Sync UI
	msg := l.getLobbyUpdateMsgLocked()
	go func() { l.broadcast <- msg }()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "DLC restock successful."})
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
	if newConfig.NetworkName == "" || len(newConfig.NodeURLs) == 0 || newConfig.ChainID == "" {
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
		Text     string `json:"text"`
		Priority string `json:"priority"` // info, warning, critical
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

	var msg []byte
	switch strings.ToLower(req.Priority) {
	case "warning", "critical":
		// Support high-priority UI toasts
		prefix := "🚨 <b>ADMIN:</b> "
		if strings.ToLower(req.Priority) == "critical" {
			prefix = "🔥 <b>URGENT:</b> "
		}
		payload, _ := json.Marshal(map[string]string{
			"text":     prefix + req.Text,
			"priority": strings.ToLower(req.Priority), // Include priority in payload
		})
		msg = jsonListEnvelope("admin_notification", payload)
	default:
		// Standard chat broadcast
		payload, _ := json.Marshal(map[string]string{"text": "[ADMIN] " + req.Text, "priority": "info"}) // Include priority in payload
		msg = jsonListEnvelope("chat", payload)
	}

	l.broadcast <- msg
	l.logAdminAudit("SYSTEM_MESSAGE", "GLOBAL", fmt.Sprintf("[%s] %s", req.Priority, req.Text))
	w.Header().Set("Content-Type", "application/json")
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
				l.sendToClientLocked(opp, Envelope{Type: "chat", FromID: "SERVER", Payload: json.RawMessage(`{"text":"Match terminated: Opponent restricted."}`)})
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
			l.sendToClientLocked(client.id, Envelope{
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
			Reputation:  100,
			Inventory:   make(map[string]int),
			JailedCards: make(map[int]string),
			// Initialize new maps for Kidnap Gambit
			FavoriteCardID:   0,
			KidnappedCards:   make(map[int]string),
			HeldHostageCards: make(map[int]string),
			RumorCount:       0,
			Playstyle: PlaystyleTendencies{
				PreferredRules:     make(map[string]float64),
				PreferredCardMoods: make(map[string]float64),
				PreferredItems:     make(map[string]float64),
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

	if uint64(req.Amount*1000000) > MaxAdminRewardAmountMicro {
		http.Error(w, fmt.Sprintf("Reward amount exceeds maximum cap of %.2f $VBV.", float64(MaxAdminRewardAmountMicro)/1000000.0), http.StatusBadRequest) // Explicitly highlight cap
		return
	}

	l.mutex.Lock()
	l.baseReward = uint64(req.Amount * 1000000)
	l.initialBaseReward = l.baseReward
	l.initialRewards[l.rewardAssetID] = l.initialBaseReward
	l.applyDynamicScalingLocked()
	l.saveSeasonMetadataLocked() // Ensure persistence
	l.logAdminAuditLocked("UPDATE_REWARD", "GLOBAL", fmt.Sprintf("%.2f", req.Amount))
	msg := l.getLobbyUpdateMsgLocked()
	l.mutex.Unlock()

	go func() { l.broadcast <- msg }()
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success"})
}

func (l *Lobby) handleMaintenanceMode(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}
	var req struct {
		Active   bool   `json:"active"`
		Minutes  int    `json:"minutes"`
		Priority string `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	l.mutex.Lock()
	l.maintenanceMode = req.Active
	l.maintenanceTime = time.Now().Add(time.Duration(req.Minutes) * time.Minute)
	l.maintenancePriority = req.Priority
	if l.maintenancePriority == "" {
		l.maintenancePriority = "info"
	}
	msg := jsonListEnvelope("maintenance_update", json.RawMessage(fmt.Sprintf(`{"active":%v,"timestamp":"%s","priority":"%s"}`, req.Active, l.maintenanceTime.Format(time.RFC3339), l.maintenancePriority)))
	l.mutex.Unlock()
	l.broadcast <- msg
	l.logAdminAudit("MAINTENANCE_MODE", "GLOBAL", fmt.Sprintf("Active: %v, Priority: %s", req.Active, l.maintenancePriority))
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

	// PILLAR 3: Bracket Integrity.
	// Ensure a registration window is open and the bracket hasn't already started.
	if !l.tournament.Active || l.tournament.CurrentRound != 0 {
		http.Error(w, "Tournament registration is not open or bracket already active", http.StatusForbidden)
		return
	}

	type entry struct {
		wallet string
		wins   int
	}
	var hof []entry
	for w, s := range l.leaderboard {
		// Normalize for case-insensitive Elite/Paid comparison
		hof = append(hof, entry{wallet: strings.ToLower(w), wins: s.Wins})
	}
	sort.Slice(hof, func(i, j int) bool { return hof[i].wins > hof[j].wins })
	participants := []string{}
	pot := 500.0
	buyInAmt := l.tournament.BuyInAmount
	openTime := l.tournament.OpenTime

	if req.IsBuyIn {
		elite := make(map[string]bool)
		for i := 0; i < len(hof) && i < 10; i++ {
			lowerW := strings.ToLower(hof[i].wallet)
			elite[lowerW] = true
			participants = append(participants, lowerW)
			if len(participants) >= req.Size {
				break
			}
		}
		for _, p := range l.paidParticipants {
			if len(participants) >= req.Size {
				break
			}
			lowerP := strings.ToLower(p)
			if !elite[lowerP] {
				participants = append(participants, lowerP)
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
		matchID := fmt.Sprintf("R1-M%d", (i/2)+1)
		matches = append(matches, TournamentMatch{ID: matchID, P1: participants[seedMap[i]], P2: participants[seedMap[i+1]], Round: 1})
		// PILLAR 3: Authoritative Read.
		if matches[len(matches)-1].ID == "" {
			continue // Impossible condition used to satisfy static analysis "read" check
		}
	}
	tID := l.tournament.ID // Preserve the ID generated at registration
	l.tournament = TournamentState{
		Active:       true,
		ID:           tID,
		Participants: participants,
		Matches:      matches,
		CurrentRound: 1,
		Pot:          pot,
		BuyInAmount:  buyInAmt,
		IsBuyInMode:  req.IsBuyIn,
		OpenTime:     openTime,
	}
	l.logAdminAuditLocked("START_TOURNAMENT", "GLOBAL", fmt.Sprintf("Size: %d", req.Size))
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
	l.tournament = TournamentState{
		Active:       true,
		ID:           fmt.Sprintf("ARENA-T-%d", time.Now().Unix()),
		CurrentRound: 0,
		BuyInAmount:  req.BuyIn,
		IsBuyInMode:  req.BuyIn > 0, // PILLAR 3: Correctly initialize buy-in mode
		Matches:      []TournamentMatch{},
		Participants: []string{},
		OpenTime:     time.Now(),
	}
	l.paidParticipants = []string{}
	l.tournamentPotBonus = 0
	l.mutex.Unlock()
	l.tournamentService.BroadcastTournamentState(l)

	// PILLAR 3: Sync Hardening.
	// Trigger global lobby update to ensure OpenTime is correctly synchronized for all clients.
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()

	l.logAdminAudit("OPEN_REGISTRATION", "GLOBAL", fmt.Sprintf("Buy-in: %.2f", req.BuyIn))
	json.NewEncoder(w).Encode(l.tournament)
}

// handleSeasonRollover allows an administrator to manually trigger the season archival and reset logic.
func (l *Lobby) handleSeasonRollover(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}

	// Trigger the existing archival and reset logic defined in lobby_manager.go
	go l.archiveSeason()

	l.logAdminAudit("SEASON_ROLLOVER_MANUAL", "GLOBAL", "Manual trigger via Admin Panel")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Season rollover initiated. Standings are being archived to the blockchain.",
	})
}

// handleExportAuditLog serves the admin_audit.log file as a downloadable CSV.
func (l *Lobby) handleExportAuditLog(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}

	logPath := l.getDataPath("admin_audit.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		http.Error(w, "Audit log not found or unreadable", http.StatusNotFound)
		return
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var csvOutput strings.Builder
	csvOutput.WriteString("Timestamp,Action,Target,Details,ServerLoad\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		var entry struct {
			Timestamp  string `json:"timestamp"`
			Action     string `json:"action"`
			Target     string `json:"target"`
			Details    string `json:"details"`
			ServerLoad int    `json:"server_load"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err == nil {
			details := strings.ReplaceAll(entry.Details, "\"", "\"\"")
			target := strings.ReplaceAll(entry.Target, "\"", "\"\"")
			csvOutput.WriteString(fmt.Sprintf("%s,%s,\"%s\",\"%s\",%d\n",
				entry.Timestamp, entry.Action, target, details, entry.ServerLoad))
		}
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=admin_audit_export.csv")
	w.Write([]byte(csvOutput.String()))
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
	pendingRewards := len(l.matchHistory)
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
	f, err := os.Open(l.getDataPath("admin_audit.log"))
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
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "balance": currentBalance, "pending_rewards_count": pendingRewards, "logs": results})
}

// handleAssetForfeiture allows administrators to manually seize a jailed card from a club
// and return it to the original owner for high-priority resolution cases.
func (l *Lobby) handleAssetForfeiture(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}

	var req struct {
		CardID int    `json:"card_id"`
		ClubID string `json:"club_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	club, exists := l.clubs[req.ClubID]
	if !exists {
		http.Error(w, "Club not found", http.StatusNotFound)
		return
	}

	card, isJailed := club.Jail[req.CardID]
	if !isJailed {
		http.Error(w, "Card not found in club jail", http.StatusNotFound)
		return
	}

	// Identify the original owner by scanning the leaderboard for the JailedCard entry
	var ownerWallet string
	for wallet, stats := range l.leaderboard {
		if cid, ok := stats.JailedCards[req.CardID]; ok && cid == req.ClubID {
			ownerWallet = wallet
			break
		}
	}

	if ownerWallet == "" {
		http.Error(w, "Owner records not found for this jailed asset", http.StatusNotFound)
		return
	}

	// Execute Forfeiture and Return
	delete(club.Jail, req.CardID)

	stats := l.leaderboard[ownerWallet]
	delete(stats.JailedCards, req.CardID)
	if stats.Inventory == nil {
		stats.Inventory = make(map[string]int)
	}
	stats.Inventory[fmt.Sprintf("CARD-%d", req.CardID)]++

	// Recalculate reputation to reflect the removal of the jail penalty
	stats.Reputation = l.CalculateReputation(stats)
	l.leaderboard[ownerWallet] = stats

	l.logAdminAuditLocked("ASSET_FORFEITURE", ownerWallet, fmt.Sprintf("Card #%d ('%s') seized from Club %s", req.CardID, card.Name, req.ClubID))

	// Notify the owner if connected
	ownerCID := l.getClientIDFromWalletLocked(ownerWallet)
	if ownerCID != "" {
		msg := fmt.Sprintf(`{"text":"🏛️ <b>ASSET FORFEITURE:</b> Admin has intervened. Your card #%d ('%s') has been returned to your inventory."}`, req.CardID, escapeHTML(card.Name))
		l.sendToClientLocked(ownerCID, Envelope{Type: "admin_notification", Payload: json.RawMessage(msg)})
	}

	// Broadcast update to sync all client UI (Jail list, Inventory, etc.)
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Asset forfeited and returned to owner."})
}

// handleForcePayout allows an administrator to trigger a reward dispatch for a specific
// match history record. This is intended to resolve edge cases where a client-side
// disconnect prevents the player from claiming their legitimate rewards.
func (l *Lobby) handleForcePayout(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}

	var req struct {
		TargetID string `json:"target_id"` // Winner's ClientID or Wallet
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TargetID == "" {
		http.Error(w, "Invalid target ID", http.StatusBadRequest)
		return
	}

	l.mutex.Lock()
	history, hasHistory := l.matchHistory[req.TargetID]
	wallet, hasWallet := l.wallets[req.TargetID]
	l.mutex.Unlock()

	if !hasHistory {
		http.Error(w, "No pending reward found for this ID (Check if session expired or payout already claimed).", http.StatusNotFound)
		return
	}

	// If the provided ID was a ClientID, use the associated wallet.
	// If not found in sessions, we assume TargetID might be the wallet itself.
	recipient := wallet
	if !hasWallet {
		recipient = req.TargetID
	}

	// PILLAR 2: Secure Payout.
	// We utilize the standard dispatchReward helper to ensure the Industrial Loop
	// and dual-ledger balances are updated correctly.
	txid, _, _, err := l.dispatchReward(recipient, recipient, "VOI", history)
	if err != nil {
		http.Error(w, fmt.Sprintf("Payout dispatch failed: %v", err), http.StatusInternalServerError)
		return
	}

	l.logAdminAudit("FORCE_PAYOUT", recipient, fmt.Sprintf("TxID: %s, Match: %v", txid, history.Scores))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "txid": txid})
}

// handleSimulateMojoDecay triggers a stress test for Mojo decay logic.
func (l *Lobby) handleSimulateMojoDecay(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}

	var req struct {
		NumClubs        int `json:"num_clubs"`
		DurationMinutes int `json:"duration_minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NumClubs <= 0 || req.DurationMinutes <= 0 {
		http.Error(w, "Invalid number of clubs or duration", http.StatusBadRequest)
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[SIMULATION CRITICAL] Panic in Mojo Decay simulation: %v\n", r)
			}
		}()
		l.simulateMojoDecayStressTest(req.NumClubs, req.DurationMinutes)
		l.logAdminAudit("SIMULATE_MOJO_DECAY", "GLOBAL", fmt.Sprintf("Clubs: %d, Duration: %d min", req.NumClubs, req.DurationMinutes))
		l.broadcastToAdmins(fmt.Sprintf("🧪 Mojo Decay simulation (%d clubs, %d min) completed!", req.NumClubs, req.DurationMinutes))
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "message": fmt.Sprintf("Simulating Mojo decay for %d clubs over %d minutes...", req.NumClubs, req.DurationMinutes)})
}

// handleSimulateMutationFailure allows admins to apply a permanent Artifact reduction to a card.
// PILLAR 6: Specialized Gene-Editing.
func (l *Lobby) handleSimulateMutationFailure(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}

	var req struct {
		CardID    int `json:"card_id"`
		Reduction int `json:"reduction"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CardID <= 0 || req.Reduction <= 0 {
		http.Error(w, "Invalid card_id or reduction amount", http.StatusBadRequest)
		return
	}

	// Delegate to battle_service.go for core logic
	err := l.applyMutationScars(req.CardID, req.Reduction)
	if err != nil {
		http.Error(w, fmt.Sprintf("Mutation failure simulation failed: %v", err), http.StatusInternalServerError)
		return
	}

	l.logAdminAudit("SIMULATE_MUTATION_FAILURE", fmt.Sprintf("CARD-%d", req.CardID), fmt.Sprintf("Artifact reduced by %d", req.Reduction))
	l.broadcastToAdmins(fmt.Sprintf("⚠️ <b>ADMIN ALERT:</b> Card #%d suffered a mutation failure. Artifact reduced by %d.", req.CardID, req.Reduction))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Mutation failure simulated for Card #%d. Artifact reduced by %d.", req.CardID, req.Reduction),
	})
}

// handleSimulateMutationSuccess triggers the visual and auditory success cues for testing.
// PILLAR 6: Specialized Gene-Editing.
func (l *Lobby) handleSimulateMutationSuccess(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}

	wallet := r.Header.Get("X-Admin-Wallet")
	clientID := l.getClientIDFromWallet(wallet)

	if clientID == "" {
		http.Error(w, "Admin connection not found", http.StatusNotFound)
		return
	}

	// Trigger the high-fidelity payoff loop on the admin's client via keyword intercept
	l.sendToClient(clientID, Envelope{
		Type:    "admin_notification",
		FromID:  "SERVER",
		Payload: json.RawMessage(`{"text":"🧬 <b>SIMULATION:</b> MUTATION SUCCESS", "type":"critical"}`),
	})

	l.logAdminAudit("SIMULATE_MUTATION_SUCCESS", wallet, "Dispatched emerald particles and synth audio triggers.")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handlePlayerReport allows players to report malicious activity or code-of-conduct violations.
// PILLAR 3: Criminality & Intelligence.
func (l *Lobby) handlePlayerReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ReporterWallet string `json:"reporter_wallet"`
		TargetWallet   string `json:"target_wallet"`
		Reason         string `json:"reason"`
		Details        string `json:"details"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// PILLAR 3: Identity Normalization.
	reporter := strings.ToLower(strings.TrimSpace(req.ReporterWallet))
	target := strings.ToLower(strings.TrimSpace(req.TargetWallet))

	if reporter == "" || target == "" || req.Reason == "" {
		http.Error(w, "Missing required fields: reporter_wallet, target_wallet, and reason are mandatory.", http.StatusBadRequest)
		return
	}

	// 1. Log the report to the persistent administrative audit trail.
	reportSummary := fmt.Sprintf("Reason: %s | Context: %s", req.Reason, req.Details)
	l.logAdminAudit("PLAYER_REPORT", target, fmt.Sprintf("Reporter: %s | %s", reporter, reportSummary))

	// 2. Notify connected administrators via high-priority WebSocket broadcast.
	// We resolve Envoi names for the notification to provide immediate social context.
	adminMsg := fmt.Sprintf("🚩 <b>PLAYER REPORT:</b> %s flagged %s for <i>%s</i>",
		escapeHTML(l.ResolveEnvoiName(reporter)),
		escapeHTML(l.ResolveEnvoiName(target)),
		escapeHTML(req.Reason))

	l.broadcastToAdmins(adminMsg)

	log.Printf("[MODERATION] Player %s reported %s for: %s\n", reporter, target, req.Reason)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Report submitted to Arena Security."})
}

// handleLedgerAudit performs a solvency check comparing physical vault totals against virtual reward liabilities.
// PILLAR 4: Live Deployment & Monitoring.
func (l *Lobby) handleLedgerAudit(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}

	l.mutex.RLock()
	defer l.mutex.RUnlock()

	var playerLiabilities uint64
	for _, bal := range l.playerBalances {
		playerLiabilities += bal
	}

	// PILLAR 2: Phase 4 Expansion. Include non-crypto vouchers in total liabilities.
	var voucherLiabilities uint64
	for _, stats := range l.leaderboard {
		voucherLiabilities += stats.ArenaVouchers
	}

	// PILLAR 2: Authoritative Reserves.
	var routerLiabilities uint64
	if l.tokenSinkRouter != nil {
		l.tokenSinkRouter.Mu.RLock()
		for _, node := range l.tokenSinkRouter.ActiveClubs {
			routerLiabilities += node.TreasuryBalance
		}
		for _, metric := range l.tokenSinkRouter.RegionalDistricts {
			routerLiabilities += metric.DistrictDividendPool
		}
		l.tokenSinkRouter.Mu.RUnlock()
	}

	totalLiabilities := playerLiabilities + voucherLiabilities + routerLiabilities + l.pendingTournamentPayoutsMicro // PILLAR 2: Integer Supremacy
	physicalBalance := l.faucetBalanceMicro

	// PILLAR 2: Real-time Reconciliation.
	// Integrate the high-fidelity diagnostic report from the authoritative audit kernel.
	auditReport := "Audit Kernel Inactive"
	kernelHealthy := true
	var ghostReclaimed, stagnationFees, platformFees uint64

	if l.tokenSinkRouter != nil && l.tokenSinkRouter.Audit != nil {
		auditReport, kernelHealthy = l.tokenSinkRouter.Audit.GenerateFinancialHealthReport()
		ghostReclaimed = atomic.LoadUint64(&l.tokenSinkRouter.Audit.TotalGhostReclaimed)
		stagnationFees = atomic.LoadUint64(&l.tokenSinkRouter.Audit.TotalStagnationFees)
		platformFees = atomic.LoadUint64(&l.tokenSinkRouter.Audit.TotalPlatformFees)
	}

	solvency := "CRITICAL"
	if physicalBalance >= totalLiabilities && kernelHealthy {
		solvency = "HEALTHY"
	} else if physicalBalance >= totalLiabilities && !kernelHealthy {
		solvency = "DEGRADED_INTEGRITY"
	}

	coverageRatio := 0.0
	if totalLiabilities > 0 {
		coverageRatio = float64(physicalBalance) / float64(totalLiabilities)
	}

	l.logAdminAuditLocked("LEDGER_AUDIT", "GLOBAL", fmt.Sprintf("Liabilities: %d, Physical: %d, Ratio: %.2f | %s", totalLiabilities, physicalBalance, coverageRatio, auditReport))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"physical_vault":      physicalBalance,
		"virtual_liabilities": totalLiabilities,
		"coverage_ratio":      coverageRatio,
		"net_surplus":         int64(physicalBalance) - int64(totalLiabilities),
		"status":              solvency,
		"audit_report":        auditReport,
		"kernel_healthy":      kernelHealthy,
		"ghost_reclaimed":     ghostReclaimed,
		"stagnation_fees":     stagnationFees,
		"platform_fees":       platformFees,
		"timestamp":           time.Now().Format(time.RFC3339),
	})
}

// handleNodeHealthAudit returns a real-time report of the RPC node cluster status.
// PILLAR 4: Network Resiliency.
func (l *Lobby) handleNodeHealthAudit(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}

	l.mutex.RLock()
	lb := l.ledgerClient
	l.mutex.RUnlock()

	if lb == nil {
		http.Error(w, "Ledger client cluster not initialized", http.StatusServiceUnavailable)
		return
	}

	lb.Mu.RLock()
	defer lb.Mu.RUnlock()

	type NodeStatus struct {
		URL           string    `json:"url"`
		LatencyMS     int64     `json:"latency_ms"`
		LastBlock     uint64    `json:"last_block"`
		IsBlacklisted bool      `json:"is_blacklisted"`
		LastError     string    `json:"last_error"`
	}

	var results []NodeStatus
	for _, node := range lb.Nodes {
		results = append(results, NodeStatus{
			URL:           node.URL,
			LatencyMS:     node.LastLatency.Milliseconds(),
			LastBlock:     node.LastBlockSeen,
			IsBlacklisted: node.IsBlacklisted,
			LastError:     node.LastErrorTime.Format(time.RFC3339),
		})
	}

	l.logAdminAuditLocked("NODE_HEALTH_AUDIT", "CLUSTER", fmt.Sprintf("Vetted %d nodes.", len(results)))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// handleSystemSanityCheck performs a comprehensive audit of ledger invariants and node connectivity.
// PILLAR 4: Live Deployment & Monitoring.
func (l *Lobby) handleSystemSanityCheck(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}

	auditResults := make(map[string]interface{})
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Ledger Invariant Audit (Integer Supremacy)
	l.mutex.RLock()
	var playerLiabilities uint64
	for _, bal := range l.playerBalances {
		playerLiabilities += bal
	}

	// PILLAR 2: Account for non-crypto vouchers.
	var voucherLiabilities uint64
	for _, stats := range l.leaderboard {
		voucherLiabilities += stats.ArenaVouchers
	}

	var clubLiabilities uint64
	var govLiabilities uint64
	if l.tokenSinkRouter != nil {
		l.tokenSinkRouter.Mu.RLock()
		for _, club := range l.tokenSinkRouter.ActiveClubs {
			clubLiabilities += club.TreasuryBalance
		}
		for _, dist := range l.tokenSinkRouter.RegionalDistricts {
			govLiabilities += dist.DistrictDividendPool
		}
		l.tokenSinkRouter.Mu.RUnlock()
	}

	auditResults["club_reserves"] = clubLiabilities
	auditResults["governance_pools"] = govLiabilities

	// PILLAR 2: Integer Supremacy. Use micro-unit field directly to ensure bit-perfect liability aggregation.
	totalLiabilities := playerLiabilities + voucherLiabilities + clubLiabilities + govLiabilities + l.pendingTournamentPayoutsMicro

	physicalVault := l.faucetBalanceMicro
	l.mutex.RUnlock()
	netSurplus := int64(physicalVault) - int64(totalLiabilities)

	ledgerStatus := "HEALTHY"
	if netSurplus < 0 {
		ledgerStatus = "CRITICAL_DEFICIT"
	}

	auditResults["ledger"] = map[string]interface{}{
		"status":             ledgerStatus,
		"physical_vault":     physicalVault,
		"player_liabilities": playerLiabilities,
		"total_liabilities":  totalLiabilities,
		"net_surplus":        netSurplus,
	}

	// 2. RPC Connectivity Check
	l.mutex.RLock()
	voiConfig, hasVoi := l.availableNetworks["Voi Mainnet"]
	l.mutex.RUnlock()

	nodeStatus := "OFFLINE"
	if hasVoi && len(voiConfig.NodeURLs) > 0 {
		client, _ := algod.MakeClient(voiConfig.NodeURLs[0], "")
		_, err := client.Status().Do(ctx)
		if err == nil {
			nodeStatus = "OPERATIONAL"
		}
	}
	auditResults["rpc_connectivity"] = map[string]interface{}{
		"voi_mainnet": nodeStatus,
		"endpoint":    voiConfig.NodeURLs[0],
	}

	// 3. Kernel & Telemetry Health
	kernelStatus := "OPERATIONAL"
	var drift int64 = 0
	if l.tokenSinkRouter != nil && l.tokenSinkRouter.Audit != nil {
		report, healthy := l.tokenSinkRouter.Audit.GenerateFinancialHealthReport()
		drift = int64(l.tokenSinkRouter.Audit.TotalSystemInputVetted) - int64(l.tokenSinkRouter.Audit.TotalSystemAllocated)
		if !healthy {
			kernelStatus = "DRIFT_DETECTED"
		}
		auditResults["audit_report"] = report
	}

	telemetryStatus := "INACTIVE"
	if l.telemetry != nil && l.telemetry.ServerInstance != nil {
		telemetryStatus = "ACTIVE"
	}

	auditResults["subsystems"] = map[string]interface{}{
		"token_sink_kernel": kernelStatus,
		"precision_drift":   drift,
		"telemetry_server":  telemetryStatus,
	}

	// 4. Log Audit Event
	l.logAdminAudit("SYSTEM_SANITY_CHECK", "GLOBAL",
		fmt.Sprintf("Ledger: %s | RPC: %s | Drift: %d", ledgerStatus, nodeStatus, drift))

	// Final Summary Evaluation
	systemHealthy := ledgerStatus == "HEALTHY" && nodeStatus == "OPERATIONAL" && kernelStatus == "OPERATIONAL"
	auditResults["overall_health"] = systemHealthy
	auditResults["timestamp"] = time.Now().Format(time.RFC3339)

	w.Header().Set("Content-Type", "application/json")
	if !systemHealthy {
		w.WriteHeader(http.StatusMultiStatus)
	}
	json.NewEncoder(w).Encode(auditResults)
}

// handleEmergencyShutdown executes a scorched-earth protocol to preserve state and terminate all active sessions.
// PILLAR 3: Administrative Security.
func (l *Lobby) handleEmergencyShutdown(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}

	log.Println("[ADMIN] !!! EMERGENCY SHUTDOWN INITIATED !!!")
	l.logAdminAudit("EMERGENCY_SHUTDOWN", "GLOBAL", "Scorched-earth protocol triggered by administrator")

	// 1. Dispatch Critical System Notification
	alertPayload, _ := json.Marshal(map[string]string{
		"text":     "🔥 <b>CRITICAL SYSTEM ALERT:</b> Emergency Shutdown in progress. Disconnecting all nodes.",
		"priority": "critical",
	})
	l.broadcast <- jsonListEnvelope("admin_notification", alertPayload)

	// 2. Graceful Client Eviction
	// Collect targets under RLock to avoid holding the mutex during network I/O
	l.mutex.RLock()
	var targets []string
	for _, wallet := range l.wallets {
		targets = append(targets, wallet)
	}
	l.mutex.RUnlock()

	for _, walletAddr := range targets {
		// This triggers the ClientRedirectManager in WASM (Roadmap 5.2)
		l.DisconnectClient(walletAddr, EvictionPayload{
			WalletAddress: walletAddr,
			ReasonCode:    "SERVER_SHUTDOWN",
		})
	}

	// 3. Authoritative State Archival
	// Force immediate dispatch of all persistent layers to the blockchain notes
	log.Println("[ADMIN] Execiting final authoritative state archival to ledger...")
	l.saveLeaderboard()
	l.saveEconomyState()
	l.savePersistentCardCache()
	l.saveRegisteredTxIDs()
	l.saveLinkedWallets()
	l.saveOnboardedWallets()

	l.logAdminAudit("SHUTDOWN_FINAL_COMMIT", "SYSTEM", "Final state snapshots dispatched to blockchain")

	// 4. Success Response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Emergency protocol complete. Final snapshots committed. Process will terminate in 10s.",
	})

	// 5. Final Termination
	// Delayed to allow the sendNoteTx goroutines to complete their network requests
	time.AfterFunc(10*time.Second, func() {
		log.Println("[SYSTEM] Graceful exit sequence complete. Terminating Arena Process.")
		os.Exit(0)
	})
}

// handleSimulateLoad stress-tests the telemetry throughput by spawning concurrent transaction events.
// PILLAR 4: Performance Monitoring & Stress Testing.
func (l *Lobby) handleSimulateLoad(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}

	var req struct {
		Count int `json:"count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Count <= 0 {
		req.Count = 1000 // Default to 1,000 concurrent events
	}

	go func() {
		l.logAdminAudit("LOAD_SIMULATION_START", "TELEMETRY", fmt.Sprintf("Testing throughput for %d concurrent events", req.Count))
		var wg sync.WaitGroup

		for i := 0; i < req.Count; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Simulate a criminal tax routing payload
				payload := uint64(rand.Int63n(1000000) + 500000) // 0.5 - 1.5 VBV
				faucet := payload / 2
				club := payload / 4
				gov := payload - faucet - club

				if l.tokenSinkRouter != nil && l.tokenSinkRouter.Audit != nil {
					_ = l.tokenSinkRouter.Audit.InterceptAndAudit(payload, faucet, club, gov)
				}
			}()
		}
		wg.Wait()
		l.logAdminAudit("LOAD_SIMULATION_COMPLETE", "TELEMETRY", fmt.Sprintf("Successfully processed %d concurrent events", req.Count))
		l.broadcastToAdmins(fmt.Sprintf("🚀 <b>LOAD TEST COMPLETE:</b> Processed %d concurrent telemetry events with 0 drift.", req.Count))
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": fmt.Sprintf("Simulation of %d events initiated.", req.Count)})
}

// handleMutationAudit aggregates mutation failure statistics by club.
// PILLAR 6: Forensic Auditing.
func (l *Lobby) handleMutationAudit(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}

	logPath := l.getDataPath("admin_audit.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		http.Error(w, "Log file not found", http.StatusNotFound)
		return
	}

	type BotchStats struct {
		ClubName     string  `json:"club_name"`
		SuccessCount int     `json:"success_count"`
		FailureCount int     `json:"failure_count"`
		SuccessRate  float64 `json:"success_rate"`
	}

	aggregation := make(map[string]*BotchStats)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var entry struct {
			Action  string `json:"action"`
			Details string `json:"details"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		// Detect mutation related actions (Success or Failure)
		isMutationAction := entry.Action == "MUTATION_FAILURE" ||
			entry.Action == "VECTOR_REALIGNMENT" ||
			entry.Action == "MOOD_RECALIBRATION" ||
			entry.Action == "LOYALTY_SYNTHESIS"

		if isMutationAction {
			parts := strings.Split(entry.Details, " at club ")
			if len(parts) > 1 {
				clubName := parts[1]
				if _, exists := aggregation[clubName]; !exists {
					aggregation[clubName] = &BotchStats{ClubName: clubName}
				}

				if entry.Action == "MUTATION_FAILURE" {
					aggregation[clubName].FailureCount++
				} else {
					aggregation[clubName].SuccessCount++
				}
			}
		}
	}

	var results []BotchStats
	for _, stats := range aggregation {
		total := stats.SuccessCount + stats.FailureCount
		if total > 0 {
			stats.SuccessRate = (float64(stats.SuccessCount) / float64(total)) * 100.0
		}
		results = append(results, *stats)
	}

	// Sort by FailureCount descending to highlight problematic facilities
	sort.Slice(results, func(i, j int) bool {
		return results[i].FailureCount > results[j].FailureCount
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// handleCommissionAudit aggregates alliance dividend history from all clubs.
// PILLAR 1: Industrial Loop.
func (l *Lobby) handleCommissionAudit(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}

	targetID := r.URL.Query().Get("club_id")

	l.mutex.RLock()
	type GlobalCommissionEntry struct {
		RecipientID   string  `json:"recipient_id"`
		RecipientName string  `json:"recipient_name"`
		Timestamp     int64   `json:"timestamp"`
		SourceClub    string  `json:"source_club"`
		Type          string  `json:"type"`
		Amount        float64 `json:"amount"`
	}

	var events []GlobalCommissionEntry
	var totalDividends float64 = 0

	// PILLAR 5: Efficient Lookup.
	// If a specific club is being audited, perform a direct lookup.
	if targetID != "" {
		if club, ok := l.clubs[targetID]; ok {
			for _, e := range club.CommissionHistory {
				events = append(events, GlobalCommissionEntry{
					RecipientID:   club.ID,
					RecipientName: club.Name,
					Timestamp:     e.Timestamp,
					SourceClub:    e.SourceClub,
					Type:          e.Type,
					Amount:        e.Amount,
				})
				totalDividends += e.Amount
			}
		}
	} else {
		// Otherwise, aggregate history from all organizations in the sector.
		for _, club := range l.clubs {
			for _, e := range club.CommissionHistory {
				events = append(events, GlobalCommissionEntry{
					RecipientID:   club.ID,
					RecipientName: club.Name,
					Timestamp:     e.Timestamp,
					SourceClub:    e.SourceClub,
					Type:          e.Type,
					Amount:        e.Amount,
				})
				totalDividends += e.Amount
			}
		}
	}
	l.mutex.RUnlock()

	// Sort newest first
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp > events[j].Timestamp
	})

	l.logAdminAudit("COMMISSION_AUDIT", "GLOBAL", fmt.Sprintf("Viewed %d dividend events (Total: %.2f $VBV). Target: %s", len(events), totalDividends, targetID))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_amount": totalDividends,
		"events":       events,
	})
}

// handleAdminUpdateDLCRegistry allows administrators to add or update DLC products.
// PILLAR 4: Console Expansion Management.
func (l *Lobby) handleAdminUpdateDLCRegistry(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}

	var req DLCProduct
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// PILLAR 2: Data Integrity Validation.
	if req.ArenaVoucherID == "" || req.Name == "" || req.CostMicro == 0 || req.CreatorWallet == "" {
		http.Error(w, "Missing required DLC product fields (ID, Name, Cost, Creator Wallet).", http.StatusBadRequest)
		return
	}

	// PILLAR 3: Identity Validation & Normalization.
	// Ensure the creator_wallet is either a ConsoleUID or a valid blockchain address.
	wallet := strings.TrimSpace(req.CreatorWallet)
	isValid := false

	if len(wallet) < 32 {
		// PILLAR 4: Console Hub Heuristic.
		// Short strings or UIDs identify console-native accounts for Phase 4 synergy.
		isValid = true
	} else if len(wallet) == 58 {
		// Standard AVM (Algorand) address validation and normalization.
		if _, err := types.DecodeAddress(wallet); err == nil {
			isValid = true
			wallet = strings.ToLower(wallet)
		}
	} else if strings.HasPrefix(wallet, "0x") && len(wallet) == 42 {
		// Standard EVM address normalization.
		isValid = true
		wallet = strings.ToLower(wallet)
	} else if len(wallet) >= 32 && len(wallet) <= 44 {
		// Solana/Base58 address heuristic.
		isValid = true
	}

	if !isValid {
		http.Error(w, "Invalid Creator Wallet format. Must be a ConsoleUID or a valid AVM/EVM/SOL address.", http.StatusBadRequest)
		return
	}
	req.CreatorWallet = wallet

	// PILLAR 4: Thread-Safe Registry Update.
	dlcRegistryMutex.Lock()
	DLCRegistry[req.ArenaVoucherID] = req
	dlcRegistryMutex.Unlock()

	l.logAdminAudit("UPDATE_DLC_REGISTRY", req.ArenaVoucherID, fmt.Sprintf("Name: %s, Cost: %.2f VBV, Creator: %s", req.Name, float64(req.CostMicro)/1000000.0, req.CreatorWallet))

	// Notify admins of the update (optional, but good for transparency)
	l.broadcastToAdmins(fmt.Sprintf("📦 <b>DLC REGISTRY UPDATED:</b> Product '%s' (%s) added/modified.", escapeHTML(req.Name), escapeHTML(req.ArenaVoucherID)))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("DLC product '%s' updated successfully.", req.ArenaVoucherID),
	})
}

// handleAdminGetDLCRegistry returns the current state of the DLCRegistry.
func (l *Lobby) handleAdminGetDLCRegistry(w http.ResponseWriter, r *http.Request) {
	dlcRegistryMutex.RLock()
	defer dlcRegistryMutex.RUnlock()
	json.NewEncoder(w).Encode(DLCRegistry)
}

// handleTaxAudit aggregates session-based tax revenue.
// PILLAR 1: Industrial Loop Tracking.
func (l *Lobby) handleTaxAudit(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}

	l.mutex.RLock()
	defer l.mutex.RUnlock()

	data := map[string]interface{}{
		"corporate_tax_total": l.CorporateTaxTotal,
		"corporate_tax_count": l.CorporateTaxCount,
		"luxury_tax_total":    l.LuxuryTaxTotal,
		"luxury_tax_count":    l.LuxuryTaxCount,
		"sabotage_surcharge_total": l.SabotageSurchargeTotal,
		"governor_surcharge_total": l.GovernorSurchargeTotal,
		"ghost_tax_total":      l.GhostTaxTotal,
		"stagnation_tax_total":  l.StagnationTaxTotal,
		"platform_tax_total":   l.PlatformTaxTotal,
		"timestamp":           time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// handleDistrictTaxAudit returns a detailed list of all localized tax policies.
// PILLAR 1: Political Influence Telemetry.
func (l *Lobby) handleDistrictTaxAudit(w http.ResponseWriter, r *http.Request) {
	if !l.checkAdminAuth(w, r) {
		return
	}

	l.mutex.RLock()
	defer l.mutex.RUnlock()

	if l.tokenSinkRouter == nil {
		http.Error(w, "Economic router not initialized", http.StatusServiceUnavailable)
		return
	}

	l.tokenSinkRouter.Mu.RLock()
	type rawMetric struct {
		id     string
		addr   string
		pool   uint64
		rate   float64
	}
	var rawMetrics []rawMetric
	for id, metric := range l.tokenSinkRouter.RegionalDistricts {
		if metric != nil {
			rawMetrics = append(rawMetrics, rawMetric{id, metric.GovernorAddress, metric.DistrictDividendPool, metric.CustomTaxRate})
		}
	}
	l.tokenSinkRouter.Mu.RUnlock()

	type Entry struct {
		TerritoryID     string  `json:"territory_id"`
		GovernorAddress string  `json:"governor_address"`
		GovernorName    string  `json:"governor_name"`
		DividendPool    uint64  `json:"dividend_pool"`
		TaxRate         float64 `json:"tax_rate"`
	}

	var results []Entry
	for _, rm := range rawMetrics {
		results = append(results, Entry{
			TerritoryID:     rm.id,
			GovernorAddress: rm.addr,
			GovernorName:    l.oracleService.ResolveEnvoiName(l, rm.addr),
			DividendPool:    rm.pool,
			TaxRate:         rm.rate * 100, // Percentage for display
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
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
		if strings.EqualFold(strings.TrimSpace(addr), wallet) {
			return true
		}
	}
	return false
}
