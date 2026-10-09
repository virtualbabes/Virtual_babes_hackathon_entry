<DIRECTIVE>
# Session 2026-09-15 — ADMIN AUTHORITY · REWARD REGISTRY · ONE ADMIN CONSOLE
(PLAN v3 — Brendan approved in plan mode, executed under YOLO=true)

## Goal
Rebuild the admin surface as ONE reachable signature-gated surface, enforce env-only secrets +
$VBV/vault authority, add an app-wide reward-token registry, and record the tenant-world lease design.

## Authorization
- YOLO=true (Session-Handoff `[PERMIT_YOLO:TRUE]`)
- Directive: "rebuild the admin panel as one reachable signature-gated surface, enforce env-only
  secrets + $VBV/vault authority, add an app-wide (and tenant-visible) reward-token registry, and
  design the tenant-world lease record (spawning deferred)."

## Status
- ✅ SLICE 1 — ENV-ONLY SECRETS + VAULT/ASSET AUTHORITY. `network_registry_view.go` (NEW) is the one
  owner of the public projection; the git-tracked `networks.json` is now redacted before every write
  and `lobby_update` broadcasts `NetworkConfigPublic` (a secret's PRESENCE only). New admin-gated
  read-only `GET /api/admin/networks`. `handleUpdateRewardAsset` REFUSES (env-owned). False
  `ADMIN_KEY` / `X-Admin-Key` claims removed. `IPFS_API_KEY` threaded from the environment.
- ✅ SLICE 2 — REWARD-TOKEN REGISTRY. `reward_registry.go` (NEW) owns the descriptor, the
  reconciliation (prunes + REPORTS a template key with no enabled entry) and the served view;
  `amount_micro uint64` is the only amount field; the empty-asset-id hole is closed; the primary
  token is env-owned and read-only; `BASE_REWARD=5.0` no longer silently pays 0.
- ✅ SLICE 3 — DELETE 4, BUILD 1. `admin_panel.js` (hard-coded password), `admin_dashboard.js`,
  `operations_dashboard.js`, `final_dashboard.js` DELETED (+ imports, the empty overlay container and
  the purchasable "Admin Suite" entry). `Public/js/admin_console.js` (NEW) + `WD_ROUTES.admin` is the
  one way in (WD ▸ System & Ops ▸ Admin Console).

## Remaining Priority
- ✅ SLICE 3 tail — DONE (2026-09-15 b). All 24 Admin Suite controls converted to
  `data-admin-action` + `addEventListener` in ONE atomic change (0 inline handlers remain in the panel).
  NEW `Public/js/admin_suite.js` owns the wiring table and publishes the suite's globals, which FIXED
  EIGHT DEAD CONTROLS (`adminUpdatePowerScaling`, `adminBanWallet`, `adminAvatarBan`, `adminResetStats`,
  `adminUpdateDLCProduct`, `fetchDLCRegistry`, `fetchAdminLogs`, `renderShopTokenPresetEditor` +
  `saveShopTokenPresetsFromEditor`) that threw `ReferenceError` because nothing published them.
- ✅ SLICE 4 remainder — DONE. `handleUpdatePowerScaling` PERSISTS (its readers are club verification and
  artifact power, and the registry map is loaded at boot, so an unpersisted value died on restart),
  validates the divisor (finite, > 0) and the base (>= 0) BEFORE any write, and REFUSES an unknown focus
  network with 409 instead of answering `{"status":"success"}` while changing nothing.
- ✅ SLICE 5 — DONE (design only, as specified). The tenant world lease record + lifecycle vocabulary +
  THE blocker list live in `infrastructure_lease.go`; `CreateLease` refuses AT THE DOMAIN OWNER naming
  every blocker (no spawning, no world selector, no per-tenant token/network wiring, no gzip meter
  enforcement, no billing run) and ignores a client-declared rate; the catalogue serves
  `creation_available:false` + `creation_blocked_by`; the World Dashboard's infrastructure panel (which
  read `res.leases[].price_micro`, a shape the route has never served) now reads the served `systems[]`.
- ✅ SCSS AUDIT — DONE (2026-09-15 c). No `.od-*`/`.fd-*` rules exist anywhere and the only `.ad-*` rules
  (`.ad-card`, `.ad-status`, `.ads-*`) are the LIVE advertising surface's (kept). The dead rules were
  inside `_admin_panel.scss` (`.admin-panel-overlay`, `.admin-panel`, `.admin-panel-close`, `.admin-login*`,
  `.admin-error`, `.admin-content h2`, `.admin-feature-row*`, `.admin-status` — all matching NO element),
  now removed while `.admin-section*` (the kept panel) stays; `_ux_enhancements.scss` dropped its dead
  `.admin-panel-overlay` token. `sass` rc 0 and the compiled CSS verified.
- ✅ SYSTEMIC INLINE-HANDLER SWEEP — DONE (2026-09-15 c). 104 dead controls (98 unpublished + 6 naming a
  function that exists nowhere) fixed across 15 modules; NEW `Public/js`-wide gate
  `tools/server/verify_ui_handlers.js` (`npm run verify:handlers`) reports **0 dead, 0 boot errors**.
- `networks.json` still ships EMPTY `asset_id`/`app_id` for Voi Mainnet (settable in the panel; the value
  must be NAMED by Brendan, not guessed).
- DEFERRED (not this directive): world spawning, an in-app world selector, per-tenant token/network
  wiring, gzip meter enforcement, maintenance pay/revocation; GIT PUSH remains Brendan's.

## Constraints (binding)
- No cloud models. AI citizens own wallets. Integer/uint64 micro ledger math only (no float in the
  ledger; floats are display-only).
- Contained overlay mandate: `position:fixed; inset:0; overflow:hidden` shell; never nest an overlay
  root inside another.
- Secrets are ENV-ONLY: never served, never persisted to a git-tracked file.
- Verify before reporting; stop after 2 failed attempts.
</DIRECTIVE>

# SESSION DIRECTIVE RECORD — 2026-09-15 (d) · ORPHAN ARCHIVE RECONCILIATION (SUPERSEDES NOTHING)
*(Written by the executing agent. It opens NO new `<DIRECTIVE>` block on purpose: the ADMIN AUTHORITY
directive above remains the active one, and its one open item — `networks.json`'s empty Voi
`asset_id`/`app_id`, which only Brendan can NAME — is unchanged by this work.)*

## Goal
Reconcile the pre-refactor (2026-09-03/04) `AI-Brain/RAG/00–13` archive + `archive/docs-2026-09-07/orphan_fix_list.md`
against the LIVE repository, then act on the placement plan in the order the plan set: **Slice 0** (add the
missing gates; fix the two confirmed defects) and **Slice 1** (close the taxonomy gap) — BEFORE any
module retire verdict, because a retire decision made on the old metric would have been made on a false number.

