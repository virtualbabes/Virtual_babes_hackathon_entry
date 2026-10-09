# VEHICLE UPGRADE PLAN — making §25.6 vehicles rival §26.4 pet breeding

Version: 1.0
Status: **Implemented** (2026-09-14) — backend + routes + Life Assets UI + Portfolio analytics
Owner: `asset_life_engine.go` (§25.6), `entity_event_engine.go` (§30), `entity_market.go` (§31)
Related: `AI-Brain/MASTER-PLAN.md` §25.6 / §26.4 / §30, `.clinerules/app-entry-mandate.md` §3

---

## 1. Why this document exists

Brendan: *"investigate how we can ensure the vehicles have an upgrade section to rival the pet
breeding."*

This is the investigation result: the evidence, the gap, and the deterministic design that closes it.

---

## 2. Evidence — the current state of vehicles (measured, not assumed)

| Area | Finding | Evidence |
| --- | --- | --- |
| Type | `VehicleNFT{VehicleID, Owner, Name, Kind, MinLevel, CreatedAt}` — **no stats, no level, no region, no provenance** | `backend_types.go:334-341` |
| Creation | `SpawnVehicle(...)` — creation only; `min_level` taken straight from the request body, **unvalidated**, **no fee** | `asset_life_engine.go:125-141`, `server_main.go:1163-1184` |
| Routes | only `POST /api/vehicles/spawn` + `GET /api/vehicles` — no upgrade, no deploy, no arena | `server_main.go:1163`, `:1185` |
| Market | a vehicle listing sets `Level = MinLevel`, `Rarity = "common"`, and **never sets `Stats`**, so `PowerScore` stays 0 | `entity_market.go:211-218` (compare pets `:198`, bots `:209`) |
| §30 power overlay | `BuildRegionPowerOverlay(region, pets, bots)` enumerates **pets + bots only** — vehicles contribute nothing | `entity_event_engine.go:279-305`, caller `seasonal_event_engine.go:870` |
| UI | `renderVeh()` prints `kind · L<min_level>+` and reads `v.region` — a field **no struct supplies**, so a vehicle can never show a region or a "View in 3D" button | `Public/js/life_assets.js:158-170` |
| Progression | none of any kind | repo-wide grep: no `UpgradeVehicle`, no vehicle level, no vehicle stat |

For contrast, the pet path (§26.4) already has: lineage (`SireID`/`DamID`), a deterministic trait
bitfield, `Certified` / `BlackMarketAdopted` provenance, a maturity gate, an `EntityStats` vector,
`PetLevel`, `OwnerOpinion`, a fee routed to a deterministic sink, a battle arena, and participation
in the power overlay.

**Conclusion:** vehicles are a *dead-end* asset. Pets have an entire progression economy; vehicles
have a name and a level requirement. That asymmetry is the gap.

---

## 3. The design — parts & tuning (breeding's complement)

Breeding answers *"how do I improve my companion?"* with **lineage + time**.
Vehicles answer it with **parts + investment**. Same engine, different verb:

| §26.4 Pets | §25.6 Vehicles (this design) |
| --- | --- |
| `SpawnPet` base | `SpawnVehicle` base (**+ fee, + validated class**) |
| bloodline `SireID`/`DamID` | build class `Kind` (GROUND/FLYER/DIGGER) + part configuration |
| deterministic trait bitfield | deterministic `EntityStats` vector advanced by parts |
| maturity gate (30 d) | break-in gate (`BreakInMs`, integer/time-based, same shape) |
| `BreedPet` (two parents + fee → certified offspring) | `UpgradeVehicle` (one vehicle + part + fee → higher build level) |
| `Certified` / `BlackMarketAdopted` (§27.7.3) | **identical** provenance rule |
| `PetLevel` → power overlay → arena | `VehicleLevel` → power overlay → (future) races |

### 3.1 Parts (one per `EntityStats` axis)

| Part | Axis | Notes |
| --- | --- | --- |
| `ENGINE` | Speed | powertrain |
| `CHASSIS` | Strength | frame |
| `AVIONICS` | Intelligence | flight / sensor suite (FLYER-favoured) |
| `SUSPENSION` | Agility | handling |
| `WILLPLANT` | Willpower | drive-train control unit |
| `TUNING` | Charisma | paintwork / presence |

### 3.2 Deterministic calibration (integer only — Architecture Ledger)

