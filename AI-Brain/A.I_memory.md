- **Domain Separation**: Logic is now split across specialized services (Battle, Economy, Club, Employment, Oracle) to reduce `Lobby` mutex contention.

# NFT Seduction Tasks

## Project Context

- **Core Objective**: Evolving a tactical card battler into a Social Economic Simulation.
- **Tech Stack**: Go, WASM, WebSockets, Algorand/Voi Blockchain.
- **Current Phase**: Production-Ready Beta / Hardened Launch Readiness.
## Phase 1: Stability & Security Audit
1. **Complete:** Validated `verifyBuyInTransaction` logic against actual Indexer responses.
2. **Complete:** Implemented `sync.RWMutex` in `main.go` to harden the WASM Engine against race conditions during async card imports (Plan F).
3. **Complete:** Performed 16-player tournament stress tests to verify bracket archival and treasury kickback logic under load.
4. **Identity:** Ensure `LinkedWallet` verification handles all target chains (ETH/SOL/POL) consistently.

## Phase 2: Performance Optimization
- Refactored `handleVoiOnboarding` logic from `bridge_service.go` to `onboarding_service.go`.
1. Complete: Implemented partial state synchronization in `GetGameState` (Go & JS) to reduce serialization overhead.
2. Complete: Optimized Market Ticker canvas rendering with text measurement caching and viewport culling.
3. Complete: Hardened `syncUI` logic to support partial snapshots and prevent DOM flickering during high-frequency lobby updates.
4. Complete: Implemented granular node-diffing for `board-container` in `syncUI` to prevent flickering during card flips.
5. Complete: Displayed Cunning and Nurturing values in the `syncUI` profile overlay with Cyberpunk styling.
6. Complete: Implemented Club Lease Board overlay with regional prioritization logic.# A.I. Memory: Virtualbabes Arena

## Critical Logic Patterns
- **Switchboard Pattern**: Server-side signing for faucet rewards; client-side proof of intent.
- **Sybil Protection**: Onboarding is gated by historical paged indexer checks; addresses are normalized to lowercase.
- **WASM Determinism**: Core rules (Same/Plus/Combo) must remain identical between client and server.
- **Industrial Leases**: Club members can lease cards; revenue split: 50% Lender, 20% Faucet, 20% Treasury, 10% Members.
- **Domain Separation**: Logic is now split across specialized services (Battle, Economy, Club, Employment, Oracle) to reduce `Lobby` mutex contention.

## Active Priorities
1. **Production Finalization**: Finalizing WalletConnect Project ID and Node/Indexer redundancy.

