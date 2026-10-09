# NFT-Seduction — Repo Stability File (RSF)
> Master RAG anchor for the codebase. Generated 2026-09-04 by Grip (vbabes).
> This file is the SINGLE SOURCE OF TRUTH for repo structure, conventions, and agent coordination.
> All agents MUST read this file before making changes. Update it after every structural change.

---

## 1. REPO IDENTITY

| Field | Value |
|---|---|
| **Name** | NFT-Seduction (Virtualbabes Arena) |
| **Root** | `Z:\Crypto_Draught\NFT-Seduction` |
| **Branch** | Dev2 → origin/Dev2 |
| **Architecture** | Intentional split-WASM: client (js/wasm) + server (linux/amd64) |
| **Backend** | Go (authoritative sim) — `//go:build !js && !wasm` |
| **Frontend** | Go→WASM (`main.go` js/wasm) + SASS/JS modules in `Public/` |
| **Blockchain** | Algorand $VBV (Voi Mainnet primary, Algorand Mainnet secondary) |
| **Ledger rule** | uint64 micro-units ONLY — no float in ledger/economy math |
| **Dev server** | Port 8090 (`server-bin.exe`) |
| **REST routes** | ~322 |
| **WS handlers** | ~69 |
| **Go services** | 82 files |
| **JS modules** | 65+ in `Public/js/` |
| **Knowledge graph** | 6,780 nodes / 26,852 edges (codebase-memory-mcp) |

---

## 2. SPLIT-WASM BUILD REALITY (CRITICAL)

A bare `go build` defaults to wasm and EXCLUDES server files. **Both targets must be green:**

```bash
# Server (authoritative)
GOOS=linux GOARCH=amd64 go build ./...

# Client (browser UI)
GOOS=js GOARCH=wasm go build ./...

# Console authority
GOOS=windows GOARCH=amd64 go build -tags console .

# Frontend build (SCSS + WASM)
npm run build

# JS syntax check
node --check Public/js/<file>.go
```

**Verification triple:** both `go build` targets + `npm run build` must be green after any change.

---

## 3. CONSTITUTIONAL CONSTRAINTS (HARD RULES)

1. **uint64 ledger math ONLY** — no float64/double anywhere in ledger, $VBV economy, or economic simulation. Floats permitted for UI display only. Never feed float to ledger logic.
2. **AI citizens MUST hold OWN wallets** — enforced at spawn (Certified path). Forbidden from personal wallets.
3. **AI citizens forbidden from becoming developers** — no leasing/launching infra.
4. **NO cloud models** — all local inference only.
5. **Server authoritative** — frontend renders, backend owns state.
6. **Manual git push only** — agent builds/edits, Brendan commits/pushes.
7. **Owner boundary** — `setup_custom_quant_ornith.bat` is Brendan's personal project; repo UTILIZES but does NOT edit it.

---

## 4. FILE MAP — CORE ARCHITECTURE

### 4.1 Entrypoints
| File | Role |
|---|---|
| `main.go` (js/wasm) | WASM client entrypoint — AlertEngine, ClientReplayEngine, WasmSignerHook |
| `server.go` | Server entrypoint — newLobby(), CORS, WS upgrade, network config, justice wrappers |
| `server_main.go` | `//go:build !js && !wasm && !console` — production main() |
| `console_server.go` | `//go:build console` — console authority main() loopback |

### 4.2 Foundational Types
| File | Key Types |
|---|---|
| `common_types.go` | Club, ServerCard, MatchState, PlayerStats, NetworkConfig, Envelope, MoveData |
| `common_types_wasm.go` | WASM-only type aliases |
| `backend_types.go` | Lobby, Client, TokenSinkRouter, EntityMarketNode, SeasonalEventEngine, EntityStats, PetNFT, VehicleNFT, RivalryEngine, AssetSignature |

