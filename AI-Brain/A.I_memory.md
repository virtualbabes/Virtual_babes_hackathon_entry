# A.I. Memory: Virtualbabes Arena

## Project Context
- **Core Objective**: Evolving a tactical card battler into a Social Economic Simulation.
- **Tech Stack**: Go, WASM, WebSockets, Algorand/Voi Blockchain.
- **Current Phase**: Production-Ready Beta / Hardened Launch Readiness.

## Critical Logic Patterns
- **Switchboard Pattern**: Server-side signing for faucet rewards; client-side proof of intent.
- **Sybil Protection**: Onboarding is gated by historical paged indexer checks; addresses are normalized to lowercase.
- **WASM Determinism**: Core rules (Same/Plus/Combo) must remain identical between client and server.
- **Industrial Leases**: Club members can lease cards; revenue split: 50% Lender, 20% Faucet, 20% Treasury, 10% Members.
- **Domain Separation**: Logic is now split across specialized services (Battle, Economy, Club, Employment, Oracle) to reduce `Lobby` mutex contention.

## Active Priorities
1. **Mainnet Secrets**: Secure wiring of `FAUCET_MNEMONIC` and `ADMIN_WALLETS` for production launch.

## Completed Tasks
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
- **Auction Model**: Uses Server-Authoritative Internal Escrow. $VBV payments are live value; items are internal state.
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
-- `payoutAddress`: (Resolved) Unified verification implemented in `faucet_service.go` reward loop.

## Analysis: Orphaned UI CSS Logic and Improvement Suggestions

Based on a review of the specified SCSS files, HTML, and JS, here's an assessment of orphaned CSS (unused styles) and recommendations for enhancing aesthetics and functional immersion in Virtualbabes Arena.

### Orphaned CSS Logic
These SCSS files define comprehensive styles for UI components that are **not present** in index.html, `app.js`, or collective-intelligence.js. They appear to be prepared for future features but are currently unused, making them "orphaned."

1. **_shops.scss** (Fully Orphaned)
   - Defines styles for `.shops-panel`, `.shop-categories`, `.shop-items`, `.shop-cart`, `.shop-offers`, etc.
   - No corresponding HTML elements or JS functions (e.g., no `shops-panel` in DOM, no shop-related overlays in `app.js`).
   - **Impact**: ~700 lines of unused CSS for shop browsing, purchasing, and special offers.

2. **_criminality.scss** (Fully Orphaned)
   - Defines styles for `.criminality-panel`, `.criminality-actions`, criminal action grids, etc.
   - No usage in HTML/JS (no criminality overlays or panels).
   - **Impact**: ~400 lines of unused CSS for heist/kidnap UI.

3. **_economy.scss** (Partially Orphaned)
   - Defines `.economy-panel`, `.market-ticker` (used for live ticker), `.auction-house`, etc.
   - `.market-ticker` is used (created dynamically in `app.js`), but `.economy-panel` and auction styles are unused.
   - **Impact**: ~300 lines partially unused; market ticker is active.

4. **_social.scss** (Partially Orphaned)
   - Defines `.social-panel`, `.achievement-system` (trophy badges used), `.leaderboard-enhanced`, etc.
   - Trophy system is used (`.trophy-badge` in `openTrophyView`), but `.social-panel` and extended social features are unused.
   - **Impact**: ~300 lines partially unused; achievements are active.

5. **_territory.scss** (Fully Orphaned)
   - Defines `.territory-panel`, `.territory-map`, 3D world map styles, etc.
   - No territory UI in HTML/JS.
   - **Impact**: ~400 lines of unused CSS for regional/club territory visualization.

6. **Other Files** (Not Orphaned)
   - _dashboard.scss, _main-layout.scss, _neon-glass.scss, _animations.scss, _spacing.scss: All actively used in index.html and `app.js`.
   - styles.css: Compiled output, reflects used styles.
   - collective-intelligence.js: Used for NPC taunts in `app.js`.

**Total Orphaned CSS**: ~2,100+ lines across feature files, representing planned but unimplemented UI for shops, criminality, economy overlays, social hubs, and territories.
