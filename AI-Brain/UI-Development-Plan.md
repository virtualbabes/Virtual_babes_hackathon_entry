# NFT-Seduction: UI Development Plan — Category Tree Audit

> **Purpose:** Map every World Dashboard category to its current UI state and define what needs building.
> **Last updated:** 2026-09-07
> **Source:** `Public/js/world_dashboard.js` (12 categories, 43 tabs)

---

## Current UI Architecture

```
MAIN MENU (menu-constellation.js)
├── 🌍 World Dashboard (permanent button)
├── ⚡ Quick Play
├── 👤 Profile
├── 📚 Tutorial
├── ★ Favorites Grid (user-curated from starred tabs)
└── 🎨 Customize Menu button
    │
    ▼
WORLD DASHBOARD (world_dashboard.js)
├── 🎮 Player Hub
│   ├── Identity
│   ├── Career
│   ├── Achievements
│   └── Stats
├── 🎒 Assets
│   ├── Items
│   ├── Pets
│   ├── Vehicles
│   └── Clubs
├── 💰 Economy & Trade
│   ├── Markets
│   ├── Dividends
│   ├── Loans
│   └── Black Market
├── ⚖️ Faction
│   ├── Justice
│   ├── Counterfeit
│   └── Contracts
├── 🏛️ Governance
│   ├── Governance
│   ├── Governor
│   ├── Territory
│   ├── Regions
│   ├── Compliance
│   └── Leaderboard
├── 🌍 World & Events
│   ├── Season
│   ├── Events
│   ├── Tournament
│   ├── Replay
│   ├── Treasure
│   ├── World Content
│   ├── Industrial
│   ├── Maintenance
│   └── System Msg
├── ⛪ Faith & Church
│   ├── Faith
│   └── Church
├── 🐾 Orphans
│   └── Orphans
├── 🎨 Creator Economy
│   ├── Creator
│   ├── Launches
│   └── Ads
├── 🤖 Infrastructure
│   ├── Gaming OS
│   └── Local LLM
├── 🏆 Competition
│   ├── Rivalry
│   ├── Match
│   └── Rewards
└── 🛡️ Moderation
    └── Report
```

---

## Per-Category UI Audit

### 🎮 PLAYER HUB (4 tabs)

| Tab | Current UI | Status | Needs |
|-----|------------|--------|-------|
| Identity | Static display (handle, reputation, social rank) | ⚠️ Read-only | Make editable, add avatar |
| Career | Static display (path, level, XP) | ⚠️ Read-only | Add career tree visualization |
| Achievements | List with unlock buttons | ⚠️ Manual unlock only | Add progress tracking, auto-unlock |
| Stats | Static display (power, effective, sum) | ⚠️ Read-only | Add stat history graph |

**Unique Needs:**
- Career tree with 12 pathway branches
- Achievement progress bars
- Identity editor (handle, bio, avatar)

---

### 🎒 ASSETS (4 tabs)

| Tab | Current UI | Status | Needs |
|-----|------------|--------|-------|
| Items | Archetype list | ⚠️ Display only | Add equip/unequip, filtering |
| Pets | List (name, element, level) | ⚠️ Basic | Add pet detail modal, breeding UI |
| Vehicles | List (name, status) | ⚠️ Basic | Add deploy to region UI |
| Clubs | Club list + "Found a Club" button | ⚠️ Basic | Add club detail, member management |

**Unique Needs:**
- Pet breeding modal (select sire/dam)
- Vehicle deployment map
- Club member management interface
- Item equip slots

---

### 💰 ECONOMY & TRADE (4 tabs)

| Tab | Current UI | Status | Needs |
|-----|------------|--------|-------|
| Markets | Listings, auctions, black market | ⚠️ Basic buy | Add chart, sell flow, price history |
| Dividends | Claim button | ⚠️ Basic | Add yield history, auto-compound |
| Loans | Take/repay buttons | ⚠️ Basic | Add loan terms, collateral view |
| Black Market | Item list | ⚠️ Basic | Add Wanted Level gate, fence flow |

**Unique Needs:**
- AMM curve visualization
- Share price charts
- Loan collateral calculator
- Fence commission display

---

### ⚖️ FACTION (3 tabs)

