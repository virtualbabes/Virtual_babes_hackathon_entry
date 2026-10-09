# NFT-Seduction: Unfinished Work — Implementation Plan

> **Date:** 2026-09-07
> **Scope:** 4 items found during codebase exploration
> **Total Effort:** 6-9 hours

---

## 📋 Work Items Overview

| # | Item | File | Effort | Priority |
|---|------|------|--------|----------|
| 1 | Seasonal Events UI | `seasonal_events.js` | 2-3h | 🟢 Medium |
| 2 | Menu Customization TODOs | `menu_customization_panel.js` | 1-2h | 🟢 Medium |
| 3 | Placeholder Images | `economy.js` | 30min | 🟢 Low |
| 4 | Tournament Bracket View | `tournament_brackets.js` | 2-3h | 🟢 Medium |

---

## 1. Seasonal Events UI (2-3h)

### Current State
- `Public/js/seasonal_events.js` — **0 bytes** (empty file)
- World Dashboard tab `seasonal` exists but has no UI
- Backend `seasonEngine` exists with full functionality

### What to Build

#### UI Components
```
┌─────────────────────────────────────────┐
│  🎭 Seasonal Events                     │
│  ┌─────────────────────────────────┐   │
│  │  Active Season: Summer Festival │   │
│  │  Ends in: 3d 14h 22m            │   │
│  └─────────────────────────────────┘   │
│                                         │
│  📅 Current Events                      │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐  │
│  │ Event 1 │ │ Event 2 │ │ Event 3 │  │
│  │ Progress│ │ Progress│ │ Progress│  │
│  └─────────┘ └─────────┘ └─────────┘  │
│                                         │
│  🏆 Rewards Available                   │
│  ┌─────────────────────────────────┐   │
│  │ 🎁 Reward 1 — 500 SP           │   │
│  │ 🎁 Reward 2 — 1000 SP          │   │
│  └─────────────────────────────────┘   │
└─────────────────────────────────────────┘
```

#### Features
- Season countdown timer
- Active events with progress bars
- Reward claim buttons
- Event history
- Real-time WebSocket updates (seasonal_event_joined, seasonal_event_reward, etc.)

#### Files to Create/Modify
| File | Action | Lines |
|------|--------|-------|
| `Public/js/seasonal_events.js` | Create | ~200 |
| `Public/src/scss/features/_seasonal_events.scss` | Create | ~100 |
| `Public/src/scss/main.scss` | Add import | 1 |
| `Public/app.js` | Add initSeasonalEvents | 2 |
| `Public/js/world_dashboard.js` | Already wired | 0 |

#### Integration Points
- `seasonEngine` in `server.go`
- WebSocket events: `seasonal_event_joined`, `seasonal_event_created`, `seasonal_event_reward`, `seasonal_event_expired`, `seasonal_event_activated`
- API: `/api/season/status`, `/api/season/events`, `/api/season/claim`

---

## 2. Menu Customization TODOs (1-2h)

### Current State
- 6 empty stub functions in `menu_customization_panel.js`
- Controller tab and per-button tab not rendered

### Stubs to Implement

| Function | Line | What It Does |
|----------|------|--------------|
| `setControllerScheme(scheme)` | 397 | Apply controller scheme (gamepad/keyboard/tilt) |
| `testController()` | 398 | Test controller vibration/feedback |
| `selectPerButton(id)` | 399 | Select individual button for customization |
| `renderPerButtonList()` | 407 | Render list of buttons to customize |
| `renderControllerTab()` | 408 | Render controller settings tab |
| `renderGridDiagram()` | 409 | Render visual grid diagram |

### Implementation Plan

#### Controller Tab
```
┌─────────────────────────────────────────┐
│  🎮 Controller Settings                 │
│                                         │
│  Scheme: [Gamepad ▼] [Keyboard] [Tilt] │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │  Vibration: [ON] [OFF]          │   │
│  │  Sensitivity: [====●====]       │   │
│  │  Dead Zone: [●========]         │   │
│  └─────────────────────────────────┘   │
│                                         │
│  [Test Controller]                      │
└─────────────────────────────────────────┘
```

#### Per-Button Tab
```
┌─────────────────────────────────────────┐
│  🔘 Per-Button Customization            │
│                                         │
│  Buttons:                               │
│  ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐      │
│  │ 🌍  │ │ ⚔   │ │ ⚖   │ │ 💰  │      │
│  └─────┘ └─────┘ └─────┘ └─────┘      │
│                                         │
│  Selected: World                        │
│  Shape: [Circle ▼]  Size: [MD ▼]       │
│  Color: [●]  Icon: [🌍]                │
└─────────────────────────────────────────┘
```

