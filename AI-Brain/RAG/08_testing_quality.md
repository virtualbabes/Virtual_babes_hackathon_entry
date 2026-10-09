# 08 — Testing & Quality Gates

> **Scope:** All `*_test.go` files (51), the lock-order gate, self-lock gate, mutex-relock gate, signature verification suite, tenant vault security pin, career-path property tests, note-vocabulary parity instrument, and the mojibake census.
> **Architecture:** Tests are `//go:build !js && !wasm` — they run only against the server binary. No test runs in the WASM target. The gate tests are static-analysis instruments: they parse Go source with `go/ast` and assert properties over the tree, not over a running process.

---

## 1. THE GATE SYSTEM — STATIC ANALYSIS AS A SECURITY LAYER

The repo carries three lock-order gates. They are not integration tests — they parse the source tree with `go/ast` and report properties. A gate that matches nothing **fails** rather than reporting clean, so a blind spot is visible on every run.

### 1.1 `lock_order_gate_test.go` — Cross-Mutex Deadlock Detector (990 lines)

**THE cross-mutex gate.** The other two gates (`selflock_gate_test.go`, `mutex_relock_gate_test.go`) model ONE mutex each: they ask whether a function acquires a lock it already holds. Both are blind to the deadlock that needs TWO locks held in opposite orders. This gate covers that gap.

**The shape it models:**
```
Two goroutines that take A→B and B→A block forever.
An RWMutex queued writer blocks every later reader, so a reader-reader pair
that looks harmless in isolation is fatal as soon as any third goroutine
queues a write. Request paths in this tree hold the lobby WRITE lock across
their bodies, so a queued lobby writer is the NORMAL state.
```

**What it measures — one line:**
> Build the directed graph of ORDER edges (holding X, then acquiring Y) over the whole tree, and report every pair that appears in both directions.

**How it derives its subject (not curated):**
- Struct fields are read from declarations; each mutex identity is canonicalised to `<StructType>.<Field>` **through those declarations**.
- Without canonicalisation, `ace.mu` and `l.aiEngine.mu` are two unrelated strings and the analysis is meaningless.
- If the derivation finds ZERO mutex fields the gate **FAILS** — a detector that matches nothing reports a clean repository forever.

**Analysis limits (stated, not implied):**
| Limit | What it means |
|-------|---------------|
| MUST-HELD | `if`/`else`, `switch`/`select` clauses and loops are JOINED; a loop may run zero times, so it can only carry out what was already held |
| `defer` release | Does not clear the held state at its source line (happens at exit) |
| `go` literal | A new goroutine starts from an EMPTY held set |
| `defer`red CALL | Not analysed as an edge at all (LIFO — runs after the body) |
| ONE CALL EDGE | A callee's acquired receiver fields are resolved against the CALL-SITE receiver expression |
| Unresolvable type | Skipped AND counted; the census is printed on every run so the blind spot is visible |

**Measured on first run — exactly TWO inversions, both live and both now fixed:**
1. `AICitizenEngine.mu` ↔ `Lobby.mutex` — the behavioural tick held `ace.mu` for its whole body and took the lobby lock; `runAutonomousTournament` (15-minute daemon) and the Zen-Garden doors held the lobby lock and reached into the engine.
2. `Lobby.mutex` ↔ `TokenSinkRouter.Mu` — the persistence worker held the router lock and took the lobby lock; 34 fee-routing sites take the lobby lock and then `RouteCriminalTax` (which takes the router lock for writing — the side that decides the deadlock).

**Baseline is EMPTY** and stays that way by measurement, not by trust.

### 1.2 `selflock_gate_test.go` — Single-Mutex Re-Entry Detector

