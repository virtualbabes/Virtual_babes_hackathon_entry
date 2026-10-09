# 17 — ASPECTS FLOW (the whole application, aspect by aspect)

> **Owner:** the ONE document that answers, for any aspect of this application: *which files own it · what routes serve it · what state it holds · how a request flows through it · what is broken.*
> **Relationship to the other RAGs:** 14/15/16/18 are **per-file**. This file is **per-aspect** and is **DERIVED** from them — it is not written from memory and it is not authored ahead of the evidence.
> **Coverage proof:** `19_coverage_ledger.md`.
> **POSITION (2026-09-20) — THE DERIVATION IS OPENED, AND ITS INPUTS ARE ALL MEASURED.** The four per-file flow owners are **complete** (Go 143 read + 11 `partial` · JS 169 · CSS 110 · misc 73 = 506 rows). **READ ORDER:** the *sources* and the *spine* were appended AFTER the rows were written (the rows came first, from entries in hand), so read `## MEASURED ASPECT SOURCES` and `## THE ASPECT SPINE` before the rows — they are the inputs the rows rest on. This file now carries: the **three admissible aspect sources, each confirmed to exist and counted** (the route table · the 11 `WD_CATEGORIES` · `App-Aspect-Index.md`'s 39 aspects) · the **machine-derived ASPECT SPINE** (326 `server_main.go` routes → `handle*` → the non-test file that defines it, **88 route prefixes**, with unresolved handler names COUNTED not guessed) · **35 derived rows**, each written only because this session read the owning entries · and the **bounded remainder** (**EMPTY — all 39 aspects covered, held empty by the verify:aspects gate**). **A fourth candidate source was measured and REJECTED:** keyword-matching the prose `OWNS` field yields false owners (`identity` → 59 refs, `frontend/nav` → 101), so ownership must be read per entry.

## THE DERIVATION RULES (binding, so this file cannot become an opinion)

An aspect row is only written when all five inputs exist:

| Input | Measured from |
|---|---|
| **OWNER FILES** | the `OWNS` field of entries in `14_flow_go.md` / `15_flow_js.md` / `16_flow_css.md` / `18_mic_flow.md` |
| **ROUTES** | `server_main.go` + `console_server.go` route registrations, and the `WD_ROUTES` map in `world_dashboard.js` |
| **STATE** | the `STATE & LOCKS` field of the owning Go entries |
| **FLOWS** | the `CALLS OUT` / `CALLED BY` fields, walked to a short chain |
| **GAPS** | the `OBSERVED DEFECTS` fields, plus any capability with **no owner** and any owner with **no caller** |

**The aspect LIST itself is measured, never invented.** Its three admissible sources: the route table, the World Dashboard taxonomy (`WD_CATEGORIES`), and the repository's own aspect index — to be **read and confirmed to exist** before it is used. A category name that appears in a document but owns no file is recorded as a GAP, not as an aspect.

## ASPECT ROW SCHEMA

```
### <aspect> - <one-line statement of what it is>
- **OWNER FILES:**     path:lines for each (from the per-file entries)
- **ROUTES:**          method + path, with the file:line of the registration
- **STATE:**           the maps/registries and their mutexes
- **FLOW:**            <entry point> -> <owner> -> <state> -> <response/chain>
- **UI:**              the surfaces that reach it, and how (WD route/embed/overlay)
- **GAPS:**            defects, owners with no caller, capabilities with no owner, each with a cite
- **STATUS:**          complete | partial | design-only | broken | absent
```

## ASPECTS — DERIVED ROWS (each names the evidence it rests on)

> **Rule applied:** a row is written only when this session READ the owning entries. A row that would rest on a keyword match over prose is NOT written — it is listed in the remainder below instead. Every cite is to a file that was read and whose reading is recorded with ranges in `19_coverage_ledger.md`.

### Records & chain transport — the save/restore rail for the whole civilization (§35 Economy Bootstrap & State Recovery)
- **OWNER FILES:** `record_families.go` · `record_envelope.go` · `record_batch.go` · `record_batch_read.go` · `checkpoint_indexer_read.go` · `economy_service.go` (the cadence) · `tenant_vault.go` + `tenant_vault_contract.go` (Stage A)
- **ROUTES:** `/api/notes/vocabulary` (read-only, `handlers_public.go`). The record rail itself is **not routed** — it is a daemon plus a boot path.
- **STATE:** `PrimaryRecordFamilies` (**22 declared / 0 pending**) · per-family nonce SEQUENCES · `Lobby.playerBalances`/`leaderboard` copied under `l.mutex` for the cadence.
- **FLOW:** boot `Load` → `readRecordFamilyWithSource` → family record vs batch, chosen **BY NONCE, never by clock** → `assembleRecords` names every incomplete nonce as a GAP → restore. Write: cadence → copy under the owner lock → `saveRecordBatchLocked` → chunked under the **1024-byte AVM note cap** → one dispatch.
- **UI:** none — no surface reads records; the wizard's seed step is Stage C and is not built.
- **GAPS:** `RECORDS_DISPATCH` is **OFF**, so no record has ever been written to chain · the two live holes (`leaderboard`, `onboarded_wallets`) each had a READER and **no writer** until given one · `tenant_vault.go`'s comment contradicts its own `:134` dispatch · `indexerGet` cancelled its context before the body was read (**fixed**, `Problems.md` §38) · **ten files carry the mojibake signature**, `economy_service.go` worst (one 243 KB comment line, 19 lines over 2,000 chars).
- **STATUS:** **partial** — transport, envelope, batch and reader are complete and gated; the seed is deliberately unplanted.

### Theme & bonded branding — the ecosphere-wide visual layer (§13 Theme Engine)
- **OWNER FILES:** `bonded_asset_registry.go` · `bonded_branding.go` · `bonded_market_service.go` · `slide_theming.go` · `card_view_skins.go` · `placeholder_assets.go` · `placeholder_derivatives.go` · `ui_tree_theming.go` · `Public/js/bonded_branding.js` · `slide_theming.js` · `card_view_skins.js` · `theme_engine.js`
- **ROUTES:** `/api/assets` — **21 routes, the widest prefix in the spine** → `bonded_market_service.go` · `placeholder_assets.go` · `bonded_asset_registry.go` · `slide_theming.go` · `bonded_branding.go` · `card_view_skins.go`; `/api/theme` (3 routes, un=0); `/api/market` → `theme_engine.go`.
- **STATE:** `BondedAssetRegistry.Assets`/`Bindings`/`SlideThemes`/`CardViews`/`Listings` (its own mutex, plus a FILE mutex for the write-through) · `l.themeEngine`.
- **FLOW:** WD ▸ Assets ▸ Bonded Branding Studio → `GET /api/assets` (chest + served policy) → mint / bind / theme → `ValidateBondedMedia` → registry write → chain mirror from the registry's own private snapshot.
- **UI:** WD ▸ Assets ▸ **Bonded Branding Studio** (mint · wear · card eyes · market · app slides) · Portfolio ▸ Bonded Assets (read-only) · every dashboard tree via `--wd-tree-art`.
- **GAPS:** **`BindThemeAsset`/`handleBindThemeAsset` take NO caller wallet and never call `guardOwnerHolder`** while every sibling writer does — any wallet knowing an asset id can create and OVERWRITE a binding, and because `MoodTagForWallet` excludes a `Locked` binding, a foreign locked binding removes the TRUE OWNER's asset from their own theme · `Save`/`Load` dereference map entries with no nil guard (a `null` entry panics the persistence worker) · `Burn` relies on a raw prefix compare that is valid only because ids are `BA-`+uuid (stated nowhere) · the money/ownership tear is auditable but not closable without one atomic snapshot (`Problems.md` §14).
- **STATUS:** **partial** — complete as a feature, with one live authorization gap.

### The lock & data-race gate family — how this repository proves its own concurrency (§38 Infrastructure & Security)
- **OWNER FILES:** `selflock_gate_test.go` · `map_race_gate_test.go` · `mutex_relock_gate_test.go` · `lock_order_gate_test.go` · `market_nodes_alias_gate_test.go` · `career_award_lock_test.go` · `record_family_writer_gate_test.go` · `record_note_gate_test.go`
- **ROUTES:** none — these are `go test` artefacts; their acceptance surface is the test binary.
- **STATE:** each gate DERIVES its own subject (**197 → 194** `*Lobby` lock-taking helpers · **47** map fields of `Lobby` · **8** mutex field names from **371** struct declarations · alias pairs from cross-container assignments) and carries a **baseline that fails on a NEW entry AND on a STALE one**.
- **FLOW:** `go test .` → derive subject → walk MUST-held (branches joined, `defer` not releasing at its source) → fail closed → each gate has a synthetic control (must-report + must-not) and, where a static census cannot prove behaviour, a **watchdog** test that fails by TIMEOUT.
- **UI:** none.
- **GAPS:** every gate is **intra-procedural or depth-1** — a re-lock across a CALL EDGE is invisible to the re-lock gate, and the self-lock gate cannot see a self-locking method on a NON-`*Lobby` type · **`-race` is UNAVAILABLE on this host** (no cgo/C toolchain), so **no gate here is a race proof** · unresolvable expressions are SKIPPED AND COUNTED, never guessed.
- **STATUS:** **complete as a system** — 0 violations, with the classes it closed named (the aliased-map hazard, the two ABBA inversions, the transitive self-lock, the recursive read lock).

### Local-LLM pipeline — the bot/pet behavioural corpus (§26 Local LLM Pipeline)
- **OWNER FILES:** `local_model_promotion.go` · `setup_bot_pathway.bat` · `setup_custom_quant_ornith.bat` · `llama_tools/start_ornith_dev.bat` · `llama_tools/start_ornith_dev.txt` · `local_bots/corpus/**` · `Modelfile-Gemma4.txt`
- **ROUTES:** `/api/local-model` (**2 routes**, un=0) → `local_model_promotion.go`; both call the wrapper `.bat`.
- **STATE:** `LocalModelPromotionRegistry.Promotions` (own mutex + `local_model_promotions.json`) — **its field was UNEXPORTED, so `json.MarshalIndent` wrote `{}` and the file was persisted EMPTY all along** (fixed; the file is now the one-line proof of that defect).
- **FLOW:** bot promotion (gated Certified + !BlackMarketAdopted + owner + Level≥25) → `setup_bot_pathway.bat` → corpus concatenated to `%TEMP%\zap_matrix_build\training_code.txt` → the owner harness **skips its workspace snapshot** and runs imatrix + quantize → runner emitted.
- **UI:** Children Bots learning UI and §32 Dev-Game Hub — catalogued, not a purchase surface.
- **GAPS:** **the §31.1 port-namespacing can NEVER run** — the wrapper looks for the harness output in `%USERPROFILE%\OneDrive\Desktop\models` while the harness writes it into `Z:\Model-matrixs`, so `if exist "%EMITTED%"` is false for ever and every identity collides on port 11434 (**proved only by reading the two files against each other**) · the owner harness's emitted launcher writes an **unescaped** `set PATH=%TOOL_DIR%;%PATH%` inside a redirected `echo`, freezing this machine's PATH into the generated file · `start_ornith_dev.txt` is a **`.bat` with a `.txt` extension** whose input glob disagrees with its `.bat` twin · `Modelfile-Gemma4.txt`'s base model is `qwen3.6:27b`, **not Gemma** · the calibration pass **dumps the entire repository's source** into the GPU input.
- **STATUS:** **partial** — the corpus and the promotion path are real; the per-identity runner, and the tier/backend the header documents, are not.

### Justice & criminality — the courthouse, warrants and the bounty board (§9 Justice System · §10 Underworld & Criminality)
- **OWNER FILES:** `justice_handlers.go` · `courthouse_service.go` · `handlers_criminality.go` · `black_market_service.go` · `counterfeit_service.go` · `rival_career_engine.go`
- **ROUTES:** `/api/justice` (**8 routes, un=5**) · `/api/courthouse` (1) · `/api/criminality` (1) · `/api/bounty` (1) · `/api/black-market` (4, all un) · `/api/counterfeit` (2, all un) · `/api/contracts` (2, `server.go`).
- **STATE:** `l.leaderboard` (wanted level, jailed/kidnapped cards, `CareerXP`) under `l.mutex` · the notes vocabulary's nine money-door prefixes · `registeredTxIDs`/`pendingTxIDs` (the idempotency memo).
- **FLOW:** WD ▸ Careers & Factions ▸ **Criminality** hub → courthouse fine / bail / bounty → `claimTxID` **reserves BEFORE** the indexer round-trip → `VerifyBuyInTransaction` (note + txid, fail-closed) → apply under the write lock → `commitTxID`.
- **UI:** `openCriminality` (a hub that always opens, with the courthouse DISABLED when nothing is owed) · `openBountyBoard` · the Intel-Agent cyber-intercept · Portfolio ▸ Legal & Custody (read-only).
- **GAPS:** **`handleCaptureBounty` answers `200 {success:true,reward:N}`, broadcasts and logs a "capture" with NO balance change, no audit and no state change** — an endpoint that reports a payment it does not make · `black_market_service.go` is the densest defect cluster the sweep found (two crash paths — an unguarded `stats.Inventory[...]++` and two `wallet[:8]` slices; a card silently LOST on expiry; four endpoints answering a bare array with three encoding `null`; three floats on money paths) · `criminality.js`'s `wantedLevel` has **no writer**, so the badge is always 0 and its colour ladder can never fire.
- **STATUS:** **partial** — the courthouse rail is complete and hardened; the bounty capture door and the black market are not.

### Tooling & the local dev loop — how the repository builds, watches and verifies itself (§38 Infrastructure & Security)
- **OWNER FILES:** `tools/server/dev_server.ps1` · `tools/config.toml` · `tools/TOOLSET.md` · `package.json` · `package-lock.json` · `render.yaml` · `Dockerfile` + `entrypoint.sh`
- **ROUTES:** none.
- **STATE:** none in-process — the state is the manifests plus the `.env` / `./devdata` pair the dev server seeds.
- **FLOW:** `npm run dev` → `dev_server.ps1` → isolated `./devdata` + `.env` seed (**secrets never printed**) → optional WASM/SASS/server build → start with a port-conflict resolver → a **90 s health budget** on `/api/faucet/status` → watcher + health-watchdog loop.
- **UI:** none.
- **GAPS:** the watcher **did not watch `Public\*.js`**, so a change to **`app.js` — the composition root — was silent** (**fixed**: `Public\*.js` and `Public\*.html` added) · a JS change **is implemented as the sentence** "refresh the browser" where the spec requires cache-clear + refresh and the CDP port is already open · **`-Test` cannot fail a build** (`$testRc` is printed, then a hard-coded `exit 0`) · `render.yaml` declared ~40 env vars of which **the two the server READS were absent** while `ADMIN_KEY` (no reader) was present and `AVOI_ASSET_ID` was listed three times (**fixed**: 21 dropped, 2 added, 22 remain, 0 duplicates) · `package.json` still carries the old name `voiconomy-faucet`, declares `three` while the app loads the vendored copy, and `immutable` (which is also a transitive dep of `sass`, so deleting it changes nothing) · the toolchain floor **Node ≥ 20.19.0** is recorded only in the generated lockfile · `python` is **absent from this host**, so the whole `tools/main.py` dispatcher is un-runnable here.
- **STATUS:** **complete for its purpose** — the dev loop works; the manifest and one watcher clause did not.

### Persistent Identity — reputation as a chain of recorded events (§25 Persistent Identity)
- **OWNER FILES:** `persistent_identity.go` (253) · `identity_bridge.go` (the linked-wallet half)
- **ROUTES:** `/api/identity` (**8 routes, un=4** — the four are inline closures, see the spine note) · `/api/envoi-name` → `handlers_public.go`.
- **STATE:** a **PACKAGE-LEVEL GLOBAL** `persistentIdentity` (:53) holding `profiles` + `events` behind its own mutex — **not lobby-owned and in NO record family** (nothing snapshots it), although the file's own header promises "nothing important should disappear".
- **FLOW:** event → `RecordEvent` → `computeIdentityScore` = `TotalReputation + BestStreak*10 + len(Achievements)*5 + Redemptions*15` (:180-187) → tiers at 50/100/250/500/1000 (:189-203) → `CanLead` (:159, ≥100) / `CanInvest` (:170, ≥50).
- **UI:** `/api/identity/snapshot` + `/api/identity/events` feed Portfolio ▸ Identity ▸ **Linked Identity** (read-only).
- **GAPS:** **`computeIdentityScore` casts a SIGNED reputation to `uint64`** (`uint64(p.TotalReputation)` :182) — a punished player with a negative total wraps to an astronomically HIGH score, so **the worse the reputation, the closer to LEGENDARY** · **`handleIdentityRecord` (:238-252) accepts a CLIENT-DECLARED `impact`** and stores it with `Permanent: true` for the caller's own wallet — self-awarded reputation through a public door · the global is unsnapshotted · `GetProfile`/`GetEvents` return live pointers/slices.
- **STATUS:** **partial** — an owner with a signedness bug that inverts its own score and a client-writable reputation door.

### Industrial Loop — the perpetual value-circulation engine (§24 Industrial Loop)
- **OWNER FILES:** `industrial_loop.go` (257)
- **ROUTES:** `/api/industrial-loop` (**3 routes**) → `industrial_loop.go`.
- **STATE:** a **PACKAGE-LEVEL GLOBAL** `industrialLoop` (:59) — eight phase counters plus a `metrics` map behind its own mutex; **not lobby-owned, in NO record family**.
- **FLOW:** eight phases (Activity → Business → Employment → Purchasing → Taxation → Treasury → Development → Event), each incrementing a counter and calling `updateMetrics` (:194) → `GetMetrics` / `GetFlowSummary` (~:146) / `GetLoopHealth` (:159).
- **UI:** the World Dashboard's infrastructure/industrial panel (read-only).
- **GAPS:** **`handleIndustrialLoopRecord` (:223-257) lets a CLIENT declare any phase and any amount**, which are added to the loop's counters — **fabricated economic telemetry with no authority check**, and this is the loop the Civilization Flywheel is priced from · `GetLoopHealth` computes `efficiency` as a **FLOAT** ratio of circulations to flow (:169-172) and thresholds it at float boundaries (:183-192) — the number the health surface reports · the engine is a package global with no snapshot.
- **STATUS:** **partial** — an owner whose headline metric is client-writable.

### Faith System — churches, rituals and the 24-cap religion chain (§12 Faith System)
- **OWNER FILES:** `faith_church.go` (487) · `religion_governance.go` (458) · `data/religions.json` · `data/faith_churches.json`
- **ROUTES:** `/api/church` (**12 routes, un=1**) → `faith_church.go` · `/api/faith` (**9 routes**) → `entity_event_engine.go` · `religion_governance.go`.
- **STATE:** `FaithChurchEngine` (package global) with `mu` + churches/items/rituals maps — and beside it a **SECOND global, `globalLobbyRef`** (`SetGlobalLobbyRef`/`getGlobalLobby`), used so a method that does not receive the lobby can still reach `l.clubs` (:69-79) · `religionGovernanceEngine` (also a global) · the seeded 24-slot chain in `data/religions.json`.
- **FLOW:** WD ▸ Faith & Church ▸ storefront/religion governance → `POST /api/church/ritual` → `PerformRitual` → `church.FaithPower += cost/10` → church leaderboard → theme/region signals. The chain mirror is dispatched from **inside `Save`'s own read lock** and says so (:82-101).
- **UI:** church storefront · religion governance · Portfolio ▸ Faith (read-only).
- **GAPS:** **`PerformRitual` TAKES NO MONEY AND RAISES FAITH POWER FROM A CLIENT-DECLARED NUMBER** — `church.FaithPower += cost/10` with `cost` read from the **request body** (:458-476, :295-317), so `{"cost": 999999999}` grants 99,999,999 faith power for free · **a church's `opening_cost` is likewise client-declared and never charged** (`handleChurchOpen` → `OpenChurch` stores it as `OpeningCost` with no debit) · **a lock-order edge `faithChurchEngine.mu → Lobby.mutex`** (`OpenChurch` holds its own lock then takes `l.mutex` to create the mirror `Club{Type:"Faith"}`, :159-178) · **the `faith` record family's "religions" claim is NOT satisfied by this writer** (the snapshot is `{churches, items, rituals}`; `religionGov.Religions` is absent, so religion state has a file and **no transport mirror**) · `religion_governance.go`'s ritual door debits `playerBalances` and credits the faucet **with no `l.mutex`** and charges the fee BEFORE `AddRitual` can refuse, never refunding it · `Save` is not atomic and `Load` swallows every error · body-named subjects with no caller check (anyone can add ANY wallet to ANY church).
- **STATUS:** **partial** — an owner carrying a free faith-power mint from a body field.



**13 of the 39 indexed aspects have no row yet, and the reason is a measurement, not a shortcut:** the spine supplies ROUTES and (through the handler→file map) OWNER FILES for the **routed** aspects only, and the keyword pass proved OWNER FILES **cannot** be taken from the prose `OWNS` field by matching. So each remaining row needs its owner entries **read** — and the spine narrows that to a named file set, which is the whole point of having built it.

### Creator Economy & DLC Store — packs, royalties and subscriptions (§22 Creator Economy & DLC Store)
- **OWNER FILES:** `creator_economy.go` (333) · `creator_store_service.go` (470) · `shop_registry.go` (350) · `item_shop_archetype.go` · `Public/js/creator_store.js` · `creator_storefront.js`
- **ROUTES:** `/api/creator` (**19 routes, un=10**) → `creator_economy.go` · `/api/shop` (1r, un) · `/api/v1` (1r) → `redemption_gateway.go` (the console redeems `DLCRegistry`).
- **STATE:** a **PACKAGE-LEVEL GLOBAL** `creatorEconomy` (:74) with `dlcs`/`royalties`/`events`/`subscriptions` behind its own mutex — **not lobby-owned, in NO record family** · `DLCRegistry` (+ `dlcRegistryMutex`, populated in `init()`) · `GlobalShopRegistry` (~40 items across six families).
- **FLOW:** WD ▸ Creator Economy → create a pack → subscribe → purchase. **The purchase path reaches no ledger** (see GAPS).
- **UI:** the Creator Store panel + the storefront; Portfolio ▸ Economy (read-only).
- **GAPS:** **NO MUTATOR TAKES A `*Lobby`, SO NO MONEY CAN MOVE** — `PurchaseDLC(dlcID, buyer)` increments `SalesCount`/`RevenueTotal` and appends a royalty (:110-114) with no ledger access available to it, so **a priced DLC is delivered for free** (the same shape as `entity_shares.go:75` and `launchpad.go:93`) · `creator_store_service.go` computes a platform fee, a creator revenue and a royalty and then updates **only a profile counter, a sales count and a record** — no debit, no credit, no sink routing, no buyer check — and `ProcessSecondarySale` explicitly discards `_ = sellerNetRevenue` · **prices and amounts are CLIENT-DECLARED** (`CreateDLC(..., price)` :82, `CreateSubscription(..., amount, ...)` :325) and `handleCreatorSubCreate` takes the **CREATOR from the body** (:316) · `ShopItem.Price float64` — **a shop price carried as a float** · **the DLC registry is seeded with PLACEHOLDER creator wallets** (`browser_creator_wallet_001`/`_002`) that exist nowhere in the engine, so a redemption **pays a wallet no player owns** · `RateProduct` pairs `mu.Lock()` with a deferred `mu.RUnlock()` (**a guaranteed panic on every rating**) plus three `wallet[:8]` panic sites.
- **STATUS:** **broken** as an economy — the whole purchase path is money-free — with a live panic in the rating door.

### AI Citizens — autonomous economic participants with their own wallets (§11 AI Citizens)
- **OWNER FILES:** `ai_citizen_engine.go` (**1731**) · `entity_event_engine.go` · `Public/js/ai_citizens.js` · `children_bots.js` · the §26 corpus wiring (`setup_bot_pathway.bat`)
- **ROUTES:** `/api/ai` (**12 routes**; **the 8 the spine could not resolve are NOT naming deviations — they are INLINE CLOSURES registered directly**, which is exactly what `un=` counts. The four named doors are `handleMarryAI`, `handleBreedAI`, `handleAdoptPetAI`, `handleAIProgression`) · `/api/children-bots` (1r).
- **STATE:** `AICitizenEngine` (`citizens`/`lobby`/`mu`/`tickInterval`/`regionCounts`) — **its own mutex, with the LOBBY reached inside it** · `AICitizen` (career, tier, treasury, savings rate, learning XP, pathway, BondGraph, §27.7.3 flags, §30 `Stats`/`BotLevel`) · the **12 §15-v3 pathways** (`aiPathwayByCareer`) · `regionCounts` for the region caps.
- **FLOW:** WD ▸ Assets ▸ AI Citizens (or Children Bots) → spawn/adopt → the **1-minute `BehavioralTick`** drives the career actions (heist planning, laundering, fencing, smuggling, bounty hunting, recruitment, evidence analysis, cyber surveillance) → treasury → `SaveCitizens` (its own chain mirror, taken under the engine lock and **RELEASED before dispatch**).
- **UI:** WD ▸ Assets ▸ AI Citizens · Children Bots · Portfolio ▸ Entities (read-only) · the learning UI feeds the §26 corpus.
- **GAPS:** **the file's own header states the contract — `BehavioralTick` holds `ace.mu` for its whole body, so it MUST NOT acquire `lobby.mutex` — and `triggerEntityInvestment` still calls `ace.lobby.handleInvestEntity(...)`, which takes `l.mutex.Lock()`**, from inside that body (reachable through `executeMarketActivity`, `executeSmugglingRoute`, `executeCyberSurveillance`); **the lock-order gate cannot see it because the mutex belongs to ANOTHER object** · **`ace.lobby.playerBalances[citizen.Wallet] = citizen.Treasury` is a lobby-owned map write under only `ace.mu`** (and DISCARDS the wallet's real balance) · `math/rand` decides every reward-bearing AI outcome and all eight incomes · floats sit on AI treasury/XP · every career action **mints treasury with no counterparty**.
- **STATUS:** **partial** — the tick, the pathways, the corpus and the chain mirror are real; the cross-object lock edge and the lobby-map write are not fixed.

