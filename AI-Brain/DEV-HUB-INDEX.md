# Dev-Game-Hub Index — Composable Framework Functions

> **Purpose:** Living catalog of the app's base framework functions + combinations.
> The **dev-game-hub** (deferred, built LAST) is a fine-grain storefront that SELLS these
> functions to a dev-user so they can build their own game. Its only tie to the leaderboard
> is the power it adds to a dev-user. This index is the source-of-truth the hub will reflect.
> Each entry = one composable, fine-grain function. Grow it as systems are fleshed/built.
> Rule (Brendan 2026-08-31): keep every piece discrete + composable so the hub can expose
> the FULL set of base framework functions + combinations.

## Build status legend
- ✅ built + wired (front + back where applicable)
- 🟡 backend only (no player UI yet)
- ⬜ deferred / not started

---

## 1. Economy & Ledger (uint64 micro, no float — constitutional)
- ✅ `HandlePurchaseItem` — buy club item (VBV only; NUGGET/UNIT settlement deferred)
- ✅ `submitTerritoryPurchase` / `HandlePurchaseTerritory` — expand club to 2nd+ territory (2,500 VBV on-chain)
- ✅ `resolveShopToken` — category→token preset resolver (admin sets presets)
- ✅ `vault_donation` (WS) — player donates $VBV to House vault (credits faucetBalanceMicro)
- ✅ `set_district_tax` (WS) — governor sets localized sales tax (0–20%)

## 2. Clubs / Territories / Regions
- ✅ `create_club` / `HandleCreateClub` — found club (1 territory, 5,000 VBV)
- ✅ `purchase_territory` / `HandlePurchaseTerritory` — acquire unclaimed district
- ✅ `open_regional_manager` / `HandleOpenRegionalManager` — player-initiated governor ability (≥2 territories)
- ✅ `IsClubRegionalLocked` — governor at ≥2 territories (incl. alliance)
- ✅ `refreshRegionalRoles` — governor refresh at season rollover only

## 3. AI Citizens (own-wallet mandate ENFORCED)
- ✅ `SpawnAI` — dedicated `0xai…` wallet (never personal); guard at ai_citizen_engine.go:295
- ✅ `triggerEntityInvestment` — citizen invests from OWN wallet
- ✅ `SpawnBusiness` — citizen spawns NPC business (Journeyman+)

## 4. Matchmaking / Combat
- ✅ `join_queue` / `leave_queue` (WS) — matchmaking pool enter/exit
- ✅ `challenge` (WS) — PvP challenge
- ✅ `move` (WS) — in-match move
- ✅ `update_rating` (WS) — sync BestRating + MatchRating
- ✅ autonomous tournament scheduler (`entity_tournament_scheduler.go`, 15m ticker)

## 5. Justice / Underworld / Recovery
- ✅ `justice_flag_player`, `accept_justice_mission`, `abort_justice_mission`
- ✅ `accept_underworld_contract`, `abort_underworld_contract`, `launder_capital`, `purify_card`
- ✅ `aos_raid`, `purchase_raid_insurance`, `purchase_bounty_license`, `purchase_bounty_bond`, `refund_bounty_bond`
- ✅ `initiate_recovery`, `list_recovery_bounty`, `pay_ransom`, `release_hostage`, `spread_rumor`
- ✅ `bounty_board` / `capture_bounty` (REST + WS)

## 6. Social / Communication
- ✅ `chat` (WS) — lobby chat broadcast
- ✅ `spectate` (WS) — watch a player's match (cycle-feed hook for §25 3D)
- ✅ `report_gloat`, `alliance_invite` / `accept` / `dissolve`

## 7. Markets / Shops / Creator
- ✅ `GlobalShopRegistry` — 6 shop categories (Elemental/Tactical/Vitality/Hardware/Nugget/Unit)
- ✅ `buyClubItem` — token-locked purchase (VBV default; Nugget/Unit per preset)
- ✅ Creator Store (`/api/creator/store/*`), Consignment, Black Market, Auctions
- ✅ Item Builder (`/api/items/build`), Bonded Asset registry (§23 NFT-everything)

## 8. World / Events / Seasons
- ✅ Treasure caches (`/api/treasure/spawn|claim`)
- ✅ User events (`/api/events/create|enter`)
- ✅ Seasonal engine (`/api/season/*`, `GetRegionViews` → `/api/regions`)
- ✅ §25 3D world (`world3d.js`, `enter3DWorld`/`enterMenuWorld`, region warp + spectator cycle)