```
VehiclePartMaxLevel     = 10        // per-part ceiling
VehicleStatGain         = 2         // stat points per part level (mirrors EventStatGain)
VehicleUpgradeBaseMicro = 250_000   // 0.25 $VBV at part level 0
cost(part)              = VehicleUpgradeBaseMicro * (partLevel + 1)   // linear, integer
VehicleLevel            = 1 + Σ partLevels                            // function-based build level
```

No float is stored, summed or compared. `StatMax = 100` and `ComputeEffectivePowerLevel`
(§30: `base + floor(statSum/PowerOverlayScale)`, clamp 600) are reused unchanged.

### 3.3 Gates (each one verifiable)

1. **Ownership** — `v.Owner` must equal the caller (§23/§27.8 guard pattern).
2. **Legitimacy** — `Certified && !BlackMarketAdopted` (§27.7.3: legitimacy = right to progress).
3. **Part validity** — the part must exist in the calibrated table.
4. **Caps** — `partLevel < VehiclePartMaxLevel`; every stat stays clamped `1..StatMax`.
5. **Economic** — the fee is debited from `playerBalances` (uint64 micro) and routed through
   `tokenSinkRouter.RouteCriminalTax("VEHICLE_UPGRADE_FEE", fee, {FaucetShare:1.0}, 0, "")` —
   the *exact* sink path `BreedPet` already uses, so the Industrial Loop reconciles
   (no silent mint, no silent burn).

### 3.4 Known pre-existing weakness (recorded here, deliberately not silently changed)

`SpawnVehicle` accepted any client-supplied `min_level` and charged nothing. The upgrade path now
charges for progress, but **spawn itself is still free and unvalidated** — hardening it needs a
spawn-fee decision (economic impact) that belongs to Brendan. Recorded in `Problems.md`.



---

## 4. Implementation map

| # | Change | File |
| --- | --- | --- |
| 1 | `VehicleNFT` gains `Stats`, `VehicleLevel`, `Upgrades`, `Region`, `BreakInMs`, `BuiltAt`, `Certified`, `BlackMarketAdopted` | `backend_types.go` |
| 2 | Calibration constants + `vehiclePartDefs` | `asset_life_engine.go` |
| 3 | `UpgradeVehicle(owner, vehicleID, part)` — deterministic, sink-routed | `asset_life_engine.go` |
| 4 | `DeployVehicle(vehicleID, region)` — gives `v.region` a real owner | `asset_life_engine.go` |
| 5 | `SpawnVehicle` sets provenance + break-in window + base stats | `asset_life_engine.go` |
| 6 | `BuildRegionPowerOverlay(region, pets, bots, vehicles)` — vehicles join §30 | `entity_event_engine.go` + caller `seasonal_event_engine.go` |
| 7 | Vehicle market listing carries `Stats` / `Level` / `PowerScore` / `Region` / `Rarity` | `entity_market.go` |
| 8 | `POST /api/vehicles/upgrade` (economy-tight) + `POST /api/vehicles/deploy` (wallet-default) | `server_main.go` |
| 9 | Life Assets ▸ Vehicles gains the parts panel (upgrade + deploy) | `Public/js/life_assets.js` |
| 10 | Portfolio ▸ Companions ▸ Vehicles gains read-only upgrade analytics | `Public/js/portfolio.js` |

**Ownership rule (app-entry-mandate §2/§3):** the *actions* (upgrade, deploy) live on the World
Dashboard path (Assets ▸ Life Assets). The Portfolio only **reads** the ladder.

---

## 5. Verification performed (measured, not assumed)

* `go build ./...` → rc 0 for **native**, **`GOOS=linux GOARCH=amd64`** and **`GOOS=js GOARCH=wasm`**
  (these files are `!js && !wasm`, so WASM is unaffected but was still re-run).
* `go test -vet=off -run TestVehicleUpgradeLadder -v .` → **4/4 PASS**
  (`SpawnProvenance`, `Gates`, `DeterministicProgress`, `Deploy`). `-vet=off` is required only because of
  three PRE-EXISTING vet failures in unrelated files (recorded in `Problems.md` §11).
* `node --check` on `portfolio.js`, `life_assets.js`, `underworld.js`, `world_dashboard.js` → rc 0.
* **Live HTTP round-trip** (`npm run dev` server on :8090):
  * `POST /api/vehicles/spawn?wallet=…` → `certified:true`, `vehicle_level:1`, stat floor 1 per axis,
    `break_in_ms:86400000`.
  * `GET /api/vehicles?wallet=…` → `parts[6]`, `part_max_level:10`, `base_cost_micro:250000`,
    `stat_gain:2`.
  * `POST /api/vehicles/upgrade` unfunded → `400 insufficient balance for the part`;
    bad part → `400 unknown part "TURBO"`; missing vehicle → `400 vehicle not found`.
  * `POST /api/vehicles/deploy` → region persisted, `GET` then reports `region:"Base"`.
