# Voi Tenant Framework Plan

> **Status:** OPEN — planning only. **NOT approved for execution.** Created during the plan-mode session of 2026-09-19.
> **Owner of the vision:** Brendan.
> **Rule:** nothing in this file may be built until the sequence in §8 is agreed.

---

## 1. The model (the vision, whole)

- **Voi is the sole base chain.** Every other chain is a **plug-in**.
- **Payouts happen only in $VBV, on Voi.** There is no other outbound rail.
- **Every token that is not $VBV is INWARD ONLY** — a payment rail into the system:

| Chain | Primary intake | Fallback intake | Outbound |
| --- | --- | --- | --- |
| Voi (base) | $VBV (ARC-200 `40227315`) | $VOI (native) | **$VBV only** |
| Algorand (plug-in) | $AVOI (`2320775407`) | $ALGO (native) | — none |
| Both chains | $NUGGET, $UNIT (ids supplied by Brendan) | — | — |

- **All records are kept through $VBV transaction notes.**
- **A record is a NOTE on an ARC-200 transaction from the faucet vault — NOT an app-call** (confirmed by Brendan, 2026-09-19). The payload is **gzipped**; the reader extracts (gunzips) the note to recompile the save data, **compares it with the local cache and updates the cache to match** — the on-chain note is the authority, the local cache syncs to it.
- **The record is the WHOLE game state plus every vault interaction — not a summary.** The local cache exists **only for runtime smoothness** and is always **calibrated by the blockchain record**. A full restore may therefore require **sequential gzip pulls**: the vault's record transactions pulled **in order**, gunzipped, and reassembled into the complete state.
- **The seed IS the record/save file**, and it records **everything relevant to the faucet vault's interactions** (enumerated in §9).
- **A `Nugget` or `Unit` shop is a UNIVERSAL SELLER.** It may sell **everything the app offers through every other shop** — it is not limited to its own inventory — but the whole purchase is transacted **in its own token** and **recorded on the faucet vault**, the interaction being pushed as a **self-nonce update** to the vault's record (§10).
- **$NUGGET and $UNIT are inward-only, and can never be acquired inside this app** except by (a) **running a shop that accepts that token** (the seller receives it), or (b) **transacting through the dev/game hub** by choice of framework or chain. The app never mints and never grants them.
- **House-side token isolation is strict:** only the `Nugget` shop deals in $NUGGET and only the `Unit` shop deals in $UNIT — each accepting **only its own token** — **unless a dev/game-hub tenant chooses to build differently.**
- **The records are IN the seed.** The seed defines every save type; nothing seeds until the seed is ready for all of them.
- **$NUGGET and $UNIT transact only inside their own shop fronts.**
- **An Algorand user never needs Voi.** Voi and Algorand are both AVM chains, so a user's Algorand address *is* their Voi address. Onboarding provisions that same address on Voi (gas + $VBV), and the vault tracks the user's cross-chain economy in its **seeded records**.
- **Dev/game hub builders** may add **their own payment forms for their own shops, wallets and chains** — and they **must pay lease fees in $VBV**, which creates the need for $VBV while allowing users to establish their own frameworks.
- **Tenants must seed on $VBV for records**, using a **vault-ONLY address and mnemonic** they provide, so their service works the same as ours, with their own vault-only seed/record address.

---

## 2. Verified foundations (backend, measured — not assumed)

| Foundation | Verified state |
| --- | --- |
| Tenant token role | `reward_registry.go:35-38` — the role vocabulary already includes **`tenant`** (`primary` / `distribution` / `tenant` / `legacy`) |
| Lease catalogue + money rule | `/api/lease/available`, `/api/lease/list`, `/api/lease/create`; the served payload already states `money_rule: "rates are integer micro units of $VBV"` and `price_rule: "the rate is the SERVER's"`; `CreateLease` **refuses**, naming 5 blockers (`infrastructure_lease.go:79-107`) |
| Tenant world record (the shape requested) | `TenantWorldLease{tenant_wallet, world_id, network, asset_id, app_id, rate_micro, period_seconds, meter_basis_bytes, state, spawned}`; states `proposed` / `active` / `suspended` / `revoked`; boot-time dialect guard |
| Gaming OS modules | `GamingOSEngine` / `OSModule{monthly_rate, lease_count}` + `/api/os/{modules,module/register,lease,leases,summary}`; `LeaseModule()` **charges nothing** |
| Vault identity slots (unwired) | `MultiChainRouter` declares `VaultAddress`, `VaultMnemonic`, `AlgorandVaultAddress`, `AlgorandVaultMnemonic` — **assigned nowhere**; the live identity is `l.vaultAddress` + `FAUCET_MNEMONIC` (env-only) |
| The record system | `note_vocabulary.go` owns 36 purposes; `checkpoint_indexer_read.go` searches `note-prefix` (base64) and **asserts the author IS the vault** — a third party's note cannot be read as our state |
| The signature rail (reuse this) | the nonce is issued over the WebSocket (`lobby_manager.go:1042` sends `nonce_response`); the HTTP doors verify wallet signatures — **no new crypto needed** |
| Onboarding (already the pattern) | `HandleVoiOnboarding` sends an atomic **1 VOI (gas) + 1 $VBV (ARC-200)** group, notes `NotePrefixOnboard+"GAS"` / `+"TOKEN"`, guarded by `onboardedWallets` → `VBT_ONBOARD_SNAPSHOT`; **it has never executed on chain** (the vault has sent 0 `appl` txs) |
| The dev hub surface | `dev_game_hub.js` — a read-only catalogue of **38 composable functions** with config-side `priceMicro`/`power`; **no purchase door exists and no Go file references any catalogue id**; `DEV-HUB-INDEX.md` is the source of truth and says the hub is built **LAST** |

---

## 3. Security decisions (the key questions, answered)

