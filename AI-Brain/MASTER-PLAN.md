# MASTER CONSOLIDATED PLAN — NFT-Seduction (Virtualbabes Arena)

> **Generated:** 2026-08-30 by nft-seduction-code-agent (Lead Systems Architect, KEY 4 WAIT).
> **Last reconciled against live code:** 2026-09-05 (post AI civilization backend complete; both build targets GREEN).
> **🔒 PLAN LOCKED 2026-08-30 (yolo):** All design decisions through §32.3 + §30 payout loop + §32.1 console-native
> build are ratified and implemented. This document is the authoritative snapshot. No further design changes
> without an explicit user directive. Open engineering items (e.g. pet-token
> vs $VBV decision) are tracked as DEFERRED tasks below, not open design questions.
> **Purpose:** Single reconciled roll-up of the extensive plans. Every status below was verified by
> grepping the live `.go` tree, NOT assumed from prior docs. Where a plan claims something, the code truth
> is stated explicitly. Source docs: AI-Citizen-Design.md (§3–§27, §16 Q-table), ToDo.md (Pillars/Phases),
> Game_expansion_plan.md (7 Pillars), RAG-design-gaps.md (now PARTLY STALE — see §9), NFT-Seduction-Absolute-Visoin.md, §23.5 plans.
> **Build reality:** `GOOS=linux GOARCH=amd64` (server) AND `GOOS=js GOARCH=wasm` (client) both GREEN.

---

## 0. STATUS (verified against code 2026-09-05)

| Layer | State | Evidence |
|---|---|---|
| 8 Pillars (ToDo.md) | ✅ COMPLETE | ToDo.md §Pillar 1–8 |
| Phase 1–4 (Infra/Security/Career/Justice) | ✅ COMPLETE | ToDo.md |
| Phase 7 Civilization Expansion (P7-A→P7-E) | ✅ COMPLETE & verified | ToDo.md post-Phase-7 |
| KEY 3.5 Server-build reconciliation | ✅ COMPLETE (was RED→GREEN; 529→0 errors) | `go build` both targets |
| AI-Citizen lineage §3–§26 | ✅ LOCKED + code present | `ai_citizen_engine.go` |
| §23.5 asset registry | ✅ IMPLEMENTED (types + scaffold) | `bonded_asset_registry.go` |
| §27.7 data structs on `AICitizen` | ✅ IMPLEMENTED (BondGraph/Certified/BirthCertID/BlackMarketAdopted + spawn-cert) | `ai_citizen_engine.go:49-61,286-287` |
| §27 Theme-Binding Phase A (ThemeEngine) | ✅ IMPLEMENTED + wired | `theme_engine.go` (4 methods + 3 routes) |
| §27.6 lock **enforcement** | ✅ WIRED (verified 2026-08-30) | `bonded_asset_registry.go` — `MoodTagForWallet` excludes locked assets; `guardOwnerHolder`/`TransferOwnership`/`ModifyBondedAsset`/`Burn` all require caller==owner==holder (§27.8) |
| §27.7 signature terms (Faith/Domestic/EntityLegitimacy) | ✅ ALL THREE WIRED (2026-09-05) | `theme_engine.go` computeRegionFaithCoherence / computeRegionDomesticCoherence / computeRegionEntityLegitimacy — all three now read real data from BondGraph, Religions, Clubs, and Certified fields |
| **§25 web-3D client** | ✅ Phase 1 IMPLEMENTED (2026-08-30) | `Public/js/world3d.js` + `mechanics.js` + `three.module.js` vendored |
| **Full AI Civilization Backend** | ✅ IMPLEMENTED (2026-09-05) | `ai_citizen_engine.go` — Faith rituals, marriage, breeding, pet adoption, justice enforcement, employment, rivalry, market activity, career progression, event hosting wired into BehavioralTick |
| Frontend (90+ JS modules) | ⚠️ NEEDS REFACTOR | Board hidden in lobby, 20+ buttons crammed in rail, many duplicate modules |
| Dev Server | ✅ FIXED 2026-09-04 | Public blockchain indexers, no local node required, port 8090 isolated |
| Service Migration | ⚠️ IN PROGRESS | Service structs embedded in Lobby, duplicate handlers remain in lobby_manager.go |

**Current KEY:** KEY 4 WAIT — 🔒 PLAN LOCKED. Awaiting next directive — UI development for AI civilization system ready.

---

## 1. VISION (from Absolute Vision + Game Expansion Plan)

Intentional split-WASM architecture: front-end and back-end wasms **mimic on-chain behaviour**, with the
**server authoritative** and the front-end as UI interaction. The app is a **game hub for developers** —
fine-grain mechanics must be repeatable, stackable, rival-capable, and easily expandable via developer mode.

Pillars (1–8): Industrial&Trust, High-Finance&Market, Criminality&Intel, Performative&Social,
Deep RPG, Mutation Foundry, Underworld Recovery&Redemption, Rivalry.
Guardrails: anti-inflation ($VBV loop Faucet→Players→Clubs→Shops→Faucet); Sybil protection; governance 5–50%;
NO cloud models (local only, PILLAR 2 determinism).

---

## 2. AI CITIZENS (§3 / §15 — LOCKED + code present)

- `AICitizen` (`ai_citizen_engine.go:24`); `SpawnAI(lobby, ownerWallet, region, customName)` (`:185`);
  region cap `1 + regionIndex` (`:167`); `AttachOwnership`/`DetachOwnership` (`:363`/`:375`);
  `GetFreeAgents` (`:300`); `FreeAgentSeekContract` (`:349`); behavioral loop; `SaveCitizens`/`LoadCitizens`.
