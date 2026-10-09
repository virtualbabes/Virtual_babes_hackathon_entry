# 05 — Frontend / JS / WASM System

> RAG snapshot of the NFT-Seduction client architecture. Covers the WASM entrypoint
> (`main.go` `js/wasm`), the WebSocket message contract, the composable Mechanic
> framework, the Three.js 3D engine, and the mapping of every major UI panel to
> its backend wiring status (real API vs mock data).

---

## 1. Architecture Overview

The client is a **Split-WASM** design: the authoritative game engine lives in a Go
WASM module (`main.go` compiled to `main.wasm`), while the DOM, network, and 3D
rendering run in the browser's JavaScript layer. Communication between the two is
bidirectional via `js.FuncOf` bridges (Go → JS) and `js.Value.Call` invocations
(JS → Go).

```
┌──────────────────────────────────────────────────────────────────┐
│  Browser (index.html)                                            │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────────┐ │
│  │  app.js      │  │  network.js  │  │  ui.js / panels        │ │
│  │  (orchestr.) │  │  (WS bridge) │  │  (DOM rendering)       │ │
│  └──────┬───────┘  └──────┬───────┘  └──────────┬─────────────┘ │
│         │                 │                      │               │
│         └─────────────────┼──────────────────────┘               │
│                           │  window.* bridge                     │
│  ┌────────────────────────┴───────────────────────────────────┐  │
│  │  main.go (WASM) — authoritative Engine state machine       │  │
│  │  Engine struct, Player, Card, Board[9], Rules, Clubs, etc. │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

---

## 2. WASM Entrypoint — `main.go` (`js/wasm` build tag)

**File:** `main.go` (build constraint `//go:build js && wasm`)

The entire game engine is a single Go program targeting `wasm32`. The global
singleton `var Game = Engine{...}` (`main.go:680`) is the **single source of truth**.

### 2.1 Engine State Machine

| Field | Type | Purpose |
|-------|------|---------|
| `Phase` | `string` | `"Lobby"`, `"Setup"`, `"Active"`, `"Finished"`, `"TournamentLobby"` |
| `Players` | `[2]Player` | 2P lobby system (local + opponent/AI) |
| `Board` | `[9]*Card` | 3×3 combat grid |
| `Rules` | `map[string]bool` | `"Open"`, `"Power_copy"`, `"Power_up"`, `"Elemental_sync"`, `"Fallen_penalty"`, `"Artifact_bonus"` |
| `Turn` | `int` | 0 for P1, 1 for P2 |
| `Scores` | `[2]int` | Final scores `[P1, P2]` |
| `Clubs` | `map[string]*Club` | Global club registry |
| `ReplayEngine` | `*ClientReplayEngine` | PILLAR 4: replay resilience |
| `Alerts` | `*AlertEngine` | PILLAR 3: localized UI feedback |
| `RedirectManager` | `*ClientRedirectManager` | PILLAR 3: session eviction |
| `Signer` | `*WasmSignerHook` | PILLAR 3: wallet bridge |

### 2.2 Key Go → JS Bridges (exposed via `js.Global().Set`)

| Bridge | File:Line | Description |
|--------|-----------|-------------|
| `connectWallet` | `main.go:743` | Wallet connect → transitions to `"Setup"` phase |
| `SetAvatar` | `main.go:794` | Sets avatar URL, gloat, notice, favorite card; transitions to `"Lobby"` |
| `StartMatch` | `main.go:1945` | Begins a match (local or multiplayer); resets board, scores, turn |
| `PlaceCard` | `main.go:2012` | Places a card on the grid, runs `checkCaptures`, switches turn |
| `SyncMove` | `main.go:1265` | Receives `AuthoritativeFrame` from opponent, validates sequence |
| `SyncFullProfile` | `main.go:1066` | Ingests full server profile into local engine |
| `SyncClubs` | `main.go:1032` | Ingests global club registry |
| `SyncPortfolio` | `main.go:1226` | Updates local stock holdings |
| `SyncTournament` | `main.go:863` | Syncs tournament bracket state |
| `ResetGame` | `main.go:1916` | Resets engine to `"Lobby"` |
| `GetGameState` | `main.go:~2650` | Returns scoped state to JS (`"combat"`, `"meta"`, `"all"`) |
| `SendReward` | `main.go:827` | Triggers payout via backend fetch |
| `SetBoardState` | `main.go:1711` | Bulk-loads 3×3 grid for spectator sync |