### Battle, QuickPlay & Matchmaking — the server-authority half of resolution (§3 Battle System · §6 QuickPlay · §7 Matchmaking & Multiplayer)
- **OWNER FILES:** `battle_service.go` (**2330**) · `main.go` (the `js && wasm` mirror) · `lobby_manager.go` (`processMatchmaking`/`initiatePairedMatch`) · `handlers_public.go` (`/api/match/active`) · `Public/js/game.js`, `game_board.js`, `game_multiplayer.js`, `match_arena.js`
- **ROUTES:** `/api/match` (**2 routes**) → `handlers_public.go`; the match itself rides the **WebSocket** (`handleGameProtocol`), not HTTP.
- **STATE:** `l.matches` · `l.leaderboard` (the whole career-XP ladder writes it) · `l.inventory` + `persistentCardCache` (the Artifact scar) · `MatchState.P1/P2BondedBoostPct`, snapshotted at pair time so Server and WASM apply one integer to one base.
- **FLOW:** `processMatchmaking` (**holds the WRITE lock for its whole body**) → `initiatePairedMatch` → the WS game protocol → `verifyWinner` → captures / the 16-block career-XP ladder / wagering / jailing / fatigue → the match deleted. `getEffectiveServerPower` is a **line-for-line twin** of the client's `getEffectivePower`, and `ComputeBoardHash` is sha256 over **fixed-width big-endian** fields "to ensure cross-platform parity between 64-bit Server and 32-bit WASM".
- **UI:** the board (`game_board.js`) · Match Arena · QuickPlay · the multiplayer panel.
- **GAPS:** **a TEAM-SYNERGY XP AWARD IS BROADCAST TO EVERY KIDNAPPER ON THE SERVER** — its loop ranges `l.leaderboard` behind a comment that says "same organization" **and writes that map while ranging it** · **two live underflow-shaped rival awards survive** (`uint64(int(rivalXP)-int(scaledXP))` with a `> 0` guard that cannot protect, and an unguarded `ComputeScaledXP(rivalXP-hpXP, …)`) — the class repaired elsewhere in session (j) · **floats on XP throughout** while the INTEGER owner `ComputeScaledXPPermille` is used **nowhere** in the file · the **sudden-death redistribution is `rand.Shuffle`d**, so the tie-breaker cannot be reproduced or mirrored by WASM · `CalculateReputation` is called under the write lock and its result ADDED to the existing reputation · five payouts credited with no counterparty.
- **STATUS:** **partial** — the authority, the parity design and the ladder are real; the broadcast leak, the underflow shapes and the non-deterministic tie-breaker are not fixed.

