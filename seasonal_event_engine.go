//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// InitSeasonalEventEngine initializes the seasonal event engine for a lobby.
func (l *Lobby) InitSeasonalEventEngine() {
	l.seasonEngine = &SeasonalEventEngine{
		ActiveEvents:      make(map[string]*SeasonEvent),
		CurrentRewardPool: make(map[string]*SeasonRewardPool),
		Caches:            make(map[string]*TreasureCache),  // §26.1
		UserEvents:        make(map[string]*UserEvent),     // §26.2
	}

	log.Printf("[INDUSTRIAL] Seasonal Event Engine initialized")
}

// CreateSeasonEvent creates a new seasonal event and adds it to the engine.
func (l *Lobby) CreateSeasonEvent(eventType SeasonEventType, title string, description string, duration time.Duration, multiplier float64, treasuryBudget uint64, createdBy string) (*SeasonEvent, error) {
	l.seasonEngine.Mu.Lock()
	defer l.seasonEngine.Mu.Unlock()

	// Validate inputs
	if multiplier <= 0 || multiplier > 5.0 {
		return nil, fmt.Errorf("multiplier must be between 0 and 5")
	}
	if treasuryBudget == 0 {
		treasuryBudget = uint64(1_000_000) // Default: 1 VBV in micro-units
	}

	now := time.Now()
	eventID := uuid.New().String()

	event := &SeasonEvent{
		EventID:        eventID,
		Type:           eventType,
		Title:          title,
		Description:    description,
		StartTime:      now, // Announcement phase starts immediately
		EndTime:        now.Add(duration),
		Multiplier:     multiplier,
		TreasuryBudget: treasuryBudget,
		Status:         SeasonAnnouncement,
		CreatedBy:      createdBy,
	}

	l.seasonEngine.ActiveEvents[eventID] = event
	l.seasonEngine.CurrentRewardPool[eventID] = &SeasonRewardPool{
		TotalAllocated: treasuryBudget,
		Distributed:    0,
		PerParticipant: make(map[string]uint64),
	}

	log.Printf("[INDUSTRIAL] Seasonal event created: %s (%s) — multiplier %.2fx, budget %d micro-VBV", title, eventType, multiplier, treasuryBudget)

	return event, nil
}

// ActivateEvent transitions an announcement-phase event to active.
func (l *Lobby) ActivateEvent(eventID string) error {
	l.seasonEngine.Mu.Lock()
	defer l.seasonEngine.Mu.Unlock()

	event, exists := l.seasonEngine.ActiveEvents[eventID]
	if !exists {
		return fmt.Errorf("event %s not found", eventID)
	}
	if event.Status != SeasonAnnouncement {
		return fmt.Errorf("event already in phase: %s", event.Status)
	}

	event.Status = SeasonActive
	log.Printf("[INDUSTRIAL] Event activated: %s (%s)", event.Title, event.Type)

	// Broadcast activation to all connected clients
	l.broadcast <- jsonListEnvelope("seasonal_event_activated", json.RawMessage(fmt.Sprintf(`{"event_id":"%s","title":"%s"}`, eventID, event.Title)))

	return nil
}

// RegisterForEvent allows a player wallet to register for an active seasonal event.
func (l *Lobby) RegisterForEvent(eventID string, walletAddress string) error {
	l.seasonEngine.Mu.Lock()
	defer l.seasonEngine.Mu.Unlock()

	event, exists := l.seasonEngine.ActiveEvents[eventID]
	if !exists {
		return fmt.Errorf("event %s not found", eventID)
	}
	if event.Status != SeasonActive && event.Status != SeasonAnnouncement {
		return fmt.Errorf("cannot register for event in phase: %s", event.Status)
	}

	event.ActiveParticipants[walletAddress] = time.Now()
	log.Printf("[INDUSTRIAL] Player registered for event %s: %s", eventID, walletAddress)

	return nil
}

// GetEventMultiplier returns the multiplier to apply during an active seasonal event.
func (l *Lobby) GetEventMultiplier(eventType SeasonEventType) float64 {
	l.seasonEngine.Mu.RLock()
	defer l.seasonEngine.Mu.RUnlock()

	for _, event := range l.seasonEngine.ActiveEvents {
		if event.Type == eventType && event.Status == SeasonActive {
			return event.Multiplier
		}
	}
	return 1.0 // No active event — default multiplier
}

