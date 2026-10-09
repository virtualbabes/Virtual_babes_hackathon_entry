# DRIFT MATRIX: Backend Routes × Frontend JS Wiring

**Generated:** 2026-09-03  
**Repo:** Z:\Crypto_Draught\NFT-Seduction  
**Sources:**
- Backend REST routes: `server_main.go` (257 API routes)
- Backend WS handlers: `lobby_manager.go` → `handleGameProtocol()` (63 message types)
- Frontend JS: 71 files in `Public/js/` (excluding vendor/)

---

## Executive Summary

| Layer | Total Exposed | Wired in JS | Orphaned | Coverage |
|-------|--------------|-------------|----------|----------|
| REST API Routes | 257 | 40 | 217 | **15.6%** |
| WS Message Types | 63 | 51 | 12 | **81.0%** |
| **Combined** | **320** | **91** | **229** | **28.4%** |

**Key Finding:** The REST layer has extreme drift — 84.4% of backend API routes have zero JS wiring.  
The WebSocket layer is much healthier — 81% of message types are referenced in frontend code.

---

## REST API Drift Analysis

### Wired Routes (40)

- **/api/ad/create**
  - `advertising.js`
- **/api/ads/advertiser**
  - `advertising.js`
- **/api/bridge/asset**
  - `bridge_router.js`
- **/api/bridge/assets**
  - `bridge_router.js`
- **/api/bridge/summary**
  - `bridge_router.js`
- **/api/church/leaderboard**
  - `faith_church.js`
- **/api/church/open**
  - `faith_church.js`
- **/api/church/owner**
  - `faith_church.js`
- **/api/compliance/records**
  - `gaming_os.js`
- **/api/creator/dlc/purchase**
  - `creator_economy.js`
- **/api/creator/dlcs**
  - `creator_economy.js`
- **/api/creator/event/attend**
  - `creator_economy.js`
- **/api/creator/events**
  - `creator_economy.js`
- **/api/creator/royalties**
  - `creator_economy.js`
- **/api/creator/store/product/**
  - `creator_storefront.js`
- **/api/creator/store/products**
  - `creator_storefront.js`
- **/api/creator/store/rate/**
  - `creator_storefront.js`
- **/api/creator/subs**
  - `creator_economy.js`
- **/api/entity-event/host**
  - `combined_events.js`
- **/api/entity-event/resolve**
  - `combined_events.js`
- **/api/entity-events/regions**
  - `combined_events.js`
- **/api/faith/war-gambit**
  - `faith_system.js`
- **/api/faucet/status**
  - `faucet_dashboard.js`
- **/api/governance/weight**
  - `governance.js`
- **/api/justice/capture-bounty**
  - `bounty_tracker.js`
  - `justice_dashboard.js`
- **/api/justice/dashboard**
  - `justice_dashboard.js`
- **/api/justice/use-truth-serum**
  - `justice_dashboard.js`
- **/api/launch/back**
  - `launchpad.js`
- **/api/launch/create**
  - `launchpad.js`
- **/api/launch/creator**
  - `launchpad.js`
- **/api/lease/available**
  - `infrastructure_lease.js`
- **/api/lease/create**
  - `infrastructure_lease.js`
- **/api/lease/list**
  - `infrastructure_lease.js`
- **/api/os/lease**
  - `gaming_os.js`
- **/api/os/leases**
  - `gaming_os.js`
- **/api/os/modules**
  - `gaming_os.js`
- **/api/shares/buy**
  - `entity_shares.js`
- **/api/shares/holdings**
  - `entity_shares.js`
- **/api/shares/issue**
  - `entity_shares.js`
- **/api/shares/tokens**
  - `entity_shares.js`


### Orphaned Routes (217)

These routes are registered in `server_main.go` but never referenced in any JS file:

