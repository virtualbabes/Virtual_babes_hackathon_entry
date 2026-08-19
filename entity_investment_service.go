//go:build !js && !wasm
// +build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"log"
)

// EntityInvestmentRecord tracks a player's direct investment in an entity.
// PILLAR 2: uint64 Precision — all amounts stored as micro-units.
type EntityInvestmentRecord struct {
	EntityID      string    `json:"entity_id"`       // Target entity wallet address
	AmountMicro   uint64    `json:"amount_micro"`    // Amount invested in micro-VBV
	Timestamp     time.Time `json:"timestamp"`       // Investment timestamp
	CumulativeYield float64  `json:"cumulative_yield"` // Yield accrued at investment time (for dividend calculation)
}

// EntityPortfolio tracks all investments for a single player.
type EntityPortfolio struct {
	WalletAddress string                  `json:"wallet_address"`
	Investments   map[string]*EntityInvestmentRecord `json:"investments"` // Key: entity_id -> InvestmentRecord
	TotalClaimed  uint64                 `json:"total_claimed_micro"` // Total dividends claimed across all entities
}

// EntityDividendTracker manages per-entity dividend distribution state.
type EntityDividendTracker struct {
	Mu              sync.RWMutex          `json:"-"`
	EntityPools     map[string]*uint64    `json:"-"`  // Key: entity_id -> pointer to DividendPoolMicro in AMM node
	LastDistribution map[string]time.Time   `json:"-"`  // Key: entity_id -> last distribution timestamp
}

// NewEntityDividendTracker creates a new dividend tracker for all active market nodes.
func (l *Lobby) NewEntityDividendTracker() *EntityDividendTracker {
	tracker := &EntityDividendTracker{
		EntityPools:     make(map[string]*uint64),
		LastDistribution: make(map[string]time.Time),
	}

	if l.tokenSinkRouter == nil {
		return tracker
	}

	l.tokenSinkRouter.Mu.RLock()
	for entityID, node := range l.tokenSinkRouter.MarketNodes {
		tracker.EntityPools[entityID] = &node.DividendPoolMicro
		tracker.LastDistribution[entityID] = time.Now() // Initialize to now (no accrued yield)
	}
	l.tokenSinkRouter.Mu.RUnlock()

	return tracker
}

// GetPortfolioForPlayer returns the complete investment portfolio for a player.
func (l *Lobby) GetPortfolioForPlayer(wallet string) (*EntityPortfolio, error) {
	stats := l.leaderboard[wallet]
	if stats == nil {
		return &EntityPortfolio{WalletAddress: wallet, Investments: make(map[string]*EntityInvestmentRecord)}, nil
	}

	portfolio := &EntityPortfolio{
		WalletAddress: wallet,
		TotalClaimed:  stats.TotalDividendClaimedMicro, // Track claimed dividends on leaderboard state
		Investments:   make(map[string]*EntityInvestmentRecord),
	}

	// Gather investments from AMM portfolio (shares held) + direct investment records
	if l.playerDirectInvestments == nil {
		l.playerDirectInvestments = make(map[string]map[string]uint64) // Key: wallet -> map[entity_id]amountMicro
	}

	for entityID, amount := range l.playerDirectInvestments[wallet] {
		record := &EntityInvestmentRecord{
			EntityID:      entityID,
			AmountMicro:   amount,
			Timestamp:     time.Now(), // Will be set by InvestInEntity handler
			CumulativeYield: 0.0,    // Calculated at investment time
		}

		// Get current cumulative yield for display
		if l.tokenSinkRouter != nil {
			l.tokenSinkRouter.Mu.RLock()
			if node, exists := l.tokenSinkRouter.MarketNodes[entityID]; exists {
				record.CumulativeYield = float64(node.CumulativeYieldPerShare) / 100.0 // Convert micro to decimal
			}
			l.tokenSinkRouter.Mu.RUnlock()
		}

		portfolio.Investments[entityID] = record
	}

	return portfolio, nil
}