### 4.3 Core Services (82 Go files)
| File | System | Status |
|---|---|---|
| `lobby_manager.go` | Central Lobby.run() loop, matchmaking, replay frames, leaderboard persistence | ✅ WIRED |
| `battle_service.go` | Card battle resolution, capture logic, board hash, religious modifiers | ✅ WIRED |
| `server.go` | newLobby() bootstrap, all service init, WS serve, HTTP handlers | ✅ WIRED |
| `ai_citizen_engine.go` | AICitizen spawn/behavior, 12 pathways, free-agent job-seeking | ✅ WIRED |
| `entity_event_engine.go` | §30 Pet World: events, stats, power overlay, orphan/adopt/reclaim | ✅ WIRED |
| `entity_tournament_scheduler.go` | Autonomous tournament scheduler (15m ticker) | ✅ WIRED |
| `theme_engine.go` | §27 ThemeVector, WorldDynamicsSignature, 10 rivalry weights | ✅ WIRED |
| `rivalry_engine.go` | §25.10 Region/Territory rivalry matrix | ✅ WIRED |
| `rival_career_engine.go` | Career rivalry pairs, cross-XP evaluation | ✅ WIRED |
| `asset_life_engine.go` | §26.4 PetNFT breed/spawn, §25.6 vehicles, §25.7 world content | ✅ WIRED |
| `bonded_asset_registry.go` | §23.5 BondedAsset, ThemeBinding, lock enforcement | ✅ WIRED |
| `local_model_promotion.go` | §24.5/§24.6 Local-model promotion bridge, ornith harness | ✅ WIRED |
| `faith_church.go` | §32 Faith church storefront, rituals, leaderboard | ✅ WIRED |
| `seasonal_event_engine.go` | Seasonal events, treasure caches, user events | ✅ WIRED |
| `stat_overlay.go` | §30 Power overlay computation | ✅ WIRED |
| `economy_service.go` | Economic engine core | ✅ WIRED |
| `economy_processing.go` | Economic processing pipeline | ✅ WIRED |
| `economy_bootstrap.go` | Bootstrap authoritative state from disk/chain | ✅ WIRED |
| `economy_audit.go` | TokenSink audit reporter, invariant monitoring | ✅ WIRED |
| `economy_persistence.go` | 15-min persistence sync worker | ✅ WIRED |
| `economy_telemetry.go` | Telemetry logger on port 9090 | ✅ WIRED |
| `faucet_service.go` | Faucet dynamics, applyDynamicScaling(), anti-whale AMM | ✅ WIRED |
| `market_service.go` | AMM trading, market weather | ✅ WIRED |
| `entity_market.go` | Entity share market, dividend tracker | ✅ WIRED |
| `entity_shares.go` | Entity share issuance/trading | ✅ WIRED |
| `entity_investment_service.go` | §30 entity investment layer | ✅ WIRED |
| `justice_service.go` | Justice hegemony path, power bonuses | ✅ WIRED |
| `justice_handlers.go` | Justice HTTP presentation layer | ✅ WIRED |
| `career.go` | Career tiers, salary dispenser, XP tracking | ✅ WIRED |
| `underworld_contracts.go` | Dynamic underworld contract engine, templates | ✅ WIRED |
| `counterfeit_service.go` | Counterfeit detection + rate limiting | ✅ WIRED |
| `black_market_service.go` | Black market trading | ✅ WIRED |
| `advertising.go` | Advertising system | ✅ WIRED |
| `employment_service.go` | Employment/club staff management | ✅ WIRED |
| `creator_economy.go` | Creator economy royalties | ✅ WIRED |
| `creator_store_service.go` | Creator storefront | ✅ WIRED |
| `loan_service.go` | Loan marketplace, default processing | ✅ WIRED |
| `auction_service.go` | Auction house | ✅ WIRED |
| `narrative_service.go` | Narrative/event storytelling | ✅ WIRED |
| `oracle_service.go` | Blockchain oracle, indexer queries, vault balance | ✅ WIRED |
| `replay_engine.go` | Replay frame capture/playback | ✅ WIRED |
| `infrastructure_lease.go` | Infrastructure leasing system | ✅ WIRED |
| `item_service.go` | Item shop services | ✅ WIRED |
| `item_shop_archetype.go` | Item archetype registry (v4 §14) | ✅ WIRED |
| `shop_registry.go` | Shop registry | ✅ WIRED |
| `launchpad.go` | Token launchpad | ✅ WIRED |
| `governance.go` | Governance voting | ✅ WIRED |
| `religion_governance.go` | Faith governance | ✅ WIRED |
| `club_service.go` | Club management, mojo, leases, alliances | ✅ WIRED |
| `tournament_manager.go` | Tournament lifecycle | ✅ WIRED |
| `onboarding_service.go` | New player onboarding | ✅ WIRED |
| `achievement_service.go` | Achievement/trophy system | ✅ WIRED |
| `player_service.go` | Player profile, hegemony path resolution | ✅ WIRED |
| `identity_bridge.go` | Cross-platform identity ownership | ✅ WIRED |
| `persistent_identity.go` | Persistent identity rehydration | ✅ WIRED |
| `bridge_router.go` | Multi-chain bridge routing | ✅ WIRED |
| `bridge_service.go` | Bridge service | ✅ WIRED |
| `ethereum_client.go` | Ethereum trust anchor client | ✅ WIRED |
| `nautilus_dex_path.go` | Nautilus DEX path (PILLAR 2 console creator payouts) | ✅ WIRED |
| `industrial_loop.go` | Industrial loop (PILLAR 1) | ✅ WIRED |
| `redemption_gateway.go` | Underworld recovery/redemption | ✅ WIRED |
| `gaming_os.go` | Gaming OS layer | ✅ WIRED |
| `handlers_admin.go` | Admin HTTP handlers | ✅ WIRED |
| `handlers_criminality.go` | Criminality HTTP handlers | ✅ WIRED |
| `handlers_public.go` | Public HTTP handlers | ✅ WIRED |
| `handlers_rumor.go` | Rumor HTTP handlers | ✅ WIRED |
| `resilience_utils.go` | Resilience utilities | ✅ WIRED |
| `rate_limiter.go` | Rate limiting service | ✅ WIRED |