- /api/achievement/unlock
- /api/achievements
- /api/achievement-stats
- /api/ad/activate
- /api/ad/click
- /api/ad/impression
- /api/ad/pause
- /api/ad/stats
- /api/admin/asset-forfeiture
- /api/admin/avatar-ban
- /api/admin/commission-audit
- /api/admin/district-tax-audit
- /api/admin/dlc-registry
- /api/admin/dlc-registry/restock
- /api/admin/dlc-registry/update
- /api/admin/emergency-shutdown
- /api/admin/export-logs
- /api/admin/force-payout
- /api/admin/gloat-ban
- /api/admin/ledger-audit
- /api/admin/logs
- /api/admin/mutation-audit
- /api/admin/network/add
- /api/admin/open-registration
- /api/admin/sanity-check
- /api/admin/season-rollover
- /api/admin/set-admin-focus-network
- /api/admin/simulate-load
- /api/admin/simulate-mojo-decay
- /api/admin/simulate-mutation-failure
- /api/admin/simulate-mutation-success
- /api/admin/simulate-tournament
- /api/admin/start-tournament
- /api/admin/tax-audit
- /api/admin/update-power
- /api/ads
- /api/ads/region
- /api/ai/citizens/adopt
- /api/ai/citizens/business/spawn
- /api/ai/citizens/challenge
- /api/ai/citizens/free-agents
- /api/ai/citizens/list
- /api/ai/citizens/release
- /api/ai/citizens/spawn
- /api/ai/citizens/stats
- /api/assets
- /api/assets/burn
- /api/assets/mint
- /api/assets/modify
- /api/assets/transfer
- /api/auctions
- /api/ban-player
- /api/black-market/buy
- /api/black-market/buy-stolen
- /api/black-market/fence-goods
- /api/black-market/sell-tokens
- /api/bridge/confirm
- /api/bridge/onboard
- /api/bridge/txs
- /api/card-details
- /api/card-stats
- /api/career/progress
- /api/children-bots
- /api/church/add-item
- /api/church/add-member
- /api/church/get
- /api/church/items
- /api/church/region
- /api/church/remove-member
- /api/church/ritual
- /api/church/rituals
- /api/claim/dividends
- /api/client-error
- /api/compliance/escalate
- /api/compliance/record
- /api/compliance/resolve
- /api/compliance/summary
- /api/compliance/wallet
- /api/contracts/assign
- /api/contracts/list
- /api/counterfeit/detect
- /api/counterfeit/generate
- /api/courthouse/reset
- /api/creator/dlc/create
- /api/creator/event/create
- /api/creator/store/deactivate/
- /api/creator/store/profile/
- /api/creator/store/purchase/
- /api/creator/store/reactivate/
- /api/creator/store/resell
- /api/creator/store/royalty-history
- /api/creator/sub/create
- /api/criminality/cyber-intercept
- /api/entity/market/create
- /api/entity/market/list
- /api/entity/market/purchase
- /api/events/create
- /api/events/enter
- /api/faction/shop/
- /api/faith/coherence
- /api/faith/converted
- /api/faith/high-tier
- /api/faith/religion/buy
- /api/faith/religion/buyout
- /api/faith/religion/join
- /api/faith/religion/ritual
- /api/faith/religion/rivalry
- /api/faith/religions
- /api/governance/close
- /api/governance/election
- /api/governance/governor
- /api/governance/leaderboard
- /api/governance/register
- /api/governance/vote
- /api/identity/events
- /api/identity/leaderboard
- /api/identity/link
- /api/identity/profile
- /api/identity/record
- /api/identity/resolve
- /api/identity/snapshot
- /api/identity/unlink
- /api/industrial-loop/health
- /api/industrial-loop/metrics
- /api/industrial-loop/record
- /api/invest/dividends/history
- /api/invest/entity
- /api/invest/portfolio
- /api/items/archetypes
- /api/items/bind-nft
- /api/items/build
- /api/items/collection
- /api/items/registry
- /api/justice/award-card
- /api/justice/bounty-board
- /api/justice/use-rep-shield
- /api/launch/activate
- /api/launch/get
- /api/launch/integrate
- /api/launches
- /api/leaderboard
- /api/loans
- /api/loans/repay
- /api/loans/take
- /api/local-model/promote
- /api/local-model/status
- /api/maintenance-mode
- /api/market/weather
- /api/match/wager
- /api/orphan/adopt
- /api/orphan/reclaim
- /api/orphan/status
- /api/os/module/register
- /api/os/summary
- /api/owner/combined-stats
- /api/pet-battle/challenge
- /api/pet-battle/list
- /api/pet-battle/resolve
- /api/pets
- /api/pets/breed
- /api/pets/spawn
- /api/player/progression
- /api/refill-vault
- /api/regions
- /api/replay/capture
- /api/replay/frames
- /api/replay/latest
- /api/replay/player
- /api/replay/start
- /api/replay/state
- /api/replay/stop
- /api/report-player
- /api/reset-stats
- /api/re-sync-stats
- /api/reward
- /api/reward/add
- /api/reward/remove
- /api/reward/update-asset
- /api/reward/update-base
- /api/rivalry/action
- /api/rivalry/detect
- /api/rivalry/factions
- /api/rivalry/join
- /api/rivalry/list
- /api/rivalry/recompute
- /api/rivalry/request
- /api/rivalry/resolve
- /api/rivalry/state
- /api/rivalry/world-dynamics
- /api/season/admin/create-event
- /api/season/admin/end-event
- /api/season/admin/update-reward-pool
- /api/season/events
- /api/season/events/join
- /api/season/events/reward
- /api/season/history
- /api/season/status
- /api/stat-overlay
- /api/stat-overlay/leaderboard
- /api/stat-overlay/owner
- /api/stat-overlay/region
- /api/system-message
- /api/theme/bind
- /api/theme/lock
- /api/theme/vector
- /api/tournament/history
- /api/tournament/register
- /api/treasure/claim
- /api/treasure/spawn
- /api/underworld/contracts
- /api/update-rules
- /api/v1/redemption_gateway
- /api/vehicles
- /api/vehicles/spawn
- /api/world-content
- /api/world-content/create
- /api/world-content/deploy


