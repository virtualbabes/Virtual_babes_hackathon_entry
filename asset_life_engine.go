//go:build !js && !wasm

package main

// asset_life_engine.go
// §26.4 Companion Pets (PetNFT) — breedable, maturity-gated.
// §25.6 Vehicles (VehicleNFT) — level-gated, custom-named.
// §25.7 World Content (WorldContentNFT) — native entities, deployable to regions.
// All are bonded assets (§23) and use deterministic integer math (PILLAR 2). No cloud RNG.

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ── §26.4 Pets ─────────────────────────────────────────────────────────────────

// BreedPet deterministically produces an offspring from sire+dam (PILLAR 2 integer math).
// Trait bits: even-indexed bits from sire, odd from dam; recessive masked by maturity.
//
// §26.4.2: the breeding fee is SERVER-authoritative (PetBreedFeeMicro) — the handler ignores
// any fee in the request body, because a client must never price its own account upgrade. The
// offspring inherits a stat floor from the MEAN of both parents, so a groomed + bred lineage
// compounds: that is the whole point of pairing the breeding ladder with the grooming ladder.
func (l *Lobby) BreedPet(owner, name, sireID, damID string) (*PetNFT, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	sire, okS := l.pets[sireID]
	dam, okD := l.pets[damID]
	if !okS || !okD {
		return nil, errors.New("sire or dam not found")
	}
	if sire.Owner != owner || dam.Owner != owner {
		return nil, errors.New("you must own both parents")
	}
	// §27.7.3 / §26.4.1: legitimate breeding requires certified parents. A black-market-adopted
	// parent is ineligible for legitimate breeding — its lineage cannot produce certified offspring.
	if sire.BlackMarketAdopted || dam.BlackMarketAdopted {
		return nil, errors.New("black-market lineage is ineligible for legitimate breeding (§26.4.1)")
	}
	if !sire.Certified || !dam.Certified {
		return nil, errors.New("both parents must be certified for legitimate breeding (§26.4.1)")
	}
	// Maturity gate: both parents must be mature to breed (§26.4 / §25.6 riding-age parallel).
	if !l.petMatureLocked(sire) || !l.petMatureLocked(dam) {
		return nil, errors.New("both parents must be mature to breed")
	}
	// PURCHASE (server-authoritative price). Charged only after every gate above has passed,
	// so a refused breeding can never cost a player a single micro-unit.
	if err := l.chargeBondedAssetFeeLocked(owner, "PET_BREED_FEE", PetBreedFeeMicro); err != nil {
		return nil, err
	}
	traits := (sire.Traits & 0x5555555555555555) | (dam.Traits & 0xAAAAAAAAAAAAAAAA)
	offspring := &PetNFT{
		PetID:      "PET-" + uuid.New().String(),
		Owner:      owner,
		Name:       name,
		SireID:     sireID,
		DamID:      damID,
		BirthAt:    time.Now(),
		MaturityMs: 30 * 24 * 60 * 60 * 1000, // 30-day maturity (riding/breeding age)
		Traits:     traits,
		CreatedAt:  time.Now(),
		// §26.4.2 lineage stat floor: the mean of both parents (integer division, clamped).
		Stats:    meanEntityStats(sire.Stats, dam.Stats),
		PetLevel: 1,
		Grooming: make(map[string]uint64),
		// §27.7.3 lineage inheritance: offspring certified only if BOTH parents certified
		// (both parents already validated certified above); inherits clean lineage.
		Certified:          true,
		BlackMarketAdopted: false,
	}
	l.pets[offspring.PetID] = offspring
	l.logAdminAuditLocked("PET_BRED", owner, fmt.Sprintf("Offspring %s from %s x %s", offspring.PetID, sireID, damID))

	// §27.5 — ThemeEngine OutcomeBias hook (Phase A). BreedPet runs under l.mutex.Lock();
	// use the lock-free core (computeThemeVectorLocked) to avoid RWMutex re-entrant deadlock.
	// Non-structural precompute; consumers call l.OutcomeBias on demand.
	wk := normalizeWallet(owner)
	_ = l.computeThemeVectorLocked(wk, l.leaderboard[wk])

	return offspring, nil
}