Asks whether a function calls a helper that takes **the lock it already holds**. Models one mutex. Blind to cross-mutex deadlocks (that is the gate's job).

### 1.3 `mutex_relock_gate_test.go` — Mutex Re-Lock Detector

Asks whether a body acquires a mutex it **already holds**. Models one mutex. The read+read variant was turned from a "note" into a failure when the tree's normal state (queued lobby writer) made the harmless-looking case fatal.

---

## 2. THE SIGNATURE VERIFICATION SUITE

**File:** `signature_verification_test.go` (74 lines)

Two negative-control tests. Each verifies that a real signature passes and every way of lying about it is refused. These are the **pin** for the blockchain dispatch gates: if the verifier accepts a tampered message, every record door that relies on it is unguarded.

### 2.1 AVM Primitive (`TestVerifyAVMSignatureAcceptsAndRefuses`)

Tests `verifyAVMSignature(addr, message, base64Sig)`:

| Case | Expected | Why |
|------|----------|-----|
| Real key, real message, real signature | **Accept** | The happy path must work |
| Same signature, tampered message (`message + "tampered"`) | **Refuse** | Signature is message-bound |
| Real signature, malformed address (`"0xnotanalgorandaddress"`) | **Refuse** | Must not guess at malformed addresses |
| Real message, non-base64 signature (`"!!!not-base64!!!"`) | **Refuse** | Input validation before crypto |
| Real message, signature from a DIFFERENT key | **Refuse** | Key-binding check |

The message format includes the tenant vault identifier: `"Algorand Signed Message:\nVirtualbabes Arena Tenant Vault:nonce-1"`.

### 2.2 EVM Primitive (`TestVerifyEVMSignatureAcceptsAndRefuses`)

Tests `verifyEVMSignature(wallet, nonce, hexSig)`:

| Case | Expected | Why |
|------|----------|-----|
| Real key, real nonce, real signature | **Accept** | The happy path must work |
| Same signature, different nonce (`nonce + "x"`) | **Refuse** | Nonce-binding check |
| Real signature truncated to 64 bytes | **Refuse** | Length validation |
| Real signature, different wallet's address | **Refuse** | Wallet-binding check |

The EVM path uses `ethcrypto.Keccak256(evmPersonalSignMessage(nonce))` as the signing hash.

---

## 3. THE TENANT VAULT SECURITY PIN

**File:** `tenant_vault_test.go` (72 lines)

`TestTheTenantVaultIsVaultOnly` is the **security pin** for the tenant vault registry. It asserts that a wallet the engine knows as a PLAYER cannot also be a tenant vault, and that the answer is case-insensitive (because the engine stores whatever spelling it was handed).

| Case | Expected | Why |
|------|----------|-----|
| Register `0xPLAYER` as vault (player wallet in lobby) | **Refuse**, message contains "PLAYER wallet" | A vault that also plays proves nothing |
| Register `0xplayer` (lowercase) as vault | **Refuse** | Check must be case-insensitive |
| Register `0xsame` as both tenant AND vault | **Refuse** | Vault must differ from tenant wallet |
| Register `""` as vault (no tenant) | **Refuse** | A vault with no tenant is meaningless |
| Register `"   "` as vault (blank) | **Refuse** | Blank addresses must be refused |
| Register with `nil` lobby | **Refuse**, not panic | Must guard nil, not crash |
| Register `0xVaultOnly` with tenant `0xTenant` | **Accept** | State = `tenantVaultStateProposed`, Proven = false |
| Network must be `"Voi"` | **Asserted** | Voi is the sole base chain |
| Duplicate `0xVaultOnly` (any spelling) | **Refuse**, case-insensitive | Duplicate vaults are meaningless |
| `Snapshot()` returns copies | **Asserted** | Mutating a snapshot entry must not change the registry |

**THE allowed path:** state = `tenantVaultStateProposed`, `Proven = false` — the proof is Stage A build 2. The registry does not trust the registration; it records a proposal that a verifier must land before the vault is usable.

---

## 4. CAREER PATH PROPERTIES

**File:** `career_path_test.go` (~575+ lines)

The strongest journey test in the tree. It walks each promotion gate ONE STEP AT A TIME so every refusal is attributable. Key properties:

| Property | How it is proven |
|----------|-----------------|
| Thresholds derived from engine constants | Each threshold read from the engine's own constants, not copied beside them |
| Path matrix vs career-pair matrix | Compared in BOTH directions: every declared enemy pair must CROSS the paths, every ally pair must SHARE one, matched count must equal `rivalMatrixEnemyPairs` |
| Fold's seven negative controls | Prove no accidental aliasing in the career fold |
| Civil rank is DERIVED | Two territories without an opened region stays `manager`; an alliance's territories count because `IsClubRegionalLocked` counts them |
| Demotion lifecycle pinned in all four states | Warning takes nothing; clock not re-stamped; expiry demotes with exact arithmetic; recovery `cleared` and settles |
| `CheckCareerTierGate` passes a funded player carrying a stale warning | While still reporting its age |

**Instrument limits recorded:** no locks are taken (behaviour only — the `...Locked` contract and the self-lock gate own concurrency); the enemy-pair prices live in COMMENTS beside a membership assertion, so a price change would drift the comment while the test still passes.

---

## 5. ROLE VOCABULARY — THE PIN FOR THREE REPAIRS

**File:** `career_role_vocabulary_test.go`

No production defect — it is the **PIN** for three defects repaired earlier: the `RoleKey` fold, the `isJusticeAligned` self-deadlock, and the unreachable `X-Client-ID` wallet.

| Property | What it enforces |
|----------|-----------------|
| Negative controls | `TestRoleKeyFoldsOneCareerAcrossEverySpelling`'s five `differ` pairs stop the fold from over-reaching |
| `TestEveryDeclaredPairHasAPrice` | Fails in **BOTH** directions — so the two unpriced pairs (`MutationLogAuditor↔Kidnapper` P2-D10, `JusticeRecruiter↔MutationLogAuditor` P2-D8) remain an operator-facing balance call recorded in code |
| 5-second watchdogs | The only instrument that can see a deadlock: it raises no error at all |

---

## 6. NOTE VOCABULARY — THE STRONGEST PARITY INSTRUMENT

**File:** `note_vocabulary_test.go`

No production defect — it is the **PIN**, and the strongest parity instrument in the tree. It derives its own subject from **SOURCE** (the constants in the owner file, every `VerifyBuyInTransaction` call site, every `Public/js/*.js`) and **FAILS if the derivation finds nothing**:

- `firstPartyGoSources` zero-files → fail
- <8-call-site floor → fail
- `scanned == 0` → fail
- A stale ambiguity acknowledgement → fail

**Its ONE hole:** neither the payload-shape test nor `assertNoteVocabularyPolicy` asserts TEXT ENCODING, so the 688 U+00E2 in `note_vocabulary.go` — including SERVED `Description` strings — passes the init panic, this file, and every repo gate. (See §15 of `19_coverage_ledger.md` for the mojibake census.)

---

## 7. CARD VIEW SKINS — THE BRANDING REFUSAL PIN

**File:** `card_view_skins_test.go`

No production defect — it is the **PIN**. Its value is the disjunction it enforces:

- `cardViewRefusesCardAlias` must be **FALSE for all three audience scopes** (so the layer's own vocabulary is usable)
- `cardViewRefusesCardAlias` must be **TRUE for eight entity spellings** (:66-73)
- `TestCardViewNeverWritesBrandingState` (:178-197) drives the §10.1 writer FROM the §10.5 feature and requires `BindBondedAssetToTarget` to refuse nine card kinds

**"The card-view layer cannot brand a card"** is therefore mechanical, not a reading. Instrument limits: refusal assertions are SUBSTRING matches, and every test builds a FRESH lobby, so no cross-lobby/aliasing hazard is observable from here.

---

## 8. THE MOJIBAKE CENSUS

**Source:** §15 of `19_coverage_ledger.md`

Every `.go` file measured at byte level for three levels of double-encoding damage:
- `U+00E2` = one level (UTF-8 sequence re-read as CP1252)
- `U+00C3`/`U+00C2` = second level (the `Ã¢â‚¬` family)

**Ten damaged files** (not three — the corruption tracks the edit history of the record/cadence work):

| File | L1 (U+00E2) | L2 (U+00C3) | U+00C2 |
|------|-------------|-------------|--------|
| `economy_service.go` | **34,368** | **68,273** | **30,652** |
| `note_vocabulary.go` | 688 | 0 | 7 |
| `checkpoint_indexer_read.go` | 326 | 316 | 305 |
| `record_families.go` | 209 | 0 | 4 |
| `faith_church.go` | 108 | 0 | 1 |
| `bonded_asset_registry.go` | 57 | 24 | 39 |
| `record_families_test.go` | 34 | 108 | 67 |
| `ai_citizen_engine.go` | 30 | 0 | 47 |
| `local_model_promotion.go` | 8 | 0 | 14 |
| `item_shop_archetype.go` | 3 | 0 | 16 |
| `console_server.go` | 3 | 0 | 1 |

**`economy_service.go` carries the overwhelming majority** — 133,293 damaged sequences in one file. Line 234 is a 243,565-character `//` COMMENT (~62% of the file), re-encoded through CP1252 misinterpretation **four times**. The rest of the corpus (144 `.go` files) measures clean.

**The remaining 144 `.go` files are clean** — this is not a reader artefact and not repo-wide damage. It tracks the edit history of the record/cadence work specifically.

---

## 9. THE TEST COUNT BY CATEGORY

| Category | Count | Example files |
|----------|-------|---------------|
| Lock-order / deadlock gates | 3 | `lock_order_gate_test.go`, `selflock_gate_test.go`, `mutex_relock_gate_test.go` |
| Signature verification | 1 | `signature_verification_test.go` |
| Tenant vault security | 1 | `tenant_vault_test.go` |
| Career path / role vocabulary | 2 | `career_path_test.go`, `career_role_vocabulary_test.go` |
| Note vocabulary parity | 1 | `note_vocabulary_test.go` |
| Card view / branding refusal | 1 | `card_view_skins_test.go` |
| Bonded asset / market | 5 | `bonded_market_test.go`, `bonded_branding_test.go`, `bonded_asset_purchase_test.go`, `bonded_asset_capacity_test.go`, `market_service_test.go` |
| Record / checkpoint transport | 8 | `record_envelope_test.go`, `record_batch_test.go`, `record_families_test.go`, `record_transport_test.go`, `record_note_gate_test.go`, `record_batch_read_test.go`, `record_batch_prefix_test.go`, `record_family_writer_gate_test.go` |
| Entity investment / dividend | 3 | `entity_investment_lock_test.go`, `revenue_distribution_lock_test.go`, `ai_citizen_persist_test.go` |
| Lock-order watchdogs | 2 | `lock_order_watchdog_test.go`, `mutex_relock_gate_test.go` |
| Slide theming | 1 | `slide_theming_test.go` |
| Admin / economy | 2 | `admin_power_scaling_test.go`, `market_nodes_alias_gate_test.go` |
| Rivalry / XP arithmetic | 1 | `rival_xp_arithmetic_test.go` |
| Snapshot / guard | 2 | `snapshot_guard_test.go`, `mutex_relock_gate_test.go` |
| Placeholder / asset | 2 | `placeholder_assets_test.go`, `placeholder_derivatives_test.go` |
| Identity / bridge | 2 | `identity_bridge_test.go`, `arc200_read_test.go` |
| Network / config | 2 | `networks_config_test.go`, `network_registry_view_test.go` |
| Other | 3 | `map_race_gate_test.go`, `txid_memo_test.go`, `selflock_gate_test.go` |

**Total: 51 test files** (measured at filesystem level, not estimated).

---

## 10. WHAT THE TESTS DO NOT COVER

| Gap | Why it matters |
|-----|---------------|
| No WASM-target tests | All tests are `!js && !wasm`; the client WASM has no test suite |
| No integration tests against a live server | Gates are static analysis; no test boots a server and exercises HTTP routes |
| No mojibake test | The note-vocabulary test does not assert text encoding, so the 688 U+00E2 in `note_vocabulary.go` passes every gate |
| No cross-file type parity test | `common_types.go` vs `common_types_wasm.go` drift is not mechanically detected — it is recorded in `19_coverage_ledger.md` as a finding, not tested |
| Enemy-pair prices live in comments | `career_path_test.go`'s price assertions are beside a membership assertion in comments; a price change drifts the comment while the test still passes |

---

*End of 08 — Testing & Quality Gates. Generated from measured filesystem + source reads of the 51 test files and the three gate instruments.*
