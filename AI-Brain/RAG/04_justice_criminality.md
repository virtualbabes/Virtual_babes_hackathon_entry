# 04 — Justice / Criminality / Underworld System

> **Scope:** Justice Hegemony faction, bounty/raid/kidnap mechanics, underworld contract engine, career tiers, counterfeit economy, black market fencing, loan liquidation pipeline, and Ghost Protocol signal-scrambling.
> **Architecture:** Backend-authoritative, uint64 micro-unit ledger. Services are stateless; state lives in `Lobby` (mutex-guarded maps). All economic math uses micro-units (1 $VBV = 1,000,000 μVBV).

---

## 1. Justice Hegemony Path (Pillar 7)

### 1.1 Justice Card Archetypes
- **File:** `justice_service.go:25-30`
- Four card types: `ENFORCER`, `MEDIATOR`, `WARDEN`, `COMMISSIONER`
- Each grants **+10% power vs outlaws** (multiplier 1.10, `justice_service.go:538`)
- Flat power bonus per card: Enforcer/Mediator = 500,000 μVBV, Warden = 750,000, Commissioner = 1,000,000 (`justice_service.go:258-263`)

### 1.2 Justice Tier Bounty Center Dashboard
- **File:** `justice_service.go:122-128` (struct), `:442-472` (logic)
- Access gate: **3+ Justice Cards AND BountyRank ≥ 2 (Warden)** (`justice_service.go:227`, `:588-590`)
- Constants: `JusticeTierAccessCards = 3`, `JusticeTierAccessRank = 2`
- Dashboard shows: active missions, high-Wanted targets, Ghost Protocol scrambled count
- **Ghost Protocol resolution:** Warden+ players see through scrambled IDs via "enhanced tracking" — sets `RealIDAvailable = true` (`justice_service.go:462-467`)

### 1.3 Bounty Mechanics
- **Wanted Level threshold for Justice bonus:** `JusticeOutlawBonusThreshold = 15` (`justice_service.go:591-592`)
- Bounty reward formula: `wantedLevel * 100_000 μVBV` per target (`justice_handlers.go:198`)
- Capture bounty endpoint: `POST /api/justice/capture-bounty` — base reward 1,500,000 μVBV, scaled by `BountyRank * 500_000` (`justice_handlers.go:297-300`)
- **Bounty Hunter Tax:** 5% Justice Faction tax on bounties, routed to faucet (`economy_service.go:444-459`)

### 1.4 Truth Serum (Intelligence Item)
- **File:** `justice_service.go:91-95`, `:296-323`
- Reveals target's active buffs/debuffs for 30 seconds (`TruthSerumDefaultDuration = 30s`, `:594`)
- Auto-expires via goroutine after duration
- Endpoint: `POST /api/justice/use-truth-serum`

### 1.5 Reputation Shield
- **File:** `justice_service.go:114-119`, `:338-364`
- Absorbs reputation loss up to `ProtectionAmount` (default 50, `:596`)
- Drains per-absorption; multiple shields iterate until loss is fully absorbed or shields exhausted (`:367-398`)

### 1.6 Justice Faction State
- **File:** `justice_service.go:33-41`
- Fields: Alignment, JusticeCards[], BountyRank (0=Hunter, 1=Icon, 2=Warden, 3=Commissioner), TrophyCount, ActiveBuffs, Missions
- BountyRank drives salary-scale rewards and dashboard access tier

### 1.7 HTTP Endpoints (Justice)
| Endpoint | Method | File:Line |
|----------|--------|-----------|
| `/api/justice/dashboard` | GET | `justice_handlers.go:165` |
| `/api/justice/use-truth-serum` | POST | `justice_handlers.go:217` |
| `/api/justice/capture-bounty` | POST | `justice_handlers.go:264` |
| `/api/justice/bounty-board` | GET | `justice_handlers.go:321` (alias) |
| `/api/justice/award-card` | POST | `justice_handlers.go:335` |
| `/api/justice/use-rep-shield` | POST | `justice_handlers.go:385` |

---

## 2. Career Progression System (Pillar 13)

### 2.1 Career Tiers
- **File:** `rival_career_engine.go:17-23`
- Tiers: `CareerTierPeon=0`, `Apprentice=5`, `Journeyman=15`, `Expert=30`, `Master=50`, `Boss=75`
- **$VBV-sustained thresholds** (`rival_career_engine.go:27-34`):
  - Peon: 0, Apprentice: 5K $VBV, Journeyman: 25K, Expert: 100K, Master: 500K, Boss: 2M $VBV
  - Demotion grace period: 7 days after warning

