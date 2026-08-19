//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"
)

// ============================================================================
// PILLAR 13: CAREER PROGRESSION SYSTEM (Section 13)
// XP/Level gating for all 20 career roles across Underworld & Justice layers.
// ============================================================================

const (
	CareerTierPeon       = 0
	CareerTierApprentice = 5
	CareerTierJourneyman = 15
	CareerTierExpert     = 30
	CareerTierMaster     = 50
	CareerTierBoss       = 75
)

// PILLAR 13: $VBV-Sustained Tier Thresholds (micro-units)
const (
	VBVTierPeonMicro       = 0                  // Tier 0: no gate
	VBVTierApprentice      = 5_000_000_000      // 5K $VBV
	VBVTierJourneyman      = 25_000_000_000     // 25K $VBV
	VBVTierExpert          = 100_000_000_000    // 100K $VBV
	VBVTierMaster          = 500_000_000_000    // 500K $VBV
	VBVTierBoss            = 2_000_000_000_000  // 2M $VBV
	DemotionGracePeriodDays = 7
)

// CareerXP records a player's progression through career tiers.
// Pillar 13 ($VBV-Sustained Progression): LiquiditySamples + AvgSustainedMicro gate all tier advancement.
type CareerXP struct {
	RoleXP          map[string]uint64 `json:"role_xp"`          // Role name -> XP earned
	LessonLevel     int               `json:"level"`            // Overall career level (0-100)
	PromotedRoles   []string          `json:"promoted_roles"`   // Roles this player qualifies for
	CurrentPrompts  []string          `json:"current_prompts"`  // Active promotion offers (UI)

	// Pillar 13: $VBV-sustained balance tracking
	LiquiditySamples  []uint64  `json:"-"`                // Recent player balance snapshots (micro-$VBV), last 14
	AvgSustainedMicro uint64    `json:"avg_sustained_micro"` // Computed average from samples (micro-$VBV)
	DemotionWarningAt time.Time `json:"demotion_warning_at"` // When demotion warning was issued (0 = none)
}

// CalculateLessonLevel returns the player's career level based on total XP.
func (cxp *CareerXP) CalculateLessonLevel() int {
	totalXP := uint64(0)
	for _, xp := range cxp.RoleXP {
		totalXP += xp
	}
	// Exponential curve: each level requires 1500 more XP than previous.
	level := 0
	for totalXP >= uint64(level*level*30+level*1500) && level < 100 {
		level++
	}
	cxp.LessonLevel = level
	return level
}

// UnlockAchievementForRole grants career-related achievements.
func (cxp *CareerXP) UnlockAchievementForRole(role string) bool {
	for _, r := range cxp.PromotedRoles {
		if r == role {
			return true
		}
	}
	return false
}

// TrackCareerXP adds XP to a specific role's progression.
func (cxp *CareerXP) TrackCareerXP(role string, xp uint64) {
	if cxp == nil {
		return
	}
	if cxp.RoleXP == nil {
		cxp.RoleXP = make(map[string]uint64)
	}
	cxp.RoleXP[role] += xp
}

// CareerHasRole checks if a player has promoted to a specific role.
func CareerHasRole(cxp *CareerXP, role string) bool {
	if cxp == nil {
		return false
	}
	for _, r := range cxp.PromotedRoles {
		if r == role {
			return true
		}
	}
	return false
}