### Entity Markets — the public market for pets, children bots and vehicles (§14 Entity Markets)
- **OWNER FILES:** `entity_market.go` (613) · `Public/js/entity_market.js` · `life_assets.js`
- **ROUTES:** `/api/entity` (3r) · `/api/pet-battle` (3r) · `/api/rivalry/factions|join` · `/api/children-bots` (1r) → all `entity_market.go`.
- **STATE:** `EntityMarket` (**its own `sync.RWMutex`**, `listings`, `tradeLog`) · `RivalryGroup`, `Faction`, `PetBattleArena` — **THREE PACKAGE GLOBALS** (`entityMarket`, `rivalryGroups`, `petBattleArena`), seeded by an `init()` that creates two factions **before any Lobby exists** — none in a record family.
- **FLOW:** WD ▸ Assets / Competition → list an entity → `PurchaseListing` transfers `PriceMicro` exactly (a pure player-to-player transfer); the `init()`-seeded factions and the pet-battle arena sit beside it.
- **UI:** the entity-market panel · the pet/vehicle arena placeholders · Portfolio ▸ Bonded Assets (read-only).
- **GAPS:** **THE MONEY MOVES UNDER A LOCK THAT DOES NOT OWN IT** — `PurchaseListing` debits the buyer and credits the seller under `em.mu` only, while `l.playerBalances` is **LOBBY-owned** (`l.mutex`) and is one of the maps the unfenced-map census covers, so a purchase concurrent with a fee route touches the same map with **no common lock** (the §31/§35 hazard class, on the money map itself) · **FOUR lock domains in one file** (`EntityMarket.mu`, `RivalryGroup.mu`, `PetBattleArena.mu`) **and no lock at all for the lobby state they reach into** (`l.pets`, `l.vehicles`, `l.playerBalances`, `l.aiEngine`) · `computeRivalryBonus` takes `rivalryGroups.mu.RLock` **inside** `ResolveBattle`'s `ba.mu.Lock` window — an order no other site uses · `ResolveBattle` credits **0.5 $VBV with no debit anywhere**, and a SIGNED ±5 rivalry bonus is cast to `uint64`, turning a defender's advantage into 18,446,744,073,709,551,611 — the class session (j) repaired on rival XP, live here.
- **STATUS:** **partial** — the market trades, but under the wrong owner's lock, and it mints on the arena path.

### Auctions — the Art Gallery Commission rail (§19 Auctions)
- **OWNER FILES:** `auction_service.go` (472) · `Public/js/economy.js`
- **ROUTES:** `/api/auctions` (1r) — create is a POST on the same path; bid and settle ride the service.
- **STATE:** `l.auctions` (a Lobby map, under `l.mutex`) · the art-gallery club's `TreasuryMicro`, resolved by `getClubByTerritoryID("the_archive")`.
- **FLOW:** list (escrows the bundle via `TransferBundleItems`) → `HandlePlaceBid` (escrows an internal balance, or verifies an on-chain approval and **refunds the previous bidder only if that bid was virtual**) → `ProcessAuctions` (expiry) → `(bid*10+50)/100` commission, net to the seller, the remainder to the club treasury, then `applyDynamicScalingLocked`.
- **UI:** the Markets panel · Portfolio ▸ Entities (read-only).
- **GAPS:** **IT PAYS THE SELLER EVEN WHEN `pullApprovedTokens` FAILS** — the failure is logged and settlement proceeds ("maintaining the Token-Sink promise") · the on-chain record fires only at `amountBase >= 100.0` and **its payload carries a FLOAT `amount`** · `StartPrice float64` with `uint64(req.StartPrice*1000000 + 0.5)` — **a float listing price and a float-derived micro value** · **a race on the shared `Auction` struct** (name fields written on pointers after the lock is released) · **four `_`-discarded errors on the ARC-200 payment path** (`DecodeAddress` ×2, `ParseUint`), so a malformed address yields a zero-value address / appID 0 and **the transaction is still built and signed** · `CalculateReputation` is called under the write lock.
- **STATUS:** **partial** — the escrow/refund logic is careful; the settlement completes when its own transfer has failed.

### Tournaments — the pot, the top-5 split and the on-chain archive (§16 Tournaments)
- **OWNER FILES:** `tournament_manager.go` (**~1900**) · `Public/js/tournament_brackets.js`
- **ROUTES:** `/api/tournament` (**2 routes, un=2** — inline closures, per the spine note) · `/api/tournament/history`.
- **STATE:** the **live `tournament`** field on the Lobby (state the economy snapshot marshals that **no record row names**) · `paidParticipants` · the arena centre club's `TreasuryMicro` (overwritten from a router node — see GAPS).
- **FLOW:** buy-in → the bracket → `DetermineTop5` → the pot split (**Governors 25 % · top-5 15 %**, per the stated Pillar-1 political rule) → `govTaxMicro = (PotMicro*5)/100` routed for governance → the on-chain archive via `sendNoteTx`.
- **UI:** the Tournament panel (bracket) · Portfolio ▸ Record (read-only).
- **GAPS:** **THE ARENA CENTRE CLUB'S TREASURY IS OVERWRITTEN FROM A ROUTER NODE** — `centerClub.TreasuryMicro = node.TreasuryBalance` (:674-677) **DISCARDS whatever the club's own accounting held**, and `strconv.ParseUint(strings.TrimPrefix(centerClub.ID, "CLUB-"), 10, 64)` discards its error · **`DetermineTop5` is INTENTIONALLY UNEQUAL and says so** — Governors (2+ territories) sort ABOVE higher-reputation semi-finalists, deciding 25 % + 15 % of the pot, **and the two lookups are not the same comparison** (`getRep` lowercases while `isGov` lowercases then `EqualFold`s) · **THE HISTORY READ IS PAGE-LIMITED TO 1000 TRANSFERS WITH NO PAGINATION**, so past 1,000 vault transfers the archive silently truncates and the checksum/receipt verification runs against a partial history **while reporting success for whatever it saw**.
- **STATUS:** **partial** — the pot splits and pays; its own club accounting and its history verification are not trustworthy.

### Seasonal Events, Treasure & the Region Views (§17 Seasonal Events)
- **OWNER FILES:** `seasonal_event_engine.go` (**882**) · `Public/js/seasonal_events.js`, `season_countdown.js`, `treasure_map.js`, `world3d.js`
- **ROUTES:** `/api/season` (**8 routes, un=7** — inline closures) · `/api/events` (2r, un=2) · `/api/treasure` (2r, un=2) · `/api/regions` (1r, un) → `GetRegionViews`.
- **STATE:** **four maps on the engine** (`ActiveEvents`, `CurrentRewardPool`, `Caches`, `UserEvents`) plus `GenerateRandomSeasonalEvent`/`ScheduleSeasonalEvent` (**a goroutine + `time.Sleep`**) — the engine is reached from the lobby but its maps are **not in a record family**.
- **FLOW:** create → activate → register → resolve → `DistributeEventRewards` (a treasury pool × a multiplier) · treasure caches spawn/claim · user-authored events take an entry fee and pay a royalty · **`GetRegionViews` is the `/api/regions` payload the 3D explorer and the menu warp read**.
- **UI:** the Seasonal Events panel · the Treasure Map · the Season countdown · **the 3D world's region meshes** (live world-dynamics readout).
- **GAPS:** **`GenerateRandomSeasonalEvent` and the schedulers are the only non-deterministic producers in an otherwise deterministic world** (a goroutine + `time.Sleep` deciding an event) · the engine's four maps are unsnapshotted · `season_countdown.js` was reading fields `/api/season/status` (a bare ARRAY) has never carried, so it always showed "Season 1 · Active" with a timer that never started (**that read was repaired to the served shape in session (f)**).
- **STATUS:** **partial** — the event layer, the treasure rail and the region views are real and load-bearing (the 3D client depends on them); the scheduling is non-deterministic and the state is unrecorded.

### Achievements & Trophies — the trophy hall (§18 Achievements & Trophies)
- **OWNER FILES:** `achievement_handlers.go` (202) · `achievement_service.go` (296) · `Public/js/achievements.js`, `achievement_progress.js`
- **ROUTES:** `/api/achievement*` (**3 routes**) → `achievement_handlers.go`; unlocks also fire from `battle_service.go`, `courthouse_service.go` and the economy checks.
- **STATE:** `PlayerStats.Achievements` + `Reputation` (lobby-owned, under `l.mutex`) · 17 canonical definitions held as a literal (:22-52).
- **FLOW:** an event (match, fine, treasury recovery, tax milestone, philanthropy) → `UnlockAchievement` (**takes the lock and DELEGATES** to `UnlockAchievementLocked` — the one-owner pair) → audit + notify every session for that wallet + a lobby update and `saveLeaderboard` **in goroutines**.
- **UI:** Portfolio ▸ Achievements (read-only) · the achievement panel reads `/api/achievements` with `wallet` **OPTIONAL**, so it works wallet-less.
- **GAPS:** **`CheckPhilanthropistAchievementLocked` uses `rand.Float64() < 0.25` (:232)** — a **random trophy unlock** driven by the global RNG, in a codebase whose mandate is determinism and which elsewhere insists on sha256 seeding; the same trophy is unreproducible across replays · **`handleUnlockAchievement` takes the wallet from the REQUEST BODY with no caller authentication** (:161-194), so any caller can unlock **any wallet's** trophy — the only gate is that the id exists in the canonical list · `TransferBundleItems` writes `l.leaderboard[wallet]` with the **RAW** spelling (:295) while `UnlockAchievementLocked` lower-cases (:27), so a mixed-case caller can create a **second leaderboard key** · `allAchievements()` is called **inside the inner loop** (:134), re-allocating the 17-entry slice once per achievement per player under the held read lock.
- **STATUS:** **partial** — the pair, the optional-wallet read and the non-null `[]` are right; a random unlock and a body-named unlock door are not.

