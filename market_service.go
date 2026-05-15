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

	l.mutex.Lock()
	defer l.mutex.Unlock()

	wallet, ok := l.wallets[env.FromID]
	if !ok {
		return
	}

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
	for _, rumor := range l.rumors {
		if strings.EqualFold(rumor.TargetWallet, targetWallet) && time.Now().Before(rumor.ExpiresAt) {
			basePrice *= rumor.Strength
		}
	}
	pricePerShare := basePrice
	totalValueMicro := uint64(math.Round(data.Amount * pricePerShare * 1000000.0))
	totalValueBase := float64(totalValueMicro) / 1000000.0

	stats := l.leaderboard[wallet]
	if stats.Portfolio == nil {
		stats.Portfolio = make(map[string]float64)
	}

	if data.Action == "buy" {
		if l.rewards[wallet] >= totalValueMicro {
			l.rewards[wallet] -= totalValueMicro
			currentShares := stats.Portfolio[targetWallet]
			stats.Portfolio[targetWallet] = currentShares + data.Amount

			// Industrial Loop: Investment returns to Faucet
			l.faucetBalance += totalValueBase
			l.applyDynamicScalingLocked()
			l.logAdminAuditLocked("STOCK_BUY", wallet, fmt.Sprintf("Bought %.2f shares of %s", data.Amount, targetWallet))
		} else {
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Insufficient reward balance."}`)})
			return
		}
	} else if data.Action == "sell" {
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

			l.rewards[wallet] += totalValueMicro

			// Industrial Loop: Payout from Faucet
			l.faucetBalance -= totalValueBase
			l.applyDynamicScalingLocked()
			l.logAdminAuditLocked("STOCK_SELL", wallet, fmt.Sprintf("Sold %.2f shares of %s", data.Amount, targetWallet))
		} else {
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Insufficient shares."}`)})
			return
		}
	} else {
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

// generateNPCCommentary picks an NPC to comment on a player's style via chat.
func (l *Lobby) generateNPCCommentary(clientID string, trigger string) {
	l.mutex.RLock()
	wallet, ok := l.wallets[clientID]
	if !ok {
		l.mutex.RUnlock()
		return
	}
	stats, exists := l.leaderboard[wallet]
	global := l.globalSentiment

	if !exists || time.Since(global.UpdatedAt) > 1*time.Hour {
		l.mutex.RUnlock()
		return
	}

	// PILLAR 3: Narrative Snapshot.
	// Identify trends under lock to prevent concurrent map access panics.
	metaRule := ""
	maxMetaWeight := 0.0
	for r, w := range global.DominantRules {
		if w > maxMetaWeight {
			maxMetaWeight = w
			metaRule = r
		}
	}

	playerTopRule := ""
	pMax := 0.0
	for r, w := range stats.Playstyle.PreferredRules {
		if w > pMax {
			pMax = w
			playerTopRule = r
		}
	}

	// Copy values for logic processing outside of the lock
	risk := stats.Playstyle.RiskTolerance
	agg := stats.Playstyle.Aggressiveness
	avgRisk := global.AvgRiskTolerance
	avgAgg := global.AvgAggressiveness
	l.mutex.RUnlock()

	displayName := l.ResolveEnvoiName(wallet)
	message := ""
	if trigger == "LOBBY_ENTRY" {
		if risk > avgRisk*1.5 {
			message = fmt.Sprintf("Back for more, %s? Your reckless placements are becoming legendary.", displayName)
		} else if agg > avgAgg*1.4 {
			message = fmt.Sprintf("Make way! %s is here. I can smell the thirst for captures from across the sector.", displayName)
		}
	} else if trigger == "MATCH_START" {
		if playerTopRule != "" && playerTopRule == metaRule {
			message = fmt.Sprintf("Attention spectators: %s is a specialist in the current %s meta. A surgical display expected.", displayName, playerTopRule)
		} else if agg > avgAgg*1.3 {
			message = fmt.Sprintf("Watch your flanks. %s plays like a predator in the neon deep.", displayName)
		} else if risk > avgRisk*1.3 {
			message = fmt.Sprintf("%s is known for high-infamy gambits. This should be interesting.", displayName)
		} else if playerTopRule != "" && playerTopRule != metaRule {
			message = fmt.Sprintf("%s is sticking to %s. A bold choice against the crowd.", displayName, playerTopRule)
		}
	}

	if message != "" {
		time.Sleep(1 * time.Second)
		payload, _ := json.Marshal(map[string]string{"text": message})
		l.broadcast <- jsonListEnvelope("chat", payload)
	}
}
