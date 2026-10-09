# NFT-Seduction — Security Architecture Deep-Dive
> The split WASM is a SECURITY DESIGN, not a build detail. This is the most critical concept to understand.
> The server WASM (linux/amd64) is AUTHORITATIVE. The client WASM (js/wasm) is for INTERACTION ONLY.

---

## 1. THE SECURITY MODEL — TWO WASMS, ONE TRUTH

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         SERVER WASM (linux/amd64)                       │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                        Lobby (authoritative)                      │  │
│  │  ┌─────────────┐  ┌──────────────┐  ┌────────────────────────┐  │  │
│  │  │ MatchState   │  │ playerBalances│  │ leaderboard (PlayerStats)│  │  │
│  │  │ (per match)  │  │ (uint64 μVBV)│  │ (all player stats)     │  │  │
│  │  └─────────────┘  └──────────────┘  └────────────────────────┘  │  │
│  │  ┌─────────────┐  ┌──────────────┐  ┌────────────────────────┐  │  │
│  │  │ SyncHandshaker│ │ VerificationHook│ │ SessionWatchdog      │  │  │
│  │  │ (frame hist.)│  │ (nonce/audit) │  │ (session eviction)    │  │  │
│  │  └─────────────┘  └──────────────┘  └────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                              ▲                                          │
│                              │ AuthoritativeFrame{SequenceID,          │
│                              │   MoveIntent, StateHash}                 │
│                              │                                          │
│  ┌───────────────────────────┴───────────────────────────────────────┐  │
│  │                    WebSocket (gorilla/websocket)                    │  │
│  └───────────────────────────┬───────────────────────────────────────┘  │
└──────────────────────────────┼──────────────────────────────────────────┘
                               │
                               │ Envelope{type, from_id, to_id, payload}
                               │
┌──────────────────────────────┼──────────────────────────────────────────┐
│                         CLIENT WASM (js/wasm)                           │
│  ┌───────────────────────────┴───────────────────────────────────────┐  │
│  │                    Engine (interaction only)                        │  │
│  │  ┌─────────────────────────────────────────────────────────────┐  │  │
│  │  │                 ClientReplayEngine                           │  │  │
│  │  │  - Receives AuthoritativeFrame from server                   │  │  │
│  │  │  - Verifies SequenceID continuity                            │  │  │
│  │  │  - Computes local board hash                                 │  │  │
│  │  │  - Compares to server StateHash                              │  │  │
│  │  │  - Mismatch → InitiateRecovery() → UI freeze → catch-up      │  │  │
│  │  └─────────────────────────────────────────────────────────────┘  │  │
│  │  ┌─────────────────────────────────────────────────────────────┐  │  │
│  │  │                 Game (local render)                          │  │  │
│  │  │  - Board[9]*Card (local mirror)                             │  │  │
│  │  │  - Players[2]Player (local state)                           │  │  │
│  │  │  - Phase: "Lobby"/"Setup"/"Active"/"Finished"               │  │  │
│  │  │  - Turn, Scores, Rules (local cache)                        │  │  │
│  │  └─────────────────────────────────────────────────────────────┘  │  │
│  │  ┌─────────────────────────────────────────────────────────────┐  │  │
│  │  │                 WasmSignerHook                               │  │  │
│  │  │  - Requests signature from browser wallet extension          │  │  │
│  │  │  - Server verifies on-chain                                  │  │  │
│  │  └─────────────────────────────────────────────────────────────┘  │  │
│  │  ┌─────────────────────────────────────────────────────────────┐  │  │
│  │  │                 ClientRedirectManager                        │  │  │
│  │  │  - Locks canvas on eviction                                  │  │  │
│  │  │  - Flushes session storage                                   │  │  │
│  │  │  - Redirects to /login.html after 4s                         │  │  │
│  │  └─────────────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 2. THE AUTHORITATIVE FRAME PROTOCOL

This is the core anti-cheat mechanism. Every move in a multiplayer match goes through this cycle:

