//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"
)

var (
	ErrExcessiveSlippage = fmt.Errorf("transaction rejected: slippage limits exceeded due to low liquidity density")
)

// getOrCreateMarketNodeLocked retrieves the AMM state for a wallet, initializing it if necessary.
func (l *Lobby) getOrCreateMarketNodeLocked(wallet string) *EntityMarketNode {
	// PILLAR 2: Authoritative Map Linking.
	// Ensure the service utilizes the linked memory space established in the TokenSinkRouter.
	if l.marketNodes == nil {
		if l.tokenSinkRouter != nil && l.tokenSinkRouter.MarketNodes != nil {
			l.marketNodes = l.tokenSinkRouter.MarketNodes
		} else {
			l.marketNodes = make(map[string]*EntityMarketNode)
			if l.tokenSinkRouter != nil {
				l.tokenSinkRouter.MarketNodes = l.marketNodes
			}
		}
	}
	node, exists := l.marketNodes[wallet]
	if !exists {
		// Establish an active baseline supply and liquidity reserve.
		// PILLAR 2: Bancor seed parameters.
		node = &EntityMarketNode{
			EntityID:          wallet,
			TotalSharesIssued: 10000,           // 100 shares (100 units/share)
			ReserveBalance:    50000 * 1000000, // 50,000 VBV Initial Reserve
			ReserveRatio:      0.33,            // Quadratic coefficient
			DividendPoolMicro: 0,               // PILLAR 1: Initialize yield pool
			CumulativeYieldPerShare: 0,        // PILLAR 2: Initialize yield baseline
			IsDividendFrozen:  false,           // PILLAR 3: Initialize freeze status
		}
		l.marketNodes[wallet] = node
	}
	return node
}

/**
 * HandleClaimDividends allows a shareholder to collect accumulated organization yield.
 * PILLAR 1: Yield-Bearing Assets.
 */
func (l *Lobby) HandleClaimDividends(env *Envelope) {
	var data struct { EntityID string `json:"entity_id"` }
	if err := json.Unmarshal(env.Payload, &data); err != nil { return }

	l.mutex.Lock()
	defer l.mutex.Unlock()
	wallet, ok := l.wallets[env.FromID]
	if !ok { return }
	stats := l.leaderboard[wallet]

	targetEntity := strings.ToLower(data.EntityID)
	sharesHeld := stats.Portfolio[targetEntity]
	if sharesHeld == 0 {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Claim Failed: You hold no equity in this entity."}`)})
		return
	}

	node := l.getOrCreateMarketNodeLocked(targetEntity)
	
	// Calculate payout based on the gap between current cumulative yield and shareholder's last claim point.
	// PILLAR 2: Integer Supremacy. (Scaled by 1e12 for fixed-point math).
	if stats.LastClaimedYield == nil { stats.LastClaimedYield = make(map[string]uint64) }
	
	yieldDelta := node.CumulativeYieldPerShare - stats.LastClaimedYield[targetEntity]
	if yieldDelta == 0 {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"⚠️ <b>YIELD STAGNANT:</b> No new dividends accumulated since last claim."}`)})
		return
	}

	payoutMicro := (yieldDelta * sharesHeld) / 1000000000000
	
	// Execute Liability Shift
	l.playerBalances[wallet] += payoutMicro
	stats.LastClaimedYield[targetEntity] = node.CumulativeYieldPerShare
	l.leaderboard[wallet] = stats

	l.logAdminAuditLocked("DIVIDEND_CLAIMED", wallet, fmt.Sprintf("Entity: %s, Amount: %d", targetEntity, payoutMicro))
	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>YIELD HARVESTED:</b> You claimed %.2f $VBV in dividends."}`, float64(payoutMicro)/1000000.0))})
	l.applyDynamicScalingLocked()
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

/**
 * HandleJusticeFreezeDividends allows a Tax Auditor to disrupt a CEO's revenue stream.
 * PILLAR 3: Justice path.
 */
