# Development Plan

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

## Phase 1: Stability & Security Audit
1. Complete: Validated `verifyBuyInTransaction` logic against actual Indexer responses.
3. Ensure `LinkedWallet` verification handles all target chains (ETH/SOL/POL) consistently.

## Phase 2: Performance Optimization
- Refactored `handleVoiOnboarding` logic from `bridge_service.go` to `onboarding_service.go`.
1. Complete: Implemented partial state synchronization in `GetGameState` (Go & JS) to reduce serialization overhead.
2. Complete: Optimized Market Ticker canvas rendering with text measurement caching and viewport culling.