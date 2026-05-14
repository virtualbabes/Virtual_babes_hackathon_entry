# NFT Seduction Tasks

## Remaining Critical Launch Readiness
- [x] Resolve Git push rejection by flattening repository history to purge large blobs.
- [ ] Perform 16-player tournament stress tests.
- [x] **Accessibility Audit**: Add `<label>` elements or `aria-label` attributes to all form inputs/selects in `index.html`.
- [x] **A11y Metadata**: Ensure all interactive elements have `title` and/or `placeholder` attributes.
- [x] **CSS Refactoring**: Extract all inline `style="..."` attributes from `index.html` into `_main-layout.scss` or `_neon-glass.scss`.
- [x] **Cross-Browser Hardening**: Added `-webkit-` and `-moz-` prefixes for `text-size-adjust` and custom scrollbars.
- [x] **Property Correction**: Switched `user-drag` to `-webkit-user-drag` for standard compliance.
- [x] **Frontend Refactor**: Removed redundant `initWebSocket` and `handleServerMessage` definitions and associated global variables from `app.js`.
- [x] **Admin Panel Refactor**: Eliminated all remaining inline styles from the Admin Control Panel in `index.html`.
- [x] **Ticker Audit**: Hardened NPC identification and `.npc-taunt` style synchronization in `updateMarketTicker`.
- [x] **Network Sync**: Call `window.SyncClubs(msg.payload.clubs)` in `handleServerMessage` on `lobby_update`.
- [x] Implement canvas-based 'sparks' particle effects and integrate into SCSS.
- [x] Audit `renderCardHTML` function in `ui.js` for performance bottlenecks and re-allocations.
- [x] **A11y Typography**: Hardened neon text contrast and outlines in `_typography.scss`.
- [x] **Documentation Sync**: Synchronized `Game_expansion_plan.md` with implementation reality to establish the Beta baseline.
- [x] **Particle Immersion**: Hardened `particles.js` with dynamic behavior shifting based on Cunning, Mojo, and Club Industry types.
- [x] **Foundry Visuals**: Integrated dynamic `triggerFoundryFusion` call in `economy.js` for Club Industry types.
- [x] **Ambient Immersion**: Implemented tile mood motes in `particles.js` and throttled audio cues in `audio.js`, triggered by the `board_moods` state in `ui.js`.
- [x] **Performance Audio**: Enhanced `audio.js` with `AudioContext` for low-latency SFX and polyphonic support.
- [x] **Combat Feedback**: Integrated `triggerConnectionPulse`, `playConnectionSFX`, and `playBattleStartSFX` for multiplayer match initialization in `network.js`.
- [x] **Audio Integration**: Wired `syncSFXGain` into `app.js` to normalize Web Audio gain during volume changes.
- [x] **Audio Feedback**: Implemented character-based victory/defeat voice lines for match conclusions.
- [x] **Audio Policy**: Wired `initAudioContext` to user gestures in `wallet.js` for autoplay compliance.
- [x] **Audio Fallbacks**: Integrated `initAudioContext` into high-traffic match and entry gestures in `app.js` and `game.js`.
- [x] **Ambient Immersion**: Integrated `syncBoardParticles` into `app.js:syncUI` for continuous ambient board effects.
- [x] **Combat Feedback**: Integrated `triggerConnectionPulse` for multiplayer match initialization in `network.js`.
- [x] **A11y Audit**: Hardened market ticker legibility and motion accessibility in `_economy.scss`.
- [x] **A11y Audit**: Hardened heist risk meter color contrast and motion accessibility in `_criminality.scss`.
- [x] **A11y Audit**: Hardened social status contrast and locked-state legibility in `_social.scss`.
- [x] **UI Performance**: Optimized achievement badge and social hub rendering in `_social.scss`.
- [x] **UI Performance**: Optimized shop item grid rendering and transitions in `_shops.scss`.
- [x] **UI Performance**: Optimized market ticker and auction gallery rendering in `_economy.scss`.
- [x] Fix `CardID` escrow logic in `auction_service.go` and `economy_processing.go`.
- [x] Audit `initWebSocket` message handler in `network.js` for high-frequency move overhead; implemented batching logic.
- [x] Resolve Faceplate Residue: Implemented Mojo/Cunning bonuses for Faceplates.
- [x] Audit `processMojoDecay` and update `club.LastActivity` in economic handlers.
- [x] Implement 'equip_cosmetic' message type in `lobby_manager.go`.
- [x] Audit `handleHeist` in `club_service.go` for `GetEffectiveCunning` integration.