```
Client A (js/wasm)                          Server (linux/amd64)
        │                                          │
        │  1. PlaceCard() → MoveData{              │
        │       grid_index, card_id, power[4]      │
        │     }                                    │
        ├──────────────────────────────────────────▶│
        │                                          │ 2. Server validates:
        │                                          │    - Is it this player's turn?
        │                                          │    - Is the card in their hand?
        │                                          │    - Is the grid index valid?
        │                                          │    - Compute captures server-side
        │                                          │      via serverCheckCaptures()
        │                                          │    - Compute board hash
        │                                          │      via ComputeBoardHash()
        │                                          │    - Store frame in SyncHandshaker
        │                                          │
        │  3. AuthoritativeFrame{                  │
        │       SequenceID,                        │
        │       MoveIntent: {grid_index,          │
        │         card_id, power, player_index},  │
        │       StateHash: SHA256(board)          │
        │     }                                    │
        │◄──────────────────────────────────────────┤
        │                                          │
        │  4. Client WASM:                         │
        │     - Validates SequenceID continuity     │
        │     - Applies move to local Board[9]     │
        │     - Computes local board hash          │
        │     - Compares to server StateHash       │
        │     - Mismatch → InitiateRecovery()      │
        │                                          │
        │  5. SyncMove(frame) → applies to         │
        │     Game.Board, checks captures          │
        │     locally, verifies hash               │
        │                                          │
```

### Why this matters:
- **Client cannot fabricate a win.** Only the server calls `verifyWinner()`.
- **Client cannot alter power.** `getEffectiveServerPower()` runs server-side with snapshotted stats.
- **Client cannot skip turns.** SequenceID must be continuous; gaps trigger recovery.
- **Client cannot modify ledger.** `playerBalances` lives on the server; client only sees a mirror.
- **Client cannot fake board state.** SHA-256 hash of board is computed and verified.

---

## 3. THE SNAPSHOT MODEL — MATCH STATE IS FROZEN AT START

When a match begins, the server snapshots ALL relevant player stats into `MatchState`:

```
initiatePairedMatch() → MatchState{
    P1Wallet, P2Wallet,           // Wallets (snapshotted)
    P1Deck, P2Deck,               // Card IDs (snapshotted)
    P1WantedLevel, P2WantedLevel, // Justice state (snapshotted)
    P1Cunning, P2Cunning,         // Criminal state (snapshotted)
    P1Nurturing, P2Nurturing,     // Garden state (snapshotted)
    P1RegionalBoost, P2RegionalBoost, // Territory state (snapshotted)
    P1CoalitionBoost, P2CoalitionBoost, // Alliance state (snapshotted)
    BoardMoods[9],                // Tile moods (authoritative)
    Rules,                        // Active rules
    FaithComposition,             // Per-faith card counts (snapshotted)
    ActiveItemBuffs,              // Item buffs (snapshotted)
    TerritoryID,                  // Where match is played
    WagersMicro,                  // Spectator wagers
}
```

### Why this matters:
- **Mid-match stat changes don't affect the match.** If your Wanted Level rises during a match, your power isn't penalized retroactively.
- **Prevents race conditions.** You can't alter your stats to change match outcome.
- **Ensures replayability.** The exact match state can be reconstructed from the snapshot.

---

## 4. THE DESYNC DETECTION & RECOVERY SYSTEM

This is PILLAR 4 — the replay resilience engine. It detects and corrects client-side tampering or network drift.

### Detection (Client-side):
```go
// main.go:1342-1350 (js/wasm)
// After applying a server frame:
if frame.StateHash != (BoardStateHash{}) {
    localHash := ComputeBoardHash(Game.Board, pIdx)
    if localHash != frame.StateHash {
        fmt.Printf("[WASM DESYNC] Real-time drift detected! Local: %x, Authority: %x\n", localHash, frame.StateHash)
        Game.ReplayEngine.InitiateRecovery() // Trigger UI freeze and catch-up
    }
}
```

### Recovery:
1. `InitiateRecovery()` — freezes UI, calls `requestMatchSync()` in JS
2. JS sends `sync_request` to server with `last_sequence_id`
3. Server `SyncHandshaker.CatchUpPlayer(fromSequence)` returns `[]FrameDelta`
4. Server pushes frames via `sync_response` WS message
5. Client `ExecuteSyncHandshake()` replays frames until caught up
6. If reconstruction > 10s → `ErrBufferOverflow` → alert + re-recovery
7. `unlockCanvasInteractions()` when buffer clears