## Why the gates came first
The archive scored a route by whether it is REFERENCED in client code (257 routes, 40 unreferenced; the live
equivalent is 312 / 284 / 28). The five panels written to close that gap (`misc_panel` 152 route literals,
`systems_panel` 30, `market_creator_panel` 25, `faith_extended` 8, `governance_extended` 6) are themselves
absent from the composition graph, so they contribute references while rendering nothing.
**Referenced is not reachable**, and nothing measured the difference — so Slice 0 built the measurement.

## Status
- ✅ SLICE 0a — THREE GATES ADDED. `verify:reachability` (`tools/server/verify_module_reachability.js`),
  `verify:routes` (`tools/server/verify_routing_tables.js`), `verify:duplicates`
  (`tools/server/verify_duplicate_modules.js`). Each fails closed, each carries a stated baseline/self-check,
  and each has a `--selftest` proving it can report a failure. Measured: **150 modules / 141 reachable /
  9 unreachable (all baselined with an owner decision) / 0 broken imports**; **40 routing references,
  729 JS + 68 Go/WASM publishers, 27 routes, 67 feature tabs, 0 dead names, 0 orphan routes**; **1 duplicate
  basename = 1 implementation + 1 forwarding alias**.
- ✅ SLICE 0b — TWO CONFIRMED DEFECTS FIXED. `constellation_hub.js` `panelMap.governance` named
  `'openGovernance'` (exists NOWHERE; the publisher is `window.openGovernancePanel`) so the hub's Governance
  node was a silent no-op; and `Public/js/collective-intelligence.js` was a second, EMPTY registry beside
  the real one — now a documented re-export alias, with its two consumers importing it in-directory.
- ✅ SLICE 1 — TAXONOMY GAP CLOSED. AI Citizens + Children Bots are **Assets** features now (routed, ↗),
  so `openAICitizens` is reachable and `openChildrenBots` no longer depends on the hub. `WD_ROUTES` 25 → 27,
  feature tabs 65 → 67. The §10.8.1 server-side tree list was extended (2 ids) — the omission was caught
  independently by the entry-probe paint assertion and by the Go drift test, which is why the list is
  covered by a test rather than by memory.
- ✅ VERIFIED. Native + `linux/amd64` + `js/wasm` rc 0; server rebuilt + restarted; the three gates PASS and
  their selftests PASS; `verify:handlers` 0 dead (both tiers); overlays 24/24 visible; **entry probe 89/0**
  (+4 assertions); `go test -run TestUiTreeArt` ok; live `/api/assets/ui-trees` 75 unique trees with the two
  new trees carrying light renditions.

## Remaining (needs Brendan — deliberately NOT acted on)
1. **Wire-or-retire verdicts for the 9 unreachable modules** (`_load_spectate_c.js`, `constellation_config`,
   `theme_engine`, `menu-dock`, and the five orphan-closure panels). No module was deleted: retirement needs
   the operator's consent. The baseline is the ledger of that decision and fails on any NEW unreachable module.
2. The 48 handler names inside the five uncomposed panels stay unreachable markup until (1) is decided.
3. Carried from the ADMIN directive: `networks.json`'s empty Voi `asset_id`/`app_id` (must be NAMED), the
   ARC-200 indexer base, and the deferred world-spawning/selector/per-tenant/gzip items.

## Constraints (binding, unchanged)
No cloud models · AI citizens own wallets · integer/uint64 micro ledger math only · contained overlay mandate ·
secrets ENV-ONLY · verify before reporting · **GIT PUSH remains Brendan's**.

# SESSION DIRECTIVE RECORD — 2026-09-15 (g) · BUILD WHAT IS NEEDED (SUPERSEDES THE (d) RECORD'S "REMAINING" 1–2)
*(Written by the executing agent. It opens NO new `<DIRECTIVE>` block: the ADMIN AUTHORITY directive above remains
the active one, and its one open item — `networks.json`'s empty Voi `asset_id`/`app_id`, which only Brendan can
NAME — is unchanged by this work.)*

## Authority
The operator's standing instruction, given directly: **"you are meant to build out the app, not look for reasons
to stop, build what is needed."** Applied to the two items the previous session left as *operator decisions* that
were in fact defects in this repository — and to the (d) record's "Remaining 1–2", which this record supersedes:

* **(d) Remaining 1 — "wire-or-retire verdicts for the 9 unreachable modules":** DELIVERED. `_load_spectate_c.js`
  was MOVED, `constellation_config`/`menu-dock` retired (2026-09-15 e), `theme_engine` WIRED, and the five
  orphan-closure panels RETIRED in this pass — **the reachability pen is EMPTY (143 modules / 143 reachable / 0
  baselined)**.
* **(d) Remaining 2 — "48 handler names stay unreachable markup":** RESOLVED with them; the handler gate now
  reports **DEAD 0** with 560 attributes (was 636) and only 8 boot-invisible NOTES (4 lazily published, 4 on
  another page).

## Executed
1. **The fifth self-deadlock** (`handleCyberIntercept` calling the self-locking `isJusticeAligned` while holding
   the write lock) → `isJusticeAlignedLocked`.
2. **The career role vocabulary** → ONE owner (`RoleKey` + two declared aliases + `roleXP`), which makes
   `GetRivalPairName`, `getTierFor`, `GetCareerTier`, `HasCareer` and `IsJusticeAligned` agree; `rivalPairTable`
   is a package var with test-pinned invariants; three unreachable rows deleted.
3. **A float in the career-XP path** → `GetVBVGatingPermille` owns the gate; the float getter is display-only.
4. **`/api/criminality/cyber-intercept`** → resolves the caller like every other door, refuses with JSON, and now
   has a reachable owner (`openCriminality` hub + `openCyberIntercept`/`submitCyberIntercept`).
5. **The five duplicate shells DELETED** on measurement (nothing names them; no capability exclusive to them).

## Remaining (genuinely needs Brendan; deliberately NOT acted on)
1. **`PromotedRoles` has no writer and `JobRole` is effectively unset** — so no player HOLDS a stored career.
   `RoleKey` makes the inline spellings agree; *what sets a career* is a schema decision.
2. **`networks.json` still ships EMPTY `asset_id`/`app_id` for Voi Mainnet**, and the ARC-200 indexer base is
   still unnamed. Both are VALUES: they must be NAMED, never guessed.
3. The courthouse rival bonus remains dead arithmetic (`Problems.md` §15 pass 2 E) — an XP-economics call.
4. Deferred by the ADMIN directive: world spawning, an in-app world selector, per-tenant token/network wiring,
   gzip meter enforcement, maintenance pay/revocation.
