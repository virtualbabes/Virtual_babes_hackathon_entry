# 02 — Economy / Faucet / AMM System

> Backend-authoritative economy. Ledger = uint64 micro-units (1 $VBV = 1,000,000 μVBV). No float arithmetic for balance derivation. All routing flows through `TokenSinkRouter`.

---

## 1. File Map

| File | Lines | Role |
|------|-------|------|
| `economy_service.go` | 724 | Core engine: `applyDynamicScalingLocked`, `CalculateReputation`, `handleSetDistrictTax`, district tax, bounty hunter bond, sabotage cost |
| `economy_processing.go` | 547 | `TokenSinkRouter` (RouteCriminalTax, RouteEntityDividend), `PayoutScheduler`, `RevenueSplitMatrix`, corporate bailouts |
| `economy_bootstrap.go` | 169 | `BootstrapEngine` — JSON snapshot hydration on boot |
| `economy_audit.go` | 179 | `TokenSinkAuditReporter` — invariant validation, drift detection, financial health report |
| `economy_persistence.go` | 209 | `PersistenceSyncWorker` — 15-min disk snapshots + backup rotation |
| `economy_telemetry.go` | 152 | `TelemetryLogger` — Prometheus metrics on `:9090/metrics` |
| `faucet_service.go` | 605 | `handleReward`, `dispatchReward`, `TransferTokens` (on-chain ARC-200), exit siphon |
| `market_service.go` | 561 | `handleTradeShares`, `GetSpotPrice`, `CalculateBuyCost`, `CalculateSellReturn`, `CalculateDynamicRumorFee` |
| `entity_market.go` | 599 | Public listings (pets/child-bots/vehicles), `PurchaseListing`, `PetBattleArena`, rivalry factions |
| `entity_shares.go` | 160 | `ShareToken` issuance + buy/sell (simplified, non-AMM) |
| `backend_types.go` | 719 | Core structs: `Lobby`, `TokenSinkRouter`, `EntityMarketNode`, `RevenueSplitMatrix`, `RegionalGovernanceMetric`, `ClubTreasuryNode` |
| `server.go` | 754 | Engine init, `newLobby`, bootstrap, persistence wiring, `initialBaseReward` setup |
| `lobby_manager.go` | 5019 | WS dispatch: `trade_shares`, `claim_dividends`, `harvest_all_dividends`, `vault_donation` |
| `common_types.go` | 622 | Constants: `MaxSinglePayoutMicro` (1000 VBV), `MaxGovPayoutMicro` (2000 VBV) |
| `handlers_public.go` | 439 | `handleFaucetStatus` — full faucet+AMM JSON state |
| `server_main.go` | ~600 | Route registration for `/api/reward`, `/api/faucet/status`, `/api/entity/market/*`, `/api/shares/*`, `/api/invest/*` |

---

## 2. Core Constants

| Constant | Value | Location |
|----------|-------|----------|
| `MaxSinglePayoutMicro` | 1,000,000,000 (1,000 $VBV) | `common_types.go:16` |
| `MaxGovPayoutMicro` | 2,000,000,000 (2,000 $VBV) | `common_types.go:19` |
| `maxFaucetCapacity` | 10,000.0 (float64 display) | `server.go:159` |
| Reward safety limit | `MaxSinglePayoutMicro / 2` | `economy_service.go:97` |
| Exit siphon rate | 2% of virtual balance | `faucet_service.go:461` |
| Exchange fee | 1% of trade value | `market_service.go:411` |
| Trade dividend seed | 15% of buy-side fees → entity pool | `market_service.go:280` |
| AMM initial reserve | 50,000 $VBV (50,000,000,000 μ) | `market_service.go:39` |
| AMM initial shares | 10,000 (100 shares × 100 units) | `market_service.go:38` |
| AMM reserve ratio | 0.33 | `market_service.go:40` |
| Governor payout interval | 24 hours | `server.go:185` |
| Persistence interval | 15 minutes | `server.go:198` |

---

## 3. Key Structs