### 2.3 PILLAR 4 — Client Replay Engine

**File:** `main.go:378–521`

The `ClientReplayEngine` handles sequence-gap detection and authoritative
recovery:

- `InitiateRecovery()` (`main.go:398`): Freezes UI, calls `requestMatchSync()` in JS
- `ProcessIncomingPacket()` (`main.go:422`): Detects gaps in `SequenceID`
- `ExecuteSyncHandshake()` (`main.go:443`): Processes catch-up frames, triggers
  overflow alert if reconstruction exceeds 10 seconds
- `applyAuthoritativeFrame()` (`main.go:497`): Applies state delta, updates
  `LastSequenceID`, `LastVerifiedStateHash`, pulses progress bar, unlocks UI
  when buffer is clear

### 2.4 PILLAR 3 — Wallet Bridge & Redirect Manager

**File:** `main.go:172–361`

- `WasmSignerHook.RequestWalletSignature()` (`main.go:184`): Invokes
  `window.WasmWalletBridge.signMarketAction()` (JS-side, `index.html:39–76`)
- `ClientRedirectManager.HandleIncomingEvictionFrame()` (`main.go:110`): Locks
  canvas, dispatches friendly alert, flushes session storage, redirects to
  `/login.html` after 4 seconds

---

## 3. WebSocket Message Contract

**File:** `Public/js/network.js` (483 lines)

### 3.1 Connection Lifecycle

| Step | File:Line | Detail |
|------|-----------|--------|
| Connect | `network.js:34–42` | `initWebSocket()` → `new WebSocket(\`${protocol}${backendHost}/ws\`)` |
| Identity sync | `network.js:50–65` | 5-second watchdog; reconnects up to 3× if no identity |
| Auto-reconnect | `network.js:73–84` | On close, retries after 3 seconds |
| Ping/Pong | `network.js:87–91, 131–140` | `sendPing()` every 30s; latency calculated from `pong` |
| Sync request | `network.js:98–105` | `requestMatchSync()` invoked by WASM when gaps detected |

### 3.2 Message Types (Server → Client)

