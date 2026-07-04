# Virtualbabes Arena: Engineering Roadmap & Implementation Pipeline

### 📑 DOCUMENT STATUS: AUTHORITATIVE TASKS TO COMPLETE
This document is the authoritative source for the development pipeline, bridging implemented features with expansion plans. It defines the next logical steps for the Arena.
*   **Architectural Blueprint:** Consult `AI-Brain/File-Flow-Overview-1.md`
*   **Verified Completions:** Consult `AI-Brain/A.I_memory.md`
*   **Expansion Plans:** Consult `Development-production-build/Markdown_developer_volume/Game_expansion_plan.md`
*   **Behavioral Rules:** All tasks must be executed according to `AI-Brain/Rules.md`.

---

## 🎯 CURRENT BETA BASELINE (Production-Ready State)
**Status: HARDENED / BUILD-STABILIZED**

All 6 implementation roadmaps complete. Core loop verified via simulated stress tests and blockchain-native persistence. The following foundational systems are fully operational:

### ✅ Pillar 1: Industrial & Trust Layer
- [x] Domain-Driven Refactor (services decoupled from `lobby_manager.go`)
- [x] In-Game Employment (Manager, Security, Clerk roles)
- [x] Courthouse Rerouting (50% Faucet / 50% Clubs)
- [x] Club Mojo & Tiered Unlocks
- [x] Regional Expansion (Governor status, +5% power boost, tax routing)
- [x] Industrial Leases (Lender/Club/Faucet revenue splits)
- [x] Regional Alliances (shared territory counts and defensive buffs)

### ✅ Pillar 2: High-Finance & Market Layer
- [x] Art Gallery Auctions (server-authoritative internal escrow, 10% commission)
- [x] Loans & Collateral (Market Token liquidation on default)
- [x] Black Market (Wanted/Cunning-gated access, stolen card tags)
- [x] Entity Share Trading (AMM bonding curve with slippage-aware pricing)
- [x] Rumor Mill (positive/negative price multipliers, 20% Governor tax)
- [x] Token-Sink Router (forensic ledger integrity, all fees auditable)
- [x] Anti-Whale AMM (quadratic bonding curve, `market_service_test.go`)
- [x] Economy Persistence (`PersistenceSyncWorker` for disk snapshots)
- [x] Telemetry Exporter (Prometheus metrics on port 9090)

### ✅ Pillar 3: Criminality & Intelligence
- [x] Kidnap Gambit (multi-slot `VictimRegistry`, ransom economy, insurance recovery)
- [x] Bounty Board (real-time tracking of high-Wanted players)
- [x] Ghost Protocol (signal scrambling to evade bounty tracking)
- [x] Cyber-Audit / Cyber-Lock / Cyber-Counter (treasury espionage layer)
- [x] Sabotage Protocol (disable hardware defenses for heists)
- [x] District Scanner (sector-wide trap revelation)
- [x] Collective NPC Intelligence (playstyle-aware taunts)

### ✅ Pillar 4: Performative & Social Layer
- [x] Enhanced Portfolio View (valuation, share trading, trophy display)
- [x] Social Sharing (X/Twitter integration for victories)
- [x] Achievement System (Valor badges influencing market multipliers)
- [x] Social Hub (Alliance management, Career showcase)
- [x] Faction Sovereignty (+10% Coalition Defense boost for allied members)
- [x] Transient Asset Redemption (3-Win challenges for 'Fallen' cards)
- [x] Identity Sinks (100 $VBV handle/bio refresh protocol)

### ✅ Pillar 5: Deep RPG Mechanics
- [x] Fatigue/Loyalty Loop (-1 power per usage > 50; +25 at max loyalty)
- [x] Elemental Synthesis (Mood-tile alignment bonuses)
- [x] Faceplates (cosmetic items with Mojo/Cunning functionality)
- [x] Mutation Foundry (vector realignment, mood recalibration, loyalty synthesis)

### ✅ Pillar 6: Resilience & Persistence Engine
- [x] Blockchain-Native Snapshots (`VBT_ECONOMY_SNAPSHOT`, `VBT_CARD_CACHE_SNAPSHOT`)
- [x] Deterministic Replay Kernel (WASM frame sequencing with state hashing)
- [x] Client Replay Engine (state catch-up and fast-forwarding)
- [x] WASM Wallet Bridge (non-custodial Ed25519 signing from extensions)
- [x] Session Watchdog (mid-match eligibility monitoring)
- [x] 100% Blockchain-Native State Recovery (no local file persistence dependency)

### ✅ Pillar 7: Justice Hegemony Path (NEW — Verified 2026-06-18)
- [x] Core justice service (`justice_service.go`): `JusticeService`, `JusticeFaction`, `JusticeCard`, `JusticeMission`, `BountyTargetInfo` types
- [x] Power bonus engine: `GetPowerBonus()`, `CalculateJusticePowerMultiplier()` (base +10% vs Wanted≥15)
- [x] Justice card pool: ENFORCER, MEDIATOR, WARDEN, COMMISSIONER archetypes with tiered power bonuses
- [x] Truth Serum item: `ApplyTruthSerum()`, `GetRevealedBuffs()` — reveals opponent card buff/debuff state for 30s
- [x] Reputation Shield item: `ApplyReputationShield()`, `AbsorbReputationLoss()`, `GetShieldRemaining()` — shields reputation loss with capacity tracking
- [x] Justice Bounty Dashboard: `GetDashboardForPlayer()`, `FormatDashboardHTML()` — Warden+ tier access gate (3 cards + rank≥2)
- [x] Mission generation: `GenerateJusticeMission()` with 24h expiry window, COMPLETED/EXPIRED status tracking
- [x] Constants: `JusticeTierAccessCards=3`, `JusticeTierAccessRank=2` (Warden), `TruthSerumDefaultDuration=30s`, `ReputationShieldDefaultProtection=50`

### ✅ Pillar 8: Rivalry System (NEW — Verified 2026-06-18)
- [x] Backend types (`rival_career_engine.go`): `RivalXP`, `RivalryXP`, `RivalInteractionPair`, XP multipliers, rival tier bonuses, `TrackRivalInteraction()` defined (698 lines)
- [x] HTTP handlers wired in `server.go` → `rivalry_handlers.go`:
  - ✅ `POST /api/rivalry/request` — `HandleRivalryRequest()` creates pending invitations, broadcasts to target via WebSocket
  - ✅ `POST /api/rivalry/action?action={accept|decline|challenge}` — `HandleRivalryAction()` bilateral connection, invite cleanup, challenge notification
  - ✅ `GET /api/rivalry/state` — `HandleGetRivalryState()` returns active rivals + pending invites
  - ✅ `GET /api/faction/shop/{faction}` — `HandleGetFactionShop()` returns Justice/Underworld item catalogs
  - ✅ `POST /api/faction/shop/buy` — `HandleBuyFactionItem()` validates faction role, deducts micro-units, adds to inventory
  - ✅ `GET /api/career/progress` — `HandleGetCareerProgress()` returns career XP + tier data
- [x] Frontend module (`Public/js/rivalry.js`): `RivalryEngine` with `requestRival()`, `acceptRival()`, `declineRival()`, `challengeRival()`, `getFactionShop()`, `buyFactionItem()`, `syncCareerProgress()`
- [x] Shop items defined in handlers (not shop_registry — inline):
  - JUSTICE: truth_serum (2.5M μVBV, 5min), reputation_shield (3M μVBV, 60min), bounty_license (50K μVBV, recurring), arc_net_spy (5M μVBV)
  - UNDERWORLD: data_scramble (4M μVBV), signal_dampener (800K μVBV), security_override (5M μVBV), regulatory_bypass (100K μVBV)

