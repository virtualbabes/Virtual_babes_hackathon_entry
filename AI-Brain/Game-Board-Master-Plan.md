# Game Board & Card System — Master Plan

> **Last updated:** 2026-09-07
> **Status:** PLANNING — awaiting approval
> **Scope:** Game board UI, hand sets, card animations, rule sets, elemental/mood/faction/religious effects

---

## What Exists (Don't Rebuild)

| Component | Location | Status |
|-----------|----------|--------|
| 3x3 Grid Board | `game.js:35-45` | ✅ Working |
| Card Placement | `PlaceCard()` WASM bridge | ✅ Working |
| Basic Capture | `main.go:2405-2580` | ✅ Working (power comparison) |
| Rule Toggles | `simulateCapturesOnBoard()` | ✅ Backend rules: Power_copy, Power_up |
| Faction Scaling | `main.go:2452-2460` | ✅ JUSTICE/UNDERWORLD bonuses |
| Mood Modifiers | `getEffectivePower()` | ✅ Applied in calculations |
| Theme Engine | `theme_engine.go` | ✅ ThemeVector computed |
| AI Opponent | `main.go:2584-2810` | ✅ simulateCaptures for scoring |
| Multiplayer | `socket.send()` envelopes | ✅ Move sync works |

## What's Missing (Needs Build)

| Component | Effort | Priority |
|-----------|--------|----------|
| Game Board Screen (World Dashboard tab) | High | P0 |
| Hand Set / Deck Builder | High | P0 |
| Card Flip Animations (3D CSS) | Medium | P1 |
| Match Creation UI (rule toggles) | High | P0 |
| Elemental Affinity Display | Medium | P1 |
| Mood Board Tile Visuals | Medium | P1 |
| Religious/Faction Effect Panel | Medium | P2 |
| Capture Particle Effects | Low | P2 |
| Card Detail / Inspect Overlay | Medium | P2 |

---

## 1. Game Board Screen (P0)

### Concept
Dedicated World Dashboard tab replacing the inline battle overlay. Full-featured board with:
- Player vs Player / Player vs AI toggle
- Live power comparison display
- Turn indicator with timer
- Score tally (cards owned)

### Layout
```
┌─────────────────────────────────────────┐
│  [Opponent Hand - 5 cards face down]    │
├─────────────────────────────────────────┤
│  ┌───┐  ┌───┐  ┌───┐  │  Score: 3-2  │
│  │   │  │   │  │   │  │  Turn: Yours  │
│  └───┘  └───┘  └───┘  │               │
│  ┌───┐  ┌───┐  ┌───┐  │  [Power] [Mood│
│  │   │  │YOU│  │OPP│  │  [Element]    │
│  └───┘  └───┘  └───┘  │  [Faction]    │
│  ┌───┐  ┌───┐  ┌───┐  │               │
│  │   │  │   │  │   │  │  [Rule: Power │
│  └───┘  └───┘  └───┘  │   Copy: ON]   │
├─────────────────────────────────────────┤
│  [Your Hand - 5 cards face up]          │
├─────────────────────────────────────────┤
│  [Theme Lock] [QuickCast] [Forfeit]     │
└─────────────────────────────────────────┘
```

### New Files
- `Public/js/game_board.js` — Board orchestrator
- `Public/src/scss/features/_game_board.scss` — Board styling
- Wired through `app.js` as `initGameBoard()`

### Integration
- `world_dashboard.js` adds `game` tab → `window.initGameBoard()`
- Replaces inline `board-container` div with full screen
- Connects to existing `clickGrid()`, `PlaceCard()`, `syncUI()`

---

## 2. Hand Set / Deck Builder (P0)

### Concept
Pre-match deck construction screen. Player selects 5 cards from inventory:
- Card pool view (all owned cards)
- Drag-to-deck or click-to-add
- Deck validation (5 cards required)
- Deck rating display (sum of card power)
- Save multiple deck presets

### Layout
```
┌─────────────────────────────────────────┐
│  DECK BUILDER                           │
├─────────────────────┬───────────────────┤
│  YOUR CARDS         │  CURRENT DECK     │
│  ┌───┐ ┌───┐       │  ┌───┐ ┌───┐     │
│  │ C1│ │ C2│       │  │   │ │   │     │
│  └───┘ └───┘       │  └───┘ └───┘     │
│  ┌───┐ ┌───┐       │  ┌───┐ ┌───┐     │
│  │ C3│ │ C4│       │  │   │ │   │     │
│  └───┘ └───┘       │  └───┘ └───┘     │
│  ┌───┐ ┌───┐       │  ┌───┐           │
│  │ C5│ │ C6│       │  │   │           │
│  └───┘ └───┘       │  └───┘           │
├─────────────────────┴───────────────────┤
│  Deck Rating: 450    [Save] [Clear]     │
│  [Start Match with This Deck]           │
└─────────────────────────────────────────┘
```