// petMatureLocked assumes l.mutex is held by caller.
func (l *Lobby) petMatureLocked(p *PetNFT) bool {
	return time.Since(p.BirthAt).Milliseconds() >= p.MaturityMs
}

// SpawnPet PURCHASES a base pet (no parents) for an owner (§26.4.2). A companion is an
// account upgrade, never a hand-out: the price is server-authoritative, debited from the
// owner's micro-$VBV balance and routed to the faucet sink. The stat floor is derived
// deterministically from the supplied trait bitfield (2 bits per axis, capped).
func (l *Lobby) SpawnPet(owner, name string, traits uint64) (*PetNFT, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if owner == "" || name == "" {
		return nil, errors.New("owner and name required")
	}
	if err := l.chargeBondedAssetFeeLocked(owner, "PET_SPAWN_FEE", PetSpawnFeeMicro); err != nil {
		return nil, err
	}
	p := &PetNFT{
		PetID:      "PET-" + uuid.New().String(),
		Owner:      owner,
		Name:       name,
		SireID:     "",
		DamID:      "",
		BirthAt:    time.Now(),
		MaturityMs: 30 * 24 * 60 * 60 * 1000,
		Traits:     traits,
		CreatedAt:  time.Now(),
		Stats:      petBaseStatsFromTraits(traits),
		PetLevel:   1,
		Grooming:   make(map[string]uint64),
		Certified:  true, // §27.7.3: legitimate spawn path (own-wallet owner) → certified lineage
	}
	l.pets[p.PetID] = p
	l.logAdminAuditLocked("PET_PURCHASED", owner, fmt.Sprintf(
		"Companion %s acquired for %d micro-$VBV (stat floor %d)", p.PetID, PetSpawnFeeMicro, p.Stats.StatSum()))
	return p, nil
}

// chargeBondedAssetFeeLocked debits one bonded-asset purchase fee from the owner's balance and
// routes it to the faucet sink (FaucetShare 1.0) through the SAME router every §25.6.1/§26.4.2
// fee uses, so the Industrial Loop reconciles: no silent mint, no silent burn.
// Caller must hold l.mutex.
func (l *Lobby) chargeBondedAssetFeeLocked(owner, context string, fee uint64) error {
	if owner == "" {
		return errors.New("owner required")
	}
	if fee == 0 {
		return errors.New("fee must be non-zero (bonded assets are purchased, never free)")
	}
	if l.playerBalances[owner] < fee {
		return fmt.Errorf("insufficient balance: %d micro-$VBV required for %s", fee, context)
	}
	l.playerBalances[owner] -= fee
	l.routeBondedFeeToSinkLocked(context, fee)
	return nil
}

// routeBondedFeeToSinkLocked sends a bonded-asset fee that has ALREADY been debited to the faucet
// sink through the same router every other fee uses, so the Industrial Loop reconciles: no silent
// mint, no silent burn.
//
// It is split out of chargeBondedAssetFeeLocked because the player-to-player market debits the
// BUYER and credits the SELLER itself (one transfer, one fee), so it must be able to route just the
// house's cut without taking a second debit. Caller must hold l.mutex.
func (l *Lobby) routeBondedFeeToSinkLocked(context string, fee uint64) {
	if fee == 0 || l.tokenSinkRouter == nil {
		return
	}
	matrix := RevenueSplitMatrix{FaucetShare: 1.0, ClubShare: 0.0, GovernanceShare: 0.0}
	_ = l.tokenSinkRouter.RouteCriminalTax(context, fee, matrix, 0, "")
}

// petBaseStatsFromTraits derives a deterministic stat floor (1..PetSpawnBaseStatMax per axis)
// from the caller's trait bitfield: 2 bits per axis in table order (SPEED, INTELLIGENCE,
// WILLPOWER, STRENGTH, CHARISMA, AGILITY). Pure integer, no RNG, and CAPPED so a self-declared
// trait mask can never mint a maxed companion.
func petBaseStatsFromTraits(traits uint64) EntityStats {
	clamp := func(v uint64) uint64 {
		n := 1 + v
		if n > PetSpawnBaseStatMax {
			n = PetSpawnBaseStatMax
		}
		return n
	}
	return EntityStats{
		Speed:        clamp(traits & 0x3),
		Intelligence: clamp((traits >> 2) & 0x3),
		Willpower:    clamp((traits >> 4) & 0x3),
		Strength:     clamp((traits >> 6) & 0x3),
		Charisma:     clamp((traits >> 8) & 0x3),
		Agility:      clamp((traits >> 10) & 0x3),
	}
}