func (l *Lobby) HandleJusticeFreezeDividends(env *Envelope) {
	var data struct { TargetCEO string `json:"target_wallet"` }
	if err := json.Unmarshal(env.Payload, &data); err != nil { return }

	l.mutex.Lock()
	defer l.mutex.Unlock()
	wallet, ok := l.wallets[env.FromID]
	if !ok { return }
	stats := l.leaderboard[wallet]

	if stats.JobRole != "Tax Auditor" {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Access Denied: Dividend Freeze restricted to 'Tax Auditor' career path."}`)})
		return
	}

	targetCEO := strings.ToLower(data.TargetCEO)
	ceoStats, exists := l.leaderboard[targetCEO]
	if !exists || ceoStats.WantedLevel <= 30 {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Action Blocked: Target infamy below regulatory threshold (Wanted 30+)."}`)})
		return
	}

	node := l.getOrCreateMarketNodeLocked(targetCEO)
	node.IsDividendFrozen = true

	l.logAdminAuditLocked("DIVIDEND_FROZEN", wallet, fmt.Sprintf("CEO: %s", targetCEO))
	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"⚖️ <b>ASSET FREEZE:</b> Dividends are now being diverted to the Faucet as Regulatory Fines."}`)})
	
	if ceoCID := l.getClientIDFromWalletLocked(targetCEO); ceoCID != "" {
		l.sendToClientLocked(ceoCID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"🔥 <b>REGULATORY ALERT:</b> Your organization's dividend pool has been FROZEN due to extreme infamy."}`)})
	}

	// PILLAR 3: Justice Path (Sector-wide broadcast)
	payload, _ := json.Marshal(map[string]string{"text": fmt.Sprintf("⚖️ <b>REGULATORY ALERT:</b> Dividends for %s have been FROZEN by the Tax Auditor.", l.oracleService.ResolveEnvoiName(l, targetCEO))})
	l.broadcast <- jsonListEnvelope("chat", payload)

	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

/**
 * HandleHarvestAllDividends allows a player to collect all pending organization yield in one action.
 * PILLAR 1: Yield-Bearing Assets.
 */
func (l *Lobby) HandleHarvestAllDividends(env *Envelope) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	wallet, ok := l.wallets[env.FromID]
	if !ok { return }
	stats := l.leaderboard[wallet]

	if stats.Portfolio == nil || len(stats.Portfolio) == 0 {
		return
	}

	if stats.LastClaimedYield == nil { stats.LastClaimedYield = make(map[string]uint64) }
	
	var totalPayoutMicro uint64
	entityCount := 0

	// PILLAR 2: Integer Supremacy (1e12 Scaling for fixed-point math)
	for targetEntity, sharesHeld := range stats.Portfolio {
		if sharesHeld == 0 { continue }
		node := l.getOrCreateMarketNodeLocked(targetEntity)
		
		yieldDelta := node.CumulativeYieldPerShare - stats.LastClaimedYield[targetEntity]
		if yieldDelta > 0 {
			payoutMicro := (yieldDelta * sharesHeld) / 1000000000000
			totalPayoutMicro += payoutMicro
			stats.LastClaimedYield[targetEntity] = node.CumulativeYieldPerShare
			entityCount++
		}
	}

	if totalPayoutMicro == 0 {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"⚠️ <b>HARVEST:</b> No pending dividends found in portfolio."}`)})
		return
	}

	l.playerBalances[wallet] += totalPayoutMicro
	l.leaderboard[wallet] = stats

	l.logAdminAuditLocked("DIVIDENDS_HARVESTED_ALL", wallet, fmt.Sprintf("Payout: %d, Entities: %d", totalPayoutMicro, entityCount))
	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>HARVEST SUCCESS:</b> Collected %.2f $VBV from %d organizations."}`, float64(totalPayoutMicro)/1000000.0, entityCount))})
	
	l.applyDynamicScalingLocked()
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// GetSpotPrice calculates the instantaneous spot price of a single unit.
// PILLAR 2: Integer-Supremacy ratio.
func (node *EntityMarketNode) GetSpotPrice() float64 {
	// PILLAR 2: Arithmetic Hardening.
	// Ensure price never hits zero to prevent divide-by-zero errors in slippage calculations.
	// We also guard against an uninitialized or zero ReserveRatio.
	if node.TotalSharesIssued == 0 || node.ReserveBalance == 0 || node.ReserveRatio <= 0 {
		return 0.01 // Base minimum floor price (0.01 micro-VBV)
	}
	// Price = ReserveBalance / (TotalSharesIssued * ReserveRatio)
	price := float64(node.ReserveBalance) / (float64(node.TotalSharesIssued) * node.ReserveRatio)
	
	if price < 0.01 { return 0.01 }
	return price
}

