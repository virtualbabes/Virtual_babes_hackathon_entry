# THE RIVALRY MATRIX — ONE OWNER

> **Status:** authoritative for the rivalry architecture. **Last reconciled:** 2026-09-15.
> **Owner of record:** this document. Code owners are named per matrix below; a claim about
> rivalry that this file does not make should be treated as unverified until it is added here.

## WHY THIS FILE EXISTS

The rivalry system was documented in **seven** places with no owner — `AI-Brain/AI-Citizen-Design.md`
§27.4, `AI-Brain/Game-Mechanics-Index.md` §"Rivalry Matrix", `AI-Brain/File-Flow-Overview-1.md`
§10.3, `AI-Brain/RAG/03_theme_rivalry.md`, `AI-Brain/MASTER-PLAN.md`, `AI-Brain/Problems.md` §24 and
`AI-Brain/Manuals/RIVALRY-MANUAL.md`. They disagreed, and the manual disagreed with the *code*:
it described career types (`Warrior`, `Mage`, `Rogue`, `Healer`, `Engineer`, `Breeder`) that exist
nowhere in the repository, and named **five endpoints that are registered nowhere** (listed under
"Corrections applied" below). A reader could not tell which matrix a number came from.

**There are THREE matrices, not one.** They are separately owned, separately calibrated, and a
player meets all three in different places. Everything else in the docs is a view of one of them.

---

## 1. THE THREE MATRICES

| Stage id | Owner (code) | What it scores | Where a player meets it |
| --- | --- | --- | --- |
| `path` | `career_path.go` | a PLAYER's alignment against another player's: `justice` ↔ `criminal` is the one DIRECT rivalry; `neutral` holds no side | the career panel's path matrix; the direct-rival XP bonus |
| `career` | `rival_career_engine.go` (`rivalPairTable`, `GetRivalXPDelta`) | ROLE ↔ ROLE: 13 declared pairs, priced −15..+12 | every cross-career interaction (combat, courthouse, contracts, heists) |
| `region` | `rivalry_engine.go` + `theme_engine.go` | TERRITORY/REGION asset strength and living-world vitality | `/api/regions`, `/api/rivalry/*`, the 3D world, the leaderboard hub |

`career_path.go` serves the stage ids (`MatrixStagePath` / `MatrixStageCareer` / `MatrixStageRegion`)
on every read, so a UI never shows a number without the layer it came from.

---

## 2. MATRIX 1 — THE PATH MATRIX (`career_path.go`)

### 2.1 Three paths, one key each

`justice` · `criminal` · `neutral`. `PathKey()` folds casing, separators and whitespace, and carries
**three declared aliases, each naming its source**:

| Alias | Folds to | Source in the repository |
| --- | --- | --- |
| `law` | `justice` | the justice item families and `item_shop_archetype.go` |
| `underworld` | `criminal` | the criminal side's own vocabulary; `career_tree.js` used it as a faction label |
| `hybrid` | `neutral` | a pathway that serves both sides; the three paths are exclusive, and "serves both, owns neither" IS the neutral position |

An undeclared spelling folds to ITSELF and `IsCareerPath()` returns false, so an unknown club type
can never be read as a career path.

### 2.2 The relation table (4 values, resolved on the server)

| A | B | Relation | Bonus |
| --- | --- | --- | --- |
| justice | criminal | `direct` | +10% (`PathDirectRivalryBonusBps`) |
| criminal | justice | `direct` | +10% |
| justice | justice | `shared` | 0 (an ally relation) |
| criminal | criminal | `shared` | 0 |
| neutral | neutral | `interpreted` | 0 — a neutral reads a rivalry off another neutral |
| neutral | (either side) | `interpreted` | 0 — holds no direct enemy; reads the matrices instead |
| (unset) | anything | `none` | 0 |

**Neutrality is a choice, not a strict upgrade.** Neutral buys BREADTH — a neutral player may
promote into EITHER side's careers — and pays for it with no direct-rival leverage. Without that
trade the other two paths would be strictly worse.

### 2.3 The path of each career — READ OFF MATRIX 2, not re-declared

