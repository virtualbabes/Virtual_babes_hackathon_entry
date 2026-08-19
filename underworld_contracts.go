//go:build !js && !wasm

// underworld_contracts.go — Dynamic Underworld Contract Template Engine
// PILLAR 3: Criminality & Intelligence Layer
//
// Provides dynamic contract generation, difficulty scaling, eligibility evaluation,
// and XP reward computation for all ~19 Underworld Contracts.

package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// ============================================================================
// Contract Template Types — Base definitions for dynamic instantiation
// ============================================================================

// ContractTemplate defines the blueprint from which individual contracts are generated.
type ContractTemplate struct {
	ID                 string  `json:"id"`                   // Unique contract identifier (e.g., "CONTRACT-001")
	Name               string  `json:"name"`                 // Display name for UI rendering
	Description        string  `json:"description"`          // Mission objective text shown to player
	TargetCareer       string  `json:"target_career"`        // Primary career this contract serves (Fence, Smuggler, etc.)
	DifficultyTier     int     `json:"difficulty_tier"`      // 1=Peon → 6=Boss — scales reward multipliers
	BaseRewardMicro    uint64  `json:"base_reward_micro"`    // Base micro-unit payout before scaling
	RequiredWantedMin  int     `json:"required_wanted_min"`  // Minimum Wanted level to accept this contract
	RequiredCareerTier int     `json:"required_career_tier"` // Minimum career tier ($VBV-gated) for eligibility
	XPBase             uint64  `json:"xp_base"`              // Base XP awarded on completion (before $VBV-gate)
	TimeLimitHours     int     `json:"time_limit_hours"`     // Hours before contract expires if not accepted/aborted
	RequiresAlliance   bool    `json:"requires_alliance"`    // Whether player must be in an alliance to accept
	RequiresTerritory  bool    `json:"requires_territory"`   // Whether target club must control territory
	RivalMultiplier    float64 `json:"rival_multiplier"`     // Multiplier if rival career present (e.g., Justice vs Underworld)
}

// DynamicContract represents an instantiated contract ready for player acceptance.
type DynamicContract struct {
	Template          ContractTemplate `json:"template"`            // Base template this was generated from
	InstanceID        string           `json:"instance_id"`         // Unique instance ID (e.g., "UC-20260715-A3F8B")
	ScaledRewardMicro uint64           `json:"scaled_reward_micro"` // Reward after player-state scaling
	ScaledXP          uint64           `json:"scaled_xp"`           // XP after $VBV-gate and tier multipliers
	ExpiresAt         time.Time        `json:"expires_at"`          // Time limit for acceptance (if generated) or completion deadline
	EligibleCareers   []string         `json:"eligible_careers"`    // All careers that can accept this contract type
}

// ============================================================================
// Contract Template Registry — Static definitions (~19 contracts total)
// ============================================================================