// ResolveEvent transitions an active event to resolution and computes payouts.
func (l *Lobby) ResolveEvent(eventID string) error {
	l.seasonEngine.Mu.Lock()
	defer l.seasonEngine.Mu.Unlock()

	event, exists := l.seasonEngine.ActiveEvents[eventID]
	if !exists {
		return fmt.Errorf("event %s not found", eventID)
	}
	if event.Status != SeasonActive {
		return fmt.Errorf("cannot resolve event in phase: %s", event.Status)
	}

	event.Status = SeasonResolution
	pool := l.seasonEngine.CurrentRewardPool[eventID]

	// §27.5 — ThemeEngine OutcomeBias hook (Phase A). For each registered participant we
	// precompute the theme vector (held lock is seasonEngine.Mu; ComputeThemeVector uses the
	// independent l.mutex, so this is safe). Resolution logic may consume l.OutcomeBias on demand.
	for wallet := range event.ActiveParticipants {
		_ = l.ComputeThemeVector(wallet)
	}

	log.Printf("[INDUSTRIAL] Event resolving: %s — participants: %d, budget: %d micro-VBV", event.Title, len(event.ActiveParticipants), pool.TotalAllocated)

	return nil
}

// DistributeEventRewards distributes treasury rewards to active event participants.
func (l *Lobby) DistributeEventRewards(eventID string) error {
	l.seasonEngine.Mu.Lock()
	defer l.seasonEngine.Mu.Unlock()

	event, exists := l.seasonEngine.ActiveEvents[eventID]
	if !exists {
		return fmt.Errorf("event %s not found", eventID)
	}
	pool, poolExists := l.seasonEngine.CurrentRewardPool[eventID]
	if !poolExists || len(event.ActiveParticipants) == 0 {
		return fmt.Errorf("no reward pool or participants for event %s", eventID)
	}

	event.Status = SeasonTreasuryPayout

	// Calculate per-participant share (equal distribution of treasury budget)
	rewardPerParticipant := uint64(100_000) // Default: 0.1 VBV in micro-units
	if len(event.ActiveParticipants) > 0 {
		rewardPerParticipant = pool.TotalAllocated / uint64(len(event.ActiveParticipants))
	}

	totalDistributed := uint64(0)
	for walletAddr := range event.ActiveParticipants {
		pool.PerParticipant[walletAddr] += rewardPerParticipant
		pool.Distributed += rewardPerParticipant
		totalDistributed += rewardPerParticipant
		log.Printf("[INDUSTRIAL] Event reward distributed to %s: %d micro-VBV", walletAddr, rewardPerParticipant)

		// Broadcast individual reward notification via WebSocket
		if cid := l.getClientIDFromWalletLocked(walletAddr); cid != "" {
			l.sendToClientLocked(cid, Envelope{Type: "seasonal_event_reward", Payload: json.RawMessage(fmt.Sprintf(`{"event_id":"%s","reward_micro":%d,"title":"%s"}`, eventID, rewardPerParticipant, event.Title))})
		}
	}

	log.Printf("[INDUSTRIAL] Event rewards distributed for %s: total %d micro-VBV across %d participants", event.Title, totalDistributed, len(event.ActiveParticipants))

	return nil
}

// AutoTriggerEvent checks all active events and triggers resolution if end time has passed.
func (l *Lobby) AutoTriggerEvents() {
	l.seasonEngine.Mu.Lock()
	defer l.seasonEngine.Mu.Unlock()

	now := time.Now()
	for eventID, event := range l.seasonEngine.ActiveEvents {
		if event.Status == SeasonActive && now.After(event.EndTime) {
			log.Printf("[INDUSTRIAL] Auto-triggering resolution for expired event: %s", event.Title)
			event.Status = SeasonResolution

			// Broadcast expiration to all clients
			l.broadcast <- jsonListEnvelope("seasonal_event_expired", json.RawMessage(fmt.Sprintf(`{"event_id":"%s","title":"%s"}`, eventID, event.Title)))
		}
	}
}

// GetActiveEvents returns a list of current seasonal events for frontend display.
func (l *Lobby) GetActiveEvents() []SeasonEvent {
	l.seasonEngine.Mu.RLock()
	defer l.seasonEngine.Mu.RUnlock()

	events := make([]SeasonEvent, 0, len(l.seasonEngine.ActiveEvents))
	for _, event := range l.seasonEngine.ActiveEvents {
		events = append(events, *event)
	}
	return events
}

