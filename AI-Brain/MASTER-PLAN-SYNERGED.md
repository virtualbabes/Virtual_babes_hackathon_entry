# MASTER PLAN — SYNERGED (Existing + Vision)

> **Generated:** 2026-09-01 by Crypto-Seraph (Lead Systems Architect, YOLO)
> **Purpose:** Synergize existing codebase with Absolute Vision. Build on what exists. Fill what's missing.
> **Principle:** Maximum leverage — every new system strengthens five existing ones.
> **Last updated:** 2026-09-05 — AI civilization backend fully implemented.

---

## Existing Foundations (What We Have)

| System | Lines | What It Does | Leverage Point |
|--------|-------|--------------|----------------|
| Club Service | 3294 | Foundry, territories, regional managers, alliances | Civilization backbone |
| Battle Service | 2320 | Combat, tournaments, ratings, XP | Story generator |
| AI Citizens | 1550 | Spawn, own wallet, behavioral loop (10 dynamics), pathways, BondGraph | AI civilization |
| Entity Events | 828 | Pet world, stats, payouts, tournaments | Stat overlay system |
| Theme Engine | 821 | World dynamics, weights, coherence | 3D world binding |
| Seasonal Events | 874 | Events, treasure caches, user events | Event engine |
| Tournament Manager | 950 | Brackets, registration, rewards | Competition system |
| Economy Service | 724 | Balances, faucet, trading, shops | Economic engine |
| Religion Governance | 402 | 24-cap chain store, buyouts, rituals | Faith civilization |
| Entity Market | 599 | Public listings, trade history | Asset exchange |
| Asset Life | 216 | Pets, vehicles, world content | Entity assets |
| Rivalry Engine | 258 | Career rivalries, XP modifiers | Social dynamics |
| Faith Church | ~300 | Church storefront, members, items, rituals, leaderboard | Faith economy |

---

## Current Session Systems (2026-09-07) ✅ DONE

These systems were built in the most recent sessions and are NOT in the legacy Phase structure below. They are fully implemented and verified.

### Menu Customization System
- **Button Shapes:** 60+ shapes in `menu_customization.js` — base shapes (circle, star, triangle, hexagon, etc.) + domain-specific (justice-scales, skull, church, flame, etc.)
- **Size Tiers:** 6 tiers (xs/sm/md/lg/xl/xxl) with responsive scaling
- **Grid Layouts:** 9 layouts — grid, circle (radial wheel), triangle (pyramid), cross, arc, diamond, linearH, linearV, freeform
- **Controller Nav:** `controller_nav.js` — grid-specific navigation models (radial, layered, cardinal, diagonal, sweep, etc.) + gamepad polling + analog stick support
- **Customization Panel:** `menu_customization_panel.js` — 4-tab overlay (Shape, Grid, Buttons, Controller) with live preview
- **Per-Button Override:** Individual buttons can break from global defaults

### World Dashboard
- **Categories:** 12 categories grouping 43 backend routes (Player Hub, Assets, Economy, Faction, Governance, World & Events, Faith, Orphans, Creator, Infrastructure, Competition, Moderation)
- **Starring:** ☆/★ toggle on each tab → appears in Main Menu favorites grid
- **Dynamic Layout:** Grid type, size, shape all customizable via UserPreferences

### Pathway Avenues
- **12 Pathways:** P-Shadow, P-Lockdown, P-Ledger, P-Syndicate, P-Tax, P-Peace, P-Intel, P-Justice, P-AOS, P-Commissioner, P-Forensic, P-Boss
- **Styling:** Each has unique color palette, icon, animation style
- **Tier System:** 5 tiers (Initiate → Master) with XP requirements

### Debug Tools
- **`flow debug`:** 53-check module debugger (shapes, sizes, grids, controller nav)
- **`flow verify`:** 22-check pipeline integration verifier
- **Usage:** `python tools/main.py flow debug` → exit 0 (all pass)

