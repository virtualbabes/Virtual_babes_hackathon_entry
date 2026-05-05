# Orphan Logic & Fix List

## Current Orphans
- `payoutAddress`: Initialized in `app.js` but logic for ensuring user opt-in to $VBV on Voi before payout is distributed across `app.js` and `server.go`. Needs unification.
- `adminAvatarBan`: Function mentioned in logs in `app.js` but actual implementation details in `handlers_admin.go` need verification.
- `checkVaultBalanceOnChain`: Real-time ARC-200 balance monitoring implemented in `oracle_service.go`. (Resolved)
- `processLoans`: Periodic loan processing logic. (Resolved by moving to `economy_processing.go`)
- `handleGetAuctions`, `handleCreateAuction`, `handlePlaceBid`: Moved to `auction_service.go`.
- `handleGetBlackMarket`, `handleSellMarketTokens`, `handleBuyBlackMarket`: Moved to `black_market_service.go`.
- `handleVoiOnboarding`: Moved to `onboarding_service.go`.
- `handleGetLoans`, `handleTakeLoan`, `handleRepayLoan`: Moved to `loan_service.go`.
- `handleReward`, `dispatchReward`: Moved to `faucet_service.go`.
- `onboardedWallets`: Populated via `loadOnboardedWalletsFromIndexer` in `oracle_service.go`. (Resolved)
- `handleHirePlayer`, `handleSetSalary`: Moved to `employment_service.go`.

Identified Orphans
payoutAddress: Variable initialized in app.js (line ~30: let payoutAddress = localStorage.getItem("vbabes_payout_address") || null;) for user payout settings. Logic is spread across app.js (e.g., savePayoutAddress, updatePayoutUI) and server.go/faucet_service.go (e.g., handleReward for on-chain payouts). Issue: No unified validation or opt-in check for $VBV on Voi before payouts; risks failed transactions. Fix: Consolidate in faucet_service.go with indexer verification.
adminAvatarBan: Referenced in app.js (e.g., adminGloatBan function calls), but implementation in handlers_admin.go (adminBanWallet) lacks avatar-specific logic. Issue: No dedicated avatar banning; relies on general wallet bans. Fix: Add avatar URL banning in handlers_admin.go with image hash checks.
Lost Synergy
Refactored Services: Many handlers moved to dedicated files (e.g., auction_service.go, loan_service.go), but some integrations are incomplete. For example, processLoans and processAuctions in economy_processing.go run via tickers, but UI updates (e.g., in syncUI) don't reflect real-time changes, leading to stale portfolio views.
Achievement System: unlockAchievement in achievement_service.go broadcasts updates, but openTrophyView in app.js uses mock data instead of pulling from PlayerStats.Achievements. Synergy Loss: Trophies aren't dynamically populated from backend state.
Portfolio & Map Views: openPortfolioView and openWorldMap use classes like portfolio-view and map-grid-3d, but backend state (e.g., PlayerStats.Portfolio) isn't fully synced in real-time, causing discrepancies in holdings display.
TODO/FIXME Comments: Found in SCSS (e.g., /* TODO: Implement styles */ in _social.scss and _territory.scss), but no active TODOs in Go/JS code. Indicates placeholders for future features (e.g., dynamic trophy glows) are pending implementation.
