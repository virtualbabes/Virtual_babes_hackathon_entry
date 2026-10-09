# Game Mechanics Index — NFT-Seduction

> Auto-generated scan of repository source (`.go` + `Public/js`), excluding `node_modules` and `llama`.

- **Go source files scanned:** 58
- **UI panels scanned:** 28
- **HTTP routes discovered:** 129
- **Core struct types:** 200
- **Exported functions/methods:** 439


---

## 1. Subsystem / Mechanic Inventory (by game aspect)

### AI Citizens & Lineage

- `ai_citizen_engine.go`
- `ai_citizen_persist_test.go`

### Achievements

- `achievement_handlers.go`
- `achievement_service.go`

### Admin Panel

- `handlers_admin.go`

### Auctions

- `auction_service.go`

### Battle/Combat

- `battle_service.go`

### Black Market

- `black_market_service.go`

### Bootstrap/Main

- `main.go`

### Careers/Employment

- `career.go`
- `employment_service.go`
- `rival_career_engine.go`

### Chain Bridges (ETH/Base/Arbitrum)

- `ethereum_client.go`

### Clubs/Territories/Regions

- `club_service.go`
- `club_service_test.go`

### Core Types

- `backend_types.go`
- `common_types.go`
- `common_types_wasm.go`

### Counterfeit

- `counterfeit_service.go`

### Courthouse

- `courthouse_service.go`

### Creator Store

- `creator_store_service.go`

### Criminality/Justice

- `handlers_criminality.go`

### DEX Path

- `nautilus_dex_path.go`

### Economy/VBV

- `economy_audit.go`
- `economy_bootstrap.go`
- `economy_persistence.go`
- `economy_processing.go`
- `economy_service.go`
- `economy_telemetry.go`

### Entity Market / $VBV

- `entity_investment_service.go`

### Faucet Vault

- `faucet_service.go`

### HTTP Server/Routes

- `server.go`

### Identity Bridge

- `bridge_service.go`
- `identity_bridge.go`
- `identity_bridge_test.go`

### Items/Inventory

- `item_service.go`

### Justice/Courts

- `justice_handlers.go`
- `justice_service.go`

### Life Assets (Pets/Vehicles/World)

- `asset_life_engine.go`

### Loans/Debt

- `loan_service.go`

### Lobby/Orchestration

- `lobby_manager.go`

### Market/AMM

- `market_service.go`
- `market_service_test.go`

### Narrative

- `narrative_service.go`

### Onboarding

- `onboarding_service.go`

### Oracle/Price Feeds

- `oracle_service.go`

### Player Profiles

- `player_service.go`

### Public API

- `handlers_public.go`

### Rate Limiter

- `rate_limiter.go`

### Redemption Gateway

- `redemption_gateway.go`

### Resilience/Reliability

- `resilience_utils.go`

### Rivalry Matrix

- **OWNER OF RECORD: `AI-Brain/Rivalry-Matrix.md`** — the three matrices, their weights, their
  gates and their endpoints. This index lists the CODE only; the architecture lives there.
- `rivalry_engine.go` — region/territory asset signatures and rivalry resolution
- `rivalry_handlers.go` — the `/api/rivalry/*` HTTP surface
- `rival_career_engine.go` — the career-pair matrix (`rivalPairTable`) and cross-career XP
- `career_path.go` — the PLAYER path matrix (justice/criminal/neutral), the civil rank and the
  promotion/demotion lifecycle
- `theme_engine.go` — the 10 world-dynamics weights

### Rumours

- `handlers_rumor.go`

### Shops

- `shop_registry.go`

### Tournaments

- `tournament_manager.go`

### Treasure Hunts & Events

- `seasonal_event_engine.go`

### Underworld Contracts

- `underworld_contracts.go`


---
## 2. User-Facing Mechanics (HTTP API surface)

### HTTP Server/Routes

