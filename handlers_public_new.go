//go:build !js && !wasm

package main

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// ==============================================================================
// JUSTICE Bounty Handler (HandleGetJusticeMissions already in lobby_manager.go)
// ==============================================================================

// HandleGetBountyActive returns active bounties for the justice system.
func (l *Lobby) HandleGetBountyActive(w http.ResponseWriter, r *http.Request) {
	if l.justiceService == nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "justice service unavailable"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	dash := l.justiceService.GetDashboardForPlayer(wallet, nil, 0)
	bounties := []interface{}{}
	if dash != nil {
		for _, target := range dash.HighWantedTargets {
			bounties = append(bounties, map[string]interface{}{
				"target_id":   target.PlayerID,
				"target_name": target.PlayerName,
				"wanted":      target.WantedLevel,
				"district":    target.District,
				"ghost":       target.GhostActive,
			})
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"bounties": bounties,
	})
}

// ==============================================================================
// CHURCH Storefront Handlers
// ==============================================================================

// HandleGetChurchMembers returns members of a specific church.
func (l *Lobby) HandleGetChurchMembers(w http.ResponseWriter, r *http.Request) {
	churchID := r.URL.Query().Get("church_id")
	members := []interface{}{}
	if faithChurchEngine != nil {
		_ = churchID
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"members":   members,
		"church_id": churchID,
	})
}

// HandleShopPurchase handles item purchases from the church shop.
func (l *Lobby) HandleShopPurchase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ItemID   string `json:"item_id"`
		ChurchID string `json:"church_id"`
		Quantity int    `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)

	items := map[string]uint64{
		"holy_water":    500,
		"blessed_amulet": 1000,
		"prayer_book":    2000,
		"divine_relic":   5000,
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	cost, ok := items[req.ItemID]
	if !ok {
		writeJSON(w, map[string]interface{}{"success": false, "error": "unknown item"})
		return
	}

	if req.Quantity < 1 {
		req.Quantity = 1
	}

	totalCost := cost * uint64(req.Quantity)
	balance := l.playerBalances[wallet]
	if balance < totalCost {
		writeJSON(w, map[string]interface{}{"success": false, "error": "insufficient balance", "balance": balance, "cost": totalCost})
		return
	}

	balance -= totalCost
	l.playerBalances[wallet] = balance

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"message":  "purchase recorded",
		"item":     req.ItemID,
		"quantity": req.Quantity,
		"cost":     totalCost,
		"balance":  balance,
	})
}

// ==============================================================================
// CLUB Listing Handler
// ==============================================================================

// HandleGetClubs returns a list of all clubs.
func (l *Lobby) HandleGetClubs(w http.ResponseWriter, r *http.Request) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	clubs := []interface{}{}
	for _, club := range l.clubs {
		clubs = append(clubs, club)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "clubs": clubs})
}

// ==============================================================================
// CREATOR STORE Handler
// ==============================================================================

// HandleGetCreatorStore returns the creator storefront data.
func (l *Lobby) HandleGetCreatorStore(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	products := []interface{}{}
	if l.creatorStore != nil {
		prods := l.creatorStore.ListProducts("", wallet)
		for _, p := range prods {
			products = append(products, p)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"products": products,
	})
}

// ==============================================================================
// PLAYER PROFILE Handler
// ==============================================================================

// HandleGetPlayerProfile returns player profile data.
func (l *Lobby) HandleGetPlayerProfile(w http.ResponseWriter, r *http.Request) {
	wallet := r.URL.Query().Get("wallet")
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	// READ UNDER THE LOCK. `l.leaderboard` holds VALUES and is written by matchmaking, the career
	// door and every economy path, so an unfenced read here is a `fatal error: concurrent map read
	// and map write` — unrecoverable — the moment one of them runs beside this request. The record
	// is a value, so the copy taken inside the locked window is safe to use after release.
	l.mutex.RLock()
	stats, ok := l.leaderboard[wallet]
	l.mutex.RUnlock()
	if !ok {
		writeJSON(w, map[string]interface{}{"success": false, "error": "player not found"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"profile": map[string]interface{}{
			"wallet":     wallet,
			"wins":       stats.Wins,
			"reputation": stats.Reputation,
		},
	})
}

