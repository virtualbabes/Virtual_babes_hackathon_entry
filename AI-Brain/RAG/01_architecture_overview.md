# NFT-Seduction — Architecture Overview
> System interaction map, data flows, and module dependencies.
> Read this after `00_repo_stability.md` for the full picture.

---

## 1. HIGH-LEVEL ARCHITECTURE

```
┌──────────────────────────────────────────────────────────────────────┐
│                          BROWSER (Client)                            │
│  ┌─────────┐  ┌──────────┐  ┌──────────┐  ┌─────────────────────┐  │
│  │ main.wasm│  │ Public/js│  │ SASS/CSS │  │ Three.js 3D Engine  │  │
│  │ (js/wasm)│  │ modules  │  │ styles   │  │ (world3d.js)        │  │
│  └────┬─────┘  └────┬─────┘  └──────────┘  └──────────┬──────────┘  │
│       │              │                                  │            │
│       └──────────────┴─────────── WebSocket ───────────┘            │
│                              │                                       │
└──────────────────────────────┼───────────────────────────────────────┘
                               │
                               ▼
┌──────────────────────────────────────────────────────────────────────┐
│                     GO SERVER (linux/amd64)                           │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │                     Lobby (lobby_manager.go)                    │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │  │
│  │  │ AI Engine│ │ Pet World│ │  Theme   │ │ TokenSinkRouter  │ │  │
│  │  │ (citizens│ │ (events, │ │ Engine   │ │ (economy, AMM)  │ │  │
│  │  │  bots)   │ │  stats)  │ │          │ │                  │ │  │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────────────┘ │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │  │
│  │  │ Justice  │ │ Faith    │ │ Rivalry  │ │ Seasonal Events │ │  │
│  │  │ Service  │ │ Church   │ │ Engine   │ │ (industrial loop)│ │  │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────────────┘ │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │  │
│  │  │ Career   │ │ Under-   │ │ Local    │ │ Bonded Asset    │ │  │
│  │  │ Service  │ │ world    │ │ Model    │ │ Registry        │ │  │
│  │  │          │ │ Contracts│ │ Promotion│ │ (lock enforce)  │ │  │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────────────┘ │  │
│  └────────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────────┐   │
│  │ Voi/Algo │ │ Ethereum │ │ MultiChain│ │ Persistence (disk/ │   │
│  │ Ledger   │ │ Trust    │ │ Router   │ │ chain snapshots)   │   │
│  │ Clients  │ │ Anchor   │ │          │ │                     │   │
│  └──────────┘ └──────────┘ └──────────┘ └──────────────────────┘   │
└──────────────────────────────────────────────────────────────────────┘
```

---

## 2. DATA FLOW — MATCH LIFECYCLE

```
Player A ──challenge──▶ Lobby.matchmakingPool
                            │
                            ▼
                    processMatchmaking()
                            │
                            ▼
                initiatePairedMatch() ──▶ MatchState created
                (snapshots: wallets, cunning, nurturing,                 │
                 wanted level, faith composition)                        │
                            │
                            ▼
                WS: challenge ──▶ Player B
                            │
                            ▼
                ◄──── accept ────▶
                            │
                            ▼
                Game loop (MoveData over WS)
                ├─ serverCheckCaptures() ──▶ CapturedCardInfo[]
                ├─ ComputeBoardHash() ──▶ BoardStateHash
                └─ getEffectiveServerPower() ──▶ ± modifiers
                            │
                            ▼
                verifyWinner()
                ├─ increment DNF/wins
                ├─ jailing logic (CapturedCards → Club.Jail)
                └─ tournament result (if tournament match)
                            │
                            ▼
                WS: match_finished + rewards_update
```

---

## 3. DATA FLOW — REPLAY RESILIENCE (PILLAR 4)

```
Server ──AuthoritativeFrame{SequenceID, MoveIntent, StateHash}──▶ ClientReplayEngine
                            │
                            ▼
                ProcessIncomingPacket(frame)
                ├─ frame.SequenceID == LastSequenceID+1 → applyAuthoritativeFrame()
                └─ frame.SequenceID > LastSequenceID+1 → InitiateRecovery()
                            │
                            ▼
                InitiateRecovery()
                ├─ CurrentState = RECONNECTING
                ├─ lockCanvasInteractions()
                └─ JS: requestMatchSync() ──▶ backend
                            │
                            ▼
                SyncHandshaker.CatchUpPlayer(fromSequence) → []FrameDelta
                            │
                            ▼
                ExecuteSyncHandshake() replays frames → StateSynchronized
                            │
                            ▼
                unlockCanvasInteractions()
```

