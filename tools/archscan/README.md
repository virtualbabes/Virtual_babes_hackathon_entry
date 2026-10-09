# Architecture Scanner — Developer Manual

**Location:** `tools/archscan/`
**Version:** 2.0.0
**Last Updated:** 2026-09-07

---

## Overview

The Architecture Scanner is an autonomous flow detection system that:
- Scans the codebase for Go bridge functions, JS modules, HTML scripts, and CSS partials
- Detects wired vs orphaned files using `.gitignore` rules (with negation support)
- Generates interactive architecture diagrams with analytics
- Tracks flow integrity (registration ↔ handler pairing)
- Computes architecture health from wired ratio + flow integrity + connectedness + violations
- Watches for changes and updates manifests automatically

---

## Tools

### 1. `scan.js` — Architecture Bridge Scanner

**Purpose:** Scans Go bridge functions, JS ES modules, HTML scripts, and CSS partials.

**Usage:**
```bash
node tools/archscan/scan.js
```

**Output:** `tools/archscan/manifest.json`

**Detects:**
- `js.Global().Set("FuncName", ...)` registrations in Go files
- `func Xxx(this js.Value...)` handler implementations
- `export function/class` in JS files
- `import ... from ...` in JS files
- `window.X = ...` bindings in JS files
- `<script src="...">` tags in HTML
- SCSS partials in `Public/src/scss/features/`
- Build tag violations
- **Flow integrity** — pairs registrations with their handlers

**Manifest Structure:**
```json
{
  "timestamp": "2026-09-07T...",
  "bridgeFunctions": [
    {
      "name": "GetGameState",
      "file": "bridge_service.go",
      "registeredIn": "bridge_service.go",
      "implementedIn": "main.go",
      "type": "complete",
      "line": 40,
      "hasRegistration": true,
      "hasHandler": true
    }
  ],
  "flowIntegrity": {
    "completeFlows": 100,
    "registrationOnly": 5,
    "handlerOnly": 3,
    "totalFlows": 108,
    "integrityPercent": 93
  },
  "jsModules": [
    {
      "file": "Public/js/game.js",
      "fileName": "game.js",
      "category": "game",
      "isESModule": true,
      "exports": [...],
      "imports": [...],
      "windowBindings": [...],
      "connectionCount": 12
    }
  ],
  "summary": {
    "bridgeFunctions": 108,
    "completeFlows": 100,
    "flowIntegrityPercent": 93,
    "jsModules": 96,
    "esModules": 22,
    "iifeModules": 0,
    "htmlScripts": 75,
    "scssPartials": 60,
    "violations": 0
  }
}
```

---

### 2. `wired.js` — Wired vs Orphaned Detector

**Purpose:** Compares repository tree with architecture manifest to find orphaned files.

**Usage:**
```bash
node tools/archscan/wired.js
```

**Output:** `tools/archscan/wired-manifest.json`

**How It Works:**
1. Scans repo tree (respects `.gitignore` with negation + anchoring)
2. Builds set of referenced files from architecture manifest
3. Classifies each file as wired or orphaned
4. Categorizes by type: frontend, backend, styles, media, docs, vendor, config

**Gitignore Support:**
- Negation patterns (`!pattern`) — re-include previously excluded files
- Anchored patterns (`/pattern`) — match from root only
- Directory patterns (`pattern/`) — match directories only
- Wildcards (`*`, `**`, `?`)