// meanEntityStats returns the integer mean of two stat vectors, clamped to 1..StatMax.
// Used for §26.4.2 lineage inheritance: an offspring inherits half of each parent's training.
func meanEntityStats(a, b EntityStats) EntityStats {
	mean := func(x, y uint64) uint64 {
		n := (x + y) / 2
		if n < 1 {
			n = 1
		}
		if n > StatMax {
			n = StatMax
		}
		return n
	}
	return EntityStats{
		Speed:        mean(a.Speed, b.Speed),
		Intelligence: mean(a.Intelligence, b.Intelligence),
		Willpower:    mean(a.Willpower, b.Willpower),
		Strength:     mean(a.Strength, b.Strength),
		Charisma:     mean(a.Charisma, b.Charisma),
		Agility:      mean(a.Agility, b.Agility),
	}
}

// GetPets lists the companions owned by a wallet (owner=="" lists every companion, which the
// admin/read-only surfaces use). Bonded-asset ownership rule §23/§27.8: filtered by owner.
func (l *Lobby) GetPets(owner string) []*PetNFT {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	out := make([]*PetNFT, 0)
	for _, p := range l.pets {
		if owner == "" || p.Owner == owner {
			out = append(out, p)
		}
	}
	return out
}

// GroomPet buys one grooming level on ONE stat axis of one companion (§26.4.2). It is the
// deliberate mirror of UpgradeVehicle: same provenance gates, same per-axis ceiling, same
// linear integer price, same sink-routed fee. Returns the pet, the fee charged and the stat
// points actually applied. Nothing is charged when the axis is already capped or at StatMax —
// the ladder is bought, never wasted.
func (l *Lobby) GroomPet(owner, petID, focus string) (*PetNFT, uint64, uint64, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	p, ok := l.pets[petID]
	if !ok {
		return nil, 0, 0, errors.New("pet not found")
	}
	if p.Owner != owner {
		return nil, 0, 0, errors.New("you must own the companion to groom it")
	}
	// §27.7.3 / §26.4.1 provenance: progress requires a legitimate companion.
	if p.BlackMarketAdopted {
		return nil, 0, 0, errors.New("black-market companions are ineligible for grooming (§26.4.1)")
	}
	if !p.Certified {
		return nil, 0, 0, errors.New("companion is not certified for legitimate grooming")
	}
	axis, found := lookupGroomAxis(focus)
	if !found {
		return nil, 0, 0, fmt.Errorf("unknown grooming focus %q", focus)
	}
	ptr := vehicleAxisPtr(&p.Stats, axis)
	if ptr == nil {
		return nil, 0, 0, fmt.Errorf("grooming focus %s has no stat axis", axis)
	}
	if p.Grooming == nil {
		p.Grooming = make(map[string]uint64)
	}
	key := strings.ToUpper(axis)
	if p.Grooming[key] >= PetGroomMaxLevel {
		return nil, 0, 0, fmt.Errorf("%s is already at maximum grooming level (%d)", key, PetGroomMaxLevel)
	}
	if *ptr >= StatMax {
		return nil, 0, 0, fmt.Errorf("%s is already at the stat ceiling (%d)", key, StatMax)
	}

	fee := PetGroomCostMicro(p.Grooming[key])
	if err := l.chargeBondedAssetFeeLocked(owner, "PET_GROOM_FEE", fee); err != nil {
		return nil, 0, 0, err
	}

	p.Grooming[key] = p.Grooming[key] + 1
	before := *ptr
	n := before + PetGroomStatGain
	if n > StatMax {
		n = StatMax
	}
	*ptr = n
	applied := *ptr - before
	p.PetLevel = petGroomLevelLocked(p)
	l.logAdminAuditLocked("PET_GROOMED", owner, fmt.Sprintf(
		"Companion %s %s -> L%d (level L%d, +%d %s, fee %d micro-$VBV)",
		petID, key, p.Grooming[key], p.PetLevel, applied, axis, fee))
	return p, fee, applied, nil
}

