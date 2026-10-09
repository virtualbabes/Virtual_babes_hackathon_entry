//go:build !js && !wasm

package main

import (
	"testing"
	"time"
)

// newTestLobbyForBonded builds the minimum Lobby the §25.6.1 / §26.4.2 bonded-asset tests
// need. The router comes from the production constructor so the FEE PATH under test is the
// real one, and the faucet pointer is returned so a test can prove the fee was routed to the
// sink rather than silently burned (Architecture Ledger: the Industrial Loop reconciles).
func newTestLobbyForBonded(t *testing.T) (*Lobby, *uint64) {
	t.Helper()
	var faucetPool, adminPool uint64
	l := &Lobby{
		DataDir:         t.TempDir(),
		pets:            make(map[string]*PetNFT),
		vehicles:        make(map[string]*VehicleNFT),
		playerBalances:  make(map[string]uint64),
		leaderboard:     make(map[string]PlayerStats),
		tokenSinkRouter: NewTokenSinkRouter(&faucetPool, &adminPool),
	}
	l.leaderboard["OWNER"] = PlayerStats{}
	return l, &faucetPool
}

// allStats builds a stat vector with one value on every axis (test convenience).
func allStats(v uint64) EntityStats {
	return EntityStats{Speed: v, Intelligence: v, Willpower: v, Strength: v, Charisma: v, Agility: v}
}

// TestBondedAssetPurchase_PetSpawn pins §26.4.2: a companion is BOUGHT. The price is the
// server constant, it leaves the owner's balance exactly, and it arrives at the faucet sink.
func TestBondedAssetPurchase_PetSpawn(t *testing.T) {
	l, faucet := newTestLobbyForBonded(t)
	const owner = "OWNER"
	l.playerBalances[owner] = 10_000 * 1_000_000
	before := l.playerBalances[owner]

	// Trait bits all set: 2 bits per axis → every axis would be 4, capped at PetSpawnBaseStatMax.
	p, err := l.SpawnPet(owner, "Rex", 0xFFF) // every axis' 2-bit field set → capped floor
	if err != nil {
		t.Fatalf("SpawnPet: %v", err)
	}
	if spent := before - l.playerBalances[owner]; spent != PetSpawnFeeMicro {
		t.Errorf("owner was charged %d micro-$VBV, want %d", spent, PetSpawnFeeMicro)
	}
	if *faucet != PetSpawnFeeMicro {
		t.Errorf("faucet sink received %d, want %d (fee must be routed, never burned)", *faucet, PetSpawnFeeMicro)
	}
	if !p.Certified || p.BlackMarketAdopted {
		t.Errorf("provenance: certified=%v blackMarket=%v, want true/false", p.Certified, p.BlackMarketAdopted)
	}
	if p.PetLevel != 1 {
		t.Errorf("level: got %d want 1", p.PetLevel)
	}
	if p.Grooming == nil {
		t.Error("grooming map must be initialised so the ladder has a baseline")
	}
	if want := PetSpawnBaseStatMax * 6; p.Stats.StatSum() != want {
		t.Errorf("stat floor: got sum %d want %d (capped per axis)", p.Stats.StatSum(), want)
	}
	if got := l.GetPets(owner); len(got) != 1 {
		t.Errorf("owner lookup: got %d companions want 1", len(got))
	}
	if got := l.GetPets("SOMEONE"); len(got) != 0 {
		t.Errorf("foreign lookup: got %d companions want 0", len(got))
	}
}