* **Regression suite all green after the change:** `npm run verify:portfolio` → **60/60 screens** render,
  association-export contract OK (200, null-free, float-free), 0 page errors; `verify_entry_probe.js` →
  **38/38**; `verify_overlay_visibility.js` → **22/22**; `ui_test_harness.js` → **12/12**.

> The success path (a funded upgrade) is proven by the Go contract test rather than HTTP, because the dev
> sandbox has no funded wallet to debit. The HTTP surface proves the gates.

---

## 6. Deliberately deferred

* Vehicle **races / arena** (the pet-battle equivalent) — needs its own service; the stat vector and
  build level installed here are its prerequisites.
* **Spawn fee + `kind`/`min_level` validation** (pre-existing weakness, §3.4).
* Vehicle **black-market acquisition path** — the provenance field exists and is honoured by the
  upgrade gate, but nothing sets `BlackMarketAdopted` yet.

---

# §25.6.1 / §26.4.2 / §26.4.3 — BONDED-ASSET ACCOUNT UPGRADES (2026-09-14 g)

**Design rule (Brendan):** *bonded assets are NOT free — they are account upgrades and are purchased.*
The ladders are the deliberate pair:

```
COMPANION                                     VEHICLE
purchase   PetSpawnFeeMicro  (2,000 $VBV)      purchase  class table (GROUND 3,000 / FLYER 5,000 / DIGGER 8,000 $VBV)
breed      PetBreedFeeMicro  (1,500 $VBV)      parts     VehicleUpgradeBaseMicro 150 $VBV × (level+1)
groom      PetGroomBaseMicro 100 $VBV × (l+1)  deploy    region binding (free, ownership-gated)
           → +2 stat / level, per-axis cap 10            → +2 stat / level, per-part cap 10
           → level = 1 + Σ grooming levels               → build level = 1 + Σ part levels
LINEAGE + TIME + INVESTMENT                    PARTS + INVESTMENT
```

* **Delivered stat floor**: vehicles from their class (9–11 stat points); companions from the trait
  bitfield (deterministically 1..3 per axis, capped so a self-declared mask cannot mint a maxed pet).
* **Provenance (§27.7.3)**: only `Certified` + non-`BlackMarketAdopted` assets can breed/groom/upgrade
  or lend their owner deck power. Off-ledger assets are inert (`bondedDeckBoostPctLocked`).
* **Default gates**: `SpawnVehicle` no longer accepts a caller-supplied `min_level` (the CLASS owns the
  gate); `POST /api/pets/breed` no longer accepts a caller-supplied fee.
* **Owner bonus (§26.4.3 / §25.6.2)**: `BondedDeckBoostDivisor = 60`, `BondedDeckBoostMaxPct = 10` —
  every 60 household stat points = +1% card power, capped at +10%. Snapshotted into
  `MatchState.P1BondedBoostPct` / `P2BondedBoostPct` in `initiatePairedMatch` and pushed in both
  `challenge` payloads → `SyncMatchMetadata` → `Game.P1/P2BondedBoostPct` → applied by BOTH
  `getEffectiveServerPower` (battle_service.go) and `getEffectivePower` (main.go) at the same point in
  the same base (immediately after the coalition/regional boost), so preview and authority agree.
* **Arenas are placeholders** pointing at the 3D world (transport + gang-up events). See
  `.clinerules/app-entry-mandate.md` §8.
* **Served calibration** (no client re-declares a price): `GET /api/pets` →
  `spawn_fee_micro`, `breed_fee_micro`, `groom_base_micro`, `groom_max_level`, `groom_stat_gain`,
  `groom_axes`; `GET /api/vehicles` → `spawn_fees[]`, `parts[]`, `part_max_level`, `base_cost_micro`,
  `stat_gain`, `break_in_ms`; `GET /api/owner/combined-stats` → `bonded_deck_boost_pct`.
* **Verification**: `bonded_asset_purchase_test.go` (6 tests) + the updated
  `vehicle_upgrade_test.go` (4 tests) + probe 44/44 (6 new `bonded.*` assertions) + overlay 22/22 +
  harness 12/12 + `verify:portfolio` 60/60. `go test .` now runs WITHOUT `-vet=off`.