### Frontend Migration
- **ES Modules:** All new code uses `import/export` (no IIFE, no inline `<script>`)
- **SCSS:** All styling in `Public/src/scss/features/` (no inline CSS in index.html)
- **Relative Paths:** All code uses relative paths (`./js/file.js`, `./main.wasm`)
- **Architecture:** All UI modules import through `app.js` orchestrator — no standalone `<script>` tags in `index.html`

### World Dashboard UI (43 tabs — all unique visual themes)

| # | Tab | Visual Theme | File |
|---|-----|--------------|------|
| 1 | Career Tree | Dark geometric, faction cards | `career_tree.js` |
| 2 | Club Foundry | Industrial/foundry | `club_foundry.js` |
| 3 | Tournament Brackets | Orange competitive arena | `tournament_brackets.js` |
| 4 | Justice Dashboard | Wanted poster/law enforcement | `justice_dashboard.js` |
| 5 | Loan Terms | Financial/banking | `loan_terms.js` |
| 6 | AMM Chart | Trading terminal, SVG bonding curve | `amm_chart.js` |
| 7 | Pet Breeder | Genetics lab/DNA helix | `pet_breeder.js` |
| 8 | Church Storefront | Cathedral/stained glass | `church_storefront.js` |
| 9 | Rivalry Challenge | Gladiator arena/duel | `rivalry_challenge.js` |
| 10 | Achievement Progress | Trophy hall/medal cabinet | `achievement_progress.js` |
| 11 | Report History | File cabinet/case management | `report_history.js` |
| 12 | Industrial Flow | Machinery/pipeline | `industrial_flow.js` |
| 13 | Governance Chambers | Senate/political radar | `governance_chambers.js` |
| 14 | Treasure Map | Pirate map/hunt | `treasure_map.js` |
| 15 | Season Countdown | Celestial/calendar orbit | `season_countdown.js` |
| 16 | Replay Viewer | Film player/theater | `replay_viewer.js` |
| 17 | Identity Editor | Passport/holographic seal | `identity_editor.js` |
| 18 | Dividend Yield | Growth garden/orchard | `dividend_yield.js` |
| 19 | Underworld Contracts | Criminal notice board | `underworld_contracts.js` |
| 20 | Black Market | Neon-noir bazaar | `black_market.js` |
| 21 | Items Equip | Inventory/slots | `items_equip.js` |
| 22 | Events Arena | Festival/arena tickets | `events_arena.js` |
| 23 | Territory Map | Fantasy map/flags | `territory_map.js` |
| 24 | Rewards Center | Gift shop/vault | `rewards_center.js` |
| 25 | Match Arena | Arena entrance/spotlight | `match_arena.js` |
| 26 | Counterfeit Scanner | Forensic lab/UV scan | `counterfeit_scanner.js` |
| 27 | Creator Studio | Artist studio/workshop | `creator_studio.js` |
| 28 | Faith War Gambit | Cathedral/candlelight | `faith_war_gambit.js` |
| 29-43 | Remaining tabs | Various themes | `remaining_tabs.js` |

**Files Created:** 30 JS modules + 30 SCSS partials = 60 new files

**Verification:** `python tools/main.py flow verify` → 22/22 pass. Both build targets GREEN.

---

## Legacy Phases (2026-08-30 and earlier) — ALL ✅ DONE

> The following phases are complete and verified. They remain for historical reference.

## Phase 1: Stat Overlay + Industrial Loop ✅ DONE

### Stat Overlay System
- **Backend:** `stat_overlay.go` — ComputeStatOverlay for ALL entity types
- **Frontend:** `stat_overlay.js` — Leaderboard, My Entities, Region views
- **Entities:** Players, AI Citizens, LLM Bots, Rogue Bots, Children Bots, Pets, Vehicles
- **Features:** Power buff system, effective power calculation, dominant stat, earning capabilities
- **Routes:** `/api/stat-overlay`, `/api/stat-overlay/region`, `/api/stat-overlay/owner`, `/api/stat-overlay/leaderboard`

