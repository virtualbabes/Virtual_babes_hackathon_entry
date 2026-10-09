# NFT-Seduction: UI Architecture & Build Guide

> **Purpose:** Define the correct architecture for building UI features in this repo.
> **Last updated:** 2026-09-07
> **Critical:** All UI must go through the WASM bridge. Direct DOM manipulation outside the module system will fail.

---

## 1. Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│                    index.html                        │
│  <script src="wasm_exec.js"></script>               │
│  <script type="module" src="app.js"></script>       │
│  (NO other <script> tags for new code)              │
└─────────────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────┐
│                    app.js                            │
│  ← Central hub, imports ALL domain JS as ES modules │
│  ← Exposes functions to window.* bridge             │
│  ← Boots WASM engine (main.wasm)                    │
└─────────────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────┐
│              WASM Engine (main.wasm)                 │
│  ← Authoritative state machine (Engine struct)      │
│  ← Validates all state changes                      │
│  ← Pushes updates via window.syncUI()               │
└─────────────────────────────────────────────────────┘
```

### Key Rules

1. **ALL new JS files must be ES modules** — `import`/`export` syntax
2. **ALL new JS files must be imported by app.js** — no standalone `<script>` tags
3. **State lives in WASM** — read via `window.GetGameState(scope)`, write via `window.SyncX()` functions
4. **UI is read-only projection** — DOM renders state, user actions call WASM functions, WASM pushes updates back

---

## 2. File Organization

### Core Files

| File | Role |
|------|------|
| `Public/app.js` | Central hub — imports all domain JS, exposes to window bridge, boots WASM |
| `Public/index.html` | Loads only `wasm_exec.js` + `app.js` (module) |
| `main.go` (WASM) | Authoritative state machine — all state validation |
| `bridge_service.go` | WASM↔JS bridge function registry |
| `menu_state.go` | Menu/controller state management |

### Domain JS Files (All ES Modules)

| File | Responsibility |
|------|----------------|
| `ui.js` | Core UI functions (toasts, overlays, HUD, map) |
| `wallet.js` | Wallet connection, signing, balances |
| `network.js` | WebSocket, server sync |
| `game.js` | Battle mechanics, match state |
| `deck.js` | Deck management |
| `admin.js` | Admin controls |
| `economy.js` | Shops, markets, trading |
| `criminality.js` | Underworld, heists, kidnaps |
| `audio.js` | Sound effects, music |
| `audio_context.js` | Ambient audio transitions |
| `rivalry.js` | Rivalry system |
| `particles.js` | Visual effects |
| `utils.js` | Helpers (address shortening, asset resolution) |
| `config.js` | Deployment configuration |

### Menu & Controller Files (New — All ES Modules)

| File | Responsibility |
|------|----------------|
| `menu-constellation.js` | Main screen (World Dashboard + Quick Play + Profile + Favorites) |
| `menu_customization.js` | Shape/size/grid library (60+ shapes, 6 sizes, 9 grids) |
| `menu_customization_panel.js` | Customization UI overlay |
| `controller_nav.js` | Gamepad/keyboard/tilt navigation |
| `user_preferences.js` | Starred items, layout config, WASM sync |
| `pathway_avenues.js` | 12 career pathways, tiers, styling |

---

## 3. Data Flow

### Reading State

```javascript
// Get authoritative state from WASM
const state = window.GetGameState("all");      // Full state
const combat = window.GetGameState("combat");  // Combat only
const menu = window.GetMenuState();            // Menu/controller state (JSON)
```

### Writing State

```javascript
// Push state changes to WASM
window.SyncFullProfile(profileData);
window.SetPhase("Active");
window.SyncVaultBalance(amount);
window.SetMenuLayout(JSON.stringify(layout));
window.ToggleStarItem(JSON.stringify(starredItem));
window.HandleControllerInput("right");  // Returns new focus index
```

### Beacon Persistence (Warm Boot)

```javascript
// app.js:193-226 restores state from localStorage beacon
const cachedBeacon = localStorage.getItem("vbabes_state_beacon");
if (cachedBeacon) {
    const beacon = JSON.parse(cachedBeacon);
    if (window.SyncFullProfile) window.SyncFullProfile(beacon.profile);
    if (window.SyncVaultBalance) window.SyncVaultBalance(beacon.vault_balance);
    if (window.SyncRewards) window.SyncRewards(beacon.rewards);
    if (window.SyncClubs) window.SyncClubs(beacon.clubs);
    // TODO: Add window.SyncMenuState(beacon.menu_state)
}
```

---

## 4. Menu & Controller System (NEW — 2026-09-06)

### Architecture

```
┌─────────────────────────────────────────────────────┐
│                 menu-constellation.js                │
│  ← Main screen: World Dash + Quick Play + Profile   │
│  ← Renders favorites grid from UserPreferences       │
│  ← Imports from menu_customization.js                │
│  ← Imports from user_preferences.js                  │
└─────────────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────┐
│              menu_customization.js                   │
│  ← Shape library: 60+ clip-path shapes              │
│  ← Size tiers: xs/sm/md/lg/xl/xxl                   │
│  ← Grid types: grid/circle/triangle/cross/arc/...   │
│  ← computePositions() — layout calculator            │
│  ← getClipPath() — CSS clip-path for shape           │
│  ← getShapeForSystem() — default shape per system    │
└─────────────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────┐
│           menu_customization_panel.js                │
│  ← 6-tab overlay: Shape/Size/Grid/Controller/...    │
│  ← Live preview rendering                            │
│  ← Save/Load/Reset/Export/Import                     │
└─────────────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────┐
│               controller_nav.js                      │
│  ← Gamepad API (Xbox/PS/Nintendo/Generic)           │
│  ← Keyboard fallback (Arrows/WASD)                   │
│  ← Android tilt (DeviceOrientationEvent)             │
│  ← Haptic feedback                                   │
│  ← Grid-aware navigation (different per grid type)   │
│  ← WASM sync: HandleControllerInput() → new focus    │
└─────────────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────┐
│              user_preferences.js                     │
│  ← localStorage persistence (vbabes_user_prefs_v1)   │
│  ← Starred items (favorites)                         │
│  ← Menu layout config (gridType, shape, size, etc.)  │
│  ← Per-button overrides                              │
│  ← WASM sync: syncToWasm() / syncFromWasm()          │
└─────────────────────────────────────────────────────┘
```

### MenuState (WASM Struct)

```go
type MenuState struct {
    GridType        string            // grid/circle/triangle/cross/arc/diamond/linear-h/linear-v/freeform
    DefaultShape    string            // circle/star/hexagon/etc
    DefaultSize     string            // xs/sm/md/lg/xl/xxl
    GridCols        int               // Number of columns (grid type)
    GridGap         string            // CSS gap value
    GridRadius      string            // Radius for circle/arc types
    GridRotation    string            // Rotation for circle/arc types
    StarredItems    []StarredItem     // User-curated favorites
    ButtonOverrides map[string]ButtonOverride // Per-button shape/size
    ControllerType  string            // xbox/playstation/nintendo/keyboard/touch_tilt
    FocusIndex      int               // Current controller focus position
    ActivePathway   string            // P-Shadow/P-Justice/etc for theme
    QuickPlayVisible bool
    TutorialVisible bool
}
```

### WASM Bridge Functions (Menu/Controller)

| Function | Direction | Description |
|----------|-----------|-------------|
| `GetMenuState()` | JS ← WASM | Returns JSON snapshot of menu state |
| `SyncMenuState(json)` | JS → WASM | Ingest full menu state |
| `SetMenuLayout(json)` | JS → WASM | Update grid/shape/size config |
| `SetControllerType(type)` | JS → WASM | Set active controller |
| `SetControllerFocus(idx)` | JS → WASM | Set focus index |
| `HandleControllerInput(action)` | JS → WASM | Process input, returns new focus |
| `ToggleStarItem(json)` | JS → WASM | Add/remove favorite |
| `SetButtonOverrideWASM(id, json)` | JS → WASM | Per-button shape/size |

---

## 5. Navigation Architecture

```
┌─────────────────────────────────────────────┐
│              MAIN MENU                       │
│  (Permanent: World Dash + Quick Play + Hub) │
│  [★ Favorites Grid — user-curated]          │
│  [NPC Helpers: Anya, Vbabes, Crypto-Seraph] │
└─────────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────────┐
│         WORLD DASHBOARD                      │
│  (12 categories, 43 tabs, ☆ starring)       │
│  User stars tabs → they appear as favorites  │
└─────────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────────┐
│       CONSTELLATION HUB (Living Nexus)       │
│  (Starred items as nodes + Pathway Avenues) │
│  Nodes = favorited features + unlock states  │
└─────────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────────┐
│         PATHWAY AVENUE UNLOCKS               │
│  (12 unique stylings — one per career path) │
│  Each avenue = tiered progression tree       │
└─────────────────────────────────────────────┘
```

---

## 6. The 12 World Dashboard Categories

| # | Icon | Name | Tab Count | Tab IDs |
|---|------|------|-----------|---------|
| 1 | 🎮 | Player Hub | 4 | identity, career, achievements, stats |
| 2 | 🎒 | Assets | 4 | items, pets, vehicles, clubs |
| 3 | 💰 | Economy & Trade | 4 | markets, dividends, loans, blackmarket |
| 4 | ⚖️ | Faction | 3 | justice, counterfeit, contracts |
| 5 | 🏛️ | Governance | 6 | governance, governor, territory, regions, compliance, leaderboard |
| 6 | 🌍 | World & Events | 9 | season, events, tournament, replay, treasure, worldcontent, industrial, maintenance, systemmsg |
| 7 | ⛪ | Faith & Church | 2 | faith, church |
| 8 | 🐾 | Orphans | 1 | orphan |
| 9 | 🎨 | Creator Economy | 3 | creator, launches, ads |
| 10 | 🤖 | Infrastructure | 2 | gamingos, localmodel |
| 11 | 🏆 | Competition | 3 | rivalry, match, rewards |
| 12 | 🛡️ | Moderation | 1 | report |

---

## 7. The 12 Career Pathways

| ID | Name | Domain | Faction | Color |
|----|------|--------|---------|-------|
| P-Shadow | Shadow Ops | Gossip, Justice Recruiter | JUSTICE | `#4a148c` |
| P-Lockdown | Lockdown | Kidnapper, AOS, Warden | JUSTICE | `#b71c1c` |
| P-Ledger | Ledger | Launderer, Mutation Auditor | UNDERWORLD | `#1b5e20` |
| P-Syndicate | Syndicate | Underworld Boss, Judge | UNDERWORLD | `#ff6f00` |
| P-Tax | Tax | Tax Auditor, Launderer | HYBRID | `#f57f17` |
| P-Peace | Peace | Sector Peacekeeper, Smuggler | JUSTICE | `#0d47a1` |
| P-Intel | Intel | Intel Agent, ArcNet Operative | JUSTICE | `#006064` |
| P-Justice | Justice | Justice Recruiter, Bounty Hunter | JUSTICE | `#1a237e` |
| P-AOS | AOS | AOS, Sector Peacekeeper | JUSTICE | `#33691e` |
| P-Commissioner | Commissioner | Justice Commissioner, Tax Auditor | HYBRID | `#880e4f` |
| P-Forensic | Forensic | Mutation Auditor, Launderer | UNDERWORLD | `#3e2723` |
| P-Boss | Boss | Underworld Boss, Judge | UNDERWORLD | `#263238` |