// PetGroomCostMicro is the deterministic price of a grooming level: base*(level+1).
// Integer arithmetic only (Architecture Ledger).
func PetGroomCostMicro(currentLevel uint64) uint64 {
	return PetGroomBaseMicro * (currentLevel + 1)
}

// groomAxes is the authoritative grooming table (one entry per EntityStats axis), ordered so
// the served calibration and the UI can never disagree with the server.
var groomAxes = []string{"SPEED", "INTELLIGENCE", "WILLPOWER", "STRENGTH", "CHARISMA", "AGILITY"}

// lookupGroomAxis resolves a grooming focus to its canonical lowercase EntityStats axis name.
func lookupGroomAxis(focus string) (string, bool) {
	f := strings.ToUpper(strings.TrimSpace(focus))
	for _, a := range groomAxes {
		if a == f {
			return strings.ToLower(a), true
		}
	}
	return "", false
}

// PetGroomTableView serves the grooming calibration so no client re-declares a price.
func PetGroomTableView() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(groomAxes))
	for _, a := range groomAxes {
		out = append(out, map[string]interface{}{"focus": a, "axis": strings.ToLower(a)})
	}
	return out
}

// petGroomLevelLocked derives a companion's level from its purchased grooming: 1 + Σ levels.
// Function-based, the mirror of a vehicle's build level. Caller must hold l.mutex.
func petGroomLevelLocked(p *PetNFT) uint64 {
	var total uint64
	for _, a := range groomAxes {
		total += p.Grooming[a]
	}
	return 1 + total
}

// ── §25.6 Vehicles ───────────────────────────────────────────────────────────────
//
// §25.6.1 VEHICLE UPGRADE LADDER — the deliberate counterpart to §26.4 pet breeding.
//
// Pets progress by LINEAGE + TIME (breed two mature certified parents, pay a fee).
// Vehicles progress by PARTS + INVESTMENT (fit a part to one certified vehicle, pay a fee).
// Both are deterministic, integer-only and sink-routed, so progress means the same thing
// for a vehicle as it does for a companion.
//
// Calibration lives here as the single source of truth (§25.10.1 pattern). No float is
// stored, summed or compared anywhere in this file (Architecture Ledger).
// ── §25.6.1 / §26.4.2 BONDED-ASSET ACCOUNT UPGRADES ───────────────────────────
//
// A companion or a vehicle is an ACCOUNT UPGRADE, not a free hand-out (§15.3 ownership,
// §23 bonded asset). Every acquisition and every progression step is therefore a PURCHASE:
// the fee is computed HERE (server-authoritative — never trusted from the request body),
// debited from the owner's uint64 micro-$VBV balance and routed to a deterministic sink so
// the Industrial Loop reconciles (no silent mint, no silent burn). Integer arithmetic only.
const (
	// ── Pet (§26.4.2) ──
	// PetSpawnFeeMicro is the purchase price of a new companion account.
	PetSpawnFeeMicro uint64 = 2_000 * 1_000_000
	// PetBreedFeeMicro is the price of a breeding. It is SERVER-authoritative: the handler
	// deliberately ignores any fee in the request body (a client must never price its own
	// account upgrade).
	PetBreedFeeMicro uint64 = 1_500 * 1_000_000
	// PetGroomBaseMicro is the cost of a groom's FIRST level; level n costs base*(n+1).
	PetGroomBaseMicro uint64 = 100 * 1_000_000
	// PetGroomMaxLevel is the per-axis grooming ceiling (six axes → 60 purchased levels).
	PetGroomMaxLevel uint64 = 10
	// PetGroomStatGain is the stat points a groom contributes per level.
	PetGroomStatGain uint64 = 2
	// PetSpawnBaseStatMax caps the stat floor derived from the client-supplied trait
	// bitfield, so a self-declared trait mask can never mint a maxed companion.
	PetSpawnBaseStatMax uint64 = 3

	// ── Vehicle (§25.6.1) ──
	// VehiclePartMaxLevel is the per-part ceiling (six parts → 60 levels of headroom).
	VehiclePartMaxLevel uint64 = 10
	// VehicleStatGain is the stat points a part contributes per level (mirrors EventStatGain).
	VehicleStatGain uint64 = 2
	// VehicleUpgradeBaseMicro is the cost of a part's FIRST level, in uint64 micro-$VBV.
	// Level n costs base*(n+1), so a full 60-level build is base*55 — a long-term sink
	// calibrated against the item market (2,500 $VBV items, 2,500 $VBV territory).
	VehicleUpgradeBaseMicro uint64 = 150 * 1_000_000
	// VehicleBreakInMs is the break-in window a freshly built vehicle needs before it is
	// race-ready — the time-based mirror of the §26.4 pet maturity gate.
	VehicleBreakInMs int64 = 24 * 60 * 60 * 1000
)

