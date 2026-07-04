# Virtualbabes Arena: Orphan Fix & Logic Preservation Ledger

**Alignment Note:** All fixes recorded here must be cross-referenced with the authoritative `A.I_memory.md` completion log. Maintenance tasks are derived from the active `ToDo.md` roadmap.
**Structural Baseline:** Refer to `File-Flow-Overview-1.md` (Blueprint) to ensure logic preservation matches system topology.

## 1. Protected Placeholders (Do Not Remove)
These files or code blocks may appear orphaned but are strategic hooks for future expansion.
*   **`bridge_service.go`**: Active IPC Bridge. Defines the explicit IPC (Inter-Process Communication) translation registry between Go WASM and Browser JavaScript (Pillar 4). Logic for onboarding was migrated to `onboarding_service.go` to keep the bridge scope pure.
*   **`isPerishable` flag (`club_service.go`)**: Placeholder within the shop revenue distribution logic. Will eventually handle different tax implications for consumable items (food/meds).
*   **`Console Hub`**: Strategic placeholder for the Phase 4 strictly non-crypto client environment integration.
*   **`TournamentID` vs `MatchID` redundancy**: Both are required in `MatchState` to support deep verification of tournament history while maintaining ephemeral session integrity.
*   **`VirtualBalance` in UI**: Appears redundant with physical balance but is the primary accumulator for non-battle rewards (Salaries, Heists) before they are committed to the on-chain vault.

## 2. Pillar 3: Intelligence & Administrative Repairs
Logic restored and hardened during the social/criminal expansion phase.
*   **Recursive Deadlock Resolution**: Systemic repair across `club_service.go`, `handlers_criminality.go`, and `handlers_admin.go`. Switched to `Locked` variant helpers (e.g., `sendToClientLocked`, `logAdminAuditLocked`) to prevent RLock/Lock conflicts during high-frequency events.
*   **Micro-Unit Precision Math**: Hardened rounding logic for:
    *   **Salaries (`career.go`)**: Round-to-nearest micro-unit for net payout and Outlaw Tax.
    *   **Heist Loot (`club_service.go`)**: 10% Fence Fee redistribution.
    *   **Bail Payments (`handlers_criminality.go`)**: Distribution of bail funds to club treasuries.
    *   **Market Trading (`market_service.go`)**: `totalValueMicro` calculation using integer math to match tournament payout parity.
    *   **Kidnap Gambit (`handlers_criminality.go`)**: Transitioned to `VictimRegistry` for multi-slot attacker isolation, eliminating the self-kidnap immunity exploit.
*   **Mojo Lifecycle Restoration**:
    *   Implemented `calculateMojoGain` in `club_service.go` to reward revenue, defense, and jailing.
    *   Hardened `processMojoDecay` in `lobby_manager.go` to include the **District Stabilizer** mitigation field.
*   **Black Market Restoration**:
    *   Restored truncated `black_market_service.go` and implemented `HandleSellToBlackMarket` to complete the Silkroad loop (Task 2182).
*   **Intelligence Field Enforcement**:
    *   `Cyber-Audit` logic in `item_service.go` checks for `CYBER_LOCK` and allows bypass via `SABOTAGE`.
    *   `Cyber-Lock` enforcement in `club_service.go` blocks heist initiation unless the target club is sabotaged.
    *   `Cyber-Counter` identity revelation is single-use; the trap is now correctly pruned after the notification is dispatched.
    *   `Sabotage Warning` system in `club_service.go` alerts all connected club members via chat when defenses are compromised.
    *   `Sabotage Penalty` in `economy_service.go` reduces employee Reputation by 20% during active breach windows.
    *   `Cyber-Jammer` in `club_service.go` suppresses the Sabotage Warning for a single attempt.

