# NFT Seduction Tasks

## Critical Launch Readiness
- [x] Hardened Card Tooltip: `showPowerTooltip` now accurately reflects all card and player-level modifiers.
- [x] Hardened Card Visuals: `renderCardHTML` now displays Mood, Artifacts, Fatigue, and Loyalty.
- [x] Resolved profile synergy: Achievement, Mojo, and Jailed Card state now flows through Go WASM to UI.
- [x] Audit `updateDynamicArenaFloor` function for correct background texture selection.
- [x] Audit `processInsuranceRecovery` to ensure recovered cards are returned to victim's inventory.
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

## Next steps prompts:
Check the `PerformAIMove` logic in `main.go` to ensure the bot accounts for card Fatigue and Loyalty modifiers when evaluating the optimal move.