`careerPathByRole` assigns every career a side, and the assignment is **derived from the enemy pairs
in `rivalPairTable`**: the attacker side of every enemy pair is JUSTICE, the defender side is
CRIMINAL. The careers that appear in no enemy pair are placed from the pathway they already sit in
(`aiPathwayByCareer`). Each entry names its evidence in a comment.

`TestCareerPathAgreesWithTheRivalMatrix` **pins the agreement**: every ENEMY pair must CROSS paths
and every ALLY pair must SHARE one. Add a pair that contradicts the assignment and the test fails,
so the two matrices cannot drift apart.

`PathAllowsRole(path, role)`: a sided player may hold their own side's careers only; a NEUTRAL
player may hold either; a career the matrix does not know is refused (fail closed).

---

## 3. MATRIX 2 — THE CAREER-PAIR MATRIX (`rivalPairTable`)

One row per UNORDERED pair, and the scan is order-independent, so a duplicate row could never fire.
`GetRivalXPDelta` prices each by NAME.

| Pair | Sides | Delta | Kind |
| --- | --- | --- | --- |
| `BountyHunter↔Kidnapper` | J ↔ C | −15 | enemy |
| `ForensicAnalyst↔Gossip` | J ↔ C | −10 | enemy |
| `TaxAuditor↔Launderer` | J ↔ C | −10 | enemy |
| `Warden↔HeistPlanner` | J ↔ C | −10 | enemy |
| `SectorPeacekeeper↔Smuggler` | J ↔ C | −10 | enemy |
| `IntelAgent↔ArcNetOperative` | J ↔ C | −10 | enemy |
| `MutationLogAuditor↔Kidnapper` | J ↔ C | 0 | enemy (P2-D10; declared antagonistic in the table, **unpriced** in the XP switch) |
| `JusticeRecruiter↔BountyHunter` | J ↔ J | +8 | ally |
| `Launderer↔Fence` | C ↔ C | +5 | ally |
| `HeistPlanner↔Kidnapper` | C ↔ C | +12 | ally |
| `AOS↔SectorPeacekeeper` | J ↔ J | +6 | ally |
| `TaxAuditor↔JusticeCommissioner` | J ↔ J | +7 | ally |
| `JusticeRecruiter↔MutationLogAuditor` | J ↔ J | 0 | ally (P2-D8) |

A NEGATIVE delta is antagonistic (the attacker gains); a POSITIVE delta is synergistic. The
relation is resolved by `GetRivalPairName`, which compares **by role** (`RoleKey`), so a caller
passing `"IntelAgent"` finds the pair the table declares as `"Intel-Agent"`.

---

## 4. MATRIX 3 — THE REGION/WORLD MATRIX (`rivalry_engine.go`, `theme_engine.go`)

### 4.1 Asset signature (`ComputeAssetSignature`) — weighted, integer, `bps×1000`

| Component | Weight | Source |
| --- | --- | --- |
| `W_USER_WORKER` | 100,000 | employed players (`career.go`: `JobRole != "" && EmployerClubID == territory && Salary > 0`) |
| `W_AI_PRIDE` | 80,000 | Σ `Reputation × (AttachmentTier+1)` |
| `W_MODEL_CITIZEN` | 60,000 | Σ tier, clamped 75 |
| `W_VEHICLE` | 50,000 | club treasury tier proxy |
| `W_PET_BLOODLINE` | 45,000 | club treasury tier proxy |
| `W_WORLD_CONTENT` | 40,000 | per unclaimed cache in the region |
| `W_ITEM_ARCHETYPE` | 35,000 | club treasury tier proxy |
| `W_EVENT_TYPE` | 30,000 | per unresolved user event |

Calibration constants (`backend_types.go`): `RIVAL_COLLISION_THRESHOLD = 40000` (a 40% signature
overlap auto-declares a rivalry), `RIVAL_SHOWCASE_BUFF_BPS = 5000`, `RIVAL_CACHE_BASE_MICRO = 50e6`,
`RIVAL_GOV_REP = 1000`, `RIVAL_WORKER_BONUS_CAP = 200e6`, `RIVAL_CREATOR_MULT = 1200`.

Prize on resolution: the 4-part award — showcase buff, resident cache (routed through the ONE sink
door), governor reputation, worker bonus.