// CalculateBuyCost determines the exact cost + whale slippage penalty for a block order.
// Returns total cost in micro-VBV and the slippage percentage.
func (node *EntityMarketNode) CalculateBuyCost(unitsToBuy uint64) (uint64, float64) {
	spotPriceBefore := node.GetSpotPrice()

	// Bancor Formula integration for supply-elastic bonding curves.
	// Cost = ReserveBalance * ((1 + unitsToBuy/TotalSharesIssued)^(1/ReserveRatio) - 1)
	supplyRatio := float64(unitsToBuy) / float64(node.TotalSharesIssued+1)
	costModifier := math.Pow(1.0+supplyRatio, 1.0/node.ReserveRatio) - 1.0
	baseCost := float64(node.ReserveBalance) * costModifier

	// Anti-Whale Slippage Multiplier: Penalize orders that drain high percentages of current supply.
	// PILLAR 2: Quadratic Whale Penalty.
	slippageFactor := 1.0 + math.Pow(supplyRatio, 2.0)*5.0
	finalCost := baseCost * slippageFactor

	effectiveSpotPriceAfter := finalCost / float64(unitsToBuy)
	slippagePercent := ((effectiveSpotPriceAfter - spotPriceBefore) / spotPriceBefore) * 100.0

	return uint64(math.Ceil(finalCost)), slippagePercent
}

// CalculateSellReturn determines the VBV returned for selling a block of shares.
// PILLAR 2: AMM Liquidation logic.
func (node *EntityMarketNode) CalculateSellReturn(unitsToSell uint64) (uint64, float64) {
	if unitsToSell == 0 || unitsToSell >= node.TotalSharesIssued {
		return 0, 0
	}

	spotPriceBefore := node.GetSpotPrice()

	// Bancor Sell Formula:
	// Return = ReserveBalance * (1 - (1 - unitsToSell/TotalSharesIssued)^(1/ReserveRatio))
	supplyRatio := float64(unitsToSell) / float64(node.TotalSharesIssued)
	returnModifier := 1.0 - math.Pow(1.0-supplyRatio, 1.0/node.ReserveRatio)
	baseReturn := float64(node.ReserveBalance) * returnModifier

	// Apply slippage penalty to sell orders as well to prevent exit manipulation.
	slippageFactor := 1.0 - (math.Pow(supplyRatio, 2.0) * 2.0)
	if slippageFactor < 0.1 {
		slippageFactor = 0.1
	} // Max 90% penalty
	finalReturn := baseReturn * slippageFactor

	effectiveSpotPriceAfter := finalReturn / float64(unitsToSell)
	slippagePercent := ((spotPriceBefore - effectiveSpotPriceAfter) / spotPriceBefore) * 100.0

	return uint64(math.Floor(finalReturn)), slippagePercent
}

// CalculateDynamicRumorFee links the Rumor Mill engine directly to entity valuation.
// PILLAR 3: Algorithmic Manipulation Scaling.
func (node *EntityMarketNode) CalculateDynamicRumorFee() uint64 {
	// Market Cap = Current Spot Price * Total Shares Issued
	marketCap := node.GetSpotPrice() * float64(node.TotalSharesIssued)

	// Base Fee floor is 500 VBV. Scaled fee is 2.5% of total asset Market Cap.
	dynamicFee := (500.0 * 1000000.0) + (marketCap * 0.025)

	// Soft cap limit: 50,000 VBV (micro-units)
	maxFee := uint64(50000 * 1000000)
	if uint64(dynamicFee) > maxFee {
		return maxFee
	}

	return uint64(dynamicFee)
}

