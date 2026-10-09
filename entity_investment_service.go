//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"math/bits"
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
//
// THE WRITE LOCK, because this function MUTATES: it lazily initialises
// `l.playerDirectInvestments` for the wallet below. A lazy write under NO lock — or under a READ
// lock, which is the same defect — is a concurrent map write against every other reader, i.e. a
// `fatal error`, not an error value. Its single caller (HandleGetPortfolio, from the HTTP
// goroutine) holds nothing, so acquiring the lock here cannot self-lock.
func (l *Lobby) GetPortfolioForPlayer(wallet string) (*EntityPortfolio, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	stats, ok := l.leaderboard[wallet]
	if !ok {
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

	// Scoped router lock for the LOOKUP ONLY: the node's own lock is taken further down, so the two are
	// never held together — `node.Mu -> tsr.Mu` is an existing order in this tree (the investment doors
	// call RouteCriminalTax while holding a node), and taking them the other way round would invert it.
	var node *EntityMarketNode
	var exists bool
	l.tokenSinkRouter.Mu.RLock()
	node, exists = l.tokenSinkRouter.MarketNodes[data.EntityID]
	l.tokenSinkRouter.Mu.RUnlock()
	if !exists {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Entity not found in market."}`)})
		return
	}

	// Price the portfolio BEFORE the node lock is taken. CalculateTotalPortfolioValueLocked walks
	// the portfolio's market nodes and takes THEIR read lock, and this node is one of them whenever
	// the wallet already holds shares in the entity it is investing in — a re-entrant read lock on
	// the mutex this goroutine is about to hold for writing, i.e. a self-deadlock.
	portfolioValueMicro := l.CalculateTotalPortfolioValueLocked(wallet)

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
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"❌ Minimum investment is %.2f $VBV."}`, float64(minInvestment)/1000000.0))})
		return
	}

	// Check player balance (use micro-units for precision)
	playerBalance := l.playerBalances[wallet]
	if playerBalance < data.AmountMicro {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Insufficient reward balance."}`)})
		return
	}

	// Cap per-entity investment at 25% of total player portfolio (anti-concentration).
	// The value was priced before the node lock was taken — see the hoist above.
	maxEntityInvestment := uint64(float64(portfolioValueMicro) * 0.25)

	if data.AmountMicro > maxEntityInvestment {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"❌ Maximum investment per entity is %.2f $VBV (25%% of portfolio)."}`, float64(maxEntityInvestment)/1000000.0))})
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

	// Broadcast portfolio update to player with current entity state.
	// The node's WRITE lock is held for the whole function (deferred above), so these fields are
	// already protected: re-taking its READ lock here is what froze the process — a goroutine
	// cannot take the same non-re-entrant sync.RWMutex twice.
	currentYield := float64(node.CumulativeYieldPerShare) / 100.0
	l.nodeReserveBalance = node.ReserveBalance // Cache for display

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
		// Scoped router lock for the LOOKUP ONLY: the node's own lock is taken below.
		l.tokenSinkRouter.Mu.RLock()
		node, exists := l.tokenSinkRouter.MarketNodes[entityID]
		l.tokenSinkRouter.Mu.RUnlock()
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
	stats, ok := l.leaderboard[wallet]
	if ok {
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
//
// THE LOCK ORDER, which is the whole reason this function has this shape:
//  0. it takes NO lobby lock. It reads nothing the lobby owns, and `l.mutex` is held across this tree's
//     request bodies, so acquiring it here would turn this function into a self-deadlock for any caller
//     that already holds it. (It had ZERO callers when this was written — verified by grep, and a dormant
//     function is exactly where a lock bug hides longest.)
//  1. `tokenSinkRouter.Mu` is held for the LOOKUP AND NOTHING ELSE: the node pointers are copied out and
//     the lock is released BEFORE any node lock is taken. `node.Mu -> router.Mu` already exists in this
//     tree (the direct-invest doors call RouteCriminalTax while holding a node), so taking a node lock
//     inside that window would CREATE the reverse edge — a genuine two-goroutine deadlock rather than a
//     re-lock.
//  2. each node's READ lock is taken and released to snapshot the reserve its share is priced from.
//  3. each node's WRITE lock is taken ONCE, ALONE, to apply the share.
//
// The body this replaced held `l.mutex` for the whole of it, took `router.Mu.Lock()` and then mutated
// `node.DividendPoolMicro` while holding NEITHER node lock — the same class of defect as the two
// direct-invest doors, one lock down.
func (l *Lobby) ProcessEntityRevenueDistribution(revenueAmount uint64, source string) {
	if l.tokenSinkRouter == nil || revenueAmount == 0 {
		return // No router, or nothing to distribute
	}

	// STEP 1 — LOOKUP ONLY, under the map's own mutex.
	l.tokenSinkRouter.Mu.RLock()
	entityIDs := make([]string, 0, len(l.tokenSinkRouter.MarketNodes))
	nodes := make([]*EntityMarketNode, 0, len(l.tokenSinkRouter.MarketNodes))
	for entityID, node := range l.tokenSinkRouter.MarketNodes {
		if node == nil {
			continue
		}
		entityIDs = append(entityIDs, entityID)
		nodes = append(nodes, node)
	}
	l.tokenSinkRouter.Mu.RUnlock()

	if len(nodes) == 0 {
		return // No active entities to distribute revenue to
	}

	// STEP 2 — READ SNAPSHOT, one node at a time, with NO other lock held. The reserve is read ONCE and
	// used for BOTH the denominator and the share, so the arithmetic is internally consistent; the body
	// this replaced read it twice, the second time with no lock at all.
	reserves := make([]uint64, len(nodes))
	totalReserveBalance := uint64(1) // The original seed, which also keeps the denominator non-zero.
	for i, node := range nodes {
		node.Mu.RLock()
		reserves[i] = node.ReserveBalance
		node.Mu.RUnlock()
		totalReserveBalance += reserves[i]
	}

	distributedAmount := uint64(0)
	for i, node := range nodes {
		entityID := entityIDs[i]

		// Proportional share of revenue based on reserve balance. INTEGER arithmetic: a float multiply
		// here is a float inside the ledger (Architecture Ledger), and `proportionalMicro` is the exact
		// floored form of the same ratio without overflowing uint64.
		entityShare := proportionalMicro(revenueAmount, reserves[i], totalReserveBalance)

		if entityShare > 0 {
			// STEP 3 — the node's WRITE lock, taken ALONE: no router lock and no lobby lock is held here.
			node.Mu.Lock()
			node.DividendPoolMicro += entityShare
			node.Mu.Unlock()
			distributedAmount += entityShare

			log.Printf("[ENTITY_REVENUE] Distributed %.2f $VBV from %s to entity %s (reserve: %.2f)",
				float64(entityShare)/1000000.0, source, entityID, float64(reserves[i])/1000000.0)
		}
	}

	log.Printf("[ENTITY_REVENUE] Total distributed from %s: %.2f $VBV", source, float64(distributedAmount)/1000000.0)
}

// proportionalMicro returns floor(amount × part / whole): the exact integer form of the ratio, with a
// 128-bit intermediate so two uint64 operands cannot wrap around it.
//
// `part >= whole` (a "share" that is not a share of the whole) is CLAMPED to `amount` rather than allowed
// to overflow the quotient, and a zero operand returns 0. It takes NO lock and must never be called while
// one is held: the caller reads the operands inside their own critical section and passes values in.
func proportionalMicro(amount, part, whole uint64) uint64 {
	if amount == 0 || part == 0 || whole == 0 {
		return 0
	}
	if part >= whole {
		return amount
	}
	hi, lo := bits.Mul64(amount, part)
	// part < whole puts the quotient below `amount`, i.e. inside 64 bits, and `bits.Div64` panics only
	// when the quotient would NOT fit — the case this guard has already excluded.
	quotient, _ := bits.Div64(hi, lo, whole)
	return quotient
}

// CalculateTotalPortfolioValue computes the total value of a player's portfolio including investments
// and AMM shares.
//
// THE DOOR: it takes the lobby read lock and delegates. A caller that ALREADY holds the lock (read or
// write) must call CalculateTotalPortfolioValueLocked instead — a `sync.RWMutex` is not re-entrant, so
// taking a read lock inside a held lock is an unconditional self-deadlock that freezes the WHOLE
// process (every request, every WebSocket), not merely the calling request.
func (l *Lobby) CalculateTotalPortfolioValue(wallet string) uint64 {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.CalculateTotalPortfolioValueLocked(wallet)
}

// CalculateTotalPortfolioValueLocked is the lock-held form of CalculateTotalPortfolioValue.
//
// CALLER CONTRACT — both halves are measured somewhere, so neither is a comment nobody reads:
//   - the LOBBY lock MUST be held (read or write). It owns `l.leaderboard`, `l.playerBalances` and the
//     `Portfolio` map inside the leaderboard entry, all three of which are read here.
//   - NO EntityMarketNode.Mu may be held. This function takes each portfolio node's READ lock to price
//     its shares, and a wallet that already holds shares in the entity being transacted on resolves to
//     THE SAME node — a re-entrant read lock on a mutex this goroutine holds for writing. The two
//     direct-invest doors therefore price the portfolio BEFORE they lock the node.
func (l *Lobby) CalculateTotalPortfolioValueLocked(wallet string) uint64 {
	stats, ok := l.leaderboard[wallet]
	if !ok {
		return 0
	}

	totalValueMicro := uint64(1) // Avoid zero division
	
	// Add player's liquid balance
	totalValueMicro += l.playerBalances[wallet]

	// Calculate AMM share values (existing portfolio holdings converted to cash value)
	if l.tokenSinkRouter != nil {
		for entityID, shares := range stats.Portfolio {
			// MarketNodes is owned by the token sink router's OWN mutex. The window holds that
			// mutex for the LOOKUP AND NOTHING ELSE: the node lock below is taken after it is
			// released, so this cannot invert against the node -> router order the investment
			// doors use when they call RouteCriminalTax while holding a node.
			l.tokenSinkRouter.Mu.RLock()
			node, exists := l.tokenSinkRouter.MarketNodes[entityID]
			l.tokenSinkRouter.Mu.RUnlock()
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
		writeJSONStatus(w, 400, map[string]string{"error": "Investment system not available"})
		return
	}

	// Scoped router lock for the LOOKUP ONLY (the same reason as the WS door): `node.Mu` is taken below.
	var node *EntityMarketNode
	var exists bool
	lobby.tokenSinkRouter.Mu.RLock()
	node, exists = lobby.tokenSinkRouter.MarketNodes[req.EntityID]
	lobby.tokenSinkRouter.Mu.RUnlock()
	if !exists || node == nil {
		writeJSONStatus(w, 404, map[string]string{"error": fmt.Sprintf("Entity %s not found in market", req.EntityID)})
		return
	}

	// Price the portfolio BEFORE the node lock is taken — the same reason as the WS door:
	// CalculateTotalPortfolioValueLocked takes the READ lock of every node in the portfolio, and
	// this node is one of them whenever the wallet already holds shares in the entity it is buying
	// into. This handler already holds the LOBBY write lock, so the lock-held form is the right one.
	portfolioValueMicro := lobby.CalculateTotalPortfolioValueLocked(wallet)

	node.Mu.Lock()
	defer node.Mu.Unlock()

	// Check if dividend pool is frozen (justice counter-play)
	if node.IsDividendFrozen {
		writeJSONStatus(w, 403, map[string]string{"error": "This entity's dividends are currently frozen"})
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
		writeJSONStatus(w, 400, map[string]string{"error": fmt.Sprintf("Minimum investment is %.2f $VBV", float64(minInvestment)/1000000.0)})
		return
	}

	// Check player balance (use micro-units for precision)
	playerBalance := lobby.playerBalances[wallet]
	if playerBalance < req.AmountMicro {
		writeJSONStatus(w, 402, map[string]string{"error": "Insufficient reward balance"})
		return
	}

	// Cap per-entity investment at 25% of total player portfolio (anti-concentration).
	// The value was priced before the node lock was taken — see the hoist above.
	maxEntityInvestment := uint64(float64(portfolioValueMicro) * 0.25)

	if req.AmountMicro > maxEntityInvestment {
		writeJSONStatus(w, 400, map[string]string{"error": fmt.Sprintf("Maximum investment per entity is %.2f $VBV (25%% of portfolio)", float64(maxEntityInvestment)/1000000.0)})
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
	_, _ = json.Marshal(investmentPayload)

	// The node's WRITE lock is held for the whole function (deferred above), so this field is already
	// protected: re-taking its READ lock here is what froze the process on every successful invest.
	currentYield := float64(node.CumulativeYieldPerShare) / 100.0

	writeJSONStatus(w, 200, map[string]interface{}{
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
	_, _ = json.Marshal(portfolioUpdate)

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
		writeJSONStatus(w, 400, map[string]string{"error": "Dividend system not available"})
		return
	}

	investments := lobby.playerDirectInvestments[wallet]
	if len(investments) == 0 {
		writeJSONStatus(w, 400, map[string]string{"error": "No active investments to claim dividends from"})
		return
	}

	var totalDividend uint64
	for entityID, investedAmount := range investments {
		// Scoped router lock for the LOOKUP ONLY: the node's own lock is taken below.
		lobby.tokenSinkRouter.Mu.RLock()
		node, exists := lobby.tokenSinkRouter.MarketNodes[entityID]
		lobby.tokenSinkRouter.Mu.RUnlock()
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
		writeJSONStatus(w, 400, map[string]string{"error": "No dividends available to claim yet"})
		return
	}

	// Credit dividend to player's balance (micro-units for precision)
	lobby.playerBalances[wallet] += totalDividend

	// Track claimed dividends on leaderboard state
	stats, ok := lobby.leaderboard[wallet]
	if ok {
		stats.TotalDividendClaimedMicro += totalDividend
	}

	writeJSONStatus(w, 200, map[string]interface{}{
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
		writeJSONStatus(w, 500, map[string]string{"error": err.Error()})
		return
	}

	writeJSONStatus(w, 200, portfolio)
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

	// [] and never null: a consumer iterating `res.history` throws on null, and this repository's
	// convention for an empty list is an empty list.
	history := make([]DividendRecord, 0)
	if lobby.dividendTracker != nil {
		// SNAPSHOT UNDER THE READ LOCK, DERIVE OUTSIDE IT. This route holds NO lock of its own,
		// while the hourly distribution ticker writes LastDistribution and every investment writes
		// playerDirectInvestments — an unfenced RANGE over a concurrent map is a `fatal error:
		// concurrent map iteration and map write`, which is not recoverable. The locked window is
		// therefore two map walks and nothing else. (EntityDividendTracker.Mu is decorative: no
		// reader or writer in this tree ever takes it, so the lobby lock is the map's real owner.)
		mine := lobby.playerDirectInvestments[wallet]
		last := make(map[string]time.Time, len(lobby.dividendTracker.LastDistribution))
		invested := make(map[string]uint64, len(mine))
		lobby.mutex.RLock()
		for entityID, t := range lobby.dividendTracker.LastDistribution {
			last[entityID] = t
		}
		for entityID, amount := range mine {
			invested[entityID] = amount
		}
		lobby.mutex.RUnlock()

		for entityID, lastDist := range last {
			if lobby.tokenSinkRouter == nil {
				continue
			}
			// MarketNodes is owned by the token sink router's OWN mutex, and the lock order here
			// (lobby.mutex released above, router.Mu taken alone) matches the write path.
			lobby.tokenSinkRouter.Mu.RLock()
			node, exists := lobby.tokenSinkRouter.MarketNodes[entityID]
			lobby.tokenSinkRouter.Mu.RUnlock()
			if !exists || node == nil {
				continue
			}

			investedAmount := invested[entityID] // absent = 0, the zero value the old branch used

			history = append(history, DividendRecord{
				EntityID:    entityID,
				AmountMicro: investedAmount, // Track investment amount for reference
				Timestamp:   lastDist,
				DaysAccrued: time.Since(lastDist).Hours() / 24.0,
			})
		}
	}

	writeJSONStatus(w, 200, map[string]interface{}{
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
	// NO LOBBY LOCK, and no UNLOCKED read of the map either: the `len(...)` this body used to do
	// BEFORE taking any lock was a read of a map the persistence worker iterates — a `fatal error`,
	// not a stale count. The map's ONE owner mutex is the router's, so the walk starts from
	// `marketNodeRefs()`, a value snapshot taken and RELEASED inside it; each node's WRITE lock is then
	// taken ALONE (taking it while holding `tsr.Mu` would create `tsr.Mu -> node.Mu`, the REVERSE of
	// the order the investment doors use when they call RouteCriminalTax while holding a node).
	refs := l.tokenSinkRouter.marketNodeRefs()
	if len(refs) == 0 {
		return
	}

	totalInjected := uint64(0)
	
	for _, ref := range refs {
		node := ref.Node
		entityID := ref.EntityID

		// THE WRITE LOCK, because this block MUTATES `DividendPoolMicro` below. A `+=` under a READ
		// lock is not merely weak: two ticks — or a tick and an AMM buy, which writes the same field
		// under the write lock — interleave and one injection is silently LOST, which a player reads
		// as a yield that never arrived rather than as an error.
		node.Mu.Lock()
		
		// The freeze check lives INSIDE the node's write lock now: reading the flag BEFORE taking it
		// was a race against a Tax Auditor freezing the entity mid-tick.
		if node.IsDividendFrozen {
			node.Mu.Unlock()
			continue // Skip frozen entities
		}

		// Inject revenue based on entity's reserve balance (sustainable long-term yield).
		// INTEGER ONLY: 0.1% per day spread over 24 hours is 1/24000 of the reserve, floored. The float
		// form this replaced put a float inside the ledger and produced a value that is not
		// reproducible across platforms (Architecture Ledger: floats are display-only).
		hourlyYield := node.ReserveBalance / 24000
		
		if hourlyYield > 0 {
			node.DividendPoolMicro += hourlyYield
			totalInjected += hourlyYield
			
			log.Printf("[ENTITY_INVESTMENT] Injected %.6f $VBV into entity %s dividend pool", 
				float64(hourlyYield)/1000000.0, entityID)
		}

		node.Mu.Unlock()
	}

	if totalInjected > 0 {
		log.Printf("[ENTITY_INVESTMENT] Total hourly injection: %.2f $VBV", float64(totalInjected)/1000000.0)
		
		// Trigger global update for all players to see updated entity states
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	}
}