---

## 4. DATA FLOW — AI CITIZEN LIFECYCLE

```
SpawnAI(lobby, ownerWallet, region, customName)
    │
    ├─ regionCap check (1 + regionIndex)
    ├─ random career from UnderworldContractTemplates
    ├─ generateAIVault(name) → dedicated 0xai... wallet
    ├─ Certified=true, BirthCertID=UUID
    ├─ own-wallet mandate enforced (wallet != ownerWallet)
    └─ Broadcast: ai_citizen_spawned
                │
                ▼
        BehavioralTick() [every 1min]
        ├─ Free-agent? → FreeAgentSeekContract()
        ├─ Career action (executeHeistPlanning, etc.)
        ├─ Combat matchmaking (career probability)
        └─ LastAction = now
                │
                ▼
        AttachOwnership(ownerWallet, tier) / DetachOwnership()
                │
                ▼
        LocalModelPromotion (if Level>=25 + Certified + owner-match)
                │
                ▼
        ┌───────────────────────┐
        │ setup_bot_pathway.bat │
        │ (feeds curated corpus  │
        │  to ornith harness)   │
        └───────────────────────┘
```

---

## 5. DATA FLOW — PET WORLD EVENT LIFECYCLE

```
HostEvent(hostWallet, region, eventTier, kind, costMicro, hostTier, participants)
    │
    ├─ Reject if eventTier > hostTier
    └─ Create EntityEvent
                │
                ▼
ProcessEntityEvent(submitted outcomes: WIN/TRY/QUIT)
    │
    ├─ ApplyEventResult() → stats delta + reward
    │   ├─ WIN: +2 stats (all), reward = 2000 micro (if mature + not black-market)
    │   ├─ TRY: +1 stats, reward = 1000 micro
    │   └─ QUIT: -4 stats, reward = 0
    │
    ├─ ComputeEffectivePowerLevel(baseLevel, stats)
    │   └─ effective = baseLevel + floor(statSum/50), capped at 600
    │
    └─ ResolveEntityEventPayout() → credits owner wallet
                │
                ▼
CombinedEvent (household = [owner + pets + bots])
    │
    ├─ CombineOwnerStats() → pathway-weighted + opinion-weighted
    └─ Each member trains/rewards/punishes
                │
                ▼
Orphan/Adopt/Reclaim
    ├─ IsOrphaned(lastActive, now) → 90 days
    ├─ AdoptEntity() → alimony (5%) to original owner + faucet grant
    └─ ReclaimEntity() → reimburse adoptive parent, restore ownership
```

---

## 6. DATA FLOW — FAUCET / TOKEN SINK

```
                        ┌──────────────────┐
                        │   Faucet Pool    │
                        │ (faucetBalance   │
                        │  Micro uint64)   │
                        └────────┬─────────┘
                                 │
                    applyDynamicScaling()
                    ratio = usable/maxCapacity
                    clamped 0.1–1.0
                                 │
                                 ▼
    ┌─────────────────────────────────────────────────┐
    │              TokenSinkRouter                     │
    │  ┌─────────────┐  ┌──────────────┐  ┌────────┐ │
    │  │ GlobalFaucet │  │ AdminMaint.  │  │ Audit  │ │
    │  │ Pool         │  │ Pool         │  │        │ │
    │  └─────────────┘  └──────────────┘  └────────┘ │
    │  ┌─────────────┐  ┌──────────────┐             │
    │  │ MarketNodes  │  │ Regional     │             │
    │  │ (AMM)        │  │ Districts    │             │
    │  └─────────────┘  └──────────────┘             │
    └─────────────────────────────────────────────────┘
                │
                ├─ RouteCriminalTax() → FaucetShare/ClubShare/GovernanceShare
                ├─ Governor payouts (PayoutScheduler, 24h)
                └─ LiquiditySamplingDaemon (24h, 14-sample sliding window)
```