### ✅ Build & Quality Infrastructure
- [x] Dual-Target Build (`go build` + WASM compilation)
- [x] SCSS Modular System (neon-glass theme, design tokens, a11y compliant)
- [x] Audio Engine (`AudioContext` low-latency polyphony, character-based fanfares)
- [x] Particle Effects (state-aware physics for captures, heists, foundry)
- [x] Telemetry Monitoring (Prometheus metrics, Prometheus-compatible dashboards)
- [x] Admin Tooling (Sanity Check, Emergency Shutdown, Load Simulator)

---

## 🚀 EXPANSION PIPELINE (Post-Beta)
Tasks organized by strategic priority and implementation dependency.

### 🔥 HIGH PRIORITY: Production Sealing & Mainnet Launch

#### P1-A: Secret Management & Deployment Hardening
- [ ] **Task 3001**: Wire `FAUCET_MNEMONIC` via Render Secrets (currently placeholder)
- [ ] **Task 3002**: Configure production `ADMIN_WALLETS` for multi-chain signature auth
- [ ] **Task 3003**: Finalize WalletConnect Project ID for mainnet deployment
- [ ] **Task 3004**: Configure Render volume mounts for `admin_audit.log` and blockchain state files
- [x] **Task 3005**: Production RPC endpoint hardening (LlamaRPC failover validation) ✅ — `SetProductionMode(true)` wired in `newLobby()` with hardened config: 329 max rate limits, 5s sync lag detection, 15m cooldown
- [x] **Task 3006**: Implement health check endpoints (`/health`, `/live`, `/ready`) ✅ — Routes registered in `server.go`; handlers serve liveness/readiness for orchestration (Render/K8s)

#### P1-B: Security & Sybil Finalization
- [ ] **Task 3010**: Finalize sybil protection threshold analysis for production deployment
- [x] **Task 3012**: Add CORS policy configuration for production domains ✅ VERIFIED — `AllowedOrigins` configured in `server.go` with configurable origins and `AllowCredentials`.
- [ ] **Task 3013**: Audit all admin handlers for signature replay attacks 📋 FINDINGS COMPILED — see Session-Handoff.md Section-Handoff for audit report

#### P1-C: Rate Limiting & DDoS Mitigation (NEW Pillar)
**Goal:** Prevent API abuse across all public endpoints before mainnet launch. Uses existing `httpRateLimits map[string]*RateBucket` in Lobby struct.

| Task | Component | Priority | Notes |
|------|-----------|----------|-------|
| 3101 | RateLimiterService type | 🔴 HIGH | Create `rate_limiter.go`: token bucket + sliding window logic using existing `RateBucket` |
| 3102 | Middleware wiring | 🔴 HIGH | Wrap `/api/*` routes in server.go with rate limit middleware |
| 3103 | Per-Wallet limits | 🟡 MED | Configurable limits: auth endpoints (5/min), economy actions (10/min), admin (2/min) |
| 3104 | IP-based fallback | 🟡 MED | Client-side wallet detection fails → fall back to IP header rate limiting |
| 3105 | Admin bypass | 🟢 LOW | `ADMIN_WALLETS` list gets infinite quota for operational needs |
| 3106 | Telemetry integration | 🟢 LOW | Report rate limit hits to Prometheus counter (`api_rate_limited_total`) |

### ✅ PHASE 1 COMPLETE — Infrastructure & Security Finalization — Completed 2026-06-23

**Phase 1 Summary:** Session-Handoff.md restored to v1.0 (constitutional gap resolved). Rate limiting pillar P1-C implemented (token bucket + sliding window, middleware wiring, per-wallet tiers, IP fallback, admin bypass, Prometheus telemetry). $VBV-gate validated with Gossip career as test subject. Career XP engine validated but Fence remains the only wired career. Build verified, determinism confirmed, no orphaned systems introduced.

**Phase 1 Files Modified:** `rate_limiter.go`, `server.go` (middleware wiring), `lobby_manager.go` (per-wallet limits), `career.go` ($VBV-gate integration for Gossip), `rival_career_engine.go` (validation). Session-Handoff.md created.

**Phase 1 Deliverables:**
- Rate limiting: Token bucket + sliding window per wallet/IP, admin bypass, Prometheus counters (`api_rate_limited_total`)
- $VBV-gate: Career tier thresholds now $VBV-sustained-gated (not XP-only), liquidity sampling every 24h, demotion grace periods
- Gossip career: XP trigger + fee discount + buff tag wired and validated with $VBV gate

### ✅ PHASE 2 COMPLETE — Career Wiring (Underworld + Justice) — Completed 2026-06-22

### ✅ PHASE 3 COMPLETE: CAREER XP SCALING MULTIPLIERS — COMPLETED 2026-06-25

**Phase 3 Summary:** Added `computeScaledXP(base uint64, role string, loyaltyBonus float64, fameBonus float64)` to `rival_career_engine.go`. This helper scales raw XP by `loyaltyMultiplier * fameMultiplier` before calling the existing `TrackCareerXP(role, scaledXP)`. All 20+ existing `TrackCareerXP()` call sites across `battle_service.go`, `handlers_criminality.go`, `courthouse_service.go`, `black_market_service.go`, and `counterfeit_service.go` are confirmed live and wired. Multiplier values must now be computed at each call site and passed to `computeScaledXP()`.

**Files Modified:** `rival_career_engine.go` (added `computeScaledXP`)

- [ ] **Task 4301**: Wire loyalty/fame multipliers at existing TrackCareerXP call sites — pass scaling params to `computeScaledXP()` from `battle_service.go`, `handlers_criminality.go`, `courthouse_service.go`
- [ ] **Task 4302**: Test scaled XP distribution across rival pairs in live game session
- [ ] **Task 4303**: Add loyalty/fame multiplier tests to `rival_career_test.go`
**Recommendation:** Add loyalty/fame scaling multipliers to all careers that currently use flat XP rates. Gossip already has `checkLoyalty` (×1.05 per lesson) and `checkFame` (×1.20 per fame tier). Other careers need equivalent scaling.

**Priority order:**
1. Fence — add loyalty/fame to black market sell XP (highest revenue impact)
2. Bounty Hunter — add scaling to bounty capture XP (highest engagement Justice career)
3. Tax Auditor — add scaling to tax audit XP
4. Remaining ~15 careers deferred until Phase 3 validation

**Implementation pattern:** Add `XPBase * loyaltyMultiplier * fameMultiplier` computation in career engine, apply universally via shared `computeScaledXP(base uint64, role string)` helper.

**Files to modify:** `rival_career_engine.go` (add scaling helper + unified XP computation), targeted handler files for each career.

**Phase 2 Files Modified:** `career.go`, `rival_career_engine.go`, `rivalry_handlers.go`, `justice_service.go`, `shop_registry.go`, `rival_career_test.go` (new)

**Phase 2 Deliverables:**
- XP triggers: uint64 micro-XP values with deterministic floor rounding — all career-specific calculations complete
- Mechanic hooks: shop_registry gates via `RegisterShopItem()` with career-level thresholds
- Rival engine: `evaluateCrossCareerRivalries()` dispatcher routes after every XP trigger
- Builder pattern: `NewCareerXP().WithCategory().WithBaseRate().Build()`, `NewRivalEngine().WithScoringMatrix().WithThresholds().WithXPConfig().Build()`
- Tests: 12 test cases in `rival_career_test.go` — XP calculations, rival mechanics, tier unlocks

---

### ⚡ MEDIUM PRIORITY: Feature Expansion (Pillar 7+)

