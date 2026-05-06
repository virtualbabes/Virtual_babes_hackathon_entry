package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// processLoans checks for defaulted loans and handles collateral liquidation.
func (l *Lobby) processLoans() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	now := time.Now()

	for id, loan := range l.loans {
		if loan.Status == "active" && now.After(loan.DueAt) {
			loan.Status = "defaulted"

			// Residual Value: 15% of the loan amount is returned as Market Tokens
			tokenReward := uint64(float64(loan.LoanAmount) * 0.15)

			borrowerWallet := loan.BorrowerWallet
			borrowerStats, exists := l.leaderboard[borrowerWallet]
			if exists {
				borrowerStats.MarketTokens += tokenReward
				borrowerStats.Reputation -= 50
				if borrowerStats.Reputation < 0 {
					borrowerStats.Reputation = 0
				}
				// RECONCILE: Ensure calculated stats are in sync
				borrowerStats.Reputation = l.CalculateReputation(borrowerStats)
				l.leaderboard[borrowerWallet] = borrowerStats
				l.leaderboard[borrowerWallet] = borrowerStats

				l.sendToClient(l.getClientIDFromWallet(borrowerWallet), Envelope{
					Type:    "admin_notification",
					Payload: json.RawMessage(fmt.Sprintf(`{"text":"🚨 <b>LOAN DEFAULTED:</b> Collateral moved to Black Market. You received %.2f Market Tokens as equity."}`, float64(tokenReward)/1000000.0)),
				})
			}

			// Update playstyle on loan default (Internal call to avoid deadlock)
			l.updatePlayerPlaystyleTendenciesLocked(borrowerWallet, false, [2]int{}, []int{}, false)
			l.logAdminAudit("LOAN_LIQUIDATED", borrowerWallet, fmt.Sprintf("ID: %s, Tokens: %d", loan.ID, tokenReward))

			// Add the defaulted loan to the black market
			l.blackMarket = append(l.blackMarket, *loan)

			delete(l.loans, id)
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		}
	}
}

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
let activeCardId = null; // Tracks the card you clicked in your hand
let aiThinking = false; // To track if AI is currently performing a move
let lastBoardState = Array(9).fill(null); // Track state to detect captures
let socket = null;
let currentChallengerId = null; // Stores the ID of the player who sent the current challenge
let reconnectAttempts = 0; // Tracks WebSocket reconnection attempts for identity sync
let nonceResolver = null;
let currentOpponentId = null;   // The player we are currently battling
let walletProvider = null;      // Current active provider (nautilus, kibisis, etc.)
let myClientId = null;
let currentTournamentPage = 1;
const tournamentLimit = 10;
let totalTournaments = 0;
let spectatorMatchState = null; // Stores P1/P2 mapping for spectators
let lastTournamentData = null;  // Cache for Go WASM synchronization
let isVerified = false;         // Global verification status
let seasonEnd = null;
let seasonTimerInterval = null;
let identitySyncTimeout = null;
let lastTauntPhase = null;      // Tracks narrative state to prevent duplicate taunts
let lastTauntTurn = null;
let activeRumors = []; // Global array to hold active rumors

// --- Ticker State ---
let tickerItems = [];
let tickerOffset = 0;
let tickerAnimId = null;

// Global Audio Controls
let masterVolume = 0.5;
let sfxVolume = 0.5;
let musicVolume = 0.5;

let myPlayerIndex = 0;          // 0 for P1, 1 for P2
let userAddress = null;
let currentAvatarUrl = null;
let cropState = { zoom: 1.0, x: 0, y: 0 }; // Shared state for aspect-ratio aware cropping
let inMatchmakingQueue = false;
let matchHistorySaved = false;
let cachedAdminHeaders = null;
let availableNetworks = {};
let globalClubs = {};
let lastLobbyPlayers = []; // Cache for portfolio valuation
let adminFocusNetwork = ""; // Renamed to clarify its purpose
let payoutAddress = localStorage.getItem("vbabes_payout_address") || null;
let linkedWallets = JSON.parse(localStorage.getItem("vbabes_linked_wallets") || "[]");
let ignoredReporters = new Set(JSON.parse(localStorage.getItem("vbabes_ignored_reporters") || "[]"));

// --- Global Deployment Configuration ---
const CONFIG = (() => {
	const isLocal = window.location.hostname === "localhost" || window.location.hostname === "127.0.0.1";
	const backendHost = window.location.host; // Dynamically uses current host (localhost:8082 or Render URL)
	return {
		IS_LOCAL: isLocal,
		BACKEND_URL: backendHost,
		API_BASE: (window.location.protocol === "https:" ? "https://" : "http://") + backendHost,
		// Production CDN: Link to the Public folder in the deploy branch
		ASSET_URL: isLocal ? "/" : "https://raw.githubusercontent.com/slapkarnts/VOiconomy-faucet/deploy/Public/",
		WC_PROJECT_ID: 'your_walletconnect_project_id', // Matches .env.example placeholder
		VOI_CHAIN_ID: 'algorand:wGHE2Pwd1-YdV4EuJFy9u6C24-L-2B05',
		ALGO_CHAIN_ID: 'algorand:mainnet-v1.0',
		VAULT_ADDRESS: null,                      // Dynamic: Synced from server on connect
		VBV_ASSET_ID: null,                       // Dynamic: Synced from server on connect
		AVOI_ASSET_ID: null                       // Dynamic: Synced from server on connect
	};
})();

let signClient = null; // WalletConnect State
let wcModal = null;    // WalletConnect Modal State

let lastPingTime = null;
let currentLatency = null;

// --- Particle System ---
let particleCanvas = null;
let particleCtx = null;
let particles = [];
let particleAnimationId = null;
// --- Asset Symbol Resolution ---
const assetCache = {
	"40227315": "$VBV" // Pre-seed default 
};
const envoiCache = {};
let tooltipEl = null;

function getCachedEnvoiName(address) {
	if (!address || address === "TBD" || address === "BYE") return address || "TBD";
	return envoiCache[address] || (address.substring(0, 6) + "..." + address.substring(address.length - 4));
}

async function resolveAssetSymbol(assetId) {
	if (assetCache[assetId]) return assetCache[assetId];
	
	const state = window.GetGameState();
	
	// Use the active network's indexer URL from the server's availableNetworks
	// This assumes the assetId is from the currently active network.
	// For cross-chain assets, this needs to be more sophisticated (e.g., pass network with assetId).
	const currentNetworkConfig = Object.values(availableNetworks).find(
		n => n.network_name.includes(state.network) // Match "VOI" with "Voi Mainnet"
	);
	const baseUrl = currentNetworkConfig ? currentNetworkConfig.indexer_url : "";

	// Fallback if no specific indexer URL is found or network is unknown

	if (!baseUrl) {
		assetCache[assetId] = `$`;
		return assetCache[assetId];
	}

	try {
		const response = await fetch(`/collections?contractId=`);
		if (response.status === 404) {
			// Asset not found on indexer (common for new/mock assets)
			assetCache[assetId] = `$`;
			return assetCache[assetId];
		}
		if (!response.ok) throw new Error(`HTTP ${response.status}`);
		const data = await response.json();
		
		if (data.collection && data.collection.length > 0) {
			const col = data.collection[0];
			let symbol = "";

			// 1. Explicitly check for 'Symbol' or 'UnitName' in metadata
			if (col.firstToken?.metadata) {
				try {
					const tokenMeta = JSON.parse(col.firstToken.metadata);
					symbol = tokenMeta.symbol || tokenMeta.unitName || "";
				} catch (e) { /* metadata not JSON, ignore */ }
			}

			// 2. Fallback to name-based heuristic if no explicit symbol
			if (!symbol) {
				let name = col.firstToken?.metadata?.name || col.name || "";
			
				symbol = name.toUpperCase();
				if (symbol.length > 6) {
					const words = name.split(' ');
					if (words.length > 1) {
						symbol = words.map(w => w[0]).join('').toUpperCase();
					} else {
						symbol = name.substring(0, 4).toUpperCase();
					}
				}
			}
			
			if (!symbol.startsWith('$')) symbol = '$' + symbol;
			assetCache[assetId] = symbol;
			return symbol;
		}
	} catch (err) {
		console.warn(`[API] Asset  resolution failed:`, err);
	}
	
	// Robust Fallback: Cache the ID as a symbol to prevent repeated failed requests
	assetCache[assetId] = `$`;
	return assetCache[assetId];
}

function getAssetSymbol(assetId) {
	return assetCache[assetId] || `$`;
}

// Helper to get network config from the global availableNetworks map
function getNetworkConfig(networkShortName) {
	if (networkShortName === "VOI") {
		return availableNetworks["Voi Mainnet"];
	} else if (networkShortName === "ALGO") {
		return availableNetworks["Algorand Mainnet"];
	}
	return null;
}

// --- WalletConnect Initialization ---
async function initWalletConnect() {
	if (!CONFIG.WC_PROJECT_ID || CONFIG.WC_PROJECT_ID === 'YOUR_WALLETCONNECT_PROJECT_ID') {
		console.warn("[WC] WalletConnect Project ID not configured.");
		return;
	}

	try {
		// The UMD build of sign-client exports globally as SignClient
		const SignClient = window.SignClient;
		if (!SignClient) return;

		const WalletConnectModal = window.WalletConnectModal;
		if (WalletConnectModal) {
			wcModal = new WalletConnectModal.WalletConnectModal({
				projectId: CONFIG.WC_PROJECT_ID,
				chains: [CONFIG.VOI_CHAIN_ID, CONFIG.ALGO_CHAIN_ID]
			});
		}

		signClient = await SignClient.init({
			projectId: CONFIG.WC_PROJECT_ID,
			metadata: {
				name: "Virtualbabes Arena",
				description: "The premier NFT Seduction battleground on Voi.",
				url: window.location.origin,
				icons: [(CONFIG.IS_LOCAL ? window.location.origin : "") + CONFIG.ASSET_URL + "Assets/logo.png"],
			},
		});

		// Handle session events
		signClient.on("session_event", ({ event }) => { console.log("[WC] Event:", event); });
		signClient.on("session_update", ({ topic, params }) => { console.log("[WC] Session Updated:", params); });
		signClient.on("session_delete", () => { 
			console.log("[WC] Session Deleted");
			disconnectUserWallet();
		});

		// Restore existing session
		const sessions = signClient.session.getAll();
		if (sessions.length > 0) {
			const session = sessions[0];
			const account = session.namespaces.algorand.accounts[0];
			const addr = account.split(":")[2];
			walletProvider = 'walletconnect';
			console.log("[WC] Session Restored:", addr);
			updateWalletUI(addr);
		}

		console.log("[WC] Initialization Complete.");
	} catch (err) {
		console.error("[WC] Initialization Failed:", err);
		showToast("WalletConnect failed to initialize.", "error");
	}
}

// 1. Initialize Go WASM Engine
window.onload = async () => {
	const go = new Go();
	try {
		const response = await fetch("main.wasm");
		const buffer = await response.arrayBuffer();
		const obj = await WebAssembly.instantiate(buffer, go.importObject);
		
		// Initialize volume sliders
		document.getElementById("master-volume").value = masterVolume;
		document.getElementById("music-volume").value = musicVolume;
		document.getElementById("sfx-volume").value = sfxVolume;
		go.run(obj.instance);
		if (window.SetApiBase) window.SetApiBase(CONFIG.API_BASE);
		if (window.SetAssetBase) {
			window.SetAssetBase(CONFIG.ASSET_URL);
			// Set specific CSS variables for background textures as CSS url() doesn't support concatenation
			const base = CONFIG.ASSET_URL;
			document.documentElement.style.setProperty('--bg-arena-floor', `url(Assets/Textures/arena_floor.png)`);
			document.documentElement.style.setProperty('--bg-glass-texture', `url(Assets/Textures/glass_texture.webp)`);
			// NEW: Define dynamic arena floor textures
			document.documentElement.style.setProperty('--texture-solo', `url(Assets/Textures/arena_solo.webp)`);
			document.documentElement.style.setProperty('--texture-challenge', `url(Assets/Textures/arena_challenge.webp)`);
			document.documentElement.style.setProperty('--texture-tournament', `url(Assets/Textures/arena_tournament.webp)`);
			document.documentElement.style.setProperty('--texture-semi', `url(Assets/Textures/arena_semi_final.webp)`);
			document.documentElement.style.setProperty('--texture-final', `url(Assets/Textures/arena_final.webp)`);
		}
		document.getElementById("engine-status").innerHTML = "<span class='status-active'>ACTIVE</span>";
		buildEmptyBoard();
		initWebSocket();
		initWalletConnect(); // Initialize WC alongside WS
		renderMatchHistory();
		fetchLeaderboard();
		setupCropEvents();
		initParticleSystem(); // Initialize particle system

		updatePayoutUI();
		// Check for soft-reload resume
		if (localStorage.getItem("vbabes_soft_reload") === "true") {
			const lastWallet = localStorage.getItem("vbabes_last_wallet");
			const lastProvider = localStorage.getItem("vbabes_last_provider");
			localStorage.removeItem("vbabes_soft_reload");
			if (lastWallet && lastProvider) {
				setTimeout(() => connectWith(lastProvider), 500); // Small delay for WASM stability
			}
		}
	} catch (err) {
		console.error("WASM Load Fail:", err);
		document.getElementById("engine-status").innerHTML = "<span style='color: #ff0844;'>OFFLINE</span>";
	}
	syncUI(); // Initial UI sync after WASM loads
};

// 2. Hall of Fame Rendering
async function fetchLeaderboard() {
	let data = [];
	try {
		const response = await fetch(`${CONFIG.API_BASE}/api/leaderboard`);
		if (!response.ok) throw new Error("API Unreachable");
		data = await response.json();
		localStorage.setItem("vbabes_leaderboard_cache", JSON.stringify(data));
	} catch (err) {
		console.warn("[LEADERBOARD] API offline, loading from local cache...");
		data = JSON.parse(localStorage.getItem("vbabes_leaderboard_cache") || "[]");
		if (data.length > 0) showToast("📡 Using cached rankings (Offline Mode)", "info");
	}
	
	const top10Container = document.getElementById("hof-top-10");
	const scrollerContainer = document.getElementById("hof-scrolling-content");
	const sideDisplay = document.getElementById("leaderboard-display");

	top10Container.innerHTML = "";
	scrollerContainer.innerHTML = "";
	sideDisplay.innerHTML = "";

	for (let i = 0; i < data.length; i++) {
		const entry = data[i];
		const envoiName = await resolveEnvoiName(entry.wallet);
		const tierData = window.GetTierInfo(entry.reputation);
		
		const row = document.createElement("div");
		row.className = "leaderboard-row";
		row.style.borderLeftColor = tierData.color;
		
		row.innerHTML = `
			<span class="rank-badge" style="color: ${tierData.color}">#${i+1}</span>
			<span class="player-name"> <small class="tier-label" style="color: ${tierData.color}">[${tierData.tier.toUpperCase()}]</small></span>
			<span class="player-stats">
				<b class="text-neon-green">${entry.wins}</b> Wins 
				<b class="text-neon-cyan rating-badge">${entry.best_rating || '[Z]'}</b>
				<span class="stats-separator">|</span> ${entry.reputation} REP
			</span>
		`;

		if (i < 10) {
			top10Container.appendChild(row);
		} else {
			scrollerContainer.appendChild(row);
		}
	}

	// Adjust scroller animation duration based on count
	// 11 seconds (Still) + (count * 0.5s for scroll)
	const scrollCount = Math.max(0, data.length - 10);
	const duration = 11 + (scrollCount * 0.5);
	scrollerContainer.style.animation = `hof-recursive-scroll s linear infinite`;
}

async function renderSideLeaderboard() {
	const display = document.getElementById("leaderboard-display");
	try {
		let data = [];
		try {
			const response = await fetch(`${CONFIG.API_BASE}/api/leaderboard`);
			if (!response.ok) throw new Error("API Unreachable");
			data = await response.json();
			localStorage.setItem("vbabes_leaderboard_cache", JSON.stringify(data));
		} catch (err) {
			// Fallback to cache silently for side display unless it's empty
			data = JSON.parse(localStorage.getItem("vbabes_leaderboard_cache") || "[]");
		}
		
		
		display.innerHTML = "";
		if (data.length === 0) {
			display.innerHTML = '<div class="chat-msg system">No legends yet.</div>';
			return;
		}

		for (let i = 0; i < data.length; i++) {
			const entry = data[i];
			const envoiName = await resolveEnvoiName(entry.wallet);

			// Reputation Tier Logic
			const rep = entry.reputation;
			let tier = "Iron";
			let tierColor = "#888"; // Gray
			if (rep >= 1000) { tier = "Diamond"; tierColor = "#b9f2ff"; }
			else if (rep >= 600) { tier = "Gold"; tierColor = "#ffd700"; }
			else if (rep >= 300) { tier = "Silver"; tierColor = "#c0c0c0"; }
			else if (rep >= 100) { tier = "Bronze"; tierColor = "#cd7f32"; }

			// Mardon Badge Logic (50+ Wins, 0 Disconnect Streak)
			const mardonBadge = (entry.wins >= 50 && entry.disconnect_streak === 0) ? `<span title="Mardon Badge: Elite Reliability (50+ Wins, 0 Streak)" style="color: var(--neon-green); margin-left: 5px; cursor: help;">🎖️</span>` : '';

			// Ban & Streak UI Logic
			const banDate = new Date(entry.ban_expires);
			const isBanned = banDate > Date.now();
			let banInfo = '';
			if (isBanned) {
				const diff = banDate - Date.now();
				const hours = Math.floor(diff / (1000 * 60 * 60));
				const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
				banInfo = `<span title="Banned: h m remaining" style="color: #ff4b4b; margin-left: 5px; cursor: help;">🔒</span>`;
			}

			const streakWarning = entry.disconnect_streak >= 2 ? `<span title="Disconnect Streak: ${entry.disconnect_streak}" style="color: #ff4b4b; margin-left: 5px; cursor: help;">🚩</span>` : '';
			
			const div = document.createElement("div");
			div.className = "chat-msg";
			const rankColor = i === 0 ? "var(--neon-cyan)" : (i < 3 ? "var(--neon-purple)" : "white");
			
			div.innerHTML = `
				<b style="color: ">#${i+1}</b> 
				
				<span style="color: ; font-size: 0.75em; font-weight: bold; margin-right: 5px;">[]</span>
				<span style="color: var(--neon-cyan)"></span> 
				<span style="float: right; opacity: 0.8;"><b>${entry.wins}W</b> <small style="color: var(--neon-cyan); font-weight: bold;">${entry.best_rating || '[Z]'}</small> | R</span>
			`;
			display.appendChild(div);
		}
	} catch (err) {
		console.error("Leaderboard fetch failed", err);
	}
}

async function resolveEnvoiName(address) {
	if (envoiCache[address]) return envoiCache[address];
	if (address === "BYE" || address === "TBD") return address;

	const state = window.GetGameState();
	const networkConfig = getNetworkConfig(state.network);
	const baseUrl = networkConfig ? networkConfig.indexer_url : "";
	const truncated = address.substring(0, 6) + "..." + address.substring(address.length - 4);
	if (!baseUrl) {
		envoiCache[address] = truncated;
		return truncated;
	}

	try {
		const response = await fetch(`/tokens?owner=`);
		if (response.status === 404) {
			envoiCache[address] = truncated;
			return truncated;
		}
		if (!response.ok) throw new Error(`HTTP ${response.status}`);
		const data = await response.json();
		
		const suffix = state.network === "VOI" ? ".voi" : ".algo";
		const envoiToken = data.tokens?.find(t => {
			if (!t.metadata) return false;
			try {
				const meta = JSON.parse(t.metadata);
				return meta.name?.toLowerCase().endsWith(suffix);
			} catch(e) { return false; }
		});
		
		if (envoiToken) {
			const meta = JSON.parse(envoiToken.metadata);
			envoiCache[address] = meta.name;
			return meta.name;
		}
	} catch (err) {
		console.warn(`[API] Name resolution failed for :`, err);
	}
	
	envoiCache[address] = truncated; // Negative cache: prevent repeated failed fetches
	return truncated;
}

// --- Overlay Management Functions ---
function hideAllOverlays() {
	document.querySelectorAll('.overlay').forEach(el => el.classList.add('hidden'));
}
let userNFTs = []; // Local cache for filtering

async function handleWalletAction() {
	if (userAddress) {
		await disconnectUserWallet();
	} else {
		await connectUserWallet();
	}
}

// Wallet Selector
function closeWalletSelector() {
	document.getElementById("wallet-selector-overlay").classList.add("hidden");
	showMainGameContainer(); // Show main game if no other overlay is active
}
// This function is called by the "Connect Wallet" button
async function connectUserWallet() {
	document.getElementById("wallet-selector-overlay").classList.remove("hidden");
}

async function connectWith(provider) {
	// Guard: Ensure CONFIG has been populated by the WebSocket identity message
	if (CONFIG.VAULT_ADDRESS === null) {
		showToast("⚠️ Arena configuration not yet synced. Please wait a moment.", "warning");
		return;
	}

	closeWalletSelector();
	showToast(`Connecting to ...`, "info");
	
	try {
		let address = null;
		if (provider === 'nautilus') {
			if (!window.algo) throw new Error("Nautilus not installed");
			const accounts = await window.algo.enable();
			address = accounts[0];
			walletProvider = 'nautilus';
		} else if (provider === 'kibisis') {
			if (!window.kibisis) throw new Error("Kibisis not installed");
			const accounts = await window.kibisis.enable();
			address = accounts[0];
			walletProvider = 'kibisis';
		} else if (provider === 'walletconnect') {
			if (!signClient || !wcModal) throw new Error("WalletConnect not initialized");

			const { uri, approval } = await signClient.connect({
				optionalNamespaces: {
					algorand: {
						methods: ["algo_signTxn", "algo_signMessage"],
						chains: [CONFIG.VOI_CHAIN_ID, CONFIG.ALGO_CHAIN_ID],
						events: ["chainChanged", "accountsChanged"],
					},
					eip155: {
						methods: ["eth_signTransaction", "eth_sendTransaction", "personal_sign"],
						chains: ["eip155:1", "eip155:137"], // ETH & Polygon
						events: ["chainChanged", "accountsChanged"],
					}
				},
			});
			if (uri) {
				wcModal.openModal({ uri });
				const session = await approval();
				wcModal.closeModal();
				
				// Extract address from namespaces: algorand:caip2_chain_id:address
				const account = session.namespaces.algorand.accounts[0];
				address = account.split(":")[2];
				walletProvider = 'walletconnect';
			}
		}

		if (address) {
			const result = window.connectWallet(address);
			if (result.status === "success") {
				updateWalletUI(result.address);
				showToast("Wallet Connected!", "success");
				closeWalletSelector(); // Close selector after successful connection
			}
		}
	} catch (err) {
		console.error("Connection failed", err);
		showToast(err.message, "error");
	}
}

function updateWalletUI(address) {
	userAddress = address;
	const btn = document.getElementById("wallet-btn");
	if (address) {
		btn.innerText = address.substring(0, 8) + "...";
		btn.classList.add("outline");

		// Register wallet with the Live Lobby server
		if (socket && socket.readyState === WebSocket.OPEN) {
			socket.send(JSON.stringify({
				type: "register_wallet",
				payload: { wallet: address }
			}));
		}

		// BRIDGE LOGIC: If on Algorand, ensure the user is "birthed" into Voi
		const state = window.GetGameState();
		if (state.network === "ALGO") {
			checkVoiReadiness(address);
		}

		// If we are in the Setup phase, fetch the player's NFT collection for avatar selection
		if (state.phase === "Setup") {
			fetchUserNFTs(address);
		}
	} else {
		btn.innerText = "Connect Wallet";
		btn.classList.remove("outline");
	}
	syncUI();
	showMainGameContainer(); // Ensure main game is visible after wallet action
}

async function checkVoiReadiness(address) {
	console.log("[BRIDGE] Checking Voi readiness for Algorand wallet...");
	setTransactionStatus("Checking Voi onboarding status...", "info");
	try {
		const response = await fetch(`${CONFIG.API_BASE}/api/bridge/onboard`, {
			method: "POST",
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ wallet: address })
		});
		if (response.ok && response.status !== 204) {
			const data = await response.json();
			let message = `🌉 ${data.message}`;
			if (data.txid) {
				const voiConfig = availableNetworks["Voi Mainnet"];
				if (voiConfig && voiConfig.explorer_url) {
					const explorerLink = `${voiConfig.explorer_url}/tx/${data.txid}`;
					message += `<br><a href="" target="_blank" style="color: var(--neon-green); text-decoration: underline;">View Transaction</a>`;
				} else {
					message += `<br>TxID: ${data.txid.substring(0, 14)}...`;
				}
			}
			showToast(message, "success", 10000);
		}
		if (response.status === 503) { // Handle Service Unavailable specifically
			const errorText = await response.text();
			showToast(`⚠️ <b>ONBOARDING UNAVAILABLE:</b> `, "warning", 10000);
			return;
		}
	} catch (err) {
		console.error("[BRIDGE] Onboarding check failed", err);
	} finally {
		setTransactionStatus(null);
	}
}

async function adminGloatBan(walletToBan = null, hoursToBan = null) {
	const wallet = walletToBan || document.getElementById("admin-ban-wallet").value.trim();
	const hours = hoursToBan || parseInt(document.getElementById("admin-ban-hours").value);
	
	if (!wallet) return;
	const headers = await getAdminHeaders();
	if (!headers) return;

	try {
		const response = await fetch(`${CONFIG.API_BASE}/api/admin/gloat-ban`, {
			method: "POST",
			headers: { ...headers, 'Content-Type': 'application/json' },
			body: JSON.stringify({ wallet, hours: isNaN(hours) ? 48 : hours })
		});
		if (response.ok) {
			showToast(`🚫 Gloat messaging restricted for ${wallet.substring(0,6)}...`, "success");
			fetchLastAdminAction();
		} else {
			const errText = await response.text();
			showToast(`❌ Gloat ban failed: `, "error");
		}
	} catch (err) { showToast("❌ Server connection error", "error"); }
}

function ignoreReporter(wallet) { // This function was orphaned due to the previous error.
	if (!wallet) return;
	ignoredReporters.add(wallet);
	localStorage.setItem("vbabes_ignored_reporters", JSON.stringify(Array.from(ignoredReporters)));
	showToast(`🙈 Ignoring reports from ${wallet.substring(0,6)}...`, "info");
	fetchAdminLogs(); // Re-render to apply filter
}

async function fetchAdminLogs() {
	const headers = await getAdminHeaders();
	if (!headers) return;

	const filter = document.getElementById("admin-log-filter").value.trim();
	const startVal = document.getElementById("admin-log-start").value;
	const endVal = document.getElementById("admin-log-end").value;

	const logDisplay = document.getElementById("admin-logs-display");
	logDisplay.innerHTML = `<div class="chat-msg system">Fetching logs...</div>`;

	try {
		let url = `${CONFIG.API_BASE}/api/admin/logs?filter=${encodeURIComponent(filter)}`;
		if (startVal) {
			url += `&start_date=${encodeURIComponent(startVal + ":00Z")}`;
		}
		if (endVal) {
			url += `&end_date=${encodeURIComponent(endVal + ":00Z")}`;
		}

		const response = await fetch(url, { headers });
		if (!response.ok) throw new Error("Failed to fetch admin logs");
		const data = await response.json();

		logDisplay.innerHTML = "";
		if (data.logs && data.logs.length > 0) {
			data.logs.forEach(logJson => {
				let logEntry;
				try {
					logEntry = JSON.parse(logJson);
				} catch (e) {
					// Handle raw string logs (e.g., from older entries or malformed)
					logDisplay.innerHTML += `<div class="chat-msg system">${logJson.raw || logJson}</div>`;
					return;
				}

				// Extract reporter wallet if possible for ignoring logic
				let reporterWallet = "";
				let avatarUrl = "";
				if (logEntry.action === "REPORT_GLOAT") {
					const rMatch = logEntry.details.match(/Reported by (.*?) for/);
					if (rMatch) reporterWallet = rMatch[1];
					
					const aMatch = logEntry.details.match(/\[Avatar: (.*?)\]/);
					if (aMatch) avatarUrl = aMatch[1];
				}

				// Local Filter for Ignored Reporters
				if (reporterWallet && ignoredReporters.has(reporterWallet)) return;

				const logDiv = document.createElement("div");
				logDiv.className = "chat-msg";
				if (logEntry.action === "REPORT_GLOAT") {
					logDiv.classList.add("report-alert");
				}

				let logText = `[${new Date(logEntry.timestamp).toLocaleString()}] <b>${logEntry.action}</b>: ${logEntry.target} - ${logEntry.details}`;

				if (logEntry.action === "REPORT_GLOAT") {
					const rawGloat = logEntry.details.split('for offensive gloat: "')[1]?.split('" [Avatar:')[0] || "REDACTED";
					logText = `🚨 <b>REPORTED GLOAT</b>: ${logEntry.target} - ""`;
					
					logDiv.innerHTML = ` 
						<div class="flex-row gap-5 mt-5 justify-end">
							<button class="outline admin-action-btn-small text-error" onclick="adminBanWallet('${logEntry.target}', 24)">BAN PLAYER</button>
							<button class="outline admin-action-btn-small text-error" onclick="adminGloatBan('${logEntry.target}', 48)">BAN GLOATS</button>
							${avatarUrl ? `<button class="outline admin-action-btn-small" style="border-color: #ffa657; color: #ffa657;" onclick="adminAvatarBan('', 720)">BAN AVATAR</button>` : ''}
							<button class="outline admin-action-btn-small" style="border-color: #888; color: #888;" onclick="ignoreReporter('')">IGNORE REPORTER</button>
						</div>`;
				} else if (logEntry.action.startsWith("SECURITY")) {
					logDiv.style.backgroundColor = "rgba(255, 166, 87, 0.1)";
					logDiv.style.border = "1px solid #ffa657";
					logDiv.innerHTML = logText;
				} else {
					logDiv.innerHTML = logText;
				}
				logDisplay.appendChild(logDiv);
			});
		} else {
			logDisplay.innerHTML = `<div class="chat-msg system">No logs found matching filter.</div>`;
		}
	} catch (err) {
		console.error("Failed to fetch admin logs:", err);
		logDisplay.innerHTML = `<div class="chat-msg system" style="color: #ff4b4b;">Error fetching logs.</div>`;
	}
}

// --- Admin Authentication Helper ---
async function getAdminHeaders() {
	if (!userAddress) { 
		showToast("Please connect your admin wallet.", "error");
		return null;
	}

	try {
		setTransactionStatus("Requesting admin nonce...", "info");
		// Request Nonce from Server via WebSocket (Anti-Replay)
		const nonce = await new Promise((resolve, reject) => {
			const timeout = setTimeout(() => reject(new Error("Nonce request timed out")), 10000);
			nonceResolver = (n) => { clearTimeout(timeout); resolve(n); };
			socket.send(JSON.stringify({ type: "nonce_request" }));
		});

		const msg = `Virtualbabes Arena Admin Auth:`;
		const msgBytes = new TextEncoder().encode(msg);

		let signature = null;
		setTransactionStatus("Signing admin session...", "info");
		const isEVM = userAddress.startsWith("0x");

		if (isEVM) {
			if (walletProvider === 'walletconnect' && signClient) {
				const sessions = signClient.session.getAll();
				const hexMsg = "0x" + Array.from(msgBytes).map(b => b.toString(16).padStart(2, '0')).join('');
				signature = await signClient.request({
					topic: sessions[0].topic,
					chainId: "eip155:1",
					request: {
						method: "personal_sign",
						params: [hexMsg, userAddress]
					}
				});
			} else {
				throw new Error("EVM Admin Auth requires WalletConnect.");
			}
		} else if (walletProvider === 'nautilus') {
			const result = await window.algo.signMessage(msgBytes, userAddress);
			// Nautilus returns the raw Uint8Array signature
			signature = btoa(String.fromCharCode(...result));
		} else if (walletProvider === 'walletconnect' && signClient) {
			const sessions = signClient.session.getAll();
			const response = await signClient.request({
				topic: sessions[0].topic,
				chainId: (window.GetGameState().network === "VOI" ? CONFIG.VOI_CHAIN_ID : CONFIG.ALGO_CHAIN_ID),
				request: {
					method: "algo_signMessage",
					params: [userAddress, btoa(String.fromCharCode(...msgBytes))]
				}
			});
			signature = response; // WC response is usually the base64 string
		} else {
			throw new Error("Active wallet provider does not support signature authentication.");
		}

		if (!signature) throw new Error("Signature denied.");

		const headers = {
			'X-Admin-Wallet': userAddress,
			'X-Admin-Nonce': nonce,
			'X-Admin-Signature': signature
		};
		
		cachedAdminHeaders = headers; // Cache for background polling
		setTransactionStatus(null);
		return headers;
	} catch (err) {
		showToast(`❌ Admin Auth Failed: ${err.message}`, "error");
		setTransactionStatus(null);
		return null;
	}
}

// --- Admin Action Handlers ---
async function adminRefillVault() {
	const amount = parseFloat(document.getElementById("admin-refill-amt").value);
	if (isNaN(amount)) return;
	const headers = await getAdminHeaders();
	if (!headers) return;

	try {
		const response = await fetch(`${CONFIG.API_BASE}/api/refill-vault`, {
			method: "POST",
			headers: { ...headers, 'Content-Type': 'application/json' },
			body: JSON.stringify({ amount })
		});
		if (response.ok) showToast("⚡ Vault balance updated successfully", "success");
		fetchLastAdminAction();
	} catch (err) { showToast("❌ Refill failed", "error"); }
}

function updateAdminRewardList(rewards) {
	const container = document.getElementById("admin-reward-list");
	container.innerHTML = "";
	Object.entries(rewards || {}).forEach(([id, amt]) => {
		const symbol = getAssetSymbol(id);
		const div = document.createElement("div");
		div.className = "player-item";
		div.style.padding = "5px";
		div.innerHTML = `<span style="font-size: 11px;">: <b> units</b></span>
						 <button class="outline" style="padding: 2px 8px; font-size: 9px; border-color: #ff4b4b; color: #ff4b4b;" onclick="adminRemoveReward()">DEL</button>`;
		container.appendChild(div);
	});
}

async function adminAddReward() {
	const assetId = parseInt(document.getElementById("admin-add-asset").value);
	const amount = parseFloat(document.getElementById("admin-add-amt").value);
	if (!assetId || isNaN(amount)) return;
	const headers = await getAdminHeaders();
	if (!headers) return;

	try {
		const response = await fetch(`${CONFIG.API_BASE}/api/reward/add`, {
			method: "POST",
			headers: { ...headers, 'Content-Type': 'application/json' },
			body: JSON.stringify({ asset_id: assetId, amount: amount })
		});
		if (response.ok) showToast("➕ Reward added to stack", "success");
		fetchLastAdminAction();
	} catch (err) { showToast("❌ Action failed", "error"); }
}

async function adminRemoveReward(assetId) {
	const headers = await getAdminHeaders();
	if (!headers) return;

	try {
		const response = await fetch(`${CONFIG.API_BASE}/api/reward/remove`, {
			method: "POST",
			headers: { ...headers, 'Content-Type': 'application/json' },
			body: JSON.stringify({ asset_id: assetId })
		});
		if (response.ok) showToast("➖ Reward removed", "info");
		fetchLastAdminAction();
	} catch (err) { showToast("❌ Update failed", "error"); }
}

async function adminAddNetwork() {
	const headers = await getAdminHeaders();
	if (!headers) return;

	const newNetwork = {
		network_name: document.getElementById("new-network-name").value.trim(),
		explorer_url: document.getElementById("new-explorer-url").value.trim(),
		indexer_url: document.getElementById("new-indexer-url").value.trim(),
		node_url: document.getElementById("new-node-url").value.trim(),
		faucet_url: document.getElementById("new-faucet-url").value.trim(),
		asset_id: parseInt(document.getElementById("new-asset-id").value) || 0,
		app_id: parseInt(document.getElementById("new-app-id").value) || 0,
		chain_id: document.getElementById("new-chain-id").value.trim(),
		power_divisor: parseFloat(document.getElementById("new-power-divisor").value) || 1000000,
		power_base: parseInt(document.getElementById("new-power-base").value) || 50,
	};

	// Basic validation
	if (!newNetwork.network_name || !newNetwork.node_url || !newNetwork.chain_id) {
		showToast("❌ Network Name, Node URL, and Chain ID are required.", "error");
		return;
	}

	try {
		const response = await fetch(`${CONFIG.API_BASE}/api/admin/network/add`, {
			method: "POST",
			headers: { ...headers, 'Content-Type': 'application/json' },
			body: JSON.stringify(newNetwork)
		});
		if (response.ok) showToast(`➕ Network '${newNetwork.network_name}' added`, "success");
		fetchLastAdminAction();
	} catch (err) { showToast("❌ Failed to add network", "error"); }
}

async function adminBroadcast() {
	const text = document.getElementById("admin-msg-text").value;
	if (!text) return;
	const headers = await getAdminHeaders();
	if (!headers) return;

	try {
		const response = await fetch(`${CONFIG.API_BASE}/api/system-message`, {
			method: "POST",
			headers: { ...headers, 'Content-Type': 'application/json' },
			body: JSON.stringify({ text })
		});
		if (response.ok) {
			showToast("📣 System message broadcasted", "success");
			fetchLastAdminAction();
			document.getElementById("admin-msg-text").value = "";
		}
	} catch (err) { showToast("❌ Broadcast failed", "error"); }
}

async function adminUpdateRules() {
	const req = {
		Open: document.getElementById("rule-open").checked,
		Power_copy: document.getElementById("rule-same").checked,
		Power_up: document.getElementById("rule-plus").checked,
		Elemental_sync: document.getElementById("rule-elemental").checked,
		Fallen_penalty: document.getElementById("rule-fallen").checked,
		Artifact_bonus: document.getElementById("rule-artifact").checked
	};
	const headers = await getAdminHeaders();
	if (!headers) return;

	try {
		const response = await fetch(`${CONFIG.API_BASE}/api/update-rules`, {
			method: "POST",
			headers: { ...headers, 'Content-Type': 'application/json' },
			body: JSON.stringify(req)
		});
		if (response.ok) showToast("⚙️ Global rules synchronized", "success");
		fetchLastAdminAction();
	} catch (err) { showToast("❌ Rules update failed", "error"); }
}

async function adminBanWallet(walletToBan = null, hoursToBan = null) {
	const wallet = walletToBan || document.getElementById("admin-ban-wallet").value.trim();
	const hours = hoursToBan || parseInt(document.getElementById("admin-ban-hours").value);
	if (!wallet) return;
	const headers = await getAdminHeaders();
	if (!headers) return;

	try {
		const response = await fetch(`${CONFIG.API_BASE}/api/ban-player`, {
			method: "POST",
			headers: { ...headers, 'Content-Type': 'application/json' },
			body: JSON.stringify({ wallet, hours: isNaN(hours) ? 24 : hours })
		});
		if (response.ok) {
			showToast(`🚫 Wallet ${wallet.substring(0,6)}... restricted`, "success");
			fetchLastAdminAction();
			if (!walletToBan) { // Only clear input if not called from log
				document.getElementById("admin-ban-wallet").value = "";
				document.getElementById("admin-ban-hours").value = "";
			}
			fetchLeaderboard(); // Refresh to show the lock icon
		} else {
			const errText = await response.text();
			showToast(`❌ Ban failed: `, "error");
		}
	} catch (err) { showToast("❌ Server connection error", "error"); }
}

async function adminAvatarBan(url = null, hours = null) {
	const targetUrl = url || document.getElementById("admin-ban-avatar-url").value.trim();
	if (!targetUrl) return;
	const headers = await getAdminHeaders();
	if (!headers) return;

	try {
		const response = await fetch(`${CONFIG.API_BASE}/api/admin/avatar-ban`, {
			method: "POST",
			headers: { ...headers, 'Content-Type': 'application/json' },
			body: JSON.stringify({ url: targetUrl, hours: isNaN(hours) ? 720 : hours })
		});
		if (response.ok) {
			showToast(`🚫 Avatar asset restricted`, "success");
			fetchLastAdminAction();
			if (!url) {
				document.getElementById("admin-ban-avatar-url").value = "";
			}
		} else {
			const errText = await response.text();
			showToast(`❌ Avatar ban failed: `, "error");
		}
	} catch (err) {
		showToast("❌ Server connection error", "error");
	}
}

function adminBanWalletFromLog(wallet) {
	// Default to 24 hours for a quick ban from logs
	adminBanWallet(wallet, 24);
}

async function adminUpdatePowerScaling() {
	const divisor = parseFloat(document.getElementById("admin-power-divisor").value);
	const base = parseInt(document.getElementById("admin-power-base").value);
	const headers = await getAdminHeaders();
	if (!headers) return;

	try {
		const response = await fetch(`${CONFIG.API_BASE}/api/admin/update-power`, {
			method: "POST",
			headers: { ...headers, 'Content-Type': 'application/json' },
			body: JSON.stringify({ divisor, base })
		});
		if (response.ok) showToast("⚖️ Power scaling adjusted", "success");
		fetchLastAdminAction();
	} catch (err) { showToast("❌ Power update failed", "error"); }
}

async function addXChainWallet() {
	if (!signClient || !wcModal || !userAddress) return;
	showToast("Opening WalletConnect for X-Chain Link...", "info");

	try {
		// Allow connection to any chain supported by the current admin config
		const chainIds = Object.values(availableNetworks).map(n => n.chain_id).filter(id => id);
		const algoChains = chainIds.filter(id => id.startsWith("algorand:"));
		const evmChains = chainIds.filter(id => id.startsWith("eip155:"));
		const solanaChains = chainIds.filter(id => id.startsWith("solana:"));
		
		const { uri, approval } = await signClient.connect({
			optionalNamespaces: {
				algorand: {
					methods: ["algo_signTxn", "algo_signMessage"],
					chains: algoChains.length > 0 ? algoChains : [CONFIG.VOI_CHAIN_ID, CONFIG.ALGO_CHAIN_ID],
					events: ["accountsChanged"]
				},
				eip155: {
					methods: ["eth_sendTransaction", "personal_sign"],
					chains: evmChains.length > 0 ? evmChains : ["eip155:1", "eip155:137"],
					events: ["accountsChanged", "chainChanged"]
				},
				solana: {
					methods: ["solana_signTransaction", "solana_signMessage"],
					chains: solanaChains.length > 0 ? solanaChains : ["solana:5eykt4UsFvXYfy2khQbSsLurFBXY"],
					events: ["accountsChanged"]
				}
			}
		});

			if (uri) {
				wcModal.openModal({ uri });
				const session = await approval();
				// The inner try-catch was for a specific check, not the whole block.
				if (session.namespaces.eip155) { // This check was misplaced
					// Approval logic
				}
				wcModal.closeModal(); // This should be here

			let namespace = "";
			let accountStr = "";

			if (session.namespaces.eip155) {
				namespace = "eip155";
				accountStr = session.namespaces.eip155.accounts[0];
			} else if (session.namespaces.solana) {
				namespace = "solana";
				accountStr = session.namespaces.solana.accounts[0];
			} else if (session.namespaces.algorand) {
				namespace = "algorand";
				accountStr = session.namespaces.algorand.accounts[0];
			}

			if (!accountStr) throw new Error("No account returned from session");

			const [ns, chainId, address] = accountStr.split(":");
			const newWalletInfo = { // Renamed to avoid redeclaration
				address: address,
				chain: ns === 'eip155' ? (chainId === '1' ? 'ETH' : (chainId === '137' ? 'POLY' : 'EVM')) : 
					   (ns === 'solana' ? 'SOL' : (chainId === 'mainnet-v1.0' ? 'ALGO' : 'VOI')),
				id: `Linked-${Date.now()}`
			};

			// 1. Request Nonce for Link Verification
			showToast("Verifying ownership... please sign the request.", "info");
			const nonce = await new Promise((resolve, reject) => {
				const timeout = setTimeout(() => reject(new Error("Nonce request timed out")), 10000);
				nonceResolver = (n) => { clearTimeout(timeout); resolve(n); };
				socket.send(JSON.stringify({ type: "nonce_request" }));
			});

			// 2. Sign Nonce with the newly linked wallet
			let signature = null;
			if (namespace === "eip155") {
				const hexMsg = "0x" + Array.from(new TextEncoder().encode(nonce)).map(b => b.toString(16).padStart(2, '0')).join('');
				signature = await signClient.request({
					topic: session.topic,
					chainId: `eip155:`,
					request: {
						method: "personal_sign",
						params: [hexMsg, address]
					}
				});
			} else if (namespace === "solana") {
				const result = await signClient.request({
					topic: session.topic,
					chainId: `solana:`,
					request: {
						method: "solana_signMessage",
						params: { message: btoa(nonce), pubkey: address }
					}
				});
				signature = result.signature; // WC Solana return pattern
			} else if (namespace === "algorand") {
				const result = await signClient.request({
					topic: session.topic,
					chainId: `algorand:`,
					request: {
						method: "algo_signMessage",
						params: [address, btoa(nonce)]
					}
				});
				signature = result;
			}

			if (!signature) throw new Error("Signature denied or failed");

			// 3. Send Request to Server for persistent linking
			socket.send(JSON.stringify({
				type: "link_wallet_request",
				payload: {
					primary_avm_wallet: userAddress,
					linked_address: address,
					linked_chain: newWalletInfo.chain,
					signature: signature,
					nonce: nonce
				}
			}));

			// Prevent duplicates
			if (!linkedWallets.find(w => w.address.toLowerCase() === newWalletInfo.address.toLowerCase())) {
				linkedWallets.push(newWalletInfo);
				localStorage.setItem("vbabes_linked_wallets", JSON.stringify(linkedWallets));
				showToast(`🔗 Requested link for ${newWalletInfo.chain} wallet...`, "info");
				refreshInventory();
			}
		}

	} catch (err) {
		console.error("X-Chain link failed", err);
	}
}

function updateLinkedWalletsUI() {
	const container = document.getElementById("linked-wallets-list");
	if (!container) return;
	container.innerHTML = "";
	linkedWallets.forEach((w, idx) => {
		const div = document.createElement("div");
		div.className = "player-item";
		div.style.padding = "2px 5px";
		div.style.fontSize = "10px";
		div.innerHTML = `<span>[${w.chain}] ${w.address.substring(0,6)}...</span>
						 <button class="outline" style="padding: 0 4px; color: #ff4b4b; border-color: #ff4b4b;" onclick="removeLinkedWallet()">×</button>`;
		container.appendChild(div);
	});
}

function removeLinkedWallet(index) {
	linkedWallets.splice(index, 1);
	localStorage.setItem("vbabes_linked_wallets", JSON.stringify(linkedWallets));
	refreshInventory();
}

function openPayoutSettings() {
	document.getElementById("payout-settings-overlay").classList.remove("hidden");
	document.getElementById("payout-address-input").value = payoutAddress || "";
}

function savePayoutAddress() {
	const addr = document.getElementById("payout-address-input").value.trim();
	if (addr && addr.length === 58) {
		payoutAddress = addr;
		localStorage.setItem("vbabes_payout_address", addr);
		showToast("✅ Voi payout address updated", "success");
		updatePayoutUI();
		hideAllOverlays();
	} else {
		showToast("❌ Invalid Voi Address", "error");
	}
}

function updatePayoutUI() {
	const display = document.getElementById("payout-address-display");
	if (display) {
		display.innerText = payoutAddress ? (payoutAddress.substring(0, 6) + "..." + payoutAddress.substring(54)) : "Default Wallet";
	}
}

function updateAdminNetworkUI() {
	const select = document.getElementById("admin-network-select");
	if (!select) return;

	const currentSelection = select.value;
	select.innerHTML = "";
	
	Object.keys(availableNetworks).forEach(name => {
		const opt = document.createElement("option");
		opt.value = name;
		opt.innerText = name;
		if (name === adminFocusNetwork && !currentSelection) opt.selected = true;
		else if (name === currentSelection) opt.selected = true;
		select.appendChild(opt);
	});

	onAdminNetworkSelectChange();
}

function onAdminNetworkSelectChange() {
	const name = document.getElementById("admin-network-select").value;
	const config = availableNetworks[name];
	const details = document.getElementById("admin-network-details");
	if (!details || !config) return;

	details.innerHTML = `
		<div><b>Node:</b> ${config.node_url}</div>
		<div><b>Asset ID:</b> ${config.asset_id}</div>
		<div><b>App ID:</b> ${config.app_id}</div>
		<div style="color: ${name === adminFocusNetwork ? 'var(--neon-green)' : 'inherit'}"><b>Status:</b> ${name === adminFocusNetwork ? 'ACTIVE' : 'Standby'}</div>
	`;

	if (config) {
		document.getElementById("admin-power-divisor").value = config.power_divisor || 1000000;
		document.getElementById("power-divisor-val").innerText = (config.power_divisor || 1000000).toFixed(1);
		document.getElementById("admin-power-base").value = config.power_base || 50;
		document.getElementById("power-base-val").innerText = config.power_base || 50;
	}
}

async function adminSetActiveNetwork() {
	const networkName = document.getElementById("admin-network-select").value;
	const headers = await getAdminHeaders();
	if (!headers) return;

	try { // Renamed endpoint for clarity
		const response = await fetch(`${CONFIG.API_BASE}/api/admin/set-admin-focus-network`, {
			method: "POST",
			headers: { ...headers, 'Content-Type': 'application/json' },
			body: JSON.stringify({ network_name: networkName })
		}); // Updated toast message
		if (response.ok) showToast(`🌐 Admin focus switched to `, "success");
		fetchLastAdminAction();
	} catch (err) { showToast("❌ Admin focus switch failed", "error"); }
}
async function adminToggleMaintenance(active) {
	const minsInput = document.getElementById("admin-maint-mins");
	const minutes = parseInt(minsInput.value) || 0;
	const headers = await getAdminHeaders();
	if (!headers) return;

	try {
		const response = await fetch(`${CONFIG.API_BASE}/api/maintenance-mode`, {
			method: "POST",
			headers: { ...headers, 'Content-Type': 'application/json' },
			body: JSON.stringify({ active, minutes })
		});
		if (response.ok) {
			showToast(`🛠️ Maintenance Mode ${active ? 'Activated' : 'Deactivated'}`, "info");
			fetchLastAdminAction();
			if (!active) minsInput.value = "";
		} else {
			const errText = await response.text();
			showToast(`❌ Action failed: `, "error");
		}
	} catch (err) { showToast("❌ Server connection error", "error"); }
}

async function adminToggleDevMode() {
	const enabled = document.getElementById("dev-mode-toggle").checked;
	// Add a safety check when enabling
	if (enabled && !confirm("⚠️ DEV MODE: This will force a 100% win rate against the bot for reward testing. Enable?")) {
		document.getElementById("dev-mode-toggle").checked = false;
		return;
	}
	window.SetTestingMode(enabled);
	showToast(`🛠️ Dev Mode ${enabled ? 'Enabled' : 'Disabled'}`, enabled ? "success" : "info");
}

async function adminResetStats() {
	const wallet = document.getElementById("admin-ban-wallet").value.trim();
	if (!wallet) return;
	if (!confirm(`⚠️ CRITICAL: You are about to PERMANENTLY WIPE all stats for wallet: . This cannot be undone. Proceed?`)) return;

	const headers = await getAdminHeaders();
	if (!headers) return;

	try {
		const response = await fetch(`${CONFIG.API_BASE}/api/reset-stats`, {
			method: "POST",
			headers: { ...headers, 'Content-Type': 'application/json' },
			body: JSON.stringify({ wallet })
		});
		if (response.ok) {
			showToast(`✨ Stats wiped for ${wallet.substring(0,6)}...`, "success");
			fetchLeaderboard();
			fetchLastAdminAction();
		} else {
			const errText = await response.text();
			showToast(`❌ Reset failed: `, "error");
		}
	} catch (err) { showToast("❌ Server connection error", "error"); }
}

async function adminSimulateTournament() {
	const size = parseInt(document.getElementById("admin-sim-size").value);
	const isBuyIn = document.getElementById("admin-sim-buyin").checked;
	if (isNaN(size) || (size !== 8 && size !== 16)) {
		showToast("❌ Invalid tournament size (must be 8 or 16)", "error");
		return;
	}
	if (!confirm(`Are you sure you want to simulate a -player tournament? This will overwrite the current tournament state.`)) {
		return;
	}

	const headers = await getAdminHeaders();
	if (!headers) return;

	try {
		const response = await fetch(`${CONFIG.API_BASE}/api/admin/simulate-tournament`, {
			method: "POST",
			headers: { ...headers, 'Content-Type': 'application/json' },
			body: JSON.stringify({ size, is_buy_in: isBuyIn })
		});
		if (response.ok) showToast(`🏆 Simulating -player tournament...`, "success");
		fetchLastAdminAction();
	} catch (err) {
		showToast("❌ Simulation failed", "error");
	}
}

let adminLogTicker = null;
function startAdminLogPolling() {
	if (adminLogTicker) return;
	adminLogTicker = setInterval(fetchLastAdminAction, 15000); // Check every 15s for status bar
}

async function fetchLastAdminAction() { 
	if (!lastAdminKey && !cachedAdminHeaders) { 
		document.getElementById("admin-last-action").innerText = "Awaiting first action..."; 
		return; 
	} 
	const headers = lastAdminKey ? { 'X-Admin-Key': lastAdminKey } : cachedAdminHeaders;
	try {
		const response = await fetch(`${CONFIG.API_BASE}/api/admin/logs`, {
			headers: headers
		});
		if (response.ok) {
			const entry = await response.json();
			if (entry.timestamp) {
				const time = entry.timestamp.split('T')[1].substring(0, 8);
				document.getElementById("admin-last-action").innerText = 
					`[] ${entry.action}: ${entry.target}`;
			} else {
				document.getElementById("admin-last-action").innerText = entry.details || "No logs found";
			}
		}
	} catch (err) {}
}

async function disconnectUserWallet() {
	console.log("[WALLET] Disconnecting...");
	try {
		if (walletProvider === 'walletconnect' && signClient) {
			const sessions = signClient.session.getAll();
			if (sessions.length > 0) {
				await signClient.disconnect({
					topic: sessions[0].topic,
					reason: { code: 6000, message: "User disconnected" }
				});
			}
		}
		walletProvider = null;
		
		window.disconnectWallet(); // Reset Go Engine
		isVerified = false;
		userAddress = null; // Clear user address
		updateWalletUI(null);
	} catch (err) {
		console.error("Disconnect failed", err);
	}
}

window.highlightStartButton = (isReady) => {
	const btn = document.getElementById("start-btn");
	if (isReady) {
		btn.disabled = false;
		btn.style.boxShadow = "0 0 30px #3fb950";
		btn.innerText = "BATTLE READY - CLICK TO START!";
	} else {
		btn.disabled = true;
		btn.style.boxShadow = "none";
		btn.innerText = "Start Battle (Waiting for Ready)";
	}
};

// WebSocket Logic
function initWebSocket() {
	const protocol = window.location.protocol === "https:" ? "wss://" : "ws://";
	console.log(`[WS] Connecting to ${CONFIG.BACKEND_URL}/ws ...`);
	socket = new WebSocket(`${CONFIG.BACKEND_URL}/ws`);

	socket.onopen = () => {
		console.log("[WS] Connected to Live Lobby");

		// WATCHDOG: Start 5s timer for identity sync validation.
		// If identity is not received, attempt reconnection.
		if (identitySyncTimeout) clearTimeout(identitySyncTimeout);
		identitySyncTimeout = setTimeout(() => {
			if (!myClientId) {
				if (reconnectAttempts < 3) {
					reconnectAttempts++;
					console.warn(`[WS] Identity sync timeout reached. Attempting reconnect /3...`);
					showToast(`⚠️ Sync failed. Retrying connection (/3)...`, "warning", 3000);
					socket.close(); // Force close to trigger onclose and re-init
				} else {
					console.error("[WS] Identity sync timeout reached after multiple attempts.");
					showToast("⚠️ <b>SYNC FAILURE:</b> Arena configuration not received after multiple attempts. Faucet payouts and tournament registrations may be unavailable. Please refresh.", "error", 0);
				}
			}
		}, 5000);
	};

	socket.onmessage = (event) => {
		const msg = JSON.parse(event.data);
		handleServerMessage(msg);
	};

	socket.onclose = () => {
		console.warn("[WS] Disconnected. Retrying...");
		if (identitySyncTimeout) clearTimeout(identitySyncTimeout);
		// Only attempt immediate reconnect if not due to identity sync timeout already handling it
		if (identitySyncTimeout && reconnectAttempts < 3) {
			setTimeout(initWebSocket, 3000);
		}
	};
}

function handleServerMessage(msg) {
	switch(msg.type) {
		case "pong":
			if (lastPingTime) {
				currentLatency = Date.now() - lastPingTime;
				lastPingTime = null;
				if (window.SyncLatency) window.SyncLatency(currentLatency);
				syncUI("meta");
			}
			break;
		case "matchmaking_status":
			handleMatchmakingUpdate(msg.payload);
			break;
		case "tournament_update":
			console.log("[WS] Tournament update received:", msg.payload);
			if (window.SyncTournament) window.SyncTournament(msg.payload);
			handleTournamentUI(msg.payload);
			syncUI("meta");
			break;
		case "tournament_round_transition":
			console.log("[WS] Tournament round transition received:", msg.payload.round);
			// Trigger frontend animation here
			showTournamentTransition(msg.payload.round);
			break;
		case "nonce_response":
			if (nonceResolver) {
				nonceResolver(msg.payload.nonce);
				nonceResolver = null;
			}
			break;
		case "identity":
			myClientId = msg.to_id;
			// Clear watchdog timer on successful sync
			if (identitySyncTimeout) {
				clearTimeout(identitySyncTimeout);
				identitySyncTimeout = null;
				reconnectAttempts = 0; // Reset attempts on successful sync
			}
			// TACTICAL SYNC: Update local CONFIG with authoritative server values
			if (msg.payload) {
				if (msg.payload.vault) CONFIG.VAULT_ADDRESS = msg.payload.vault;
				if (msg.payload.vbv) CONFIG.VBV_ASSET_ID = msg.payload.vbv;
				if (msg.payload.avoi) CONFIG.AVOI_ASSET_ID = msg.payload.avoi;
				console.log("[CONFIG] Authoritative environment synced from server.");
			}
			break;
		case "sudden_death_start":
			console.log("[WS] Sudden Death event received:", msg.payload);
			// Preserve current player index as ResetGame() in WASM clears it
			const savedPlayerIndex = myPlayerIndex;

			// 1. Reset engine state (clears board, scores, and current decks)
			window.ResetGame();

			// 2. Restore local identity and redistribute decks
			if (window.SetLocalPlayerIndex) window.SetLocalPlayerIndex(savedPlayerIndex);
			if (window.SyncOpponentDeck) {
				window.SyncOpponentDeck(0, msg.payload.p1_deck);
				window.SyncOpponentDeck(1, msg.payload.p2_deck);
			}

			// 3. Re-trigger the active phase in multiplayer mode
			window.StartMatch(true); 
			showToast(msg.payload.text, "warning", 10000);
			syncUI();
			break;
		case "link_wallet_response":
			if (msg.payload.status === "success") {
				showToast(`✅ ${msg.payload.message}`, "success");
			} else {
				showToast(`❌ ${msg.payload.message}`, "error");
				// Remove unverified wallet from local storage and UI
				if (msg.payload.address) {
					linkedWallets = linkedWallets.filter(w => w.address !== msg.payload.address);
					localStorage.setItem("vbabes_linked_wallets", JSON.stringify(linkedWallets));
					updateLinkedWalletsUI();
					refreshInventory();
				}
			}
			break;
		case "lobby_update":
			// Update the player list from the nested 'players' array
			lastLobbyPlayers = msg.payload.players;
			updatePlayerList(msg.payload.players);
			updateMarketTicker(msg.payload.players);

			// TACTICAL SYNC: If server altered our profile (Moderation), update local engine
			const me = msg.payload.players.find(p => p.id === myClientId);
			if (me) {
				// Fully synchronize player metadata into WASM to ensure real state in syncUI
				if (window.SyncFullProfile) window.SyncFullProfile(me);
				if (me.avatar_url && window.SetAvatar) {
					window.SetAvatar(me.avatar_url, me.gloat, me.avatar_notice);
				}
			}
			
			// Sync maintenance status immediately for late joiners
			handleMaintenanceUI(msg.payload.maintenance_active, msg.payload.maintenance_time);

			// Sync Tournament UI
			if (window.SyncTournament) window.SyncTournament(msg.payload.tournament);
			handleTournamentUI(msg.payload.tournament);

			// Sync economy state for late joiners
			if (msg.payload.faucet_balance !== undefined) window.SyncVaultBalance(msg.payload.faucet_balance);
			if (msg.payload.rewards !== undefined) window.SyncRewards(msg.payload.rewards);
			
			// Sync Network Config for Admin
			if (msg.payload.available_networks) {
				availableNetworks = msg.payload.available_networks;
				globalClubs = msg.payload.clubs || {};
				adminFocusNetwork = msg.payload.admin_focus_network; // Renamed
				updateAdminNetworkUI();
			}
			updateActiveRumors(msg.payload.rumors); // Sync all active rumors from lobby update

			if (msg.payload.season_end) {
				seasonEnd = new Date(msg.payload.season_end);
				document.getElementById("season-num-display").innerText = msg.payload.season_number;
				document.getElementById("season-countdown-widget").classList.remove("hidden");
				startSeasonTimer();
			}
			syncUI("all");
			break;
		case "portfolio_update":
			if (window.SyncPortfolio) window.SyncPortfolio(msg.payload);
			syncUI("economy");
			renderRumorBoard(); // Ensure rumor board is updated after any state change
			break;
		case "heist_result":
			handleHeistResult(msg.payload); // Call the standalone function
			break;
		case "challenge": // Corrected structure for the challenge case
			const action = msg.payload.action;
			if (action === "invite") {
				showChallengeNotification(msg.from_id);
			} else if (action === "accept") {
				// Challenger side: Receive acceptor's deck and send own deck back
				console.log("[MATCH] Challenge accepted. Syncing decks...");
				currentOpponentId = msg.from_id;
				myPlayerIndex = 0; // Challenger is P1
				if (window.SetLocalPlayerIndex) window.SetLocalPlayerIndex(0);
				if (window.SyncOpponentProfile) window.SyncOpponentProfile(1, msg.payload.avatar || "", msg.payload.gloat || "");
				if (window.SyncOpponentWanted) window.SyncOpponentWanted(1, msg.payload.wanted_level || 0);
				window.SyncOpponentDeck(1, msg.payload.deck);
				sendMatchSync(msg.from_id);
				window.StartMatch(true);
				syncUI("combat");
			} else if (action === "decline") {
				alert(`Challenge declined by ${msg.from_id}.`);
			} else if (action === "sync_back") {
				// Acceptor side: Receive challenger's deck and start
				currentOpponentId = msg.from_id;
				myPlayerIndex = 1; // Acceptor is P2
				if (window.SetLocalPlayerIndex) window.SetLocalPlayerIndex(1);
				if (window.SyncOpponentProfile) window.SyncOpponentProfile(0, msg.payload.avatar || "", msg.payload.gloat || "");
				if (window.SyncOpponentWanted) window.SyncOpponentWanted(0, msg.payload.wanted_level || 0);
				window.SyncOpponentDeck(0, msg.payload.deck);
				window.StartMatch(true);
				syncUI("combat");
			}
			break;
		case "match_start":
			console.log("[WS] Synchronizing match state...", msg.payload);
			spectatorMatchState = msg.payload;
			// Instead of immediate sync, show the Preview Pop-up
			showMatchPreview(msg.payload);
			break;
		case "move":
			console.log(`[WS] Move received from ${msg.from_id} at grid ${msg.payload.grid_index}`);
			
			if (msg.from_id !== myClientId) {
				let success = false;
				if (spectatorMatchState) {
					// We are a spectator: Determine player index from match state
					const pIdx = (msg.from_id === spectatorMatchState.p1_id) ? 0 : 1;
					success = window.SyncMove(msg.payload.grid_index, msg.payload.card_id, pIdx);
				} else {
					// We are a player: Standard turn-based placement
					success = window.PlaceCard(msg.payload.grid_index, msg.payload.card_id);
				}
				if (!success) console.warn("[WS] Move sync failed.");
				syncUI();
			}
			break;
		case "chat":
			renderChatMessage(msg.from_id, msg.payload.text);
			
			// Handle automatic match invalidation on opponent disconnect
			if (msg.from_id === "SERVER" && msg.payload.text.includes("Match invalidated")) {
				window.ResetGame();
				syncUI();
				showToast("⚠️ Match terminated: Opponent left.", "error");
			}
			break;
		case "vault_update":
			console.log("[WS] Vault balance update received:", msg.payload.balance);
			window.SyncVaultBalance(msg.payload.balance);
			syncUI();
			break;
		case "rules_update":
			console.log("[WS] Global rules update received:", msg.payload);
			window.SyncRules(msg.payload);
			showToast("⚙️ Global Game Rules Updated by Admin", "info");
			syncUI();
			break;
		case "rewards_update":
			console.log("[WS] Reward stack update received:", msg.payload);
			window.SyncRewards(msg.payload);
			syncUI();
			break;
		case "match_start":
			console.log("[WS] Synchronizing match state...", msg.payload);
			spectatorMatchState = msg.payload;
			// Instead of immediate sync, show the Preview Pop-up
			showMatchPreview(msg.payload);
			break;
		case "move":
			console.log(`[WS] Move received from ${msg.from_id} at grid ${msg.payload.grid_index}`);
			
			if (msg.from_id !== myClientId) {
				let success = false;
				if (spectatorMatchState) {
					// We are a spectator: Determine player index from match state
					const pIdx = (msg.from_id === spectatorMatchState.p1_id) ? 0 : 1;
					success = window.SyncMove(msg.payload.grid_index, msg.payload.card_id, pIdx);
				} else {
					// We are a player: Standard turn-based placement
					success = window.PlaceCard(msg.payload.grid_index, msg.payload.card_id);
				}
				if (!success) console.warn("[WS] Move sync failed.");
				syncUI("combat");
			}
			break;
		case "chat":
			renderChatMessage(msg.from_id, msg.payload.text);
			
			// Handle automatic match invalidation on opponent disconnect
			if (msg.from_id === "SERVER" && msg.payload.text.includes("Match invalidated")) {
				window.ResetGame();
				syncUI("combat");
				showToast("⚠️ Match terminated: Opponent left.", "error");
			}
			break;
		case "vault_update":
			console.log("[WS] Vault balance update received:", msg.payload.balance);
			window.SyncVaultBalance(msg.payload.balance);
			syncUI("economy");
			break;
		case "rules_update":
			console.log("[WS] Global rules update received:", msg.payload);
			window.SyncRules(msg.payload);
			showToast("⚙️ Global Game Rules Updated by Admin", "info");
			syncUI("combat");
			break;
		case "rewards_update":
			console.log("[WS] Reward stack update received:", msg.payload);
			window.SyncRewards(msg.payload);
			syncUI("economy");
			break;
		case "maintenance_update":
			console.log("[WS] Maintenance update received:", msg.payload);
			handleMaintenanceUI(msg.payload.active, msg.payload.timestamp);
			break;
		case "admin_notification":
			showToast(msg.payload.text, "warning", 8000);
			// Auto-refresh logs if the admin suite is currently open
			const adminPanel = document.getElementById("admin-control-panel");
			if (adminPanel && !adminPanel.classList.contains("hidden")) {
				fetchAdminLogs();
			}
			break;
		case "kidnap_success":
			showToast("Kidnap successful! Card held hostage.", "success", 5000);
			break;
		case "ransom_demand":
			showKidnapOverlay(msg.payload);
			break;
		case "ransom_paid":
			showToast("Ransom paid. Card released.", "success", 5000);
			hideAllOverlays();
			break;
		case "insurance_recovery":
			showToast("Insurance recovery: Hostage card released.", "info", 5000);
			break;
		case "rumor_update":
			// The server now sends the full rumor object in the payload
			if (msg.payload && msg.payload.rumor) {
				updateActiveRumors(msg.payload.rumor);
			}
			break;
	}
}

// --- Matchmaking Logic ---

function toggleMatchmakingQueue() {
	if (!userAddress) { showToast("Connect wallet first", "error"); return; }
	const state = window.GetGameState();
	if (state.deck.length < 5) { showToast("Deck must have 5 cards", "error"); return; }

	if (!inMatchmakingQueue) {
		socket.send(JSON.stringify({
			type: "join_queue",
			payload: { 
				deck: state.deck.map(c => c.id),
				deck_rating: state.deck_rating
			}
		}));
		const btn = document.getElementById("btn-matchmaking");
		btn.disabled = true;
		btn.innerText = "Joining Queue...";
	} else {
		socket.send(JSON.stringify({ type: "leave_queue" }));
		const btn = document.getElementById("btn-matchmaking");
		btn.disabled = true;
		btn.innerText = "Leaving Queue...";
	}
}

function handleMatchmakingUpdate(data) {
	const btn = document.getElementById("btn-matchmaking");
	const status = document.getElementById("queue-status");

	if (data.status === "queued") {
		inMatchmakingQueue = true;
		btn.innerText = "Leave Queue";
		btn.style.background = "var(--neon-purple)";
		status.innerHTML = `<span class="status-active">SEARCHING FOR OPPONENT...</span>`;
		showToast("🛰️ Entered global matchmaking pool.", "info");
		btn.disabled = false; // Re-enable after status update
	} else if (data.status === "idle") {
		inMatchmakingQueue = false;
		btn.innerText = "Join Matchmaking Pool";
		btn.style.background = "";
		status.innerText = "Ready for automatic pairing?";
		btn.disabled = false; // Re-enable after status update
		showToast("🛰️ Left matchmaking pool.", "info");
	} else if (data.status === "match_found") {
		inMatchmakingQueue = false;
		btn.innerText = "Join Matchmaking Pool";
		status.innerText = "Ready for automatic pairing?";
		showToast(`⚔️ MATCH FOUND! Battle vs ${data.opponent.substring(0,8)}...`, "success");
		window.SetPhase("Active"); // Optional: logic to transition visual state
		btn.disabled = false; // Re-enable after status update
	}
}

function sendPing() {
	if (!socket || socket.readyState !== WebSocket.OPEN) return;
	lastPingTime = Date.now();
	socket.send(JSON.stringify({ type: "ping" }));
}

function updatePlayerList(players) {
	const list = document.getElementById("active-players");
	list.innerHTML = "";
	
	// Check if current user is banned
	const me = players.find(p => p.id === myClientId);
	handleLocalBanUI(me ? me.ban_expires : null);
	const iAmBanned = me && me.ban_expires && new Date(me.ban_expires) > Date.now();

	players.forEach(p => {
		const li = document.createElement("li");
		li.className = "player-item";
		const isMe = p.id === myClientId;
		
		const targetBanned = p.ban_expires && new Date(p.ban_expires) > Date.now();
		const isDisabled = !isMe && (iAmBanned || targetBanned);
		const adminBadge = p.is_admin ? `<span style="color: var(--neon-cyan); font-weight: bold; font-size: 0.8em; margin-left: 5px;">[ADMIN]</span>` : '';
		const btnTitle = targetBanned ? "Player Banned" : (iAmBanned ? "You are Banned" : "Challenge");

		li.innerHTML = `<span>${p.id} ${isMe ? '(You)' : ''} </span>
						<div style="display: flex; gap: 5px;">
							${!isMe ? `<button class="outline" style="padding: 5px 10px; font-size: 10px;" ${isDisabled ? 'disabled' : ''} title="" onclick="sendChallenge('${p.id}')">Challenge</button>` : ''}
							${!isMe ? `<button class="outline" style="padding: 5px 10px; font-size: 10px; border-color: var(--neon-purple); color: var(--neon-purple);" onclick="sendSpectate('${p.id}')">Watch</button>` : ''}
						</div>`;
		list.appendChild(li);
	});
}

function updateMarketTicker(players) {
	const spacing = 60;
	let tickerContainer = document.getElementById("market-ticker");
	if (!tickerContainer) {
		tickerContainer = document.createElement("div");
		tickerContainer.id = "market-ticker";
		tickerContainer.className = "market-ticker-container";
		// Performance Refactor: Use Canvas instead of DOM string manipulation
		tickerContainer.innerHTML = `
			<div class="ticker-label">LIVE MARKET:</div>
			<canvas id="market-ticker-canvas" style="flex: 1; height: 30px; cursor: default;"></canvas>
		`;
		document.body.prepend(tickerContainer);

		const canvas = document.getElementById("market-ticker-canvas");
		const resize = () => {
			const dpr = window.devicePixelRatio || 1;
			const rect = canvas.getBoundingClientRect();
			canvas.width = rect.width * dpr;
			canvas.height = 30 * dpr;
			const ctx = canvas.getContext('2d');
			ctx.scale(dpr, dpr);
		};
		window.addEventListener('resize', resize);
		resize();
	}

	// Sort by Wins -> Reputation to find "Top Performers"
	const topPerformers = [...players]
		.sort((a, b) => (b.wins - a.wins) || (b.reputation - a.reputation))
		.slice(0, 5);

	const newItems = [];
	
	// Add Global Market Token
	newItems.push({
		symbol: "MKT TOKEN",
		val: "0.80 ",
		trend: "▲",
		color: "#3fb950" // Neon Green
	});

	topPerformers.forEach(p => {
		const basePrice = (p.wins * 10) + (p.reputation / 2) + 100;
		const volatility = (p.id.charCodeAt(p.id.length - 1) % 5);
		const finalPrice = basePrice + volatility;
		
		newItems.push({
			symbol: getCachedEnvoiName(p.wallet),
			badge: (p.achievements && p.achievements.length > 0) ? "🏆" : "",
			val: finalPrice.toFixed(2),
			trend: (p.wins > 0) ? "▲" : "─",
			color: (p.wins > 0) ? "#3fb950" : "#888"
		});
	});

	// PERFORMANCE OPTIMIZATION: Pre-calculate widths and measure text only when data changes
	const canvas = document.getElementById("market-ticker-canvas");
	const ctx = canvas ? canvas.getContext('2d') : null;
	if (ctx) {
		ctx.font = "bold 12px 'Rajdhani', sans-serif";
		tickerItems = newItems.map(item => {
			const str = `${item.symbol}${item.badge ? ' ' + item.badge : ''} ${item.val} ${item.trend}`;
			item.width = ctx.measureText(str).width + spacing;
			return item;
		});
	} else {
		tickerItems = newItems;
	}

	if (!tickerAnimId) {
		startTickerAnimation();
	}
}

function startTickerAnimation() {
	const canvas = document.getElementById("market-ticker-canvas");
	if (!canvas) return;
	const ctx = canvas.getContext('2d');

	const animate = () => {
		if (tickerItems.length === 0) {
			tickerAnimId = requestAnimationFrame(animate);
			return;
		}

		const width = canvas.width / (window.devicePixelRatio || 1);
		const height = 30;
		ctx.clearRect(0, 0, width, height);

		ctx.font = "bold 12px 'Rajdhani', sans-serif";
		ctx.textBaseline = "middle";

		// Optimized total width calculation using cached widths
		const totalContentWidth = tickerItems.reduce((sum, item) => sum + (item.width || 0), 0);
		if (totalContentWidth <= 0) {
			tickerAnimId = requestAnimationFrame(animate);
			return;
		}

		tickerOffset += 0.8; // Scrolling speed
		if (tickerOffset >= totalContentWidth) tickerOffset = 0;

		let x = -tickerOffset;
		while (x < width) {
			for (let i = 0; i < tickerItems.length; i++) {
				const item = tickerItems[i];
				const itemWidth = item.width || 100; // Fallback

				if (x + itemWidth > 0 && x < width) {
					let curX = x;
					ctx.fillStyle = "#00f2fe"; // Neon Cyan
					ctx.fillText(item.symbol, curX, height / 2);
					curX += ctx.measureText(item.symbol).width;

					if (item.badge) {
						ctx.fillStyle = "#ffd700"; // Gold
						ctx.fillText(" " + item.badge, curX, height / 2);
						curX += ctx.measureText(" " + item.badge).width;
					}

					ctx.fillStyle = "#ffffff";
					ctx.fillText(" " + item.val, curX, height / 2);
					curX += ctx.measureText(" " + item.val).width;

					ctx.fillStyle = item.color;
					ctx.fillText(" " + item.trend, curX, height / 2);
				}
				x += itemWidth;
			}
			if (totalContentWidth <= 0) break;
		}

		tickerAnimId = requestAnimationFrame(animate);
	};
	tickerAnimId = requestAnimationFrame(animate);
}

let banTicker = null;
function handleLocalBanUI(banExpires) {
	const container = document.getElementById("local-ban-cooldown");
	const fill = document.getElementById("ban-progress-fill");
	const timer = document.getElementById("ban-countdown-timer");
	
	if (banTicker) clearInterval(banTicker);

	if (!banExpires || new Date(banExpires) <= Date.now()) {
		container.classList.add("hidden");
		return;
	}

	container.classList.remove("hidden");
	const expiry = new Date(banExpires).getTime();
	const totalDuration = 24 * 60 * 60 * 1000; // 24 Hours

	const tick = () => {
		const now = Date.now();
		const remaining = expiry - now;

		if (remaining <= 0) {
			container.classList.add("hidden");
			clearInterval(banTicker);
			return;
		}

		const hours = Math.floor(remaining / (1000 * 60 * 60));
		const minutes = Math.floor((remaining % (1000 * 60 * 60)) / (1000 * 60));
		const seconds = Math.floor((remaining % (1000 * 60)) / 1000);
		timer.innerText = `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;

		const progress = ((totalDuration - remaining) / totalDuration) * 100;
		fill.style.width = `${Math.max(0, Math.min(100, progress))}%`;
	};

	tick();
	banTicker = setInterval(tick, 1000);
}

function startSeasonTimer() {
	if (seasonTimerInterval) clearInterval(seasonTimerInterval);
	const timerEl = document.getElementById("season-timer");
	
	const update = () => {
		if (!seasonEnd) return;
		const now = new Date();
		const diff = seasonEnd - now;
		
		if (diff <= 0) {
			timerEl.innerText = "ROLLOVER IMMINENT";
			timerEl.style.color = "var(--neon-green)";
			return;
		}
		
		const days = Math.floor(diff / (1000 * 60 * 60 * 24));
		const hours = Math.floor((diff / (1000 * 60 * 60)) % 24);
		const mins = Math.floor((diff / 1000 / 60) % 60);
		
		timerEl.innerText = `d h m`;
	};
	
	update();
	seasonTimerInterval = setInterval(update, 60000); // Check once per minute
}

let maintenanceTicker = null;
function handleMaintenanceUI(active, targetTimestamp) {
	const bar = document.getElementById("maintenance-bar");
	const timerDisplay = document.getElementById("maintenance-timer");

	if (maintenanceTicker) clearInterval(maintenanceTicker);

	// Notify the Go WASM Engine to enforce maintenance guards
	if (window.SetMaintenanceState) window.SetMaintenanceState(active);

	if (!active) {
		bar.classList.add("hidden");
		return;
	}

	bar.classList.remove("hidden");
	const targetTime = new Date(targetTimestamp).getTime();

	const tick = () => {
		const now = Date.now();
		const diff = targetTime - now;

		if (diff <= 0) {
			timerDisplay.innerText = "STARTING NOW";
			return;
		}

		const mins = Math.floor(diff / 60000);
		const secs = Math.floor((diff % 60000) / 1000);
		timerDisplay.innerText = `${String(mins).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
	};

	tick();
	maintenanceTicker = setInterval(tick, 1000);
}

function sendChatMessage() {
	const input = document.getElementById("chat-input");
	const text = input.value.trim();
	if (!text || !socket) return;

	const envelope = {
		type: "chat",
		payload: { text: text }
	};
	socket.send(JSON.stringify(envelope));
	input.value = "";
}

// --- Tournament History Logic ---
function switchHofTab(tab) {
	document.getElementById("hof-rankings-view").classList.add("hidden");
	document.getElementById("hof-history-view").classList.add("hidden");
	document.getElementById("hof-seasons-view").classList.add("hidden");
	document.getElementById("tab-rankings").classList.remove("active");
	document.getElementById("tab-history").classList.remove("active");
	document.getElementById("tab-seasons").classList.remove("active");

	if (tab === 'rankings') {
		document.getElementById("hof-rankings-view").classList.remove("hidden");
		document.getElementById("tab-rankings").classList.add("active");
		fetchLeaderboard();
	} else if (tab === 'history') {
		document.getElementById("hof-history-view").classList.remove("hidden");
		document.getElementById("tab-history").classList.add("active");
		fetchTournamentHistory(1);
	} else if (tab === 'seasons') {
		// Clear previous filter when opening the tab
		const filterInput = document.getElementById("season-filter-input");
		if (filterInput) filterInput.value = "";
		document.getElementById("hof-seasons-view").classList.remove("hidden");
		document.getElementById("tab-seasons").classList.add("active");
		fetchSeasonHistory();
	}
}

function toggleTournamentDetails(id) {
	const details = document.getElementById(`details-`);
	if (!details) return;
	details.classList.toggle("hidden");
}

async function fetchTournamentHistory(page = 1, deepVerify = false) {
	const prevBtn = document.getElementById("prev-tournament-btn");
	const nextBtn = document.getElementById("next-tournament-btn");
	if (prevBtn) prevBtn.disabled = true;
	if (nextBtn) nextBtn.disabled = true;

	const oldPage = currentTournamentPage;
	currentTournamentPage = page;
	const list = document.getElementById("tournament-history-list");
	list.innerHTML = `<div class="chat-msg system">${deepVerify ? 'Executing Deep Reconstruction...' : 'Decrypting archives...'}</div>`;
	
	try {
		const url = `${CONFIG.API_BASE}/api/tournament/history?page=&limit=${deepVerify ? '&deep_verify=true' : ''}`;
		const response = await fetch(url);

		// Handle Indexer 404/General Failure (Bad Gateway from server)
		if (response.status === 502) {
			list.innerHTML = '<div class="chat-msg system" style="color: var(--warning-orange);">⚠️ The Indexer is temporarily unreachable. Historical tournament data cannot be retrieved at this time.</div>';
			if (document.getElementById("tournament-pagination")) document.getElementById("tournament-pagination").classList.add("hidden");
			return;
		}

		if (!response.ok) throw new Error("Server error");
		
		const result = await response.json();
		const history = result.history || [];
		totalTournaments = result.total || 0;
		
		list.innerHTML = "";
		if (history.length === 0) {
			list.innerHTML = '<div class="chat-msg system">No recorded events found in the database.</div>';
			document.getElementById("tournament-pagination").classList.add("hidden");
			return;
		}

		document.getElementById("tournament-pagination").classList.remove("hidden");
		updateTournamentPaginationUI();

		// Batch resolve Envoi names for all participants in the history page
		const participants = new Set();
		history.forEach(t => {
			participants.add(t.winner);
			t.matches.forEach(m => {
				if (m.p1) participants.add(m.p1);
				if (m.p2) participants.add(m.p2);
			});
		});
		await Promise.all(Array.from(participants).filter(p => p && p !== "TBD").map(p => resolveEnvoiName(p)));

		// API is already sorted newest first
		history.forEach(t => {
			const div = document.createElement("div");
			div.className = "glass-panel";
			div.style.margin = "0";
			div.style.borderColor = "var(--neon-purple)";
			
			const date = new Date(t.timestamp).toLocaleDateString();
			const time = new Date(t.timestamp).toLocaleTimeString();
			
			const verifyBtn = !t.is_verified ? `
				<div style="margin-top: 10px;">
					<button class="outline" style="font-size: 10px; padding: 4px 12px; border-color: #ffa657; color: #ffa657;" 
							onclick="event.stopPropagation(); fetchTournamentHistory(, true);">
						🔍 DEEP VERIFY DATA
					</button>
				</div>
			` : '';

			div.innerHTML = `
				<div style="cursor: pointer;" onclick="toggleTournamentDetails('${t.id}')">
					<div style="display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid var(--glass-border); padding-bottom: 10px; margin-bottom: 15px;">
					<div style="display: flex; align-items: center;">
						<span style="font-weight: bold; color: var(--neon-purple); letter-spacing: 1px;">${t.id}</span>
						${window.GetTournamentArchiveBadge ? window.GetTournamentArchiveBadge(t.is_verified, t.links) : ''}
					</div>
					<span style="color: var(--neon-cyan); font-weight: bold;">POT: ${t.pot.toFixed(1)} </span>
				</div>
				<div style="display: flex; align-items: center; gap: 20px; justify-content: center; margin: 20px 0;">
					 <div style="text-align: center;">
						<div style="font-size: 0.8em; opacity: 0.6; margin-bottom: 5px;">TOURNAMENT CHAMPION</div>
						<div style="font-size: 1.5em; font-weight: bold; color: var(--neon-green); text-shadow: 0 0 10px var(--neon-green);">${getCachedEnvoiName(t.winner)}</div>
						
					 </div>
				</div>
				<div style="display: flex; justify-content: space-between; font-size: 0.8em; opacity: 0.5;">
					<span>Matches: ${t.matches.length}</span>
					<span>Archived:  </span>
				</div>
				</div>
				<div id="details-${t.id}" class="hidden" style="margin-top: 20px; padding-top: 20px; border-top: 1px dashed var(--glass-border); display: flex; gap: 30px; overflow-x: auto; padding-bottom: 15px; scrollbar-width: thin;">
					${generateBracketHTML(t.matches, -1)}
				</div>
			`;
			list.appendChild(div);
		});
	} catch (err) {
		currentTournamentPage = oldPage;
		updateTournamentPaginationUI();
		list.innerHTML = '<div class="chat-msg system" style="color: #ff4b4b;">Database Error: Could not retrieve archives.</div>';
	}
}

function filterSeasonHistory() {
	const input = document.getElementById("season-filter-input");
	const val = input.value.trim();
	fetchSeasonHistory(val ? parseInt(val) : null);
}

async function fetchSeasonHistory(seasonNum = null) {
	const list = document.getElementById("season-history-list");
	list.innerHTML = `<div class="chat-msg system">Consulting the Oracle for past epochs...</div>`;
	
	try {
		const url = seasonNum ? `${CONFIG.API_BASE}/api/season/history?season=` : `${CONFIG.API_BASE}/api/season/history`;
		const response = await fetch(url);
		
		// Handle Indexer 404 (Bad Gateway from server) when reward asset is not found
		if (response.status === 502) {
			list.innerHTML = '<div class="chat-msg system" style="color: var(--warning-orange);">⚠️ The Indexer is currently unable to locate the reward asset history. It may be initializing or the asset ID is invalid.</div>';
			return;
		}

		if (!response.ok) throw new Error("Server error");
		
		const history = await response.json();
		list.innerHTML = "";

		if (history.length === 0) {
			const msg = seasonNum ? `No records found for Season .` : "The history books are currently empty. Check back after the next rollover!";
			list.innerHTML = `<div class="chat-msg system"></div>`;
			return;
		}

		for (const s of history) {
			const div = document.createElement("div");
			div.className = "glass-panel";
			div.style.margin = "0";
			div.style.borderColor = "var(--neon-cyan)";
			
			const startDate = new Date(s.start).toLocaleDateString();
			const endDate = new Date(s.end).toLocaleDateString();

			let winnersHTML = "";
			for (let i = 0; i < s.top.length; i++) {
				const entry = s.top[i];
				const name = await resolveEnvoiName(entry.w);
				winnersHTML += `
					<div class="leaderboard-row season-winner-row">
						<span class="rank-badge" style="color: ${i === 0 ? 'var(--neon-green)' : 'inherit'}">#${i+1}</span>
						<span class="player-name"></span>
						<span class="player-stats"><b>${entry.v}</b> Wins | <small>${entry.r}</small></span>
					</div>
				`;
			}

			div.innerHTML = `
				<div class="season-card-header">
					<span style="font-weight: bold; color: var(--neon-cyan); letter-spacing: 2px;">SEASON ${s.season}</span>
					<span style="font-size: 0.8em; opacity: 0.6;"> — </span>
				</div>
				<div class="season-performers-label">TOP PERFORMERS</div>
				<div class="season-winners-list">
					
				</div>
			`;
			list.appendChild(div);
		}
	} catch (err) {
		console.error("[SEASON HISTORY] Fetch failed:", err);
		list.innerHTML = '<div class="chat-msg system" style="color: #ff4b4b;">Indexer connection failed. Archives are temporarily unavailable.</div>';
	}
}

function handleChatKey(e) {
	if (e.key === 'Enter') sendChatMessage();
}

function renderChatMessage(sender, text) {
	const display = document.getElementById("chat-display");
	const msgDiv = document.createElement("div");
	msgDiv.className = "chat-msg";
	
	if (sender === "SERVER") msgDiv.classList.add("system");
	
	msgDiv.innerHTML = `<b>:</b> `;
	display.appendChild(msgDiv);
	
	// Auto-scroll to bottom
	display.scrollTop = display.scrollHeight;
}

// --- Match History Persistence ---
async function saveMatchResult(state) {
	const history = JSON.parse(localStorage.getItem("vbabes_history") || "[]");
	const opponent = currentOpponentId || (state.multiplayer ? "Unknown Human" : "Vbabe Bot");
	
	const newEntry = {
		winner: state.winner,
		scores: state.scores,
		opponent: opponent,
		timestamp: new Date().toLocaleString()
	};

	history.unshift(newEntry);
	if (history.length > 10) history.pop(); // Keep last 10 matches
	localStorage.setItem("vbabes_history", JSON.stringify(history));
	await renderMatchHistory();
}

async function renderMatchHistory() {
	const history = JSON.parse(localStorage.getItem("vbabes_history") || "[]");
	const display = document.getElementById("history-display");
	if (!display || history.length === 0) return;
	
	// Batch resolve names for wallets in local history
	const wallets = history.map(e => e.opponent).filter(o => o && o.length > 50);
	await Promise.all(wallets.map(w => resolveEnvoiName(w)));
	
	display.innerHTML = "";
	history.forEach(entry => {
		const div = document.createElement("div");
		div.className = "chat-msg";
		const colors = ["var(--neon-green)", "#ff4b4b", "var(--neon-cyan)"]; // Win, Loss, Draw
		const labels = ["WIN", "LOSS", "DRAW"];
		const color = colors[entry.winner] || "white";
		const label = labels[entry.winner] || "END";

		const opponentDisplay = getCachedEnvoiName(entry.opponent);

		div.innerHTML = `<span style="color: ; font-weight: bold;"></span> vs  <br/> 
						 <small style="opacity: 0.7;">${entry.scores[0]}-${entry.scores[1]} | ${entry.timestamp}</small>`;
		display.appendChild(div);
	});
}

// --- Transaction Feedback (Toast) ---
function showToast(message, type = 'info', duration = 5000) {
	const container = document.getElementById("toast-container");
	const toast = document.createElement("div");
	toast.className = `toast `;
	toast.innerHTML = message;
	container.appendChild(toast);

	if (duration > 0) {
		setTimeout(() => {
			toast.style.opacity = '0';
			toast.style.transform = 'translateX(100%)';
			toast.style.transition = '0.5s';
			setTimeout(() => toast.remove(), 500);
		}, duration);
	}
}

// Bridge for Go to trigger Payout UI flow
window.processRewardPayout = async (payloadStr) => {
	const payload = JSON.parse(payloadStr);
	showToast("🛰️ Requesting secure nonce from server...", "info");

	try {
		// 1. Request Nonce from Server via WebSocket (Anti-Replay)
		const nonce = await new Promise((resolve, reject) => {
			const timeout = setTimeout(() => reject(new Error("Nonce request timed out")), 10000);
			nonceResolver = (n) => { clearTimeout(timeout); resolve(n); };
			socket.send(JSON.stringify({ type: "nonce_request" }));
			setTransactionStatus("Requesting secure nonce...", "info");
		});

		showToast("🔐 Please sign the verification request in your wallet...", "info");

		// 2. Construct 0-amount "Reverse Sign" Transaction
		const tx = {
			from: userAddress,
			to: userAddress,
			amount: 0,
			note: new TextEncoder().encode(nonce),
			type: 'pay'
		};

		setTransactionStatus("Signing verification request...", "info");
		// 3. Request Signature via active provider
		let signedTx = null;
		
		// The payload identifies the winner, but the faucet pays the specified recipient.
		payload.claimant = payload.recipient; // The authenticated playing address
		payload.recipient = payoutAddress || payload.recipient; // The Voi payout target

		const isEVM = payload.claimant.startsWith("0x");

		if (isEVM) {
			if (walletProvider === 'walletconnect' && signClient) {
				const sessions = signClient.session.getAll();
				if (sessions.length === 0) throw new Error("No active WalletConnect session found for EVM");
				
				// Encode nonce as hex for EVM personal_sign
				const msgBuffer = new TextEncoder().encode(nonce);
				const hexMsg = "0x" + Array.from(msgBuffer).map(b => b.toString(16).padStart(2, '0')).join('');
				
				const signature = await signClient.request({
					topic: sessions[0].topic,
					chainId: "eip155:1", // Default to mainnet for verification
					request: {
						method: "personal_sign",
						params: [hexMsg, payload.claimant]
					}
				});
				if (!signature) throw new Error("Signature denied.");
				signedTx = new TextEncoder().encode(signature);
			} else {
				throw new Error("EVM verification requires WalletConnect.");
			}
		} else if (walletProvider === 'nautilus') {
			const result = await window.algo.signTxn([
				{ txn: algosdk.encodeObj(tx), signers: [payload.claimant] }
			]);
			signedTx = result[0];
		} else if (walletProvider === 'kibisis') {
			const txnB64 = btoa(String.fromCharCode(...algosdk.encodeObj(tx)));
			const result = await window.kibisis.signTxns([{ txn: txnB64 }]);
			// Kibisis returns base64 strings
			signedTx = new Uint8Array(atob(result[0]).split("").map(c => c.charCodeAt(0)));
		} else if (walletProvider === 'walletconnect' && signClient) {
			const sessions = signClient.session.getAll();
			const txnB64 = btoa(String.fromCharCode(...algosdk.encodeObj(tx)));
			const response = await signClient.request({
				topic: sessions[0].topic,
				chainId: (window.GetGameState().network === "VOI" ? CONFIG.VOI_CHAIN_ID : CONFIG.ALGO_CHAIN_ID),
				request: {
					method: "algo_signTxn",
					params: [[{ txn: txnB64, signers: [payload.claimant] }]]
				}
			});
			signedTx = new Uint8Array(atob(response[0]).split("").map(c => c.charCodeAt(0)));
		}

		if (!signedTx) throw new Error("Signature cancelled or provider error");

		payload.signed_tx = signedTx;
		setTransactionStatus("Submitting proof to Switchboard...", "info");
		showToast("🛰️ Submitting proof to Switchboard...", "info");

		const response = await fetch(`${CONFIG.API_BASE}/api/reward`, {
			method: "POST",
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(payload)
		});
		
		if (!response.ok) {
			const errorMsg = await response.text();
			showToast(`❌ Payout Error: `, "error");
			setTransactionStatus("Payout Failed!", "error");
			return;
		}

		const data = await response.json();
		
		let successMsg = `✅ Reward Sent! TxID: ${data.txid.substring(0, 14)}...`;
		if (data.bonus_applied) {
			successMsg += `<br><span style="color: var(--neon-cyan); font-weight: bold; font-size: 0.85em; text-shadow: 0 0 8px var(--neon-cyan);">[ REPUTATION BONUS ACTIVE ⚡ ]</span>`;
		}
		
		// Handle skipped assets UI
		if (data.skipped_assets && data.skipped_assets.length > 0) {
			const symbols = [];
			for (const id of data.skipped_assets) {
				// Use resolveAssetSymbol to populate cache then pull symbol
				const sym = await resolveAssetSymbol(id);
				symbols.push(sym);
			}
			successMsg += `<br><small style="color: #ffa657; font-size: 0.8em; font-style: italic;">⚠️ Some rewards skipped (Low Pool): ${symbols.join(", ")}</small>`;
		}

		showToast(successMsg, "success", 8000);
		setTransactionStatus("Reward Sent! Confirming...", "success");
		updateWalletUI(userAddress);
		syncUI(); // Trigger UI sync to show updated balance
	} catch (err) {
		showToast("⚠️ Payout Failed: " + err.message, "error");
		setTransactionStatus("Payout Failed!", "error");
	} finally {
		setTimeout(() => setTransactionStatus(null), 3000);
	}
};

function showChallengeNotification(challengerId) {
	currentChallengerId = challengerId;
	const challengeOverlay = document.getElementById("challenge-overlay");
	const challengeText = document.getElementById("challenge-text");

	challengeText.innerText = ``;
	challengeOverlay.classList.remove("hidden");
	// Optionally play a sound or vibrate
}

function acceptChallenge() {
	if (!socket || !currentChallengerId) return;
	const state = window.GetGameState();
	const envelope = {
		type: "challenge",
		to_id: currentChallengerId,
			from_id: myClientId, // Ensure from_id is set for server
		payload: { 
			action: "accept",
			to_id: currentChallengerId,
			deck: state.deck.map(c => c.id),
			avatar: state.p1_avatar,
			gloat: state.p1_gloat,
			rules: state.rules
		}
	};

	socket.send(JSON.stringify(envelope));
	document.getElementById("challenge-overlay").classList.add("hidden");
	currentChallengerId = null;
}

function sendMatchSync(targetId) {
	const state = window.GetGameState();
	const envelope = {
		type: "challenge",
		to_id: targetId,
		from_id: myClientId, // Ensure from_id is set for server
		payload: { 
			action: "sync_back", 
			deck: state.deck.map(c => c.id),
			avatar: state.p1_avatar,
			gloat: state.p1_gloat
		}
	};
	socket.send(JSON.stringify(envelope));
}

function reportGloat(opponentClientId, gloatText) {
	if (!socket || socket.readyState !== WebSocket.OPEN) {
		showToast("Cannot report: Not connected to server.", "error");
		return;
	}
	if (!confirm("Are you sure you want to report this gloat message as offensive?")) {
		return;
	}

	const envelope = {
		type: "report_gloat",
		payload: {
			opponent_client_id: opponentClientId,
			gloat_text: gloatText
		}
	};
	socket.send(JSON.stringify(envelope));
	showToast("Gloat message reported. Thank you for helping keep the arena clean!", "success");
}

function declineChallenge() {
	if (!socket || !currentChallengerId) return;

	const envelope = {
		type: "challenge",
		from_id: myClientId, // Ensure from_id is set for server
		to_id: currentChallengerId,
		payload: { action: "decline" }
	};

	socket.send(JSON.stringify(envelope));
	document.getElementById("challenge-overlay").classList.add("hidden");
	currentChallengerId = null;
}

function sendSpectate(targetId) {
	if (!socket || socket.readyState !== WebSocket.OPEN) return;

	const envelope = {
		type: "spectate",
		from_id: myClientId, // Ensure from_id is set for server
		payload: { target_id: targetId }
	};
	spectatorMatchState = null; // Reset for new spectate session

	socket.send(JSON.stringify(envelope));
	showToast(`👁️ Requesting access to stream...`, "info");
}

function showMatchPreview(data) {
	document.getElementById("preview-p1-id").innerText = data.p1_id;
	document.getElementById("preview-p1-rating").innerText = data.p1_rating || "[Z]";
	document.getElementById("preview-p2-id").innerText = data.p2_id;
	document.getElementById("preview-p2-rating").innerText = data.p2_rating || "[Z]";
	
	document.getElementById("match-preview-overlay").classList.remove("hidden");
}

function proceedToWarRoom() {
	if (!spectatorMatchState) return;
	
	document.getElementById("match-preview-overlay").classList.add("hidden");
	window.ResetGame();
	window.SetBoardState(spectatorMatchState);
	window.ForceActive();
	syncUI("all");
}

function sendChallenge(targetId) {
	if (!socket || socket.readyState !== WebSocket.OPEN) return;

	const state = window.GetGameState();
	const envelope = {
		type: "challenge",
		from_id: myClientId, // Ensure from_id is set for server
		to_id: targetId,
		payload: { 
			action: "invite",
			avatar: state.p1_avatar || "",
			gloat: state.p1_gloat || "",
			deck: state.deck.map(c => c.id)
		}
	};

	socket.send(JSON.stringify(envelope));
	alert(`Challenge sent to `);
}

function triggerToggleNetwork() {
	window.toggleNetwork();
	syncUI();
}

function selectCard(id) {
	activeCardId = id;
	if (window.PlaySelectSound) window.PlaySelectSound();
	syncUI("inventory"); // Re-render to show the selected card glowing
}

function clickGrid(index) {
	const state = window.GetGameState();
	
	// Multiplayer Guard: Only allow move if it's actually our turn
	if (state.phase === "Active" && state.turn !== myPlayerIndex) {
		console.warn("It is not your turn!");
		return;
	}

	if (activeCardId === null) {
		return;
	}

	const selectedCardId = activeCardId;

	// Execute locally
	const success = window.PlaceCard(index, activeCardId);
	if (success) {
		// If in multiplayer, broadcast the move to the opponent
		if (state.phase === "Active" && currentOpponentId) {
			// Find card power for server verification
			const card = state.deck.find(c => c.id === selectedCardId);
			const envelope = {
				type: "move",
				to_id: currentOpponentId,
				payload: {
					grid_index: index,
					card_id: selectedCardId,
					power: card ? card.power : [0,0,0,0]
				}
			};
			socket.send(JSON.stringify(envelope));
		}
		activeCardId = null; 
		syncUI("combat");
	}
}

// --- Deck Manager Logic ---
function openDeckManager() {
	document.getElementById("deck-manager-overlay").classList.remove("hidden");
	renderDeckManager();
}

function closeDeckManager() {
	document.getElementById("deck-manager-overlay").classList.add("hidden");
	
	// TACTICAL SYNC: Report the highest possible deck rating to the Hall of Fame
	const state = window.GetGameState();
	const rating = calculateDeckRating(state.deck);
	if (socket && socket.readyState === WebSocket.OPEN) {
		socket.send(JSON.stringify({
			type: "update_rating",
			payload: { best_rating: rating }
		}));
	}
	syncUI("all");
}

function renderDeckManager() {
	const state = window.GetGameState();
	const invGrid = document.getElementById("inventory-grid");
	const deckZone = document.getElementById("deck-drop-zone");
	const selector = document.getElementById("deck-selector-bar");
	const deckRatingEl = document.getElementById("deck-stat-summary"); // Target summary area
	const atkEl = document.getElementById("total-atk");
	const defEl = document.getElementById("total-def");

	invGrid.innerHTML = "";
	deckZone.innerHTML = "";
	selector.innerHTML = "";
	const isMobile = window.innerWidth <= 768;

	let totalAtk = 0;
	let totalDef = 0;

	// 1. Render Inventory
	state.inventory.forEach(card => {
		const cardEl = document.createElement("div");
		const isSelected = activeCardId === card.id;
		cardEl.className = `card-mini ${isSelected ? 'selected-item' : ''}`;
		cardEl.draggable = true;
		cardEl.innerHTML = renderCardHTML(card);
		cardEl.ondragstart = (e) => e.dataTransfer.setData("cardID", card.id);
		
		// Mobile Fallback: Tap to select
		cardEl.onclick = () => {
			activeCardId = card.id;
			renderDeckManager();
			if (window.PlaySelectSound) window.PlaySelectSound();
		};

		invGrid.appendChild(cardEl);
	});

	// 2. Render Active Deck
	state.deck.forEach((card, idx) => {
		const cardEl = document.createElement("div");
		cardEl.className = "card-mini";
		cardEl.style.width = "100%";
		cardEl.style.height = "60px";
		cardEl.innerHTML = `<span style="font-size: 10px;">${card.name}</span><button onclick="window.RemoveFromDeck(); renderDeckManager();" style="float: right; padding: 2px 5px; font-size: 9px;">X</button>`;
		
		// Calculate Stats: Attack (Top + Right), Defense (Bottom + Left)
		totalAtk += (card.power[0] + card.power[1]);
		totalDef += (card.power[2] + card.power[3]);
		
		deckZone.appendChild(cardEl);
	});

	// Mobile Fallback: Tap zone to place primed card
	deckZone.onclick = () => {
		if (activeCardId !== null) {
			window.AddToDeck(activeCardId);
			activeCardId = null;
			renderDeckManager();
		}
	};

	atkEl.innerText = totalAtk;
	defEl.innerText = totalDef;

	// 3. Render Deck Selectors (Unlocks)
	const thresholds = [0, 250, 600, 1000];
	for(let i=0; i<4; i++) {
		const btn = document.createElement("button");
		const isLocked = state.reputation < thresholds[i];
		btn.className = `deck-slot-btn ${i === state.active_deck ? 'active' : ''} ${isLocked ? 'locked' : ''}`;
		btn.innerText = isLocked ? `🔒 ${thresholds[i]} REP` : `Deck ${i+1}`;
		btn.onclick = () => { if(!isLocked) { window.SelectDeck(i); renderDeckManager(); } };
		selector.appendChild(btn);
	}
}

// Initialize Drag & Drop
const dropZone = document.getElementById("deck-drop-zone");
dropZone.ondragover = (e) => { e.preventDefault(); dropZone.classList.add("drag-over"); };
dropZone.ondragleave = () => dropZone.classList.remove("drag-over");
dropZone.ondrop = (e) => {
	e.preventDefault();
	dropZone.classList.remove("drag-over");
	const id = parseInt(e.dataTransfer.getData("cardID"));
	window.AddToDeck(id);
	renderDeckManager();
};

// 4. THE RENDER LOOP (The Camera fetching Go State)
async function syncUI(scope = "all") {
	if (!window.GetGameState) return; // Ensure Go function exists
	// Partial state sync: only update sections present in the current state snapshot
	const state = window.GetGameState(scope);
	
	// --- Update Dynamic Environment ---
	if (state.phase !== undefined || state.multiplayer !== undefined || state.tournament !== undefined) {
		updateDynamicArenaFloor(state);
	}

	// Update Deck Rating in UI
	if (state.deck_rating !== undefined) {
		document.getElementById("deck-rating-display").innerText = state.deck_rating;
	}

	// Update Mojo Display
	const mojoEl = document.getElementById("mojo-display"); // Assuming this element exists in index.html
	if (mojoEl && state.mojo !== undefined) mojoEl.innerHTML = `MOJO: ${state.mojo || 0} [${state.social_rank || 'Nobody'}] <span style="font-size: 0.7em; opacity: 0.7; margin-left: 10px;">RUMORS: ${state.rumor_count || 0}</span>`;

	// 0. Resolve missing reward symbols concurrently to prevent UI flickering
	if (state.rewards) {
		const rewardIds = Object.keys(state.rewards || {});
		const missingSymbols = rewardIds.filter(id => !assetCache[id]);
		if (missingSymbols.length > 0) {
			await Promise.all(missingSymbols.map(id => resolveAssetSymbol(id)));
		}
	}
	
	// --- Update Dashboard ---
	// Overlay Management
	if (state.phase !== undefined || state.show_leaderboard !== undefined) {
		hideAllOverlays();
		const mainContainer = document.getElementById("main-game-container");
		mainContainer.classList.add('hidden'); // Hide main game by default

		if (state.show_leaderboard) {
			document.getElementById("leaderboard-overlay").classList.remove("hidden");
		} else if (state.phase === "TournamentLobby") {
			document.getElementById("tournament-overlay").classList.remove("hidden");
			// Populate bracket visualization if data exists
			if (state.tournament) {
				await renderTournamentBracket(state.tournament);
			}
		} else if (state.phase === "Setup" && userAddress) {
			document.getElementById("setup-overlay").classList.remove("hidden");
		} else if (!userAddress) {
			// If no wallet connected, show wallet selector
			document.getElementById("wallet-selector-overlay").classList.remove("hidden");
			renderRumorBoard(); // Ensure rumor board is rendered even if no wallet is connected
		} else {
			// Default to showing main game container if no specific overlay is needed
			mainContainer.classList.remove('hidden');
		}
	}

	// --- Narrative Intelligence Hook & AI Indicator ---
	if (state.phase === "Active" && !state.multiplayer) {
		// 1. Show thinking indicator on AI turn
		if (state.turn === 1) {
			document.getElementById("ai-thinking-indicator").classList.remove("hidden");
		} else {
			document.getElementById("ai-thinking-indicator").classList.add("hidden");
		}

		// 2. Trigger taunt on phase entry or turn change
		if (state.phase !== lastTauntPhase || state.turn !== lastTauntTurn) {
			if (state.playstyle) {
				const npcName = state.p2_id || "Bot";
				const taunt = collectiveIntelligence.generatePlaystyleTaunt(npcName, state.playstyle);
				if (taunt) renderChatMessage("SYSTEM", taunt);
			}
			lastTauntPhase = state.phase;
			lastTauntTurn = state.turn;
		}
	} else {
		document.getElementById("ai-thinking-indicator")?.classList.add("hidden");
		lastTauntPhase = state.phase;
		lastTauntTurn = null;
	}

	// --- Winner Overlay: Character-Aware Feedback ---
	if (state.phase === "Finished" && state.winner !== undefined) {
		const overlay = document.getElementById("winner-overlay");
		const winText = document.getElementById("winner-text");
		const scoreText = document.getElementById("score-text");

		if (overlay) overlay.classList.remove("hidden");

		if (winText && scoreText) {
			let title = "MATCH OVER";
			let gloat = "";
			
			const localPIdx = state.local_player_index !== undefined ? state.local_player_index : myPlayerIndex;
			const isWinner = state.winner === localPIdx;
			const isDraw = state.winner === 2;

			if (isDraw) {
				title = "DRAW";
				winText.style.color = "var(--neon-cyan)";
				gloat = "Perfect balance. Neither side could find the opening.";
			} else if (isWinner) {
				title = "VICTORY";
				winText.style.color = "var(--neon-green)";
				const winnerGloat = (localPIdx === 0) ? state.p1_gloat : state.p2_gloat;
				const defaultGloat = state.multiplayer ? "Victory achieved in combat." : "The Arena recognizes your dominance.";
				gloat = (state.multiplayer && winnerGloat) ? winnerGloat : defaultGloat;
			} else {
				title = "DEFEAT";
				winText.style.color = "#ff4b4b";
				const opponentGloat = (localPIdx === 0) ? state.p2_gloat : state.p1_gloat;

				if (state.multiplayer) {
					const rawGloat = opponentGloat || "Your opponent has prevailed.";
					gloat = rawGloat + `<span class="report-gloat-icon" onclick="reportGloat('', '${rawGloat.replace(/'/g, "\'")}')" title="Report offensive gloat"> 🚨</span>`;
				} else {
					// Archetype gloats based on SpecialFanfare assigned in main.go
					switch (state.special_fanfare) {
						case "Witch": gloat = "A charming attempt, but your luck has run out! Hexed!"; break;
						case "Boss": gloat = "Calculated. Efficient. You were never a variable in my success."; break;
						case "Lady": gloat = "Don't look so sad, darling. You simply weren't a match for me."; break;
						case "cute": gloat = "Tee-hee! I won! You're still my favorite person to play with though!"; break;
						default: gloat = "The Vbabe Bot has outplayed you this time.";
					}
				}
			}

			winText.innerText = title;
			scoreText.innerHTML = `${state.scores[0]} - ${state.scores[1]}<br/><span style="font-size: 0.5em; opacity: 0.8; letter-spacing: 2px; display: block; margin-top: 15px; color: #fff; font-family: 'Rajdhani', sans-serif; text-transform: uppercase;">""</span>`;
		}
	}

	// Challenge Overlay (if active)
	if (currentChallengerId) { // This is managed by `showChallengeNotification`
		document.getElementById("challenge-overlay").classList.remove("hidden");
	}

	if (state.faucet !== undefined) {
		const faucetEl = document.getElementById("faucet-display");
		const faucetValue = state.faucet.toFixed(2);
		const currentHTML = faucetEl.innerHTML;
		let newHTML = "";
		
		if (state.faucet < 50) {
			newHTML = `  <span style="font-size: 0.7em; margin-left: 5px;">[ VAULT LOW ]</span>`;
			faucetEl.classList.add("faucet-depleted");
		} else {
			newHTML = faucetValue + " ";
			faucetEl.classList.remove("faucet-depleted");
		}
		
		if (currentHTML !== newHTML) faucetEl.innerHTML = newHTML;
	}

	// --- Update Rewards Dashboard (Economy Scope) ---
	if (state.rewards !== undefined) {
		const rewardsDashboard = document.getElementById("rewards-dashboard");
		if (rewardsDashboard) {
			let totalValue = 0;
			let rewardItems = [];
			Object.entries(state.rewards || {}).forEach(([id, amt]) => {
				totalValue += amt;
				const symbol = getAssetSymbol(id);
				rewardItems.push(`<span style="color: var(--neon-green)">${amt.toFixed(1)}</span> <small></small>`);
			});
			const playerRewards = state.rewards[CONFIG.VBV_ASSET_ID] || 0;
			const rumorCost = 500;
			const myJailedCards = state.jailed_cards || {};
			const myKidnappedCards = state.kidnapped_cards || {};
			const myHeldHostageCards = state.held_hostage_cards || {};
			const wantedVal = state.wanted_level || 0;
			const cunningVal = state.cunning || 0;
			const nurturingVal = state.nurturing || 0; // Get nurturing value
			const jobRole = state.job_role || "";
			const outlawsInLobby = lastLobbyPlayers.filter(p => (p.wanted_level || 0) >= 10);

			const courthouseBtn = wantedVal > 0 ? ` <button class="outline" style="padding: 2px 8px; font-size: 10px; margin-left: 10px; border-color: #ff4b4b; color: #ff4b4b;" onclick="openCourthouse()">⚖️ COURTHOUSE ()</button>` : '';
			const blackMarketBtn = (wantedVal >= 5 && cunningVal >= 10) ? ` <button class="outline" style="padding: 2px 8px; font-size: 10px; margin-left: 10px; border-color: #ff4b4b; color: #ff4b4b;" onclick="openBlackMarket()">🏴‍☠️ BLACK MARKET</button>` : '';
			const rumorMillBtn = (playerRewards >= rumorCost) ? ` <button class="outline" style="padding: 2px 8px; font-size: 10px; margin-left: 10px; border-color: var(--neon-green); color: var(--neon-green);" onclick="openRumorMill()">📢 RUMOR MILL</button>` : '';
			const securityBtn = (jobRole === "Security") ? ` <button class="outline" style="padding: 2px 8px; font-size: 10px; margin-left: 10px; border-color: var(--neon-cyan); color: var(--neon-cyan);" onclick="openSecuritySentry()">🛡️ SECURITY SENTRY</button>` : '';
			const bountyBoardBtn = (outlawsInLobby.length > 0 || wantedVal <= 2) ? ` <button class="outline" style="padding: 2px 8px; font-size: 10px; margin-left: 10px; border-color: #ffd700; color: #ffd700;" onclick="openBountyBoard()">🎯 BOUNTY BOARD (${outlawsInLobby.length})</button>` : '';
			const leaseBoardBtn = ` <button class="outline" style="padding: 2px 8px; font-size: 10px; margin-left: 10px; border-color: var(--neon-purple); color: var(--neon-purple);" onclick="openClubLeaseBoard()">📜 LEASE BOARD</button>`;

			const newHTML = `Win Total: <b style="color: var(--neon-green); text-shadow: 0 0 10px var(--neon-green);">${totalValue.toFixed(1)}</b> | ` + rewardItems.join(" + ") +
				` <span style="margin-left: 10px; color: var(--neon-cyan); font-weight: bold;">CUNNING: </span>` + // Display Cunning
				` <span style="margin-left: 10px; color: var(--neon-purple); font-weight: bold;">NURTURING: </span>` + // Display Nurturing
				` <button class="outline" style="padding: 2px 8px; font-size: 10px; margin-left: 10px; border-color: var(--neon-cyan); color: var(--neon-cyan);" onclick="openTrophyView()">🏆 TROPHIES (${unlocked.size})</button>` +
				` <button class="outline" style="padding: 2px 8px; font-size: 10px; margin-left: 10px; border-color: var(--neon-purple); color: var(--neon-purple);" onclick="openPortfolioView()">VIEW PORTFOLIO</button>` + 
				courthouseBtn + blackMarketBtn + rumorMillBtn + securityBtn + bountyBoardBtn + leaseBoardBtn + 
				(Object.keys(myJailedCards).length > 0 ? ` <button class="outline" style="padding: 2px 8px; font-size: 10px; margin-left: 10px; border-color: #ff4b4b; color: #ff4b4b;" onclick="openPortfolioView('jailed')">⛓️ JAILED CARDS (${Object.keys(myJailedCards).length})</button>` : '') + 
				(Object.keys(myKidnappedCards).length > 0 ? ` <button class="outline" style="padding: 2px 8px; font-size: 10px; margin-left: 10px; border-color: #ff4b4b; color: #ff4b4b;" onclick="openPortfolioView('kidnapped')">😈 KIDNAPPED (${Object.keys(myKidnappedCards).length})</button>` : '') + 
				(Object.keys(myHeldHostageCards).length > 0 ? ` <button class="outline" style="padding: 2px 8px; font-size: 10px; margin-left: 10px; border-color: #ffd700; color: #ffd700;" onclick="openPortfolioView('hostage')">🛑 HOSTAGE (${Object.keys(myHeldHostageCards).length})</button>` : '');
			
			if (rewardsDashboard.innerHTML !== newHTML) {
				rewardsDashboard.innerHTML = newHTML;
			}
		}
	}

	// --- Update Latency ---
	const latencyEl = document.getElementById("latency-display");
	if (state.latency > 0) {
		latencyEl.innerText = `${state.latency} ms (${state.network_health})`;
		// Use health levels for UI color
		const colors = {"Excellent": "var(--neon-green)", "Good": "#ffd700", "Poor": "#ffa657", "Critical": "#ff4b4b"};
		latencyEl.style.color = colors[state.network_health] || "white";
	} else {
		latencyEl.innerText = "-- ms";
	}
	renderRumorBoard(); // Ensure rumor board is updated

	updateAdminRewardList(state.rewards);

	document.getElementById("network-display").innerText = state.network;

	// --- Update Avatars from WASM URLs ---
	// Update music toggle button icon
	const musicToggleBtn = document.getElementById("music-toggle-btn");
	if (musicToggleBtn) {
		musicToggleBtn.innerText = state.musicVolume === 0 ? "🔇" : "🎵";
		musicToggleBtn.title = state.musicVolume === 0 ? "Unmute Music" : "Mute Music";
	}

	document.getElementById("p1-avatar").style.backgroundImage = `url('${state.p1_avatar}')`;
	document.getElementById("p2-avatar").style.backgroundImage = `url('${state.p2_avatar}')`;

	// --- Update Avatar Ban Notice ---
	const noticeEl = document.getElementById("avatar-notice-banner");
	if (state.p1_avatar_notice) {
		if (noticeEl) {
			noticeEl.classList.remove("hidden");
			noticeEl.innerText = state.p1_avatar_notice;
		}
	} else if (noticeEl) {
		noticeEl.classList.add("hidden");
	}

	// --- Admin Panel Visibility ---
	const adminPanel = document.getElementById("admin-control-panel");
	if (state.is_admin) {
		adminPanel.classList.remove("hidden");
		
		// Sync checkbox states from engine
		if (state.rules) {
			document.getElementById("rule-open").checked = state.rules.Open;
			document.getElementById("rule-same").checked = state.rules.Power_copy;
			document.getElementById("rule-plus").checked = state.rules.Power_up;
			document.getElementById("rule-elemental").checked = state.rules.Elemental_sync;
			document.getElementById("rule-fallen").checked = state.rules.Fallen_penalty;
			document.getElementById("rule-artifact").checked = state.rules.Artifact_bonus;
		}

		// Update Power Scaling Sliders
		if (state.power_divisor) {
			document.getElementById("admin-power-divisor").value = state.power_divisor;
			document.getElementById("admin-power-base").value = state.power_base;
		}

		document.getElementById("dev-mode-toggle").checked = state.testing_mode;
		if (!adminLogTicker) startAdminLogPolling();
	} else {
		if (adminLogTicker) clearInterval(adminLogTicker); // Stop polling if admin panel is closed
		adminPanel.classList.add("hidden");
	}

	// --- Logic for Saving History (Moved to after overlay logic) ---
	if (state.phase === "Active") { matchHistorySaved = false; }
	else if (state.phase === "Finished" && !matchHistorySaved) { await saveMatchResult(state); matchHistorySaved = true; }

	// --- Update Turn Display ---
	let turnDisplay = "Lobby";
	if (state.phase === "Active") turnDisplay = state.turn === 0 ? "Your Turn" : "Bot Thinking...";
	if (state.phase === "Finished") turnDisplay = "Match Over";
	document.getElementById("turn-display").innerText = turnDisplay;

	// --- Render 3x3 Board ---
	if (scope === "all" || scope === "combat") {
		const boardContainer = document.getElementById("board-container");
		if (boardContainer) {
			// Initialize grid slots if they don't exist (first render)
			if (boardContainer.children.length === 0) {
				for (let i = 0; i < 9; i++) {
					const slot = document.createElement("div");
					slot.className = "grid-slot";
					slot.onclick = () => clickGrid(i);
					boardContainer.appendChild(slot);
				}
			}

			state.board.forEach((card, index) => {
				const slot = boardContainer.children[index];
				const prevCard = lastBoardState[index];
				const tileMood = state.board_moods ? state.board_moods[index] : "Neutral";

				// Update Mood Visuals for the slot
				const moodClass = `mood-${tileMood.toLowerCase()}`;
				// Remove old mood classes, add new one if not neutral
				// Resetting className to "grid-slot" first ensures only current mood is applied
				if (!slot.classList.contains("grid-slot")) { // Avoid unnecessary re-assignment
					slot.className = "grid-slot";
				}
				if (tileMood !== "Neutral" && !slot.classList.contains(moodClass)) {
					slot.classList.add(moodClass);
				} else if (tileMood === "Neutral" && slot.classList.contains(moodClass)) {
					slot.classList.remove(moodClass);
				}

				// Handle card presence and changes
				if (card) {
					let cardDiv = slot.querySelector(".playing-card");
					const isCaptured = card && prevCard && card.owner !== prevCard.owner;

					if (!cardDiv) {
						// Card added to an empty slot
						cardDiv = document.createElement("div");
						cardDiv.className = "playing-card";
						slot.appendChild(cardDiv);
					}

					// Apply flip animation if captured
					if (isCaptured) {
						cardDiv.classList.add("flip-capture");
						// Remove after animation to allow re-triggering
						cardDiv.addEventListener('animationend', () => {
							cardDiv.classList.remove("flip-capture");
						}, { once: true });
					}

					// Update card content and styling
					const newCardHTML = renderCardHTML(card);
					if (cardDiv.innerHTML !== newCardHTML) { // Only update if content changed
						cardDiv.innerHTML = newCardHTML;
					}
					const newBorderColor = card.owner === 0 ? "var(--neon-cyan)" : "#ff4b4b";
					if (cardDiv.style.borderColor !== newBorderColor) { // Only update if color changed
						cardDiv.style.borderColor = newBorderColor;
					}

					// Tooltip Interaction (re-attach if cardDiv was new or replaced)
					cardDiv.onmouseenter = (e) => {
						if (tooltipEl && tooltipEl.style.opacity === "1") return;
						showPowerTooltip(e, card, index, state);
					};
					cardDiv.onmousemove = (e) => movePowerTooltip(e);
					cardDiv.onmouseleave = (e) => {
						if (e.relatedTarget === tooltipEl) return;
						hidePowerTooltip();
					};

				} else {
					// Slot is empty
					const cardDiv = slot.querySelector(".playing-card");
					if (cardDiv) {
						slot.removeChild(cardDiv);
					}
				}
			});
		}
	}

	// Update local state tracking for next sync
	lastBoardState = JSON.parse(JSON.stringify(state.board));

	// --- Render Player Hand ---
	const handContainer = document.getElementById("hand-container");
	handContainer.innerHTML = "";
	const placedIds = state.board.filter(c => c !== null).map(c => c.id);
	state.deck.forEach(card => {
		if (!placedIds.includes(card.id)) {
			const cardDiv = document.createElement("div");
			const isSelected = activeCardId === card.id ? 'selected-card' : '';
			cardDiv.className = `playing-card hand-card `;
			cardDiv.onclick = () => selectCard(card.id);
			cardDiv.innerHTML = renderCardHTML(card);
			handContainer.appendChild(cardDiv);
		}
	});
}

// --- Rumor System UI ---
let rumorTimers = {}; // To store setInterval IDs for each rumor countdown

function updateActiveRumors(rumorsData) {
	// Clear existing timers
	for (const id in rumorTimers) {
		clearInterval(rumorTimers[id]);
	}
	rumorTimers = {};

	activeRumors = [];
	if (rumorsData) {
		// If rumorsData is an object (from lobby_update), convert to array
		if (typeof rumorsData === 'object' && !Array.isArray(rumorsData)) {
			for (const id in rumorsData) {
				activeRumors.push(rumorsData[id]);
			}
		} else if (Array.isArray(rumorsData)) { // If it's already an array
			activeRumors = rumorsData;
		} else if (rumorsData.id) { // If it's a single rumor object (from rumor_update)
			activeRumors.push(rumorsData);
		}
	}

	// Filter out expired rumors immediately
	activeRumors = activeRumors.filter(r => new Date(r.ExpiresAt) > Date.now());

	renderRumorBoard();
}

async function renderRumorBoard() {
	let rumorBoard = document.getElementById("rumor-board");
	if (!rumorBoard) {
		rumorBoard = document.createElement("div");
		rumorBoard.id = "rumor-board";
		rumorBoard.className = "rumor-board-container";
		// Find a good place to insert it, e.g., before the chat container
		const chatContainer = document.getElementById("chat-container");
		if (chatContainer) {
			chatContainer.parentNode.insertBefore(rumorBoard, chatContainer);
		} else {
			document.querySelector('.column.right').prepend(rumorBoard); // Fallback
		}
	}

	rumorBoard.innerHTML = "";
	if (activeRumors.length === 0) {
		rumorBoard.classList.add("hidden");
		return;
	}
	rumorBoard.classList.remove("hidden");

	rumorBoard.innerHTML += `<div class="rumor-board-title">ACTIVE RUMORS</div>`;

	for (const rumor of activeRumors) {
		const targetName = await getCachedEnvoiName(rumor.TargetWallet);
		const rumorEl = document.createElement("div");
		rumorEl.className = `rumor-item rumor-${rumor.Type}`;
		rumorEl.innerHTML = `
			<span class="rumor-text">📣 ${rumor.Type.toUpperCase()}: </span>
			<span class="rumor-timer" id="rumor-timer-${rumor.ID}"></span>
		`;
		rumorBoard.appendChild(rumorEl);

		// Start countdown for this rumor
		startRumorCountdown(rumor.ID, rumor.ExpiresAt);
	}
}

function startRumorCountdown(rumorID, expiresAt) {
	const timerEl = document.getElementById(`rumor-timer-`);
	if (!timerEl) return;

	rumorTimers[rumorID] = setInterval(() => {
		const remaining = new Date(expiresAt).getTime() - Date.now();
		if (remaining <= 0) {
			clearInterval(rumorTimers[rumorID]);
			delete rumorTimers[rumorID];
			updateActiveRumors(); // Re-render to remove expired rumor
			return;
		}
		const minutes = Math.floor((remaining % (1000 * 60 * 60)) / (1000 * 60));
		const seconds = Math.floor((remaining % (1000 * 60)) / 1000);
		timerEl.textContent = `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
	}, 1000);
}

// --- Initial UI State ---
// This will be called once after WASM loads and then by syncUI()
if (!userAddress) document.getElementById("wallet-selector-overlay").classList.remove("hidden");

function buildEmptyBoard() {
	const boardContainer = document.getElementById("board-container");
	boardContainer.innerHTML = "";
	for(let i=0; i<9; i++) {
		boardContainer.innerHTML += `<div class="grid-slot" onclick="clickGrid()">Slot </div>`;
	}
}

function renderCardHTML(card) {
	const rarityBadge = (card.rarity && card.rarity > 1.0) ? `<div class="rarity-badge">${card.rarity.toFixed(1)}x</div>` : '';
	
	// Mood Icon Mapping
	let moodHTML = '';
	if (card.mood && card.mood !== "Neutral") {
		const moodClassMap = { "Volatile": "fire", "Serene": "water", "Spirited": "lightning", "Grounded": "earth" };
		const moodEmojiMap = { "Volatile": "🔥", "Serene": "💧", "Spirited": "⚡", "Grounded": "🌿" };
		const mClass = moodClassMap[card.mood] || "";
		const mEmoji = moodEmojiMap[card.mood] || "✨";
		if (mClass) moodHTML = `<div class="card-type-icon " title="Mood: ${card.mood}"></div>`;
	}

	// Artifact / Bonus Display
	let artifactHTML = '';
	if (card.artifact > 0) {
		artifactHTML = `<div class="artifact-badge" style="position: absolute; bottom: 30px; right: 5px; color: var(--neon-cyan); font-size: 9px; font-weight: bold; text-shadow: 0 0 5px var(--neon-cyan);">+${card.artifact}</div>`;
	} else if (card.artifact < 0) {
		artifactHTML = `<div class="debuff-badge">PRISONER ${card.artifact}</div>`;
	}

	// Fatigue & Loyalty Indicators
	const fatigue = card.fatigue || 0;
	const loyalty = card.loyalty || 0;
	const statsHTML = `
		<div class="card-mini-stats" style="position: absolute; bottom: 23px; left: 5px; right: 5px; display: flex; justify-content: space-between; font-size: 7px; font-family: 'Rajdhani', sans-serif; letter-spacing: 0.5px; pointer-events: none;">
			<span style="color: ${fatigue > 50 ? '#ff4b4b' : '#8b949e'}">F:</span>
			<span style="color: ${loyalty >= 100 ? 'var(--neon-green)' : '#8b949e'}">L:</span>
		</div>
	`;

	return `
		
		
		
		<div class="power-grid">
			<div style="grid-area: top">${window.GetLevelLabelForDisplay(card.power[0])}</div>
			<div style="grid-area: left">${window.GetLevelLabelForDisplay(card.power[3])}</div>
			<div style="grid-area: right">${window.GetLevelLabelForDisplay(card.power[1])}</div>
			<div style="grid-area: bottom">${window.GetLevelLabelForDisplay(card.power[2])}</div>
		</div>
		
		<div class="card-name">${card.name}</div>
	`;
}

function showPowerTooltip(e, card, index, state) {
	if (!tooltipEl) {
		tooltipEl = document.createElement("div");
		tooltipEl.className = "power-tooltip";
		document.body.appendChild(tooltipEl);
	}

	const tileMood = state.board_moods ? state.board_moods[index] : "Neutral";
	const moodWeaknesses = { "Volatile": "Serene", "Serene": "Spirited", "Spirited": "Grounded", "Grounded": "Volatile" };
	
	let html = `<div style="color: var(--neon-cyan); font-weight: bold; margin-bottom: 8px; border-bottom: 1px solid var(--neon-cyan); padding-bottom: 5px;">${card.name.toUpperCase()} DATA</div>`;
	
	const sides = ["TOP", "RIGHT", "BOTTOM", "LEFT"];
	
	// Get player stats for the card owner to calculate player-level modifiers
	const ownerPlayerIndex = card.owner;
	const ownerWantedLevel = (ownerPlayerIndex === 0 ? state.p1_wanted_level : state.p2_wanted_level) || 0;
	const ownerCunning = (ownerPlayerIndex === 0 ? state.p1_cunning : state.p2_cunning) || 0;
	const ownerNurturing = (ownerPlayerIndex === 0 ? state.p1_nurturing : state.p2_nurturing) || 0;

	// Calculate player-level modifiers once
	let netWantedPenalty = 0;
	if (ownerWantedLevel > 0) {
		const baseWantedPenalty = ownerWantedLevel * 5;
		const mitigation = ownerCunning * 2;
		netWantedPenalty = -(baseWantedPenalty - Math.min(mitigation, baseWantedPenalty));
	}

	sides.forEach((side, sideIndex) => {
		const base = card.power[i];
		const artifactBonus = card.artifact || 0;
		
		let moodModifier = 0;
		if (state.rules?.Elemental_sync && tileMood !== "Neutral" && card.mood && card.mood !== "Neutral") {
			if (card.mood === tileMood) {
				moodModifier = 50; // Match bonus
			} else if (moodWeaknesses[card.mood] === tileMood) {
				moodModifier = -50; // Weakness penalty
			}
		}

		let netFatiguePenalty = 0;
		if (card.fatigue > 50) {
			const baseFatiguePenalty = (card.fatigue - 50);
			const reduction = ownerNurturing;
			netFatiguePenalty = -(baseFatiguePenalty - Math.min(reduction, baseFatiguePenalty));
		}

		const loyaltyBonus = card.loyalty >= 100 ? 25 : 0;

		const totalEffectivePower = base + artifactBonus + moodModifier + netFatiguePenalty + loyaltyBonus + netWantedPenalty;
		const grade = window.GetLevelLabelForDisplay(totalEffectivePower);
		
		// Build the HTML for modifiers
		let modifiersHtml = '';
		if (artifactBonus !== 0) {
			modifiersHtml += `<span style="color: ${artifactBonus > 0 ? 'var(--neon-cyan)' : '#ff4b4b'}">${artifactBonus > 0 ? '+' : ''}A</span> `;
		}
		if (moodModifier !== 0) {
			modifiersHtml += `<span style="color: ${moodModifier > 0 ? 'var(--neon-green)' : '#ff4b4b'}">${moodModifier > 0 ? '+' : ''}M</span> `;
		}
		if (netFatiguePenalty !== 0) {
			modifiersHtml += `<span style="color: #ff4b4b">F</span> `;
		}
		if (loyaltyBonus !== 0) {
			modifiersHtml += `<span style="color: var(--neon-cyan)">+L</span> `;
		}
		if (netWantedPenalty !== 0) {
			modifiersHtml += `<span style="color: #ff4b4b">W</span> `;
		}

		html += `
			<div class="tooltip-row">
				<span style="opacity: 0.7;">:</span>
				<span style="display: flex; align-items: center; gap: 5px;">
					<span></span>
					${modifiersHtml ? `<span style="font-size: 0.8em; opacity: 0.8;">(${modifiersHtml.trim()})</span>` : ''}
					<span>=</span>
					<b style="color: var(--neon-cyan)"> ()</b>
				</span>
			</div>
		`;
	});

	if (state.rules?.Artifact_bonus && card.owner === myPlayerIndex) {
		html += `
			<div class="tooltip-quickcast">
				<button onclick="event.stopPropagation(); showQuickCastMenu()">
					⚡ QUICK-CAST ITEM
				</button>
			</div>
		`;
	}

	if (card.mood && card.mood !== "Neutral") {
		html += `<div style="margin-top: 8px; font-size: 10px; opacity: 0.6; text-align: center;">MOOD: ${card.mood.toUpperCase()} vs TILE: ${tileMood.toUpperCase()}</div>`;
	}

	tooltipEl.innerHTML = html;
	tooltipEl.style.opacity = "1";
	tooltipEl.style.pointerEvents = (state.rules?.Artifact_bonus && card.owner === myPlayerIndex) ? "auto" : "none";
	tooltipEl.onmouseleave = () => hidePowerTooltip();
	movePowerTooltip(e);
}

function movePowerTooltip(e) {
	if (!tooltipEl) return;
	const padding = 15;
	let x = e.clientX + padding;
	let y = e.clientY + padding;

	// Boundary check to keep tooltip on screen
	if (x + 220 > window.innerWidth) x = e.clientX - 230;
	if (y + 180 > window.innerHeight) y = e.clientY - 190;

	tooltipEl.style.left = x + "px";
	tooltipEl.style.top = y + "px";
}

function hidePowerTooltip() {
	if (tooltipEl) tooltipEl.style.opacity = "0";
}

function showQuickCastMenu(gridIndex) {
	const container = document.querySelector(".tooltip-quickcast");
	if (!container) return;

	const state = window.GetGameState();
	// Filter inventory for items that aren't currently in the active deck
	const deckIds = state.deck.map(c => c.id);
	const artifacts = state.inventory.filter(c => !deckIds.includes(c.id) && c.artifact > 0);
	
	if (artifacts.length === 0) {
		container.innerHTML = `<span style="color: #ff4b4b; font-size: 11px; font-weight: bold;">NO ITEMS AVAILABLE</span>`;
		return;
	}

	let html = `<div class="quickcast-item-list">`;
	artifacts.forEach(item => {
		html += `
			<button class="quickcast-item-btn" onclick="event.stopPropagation(); executeQuickCast(${item.id}, )">
				<span>${item.name}</span>
				<b style="color: inherit;">+${item.artifact}</b>
			</button>
		`;
	});
	html += `</div>`;
	container.innerHTML = html;
}

async function executeQuickCast(itemId, gridIndex) {
	const state = window.GetGameState();
	const item = state.inventory.find(c => c.id === itemId);
	if (!item) return;

	const success = window.ApplyArtifactToBoard(gridIndex, item.artifact);

	if (success) {
		showToast(`⚡ Used ${item.name} on ${state.board[gridIndex].name}!`, "success");
		if (state.multiplayer && currentOpponentId) {
			socket.send(JSON.stringify({
				type: "use_item",
				to_id: currentOpponentId,
				payload: { grid_index: gridIndex, bonus: item.artifact }
			}));
		}
		hidePowerTooltip();
		syncUI();
	}
}

function openClubFoundry() {
	const overlay = document.createElement("div");
	overlay.id = "club-foundry-overlay";
	overlay.className = "overlay";
	overlay.innerHTML = `
		<div class="glass-panel" style="width: 450px; text-align: center;">
			<h2 style="color: var(--neon-purple);">CLUB FOUNDRY</h2>
			<p style="font-size: 0.9em; opacity: 0.8;">Founding a club costs a fortune (5,000 ).<br>Owners earn commissions from relative buffs sold in their territory.</p>
			
			<div class="flex-col gap-10 mt-20">
				<input type="text" id="foundry-club-name" class="glass-input w-full" placeholder="Enter Club Name (max 20 chars)" maxlength="20">
				
				<select id="foundry-shop-type" class="glass-input w-full">
					<option value="Elemental">Elemental Forge (Mood Buffs)</option>
					<option value="Tactical">Tactical Syndicate (Rule Mastery)</option>
					<option value="Vitality">Vitality Lab (Health/Loyalty)</option>
				</select>
				
				<select id="foundry-territory" class="glass-input w-full">
					<option value="the_lab">The Lab</option>
					<option value="casino">The Casino</option>
					<option value="arena_center">The Central Arena</option>
				</select>
			</div>

			<div class="mt-20 flex-row justify-center gap-15">
				<button class="outline" onclick="document.getElementById('club-foundry-overlay').remove()">CANCEL</button>
				<button id="foundry-submit-btn" onclick="submitClubFoundry()">FOUND CLUB (5,000 )</button>
			</div>
		</div>
	`;
	document.body.appendChild(overlay);
}

async function submitClubFoundry() {
	const name = document.getElementById("foundry-club-name").value.trim();
	const type = document.getElementById("foundry-shop-type").value;
	const territory = document.getElementById("foundry-territory").value;
	const btn = document.getElementById("foundry-submit-btn");

	if (!name) return showToast("Club name required", "error");
	if (!userAddress) return showToast("Connect wallet first", "error");

	btn.disabled = true;
	btn.innerText = "Processing...";

	try {
		const state = window.GetGameState();
		const network = state.network;
		const assetId = network === "VOI" ? CONFIG.VBV_ASSET_ID : CONFIG.AVOI_ASSET_ID;
		const amountMicro = 5000 * 1000000;

		showToast("Signing 5,000  Fortune Burn...", "info");

		let txid = "";
		// Reusing construction logic from registerForTournament
		if (network === "VOI") {
			const methodSelector = new Uint8Array([0x2b, 0x42, 0x6d, 0xec]); // transfer(address,uint256)
			const recipientAddr = algosdk.decodeAddress(CONFIG.VAULT_ADDRESS).publicKey;
			const amountArg = new Uint8Array(32);
			const amountBI = BigInt(amountMicro);
			for (let i = 0; i < 8; i++) amountArg[31 - i] = Number((amountBI >> BigInt(i * 8)) & 0xffn);

			const txObj = {
				from: userAddress, type: 'appl', appIndex: parseInt(assetId),
				appArgs: [methodSelector, recipientAddr, amountArg],
				note: new TextEncoder().encode(`FOUND_CLUB:`)
			};
			
			if (walletProvider === 'nautilus') {
				const signed = await window.algo.signTxn([{ txn: algosdk.encodeObj(txObj), signers: [userAddress] }]);
				const { txId } = await window.algo.sendRawTransaction(signed[0]);
				txid = txId;
			}
			// Additional providers would be handled here as in registerForTournament
		}

		if (!txid) throw new Error("Transaction cancelled or provider not supported.");

		socket.send(JSON.stringify({
			type: "create_club",
			payload: {
				name: name,
				type: type,
				territory_id: territory,
				txid: txid,
				network: network
			}
		}));

		document.getElementById("club-foundry-overlay").remove();
	} catch (err) {
		showToast(`Founding Failed: ${err.message}`, "error");
		btn.disabled = false;
		btn.innerText = "FOUND CLUB (5,000 )";
	}
}

function openTerritoryView(territoryId) {
	const club = Object.values(globalClubs).find(c => c.territory === territoryId);
	const overlay = document.createElement("div");
	overlay.id = "territory-view-overlay";
	overlay.className = "overlay";
	
	let header = `<h2>TERRITORY: ${territoryId.replace('_', ' ').toUpperCase()}</h2>`;
	let body = `<p style="opacity: 0.7;">This territory is currently unclaimed. Found a Club to take control!</p>`;

	if (club) {
		header = `
			<h2 style="color: var(--neon-cyan); margin-bottom: 5px;">${club.name}</h2>
			<div style="font-size: 0.8em; opacity: 0.6; margin-bottom: 15px;">Controlled by: ${club.owner_wallet.substring(0,8)}...</div>
			<div class="flex-row justify-center gap-15 mb-20">
				<div class="glass-panel p-10 m-0" style="min-width: 120px;">
					<div style="font-size: 0.7em; opacity: 0.5;">TREASURY</div>
					<b style="color: var(--neon-green);">${club.treasury.toFixed(2)} </b>
				</div>
				<div class="glass-panel p-10 m-0" style="min-width: 120px;">
					<div style="font-size: 0.7em; opacity: 0.5;">MOJO</div>
					<b style="color: var(--neon-purple);">${club.club_mojo}</b>
				</div>
			</div>
		`;

		const shopItems = {
			"Elemental": [
				{ id: "mood_catalyst", name: "Mood Catalyst", price: 100, desc: "+50 Mood Bonus (3 Matches)" },
				{ id: "grounded_shield", name: "Grounded Shield", price: 250, desc: "Immunity to Mood Penalties (5 Matches)" }
			],
			"Tactical": [
				{ id: "rule_breaker", name: "Rule Breaker", price: 150, desc: "Force PLUS trigger (1 Match)" },
				{ id: "intel_report", name: "Intel Report", price: 300, desc: "See Opponent Hand (3 Matches)" }
			],
			"Vitality": [
				{ id: "stamina_stim", name: "Stamina Stim", price: 100, desc: "-20 Fatigue Immediately" },
				{ id: "loyalty_pledge", name: "Loyalty Pledge", price: 500, desc: "+10 Loyalty Immediately" }
			]
		};

		const items = shopItems[club.type] || [];
		body = `
			<div class="flex-col gap-10">
				${items.map(item => `
					<div class="glass-panel p-15 m-0 flex-row justify-between align-center">
						<div style="text-align: left;">
							<b style="color: var(--neon-cyan);">${item.name}</b>
							<div style="font-size: 0.8em; opacity: 0.6;">${item.desc}</div>
						</div>
						<button class="outline" style="min-width: 100px; padding: 8px;" onclick="buyClubItem('${club.id}', '${item.id}', ${item.price}, '')">
							${item.price} 
						</button>
					</div>
				`).join('')}
			</div>
		`;
	}

	overlay.innerHTML = `
		<div class="glass-panel" style="width: 500px; text-align: center;">
			
			
			<div class="mt-20">
				<button class="outline" onclick="document.getElementById('territory-view-overlay').remove()">CLOSE MAP</button>
				${!club ? `<button onclick="document.getElementById('territory-view-overlay').remove(); openClubFoundry()">FOUND CLUB</button>` : ''}
			</div>
		</div>
	`;
	document.body.appendChild(overlay);
}

async function buyClubItem(clubId, itemId, price, territoryId) {
	if (!userAddress) return showToast("Connect wallet first", "error");
	
	try {
		const state = window.GetGameState();
		showToast(`Purchasing  for  ...`, "info");
		
		// In a full implementation, this would involve an ARC-200/ASA transfer to the Club Owner's address
		// For now, we simulate the economic signal to the server
		socket.send(JSON.stringify({
			type: "purchase_item",
			payload: {
				item_id: itemId,
				territory_id: territoryId,
				price: price * 1000000 // Convert to micro-units
			}
		}));

		// If it's a Vitality item, apply it immediately to the local engine
		if (itemId === "stamina_stim") {
			showToast("⚡ Fatigue reduced! Your cards are feeling refreshed.", "success");
		}

		document.getElementById("territory-view-overlay")?.remove();
	} catch (err) {
		showToast(`Purchase Failed: ${err.message}`, "error");
	}
}

function openWorldMap() {
	const overlay = document.createElement("div");
	overlay.id = "world-map-overlay";
	overlay.className = "overlay";
	
	// Mapping 0-8 to territory names for the grid
	const territoryMap = [
		{ id: "the_lab", name: "The Lab", icon: "🧪" },
		{ id: "north_district", name: "North Gate", icon: "⛩️" },
		{ id: "the_archive", name: "The Archive", icon: "📜" },
		{ id: "west_port", name: "West Port", icon: "⚓" },
		{ id: "arena_center", name: "Arena Center", icon: "⚔️" },
		{ id: "east_gate", name: "East Gate", icon: "🏯" },
		{ id: "south_slums", name: "The Slums", icon: "🏚️" },
		{ id: "casino", name: "The Casino", icon: "🎰" },
		{ id: "data_haven", name: "Data Haven", icon: "💾" }
	];

	let tilesHTML = "";
	territoryMap.forEach(t => {
		const club = Object.values(globalClubs).find(c => c.territory === t.id);
		const isOwned = !!club; // Check if any club owns this territory
		const isGovernorControlled = isOwned && club.region_name; // Check if the owning club is a Governor

		let tileClasses = "map-tile-3d";
		if (isOwned) tileClasses += " owned";
		if (isGovernorControlled) tileClasses += " governor-controlled"; // Add new class for Governor
		
		tilesHTML += `
			<div class="" onclick="event.stopPropagation(); document.getElementById('world-map-overlay').remove(); openTerritoryView('${t.id}')">
				<div class="tile-label">
					<div style="font-size: 24px; margin-bottom: 5px;">${t.icon}</div>
					<div>${t.name.toUpperCase()}</div>
					${isOwned ? `<div style="color: var(--neon-purple); margin-top: 5px; font-size: 8px;">[ ${club.name} ]</div>` : '<div style="opacity: 0.4; margin-top: 5px; font-size: 8px;">UNCLAIMED</div>'}
					${isGovernorControlled ? `<div style="color: var(--neon-cyan); font-size: 7px; font-weight: bold; margin-top: 2px;">GOVERNOR</div>` : ''}
				</div>
			</div>
		`;
	});

	overlay.innerHTML = `
		<div class="glass-panel" style="width: 80%; max-width: 900px; text-align: center; background: rgba(0,0,0,0.8);">
			<h1 style="font-size: 2em; margin-bottom: 10px;">NEON TOPOGRAPHY</h1>
			<p style="opacity: 0.6; font-size: 0.9em;">Tactical ownership map of the Arena. Select a district to visit shops or found a Club.</p>
			
			<div class="map-perspective-container">
				<div class="map-grid-3d">
					
				</div>
			</div>

			<div class="flex-row justify-center gap-20 mt-20">
				<div class="flex-row gap-5"><div style="width: 12px; height: 12px; background: rgba(0, 242, 254, 0.1); border: 1px solid var(--glass-border);"></div> <span style="font-size: 10px;">NEUTRAL</span></div>
				<div class="flex-row gap-5"><div style="width: 12px; height: 12px; background: rgba(155, 81, 224, 0.2); border: 1px solid var(--neon-purple);"></div> <span style="font-size: 10px;">CLUB CONTROLLED</span></div>
			</div>

			<button class="outline mt-20" onclick="document.getElementById('world-map-overlay').remove()">RETURN TO LOBBY</button>
		</div>
	`;

	document.body.appendChild(overlay);
}

function openCourthouse() {
	const state = window.GetGameState();
	const wanted = state.wanted_level || 0;
	if (wanted <= 0) return;

	const fine = wanted * 100;
	const overlay = document.createElement("div");
	overlay.id = "courthouse-overlay";
	overlay.className = "overlay";
	overlay.innerHTML = `
		<div class="glass-panel courthouse-panel">
			<h2 class="text-error" style="letter-spacing: 3px;">ARENA COURTHOUSE</h2>
			<p style="font-size: 0.9em; opacity: 0.8;">The High Council has flagged you for criminal activities.<br>Infamy Status: <b>LEVEL </b></p>
			
			<div class="glass-panel fine-display-box">
				<div style="font-size: 0.75em; opacity: 0.7;">REHABILITATION FINE</div>
				<b class="fine-amount"> </b>
				<div style="font-size: 0.7em; opacity: 0.5; margin-top: 5px;">(100  per Wanted point)</div>
			</div>

			<p style="font-size: 0.8em; opacity: 0.6; padding: 0 20px;">Settling your debt to society will clear your Wanted Level and restore your cards to peak combat performance.</p>

			<div class="mt-20 flex-row justify-center gap-15">
				<button class="outline" onclick="document.getElementById('courthouse-overlay').remove()">LURK IN SHADOWS</button>
				<button id="courthouse-pay-btn" style="background: linear-gradient(45deg, #ff4b4b, #ff0844);" onclick="submitCourthouseFine()">PAY FINE & CLEAR NAME</button>
			</div>
		</div>
	`;
	document.body.appendChild(overlay);
}

async function submitCourthouseFine() {
	const state = window.GetGameState();
	const wanted = state.wanted_level || 0;
	if (wanted <= 0) return;

	const btn = document.getElementById("courthouse-pay-btn");
	const amountMicro = wanted * 100 * 1000000;
	const network = state.network;
	const assetId = network === "VOI" ? CONFIG.VBV_ASSET_ID : CONFIG.AVOI_ASSET_ID;

	btn.disabled = true;
	btn.innerText = "PROCESSING...";

	try {
		showToast(`⚖️ Signing ${wanted * 100}  Fine...`, "info");
		let txid = "";
		let txObj = null;

		if (network === "VOI") {
			const methodSelector = new Uint8Array([0x2b, 0x42, 0x6d, 0xec]); // transfer(address,uint256)
			const recipientAddr = algosdk.decodeAddress(CONFIG.VAULT_ADDRESS).publicKey;
			const amountArg = new Uint8Array(32);
			const amountBI = BigInt(amountMicro);
			for (let i = 0; i < 8; i++) amountArg[31 - i] = Number((amountBI >> BigInt(i * 8)) & 0xffn);

			txObj = {
				from: userAddress, type: 'appl', appIndex: parseInt(assetId),
				appArgs: [methodSelector, recipientAddr, amountArg],
				note: new TextEncoder().encode(`ARENA_COURTHOUSE_FINE:`)
			};
		} else {
			txObj = {
				from: userAddress, to: CONFIG.VAULT_ADDRESS, type: 'axfer',
				assetIndex: parseInt(assetId), amount: amountMicro,
				note: new TextEncoder().encode(`ARENA_COURTHOUSE_FINE:`)
			};
		}

		if (walletProvider === 'nautilus') {
			const signed = await window.algo.signTxn([{ txn: algosdk.encodeObj(txObj), signers: [userAddress] }]);
			const { txId } = await window.algo.sendRawTransaction(signed[0]);
			txid = txId;
		} else if (walletProvider === 'kibisis') {
			const txnB64 = btoa(String.fromCharCode(...algosdk.encodeObj(txObj)));
			const signed = await window.kibisis.signTxns([{ txn: txnB64 }]);
			const { txId } = await window.kibisis.pushTxns(signed);
			txid = txId;
		} else if (walletProvider === 'walletconnect' && signClient) {
			const sessions = signClient.session.getAll();
			const chainId = network === "VOI" ? CONFIG.VOI_CHAIN_ID : CONFIG.ALGO_CHAIN_ID;
			const txnB64 = btoa(String.fromCharCode(...algosdk.encodeObj(txObj)));
			const response = await signClient.request({
				topic: sessions[0].topic, chainId: chainId,
				request: { method: "algo_signTxn", params: [[{ txn: txnB64, signers: [userAddress] }]] }
			});
			const signedTxnBytes = new Uint8Array(atob(response[0]).split("").map(c => c.charCodeAt(0)));
			const netCfg = getNetworkConfig(network);
			const client = new algosdk.Algodv2("", netCfg.node_url, "");
			const { txId } = await client.sendRawTransaction(signedTxnBytes).do();
			txid = txId;
		}

		if (!txid) throw new Error("Transaction cancelled or failed.");

		const response = await fetch(`${CONFIG.API_BASE}/api/courthouse/reset`, {
			method: "POST",
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ wallet: userAddress, txid: txid, network: network })
		});

		if (response.ok) {
			const result = await response.json();
			showToast(`⚖️ ${result.message}`, "success");
			document.getElementById("courthouse-overlay")?.remove();
			if (window.SyncOpponentWanted) window.SyncOpponentWanted(myPlayerIndex, 0);
			syncUI();
		} else {
			const err = await response.text();
			showToast(`❌ Courthouse Error: `, "error");
		}
	} catch (err) {
		showToast(`Fine Payment Failed: ${err.message}`, "error");
	} finally {
		btn.disabled = false;
		btn.innerText = "PAY FINE & CLEAR NAME";
	}
}

async function openPortfolioView(initialTab = 'portfolio') {
	const state = window.GetGameState();
	const overlay = document.createElement("div");
	const myJailedCards = state.jailed_cards || {};
	const myKidnappedCards = state.kidnapped_cards || {};
	const myHeldHostageCards = state.held_hostage_cards || {};
	overlay.id = "portfolio-view-overlay";
	overlay.className = "overlay";
	
	overlay.innerHTML = `
		<div class="glass-panel" style="width: 500px; text-align: center;">
			<h2 style="color: var(--neon-cyan);">ENTITY PORTFOLIO</h2>
			<div class="flex-row justify-center gap-10 mt-10 mb-20">
				<button id="tab-holdings" class="tab-btn ${initialTab === 'portfolio' ? 'active' : ''}" onclick="switchPortfolioTab('portfolio')">📈 HOLDINGS</button>
				<button id="tab-jailed" class="tab-btn ${initialTab === 'jailed' ? 'active' : ''}" onclick="switchPortfolioTab('jailed')">⛓️ JAILED (${Object.keys(myJailedCards).length})</button>
				<button id="tab-kidnapped" class="tab-btn ${initialTab === 'kidnapped' ? 'active' : ''}" onclick="switchPortfolioTab('kidnapped')">😈 KIDNAPPED (${Object.keys(myKidnappedCards).length})</button>
				<button id="tab-hostage" class="tab-btn ${initialTab === 'hostage' ? 'active' : ''}" onclick="switchPortfolioTab('hostage')">🛑 HOSTAGE (${Object.keys(myHeldHostageCards).length})</button>
			</div>
			
			<div id="portfolio-content-area" class="flex-col gap-10" style="max-height: 400px; overflow-y: auto; padding-right: 5px;">
				<!-- Content injected by switchPortfolioTab -->
			</div>

			<button class="outline mt-20 w-full" onclick="document.getElementById('portfolio-view-overlay').remove()">CLOSE</button>
		</div>
	`;

	document.body.appendChild(overlay);
	switchPortfolioTab(initialTab);
}

async function switchPortfolioTab(tab) {
	const container = document.getElementById("portfolio-content-area");
	const holdingsBtn = document.getElementById("tab-holdings");
	const jailedBtn = document.getElementById("tab-jailed");
	const kidnappedBtn = document.getElementById("tab-kidnapped");
	const hostageBtn = document.getElementById("tab-hostage");
	const state = window.GetGameState();

	holdingsBtn.classList.toggle("active", tab === 'portfolio');
	jailedBtn.classList.toggle("active", tab === 'jailed');
	if (kidnappedBtn) kidnappedBtn.classList.toggle("active", tab === 'kidnapped');
	if (hostageBtn) hostageBtn.classList.toggle("active", tab === 'hostage');
	container.innerHTML = `<div style="padding: 20px; opacity: 0.5;">Loading details...</div>`;

	if (tab === 'portfolio') {
		const portfolio = state.portfolio || {};
		const entries = Object.entries(portfolio);
		let html = "";
		let totalMarketValue = 0;

		if (entries.length === 0) {
			html = `<div style="padding: 40px; opacity: 0.5;">No active investments found.</div>`;
		} else {
			// Batch resolve Envoi names for all portfolio keys (wallets)
			const walletsToResolve = entries.map(([w]) => w);
			await Promise.all(walletsToResolve.map(w => resolveEnvoiName(w)));

			entries.forEach(([id, amount]) => {
				if (amount <= 0) return;
				// id is now the persistent wallet address
				const p = lastLobbyPlayers.find(pl => pl.wallet && pl.wallet.toLowerCase() === id.toLowerCase());
				const price = p ? ((p.wins * 10) + (p.reputation / 2) + 100) : 100;
				const marketValue = amount * price;
				totalMarketValue += marketValue;
				const displayName = getCachedEnvoiName(id);
				html += `
					<div class="player-item" style="padding: 15px;">
						<div style="text-align: left;">
							<b style="color: var(--neon-cyan);"></b>
							<div style="font-size: 0.75em; opacity: 0.6;">Holding: ${amount.toFixed(2)} Shares</div>
						</div>
						<div style="text-align: right;">
							<div style="color: var(--neon-green); font-weight: bold;">${marketValue.toFixed(2)} </div>
							<button class="outline" style="font-size: 9px; padding: 4px 8px; border-color: #ff4b4b; color: #ff4b4b;" 
									onclick="tradeShares('', 'sell', )">SELL ALL</button>
						</div>
					</div>`;
			});
			html += `
				<div style="margin-top: 10px; padding-top: 10px; border-top: 1px solid var(--glass-border); text-align: right;">
					<small style="opacity: 0.5;">EST. LIQUIDITY VALUE</small><br>
					<b style="color: var(--neon-green); font-size: 1.2em;">${totalMarketValue.toFixed(2)} </b>
				</div>`;
		}
		container.innerHTML = html;
	} else if (tab === 'jailed') {
		const jailed = state.jailed_cards || {};
		const cardIds = Object.keys(jailed);
		if (cardIds.length === 0) {
			container.innerHTML = `<div style="padding: 40px; opacity: 0.5;">No cards currently in custody.</div>`;
			return;
		}

		let html = "";
		for (const cardId of cardIds) {
			const clubId = jailed[cardId];
			const club = globalClubs[clubId] || { name: "Underworld Entity" };
			html += `
				<div class="player-item" style="padding: 15px; border-color: #ff4b4b;">
					<div style="text-align: left;">
						<b style="color: #ff4b4b;">ID: #</b>
						<div style="font-size: 0.75em; opacity: 0.6;">Held by: ${club.name}</div>
					</div>
					<div style="text-align: right;">
						<button class="outline" style="font-size: 10px; padding: 6px 12px; border-color: var(--neon-green); color: var(--neon-green);" 
								onclick="initiateBail(, '')">PAY BAIL (200 )</button>
					</div>
				</div>`;
		}
		container.innerHTML = html;
	} else if (tab === 'kidnapped') {
		const kidnapped = state.kidnapped_cards || {};
		const cardIds = Object.keys(kidnapped);
		if (cardIds.length === 0) {
			container.innerHTML = `<div style="padding: 40px; opacity: 0.5;">No kidnapped cards at the moment.</div>`;
			return;
		}

		let html = "";
		for (const cardId of cardIds) {
			const victimWallet = kidnapped[cardId] || "Unknown";
			html += `
				<div class="player-item" style="padding: 15px; border-color: #ffa500;">
					<div style="text-align: left;">
						<b style="color: #ffa500;">ID: #</b>
						<div style="font-size: 0.75em; opacity: 0.6;">Victim Wallet: </div>
					</div>
					<div style="text-align: right;">
						<button class="outline" style="font-size: 9px; padding: 4px 8px; border-color: #ffd700; color: #ffd700;" onclick="releaseHostage()">RELEASE HOSTAGE</button>
					</div>
				</div>`;
		}
		container.innerHTML = html;
	} else if (tab === 'hostage') {
		const heldHostage = state.held_hostage_cards || {};
		const cardIds = Object.keys(heldHostage);
		if (cardIds.length === 0) {
			container.innerHTML = `<div style="padding: 40px; opacity: 0.5;">No cards currently held hostage.</div>`;
			return;
		}

		let html = "";
		for (const cardId of cardIds) {
			const perpWallet = heldHostage[cardId] || "Unknown";
			html += `
				<div class="player-item" style="padding: 15px; border-color: #ffd700;">
					<div style="text-align: left;">
						<b style="color: #ffd700;">ID: #</b>
						<div style="font-size: 0.75em; opacity: 0.6;">Kidnapper: </div>
					</div>
					<div style="text-align: right;">
						<button class="outline" style="font-size: 9px; padding: 4px 8px; border-color: #ff4b4b; color: #ff4b4b;" onclick="payRansom(, '')">PAY RANSOM</button>
					</div>
				</div>`;
		}
		html += `<div style="margin-top: 10px; padding: 12px; border: 1px dashed var(--glass-border); color: #ffd700; font-size: 0.85em;">Ransom amount will be requested after you initiate payment.</div>`;
		container.innerHTML = html;
	} else {
		container.innerHTML = `<div style="padding: 40px; opacity: 0.5;">No details available for this tab.</div>`;
	}
}

async function initiateBail(cardId, clubId) {
	if (!userAddress) return;
	if (!confirm(`Are you sure you want to pay 200  to release Card #?`)) return;

	try {
		const state = window.GetGameState();
		const network = state.network;
		const bailAmountMicro = 200 * 1000000;
		const assetId = network === "VOI" ? CONFIG.VBV_ASSET_ID : CONFIG.AVOI_ASSET_ID;

		showToast(`⚖️ Signing Bail Payment for Card #...`, "info");
		
		let txid = "";
		// Construction logic mirroring courthouse fine
		if (network === "VOI") {
			const methodSelector = new Uint8Array([0x2b, 0x42, 0x6d, 0xec]); 
			const recipientAddr = algosdk.decodeAddress(CONFIG.VAULT_ADDRESS).publicKey;
			const amountArg = new Uint8Array(32);
			const amountBI = BigInt(bailAmountMicro);
			for (let i = 0; i < 8; i++) amountArg[31 - i] = Number((amountBI >> BigInt(i * 8)) & 0xffn);

			const txObj = {
				from: userAddress, type: 'appl', appIndex: parseInt(assetId),
				appArgs: [methodSelector, recipientAddr, amountArg],
				note: new TextEncoder().encode(`BAIL_PAYMENT:`)
			};
			
			if (walletProvider === 'nautilus') {
				const signed = await window.algo.signTxn([{ txn: algosdk.encodeObj(txObj), signers: [userAddress] }]);
				const { txId } = await window.algo.sendRawTransaction(signed[0]);
				txid = txId;
			} else if (walletProvider === 'walletconnect') {
				const sessions = signClient.session.getAll();
				const response = await signClient.request({
					topic: sessions[0].topic, chainId: CONFIG.VOI_CHAIN_ID,
					request: { method: "algo_signTxn", params: [[{ txn: btoa(String.fromCharCode(...algosdk.encodeObj(txObj))), signers: [userAddress] }]] }
				});
				const signedTxnBytes = new Uint8Array(atob(response[0]).split("").map(c => c.charCodeAt(0)));
				const netCfg = availableNetworks["Voi Mainnet"];
				const client = new algosdk.Algodv2("", netCfg.node_url, "");
				const { txId: broadcastId } = await client.sendRawTransaction(signedTxnBytes).do();
				txid = broadcastId;
			}
		}

		if (!txid) throw new Error("Transaction verification failed.");

		socket.send(JSON.stringify({
			type: "bail_card",
			payload: {
				card_id: parseInt(cardId),
				club_id: clubId,
				txid: txid,
				network: network
			}
		}));

		document.getElementById("portfolio-view-overlay")?.remove();
	} catch (err) {
		showToast(`Bail Request Failed: ${err.message}`, "error");
	}
	
	document.body.appendChild(overlay);
}

function openSecuritySentry() {
	const state = window.GetGameState();
	const club = globalClubs[state.employer_id];
	if (!club) return;

	const overlay = document.createElement("div");
	overlay.id = "security-sentry-overlay";
	overlay.className = "overlay";

	// Heuristic: Traps are items with "tripwire", "sentry", or "dog" in ID
	const availableTraps = [
		{ id: "tripwire", name: "Laser Tripwire", desc: "+10% Heist Failure" },
		{ id: "sentry_turret", name: "Sentry Turret", desc: "+25% Heist Failure" },
		{ id: "guard_dog", name: "Bio-Guard Dog", desc: "Forces Jail on Failure" }
	];

	const activeTraps = Object.entries(club.active_buffs || {})
		.filter(([key]) => key.startsWith("TRAP_"));

	overlay.innerHTML = `
		<div class="glass-panel" style="width: 550px; text-align: center; border-color: var(--neon-cyan);">
			<h2 style="color: var(--neon-cyan); letter-spacing: 2px;">🛡️ SECURITY SENTRY: ${club.name.toUpperCase()}</h2>
			<p style="font-size: 0.8em; opacity: 0.7; margin-bottom: 20px;">Deploy tactical hardware to protect the Treasury from heisters.</p>
			
			<div style="text-align: left; margin-bottom: 20px;">
				<small style="color: var(--neon-cyan); font-weight: bold; opacity: 0.5;">ACTIVE DEFENSES (${activeTraps.length}/3)</small>
				<div class="flex-col gap-5 mt-5">
					${activeTraps.length === 0 ? '<div style="opacity: 0.3; font-style: italic;">No active traps detected.</div>' : 
					  activeTraps.map(([id, type]) => `
						<div class="player-item" style="padding: 8px 12px; border-color: var(--neon-green);">
							<span>🛰️ ${type.toUpperCase()}</span>
							<span style="color: var(--neon-green); font-size: 10px;">ONLINE</span>
						</div>
					  `).join('')}
				</div>
			</div>

			<div style="text-align: left;">
				<small style="color: var(--neon-cyan); font-weight: bold; opacity: 0.5;">AVAILABLE HARDWARE</small>
				<div class="flex-col gap-10 mt-5">
					${availableTraps.map(trap => {
						const count = state.inventory[trap.id] || 0;
						return `
							<div class="glass-panel p-10 m-0 flex-row justify-between align-center">
								<div>
									<b>${trap.name}</b>
									<div style="font-size: 0.75em; opacity: 0.6;">${trap.desc}</div>
								</div>
								<div class="flex-row align-center gap-10">
									<span style="font-size: 11px; opacity: 0.8;">Owned: </span>
									<button class="outline" style="font-size: 10px; padding: 5px 15px;" 
											${count === 0 || activeTraps.length >= 3 ? 'disabled' : ''} 
											onclick="deployTrap('${trap.id}')">DEPLOY</button>
								</div>
							</div>
						`;
					}).join('')}
				</div>
			</div>

			<button class="outline mt-20 w-full" onclick="document.getElementById('security-sentry-overlay').remove()">CLOSE TERMINAL</button>
		</div>
	`;
	document.body.appendChild(overlay);
}

function deployTrap(trapId) {
	if (!socket || socket.readyState !== WebSocket.OPEN) return;
	
	showToast(`🛰️ Deploying ${trapId.replace('_', ' ')}...`, "info");
	socket.send(JSON.stringify({
		type: "use_item",
		payload: {
			item_id: trapId
		}
	}));
	document.getElementById("security-sentry-overlay")?.remove();
}

async function openBountyBoard() {
	const state = window.GetGameState();
	const myWanted = state.wanted_level || 0;
	const isHunter = myWanted <= 2;
	
	const overlay = document.createElement("div");
	overlay.id = "bounty-board-overlay";
	overlay.className = "overlay";
	
	const outlaws = lastLobbyPlayers.filter(p => (p.wanted_level || 0) >= 10);
	
	let targetsHtml = "";
	if (outlaws.length === 0) {
		targetsHtml = `<div style="padding: 40px; opacity: 0.5;">No active bounties in this sector.</div>`;
	} else {
		// Pre-resolve envoi names
		const wallets = outlaws.map(p => p.wallet);
		await Promise.all(wallets.map(w => resolveEnvoiName(w)));

		outlaws.forEach(p => {
			const name = getCachedEnvoiName(p.wallet);
			const bounty = p.wanted_level * 50;
			const isMe = p.id === myClientId;
			
			targetsHtml += `
				<div class="player-item" style="padding: 15px; border-color: #ffd700;">
					<div style="text-align: left;">
						<b style="color: #ffd700;"></b>
						<div style="font-size: 0.75em; opacity: 0.6;">WANTED LEVEL: ${p.wanted_level}</div>
					</div>
					<div style="text-align: right;">
						<div style="color: var(--neon-green); font-weight: bold;"> </div>
						${isHunter && !isMe ? `<button class="outline" style="font-size: 10px; padding: 6px 12px; border-color: #ffd700; color: #ffd700;" onclick="document.getElementById('bounty-board-overlay').remove(); sendChallenge('${p.id}')">HUNT TARGET</button>` : ''}
						${isMe ? `<span style="font-size: 10px; color: #ff4b4b;">YOU ARE THE TARGET</span>` : ''}
					</div>
				</div>`;
		});
	}

	overlay.innerHTML = `
		<div class="glass-panel" style="width: 500px; text-align: center; border-color: #ffd700;">
			<h2 style="color: #ffd700; letter-spacing: 3px;">🎯 BOUNTY BOARD</h2>
			<p style="font-size: 0.8em; opacity: 0.7; margin-bottom: 20px;">High-infamy outlaws currently in the lobby. Hunters (Wanted ≤ 2) earn 50  per Wanted point on victory.</p>
			<div class="flex-col gap-10" style="max-height: 400px; overflow-y: auto; padding-right: 5px;"></div>
			<button class="outline mt-20 w-full" onclick="document.getElementById('bounty-board-overlay').remove()">CLOSE BOARD</button>
		</div>`;
	document.body.appendChild(overlay);
}

async function openBlackMarket() {
	const state = window.GetGameState();
	const wanted = state.wanted_level || 0;
	const cunning = state.cunning || 0;

	if (wanted < 5 || cunning < 10) {
		showToast("❌ Access Denied: Black Market requires Wanted Level 5+ and Cunning 10+.", "error");
		return;
	}

	const overlay = document.createElement("div");
	overlay.id = "black-market-overlay";
	overlay.className = "overlay";

	let html = `
		<div class="glass-panel" style="width: 600px; text-align: center; border-color: #ff4b4b;">
			<h2 style="color: #ff4b4b; letter-spacing: 3px;">BLACK MARKET</h2>
			<p style="font-size: 0.8em; opacity: 0.7; margin-bottom: 20px;">Liquidated assets from defaulted loans. High risk, high reward.</p>
			<div class="flex-col gap-10" style="max-height: 400px; overflow-y: auto; padding-right: 5px;">
	`;

	try {
		const response = await fetch(`${CONFIG.API_BASE}/api/black-market?wallet=`);
		if (!response.ok) {
			const errorText = await response.text();
			throw new Error(errorText);
		}
		const blackMarketItems = await response.json();

		if (blackMarketItems.length === 0) {
			html += `<div style="padding: 40px; opacity: 0.5;">No hot items currently available. Check back later.</div>`;
		} else {
			// Pre-resolve envoi names for all borrowers
			const borrowerWallets = new Set(blackMarketItems.map(item => item.borrower_wallet));
			await Promise.all(Array.from(borrowerWallets).map(w => resolveEnvoiName(w)));

			for (const item of blackMarketItems) {
				const cardName = item.collateral_bundle.card_id ? `CARD-${item.collateral_bundle.card_id}` : 'N/A';
				const weaponName = item.collateral_bundle.weapon_id || 'N/A';
				const faceplateName = item.collateral_bundle.faceplate_id || 'N/A';
				const borrowerName = getCachedEnvoiName(item.borrower_wallet);

				// Scavenger price is 75% of the original repayment amount
				const scavengePrice = (item.repayment_amount * 0.75) / 1000000; // Convert micro-units to VBV

				html += `
					<div class="player-item" style="padding: 15px; border-color: #ff4b4b;">
						<div style="text-align: left;">
							<b style="color: var(--neon-cyan);">Collateral from </b>
							<div style="font-size: 0.75em; opacity: 0.6;">
								Card:  <br>
								Weapon:  <br>
								Faceplate: 
							</div>
						</div>
						<div style="text-align: right;">
							<div style="color: var(--neon-green); font-weight: bold;">${scavengePrice.toFixed(2)} </div>
							<button class="outline" style="font-size: 9px; padding: 4px 8px; border-color: #ff4b4b; color: #ff4b4b;" 
									onclick="buyBlackMarketItem('${item.id}', )">BUY (RISKY)</button>
						</div>
					</div>
				`;
			}
		}
	} catch (err) {
		showToast(`❌ Black Market Access Failed: ${err.message}`, "error");
		html += `<div style="padding: 40px; opacity: 0.5; color: #ff4b4b;">Error loading Black Market: ${err.message}</div>`;
	}

	html += `
			</div>
			<button class="outline mt-20" onclick="document.getElementById('black-market-overlay').remove()">CLOSE</button>
		</div>
	`;

	overlay.innerHTML = html;
	document.body.appendChild(overlay);
}

async function buyBlackMarketItem(loanId, price) {
	if (!userAddress) return showToast("Connect wallet first", "error");
	if (!confirm(`Are you sure you want to buy this item for ${price.toFixed(2)} ? This will increase your Wanted Level.`)) return;

	try {
		const state = window.GetGameState();
		const network = state.network;

		const response = await fetch(`${CONFIG.API_BASE}/api/black-market/buy`, {
			method: "POST",
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ wallet: userAddress, loan_id: loanId, network: network })
		});

		if (response.ok) {
			const result = await response.json();
			showToast(`🏴‍☠️ ${result.message}`, "success");
			document.getElementById("black-market-overlay")?.remove();
			syncUI();
		} else {
			const err = await response.text();
			showToast(`❌ Black Market Purchase Failed: `, "error");
		}
	} catch (err) {
		showToast(`Purchase Failed: ${err.message}`, "error");
	}
}

async function openRumorMill() {
	const state = window.GetGameState();
	const playerRewards = state.rewards[CONFIG.VBV_ASSET_ID] || 0;
	const rumorCost = 500; // Matches server-side cost

	if (playerRewards < rumorCost) {
		showToast(`❌ Insufficient . Spreading a rumor costs  .`, "error");
		return;
	}

	const overlay = document.createElement("div");
	overlay.id = "rumor-mill-overlay";
	overlay.className = "overlay";

	let targetsHtml = '';
	if (lastLobbyPlayers.length === 0) {
		targetsHtml = `<div style="padding: 20px; opacity: 0.5;">No other players in the lobby to spread rumors about.</div>`;
	} else {
		// Filter out self
		const otherPlayers = lastLobbyPlayers.filter(p => p.id !== myClientId);
		if (otherPlayers.length === 0) {
			targetsHtml = `<div style="padding: 20px; opacity: 0.5;">No other players in the lobby to spread rumors about.</div>`;
		} else {
			// Pre-resolve envoi names for all targets
			const targetWallets = new Set(otherPlayers.map(p => p.wallet));
			await Promise.all(Array.from(targetWallets).map(w => resolveEnvoiName(w)));

			targetsHtml = otherPlayers.map(p => {
				const targetName = getCachedEnvoiName(p.wallet);
				return `
					<div class="player-item" style="padding: 10px; border-color: var(--glass-border);">
						<div style="text-align: left;">
							<b style="color: var(--neon-cyan);"></b>
							<div style="font-size: 0.75em; opacity: 0.6;">${p.reputation} REP | ${p.wins} WINS</div>
						</div>
						<div class="flex-row gap-5">
							<button class="outline" style="font-size: 9px; padding: 4px 8px; border-color: var(--neon-green); color: var(--neon-green);" 
									onclick="spreadRumor('${p.wallet}', 'positive', 1.1, 60)">+ POSITIVE</button>
							<button class="outline" style="font-size: 9px; padding: 4px 8px; border-color: #ff4b4b; color: #ff4b4b;" 
									onclick="spreadRumor('${p.wallet}', 'negative', 0.9, 60)">- NEGATIVE</button>
						</div>
					</div>
				`;
			}).join('');
		}
	}

	overlay.innerHTML = `
		<div class="glass-panel" style="width: 600px; text-align: center;">
			<h2 style="color: var(--neon-green); letter-spacing: 3px;">RUMOR MILL</h2>
			<p style="font-size: 0.8em; opacity: 0.7; margin-bottom: 20px;">Influence market sentiment. Cost: <b style="color: var(--neon-green);"> </b></p>
			<div class="flex-col gap-10" style="max-height: 400px; overflow-y: auto; padding-right: 5px;">
				
			</div>
			<button class="outline mt-20" onclick="document.getElementById('rumor-mill-overlay').remove()">CLOSE</button>
		</div>
	`;

	document.body.appendChild(overlay);
}

async function spreadRumor(targetWallet, type, strength, durationMinutes) {
	if (!socket || socket.readyState !== WebSocket.OPEN) return showToast("❌ Not connected to server.", "error");
	if (!userAddress) return showToast("❌ Connect wallet first.", "error");

	const rumorCost = 500; // Matches server-side cost
	const state = window.GetGameState();
	const playerRewards = state.rewards[CONFIG.VBV_ASSET_ID] || 0;

	if (playerRewards < rumorCost) {
		showToast(`❌ Insufficient . Spreading a rumor costs  .`, "error");
		return;
	}

	if (!confirm(`Are you sure you want to spread a  rumor about ${getCachedEnvoiName(targetWallet)} for  ?`)) return;

	try {
		showToast(`📢 Spreading  rumor about ${getCachedEnvoiName(targetWallet)}...`, "info");
		
		socket.send(JSON.stringify({
			type: "spread_rumor",
			payload: {
				target_wallet: targetWallet,
				type: type,
				strength: strength,
				duration_minutes: durationMinutes
			}
		}));
		
		document.getElementById("rumor-mill-overlay")?.remove();
	} catch (err) {
		showToast(`❌ Failed to spread rumor: ${err.message}`, "error");
	}
}

function openTrophyView() {
	const overlay = document.createElement("div");
	overlay.id = "trophy-view-overlay";
	overlay.className = "overlay";

	const state = window.GetGameState() || {};
	const unlocked = new Set(state.achievements || []);
	const trophyCatalog = [
		{ id: "FIRST_VICTORY", name: "First Victory", description: "Win your first match.", tier: 1 },
		{ id: "TOURNAMENT_CHAMPION", name: "Tournament Champion", description: "Win a tournament.", tier: 2 },
		{ id: "FIRST_HEIST", name: "First Heist", description: "Complete a successful Club heist.", tier: 1 },
		{ id: "OUTLAW_SLAYER", name: "Outlaw Slayer", description: "Defeat a high-infamy opponent.", tier: 2 },
		{ id: "ARENA_LEGEND", name: "Arena Legend", description: "Achieve legendary status in the arena.", tier: 3 },
		{ id: "REHABILITATED", name: "Rehabilitated", description: "Pay off your courthouse fine and reset wanted level.", tier: 2 },
		{ id: "GOVERNOR", name: "Governor", description: "Control 2+ territories as a club leader.", tier: 3 }
	];

	let trophiesHTML = "";
	trophyCatalog.forEach(trophy => {
		const hasUnlocked = unlocked.has(trophy.id);
		const glowClass = `trophy-badge tier-${trophy.tier}` + (hasUnlocked ? "" : " locked");
		trophiesHTML += `
			<div class="" style="opacity: ${hasUnlocked ? 1 : 0.35};">
				<div style="font-size: 2em;">${hasUnlocked ? '🏆' : '🔒'}</div>
				<div style="font-size: 0.85em; margin-top: 5px;">${trophy.name}</div>
				<div style="font-size: 0.65em; opacity: 0.7;">${trophy.description}</div>
			</div>
		`;
	});

	if (trophyCatalog.length === 0) {
		trophiesHTML = `<div style="padding: 40px; opacity: 0.5;">No achievements have been loaded yet.</div>`;
	}

	overlay.innerHTML = `
		<div class="glass-panel" style="width: 600px; text-align: center;">
			<h2 style="color: var(--neon-cyan);">ACHIEVEMENT TROPHIES</h2>
			<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(120px, 1fr)); gap: 1rem; margin: 2rem 0;">
				
			</div>
			<button class="outline mt-20 w-full" onclick="document.getElementById('trophy-view-overlay').remove()">CLOSE</button>
		</div>
	`;

	document.body.appendChild(overlay);
}

function tradeShares(entityId, action, amount) {
	if (!socket || socket.readyState !== WebSocket.OPEN) return;

	socket.send(JSON.stringify({
		type: "trade_shares",
		payload: {
			entity_id: entityId,
			action: action,
			amount: amount
		}
	}));

	document.getElementById("portfolio-view-overlay")?.remove();
}

// Function to show the main game container and hide other overlays
function showMainGameContainer() {
	document.getElementById("main-game-container").classList.remove("hidden");
}

// Placeholder for setupCropEvents - assuming it's defined elsewhere or will be moved here
// --- Avatar Setup & Cropping Logic ---

async function refreshInventory() {
	if (!userAddress) return;
	
	const grid = document.getElementById("avatar-grid");
	const loader = document.getElementById("setup-loader");
	if (loader) loader.classList.remove("hidden");

	userNFTs = []; // Clear for aggregate fetch
	const state = window.GetGameState();
	
	// 1. Compile list of wallets to scan.
	// The primary userAddress is assumed to be on the currently selected game network.
	const primaryNetworkShortName = state.network; // e.g., "VOI" or "ALGO"
	const sources = [{ address: userAddress, chain: primaryNetworkShortName }];
	linkedWallets.forEach(w => sources.push(w));

	// 2. Fetch from all sources in parallel
	await Promise.all(sources.map(async (src) => {
		try {
			// Use Indexer URL from admin availableNetworks if available, otherwise fallback
			const networkConfig = getNetworkConfig(src.chain); // Use helper for consistency
			const baseUrl = networkConfig ? networkConfig.indexer_url : "";

			if (!baseUrl) {
				console.warn(`[FETCH] No indexer URL found for network ${src.chain}. Skipping NFT fetch for ${src.address}.`);
				return; // Skip if no base URL
			}

			if (src.chain === "SOL") {
				// Solana DAS API specific fetch via POST to NodeURL
				const solRes = await fetch(baseUrl, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ jsonrpc: "2.0", id: 1, method: "getAssetsByOwner", params: { ownerAddress: src.address, page: 1, limit: 50 }})
				});
				const solData = await solRes.json();
				if (solData.result && solData.result.items) userNFTs = [...userNFTs, ...solData.result.items];
				return;
			}

			const response = await fetch(`/tokens?owner=${src.address}`);
			if (!response.ok) {
				console.warn(`[FETCH] Indexer returned error for ${src.address}: ${response.status}`);
				return;
			}

			const data = await response.json();
			if (data.tokens) userNFTs = [...userNFTs, ...data.tokens];
		} catch (err) { console.warn(`[FETCH] Source ${src.address} failed:`, err); }
	}));

	renderAvatarGrid(userNFTs);
	updateLinkedWalletsUI();
	if (loader) loader.classList.add("hidden");
}

function renderAvatarGrid(nfts) {
	const grid = document.getElementById("avatar-grid");
	if (!grid) return;
	grid.innerHTML = "";
	
	// Filter out banned avatars
	nfts.forEach(nft => {
		let meta = {};
		try { meta = JSON.parse(nft.metadata || "{}"); } catch(e) {}
		const url = meta.image || "";
		if (!url) return;
		
		// Check if this URL is banned
		const state = window.GetGameState();
		const isBanned = state.banned_avatars && state.banned_avatars[url];
		const item = document.createElement("div"); // Create the element regardless
		item.className = "avatar-item";
		item.style.backgroundImage = `url()`;
		item.onclick = () => selectAvatar(url);
		grid.appendChild(item);
	});
}

function applyAvatarFilters() {
	const search = document.getElementById("avatar-search").value.toLowerCase();
	const sort = document.getElementById("avatar-sort").value;
	
	let filtered = userNFTs.filter(nft => {
		let meta = {};
		try { meta = JSON.parse(nft.metadata || "{}"); } catch(e) {}
		return (meta.name || "").toLowerCase().includes(search);
	});
	
	if (sort === "oldest") {
		filtered.sort((a, b) => a.mintRound - b.mintRound);
	} else if (sort === "newest") {
		filtered.sort((a, b) => b.mintRound - a.mintRound);
	}
	
	renderAvatarGrid(filtered);
}

// --- Arena Background Controller ---
function updateDynamicArenaFloor(state) {
	let texture = "var(--texture-solo)"; // Default AI/Solo

	if (state.phase === "TournamentLobby") {
		// Always show a tournament background in the tournament lobby
		texture = "var(--texture-tournament)";
	} else if (state.phase === "Active") {
		if (state.multiplayer) {
			if (state.tournament && state.tournament.active) {
				const currentRound = state.tournament.current_round;
				const participants = state.tournament.participants ? state.tournament.participants.length : 8;
				const maxRounds = Math.log2(participants); // 8 = 3 rounds, 16 = 4 rounds

				if (currentRound === maxRounds) {
					texture = "var(--texture-final)";
				} else if (currentRound === maxRounds - 1) {
					texture = "var(--texture-semi)";
				} else {
					texture = "var(--texture-tournament)";
				}
			} else {
				// Standard 2 Player Match (Challenge)
				texture = "var(--texture-challenge)";
			}
		}
	}

	// Apply to body background
	document.body.style.backgroundImage = `, radial-gradient(circle at top center, #1a0b2e, var(--bg-dark), #000000)`;
}

function selectAvatar(url) {
	const preview = document.getElementById("avatar-preview-section");
	const img = document.getElementById("crop-image");
	if (!preview || !img) return;
	
	currentAvatarUrl = url;
	img.src = url;
	
	// Pre-populate gloat from cache
	const cachedGloat = localStorage.getItem("vbabes_gloat_msg") || "";
	document.getElementById("gloat-message-input").value = cachedGloat;

	preview.classList.remove("hidden");
	// Calibration is handled by the img.onload listener in setupCropEvents
}

function setupCropEvents() {
	const frame = document.getElementById("crop-frame");
	const img = document.getElementById("crop-image");
	const slider = document.getElementById("zoom-slider");
	const zoomVal = document.getElementById("zoom-val");
	const confirmBtn = document.getElementById("confirm-avatar-btn");
	
	if (!frame || !img || !slider || !confirmBtn) return;
	if (isCropInitialized) return; // Prevent duplicate global listeners
	isCropInitialized = true;

	let isDragging = false;
	let startX, startY;

	const updateTransform = () => {
		img.style.transform = `translate(${cropState.x}px, ${cropState.y}px) scale(${cropState.zoom})`;
	};

	// ASPECT RATIO & INITIAL CALIBRATION: Ensure image covers the 220px circle frame
	img.onload = () => {
		const frameSize = 220; // Diameter of the circle
		const w = img.naturalWidth;
		const h = img.naturalHeight;

		// Calculate minimal scale to completely fill the frame (CSS 'cover' behavior)
		const scaleW = frameSize / w;
		const scaleH = frameSize / h;
		const baseScale = Math.max(scaleW, scaleH);

		// Initialize state variables for pan/zoom logic
		cropState.zoom = baseScale;
		cropState.x = (frameSize - (w * baseScale)) / 2;
		cropState.y = (frameSize - (h * baseScale)) / 2;

		// Sync UI Sliders
		slider.min = baseScale.toFixed(2);
		slider.max = (baseScale * 4).toFixed(2);
		slider.value = baseScale;
		if (zoomVal) zoomVal.innerText = "1.0x";
		
		updateTransform();
	};

	slider.oninput = () => {
		cropState.zoom = parseFloat(slider.value);
		const relZoom = cropState.zoom / parseFloat(slider.min);
		if (zoomVal) zoomVal.innerText = relZoom.toFixed(1) + "x";
		updateTransform();
	};

	frame.onmousedown = (e) => {
		if (e.button !== 0) return; // Only primary mouse button
		isDragging = true;
		startX = e.clientX - cropState.x;
		startY = e.clientY - cropState.y;
		frame.style.cursor = "grabbing";
	};

	window.addEventListener('mousemove', (e) => {
		if (!isDragging) return;
		cropState.x = e.clientX - startX;
		cropState.y = e.clientY - startY;
		updateTransform();
	});

	window.addEventListener('mouseup', () => {
		isDragging = false;
		if (frame) frame.style.cursor = "grab";
	});

	// Mobile Touch Support
	frame.ontouchstart = (e) => {
		isDragging = true;
		startX = e.touches[0].clientX - cropState.x;
		startY = e.touches[0].clientY - cropState.y;
	};
	frame.ontouchmove = (e) => {
		if (!isDragging) return;
		e.preventDefault();
		cropState.x = e.touches[0].clientX - startX;
		cropState.y = e.touches[0].clientY - startY;
		updateTransform();
	};
	frame.ontouchend = () => isDragging = false;

	confirmBtn.onclick = () => {
		if (window.SetAvatar && currentAvatarUrl) {
			const gloat = document.getElementById("gloat-message-input").value.trim();
			localStorage.setItem("vbabes_gloat_msg", gloat);

			// Pass the favorite card ID to the server
			const state = window.GetGameState();
			window.SetAvatar(currentAvatarUrl, gloat, "", state.favorite_card_id || 0);

			// Synchronize profile metadata with the server for lobby visibility and moderation
			if (socket && socket.readyState === WebSocket.OPEN) {
				socket.send(JSON.stringify({
					type: "register_avatar",
					payload: { 
						url: currentAvatarUrl,
						gloat: gloat
					}
				}));
			}

			showToast("Avatar verified. Entering Arena.", "success");
		}
	};
	
	// Export to window for access from index.html attributes
	window.applyAvatarFilters = applyAvatarFilters;
}

function generateBracketHTML(matches, activeRound = -1) {
	if (!matches || matches.length === 0) {
		const msg = activeRound === -1 ? "Match data pending blockchain verification or unavailable." : "Matches will be generated once tournament starts...";
		return `<div style="color: #888; font-style: italic; padding: 10px; text-align: center; width: 100%;"></div>`;
	}

	// Group matches by round
	const rounds = {};
	matches.forEach(m => {
		if (!rounds[m.round]) rounds[m.round] = [];
		rounds[m.round].push(m);
	});

	const sortedRounds = Object.keys(rounds).sort((a, b) => a - b);
	
	let html = "";
	sortedRounds.forEach(r => {
		const isCurrentRound = (activeRound == r);

		html += `<div class="bracket-round">`;
		html += `<div class="bracket-round-title">ROUND </div>`;
		rounds[r].forEach(m => {
			const p1Short = getCachedEnvoiName(m.p1);
			const p2Short = getCachedEnvoiName(m.p2);
			
			let p1Class = "";
			let p2Class = "";
			if (m.winner) {
				if (m.winner === m.p1) {
					p1Class = "winner"; p2Class = "loser";
				} else if (m.winner === m.p2) {
					p2Class = "winner"; p1Class = "loser";
				}
			}
			
			html += `
				<div class="bracket-match ${isCurrentRound && !m.winner ? 'active' : ''}">
					<div class="bracket-player "></div>
					<div class="vs-label">VS</div>
					<div class="bracket-player "></div>
				</div>
			`;
		});
		html += `</div>`;
	});
	return html;
}

function updateTournamentPaginationUI() {
	const prevBtn = document.getElementById("prev-tournament-btn");
	const nextBtn = document.getElementById("next-tournament-btn");
	const info = document.getElementById("tournament-page-info");
	
	if (!prevBtn || !nextBtn || !info) return;

	const totalPages = Math.ceil(totalTournaments / tournamentLimit);
	info.innerText = `Page  of ${totalPages || 1}`;

	prevBtn.disabled = (currentTournamentPage <= 1);
	nextBtn.disabled = (currentTournamentPage >= totalPages || totalPages === 0);

	const prevIdx = currentTournamentPage - 1;
	const nextIdx = currentTournamentPage + 1;

	prevBtn.onclick = () => {
		fetchTournamentHistory(prevIdx);
		document.getElementById("hof-history-view").scrollTop = 0;
	};
	nextBtn.onclick = () => {
		fetchTournamentHistory(nextIdx);
		document.getElementById("hof-history-view").scrollTop = 0;
	};
}

// Placeholder for handleTournamentUI - assuming it's defined elsewhere or will be moved here
function handleTournamentUI(tournamentState) {
	const banner = document.getElementById("tournament-banner");
	const statusText = document.getElementById("tournament-status-text");
	const regBtn = document.getElementById("tournament-reg-btn");

	if (!tournamentState || !tournamentState.active) {
		if (banner) banner.classList.add("hidden");
		return;
	}

	if (banner) banner.classList.remove("hidden");
	if (statusText) {
		const network = window.GetGameState()?.network || "VOI";
		const currency = network === "VOI" ? "" : "";

		if (tournamentState.current_round === 0) {
			statusText.innerText = `Registration Open! Buy-in: ${tournamentState.buy_in_amount} `;
			
			// PROACTIVE CHECK: Only show the Join button if critical network config has arrived
			const assetId = network === "VOI" ? CONFIG.VBV_ASSET_ID : CONFIG.AVOI_ASSET_ID;
			if (CONFIG.VAULT_ADDRESS && assetId) {
				if (regBtn) regBtn.classList.remove("hidden");
			} else {
				// If config is missing, inform the user why the button isn't visible yet
				statusText.innerText += " (Establishing Secure Sync...)";
				if (regBtn) regBtn.classList.add("hidden");
			}
		} else {
			statusText.innerText = `Tournament Active - Round ${tournamentState.current_round}`;
			if (regBtn) regBtn.classList.add("hidden");
		}
	}
}

async function renderTournamentBracket(state) {
	// Prime Envoi names for all bracket participants
	const participants = new Set();
	state.matches.forEach(m => {
		if (m.p1) participants.add(m.p1);
		if (m.p2) participants.add(m.p2);
		if (m.winner) participants.add(m.winner);
	});
	await Promise.all(Array.from(participants).filter(p => p && p !== "TBD").map(p => resolveEnvoiName(p)));

	const potEl = document.getElementById("tournament-pot-display");
	if (potEl) potEl.innerText = `POT: ${state.pot.toFixed(1)} `;
	
	const visualization = document.getElementById("bracket-visualization");
	if (visualization) visualization.innerHTML = generateBracketHTML(state.matches, state.current_round);
}

async function registerForTournament() {
	const regBtn = document.getElementById("tournament-reg-btn");
	if (!userAddress) { showToast("Connect wallet first", "error"); return; }
	const state = window.GetGameState();
	if (!state.tournament) return;

	try {
		const buyInBase = state.tournament.buy_in_amount;
		const buyInMicro = Math.floor(buyInBase * 1000000);
		const network = state.network;
		const currency = network === "VOI" ? "" : "";
		const assetId = network === "VOI" ? CONFIG.VBV_ASSET_ID : CONFIG.AVOI_ASSET_ID;

		// HARD GUARD: Block registration if configuration hasn't been synced from the server identity message yet
		if (!assetId || !CONFIG.VAULT_ADDRESS) {
			showToast("⚠️ <b>CRITICAL SYNC ERROR:</b> Arena configuration is missing. Registration is impossible at this time. Please try refreshing.", "error", 10000);
			regBtn.disabled = false;
			regBtn.innerText = "JOIN EVENT";
			return;
		}

		const originalBtnText = regBtn.innerText;
		regBtn.disabled = true;
		regBtn.innerText = "Processing...";

		showToast(`✍️ Signing   Buy-in...`, "info");

		let txid = "";
		let txObj = null;

		// 1. CONSTRUCT TRANSACTION BASED ON NETWORK
		if (network === "VOI") {
			// ARC-200 Transfer (Application Call)
			// Selector for transfer(address,uint256): 0x2b426dec
			const methodSelector = new Uint8Array([0x2b, 0x42, 0x6d, 0xec]);
			const recipientAddr = algosdk.decodeAddress(CONFIG.VAULT_ADDRESS).publicKey;
			
			// Encode amount as 32-byte uint256 for ARC-200
			const amountArg = new Uint8Array(32);
			const amountBI = BigInt(buyInMicro);
			for (let i = 0; i < 8; i++) {
				amountArg[31 - i] = Number((amountBI >> BigInt(i * 8)) & 0xffn);
			}

			txObj = {
				from: userAddress,
				type: 'appl',
				appIndex: assetId,
				appArgs: [methodSelector, recipientAddr, amountArg],
				note: new TextEncoder().encode(`ARENA_TOURN_BUYIN:${Date.now()}`)
			};
		} else if (network === "ALGO") {
			// Standard ASA Transfer
			txObj = {
				from: userAddress,
				to: CONFIG.VAULT_ADDRESS,
				type: 'axfer',
				assetIndex: assetId,
				amount: buyInMicro,
				note: new TextEncoder().encode(`ARENA_TOURN_BUYIN:${Date.now()}`)
			};
		}

		if (!txObj) throw new Error(`Unsupported network configuration: `);

		// 2. SIGN AND BROADCAST BASED ON PROVIDER
		if (walletProvider === 'nautilus') {
			const signed = await window.algo.signTxn([{ txn: algosdk.encodeObj(txObj), signers: [userAddress] }]);
			const { txId: broadcastId } = await window.algo.sendRawTransaction(signed[0]);
			txid = broadcastId;
		} else if (walletProvider === 'kibisis') {
			const txnB64 = btoa(String.fromCharCode(...algosdk.encodeObj(txObj)));
			const signed = await window.kibisis.signTxns([{ txn: txnB64 }]);
			const { txId: broadcastId } = await window.kibisis.pushTxns(signed);
			txid = broadcastId;
		} else if (walletProvider === 'walletconnect' && signClient) {
			const sessions = signClient.session.getAll();
			if (sessions.length === 0) throw new Error("WalletConnect session not found.");
			
			const chainId = network === "VOI" ? CONFIG.VOI_CHAIN_ID : CONFIG.ALGO_CHAIN_ID;
			const txnB64 = btoa(String.fromCharCode(...algosdk.encodeObj(txObj)));
			
			const response = await signClient.request({
				topic: sessions[0].topic,
				chainId: chainId,
				request: {
					method: "algo_signTxn",
					params: [[{ txn: txnB64, signers: [userAddress] }]]
				}
			});

			if (!response || !response[0]) throw new Error("WalletConnect signing failed or was cancelled.");
			
			const signedTxnBytes = new Uint8Array(atob(response[0]).split("").map(c => c.charCodeAt(0)));
			const netCfg = getNetworkConfig(network);
			if (!netCfg || !netCfg.node_url) throw new Error(`Node configuration for  not found. Syncing...`);
			
			const client = new algosdk.Algodv2("", netCfg.node_url, "");
			const { txId: broadcastId } = await client.sendRawTransaction(signedTxnBytes).do();
			txid = broadcastId;
		} else {
			throw new Error("Active wallet provider is not supported for tournament buy-ins.");
		}

		if (!txid) throw new Error("Transaction failed or was cancelled.");

		showToast(`🛰️ Payout Confirmed: ${txid.substring(0,8)}... Registering with server.`, "info");

		// 2. SUBMIT REGISTRATION TO BACKEND
		const response = await fetch(`${CONFIG.API_BASE}/api/tournament/register`, {
			method: "POST",
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				wallet: userAddress,
				txid: txid,
				network: network
			})
		});

		if (response.ok) {
			const result = await response.json();
			showToast(`🏆 Registration Finalized! ${result.message}`, "success", 8000);
			document.getElementById("tournament-reg-btn")?.classList.add("hidden");
		} else {
			const err = await response.text();
			if (response.status === 403) {
				if (err.includes("Opt-in Required")) {
					showToast(`🚫 <b>PROTOCOL BLOCKED</b><br>`, "error", 20000);
				} else if (err.includes("Wallet already registered")) {
					showToast(`🚫 <b>ALREADY REGISTERED:</b> `, "warning", 10000);
					document.getElementById("tournament-reg-btn")?.classList.add("hidden"); // Hide button if already registered
				} else {
					showToast(`❌ Server Sync Failed (403): . Please contact support with TxID: `, "error", 15000);
				}
				return;
			} else if (response.status === 409) { // Handle Conflict specifically
				showToast(`⚠️ <b>REGISTRATION CONFLICT:</b> `, "warning", 10000);
				return;
			}
			showToast(`❌ Server Sync Failed: . Please contact support with TxID: `, "error", 15000);

		}
	} catch (err) {
		console.error("[TOURNAMENT ERROR]", err);
		showToast(`⚠️ Payment aborted: ${err.message}`, "error");
	} finally {
		regBtn.disabled = false;
		regBtn.innerText = originalBtnText;
	}
}

function openTournamentBracket() {
	window.SetPhase("TournamentLobby");
	syncUI();
}

function closeTournamentBracket() {
	window.SetPhase("Lobby");
	syncUI();
}

function openSettingsOverlay() {
	document.getElementById("settings-overlay").classList.remove("hidden");
}

function closeSettingsOverlay() {
	document.getElementById("settings-overlay").classList.add("hidden");
}

function setMasterVolume(value) {
	masterVolume = parseFloat(value);
	window.SetMasterVolume(masterVolume);
	syncUI();
}

function setMusicVolume(value) {
	musicVolume = parseFloat(value);
	window.SetMusicVolume(musicVolume);
	syncUI();
}

function setSfxVolume(value) {
	sfxVolume = parseFloat(value);
	window.SetSfxVolume(sfxVolume);
	syncUI();
}

function toggleMuteMaster() {
	masterVolume = masterVolume === 0 ? 0.5 : 0;
	document.getElementById("master-volume").value = masterVolume;
	setMasterVolume(masterVolume);
}

function toggleMuteMusic() {
	const state = window.GetGameState();
	let newMusicVolume = state.musicVolume === 0 ? 0.5 : 0; // Toggle between 0 and 0.5
	window.SetMusicVolume(newMusicVolume); // Update WASM engine
	document.getElementById("music-volume").value = newMusicVolume; // Update settings slider
	syncUI(); // Re-sync UI to reflect changes, including the new button
}

function toggleMuteSfx() {
	sfxVolume = sfxVolume === 0 ? 0.5 : 0;
	document.getElementById("sfx-volume").value = sfxVolume;
	setSfxVolume(sfxVolume);
}

// Global function to manage transaction status display
function setTransactionStatus(message, type = 'info') {
	const statusEl = document.getElementById("transaction-status");
	if (!statusEl) return;

	if (message) {
		statusEl.classList.remove("hidden");
		statusEl.innerHTML = `<span style="color: ${type === 'error' ? '#ff4b4b' : type === 'success' ? 'var(--neon-green)' : 'var(--neon-cyan)'};"></span>`;
	} else {
		statusEl.classList.add("hidden");
		statusEl.innerHTML = "";
	}
}

// --- Social Sharing Logic ---
function shareTournamentVictory() {
	const state = window.GetGameState();
	const rating = state.deck_rating || "[Z]";
	const score = `${state.scores[0]}-${state.scores[1]}`;
	const arenaUrl = window.location.origin;

	// Construct the text for the tweet
	const tweetText = `🏆 Just dominated the Virtualbabes Arena!\n\n` +
					  `⚔️ Victory: \n` +
					  `🎴 Deck Rating: \n\n` +
					  `Come challenge me on @Voi_Network! 🚀\n\n` +
					  `#Virtualbabes #Voi #NFTGaming #Web3`;

	const twitterUrl = `<https://x.com/intent/tweet?text=${encodeURIComponent(tweetText)}&url=${encodeURIComponent(arenaUrl)}>`;
	
	// Open in a new tab
	window.open(twitterUrl, '_blank');
	
	showToast("Opening X Social Share...", "info");
}

// Placeholder for tournament round transition animation
function showTournamentTransition(roundNumber) {
	const overlay = document.getElementById("tournament-transition-overlay"); // Assume an overlay exists
	if (!overlay) return;
	
	overlay.querySelector(".round-number-display").innerText = `ROUND `;
	overlay.classList.remove("hidden");

	// Trigger fanfare sound effect for high-intensity round advancement
	if (window.PlaySound) {
		window.PlaySound('Pay_out-in.mp3');
	}

	setTimeout(() => overlay.classList.add("hidden"), 3000); // Hide after 3 seconds
}

// Show kidnap overlay with ransom demand
function showKidnapOverlay(payload) {
	const overlay = document.getElementById("kidnap-overlay");
	const content = document.getElementById("kidnap-content");
	if (!overlay || !content) return;

	const ransomValue = payload.ransom || payload.ransom_amount || 0;
	const perpWallet = payload.perp_wallet || "Unknown";

	content.innerHTML = `
		<p>Your card <strong>${payload.card_name}</strong> has been kidnapped!</p>
		<p>Ransom: <span class="ransom-amount">${(ransomValue / 1000000).toFixed(2)} </span></p>
		<p style="opacity:0.7; font-size:0.9em;">Kidnapper: </p>
		<button class="pay-ransom-btn" onclick="payRansom(${payload.card_id}, '', )">Pay Ransom</button>
		<p class="insurance-timer">Insurance recovery in: <span id="recovery-timer">48:00:00</span></p>
	`;
	overlay.classList.remove("hidden");

	// Start countdown timer
	startRecoveryTimer(payload.expires_at);
}

// Pay ransom for kidnapped card
function payRansom(cardId, perpWallet, ransomAmount) {
	if (!socket || socket.readyState !== WebSocket.OPEN) return;
	if (!perpWallet) {
		showToast("Unable to pay ransom: missing kidnapper wallet.", "error");
		return;
	}

	if (!ransomAmount || ransomAmount <= 0) {
		const amountInput = prompt("Enter the ransom amount in VBV to pay for this hostage card:", "0");
		if (!amountInput) return;
		const amountNumber = Number(amountInput);
		if (isNaN(amountNumber) || amountNumber <= 0) {
			showToast("Invalid ransom amount entered.", "error");
			return;
		}
		ransomAmount = Math.round(amountNumber * 1000000);
	}

	socket.send(JSON.stringify({
		type: "pay_ransom",
		payload: { card_id: cardId, perp_wallet: perpWallet, ransom_amount: ransomAmount }
	}));
}

function releaseHostage(cardId) {
	if (!socket || socket.readyState !== WebSocket.OPEN) return;
	if (!confirm(`Release Card # back to its victim?`)) return;

	socket.send(JSON.stringify({
		type: "release_hostage",
		payload: { card_id: cardId }
	}));
}

// Start countdown for insurance recovery
function startRecoveryTimer(expiresAt) {
	const timerEl = document.getElementById("recovery-timer");
	if (!timerEl) return;

	const interval = setInterval(() => {
		const now = Date.now();
		const remaining = expiresAt - now;
		if (remaining <= 0) {
			clearInterval(interval);
			timerEl.textContent = "00:00:00";
			return;
		}
		const hours = Math.floor(remaining / (1000 * 60 * 60));
		const minutes = Math.floor((remaining % (1000 * 60 * 60)) / (1000 * 60));
		const seconds = Math.floor((remaining % (1000 * 60)) / 1000);
		timerEl.textContent = `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
	}, 1000);
}

// --- Industrial Lease Board Logic ---
async function openClubLeaseBoard() {
	const state = window.GetGameState();
	const overlay = document.createElement("div");
	overlay.id = "lease-board-overlay";
	overlay.className = "overlay";

	// Detect priority region from employment
	const myClub = globalClubs[state.employer_id];
	const myRegion = myClub ? myClub.region_name : null;

	let html = `
		<div class="glass-panel" style="width: 700px; text-align: center; border-color: var(--neon-purple);">
			<h2 style="color: var(--neon-purple); letter-spacing: 2px;">INDUSTRIAL LEASE BOARD</h2>
			<p style="font-size: 0.8em; opacity: 0.7; margin-bottom: 20px;">
				Secure high-value tactical assets through the Club rental network.
				${myRegion ? `<br><span style="color: var(--neon-cyan);">Priority Access: <b></b></span>` : ''}
			</p>
			<div id="lease-list-container" class="flex-col gap-10" style="max-height: 450px; overflow-y: auto; padding-right: 10px;">
	`;

	const clubs = Object.values(globalClubs);
	// Sort: Priority Region first, then Mojo
	clubs.sort((a, b) => {
		if (a.region_name === myRegion && b.region_name !== myRegion) return -1;
		if (b.region_name === myRegion && a.region_name !== myRegion) return 1;
		return b.club_mojo - a.club_mojo;
	});

	let found = 0;
	for (const club of clubs) {
		if (!club.leases) continue;
		const available = Object.values(club.leases).filter(l => !l.borrower_wallet);
		if (available.length === 0) continue;

		html += `
			<div style="text-align: left; margin-bottom: 5px; margin-top: 15px; border-bottom: 1px solid rgba(155, 81, 224, 0.4);">
				<small style="color: var(--neon-purple); font-weight: bold; letter-spacing: 1px;">${club.name.toUpperCase()} / ${club.region_name || 'District Sector'}</small>
			</div>
		`;

		for (const lease of available) {
			found++;
			const lender = getCachedEnvoiName(lease.lender_wallet);
			html += `
				<div class="player-item" style="padding: 12px; border-color: var(--glass-border); background: rgba(0,0,0,0.25);">
					<div style="text-align: left; flex: 1;">
						<b style="color: var(--neon-cyan); font-size: 1.1em;">${lease.card_name}</b>
						<div style="font-size: 0.7em; opacity: 0.6;">Lender:  | Term: ${lease.duration_hours}h</div>
					</div>
					<div style="text-align: right; display: flex; align-items: center; gap: 15px;">
						<div style="color: var(--neon-green); font-weight: bold; font-family: 'Rajdhani', sans-serif;">${lease.price.toFixed(1)} </div>
						<button class="outline" style="min-width: 100px; padding: 8px; border-color: var(--neon-purple); color: var(--neon-purple);" 
								onclick="takeLease('${club.id}', '${lease.id}', ${lease.price})">RENT</button>
					</div>
				</div>
			`;
		}
	}

	if (found === 0) {
		html += `<div style="padding: 60px; opacity: 0.4; font-style: italic;">No tactical assets are currently listed for lease.</div>`;
	}

	html += `
			</div>
			<button class="outline mt-20 w-full" onclick="document.getElementById('lease-board-overlay').remove()">DISCONNECT BOARD</button>
		</div>
	`;

	overlay.innerHTML = html;
	document.body.appendChild(overlay);
}

async function takeLease(clubId, leaseId, price) {
	if (!userAddress) return showToast("Connect wallet first", "error");
	if (!confirm(`Rent this card for  ?\n\nProceeding will commit funds from your victory balance.`)) return;
	socket.send(JSON.stringify({ type: "take_lease", payload: { club_id: clubId, lease_id: leaseId } }));
	document.getElementById("lease-board-overlay")?.remove();
}

window.openClubLeaseBoard = openClubLeaseBoard;
window.takeLease = takeLease;

// --- Particle System ---
function initParticleSystem() {
	particleCanvas = document.getElementById("particle-canvas");
	if (!particleCanvas) return;

	particleCtx = particleCanvas.getContext("2d");
	
	// Resize canvas to match its parent (battle-board)
	const battleBoard = document.getElementById("board-container");
	if (battleBoard) {
		const rect = battleBoard.getBoundingClientRect();
		particleCanvas.width = rect.width;
		particleCanvas.height = rect.height;
		particleCanvas.style.left = battleBoard.offsetLeft + "px";
		particleCanvas.style.top = battleBoard.offsetTop + "px";
	}

	// Start animation loop
	if (!particleAnimationId) {
		particleAnimationId = requestAnimationFrame(animateParticles);
	}
}

function animateParticles() {
	if (!particleCtx) return;

	particleCtx.clearRect(0, 0, particleCanvas.width, particleCanvas.height);

	for (let i = particles.length - 1; i >= 0; i--) {
		const p = particles[i];

		// Update position
		p.x += p.vx;
		p.y += p.vy;
		p.vy += 0.1; // Gravity
		p.life--;

		// Fade out
		p.alpha = p.life / p.initialLife;

		// Draw particle
		particleCtx.fillStyle = `rgba(${p.color.r}, ${p.color.g}, ${p.color.b}, ${p.alpha})`;
		particleCtx.beginPath();
		particleCtx.arc(p.x, p.y, p.size * p.alpha, 0, Math.PI * 2);
		particleCtx.fill();

		if (p.life <= 0) {
			particles.splice(i, 1);
		}
	}

	if (particles.length > 0) {
		particleAnimationId = requestAnimationFrame(animateParticles);
	} else {
		particleAnimationId = null; // Stop animation if no particles
	}
}

window.triggerCaptureParticles = (gridIndex, owner) => {
	if (!particleCtx) return;

	const boardContainer = document.getElementById("board-container");
	const slotSize = boardContainer.offsetWidth / 3; // Assuming 3x3 grid
	const col = gridIndex % 3;
	const row = Math.floor(gridIndex / 3);

	const centerX = col * slotSize + slotSize / 2;
	const centerY = row * slotSize + slotSize / 2;

	let color = { r: 0, g: 242, b: 254 }; // Neon Cyan for P1
	if (owner === 1) {
		color = { r: 255, g: 75, b: 75 }; // Error Red for P2
	}

	for (let i = 0; i < 30; i++) { // 30 particles per capture
		const angle = Math.random() * Math.PI * 2;
		const speed = Math.random() * 3 + 1;
		particles.push({
			x: centerX,
			y: centerY,
			vx: Math.cos(angle) * speed,
			vy: Math.sin(angle) * speed,
			size: Math.random() * 3 + 1,
			color: color,
			life: Math.random() * 60 + 30, // 30-90 frames life
			initialLife: Math.random() * 60 + 30,
			alpha: 1
		});
	}

	if (!particleAnimationId) {
		particleAnimationId = requestAnimationFrame(animateParticles);
	}
};
//go:build js && wasm
// +build js,wasm

package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall/js"
	"time"
)

// TournamentMatch represents a single duel within the bracket.
// This mirrors the server's TournamentMatch for client-side display.
type TournamentMatch struct {
	ID     string `json:"id"`
	P1     string `json:"p1"` // Wallet Address
	P2     string `json:"p2"` // Wallet Address
	Winner string `json:"winner,omitempty"`
	Round  int    `json:"round"`
}

// LinkedWallet represents a non-AVM wallet linked to a primary AVM wallet.
type LinkedWallet struct {
	Address   string    `json:"address"`
	Chain     string    `json:"chain"` // e.g., "ETH", "POLY", "SOL"
	Verified  bool      `json:"verified"`
	Timestamp time.Time `json:"timestamp"` // When it was linked/verified
}

// PlaystyleTendencies tracks behavioral metrics for narrative hooks.
type PlaystyleTendencies struct {
	Aggressiveness     float64            `json:"aggressiveness"`
	RiskTolerance      float64            `json:"risk_tolerance"`
	PreferredRules     map[string]float64 `json:"preferred_rules"`
	PreferredCardMoods map[string]float64 `json:"preferred_card_moods"`
}

// TournamentState tracks the progress of an automated event on the client side.
type TournamentState struct {
	Active       bool              `json:"active"`
	Matches      []TournamentMatch `json:"matches"`
	CurrentRound int               `json:"current_round"`
	Participants []string          `json:"participants"`
	Pot          float64           `json:"pot"`
	BuyInAmount  float64           `json:"buy_in_amount"`
	IsBuyInMode  bool              `json:"is_buy_in_mode"`
	OpenTime     time.Time         `json:"open_time"`
}

// WalletLinkInfo stores the primary AVM wallet and its linked non-AVM wallets.
type WalletLinkInfo struct {
	PrimaryAVMWallet string         `json:"primary_avm_wallet"`
	Linked           []LinkedWallet `json:"linked_wallets"`
}

// -----------------------------------------------------------------------------
// 1. ASSET EMBEDDING
// -----------------------------------------------------------------------------
//
// Assets are served statically from the /Public directory by the backend.

// -----------------------------------------------------------------------------
// 2. DATA VAULT (The Unified State Machine)
// -----------------------------------------------------------------------------

type Card struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	Power     [4]int  `json:"power"` // [Top, Right, Bottom, Left]
	Owner     int     `json:"owner"` // 0 for Player 1, 1 for Player 2
	Image     string  `json:"image"`
	Tier      string  `json:"tier"`       // Iron, Bronze, Gold, Diamond
	GlowColor string  `json:"glow_color"` // Hex color for UI effects
	IsCombo   bool    `json:"is_combo"`   // True if flipped during a chain reaction
	Rarity    float64 `json:"rarity"`     // Power multiplier based on supply
	Mood      string  `json:"mood"`       // Volatile, Serene, Spirited, Grounded
	Artifact  int     `json:"artifact"`   // Flat power bonus from equipped items
	Fatigue   int     `json:"fatigue"`    // 0-100, high fatigue penalizes power
	Loyalty   int     `json:"loyalty"`    // 0-100, high loyalty grants combo bonuses
}

type ActiveBuff struct {
	Type      string `json:"type"` // "mood_resist", "stat_boost", "workout"
	Value     int    `json:"value"`
	Remaining int    `json:"matches_left"`
}

type Player struct {
	ID              string       `json:"id"`
	Wallet          string       `json:"wallet"` // The connected blockchain address
	Decks           [4][]Card    `json:"decks"`  // 4 saved deck slots
	ActiveDeck      int          `json:"active_deck"`
	Ready           bool         `json:"ready"`
	Reputation      int          `json:"reputation"`
	WantedLevel     int          `json:"wanted_level"`
	GloatMessage    string       `json:"gloat_message"`
	AvatarURL       string       `json:"avatar_url"`
	Buffs           []ActiveBuff `json:"buffs"`
	AvatarBanNotice string       `json:"avatar_ban_notice"`
	EquippedFaceplate string       `json:"equipped_faceplate"` // For UI rendering
	Mojo            int          `json:"mojo"`             // Social standing for Club unlocks
	SocialRank      string       `json:"social_rank"`      // e.g., "Nobody", "Regular", "Icon"
	JobRole         string       `json:"job_role"`         // Manager, Security, Clerk, Freelancer
	EmployerClubID  string       `json:"employer_club_id"` // The club currently paying this user
	Cunning         int          `json:"cunning"`
	Nurturing       int          `json:"nurturing"`
	Achievements    []string     `json:"achievements"`
	// EmployerClubID string             `json:"employer_club_id"` // The club currently paying this user
	JailedCards      map[int]string      `json:"jailed_cards"`       // CardID -> ClubID (cards currently in jail)
	KidnappedCards   map[int]string      `json:"kidnapped_cards"`    // CardID -> VictimWallet (cards player has kidnapped)
	HeldHostageCards map[int]string      `json:"held_hostage_cards"` // CardID -> KidnapperWallet (cards player has lost to kidnapping)
	FavoriteCardID   int                 `json:"favorite_card_id"`   // Added for Collective Intelligence
	RumorCount       int                 `json:"rumor_count"`        // Number of rumors spread by this player
	Portfolio        map[string]float64  `json:"portfolio"`
	Playstyle        PlaystyleTendencies `json:"playstyle"`
}

// Engine acts as the supreme state machine for the entire App
type Engine struct {
	Network   string
	Faucet    float64
	Phase     string          // "Lobby", "Setup", "TournamentLobby", "Active", "Finished"
	Rules     map[string]bool // Holds Custom Rules (Open, Same, Plus)
	Rewards   map[uint64]float64
	Inventory []Card // Global pool of cards to pick from

	// Asset Pools for Demo/AI
	DemoPool       []string
	AIPortraitPool []string // New pool for AI character avatars
	WitchPool      []string
	LadyPool       []string

	Players             [2]Player // 2P Lobby System
	Board               [9]*Card  // 3x3 Battle Grid
	BoardMoods          [9]string // Moods assigned to specific tiles
	Multiplayer         bool      // True if playing against a human, false for AI
	LocalPlayerIndex    int       // 0 for P1, 1 for P2
	Turn                int       // 0 for Player 1, 1 for Player 2
	Scores              [2]int    // Final scores [P1, P2]
	Maintenance         bool      // True if the arena is under maintenance
	TestingMode         bool      // If true, Player 1 always wins against AI
	IsAdmin             bool      // True if the connected wallet is an administrator
	Winner              int       // -1: None, 0: P1, 1: P2, 2: Draw
	AssetBase           string    // The CDN URL for sounds/images (e.g., GitHub Pages)
	ApiBase             string    // The backend API URL for production
	AmbientAudio        js.Value  // Current background music object
	CurrentAmbientTrack string
	ShowLeaderboard     bool                      // UI Toggle for Hall of Fame
	HardMode            bool                      // If true, AI uses tactical weighted scoring
	AIScore             int                       // Tactical value of the bot's intended move
	ServerLoad          int                       // Current active matches on the server
	SpecialFanfare      string                    // Archetype for specific win/loss tracks: "Emotional", "Witch"
	TerritoryID         string                    // The location of the current match
	ActiveItemBuffs     map[string]map[string]int // PlayerID -> ItemID -> MatchesRemaining
	VaultLow            bool                      // Warning flag for low faucet balance
	DeckRating          string                    // Current player's active deck rating (e.g., [A++])
	MasterVolume        float64                   // Global master volume (0.0 - 1.0)
	MusicVolume         float64                   // Music volume (0.0 - 1.0)
	SfxVolume           float64                   // Sound effects volume (0.0 - 1.0)
	Latency             int                       // WebSocket ping in milliseconds
	NetworkHealth       string                    // "Excellent", "Good", "Poor", "Critical"
	Tournament          TournamentState           `json:"tournament"` // Current bracket info
	linkedWallets       map[string]WalletLinkInfo // Key: Primary AVM Wallet Address
	mutex               sync.RWMutex              // Protects Engine state from concurrent WASM/JS events
}

// Initialize the single source of truth
var Game = Engine{
	Network:          "VOI",
	Faucet:           1000.0,
	Phase:            "Lobby",
	Rules:            map[string]bool{"Open": true, "Power_copy": false, "Power_up": false},
	Rewards:          map[uint64]float64{40227315: 5.0},
	Players:          [2]Player{{ID: "Player 1"}, {ID: "Player 2"}},
	Board:            [9]*Card{},
	BoardMoods:       [9]string{},
	LocalPlayerIndex: 0,
	Multiplayer:      false,
	Turn:             0,
	Winner:           -1,
	DemoPool: []string{ // For AI card images, aligned with Public/Assets/Images/Cards/
		"Cards/Alana.webp",
		"Cards/Bella.webp",
		"Cards/Clohey.webp",
		"Cards/Ellie.webp",
		"Cards/Fran.webp",
		"Cards/Karren.webp",
		"Cards/Kat.webp",
		"Cards/Kay.webp",
		"Cards/Lucy.webp",
		"Cards/Pip.webp",
		"Cards/Roxy.webp",
		"Cards/Sally.webp",
		"Cards/Tammara.webp",
		"Cards/Taya.webp",
		"Cards/Triz.webp",
		"Cards/Xai.webp",
	},
	AIPortraitPool: []string{ // For AI avatar images, aligned with Public/Assets/Images/portraits/
		"portraits/Boss/The_architect/The_architect.webp",
		"portraits/cute/Angelina/Angelina.webp",
		"portraits/cute/Crypto_seraph/Crypto_seraph.webp",
		"portraits/Lady/Casino_sucubus/Casino_sucubus.webp",
		"portraits/Mini-Boss/Evil_angelina/Evil_angelina.webp",
		"portraits/Witch/Evil_jackpot_Jessica/Evil_jackpot_Jessica.webp",
		"portraits/Witch/Jackpot_jessica/Jackpot_jessica.webp",
	},
	TestingMode:    false,
	HardMode:       false,
	AIScore:        0,
	ServerLoad:     0,
	SpecialFanfare: "",
	TerritoryID:    "",
	VaultLow:       false,
	DeckRating:     "[Z]",
	Latency:        0,
	MasterVolume:   0.5, // Default
	MusicVolume:    0.5, // Default
	SfxVolume:      0.5, // Default
	NetworkHealth:  "Excellent",
	ApiBase:        "", // Default to relative
	AssetBase:      "", // Default to relative, can be set via SetAssetBase
}

// -----------------------------------------------------------------------------
// 3. FAUCET & NETWORK (The Ecosystem)
// -----------------------------------------------------------------------------

func connectWallet(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{"status": "error", "message": "No address provided"}
	}
	address := args[0].String()
	
	Game.mutex.Lock()
	Game.Players[0].Wallet = address
	Game.Players[0].ID = address[:6] + "..." + address[len(address)-4:]
	// Transition to Setup Phase for Avatar selection
	Game.Phase = "Setup"
	Game.mutex.Unlock()

	// UpdateAmbientMusic and PlaySound use their own internal logic or are safe
	// but we should ideally ensure Game state is consistent during calls.

	fmt.Printf("[ENGINE] Wallet %s Connected to: %s\n", address, Game.Network)
	PlaySound("click.mp3") // click.mp3 is lowercase in DIR.md
	UpdateAmbientMusic()
	return map[string]interface{}{"status": "success", "address": address, "network": Game.Network}
}

func disconnectWallet(this js.Value, args []js.Value) interface{} {
	Game.mutex.Lock()
	Game.Players[0].Wallet = ""
	Game.Players[0].ID = "Player 1"
	Game.IsAdmin = false
	Game.Players[0].Ready = false
	Game.mutex.Unlock()

	fmt.Println("[ENGINE] Wallet Disconnected.")
	PlaySound("click.mp3")
	UpdateAmbientMusic()
	return true
}

func toggleNetwork(this js.Value, args []js.Value) interface{} {
	Game.mutex.Lock()
	if Game.Network == "VOI" {
		Game.Network = "ALGO"
	} else {
		Game.Network = "VOI"
	}
	network := Game.Network
	Game.mutex.Unlock()
	fmt.Printf("[ENGINE] Network Switched to: %s\n", network)
	PlaySound("click.mp3")
	return network
}

// SetAvatar sets the player's profile image and transitions the game to Lobby phase
func SetAvatar(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	// The payload now includes an optional FavoriteCardID
	url := args[0].String()
	gloat := ""
	if len(args) > 1 {
		gloat = args[1].String()
	}
	notice := ""
	if len(args) > 2 {
		notice = args[2].String()
	}

	favoriteCardID := 0
	if len(args) > 3 {
		favoriteCardID = args[3].Int()
	}

	Game.mutex.Lock()
	Game.Players[0].AvatarURL = url
	Game.Players[0].GloatMessage = gloat
	Game.Players[0].AvatarBanNotice = notice
	Game.Players[0].FavoriteCardID = favoriteCardID
	Game.Phase = "Lobby"
	Game.mutex.Unlock()
	fmt.Println("[ENGINE] Avatar Set. Transitioning to LOBBY.")
	PlaySound("Gear_up_shot.mp3")
	return true
}

func SendReward(this js.Value, args []js.Value) interface{} {
	recipientAddr := Game.Players[0].Wallet
	if recipientAddr == "" {
		fmt.Println("[FAUCET ERROR] No wallet connected. Payout aborted.")
		return Game.Faucet
	}

	// Decrement locally for immediate UI feedback.
	Game.mutex.Lock()
	for _, amt := range Game.Rewards {
		Game.Faucet -= amt
	}
	Game.mutex.Unlock()

	// Payout is now handled by the secure backend to prevent mnemonic exposure.
	// We use a goroutine to trigger a JS fetch call so we don't block the WASM thread.
	go func() {
		fmt.Printf("[ENGINE] Requesting Payout for %s via Backend...\n", recipientAddr)

		clientID := js.Global().Get("myClientId").String()

		payload, _ := json.Marshal(map[string]interface{}{
			"recipient":    recipientAddr,
			"network":      Game.Network,
			"client_id":    clientID,
			"client_score": Game.Scores,
		})

		// Hand off to JavaScript to manage the UI feedback and Transaction lifecycle
		js.Global().Call("processRewardPayout", string(payload))
	}()

	return Game.Faucet
}

// SyncTournament synchronizes the local engine with the server's bracket state
func SyncTournament(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	// Convert JS object to JSON string for easy unmarshaling
	jsonStr := js.Global().Get("JSON").Call("stringify", args[0]).String()

	var ts TournamentState
	if err := json.Unmarshal([]byte(jsonStr), &ts); err != nil {
		fmt.Printf("[ENGINE ERROR] Tournament Sync failed: %v\n", err)
		return false
	}

	Game.mutex.Lock()
	Game.Tournament = ts
	Game.mutex.Unlock()
	fmt.Println("[ENGINE] Tournament State Synchronized.")
	return true
}

// -----------------------------------------------------------------------------
// 4. LOBBY & DECK LOGIC (The Preparation)
// -----------------------------------------------------------------------------

func ToggleRule(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	rule := args[0].String()

	Game.mutex.Lock()
	Game.Rules[rule] = !Game.Rules[rule]
	Game.mutex.Unlock()
	fmt.Printf("[LOBBY] Rule '%s' set to: %v\n", rule, Game.Rules[rule])
	PlaySound("click.mp3")
	return Game.Rules[rule]
}

// PlaySelectSound triggers the card selection audio feedback
func PlaySelectSound(this js.Value, args []js.Value) interface{} {
	PlaySound("Select-place-card.mp3")
	return nil
}

// SelectDeck changes the active deck slot for a player after checking reputation
func SelectDeck(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	slot := args[0].Int()
	if slot < 0 || slot > 3 {
		return false
	}

	// Reputation Thresholds
	thresholds := [4]int{0, 250, 600, 1000}
	if Game.Players[0].Reputation < thresholds[slot] {
		fmt.Printf("[ENGINE] Deck slot %d locked. Need %d Reputation.\n", slot+1, thresholds[slot])
		return false
	}

	Game.mutex.Lock()
	Game.Players[0].ActiveDeck = slot
	Game.mutex.Unlock()
	fmt.Printf("[ENGINE] Deck slot %d selected.\n", slot+1)
	PlaySound("Gear_up_shot.mp3")
	return true
}

// RemoveFromDeck clears a specific card from the current active deck
func RemoveFromDeck(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	cardIdx := args[0].Int()
	p := &Game.Players[0]
	Game.mutex.Lock()
	if cardIdx >= 0 && cardIdx < len(p.Decks[p.ActiveDeck]) {
		p.Decks[p.ActiveDeck] = append(p.Decks[p.ActiveDeck][:cardIdx], p.Decks[p.ActiveDeck][cardIdx+1:]...)
		Game.mutex.Unlock()
		PlaySound("click.mp3")
		return true
	}
	return false
}

func AddToDeck(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	cardID := args[0].Int()
	p := &Game.Players[0]

	Game.mutex.Lock()
	defer Game.mutex.Unlock()
	activeSlot := p.ActiveDeck

	// Guardrail: Max 5 cards per deck
	if len(p.Decks[activeSlot]) >= 5 {
		return false
	}

	// Prevent Duplicates: Check if the card is already in the player's deck
	for _, dc := range p.Decks[activeSlot] {
		if dc.ID == cardID {
			return false
		}
	}

	if c, found := findCard(cardID); found {
		c.Owner = 0
		p.Decks[activeSlot] = append(p.Decks[activeSlot], c)
		PlaySound("click.mp3")
		UpdateAmbientMusic()
		return true
	}
	return false
}

func SetPlayerReady(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	pIndex := args[0].Int()

	// HOTSEAT SIMULATOR: Auto-generate a deck for Player 2 if they are empty
	Game.mutex.Lock()
	if !Game.Multiplayer && pIndex == 1 && len(Game.Players[1].Decks[0]) == 0 {
		fmt.Println("[ENGINE] Generating Demo Deck for CPU...")
		Game.Players[1].ID = "Vbabe Bot"

		portraitPath := Game.AIPortraitPool[rand.Intn(len(Game.AIPortraitPool))]
		Game.Players[1].AvatarURL = Game.resolvePath("Images", portraitPath)

		// Map portrait folder to SpecialFanfare archetype for consistent character feedback
		if strings.Contains(portraitPath, "portraits/Witch/") {
			Game.SpecialFanfare = "Witch"
		} else if strings.Contains(portraitPath, "portraits/Boss/") || strings.Contains(portraitPath, "portraits/Mini-Boss/") {
			Game.SpecialFanfare = "Boss"
		} else if strings.Contains(portraitPath, "portraits/Lady/") {
			Game.SpecialFanfare = "Lady"
		} else {
			Game.SpecialFanfare = "cute"
		}

		Game.Players[1].ActiveDeck = 0
		for i := 0; i < 5; i++ {
			img := Game.DemoPool[i%len(Game.DemoPool)]
			simCard := GenerateCard(1000+i, fmt.Sprintf("Demo Babe %d", i+1), 60.0)
			simCard.Owner = 1
			simCard.Image = img
			Game.Players[1].Decks[0] = append(Game.Players[1].Decks[0], simCard)
		}
	}

	p := &Game.Players[pIndex]
	if len(p.Decks[p.ActiveDeck]) == 5 {
		Game.Players[pIndex].Ready = true
		fmt.Printf("[LOBBY] %s is READY.\n", Game.Players[pIndex].ID)
		PlaySound("click.mp3")
	}
	Game.mutex.Unlock()

	// Trigger UI Start Button if both are ready
	updateStartButton()
	return true
}

// SyncPlayerStats updates reputation and other metrics for players in the lobby
func SyncPlayerStats(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return false
	}
	pIdx := args[0].Int()
	rep := args[1].Int()

	Game.mutex.Lock()
	if pIdx >= 0 && pIdx < 2 {
		Game.Players[pIdx].Reputation = rep
	}
	Game.mutex.Unlock()
	return true
}

// SyncFullProfile ingests a complete player profile from the server to ensure high-fidelity UI rendering.
func SyncFullProfile(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 { return false }
	data := args[0]

	Game.mutex.Lock()
	defer Game.mutex.Unlock()

	p := &Game.Players[0]
	p.Reputation = data.Get("reputation").Int()
	p.Mojo = data.Get("mojo").Int()
	p.SocialRank = data.Get("social_rank").String()
	p.JobRole = data.Get("job_role").String()
	p.EmployerClubID = data.Get("employer_id").String()
	p.WantedLevel = data.Get("wanted_level").Int()
	p.Cunning = data.Get("cunning").Int()
	p.Nurturing = data.Get("nurturing").Int()
	p.RumorCount = data.Get("rumor_count").Int()

	p.EquippedFaceplate = data.Get("equipped_faceplate").String()
	p.FavoriteCardID = data.Get("favorite_card_id").Int()
	// Sync Jailed Cards map
	p.JailedCards = make(map[int]string)
	jsJailed := data.Get("jailed_cards")
	if jsJailed.Type() == js.TypeObject {
		keys := js.Global().Get("Object").Call("keys", jsJailed)
		for i := 0; i < keys.Length(); i++ {
			k := keys.Index(i).String()
			id, _ := strconv.Atoi(k)
			p.JailedCards[id] = jsJailed.Get(k).String()
		}
	}

	// Sync Achievements slice
	p.Achievements = []string{}
	jsAch := data.Get("achievements")
	if jsAch.Type() == js.TypeObject && jsAch.Get("length").Truthy() {
		for i := 0; i < jsAch.Length(); i++ {
			p.Achievements = append(p.Achievements, jsAch.Index(i).String())
		}
	}

	p.KidnappedCards = make(map[int]string) // Reset ephemeral criminal tracking for sync
	p.HeldHostageCards = make(map[int]string)

	fmt.Printf("[ENGINE] Profile Synergized: %d Achievements, %d REP, Faceplate: %s, Fav Card: %d\n", len(p.Achievements), p.Reputation, p.EquippedFaceplate, p.FavoriteCardID)
	return true
}

// SyncPortfolio updates the local player's stock holdings
func SyncPortfolio(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	jsMap := args[0]

	Game.mutex.Lock()
	defer Game.mutex.Unlock()

	Game.Players[0].Portfolio = make(map[string]float64)
	keys := js.Global().Get("Object").Call("keys", jsMap)
	for i := 0; i < keys.Length(); i++ {
		k := keys.Index(i).String()
		Game.Players[0].Portfolio[k] = jsMap.Get(k).Float()
	}

	return true
}

// SyncPlaystyle updates the local player's behavioral analysis from the server
func SyncPlaystyle(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	jsonStr := js.Global().Get("JSON").Call("stringify", args[0]).String()

	var ps PlaystyleTendencies
	if err := json.Unmarshal([]byte(jsonStr), &ps); err != nil {
		return false
	}

	Game.mutex.Lock()
	Game.Players[0].Playstyle = ps
	Game.mutex.Unlock()
	return true
}

// SyncMove allows spectators or remote syncs to place a card for a specific player
func SyncMove(this js.Value, args []js.Value) interface{} {
	if len(args) < 3 {
		return false
	}
	gridIndex := args[0].Int()
	cardID := args[1].Int()
	pIdx := args[2].Int() // 0 for P1, 1 for P2

	// Reset combo flags for visual feedback
	for _, bc := range Game.Board {
		if bc != nil {
			bc.IsCombo = false
		}
	}

	c, found := findCard(cardID)
	if !found {
		// Fallback for missing metadata: provide a baseline card so the UI doesn't crash
		c = Card{ID: cardID, Name: "Syncing...", Power: [4]int{5, 5, 5, 5}, Owner: pIdx}
	}

	Game.mutex.Lock()
	c.Owner = pIdx
	tier, color, _ := calculateTier(Game.Players[pIdx].Reputation)
	c.Tier = tier
	c.GlowColor = color

	heapCard := new(Card)
	*heapCard = c
	Game.Board[gridIndex] = heapCard

	checkCaptures(heapCard, gridIndex)
	Game.Turn = 1 - pIdx // Keep local turn state in sync
	Game.mutex.Unlock()
	return true
}

// SetPhase allows the UI to manually transition the engine's state machine
func SetPhase(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	Game.mutex.Lock()
	Game.Phase = args[0].String()
	Game.mutex.Unlock()
	fmt.Printf("[ENGINE] Phase transitioned to: %s\n", Game.Phase)
	UpdateAmbientMusic()
	return true
}

// SyncServerLoad updates the current count of active matches from the lobby
func SyncServerLoad(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	Game.mutex.Lock()
	Game.ServerLoad = args[0].Int()
	Game.mutex.Unlock()
	return true
}

// SyncLatency updates the engine's network performance state from the JS WebSocket
func SyncLatency(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	ms := args[0].Int()
	Game.mutex.Lock()
	Game.Latency = ms

	if ms < 100 {
		Game.NetworkHealth = "Excellent"
	} else if ms < 300 {
		Game.NetworkHealth = "Good"
	} else if ms < 500 {
		Game.NetworkHealth = "Poor"
	} else {
		Game.NetworkHealth = "Critical"
	}
	Game.mutex.Unlock()
	return true
}

// startLatencyMonitor runs in a background goroutine to periodically trigger pings via JS
func startLatencyMonitor() {
	go func() {
		for {
			// Ping the server every 15 seconds to monitor connection health
			time.Sleep(15 * time.Second)
			js.Global().Call("sendPing")
		}
	}()
}

// AutoBuildDeck picks the 5 strongest cards from inventory, prioritizing the highest possible
// letter grade and maximizing the '+' count (number of cards in that tier).
func AutoBuildDeck(this js.Value, args []js.Value) interface{} {
	if len(Game.Inventory) < 5 {
		fmt.Println("[ENGINE] Not enough cards to auto-build.")
		return false
	}

	Game.mutex.RLock()
	// 1. Create a copy of inventory to sort
	tempInv := make([]Card, len(Game.Inventory))
	copy(tempInv, Game.Inventory)
	Game.mutex.RUnlock()

	// 2. Sort by Tactical Tiering
	// Primary: Highest Letter Grade (Bin) - e.g., A > B
	// Secondary: Power Sum * Scarcity (Battle Score)
	sort.Slice(tempInv, func(i, j int) bool {
		getMaxBin := func(card Card) int {
			maxP := 1
			for _, p := range card.Power {
				if p > maxP {
					maxP = p
				}
			}
			return (maxP - 1) / 100
		}

		binI := getMaxBin(tempInv[i])
		binJ := getMaxBin(tempInv[j])

		if binI != binJ {
			return binI > binJ
		}

		scoreI := float64(tempInv[i].Power[0]+tempInv[i].Power[1]+tempInv[i].Power[2]+tempInv[i].Power[3]) * tempInv[i].Rarity
		scoreJ := float64(tempInv[j].Power[0]+tempInv[j].Power[1]+tempInv[j].Power[2]+tempInv[j].Power[3]) * tempInv[j].Rarity
		return scoreI > scoreJ
	})

	// 3. Populate active deck
	Game.mutex.Lock()
	p := &Game.Players[0]
	p.Decks[p.ActiveDeck] = []Card{}
	for i := 0; i < 5; i++ {
		c := tempInv[i]
		c.Owner = 0
		p.Decks[p.ActiveDeck] = append(p.Decks[p.ActiveDeck], c)
	}
	Game.mutex.Unlock()

	fmt.Printf("[ENGINE] Auto-Built Deck %d (Max Tier Strategy).\n", p.ActiveDeck+1)
	PlaySound("Gear_up_shot.mp3")
	return true
}

// SetTestingMode toggles the 100% win rate against AI for rapid development
func SetTestingMode(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	Game.mutex.Lock()
	Game.TestingMode = args[0].Bool()
	Game.mutex.Unlock()
	fmt.Printf("[ENGINE] Testing Mode set to: %v\n", Game.TestingMode)
	return true
}

// SetHardMode toggles the tactical weighted scoring for the AI bot
func SetHardMode(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	Game.mutex.Lock()
	Game.HardMode = args[0].Bool()
	Game.mutex.Unlock()
	fmt.Printf("[ENGINE] Hard Mode AI: %v\n", Game.HardMode)
	return true
}

// updateStartButton re-evaluates if the "Start Battle" button should be enabled
func updateStartButton() {
	Game.mutex.RLock()
	defer Game.mutex.RUnlock()
	canStart := (Game.Players[0].Ready && Game.Players[1].Ready) && (!Game.Maintenance || Game.IsAdmin)
	js.Global().Call("highlightStartButton", canStart)
}

// SetAdminState allows manual override of admin status (e.g., from server or testing)
func SetAdminState(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	Game.mutex.Lock()
	Game.IsAdmin = args[0].Bool()
	Game.mutex.Unlock()
	fmt.Printf("[ENGINE] Admin State manually set to: %v\n", Game.IsAdmin)
	updateStartButton()
	return true
}

// SetMaintenanceState informs the engine about the arena's maintenance status
func SetMaintenanceState(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	Game.mutex.Lock()
	Game.Maintenance = args[0].Bool()
	Game.mutex.Unlock()
	fmt.Printf("[ENGINE] Maintenance Mode set to: %v\n", Game.Maintenance)
	updateStartButton()
	return true
}

// SyncOpponentProfile updates the metadata for an opponent during a multiplayer match
func SyncOpponentProfile(this js.Value, args []js.Value) interface{} {
	if len(args) < 3 {
		return false
	}
	pIdx := args[0].Int()
	avatar := args[1].String()
	gloat := args[2].String()

	Game.mutex.Lock()
	if pIdx >= 0 && pIdx < 2 {
		Game.Players[pIdx].AvatarURL = avatar
		Game.Players[pIdx].GloatMessage = gloat
	}
	Game.mutex.Unlock()
	return true
}

// SyncOpponentDeck populates a player's deck from a list of IDs (used for Multiplayer Handshakes)
func SyncOpponentDeck(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return false
	}
	pIndex := args[0].Int()
	idsVal := args[1]

	if idsVal.Type() != js.TypeObject {
		return false
	}

	// Clear existing deck for the specified player
	Game.mutex.Lock()
	Game.Players[pIndex].Decks[0] = []Card{}

	for i := 0; i < idsVal.Length(); i++ {
		id := idsVal.Index(i).Int()
		if c, found := findCard(id); found {
			c.Owner = pIndex
			Game.Players[pIndex].Decks[0] = append(Game.Players[pIndex].Decks[0], c)
		}
	}

	Game.Players[pIndex].Ready = true
	Game.mutex.Unlock()
	return true
}

// ForceActive allows spectators to bypass lobby requirements and enter combat mode
func ForceActive(this js.Value, args []js.Value) interface{} {
	Game.mutex.Lock()
	Game.Phase = "Active"
	Game.Board = [9]*Card{} // Clear local board for fresh sync
	Game.mutex.Unlock()
	fmt.Println("[ENGINE] Phase forced to ACTIVE for Spectating.")
	return true
}

// SyncVaultBalance updates the local faucet state from a server broadcast
func SyncVaultBalance(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	newBalance := args[0].Float()
	Game.mutex.Lock()
	Game.Faucet = newBalance
	Game.VaultLow = newBalance < 1000.0
	Game.mutex.Unlock()
	fmt.Printf("[ENGINE] Vault Balance Synced: %.2f $VBV\n", Game.Faucet)
	return true
}

// TriggerManualSync allows the UI to force a server-side re-sync of the player's on-chain stats.
func TriggerManualSync(this js.Value, args []js.Value) interface{} {
	wallet := Game.Players[0].Wallet
	if wallet == "" {
		fmt.Println("[ENGINE ERROR] Cannot trigger sync: No wallet connected.")
		return false
	}

	go func() {
		fmt.Printf("[ENGINE] Requesting manual stats re-sync for %s...\n", wallet)

		payload, _ := json.Marshal(map[string]string{"wallet": wallet})
		window := js.Global()

		// Construct fetch options
		options := window.Get("Object").New()
		options.Set("method", "POST")
		headers := window.Get("Object").New()
		headers.Set("Content-Type", "application/json")
		options.Set("headers", headers)
		options.Set("body", string(payload))

		// Execute fetch to the backend endpoint
		promise := window.Call("fetch", Game.ApiBase+"/api/re-sync-stats", options)

		success := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			fmt.Println("[ENGINE] Manual sync initiated successfully.")
			return nil
		})

		failure := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			fmt.Printf("[ENGINE ERROR] Manual sync request failed: %v\n", args[0])
			return nil
		})

		promise.Call("then", success).Call("catch", failure)
	}()

	return true
}

// SyncRewards updates the multi-reward registry from server
func SyncRewards(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	jsMap := args[0]
	Game.mutex.Lock()
	// Refactor: Atomic Map Update (Clearing and repopulating avoids panic)
	Game.Rewards = make(map[uint64]float64)
	keys := js.Global().Get("Object").Call("keys", jsMap)
	for i := 0; i < keys.Length(); i++ {
		k := keys.Index(i).String()
		id, _ := strconv.ParseUint(k, 10, 64)
		Game.Rewards[id] = jsMap.Get(k).Float() / 1000000.0 // Convert micro to base
	}
	Game.mutex.Unlock()

	fmt.Printf("[ENGINE] Multi-Rewards Synced: %d active assets\n", len(Game.Rewards))
	return true
}

// SyncRules updates the internal rule set from a server broadcast (Admin control)
func SyncRules(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	jsRules := args[0]
	if jsRules.Type() != js.TypeObject {
		return false
	}

	Game.mutex.Lock()
	Game.Rules["Open"] = jsRules.Get("Open").Bool()
	Game.Rules["Same"] = jsRules.Get("Same").Bool()
	Game.Rules["Plus"] = jsRules.Get("Plus").Bool()
	Game.mutex.Unlock()

	fmt.Printf("[ENGINE] Rules Synchronized: %v\n", Game.Rules)
	return true
}

// SetBoardState bulk-loads the 3x3 grid and rules (used for Spectator Sync)
func SetBoardState(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	data := args[0]

	Game.mutex.Lock()
	defer Game.mutex.Unlock()
	jsBoard := data.Get("board")
	if jsBoard.Type() == js.TypeObject {
		for i := 0; i < 9; i++ {
			val := jsBoard.Index(i)
			if val.IsNull() || val.IsUndefined() {
				Game.Board[i] = nil
				continue
			}
			id := val.Get("id").Int()
			owner := val.Get("owner").Int()
			if c, found := findCard(id); found {
				c.Owner = owner
				// Create a new pointer instance for the board
				heapCard := new(Card)
				*heapCard = c
				Game.Board[i] = heapCard
			}
		}
	}

	// 3. Sync Board Moods (Authoritative Environmental Hazards)
	jsMoods := data.Get("board_moods")
	if jsMoods.Type() == js.TypeObject {
		for i := 0; i < 9; i++ {
			mood := jsMoods.Index(i)
			if mood.IsString() {
				Game.BoardMoods[i] = mood.String()
			} else {
				Game.BoardMoods[i] = "Neutral"
			}
		}
	}

	// 4. Sync Ruleset (Deterministic Alignment)
	jsRules := data.Get("rules")
	if jsRules.Type() == js.TypeObject {
		ruleKeys := []string{"Open", "Power_copy", "Power_up", "Elemental_sync", "Fallen_penalty", "Artifact_bonus"}
		for _, k := range ruleKeys {
			if r := jsRules.Get(k); !r.IsUndefined() {
				Game.Rules[k] = r.Bool()
			}
		}
	}

	// 5. Sync Tactical & Penalty Metadata for Spectator Accuracy
	if tid := data.Get("territory_id"); !tid.IsUndefined() {
		Game.TerritoryID = tid.String()
	}

	// 5.1 Sync Active Item Buffs
	jsActiveItemBuffs := data.Get("active_item_buffs")
	if jsActiveItemBuffs.Type() == js.TypeObject {
		Game.ActiveItemBuffs = make(map[string]map[string]int)
		js.Global().Get("Object").Call("keys", jsActiveItemBuffs).Call("forEach", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			playerID := args[0].String()
			Game.ActiveItemBuffs[playerID] = make(map[string]int)
			js.Global().Get("Object").Call("keys", jsActiveItemBuffs.Get(playerID)).Call("forEach", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				itemID := args[0].String()
				Game.ActiveItemBuffs[playerID][itemID] = jsActiveItemBuffs.Get(playerID).Get(itemID).Int()
				return nil
			}))
			return nil
		}))
	}
	if a1 := data.Get("p1_avatar"); !a1.IsUndefined() { Game.Players[0].AvatarURL = a1.String() }
	if g1 := data.Get("p1_gloat"); !g1.IsUndefined() { Game.Players[0].GloatMessage = g1.String() }
	if a2 := data.Get("p2_avatar"); !a2.IsUndefined() { Game.Players[1].AvatarURL = a2.String() }
	if g2 := data.Get("p2_gloat"); !g2.IsUndefined() { Game.Players[1].GloatMessage = g2.String() }

	if w1 := data.Get("p1_wanted_level"); !w1.IsUndefined() { Game.Players[0].WantedLevel = w1.Int() }
	if c1 := data.Get("p1_cunning"); !c1.IsUndefined() { Game.Players[0].Cunning = c1.Int() }
	if n1 := data.Get("p1_nurturing"); !n1.IsUndefined() { Game.Players[0].Nurturing = n1.Int() }
	
	if w2 := data.Get("p2_wanted_level"); !w2.IsUndefined() { Game.Players[1].WantedLevel = w2.Int() }
	if c2 := data.Get("p2_cunning"); !c2.IsUndefined() { Game.Players[1].Cunning = c2.Int() }
	if n2 := data.Get("p2_nurturing"); !n2.IsUndefined() { Game.Players[1].Nurturing = n2.Int() }

	jsScores := data.Get("scores")
	if jsScores.Type() == js.TypeObject && jsScores.Length() >= 2 {
		Game.Scores[0] = jsScores.Index(0).Int()
		Game.Scores[1] = jsScores.Index(1).Int()
	}

	// 6. Recalculate Turn based on board occupancy (Deterministic Inference)
	placedCount := 0
	for _, c := range Game.Board {
		if c != nil {
			placedCount++
		}
	}
	Game.Turn = placedCount % 2

	fmt.Printf("[ENGINE] Spectator state synchronized. Phase: %s, Territory: %s, Turn: %d\n", Game.Phase, Game.TerritoryID, Game.Turn)
	return true
}

// findCard is a private helper to retrieve a card from the global inventory by ID
func findCard(id int) (Card, bool) {
	Game.mutex.RLock()
	defer Game.mutex.RUnlock()
	for _, c := range Game.Inventory {
		if c.ID == id {
			return c, true
		}
	}
	return Card{}, false
}

func ResetGame(this js.Value, args []js.Value) interface{} {
	Game.mutex.Lock()
	defer Game.mutex.Unlock()
	Game.Phase = "Lobby"
	Game.Board = [9]*Card{}
	Game.Multiplayer = false
	Game.Turn = 0
	Game.Winner = -1
	Game.Scores = [2]int{0, 0}
	Game.LocalPlayerIndex = 0
	Game.SpecialFanfare = ""

	// Clear player readiness and decks
	for i := range Game.Players {
		Game.Players[i].Ready = false
		for d := 0; d < 4; d++ {
			Game.Players[i].Decks[d] = []Card{}
		}
	}

	fmt.Println("[ENGINE] Game Reset to Lobby. State Cleared.")
	PlaySound("click.mp3")
	return true
}

// -----------------------------------------------------------------------------
// 5. THE WAR ROOM (The Combat Grid)
// -----------------------------------------------------------------------------

func StartMatch(this js.Value, args []js.Value) interface{} {
	if Game.Maintenance && !Game.IsAdmin {
		fmt.Println("[ENGINE ERROR] Cannot start match: Maintenance in progress.")
		return false
	}

	if !Game.Players[0].Ready || !Game.Players[1].Ready {
		return false
	}

	// Optional: Check if StartMatch(true) was passed for Multiplayer
	Game.Multiplayer = false
	if len(args) > 0 && args[0].Type() == js.TypeBoolean {
		Game.Multiplayer = args[0].Bool()
	}

	if Game.Multiplayer {
		Game.Players[1].ID = "Opponent"
	}

	Game.mutex.Lock()
	defer Game.mutex.Unlock()
	Game.Phase = "Active"
	Game.Turn = 0           // Player 1 starts
	Game.Board = [9]*Card{} // Clear board

	// Randomize Board Moods if Elemental Rule is active
	moodTypes := []string{"Volatile", "Serene", "Spirited", "Grounded", "Neutral"}
	for i := 0; i < 9; i++ {
		if Game.Rules["Elemental_sync"] && rand.Intn(10) > 6 {
			Game.BoardMoods[i] = moodTypes[rand.Intn(4)]
		} else {
			Game.BoardMoods[i] = "Neutral"
		}
	}

	Game.Winner = -1
	Game.Scores = [2]int{0, 0}

	UpdateAmbientMusic()
	fmt.Println("=================================")
	fmt.Printf(" BATTLE START! Rules: %v\n", Game.Rules)
	fmt.Println("=================================")
	PlaySound("flip.mp3")
	return true
}

func PlaceCard(this js.Value, args []js.Value) interface{} {
	if Game.Phase != "Active" || len(args) < 2 {
		return false
	}

	gridIndex := args[0].Int()
	cardID := args[1].Int()

	// Reset combo flags for all cards on board at the start of a move
	Game.mutex.Lock()
	for _, boardCard := range Game.Board {
		if boardCard != nil {
			boardCard.IsCombo = false
		}
	}
	Game.mutex.Unlock()

	// Reset AI score when the player takes their turn
	Game.AIScore = 0

	// Guardrails: Check if grid is valid and empty
	if gridIndex < 0 || gridIndex > 8 || Game.Board[gridIndex] != nil {
		fmt.Println("[BATTLE ERROR] Invalid or occupied slot.")
		return false
	}

	pIndex := Game.Turn

	p := &Game.Players[pIndex]
	for i, c := range p.Decks[p.ActiveDeck] {
		Game.mutex.Lock()
		defer Game.mutex.Unlock()
		if c.ID == cardID {
			Game.Board[gridIndex] = &p.Decks[p.ActiveDeck][i]

			// Apply Visual Tier Effects based on the player's current reputation
			tier, color, _ := calculateTier(Game.Players[pIndex].Reputation)
			Game.Board[gridIndex].Tier = tier
			Game.Board[gridIndex].GlowColor = color

			fmt.Printf("[BATTLE] %s placed %s at Grid %d\n", Game.Players[pIndex].ID, c.Name, gridIndex)

			checkCaptures(&p.Decks[p.ActiveDeck][i], gridIndex)
			PlaySound("Select-place-card.mp3")

			// Switch Turn
			if Game.Turn == 0 {
				Game.Turn = 1

				// TACTICAL REFACTOR: Implement 'Lag Guard' for AI triggers
				if !Game.Multiplayer {
					health := Game.NetworkHealth
					go func() {
						// If network is critical, wait for socket to settle
						if health == "Critical" {
							// Use state variables carefully across the boundary
							fmt.Printf("[LAG GUARD] Latency at %dms. Delaying AI response for stability...\n", Game.Latency)
							time.Sleep(2 * time.Second)
						}
						PerformAIMove()
					}()
				}
			} else {
				Game.Turn = 0
			}

			checkWinCondition()
			UpdateAmbientMusic()
			return true
		}
	}
	return false
}

// getEffectivePower applies Mood and Artifact modifiers to a card's side
func getEffectivePower(c *Card, sideIdx int, gridIdx int) int {
	base := c.Power[sideIdx] + c.Artifact

	player := &Game.Players[c.Owner]

	// Apply Wanted Level Penalty (Mitigated by Cunning)
	wantedPenalty := (player.WantedLevel * 5)
	// Cunning mitigates penalty: every 1 point of Cunning reduces penalty by 2
	mitigation := player.Cunning * 2
	if mitigation > wantedPenalty { mitigation = wantedPenalty }
	base -= (wantedPenalty - mitigation)

	// Fatigue Penalty: -1 power per point above 50
	if c.Fatigue > 50 {
		fatiguePenalty := (c.Fatigue - 50)
		// Nurturing reduces fatigue impact: 1 power back per Nurturing point
		reduction := player.Nurturing
		if reduction > fatiguePenalty { reduction = fatiguePenalty }
		base -= (fatiguePenalty - reduction)
	}

	// Loyalty Bonus: +25 power for soul-bonded cards
	if c.Loyalty >= 100 {
		base += 25
	}

	if Game.Rules["Elemental_sync"] {
		tileMood := Game.BoardMoods[gridIdx]
		if tileMood != "Neutral" && c.Mood != "" && c.Mood != "Neutral" {
			moodWeaknesses := map[string]string{
				"Volatile": "Serene",
				"Serene":   "Spirited",
				"Spirited": "Grounded",
				"Grounded": "Volatile",
			}

			if c.Mood == tileMood {
				base += 50 // Match bonus: +0.5 Tier
			} else if moodWeaknesses[c.Mood] == tileMood {
				base -= 50 // Weakness penalty: -0.5 Tier
			}
		}
	}

	return base
}

// triggerCaptureParticlesInternal is the Go-internal call to JS
func triggerCaptureParticlesInternal(gridIndex int, owner int) {
	js.Global().Call("triggerCaptureParticles", gridIndex, owner)
}

// PlayCaptureEffect is the bridge function for js.FuncOf
func PlayCaptureEffect(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 { return nil }
	triggerCaptureParticlesInternal(args[0].Int(), args[1].Int())
	return nil
}

// checkCaptures applies the combat logic for a newly placed card
func checkCaptures(placedCard *Card, gridIndex int) int {
	totalFlips := 0
	// Define relative indices for neighbors and corresponding power indices
	// Power: [Top, Right, Bottom, Left]
	// {offset_from_current_index, placed_card_power_index, neighbor_card_power_index, boundary_check_function}
	neighbors := []struct {
		offset           int
		placedPowerIdx   int
		neighborPowerIdx int
		boundaryCheck    func(int) bool // Function to check if neighbor is within bounds
	}{
		{-3, 0, 2, func(idx int) bool { return idx >= 3 }},   // Top: placed.Top vs neighbor.Bottom
		{+1, 1, 3, func(idx int) bool { return idx%3 != 2 }}, // Right: placed.Right vs neighbor.Left
		{+3, 2, 0, func(idx int) bool { return idx <= 5 }},   // Bottom: placed.Bottom vs neighbor.Top
		{-1, 3, 1, func(idx int) bool { return idx%3 != 0 }}, // Left: placed.Left vs neighbor.Right
	}

	// Groups to track rule matches (Value/Sum -> list of neighbor indices)
	sameGroups := make(map[int][]int)
	plusGroups := make(map[int][]int)

	var comboQueue []int // Indices of cards flipped by Same/Plus to start combos

	for _, n := range neighbors {
		neighborIndex := gridIndex + n.offset

		// Check if the neighbor index is within board bounds and the slot is occupied
		if n.boundaryCheck(gridIndex) && Game.Board[neighborIndex] != nil {
			neighborCard := Game.Board[neighborIndex]
			placedPower := getEffectivePower(placedCard, n.placedPowerIdx, gridIndex)
			neighborPower := getEffectivePower(neighborCard, n.neighborPowerIdx, neighborIndex)

			// 1. Prepare Power_copy Rule Data (Equality check)
			if Game.Rules["Power_copy"] && placedPower == neighborPower {
				sameGroups[placedPower] = append(sameGroups[placedPower], neighborIndex)
			}

			// 2. Prepare Power_up Rule Data (Sum check)
			if Game.Rules["Power_up"] {
				sum := placedPower + neighborPower
				plusGroups[sum] = append(plusGroups[sum], neighborIndex)
			}

			// 3. Basic Capture (Direct Power Comparison)
			if neighborCard.Owner != placedCard.Owner && placedPower > neighborPower {
				if flipCard(neighborIndex, placedCard.Owner, "BASIC") {
					totalFlips++
				}
			}
		}
	}

	// 4. Process Power_copy Rule triggers (Requires at least 2 matching sides)
	for _, indices := range sameGroups {
		if len(indices) >= 2 {
			for _, idx := range indices {
				if flipCard(idx, placedCard.Owner, "SAME") {
					totalFlips++
					comboQueue = append(comboQueue, idx)
				}
			}
		}
	}

	// 5. Process Power_up Rule triggers (Requires at least 2 matching sums)
	for _, indices := range plusGroups {
		if len(indices) >= 2 {
			for _, idx := range indices {
				if flipCard(idx, placedCard.Owner, "POWER_UP") {
					totalFlips++
					comboQueue = append(comboQueue, idx)
				}
			}
		}
	}

	// 6. Process Combo Chain (Recursive Basic Captures)
	for len(comboQueue) > 0 {
		currentIndex := comboQueue[0]
		comboQueue = comboQueue[1:]
		currentCard := Game.Board[currentIndex]

		// Define neighbors for the combo card
		comboNeighbors := []struct {
			offset           int
			placedPowerIdx   int
			neighborPowerIdx int
			boundaryCheck    func(int) bool
		}{
			{-3, 0, 2, func(idx int) bool { return idx >= 3 }},
			{+1, 1, 3, func(idx int) bool { return idx%3 != 2 }},
			{+3, 2, 0, func(idx int) bool { return idx <= 5 }},
			{-1, 3, 1, func(idx int) bool { return idx%3 != 0 }},
		}

		for _, cn := range comboNeighbors {
			nbIdx := currentIndex + cn.offset
			if cn.boundaryCheck(currentIndex) && Game.Board[nbIdx] != nil {
				neighbor := Game.Board[nbIdx]
				// Combo only triggers Basic Capture logic (Power Comparison)

				cPower := getEffectivePower(currentCard, cn.placedPowerIdx, currentIndex)
				nPower := getEffectivePower(neighbor, cn.neighborPowerIdx, nbIdx)

				if neighbor.Owner != currentCard.Owner && cPower > nPower {
					if flipCard(nbIdx, currentCard.Owner, "COMBO") {
						totalFlips++
						comboQueue = append(comboQueue, nbIdx)
					}
				}
			}
		}
	}
	return totalFlips
}

// flipCard handles owner change and rule-specific debuffs
func flipCard(idx int, newOwner int, reason string) bool {
	c := Game.Board[idx]
	if c == nil || c.Owner == newOwner {
		return false
	}

	c.Owner = newOwner
	c.IsCombo = (reason == "COMBO" || reason == "SAME" || reason == "POWER_UP")

	if Game.Rules["Fallen_penalty"] {
		c.Artifact -= 20 // Permanent debuff for being captured
		triggerCaptureParticlesInternal(idx, newOwner) // Trigger particle effect on capture
	}
	return true
}

// simulateCapturesOnBoard is a helper to calculate score for a move on a given board.
// It's a slightly modified version of the main simulateCaptures logic,
// adapted to work on a passed board and player index.
func simulateCapturesOnBoard(board [9]*Card, placedCard *Card, gridIndex int, playerIndex int, rules map[string]bool) int {
	totalScore := 0
	flipped := make(map[int]bool) // Tracks cards that would be flipped in this simulation

	neighbors := []struct {
		offset           int
		placedPowerIdx   int
		neighborPowerIdx int
		boundaryCheck    func(int) bool
	}{
		{-3, 0, 2, func(idx int) bool { return idx >= 3 }},
		{+1, 1, 3, func(idx int) bool { return idx%3 != 2 }},
		{+3, 2, 0, func(idx int) bool { return idx <= 5 }},
		{-1, 3, 1, func(idx int) bool { return idx%3 != 0 }},
	}

	sameGroups := make(map[int][]int)
	plusGroups := make(map[int][]int)
	var comboQueue []int

	// These weights are for evaluating the *player's* potential, so they should reflect
	// how much the AI *doesn't* want the player to achieve these.
	// Using the HardMode weights for the player's potential makes sense.
	playerRuleTriggerWeight := 250
	playerRuleFlipWeight := 100
	playerComboFlipWeight := 60

	// 1. Initial Scan
	for _, n := range neighbors {
		neighborIndex := gridIndex + n.offset
		if n.boundaryCheck(gridIndex) && board[neighborIndex] != nil {
			neighborCard := board[neighborIndex]
			pPower := getEffectivePower(placedCard, n.placedPowerIdx, gridIndex)
			nPower := getEffectivePower(neighborCard, n.neighborPowerIdx, neighborIndex)

			if rules["Power_copy"] && pPower == nPower {
				sameGroups[pPower] = append(sameGroups[pPower], neighborIndex)
			}
			if rules["Power_up"] {
				sum := pPower + nPower
				plusGroups[sum] = append(plusGroups[sum], neighborIndex)
			}
			// Basic Capture (10 points per flip)
			// Check if it flips an *opponent's* card (from the perspective of playerIndex)
			if neighborCard.Owner != playerIndex && pPower > nPower {
				flipped[neighborIndex] = true
				totalScore += 10
			}
		}
	}

	// 2. Rules check
	for _, indices := range sameGroups {
		if len(indices) >= 2 {
			totalScore += playerRuleTriggerWeight
			for _, idx := range indices {
				// Check if it flips an *opponent's* card (from the perspective of playerIndex)
				if board[idx].Owner != playerIndex {
					flipped[idx] = true
					totalScore += playerRuleFlipWeight
					comboQueue = append(comboQueue, idx)
				}
			}
		}
	}
	for _, indices := range plusGroups {
		if len(indices) >= 2 {
			totalScore += playerRuleTriggerWeight
			for _, idx := range indices {
				// Check if it flips an *opponent's* card (from the perspective of playerIndex)
				if board[idx].Owner != playerIndex {
					flipped[idx] = true
					totalScore += playerRuleFlipWeight
					comboQueue = append(comboQueue, idx)
				}
			}
		}
	}

	// 3. Combo Chain Simulation
	for len(comboQueue) > 0 {
		currIdx := comboQueue[0]
		comboQueue = comboQueue[1:]
		currentCard := board[currIdx]

		for _, n := range neighbors {
			nbIdx := currIdx + n.offset
			if n.boundaryCheck(currIdx) && board[nbIdx] != nil {
				neighbor := board[nbIdx]
				// Combo only triggers Basic Capture logic (Power Comparison)
				// Check if it flips an *opponent's* card (from the perspective of playerIndex)
				if neighbor.Owner != playerIndex && !flipped[nbIdx] {
					if getEffectivePower(currentCard, n.placedPowerIdx, currIdx) > getEffectivePower(neighbor, n.neighborPowerIdx, nbIdx) {
						flipped[nbIdx] = true
						totalScore += playerComboFlipWeight
						comboQueue = append(comboQueue, nbIdx)
					}
				}
			}
		}
	}
	return totalScore
}

// calculateMaxPlayerPotential simulates the best possible move for the player
// on their next turn given a board state.
func calculateMaxPlayerPotential(board [9]*Card, playerHand []Card, playerIndex int, rules map[string]bool) int {
	maxPotentialScore := 0
	emptySlots := []int{}
	for i, c := range board {
		if c == nil {
			emptySlots = append(emptySlots, i)
		}
	}

	if len(emptySlots) == 0 || len(playerHand) == 0 {
		return 0
	}

	for _, playerCard := range playerHand {
		for _, slot := range emptySlots {
			// Create a deep copy of the playerCard to avoid modifying the original in hand
			playerCardCopy := playerCard

			// Simulate the player placing their card on a temporary board
			simulatedBoard := [9]*Card{}
			for i, c := range board {
				if c != nil {
					// Deep copy the card on the board to avoid modifying the original Game.Board cards
					tempCard := *c
					simulatedBoard[i] = &tempCard
				}
			}
			simulatedBoard[slot] = &playerCardCopy // Place the player's card on the simulated board

			// Calculate the score the player would get from this move
			score := simulateCapturesOnBoard(simulatedBoard, &playerCardCopy, slot, playerIndex, rules)
			if score > maxPotentialScore {
				maxPotentialScore = score
			}
		}
	}
	return maxPotentialScore
}

// simulateCaptures calculates how many cards would be flipped without modifying the board
func simulateCaptures(placedCard *Card, gridIndex int) int {
	totalScore := 0
	flipped := make(map[int]bool)

	// Helper for checking neighbors (identical to checkCaptures logic)
	neighbors := []struct {
		offset           int
		placedPowerIdx   int
		neighborPowerIdx int
		boundaryCheck    func(int) bool
	}{
		{-3, 0, 2, func(idx int) bool { return idx >= 3 }},
		{+1, 1, 3, func(idx int) bool { return idx%3 != 2 }},
		{+3, 2, 0, func(idx int) bool { return idx <= 5 }},
		{-1, 3, 1, func(idx int) bool { return idx%3 != 0 }},
	}

	sameGroups := make(map[int][]int)
	plusGroups := make(map[int][]int)
	var comboQueue []int

	// Tactical Weights for Hard Mode
	ruleTriggerWeight := 50
	ruleFlipWeight := 20
	comboFlipWeight := 15

	if Game.HardMode {
		ruleTriggerWeight = 250 // Heavily prioritize setting up Same/Plus
		ruleFlipWeight = 100
		comboFlipWeight = 60
	}

	// 1. Initial Scan
	for _, n := range neighbors {
		neighborIndex := gridIndex + n.offset
		if n.boundaryCheck(gridIndex) && Game.Board[neighborIndex] != nil {
			neighborCard := Game.Board[neighborIndex]
			pPower := getEffectivePower(placedCard, n.placedPowerIdx, gridIndex)
			nPower := getEffectivePower(neighborCard, n.neighborPowerIdx, neighborIndex)

			if Game.Rules["Power_copy"] && pPower == nPower {
				sameGroups[pPower] = append(sameGroups[pPower], neighborIndex)
			}
			if Game.Rules["Power_up"] {
				sum := pPower + nPower
				plusGroups[sum] = append(plusGroups[sum], neighborIndex)
			}
			// Basic Capture (10 points per flip)
			if neighborCard.Owner != placedCard.Owner && pPower > nPower {
				flipped[neighborIndex] = true
				totalScore += 10
			}
		}
	}

	// 2. Rules check
	for _, indices := range sameGroups {
		if len(indices) >= 2 {
			totalScore += ruleTriggerWeight // Rule trigger bonus (Tactical Priority)
			for _, idx := range indices {
				if Game.Board[idx].Owner != placedCard.Owner {
					flipped[idx] = true
					totalScore += ruleFlipWeight // Rule flip weight
					comboQueue = append(comboQueue, idx)
				}
			}
		}
	}
	for _, indices := range plusGroups {
		if len(indices) >= 2 {
			totalScore += ruleTriggerWeight // Rule trigger bonus
			for _, idx := range indices {
				if Game.Board[idx].Owner != placedCard.Owner {
					flipped[idx] = true
					totalScore += ruleFlipWeight // Rule flip weight
					comboQueue = append(comboQueue, idx)
				}
			}
		}
	}

	// 3. Combo Chain Simulation
	for len(comboQueue) > 0 {
		currIdx := comboQueue[0]
		comboQueue = comboQueue[1:]
		currCard := Game.Board[currIdx]

		for _, n := range neighbors {
			nbIdx := currIdx + n.offset
			if n.boundaryCheck(currIdx) && Game.Board[nbIdx] != nil {
				neighbor := Game.Board[nbIdx]
				if neighbor.Owner != placedCard.Owner && !flipped[nbIdx] {
					if getEffectivePower(currCard, n.placedPowerIdx, currIdx) > getEffectivePower(neighbor, n.neighborPowerIdx, nbIdx) {
						flipped[nbIdx] = true
						totalScore += comboFlipWeight // Combo flip weight
						comboQueue = append(comboQueue, nbIdx)
					}
				}
			}
		}
	}

	if Game.HardMode {
		// Defensive penalty: avoid placing weak sides against open board slots
		for _, n := range neighbors {
			if n.boundaryCheck(gridIndex) && Game.Board[gridIndex+n.offset] == nil {
				power := getEffectivePower(placedCard, n.placedPowerIdx, gridIndex)
				if power < 1500 { // Significant penalty for exposing sides lower than Level P (1500)
					totalScore -= (1500 - power) / 10
				}
			}
		}
		return totalScore
	}

	return len(flipped)
}

// SyncOpponentWanted updates the wanted level for a specific player (P1 or P2)
func SyncOpponentWanted(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return false
	}
	pIndex := args[0].Int()
	wantedLevel := args[1].Int()

	Game.mutex.Lock()
	if pIndex >= 0 && pIndex < 2 {
		Game.Players[pIndex].WantedLevel = wantedLevel
	}
	Game.mutex.Unlock()
	return true
}

// PerformAIMove implements the AI's decision-making logic.
func PerformAIMove() {
	Game.mutex.Lock()
	defer Game.mutex.Unlock()

	if Game.Phase != "Active" || Game.Turn != 1 {
		return
	}

	// AI Thinking Delay
	time.Sleep(time.Duration(rand.Intn(1000)+500) * time.Millisecond) // 0.5 to 1.5 seconds

	aiPlayer := &Game.Players[1]
	aiHand := aiPlayer.Decks[aiPlayer.ActiveDeck]

	bestScore := -1
	bestCardIdx := -1
	bestGridIdx := -1

	emptySlots := []int{}
	for i, c := range Game.Board {
		if c == nil {
			emptySlots = append(emptySlots, i)
		}
	}

	if len(emptySlots) == 0 || len(aiHand) == 0 {
		fmt.Println("[AI] No valid moves for AI.")
		return
	}

	// AI Strategy: Iterate through all possible moves and pick the one that maximizes flips
	for cardHandIdx, card := range aiHand {
		for _, gridIdx := range emptySlots {
			// Simulate the move without actually modifying the game board
			score := simulateCaptures(&card, gridIdx)

			// If HardMode is active, also consider the opponent's potential next move
			if Game.HardMode {
				// Create a temporary board state for opponent's turn simulation
				tempBoard := [9]*Card{}
				for i, c := range Game.Board {
					if c != nil {
						tempCard := *c
						tempBoard[i] = &tempCard
					}
				}
				tempCard := card // Copy the AI's card for the temporary board
				tempBoard[gridIdx] = &tempCard

				// Calculate opponent's hand (cards not yet on board)
				opponentHand := []Card{}
				placedCardIDs := make(map[int]bool)
				for _, bc := range tempBoard {
					if bc != nil {
						placedCardIDs[bc.ID] = true
					}
				}
				for _, c := range Game.Players[0].Decks[Game.Players[0].ActiveDeck] {
					if !placedCardIDs[c.ID] {
						opponentHand = append(opponentHand, c)
					}
				}

				// Calculate the maximum score the opponent could achieve after this AI move
				opponentPotential := calculateMaxPlayerPotential(tempBoard, opponentHand, 0, Game.Rules)

				// Adjust AI's score: maximize own score, minimize opponent's potential
				score = score - opponentPotential
			}

			if score > bestScore {
				bestScore = score
				bestCardIdx = cardHandIdx
				bestGridIdx = gridIdx
			}
		}
	}

	if bestCardIdx != -1 && bestGridIdx != -1 {
		chosenCard := aiHand[bestCardIdx]
		Game.Board[bestGridIdx] = &chosenCard
		chosenCard.Owner = 1

		// Apply Visual Tier Effects based on the player's current reputation
		tier, color, _ := calculateTier(Game.Players[1].Reputation)
		Game.Board[bestGridIdx].Tier = tier
		Game.Board[bestGridIdx].GlowColor = color

		fmt.Printf("[AI] %s placed %s at Grid %d (Score: %d)\n", aiPlayer.ID, chosenCard.Name, bestGridIdx, bestScore)
		Game.AIScore = bestScore // Store AI's chosen score for UI feedback

		checkCaptures(&chosenCard, bestGridIdx)
		PlaySound("Select-place-card.mp3")

		// Remove card from AI's hand
		aiPlayer.Decks[aiPlayer.ActiveDeck] = append(aiHand[:bestCardIdx], aiHand[bestCardIdx+1:]...)
		Game.Turn = 0 // Switch turn back to Player 1
		checkWinCondition()
	}
}

// -----------------------------------------------------------------------------
// 6. THE STATE EXPORTER (The Camera)
// -----------------------------------------------------------------------------

// GetGameState sends a secure snapshot of the vault to the JavaScript UI
func GetGameState(this js.Value, args []js.Value) interface{} {
	Game.mutex.RLock()
	defer Game.mutex.RUnlock()

	filter := "all"
	if len(args) > 0 && args[0].Type() == js.TypeString {
		filter = args[0].String()
	}

	state := make(map[string]interface{})

	// Profile & Stats
	if filter == "all" || filter == "profile" {
		state["reputation"] = Game.Players[0].Reputation
		state["mojo"] = Game.Players[0].Mojo
		state["social_rank"] = Game.Players[0].SocialRank
		state["job_role"] = Game.Players[0].JobRole
		state["employer_id"] = Game.Players[0].EmployerClubID
		state["wanted_level"] = Game.Players[0].WantedLevel
		state["jailed_cards"] = Game.Players[0].JailedCards
		state["kidnapped_cards"] = Game.Players[0].KidnappedCards
		state["held_hostage_cards"] = Game.Players[0].HeldHostageCards
		state["rumor_count"] = Game.Players[0].RumorCount
		state["playstyle"] = Game.Players[0].Playstyle
		state["favorite_card_id"] = Game.Players[0].FavoriteCardID
		state["cunning"] = Game.Players[0].Cunning
		state["nurturing"] = Game.Players[0].Nurturing
		state["achievements"] = Game.Players[0].Achievements
	}

	// Combat State
	if filter == "all" || filter == "combat" {
		state["phase"] = Game.Phase
		state["turn"] = Game.Turn
		state["board"] = Game.Board
		state["board_moods"] = Game.BoardMoods
		state["scores"] = Game.Scores
		state["winner"] = Game.Winner
		state["ai_score"] = Game.AIScore
		state["p1_avatar"] = Game.Players[0].AvatarURL
		state["p2_avatar"] = Game.Players[1].AvatarURL
		state["p1_gloat"] = Game.Players[0].GloatMessage
		state["p2_gloat"] = Game.Players[1].GloatMessage
		state["p1_avatar_notice"] = Game.Players[0].AvatarBanNotice
		state["p2_id"] = Game.Players[1].ID
		state["multiplayer"] = Game.Multiplayer
		state["special_fanfare"] = Game.SpecialFanfare
		state["territory_id"] = Game.TerritoryID
		state["active_item_buffs"] = Game.ActiveItemBuffs
		// Expose player-specific stats for accurate client-side power calculations in tooltips
		state["p1_wanted_level"] = Game.Players[0].WantedLevel
		state["p1_cunning"] = Game.Players[0].Cunning
		state["p1_nurturing"] = Game.Players[0].Nurturing
		state["p2_wanted_level"] = Game.Players[1].WantedLevel
		state["p2_cunning"] = Game.Players[1].Cunning
		state["p2_nurturing"] = Game.Players[1].Nurturing
		state["rules"] = Game.Rules
		state["local_player_index"] = Game.LocalPlayerIndex
	}

	// Economy
	if filter == "all" || filter == "economy" {
		state["rewards"] = Game.Rewards
		state["faucet"] = Game.Faucet
		state["vault_low"] = Game.VaultLow
		state["portfolio"] = Game.Players[0].Portfolio
	}

	// Player Assets
	if filter == "all" || filter == "inventory" {
		state["deck"] = Game.Players[Game.Turn].Decks[Game.Players[Game.Turn].ActiveDeck]
		state["inventory"] = Game.Inventory
		state["active_deck"] = Game.Players[0].ActiveDeck
		state["deck_rating"] = calculateDeckRating(Game.Players[0].Decks[Game.Players[0].ActiveDeck])
	}

	// System / Meta
	if filter == "all" || filter == "meta" {
		state["maintenance"] = Game.Maintenance
		state["testing_mode"] = Game.TestingMode
		state["is_admin"] = Game.IsAdmin
		state["show_leaderboard"] = Game.ShowLeaderboard
		state["tournament"] = Game.Tournament
		state["server_load"] = Game.ServerLoad
		state["latency"] = Game.Latency
		state["network_health"] = Game.NetworkHealth
		state["api_base"] = Game.ApiBase
		state["network"] = Game.Network
		state["server_load_color"] = calculateLoadColor(Game.ServerLoad)
		state["master_volume"] = Game.MasterVolume
		state["music_volume"] = Game.MusicVolume
		state["sfx_volume"] = Game.SfxVolume
	}

	stateJSON, err := json.Marshal(state)
	if err != nil {
		fmt.Printf("[ENGINE ERROR] State serialization failed: %v\n", err)
		return nil
	}

	// Use browser's native JSON.parse to efficiently materialize the JS object
	return js.Global().Get("JSON").Call("parse", string(stateJSON))
}

// checkWinCondition checks if the board is full and determines the winner.
func checkWinCondition() {
	Game.mutex.Lock()
	defer Game.mutex.Unlock()

	full := true
	for _, slot := range Game.Board {
		if slot == nil {
			full = false
			break
		}
	}

	if full {
		p1Score, p2Score := 0, 0
		for _, card := range Game.Board {
			if card != nil {
				if card.Owner == 0 {
					p1Score++
				} else {
					p2Score++
				}
			}
		}
		Game.Scores = [2]int{p1Score, p2Score}
		Game.Phase = "Finished"
		if p1Score > p2Score {
			Game.Winner = 0
		} else if p2Score > p1Score {
			Game.Winner = 1
		} else {
			Game.Winner = 2
		}
	}
}

// -----------------------------------------------------------------------------
// 7. BROWSER BRIDGES & AUDIO
// -----------------------------------------------------------------------------

// calculateTier is an internal helper to determine ranking metadata
func calculateTier(rep int) (string, string, bool) {
	tier := "Iron"
	color := "#a19d94" // Iron Grey

	if rep >= 500 {
		tier = "Diamond"
		color = "#b9f2ff" // Diamond Blue
	} else if rep >= 300 {
		tier = "Gold"
		color = "#ffd700" // Classic Gold
	} else if rep >= 150 {
		tier = "Bronze"
		color = "#cd7f32" // Bronze Orange
	}

	return tier, color, rep >= 500
}

// calculateLoadColor determines the hex color based on the match count
func calculateLoadColor(load int) string {
	if load >= 25 {
		return "#ff0000" // Red (Heavy)
	} else if load >= 10 {
		return "#ffff00" // Yellow (Moderate)
	}
	return "#00ff00" // Green (Optimal)
}

// GetServerLoadColor returns the current load color to the UI
func GetServerLoadColor(this js.Value, args []js.Value) interface{} {
	color := calculateLoadColor(Game.ServerLoad)
	status := "Optimal"
	if Game.ServerLoad >= 25 {
		status = "Heavy"
	} else if Game.ServerLoad >= 10 {
		status = "Moderate"
	}
	return map[string]interface{}{"color": color, "status": status}
}

// GetTierInfo returns the tier name and thematic color for a given reputation score.
func GetTierInfo(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return nil
	}
	rep := args[0].Int()
	tier, color, bonus := calculateTier(rep)

	return map[string]interface{}{
		"tier":  tier,
		"color": color,
		"bonus": bonus,
	}
}

func ToggleLeaderboard(this js.Value, args []js.Value) interface{} {
	Game.ShowLeaderboard = !Game.ShowLeaderboard
	Game.mutex.Lock()
	fmt.Printf("[ENGINE] Leaderboard Visible: %v\n", Game.ShowLeaderboard)
	Game.mutex.Unlock()
	PlaySound("click.mp3")
	return Game.ShowLeaderboard
}

func UpdateAmbientMusic() {
	var track string
	var category string

	// 1. Determine Category based on Game State
	if Game.Players[0].Wallet == "" {
		category = "not_connected"
		track = "Not_connected_ambient" // Correct TitleCase
	} else if Game.Phase == "TournamentLobby" {
		category = "tournament_menu"
		// Use one of the high-intensity tournament tracks for the bracket view
		track = "Tournament_game_ambient" // Correct TitleCase
	} else if len(Game.Players[0].Decks[Game.Players[0].ActiveDeck]) < 5 {
		category = "unbuilt"
		track = "Unbuilt_deck_ambient" // Correct TitleCase
	} else if Game.Phase == "Active" {
		category = "match"
		matchPool := []string{
			"2_player_ambient_1", "2_player_ambient_2", "2_player_ambient_3",
			"quick_play_ambient_1", "quick_play_ambient_2", "quick_play_ambient_3",
			"Tournament_game_ambient", "Tournament_game_ambient_2", "Tournament_game_ambient3", "Tournament_game_ambient4", "Tournament_game_ambient5", // Correct TitleCase
		}
		track = matchPool[rand.Intn(len(matchPool))]
	} else {
		category = "menu"
		menuPool := []string{
			"ambient_menu_music_1", "ambient_menu_music_2", "ambient_menu_music_3", "ambient_menu_music_4",
		}
		track = menuPool[rand.Intn(len(menuPool))]
	}

	// 2. Only switch if the category or track has changed to prevent resetting audio on every UI click
	if Game.CurrentAmbientTrack == category && (category == "not_connected" || category == "unbuilt") {
		return
	}

	StopAmbient()
	Game.CurrentAmbientTrack = category
	PlayAmbient(track)
}

// SetMasterVolume updates the global master volume.
func SetMasterVolume(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		Game.MasterVolume = args[0].Float()
		UpdateAmbientMusic() // Re-apply volume to current track
	}
	return nil
}

// SetMusicVolume updates the music volume.
func SetMusicVolume(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		Game.MusicVolume = args[0].Float()
		UpdateAmbientMusic() // Re-apply volume to current track
	}
	return nil
}

// SetSfxVolume updates the sound effects volume.
func SetSfxVolume(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		Game.SfxVolume = args[0].Float()
	}
	return nil
}

func StopAmbient() {
	if Game.AmbientAudio.Type() == js.TypeObject {
		Game.AmbientAudio.Call("pause")
		Game.AmbientAudio.Set("currentTime", 0)
	}
}

// Global variable to store the current ambient audio element for volume control
var currentAmbientAudio js.Value

func PlayAmbient(path string) {
	fullPath := Game.resolvePath("Audio", path)
	audio := js.Global().Get("Audio").New(fullPath)
	if audio.Type() == js.TypeObject {
		audio.Set("loop", true)
		audio.Set("volume", 0.5)                                // Lower volume for background music
		audio.Set("volume", Game.MusicVolume*Game.MasterVolume) // Apply current volume settings
		currentAmbientAudio = audio                             // Store for later volume adjustments
		Game.AmbientAudio = audio                               // Keep for the old reference

		// Play requires a promise handle in modern browsers
		promise := audio.Call("play")
		if promise.Type() == js.TypeObject {
			promise.Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				fmt.Printf("[AUDIO] Ambient blocked by browser: %v\n", args[0])
				return nil
			}))
		}
		fmt.Printf("[AUDIO] Playing Ambient: %s\n", path)
	}
}

func SetAssetBase(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		Game.mutex.Lock()
		Game.AssetBase = args[0].String()
		Game.mutex.Unlock()
		fmt.Printf("[ENGINE] Asset Base URL set to: %s\n", Game.AssetBase)
	}
	return nil
}

func SetApiBase(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		Game.mutex.Lock()
		Game.ApiBase = args[0].String()
		Game.mutex.Unlock()
		fmt.Printf("[ENGINE] API Base URL set to: %s\n", Game.ApiBase)
	}
	return nil
}

// resolvePath unifies asset pathing, handling AssetBase and Category folders.
func (e *Engine) resolvePath(category string, subPath string) string {
	e.mutex.RLock()
	base := e.AssetBase
	e.mutex.RUnlock()

	// category: "Audio", "Images"
	cleanSub := strings.TrimPrefix(subPath, "Public/Assets/")
	cleanSub = strings.TrimPrefix(cleanSub, "Assets/")
	cleanSub = strings.TrimPrefix(cleanSub, category+"/")
	cleanSub = strings.TrimLeft(cleanSub, "/")

	// Handle Audio extension defaulting to .mp3 if no extension is present
	if category == "Audio" && !strings.Contains(cleanSub, ".") {
		cleanSub += ".mp3"
	}

	// Restored case sensitivity: Path must match DIR.md exactly
	resolved := fmt.Sprintf("%sAssets/%s/%s", base, category, cleanSub)
	// fmt.Printf("[ENGINE] Path Resolved: %s -> %s\n", subPath, resolved)
	return resolved
}

func PlaySound(name string) {
	audio := js.Global().Get("Audio").New(Game.resolvePath("Audio", name))
	if audio.Type() == js.TypeObject {
		audio.Set("volume", Game.SfxVolume*Game.MasterVolume) // Apply SFX volume
		audio.Call("play")
	}
}

func SetLocalPlayerIndex(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	idx := args[0].Int()
	if idx < 0 || idx > 1 {
		return false
	}
	Game.LocalPlayerIndex = idx
	fmt.Printf("[ENGINE] Local Player Index set to: %d\n", idx)
	return true
}

// ApplyArtifactToBoard adds a power bonus to a card already placed on the board
func ApplyArtifactToBoard(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return false
	}
	gridIdx := args[0].Int()
	bonus := args[1].Int()

	Game.mutex.Lock()
	defer Game.mutex.Unlock()

	if gridIdx < 0 || gridIdx >= 9 || Game.Board[gridIdx] == nil {
		return false
	}

	Game.Board[gridIdx].Artifact += bonus
	return true
}

func registerFunctions() {
	js.Global().Set("connectWallet", js.FuncOf(connectWallet))
	js.Global().Set("disconnectWallet", js.FuncOf(disconnectWallet))
	js.Global().Set("toggleNetwork", js.FuncOf(toggleNetwork))
	js.Global().Set("SetAvatar", js.FuncOf(SetAvatar))
	js.Global().Set("SendReward", js.FuncOf(SendReward))

	js.Global().Set("ToggleRule", js.FuncOf(ToggleRule))
	js.Global().Set("AddToDeck", js.FuncOf(AddToDeck))
	js.Global().Set("AutoBuildDeck", js.FuncOf(AutoBuildDeck))
	js.Global().Set("PlaySelectSound", js.FuncOf(PlaySelectSound))
	js.Global().Set("SelectDeck", js.FuncOf(SelectDeck))
	js.Global().Set("RemoveFromDeck", js.FuncOf(RemoveFromDeck))
	js.Global().Set("SyncOpponentDeck", js.FuncOf(SyncOpponentDeck))
	js.Global().Set("SyncOpponentProfile", js.FuncOf(SyncOpponentProfile))
	js.Global().Set("SetPlayerReady", js.FuncOf(SetPlayerReady))
	js.Global().Set("SyncFullProfile", js.FuncOf(SyncFullProfile))
	js.Global().Set("SyncPlaystyle", js.FuncOf(SyncPlaystyle))
	js.Global().Set("SyncOpponentWanted", js.FuncOf(SyncOpponentWanted))
	js.Global().Set("SyncPortfolio", js.FuncOf(SyncPortfolio))

	js.Global().Set("StartMatch", js.FuncOf(StartMatch))
	js.Global().Set("PlaceCard", js.FuncOf(PlaceCard))
	js.Global().Set("GetGameState", js.FuncOf(GetGameState)) // Expose the Camera
	js.Global().Set("SetAdminState", js.FuncOf(SetAdminState))
	js.Global().Set("SyncPlayerStats", js.FuncOf(SyncPlayerStats))
	js.Global().Set("SyncServerLoad", js.FuncOf(SyncServerLoad))
	js.Global().Set("SyncLatency", js.FuncOf(SyncLatency))
	js.Global().Set("GetLevelLabelForDisplay", js.FuncOf(GetLevelLabelForDisplay))
	js.Global().Set("TriggerManualSync", js.FuncOf(TriggerManualSync))
	js.Global().Set("SyncTournament", js.FuncOf(SyncTournament))
	js.Global().Set("GetTournamentArchiveBadge", js.FuncOf(GetTournamentArchiveBadge))
	js.Global().Set("SyncMove", js.FuncOf(SyncMove))
	js.Global().Set("SetPhase", js.FuncOf(SetPhase))
	js.Global().Set("GetServerLoadColor", js.FuncOf(GetServerLoadColor))
	js.Global().Set("SetTestingMode", js.FuncOf(SetTestingMode))
	js.Global().Set("SetHardMode", js.FuncOf(SetHardMode))
	js.Global().Set("GetTierInfo", js.FuncOf(GetTierInfo))
	js.Global().Set("SyncRules", js.FuncOf(SyncRules))
	js.Global().Set("SyncRewards", js.FuncOf(SyncRewards))
	js.Global().Set("SyncVaultBalance", js.FuncOf(SyncVaultBalance))
	js.Global().Set("SetMaintenanceState", js.FuncOf(SetMaintenanceState))
	js.Global().Set("ForceActive", js.FuncOf(ForceActive))
	js.Global().Set("SetBoardState", js.FuncOf(SetBoardState))
	js.Global().Set("ResetGame", js.FuncOf(ResetGame))
	js.Global().Set("SetMasterVolume", js.FuncOf(SetMasterVolume))
	js.Global().Set("SetMusicVolume", js.FuncOf(SetMusicVolume))
	js.Global().Set("SetSfxVolume", js.FuncOf(SetSfxVolume))
	js.Global().Set("SetAssetBase", js.FuncOf(SetAssetBase))
	js.Global().Set("SetApiBase", js.FuncOf(SetApiBase))
	js.Global().Set("SetLocalPlayerIndex", js.FuncOf(SetLocalPlayerIndex))
	js.Global().Set("ImportARC72Card", js.FuncOf(ImportARC72Card))
	js.Global().Set("ApplyArtifactToBoard", js.FuncOf(ApplyArtifactToBoard))
	js.Global().Set("PlayCaptureEffect", js.FuncOf(PlayCaptureEffect))
}

// GetTournamentArchiveBadge returns a stylized HTML badge based on verification status
func GetTournamentArchiveBadge(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 { // Now expects 2 arguments: isVerified (bool) and links (JS array)
		return ""
	}
	isVerified := args[0].Bool()
	jsLinks := args[1] // This is a JS array

	var links []string
	if jsLinks.Type() == js.TypeObject && jsLinks.Get("length").Truthy() {
		for i := 0; i < jsLinks.Length(); i++ {
			links = append(links, jsLinks.Index(i).String())
		}
	}

	tooltipText := ""
	if len(links) > 0 {
		tooltipText = "Blockchain Links:\n" + strings.Join(links, "\n")
	} else if isVerified {
		tooltipText = "Archive verified on-chain."
	} else {
		tooltipText = "Data could not be fully verified or reconstructed."
	}

	if isVerified {
		return fmt.Sprintf(`<span class="verified-badge" title="%s" style="font-size: 0.7em; padding: 2px 6px; border: 1px solid var(--neon-green); color: var(--neon-green); border-radius: 4px; margin-left: 10px; background: rgba(63, 185, 80, 0.1); box-shadow: 0 0 10px rgba(63, 185, 80, 0.2); vertical-align: middle;">✓ VERIFIED ARCHIVE</span>`, tooltipText)
	}
	return fmt.Sprintf(`<span style="font-size: 0.7em; padding: 2px 6px; border: 1px solid #ffa657; color: #ffa657; border-radius: 4px; margin-left: 10px; opacity: 0.8; background: rgba(255, 166, 87, 0.1); vertical-align: middle;" title="%s">⚠ PARTIAL DATA</span>`, tooltipText)
}

// ImportARC72Card validates raw JSON metadata and converts it into a playable Card
// It now fetches card details from the backend's centralized cache.
func ImportARC72Card(this js.Value, args []js.Value) interface{} { // Renamed to accept network
	if len(args) < 2 { // Now expects 2 arguments: tokenID and networkName
		fmt.Println("[ENGINE ERROR] ImportARC72Card: Token ID or network name not provided.")
		return false
	}
	tokenID := args[0].Int()
	networkName := args[1].String() // New argument: network name for the card

	Game.mutex.RLock()
	apiBase := Game.ApiBase
	Game.mutex.RUnlock()

	// Make a fetch call to the backend's /api/card-details endpoint
	go func() {
		url := fmt.Sprintf("%s/api/card-details?ids=%d&network=%s", apiBase, tokenID, networkName)
		window := js.Global()

		promise := window.Call("fetch", url)

		// Handle success
		success := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resp := args[0]
			if !resp.Get("ok").Bool() {
				fmt.Printf("[ENGINE ERROR] Failed to fetch card %d from backend: %s\n", tokenID, resp.Get("statusText").String())
				return nil
			}

			jsonPromise := resp.Call("json")
			jsonPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				cardsJS := args[0] // This will be a JS array of card objects
				if cardsJS.Length() == 0 {
					fmt.Printf("[ENGINE ERROR] Backend returned no data for card %d\n", tokenID)
					return nil
				}

				// Convert JS object to Go struct
				cardJSON := cardsJS.Index(0)
				var newCard Card
				newCard.ID = cardJSON.Get("id").Int()
				newCard.Name = cardJSON.Get("name").String()
				newCard.Image = cardJSON.Get("image").String()
				newCard.Rarity = cardJSON.Get("rarity").Float()

				// Extract power values with safety checks
				jsPower := cardJSON.Get("power")
				if jsPower.Type() == js.TypeObject && jsPower.Get("length").Int() >= 4 {
					for i := 0; i < 4; i++ {
						newCard.Power[i] = jsPower.Index(i).Int()
					}
				}

				// Set game-state specific defaults
				newCard.Owner = -1 // Not owned yet
				newCard.Tier = "Iron"
				newCard.GlowColor = "#a19d94"
				newCard.IsCombo = false
				newCard.Image = Game.resolvePath("Images", newCard.Image)

				Game.mutex.Lock()
				Game.Inventory = append(Game.Inventory, newCard)

				// CORRECTION MECHANISM: Scan board for "Syncing..." dummies and update them
				for _, boardCard := range Game.Board {
					if boardCard != nil && boardCard.ID == newCard.ID && boardCard.Name == "Syncing..." {
						boardCard.Name = newCard.Name
						boardCard.Power = newCard.Power
						boardCard.Image = newCard.Image
						boardCard.Rarity = newCard.Rarity
						fmt.Printf("[ENGINE] Corrected dummy card on board: %s\n", boardCard.Name)
					}
				}
				Game.mutex.Unlock()

				fmt.Printf("[ENGINE] Imported %s | ID: %d | Power: %v | Rarity: %.2f\n", newCard.Name, newCard.ID, newCard.Power, newCard.Rarity)
				js.Global().Call("syncUI") // Trigger UI update
				return nil
			}))
			return nil
		})

		// Handle error
		failure := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			err := args[0]
			fmt.Printf("[ENGINE ERROR] Fetching card %d from backend failed: %v\n", tokenID, err.String())
			return nil
		})

		promise.Call("then", success).Call("catch", failure)
	}()

	return true // Return immediately, actual import happens asynchronously
}

// Helper to generate test inventory
func GenerateCard(id int, name string, price float64) Card {
	base := int(price / 10)
	if base < 2 {
		base = 2
	}
	if base > 9 {
		base = 9
	}
	return Card{ID: id, Name: name, Power: [4]int{base, base - 1, base + 1, base}, Image: fmt.Sprintf("Cards/%d.webp", id), Rarity: 1.0}
}

// getLevelFromValue maps a power value (1-2600) to an A-Z letter grade.
func getLevelFromValue(val int) string {
	alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	// Corrected Mapping: 1-100 = Z, 101-200 = Y, ..., 2501-2600 = A
	// Bin 0: 1-100 (Z) -> index 25
	// Bin 1: 101-200 (Y) -> index 24
	// Bin 25: 2501-2600 (A) -> index 0
	bin := (val - 1) / 100
	if bin < 0 {
		bin = 0
	} // Handle 0 or negative values as lowest tier
	if bin > 25 {
		bin = 25
	} // Handle values > 2600 as highest tier
	return string(alphabet[25-bin])
}

// GetLevelLabelForDisplay is a bridge function to expose getLevelFromValue to JS.
func GetLevelLabelForDisplay(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return "Z" // Default for invalid input
	}
	val := args[0].Int()
	return getLevelFromValue(val)
}

// calculateDeckRating computes the [Letter++] rating for a given deck.
func calculateDeckRating(deck []Card) string {
	if len(deck) == 0 {
		return "[Z]"
	}

	maxBin := -1
	// 1. Find the highest card tier (bin) in the deck
	for _, card := range deck {
		highestPower := 0
		for _, p := range card.Power {
			if p > highestPower {
				highestPower = p
			}
		}
		bin := (highestPower - 1) / 100
		if bin < 0 {
			bin = 0
		}
		if bin > maxBin {
			maxBin = bin
		}
	}

	if maxBin == -1 {
		return "[Z]"
	} // Should not happen with non-empty deck

	// 2. Map maxBin to Letter
	baseLetter := getLevelFromValue((maxBin * 100) + 1) // Get the letter for the start of the bin

	// 3. Count how many cards share this highest tier
	plusCount := 0
	for _, card := range deck {
		highestPower := 0
		for _, p := range card.Power {
			if p > highestPower {
				highestPower = p
			}
		}
		bin := (highestPower - 1) / 100
		if bin == maxBin {
			plusCount++
		}
	}

	// 4. Construct Suffix
	suffix := ""
	for i := 0; i < plusCount; i++ {
		suffix += "+"
	}

	return fmt.Sprintf("[%s%s]", baseLetter, suffix)
}

func main() {
	wait := make(chan struct{})

	fmt.Println("-------------------------------------------------")
	fmt.Println(" NFT Seduction WASM Engine: SYNC ONLINE          ")
	fmt.Println(" Camera Exporter & AI Simulation Active          ")
	fmt.Println("-------------------------------------------------")

	// Seed Global Inventory with Demo Assets
	for i, path := range Game.DemoPool {
		if i >= 5 {
			break
		} // Seed first 5 as inventory
		c := GenerateCard(100+i, fmt.Sprintf("Babe %d", i+1), 50.0+float64(i*10))
		c.Image = path
		Game.Inventory = append(Game.Inventory, c)
	}

	registerFunctions()
	<-wait
}
package main

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Matchmaking Logic
type QueueEntry struct {
	ClientID   string    `json:"client_id"`
	Wallet     string    `json:"wallet"`
	Reputation int       `json:"reputation"`
	DeckRating string    `json:"deck_rating"`
	JoinedAt   time.Time `json:"joined_at"`
}

// NonceData stores the nonce value and its creation time for expiration logic.
type NonceData struct {
	Value     string
	CreatedAt time.Time
}

// RateBucket implements the Leaky Bucket state for a single entity (IP).
type RateBucket struct {
	Tokens     float64
	LastUpdate time.Time
}

// HoldingBonus defines a multiplier for a specific reward if a player holds a certain asset.
type HoldingBonus struct {
	HoldingAssetID string  `json:"holding_asset_id"` // The NFT/Token required to be held
	Network        string  `json:"network"`          // Chain to check (VOI or ALGO)
	Multiplier     float64 `json:"multiplier"`       // Reward boost (e.g., 1.1 for 10% bonus)
	MinAmount      uint64  `json:"min_amount"`       // Minimum micro-units required to qualify
}

// Club represents a player-owned organization with specialized shops.
type Club struct {
	ID              string               `json:"id"`
	Name            string               `json:"name"`
	OwnerWallet     string               `json:"owner_wallet"`
	Type            string               `json:"type"`        // Elemental, Tactical, Vitality
	Territories     []string             `json:"territories"` // Supports multiple districts
	RegionName      string               `json:"region_name,omitempty"`
	Treasury        float64              `json:"treasury"`
	Commission      float64              `json:"commission_rate"` // e.g., 0.05 for 5%
	Inventory       map[string]int       `json:"inventory"`       // ItemID -> Quantity
	Staff           map[string]string    `json:"staff"`           // Wallet -> Role (Manager, Security, Clerk)
	ActiveBuffs     map[string]string    `json:"active_buffs"`
	BuffExpirations map[string]time.Time `json:"buff_expirations"` // Key -> Expiration Timestamp
	Members         map[string]time.Time `json:"members"`          // Wallet -> Join Timestamp
	Leases          map[string]*Lease    `json:"leases"`           // LeaseID -> Lease (cards available for rent)
	Mojo            int                  `json:"club_mojo"`        // Unlocks higher tier items
	Jail            map[int]ServerCard   `json:"jail"`             // CardID -> ServerCard (captured cards)

// Lease represents a card available for temporary use within a club.
type Lease struct {
	ID            string    `json:"id"`
	LenderWallet  string    `json:"lender_wallet"`
	CardID        int       `json:"card_id"`
	CardName      string    `json:"card_name"`
	Price         float64   `json:"price"` // Base units of $VBV
	DurationHours int       `json:"duration_hours"`
	ExpiresAt     time.Time `json:"expires_at,omitempty"` // Set once taken
	Borrower      string    `json:"borrower_wallet,omitempty"`
	ClubID        string    `json:"club_id"`
}

// FaceplateStats defines the RPG modifiers provided by cosmetic items.
type FaceplateStats struct {
	MojoBonus    int
	CunningBonus int
}

// FaceplateRegistry maps legacy cosmetic IDs to functional social simulation bonuses.
var FaceplateRegistry = map[string]FaceplateStats{
	"faceplate_neon_vibe":   {MojoBonus: 15, CunningBonus: 5},
	"faceplate_shadow":      {MojoBonus: 5, CunningBonus: 20},
	"faceplate_governor":    {MojoBonus: 50, CunningBonus: 10},
	"faceplate_placeholder": {MojoBonus: 0, CunningBonus: 0},
}

// GetEffectiveCunning returns base cunning plus cosmetic bonuses.
func (p PlayerStats) GetEffectiveCunning() int {
	if p.EquippedFaceplate != "" {
		if fp, exists := FaceplateRegistry[p.EquippedFaceplate]; exists {
			return p.Cunning + fp.CunningBonus
		}
	}
	return p.Cunning
}

// GetEffectiveMojo returns base mojo plus cosmetic bonuses.
func (p PlayerStats) GetEffectiveMojo() int {
	if p.EquippedFaceplate != "" {
		if fp, exists := FaceplateRegistry[p.EquippedFaceplate]; exists {
			return p.Mojo + fp.MojoBonus
		}
	}
	return p.Mojo
}
	LastActivity    time.Time            `json:"last_activity"`    // For Mojo decay tracking
	CreatedAt       time.Time            `json:"created_at"`
}

// UseItemData defines the payload for the "use_item" WebSocket message.
type UseItemData struct {
	ItemID          string `json:"item_id"`
	TargetCardID    int    `json:"target_card_id,omitempty"`    // For card-specific buffs (e.g., Stim, Pledge)
	TargetGridIndex int    `json:"target_grid_index,omitempty"` // For board-specific buffs (e.g., Mood Catalyst)
}

// BailCardData defines the payload for the "bail_card" WebSocket message.
type BailCardData struct {
	CardID  int    `json:"card_id"`
	ClubID  string `json:"club_id"`
	TxID    string `json:"txid"`
	Network string `json:"network"`
}

// Envelope is the standard wrapper for all messages.
type Envelope struct {
	Type    string          `json:"type"`    // "lobby_update", "challenge", "move", "chat", "identity", "vault_update", "rules_update", "rewards_update", "maintenance_update", "ping", "pong", "report_gloat", "admin_notification"
	FromID  string          `json:"from_id"` // Sender ID
	ToID    string          `json:"to_id,omitempty"`
	Payload json.RawMessage `json:"payload"` // Flexible JSON content
}

// ChallengeData handles the matchmaking handshake.
type ChallengeData struct {
	Action string          `json:"action"` // "invite", "accept", "decline", "sync_back"
	Deck   []int           `json:"deck,omitempty"`
	Avatar string          `json:"avatar,omitempty"`
	Gloat  string          `json:"gloat,omitempty"`
	Rules  map[string]bool `json:"rules,omitempty"`
	Wanted int             `json:"wanted_level,omitempty"`
}

// MoveData synchronizes gameplay actions between two human players.
type MoveData struct {
	GridIndex int    `json:"grid_index"`
	CardID    int    `json:"card_id"`
	Power     [4]int `json:"power"`
}

// ReportGloatData captures information about a reported gloat message.
type ReportGloatData struct {
	OpponentClientID string `json:"opponent_client_id"`
	GloatText        string `json:"gloat_text"`
}

// NetworkConfig holds the configuration details for a specific blockchain network.
type NetworkConfig struct {
	NetworkName  string  `json:"network_name"`
	ExplorerURL  string  `json:"explorer_url"`
	IndexerURL   string  `json:"indexer_url"`
	NodeURL      string  `json:"node_url"`
	FaucetURL    string  `json:"faucet_url"`
	AssetID      string  `json:"asset_id"` // The primary game asset ID on this network
	AppID        string  `json:"app_id"`   // The main game smart contract ID on this network
	ChainID      string  `json:"chain_id"` // WalletConnect / CAIP-2 Chain ID
	PowerDivisor float64 `json:"power_divisor"`
	PowerBase    int     `json:"power_base"`
}

// ServerCard mirrors the client Card for verification logic.
type ServerCard struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Power         [4]int    `json:"power"`
	Image         string    `json:"image"`
	Rarity        float64   `json:"rarity"` // Power multiplier based on supply
	Owner         int       `json:"owner"`
	Artifact      int       `json:"artifact"`
	Fatigue       int       `json:"fatigue"`        // 0-100
	Loyalty       int       `json:"loyalty"`        // 0-100
	LastUpdated   time.Time `json:"last_updated"`   // TTL tracking for cache refresh
	MetadataValid bool      `json:"metadata_valid"` // Indicates if metadata was successfully parsed
	Mood          string    `json:"mood"`           // Volatile, Serene, Spirited, Grounded
}

type MetadataAttribute struct {
	TraitType string      `json:"trait_type"`
	Value     interface{} `json:"value"`
}

type ARC72Metadata struct {
	Name       string              `json:"name"`
	Image      string              `json:"image"`
	Attributes []MetadataAttribute `json:"attributes"`
}

// MatchState tracks an ongoing game on the server for win verification.
type MatchState struct {
	P1ID              string                    `json:"p1_id"`
	P2ID              string                    `json:"p2_id"`
	P1Wallet          string                    `json:"p1_wallet"` // Snapshotted for penalty calculation stability
	P2Wallet          string                    `json:"p2_wallet"`
	TournamentMatchID string                    `json:"tournament_match_id,omitempty"` // Link to tournament bracket
	P1Deck            []int                     `json:"p1_deck"`                       // Card IDs in P1's deck
	P1Avatar          string                    `json:"p1_avatar"`
	P1Gloat           string                    `json:"p1_gloat"`
	P2Deck            []int                     `json:"p2_deck"` // Card IDs in P2's deck
	P2Avatar          string                    `json:"p2_avatar"`
	BoardMoods        [9]string                 `json:"board_moods"` // Moods assigned to specific tiles
	P2Gloat           string                    `json:"p2_gloat"`
	Board             [9]*ServerCard            `json:"board"`
	Rules             map[string]bool           `json:"rules"`
	IsFinished        bool                      `json:"is_finished"`
	Spectators        []string                  `json:"spectators"` // Client IDs spectating this match
	P1WantedLevel     int                       `json:"p1_wanted_level"`
	P2WantedLevel     int                       `json:"p2_wanted_level"`
	P1Cunning         int                       `json:"p1_cunning"`
	P1Nurturing       int                       `json:"p1_nurturing"`
	P2Cunning         int                       `json:"p2_cunning"`
	P2Nurturing       int                       `json:"p2_nurturing"`
	FinalScores       [2]int
	CapturedCards     []CapturedCardInfo        `json:"captured_cards,omitempty"` // Tracking for jailing
	TerritoryID       string                    `json:"territory_id,omitempty"`   // The territory where the match is played
	ActiveItemBuffs   map[string]map[string]int `json:"active_item_buffs"` // PlayerID -> ItemID -> MatchesRemaining
	IsBountyMatch     bool
}

// CapturedCardInfo tracks details of a card that was flipped during a match.
type CapturedCardInfo struct {
	CardID                int
	OriginalOwnerWallet   string // Wallet of the player who originally owned the card
	CapturingPlayerWallet string // Wallet of the player who captured the card
	CaptureType           string // "BASIC", "SAME", "POWER_UP", "COMBO"
	GridIndex             int
}

// MatchHistory stores the result of a completed game for reward verification.
type MatchHistory struct {
	WinnerID         string    `json:"winner_id"`
	Opponent         string    `json:"opponent_wallet"`
	TournamentMatchID string    `json:"tournament_id"`
	Scores           [2]int    `json:"scores"`
	Timestamp        time.Time `json:"timestamp"`
	WinnerIndex      int       `json:"winner_index"` // 0 for P1, 1 for P2
	IsBountyMatch    bool      `json:"is_bounty_match,omitempty"`
	BountyReward     float64   `json:"bounty_reward,omitempty"`
}

// PlayerStats tracks the performance and reliability of a player.
type PlayerStats struct {
	Wins              int                `json:"wins"`
	DNFs              int                `json:"dnfs"`
	DisconnectStreak  int                `json:"disconnect_streak"`
	BanExpires        time.Time          `json:"ban_expires"`
	GloatBannedUntil  time.Time          `json:"gloat_banned_until"`
	Reputation        int                `json:"reputation"`
	Mojo              int                `json:"mojo"`                // Social standing for Club unlocks
	SocialRank        string             `json:"social_rank"`         // e.g., "Nobody", "Regular", "Icon"
	JobRole           string             `json:"job_role"`            // Manager, Security, Clerk, Freelancer
	EmployerClubID    string             `json:"employer_id"`         // The club currently paying this user
	Salary            uint64             `json:"salary"`              // Micro-units of  per payment cycle
	LastSalaryPayment time.Time          `json:"last_salary_payment"` // Timestamp of last payment
	Inventory         map[string]int     `json:"inventory"`           // ItemID -> Quantity
	MarketTokens      uint64             `json:"market_tokens"`       // Equity from liquidated loans
	Relationships     map[string]int     `json:"relationships"`       // Character Name -> Score (0-100)
	BestRating        string             `json:"best_rating"`
	Achievements      []string           `json:"achievements"`   // List of unlocked IDs
	Portfolio         map[string]float64 `json:"portfolio"`      // EntityID -> Shares
	WantedLevel       int                `json:"wanted_level"`   // Risk factor for heists
	HeistAttempts     int                `json:"heist_attempts"` // Number of times player attempted a heist
	Cunning           int                `json:"cunning"`        // Success modifier for criminal actions
	Nurturing         int                `json:"nurturing"`      // Success modifier for garden/donations
	JailedCards       map[int]string     `json:"jailed_cards"`   // CardID -> ClubID (cards currently in jail)
	// New fields for Kidnap Gambit
	FavoriteCardID   int            `json:"favorite_card_id"`   // The card ID the player has designated as their favorite
	KidnappedCards   map[int]string `json:"kidnapped_cards"`    // CardID -> VictimWallet (cards player has kidnapped)
	HeldHostageCards map[int]string `json:"held_hostage_cards"` // CardID -> KidnapperWallet (cards player has lost to kidnapping)
	// New fields for Collective NPC Intelligence
	RumorCount     int                 `json:"rumor_count"`     // Number of rumors spread by this player
	Aggressiveness float64             `json:"aggressiveness"`  // 0-1 scale of aggressive play
	RiskTolerance  float64             `json:"risk_tolerance"`  // 0-1 scale of risk-taking
	PreferredRules map[string]int      `json:"preferred_rules"` // Rule name -> usage count
	Moods          map[string]int      `json:"moods"`           // Mood -> count (e.g., "aggressive", "defensive")
	Playstyle      PlaystyleTendencies `json:"playstyle"`       // Observed playstyle tendencies
}

// PlaystyleTendencies captures observed player behaviors for Collective Intelligence.
type PlaystyleTendencies struct {
	Aggressiveness     float64            `json:"aggressiveness"`       // 0.0 - 1.0, higher means more aggressive
	RiskTolerance      float64            `json:"risk_tolerance"`       // 0.0 - 1.0, higher means more risky
	PreferredRules     map[string]float64 `json:"preferred_rules"`      // RuleName -> Weighted Preference Score
	PreferredCardMoods map[string]float64 `json:"preferred_card_moods"` // Mood -> Weighted Preference Score
	FavoriteCardID     int                `json:"favorite_card_id"`     // The card ID set as favorite
	PreferredItems     map[string]float64 `json:"preferred_items"`      // ItemID -> Weighted Usage Score
}

// CardBundle represents a set of items listed together in an auction.
type CardBundle struct {
	CardID      int    `json:"card_id"`
	WeaponID    string `json:"weapon_id,omitempty"`
	FaceplateID string `json:"faceplate_id,omitempty"`
}

// Auction represents a live listing in the Art Gallery.
type Auction struct {
	ID            string     `json:"id"`
	SellerWallet  string     `json:"seller_wallet"`
	SellerName    string     `json:"seller_name"` // Pre-resolved Envoi name
	Bundle        CardBundle `json:"bundle"`
	CurrentBid    uint64     `json:"current_bid"` // Micro-units of 
	HighestBidder string     `json:"highest_bidder"`
	HighestBidderName string `json:"highest_bidder_name"` // Pre-resolved Envoi name
	EndsAt        time.Time  `json:"ends_at"`
	TerritoryID   string     `json:"territory_id"` // For commission distribution
}

// Loan represents a collateralized loan from the Second-Hand Store.
type Loan struct {
	ID               string     `json:"id"`
	BorrowerWallet   string     `json:"borrower_wallet"`
	BorrowerName     string     `json:"borrower_name"` // Pre-resolved Envoi name
	CollateralBundle CardBundle `json:"collateral_bundle"`
	LoanAmount       uint64     `json:"loan_amount"`      // Micro-units of 
	RepaymentAmount  uint64     `json:"repayment_amount"` // LoanAmount + Interest
	DueAt            time.Time  `json:"due_at"`
	Status           string     `json:"status"`       // "active", "repaid", "defaulted"
	TerritoryID      string     `json:"territory_id"` // For commission distribution (Second-Hand Store)
}

// Rumor represents an active rumor affecting an entity's share price.
type Rumor struct {
	ID             string    `json:"id"`
	SpreaderWallet string    `json:"spreader_wallet"`
	TargetWallet   string    `json:"target_wallet"`
	Type           string    `json:"type"`     // "positive", "negative"
	Strength       float64   `json:"strength"` // Multiplier (e.g., 1.1 for +10%, 0.9 for -10%)
	ExpiresAt      time.Time `json:"expires_at"`
}

// KidnapState tracks the details of an active kidnapping for recovery logic.
type KidnapState struct {
	VictimWallet string    `json:"victim_wallet"`
	PerpWallet   string    `json:"perp_wallet"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// TournamentMatch represents a single duel within the bracket.
type TournamentMatch struct {
	ID     string `json:"id"`
	P1     string `json:"p1"` // Wallet Address
	P2     string `json:"p2"` // Wallet Address
	Winner string `json:"winner,omitempty"`
	Round  int    `json:"round"`
}

// TournamentState tracks the progress of an automated event.
type TournamentState struct {
	Active       bool              `json:"active"`
	Matches      []TournamentMatch `json:"matches"`
	CurrentRound int               `json:"current_round"`
	Participants []string          `json:"participants"`
	Pot          float64           `json:"pot"`
	BuyInAmount  float64           `json:"buy_in_amount"`
	IsBuyInMode  bool              `json:"is_buy_in_mode"`
	OpenTime     time.Time         `json:"open_time"` // Registration window start
}

// TournamentSummary represents a finalized tournament for archival.
type TournamentSummary struct {
	ID         string            `json:"id"`
	Timestamp  time.Time         `json:"timestamp"`
	Pot        float64           `json:"pot"`
	Winner     string            `json:"winner"`
	IsVerified bool              `json:"is_verified"`        // Indicates successful blockchain reconstruction
	Checksum   string            `json:"checksum,omitempty"` // SHA256 of full match data
	Links      []string          `json:"links,omitempty"`    // TxIDs for additional match data
	Matches    []TournamentMatch `json:"matches"`
}

// Client represents one connected WebSocket user.
type Client struct {
	conn              *websocket.Conn
	send              chan []byte
	id                string
	isAdmin           bool
	avatarURL         string
	gloat             string
	avatarBanNotice   string
	messageTimestamps []time.Time
	msgMutex          sync.Mutex
	lobby             *Lobby
}

// LinkedWallet represents a non-AVM wallet linked to a primary AVM wallet.
type LinkedWallet struct {
	Address   string    `json:"address"`
	Chain     string    `json:"chain"` // e.g., "ETH", "POLY", "SOL"
	Verified  bool      `json:"verified"`
	Timestamp time.Time `json:"timestamp"` // When it was linked/verified
}

// WalletLinkInfo stores the primary AVM wallet and its linked non-AVM wallets.
type WalletLinkInfo struct {
	PrimaryAVMWallet string         `json:"primary_avm_wallet"`
	Linked           []LinkedWallet `json:"linked_wallets"`
}

// Lobby manages the central state of the arena.
type Lobby struct {
	clients              map[string]*Client
	matches              map[string]*MatchState
	inventory            map[int]ServerCard
	persistentCardCache  map[int]ServerCard
	tournamentPotBonus   float64
	tournamentCache      map[string]*interface{} // Using interface{} for element storage
	paidParticipants     []string
	matchmakingPool      []QueueEntry
	bannedAvatars        map[string]time.Time
	registeredTxIDs      map[string]time.Time
	processingRewards    map[string]time.Time
	processingOnboarding map[string]time.Time
	activeKidnappings    map[int]KidnapState // CardID -> State
	wallets              map[string]string
	clubs                map[string]*Club // Key: ClubID
	blackMarket          []Loan           // Defaulted loans available for purchase
	rumors               map[string]*Rumor
	loans                map[string]*Loan
	auctions             map[string]*Auction
	leaderboard          map[string]PlayerStats
	matchHistory         map[string]MatchHistory
	linkedWallets        map[string]WalletLinkInfo
	vaultAddress         string
	faucetBalance        float64
	rewards              map[string]uint64
	initialRewards       map[string]uint64 // Unscaled base values for all assets in the reward stack
	holdingBonuses       map[string][]HoldingBonus
	initialBaseReward    uint64
	seasonStart          time.Time
	seasonNumber         int
	maxFaucetCapacity    float64
	rewardAssetID        string
	avoiAssetID          string
	baseReward           uint64
	nonces               map[string]NonceData
	availableNetworks    map[string]NetworkConfig
	adminFocusNetwork    string
	maintenanceMode      bool
	maintenanceTime      time.Time
	rateLimits           map[string]time.Time
	httpRateLimits       map[string]*RateBucket
	tournament           TournamentState
	globalSentiment      GlobalSentiment
	register             chan *Client
	unregister           chan *Client
	broadcast            chan []byte
	onboardedWallets     map[string]bool // Tracks wallets that have received an onboarding pack
	onboardingSemaphore  chan struct{}
	envoiCache           map[string]string // Wallet -> Envoi Name Cache
	envoiMutex           sync.RWMutex      // Dedicated lock for name resolution
	SybilSyncComplete    bool // Indicates historical claim state is fully restored
	mutex                sync.RWMutex
}
// processAuctions checks for expired auctions and settles them.
func (l *Lobby) processAuctions() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	now := time.Now()
	for id, a := range l.auctions {
		if now.After(a.EndsAt) {
			if a.HighestBidder != "" {
				commissionMicro := uint64(float64(a.CurrentBid) * 0.10)
				payoutMicro := a.CurrentBid - commissionMicro

				// 2. Settle $VBV rewards (deduct bid from highest bidder, pay seller)
				if l.rewards[a.HighestBidder] >= a.CurrentBid { // Ensure bidder still has funds
					// Distribute commission: 10% to the club owning the Art Gallery territory, else to faucet
					artGalleryClub := l.getClubByTerritoryID(a.TerritoryID) // a.TerritoryID is "the_art_gallery"
					if artGalleryClub != nil {
						artGalleryClub.Treasury += float64(commissionMicro) / 1000000.0
						artGalleryClub.LastActivity = time.Now() // Update club activity
						log.Printf("[AUCTION] Club %s (%s) earned %.2f $VBV commission from auction %s.\n", artGalleryClub.Name, artGalleryClub.ID, float64(commissionMicro)/1000000.0, a.ID)
					} else {
						// Fallback: If no club owns the Art Gallery, the commission goes to the faucet
						l.faucetBalance += float64(commissionMicro) / 1000000.0
						log.Printf("[AUCTION] No club owns 'the_art_gallery'. Commission from auction %s added to faucet.\n", a.ID)
					}

					l.rewards[a.HighestBidder] -= a.CurrentBid
					l.rewards[a.SellerWallet] += payoutMicro

					// 3. Apply dynamic scaling due to faucet balance change
					l.applyDynamicScalingLocked() // Call the locked version

					// Deliver items
					stats := l.leaderboard[a.HighestBidder]
					if stats.Inventory == nil {
						stats.Inventory = make(map[string]int)
					}
					if a.Bundle.CardID != 0 {
						stats.Inventory[fmt.Sprintf("CARD-%d", a.Bundle.CardID)]++
					}
					if a.Bundle.WeaponID != "" {
						stats.Inventory[a.Bundle.WeaponID]++
					}
					if a.Bundle.FaceplateID != "" {
						stats.Inventory[a.Bundle.FaceplateID]++
					}
					l.leaderboard[a.HighestBidder] = stats

					l.logAdminAudit("AUCTION_FINALIZED", a.SellerWallet, fmt.Sprintf("Sold to %s for %.2f", a.HighestBidder, float64(a.CurrentBid)/1000000.0))
					l.sendToClient(l.getClientIDFromWallet(a.HighestBidder), Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🎉 <b>AUCTION WON:</b> You won the auction for %s!"}`, a.Bundle.WeaponID))})
					l.sendToClient(l.getClientIDFromWallet(a.SellerWallet), Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>AUCTION SETTLED:</b> Your item was sold for %.2f $VBV!"}`, float64(payoutMicro)/1000000.0))})
				} else {
					// Bidder no longer has funds, return item to seller
					log.Printf("[AUCTION] Bidder %s for auction %s has insufficient funds. Returning item to seller %s.\n", a.HighestBidder, a.ID, a.SellerWallet)
					stats := l.leaderboard[a.SellerWallet]
					if stats.Inventory == nil {
						stats.Inventory = make(map[string]int)
					}
					if a.Bundle.CardID != 0 {
						stats.Inventory[fmt.Sprintf("CARD-%d", a.Bundle.CardID)]++
					}
					if a.Bundle.WeaponID != "" {
						stats.Inventory[a.Bundle.WeaponID]++
					}
					if a.Bundle.FaceplateID != "" {
						stats.Inventory[a.Bundle.FaceplateID]++
					}
					l.leaderboard[a.SellerWallet] = stats
					l.logAdminAudit("AUCTION_FAILED_BIDDER_FUNDS", a.SellerWallet, fmt.Sprintf("Auction: %s, Bidder: %s", a.ID, a.HighestBidder))
					l.sendToClient(l.getClientIDFromWallet(a.SellerWallet), Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"⚠️ <b>AUCTION FAILED:</b> Bidder had insufficient funds. Item returned."}`))})
				}
			} else {
				// No bids: return items to seller
				stats := l.leaderboard[a.SellerWallet]
				if stats.Inventory == nil {
					stats.Inventory = make(map[string]int)
				}
				if a.Bundle.CardID != 0 {
					stats.Inventory[fmt.Sprintf("CARD-%d", a.Bundle.CardID)]++
				}
				if a.Bundle.WeaponID != "" {
					stats.Inventory[a.Bundle.WeaponID]++
				}
				if a.Bundle.FaceplateID != "" {
					stats.Inventory[a.Bundle.FaceplateID]++
				}
				l.leaderboard[a.SellerWallet] = stats
				l.logAdminAudit("AUCTION_EXPIRED", a.SellerWallet, "No bidders found.")
				l.sendToClient(l.getClientIDFromWallet(a.SellerWallet), Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"😔 <b>AUCTION EXPIRED:</b> No bids received. Item returned."}`)})
			}
			delete(l.auctions, id)
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		}
	}
}
