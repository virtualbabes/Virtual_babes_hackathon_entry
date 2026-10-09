# CONSOLE DLC STORE & VOUCHER-BRIDGE COMPLETION — PLAN (prepared 2026-09-20)

> **Owner:** this file. It exists so the console economy work can be picked up without re-deriving the model.
> **Status:** **PREPARED — nothing built.** Written from measured code plus the operator's own description of the model.
> **Why now:** the README advertises this as the strongest unbuilt part of the architecture — the console DLC store, the
> voucher bridge, and the mirrored economy that makes a crypto game shippable on a locked platform.

## 1. THE MODEL, AS MEASURED (not as remembered)

| Piece | What exists | Where |
| --- | --- | --- |
| **The virtual ledger** | `playerBalances` is the **virtual liability ledger** — every player's spendable balance lives there | `faucet_service.go:382-386` |
| **The mirror is exact** | a trade between players moves `$VBV` **inside that ledger** while the **physical vault balance is unchanged** | `auction_service.go:240` |
| **Console reward representation** | `ArenaVouchers` — **non-crypto**, a declared liability; the code says it must be *converted*, never spent as crypto | `common_types.go:445`, `economy_service.go:60`, `faucet_service.go:384` |
| **The bridge** | `HandleVoucherConversion` converts vouchers into the virtual ledger — and it lives in `onboarding_service.go`, the file the console build **excludes by ruling** | `onboarding_service.go` |
| **Vault payouts** | the claim path reads the virtual balance, dispatches on-chain, applies an **exit siphon of 2%**, and **rolls the balance back** if the chain leg fails | `faucet_service.go:454-589` |
| **Redemption gateway** | consumes a voucher against a `DLCRegistry` product, routes the fee, buys the market to pay the creator, decrements stock | `redemption_gateway.go` |
| **The receipt contract** | `ConsoleAssetReceipt` — *"a secure purchase or lease confirmation from an external platform"* — **its WRITER LANDED 2026-10-08 (§10.11)**: a verified platform purchase grants an entitlement, keyed by the platform purchase id | `common_types.go:543` + its WASM mirror · **`console_entitlement.go`** (the one owner) |
| **Console segregation (the proof)** | the console entrypoint holds **0** signing calls, **0** application-call constructions, **0** raw submissions, **0 of 5** chain-rail routes, serves no HTML roots, binds loopback | `console_server.go` |

## 2. WHAT IS MISSING (the actual work)

1. **The store itself** — no purchase surface, no product browsing, no purchase→entitlement flow on the console side.
2. **The USDC TOP-UP rail** — refined by the operator's model (2026-09-20): DLC, shops and assets are bought with
   **virtual balance** (instant payout to the seller, exactly as on PC), and virtual balance is bought as a **top-up with
   USDC that lands in the admin wallet**. **None of it is built.** Two notes measured while preparing this: the only
   "USDC" string in the tree is a **mislabelled comment** — `economy_processing.go:268-278` "Administrative USDC Siphon"
   actually routes **10% of the `$VBV` tax payload** into an `AdminMaintenancePool` when faucet coverage exceeds 150% —
   and the USDC asset ids are documented but **commented out** in `.env.example`
   (`VOI_NATIVE_USDC_ASSET_ID=34426428`, `ALGO_NATIVE_USDC_ASSET_ID=31566704`).
3. **The cap** — by standing ruling the rail **ships with its cap or not at all**. The cap value is the operator's call.
4. ✅ **BUILT 2026-10-08 — the writer for `ConsoleAssetReceipt`** (`console_entitlement.go`, §10.11): the platform→server
   direction now has a door, a verifier contract, an idempotent grant, a record family and a read path. What it still
   needs is **platform credentials** — `CONSOLE_VERIFIER_URL_<PLATFORM>` and its secret, which only the operator can name.
5. **A console client surface** — not verified to exist; today the console server is a headless loopback authority.

## 3. THE OPERATOR DECISIONS THIS IS BLOCKED ON (none of these may be guessed)