### REST Drift by Category

| Category | Total | Wired | Orphaned | Coverage |
|----------|-------|-------|----------|----------|
| achievement | 1 | 0 | 1 | 0% |
| achievement-stats | 1 | 0 | 1 | 0% |
| achievements | 1 | 0 | 1 | 0% |
| ad | 6 | 1 | 5 | 17% |
| admin | 27 | 0 | 27 | 0% |
| ads | 3 | 1 | 2 | 33% |
| ai | 8 | 0 | 8 | 0% |
| assets | 5 | 0 | 5 | 0% |
| auctions | 1 | 0 | 1 | 0% |
| ban-player | 1 | 0 | 1 | 0% |
| black-market | 4 | 0 | 4 | 0% |
| bridge | 6 | 3 | 3 | 50% |
| card-details | 1 | 0 | 1 | 0% |
| card-stats | 1 | 0 | 1 | 0% |
| career | 1 | 0 | 1 | 0% |
| children-bots | 1 | 0 | 1 | 0% |
| church | 11 | 3 | 8 | 27% |
| claim | 1 | 0 | 1 | 0% |
| client-error | 1 | 0 | 1 | 0% |
| compliance | 6 | 1 | 5 | 17% |
| contracts | 2 | 0 | 2 | 0% |
| counterfeit | 2 | 0 | 2 | 0% |
| courthouse | 1 | 0 | 1 | 0% |
| creator | 18 | 9 | 9 | 50% |
| criminality | 1 | 0 | 1 | 0% |
| entity | 3 | 0 | 3 | 0% |
| entity-event | 2 | 2 | 0 | 100% |
| entity-events | 1 | 1 | 0 | 100% |
| events | 2 | 0 | 2 | 0% |
| faction | 1 | 0 | 1 | 0% |
| faith | 10 | 1 | 9 | 10% |
| faucet | 1 | 1 | 0 | 100% |
| governance | 7 | 1 | 6 | 14% |
| identity | 8 | 0 | 8 | 0% |
| industrial-loop | 3 | 0 | 3 | 0% |
| invest | 3 | 0 | 3 | 0% |
| items | 5 | 0 | 5 | 0% |
| justice | 6 | 3 | 3 | 50% |
| launch | 6 | 3 | 3 | 50% |
| launches | 1 | 0 | 1 | 0% |
| leaderboard | 1 | 0 | 1 | 0% |
| lease | 3 | 3 | 0 | 100% |
| loans | 3 | 0 | 3 | 0% |
| local-model | 2 | 0 | 2 | 0% |
| maintenance-mode | 1 | 0 | 1 | 0% |
| market | 1 | 0 | 1 | 0% |
| match | 1 | 0 | 1 | 0% |
| orphan | 3 | 0 | 3 | 0% |
| os | 5 | 3 | 2 | 60% |
| owner | 1 | 0 | 1 | 0% |
| pet-battle | 3 | 0 | 3 | 0% |
| pets | 3 | 0 | 3 | 0% |
| player | 1 | 0 | 1 | 0% |
| re-sync-stats | 1 | 0 | 1 | 0% |
| refill-vault | 1 | 0 | 1 | 0% |
| regions | 1 | 0 | 1 | 0% |
| replay | 7 | 0 | 7 | 0% |
| report-player | 1 | 0 | 1 | 0% |
| reset-stats | 1 | 0 | 1 | 0% |
| reward | 5 | 0 | 5 | 0% |
| rivalry | 10 | 0 | 10 | 0% |
| season | 8 | 0 | 8 | 0% |
| shares | 4 | 4 | 0 | 100% |
| stat-overlay | 4 | 0 | 4 | 0% |
| system-message | 1 | 0 | 1 | 0% |
| theme | 3 | 0 | 3 | 0% |
| tournament | 2 | 0 | 2 | 0% |
| treasure | 2 | 0 | 2 | 0% |
| underworld | 1 | 0 | 1 | 0% |
| update-rules | 1 | 0 | 1 | 0% |
| v1 | 1 | 0 | 1 | 0% |
| vehicles | 2 | 0 | 2 | 0% |
| world-content | 3 | 0 | 3 | 0% |


