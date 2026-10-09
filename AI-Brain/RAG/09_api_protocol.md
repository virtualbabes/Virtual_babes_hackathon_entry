# 09 — API & Protocol Surface

> **Scope:** All HTTP handler functions across `handlers_public.go`, `handlers_public_new.go`, `handlers_admin.go`, `handlers_criminality.go`, `handlers_dashboard.go`, `handlers_rumor.go`, `career_path_handlers.go`, and the WebSocket message protocol in `lobby_manager.go`.
> **Architecture:** Backend-authoritative. Every HTTP handler lives in a `//go:build !js && !wasm` file. The WASM client never implements an HTTP server. All state mutations go through the Lobby mutex or the appropriate subsystem mutex. The WebSocket protocol is the primary live interaction channel; REST is primarily a read layer and an onboarding/bootstrap channel.

---

## 1. THE HTTP HANDLER LANDSCAPE

### 1.1 Handler File Map

| File | Handle functions | Role |
|------|-----------------|------|
| `handlers_public.go` | 19 | Public-facing read routes: status, card stats, token balances, health probes, spectator wager, faucet dashboard, leaderboard, underworld reads |
| `handlers_public_new.go` | 20 | New public routes: shop purchase, equip card title, accept justice mission, church members, wagers, adjust mood, brew tea, resolve wager |
| `handlers_admin.go` | 39 | Admin-only routes: player management, economy controls, district governance, tournament ops, faith church admin, entity management, AI citizen controls |
| `handlers_criminality.go` | 5 | Underworld/criminality routes: contract acceptance, black market listing, fence goods, sabotage, heist |
| `handlers_dashboard.go` | 2 | Dashboard routes: player dashboard, justice dashboard |
| `handlers_rumor.go` | 1 | Rumor spread route |
| `career_path_handlers.go` | 4 | Career path routes: path query, promotion history, role vocabulary, career XP |
| **Total** | **~90** | |

### 1.2 The Handler Pattern

Every handler follows the same shape:

```go
func (l *Lobby) handleXxx(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {  // or MethodGet
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    // Decode body
    var req struct { ... }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil { ... }
    // Lock
    l.mutex.Lock()
    defer l.mutex.Unlock()
    // Validate
    // Mutate
    // Respond
}
```

**Exceptions that document themselves:**
- Garden/mood doors deliberately take **NO lobby lock** and document the ABBA cycle they closed, delegating to engine-owned `AddRitual`/`Meditate`/`AdjustReputation`.
- Some justice reads pass placeholder arguments to the service (`GetDashboardForPlayer(wallet, nil, 0)`).

---

## 2. THE READ LAYER — PUBLIC STATUS & POPULATION

### 2.1 `handlePublicStatus` (`handlers_public.go`)

**What it serves:** A JSON blob with the lobby's public-facing state.

**Fields it publishes:**
- `total_players` = `len(l.clients)` — a **CONNECTION count**, not a civilization population, on an endpoint written for external sites
- `faucet_balance` = the **float mirror** only (`l.faucetBalance`), with no integer twin beside it

**The drift:** The `total_players` field misleads external consumers. It counts WebSocket connections, not unique players, and says nothing about AI citizens, offline players, or dormant wallets.

### 2.2 `handleHealthCheck` / `handleReadyEndpoint` (Kubernetes probes)

Two probe implementations that implement one rule differently:
- `handleHealthCheck` breaks on **first success**
- `handleReadyEndpoint` sets `isReady=false` inside the loop so a **later success cannot restore it**

The two probes disagree about what "ready" means. A failing dependency that recovers mid-check is reported as unhealthy by one probe and ready by the other.

### 2.3 `handleCardStats` (`handlers_public.go`)

Calls the wallet-scoped `getVerifiedCard("", …)` with **NO wallet** (:792 vs :816). The helper collapses "verification failed" with "no such card" (:833-840), so a caller cannot tell whether the card doesn't exist or the verification failed.

### 2.4 `handleGetPlayerTokens` (`handlers_public.go`)

Serves `nil` for both token balances with the message `"placeholders — these require ASA IDs from env vars"` (:262-264). This is the **only** statement of that state, and it is the opposite of the `nonNilMap`'s `{}`/`[]` rule that the rest of the tree follows.

---

## 3. THE SPECTATOR WAGER — DESTROYED STAKE

### 3.1 `handleSpectatorWager` (`handlers_public.go`)

**What it does:**
```go
l.playerBalances[spectatorWallet] -= req.WagerMicro
match.WagersMicro += req.WagerMicro
```

**The problem:** The stake goes into `match.WagersMicro` — a counter on the `MatchState` that is **deleted at match end**. This file has **no resolution path**: no winner credit, no refund, no escrow release. The wagered micro-VBV is destroyed at match end with no accounting.

**The wallet problem:** The wallet comes from `req.SpectatorWallet` — `getWalletFromRequest` is **never called**. Any client can wager and drain **any wallet** that happens to be spectating. The `match.Spectators` check proves the wallet is *watching*, not that the requester *owns* it.

**Status:** Live route. No fix recorded.

---