| # | Decision | The consequence of guessing it |
| --- | --- | --- |
| 1 | ✅ **The cap value — ANSWERED 2026-09-20** | the cap is **derived from the vault's holdings**, not fixed: a sale must fit inside the vault's usable surplus *after* every obligation and every other active listing — see **§7**. A solvency rule cannot bounce. |
| 2 | ✅ **Where the USDC lands — ANSWERED 2026-09-20** | the **admin wallet** — `ADMIN_WALLETS` already exists, is signature-gated and is served as `admin_wallets_configured` |
| 3 | ✅ **The payout shape — ANSWERED 2026-09-20** | **instant**, exactly as PC shop/asset sales: the buyer's virtual balance moves and the seller is credited in the same step (the `auction_service.go:240` shape) |
| 6 | **Platform native-DLC payout mechanics and commission** — **open research item**, deliberately not estimated | the top-up path exists so the store does **not** depend on platform payouts; guessing a commission would misprice every sale |
| 4 | **Refund / chargeback policy** | platform store rules are not on-chain finality |
| 5 | **Which platform store is first** (one, or all three) | decides the receipt and reconciliation surface |

## 4. PROPOSED BUILD ORDER (each step leaves the tree green)

1. **Design-only first, in code:** a served `consoleStoreContract()` reporting `available:false` and **naming every
   blocker** — exactly the discipline the tenant-lease catalogue already uses. Nothing may pretend the store works
   before it does.
2. ✅ **DONE 2026-10-08 — `ConsoleAssetReceipt` writer + reader** (§10.11): one issuing function
   (`GrantConsoleEntitlement`), one read path, round-trip **and** idempotency tests, and the receipt carried by the record
   set as the **23rd primary family** (`console_entitlements`, dispatched through the A5 transport mirror).
3. **The purchase → entitlement flow over the virtual ledger** (no USDC yet): `$VBV` purchase → entitlement → receipt.
   Fully testable end to end without touching an external payment rail.
4. **The USDC rail, with its cap, as ONE change** — cap, charge, creator payout through the market, audit record.
5. **The console store surface** — only once 1–4 hold.

## 5. ACCEPTANCE (what "done" means)

* The console target still holds **0** signing calls and **0** chain-rail routes — a store must not breach the
  segregation it exists inside.
* A purchase produces a **receipt a reader can fetch**, and the creator is paid in `$VBV` with the fee arithmetic
  **asserted to reconcile before any write**.
* The cap refuses an over-limit purchase **before** a charge, with the reason served.
* Every refusal in the flow is proven to move **no** state, and the path carries a test that fails **by timeout** if a
  lock is ever held across the payment call (this repository has found that class eight times).

## 6. RISKS RECORDED UP FRONT

* **The receipt type being unwritten** means the browser→console direction currently has no proof of entitlement — and a
  console platform will ask exactly what proves a purchase.
* **USDC custody** introduces a real-world custody question this app has so far avoided by design (it holds no user key).
  Whoever holds the USDC wallet is a decision with legal weight, not a configuration value.
* **The 2% exit siphon and the rollback** are existing behaviours the store must not bypass or duplicate.
* **The virtual ledger is a liability** (`economy_service.go:60` sums it) — the store must keep that accounting true, or
  the console economy silently stops reconciling.

## 7. THE SOLVENCY-DERIVED CAP — OPERATOR RULING (2026-09-20)

> *"the cap value would need to derive from the faucet vault holdings, it needs to ensure that the faucet vault has
> enough supply to pay out the sale as well as all other active listings, this would stop the vault from ever bouncing
> the transaction"*

**The cap is not a number. It is a computation — and this repository already owns every input.**

### The arithmetic that already exists (integer, no float on the path)

`economy_service.go:44-92` already computes exactly this, for reward scaling. It is the model to follow:

```text
totalLiabilities = Σ playerBalances                 (the virtual ledger)
                 + Σ ArenaVouchers                  (console — non-crypto)
                 + Σ RecoveryBounties
                 + Σ BountyHunterBondMicro

committed        = tournamentPotBonusMicro + pendingTournamentPayoutsMicro
                 + tournament.PotMicro              (only while a tournament is Active)

unusable         = gasFloorMicro (1.0 unit)
                 + totalLiabilities
                 + Σ club treasuries
                 + committed

usable           = faucetMicro − unusable           (REFUSED if that would go negative)
```

`lobby_manager.go:2232-2296` computes a richer version of the same census and **serves it as
`TotalVirtualLiability`**, including **district dividend pools** (`districtNode.DistrictDividendPool`).
`handlers_admin.go:1466-1501` already audits **coverage**: `physicalBalance >= totalLiabilities`, a **coverage ratio**,
and a **net surplus**.

### The rule for the store

1. **A payout may only ever be promised out of `usable`.** The store refuses a listing whose `$VBV` obligation does not
   fit inside the vault's usable surplus.
2. **An active listing IS an obligation.** Add a **reserved-payout accumulator** (`dlcListingReserveMicro`) and include
   it in the **same** census — so reward scaling, the admin coverage audit and the served liability figure all see the
   store's promises automatically. A second private sum would be a second owner, and it would drift.
3. **Never promise against a balance you could not read.** If `vaultBalanceLive` is false (the last on-chain read
   failed), the cap is **0** and the store refuses *with that reason*. An unreadable vault is not an empty vault.
4. **Reserve at listing, release at settlement or cancellation** — the same integer lifecycle as every other obligation
   here, so withdrawing a listing returns its headroom.
5. **Integer only, in `uint64` micro-units** — computed in the ledger's own units. A float anywhere on this path is the
   class the Architecture Ledger forbids.

**The consequence, stated plainly:** with the cap derived this way, a sale cannot take the vault below its own
obligations, so **the payout cannot bounce** — the refusal happens at *listing* time, with a reason, before any money
moves. That is a solvency property, not a spending limit.

**And the same figure is the TOP-UP ceiling (§8):** when the vault sells its own `$VBV` for USDC, it draws on `usable` —
and because the buyer's balance rises by exactly what the vault's free stock falls by, **the sale is coverage-neutral**.
Stock moves; coverage stays whole.

## 8. THE TOP-UP MODEL — THE VAULT IS A SELLER, NOT A MINTER (operator, 2026-09-20)

**The operator's model, in his words:** *"virtual sales on console are instant pay outs just like the pc for the shops and
assets … the app will sell $VBV top ups for Virtual balance to allow users to convert to the apps payment method, the top
ups will cost USDC that will go to the admin wallet …"* — and, correcting this plan's first framing:

> *"we arent minting virtual we are transfering, the faucet is selling its own $VBV, yes it should not sell what it does
> not have and will need to expose that the certain $VBV top up package size is unavailable if that is the case."*

### The flow — a SALE, not a mint

```text
   console player                    the app                       the operator
   --------------                    -------                       ------------
   buys a TOP-UP  ---- USDC -------> payment system ---- USDC ----> ADMIN WALLET
        |                                                                  |
        +-- VIRTUAL BALANCE rises  <-- the VAULT SELLS $VBV IT ALREADY HOLDS
            (the vault's free stock falls by exactly that amount)
   then SPENDS virtual balance on DLC / shops / assets
        +-- seller or creator credited INSTANTLY, in the same step (as on PC)
```

**Nothing is created.** The vault transfers `$VBV` it already holds into the buyer's balance: the buyer's balance rises by
exactly what the vault's unencumbered stock falls by. The sale is therefore **coverage-neutral by construction** — which
is why the `usable` figure from §7 is the correct ceiling for it and why **no new accounting is needed**.

### The rule: never sell what you do not have — and SAY SO

1. **The sellable stock is `usable`** (§7): the vault's own holdings minus every obligation and every active listing. A
   top-up may only be sold inside that figure.
