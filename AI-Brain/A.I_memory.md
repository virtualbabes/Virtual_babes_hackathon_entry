- **Domain Separation**: Logic is now split across specialized services (Battle, Economy, Club, Employment, Oracle) to reduce `Lobby` mutex contention.

# NFT Seduction Tasks

## Project Context

- **Core Objective**: Evolving a tactical card battler into a Social Economic Simulation.
- **Tech Stack**: Go, WASM, WebSockets, Algorand/Voi Blockchain.
- **Current Phase**: Production-Ready Beta / Hardened Launch Readiness.
## Phase 1: Stability & Security Audit
1. **Complete:** Logic audit of `verifyBuyInTransaction` against Indexer schema patterns.
2. **Complete:** Implemented `sync.RWMutex` in `main.go` to harden the WASM Engine against race conditions during async card imports (Plan F).
3. **Logic Verified:** Internal simulation (8/16 player) confirms bracket and kickback logic; **Live Stress Test Pending**.
4. **Identity:** Ensure `LinkedWallet` verification handles all target chains (ETH/SOL/POL) consistently.

## Phase 2: Performance Optimization
- Refactored `handleVoiOnboarding` logic from `bridge_service.go` to `onboarding_service.go`.
1. Complete: Implemented partial state synchronization in `GetGameState` (Go & JS) to reduce serialization overhead.
2. Complete: Optimized Market Ticker canvas rendering with text measurement caching and viewport culling.
3. Complete: Hardened `syncUI` logic to support partial snapshots and prevent DOM flickering during high-frequency lobby updates.
4. Complete: Implemented granular node-diffing for `board-container` in `syncUI` to prevent flickering during card flips.
5. Complete: Optimized `renderCardHTML` in `ui.js` by hoisting static maps and caching global lookups.
5. Complete: Displayed Cunning and Nurturing values in the `syncUI` profile overlay with Cyberpunk styling.
6. Complete: Implemented Club Lease Board overlay with regional prioritization logic.# A.I. Memory: Virtualbabes Arena
7. Complete: Optimized `handleServerMessage` in `network.js` by batching UI sync calls via `requestAnimationFrame` and removing high-frequency console logs.
8. Complete: Hardened `syncUI` in `app.js` with DOM caching and non-blocking asynchronous asset resolution.
9. Complete: Hardened `processLoans` in `economy_processing.go` to synchronize liquidation fees with Faucet dynamic scaling.
10. Complete: Hardened `handleSellMarketTokens` in `black_market_service.go` to deduct payouts from the Faucet balance and trigger dynamic scaling.
83. **Awaiting Verification**: Live environment stress test for 16-player tournaments.

- **Switchboard Pattern**: Server-side signing for faucet rewards; client-side proof of intent.
- **Sybil Protection**: Onboarding is gated by historical paged indexer checks; addresses are normalized to lowercase.
- **WASM Determinism**: Core rules (Same/Plus/Combo) must remain identical between client and server.
- **Industrial Leases**: Club members can lease cards; revenue split: 50% Lender, 20% Faucet, 20% Treasury, 10% Members.
- **Domain Separation**: Logic is now split across specialized services (Battle, Economy, Club, Employment, Oracle) to reduce `Lobby` mutex contention.

# A.I. Memory: Virtualbabes Arena

## Active Priorities
1. **Deployment Hardening**: Finalizing Render + GitHub Asset sync for launch via vbvfaucet.carrd.co.

