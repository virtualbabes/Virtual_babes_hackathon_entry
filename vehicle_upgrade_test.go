//go:build !js && !wasm

package main

import (
	"testing"
)

// newTestLobbyForVehicles builds the minimum Lobby a vehicle-ladder test needs.
// The sink router comes from the production constructor so the fee path under test is
// the real one (a zero-value router has a nil Audit and would panic at the first route).
//
// §25.6.1 (2026-09-14): vehicles are PURCHASED account upgrades, so the helper funds the
// test owner — otherwise every spawn would (correctly) be refused for insufficient balance.
func newTestLobbyForVehicles(t *testing.T) *Lobby {
	t.Helper()
	var faucetPool, adminPool uint64
	return &Lobby{
		DataDir:         t.TempDir(),
		vehicles:        make(map[string]*VehicleNFT),
		playerBalances:  map[string]uint64{"OWNER": 1_000_000 * 1_000_000},
		tokenSinkRouter: NewTokenSinkRouter(&faucetPool, &adminPool),
	}
}

// TestVehicleUpgradeLadder_SpawnProvenance pins the §25.6.1 spawn contract: a legitimate
// build is purchased at its CLASS price, is certified, starts at build level 1 from the
// class stat floor, carries the break-in window, and the class — not the caller — owns the
// Level gate.
func TestVehicleUpgradeLadder_SpawnProvenance(t *testing.T) {
	l := newTestLobbyForVehicles(t)

	bal := l.playerBalances["OWNER"]
	// 999 is deliberately absurd: the FLYER class owns the gate (5), so the request is ignored.
	v, err := l.SpawnVehicle("OWNER", "Ride", "FLYER", 999)
	if err != nil {
		t.Fatalf("SpawnVehicle: %v", err)
	}
	flyer, ok := lookupVehicleKind("FLYER")
	if !ok {
		t.Fatal("FLYER class missing from the table")
	}
	if spent := bal - l.playerBalances["OWNER"]; spent != flyer.FeeMicro {
		t.Errorf("purchase charged %d, want the class price %d", spent, flyer.FeeMicro)
	}
	if !v.Certified {
		t.Error("a legitimately spawned vehicle must be Certified")
	}
	if v.BlackMarketAdopted {
		t.Error("a legitimately spawned vehicle must not be BlackMarketAdopted")
	}
	if v.VehicleLevel != 1 {
		t.Errorf("build level: got %d want 1", v.VehicleLevel)
	}
	if v.Stats.StatSum() != flyer.Base.StatSum() {
		t.Errorf("class stat floor: got sum %d want %d", v.Stats.StatSum(), flyer.Base.StatSum())
	}
	if v.MinLevel != flyer.MinLevel {
		t.Errorf("min level: got %d want %d (caller-supplied gate must be ignored)", v.MinLevel, flyer.MinLevel)
	}
	if v.BreakInMs != VehicleBreakInMs {
		t.Errorf("break-in window: got %d want %d", v.BreakInMs, VehicleBreakInMs)
	}
	if got := l.GetVehicles("OWNER"); len(got) != 1 {
		t.Errorf("owner lookup: got %d vehicles want 1", len(got))
	}
	if got := l.GetVehicles("SOMEONE"); len(got) != 0 {
		t.Errorf("foreign lookup: got %d vehicles want 0", len(got))
	}
}