// TestBondedAssetPurchase_RefusedWithoutFunds pins that an unfunded purchase is refused and
// costs nothing — no companion, no sink credit, unchanged balance.
func TestBondedAssetPurchase_RefusedWithoutFunds(t *testing.T) {
	l, faucet := newTestLobbyForBonded(t)
	const owner = "OWNER"
	l.playerBalances[owner] = PetSpawnFeeMicro - 1

	if _, err := l.SpawnPet(owner, "Rex", 0); err == nil {
		t.Fatal("purchase with insufficient funds must fail")
	}
	if l.playerBalances[owner] != PetSpawnFeeMicro-1 {
		t.Error("a refused purchase must not move money")
	}
	if *faucet != 0 {
		t.Errorf("refused purchase credited the sink %d", *faucet)
	}
	if len(l.GetPets(owner)) != 0 {
		t.Error("refused purchase created a companion")
	}

	// A vehicle class is priced too: the cheapest class must also refuse cleanly.
	l.playerBalances[owner] = 1
	if _, err := l.SpawnVehicle(owner, "Ride", "GROUND", 0); err == nil {
		t.Fatal("vehicle purchase with insufficient funds must fail")
	}
	if len(l.GetVehicles(owner)) != 0 {
		t.Error("refused vehicle purchase created a vehicle")
	}
}
// TestPetGroomLadder pins §26.4.2 — the mirror of §25.6.1: linear integer price, +stat per
// level, derived level, per-axis ceiling, and a refusal path that never charges.
func TestPetGroomLadder(t *testing.T) {
	l, _ := newTestLobbyForBonded(t)
	const owner = "OWNER"
	l.playerBalances[owner] = 50_000 * 1_000_000

	p, err := l.SpawnPet(owner, "Rex", 0) // traits 0 → floor 1 per axis
	if err != nil {
		t.Fatalf("SpawnPet: %v", err)
	}
	bal := l.playerBalances[owner]

	got, fee, applied, err := l.GroomPet(owner, p.PetID, "speed")
	if err != nil {
		t.Fatalf("GroomPet: %v", err)
	}
	if fee != PetGroomCostMicro(0) {
		t.Errorf("first groom fee: got %d want %d", fee, PetGroomCostMicro(0))
	}
	if applied != PetGroomStatGain {
		t.Errorf("stat applied: got %d want %d", applied, PetGroomStatGain)
	}
	if bal-l.playerBalances[owner] != fee {
		t.Errorf("owner charged %d, want %d", bal-l.playerBalances[owner], fee)
	}
	if got.Grooming["SPEED"] != 1 {
		t.Errorf("SPEED level: got %d want 1", got.Grooming["SPEED"])
	}
	if got.PetLevel != 2 {
		t.Errorf("derived level: got %d want 2 (1 + Σ grooming)", got.PetLevel)
	}
	if got.Stats.Speed != 1+PetGroomStatGain {
		t.Errorf("SPEED stat: got %d want %d", got.Stats.Speed, 1+PetGroomStatGain)
	}

	// Second level costs base*2 (linear, deterministic).
	if _, fee2, _, err := l.GroomPet(owner, p.PetID, "SPEED"); err != nil {
		t.Fatalf("second groom: %v", err)
	} else if fee2 != PetGroomCostMicro(1) {
		t.Errorf("second groom fee: got %d want %d", fee2, PetGroomCostMicro(1))
	}

	// Fill the axis to its ceiling, then prove the ceiling bites and costs nothing.
	for lvl := p.Grooming["SPEED"]; lvl < PetGroomMaxLevel; lvl++ {
		if _, _, _, err := l.GroomPet(owner, p.PetID, "speed"); err != nil {
			t.Fatalf("groom to ceiling: %v", err)
		}
	}
	balAtCap := l.playerBalances[owner]
	if _, _, _, err := l.GroomPet(owner, p.PetID, "SPEED"); err == nil {
		t.Error("grooming past the per-axis ceiling must be refused")
	}
	if l.playerBalances[owner] != balAtCap {
		t.Error("a refused groom must not move money")
	}
	if p.PetLevel != 1+PetGroomMaxLevel {
		t.Errorf("level at ceiling: got %d want %d", p.PetLevel, 1+PetGroomMaxLevel)
	}

	// Unknown focus and foreign ownership are both refused without charge.
	balBefore := l.playerBalances[owner]
	if _, _, _, err := l.GroomPet(owner, p.PetID, "TURBO"); err == nil {
		t.Error("unknown grooming focus must be refused")
	}
	if _, _, _, err := l.GroomPet("SOMEONE", p.PetID, "agility"); err == nil {
		t.Error("grooming another wallet's companion must be refused")
	}
	if l.playerBalances[owner] != balBefore {
		t.Error("refusals must not move money")
	}

	// §27.7.3: an off-ledger companion is ineligible for legitimate progression.
	p.BlackMarketAdopted = true
	if _, _, _, err := l.GroomPet(owner, p.PetID, "agility"); err == nil {
		t.Error("black-market companions must be ineligible for grooming")
	}
}