## Completed Tasks
- **Hardening**: Applied `sync.RWMutex` to `main.go` (Plan F) and refactored async fetch logic.
- **Security**: Hardened `faucet_service.go:dispatchReward` to gracefully handle mnemonic-to-private-key failures.
- **Bug Fix**: Resolved recursive deadlocks in `courthouse_service.go` by switching to `logAdminAuditLocked`.
- **UI Hardening**: Scrubbed diff artifacts and corrupted template strings from `_overlays.scss`. Fixed malformed media queries.
- **Bug Fix**: Resolved struct boundary error in `common_types.go` (missing closing brace on `Club`).
- **Economy**: Hardened `economy_processing.go` to prevent fractional VBV dust in Market Token liquidation.
- **Economy**: Performed deep analysis of "Industrial Loop" and created `economy_loop_maps.md` for balancing.
- **Hardening**: Implemented Faucet Native Gas Guard in `oracle_service.go` to monitor reward dispatch liquidity.
- **Audit**: Completed comprehensive Game, UI, and File flow audit; identified struct drift between WASM and Server.
- **Concurrency Audit**: Verified `tournament_manager.go` is free of race conditions for `l.tournament.Pot` and concurrent registrations.
- **Bug Fix**: Added explicit error and status code handling in `oracle_service.go:loadOnboardedWalletsFromIndexer`.
- **Bug Fix**: Added explicit logging for non-200 indexer responses in `oracle_service.go:getVerifiedCards`.
- **Bug Fix**: Resolved recursive deadlocks in `employment_service.go` by switching to `Locked` variant helpers.
- **Bug Fix**: Resolved recursive deadlocks in `loan_service.go` by switching to `logAdminAuditLocked`.
- **Bug Fix**: Resolved recursive deadlocks in `market_service.go:handleTradeShares` by using `Locked` variant helpers.
- **Bug Fix**: Resolved critical deadlock in `faucet_service.go` and fixed `skippedAssets` variable scope error.
- **Bug Fix**: Added missing `strings` import to `onboarding_service.go` required for wallet normalization.
- **Validation**: Performed simulated 8/16-player tournament stress tests; verified bracket archival and kickback logic.
- **Rewards**: Hardened Top 5 placement identification and implemented atomic multi-asset distribution.
- **Maintenance**: Verified `cleanupNonces` correctly prunes history without affecting active spectators.
- **Audit**: Verified Multi-Chain `power_divisor` and `power_base` configurations in `availableNetworks`.
- **Visuals**: Implemented canvas-based particle effects for card captures (Phase 2).
- **SCSS Refactor**: Integrated `.particle-canvas` styles into the modular utility system.
- **Criminality**: Hardened heist logic to utilize `GetEffectiveCunning` (including faceplate bonuses) for success probability and kidnap eligibility.
- **Frontend Display**: Implemented display of Cunning and Nurturing values in `syncUI` with Cyberpunk styling.
- **Social UI**: Implemented the Social Hub (Alliances, Career, Achievements) using orphaned `_social.scss` styles.
- **Criminality UI**: Wired the Kidnap Selection interface following successful heists using orphaned `_criminality.scss` styles.
- **Shop UI Refactor**: Fully wired the District Shops overlay using orphaned `_shops.scss` styles, including category filtering.
- **Heist Planning Terminal**: Implemented full tactical interface for heists using `_criminality.scss` grid and risk meter styles.
- **Territory Map UI**: Implemented the 3D territory map visualization in `app.js` using the orphaned `_territory.scss` styles.
- **Heist Planning UI**: Implemented the Heist Planning interface in `app.js` using orphaned `_criminality.scss` styles.
- **Rewards**: Hardened `payoutAddress` validation in `faucet_service.go` to handle granular asset opt-ins.
- **Economy Audit**: Ensured loan interest and auction commissions are added to `l.faucetBalance` before dynamic scaling.
- **Stability**: Resolved critical deadlocks between `economy_processing.go` and `lobby_manager.go`.
- **Heist Logic Alignment**: Modified `handleHeist` in `club_service.go` to precisely match frontend heuristic by temporarily removing trap modifier calculation.
- **Consignment flow**: Implemented the auction creation and Art Gallery interface in `app.js`.
- **Economy UI**: Refactored Entity Market and Black Market UI; implemented Art Gallery (Auctions) using orphaned `_economy.scss` styles.
- **Social UI**: Implemented the Social Hub and Alliance Management UI using orphaned `_social.scss` styles.
- **UI Immersion**: Implemented 3D Territory Map zoom controls and integrated remaining topographic styles from `_territory.scss`.
- **Liquidity**: Implemented "Industrial Loop" recovery where black market scavenge fees return to the Faucet pool.
- **UI Textures**: Implemented dynamic arena floor textures based on game phase and tournament status.
- **Faceplates**: Cosmetic items now provide functional Mojo/Cunning bonuses via `FaceplateRegistry`. Mojo bonuses boost Reputation.
- **Criminality UI**: Wired the Kidnap Selection interface following successful heists using orphaned `_criminality.scss` styles.
- **Heist Audit**: Aligned Heist success probabilities by broadcasting effective stats in lobby updates and fixing scope bugs in `club_service.go`.
- **Documentation**: Merged `ReadMe.txt` into `README.md`.
- **Hardening**: Applied `sync.RWMutex` to `main.go` (Plan F) and refactored async fetch logic.
- **Validation**: Performed simulated 16-player tournament stress tests; verified bracket archival and kickback logic.
- **Rewards**: Hardened Top 5 placement identification and implemented atomic multi-asset distribution.
- **Maintenance**: Verified `cleanupNonces` correctly prunes history without affecting active spectators.
- **Audit**: Verified EVM `power_divisor` and `power_base` configurations.
- **Visuals**: Implemented canvas-based particle effects for card captures (Phase 2).
- **SCSS Refactor**: Integrated `.particle-canvas` styles into the modular utility system.
- **Moderation**: Hardened `handleAvatarBan` with URL normalization and enforced check in `register_avatar`; removed duplicate tournament simulation handlers.
- **Cosmetics**: Implemented 'equip_cosmetic' WebSocket protocol for switching active faceplates with inventory validation.
- **Admin Security**: Strictly enforced signature-based auth in `handlers_admin.go`; removed legacy keys; added EVM admin support.
- **Criminality**: Hardened heist logic to utilize `GetEffectiveCunning` (including faceplate bonuses) for success probability and kidnap eligibility.
- **Frontend Display**: Implemented display of Cunning and Nurturing values in `syncUI` with Cyberpunk styling.
- **Rewards**: Hardened `payoutAddress` validation in `faucet_service.go` to handle granular asset opt-ins.
- **Economy Audit**: Ensured loan interest and auction commissions are added to `l.faucetBalance` before dynamic scaling.
- **Stability**: Resolved critical deadlocks between `economy_processing.go` and `lobby_manager.go`.
- **Liquidity**: Implemented "Industrial Loop" recovery where black market scavenge fees return to the Faucet pool.
- **Attributes**: Wired Cunning (Stealth) and Nurturing (Fatigue Care) into the deterministic power calculation.
- **Hardening**: Aligned server-side power validation with WASM Cunning/Nurturing logic and secured `handleMove` against power spoofing.
- **Economy**: Implemented stack-wide Dynamic Scaling in `economy_service.go` with unscaled target tracking.
- **Rewards Audit**: Verified Reputation Bonus multiplier is consistently applied across all reward assets in `faucet_service.go`.
- **Unit Logic**: Verified that `Treasury` and `FaucetBalance` use base $VBV units, while rewards use micro-units (1M conversion).
- **Lease UI**: Implemented `openClubLeaseBoard` in `app.js` with region-aware sorting based on player employment.
- **Economic Precision**: Hardened `handleTakeLease` to recover rounding remainders into Club Treasuries.
- **Market Hardening**: Refactored Portfolios to use persistent wallet keys; linked trades to `faucetBalance` to prevent inflation and ensure cross-session holdings.
- **Continuity**: Hardened `InitialRewards` persistence via `season.json` to ensure economic state survives restarts.
- **Sybil UI Feedback**: Frontend `app.js` now correctly informs users if Sybil protection is still warming up.
- **Frontend Optimization**: `syncUI` in `app.js` now uses string comparison flicker guards and filter-aware partial updates.
- **Identity Cache**: Implemented backend-side `envoiCache` and resolved potential recursive RLock deadlocks.
- **Auction Model**: Uses Server-Authoritative Internal Escrow.  payments are live value; items are internal state.
- **Auction Card Escrow**: Fixed `CardID` missing from escrow logic in `auction_service.go` and `economy_processing.go`.
- **Identity Cache**: Implemented backend-side `envoiCache` and dedicated resolution logic for economic results.
- **Auction Bid Logic**: Hardened `handlePlaceBid` to deduct new bids and immediately refund previous highest bidders.
- **Loan UI Consistency**: Implemented non-blocking Envoi name resolution for borrowers in `loan_service.go`.
- **Mojo Decay**: Hardened `processMojoDecay` with UI broadcast triggers and ensured courthouse revenue refreshes `LastActivity`.
- **Profile Synergy**: Achievement and Player state now correctly sync from `lobby_update` into WASM `Engine` via `SyncFullProfile`.
- **Industrial Hardening**: Implemented inventory capacity guards in `handleRestockInventory` to prevent state bloat.
- **Periodic Tickers**: Resolved deadlocks in `processAuctions`, `checkVaultBalanceOnChain`, and `handleUnregister` for improved stability.
- **Reward Consistency**: Verified `dispatchReward` correctly applies reputation bonus multiplier to all reward assets.
- **Black Market Hardening**: Resolved deadlock in `handleBuyBlackMarket`, ensured reputation recalculation on Wanted Level increase, and added UI sync triggers.
- **Tournament Mechanics**: Verified `handleTournamentRegister` correctly bypasses buy-in verification for elite players.
- **Protocol Hardening**: Refactored `handleGameProtocol` to delegate to service files, fixed deadlocks, and added unhandled message logging.
- **Rumor Management**: Verified `processRumors` correctly deletes expired entries while holding the mutex.
- **Tournament History**: Implemented conditional deep verification in `handleTournamentHistory` based on `deep_verify` parameter.
- **Spectator Stability**: Hardened `initiatePairedMatch` in `lobby_manager.go` to snapshot Avatars, Gloats, and authoritative Board Moods into `MatchState`.
- **Kidnap Gambit**: Hardened `handleKidnapRequest` with robust card selection (favorite vs. rarest) and explicit removal from victim's inventory.
- **Spectator Accuracy**: Hardened `SetBoardState` in `main.go` to synchronize authoritative board moods, territory, and penalty snapshots for accurate spectator tooltips.
- **Kidnap Economy**: Audited `handlePayRansom` to implement a 20% 'Laundering Tax' returning to the faucet, completing the Industrial Loop for kidnappings.
- **Tooltip Accuracy**: Hardened `showPowerTooltip` in `app.js` to correctly calculate and display effective power, including player penalties and item buffs, by synchronizing `ActiveItemBuffs` from WASM.
- **Heist Economy**: Audited `handleHeist` to implement a 10% "Fence Fee" on successful loot, returning to the faucet and triggering dynamic scaling.
- **Match Completion Hardening**: Audited `processFallenPenaltyJail` for AI/BYE guards and refactored match finalization to a Locked pattern to resolve recursive deadlocks.
- **Jailing Mechanics**: Secured `processFallenPenaltyJail` and `processPrisonerRule` to use decrementing inventory logic and verified card existence before jailing. Utilized `CaptureType` for tactical feedback.
- **Tournament Kickback Accuracy**: Ensured `distributeTournamentKickback` uses the precise blockchain transaction time for club membership verification.
- **Mojo Decay**: Hardened `processMojoDecay` with periodic resets and added `LastActivity` triggers to management actions.
- **Club Restock**: Audited `handleRestockInventory` for correct authorization and improved error feedback.
- **Heist Mechanics**: Audited `handleHeist` security multipliers; implemented lazy pruning for expired traps and verified activity tracking/UI synchronization.
- **AI Evaluation**: Hardened `PerformAIMove` simulation logic to use `getEffectivePower` and authoritative rule keys (`Power_copy`/`Power_up`).
- **Card Tooltip Accuracy**: `showPowerTooltip` now accurately reflects all card and player-level modifiers, including Wanted Level, Cunning, and Nurturing.
- **Card Visuals**: Enhanced `renderCardHTML` to display Mood icons, Artifact bonuses, Fatigue levels, and Loyalty status on the card face.
- **Insurance Recovery**: Fixed `processInsuranceRecovery` to correctly return recovered cards to the victim's inventory.
- **UI Textures**: Implemented dynamic arena floor textures based on game phase and tournament status.
- **Faceplates**: Cosmestic items now provide functional Mojo/Cunning bonuses via `FaceplateRegistry`. Mojo bonuses boost Reputation.

