//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"
)

// ── Persistent Identity System ───────────────────────────────────────────────
// History creates reputation. Reputation creates opportunity.
// Nothing important should disappear.

// IdentityEvent represents a permanent historical record.
type IdentityEvent struct {
	ID        string    `json:"id"`
	Wallet    string    `json:"wallet"`
	Type      string    `json:"type"`      // achievement, failure, milestone, redemption
	Category  string    `json:"category"`  // combat, economy, social, creative, faith
	Title     string    `json:"title"`
	Detail    string    `json:"detail"`
	Timestamp time.Time `json:"timestamp"`
	Impact    int64     `json:"impact"`    // reputation impact (can be negative)
	Permanent bool      `json:"permanent"` // if true, never expires
}

// ReputationProfile holds the full reputation state for a player.
type ReputationProfile struct {
	Wallet          string          `json:"wallet"`
	TotalReputation int64           `json:"total_reputation"`
	CategoryReps    map[string]int64 `json:"category_reps"`
	Achievements    []IdentityEvent `json:"achievements"`
	Failures        []IdentityEvent `json:"failures"`
	CurrentStreak   int             `json:"current_streak"`
	BestStreak      int             `json:"best_streak"`
	Redemptions     int             `json:"redemptions"`
	LastActive      time.Time       `json:"last_active"`
	IdentityScore   uint64          `json:"identity_score"`
	Tier            string          `json:"tier"`
}

// PersistentIdentity manages all identity records.
type PersistentIdentity struct {
	mu       sync.RWMutex
	profiles map[string]*ReputationProfile // wallet -> profile
	events   map[string][]*IdentityEvent    // wallet -> events
}

var persistentIdentity = &PersistentIdentity{
	profiles: make(map[string]*ReputationProfile),
	events:   make(map[string][]*IdentityEvent),
}

// RecordEvent records a permanent identity event.
func (pi *PersistentIdentity) RecordEvent(wallet, eventType, category, title, detail string, impact int64, permanent bool) *IdentityEvent {
	pi.mu.Lock()
	defer pi.mu.Unlock()

	event := &IdentityEvent{
		ID:        fmt.Sprintf("id_%d", time.Now().UnixNano()),
		Wallet:    wallet,
		Type:      eventType,
		Category:  category,
		Title:     title,
		Detail:    detail,
		Timestamp: time.Now(),
		Impact:    impact,
		Permanent: permanent,
	}

	pi.events[wallet] = append(pi.events[wallet], event)

	profile, ok := pi.profiles[wallet]
	if !ok {
		profile = &ReputationProfile{
			Wallet:       wallet,
			CategoryReps: make(map[string]int64),
		}
		pi.profiles[wallet] = profile
	}

	profile.TotalReputation += impact
	profile.CategoryReps[category] += impact
	profile.LastActive = time.Now()

	if eventType == "achievement" || eventType == "milestone" {
		profile.Achievements = append(profile.Achievements, *event)
		profile.CurrentStreak++
		if profile.CurrentStreak > profile.BestStreak {
			profile.BestStreak = profile.CurrentStreak
		}
	} else if eventType == "failure" {
		profile.Failures = append(profile.Failures, *event)
		profile.CurrentStreak = 0
	} else if eventType == "redemption" {
		profile.Redemptions++
		profile.CurrentStreak = 1
	}

	profile.IdentityScore = pi.computeIdentityScore(profile)
	profile.Tier = pi.computeTier(profile.IdentityScore)

	return event
}

// GetProfile returns the reputation profile for a wallet.
func (pi *PersistentIdentity) GetProfile(wallet string) (*ReputationProfile, bool) {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	profile, ok := pi.profiles[wallet]
	return profile, ok
}

// GetEvents returns all identity events for a wallet.
func (pi *PersistentIdentity) GetEvents(wallet string, limit int) []*IdentityEvent {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	events := pi.events[wallet]
	if limit > 0 && len(events) > limit {
		return events[len(events)-limit:]
	}
	return events
}

// GetLeaderboard returns top N players by reputation.
func (pi *PersistentIdentity) GetLeaderboard(n int) []*ReputationProfile {
	pi.mu.RLock()
	defer pi.mu.RUnlock()

	var all []*ReputationProfile
	for _, p := range pi.profiles {
		all = append(all, p)
	}
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].TotalReputation > all[j].TotalReputation
	})
	if len(all) > n {
		return all[:n]
	}
	return all
}

// GetReputationTier returns the reputation tier for a given action.
func (pi *PersistentIdentity) GetReputationTier(wallet string) string {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	profile, ok := pi.profiles[wallet]
	if !ok {
		return "UNKNOWN"
	}
	return profile.Tier
}

// CanLead checks if a player has sufficient reputation to lead.
func (pi *PersistentIdentity) CanLead(wallet string) bool {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	profile, ok := pi.profiles[wallet]
	if !ok {
		return false
	}
	return profile.IdentityScore >= 100
}

// CanInvest checks if a player can invest in entities.
func (pi *PersistentIdentity) CanInvest(wallet string) bool {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	profile, ok := pi.profiles[wallet]
	if !ok {
		return false
	}
	return profile.IdentityScore >= 50
}

func (pi *PersistentIdentity) computeIdentityScore(p *ReputationProfile) uint64 {
	score := uint64(0)
	score += uint64(p.TotalReputation)
	score += uint64(p.BestStreak * 10)
	score += uint64(len(p.Achievements) * 5)
	score += uint64(p.Redemptions * 15)
	return score
}

func (pi *PersistentIdentity) computeTier(score uint64) string {
	switch {
	case score >= 1000:
		return "LEGENDARY"
	case score >= 500:
		return "HEROIC"
	case score >= 250:
		return "ESTABLISHED"
	case score >= 100:
		return "RISING"
	case score >= 50:
		return "EMERGING"
	default:
		return "NEWCOMER"
	}
}

// ── HTTP Handlers ────────────────────────────────────────────────────────────

func (l *Lobby) handleIdentityProfile(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	profile, ok := persistentIdentity.GetProfile(wallet)
	if !ok {
		writeJSON(w, map[string]interface{}{"success": false, "error": "profile not found"})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true, "profile": profile})
}

func (l *Lobby) handleIdentityEvents(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	limit := 50
	events := persistentIdentity.GetEvents(wallet, limit)
	writeJSON(w, map[string]interface{}{"success": true, "events": events})
}

func (l *Lobby) handleIdentityLeaderboard(w http.ResponseWriter, r *http.Request) {
	leaderboard := persistentIdentity.GetLeaderboard(50)
	writeJSON(w, map[string]interface{}{"success": true, "leaderboard": leaderboard})
}

func (l *Lobby) handleIdentityRecord(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type     string `json:"type"`
		Category string `json:"category"`
		Title    string `json:"title"`
		Detail   string `json:"detail"`
		Impact   int64  `json:"impact"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	event := persistentIdentity.RecordEvent(wallet, req.Type, req.Category, req.Title, req.Detail, req.Impact, true)
	writeJSON(w, map[string]interface{}{"success": true, "event": event})
}