### 2.2 CareerXP Struct
- **File:** `rival_career_engine.go:45-57`
- Fields: `RoleXP map[string]uint64`, `LessonLevel`, `PromotedRoles`, `LiquiditySamples`, `AvgSustainedMicro`
- Tier derivation: `int(xp / 1500)` per role (`:60-69`)
- VBV gating multiplier: `GetVBVGatingMultiplier()` returns ×1 (Peon) through ×32+ (Boss+) based on `AvgSustainedMicro` (`:572`)

### 2.3 Underworld + Justice Career Roles
- **Underworld:** Saboteur, Smuggler, Fence, Launderer, Kidnapper, Gossip, Counterfeiter, Black Market Dealer, Hostage Host, Heist Planner
- **Justice:** Warden (D3), Commissioner, Enforcer, Mediator, Bounty Hunter, Forensic Analyst (D5), Sector Peacekeeper (D6), Judge, Intel Agent, Compliance Auditor, Ethics Overseer, Tax Auditor
- Cross-career compatibility (`underworld_contracts.go:325-344`): Smuggler↔Fence, Saboteur↔Kidnapper, Launderer↔Fence

### 2.4 Career Tier → Reward Multipliers
- `ComputeScaledXP` applies $VBV-gate multiplier then tier bonus (+10% per tier above Tier 1, capped ×1.5) (`underworld_contracts.go:412-427`)
- Fence fee discount: `GetFenceFeeDiscount()` reduces black-market commission from 10% → 5% at Tier 3+ (`rival_career_engine.go:526`)

---

## 3. Underworld Contract Engine (Pillar 3)

### 3.1 Contract Templates
- **File:** `underworld_contracts.go:54-225`
- **~33 contract templates** (not 19 — the doc comment says ~19 but registry has 33 entries)
- Categories:
  - **Sabotage (CONTRACT-001,004,007,008,011,014,016,018,021,025,026):** District/arena sabotage, scaling 1.5K–50K $VBV
  - **Heist (CONTRACT-009,012,013,015,017,019,020,022,023,027):** Club treasury heists, 4K–100K $VBV
  - **Laundering/Finance (CONTRACT-024,028):** Data haven audit, capture protocols, 15K–30K $VBV
  - **Kidnap Gambit (CONTRACT-005,010):** Standard + Governor favorite card, 3K–5K $VBV
  - **Rumor Spread (CONTRACT-006):** Governor defamation, 1.5K $VBV
  - **UnderworldBoss (CONTRACT-029-033):** Ascension/final protocols, 50K–200K $VBV, Wanted 40-100+

### 3.2 Dynamic Contract Generation
- **File:** `underworld_contracts.go:246-322`
- Eligibility gates: `RequiredWantedMin`, `RequiredCareerTier`, `RequiresAlliance`, `RequiresTerritory`
- Reward scaling:
  - Base × `vbvMultiplier` (from `CareerXP.GetVBVGatingMultiplier()`)
  - +5% per 10 Wanted above minimum (`:292-295`)
  - × `RivalMultiplier` if Justice career present (`:298-300`, ranges 1.2–5.0)

### 3.3 Contract Lifecycle
- **Assignment:** `HandleAssignContract()` (`:539-668`) — one-contract-at-a-time rule, validates all eligibility, broadcasts `underworld_contract_assigned` WS event
- **Completion:** `HandleCompleteContract()` (`:670-760`) — verifies active contract, computes reward + XP, routes payout to `playerBalances`, broadcasts `underworld_contract_completed`
- **Completion hooks:** Called from `battle_service.go`, `club_service.go`, `handlers_criminality.go` at resolution points

### 3.4 Black Market Integration
- **File:** `black_market_service.go:240-472`
- Static `HandleGetUnderworldContracts()` — legacy list of 28 contracts (superseded by dynamic engine but retained for compat)
- `HandleAcceptUnderworldContract()` (`:598-765`) — career-gated: CONTRACT-024→Fence, 025→Kidnapper, 026→Saboteur, 027→Smuggler, 028→Launderer
- `HandleAbortUnderworldContract()` (`:769-799`) — -50 reputation penalty

---

## 4. Kidnapping & Hostage System (Pillar 3)

### 4.1 Victim Registry
- **File:** `handlers_criminality.go:19-56`
- Multi-slot attacker isolation: one attacker per victim slot, 48-hour expiration
- `ErrAttackerAlreadyHoldsHostage` prevents duplicate kidnaps

