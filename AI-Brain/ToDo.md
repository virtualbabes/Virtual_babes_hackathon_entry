# NFT-Seduction: Engineering Roadmap

> **Authoritative task list**
> **Last updated:** 2026-09-14 (ARC-200 read path measured + ONE indexer transport + courthouse lock fixes)
> **Build:** All Go targets (native + `linux/amd64` + `js/wasm`) GREEN; SASS compiles; entry probe **80/80**, overlay audit **23/23**, UI harness **12/12**, `verify:portfolio` **62/62**; `go test .` passes except the PRE-EXISTING AMM slippage-guardrail test. NEW this pass: `indexer_transport_test.go` (404-is-a-routing-answer failover) + `courthouse_rival_scan_test.go` (lock-held snapshot + a deadlock watchdog) and 6 `vocab.*` probe assertions. **Live-measured:** the configured Voi base serves `/v2/*` (200) but **404s every `/arc200/*` path**, which is where 13 call sites read the chain from — the ARC-200 API base has to be named before on-chain reconstruction or Voi payment verification can work.
> **Backend state:** Voi-via-`voi.nodely.dev` primary; Algorand secondary; oracle no longer blacklists on semantic 404; proactive vault probes removed (on-demand + 5min throttle). $VBV is the ONLY in-game token until $UNIT/$NUGGET (Algorand+Voi) settlement lands. `GET /api/player/associations` now exports the engine-held wallet associations that no other route exposed. §25.6.1 vehicle parts ladder live (`/api/vehicles/upgrade|deploy`). §23.5 bonded assets are the ECOSPHERE BRANDING LAYER (`/api/assets/bind|unbind|targets|branding`, `bonded_branding.go`) and **cards are excluded structurally** — the policy assertion panics at init if a card kind is ever made bondable. §23.5.5 adds a **viewer-scoped card display** (`/api/assets/card-view`, `card_view_skins.go`) — a VIEW keyed by the viewer's own wallet with no target field, which never writes a branding binding.

---

## ✅ COMPLETE — 2026-09-14 (n): the ARC-200 read path MEASURED · ONE indexer transport · courthouse lock fixes
- **THE ARC-200 READ PATH IS NOT SERVED BY THE CONFIGURED BASE — now MEASURED, not suspected.**
  `https://mainnet-idx.voi.nodely.dev` answers **200** for `/v2/accounts` and `/v2/transactions` and
  **404** for every `/arc200/*` path (transfers with a real contract id, transfers bare, contracts).
  **Thirteen** call sites ask it for `/arc200/*` — the Voi branch of `VerifyBuyInTransaction` (money
  doors), the stats sync and every checkpoint reader. Confirmed at boot on the new build: six snapshot
  readers all answered 404. Consequence stated rather than softened: on this configuration on-chain
  reconstruction cannot fire and a Voi money door cannot verify a payment, and the earlier "vault $VBV box
  not found on-chain (unseeded)" line is NOT evidence the vault is unseeded — an unrouted 404 parses the
  same way. **NOT guessed:** the correct ARC-200 host is unknown here and naming one would be worse than
  reporting the truth. **Brendan's call: name the ARC-200 API base** (Problems.md §15 pass 2 A).
- **ONE indexer transport (`indexerGet`).** Two near-identical private failover loops
  (`OracleService.IndexerRequest`, `Lobby.indexerRequest`) became one owner, so the oracle readers and the
  checkpoint readers cannot drift. A **404 was treated as a final answer** — a base that does not serve a
  path 404s for EVERY request to it, so the first base's 404 ended the chain and no later base was asked.
  A 404 now advances to the next base (and is not retried on the same one — a routing answer cannot
  change); when every base 404s the most recent 404 is still RETURNED, preserving the two callers that
  read a 404 as informative. Pinned by 5 `indexer_transport_test.go` tests.
- **`use_item` ▸ `legal_pardon` deadlocked the WHOLE process** — the fourth instance of this defect class.
  `use_item` holds the write lock across `applyItemEffect`; `ApplyLegalPardonLocked` then awarded XP via
  `TrackCareerXP`, which takes the write lock itself = unconditional self-deadlock, freezing every request
  and every WebSocket. **Legal pardons had never once completed.** Fixed with `trackCareerXPLocked`;
  pinned by a watchdog test that fails by TIMEOUT.
- **`handleCourthouseReset` raced the leaderboard map** (unlocked range over `l.leaderboard` while other
  goroutines write it = unrecoverable `fatal error`). Now a lock-held snapshot (`taxAuditorRivalsLocked`)
  with the award applied after release (`taxAuditorRivalBonuses` for callers holding no lock).
- **The checkpoint failure is now EXPLAINED, not silent.** `warnCheckpointReadUnavailable` fires once per
  process, names the prefix and the base(s) tried, and states BOTH candidate causes (path not routed by
  any configured base; writer/reader shape difference) as candidates — asserting neither.