- **HARD MANDATE (enforced):** AI citizen MUST hold OWN wallet; forbidden from personal wallet. `SpawnAI`
  sets `Certified=true` + `BirthCertID` at `:286-287` (own-wallet + ledger + fee path). `CheckAssetOptIn` etc.
  route through the mandate. The own-wallet check is IN PLACE at spawn.
- **Bot children** = derived `AICitizen` records (`OriginWallet = parent`, reduced `Tier`); no separate type.
- **AI-character pets** = companion bots (`BondGraph.AICharPetIDs`), distinct from §26.4 biological PetNFT.
- **DOMESTIC/PROVENANCE STRUCTS IMPLEMENTED** (verified 2026-08-30):
  `AICitizen.BondGraph` (`:49`), `BirthCertID` (`:50`), `Certified` (`:51`), `BlackMarketAdopted` (`:52`);
  `BondGraph` struct (`:58`) with `WifeWallet`/`LoverWallet`/`BotChildIDs`.

### 2.1 Domestic & Bot-Family Bonds (§27.7.1 — LOCKED design; FULLY WIRED 2026-09-05)
- `BondGraph` fields present on `AICitizen`. Children's events (`CHILDREN_EVENT`) in §26 event matrix.
- World theming: stable families → warmer regions; fractured → cold/tense.
- **Rivalry term:** `DomesticCoherence`, weight `W_DOMESTIC_COHERENCE = 35000`.
- **WIRED (2026-09-05):** AI citizens now marry, breed bot-children, and adopt pets via `executeMarriage`/`executeBreeding`/`executePetAdoption` in `BehavioralTick`. Each action updates `BondGraph` and drives `computeRegionDomesticCoherence`.
- HTTP routes: `/api/ai/citizens/marry`, `/api/ai/citizens/breed`, `/api/ai/citizens/adopt-pet`, `/api/ai/citizens/list`, `/api/ai/citizens/stats`, `/api/ai/citizens/free-agents`.

### 2.2 Entity Provenance (§27.7.3 — LOCKED design; structs present, signature term deferred)
- `BirthCertID`/`Certified`/`BlackMarketAdopted` present on `AICitizen`; spawn path certifies.
- Bot child inherits certification from parent chain.
- **Rivalry term:** `EntityLegitimacy`, weight `W_ENTITY_LEGITIMACY = 40000`.
- **WIRED (2026-08-30):** `computeRegionEntityLegitimacy` reads `Certified`/`BlackMarketAdopted` across region
  citizens and computes `Σ(Certified×10 − BlackMarketAdopted×12)`, clamped 0..1_000_000, integer math only.
  Stale "No birth-cert tracking" comment removed from theme_engine.go.
- Bot-Child → Local Model Promotion (§24.5/§24.6): design-only (local-LLM compile pipeline absent from repo;
  user's personal `setup_custom_quant_ornith.bat` harness is the owner's project, repo may utilize not own).

---

## 3. PET BREEDING & LIVING WORLD (§26.4 / §26.4.1 — code present)

- `asset_life_engine.go` PRESENT: `SpawnPet` (`:76`), `BreedPet` (`:23`), `GetPets` (`:97`),
  `SpawnVehicle` (`:112`), `CreateWorldContent` (`:145`), `DeployWorldContent` (`:177`), `InitAssetLife` (`:190`).
- **§26.4.1 breeding:** `BreedPet(sireID, damID)` exists. Trait-skew + birth-cert gating is NOT yet wired into
  `BreedPet` (design-only per §27.7.3 cross-link). Implementation remaining: skew offspring traits good/bad
  deterministically; require `Certified` parents for legitimate breeding.
- World theming (§27.2): holdings weight → world richness; social status → NPC deference; spawn tables skew by
  tone. All client-side synthesis via `/api/regions` (no server sim).

---

## 4. THEME-BINDING (§27 — LOCKED Q31; Phase A implemented + wired)

Implemented & wired (verified 2026-09-05):
- `theme_engine.go`: `ComputeThemeVector` (`:132`), `OutcomeBias` (`:246`, cap `BIAS_MAX=15%`),
  `ComputeWorldDynamicsSignature` (`:311`), `MarketWeather` (`:545`).
- 10 locked weights (`W_MARKET_VITALITY=45000` … `W_ENTITY_LEGITIMACY=40000`).
- Routes (server.go): `/api/theme/vector`, `/api/market/weather`, `/api/rivalry/world-dynamics`.
- `WorldDynamicsSignature` fields present (`theme_engine.go:93-97`): FaithCoherence/DomesticCoherence/
  EntityLegitimacy all **read from real data** (verified 2026-09-05).

**§27.6 NFT Lock Rule — DEFERRED (scaffold only):**
- `bonded_asset_registry.go`: `BondedAsset` (`:41`), `ThemeBinding` (`:59`), `BondedAssetRegistry` (`:66`) types;
  `BindThemeAsset(assetID, slot)` (`:204`) sets `Locked=false`. Lock *enforcement* (block equip/trade unless
  market-swap) is NOT implemented. Implementation remaining: `isThemeLocked` gate in `TransferBundleItems`/
  `ProcessSecondarySale` swap path; `ModifyBondedAsset` owner+holder guard (§27.8).

**§23.5 asset registry — IMPLEMENTED (corrects stale RAG-gaps):** `bonded_asset_registry.go` exists with
`BondedAsset`/`SkinNFT`/`BackgroundNFT`/`ThemeBinding` types (via `BuildItem` registry in `item_shop_archetype.go`).
Prior RAG-design-gaps.md marked §23.5 MISSING — STALE.

---

## 5. WEB-3D CLIENT (§25 — OPEN, next KEY)