### 4.2 Kidnap Gambit Flow
- **File:** `handlers_criminality.go:64-120+`
- Target selection: FavoriteCardID first, fallback to rarest card in victim inventory
- Ransom in μVBV; hostage stored in `stats.KidnappedCards` and `stats.HeldHostageCards`
- Black market can liquidate kidnapped cards (`black_market_service.go:115-136`) — clears hostage records and victim's profile on sale

### 4.3 Raid Insurance
- **File:** `economy_service.go:615-645`
- Hostage Host career: 24h protection buffer, 1 claim remaining, blocks one AOS Raid
- Bounty Hunter Bond: 1,000 $VBV deposit to secure against false captures (`:651-674`)

---

## 5. Counterfeit Economy

### 5.1 Counterfeit Generation
- **File:** `counterfeit_service.go:17-133`
- **Career gate:** Counterfeiter role OR CareerTier ≥ 3 (`:62-65`)
- Cost: 5% of counterfeit value (`:68`)
- Detection chance: 15% base (Tier 1), 12% (Tier 3-4), 8% (Tier 5+) (`:87-92`)
- Rate-limited: 60s cooldown per player (`:81-83`)
- XP: 70 base, 120 at Tier 5+ (`:114-118`)

### 5.2 Counterfeit Detection (Justice)
- **File:** `counterfeit_service.go:136-271`
- **Career gate:** Warden, Forensic Analyst, Sector Peacekeeper only (`:173-176`)
- Forensic Analyst: halves detection chance (×0.5) — easier to find notes (`:216-218`)
- Sector Peacekeeper: ×0.7 additional improvement with sector tiles (`:220-223`)
- Detection cost: 1% of credits (min 100 μVBV), verified via oracle transaction
- XP on detection: Warden=25, Forensic Analyst=60, Sector Peacekeeper=20 (`:251-255`)

### 5.3 Seizure & Cleanup
- `SeizeCounterfeitNoteLocked()` (`:274-295`): forced removal, creator loses 50 rep + 10 Wanted, seizing agent gains 40 XP
- `CleanupExpiredCounterfeits()` (`:298-316`): notes degrade after 24 hours, removed from active pool

---

## 6. Black Market & Fencing

### 6.1 Selling to Black Market
- **File:** `black_market_service.go:33-226`
- Accepts cards from inventory OR kidnapped hostage cards
- Price: 50% of estimated value (sum of 4 power stats × 100,000 μVBV) (`:79-80`)
- **Fence Fee:** 10% standard, 5% for Fence role, further reduced by `GetFenceFeeDiscount()` at Tier 3+ (`:85-106`)
- Card marked `Fallen = true` on sale (`:140-141`)
- +1 Wanted Level per sale (`:144`)
- Fence XP: 30 base, scaled by loyalty/fame + $VBV-gate multiplier (`:169-208`)

### 6.2 Fenced Goods Marketplace
- **File:** `black_market_service.go:801-1093`
- `HandleListFenceGoods`: list card for sale, 8% commission (5% Fence Tier 3+), 24h expiry
- `HandleBuyFencedGood`: escrow purchase, commission routed to faucet via token-sink router
- `cleanupExpiredFencedListings`: returns card to seller after 24h

### 6.3 Black Market (Loan Defaults)
- **File:** `black_market_service.go:478-593`
- Defaulted loans become black-market listings with Dutch Auction pricing (`:504-517`)
- Price starts at 75% of repayment, decays 5%/hour, max 70% decay
- Buy with `HandleBuyBlackMarket`: costs virtual balance, +5 Wanted (`:582`)
- Access gate: Wanted Level 5+ OR Cunning 10+ (`:490-491`)

---

## 7. Loan System & Liquidation Pipeline

### 7.1 Collateralized Loans
- **File:** `loan_service.go:25-152`
- Collateral: CardBundle (card + weapon + faceplate)
- Interest: 10% with nearest-micro-unit rounding (`:120`)
- Territory: `south_slums` (Second-Hand Store / Loan Office) (`:132`)

### 7.2 Liquidation (Default)
- **File:** `loan_service.go:230-372`
- Auto-repayment: if borrower has vault approval ≥ repayment, on-chain pull executes (`:249-287`)
- On default:
  - Borrower receives 15% of loan as Market Tokens (`:299`)
  - +5 Wanted Level (`:308`)
  - **5% liquidation fee** to Second-Hand Store district owner (`:319-328`)
  - Collateral moved to **Black Market** (FIFO 50-item cap) (`:354-359`)
  - On-chain forensic log: `VBT_LOAN_LIQUIDATE` (`:349-352`)

---