// TestVehicleUpgradeLadder_Gates pins every refusal. A gate that silently passes is an
// economic hole, so each one is asserted to leave the vehicle untouched.
func TestVehicleUpgradeLadder_Gates(t *testing.T) {
	l := newTestLobbyForVehicles(t)
	const owner = "OWNER"
	v, err := l.SpawnVehicle(owner, "Ride", "GROUND", 1)
	if err != nil {
		t.Fatalf("SpawnVehicle: %v", err)
	}
	before := v.Stats.StatSum()

	refusals := []struct {
		name    string
		owner   string
		id      string
		part    string
		prepare func()
	}{
		{"unknown part", owner, v.VehicleID, "TURBO", nil},
		{"missing vehicle", owner, "VEH-does-not-exist", "ENGINE", nil},
		{"non-owner", "SOMEONE_ELSE", v.VehicleID, "ENGINE", nil},
		{"unfunded", owner, v.VehicleID, "ENGINE", func() {
			// The helper funds the owner so purchases succeed; an upgrade still needs funds.
			l.playerBalances[owner] = 0
		}},
		{"black market", owner, v.VehicleID, "ENGINE", func() {
			l.vehicles[v.VehicleID].BlackMarketAdopted = true
			l.playerBalances[owner] = VehicleUpgradeBaseMicro * 10
		}},
		{"uncertified", owner, v.VehicleID, "ENGINE", func() {
			l.vehicles[v.VehicleID].BlackMarketAdopted = false
			l.vehicles[v.VehicleID].Certified = false
		}},
	}
	for _, tc := range refusals {
		if tc.prepare != nil {
			tc.prepare()
		}
		if _, _, _, err := l.UpgradeVehicle(tc.owner, tc.id, tc.part); err == nil {
			t.Errorf("%s: upgrade was ACCEPTED but must be refused", tc.name)
		}
	}

	// Restore legitimacy; the ladder must still refuse an over-cap upgrade.
	l.vehicles[v.VehicleID].Certified = true
	l.playerBalances[owner] = VehicleUpgradeBaseMicro * 10000
	l.vehicles[v.VehicleID].Upgrades["ENGINE"] = VehiclePartMaxLevel
	if _, _, _, err := l.UpgradeVehicle(owner, v.VehicleID, "ENGINE"); err == nil {
		t.Error("part at VehiclePartMaxLevel: upgrade was ACCEPTED but must be refused")
	}
	if l.vehicles[v.VehicleID].Stats.StatSum() < before {
		t.Error("a refused upgrade must never reduce the stat vector")
	}
}

// TestVehicleUpgradeLadder_DeterministicProgress pins the happy path: one axis per part,
// the calibrated integer fee debited from the player, the build level derived from the
// parts table, and the hard clamp at StatMax.
func TestVehicleUpgradeLadder_DeterministicProgress(t *testing.T) {
	l := newTestLobbyForVehicles(t)
	const owner = "OWNER"
	v, err := l.SpawnVehicle(owner, "Ride", "GROUND", 1)
	if err != nil {
		t.Fatalf("SpawnVehicle: %v", err)
	}

	// Level-0 ENGINE costs exactly base×1 and adds exactly VehicleStatGain to Speed.
	l.playerBalances[owner] = VehiclePartCostMicro(0)
	got, fee, applied, err := l.UpgradeVehicle(owner, v.VehicleID, "engine")
	if err != nil {
		t.Fatalf("UpgradeVehicle(ENGINE): %v", err)
	}
	// Expectations are derived from the CLASS table, so the ladder math is pinned without
	// hard-coding a stat floor that a future calibration change would silently invalidate.
	ground, ok := lookupVehicleKind("GROUND")
	if !ok {
		t.Fatal("GROUND class missing from the table")
	}
	if fee != VehicleUpgradeBaseMicro {
		t.Errorf("fee: got %d want %d", fee, VehicleUpgradeBaseMicro)
	}
	if applied != VehicleStatGain {
		t.Errorf("stat gain: got %d want %d", applied, VehicleStatGain)
	}
	if got.Stats.Speed != ground.Base.Speed+VehicleStatGain {
		t.Errorf("speed: got %d want %d", got.Stats.Speed, ground.Base.Speed+VehicleStatGain)
	}
	if got.Stats.Strength != ground.Base.Strength {
		t.Errorf("a part must only touch its own axis; strength got %d want %d", got.Stats.Strength, ground.Base.Strength)
	}
	if got.VehicleLevel != 2 {
		t.Errorf("build level: got %d want 2", got.VehicleLevel)
	}
	if got.Upgrades["ENGINE"] != 1 {
		t.Errorf("part level: got %d want 1", got.Upgrades["ENGINE"])
	}
	if bal := l.playerBalances[owner]; bal != 0 {
		t.Errorf("fee debit: balance got %d want 0 (the fee must leave the player)", bal)
	}

	// Determinism: replaying the same sequence on a fresh lobby must produce identical state.
	l2 := newTestLobbyForVehicles(t)
	v2, _ := l2.SpawnVehicle(owner, "Ride", "GROUND", 1)
	l2.playerBalances[owner] = VehicleUpgradeBaseMicro * 100
	if _, _, _, err := l2.UpgradeVehicle(owner, v2.VehicleID, "ENGINE"); err != nil {
		t.Fatalf("replay upgrade: %v", err)
	}
	if v2.Stats.Speed != got.Stats.Speed || v2.VehicleLevel != got.VehicleLevel || v2.Upgrades["ENGINE"] != got.Upgrades["ENGINE"] {
		t.Errorf("non-deterministic ladder: %+v vs %+v", v2, got)
	}

	// StatMax is a hard ceiling: a part at the ceiling reports the CLAMPED gain, never an overflow.
	l.playerBalances[owner] = VehicleUpgradeBaseMicro * 100
	v.Stats.Speed = StatMax - 1
	_, _, applied, err = l.UpgradeVehicle(owner, v.VehicleID, "ENGINE")
	if err != nil {
		t.Fatalf("ceiling upgrade: %v", err)
	}
	if applied != 1 {
		t.Errorf("clamped gain: got %d want 1", applied)
	}
	if v.Stats.Speed != StatMax {
		t.Errorf("ceiling: speed got %d want %d", v.Stats.Speed, StatMax)
	}

	// Every part in the table resolves and maps to its own axis.
	l.playerBalances[owner] = VehicleUpgradeBaseMicro * 1000
	for _, d := range vehiclePartDefs {
		if _, _, _, err := l.UpgradeVehicle(owner, v.VehicleID, d.Part); err != nil {
			t.Errorf("part %s (%s) refused: %v", d.Part, d.Axis, err)
		}
	}
	if v.Stats.StatSum() < 6 {
		t.Errorf("stat vector collapsed below the floor: %d", v.Stats.StatSum())
	}
	if v.VehicleLevel != vehicleBuildLevelLocked(v) {
		t.Errorf("build level drifted from the parts table: %d vs %d", v.VehicleLevel, vehicleBuildLevelLocked(v))
	}
}