1. **Hashing cannot store a signing key.** A hash is one-way; signing needs the key back. Hashing's only legitimate role here is verification — and even a *salted hash* of a mnemonic must never be published (a low-entropy phrase can be ground against it offline).
2. **The seed records carry the vault ADDRESS only** — never a mnemonic, never any derivative of one.
3. **NO CUSTODY — this is the law (Brendan, 2026-09-19).** The app holds **no tenant key, ever, in any form**. The tenant maintains custody of their secrets **through the bot they assign**; the tenant gives the bot the vault **address** (and key), never us. Control is proven by **a signature over a server nonce**, never by possession of a key.
4. **WalletConnect cannot automate.** It is human-in-the-loop by design (a push the person must approve, with session expiry). It is for human moments only: link the vault, approve the first seed, change the policy.
5. **Split the ink from the money.** Record notes are frequent and carry no value; payouts are rare and valuable. Automation therefore runs on a hot key that **cannot touch value**.
6. **Automation rails:** (1) **tenant-run signer bot, pull jobs** — build first; (2) **rekey / operator key** — chain-native, and Voi's rekey support must be verified before relying on it; (3) **LogicSig vault with on-chain caps** — the target (only these destinations, only these note prefixes, a fee ceiling, no close-out).
7. **House custody is REJECTED (Brendan, 2026-09-19).** There is no opt-in, no envelope encryption and no tenant key on our box, ever. The only signer of a tenant's records is **the tenant's own bot**.

---

## 4. The build — four stages, each one owner

**Stage A · Tenant vault identity + the guided setup (no money moves).**
A tenant registers a **vault-only address** — never a key. Enforced: the address may not be a registered player wallet, and may not already be a tenant vault. Ownership is proven by **signing a server nonce** (the rail above). Nothing is charged, nothing is seeded.
*Deliverable: a builder can stand up their vault identity safely, and the hand-holding is visible before any value is involved.*

**Stage B · The toll — the lease fee in $VBV.**
Split in two, because the CATALOGUE cannot be complete until the app is (Brendan, 2026-09-19):

- **B1 — the toll mechanism (buildable early).** `CreateLease` begins to work: it charges the **server-priced** fee in **$VBV** through the ONE existing sink door and writes the **first lease record family** (authored by the house vault — the toll is house-side, so no tenant key is involved). Proven with a minimal catalogue.
- **B2 — the full catalogue (built LAST, with the hub).** The catalogue must be able to lease **all architecture and frameworks possible**, so it can only be completed once the app is complete **minus the dev/game hub**. That is exactly why the hub is last.