| Tab | Current UI | Status | Needs |
|-----|------------|--------|-------|
| Justice | Power bonus, tier, bounties count | ⚠️ Read-only | Add bounty claim flow, mission list |
| Counterfeit | Detection count, generate button | ⚠️ Basic | Add scan history, seizure log |
| Contracts | List with accept buttons | ⚠️ Basic | Add contract detail, abort flow |

**Unique Needs:**
- Bounty board with claim flow
- Justice mission tracker
- Contract detail modal with terms
- Counterfeit scan animation

---

### 🏛️ GOVERNANCE (6 tabs)

| Tab | Current UI | Status | Needs |
|-----|------------|--------|-------|
| Governance | 6-dim score | ⚠️ Read-only | Add proposal creation, voting |
| Governor | Voting power, delegated | ⚠️ Read-only | Add delegation UI, proposal list |
| Territory | Region list | ⚠️ Basic | Add contest flow, district view |
| Regions | Region status | ⚠️ Basic | Add migration UI |
| Compliance | Records list | ⚠️ Basic | Add resolve flow, escalation |
| Leaderboard | Top 10 players | ⚠️ Read-only | Add filtering, categories |

**Unique Needs:**
- 6-dimension radar chart
- Election voting interface
- Territory map with districts
- Proposal creation form

---

### 🌍 WORLD & EVENTS (9 tabs)

| Tab | Current UI | Status | Needs |
|-----|------------|--------|-------|
| Season | Season info, end date | ⚠️ Read-only | Add reward preview, countdown |
| Events | List with join button | ⚠️ Basic | Add event detail, participant list |
| Tournament | History, register prompt | ⚠️ Basic | Add bracket view, live spectate |
| Replay | Frame count, match ID | ⚠️ Basic | Add replay viewer |
| Treasure | Claim button | ⚠️ Basic | Add map view, treasure locations |
| World Content | Content list | ⚠️ Basic | Add deploy UI, purchase flow |
| Industrial | Metrics (businesses, employees, treasury) | ⚠️ Read-only | Add flow visualization |
| Maintenance | Status message | ⚠️ Read-only | Admin: toggle maintenance |
| System Msg | Message list | ⚠️ Read-only | Add dismiss, priority filter |

**Unique Needs:**
- Season countdown timer
- Event bracket visualization
- Replay viewer with frame scrubber
- Treasure map with locations
- Industrial loop flow diagram

---

### ⛪ FAITH & CHURCH (2 tabs)

| Tab | Current UI | Status | Needs |
|-----|------------|--------|-------|
| Faith | Coherence score | ⚠️ Read-only | Add ritual history, dogma info |
| Church | Church list, income | ⚠️ Basic | Add church detail, member management |

**Unique Needs:**
- Church storefront UI
- Ritual performance interface
- Faith war gambit stake UI
- Religious leader card display

---

### 🐾 ORPHANS (1 tab)

| Tab | Current UI | Status | Needs |
|-----|------------|--------|-------|
| Orphans | Adopt/reclaim buttons with prompt | ⚠️ Basic | Add orphan list, grace period display |

**Unique Needs:**
- Orphan marketplace
- Grace period countdown
- Alimony calculator

---

### 🎨 CREATOR ECONOMY (3 tabs)

| Tab | Current UI | Status | Needs |
|-----|------------|--------|-------|
| Creator | Product list | ⚠️ Basic | Add create product form, storefront |
| Launches | Launch list | ⚠️ Basic | Add launch detail, back/activate |
| Ads | Ad list | ⚠️ Basic | Add create ad, stats view |

**Unique Needs:**
- Product creation form
- Launch campaign creator
- Ad performance dashboard
- Royalty history view

---

### 🤖 INFRASTRUCTURE (2 tabs)

| Tab | Current UI | Status | Needs |
|-----|------------|--------|-------|
| Gaming OS | Module list | ⚠️ Read-only | Add register module, lease UI |
| Local LLM | Tier, instances, backend | ⚠️ Basic | Add promote bot-child, hardware scan |

**Unique Needs:**
- Bot-child promotion interface
- Hardware scan results display
- Module lease creation form

---

### 🏆 COMPETITION (3 tabs)