### The hash is deterministic:
```go
// battle_service.go:179-211 (linux/amd64)
func (l *Lobby) ComputeBoardHash(board [9]*ServerCard, playerIndex int) BoardStateHash {
    hasher := sha256.New()
    for _, c := range board {
        // Write ID, Owner, Artifact as fixed-width binary
        // (ensures cross-platform parity between 64-bit server and 32-bit WASM)
    }
    // Include turn actor in hash
    turnBuf := make([]byte, 4)
    binary.BigEndian.PutUint32(turnBuf, uint32(playerIndex))
    hasher.Write(turnBuf)
    var hash BoardStateHash
    copy(hash[:], hasher.Sum(nil))
    return hash
}
```

---

## 5. THE SESSION WATCHDOG — CONTINUOUS VERIFICATION

The server actively monitors connected clients and can evict them for:

### Eviction triggers:
| Trigger | Code | Action |
|---------|------|--------|
| Session age > 24h | `AuditActivePlayerSessions()` | `DisconnectClient(SESSION_EXPIRED)` |
| On-chain balance < 0.1 VOI | `AuditActivePlayerSessions()` | `DisconnectClient(INSUFFICIENT_LIQUIDITY)` |
| Gloat spam (30 msg/10s) | `allowMessage()` | Message drop |
| Rate limit exceeded | `RateLimiterService` | HTTP 429 |
| Unauthorized origin | `websocket.CheckOrigin` | WS rejection |

### Eviction flow (client-side):
```go
// main.go:110-148 (js/wasm)
func (crm *ClientRedirectManager) HandleIncomingEvictionFrame(rawFramePayload []byte) bool {
    // 1. Instantly freeze interface controls
    crm.lockCanvasInteractions()
    // 2. Map system eviction codes to alert
    crm.Alerts.DispatchFriendlyAlert(mappingCode)
    // 3. Purge Local Memory
    crm.flushLocalMemoryHeap()
    // 4. Delayed Redirection (4 Seconds)
    crm.GlobalWindow.Call("setTimeout", redirectFunc, 4000)
    return true
}
```

---

## 6. THE LEDGER ISOLATION — SERVER OWNS ALL VALUE

### Server-side (linux/amd64):
```go
// backend_types.go:469-512
type Lobby struct {
    playerBalances  map[string]uint64  // THE authoritative ledger
    rewardStack     map[string]uint64  // Pending rewards
    initialRewards  map[string]uint64  // Initial reward config
    treasuryAverages map[string]float64 // Display only
    treasuryCrashed  map[string]bool    // Crash flags
}
```

### Client-side (js/wasm):
- NEVER holds a balance directly
- Receives `vault_update` WS messages → updates local display
- Receives `rewards_update` WS messages → updates reward display
- All spending is a REQUEST to the server
- Server validates: `if l.playerBalances[spectatorWallet] < req.WagerMicro { http.Error(...) }`

### The flow:
```
Client wants to place wager (100 μVBV)
    │
    ▼
Client sends: {type: "match_wager", payload: {wager_micro: 100}}
    │
    ▼
Server: handleSpectatorWager()
    ├─ if l.playerBalances[spectatorWallet] < req.WagerMicro → REJECT
    ├─ l.playerBalances[spectatorWallet] -= req.WagerMicro → DEDUCT
    ├─ match.WagersMicro += req.WagerMicro → ADD TO POT
    └─ logAdminAuditLocked("SPECTATOR_WAGER", ...) → AUDIT TRAIL
    │
    ▼
Client receives: {type: "vault_update", payload: {balance: new_balance}}
Client updates display ONLY — never the source of truth
```

---

## 7. THE ANTI-CHEAT SUMMARY