// vehicleKindDef is one purchasable vehicle class: its price, its Level gate (§24.5) and the
// stat floor it is delivered with. The request body cannot choose any of the three — the kind
// selects them, so a client cannot buy a cheap hull and declare it a flyer at level 1.
type vehicleKindDef struct {
	Kind     string
	FeeMicro uint64
	MinLevel uint64
	Base     EntityStats
}

// vehicleKindDefs is the authoritative class table (ordered for deterministic listing).
var vehicleKindDefs = []vehicleKindDef{
	{Kind: "GROUND", FeeMicro: 3_000 * 1_000_000, MinLevel: 1, Base: EntityStats{Speed: 2, Strength: 2, Agility: 2, Intelligence: 1, Willpower: 1, Charisma: 1}},
	{Kind: "FLYER", FeeMicro: 5_000 * 1_000_000, MinLevel: 5, Base: EntityStats{Speed: 3, Agility: 3, Intelligence: 2, Strength: 1, Willpower: 1, Charisma: 1}},
	{Kind: "DIGGER", FeeMicro: 8_000 * 1_000_000, MinLevel: 10, Base: EntityStats{Strength: 3, Willpower: 3, Agility: 2, Speed: 1, Intelligence: 1, Charisma: 1}},
}

// lookupVehicleKind resolves a class name to its definition (case-insensitive).
func lookupVehicleKind(kind string) (vehicleKindDef, bool) {
	k := strings.ToUpper(strings.TrimSpace(kind))
	for _, d := range vehicleKindDefs {
		if d.Kind == k {
			return d, true
		}
	}
	return vehicleKindDef{}, false
}

// VehicleKindTableView serves the purchase calibration so no client re-declares a price.
func VehicleKindTableView() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(vehicleKindDefs))
	for _, d := range vehicleKindDefs {
		out = append(out, map[string]interface{}{
			"kind":          d.Kind,
			"fee_micro":     d.FeeMicro,
			"min_level":     d.MinLevel,
			"base_stat_sum": d.Base.StatSum(),
		})
	}
	return out
}

// ── §25.6.1 / §26.4.2 upgrade ladders ─────────────────────────────────────────

// vehiclePartDef is one tunable part and the EntityStats axis it improves.
type vehiclePartDef struct {
	Part string
	Axis string
}

// vehiclePartDefs is the authoritative part table, ordered for deterministic listing.
var vehiclePartDefs = []vehiclePartDef{
	{Part: "ENGINE", Axis: "speed"},
	{Part: "CHASSIS", Axis: "strength"},
	{Part: "AVIONICS", Axis: "intelligence"},
	{Part: "SUSPENSION", Axis: "agility"},
	{Part: "WILLPLANT", Axis: "willpower"},
	{Part: "TUNING", Axis: "charisma"},
}

// VehiclePartCostMicro is the deterministic cost of a part's NEXT level:
// VehicleUpgradeBaseMicro * (currentLevel + 1). Integer arithmetic only.
func VehiclePartCostMicro(currentLevel uint64) uint64 {
	return VehicleUpgradeBaseMicro * (currentLevel + 1)
}

// lookupVehiclePart resolves a part name to its definition (case-insensitive).
func lookupVehiclePart(part string) (vehiclePartDef, bool) {
	p := strings.ToUpper(strings.TrimSpace(part))
	for _, d := range vehiclePartDefs {
		if d.Part == p {
			return d, true
		}
	}
	return vehiclePartDef{}, false
}