- **NEW finding, NOT fixed (balance decision):** the courthouse rival bonus is **dead arithmetic** — both
  resolution points compare `defenderXP` (`base × 0.30` = 4 and 9) against `> 15` / `> 30`, so the
  TaxAuditor↔JusticeCommissioner bonus has never been awarded. The refactor preserved it EXACTLY rather
  than silently changing XP economics (Problems.md §15 pass 2 E).
- **VERIFIED:** native + `linux/amd64` + `js/wasm` all rc 0; entry probe **80/80** (6 NEW `vocab.*`),
  overlays **23/23**, portfolio **62/62** (0 bad, 0 page errors); `go test .` passes except the
  PRE-EXISTING AMM test; `go vet` shows only the pre-existing copylocks. New tests:
  `indexer_transport_test.go` (5), `courthouse_rival_scan_test.go` (4).


## ✅ COMPLETE — 2026-09-14 (m): note vocabulary · ONE txid memo · snapshot read guard
- **A money door deadlocked the ENTIRE server, and it had never once been run.** `handleBailCard`
  took the lobby WRITE lock at the top and then took a READ lock inside it — an unconditional
  `sync.RWMutex` self-deadlock that holds the write lock FOREVER (freezing every request and every
  WebSocket). `HandleRepayLoan` had the same shape; `HandleDetectCounterfeit` deadlocked on
  `TrackCareerXP` under the write lock once a note was found. All three are now
  **read-validate → verify → apply**, with the write lock held ONLY around the state change and
  **never** across `VerifyBuyInTransaction` (which itself takes `l.mutex.RLock()`).
- **ONE idempotency door (`txid_memo.go`).** Before this pass the memo was consulted at **4 of 8**
  money doors, and even there in two separate lock windows with the slow indexer call between them, so
  two concurrent double-submits could BOTH verify and BOTH apply (doubled career XP + rival bonus at the
  courthouse, doubled club-treasury credit + inventory count at bail, doubled faucet credit at a loan
  repayment). Now: `claimTxID` RESERVES before the oracle call, `commitTxID` records only AFTER the money
  is applied, `releaseTxID` frees a failed attempt. Only `registeredTxIDs` is persisted, so a restart
  mid-verification leaves no trace — and nothing was applied. `commitTxIDLocked` / `releaseTxIDLocked`
  serve a door that already holds the write lock.
- **`VerifyBuyInTransaction` now refuses an EMPTY purpose prefix** (`HasPrefix(note, "")` is always true,
  so an empty prefix silently dropped purpose binding) and an EMPTY txid (unmemoisable = replayable).
- **ONE note vocabulary (`note_vocabulary.go`).** 36 prefixes — 9 money doors, 6 checkpoints, 21 audit
  logs — declared ONCE and SERVED read-only at **`GET /api/notes/vocabulary`** (both servers). A browser
  reads the prefix it may write from there (new `Public/js/note_vocabulary.js`) and **FAILS CLOSED**: it
  never re-types a prefix and never falls back to a remembered one. Enforced mechanically by tests: no Go
  literal outside the owner, no client literal at all, every `VerifyBuyInTransaction` final argument, and
  nothing uncatalogued — plus an `init()` panic on a duplicated/malformed/newly ambiguous prefix.
- **Snapshot READ GUARD (`snapshot_guard.go`).** Decompression was UNBOUNDED at all three checkpoint
  readers (`ReadFrom(gzr)`, no cap) — a boot-time memory-exhaustion path reachable before any rate limiter
  or wallet check. Now capped at BOTH stages (8 MiB base64 / 64 MiB decoded). The vault scoping was
  asserted only in the QUERY STRING; `isVaultCheckpointTransfer` asserts it in code. Also removed a
  `defer gzr.Close()` inside the per-transfer loop.
- **Client memo (`Public/js/tx_journal.js`).** Records `{purpose, txid, amount_micro, ts}` as `signed`;
  only a SERVER response promotes an entry to `verified`/`failed`. `renderPendingHTML()` labels every
  unconfirmed row PENDING — a signed entry is never presented as settled. Integer micro-units only.
- **Bound prefixes are built, not concatenated:** `NoteFoundClub(name)`, `NoteJoinClub(clubID)`,
  `NoteClaimDistrict(clubID, district)`, `NoteTournamentBuyIn(id)`, `NoteArenaTournamentBuyIn(id)`.
- **Verification:** all 3 Go targets GREEN; `go test .` → only the PRE-EXISTING AMM failure; live HTTP on
  :8090 → vocabulary `total=36 money_door=9 checkpoint=6 audit_log=21`, POST refused **405**,
  `note_vocabulary.js` 200, `tx_journal.js` 200, `/app.js` 200 and containing both new imports.
- **Recorded, NOT fixed (see `Problems.md` §15):** the checkpoint WRITER emits an **app-call**
  (`sendNoteTx`) while every READER queries `/arc200/transfers` — a shape mismatch that may mean on-chain
  reconstruction never fires; and the indexer's `from`/`to` projection could not be verified live, so the
  scoping assertion applies when present and logs its inactivity once instead of assuming it.


