# Orphan Logic & Fix List

## Current Orphans
- `payoutAddress`: (Resolved) Verification loop in `faucet_service.go:dispatchReward` now checks opt-in for all assets in the stack.
- `adminAvatarBan`: (Resolved) Implemented URL-based banning with immediate enforcement in `handlers_admin.go` and `lobby_manager.go`.
- `checkVaultBalanceOnChain`: Real-time ARC-200 balance monitoring implemented in `oracle_service.go`. (Resolved)
- `processLoans`: Periodic loan processing logic. (Resolved by moving to `economy_processing.go`)
- `handleGetAuctions`, `handleCreateAuction`, `handlePlaceBid`: Moved to `auction_service.go`.
- `handleGetBlackMarket`, `handleSellMarketTokens`, `handleBuyBlackMarket`: Moved to `black_market_service.go`.
- `handleVoiOnboarding`: Moved to `onboarding_service.go`.
- `handleTradeShares`: (Resolved) Logic delegated to `market_service.go`.
- `handleHeist`: (Resolved) Logic delegated to `club_service.go`.
- `handleCreateClub`: (Resolved) Logic delegated to `club_service.go`.
- `handleJoinClub`: (Resolved) Logic delegated to `club_service.go`.
- `handleHirePlayer`: (Resolved) Logic delegated to `employment_service.go`.
- `handlePurchaseTerritory`: (Resolved) Logic delegated to `club_service.go`.
- `handleGetLoans`, `handleTakeLoan`, `handleRepayLoan`: Moved to `loan_service.go`.
- `handleReward`, `dispatchReward`: Moved to `faucet_service.go`.
- `onboardedWallets`: Populated via `loadOnboardedWalletsFromIndexer` in `oracle_service.go`. (Resolved)
- `handleHirePlayer`, `handleSetSalary`: Moved to `employment_service.go`.
- `initialRewards`: (Added/Resolved) Tracks unscaled reward targets in `common_types.go` for stack-wide dynamic scaling.
- `Cunning/Nurturing WASM`: (Resolved) Player struct updated and side-effects implemented in `main.go`.
- `Lease Logic`: (Added/Resolved) Implemented card leasing with revenue sharing in `club_service.go`.
- `Auction Card Escrow`: (Resolved) Fixed `CardID` missing from escrow logic in `auction_service.go` and `economy_processing.go`.
- `Faceplates`: (Resolved) Mojo/Cunning bonuses implemented in `common_types.go` and linked to `CalculateReputation`.

Identified Orphans
payoutAddress: Variable initialized in app.js (line ~30: let payoutAddress = localStorage.getItem("vbabes_payout_address") || null;) for user payout settings. Logic is spread across app.js (e.g., savePayoutAddress, updatePayoutUI) and server.go/faucet_service.go (e.g., handleReward for on-chain payouts). Issue: No unified validation or opt-in check for $VBV on Voi before payouts; risks failed transactions. Fix: Consolidate in faucet_service.go with indexer verification.
adminAvatarBan: Referenced in app.js (e.g., adminGloatBan function calls), but implementation in handlers_admin.go (adminBanWallet) lacks avatar-specific logic. Issue: No dedicated avatar banning; relies on general wallet bans. Fix: Add avatar URL banning in handlers_admin.go with image hash checks.
Lost Synergy
Refactored Services: Many handlers moved to dedicated files (e.g., auction_service.go, loan_service.go), but some integrations are incomplete. For example, processLoans and processAuctions in economy_processing.go run via tickers, but UI updates (e.g., in syncUI) don't reflect real-time changes, leading to stale portfolio views.
Achievement System: (Resolved) Profile metadata now fully syncs into WASM state; UI Trophy button displays live count.
Portfolio & Map Views: (Resolved) Portfolios now use persistent wallet keys and Envoi names are resolved in the UI.
TODO/FIXME Comments: Found in SCSS (e.g., /* TODO: Implement styles */ in _social.scss and _territory.scss), but no active TODOs in Go/JS code. Indicates placeholders for future features (e.g., dynamic trophy glows) are pending implementation.