5. **GIT PUSH remains Brendan's.**

## Constraints (binding, unchanged)
No cloud models · AI citizens own wallets · integer/uint64 micro ledger math only · contained overlay mandate ·
secrets ENV-ONLY · verify before reporting · **GIT PUSH remains Brendan's**.

# SESSION DIRECTIVE RECORD — 2026-09-15 (h) · THE CAREER PATH, THE CIVIL RANK, PROMOTION AND DEMOTION
*(Written by the executing agent. It opens NO new `<DIRECTIVE>` block: the ADMIN AUTHORITY directive above remains the
active one. This record SUPERSEDES the (g) record's "Remaining" item 1, which asked the operator what sets a career:
he answered with the whole system below.)*

## Authority
The operator's specification, given directly:
> *"a user is meant to get a choice in the career when they open a region after owning 2 territories, these paths
> should consist of justice, criminal, neutral, they should be rivalled via multiple rival matrixes across the entire
> user experience, direct rivalry is justice/criminal, a neutral user will interpret rivals from other neutral users
> and the rival matrix dynamic behind other driving aspects; the promotion of careers is in accordance to level cap
> unlocks and when a user drains their level from cashing out it drains their career opportunities and will warn them
> to make the $VBV back to re-main employable in the career that they have promoted to or they will be demoted
> accordingly, you may need to find the rivalry matrix documentation across documents and consolidate it better;
> there is also a three tier 2nd career system that works like this, user[no shops/territories], manager[owns
> territories and may own regions], governor[must own a region], these three roles are career gates to force a worker
> society of lower tier lower level lower activity players to fill as high level careers will need active user
> management."*

## Executed
1. **THE CAREER PATH** — `justice` | `criminal` | `neutral`, chosen ONCE, gated on opening a region with ≥2
   territories, with a four-valued relation (`direct` / `interpreted` / `shared` / `none`). Direct = justice↔criminal;
   a neutral INTERPRETS (other neutrals, and the matrices behind the other driving aspects) and pays for its breadth
   (it may promote into EITHER side's careers) with no direct-rival leverage.
2. **MULTIPLE MATRICES, SERVED** — the path matrix is DERIVED from the career-pair matrix (attacker side of every
   enemy pair = justice, defender = criminal), and a test pins the agreement so they cannot drift. The three matrices
   (path / career / region) are served WITH THEIR OWNERS so a number is never shown without its layer.
3. **PROMOTION** — level cap + role tier + sustained $VBV + civil rank + path, each refusal naming the ONE missing
   piece with its exact shortfall. This is the WRITER `PromotedRoles` never had.
4. **DEMOTION** — `warn → grace (7 days) → demote`, plus `cleared` when the balance recovers. A demotion removes the
   career and records the exact integer arithmetic.
5. **THE THREE-TIER CIVIL RANK** — `user` / `manager` / `governor`, DERIVED from owned clubs and territories (never
   declared), used as the promotion gate that forces the worker society behind high careers.
6. **THE DOCUMENTATION CONSOLIDATED** — NEW `AI-Brain/Rivalry-Matrix.md` is the ONE owner; pointers from
   `Game-Mechanics-Index.md`, `RAG/03_theme_rivalry.md` and `Documentation.md`; `Manuals/RIVALRY-MANUAL.md` rewritten
   to v2.0 because v1.0 asserted careers, factions, a tier rung and FIVE endpoints that exist nowhere.
7. **FOUR DEFECTS FOUND BY MEASURING THE REQUEST** (all fixed; `Problems.md` §25): the $VBV-sustained gate could
   never be met by anyone (`VBVBalance` is assigned nowhere, and the sample scaled it by a float); a recovered player
   was failed by their own gate forever; every player's demotion was broadcast to every connected client; and nothing
   was ever demoted.

## Remaining (genuinely needs Brendan; deliberately NOT acted on)
1. **`networks.json` still ships EMPTY `asset_id`/`app_id` for Voi Mainnet**, and the ARC-200 indexer base is still
   unnamed. Both are VALUES: they must be NAMED, never guessed.
2. The courthouse rival bonus remains dead arithmetic (`Problems.md` §15 pass 2 E) — an XP-economics call.
3. **Recorded balance calls** (`Problems.md` §25 I): the float in `TrackRivalInteraction`; the unpriced P2-D10 enemy
   pair; the now-dead `Gossip↔ForensicAnalyst` +5 case.
4. A career's promotion is granted by the door — nothing calls it automatically on a level-up yet.
5. Deferred by the ADMIN directive: world spawning, an in-app world selector, per-tenant token/network wiring,
   gzip meter enforcement, maintenance pay/revocation.
6. **GIT PUSH remains Brendan's.**

## Constraints (binding, unchanged)
No cloud models · AI citizens own wallets · integer/uint64 micro ledger math only · contained overlay mandate ·
secrets ENV-ONLY · verify before reporting · **GIT PUSH remains Brendan's**.


# SESSION DIRECTIVE RECORD — 2026-09-15 (i) · "UNLOCKED, NEVER FORCED" (SUPERSEDES THE (h) RECORD'S "REMAINING" 4)
*(Written by the executing agent. It opens NO new `<DIRECTIVE>` block: the ADMIN AUTHORITY directive above
remains the active one, and its one open item — `networks.json`'s empty Voi `asset_id`/`app_id`, which only
Brendan can NAME — is unchanged by this work.)*

## Authority
The operator's specification, given directly — and it CORRECTS the previous session's own record, which had
booked the gap as a MISSING AUTOMATION:

> *"a user does not have to upgrade career it is only unlocked to upgrade if level cap allows it, a user may
> request staff users to upgrade when they notice there staff can upgrade and it is upto the user to upgrade
> them seflves, yes they may be notified not forced."*

The (h) record's Remaining item 4 read *"A career's promotion is granted by the door — nothing calls it
automatically on a level-up yet."* That framing was WRONG: there was no automation to add, and adding one would
have GRANTED what the rule says is only ever UNLOCKED. **This record supersedes it.**

## Executed
1. **THE OFFER MADE EXPLICIT.** `UnlockedCareerRolesLocked` (the level-cap unlock, derived from the ONE gate
   evaluator, granting nothing) and the served `upgrades` block — statement, `is_optional`, `unlock_basis`,
   `notice_rule`, the unlocked list, the staff projection with its `staff_basis`, and `requests_to_me`. It
   NEVER mutates.
2. **THE EMPLOYER MAY ASK.** `RequestStaffCareerUpgradeLocked` records ONE request on the EMPLOYER's own club
   (`Club.StaffUpgradeRequests`), notifies the ONE staff wallet, and **writes nothing on the employee's
   `PromotedRoles`/`JobRole`/level** — so an employer can ASK and only the employee can ACT.
3. **THE GATE IS REAL IN BOTH DIRECTIONS.** An employer cannot even ask for an upgrade the level cap has not
   opened; the door evaluates the EMPLOYEE's record and quotes their missing gate.
4. **NOTIFIED, NOT FORCED.** ONE notice per unlock, to the ONE wallet, from the 24 h liquidity daemon; the record
   is CLEARED on promotion, so a career lost to a demotion and re-earned re-notifies.
5. **"STAFF" WAS MEASURED BEFORE IT WAS NAMED** — `Club.Staff` / `PlayerStats.EmployerClubID` / `handleHirePlayer`.
6. **ONE DEFECT FOUND AND FIXED WHILE BUILDING IT:** a request could be STORED BUT INVISIBLE to the employee if
   their `EmployerClubID` had been cleared; the read now falls back to the request records themselves.

## Remaining (genuinely needs Brendan; deliberately NOT acted on)
1. **The courthouse rival bonus remains dead arithmetic** (`Problems.md` §15 pass 2 E) — an XP-economics call.
2. **Recorded balance calls:** the float in `TrackRivalInteraction`; the unpriced P2-D10 enemy pair; the now-dead
   `Gossip↔ForensicAnalyst` +5 case.
3. **`networks.json` still ships EMPTY `asset_id`/`app_id` for Voi Mainnet**, and the ARC-200 indexer base is
   still unnamed. Both are VALUES: they must be NAMED, never guessed.
4. **The notice cadence is the 24 h liquidity daemon** — one sampling window at most. A tighter, level-up-
   triggered notice is a deliberate non-change (one owner for the standing events).
5. **GIT PUSH remains Brendan's.**

## Constraints (binding, unchanged)
No cloud models · AI citizens own wallets · integer/uint64 micro ledger math only · contained overlay mandate ·
secrets ENV-ONLY · verify before reporting · **GIT PUSH remains Brendan's**.

---

# SESSION DIRECTIVE RECORD — 2026-09-15 (j) · THE RIVAL-XP ARITHMETIC (SUPERSEDES THE (i) RECORD'S "REMAINING" 1–2)
*(Written by the executing agent. It opens NO new `<DIRECTIVE>` block: the ADMIN AUTHORITY directive above remains
the active one, and its one open item — `networks.json`'s empty Voi `asset_id`/`app_id`, which only Brendan can
NAME — is unchanged by this work.)*

## Authority
The operator's directive, given directly:

> *"fix the courthouse↔criminal-system rival XP so it actually pays (the justice↔criminal path bonus was
> declared/served but awarded nowhere) and remove all float arithmetic from career XP — one integer owner,
> integer ledger."*