---

## 7. DEPENDENCY GRAPH — SERVICES ON THE LOBBY

```
Lobby
 ├── aiEngine (*AICitizenEngine)
 │    └── citizens map → behavioral loop
 ├── entityEvents (*EntityEventEngine)
 │    ├── events map → HostEvent/ApplyEventResult/ResolveEntityEventPayout
 │    └── graph (*OwnerRelationGraph) → opinion/cross-opinion
 ├── themeEngine (*ThemeEngine)
 │    ├── ComputeThemeVector → 10 rivalry weights
 │    ├── ComputeWorldDynamicsSignature → Faith/Domestic/EntityLegitimacy
 │    ├── OutcomeBias (cap 15%)
 │    └── MarketWeather
 ├── rivalryEngine (*RivalryEngine)
 │    ├── Signatures → AssetSignature
 │    └── Rivalries → RegionRivalry
 ├── tokenSinkRouter (*TokenSinkRouter)
 │    ├── GlobalFaucetPool, AdminMaintenancePool
 │    ├── MarketNodes → EntityMarketNode (AMM)
 │    ├── RegionalDistricts → RegionalGovernanceMetric
 │    └── Audit → TokenSinkAuditReporter
 ├── bondedAssets (*BondedAssetRegistry)
 │    ├── BondedAssets map
 │    └── lock enforcement (guardOwnerHolder, TransferOwnership, ModifyBondedAsset, Burn)
 ├── localModelPromotions (*LocalModelPromotionRegistry)
 │    └── promotions map → RequestLocalModelPromotion / TriggerBuild
 ├── seasonEngine (*SeasonalEventEngine)
 │    ├── ActiveEvents map
 │    ├── Caches map (treasure hunts)
 │    └── UserEvents map
 ├── payoutScheduler (*PayoutScheduler)
 ├── clubService (*ClubService)
 ├── careerService (*CareerService)
 ├── courthouseService (*CourthouseService)
 ├── onboardingService (*OnboardingService)
 ├── achievementService (*AchievementService)
 ├── oracleService (*OracleService)
 ├── tournamentService (*TournamentService)
 ├── creatorStore (*CreatorStore)
 ├── entityInvestmentService (*EntityInvestmentService)
 ├── loanService (*LoanService)
 ├── auctionService (*AuctionService)
 ├── blackMarketService (*BlackMarketService)
 ├── counterfeitService (*CounterfeitService)
 ├── narrativeService (*NarrativeService)
 ├── playerService (*PlayerService)
 ├── justiceService (*JusticeService)
 ├── justiceHandlers (*JusticeHandlers)
 ├── identityBridge (*IdentityBridge)
 ├── contractEngine (*ContractEngine)
 └── gracePeriodMatrix (*GracePeriodMatrix)
```

---

## 8. WEBSOCKET MESSAGE CONTRACT

```
Client ──▶ Server                  Server ──▶ Client
─────────────────────────────       ─────────────────────────────
Envelope{type, from_id,             Envelope{type, from_id,
  to_id, payload}                     to_id, payload}

Core types:
  "lobby_update"                    Active players + matches
  "challenge"                       Match invite
  "move"                            Card placement
  "match_finished"                  Result + rewards
  "identity"                        Vault/config handshake
  "vault_update"                    Balance change
  "rules_update"                    Active rule changes
  "rewards_update"                  Reward notification
  "ai_citizen_spawned"              New AI citizen
  "ai_citizen_decommissioned"       AI removed
  "career_tier_demoted"             Demotion warning
  "maintenance_update"              Admin message
  "report_gloat"                    Player report
  "chat"                            Chat message
  "spectate"                        Watch request
```

---

## 9. REST API ROUTE PREFIXES