### New Files
- `Public/js/deck_builder.js` — Deck builder orchestrator
- `Public/src/scss/features/_deck_builder.scss` — Builder styling
- Extends `deck.js` (existing card rendering)

### Integration
- `world_dashboard.js` adds `deck` tab → `window.initDeckBuilder()`
- Stores deck in `localStorage` + syncs to WASM
- Passes deck to `toggleMatchmakingQueue()`

---

## 3. Match Creation UI (P0)

### Concept
Pre-match lobby where players configure rules before queuing:
- Toggle rules ON/OFF (Power Copy, Plus, Same, Combo, etc.)
- Select game mode (Casual / Ranked / Tournament)
- Choose elemental theme (affects board tiles)
- Set mood modifiers (calm / wild / chaotic)
- Religious blessing selection (if applicable)
- Faction power toggle

### Rule Toggles (Backend Exists)

| Rule | Backend | Description |
|------|---------|-------------|
| Power Copy | `rules["Power_copy"]` | Same power = capture group |
| Plus | `rules["Power_up"]` | Sum powers = capture group |
| Same | `rules["Same"]` | Same element = bonus |
| Combo | `rules["Combo"]` | Chain flips allowed |
| Elements | `rules["Elements"]` | Elemental affinities active |
| Mood | `rules["Mood"]` | Mood modifiers active |
| Faction | `rules["Faction"]` | Faction powers active |
| Religious | `rules["Religious"]` | Religious blessings active |

### Layout
```
┌─────────────────────────────────────────┐
│  CREATE MATCH                           │
├─────────────────────────────────────────┤
│  Mode: [Casual] [Ranked] [Tournament]   │
├─────────────────────────────────────────┤
│  RULES                                  │
│  [✓] Power Copy  [✓] Plus  [✓] Same    │
│  [ ] Combo       [✓) Elements          │
│  [✓] Mood        [ ] Faction           │
│  [ ] Religious                          │
├─────────────────────────────────────────┤
│  ELEMENTAL THEME                        │
│  [Fire] [Water] [Earth] [Air] [Neutral] │
├─────────────────────────────────────────┤
│  MOOD MODIFIER                          │
│  [Calm] [Wild] [Chaotic]                │
├─────────────────────────────────────────┤
│  [Queue for Match]                      │
└─────────────────────────────────────────┘
```

### New Files
- `Public/js/match_creator.js` — Match config UI
- `Public/src/scss/features/_match_creator.scss` — Styling
- Backend: new route `/api/match/create` with config payload

### Integration
- `world_dashboard.js` adds `create_match` tab
- Payload sent to server: `{ rules, theme, mood, mode }`
- Server creates lobby with custom rules
- `toggleMatchmakingQueue()` reads config from UI

---

## 4. Card Flip Animations (P1)

### Concept
3D CSS flip animations when cards are captured:
- Smooth Y-axis rotation (0° → 180° → 360°)
- Color flash on flip (blue → red or red → blue)
- Scale pulse on capture
- Particle burst on combo flips

### Implementation
```css
/* Card flip animation */
.card-slot {
    perspective: 1000px;
    transition: transform 0.6s;
    transform-style: preserve-3d;
}

.card-slot.flipped {
    animation: cardFlip 0.6s ease-in-out;
}

@keyframes cardFlip {
    0% { transform: rotateY(0deg) scale(1); }
    50% { transform: rotateY(90deg) scale(1.1); }
    100% { transform: rotateY(180deg) scale(1); }
}

.card-front, .card-back {
    backface-visibility: hidden;
}

.card-back {
    transform: rotateY(180deg);
}
```

### Integration
- `game.js` adds class on capture detection
- `ui.js` `syncBoardParticles()` enhanced for flip events
- Optional: WebGL shader for glow effect

---

## 5. Elemental Affinity Display (P1)

### Concept
Visual display of card elemental types:
- Fire (red glow), Water (blue), Earth (green), Air (white), Neutral (gray)
- Board tiles have element → card on matching tile gets +50 power
- Mismatched element → -50 power
- Visual indicator on each tile (background color / icon)

### Backend
- `getEffectivePower()` already applies elemental sync
- Need: `/api/board/elements` returns tile elements per match