## Technical Debt & Refactoring (Completed / Hardened)
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
- [x] Resolved recursive deadlocks in `courthouse_service.go`.
- [x] Hardened `common_types.go` by adding missing `EquippedFaceplate` field to `PlayerStats`.
- [x] Cleaned up corrupted JavaScript fragments and media query syntax errors in `_overlays.scss`.
- [x] Resolved SCSS syntax errors (orphans) in `_dashboard.scss`, `_cards.scss`, `_social.scss`, `_territory.scss`, and `_overlays.scss`.
- [x] Cleaned up accidentally included JavaScript logic in `_overlays.scss`.
- [x] Verified `_shops.scss` and `_economy.scss` are free of JavaScript fragments.
- [x] Resolved cross-browser compatibility issues for `scrollbar-width` and `user-drag` in `_overlays.scss` and `app.js`.
- [x] Resolved SCSS syntax errors and Safari compatibility issues in `layouts/_dashboard.scss`.
- [x] Resolved Safari compatibility issues for `backdrop-filter` in `layouts/_main-layout.scss`.
- [x] Resolved cross-browser compatibility issues for `text-size-adjust` in `base/_reset.scss`.
- [x] Resolved Safari compatibility issues for `backdrop-filter` in `base/_dashboard.scss`.
- [x] Hardened `oracle_service.go:loadOnboardedWalletsFromIndexer` with explicit error and status checks.
- [x] Hardened `oracle_service.go:getVerifiedCards` with explicit logging for non-200 indexer responses.
- [x] Resolved recursive deadlocks in `employment_service.go`.
- [x] Resolved recursive deadlocks in `loan_service.go`.
- [x] Resolved recursive deadlocks in `market_service.go:handleTradeShares`.
- [x] Hardened `faucet_service.go:dispatchReward` for mnemonic-to-private-key failures.
- [x] Resolved recursive deadlock and scope errors in `faucet_service.go`.
- [x] Resolved missing `strings` import in `onboarding_service.go`.
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

## Recommendations for Aesthetics and Functional Immersion Improvements (Partially Completed)
- [x] Add shop overlays (using _shops.scss) for item purchasing, with animated item reveals and purchase confirmations.
- [x] Implement criminality UI (using _criminality.scss) for heist planning, with risk meters and success animations.
- [x] Implement the 3D territory map visualization in app.js using the remaining orphaned _territory.scss styles.
- [x] Refactor Shop Overlay to use high-fidelity grid and category filtering from `_shops.scss`.
- [x] Implement the Heist Planning interface in app.js using the orphaned _criminality.scss styles to activate the tactical grid.
- [x] Implemented Neon Social Hub (Alliances, Career, Valor/Achievements).
- [x] Use _economy.scss for auction houses and portfolio management overlays.
- [ ] Add more gradient animations in _neon-glass.scss (e.g., pulsing borders on active elements).
- [ ] Implement dynamic background shifts based on game phase (e.g., red tint during criminal actions, using CSS variables set by `app.js`).
- [ ] Enhance hover effects: Add micro-animations (scale + glow) to buttons/cards using _animations.scss.
- [ ] Introduce particle effects for more events (e.g., victory sparks, defeat smoke) by expanding the existing particle system in `app.js`.
- [ ] Expand _animations.scss with game-specific keyframes (e.g., card flip with neon trails, territory conquest waves).
- [ ] Add loading states with shimmer effects (`.animate-shimmer`) for async operations like NFT resolution.
- [ ] Implement sound cues tied to animations (e.g., hover sounds, capture effects) by integrating with existing audio in `app.js`.
- [ ] Enhance tooltips (already in `app.js`) with animated reveals and contextual info.
- [ ] Add micro-interactions: Button presses with scale-down, form inputs with focus glows.
- [ ] Use _spacing.scss utilities for responsive layouts on mobile, ensuring immersion on all devices.
- [ ] Tie collective-intelligence.js taunts to UI animations (e.g., taunt text with typewriter effect).
- [ ] Add mood-based UI changes (e.g., volatile card captures trigger screen shake).
- [ ] Clean Up: Remove or comment out orphaned CSS to reduce bundle size, or add TODO comments for future implementation.
- [ ] Modular Enhancements: Use SCSS variables for consistent theming (e.g., dynamic color shifts for different game modes).
- [ ] Testing: After implementing features, ensure CSS is used (e.g., via browser dev tools to check for unused rules).
- [ ] In `app.js`: Add functions like `openShopsOverlay()` using the orphaned styles.
- [ ] In index.html: Add hidden overlay containers for shops/criminality/territories.
- [ ] In _animations.scss: Add `.animate-card-capture { animation: capture-burst 0.5s; }` for better feedback.
- [ ] Overall: Increase use of CSS custom properties for dynamic theming (e.g., `--arena-mood: red` for criminal phases).