| `msg.type` | Handler | What it does |
|------------|---------|--------------|
| `pong` | `network.js:133–139` | Calculates latency, calls `window.SyncLatency` |
| `identity` | `network.js:141–157` | Sets `myClientId`, syncs `CONFIG` (vault, asset IDs, WC project ID) |
| `lobby_update` | `network.js:158–215` | Updates player list, market ticker, bounty ticker, tournament, season, clubs, faucet balance, rumors |
| `matchmaking_status` | `network.js:216–218` | `handleMatchmakingUpdate()` — queued/idle/match_found |
| `portfolio_update` | `network.js:219–221` | `window.SyncPortfolio()` (WASM) |
| `heist_result` | `network.js:222–224` | `handleHeistResult()` (criminality.js) |
| `challenge` | `network.js:226–271` | `invite`/`accept`/`decline`/`sync_back` — matchmaking handshake |
| `match_start` | `network.js:273–277` | Sets spectator match state, shows preview |
| `sudden_death_start` | `network.js:278–297` | Redistributes hands, syncs card metadata |
| `move` | `network.js:298–312` | `window.SyncMove()` with full `AuthoritativeFrame` |
| `sync_response` | `network.js:314–326` | Pushes replay frames to WASM `window.PushReplayFrame()` |
| `turn_change` | `network.js:327–332` | Triggers combo SFX if `msg.payload.combo` |
| `chat` | `network.js:334–341` | Renders chat; detects "Match invalidated" |
| `vault_update` | `network.js:342–345` | `window.SyncVaultBalance()` |
| `rules_update` | `network.js:346–349` | `window.SyncRules()` |
| `rewards_update` | `network.js:351–354` | `window.SyncRewards()` |
| `maintenance_update` | `network.js:355–359` | `handleMaintenanceUI()` |
| `tournament_update` | `network.js:360–366` | `window.SyncTournament()` + `handleTournamentUI()` |
| `admin_notification` | `network.js:367–373` | `showToast()` + refresh admin logs |
| `kidnap_success` | `network.js:374–376` | Toast notification |
| `ransom_demand` | `network.js:377–379` | `showKidnapOverlay()` |
| `ransom_paid` | `network.js:380–383` | Toast + hide overlays |
| `insurance_recovery` | `network.js:384–386` | Toast notification |
| `rumor_update` | `network.js:387–391` | `updateActiveRumors()` |
| `achievement_unlock` | `network.js:392–393` | `handleAchievementUnlock()` |
| `justice_card_awarded` | `network.js:396–399` | `window.onJusticeCardAwarded()` + toast |
| `truth_serum_applied` | `network.js:400–403` | `window.onTruthSerumApplied()` + toast |
| `shield_active` | `network.js:404–407` | `window.onShieldActive()` + toast |
| `dashboard_refresh` | `network.js:408–410` | `window.onDashboardRefresh()` |
| `bounty_updated` | `network.js:411–414` | `window.onBountyUpdated()` + toast |
| `underworld_contract_assigned` | `network.js:416–418` | `window.onContractAssigned()` + toast |
| `underworld_contract_completed` | `network.js:420–422` | `window.onContractCompleted()` + toast |
| `seasonal_event_joined` | `network.js:425–429` | `window.SeasonalEvents.onEventJoined()` + toast |
| `seasonal_event_created` | `network.js:431–436` | `window.SeasonalEvents.onEventCreated()` + toast |
| `seasonal_event_reward` | `network.js:437–441` | `window.SeasonalEvents.onRewardReceived()` + toast |
| `seasonal_event_pool_updated` | `network.js:443–448` | `window.SeasonalEvents.onPoolUpdated()` + syncUI |
| `seasonal_event_expired` | `network.js:449–455` | `window.SeasonalEvents.onEventExpired()` + syncUI |
| `seasonal_event_activated` | `network.js:456–462` | `window.SeasonalEvents.onEventActivated()` + syncUI |

### 3.3 Message Types (Client → Server)

| `msg.type` | File:Line | Payload |
|------------|-----------|---------|
| `ping` | `network.js:90` | `{}` |
| `sync_request` | `network.js:104` | `{ last_sequence_id }` |
| `join_queue` | `game.js:62–67` | `{ deck, deck_rating }` |
| `leave_queue` | `game.js:72` | `{}` |
| `challenge` | `game.js:387–389` | `{ action, avatar, gloat, deck, faceplate }` |
| `move` | `game.js:484–493` | `{ grid_index, card_id, power }` |
| `chat` | `game.js:132–136` | `{ text }` |
| `spectate` | `game.js:339–343` | `{ target_id }` |
| `refresh_identity` | `game.js:420–425` | `{ handle, bio }` |

---

## 4. Composable Mechanic Framework

**File:** `Public/js/mechanics.js` (165 lines) — Framework
**File:** `Public/js/mechanic_defs.js` (96 lines) — Concrete definitions

### 4.1 Design Principles