### Industrial Loop Engine
- **Backend:** `industrial_loop.go` — Perpetual circulation engine
- **Frontend:** `industrial_loop.js` — Flow visualization, health metrics
- **Phases:** Activity → Business → Employment → Purchasing → Taxation → Treasury → Development → Events
- **Features:** Flow tracking, loop health, circulation efficiency
- **Routes:** `/api/industrial-loop/metrics`, `/api/industrial-loop/health`, `/api/industrial-loop/record`

**Existing:** Faucet → Players → Shops → Faucet (partial loop)
**Vision:** Player Activity → Businesses → Employment → Purchasing → Taxes → Treasuries → Development → Events → Player Activity

| Step | What | Existing Foundation | New Work |
|------|------|---------------------|----------|
| 1 | **Player Activity** | Battle rewards, tournament payouts | Activity tracker |
| 2 | **Businesses** | Shops, creator store | Business entities, production |
| 3 | **Employment** | hire_player, set_salary | Job market, AI employees |
| 4 | **Purchasing** | buyClubItem, shops | Purchase tracking |
| 5 | **Taxes** | set_district_tax | Auto-taxation, treasury routing |
| 6 | **Treasuries** | Club treasury | Treasury management UI |
| 7 | **Development** | — | Regional development projects |
| 8 | **Events** | Seasonal, tournaments | Event generation from activity |
| 9 | **Loop Close** | — | Automated circulation engine |

**Status:** ✅ DONE — backend + frontend complete. Both build targets GREEN.

### Stat Overlay System
- **Backend:** `stat_overlay.go` — ComputeStatOverlay for ALL entity types
- **Frontend:** `stat_overlay.js` — Leaderboard, My Entities, Region views
- **Entities:** Players, AI Citizens, LLM Bots, Rogue Bots, Children Bots, Pets, Vehicles
- **Features:** Power buff system, effective power calculation, dominant stat, earning capabilities
- **Routes:** `/api/stat-overlay`, `/api/stat-overlay/region`, `/api/stat-overlay/owner`, `/api/stat-overlay/leaderboard`

### Industrial Loop Engine
- **Backend:** `industrial_loop.go` — Perpetual circulation engine
- **Frontend:** `industrial_loop.js` — Flow visualization, health metrics
- **Phases:** Activity → Business → Employment → Purchasing → Taxation → Treasury → Development → Events
- **Features:** Flow tracking, loop health, circulation efficiency
- **Routes:** `/api/industrial-loop/metrics`, `/api/industrial-loop/health`, `/api/industrial-loop/record`

**Existing:** Faucet → Players → Shops → Faucet (partial loop)
**Vision:** Player Activity → Businesses → Employment → Purchasing → Taxes → Treasuries → Development → Events → Player Activity

| Step | What | Existing Foundation | New Work |
|------|------|---------------------|----------|
| 1 | **Player Activity** | Battle rewards, tournament payouts | Activity tracker |
| 2 | **Businesses** | Shops, creator store | Business entities, production |
| 3 | **Employment** | hire_player, set_salary | Job market, AI employees |
| 4 | **Purchasing** | buyClubItem, shops | Purchase tracking |
| 5 | **Taxes** | set_district_tax | Auto-taxation, treasury routing |
| 6 | **Treasuries** | Club treasury | Treasury management UI |
| 7 | **Development** | — | Regional development projects |
| 8 | **Events** | Seasonal, tournaments | Event generation from activity |
| 9 | **Loop Close** | — | Automated circulation engine |

## Phase 2: Persistent Identity ✅ COMPLETE
- **Backend:** `persistent_identity.go` — History chain, reputation engine, identity score, tiers
- **Frontend:** `persistent_identity.js` — Profile, history, leaderboard
- **Routes:** `/api/identity/profile`, `/api/identity/events`, `/api/identity/leaderboard`, `/api/identity/record`

