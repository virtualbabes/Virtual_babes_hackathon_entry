# RAG: Theme / Rivalry / Region System

> **The rivalry ARCHITECTURE has ONE owner: `AI-Brain/Rivalry-Matrix.md`** (three matrices, their
> weights, the path/civil-rank gates, the promotion/demotion lifecycle, the endpoint list). This
> file is a per-file CODE MAP for the same area; where the two disagree, the owner file wins and
> this one is wrong.

> Deterministic integer-only (PILLAR 2) theme projection, region-asset signatures, auto-detected rivalries, seasonal events, and faith governance. Backend authoritative, uint64 ledger.

---

## 1. Theme Engine (§27) — `theme_engine.go`

### 1.1 ThemeVector (§27.1) — per-player deterministic projection
- **Struct**: `theme_engine.go:76-83`
  - `Wallet`, `Tone` (0..6), `Element` (0..6), `Intensity` (0..100), `Entropy` (0..100), `MoodTag` (0=NEUTRAL default)
- **Tone axis** (`ThemeTone`): `theme_engine.go:49-59` — Void(0), Malevolent(1), Neutral(2), Benevolent(3), Order(4), Nature(5), Machine(6)
- **Element axis** (`ThemeElement`): `theme_engine.go:62-72` — None(0), Fire(1), Water(2), Earth(3), Air(4), Aether(5), Machine(6)
- **ComputeThemeVector(wallet)**: `theme_engine.go:133-144` — lock-free path via `computeThemeVectorLocked` (`theme_engine.go:149-236`)
  - **Inputs**: Holdings (micro-VBV via `CalculateTotalPortfolioValue`), PlayerStats (Reputation, Mojo, SocialRank, Relationships, Nurturing, Cunning, WantedLevel, Achievements, Wins, JobRole, EmployerClubID), cosmetic MoodTag from BondedAssetRegistry
  - **Tone logic** (`theme_engine.go:177-193`): benevolent vs malevolent axis — `good = nurt + social/4 + holdingsScore`, `bad = cun + want`. Thresholds at 120/180 with zero-want/nurt conditions.
  - **Element logic** (`theme_engine.go:197-210`): holdings-driven — ≥5,000 $VBV → Aether, ≥1,000 → Machine, ≥250 → Earth, ≥50 → Water, >0 → Fire, 0 → None
  - **Intensity** (`theme_engine.go:213-215`): `(holdScore*2 + standing*1 + social/3) / 4`, clamped 0..100
  - **Entropy** (`theme_engine.go:218-219`): `|good - bad| / 3`, clamped 0..100
  - **MoodTag** (`theme_engine.go:230-234`): sourced from `BondedAssetRegistry.MoodTagForWallet` (§23.5/§27.6, locked assets excluded)

### 1.2 OutcomeBias (§27.5) — deterministic integer bias
- **Signature**: `theme_engine.go:247-307`
- Returns integer delta in `[-BIAS_MAX, +BIAS_MAX]` percent of base (BIAS_MAX = 15, `theme_engine.go:46`)
- **Direction**: ToneBenevolent/Order/Nature → +1, Malevolent/Void → -1, Neutral → 0
- **Magnitude**: `(holdScore + social) vs (anti = wanted + cunning)`, scaled via `(pos - neg) * 15 / 200`
- **Entropy scale** (`theme_engine.go:295-296`): `50 + entropy/2` (50..100), widens variance at high entropy
- Applied at: Rivalry resolution (`rivalry_engine.go:190-204`), Seasonal event resolution (`seasonal_event_engine.go:141-146`)

### 1.3 MarketWeather (§27.2.1) — deterministic climate projection
- **Struct**: `theme_engine.go:103-107` — `WeatherState{Region, Climate, DisasterTier}`
- **Compute**: `theme_engine.go:630-658`
  - **Climate** from MarketVitality: ≥5000 CALM, ≥1000 BREEZY, ≥100 STORMY, else BLIGHTED
  - **DisasterTier** (0..3): vitality≥5000 + gravity≥5000 → 0; ≥1000 → 1; ≥100 → 2; else 3

---

## 2. WorldDynamicsSignature (§27.4) — 10 LOCKED Weighted Signals

