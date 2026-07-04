//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// ============================================================================
// PILLAR 12-13: HTTP HANDLERS for Career & Rivalry Systems
// ============================================================================

// rivalryRequestData is the payload for /api/rivalry/request POST
type rivalryRequestData struct {
	TargetWallet string `json:"target_wallet"`
}

// rivalryActionData is the payload for /api/rivalry/accept, /api/rivalry/decline, /api/rivalry/challenge POST
type rivalryActionData struct {
	RivalWallet string `json:"rival_wallet"`
	Status      string `json:"status,omitempty"` // "accepted", "declined"
}

// factionBuyData is the payload for /api/faction/shop/buy POST
type factionBuyData struct {
	Faction  string `json:"faction"`
	ItemName string `json:"item_name"`
}

// HandleRivalryRequest handles rival request WebSocket messages.
func (l *Lobby) HandleRivalryRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req rivalryRequestData
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	targetWallet := strings.TrimSpace(req.TargetWallet)
	if targetWallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "target_wallet is required"})
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Verify sender exists in leaderboard via wallet
	senderID := r.URL.Query().Get("wallet")
	if senderID == "" {
		senderID = r.Header.Get("X-Player-Wallet")
	}
	if senderID == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "unauthorized"})
		return
	}

	// Verify target exists in leaderboard
	targetStats, targetExists := l.leaderboard[targetWallet]
	if !targetExists {
		writeJSON(w, map[string]interface{}{"success": false, "error": "target player not found"})
		return
	}

	// Can't rival yourself
	if senderID == targetWallet {
		writeJSON(w, map[string]interface{}{"success": false, "error": "cannot rival yourself"})
		return
	}

	senderStats := l.leaderboard[senderID]

	// Check if already rivals
	if senderStats.Rivalry != nil && senderStats.Rivalry.ActiveRivals != nil {
		for _, r := range senderStats.Rivalry.ActiveRivals {
			if r == targetWallet {
				writeJSON(w, map[string]interface{}{"success": false, "error": "already rivals"})
				return
			}
		}
	}

	// Create pending invitation
	if senderStats.Rivalry == nil {
		senderStats.Rivalry = &RivalryState{
			ActiveRivals:     []string{},
			PendingInvitations: []PendingRivalInvite{},
		}
	}
	if senderStats.Rivalry.ActiveRivals == nil {
		senderStats.Rivalry.ActiveRivals = []string{}
	}
	if senderStats.Rivalry.PendingInvitations == nil {
		senderStats.Rivalry.PendingInvitations = []PendingRivalInvite{}
	}

	inviteID := fmt.Sprintf("INVITE-%s-%d", targetWallet, time.Now().UnixNano())
	level := senderStats.CareerXP.CalculateLessonLevel()
	if senderStats.CareerXP == nil {
		level = 1
	}

	senderStats.Rivalry.PendingInvitations = append(senderStats.Rivalry.PendingInvitations, PendingRivalInvite{
		FromWallet: senderID,
		ID:         inviteID,
		Timestamp:  time.Now(),
		Level:      level,
	})
	l.leaderboard[senderID] = senderStats

	// Broadcast invitation to target
	targetClientID := l.wallets[targetWallet]
	if targetClientID != "" {
		if targetClient := l.clients[targetClientID]; targetClient != nil {
			msg := Envelope{
				Type: "rivalry_invitation",
				FromID: "SERVER",
				ToID:   targetClient.id,
				Payload: json.RawMessage(fmt.Sprintf(
					`{"from_wallet":"%s","level":%d,"invite_id":"%s"}`,
					senderID[:8]+"...", level, inviteID),
				),
			}
			broadcastBytes, _ := json.Marshal(msg)
			targetClient.send <- broadcastBytes
		}
	}

	log.Printf("[RIVALRY] Invitation sent: %s -> %s (ID: %s)", senderID[:8]+"...", targetWallet, inviteID)
	writeJSON(w, map[string]interface{}{
		"success":       true,
		"invite_id":     inviteID,
		"target_wallet": targetWallet,
	})
}

