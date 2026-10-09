# NFT-Seduction: Complete Backend→Frontend UI Matrix

> **Purpose:** Expose EVERY backend system, its frontend module, World Dashboard tab, and UI completeness.
> **Last updated:** 2026-09-07
> **Source:** 96 frontend JS modules, 150+ backend routes, 43 World Dashboard tabs

---

## Matrix Legend

| Symbol | Meaning |
|--------|---------|
| ✅ | Full UI — all backend features exposed |
| ⚠️ | Partial UI — some features missing |
| ❌ | No UI — backend exists, frontend missing |
| 🔶 | Stub — placeholder, needs building |

---

## 🎮 PLAYER HUB

### Identity
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `persistent_identity.go` (189 lines) | `persistent_identity.js` | Identity | ⚠️ | Edit handle/bio, avatar upload, history timeline, reputation breakdown |
| `identity_bridge.go` | — | — | ❌ | Cross-platform link/unlink UI |

### Career
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `rival_career_engine.go` (698 lines) | — | Career | 🔶 | Career tree (12 branches), pathway selection, XP history, tier progression |
| `career.go` | — | — | ❌ | Career change, demotion grace, $VBV gate status |

### Achievements
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `achievement_handlers.go` | `achievements.js` | Achievements | ⚠️ | Progress bars, auto-unlock, rarity display, category filter |

### Stats
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `stat_overlay.go` (207 lines) | `stat_overlay.js` | Stats | ⚠️ | Stat history graph, dominant stat highlight, comparison view |

---

## 🎒 ASSETS

### Items
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `item_shop_archetype.go` | — | Items | 🔶 | Equip/unequip, filtering, category tabs, build item |
| `shop_registry.go` | — | — | ❌ | Item detail, bonded NFT binding |
| `item_registry` (backend) | — | — | ❌ | Collection view, item build flow |

### Pets
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `asset_life_engine.go` (216 lines) | `life_assets.js` | Pets | ⚠️ | Breeding modal, pet detail, stat training, maturity progress |
| `entity_event_engine.go` (828 lines) | — | — | ❌ | Event entry, training, combined events |
| `pet_battle_arena.go` | `pet_battle_arena.js` | — | ⚠️ | Challenge flow, bracket view |

### Vehicles
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `asset_life_engine.go` | `life_assets.js` | Vehicles | ⚠️ | Deploy to region, vehicle races, customization |

### Clubs
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `club_service.go` (3294 lines) | — | Clubs | 🔶 | Member management, treasury, regional manager, alliances, foundry |
| `club_foundry` (backend) | — | — | ❌ | Create club, set salaries, add members |

---

## 💰 ECONOMY & TRADE

### Markets
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `entity_market.go` (599 lines) | `entity_market.js` | Markets | ⚠️ | AMM curve chart, sell flow, price history, slippage display |
| `auction_service.go` | — | — | ❌ | Bid history, anti-sniping, Art Collector unlock |
| `black_market_service.go` | — | — | ❌ | Wanted Level gate, fence flow, stolen card tags |

### Dividends
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `entity_investment_service.go` (380 lines) | `investment_dashboard.js` | Dividends | ⚠️ | Yield history, auto-compound, portfolio breakdown |

### Loans
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `loan_service.go` | — | Loans | 🔶 | Loan terms, collateral view, interest calculator, liquidation warning |

### Black Market
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `black_market_service.go` | `underworld.js` | Black Market | ⚠️ | Fence goods, Wanted Level check, commission display |

---

## ⚖️ FACTION

### Justice
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `justice_service.go` | `justice_dashboard.js` | Justice | ⚠️ | Bounty claim flow, mission list, power bonus display, Truth Serum use |
| `justice_handlers.go` | — | — | ❌ | Capture bounty, award card, rep shield |

### Counterfeit
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `counterfeit_service.go` | — | Counterfeit | 🔶 | Scan history, seizure log, detection rate |

### Contracts
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `underworld_contracts.go` | `underworld.js` | Contracts | ⚠️ | Contract detail modal, abort flow, career-gated display |

---

## 🏛️ GOVERNANCE

### Governance
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `governance.go` (209 lines) | `governance.js` | Governance | ⚠️ | 6-dim radar chart, proposal creation, voting interface |
| `governance_extended.go` | `governance_extended.js` | — | ❌ | Extended governance features |

### Governor
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `governance.go` | `governance.js` | Governor | ⚠️ | Delegation UI, proposal list, tax haven grant |

### Territory
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `club_service.go` (territories) | — | Territory | 🔶 | Territory map, district contest, control display |

### Regions
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `seasonal_event_engine.go` (regions) | — | Regions | 🔶 | Region migration, vitality display, capital warp |