// TestPetBreedServerFee pins that the breeding price is server-authoritative (a client cannot
// waive it) and that the offspring inherits the MEAN of both parents (§26.4.2 lineage).
func TestPetBreedServerFee(t *testing.T) {
	l, faucet := newTestLobbyForBonded(t)
	const owner = "OWNER"
	l.playerBalances[owner] = 100_000 * 1_000_000

	old := time.Now().Add(-40 * 24 * time.Hour) // past the 30-day maturity gate
	mk := func(id string, v uint64) {
		l.pets[id] = &PetNFT{
			PetID: id, Owner: owner, Name: id, BirthAt: old,
			MaturityMs: 30 * 24 * 60 * 60 * 1000,
			Certified:  true, Stats: allStats(v), PetLevel: 1, Grooming: map[string]uint64{},
		}
	}
	mk("S1", 10)
	mk("D1", 20)

	bal := l.playerBalances[owner]
	faucetBefore := *faucet
	off, err := l.BreedPet(owner, "Pup", "S1", "D1")
	if err != nil {
		t.Fatalf("BreedPet: %v", err)
	}
	if bal-l.playerBalances[owner] != PetBreedFeeMicro {
		t.Errorf("breeding charged %d, want the server constant %d", bal-l.playerBalances[owner], PetBreedFeeMicro)
	}
	if *faucet-faucetBefore != PetBreedFeeMicro {
		t.Errorf("breeding fee routed %d to the sink, want %d", *faucet-faucetBefore, PetBreedFeeMicro)
	}
	if off.SireID != "S1" || off.DamID != "D1" {
		t.Errorf("lineage: sire=%q dam=%q", off.SireID, off.DamID)
	}
	if want := uint64(15); off.Stats.StatSum() != want*6 {
		t.Errorf("inherited stats: got sum %d want %d (mean of both parents)", off.Stats.StatSum(), want*6)
	}
	if off.PetLevel != 1 || len(off.Grooming) != 0 {
		t.Errorf("offspring must start a fresh ladder: level=%d grooming=%v", off.PetLevel, off.Grooming)
	}
	if !off.Certified || off.BlackMarketAdopted {
		t.Error("offspring of certified parents must be certified")
	}

	// Off-ledger lineage is ineligible: no charge, no offspring.
	l.pets["D1"].BlackMarketAdopted = true
	balBefore := l.playerBalances[owner]
	petsBefore := len(l.GetPets(owner))
	if _, err := l.BreedPet(owner, "Bad", "S1", "D1"); err == nil {
		t.Fatal("black-market lineage must be ineligible for legitimate breeding")
	}
	if l.playerBalances[owner] != balBefore || len(l.GetPets(owner)) != petsBefore {
		t.Error("a refused breeding must cost nothing and create nothing")
	}
}
// TestVehiclePurchaseClasses pins §25.6.1: the CLASS — never the request body — owns the
// price, the Level gate and the delivered stat floor, and an unknown class is refused free.
func TestVehiclePurchaseClasses(t *testing.T) {
	l, faucet := newTestLobbyForBonded(t)
	const owner = "OWNER"
	l.playerBalances[owner] = 100_000 * 1_000_000

	for _, def := range vehicleKindDefs {
		bal := l.playerBalances[owner]
		sink := *faucet
		// The caller asks for an absurd gate; the class must override it.
		v, err := l.SpawnVehicle(owner, "Ride-"+def.Kind, def.Kind, 999)
		if err != nil {
			t.Fatalf("SpawnVehicle(%s): %v", def.Kind, err)
		}
		if bal-l.playerBalances[owner] != def.FeeMicro {
			t.Errorf("%s charged %d, want %d", def.Kind, bal-l.playerBalances[owner], def.FeeMicro)
		}
		if *faucet-sink != def.FeeMicro {
			t.Errorf("%s routed %d to the sink, want %d", def.Kind, *faucet-sink, def.FeeMicro)
		}
		if v.MinLevel != def.MinLevel {
			t.Errorf("%s min level: got %d want %d (caller-supplied gate must be ignored)", def.Kind, v.MinLevel, def.MinLevel)
		}
		if v.Stats != def.Base {
			t.Errorf("%s stat floor: got %+v want %+v", def.Kind, v.Stats, def.Base)
		}
		if !v.Certified || v.VehicleLevel != 1 || v.BreakInMs != VehicleBreakInMs {
			t.Errorf("%s provenance/build/break-in wrong: %+v", def.Kind, v)
		}
	}

	// Unknown class: refused, and no money moves.
	bal := l.playerBalances[owner]
	if _, err := l.SpawnVehicle(owner, "Weird", "SUBMARINE", 1); err == nil {
		t.Fatal("an unknown vehicle class must be refused")
	}
	if l.playerBalances[owner] != bal {
		t.Error("a refused class must not move money")
	}
}