// HandleRivalryAction handles rival accept/decline/challenge WebSocket messages.
func (l *Lobby) HandleRivalryAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	action := r.URL.Query().Get("action") // "accept" or "decline" or "challenge"
	if action == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "action parameter required (accept/decline/challenge)"})
		return
	}

	var req rivalryActionData
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	rivalWallet := strings.TrimSpace(req.RivalWallet)
	if rivalWallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "rival_wallet is required"})
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	clientID := r.URL.Query().Get("wallet")
	if clientID == "" {
		clientID = r.Header.Get("X-Player-Wallet")
	}
	if clientID == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "unauthorized"})
		return
	}

	stats, ok := l.leaderboard[clientID]
	if !ok {
		writeJSON(w, map[string]interface{}{"success": false, "error": "player not found"})
		return
	}

	if stats.Rivalry == nil {
		stats.Rivalry = &RivalryState{
			ActiveRivals:     []string{},
			PendingInvitations: []PendingRivalInvite{},
		}
	}
	if stats.Rivalry.ActiveRivals == nil {
		stats.Rivalry.ActiveRivals = []string{}
	}
	if stats.Rivalry.PendingInvitations == nil {
		stats.Rivalry.PendingInvitations = []PendingRivalInvite{}
	}

	switch action {
	case "accept":
		// Find and accept the invitation
		found := false
		for i, inv := range stats.Rivalry.PendingInvitations {
			if inv.FromWallet == rivalWallet && inv.ToWallet == clientID {
				// Make bilateral connection
				senderStats, senderOk := l.leaderboard[rivalWallet]
				if senderOk {
					if senderStats.Rivalry == nil {
						senderStats.Rivalry = &RivalryState{ActiveRivals: []string{}, PendingInvitations: []PendingRivalInvite{}}
					}
					if senderStats.Rivalry.ActiveRivals == nil {
						senderStats.Rivalry.ActiveRivals = []string{}
					}

					// Add to both sides
					senderStats.Rivalry.ActiveRivals = append(senderStats.Rivalry.ActiveRivals, clientID)
					stats.Rivalry.ActiveRivals = append(stats.Rivalry.ActiveRivals, rivalWallet)

					// Remove invitation from the pending list
					remaining := make([]PendingRivalInvite, 0)
					remaining = append(remaining, stats.Rivalry.PendingInvitations[:i]...)
					remaining = append(remaining, stats.Rivalry.PendingInvitations[i+1:]...)
					stats.Rivalry.PendingInvitations = remaining

					l.leaderboard[rivalWallet] = senderStats

					// Notify the other party
					rivalClientID := l.wallets[rivalWallet]
					if rivalClientID != "" {
						if rivalClient := l.clients[rivalClientID]; rivalClient != nil {
							notif := Envelope{
								Type: "rivalry_accepted",
								FromID: "SERVER",
								ToID:   rivalClient.id,
								Payload: json.RawMessage(fmt.Sprintf(
									`{"text":"<b>RIVALRY ACCEPTED</b><br/>%s now accepts your challenge!", "rival_wallet": "%s"}`, clientID[:8]+"...", rivalWallet[:8]+"..."),
								),
							}
							bcast, _ := json.Marshal(notif)
							rivalClient.send <- bcast
						}
					}

					found = true
					log.Printf("[RIVALRY] Rivalry established: %s <-> %s", clientID, rivalWallet)
				}
				break
			}
		}

		if !found {
			writeJSON(w, map[string]interface{}{"success": false, "error": "invitation not found"})
			return
		}

		l.leaderboard[clientID] = stats
		writeJSON(w, map[string]interface{}{
			"success":      true,
			"rival_wallet": rivalWallet,
		})

	case "decline":
		// Remove the invitation without accepting
		found := false
		newInvites := make([]PendingRivalInvite, 0)
		for _, inv := range stats.Rivalry.PendingInvitations {
			if inv.FromWallet == rivalWallet && inv.ToWallet == clientID {
				found = true
			} else {
				newInvites = append(newInvites, inv)
			}
		}
		stats.Rivalry.PendingInvitations = newInvites
		l.leaderboard[clientID] = stats

		if found {
			log.Printf("[RIVALRY] Rivalry invitation declined: %s -> %s", clientID, rivalWallet)
		}
		writeJSON(w, map[string]interface{}{"success": true})

	case "challenge":
		// Send PvP challenge to active rival
		challengeFound := false
		for _, r := range stats.Rivalry.ActiveRivals {
			if r == rivalWallet {
				challengeFound = true
				rivalClientID := l.wallets[rivalWallet]
				if rivalClientID != "" {
					if rivalClient := l.clients[rivalClientID]; rivalClient != nil {
						chalMsg := Envelope{
							Type: "rivalry_challenge",
							FromID: "SERVER",
							ToID:   rivalClient.id,
							Payload: json.RawMessage(fmt.Sprintf(
								`{"challenger":"%s","text":"<b>RIVAL CHALLENGE!</b><br/>%s challenges you to a match!"}`,
								clientID[:8]+"...", clientID[:8]+"..."),
							),
						}
						bcast, _ := json.Marshal(chalMsg)
						rivalClient.send <- bcast
					}
				}

				log.Printf("[RIVALRY] Challenge sent: %s -> %s", clientID[:8]+"...", rivalWallet)
			}
		}

		if !challengeFound {
			writeJSON(w, map[string]interface{}{"success": false, "error": "player is not an active rival"})
			return
		}

		writeJSON(w, map[string]interface{}{
			"success": true,
			"text":    fmt.Sprintf("Challenge sent to %s!", rivalWallet[:8]+"..."),
		})

	default:
		writeJSON(w, map[string]interface{}{"success": false, "error": fmt.Sprintf("unknown action: %s", action)})
	}
}