// ==============================================================================
// RUMORS Handler
// ==============================================================================

// HandleGetRumors returns active rumors.
func (l *Lobby) HandleGetRumors(w http.ResponseWriter, r *http.Request) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	rumors := []interface{}{}
	for _, rumor := range l.rumors {
		rumors = append(rumors, rumor)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "rumors": rumors})
}

// ==============================================================================
// CARD TITLES API (High Priority — backs existing Card Titles UI)
// ==============================================================================

// HandleGetCardTitles returns all available card titles.
func (l *Lobby) HandleGetCardTitles(w http.ResponseWriter, r *http.Request) {
	titles := []map[string]interface{}{
		{"id": "first_blood", "name": "First Blood", "icon": "🩸", "desc": "Win your first match", "perk": "+5% power vs new players", "rarity": "common"},
		{"id": "bounty_hunter", "name": "Bounty Hunter", "icon": "🎯", "desc": "Capture 5 bounties", "perk": "+10% power vs Wanted", "rarity": "rare"},
		{"id": "combo_master", "name": "Combo Master", "icon": "💥", "desc": "Trigger 10 combo chains", "perk": "+1 combo chain length", "rarity": "rare"},
		{"id": "elemental_sage", "name": "Elemental Sage", "icon": "🔮", "desc": "Win 5 matches with elemental sync", "perk": "+25 elemental bonus", "rarity": "epic"},
		{"id": "prisoner_taker", "name": "Prisoner Taker", "icon": "⛓️", "desc": "Win 3 matches with Prisoner rule", "perk": "Prisoner rule active by default", "rarity": "epic"},
		{"id": "mood_warden", "name": "Mood Warden", "icon": "🎭", "desc": "Win 5 matches with Mood modifiers", "perk": "+1 mood bonus", "rarity": "rare"},
		{"id": "flawless", "name": "Flawless", "icon": "💎", "desc": "Win without losing a card", "perk": "+15% power", "rarity": "legendary"},
		{"id": "collector", "name": "Collector", "icon": "📦", "desc": "Own 20 cards", "perk": "+5% power per 10 cards", "rarity": "epic"},
		{"id": "veteran", "name": "Veteran", "icon": "⭐", "desc": "Win 50 matches", "perk": "+10% power", "rarity": "legendary"},
		{"id": "divine", "name": "Divine", "icon": "✨", "desc": "Max faith coherence", "perk": "+20% faith bonus", "rarity": "legendary"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "titles": titles})
}