// CheckCareerTierGate validates if player's sustained $VBV balance meets tier requirement.
// PILLAR 13: Returns (tierGatePass, currentTier, requiredMicro, isDemotionWarning).
func (cxp *CareerXP) CheckCareerTierGate(requiredTier int) (bool, int, uint64, bool) {
	if cxp == nil || len(cxp.LiquiditySamples) == 0 {
		return requiredTier == 0, 0, 0, false
	}

	// Compute average sustained balance
	sum := uint64(0)
	for _, sample := range cxp.LiquiditySamples {
		sum += sample
	}
	avg := sum / uint64(len(cxp.LiquiditySamples))
	cxp.AvgSustainedMicro = avg

	// Determine current tier from average balance
	currentTier := 0
	switch {
	case avg >= VBVTierBoss:
		currentTier = 5
	case avg >= VBVTierMaster:
		currentTier = 4
	case avg >= VBVTierExpert:
		currentTier = 3
	case avg >= VBVTierJourneyman:
		currentTier = 2
	case avg >= VBVTierApprentice:
		currentTier = 1
	default:
		currentTier = 0
	}

	// Determine required micro value for requested tier
	requiredMicro := uint64(0)
	switch requiredTier {
	case 5:
		requiredMicro = VBVTierBoss
	case 4:
		requiredMicro = VBVTierMaster
	case 3:
		requiredMicro = VBVTierExpert
	case 2:
		requiredMicro = VBVTierJourneyman
	case 1:
		requiredMicro = VBVTierApprentice
	default:
		requiredMicro = 0
	}

	// Check demotion warning
	isDemotionWarning := false
	if !cxp.DemotionWarningAt.IsZero() && time.Since(cxp.DemotionWarningAt) > time.Duration(DemotionGracePeriodDays)*24*time.Hour {
		isDemotionWarning = true
	}

	gatePass := avg >= requiredMicro && !isDemotionWarning
	return gatePass, currentTier, requiredMicro, isDemotionWarning
}

// RivalryState tracks dynamic rivalry mechanics between players.
type RivalryState struct {
	SoloHunterScore      int                `json:"solo_hunter_score"`        // Bounty hunter solo capture score
	AOSRivalryActive     bool               `json:"aos_rivalry_active"`       // Whether AOS rival is active
	AOSTeamID            string             `json:"aos_team_id"`              // Associated AOS team
	SilkRoadHoardCount   int                `json:"silk_road_hoard_count"`    // Cards hoarded by Hostage Hosts
	HoardPressure        int                `json:"hoard_pressure"`           // Ransom pressure multiplier
	InfoBrokerDeals      []InfoBrokerDeal   `json:"info_broker_deals"`        // Active info broker transactions
	ActiveRivals         []string           `json:"active_rivals,omitempty"`  // Wallet addresses of active rivals
	PendingInvitations   []PendingRivalInvite `json:"pending_invitations,omitempty"` // Pending rival invitations
	BountyLicenseActive  bool               `json:"bounty_license_active"`    // Active bounty hunter license status
	ArcNetActive         bool               `json:"arc_net_active,omitempty"` // Arc-Net spy vision active
}

// ============================================================================
// ADDITIONAL STRUCTS (Rivalry & Info Broker)
// ============================================================================

// InfoBrokerDeal represents an active info trade between two players.
type InfoBrokerDeal struct {
	ID          string    `json:"id"`
	SellerWallet string    `json:"seller_wallet"`
	BuyerWallet string    `json:"buyer_wallet"`
	InfoType    string    `json:"info_type"`    // e.g., "player_location", "card_pool"
	PriceVBV    uint64    `json:"price_vbv"`
	ExpiresAt   time.Time `json:"expires_at"`
	Completed   bool      `json:"completed"`
}

// PendingRivalInvite represents a pending rival request between players.
type PendingRivalInvite struct {
	ID               string    `json:"id"`
	InitiatorWallet  string    `json:"initiator_wallet"`
	TargetWallet     string    `json:"target_wallet"`
	InitiatorCareer  string    `json:"initiator_career"`
	TargetCareer     string    `json:"target_career"`
	CreatedAt        time.Time `json:"created_at"`
	ExpiresAt        time.Time `json:"expires_at"`
}

// GetRivalPairModifier returns the XP bonus for this rivalry pair based on tier.
// Tier scaling: higher tier = higher bonus (reward commitment).
func (cxp *CareerXP) GetRivalPairModifier(rivalCareer string, tier int) float64 {
	baseModifier := 1.0
	tierBonus := 0.1 * float64(tier) // 0.1 per tier level (10% per tier)
	return baseModifier + tierBonus
}

// ============================================================================
// CAREER XP & TIER GATE HELPERS
// ============================================================================