// HandleGetFactionShop handles GET /api/faction/shop/{faction}
func (l *Lobby) HandleGetFactionShop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract faction from path: /api/faction/shop/{faction}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	var faction string
	for i, part := range parts {
		if part == "shop" && i+1 < len(parts) {
			faction = strings.ToUpper(parts[i+1])
			break
		}
	}

	if faction != "JUSTICE" && faction != "UNDERWORLD" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "faction must be JUSTICE or UNDERWORLD"})
		return
	}

	l.mutex.RLock()
	defer l.mutex.RUnlock()

	// Build shop items for this faction
	var items []map[string]interface{}

	if faction == "JUSTICE" {
		justiceItems := map[string]JusticeItem{
			"truth_serum":      {ItemDef: ItemDef{ID: "ITEM-JUSTICE-001", Name: "Truth Serum", Description: "Temporarily reveals all active item buffs and debuffs on an opponent's cards.", CostMicro: 2500 * 1000000, Category: JusticeCategory, MaxStack: 3}, JusticeBonus: map[string]int{"reveals_opponent_buffs": 1}, DurationMin: 5},
			"reputation_shield": {ItemDef: ItemDef{ID: "ITEM-JUSTICE-002", Name: "Reputation Shield", Description: "Reduces Reputation penalties from failed pro-social actions by 75% for 1 hour.", CostMicro: 3000 * 1000000, Category: JusticeCategory, MaxStack: 1}, JusticeBonus: map[string]int{"rep_penalty_reduction": 75}, DurationMin: 60},
			"bounty_license":   {ItemDef: ItemDef{ID: "ITEM-JUSTICE-003", Name: "Bounty Hunter License", Description: "Recurring license (50 $VBV/week) to maintain Clean Hunter status and access the Justice Tier Dashboard.", CostMicro: 50 * 1000000, Category: JusticeCategory, MaxStack: 1, Recurring: true, RecurringDays: 7}, JusticeBonus: map[string]int{"access_justice_dashboard": 1}, DurationMin: 10080},
			"arc_net_spy":      {ItemDef: ItemDef{ID: "ITEM-JUSTICE-004", Name: "Arc-Net-Spy", Description: "Reveals the full inventory of a target player for 5 minutes.", CostMicro: 5000 * 1000000, Category: JusticeCategory, MaxStack: 2}, JusticeBonus: map[string]int{"reveal_inventory": 1}, DurationMin: 5},
		}
		for name, item := range justiceItems {
			items = append(items, map[string]interface{}{
				"id":          name,
				"name":        item.Name,
				"description": item.Description,
				"cost_micro":  item.CostMicro,
				"category":    item.Category,
				"max_stack":   item.MaxStack,
			})
		}
	} else {
		worldItems := map[string]UnderworldItem{
			"data_scramble":     {ItemDef: ItemDef{ID: "ITEM-UNDERWORLD-001", Name: "Data Scramble", Description: "Temporarily hides a player's entire match history from public view for 30 minutes.", CostMicro: 4000 * 1000000, Category: UnderworldCategory, MaxStack: 2}, UnderworldBonus: map[string]int{"hide_match_history": 1}, DurationMin: 30},
			"signal_dampener":   {ItemDef: ItemDef{ID: "ITEM-UNDERWORLD-002", Name: "Signal Dampener", Description: "Hides criminality signatures from bounty tracking. Stacks with team members.", CostMicro: 800 * 1000000, Category: UnderworldCategory, MaxStack: 5}, UnderworldBonus: map[string]int{"dampen_signal": 20}, DurationMin: 60},
			"security_override": {ItemDef: ItemDef{ID: "ITEM-UNDERWORLD-003", Name: "Security Override", Description: "Bribes underworld security to force return of a fenced card (5,000 $VBV).", CostMicro: 5000 * 1000000, Category: UnderworldCategory, MaxStack: 1}, UnderworldBonus: map[string]int{"force_card_return": 1}, DurationMin: 0},
			"regulatory_bypass": {ItemDef: ItemDef{ID: "ITEM-UNDERWORLD-004", Name: "Regulatory Bypass Permit", Description: "Reduces corporate tax on salary by 50% for 24 hours.", CostMicro: 100 * 1000000, Category: UnderworldCategory, MaxStack: 1}, UnderworldBonus: map[string]int{"reduce_corp_tax": 50}, DurationMin: 1440},
		}
		for name, item := range worldItems {
			items = append(items, map[string]interface{}{
				"id":          name,
				"name":        item.Name,
				"description": item.Description,
				"cost_micro":  item.CostMicro,
				"category":    item.Category,
				"max_stack":   item.MaxStack,
			})
		}
	}

	writeJSON(w, map[string]interface{}{
		"success": true,
		"faction": faction,
		"items":   items,
	})
}