// SpawnVehicle PURCHASES a player-owned vehicle (§25.6.1). The class table — not the request
// body — decides the price, the Level gate and the delivered stat floor, so a client cannot
// buy a cheap hull and declare it a flyer at level 1. Fee is debited and sink-routed.
func (l *Lobby) SpawnVehicle(owner, name, kind string, minLevel uint64) (*VehicleNFT, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if owner == "" || name == "" {
		return nil, errors.New("owner and name required")
	}
	def, ok := lookupVehicleKind(kind)
	if !ok {
		return nil, fmt.Errorf("unknown vehicle class %q (GROUND, FLYER, DIGGER)", kind)
	}
	// The class owns MinLevel; a caller-supplied value is ignored (and a caller asking for a
	// gate it cannot meet is simply not trusted — the class gate is the gate).
	_ = minLevel
	if err := l.chargeBondedAssetFeeLocked(owner, "VEHICLE_SPAWN_FEE", def.FeeMicro); err != nil {
		return nil, err
	}
	v := &VehicleNFT{
		VehicleID:    "VEH-" + uuid.New().String(),
		Owner:        owner,
		Name:         name,
		Kind:         def.Kind,
		MinLevel:     def.MinLevel,
		CreatedAt:    time.Now(),
		Stats:        def.Base,
		VehicleLevel: 1,
		Upgrades:     make(map[string]uint64),
		BreakInMs:    VehicleBreakInMs,
		Certified:    true, // legitimate own-wallet build path
	}
	l.vehicles[v.VehicleID] = v
	l.logAdminAuditLocked("VEHICLE_PURCHASED", owner, fmt.Sprintf(
		"Vehicle %s (%s) acquired for %d micro-$VBV (min level %d, stat floor %d)",
		v.VehicleID, def.Kind, def.FeeMicro, def.MinLevel, def.Base.StatSum()))
	return v, nil
}

func (l *Lobby) GetVehicles(owner string) []*VehicleNFT {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	out := make([]*VehicleNFT, 0)
	for _, v := range l.vehicles {
		if owner == "" || v.Owner == owner {
			out = append(out, v)
		}
	}
	return out
}

// vehicleBuildLevelLocked derives the build level from the part table: 1 + Σ part levels.
// Function-based, the mirror of a pet's function-based level. Caller must hold l.mutex.
func vehicleBuildLevelLocked(v *VehicleNFT) uint64 {
	var total uint64
	for _, d := range vehiclePartDefs {
		total += v.Upgrades[d.Part]
	}
	return 1 + total
}

// vehicleBreakInCompleteLocked reports whether the break-in window has elapsed.
// Caller must hold l.mutex.
func vehicleBreakInCompleteLocked(v *VehicleNFT) bool {
	if v.BreakInMs <= 0 {
		return true
	}
	return time.Since(v.CreatedAt).Milliseconds() >= v.BreakInMs
}

// vehicleAxisPtr returns a pointer to the single stat axis a part improves.
func vehicleAxisPtr(s *EntityStats, axis string) *uint64 {
	switch axis {
	case "speed":
		return &s.Speed
	case "strength":
		return &s.Strength
	case "intelligence":
		return &s.Intelligence
	case "agility":
		return &s.Agility
	case "willpower":
		return &s.Willpower
	case "charisma":
		return &s.Charisma
	}
	return nil
}

// VehiclePartTableView returns the part table for read-only clients. It exists so the UI
// never re-declares the calibration (single source of truth); the server still validates
// every part name it receives. Deterministic order.
func VehiclePartTableView() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(vehiclePartDefs))
	for _, d := range vehiclePartDefs {
		out = append(out, map[string]interface{}{"part": d.Part, "axis": d.Axis})
	}
	return out
}

// vehicleRarity maps a build class to the §31 listing-rarity vocabulary. Deterministic:
// the class is the vehicle's "bloodline", so it drives rarity exactly as pet traits do.
func vehicleRarity(kind string) string {
	switch strings.ToUpper(strings.TrimSpace(kind)) {
	case "FLYER":
		return "rare"
	case "DIGGER":
		return "epic"
	}
	return "common"
}