### `TokenSinkRouter` — `backend_types.go:72`
```go
type TokenSinkRouter struct {
    Mu                   sync.RWMutex
    GlobalFaucetPool     *uint64                        // Points to l.faucetBalanceMicro
    AdminMaintenancePool *uint64                        // Infrastructure siphon (§11)
    Ledger               interface{}                    // *Lobby back-ref
    ActiveClubs          map[uint64]*ClubTreasuryNode   // Club treasuries
    MarketNodes          map[string]*EntityMarketNode   // AMM state (linked to l.marketNodes)
    RegionalDistricts    map[string]*RegionalGovernanceMetric
    Audit                *TokenSinkAuditReporter
    SiphonNotifier       func(string)
}
```

### `EntityMarketNode` — `backend_types.go:86`
```go
type EntityMarketNode struct {
    Mu                      sync.RWMutex
    EntityID                string
    TotalSharesIssued       uint64  // Micro-shares (1 share = 100 units)
    ReserveBalance          uint64  // Micro-VBV liquidity pool
    ReserveRatio            float64 // e.g., 0.33 (quadratic)
    DividendPoolMicro       uint64  // Yield-bearing assets
    CumulativeYieldPerShare uint64  // 1e12 scaled fixed-point
    IsDividendFrozen        bool    // Justice path
    Reputation              uint64  // 1 rep = 0.0001x spot boost
}
```

### `RevenueSplitMatrix` — `economy_processing.go:25`
```go
type RevenueSplitMatrix struct {
    FaucetShare     float64 // 0.0–1.0
    ClubShare       float64
    GovernanceShare float64
    EntityDividend  float64 // PILLAR 7-A
    CreatorRoyalty  float64 // PILLAR 7-C
}
```

### `Lobby` (economic fields) — `backend_types.go:548-566`
- `faucetBalanceMicro uint64` — authoritative integer reservoir
- `faucetBalance float64` — display-only derived
- `rewardStack map[string]uint64` — scaled per-asset reward amounts
- `initialRewards map[string]uint64` — unscaled template
- `initialBaseReward uint64` — primary reward template
- `rewardAssetID string` — ARC-200 app ID
- `maxFaucetCapacity float64` — 10,000
- `RewardRatio float64` — persisted for UI transparency
- `tokenSinkRouter *TokenSinkRouter`
- `marketNodes map[string]*EntityMarketNode`

---

## 4. Route Listings

### REST (HTTP)
| Method | Path | Handler | Location |
|--------|------|---------|----------|
| POST | `/api/reward` | `handleReward` → `dispatchReward` | `server_main.go:135`, `faucet_service.go:158` |
| GET | `/api/faucet/status` | `handleFaucetStatus` | `server_main.go:212`, `handlers_public.go:225` |
| GET | `/api/entity/market/list` | `handleEntityMarketList` | `server_main.go:215`, `entity_market.go:463` |
| POST | `/api/entity/market/create` | `handleEntityMarketCreate` | `server_main.go:216`, `entity_market.go:470` |
| POST | `/api/entity/market/purchase` | `handleEntityMarketPurchase` | `server_main.go:217`, `entity_market.go:491` |
| POST | `/api/shares/issue` | `handleEntitySharesIssue` | `server_main.go:245`, `entity_shares.go:114` |
| POST | `/api/shares/buy` | `handleEntitySharesBuy` | `server_main.go:246`, `entity_shares.go:136` |
| GET | `/api/shares/tokens` | `handleEntitySharesTokens` | `server_main.go:247`, `entity_shares.go:153` |
| GET | `/api/shares/holdings` | `handleEntitySharesHoldings` | `server_main.go:248`, `entity_shares.go:157` |
| POST | `/api/claim/dividends` | dividend claim | `server_main.go:447` |
| GET | `/api/invest/dividends/history` | dividend history | `server_main.go:463` |
| GET | `/api/invest/portfolio` | portfolio holdings | `server_main.go` |
| GET | `/api/invest/entity` | entity detail | `server_main.go` |
| POST | `/api/reward/add` | admin add reward | `server_main.go:367` |
| POST | `/api/reward/remove` | admin remove reward | `server_main.go:368` |
| POST | `/api/reward/update-base` | admin update base reward | `server_main.go:369` |

