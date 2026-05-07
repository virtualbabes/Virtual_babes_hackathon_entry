# NFT Seduction Tasks

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
- [x] Implement `checkVaultBalanceOnChain` and `checkNativeVaultBalanceOnChain` in `lobby_manager.go` ticker loop.
- [ ] Perform 16-player tournament stress tests.
- [x] Implement canvas-based 'sparks' particle effects and integrate into SCSS.
- [x] Fix `CardID` escrow logic in `auction_service.go` and `economy_processing.go`.
- [x] Resolve Faceplate Residue: Implemented Mojo/Cunning bonuses for Faceplates.
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
- [x] Merge ReadMe.txt into README.md.
- [x] Wire the Kidnap Selection UI in `app.js` using the orphaned criminality SCSS styles.
- [x] Implement the 3D territory map visualization in app.js using the remaining orphaned _territory.scss styles.


### Recommendations for Aesthetics and Functional Immersion Improvements

#### 1. **Implement Orphaned Features for Immersion**
   - **Priority**: High. The orphaned CSS indicates planned features (shops, criminality, territories) that would significantly enhance gameplay immersion.
   - **Suggestions**:
     - [x] Add shop overlays (using _shops.scss) for item purchasing, with animated item reveals and purchase confirmations.
     - [x] Implement criminality UI (using _criminality.scss) for heist planning, with risk meters and success animations.
     - [x] Implement the 3D territory map visualization in app.js using the remaining orphaned _territory.scss styles.
     - [x] Refactor Shop Overlay to use high-fidelity grid and category filtering from `_shops.scss`.
     - [x] Implement the Heist Planning interface in app.js using the orphaned _criminality.scss styles to activate the tactical grid.
     - [x] Expand social panel (using _social.scss) for player interactions, alliances, and extended leaderboards.
     - [x] Use _economy.scss for auction houses and portfolio management overlays.
   - **Immersion Boost**: These features would make the "Social Economic Simulation" feel more interactive, moving beyond basic matchmaking to full ecosystem engagement.

#### 2. **Enhance Existing Neon-Glass Aesthetics**
   - **Current State**: Strong glassmorphism with neon accents, but could be more dynamic.
   - **Suggestions**:
     - Add more gradient animations in _neon-glass.scss (e.g., pulsing borders on active elements).
     - Implement dynamic background shifts based on game phase (e.g., red tint during criminal actions, using CSS variables set by `app.js`).
     - Enhance hover effects: Add micro-animations (scale + glow) to buttons/cards using _animations.scss.
     - Introduce particle effects for more events (e.g., victory sparks, defeat smoke) by expanding the existing particle system in `app.js`.

#### 3. **Improve Functional Immersion**
   - **Animations & Feedback**:
     - Expand _animations.scss with game-specific keyframes (e.g., card flip with neon trails, territory conquest waves).
     - Add loading states with shimmer effects (`.animate-shimmer`) for async operations like NFT resolution.
     - Implement sound cues tied to animations (e.g., hover sounds, capture effects) by integrating with existing audio in `app.js`.
   - **Interactive Elements**:
     - Enhance tooltips (already in `app.js`) with animated reveals and contextual info.
     - Add micro-interactions: Button presses with scale-down, form inputs with focus glows.
     - Use _spacing.scss utilities for responsive layouts on mobile, ensuring immersion on all devices.
   - **Narrative Integration**:
     - Tie collective-intelligence.js taunts to UI animations (e.g., taunt text with typewriter effect).
     - Add mood-based UI changes (e.g., volatile card captures trigger screen shake).

#### 4. **Performance & Maintenance**
   - **Clean Up**: Remove or comment out orphaned CSS to reduce bundle size, or add TODO comments for future implementation.
   - **Modular Enhancements**: Use SCSS variables for consistent theming (e.g., dynamic color shifts for different game modes).
   - **Testing**: After implementing features, ensure CSS is used (e.g., via browser dev tools to check for unused rules).

#### 5. **Specific Code Suggestions**
   - In `app.js`: Add functions like `openShopsOverlay()` using the orphaned styles.
   - In index.html: Add hidden overlay containers for shops/criminality/territories.
   - In _animations.scss: Add `.animate-card-capture { animation: capture-burst 0.5s; }` for better feedback.
   - Overall: Increase use of CSS custom properties for dynamic theming (e.g., `--arena-mood: red` for criminal phases).

Implementing these would transform the UI from functional to deeply immersive, aligning with the "high-stakes Social Economic Simulation" vision. The existing foundation (neon-glass, animations) is solid—expanding it with the orphaned features would complete the aesthetic.

## Next Tactical Steps
1. [x] Implement the Social Panel alliance management UI in app.js using the orphaned _social.scss styles.
2. [x] Audit the handleHeist logic in club_service.go to ensure the success probability matches the frontend heuristic exactly.
3. **Mainnet Secret Audit**: Final verification of mnemonic encryption and environment variable injection for production.
4. **Tournament Stress Test**: Execute a full 16-player mock tournament via `simulateTournament` while monitoring indexer latency.