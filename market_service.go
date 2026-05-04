package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// handleTradeShares allows players to trade equity in entities.
func (l *Lobby) handleTradeShares(env *Envelope) {
	var data struct {
		EntityID string  `json:"entity_id"`
		Action   string  `json:"action"`
		Amount   float64 `json:"amount"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil { return }

	l.mutex.Lock()
	defer l.mutex.Unlock()

	wallet, ok := l.wallets[env.FromID]
	if !ok { return }

	targetWallet, targetExists := l.wallets[data.EntityID]
	if !targetExists { return }
	targetStats := l.leaderboard[targetWallet]

	basePrice := float64((targetStats.Wins * 10) + (targetStats.Reputation / 2) + 100)
	for _, rumor := range l.rumors {
		if rumor.TargetWallet == targetWallet && time.Now().Before(rumor.ExpiresAt) {
			basePrice *= rumor.Strength
		}
	}
	pricePerShare := basePrice
	totalValueMicro := uint64(data.Amount * pricePerShare * 1000000)

	stats := l.leaderboard[wallet]
	if stats.Portfolio == nil { stats.Portfolio = make(map[string]float64) }

	if data.Action == "buy" {
		if l.rewards[wallet] >= totalValueMicro {
			l.rewards[wallet] -= totalValueMicro
			stats.Portfolio[data.EntityID] += data.Amount
		} else {
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Insufficient reward balance."}`)})
			return
		}
	} else if data.Action == "sell" {
		if stats.Portfolio[data.EntityID] >= data.Amount {
			stats.Portfolio[data.EntityID] -= data.Amount
			l.rewards[wallet] += totalValueMicro
		} else {
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Insufficient shares."}`)})
			return
		}
	}

	l.leaderboard[wallet] = stats
	portfolioPayload, _ := json.Marshal(stats.Portfolio)
	l.sendToClient(env.FromID, Envelope{Type: "portfolio_update", Payload: portfolioPayload})
}

// observeGlobalSentiments aggregates playstyle data to identify meta-trends.
func (l *Lobby) observeGlobalSentiments() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	var totalAgg, totalRisk float64
	ruleCounts := make(map[string]float64)
	count := float64(len(l.leaderboard))
	if count == 0 { return }

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
	stats, exists := l.leaderboard[wallet]
	global := l.globalSentiment
	l.mutex.RUnlock()

	if !ok || !exists || time.Since(global.UpdatedAt) > 1*time.Hour { return }

	message := ""
	if trigger == "LOBBY_ENTRY" {
		if stats.Playstyle.RiskTolerance > global.AvgRiskTolerance*1.5 {
			message = fmt.Sprintf("Back for more, %s? Your reckless placements are becoming legendary.", clientID)
		}
	}

	if message != "" {
		time.Sleep(1 * time.Second)
		payload, _ := json.Marshal(map[string]string{"text": message})
		l.broadcast <- jsonListEnvelope("chat", payload)
	}
}