This SUPERSEDES the (i) record's Remaining items 1 and 2 (the courthouse dead arithmetic, and the float in
`TrackRivalInteraction`): both are now FIXED rather than recorded, and the same directive closed the unpriced
mechanism behind them.

## Executed
1. **THE PATH MATRIX PAYS.** `PathRivalryBonusBps` (+1000 bps, the ONE direct justice↔criminal rivalry) was
   computed, SERVED and pinned by a test while being awarded in ZERO places. It is now awarded at BOTH courthouse
   resolution points (`courthouseRivalAwardsLocked` → `payCourthouseRivalAwardsLocked`).
2. **ONE OWNER, STRUCTURAL.** `ResolveRivalXPAward` returns a STRUCT (`RivalXPAward{…, Bonus, Layers, Explain}`),
   so the wrong return value cannot be selected; `EvaluateCrossCareerXP` and `TrackRivalInteraction` delegate to it.
3. **FIVE DEAD HOOKS REPAIRED.** Six sites compared the DEFENDER's 30 % share against the base (`4 > 15`); the
   `uint64(rivalXP - base)` behind each would have underflowed to 18,446,744,073,709,551,605.
4. **NO FLOAT ON XP.** Permille integers throughout; the float getter is a display wrapper pinned to the owner, and
   `black_market_service.go` + `handlers_rumor.go` were de-floated with it.
5. **A SEVENTH SELF-LOCK FIXED** (`handleKidnapRequest` → `trackCareerXPLocked`, 17 sites), and **A GATE NOW
   MEASURES THE CLASS** (`career_award_lock_test.go`, AST-based, with its own negative control; 94 files,
   0 violations).
6. **A STALE TEST FIXED** — `courthouse_rival_scan_test.go` named two functions the new owner deleted, so the
   package did not compile at all.

## Remaining (genuinely needs Brendan; deliberately NOT acted on)
1. **`MutationLogAuditor↔Kidnapper` is declared antagonistic but PRICED 0** — an XP-economics call.
2. **`Gossip↔ForensicAnalyst` (+5) is dead in `GetRivalXPDelta`** — removing the NAME rather than the PRICE is a call.
3. **`networks.json` still ships EMPTY `asset_id`/`app_id` for Voi Mainnet**, and the ARC-200 indexer base is still
   unnamed. Both are VALUES: they must be NAMED, never guessed.
4. Deferred by the ADMIN directive: world spawning, an in-app world selector, per-tenant token/network wiring,
   gzip meter enforcement, maintenance pay/revocation.
5. **GIT PUSH remains Brendan's.**

## Constraints (binding, unchanged)
No cloud models · AI citizens own wallets · integer/uint64 micro ledger math only · contained overlay mandate ·
secrets ENV-ONLY · verify before reporting · **GIT PUSH remains Brendan's**.


# SESSION DIRECTIVE RECORD — 2026-09-15 (k) · THE SELF-LOCK GATE GENERALISED · 20 REAL DEADLOCKS
*(Written by the executing agent. It opens NO new `<DIRECTIVE>` block on purpose: the ADMIN AUTHORITY
directive above remains the active one, and its one open item — `networks.json`'s empty Voi
`asset_id`/`app_id`, which only Brendan can NAME — is unchanged by this work.)*

## Authority
The operator's standing instruction, given directly: **"you are meant to build out the app, not look for
reasons to stop, build what is needed."** Applied to the one follow-up this repository had recorded against
ITSELF — session (n)'s *"Nothing enforces the 'never self-lock' rule — the 4th instance was found by reading,
not by a linter. A repo-wide audit of `...Locked` functions for self-locking callees is the obvious
follow-up"* — and to the (j) record's gate, which had closed only ONE shape of the class (a locked caller
reaching a self-locking CAREER-AWARD helper). **This record SUPERSEDES that follow-up: it is DONE.**