// GetParticipantCount returns the number of active participants for an event.
func (l *Lobby) GetEventParticipants(eventID string) int {
	l.seasonEngine.Mu.RLock()
	defer l.seasonEngine.Mu.RUnlock()

	event, exists := l.seasonEngine.ActiveEvents[eventID]
	if !exists {
		return 0
	}
	return len(event.ActiveParticipants)
}

// GetEventRewardPool returns the reward pool status for an event.
func (l *Lobby) GetEventRewardPool(eventID string) SeasonRewardPool {
	l.seasonEngine.Mu.RLock()
	defer l.seasonEngine.Mu.RUnlock()

	pool, exists := l.seasonEngine.CurrentRewardPool[eventID]
	if !exists {
		return SeasonRewardPool{}
	}
	return *pool
}

// ScheduleSeasonalEvent creates a seasonal event that auto-activates after a delay.
func (l *Lobby) ScheduleSeasonalEvent(eventType SeasonEventType, title string, description string, duration time.Duration, multiplier float64, treasuryBudget uint64, createdBy string, activationDelay time.Duration) {
	go func() {
		time.Sleep(activationDelay)

		event, err := l.CreateSeasonEvent(eventType, title, description, duration, multiplier, treasuryBudget, createdBy)
		if err != nil {
			log.Printf("[INDUSTRIAL] Failed to schedule seasonal event: %v", err)
			return
		}

		// Auto-activate after creation delay (e.g., 5 minutes for announcement phase)
		time.Sleep(5 * time.Minute)
		if err := l.ActivateEvent(event.EventID); err != nil {
			log.Printf("[INDUSTRIAL] Failed to activate scheduled event: %v", err)
		}

		log.Printf("[INDUSTRIAL] Scheduled seasonal event activated: %s", title)
	}()
}

// GenerateRandomSeasonalEvent generates a random seasonal event based on available types.
func (l *Lobby) GenerateRandomSeasonalEvent(createdBy string) (*SeasonEvent, error) {
	eventTypes := []SeasonEventType{
		SeasonHarvestFest,
		SeasonShadowAuction,
		SeasonTerritoryWar,
		SeasonCryptoRaid,
		SeasonDiamondWeek,
	}

	rngType := eventTypes[rand.Intn(len(eventTypes))]

	titles := map[SeasonEventType][]string{
		SeasonHarvestFest:   {"Harvest Fest 2026-Q3", "Golden Harvest Week", "Resource Bonanza"},
		SeasonShadowAuction: {"Shadow Auction Night", "Underworld Special Sale", "Black Market Blitz"},
		SeasonTerritoryWar:  {"Grand Territory War", "Clash of Titans", "District Domination"},
		SeasonCryptoRaid:    {"Crypto Raid Week", "Bounty Hunter's Paradise", "Capture Frenzy"},
		SeasonDiamondWeek:   {"Diamond Player Appreciation", "Elite Rewards Week", "VIP Bonanza"},
	}

	titleOptions := titles[rngType]
	if len(titleOptions) == 0 {
		titleOptions = []string{"Mystery Event"}
	}

	rngTitle := titleOptions[rand.Intn(len(titleOptions))]
	duration := time.Duration(3+rand.Intn(5)) * 24 * time.Hour // 3-7 days
	multiplier := float64(1.0 + rand.Float64()*2.0)            // 1.0 - 3.0x

	return l.CreateSeasonEvent(rngType, rngTitle, fmt.Sprintf("A special seasonal event: %s", rngTitle), duration, multiplier, uint64(500_000+rand.Intn(950_000)), createdBy)
}

// HandleCreateSeasonEventHTTP is the HTTP handler for creating a new seasonal event.
func (l *Lobby) handleCreateSeasonEventHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Type           SeasonEventType `json:"type"`
		Title          string          `json:"title"`
		Description    string          `json:"description"`
		DurationHours  float64         `json:"duration_hours"`
		Multiplier     float64         `json:"multiplier"`
		TreasuryBudget uint64          `json:"treasury_budget_micro"`
		CreatedBy      string          `json:"created_by"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	duration := time.Duration(req.DurationHours * float64(time.Hour))
	event, err := l.CreateSeasonEvent(req.Type, req.Title, req.Description, duration, req.Multiplier, req.TreasuryBudget, req.CreatedBy)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
	log.Printf("[INDUSTRIAL] HTTP seasonal event created: %s by %s", req.Title, req.CreatedBy)
}

// HandleActivateSeasonEventHTTP is the HTTP handler for activating a scheduled event.
func (l *Lobby) handleActivateSeasonEventHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		EventID string `json:"event_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.EventID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing event_id"})
		return
	}

	err := l.ActivateEvent(req.EventID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	log.Printf("[INDUSTRIAL] HTTP event activated: %s", req.EventID)
}