### Spectator & Replay — the live frame broadcast (§20 Spectator Mode)
- **OWNER FILES:** `replay_engine.go` (304) · `Public/js/constellation_spectate.js`, `spectate.html`, `watch-feed.html`
- **ROUTES:** `/api/replay` (**7 routes**) · `handleDiagnostics` · `handleClientError`.
- **STATE:** a **PACKAGE-LEVEL GLOBAL** `replayEngine` (:68) — `frames`, `frameID`, `isRecording`, `isPlaying`, `players`, `subscribers` behind its own mutex · a rolling **160-frame** buffer.
- **FLOW:** `StartRecording` → `CaptureFrame` → the broadcast **skips a slow subscriber rather than blocking** (`select` + `default` :106-109) — the correct broadcast shape · the replay path compares the client's `ComputeBoardHash` against the server's `frame.StateHash` and triggers recovery on a mismatch, **skipping the check when the server sends a zero hash** (legacy/spectator compatibility) — recorded as the tree's cleanest cross-boundary determinism contract.
- **UI:** the constellation spectate surface · `spectate.html` · `watch-feed.html` · the 3D world's fly-through.
- **GAPS:** **`handleDiagnostics` prints a FABRICATED uptime** — `time.Since(time.Now().Add(-time.Minute))` is always ~1 minute regardless of the server's age **and the page labels it "Uptime"**; the same page promises an error feed that `handleClientError` **never feeds** (it only logs) · **`handleReplayCapture`/`Start`/`Stop` carry NO authorization** — any caller can inject frames or toggle recording · `GetLatestFrame` returns `&re.frames[len-1]` and `GetPlayerFrame` a live pointer, **both after the read lock is released**, while `CaptureFrame` re-slices the same buffer — a caller holding the returned pointer reads a slice another goroutine may be re-headering · `players` is keyed by the **RAW** wallet with no canonicalisation.
- **STATUS:** **partial** — the buffer, the skip-don't-block shape and the hash contract are right; a live page states a false uptime and the record doors are unauthorised.

### Governance — regional governors, elections and the weight formula (§21 Governance)
- **OWNER FILES:** `governance.go` (315) · `club_service.go` (the territory/regional half) · `Public/js/governance.js`
- **ROUTES:** `/api/governance` (**7 routes**) → `governance.go` · `/api/regions` (1r) · `/api/clubs` (1r).
- **STATE:** a **PACKAGE-LEVEL GLOBAL** `governanceEngine` (:60) — `governors` + `elections` behind its own mutex — **not lobby-owned, in NO record family**.
- **FLOW:** `ComputeWeight` (every sub-score an integer division of the identity score: `trust = rep/2`, `econ = comp = rep/4`, `community = creative = rep/8`) → `RegisterCandidate` → `Vote` → `CloseElection` → `GetGovernor`.
- **UI:** WD ▸ Governance → governor/election panels · Portfolio ▸ Obligations ▸ Governance (read-only weight).
- **GAPS:** **`handleGovernanceVote` takes a client-supplied `Weight` (:262) and passes it to `Vote` (:269)** — **a voter declares their own voting power** instead of having it derived · **the weight is derived from `uint64(profile.TotalReputation)` (:72) — the SAME signedness cast recorded in `persistent_identity.go:182`**, so a negative reputation feeds a huge governance weight · **`ComputeWeight` IS DUPLICATED IN THE SAME FILE** (the identical block at :66-93 and again at :~205-232) · `handleGovernanceClose` closes **any region's** election with no role check · a cross-mutex edge `ge.mu → pi.mu` (the engine lock held while `persistentIdentity.GetProfile` takes its own) · the global is unsnapshotted.
- **STATUS:** **partial** — the ladder and the integer weight formula exist; the vote door accepts its own power and the identity cast undermines the formula.

### Admin Tools — the entire administrative surface, signature-gated (§31 Admin Tools)
- **OWNER FILES:** `handlers_admin.go` (**2182**) · `network_registry_view.go` (the redaction owner) · `reward_registry.go` · `Public/js/admin_console.js`, `admin_suite.js`, `admin.js`
- **ROUTES:** `/api/admin` (**28 routes — the WIDEST prefix in the spine**) · `/api/os` (5r) · `/api/compliance` (6r) · plus `/api/ban-player`, `/api/reset-stats`, `/api/re-sync-stats`, `/api/refill-vault`, `/api/maintenance-mode`, `/api/report-player`, `/api/system-message`, `/api/update-rules`.
- **STATE:** the audit log (**5 MiB rotation to `.old`**, with a `server_load` field) · `l.availableNetworks` (via the redacted projection) · `l.rewardTokens` · `l.nonces` (read under `RLock` by `verifyAdminSignature`).
- **FLOW:** WD ▸ **System & Ops ▸ Admin Console** (one reachable surface) → `checkAdminAuth` is the **first statement of every door** → `verifyAdminSignature` → `isAdminWallet`. The 40 doors cover refill, rules, the reward registry, networks, power scaling, system messages, bans, stat resets, base reward, maintenance, tournaments, season rollover, audit export, simulations, forfeiture, force payout, node/sanity/ledger audits, emergency shutdown, the DLC registry and the tax audits.
- **UI:** the Admin Console (signature-gated, with a banner that STATES a 401 refusal) · the Admin Suite's 24 `data-admin-action` controls.
- **GAPS:** **`killExistingServer` IS REFERENCED BUT NEVER DEFINED — the console target cannot compile** (it is one of the two measured errors) · this file is also where the tree's **canonical `...Locked` pairs** live (`logAdminAudit`/`logAdminAuditLocked`, `broadcastToAdmins`/`broadcastToAdminsLocked`), the second of which exists because two callers hold the WRITE lock and reaching the read-locking form from under it *"freezes every request and every WebSocket in the process"* · **the console target's route table is a hand-maintained mirror of `server_main.go` with nothing asserting parity** — until this unit, when `npm run verify:routes-parity` began measuring it (**189** routes in `server_main.go` alone, **3** in `console_server.go` alone).
- **STATUS:** **complete as a surface** — one reachable gate, a redaction owner that panics at boot if a secret-shaped key appears, and a parity gate that now holds the asymmetry visible.

### Console Linking & the Console Target — the native build entry (§34 Console Linking · §33 Nautilus DEX Path)
- **OWNER FILES:** `console_server.go` (271, `//go:build console`) · `redemption_gateway.go` (222) · `nautilus_dex_path.go` (97) · `shop_registry.go` (`DLCRegistry`) · `tools/build/build_console.ps1` / `.sh` · `tools/build/build_mobile.ps1` / `.sh`
- **ROUTES:** `/api/v1` (1r) → `redemption_gateway.go`; **`console_server.go` registers 268 routes and `server_main.go` 326 — 58 exist in server_main.go alone and ZERO in the other** (parity gate; the last five additions to that 58 are the CHAIN-RAIL RULING below)
- **STATE:** `l.linkedWallets` (`identity_bridge.go`) — the redemption path resolves a wallet through it · `DLCRegistry` (under `dlcRegistryMutex`) · `ArenaVouchers` and the three tax totals the redemption writes.
- **FLOW:** a console account → `handleRedemptionGateway` (POST) → `executeRedemptionLocked` under the lobby write lock → `RouteCriminalTax(..., {FaucetShare:1.0}, ...)` + `SimulateVoiToVbvSwap` + `ExecuteMarketBuyLocked` for the creator payout → stock decrement → `ArenaVouchers -= dlcValueMicro`. The console binary binds to **LOOPBACK only** so the simulation never exposes a public surface, and kills any process holding :8090 first.
- **UI:** none in-app — the console is a native build. The voucher path is reached by the console client.
- **GAPS:** **THE CONSOLE TARGET NOW BUILDS — CLOSED 2026-09-20, and it was TWO DIFFERENT CAUSES, neither of them a missing implementation:** `killExistingServer` EXISTED but sat in `server_main.go` (tag `!js && !wasm && !console`), so it was never compiled into the console target — it moved to `server.go`; `handleFaithConverted` existed in **NO `.go` file** at all, so the line was a FABRICATED route (one of three console-only) and was REMOVED, with the other two console-only routes aligned onto the primary server's own paths. **Measured: `go build -tags console .` rc 0 (32,208,896 B) · console routes 140 → 139 · console-only 3 → 0 · the parity exemption list SHRANK 189 → 187** (the gate first FAILED naming both stale exemptions). Its 139 routes are now COMPILED, not parse-verified · **the redemption path trusts a `Verified` flag that is set with NO verification anywhere** (`identity_bridge.go` sets `Verified: true` from a client-asserted address, so the gate trusts a flag a client can cause to be set) · the path **pays out even when its own `pullApprovedTokens`-shaped transfer fails** (the same shape as the auction) · `nautilus_dex_path.go` **credits raised funds with no debit** (the silent-mint class recorded at `entity_shares.go:75` and `launchpad.go:93`) · `build_mobile.ps1` sets `CGO_ENABLED="1"` for iOS then **overwrites it with `"0"`** — **this was the other half of the same blocker and is now clearable: both mobile wrappers depended on the console target, so the target building is the precondition, but the CGO overwrite is still theirs to fix.** · the 187 remaining exemptions are the honest ledger of what parity has not yet landed.
- **STATUS:** `partial` — **the target BUILDS (2026-09-20: `go build -tags console .` rc 0, a 32,756,736 B binary with 273 routes), so the entry point defect that made this row `broken` is CLOSED and its routes are COMPILED rather than parse-verified.** What remains is (a) a TRUE parity residue of 53 paths — the deliberately-excluded operator family plus the served HTML roots — and (b) the rail defect below. The mobile twins built on this target now PRODUCE artifacts for Android (measured via the script); iOS remains unbuildable from Windows by requirement, not by defect.

- **OPERATOR DECISIONS (2026-09-20, given directly — BINDING, and they ANSWER the parity question this row was holding open):**
  1. **NO ADMIN ENTRY POINT ON CONSOLE — BY DESIGN, not by omission.** Console users cannot utilise the crypto side on that device, so the 53 residual parity exemptions are **POLICY, not gaps**: the operator family is deliberately absent. A console is a **virtual mirror**, never a spending surface.
  2. **VIRTUAL MIRROR + MANUAL BRIDGE.** When accounts are linked and bridged and the user continues on the console, the console **mirrors virtual** and **waits for a MANUAL bridge to the PC version**. Consequence for code: no console path may present a SELF-SERVICE crypto rail, and nothing may imply the console is authoritative for chain value.
  3. **THE NAUTILUS DEX PATH IS MANUALLY EXECUTED.** A user may withdraw $VBV directly as $VOI at Nautilus; **the application does not automate it.** So `nautilus_dex_path.go`'s "credits raised funds with no debit" is not to be fixed by adding an automated debit — the rail is manual, and what the app owes is a DECLARED manual path plus a record, not a hidden transfer.
  4. **CONSOLE DLC BUYS LAND AS USDC IN THE ADMIN WALLET.** The admin then tops the faucet up BY HAND with $VBV / $UNIT / $NUGGET — or any other vault.
  5. **A CAP / LOCK IS REQUIRED BEFORE THAT RAIL MAY BE ENABLED:** a spam of sales must not be able to drain the vault. **This is a PRECONDITION, not a nicety — the rail ships with the cap or it does not ship**, and the cap must be server-owned (never client-declared), integer, and observable in the served state.
  6. **THE FAUCET VAULT MAY HOLD SOME USDC** to enable self top-up, if possible — an operator configuration, not a client control.
  - **WHAT THIS CHANGES HERE:** the redemption rail recorded in FLOW is the very path that must be CAPPED (item 5); the `Verified` flag trust below still stands unchanged; and the parity gate's remainder is hereby declared rather than pending, so a future session must not "close" it by mirroring the operator doors.