## 3. Pillar 1 & 2: Infrastructure & High-Finance Fixes
*   **Sybil Protection Restoration**: `loadOnboardedWalletsFromIndexer` was implemented to reconstruct the historical claim map from blockchain data on server restart.
*   **Dual-Ledger Model**: Separated the physical vault balance (`faucetBalance`) from virtual reward liabilities (`playerBalances`).
*   **Insurance Recovery Logic**: Restored the 48-hour automated return cycle for kidnapped cards in `handlers_criminality.go`.
*   **Auction Escrow Model**: Hardened `transferBundleItems` in `auction_service.go` to ensure CardIDs are properly escrowed and returned on expiration or outbid.
*   **Envoi Name Resolution**: Migrated to a dedicated `envoiMutex` and a 500-entry negative cache to prevent I/O deadlocks and indexer spam during lobby updates.

## 4. Active Strategic Debt (Monitor)
*   **Tournament BYE Handling**: Logic in `determineTop5` handles uneven brackets by defaulting to Reputation for rank 3-5. This is stable but should be revisited if bracket sizes exceed 16.
*   **Achievement Persistence**: Currently relies on `saveLeaderboard` (VBT_STATE_SNAPSHOT). If state reconstruction takes longer than 30s, UI may show "Nobody" rank briefly.
*   **HTML Escaping**: Applied to Envoi names and Player Reports. Ensure any new chat triggers utilize `escapeHTML` to prevent XSS.
*   **Blockchain-Native State**: All authoritative state (Leaderboard, Economy, Linked Wallets, Registered TxIDs) is now exclusively paged from blockchain notes. Local disk persistence is deprecated for core game state.