---

## WebSocket Drift Analysis

### Wired WS Message Types (51)

These backend message types are referenced in frontend JS:

- ✓ `abort_justice_mission`
- ✓ `abort_underworld_contract`
- ✓ `accept_justice_mission`
- ✓ `accept_underworld_contract`
- ✓ `alliance_accept`
- ✓ `alliance_dissolve`
- ✓ `alliance_invite`
- ✓ `aos_raid`
- ✓ `bail_card`
- ✓ `chat`
- ✓ `claim_dividends`
- ✓ `create_club`
- ✓ `create_lease`
- ✓ `freeze_dividends`
- ✓ `harvest_all_dividends`
- ✓ `heist`
- ✓ `initiate_recovery`
- ✓ `join_queue`
- ✓ `justice_flag_player`
- ✓ `kidnap_request`
- ✓ `launder_capital`
- ✓ `leave_queue`
- ✓ `link_wallet_request`
- ✓ `list_recovery_bounty`
- ✓ `loyalty_synthesis`
- ✓ `mood_recalibration`
- ✓ `move`
- ✓ `nonce_request`
- ✓ `open_regional_manager`
- ✓ `pay_ransom`
- ✓ `purchase_bounty_bond`
- ✓ `purchase_bounty_license`
- ✓ `purchase_item`
- ✓ `purchase_raid_insurance`
- ✓ `purchase_territory`
- ✓ `refresh_identity`
- ✓ `refund_bounty_bond`
- ✓ `regional_sabotage`
- ✓ `register_avatar`
- ✓ `release_hostage`
- ✓ `report_gloat`
- ✓ `set_district_tax`
- ✓ `spectate`
- ✓ `spread_rumor`
- ✓ `sync_request`
- ✓ `take_lease`
- ✓ `trade_shares`
- ✓ `update_rating`
- ✓ `use_item`
- ✓ `vault_donation`
- ✓ `vector_realignment`