**The writer change this model requires (from §1's record definition).**
Today `dispatchBlockchainSnapshot` gzips → base64 → `sendNoteTx(note)` and `sendNoteTx` dispatches a **`MakeApplicationNoOpTx` app-call**. The model says a record is a **note on an ARC-200 transaction from the vault**, so the transport becomes an **ARC-200 `transfer(address,uint256)`** (selector `0x2b426dec`, already used in four places) sent **from the vault** with the gzipped note attached. The gzip + base64 payload logic is unchanged, and the reader built on 2026-09-19 already reads **transaction** notes by base64 `note-prefix`, so it needs no change. Two on-chain facts to verify before relying on it: whether the token app accepts a **0-amount** transfer (if not, a 1-micro-unit self-transfer), and that the note survives on the transfer (notes are per-transaction, so it does).

> **✅ DONE 2026-09-19 (build 1).** `economy_service.go` now owns the record transport: `buildRecordTransferTx` builds the **ARC-200 self-transfer carrying the note** (selector `2b426dec`, recipient = the vault, amount = **1 micro** — not 0, because an ARC-200 app may reject a zero transfer and a rejected record is silent state loss), `dispatchRecordNote` is the ONE sign-and-send, and `sendNoteTx` is now the **GATED** door (`RECORDS_DISPATCH`, default OFF, so no daemon spends fees while the save set is incomplete) with `sendRecordNoteNow` as the explicit proof path. Pinned by `record_transport_test.go`: the transaction is an **appl** call to **40227315** whose args are exactly `[selector, vault, 1]` with the note attached, and the gate defaults OFF and refuses (never a silent success). Native + `linux/amd64` + `js/wasm` rc 0; `Public/main.wasm` untouched (11,375,951 B, same mtime); `go test .` green except the PRE-EXISTING AMM test.
>
> **✅ CONFIRMED by the operator (2026-09-19): the $VBV token app ACCEPTS a micro self-transfer**, so the ARC-200 self-transfer carrying the note is the correct transport. (Operator-confirmed, not machine-measured — the on-chain proof is still a deliberate, fee-spending step for Brendan.)
>
> **⚠ AND THE RULE THAT GOES WITH IT — ONE TRANSACTION, NEVER TWO.** Brendan (2026-09-19): *"we must ensure we do not duplicate a call to transaction."* A record must ride **on the transaction that already moves the value**; only a record with **no** accompanying movement (a state checkpoint) may be its own single self-transfer. **Measured today: ten call sites fire a SEPARATE record transaction beside their movement** — `NotePrefixAuctionSettle` (auction_service.go:330), `NotePrefixHeistLog` (club_service.go:336), `NotePrefixSabotageLog` (club_service.go:1551), `NotePrefixLeaseTake`/`NotePrefixLeaseReturn` (club_service.go:3240/3301), `NotePrefixRansomLog` (handlers_criminality.go:386), `NotePrefixBailLog` (:737), `NotePrefixInsuranceReturn` (:823), `NotePrefixLoanPayback` ×2 (loan_service.go:269/325) and `NotePrefixLoanLiquidate` (:400). Each of those is a **duplicated transaction call** and must be folded into the movement's own note. See §12.

**Stage C · The ink — the tenant record write path (automation, rail 1).**
The architecture becomes **compose → sign → submit**: the server composes the tenant's record transaction group and parks it as a **signing job**; the tenant's **bot** long-polls `GET /api/tenant/signing-jobs`, signs **locally**, and posts the signed bytes back; we broadcast and verify the author is **their** vault. Their key never leaves their machine and their service runs unattended. **The bot is seeded by the user on first creation** — the one human moment — and from then on it signs unattended (§8 item 2).

**Stage D · The seed — genesis, once and complete.**
The tenant funds their vault (**VOI for gas + $VBV for ink**), then **one signed action** writes their complete initial record set; the reader restores it. Gated on the record set being complete, because a seed is written once.

---

## 5. The guided setup UI (holding the builder's hand)

A staged wizard in the **Dev/Game Hub** — one screen per step, **verifiable and resumable** (every step's state is server-known, so a builder can leave and come back):

0. **What you're setting up** — plain words: you are leasing the architecture; you will need a vault-only wallet, VOI for gas, and $VBV for ink and the toll.
1. **Create or connect your VAULT wallet** — WalletConnect (the human moment), then the **vault-only check**, with the reason stated ("a vault key must never be a player key" — the discipline AI citizens already live under).
2. **Prove it's yours** — one signature over a server nonce. **No mnemonic field exists anywhere in this UI.**
3. **Fund it** — the exact amounts and the exact addresses, **live balance reads** (we can now read `/arc200/balances` and the account's VOI) and a **Re-check** button; it will not advance until funded.
4. **Install your signer bot** — a copy-paste job token, a one-line command, and a live **"bot connected"** light that turns on from the first successful job poll.
5. **Confirm the lease, pay the toll in $VBV** — server-priced, exact integer micro amount, and the resulting record txid.
6. **Seed your world** — the record types that will be written, **named one by one**, one signed action, per-item success/failure with txids.
7. **Go live** — vault, lease state, last record, bot heartbeat, and a **"what if…"** panel: bot key lost → rotate; low gas → top up.

Every refusal names the **one** missing thing with its exact shortfall (the repo's established style). Built with `data-*` attributes and one delegated listener (no inline handlers) so it passes the existing handler gate, and it reuses the contained-overlay mandate.

---

## 6. Invariants (binding for anything built from this plan)

- The house never holds a tenant key (the default).
- The seed records carry the vault **address only** — never a mnemonic, never a derivative of one.
- The lease fee is **$VBV-only** and **server-priced** (a caller-supplied rate is ignored and the refusal says so).
- Every token other than $VBV is **inward only**.
- Payouts are **$VBV on Voi only**.
- A tenant's own shops may accept their own tokens, their own wallets and their own chains — which is exactly why the **toll keeps $VBV necessary**.
- **Nothing seeds until the record set is complete.**

---

## 7. Verification per stage

| Stage | How it is proven |
| --- | --- |
| A | A Go test for the vault-only check (refuses a player wallet, refuses a duplicate tenant vault) + a **live** nonce → sign → verify round trip |
| B | An exact-integer fee test, the sink reconciliation (debit == faucet credit), and a **real record txid** on chain |
| C | A test bot driven end-to-end in a Go test, then a **live** job → sign → submit → verify, with the reader scoped per vault |
| D | The record set enumerated with each item restored, proven with the on-chain txids |

---

## 8. Decisions taken (2026-09-19) — all settled

**RESOLVED**

| # | Item | Answer |
| --- | --- | --- |
| 1 | What is in the seed | **The seed IS the record/save file** — it records **everything relevant to the faucet vault's interactions**; the concrete list is §9, for confirmation |
| 2 | The $NUGGET / $UNIT shops | They exist as **explicit shop categories `Nugget` and `Unit`** — `admin.js:1848` ("distinct ClubType values from GlobalShopRegistry **+ explicit Nugget/Unit shops**"), and the panel's own rule (`index.html:547`) is "**$VBV is default; $NUGGET and $UNIT shops accept only their respective token**" |
| 3 | Record format | **A note on an ARC-200 transaction from the faucet vault — NOT an app-call**; **gzipped**, extracted to recompile the save data, and compared with the local cache so the cache syncs to the chain |
| 4 | Tenant vault provisioning | **The tenant gives the address to their own bot.** We never receive the key |
| 6 | Custody law | **NO CUSTODY — the app holds no tenant key, ever.** The user maintains custody through the assigned bot |
| 7 | Lease catalogue | **Cannot be fully expanded until the app is complete minus the dev/game hub** — the catalogue must be able to lease ALL architecture and frameworks possible, so the hub is **last** (matches `DEV-HUB-INDEX.md`) |

**SETTLED on 2026-09-19 — no further questions on these**

1. **The record persists the ENTIRE game state**, plus every vault interaction, and may require **sequential gzip pulls** to reassemble. The local cache is runtime-only and is **calibrated by the chain**.
2. **A tenant's bot is seeded by the user on first creation** — the one human moment. From then on it signs unattended, and the app holds **no key** (§3 law).
3. **The self-nonce** is recorded as: a **monotonic per-vault sequence number** inside the gzipped note, making the record an **ordered, replay-proof log with detectable gaps**.
4. **Stage order accepted:** identity → toll → automation → seed → hub, with the **full catalogue last**.
5. **Lease catalogue — decided by the architect: ONE catalogue** (the union of the house systems and the Gaming OS modules), served by one route, completed at B2 with the hub.

**ANSWERED on 2026-09-19 (folded into §1)**

- $NUGGET/$UNIT shops = **universal sellers**, transacting in their own token, recorded on the vault via a self-nonce update.
- $NUGGET/$UNIT = **inward-only, never acquirable in-app** except by running a shop that accepts them, or via the dev/game hub.
- **Only** the relevant shop transacts the relevant token, and only that token — **unless a dev/game-hub tenant chooses to build differently**.

---

## 9. The seed's record set — the ENTIRE server state (settled 2026-09-19)

The scope given on 2026-09-19 is *"everything relevant to our faucet vault interactions"*. These are the vault interactions the code already performs; the seed must be able to record **every one** of them, plus the state they describe.

**Outward (the vault pays — $VBV on Voi only):**
1. Onboarding starter pack — 1 VOI (gas) + 1 $VBV (`HandleVoiOnboarding`)
2. Faucet reward payouts — win / DNF / bounty
3. Tournament payouts and the pot share
4. Entity + governance dividends

**Inward (players pay the vault — any supported rail, inward only):**
5. Tournament / arena buy-ins
6. Courthouse fines and bail
7. Loan repayment
8. Club founding and territory purchase
9. Lease toll (**to be built**, §4 Stage B1)
10. Vault donations

**Returns / automated movement:**
11. Loan auto-pull, insurance return, auction settle/pull

**State records (the save data the vault keeps, one family per domain):**
12. `VBT_STATE_SNAPSHOT` (leaderboard) · `VBT_ECONOMY_SNAPSHOT` · `VBT_REG_TX_SNAPSHOT` · `VBT_LINK_SNAPSHOT` · `VBT_ONBOARD_SNAPSHOT` · `VBT_CARD_CACHE_SNAPSHOT`

**SETTLED — the list IS the entire state (Brendan, 2026-09-19).** Everything the game can transact must be persisted, and the **entire server state** is in the record — "even if it requires sequential gzip pulls to get the full record". The six families above are therefore **a start, not the list**: clubs, territories, loans, auctions, tournaments, the entity market, pets, vehicles, world content, rivalries and every other transactible each need their own state family. **None has ever been written**, so the seed will be their first real record — and because a seed is written **once**, **every family must exist before it is written.**

---

## 10. $NUGGET / $UNIT token rules (settled 2026-09-19)

1. **They are universal sellers, not niche shops.** The `Nugget` shop and the `Unit` shop may offer **everything the app offers through every other shop** — the whole catalogue — but the purchase is transacted **in that shop's token**.
2. **Inward only.** The app never mints or grants $NUGGET or $UNIT. A user **cannot acquire them inside this app** unless they (a) **run a shop that accepts that token** (so they receive it as a seller), or (b) **transact through the dev/game hub** by choice of framework or chain.
3. **Strict house-side isolation.** Only the `Nugget` shop transacts $NUGGET — and only $NUGGET. Only the `Unit` shop transacts $UNIT — and only $UNIT. **A dev/game-hub tenant may build differently** for their own framework (this is the same freedom as their own payment forms, §1).
4. **Every such interaction is recorded on the faucet vault**, pushed as a **self-nonce update** to the vault's record.

**The self-nonce (to be confirmed — §8 item 1).** As understood: each record carries a **monotonic per-vault sequence number** inside its gzipped note, so the vault's record log is **ordered** and **replay-proof**, and a reader can detect a **gap** (a missing update). This is what "self-nonce update to the record" means in the plan until corrected.

---

## 11. What a "record family" means (§8 item 1, in plain terms)

- A **record** is one note on an ARC-200 transaction from the vault, carrying a **gzipped** payload and a self-nonce.
- There are two kinds:
  - **Event records** — *something happened*: an onboarding pack, a payout, a fine, a buy-in, a lease toll. These **accumulate as the app runs**; they do not need to be seeded, they *are* the history.
  - **State records** — *the app's memory*: the leaderboard, the economy, the registered txids, the linked identities, the onboarded wallets, the card cache. These are written periodically and are what a **restart restores from**.
- The **seed** is the *initial* set of both — but the part that must be **complete before seeding** is the **state** list, because a missing state family means that part of the app **cannot be rebuilt from the chain**.
- **SETTLED (2026-09-19): the list IS the entire state.** Every transactible the game holds must have a state family — the six that exist are a **start, not the list**. Because a seed is written **once**, **every family must exist before it is written**, and a full restore may take **sequential gzip pulls**.

---

## 12. Binding rules from the 2026-09-19 review

**12.1 ONE TRANSACTION PER MOVEMENT — NEVER A DUPLICATED CALL.**
Brendan: *"we must ensure we do not duplicate a call to transaction."*
A record rides **on the transaction that already moves the value**. Only a record that has **no** movement of its own (a state checkpoint) may be a single 1-micro self-transfer (now operator-confirmed to be accepted by the $VBV app). **Measured: ten call sites currently fire a SEPARATE record transaction beside their movement** (§4 lists them), so each is a duplicate and must be folded into the movement's own note. It is also cheaper: one fee instead of two.

**12.2 THE SAVE SET MUST RESTORE THE ENTIRE SERVER — a MAJOR work item.**
Brendan: *"this needs major updating as it will not restore the entirety of the server."* The six existing state families do **not** rebuild the server, so the save/restore contract itself must be reworked. Shape:
1. **Enumerate every state owner** — the `Lobby` maps/slices, the engine-owned registries (AI citizens, faith, religion governance, items, bonded assets, local-model promotion) and the file-backed ones — the map-race gate already derives one such list.
2. **Give each a record family and a restore path**, marking those that are process-only or file-only today.
3. **Batch into ONE record per cadence** (gzipped JSON), chunked across **sequential** notes when the state is too large — which is exactly what the reader's ordered note-prefix pull is for.
4. **Only when (1)–(3) are complete is `RECORDS_DISPATCH` switched on.**

**12.3 The reader stays single** (§5 rec 5, accepted): every family is added to `checkpoint_indexer_read.go`; no second note reader is created.

**12.4 Batching** (§5 rec 3, accepted): one record per cadence, not one per family.

---

## 14. SESSION STATE — COMPRESSED (2026-09-19, for continuation)

**DONE**
- **Build 1 — the record transport.** `economy_service.go`: `buildRecordTransferTx` (ARC-200 self-transfer carrying the note: selector `2b426dec`, recipient = vault, amount **1 micro**, operator-confirmed accepted), `dispatchRecordNote` (one sign-and-send), `sendNoteTx` **GATED** by `RECORDS_DISPATCH` (default OFF), `sendRecordNoteNow` for an explicit proof. Pinned by `record_transport_test.go`.
- **Item 1 — the record-set owner.** `record_families.go`: `PrimaryRecordFamilies`, `DerivedState` (9, each naming its source), `EphemeralState` (7, named), `assertRecordSetDialect()` at init (boot-fails on a contradiction — it caught `treasuryAverages` declared primary AND derived), `RecordSetCensus()`. Tests in `record_families_test.go` pin the census and "one fact, one owner".
- **A1 — the thirteen remaining families DECLARED (2026-09-19, second pass).** `entity_market` · `pets` · `vehicles` · `world_content` · `rivalries` · `fenced_listings` · `match_history` · `ai_citizens` · `bonded_assets` · `items` · `faith` · `local_models` · `season_archive` — a vocabulary constant + catalogue entry each, the family's `Prefix` set and its `Pending:` blocker DELETED in the same edit, then the census advanced. **`RecordSetCensus()` now reports 22 declared / 0 pending: the §13 C list is CLOSED.** Build-breaking trap recorded: the family key `season_archive` was ALREADY a catalogue key (the audit-log entry), so the family is keyed `season_archive_state`, and `VBT_SEASON_ARCHIVE_SNAPSHOT:` is not ambiguous with `VBT_SEASON_ARCHIVE:` (the colon, not an underscore). **The ledger drift this pass measured is in §13 D — the census now counts declarations, which is not yet coverage.**
- **A2 — THE READER + the record envelope (2026-09-19, third pass).** See NEXT item 2 for the whole shape. Plus ONE real
  TRANSPORT defect found by the new end-to-end test and fixed: `indexerGet` cancelled the attempt context immediately
  after `Do`, before its caller decoded the body, so any response larger than the socket buffer answered
  `context canceled` — the read path worked only because every earlier consumer read a tiny body or a 404. NEW
  `indexerBufferedResponse` reads the body while the context is alive (32 MiB cap) and hands back a buffered response;
  the 404 branch is buffered too, because a 404 is returned to its callers. Pinned by
  `TestIndexerGetReadsABodyLargerThanTheSocketBuffer` (~128 KB). `Problems.md` §38 B.

- **THE OUTBOUND RAIL IS CLOSED (2026-09-19, fourth pass; operator adjudication).** `TransferToChain` REFUSES a non-Voi hint and refuses when no healthy Voi node exists (no plug-in fallback), `selectChain` takes no hint and answers only "voi"/"" , and `transferToAlgorandMainnet` REFUSES (its 61-line broadcast body removed; the function is kept so a caller gets a stated refusal). The Algorand client remains for INBOUND verification only, and the ONE permitted native-VOI departure — the onboarding gas stipend — is NAMED at its call site. This is load-bearing: the payout rail and the RECORD rail are the same ARC-200 $VBV rail, so one rail keeps one reader, one vault identity and one nonce stream. `Problems.md` §40 A.

**NEXT, SEQUENTIALLY (each step must leave the tree green)**
1. ✅ **DONE 2026-09-19 (second pass) — the thirteen remaining families are declared; the census reads 22/0.** See the A1 bullet above.
2. ✅ **DONE 2026-09-19 (third pass) — THE READER, and the envelope it needed.** NEW `record_envelope.go` owns the note grammar `<prefix>R1.<nonce>.<chunk>.<total>.<base64(gzip(data))>`: `encodeRecordNotes` splits a record into the smallest number of notes that fit the AVM's **1024 byte** cap (solved as a fixed point, then every note MEASURED and the chunks re-joined and compared with the source bytes), the self-nonce is a monotonic per-family **SEQUENCE** (so a missing update is detectable), and `assembleRecords` returns the newest COMPLETE record plus every INCOMPLETE nonce as a **named gap**. `readRecordFamily` (`checkpoint_indexer_read.go`) is now THE reader for all three callers — `loadLeaderboard`, `loadEconomyState`, `loadBlockchainStateSnapshotLocked` — and the writer writes CHUNK NOTES and **REFUSES an undeclared prefix**. It also found and fixed a real transport defect: `indexerGet` cancelled the attempt context BEFORE its caller read the body, so any response above the socket buffer answered `context canceled` (`Problems.md` §38).
3. ✅ **DONE 2026-09-19 (third pass) — DE-DUPLICATION: MEASURED, AND THE FOLDING IS ALREADY IN PLACE WHERE IT APPLIES.** Every vault-authored payout passes its note INTO the value-moving transaction (`onboarding_service.go:318/322` · `faucet_service.go:118/451` · `economy_service.go:287` · `tournament_manager.go:871` · `loan_service.go:448`), and the eleven `sendNoteTx` sites (this list says "ten" but names ELEVEN) are AUDIT LOGS for IN-MEMORY movements — no vault transaction exists to ride on, so their single 1-micro self-transfer is the sanctioned shape and they were deliberately NOT changed. The pass instead found and fixed THREE defects: the tournament archive was "written" by an indexer **GET** (so it was written NOWHERE while `Links` carried fabricated URLs), and two note LITERALS (`VBET_ALGO_DIV`, `VBET_VOI_DIV`) sat on real payouts — now declared in the vocabulary and pinned by a NEW shape-based gate. `Problems.md` §39.
4. **Batching** — one record per cadence (gzipped JSON), chunked sequentially when too large — not one record per family.
5. ✅ **DONE 2026-09-19 (fourth pass) — THE A4 VERDICT, APPLIED BY MEASURING THE READERS.** Measured: `treasuryAverages` has a WRITER (`processTreasuryAnalytics`) and TWO READERS (`item_service.go`) so it is PRIMARY (a path-dependent EMA cannot be rebuilt) and is now carried by the `clubs` family; `rewardStack`/`rewardTokens` are written by the economy snapshot and an admin token exists nowhere else, so they are PRIMARY and carried by the `economy` family; `holdingBonuses` (no writer, no reader) and `CollectorMap` (written, never read) had their FIELDS RETIRED; `RegionalDistricts` kept with its source CORRECTED to what the code does. **`DerivedState` 9 → 5 rows, every one naming a source that exists**, and the `economy` family now names the nine facts its payload marshals that no row named. No new reconstruction function was needed, and that is a measurement: each remaining row is already rebuilt by code that exists. `Problems.md` §40.
6. **The TENANT VAULT family** — new primary state introduced by Stage A; it must land BEFORE the registry is built.
7. **Stage A backend:** tenant vault registry + vault-only checks + **one** extracted signature verifier (reused by the link handler) + the served setup contract the wizard reads.

**CONSTRAINTS IN FORCE:** `RECORDS_DISPATCH` stays **OFF**; **no live/on-chain testing until the UI is complete**; the **seed is planted only after the UI**; no duplicated transaction calls; secrets are env-only; nothing seeds while any primary fact is still PENDING.

---

## 13. THE SAVE CONTRACT, AS MEASURED (2026-09-19) — the gate for the UI

Brendan: *"this will not restore the entirety of the server."* Measured, the server's memory is split **three ways**, and a restart restores only the first:

### A · Chain record families — what CAN be rebuilt from the chain today (6)
| Family | Written from |
| --- | --- |
| `VBT_STATE_SNAPSHOT` (leaderboard) | `lobby_manager.go:568` via `dispatchBlockchainSnapshot` |
| `VBT_ECONOMY_SNAPSHOT` | `economy_service.go:157` |
| `VBT_REG_TX_SNAPSHOT` | `lobby_manager.go:2830` |
| `VBT_LINK_SNAPSHOT` | `identity_bridge.go:89/136`, `lobby_manager.go:2848/2878` |
| `VBT_ONBOARD_SNAPSHOT` | `SaveOnboardedWallets` (`lobby_manager.go:4955` → `oracle_service.go:967`) |
| `VBT_CARD_CACHE_SNAPSHOT` | `oracle_service.go:2049` |

### B · Local-JSON registries — restorable from DISK only, in NO record family (11)
`ai_citizens.json` · `bonded_assets.json` · `card_cache.json` · `derivatives.json` · `economy_state_authoritative.json` · `faith_churches.json` · `item_registry.json` · `linked_wallets.json` · `local_model_promotions.json` · `religions.json` · `season_<n>_archive.json`.
Each has a working save+load pair, so a restart on the **same disk** is fine — but **nothing is on chain**, so a lost disk loses them and no record can rebuild them.

### C · Process-only state — lost on ANY restart, in NO family and NO file (the major gap)
The `Lobby` maps with no save path at all, including: `clubs` (+ their `Territories` and inventories), `loans`, `auctions`, `matches` and `matchHistory`, `marketNodes`, `pets`, `vehicles`, `worldContent`, `Rivalries`, `fencedListings`, `playerDirectInvestments` and `playerInvestmentRecords`, `treasuryAverages`, `RegionalDistricts`, `rewardStack`/`rewardTokens`, `bannedAvatars`, `rumors`, `Caches`, `HistoricalFrames`, `Upgrades`, `CollectorMap`, `holdingBonuses`, `Grooming` (inside pets).

### D · THE DECLARATIONS ARE COMPLETE; THE LEDGER STILL UNDER-STATES THE WRITERS (measured 2026-09-19, second pass)

All **22** families are now declared (census **22 declared / 0 pending**), so the reader has ONE list to iterate and no prefix has to be invented later. What that pass ALSO measured — by reading the two live writers against this file's own ledger — is that the ledger's statements are wrong in four ways. **Recorded, NOT applied:** this file owns the classification, so changing it is an operator decision, not an implementation detail. Full evidence: `Problems.md` §37.

| # | Finding | Evidence |
| --- | --- | --- |
| D1 | **State written today that NO row names** — `active_kidnappings`, the live `tournament`, `season_num`/`season_start`, `paid_participants`, and the four token-sink audit counters (`audit_inflow`/`allocated`/`siphoned`/`exited`). Also `bannedAvatars`, named in §13 C's prose but present in NONE of the three lists in `record_families.go` | `saveEconomyState` payload, `lobby_manager.go:795-827` |
| D2 | **One fact, two owners** — `match_history` and `market_nodes` are written by the economy snapshot AND declared as families of their own; `onboarded_wallets` has two writers (the economy snapshot and `VBT_ONBOARD_SNAPSHOT`, `oracle_service.go:982`) | as above. The "one fact, one owner" test compares `Carries` STRINGS, so it cannot see any of it |
| D3 | **A class claim contradicted by the writer** — `initial_rewards` (`initialRewards`) and `reward_tokens` (`rewardTokens`) are listed Class 2 (derived). They are WRITTEN, and an ADMIN-created reward token exists nowhere else (the env half is re-seeded at boot, the admin half is not), so they are PRIMARY | `DerivedState` row vs `lobby_manager.go:803-804`; `reward_registry.go` |
| D4 | **Three Class 2 rows name a rebuild source that does not exist** — `treasuryAverages`, `holdingBonuses` and `CollectorMap` each appear exactly ONCE in all `*.go` (their own declaration): no writer, no reader | `backend_types.go:622/593/170` |

**A note that matters BEFORE D2 is "fixed":** batching (NEXT item 4) puts several families into ONE record, so the economy payload carrying `match_history` is a TRANSPORT fact, not automatically two owners. What is actually wrong is that the ledger never says WHICH family owns it. **Fix the ledger, not the payload** — and only then decide whether any of D1–D4 changes a class.

### The contract this defines
1. **Every row in A, B and C must end up in a record family with a restore path**, or be explicitly declared *derived* (rebuildable from other records) — no row may stay silent.
2. **B's files stay as the runtime cache** (smoothness, per §1) — the chain becomes their authority.
3. **C is the work**: it is the difference between "the server restarts" and "the server restores".
4. **`RECORDS_DISPATCH` stays OFF until A+B+C are covered**, because a partial record log is worse than none — it would *look* complete.

### ⭐ THE RULE (Brendan, 2026-09-19): **A IS THE SOURCE — B AND C MUST BE RECONSTRUCTIBLE FROM A**

So the design question per item is **not** "give it its own family", it is:

| Class | Meaning | What happens to it |
| --- | --- | --- |
| **1 · PRIMARY** | A fact the world cannot rebuild from anything else (ownership, membership, balances, contracts, lineage) | **Must be recorded in A** — one family per domain |
| **2 · DERIVED** | Rebuildable from A (or from another authoritative source, e.g. a manifest that can be regenerated) | **Stays out of A**; it becomes a projection/cache rebuilt from A |
| **3 · EPHEMERAL** | Meaningless after a restart by nature | **Declared runtime-only**, and *named* so silence is never mistaken for coverage |

**Classification of B and C (first pass, to be completed domain by domain):**

- **B → Class 1 (must be in A):** `ai_citizens.json`, `bonded_assets.json`, `faith_churches.json`, `item_registry.json`, `local_model_promotions.json`, `religions.json`, `season_<n>_archive.json`.
- **B → Class 2 (already reconstructible):** `linked_wallets.json` (has `VBT_LINK_SNAPSHOT`), `economy_state_authoritative.json` (has `VBT_ECONOMY_SNAPSHOT`), `card_cache.json` (has `VBT_CARD_CACHE_SNAPSHOT` *and* is refetchable), `derivatives.json` (a regenerable manifest).
- **C → Class 1 (must be in A):** `clubs` (+ territories/inventories/staff/mojo/treasury), `loans`, `auctions`, `marketNodes` + `playerDirectInvestments` + `playerInvestmentRecords`, `pets` (incl. `Grooming`), `vehicles`, `worldContent`, `Rivalries`, `fencedListings`, `matchHistory`.
- **C → Class 2 (derived):** `treasuryAverages` (from the treasury state), `holdingBonuses` (from holdings), `rewardStack`/`rewardTokens` (from the reward registry + pool), `CollectorMap`, `RegionalDistricts` (from clubs + territories).
- **C → Class 3 (ephemeral, named):** live `matches` (a restart mid-match loses that match and says so), `rumors`, `Caches`, `HistoricalFrames`, `bannedAvatars` *(verify — may be Class 1)*, plus the limiter/nonce/session maps.

**Consequence:** A grows **only by primary facts**; every Class 2 item gets a **reconstruction path** rather than a family; every Class 3 item is **listed in the plan** so "not recorded" is a decision, not an oversight. The wizard's step 6 can then name exactly which families the seed writes, and step 7 can truthfully claim the world restores.
The wizard's later steps (fund → seed → go live) can only make honest promises once this contract is complete: step 6 ("Seed your world") names the record types that will be written, and step 7 ("Go live") claims the world survives a restart. Neither can be truthful while B and C are outside the record set.

**GIT PUSH remains Brendan's.**

---

## 15. A5 - BATCHING: THE EXECUTION SPEC (locked 2026-09-19)

**THE SHAPE.** One record per cadence carrying SEVERAL families, chunked sequentially when too large. A batch
is a TRANSPORT envelope, NOT state: it introduces no new record family, and no new owner of any fact.

1. **VOCABULARY.** `NotePrefixBatchSnapshot = "VBT_BATCH_SNAPSHOT:"` in `note_vocabulary.go`, with its catalogue
   entry (Scope=NoteScopeCheckpoint, Direction=NoteDirectionVaultToVault, description naming it the batch
   envelope). The init ambiguity assertion is automatic; the trailing colon keeps it unambiguous.
2. **THE PREFIX CHECK.** The writer refuses an undeclared prefix. It must accept a DECLARED FAMILY prefix OR the
   batch prefix, and nothing else, so the refusal stays a refusal.
3. **THE WRITER.** `saveRecordBatchLocked()` replaces the per-family dispatch INSIDE the cadence. It builds ONE
   payload `{"nonce": <seq>, "families": {"<family key>": <raw json>, ...}}`, gzips it ONCE, and encodes it with
   the EXISTING `encodeRecordNotes(batchPrefix, seq, payload)` - so the 1024 byte AVM cap, the fixed-point chunk
   count and the measured re-join are unchanged.
4. **THE NONCE.** The self-nonce stays a per-family SEQUENCE for ordering; the batch carries its own sequence and
   records the family sequences INSIDE the payload, so a missing batch is detectable AND each family is still
   ordered against its own history.
5. **THE READER.** `readRecordFamily(cfg, vault, prefix)` gains ONE fallback: when the family prefix yields no
   COMPLETE record, take that family slice from the NEWEST complete BATCH. It must REPORT which source answered,
   because a silent fallback hides a broken writer. An incomplete batch is a NAMED GAP, exactly as an incomplete
   nonce is today.
6. **WHAT DOES NOT CHANGE.** The vault identity, the ARC-200 self-transfer transport, the 1024 byte cap, the gzip
   envelope, the sender==vault assertion, the refusal to write an undeclared prefix, and RECORDS_DISPATCH OFF.

**FAILURE MODES TO PIN - each is silent state loss if wrong:**
- a family present in the batch but missing from the reader slice restores EMPTY and looks fine;
- a batch larger than the chunk count lets the AVM reject every note and nothing is written;
- a stale batch beating a newer family record - ordering must be by nonce SEQUENCE, never by timestamp;
- the batch and a per-family writer both writing one family is two owners of one fact.

**TESTS:** (a) a batch round-trips two families and the reader returns both WITH the source named; (b) a batch
split across N chunks reassembles and the re-join equals the source bytes; (c) a family record NEWER than the
batch wins; (d) an incomplete batch is a NAMED GAP, never a silent empty; (e) a prefix that is neither a family
nor the batch is still REFUSED by the writer; (f) the writer census stays green as its 16 baseline entries are
DELETED one per family landed - the baseline EMPTYING is the progress meter for A5.

**WHY IT MATTERS ECONOMICALLY:** the eleven in-memory audit records (A3) stay one transaction per event, because
they have no movement to ride on; batching is what keeps the SNAPSHOT side at ONE FEE PER CADENCE instead of one
per family.

---

## 16. THE $UNIT / $NUGGET PLACEHOLDER IDS (operator-authorised 2026-09-19)

Brendan authorised TEMPORARY placeholder asset ids on BOTH chains, to be replaced later. The binding safe design:
ONE owner declares them WITH an explicit `placeholder: true` marker; the boot announces them; the registry, the
shops and the wizard UI work fully; and a money door REFUSES to move real value while an id is a placeholder -
because a wrong id spends IRRECOVERABLY and is indistinguishable on chain from a right one. Replacing a
placeholder with the real id is a one-line change in that owner plus the boot announcement. This is a stated
limitation, not a shortcut: the plumbing is complete and testable, the VALUE MOVEMENT is refused.


---

## 17. A5 STEPS 4-5 - THE WIRING SPEC, AT FUNCTION LEVEL (measured 2026-09-19)

**THE API SURFACE AS IT ACTUALLY IS (READ, not assumed):**
- `readRecordFamily(cfg NetworkConfig, vaultAddr, prefix string) ([]byte, int64, bool, error)` is the ONE reader. Its body: `indexerCheckpointTransfers` -> per tx `isVaultCheckpointTransfer` + `parseRecordNote(prefix, tx.Metadata, tx.TransactionID, tx.Timestamp)` -> `assembleRecords(notes)` -> `(complete []recordSelection, gaps []recordGap)` -> the newest complete selection that `decodeSnapshotPayload(prefix, sel.Body)` accepts -> `recordNonceSeed(familyKey, sel.Nonce)` -> `return (data, sel.Timestamp, true, nil)`.
- `recordSelection{Nonce, Chunks, Body, Timestamp, TxIDs, Legacy}` is ONE COMPLETE record, and `Body` is still base64(gzip(data)).
- `parseRecordNote(prefix, metadata, txid, timestamp) (recordNote, error)` | `assembleRecords(notes) ([]recordSelection, []recordGap)` | `decodeSnapshotPayload(prefix, body) ([]byte, error)`.
- The nonce sequence: `recordNonceNext(familyKey)`, `recordNonceSeed(familyKey, seen)`, `recordNonceCurrentValue(familyKey)`.
- `recordSourceOrdering(familySeq, familyComplete, batchSeq, batchComplete) (string, bool)` is BUILT, with the clock pinned out by reflection.

**EDIT 1 - EXTRACT `readRecordSelections` (a pure refactor, NO behaviour change).**
`(l *Lobby) readRecordSelections(cfg, vaultAddr, prefix) ([]recordSelection, error)` holds everything from `indexerCheckpointTransfers` through `assembleRecords` plus the gap logging, returning the selections newest-first. `readRecordFamily` keeps its four-value contract by decoding the newest acceptable selection, so its THREE callers are untouched.

**EDIT 2 - `readRecordFamilyWithSource` (the fallback).**
`(l *Lobby) readRecordFamilyWithSource(cfg, vaultAddr, prefix) ([]byte, int64, bool, string, error)`: read the family selections, then ALSO read the batch selections, extract the family slice with `recordBatchFamily(payload, familyKey)` where `familyKey` comes from `RecordFamilyKeyForPrefix(prefix)`, and let `recordSourceOrdering` choose - REPORTING the source in the result. `none` with no error means the chain holds no record for this family, which is NOT an error: it is the honest empty state a fresh world has.

**EDIT 3 - THE WRITER `saveRecordBatchLocked(families map[string]any)`.**
Marshal each value once, refuse an undeclared family key (the payload owner already does), take ONE nonce with `recordNonceNext(NotePrefixBatchSnapshot)`, then `buildRecordBatchPayload` -> `encodeRecordNotes(NotePrefixBatchSnapshot, nonce, payload)` -> dispatch the chunks SEQUENTIALLY (the existing chunk loop already does). The cadence feeds it the families it owns (`economy`, `leaderboard`, `onboarded_wallets` today), so the per-family writes inside `saveSeasonMetadataLocked` are REPLACED, not duplicated. A family still written under its OWN prefix stays readable: the reader tries the family record first and the batch second.

**THREE THINGS THAT MUST NOT CHANGE:** the `sender == vault` assertion, the 8 MiB / 64 MiB bounds in `snapshot_guard.go`, and the refusal to write an undeclared prefix - which must accept the batch prefix BY NAME, never "anything that is not a family".

**TEST APPROACH (the pattern already in the tree):** `record_envelope_test.go` drives `readRecordFamily` against an `httptest` indexer, so the batch test does the same: serve notes built by `encodeRecordNotes(NotePrefixBatchSnapshot, seq, payload)` and assert (a) the family slice comes back, (b) the SOURCE is reported as batch, (c) a NEWER family record beats the batch, (d) a batch split across chunks reassembles, (e) an incomplete batch is a NAMED GAP.