// GetRivalXPDelta returns the XP modifier delta for a rival pair.
// Negative = antagonistic (attacker gains bonus), Positive = synergistic (attacker gains less).
func GetRivalXPDelta(pairName string) int {
	// Enemy pairs (antagonistic - attacker gains bonus)
	switch pairName {
	case "BountyHunter↔Kidnapper", "BountyHunter_Kidnapper":
		return -15
	case "ForensicAnalyst↔Gossip", "ForensicAnalyst_Gossip":
		return -10
	case "TaxAuditor↔Launderer", "TaxAuditor_Launderer":
		return -10
	case "Warden↔HeistPlanner", "Warden_HeistPlanner":
		return -10
	case "SectorPeacekeeper↔Smuggler", "SectorPeacekeeper_Smuggler":
		return -10
	case "IntelAgent↔ArcNetOperative", "IntelAgent_ArcNetOperative":
		return -10
	// Synergistic pairs (existing)
	case "JusticeRecruiter↔BountyHunter", "JusticeRecruiter_BountyHunter":
		return +8
	case "Launderer↔Fence", "Launderer_Fence":
		return +5
	case "HeistPlanner↔Kidnapper", "HeistPlanner_Kidnapper":
		return +12
	case "AOS↔SectorPeacekeeper", "AOS_SectorPeacekeeper":
		return +6
	case "TaxAuditor↔JusticeCommissioner", "TaxAuditor_JusticeCommissioner":
		return +7
	case "Gossip↔ForensicAnalyst", "Gossip_ForensicAnalyst":
		return +5
	default:
		return 0
	}
}

// GetRivalPairName returns the canonical rival pair name for a given attacker-defender combination.
func GetRivalPairName(attackerCareer, defenderCareer string) string {
	// Check all pairs including P2-D7 through P2-D10
	pairs := []struct{ A, B, Name string }{
		{"Bounty Hunter", "Kidnapper", "BountyHunter↔Kidnapper"},
		{"Forensic Analyst", "Gossip", "ForensicAnalyst↔Gossip"},
		{"Tax Auditor", "Launderer", "TaxAuditor↔Launderer"},
		{"Warden", "Heist Planner", "Warden↔HeistPlanner"},
		{"Sector Peacekeeper", "Smuggler", "SectorPeacekeeper↔Smuggler"},
		{"Intel-Agent", "Arc-Net Operative", "IntelAgent↔ArcNetOperative"},
		{"Justice Recruiter", "Bounty Hunter", "JusticeRecruiter↔BountyHunter"},
		{"Launderer", "Fence", "Launderer↔Fence"},
		{"Heist Planner", "Kidnapper", "HeistPlanner↔Kidnapper"},
		{"AOS Leader", "Sector Peacekeeper", "AOS↔SectorPeacekeeper"},
		{"Tax Auditor", "Justice Commissioner", "TaxAuditor↔JusticeCommissioner"},
		{"Gossip", "Forensic Analyst", "Gossip↔ForensicAnalyst"},
		{"AOS Leader", "Sector Peacekeeper", "AOS↔SectorPeacekeeper"},
		{"Tax Auditor", "Justice Commissioner", "TaxAuditor↔JusticeCommissioner"},
		{"Gossip", "Forensic Analyst", "Gossip↔ForensicAnalyst"},
		// P2-D8: Justice Recruiter ↔ Mutation Log Auditor (ally pair)
		{"Justice Recruiter", "Mutation Log Auditor", "JusticeRecruiter↔MutationLogAuditor"},
		// P2-D9: Justice Commissioner ↔ Tax Auditor (allied override)
		{"Justice Commissioner", "Tax Auditor", "JusticeCommissioner↔TaxAuditor"},
		// P2-D10: Mutation Log Auditor ↔ Kidnapper (antagonistic - tracks their genetic tampering)
		{"Mutation Log Auditor", "Kidnapper", "MutationLogAuditor↔Kidnapper"},
	}
	for _, p := range pairs {
		if (attackerCareer == p.A && defenderCareer == p.B) ||
			(attackerCareer == p.B && defenderCareer == p.A) {
			return p.Name
		}
	}
	return ""
}

