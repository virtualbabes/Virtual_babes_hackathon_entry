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