### 2.1 The 10 Locked Weights (single source of truth)
- **Declaration**: `theme_engine.go:30-42`

| # | Signal | Weight | Source |
|---|--------|--------|--------|
| 1 | W_MARKET_VITALITY | 45,000 | `theme_engine.go:32` |
| 2 | W_CITIZEN_GRAVITY | 55,000 | `theme_engine.go:33` |
| 3 | W_THEME_COHERENCE | 25,000 | `theme_engine.go:34` |
| 4 | W_EVENT_DYNAMICS | 30,000 | `theme_engine.go:35` |
| 5 | W_PROFILE_IMPACT | 50,000 | `theme_engine.go:36` |
| 6 | W_FAITH_COHERENCE | 40,000 | `theme_engine.go:37` |
| 7 | W_DOMESTIC_COHERENCE | 35,000 | `theme_engine.go:38` |
| 8 | W_RUMOR_COHERENCE | 20,000 | `theme_engine.go:39` |
| 9 | W_ECON_PERK | 30,000 | `theme_engine.go:40` |
| 10 | W_ENTITY_LEGITIMACY | 40,000 | `theme_engine.go:41` |

### 2.2 ComputeWorldDynamicsSignature(region)
- **Struct**: `theme_engine.go:87-100` — WorldDynamicsSignature{Region, 10 signals, Score}
- **Compute**: `theme_engine.go:312-401`
  - **1. MarketVitality** (`theme_engine.go:316-328`): Σ ReserveBalance + DividendPoolMicro across MarketNodes, clamped 0..1,000,000
  - **2. CitizenGravity** (`theme_engine.go:331-339`): Σ `Reputation × (AttachmentTier+1)` for AI citizens in region, clamped 0..1,000,000
  - **3. ThemeCoherence** (`theme_engine.go:342, 487-528`): inverse-variance of player tone values in region (employer club proxy), 0..100
  - **4. EventDynamics** (`theme_engine.go:347-361`): +10 per unclaimed cache, +15 per active user event, clamped 0..100,000
  - **5. ProfileImpact** (`theme_engine.go:364, 533-553`): Σ Reputation + Wins/10 + Achievements×5 for region players, clamped 0..100,000
  - **6. FaithCoherence** (`theme_engine.go:369, 457-483`): Σ citizens(rituals×10 + 5 faithful) + Σ faith-clubs(rituals×8), clamped 0..1,000,000
  - **7. DomesticCoherence** (`theme_engine.go:374, 406-429`): Σ citizens(hasSpouse×8 + hasLover×4 + children×6 + aipets×3), clamped 0..1,000,000
  - **8. RumorCoherence** (`theme_engine.go:378, 558-586`): net sentiment Σ(positive − negative) × Strength×10, clamped ±100,000
  - **9. EconomicPerk** (`theme_engine.go:381, 591-608`): count of active tax-haven clubs, clamped 0..10,000
  - **10. EntityLegitimacy** (`theme_engine.go:385, 434-451`): Σ citizens(Certified×10 − BlackMarketAdopted×12), clamped 0..1,000,000
  - **Score** (`theme_engine.go:388-399`): `Σ(signal[i] × weight[i])` as int64

### 2.3 Wiring Status
- **NOW WIRED**: FaithCoherence, DomesticCoherence, EntityLegitimacy (previously deferred per §27.7 design doc)
- **Cosmetic only**: MoodTag from BondedAssetRegistry (§23.5/§27.6)
- **NOT fabricated**: signals without real data sources score 0 (per `theme_engine.go:311`)

---

## 3. AssetSignature / RegionRivalry Lifecycle (§25.10)

### 3.1 AssetSignature (§25.10.1) — region/territory strength
- **Struct**: `backend_types.go:405-418`
  - `Region`, `Territory`, `IsCapital`, `UserWorkers`, `AICitizenPride`, `ModelCitizens`, `Vehicles`, `Pets`, `WorldContent`, `Items`, `Events`, `Score`