#### P2-A: Justice Hegemony Path — REMAINING WORK (Post-Phase 2)
**Status as of 2026-06-22:** Core service + shop items + HTTP handlers wired. Career XP triggers now complete for P2-D1 through P2-D6. Remaining: combat integration hooks, frontend dashboard UI, SCSS styles, WebSocket events.

| Task | Component | Status | Notes |
|------|-----------|--------|-------|
| 4002 | Frontend Justice Dashboard JS | ❌ Not started | `Public/js/justice_dashboard.js` — bounty board view, Truth Serum overlay, shield status |
| 4004 | Combat power bonus hooks in `battle_service.go` | ❌ Not started | Call `GetPowerBonus()` during damage calc vs Wanted≥15 targets |
| 4005 | Justice career paths definition | ❌ Not started | Bounty Hunter, Intel-Agent, AOS, Justice Recruiter, Warden, Mutation Auditor, Tax Auditor, Sector Peacekeeper |
| 4006 | Justice API handlers (NEW file) | ❌ Not started | `POST /api/justice/award-card`, `POST /api/justice/use-truth-serum`, `POST /api/justice/use-rep-shield`, `GET /api/justice/alignment/{id}` |
| 4007 | SCSS Justice styles | ❌ Not started | `Public/src/scss/features/_justice.scss` — card archetype visuals, bounty dashboard neon-glass, Truth Serum animation |
| 4008 | Power bonus combat integration | ❌ Not started | `CalculateJusticePowerMultiplier()` in battle resolution |
| 4009 | Wire `NewJusticeService()` into server init | ✅ COMPLETE | Added `justiceService *JusticeService` field to `Lobby` struct in `backend_types.go`; instantiated as `&JusticeService{}` in `newLobby()` in `server.go` (2026-06-18) |
| 4010 | WebSocket justice event types in client | ❌ Not started | `justice_card_awarded`, `truth_serum_applied`, `shield_active`, `dashboard_refresh` |

#### P2-B: Underworld Dominance Path — CONCRETE IMPLEMENTATION TASKS
**Status as of 2026-06-19:** Core types exist in `justice_service.go` + `rival_career_engine.go`. Need handler wiring, shop_registry migration, and UI.

##### P2-B1: Underworld Boss PvE System
| Task | Component | Scope | Dependencies |
|------|-----------|-------|--------------|
| 4101-1A | Define boss templates in `battle_service.go` | HP, attacks, drops (items + XP) | None — pure data def |
| 4101-1B | `HandleUnderworldBoss()` handler | POST `/api/battle/underworld-boss` | Needs `boss_templates` map in `battle_service.go` |
| 4101-1C | Boss difficulty scaling (tier-gated) | Tier 5+ required; harder bosses = more XP | `$VBV gate` (Tier 5+) |
| 4101-1D | Boss drop table (`items_service.go`) | Stolen cards, rare faceplates, micro-units | `item_service.go:DropLoot()` |
| 4101-1E | UI overlay: Boss encounter modal | Neon-glass panel, HP bar, attack buttons | SCSS `_boss.scss` + JS `boss_encounter.js` |

##### P2-B2: Arc-Net-Spy / Data Scramble Intelligence Items
| Task | Component | Notes |
|------|-----------|-------|
| 4102-2A | Add item types to `item_service.go` | `ArcNetSpyItem`, `DataScrambleItem` — scan sectors, encrypt signals |
| 4102-2B | Wire buy endpoint in `rivalry_handlers.go` → migrate to `shop_registry.go` | Current shop items are inline; move to registry for consistency |
| 4102-2C | Implement Arc-Net-Spy logic: reveal hidden NPCs/cards in sector for 60s | Uses `player_service.go:GetSectorCards()` + filter by `ArcNetSpyActive` buff |
| 4102-2D | Implement Data Scramble logic: -3 wanted level, -50% detection chance for 30min | Modifies `wantedLevel` and `detectionMultiplier` in `player.Stats.Buffs` |
| 4102-2E | Timed expiration: both items need 30m/60m timers → `buff_expiry` checks in tick loop | Reuse existing buff expiry pattern from `justice_service.go` |

##### P2-B3: Black Market Expansion (Fenced Goods Marketplace)
| Task | Component | Notes |
|------|-----------|-------|
| 4104-3A | New endpoint: `POST /api/black-market/fence-goods` — list stolen items for sale | Uses `black_market_service.go`, adds `listedAt` + `sellerWallet` fields |
| 4104-3B | Buyer flow: `POST /api/black-market/buy-stolen` — transfers ownership, deducts price | Needs escrow until trade completes; refund on cancel |
| 4104-3C | Commission fee: 8% (vs 10% standard) — Fence career gets reduced rate at Tier 3 → 5% | Reuses `GetFenceFeeDiscount()` from Task 4201-2B |
| 4104-3D | Black Market listing expiry: listings auto-remove after 24h, unsold items returned | Timer stored in `listedAt`; cleanup via daily `processBlackMarketExpiry()` |
| 4104-3E | UI: Fenced goods panel — scroll of available items, buy/sell buttons | New tab in existing Black Market overlay (HTML/SCSS) |

##### P2-B4: Underworld Contracts System (Criminal Missions)
| Task | Component | Notes |
|------|-----------|-------|
| 4105-4A | Contract template engine in `narrative_service.go` | 6 contract types: Heist, Smuggle, Fence, Intimidate, Launder, Protect |
| 4105-4B | `POST /api/contracts/accept` — assigns contract to player with XP + reward targets | Contracts expire after 48h; partial completion = half XP |
| 4105-4C | Contract completion triggers: e.g., "Fence" → `black_market_service.go` tracks `contractsCompleted["Fence"]++` | Check on each action if threshold met (e.g., 3 fences = complete) |
| 4105-4D | Reward: lump sum μVBV + rare item roll + career XP bonus | Same payout as heist rewards — reuse `payoutPlayer()` from `club_service.go` |
| 4105-4E | UI: Contract board overlay — list of active/available contracts, accept/decline buttons | Reuse existing modal pattern in `Public/js/ui.js` |

##### P2-B Priority Ordering (Revenue Impact):
1. 🔥 **Fenced Goods Marketplace (P2-B3)** — Direct revenue stream; Fence career synergy
2. ⚡ **Arc-Net/Spy + Data Scramble (P2-B2)** — Shop item sales drive μVBV burn
3. ⚡ **Underworld Contracts (P2-B4)** — Retention mechanic; daily active users ↑
4. 📋 **Underworld Boss PvE (P2-B1)** — High XP/low revenue; prestige-driven (unlock at Tier 5+)

#### P2-C: Deep Career Trajectories — **REVISION: $VBV-SUSTAINED PROGRESSION MODEL**
**Status as of 2026-06-18:** `rival_career_engine.go` has full career XP engine (698 lines): `CareerXP`, `TrackCareerXP()`, `GetCareerProgress()`, tier gates, role multipliers, XP thresholds. **NO gameplay handler calls these.** Shop items in `rivalry_handlers.go` are inline (not in shop_registry).

**PILLAR 13: $VBV-SUSTAINED CAREER PROGRESSION — IMPLEMENTATION PHILOSOPHY SHIFT:**
Career tier advancement MUST now be gated by **sustained average $VBV balance**, not raw action-XP accumulation. This prevents XP-farming and ensures only economically engaged players reach elite tiers.

**Core Mechanic Design (see `common_types.go` lines 429-434):**
```go
// In PlayerStats:
LiquiditySamples    []uint64        // Recent balance snapshots (micro-$VBV)
AvgSustainedMicro   uint64          // Computed from samples (micro-$VBV)
DemotionWarningAt   time.Time       // When demotion warning was issued (0 = none)
```