// HandleRegisterForSeasonEventHTTP is the HTTP handler for player registration.
func (l *Lobby) handleRegisterForSeasonEventHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		EventID string `json:"event_id"`
		Wallet  string `json:"wallet"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.EventID == "" || req.Wallet == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing event_id or wallet"})
		return
	}

	err := l.RegisterForEvent(req.EventID, req.Wallet)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	log.Printf("[INDUSTRIAL] Player registered for event: %s wallet=%s", req.EventID, req.Wallet)
}

// HandleGetSeasonEventsHTTP is the HTTP handler to list all active seasonal events.
func (l *Lobby) handleGetSeasonEventsHTTP(w http.ResponseWriter, r *http.Request) {
	events := l.GetActiveEvents()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// HandleResolveEventHTTP is the HTTP handler for resolving an active event.
func (l *Lobby) handleResolveSeasonEventHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		EventID string `json:"event_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.EventID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing event_id"})
		return
	}

	err := l.ResolveEvent(req.EventID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	log.Printf("[INDUSTRIAL] HTTP event resolved: %s", req.EventID)
}

// HandleListSeasonEvents is the SeasonalEventEngine receiver wrapper for listing events.
func (se *SeasonalEventEngine) HandleListSeasonEvents(lobby *Lobby, w http.ResponseWriter, r *http.Request) {
	events := lobby.GetActiveEvents()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// HandleJoinSeasonEvent is the SeasonalEventEngine receiver wrapper for player event registration.
func (se *SeasonalEventEngine) HandleJoinSeasonEvent(lobby *Lobby, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		EventID string `json:"event_id"`
		Wallet  string `json:"wallet"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.EventID == "" || req.Wallet == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing event_id or wallet"})
		return
	}

	err := lobby.RegisterForEvent(req.EventID, req.Wallet)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Broadcast join event to all clients via WebSocket
	lobby.broadcast <- jsonListEnvelope("seasonal_event_joined", json.RawMessage(fmt.Sprintf(`{"event_id":"%s","wallet":"%s"}`, req.EventID, req.Wallet)))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "joined", "event_id": req.EventID})
	log.Printf("[INDUSTRIAL] Player joined seasonal event: %s wallet=%s", req.EventID, req.Wallet)
}

// HandleClaimSeasonReward is the SeasonalEventEngine receiver wrapper for claiming rewards.
func (se *SeasonalEventEngine) HandleClaimSeasonReward(lobby *Lobby, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Wallet string `json:"wallet"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Wallet == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing wallet"})
		return
	}

	// Find any active/payout event where this player has participated and unclaimed rewards
	lobby.seasonEngine.Mu.Lock()
	var reward uint64
	var foundEventID string
	for eid, event := range lobby.seasonEngine.ActiveEvents {
		if _, participant := event.ActiveParticipants[req.Wallet]; !participant {
			continue
		}
		pool, poolExists := lobby.seasonEngine.CurrentRewardPool[eid]
		if !poolExists || len(event.ActiveParticipants) == 0 {
			continue
		}

		rewardPerParticipant := uint64(100_000) // Default: 0.1 VBV in micro-units
		if len(event.ActiveParticipants) > 0 {
			rewardPerParticipant = pool.TotalAllocated / uint64(len(event.ActiveParticipants))
		}

		reward += rewardPerParticipant
		foundEventID = eid
		break // Claim from first found eligible event only per request
	}
	lobby.seasonEngine.Mu.Unlock()

	if reward == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "no rewards to claim"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "reward_claimed",
		"event_id":     foundEventID,
		"reward_micro": reward,
	})
	log.Printf("[INDUSTRIAL] Player claimed seasonal event reward: wallet=%s amount=%d micro-VBV", req.Wallet, reward)
}