- **Deterministic**: all ledger math is `uint64` micro-units only (`bigint`)
- **Integer-only**: `MICRO = 1_000_000n`; no floats in economic/scoring paths
- **Stackable**: multiple mechanics on one scope SUM their integer effects
- **Repeatable**: same mechanic can fire many times (cooldown/once-per guards)
- **Rivalry-capable**: any mechanic may carry a `rivalryTag`
- **Developer-extensible**: register new mechanics at runtime via `engine.register()`

### 4.2 Mechanic Definition Shape

```javascript
{
  id:        string,           // unique key
  kind:      'vitality'|'mojo'|'aura'|'gravity'|'tax'|'custom',
  scope:     'region'|'club'|'player'|'world',
  weight:    uint,             // integer multiplier, default 1
  repeatable:boolean,          // default true
  stackable: boolean,          // default true (effects sum vs replace)
  rivalryTag:string|null,      // contributes to rivalry matrix
  oncePer:   string|null,      // dedupe key (e.g. 'per-day')
  compute:   (ctx) => bigint,  // PURE integer micro-delta; ctx = {seed, scopeState, meta}
}
```

### 4.3 MechanicEngine API

| Method | File:Line | Description |
|--------|-----------|-------------|
| `register(m)` | `mechanics.js:77–81` | Registers a `Mechanic` instance |
| `resolve(scope, scopeKey, scopeState, ctxMeta)` | `mechanics.js:86–137` | Returns `{ totalMicro, breakdown[], rivalry{} }` |
| `accumulateRivalry(rivalryObj)` | `mechanics.js:140–145` | Aggregates rivalry across scopes |
| `getRivalryMatrix()` | `mechanics.js:147–151` | Returns persistent rivalry matrix |

### 4.4 Built-in Mechanic Definitions

| ID | Kind | Scope | Compute Logic | File:Line |
|----|------|-------|---------------|-----------|
| `M_REGION_VITALITY` | vitality | region | `citizens * 50000 + caches * 120000` | `mechanic_defs.js:15–30` |
| `M_CLUB_MOJO` | mojo | region | `mojoMicro / 100` (1% bleed) | `mechanic_defs.js:33–47` |
| `M_RIVALRY_AURA` | aura | region | `contestedMicro` (once-per-resolve) | `mechanic_defs.js:50–60` |
| `M_THEME_GRAVITY` | gravity | region | `gravityMicro` | `mechanic_defs.js:63–73` |
| `M_CAREER_RIPPLE` | vitality | player | `careerXPMicro / 10` | `mechanic_defs.js:76–86` |

### 4.5 Helpers

| Function | File:Line | Description |
|----------|-----------|-------------|
| `toMicro(units)` | `mechanics.js:19–24` | Converts integer units to micro `bigint` |
| `addMicro(a, b)` | `mechanics.js:27–28` | Safe micro addition |
| `hashU64(seedStr)` | `mechanics.js:31–38` | FNV-1a 64-bit deterministic hash |
| `microToTint(microStr, maxMicroStr)` | `mechanics.js:156–162` | Derives 0..1 view float from integer micro (VIEW ONLY) |

---

## 5. Three.js 3D Engine — `world3d.js`

**File:** `Public/js/world3d.js` (348 lines)

### 5.1 Engine Class

`World3DEngine` renders region capitals from `/api/regions` as a Three.js scene.
It uses the Mechanic framework to compute emergent vitality/rivalry tints and
auto-cycles the spectator feed via `window.sendSpectate`.

| Property | File:Line | Description |
|----------|-----------|-------------|
| `engine` | `world3d.js:21` | `MechanicEngine` instance with DEFAULT_MECHANICS |
| `regions` | `world3d.js:23` | Array from `/api/regions` |
| `regionMeshes` | `world3d.js:30` | `Map<regionKey, {group, mesh, baseColor, totalMicro}>` |
| `lastLobbyPlayers` | `world3d.js:24` | Fed by `window.__setWorld3DLobbyPlayers()` |

### 5.2 Key Methods

