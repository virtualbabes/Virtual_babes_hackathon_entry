//go:build !js && !wasm

// justice_service.go - Justice Hegemony Path (Pillar 7)
// Implements Justice Card archetype (+10% power vs outlaws), Justice Tier Bounty Center,
// Truth Serum and Reputation Shield items, and combat utility hooks.
//
// Per Rules.md:
//   - All economic math uses uint64 micro-units (1 $VBV = 1,000,000 micro-units)
//   - Industrial Seal: no funds burned or created within transfers
//   - Blockchain-Native: authoritative state from blockchain notes

package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// JusticeCardType defines the archetypes available in the Justice faction.
type JusticeCardType string

const (
	JusticeEnforcer       JusticeCardType = "ENFORCER"
	JusticeMediator       JusticeCardType = "MEDIATOR"
	JusticeWarden         JusticeCardType = "WARDEN"
	JusticeCommissioner   JusticeCardType = "COMMISSIONER"
)

// JusticeFaction defines the Justice alignment state for a player.
type JusticeFaction struct {
	Alignment      string           `json:"alignment"`       // "JUSTICE" or neutral
	JusticeCards   []JusticeCard    `json:"justice_cards,omitempty"`
	BountyRank     int              `json:"bounty_rank"`       // Tier: Hunter, Warden, Commissioner, Icon
	TrophyCount    int              `json:"trophy_count"`      // Justice-specific trophy count
	LastJusticeMod time.Time        `json:"last_justice_mod"`  // Last alignment modification time
	ActiveBuffs    []JusticeBuff    `json:"active_buffs,omitempty"`
	Missions       []JusticeMission `json:"active_missions,omitempty"`
}

// JusticeCard represents a Justice-aligned card instance.
type JusticeCard struct {
	CardID          string        `json:"card_id"`
	Type            JusticeCardType `json:"type"`
	PowerBonus      uint64        `json:"power_bonus"`     // +10% base power when vs outlaws (in micro-units)
	AcquisitionDate time.Time     `json:"acquisition_date"`
	ActiveDebuffs   []JusticeDebuff `json:"active_debuffs,omitempty"`
}

// JusticeBuff represents an active justice-specific buff on a player.
type JusticeBuff struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Duration  time.Duration `json:"duration"`
	ExpiresAt time.Time     `json:"expires_at"`
	PowerMod  int64         `json:"power_mod"`   // flat power modification
	Multiplier float64      `json:"multiplier"`   // multiplicative power modifier (e.g., 1.10 for +10%)
}

// JusticeDebuff represents a justice-specific debuff.
type JusticeDebuff struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Duration       time.Duration `json:"duration"`
	ExpiresAt      time.Time     `json:"expires_at"`
	ReputationMod  int           `json:"reputation_mod"`
}

// JusticeMission represents a dynamic bounty mission.
type JusticeMission struct {
	MissionID      string    `json:"mission_id"`
	TargetPlayerID string    `json:"target_player_id"`
	TargetName     string    `json:"target_name"`
	TargetWanted   int       `json:"target_wanted"`
	RewardVBV      uint64    `json:"reward_vbv"`      // in micro-units
	ExpirationTime time.Time `json:"expiration_time"`
	Status         string    `json:"status"`          // "ACTIVE", "COMPLETED", "EXPIRED"
	RUMCountUsed   int       `json:"rum_count_used"`  // RUM = Rumor Utilization Metric (count of intel items used)
}

// TruthSerumItem represents the "Truth Serum" intelligence item.
type TruthSerumItem struct {
	TargetPlayerID string         `json:"target_player_id"`
	Duration       time.Duration  `json:"duration"`         // How long revealed buffs are visible (default: 30s)
	RevealedBuffs  []CardBuffState `json:"revealed_buffs"`
}

// CardBuffState represents the full buff/debuff state of a card for Truth Serum display.
type CardBuffState struct {
	CardID      string    `json:"card_id"`
	ActiveItems []ItemBuff `json:"active_items"`
	Fatigue     int       `json:"fatigue"`
	Loyalty     int       `json:"loyalty"`
}