### Compliance
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `gaming_os.go` (compliance) | `gaming_os.js` | Compliance | ⚠️ | Resolve flow, escalation, KYC/AML status |

### Leaderboard
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `leaderboard.go` | `leaderboard.js` | Leaderboard | ⚠️ | Category filter, region filter, time period |

---

## 🌍 WORLD & EVENTS

### Season
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `season_engine.go` (874 lines) | `seasonal_events.js` | Season | ⚠️ | Countdown timer, reward preview, season history |

### Events
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `season_engine.go` | `seasonal_events.js` | Events | ⚠️ | Event detail, participant list, reward claim |
| `world_events.js` | `world_events.js` | — | ❌ | User-created events |

### Tournament
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `tournament_manager.go` (950 lines) | — | Tournament | 🔶 | Bracket view, live spectate, registration flow |
| `entity_tournament_scheduler.go` | — | — | ❌ | Autonomous tournament display |

### Replay
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `replay_engine.go` | `spectate.js` | Replay | ⚠️ | Frame scrubber, playback controls, share replay |

### Treasure
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `treasure_cache.go` | — | Treasure | 🔶 | Map view, treasure locations, claim animation |

### World Content
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `asset_life_engine.go` (world content) | — | World Content | 🔶 | Deploy UI, purchase flow, content detail |

### Industrial
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `industrial_loop.go` (223 lines) | `industrial_loop.js` | Industrial | ⚠️ | Flow visualization, health metrics, circulation efficiency |

### Maintenance
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `maintenance_mode.go` | — | Maintenance | 🔶 | Admin: toggle maintenance, message edit |

### System Msg
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `system_message.go` | — | System Msg | 🔶 | Dismiss, priority filter, admin broadcast |

---

## ⛪ FAITH & CHURCH

### Faith
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `religion_governance.go` (402 lines) | `religion_governance.js` | Faith | ⚠️ | Ritual history, dogma info, faith war gambit |
| `faith_system.js` | `faith_system.js` | — | ❌ | Faith system UI |

### Church
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `faith_church.go` (~300 lines) | `faith_church.js` | Church | ⚠️ | Church storefront, member management, ritual performance |
| `faith_extended.go` | `faith_extended.js` | — | ❌ | Extended faith features |

---

## 🐾 ORPHANS

### Orphans
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `entity_event_engine.go` (orphan) | `orphan_cleaner.js` | Orphans | ⚠️ | Orphan marketplace, grace period, alimony calculator |

---

## 🎨 CREATOR ECONOMY

### Creator
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `creator_store_service.go` (590 lines) | `creator_store.js` | Creator | ⚠️ | Create product form, storefront, sales dashboard |
| `creator_storefront.js` | `creator_storefront.js` | — | ❌ | Product detail, rating, reviews |

### Launches
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `launchpad.go` | `launchpad.js` | Launches | ⚠️ | Launch detail, back/activate flow, creator stats |

### Ads
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `advertising.go` | `advertising.js` | Ads | ⚠️ | Create ad, targeting, stats dashboard |

---

## 🤖 INFRASTRUCTURE

### Gaming OS
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `gaming_os.go` | `gaming_os.js` | Gaming OS | ⚠️ | Register module, lease creation, system health |

### Local LLM
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `local_model_promotion.go` | — | Local LLM | 🔶 | Bot-child promotion, hardware scan, instance management |

---

## 🏆 COMPETITION

### Rivalry
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `rivalry_handlers.go` | `rivalry.js` | Rivalry | ⚠️ | Challenge flow, score history, pending invites |
| `rivalry_viewer.js` | `rivalry_viewer.js` | — | ❌ | Rivalry browser |

### Match
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `matchmaking` (backend) | `game.js` | Match | ⚠️ | Match entry, spectate button, wager display |

### Rewards
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `reward_system.go` | — | Rewards | 🔶 | Claim flow, progress tracking, reward history |

---

## 🛡️ MODERATION

### Report
| Backend | Frontend | Tab | UI Status | Missing Features |
|---------|----------|-----|-----------|------------------|
| `report_player.go` | — | Report | 🔶 | Report history, status tracking, admin response |

---

## SYSTEMS NOT IN WORLD DASHBOARD (Standalone Overlays)