## Phase 3: Entity Markets ✅ COMPLETE
- **Backend:** `entity_shares.go` — Share tokens, buy/sell, holdings, dividends
- **Frontend:** `entity_shares.js` — Tokens, holdings, issue
- **Routes:** `/api/shares/issue`, `/api/shares/buy`, `/api/shares/tokens`, `/api/shares/holdings`

## Phase 4: Infrastructure Leasing ✅ COMPLETE
- **Backend:** `infrastructure_lease.go` — 10 system types (wallet, auth, economy, AI, social, tournament, leaderboard, crosschain, creator, analytics)
- **Frontend:** `infrastructure_lease.js` — Available systems, my leases
- **Routes:** `/api/lease/create`, `/api/lease/list`, `/api/lease/available`

## Phase 5: AI Civilization ✅ COMPLETE (2026-09-05)
- **Backend:** `ai_citizen_engine.go` — Full behavioral loop with 10 dynamics, BondGraph, faith, marriage, breeding, pets, justice, employment, rivalry, market, career progression, event hosting
- **Faith System:** 24 religions seeded, church storefront, rituals, faith coherence, war gambit
- **Domestic System:** Marriage, bot-children, AI-character pets via BondGraph
- **Persistence:** gzip-backed Save/Load for citizens, religions, churches
- **HTTP Routes:** `/api/ai/citizens/*` (spawn, marry, breed, adopt-pet, progress, list, stats, free-agents), `/api/faith/coherence`, `/api/faith/war-gambit`
- **Dev Seed:** 2 AI citizens spawn on boot for testing
- **Frontend:** Ready for UI development (backend complete)

## Phase 6: Stat Overlay ✅ COMPLETE
- **Backend:** `stat_overlay.go` — ComputeStatOverlay for ALL entity types
- **Frontend:** `stat_overlay.js` — Leaderboard, My entities, Region
- **Routes:** `/api/stat-overlay`, `/api/stat-overlay/region`, `/api/stat-overlay/owner`, `/api/stat-overlay/leaderboard`

## Phase 7: Creator Economy ✅ COMPLETE
- **Backend:** `creator_economy.go` — DLC, royalties, events, subscriptions
- **Frontend:** `creator_economy.js` — DLC, events, royalties, subscriptions
- **Routes:** `/api/creator/dlc/create`, `/api/creator/dlc/purchase`, `/api/creator/dlcs`, `/api/creator/royalties`, `/api/creator/event/create`, `/api/creator/events`, `/api/creator/event/attend`, `/api/creator/sub/create`, `/api/creator/subs`

## Phase 8: Multi-Chain ✅ COMPLETE
- **Backend:** `bridge_router.go` — ETH/MATIC/BTC/SOL bridges, asset origin tracking
- **Frontend:** `bridge_router.js` — Assets, bridge form, summary
- **Routes:** `/api/bridge/asset`, `/api/bridge/confirm`, `/api/bridge/assets`, `/api/bridge/txs`, `/api/bridge/summary`

## Phase 9: Governance ✅ COMPLETE
- **Backend:** `governance.go` — Weighted elections, regional governors, trust/reputation/economic/competitive/community/creative scores
- **Frontend:** `governance.js` — My weight, election, leaderboard
- **Routes:** `/api/governance/weight`, `/api/governance/register`, `/api/governance/vote`, `/api/governance/close`, `/api/governance/governor`, `/api/governance/election`, `/api/governance/leaderboard`

## Phase 10: Faith Church Storefront ✅ COMPLETE
- **Backend:** `faith_church.go` — Open church, members, items, rituals, leaderboard
- **Frontend:** `faith_church.js` — My churches, region, leaderboard, open form
- **Routes:** `/api/church/open`, `/api/church/get`, `/api/church/owner`, `/api/church/region`, `/api/church/add-member`, `/api/church/remove-member`, `/api/church/add-item`, `/api/church/items`, `/api/church/ritual`, `/api/church/rituals`, `/api/church/leaderboard`