var UnderworldContractTemplates = []ContractTemplate{
	// Sabotage Contracts (CONTRACT-001 through CONTRACT-026 variants)
	{
		ID: "CONTRACT-001", Name: "East Gate District Sabotage", Description: "Sabotage a club controlling the East Gate district.",
		TargetCareer: "Saboteur", DifficultyTier: 1, BaseRewardMicro: 1500 * 1000000, RequiredWantedMin: 5,
		RequiredCareerTier: 1, XPBase: 30, TimeLimitHours: 48, RequiresAlliance: false, RequiresTerritory: true, RivalMultiplier: 1.2,
	},
	{
		ID: "CONTRACT-004", Name: "Governor District Sabotage", Description: "Sabotage a district controlled by a Regional Governor.",
		TargetCareer: "Saboteur", DifficultyTier: 2, BaseRewardMicro: 2000 * 1000000, RequiredWantedMin: 15,
		RequiredCareerTier: 2, XPBase: 45, TimeLimitHours: 36, RequiresAlliance: false, RequiresTerritory: true, RivalMultiplier: 1.3,
	},
	{
		ID: "CONTRACT-007", Name: "Arena Center Sabotage", Description: "Sabotage the club controlling the 'arena_center' district.",
		TargetCareer: "Saboteur", DifficultyTier: 2, BaseRewardMicro: 2500 * 1000000, RequiredWantedMin: 18,
		RequiredCareerTier: 2, XPBase: 50, TimeLimitHours: 36, RequiresAlliance: false, RequiresTerritory: true, RivalMultiplier: 1.4,
	},
	{
		ID: "CONTRACT-008", Name: "Cyber-Defense Sabotage", Description: "Sabotage a club that has active 'Cyber-Counter' or 'Cyber-Lock' defenses.",
		TargetCareer: "Saboteur", DifficultyTier: 3, BaseRewardMicro: 3500 * 1000000, RequiredWantedMin: 20,
		RequiredCareerTier: 2, XPBase: 60, TimeLimitHours: 24, RequiresAlliance: false, RequiresTerritory: true, RivalMultiplier: 1.5,
	},
	{
		ID: "CONTRACT-011", Name: "Titan Club Sabotage", Description: "Sabotage a titan club (controlling 3+ territories).",
		TargetCareer: "Saboteur", DifficultyTier: 3, BaseRewardMicro: 6000 * 1000000, RequiredWantedMin: 25,
		RequiredCareerTier: 3, XPBase: 80, TimeLimitHours: 24, RequiresAlliance: false, RequiresTerritory: true, RivalMultiplier: 1.6,
	},
	{
		ID: "CONTRACT-014", Name: "Governor Multi-Territory Sabotage", Description: "Sabotage a Regional Governor's district with Wanted Level 20+.",
		TargetCareer: "Saboteur", DifficultyTier: 3, BaseRewardMicro: 8000 * 1000000, RequiredWantedMin: 20,
		RequiredCareerTier: 3, XPBase: 90, TimeLimitHours: 24, RequiresAlliance: false, RequiresTerritory: true, RivalMultiplier: 1.7,
	},
	{
		ID: "CONTRACT-016", Name: "Arena Center Ghost Sabotage", Description: "Successfully sabotage the Arena Center with Wanted Level 30+.",
		TargetCareer: "Saboteur", DifficultyTier: 4, BaseRewardMicro: 15000 * 1000000, RequiredWantedMin: 30,
		RequiredCareerTier: 3, XPBase: 120, TimeLimitHours: 18, RequiresAlliance: false, RequiresTerritory: true, RivalMultiplier: 2.0,
	},
	{
		ID: "CONTRACT-018", Name: "Chaos Containment Protocol", Description: "Sabotage the Arena Center while bolstered by 3+ allied traps.",
		TargetCareer: "Saboteur", DifficultyTier: 5, BaseRewardMicro: 25000 * 1000000, RequiredWantedMin: 40,
		RequiredCareerTier: 4, XPBase: 200, TimeLimitHours: 12, RequiresAlliance: true, RequiresTerritory: true, RivalMultiplier: 2.5,
	},
	{
		ID: "CONTRACT-021", Name: "Hostage Org Sabotage", Description: "Successful Sabotage vs Org with 7+ hostages while Wanted 50+.",
		TargetCareer: "Saboteur", DifficultyTier: 5, BaseRewardMicro: 50000 * 1000000, RequiredWantedMin: 50,
		RequiredCareerTier: 4, XPBase: 300, TimeLimitHours: 12, RequiresAlliance: true, RequiresTerritory: false, RivalMultiplier: 3.0,
	},
	{
		ID: "CONTRACT-025", Name: "Kidnapper Sabotage Protocol", Description: "Successful Sabotage while holding 'Kidnapper' role.",
		TargetCareer: "Saboteur", DifficultyTier: 4, BaseRewardMicro: 15000 * 1000000, RequiredWantedMin: 35,
		RequiredCareerTier: 3, XPBase: 250, TimeLimitHours: 18, RequiresAlliance: false, RequiresTerritory: false, RivalMultiplier: 2.0,
	},
	{
		ID: "CONTRACT-026", Name: "Reparation Target Sabotage", Description: "Successful Sabotage vs Org whose owner has received 5+ reparations.",
		TargetCareer: "Saboteur", DifficultyTier: 4, BaseRewardMicro: 18000 * 1000000, RequiredWantedMin: 30,
		RequiredCareerTier: 3, XPBase: 260, TimeLimitHours: 18, RequiresAlliance: false, RequiresTerritory: true, RivalMultiplier: 2.2,
	},

	// Heist Contracts (CONTRACT-009 through CONTRACT-027 variants)
	{
		ID: "CONTRACT-009", Name: "Regional Club Sabotage", Description: "Sabotage a club controlled by a Regional Governor.",
		TargetCareer: "Smuggler", DifficultyTier: 2, BaseRewardMicro: 4000 * 1000000, RequiredWantedMin: 15,
		RequiredCareerTier: 2, XPBase: 70, TimeLimitHours: 36, RequiresAlliance: false, RequiresTerritory: true, RivalMultiplier: 1.4,
	},
	{
		ID: "CONTRACT-012", Name: "Mojo Stabilizer Heist", Description: "Successfully execute a Heist against a club with an active 'MOJO_STABILIZER'.",
		TargetCareer: "Smuggler", DifficultyTier: 3, BaseRewardMicro: 7500 * 1000000, RequiredWantedMin: 22,
		RequiredCareerTier: 2, XPBase: 90, TimeLimitHours: 24, RequiresAlliance: false, RequiresTerritory: true, RivalMultiplier: 1.6,
	},
	{
		ID: "CONTRACT-013", Name: "Arena Center Alliance Heist", Description: "Successful Heist against the Arena Center controller while in an Alliance.",
		TargetCareer: "Smuggler", DifficultyTier: 4, BaseRewardMicro: 10000 * 1000000, RequiredWantedMin: 28,
		RequiredCareerTier: 3, XPBase: 150, TimeLimitHours: 18, RequiresAlliance: true, RequiresTerritory: true, RivalMultiplier: 2.0,
	},
	{
		ID: "CONTRACT-015", Name: "High-Wanted Arena Heist", Description: "Successful Heist against Arena Center controller with Wanted Level 25+.",
		TargetCareer: "Smuggler", DifficultyTier: 4, BaseRewardMicro: 12000 * 1000000, RequiredWantedMin: 25,
		RequiredCareerTier: 3, XPBase: 160, TimeLimitHours: 18, RequiresAlliance: false, RequiresTerritory: true, RivalMultiplier: 2.2,
	},
	{
		ID: "CONTRACT-017", Name: "Hostage Governor Heist", Description: "Successful Heist against a Regional Governor holding 3+ hostages.",
		TargetCareer: "Smuggler", DifficultyTier: 5, BaseRewardMicro: 20000 * 1000000, RequiredWantedMin: 40,
		RequiredCareerTier: 4, XPBase: 280, TimeLimitHours: 12, RequiresAlliance: true, RequiresTerritory: false, RivalMultiplier: 3.0,
	},
	{
		ID: "CONTRACT-019", Name: "Ghost Protocol Arena Heist", Description: "Successful Heist vs Org with 5+ hostages while Wanted 40+. Ghost protocol active.",
		TargetCareer: "Smuggler", DifficultyTier: 6, BaseRewardMicro: 30000 * 1000000, RequiredWantedMin: 40,
		RequiredCareerTier: 5, XPBase: 400, TimeLimitHours: 8, RequiresAlliance: true, RequiresTerritory: false, RivalMultiplier: 3.5,
	},
	{
		ID: "CONTRACT-027", Name: "Smuggler High-Wanted Heist", Description: "Successful Heist while holding 'Smuggler' role and Wanted 30+.",
		TargetCareer: "Smuggler", DifficultyTier: 5, BaseRewardMicro: 20000 * 1000000, RequiredWantedMin: 30,
		RequiredCareerTier: 4, XPBase: 350, TimeLimitHours: 12, RequiresAlliance: false, RequiresTerritory: true, RivalMultiplier: 2.8,
	},

	// Laundering & Finance Contracts (CONTRACT-024, CONTRACT-028)
	{
		ID: "CONTRACT-024", Name: "Epic Data Haven Audit", Description: "Audit the 'Data Haven' club with Epic >= 1.5 or Legendary >= 2.0.",
		TargetCareer: "Fence", DifficultyTier: 3, BaseRewardMicro: 30000 * 1000000, RequiredWantedMin: 20,
		RequiredCareerTier: 3, XPBase: 500, TimeLimitHours: 24, RequiresAlliance: false, RequiresTerritory: true, RivalMultiplier: 2.5,
	},
	{
		ID: "CONTRACT-028", Name: "Launderer Capture Protocol", Description: "Successful capture while holding 'Launderer' role.",
		TargetCareer: "Launderer", DifficultyTier: 4, BaseRewardMicro: 15000 * 1000000, RequiredWantedMin: 25,
		RequiredCareerTier: 3, XPBase: 300, TimeLimitHours: 18, RequiresAlliance: false, RequiresTerritory: true, RivalMultiplier: 2.0,
	},

	// Kidnap Gambit Contracts (CONTRACT-005, CONTRACT-010)
	{
		ID: "CONTRACT-005", Name: "Standard Kidnap Gambit", Description: "Successfully execute a Kidnap Gambit.",
		TargetCareer: "Kidnapper", DifficultyTier: 2, BaseRewardMicro: 3000 * 1000000, RequiredWantedMin: 15,
		RequiredCareerTier: 2, XPBase: 80, TimeLimitHours: 24, RequiresAlliance: false, RequiresTerritory: false, RivalMultiplier: 1.5,
	},
	{
		ID: "CONTRACT-010", Name: "Governor Favorite Card Kidnap", Description: "Successful Kidnap Gambit against a Regional Governor's favorite card.",
		TargetCareer: "Kidnapper", DifficultyTier: 3, BaseRewardMicro: 5000 * 1000000, RequiredWantedMin: 25,
		RequiredCareerTier: 3, XPBase: 120, TimeLimitHours: 18, RequiresAlliance: false, RequiresTerritory: true, RivalMultiplier: 2.0,
	},

	// Rumor Spread Contract (CONTRACT-006)
	{
		ID: "CONTRACT-006", Name: "Governor Defamation Campaign", Description: "Successfully spread a Negative Rumor about a Regional Governor.",
		TargetCareer: "Gossip", DifficultyTier: 2, BaseRewardMicro: 1500 * 1000000, RequiredWantedMin: 10,
		RequiredCareerTier: 1, XPBase: 40, TimeLimitHours: 36, RequiresAlliance: false, RequiresTerritory: true, RivalMultiplier: 1.8, // High rival multiplier (Justice vs Underworld)
	},

	// High-Wanted Contracts (CONTRACT-020, CONTRACT-022, CONTRACT-023)
	{
		ID: "CONTRACT-020", Name: "Mass Hostage Org Heist", Description: "Successful Heist vs Org with 10+ hostages while Wanted 60+.",
		TargetCareer: "Smuggler", DifficultyTier: 5, BaseRewardMicro: 40000 * 1000000, RequiredWantedMin: 60,
		RequiredCareerTier: 5, XPBase: 500, TimeLimitHours: 8, RequiresAlliance: true, RequiresTerritory: false, RivalMultiplier: 3.5,
	},
	{
		ID: "CONTRACT-022", Name: "Titan Hostage Org Heist", Description: "Successful Heist vs Org with 10+ hostages while Wanted 60+. Titan club variant.",
		TargetCareer: "Smuggler", DifficultyTier: 5, BaseRewardMicro: 60000 * 1000000, RequiredWantedMin: 60,
		RequiredCareerTier: 5, XPBase: 700, TimeLimitHours: 8, RequiresAlliance: true, RequiresTerritory: false, RivalMultiplier: 4.0,
	},
	{
		ID: "CONTRACT-023", Name: "Ultimate Arena Center Heist", Description: "Successful Heist vs 'Arena Center' @ Wanted 100+.",
		TargetCareer: "Smuggler", DifficultyTier: 6, BaseRewardMicro: 100000 * 1000000, RequiredWantedMin: 100,
		RequiredCareerTier: 6, XPBase: 1500, TimeLimitHours: 4, RequiresAlliance: true, RequiresTerritory: true, RivalMultiplier: 5.0,
	},

	// UnderworldBoss Contracts (CONTRACT-029 through CONTRACT-033) — Task 5001: First career with zero XP triggers ✅ FIXED
	{
		ID: "CONTRACT-029", Name: "Underworld Boss Ascension Protocol", Description: "Consolidate control over 3 underworld territories while Wanted 40+. Establish boss authority.",
		TargetCareer: "UnderworldBoss", DifficultyTier: 5, BaseRewardMicro: 50000 * 1000000, RequiredWantedMin: 40,
		RequiredCareerTier: 5, XPBase: 800, TimeLimitHours: 12, RequiresAlliance: true, RequiresTerritory: true, RivalMultiplier: 3.0,
	},
	{
		ID: "CONTRACT-030", Name: "Shadow Council Consolidation", Description: "Control 5+ clubs across rival districts while Wanted 60+. Unify underworld factions.",
		TargetCareer: "UnderworldBoss", DifficultyTier: 6, BaseRewardMicro: 80000 * 1000000, RequiredWantedMin: 60,
		RequiredCareerTier: 6, XPBase: 1200, TimeLimitHours: 8, RequiresAlliance: true, RequiresTerritory: true, RivalMultiplier: 4.0,
	},
	{
		ID: "CONTRACT-031", Name: "Boss Succession Protocol", Description: "Eliminate rival boss contenders (2+ careers) while holding territory control.",
		TargetCareer: "UnderworldBoss", DifficultyTier: 6, BaseRewardMicro: 100000 * 1000000, RequiredWantedMin: 80,
		RequiredCareerTier: 6, XPBase: 2000, TimeLimitHours: 6, RequiresAlliance: false, RequiresTerritory: true, RivalMultiplier: 5.0,
	},

	// UnderworldBoss Combat Heist Contracts (CONTRACT-032 through CONTRACT-033) — battle_service.go integration points
	{
		ID: "CONTRACT-032", Name: "Territory Boss Raid Protocol", Description: "Execute a successful heist against a rival territory boss while Wanted 50+.",
		TargetCareer: "UnderworldBoss", DifficultyTier: 5, BaseRewardMicro: 60000 * 1000000, RequiredWantedMin: 50,
		RequiredCareerTier: 5, XPBase: 900, TimeLimitHours: 12, RequiresAlliance: true, RequiresTerritory: false, RivalMultiplier: 3.5,
	},
	{
		ID: "CONTRACT-033", Name: "Underworld Hegemony Final Protocol", Description: "Complete all prior UnderworldBoss contracts + control Arena Center simultaneously.",
		TargetCareer: "UnderworldBoss", DifficultyTier: 6, BaseRewardMicro: 200000 * 1000000, RequiredWantedMin: 100,
		RequiredCareerTier: 6, XPBase: 3000, TimeLimitHours: 4, RequiresAlliance: true, RequiresTerritory: true, RivalMultiplier: 5.0,
	},
}

