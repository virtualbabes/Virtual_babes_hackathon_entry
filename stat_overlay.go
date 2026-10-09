//go:build !js && !wasm

package main

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
)

// ── Stat Overlay System ──────────────────────────────────────────────────────
// The stat overlay intercepts/overlays a power buff across ALL entities.
// Related to 3D world functionality and entity power combinations.
// Every real-world entity has them as base stats.

// EntityType distinguishes entity categories for stat overlay.
type EntityType string

const (
	EntityTypePlayer      EntityType = "player"
	EntityTypeAICitizen   EntityType = "ai_citizen"
	EntityTypeLLMBot      EntityType = "llm_bot"
	EntityTypeRogueBot    EntityType = "rogue_bot"
	EntityTypeChildBot    EntityType = "child_bot"
	EntityTypePet         EntityType = "pet"
	EntityTypeVehicle     EntityType = "vehicle"
)

// StatOverlay represents the power buff applied to an entity in the 3D world.
type StatOverlay struct {
	EntityID      string     `json:"entity_id"`
	EntityType    EntityType `json:"entity_type"`
	Owner         string     `json:"owner"`
	Stats         EntityStats `json:"stats"`
	EffectivePower uint64    `json:"effective_power"`
	DominantStat  string     `json:"dominant_stat"`
	PowerBuff     int        `json:"power_buff"` // signed buff from events/rivalries
	Region        string     `json:"region"`
	IsMature      bool       `json:"is_mature"`
	CanEarnVBV    bool       `json:"can_earn_vbv"`
	CanEarnStats  bool       `json:"can_earn_stats"`
}

// StatOverlayEngine manages stat overlays for all entities.
type StatOverlayEngine struct {
	mu       sync.RWMutex
	overlays map[string]*StatOverlay // entity_id -> overlay
}

var statOverlayEngine = &StatOverlayEngine{
	overlays: make(map[string]*StatOverlay),
}

// ComputeStatOverlay computes the stat overlay for any entity.
func (e *StatOverlayEngine) ComputeStatOverlay(entityID string, entityType EntityType, owner string, stats EntityStats, baseLevel uint64, region string, isMature bool) *StatOverlay {
	effectivePower := ComputeEffectivePowerLevel(baseLevel, stats)
	
	// Determine earning capabilities
	canEarnVBV := true
	canEarnStats := true
	
	switch entityType {
	case EntityTypeChildBot, EntityTypePet:
		if !isMature {
			canEarnVBV = false // immature pets + children bots earn ONLY stats
		}
	}

	overlay := &StatOverlay{
		EntityID:       entityID,
		EntityType:     entityType,
		Owner:          owner,
		Stats:          stats,
		EffectivePower: effectivePower,
		DominantStat:   stats.DominantStat(),
		PowerBuff:      0,
		Region:         region,
		IsMature:       isMature,
		CanEarnVBV:     canEarnVBV,
		CanEarnStats:   canEarnStats,
	}

	e.mu.Lock()
	e.overlays[entityID] = overlay
	e.mu.Unlock()

	return overlay
}

// GetOverlay returns the stat overlay for an entity.
func (e *StatOverlayEngine) GetOverlay(entityID string) (*StatOverlay, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	overlay, ok := e.overlays[entityID]
	return overlay, ok
}

// GetOverlaysByRegion returns all overlays in a region, sorted by power.
func (e *StatOverlayEngine) GetOverlaysByRegion(region string) []*StatOverlay {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var results []*StatOverlay
	for _, o := range e.overlays {
		if o.Region == region {
			results = append(results, o)
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].EffectivePower > results[j].EffectivePower
	})
	return results
}

// GetOverlaysByOwner returns all overlays owned by a wallet.
func (e *StatOverlayEngine) GetOverlaysByOwner(owner string) []*StatOverlay {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var results []*StatOverlay
	for _, o := range e.overlays {
		if o.Owner == owner {
			results = append(results, o)
		}
	}
	return results
}

// ApplyPowerBuff applies a signed power buff to an entity.
func (e *StatOverlayEngine) ApplyPowerBuff(entityID string, buff int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	overlay, ok := e.overlays[entityID]
	if !ok {
		return fmt.Errorf("entity not found")
	}
	overlay.PowerBuff += buff
	return nil
}

// GetHighPowerEntities returns the top N entities by effective power.
func (e *StatOverlayEngine) GetHighPowerEntities(n int) []*StatOverlay {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var all []*StatOverlay
	for _, o := range e.overlays {
		all = append(all, o)
	}
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].EffectivePower > all[j].EffectivePower
	})
	if len(all) > n {
		return all[:n]
	}
	return all
}

// RecomputeAll recomputes all stat overlays (called after events).
func (e *StatOverlayEngine) RecomputeAll(l *Lobby) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, overlay := range e.overlays {
		overlay.EffectivePower = ComputeEffectivePowerLevel(0, overlay.Stats)
		overlay.DominantStat = overlay.Stats.DominantStat()
	}
}

// ── HTTP Handlers ────────────────────────────────────────────────────────────

func (l *Lobby) handleStatOverlayGet(w http.ResponseWriter, r *http.Request) {
	entityID := r.URL.Query().Get("entity_id")
	if entityID == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "entity_id required"})
		return
	}
	overlay, ok := statOverlayEngine.GetOverlay(entityID)
	if !ok {
		writeJSON(w, map[string]interface{}{"success": false, "error": "not found"})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true, "overlay": overlay})
}

func (l *Lobby) handleStatOverlayRegion(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	if region == "" {
		region = "Base"
	}
	overlays := statOverlayEngine.GetOverlaysByRegion(region)
	writeJSON(w, map[string]interface{}{"success": true, "overlays": overlays})
}

func (l *Lobby) handleStatOverlayOwner(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	overlays := statOverlayEngine.GetOverlaysByOwner(wallet)
	writeJSON(w, map[string]interface{}{"success": true, "overlays": overlays})
}

func (l *Lobby) handleStatOverlayLeaderboard(w http.ResponseWriter, r *http.Request) {
	overlays := statOverlayEngine.GetHighPowerEntities(50)
	writeJSON(w, map[string]interface{}{"success": true, "leaderboard": overlays})
}