```
/api/health                      → server up check
/api/regions                     → region views (§25.3/§26/§30)
/api/theme/vector                → ComputeThemeVector (§27)
/api/market/weather              → MarketWeather (§27)
/api/rivalry/world-dynamics      → WorldDynamicsSignature (§27)
/api/faith/coherence             → FaithCoherence (§32)
/api/faith/war-gambit            → Religious card battle (§32.3)
/api/entity-events/regions       → Pet World region overlay (§30)
/api/entity-event/resolve        → Submit event outcome (§30)
/api/owner/combined-stats        → CombineOwnerStats (§31)
/api/orphan/{adopt,reclaim,status} → Orphan flow (§31)
/api/local-model/promote         → RequestLocalModelPromotion (§24.5)
/api/local-model/status          → Promotion status (§24.5)
/api/church/open                 → Open church (§32)
/api/church/get                  → Get church
/api/church/owner                → Churches by owner
/api/church/region               → Churches by region
/api/church/leaderboard          → Church leaderboard
/api/faith/*                     → Faith coherence/war routes
/api/pets/*                      → Pet spawn/breed/get
/api/cards/*                     → Card management
/api/wallet/*                    → Wallet operations
/api/leaderboard                 → Global leaderboard
/api/tournament/*                → Tournament lifecycle
/api/club/*                      → Club management
/api/career/*                    → Career progression
/api/justice/*                   → Justice system
/api/admin/*                     → Admin operations
```

---

## 10. FRONTEND STATE FLOW

```
index.html
    │
    ├─ app.js — bootstrap, WASM init
    │
    ├─ network.js — WS connection
    │    ├─ lobby_update → game.js:updatePlayerList()
    │    ├─ identity → wallet_state.js
    │    ├─ challenge → game.js:showChallenge()
    │    └─ move → game.js:applyMove()
    │
    ├─ game.js — core game loop
    │    ├─ renderBoard()
    │    ├─ sendMove()
    │    ├─ sendSpectate() → spectate.js
    │    └─ updatePlayerList() → Watch buttons
    │
    ├─ ui.js — territory map overlay
    ├─ player_profile.js — profile categories
    ├─ constellation_hub.js — Living Nexus hub
    ├─ leaderboard_region.js — region hub + 3D entry
    ├─ world3d.js — Three.js 3D engine
    │    ├─ enter3DWorld() / enterMenuWorld()
    │    ├─ render /api/regions as meshes
    │    ├─ auto-cycle spectator (9s timer)
    │    └─ click-to-warp + §27.4 readout
    ├─ mechanics.js — composable Mechanic framework
    ├─ pet_battle_arena.js — pet battles
    ├─ faith_church.js — church UI
    ├─ faith_system.js — faith system UI
    └─ faucet_dashboard.js — faucet UI
```

---

## 11. BUILD TAG MATRIX

| Tag | Target | Entrypoint | Purpose |
|---|---|---|---|
| `js && wasm` | GOOS=js GOARCH=wasm | `main.go` | Browser UI (WASM) |
| `!js && !wasm` | GOOS=linux GOARCH=amd64 | `server_main.go` | Production server |
| `!js && !wasm && !console` | GOOS=linux GOARCH=amd64 | `server_main.go` | Production server (alt) |
| `console` | GOOS=windows GOARCH=amd64 | `console_server.go` | Console authority |

---

## 12. KEY DESIGN PATTERNS

| Pattern | Where | Purpose |
|---|---|---|
| **Service on Lobby** | All `*Service` structs | Single shared state, dependency injection via `newLobby()` |
| **Mutex-per-service** | Each service has `sync.RWMutex` | Fine-grained locking, avoid global mutex contention |
| **Channel broadcast** | `Lobby.broadcast chan []byte` | WS message fan-out |
| **Envelope wrapper** | All WS messages | Type + FromID + ToID + RawMessage |
| **Deterministic hash** | FNV (mechanics.js), SHA-256 (board, tournaments) | No RNG, no float, cross-platform parity |
| **uint64 micro-units** | All ledger math | Integer supremacy, no float drift |
| **Composable Mechanic** | `mechanics.js` | Developer-extensible, repeatable/stackable/rivalry-capable |
| **ClientReplayEngine** | `main.go` (js/wasm) | Replay resilience, authoritative recovery |
| **GracePeriodMatrix** | `lobby_manager.go` | Connection quarantine, forfeit eviction |
| **Snapshot-on-spawn** | `initiatePairedMatch()` | MatchState snapshots all player stats at start |
| **BondedAsset lock** | `bonded_asset_registry.go` | Owner+holder guard on transfer/modify/burn |

---

*End of Architecture Overview. Next: read `02_economy_faucet.md` for the economy deep-dive.*