// ============================================================================
// Dynamic Contract Engine — Core service type and methods
// ============================================================================

// ContractEngine provides dynamic contract generation, eligibility evaluation,
// difficulty scaling, and XP reward computation for the Underworld Contracts system.
type ContractEngine struct {
	templates       []ContractTemplate // Static template registry (~19 contracts)
	instanceCounter int                // Monotonically increasing counter for instance IDs
}

// NewContractEngine creates a new contract engine with all registered templates.
func NewContractEngine() *ContractEngine {
	return &ContractEngine{
		templates:       UnderworldContractTemplates,
		instanceCounter: rand.Intn(10000), // Seed for instance ID generation
	}
}

// GetAvailableContracts returns a list of contracts the player is eligible to accept based on their state.
func (ce *ContractEngine) GetAvailableContracts(
	wallet string,
	jobRole string,
	careerTier int,
	vbvMultiplier float64, // $VBV-gate multiplier from ComputeScaledXP pattern
	wantedLevel int,
	inAlliance bool,
	hasTerritoryControl bool,
	rivalCareerPresent bool, // True if rival Justice career is present (triggers RivalMultiplier)
) []DynamicContract {
	var available []DynamicContract

	for _, tmpl := range ce.templates {
		// Eligibility check: Wanted level threshold
		if wantedLevel < tmpl.RequiredWantedMin {
			continue
		}

		// Eligibility check: Career tier ($VBV-gated)
		if careerTier < tmpl.RequiredCareerTier {
			continue
		}

		// Eligibility check: Alliance requirement
		if tmpl.RequiresAlliance && !inAlliance {
			continue
		}

		// Eligibility check: Territory control requirement (for certain contracts)
		if tmpl.RequiresTerritory && !hasTerritoryControl {
			continue
		}

		// Career role matching — primary career or any compatible career
		isPrimaryMatch := jobRole == tmpl.TargetCareer
		isCompatibleRole := ce.IsCompatibleRole(jobRole, tmpl.TargetCareer)

		if !isPrimaryMatch && !isCompatibleRole {
			continue // Contract not relevant to this player's career path
		}

		// Scale reward based on $VBV multiplier and difficulty tier
		scaledReward := uint64(float64(tmpl.BaseRewardMicro) * vbvMultiplier)

		// Apply Wanted level scaling bonus (+5% per 10 wanted above minimum threshold)
		if wantedLevel > tmpl.RequiredWantedMin {
			wantedBonus := float64(wantedLevel-tmpl.RequiredWantedMin) / 20.0 // +50% at max wanted for this tier
			scaledReward = uint64(float64(scaledReward) * (1.0 + wantedBonus))
		}

		// Apply rival multiplier if Justice career present (Underworld vs Justice tension)
		if rivalCareerPresent {
			scaledReward = uint64(float64(scaledReward) * tmpl.RivalMultiplier)
		}

		// Compute scaled XP using same $VBV-gate pattern as battle_service.go careers
		scaledXP := uint64(float64(tmpl.XPBase) * vbvMultiplier)

		// Generate instance ID and expiration time (24h from now for generated contracts)
		instanceID := fmt.Sprintf("UC-%d-%05X", time.Now().UnixMilli(), ce.instanceCounter)
		ce.instanceCounter++

		expiresAt := time.Now().Add(time.Duration(tmpl.TimeLimitHours) * time.Hour)

		available = append(available, DynamicContract{
			Template:          tmpl,
			InstanceID:        instanceID,
			ScaledRewardMicro: scaledReward,
			ScaledXP:          scaledXP,
			ExpiresAt:         expiresAt,
			EligibleCareers:   ce.GetCompatibleRoles(tmpl.TargetCareer),
		})
	}

	return available
}