### WebSocket (envelope types)
| Type | Handler | Location |
|------|---------|----------|
| `trade_shares` | `handleTradeShares` | `lobby_manager.go:1482`, `market_service.go:277` |
| `claim_dividends` | `HandleClaimDividends` | `lobby_manager.go:1537`, `market_service.go:55` |
| `harvest_all_dividends` | `HandleHarvestAllDividends` | `lobby_manager.go:1541`, `market_service.go:144` |
| `vault_donation` | inline in lobby_manager | `lobby_manager.go:1644` |
| `set_district_tax` | `handleSetDistrictTax` | `economy_service.go:463` |
| `purchase_bounty_hunter_bond` | `HandlePurchaseBountyHunterBond` | `economy_service.go:654` |
| `refund_bounty_hunter_bond` | `HandleRefundBountyHunterBond` | `economy_service.go:688` |
| `purchase_raid_insurance` | `HandlePurchaseRaidInsurance` | `economy_service.go:618` |
| `freeze_dividends` | `HandleJusticeFreezeDividends` | `market_service.go:101` |

---

## 5. Faucet Dynamic-Scaling Mechanism

### `applyDynamicScalingLocked()` — `economy_service.go:30-111`

The faucet reward scaling adjusts all base rewards proportionally to how full the faucet is relative to its target maximum capacity.

#### Step-by-step:

1. **Calculate total liabilities** (line 42-56):
   ```go
   var totalLiabilitiesMicro uint64
   for _, bal := range l.playerBalances {
       totalLiabilitiesMicro += bal
   }
   // + ArenaVouchers + RecoveryBounties + BountyHunterBondMicro per leaderboard entry
   ```

2. **Calculate total club reserves** (line 58-61):
   ```go
   var totalClubReserves float64
   for _, club := range l.clubs {
       totalClubReserves += club.Treasury
   }
   ```

3. **Calculate tournament commitment** (line 63-66):
   ```go
   tournamentCommitment := float64(l.tournamentPotBonusMicro) / 1000000.0
   if l.tournament.Active {
       tournamentCommitment += float64(l.tournament.PotMicro) / 1000000.0
   }
   ```

4. **Compute usable balance** (line 68):
   ```go
   usableBalance := l.faucetBalance - 1.0 - totalLiabilities - totalClubReserves - tournamentCommitment - pendingTournamentPayouts
   // 1.0 = gas floor reserve
   // Clamped to 0 if negative
   ```

5. **Compute ratio** (line 73-79):
   ```go
   ratio := usableBalance / l.maxFaucetCapacity
   // Clamped to [0.1, 1.0]
   ```

6. **Scale the reward stack** (line 86-106):
   ```go
   l.rewardStack = make(map[string]uint64)
   for assetID, initialAmt := range l.initialRewards {
       scaledAmt := uint64(float64(initialAmt) * ratio)
       l.rewardStack[assetID] = scaledAmt
   }
   ```

7. **Safety cap alignment** (line 97-106): If aggregated base rewards exceed 50% of `MaxSinglePayoutMicro`, clamp further to preserve headroom for reputation multipliers, bounties, and virtual balances.

8. **Persist ratio** (line 108): `l.RewardRatio = ratio` for UI transparency.

#### Trigger points:
- After every `TransferTokens` (faucet_service.go:76, 146)
- After `dispatchReward` (faucet_service.go:596)
- After every trade (market_service.go:443, 499)
- After dividend claim (market_service.go:93, 185)
- After district tax surcharge (economy_service.go:541)
- After bond/insurance purchase (economy_service.go:646, 680, 722)
- After corporate bailout (economy_processing.go:544)

---

## 6. Anti-Whale Bonding Curve (AMM)

### `GetSpotPrice()` — `market_service.go:191-206`
```go
price := ReserveBalance / (TotalSharesIssued * ReserveRatio)
// Reputation multiplier: 1.0 + rep/1,000,000
// Floor: 0.01 micro-VBV
```