**Implementation Pattern:**
1. **XP Trigger** — Still fires on actions (heist, rumor, etc.) but serves as a **bonus multiplier**, not the primary gate
2. **Sustained Balance Gate** — Career tier unlocks require `AvgSustainedMicro >= tier_threshold_micro` for minimum duration
3. **Demotion Window** — If balance drops below threshold → 7-day grace period (warning issued), then demote
4. **Sampling** — Sample player balance every 24h during active play, stored in `LiquiditySamples` (keep last 14 samples = 14-day window)
5. **Computation** — `AvgSustainedMicro = sum(LiquiditySamples) / len(LiquiditySamples)` converted to $VBV

**Wiring Priority Order:** Underworld first (revenue), then Justice (control). Each career gets: liquidity gate → action bonus → mechanic hook → shop validation.

##### $VBV-SUSTAINED TIER THRESHOLDS (All Careers — New Standard)
**Career tier gates are now $VBV average-based, not XP-based.** Action-XP is a bonus multiplier only.

| Career Tier | Avg Sustained $VBV (micro-units) | Min Samples Required | Demotion Trigger |
|-------------|-----------------------------------|---------------------|------------------|
| Peon (Tier 0) | 0 ($VBV ≥ 0) | 0 | N/A — base tier |
| Apprentice (Tier 1) | ≥ 5,000 VBV (5B μVBV) | 7 samples over 7 days | Drop below 3K for 14 days → demote |
| Journeyman (Tier 2) | ≥ 25,000 VBV (25B μVBV) | 10 samples over 10 days | Drop below 15K for 14 days → warning issued |
| Expert (Tier 3) | ≥ 100,000 VBV (100B μVBV) | 12 samples over 12 days | Drop below 60K for 7 days → demote |
| Master (Tier 4) | ≥ 500,000 VBV (500B μVBV) | 14 samples over 14 days | Drop below 300K for 7 days → demote |
| Boss (Tier 5) | ≥ 2,000,000 VBV (2T μVBV) | 14 samples over 14 days | Drop below 1M for 7 days → demote |

**Key formulas:**
- `LiquiditySample = player.VBVBalance * 1_000_000` (micro-conversion at sample time)
- `AvgSustainedMicro = sum(LiquiditySamples) / len(LiquiditySamples)` 
- Demotion grace period: 7 days from first warning at `DemotionWarningAt` timestamp
- Samples kept: last 14 (rolling window); new sample overwrites oldest each active play day

**$VBV-gated career access rules:**
- Peon: no gate — all players start here
- Apprentice: sustained ≥ 5K $VBV for 7 days minimum
- Journeyman: sustained ≥ 25K $VBV for 10 days + Junior role (Gossip/Fence/Kidnapper/etc.) unlocked via bonus XP
- Expert: sustained ≥ 100K $VBV for 12 days + Action XP threshold per role (unchanged from `rival_career_engine.go` tier gates)
- Master: sustained ≥ 500K $VBV for 14 days + Expert role + faction shop access
- Boss: sustained ≥ 2M $VBV for 14 days + all prior tiers + PvP leaderboard top-100

**Action-XP bonus multiplier table (stacks with $VBV gate):**
| Avg Sustained $VBV Tier | XP Multiplier on Actions | Shop Discount | Special Access |
|--------------------------|-------------------------|---------------|----------------|
| Apprentice | ×1.0 (baseline) | None | — |
| Journeyman | ×1.25 | 5% faction shop discount | Junior mechanic hooks |
| Expert | ×1.5 | 10% faction shop discount | Elite mechanic hooks |
| Master | ×2.0 | 15% faction shop discount | Boss-tier PvE, Arc-Net full vision |
| Boss | ×3.0 | 25% faction shop discount | Underworld boss access, Justice commissioner tier |

---

##### UNDERWORLD CAREERS — Wiring Priority Order (Now $VBV-Gated)

**Underworld Career #1: Gossip (Rumor Manipulator) — COMPLETED ✅ 2026-06-18**
- [x] **Task 4201-1A**: Wire `l.TrackCareerXP(wallet, "Gossip", XP_AMT)` in `handlers_rumor.go` after Rumor Mill manipulation (+50 XP per use) — **COMPLETE**
- [x] **Task 4201-1B**: Implement `GetRumorFeeDiscount()` — checks `player.CareerXP.Tiers["Gossip"] >= 3` → returns `0.80` (20% discount). Replace hardcoded fee multiplier in `HandlePostRumor` — **COMPLETE**
- [x] **Task 4201-1C**: Add "Rumor Discount Active" buff tag in player stats when Gossip ≥ Tier 3 — **COMPLETE**
- [x] **Task 4201-1D**: Validate: rumor fee = (500 * 0.80 = 400 VBV) instead of full 500 VBV — **COMPLETE**

**Underworld Career #2: Fence (Black Market Liquidator)** — **WIP ✅ 2026-06-18**
- [ ] **Task 4201-2A**: Wire `l.TrackCareerXP(wallet, "Fence", XP_AMT)` in `black_market_service.go` after sell (+30 XP per sale)
- [x] **Task 4201-2B**: Implemented `GetFenceFeeDiscount()` — checks `player.CareerXP.RoleXP["Fence"] >= Tier3 (1500)` → returns `0.50` (50% discount). Wired into `black_market_service.go:HandleSellToBlackMarket()` at line ~139 — **COMPLETE**
- [x] **Task 4201-2C**: "Fenced Rate Active" buff wired in `black_market_service.go` — sets `stats.Buffs["fenced_rate_active"] = true` when discount applied — **COMPLETE**
- [ ] **Task 4201-2D**: Validate: `player_black_market_fee` = (lootMicro * 5/100) instead of default (lootMicro * 10/100) — mechanics correct, blocked until Task 2A XP trigger wired

**Underworld Career #3: Kidnapper (Hostage Specialist)** — ✅ PHASE 2 COMPLETE
- [x] **Task 4201-3A**: Hook `l.TrackCareerXP(wallet, "Kidnapper", XP_AMT)` in `battle_service.go` resolveBattle → add `TrackCareerXP("Kidnapper")` +80 per hostage captured via battle capture path
- [x] **Task 4201-3B**: Implement `GetKidnapSuccessMultiplier()` — Tier≥3 returns `2.0`; call from `battle_service.go` in capture logic before heist resolution
- [x] **Task 4201-3C**: "Double Hostage Chance" buff tag wired into club_service.go heist branch when Kidnapper tier≥3 active
- [x] **Task 4201-3D**: Validation gate in rival_career_engine.go: multiplier applies only if attacker has `Kidnapper` role active

**Underworld Career #4: Hostage Host (Collective Hoarder)**
- [ ] **Task 4201-4A**: Wire XP after each card held past 6h (+40 XP)
- [ ] **Task 4201-4B**: Implement `GetSignalDampenerStacking()` — ally dampener detection resistance stacks (max 4 members = 4×)
- [ ] **Task 4201-4C/D**: "Collective Stealth" tag + validation

**Underworld Career #5: Lawyer-Commissioner (Underworld Administrator)**
- [ ] **Task 4201-5A**: Wire XP after processing tax evasion services (+25 XP)
- [ ] **Task 4201-5B**: Implement `GetCommissionOverride()` — custom commission rates ±10% in `shop_registry.go`

**Underworld Career #6: Underworld Boss (Elite Leader)**
- [ ] **Task 4201-6A**: Wire XP after NPC boss defeat (+500 XP)
- [ ] **Task 4201-6B**: Implement `GetBossAccess()` — unlocks exclusive PvE bosses in `battle_service.go`