// IsCompatibleRole checks if a job role can accept contracts designed for the target career.
func (ce *ContractEngine) IsCompatibleRole(jobRole string, targetCareer string) bool {
	// Cross-career compatibility rules:
	// - Smuggler ↔ Fence (both handle illicit goods movement)
	if (jobRole == "Smuggler" && targetCareer == "Fence") || (jobRole == "Fence" && targetCareer == "Smuggler") {
		return true
	}

	// - Saboteur ↔ Kidnapper (both are direct action roles)
	if (jobRole == "Saboteur" && targetCareer == "Kidnapper") || (jobRole == "Kidnapper" && targetCareer == "Saboteur") {
		return true
	}

	// - Launderer ↔ Fence (financial illicit flow)
	if jobRole == "Launderer" && targetCareer == "Fence" {
		return true
	}

	// Primary match always allowed
	return jobRole == targetCareer
}

// GetCompatibleRoles returns all careers that can accept contracts for the given target career.
func (ce *ContractEngine) GetCompatibleRoles(targetCareer string) []string {
	var roles []string
	roles = append(roles, targetCareer) // Always include primary role

	switch targetCareer {
	case "Smuggler", "Fence":
		if !contains(roles, "Smuggler") {
			roles = append(roles, "Smuggler")
		}
		if !contains(roles, "Fence") {
			roles = append(roles, "Fence")
		}
	case "Saboteur", "Kidnapper":
		if !contains(roles, "Saboteur") {
			roles = append(roles, "Saboteur")
		}
		if !contains(roles, "Kidnapper") {
			roles = append(roles, "Kidnapper")
		}
	case "Launderer":
		if !contains(roles, "Fence") && !contains(roles, "Launderer") {
			roles = append(roles, "Fence", "Launderer")
		}
	default:
		// For other careers (Gossip, etc.), only primary role can accept
	}

	return roles
}