// InvestInEntity allows a player to directly invest in an entity's dividend pool.
// PILLAR 1: Entity Markets are not stock exchanges — investments must be $VBV-gated.
func (l *Lobby) handleInvestEntity(env Envelope, data InvestmentData) {
	wallet := env.FromID // Use wallet address as identifier

	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Validate entity exists in market nodes
	if l.tokenSinkRouter == nil {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Investment system not available."}`)})
		return
	}

	node, exists := l.tokenSinkRouter.MarketNodes[data.EntityID]
	if !exists {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Entity not found in market."}`)})
		return
	}

	node.Mu.Lock()
	defer node.Mu.Unlock()

	// Check if dividend pool is frozen (justice counter-play)
	if node.IsDividendFrozen {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ This entity's dividends are currently frozen."}`)})
		return
	}

	// $VBV-gate check: minimum investment threshold based on player liquidity tier
	minInvestment := uint64(100 * 1000000) // Base: 100 VBV micro-units
	if l.playerLiquidityTiers != nil {
		tier := l.playerLiquidityTiers[wallet]
		switch tier {
		case "Peon":
			minInvestment = uint64(50 * 1000000) // Peons: minimum 50 VBV
		case "Apprentice", "Journeyman":
			minInvestment = uint64(25 * 1000000) // Mid-tier: 25 VBV
		default:
			minInvestment = uint64(10 * 1000000) // Expert+: 10 VBV (lower barrier for experienced investors)
		}
	}

	if data.AmountMicro < minInvestment {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"❌ Minimum investment is %.2f $VBV.", float64(minInvestment)/1000000.0})}`)))
		return
	}

	// Check player balance (use micro-units for precision)
	playerBalance := l.playerBalances[wallet]
	if playerBalance < data.AmountMicro {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Insufficient reward balance."}`)})
		return
	}

	// Cap per-entity investment at 25% of total player portfolio (anti-concentration)
	totalPortfolioValue := l.CalculateTotalPortfolioValue(wallet)
	maxEntityInvestment := uint64(float64(totalPortfolioValue) * 0.25)

	if data.AmountMicro > maxEntityInvestment {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"❌ Maximum investment per entity is %.2f $VBV (25%% of portfolio).", float64(maxEntityInvestment)/1000000.0})}`)))
		return
	}

	// Execute investment: transfer from player balance to entity dividend pool
	l.playerBalances[wallet] -= data.AmountMicro
	node.DividendPoolMicro += data.AmountMicro

	// Track direct investment on player's portfolio state
	if l.playerDirectInvestments == nil {
		l.playerDirectInvestments = make(map[string]map[string]uint64)
	}
	if _, exists := l.playerDirectInvestments[wallet]; !exists {
		l.playerDirectInvestments[wallet] = make(map[string]uint64)
	}
	l.playerDirectInvestments[wallet][data.EntityID] += data.AmountMicro

	// Record investment for portfolio tracking
	investmentRecord := &EntityInvestmentRecord{
		EntityID:      data.EntityID,
		AmountMicro:   data.AmountMicro,
		Timestamp:     time.Now(),
		CumulativeYield: float64(node.CumulativeYieldPerShare) / 100.0, // Capture yield at investment time
	}

	if l.playerInvestmentRecords == nil {
		l.playerInvestmentRecords = make(map[string][]*EntityInvestmentRecord)
	}
	l.playerInvestmentRecords[wallet] = append(l.playerInvestmentRecords[wallet], investmentRecord)

	// Route 1% protocol fee via TokenSinkRouter (same as AMM trades)
	feeMicro := uint64(float64(data.AmountMicro)*0.01 + 0.5)
	if l.tokenSinkRouter != nil {
		matrix := RevenueSplitMatrix{FaucetShare: 0.80, ClubShare: 0.0, GovernanceShare: 0.20}
		l.tokenSinkRouter.RouteCriminalTax("ENTITY_INVESTMENT_FEE", feeMicro, matrix, 0, "arena_center")
	}

	netInvestment := data.AmountMicro - feeMicro // Net amount after protocol fee (already added to DividendPool)

	log.Printf("[INVESTMENT] Player %s invested %.2f $VBV in entity %s (net: %.2f)", wallet, float64(data.AmountMicro)/1000000.0, data.EntityID, float64(netInvestment)/1000000.0)

	// Send confirmation to player
	investmentPayload := map[string]interface{}{
		"text": fmt.Sprintf("✅ Invested %.2f $VBV in %s (net: %.2f after fees)", 
			float64(data.AmountMicro)/1000000.0, data.EntityID, float64(netInvestment)/1000000.0),
	}
	jsonPayload, _ := json.Marshal(investmentPayload)
	l.sendToClientLocked(env.FromID, Envelope{Type: "investment_confirmed", Payload: json.RawMessage(jsonPayload)})

	// Broadcast portfolio update to player with current entity state
	node.Mu.RLock()
	currentYield := float64(node.CumulativeYieldPerShare) / 100.0
	l.nodeReserveBalance := node.ReserveBalance // Cache for display
	node.Mu.RUnlock()

	portfolioUpdate := map[string]interface{}{
		"entity_id":          data.EntityID,
		"investment_amount":  float64(data.AmountMicro) / 1000000.0,
		"net_investment":     float64(netInvestment) / 1000000.0,
		"current_yield_per_share": currentYield,
	}
	portfolioJSON, _ := json.Marshal(portfolioUpdate)
	l.sendToClientLocked(env.FromID, Envelope{Type: "investment_update", Payload: json.RawMessage(portfolioJSON)})

	// Trigger global sync for all players to see updated entity valuations
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// ClaimDividends allows a player to claim accrued dividends from their investments.
func (l *Lobby) handleClaimDividends(env Envelope, data DividendClaimData) {
	wallet := env.FromID

	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.tokenSinkRouter == nil || l.playerDirectInvestments == nil {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Dividend system not available."}`)})
		return
	}

	investments := l.playerDirectInvestments[wallet]
	if len(investments) == 0 {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ No active investments to claim dividends from."}`)})
		return
	}

	var totalDividend uint64
	for entityID, investedAmount := range investments {
		node, exists := l.tokenSinkRouter.MarketNodes[entityID]
		if !exists || node.IsDividendFrozen {
			continue // Skip frozen or non-existent entities
		}

		node.Mu.Lock()
		
		// Calculate accrued yield since last distribution
		lastDistTime := l.dividendTracker.LastDistribution[entityID]
		now := time.Now()
		daysSinceLastDistribution := now.Sub(lastDistTime).Hours() / 24.0
		
		if daysSinceLastDistribution < 1/24.0 { // Less than 1 hour — no yield yet
			node.Mu.Unlock()
			continue
		}

		// Dividend accrual: entity generates revenue proportional to its reserve balance
		// Base rate: 0.5% daily yield on dividend pool (sustainable long-term)
		dailyYieldRate := 0.005 // 0.5% per day
		
		accruedDividends := uint64(float64(node.DividendPoolMicro) * dailyYieldRate * daysSinceLastDistribution)
		
		if accruedDividends == 0 {
			node.Mu.Unlock()
			continue
		}

		// Player's share of dividends proportional to their investment in the pool
		playerShare := uint64(float64(accruedDividends) * float64(investedAmount) / float64(node.DividendPoolMicro))
		
		if playerShare == 0 {
			node.Mu.Unlock()
			continue
		}

		totalDividend += playerShare
		
		// Update last distribution timestamp
		l.dividendTracker.LastDistribution[entityID] = now
		
		node.Mu.Unlock()
	}

	if totalDividend == 0 {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ No dividends available to claim yet."}`)})
		return
	}

	// Credit dividend to player's balance (micro-units for precision)
	l.playerBalances[wallet] += totalDividend
	
	// Track claimed dividends on leaderboard state
	stats := l.leaderboard[wallet]
	if stats != nil {
		stats.TotalDividendClaimedMicro += totalDividend
	}

	// Send confirmation to player
	dividendPayload := map[string]interface{}{
		"text": fmt.Sprintf("✅ Claimed %.2f $VBV in dividends from %d entity investments.", 
			float64(totalDividend)/1000000.0, len(investments)),
	}
	jsonPayload, _ := json.Marshal(dividendPayload)
	l.sendToClientLocked(env.FromID, Envelope{Type: "dividend_claimed", Payload: json.RawMessage(jsonPayload)})

	log.Printf("[DIVIDEND] Player %s claimed %.2f $VBV from %d entity investments", wallet, float64(totalDividend)/1000000.0, len(investments))

	// Trigger global sync for all players to see updated dividend states
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// ProcessEntityRevenueDistribution routes entity revenue (from AMM trades) into dividend pools.
func (l *Lobby) ProcessEntityRevenueDistribution(revenueAmount uint64, source string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.tokenSinkRouter == nil || len(l.tokenSinkRouter.MarketNodes) == 0 {
		return // No active entities to distribute revenue to
	}

	// Distribute entity revenue proportionally across all market nodes based on reserve balance
	totalReserveBalance := uint64(1) // Avoid division by zero
	for _, node := range l.tokenSinkRouter.MarketNodes {
		node.Mu.RLock()
		totalReserveBalance += node.ReserveBalance
		node.Mu.RUnlock()
	}

	l.tokenSinkRouter.Mu.Lock()
	defer l.tokenSinkRouter.Mu.Unlock()

	distributedAmount := uint64(0)
	for entityID, node := range l.tokenSinkRouter.MarketNodes {
		if totalReserveBalance == 0 {
			break
		}

		// Proportional share of revenue based on reserve balance
		entityShare := uint64(float64(revenueAmount) * float64(node.ReserveBalance) / float64(totalReserveBalance))
		
		if entityShare > 0 {
			node.DividendPoolMicro += entityShare
			distributedAmount += entityShare
			
			log.Printf("[ENTITY_REVENUE] Distributed %.2f $VBV from %s to entity %s (reserve: %.2f)", 
				float64(entityShare)/1000000.0, source, entityID, float64(node.ReserveBalance)/1000000.0)
		}
	}

	log.Printf("[ENTITY_REVENUE] Total distributed from %s: %.2f $VBV", source, float64(distributedAmount)/1000000.0)
}