## Executed
1. **THE GATE WAS GENERALISED TO THE WHOLE CLASS.** NEW `selflock_gate_test.go`: the registry is DERIVED from
   the tree (every `*Lobby` method that takes `l.mutex` itself — **197 of 551**), and a call to one of them is
   reported wherever the lock is DEFINITELY held. Measured: **20 real violations in NINE functions**, every one
   the unconditional self-deadlock on a non-re-entrant `sync.RWMutex` that freezes the whole process — not
   merely the request that happened to be running.
2. **FOUR WERE LIVE PLAYER PATHS THAT COULD NEVER HAVE COMPLETED:** selling a card to the Black Market (all
   four of its refusals), restocking a club shop (all four refusals), creating a lease, and forming a regional
   alliance. Three more were the **mutation ladder** — the scar path is random, so it fired ~25 % of the time
   and never in a scripted test.
3. **TWO `...Locked` SIBLINGS WERE CREATED** — `broadcastToAdminsLocked` and `isWalletRegisteredLocked` —
   because the gate's own recommendation ("call the `...Locked` sibling") was otherwise unavailable. They
   follow the ONE-OWNER PAIR this tree already uses: the self-locking form takes the lock and DELEGATES, so
   behaviour at every call site is identical minus the lock acquisition. `isWalletRegistered` now has no
   production caller; that is the precedent `isJusticeAligned` set, and the gate PINS the pair, so deleting
   the locking form fails the build instead of silently widening the hole.
4. **THE DETECTOR'S OWN TWO FALSE-POSITIVE CLASSES WERE FIXED RATHER THAN WORKED AROUND**, because a gate that
   reports correct code gets switched off. A flat source-order walk let ONE `switch` case's `Lock()` poison
   every LATER case (**40 of 44 reports** were that single artefact). And `go l.sendNoteTx(…)` was reported
   although **a `go` STATEMENT STARTS A NEW GOROUTINE WHICH DOES NOT INHERIT THE LOCK** — the callee cannot
   deadlock, it only waits until the parent releases. The detector now skips the CALLEE and still walks its
   ARGUMENTS; both halves are pinned by a control. **The CODE was not changed where the GATE was wrong.**
5. **A BEHAVIOURAL HALF WAS ADDED.** `TestTheRecommendedLockedFormsRunWhileTheWriteLockIsHeld` proves the two
   siblings this session CREATED actually run under the WRITE lock, failing by TIMEOUT — because a self-lock
   raises no error at all; it simply never returns.
6. **THE GATE'S LIMITS ARE STATED IN IT, NOT IMPLIED:** intra-procedural (a call graph is the next step if
   that shape ever appears); function literals and a `go` callee are their own scopes; a `defer`red call is
   reported at its `defer` statement; `_test.go` excluded; an untyped identifier is skipped rather than
   guessed; a read-held → read-wanted call is a NOTE, not a failure (3 remain).

## Remaining (genuinely needs Brendan; deliberately NOT acted on)
1. **`MutationLogAuditor↔Kidnapper` is declared antagonistic but PRICED 0** — an XP-economics call.
2. **`Gossip↔ForensicAnalyst` (+5) is dead in `GetRivalXPDelta`** — removing the NAME rather than the PRICE is
   a call.
3. **`networks.json` still ships EMPTY `asset_id`/`app_id` for Voi Mainnet**, and the ARC-200 indexer base is
   still unnamed. Both are VALUES: they must be NAMED, never guessed.
4. Deferred by the ADMIN directive: world spawning, an in-app world selector, per-tenant token/network
   wiring, gzip meter enforcement, maintenance pay/revocation.
5. **GIT PUSH remains Brendan's.**

## Constraints (binding, unchanged)
No cloud models · AI citizens own wallets · integer/uint64 micro ledger math only · contained overlay mandate ·
secrets ENV-ONLY · verify before reporting · **GIT PUSH remains Brendan's**.


---

# SESSION DIRECTIVE RECORD — 2026-09-15 (l) · THE TRANSITIVE SELF-LOCK · MATCHMAKING UNFROZEN
*(Written by the executing agent. It opens NO new `<DIRECTIVE>` block on purpose: the ADMIN AUTHORITY
directive above remains the active one, and its one open item — `networks.json`'s empty Voi
`asset_id`/`app_id`, which only Brendan can NAME — is unchanged by this work.)*

## Authority
The (k) record's own follow-up, in its own words: *"THE ANALYSIS IS INTRA-PROCEDURAL … A call graph is the next
step if that shape ever appears."* The operator's standing instruction — **"you are meant to build out the app,
not look for reasons to stop, build what is needed"** — applied to it in the order the record set: MEASURE first,
build only if the shape exists.

## Executed
1. **MEASURED BEFORE BUILDING.** A standalone stdlib-only probe (temp dir, no repo pollution) reimplemented the
   committed gate's MUST-held walk verbatim and propagated the held lock across call edges to a fixpoint. It
   **reproduced the gate exactly — 197 self-locking helpers and the same 3 read+read notes** — which is what makes
   its findings trustworthy. Its controls (3 must-report / 3 must-not) caught one of MY OWN classification
   defects before any finding was believed.
2. **THE SHAPE EXISTS: `processMatchmaking` → `initiatePairedMatch` → `sendToClient`.** `processMatchmaking`
   (lobby_manager.go:2384) holds the WRITE lock for its whole body and pairs at 2424/2480/2531;
   `initiatePairedMatch` is a lock-EXPECTED helper whose name does not say so, and at 2746/2747 it called the
   SELF-LOCKING `sendToClient` (`RLock`) — one goroutine taking a non-re-entrant `RWMutex` twice, freezing every
   request and every WebSocket. It fired on **every successful pairing** (standard, bounty, tournament), so
   matchmaking had been permanently broken.
3. **FIXED** with `sendToClientLocked` (the `...Locked` sibling) plus a doc comment naming the contract, because a
   name that does not say "the lock is held" is what hid it from seven earlier rounds of reading.
4. **THE CLASS IS NOW MEASURED, NOT READ.** NEW `TestNoTransitiveSelfLockingLobbyHelper` (1440 bodies, 6264 edges
   propagated, 217 bodies enterable under the lock, **0 findings**) reporting each finding with its `...Locked`
   sibling AND its WITNESS edge; NEW `TestTransitiveSelfLockDetectsAViolation` (3 must-report + 5 must-not, incl.
   an AMBIGUOUS held-service name that is skipped rather than guessed); NEW
   `TestTheMatchmakingPairingPathCompletesUnderTheWriteLock` (fails by TIMEOUT).

## Remaining (genuinely needs Brendan; deliberately NOT acted on)
1. The transitive pass is per-package and name-based: an ambiguous held-service name is SKIPPED, never guessed, and
   paths are JOINED so a finding names its witness rather than claiming a proof.