## Phase 11: Launchpad + Advertising + Gaming OS + Compliance ✅ COMPLETE

### Launchpad System
- **Backend:** `launchpad.go` — Create, back, activate, integrate launches
- **Frontend:** `launchpad.js` — Active launches, my launches, create form
- **Routes:** `/api/launch/create`, `/api/launch/back`, `/api/launch/activate`, `/api/launch/integrate`, `/api/launches`, `/api/launch/get`, `/api/launch/creator`

### Advertising System
- **Backend:** `advertising.go` — Create ads, impressions, clicks, targeting
- **Frontend:** `advertising.js` — Active ads, my ads, region ads, create form
- **Routes:** `/api/ad/create`, `/api/ad/activate`, `/api/ad/pause`, `/api/ad/impression`, `/api/ad/click`, `/api/ads`, `/api/ads/advertiser`, `/api/ads/region`, `/api/ad/stats`

### Gaming OS System
- **Backend:** `gaming_os.go` — OS modules, leases, civilization-as-a-service
- **Frontend:** `gaming_os.js` — Modules, leases, compliance
- **Routes:** `/api/os/modules`, `/api/os/module/register`, `/api/os/lease`, `/api/os/leases`, `/api/os/summary`

### Compliance System
- **Backend:** `gaming_os.go` — Compliance records, KYC/AML, audit trails
- **Frontend:** `gaming_os.js` — Compliance records, resolve, escalate
- **Routes:** `/api/compliance/record`, `/api/compliance/resolve`, `/api/compliance/escalate`, `/api/compliance/records`, `/api/compliance/wallet`, `/api/compliance/summary`

---

## Integration Map (How Systems Strengthen Each Other)

```
Industrial Loop ←→ Persistent Identity (activity builds history)
Industrial Loop ←→ Entity Markets (businesses need investment)
Industrial Loop ←→ Infrastructure Leasing (systems need users)
Industrial Loop ←→ AI Civilization (AI workers drive economy)
Industrial Loop ←→ Creator Economy (creators drive activity)

Persistent Identity ←→ Entity Markets (reputation affects valuation)
Persistent Identity ←→ Governance (reputation affects leadership)
Persistent Identity ←→ Faith (religious reputation)

Entity Markets ←→ Infrastructure Leasing (investors lease systems)
Entity Markets ←→ Creator Economy (creators get funded)

AI Civilization ←→ Industrial Loop (AI workers drive production)
AI Civilization ←→ Entity Markets (AI entities get invested in)
AI Civilization ←→ Stat Overlay (AI has stats)

Stat Overlay ←→ All Entities (everyone has stats)
Stat Overlay ←→ 3D World (stats render in 3D)
Stat Overlay ←→ Events (stats affect outcomes)

Faith ←→ Governance (religious leadership)
Faith ←→ Persistent Identity (religious reputation)
Faith ←→ Entity Markets (religious entities)
```

---

## Priority Order (Updated 2026-09-07)

| Priority | System | Status |
|----------|--------|--------|
| 1 | **World Dashboard UI (43 tabs)** | ✅ DONE |
| 2 | **Main Menu Customization** | ✅ DONE |
| 3 | **Pathway Avenues** | ✅ DONE |
| 4 | **Controller Navigation** | ✅ DONE |
| 5 | **Debug Tools (flow debug/verify)** | ✅ DONE |
| 6 | **Code Audit & Fix (duplicates, orphans)** | ✅ DONE |
| 7 | **Architecture Diagram Refresh** | ✅ DONE |
| — | **All backend Phases 1-11** | ✅ DONE |

---

*This is the synergized master plan. Build on what exists. Fill what's missing. Maximum leverage always.*