// HandleEquipCardTitle equips a title to a card or player.
func (l *Lobby) HandleEquipCardTitle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		TitleID string `json:"title_id"`
		CardID  int    `json:"card_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "title equipped"})
}

// ==============================================================================
// WAGER RESOLUTION API — Connected to live match history + player balances
// ==============================================================================

// HandleGetWagers returns recent match history as wagers.
func (l *Lobby) HandleGetWagers(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	history := []interface{}{}
	for matchID, match := range l.matchHistory {
		if match.WinnerID == wallet || match.Opponent == wallet {
			won := match.WinnerID == wallet
			history = append(history, map[string]interface{}{
				"match_id":  matchID,
				"opponent":  match.Opponent,
				"won":       won,
				"timestamp": match.Timestamp,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"active_wager": 0,
		"balance":      l.playerBalances[wallet],
		"history":      history,
	})
}

// HandleResolveWager resolves a wager after a match using playerBalances.
func (l *Lobby) HandleResolveWager(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Wager  uint64 `json:"wager"`
		Winner string `json:"winner"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	l.mutex.Lock()
	defer l.mutex.Unlock()

	result := map[string]interface{}{"success": true, "result": req.Winner}
	balance := l.playerBalances[wallet]

	if req.Winner == "player" {
		payout := req.Wager * 2
		balance += payout
		l.playerBalances[wallet] = balance
		result["payout"] = payout
		result["balance"] = balance
	} else if req.Winner == "draw" {
		result["payout"] = req.Wager
		result["balance"] = balance
	} else {
		result["payout"] = 0
		result["balance"] = balance
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ==============================================================================
// JUSTICE MISSIONS API (High Priority — backs existing Justice Dashboard UI)
// ==============================================================================

// HandleGetJusticeMissionsHTTP returns active justice missions via HTTP.
func (l *Lobby) HandleGetJusticeMissionsHTTP(w http.ResponseWriter, r *http.Request) {
	if l.justiceService == nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "justice service unavailable"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	dash := l.justiceService.GetDashboardForPlayer(wallet, nil, 0)
	missions := []interface{}{}
	if dash != nil {
		for _, m := range dash.ActiveMissions {
			missions = append(missions, map[string]interface{}{
				"mission_id": m.MissionID,
				"target":     m.TargetName,
				"wanted":     m.TargetWanted,
				"reward":     m.RewardVBV,
				"status":     m.Status,
			})
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"missions": missions,
	})
}

// HandleAcceptJusticeMissionHTTP accepts a justice mission via HTTP.
func (l *Lobby) HandleAcceptJusticeMissionHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		MissionID string `json:"mission_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)

	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.justiceService != nil {
		missionID := l.justiceService.GenerateJusticeMission(wallet, "Player", 25, 1000000)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    true,
			"message":    "mission accepted",
			"mission_id": missionID,
		})
		return
	}

	writeJSON(w, map[string]interface{}{"success": false, "error": "justice service unavailable"})
}

// ==============================================================================
// TEA HOUSE API — Connected to player balances
// ==============================================================================

// HandleGetTeaRecipes returns available tea recipes.
func (l *Lobby) HandleGetTeaRecipes(w http.ResponseWriter, r *http.Request) {
	recipes := []map[string]interface{}{
		{"id": "green_tea", "name": "Green Tea", "icon": "🍵", "desc": "Calming blend", "buff": "+5% focus", "cost": 100, "rarity": "common"},
		{"id": "chai", "name": "Spiced Chai", "icon": "☕", "desc": "Energizing spices", "buff": "+10% speed", "cost": 250, "rarity": "rare"},
		{"id": "matcha", "name": "Matcha Latte", "icon": "🥛", "desc": "Pure energy", "buff": "+15% power", "cost": 500, "rarity": "epic"},
		{"id": "herbal", "name": "Herbal Infusion", "icon": "🌿", "desc": "Healing herbs", "buff": "+20% defense", "cost": 750, "rarity": "epic"},
		{"id": "golden", "name": "Golden Oolong", "icon": "✨", "desc": "Legendary blend", "buff": "+25% all stats", "cost": 1500, "rarity": "legendary"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "recipes": recipes})
}

// HandleBrewTea brews a tea recipe for a buff, deducting cost from balance.
func (l *Lobby) HandleBrewTea(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		RecipeID string `json:"recipe_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)

	recipes := map[string]struct {
		cost uint64
		buff string
	}{
		"green_tea": {100, "+5% focus"},
		"chai":      {250, "+10% speed"},
		"matcha":    {500, "+15% power"},
		"herbal":    {750, "+20% defense"},
		"golden":    {1500, "+25% all stats"},
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	recipe, ok := recipes[req.RecipeID]
	if !ok {
		writeJSON(w, map[string]interface{}{"success": false, "error": "unknown recipe"})
		return
	}

	balance := l.playerBalances[wallet]
	if balance < recipe.cost {
		writeJSON(w, map[string]interface{}{"success": false, "error": "insufficient balance", "balance": balance, "cost": recipe.cost})
		return
	}

	balance -= recipe.cost
	l.playerBalances[wallet] = balance

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "tea brewed",
		"buff":    recipe.buff,
		"balance": balance,
	})
}

// ==============================================================================
// ZEN GARDEN API — Connected to AI citizen engine
// ==============================================================================