2. `MutationLogAuditor↔Kidnapper` is declared antagonistic but PRICED 0 — an XP-economics call.
3. `Gossip↔ForensicAnalyst` (+5) is dead in `GetRivalXPDelta` — removing the NAME rather than the PRICE is a call.
4. **`networks.json` still ships EMPTY `asset_id`/`app_id` for Voi Mainnet**, and the ARC-200 indexer base is still
   unnamed. Both are VALUES: they must be NAMED, never guessed.
5. Deferred by the ADMIN directive: world spawning, an in-app world selector, per-tenant token/network wiring, gzip
   meter enforcement, maintenance pay/revocation.
6. **GIT PUSH remains Brendan's.**

## Constraints (binding, unchanged)
No cloud models · AI citizens own wallets · integer/uint64 micro ledger math only · contained overlay mandate ·
secrets ENV-ONLY · verify before reporting · **GIT PUSH remains Brendan's**.


---

# SESSION DIRECTIVE RECORD — 2026-09-15 (m) · THE RECURSIVE READ LOCK · THREE "NOTES" THAT WERE LIVE DEFECTS
*(Written by the executing agent. It opens NO new `<DIRECTIVE>` block on purpose: the ADMIN AUTHORITY
directive above remains the active one, and its one open item — `networks.json`'s empty Voi
`asset_id`/`app_id`, which only Brendan can NAME — is unchanged by this work.)*

## Authority
The previous record's own honest limit, in its own words: *"A read lock held while the callee takes a read lock
is reported as a NOTE, not a failure: Go discourages it, but it deadlocks only if a writer is already waiting."*
The operator's standing instruction — **"you are meant to build out the app, not look for reasons to stop, build
what is needed"** — applied to asking that question again instead of accepting the note.

## Executed
1. **THE THREE "NOTES" WERE LIVE.** All three held the lobby lock and called a helper that takes it again:
   `computeRegionThemeCoherence` → `ComputeThemeVector` once per RESIDENT PLAYER;
   `RecomputeAllSignatures` → `ComputeAssetSignature` once per CLUB (reachable from
   `POST /api/rivalry/recompute`); and `HandleGetCareerProgress` → the self-locking `GetCareerProgress`
   **while WRITING `l.leaderboard` in the same block** — a map of VALUES, so the lazy `CareerXP`
   initialisation was a map write under a READ lock (a concurrent map write against every reader: `fatal
   error`, not recoverable). One door, two defects.
2. **WHY THE NOTE WAS WRONG.** The write lock is taken by matchmaking, club, market, treasury, admin and
   economy paths CONSTANTLY, so "a writer is already waiting" is the normal state under load rather than a
   corner case; and once one queues, the recursive `RLock` never returns because the first read lock is still
   held by the same goroutine. The gate did not miss these — it SAW them and called them harmless.
3. **FIXED AS READ-VALIDATE → RELEASE → COMPUTE.** Both loops snapshot under the read lock and derive every
   value outside it; the career door takes the WRITE lock once, builds the payload with the newly extracted
   ONE builder `careerProgressLocked`, and releases before it encodes (`GetCareerProgress` is lock + delegate).
4. **THE GATE NOW FAILS ON THE SHAPE.** `deadlocksUnder` returns `held != heldNone` (any held lock against a
   callee that takes it is fatal) and the new `reasonAt(held)` states the mechanism at the call site; the
   `notes` bucket is deleted from both passes, because an always-empty field is the false statement §14
   recorded for `derivative_status.reused`.
5. **THE CONTROL CAUGHT THE HALF-WIRED EDIT:** `TestSelfLockGateDetectsAViolation` failed *"expected exactly 8
   violations … got 7"* when the message was updated but `deadlocksUnder` was not; it now carries
   `badRecursiveRead` intra, and the transitive control carries the same shape across an EDGE. NEGATIVE CONTROL
   ON THE REAL TREE: re-introducing the recursive read makes the gate FAIL naming
   `computeThemeVectorLocked()`; removing it passes.

## Verified
- `TestNoSelfLockingLobbyHelper` **0 violations** (was 3 notes); transitive pass **1441 bodies / 6245 edges /
  215 enterable / 0 findings / 0 intra-procedural / 221 correct calls** — both passes agree exactly.
- All 8 self-lock tests PASS; `go test .` green except the PRE-EXISTING AMM test; native + `linux/amd64` +
  `js/wasm` rc 0; `Public/main.wasm` PROVED untouched (all four changed files `!js && !wasm`, consecutive
  rebuilds byte-identical, the new symbols ABSENT while `sync.RWMutex`/`GetGameState` are present);
  `selflock_gate_test.go` gofmt-clean, and the 37 `lobby_manager.go` hunks are provably PRE-EXISTING (HEAD has
  the same 37, none covering the edit).
- **Live** (server rebuilt + restarted on :8090): `/api/faucet/status` 200, `GET /api/career/progress` **200**,
  `GET /api/rivalry/world-dynamics?region=Base` **200**, `POST /api/rivalry/recompute` **200 {"count":1}**.
- Gates: `verify:duplicates` PASS, `verify:routes` PASS (0 dead names), `verify:reachability` PASS (143/143).

## Remaining (genuinely needs Brendan; deliberately NOT acted on)
1. **`MutationLogAuditor↔Kidnapper` is declared antagonistic but PRICED 0** — an XP-economics call.
2. **`Gossip↔ForensicAnalyst` (+5) is dead in `GetRivalXPDelta`** — removing the NAME rather than the PRICE is a call.
3. **`networks.json` still ships EMPTY `asset_id`/`app_id` for Voi Mainnet**, and the ARC-200 indexer base is
   still unnamed. Both are VALUES: they must be NAMED, never guessed.
4. Deferred by the ADMIN directive: world spawning, an in-app world selector, per-tenant token/network wiring,
   gzip meter enforcement, maintenance pay/revocation.
5. **GIT PUSH remains Brendan's.**

## Constraints (binding, unchanged)
No cloud models · AI citizens own wallets · integer/uint64 micro ledger math only · contained overlay mandate ·
secrets ENV-ONLY · verify before reporting · **GIT PUSH remains Brendan's**.

---

# SESSION DIRECTIVE RECORD — 2026-09-15 (n) · THE UNFENCED LOBBY MAP
*(Written by the executing agent. It opens NO new `<DIRECTIVE>` block on purpose: the ADMIN AUTHORITY
directive above remains the active one, and its one open item — `networks.json`'s empty Voi
`asset_id`/`app_id`, which only Brendan can NAME — is unchanged by this work.)*