## Technical Notes
- **Economy**: `economy_processing.go` handles temporal cleanup (loans/auctions) independently of main handlers.
- **Coupling**: `app.js` is the primary consumer of WASM state; state changes in `main.go` require `app.js:syncUI` updates.
- **Narrative**: NPC Taunts are client-side based on server-evaluated tendencies (`collective-intelligence.js`).
- **Visuals**: Modular SCSS system with 3D CSS perspective for the world map.

## Orphans & Fixed Knowledge
- `bridge_service.go`: Placeholder for future expansion; current onboarding is in `onboarding_service.go`.
-- `payoutAddress`: (Resolved) Unified verification implemented in `faucet_service.go` reward loop.import { collectiveIntelligence } from './collective-intelligence.js';

## Critical Launch Readiness
- [x] Audit `renderCardHTML` and `showPowerTooltip` for full penalty and buff synchronization.
- [x] Hardened spectator synchronization: `handleSpectate` populates penalty snapshots; hardened move routing and session cleanup.
- [x] Hardened spectator synchronization: `SetBoardState` now ingests moods, territory, and full ruleset.
- [x] Audit `handleHeist` logic for "Fence Fee" and Industrial Loop integration.
- [x] Audit `handlePayRansom` and integrated kidnapping into the Industrial Loop.
- [x] Audit `processFallenPenaltyJail` for AI/BYE handling and resolved match completion deadlocks.
- [x] Hardened Jailing Mechanics: Implemented capture-type tracking and inventory safety guards in `battle_service.go`.
- [x] Review and harden `initiateSuddenDeath` to prevent card loss/theft during tie-breakers.
- [x] Audit `handleRestockInventory` for micro-unit precision in treasury deductions.
- [x] Hardened Move Validation: `CapturedCards` now distinguishes capture types (BASIC, SAME, etc.) for jailing accuracy.
- [x] Resolved profile synergy: Achievement, Mojo, and Jailed Card state now flows through Go WASM to UI.
- [x] Audit `processAuctions` logic for Art Gallery commission distribution.
- [x] Hardened Industrial Leases: Verified ownership transfer, fixed deadlocks, and standardized payouts.
- [x] Audit `distributeCourthouseFineToClubs` for correct split and activity updates; fixed deadlock.
- [x] Audit `CalculateReputation` for correct Employment Multiplier and Cosmetic Prestige bonuses.
- [x] Hardened Club Management: UI sync added to Create/Join/Purchase handlers.
- [x] Audit `handleHeist` for activity tracking and social standing updates.
- [x] Audit `archiveSeason` logic for correct seasonal stats export and leaderboard wipe.
- [x] Audit `processInsuranceRecovery` and harden Kidnap Gambit recovery paths.
- [x] Hardened `dispatchTournamentRewards` with granular opt-in checks and skipped asset tracking.
- [x] Audit `processLeaseExpirations` for correct card return and UI sync.
- [x] Audit `handleMove` in `lobby_manager.go` for Wanted/Fatigue penalty persistence.
- [x] Audit `handleMove` in `lobby_manager.go` for Fallen Penalty state snapshots.
- [x] Audit `handleGameProtocol` for robust handling of unregistered wallets.
- [x] Audit `checkAssetOptIn` for robust indexer error handling.
- [x] Audit `handleSpreadRumor` and `handleTradeShares` for rumor application accuracy.
- [x] Audit `handleRepayLoan` for interest verification and scaling concurrency.
- [x] Hardened `processAuctions` to ensure inventory safety and immediate UI synchronization.
- [x] Audit `processLoans` to ensure 15% residual value and UI sync are correct.
- [x] Audit `handleTakeLoan` to ensure collateral escrow and faucet deductions are accurate.
- [x] Audit `handleTournamentRegister` to ensure `distributeTournamentKickback` uses accurate `registrationTime`.
- [x] Review the `handleKidnapRequest` function in `handlers_criminality.go` to ensure that the selection logic for the target card (favorite vs. rarest) is robust and handles edge cases where no cards are found.
- [x] Audit `handleTournamentHistory` for `deep_verify` parameter to trigger full re-verification.
- [x] Audit `processRumors` for correct deletion of expired rumors under mutex.
- [x] Audit `handleTournamentRegister` to ensure `isElite` bypasses buy-in verification.
- [x] Audit `dispatchReward` to ensure reputation bonus applies to all assets.
- [x] Audit `run` function for mutexes and race conditions.
- [x] Fix recursive RLock deadlock risk in `ResolveEnvoiName`.
- [x] Audit `handleGameProtocol` for unhandled message types and race conditions.
- [x] Audit `handlePlaceBid` to ensure outbid funds are immediately returned.
- [x] Optimize `handleGetAuctions` to resolve names outside the global state lock.
- [x] Hardened Portfolio UI: Implemented wallet-based key lookups and Envoi name resolution.
- [x] Audit and harden `handleTradeShares` unit conversion and economic flow.
- [x] Ensure `InitialRewards` targets are persisted in `season.json`.
- [x] Implement Industrial Leasing Layer in Clubs with revenue sharing.
- [x] Audit `club_service.go` for base-unit vs micro-unit consistency.
- [x] Hardened `handleRestockInventory` with club inventory capacity limits.
- [x] Hardened `handleBuyBlackMarket` with reputation recalculation and UI sync.
- [x] Implement stack-wide Dynamic Scaling based on faucet liquidity.
- [x] Hardened administrative authentication (Strict signatures, multi-chain support).
- [ ] Securely wire `FAUCET_MNEMONIC` and `ADMIN_WALLETS` for Mainnet.
- [x] Verify Mainnet Node/Indexer stability in `networks.json`.
- [x] Hardened `checkVaultBalanceOnChain` and `checkNativeVaultBalanceOnChain` in `lobby_manager.go` ticker loop.
- [x] Perform 16-player tournament stress tests.
- [x] Implement canvas-based 'sparks' particle effects and integrate into SCSS.
- [x] Fix `CardID` escrow logic in `auction_service.go` and `economy_processing.go`.
- [x] RPG Integration: Implemented Mojo/Cunning bonuses for Faceplates.
- [x] Audit `processMojoDecay` and update `club.LastActivity` in economic handlers.
- [x] Implement 'equip_cosmetic' message type in `lobby_manager.go`.
- [x] Audit `handleHeist` in `club_service.go` for `GetEffectiveCunning` integration.