// handleTradeShares allows players to trade equity in entities.
func (l *Lobby) handleTradeShares(env *Envelope) {
	var data struct {
		EntityID string  `json:"entity_id"` // This can be ClientID or Wallet Address
		Action   string  `json:"action"`
		Amount   float64 `json:"amount"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}

	if data.Amount <= 0 {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Market Error: Trade amount must be positive."}`)})
		return
	}

	// PILLAR 2: Precision Hardening.
	// Force a strict 2-decimal limit on share quantities to prevent floating-point drift and portfolio dust.
	data.Amount = math.Round(data.Amount*100) / 100

	wallet, ok := l.wallets[env.FromID]
	if !ok {
		return
	}

	// PILLAR 5: Performance Hardening.
	// Resolve names and metadata before acquiring the global lock to prevent I/O blocking.
	entityName := l.ResolveEnvoiName(data.EntityID)

	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Resolve target wallet: check active session map first, then leaderboard (NPCs/Offline), then fallback to direct address
	var targetWallet string
	if w, exists := l.wallets[data.EntityID]; exists {
		targetWallet = w
	} else if _, exists := l.leaderboard[strings.ToLower(data.EntityID)]; exists {
		targetWallet = data.EntityID
	} else if strings.HasPrefix(strings.ToLower(data.EntityID), "voi") || strings.HasPrefix(strings.ToLower(data.EntityID), "0x") {
		targetWallet = data.EntityID
	} else {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Market Error: Target entity not found in Arena records."}`)})
		return
	}
	targetWallet = strings.ToLower(targetWallet)

	l.ensurePlayerStatsMapsInitialized(targetWallet)

	// PILLAR 3: Justice Layer - Enforcement Check.
	if time.Now().Before(l.leaderboard[targetWallet].MarketFrozenUntil) {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Market Error: Trading is currently suspended for this entity by order of the Tax Auditor."}`)})
		return
	}

	// PILLAR 2: AMM Integration.
	// Fetch the market node for this entity to apply bonding curve logic (Slippage Enforcement).
	node := l.getOrCreateMarketNodeLocked(targetWallet)
	
	// PILLAR 1: Organizational Yield (Capture pending before adjustment)
	// PILLAR 2: Integer Supremacy (1e12 Scaling)
	stats := l.leaderboard[wallet]
	if stats.LastClaimedYield == nil { stats.LastClaimedYield = make(map[string]uint64) }
	if sharesHeld := stats.Portfolio[targetWallet]; sharesHeld > 0 {
		yieldDelta := node.CumulativeYieldPerShare - stats.LastClaimedYield[targetWallet]
		if yieldDelta > 0 {
			payoutMicro := (yieldDelta * sharesHeld) / 1000000000000
			l.playerBalances[wallet] += payoutMicro
			l.logAdminAuditLocked("DIVIDEND_AUTO_CLAIM", wallet, fmt.Sprintf("Entity: %s, Amount: %d (Trade adjustment)", targetWallet, payoutMicro))
		}
	}
	// Sync claim point to current yield baseline
	stats.LastClaimedYield[targetWallet] = node.CumulativeYieldPerShare

	// Refresh target Standing to ensure current performance/infamy is reflected in the spot price.
	// FIXED: Replace updateLeaderboard call to avoid erroneously incrementing wins (Free Win Exploit).
	tStats := l.leaderboard[targetWallet]
	tStats.Reputation = l.CalculateReputation(tStats)
	l.leaderboard[targetWallet] = tStats

	node.Mu.Lock()
	defer node.Mu.Unlock()

	rumorMultiplier := 1.0
	now := time.Now()
	for _, rumor := range l.rumors {
		if strings.EqualFold(rumor.TargetWallet, targetWallet) && now.Before(rumor.ExpiresAt) {
			rumorMultiplier *= rumor.Strength
		}
	}

	// PILLAR 1: Political Influence - Tax Haven Check.
	isExempt := false
	if stats.EmployerClubID != "" {
		if club, exists := l.clubs[stats.EmployerClubID]; exists {
			if !club.TaxHavenExpiresAt.IsZero() && now.Before(club.TaxHavenExpiresAt) {
				isExempt = true
			}
		}
	}

	// Convert float64 amount to uint64 units (1 share = 100 units)
	unitsToTrade := uint64(data.Amount*100 + 0.5)
	var totalValueMicro uint64
	var slippage float64

	// PILLAR 3: Financial Proof.
	// Define trade details for the on-chain audit trail.
	tradeDetails := map[string]interface{}{
		"action":    data.Action,
		"symbol":    entityName,  // Asset Symbol (Envoi Name)
		"qty":       data.Amount, // Share Quantity
		"spot":      node.GetSpotPrice() * rumorMultiplier,
		"total":     0.0,            // Updated below
		"sector_id": "arena_center", // PILLAR 3: Localized Economic Auditing.
	}

	if stats.Portfolio == nil { // PILLAR 2: Integer Supremacy
		stats.Portfolio = make(map[string]uint64)
	} // This is a direct check on 'stats.Portfolio', not PlayerStats.GetEffectiveMojo

	switch data.Action {
	case "buy":
		totalValueMicro, slippage = node.CalculateBuyCost(unitsToTrade)
		// Apply rumor strength to the base cost
		totalValueMicro = uint64(float64(totalValueMicro) * rumorMultiplier)

		feeMicro := uint64(0)
		if !isExempt {
			// PILLAR 2: Industrial Loop (Protocol Fee).
			// Extract a 1% exchange fee from all trades to fund the Faucet and Regional Governors.
			feeMicro = uint64(float64(totalValueMicro)*0.01 + 0.5)
		}
		netToReserveMicro := totalValueMicro - feeMicro

		if l.playerBalances[wallet] >= totalValueMicro {
			l.playerBalances[wallet] -= totalValueMicro
			currentSharesMicro := stats.Portfolio[targetWallet] // PILLAR 2: Integer Supremacy
			stats.Portfolio[targetWallet] = currentSharesMicro + unitsToTrade // PILLAR 2: Integer Supremacy

			// Commit AMM State Change
			node.TotalSharesIssued += unitsToTrade
			node.ReserveBalance += netToReserveMicro

			// Route protocol fee via Token-Sink Router for audit and distribution
			if l.tokenSinkRouter != nil {
				matrix := RevenueSplitMatrix{FaucetShare: 0.80, ClubShare: 0.0, GovernanceShare: 0.20}
				_ = l.tokenSinkRouter.RouteCriminalTax("SHARE_TRADE_FEE", feeMicro, matrix, 0, "arena_center")
			}

			// PILLAR 2: Ledger Integrity.
			// Physical balance remains unchanged. Scaling is recalculated
			// based on the reduction in virtual liabilities (playerBalances).
			l.applyDynamicScalingLocked()
			l.logAdminAuditLocked("STOCK_BUY", wallet, fmt.Sprintf("Bought %.2f shares of %s", data.Amount, targetWallet))

			// Record buy on-chain if significant
			if totalValueMicro >= 100*1000000 {
				tradeDetails["total"] = float64(totalValueMicro) / 1000000.0
				tradeDetails["slippage"] = slippage
				jsonPayload, _ := json.Marshal(tradeDetails)
				go l.sendNoteTx(fmt.Sprintf("VBT_SHARE_TRADE:%s", string(jsonPayload)))
			}
		} else {
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Insufficient reward balance."}`)})
			return
		}
	case "sell":
		if stats.Portfolio[targetWallet] >= unitsToTrade { // PILLAR 2: Integer Supremacy
			totalValueMicro, slippage = node.CalculateSellReturn(unitsToTrade)
			totalValueMicro = uint64(float64(totalValueMicro) * rumorMultiplier)

			feeMicro := uint64(0)
			if !isExempt {
				// PILLAR 2: Industrial Loop (Protocol Fee).
				feeMicro = uint64(float64(totalValueMicro)*0.01 + 0.5)
			}
			netToPlayerMicro := totalValueMicro - feeMicro
			
			// Route protocol fee via Token-Sink Router
			if l.tokenSinkRouter != nil {
				matrix := RevenueSplitMatrix{FaucetShare: 0.80, ClubShare: 0.0, GovernanceShare: 0.20} // PILLAR 2: Integer Supremacy
				_ = l.tokenSinkRouter.RouteCriminalTax("SHARE_TRADE_FEE", feeMicro, matrix, 0, "arena_center")
			}

			totalValueBase := float64(totalValueMicro) / 1000000.0

			// Check Faucet Liquidity for payout
			if l.faucetBalance < totalValueBase {
				l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Market Illiquid: Payout exceeds Arena capacity."}`)})
				return
			}

			stats.Portfolio[targetWallet] -= unitsToTrade // PILLAR 2: Integer Supremacy

			// Share Dust Cleanup: remove mapping if amount is effectively zero to prevent state bloat
			if stats.Portfolio[targetWallet] == 0 { // PILLAR 2: Integer Supremacy
				delete(stats.Portfolio, targetWallet)
			}

			l.playerBalances[wallet] += netToPlayerMicro

			// Commit AMM State Change
			node.TotalSharesIssued -= unitsToTrade
			node.ReserveBalance -= totalValueMicro // Gross reduction maintains curve integrity

			// PILLAR 2: Ledger Integrity.
			// Physical balance remains unchanged. Scaling is recalculated
			// based on the increase in virtual liabilities (playerBalances).
			l.applyDynamicScalingLocked()
			l.logAdminAuditLocked("STOCK_SELL", wallet, fmt.Sprintf("Sold %.2f shares of %s", data.Amount, targetWallet))

			// PILLAR 3: Financial Proof. Record high-value share sale on-chain for the audit trail.
			if totalValueMicro >= 100*1000000 {
				tradeDetails["total"] = totalValueBase
				tradeDetails["slippage"] = slippage
				jsonPayload, _ := json.Marshal(tradeDetails)
				go l.sendNoteTx(fmt.Sprintf("VBT_SHARE_TRADE:%s", string(jsonPayload)))
			}
		} else {
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Insufficient shares."}`)})
			return
		}
	default:
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Market Error: Invalid action."}`)})
		return
	}

	// REPUTATION SYNC: Ensure the trader's Standing reflects the current world state post-trade.
	stats.Reputation = l.CalculateReputation(stats)

	l.leaderboard[wallet] = stats
	// PILLAR 2: Convert uint64 micro-shares to float64 shares for client-side display
	displayPortfolio := make(map[string]float64)
	for k, v := range stats.Portfolio {
		displayPortfolio[k] = float64(v) / 100.0
	}
	portfolioPayload, _ := json.Marshal(displayPortfolio)
	l.sendToClientLocked(env.FromID, Envelope{Type: "portfolio_update", Payload: portfolioPayload})

	// Trigger Global Sync to update Faucet Balance and Market valuations for all players
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// observeGlobalSentiments aggregates playstyle data to identify meta-trends.
func (l *Lobby) observeGlobalSentiments() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	var totalAgg, totalRisk float64
	ruleCounts := make(map[string]float64)
	count := float64(len(l.leaderboard))
	if count == 0 {
		return
	}

	for _, s := range l.leaderboard {
		totalAgg += s.Playstyle.Aggressiveness
		totalRisk += s.Playstyle.RiskTolerance
		for rule, weight := range s.Playstyle.PreferredRules {
			ruleCounts[rule] += weight
		}
	}

	l.globalSentiment = GlobalSentiment{
		AvgAggressiveness: totalAgg / count,
		AvgRiskTolerance:  totalRisk / count,
		DominantRules:     ruleCounts,
		UpdatedAt:         time.Now(),
	}
	log.Printf("[INTELLIGENCE] Meta-Sentiment Updated. Avg Agg: %.2f\n", l.globalSentiment.AvgAggressiveness)
}