- **THE CHAIN-RAIL RULING (2026-09-20 b) — the line is CRYPTO vs VIRTUAL, and it was drawn by MEASUREMENT.** Every one of the nine chain-touching routes had its handler body **CENSUSED from its `func` line to the next function** (not a fixed window — a 90-line window missed `HandleVoiOnboarding`'s five signing calls, which sit at :321-333 of a 232-line body) and searched for on-chain primitives (`SignTransaction`, `SendRawTransaction`, `MakeApplicationNoOpTx`, `VerifyBuyInTransaction`, `TransferToChain`, `SimulateVoiToVbvSwap`, `dispatchReward`, `sendNoteTx`).

  | Route | handler | body | chain calls | verdict |
  | --- | --- | ---: | ---: | --- |
  | `/api/bridge/onboard` | `HandleVoiOnboarding` | 232 ln | **5** | **REMOVED** — it SIGNS a two-transaction atomic group from the vault |
  | `/api/v1/redemption_gateway` | `handleRedemptionGateway` | 30 ln | **2** | **REMOVED** — swaps and buys; this is the rail the CAP must protect |
  | `/api/loans/repay` | `HandleRepayLoan` | 123 ln | **3** | **REMOVED** — the repayment path signs |
  | `/api/loans/take` | `HandleTakeLoan` | 100 ln | 0 | **REMOVED ANYWAY** — a loan that cannot be repaid on this device is a TRAP; the pair goes together and only the READ remains |
  | `/api/faucet/claim` | `handleFaucetClaim` | 31 ln | 0 | **REMOVED** — it moves nothing and answers success: a fabricated rail, excluded on principle rather than on chrome |
  | `/api/shop/purchase` | `HandleShopPurchase` | 62 ln | 0 | **KEPT** — an internal $VBV spend is exactly what a virtual mirror does |
  | `/api/contracts/assign` | `handleAssignContract` | 25 ln | 0 | **KEPT** — internal escrow |
  | `/api/loans` | `HandleGetLoans` | 27 ln | 0 | **KEPT** — a read |
  | `/api/bridge/txs` | `handleBridgeTxs` | 6 ln | 0 | **KEPT** — a read |

  **Effect: console 273 → 268 routes · parity exemptions 53 → 58 · `go build -tags console .` rc 0 · duplicates 0.** The gate first FAILED naming all five as unbaselined, then the derived baseline was regenerated. The rule is recorded in `console_server.go` itself, in a comment that deliberately quotes no path (the parity gate raw-scans comments). **A console client can still SEE its loans and its bridge history; it cannot transact.**


### Economy Bootstrap & State Recovery — the boot-time restore (§35 Economy Bootstrap & State Recovery)


- **OWNER FILES:** `economy_bootstrap.go` (180) · `economy_persistence.go` (216) · `snapshot_guard.go` (170) · `data/economy_state_authoritative.json`
- **ROUTES:** none — this runs at boot and on a 15-minute cadence.
- **STATE:** `RouterSnapshot` (:23-37) and the hydration of `ActiveClubs` / `MarketNodes` / `RegionalDistricts` / `linkedWallets`.
- **FLOW:** `NewBootstrapEngine` → `BootstrapAuthoritativeState()` → read the file **or initialise a clean tree** when it is absent (`loadDefaultStructures`). **The lock order is exemplary and says so:** the router lock is taken at :102 and **released EXPLICITLY at :147 BEFORE the lobby lock at :150**, with the ABBA reasoning in the comment. Fresh mutexes are initialised for restored nodes (:135) — a serialized mutex must never be reused — and each node is pointer-copied to unique memory.
- **UI:** none.
- **GAPS:** **THE MAPS ARE REPLACED WHOLESALE AND ONE OF THEM IS ALIASED** — :123/:132/:140 assign a **brand-new map** to `Router.ActiveClubs` / `Router.MarketNodes` / `Router.RegionalDistricts`, while **`Lobby.marketNodes` IS `TokenSinkRouter.MarketNodes`** (the alias §35 fixed and a THIRD gate now measures). **If the bind happens BEFORE this call, the lobby's field points at the map this function just discarded** — the alias is broken and every lobby-side read sees stale/empty market state. **Boot ORDER must be confirmed in `server.go` — flagged, not asserted.** Also: `rotateBackups` renames the CURRENT authoritative file to `.1` **before** the `.tmp` rename, so a crash in that window leaves only the backup — the "atomic commit" covers the temp write but not the rotation; and the telemetry boundary converts uint64→float64, losing precision above 2^53 micro.
- **STATUS:** **partial** — the lock order and the fresh-mutex discipline are models; the wholesale map replacement may detach the lobby's aliased market map.

### Item Shop Archetype — player-created items and their bonded NFT (§28 Item Shop Archetype)
- **OWNER FILES:** `item_shop_archetype.go` (292) · `item_service.go` · `Public/js/items_equip.js`, `shop_registry.js`
- **ROUTES:** `/api/items` (**5 routes, all un** — inline closures) → the registry's build/bind/collection doors.
- **STATE:** `itemArchetypeRegistry` (12 archetypes → ClubType/Careers/Pathway/Rivalry) · `itemArchetypeByClubType` · `ItemRegistry.items` + `item_registry.json`.
- **FLOW:** `BuildItem` → `BindNFT` → `GetRegistry`/`GetCollection`. **The persistence shape is CORRECT and is the model the other registries follow:** `Save` takes the snapshot under the REGISTRY's own lock and **RELEASES it**, then dispatches the chain mirror (`NotePrefixItemSnapshot`) and writes `.tmp` + rename — the record is a transport mirror of the same payload, **never a live map**.
- **UI:** WD ▸ Assets ▸ Inventory & Equipment · Portfolio ▸ Deck/Economy (read-only).
- **GAPS:** **`PowerScale` is a FLOAT** (`math` is imported and `PowerScaleForPrice` is the float owner) — a built item's power is carried as a float rather than permille, contrary to the Ledger · `GetRegistry`/`GetCollection` return **live `*BuiltItem` pointers**, so a caller can mutate registry state outside the lock · the archetype table **names pathways and careers as literals with nothing asserting they exist** in `aiPathwayByCareer` / `rivalPairTable` — the drift class the career-path guards closed elsewhere.
- **STATUS:** **partial** — the persistence discipline is the tree's best example; the power figure is a float and the archetype literals are unguarded.

### Multi-Chain Bridge & the funding rails — where a priced thing is sold (§23 Multi-Chain Bridge · §1 Architecture)
- **OWNER FILES:** `bridge_service.go` (entry at `14_flow_go.md:198`) · `bridge_router.go` (entry at `:522`) · `entity_shares.go` (160) · `launchpad.go` (entry at `:642`) · `economy_bootstrap.go` · `onboarding_service.go` (475)
- **ROUTES:** `/api/bridge` (**6 routes, un=1**) · `/api/launch*` (7r, un=0) · `/api/shares` (4r, un=0).
- **STATE:** a **PACKAGE-LEVEL GLOBAL** `entityShares` (:42 — `tokens` + `holdings` behind its own `sync.RWMutex`, **not lobby-owned and in NO record family**) · a **PACKAGE-LEVEL GLOBAL** `launchEngine` (:64 — `projects`, the same shape).
- **FLOW:** issue a share → buy → get holdings; create a launch → back → activate → integrate; `/api/onboard` provisions a dual-chain AVM address and pays the gas stipend (a two-transaction atomic group).
- **UI:** `entity_shares.js` · `launchpad.js` · Portfolio ▸ Bonded Assets ▸ Share Holdings (read-only) · `first_run.js` for onboarding.
- **GAPS:** **TWO PACKAGE GLOBALS ON THIS RAIL SELL A PRICED THING AND MOVE NO MONEY.** `entity_shares.go`'s `BuyShares` carries the comment *"Transfer VBV from holder to issuer (simplified: deduct from faucet for demo)"* and **the body below it contains no debit and no credit** — a mint with a price tag on it, the Ledger's "no silent minting" broken directly; and `launchpad.go`'s `BackLaunch` **credits `RaisedMicro` without debiting anything**, then flips the status to `funded` on the strength of money that does not exist. **Both are unsnapshotted**, so neither survives a restart even though the record set exists to make state survive. Also: `TotalSupply` is stored and **never enforced** · `handleEntitySharesIssue` applies **no ownership check** on `issuer_id` (any caller can issue in any wallet's name) · `ActivateLaunch`/`IntegrateLaunch` have **no role check** · `GetHoldings`/`GetLaunches*` return live slices/pointers · `IssueToken` derives an identifier **from a container size** (`len(m.tokens)`).
- **STATUS:** **broken** on the funding half — two rails that price a sale and never charge for it, outside every record family.
- **EVIDENCE NOTE (UPDATED, 2026-09-20):** `bridge_service.go` and `bridge_router.go` have now been **read**, so the reservation this row previously carried is discharged. **`bridge_service.go` (104 lines) is the EXPLICIT WASM→JS export table** (`//go:build js && wasm`) — ~85 `js.Global().Set` hooks in 8 groups (identity, battle, state export, intelligence/HUD, inventory, audio, system overrides, menu/controller) — and it carries its own two findings: **`handleJSCallback` is defined and NEVER used**, with a comment promising a yield to the JS event loop that the body does not perform; and **this file is the ONLY authoritative list of client-callable Go globals while the handler gate counts 68 Go/WASM publishers**, so the two must be reconciled — a mismatch is either an unreachable handler or a dead export. **`bridge_router.go` (240 lines) is the CHAIN half and it confirms the rail's shape:** `ConfirmBridge` (:104-131) **MINTS a `ChainAsset` on request** — a POST to the confirm door creates bridged ownership with **no verification of any on-chain event and no ownership check on the caller**; `BridgeAsset` **never moves money** (`FeeMicro` is recorded from the table but no balance is debited and no sink is credited); an **UNKNOWN `fromChain` bridges FREE** because `chainFees[fromChain]` is a map miss returning 0 and nothing validates the chain names; `AmountMicro` is accepted unchecked (no supply, balance or cap is consulted); and its read helpers return **nil slices, which encode as `null`**. Like its siblings it is a **PACKAGE-LEVEL GLOBAL** (`bridgeRouter`, :67) holding `assets`/`txs`/`chainFees` behind its own mutex, **in no record family and unsnapshotted**.

### Composable Framework — modules, leases and the compliance engine (§32 Composable Framework)
- **OWNER FILES:** `gaming_os.go` (402) · `Public/js/gaming_os.js` · `mechanic_defs.js`, `mechanics.js` (the authoring surface)
- **ROUTES:** `/api/os` (**5 routes**) · `/api/compliance` (**6 routes**) → `gaming_os.go`.
- **STATE:** **TWO PACKAGE-LEVEL GLOBALS IN ONE FILE** — `gamingOSEngine` (:49, `modules` + `leases`) **and `complianceEngine`** — neither lobby-owned, neither snapshotted, both in no record family.
- **FLOW:** `RegisterModule` → `GetModules`/`GetModulesByCategory` → `LeaseModule(tenantID, moduleID, durationDays)` records a lease; compliance runs parallel — `CreateRecord` → `ResolveRecord`/`EscalateRecord` → `GetRecords`/`GetRecordsByWallet`.
- **UI:** WD ▸ System & Ops · the §32 Dev/Game Hub (a stated read-only catalogue) · Portfolio ▸ Obligations (read-only).
- **GAPS:** **`LeaseModule` (:104) RECORDS A `MonthlyRate` AND A LEASE WITH NO PAYMENT AND NO BILLING** — the identical gap the tenant-world-lease design names as its blockers (*"no billing run, so no suspension or eviction can be honest"*), so module leasing is free for ever · **`RegisterModule` takes a CLIENT-DECLARED monthly rate (:55) and any caller can register a module** · **`handleComplianceRecord` (:343) LETS ANY CALLER CREATE A COMPLIANCE RECORD NAMING ANY WALLET WITH ANY SEVERITY** — an accusation door with **no authorization, no evidence requirement and no review** — and `handleComplianceResolve` (:358) and `handleComplianceEscalate` (:373) are **equally unauthenticated**, so anyone can clear or escalate anyone's flag · **two package globals, neither snapshotted**, so modules and compliance records vanish on restart.
- **STATUS:** **partial** — the module catalogue and the compliance lifecycle exist; the lease bills nothing and the accusation door is open to all.