### Avenue Tier Structure

| Tier | Name | Requirement | Unlocks |
|------|------|-------------|---------|
| 1 | Initiate | Career selection | Base styling, tier-1 card pool |
| 2 | Apprentice | 100 XP + 1 achievement | +5% pathway bonus, tier-2 items |
| 3 | Journeyman | 500 XP + 3 achievements | +10% pathway bonus, unique ritual/contract |
| 4 | Expert | 2000 XP + 7 achievements | +15% pathway bonus, tier-4 card pool |
| 5 | Master | 10000 XP + 15 achievements | +20% pathway bonus, Governor/Boss eligibility |

---

## 8. Build Status (2026-09-07)

### ✅ Done

| Component | Files | Status |
|-----------|-------|--------|
| WASM MenuState struct | `main.go` | ✅ Implemented |
| WASM bridge functions | `bridge_service.go`, `menu_state.go` | ✅ 8 functions |
| Main screen | `menu-constellation.js` | ✅ Complete |
| Shape/size/grid library | `menu_customization.js` | ✅ 60+ shapes, 6 sizes, 9 grids |
| Customization panel | `menu_customization_panel.js` | ✅ 6-tab overlay |
| Controller navigation | `controller_nav.js` | ✅ Gamepad/keyboard/tilt |
| User preferences | `user_preferences.js` | ✅ localStorage + WASM sync |
| Pathway avenues | `pathway_avenues.js` | ✅ 12 pathways, 5 tiers |
| App.js bridge | `app.js` | ✅ 45+ window.* bindings |

