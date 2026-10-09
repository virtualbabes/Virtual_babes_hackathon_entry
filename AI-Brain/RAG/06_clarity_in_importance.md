# NFT-Seduction — Clarity in Importance
> Distilled priority guide. What actually matters, what's broken, what to build first.
> This is the SIGNAL filter on top of the 127KB RAG. Read this BEFORE touching code.

---

## 1. THE 5 CONSTITUTIONAL LANDMINES (NEVER VIOLATE)

| # | Rule | Why | Violation Cost |
|---|---|---|---|
| 1 | **uint64 micro-units ONLY in ledger** | Float drift = economic collapse | Corrupts entire economy |
| 2 | **AI citizens MUST hold own wallets** | Personal wallet = exploit vector | Breaks provenance/orphan system |
| 3 | **Server authoritative** | Client renders only | Desync, replay breakage |
| 4 | **Both build targets must be green** | Bare `go build` = false positive | Ship broken server |
| 5 | **Manual git push only** | Brendan owns the tree | Unauthorized commits |

---

## 2. THE 10 MOST IMPORTANT FILES (IN ORDER)

| Rank | File | Why It Matters |
|---|---|---|
| 1 | `server.go` | `newLobby()` — wires EVERYTHING. All services init here. |
| 2 | `lobby_manager.go` | `Lobby.run()` — the heartbeat. Matchmaking, replay, persistence. |
| 3 | `backend_types.go` | `Lobby` struct + ALL core types. The data model. |
| 4 | `common_types.go` | `PlayerStats`, `MatchState`, `ServerCard`, `Envelope`. |
| 5 | `economy_service.go` | `applyDynamicScalingLocked` — faucet scaling. |
| 6 | `economy_processing.go` | `TokenSinkRouter.RouteCriminalTax` — ALL money flows through here. |
| 7 | `battle_service.go` | Card battle resolution + `getEffectiveServerPower`. |
| 8 | `ai_citizen_engine.go` | AI citizen spawn/behavior — the autonomous economy. |
| 9 | `entity_event_engine.go` | Pet World events, stats, power overlay, orphan/adopt. |
| 10 | `theme_engine.go` | `ComputeWorldDynamicsSignature` — 10 rivalry weights. |

---

## 3. THE 5 MOST IMPORTANT REST ENDPOINTS

| Rank | Endpoint | Why |
|---|---|---|
| 1 | `/ws` | WebSocket — ALL real-time state flows through here. |
| 2 | `/api/reward` | Faucet payout — the economy's heartbeat. |
| 3 | `/api/regions` | Region views — feeds 3D world + explorer. |
| 4 | `/api/faucet/status` | Faucet health + AMM state — the dashboard. |
| 5 | `/api/leaderboard` | Global standings — the competitive anchor. |

---

## 4. THE 5 MOST IMPORTANT WS MESSAGE TYPES

| Rank | Type | Why |
|---|---|---|
| 1 | `lobby_update` | Active players + matches — the live feed. |
| 2 | `move` | Card placement — the game itself. |
| 3 | `challenge` | Matchmaking handshake. |
| 4 | `identity` | Vault/config — client bootstrap. |
| 5 | `rewards_update` | Payout notification — economy feedback. |

---

## 5. CRITICAL GAP — FRONTEND EXPOSES ~10-15%

This is the #1 user frustration. Backend is civilization-grade; frontend is a mock-data shell.

### What's WIRED (real backend data):
- Faith Church (`/api/church/*`, `/api/faith/*`)
- Pet Battle Arena (`/api/pets`)
- World 3D (`/api/regions`, `/api/rivalry/world-dynamics`)
- Leaderboard Region (`/api/leaderboard`)
- Game combat (WebSocket `move`, `challenge`)
- Network/lobby (WebSocket `lobby_update`, `identity`)

### What's MOCK DATA (client-side only):
- Entity Market — static prices, no AMM
- Economy panel — partial mock
- Theme Engine visualization — partial
- AI Citizens management — partial
- Pet battle combat — local `startBattle()`, no server
- Pet breeding — local `performBreed()`, no server
- Constellation Hub — fallback mock when API fails