// HandleBuyFactionItem handles POST /api/faction/shop/buy
func (l *Lobby) HandleBuyFactionItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req factionBuyData
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	faction := strings.ToUpper(strings.TrimSpace(req.Faction))
	itemName := strings.TrimSpace(req.ItemName)

	if faction != "JUSTICE" && faction != "UNDERWORLD" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "faction must be JUSTICE or UNDERWORLD"})
		return
	}
	if itemName == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "item_name is required"})
		return
	}

	clientID := r.URL.Query().Get("wallet")
	if clientID == "" {
		clientID = r.Header.Get("X-Player-Wallet")
	}
	if clientID == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "unauthorized"})
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	stats, ok := l.leaderboard[clientID]
	if !ok {
		writeJSON(w, map[string]interface{}{"success": false, "error": "player not found"})
		return
	}

	// Validate faction match with player's job role
	var hegemonyPath string
	if l.playerService != nil {
		hegemonyPath = l.playerService.GetHegemonyPath(stats.JobRole)
	} else {
		hegemonyPath = "UNKNOWN"
	}

	// Determine the faction for this item and validate
	itemFaction := ""
	if hegemonyPath == "JUSTICE" {
		itemFaction = "JUSTICE"
	} else if hegemonyPath == "UNDERWORLD" {
		itemFaction = "UNDERWORLD"
	}

	if itemFaction != faction {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("This item requires the %s faction. Your current role: %s", faction, stats.JobRole),
		})
		return
	}

	// Look up item cost based on known items
	itemCost := uint64(0)
	switch itemName {
	case "truth_serum":
		itemCost = 2500 * 1000000
	case "reputation_shield":
		itemCost = 3000 * 1000000
	case "bounty_license":
		itemCost = 50 * 1000000
	case "arc_net_spy":
		itemCost = 5000 * 1000000
	case "data_scramble":
		itemCost = 4000 * 1000000
	case "signal_dampener":
		itemCost = 800 * 1000000
	case "security_override":
		itemCost = 5000 * 1000000
	case "regulatory_bypass":
		itemCost = 100 * 1000000
	default:
		writeJSON(w, map[string]interface{}{"success": false, "error": "unknown item"})
		return
	}

	// Check balance
	if l.playerBalances[clientID] < itemCost {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "insufficient funds",
			"required":  itemCost,
			"balance":   l.playerBalances[clientID],
		})
		return
	}

	// Deduct payment
	l.playerBalances[clientID] -= itemCost

	// Add to inventory
	if stats.Inventory == nil {
		stats.Inventory = make(map[string]int)
	}
	stats.Inventory[itemName]++

	// Apply item effect (simplified - in production this would use the engine methods)
	switch itemName {
	case "truth_serum", "reputation_shield":
		stats.ActiveItemBuffs[itemName] = 5 // default duration in minutes
	case "bounty_license":
		stats.BountyLicenseActive = true
	}

	l.leaderboard[clientID] = stats

	log.Printf("[FACTION SHOP] %s purchased %s for %.2f $VBV", clientID[:8]+"...", itemName, float64(itemCost)/1000000.0)

	writeJSON(w, map[string]interface{}{
		"success": true,
		"text":    fmt.Sprintf("Purchased %s!", itemName),
	})
}