## 5. Deployment Hardware Alignment
*   **`DATA_DIR` Pathing**: Refactored all service files to use `l.getDataPath()` to support Render persistent volume mounts.
*   **Sequence Carry-over**: Fixed `initiatePairedMatch` to explicitly reset `matchHandshakers` for new matches, preventing false desync detections from previous session history.
*   **Hand Flickering**: Refactored `GetGameState` to scope `deck` assets via `LocalPlayerIndex` instead of `Turn`, ensuring visual stability during opponent moves.
*   **Identity Normalization**: Standardized `strings.ToLower` across Auction, Loan, and Black Market entry points to prevent duplicate portfolio creation.
*   **Grace Matrix Leaks**: Refactored `handleUnregister` to prune non-combatant sessions from the `GracePeriodMatrix`, preventing memory accumulation during long server uptimes.
*   **Alliance Blackouts**: Updated `HandleAllianceDissolve` to clear `DISRUPTION_` tags, ensuring coordinated-defense sabotage is purged when an alliance is terminated.
*   **Tournament Kickbacks**: Migrated kickback logic to `ClubService` and integrated the `TokenSinkRouter` to enforce the Industrial Seal on competitive revenue.
*   **Bail Payments**: Migrated `handleBailCard` to the `TokenSinkRouter`, ensuring underworld revenue is vetted and audited with micro-unit precision.
*   **Heist Fence Fees**: Migrated heist loot redistribution in `club_service.go` to the `TokenSinkRouter` for centralized accounting and organizational sync.
*   **Territory Purchase Fees**: Migrated territory acquisition inflows to the `TokenSinkRouter`, enforcing the 5% Regional Governor protocol fee through vetted audit logs.
*   **Governor Surcharges**: Migrated tax policy surcharges to the `TokenSinkRouter`, ensuring political influence fees are vetted with micro-unit precision.
*   **Non-Custodial Pulls**: Integrated `AUCTION_PULL` and `LOAN_AUTO_PULL` into the `AuditReporter` kernel for 100% transactional visibility.
*   **Mojo Auditing**: Wired item-based Mojo gains into the `admin_audit.log` to ensure organizational growth is forensic and transparent.
*   **`clickGrid` Interaction Bridge**: Hardened `game.js` to prevent crashes when WASM engine is not fully initialized and ensured card IDs are passed as integers.
*   **Sanity Check RPC Cluster**: Refactored `handleSystemSanityCheck` to use the `LoadBalancedLedgerClient` instead of a static endpoint.
*   **`setTransactionStatus` Critical Types**: Hardened UI feedback to apply bold, uppercase, and tracking styles for critical admin-level local notifications.
*   **Critical Alert Animations**: Implemented pulsing animations in `_main-layout.scss` for `priority-critical` and `priority-warning` status states.
*   **Administrative Feedback**: Hardened `admin.js` to ensure all command failures are reflected in the `transaction-status` HUD with the `critical` priority level.
*   **Combat Grid Rendering**: Restored granular node-diffing to `app.js:syncUI` to ensure cards are rendered into slots without overwriting the grid's interaction listeners.
*   **WASM Score Parity**: Updated `checkWinConditionLocked` in `main.go` to account for hand cards, ensuring deterministic finalization during catch-up replays.
*   **Cryptographic Board Parity**: Implemented matching `ComputeBoardHash` functions in `battle_service.go` and `main.go` to enable state verification.
*   **main.go Structural Repair**: Fixed dangling code snippets and missing imports in the WASM core to restore compilation.
*   **Real-time Cryptographic Sync**: Hardened `SyncMove` in `main.go` to perform state hash verification, ensuring parity with the Replay Kernel.
*   **Hash Consolidation**: Removed legacy `ComputeStateHash` from `lobby_manager.go` and unified `BoardStateHash` type into `common_types.go` to eliminate compilation redundancy.
*   **PlayerStats Entropy**: Resolved catastrophic field duplication and tag drift in `common_types.go` to restore compilation and bridge stability.
*   **SyncMove Deadlock Fix**: Refactored `SyncMove` to check sequence authority before acquiring game state locks, preventing catch-up deadlocks.
*   **WASM Sync Tags**: Aligned `Player` struct tags in `main.go` with `common_types.go` and restored missing economic fields to the `SyncFullProfile` loop.
*   **Tournament Match Tag**: Synchronized `TournamentMatch` struct to use `match_id` JSON tag for consistency across bracket and history models.
*   **Match Identification UI**: Updated `generateBracketHTML` and `updateSpectatorHUD` in `ui.js` to utilize the standardized `match_id` property.
*   **WASM Match ID Sync**: Added `TournamentMatchID` to WASM engine in `main.go` and hardened `lobby_manager.go` to ensure high-fidelity match identification in the Tactical HUD.
*   **Beacon Match Identification**: Expanded `app.js` state beacon to include `tournament_match_id` and supported restoration during active match warm-boots.
*   **Matchmaking Persistence**: Refactored `SyncFullProfile` in `main.go` to correctly restore `Game.InMatchmakingQueue` from the state beacon.
*   **PlayerStats Entropy**: Resolved catastrophic field duplication and tag drift in `common_types.go` to restore compilation and bridge stability.
*   **WASM Sync Tags**: Aligned `Player` struct tags in `main.go` with `common_types.go` and restored missing economic fields to the `SyncFullProfile` loop.
*   **Tournament Match Tag**: Synchronized `TournamentMatch` struct to use `match_id` JSON tag for consistency across bracket and history models.
*   **Tournament History Identity**: Hardened `fetchTournamentHistory` in `leaderboard.js` to batch-resolve Envoi names for archived match participants.
*   **Market Ticker Corruption**: Resolved duplicate `updateMarketTicker` declarations and structural entropy in `economy.js` to restore module stability.
*   **Quick-Cast Desync**: Refactored `ApplyArtifactToBoard` in `main.go` to synchronize the cryptographic benchmark with board mutations, preventing false recovery triggers.
*   **Match Re-entry**: Implemented `rejoinActiveMatch` in `game.js` and wired it into `app.js` to restore interrupted combat sessions via the Replay Engine.
*   **Selection Race Condition**: Hardened `selectCard` in `game.js` to ensure atomic variable update and integer casting before UI re-renders.
*   **Selection Glow Ghosting**: Integrated selection state into `renderCardHTML` (ui.js) with context-aware memoization to ensure consistent interaction feedback across hand and board.
*   **Selection Reset Pulse**: Refactored `clickGrid` in `game.js` to reset `activeCardId` immediately after successful placement, ensuring visual clarity during network broadcasts.
*   **Deck Manager Selection**: Excised redundant manual class manipulation in `deck.js` to enforce `renderCardHTML` as the single point of truth for card state rendering.
*   **Quick-Cast Redundancy**: Hardened `showQuickCastMenu` in `ui.js` to filter out board-deployed assets and "in-flight" items using `pendingQuickCastId`, preventing duplicate use-item events.