## 4. THE WAGER RESOLUTION — MINTED WIN

### 4.1 `HandleResolveWager` (`handlers_public_new.go`)

**What it does:**
```go
if req.Winner == "player" {
    payout := req.Wager * 2
    balance += payout
    l.playerBalances[wallet] = balance
}
```

**The problem:** BOTH the `Wager` and the `Winner` come from the **request body**. There is no match reference, no stake reference, and **no debit**. It is the mirror of `handlers_public.go`'s spectator wager: that door **destroys** the stake; this one **invents** the win.

**Status:** Live route. No fix recorded.

---

## 5. THE SHOP PURCHASE — TAKES MONEY, DELIVERS NOTHING

### 5.1 `HandleShopPurchase` (`handlers_public_new.go`)

Debits `cost × quantity` and answers `"purchase recorded"` — **no inventory write, no grant, no sink routing, no audit**. The price map is a hard-coded four-item catalogue (:82-87).

### 5.2 `HandleBrewTea` (`handlers_public_new.go`)

The same shape: debits, answers `"tea brewed"`, returns a buff as a **STRING** (:458), with a **second copy** of its own catalogue (:411-418 vs the served :390-396).

---

## 6. THE ADMIN ROUTE SURFACE

**File:** `handlers_admin.go` (39 handle functions)

The largest handler file. Covers:

| Area | Sample routes |
|------|--------------|
| Player management | Set balance, reset player, ban/unban, set wanted level |
| Economy controls | Set faucet rate, adjust treasury, trigger distribution, set club treasury |
| District governance | Set tax rate, set governance metric, distribute district rewards |
| Tournament ops | Create tournament, set prize pool, finalize tournament |
| Faith church admin | Open/close church, set ritual cost, set faith power |
| Entity management | Spawn entity, set entity level, transfer entity, reclaim entity |
| AI citizen controls | Set career, set treasury, trigger investment, toggle autonomy |
| Rivalry / theme | Set rivalry weight, set theme vector, set slide theme |

**Note:** The admin surface is large and most routes take the lobby write lock. The `slide_theming.go` family (served by admin routes) has the **HTTP 200 on refusal** defect documented in §18 of `19_coverage_ledger.md`.

---

## 7. THE WEBSOCKET PROTOCOL — 65 MESSAGE TYPES

**File:** `lobby_manager.go` — `handleGameProtocol()`

The WS protocol is the primary live interaction channel. 65 message types are handled in a `switch` on the `type` field of the incoming envelope.

### 7.1 Message Type Catalogue (first 30 of 65)

| Message type | Direction | Purpose |
|-------------|-----------|---------|
| `abort_justice_mission` | Client→Server | Abort an active justice mission |
| `abort_underworld_contract` | Client→Server | Abort an active underworld contract |
| `accept_justice_mission` | Client→Server | Accept a justice mission |
| `accept_underworld_contract` | Client→Server | Accept an underworld contract |
| `alliance_accept` | Client→Server | Accept an alliance invite |
| `alliance_dissolve` | Client→Server | Dissolve an alliance |
| `alliance_invite` | Client→Server | Invite a player to an alliance |
| `aos_raid` | Client→Server | Initiate an AOS raid |
| `bail_card` | Client→Server | Bail a jailed card |
| `base` | Both | Base/ping message |
| `chat` | Client→Server | Gloat/chat message (rate-limited 30 msg/10s) |
| `claim_dividends` | Client→Server | Claim entity dividends |
| `create_club` | Client→Server | Create a club |
| `create_lease` | Client→Server | Create an infrastructure lease |
| `equip_cosmetic` | Client→Server | Equip a cosmetic item |
| `freeze_dividends` | Client→Server | Freeze dividend claims |
| `harvest_all_dividends` | Client→Server | Harvest all available dividends |
| `heist` | Client→Server | Initiate a heist |
| `hire_player` | Client→Server | Hire a player for a club |
| `initiate_recovery` | Client→Server | Request match sync (desync recovery) |
| `join_club` | Client→Server | Join a club |
| `join_queue` | Client→Server | Join the matchmaking queue |
| `justice_flag_player` | Client→Server | Flag a player for justice review |
| `kidnap_request` | Client→Server | Request a kidnap contract |
| `launder_capital` | Client→Server | Initiate capital laundering |
| `leave_queue` | Client→Server | Leave the matchmaking queue |
| `link_wallet_request` | Client→Server | Request wallet linkage |
| `list_recovery_bounty` | Client→Server | List available recovery bounties |
| `loyalty_synthesis` | Client→Server | Synthesize loyalty |
| `arbitrum` | Client→Server | Arbitrum-related message |

### 7.2 The Envelope Protocol

Every WS message uses the same envelope shape:

```go
type WSEnvelope struct {
    Type    string          `json:"type"`
    FromID  string          `json:"from_id"`
    ToID    string          `json:"to_id"`
    Payload json.RawMessage `json:"payload"`
}
```

**The `base` message** is the ping/heartbeat. Absence of a `base` within the session watchdog window triggers eviction.

### 7.3 Rate Limiting on `chat`