// TestVehicleUpgradeLadder_Deploy pins the region binding that feeds the §30 power overlay.
func TestVehicleUpgradeLadder_Deploy(t *testing.T) {
	l := newTestLobbyForVehicles(t)
	const owner = "OWNER"
	v, err := l.SpawnVehicle(owner, "Ride", "DIGGER", 1)
	if err != nil {
		t.Fatalf("SpawnVehicle: %v", err)
	}
	if _, err := l.DeployVehicle(owner, v.VehicleID, ""); err == nil {
		t.Error("empty region: deploy was ACCEPTED but must be refused")
	}
	if _, err := l.DeployVehicle("SOMEONE_ELSE", v.VehicleID, "Base"); err == nil {
		t.Error("non-owner: deploy was ACCEPTED but must be refused")
	}
	if _, err := l.DeployVehicle(owner, "VEH-nope", "Base"); err == nil {
		t.Error("missing vehicle: deploy was ACCEPTED but must be refused")
	}
	got, err := l.DeployVehicle(owner, v.VehicleID, "Base")
	if err != nil {
		t.Fatalf("DeployVehicle: %v", err)
	}
	if got.Region != "Base" {
		t.Errorf("region: got %q want %q", got.Region, "Base")
	}
	// Provenance feeds the market listing, so rarity must be class-derived, not constant.
	if r := vehicleRarity("DIGGER"); r != "epic" {
		t.Errorf("DIGGER rarity: got %q want epic", r)
	}
	if r := vehicleRarity("FLYER"); r != "rare" {
		t.Errorf("FLYER rarity: got %q want rare", r)
	}
	if r := vehicleRarity("GROUND"); r != "common" {
		t.Errorf("GROUND rarity: got %q want common", r)
	}
	if len(VehiclePartTableView()) != len(vehiclePartDefs) {
		t.Error("the published part table must match the authoritative table")
	}
}