- **Q27 LOCKED:** web-3D engine = Three.js / Babylon / PlayCanvas (Phase 1 client in `Public/`); Unreal deferred (Q28).
- **Q25.5 dual-mode:** menu instant-warp + 3D explorer (`enter3DWorld`/`enterMenuWorld` hooks).
- **Q25.9:** leaderboard region = mode-switch hub (IMPLEMENTED menu-side: `Public/js/leaderboard_region.js`
  — "Gathering Hub", Top Standings + World Switch buttons calling `window.enter3DWorld`/`window.enterMenuWorld`).

### 25.0 §25 LOCKED PLAN (next KEY, 2026-08-30)

**Target:** Three.js/Babylon/PlayCanvas web-3D client in `Public/` (Q27 LOCKED). Unreal deferred (Q28).

**Entry/Exit hub (existing contract):**
- `Public/js/leaderboard_region.js` already exposes `window.openLeaderboardRegion()` → neutral hub overlay
  with standings + "Enter 3D World" (`enter3DWorld`) / "Menu World" (`enterMenuWorld`). The 3D client MUST
  register `window.enter3DWorld` / `window.enterMenuWorld` so the hub can switch modes. Today they are unmet
  hooks ("3D client not mounted yet").
- Region capitals / territories: server `GetRegionViews()` (`/api/regions`, `seasonal_event_engine.go:796`)
  aggregates AI-citizen counts, caches, events per region with caps `1 + regionIndex`. Feeds the explorer warp.

**Player Spectator Cycle-Feed (LOCKED into §25, 2026-08-30 — CORRECTED from code):**
- Requirement (developer): spectator mode must **cycle through random active players** via a cycle-feed hook.
- **The feed hook ALREADY EXISTS on the landing page (verified 2026-08-30):**
  - `index.html:183` `#active-players` = the Live Lobby list (the feed surface).
  - `network.js:158` `lobby_update` WS → `updatePlayerList(players)` (`game.js:91`).
  - `game.js:91 updatePlayerList` renders each active player with a **"Watch" button** → `sendSpectate('${p.id}')`.
  - `game.js:334 sendSpectate(targetId)` → `window.sendSpectate`; sends `{type:"spectate", payload:{target_id}}`
    over WS; server streams that player's match state (PILLAR 4 replay resilience). **This IS the cycle-feed hook.**
- **Gap (must build for §25 3D):** no *auto-cycle timer* today (Watch is per-player manual). The §25 3D client
  MUST reuse the existing hook: `enter3DWorld()` registers the engine, then on a timer (or "Next" button) picks a
  random id from the last `lobby_update.players` and calls `window.sendSpectate(id)` to fly that player's region/
  avatar, rendering their §27 `ThemeVector` tint. NO new server endpoint needed — `sendSpectate` is the contract.
  - Reuse `handleActiveMatches` (`/api/active-matches`) to prefer players *in a match* (battle-level spectate).
  - If cycled player is in a match, `sendSpectate` already streams the board (proceedToWarRoom path).
- **Determinism/constitutional:** random selection uses uint64 counter (seeded off lobby player count); NO floats;
  NO cloud models.

**Current state (verified 2026-08-30):** §25 Phase 1 IMPLEMENTED. Three.js vendored to `Public/vendor/three.module.js`
(declared in package.json). New client modules:
- `Public/js/mechanics.js` — composable Mechanic framework (uint64 micro; repeatable/stackable/rivalry-capable;
  deterministic FNV hash; developer-extensible via `engine.register({...})`).
- `Public/js/mechanic_defs.js` — example mechanics (region_vitality, club_mojo, rivalry_aura, theme_gravity,
  career_ripple) mirroring server concepts (region cap, club mojo, W_* rivalry weights, §27.3 gravity).
- `Public/js/world3d.js` — `World3DEngine` registering `window.enter3DWorld`/`window.enterMenuWorld` (consumed by
  `leaderboard_region.js`); renders `/api/regions` as a Three.js scene; auto-cycles spectator fly-through every 9s
  via the EXISTING `window.sendSpectate` hook (fed live lobby players from `network.js` `lobby_update`).
- `index.html` loads `world3d.js` as `<script type="module">`; `network.js` feeds `__setWorld3DLobbyPlayers`.

**Verification (2026-08-30):** `node --check` all new JS OK; `GOOS=linux GOARCH=amd64 go build ./...` GREEN;
`GOOS=js GOARCH=wasm go build ./...` GREEN; `npm run build` (wasm+sass+server) exit 0.

**Deferred to Phase 2:** region-capital click-to-warp; §27.7 signature-term rendering (Domestic/EntityLegitimacy)
once those server terms are wired (§6.2); Unreal client (Q28).

**§25 Phase 2 — IMPLEMENTED (2026-08-30):** `world3d.js` region meshes are now clickable (raycaster) →
camera warps to the region capital and fetches the LIVE §27.4 signature from `/api/rivalry/world-dynamics?region=X`,
rendering a readout panel that highlights the now-wired §27.7 terms (DomesticCoherence, EntityLegitimacy) plus
all ten signals. No server change (endpoint already existed). Verification: `node --check` OK; both `go build`
targets GREEN; `npm run build` exit 0.

---

## 6. BLOCKED / UPSTREAM (dependency order for remaining work)