| Method | File:Line | Description |
|--------|-----------|-------------|
| `mount()` | `world3d.js:37–74` | Creates overlay, inits Three.js, starts render loop + spectator cycle |
| `_initThree()` | `world3d.js:76–127` | Scene, camera, lights, starfield, raycaster for click-to-warp |
| `refreshRegions()` | `world3d.js:130–139` | Fetches `/api/regions`, builds meshes |
| `_buildRegionMeshes()` | `world3d.js:141–196` | Computes vitality via mechanic engine, creates CylinderGeometry towers |
| `setLobbyPlayers(players)` | `world3d.js:212–215` | Receives live lobby list from network.js |
| `cycleSpectator()` | `world3d.js:222–237` | Calls `window.sendSpectate()` every 9 seconds |
| `_selectRegion(key)` | `world3d.js:250–257` | Click-to-warp camera + fetch region dynamics |
| `_fetchRegionDynamics(region)` | `world3d.js:259–285` | Fetches `/api/rivalry/world-dynamics`, renders §27.7 terms |
| `_loop()` | `world3d.js:287–303` | Render loop with camera lerp + mesh pulse |
| `warpToRegion(region)` | `world3d.js:306–321` | Public: warp to a region capital (consumed by AI Citizens panel) |
| `unmount()` | `world3d.js:326–335` | Cleanup overlay, renderer, event listeners |

### 5.3 Global Hooks

| Hook | File:Line | Consumer |
|------|-----------|----------|
| `window.World3DEngine` | `world3d.js:340` | singleton |
| `window.enter3DWorld` | `world3d.js:342` | `leaderboard_region.js` |
| `window.enterMenuWorld` | `world3d.js:343` | `leaderboard_region.js` |
| `window.__setWorld3DLobbyPlayers` | `world3d.js:346` | `network.js:166` |

### 5.4 Rendering Rules

- **Integer math only** in mechanic compute (uint64 micro)
- **VIEW-ONLY floats** derived via `microToTint()` for Three.js colors
- Region tower height = `(4 + (totalMicro * 40) / 100000000) * powerScale`
- Power overlay (§30 Pet World): `avg_entity_power` (0..600 integer) scales tower
- Tint from rivalry: `microToTint(rivalMicro, 5000000n)` → HSL color

---

## 6. UI Panels — Backend Wiring Status

### 6.1 Panels with REAL Backend Wiring

| Panel | File | API Endpoint | Status |
|-------|------|--------------|--------|
| **Faith Church** | `faith_church.js` | `/api/church/owner`, `/api/church/open`, `/api/church/region`, `/api/church/leaderboard` | ✅ Real API |
| **Pet Battle Arena** | `pet_battle_arena.js` | `/api/pets`, `/api/rivalry/factions` | ✅ Real API (pets), fallback mock (factions) |
| **Player Profile → Territories** | `player_profile.js` | `/api/regions` (via `loadRegions()`) | ✅ Real API |
| **Constellation Hub** | `constellation_hub.js` | `/api/player/progression` | ✅ Real API, fallback mock |
| **World 3D** | `world3d.js` | `/api/regions`, `/api/rivalry/world-dynamics` | ✅ Real API |
| **Leaderboard Region** | `leaderboard_region.js` | `/api/leaderboard` | ✅ Real API |
| **Game (Combat)** | `game.js` | WebSocket `move`, `challenge`, `sync_response` | ✅ Real-time WS |
| **Network (Lobby)** | `network.js` | WebSocket `lobby_update`, `identity`, `matchmaking_status` | ✅ Real-time WS |

### 6.2 Panels with MOCK DATA (client-side only)