// TrackRivalInteraction evaluates a cross-career interaction between attacker and defender,
// returns (xpAwarded, rivalPairName, isRivalPair).
// P2-A: Core wiring function — called at criminality handlers.
func TrackRivalInteraction(
	attackerCareer, defenderCareer string,
	baseXP uint64,
	myStats *PlayerStats,
	targetStats *PlayerStats,
) (uint64, string, bool) {
	if baseXP == 0 || attackerCareer == "" || defenderCareer == "" || myStats == nil || targetStats == nil {
		return baseXP, "", false
	}

	pairName := GetRivalPairName(attackerCareer, defenderCareer)
	if pairName == "" {
		return baseXP, "", false
	}

	delta := GetRivalXPDelta(pairName)
	modifier := myStats.CareerXP.GetRivalPairModifier(defenderCareer, getTierFor(myStats.CareerXP, attackerCareer))

	var xpAwarded uint64
	if delta < 0 {
		// Antagonistic: attacker bonus scaled by absolute delta + modifier
		scaling := 1.0 + float64(-delta)/100.0
		xpAwarded = uint64(float64(baseXP) * scaling * modifier)
	} else {
		// Synergistic: attacker bonus scaled by delta + modifier
		scaling := 1.0 + float64(delta)/100.0
		xpAwarded = uint64(float64(baseXP) * scaling * modifier)
	}

	return xpAwarded, pairName, true
}

// getTierFor returns the career tier for a role from the player's XP map.
func getTierFor(cxp *CareerXP, role string) int {
	if cxp == nil || cxp.RoleXP == nil {
		return 1
	}
	xp := cxp.RoleXP[role]
	switch {
	case xp >= uint64(CareerTierMaster*100): // Tier 4: 50 * 100 = 5000 XP
		return 4
	case xp >= uint64(CareerTierExpert*100): // Tier 3: 30 * 100 = 3000 XP
		return 3
	case xp >= uint64(CareerTierApprentice*100): // Tier 2: 5 * 100 = 500 XP
		return 2
	default:
		return 1
	}
}

// EvaluateCrossCareerXP applies rival modifiers for both attacker and defender.
// Returns (attackerXP, defenderXP, pairName, isRivalPair).
// P2-A: Called at interaction resolution to distribute XP fairly between both parties.
func EvaluateCrossCareerXP(
	attackerCareer, defenderCareer string,
	baseAttackerXP uint64,
	myStats *PlayerStats,
	targetStats *PlayerStats,
) (uint64, uint64, string, bool) {
	if baseAttackerXP == 0 || attackerCareer == "" || defenderCareer == "" || myStats == nil || targetStats == nil {
		return baseAttackerXP, 0, "", false
	}

	pairName := GetRivalPairName(attackerCareer, defenderCareer)
	if pairName == "" {
		return baseAttackerXP, 0, "", false
	}

	delta := GetRivalXPDelta(pairName)
	modifier := myStats.CareerXP.GetRivalPairModifier(defenderCareer, getTierFor(myStats.CareerXP, attackerCareer))

	var attackerXP uint64
	if delta < 0 {
		scaling := 1.0 + float64(-delta)/100.0
		attackerXP = uint64(float64(baseAttackerXP) * scaling * modifier)
	} else {
		scaling := 1.0 + float64(delta)/100.0
		attackerXP = uint64(float64(baseAttackerXP) * scaling * modifier)
	}

	// Defender gets a reduced reward (30% of base, representing monitoring/suppression activity)
	defenderXP := uint64(float64(baseAttackerXP) * 0.30)

	return attackerXP, defenderXP, pairName, true
}

// ============================================================================
// HELPER: Map wallet to match player ID for buff application.
// ============================================================================

func playerStatsToID(stats PlayerStats) string {
	// This is a placeholder - in production this would resolve via the lobby's wallet-to-ID map
	return stats.ID
}