## 8. Employment & Salary (Pillar 5)

### 8.1 Career Service (Salary Dispenser)
- **File:** `career.go:17-130`
- Daily ticker pays salaries from Club Treasuries (`:19`)
- **Corporate Tax:** 2% on salaries ≥ 500 $VBV, funds Global Faucet (`:38-53`)
- **Regulatory Bypass Permit:** active buff reduces corporate tax by 50% (`:47-50`)
- **Outlaw Tax:** garnish earnings based on Wanted Level (2% per Wanted, max 40%) (`:56-62`)
- Net salary = gross − (corpTax + outlawTax); taxes route to faucet (`:67-74`)
- Contract termination on club insolvency/dissolution → player becomes Freelancer (`:90-113`)

### 8.2 Hiring & Roles
- **File:** `employment_service.go:13-85`
- Club owner hires player, assigns role (Manager, Security, Clerk)
- Salary set via `handleSetSalary` — EXECUTIVE_PAY achievement at ≥500 $VBV (`:133-136`)

### 8.3 Capital Laundering
- **File:** `employment_service.go:150-189`
- Launderer career: pay 1,000 $VBV fee → reduce Wanted Level by 3

---

## 9. Ghost Protocol (Signal Scrambling)

- **File:** `lobby_manager.go:2423-2440`, `justice_service.go:462-467`
- Player field: `GhostProtocolExpiresAt time.Time` (`common_types.go:408`)
- **Effect on bounty board:** scrambling hides player identity from non-Justice players
- **Justice counter:** Warden+ Bounty Center uses "enhanced tracking" to resolve scrambled IDs (`justice_service.go:462-467`)
- Active contracts (CONTRACT-019) require Ghost Protocol for high-stakes heists

---

## 10. Wired vs. Scaffold Status

### Fully Wired ✅
- Justice card awarding + power bonus computation
- Bounty dashboard with Ghost Protocol resolution
- Truth Serum + Reputation Shield application
- Full career tier system with $VBV gating
- Underworld contract engine (assign/complete/abort)
- Kidnapping registration + multi-slot isolation
- Black market sale with Fence discount
- Fenced goods marketplace (list/buy/expire)
- Loan origination + liquidation → black market pipeline
- Counterfeit generation + Justice detection
- Salary dispenser with corporate + outlaw taxes
- Capital laundering (Wanted reduction)
- Raid insurance + Bounty Hunter bond
- Dutch Auction pricing for defaulted loans

### Scaffold / Partial ⚠️
- `HandleListRecoveryBounty` (`black_market_service.go:20-22`) — empty stub (logs only)
- `HandleSellMarketTokens` (`black_market_service.go:25-28`) — returns "unavailable in reconciled build"
- Static `HandleGetUnderworldContracts` list (`black_market_service.go:240-472`) — legacy scaffold, superseded by dynamic `ContractEngine` but retained
- `NarrativeService.GenerateNPCCommentary` (`narrative_service.go:19-87`) — generates flavor text for lobby/match events but has no mechanical effect on justice/criminality systems
- `ReplayEngine` (`replay_engine.go:1-305`) — captures frames for spectating; no justice/criminality logic, infrastructure-only

---

## 11. Key Cross-References

| System | Drives | Driven By |
|--------|--------|-----------|
| Justice Cards | Power bonus vs outlaws | BountyRank progression |
| Bounty Hunter Tax | Faucet funding | Successful bounty captures |
| Kidnapped Cards | Black Market liquidity | Kidnap Gambit success |
| Loan Defaults | Black Market listings | Collateral liquidation |
| Counterfeit Detection | Reputation + Wanted penalties | Forensic Analyst/Warden actions |
| Career Tier ($VBV) | Reward multipliers, fence discounts | AvgSustainedMicro liquidity |
| Ghost Protocol | Scrambled identity on dashboard | Enhanced tracking (Warden+) |
| Fence Fee | Faucet via token-sink router | Black market sales |
| Corporate/Outlaw Tax | Faucet funding | Salary payments |

---

## 12. Economic Invariants

- **Industrial Seal:** no funds burned or created in transfers — only rerouted (`justice_service.go:9`)
- **All math in uint64 μVBV** — float64 used only for UI display ratios
- **One-contract-at-a-time:** `ActiveUnderworldContractID` must be empty before accepting new contract (`underworld_contracts.go:550-552`)
- **Tax rerouting:** All taxes (corporate, outlaw, fence, bounty) route to `faucetBalanceMicro` to fund dynamic rewards
- **Black market cap:** max 50 items FIFO (`loan_service.go:357-359`)