### `CalculateBuyCost(unitsToBuy)` — `market_service.go:210-228`
```go
// Bancor Formula:
supplyRatio := unitsToBuy / (TotalSharesIssued + 1)
costModifier := math.Pow(1.0 + supplyRatio, 1.0/ReserveRatio) - 1.0
baseCost := ReserveBalance * costModifier

// Quadratic Whale Penalty:
slippageFactor := 1.0 + math.Pow(supplyRatio, 2.0) * 5.0
finalCost := baseCost * slippageFactor
// Returns (uint64.ceil(finalCost), slippagePercent)
```

### `CalculateSellReturn(unitsToSell)` — `market_service.go:232-256`
```go
// Bancor Sell Formula:
supplyRatio := unitsToSell / TotalSharesIssued
returnModifier := 1.0 - math.Pow(1.0 - supplyRatio, 1.0/ReserveRatio)
baseReturn := ReserveBalance * returnModifier

// Sell slippage (max 90% penalty):
slippageFactor := 1.0 - math.Pow(supplyRatio, 2.0) * 2.0
if slippageFactor < 0.1 { slippageFactor = 0.1 }
finalReturn := baseReturn * slippageFactor
// Returns (uint64.floor(finalReturn), slippagePercent)
```

### `CalculateDynamicRumorFee()` — `market_service.go:260-274`
```go
marketCap := GetSpotPrice() * TotalSharesIssued
dynamicFee := (500 * 1,000,000) + (marketCap * 0.025)
// Soft cap: 50,000 $VBV
```

### Rumor Multiplier in Trades — `market_service.go:361-367`
```go
rumorMultiplier := 1.0
for _, rumor := range l.rumors {
    if rumor.TargetWallet == targetWallet && now.Before(rumor.ExpiresAt) {
        rumorMultiplier *= rumor.Strength
    }
}
// Applied to buy cost AND sell return
```

---

## 7. TokenSinkRouter Flow

### `RouteCriminalTax()` — `economy_processing.go:236-474`

All economic redistribution flows through this function:

1. **Validation** (line 242-245): `FaucetShare + ClubShare + GovernanceShare` must equal 1.0 (±1e-9)

2. **Admin Siphon** (line 252-264): If inflow/allocated > 150%, extract 10% to `AdminMaintenancePool`

3. **Console Siphon** (line 271-296): For `CONSOLE_DLC`/`GHOST_TAX_ENFORCED`/`STAGNATION_TAX` contexts when faucet > 80% capacity, 10% → infrastructure

4. **Routing** (line 300-356):
   - `FaucetShare` → `*GlobalFaucetPool` (points to `l.faucetBalanceMicro`)
   - `ClubShare` → `ActiveClubs[targetClubID].TreasuryBalance` or split globally
   - `GovernanceShare` → `RegionalDistricts[targetDistrict].DistrictDividendPool`

5. **Remainder accounting** (line 361-368): Reroute rounding fractions to Faucet Pool

6. **Audit** (line 372-375): `InterceptAndAudit()` — asserts zero drift

7. **Entity dividend seeding** (line 381-403): For `CONTRACT_FEE`/`UNDERWORLD_CONTRACT` contexts, route portion to entity dividend pools

### `RouteEntityDividend()` — `economy_processing.go:35-63`
- Adds amount to `MarketNodes[entityID].DividendPoolMicro`
- Audited via `ENTITY_DIVIDEND_ROUTED`

### `PayoutScheduler.ProcessAllRegionalDividends()` — `economy_processing.go:109-220`
- Runs every 24h via ticker
- Isolates `DistrictDividendPool` values, resets to 0
- Disburses via `ExternalLedgerClient.TransferTokens`
- Fail-safe rollback on transfer failure
- Syncs club treasuries and triggers broadcast

---

## 8. Exit Siphon (On-Chain Bridge)

### `dispatchReward()` — `faucet_service.go:381-605`

1. **Virtual balance reset** (line 454-456): `l.playerBalances[claimant] = 0`

2. **2% exit siphon** (line 461-462):
   ```go
   exitSiphonMicro := (virtualBalance * 2) / 100
   actualPayoutMicro := virtualBalance - exitSiphonMicro
   ```