#### Files to Modify
| File | Action | Lines |
|------|--------|-------|
| `Public/js/menu_customization_panel.js` | Implement stubs | ~100 |
| `Public/src/scss/features/_menu_customization_panel.scss` | Add styles | ~50 |

---

## 3. Placeholder Images (30min)

### Current State
- `economy.js` uses `PLACEHOLDER_IMG` (SVG with "?" character)
- Used for hardware and exhibit images

### What to Do
1. **Option A:** Generate simple colored gradient images with icons
2. **Option B:** Use emoji-based SVG data URIs (no external files needed)
3. **Option C:** Create `Public/Assets/` directory with actual images

### Implementation (Option B — Zero External Files)
```javascript
// Replace PLACEHOLDER_IMG with contextual placeholders
const PLACEHOLDERS = {
    hardware: "data:image/svg+xml,...", // Circuit board pattern
    exhibit: "data:image/svg+xml,...",   // Gallery frame pattern
    card: "data:image/svg+xml,...",      // Card back pattern
};
```

#### Files to Modify
| File | Action | Lines |
|------|--------|-------|
| `Public/js/economy.js` | Replace PLACEHOLDER_IMG | ~10 |

---

## 4. Tournament Bracket View (2-3h)

### Current State
- `tournamentViewBracket()` shows `alert('Full bracket view coming soon')`
- `renderBracketTree()` already exists but is basic
- Need full bracket overlay/modal

### What to Build

#### Full Bracket Modal
```
┌─────────────────────────────────────────────────────┐
│  🏆 Arena Championship — Full Bracket              │
│  ┌─────────────────────────────────────────────┐   │
│  │  Round 1    Quarter    Semi    Finals       │   │
│  │  ┌───┐      ┌───┐     ┌───┐    ┌───┐     │   │
│  │  │P1 │─────▶│   │────▶│   │───▶│   │     │   │
│  │  │P2 │      │   │     │   │    │   │     │   │
│  │  └───┘      └───┘     └───┘    └───┘     │   │
│  │  ┌───┐      ┌───┐     ┌───┐              │   │
│  │  │P3 │─────▶│   │────▶│   │              │   │
│  │  │P4 │      │   │     │   │              │   │
│  │  └───┘      └───┘     └───┘              │   │
│  └─────────────────────────────────────────────┘   │
│                                                     │
│  Match Details:                                     │
│  ┌─────────────────────────────────────────────┐   │
│  │ Player1 vs Player2 — Score: 5-3            │   │
│  │ Winner: Player1                             │   │
│  │ [Watch Replay]                              │   │
│  └─────────────────────────────────────────────┘   │
│                                                     │
│  [Close]                                            │
└─────────────────────────────────────────────────────┘
```

#### Features
- Interactive bracket tree (click to zoom)
- Match details panel
- Watch replay button
- Animated transitions
- Responsive layout

#### Files to Modify
| File | Action | Lines |
|------|--------|-------|
| `Public/js/tournament_brackets.js` | Implement full bracket modal | ~150 |
| `Public/src/scss/features/_tournament_brackets.scss` | Add modal styles | ~80 |

---

## 📊 Implementation Order

### Phase 1: Quick Wins (1h)
1. **Placeholder images** — 30min
2. **Menu Customization TODOs** — 30min

### Phase 2: Major Features (4-6h)
3. **Seasonal Events UI** — 2-3h
4. **Tournament Bracket View** — 2-3h

---

## 📁 File Summary

### Files to Create (2)
| File | Lines |
|------|-------|
| `Public/js/seasonal_events.js` | ~200 |
| `Public/src/scss/features/_seasonal_events.scss` | ~100 |

### Files to Modify (4)
| File | Lines |
|------|-------|
| `Public/js/menu_customization_panel.js` | ~100 |
| `Public/src/scss/features/_menu_customization_panel.scss` | ~50 |
| `Public/js/economy.js` | ~10 |
| `Public/js/tournament_brackets.js` | ~150 |

### Files to Update (3)
| File | Change |
|------|--------|
| `Public/src/scss/main.scss` | Add seasonal_events import |
| `Public/app.js` | Add initSeasonalEvents |
| `AI-Brain/ToDo.md` | Mark complete |

---

## ✅ Acceptance Criteria

- [ ] `seasonal_events.js` renders full UI with countdown, events, rewards
- [ ] All 6 menu customization stubs are functional
- [ ] No `PLACEHOLDER_IMG` in economy.js
- [ ] Tournament bracket opens as interactive modal (not alert)
- [ ] Both build targets remain GREEN
- [ ] Dev tools pass (53/53 + 22/22)

---

*Plan created 2026-09-07*
