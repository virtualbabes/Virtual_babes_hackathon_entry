# NFT-Seduction: Orphan Analysis & Implementation Plan

## 1. ARCHITECTURAL OVERVIEW (System Map)

The codebase is a **Social Economic Simulation** card battler on the Voi Network with these core layers:

### Layer 1: Entry & Server Bootstrap (`main.go`, `server.go`)
- `main.go`: Binary entry point, HTTP server bootstrap, WASM Engine initialization
- `server.go`: Route registration, service kernel initialization, periodic ticker bootstrapping

### Layer 2: Core Game Loop (`lobby_manager.go`, `backend_types.go`, `common_types.go`)
- `lobby_manager.go`: **Central orchestrator** - matchmaking, protocol dispatch, spectator handling, tournament state
- `backend_types.go`: Backend-specific type definitions, persistence methods
- `common_types.go`: **Shared bridge types** between Go backend, WASM engine, and JS frontend

### Layer 3: Domain-Specialized Services (Decomposed from lobby_manager)
| Service | File(s) | Responsibility |
|---------|---------|----------------|
| Battle | `battle_service.go` | Combat rules, winner verification, sudden death, fatigue, card capture tracking |
| Economy | `economy_service.go`, `economy_bootstrap.go`, `economy_processing.go`, `economy_audit.go`, `economy_persistence.go`, `economy_telemetry.go` | Dynamic scaling, AMM bonding curves, token-sink routing, liquidity audits, state persistence sync, telemetry export |
| Club | `club_service.go`, `club_service_test.go` | Club creation/joining, foundry/industry, territory purchase, lease management, shop revenue distribution, heist defense |
| Auction | `auction_service.go` | Art Gallery auctions, bid handling, escrow (internal), commission routing |
| Black Market | `black_market_service.go` | Fencing, AMM pricing, cunning/wanted gated items |
| Loan | `loan_service.go` | Loan origination, default processing, collateral management |
| Faucet | `faucet_service.go` | $VBV faucet rewards, dynamic scaling, granular opt-in checks |
| Oracle | `oracle_service.go` | Voi/Algorand indexer communication, card discovery, buy-in verification, Sybil scans, blockchain state snapshots |
| Onboarding | `onboarding_service.go` | Player registration, Sybil protection, session watchdog, continuous eligibility monitoring |
| Employment | `employment_service.go`, `career.go` | Career hiring, salary dispatch, club employment tracking |
| Achievement | `achievement_service.go` | Trophy/achievement logic (GOVERNOR, ART_COLLECTOR, HEIST_SABOTEUR, etc.) |
| Courthouse | `courthouse_service.go` | Fines, bailing, prisoner processing, governor revenue routing |
| Tournament | `tournament_manager.go` | Bracket automation, sudden death, prize distribution, buy-in verification |
| Narrative | `narrative_service.go` | NPC taunts, commentary generation, global sentiments |
| Item | `item_service.go`, `shop_registry.go` | Faceplates, artifacts, cyber-audit/jammer/sabotage items, shop catalog registry |
| Bridge | `bridge_service.go` | Placeholder for future expansion (currently unused) |

### Layer 4: Handler Layers (HTTP/WebSocket API endpoints)
| Handler | File(s) | Routes |
|---------|---------|--------|
| Public | `handlers_public.go` | Health, leaderboard, economy state, lobby updates |
| Admin | `handlers_admin.go` | Ban, tournament control, vault management, emergency shutdown, maintenance mode, ledger audit |
| Criminality | `handlers_criminality.go` | Kidnap gambit, ransom, bounty, hostage, insurance recovery |
| Rumor | `handlers_rumor.go` | Spread rumor, rumor processing |

### Layer 5: WASM Deterministic Engine (`Public/wasm_exec.js` + Go WASM source)
- WASM source files are **not present in current repo** (compiled externally)
- WASM exposes: `GetGameState`, `ApplyArtifactToBoard`, `ToggleLeaderboard`, `ClientReplayEngine`, `WasmSignerHook`, `AlertEngine`, `ClientRedirectManager`
- WASM Engine handles: deterministic combat rules (Same/Plus/Combo), board hashing, replay frame sync

### Layer 6: Frontend Orchestrator & Domain Modules (`Public/js/*.js`)
| Module | Responsibility |
|--------|----------------|
| `app.js` | **Primary frontend orchestrator** - imports all modules, delegates to domain modules, `syncUI` state sync |
| `network.js` | WebSocket client, reconnection, message routing to WASM engine and JS modules |
| `economy.js` | AMM trading UI, Black Market interface, auction/bidding overlay, lease board, portfolio view |
| `ui.js` | Card rendering (`renderCardHTML`), tooltip system (`showPowerTooltip`), deck manager, map zoom, District Scanner timer |
| `game.js` | Combat interactions, move execution, match state management, power calculation display |
| `deck.js` | Deck building, card selection, hand management |
| `wallet.js` | WalletConnect integration, multi-chain signature handling, connection state |
| `criminality.js` | Bounty board UI, alliance hub, heist planning terminal, kidney gambit interface |
| `leaderboard.js` | Rankings display, achievement tracking |
| `particles.js` | Canvas-based particle effects (captures, heists, foundry fusion) |
| `audio.js` | Low-latency SFX via AudioContext, music tracks, volume controls |
| `audio_context.js` | Audio context initialization manager |
| `utils.js` | Shared utilities, cache management, address shortening |
| `collective-intelligence.js` | NPC behavior tendencies, narrative personality engine |