## 9. Rivalry / Synergy / Faith (place + household)
- ✅ Rivalry engine (`rivalry_engine.go`) + 8 `/api/rivalry/*` routes + `rivalry.js`/`rivalry_viewer.js` UI
- ✅ Faith (`/api/faith/coherence`, `/api/faith/war-gambit`) + Religious Leader card modifier
- ✅ Synergy (`/api/owner/combined-stats`, `CombineOwnerStats`)
- ✅ Orphan/Adopt (`/api/orphan/{adopt,reclaim,status}`)

## 10. Pet World / Entities (§30/§31)
- ✅ `EntityStats` (6 uint64 stats) on PetNFT + AICitizen + RegionView
- ✅ `ComputeEffectivePowerLevel` (caps 600), `BuildRegionPowerOverlay`
- ✅ `/api/pets/spawn|breed`, `/api/vehicles/spawn`, `/api/entity-event/resolve`, `/api/entity-event/host`
- ✅ event-reward payout loop + autonomous scheduler (G4 closed)
- ✅ **Stat Overlay System** — separate power system (not a token), applies to all entities
- ✅ Children Bots (derived AICitizen, distinct from AI citizens)
- ✅ Entity Market (public listings for pets/children bots/vehicles)
- ✅ Pet Battle Arena (stat overlay rewards, rivalry factions)
- ✅ Combined Events (all entity types together)

## 11. Identity / Admin / Dev
- ✅ `link_wallet_request` / `refresh_identity` / `identity/*` (WS + REST)
- ✅ Admin panel (`admin.js`): broadcast, maintenance, token presets, ledger/commission/tax audits, season rollover, simulate-*, start-tournament, emergency-shutdown
- ✅ Local-model promotion (`/api/local-model/promote|status`) — per-user ornith model (§24.5/§24.6)
- ✅ **Dev-Game-Hub storefront** — 40+ framework functions across 14 categories

## 12. Composable Mechanic Framework (§25.5 developer game-hub)
- ✅ `mechanics.js` — `engine.register({...})` composable mechanics (uint64 micro, deterministic FNV hash)
- ✅ `mechanic_defs.js` — region_vitality / club_mojo / rivalry_aura / theme_gravity / career_ripple
- This framework IS the dev-hub's "base framework" surface — every mechanic a dev-user can compose.

## 13. Religion Governance (§30.4)
- ✅ 24-cap faucet-owned chain store (`religion_governance.go`)
- ✅ Buy governorship + buyout system (10% above last price)
- ✅ All buyouts → faucet (no commission for old owner)
- ✅ Converted users can ONLY buy out another to return
- ✅ On buyout: restore old OR create new
- ✅ High-tier rivals (top 12 by power)
- ✅ Rituals raise FaithCoherence
- ✅ Governors define rivalry sets

## 14. Stat Overlay System (Phase 1)
- ✅ `stat_overlay.go` — ComputeStatOverlay for ALL entity types
- ✅ `stat_overlay.js` — Leaderboard, My Entities, Region views
- ✅ Entities: Players, AI Citizens, LLM Bots, Rogue Bots, Children Bots, Pets, Vehicles
- ✅ Power buff system, effective power calculation, dominant stat, earning capabilities
- ✅ Routes: `/api/stat-overlay`, `/api/stat-overlay/region`, `/api/stat-overlay/owner`, `/api/stat-overlay/leaderboard`

## 15. Industrial Loop Engine (Phase 1)
- ✅ `industrial_loop.go` — Perpetual circulation engine
- ✅ `industrial_loop.js` — Flow visualization, health metrics
- ✅ Phases: Activity → Business → Employment → Purchasing → Taxation → Treasury → Development → Events
- ✅ Flow tracking, loop health, circulation efficiency
- ✅ Routes: `/api/industrial-loop/metrics`, `/api/industrial-loop/health`, `/api/industrial-loop/record`

## 16. Persistent Identity System (Phase 2)
- ✅ `persistent_identity.go` — History chain, reputation engine, identity score, tiers
- ✅ `persistent_identity.js` — Profile, history, leaderboard
- ✅ Routes: `/api/identity/profile`, `/api/identity/events`, `/api/identity/leaderboard`, `/api/identity/record`

## 17. Entity Shares System (Phase 3)
- ✅ `entity_shares.go` — Share tokens, buy/sell, holdings, dividends
- ✅ `entity_shares.js` — Tokens, holdings, issue
- ✅ Routes: `/api/shares/issue`, `/api/shares/buy`, `/api/shares/tokens`, `/api/shares/holdings`