### Orphaned WS Message Types (12)

These backend message types are NEVER referenced in frontend JS:

- ✗ `equip_cosmetic`
- ✗ `eth`
- ✗ `hire_player`
- ✗ `join_club`
- ✗ `purify_card`
- ✗ `redemption_gateway`
- ✗ `register_wallet`
- ✗ `restock_inventory`
- ✗ `sabotage`
- ✗ `sell_to_black_market`
- ✗ `set_salary`
- ✗ `sol`


### WS Drift by Category

| Category | Message Types | Wired | Orphaned | Coverage |
|----------|--------------|-------|----------|----------|
| Game Actions | 15 | 15 | 0 | 100% |
| Justice System | 10 | 10 | 0 | 100% |
| Underworld | 8 | 6 | 2 | 75% |
| Club/Alliance | 7 | 5 | 2 | 71% |
| Mutation Foundry | 3 | 3 | 0 | 100% |
| Identity | 3 | 2 | 1 | 67% |
| Match/Queue | 4 | 2 | 2 | 50% |
| Admin/Config | 5 | 3 | 2 | 60% |
| Other | 8 | 5 | 3 | 63% |

---

## Key Findings

### 1. REST API: Massive Drift (84.4% Orphaned)

The backend exposes 257 REST API routes but only 40 (15.6%) are wired to JS. This means:
- **217 routes** are completely dead code from the frontend perspective
- The largest orphaned categories: `admin` (27 routes), `ai/citizens` (8 routes), `faith` (9 routes), `creator` (9 routes), `identity` (8 routes), `replay` (7 routes), `season` (8 routes), `rivalry` (10 routes)

### 2. WebSocket: Healthy Coverage (81% Wired)

The WS layer is much better aligned:
- 51 of 63 message types are referenced in JS
- Only 12 orphaned types, mostly edge-case features (equip_cosmetic, hire_player, join_club, etc.)

### 3. JS-Referenced Endpoints NOT in Backend

The JS references several endpoints that don't exist in the backend:
- `/api/active-matches`
- `/api/auctions/bid`, `/api/auctions/create`
- `/api/bounty/active`
- `/api/creator/marketplace`, `/api/creator/products`
- `/api/creator/store/buy`
- `/api/envoi-name`
- `/api/justice/missions`
- `/api/pets/owner`
- `/api/player/profile`, `/api/player/tokens`
- `/api/players/constellation`
- `/api/rumors`
- `/api/shop/purchase`
- `/api/underworld/heists`, `/api/underworld/kidnaps`

These represent either: (a) planned but unimplemented backend routes, or (b) dead JS code.

### 4. Methodology Notes

- **REST extraction:** `grep -oP 'mux\.HandleFunc\("\K[^"]+' server_main.go` → 257 `/api/*` routes
- **WS extraction:** `awk 'NR>=950 && NR<=1672' lobby_manager.go | grep -oP 'case "\K[^"]+'` → 63 message types
- **JS cross-reference:** `grep -ohP 'type:\s*["'''][^"''']+'` for WS, `grep -ohP '/api/[^"''']+'` for REST
- **Coverage formula:** `wired / total * 100`

---

## Recommendations

1. **REST Cleanup:** Audit the 217 orphaned routes. Consider removing or implementing them.
2. **WS Completion:** Wire the 12 orphaned message types or remove them from the backend.
3. **JS Dead Code:** Remove references to non-existent backend endpoints.
4. **Documentation:** This drift matrix should be regenerated after major feature work.

---

*End of Drift Matrix*