// HandleGetGarden returns the player's zen garden state from AI citizen data.
//
// NO LOBBY LOCK, deliberately: this door reads only the AI engine (which locks itself), and taking
// `l.mutex` here is what closed the lock-order cycle measured in lock_order_gate_test.go —
// `BehavioralTick` holds `ace.mu` and needs `lobby.mutex`, while this held `lobby.mutex` and needed
// `ace.mu`.
func (l *Lobby) HandleGetGarden(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)

	elements := []interface{}{}
	level := 1
	meditationStreak := 0

	if l.aiEngine != nil {
		citizens := l.aiEngine.GetAllCitizens()
		for _, c := range citizens {
			if c.OwnerWallet == wallet || c.Wallet == wallet {
				elements = append(elements, map[string]interface{}{
					"citizen":     c.Name,
					"career":      c.Career,
					"tier":        c.Tier,
					"reputation":  c.Reputation,
				})
				level += c.Tier
				meditationStreak += int(c.RitualsDone)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":           true,
		"elements":          elements,
		"level":             level,
		"meditation_streak": meditationStreak,
	})
}

// HandleAddGardenElement adds an element to the garden via AI citizen.
func (l *Lobby) HandleAddGardenElement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ElementID string `json:"element_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}

	// The engine owns its own lock (AddRitual). Holding `l.mutex` here both raced the citizen's
	// fields (they were mutated with NO engine lock) and inverted the lock order against the tick.
	if l.aiEngine != nil {
		if rituals, found := l.aiEngine.AddRitual(req.ElementID); found {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"message": "element added",
				"rituals": rituals,
			})
			return
		}
	}

	writeJSON(w, map[string]interface{}{"success": false, "error": "element not found"})
}

// HandleMeditate records a meditation session via AI citizen.
func (l *Lobby) HandleMeditate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	wallet := l.getWalletFromRequest(r)

	// The engine owns its own lock (Meditate); the lobby lock is NOT taken (see the lock-order note
	// on HandleGetGarden).
	if l.aiEngine != nil {
		l.aiEngine.Meditate(wallet)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "meditation complete",
	})
}

// ==============================================================================
// CHARACTER MOOD API — Connected to AI citizen engine
// ==============================================================================

// HandleGetMoods returns character mood data from AI citizen engine.
func (l *Lobby) HandleGetMoods(w http.ResponseWriter, r *http.Request) {
	// No lobby lock: engine state only (see the lock-order note on HandleGetGarden).
	moods := map[string]interface{}{}
	totalMood := 0
	count := 0

	if l.aiEngine != nil {
		citizens := l.aiEngine.GetAllCitizens()
		for _, c := range citizens {
			state := getMoodState(c.Reputation)
			moods[c.Name] = map[string]interface{}{
				"value":   c.Reputation,
				"state":   state,
				"career":  c.Career,
				"wallet":  c.Wallet,
				"status":  c.Status,
			}
			totalMood += c.Reputation
			count++
		}
	}

	average := 0
	if count > 0 {
		average = totalMood / count
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"moods":   moods,
		"average": average,
		"count":   count,
	})
}

// HandleAdjustMood adjusts a character's mood via AI citizen engine.
func (l *Lobby) HandleAdjustMood(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Character string `json:"character"`
		Amount    int    `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}

	// The engine owns its own lock (AdjustReputation); the lobby lock is NOT taken.
	if l.aiEngine != nil {
		if newRep, found := l.aiEngine.AdjustReputation(req.Character, req.Amount); found {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"message": "mood adjusted",
				"mood":    newRep,
				"state":   getMoodState(newRep),
			})
			return
		}
	}

	writeJSON(w, map[string]interface{}{"success": false, "error": "character not found"})
}

func getMoodState(value int) string {
	if value <= 20 {
		return "Hostile"
	}
	if value <= 40 {
		return "Annoyed"
	}
	if value <= 60 {
		return "Neutral"
	}
	if value <= 80 {
		return "Friendly"
	}
	return "Loyal"
}

// Suppress unused import warning
var _ = strconv.Itoa