## 18. Infrastructure Leasing System (Phase 4)
- ✅ `infrastructure_lease.go` — 10 system types (wallet, auth, economy, AI, social, tournament, leaderboard, crosschain, creator, analytics)
- ✅ `infrastructure_lease.js` — Available systems, my leases
- ✅ Routes: `/api/lease/create`, `/api/lease/list`, `/api/lease/available`

## 19. Governance System (Phase 9)
- ✅ `governance.go` — Weighted elections, regional governors, trust/reputation/economic/competitive/community/creative scores
- ✅ `governance.js` — My weight, election, leaderboard
- ✅ Routes: `/api/governance/weight`, `/api/governance/register`, `/api/governance/vote`, `/api/governance/close`, `/api/governance/governor`, `/api/governance/election`, `/api/governance/leaderboard`

## 20. Bridge Router System (Phase 8)
- ✅ `bridge_router.go` — ETH/MATIC/BTC/SOL bridges, asset origin tracking
- ✅ `bridge_router.js` — Assets, bridge form, summary
- ✅ Routes: `/api/bridge/asset`, `/api/bridge/confirm`, `/api/bridge/assets`, `/api/bridge/txs`, `/api/bridge/summary`

## 21. Creator Economy System (Phase 7)
- ✅ `creator_economy.go` — DLC, royalties, events, subscriptions
- ✅ `creator_economy.js` — DLC, events, royalties, subscriptions
- ✅ Routes: `/api/creator/dlc/create`, `/api/creator/dlc/purchase`, `/api/creator/dlcs`, `/api/creator/royalties`, `/api/creator/event/create`, `/api/creator/events`, `/api/creator/event/attend`, `/api/creator/sub/create`, `/api/creator/subs`

## 22. Faith Church Storefront (Phase 10)
- ✅ `faith_church.go` — Open church, members, items, rituals, leaderboard
- ✅ `faith_church.js` — My churches, region, leaderboard, open form
- ✅ Routes: `/api/church/open`, `/api/church/get`, `/api/church/owner`, `/api/church/region`, `/api/church/add-member`, `/api/church/remove-member`, `/api/church/add-item`, `/api/church/items`, `/api/church/ritual`, `/api/church/rituals`, `/api/church/leaderboard`

## 23. Native Build Systems (Console/Android/iPhone)
- ✅ `console_server.go` — Full loopback-only console authority with all Phase 1-11 routes
- ✅ `build_console.ps1` / `build_console.sh` — Windows/Linux console builds
- ✅ `build_mobile.ps1` / `build_mobile.sh` — Android (CGO_ENABLED=0) / iOS (CGO_ENABLED=1) builds
- ✅ Verified: Windows console GREEN, Android arm64 GREEN, iOS arm64 requires CGO_ENABLED=1

## 24. Launchpad System (Phase 11)
- ✅ `launchpad.go` — Create, back, activate, integrate launches
- ✅ `launchpad.js` — Active launches, my launches, create form
- ✅ Routes: `/api/launch/create`, `/api/launch/back`, `/api/launch/activate`, `/api/launch/integrate`, `/api/launches`, `/api/launch/get`, `/api/launch/creator`

## 25. Advertising System (Phase 11)
- ✅ `advertising.go` — Create ads, impressions, clicks, targeting
- ✅ `advertising.js` — Active ads, my ads, region ads, create form
- ✅ Routes: `/api/ad/create`, `/api/ad/activate`, `/api/ad/pause`, `/api/ad/impression`, `/api/ad/click`, `/api/ads`, `/api/ads/advertiser`, `/api/ads/region`, `/api/ad/stats`

## 26. Gaming OS + Compliance System (Phase 11)
- ✅ `gaming_os.go` — OS modules, leases, compliance records
- ✅ `gaming_os.js` — Modules, leases, compliance
- ✅ Routes: `/api/os/modules`, `/api/os/module/register`, `/api/os/lease`, `/api/os/leases`, `/api/os/summary`, `/api/compliance/record`, `/api/compliance/resolve`, `/api/compliance/escalate`, `/api/compliance/records`, `/api/compliance/wallet`, `/api/compliance/summary`

---
*Last updated: 2026-09-01 (KEY 3.5 Full Build-Out YOLO). Authoritative build gate: `GOOS=linux GOARCH=amd64 go build ./...` GREEN. ALL PHASES + NATIVE BUILDS + 100% VISION COVERAGE COMPLETE. No remaining DEFERRED items.*