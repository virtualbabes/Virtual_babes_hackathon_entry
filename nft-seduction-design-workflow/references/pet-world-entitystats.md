# §30 Pet World — EntityStats + 3D Power Overlay + Event Engine (recipe)

BUILT 2026-08-30 (yolo=true). Both `go build` targets GREEN; `go vet .` OK; `npm run build` exit 0.

## What was added
- `backend_types.go`: `EntityStats` (uint64 1..100, Speed/Intelligence/Willpower/Strength/Charisma/Agility)
  + `StatSum()`/`DominantStat()`; added to `PetNFT` (Stats, PetLevel, Mature, OwnerOpinion, OwnerCrossOpinion)
  and `AICitizen` (Stats, BotLevel, OwnerOpinion, OwnerCrossOpinion). `RegionView.EntityPowerOverlay *EntityPowerOverlay`.
- `entity_event_engine.go` (NEW, `//go:build !js && !wasm`): `EntityEventEngine`, `EntityEvent`, `EntityEventResult`,
  `OwnerRelationGraph` (Opinion + CrossOpinion), `ComputeEffectivePowerLevel`, `BaseUserEntityStats(PlayerStats)`,
  `CombineStats`, `ApplyEventResult` (train/reward/punish; immature + black-market earn 0), `BuildRegionPowerOverlay`,
  `SortEntitiesByPower`, `handleEntityRegions`.
- `server.go`: `l.entityEvents = NewEntityEventEngine()`; route `/api/entity-events/regions`.
- `seasonal_event_engine.go`: `GetRegionViews` populates `EntityPowerOverlay` (pets via OWNER region; bots via Region).
- `world3d.js`: tower height + emissive glow scale by `avg_entity_power`; label `PWR <n> [dominant]`.

## Design rules (user-locked)
- 3D power overlay IS the cap: `ComputeEffectivePowerLevel = baseLevel + floor(statSum/50)`, clamp 600.
- USERS also capped: `BaseUserEntityStats(profile)` floor (Reputation→Charisma, Mojo→Strength, Cunning→Intel/Will,
  Nurturing→Agil; WantedLevel degrades lawful stats) + `CombineStats(base, overlay)` for event training.
  User refinement mid-build: *"stats need a profile-derived base for users before applying the overlay"* → never
  apply overlay without the profile base for users.
- Events: WIN/TRY train+reward; QUIT punishes (stat debuff, no reward). uint64 micro only.
- Bonuses/Degradations from rivalries (`rival_career_engine`) + regional dynamics (`ComputeWorldDynamicsSignature`)
  + owner relationship (OwnerOpinion/OwnerCrossOpinion).
- Tournaments = mature-only. Hosting gated to host profile tier (enter bots into other dynamics, cannot CREATE outside own tier).

## Data-model pitfalls (hit + fixed this session)
- **`clampInt` already exists with signature `clampInt(v, lo, hi int) int`** in `rivalry_engine.go`. Do NOT
  redeclare a 1-arg `clampInt` in a new file — use `clampInt(x, 1, int(StatMax))`. (A duplicate `clampInt(v int)`
  caused a `DuplicateDecl` LSP error; removed and reused the 3-arg form.)
- **`PetNFT` has NO `Region` field.** Pets resolve to a region via their OWNER's citizen region
  (`l.aiEngine.GetCitizen(owner).Region`, else "Base"). Do not iterate `l.pets` by `p.Region` — it doesn't exist.
- **Pets live in `l.pets` map** (keyed by pet ID, `Owner` field), NOT a `petRegistry`/`GetAllPets` registry.
  Iterate `for _, p := range l.pets` under `l.mutex.RLock()` when building region overlays.
- **`PlayerStats` has no `Level` field.** User level is derived (career Tier / Reputation). Don't reference
  `p.Level`; if you need an effective level, compute from `CareerXP`/`Reputation`.
- **Legacy float debt (do NOT replicate):** `PlayerStats.Strength float64` and `TournamentService.DispatchTournamentRewards`
  multiplies `potShareMicro * 1.1` (float) then casts to uint64. Keep new §30 math pure uint64/int.
- **New `.go` file MUST carry `//go:build !js && !wasm`** (entity_event_engine.go does). Without it, the wasm
  default build excludes `backend_types.go` and throws `undefined: Lobby`/`undefined: PetNFT` for real types.
- **Adding a server-struct field auto-serializes** into the existing `/api/regions` JSON — no new route needed to
  surface `EntityPowerOverlay` to the WASM client. `world3d.js` just reads `rv.EntityPowerOverlay`.

## Deferred (future Pet World plan, not this pass)
- Event reward accrual payout loop (the `/api/events/world-shaking` hook contract from §24.5/§24.6).
- Bot/LLM tournament service reusing `TournamentService.DispatchTournamentRewards`.
- Pet-specific token vs shared $VBV decision (constitutional: uint64 micro either way).
- §31 Synergy & Orphan Dynamics (see `references/synergy-orphan-dynamics.md`) implements on top of this.