**Underworld Career #7: Arc-Net Operative (Digital Specialist)**
- [ ] **Task 4201-7A**: Wire XP after Cyber-Lock/Cyber-Jammer deploy (+20 XP)
- [ ] **Task 4201-7B**: Implement `GetCyberLockDuration()` — returns `baseDuration × 1.5`

**Underworld Career #8: Smuggler (Cross-Sector Transporter)**
- [ ] **Task 4201-8A**: Wire XP after cross-sector transfer (+35 XP)
- [ ] **Task 4201-8B**: Implement `GetTransitTaxExemption()` — bypasses regional transit taxes

**Underworld Career #9: Heist Planner (Tactical Mastermind) — CORRECTED STATUS ✅ 2026-06-18**
- [ ] **Task 4201-9A**: Wire XP after heists where planner's org involved (+60 XP) — *NO code exists yet*
- [ ] **Task 4201-9B**: Implement `GetPlanningBuff()` from scratch — +5% success rate for team heists (function does NOT exist; previous "partial mechanic" note was incorrect)
- [ ] **Task 4201-9C/D**: Planner dividend tracking + validation — *needs definition*

**Underworld Career #10: Launderer (Financial Cleaner)**
- [ ] **Task 4201-10A**: Wire XP after processing ransom/stolen capital (+45 XP)
- [ ] **Task 4201-10B**: Implement `GetWantedLevelReduction()` — returns `min(3, wantedGain)`

##### P2-D: Justice Careers — CONCRETE IMPLEMENTATION TASKS

**Wiring Priority Order:** Bounty Hunter → Tax Auditor → Warden → AOS → Forensic Analyst → Sector Peacekeeper → Intel-Agent → Justice Recruiter → Commissioner → Mutation Log Auditor. Each career requires: XP trigger in existing handler → mechanic function → shop item → UI indicator.

###### P2-D1: Bounty Hunter (Primary Track — Highest Revenue Impact)
| Task | Component | Scope | Dependencies |
|------|-----------|-------|--------------|
| 4301-1A | Wire XP trigger in `handlers_criminality.go` after bounty capture | +80 XP per successful capture via `TrackSoloCapture()` | Uses existing `bounty_service.go:HandleBountyCapture()` — add `l.TrackCareerXP(wallet, "BountyHunter", 80)` at line ~142 |
| 4301-1B | Implement `GetBountyTrackingBonus(tier int) float64` | Tier≥3 → returns `1.15` (+15% tracking speed); Tier≥5 → `1.25` | Reuse bonus multiplier pattern from `rival_career_engine.go:TrackRivalInteraction()` |
| 4301-1C | Bounty Hunter shop items in `shop_registry.go` | bounty_license (recurring 50K μVBV), tracker_drone (500K μVBV, 2h), warrant_stamper (1M μVBV) | Move from `rivalry_handlers.go` inline definitions to registry |
| 4301-1D | UI: Active bounty tracker overlay — show nearby bounties with Bounty Hunter buff bonus | Neon-glass panel highlighting high-value targets | JS `bounty_tracker.js` + SCSS `_bounty.scss` |