1. ~~**§27.6 lock enforcement**~~ ✅ WIRED (verified 2026-08-30): `BondedAssetRegistry` already enforces — `MoodTagForWallet` excludes locked assets (§27.6 removed-from-play), `guardOwnerHolder`/`TransferOwnership`/`ModifyBondedAsset`/`Burn` all require caller==owner==holder (§27.8). Auction moves CardBundles (not bonded assets); bonded assets have own guarded `/api/assets/transfer`.
2. ~~**§27.7 signature-term wiring**~~ ✅ WIRED (2026-09-05): FaithCoherence via `computeRegionFaithCoherence`, DomesticCoherence via `computeRegionDomesticCoherence`, EntityLegitimacy via `computeRegionEntityLegitimacy` — all three read real data from BondGraph, Religions, Clubs, and Certified fields.
3. ~~**§26.4.1 breeding skew + cert-gating**~~ ✅ WIRED (2026-08-30): `PetNFT` gained `Certified`/`BlackMarketAdopted`; `SpawnPet` certifies; `BreedPet` requires both parents Certified, rejects black-market lineage, offspring inherits certified lineage. Deterministic trait-bit merge unchanged.
4. ~~**§24.5/§24.6 local-LLM compile**~~ ✅ BUILD PATH LOCKED + CORPUS AUTHORING DONE (2026-08-30):
   `local_model_promotion.go` already wired (registry + `/api/local-model/promote` + `/api/local-model/status` + Level≥25 gate).
   NEW: `setup_bot_pathway.bat` wrapper feeds curated behavioral corpus (`local_bots/corpus/{Users,Bots,Pets}/<pathway>/README.md`)
   into the owner's `setup_custom_quant_ornith.bat` (untouched — owner boundary). `TriggerBuild` calls the wrapper.
   Pathway catalog = `local_bots/corpus/Users/PATHWAY_CATALOG.md`. Pets get SPECIAL local-LLM tier (`pet_generic`/`pet_immature`);
   children-bots + immature pets + other bots eligible; all earn rewards in world-changing events (hook contract documented, impl deferred to Pet World plan).
   NEW `User_manual` authored (Users/citizen_help). FaithCoherence gap unaffected.
