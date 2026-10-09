# NFT-Seduction Toolset

Domain of **Toolio** (`@tool-maker`). Everything under `tools/` exists so agents can **see, route, and verify** work on NFT-Seduction without guessing. It is not the app. Do not bridge app logic into these tools, and do not treat this folder as a place to implement game features.

**Canonical CLI**

```bash
python tools/main.py diagnose
python tools/main.py <category> <tool>
```

Categories and tools are defined in `tools/config.toml`. If a command is missing, the config is wrong — do not invent paths.

| Category | Tools | Purpose |
|---|---|---|
| `archscan` | `scan`, `wired`, `render`, `watch` | Ground truth of the architecture |
| `build` | `console_ps`, `console_sh`, `mobile_ps`, `mobile_sh`, `integrity` | Console / mobile binaries |
| `server` | `dev` | Local dev server — build + watch + health watchdog + optional CDP test harness |
| ~~`convert`~~ | **RETIRED + DELETED 2026-09-20** | Seven one-shot JS IIFE converters, removed from the tree. Each wrote product source **in place** with no backup and no dry-run through a **hard-coded absolute path**, toward an **IIFE state that is the OPPOSITE of the single-entry ESM architecture** — and their own target defects survived them: the `convert_v2` repair of the `initwindow.AudioContext` mangling never fired, and the `fix_syntax` nullish-precedence class is still live in six modules. Git history only. |
| `flow` | `debug`, `verify` | Debug + verify menu customization pipeline (shapes, sizes, grids, controller nav) |

---

## Why an agent should use this

NFT-Seduction is a Go server + WASM client + ES-module frontend. Agents fail here by:

1. Editing a `.go` service that is never registered on `js.Global()`
2. Adding a JS file that `app.js` never imports (orphan)
3. Implementing a handler with no matching `js.Global().Set`
4. Assuming “language mix” or “file counts” from memory
5. Touching `llama_tools/` (app dependency, not this domain)

Archscan replaces that guesswork with **two JSON maps** plus a diagram.

| Artifact | Path | What an agent reads it for |
|---|---|---|
| Architecture manifest | `tools/archscan/manifest.json` | Bridge functions, JS imports/exports, flow integrity |
| Wired manifest | `tools/archscan/wired-manifest.json` | Reachable vs orphan files, by category |
| Interactive diagram | `tools/archscan/architecture.html` (master) and `AI-Brain/architecture.html` (copy) | Human + agent visual of layers, health, language, orphans |
| Scanner manual | `tools/archscan/README.md` | Field-level schema |

Refresh all three:

```bash
python tools/main.py archscan scan
python tools/main.py archscan wired
python tools/main.py archscan render
```

Or `npm run arch:scan` from repo root (app script; still writes into `tools/archscan/`).

---

## Agent playbook

### 1. Orient before coding

```bash
python tools/main.py diagnose
python tools/main.py archscan render
```

Then open `tools/archscan/architecture.html` (wheel zoom, drag pan, Analytics tabs). Read `manifest.json` → `summary` and `flowIntegrity` before proposing a change.

### 2. Adding or changing a WASM bridge

Query `bridgeFunctions` in `manifest.json`:

- `type: "complete"` — registration **and** handler exist. Safe to extend.
- `hasRegistration` only — exposed to JS, no Go handler. Do not call from JS until paired.
- `hasHandler` only — implemented, not on `js.Global()`. JS cannot see it.

After the change: re-scan. `flowIntegrity.integrityPercent` must not drop.

### 3. Adding a JS module

A new file under `Public/js/` is **orphaned** until something imports it (almost always `app.js`). Confirm with `wired-manifest.json`:

- `wired[]` with `category: "frontend"` — live in the graph
- `orphaned[]` — dead weight; do not “fix” an orphan by rewriting it. Wire it, or leave it.

Rule of the house: **new JS is ES modules**. The convert scripts that migrated the codebase off IIFE were **RETIRED and DELETED (2026-09-20)** — they mutated `Public/js` in place toward the OPPOSITE architecture, so re-running one would destroy the composition graph.

### 4. After a patch (regression check)

Re-render and compare:

| Signal | Healthy direction |
|---|---|
| Health % (diagram / analytics) | Stable or up |
| Flow integrity % | Stable or up |
| Wired % | Up if you intended to connect files |
| Orphan count | Down only if you actually wired or deleted |
| Build tag violations | Stay at 0 |

If health falls and you did not mean to detach a flow, you broke pairing or left a module unimported.

### 5. Finding “where does X live?”

Do not glob the whole repo first.

1. Language tab / `ANALYTICS_DATA.languages` — real extension counts from the inventory (Go, JS, CSS, Markdown, Images, Media…). Not a guess from bridge files.
2. `jsModules[].file` + `imports` / `exports` — module graph.
3. `bridgeFunctions[].name` — WASM surface area.
4. Wired `byCategory` — backend vs frontend vs styles vs media.

### 6. Dev server

Use the toolchain, not the stale npm script:

```bash
python tools/main.py server dev
```

`package.json` `"dev"` pointed at `launch_dev_server.ps1` in the **repo root** — CORRECTED 2026-09-09: `npm run dev` now runs `tools/server/dev_server.ps1`, the unified replacement. That file lives at `tools/server/launch_dev_server.ps1`. Fixing `package.json` is an **app** change — out of Toolio scope. Agents working the app may patch that script path; Toolio must not.

`llama_tools/` is an app runtime dependency. Out of scope.

---

## Isolation rules

- **In scope:** `tools/**`, this file, `config.toml`, archscan outputs.
- **Generated copies:** `AI-Brain/architecture.html` is a render target, not app source. Regenerating it is allowed. Hand-editing it is wasted — `render.js` overwrites it.
- **Out of scope:** Go services, `Public/js` features, WASM game state, `llama_tools/`.
- **Convert scripts:** RETIRED + DELETED (2026-09-20) — see git history. Never re-introduce an IIFE converter; section 1 of `app-entry-mandate.md` makes `app.js` the single composition root.

---

## Layout

```
tools/
├── main.py              Unified CLI
├── config.toml          Tool registry (source of command names)
├── TOOLSET.md           This file — agent contract
├── archscan/            Architecture scanner + diagram
│   ├── scan.js          → manifest.json
│   ├── wired.js         → wired-manifest.json
│   ├── render.js        → architecture.html (here + AI-Brain/)
│   ├── views.js         Diagram views (incl. language / health)
│   ├── template.html    Diagram shell (zoom camera lives here)
│   └── README.md        Scanner schema
├── build/               Console / mobile builders
├── server/              Unified dev server (build + watch + health)
├── convert/             (RETIRED 2026-09-20 — directory removed)
└── Debug-screens/       Static debug HTML snapshots
```

When you add a tool: register it in `config.toml`, then `python tools/main.py diagnose`. If it is not in the config, it does not exist.