###### P2-D2: Tax Auditor (Financial Control Path)
| Task | Component | Scope | Dependencies |
|------|-----------|-------|--------------|
| 4302-2A | Wire XP trigger in `economy_service.go` after tax collection events | +30 XP per tax transaction audited; bonus +50 for flagged violations | Hook into `ProcessTax()` at line ~88 — add `l.TrackCareerXP(wallet, "TaxAuditor", 30)` |
| 4302-2B | Implement `GetAuditPrecisionBonus(tier int) float64` | Reveals hidden revenue (Fence/Smuggler income); Tier≥3 → reveal +50% of laundered amount | Needs access to `player.Buffs["hiddenRevenue"]` from black_market_service.go |
| 4302-2C | Tax Auditor shop items: audit_warrant (800K μVBV, reveals target's revenue), compliance_notice (300K μVBV, freezes assets 1h) | Revenue-generating items (players buy to audit rivals) | Add to `shop_registry.go` under JUSTICE faction |
| 4302-2D | UI: Audit log — shows revealed hidden revenue, frozen assets | Reuse existing transaction log overlay pattern in `ui.js` | HTML/SCSS via `_audit.scss` |

###### P2-D3: Warden (High-Tier Control Path)
| Task | Component | Scope | Dependencies |
|------|-----------|-------|--------------|
| 4303-3A | Wire XP trigger after mission completions in `justice_service.go` | +100 XP per COMPLETED justice mission; +50 for EXPIRED (partial credit) | Hook into `GenerateJusticeMission()` — add XP on status transition to COMPLETED |
| 4303-3B | Implement `GetWardenDetentionBonus(tier int) float64` | Tier≥2 → detention duration ×1.5; Tier≥4 → ×2.0 (longer incarceration = more revenue) | Reuse buff stacking pattern from `justice_service.go:ApplyTruthSerum()` |
| 4303-3C | Warden-exclusive justice cards upgrade: ENFORCER+ tier gets +20% base power instead of +10% | Uses existing `GetPowerBonus()` in `justice_service.go` — modify return value for Warden rank≥4 | Only modifies `justice_card_pool` tier thresholds |
| 4303-3D | UI: Detention timer overlay — shows remaining incarceration time for captured targets | WebSocket event `warden_detention_update` pushes countdown to client | JS `warden_timer.js` + WS handler in `network.go` |

###### P2-D4: AOS (Armed Offender Squad — Team Capture Path)
| Task | Component | Scope | Dependencies |
|------|-----------|-------|--------------|
| 4304-4A | Wire XP trigger in `battle_service.go` after team heist/capture completion | +60 XP per team capture via `TrackTeamCapture()` — requires ≥2 org members present | Hook into `resolveBattle()` at line ~215 — add `l.TrackCareerXP(wallet, "AOS", 60)` when team_capture = true |
| 4304-4B | Implement `GetAOSCoordinationBonus(tier int) float64` | Tier≥3 → org members get +10% power buff during joint ops; tracks via `org.Buffs["aos_coordination"]` | Reuse org buff system from `club_service.go:ProcessAllianceBuff()` |
| 4304-4C | AOS shop items: tactical_vest (1.2M μVBV, +15% damage resistance for 1h), comms_array (700K μVBV, org-wide detection reveal 30min) | Items purchased per-org; all members benefit | Add to `shop_registry.go` under JUSTICE faction with org-ownership validation |
| 4304-4D | UI: Org coordination status — show active AOS buffs for each member | Reuse social hub overlay pattern in existing JS | HTML/SCSS via `_aos.scss` |

###### P2-D5: Forensic Analyst (Evidence Path)
| Task | Component | Scope |_dependencies |
|------|-----------|-------|--------------|
| 4305-5A | Wire XP trigger in `black_market_service.go` after evidence collection on target | +40 XP per stolen card/asset recovered from black market raids | Hook into `raidenBlackMarket()` — add `l.TrackCareerXP(wallet, "ForensicAnalyst", 40)` |
| 4305-5B | Implement `GetEvidenceAccuracyBonus(tier int) float64` | Tier≥3 → cleans Gossip-altered records at 2× effectiveness (reverses rumor manipulation) | Cross-references `rumor_service.go:GetAlteredRecords()` and applies reversal |
| 4305-5C | Forensic Analyst shop items: evidence_kit (600K μVBV, reveals target's hidden buffs), chain_of_custody (900K μVBV, preserves evidence for 24h) | Evidence-based items — reveal/rival intelligence | Add to `shop_registry.go` under JUSTICE faction |
| 4305-5D | UI: Evidence board — visual chain of discovered evidence links between targets | New overlay (replaces skeleton note "Orphaned") | JS `evidence_board.js` + SCSS `_evidence.scss` |

###### P2-D6–P2-D10: Remaining Justice Careers (Quick-Def)
| Career | XP Trigger | Mechanic | Shop Items | Notes |
|--------|-----------|----------|------------|-------|
| **Sector Peacekeeper** | +50 XP per sector patrol (wire in `territory_service.go` after sector check) | `GetPatrolBlockBonus()` — blocks Smuggler routes for 10min (checks `TransitTaxExemption`) | patrol_stake (400K μVBV), route_blocker (600K μVBV) | Rival pair vs Smuggler (Task 4201-21E) |
| **Intel-Agent** | +35 XP per Cyber-Lock intercepted (wire in `battle_service.go` after cyber counter) | `GetDecryptBonus()` — decrypts Arc-Net-Spy visibility for 60s | signal_interceptor (1.5M μVBV), decrypt_key (800K μVBV) | Rival pair vs Arc-Net Operative (Task 4201-21F) |
| **Justice Recruiter** | +20 XP per new justice-aligned player onboarding | `GetRecruitmentBonus()` — recruits get +5% starting power | recruitment_bounty (100K μVBV, recurring payout) | Low priority; cosmetic revenue only |
| **Justice Commissioner** | +200 XP per justice mission completed at Tier 5+ | `GetCommissionOverride()` — sets custom tax rates ±15% in shop_registry | commissioner_warrant (5M μVBV, permanent) | Highest tier justice career; $VBV gate ≥500K |
| **Mutation Log Auditor** | +45 XP per foundry transaction audited (wire in `item_service.go` after mutation) | `GetAuditMutation()` — reveals mutation vector history for counter-synthesis | mutation_scan (2M μVBV, 30min vision), log_seal (1.5M μVBV) | Lowest priority; niche mechanic |

###### P2-D Priority Reordering (Revenue + Control Impact):
1. 🔥 **Bounty Hunter (P2-D1)** — Direct revenue (shop items); highest player engagement
2. ⚡ **Tax Auditor (P2-D2)** — Revenue-generating audits; counters Underworld economy
3. ⚡ **Warden (P2-D3)** — High-tier control; mission system driver
4. 📋 **AOS → Forensic → Sector Peacekeeper → Intel-Agent** — Group 4-7 in priority order
5. 📋 **Justice Recruiter → Commissioner → Mutation Auditor** — Lowest revenue, prestige-only


##### P2-E: Rival Mechanics (Cross-Career Pair Interactions) — NOW WITH CONCRETE WIRING

**Status:** `TrackRivalInteraction()` skeleton in `rival_career_engine.go` has all types + XP delta computation. Zero handlers call it. Below: concrete task breakdown for each rival/ally pair, handler locations, and client-side effects.

###### P2-E1: Enemy Rival Pairs (Negative Interaction — Detriments)
| Task | Pair | Mechanic Implementation | Handler Location | Client Effect |
|------|------|------------------------|------------------|---------------|
| 4201-21A-1 | Bounty Hunter ↔ Kidnapper | On capture: `if target.CareerXP.RoleXP["Kidnapper"] > 0 { trackingBonus = GetBountyTrackingBonus(tier) }` — return `1.15` at tier≥3, `1.25` at tier≥5 | `handlers_criminality.go:HandleBountyCapture()` add rival check before finalizing XP | Show "Enhanced Tracking Active" buff icon in bounty tracker UI |
| 4201-21A-2 | Wire `TrackRivalInteraction(bountyWallet, kidnapperWallet, "BountyHunter", "Kidnapper", -10)` in bounty capture flow | Uses existing `RivalInteractionPair{Attacker, Defender, RivalryXP int}` struct; updates both players' `RivalXP` maps in `player.CareerXP.Rivals` | Same handler — insert before payout | WebSocket `rivalry_triggered` broadcast to both players |
| 4201-21B-1 | Forensic Analyst ↔ Gossip | On forensic raid: `if target.CareerXP.RoleXP["Gossip"] > 0 { evidenceMult = GetEvidenceAccuracyBonus(tier) }` — tier≥3 returns `2.0` (double-clean) | `black_market_service.go:HandleBlackMarketRaid()` add rival check | "Records Cleansed" flash notification on forensic dashboard |
| 4201-21B-2 | Wire rivalry XP delta in same raid handler | `TrackRivalInteraction(forensicWallet, gossipWallet, "ForensicAnalyst", "Gossip", -10)`; Gossip player gets +5XP (grudging respect) | Same as 4201-21B-1 | Gossip player sees "Professional Respect (+5)" toast |
| 4201-21C-1 | Tax Auditor ↔ Launderer | On audit: `if target.CareerXP.RoleXP["Launderer"] > 0 { revealedAmount = hiddenRevenue * 1.5 }` — tier≥3 reveals +50% laundered amount | `economy_service.go:ProcessTax()` add rival check on revenue source | Audit log shows "Laundering Obfuscated — Revealed via Auditor Bonus" |
| 4201-21C-2 | Wire rivalry in tax handler | `TrackRivalInteraction(auditorWallet, laundererWallet, "TaxAuditor", "Launderer", -10)` | Same as 4201-21C-1 | Launderer player gets "Audit Hit" notification with revealed amount |
| 4201-21D-1 | Warden ↔ Heist Planner | On warden mission completion: `if target.CareerXP.RoleXP["HeistPlanner"] > 0 { detentionBonus = GetWardenDetentionBonus(tier) }` — anticipates strike (double detention) | `justice_service.go:CompleteJusticeMission()` add rival check | Warden UI shows "Strike Pattern Anticipated" badge on mission panel |
| 4201-21D-2 | Wire rivalry in mission handler | `TrackRivalInteraction(wardenWallet, plannerWallet, "Warden", "HeistPlanner", -10)` | Same as 4201-21D-1 | Heist Planner gets "Strike Pattern Discovered" warning toast |
| 4201-21E-1 | Sector Peacekeeper ↔ Smuggler | On patrol: `if sector has active smuggler transit { blocked = true; GetPatrolBlockBonus() returns timeBlocked }` | `territory_service.go` (needs creation or wire into existing sector check) — patrol loop checks `TransitTaxExemption` flag | Smuggler gets "Route Blocked" notification; Peacekeeper UI shows route map |
| 4201-21E-2 | Wire rivalry in patrol handler | `TrackRivalInteraction(peacekeeperWallet, smugglerWallet, "SectorPeacekeeper", "Smuggler", -10)` | Same as 4201-21E-1 | Both players see route block visualization on maps |
| 4201-21F-1 | Intel-Agent ↔ Arc-Net Operative | On cyber-intercept: `if target has ArcNetSpyActive buff { GetDecryptBonus() returns visibilityOverride }` — decrypts sector vision for 60s | `battle_service.go` after cyber counter resolve — check ArcNetSpyActive buff on opponent | Intel-Agent UI shows decrypted sector map; Arc-Net Operative gets "Cover Burned" warning |
| 4201-21F-2 | Wire rivalry in cyber handler | `TrackRivalInteraction(intelWallet, arcnetWallet, "IntelAgent", "ArcNetOperative", -10)` | Same as 4201-21F-1 | Both players get cyber-intercept event toast |

###### P2-E2: Ally Rival Pairs (Positive Interaction — Bonuses)
| Task | Pair | Mechanic Implementation | Handler Location | Client Effect |
|------|------|------------------------|------------------|---------------|
| 4201-21G-1 | Justice Recruiter ↔ Bounty Hunter | On recruit: `GetRecruitmentBonus() — new justice player starts with +5% power`; bounty hunter gets +10XP per recruited justice career | `onboarding_service.go` after new justice career onboarding | New justice player sees "Mentored by Bounty Network" buff; recruiter gets XP toast |
| 4201-21G-2 | Wire ally XP in onboarding handler | `TrackRivalInteraction(recruiterWallet, recruitWallet, "JusticeRecruiter", "BountyHunter", +8)` — reciprocal positive delta | Same as 4201-21G-1 | Both players see alliance bond notification |
| 4201-21H-1 | Launderer ↔ Fence | On fence transaction: `GetFenceFeeDiscount() also reduces launderer processing time by 20%` — synergistic money flow | `black_market_service.go` after sell completes + `economy_service.go` on laundering | Both UIs show "Synergy Active" buff icons; reduced timers |
| 4201-21H-2 | Wire ally XP in black market handler | `TrackRivalInteraction(fenceWallet, laundererWallet, "Fence", "Launderer", +5)` after each linked transaction | Same as 4201-21H-1 | Transaction summary shows allied bonus breakdown |
| 4201-21I-1 | Heist Planner ↔ Kidnapper | On heist with hostage: `GetPlanningBuff() — team heist success ×1.05; kidnapper gets +25% capture chance` | `battle_service.go:resolveBattle()` team capture branch — add planner+kidnapper check | Heist UI shows "Mastermind + Hostage Specialist Synergy" bonus line |
| 4201-21I-2 | Wire ally XP in battle handler | `TrackRivalInteraction(plannerWallet, kidnapperWallet, "HeistPlanner", "Kidnapper", +12)` on successful heist | Same as 4201-21I-1 | Both players get team-bonus XP toast |
| 4201-21J-1 | AOS ↔ Sector Peacekeeper | On org patrol: `GetAOSCoordinationBonus() — org members in same sector get +8% detection`; peacekeeper gets +10% block duration | `club_service.go:ProcessAllianceBuff()` extend to include sector patrol bonus | Org UI shows coordinated patrol map; peacekeeper UI shows allied presence |
| 4201-21J-2 | Wire ally XP in org buff handler | `TrackRivalInteraction(aosWallet, peacekeeperWallet, "AOS", "SectorPeacekeeper", +6)` per coordinated patrol | Same as 4201-21J-1 | Org members see coordination bonus on dashboard |
| 4201-21K-1 | Tax Auditor ↔ Justice Commissioner | On commissioner override: `GetCommissionOverride() — auditor gets +20% revealed revenue share`; commissioner gets audit efficiency +15% | `shop_registry.go` custom commission logic + `economy_service.go` tax processing | Both UIs show "Fiscal Alliance" buff; enhanced revenue numbers |
| 4201-21K-2 | Wire ally XP on commission event | `TrackRivalInteraction(taxWallet, commWallet, "TaxAuditor", "JusticeCommissioner", +7)` | Same as 4201-21K-1 | Revenue report shows allied bonus breakdown |
| 4201-21L-1 | Gossip ↔ Forensic Analyst (Ally variant) | When both on same team/org: `GetRumorDiscount() also gives forensic +10% evidence accuracy` — gossip amplifies forensic reach | `justice_service.go` or `club_service.go` org buff calc | Org UI shows "Intelligence Network" synergy buff icon |
| 4201-21L-2 | Wire ally XP in org calculation | `TrackRivalInteraction(gossipWallet, forensicWallet, "Gossip", "ForensicAnalyst", +5)` per org tick | Same as 4201-21L-1 | Both players see synergy notification on org dashboard |

###### P2-E3: Rival XP Engine Wiring (Core Infrastructure)
| Task | Component | Scope | Notes |
|------|-----------|-------|-------|
| 4201-21X-1 | `TrackRivalInteraction()` caller injection across all handlers | Add rival check after every cross-player interaction action | Pattern: `if rivalPair, exists := RivalPairs[attackerCareer][defenderCareer]; exists { l.TrackRivalInteraction(attackerWallet, defenderWallet, ...) }` |
| 4201-21X-2 | Rival threshold trigger at -50/+50 rivalry points | At -50: "Bitter Enemy" debuff (×1.2 power penalty); at +50: "Trusted Ally" buff (×1.1 power boost) | Check in `GetCareerProgress()` — add `RivalStatus` field to response |
| 4201-21X-3 | Rivalry decay: -1 point per 24h of no interaction | Prevents permanent state; uses existing timer pattern from `buff_expiry` | Run in daily tick loop (already exists for economy decay) |
| 4201-21X-4 | WebSocket event broadcasting for all rivalry triggers | `rivalry_triggered`, `rivalry_threshold_reached`, `ally_buff_activated` | Add to `network.go` WS message types; reuse existing broadcast pattern |

###### P2-E Priority Ordering (Gameplay Impact):
1. 🔥 **Enemy pairs G-F (4201-21A–21F)** — Direct combat/economy impact; highest player engagement
2. ⚡ **Enemy pair wiring infrastructure (4201-21X)** — Core TrackRivalInteraction callers
3. ⚡ **Ally pairs G-L (4201-21G–21L)** — Retention via alliance mechanics
4. 📋 **Rivalry thresholds + decay (4201-21X-2/3)** — Polish; adds long-term rivalry state
5. 📋 **WS events (4201-21X-4)** — UX polish; clients need events to show rivalry UI

##### UI & FRONTEND (Career Dashboard)
- [ ] **Task 4201-22A**: Create `Public/js/career_dashboard.js` — career XP progress bars, tier display, rivalry log
- [ ] **Task 4201-22B**: Create `Public/src/scss/features/_careers.scss` — neon-glass career cards, XP bar animations, tier badges
- [ ] **Task 4201-22C**: Wire API: `GET /api/career/progress/{wallet}`, `GET /api/career/rivalries/{wallet}`, `POST /api/career/select-role`
- [ ] **Task 4201-22D**: Integrate career dashboard into existing UI overlay (tab/modal in Social Hub)
- [ ] **Task 4201-22E**: WebSocket events: `career_xp_update`, `career_tier_up`, `rivalry_triggered`

##### BACKEND CONFIRMATION & TESTING
- [ ] **Task 4201-23A**: Create `career_service_test.go` — unit tests for each career mechanic
- [ ] **Task 4201-23B**: Integration test: 100 heists with Kidnapper → verify XP + tier advancement
- [ ] **Task 4201-23C**: Regression test: all existing economic loops pass (no token creation/destruction)
- [ ] **Task 4201-23D**: Add `l.logAdminAuditLocked("CAREER_XP_GAINED", wallet, ...)` for forensic audit trail

#### P2-C Priority Reordering (Revenue Impact Analysis):
1. 🔥 **Gossip** — ✅ COMPLETE (XP trigger + fee discount + buff tag wired)
2. ⚡ **Heist Planner** — Mechanism from scratch (GetPlanningBuff does NOT exist; status corrected)
3. ⚡ **Launderer** — Wanted reduction prevents runaway wanted scaling
4. ⚡ **Fence** — Black Market fee optimization (50% discount at Tier 3)
5. 📋 All remaining careers deferred until core loop validated

---

### 🛠 MAINTENANCE & OPTIMIZATION

#### P3-A: Performance Hardening
- [ ] **Task 5001**: Profile `processMojoDecay` for high-club-count scenarios (>50 active clubs)
- [ ] **Task 5002**: Optimize blockchain snapshot serialization for >1000-player lobby
- [ ] **Task 5003**: Implement WebSocket message compression for large state updates
- [ ] **Task 5004**: Add connection pooling optimizations for indexer queries

#### P3-B: Data Integrity & Monitoring
- [ ] **Task 5101**: Implement automated solvency monitoring alerts
- [ ] **Task 5102**: Create Grafana dashboard templates for Telemetry metrics
- [ ] **Task 5103**: Add economy drift detection (virtual vs physical balance divergence)
- [ ] **Task 5104**: Implement automatic backup verification system

#### P3-C: Documentation & Developer Experience
- [ ] **Task 5201**: Update API documentation with all service layer endpoints
- [ ] **Task 5202**: Create architecture decision records (ADRs) for key design decisions
- [ ] **Task 5203**: Implement development server hot-reload configuration
- [ ] **Task 5204**: Add integration test suite for economic loops

---

## 📋 PENDING FEATURE REQUESTS

### Feature: Seasonal Structure (Game Expansion Plan §12)
**Status: Planned - Requires Design Phase First**
- [ ] **Task 6001**: Define seasonal timeline and reward tiers
- [ ] **Task 6002**: Implement automated season rollover with leaderboard archival
- [ ] **Task 6003**: Create seasonal achievements and cosmetic rewards
- [ ] **Task 6004**: Design "Fresh Start" mechanics for returning players

### Feature: External Spectator Integration (Implemented - Phase 6)
**Status: Partial - Needs UI Polish**
- [ ] **Task 6101**: Complete external spectator portal template (Carrd integration)
- [ ] **Task 6102**: Implement neon-glass VBT Cyber-HUD for third-party overlays
- [ ] **Task 6103**: Add spectator chat/interaction channel

### Feature: Mobile Responsiveness (Game Expansion Plan §8)
**Status: Partial - SCSS hardening done, JS needs attention**
- [ ] **Task 6201**: Audit all UI overlays for mobile viewport compatibility
- [ ] **Task 6202**: Implement touch gesture support for card interactions
- [ ] **Task 6203**: Test and optimize particle effects on mobile GPUs

---

## 🔄 IMPLEMENTATION PROTOCOL

### Standard Implementation Workflow
1. **Design**: Create feature specification in `Development-production-build/`
2. **Implementation**: Code changes following domain separation (service files)
3. **Verification**: Unit tests + simulated stress testing
4. **Documentation**: Update relevant AI-Brain documents
5. **Review**: Cross-document consistency verification

### Critical Constraints (From Rules.md)
- All economic math uses `uint64` micro-units (1 $VBV = 1,000,000 micro-units)
- Deterministic rules must be identical between Go WASM and client JS
- No private key exposure (Switchboard Pattern for server-side signing)
- Blockchain is authoritative; localStorage is a warm-boot beacon only
- All admin actions require WalletConnect/ARC-14 signature verification

---

## 📊 PROJECT METRICS (Current State - Updated 2026-06-18)

| Metric | Value |
|--------|-------|
| Implementation Milestones Complete | 8 (was 7, +Rate Limiting Pillar) |
| Total Tasks Completed (A.I_memory.md) | ~826+ tracked implementations |
| Service Files Created | 19 domain services (+ `justice_service.go`, `rivalry_handlers.go`) |
| SCSS Modular Partials | 12 partials |
| Telemetry Metric Types | GaugeVec, CounterVec, HistogramVec |
| On-Chain State Snapshots | `VBT_STATE_SNAPSHOT`, `VBT_ECONOMY_SNAPSHOT`, `VBT_CARD_CACHE_SNAPSHOT` |
| Active TODO Items | ~51 (across all priorities) |
| **Health Checks / RPC Hardening** | ✅ Task 3005: `SetProductionMode(true)` wired in `newLobby()` (329 max rate limits, 5s sync lag, 15m cooldown). ✅ Task 3006: `/health`, `/live`, `/ready` routes registered in `server.go`. |
| **Rate Limiting & DDoS** | 🆕 NEW — Pillar 1-C defined with 6 tasks (3101–3106). ❌ Not implemented yet. Core blocker for P1-A: rate_limiter.go type, middleware wiring, per-wallet limits, IP fallback, admin bypass, Prometheus counter. |
| **Justice Hegemony** | ✅ Core service + shop items + HTTP handlers wired. ❌ Combat hooks, dashboard UI, SCSS, WS events, server init wiring pending |
| **Rivalry System** | ✅ Backend API + frontend module fully wired. Shop items inline in handlers (not shop_registry). Career XP engine skeleton: ZERO callers |
| **Career XP Engine** | ⚠️ Full engine in `rival_career_engine.go` (698 lines). **Paradigm shift: career tiers now $VBV-sustained-gated, not XP-only.** 20 careers × 10 roles — ZERO gameplay XP triggers wired. Need to add `TrackCareerXP()` calls + `UpdateLiquiditySample()` sampling to every major handler. |
| **Rival Mechanics** | ✅ Concrete wiring plan complete (P2-E section added 2026-06-19): 12 rivalry pairs with handler-level tasks (4201-21A–21L), infrastructure hooks (4201-21X) for `TrackRivalInteraction()` injection, threshold triggers at ±50 XP, daily decay (-1/24h), and WS event broadcast. Zero callers in production yet — spec is locked for implementation. |
| **Gossip Career #1** | ✅ COMPLETE — XP trigger + fee discount + buff tag fully wired (preliminary; needs $VBV gate integration) |
| **Heist Planner #9** | ⚠️ STATUS CORRECTED — GetPlanningBuff does NOT exist; mechanism must be built from scratch |
| **Pillar 13 ($VBV Career Gate)** | 🆕 NEW — 6-tier $VBV-gated career system defined. Requires: liquidity sampling every 24h, demotion grace periods, XP multiplier table wired into all handlers. Implementation starts after Gossip/Fence validation. |

---

## 🗺 STRATEGIC TIMELINE (Recommended — Updated)

```
Phase 1: Production Sealing (Weeks 1-2)
├── P1-A: Secret management wiring
├── P1-B: Rate limiting & DDoS mitigation (tasks 3101–3106)
├── Final security audit (sybil, CORS, replay attacks)
└── Deployment configuration

Phase 1.5: Security Gate (Week 2 end)
├── ✅ All P1-A + P1-B tasks complete
├── ✅ Sybil threshold analysis finalized
├── ✅ CORS policy locked for production domains
└── ✅ Admin handler replay attack audit passed

Phase 2: Justice Hegemony Completion (Weeks 3-6)
├── P2-A: Dashboard UI JS + SCSS
├── P2-A: Combat power bonus hooks in battle_service.go
├── P2-A: Justice API handlers (new file)
├── P2-A: Wire NewJusticeService() into server init
├── P2-A: WebSocket justice event types
├── P2-B: Underworld Dominance base
└── P2-B: Underworld career paths

Phase 3: Deep Career System Pillar 8 (Weeks 7-10)
├── Wire underworld careers 2-4 (Fence → Kidnapper) [Gossip #1 complete]
├── Wire Heist Planner from scratch [mechanism needs definition]
├── Wire justice careers 1-5
├── Implement rivalry pair mechanics (12 pairs)
└── UI/UX for career tracking + rivalries

Phase 4: Seasonal Structure Pillar 9 (Weeks 11-12)
├── Season mechanics design
├── Automated rollover system
└── Seasonal rewards pipeline

Phase 5: Polish & Launch (Weeks 13-14)
├── Mobile responsiveness completion
├── Performance optimization
├── Documentation finalization
└── Public beta launch
```

---

*Last Updated: 2026-06-18*  
*Base Revision: Production-Ready Beta (Post-Roadmap Step 6 + Justice Service v1 + Rivalry API v1)*  
*Next Review Point: Phase prioritization approval for expansion pipeline*  
*Verified Files: `justice_service.go` (587 lines), `rivalry_handlers.go` (619 lines), `server.go` (routing confirmed), `Public/js/rivalry.js` (137 lines), `rival_career_engine.go` (698 lines skeleton)*