3. **Per-asset reward construction** (line 468-532):
   - Base amount scaled by reputation multiplier (1.1x if Rep ≥ 500)
   - Mojo tier bounty multiplier (1.05x–1.25x)
   - Opt-in verification via `oracleService.CheckAssetOptIn`
   - Granular skip if asset not opted in

4. **Safety cap** (line 537-543): If total > `MaxSinglePayoutMicro` (1000 VBV), rollback and reject

5. **Group transaction** (line 563-573): `crypto.ComputeGroupID`, sign all, atomic broadcast

6. **Post-dispatch** (line 587-601):
   - Decrement `l.faucetBalanceMicro`
   - Re-derive `l.faucetBalance` float
   - `applyDynamicScalingLocked()`
   - `Audit.LogPhysicalOutflow(totalMicro)`

---

## 9. `TransferTokens()` — On-Chain Dispatch

### `faucet_service.go:33-154`

Implements `ExternalLedgerClient` interface.

1. **Gas floor check** (line 55-59): Vault native balance ≥ 1.0 VOI

2. **Multi-chain dispatch** (line 62-84): Via `multiChainRouter.TransferToChain()` if available

3. **Fallback ARC-200 transfer** (line 86-153):
   - Method: `transfer(address,uint256)` — selector `0x2b426dec`
   - Construct `MakeApplicationNoOpTx` with ABI-encoded args
   - Sign and dispatch
   - Wait for confirmation (4 rounds)

4. **Post-transfer** (line 139-152):
   - Decrement `faucetBalanceMicro`
   - `applyDynamicScalingLocked()`
   - `Audit.LogPhysicalOutflow()`

---

## 10. EntityMarketNode AMM Math

### Seed Parameters — `market_service.go:36-45`
```go
node = &EntityMarketNode{
    TotalSharesIssued: 10000,           // 100 shares × 100 units
    ReserveBalance:    50000 * 1000000, // 50,000 $VBV
    ReserveRatio:      0.33,            // Quadratic coefficient
    DividendPoolMicro: 0,
    CumulativeYieldPerShare: 0,
}
```

### Buy Flow — `market_service.go:399-456`
1. `CalculateBuyCost(unitsToTrade)` returns (cost, slippage)
2. Apply `rumorMultiplier` to cost
3. Deduct from `playerBalances[wallet]`
4. Credit shares: `stats.Portfolio[targetWallet] += unitsToTrade`
5. Update AMM state:
   ```go
   node.TotalSharesIssued += unitsToTrade
   node.ReserveBalance += netToReserveMicro
   ```
6. Route 1% fee via `RouteCriminalTax()` with matrix `{FaucetShare: 0.80, GovernanceShare: 0.20}`
7. Route 15% of fee as entity dividend seed

### Sell Flow — `market_service.go:457-512`
1. `CalculateSellReturn(unitsToTrade)` returns (return, slippage)
2. Apply `rumorMultiplier` to return
3. Faucet liquidity check: `l.faucetBalance >= totalValueBase`
4. Credit `playerBalances[wallet]` with net after fee
5. Update AMM state:
   ```go
   node.TotalSharesIssued -= unitsToTrade
   node.ReserveBalance -= totalValueMicro  // Gross reduction
   ```

---

## 11. Dividend System

### `HandleClaimDividends` — `market_service.go:55-95`
```go
yieldDelta := node.CumulativeYieldPerShare - stats.LastClaimedYield[targetEntity]
payoutMicro := (yieldDelta * sharesHeld) / 1e12  // Fixed-point scaling
l.playerBalances[wallet] += payoutMicro
// Sync claim point
stats.LastClaimedYield[targetEntity] = node.CumulativeYieldPerShare
```

### `HandleHarvestAllDividends` — `market_service.go:144-187`
- Iterates all portfolio positions, claims each in one action
- Caps via `applyDynamicScalingLocked()`

### `HandleJusticeFreezeDividends` — `market_service.go:101-138`
- Restricted to `JobRole == "Tax Auditor"`
- Target must have `WantedLevel > 30`
- Sets `node.IsDividendFrozen = true`
- Frozen dividends diverted to Faucet as Regulatory Fines