| Attack Vector | Defense |
|---|---|
| **Fabricate win** | Server `verifyWinner()` is the only authority |
| **Alter card power** | `getEffectiveServerPower()` runs server-side with snapshotted stats |
| **Modify balance** | `playerBalances` lives on server; client only sees mirror via `vault_update` |
| **Skip/reorder turns** | SequenceID must be continuous; gaps trigger `InitiateRecovery()` |
| **Tamper board state** | SHA-256 hash of board verified after every move |
| **Replay old moves** | `VerificationHook` tracks consumed nonces with TTL |
| **Spam/DoS** | `RateLimiterService` per-wallet + `allowMessage()` 30msg/10s |
| **Session hijack** | `SessionWatchdog` evicts on 24h timeout or liquidity < 0.1 VOI |
| **Origin spoofing** | `websocket.CheckOrigin` validates `ALLOWED_ORIGINS` env |
| **Sybil attack** | `OnboardingService` + indexer wallet history check |
| **Mid-match stat race** | All stats snapshotted at match start in `MatchState` |
| **Client-side desync** | `ClientReplayEngine` detects hash mismatch → UI freeze → catch-up |

---

## 8. THE CONSTITUTIONAL IMPLICATIONS

### Why the split WASM exists:
1. **Security through physical isolation** — The authoritative state lives in a binary that never touches the browser. Even if you hack the client WASM, you can't touch the ledger.
2. **Deterministic verification** — Both WASMs can compute the same SHA-256 hash from the same board state, proving the client hasn't tampered.
3. **Replay resilience** — If the client desyncs (or cheats), the server detects it via hash mismatch and forces a full state reconstruction.
4. **Session integrity** — The server can evict any client at any time; the client must obey (lock canvas, flush memory, redirect).

### Why this matters for development:
- **Never put ledger logic in `main.go` (js/wasm).** It will be compiled away in the server build.
- **Never trust client-side state for economic decisions.** Always re-validate on server.
- **Never assume the client has applied a frame correctly.** The hash verification is the only proof.
- **Never skip the snapshot.** If you add a new PlayerStats field that affects battles, snapshot it in `initiatePairedMatch()`.

---

## 9. THE BUILD TAG BOUNDARY

```go
// server.go, lobby_manager.go, battle_service.go, etc.
//go:build !js && !wasm
```
These files are **ONLY** in the server binary. They are **NEVER** compiled into the client WASM. The client cannot import them, cannot call their functions, cannot access their state.

```go
// main.go
//go:build js && wasm
```
This file is **ONLY** in the client WASM. It contains the `Engine`, `ClientReplayEngine`, `WasmSignerHook`, `ClientRedirectManager` — all interaction-layer code.

```go
// common_types.go
//go:build !js && !wasm
```
Shared types for the server. The client has its OWN copy in `common_types_wasm.go` (manually kept in sync).

### The boundary is enforced by the Go compiler:
- If you try to import a `!js && !wasm` file from a `js && wasm` file → **compile error**
- If you try to use `js.Syscall` in a `!js && !wasm` file → **compile error**
- The build tags are not conventions — they are **hard security boundaries**

---

## 10. THE CRITICAL FILES FOR SECURITY

| File | Role | Security Function |
|---|---|---|
| `battle_service.go` | `serverCheckCaptures()`, `verifyWinner()`, `ComputeBoardHash()`, `getEffectiveServerPower()` | Card battle authority |
| `lobby_manager.go` | `initiatePairedMatch()`, `SyncHandshaker`, `handleAuthoritativeForfeit()` | Match lifecycle + frame verification |
| `server.go` | `newLobby()`, all service init | Bootstrap authority |
| `economy_service.go` | `applyDynamicScalingLocked()`, balance mutations | Ledger authority |
| `economy_audit.go` | `InterceptAndAudit()` | Invariant enforcement |
| `onboarding_service.go` | `AuditActivePlayerSessions()` | Session eviction |
| `main.go` (js/wasm) | `ClientReplayEngine`, `SyncMove()` hash verification | Client-side desync detection |
| `common_types.go` | `BoardStateHash`, `AuthoritativeFrame` | Wire format for verification |

---

*End of Security Architecture Deep-Dive. This is the #1 concept to understand before touching any code.*