// UpgradeVehicle fits one part to one vehicle: a deterministic, sink-routed step up the
// §25.6.1 ladder. Mirrors BreedPet's contract — same provenance rule, same fee-to-sink
// path, same integer-only discipline.
//
// Returns the vehicle, the micro fee charged, and the stat points actually added.
func (l *Lobby) UpgradeVehicle(owner, vehicleID, part string) (*VehicleNFT, uint64, uint64, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	v, ok := l.vehicles[vehicleID]
	if !ok {
		return nil, 0, 0, errors.New("vehicle not found")
	}
	if v.Owner != owner {
		return nil, 0, 0, errors.New("you must own the vehicle to upgrade it")
	}
	// §27.7.3 / §25.6.1: progress requires a legitimate build. An off-ledger vehicle is
	// ineligible, exactly as a black-market pet is ineligible for legitimate breeding.
	if v.BlackMarketAdopted {
		return nil, 0, 0, errors.New("black-market vehicles are ineligible for upgrading (§27.7.3)")
	}
	if !v.Certified {
		return nil, 0, 0, errors.New("vehicle is not certified for legitimate upgrading")
	}

	def, found := lookupVehiclePart(part)
	if !found {
		return nil, 0, 0, fmt.Errorf("unknown part %q", part)
	}
	// Resolve the axis BEFORE the fee is taken, so a malformed table can never cost a player.
	ptr := vehicleAxisPtr(&v.Stats, def.Axis)
	if ptr == nil {
		return nil, 0, 0, fmt.Errorf("part %s has no stat axis", def.Part)
	}
	if v.Upgrades == nil {
		v.Upgrades = make(map[string]uint64)
	}
	level := v.Upgrades[def.Part]
	if level >= VehiclePartMaxLevel {
		return nil, 0, 0, fmt.Errorf("%s is already at maximum level (%d)", def.Part, VehiclePartMaxLevel)
	}

	fee := VehiclePartCostMicro(level)
	// Industrial Loop: the fee leaves the player and is routed to a deterministic sink —
	// the SAME helper every §25.6.1/§26.4.2 purchase uses (FaucetShare 1.0), no silent mint/burn.
	if err := l.chargeBondedAssetFeeLocked(owner, "VEHICLE_UPGRADE_FEE", fee); err != nil {
		return nil, 0, 0, err
	}

	v.Upgrades[def.Part] = level + 1

	before := *ptr
	n := before + VehicleStatGain
	if n > StatMax {
		n = StatMax
	}
	if n < 1 {
		n = 1
	}
	*ptr = n
	applied := *ptr - before

	v.VehicleLevel = vehicleBuildLevelLocked(v)
	l.logAdminAuditLocked("VEHICLE_UPGRADED", owner, fmt.Sprintf(
		"Vehicle %s %s -> L%d (build L%d, +%d %s, fee %d micro-$VBV)",
		vehicleID, def.Part, level+1, v.VehicleLevel, applied, def.Axis, fee))
	return v, fee, applied, nil
}

// DeployVehicle parks a vehicle in a region so it appears in the 3D world and feeds the
// §30 region power overlay. Ownership is enforced (bonded-asset rule §23/§27.8).
func (l *Lobby) DeployVehicle(owner, vehicleID, region string) (*VehicleNFT, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if region == "" {
		return nil, errors.New("region required")
	}
	v, ok := l.vehicles[vehicleID]
	if !ok {
		return nil, errors.New("vehicle not found")
	}
	if v.Owner != owner {
		return nil, errors.New("you must own the vehicle to deploy it")
	}
	v.Region = region
	l.logAdminAuditLocked("VEHICLE_DEPLOYED", owner, fmt.Sprintf("Vehicle %s -> %s", vehicleID, region))
	return v, nil
}

// ── §25.7 World Content ──────────────────────────────────────────────────────────

// CreateWorldContent authors a native world entity (NPC/animal/scenery/weather).
func (l *Lobby) CreateWorldContent(creator, kind string) (*WorldContentNFT, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if creator == "" || kind == "" {
		return nil, errors.New("creator and kind required")
	}
	c := &WorldContentNFT{
		ContentID: "WC-" + uuid.New().String(),
		Creator:   creator,
		Kind:      kind,
		Region:    "",
		Deployed:  false,
		CreatedAt: time.Now(),
	}
	l.worldContent[c.ContentID] = c
	return c, nil
}