// HandleGetCareerProgress handles GET /api/career/progress
func (l *Lobby) HandleGetCareerProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := r.URL.Query().Get("wallet")
	if clientID == "" {
		clientID = r.Header.Get("X-Player-Wallet")
	}
	if clientID == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "unauthorized"})
		return
	}

	l.mutex.RLock()
	defer l.mutex.RUnlock()

	stats, ok := l.leaderboard[clientID]
	if !ok {
		writeJSON(w, map[string]interface{}{"success": false, "error": "player not found"})
		return
	}

	// Ensure CareerXP exists
	if stats.CareerXP == nil {
		stats.CareerXP = &CareerXP{RoleXP: make(map[string]uint64)}
		l.leaderboard[clientID] = stats
	}

	progress := l.GetCareerProgress(clientID)
	writeJSON(w, map[string]interface{}{
		"success": true,
		"data":    progress,
	})
}

// HandleGetRivalryState handles GET /api/rivalry/state
func (l *Lobby) HandleGetRivalryState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := r.URL.Query().Get("wallet")
	if clientID == "" {
		clientID = r.Header.Get("X-Player-Wallet")
	}
	if clientID == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "unauthorized"})
		return
	}

	l.mutex.RLock()
	defer l.mutex.RUnlock()

	stats, ok := l.leaderboard[clientID]
	if !ok {
		writeJSON(w, map[string]interface{}{"success": false, "error": "player not found"})
		return
	}

	var activeRivals []string
	var pendingInvites []map[string]interface{}
	if stats.Rivalry != nil {
		activeRivals = stats.Rivalry.ActiveRivals
		for _, inv := range stats.Rivalry.PendingInvitations {
			pendingInvites = append(pendingInvites, map[string]interface{}{
				"from_wallet": inv.FromWallet,
				"level":       inv.Level,
				"id":          inv.ID,
			})
		}
	}

	writeJSON(w, map[string]interface{}{
		"success":             true,
		"active_rivals":       activeRivals,
		"pending_invitations": pendingInvites,
	})
}

// ============================================================================
// HELPER: Write JSON response
// ============================================================================

func writeJSON(w http.ResponseWriter, data map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if data["success"] == nil {
		data["success"] = false
	}
	json.NewEncoder(w).Encode(data)
}
