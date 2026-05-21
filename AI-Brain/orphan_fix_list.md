# Virtualbabes Arena: Orphan Fix & Logic Preservation Ledger

## 1. Protected Placeholders (Do Not Remove)
These files or code blocks may appear orphaned but are strategic hooks for future expansion.
*   **`bridge_service.go`**: Currently empty. Reserved for Phase 4: Direct cross-chain asset transfers. Logic for onboarding was migrated to `onboarding_service.go` to keep the bridge scope pure.
*   **`isPerishable` flag (`club_service.go`)**: Placeholder within the shop revenue distribution logic. Will eventually handle different tax implications for consumable items (food/meds).
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
*   **Mojo Lifecycle Restoration**:
    *   Implemented `calculateMojoGain` in `club_service.go` to reward revenue, defense, and jailing.
    *   Hardened `processMojoDecay` in `lobby_manager.go` to include the **District Stabilizer** mitigation field.
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
*   **CORS Hardening**: `CheckOrigin` in `server.go` is now strictly linked to the `ALLOWED_ORIGINS` environment variable.