// computeScaledXP scales raw career XP by loyalty/fame bonuses from rival pairings and active stats.
// Loyalty bonus represents social influence (rumor discounts, trust).
// Fame bonus represents public visibility/recognition (bounty multipliers, reputation buffs).
func (cxp *CareerXP) computeScaledXP(baseXP uint64, loyaltyBonus, fameBonus float64) uint64 {
	if cxp == nil || baseXP == 0 {
		return baseXP
	}

	scaler := 1.0

	// Apply loyalty bonus (social influence)
	if loyaltyBonus > 0.0 {
		scaler *= 1.0 + loyaltyBonus
	}

	// Apply fame bonus (public visibility / recognition)
	if fameBonus > 0.0 {
		scaler *= 1.0 + fameBonus
	}

	xp := uint64(float64(baseXP) * scaler)

	return xp
}

// ComputeScaledXP is the public unified XP scaling helper for Task 4301 (PILLAR 13 career wiring).
// It computes loyalty and fame bonuses internally from CareerXP state, then applies them to baseXP.
// Usage: scaledXP := stats.CareerXP.ComputeScaledXP(baseXP, role)
func (cxp *CareerXP) ComputeScaledXP(baseXP uint64, role string) uint64 {
	if cxp == nil || baseXP == 0 {
		return baseXP
	}

	var loyaltyBonus float64
	var fameBonus float64

	// Only compute bonuses if the player has active career XP for this role.
	hasCareer := false
	for rName, xpVal := range cxp.RoleXP {
		if strings.EqualFold(rName, role) || CareerHasRole(cxp, rName) && HasCareer(cxp, role) {
			hasCareer = true
			break
		}
	}

	// Also check if the player has any career XP at all (indicates active progression).
	if !hasCareer {
		for _, xpVal := range cxp.RoleXP {
			if xpVal > 0 {
				hasCareer = true
				break
			}
		}
	}

	if !hasCareer {
		return baseXP // No career match — return raw XP
	}

	// Loyalty bonus: scales from total RoleXP across all tiers. +1 loyalty point per 50 XP, max +0.50.
	var totalRoleXP uint64
	for _, xpVal := range cxp.RoleXP {
		totalRoleXP += xpVal
	}
	if totalRoleXP > 0 {
		loyaltyBonus = math.Min(0.50, float64(totalRoleXP)/1000.0) // +0.50 at 2K+ total XP
	}

	// Fame bonus: scales from the player's highest single-role XP tier.
	// Each career tier (Peon=1 through Boss=7) contributes +0.10 per tier level, max +0.60.
	var maxRoleXP uint64
	for _, xpVal := range cxp.RoleXP {
		if xpVal > maxRoleXP {
			maxRoleXP = xpVal
		}
	}
	if maxRoleXP >= 500 { // Apprentice tier threshold (5 * 100)
		tierLevel := getTierFor(cxp, role)
		fameBonus = math.Min(0.60, float64(tierLevel)*0.10)
	}

	return cxp.computeScaledXP(baseXP, loyaltyBonus, fameBonus)
}

// HasCareer checks if a player has any career XP for the given role name (case-insensitive).
func HasCareer(cxp *CareerXP, role string) bool {
	if cxp == nil || cxp.RoleXP == nil {
		return false
	}
	for rName := range cxp.RoleXP {
		if strings.EqualFold(rName, role) {
			return true
		}
	}
	return CareerHasRole(cxp, role)
}

// ============================================================================
// PILLAR 8: FENCE CAREER METHODS
// Fence fee discount and tier gating for black market operations.
// ============================================================================

// GetFenceFeeDiscount returns the fee discount multiplier for a Fence career role.
// Tier scaling (based on $VBV-sustained thresholds):
//   - Tier < 3 (below Journeyman): returns 1.0 (no discount)
//   - Tier >= 3 (Journeyman+):     returns 0.50 (50% fee reduction per PILLAR 8 spec)
func (cxp *CareerXP) GetFenceFeeDiscount() float64 {
	if cxp == nil || cxp.RoleXP == nil {
		return 1.0 // Default: no discount if no career data
	}

	fenceXP := cxp.RoleXP["Fence"]
	tier := getTierFor(cxp, "Fence")

	switch {
	case tier >= 3: // Journeyman+ (25K $VBV sustained) — full PILLAR 8 discount
		return 0.50
	case tier >= 2: // Apprentice+ (5K $VBV sustained) — partial discount
		return 0.75
	default: // Peon (< 5K $VBV) — no career discount
		return 1.0
	}
}