### What's SCAFFOLD (stub only):
- Dev Game Hub — **EXPLICITLY DEFERRED** (don't build until all else done)
- Launchpad — no ecosystem integration
- Infrastructure Leasing — no billing enforcement
- DLC Registry — stub products

---

## 6. THE FAUCET DYNAMICS (PRIORITY BUILD)

The user explicitly said: **"Build faucet dashboard FIRST."**

### Backend (already wired):
- `applyDynamicScalingLocked()` — `ratio = usableBalance / maxFaucetCapacity` clamped [0.1, 1.0]
- Anti-whale AMM — `slippage = 1 + (units/supply)² × 5` (buy), `1 - (units/supply)² × 2` (sell)
- 25% per-entity cap
- Exit siphon 2%
- Revenue split: 80% Faucet / 20% Governor on trade fees

### Frontend (already built):
- `faucet_dashboard.js` — health bar, scaling display, drain simulator, AMM bonding curve SVG chart
- Reads `/api/faucet/status` (real endpoint)

### What's MISSING:
- The faucet dashboard is built but NOT surfaced in the main menu action dock
- No link from player profile → faucet dashboard
- No prominent entry point for new players to see it

---

## 7. THE PLAYER PROFILE — 12 CATEGORIES, MANY INCOMPLETE

The profile is the main UI hub. Each category is a "territory inside the player's profile."

### Category Status:

| Category | Sub-Panels | Wiring |
|---|---|---|
| Lobby | Players, Spectate, Chat, Wallet | ✅ Partial (spectate state from WASM) |
| Territories | Overview, Districts, Claims | ✅ Real (clubs from WASM state) |
| Investments | Market, Portfolio, Dividends | ⚠️ Partial mock |
| Justice | Bounties, Effects, Missions | ✅ Real (`/api/justice/*`) |
| Market | Shops, Black Market, Presets | ⚠️ Partial mock |
| Character | Identity, Assets, Career, Deck | ✅ Real |
| Life Assets | Companions, Vehicles, World | ✅ Real (`/api/pets`, `/api/vehicles`) |
| AI Citizens | Roster, Spawn, Pathways | ⚠️ Partial mock |
| Events | World, Seasonal, Treasure | ✅ Real |
| Underworld | Contracts, Kidnap, Cyber | ✅ Real |
| Rivalry | Pairs, Dynamics, Territory | ✅ Real |
| Dev/Game Hub | Framework, Mechanics, Shop | ❌ DEFERRED |

---

## 8. THE CONSTELLATION HUB — THE LOCKED SOLUTION

The user explicitly said: **"Constellation hub (Living Nexus) is the locked solution."**

- Star-cluster node progression map
- Nodes: Dormant → Dawning → Alive
- 17 nodes + 3 NPCs (Anya, Vbabes, Crypto-Seraph)
- Reads `/api/player/progression` (real endpoint, fallback mock)
- Connection lines light up when both nodes Alive
- Clicking Alive node opens that panel as overlay

### Why it matters:
- It's the progressive unlock UI the user wants
- It's the "wonderous and immersive" aesthetic target
- It's the navigation framework for the whole app

---

## 9. THE DEV GAME HUB — DO NOT BUILD

The user explicitly said: **"Dev game hub is DEFERRED final build."**

- It's a storefront that SELLS the app's base framework functions to dev-users
- It must be built AFTER ALL ELSE so it can infer/reflect the full set of base framework functions
- `dev_game_hub.js` exists with a CATALOG of 40+ functions, but it's pure UI — no backend
- Building it now = building a shop with no inventory

---

## 10. THE 5 AGENT LANDMINES (FROM USER CORRECTIONS)

| # | Correction | Lesson |
|---|---|---|
| 1 | "you should have fixed this with the knowledge from the repo" | **Deep-dive FIRST.** Never invent files. |
| 2 | "remember to speak with the web-ui architect" | **ALWAYS proc the-architect** before building UI. |
| 3 | "for the 600th time its not a browser cache issue" | **Check duplicate server processes** first (`netstat -ano | grep 8090`). |
| 4 | "get your shit together" | **STOP making changes.** Analyze properly. |
| 5 | "you have forgotten the split was with authoritative server side!!" | **Backend owns state.** Frontend renders only. |

---

## 11. THE ECONOMIC INVARIANTS (NEVER BREAK)

| Invariant | Enforcement |
|---|---|
| No funds created or destroyed in transfers | `TokenSinkRouter.RouteCriminalTax` splits only |
| Zero-drift audit | `InterceptAndAudit`: `payload == faucet + club + gov + siphoned` |
| Max single payout | 1,000 $VBV (`MaxSinglePayoutMicro`) |
| Max governor payout | 2,000 $VBV (`MaxGovPayoutMicro`) |
| Reward safety limit | `MaxSinglePayoutMicro / 2` |
| Black market cap | 50 items FIFO |
| One-contract-at-a-time | `ActiveUnderworldContractID` must be empty |
| AI citizen own-wallet | Enforced at spawn (`wallet != ownerWallet`) |

---

## 12. THE 3 MOST IMPORTANT DESIGN PATTERNS

| Pattern | Where | Why |
|---|---|---|
| **Service on Lobby** | All `*Service` structs | Single shared state, dependency injection via `newLobby()` |
| **Envelope wrapper** | All WS messages | Type + FromID + ToID + RawMessage — the wire contract |
| **Deterministic hash** | FNV (mechanics.js), SHA-256 (board, tournaments) | No RNG, no float, cross-platform parity |

---

## 13. THE BUILD TAG MATRIX (NEVER FORGET)

| Tag | Target | Entrypoint |
|---|---|---|
| `js && wasm` | GOOS=js GOARCH=wasm | `main.go` (browser UI) |
| `!js && !wasm` | GOOS=linux GOARCH=amd64 | `server_main.go` (production server) |
| `console` | GOOS=windows GOARCH=amd64 | `console_server.go` (console authority) |

**Verify ALL THREE after any change.**

---

## 14. THE 5 HIGHEST-LEVERAGE BUILD TARGETS (RIGHT NOW)

| Priority | Target | Impact |
|---|---|---|
| 1 | **Faucet dashboard entry point** | User explicitly prioritized this. Exists but not surfaced. |
| 2 | **Entity Market → real AMM** | Backend has full AMM; frontend shows mock static prices. |
| 3 | **AI Citizens management → real data** | Backend has full engine; frontend is partial mock. |
| 4 | **Theme Engine visualization** | Backend computes 10 signals; frontend barely shows them. |
| 5 | **Pet battle/breeding → server** | Backend has BreedPet/SpawnPet; frontend is local mock. |

---

## 15. THE 5 LOWEST-LEVERAGE TARGETS (DON'T TOUCH)

| Target | Why |
|---|---|
| Dev Game Hub | Explicitly DEFERRED until all else done |
| Launchpad | No ecosystem integration |
| Infrastructure Leasing | No billing enforcement |
| DLC Registry | Stub products |
| Random event generation | Non-deterministic `rand`, dev-only |

---

## 16. THE 3 NPC CHARACTERS

| NPC | Role | Status |
|---|---|---|
| Anya | Tutorial helper (alive by default) | In constellation hub as node |
| Vbabes | Tutorial helper (alive by default) | In constellation hub as node |
| Crypto-Seraph | Rare reward-giver (dormant until high-tier) | In constellation hub as node ("??? — High tier") |

User will provide media assets (images) for them — folder path coming.

---

## 17. THE 12 AI CITIZEN PATHWAYS

| Pathway | Domain |
|---|---|
| P-Shadow | Gossip / JusticeRecruiter |
| P-Lockdown | Kidnapper / AOS |
| P-Ledger | Launderer / MutationLogAuditor |
| P-Syndicate | UnderworldBoss / Judge |
| P-Tax | TaxAuditor / Launderer |
| P-Peace | SectorPeacekeeper / Smuggler |
| P-Intel | IntelAgent / ArcNetOperative |
| P-Justice | JusticeRecruiter / BountyHunter |
| P-AOS | AOS / SectorPeacekeeper |
| P-Commissioner | JusticeCommissioner / TaxAuditor |
| P-Forensic | MutationLogAuditor / Launderer |
| P-Boss | UnderworldBoss / Judge |

---

## 18. THE 6 PET WORLD STATS

| Stat | Range |
|---|---|
| Speed | 1..100 |
| Intelligence | 1..100 |
| Willpower | 1..100 |
| Strength | 1..100 |
| Charisma | 1..100 |
| Agility | 1..100 |

**Effective Power Level** = `baseLevel + floor(statSum / 50)`, capped at 600.

---

## 19. THE 4 FAITH CHURCH DOGMA TYPES

| Dogma | Effect |
|---|---|
| Purist | Heresy-war rivalry bonus vs other faiths |
| Syncretic | Coalition synergy with allied dogmas |
| Orthodox | Same-faith weaken (orthodoxy/heresy-purge dominance) |

---

## 20. THE 5 SEASONAL EVENT TYPES

| Type | Effect |
|---|---|
| HARVEST_FEST | Resource gathering multiplier |
| SHADOW_AUCTION | Criminality rewards doubled |
| TERRITORY_WAR | Club territory bonuses ×2 |
| CRYPTO_RAID | Bounty capture XP bonus |
| DIAMOND_WEEK | All rewards +50% for Diamond+ rep |

---

*End of Clarity in Importance. This is the filter. Read the full RAG for detail.*