## Authority
The (m) record's own Next item 2, in its own words: *"Audit the **other** lock-discipline class the same
way: unfenced map reads/writes on lobby-owned state (`CalculateTotalPortfolioValue` and any sibling),
using this session's proven method — synthetic negative controls first, then measure the real tree
before changing code."* The operator's standing instruction — **"you are meant to build out the app, not
look for reasons to stop, build what is needed"** — applied to it in the order the record set: MEASURE
first, build only what the measurement demands.


---

# SESSION DIRECTIVE RECORD — 2026-09-16 (o) · THE SAME-MUTEX RE-LOCK · TWO DOORS THAT FROZE THE PROCESS
*(Written by the executing agent. It opens NO new `<DIRECTIVE>` block: the ADMIN AUTHORITY directive above
remains the active one, and its one open item — `networks.json`'s empty Voi `asset_id`/`app_id`, which only
Brendan can NAME — is unchanged by this work.)*

## Authority
The previous record's own Next item 1 — *"build the `...Locked` pair for `CalculateTotalPortfolioValue` …
that is a design change, not a line"* — executed in the order it set: **MEASURE first**, then build only
what the measurement demands. The operator's standing instruction (*"you are meant to build out the app,
not look for reasons to stop, build what is needed"*) is what says a measured defect gets FIXED rather than
recorded.

## Executed
1. **MEASURED FIRST, AND THE MEASUREMENT PAID FOR ITSELF.** The task was one `...Locked` pair; the same
   file held **two unconditional, process-freezing self-deadlocks on `EntityMarketNode.Mu`** —
   `handleInvestEntity` (WS) `Lock()`+`defer Unlock()` then `RLock()` at 219, and `HandleDirectInvest`
   (`POST /api/invest/entity`) the same at 541. `git blame` dates both to **2026-07-16** (`6e68e19c`), so
   this is PRE-EXISTING, not a regression: **every successful direct investment, from either surface,
   froze the whole process** (the lobby write lock is held across the body → every request, every WebSocket).
2. **THE LOCK ORDER WAS MEASURED BEFORE A LOCK WAS ADDED.** `RouteCriminalTax` takes `tsr.Mu`
   (`economy_processing.go:247`) and both doors call it **while holding a node**, so `node.Mu → router.Mu`
   already exists. The obvious "fence the map read" patch (holding `router.Mu` across the node lock) would
   have created the reverse edge and a genuine two-goroutine deadlock. The router mutex is therefore held
   **for the lookup and nothing else** — the same shape §31's `HandleDividendHistory` fix already uses.
3. **THE PAIR BUILT (the stated task).** `CalculateTotalPortfolioValue` is now the DOOR (takes
   `l.mutex.RLock()`, releases, delegates) and **`CalculateTotalPortfolioValueLocked`** is the lock-held
   form, its contract naming BOTH halves: the lobby lock MUST be held, and **no `EntityMarketNode.Mu` may
   be held**. Three lock-holding callers converted (`handleInvestEntity:133`, `HandleDirectInvest:491`,
   `computeThemeVectorLocked:151`); `OutcomeBias` keeps the door, consistent with the read lock it already
   took itself. §31's `UNFENCED` baseline entry was DELETED with a note, and the map gate now reports it
   fenced.
4. **THE PORTFOLIO IS PRICED OUTSIDE THE NODE SECTION, in both doors** — the anti-concentration cap is
   computed before `node.Mu.Lock()` and consumed where its refusal already lived, so the ORDER OF REFUSALS
   is unchanged and no behaviour moved except the deadlock. The redundant re-read of
   `CumulativeYieldPerShare`/`ReserveBalance` is deleted, with the reason stated in place.
5. **A WRITE UNDER A READ LOCK, FIXED ADJACENTLY:** `ProcessHourlyEntityRevenueDistribution` (a live
   daemon) ran `node.DividendPoolMicro += …` under `node.Mu.RLock()` — two ticks, or a tick and an AMM
   buy, SILENTLY LOSE one injection. It now takes the write lock.