// GetFenceTier returns the integer tier level for the Fence career role.
// Tier mapping (consistent with CareerXP progression):
//   - Tier 1 (Peon):        < 5K $VBV sustained
//   - Tier 2 (Apprentice):  >= 5K $VBV sustained
//   - Tier 3 (Journeyman):  >= 25K $VBV sustained
//   - Tier 4 (Expert):      >= 100K $VBV sustained
//   - Tier 5 (Master):      >= 500K $VBV sustained
//   - Tier 6 (Boss):        >= 2M $VBV sustained
func (cxp *CareerXP) GetFenceTier() int {
	if cxp == nil || cxp.RoleXP == nil {
		return 1 // Default: Peon tier if no career data
	}

	tier := getTierFor(cxp, "Fence")
	if tier < 1 {
		return 1
	}
	return tier
}

// ============================================================================
// PILLAR 13: $VBV-SUSTAINED GATE MULTIPLIER (Section 15)
// Returns the XP multiplier based on sustained liquidity tier.
// Multiplier tiers: Peon(0)=×1, Apprentice=×2, Journeyman=×4, Expert=×8, Master=×16, Boss=×32
// ============================================================================

// GetVBVGatingMultiplier returns the $VBV-gated XP multiplier based on AvgSustainedMicro.
func (cxp *CareerXP) GetVBVGatingMultiplier() float64 {
	if cxp == nil || cxp.AvgSustainedMicro < VBVTierApprentice {
		return 1.0 // Tier 0: no gate — baseline multiplier
	}

	switch {
	case cxp.AvgSustainedMicro >= VBVTierBoss:
		return 32.0
	case cxp.AvgSustainedMicro >= VBVTierMaster:
		return 16.0
	case cxp.AvgSustainedMicro >= VBVTierExpert:
		return 8.0
	case cxp.AvgSustainedMicro >= VBVTierJourneyman:
		return 4.0
	default: // Apprentice tier (5K–25K $VBV sustained)
		return 2.0
	}
}

// ============================================================================
// PILLAR 13: TAX AUDITOR CAREER — GetAuditPrecisionBonus
// Returns the audit accuracy multiplier for TaxAuditor career tier.
// Tier 1 (Junior): ×1.0 baseline precision
// Tier 2 (Auditor): ×1.15 +15% detection rate on hidden treasury operations
// Tier 3 (Senior Auditor): ×1.35 flags cross-club fund routing
// Tier 4 (Chief Auditor): ×1.60 uncovers offshore shell entities
// Tier 5 (Commissioner): ×2.00 full civilization-wide audit capability
// ============================================================================

// GetAuditPrecisionBonus returns the audit accuracy multiplier for TaxAuditor career tier.
func (cxp *CareerXP) GetAuditPrecisionBonus() float64 {
	if cxp == nil || cxp.RoleXP == nil {
		return 1.0 // no bonus if invalid input or no role data
	}

	hasTaxAuditor := false
	for rName := range cxp.RoleXP {
		if strings.EqualFold(rName, "TaxAuditor") {
			hasTaxAuditor = true
			break
		}
	}
	if !hasTaxAuditor && !CareerHasRole(cxp, "TaxAuditor") {
		return 1.0 // no bonus for non-TaxAuditor players
	}

	tier := getTierFor(cxp, "TaxAuditor")
	switch {
	case tier >= 5: // Commissioner+
		return 2.00
	case tier >= 4: // Chief Auditor+
		return 1.60
	case tier >= 3: // Senior Auditor+
		return 1.35
	case tier >= 2: // Auditor+
		return 1.15
	default: // Junior (Tier 1) — baseline precision
		return 1.0
	}
}

// IsJusticeAligned returns true if the given career choice string corresponds to a Justice-aligned career path.
func IsJusticeAligned(careerChoice string) bool {
	switch careerChoice {
	case "BountyHunter", "IntelAgent", "AOSLeader", "JusticeRecruiter",
		"Warden", "MutationAuditor", "TaxAuditor", "SectorPeacekeeper",
		"JusticeCommissioner":
		return true
	default:
		return false
	}
}