### Web-3D Client & the WASM boundary — the export table and the parity contract (§27 Web-3D Client)
- **OWNER FILES:** `bridge_service.go` (104, `js && wasm`) · `main.go` (the `js && wasm` half, entry at `14_flow_go.md:2091`) · `menu_state.go` (252) · `Public/js/world3d.js`, `mechanics.js`, `mechanic_defs.js`
- **ROUTES:** none — the client reaches the engine through **`js.Global()` hooks**, not HTTP; `/api/regions` feeds the 3D world and `/api/theme/vector` feeds the palette.
- **STATE:** the client-callable global surface — **~85 `js.Global().Set` hooks in 8 groups** (identity, battle, state export, intelligence/HUD, inventory/assets, audio, system overrides, menu/controller) — plus `Game.mutex` and the WASM-side menu/controller state.
- **FLOW:** `registerWasmHooks()` at boot → the JS tier calls those globals → `GetGameState` exports the state → the board is rendered from power/board-hash values the Server and WASM must agree on.
- **UI:** `world3d.js` (the region explorer, click-to-warp, the live world-dynamics readout) · the mechanic authoring surface (`mechanics.js` + `mechanic_defs.js`) · `world.html` (a **second** composition root and a second 3D engine).
- **GAPS:** **THIS FILE IS THE ONLY AUTHORITATIVE LIST OF CLIENT-CALLABLE GO GLOBALS WHILE THE HANDLER GATE COUNTS 68 Go/WASM PUBLISHERS** — a mismatch is either an unreachable handler or a dead export, and **nothing reconciles the two** · **`handleJSCallback` is defined and NEVER used**, with a comment promising a yield to the JS event loop the body does not perform · on the `main.go` side the parity discipline is real (**`ComputeBoardHash` writes nine tiles as big-endian `uint32`s with a documented sentinel, and the replay path compares it against `frame.StateHash` and unlocks BEFORE calling `InitiateRecovery()`** — the file's best-practice moment) while its tables are not (`moodWeaknesses` and `elementalAffinity` are **re-declared per rule branch**, so the client's board is decided by tables that exist in the client only) · **`SetPlayerReady` CONTAINS A CPU DECK GENERATOR THAT MUTATES GAME STATE** — it invents "Vbabe Bot", picks a portrait with `rand.Intn`, maps the portrait PATH to a fanfare archetype **by substring**, and builds five demo cards, all inside a readiness hook · `applyDynamicStats`'s comment (*"Power = base * 1.1^(level-1), capped at UINT64 max"*) **contradicts its arithmetic** (an integer `+10 %` compounding with truncation in `int64`, and the stated cap is unreachable).
- **STATUS:** **partial** — one clean cross-boundary determinism contract, an unreconciled export table, and a readiness hook that builds card content.















### §2 Onboarding & Player Entry · §36 Session Watchdog — DERIVED
| Field | Content |
| --- | --- |
| OWNER FILES | `onboarding_service.go` (475, READ IN FULL) · `handlers_public.go` (1600+) · `first_run.js` |
| ROUTES | `/api/onboard/*` (the Voi onboarding bridge) · `/api/faucet/*` · `/api/player/*` · `/api/identity/*` |
| STATE | `onboardedWallets` (**a historical record family with a writer since 2026-09-19 and a reader before that**) · `processingOnboarding` (the per-wallet in-flight claim) · `onboardingSemaphore` (a global, 10 s acquisition timeout) · the tracked-wallet session map (lowercased keys) · `PlayerStats.SessionStartAt` · `ArenaVouchers` + the `Verified` flag on wallet links. |
| FLOW | connect → the on-chain native-VOI pre-check (≥ 1 VOI ⇒ **204, no pack**) → ONE ATOMIC GROUP of two transactions from the vault: a native-VOI payment carrying `NotePrefixOnboard+"GAS"` and an ARC-200 `transfer` of 1 $VBV carrying `NotePrefixOnboard+"TOKEN"` — **the comment NAMES it "THE ONE PERMITTED NATIVE-VOI DEPARTURE (operator adjudication 2026-09-19, plan §1)"** — with a vault reservation taken atomically and REFUNDED on failure, and `onboardedWallets[...] = true` set only AFTER a txid returns. The watchdog's 10-minute ticker checks a **24 h session limit** and audits native-VOI liquidity, calling `DisconnectClient(…, "INSUFFICIENT_LIQUIDITY")` below **100000 micro-VOI**. `HandleVoucherConversion` harvests `ArenaVouchers` only from `Verified` links, ZEROES each source as it harvests ("Integer Supremacy"), credits 1:1 and re-checks solvency. `HandleIdentityRefresh` charges a flat **100 $VBV**, routing 90 % faucet / 10 % club. |
| UI | `first_run.js` (connect → faucet claim → hub intro) · the faucet dashboard · `wallet.js`'s link flow. |
| GAPS | **Signing errors on the onboarding money path are DISCARDED** — `_, stx1, _ := crypto.SignTransaction(...)` then `SendRawTransaction(append(stx1, stx2...))`, so a signing failure submits zero-value bytes and the error is never surfaced. **A balance threshold gates session continuity** — a player under the gas floor is disconnected mid-session, an economic gate on ACCESS with no in-game notice. `HandleIdentityRefresh` discards a `ParseUint`, so any employer id not shaped `CLUB-<n>` silently yields club 0 and the 10 % share routes to club zero. FLOAT MIRRORS (`l.faucetBalance = float64(l.faucetBalanceMicro)/1e6`) are written at the reservation, the refund and the refresh. `AuditActivePlayerSessions` spawns an unbounded goroutine per wallet per audit. |
| STATUS | `partial` — the one permitted native-VOI departure is implemented and NAMED; four measured defects sit around it. |


### §5 Shop & Item Economy — DERIVED
| Field | Content |
| --- | --- |
| OWNER FILES | `shop_registry.go` (350) · `economy_processing.go` (565, READ IN FULL) · `item_shop_archetype.go` · `redemption_gateway.go` · `club_service.go`'s `HandlePurchaseItem` :2365 / `HandleRestockInventory` :2264 / `DistributeShopRevenue` :2728 |
| ROUTES | `/api/shop` (1 route, unresolved) · `/api/clubs`' shop paths · `/api/items/build` · the redemption/gateway family |
| STATE | **`GlobalShopRegistry` (:36-326, ~40 items across Elemental/Tactical/Intelligence/Justice/Underworld_Admin/Hardware) and `DLCRegistry` (:20) — BOTH EXPORTED PACKAGE-LEVEL VARS populated in `init()`, so any file may rewrite the catalogue at runtime.** `dlcRegistryMutex` guards the DLC reads (e.g. `redemption_gateway.go:79-81`). `Club.TreasuryMicro`, `ItemRegistry`. |
| FLOW | catalogue (a literal) → purchase (`HandlePurchaseItem`, **VBV-only**) → restock → `DistributeShopRevenue` → the redemption gateway against `DLCRegistry`. Every fee in the game passes through `economy_processing.go`'s ONE split function. |
| UI | `economy.js` (the club shop + `resolveShopToken`) · DISTRICT MARKET · `admin.js`'s token-preset editor. |
| GAPS | **`ShopItem.Price float64` (:26)** with `HeistSuccessModifier`/`MutationSuccessModifier` float too; the DLC registry is seeded with **PLACEHOLDER creator wallets** (`browser_creator_wallet_001`/`_002`, :340/:347) that exist nowhere in the engine, so a redemption pays a wallet no player owns; **`RequiredRole` strings are validated against no vocabulary owner**, so a renamed role silently makes an item unbuyable. In `economy_processing.go`: **every fee split is `uint64(math.Floor(float64(actualTaxPayload) * …))`** — five float computations per event through the ONE function every fee passes through — its admin-siphon AUDIT reports only the second siphon, a payout rollback can return a governor's combined multi-district payout to ONE district, and the Tax-Auditor hook still scales XP with the float wrapper the integer owner replaced. |
| STATUS | `broken` — a float price on the catalogue, a redemption paying a wallet nobody owns, and float arithmetic on the single fee-split path. |


### §4 Card Enhancement — DERIVED
| Field | Content |
| --- | --- |
| OWNER FILES | **`item_service.go` (974, READ IN FULL — "the ONE item-effect engine")** · `club_service.go`'s three mutation handlers (`HandleMutationVectorRealignment` :885, `HandleMutationMoodRecalibration` :1064, `HandleMutationLoyaltySynthesis` :1241) · `battle_service.go`'s `applyItemEffectToMatch` · `item_shop_archetype.go` |
| ROUTES | WS `use_item` (the `handleGameProtocol` case) · `/api/items*` (5 routes, all unresolved names) · `/api/items/build\|bind-nft` |
| STATE | ~45 item ids across six club categories resolved in ONE `switch` · `PlayerStats.Inventory` / `PurchasedItems` / `EquippedItems` / `PreferredItems` · the mutation bitfield + `MutationHistory` · `Certified` / `BlackMarketAdopted` provenance. |
| FLOW | `use_item` (WRITE lock held) → `applyItemEffect(id)` → per-category effect → the two self-owning helpers `applyCyberAudit` :789 and `applyDistrictScanner` :946 → mutation scars via `applyMutationScarsLocked`. The contract is stated in the file's own header: *"This function assumes the main lobby mutex is already held by the caller"* — which is why `applyItemEffect`'s self-lock-gate baseline entry is CONTRACT. |
| UI | WD ▸ Assets ▸ Inventory & Equipment (`items_equip.js`) · `club_foundry` item shop · Portfolio ▸ Deck/Progression (read-only). |
| GAPS | **The `leaderboard` is RANGED and written back inline ELEVEN times** (once per club-affecting item), each time re-deriving `Reputation` for every employee of the club — an O(club) write per item use inside the write lock. The mutation ladder's scar path is random, so it fires ~25 % of the time and never in a scripted test — which is how three self-locks hid in it until session (k)'s gate found them. Its `RequiredRole` strings are validated against no vocabulary owner. |
| STATUS | `partial` — the effect engine is complete and honestly owned; the gaps are cost-at-scale and random-path reachability, both measured. |