2. **Expose availability per package.** The top-up catalogue must state, for each package size, whether it can be sold
   **right now** — and when it cannot, **the shortfall is named**. This is the convention this repository already uses
   everywhere: the placeholder catalogue serves `purchasable:false` with its reason, the lease catalogue serves
   `creation_available:false` + `creation_blocked_by`, and the reward registry prunes **and reports**. A package that
   looks buyable and then fails is the one outcome that is not allowed.
3. **Never sell against a balance you could not read.** If `vaultBalanceLive` is false, sellable stock is **0** and the
   catalogue states why. An unreadable vault is not an empty vault.
4. **A native console DLC payout is a SEPARATE rail.** Platform payouts are slow, commission-heavy and may arrive as a
   bank transfer, so the app's own top-up path is the primary route and the platform's payout is a **reconciliation
   item**, never a dependency of the store.
5. **In-game sales need no new solvency rule.** Buyer to seller is a transfer *inside* the virtual ledger (the
   `auction_service.go:240` shape), so liabilities are unchanged and the credit is instant.

### Open research item — ✅ **ANSWERED 2026-10-08, see §10**

**How each console platform pays out native DLC, and what it takes in commission** — researched by direct source fetch
and recorded in **§10**. Summary of what the public record supports: **Microsoft publishes its payout MECHANICS in detail**
(payments released by the **15th of the maturity month**; a **30-day** hold on credit-card orders; a validated payout/tax
profile required before payment) but **not** its revenue-share percentage; **Sony publishes nothing** (its portal returned
an identical 4,539-byte shell on two paths) and **Nintendo publishes no money terms** (the publishing agreement is behind
an NDA). **No commission figure is asserted**, so the store's numbers remain derived from the vault's solvency (§7) and
the top-up stock (§8), exactly as ruled. **§10.3 also adds an external policy constraint that reinforces the standing
ruling:** Microsoft Store Policies §11.14 defines real-world gambling to include *"any payout of winnings which can be
converted into items of real-world value"* and prohibits it in the US and eight other markets — so **a console build must
not carry a rail that converts winnings into real-world value.**

## 9. BRIDGE AVAILABILITY — OFFLINE, NOT FAILED (operator, 2026-09-20)

> *"if the vault balance is to low to bridg a player into pc or do a bridge sync the app should expose that the bridge is
> temp offline awaiting maintanence"*

**A bridge that cannot complete must not be offered.** This is an availability state, not an error message: the surface
says it is offline *before* a player commits, so nobody begins a sync that cannot finish.

### What the bridge surface must serve

| Field | Meaning |
| --- | --- |
| `bridge_available` | boolean — may a bridge be started right now? |
| `status` | `online` · `offline_maintenance` · `offline_unreadable_vault` |
| `reason` | the human reason — e.g. *"temporarily offline, awaiting maintenance"* |
| `resume_hint` | what clears it: the vault being supplied, or the chain read recovering |

### The three triggers

1. **The vault's free stock is too low** to cover the bridge — the same `usable` figure as §7 and §8.
2. **`vaultBalanceLive` is false** — an unreadable vault is **offline**, not empty. The same rule the top-up catalogue
   follows, for the same reason.
3. **The operator's maintenance mode is on** — `l.maintenanceMode` (`backend_types.go:603`), set by
   `handleMaintenanceMode` (`handlers_admin.go:808-823`; route `/api/maintenance-mode` in **both** servers) and already
   **served to clients** as `Maintenance:` (`handlers_public.go:492`) and `MaintenanceActive:`
   (`lobby_manager.go:2289`), with the engine informed through `SetMaintenanceState` (`main.go:1623`).

**ONE OWNER, deliberately:** this reuses the existing maintenance mechanism and adds **no second flag**. A separate
"bridge offline" boolean would be a second owner of the same fact, and the two would eventually disagree — the failure
mode this repository has recorded more than once.

### The in-flight case is ALREADY safe — which is not a reason to skip this

`faucet_service.go:549/559` **rolls the virtual balance back** if the on-chain leg fails, so an interrupted bridge returns
the player's balance rather than stranding it. This requirement governs the **entrance**: a player must never begin a
bridge the vault cannot complete, and the refusal has to read as **maintenance**, never as a rejection of their balance.