| Panel | File | Fallback Behavior |
|-------|------|-------------------|
| **Constellation Hub** | `constellation_hub.js:178–187` | `applyMockData()` when `/api/player/progression` fails |
| **Pet Battle Arena → Factions** | `pet_battle_arena.js:452–457` | `generateMockFactions()` when `/api/rivalry/factions` fails |
| **Pet Battle Arena → Tournaments** | `pet_battle_arena.js:301–305` | Hardcoded `tournaments` array (T1/T2/T3) |
| **Pet Battle Arena → Combat** | `pet_battle_arena.js:337–352` | Local `startBattle()` / `generateOpponent()` — no server |
| **Pet Battle Arena → Breeding** | `pet_battle_arena.js:250–269` | Local `performBreed()` — no server |
| **Faith Church → War Gambit** | `faith_church.js:354–372` | Hardcoded "Recent Gambits" history |
| **Faith Church → Rivalry** | `faith_church.js:384–403` | Hardcoded rivalry entries (Temple of Dawn, Shadow Shrine) |
| **Player Profile → Lobby** | `player_profile.js:477–481` | Reads from `window.lastLobbyPlayers` (WS-fed) |
| **Player Profile → Investments** | `player_profile.js:~501+` | Local club data from WASM state |

### 6.3 Panel Inventory (index.html Action Dock)

| Button | File:Line | Opener |
|--------|-----------|--------|
| Constellation | `index.html:277` | `window.openConstellationHub()` |
| Player Profile | `index.html:279` | `window.openPlayerProfile()` |
| Deck Manager | `index.html:281` | `openDeckManager()` |
| District Shops | `index.html:282` | `openShopsOverlay()` |
| World Map | `index.html:283` | `openTerritoryMapOverlay()` |
| Justice Dashboard | `index.html:285` | `window.openJusticeDashboard()` |
| Entity Investments | `index.html:286` | `window.openInvestmentDashboard()` |
| Seasonal Events | `index.html:288` | `window.openSeasonalEvents()` |
| Creator Store | `index.html:290` | `window.CreatorStorefront.openStore()` |
| AI Citizens | `index.html:292` | `window.openAICitizens()` |
| World Events | `index.html:294` | `window.openWorldEvents()` |
| Leaderboard Hub | `index.html:296` | `window.openLeaderboardRegion()` |
| Rivalries | `index.html:298` | `window.openRivalryViewer()` |
| Life Assets | `index.html:300` | `window.openLifeAssets()` |
| Faith Church | `index.html:301` | `window.openFaithChurch()` |
| 3D World | `index.html:302` | `window.open3DWorld()` |
| Early Tasks | `index.html:304` | `window.openEarlyTasks()` |
| Asset Viewer | `index.html:306` | `window.openAssetViewer()` |

---

## 7. Key Type Definitions — `common_types_wasm.go`

**File:** `common_types_wasm.go` (557 lines)

This file (build constraint `//go:build js && wasm`) defines the shared types
used by the WASM engine. Critical structs:

| Struct | File:Line | Key Fields |
|--------|-----------|------------|
| `Envelope` | `common_types_wasm.go:159–164` | `{ type, from_id, to_id, payload }` — WS message wrapper |
| `AuthoritativeFrame` | `common_types_wasm.go:296–300` | `{ sequence_id, move_intent, state_hash }` |
| `ChallengeData` | `common_types_wasm.go:167–175` | `{ action, deck, avatar, gloat, rules, wanted_level, faceplate }` |
| `MoveData` | `common_types_wasm.go:178–183` | `{ grid_index, card_id, power[4], player_index }` |
| `ServerCard` | `common_types_wasm.go:210–226` | `{ id, name, power[4], rarity, owner, artifact, fatigue, loyalty, mood, fallen, scars, equipped_items }` |
| `MatchState` | `common_types_wasm.go:254–293` | Full match snapshot with players, board, rules, boosts, buffs, wagers |
| `MatchHistory` | `common_types_wasm.go:321–340` | `{ winner_id, opponent_wallet, scores, receipt_txid, bounty_reward_micro, ... }` |
| `PlayerStats` | `common_types_wasm.go:343–417` | Comprehensive player state (60+ fields) |
| `Club` | `common_types_wasm.go:30–62` | `{ id, name, owner_wallet, type, territories, treasury_micro, staff, active_buffs, leases, mojo, jail }` |
| `ItemDef` | `common_types_wasm.go:420–429` | `{ id, name, cost_micro, category, max_stack, recurring }` |
| `NetworkConfig` | `common_types_wasm.go:192–207` | `{ network_name, explorer_url, indexer_urls, node_urls, faucet_url, asset_id, app_id, chain_id }` |