**Import Resolution:**
- Resolves `./foo` and `../foo` relative imports
- Resolves bare specifiers (returns null — can't map to file)
- Tries `.js` extension and `index.js` fallback

**Always-Wired Files:**
- `bridge_service.go`, `main.go`, `server.go`, `menu_state.go`
- `package.json`, `go.mod`, `go.sum`, `Dockerfile`, `.gitignore`
- `index.html`, `Public/wasm_exec.js`, `Public/src/scss/main.scss`

**Manifest Structure:**
```json
{
  "timestamp": "2026-09-07T...",
  "wired": [
    { "path": "Public/js/game.js", "category": "frontend", "size": 39221, "isReferenced": true }
  ],
  "orphaned": [
    { "path": "battle_service.go", "category": "backend", "size": 97400, "isReferenced": false }
  ],
  "stats": {
    "total": 715,
    "wired": 231,
    "orphaned": 484,
    "wiredPercent": 32
  },
  "byCategory": {
    "frontend": { "wired": 98, "orphaned": 6 },
    "backend": { "wired": 4, "orphaned": 79 },
    "media": { "wired": 0, "orphaned": 245 }
  }
}
```

**Categories:**
| Category | Description |
|----------|-------------|
| `frontend` | JS, TS, JSX, TSX files |
| `backend` | Go files |
| `styles` | CSS, SCSS, SASS, LESS |
| `media` | Images, audio, video, fonts |
| `docs` | Markdown files |
| `vendor` | Files in `vendor/` directory |
| `config` | JSON, YAML, TOML, ENV files |
| `generated` | WASM, node_modules |

---

### 3. `render.js` — Architecture Diagram Renderer

**Purpose:** Generates interactive HTML architecture diagram from manifests.

**Usage:**
```bash
node tools/archscan/render.js
```

**Output:** `AI-Brain/architecture.html`

**Dependencies:**
- `manifest.json` (from `scan.js`)
- `wired-manifest.json` (from `wired.js`)
- `template.html` (HTML skeleton)
- `views.js` (view definitions — auto-injected)

**Features:**
- 7 views: Full System, Server, WASM Bridge, JavaScript, CSS/SCSS, Data Flow, Analytics
- Click nodes to drill into details
- Click highlighting shows connected paths
- **Viewport camera:** wheel zoom toward cursor, drag empty canvas to pan, `+`/`−`/`⌂` HUD, double-click or Ctrl+0 to fit. No page scroll.
- Analytics sub-tabs: Overview, Language, Connections, Health, Wired
- Pie charts for distribution visualization
- **Health score** calculated from 4 factors (see below)

---

### 4. `views.js` — View Definitions

**Purpose:** All diagram view functions in one data-driven module.

**Key Functions:**
- `viewFull`, `viewServer`, `viewWasm`, `viewJS`, `viewCSS`, `viewData`, `viewAnalytics`
- `renderHealthChart` — Health visualization with 4-factor breakdown
- `renderOverviewChart`, `renderWiredChart`, `renderLanguageChart`, `renderConnectionsChart`

**Category Derivation:**
```javascript
// Prefix-based categorization — no hardcoded filename lists
const CATEGORY_PREFIXES = [
  { prefix: 'app', category: 'core' },
  { prefix: 'config', category: 'core' },
  { prefix: 'game', category: 'game' },
  { prefix: 'deck', category: 'game' },
  { prefix: 'economy', category: 'economy' },
  { prefix: 'criminality', category: 'social' },
  { prefix: 'menu', category: 'menu' },
  { prefix: 'collective', category: 'ai' },
  // ... add new prefixes as codebase grows
];
```

---

### 5. `watch.js` — Continuous Change Detection

**Purpose:** Watches codebase and updates manifests automatically.

**Usage:**
```bash
node tools/archscan/watch.js
```

**Behavior:**
- Runs every 5 seconds
- Compares current state with last manifest
- Prints changes to console
- Auto-refreshes both `manifest.json` and `wired-manifest.json`

**Detected Changes:**
- Bridge functions added/removed
- JS module export/import count changes
- Wired/orphaned file count changes

---

## Health Score Calculation

The health score (0-100) is calculated from **4 factors**, each contributing up to 25 points:

### Factor 1: Wired Ratio (0-25 points)
```
wiredScore = (wiredFiles / totalFiles) * 25
```
Measures how many files are connected to the architecture. High ratio = good.

### Factor 2: Flow Integrity (0-25 points)
```
integrityScore = (completeFlows / totalFlows) * 25
```
Measures bridge function pairing. A flow is "complete" when both registration (`js.Global().Set`) and handler (`func Xxx(this js.Value...)`) exist.

### Factor 3: Module Connectedness (0-25 points)
```
connectednessScore = (connectedModules / totalModules) * 25
```
Measures ES modules with at least 1 import or export. Modules with zero connections are orphaned in the module graph.

### Factor 4: Build Tag Violations (0-25 points, inverted)
```
violationScore = 25 - min(25, violations * 5)
```
No violations = full points. Each violation costs 5 points.

### Example Output:
```
Health: 72%
  Wired Ratio: 18/25 (72% wired)
  Flow Integrity: 23/25 (93% complete)
  Module Connectedness: 25/25 (22/22 connected)
  Build Violations: 25/25 (0 violations)
```

---

## Quick Start

### Full Refresh (Recommended)
```bash
cd tools/archscan
node scan.js && node wired.js && node render.js
```

Or via npm:
```bash
npm run arch:scan
```

### Watch Mode
```bash
node tools/archscan/watch.js
```

Then in another terminal:
```bash
# After making changes, refresh diagram:
node tools/archscan/render.js
```

### Build (includes arch:scan)
```bash
npm run build
```

---

## File Structure

```
tools/archscan/
├── scan.js           # Architecture bridge scanner
├── wired.js          # Wired vs orphaned detector
├── render.js         # Diagram renderer (orchestrator)
├── views.js          # View definitions (data-driven)
├── template.html     # HTML skeleton (extracted from render)
├── watch.js          # Change watcher
├── manifest.json     # Architecture manifest (auto-generated)
├── wired-manifest.json # Wired manifest (auto-generated)
└── README.md         # This file
```

---

## Troubleshooting

### Issue: `Cannot find module './views.js'`
**Cause:** New file not found
**Fix:** Ensure `views.js` exists in the same directory as `render.js`.

### Issue: `matchGlob is not defined`
**Cause:** Old reference to removed function
**Fix:** `wired.js` uses `matchesIgnore()` now. Update any custom code.

### Issue: `parseGitignore is not defined`
**Cause:** Function was moved/renamed
**Fix:** Export is available via `require('./wired.js').parseGitignore()`.

### Issue: Wired percentage seems low
**Cause:** Go service files aren't imported by JS (normal)
**Fix:** This is expected. Backend services are wired via Go imports, not JS imports.

### Issue: Changes not detected by watcher
**Cause:** Watcher compares manifest snapshots, not file system events
**Fix:** This is by design. Run `node tools/archscan/render.js` to force refresh.

---

## Architecture Rules

### Build Tags
- `server.go` → `//go:build !js && !wasm` (authoritative server)
- `main.go` → `//go:build js && wasm` (WASM client)

### Module Pattern
- ALL new JS must be ES modules (`import`/`export`)
- ALL new JS imported by `app.js`
- NO standalone `<script>` tags for new code
- State lives in WASM, not DOM or localStorage

### GitIgnore Compliance
Scanner respects `.gitignore` patterns including:
- `node_modules/`
- `*.exe`, `*.wasm`
- `Public/main.wasm`
- `data/`, `devdata/`
- `.env` files
- OS files (`.DS_Store`, `Thumbs.db`)
- Negation patterns (`!pattern`)

Plus hardcoded ignores:
- `.git/`
- `tools/` (scanner itself)

---

## API Reference

### scan.js
```javascript
const { scan, scanGoBridge, scanJSModules, scanHTMLScripts, scanSCSSPartials, deriveModuleCategory } = require('./scan.js');
const manifest = scan(); // Full scan
```

### wired.js
```javascript
const { detectWiredOrphaned, formatSize, parseGitignore, resolveImport } = require('./wired.js');
const wired = detectWiredOrphaned();
```

### render.js
```javascript
const { generateHTML, computeAnalytics, main } = require('./render.js');
main(); // Regenerates diagram
```

### views.js
```javascript
const { viewFull, viewServer, viewWasm, viewJS, viewCSS, viewData, viewAnalytics, renderHealthChart } = require('./views.js');
```

### watch.js
```javascript
const { watch } = require('./watch.js');
watch(); // Starts watcher
```

---

## Adding New Features

### To track a new JS module:
1. Create `Public/js/mymodule.js` with ES module exports
2. Import in `app.js`: `import { x } from './js/mymodule.js'`
3. If the module is in a new category, add a prefix rule to `CATEGORY_PREFIXES` in `views.js`
4. Run `node tools/archscan/render.js`

### To track a new Go bridge function:
1. Add function in `bridge_service.go` or `menu_state.go`
2. Register in `registerWasmHooks()`
3. Run `node tools/archscan/render.js`

### To add a new analytics view:
1. Add render function in `views.js`
2. Add sub-tab button in `viewAnalytics`
3. Add handler in `renderAnalyticsSub`

---

## Related Documents

- `AI-Brain/UI-Architecture-Plan.md` — Canonical architecture guide
- `AI-Brain/App-Aspect-Index.md` — ~99% app functionality reference
- `Public/js/problems.md` — Resolved issues + current state
- `AI-Brain/architecture.html` — Interactive diagram

---

*This manual is maintained by Seraph (Lead Systems Architect). Update when tools change.*