## 6. Missing Files (Expansion Targets)
Future files required for Phase 4 Console Expansion that are not yet present in the repository:
*   **`redemption_gateway.go`**: (REPAIRED) Migrated from `api/v1/` to root and fully implemented creator payout logic (Task 2097).
*   **`nautilus_dex_path`**: Future specialized service or logic path for executing server-side market-buys of $VBV to pay browser suppliers for redeemed console DLC.
*   **Spectator Stream Catch-up**: Hardened `handleMove` to update `LastActiveFrame` for spectators and extended the grace period matrix to non-combatant match viewers.
*   **Quick-Cast Deadlock**: Refactored `ApplyArtifactToBoard` in `main.go` to follow the hierarchical lock order (cre.Mu -> Game.mutex), preventing WASM engine hangs during item usage.
*   **Server Hash Return**: Fixed broken `return` in `battle_service.go:serverCheckCaptures` to ensure the authoritative state hash is broadcasted to clients.
*   **Quick-Cast Stale Animation**: Hardened `ApplyArtifactToBoard` in `main.go` to reset `IsCombo` flags, preventing redundant flip animations during item usage.
*   **Spectator Sync Gap**: Hardened `handleSpectate` and `sync_request` in `lobby_manager.go` to ensure handshaker availability and guaranteed responses for viewers.
*   **Warm-Boot Recovery Optimization**: Hardened `rejoinActiveMatch` to skip redundant sync requests if beacon hydration results in a synchronized engine state.
*   **Replay Progress Fidelity**: Verified `ProcessedInSession` reset in `main.go:InitiateRecovery` to ensure accurate visual feedback during catch-up cycles.
*   **Recovery Freeze Fix**: Implemented `CompleteRecovery` (main.go/network.js) to prevent UI hangs when catch-up requests return 0 frames.
*   **Speculative Move Guard**: Hardened `clickGrid` in `game.js` to validate `replay_state`, preventing interactions during Replay Recovery.
*   **Tactical Intel Guard**: Hardened `updateMapStatusIndicators` and `openTerritoryMapOverlay` in `ui.js` to suppress trap visibility during catch-up.
*   **Zero-Hash Verification Bypass**: Hardened WASM verification in `main.go` to ignore `StateHash` if it is zero, supporting legacy spectator-only matches.
*   **Inventory State Bloat**: Hardened `ImportARC72Card` in `main.go` to handle duplicate IDs, ensuring the global inventory remains unique across discovery pulses.
*   **Underworld Atmosphere cleanup**: Moved `criminal-underworld` toggle to `ui.js:updateDynamicArenaFloor` and gated by phase to ensure class is cleared upon match conclusion.
*   **Session Identity Borders**: Implemented `updateAvatarIdentityStyle` in `ui.js` and hardened visual helpers to respect `local_player_index`, resolving hardcoded P1-only effects.
*   **Challenge Identity Hardening**: Standardized `avatar_url` and `gloat_message` export in WASM and refactored `game.js` challenge protocols to utilize them, eliminating hardcoded P1 metadata.
*   **Challenge Decline Feedback**: Hardened `declineChallenge` in `game.js` to include local UI notification and verified atomic variable cleanup.
*   **WASM State Redundancy**: Finalized excision of duplicate field assignments in `main.go` (`local_player_index`, `maintenance`, `HasMutationInsurance`) to optimize serialization.
*   **WASM Metadata Duplication**: Resolved structural redundancy in `main.go:SetBoardState` and completed `match_id` integration in `SyncMatchMetadata`.
*   **Spectator Interaction Leak**: Hardened `sendSpectate` in `game.js` to reset `activeCardId` and `pendingQuickCastId`, ensuring local selection states do not persist when viewing remote matches.
*   **Spectator HUD Resilience**: Hardened `updateSpectatorHUD` in `ui.js` with inner null-guards for `active_item_buffs` to prevent synergy calculation failures.
*   **Mutation Scar Overlays**: Implemented the injection loop in `ui.js:renderCardHTML` (Task 1975) and defined high-translucency `.card-scar-overlay` styling in `_cards.scss` (Task 1976).
*   **Board Card Identity**: Hardened `updateAvatarIdentityStyle` in `ui.js` to apply `local-owner-slot` classes to grid slots (Task 1981) and finalized neon-cyan pulse styling in `_cards.scss` following application failure (Task 1991).
*   **Beacon Identity Sync**: Hardened `vbabes_state_beacon` in `app.js` to include and restore `local_player_index`, ensuring identity styling persists across page refreshes (Task 2001).
*   **Lobby Identity Sync**: Globalized `local_player_index` in `GetGameState` (`main.go`) and hardened beacon restoration in `app.js` to ensure styling continuity.
*   **Challenge Race Condition**: Hardened `acceptChallenge` and `network.js` to ensure local engine initialization precedes network dispatch, preventing phase-mismatch errors.
*   **Audio Concurrency**: Hardened `audio.js` with a master compressor and promise-based buffer caching to handle high-frequency item usage without clipping or redundant I/O.
*   **Interaction Feedback**: Hardened `clickGrid` in `game.js` to provide warning toasts on placement failure, while preserving the active selection.
*   **Capture Feedback Loop**: Implemented `.flip-capture` in `_cards.scss` to provide visual weight to board-flip events.
*   **WASM Pointer Safety**: Refactored `PlaceCard` and `PerformAIMove` in `main.go` to use heap-allocated copies for board cards, preventing pointer corruption after hand/deck mutation.
*   **Battle Board Perspective**: Implemented `perspective: 1000px` in `_dashboard.scss` and removed redundant CSS definitions to enable smooth 3D card flips.
*   **Share Trade Fees**: Migrated Entity Market trading fees to the `TokenSinkRouter` with `"SHARE_TRADE_FEE"` context for forensic auditing.
*   **Ransom Taxes**: Migrated ransom laundering fees to the `TokenSinkRouter`, ensuring criminal activity taxes are vetted with micro-unit precision.
*   **Blackout Enforcement**: Hardened `initiatePairedMatch` to check for `DISRUPTION_` tags before applying Regional or Coalition power boosts.
*   **CORS Hardening**: `CheckOrigin` in `server.go` is now strictly linked to the `ALLOWED_ORIGINS` environment variable.
*   **Economy Persistence (`economy_persistence.go`)**: Implemented atomic .tmp swap pattern to prevent state corruption during hardware failure.
*   **Struct Duplication**: Removed duplicate fields in `PlayerStats` within `common_types.go` to ensure successful isomorphic compilation.
*   **WASM Build Syntax**: Noted PowerShell-specific requirements for environment variable injection to prevent terminal command failures.
*   **Dependency Management**: Cleaned up duplicated entries in `go.sum` and fixed `go.mod` application errors.
*   **Duplicate Global Declaration**: Removed redundant `FaceplateRegistry` declaration from `main.go` to resolve "non-declaration statement outside function body" error.
*   **SyncMove Syntax**: Removed stray closing brace causing premature function termination and build failure.
*   **Replay Callback Mismatch**: Synchronized `applyAuthoritativeFrame` and `ExecuteSyncHandshake` callback signatures to use `AuthoritativeFrame`.
*   **Auction Redundancy**: Removed duplicate state checks and re-fetching of auction objects in `HandlePlaceBid`.
*   **Free Refund Exploit**: Implemented `HighestBidIsApproved` check in `HandlePlaceBid` to prevent virtual refunds for approved (non-escrowed) bids.
*   **Loan Precision**: Implemented pure-integer interest rounding in `HandleTakeLoan` and fixed `faucetBalanceMicro` bypass in `HandleRepayLoan`.
*   **Faucet Sync**: Hardened `faucetBalanceMicro` management in `faucet_service.go` to ensure integer-first decrements during reward dispatches.
*   **Achievement Map Safety**: Hardened `achievement_service.go` handlers to ensure player stats maps are initialized before conditional checks.