### 7.1 Global Constants

| Constant | File:Line | Value |
|----------|-----------|-------|
| `MaxSinglePayoutMicro` | `common_types_wasm.go:16` | `1000 * 1000000` |
| `MaxAdminNotificationAmountMicro` | `common_types_wasm.go:17` | `5000 * 1000000` |
| `MaxAdminRewardAmountMicro` | `common_types_wasm.go:18` | `5000 * 1000000` |
| `MaxGovPayoutMicro` | `common_types_wasm.go:19` | `2000 * 1000000` |
| `MICRO` | `mechanics.js:16` | `1_000_000n` (JS bigint) |

---

## 8. App Bootstrap — `Public/app.js`

**File:** `Public/app.js` (1029 lines)

### 8.1 Lifecycle

1. `window.onload` (`app.js:182`) → `WebAssembly.instantiateStreaming(fetch("main.wasm"))`
2. `go.run(result.instance)` — starts the WASM engine
3. Beacon recovery: reads `localStorage.vbabes_state_beacon`, restores profile/board/sequence
4. `initWebSocket(handleServerMessage)` — establishes WS connection
5. `initParticleSystem()`, `buildEmptyBoard()`
6. `setInterval(sendPing, 30000)` — heartbeat
7. `window.syncUI()` — initial render
8. If beacon restored `'Active'` phase → `rejoinActiveMatch()`

### 8.2 Global Bridge Pattern

`app.js` imports functions from domain modules and exposes them on `window.*`
for inline HTML `onclick` handlers. This is the primary bridge between the static
`index.html` and the ES module system.

```javascript
window.handleWalletAction = handleWalletAction;   // app.js:42
window.toggleMatchmakingQueue = toggleMatchmakingQueue; // app.js:63
window.sendChallenge = sendChallenge;             // app.js:79
// ... 100+ more bridges
```

### 8.3 Performance Layer

- `UI_CACHE` Map (`app.js:287–289`) — memoizes `getElementById` calls
- `requestBatchedSync()` (`network.js:115–129`) — batches UI syncs via `requestAnimationFrame`
- `dashboardCache` (`app.js:292`) — prevents redundant DOM writes

---

## 9. Configuration — `Public/js/config.js`

**File:** `Public/js/config.js` (20 lines)

```javascript
CONFIG = {
  IS_LOCAL, BACKEND_URL, API_BASE, ASSET_URL,
  WC_PROJECT_ID, VOI_CHAIN_ID, ALGO_CHAIN_ID,
  VAULT_ADDRESS,    // Dynamic: synced from server on connect
  VBV_ASSET_ID,     // Dynamic: synced from server on connect
  AVOI_ASSET_ID     // Dynamic: synced from server on connect
}
```

The `VAULT_ADDRESS`, `VBV_ASSET_ID`, and `AVOI_ASSET_ID` are `null` at boot and
populated from the server's `identity` WebSocket message (`network.js:148–153`).

---

## 10. File Inventory (Public/js/*.js)

