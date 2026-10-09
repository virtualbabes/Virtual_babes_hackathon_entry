# Flow Debug Tools

> **Purpose:** Debug and verify the menu customization + controller navigation pipeline.
> **Location:** `tools/flow/`
> **Part of:** `@tool-io` domain (isolated from game logic)

---

## Tools

| Tool | Command | Purpose |
|---|---|---|
| `flow_debug` | `python tools/main.py flow debug` | Test shape/size/grid/controller modules for validity |
| `flow_verify` | `python tools/main.py flow verify` | Verify full pipeline integration end-to-end |

---

## What They Check

### flow_debug.js

Tests individual modules for:

1. **Shape Library** — 60+ shapes with valid clip-paths, pathway mapping, system mapping
2. **Size Tiers** — 6 sizes with px/font/icon values
3. **Grid Layouts** — 9 grid types with navType + compute function
4. **UserPreferences** — menuLayout config, new functions, exposure on window
5. **Controller Navigation** — grid-specific nav handlers, polling, analog stick
6. **renderStarredItems** — reads prefs, computes positions, applies shapes/sizes
7. **Pathway Avenues** — tier calculation, CSS variables, animation injection
8. **app.js Integration** — all new modules imported and wired

### flow_verify.js

Tests the full data flow:

1. User stars tab → `UserPreferences.toggleStar()` saves to localStorage
2. `notify()` triggers `renderStarredItems()` in Main Menu
3. `MenuCustomization.computePositions()` calculates positions
4. Buttons rendered with shape clip-paths + size styling
5. `controller_nav.js` maps D-pad to grid-specific focus movement
6. `pathway_avenues.js` renders tier progress in Constellation Hub
7. `app.js` wires everything together

---

## Usage

```bash
# Run all debug tests
python tools/main.py flow debug

# Run pipeline verification
python tools/main.py flow verify

# Run both
python tools/main.py flow debug && python tools/main.py flow verify
```

Exit code 0 = all pass. Exit code 1 = failures (see output).

---

## Adding a New Test

1. Add checks to the relevant test function in `flow_debug.js` or `flow_verify.js`
2. Use `ok()`, `bad()`, `info()` helpers for output
3. Run `python tools/main.py flow debug` to verify

---

## Related Tools

- `archscan/scan` — Architecture manifest (bridge functions, JS modules)
- `archscan/wired` — Wired vs orphaned detection
- `server/launcher` — Dev server launcher

---

*Last updated: 2026-09-05*