### ⏳ Pending

| Component | Description | Priority |
|-----------|-------------|----------|
| Per-pathway avenue screens | Each career path gets unique UI | High |
| 3D world integration | Pathway color → region tint in Three.js | Medium |
| Beacon menu persistence | Restore menu state on warm boot | Medium |
| Controller main menu fallback | Navigate when favorites grid is empty | Low |
| `/api/player/progression` sync | Live tier calculation from server | Low |

---

## 9. How to Build a New UI Feature

### Step 1: Add to MenuState (if menu-related)

Edit `main.go`:
```go
type MenuState struct {
    // ... existing fields ...
    YourNewField string `json:"your_new_field"`
}
```

Initialize in `Game` var:
```go
MenuState: MenuState{
    // ... existing defaults ...
    YourNewField: "default_value",
},
```

### Step 2: Add WASM Bridge Function

Edit `menu_state.go`:
```go
func SetYourNewField(this js.Value, args []js.Value) interface{} {
    Game.mutex.Lock()
    defer Game.mutex.Unlock()
    Game.MenuState.YourNewField = args[0].String()
    return true
}
```

Register in `bridge_service.go`:
```go
js.Global().Set("SetYourNewField", js.FuncOf(SetYourNewField))
```

### Step 3: Add to JS Module

Create or edit the appropriate ES module file:
```javascript
// js/your_module.js
import { getMenuLayout, setMenuLayout } from './user_preferences.js';

export function yourNewFunction() {
    const state = window.GetMenuState();
    // ... your logic ...
    window.SetYourNewField(newValue);
}
```