| Tab | Current UI | Status | Needs |
|-----|------------|--------|-------|
| Rivalry | Rivalry list (side_a vs side_b) | ⚠️ Basic | Add challenge flow, score history |
| Match | Active matches, wager min | ⚠️ Basic | Add match entry, spectate button |
| Rewards | Reward list | ⚠️ Basic | Add claim flow, progress |

**Unique Needs:**
- Rivalry challenge interface
- Match spectate integration
- Reward claim flow

---

### 🛡️ MODERATION (1 tab)

| Tab | Current UI | Status | Needs |
|-----|------------|--------|-------|
| Report | Input + submit button | ⚠️ Basic | Add report history, status tracking |

**Unique Needs:**
- Report history list
- Admin response display

---

## Priority Matrix (by user impact + effort)

### 🔴 HIGH PRIORITY (High Impact, Low-Medium Effort)

| # | Feature | Category | Effort | Impact |
|---|---------|----------|--------|--------|
| 1 | Career tree visualization | Player Hub | Medium | Core identity |
| 2 | Bounty claim flow | Faction | Medium | Daily loop |
| 3 | Event bracket view | World & Events | Medium | Engagement |
| 4 | Church storefront | Faith & Church | Medium | Faith economy |
| 5 | Territory map | Governance | Medium | Governance loop |
| 6 | Pet breeding modal | Assets | Low | Pet economy |
| 7 | Loan terms view | Economy | Low | Financial depth |
| 8 | Rivalry challenge flow | Competition | Medium | Social loop |

### 🟡 MEDIUM PRIORITY (High Impact, High Effort)

| # | Feature | Category | Effort | Impact |
|---|---------|----------|--------|--------|
| 9 | Replay viewer | World & Events | High | Retention |
| 10 | Treasure map | World & Events | High | Exploration |
| 11 | AMM curve chart | Economy | High | Market depth |
| 12 | Industrial loop diagram | World & Events | High | Economic clarity |
| 13 | Election voting interface | Governance | High | Governance depth |
| 14 | 3D World integration | All | Very High | Vision fulfillment |

### 🟢 LOW PRIORITY (Low Impact, Low Effort)

| # | Feature | Category | Effort | Impact |
|---|---------|----------|--------|--------|
| 15 | Achievement progress bars | Player Hub | Low | Progress clarity |
| 16 | Identity editor | Player Hub | Low | Personalization |
| 17 | Maintenance toggle | World & Events | Low | Admin convenience |
| 18 | Report history | Moderation | Low | Trust |

---

## UI Component Library Needed

| Component | Used By | Status |
|-----------|---------|--------|
| **Stat Card** | Many tabs | ✅ Exists (`.wd-stat`) |
| **Action Button** | Many tabs | ✅ Exists (`.vbt-btn`) |
| **Grid Layout** | Many tabs | ✅ Exists (`.wd-grid`) |
| **Modal Overlay** | Many tabs | ✅ Exists (`.overlay`) |
| **Progress Bar** | Career, Achievements | ❌ Need to build |
| **Radar Chart** | Governance 6-dim | ❌ Need to build |
| **Bracket View** | Tournaments | ❌ Need to build |
| **Flow Diagram** | Industrial Loop | ❌ Need to build |
| **Line Chart** | Markets, Stats | ❌ Need to build |
| **Treasure Map** | World & Events | ❌ Need to build |
| **Bounty Board** | Faction | ❌ Need to build |
| **Pet Breeder** | Assets | ❌ Need to build |

---

## Implementation Phases

### Phase 1: Core Navigation (DONE ✅)
- World Dashboard with 12 categories
- Main Menu with favorites grid
- Customization panel (shapes, sizes, grids)
- Controller navigation

### Phase 2: Data-Rich Tabs (NEXT)
- Career tree with pathway branches
- Bounty board with claim flow
- Event brackets with live spectate
- Church storefront with ritual UI
- Territory map with district contest

### Phase 3: Advanced Visualizations
- AMM curve charts
- Industrial loop flow diagram
- Replay viewer with frame scrubber
- Treasure map with locations
- 3D World region rendering

### Phase 4: Social + Competition
- Rivalry challenge flow
- Report history + admin response
- Achievement progress tracking
- Leaderboard filtering

---

*Next: Build Phase 2 components — starting with Career Tree, Bounty Board, and Event Brackets.*