// GetDecryptBonus returns the Arc-Net vision decryption multiplier for Intel-Agent at the given tier.
// Tier≥3 → 2.0× decrypted visibility; Tier≥5 → 3.0× (full sector sweep).
func GetDecryptBonus(tier int) float64 {
	switch {
	case tier >= 5: // Commissioner+ — full sector sweep
		return 3.0
	case tier >= 3: // Expert+ — double visibility
		return 2.0
	default: // Apprentice or below — baseline reveal
		return 1.0
	}
}

// GetEvidenceAccuracyBonus returns the evidence accuracy multiplier for Forensic Analyst at the given tier.
// Tier≥3 → 2.0× effectiveness (double-clean vs Gossip records).
func GetEvidenceAccuracyBonus(tier int) float64 {
	switch {
	case tier >= 3: // Expert+ — double-clean
		return 2.0
	default: // Apprentice or below — baseline evidence capture
		return 1.0
	}
}

// GetRecruitmentBonus returns the starting power multiplier for new justice-aligned players recruited by Justice Recruiter.
func GetRecruitmentBonus() float64 {
	return 1.05 // +5% starting power bonus per recruit
}

// ============================================================================
// PILLAR 13: HEIST PLANNER CAREER — GetPlanningBuff (Task 4201-9B)
// Returns the team heist success rate multiplier for Heist Planner at given tier.
// Tier≥1 → +5% per tier level (minimum ×1.0, scales linearly up to Boss tier).
// ============================================================================

// GetPlanningBuff returns the team planning buff multiplier for Heist Planner career tier.
// Returns base 1.0 if player is not a Heist Planner or has no CareerXP data.
func (cxp *CareerXP) GetPlanningBuff() float64 {
	if cxp == nil || cxp.RoleXP == nil {
		return 1.0 // No buff for non-HeistPlanner players
	}

	hasHP := false
	for rName := range cxp.RoleXP {
		if strings.EqualFold(rName, "Heist Planner") {
			hasHP = true
			break
		}
	}
	if !hasHP && !CareerHasRole(cxp, "Heist Planner") {
		return 1.0 // No buff for non-HeistPlanner players
	}

	tier := getTierFor(cxp, "Heist Planner")
	// +5% per tier level: Tier 1 = ×1.05, Tier 2 = ×1.10, ..., Boss (tier≥7) = ×1.35 cap
	buff := 1.0 + float64(tier)*0.05
	if buff > 1.35 {
		buff = 1.35 // Cap at Boss tier equivalent (+35% max team success rate)
	}
	return buff
}

// GetHeistDividendRate returns the dividend multiplier for Heist Planner based on career tier.
// Returns base 0.02 (2%) per member in planner's org, scaled by tier level.
func (cxp *CareerXP) GetHeistDividendRate() float64 {
	if cxp == nil || cxp.RoleXP == nil {
		return 0.0 // No dividend for non-Planner players
	}

	hasHP := false
	for rName := range cxp.RoleXP {
		if strings.EqualFold(rName, "Heist Planner") {
			hasHP = true
			break
		}
	}
	if !hasHP && !CareerHasRole(cxp, "Heist Planner") {
		return 0.0 // No dividend for non-Planner players
	}

	tier := getTierFor(cxp, "Heist Planner")
	switch {
	case tier >= 5: // Boss+ — maximum dividend share
		return 0.08
	case tier >= 4: // Master+ 
		return 0.06
	case tier >= 3: // Expert+
		return 0.05
	default: // Apprentice and below
		return 0.02 + float64(tier)*0.01 // Tier 1 = ×1%, Tier 2 = ×2%
	}
}

// HasHeistPlannerRole checks if the player has promoted to Heist Planner career role.
func (cxp *CareerXP) HasHeistPlannerRole() bool {
	if cxp == nil || cxp.RoleXP == nil {
		return false
	}
	for rName := range cxp.RoleXP {
		if strings.EqualFold(rName, "Heist Planner") {
			return true
		}
	}
	return CareerHasRole(cxp, "Heist Planner")
}