## Completed Tasks
- **UI Hardening**: Resolved accessibility issues (labels/titles/associations) and removed all illegal inline styles in `index.html`.
- **A11y Audit**: Ensured all form elements in the Arena dashboard and Admin suite are WCAG compliant.
- **SCSS Audit**: Fixed cross-browser compatibility for `text-size-adjust`, `scrollbar-width`, and `user-drag`.
- **A11y Typography**: Hardened neon text contrast, outlines, and line-height accessibility in `_typography.scss` and `_spacing.scss`.
- **SCSS Hardening**: Synchronized standard and Webkit scrollbar behaviors in `_reset.scss` for full Safari/Firefox/Chrome parity.
- **SCSS Hardening**: Resolved "at-rule or selector expected" syntax error in `_overlays.scss`.
- **UI Performance**: Optimized 3D map, dashboard, button, and card rendering by replacing expensive `transition: all` declarations with specific property lists and adding GPU hints.
- **SCSS Hardening**: Audited `_cards.scss` for hardcoded colors and synced card typography with the `$font-heading` token.
- **SCSS Hardening**: Replaced all remaining hex and hardcoded RGB values in `_criminality.scss` with system design tokens.
- **UI Performance**: Optimized Social Hub and Achievement badge animations by replacing `transition: all` and adding GPU hints to `_social.scss`.
- **Mobile Normalization**: Suppressed tap highlights and eliminated 300ms tap delay in `_reset.scss` for better mobile responsiveness.
- **Haptic Feedback**: Hardened `:active` states for all button variants in `_buttons.scss` to provide tactile visual feedback on touch devices.
- **Haptic Feedback**: Implemented tactile `:active` states and selection glow animations for playing cards and miniatures in `_cards.scss`.
- **Haptic Feedback**: Extended tactile `:active` states and optimized transitions for shop items, category tabs, and checkout controls in `_shops.scss`.
- **Haptic Feedback**: Optimized transitions and added tactile `:active` feedback to market ticker, auctions, and black market items in `_economy.scss`.
- **Haptic Feedback**: Extended tactile `:active` states and optimized transitions for map tiles, district cards, and foundry controls in `_territory.scss`.
- **UI Performance**: Eliminated 'transition: all' performance bottlenecks and implemented GPU acceleration for modal overlays in `_overlays.scss`.
- **Design Tokens**: Centralized magic numbers for panel widths, mini-cards, and layout heights into `_variables.scss` for global consistency.
- **SCSS Hardening**: Standardized panel widths and card-mini dimensions across all feature modules using design tokens.
- **SCSS Hardening**: Refactored `_overlays.scss` to eliminate hardcoded margin and padding values, migrating them to the `$spacing` token scale.
- **A11y Audit**: Improved contrast ratios for card type icons in `_cards.scss` by utilizing dark text on high-luminance backgrounds.
- **A11y Audit**: Hardened button accessibility in `_buttons.scss` by implementing dark text on high-intensity neon backgrounds.
- **A11y Audit**: Hardened button keyboard navigation in `_buttons.scss` by refining `focus-visible` styles across all variants.
- **SCSS Hardening**: Standardized typography in `_overlays.scss` and `_shops.scss` by migrating all hardcoded font-sizes to the `$font-size` token scale.
- **SCSS Hardening**: Verified `_territory.scss` token compliance and refactored related hardcoded styles in `app.js` and `index.html`.
- **A11y Audit**: Hardened modal accessibility in `_overlays.scss` by adding focus indicators to close buttons, tabs, and interactive slots.
- **A11y Audit**: Hardened card selection accessibility in `_cards.scss` by adding prominent `focus-visible` styles for the Deck Manager and Board.
- **A11y Audit**: Hardened market ticker legibility in `_economy.scss` with increased opacity and motion-sensitivity overrides.
- **A11y Audit**: Improved 3D map readability in `_territory.scss` by hardening label contrast and tokenizing legend text.
- **A11y Audit**: Hardened social status indicator contrast and improved "Locked" state legibility in `_social.scss`.
- **A11y Audit**: Hardened Heist Risk Meter contrast and motion accessibility in `_criminality.scss`.
- **CSS Refactoring**: Migrated remaining inline styles from the Deck Manager and Admin Suite to modular SCSS (Admin Suite completed).
- **Hardening**: Applied `sync.RWMutex` to `main.go` (Plan F) and refactored async fetch logic.
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
- **Reputation System**: Adjusted `CalculateReputation` to use an additive bonus for `RumorCount` instead of a multiplier, ensuring players with zero wins still gain standing from social influence.
- **Documentation**: Merged `ReadMe.txt` into `README.md`.
- **Hardening**: Completed comprehensive documentation synchronization. Updated `Game_expansion_plan.md` to establish the implemented Production-Ready Beta as the new baseline for future expansion milestones.
- **Audit**: Synchronized `DIR.md` with physical `Public/js` inventory; identified and restored missing `particles.js` entry.
- **Visuals**: Hardened `particles.js` with dynamic, state-aware physics. Heists scale with Cunning, Rewards scale with Mojo, and Foundry effects adapt to Club Industry types for personalized immersion.
- **Visuals**: Integrated dynamic `triggerFoundryFusion` call in `economy.js` to reflect Club Industry type.
- **Visuals**: Implemented ambient tile mood motes in `particles.js` and wired via `ui.js:syncBoardParticles` (with throttled audio cues in `audio.js`) to enhance board immersion.
- **Audio**: Refactored SFX engine in `audio.js` to use `AudioContext` for low-latency polyphony; implemented `PlaySound` override to support overlapping combat effects.
- **Audio**: Integrated `syncSFXGain` into `app.js:setMasterVolume` to ensure low-latency gain adjustment for the Web Audio subsystem.
- **Audio**: Integrated `syncSFXGain` into `app.js:setSfxVolume` to ensure gain parity for the low-latency SFX engine during volume adjustments.
- **Audio**: Integrated `playConnectionSFX` in `audio.js` and wired to `network.js` to provide auditory feedback for the connection pulse.
- **Audio**: Integrated `playBattleStartSFX` in `audio.js` and wired to `network.js` to provide high-intensity feedback for match starts.
- **Audio**: Implemented character-based victory/defeat voice lines in `audio.js` and wired to `app.js` for NPC match conclusions.
- **Audio Hardening**: Migrated volume assignments to centralized setters in `audio.js` to ensure consistent persistence and prevent ESM binding errors.
- **Audio Hardening**: Wired `initAudioContext` to the wallet connection gesture in `wallet.js` to satisfy browser autoplay policies.
- **Audio Hardening**: Implemented `initAudioContext` fallbacks in `app.js` (Avatar Confirm) and `game.js` (Matchmaking/Challenges) to ensure robust SFX activation.
- **Visuals**: Integrated `syncBoardParticles` into `app.js:syncUI` for continuous ambient board effects during active gameplay.
- **Visuals**: Implemented `triggerConnectionPulse` in `particles.js` and wired into the `network.js` challenge handshake for tactical match-start feedback.
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
- **Git Maintenance**: Purged large `.vsix` file (219MB) and flattened history.
- **Emergency Restoration**: Restored critical economic hardening, reputation multipliers, and state clobbering fixes lost during history flattening.
- **Continuity**: Hardened `InitialRewards` persistence via `season.json` to ensure economic state survives restarts.
- **Git Repair**: Repository history flattened to resolve a 219MB blob rejection from GitHub; strictly enforced `*.vsix` exclusion in `.gitignore`.
- **Continuity**: Verified that flattening history preserves all current domain-separated logic and security hardening.
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
- **Network Sync**: Hardened `handleServerMessage` in `network.js` to call `window.SyncClubs` for real-time territory state alignment.
- **Market Ticker**: Hardened NPC identification in `updateMarketTicker` and synchronized canvas rendering with `.npc-taunt` purple/italic styles.
- **Combat Visuals**: Hardened `.flip-capture` animation trigger in `app.js` with reflow forcing to ensure reliable re-triggering during chain reactions.
- **Club Management**: Hardened `openClubFoundry` in `app.js` to dynamically filter out claimed territories from the selection dropdown and added a11y attributes.
- **Criminality UI**: Completed `renderRumorBoard` in `criminality.js` with functional real-time MM:SS countdown timers and robust ingestion logic.

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
- **Git Maintenance**: Troubleshooting `.vsix` history purge (219MB). Previous attempt failed due to missing `filter-repo` tool. Moving to manual `filter-branch` or tool installation.
9. Complete: Hardened Portfolio UI in `economy.js` to correctly render Jailed, Kidnapped, and Hostage card states with associated tactical actions.
11. Complete: Hardened `handleBuyBlackMarket` in `black_market_service.go` with integer-based precision rounding and player state existence guards.
12. Complete: Hardened `handleHeist` in `club_service.go` with integer-based precision rounding for looting and Fence Fees.
13. Complete: Hardened `handlePayRansom` in `handlers_criminality.go` with integer-based precision rounding for Laundering Taxes.
14. Complete: Hardened `distributeShopRevenueLocked` in `club_service.go` with integer-based precision rounding for Regional Governor Taxes.
15. Complete: Hardened `processMojoDecay` in `lobby_manager.go` with percentage-based dynamic scaling for high-ranking clubs.
16. Complete: Hardened Sybil protection and VBV accounting in `onboarding_service.go` by fixing variable mismatches and refund logic.
11. Complete: Updated `txn2` construction in `onboarding_service.go` to correctly execute an ARC-200 transfer of 1 VBV and fixed refund variable typos.
17. Complete: Hardened `checkAssetOptIn` in `oracle_service.go` with case-insensitive network mapping and timeout contexts for Voi box lookups.
18. Complete: Hardened `handleTradeShares` in `market_service.go` with post-trade reputation recalculation and target resolution error handling.
19. Complete: Hardened NPC intelligence in `market_service.go` by expanding `generateNPCCommentary` to handle Aggressiveness and Meta-aware taunts.
20. Complete: Hardened `processPlaystyleDecay` in `lobby_manager.go` with behavioral normalization and state-bloat pruning to maintain tactical relevance.
21. Complete: Hardened `updatePlayerPlaystyleTendenciesLocked` with dynamic intensity weighting for tournaments and resolved potential recursive deadlock.
22. Complete: Hardened AI potential evaluation in `main.go` by fixing rule name alignment and ensuring simulation logic correctly accounts for ownership shifts during combo chain reactions.
23. Complete: Hardened `serverCheckCaptures` in `battle_service.go` by fixing an attribution bug where combo captures were incorrectly identified after the flip.
24. Complete: Hardened `verifyWinner` in `battle_service.go` to prevent 'Capture Amnesty' by triggering Fallen Penalty jailing before Sudden Death redistribution.
25. Complete: Hardened `MatchHistory` and `verifyWinner` to archive snapshotted player attributes and implemented card fatigue persistence.
26. Complete: Hardened `handleReward` in `faucet_service.go` with identity verification and integrated Bounty payouts into the `dispatchReward` atomic group.
27. Complete: Hardened `tournament_manager.go` by applying reputation multipliers to payouts and implementing full Faucet reconciliation.
25. Complete: Hardened `initiatePairedMatch` in `lobby_manager.go` to authoritative snapshot BoardMoods, Rules, and Player Attributes for spectator accuracy.
28. Complete: Hardened `determineTop5` in `tournament_manager.go` with reputation-based tie-breakers for ranking losers of the same round.
29. Complete: Hardened tournament bracket progression by removing connection requirements for advancement and implementing DNF/Draw resolution safety.
30. Complete: Hardened tournament buy-in verification in `tournament_manager.go` by utilizing dynamic `PowerDivisor` from network configurations for asset decimal precision.
31. Complete: Hardened `distributeTournamentKickback` in `server.go` to utilize network-specific precision divisors and synchronized all call sites.
32. Complete: Hardened `verifyBuyInTransaction` in `oracle_service.go` with case-insensitive network resolution and support for native payment verification.
33. Complete: Hardened `handlePurchaseTerritory` in `club_service.go` with dynamic precision divisors and robust network resolution for district acquisitions.
34. Complete: Implemented `calculateMojoGain` in `club_service.go` to reward shop turnover and heist defense with Governor-weighted scaling.
35. Complete: Hardened `refreshRegionalRoles` in `lobby_manager.go` to trigger the GOVERNOR achievement upon reaching the 2-territory threshold.
36. Complete: Hardened `CalculateReputation` in `economy_service.go` by implementing variable-weight bonuses for milestone achievements like GOVERNOR and ARENA_LEGEND.
37. Complete: Hardened `handlePurchaseTerritory` in `club_service.go` to atomically trigger GOVERNOR status and achievement bonuses during district acquisition.
38. Complete: Hardened `handleCreateClub` and `handleJoinClub` in `club_service.go` with dynamic precision divisors and robust map initialization.
39. Complete: Hardened `handleHirePlayer` in `employment_service.go` by integrating reputation recalculation and the CAREER_START achievement.
40. Complete: Hardened `handleSetSalary` in `employment_service.go` to trigger the EXECUTIVE_PAY achievement for contracts >= 500 $VBV.
41. Complete: Hardened `startSalaryDispenser` in `career.go` with precision-rounded Outlaw Taxes and reputation synchronization for criminal employees.
42. Complete: Hardened `processAuctions` in `auction_service.go` to correctly handle item and fund transfers for auction settlement and no-bid returns, including commission distribution.
43. Complete: Implemented Mojo Bonuses and Role/Master-tier requirements in `shop_registry.go` to complete the tiered industrial unlock system.
44. Complete: Hardened `handlePurchaseItem` in `lobby_manager.go` to enforce Role, Mojo, and Master-tier requirements during shop transactions.
45. Complete: Updated `openShopsOverlay` in `economy.js` to visually distinguish Master-tier items and display career/mojo requirements.
46. Complete: Hardened `handleHeist` in `club_service.go` with Regional Security bonuses and Master-tier hardware synergies.
47. Complete: Hardened `processMojoDecay` in `lobby_manager.go` with tiered decay rates for Regional Governors to ensure sector accountability.
48. Complete: Hardened `calculateMojoGain` in `club_service.go` by implementing Regional Security Synergy and hardware trap weighting for defense events.
49. Complete: Hardened `handleSpreadRumor` in `handlers_rumor.go` with Governor Tax redistribution and resolved recursive deadlock vulnerabilities.
50. Complete: Hardened `generateNPCCommentary` in `market_service.go` to prevent concurrent map access panics and integrated Envoi names for taunt immersion.
51. Complete: Verified `observeGlobalSentiments` in `market_service.go` is safe from concurrent map access panics due to correct mutex usage.
52. Complete: Implemented "Governor's Tax" on tournament pots in `tournament_manager.go`, routing 5% of the pot to the club controlling "arena_center".
53. Complete: Hardened `handleCreateLease` and `handleTakeLease` in `club_service.go` by implementing micro-unit precision for revenue splitting and resolving recursive deadlocks.
54. Complete: Hardened `processLeaseExpirations` in `club_service.go` to correctly update borrower reputation upon leased card return.
55. Complete: Updated `openClubLeaseBoard` in `app.js` to display the Faucet tax and Club commission breakdown for industrial leases.
56. Complete: Hardened `handleCreateAuction` in `auction_service.go` with item existence validation, empty bundle checks, and resolved recursive deadlock.
57. Complete: Hardened `transferBundleItems` and `handleCreateAuction` in `auction_service.go` by integrating nil-safe map initialization via `ensurePlayerStatsMapsInitialized`.
58. Complete: Hardened `processAuctions` in `auction_service.go` by implementing win-tracking and the ART_COLLECTOR achievement trigger.
59. Complete: Verified `handleTradeShares` in `market_service.go` correctly weights the ART_COLLECTOR achievement through its contribution to player Reputation.
60. Complete: Updated `switchPortfolioTab` in `economy.js` to display Gallery Victories, synchronized through `lobby_manager.go` and `main.go`.
61. Complete: Refined `handleCreateClub` in `club_service.go` to explicitly initialize the Mojo field to 0 for new organizations.
62. Complete: Consolidated Black Market logic into `economy.js` and updated requirements notice to display user's current Cunning and Wanted Level relative to thresholds.
63. Complete: Hardened `handleHeist` in `club_service.go` by resolving recursive deadlock vulnerabilities in the audit and notification paths.
64. Complete: Hardened `handleKidnapRequest` in `handlers_criminality.go` by resolving recursive deadlock vulnerabilities in the error-handling paths.
65. Complete: Hardened `handleBailCard` in `handlers_criminality.go` by resolving recursive deadlocks and integrating reputation synchronization for bailed assets.
66. Complete: Hardened `processInsuranceRecovery` in `handlers_criminality.go` by resolving recursive deadlocks and integrating reputation synchronization for victims and perpetrators.
67. Complete: Hardened `handlePayRansom` and `handleReleaseHostage` in `handlers_criminality.go` by resolving recursive deadlocks and implementing reputation recalculation for both parties.
68. Complete: Hardened `CalculateReputation` in `economy_service.go` with jailing penalties and resolved clobbering risks in `handleHeist`.
69. Complete: Hardened `handleBailCard` in `handlers_criminality.go` by resolving remaining recursive deadlock vulnerabilities in the error paths.
70. Complete: Updated `openBountyBoard` in `criminality.js` to display target Mojo and current Employer (Club Name) for high-fidelity hunting.
71. Complete: Hardened `dispatchReward` in `faucet_service.go` to scale Bounty Rewards based on the Hunter's Mojo Tier.
72. Complete: Hardened `processFallenPenaltyJailLocked` in `battle_service.go` to award Mojo to the capturing player's club for each card seized, and updated `calculateMojoGain` in `club_service.go`.
73. Complete: Created `LICENSE` file in root directory stipulating proprietary codebase with open-source sound asset exception.
74. Complete: Implemented Regional Power Boost (+5%) for club members in Region-controlled territories across server, WASM, and UI.
75. Complete: Hardened `handleUnregister` in `lobby_manager.go` with scaled tournament DNF penalties and fixed state synchronization bugs.
76. Complete: Implemented 'Bounty Ticker' in `economy.js` and wired into `network.js` to scroll live rewards for hunting high-Wanted outlaws.
77. Complete: Prepared and documented Git workflow for migrating current build from dev2 branch to virtualbabes hackathon entry repository.
78. Complete: Refined and documented the Git push process to handle multi-account credential conflicts during hackathon submission.
79. Complete: Resolved Git push 'rejected' error via force-push protocol.
80. Complete: Updated README.md for accuracy regarding Social Economic Simulation pillars and Beta status.
81. Complete: Created User_manual.md as a comprehensive guide for players.
82. Complete: Synced local changes to `slapkarnts dev2` branch.
80. Complete: Updated README.md for accuracy regarding Social Economic Simulation pillars and Beta status.
81. Complete: Created User_manual.md as a comprehensive guide for players.
80. Complete: Updated `README.md` for accuracy and created `User_manual.md` for players.