6. **A NEW GATE, AND ITS OWN CONTROL CAUGHT THE DETECTOR'S BLINDNESS.** `mutex_relock_gate_test.go`
   DERIVES its subject (every `sync.Mutex`/`sync.RWMutex` struct field — **8 names from 371 struct
   declarations**), is MUST-HELD (branches/loops joined), treats a **deferred release as NOT releasing at
   its source line** (the rule that makes this tree's dominant `Lock(); defer Unlock(); …` idiom visible),
   and analyses a `go` literal from an EMPTY held-set (a goroutine does not inherit the lock). The first
   version matched the METHOD name instead of the RECEIVER and reported **0 re-locks for every input**;
   the control failed with **0 findings across all ten shapes**, which is why a clean report means nothing
   without it. **Teeth on the real tree:** an injected re-lock fails the gate naming it.
7. **A BEHAVIOURAL PIN, because a static gate cannot prove a door RUNS.** `entity_investment_lock_test.go`
   drives both doors under a **watchdog** that fails by TIMEOUT and dumps every stack (a deadlock raises no
   error at all), seeded with **shares in the target entity** and asserting that seed so the test cannot
   stop testing the shape it was written for. Re-introducing the nested `RLock` failed it with
   **`goroutine 36 [sync.RWMutex.RLock]` blocked at `entity_investment_service.go:228`** — runtime-state
   proof of the mechanism — then the control was reverted.

## Verified
- `go test .` green except the **PRE-EXISTING** `TestCalculateBuyCost_WhaleSlippagePenalty`; that single
  run also proves the map gate, the self-lock gate and every prior watchdog still pass.
- native + `linux/amd64` + `js/wasm` **rc 0**; **all four changed/new files are `//go:build !js && !wasm`**,
  so `Public/main.wasm` is untouched by construction — and unchanged on disk (11,371,182 B, same timestamp;
  no `-o` was passed to either cross build).
- `verify:duplicates`, `verify:routes` (0 dead names) and `verify:reachability` (143/143) PASS. No client JS
  changed this pass.
- **LIVE on :8090** (rebuilt, restarted, PID 8672, boot clean): `faucet=200`;
  `GET /api/invest/portfolio?wallet=0xselftest` → **200 in 10 ms**; `GET /api/invest/dividends/history` →
  **200 `{"history":[],"wallet":"0xselftest"}`**; `POST /api/invest/entity` → **404 in 81 ms** with the
  server still answering afterwards.

## Remaining (genuinely needs Brendan; deliberately NOT acted on)
1. **The re-lock gate is INTRA-PROCEDURAL** — a re-lock across a CALL EDGE is invisible to it, which is why
   `CalculateTotalPortfolioValueLocked`'s "no node lock held" half is enforced by a comment, a test and the
   hoist rather than by the gate. Interprocedural analysis for non-`*Lobby` types is the next real step.
2. **`ProcessEntityRevenueDistribution` is DORMANT and mutates a node under `router.Mu` with no `node.Mu`.**
   Recorded, not half-fixed: taking the node lock inside that window would CREATE the inversion (§Executed 2).
3. **The live probe could not exercise the fixed SUCCESS path** (a 404 refuses before the node lock; a live
   success needs a funded wallet holding shares in a seeded market node). The success-path proof is the
   watchdog's runtime stack dump.
4. Carried: the unpriced `MutationLogAuditor↔Kidnapper`; the dead `Gossip↔ForensicAnalyst` +5;
   `networks.json`'s empty Voi `asset_id`/`app_id`; the unnamed ARC-200 indexer base.
5. **GIT PUSH remains Brendan's.**

## Constraints (binding, unchanged)
No cloud models · AI citizens own wallets · integer/uint64 micro ledger math only · contained overlay mandate ·
secrets ENV-ONLY · verify before reporting · **GIT PUSH remains Brendan's**.


## Executed
1. **MEASURED BEFORE BUILDING.** A standalone stdlib-only probe (temp dir, no repo pollution) asked the
   one question that needs no control-flow modelling to be true — *which lobby-owning functions touch a
   lobby-owned map while taking NO lobby lock of their own?* It found **41 functions**, and triaging the
   ROUTE TABLE found **FIVE LIVE DOORS**: `/api/player/profile`, `/api/invest/portfolio`,
   `/api/invest/dividends/history`, `/api/contracts/list` and `/api/contracts/assign`. Two of them WROTE.
2. **WHY THIS CLASS IS THE WORSE ONE.** `l.leaderboard` is `map[string]PlayerStats` (**VALUES**), so even
   a lazy nil-initialisation is a map write; and Go's runtime answers a concurrent access with
   `fatal error: concurrent map read and map write` / `concurrent map iteration and map write`, which is
   **NOT recoverable** — not an error value, not catchable by `recover`, and it kills every connection,
   where a self-lock at least leaves the process alive and diagnosable.
3. **FIXED AFTER CHECKING EVERY CALLER FOR A LOCK ALREADY HELD** — a blind lock would have re-created
   §29/§30's self-lock, the class this repository has now closed twenty-nine times. Two shapes:
   READ-VALIDATE → RELEASE → COMPUTE for reads; **ONE CRITICAL SECTION** for the single
   read-modify-write (`HandleAssignContract` re-reads under the WRITE lock, because the stale copy both
   races AND discards any update that landed in between). `GetPortfolioForPlayer` takes the **WRITE**
   lock because it lazily writes; `HandleDividendHistory` takes `tokenSinkRouter.Mu` for `MarketNodes`,
   its real owner. One adjacency fix: `history` served `null` where the convention is `[]`.
4. **A GATE THAT DERIVES ITS OWN SUBJECT.** NEW `map_race_gate_test.go` takes **every map-typed field of
   `type Lobby struct`**, parsed from the struct — not a hand-written list. The derived set is **47**
   fields against the probe's 18, and it immediately found **6 entries the first census could not see**:
   **the gate found its own author's blind spot.** It fails CLOSED on a NEW entry, a STALE entry (a fix
   must delete its own line) and a FLIPPED entry (a `CONTRACT` helper that gained a caller holding
   nothing). Controls: 7 shapes that MUST report, 4 that MUST NOT, a clean-file case, a loud failure if
   the derivation ever finds zero maps, and a **REAL-TREE negative control** proving it fires.
5. **TWO OF THE GATE'S OWN VERDICTS WERE WRONG AND THE CONTROL CAUGHT BOTH.** The first control draft
   expected the CHILD of a chain to be reported — it touches no map, so only the parent is an entry and
   the child's name travels in the parent's EVIDENCE. And the one-hop CONTRACT verdict **over-claimed
   risk** on four entries whose caller chains DO hold the lock; a verdict asserting "a lock is owed" when
   none is owed is the false-statement family of §14's `derivative_status.reused`, so `entryHolds` now
   propagates to a **FIXPOINT** over the caller graph.

## Verified
- The probe re-run after the fixes: **41 silent → 35**, exactly the six fixed — an instrument independent
  of the gate agreeing with it.
- All 9 self-lock tests PASS; the helper count moved **197 → 199** (exactly the two new `*Lobby` lock
  sites) with the transitive pass at **0 findings**, which is what proves the new locks do not sit under
  an existing one.
- `TestNoNewUnfencedLobbyMapAccess` → **47 maps derived; 43 entries; 43 baselined** (29 UNFENCED, 14
  CONTRACT). `go test .` green except the **PRE-EXISTING** AMM test. native + `linux/amd64` + `js/wasm`
  rc 0, all four changed files `//go:build !js && !wasm`.
- **LIVE** on :8090 (rebuilt, restarted, PID 31804): profile/portfolio/contracts **200**,
  `dividends` → **`{"history":[]}`** (`[]` proves the NEW binary is answering), `assign` → **409** with a
  domain reason through the new write lock.

## Remaining (genuinely needs Brendan; deliberately NOT acted on)
1. **29 baseline entries remain UNFENCED**, each naming a lock still owed: dead-or-uncalled
   (`HandleCompleteContract` — DEAD despite its doc claiming three callers), exported doors whose callers
   are unconstrained (`CalculateReputation`, `CalculateTotalPortfolioValue`), and bodies reached from
   callers that hold nothing today.
2. **`CalculateTotalPortfolioValue` is deliberately left.** Three of its four callers hold the lock and
   the fourth (`OutcomeBias`) is dormant, so locking it requires building a `...Locked` **pair** and
   converting two call sites in the same change — a design change, not a line.
3. **The self-lock gate cannot see a self-locking method on a NON-`*Lobby` type** (`deriveHelpers` filters
   to receiver `Lobby`), so `ContractEngine.EvaluateRivalPresence` — which this pass gave an `RLock` — is
   not counted (199, not 200). Its callers were checked by hand. Type-aware derivation is the next real
   improvement to that gate, and a measured instance now justifies it.
4. `-race` is UNAVAILABLE on this host (needs cgo, no C toolchain), so the new gate is a STATIC census and
   must never be described as a race proof.
5. Carried, needing the OPERATOR: the unpriced `MutationLogAuditor↔Kidnapper`; the dead
   `Gossip↔ForensicAnalyst` +5; `networks.json`'s empty Voi `asset_id`/`app_id`; the unnamed ARC-200 base.
6. **GIT PUSH remains Brendan's.**

## Constraints (binding, unchanged)
No cloud models · AI citizens own wallets · integer/uint64 micro ledger math only · contained overlay mandate ·
secrets ENV-ONLY · verify before reporting · **GIT PUSH remains Brendan's**.


