# UI Architecture Problems — Resolved & Current State

**Date:** 2026-09-06
**Investigator:** Seraph (Lead Systems Architect)
**Status:** ✅ All critical issues resolved

---

## Architecture Rule

> **All UI must be ES modules imported by `app.js`. No standalone `<script>` tags for new code. State lives in WASM, not DOM or localStorage.**

---

## ✅ Critical Issues — RESOLVED

| # | Problem | File | Fix Applied |
|---|---------|------|-------------|
| 1 | Missing file import | `game.js:5` | Created `collective-intelligence.js` |
| 2 | Malformed import identifier | `game.js:7` | Fixed to `initAudioContext` |
| 3 | Invalid class name | `audio_context.js:5` | Renamed to `AudioContextManager` |
| 4 | Invalid function names | `audio_context.js:136,147` | Renamed to `initAudioContextManager` / `getAudioContextManager` |

## ✅ Architecture Issues — RESOLVED

| # | Problem | File | Fix Applied |
|---|---------|------|-------------|
| 5 | IIFE with no exports | `rivalry.js` | Converted to ES module with `export const RivalryEngine` |
| 6 | Double-loaded script | `rivalry.js` | Removed standalone `<script>` tag from `index.html` |

## ✅ No Duplicates Found

- **Function definitions:** 0 duplicates across all 23 ES module files
- **Window bindings:** 0 duplicates — all 172 `window.*` bindings are unique to `app.js`

---

## 📁 Current File Organization

### ✅ ES Modules (23 files — all imported by app.js)

| Category | Files |
|----------|-------|
| **Core** | `app.js`, `config.js`, `utils.js` |
| **Game** | `game.js`, `deck.js`, `network.js`, `ui.js`, `audio.js`, `audio_context.js`, `particles.js` |
| **Economy** | `economy.js`, `wallet.js`, `admin.js`, `leaderboard.js` |
| **Social** | `criminality.js`, `rivalry.js`, `player_profile.js` |
| **Menu/Controller** | `menu-constellation.js`, `menu_customization.js`, `menu_customization_panel.js`, `controller_nav.js`, `user_preferences.js`, `pathway_avenues.js` |
| **AI** | `collective-intelligence.js` |

### ⚠️ Legacy IIFE Files (still loaded via index.html `<script>` tags)

These files work correctly but should be converted to ES modules for consistency:

| File | File | File |
|------|------|------|
| `world_dashboard.js` | `constellation_hub.js` | `constellation_spectate.js` |
| `constellation_tutorial.js` | `owner_stats.js` | `system_dashboard.js` |
| `admin_dashboard.js` | `community_dashboard.js` | `utilities_dashboard.js` |
| `daily_challenges.js` | `settings_panel.js` | `underworld.js` |
| `spectate.js` | `governance.js` | `industrial_loop.js` |
| `children_bots.js` | `entity_market.js` | `faith_church.js` |
| `pet_battle_arena.js` | `audio_engine.js` | `error_handler.js` |
| `wallet_state.js` | `wallet_modal.js` | `tx_modal.js` |
| `faucet_dashboard.js` | `theme_dashboard.js` | `placeholder_assets.js` |

---

## 🔧 WASM Bridge Functions (Menu/Controller)

| Function | Direction | Description |
|----------|-----------|-------------|
| `GetMenuState()` | JS ← WASM | Returns JSON snapshot |
| `SyncMenuState(json)` | JS → WASM | Ingest full menu state |
| `SetMenuLayout(json)` | JS → WASM | Update grid/shape/size config |
| `SetControllerType(type)` | JS → WASM | Set active controller |
| `SetControllerFocus(idx)` | JS → WASM | Set focus index |
| `HandleControllerInput(action)` | JS → WASM | Process input, returns new focus |
| `ToggleStarItem(json)` | JS → WASM | Add/remove favorite |
| `SetButtonOverrideWASM(id, json)` | JS → WASM | Per-button shape/size |

---

## 📋 Build Verification

| Check | Status |
|-------|--------|
| `npm run build` | ✅ Exit 0 |
| `GOOS=js GOARCH=wasm go build` | ✅ Exit 0 |
| `node --check` (23 ES modules) | ✅ Pass |
| Duplicate function definitions | ✅ None |
| Duplicate window bindings | ✅ None |
| Missing file imports | ✅ None |

---

## 🚀 How to Add New UI

See `AI-Brain/UI-Architecture-Plan.md` section 9 for the canonical build pattern:

1. Add field to `MenuState` in `main.go`
2. Add WASM bridge function in `menu_state.go` + register in `bridge_service.go`
3. Add JS function in appropriate ES module
4. Import + bind to `window` in `app.js`
5. Call via `window.yourFunction()` in HTML

---

*For the full architecture guide, see `AI-Brain/UI-Architecture-Plan.md`.*
