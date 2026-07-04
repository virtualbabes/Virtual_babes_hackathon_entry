//go:build !js && !wasm

package main

import (
	"fmt"
	"strings"
)

// PlayerService encapsulates logic for player attribute calculations and derived stats.
// PILLAR 5: Stateless Service Design.
type PlayerService struct{}

// GetHegemonyPath determines if a career role belongs to the legal or criminal layer.
// PILLAR 3: Identity & Modular Authority.
func (s *PlayerService) GetHegemonyPath(role string) string {
	switch role {
	case "Intel-Agent", "Bounty Hunter", "Armed-Offender-Squad", "Justice Recruiter", "Justice Commissioner", "Judge", "Warden", "Forensic Analyst", "Tax Auditor", "Sector Peacekeeper":
		return "JUSTICE"
	case "Gossip", "Fence", "Kidnapper", "Hostage Host", "Lawyer-Commissioner", "Underworld Boss", "Arc-Net Operative", "Smuggler", "Heist Planner", "Launderer":
		return "UNDERWORLD"
	default:
		return "NEUTRAL"
	}
}

// GetEffectiveCunning returns base cunning plus cosmetic bonuses and infamy penalties.
func (s *PlayerService) GetEffectiveCunning(stats PlayerStats) int {
	eff := stats.Cunning
	if stats.EquippedFaceplate != "" {
		if fp, exists := FaceplateRegistry[stats.EquippedFaceplate]; exists {
			eff += fp.CunningBonus
		}
	}
	penalty := stats.WantedLevel / 5
	if eff < penalty {
		return 0
	}
	return eff - penalty
}

// GetEffectiveMojo returns base mojo plus cosmetic bonuses.
// PILLAR 3: Career Role Weighting.
func (s *PlayerService) GetEffectiveMojo(stats PlayerStats) int {
	base := stats.Mojo
	if stats.EquippedFaceplate != "" {
		if fp, exists := FaceplateRegistry[stats.EquippedFaceplate]; exists {
			base += fp.MojoBonus
		}
	}

	// Underworld Weighting: Criminal prestige scales Mojo (Social Rank) more effectively.
	if s.GetHegemonyPath(stats.JobRole) == "UNDERWORLD" {
		// PILLAR 1: Underworld Loop. 20% bonus to social rank weighting.
		base = (base * 12) / 10
	}

	return base
}

// GetReputationWeighting returns the scaling factor for Standing based on the Hegemony path.
// PILLAR 2: Integer Supremacy. Returns a value representing a percentage (e.g., 110 = 1.1x).
func (s *PlayerService) GetReputationWeighting(role string) int {
	if s.GetHegemonyPath(role) == "JUSTICE" {
		// PILLAR 3: Justice Path scaling. Legal roles gain 10% higher standing weighting
		// to reflect their institutional authority within the simulation.
		return 110
	}
	return 100
}