// HandleSeasonStatus is the SeasonalEventEngine receiver wrapper for getting full status of all events.
func (se *SeasonalEventEngine) HandleSeasonStatus(lobby *Lobby, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	events := lobby.GetActiveEvents()
	type EventWithPool struct {
		Event          SeasonEvent     `json:"event"`
		RewardPoolInfo json.RawMessage `json:"reward_pool,omitempty"`
	}

	result := make([]EventWithPool, 0, len(events))
	for _, evt := range events {
		pool := lobby.GetEventRewardPool(evt.EventID)
		poolJSON, _ := json.Marshal(pool)
		result = append(result, EventWithPool{
			Event:          evt,
			RewardPoolInfo: poolJSON,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
	log.Printf("[INDUSTRIAL] Season status requested: %d active events returned", len(result))
}

// HandleAdminCreateEvent is the SeasonalEventEngine receiver wrapper for admin event creation.
func (se *SeasonalEventEngine) HandleAdminCreateEvent(lobby *Lobby, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Type           SeasonEventType `json:"type"`
		Title          string          `json:"title"`
		Description    string          `json:"description"`
		DurationHours  float64         `json:"duration_hours"`
		Multiplier     float64         `json:"multiplier"`
		TreasuryBudget uint64          `json:"treasury_budget_micro"`
		CreatedBy      string          `json:"created_by"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	duration := time.Duration(req.DurationHours * float64(time.Hour))
	event, err := lobby.CreateSeasonEvent(req.Type, req.Title, req.Description, duration, req.Multiplier, req.TreasuryBudget, req.CreatedBy)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Broadcast event creation to all clients via WebSocket
	lobby.broadcast <- jsonListEnvelope("seasonal_event_created", json.RawMessage(fmt.Sprintf(`{"event_id":"%s","title":"%s"}`, event.EventID, event.Title)))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
	log.Printf("[INDUSTRIAL] Admin created seasonal event: %s by %s", req.Title, req.CreatedBy)
}

// HandleAdminEndEvent is the SeasonalEventEngine receiver wrapper for admin event termination.
func (se *SeasonalEventEngine) HandleAdminEndEvent(lobby *Lobby, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		EventID string `json:"event_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.EventID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing event_id"})
		return
	}

	err := lobby.ResolveEvent(req.EventID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Auto-distribute rewards if participants exist
	dErr := lobby.DistributeEventRewards(req.EventID)
	if dErr == nil {
		log.Printf("[INDUSTRIAL] Admin ended event %s and auto-distributed rewards", req.EventID)
	} else {
		log.Printf("[INDUSTRIAL] Admin ended event %s (reward distribution skipped: %v)", req.EventID, dErr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "event_ended", "event_id": req.EventID})
	log.Printf("[INDUSTRIAL] Admin ended seasonal event: %s", req.EventID)
}

// HandleAdminUpdateRewardPool is the SeasonalEventEngine receiver wrapper for admin reward pool updates.
func (se *SeasonalEventEngine) HandleAdminUpdateRewardPool(lobby *Lobby, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		EventID          string `json:"event_id"`
		AddBudgetMicro   uint64 `json:"add_budget_micro"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.EventID == "" || req.AddBudgetMicro == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing event_id or add_budget_micro"})
		return
	}

	lobby.seasonEngine.Mu.Lock()
	pool, exists := lobby.seasonEngine.CurrentRewardPool[req.EventID]
	if !exists {
		lobby.seasonEngine.Mu.Unlock()
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "event not found"})
		return
	}

	pool.TotalAllocated += req.AddBudgetMicro
	_, eExists := lobby.seasonEngine.ActiveEvents[req.EventID]
	lobby.seasonEngine.Mu.Unlock()

	if !eExists {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "event not found"})
		return
	}

	log.Printf("[INDUSTRIAL] Admin updated reward pool for event %s: +%d micro-VBV (total: %d)", req.EventID, req.AddBudgetMicro, pool.TotalAllocated)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":           "pool_updated",
		"event_id":         req.EventID,
		"added_micro":      req.AddBudgetMicro,
		"total_allocated":  pool.TotalAllocated,
	})

	// Broadcast reward pool update to all clients via WebSocket
	if cid := lobby.getClientIDFromWalletLocked(""); cid != "" {
		lobby.sendToClientLocked(cid, Envelope{Type: "seasonal_event_pool_updated", Payload: json.RawMessage(fmt.Sprintf(`{"event_id":"%s","total_allocated":%d}`, req.EventID, pool.TotalAllocated))})
	}
}

// ============================================================================
// §26 — Treasure Hunt + User Event Functionality Matrix
// ============================================================================

// SpawnTreasureCache (§26.1) creates a findable loot cache at a region coord.
func (l *Lobby) SpawnTreasureCache(region string, x, y, z float64, rewardMicro uint64, rewardKind, rewardRef string, bondMicro uint64, hidden bool, createdBy string, ttl time.Duration) *TreasureCache {
	l.seasonEngine.Mu.Lock()
	defer l.seasonEngine.Mu.Unlock()
	c := &TreasureCache{
		CacheID:    uuid.New().String(),
		Region:     region,
		X:          x, Y: y, Z: z,
		RewardMicro: rewardMicro,
		RewardKind: rewardKind,
		RewardRef:  rewardRef,
		BondMicro:  bondMicro,
		Hidden:     hidden,
		CreatedBy:  createdBy,
		ExpiresAt:  time.Now().Add(ttl),
	}
	l.seasonEngine.Caches[c.CacheID] = c
	return c
}

// ClaimTreasureCache (§26.1) — validated claim; pays reward via RouteCriminalTax (PILLAR 2, F1 closure).
func (l *Lobby) ClaimTreasureCache(cacheID, claimerWallet string) (uint64, error) {
	l.seasonEngine.Mu.Lock()
	c, ok := l.seasonEngine.Caches[cacheID]
	if !ok || c.Claimed || time.Now().After(c.ExpiresAt) {
		l.seasonEngine.Mu.Unlock()
		return 0, fmt.Errorf("cache unavailable")
	}
	// Optional entry bond (BountyHunter pattern) — escrow from claimer.
	if c.BondMicro > 0 {
		bal := l.playerBalances[strings.ToLower(claimerWallet)]
		if bal < c.BondMicro {
			l.seasonEngine.Mu.Unlock()
			return 0, fmt.Errorf("insufficient bond balance")
		}
		l.playerBalances[strings.ToLower(claimerWallet)] -= c.BondMicro
		if l.tokenSinkRouter != nil {
			_ = l.tokenSinkRouter.RouteCriminalTax("TREASURE_BOND", c.BondMicro, RevenueSplitMatrix{FaucetShare: 1.0, ClubShare: 0.0, GovernanceShare: 0.0}, 0, "")
		}
	}
	payout := c.RewardMicro
	c.Claimed = true
	l.seasonEngine.Mu.Unlock()

	if payout > 0 {
		l.mutex.Lock()
		l.playerBalances[strings.ToLower(claimerWallet)] += payout
		l.mutex.Unlock()
		if l.tokenSinkRouter != nil {
			_ = l.tokenSinkRouter.RouteCriminalTax("TREASURE_REWARD", payout, RevenueSplitMatrix{FaucetShare: 1.0, ClubShare: 0.0, GovernanceShare: 0.0}, 0, "")
		}
	}
	l.logAdminAuditLocked("TREASURE_CLAIMED", claimerWallet, fmt.Sprintf("Cache: %s, Region: %s, Reward: %d (%s)", cacheID, c.Region, payout, c.RewardKind))
	return payout, nil
}

// CreateUserEvent (§26.2) — a user-authored event; bonded via §23 (author royalty ≤1000 bps).
func (l *Lobby) CreateUserEvent(eventType UserEventType, title, description, region string, rewardMicro, entryMicro uint64, bondedNFT, creator string, royaltyBps uint64, ttl time.Duration) (*UserEvent, error) {
	if royaltyBps > 1000 {
		return nil, fmt.Errorf("royalty must be <= 1000 bps (§16 Q22)")
	}
	l.seasonEngine.Mu.Lock()
	defer l.seasonEngine.Mu.Unlock()
	e := &UserEvent{
		EventID:     uuid.New().String(),
		Type:        eventType,
		Title:       title,
		Description: description,
		Region:      region,
		RewardMicro: rewardMicro,
		EntryMicro:  entryMicro,
		BondedNFT:   bondedNFT,
		Creator:     creator,
		RoyaltyBps:  royaltyBps,
		Status:      "ANNOUNCEMENT",
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(ttl),
	}
	l.seasonEngine.UserEvents[e.EventID] = e
	l.logAdminAuditLocked("USER_EVENT_CREATED", creator, fmt.Sprintf("Type: %s, Title: %s, Region: %s", eventType, title, region))
	return e, nil
}

// EnterUserEvent (§26.2) — participant pays entry; author earns §23 royalty.
func (l *Lobby) EnterUserEvent(eventID, participantWallet string) error {
	l.seasonEngine.Mu.Lock()
	e, ok := l.seasonEngine.UserEvents[eventID]
	if !ok || e.Status == "RESOLUTION" || time.Now().After(e.ExpiresAt) {
		l.seasonEngine.Mu.Unlock()
		return fmt.Errorf("event unavailable")
	}
	entry := e.EntryMicro
	l.seasonEngine.Mu.Unlock()

	if entry > 0 {
		l.mutex.Lock()
		bal := l.playerBalances[strings.ToLower(participantWallet)]
		if bal < entry {
			l.mutex.Unlock()
			return fmt.Errorf("insufficient entry balance")
		}
		l.playerBalances[strings.ToLower(participantWallet)] -= entry
		// Author royalty via §23 split.
		royalty := (entry * e.RoyaltyBps) / 10000
		if royalty > 0 && e.Creator != "" {
			l.playerBalances[strings.ToLower(e.Creator)] += royalty
		}
		l.mutex.Unlock()
		if l.tokenSinkRouter != nil {
			_ = l.tokenSinkRouter.RouteCriminalTax("USER_EVENT_ENTRY", entry, RevenueSplitMatrix{FaucetShare: 1.0, ClubShare: 0.0, GovernanceShare: 0.0}, 0, "")
		}
	}
	l.logAdminAuditLocked("USER_EVENT_ENTER", participantWallet, fmt.Sprintf("Event: %s, Entry: %d, Royalty: %d", eventID, entry, (entry*e.RoyaltyBps)/10000))
	return nil
}

// GetRegionViews (§25.3 / §26) — aggregate regions for the explorer + menu warp.
// Region list = known region tags (AI citizens + caches + events). Cap per §15 (1 + idx).
func (l *Lobby) GetRegionViews() []*RegionView {
	l.seasonEngine.Mu.RLock()
	defer l.seasonEngine.Mu.RUnlock()

	regions := map[string]*RegionView{}
	ensure := func(r string) *RegionView {
		if r == "" {
			r = "Base"
		}
		rv, ok := regions[r]
		if !ok {
			cap := 1
			if n, err := strconv.Atoi(r); err == nil {
				cap = 1 + n
			}
			rv = &RegionView{Region: r, Cap: cap}
			regions[r] = rv
		}
		return rv
	}
	// AI citizens per region (§15)
	if l.aiEngine != nil {
		for _, c := range l.aiEngine.GetAllCitizens() {
			rv := ensure(c.Region)
			rv.Count++
		}
	}
	// Caches
	for _, c := range l.seasonEngine.Caches {
		if c.Claimed || time.Now().After(c.ExpiresAt) {
			continue
		}
		rv := ensure(c.Region)
		rv.Caches = append(rv.Caches, c)
	}
	// User events
	for _, e := range l.seasonEngine.UserEvents {
		if e.Status == "RESOLUTION" || time.Now().After(e.ExpiresAt) {
			continue
		}
		rv := ensure(e.Region)
		rv.Events = append(rv.Events, e)
	}
	out := make([]*RegionView, 0, len(regions))
	for _, rv := range regions {
		// §30 Pet World: attach the 3D-world power overlay (pets + bots + users in region).
		if l.entityEvents != nil {
			// Resolve pet region via owner's citizen region (pets have no own Region field).
			ownerRegion := func(owner string) string {
				if l.aiEngine != nil {
					if c, ok := l.aiEngine.GetCitizen(owner); ok {
						return c.Region
					}
				}
				return "Base"
			}
			var regionPets []*PetNFT
			var regionBots []*AICitizen
			var regionVehicles []*VehicleNFT
			l.mutex.RLock()
			for _, p := range l.pets {
				if ownerRegion(p.Owner) == rv.Region {
					regionPets = append(regionPets, p)
				}
			}
			// §25.6.1: deployed vehicles join the region power overlay, exactly as pets do.
			for _, v := range l.vehicles {
				if v.Region == rv.Region {
					regionVehicles = append(regionVehicles, v)
				}
			}
			l.mutex.RUnlock()
			if l.aiEngine != nil {
				for _, c := range l.aiEngine.GetAllCitizens() {
					if c.Region == rv.Region {
						regionBots = append(regionBots, c)
					}
				}
			}
			rv.EntityPowerOverlay = l.entityEvents.BuildRegionPowerOverlay(rv.Region, regionPets, regionBots, regionVehicles)
		}
		out = append(out, rv)
	}
	return out
}