// CalculateTotalPortfolioValue computes the total value of a player's portfolio including investments and AMM shares.
func (l *Lobby) CalculateTotalPortfolioValue(wallet string) uint64 {
	stats := l.leaderboard[wallet]
	if stats == nil {
		return 0
	}

	totalValueMicro := uint64(1) // Avoid zero division
	
	// Add player's liquid balance
	totalValueMicro += l.playerBalances[wallet]

	// Calculate AMM share values (existing portfolio holdings converted to cash value)
	if l.tokenSinkRouter != nil {
		for entityID, shares := range stats.Portfolio {
			node, exists := l.tokenSinkRouter.MarketNodes[entityID]
			if !exists || node == nil {
				continue
			}

			node.Mu.RLock()
			
			// Calculate current share value using AMM bonding curve (sell price)
			sharesAsMicro := uint64(shares * 100.0) // Convert float shares to micro-shares for calculation
			
			if node.TotalSharesIssued > 0 && node.ReserveBalance > 0 {
				// Simplified: value = reserve / total_shares (current market price per share in micro-units)
				valuePerShare := node.ReserveBalance / node.TotalSharesIssued
				
				entityValueMicro := sharesAsMicro * valuePerShare
				totalValueMicro += entityValueMicro
			}
			
			node.Mu.RUnlock()
		}
	}

	return totalValueMicro
}