## Technical Debt & Refactoring
- [x] Decompose `lobby_manager.go` into domain-specific service files.
- [x] Audit `club_service.go` for concurrency issues (Treasury/Inventory).
- [x] Implement partial state sync in WASM `GetGameState` (Go & JS).
- [x] Refactor `handleUseItem` logic to `item_service.go`.
- [x] Audit `battle_service.go` for potential race conditions during high-concurrency matches.
- [x] Validate and harden `verifyBuyInTransaction` logic for Voi/Algorand indexers.
- [x] Refactor `processLoans` and `processAuctions` logic to `economy_processing.go`.
- [x] Refactor Auction API handlers to `auction_service.go`.
- [x] Refactor Black Market API handlers to `black_market_service.go`.
- [x] Refactor Voi Onboarding logic to `onboarding_service.go`.
- [x] Refactor Loan API handlers to `loan_service.go`.
- [x] Audit and harden `faucet_service.go` for granular asset opt-ins and consistent reputation bonus application.
- [x] Audit `onboarding_service.go` for potential edge cases in Sybil protection.
- [x] Implement historical onboarding recovery in `oracle_service.go`.
- [x] Refactor Career API handlers to `employment_service.go`.
- [x] Audit `employment_service.go` for concurrency and case-consistency.
- [x] Audit and harden `handleAvatarBan` in `handlers_admin.go`.
- [x] Audit `handleUpdateBaseReward` and `handleUpdateRewardAsset` routes in `server.go`.
- [x] Implement non-blocking Envoi name resolution for `handleGetLoans`.
- [x] Ensure Sybil sync status check correctly informs the UI.
- [x] Pre-resolve Envoi names in `auction_service.go` for Art Gallery performance.
- [x] Ensure Sybil sync status check correctly informs the UI.
- [x] Update `package.json` with Go and WASM build scripts.
- [x] Audit `app.js` for memory leaks and handler stability.
- [x] Optimize Market Ticker canvas rendering in `app.js`.
- [x] Hardened `syncUI` in `app.js` with partial state sync and flicker guards for rewards.
- [x] Hardened `syncUI` board rendering with granular node-diffing to prevent flickering.
- [x] Analyze frontend logic and build configuration (app.js, collective-intelligence.js, package.json).
- [x] Comprehensive SCSS/CSS UI/UX audit and HTML structural analysis.
- [x] Merge ReadMe.txt into README.md.# Development Plan