### Layer 7: SCSS Styles (`Public/src/scss/`)
- `_variables.scss`, `_reset.scss`, `_typography.scss`: Foundation tokens
- `_buttons.scss`, `_cards.scss`, `_overlays.scss`: Component styles
- `_criminality.scss`, `_economy.scss`, `_shops.scss`, `_social.scss`, `_territory.scss`: Feature modules
- `_dashboard.scss`, `_main-layout.scss`: Layout templates
- `_neon-glass.scss`: Themed aesthetic
- `_animations.scss`, `_spacing.scss`: Utility classes

### Layer 8: Blockchain/Persistence Infrastructure
- `networks.json`: RPC endpoints for Voi/Algorand/EVM/Solana
- `economy_bootstrap.go`: `BootstrapAuthoritativeState` - initial state reconstruction from blockchain snapshots
- `economy_persistence.go`: `PersistenceSyncWorker`, backup rotator, periodic disk snapshots
- `economy_audit.go`: `InterceptAndAudit`, TokenSinkRouter, `TokenSinkAuditReporter`
- `economy_telemetry.go`: Prometheus metrics exporter (port 9090)
- `redemption_gateway.go`: Token redemption/exchange gateway
- `resilience_utils.go`: Retry policies, circuit breaker utilities

### Layer 9: AI Brain (`AI-Brain/`)
- `Rules.md`: Behavioral rules for implementation
- `ToDo.md`: Active engineering priorities (task completion checklist)
- `A.I_memory.md`: Authoritative forensic ledger of all completed tasks
- `File-Flow-Overview-1.md`: Architecture blueprint / Mermaid maps
- `DIR.md`: Directory structure documentation
- `Problems.md`: Pending issues and bugs

---

## 2. IDENTIFIED ORPHANS (Dead Code & Unwired Assets)

### Category A: Completely Dead/Unused Files

1. **`bridge_service.go`** - Stated in memory as "Placeholder for future expansion; current onboarding is in `onboarding_service.go`". Never wired to any handler or service init.

2. **`career.go`** - Partially migrated to `employment_service.go`. The memory docs show the migration was completed (Task 383: "Refactored Career API handlers to `employment_service.go`"). Need to verify what functionality remains in career.go vs employment_service.go.

### Category B: Partially Wired / Incomplete Features

3. **`rival_career_engine.go`** - Has file but no clear wiring path in the File Flow map. No handler references it directly.

4. **`nautilus_dex_path.go`** - Exists but unclear if wired to actual trading or auction systems. May be dead code.

5. **`deployment-wasm.yml`** / **`Dockerfile`** + **`entrypoint.sh`** - Deploy infrastructure present, verify these are current and not stale.

6. **`tournament_manager.go` visible in tabs but user has it open** - Check if this is fully functional or has remaining wiring gaps.

### Category C: Unwired Audio Assets (from ToDo.md, Phase 6)

7. **14 orphaned ambient audio tracks** - Stated as "Deferred" with implementation plan documented but not executed:
   - Menu Ambients (4 files): `ambient_menu_music_1-4.mp3`
   - Quick Play (3 files): `quick_play_ambient_1-3.mp3`
   - 2-Player Casual (3 files): `2_player_ambient_1-3.mp3`
   - Tournament Bracket (4 files): `Tournament_game_ambient*.mp3`
   - Miscellaneous (2 files): `Not_connected_ambient.mp3`, `Unbuilt_deck_ambient.mp3`

### Category D: JS Module Audit Findings

8. **`admin.js`** - Referenced in memory docs extensively but NOT visible in root dir listing. Need to verify if it exists under a subdirectory or is missing.

9. **`audio_context.js`** - Listed in js/ folder, separate from `audio.js`. Need to verify integration status.

### Category E: Documentation Orphans

10. **`Development-production-build/Markdown_developer_volume/`** - Contains expanded planning docs but these are not actively referenced by the AI Brain system.

---

## 3. IMPLEMENTATION PLAN

### Phase 1: Dead Code Cleanup (Tasks 1-4)
**Priority: HIGH | Impact: Build stability + maintainability**