- **ComputeAssetSignature(region, territory, isCapital)**: `rivalry_engine.go:23-90`
  - **UserWorkers** (`rivalry_engine.go:28-39`): Σ `RIVAL_W_WORKER_BASE × jobTier × salaryTier / 1000` (JobRole != "" && EmployerClubID == territory && Salary > 0)
  - **AICitizenPride** (`rivalry_engine.go:42-54`): Σ `Reputation × (AttachmentTier+1)` + ModelCitizens `(W_MODEL_CITIZEN × L) / 75`
  - **WorldContent/Events** (`rivalry_engine.go:58-69`): +W_WORLD_CONTENT per unclaimed cache, +W_EVENT_TYPE per active user event
  - **Items/Vehicles/Pets** (`rivalry_engine.go:70-76`): scaled by club treasury tier (Items/1,000,000 / 100)
  - **Score** (`rivalry_engine.go:79-88`): `Σ components × weights`, uint64

### 3.2 RegionRivalry — active rivalry pair
- **Struct**: `backend_types.go:421-433` — `RivalryID`, `Level`(TERRITORY|REGION), `SideA`, `SideB`, `AssetClass`, `ScoreA`, `ScoreB`, `Winner`, `Declared`, timestamps
- **RivalryEngine**: `backend_types.go:436-440` — `Signatures map[region|territory]`, `Rivalries map[RivalryID]`

### 3.3 Constants (`backend_types.go:382-402`)
| Constant | Value | Purpose |
|----------|-------|---------|
| RIVAL_COLLISION_THRESHOLD | 40,000 (40%) | Min overlap to trigger rivalry |
| RIVAL_SHOWCASE_BUFF_BPS | 5,000 (+5%) | Capital visibility buff |
| RIVAL_CACHE_BASE_MICRO | 50,000,000 ($VBV 50) | Resident loot cache base |
| RIVAL_GOV_REP | 1,000 | Governor rep gain on win |
| RIVAL_WORKER_BONUS_CAP | 200,000,000 ($VBV 200) | Per-worker cap |
| RIVAL_CREATOR_MULT | 1,200 (×1.2) | Built-asset multiplier |
| RIVAL_W_WORKER_BASE | 100,000 | User-worker base weight |
| **Asset-class weights**: | | |
| W_USER_WORKER | 100,000 | Worker component |
| W_AI_PRIDE | 80,000 | AI pride |
| W_MODEL_CITIZEN | 60,000 | Model citizen level |
| W_VEHICLE | 50,000 | Vehicle |
| W_PET_BLOODLINE | 45,000 | Pet bloodline |
| W_WORLD_CONTENT | 40,000 | World content |
| W_ITEM_ARCHETYPE | 35,000 | Item archetype |
| W_EVENT_TYPE | 30,000 | Event type |

### 3.4 Lifecycle
- **InitRivalryEngine()**: `rivalry_engine.go:14-19`
- **RecomputeAllSignatures()**: `rivalry_engine.go:93-115` — iterates AI citizens for regions, clubs for territories
- **DetectRivalries()**: `rivalry_engine.go:119-171`
  - O(n²) signature pairs; skips mismatched levels (region vs territory)
  - Collision: `min/max ratio ≥ RIVAL_COLLISION_THRESHOLD` (40%)
  - Determines `AssetClass` via `dominantClass()` (`rivalry_engine.go:224-248`)
- **ResolveRivalry(rivalryID)**: `rivalry_engine.go:174-221`
  - Winner = higher score; **4-part prize**: resident cache (RouteCriminalTax FaucetShare 1.0), governor rep +1000
  - ThemeEngine hook: precomputes `ComputeWorldDynamicsSignature` for winner region
- **dominantClass()**: `rivalry_engine.go:224-248` — picks highest-weighted component label

---

## 4. Career Rivalry XP Pairs — `rival_career_engine.go`

### 4.1 Rival XP Deltas (GetRivalXPDelta)
- **Function**: `rival_career_engine.go:254-285`

| Pair Name | Delta | Type |
|-----------|-------|------|
| BountyHunter↔Kidnapper | -15 | Antagonistic |
| ForensicAnalyst↔Gossip | -10 | Antagonistic |
| TaxAuditor↔Launderer | -10 | Antagonistic |
| Warden↔HeistPlanner | -10 | Antagonistic |
| SectorPeacekeeper↔Smuggler | -10 | Antagonistic |
| IntelAgent↔ArcNetOperative | -10 | Antagonistic |
| JusticeRecruiter↔BountyHunter | +8 | Synergistic |
| Launderer↔Fence | +5 | Synergistic |
| HeistPlanner↔Kidnapper | +12 | Synergistic |
| AOS↔SectorPeacekeeper | +6 | Synergistic |
| TaxAuditor↔JusticeCommissioner | +7 | Synergistic |
| Gossip↔ForensicAnalyst | +5 | Synergistic |