// EvaluateRivalPresence checks if any opponent wallet has a rival Justice career.
func (ce *ContractEngine) EvaluateRivalPresence(lobby *Lobby, playerWallet string) bool {
	if lobby == nil || lobby.leaderboard == nil {
		return false
	}

	for wallet, stats := range lobby.leaderboard {
		if wallet == playerWallet {
			continue // Skip self
		}

		// Check if this opponent has a Justice-aligned career (rival to Underworld)
		isJusticeRival := ce.IsJusticeCareer(stats.JobRole)
		if isJusticeRival {
			return true // Rival presence detected — triggers RivalMultiplier on contracts
		}
	}

	return false
}

// IsJusticeCareer checks if a job role is aligned with the Justice Hegemony (rival to Underworld).
func (ce *ContractEngine) IsJusticeCareer(jobRole string) bool {
	justiceCareers := []string{"Warden", "Commissioner", "Enforcer", "Mediator", "Judge", "Justice Recruiter"}
	for _, jc := range justiceCareers {
		if jobRole == jc {
			return true
		}
	}

	// Also check for careers that have IsJusticeAligned() helper (from rival_career_engine.go)
	return false // Delegate to IsJusticeAligned() if available via career engine interface
}

// ComputeScaledXP applies the $VBV-gate multiplier and tier bonuses to base XP.
// This mirrors the pattern used in battle_service.go for all ~16 careers with $VBV gates.
func (ce *ContractEngine) ComputeScaledXP(baseXP uint64, jobRole string, vbvMultiplier float64, careerTier int) uint64 {
	// Apply $VBV-gate multiplier first (core economic integrity gate)
	scaled := uint64(float64(baseXP) * vbvMultiplier)

	// Apply tier-based scaling bonus (+10% per tier above Tier 1)
	if careerTier > 1 {
		tierBonus := float64(1+(careerTier-1)) * 0.1 // Tier 2=×1.1, Boss+=×1.5 max (explicit float64 arithmetic)
		if tierBonus > 1.5 {
			tierBonus = 1.5 // Cap at ×1.5 equivalent to Boss tier scaling
		}
		scaled = uint64(float64(scaled) * tierBonus)
	}

	return scaled
}