## 10. THE PLATFORM PAYOUT RESEARCH (measured 2026-10-08) — ANSWERS §8's OPEN ITEM

> **Method:** every line below was retrieved by fetching the URL named beside it, or is an explicit statement that the
> content **could not be retrieved**. Where a platform publishes nothing, that is recorded as a **measured absence** —
> never filled in with a remembered figure. **No revenue-share percentage is asserted anywhere in this section**, because
> none could be verified against an openable source.

### 10.1 What each platform actually publishes

| Platform | Public without an account | Gated | Source (fetched) |
| --- | --- | --- | --- |
| **Microsoft — payout mechanics** | ✅ **detailed**: when payments are released, the credit-card hold, the payout/tax profile requirement, withholding, which roles can see a statement | the **percentage** the developer keeps on Store/Xbox game sales | `learn.microsoft.com/en-us/partner-center/marketplace-offers/payout-faq` |
| **Microsoft — Xbox program** | the three hard requirements only | **the money terms (NDA)** | `developer.microsoft.com/en-us/games/` (FAQ) |
| **Microsoft — Store Policies** | ✅ **full policy text (v7.20)** | — | `learn.microsoft.com/en-us/windows/apps/publish/store-policies` |
| **Sony / PlayStation** | ❌ **nothing** | **everything** — the portal returns an identical 4,539-byte shell on both `/` and `/support`, containing only "PlayStation® Partners" | `partners.playstation.net` (two paths measured) |
| **Nintendo** | program facts (free registration, pricing freedom, free IARC rating, no company required, 18+) | **the money terms — NDA + ToS acceptance, then a separate Switch application** | `developer.nintendo.com/faq`, `/the-process` |

**Nintendo's public pages state their own absence:** `/the-process` names the money step as "**sign a publishing
agreement**" and publishes no figure; `/faq` names the only cost as **development hardware** and states that "**the price,
release date and content are all set by you**". Registration and tools are "**completely free**".

### 10.2 Microsoft's payout mechanics, verbatim (the one genuinely open rail)

From the payout/tax FAQ:

* "**Microsoft releases payments by the fifteenth day of the maturity month.**"
* "For orders paid by credit card, Microsoft holds payments for **30 days**, until the earning has matured."
* "An enrollment is considered complete **only after Microsoft validates your payout and tax profile**." — and a profile
  that lapses shows "**Action required - Update bank and/or tax profile**".
* "**Withholding tax is applicable for U.S. publishers who filed a W-9 form.** Withholding tax is calculated on a monthly
  payment."
* A **payout statement** requires the **Owner** or **Financial Contributor** role (Earnings workspace).

**One attribution trap, stated because it is easy to misread:** the same FAQ's *Marketplace payout policies* section says
"**The store service fee is 3%.**" That is the **commercial Marketplace** program (Azure/AppSource offers) — **not** the
consumer Microsoft Store's app/game revenue split. They are separate programs, and this plan moves no number between them.

### 10.3 THE FINDING THAT CHANGES THE DESIGN — Microsoft Store Policies §11.14, verbatim

> **11.14 Gambling Apps**
> Apps that process real-world gambling transactions must: Be an app. Be rated with an 18+ age rating. Use a secure
> third-party payment API to process these transactions. **Real-world gambling is not permitted in the following markets:
> Brazil, Chile, China, Russia, Singapore, Taiwan, United States of America, Republic of Korea, and India. Real-world
> gambling includes any payout of winnings which can be converted into items of real-world value.**

**Why this is material to this repository, not a general disclaimer.** This product has **wagers** (card-match bets,
spectator wagering on matches, tournament buy-ins, entity and pet events) **and** a `$VBV → voucher → real value` bridge
**and** a planned `$VBV → USDC` console rail. The policy's own definition is *conversion*, so the risk is created by the
**rail**, not by the wager. **The research therefore CONFIRMS the ruling already in force** — the console entrypoint
registers **0 of 5** chain rails and the voucher bridge lives in the file the console build excludes — and that ruling now
has a **named external policy** behind it: **a console build must not carry a rail that converts winnings into real-world
value.**

