//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"time"
)

// NarrativeService manages NPC interactions, meta-sentiment analysis, and lobby atmosphere.
// PILLAR 5: Stateless Service Design.
type NarrativeService struct{}

// GenerateNPCCommentary picks an NPC to comment on a player's style via chat.
// This function has been moved from lobby_manager.go and market_service.go to
// centralize narrative logic and eliminate code duplication.
func (s *NarrativeService) GenerateNPCCommentary(l *Lobby, clientID string, trigger string) {
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
	avgAgg := global.AvgAggressiveness // FIXED: Corrected access to global.AvgAggressiveness
	l.mutex.RUnlock()

	displayName := l.oracleService.ResolveEnvoiName(l, wallet)
	message := ""
	switch trigger {
	case "LOBBY_ENTRY":
		if risk > avgRisk*1.5 {
			message = fmt.Sprintf("Back for more, %s? Your reckless placements are becoming legendary.", displayName)
		} else if agg > avgAgg*1.4 {
			message = fmt.Sprintf("Make way! %s is here. I can smell the thirst for captures from across the sector.", displayName)
		}
	case "MATCH_START":
		if playerTopRule != "" && playerTopRule == metaRule {
			message = fmt.Sprintf("Attention spectators: %s is a specialist in the current %s meta. A surgical display expected.", template.HTMLEscapeString(displayName), template.HTMLEscapeString(playerTopRule))
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