#### Task 1: Audit & Remove `bridge_service.go`
- **Action**: Verify no imports of `bridge_service.go`, then remove or archive it
- **Risk**: Zero (memory confirms it's a placeholder)
- **Files**: Delete `bridge_service.go`

#### Task 2: Audit `career.go` vs `employment_service.go`
- **Action**: Read both files, determine remaining functionality in career.go
- **If orphaned**: Remove career.go
- **If partially functional**: Complete migration to employment_service.go

#### Task 3: Audit `rival_career_engine.go`
- **Action**: Search for imports/references across the codebase
- **If no references**: Archive or remove
- **If partially wired**: Identify missing integration points

#### Task 4: Audit `nautilus_dex_path.go`
- **Action**: Search for import/use references
- **If dead**: Remove
- **If functional**: Wire into economy/market systems if applicable

### Phase 2: Feature Completion (Tasks 5-8)
**Priority: MEDIUM | Impact: Immersion + UX polish**

#### Task 5: Implement Audio Context Manager
- **Action**: Create `Public/js/audio_context_manager.js` (or add to existing audio_context.js):
  - Detect game phase transitions (menu/lobby/combat/heist/tournament)
  - Cross-fade ambient tracks based on current phase
  - User toggle for "Contextual Ambients" in settings
- **Integration**: Call from `app.js:syncUI()` when game state changes

#### Task 6: Wire All 14 Orphaned Ambient Tracks
- **Action**: In audio context manager, register each track with its trigger condition:
  - Menu phases → `ambient_menu_music_1-4.mp3` (random rotation)
  - Quick play lobby → `quick_play_ambient_1-3.mp3`
  - Casual matches → `2_player_ambient_1-3.mp3`
  - Tournament bracket views → `Tournament_game_ambient*.mp3`
  - Edge cases → `Not_connected_ambient.mp3`, `Unbuilt_deck_ambient.mp3`

#### Task 7: Verify & Complete `admin.js` (if missing)
- **Action**: Check if `admin.js` exists in Public/js/ or elsewhere
- **If missing**: Create from memory doc references (Admin Panel, Force Payout, Asset Forfeiture, Cyber-Security Audit UIs)
- **If exists but broken**: Fix import chain from `app.js`

### Phase 3: System Verification (Tasks 8-10)
**Priority: MEDIUM | Impact: Build integrity confirmation**

#### Task 8: Full Build Verification
- **Action**: Run `go build` and `npm run build` to confirm clean compilation
- **If errors**: Fix each error systematically

#### Task 9: Handler Wiring Audit
- **Action**: Verify all service methods are wired through handlers in `server.go`:
  - Check that every service's public methods have corresponding route registrations
  - Cross-reference with memory docs' "Protocol Consolidation" tasks

#### Task 10: WASM Source Verification
- **Action**: Check if WASM Go source exists (likely external build process)
- **If present**: Verify its game loop matches the documented deterministic rules
- **If absent**: Document build process requirements

### Phase 4: Documentation Synchronization (Tasks 11-12)
**Priority: LOW | Impact: Maintenance clarity**

#### Task 11: Update `File-Flow-Overview-1.md`
- **Action**: Verify all Mermaid diagrams match current service topology
- **Focus areas**: Tournament flow, Economic loop, Criminality path

#### Task 12: Clean `AI-Brain/` organization
- **Action**: Archive completed tasks from `A.I_memory.md` if any remain that aren't yet in `ToDo.md`
- **Action**: Remove any stale development docs from `Development-production-build/`

---

## 4. TASK PRIORITY MATRIX

| Priority | Task | Effort | Risk |
|----------|------|--------|------|
| P0 (Critical) | Tasks 1-2: Dead code removal | Low | Low |
| P0 (Critical) | Task 8: Build verification | Low | Low |
| P1 (High) | Task 3: Rival career engine audit | Medium | Low |
| P1 (High) | Task 4: Nautilus DEX audit | Medium | Low |
| P2 (Medium) | Tasks 5-6: Audio context manager + ambient wiring | High | Low |
| P2 (Medium) | Task 7: Admin.js completion/fix | Medium | Medium |
| P3 (Low) | Task 11-12: Documentation sync | Low | Low |

---

## 5. KEY INTERACTION PATTERNS (For Implementation Reference)

### Game State Flow
```
Client (app.js) → WebSocket → Server (lobby_manager.go)
                                    ↓
                        Protocol dispatch to specialized services
                                    ↓
                    Service modifies Lobby state (via mutex)
                                    ↓
                    Economy processing (scaling, routing)
                                    ↓
                    WASM Engine sync (deterministic combat)
                                    ↓
                    Broadcast lobby_update via WebSocket
                                    ↓
                Client syncUI() → diff-based DOM updates
```

### Economic Loop
```
Action (trade/buy/rent/heist) 
    → Service handler
    → TokenSinkRouter (route fees to Faucet/Treasury/Governor)
    → applyDynamicScalingLocked (recalculate gas floor)
    → CheckVaultBalanceOnChain (verify physical backing)
    → Broadcast liquidity update
```

### Tournament Flow
```
Registration → Buy-in verification (oracle)
    → Bracket automation (tournament_manager.go)
    → Match pairing + WebSocket challenge
    → Deterministic combat (WASM)
    → Winner verification (battle_service.go)
    → Next round advance / Sudden Death if tied
    → Prize distribution (faucet, governor tax, club kickback)
```

---

**This plan is ready for review. Please confirm which phases you want to execute first.**