**Three adjacent sections to read directly before shipping** (all appear in the v7.20 table of contents; their text fell
outside the fetched window on **two** attempts — the page truncates its middle at ~42.5 KB — so they are **recorded as
unread, not as absent**): **§10.8 Financial Transactions**, **§10.13 Gaming and XBOX**, and **§11.16 Live Generative AI
Content** (relevant to the AI-citizen / bot layer). **§11.13 Third Party Digital Storefronts Content** WAS retrieved: if
the console build exposes the player-to-player market, auctions or the entity market, the product must publish ToS and
content guidelines for listed items, provide reporting, review and enforce, age-gate higher-rated content, and comply with
storefront law.

### 10.4 What this does to §7's cap — one new term, not a redesign

A native-platform DLC sale is **an unmatured receivable**: by Microsoft's own wording the money is released **by the 15th
of the maturity month**, and credit-card orders are held **30 days**. So when a console player buys native DLC through the
platform, **the vault does not yet hold that money.**

**Therefore `usable` gains one explicit exclusion: platform receivables are NOT usable until settled.** Writing a storable
obligation against the platform's unpaid proceeds is precisely the bounce §7 was written to prevent, and it stays
invisible unless it is named. Everything else in §7 stands exactly as the operator ruled it.

### 10.5 Two preconditions that belong in the served contract (operator-account facts, not code)

| Field | Meaning | Why it is served |
| --- | --- | --- |
| `payout_profile_valid` | the platform has validated the payout + tax profile | Microsoft's FAQ makes enrolment *incomplete* until then — a store that cannot say this looks like it is holding money silently |
| `minimum_payout` + `expected_settlement` | the platform's floor and its own settlement day | a platform minimum can mean a small balance never pays; that must read as a stated limit, exactly as §8 requires a shortfall to be named |

### 10.6 The rule this research leaves behind

**Unknown is not zero.** No commission, delay or threshold is assumed anywhere in this plan; the store's numbers remain
derived from vault solvency (§7) and top-up stock (§8). When the operator reads the real figures in-account, they enter
this section **as a stated figure with its source** — never as an estimate promoted to a fact.

### 10.7 Where the operator reads the real numbers (nothing here is a substitute)

1. **Microsoft:** Partner Center → **Earnings** workspace, and **Account settings → Payout and tax profiles** (needs
   Owner or Financial Contributor). The public FAQ (§10.2) is the key to what those screens show.
2. **Xbox (console):** after signing the **NDA** the developer-portal FAQ requires — the money terms are not public.
3. **Sony / PlayStation:** **PlayStation Partners** after sign-in. **Nothing is published** (measured twice).
4. **Nintendo:** after registration, **NDA + ToS acceptance**, then the separate Switch application — the publishing
   agreement is where the money is settled.

### 10.8 Method note, so the next session does not repeat it

DuckDuckGo's HTML endpoint answered with a **bot challenge** and Bing returned only localized consumer storefront links, so
**search engines contributed no figure** to this section. The productive route was **direct source fetches** plus
**Microsoft Learn's public search API** (`learn.microsoft.com/api/search?search=…&locale=en-us`), which returns real page
URLs as JSON. One Microsoft-hosted *ifdef WINDOWS* show page was found describing "the new revenue share coming to the
Microsoft Store" (`aka.ms/ifdef-new-rev-share`) but **carries no figure in its retrievable text** — recorded as a lead,
not a fact.

### 10.9 THE FIVE PLATFORM PLANS (one owner per platform)

The platform-specific work is **NOT** in this file — it lives in five companions, so each platform has exactly one owner:

| Plan | Covers |
| --- | --- |
| **`PLATFORM-XBOX-PLAN.md`** | PC self-service vs managed console paths · the **in-game-store-only add-on mechanism** (Durable/Consumable via **XStore APIs**, priced in Partner Center) · the **XR** requirements that bind us (incl. **XR-042 No Real-Money Cash-Out**) · ratings and the Japan exclusion · workstreams mapped onto this repository |
| **`PLATFORM-PLAYSTATION-PLAN.md`** | the **measured gate** (no public information on five paths) · the **six in-account reads** that decide everything · the shared IARC position · what our architecture already gets right |
| **`PLATFORM-SWITCH-PLAN.md`** | the public **process / eligibility** facts · the **six in-account reads** · the **porting question** (Go/WASM is not a named Nintendo environment; Unity and native C++ are) · shared vs platform-specific workstreams |
| **`PLATFORM-ANDROID-PLAN.md`** *(added 2026-10-08)* | **the primary mobile target.** Google's **published service-fee matrix with its 2026 rollout dates** and the priced **external-web-link** route · the **Payments policy** (billing required for *"virtual currencies"*, **forbidden** for *"peer-to-peer payments"* and *"content that facilitates online gambling"*) · the **Real-Money Gambling** policy (*free, no In-app Billing, AO rating, licence per territory, minors and unlicensed geographies blocked*) · the **US$25** one-time registration and the personal-account testing rules · **ten named unread policies** |
| **`PLATFORM-APPLE-PLAN.md`** *(added 2026-10-08)* | **LOW PRIORITY — NON-ESSENTIAL — DROPPABLE.** Its **§0 is a drop test**: if iOS is never shipped, *nothing else changes* and no shared component loses its justification. Guideline **3.1.5** crypto verbatim (wallets require **organization** enrolment) · the published **15% / 10%** commission figures · the **3.1.3(f)** companion-app carve-out · the **Korea GRAC** and **France 18+** rating overrides · **five unread guidelines named as gates** |