---

## 12. Frontend vs Backend-Only

### Exposed to Frontend (via `/api/faucet/status`, WS broadcasts)
- `faucet_balance_micro`, `max_faucet_capacity`, `usable_balance`, `reward_ratio`
- `reward_stack` (all asset amounts)
- `amm_nodes[]` — entity_id, reserve_balance, total_shares_issued, spot_price
- `market_weather` (placeholder)
- Portfolio via WS `portfolio_update` envelope
- Leaderboard stats (Reputation, Mojo, Wins, etc.)

### Frontend Modules
- `faucet_dashboard.js` — Health bar, scaling factor, drain simulator, AMM bonding curve SVG
- `economy.js` — `openVaultInteraction`, `openMutationFoundryOverlay`, `tradeShares`, `openTradeSharesOverlay`
- `investment_dashboard.js` — 4-panel overlay: Yield Summary, Entity Marketplace, Portfolio Holdings, Dividend Tracker
- `entity_shares.js` — Token issuance + buy forms

### Backend-Only (never exposed)
- `TokenSinkRouter.RouteCriminalTax` internal splits (audit context strings)
- `InterceptAndAudit` audit trail
- `PayoutScheduler` governor dividend accumulator
- `PersistenceSyncWorker` snapshot internals
- `BootstrapEngine` hydration details
- `CalculateReputation` full formula (only final int sent)
- `dispatchReward` group transaction construction
- `TransferTokens` ARC-200 ABI construction
- `admin_audit` log entries

---

## 13. Corporate Bailouts

### `CheckCorporateBailouts()` — `economy_processing.go:490-547`

- Trigger: Club territory coverage < 20% (with at least 1 territory)
- Amount: 5,000 $VBV per club
- Source: `l.faucetBalanceMicro` (must have ≥ 5,000 $VBV)
- Liability shift: `node.TreasuryBalance += 5000 * 1e6`
- Audit: `CORPORATE_BAILOUT` context

---

## 14. Reputation Calculation

### `CalculateReputation(stats)` — `economy_service.go:235-442`

1. **Win Rep** (diminishing returns): 0–100 wins: ×100; 101–500: ×25; 501+: ×5
2. **Mojo contribution**: `GetEffectiveMojo() * 10`
3. **Penalties**: DNFs ×50, DisconnectStreak ×15, WantedLevel ×20, JailedCards ×25
4. **Achievements**: 50–300 per achievement ID
5. **Marketability multiplier**: `(1 + Aggressiveness×0.15 + RiskTolerance×0.10) × brandAmper`
   - `brandAmper` from employer club Mojo: `1.0 + effMojo/4000` (max 1.25)
6. **Employment multiplier**: `1.0 + effMojo/2000` (max 1.5), -0.20 if sabotaged, +0.10 if regional governor
7. **Cosmetic prestige**: Diamond Tier (rep≥500): `1.0 + MojoBonus×0.005`; Standard: `MojoBonus×10`
8. **Spreader bonus**: RumorCount ×10 (max 100)
9. **Intelligence bonus**: 20 per unique rival club audited (max 100)
10. **Capital presence**: 1.25x if employer controls arena_center
11. **Career role weighting**: `playerService.GetReputationWeighting(JobRole) / 100`
12. **Security bonus**: 1.1x if regional governor with 5+ reparations
13. **House Authority**: 1.5x for vault address

---

## 15. Tax & Fee Summary

| Event | Fee | Routing |
|-------|-----|---------|
| Match reward (exit siphon) | 2% of virtual balance | Returns to Faucet pool |
| Share trade | 1% of trade value | 80% Faucet, 20% Arena Center Governor |
| Buy-side trade dividend seed | 15% of fee | Entity dividend pool |
| District tax (Governor) | 0–20% (set by Governor) | Arena Center owner via TokenSinkRouter |
| Governor surcharge | 1% of club treasury | Arena Center owner via TokenSinkRouter |
| Bounty hunter tax | 5% of bounty | Justice Pool (stays in faucet) |
| Infrastructure siphon | 10% of payload | AdminMaintenancePool (if faucet > 80% capacity) |