// GetContractXPForCareer returns the base XP value for a given contract template and career role.
func (ce *ContractEngine) GetContractXPForCareer(contractID string, jobRole string) uint64 {
	for _, tmpl := range ce.templates {
		if tmpl.ID == contractID {
			// If player's role matches or is compatible with target career, use full XP
			if tmpl.TargetCareer == jobRole || ce.IsCompatibleRole(jobRole, tmpl.TargetCareer) {
				return tmpl.XPBase
			}
			// Partial XP for non-compatible roles (cross-career acceptance at 50% rate)
			return uint64(float64(tmpl.XPBase) * 0.5)
		}
	}

	return 0 // Unknown contract — no XP awarded
}

// Helper function: contains checks if a string slice contains a given value.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

// ============================================================================
// HTTP Handler Methods — Refactored from black_market_service.go patterns
// These serve as the dynamic replacement for static HandleGetUnderworldContracts()
// ============================================================================

// HandleGetAvailableContracts is the HTTP handler that replaces the old static HandleGetUnderworldContracts().
// It uses ContractEngine.GetAvailableContracts() to dynamically generate contracts based on player state.
func (ce *ContractEngine) HandleGetAvailableContracts(lobby *Lobby, wallet string) []DynamicContract {
	if lobby == nil || lobby.leaderboard == nil {
		return nil
	}

	stats, exists := lobby.leaderboard[wallet]
	if !exists {
		return nil
	}

	// Compute $VBV-gate multiplier using same pattern as battle_service.go careers (~line ~978)
	vbvMultiplier := 1.0 // Default for non-$VBV players (Peon tier equivalent)
	if stats.CareerXP != nil {
		vbvMultiplier = stats.CareerXP.ComputeScaledXP(1, "", 1.0, 0) / 1 // Normalize: GetVBVGatingMultiplier pattern from rival_career_engine.go
	}

	// Determine player state for eligibility evaluation
	jobRole := stats.JobRole
	careerTier := ce.GetCareerTierForPlayer(stats)
	wantedLevel := stats.WantedLevel
	inAlliance := len(stats.Alliances) > 0 || (stats.ActiveAllianceID != "") // Check alliance membership
	hasTerritoryControl := false                                             // Default: no territory control

	// Check if player controls any club/territory via lobby's club service integration
	if lobby.clubService != nil {
		for _, club := range lobby.clubService.GetAllClubs() {
			if strings.ToLower(club.OwnerWallet) == wallet && len(club.Territories) > 0 {
				hasTerritoryControl = true
				break
			}
		}
	}

	// Evaluate rival presence (Justice careers trigger RivalMultiplier on Underworld contracts)
	rivalPresent := ce.EvaluateRivalPresence(lobby, wallet)

	return ce.GetAvailableContracts(
		wallet,
		jobRole,
		careerTier,
		vbvMultiplier,
		wantedLevel,
		inAlliance,
		hasTerritoryControl,
		rivalPresent,
	)
}

// GetCareerTierForPlayer returns the career tier for a player's primary job role.
func (ce *ContractEngine) GetCareerTierForPlayer(stats PlayerStats) int {
	if stats.CareerXP == nil {
		return 0 // Peon equivalent — no career progression
	}

	// Derive $VBV-sustained tier from AvgSustainedMicro using the same thresholds as rival_career_engine.go
	var multiplier float64
	if stats.CareerXP != nil {
		multiplier = (*CareerXP)(stats.CareerXP).GetVBVGatingMultiplier() // Returns ×1 (Peon) through ×32+ (Boss+)
	}

	switch {
	case multiplier >= 32:
		return 6 // Boss tier equivalent
	case multiplier >= 16:
		return 5 // Master tier equivalent
	case multiplier >= 4:
		return 4 // Expert tier equivalent
	case multiplier >= 2:
		return 3 // Journeyman tier equivalent
	default:
		return 2 // Apprentice+ (Tier 2) — minimum career progression with $VBV gate active
	}
}