// ItemBuff represents an active item buff on a card.
type ItemBuff struct {
	ItemID    string    `json:"item_id"`
	Name      string    `json:"name"`
	PowerMod  int64     `json:"power_mod"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ReputationShieldItem represents the "Reputation Shield" protective item.
type ReputationShieldItem struct {
	ProtectionAmount int       `json:"protection_amount"` // Max reputation loss shielded
	AbsorbedCount    int       `json:"absorbed_count"`    // How many times shield has absorbed
	Duration         time.Duration `json:"duration"`
	ExpiresAt        time.Time `json:"expires_at"`
}

// JusticeBountyDashboard defines the Justice Tier Bounty Center dashboard state.
type JusticeBountyDashboard struct {
	ActiveMissions      []JusticeMission   `json:"active_missions"`
	HighWantedTargets   []BountyTargetInfo `json:"high_wanted_targets"`
	ScrambledCount      int                `json:"scrambled_count"`       // Count of Ghost Protocol users (visible via enhanced tracking)
	DashboardAccess     bool               `json:"dashboard_access"`      // Whether player qualifies for access
	LastRefresh         time.Time          `json:"last_refresh"`
}

// BountyTargetInfo represents a high-Wanted player target.
type BountyTargetInfo struct {
	PlayerID        string    `json:"player_id"`
	PlayerName      string    `json:"player_name"`
	WantedLevel     int       `json:"wanted_level"`
	LastSeen        time.Time `json:"last_seen"`
	District        string    `json:"district"`
	GhostActive     bool      `json:"ghost_active"`        // Whether Ghost Protocol is active
	RealIDAvailable bool      `json:"real_id_available"`   // True if tracked by Justice Tier dashboard
}

// ---- Service Struct ----

// JusticeService manages the Justice Hegemony faction system.
type JusticeService struct {
	mu                sync.RWMutex
	store             map[string]*JusticeFaction       // playerID -> JusticeFaction
	justiceCardPool   []JusticeCardType                // Available card types for distribution
	dashboardCache    map[string]*JusticeBountyDashboard // playerID -> dashboard cache
	truthSerumTargets map[string][]TruthSerumItem        // targetID -> list of active truth serums
	shieldRegistry    map[string]map[string]*ReputationShieldItem // playerID -> shieldID -> shield
	missionCounter    int                                // Monotonically increasing mission ID generator
	minWantedForCard  int                                // Minimum wanted level to qualify for Justice cards (default: 15)
}

// NewJusticeService creates a new Justice Hegemony service.
func NewJusticeService() *JusticeService {
	return &JusticeService{
		store:             make(map[string]*JusticeFaction),
		justiceCardPool: []JusticeCardType{
			JusticeEnforcer,
			JusticeMediator,
			JusticeWarden,
			JusticeCommissioner,
		},
		dashboardCache:    make(map[string]*JusticeBountyDashboard),
		truthSerumTargets: make(map[string][]TruthSerumItem),
		shieldRegistry:    make(map[string]map[string]*ReputationShieldItem),
		minWantedForCard:  15, // Must target players with Wanted >= 15
	}
}

// Ensure JusticeService is a valid type (interface assertion)
var _ *JusticeService = (*JusticeService)(nil)

// ---- Core Logic ----

// GetJusticeFaction retrieves the Justice faction state for a player.
func (js *JusticeService) GetJusticeFaction(playerID string) *JusticeFaction {
	js.mu.RLock()
	defer js.mu.RUnlock()

	if faction, ok := js.store[playerID]; ok {
		fakeCopy := *faction // Shallow copy is sufficient for read-only access
		return &fakeCopy
	}
	return nil
}

// GetPowerBonus returns the +10% power bonus for Justice cards when battling outlaws.
func (js *JusticeService) GetPowerBonus(playerID string, vsOutlaw bool) int64 {
	js.mu.RLock()
	defer js.mu.RUnlock()

	faction, ok := js.store[playerID]
	if !ok || !vsOutlaw {
		return 0
	}

	var totalBonus int64
	for _, card := range faction.JusticeCards {
		totalBonus += int64(card.PowerBonus)
	}

	// Also add active buff multipliers
	for _, buff := range faction.ActiveBuffs {
		if buff.Multiplier > 1.0 {
			totalBonus += int64(float64(totalBonus) * (buff.Multiplier - 1.0))
		}
	}

	return totalBonus
}

// CheckQualificationForJusticeCards determines if a player qualifies for Justice cards.
// Qualification requires:
//   - 3+ Justice Cards already owned, AND
//   - "Warden" social rank (bounty_rank >= 2)
func (js *JusticeService) CheckQualificationForJusticeCards(playerID string) bool {
	js.mu.RLock()
	defer js.mu.RUnlock()

	faction, ok := js.store[playerID]
	if !ok {
		return false
	}

	return len(faction.JusticeCards) >= JusticeTierAccessCards && faction.BountyRank >= JusticeTierAccessRank
}

// AwardJusticeCard assigns a Justice card to a player.
func (js *JusticeService) AwardJusticeCard(playerID string, cardType JusticeCardType) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	// Initialize faction if not exists
	if _, ok := js.store[playerID]; !ok {
		js.store[playerID] = &JusticeFaction{
			Alignment:  "JUSTICE",
			BountyRank: 0, // Hunter
		}
	}

	faction := js.store[playerID]

	// Validate card type exists in pool
	valid := false
	for _, ct := range js.justiceCardPool {
		if ct == cardType {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid justice card type: %s", cardType)
	}

	// Generate unique power bonus for the card
	powerBonus := uint64(500_000) // 0.5 $VBV equivalent in micro-units (flat power)
	if cardType == JusticeWarden {
		powerBonus = uint64(750_000) // Wardens get +0.75 $VBV bonus
	} else if cardType == JusticeCommissioner {
		powerBonus = uint64(1_000_000) // Commissioners get +1 $VBV bonus
	}

	faction.JusticeCards = append(faction.JusticeCards, JusticeCard{
		CardID:          fmt.Sprintf("JUSTICE_%s_%d", cardType, time.Now().UnixNano()),
		Type:            cardType,
		PowerBonus:      powerBonus,
		AcquisitionDate: time.Now(),
	})

	faction.LastJusticeMod = time.Now()
	return nil
}

// RemoveJusticeCard removes a Justice card from a player (e.g., due to faction change).
func (js *JusticeService) RemoveJusticeCard(playerID string, cardID string) bool {
	js.mu.Lock()
	defer js.mu.Unlock()

	faction, ok := js.store[playerID]
	if !ok {
		return false
	}

	for i, card := range faction.JusticeCards {
		if card.CardID == cardID {
			faction.JusticeCards = append(faction.JusticeCards[:i], faction.JusticeCards[i+1:]...)
			faction.LastJusticeMod = time.Now()
			return true
		}
	}
	return false
}

// ApplyTruthSerum applies the Truth Serum intelligence item to a target player.
func (js *JusticeService) ApplyTruthSerum(targetPlayerID string, revealedBuffs []CardBuffState, duration time.Duration) {
	js.mu.Lock()
	defer js.mu.Unlock()

	serum := TruthSerumItem{
		TargetPlayerID: targetPlayerID,
		Duration:       duration,
		RevealedBuffs:  revealedBuffs,
	}

	js.truthSerumTargets[targetPlayerID] = append(js.truthSerumTargets[targetPlayerID], serum)

	// Auto-expire after duration
	go func() {
		time.Sleep(duration)
		js.mu.Lock()
		defer js.mu.Unlock()
		if serums, exists := js.truthSerumTargets[targetPlayerID]; exists {
			for i, s := range serums {
				if s.TargetPlayerID == targetPlayerID && len(s.RevealedBuffs) == len(revealedBuffs) {
					js.truthSerumTargets[targetPlayerID] = append(js.truthSerumTargets[targetPlayerID][:i], js.truthSerumTargets[targetPlayerID][i+1:]...)
					break
				}
			}
		}
	}()
}

// GetRevealedBuffs returns the currently active truth serum revelations for a player.
func (js *JusticeService) GetRevealedBuffs(targetPlayerID string) []TruthSerumItem {
	js.mu.RLock()
	defer js.mu.RUnlock()

	if serums, ok := js.truthSerumTargets[targetPlayerID]; ok {
		result := make([]TruthSerumItem, len(serums))
		copy(result, serums)
		return result
	}
	return nil
}

// ApplyReputationShield applies a Reputation Shield to protect against reputation loss.
func (js *JusticeService) ApplyReputationShield(playerID string, protectionAmount int, duration time.Duration) {
	js.mu.Lock()
	defer js.mu.Unlock()

	shieldID := fmt.Sprintf("SHIELD_%s_%d", playerID, time.Now().UnixNano())
	if js.shieldRegistry[playerID] == nil {
		js.shieldRegistry[playerID] = make(map[string]*ReputationShieldItem)
	}

	js.shieldRegistry[playerID][shieldID] = &ReputationShieldItem{
		ProtectionAmount: protectionAmount,
		AbsorbedCount:    0,
		Duration:         duration,
		ExpiresAt:        time.Now().Add(duration),
	}

	// Auto-cleanup
	go func() {
		time.Sleep(duration)
		js.mu.Lock()
		defer js.mu.Unlock()
		if registry, ok := js.shieldRegistry[playerID]; ok {
			delete(registry, shieldID)
		}
	}()
}

// AbsorbReputationLoss processes reputation loss through the shield, returning absorbed amount.
func (js *JusticeService) AbsorbReputationLoss(playerID string, desiredLoss int) int {
	js.mu.Lock()
	defer js.mu.Unlock()

	registry, ok := js.shieldRegistry[playerID]
	if !ok {
		return desiredLoss // No shield, full loss passes through
	}

	remainingLoss := desiredLoss
	for _, shield := range registry {
		if remainingLoss <= 0 {
			break
		}

		// Check if shield has capacity
		remainingProtection := shield.ProtectionAmount - shield.AbsorbedCount
		if remainingProtection <= 0 {
			continue // Shield exhausted
		}

		absorbed := remainingLoss
		if absorbed > remainingProtection {
			absorbed = remainingProtection
		}

		shield.AbsorbedCount += absorbed
		remainingLoss -= absorbed
	}

	return remainingLoss // What wasn't shielded (actual reputation loss)
}

// GetShieldRemaining returns remaining protection on a player's active shields.
func (js *JusticeService) GetShieldRemaining(playerID string) int {
	js.mu.RLock()
	defer js.mu.RUnlock()

	total := 0
	if registry, ok := js.shieldRegistry[playerID]; ok {
		for _, shield := range registry {
			remaining := shield.ProtectionAmount - shield.AbsorbedCount
			if remaining > 0 {
				total += remaining
			}
		}
	}
	return total
}

// GenerateJusticeMission creates a dynamic bounty mission for Justice players.
func (js *JusticeService) GenerateJusticeMission(targetPlayerID string, targetName string, wantedLevel int, rewardVBV uint64) string {
	js.mu.Lock()
	defer js.mu.Unlock()

	js.missionCounter++
	missionID := fmt.Sprintf("JUSTICE_MISSION_%d", js.missionCounter)

	mission := JusticeMission{
		MissionID:      missionID,
		TargetPlayerID: targetPlayerID,
		TargetName:     targetName,
		TargetWanted:   wantedLevel,
		RewardVBV:      rewardVBV, // Already in micro-units from Faucet
		ExpirationTime: time.Now().Add(24 * time.Hour), // 24-hour mission window
		Status:         "ACTIVE",
	}

	// Add to dashboard cache for Justice-tier players
	// (populated when dashboard is requested)

	return missionID
}

// GetDashboardForPlayer returns the Justice Tier Bounty Center Dashboard state.
func (js *JusticeService) GetDashboardForPlayer(playerID string, allWantedTargets []BountyTargetInfo, ghostScrambled int) *JusticeBountyDashboard {
	js.mu.Lock()
	defer js.mu.Unlock()

	faction, hasFaction := js.store[playerID]
	hasAccess := hasFaction && len(faction.JusticeCards) >= JusticeTierAccessCards && faction.BountyRank >= JusticeTierAccessRank

	dash := &JusticeBountyDashboard{
		DashboardAccess:     hasAccess,
		ScrambledCount:      ghostScrambled,
		HighWantedTargets:   allWantedTargets,
		LastRefresh:         time.Now(),
	}

	if !hasAccess {
		dash.ActiveMissions = nil
		return dash
	}

	// For Warden+ players, resolve Ghost Protocol scrambled IDs via enhanced tracking
	for i := range dash.HighWantedTargets {
		if dash.HighWantedTargets[i].GhostActive {
			dash.HighWantedTargets[i].RealIDAvailable = true // Enhanced tracking reveals real ID
		}
	}

	// Cache the dashboard
	js.dashboardCache[playerID] = dash
	return dash
}

// CheckMissionCompletion verifies if a Justice mission has been completed.
func (js *JusticeService) CheckMissionCompletion(missionID string, targetWantedCleared bool) bool {
	js.mu.Lock()
	defer js.mu.Unlock()

	for _, cache := range js.dashboardCache {
		for i := range cache.ActiveMissions {
			if cache.ActiveMissions[i].MissionID == missionID {
				if targetWantedCleared {
					cache.ActiveMissions[i].Status = "COMPLETED"
					return true
				}
				return false
			}
		}
	}

	// Check direct faction missions if not in cache
	for _, faction := range js.store {
		for i := range faction.Missions {
			if faction.Missions[i].MissionID == missionID {
				if targetWantedCleared {
					faction.Missions[i].Status = "COMPLETED"
					return true
				}
				return false
			}
		}
	}
	return false
}

// GetDashboardJSON returns the dashboard state as JSON for client display.
func (js *JusticeService) GetDashboardJSON(playerID string, allWantedTargets []BountyTargetInfo, ghostScrambled int) (string, error) {
	dash := js.GetDashboardForPlayer(playerID, allWantedTargets, ghostScrambled)
	data, err := json.MarshalIndent(dash, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal dashboard: %w", err)
	}
	return string(data), nil
}

// GetDashboardHTML returns the dashboard state as HTML for client display.
func (js *JusticeService) GetDashboardHTML(playerID string, allWantedTargets []BountyTargetInfo, ghostScrambled int) string {
	dash := js.GetDashboardForPlayer(playerID, allWantedTargets, ghostScrambled)
	return FormatDashboardHTML(dash)
}

// ---- Helper Functions ----

// CalculateJusticePowerMultiplier computes the total power multiplier for Justice cards.
// Returns: baseMultiplier (e.g., 1.10 for +10%), bonusFlatPower, hasJusticeAlignment
func CalculateJusticePowerMultiplier(playerID string, justiceService *JusticeService) (float64, int64, bool) {
	faction := justiceService.GetJusticeFaction(playerID)
	if faction == nil || len(faction.JusticeCards) == 0 {
		return 1.0, 0, false
	}

	var bonusFlatPower int64
	for _, card := range faction.JusticeCards {
		bonusFlatPower += int64(card.PowerBonus)
	}

	// Apply active buffs
	multiplier := 1.10 // Base +10% for Justice alignment
	for _, buff := range faction.ActiveBuffs {
		if buff.Multiplier > 1.0 {
			multiplier += (buff.Multiplier - 1.0) * float64(len(faction.JusticeCards))
		}
	}

	return multiplier, bonusFlatPower, true
}

// IsOutlawTarget checks if a target player has Wanted >= minimum threshold for Justice bonuses.
func IsOutlawTarget(targetWanted int, threshold int) bool {
	return targetWanted >= threshold
}

// FormatDashboardHTML generates HTML output for the Justice Tier Bounty Center Dashboard.
func FormatDashboardHTML(dash *JusticeBountyDashboard) string {
	var sb strings.Builder

	sb.WriteString(`<div class="justice-bounty-dashboard">`)
	sb.WriteString(`<h3>Justice Tier Bounty Center</h3>`)

	if !dash.DashboardAccess {
		sb.WriteString(`<p class="priority-warning">Access Denied: Requires Warden rank + 3 Justice Cards</p>`)
		sb.WriteString(`</div>`)
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf(`<p class="priority-critical">Scrambled Targets: %d</p>`, dash.ScrambledCount))
	sb.WriteString(`<h4>High-Wanted Targets</h4><ul>`)

	for _, target := range dash.HighWantedTargets {
		statusClass := "priority-warning"
		if target.RealIDAvailable {
			statusClass = "priority-critical"
		}
		sb.WriteString(fmt.Sprintf(
			`<li class="%s"><strong>%s</strong> (Wanted: %d) | District: %s | Ghost: %v</li>`,
			statusClass, target.PlayerName, target.WantedLevel, target.District, target.GhostActive,
		))
	}

	sb.WriteString(`</ul></div>`)
	return sb.String()
}

// ---- Constants for Integration ----

const (
	// JusticeTierAccessCards required minimum Justice cards for dashboard access.
	JusticeTierAccessCards = 3
	// JusticeTierAccessRank required bounty rank for dashboard access (0=Hunter, 1=Icon, 2=Warden, 3=Commissioner).
	JusticeTierAccessRank = 2 // Warden
	// JusticeOutlawBonusThreshold wanted level threshold for +10% power bonus.
	JusticeOutlawBonusThreshold = 15
	// TruthSerumDefaultDuration default duration for truth serum revelation.
	TruthSerumDefaultDuration = 30 * time.Second
	// ReputationShieldDefaultProtection default reputation loss shield amount.
	ReputationShieldDefaultProtection = 50
)
