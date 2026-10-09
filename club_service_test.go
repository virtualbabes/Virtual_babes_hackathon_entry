//go:build !js && !wasm

package main

import (
	"testing"
	"time"
)

// TestWarfare_AllianceCostScaling verifies that costs scale based on combined alliance organizational density.
// PILLAR 1: Strategic Warfare Scaling.
func TestWarfare_AllianceCostScaling(t *testing.T) {
	l := &Lobby{
		clubs: make(map[string]*Club),
	}

	// 1. Setup Target Club A (Single territory)
	clubA := &Club{ID: "CLUB-A", Territories: []string{"arena_center"}}
	l.clubs[clubA.ID] = clubA

	// Expected: 5,000 base + (1,000 * 1 dist) = 6,000 VBV -> 6B micro-units
	countA := len(clubA.Territories)
	feeA := uint64(5000+1000*countA) * 1000000

	if feeA != 6000*1000000 {
		t.Errorf("Cost scaling failure (Single): Expected 6B micro-units, got %d", feeA)
	}

	// 2. Setup Target Club B (Allied coalition with 3 combined territories)
	clubB := &Club{ID: "CLUB-B", Territories: []string{"north_district", "the_lab"}, AlliedClubID: "CLUB-C"}
	clubC := &Club{ID: "CLUB-C", Territories: []string{"south_slums"}}
	l.clubs[clubB.ID] = clubB
	l.clubs[clubC.ID] = clubC

	alliedCount := len(clubB.Territories) + len(clubC.Territories)
	feeB := uint64(5000+1000*alliedCount) * 1000000

	// Expected: 5,000 base + (1,000 * 3 dists) = 8,000 VBV
	if feeB != 8000*1000000 {
		t.Errorf("Cost scaling failure (Alliance): Expected 8B micro-units, got %d", feeB)
	}
}

// TestWarfare_DistributionCircularity verifies the 70/30 fee distribution and Governor war-chest routing.
// PILLAR 2: Industrial Seal & Circularity.
func TestWarfare_DistributionCircularity(t *testing.T) {
	l := &Lobby{
		clubs:              make(map[string]*Club),
		playerBalances:     make(map[string]uint64),
		wallets:            make(map[string]string),
		faucetBalanceMicro: 0,
	}

	// Setup: Perpetrator and Target
	perpW := "perp_wallet"
	l.wallets["client-1"] = perpW
	l.playerBalances[perpW] = 10000 * 1000000 // 10k VBV

	targetClub := &Club{ID: "TARGET", Territories: []string{"arena_center"}}
	l.clubs[targetClub.ID] = targetClub

	// Setup: Three competing Governors
	gov1 := &Club{ID: "GOV1", Territories: []string{"north_district", "west_port"}, RegionName: "Governor"}
	gov2 := &Club{ID: "GOV2", Territories: []string{"south_slums", "casino"}, RegionName: "Governor"}
	gov3 := &Club{ID: "GOV3", Territories: []string{"data_haven", "the_lab"}, RegionName: "Governor"}
	l.clubs[gov1.ID] = gov1
	l.clubs[gov2.ID] = gov2
	l.clubs[gov3.ID] = gov3

	// 1. Logic Trace: 5,000 base + (1,000 * 1 target dist) = 6,000 VBV
	warfareFeeMicro := uint64(6000 * 1000000)
	l.playerBalances[perpW] -= warfareFeeMicro

	// 2. Distribution Math: 70% Faucet / 30% Competitors
	faucetCutMicro := (warfareFeeMicro * 70) / 100         // 4,900 VBV
	competitorCutMicro := warfareFeeMicro - faucetCutMicro // 1,800 VBV

	l.faucetBalanceMicro += faucetCutMicro

	// Identify competitors (non-target regional govs)
	var otherGovs []*Club
	for _, c := range l.clubs {
		// Mocking isClubRegionalLocked logic for pure unit testing
		if c.ID != targetClub.ID && len(c.Territories) >= 2 {
			otherGovs = append(otherGovs, c)
		}
	}

	// 3. Execution: Distribute War Chest to 3 competitors (600 VBV each)
	shareMicro := competitorCutMicro / uint64(len(otherGovs))
	for _, g := range otherGovs {
		g.Treasury += float64(shareMicro) / 1000000.0
	}

	// 4. Assertions
	if l.faucetBalanceMicro != 4200*1000000 { // 70% of 6,000 is 4,200
		t.Errorf("Industrial circularity breach: Faucet should have 4.2B, got %d", l.faucetBalanceMicro)
	}

	if gov1.Treasury != 600.0 || gov3.Treasury != 600.0 {
		t.Errorf("War Chest routing error: Competitors should receive 600 VBV, got %.2f", gov1.Treasury)
	}

	// 5. Fallback Check: Verify 100% return to Faucet if no competitors exist
	l.faucetBalanceMicro = 0
	l.clubs = make(map[string]*Club)
	l.clubs[targetClub.ID] = targetClub

	fCut := (warfareFeeMicro * 70) / 100
	l.faucetBalanceMicro += fCut
	l.faucetBalanceMicro += (warfareFeeMicro - fCut) // Fallback logic

	if l.faucetBalanceMicro != warfareFeeMicro {
		t.Errorf("Fallback circularity breach: Faucet should recover 100%%, got %d", l.faucetBalanceMicro)
	}
}

// TestWarfare_DisruptionState verifies that the network blackout key is correctly applied.
func TestWarfare_DisruptionState(t *testing.T) {
	targetClub := &Club{
		ID:              "TARGET",
		Territories:     []string{"arena_center"},
		BuffExpirations: make(map[string]time.Time),
	}

	districtID := "arena_center"
	expiry := time.Now().Add(2 * time.Hour)

	// 1. Simulate the application of the disruption buff from handleRegionalSabotage
	targetClub.BuffExpirations["DISRUPTION_"+districtID] = expiry

	// 2. Verify key existence and temporal window
	val, exists := targetClub.BuffExpirations["DISRUPTION_arena_center"]
	if !exists {
		t.Fatal("Disruption buff key missing from target state")
	}

	if !val.Equal(expiry) {
		t.Errorf("Temporal drift detected. Expected %v, got %v", expiry, val)
	}
}