### UI Element
```
Board tile with element:
┌─────────┐
│  🔥     │  ← Element icon (small)
│ [CARD]  │
│  +50    │  ← Bonus indicator if card matches
└─────────┘
```

---

## 6. Mood Board Tile Visuals (P1)

### Concept
Board tiles have moods that affect card power:
- Calm tile: no modifier
- Wild tile: +random power
- Chaotic tile: power fluctuates per turn
- Visual: tile background pulses with mood color

### Backend
- `getEffectivePower()` checks `tile.Mood` vs `card.Mood`
- Mood match → +50, mismatch → -50

### UI
```css
.tile-calm { background: linear-gradient(#1a1a2e, #16213e); }
.tile-wild { background: linear-gradient(#2e1a1a, #3e1616); animation: pulse 2s infinite; }
.tile-chaotic { background: linear-gradient(#1a2e1a, #163e16); animation: flicker 0.5s infinite; }
```

---

## 7. Religious/Faction Effect Panel (P2)

### Concept
Shows active religious and faction effects during match:
- JUSTICE: +10% power vs Fallen/Wanted≥15 opponents
- UNDERWORLD: +10% power vs JUSTICE/Wanted≤2 opponents
- Religious blessing: +5% power if same religion as opponent's governor

### UI
```
┌─────────────────────────────┐
│  ACTIVE EFFECTS             │
│  ⚖️ JUSTICE: +10% vs Wanted│
│  🔥 Fire: +50 on fire tiles │
│  🙏 Devotion: +5% power     │
└─────────────────────────────┘
```

---

## 8. Capture Particle Effects (P2)

### Concept
Particle system enhanced for captures:
- Blue sparks → card becomes yours
- Red sparks → card flips to opponent
- Gold sparks → combo chain triggered
- Particle burst on game win/loss

### Integration
- `ui.js` `syncBoardParticles()` extended
- New function: `triggerCaptureParticles(gridIndex, owner)`

---

## 9. Card Detail Overlay (P2)

### Concept
Click card in hand or on board → popup with full details:
- Card name, element, mood
- Power values (N/E/S/W)
- Owner, faction, religious tag
- Active buffs/debuff
- Built item modifiers

### Layout
```
┌─────────────────────────────┐
│  [CARD ART]                 │
│  Name: Fire Knight         │
│  Element: 🔥 Fire          │
│  Mood: Aggressive          │
│  Power: N:100 E:80 S:60 W:90│
│  Owner: You (Justice)      │
│  Effects: +50 Fire tile    │
│            +10% vs Wanted   │
│  [Close]                    │
└─────────────────────────────┘
```

---

## Backend Changes Needed

| Route | Purpose | Status |
|-------|---------|--------|
| `POST /api/match/create` | Create custom match with rules | ❌ New |
| `GET /api/match/config` | Get available rules/options | ❌ New |
| `GET /api/board/elements` | Get tile elements for match | ❌ New |
| `POST /api/deck/save` | Save deck preset | ❌ New |
| `GET /api/deck/load` | Load saved decks | ❌ New |
| `GET /api/cards/inventory` | Get player's card pool | ❌ New |
| `WS match_config` | WebSocket: match rule changes | ❌ New |

---

## Implementation Order

| Phase | Component | Effort |
|-------|-----------|--------|
| 1 | Game Board Screen | 4-6 hours |
| 2 | Hand Set / Deck Builder | 3-4 hours |
| 3 | Match Creation UI | 4-5 hours |
| 4 | Card Flip Animations | 1-2 hours |
| 5 | Elemental Display | 1-2 hours |
| 6 | Mood Board Tiles | 1-2 hours |
| 7 | Religious/Faction Panel | 1-2 hours |
| 8 | Capture Particles | 1 hour |
| 9 | Card Detail Overlay | 1-2 hours |
| **Total** | | **17-26 hours** |

---

## Key Design Principles

1. **All through app.js orchestrator** — no standalone scripts
2. **Unique visual theme** per component (matches existing 43-tab design)
3. **Backend exists** — don't rebuild, just expose via UI
4. **Relative paths** in all code
5. **UINT64 on ledger** — no float math for power calculations
6. **Split-WASM** — server validates, client displays

---

## Approval Needed

- [ ] Game Board Screen design
- [ ] Deck Builder layout
- [ ] Match Creation rule toggles
- [ ] Elemental theme selection
- [ ] Mood modifier options
- [ ] Card flip animation style

---

*All plans ready for review. Backend rules engine exists — this is primarily UI work.*