### Step 4: Import in app.js

```javascript
import { yourNewFunction } from './js/your_module.js';
window.yourNewFunction = yourNewFunction;
```

### Step 5: Use in index.html (if needed)

```html
<!-- Call via window bridge — no new script tags -->
<button onclick="window.yourNewFunction()">Do Thing</button>
```

---

## 10. Common Pitfalls

| Pitfall | Wrong | Right |
|---------|-------|-------|
| Loading JS | `<script src="js/myscript.js">` | `import { x } from './js/myscript.js'` in app.js |
| Reading state | `localStorage.getItem('gameState')` | `window.GetGameState("all")` |
| Writing state | `window.myVar = value` | `window.SetX(value)` WASM function |
| Exposing function | `window.myFunc = function() {}` in IIFE | `export function myFunc() {}` + bind in app.js |
| Storing secrets | `localStorage.setItem('privateKey', key)` | Never store private keys in localStorage |
| Float math | `balance / 1000000` for ledger | Use integer math (micro-units) for all ledger values |

---

## 11. Build & Verify

```bash
# Full build (WASM + SCSS + server)
npm run build

# WASM only
GOOS=js GOARCH=wasm go build -ldflags="-w -s" -o Public/main.wasm .

# Syntax check
node --check Public/js/your_module.js

# Server build
CGO_ENABLED=0 go build -ldflags="-w -s" -o server-bin.exe .
```

---

*This document is the authoritative reference for UI architecture. All future UI work must follow these patterns.*