### §30 Counterfeit · §10 Black Market · Cyber Espionage · Sabotage — DERIVED
| Field | Content |
| --- | --- |
| OWNER FILES | `counterfeit_service.go` (335) · `black_market_service.go` (1040) · `club_service.go`'s `HandleSabotage` :1410 / `HandleRegionalSabotage` :1640 · `handlers_criminality.go` |
| ROUTES | `/api/counterfeit/generate\|detect` · `/api/black-market/*` · `/api/underworld/*` |
| STATE | **`var ActiveCounterfeits = make(map[string]CounterfeitNote) — an EXPORTED PACKAGE-LEVEL GLOBAL at :29 with NO MUTEX of its own.** Written, ranged and deleted from FOUR paths (generate, detect, seize, cleanup), protected only by the LOBBY lock where a caller happens to hold it — and being package-level it is outside every map gate the repository has. Plus a per-wallet 60 s cooldown (`counterfeitLastGen`). |
| FLOW | generate (a float-derived cost, `uint64(float64(req.Amount) * 0.05)`) → a note carrying a stored `DetectionChance float64` → detect (`rand.Float64() < noteDetectionChance` decides seizure) → seize → cleanup. The black market is the fence/sale/dutch-auction surface the sweep recorded in NINE defect groups (a nil-map inventory write panic, a `wallet[:8]` panic in the log meant to record the sale, a card silently LOST on expiry, four endpoints answering a bare array with three encoding `null`, three floats on money paths, `CalculateReputation` under the WRITE lock, a buy that bypasses the sink router, a payout with no counterparty, and a 28-id contract table duplicated as a validation `switch`). |
| UI | `criminality.js` (courthouse + bounty) · `black_market.js` (every verb toasts) · `underworld.js` |
| GAPS | **`noteID` slices the wallet by byte index with no length check** — `targetWallet[len(targetWallet)-8:]` at :94, :263, :287 — so a wallet shorter than 8 bytes **panics**, reachable from a request body. `stats.Credits` is used as the player's money (:71, :105), a field distinct from the authoritative `playerBalances`; `stats.Career == "Counterfeiter"` compares raw spellings where `RoleKey` is the owner; and the log and message label micro amounts as "micro-VBV" while formatting `float64(x)/1000000.0` — **the unit is mislabelled by a factor of 10⁶**. |
| STATUS | `broken` — a package global with no owner, a non-deterministic seizure, a request-reachable panic and a mislabelled money unit. |


### §8 Factions & Careers — DERIVED
| Field | Content |
| --- | --- |
| OWNER FILES | `career_path.go` · `career_path_handlers.go` (574) · `rival_career_engine.go` · `career.go` · `career_path_test.go` (996, READ IN FULL) · `career_role_vocabulary_test.go` (501) |
| ROUTES | `GET /api/career/path` · `POST /api/career/path/choose` · `POST /api/career/promote` · `POST /api/career/staff/request` · `GET /api/career/progress` · `/api/faction/shop(.buy)` |
| STATE | Three paths (`justice`/`criminal`/`neutral`) + four declared ALIASES (`law`→justice, `Underworld`→criminal, `Hybrid`→neutral) · the four-valued path relation (`direct`/`interpreted`/`shared`/`none`) · `CivilTier` `user`/`manager`/`governor` (DERIVED from owned clubs and territories — never declared) · `PromotedRoles` + `JobRole` (the grant) · `CareerXP.RoleXP` keyed by role · the level-cap unlock record · `Club.StaffUpgradeRequests`. |
| FLOW | ≥2 territories **and** a region open → the path choice, made ONCE → promotion (level cap + role tier + sustained $VBV + civil rank + path, each refusal naming the ONE missing piece) → `warn → grace (7 days) → demote` on a drained balance, with `cleared` on recovery and `NotifyCareerStandingLocked` addressing the ONE wallet. |
| UI | `career_tree.js` (renders the SERVED payload and re-declares nothing — the client-invented `faction: JUSTICE|UNDERWORLD|HYBRID` vocabulary and ~40 dead SCSS rules were removed) · `faction_shop.js` · the "Upgrades — unlocked, not forced" section. |
| GAPS | **`PromotedRoles` had NO writer before `career_path.go`** (so `CareerHasRole` was permanently false and half the rival system could not fire); the vocabulary had no owner (`RoleKey` now folds eleven spellings, with FIVE unrelated pairs pinned NOT to collapse); one pair was declared BOTH antagonistic and synergistic; and `rivalMatrixEnemyPairs` names **P2-D10 as antagonistic while the XP switch prices it 0**. The path and career matrices are compared in BOTH directions by test (enemy ⇒ cross paths, otherwise ⇒ share), so they cannot drift. |
| STATUS | `read` — the row is derived from four owner entries and two full test reads; the one live gap is the unpriced pair, which is an operator XP call. |


### §15 Pet World & Entity Events · §29 Vehicles & World Content — DERIVED
| Field | Content |
| --- | --- |
| OWNER FILES | `entity_event_engine.go` (849, READ IN FULL) · `asset_life_engine.go` (688, READ IN FULL) · `event_scheduler.go` |
| ROUTES | `/api/entity-events/regions\|host\|resolve` · `/api/owner/combined-stats` · `…/adopt\|reclaim\|orphan-status` · `/api/faith/coherence\|faith-war-gambit` · `/api/pets*` · `/api/vehicles*` |
| STATE | The integer STAT vector (`StatMax` 100, `EventStatGain` 2, `QuitStatPenalty` 4, `PowerOverlayScale` 50) · `EntityPowerOverlay` · `EntityEvent`/`EntityEventResult` · `OwnerRelationGraph` (own mutex) · `EntityEventEngine` (own mutex) · §31 constants (`OrphanGraceDays` 90, `AlimonyCommissionBps` 500, `SynergyWeightFloor` 10, `InverseGapBonus` 3). |
| FLOW | host (tier-gated) → the event trains/rewards/punishes → `ApplyEventResult`/`applyStatDelta` → `ComputeEffectivePowerLevel` caps the entity's effective level in the 3D world → `ResolveEntityEventPayout` credits owners → `BuildRegionPowerOverlay` (pets + vehicles + bots) is what `world3d.js` renders. |
| UI | `world3d.js` (region towers scale + glow by `avg_entity_power`) · `pet_breeder.js` (Kennel, SERVED prices) · `life_assets.js` (Garage) · `pet_battle_arena.js` (honest placeholders) · Portfolio ▸ Companions. |
| GAPS | **THREE mutexes, each owned by its own object and never nested** — the file does the lock discipline CORRECTLY, so the gap is a contract stated nowhere: `ProcessEntityEvent` (:457-476) and `ResolveEntityEventPayout` (:493) mutate through live pointers (`c.Stats`, `pet.PetLevel`, `l.playerBalances`) and hold **NO lobby lock of their own**, relying on the caller. Plus: **a balance is read and written with no case resolution**. The §31 sibling `event_scheduler.go` holds the sweep's ONLY non-deterministic producers (a goroutine + `time.Sleep` deciding an event). |
| STATUS | `partial` — integer-clean arithmetic and correctly owned locks; the gaps are an unstated caller contract and a case-sensitive balance access. |


### Clubs, Alliances & Territory — DERIVED
| Field | Content |
| --- | --- |
| OWNER FILES | **`club_service.go` (3315 — 30 declarations, the largest single service in the tree)** · `region_handlers.go` · `economy_bootstrap.go` |
| ROUTES | `/api/clubs` · `/api/regions` · the WS club/heist/sabotage/alliance/lease/mutation cases in `handleGameProtocol` |
| STATE | `Club` (`Territories`, `RegionName`, `Staff`, `TreasuryMicro`, `StaffUpgradeRequests`, `RitualsDone`) · `PlayerStats.EmployerClubID` · `l.clubs` (a plain Lobby map) · the router's `RegionalDistricts`. |
| FLOW | `HandleCreateClub` → `HandleJoinClub` → `HandlePurchaseTerritory` (2,500 $VBV on-chain, auto-governor at the 2nd) → `HandleOpenRegionalManager` → restock → `DistributeShopRevenue` / `DistributeTournamentKickback` → alliances invite/accept/dissolve → leases create/take/expire → mutations (vector realignment, mood recalibration, loyalty synthesis). |
| UI | Club foundry surfaces · WD ▸ Governance ▸ Territory · Portfolio ▸ Clubs & Alliances (Alliance Links + Territory Tiles) · `religion_governance.js`. |
| GAPS | Two offered by the §16 tournament path inside the same service: **the arena club's treasury is OVERWRITTEN FROM A ROUTER NODE** — `centerClub.TreasuryMicro = node.TreasuryBalance` (:674-677) DISCARDS the club's own accounting, with a `ParseUint` error discarded beside it — and **the history read is page-limited to 1000 transfers with no pagination** (:265), so the checksum "verifies" a partial history and reports success for what it saw. `DetermineTop5` is intentionally unequal and says so. |
| STATUS | `partial` — the domain is complete and owned; both recorded gaps sit in its tournament path. |


### §39 The Civilization Flywheel — DERIVED (the composition of the rows above)
| Field | Content |
| --- | --- |
| OWNER FILES | `industrial_loop.go` · `economy_processing.go` (the ONE fee split every fee passes through) · `token_sink_router` (`RouteCriminalTax` + the ledger) · `faucet_service.go` · `reward_registry.go` · `economy_bootstrap.go` · `record_families.go` (**the 22 families that make the world survive a restart**) |
| ROUTES | `/api/industrial-loop/*` · `/api/faucet/*` · `/api/reward*` · `/api/admin/networks` · every priced door in the rows above |
| STATE | The four token-sink audit counters + `industrialLoop`'s phase counters · `faucetBalanceMicro` (authoritative) beside `faucetBalance` (a float mirror) · `rewardStack`/`rewardTokens` (the registry, reconciled from the env primary) · the 22 primary record families and the 5 derived rows that name a source that exists. |
| FLOW | **the loop the app is priced from:** every fee (`economy_processing.go`) → the token-sink router's ledger → sink shares → the faucet → the reward registry's payouts → back into the priced doors (items, pets, vehicles, territory, leases, bonded assets, placeholders) → and every movement mirrored to the chain by the record rail, whose reader is ONE function for all three callers. |
| UI | `portfolio.js` ▸ Economy (read-only) · `industrial_loop.js` · `dividends_*.js` · the admin console's reward registry. |
| GAPS | **THE FLYWHEEL IS TURNED BY A CLIENT-DECLARABLE COUNTER:** `handleIndustrialLoopRecord` accepts **any phase and any amount** into the loop's counters — the counters the flywheel is priced from — with no authority check. Its sibling gap is the same class in four more places (`PerformRitual`'s body-supplied `cost`, `handleGovernanceVote`'s `Weight`, `handleIdentityRecord`'s `impact`, `handleUnlockAchievement`'s wallet). And the class the whole sweep kept finding sits UNDER the flywheel: **eleven package-level globals own recorded state outside the Lobby and outside every record family** (`persistentIdentity`, `industrialLoop`, `creatorEconomy`, `faithChurchEngine` + its second `globalLobbyRef`, `complianceEngine`, `entityShares`, `launchEngine`, the entity-market trio, `gamingOSEngine`, `governanceEngine`, `replayEngine`), so a restart loses what the record rail was built to keep. |
| STATUS | `partial` — the flywheel's ARITHMETIC is largely integer-clean and its rail is complete (506/506 corpus read, 22/22 families written); its INPUTS are client-declarable and its state ownership is the sweep's largest single architectural gap. |

### §37 Composable Mechanic Framework — DERIVED (the LAST row; two files read in full)
| Field | Content |
| --- | --- |
| OWNER FILES | **`Public/js/mechanics.js` (165, READ IN FULL)** · **`Public/js/mechanic_defs.js` (96, READ IN FULL)** · `Public/js/world3d.js` (the only importer: `import { MechanicEngine, microToTint, addMicro } from './mechanics.js'` + `import { DEFAULT_MECHANICS } from './mechanic_defs.js'`) |
| ROUTES | **NONE — this is the only aspect with no route at all.** It is a pure client library: the engine is INSTANTIATED IN THE BROWSER (`world3d.js` builds a `MechanicEngine` and registers the five defaults). |
| STATE | `Mechanic` (id · kind ∈ vitality\|mojo\|aura\|gravity\|tax\|custom · scope ∈ region\|club\|player\|world · integer `weight` · `repeatable` · `stackable` · `rivalryTag` · `oncePer`) · `MechanicEngine.mechanics` (id → Mechanic) · `firedOnce` (oncePer key → Set of scope keys) · `rivalry` (tag → accumulated micro bigint) · `MICRO = 1_000_000n` · five shipped defs (`region_vitality`, `club_mojo`, `rivalry_aura`, `theme_gravity`, `career_ripple`). |
| FLOW | `register(def)` → `resolve(scope, scopeKey, scopeState, ctxMeta)` walks the mechanics, applies the `oncePer` dedupe, seeds each with `hashU64(id + '\|' + scopeKey)`, calls the pure `compute(ctx)` → a **bigint micro** delta → `delta * weight` → summed into `totalMicro` with an integer per-mechanic `breakdown` and a per-`rivalryTag` accumulation → `accumulateRivalry()` folds it into the engine's own matrix → `getRivalryMatrix()`. **`toMicro` THROWS on a non-integer** (`'toMicro: non-integer value violates uint64 ledger rule'`) — the framework's best property and the model this repository keeps asking for elsewhere. `microToTint` is the ONE float, derived from an integer ratio and labelled VIEW ONLY. |
| UI | None of its own: its numbers tint and scale the §25/§27 3D world. |
| GAPS | 1. ~~**THE DETERMINISM SEED HASHED AT MOST ONE CHARACTER**~~ — **FIXED 2026-09-20.** The offending line was `for (let i = 0; i < seedStr?.length ?? 0; i++)`, which `??`-precedence evaluates as `(i < seedStr.length) ?? 0` — a BOOLEAN coerced to 1 or 0 — so the loop body ran for `i = 0` only (and an empty seed returned the bare FNV offset basis), colliding every id sharing a first character: the `??`-precedence class the JS sweep found in eight other client modules, landing here on the SEED. The parenthesis is now explicit and the reason is written above the function. **PROVEN by 16 assertions that IMPORT the real modules** (a last-character difference is seen, a post-first-character difference is seen, `''`/`undefined`/`null` return the offset basis, determinism across 2000 calls, and the engine's per-id seeds now differ). **SHIPPED IMPACT, MEASURED BEFORE THE CHANGE: none** — no definition in `mechanic_defs.js` reads `ctx.seed` (all five read `ctx.meta.*`), so no shipped number moved; what was broken is the contract a developer registers against. 2. **A COMMENT STATING A RULE THE CODE DOES NOT IMPLEMENT:** the non-`stackable` branch says *"replace: keep the max absolute contribution"* and then does neither — it zeroes a negative `total` and adds, so a non-stacking negative contribution is silently swallowed. 3. `firedOnce` is never pruned — one Set per `oncePer` key grown for the life of the page. 4. ~~**THE PARITY CLAIM IS PROSE ONLY**~~ — **CORRECTED 2026-09-20, not "fixed", and the difference is stated in the file.** The header claimed the defs *"intentionally mirror server-side concepts … So the 3D world's emergent numbers agree with the authoritative simulation"*, while the mirrors are hand-copied constants (`50000n`, `120000n`, `/100n`, `/10n`) that **nothing compares** with `ai_citizen_engine.go`'s region cap, `club_service.go`'s mojo or `theme_engine.go`'s `W_*` weights. The header now says exactly that: the agreement is asserted by nobody, `cap` is passed and read by no compute, `gravityMicro` is called a "future server field" by the definition itself, and a client number is a UI-derived estimate with the server as the authority. **The comparison that would make the claim true is still NOT BUILT** — it is a gate (client literals vs the Go constants that own them) and is recorded here rather than implied. 5. `M_THEME_GRAVITY` reads `ctx.meta.gravityMicro`, which the file's OWN comment calls a *"future server field"* — so it is 0 until a server field exists; and `region_vitality`'s documented `cap` is passed but never read. 6. `M_CLUB_MOJO`'s ternary `(mojo / 100n) > 0n ? (mojo / 100n) : 0n` is a no-op (the alternative is already 0). |
| STATUS | `partial` — integer-clean by construction with a fail-closed `toMicro`, and REACHABLE (composed through `world3d.js`, which `verify:reachability` derives). **The seed defect is FIXED and behaviourally pinned (2026-09-20, 16 assertions importing the real modules); the false parity claim is CORRECTED in the file.** Its remaining gaps are the comment that contradicts the non-`stackable` branch, `firedOnce` never being pruned, and a server-parity claim no test holds. |


## ASPECT COVERAGE — EVERY ASPECT CARRIES A ROW (the close, made machine-checkable)

> **Rule:** every numbered aspect of `AI-Brain/App-Aspect-Index.md` names the derived row that covers it.
> The `title` column is the index's own wording, verbatim, and the `row` column is the PREFIX OF A REAL `### ` HEADING in this file. `npm run verify:aspects` (`tools/server/verify_aspect_rows.js`) fails on a missing aspect, an invented one, title drift, a stale/renamed row heading, and on its own parser finding nothing — so **a new aspect cannot arrive unrowed**, and a row that is renamed must update its mapping in the same change. A row may cover more than one aspect (the row value repeats); where the index splits an aspect the rows file kept as one cluster, that is stated here rather than hidden.

| # | Aspect (index title, verbatim) | Derived row (heading prefix in this file) |
|---|---|---|
| §1 | Architecture | Multi-Chain Bridge & the funding rails |
| §2 | Onboarding & Player Entry | §2 Onboarding & Player Entry · §36 Session Watchdog |
| §3 | Battle System (Core Gameplay) | Battle, QuickPlay & Matchmaking |
| §4 | Card Enhancement (NFT Upgrades) | §4 Card Enhancement |
| §5 | Shop & Item Economy | §5 Shop & Item Economy |
| §6 | QuickPlay | Battle, QuickPlay & Matchmaking |
| §7 | Matchmaking & Multiplayer | Battle, QuickPlay & Matchmaking |
| §8 | Factions & Careers | §8 Factions & Careers |
| §9 | Justice System | Justice & criminality |
| §10 | Underworld & Criminality | §30 Counterfeit · §10 Black Market |
| §11 | AI Citizens | AI Citizens |
| §12 | Faith System | Faith System |
| §13 | Theme Engine | Theme & bonded branding |
| §14 | Entity Markets | Entity Markets |
| §15 | Pet World & Entity Events | §15 Pet World & Entity Events · §29 Vehicles & World Content |
| §16 | Tournaments | Tournaments |
| §17 | Seasonal Events | Seasonal Events, Treasure & the Region Views |
| §18 | Achievements & Trophies | Achievements & Trophies |
| §19 | Auctions | Auctions |
| §20 | Spectator Mode | Spectator & Replay |
| §21 | Governance | Governance |
| §22 | Creator Economy & DLC Store | Creator Economy & DLC Store |
| §23 | Multi-Chain Bridge | Multi-Chain Bridge & the funding rails |
| §24 | Industrial Loop | Industrial Loop |
| §25 | Persistent Identity | Persistent Identity |
| §26 | Local LLM Pipeline | Local-LLM pipeline |
| §27 | Web-3D Client | Web-3D Client & the WASM boundary |
| §28 | Item Shop Archetype (Player-Created Items) | Item Shop Archetype |
| §29 | Vehicles & World Content | §15 Pet World & Entity Events · §29 Vehicles & World Content |
| §30 | Counterfeit System | §30 Counterfeit · §10 Black Market |
| §31 | Admin Tools | Admin Tools |
| §32 | Dev/Game Hub (Composable Framework) | Composable Framework |
| §33 | Nautilus DEX Path (Console Creator Payouts) | Console Linking & the Console Target |
| §34 | Console Linking & Manufacturer APIs | Console Linking & the Console Target |
| §35 | Economy Bootstrap & State Recovery | Economy Bootstrap & State Recovery |
| §36 | Session Watchdog & Player Eviction | §2 Onboarding & Player Entry · §36 Session Watchdog |
| §37 | Composable Mechanic Framework | §37 Composable Mechanic Framework |
| §38 | Infrastructure & Security | The lock & data-race gate family |
| §39 | The Civilization Flywheel | §39 The Civilization Flywheel |

## REMAINDER — EMPTY, and held empty by a gate

**Every one of the 39 aspects now carries a derived row, so this table has no rows.** It is kept as a HEADING rather than deleted, because an absent remainder is itself a claim a reader should be able to find — and because the way it stays empty is mechanical: `npm run verify:aspects` fails if the aspect index gains an aspect the coverage table does not name, so the next session is told by a failing build rather than by remembering.

**What would put a row back here:** a NEW aspect added to `AI-Brain/App-Aspect-Index.md` (the gate names it), or a row heading renamed without updating its mapping (the gate names the stale prefix). Both are one-line fixes in the coverage table above.


## MEASURED ASPECT SOURCES (all three admissible sources, each confirmed to exist)

| # | Source | Measured |
|---|---|---|
| 1 | **The route table** — `server_main.go` + `console_server.go` (`mux.HandleFunc`) | **326** distinct routes in `server_main.go` · **140** in `console_server.go` · **189 routes exist in `server_main.go` ONLY** · **254** `handle*` definitions resolve across 154 non-test Go files |
| 2 | **The World Dashboard taxonomy** — `const WD_CATEGORIES`, `Public/js/world_dashboard.js` | **11** categories: player (Player Hub) · careers (Careers & Factions) · assets (Assets) · economy (Economy & Trade) · governance (Governance) · play (Play & Board) · world (World & Events) · faith (Faith & Church) · competition (Competition) · creator (Creator Economy) · system (System & Ops) |
| 3 | **The repository own aspect index** — `AI-Brain/App-Aspect-Index.md` | **39 numbered aspects** (§1 Architecture … §39 The Civilization Flywheel), with named sub-aspects (Mutation, Artifact Enhancement, Loyalty, Fatigue, Kidnap Gambit, Underworld Contracts, Black Market, Cyber Espionage, Sabotage, Faction Assignment, Justice Card Power Bonus, Career Roles, Career XP, Item Categories, Shop Gates, Item→Battle Effects) |

> **A FOURTH CANDIDATE WAS MEASURED AND REJECTED AS A SOURCE:** the `OWNS:` field of the per-file entries is **PROSE**, so matching it by keyword produces FALSE owners — measured over the 409 `OWNS` records: `identity` matched **59** owner-refs across unrelated files, `economy/ledger` **43**, `frontend/nav` **101** (because the regex matched the words route/dashboard/overlay inside ordinary prose). **Keyword-matching prose is not an ownership derivation.** The `OWNS` field is the right INPUT, but it must be read and understood per entry, which is why the rows below are marked with the evidence they rest on.

## THE ASPECT SPINE — route prefix -> owner files (MACHINE-DERIVED, so it cannot be an opinion)

Derived by: parse every `mux.HandleFunc("<path>"` in `server_main.go` -> read the `handle*` name on that line -> find the non-test `.go` file that DEFINES it. `un=` is the count of routes whose handler name is not defined as a top-level `func handle*` (a method on another receiver, or a name that does not follow the convention) — **counted, never guessed**. This table is the measured OWNER-FILES input for every aspect the route table covers.

| route prefix | routes | un | owner files (measured) |
|---|---:|---:|---|
| `/api/(root)` | 10 | 10 |  |
| `/api/achievement` | 1 | 0 | achievement_handlers.go |
| `/api/achievements` | 1 | 0 | achievement_handlers.go |
| `/api/achievement-stats` | 1 | 0 | achievement_handlers.go |
| `/api/ad` | 6 | 0 | advertising.go |
| `/api/admin` | 28 | 0 | handlers_admin.go · network_registry_view.go |
| `/api/ads` | 3 | 0 | advertising.go |
| `/api/ai` | 12 | 8 | ai_citizen_engine.go |
| `/api/assets` | 21 | 0 | bonded_market_service.go · placeholder_assets.go · bonded_asset_registry.go · slide_theming.go · bonded_branding.go · card_view_skins.go · ui_tree_theming.go |
| `/api/auctions` | 1 | 1 |  |
| `/api/ban-player` | 1 | 0 | handlers_admin.go |
| `/api/black-market` | 4 | 4 |  |
| `/api/bounty` | 1 | 1 |  |
| `/api/bridge` | 6 | 1 | bridge_router.go |
| `/api/card-details` | 1 | 0 | handlers_public.go |
| `/api/card-stats` | 1 | 0 | handlers_public.go |
| `/api/career` | 5 | 1 | career_path_handlers.go |
| `/api/children-bots` | 1 | 0 | entity_market.go |
| `/api/church` | 12 | 1 | faith_church.go |
| `/api/claim` | 1 | 1 |  |
| `/api/client-error` | 1 | 1 |  |
| `/api/clubs` | 1 | 1 |  |
| `/api/compliance` | 6 | 0 | gaming_os.go |
| `/api/contracts` | 2 | 0 | server.go |
| `/api/counterfeit` | 2 | 2 |  |
| `/api/courthouse` | 1 | 1 |  |
| `/api/creator` | 19 | 10 | creator_economy.go |
| `/api/criminality` | 1 | 0 | server.go |
| `/api/dividends` | 1 | 0 | handlers_dashboard.go |
| `/api/entity` | 3 | 0 | entity_market.go |
| `/api/entity-event` | 2 | 0 | entity_event_engine.go |
| `/api/entity-events` | 1 | 0 | entity_event_engine.go |
| `/api/envoi-name` | 1 | 0 | handlers_public.go |
| `/api/events` | 2 | 2 |  |
| `/api/faction` | 1 | 1 |  |
| `/api/faith` | 9 | 0 | entity_event_engine.go · religion_governance.go |
| `/api/faucet` | 3 | 0 | handlers_public.go |
| `/api/garden` | 3 | 3 |  |
| `/api/governance` | 7 | 0 | governance.go |
| `/api/identity` | 8 | 4 | persistent_identity.go |
| `/api/industrial-loop` | 3 | 0 | industrial_loop.go |
| `/api/invest` | 3 | 3 |  |
| `/api/items` | 5 | 5 |  |
| `/api/justice` | 8 | 5 | server.go |
| `/api/launch` | 6 | 0 | launchpad.go |
| `/api/launches` | 1 | 0 | launchpad.go |
| `/api/leaderboard` | 1 | 0 | handlers_public.go |
| `/api/lease` | 3 | 0 | infrastructure_lease.go |
| `/api/loans` | 3 | 3 |  |
| `/api/local-model` | 2 | 0 | local_model_promotion.go |
| `/api/maintenance-mode` | 1 | 0 | handlers_admin.go |
| `/api/market` | 1 | 0 | theme_engine.go |
| `/api/match` | 2 | 0 | handlers_public.go |
| `/api/moods` | 2 | 2 |  |
| `/api/notes` | 1 | 0 | handlers_public.go |
| `/api/orphan` | 3 | 0 | entity_event_engine.go |
| `/api/os` | 5 | 0 | gaming_os.go |
| `/api/owner` | 1 | 0 | entity_event_engine.go |
| `/api/pet-battle` | 3 | 0 | entity_market.go |
| `/api/pets` | 4 | 4 |  |
| `/api/player` | 4 | 1 | handlers_public.go · theme_engine.go |
| `/api/players` | 1 | 0 | handlers_public.go |
| `/api/refill-vault` | 1 | 0 | handlers_admin.go |
| `/api/regions` | 1 | 1 |  |
| `/api/replay` | 7 | 0 | replay_engine.go |
| `/api/report-player` | 1 | 0 | handlers_admin.go |
| `/api/reset-stats` | 1 | 0 | handlers_admin.go |
| `/api/re-sync-stats` | 1 | 0 | lobby_manager.go |
| `/api/reward` | 5 | 0 | handlers_admin.go · faucet_service.go |
| `/api/rewards` | 1 | 0 | handlers_dashboard.go |
| `/api/rivalry` | 10 | 7 | theme_engine.go · entity_market.go |
| `/api/rumors` | 1 | 1 |  |
| `/api/season` | 8 | 7 | lobby_manager.go |
| `/api/shares` | 4 | 0 | entity_shares.go |
| `/api/shop` | 1 | 1 |  |
| `/api/stat-overlay` | 4 | 0 | stat_overlay.go |
| `/api/system-message` | 1 | 0 | handlers_admin.go |
| `/api/tea` | 2 | 2 |  |
| `/api/theme` | 3 | 0 | bonded_asset_registry.go · theme_engine.go |
| `/api/titles` | 2 | 2 |  |
| `/api/tournament` | 2 | 2 |  |
| `/api/treasure` | 2 | 2 |  |
| `/api/underworld` | 3 | 1 | handlers_public.go |
| `/api/update-rules` | 1 | 0 | handlers_admin.go |
| `/api/v1` | 1 | 0 | redemption_gateway.go |
| `/api/vehicles` | 4 | 4 |  |
| `/api/wagers` | 2 | 2 |  |
| `/api/world-content` | 3 | 3 |  |