## Current Progress
- Refactored `lobby_manager.go` into `club_service.go` and `market_service.go`.
- Audited `club_service.go`: Verified mutex coverage for Treasury and Inventory.
- Audited `battle_service.go`: Verified mutex coverage for high-frequency move processing.
- Implemented `checkVaultBalanceOnChain` placeholder in `lobby_manager.go`.
- Hardened `verifyBuyInTransaction` to support standard Algorand Indexers and Voi custom endpoints.
- Refactored `handleUseItem` logic from `lobby_manager.go` to `item_service.go`.
- Refactored `processLoans` and `processAuctions` logic from `economy_service.go` to `economy_processing.go`.
- Refactored Auction API handlers from `economy_service.go` to `auction_service.go`.
- Refactored Black Market handlers from `economy_service.go` to `black_market_service.go`.
- Refactored Loan API handlers from `economy_service.go` to `loan_service.go`.
- Refactored Faucet API handlers from `economy_service.go` to `faucet_service.go`.
- Audited `onboarding_service.go` for Sybil protection; identified "drain and re-claim" vulnerability.
- Implemented `loadOnboardedWalletsFromIndexer` to restore Sybil protection state on startup.
- Refactored `handleHirePlayer` and `handleSetSalary` from `career.go` to `employment_service.go`.
- Audited `employment_service.go`: Verified mutex coverage and hardened wallet string normalization.
- Implemented real-time $VBV pool monitoring via `checkVaultBalanceOnChain`.
- Updated `package.json` to include WASM and Server build pipelines.
- Audited `app.js`: Resolved WebSocket watchdog race conditions and fixed ReferenceErrors in API handlers.
- [x] Completed frontend analysis: Verified synergy between `collective-intelligence.js` and `app.js`.
- [x] Hardened Entity Market: Linked share trading to faucet liquidity and implemented persistent wallet-based portfolios.
- [x] Hardened Art Gallery: Optimized name resolution to be non-blocking and fixed recursive deadlock vulnerabilities.
- [x] Hardened Cosmetics Layer: Implemented RPG bonuses for Faceplates and integrated them into social standing (Reputation).
- [x] Hardened Periodic Tickers: Resolved deadlocks in `processAuctions`, `checkVaultBalanceOnChain`, and `handleUnregister`.
- [x] Hardened Game Protocol: Delegated message handling to service files and resolved race conditions.
- [x] Hardened Reward Dispatch: Verified reputation bonus is applied consistently across all reward assets.
- [x] Hardened Restock Logic: Added capacity limits to club inventory management.
- [x] Hardened Rumor System: Verified `processRumors` correctly handles deletion of expired entries.
- [x] Hardened Tournament History: Implemented conditional deep verification based on `deep_verify` parameter.
- [x] Hardened Tournament Registration: Verified `isElite` check correctly bypasses buy-in transaction verification.
- [x] Hardened Kidnap Gambit: Ensured robust card selection logic (favorite vs. rarest) and correct removal from victim's inventory in `handleKidnapRequest`.
- [x] Hardened Spectator Logic: Implemented `handleSpectate`, secured move routing for viewers, and fixed session cleanup bugs.
- [x] Hardened Spectator Sync: Ensured `SetBoardState` in WASM correctly ingests authoritative moods and participant penalty snapshots.
- [x] Hardened Tooltip Accuracy: Ensured `showPowerTooltip` in `app.js` correctly reflects all dynamic power modifiers, including item buffs.
- [x] Hardened Kidnap Economy: Integrated `handlePayRansom` into the Industrial Loop with 20% faucet recovery and scaling.
- [x] Hardened Jailing Mechanics: Implemented capture-type tracking for deterministic flip attribution.
- [x] Hardened Jailing Mechanics: Implemented capture-type tracking for deterministic flip attribution.
- [x] Hardened Loan System: Verified collateral escrow and implemented principal deduction from faucet liquidity.
- [x] Hardened Auction Commission: Ensured 10% commission from Art Gallery auctions is distributed to the owning club's treasury.
- [x] Hardened Courthouse Fines: Resolved deadlock and verified equitable fine distribution logic.
- [x] Hardened Industrial Leases: Resolved TakeLease deadlock and verified economic precision.
- [x] Hardened Reputation System: Confirmed correct weighting of Employment Multiplier and Cosmetic Prestige bonuses.
- [x] Hardened Lease Expirations: Ensured returned cards are restored and UI is synchronized.
- [x] Hardened Move Validation: Secured `handleMove` power logic against session-drop penalty exploits.
- [x] Hardened Opt-In Checks: Differentiated indexer errors from missing opt-ins in `checkAssetOptIn`.
- [x] Hardened Tournament Kickbacks: Ensured `distributeTournamentKickback` uses accurate blockchain transaction time for club membership.
- [x] Hardened Black Market: Resolved deadlocks and implemented immediate UI synchronization for criminal activity.
- [x] Hardened Auction Bidding: Implemented immediate deduction of new bids and refund of outbid funds.
- [x] Hardened Sybil Sync UI: Frontend now correctly displays "warming up" message for onboarding.
- [x] Hardened Market Trading: Ensured precise micro-unit conversion and `float64` division in `handleTradeShares`.
- [x] Hardened Auction Escrow: Fixed `CardID` missing from internal escrow in `auction_service.go` and `economy_processing.go`.
- [x] Audited Auction Architecture: Verified Internal Escrow model and identified CardID escrow gap.
- [x] Hardened Entity Market: Linked share trading to faucet liquidity and ensured portfolio persistence.
- [x] Hardened Art Gallery: Implemented proactive and lazy Envoi name resolution for auction participants.
- [x] Hardened Lease Economy: Implemented remainder recovery logic to prevent micro-unit loss during member payouts.
- [x] Hardened Sybil Sync UI: Frontend now correctly displays "warming up" message for onboarding.
- [x] Implemented Industrial Leasing: Added card rental market within Clubs with multi-entity payout logic.
- [x] Hardened Industrial Loop: Verified base-unit consistency in `handleRestockInventory` and refined error feedback.
- [x] Hardened Tournament Finalization: Implemented Top 5 identification and multi-asset atomic reward dispatch.
- [x] Hardened Admin Security: Strictly enforced WalletConnect/ARC-14 signatures for all administrative actions.
- [x] Hardened Move Logic: Server-side power validation now accounts for Cunning and Nurturing mitigation factors.
- [x] Hardened Economy Processing: Ensured loan interest and auction commissions correctly increase `l.faucetBalance` and trigger dynamic scaling.
- [x] Hardened Reward Flow: Implemented granular opt-in checks for all assets in the reward stack within `faucet_service.go`.
- [x] Comprehensive UI/UX Audit: Validated "Neon-Glass" SCSS modularity and 3D territory map performance.
- [x] SCSS Integration: Migrated particle effect styles to `_animations.scss`.
- [x] Audited Ephemeral Cleanup: Verified `cleanupNonces` safety for spectating sessions.
