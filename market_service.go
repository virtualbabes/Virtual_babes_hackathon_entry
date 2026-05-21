//go:build !js || !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"
)

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

	// REAL-TIME PRICING: Recalculate reputation to ensure Marketability Multipliers (Aggressiveness/Risk) are reflected.
	targetStats := l.leaderboard[targetWallet]
	targetStats.Reputation = l.CalculateReputation(targetStats)
	l.leaderboard[targetWallet] = targetStats

	basePrice := float64((targetStats.Wins * 10) + int(float64(targetStats.Reputation)/2.0) + 100.0)

	// PILLAR 2: Temporal Consistency.
	now := time.Now()
	for _, rumor := range l.rumors {
		if strings.EqualFold(rumor.TargetWallet, targetWallet) && now.Before(rumor.ExpiresAt) {
			basePrice *= rumor.Strength
		}
	}
	pricePerShare := basePrice

	// Convert to micro-units using rounded amount for absolute ledger integrity
	// PILLAR 3: Economic Precision. Use micro-unit integer math for total value.
	priceMicro := uint64(pricePerShare*1000000 + 0.5)
	amountInt := uint64(data.Amount*100 + 0.5)
	totalValueMicro := (priceMicro*amountInt + 50) / 100
	totalValueBase := float64(totalValueMicro) / 1000000.0

	// PILLAR 3: Financial Proof.
	// Define trade details for the on-chain audit trail.
	tradeDetails := map[string]interface{}{
		"action":    data.Action,
		"symbol":    entityName,  // Asset Symbol (Envoi Name)
		"qty":       data.Amount, // Share Quantity
		"price":     pricePerShare,
		"total":     totalValueBase,
		"sector_id": "arena_center", // PILLAR 3: Localized Economic Auditing.
	}

	stats := l.leaderboard[wallet]
	if stats.Portfolio == nil {
		stats.Portfolio = make(map[string]float64)
	}

	switch data.Action {
	case "buy":
		if l.playerBalances[wallet] >= totalValueMicro {
			l.playerBalances[wallet] -= totalValueMicro
			currentShares := stats.Portfolio[targetWallet]
			stats.Portfolio[targetWallet] = currentShares + data.Amount

			// PILLAR 2: Ledger Integrity.
			// Physical balance remains unchanged. Scaling is recalculated
			// based on the reduction in virtual liabilities (playerBalances).
			l.applyDynamicScalingLocked()
			l.logAdminAuditLocked("STOCK_BUY", wallet, fmt.Sprintf("Bought %.2f shares of %s", data.Amount, targetWallet))

			// Record high-value buy on-chain for the audit trail
			if totalValueBase >= 100.0 {
				go func(td interface{}) {
					jsonPayload, _ := json.Marshal(td)
					l.sendNoteTx(fmt.Sprintf("VBT_SHARE_TRADE:%s", string(jsonPayload)))
				}(tradeDetails)
			}
		} else {
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Insufficient reward balance."}`)})
			return
		}
	case "sell":
		if stats.Portfolio[targetWallet] >= data.Amount {
			// Check Faucet Liquidity for payout
			if l.faucetBalance < totalValueBase {
				l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Market Illiquid: Payout exceeds Arena capacity."}`)})
				return
			}

			stats.Portfolio[targetWallet] -= data.Amount

			// Share Dust Cleanup: remove mapping if amount is effectively zero to prevent state bloat
			if stats.Portfolio[targetWallet] < 1e-9 {
				delete(stats.Portfolio, targetWallet)
			}

			l.playerBalances[wallet] += totalValueMicro

			// PILLAR 2: Ledger Integrity.
			// Physical balance remains unchanged. Scaling is recalculated
			// based on the increase in virtual liabilities (playerBalances).
			l.applyDynamicScalingLocked()
			l.logAdminAuditLocked("STOCK_SELL", wallet, fmt.Sprintf("Sold %.2f shares of %s", data.Amount, targetWallet))

			// PILLAR 3: Financial Proof. Record high-value share sale on-chain for the audit trail.
			if totalValueBase >= 100.0 {
				go func(td interface{}) {
					jsonPayload, _ := json.Marshal(td)
					l.sendNoteTx(fmt.Sprintf("VBT_SHARE_TRADE:%s", string(jsonPayload)))
				}(tradeDetails)
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
	portfolioPayload, _ := json.Marshal(stats.Portfolio)
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