84. Complete: Hardened `handleTournamentRegister` with concurrency throttling and duplicate verification guards to protect indexer stability.
85. Complete: Hardened `verifyBuyInTransaction` in `oracle_service.go` with 429 retry policy and robust status code handling.
86. Complete: Hardened `handleTournamentHistory` in `tournament_manager.go` with 429 retry policy and improved error handling for indexer responses.
87. Complete: Hardened `checkAssetOptIn` in `oracle_service.go` with 429 retry policy for node and indexer requests.
88. Complete: Hardened `syncStatsFromBlockchain` in `oracle_service.go` with 429 retry policy and robust status code handling.
89. Complete: Hardened `getVerifiedCards` and `getVerifiedCardsCrossChain` in `oracle_service.go` with 429 retry policies for multi-chain discovery.
90. Complete: Hardened `refreshGlobalLeaderboard` in `oracle_service.go` with 429 retry policy and robust status code handling.
91. Complete: Hardened `loadOnboardedWalletsFromIndexer` in `oracle_service.go` with 429 retry policy for each page of the historical scan.
92. Complete: Hardened `checkVaultBalanceOnChain` in `oracle_service.go` with 429 retry policy for ARC-200 application box lookups.
- [x] Audited Ephemeral Cleanup: Verified `cleanupNonces` safety for spectating sessions.