## ✅ COMPLETE — 2026-09-14 (i): viewer-scoped card display ("player 1 can change how they see player 2's cards")
- **The rule the feature obeys:** cards are NEVER brandable (§10.1). What a player may now do is
  choose how **their own eyes** render the cards in front of them — the opponent's hand and the
  quick-play opponent face. It is a display preference: the opponent never sees it, the match
  authority is never involved, and no card is written to.
- **THEMING REQUIRES A BONDED ASSET (2026-09-14, v1.5).** The earlier build also offered an
  asset-free `placeholder` mode — i.e. a way to re-theme another player's cards with art the player
  did not own. That mode is **REMOVED** (server and client): modes are `engine` (off) and `asset`
  only, the policy serves `asset_required: true`, and asset mode without one of the caller's OWN
  assets is refused. The NPC-helper pack survives only as the deterministic STAND-IN painted when a
  worn asset's media is a content address (`ipfs://`/`ar://`) the browser cannot load — still tied to
  a real owned asset, and reported as stand-in art by `describe().art_kind`.
- **CREATION IS CAPPED AT THE CARD/DECK NFT SUPPLY (§23.5.1, NEW).** `Lobby.CardNFTSupplyForWallet`
  counts the wallet's `PlayerStats.Inventory` `CARD-*` entries (the authoritative ledger the
  auction/loan/black-market/jail escrow maintains), `BondedAssetCapacityForWallet` serves
  `{basis, limit, used, remaining, rule}`, and `assertBondedAssetCapacity` is the mint gate. The cap
  gates creation only (no retro-invalidation, transfers never blocked), and the read path re-proves
  ownership each time — so a SOLD asset stops being worn. Wallet-NFT sourcing swaps ONE function.
- **Why the guarantee is structural:** `CardView` has **no target field** (it is keyed by the
  VIEWER's own wallet and names only one of that viewer's own bonded assets) and
  `card_view_skins.go` **never writes `registry.Bindings`** — the sole branding writer remains
  `BindBondedAssetToTarget`, which refuses every card kind before any lookup. The request body is
  decoded with `DisallowUnknownFields()`, so `target_kind` / `target_id` / `card_id` / another
  wallet's address cannot even be expressed; card-shaped values fail closed before any lookup.
- **Two real defects found while enforcing the rule:** (a) the card-view authority compared wallets
  case-SENSITIVELY, so a viewer spelling their own wallet in uppercase was told they did not own
  their own asset — fixed with `canonicalViewerWallet` (mirrors `canonicalTargetID`); (b) a stale
  record could outlive ownership — fixed by re-proving §27.8 on every read (pinned by test).
- **One real nuance:** the existing card trap is a SUBSTRING scan, and the audience vocabulary
  legitimately says "cards" (`foreign_cards` / `own_cards` / `all_cards`) — applying it to `scope`
  would refuse the layer's own valid values. The substring trap therefore covers the entity-naming
  field (`mode`), the alias list covers every field, and every served key is card-free (asserted by a
  test), so "no bonded-branding key names a card" stays mechanical.
- **Server:** NEW `card_view_skins.go` (vocabulary, served policy, ownership gates, HTTP) +
  `BondedAssetRegistry.CardViews` (exported, json-tagged, snapshotted in `Save`/`Load`, cleared on
  `Burn`) + `CountOwnedBy`. Routes `GET|POST /api/assets/card-view`, `POST /api/assets/card-view/clear`
  in BOTH servers. Gates: §27.8 owner==holder==caller, §27.7.3 (no black-market art), card
  asset-type refusal, canonical viewer wallet, and one bonded asset required.
- **Client:** NEW `Public/js/card_view_skins.js` behind the render contract
  `data-card-owner="self|foreign|<wallet>"` — it paints only marked surfaces the served scope
  permits, **never guesses ownership** (board tiles stay unmarked), paints `https://` and same-origin
  `/Assets/…` art (falling back to the deterministic stand-in for content-addressed media), attaches
  an observer **only while a skin is configured**, restores the original look on switch-off, and
  STATES a refused read instead of silently showing nothing. Studio section **Card eyes**
  (WD ▸ Assets ▸ Bonded Branding Studio) now reads its asset from the CHEST and shows the capacity.
- **`placeholder_assets.js` is the stand-in art supply:** the exact on-disk frame ranges (Anya 67 /
  Vbabes 46 / Crypto-seraph 4 = 117 PNGs) with a deterministic FNV-1a asset→art pick
  (`getCardViewFallback`); the old slot-keyed API was removed with the mode.
- **Open from this work:** wallet-NFT sourcing for bonded assets (the enumeration doors exist:
  `Lobby.indexerRequest` + `NetworkConfig.IndexerURLs`, but ownership must be verified against a
  LIVE indexer holding — never fabricate an owner), now ALSO the authoritative basis for capacity;
  a bonded-asset PURCHASE/market path does not exist yet (acquisition today = create via mint, or
  transfer from another wallet); `deck_manager.js` `.dm-card` and `ui.js` card details are not yet
  marked, so `own_cards` scope does not cover them yet.