| Method | Path | Handler | UI Panel(s) |
|---|---|---|---|
| GET/ANY | `/` | ? | `admin.js`, `ai_citizens.js`, `asset_viewer.js`, `audio.js`, `audio_context.js`, `bounty_tracker.js`, `config.js`, `creator_store.js`, `creator_storefront.js`, `criminality.js`, `deck.js`, `early_tasks.js`, `economy.js`, `game.js`, `investment_dashboard.js`, `justice_dashboard.js`, `leaderboard.js`, `leaderboard_region.js`, `life_assets.js`, `network.js`, `particles.js`, `rivalry.js`, `rivalry_viewer.js`, `seasonal_events.js`, `ui.js`, `utils.js`, `wallet.js`, `world_events.js` |
| GET/ANY | `/api/achievement-stats` | ? | **NONE** |
| GET/ANY | `/api/achievement/unlock` | ? | **NONE** |
| GET/ANY | `/api/achievements` | ? | **NONE** |
| GET/ANY | `/api/admin/asset-forfeiture` | ? | `admin.js` |
| GET/ANY | `/api/admin/avatar-ban` | ? | `admin.js` |
| GET/ANY | `/api/admin/commission-audit` | ? | `admin.js` |
| GET/ANY | `/api/admin/district-tax-audit` | ? | `admin.js` |
| GET/ANY | `/api/admin/dlc-registry` | ? | `admin.js` |
| GET/ANY | `/api/admin/dlc-registry/restock` | ? | `admin.js` |
| GET/ANY | `/api/admin/dlc-registry/update` | ? | `admin.js` |
| GET/ANY | `/api/admin/emergency-shutdown` | ? | `admin.js` |
| GET/ANY | `/api/admin/export-logs` | ? | `admin.js` |
| GET/ANY | `/api/admin/force-payout` | ? | `admin.js` |
| GET/ANY | `/api/admin/gloat-ban` | ? | **NONE** |
| GET/ANY | `/api/admin/ledger-audit` | ? | `admin.js` |
| GET/ANY | `/api/admin/logs` | ? | `admin.js` |
| GET/ANY | `/api/admin/mutation-audit` | ? | `admin.js` |
| GET/ANY | `/api/admin/network/add` | ? | **NONE** |
| GET/ANY | `/api/admin/open-registration` | ? | **NONE** |
| GET/ANY | `/api/admin/sanity-check` | ? | **NONE** |
| GET/ANY | `/api/admin/season-rollover` | ? | `admin.js` |
| GET/ANY | `/api/admin/set-admin-focus-network` | ? | `admin.js` |
| GET/ANY | `/api/admin/simulate-load` | ? | `admin.js` |
| GET/ANY | `/api/admin/simulate-mojo-decay` | ? | `admin.js` |
| GET/ANY | `/api/admin/simulate-mutation-failure` | ? | `admin.js` |
| GET/ANY | `/api/admin/simulate-mutation-success` | ? | `admin.js` |
| GET/ANY | `/api/admin/simulate-tournament` | ? | `admin.js` |
| GET/ANY | `/api/admin/start-tournament` | ? | **NONE** |
| GET/ANY | `/api/admin/tax-audit` | ? | `admin.js` |
| GET/ANY | `/api/admin/update-power` | ? | `admin.js` |
| GET/ANY | `/api/ai/citizens/adopt` | ? | **NONE** |
| GET/ANY | `/api/ai/citizens/business/spawn` | http.StatusBadRequest | **NONE** |
| GET/ANY | `/api/ai/citizens/challenge` | ? | **NONE** |
| GET/ANY | `/api/ai/citizens/free-agents` | ? | **NONE** |
| GET/ANY | `/api/ai/citizens/list` | ? | **NONE** |
| GET/ANY | `/api/ai/citizens/release` | ? | **NONE** |
| GET/ANY | `/api/ai/citizens/spawn` | http.StatusBadRequest | **NONE** |
| GET/ANY | `/api/ai/citizens/stats` | ? | **NONE** |
| GET/ANY | `/api/auctions` | ? | `economy.js` |
| GET/ANY | `/api/ban-player` | ? | `admin.js` |
| GET/ANY | `/api/black-market/buy` | ? | `economy.js` |
| GET/ANY | `/api/black-market/buy-stolen` | ? | **NONE** |
| GET/ANY | `/api/black-market/fence-goods` | ? | **NONE** |
| GET/ANY | `/api/black-market/sell-tokens` | ? | **NONE** |
| GET/ANY | `/api/bridge/onboard` | ? | `wallet.js` |
| GET/ANY | `/api/card-details` | ? | `deck.js` |
| GET/ANY | `/api/card-stats` | ? | **NONE** |
| GET/ANY | `/api/career/progress` | func(next http.HandlerFunc | `rivalry.js` |
| GET/ANY | `/api/claim/dividends` | http.StatusMethodNotAllowed | `investment_dashboard.js` |
| GET/ANY | `/api/contracts/assign` | ? | **NONE** |
| GET/ANY | `/api/contracts/list` | ? | **NONE** |
| GET/ANY | `/api/counterfeit/detect` | http.StatusMethodNotAllowed | **NONE** |
| GET/ANY | `/api/counterfeit/generate` | http.StatusMethodNotAllowed | **NONE** |
| GET/ANY | `/api/courthouse/reset` | ? | `criminality.js` |
| GET/ANY | `/api/creator/store/deactivate/` | ? | **NONE** |
| GET/ANY | `/api/creator/store/product/` | ? | **NONE** |
| GET/ANY | `/api/creator/store/products` | ? | `creator_storefront.js` |
| GET/ANY | `/api/creator/store/profile/` | ? | **NONE** |
| GET/ANY | `/api/creator/store/purchase/` | ? | **NONE** |
| GET/ANY | `/api/creator/store/rate/` | http.StatusBadRequest | **NONE** |
| GET/ANY | `/api/creator/store/reactivate/` | ? | **NONE** |
| GET/ANY | `/api/creator/store/resell` | http.StatusBadRequest | **NONE** |
| GET/ANY | `/api/creator/store/royalty-history` | ? | **NONE** |
| GET/ANY | `/api/criminality/cyber-intercept` | ? | **NONE** |
| GET/ANY | `/api/events/create` | ? | **NONE** |
| GET/ANY | `/api/events/enter` | ? | **NONE** |
| GET/ANY | `/api/faction/shop/` | ? | `rivalry.js` |
| GET/ANY | `/api/identity/link` | ? | **NONE** |
| GET/ANY | `/api/identity/resolve` | ? | **NONE** |
| GET/ANY | `/api/identity/snapshot` | ? | **NONE** |
| GET/ANY | `/api/identity/unlink` | ? | **NONE** |
| GET/ANY | `/api/invest/dividends/history` | http.StatusMethodNotAllowed | `investment_dashboard.js` |
| GET/ANY | `/api/invest/entity` | http.StatusMethodNotAllowed | `investment_dashboard.js` |
| GET/ANY | `/api/invest/portfolio` | http.StatusMethodNotAllowed | `investment_dashboard.js` |
| GET/ANY | `/api/justice/award-card` | ? | **NONE** |
| GET/ANY | `/api/justice/bounty-board` | ? | **NONE** |
| GET/ANY | `/api/justice/capture-bounty` | ? | `bounty_tracker.js`, `justice_dashboard.js` |
| GET/ANY | `/api/justice/dashboard` | ? | `justice_dashboard.js` |
| GET/ANY | `/api/justice/use-rep-shield` | ? | **NONE** |
| GET/ANY | `/api/justice/use-truth-serum` | ? | `justice_dashboard.js` |
| GET/ANY | `/api/leaderboard` | func(next http.HandlerFunc | `leaderboard.js` |
| GET/ANY | `/api/loans` | ? | **NONE** |
| GET/ANY | `/api/loans/repay` | ? | **NONE** |
| GET/ANY | `/api/loans/take` | ? | **NONE** |
| GET/ANY | `/api/maintenance-mode` | ? | `admin.js` |
| GET/ANY | `/api/match/wager` | ? | `game.js` |
| GET/ANY | `/api/pets` | http.StatusMethodNotAllowed | `life_assets.js` |
| GET/ANY | `/api/pets/breed` | http.StatusMethodNotAllowed | **NONE** |
| GET/ANY | `/api/pets/spawn` | http.StatusMethodNotAllowed | **NONE** |
| GET/ANY | `/api/re-sync-stats` | ? | **NONE** |
| GET/ANY | `/api/refill-vault` | ? | **NONE** |
| GET/ANY | `/api/regions` | http.StatusMethodNotAllowed | **NONE** |
| GET/ANY | `/api/report-player` | ? | `criminality.js` |
| GET/ANY | `/api/reset-stats` | ? | `admin.js` |
| GET/ANY | `/api/reward` | ? | `admin.js`, `wallet.js` |
| GET/ANY | `/api/reward/add` | ? | `admin.js` |
| GET/ANY | `/api/reward/remove` | ? | `admin.js` |
| GET/ANY | `/api/reward/update-asset` | ? | **NONE** |
| GET/ANY | `/api/reward/update-base` | ? | **NONE** |
| GET/ANY | `/api/rivalry/action` | ? | **NONE** |
| GET/ANY | `/api/rivalry/detect` | http.StatusMethodNotAllowed | `rivalry_viewer.js` |
| GET/ANY | `/api/rivalry/list` | http.StatusMethodNotAllowed | `rivalry_viewer.js` |
| GET/ANY | `/api/rivalry/recompute` | http.StatusMethodNotAllowed | **NONE** |
| GET/ANY | `/api/rivalry/request` | ? | `rivalry.js` |
| GET/ANY | `/api/rivalry/resolve` | ? | `rivalry_viewer.js` |
| GET/ANY | `/api/rivalry/state` | ? | **NONE** |
| GET/ANY | `/api/season/admin/create-event` | http.StatusMethodNotAllowed | `seasonal_events.js` |
| GET/ANY | `/api/season/admin/end-event` | http.StatusMethodNotAllowed | **NONE** |
| GET/ANY | `/api/season/admin/update-reward-pool` | http.StatusMethodNotAllowed | **NONE** |
| GET/ANY | `/api/season/events` | http.StatusMethodNotAllowed | **NONE** |
| GET/ANY | `/api/season/events/join` | http.StatusMethodNotAllowed | **NONE** |
| GET/ANY | `/api/season/events/reward` | http.StatusMethodNotAllowed | **NONE** |
| GET/ANY | `/api/season/history` | ? | `leaderboard.js` |
| GET/ANY | `/api/season/status` | http.StatusMethodNotAllowed | **NONE** |
| GET/ANY | `/api/system-message` | ? | `admin.js` |
| GET/ANY | `/api/tournament/history` | ? | `leaderboard.js` |
| GET/ANY | `/api/tournament/register` | ? | **NONE** |
| GET/ANY | `/api/treasure/claim` | ? | **NONE** |
| GET/ANY | `/api/treasure/spawn` | ? | **NONE** |
| GET/ANY | `/api/underworld/contracts` | ? | `criminality.js` |
| GET/ANY | `/api/update-rules` | ? | `admin.js` |
| GET/ANY | `/api/v1/redemption_gateway` | ? | **NONE** |
| GET/ANY | `/api/vehicles` | http.StatusMethodNotAllowed | `life_assets.js` |
| GET/ANY | `/api/vehicles/spawn` | http.StatusMethodNotAllowed | **NONE** |
| GET/ANY | `/api/world-content` | http.StatusMethodNotAllowed | `life_assets.js` |
| GET/ANY | `/api/world-content/create` | http.StatusMethodNotAllowed | **NONE** |
| GET/ANY | `/api/world-content/deploy` | http.StatusMethodNotAllowed | **NONE** |
| GET/ANY | `/ws` | ? | `bounty_tracker.js`, `network.js` |


---
## 3. Coverage Gaps — Mechanics NOT yet surfaced in UI

68 of 129 routes have no matching UI panel:

- `GET/ANY /api/achievement-stats`  (handler `?`, in `server.go`)
- `GET/ANY /api/achievement/unlock`  (handler `?`, in `server.go`)
- `GET/ANY /api/achievements`  (handler `?`, in `server.go`)
- `GET/ANY /api/admin/gloat-ban`  (handler `?`, in `server.go`)
- `GET/ANY /api/admin/network/add`  (handler `?`, in `server.go`)
- `GET/ANY /api/admin/open-registration`  (handler `?`, in `server.go`)
- `GET/ANY /api/admin/sanity-check`  (handler `?`, in `server.go`)
- `GET/ANY /api/admin/start-tournament`  (handler `?`, in `server.go`)
- `GET/ANY /api/ai/citizens/adopt`  (handler `?`, in `server.go`)
- `GET/ANY /api/ai/citizens/business/spawn`  (handler `http.StatusBadRequest`, in `server.go`)
- `GET/ANY /api/ai/citizens/challenge`  (handler `?`, in `server.go`)
- `GET/ANY /api/ai/citizens/free-agents`  (handler `?`, in `server.go`)
- `GET/ANY /api/ai/citizens/list`  (handler `?`, in `server.go`)
- `GET/ANY /api/ai/citizens/release`  (handler `?`, in `server.go`)
- `GET/ANY /api/ai/citizens/spawn`  (handler `http.StatusBadRequest`, in `server.go`)
- `GET/ANY /api/ai/citizens/stats`  (handler `?`, in `server.go`)
- `GET/ANY /api/black-market/buy-stolen`  (handler `?`, in `server.go`)
- `GET/ANY /api/black-market/fence-goods`  (handler `?`, in `server.go`)
- `GET/ANY /api/black-market/sell-tokens`  (handler `?`, in `server.go`)
- `GET/ANY /api/card-stats`  (handler `?`, in `server.go`)
- `GET/ANY /api/contracts/assign`  (handler `?`, in `server.go`)
- `GET/ANY /api/contracts/list`  (handler `?`, in `server.go`)
- `GET/ANY /api/counterfeit/detect`  (handler `http.StatusMethodNotAllowed`, in `server.go`)
- `GET/ANY /api/counterfeit/generate`  (handler `http.StatusMethodNotAllowed`, in `server.go`)
- `GET/ANY /api/creator/store/deactivate/`  (handler `?`, in `server.go`)
- `GET/ANY /api/creator/store/product/`  (handler `?`, in `server.go`)
- `GET/ANY /api/creator/store/profile/`  (handler `?`, in `server.go`)
- `GET/ANY /api/creator/store/purchase/`  (handler `?`, in `server.go`)
- `GET/ANY /api/creator/store/rate/`  (handler `http.StatusBadRequest`, in `server.go`)
- `GET/ANY /api/creator/store/reactivate/`  (handler `?`, in `server.go`)
- `GET/ANY /api/creator/store/resell`  (handler `http.StatusBadRequest`, in `server.go`)
- `GET/ANY /api/creator/store/royalty-history`  (handler `?`, in `server.go`)
- `GET/ANY /api/criminality/cyber-intercept`  (handler `?`, in `server.go`)
- `GET/ANY /api/events/create`  (handler `?`, in `server.go`)
- `GET/ANY /api/events/enter`  (handler `?`, in `server.go`)
- `GET/ANY /api/identity/link`  (handler `?`, in `server.go`)
- `GET/ANY /api/identity/resolve`  (handler `?`, in `server.go`)
- `GET/ANY /api/identity/snapshot`  (handler `?`, in `server.go`)
- `GET/ANY /api/identity/unlink`  (handler `?`, in `server.go`)
- `GET/ANY /api/justice/award-card`  (handler `?`, in `server.go`)
- `GET/ANY /api/justice/bounty-board`  (handler `?`, in `server.go`)
- `GET/ANY /api/justice/use-rep-shield`  (handler `?`, in `server.go`)
- `GET/ANY /api/loans`  (handler `?`, in `server.go`)
- `GET/ANY /api/loans/repay`  (handler `?`, in `server.go`)
- `GET/ANY /api/loans/take`  (handler `?`, in `server.go`)
- `GET/ANY /api/pets/breed`  (handler `http.StatusMethodNotAllowed`, in `server.go`)
- `GET/ANY /api/pets/spawn`  (handler `http.StatusMethodNotAllowed`, in `server.go`)
- `GET/ANY /api/re-sync-stats`  (handler `?`, in `server.go`)
- `GET/ANY /api/refill-vault`  (handler `?`, in `server.go`)
- `GET/ANY /api/regions`  (handler `http.StatusMethodNotAllowed`, in `server.go`)
- `GET/ANY /api/reward/update-asset`  (handler `?`, in `server.go`)
- `GET/ANY /api/reward/update-base`  (handler `?`, in `server.go`)
- `GET/ANY /api/rivalry/action`  (handler `?`, in `server.go`)
- `GET/ANY /api/rivalry/recompute`  (handler `http.StatusMethodNotAllowed`, in `server.go`)
- `GET/ANY /api/rivalry/state`  (handler `?`, in `server.go`)
- `GET/ANY /api/season/admin/end-event`  (handler `http.StatusMethodNotAllowed`, in `server.go`)
- `GET/ANY /api/season/admin/update-reward-pool`  (handler `http.StatusMethodNotAllowed`, in `server.go`)
- `GET/ANY /api/season/events`  (handler `http.StatusMethodNotAllowed`, in `server.go`)
- `GET/ANY /api/season/events/join`  (handler `http.StatusMethodNotAllowed`, in `server.go`)
- `GET/ANY /api/season/events/reward`  (handler `http.StatusMethodNotAllowed`, in `server.go`)
- `GET/ANY /api/season/status`  (handler `http.StatusMethodNotAllowed`, in `server.go`)
- `GET/ANY /api/tournament/register`  (handler `?`, in `server.go`)
- `GET/ANY /api/treasure/claim`  (handler `?`, in `server.go`)
- `GET/ANY /api/treasure/spawn`  (handler `?`, in `server.go`)
- `GET/ANY /api/v1/redemption_gateway`  (handler `?`, in `server.go`)
- `GET/ANY /api/vehicles/spawn`  (handler `http.StatusMethodNotAllowed`, in `server.go`)
- `GET/ANY /api/world-content/create`  (handler `http.StatusMethodNotAllowed`, in `server.go`)
- `GET/ANY /api/world-content/deploy`  (handler `http.StatusMethodNotAllowed`, in `server.go`)


---
## 4. Core Entity Types (structs)

- `AICitizen` — _ai_citizen_engine.go_
- `AICitizenEngine` — _ai_citizen_engine.go_
- `APIResponseJSON` — _justice_handlers.go_
- `ARC72Metadata` — _common_types.go_
- `ARC72Metadata` — _common_types_wasm.go_
- `AchievementService` — _achievement_service.go_
- `ActiveBuff` — _main.go_
- `AlertEngine` — _main.go_
- `AlertPayload` — _main.go_
- `ApplyRepShieldRequest` — _justice_handlers.go_
- `AssetSignature` — _backend_types.go_
- `Auction` — _common_types.go_
- `Auction` — _common_types_wasm.go_
- `AuctionService` — _auction_service.go_
- `AuditLogEntry` — _economy_audit.go_
- `AuthoritativeFrame` — _common_types.go_
- `AuthoritativeFrame` — _common_types_wasm.go_
- `AwardJusticeCardRequest` — _justice_handlers.go_
- `BailCardData` — _common_types.go_
- `BailCardData` — _common_types_wasm.go_
- `BlackMarketService` — _black_market_service.go_
- `BootstrapEngine` — _economy_bootstrap.go_
- `BountyItemJSON` — _justice_handlers.go_
- `BountyTargetInfo` — _justice_service.go_
- `BridgeResponse` — _main.go_
- `CaptureBountyRequest` — _justice_handlers.go_
- `CapturedCardInfo` — _common_types.go_
- `CapturedCardInfo` — _common_types_wasm.go_
- `Card` — _main.go_
- `CardBuffState` — _justice_service.go_
- `CardBundle` — _common_types.go_
- `CardBundle` — _common_types_wasm.go_
- `CareerService` — _career.go_
- `CareerXP` — _common_types_wasm.go_
- `CareerXP` — _rival_career_engine.go_
- `ChallengeData` — _common_types.go_
- `ChallengeData` — _common_types_wasm.go_
- `Client` — _backend_types.go_
- `ClientRedirectManager` — _main.go_
- `ClientReplayEngine` — _main.go_
- `Club` — _common_types.go_
- `Club` — _common_types_wasm.go_
- `ClubService` — _club_service.go_
- `ClubTreasuryNode` — _backend_types.go_
- `CommissionEvent` — _common_types.go_
- `CommissionEvent` — _common_types_wasm.go_
- `ConsoleAssetReceipt` — _common_types.go_
- `ConsoleAssetReceipt` — _common_types_wasm.go_
- `ContractEngine` — _underworld_contracts.go_
- `ContractTemplate` — _underworld_contracts.go_
- `CounterfeitNote` — _counterfeit_service.go_
- `CounterfeitService` — _counterfeit_service.go_
- `CourthouseService` — _courthouse_service.go_
- `CreatorProfile` — _creator_store_service.go_
- `CreatorStore` — _creator_store_service.go_
- `CreatorStoreProduct` — _creator_store_service.go_
- `CyberInterceptEvent` — _backend_types.go_
- `DLCProduct` — _shop_registry.go_
- `DividendClaimData` — _entity_investment_service.go_
- `DynamicContract` — _underworld_contracts.go_
- `Engine` — _main.go_
- `EntityDividendTracker` — _entity_investment_service.go_
- `EntityInvestmentRecord` — _entity_investment_service.go_
- `EntityInvestmentService` — _entity_investment_service.go_
- `EntityMarketNode` — _backend_types.go_
- `EntityPortfolio` — _entity_investment_service.go_
- `Envelope` — _common_types.go_
- `Envelope` — _common_types_wasm.go_
- `EthereumClient` — _ethereum_client.go_
- `EvictionNotification` — _main.go_
- `EvictionPayload` — _backend_types.go_
- `EvidencePool` — _backend_types.go_
- `FaceplateStats` — _common_types.go_
- `FaceplateStats` — _common_types_wasm.go_
- `FenceListing` — _black_market_service.go_
- `FrameDelta` — _backend_types.go_
- `GameFrame` — _main.go_
- `GlobalSentiment` — _common_types.go_
- `GlobalSentiment` — _common_types_wasm.go_
- `GracePeriodMatrix` — _backend_types.go_
- `HoldingBonus` — _common_types.go_
- `HoldingBonus` — _common_types_wasm.go_
- `HostageSituation` — _backend_types.go_
- `IdentityBridge` — _identity_bridge.go_
- `IdentityProfile` — _backend_types.go_
- `InfoBrokerDeal` — _rival_career_engine.go_
- `InvestmentData` — _entity_investment_service.go_
- `ItemBuff` — _justice_service.go_
- `ItemDef` — _common_types.go_
- `ItemDef` — _common_types_wasm.go_
- `JusticeBountyDashboard` — _justice_service.go_
- `JusticeBuff` — _justice_service.go_
- `JusticeCard` — _justice_service.go_
- `JusticeDashboardAPI` — _justice_handlers.go_
- `JusticeDebuff` — _justice_service.go_
- `JusticeFaction` — _justice_service.go_
- `JusticeHandlers` — _justice_handlers.go_
- `JusticeMission` — _justice_service.go_
- `JusticeService` — _justice_service.go_
- `KidnapData` — _handlers_criminality.go_
- `KidnapState` — _common_types.go_
- `KidnapState` — _common_types_wasm.go_
- `Lease` — _common_types.go_
- `Lease` — _common_types_wasm.go_
- `LinkedWallet` — _common_types.go_
- `LinkedWallet` — _common_types_wasm.go_
- `LoadBalancedLedgerClient` — _resilience_utils.go_
- `Loan` — _common_types.go_
- `Loan` — _common_types_wasm.go_
- `LoanService` — _loan_service.go_
- `Lobby` — _backend_types.go_
- `LobbyPlayerInfo` — _lobby_manager.go_
- `ManagedNode` — _resilience_utils.go_
- `MarketActionPayload` — _oracle_service.go_
- `MarketActionRequest` — _main.go_
- `MatchHistory` — _common_types.go_
- `MatchHistory` — _common_types_wasm.go_
- `MatchState` — _common_types.go_
- `MatchState` — _common_types_wasm.go_
- `MetadataAttribute` — _common_types.go_
- `MetadataAttribute` — _common_types_wasm.go_
- `MoveData` — _common_types.go_
- `MoveData` — _common_types_wasm.go_
- `MultiChainRouter` — _resilience_utils.go_
- `MutationEvent` — _common_types.go_
- `MutationEvent` — _common_types_wasm.go_
- `NarrativeService` — _narrative_service.go_
- `NautilusDEXPathService` — _nautilus_dex_path.go_
- `NetworkConfig` — _common_types.go_
- `NetworkConfig` — _common_types_wasm.go_
- `NodeHealthReport` — _resilience_utils.go_
- `NonceData` — _backend_types.go_
- `OnboardingService` — _onboarding_service.go_
- `OracleService` — _oracle_service.go_
- `PayoutScheduler` — _economy_processing.go_
- `PendingRivalInvite` — _rival_career_engine.go_
- `PersistenceSyncWorker` — _economy_persistence.go_
- `PetNFT` — _backend_types.go_
- `Player` — _main.go_
- `PlayerService` — _player_service.go_
- `PlayerSession` — _backend_types.go_
- `PlayerStats` — _common_types.go_
- `PlayerStats` — _common_types_wasm.go_
- `PlaystyleTendencies` — _common_types.go_
- `PlaystyleTendencies` — _common_types_wasm.go_
- `QueueEntry` — _common_types.go_
- `QueueEntry` — _common_types_wasm.go_
- `RaidEvidence` — _backend_types.go_
- `RateBucket` — _backend_types.go_
- `RateLimitTier` — _rate_limiter.go_
- `RateLimiterService` — _rate_limiter.go_
- `RedemptionRequest` — _redemption_gateway.go_
- `RegionRivalry` — _backend_types.go_
- `RegionView` — _backend_types.go_
- `RegionalGovernanceMetric` — _backend_types.go_
- `ReportGloatData` — _common_types.go_
- `ReportGloatData` — _common_types_wasm.go_
- `ReputationShieldItem` — _justice_service.go_
- `RevenueSplitMatrix` — _economy_processing.go_
- `RivalryEngine` — _backend_types.go_
- `RivalryState` — _common_types_wasm.go_
- `RivalryState` — _rival_career_engine.go_
- `RouterSnapshot` — _economy_bootstrap.go_
- `RoyaltyTransaction` — _creator_store_service.go_
- `Rumor` — _common_types.go_
- `Rumor` — _common_types_wasm.go_
- `SeasonEvent` — _backend_types.go_
- `SeasonRewardPool` — _backend_types.go_
- `SeasonalEventEngine` — _backend_types.go_
- `ServerCard` — _common_types.go_
- `ServerCard` — _common_types_wasm.go_
- `SessionWatchdog` — _backend_types.go_
- `ShopItem` — _shop_registry.go_
- `SpreadRumorData` — _handlers_rumor.go_
- `SyncHandshaker` — _backend_types.go_
- `TelemetryLogger` — _economy_telemetry.go_
- `TokenSinkAuditReporter` — _economy_audit.go_
- `TokenSinkRouter` — _backend_types.go_
- `TournamentMatch` — _common_types.go_
- `TournamentMatch` — _common_types_wasm.go_
- `TournamentService` — _tournament_manager.go_
- `TournamentState` — _common_types.go_
- `TournamentState` — _common_types_wasm.go_
- `TournamentSummary` — _common_types.go_
- `TournamentSummary` — _common_types_wasm.go_
- `TreasureCache` — _backend_types.go_
- `TruthSerumItem` — _justice_service.go_
- `UnderworldContract` — _black_market_service.go_
- `UseItemData` — _common_types.go_
- `UseItemData` — _common_types_wasm.go_
- `UseTruthSerumRequest` — _justice_handlers.go_
- `UserEvent` — _backend_types.go_
- `VehicleNFT` — _backend_types.go_
- `VerificationHook` — _backend_types.go_
- `VictimRegistry` — _backend_types.go_
- `WalletLinkInfo` — _common_types.go_
- `WalletLinkInfo` — _common_types_wasm.go_
- `WalletQuota` — _rate_limiter.go_
- `WasmSignerHook` — _main.go_
- `WorldContentNFT` — _backend_types.go_