// DeployWorldContent pushes a built entity into a region (§25.7 purchase/deploy rule).
func (l *Lobby) DeployWorldContent(contentID, region string) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	c, ok := l.worldContent[contentID]
	if !ok {
		return errors.New("content not found")
	}
	c.Region = region
	c.Deployed = true
	l.logAdminAuditLocked("WORLD_CONTENT_DEPLOYED", c.Creator, fmt.Sprintf("Content %s -> %s", contentID, region))
	return nil
}

func (l *Lobby) GetWorldContent(region string) []*WorldContentNFT {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	out := make([]*WorldContentNFT, 0)
	for _, c := range l.worldContent {
		if region == "" || (c.Region == region && c.Deployed) {
			out = append(out, c)
		}
	}
	return out
}

// ── §26.4.3 / §25.6.2 BONDED-ASSET DECK-CARD BONUS ────────────────────────────
//
// Bonded assets fight for their owner. Rather than inventing a parallel combat system, the
// pet + vehicle stat vector lends the owner's OWN deck cards a capped integer percentage power
// boost — the same shape as the coalition (+10%) / regional (+5%) boosts, so it composes with
// the existing power math. The percentage is snapshotted into MatchState at match start
// (P1BondedBoostPct / P2BondedBoostPct) so Server and WASM apply an IDENTICAL constant.
// Integer only: no float ever reaches the ledger (Architecture Ledger).
const (
	// BondedDeckBoostDivisor converts household stat points into percent: every N points of
	// (pets + vehicles) stat sum grants +1% card power.
	BondedDeckBoostDivisor uint64 = 60
	// BondedDeckBoostMaxPct caps the granted percentage (mirrors the coalition ceiling).
	BondedDeckBoostMaxPct uint64 = 10
)

// bondedDeckBoostPctFromStats converts a household stat vector into a capped integer percent.
func bondedDeckBoostPctFromStats(stats EntityStats) int {
	pct := stats.StatSum() / BondedDeckBoostDivisor
	if pct > BondedDeckBoostMaxPct {
		pct = BondedDeckBoostMaxPct
	}
	return int(pct)
}

// bondedDeckBoostPctLocked computes the owner's bonded bonus from their CERTIFIED, non
// black-market pets + vehicles (§27.7.3 legitimacy rule: only legitimate assets fight for you).
// Caller must hold l.mutex.
func (l *Lobby) bondedDeckBoostPctLocked(wallet string) int {
	if wallet == "" {
		return 0
	}
	var total EntityStats
	for _, p := range l.pets {
		if p.Owner == wallet && p.Certified && !p.BlackMarketAdopted {
			total = addEntityStats(total, p.Stats)
		}
	}
	for _, v := range l.vehicles {
		if v.Owner == wallet && v.Certified && !v.BlackMarketAdopted {
			total = addEntityStats(total, v.Stats)
		}
	}
	return bondedDeckBoostPctFromStats(total)
}

// BondedDeckBoostPct is the read-only accessor (takes the lock) for API/UI surfaces, so a
// client can display the real bonus its assets lend instead of guessing one.
func (l *Lobby) BondedDeckBoostPct(wallet string) int {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.bondedDeckBoostPctLocked(wallet)
}

// addEntityStats accumulates stat vectors for the household roll-up. The accumulator is
// clamped well above StatMax so a large kennel is not truncated before the percent cap binds.
func addEntityStats(a, b EntityStats) EntityStats {
	add := func(x, y uint64) uint64 {
		n := x + y
		if n > StatMax*64 {
			n = StatMax * 64
		}
		return n
	}
	return EntityStats{
		Speed:        add(a.Speed, b.Speed),
		Intelligence: add(a.Intelligence, b.Intelligence),
		Willpower:    add(a.Willpower, b.Willpower),
		Strength:     add(a.Strength, b.Strength),
		Charisma:     add(a.Charisma, b.Charisma),
		Agility:      add(a.Agility, b.Agility),
	}
}

// InitAssetLife initializes the three maps (called from newLobby / Init paths).
func (l *Lobby) InitAssetLife() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.pets == nil {
		l.pets = make(map[string]*PetNFT)
	}
	if l.vehicles == nil {
		l.vehicles = make(map[string]*VehicleNFT)
	}
	if l.worldContent == nil {
		l.worldContent = make(map[string]*WorldContentNFT)
	}
}