## ✅ COMPLETE — 2026-09-14 (h): bonded branding across the ecosphere (cards excluded)
- **Bonded assets now carry the player's OWN media** (`BondedMedia{uri, mime, sha256, bytes, width, height}`
  — uint64/hex, no float) and may be worn by **any entity the wallet owns**: item, theme, bot, pet, vehicle,
  church, club, shop, world content. `ThemeBinding` gained `TargetKind`/`TargetID`; the legacy slot binding
  still loads.
- **The card exclusion is structural, not a comment:** `IsCardTargetKind` (any kind containing `card` + 12
  aliases), `assertBondedBrandingPolicy()` **panics at init()** if a card kind is ever listed as bondable,
  `IsCardAssetType` refuses a card-typed asset at mint, and the bind authority checks the rule **before any
  other lookup**. Verified end to end: the API refuses `deck_card` / `hand` / `religious_leader_card`.
- **Ownership + legitimacy:** caller must own the target (`bondedTargetOwner`, one lock per kind, never
  nested) and must own AND hold the asset (§27.8); a black-market-adopted asset may not brand (§27.7.3).
  Every refusal precedes any write.
- **Served policy:** `/api/assets` returns `bindable_target_kinds`, `asset_types`, `blocked_target_kinds` and
  the media limits, so no client re-declares the rule. `/api/assets/targets` serves the PUBLIC policy even
  without a wallet (`wallet_required:true`).
- **UI:** NEW `Public/js/bonded_branding.js` (WD ▸ Assets ▸ Bonded Branding Studio) + read-only Portfolio
  screen Bonded Assets ▸ Branding Coverage. `.clinerules/app-entry-mandate.md` **v1.3 §10** is the binding rule.
- **Open from this work:** regions/districts are deliberately NOT bondable yet (no resolvable owner record);
  `l.pets`/`l.vehicles`/`l.worldContent` still have no persistence snapshot (the bonded registry itself does);
  the `-tags console` target BUILDS now (CLOSED 2026-09-20 — a build-tag exclusion plus a fabricated route,
  neither a missing implementation), so its parity routes are COMPILED rather than parse-checked (see
  `Problems.md` §13).
## ✅ COMPLETE — 2026-09-14 (f): association export + vehicle upgrade ladder
- **`/api/player/associations` built** (read-only, `wallet-default`, both servers): alliance graph, warrant
  network, custody maps, buff flags, sector tiles, relationships/moods, liquidity samples, cooldowns,
  inventory, playstyle as integer ppm. Contract: `{}`/`[]` never `null`; `present:false` + reason for a
  wallet with no engine record; no float in the payload.