The `chat` message type is rate-limited to **30 messages per 10 seconds** per wallet via `allowMessage()`. Excess messages are dropped, not queued.

### 7.4 Desync Recovery Protocol

```
Client receives AuthoritativeFrame with mismatched StateHash
    → ClientReplayEngine.InitiateRecovery()
    → UI freeze + requestMatchSync(last_sequence_id)
    → Server SyncHandshaker.CatchUpPlayer(fromSequence) → []FrameDelta
    → Server pushes sync_response frames via WS
    → Client ExecuteSyncHandshake() replays until caught up
    → If >10s → ErrBufferOverflow → alert + re-recovery
    → unlockCanvasInteractions() when buffer clears
```

---

## 8. THE CAREER PATH API

**File:** `career_path_handlers.go` (4 handle functions)

| Handler | Purpose |
|---------|---------|
| Career path query | Returns the player's current career path and progression |
| Promotion history | Returns the player's promotion history |
| Role vocabulary | Returns the note-purpose vocabulary (shared with `handleNoteVocabulary`) |
| Career XP | Returns the player's career XP breakdown |

The role vocabulary handler is the **single owner** of the note-purpose vocabulary. A browser reads the prefixes it is allowed to write from here instead of re-declaring them, so a client and server can never disagree about a note prefix — a disagreement that would NOT raise an error, only silently fail verification.

---

## 9. THE DASHBOARD API

**File:** `handlers_dashboard.go` (2 handle functions)

| Handler | Purpose |
|---------|---------|
| Player dashboard | Returns the player's full dashboard: stats, progress, balances, career, club |
| Justice dashboard | Returns the justice-specific dashboard: bounties, missions, reputation, wanted level |

**Note:** Both justice reads pass placeholder arguments to the service (`GetDashboardForPlayer(wallet, nil, 0)`).

---

## 10. THE CRIMINALITY API

**File:** `handlers_criminality.go` (5 handle functions)

| Handler | Purpose |
|---------|---------|
| Accept underworld contract | Accept a specific underworld contract |
| Black market listing | List goods on the black market |
| Fence goods | List fenced goods |
| Sabotage | Initiate a sabotage contract |
| Heist | Initiate a heist contract |

---

## 11. ROUTE COUNT SUMMARY

| Layer | Count | Source |
|-------|-------|--------|
| HTTP handler functions (measured) | ~90 | 7 handler files, `func handle*` census |
| WebSocket message types (measured) | 65 | `lobby_manager.go` switch cases |
| REST routes (claimed) | ~322 | `server_main.go` claim — includes route wiring not measured here |
| WS handlers (claimed) | ~69 | Claimed in repo stability — includes broadcast/meta handlers |

**The gap between measured and claimed:** The 322 REST route claim in `00_repo_stability.md` and `01_architecture_overview.md` is not independently verified in this sweep. The measured ~90 handler functions cover the handler implementations; the additional routes are likely wiring in `server_main.go` (mux registration, middleware chains, alias routes) that this sweep did not read.

---

## 12. THE FLOAT-MONEY SURFACE IN THE API

Several handler paths expose float fields that the ledger forbids:

| Route | Float field exposed | Source |
|-------|--------------------|--------|
| `handlePublicStatus` | `faucet_balance` (float mirror only) | `l.faucetBalance` |
| `handleGetPlayerTokens` | `nil` placeholders | No real token data |
| `HandleShopPurchase` | Price map is hard-coded, no float | Safe by construction |
| Admin set balance | Accepts client-declared balance | No validation on input |

The **faucet dashboard** re-derives the scaler's number in floats from a different set than `applyDynamicScalingLocked` uses — the two disagree by the whole virtual-balance ledger (see §18 of `19_coverage_ledger.md`).

---

## 13. THE WALLET-LINKAGE PROTOCOL

`link_wallet_request` is the onboarding bridge. It connects a play wallet to an on-chain wallet. The linkage is what enables:
- On-chain signature verification for record doors
- The tenant vault registry to know which on-chain wallet owns a vault
- The session watchdog to audit on-chain balance against the play balance

**The linkage is asymmetric:** the play wallet is the identity; the on-chain wallet is the proof. A player can have multiple on-chain wallets linked, but only one play wallet per session.

---

## 14. WHAT THE API DOES NOT EXPOSE

| Missing | Why it matters |
|---------|---------------|
| No WebSocket authentication beyond origin check | `websocket.CheckOrigin` validates `ALLOWED_ORIGINS` env; no per-message auth token |
| No rate limit on HTTP handlers (per-wallet) | The rate limiter's per-route tier is not enforced (see §17 of `19_coverage_ledger.md`) |
| No pagination on leaderboard | The leaderboard serves all rows; no cursor or limit parameter |
| No WebSocket message schema versioning | The envelope has no version field; a client and server can disagree on payload shape with no error |
| No OpenAPI / Swagger spec | The 322 routes have no machine-readable contract; the drift matrix (`12_drift_matrix.md`) is the closest thing |

---

*End of 09 — API & Protocol Surface. Generated from measured handler census (7 files, ~90 handle functions, 65 WS message types) + source reads of the public handler files and lobby_manager.go.*