// ============================================================================
// EntityInvestmentService — HTTP handler wrapper (KEY 3.5)
// This struct provides a clean service boundary for PILLAR 2 entity investment routes.
// ============================================================================

type EntityInvestmentService struct{}

func NewEntityInvestmentService() *EntityInvestmentService {
	return &EntityInvestmentService{}
}

// HandleDirectInvest is the HTTP handler wrapper for POST /api/invest/entity.
func (s *EntityInvestmentService) HandleDirectInvest(lobby *Lobby, w http.ResponseWriter, r *http.Request) {
	wallet := extractWalletFromRequest(r)
	if wallet == "" {
		http.Error(w, "wallet required", http.StatusBadRequest)
		return
	}

	var req struct {
		EntityID  string `json:"entity_id"`
		AmountMicro uint64 `json:"amount_micro"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	lobby.mutex.Lock()
	defer lobby.mutex.Unlock()

	// Validate entity exists in market nodes
	if lobby.tokenSinkRouter == nil {
		writeJSON(w, 400, map[string]string{"error": "Investment system not available"})
		return
	}

	node, exists := lobby.tokenSinkRouter.MarketNodes[req.EntityID]
	if !exists || node == nil {
		writeJSON(w, 404, map[string]string{"error": fmt.Sprintf("Entity %s not found in market", req.EntityID)})
		return
	}

	node.Mu.Lock()
	defer node.Mu.Unlock()

	// Check if dividend pool is frozen (justice counter-play)
	if node.IsDividendFrozen {
		writeJSON(w, 403, map[string]string{"error": "This entity's dividends are currently frozen"})
		return
	}

	// $VBV-gate check: minimum investment threshold based on player liquidity tier
	minInvestment := uint64(100 * 1000000) // Base: 100 VBV micro-units
	if lobby.playerLiquidityTiers != nil {
		tier, _ := lobby.playerLiquidityTiers[wallet]
		switch tier {
		case "Peon":
			minInvestment = uint64(50 * 1000000) // Peons: minimum 50 VBV
		case "Apprentice", "Journeyman":
			minInvestment = uint64(25 * 1000000) // Mid-tier: 25 VBV
		default:
			minInvestment = uint64(10 * 1000000) // Expert+: 10 VBV (lower barrier for experienced investors)
		}
	}

	if req.AmountMicro < minInvestment {
		writeJSON(w, 400, map[string]string{"error": fmt.Sprintf("Minimum investment is %.2f $VBV", float64(minInvestment)/1000000.0)})
		return
	}

	// Check player balance (use micro-units for precision)
	playerBalance := lobby.playerBalances[wallet]
	if playerBalance < req.AmountMicro {
		writeJSON(w, 402, map[string]string{"error": "Insufficient reward balance"})
		return
	}

	// Cap per-entity investment at 25% of total player portfolio (anti-concentration)
	totalPortfolioValue := lobby.CalculateTotalPortfolioValue(wallet)
	maxEntityInvestment := uint64(float64(totalPortfolioValue) * 0.25)

	if req.AmountMicro > maxEntityInvestment {
		writeJSON(w, 400, map[string]string{"error": fmt.Sprintf("Maximum investment per entity is %.2f $VBV (25%% of portfolio)", float64(maxEntityInvestment)/1000000.0)})
		return
	}

	// Execute investment: transfer from player balance to entity dividend pool
	lobby.playerBalances[wallet] -= req.AmountMicro
	node.DividendPoolMicro += req.AmountMicro

	// Track direct investment on player's portfolio state
	if lobby.playerDirectInvestments == nil {
		lobby.playerDirectInvestments = make(map[string]map[string]uint64)
	}
	if _, exists := lobby.playerDirectInvestments[wallet]; !exists {
		lobby.playerDirectInvestments[wallet] = make(map[string]uint64)
	}
	lobby.playerDirectInvestments[wallet][req.EntityID] += req.AmountMicro

	// Record investment for portfolio tracking
	investmentRecord := &EntityInvestmentRecord{
		EntityID:        req.EntityID,
		AmountMicro:     req.AmountMicro,
		Timestamp:       time.Now(),
		CumulativeYield: float64(node.CumulativeYieldPerShare) / 100.0, // Capture yield at investment time
	}

	if lobby.playerInvestmentRecords == nil {
		lobby.playerInvestmentRecords = make(map[string][]*EntityInvestmentRecord)
	}
	lobby.playerInvestmentRecords[wallet] = append(lobby.playerInvestmentRecords[wallet], investmentRecord)

	// Route 1% protocol fee via TokenSinkRouter (same as AMM trades)
	feeMicro := uint64(float64(req.AmountMicro)*0.01 + 0.5)
	if lobby.tokenSinkRouter != nil {
		matrix := RevenueSplitMatrix{FaucetShare: 0.80, ClubShare: 0.0, GovernanceShare: 0.20}
		lobby.tokenSinkRouter.RouteCriminalTax("ENTITY_INVESTMENT_FEE", feeMicro, matrix, 0, "arena_center")
	}

	netInvestment := req.AmountMicro - feeMicro // Net amount after protocol fee (already added to DividendPool)

	log.Printf("[INVESTMENT] Player %s invested %.2f $VBV in entity %s (net: %.2f)", wallet, float64(req.AmountMicro)/1000000.0, req.EntityID, float64(netInvestment)/1000000.0)

	// Send confirmation to player
	investmentPayload := map[string]interface{}{
		"text": fmt.Sprintf("✅ Invested %.2f $VBV in %s (net: %.2f after fees)",
			float64(req.AmountMicro)/1000000.0, req.EntityID, float64(netInvestment)/1000000.0),
	}
	jsonPayload, _ := json.Marshal(investmentPayload)

	node.Mu.RLock()
	currentYield := float64(node.CumulativeYieldPerShare) / 100.0
	node.Mu.RUnlock()

	writeJSON(w, 200, map[string]interface{}{
		"status":              "invested",
		"entity_id":           req.EntityID,
		"invested_micro":      netInvestment,
		"net_investment":      float64(netInvestment) / 1000000.0,
		"current_yield_per_share": currentYield,
	})

	// Broadcast portfolio update to player with current entity state
	portfolioUpdate := map[string]interface{}{
		"entity_id":             req.EntityID,
		"investment_amount":     float64(req.AmountMicro) / 1000000.0,
		"net_investment":        float64(netInvestment) / 1000000.0,
		"current_yield_per_share": currentYield,
	}
	portfolioJSON, _ := json.Marshal(portfolioUpdate)

	// Trigger global sync for all players to see updated entity valuations
	go func() { lobby.broadcast <- lobby.getLobbyUpdateMsg() }()
}

// HandleClaimDividend is the HTTP handler wrapper for POST /api/claim/dividends.
func (s *EntityInvestmentService) HandleClaimDividend(lobby *Lobby, w http.ResponseWriter, r *http.Request) {
	wallet := extractWalletFromRequest(r)
	if wallet == "" {
		http.Error(w, "wallet required", http.StatusBadRequest)
		return
	}

	lobby.mutex.Lock()
	defer lobby.mutex.Unlock()

	if lobby.tokenSinkRouter == nil || lobby.playerDirectInvestments == nil {
		writeJSON(w, 400, map[string]string{"error": "Dividend system not available"})
		return
	}

	investments := lobby.playerDirectInvestments[wallet]
	if len(investments) == 0 {
		writeJSON(w, 400, map[string]string{"error": "No active investments to claim dividends from"})
		return
	}

	var totalDividend uint64
	for entityID, investedAmount := range investments {
		node, exists := lobby.tokenSinkRouter.MarketNodes[entityID]
		if !exists || node.IsDividendFrozen {
			continue // Skip frozen or non-existent entities
		}

		node.Mu.Lock()

		// Calculate accrued yield since last distribution
		lastDistTime := lobby.dividendTracker.LastDistribution[entityID]
		now := time.Now()
		daysSinceLastDistribution := now.Sub(lastDistTime).Hours() / 24.0

		if daysSinceLastDistribution < 1/24.0 { // Less than 1 hour — no yield yet
			node.Mu.Unlock()
			continue
		}

		// Dividend accrual: entity generates revenue proportional to its reserve balance
		dailyYieldRate := 0.005 // 0.5% per day

		accruedDividends := uint64(float64(node.DividendPoolMicro) * dailyYieldRate * daysSinceLastDistribution)

		if accruedDividends == 0 {
			node.Mu.Unlock()
			continue
		}

		// Player's share of dividends proportional to their investment in the pool
		playerShare := uint64(float64(accruedDividends) * float64(investedAmount) / float64(node.DividendPoolMicro))

		if playerShare == 0 {
			node.Mu.Unlock()
			continue
		}

		totalDividend += playerShare

		// Update last distribution timestamp
		lobby.dividendTracker.LastDistribution[entityID] = now

		node.Mu.Unlock()
	}

	if totalDividend == 0 {
		writeJSON(w, 400, map[string]string{"error": "No dividends available to claim yet"})
		return
	}

	// Credit dividend to player's balance (micro-units for precision)
	lobby.playerBalances[wallet] += totalDividend

	// Track claimed dividends on leaderboard state
	stats := lobby.leaderboard[wallet]
	if stats != nil {
		stats.TotalDividendClaimedMicro += totalDividend
	}

	writeJSON(w, 200, map[string]interface{}{
		"status":      "dividends_claimed",
		"amount_micro": totalDividend,
		"entities":    len(investments),
	})

	log.Printf("[DIVIDEND] Player %s claimed %.2f $VBV from %d entity investments", wallet, float64(totalDividend)/1000000.0, len(investments))

	// Trigger global sync for all players to see updated dividend states
	go func() { lobby.broadcast <- lobby.getLobbyUpdateMsg() }()
}

// HandleGetPortfolio is the HTTP handler wrapper for GET /api/invest/portfolio.
func (s *EntityInvestmentService) HandleGetPortfolio(lobby *Lobby, w http.ResponseWriter, r *http.Request) {
	wallet := extractWalletFromRequest(r)
	if wallet == "" {
		http.Error(w, "wallet required", http.StatusBadRequest)
		return
	}

	portfolio, err := lobby.GetPortfolioForPlayer(wallet)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, 200, portfolio)
}

// HandleDividendHistory is the HTTP handler wrapper for GET /api/invest/dividends/history.
func (s *EntityInvestmentService) HandleDividendHistory(lobby *Lobby, w http.ResponseWriter, r *http.Request) {
	wallet := extractWalletFromRequest(r)
	if wallet == "" {
		http.Error(w, "wallet required", http.StatusBadRequest)
		return
	}

	type DividendRecord struct {
		EntityID      string    `json:"entity_id"`
		AmountMicro   uint64    `json:"amount_micro"`
		Timestamp     time.Time `json:"timestamp"`
		DaysAccrued   float64   `json:"days_accrued"`
	}

	var history []DividendRecord
	if lobby.dividendTracker != nil {
		for entityID, lastDist := range lobby.dividendTracker.LastDistribution {
			node, exists := lobby.tokenSinkRouter.MarketNodes[entityID]
			if !exists || node == nil {
				continue
			}

			investedAmount := uint64(0)
			if lobby.playerDirectInvestments != nil {
				investedAmount = lobby.playerDirectInvestments[wallet][entityID]
			}

			history = append(history, DividendRecord{
				EntityID:    entityID,
				AmountMicro: investedAmount, // Track investment amount for reference
				Timestamp:   lastDist,
				DaysAccrued: time.Now().Sub(lastDist).Hours() / 24.0,
			})
		}
	}

	writeJSON(w, 200, map[string]interface{}{
		"wallet": wallet,
		"history": history,
	})
}

// InvestmentData represents the data structure for investment requests.
type InvestmentData struct {
	EntityID     string `json:"entity_id"`      // Target entity wallet address
	AmountMicro  uint64 `json:"amount_micro"`   // Amount to invest in micro-VBV
}

// DividendClaimData represents dividend claim request parameters.
type DividendClaimData struct {
	WalletAddress string `json:"wallet_address"` // Player claiming dividends (from envelope)
	EntityID      string `json:"entity_id,omitempty"` // Optional: specific entity, empty = all entities
	AmountMicro   uint64 `json:"amount_micro,omitempty"` // Optional: specific amount to claim
}

// InitializeEntityInvestmentSystem sets up the investment infrastructure on lobby creation.
func (l *Lobby) InitializeEntityInvestmentSystem() {
	l.dividendTracker = l.NewEntityDividendTracker()
	
	// Initialize player direct investments map if not already done
	if l.playerDirectInvestments == nil {
		l.playerDirectInvestments = make(map[string]map[string]uint64) // Key: wallet -> map[entity_id]amountMicro
	}

	log.Println("[ENTITY_INVESTMENT] Investment system initialized")
	
	// Start automatic dividend distribution ticker (every 24 hours by default)
	go l.startDividendDistributionTicker()
}

// startDividendDistributionTicker runs periodic revenue distribution to entity pools.
func (l *Lobby) startDividendDistributionTicker() {
	ticker := time.NewTicker(1 * time.Hour) // Check every hour for new distributions
	defer ticker.Stop()

	log.Println("[ENTITY_INVESTMENT] Dividend distribution ticker started")

	for range ticker.C {
		l.ProcessHourlyEntityRevenueDistribution()
	}
}

// ProcessHourlyEntityRevenueDistribution handles periodic revenue injection into entity pools.
func (l *Lobby) ProcessHourlyEntityRevenueDistribution() {
	if l.tokenSinkRouter == nil || len(l.tokenSinkRouter.MarketNodes) == 0 {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	totalInjected := uint64(0)
	
	for entityID, node := range l.tokenSinkRouter.MarketNodes {
		if node.IsDividendFrozen {
			continue // Skip frozen entities
		}

		node.Mu.RLock()
		
		// Inject revenue based on entity's reserve balance (sustainable long-term yield)
		hourlyYield := uint64(float64(node.ReserveBalance) * 0.001 / 24.0) // 0.1% daily yield, divided by 24 for hourly
		
		if hourlyYield > 0 {
			node.DividendPoolMicro += hourlyYield
			totalInjected += hourlyYield
			
			log.Printf("[ENTITY_INVESTMENT] Injected %.6f $VBV into entity %s dividend pool", 
				float64(hourlyYield)/1000000.0, entityID)
		}

		node.Mu.RUnlock()
	}

	if totalInjected > 0 {
		log.Printf("[ENTITY_INVESTMENT] Total hourly injection: %.2f $VBV", float64(totalInjected)/1000000.0)
		
		// Trigger global update for all players to see updated entity states
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	}
}