### 4.2 Pair Detection & XP Award
- **GetRivalPairName(attacker, defender)**: `rival_career_engine.go:288-320` — bidirectional match against canonical pair table
- **TrackRivalInteraction()**: `rival_career_engine.go:325-355` — core wiring, called at criminality handlers
  - Antagonistic (delta<0): `xpAwarded = baseXP × (1 + |delta|/100) × modifier`
  - Synergistic (delta>0): `xpAwarded = baseXP × (1 + delta/100) × modifier`
  - `modifier = 1.0 + 0.1 × tier` (GetRivalPairModifier)
- **EvaluateCrossCareerXP()**: `rival_career_engine.go:378-409` — both attacker & defender get XP; defender gets 30% of base

### 4.3 Career Tier System (PILLAR 13)
- **Tier thresholds** (`rival_career_engine.go:26-33`): Peon(0), Apprentice(5K $VBV), Journeyman(25K), Expert(100K), Master(500K), Boss(2M) — all micro-VBV sustained
- **CheckCareerTierGate()**: `rival_career_engine.go:138-193` — validates AvgSustainedMicro against tier requirement; demotion warning after 7 days
- **XP multiplier** (GetVBVGatingMultiplier): `rival_career_engine.go:572-589` — ×1 Peon, ×2 Apprentice, ×4 Journeyman, ×8 Expert, ×16 Master, ×32 Boss

---

## 5. Seasonal Event System (§26) — `seasonal_event_engine.go`

### 5.1 SeasonEvent Lifecycle
- **Struct**: `backend_types.go:196-208`
- **Phases** (`backend_types.go:176-180`): ANNOUNCEMENT → ACTIVE → RESOLUTION → TREASURY_PAYOUT
- **SeasonEventType** (`backend_types.go:184-192`): HARVEST_FEST, SHADOW_AUCTION, TERRITORY_WAR, CRYPTO_RAID, DIAMOND_WEEK

### 5.2 Core Operations
- **CreateSeasonEvent()**: `seasonal_event_engine.go:31-69` — creates announcement-phase event with multiplier (0..5×) and treasury budget
- **ActivateEvent()**: `seasonal_event_engine.go:72-91` — transitions ANNOUNCEMENT → ACTIVE, broadcasts WebSocket
- **RegisterForEvent()**: `seasonal_event_engine.go:94-110` — wallet registers for active/announcement event
- **GetEventMultiplier()**: `seasonal_event_engine.go:113-123` — returns active event multiplier (1.0 if none)
- **ResolveEvent()**: `seasonal_event_engine.go:126-151` — ACTIVE → RESOLUTION, calls ComputeThemeVector for each participant (OutcomeBias hook)
- **DistributeEventRewards()**: `seasonal_event_engine.go:154-191` — equal split of treasury budget per participant
- **AutoTriggerEvents()**: `seasonal_event_engine.go:194-208` — auto-resolves expired active events

### 5.3 Treasure Cache (§26.1)
- **Struct**: `backend_types.go:230-244` — CacheID, Region, X/Y/Z, RewardMicro, RewardKind, BondMicro, Hidden, Claimed
- **SpawnTreasureCache()**: `seasonal_event_engine.go:681-698`
- **ClaimTreasureCache()**: `seasonal_event_engine.go:701-734` — optional entry bond (RouteCriminalTax), payout + audit log

### 5.4 User-Authored Events (§26.2)
- **Struct**: `backend_types.go:247-261` — EventID, Type, Title, Description, Region, RewardMicro, EntryMicro, BondedNFT, Creator, RoyaltyBps, Status
- **UserEventType** (`backend_types.go:220-227`): TREASURE_HUNT, SEARCH_RESCUE, GANG_BASHING, WILD_BOT_HUNT, PET_BREEDING_SHOW, SYSTEM_EVENT
- **CreateUserEvent()**: `seasonal_event_engine.go:737-761` — royalty ≤1000 bps (§16 Q22)
- **EnterUserEvent()**: `seasonal_event_engine.go:764-794` — entry fee + author royalty via §23 split
- **GetRegionViews()**: `seasonal_event_engine.go:798-874` — aggregate regions for explorer (AI citizens + caches + events + §30 power overlay)