**One rule across all six documents:** the **store catalogue and the entitlement path have ONE owner** (this file owns the
economy; the platform plans own only that platform's commerce). **No platform's revenue share is ever estimated** — an
invented cut would silently corrupt §7's cap — **and where a platform does publish a figure, it is recorded with its source
and its date** (Google publishes a dated fee matrix; Apple publishes **15%** and **10%** but *not* its standard rate), never
averaged into another platform's.

### 10.10 What the mobile research adds to the economy model (2026-10-08)

Two more stores were researched **after** §10 was written. They changed **one** thing in the model and **confirmed** another.

1. **CONFIRMED — AND NOW BACKED THREE TIMES OVER, by policies never read against each other.** The segregation recorded in
   §10.2 is independently arrived at by: Microsoft's published prohibition (**XR-042 / Store Policies §11.14**); Google's
   **Payments** policy, which *forbids* its own billing for *"peer-to-peer payments"* and *"content that facilitates online
   gambling"* while *requiring* it for *"virtual currencies"*; and Google's **Real-Money Gambling** policy, which requires
   such an app to be **free, with no In-app Billing, AO-rated, licensed per territory and geo-blocked**. **The last two
   cannot both be satisfied**, so the classification must be **avoided by structure**: **the mobile build carries no
   real-value exit.** That is a design fact, not a preference — and it is the same shape the console already has.
2. **ONE NEW TERM, the same term as §10.4.** A **mobile store sale is an unmatured receivable too**: the store deducts its
   fee and remits on its own schedule, so **`usable` excludes mobile-store proceeds until settled**, exactly as it excludes
   platform receivables. **No new formula — one exclusion, applied per platform.**
3. **What does NOT change:** §7's vault-solvency cap, §8's top-up model, and the operator's ruling that a top-up is a **sale
   of the vault's own $VBV**. Neither mobile store publishes anything that touches them, and **no mobile figure was used to
   re-derive them** — the one number that could have tempted a guess (Google's fee cell for a top-up) is recorded as an
   in-account read instead.

## 10.11 WORKSTREAM C IS BUILT — THE ENTITLEMENT PATH (2026-10-08)

`ConsoleAssetReceipt` had been **declared since the Phase 4 expansion and written by nothing** — no handler, no reader — so
the platform→server direction was a contract rather than a rail. It now has **one owner**: **`console_entitlement.go`**.

**The flow it implements, end to end:**

```
the platform sells a DLC product
  -> its fulfilment call carries a ConsoleAssetReceipt + the platform purchase id
  -> the server asks the PLATFORM'S OWN API for its record of that purchase
  -> the record is cross-checked against the claim
  -> the entitlement is granted, idempotently, keyed by the platform purchase id
```

**Three rules, each pinned by a test:**

1. **A receipt is never trusted because a caller sent it.** The product, the console account, the platform, the lease
   shape and the duration all come from the **platform's record** and are cross-checked against the claim. A caller may
   name a **platform purchase id and nothing else** (`DisallowUnknownFields` makes that structural): there is no price
   field, no wallet field and no duration field to send.
2. **The platform collected the money, so this path moves none.** No balance, voucher, faucet credit or ledger counter is
   touched — the grant is **ownership**, and settlement with the platform is the operator's rail (§10.4: a platform sale is
   an **unmatured receivable**). A test asserts balances, vouchers and inventory are byte-identical after a grant.
3. **Idempotent through the ONE memo door** (`txid_memo.go`): the purchase id is claimed **before** the platform call and
   committed **only after** the grant, so a replayed fulfilment grants once — and a **refused** attempt releases its
   reservation, so a genuine retry is never blocked for the life of the process.

**The verifier contract (the honest core).** A platform verifier is a function that returns the **platform's own record**;
the default table is **EMPTY**, because this repository holds no platform credentials. The one shape bundled is HTTP:
`CONSOLE_VERIFIER_URL_<PLATFORM>` + `CONSOLE_VERIFIER_SECRET_<PLATFORM>` (registered at boot; the count is **stated in the
log**). **With nothing configured the door REFUSES with 409 and names the variable it needs** — the shape the lease
catalogue already uses. Inbound, `CONSOLE_FULFILMENT_SECRET` is compared **in constant time** and **fails closed** (an
unset secret authorizes nobody). So **two independent proofs** are needed: one inbound, one outbound.

**State and the record set.** Grants live in `ConsoleEntitlementRegistry` (`console_entitlements.json`), keyed by the
purchase id — so "granted once" is a property of the map, not of a caller's discipline. It is the **23rd primary family**
(`console_entitlements`, `VBT_CONSOLE_ENTITLEMENT_SNAPSHOT:`) and its mirror is dispatched from the same private snapshot
the file write uses (the A5 transport-mirror pattern).

**The asymmetry, stated rather than implied.** `POST /api/console/entitlement` is registered **on the primary server
only** — like the voucher redemption gateway beside it — because the console build holds **0 chain rails by ruling**, so a
console-local grant would be a virtual-mirror record that never reaches the account. The route is therefore **baselined
exempt with that reason** in `route_parity_baseline.json`. The **read** half (`GET /api/console/entitlements`) **is**
registered in both servers: a console build may *see* its entitlements without transacting.

**What the served contract states (and what is still blocked):** `available`, `verifiers_configured`, per-platform
`verifier_url_env`/`verifier_secret_env`, and a `blocked_by` list that always carries the **structural** blockers —
(i) **lease expiry is recorded but enforced nowhere** (no block clock is wired), and (ii) **the creator payout for a
platform sale is the operator's settlement rail: this path credits nobody**, because the platform has not settled.

**Verification.** 8 new Go tests (10 refusal sub-cases) all pass; the **negative control** proves the cross-check bites
(disabling the product check fails exactly that sub-case and nothing else); the **full** `go test .` shows **one** failure
— the recorded pre-existing AMM test; native + `linux/amd64` + `js/wasm` all rc 0 with **`Public/main.wasm` proved
untouched** (11,375,951 B, same timestamp); all **seven gates pass**, with `verify:routes-parity` first **failing** on the
new unbaselined route and then passing once the exemption carried its real reason.

**Still the operator's to name:** the platform verifier URL and secret for whichever store is entered first. **Nothing in
this path invents a platform API** — with no credential it refuses, by design.