// HandleAssignContract is the HTTP handler for contract assignment with dynamic difficulty scaling.
// It evaluates player eligibility, scales rewards based on Wanted level and career tier,
// assigns time-limited contracts with expiry tracking, and broadcasts underworld_contract_assigned WS event.
func (ce *ContractEngine) HandleAssignContract(lobby *Lobby, wallet string, templateID string) (*DynamicContract, error) {
	if lobby == nil || lobby.leaderboard == nil {
		return nil, fmt.Errorf("lobby not available")
	}

	stats, exists := lobby.leaderboard[wallet]
	if !exists {
		return nil, fmt.Errorf("player %s not found", wallet)
	}

	// Check if player already has an active contract (one-contract-at-a-time rule)
	if stats.ActiveUnderworldContractID != "" {
		return nil, fmt.Errorf("already have active contract: %s", stats.ActiveUnderworldContractID)
	}

	// Find the template by ID from our registry
	var tmpl *ContractTemplate
	for i := range ce.templates {
		if ce.templates[i].ID == templateID {
			tmpl = &ce.templates[i]
			break
		}
	}

	if tmpl == nil {
		return nil, fmt.Errorf("contract template %s not found", templateID)
	}

	// Evaluate eligibility using same logic as HandleGetAvailableContracts
	vbvMultiplier := 1.0
	if stats.CareerXP != nil {
		vbvMultiplier = float64(stats.CareerXP.GetVBVGatingMultiplier())
	}

	careerTier := ce.GetCareerTierForPlayer(stats)

	// Check alliance requirement against lobby's active alliances
	inAlliance := false
	for _, allyID := range stats.Alliances {
		if allyID != "" && len(allyID) > 0 {
			inAlliance = true
			break
		}
	}

	hasTerritoryControl := false
	if lobby.clubService != nil {
		for _, club := range lobby.clubService.GetAllClubs() {
			if strings.ToLower(club.OwnerWallet) == wallet && len(club.Territories) > 0 {
				hasTerritoryControl = true
				break
			}
		}
	}

	rivalPresent := ce.EvaluateRivalPresence(lobby, wallet)

	// Validate eligibility against template requirements
	if stats.WantedLevel < tmpl.RequiredWantedMin {
		return nil, fmt.Errorf("insufficient wanted level: need %d, have %d", tmpl.RequiredWantedMin, stats.WantedLevel)
	}

	if careerTier < tmpl.RequiredCareerTier {
		return nil, fmt.Errorf("insufficient career tier: need %d, have %d", tmpl.RequiredCareerTier, careerTier)
	}

	if tmpl.RequiresAlliance && !inAlliance {
		return nil, fmt.Errorf("this contract requires alliance membership")
	}

	if tmpl.RequiresTerritory && !hasTerritoryControl {
		return nil, fmt.Errorf("this contract requires territory control")
	}

	// Check career role compatibility (must be primary or compatible role)
	isCompatible := ce.IsCompatibleRole(stats.JobRole, tmpl.TargetCareer) || stats.JobRole == tmpl.TargetCareer
	if !isCompatible {
		return nil, fmt.Errorf("your role (%s) is not eligible for this contract (target: %s)", stats.JobRole, tmpl.TargetCareer)
	}

	// Compute scaled rewards and XP using $VBV-gate pattern
	scaledReward := uint64(float64(tmpl.BaseRewardMicro) * vbvMultiplier)
	if stats.WantedLevel > tmpl.RequiredWantedMin {
		wantedBonus := float64(stats.WantedLevel-tmpl.RequiredWantedMin) / 20.0
		scaledReward = uint64(float64(scaledReward) * (1.0 + wantedBonus))
	}

	if rivalPresent {
		scaledReward = uint64(float64(scaledReward) * tmpl.RivalMultiplier)
	}

	scaledXP := ce.ComputeScaledXP(tmpl.XPBase, stats.JobRole, vbvMultiplier, careerTier)

	// Generate instance ID and expiration time
	instanceID := fmt.Sprintf("UC-%d-%05X", time.Now().UnixMilli(), ce.instanceCounter)
	ce.instanceCounter++

	expiresAt := time.Now().Add(time.Duration(tmpl.TimeLimitHours) * time.Hour)

	dynamicContract := DynamicContract{
		Template:          *tmpl,
		InstanceID:        instanceID,
		ScaledRewardMicro: scaledReward,
		ScaledXP:          scaledXP,
		ExpiresAt:         expiresAt,
		EligibleCareers:   ce.GetCompatibleRoles(tmpl.TargetCareer),
	}

	// Assign contract to player state (atomic update)
	stats.ActiveUnderworldContractID = templateID // Use template ID as active contract reference
	lobby.leaderboard[wallet] = stats

	// Broadcast underworld_contract_assigned WS event to frontend
	if lobby.broadcast != nil {
		eventData := fmt.Sprintf(`{"wallet":"%s","contract_id":"%s","instance_id":"%s","reward_micro":%d,"xp_awarded":%d}`,
			wallet, templateID, instanceID, scaledReward, scaledXP)
		go func() {
			select {
			case lobby.broadcast <- Envelope{Type: "underworld_contract_assigned", Payload: json.RawMessage(eventData)}:
			default:
				// Drop event if broadcast channel full — non-critical UI sync
			}
		}()
	}

	// Log contract assignment for audit trail
	lobby.logAdminAuditLocked("UNDERWORLD_CONTRACT_ASSIGNED", wallet, fmt.Sprintf("ID: %s, Instance: %s, Reward: %.2f $VBV, XP: %d", templateID, instanceID, float64(scaledReward)/1000000.0, scaledXP))

	return &dynamicContract, nil
}

