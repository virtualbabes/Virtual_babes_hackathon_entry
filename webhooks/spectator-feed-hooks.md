# Web Page Hooks — Spectator Feed (Landing Page)

> **Captured:** 2026-08-30 by nft-seduction-code-agent (Lead Architect, KEY 4 WAIT).
> **Status:** EXISTING in code. Saved for the later §25 web-3D wiring pass (deferred edit — do NOT modify
> the referenced source files yet; this is documentation only).
> **Source of truth:** live code, verified via grep. All references are file:line in the current tree.

## Purpose
The landing page already exposes a **spectator feed**: the Live Lobby player list, where each active player
has a "Watch" action. This is the cycle-feed hook the §25 3D client must reuse to fly through random active
players. No new server endpoint is required — `window.sendSpectate` is the contract.

## Hook chain (verified 2026-08-30)

| Step | File:Line | What it does |
|---|---|---|
| 1. Feed surface | `Public/index.html:183` | `<ul id="active-players" class="player-list">` — the Live Lobby list rendered on the landing page. |
| 2. WS push | `Public/js/network.js:158` | `case "lobby_update":` → `updatePlayerList(msg.payload.players)` + market/bounty tickers. |
| 3. Render list | `Public/js/game.js:91` | `export function updatePlayerList(players)` — clears `#active-players`, renders each player as a row. |
| 4. Watch button | `Public/js/game.js:116` | Per non-self player: `<button onclick="sendSpectate('${p.id}')">Watch</button>`. |
| 5. Hook def | `Public/js/game.js:334` | `export function sendSpectate(targetId)` — sends `{type:"spectate", payload:{target_id}}` over WS; resets local selection/spectator state; toasts "Requesting access to stream…". |
| 6. Global hook | `Public/js/game.js:787` | `window.sendSpectate = sendSpectate;` — the contract the §25 3D client calls. |
| 7. Server stream | `handlers_public.go:21` | `handleSpectatorWager` / spectate path validates `MatchState.Spectators` (`common_types.go:289`) and streams the target's match state back (PILLAR 4 replay resilience). `proceedToWarRoom` (`game.js:353`) loads the snapshot. |

## Related (useful for the §25 fly-through)
- `Public/js/leaderboard_region.js:74` `window.openLeaderboardRegion()` — neutral gathering hub; "Enter 3D World"
  calls `window.enter3DWorld` (unmet hook today). Entry/exit hub for worlds.
- `server.go:1531` `/api/regions` → `GetRegionViews()` (`seasonal_event_engine.go:796`) — region/entity warp data.
- `handlers_public.go:339` `handleActiveMatches` (`/api/active-matches`) — prefer players *in a match* for
  battle-level spectating (richer feed).
- §27 `theme_engine.go` `ComputeThemeVector` — tint the fly-through by each cycled player's theme.

## §25 3D client integration contract (deferred)
When mounting the 3D client:
1. Register `window.enter3DWorld` / `window.enterMenuWorld` (from `leaderboard_region.js`).
2. On a timer (or "Next" button), pick a random id from the last `lobby_update.players` payload and call
   `window.sendSpectate(id)` to cycle the camera to that player's region/avatar.
3. Render the player's §27 `ThemeVector` world tint during the fly-through.
4. Determinism: random selection uses a uint64 counter seeded off lobby player count; NO floats; NO cloud models.

## Note
This file is **documentation only**. The referenced source files are NOT to be edited in this pass. The actual
wiring (auto-cycle timer + `enter3DWorld` registration) belongs to the §25 implementation KEY.