---

## 5. FRONTEND JS MODULES (Public/js/)

| File | Role | Backend Wiring |
|---|---|---|
| `game.js` | Core game loop, player list, spectate | ✅ WS lobbym_update |
| `ui.js` | Territory map overlay, main UI | ✅ Partial |
| `network.js` | WS connection, lobby_update handler | ✅ WS |
| `mechanics.js` | Composable Mechanic framework (uint64, FNV hash) | ✅ Client-side |
| `mechanic_defs.js` | Example mechanics (region_vitality, club_mojo, etc.) | ✅ Client-side |
| `world3d.js` | Three.js 3D world engine, spectator auto-cycle | ✅ /api/regions, /api/rivalry/world-dynamics |
| `leaderboard_region.js` | Region hub, enter3DWorld/enterMenuWorld hooks | ✅ /api/leaderboard |
| `constellation_hub.js` | Living Nexus constellation hub | ⚠️ Partial |
| `player_profile.js` | Player profile categories | ⚠️ Partial |
| `pet_battle_arena.js` | Pet battle arena | ✅ /api/pets |
| `faith_church.js` | Faith church UI | ✅ /api/church/* |
| `faith_system.js` | Faith system UI | ✅ /api/faith/* |
| `faucet_dashboard.js` | Faucet dashboard | ✅ /api/faucet/* |
| `economy.js` | Economy panel | ⚠️ Partial mock data |
| `entity_market.js` | Entity market | ⚠️ Mock static prices |
| `menu-constellation.js` | Constellation menu | ⚠️ Partial |
| `menu-dock.js` | Dock menu | ⚠️ Partial |
| `spectate.js` | Spectate mode | ✅ WS spectate |
| `deck.js` | Deck management | ✅ /api/cards |
| `wallet.js` | Wallet management | ✅ /api/wallet |
| `wallet_modal.js` | Wallet modal | ✅ |
| `wallet_state.js` | Wallet state | ✅ |
| `config.js` | Client config | ✅ |
| `utils.js` | Utility functions | ✅ |
| `mock_api.js` | Mock API fallback | ⚠️ Dev only |
| `error_handler.js` | Client error reporter | ✅ /api/client-error |
| `audio.js` / `audio_context.js` / `audio_engine.js` | Audio system | ✅ |
| `particles.js` | Particle effects | ✅ |
| `tx_modal.js` | Transaction modal | ✅ |
| `devsim.js` | Dev simulation | ⚠️ |
| `dev_game_hub.js` | Dev game hub (DEFERRED final build) | ❌ Deferred |
| `achievements.js` | Achievements panel | ✅ |
| `admin.js` / `admin_panel.js` | Admin panels | ✅ |
| `advertising.js` | Advertising panel | ✅ |
| `ai_citizens.js` | AI citizen management | ⚠️ Partial |
| `asset_viewer.js` | Asset viewer | ✅ |
| `bounty_tracker.js` | Bounty tracking | ✅ |
| `bridge_router.js` | Bridge routing UI | ✅ |
| `children_bots.js` | Children bots panel | ⚠️ Partial |
| `combined_events.js` | Combined household events | ✅ |
| `creator_economy.js` | Creator economy | ✅ |
| `creator_store.js` / `creator_storefront.js` | Creator store | ✅ |
| `criminality.js` | Criminality panel | ✅ |
| `daily_challenges.js` | Daily challenges | ✅ |
| `early_tasks.js` | Early tasks/onboarding | ✅ |
| `entity_shares.js` | Entity shares | ✅ |
| `governance.js` | Governance panel | ✅ |
| `industrial_loop.js` | Industrial loop panel | ✅ |
| `infrastructure_lease.js` | Infrastructure leasing | ✅ |
| `investment_dashboard.js` | Investment dashboard | ✅ |
| `justice_dashboard.js` | Justice dashboard | ✅ |
| `launchpad.js` | Launchpad | ✅ |
| `leaderboard.js` | Leaderboard | ✅ |
| `life_assets.js` | Life assets (pets/vehicles) | ✅ |
| `religion_governance.js` | Religion governance | ✅ |
| `rivalry.js` / `rivalry_viewer.js` | Rivalry panels | ✅ |
| `seasonal_events.js` | Seasonal events | ✅ |
| `settings_panel.js` | Settings | ✅ |
| `stat_overlay.js` | Stat overlay (3D power) | ✅ |
| `theme_engine.js` | Theme engine visualization | ⚠️ Partial |
| `underworld.js` | Underworld panel | ✅ |
| `world_events.js` | World events | ✅ |
| `persistent_identity.js` | Identity panel | ✅ |

---

## 6. THE 8 PILLARS (ToDo.md)

| Pillar | Domain | Status |
|---|---|---|
| P1 | Industrial & Trust | ✅ COMPLETE |
| P2 | High-Finance & Market | ✅ COMPLETE |
| P3 | Criminality & Intel | ✅ COMPLETE |
| P4 | Performative & Social | ✅ COMPLETE |
| P5 | Deep RPG | ✅ COMPLETE |
| P6 | Mutation Foundry | ✅ COMPLETE |
| P7 | Underworld Recovery & Redemption | ✅ COMPLETE |
| P8 | Rivalry | ✅ COMPLETE |

---

## 7. THEME ENGINE — 10 LOCKED RIVALRY WEIGHTS (§27)

| Weight | Value | Domain |
|---|---|---|
| W_MARKET_VITALITY | 45,000 | Market |
| W_ECONOMIC_FLOW | 40,000 | Economy |
| W_CULTURAL_RESONANCE | 35,000 | Culture |
| W_INFRASTRUCTURE_INTEGRITY | 30,000 | Infrastructure |
| W_SOCIAL_COHESION | 25,000 | Social |
| W_JUSTICE_PRESSURE | 20,000 | Justice |
| W_CRIMINAL_TENSION | 15,000 | Criminality |
| W_FAITH_COHERENCE | 40,000 | Faith |
| W_DOMESTIC_COHERENCE | 35,000 | Domestic |
| W_ENTITY_LEGITIMACY | 40,000 | Provenance |

Routes: `/api/theme/vector`, `/api/market/weather`, `/api/rivalry/world-dynamics`

---

## 8. FAUCET DYNAMICS & ANTI-WHALE (§27.4 / faucet_service.go)

- `applyDynamicScaling()` scales all rewards by `faucetRatio = usableBalance / maxFaucetCapacity` (clamped 0.1–1.0)
- Anti-whale AMM: quadratic bonding curve `slippageFactor = 1 + (units/supply)² × 5`
- 25% per-entity cap
- **NONE visible in frontend** — Entity Market shows mock static prices
- With only 7000 $VBV test env, faucet dashboard is the priority build

---

## 9. FAITH SYSTEM (§32)

- Faith = Club with `Type=="Faith"` + DogmaTag + RitualsDone
- FaithCoherence = `Σ(rituals_done×10 + godly_acts×5 − blasphemy×8)` — resolves G1
- Religious Leader Card: `Religion` + `IsReligiousLeader` on ServerCard
- Rival-faith blessing: +40 power; same-faith orthodoxy purge: −30 power
- Church storefront: open/perform rituals/leaderboard — all wired to `/api/church/*` and `/api/faith/*`

---

## 10. PET WORLD (§30)

- EntityStats: uint64 vector (Speed/Intelligence/Willpower/Strength/Charisma/Agility, 1..100)
- ComputeEffectivePowerLevel: `baseLevel + floor(statSum/50)`, capped at 600
- Events: TRAIN (stat XP), REWARD (win/try), PUNISH (quit)
- Tournaments: mature-only, bigger rewards
- Immature/black-market earn 0 (constitutional)
- Autonomous tournament scheduler: 15m ticker, SHA-256 deterministic outcomes
- Routes: `/api/entity-events/regions`, `/api/entity-event/resolve`

---

## 11. AI CITIZENS (§3 / §15)

- AICitizen: own wallet (mandatory), career, tier (0–6), pathway (12 tracks)
- Region cap: `1 + regionIndex`
- 12 pathways: P-Shadow, P-Lockdown, P-Ledger, P-Syndicate, P-Tax, P-Peace, P-Intel, P-Justice, P-AOS, P-Commissioner, P-Forensic, P-Boss
- Behavioral tick: 1min interval, career-dependent actions + combat matchmaking
- Bot children: derived AICitizen (OriginWallet = parent, reduced Tier)
- BondGraph: WifeWallet/LoverWallet/BotChildIDs/PetIDs
- Provenance: Certified/BirthCertID/BlackMarketAdopted

---

## 12. ORPHAN / ADOPT / COMBINED EVENTS (§31)

- OrphanGraceDays = 90
- AdoptEntity: adoptive parent pays AlimonyCommission (5%) to original owner + faucet micro-grant
- ReclaimEntity: original owner reimburses adoptive parent, restores ownership + OwnerOpinion=100
- CombinedEvent: [owner + pets + bots] act as ONE training unit
- CombineOwnerStats: pathway-weighted + opinion-weighted contribution
- Routes: `/api/owner/combined-stats`, `/api/orphan/{adopt,reclaim,status}`

---

## 13. LOCAL-LLM PROMOTION (§24.5/§24.6/§31.1)

- MinPromotionLevel = 25 (ornith harness Level gate)
- Eligibility: Certified, NOT BlackMarketAdopted, owner-match
- Tier model: tiny/small/base = scalable AI power figure
- Backend auto-cap: gpu|cpu|auto (advisory only, never enforced)
- Port: `11434 + hash(identity) % 1000`
- Optional Ollama-gateway mode
- setup_bot_pathway.bat wrapper feeds curated corpus into owner's ornith harness

---

## 14. CRITICAL GAPS (Backend civilization-grade, frontend exposes ~10-15%)

| Gap | Status |
|---|---|
| Battle board UI | ⚠️ Partial |
| AI citizen management UI | ⚠️ Partial |
| Entity event resolution UI | ⚠️ Partial |
| Theme engine visualization | ⚠️ Partial |
| Infrastructure leasing UI | ✅ Wired |
| Justice missions UI | ✅ Wired |
| Heist planning UI | ✅ Wired |
| Kidnap/ransom UI | ✅ Wired |
| Courthouse UI | ✅ Wired |
| Career advancement UI | ✅ Wired |
| Loan marketplace UI | ✅ Wired |
| Auction house UI | ✅ Wired |
| Black market UI | ✅ Wired |
| AMM trading UI | ⚠️ Mock prices |
| Region management UI | ⚠️ Partial |
| Dev game hub | ❌ DEFERRED final build |

---

## 15. AGENT COORDINATION PROTOCOL

| Rule | Detail |
|---|---|
| **ALWAYS proc the-architect** | Message @the-architect for design direction BEFORE building UI |
| **ALWAYS deep-dive first** | Read full codebase before implementing — never invent new files when existing systems can be extended |
| **NEVER suggest browser cache** | Check for duplicate server processes first (`netstat -ano | grep 8090`) |
| **ALWAYS verify both build targets** | `GOOS=linux GOARCH=amd64` + `GOOS=js GOARCH=wasm` + `npm run build` |
| **Manual git push** | Agent builds/edits, Brendan commits/pushes |
| **uint64 only** | No float in ledger math |
| **Server authoritative** | Frontend renders, backend owns state |

---

## 16. RAG FILE INDEX

| File | Content |
|---|---|
| `AI-Brain/RAG/00_repo_stability.md` | THIS FILE — master anchor |
| `AI-Brain/RAG/01_architecture_overview.md` | System interaction map, data flows |
| `AI-Brain/RAG/02_economy_faucet.md` | Economy/Faucet/AMM deep-dive (subagent) |
| `AI-Brain/RAG/03_theme_rivalry.md` | Theme/Rivalry/Region deep-dive (subagent) |
| `AI-Brain/RAG/04_justice_criminality.md` | Justice/Criminality/Underworld deep-dive (subagent) |
| `AI-Brain/RAG/05_frontend_js.md` | Frontend/JS/WASM deep-dive (subagent) |

---

## 17. KEYBOARD SHORTCUTS FOR AGENTS

```bash
# Check dev server health
curl http://localhost:8090/api/health  # 404 = server up

# Kill stale server processes (Windows)
tasklist /FI "IMAGENAME eq server-bin.exe"
netstat -ano | grep 8090
taskkill /F /IM server-bin.exe

# Rebuild
npm run build

# Go build both targets
GOOS=linux GOARCH=amd64 go build ./... && GOOS=js GOARCH=wasm go build ./...

# Knowledge graph status
# Use mcp__codebase_memory_mcp__index_status via tool_call
```

---

*End of Repo Stability File. Last updated: 2026-09-04 by Grip.*