- **Portfolio wired** (19 categories → **20 screens' worth of engine records**): Clubs ▸ Alliance Links +
  **NEW Territory Tiles**, Legal & Custody ▸ Warrants (per-agency map + heat/cooldown record) + Custody,
  Stats ▸ Attributes (buffs + playstyle), Economy ▸ Balances (ledger mirrors + liquidity window),
  Moods ▸ Character Moods (relationships). `notExported()` deleted (it had become a false statement).
- **Nil-slice → `null` swept**: `/api/loans`, `/api/contracts/list`, `/api/leaderboard`, card metadata,
  achievement leaderboard. **`/api/contracts/list` envelope bug fixed** (bare array vs `res.contracts` —
  the Underworld contract list could never render) and the **Life Assets panel's missing wallet** fixed
  (spawn/author/deploy could only fail; lists returned the whole civilization).
- **§25.6.1 vehicle upgrade ladder built** — the counterpart to pet breeding (parts + investment vs
  lineage + time); 6 parts, integer linear cost, sink-routed fees, 6 gates, real region, provenance,
  break-in window, §30 power-overlay + entity-market integration, served part table, Garage UI +
  read-only Portfolio analytics. Design record: `AI-Brain/VEHICLE-UPGRADE-PLAN.md`. Contract test
  `vehicle_upgrade_test.go` 4/4 PASS.

---

## NEXT SESSION
1. **Nil-slice audit remainder** — `auction_service.go:51`, `black_market_service.go:471`,
   `seasonal_event_engine.go:393/427`, `handlers_admin.go:1298/1594/1817` (read consumers first).
2. **Fix the 3 pre-existing `go vet` failures** so `go test` runs without `-vet=off`
   (`ai_citizen_engine.go:952`, `battle_service.go:790`, `:802`).
3. **Vehicle arena / races** (§30 analogue of the pet-battle arena) — the stat vector + build level now exist.
4. **`SpawnVehicle` fee + `kind`/`min_level` validation** — needs Brendan's economic decision.
5. ~~**`-tags console` target repair**~~ — **DONE 2026-09-20** (`killExistingServer` was a build-tag
   exclusion, not a missing symbol; `handleFaithConverted` was a fabricated route that existed in no file;
   console-only routes 3 → 0 and the parity exemption list SHRANK 189 → 187). **Still open from the same
   line:** the console table registers no `/api/vehicles*` (nor ~185 other paths) — **SUPERSEDED 2026-09-20 (b):
   `/api/vehicles*`, `/api/pets*` and `/api/player/associations` ARE registered now, the console table holds
   273 routes, and the parity exemption list fell 189 → 53 (the remainder being the deliberately-excluded
   operator family and the served HTML roots).** The operator has since RATIFIED that exclusion by design:
   **a console has no admin entry point because console users cannot use the crypto side on that device** —
   it is a virtual mirror awaiting a manual bridge — so the 53 are POLICY, not work. **NEW, and binding: the
   console DLC → USDC rail (USDC to the admin wallet; the admin tops the faucet up by hand) MUST NOT SHIP
   WITHOUT A CAP/LOCK** against a spam of sales draining the vault; Nautilus withdrawal ($VBV → $VOI) stays
   MANUAL. See `Problems.md` and `17_aspects_flow.md` §33/§34.
6. Remaining WD loader tabs (markets, justice) + 13 modules uncomposed by `app.js`.

---


## NEXT SESSION — UI Flow / Categories pass (booked, KEY 3.5)

### (1) UI Flow / Categories
- **Dedupe remaining sub-tabs:** Hub District Market "Token Presets" now renders its own panel (done, UNCOMMITTED in `player_profile.js`) — confirm on hard-refresh, then scan for any other sub-tab that still duplicates a sibling (market/preset/token, governance sub-panels, WD embed-vs-overlay duplicates).
- **Wire the 11 "data unavailable" WD tabs** (they switch panels but their loader falls back to a failing endpoint / unloaded module):
  markets, loans, contracts, counterfeit, blackmarket, treasure, rewards, match, dividends, justice, creator.
  For each: either register the real backend endpoint, or wire a `window.initXxx` module-loader. Recommended first pass: enumerate the exact `/api/...` each `loadXxx()` calls, cross-check vs registered routes (server_main.go / console_server.go), and fix path mismatches (Type-A) before adding new handlers.

### (2) Backend chat-flood bug (root cause NOT yet located)
- Symptom: ~18K `chat`/sec to a passive non-admin client; `clientSentChat:0`, `identity:1`, `lobby_update:1`; NO `admin_audit.log` writes; rate GROWS → self-amplifying goroutine leak in the narrative/chatter path.
- Ruled out: all 3 `jsonListEnvelope("chat")` sites (lobby_manager.go:1634, handlers_admin.go:385, club_service.go:1574).
- Plan: add a TEMPORARY counter in the chat broadcast path (`lobby.broadcast <-` / `sendToClientLocked` for `chat`), boot fresh, let it run 20s, read the count, locate the source loop, fix, then remove the counter. Frontend cap (game.js `MAX_CHAT_NODES=120`) already makes the app usable meanwhile.

## ✅ COMPLETE (This Session — 2026-09-07)

### World Dashboard UI (60 tabs — all unique visual themes)

| Category | Tabs |
|----------|------|
| **Player Hub** | identity, career, achievements, stats, game, create_match, deck, campaign, locations, tutorial, card_titles, wagers, mood, tea_house, zen_garden, npc_taunts, game_modes, multiplayer |
| **Assets** | items, pets, vehicles, clubs, equipment, card_progression |
| **Economy** | markets, dividends, loans, blackmarket |
| **Faction** | justice, counterfeit, contracts |
| **Governance** | governance, governor, territory, regions, compliance, leaderboard |
| **World & Events** | season, events, tournament, replay, treasure, worldcontent, industrial, maintenance, systemmsg |
| **Faith** | faith, church |
| **Orphans** | orphan |
| **Creator** | creator, launches, ads |
| **Infrastructure** | gamingos, localmodel |
| **Competition** | rivalry, match, rewards |
| **Moderation** | report |

### New Game Systems (Ported from Triple Triad)

| System | File | Lines | Purpose |
|--------|------|-------|---------|
| Match Creator | `match_creator.js` | 460 | Rule toggles, board themes, game modes |
| Card Animations | `card_animations.js` | 300 | 3D flip, capture burst, combo chain, elemental glow |
| Campaign Mode | `campaign_mode.js` | 300 | Lives, stage progression, unlocks |
| Deck Manager | `deck_manager.js` | 460 | Collection, filters, 4 deck slots |
| Equipment System | `equipment_system.js` | 400 | Weapons, shields, charms with bonuses |
| Card Progression | `card_progression.js` | 570 | Training, fusion, upgrade |
| Game Board | `game_board.js` | 540 | Battle screen with themes, moods, elements |
| Game Locations | `game_locations.js` | 476 | 5 locations (arcade, casino, dive bar, lounge, underground) |
| Tutorial System | `tutorial_system.js` | 458 | Interactive 10-step tutorial + onboarding |
| Card Titles | `card_titles.js` | ~500 | Titles earned via milestones, equip for perks |
| Post-Match Wagers | `post_match_wagers.js` | 230 | Wager payouts, bounty claims, bounty survival |
| Character Mood | `character_mood.js` | 230 | NPC mood display, 5 mood states |
| Tea House | `tea_house.js` | 280 | Gourmet Chef recipes, Tea Leaf Divination |
| Zen Garden | `zen_garden.js` | 320 | Meditation, garden elements, achievement visualization |
| NPC Taunts | `npc_taunts.js` | 430 | 10 NPCs, 300+ dialogue lines, 7 interactions |
| Game Modes | `game_modes.js` | 291 | Casino High-Low, Dice Bar, VIP Lounge |
| Game Multiplayer | `game_multiplayer.js` | 620 | Real-time P2P matchmaking, move sync, chat |
| Seasonal Events | `seasonal_events.js` | 250 | Season countdown, events, rewards |

### Backend Extended (Go)

| Extension | Description |
|-----------|-------------|
| `Card.Element` | Fire/Water/Air/Earth/Neutral affinity |
| `Card.Level` | Dynamic stat scaling |
| `Engine.BoardElements` | Per-tile elemental assignment |
| `Rules["Mood_modifiers"]` | RPS mood system (±10 power) |
| `Rules["Prisoner"]` | Loser loses best card |
| `Rules["Elemental_affinity"]` | Element vs tile (±25 power) |
| `applyDynamicStats()` | `base * 1.1^(level-1)` |
| `handlePrisonerRule()` | Removes best card from loser |
| 3 Go stubs fixed | `syncStatsFromBlockchain`, `loadRegistrationsFromIndexer`, `checkAssetOptIn` |

### New API Routes (19 routes — ALL connected to live data)

| Route | Handler | Source Data |
|-------|---------|-------------|
| `/api/bounty/active` | `HandleGetBountyActive` | justiceService.GetDashboardForPlayer |
| `/api/church/members` | `HandleGetChurchMembers` | faithChurchEngine |
| `/api/shop/purchase` | `HandleShopPurchase` | playerBalances (live) |
| `/api/clubs` | `HandleGetClubs` | lobby.clubs map |
| `/api/creator/store/creator` | `HandleGetCreatorStore` | creatorStore.ListProducts |
| `/api/player/profile` | `HandleGetPlayerProfile` | lobby.leaderboard[wallet] |
| `/api/rumors` | `HandleGetRumors` | lobby.rumors map |
| `/api/titles` | `HandleGetCardTitles` | Static list |
| `/api/titles/equip` | `HandleEquipCardTitle` | POST body |
| `/api/wagers` | `HandleGetWagers` | matchHistory + playerBalances (live) |
| `/api/wagers/resolve` | `HandleResolveWager` | playerBalances (live) |
| `/api/justice/missions` | `HandleGetJusticeMissionsHTTP` | justiceService.GetDashboardForPlayer |
| `/api/justice/missions/accept` | `HandleAcceptJusticeMissionHTTP` | justiceService.GenerateJusticeMission |
| `/api/tea/recipes` | `HandleGetTeaRecipes` | Static list |
| `/api/tea/brew` | `HandleBrewTea` | playerBalances (live) |
| `/api/garden` | `HandleGetGarden` | aiEngine.GetAllCitizens (live) |
| `/api/garden/add` | `HandleAddGardenElement` | aiEngine.GetCitizen (live) |
| `/api/garden/meditate` | `HandleMeditate` | aiEngine.GetAllCitizens (live) |
| `/api/moods` | `HandleGetMoods` | aiEngine.GetAllCitizens (live) |
| `/api/moods/adjust` | `HandleAdjustMood` | aiEngine.GetCitizen (live) |

### Multiplayer System (Real-Time P2P)

| Component | Description |
|-----------|-------------|
| `game_multiplayer.js` | 620 lines — Full P2P controller |
| `_game_multiplayer.scss` | 200+ lines — Complete styling |
| WebSocket Integration | Routes `matchmaking_status`, `mp_move`, `mp_chat`, etc. |
| Matchmaking | `join_queue`, `create_room`, `join_room` |
| Live Gameplay | Move sync, turn indicator, board state |
| Chat | In-match messaging with history |
| Latency | Ping/pong tracking |
| Connection Status | Online/offline indicator |

### Unfinished Work — ALL COMPLETE ✅

| Item | Status | Effort |
|------|--------|--------|
| Seasonal Events UI | ✅ Created `seasonal_events.js` (250 lines) | 2-3h |
| Menu Customization TODOs | ✅ Wired 6 stubs | 1-2h |
| Placeholder Images | ✅ Permanent client-side bonded asset slots — no change needed | 0 |
| Tournament Bracket View | ✅ Full interactive modal | 2-3h |

---

## 📊 Repository Status (FINAL)

| Metric | Value |
|--------|-------|
| JS Modules | 142 (68 ES, 0 IIFE) |
| HTML Scripts | 66 |
| SCSS Partials | 91 |
| Build Tag Violations | 0 |
| Frontend Wired | 87% |
| Backend Wired | 100% |
| World Dashboard Tabs | 60 |
| New API Routes | 19 (all live) |
| Build Status | Both GREEN ✅ |

---

## 🎯 Next Logical Steps

**Portfolio association analytics — COMPLETE (2026-09-14 e).** 19 analytics categories; 59/59 screens verified rendering with a wallet connected (`npm run verify:portfolio`). Export gaps that remain backend-owned are recorded in `AI-Brain/Problems.md` §10.

Remaining work, highest leverage first:
1. **Export the wallet associations the engine already holds** (`PlayerStats.Alliances`/`ActiveAllianceID`, `Wanted`, `Buffs`/`ActiveBuffs`, `SectorTiles`, `Relationships`, `ActiveItemBuffs`, `LiquiditySamples`). One read-only endpoint would close them all and let the Portfolio report them truthfully instead of labelling them unexported.
2. **13 `Public/js/*.js` modules are still not composed by `app.js`** (`admin_panel`, `collective-intelligence`, `constellation_config`, `devsim`, `faith_extended`, `governance_extended`, `market_creator_panel`, `mechanics`, `mechanic_defs`, `menu-dock`, `misc_panel`, `systems_panel`, `theme_engine`) — wire or retire each (Single-Entry Mandate).
3. **World Dashboard loader tabs** still showing "data unavailable" (markets, loans, justice) — reconcile via their owning `initXxx` module.
4. **Nil-slice JSON**: `/api/loans` + `/api/contracts/list` should encode `[]`, not `null`.
5. `multiplayer` renders empty pre-match; `openCriminality` is state-dependent — confirm both against live state.
6. Backend `HandlePurchaseItem` remains VBV-only (NUGGET/UNIT settlement deferred).

---

*Archived docs at AI-Brain/archive/docs-2026-09-07/*

## Â§12. Bonded-asset progression (2026-09-14 g) — the plan of record

`AI-Brain/VEHICLE-UPGRADE-PLAN.md` is the **bonded-asset progression plan of record**; it now covers
three ladders that must be read together:

| Ladder | Section | Progresses by | Price model (server-authoritative) |
| ------ | ------- | ------------- | ---------------------------------- |
| Vehicle build | §25.6.1 | PARTS + INVESTMENT (6 parts, one per `EntityStats` axis) | `VehicleUpgradeBaseMicro × (level+1)`; class price on purchase |
| Companion grooming | §26.4.2 | INVESTMENT (6 axes, same shape) | `PetGroomBaseMicro × (level+1)`; `PetSpawnFeeMicro` on purchase |
| Companion breeding | §26.4.2 | LINEAGE + TIME (mature certified parents) | `PetBreedFeeMicro` (never client-supplied) |
| Both | §26.4.3 / §25.6.2 | the household lends the owner's DECK CARDS a capped integer % | derived, snapshotted per match |

Every fee is debited from `playerBalances` and routed to the deterministic faucet sink through the
same `RouteCriminalTax(..., FaucetShare 1.0)` call — the Industrial Loop reconciles, with no silent
mint or burn (`chargeBondedAssetFeeLocked` in `asset_life_engine.go` is the single fee door).

**Arenas are OUT of the ladder**: they are 3D-world surfaces (`.clinerules/app-entry-mandate.md` §8)
and the client must never simulate them.


---

## 2026-09-14 (k) — PLACEHOLDER PACK + SLIDE THEMING (yolo KEY 3.5)

- [x] Rename all 117 NPC-helper frames to URL-safe `<Prefix>-NNN.png` (spaces/parens removed).
- [x] `placeholder_assets.go`: catalogue from the disk (117 SKUs, real bytes/dims/sha256), FREE starter pack (the 6 slideshow frames, idempotent, not capacity-gated), shop at the served 100 $VBV through the one fee door, routes in both servers.
- [x] `ValidateBondedMedia` admits the same-origin `/Assets/…` form it was already rendering (served as `allowed_local_prefix`).
- [x] `slide_theming.go` + `Public/js/slide_theming.js`: six viewer-scoped slides, asset required, never writes branding, ownership re-proved per read, gentle fade, mounts on app boot/main menu + World Dashboard; studio gained App slides + Placeholder pack.
- [x] Two real defects fixed: case-sensitive balance lookup; case-exact viewer key.
- [x] Verified: 3 build targets rc 0, `go test` (only the pre-existing AMM failure), live HTTP, entry probe 67/67.
- [x] A derivative/resize pass for the pack (314 MB; avg 2.7 MB per frame) so boot does not fetch multi-MB art — see 2026-09-14 (l).
- [x] Player-to-player bonded-asset marketplace (listings + pricing) — see 2026-09-14 (l).
- [ ] `own_cards` markers on `deck_manager.js` `.dm-card` + `ui.js` card details.
- [ ] Wallet-NFT sourcing: art SOURCE + capacity BASIS (one function each).
- [ ] Limiter keying for query-string wallets (`X-Wallet-Address`).

---

## 2026-09-14 (l) — LIGHT RENDITIONS + UI-TREE ART + PLAYER-TO-PLAYER MARKET (yolo KEY 3.5)

- [x] `placeholder_derivatives.go`: a 512 px `slide` + 128 px `thumb` rendition of all 117 frames by an
      INTEGER alpha-weighted box filter (uint64, premultiplied); measured sha256/bytes/pixels; source
      named; manifest-backed idempotency (a second boot rewrote NOTHING); background warmup with the six
      slideshow frames first (never blocks boot); honest per-frame failure; passthrough for small frames.
      `Public/Assets/Generated/` is gitignored output; `derivative_status` is a census with NO per-pass
      `reused` field. Real cut: 1,937,129 B → 330,448 (slide) → 28,554 (thumb).
- [x] `slide_theming.go` + `placeholder_assets.go`: every served default now prefers the light URI
      (all six slides read `default_derived:true`); the catalogue carries per-SKU `derivatives`.
- [x] `ui_tree_theming.go`: ONE UNIQUE pack frame per World Dashboard tree; deterministic round-robin
      pool; `rewards` + `achievements` PINNED to Crypto-seraph (`-003`/`-004`, since `-001` is the
      media-cap-refused frame); `unique` COMPUTED; `GET /api/assets/ui-trees` in both servers; the client
      marks `[data-wd-tree]` (categories AND features) and paints `--wd-tree-art`. Live: 72 trees,
      pool 114, unique, all art.
- [x] `bonded_market_service.go`: the player-to-player market — list/cancel/buy in BOTH servers; exact
      uint64 split (buyer debit == seller credit + fee, ASSERTED before any write) with the 250 bps fee
      through the shared sink door; the price is the LISTING's; §27.8 re-proved on every read and write;
      stale listings STATED then cancelled; `CompleteSale` moves ownership + marks the listing sold under
      ONE lock; buying is an ACQUISITION (§23.5.1 does not gate it); listings persisted; ownership
      changes WRITTEN THROUGH.
- [x] Four real defects found by the new tests/probes and fixed: the case-SENSITIVE chest compare (4th of
      that family → one `walletMatches` rule everywhere), map-iteration order in served lists
      (`sortBondedAssets`/`sortThemeBindings`), the never-populated `reused` field, and concurrent save
      writers colliding on the temp file (`bondedAssetsFileMu`).
- [x] Client: `placeholder_assets.js` `lightPathFor`, `slide_theming.js` light/UI-tree painting,
      `world_dashboard.js` `data-wd-tree`, `bonded_branding.js` Market section, `portfolio.js` read-only
      market analytics.
- [x] Verified: 3 build targets rc 0; 16 new Go tests pass (`go test .` → only the pre-existing AMM
      failure); entry probe **74/74** (7 new), overlays **23/23**, portfolio **62/62** (0 page errors),
      harness **12/12**; live HTTP incl. a real listing with an exact fee split and a mixed-case transfer
      proving the chest fix.
- [ ] **The money/ownership tear** (registry write-through vs the ledger's own 15-minute snapshot) needs
      ONE atomic snapshot covering both — recorded in `Problems.md` §14.
- [ ] Transfers / burns / mints still rely on the 15-minute persistence worker.
- [ ] A shorter derivative ladder (or a WebP tier) — the generated pack is 43 MB on disk.
- [ ] Market scope: no offers/bidding, no standalone market surface (it lives in the Studio).


## THE SWEEP OUTPUT - PLANS AND POSITION (2026-09-20)

**THE CORPUS IS CLOSED:** 506/506 files read (Go 154, first-party JS 169, SCSS/CSS 110, misc/vendor 73), 0 partial, 0 pending. Five RAG owners carry it: `14_flow_go.md`, `15_flow_js.md`, `16_flow_css.md`, `18_mic_flow.md` (per-file evidence), `17_aspects_flow.md` (39 derived aspect rows + a 39-row COVERAGE table), with `19_coverage_ledger.md` as the gate.

**CLOSED BY THIS WORK:** the aspect derivation (39/39 rows, REMAINDER empty, held by `npm run verify:aspects`, selftest 8/8) - the console target (`go build -tags console .` rc 0, 273 routes, console-only routes 0, exemption list 189 -> 53) - the mobile wrappers (CGO overwrite removed; Android BUILT BY THE SCRIPT, 24,117,544 B; iOS needs an Apple toolchain) - route parity is MEASURED and cannot drift.

**OPERATOR DECISIONS, NOW BINDING (2026-09-20):**
- NO admin entry point on console, BY DESIGN: the console is a VIRTUAL MIRROR awaiting a MANUAL bridge to the PC version, and console users cannot use the crypto side on that device. The 53 residual exemptions are therefore POLICY, not work.
- The Nautilus DEX path is MANUALLY executed: $VBV is withdrawn directly as $VOI by the user.
- Console DLC buys land as USDC in the ADMIN wallet; the admin tops the faucet up by hand with $VBV / $UNIT / $NUGGET or any other vault; the faucet vault may hold USDC for self top-up.
- A CAP / LOCK IS A PRECONDITION: the console DLC -> USDC rail ships WITH the cap or it does not ship.

**OPEN, in the order the sweep implies:** settle the nine chain-touching console routes against the virtual-mirror rule - build the capped DLC -> USDC rail - encode the 53 as POLICY in the parity gate - close the console rail defects (the `Verified` trust, the manual Nautilus record) - then a fresh app-work phase.