---

## 6. Region Explorer & Power Overlay (§25.3/§30)

### 6.1 RegionView
- **Struct**: `backend_types.go:264-272` — Region, Cap (1+idx), Count, Caches, Events, EntityPowerOverlay
- **GetRegionViews()**: `seasonal_event_engine.go:798-874` — builds region list from AI citizens, caches, user events

### 6.2 Stat Overlay (§30)
- **Struct**: `stat_overlay.go:31-43` — EntityID, EntityType, Owner, Stats, EffectivePower, DominantStat, PowerBuff, Region, IsMature, CanEarnVBV, CanEarnStats
- **EntityType** (`stat_overlay.go:18-28`): player, ai_citizen, llm_bot, rogue_bot, child_bot, pet, vehicle
- **ComputeStatOverlay()**: `stat_overlay.go:56-89` — effective power via `ComputeEffectivePowerLevel`; immature pets/children earn only stats (no VBV)
- **ApplyPowerBuff()**: `stat_overlay.go:131-141` — signed buff applied by events/rivalries
- **Global singleton**: `statOverlayEngine` (`stat_overlay.go:51-53`)

---

## 7. Item Shop & Archetype System (§14)

### 7.1 ItemArchetype (§14.2)
- **Struct**: `item_shop_archetype.go:27-33` — Archetype, ClubType, Careers, Pathway, Rivalry
- **Registry**: `item_shop_archetype.go:36-49` — 12 archetypes mapping ClubType → careers → AI pathway
  - Liquid (Fence/Smuggler), Direct (HeistPlanner), Transit (Smuggler), Cyber (ArcNetOperative), Bounty (BountyHunter), Authority (JusticeCommissioner), Containment (Kidnapper), Forensic (Launderer/MutationLogAuditor), Shadow (Gossip), Syndicate (UnderworldBoss), Justice (Warden/Commissioner/Recruiter), Hardware (Security)

### 7.2 BuiltItem (§14.5)
- **Struct**: `item_shop_archetype.go:62-70` — extends ShopItem with ItemID, CreatorWallet, BondedNFT, PaidMicro, PowerScale, CreatedAt
- **PowerScale** (§14.3): `item_shop_archetype.go:96-106` — `clamp(paidMicro / BaseReferenceMicro, 0.5, 3.0)`, BaseReferenceMicro = 100 $VBV
- **BuildItem()**: `item_shop_archetype.go:124-187` — routes price via RouteCriminalTax (50/25/25 faucet/club/gov), synthesizes bond ID if empty, broadcasts WebSocket
- **Persistence**: `item_shop_archetype.go:230-287` — atomic write (temp+rename), JSON file `item_registry.json`

### 7.3 ShopItem Registry
- **Struct**: `shop_registry.go:22-34` — ID, Name, Price, ClubType, HeistSuccessModifier, MutationSuccessModifier, MojoBonus, RequiredMojo, RequiredRole, IsMasterTier
- **GlobalShopRegistry**: `shop_registry.go:36-326` — ~60 items across Elemental/Tactical/Vitality/Hardware/Justice/Intelligence families
  - Career-locked: Bounty License (50K), audit_warrant (800K), compliance_notice (300K), etc.
  - Justice Hegemony Path: Enforcer/Warden/Commissioner cards (2.5K–10K)

---

## 8. Infrastructure Leasing

- **Struct**: `infrastructure_lease.go:18-27` — ID, TenantID, SystemType, MonthlyRate, LeaseStart/End, Active, AutoRenew
- **LeaseEngine**: `infrastructure_lease.go:29-36` — global singleton
- **Available Systems** (`infrastructure_lease.go:72-85`): wallet(1M), auth(500K), economy(2M), ai(1.5M), social(800K), tournament(1.2M), leaderboard(600K), crosschain(2.5M), creator(900K), analytics(700K) — all micro-VBV/month

---

## 9. Governance (§25.9)

