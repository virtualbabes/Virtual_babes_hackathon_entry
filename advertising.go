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

// ── Advertising System ───────────────────────────────────────────────────────
// Advertising is not merely commerce — it's discovery. Sponsored content,
// infrastructure modules, educational content, community creations.
// Everything should naturally integrate into the civilization rather than interrupt it.

// AdType represents the type of advertisement.
type AdType string

const (
	AdTypeSponsored   AdType = "sponsored"
	AdTypeInfrastructure AdType = "infrastructure"
	AdTypeEducational AdType = "educational"
	AdTypeCommunity   AdType = "community"
	AdTypeCreator     AdType = "creator"
)

// AdStatus represents the status of an ad.
type AdStatus string

const (
	AdStatusDraft     AdStatus = "draft"
	AdStatusActive    AdStatus = "active"
	AdStatusPaused    AdStatus = "paused"
	AdStatusCompleted AdStatus = "completed"
)

// Advertisement represents a sponsored content advertisement.
type Advertisement struct {
	ID            string    `json:"id"`
	Advertiser    string    `json:"advertiser"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	AdType        AdType    `json:"ad_type"`
	Status        AdStatus  `json:"status"`
	BudgetMicro   uint64    `json:"budget_micro"`
	SpentMicro    uint64    `json:"spent_micro"`
	CPMMicro      uint64    `json:"cpm_micro"` // cost per mille (thousand impressions)
	Impressions   uint64    `json:"impressions"`
	Clicks        uint64    `json:"clicks"`
	TargetRegions []string  `json:"target_regions"`
	TargetTiers   []string  `json:"target_tiers"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}

// AdImpression tracks an ad impression.
type AdImpression struct {
	AdID      string    `json:"ad_id"`
	Viewer    string    `json:"viewer"`
	Timestamp time.Time `json:"timestamp"`
	Region    string    `json:"region"`
}

// AdClick tracks an ad click.
type AdClick struct {
	AdID      string    `json:"ad_id"`
	Clicker   string    `json:"clicker"`
	Timestamp time.Time `json:"timestamp"`
	Region    string    `json:"region"`
}

// AdEngine manages advertising.
type AdEngine struct {
	mu         sync.RWMutex
	ads        map[string]*Advertisement
	impressions map[string][]*AdImpression
	clicks     map[string][]*AdClick
}

var adEngine = &AdEngine{
	ads:        make(map[string]*Advertisement),
	impressions: make(map[string][]*AdImpression),
	clicks:     make(map[string][]*AdClick),
}

// CreateAd creates a new advertisement.
func (ae *AdEngine) CreateAd(advertiser, title, description string, adType AdType, budgetMicro, cpmMicro uint64, targetRegions, targetTiers []string, expiresAt time.Time) *Advertisement {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	ad := &Advertisement{
		ID:            fmt.Sprintf("ad_%d", time.Now().UnixNano()),
		Advertiser:    advertiser,
		Title:         title,
		Description:   description,
		AdType:        adType,
		Status:        AdStatusDraft,
		BudgetMicro:   budgetMicro,
		SpentMicro:    0,
		CPMMicro:      cpmMicro,
		TargetRegions: targetRegions,
		TargetTiers:   targetTiers,
		CreatedAt:     time.Now(),
		ExpiresAt:     expiresAt,
	}
	ae.ads[ad.ID] = ad
	return ad
}

// ActivateAd activates an ad.
func (ae *AdEngine) ActivateAd(adID string) error {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	ad, ok := ae.ads[adID]
	if !ok {
		return fmt.Errorf("ad not found")
	}
	ad.Status = AdStatusActive
	return nil
}

// PauseAd pauses an ad.
func (ae *AdEngine) PauseAd(adID string) error {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	ad, ok := ae.ads[adID]
	if !ok {
		return fmt.Errorf("ad not found")
	}
	ad.Status = AdStatusPaused
	return nil
}

// RecordImpression records an ad impression.
func (ae *AdEngine) RecordImpression(adID, viewer, region string) error {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	ad, ok := ae.ads[adID]
	if !ok || ad.Status != AdStatusActive {
		return fmt.Errorf("ad not active")
	}

	ad.Impressions++
	ad.SpentMicro += ad.CPMMicro / 1000 // CPM / 1000 = cost per impression
	ae.impressions[adID] = append(ae.impressions[adID], &AdImpression{
		AdID:      adID,
		Viewer:    viewer,
		Timestamp: time.Now(),
		Region:    region,
	})

	return nil
}