// TestBondedDeckBoostPercent pins §26.4.3 / §25.6.2: the household stat total grants a capped
// integer percentage to the owner's deck cards, only from LEGITIMATE assets, deterministically.
func TestBondedDeckBoostPercent(t *testing.T) {
	l, _ := newTestLobbyForBonded(t)
	const owner = "OWNER"

	if got := l.BondedDeckBoostPct(owner); got != 0 {
		t.Errorf("no assets: got %d%% want 0", got)
	}

	// One companion at 30 per axis → stat sum 180 → 180/BondedDeckBoostDivisor.
	l.pets["P1"] = &PetNFT{PetID: "P1", Owner: owner, Certified: true, Stats: allStats(30)}
	want := int((30 * 6) / BondedDeckBoostDivisor)
	if got := l.BondedDeckBoostPct(owner); got != want {
		t.Errorf("one companion: got %d%% want %d%%", got, want)
	}

	// A certified vehicle adds to the same roll-up.
	l.vehicles["V1"] = &VehicleNFT{VehicleID: "V1", Owner: owner, Certified: true, Stats: allStats(30)}
	if got := l.BondedDeckBoostPct(owner); got != want*2 {
		t.Errorf("companion + vehicle: got %d%% want %d%%", got, want*2)
	}

	// The percentage is capped (a large household cannot exceed the ceiling).
	for i := 0; i < 6; i++ {
		l.pets[string(rune('A'+i))] = &PetNFT{PetID: string(rune('A' + i)), Owner: owner, Certified: true, Stats: allStats(StatMax)}
	}
	if got := l.BondedDeckBoostPct(owner); got != int(BondedDeckBoostMaxPct) {
		t.Errorf("large household: got %d%% want the cap %d%%", got, BondedDeckBoostMaxPct)
	}

	// §27.7.3 legitimacy: off-ledger assets do not fight for their owner.
	for _, p := range l.pets {
		p.BlackMarketAdopted = true
	}
	l.vehicles["V1"].BlackMarketAdopted = true
	if got := l.BondedDeckBoostPct(owner); got != 0 {
		t.Errorf("black-market household: got %d%% want 0%%", got)
	}

	// Another wallet gets nothing from these assets.
	if got := l.BondedDeckBoostPct("SOMEONE"); got != 0 {
		t.Errorf("foreign wallet: got %d%% want 0%%", got)
	}
	if got := l.BondedDeckBoostPct(""); got != 0 {
		t.Errorf("empty wallet: got %d%% want 0%%", got)
	}
}