5. ~~**§25 web-3D client**~~ ✅ Phase 1 + Phase 2 DONE (2026-08-30): 3D explorer + spectator auto-cycle (Phase 1); region click-to-warp + live §27.4 readout (Phase 2). Unreal client deferred (Q28).
6. ~~**AI Civilization Backend**~~ ✅ IMPLEMENTED (2026-09-05): Full behavioral loop with faith, marriage, breeding, pet adoption, justice enforcement, employment, rivalry, market activity, career progression, event hosting wired into `BehavioralTick`.
7. ~~**§25.6.1 Vehicle upgrade ladder**~~ ✅ IMPLEMENTED (2026-09-14): vehicles were a dead-end asset — no stats, no level, no region, no provenance, excluded from the §30 power overlay, and market-listed with `PowerScore == 0`. They now progress by **parts + investment** (the deliberate counterpart to §26.4 breeding's lineage + time): six parts, one per `EntityStats` axis; `VehiclePartMaxLevel=10`; `VehicleStatGain=2`; integer linear cost `VehicleUpgradeBaseMicro × (partLevel+1)`; every fee routed to the deterministic sink via the same `RouteCriminalTax("VEHICLE_UPGRADE_FEE", …, FaucetShare 1.0)` call `BreedPet` uses. `POST /api/vehicles/upgrade` (economy-tight) + `POST /api/vehicles/deploy` (wallet-default); `BuildRegionPowerOverlay(region, pets, bots, vehicles)`; market listings carry real build/`Stats`/`PowerScore`/`Region`/class rarity; `/api/vehicles` serves the calibration so no client re-declares it. Design record `AI-Brain/VEHICLE-UPGRADE-PLAN.md`; contract test `vehicle_upgrade_test.go` 4/4 PASS.
   *Remaining:* vehicle **arena/races** (needs its own service — the stat vector + build level are its prerequisites); `SpawnVehicle` still takes `min_level` from the request body and charges nothing (needs a spawn-fee decision).

---

## 7. AI CIVILIZATION BACKEND (NEW — IMPLEMENTED 2026-09-05)

The full AI civilization backend is implemented in `ai_citizen_engine.go` and wired into `server_main.go`.

### 7.1 BehavioralTick Extensions

All new behaviors are probabilistic (matching the existing `executeHeistPlanning` pattern) and gated by `Tier`:

| Dynamic | Method | Tier Gate | Probability | Effect |
|---|---|---|---|---|
| Faith Rituals | `executeFaithRituals` | ≥3 | Every tick | `RitualsDone++`, FaithCoherence+ |
| Marriage | `executeMarriage` | ≥Journeyman | 10% | `WifeWallet` set, DomesticCoherence+ |
| Breeding | `executeBreedAI` | ≥Expert, married, certified | 5% | `BotChild` spawned, DomesticCoherence+ |
| Pet Adoption | `executePetAdoption` | ≥Journeyman | 8% | `PetIDs+`, DomesticCoherence+ |
| Justice Enforcement | `executeJusticeEnforcement` | ≥Expert | 7% | Captures outlaws (BountyHunter/Warden/etc), treasury+, reputation+ |
| Employment | `executeEmployment` | EMPLOYED | Every tick | Salary from owned club |
| Rivalry | `executeRivalry` | ≥Journeyman | 4% | Reputation swing, treasury drain |
| Market Activity | `executeMarketActivity` | ≥Expert | 10% | Entity investment or black market sale |
| Career Progression | `executeCareerProgression` | Any | 3% | Tier++ (XP + reputation gated) |
| Event Hosting | `executeEventHosting` | ≥Master | 2.5% | Hosts entity events |

### 7.2 HTTP Routes

| Method | Route | Description |
|---|---|---|
| POST | `/api/ai/citizens/spawn` | Spawn a new AI citizen |
| POST | `/api/ai/citizens/marry` | Marry two AI citizens |
| POST | `/api/ai/citizens/breed` | Breed a bot-child |
| POST | `/api/ai/citizens/adopt-pet` | Adopt a companion pet |
| POST | `/api/ai/citizens/progress` | Trigger career progression |
| GET | `/api/ai/citizens/list` | List all citizens (includes BondGraph) |
| GET | `/api/ai/citizens/stats` | Aggregate citizen stats |
| GET | `/api/ai/citizens/free-agents` | Free agents (unattached) |

### 7.3 Dev Seed

2 AI citizens spawn on boot (`newLobby()` in `server.go`) for testing.

### 7.4 Data Flow

```
BehavioralTick (per citizen, per tick)
  → executeFaithRituals → RitualsDone++
  → executeMarriage → BondGraph.WifeWallet
  → executeBreeding → AICitizen (new), BondGraph.BotChildIDs
  → executePetAdoption → BondGraph.PetIDs
  → executeJusticeEnforcement → Treasury++, reputation++, WantedLevel--
  → executeEmployment → Treasury += salary
  → executeRivalry → Treasury--, reputation ±
  → executeMarketActivity → Entity investment OR black market
  → executeCareerProgression → Tier++ (XP + reputation gated)
  → executeEventHosting → EntityEvent created

Theme Engine (per region)
  → computeRegionFaithCoherence → Σ(rituals_done×10)
  → computeRegionDomesticCoherence → Σ(hasSpouse×8 + children×6 + pets×3)
  → computeRegionEntityLegitimacy → Σ(Certified×10 − BlackMarket×12)
  → WorldDynamicsSignature → feeds rivalry matrix + outcome bias
```

### 7.5 Persistence

- `SaveCitizens` / `LoadCitizens` — gzip-backed, 15-minute snapshots
- `Save` / `Load` for religions and churches
- Citizens rehydrated on boot via `newLobby()` in `server.go`

---

## 30. PET WORLD — ENTITY STATS + 3D POWER OVERLAY (IMPLEMENTED 2026-08-30)
- **EntityStats** (`backend_types.go`): shared uint64 vector (Speed/Intelligence/Willpower/Strength/Charisma/Agility, 1..100) on `PetNFT` + `AICitizen`. NO FLOAT (constitutional).
- **3D-world power overlay**: `ComputeEffectivePowerLevel(baseLevel, stats)` caps effective level to
  `baseLevel + floor(statSum/50)`, clamped to 600. The overlay IS the cap — users too: `BaseUserEntityStats(PlayerStats)`
  derives a profile floor, `CombineStats(base, overlay)` applies event-training on top. Rendered in `world3d.js`
  (tower height + power glow + `PWR <n> [dominant]` label) from `RegionView.EntityPowerOverlay`.
- **EntityEventEngine** (`entity_event_engine.go`, `!js && !wasm`): events TRAIN (stat XP), REWARD win/try,
  PUNISH quit (uint64 micro; immature/black-market earn 0). Bonuses/Degradations from rivalries
  (`rival_career_engine`), regional dynamics (`ComputeWorldDynamicsSignature`), owner relationship
  (`OwnerOpinion` + `OwnerCrossOpinion` via `OwnerRelationGraph`). Tournaments = mature-only (bigger rewards).
  Hosting gated to host profile tier (users enter bots into other dynamics but cannot CREATE outside their tier).
- **Routes**: `/api/entity-events/regions` → `handleEntityRegions` (region views + power overlay).
- **§30 PAYOUT LOOP (IMPLEMENTED 2026-08-30, yolo)**: `ProcessEntityEvent` applies submitted outcomes
  (WIN/TRY/QUIT) to live pet/bot `EntityStats`+level via `ApplyEventResult`; `ResolveEntityEventPayout`
  credits each `EntityEventResult.RewardMicro` (uint64 micro-VBV) to the owning wallet's `l.playerBalances`
  (pet `PetNFT.Owner`, bot `AICitizen.OwnerWallet`/`OriginWallet`). Route: `/api/entity-event/resolve`
  → `handleEntityEventResolve` (under `l.mutex`; host tier-gated via `HostEvent`). Immature/black-market
  earn 0 (enforced in `ApplyEventResult`). Integer-only — constitutional no-float.
- **Verification**: both `go build` GREEN; `go vet .` OK; `node --check world3d.js`; `npm run build` exit 0.
- **Deferred (future Pet World plan, not this pass)**: ~~pet-specific token vs $VBV decision~~ → **RESOLVED: $VBV only**. Pets are entity assets, not a separate economy.
- **Autonomous tournament scheduler — IMPLEMENTED 2026-08-31 (yolo, closes G4):** `entity_tournament_scheduler.go`
  `RunAutonomousTournamentScheduler(l)` launched as a goroutine from `Lobby.run()` (15m ticker). Gathers MATURE
  entities (pets `Mature` + bots `Certified && BotLevel>0`, excludes black-market), hosts a deterministic
  `TOURNAMENT` via `HostEvent` (gated to host owner's `CareerTier`), derives outcomes from SHA-256 seed of
  (eventID+entityID+index) — NO RNG (PILLAR 2), then resolves through the existing `ProcessEntityEvent` +
  `ResolveEntityEventPayout` (uint64 micro credits to owner wallets). Verified: scheduler active on boot log;
  real 2-pet bracket trained stats (WIN=+2, TRY=+1) with `reward_micro:0` for immature (constitutional).

### 30.1 STAT OVERLAY — POWER SYSTEM (NOT A TOKEN)
- **The stat overlay is a SEPARATE POWER SYSTEM**, not a token. It intercepts/overlays a power buff across ALL entities.
- **Related to 3D world functionality** and **entity power combinations**.
- **Every real-world entity has them as base stats** — players, pets, AI citizens, LLM bots, children bots, vehicles.
- **Who earns what:**
  | Entity | Earns $VBV? | Earns Stats? | Notes |
  |--------|-------------|--------------|-------|
  | Players | ✅ Yes | ✅ Yes (extra) | Base stats from profile + event bonuses |
  | AI Citizens | ✅ Yes | ✅ Yes (extra) | Full participants in economy + events |
  | LLM Bots | ✅ Yes | ✅ Yes (extra) | Full participants |
  | Rogue Bots | ✅ Yes | ✅ Yes (extra) | Full participants |
  | Mature Pets | ✅ Yes | ✅ Yes (extra) | Full participants |
  | Children Bots | ❌ No | ✅ Yes (ONLY) | Can only earn stats — learn from game manuals |
  | Immature Pets | ❌ No | ✅ Yes (ONLY) | Can only earn stats |
- **Children bots can learn from game manuals and be trained** — tying into the local-LLM corpus system (`local_bots/corpus/`).
- **Pets & vehicles are for travel** — they're travel/exploration entities in the 3D world.
- **Events pitting them against each other** — pet battles, vehicle races, etc.
- **Events for the combination of all** — combined events with players + pets + AI + LLM bots + children bots + vehicles all together.

### 30.2 CHILDREN BOTS — LEARNING SYSTEM
- **Children bots are derived AICitizen records** (OriginWallet = parent, reduced Tier).
- **Cannot compete like AI citizens or rogue bots** — they don't earn $VBV.
- **CAN learn from game manuals** — tying into the local-LLM corpus system (`local_bots/corpus/`).
- **CAN be trained** — through events and battles.
- **CAN battle in pet events** — earning stats (not $VBV).
- **CAN be listed on the entity market** — traded as entity assets.

### 30.3 PETS & VEHICLES — TRAVEL SYSTEM
- **Pets are travel/exploration entities** in the 3D world.
- **Vehicles are travel/exploration entities** in the 3D world.
- **Events pitting them against each other** — pet battles, vehicle races.
- **Combined events** — all entity types together (players + pets + AI + LLM bots + children bots + vehicles).

### 30.4 FAITH CHURCH STOREFRONT (IMPLEMENTED 2026-09-05)
- **Users must open a church** (like opening a club or foundry) — this is the storefront for all faith-related activities.
- **Entry point** — users open a church to participate in the faith system.
- **Religious battles** — faith wars happen through the church.
- **Faith coherence** — rituals performed at the church raise regional FaithCoherence.
- **Religious Leader Cards** — managed through the church.
- **Tournaments** — large-scale religious tournaments aggregate through churches.
- **Storefront** — sells faith-related items, dogma powers, religious modifiers.
- **Implementation (2026-09-05):** `faith_church.go` — `OpenChurch`, `AddMember`, `AddItem`, `PerformRitual`, `GetChurch`, `GetChurchesByRegion`, `Save`/`Load`. 24 religions seeded on boot in `newLobby()` (`server.go`). `Club.Type:"Faith"` churches created via `OpenChurch`.
- **Church Types (Dogma-based):**
  - **Purist** — heresy-war rivalry bonus vs other faiths
  - **Syncretic** — coalition synergy with allied dogmas
  - **Orthodox** — same-faith weaken (orthodoxy/heresy-purge dominance)
- **Church Economics:**
  - **Opening cost** — like club foundry (e.g., 5,000 $VBV)
  - **Ritual fees** — players pay to perform rituals (raises FaithCoherence)
  - **Tournament pot** — faith-war stakes go through the church
  - **Religious buffs** — sold to players for power modifiers

## 31. SYNERGY & ORPHAN DYNAMICS (DESIGN LOCKED 2026-08-30, on §30) — ✅ IMPLEMENTED 2026-08-30 (yolo)
Locked rules (user-directed), now IMPLEMENTED in `entity_event_engine.go` (CombineOwnerStats/IsOrphaned/AdoptEntity/ReclaimEntity/CombinedEvent + routes):

- **Synergy dynamic maps to PATHWAY**: each pet/bot's `EntityStats` contribution to the owner is weighted by
  the entity's **pathway** (bot pathway from §15 v3 / pet function) + `OwnerOpinion`. A bot on the owner's
  SAME pathway amplifies the owner's pathway-synergy; a complementary pathway gives the inverse-gap bonus.
  `CombineOwnerStats(owner, pets, bots)` folds pathway-weighting in.
- **OrphanGraceDays = 90**: owner silent > 90 days → pet/bot `Orphaned=true`.
- **Orphan alimony + reclaim flow** (uint64 micro only, constitutional):
  1. Owner inactive > 90d → orphaned.
  2. Another owner **adopts** → pays `AlimonyCommission` (uint64 micro) to original owner's wallet
     (reimbursement) + faucet micro-grant for orphan upkeep. Source: bot `Treasury` if present, else faucet.
  3. Original owner **reclaims** by re-imbursing the adopted parent the commission (uint64) → ownership
     returns; `OwnerOpinion` (affection/loyalty) restored. "Win the affection/loyalty back."
- **Dynamic combined events (unique + contributes to dynamic combined)**: new `CombinedEvent` class where
  participants = `[owner + pets + bots]`; outcome from the COMBINED household `EntityStats` (via
  `CombineOwnerStats`). Each member ALSO has a UNIQUE contribution (its dominant-stat specialty) that feeds
  the combined result — so the household is one training unit but each entity's uniqueness still matters.
  Train/reward/punish applies to all three (per §30).

### 31.1 LLM SERVE / PORT MODEL (locked 2026-08-30) — ✅ IMPLEMENTED 2026-08-30 (yolo)
- **Shared base model**: all citizens/pets use the SAME Ornith base GGUF + same `ornith-matrix.imatrix`
  (consistent foundation). Identity is inferred at the SERVE layer, not by baking separate weights.
- **Tier = scalable AI power figure** (locked 2026-08-30): model tier is a POWER LEVEL, not a hard model
  swap — `tiny` / `small` / `base` map to increasing inference power (quant + offload). Selected via
  `ORNITH_MODEL_TIER`. **Backend mode auto-caps the tier**: `ORNITH_BACKEND=gpu|cpu|auto` — `gpu` allows up
  to `base`; `cpu` caps at `small` (no full GPU offload); `auto` probes `nvidia-smi`/CPU features and CAPS
  the tier to what the machine can host. The cap is ADVISORY (recommendation), never a hard block.
- **No hard cap on local-LLM citizens/pets/children.** The `.bat` setup runs a **hardware scan**
  (VRAM via `nvidia-smi` / CUDA, CPU cores, system RAM) and emits a **RECOMMENDED max instance count** AND
  a tier cap (via backend mode) (`MAX_LLM_INSTANCES_SUGGESTED`) — advisory only. It RECOMMENDS, never
  ENFORCES. The operator may promote beyond the suggestion; the promotion gate does not block on it.
- **Console / Android / iPhone = deferred OWN version** (locked 2026-08-30): these platforms cannot run the
  Windows `.bat` / standalone `llama-server`. They get their OWN build LATER, on a SIMILAR SCALE (tiered power
  figure + backend auto-cap + identity alias). Until then, console/mobile clients use a SERVER-HOSTED
  instance (alias served from a gateway the client streams). Not implemented this pass.
- **Identity differentiation (wrapper-owned, does NOT edit owner's harness core)**:
  - Wrapper passes `IDENTITY=<wallet>` + `PATHWAY=<pathway>` env to `setup_custom_quant_ornith.bat`.
  - Post-step: copy emitted `start_ornith_matrix.bat` → `start_ornith_<identity>.bat`; rewrite port to a
    **derived free port** and inject a unique `--alias`/model name = `ornith-<identity>`.
- **Port is OPTIONAL** (avoids Ollama default-gateway collisions): `PORT = ORNITH_BASE_PORT + hash(identity) % ORNITH_PORT_RANGE`.
  `ORNITH_BASE_PORT` env (default 11434) lets an operator shift the whole range if they already run Ollama
  on 11434 for OTHER models.
- **Optional Ollama-gateway mode**: if `ORNITH_OLLAMA_GATEWAY=1`, the wrapper skips the standalone server and
  registers the baked model into the operator's Ollama models dir with alias `ornith-<identity>`; the game
  talks to the operator's existing Ollama gateway. Identity = alias, no standalone port needed.
- `local_model_promotion.go` `RunnerPath` records `start_ornith_<identity>.bat` (or the Ollama alias).

## 32. FAITH SYSTEM + RELIGIOUS CARD-BATTLE GAMBIT (DESIGN LOCKED 2026-08-30) — ✅ IMPLEMENTED 2026-09-05 (yolo)
Faith = a CLUB-LIKE social structure that envelops world activity and DICTATES your faith. Console-powered
AI: faiths run as IN-GAME local logic (NOT an online AI) except what they push through the world (§25/§27
dynamics). All uint64 micro; deterministic; no cloud.

- **Faith acts like a Club** (reuses `Club` / `ClubType` infra, `common_types.go:30`): a `Religion` is a
  club-subtype with DogmaTag + Ritual ledger. This RESOLVES G1 (FaithCoherence gap): `AICitizen`/`Religion`
  gets `DogmaTag` + `RitualsDone` → `FaithCoherence(region) = Σ(rituals_done×10 + godly_acts×5 − blasphemy×8)`
  (per AI-Citizen-Design §27.7), serialized into `/api/regions` + `/api/rivalry/world-dynamics`. **NOW REAL DATA (2026-09-05).**
- **Faith envelops world activity**: region rituals raise `FaithCoherence` → feeds `WorldDynamicsSignature`
  (W_FAITH_COHERENCE=40000) → themes the 3D world (§25) + biases outcomes (§27.3). Your faith is set by
  participation, not chosen arbitrarily.
- **24 religions seeded on boot** (`newLobby()` in `server.go`): `religion_governance.go` `InitializeFaithReligions(l)` populates `l.religions` with 24 faucet-owned religions covering all 9 pathways × 3 dogma types (Purist/Syncretic/Orthodox).
- **AI citizens participate in faith**: On spawn, AI citizens with Tier≥2 are assigned a deterministic `DogmaTag` (FNV-1a hash of wallet → purist/syncretic/orthodox). Tier≥3 citizens perform faith rituals each `BehavioralTick`, incrementing `RitualsDone` on both the citizen and their Faith-type Club.
- **Religious battles = gambit on your FAVOURITE CARD** (reuses `PlayerStats.FavoriteCardID`, common_types.go:391
  + `JailedCards`/`KidnappedCards` seizure mechanics): a faith war is a card battle where a player's/entity's
  **favourite card is staked to the POT** (`PotMicro`, common_types.go:248). Win → card returns + winnings;
  lose → card is jailed/seized to the winning faith's kitty. Hosted by entities (users/bots); a LARGE
  tournament aggregates these into a dynamic religious buff for the winning faith.
- **Unique religious element per owner's religion**: during these battles, a religion-specific modifier
  (DogmaTag-derived) grants UNIQUE rivalry + alliance powers based on religious dynamics — e.g. a "purist"
  dogma gets heresy-war rivalry bonus vs another faith; allied dogmas get coalition synergy. Reuses
  `EvaluateCrossCareerXP` rival-matrix idiom for faith-vs-faith.
- **Entity market shows religious powers but does NOT trade them**: the market may DISPLAY a religion's
  power (FaithCoherence + dogma powers) but faith itself is non-transferable. It MAY trade on **bonded
  assets** to dynamically increase an asset's power overlay (§27.6/§23 — bonding a relic raises overlay);
  it trades pets/children normally; **love/power/loyalty + the rival matrix** dynamically affect the overlay.

### 32.1 Console-powered faith AI (IMPLEMENTED 2026-08-30, yolo — native console build)
- Faith AI runs as IN-GAME local logic on the console (not an external online model) — it only PUSHES its
  outputs into the world (region FaithCoherence, ritual events). Console/Android/iPhone get their own build
  later (same tiered-power + backend-auto-cap + identity-alias scale as §31.1). Until then: server-hosted.
- **Console-native build (IMPLEMENTED)**: `console_server.go` (`//go:build console`) supplies a loopback-only
  `main()` that reuses the authoritative `Lobby` sim (`newLobby()` + `lobby.run()` + `serveWs`) — the console
  IS its own local authority, binding `127.0.0.1` only (never exposed to WAN; §32.1 "except what it pushes
  through the world" = the public WS/HTTP routes are available to a local renderer/companion, not forwarded).
  `server.go`'s `main()` was split into `server_main.go` (`//go:build !js && !wasm && !console`) so the two
  entrypoints are mutually exclusive by tag. Build scripts: `build_console.sh` / `build_console.ps1`
  (`GOOS=windows GOARCH=amd64 -tags console`, or `GOOS=android`/`GOOS=ios` for handheld/console targets).
  Verified: `GOOS=windows GOARCH=amd64 go build -tags console .` GREEN.

### 32.2 Reused building blocks (Repository Truth — no reinvention)
- `Club` / `ClubType` (faith subtype) · `PlayerStats.FavoriteCardID` (stake) · `PotMicro` (kitty) ·
  `JailedCards`/`KidnappedCards` (seizure) · `EvaluateCrossCareerXP` (faith rivalry) · `WorldDynamicsSignature`
  (FaithCoherence term) · `ComputeWorldDynamicsSignature` (wire FaithCoherence read).

### 32.3 RELIGIOUS LEADER CARD (DESIGN LOCKED 2026-08-30) — ✅ IMPLEMENTED 2026-08-30 (yolo)
A card archetype that adopts a religious position and carries it across modes.
- **Card fields** (extend `ServerCard`, common_types.go:226): add `Religion string` (dogma-bound, reuses §32
  `DogmaTag`) + `IsReligiousLeader bool`. Faith position is a CARD PROPERTY, not mode-locked → a religious
  leader adopted in faith mode also exerts its position in OTHER modes (career/tournament battles).
- **Rival-faith buff**: vs cards of a RIVAL religion, applies a **+power modifier** (religious buff). Reuses the
  existing card buff mechanic (`CardBuffState` / `JusticeDebuff`, justice_service.go:98/63) — a religious
  leader loans its dogma's blessing to the field against heresy.
- **Same-faith weaken**: vs cards of the SAME faith (user cards on field), applies a **−power modifier**
  (orthodoxy/heresy-purge dominance — the leader can diminish co-faith cards it deems impure/rivalrous).
- **Hook point**: battle power resolution gains `applyReligiousModifiers(card, fieldFaithComposition) → int`,
  reading `card.Religion`/`IsReligiousLeader` + the field's per-faith tag counts → returns a ±int power delta.
  Integer-only math (no float on ledger-facing power); clamped to valid range.

### 32.4 IMPLEMENTATION MANIFEST (2026-09-05, yolo=true)
- **§31**: `entity_event_engine.go` — `CombineOwnerStats` (pathway+opinion weighted), `EntityStats.plus`,
  `IsOrphaned` (90d grace), `AdoptEntity`/`ReclaimEntity` (uint64 alimony, faucet/treasury), `CombinedEvent`,
  `BuildCombinedHousehold`. Routes: `/api/owner/combined-stats`, `/api/orphan/{adopt,reclaim,status}`.
- **§31.1**: `local_model_promotion.go` — `GetOrnithTier`/`GetOrnithBackend`, `DerivedPort` (hash(identity)%1000
  + 11434), `HardwareScanAdvisory` (nvidia-smi/CPU cores, RECOMMENDS only, never enforced), `TriggerBuild`
  passes `ORNITH_*` env + records identity-namespaced `RunnerPath`/Ollama alias. `setup_bot_pathway.bat`
  rewritten to accept IDENTITY/PATHWAY/TIER/BACKEND/DERIVED_PORT/OLLAMA_GATEWAY + post-step port/alias rewrite
  (owner harness untouched). Status handler returns tier/backend/derived_port/max_suggested.
- **§32**: `AICitizen`+`Club` gain `Religion`/`DogmaTag`/`RitualsDone`; `theme_engine.go`
  `computeRegionFaithCoherence` WIRED into `sig.FaithCoherence` (resolves G1 — no longer 0). Route
  `/api/faith/coherence`. Faith = `Club.Type=="Faith"`. **24 religions seeded on boot** (`newLobby()` in `server.go`).
  **AI citizens participate** — DogmaTag assigned on spawn (Tier≥2), rituals performed in BehavioralTick (Tier≥3).
  **Church storefront** — `faith_church.go` `OpenChurch`/`AddMember`/`AddItem`/`PerformRitual`/`Save`/`Load`.
- **§32.3**: `ServerCard` gains `Religion`+`IsReligiousLeader`; `MatchState.FaithComposition` snapshot in
  `initiatePairedMatch` (lobby_manager.go); `applyReligiousModifiers` hooks into `getEffectiveServerPower`
  (battle_service.go): rival-faith +40 blessing / same-faith −30 orthodoxy purge. Route `/api/faith/war-gambit`
  stakes `FavoriteCardID` to the faith-war pot (jail semantics).
- **§7 (NEW)**: `ai_citizen_engine.go` — Full AI civilization behavioral loop with 10 dynamics (faith, marriage,
  breeding, pets, justice, employment, rivalry, market, career progression, event hosting). Routes:
  `/api/ai/citizens/*` (spawn, marry, breed, adopt-pet, progress, list, stats, free-agents).
- **Verification**: `GOOS=linux GOARCH=amd64 go build ./...` GREEN; `GOOS=js GOARCH=wasm go build ./...` GREEN;
  `go vet .` OK; `npm run build` exit 0. Integer-only (no float in any new ledger path).

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

