//go:build !js && !wasm

package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

// achievementDefinition holds the display metadata for each trophy type.
type achievementDefinition struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Rarity      string `json:"rarity"` // common, uncommon, rare, legendary
	Category    string `json:"category"`
}

// allAchievements returns the canonical list of known achievement definitions.
func allAchievements() []achievementDefinition {
	return []achievementDefinition{
		// Economy / Industrial
		{"TREASURY_RECOVERY", "Treasury Recovery", "Recovered your club's treasury to 50% after a crash.", "rare", "economy"},
		{"MOJO_SURGE", "Mojo Surge", "Gained 100 Mojo points within a 24-hour window.", "uncommon", "club"},
		{"ECOSYSTEM_GUARDIAN", "Ecosystem Guardian", "The Arena's Industrial Loop recovered 10,000 $VBV in taxes.", "legendary", "system"},

		// Criminality
		{"FIRST_HEIST", "First Blood", "Completed your first heist.", "common", "criminality"},
		{"BOUNTY_HUNTER", "Bounty Hunter", "Captured 3 unique outlaws (Wanted ≥ 15).", "rare", "criminality"},
		{"HEIST_SABOTEUR", "Heist Saboteur", "Jammed alarm systems in 3 separate heists.", "uncommon", "criminality"},
		{"CORPORATE_ESPIONAGE", "Corporate Espionage", "Audited 5 rival clubs (non-affiliated).", "rare", "intelligence"},

		// Social / Economy
		{"PHILANTHROPIST", "Philanthropist", "Made a single donation of ≥ 1,000 $VBV (25% chance).", "legendary", "social"},
		{"PATRON_OF_THE_ARTS", "Patron of the Arts", "Donated a cumulative 5,000 $VBV.", "rare", "social"},

		// Career
		{"CAREER_START", "Career Start", "Accepted your first employment contract.", "common", "career"},
		{"EXECUTIVE_PAY", "Executive Pay", "Secured an employment contract paying ≥ 500 $VBV per cycle.", "uncommon", "career"},

		// Combat / Arena
		{"PERFECT_GAME", "Perfect Game", "Won a match without taking damage.", "rare", "combat"},
		{"ARENA_LEGEND", "Arena Legend", "Reached 100 wins in the arena.", "legendary", "combat"},

		// Club / Governance
		{"GOVERNOR", "Governor", "Rose to regional governor status (top club by Mojo).", "rare", "governance"},
		{"ART_COLLECTOR", "Art Collector", "Won an auction at the Art Gallery.", "uncommon", "auction"},

		// Justice
		{"REHABILITATED", "Rehabilitated", "Had your Wanted level cleared via the Courthouse system.", "rare", "justice"},
	}
}

// handleGetAchievements returns a player's unlocked achievements with full metadata.
func (l *Lobby) handleGetAchievements(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	wallet := strings.ToLower(r.URL.Query().Get("wallet"))
	if wallet == "" {
		http.Error(w, "wallet query parameter required", http.StatusBadRequest)
		return
	}

	l.mutex.RLock()
	stats, exists := l.leaderboard[wallet]
	l.mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")

	if !exists {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"wallet":      wallet,
			"unlocked":    []string{},
			"definitions": allAchievements(),
			"progress":    map[string]int{},
		})
		return
	}

	defs := allAchievements()
	unslicedProgress := make(map[string]int)
	for _, def := range defs {
		unslicedProgress[def.ID] = 0
	}

	// Progress tracking: some achievements require incremental data.
	if stats.HeistAlarmsJammerCount > 0 {
		unslicedProgress["HEIST_SABOTEUR"] = stats.HeistAlarmsJammerCount
	}
	if len(stats.CapturedOutlaws) > 0 {
		unslicedProgress["BOUNTY_HUNTER"] = len(stats.CapturedOutlaws)
	}

	result := struct {
		Wallet      string              `json:"wallet"`
		Unlocked    []string            `json:"unlocked"`
		Definitions []achievementDefinition `json:"definitions"`
		Progress    map[string]int      `json:"progress"`
		TotalWon    int                 `json:"total_won"`
	}{
		Wallet:      wallet,
		Unlocked:    stats.Achievements,
		Definitions: defs,
		Progress:    unslicedProgress,
		TotalWon:    len(stats.Achievements),
	}

	json.NewEncoder(w).Encode(result)
}

// handleGetAchievementStats returns a summary of all players' achievement progress for leaderboards.
func (l *Lobby) handleGetAchievementStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	l.mutex.RLock()
	type playerAch struct {
		Wallet   string `json:"wallet"`
		Wins     int    `json:"wins"`
		Achieved int    `json:"achieved_count"`
		Rarity   string `json:"rarity_score"`
	}
	var list []playerAch
	for w, s := range l.leaderboard {
		rareScore := 0
		for _, id := range s.Achievements {
			for _, def := range allAchievements() {
				if def.ID == id && (def.Rarity == "legendary" || def.Rarity == "rare") {
					rareScore++
					break
				}
			}
		}
		list = append(list, playerAch{
			Wallet:   w,
			Wins:     s.Wins,
			Achieved: len(s.Achievements),
			Rarity:   rareScore,
		})
	}
	l.mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// handleUnlockAchievement is the POST endpoint for unlocking achievements via the AchievementService.
func (l *Lobby) handleUnlockAchievement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Wallet string `json:"wallet"`
		AchievementID string `json:"achievement_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.Wallet = strings.ToLower(req.Wallet)
	if req.Wallet == "" {
		http.Error(w, "wallet required", http.StatusBadRequest)
		return
	}
	if req.AchievementID == "" {
		http.Error(w, "achievement_id required", http.StatusBadRequest)
		return
	}

	// Validate achievement ID against canonical definitions
	valid := false
	for _, def := range allAchievements() {
		if def.ID == req.AchievementID {
			valid = true
			break
		}
	}
	if !valid {
		http.Error(w, "unknown achievement", http.StatusBadRequest)
		return
	}

	l.achievementService.UnlockAchievement(l, req.Wallet, req.AchievementID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":       "unlocked",
		"wallet":       req.Wallet,
		"achievement_id": req.AchievementID,
	})
}