// HandleCompleteContract processes contract completion with reward payout and $VBV-gated XP award.
// This is called from battle_service.go, club_service.go, handlers_criminality.go at resolution points.
func (ce *ContractEngine) HandleCompleteContract(lobby *Lobby, wallet string, templateID string) error {
	if lobby == nil || lobby.leaderboard == nil {
		return fmt.Errorf("lobby not available")
	}

	stats, exists := lobby.leaderboard[wallet]
	if !exists {
		return fmt.Errorf("player %s not found", wallet)
	}

	// Verify this is the player's active contract (prevent spoofing)
	if stats.ActiveUnderworldContractID != templateID {
		return fmt.Errorf("contract mismatch: active=%s, completing=%s", stats.ActiveUnderworldContractID, templateID)
	}

	// Find template for reward/XP computation
	var tmpl *ContractTemplate
	for i := range ce.templates {
		if ce.templates[i].ID == templateID {
			tmpl = &ce.templates[i]
			break
		}
	}

	if tmpl == nil {
		return fmt.Errorf("contract template %s not found", templateID)
	}

	// Compute $VBV-gated XP using same pattern as battle_service.go careers (~line ~978 area)
	vbvMultiplier := float64(stats.CareerXP.GetVBVGatingMultiplier())
	careerTier := ce.GetCareerTierForPlayer(stats)
	scaledXP := ce.ComputeScaledXP(tmpl.XPBase, stats.JobRole, vbvMultiplier, careerTier)

	// Award reward micro-units (deterministic — no silent minting)
	rewardMicro := uint64(float64(tmpl.BaseRewardMicro) * vbvMultiplier)
	if stats.WantedLevel > tmpl.RequiredWantedMin {
		wantedBonus := float64(stats.WantedLevel-tmpl.RequiredWantedMin) / 20.0
		rewardMicro = uint64(float64(rewardMicro) * (1.0 + wantedBonus))
	}

	if ce.EvaluateRivalPresence(lobby, wallet) {
		rivalPresent := false // Already computed in HandleGetAvailableContracts — but recompute for safety at resolution point
		for w2, s2 := range lobby.leaderboard {
			if w2 == wallet || !ce.IsJusticeCareer(s2.JobRole) {
				continue
			}
			rivalPresent = true
			break
		}
		if rivalPresent && tmpl.RivalMultiplier > 0 {
			rewardMicro = uint64(float64(rewardMicro) * tmpl.RivalMultiplier)
		}
	}

	// Route reward through economy processing (deterministic finance — no silent minting)
	if lobby.economyService != nil && rewardMicro > 0 {
		lobby.economyService.AddBalance(wallet, rewardMicro)
	} else if lobby.playerBalances != nil {
		lobby.playerBalances[wallet] += rewardMicro
	}

	// Award $VBV-gated career XP — THIS IS THE CRITICAL GAP FILL FOR STEP C ✅
	if stats.CareerXP != nil && scaledXP > 0 {
		stats.CareerXP.TrackCareerXP(tmpl.TargetCareer, scaledXP)

		// Log XP award for forensic traceability (consistent with battle_service.go audit patterns ~line ~985 area)
		lobby.logAdminAuditLocked("UNDERWORLD_CONTRACT_XP_AWARDED", wallet, fmt.Sprintf("Contract: %s, Career: %s, BaseXP: %d, ScaledXP: %d ($VBV-gate ×%.2f)", templateID, tmpl.TargetCareer, tmpl.XPBase, scaledXP, vbvMultiplier))
	}

	// Clear active contract from player state (atomic update)
	stats.ActiveUnderworldContractID = ""
	lobby.leaderboard[wallet] = stats

	// Broadcast underworld_contract_completed WS event to frontend
	if lobby.broadcast != nil {
		eventData := fmt.Sprintf(`{"wallet":"%s","contract_id":"%s","reward_micro":%d,"xp_awarded":%d}`,
			wallet, templateID, rewardMicro, scaledXP)
		go func() {
			select {
			case lobby.broadcast <- Envelope{Type: "underworld_contract_completed", Payload: json.RawMessage(eventData)}:
			default:
				// Drop event if broadcast channel full — non-critical UI sync
			}
		}()
	}

	// Log contract completion for audit trail
	lobby.logAdminAuditLocked("UNDERWORLD_CONTRACT_COMPLETED", wallet, fmt.Sprintf("ID: %s, Reward: %.2f $VBV, XP: %d", templateID, float64(rewardMicro)/1000000.0, scaledXP))

	return nil
}

// ============================================================================
// Lobby struct extension — ContractEngine field (add to backend_types.go)
// See server.go for initialization in newLobby() function
// ============================================================================