| File | Lines | Role |
|------|-------|------|
| `app.js` | 1029 | Central orchestrator, window bridge, HUD updaters |
| `network.js` | 483 | WebSocket lifecycle, message routing |
| `main.go` | 3836 | WASM game engine (Go) |
| `ui.js` | 1499 | DOM rendering, overlays, HUD widgets |
| `game.js` | 907 | Combat flow, matchmaking, chat |
| `economy.js` | — | Shops, trading, black market, leases |
| `criminality.js` | — | Heists, kidnaps, ransoms, rumors |
| `world3d.js` | 348 | Three.js region explorer |
| `mechanics.js` | 165 | Composable mechanic framework |
| `mechanic_defs.js` | 96 | Built-in mechanic definitions |
| `player_profile.js` | 828 | Contained profile hub (12 categories) |
| `constellation_hub.js` | 366 | Star-cluster progression map |
| `pet_battle_arena.js` | 488 | Pet combat, breeding, training |
| `faith_church.js` | 511 | Church storefront + rituals |
| `faith_system.js` | 120 | Faith coherence, leader cards, faith war |
| `leaderboard_region.js` | 95 | Mode-switch hub (3D ↔ menu) |
| `config.js` | 20 | Global deployment config |
| `wallet.js` | — | WalletConnect v2 + Voi provider |
| `audio.js` / `audio_context.js` | — | Sound effects + contextual ambients |
| `particles.js` | — | GPU particle effects |
| `utils.js` | — | Address shortening, asset symbols, envoi names |
| `admin.js` | — | Admin control panel |
| `deck.js` | — | Deck manager + avatar crop |
| `rivalry.js` | — | Rivalry engine |
| `leaderboard.js` | — | Hall of Fame |
| `justice_dashboard.js` | — | Justice Hegemony panel |
| `investment_dashboard.js` | — | Entity Investments panel |
| `seasonal_events.js` | — | Seasonal event cards |
| `governance.js` | — | Governance panel |
| `creator_store.js` / `creator_storefront.js` | — | Creator economy |
| `ai_citizens.js` | — | AI citizen roster |
| `underworld.js` | — | Underworld contracts |
| `world_events.js` | — | Treasure & events |
| `spectate.js` / `constellation_spectate.js` | — | Spectate flows |
| `industrial_loop.js` | — | Industrial loop |
| `infrastructure_lease.js` | — | Leasing panel |
| `entity_market.js` / `entity_shares.js` | — | Entity trading |
| `children_bots.js` | — | Children bots |
| `life_assets.js` | — | Life assets (pets, vehicles) |
| `launchpad.js` | — | Launchpad |
| `gaming_os.js` | — | Gaming OS |
| `stat_overlay.js` | — | Stats overlay |
| `religion_governance.js` | — | Religion governance |
| `persistent_identity.js` | — | Identity management |
| `early_tasks.js` | — | Early tasks tutorial |
| `rivalry_viewer.js` | — | Rivalry matrix viewer |
| `wallet_state.js` / `wallet_modal.js` | — | Wallet state + modal |
| `mock_api.js` | — | Mock API fallback |

---

## 11. Pillar Reference (Client-Side)

| Pillar | Concept | Key Files |
|--------|---------|-----------|
| PILLAR 1 | Infrastructure Prestige (Mojo, Stabilizers, Sabotage) | `main.go:604–606`, `app.js:298–318,449–474,642–656` |
| PILLAR 2 | Economic Safety Caps (uint64 integer supremacy) | `common_types_wasm.go:16–19`, `mechanics.js:16–38` |
| PILLAR 3 | Switchboard Security (wallet bridge, eviction, underworld) | `main.go:93–361`, `network.js:367–422` |
| PILLAR 4 | Replay Resilience (sequence engine, board hash) | `main.go:378–521`, `network.js:98–105` |
| PILLAR 5 | Deterministic Parity (isomorphic client/server logic) | `main.go:524–538`, `game.js:440–500` |
| PILLAR 6 | Specialized Feedback (audio, mutation visuals) | `app.js:264–283,405–425` |
| PILLAR 7 | Hegemony Alignment (factions, boosts) | `main.go:2197–2227`, `game.js:109` |

---

*Generated from direct file reads. All line anchors verified against source.*