### 4.2 World-dynamics signature (`ComputeWorldDynamicsSignature`) — 10 LOCKED weights

| # | Signal | Weight |
| --- | --- | --- |
| 1 | `W_MARKET_VITALITY` | 45,000 |
| 2 | `W_CITIZEN_GRAVITY` | 55,000 |
| 3 | `W_THEME_COHERENCE` | 25,000 |
| 4 | `W_EVENT_DYNAMICS` | 30,000 |
| 5 | `W_PROFILE_IMPACT` | 50,000 |
| 6 | `W_FAITH_COHERENCE` | 40,000 |
| 7 | `W_DOMESTIC_COHERENCE` | 35,000 |
| 8 | `W_RUMOR_COHERENCE` | 20,000 |
| 9 | `W_ECON_PERK` | 30,000 |
| 10 | `W_ENTITY_LEGITIMACY` | 40,000 |

`OutcomeBias` returns an integer delta capped at ±15% (`BIAS_MAX`). A signal with no real source
scores 0 and is never fabricated.

---

## 5. THE CAREER LIFECYCLE (`career_path.go`)

### 5.1 The path choice — made ONCE, when a region is opened

| Step | Rule |
| --- | --- |
| Gate | the player OPENS A REGION while holding **2+ territories** (the engine's own `HandleOpenRegionalManager` already refuses below two, so "governor" and "eligible" cannot disagree) |
| Choice | `justice` \| `criminal` \| `neutral` — **once**. A second choice is refused, because an alignment that can be flipped on demand carries no weight in any matrix |
| Server-owned | the civil rank, the eligibility, every requirement and every price. The client names the path and nothing else |

### 5.2 Promotion — the level-cap unlock

A role is promoted when ALL FIVE hold, evaluated in this order, each refusal naming the ONE that failed:

1. **the path** — a path is held, and it allows the career (`PathAllowsRole`)
2. **the level cap** — `CareerXP.LessonLevel >= gate.MinLessonLevel`
3. **the role tier** — `getTierFor(role) >= gate.MinRoleTier` (the 1..4 band; a gate above
   `CareerRoleTierMax` is refused AT BOOT, because it could never be reached)
4. **the sustained $VBV** — `AvgSustainedMicro >= gate.MinSustainedMicro`, on the SAME ladder
   `GetVBVGatingPermille`/`CheckCareerTierGate` already price (5K → 2M), reported as an exact shortfall
5. **the civil rank** — `civilRankIndex(standing.Tier) >= civilRankIndex(gate.MinCivilTier)`

`PromoteCareerLocked` is the WRITER `PromotedRoles` never had, and it is idempotent.

**The five gates UNLOCK a career; they never GRANT it.** Nothing in the repository promotes automatically,
and nothing may be added that does — see §5.5.

### 5.3 Demotion — the $VBV drain that takes the career with it

| Outcome | When | Effect |
| --- | --- | --- |
| `warning` | the sustained $VBV fell below the promoted career's requirement | NOTHING is taken. The player is told the EXACT shortfall and the deadline |
| `cleared` | the balance recovered inside the grace period | the warning is WITHDRAWN (without this a recovered player stayed permanently failed) |
| `demoted` | the grace period (`DemotionGracePeriodDays` = 7) expired with the shortfall still open | the role is removed from `PromotedRoles`, a `CareerDemotion` record keeps the exact arithmetic, and the stored `JobRole` stops naming it |

An **undeclared** career has no requirement to fall below and is never judged, so a hand-written
record cannot be silently taken away. The grace period is ONE clock per player, so only the role
with the **worst** shortfall is warned at a time — several careers off one clock would misreport
each of their deadlines.

### 5.4 The civil rank — the worker-society gate

| Rank | Derived from (never declared) | Why the gate exists |
| --- | --- | --- |
| `user` | no territory (a SHOP with no territory is reported, and is still a `user`) | a role that only works requires nothing behind it |
| `manager` | ≥1 territory, counted alliance-aware exactly as `IsClubRegionalLocked` counts | a role that directs an organisation's operations needs one |
| `governor` | a club whose REGION is opened (`RegionName != ""`) | a role whose authority IS a region (Judge, Underworld Boss, Justice Commissioner) needs a region |

The code already said this before the gate existed: `AIPathwaySyndicate` (`ai_citizen_engine.go`) is
commented *"UnderworldBoss / Judge (governor-gated)"*.

### 5.5 The upgrade is an OFFER — "unlocked, never forced"

The operator's rule (2026-09-15 i): *"a user does not have to upgrade career it is only unlocked to upgrade
if level cap allows it, a user may request staff users to upgrade when they notice there staff can upgrade
and it is upto the user to upgrade them seflves, yes they may be notified not forced."*

| Concern | Rule | Owner |
| --- | --- | --- |
| What is UNLOCKED | `UnlockedCareerRolesLocked(wallet)` asks the ONE gate evaluator, so "unlocked" cannot become a second, softer gate. It GRANTS NOTHING | `career_path.go` |
| The offer, served | `upgrades { statement, is_optional, unlock_basis, notice_rule, unlocked_roles, unlocked_count, staff[], staff_basis, can_request_staff, requests_to_me }` — derived on every read, never a write | `CareerUpgradeViewLocked` |
| The notice | ONE notice per unlock, to the ONE wallet, from the 24 h liquidity daemon; the record is CLEARED on promotion, so a career lost to a demotion and re-earned re-notifies | `CareerUnlockNoticesLocked` |
| The employer | "staff" = the roster (`Club.Staff`) of a club the caller OWNS. The employer may SEE who is ready and may **REQUEST** | `RequestStaffCareerUpgradeLocked` |

**APPROVAL IS A REQUEST, NOT A PROMOTION.** The request is stored on the EMPLOYER's own club
(`Club.StaffUpgradeRequests`, key `<RoleKey(staffWallet)>:<RoleKey(role)>`) and delivered to the one staff
wallet. It **never** writes the employee's `PromotedRoles`, `JobRole` or level — only that employee's own
door can, which is what "not forced" means mechanically. Seven refusals, each naming what is wrong, and
every one moves nothing; the seventh is the important one: **an employer cannot ask for an upgrade the level
cap has not opened**, evaluated on the EMPLOYEE's record, so a request can never become a way round the
gate. One request per (staff, role): a repeat is REPORTED, not re-sent, because a nudge that can be repeated
on demand is pressure rather than a notification.

---

## 6. WHERE EACH MATRIX REACHES THE PLAYER

| Surface | Module | Shows |
| --- | --- | --- |
| Career panel (WD ▸ Careers & Factions ▸ Career) | `Public/js/career_tree.js` | the path matrix, the choice, the civil ladder, every career's gate, promotions, demotions, the outstanding warning, and the three matrices **with their owners** |
| World Dashboard ▸ Criminality hub | `Public/js/criminality.js` | the courthouse and the Intel-Agent cyber-intercept, whose XP carries the career-pair delta |
| Rivalry viewer | `Public/js/rivalry_viewer.js` | the region/territory matrix (`/api/rivalry/*`) |
| 3D world / leaderboard hub | `Public/js/world3d.js`, `leaderboard_region.js` | region vitality, the 10 world-dynamics signals, the vitality crown |
| Portfolio (READ-ONLY) | `Public/js/portfolio.js` | the player's own associations: careers, rivalry state, clubs |

Endpoints (registered in BOTH `server_main.go` and `console_server.go` unless noted):

```
GET  /api/career/path         the whole system for the connected wallet
POST /api/career/path/choose  {path}   the one choice
POST /api/career/promote      {role}   the level-cap promotion
GET  /api/career/progress     the legacy progression snapshot (unchanged)
GET  /api/rivalry/world-dynamics | /factions | /join | /request | /action | /state
     /recompute | /detect | /list | /resolve
GET  /api/regions
GET  /api/pet-battle/list            POST /api/pet-battle/challenge | /resolve
GET  /api/faith/coherence | /religions | /religion/rivalry
POST /api/faith/war-gambit | /religion/buy | /buyout | /join | /ritual
```

---

## 7. CORRECTIONS APPLIED TO THE OTHER DOCUMENTS

`AI-Brain/Manuals/RIVALRY-MANUAL.md` (v1.0, 2026-09-01) made claims this repository does not
support. Corrected with a pointer to this file:

| Claim in the manual | Reality |
| --- | --- |
| career types `Warrior`, `Mage`, `Rogue`, `Healer`, `Engineer`, `Breeder` | **no such careers exist**. The real careers are the ~20 roles in `rivalPairTable` + `aiPathwayByCareer` (Gossip, Warden, Forensic Analyst, Bounty Hunter, Kidnapper, …) |
| "Rivalry Factions: Vehicle Builders vs Breeders" | not a system in the repository. The two SIDES are `justice` and `criminal`; the factions that DO exist are the faiths and the pathways (`P-Shadow`…`P-Boss`) |
| `GET /api/rivalry/career` | **registered nowhere** |
| `POST /api/rivalry/define-set` | **registered nowhere** |
| `GET /api/rivalry/sets` | **registered nowhere** |
| `POST /api/rivalry/regional` | **registered nowhere** |
| `GET /api/regions/:name/dynamics` | **registered nowhere** — `/api/regions` is a bare-list GET with no path-segment support |
| "Career Rivalries: +5% XP at Journeyman … +20% at Grandmaster" | there is no `Grandmaster` rung. The XP relation is the `GetRivalXPDelta` table (§3) plus the path bonus (§2.2) |
| "Rivalry trades: +5% transaction fee" | fees are the declared `RouteCriminalTax` splits, not a rivalry constant |

---

## 8. KNOWN LIMITS AND RECORDED DEFECTS (measured, not assumed)

1. **`MutationLogAuditor↔Kidnapper` is declared antagonistic but priced 0.** It is an enemy pair in
   the table (P2-D10) while the XP switch does not price it. NOT changed here: it alters XP economics
   and is the operator's call.
2. **`Gossip↔ForensicAnalyst` (+5) is DEAD in `GetRivalXPDelta`.** The pair table's row was removed as
   an unreachable duplicate, so no caller can produce that name. Removing the NAME rather than the
   PRICE is the operator's call.
3. **THREE different "tier" scales exist** — `getTierFor` (1..4 role band), `CareerXP.GetCareerTier`
   (`xp/1500`, unbounded), `CheckCareerTierGate` ($VBV, 0..5). Only the first is the career-role
   tier; a gate written against another is silently unsatisfiable, which is why
   `assertCareerPathDialect` refuses it at boot.
4. ✅ **THE FLOAT ARITHMETIC IS GONE (FIXED 2026-09-15 j).** `TrackRivalInteraction`,
   `EvaluateCrossCareerXP`, `computeScaledXP` and `GetRivalPairModifier` all multiplied career XP by a
   `float64`, against the Architecture Ledger. They now compose **permille integers** —
   `GetRivalXPGainPermille`, `GetRivalPairModifierPermille`, `RivalDefenderSharePermille`,
   `rivalTierBonusPermille`, `computeScaledXPPermille`, and the public
   `ComputeXPWithBonusesPermille`. The float getter survives ONLY as a DISPLAY wrapper that a test pins
   to the integer owner, and `math` is no longer imported by `rival_career_engine.go`. The conversion is
   exact rather than an approximation, because the float expressions were already `total/1000`.
5. ✅ **THE COURTHOUSE DEAD ARITHMETIC IS FIXED (FIXED 2026-09-15 j).** Both resolution points read the
   DEFENDER's 30 % share into `rivalXP` and compared it against the base (`4 > 15` — false for ever),
   and the `uint64(rivalXP - base)` that followed would have underflowed to
   18,446,744,073,709,551,605 had the guard ever passed. Both now resolve through
   `ResolveRivalXPAward`, which returns a STRUCT whose `Bonus` is computed and guarded in ONE place, so
   a caller cannot select the wrong field. **This also closed the §2.2 hole:** the PATH matrix's direct
   justice↔criminal bonus was computed, SERVED and pinned by a test while being awarded in ZERO places,
   so the courthouse never acted as the rival it was declared to be. It does now, at BOTH resolution
   points (`courthouseRivalAwardsLocked` → `payCourthouseRivalAwardsLocked`).
6. **`PromotedRoles` now has a writer but no automatic caller.** A promotion is granted by
   `POST /api/career/promote` (or `PromoteCareerLocked`); nothing invokes it on its own yet — and
   deliberately so: the operator's rule is *"unlocked, never forced"* (`career_path.go` §7.5).
7. **A SELF-LOCK GATE NOW EXISTS — `career_award_lock_test.go`.** Six functions in this repository have
   been found holding the lobby lock and then calling the SELF-LOCKING `l.TrackCareerXP` (which freezes
   every request and every WebSocket in the process, not one request), and three of them hid behind dead
   code, so no behaviour test could see them. The gate parses every non-test `.go` file for that shape
   and carries its own negative control. Measured: **94 non-test Go files, 0 violations.**
## 9. HOW THIS IS VERIFIED (two tiers, because they can fail differently)

The DOMAIN and the HTTP boundary are pinned by `career_path_test.go` (20 tests): the path fold and its
aliases, the agreement between `careerPathByRole` and `rivalPairTable` (every enemy pair crosses paths,
every ally pair shares one), the reachability of every declared gate, the warn → grace → demote
lifecycle, and the door's 401/400/403/405 refusals. **A GREEN Go SUITE DOES NOT MEAN THE PLAYER SEES
ANY OF IT** — the client tier is a separate failure mode, and this file's whole subject (a taxonomy the
server had never served) was a client-tier defect.