---

## 16. Safety Caps & Circuit Breakers

| Cap | Value | Location |
|-----|-------|----------|
| Max single payout | 1,000 $VBV | `common_types.go:16` |
| Max governor payout | 2,000 $VBV | `common_types.go:19` |
| Reward safety limit | 500 $VBV (50% of max) | `economy_service.go:97` |
| Gas floor | 1.0 VOI native | `faucet_service.go:403` |
| Virtual balance rollback | On skip/exceed | `faucet_service.go:537-561` |
| District tax ceiling | 20% | `economy_service.go:488` |
| AMM spot price floor | 0.01 micro-VBV | `market_service.go:196` |
| Sell slippage floor | 0.1 (90% max penalty) | `market_service.go:248` |

---

## 17. Audit & Telemetry

### `TokenSinkAuditReporter` — `economy_audit.go:31-179`

Atomic counters:
- `TotalSystemInputVetted` — cumulative inflow
- `TotalSystemAllocated` — cumulative (faucet+club+gov)
- `TotalSystemSiphoned` — infrastructure siphons
- `TotalRewardsExited` — on-chain dispatches
- `TotalGhostReclaimed` — ghost tax
- `TotalStagnationFees` — activity enforcement
- `TotalPlatformFees` — self-redemption surcharges

Invariant: `InterceptAndAudit` asserts `payload == faucet + club + gov + siphoned` (skew == 0).

### Prometheus Metrics — `economy_telemetry.go:36-90`

- `arena_bootstrap_duration_seconds` (Gauge)
- `arena_financial_health_status` (Gauge: 1=healthy, 0=compromised)
- `arena_economy_inflow_total_micro_vbv` (Counter)
- `arena_economy_outflow_total_micro_vbv` (Counter)
- `arena_economy_net_drift_micro_vbv` (Gauge: must be 0)
- `arena_enforcement_platform_fees_total_micro_vbv` (Counter)
- `arena_enforcement_ghost_reclaimed_total_micro_vbv` (Counter)
- `arena_enforcement_stagnation_fees_total_micro_vbv` (Counter)
- `arena_governor_taxes_micro_vbv` (GaugeVec, labeled by district)

---

## 18. Bootstrap & Persistence

### `BootstrapEngine` — `economy_bootstrap.go:40-169`

1. Reads `economy_state_authoritative.json` from `DataDir`
2. Validates `GlobalFaucetValue > 0` (rejects "poverty bootstrap")
3. Restores `GlobalFaucetPool` pointer
4. Restores atomic audit counters
5. Hydrates `ActiveClubs`, `MarketNodes`, `RegionalDistricts`, `linkedWallets`

### `PersistenceSyncWorker` — `economy_persistence.go:18-209`

- Every 15 minutes: snapshots router state to JSON
- Rotates 5 backups (`.1` through `.5`)
- Atomic write: `.tmp` → `Rename`
- Also saves AI citizens, bonded assets, local-model promotions

---

## 19. Key Interactions Map

```
Match Win → handleReward → dispatchReward → decrement faucetBalanceMicro → applyDynamicScalingLocked
                                                              ↓
                                                        ARC-200 on-chain
                                                              ↓
                                                    Audit.LogPhysicalOutflow

Trade Buy → handleTradeShares → CalculateBuyCost → playerBalances↓ → AMM state↑ → applyDynamicScalingLocked
                           ↓
                     RouteCriminalTax (1% fee) → Faucet (80%) + Governor (20%)
                           ↓
                     RouteEntityDividend (15% of fee) → DividendPoolMicro

Dividend Claim → HandleClaimDividends → playerBalances↑ → applyDynamicScalingLocked

Governor Payout → PayoutScheduler.ProcessAllRegionalDividends → TransferTokens (on-chain)

Vault Donation → faucetBalanceMicro↑ → applyDynamicScalingLocked

Bailout → CheckCorporateBailouts → faucetBalanceMicro↓ → Club.TreasuryMicro↑ → applyDynamicScalingLocked
```

---

*Generated from source. All line numbers refer to `Z:/Crypto_Draught/NFT-Seduction/`.*