- **Struct**: `governance.go:19-27` — GovernanceWeight{Trust, Reputation, Economic, Competitive, Community, Creative, Total}
- **RegionalGovernor**: `governance.go:30-39` — Wallet, Region, ElectedAt, TermExpiresAt (3 months), Weight, VotesFor/Against, Active
- **ComputeWeight()**: `governance.go:66-93` — all scores derived from `TotalReputation` (persistentIdentity profile)
- **Election flow**: RegisterCandidate → Vote → CloseElection (`governance.go:96-171`)
- **Global singleton**: `governanceEngine` (`governance.go:60-63`)

---

## 10. Religion Governance (§32/faith)

- **Struct**: `religion_governance.go:27-41` — ReligionID, Name, Dogma, Governor, PricePaid, Members, Rivalries, FaithPower, Coherence, RitualCount, Region, timestamps
- **Cap**: `MaxReligionsPerServer = 24` (`religion_governance.go:22`), DefaultReligionPrice = 5 $VBV
- **InitializeFaithReligions()**: `religion_governance.go:54-94` — 24 faucet-owned default religions (purist/syncretic/orthodox dogmas)
- **BuyGovernorship()**: `religion_governance.go:97-123` — buy unowned (faucet) religion at current price → faucet
- **BuyoutReligion()**: `religion_governance.go:127-175` — buyout = PricePaid × 1.10, ALL members auto-convert, old governor gets nothing, price → faucet
- **FaithPower**: `religion_governance.go:265-267` — `members×100 + rituals×10 + coherence`
- **Rituals**: `religion_governance.go:221-235` — +10 Coherence per ritual, 0.1 $VBV fee → faucet
- **HighTierRivals**: `religion_governance.go:193-199` — top 12 religions by power

---

## 11. Launchpad (Creator Accelerator)

- **Struct**: `launchpad.go:32-47` — ID, Creator, Title, Description, Category, GoalMicro, RaisedMicro, Status, Backers, BackerCounts, RewardTiers
- **LaunchStatus** (`launchpad.go:20-29`): draft → pending → active → funded → integrated (or failed)
- **Flow**: CreateLaunch → ActivateLaunch → BackLaunch (at goal → LaunchFunded) → IntegrateLaunch
- **Global singleton**: `launchEngine` (`launchpad.go:64-66`)

---

## 12. Wiring Summary — What's Live vs Scaffold

### WIRED (reads real data)
- ThemeVector from holdings + PlayerStats + BondedAssetRegistry MoodTag
- OutcomeBias with entropy scaling
- WorldDynamicsSignature (all 10 signals computable)
- MarketWeather from signature
- AssetSignature from workers + AI citizens + caches/events + club treasury
- DetectRivalries + ResolveRivalry (4-part prize)
- Career XP rival pairs (12 pairs, bidirectional)
- Seasonal event full lifecycle (create → activate → resolve → distribute)
- Treasure caches (spawn/claim with bond)
- User events (create/enter with royalty)
- RegionView aggregation
- Stat overlay (power buff, region/owner queries)
- Item build/bind with RouteCriminalTax routing
- Religion buyout/ritual/member system
- Governance weight computation + elections

### SCAFFOLD / STUB
- **DLC Registry** (`shop_registry.go:20-20`): initialized with example products, no real console integration
- **Launchpad**: full lifecycle present but no integration with item/shop ecosystems
- **Infrastructure Leasing**: lease creation/listing only, no active billing enforcement
- **Random event generation**: `GenerateRandomSeasonalEvent()` uses `rand` (non-deterministic, dev-only)
- **Player progression gates**: `handlePlayerProgression` (`theme_engine.go:824-1181`) exposes feature states (dormant/dawning/alive) but gating logic is UI-facing, not ledger-enforced
- **Career tier gates**: `CheckCareerTierGate` validates sustained $VBV but demotion is warning-only

---

## 13. Key Determinism Notes (PILLAR 2)
- All math is integer; no RNG on ledger-facing paths
- `ComputeWorldDynamicsSignature` scores 0 for signals without real sources (never fabricated)
- `clampInt64`/`clampInt` helpers enforce bounds
- Asset signature weights in `backend_types.go:392-402` are bps×1000 (integer ratio)
- OutcomeBias returns integer delta, never exceeds ±15% of base