`npm run verify:entry` now carries **nine `career.*` assertions** (added 2026-09-15 h) which compare what
is **RENDERED** against what was **SERVED in the same run** — never against a list typed into the probe:

| Assertion | What it proves |
| --- | --- |
| `career.doors_refuse_an_anonymous_caller` | 401 on all three doors, not an answer with a taxonomy |
| `career.client_cannot_declare_its_own_gate` | a body naming `civil_tier` is refused **400** by the decoder, before any lookup |
| `career.read_serves_the_whole_taxonomy` | 3 paths / 9 matrix cells / 3 ranks / ~20 careers / 3 matrices / the pair table / the rules, and every ineligible career STATES why |
| `career.three_paths_and_the_one_direct_rivalry` | justice↔criminal is the ONLY `direct` cell (priced 1000 bps, everything else 0) and the neutral row is entirely `interpreted` |
| `career.civil_ladder_is_user_manager_governor` | the three ranks, and a wallet owning nothing is DERIVED as `user` |
| `career.the_three_matrices_name_their_owners` | each matrix names the FILE that owns it (`career_path.go`, `rival_career_engine.go`, `rivalry_engine.go`) |
| `career.client_renders_the_served_taxonomy` | the panel paints the served labels, one row per served career, one missing-gate line per ineligible career, the 3 ranks, the 3 owners, every rule |
| `career.the_leaf_names_a_published_opener` | the WD declares the feature, ROUTES OUT, and `openCareers` is really published |
| `career.client_declares_no_taxonomy` | the module's CODE carries no `faction:` field and none of `JUSTICE`/`UNDERWORLD`/`HYBRID` (full-line comments stripped, so the header may document what was removed) |
| `career.upgrade_is_unlocked_not_forced` | the served advisory is OPTIONAL and states its own basis; the served unlock **AGREES with the served career table**; the panel paints the SERVED statement; and `promotePainted` equals the SERVED unlock count, so an `Upgrade` control can only exist for a career the level cap has actually opened |
| `career.staff_projection_names_its_basis` | the staff projection is an ARRAY (never null) that always names its BASIS, and "you employ nobody" is RENDERED rather than shown as an empty box |
| `career.staff_request_asks_and_cannot_grant` | the employer's door refuses `GET` **405** / anonymous **401** / a body naming a role grant **400** at the DECODER / a caller who employs nobody **403** with the reason, echoing the advisory |

Measured on 2026-09-15 i: **116 passed / 0 failed** (113 before these three were added). NOTE on a
false failure this probe can produce: `wallet-default` allows a **30-request burst** then 1 token/s, and the
probe's late `cardViewRt` block spends the same bucket the earlier blocks used, so a run on a drained bucket
can report `bonded.card_view_clears` / `sandbox.slide_wears_your_own_asset` as failed — a limiter reading,
NOT a product defect (the endpoints answer `200 {"cleared":true}` when called with 2 s spacing, and the
committed probe fails the same two assertions under the same conditions). Re-run on a rested bucket.

---