// RecordClick records an ad click.
func (ae *AdEngine) RecordClick(adID, clicker, region string) error {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	ad, ok := ae.ads[adID]
	if !ok || ad.Status != AdStatusActive {
		return fmt.Errorf("ad not active")
	}

	ad.Clicks++
	ae.clicks[adID] = append(ae.clicks[adID], &AdClick{
		AdID:      adID,
		Clicker:   clicker,
		Timestamp: time.Now(),
		Region:    region,
	})

	return nil
}

// GetAds returns all ads.
func (ae *AdEngine) GetAds(status AdStatus) []*Advertisement {
	ae.mu.RLock()
	defer ae.mu.RUnlock()

	var results []*Advertisement
	for _, a := range ae.ads {
		if status == "" || a.Status == status {
			results = append(results, a)
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})
	return results
}

// GetAdsByAdvertiser returns ads by an advertiser.
func (ae *AdEngine) GetAdsByAdvertiser(advertiser string) []*Advertisement {
	ae.mu.RLock()
	defer ae.mu.RUnlock()

	var results []*Advertisement
	for _, a := range ae.ads {
		if a.Advertiser == advertiser {
			results = append(results, a)
		}
	}
	return results
}

// GetActiveAdsForRegion returns active ads targeting a region.
func (ae *AdEngine) GetActiveAdsForRegion(region string) []*Advertisement {
	ae.mu.RLock()
	defer ae.mu.RUnlock()

	var results []*Advertisement
	for _, a := range ae.ads {
		if a.Status == AdStatusActive {
			for _, r := range a.TargetRegions {
				if r == region || r == "all" {
					results = append(results, a)
					break
				}
			}
		}
	}
	return results
}

// GetAdStats returns stats for an ad.
func (ae *AdEngine) GetAdStats(adID string) map[string]interface{} {
	ae.mu.RLock()
	defer ae.mu.RUnlock()

	ad, ok := ae.ads[adID]
	if !ok {
		return nil
	}

	return map[string]interface{}{
		"impressions": ad.Impressions,
		"clicks":      ad.Clicks,
		"spent_micro": ad.SpentMicro,
		"ctr":         float64(ad.Clicks) / float64(ad.Impressions) * 100,
	}
}

// ── HTTP Handlers ────────────────────────────────────────────────────────────

func (l *Lobby) handleAdCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title         string   `json:"title"`
		Description   string   `json:"description"`
		AdType        string   `json:"ad_type"`
		BudgetMicro   uint64   `json:"budget_micro"`
		CPMMicro      uint64   `json:"cpm_micro"`
		TargetRegions []string `json:"target_regions"`
		TargetTiers   []string `json:"target_tiers"`
		ExpiresAt     string   `json:"expires_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	expiresAt, _ := time.Parse("2006-01-02", req.ExpiresAt)
	ad := adEngine.CreateAd(wallet, req.Title, req.Description, AdType(req.AdType), req.BudgetMicro, req.CPMMicro, req.TargetRegions, req.TargetTiers, expiresAt)
	writeJSON(w, map[string]interface{}{"success": true, "ad": ad})
}

func (l *Lobby) handleAdActivate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AdID string `json:"ad_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	if err := adEngine.ActivateAd(req.AdID); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true})
}

func (l *Lobby) handleAdPause(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AdID string `json:"ad_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	if err := adEngine.PauseAd(req.AdID); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true})
}

func (l *Lobby) handleAdImpression(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AdID   string `json:"ad_id"`
		Region string `json:"region"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if err := adEngine.RecordImpression(req.AdID, wallet, req.Region); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true})
}

func (l *Lobby) handleAdClick(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AdID   string `json:"ad_id"`
		Region string `json:"region"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if err := adEngine.RecordClick(req.AdID, wallet, req.Region); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true})
}

func (l *Lobby) handleAds(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	ads := adEngine.GetAds(AdStatus(status))
	writeJSON(w, map[string]interface{}{"success": true, "ads": ads})
}

func (l *Lobby) handleAdsByAdvertiser(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	ads := adEngine.GetAdsByAdvertiser(wallet)
	writeJSON(w, map[string]interface{}{"success": true, "ads": ads})
}

func (l *Lobby) handleAdsForRegion(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	ads := adEngine.GetActiveAdsForRegion(region)
	writeJSON(w, map[string]interface{}{"success": true, "ads": ads})
}

func (l *Lobby) handleAdStats(w http.ResponseWriter, r *http.Request) {
	adID := r.URL.Query().Get("ad_id")
	stats := adEngine.GetAdStats(adID)
	writeJSON(w, map[string]interface{}{"success": true, "stats": stats})
}