## Next Tactical Steps
- [x] Implement the Social Panel alliance management UI in app.js using the orphaned _social.scss styles.
- [x] Audit the handleHeist logic in club_service.go to ensure the success probability matches the frontend heuristic exactly.
- [x] Mainnet Security: Implemented runtime injection for WalletConnect Project ID via environment variables.
- [x] Infrastructure: Defined Render (Backend/WASM), GitHub (Assets), and Carrd (Landing) stack.
- [ ] Production Readiness: Final UI polishing for mobile responsiveness and Carrd -> Render link validation.
- [x] Audit `CalculateReputation` in `economy_service.go` to ensure the Spreader Multiplier correctly applies to players with zero wins but high rumor activity. (Changed to additive bonus)
- [x] Added missing SCSS variables: $font-size-5xl, $font-size-6xl, $line-height-tight, $line-height-normal, $line-height-relaxed, $border-radius-2xl, $border-radius-3xl, $shadow-2xl, $color-neon-pink to _variables.scss.
- [x] Fixed server port conflict: Updated package.json start script to use PORT=8083 instead of default 8082 (conflicted with Algorand node).
- [x] Fixed ES6 module loading: Added type="module" to app.js script tag in index.html.
- [x] Added placeholder favicon files: Created favicon.ico and favicon.png to prevent 404 errors.
- [x] Resolved Safari compatibility issues for `backdrop-filter` in `base/_dashboard.scss`.
- [x] Hardened `oracle_service.go:loadOnboardedWalletsFromIndexer` with explicit error and status checks.
- [x] Hardened `oracle_service.go:getVerifiedCards` with explicit logging for non-200 indexer responses.
- [x] Resolved recursive deadlocks in `employment_service.go`.
- [x] Resolved recursive deadlocks in `loan_service.go`.
- [x] Resolved recursive deadlocks in `market_service.go:handleTradeShares`.
- [x] Hardened `faucet_service.go:dispatchReward` for mnemonic-to-private-key failures.
- [x] Resolved recursive deadlock and scope errors in `faucet_service.go`.
- [x] Resolved missing `strings` import in `onboarding_service.go`.
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
- [x] Comprehensive SCSS/CSS UI/UX audit and HTML structural analysis (SCSS alignment for shops/territory overlays completed).
- [x] Merge ReadMe.txt into README.md.
- [x] Wire the Kidnap Selection UI in `app.js` using the orphaned criminality SCSS styles.
- [x] Implement the 3D territory map visualization in app.js using the remaining orphaned _territory.scss styles.


### Recommendations for Aesthetics and Functional Immersion Improvements

#### 1. **Implement Orphaned Features for Immersion**
   - **Suggestions**:
     - [x] Add shop overlays (using _shops.scss) for item purchasing, with animated item reveals and purchase confirmations.
     - [x] Implement criminality UI (using _criminality.scss) for heist planning, with risk meters and success animations.
     - [x] Implement the 3D territory map visualization in app.js using the remaining orphaned _territory.scss styles.
     - [x] Refactor Shop Overlay to use high-fidelity grid and category filtering from `_shops.scss`.
     - [x] Implement the Heist Planning interface in app.js using the orphaned _criminality.scss styles to activate the tactical grid.
     - [x] Implemented Neon Social Hub (Alliances, Career, Valor/Achievements).
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
- [x] Mainnet Security: Final verification of mnemonic encryption and environment variable injection for production.
4. **Production Readiness**: Final UI polishing for mobile responsiveness and WalletConnect v2 finalization.