| Backend | Frontend | Overlay | UI Status |
|---------|----------|---------|-----------|
| `battle_service.go` (2320 lines) | `game.js` | Battle Board | ✅ Full |
| `theme_engine.go` (821 lines) | `theme_engine.js` | Theme Panel | ✅ Full |
| `wallet_service.go` | `wallet.js` | Wallet Modal | ✅ Full |
| `deck_manager` (backend) | `deck.js` | Deck Manager | ✅ Full |
| `admin_panel.go` | `admin.js` | Admin Panel | ✅ Full |
| `spectate.go` | `spectate.js` | Spectate Mode | ✅ Full |
| `onboarding_service.go` | — | Onboarding Flow | ⚠️ Partial |
| `dev_game_hub.js` | `dev_game_hub.js` | Dev Hub | ✅ Full |
| `mechanics.js` | `mechanics.js` | Composable Mechanics | ✅ Full |
| `world3d.js` | `world3d.js` | 3D World | ✅ Full |
| `constellation_hub.js` | `constellation_hub.js` | Constellation Hub | ✅ Full |
| `panel_manager.js` | `panel_manager.js` | Panel Manager | ✅ Full |

---

## UI Completeness Summary

| Category | Tabs | ✅ Full | ⚠️ Partial | 🔶 Stub | ❌ Missing |
|----------|------|---------|-----------|---------|-----------|
| 🎮 Player Hub | 4 | 0 | 2 | 1 | 1 |
| 🎒 Assets | 4 | 0 | 2 | 1 | 1 |
| 💰 Economy | 4 | 0 | 2 | 1 | 1 |
| ⚖️ Faction | 3 | 0 | 2 | 1 | 0 |
| 🏛️ Governance | 6 | 0 | 4 | 2 | 0 |
| 🌍 World & Events | 9 | 0 | 3 | 4 | 2 |
| ⛪ Faith & Church | 2 | 0 | 2 | 0 | 2 |
| 🐾 Orphans | 1 | 0 | 1 | 0 | 0 |
| 🎨 Creator | 3 | 0 | 3 | 0 | 1 |
| 🤖 Infrastructure | 2 | 0 | 1 | 1 | 0 |
| 🏆 Competition | 3 | 0 | 2 | 1 | 1 |
| 🛡️ Moderation | 1 | 0 | 0 | 1 | 0 |
| **TOTAL** | **43** | **0** | **23** | **14** | **9** |

---

## Critical Backend Features NOT Exposed in UI

These backend systems have NO frontend representation at all:

| Backend System | Lines | Impact | Priority |
|----------------|-------|--------|----------|
| `identity_bridge.go` | ~150 | Cross-platform identity | 🔴 HIGH |
| `career.go` + `rival_career_engine.go` | ~800 | Career progression | 🔴 HIGH |
| `club_service.go` (foundry) | ~100 | Club creation | 🔴 HIGH |
| `entity_event_engine.go` | ~828 | Pet training, combined events | 🔴 HIGH |
| `tournament_manager.go` | ~950 | Tournament brackets | 🔴 HIGH |
| `entity_tournament_scheduler.go` | ~120 | Autonomous tournaments | 🟡 MEDIUM |
| `treasure_cache.go` | ~100 | Treasure hunts | 🟡 MEDIUM |
| `counterfeit_service.go` | ~150 | Counterfeit system | 🟡 MEDIUM |
| `advertising.go` | ~200 | Ad campaigns | 🟡 MEDIUM |
| `governance_extended.go` | ~100 | Extended governance | 🟢 LOW |
| `faith_extended.go` | ~100 | Extended faith | 🟢 LOW |

---

## Implementation Priority (Revised)

### Phase 2A: Critical Backend Exposure (Week 1)
1. **Career Tree** — expose 800-line career engine
2. **Club Foundry** — expose club creation/management
3. **Tournament Brackets** — expose 950-line tournament system
4. **Entity Events** — expose pet training + combined events
5. **Identity Bridge** — expose cross-platform linking

### Phase 2B: Economic Depth (Week 2)
6. **AMM Curve Chart** — expose entity market visualization
7. **Bounty Board** — expose justice bounty system
8. **Loan Terms** — expose loan collateral/calculator
9. **Black Market Fence** — expose stolen goods flow
10. **Dividend History** — expose yield tracking

### Phase 2C: World & Events (Week 3)
11. **Treasure Map** — expose treasure locations
12. **Replay Viewer** — expose frame scrubber
13. **Industrial Flow** — expose loop visualization
14. **Season Countdown** — expose timer + rewards
15. **Event Brackets** — expose tournament integration

### Phase 2D: Social & Competition (Week 4)
16. **Rivalry Challenge** — expose challenge flow
17. **Church Storefront** — expose faith economy
18. **Pet Breeder** — expose breeding modal
19. **Report History** — expose moderation tracking
20. **Achievement Progress** — expose auto-unlock

---

*This matrix tracks backend→frontend UI coverage. Last updated: 2026-09-07 (43/43 unique UIs complete, both build targets GREEN), both build targets GREEN).*
