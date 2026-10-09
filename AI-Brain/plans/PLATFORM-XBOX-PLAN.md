# XBOX / MICROSOFT STORE — PLATFORM PLAN (prepared 2026-10-08)

> **Owner:** this file owns everything Microsoft-platform (PC Store, Xbox console, Xbox services).
> **Companion:** `CONSOLE-DLC-STORE-PLAN.md` owns the *economy* (§7 solvency cap, §8 top-up, §10 platform research);
> this file owns the *platform* work. Neither re-declares the other's facts.
> **Provenance:** every public fact below was fetched, with its URL beside it. **No commission percentage is asserted** —
> Microsoft does not publish it. Gated or truncated content is named as such, never filled in.

## 1. THE DECISIVE FINDING FIRST — WHICH MICROSOFT PATH, IN WHAT ORDER

Two independent paths exist and they are **not the same commitment**:

| Path | Requirements (as published) | Source |
| --- | --- | --- |
| **PC (Windows) — self-service** | a **Win32** game + **GDK** tooling + an **MSIXVC** package, submitted in **Partner Center** — *"without requiring managed partner enrollment or concept approval"* · *"Do I need to be an ID@Xbox developer…? **No**"* · *"Xbox services are not required to publish a PC-only game"* | `learn.microsoft.com/en-us/windows/apps/publish/whats-new-game-publishing` |
| **Xbox console + Xbox services** | a **managed Xbox program**: *"If you plan to publish to Xbox consoles in the future, you'll need to enroll in the appropriate managed Xbox program at that time"*, plus the **NDA**, 18+, and a country Microsoft can trade with | same page + `developer.microsoft.com/en-us/games/` (FAQ) |

**Recommended order for this product: PC (GDK, self-service) FIRST, console second.** PC-only needs no concept approval and
no ID@Xbox enrollment; it uses the same GDK toolchain and packaging model that later carries into the console programme;
and it lets the console-facing work be scheduled against a **shipped** title rather than a greenfield one. A
recommendation, not a ruling.

## 2. PROGRAM, ELIGIBILITY, COST

* **PC path:** a Partner Center account; the hub advertises *"Revamped company onboarding experience with zero registration
  fees / Free developer registration for individual developers"*. Submission, certification and pricing all happen in
  Partner Center.
* **Console / Xbox services path:** the public FAQ states three hard requirements — *"you must be at least 18 years old"*,
  *"you need to sign an NDA"*, *"you must be in a country with which Microsoft can do business"* — plus concept approval
  for managed programmes. **The money terms (revenue share) are NOT public**; they live behind that NDA.

## 3. HOW AN IN-GAME DLC STORE ACTUALLY WORKS ON XBOX — Microsoft publishes our exact model

> *"This setup enables you to hide Durable and Consumable add-on products from the Microsoft Store. You expose these
> products for purchase inside your game by using the **XStore APIs** available in the Microsoft Game Development Kit
> (GDK)."* — `learn.microsoft.com/en-us/gaming/game-publishing/how-to/how-to-does-not-have-store-listing`

* **Applies to:** `Consumable`, `Durable` / `Durable with Packages`. **Does NOT apply to:** `Game`, `Game Demo`,
  `Bundle`, `Add-On Bundle`, `Season Pass`.
* Partner Center configuration: the setup module → *"**Doesn't have its own Microsoft Store listing**"*; **Pricing and
  availability** → *"**Can be purchased (from within the parent product only)**"*; a **Release / Stop acquisition**
  schedule; and **the price is configured in Partner Center**.

**CONSEQUENCE FOR OUR ECONOMY (binding).** On Xbox the money is collected by **Microsoft's commerce**, so the console's
in-game store is a *platform-commerce front end*. Our `ConsoleAssetReceipt` — *"a secure purchase or lease confirmation
from an external platform"* — was precisely the missing piece, and **its writer landed on 2026-10-08**
(`console_entitlement.go`): **the platform sells, our server grants the entitlement from the platform's own record of the
purchase**, idempotently and without moving a single unit of balance.
Our own **USDC top-up rail therefore cannot be the console payment path**, which is the same conclusion the economy
plan's §10 reached by a different route — and it turns the segregation we already have (`console_server.go`: **0** signing
calls, **0** application-call constructions, **0 of 5** chain rails, loopback only) from a preference into a
**requirement**.

## 4. THE XBOX REQUIREMENTS (XRs) THAT BIND THIS PRODUCT

XRs are the public certification spine: *"the policies, technical requirements, and product component-related requirements
to which all developers and publishers of XBOX console games must conform"*, applying to Xbox One, Series X|S, **PC and
handheld**, and cloud — `gaming/gdk/docs/store/policies/console/console-certification-requirements-and-tests`, document
**version 16.4 / 2026-08-11**. The ones that matter here, each retrieved from that page or its public change history:

| XR | What it demands of us |
| --- | --- |
| **XR-042 No Real-Money Cash-Out** | **the named prohibition our crypto rail must respect.** Retired from the XR list into the *"XBOX Game Store policy document in the Publisher Guide"* (the change log names XR-042 verbatim) — **read that document in full before building any payout path** |
| **XR-041 Microsoft Store Token Usage** | Store tokens for Store transactions, not proprietary ones (same retirement) |
| **XR-013 Linking Microsoft Accounts with Publisher Accounts** | our wallet/identity **is** a publisher account: link **and unlink** (*"Unlinking options must not be hidden deep in the UI or require technical support as the only available method"*), **single sign-on after linking**, *"Titles must not store non-XBOX account credentials or unnecessary personal information locally on the console"*, age-appropriate access for rating-eligible users, and a documented **exception request** process |
| **XR-130 Series X \| S Generation** | *"feature and save parity, no gameplay segmentation, and identical game modes"* within a generation — a hard constraint on a heavy 3D/web client |
| **XR-074 Loss of Connectivity** | partner services are **actively blocked in certification** (*XMAT* is used to *"identify and block partner service hosts for dynamic connectivity loss and pre-launch downtime testing"*) — **our own server's outage is a tested case** |
| **XR-124 Game Invitations** | Xbox Live invites must be used for Live→Live invites |
| **XR-073 Blocking and Muting** | blocking/muting must cover our custom in-game services |
| **XR-055 / XR-062** | Achievements, Challenges and Gamerscore; achievement naming profanity rules |
| **XR-012 / XR-015 / XR-132** | secure data transfer, privacy & permissions, service-call limitations |

**Two sections could NOT be read** (reproducible truncation, recorded in the economy plan's §10): Store Policies
**§10.8 Financial Transactions** and **§10.13 Gaming and XBOX**. **One snippet of §10.13 WAS retrieved** and is recorded as
a *snippet*, not as text: *"Publishers are permitted to use user specific data from XBOX services, Microsoft Store, and
other platforms on the XBOX network, subject to the following limitations: Game Progress, InGame Items, and Statistics —
Titles can, at their discretion, sync game progress, **virtual currency wallets**, and in-game items"*. It suggests the
policy contemplates a virtual wallet synced across platforms — **read both sections in full before building**
(§10.8 is the section most likely to constrain the rail).

## 5. RATINGS — REQUIRED FOR EVERY SUBMISSION, AND OUR THEME MAKES IT LOAD-BEARING

* *"Age ratings are required for all submissions to Certification."* They are generated by answering the **IARC**
  questionnaire on Partner Center's **Age ratings** page (*"Participation is free at the point of use"*); physical releases
  need long-form certificates instead.
* Ratings consider **content, functionality and user-generated content** — the guide names *"loot boxes, reward systems for
  user logins or other pressures to play"* as **functionality** that affects the rating. **This product is affected on all
  three axes:** mature themes, retention/reward systems, and player-generated art and marketplaces.
* **A hard regional consequence:** *"Any free product (including betas or demos) rated as CERO Z or IARC 18+ is not allowed
  to release on XBOX in Japan regardless of the distribution method."* **An 18+ rating is the likely outcome for this
  title, so Japan is likely excluded on Xbox** — a business fact to absorb before any plan assumes global reach.
* **AppxManifest ratings declarations are no longer permitted** for games; ratings live in Partner Center and the Store
  (an offline console caches Store ratings, refreshed every 10 days).

## 6. PAYOUT MECHANICS (what IS public)

From `learn.microsoft.com/en-us/partner-center/marketplace-offers/payout-faq`:

* *"Microsoft releases payments by the **fifteenth day of the maturity month**."*
* *"For orders paid by credit card, Microsoft holds payments for **30 days**, until the earning has matured."*
* *"An enrollment is considered complete **only after Microsoft validates your payout and tax profile**"* — a lapsed
  profile shows *"Action required - Update bank and/or tax profile"*.
* *"**Withholding tax is applicable for U.S. publishers who filed a W-9 form.** Withholding tax is calculated on a monthly
  payment."* A payout statement requires the **Owner** or **Financial Contributor** role.
* **The revenue-share percentage is NOT published.** The FAQ's *"store service fee is 3%"* belongs to the **commercial
  Marketplace** program (Azure/AppSource offers) and is **not** the consumer Store's game split — **the two must never be
  conflated.**

## 7. WORKSTREAMS MAPPED ONTO THIS REPOSITORY

| # | Workstream | What it touches | Why the platform requires it |
| --- | --- | --- | --- |
| **A** | **Native shell** — a Win32/GDK client hosting the game | a new entrypoint beside `Public/index.html` + `app.js`; the Go server is already a Win32-compilable binary | PC self-service requires a **Win32** package; console requires a GDK title |
| **B** | **In-game store on platform commerce** | Partner Center products (Consumable/Durable, *no Store listing*, *purchasable from within the parent only*) + XStore API calls in the client | §3 — the published mechanism, and it replaces the USDC rail on console |
| **C** | ✅ **`ConsoleAssetReceipt` writer — BUILT 2026-10-08** (`console_entitlement.go`) | the door, the **platform-verifier contract**, the idempotent grant (keyed by the platform purchase id), the read path and the record family all exist now; what remains is the platform's own credential | the platform sells; **our server grants the entitlement from the platform's OWN record of that purchase** — and a receipt a caller merely sends is never trusted |
| **D** | **Account link / SSO / unlink** | wallet-connect + `identity_bridge.go` (`linkedWallets`, `/api/identity/*`) | **XR-013**: link **and** unlink, SSO after linking, never store non-Xbox credentials on the console, age-appropriate access |
| **E** | **Achievement/season mapping** | `achievement_service.go`, career tiers | **XR-055 / XR-062**: Gamerscore achievements and their naming rules |
| **F** | **Offline / connectivity resilience** | the WebSocket client (`network.js` reconnect logic) | **XR-074**: certification **blocks our server hosts** to force dynamic connectivity loss |
| **G** | **Series S parity** | the 3D world (`world3d.js`) and any heavy client feature | **XR-130**: feature and save parity, no gameplay segmentation, identical modes |
| **H** | **Blocking / muting over our own services** | chat and social surfaces | **XR-073** |
| **I** | **Ratings package** | content descriptors, store listing, pitch material | required for every submission (§5) |

**ORDER:** **C** and **B** *are* the store; **A** is the prerequisite for any submission; **D–H** is certification work
that can be scheduled against the **PC path first** — the XRs apply to PC and handheld too, following the 2026-08-06
consolidation that merged the separate PC/mobile XR sets into the single set.

## 8. ACCEPTANCE GATES FOR THIS PLATFORM

1. **Nothing above starts before §10.8 and §10.13 are read in full.** The payment rail and the virtual-wallet rules are
   unread, and building first is how a rail gets designed that cannot ship.
2. Every workstream lands with the repository's own gates green (the seven `verify:*` gates), the full `go test .`, and
   native + `linux/amd64` + `js/wasm` rc 0 with `Public/main.wasm` **proved untouched** wherever the client is uninvolved.
3. **A platform-facing change is never proven by our own server alone:** a purchase path is proven by a **sandbox**
   transaction (Partner Center dev sandbox → RETAIL), and a connectivity change by a **blocked-host** run.
4. The economy plan's §7 cap is re-read against §3's conclusion: **a platform-sold add-on is an unmatured receivable**, and
   is therefore excluded from `usable` until Microsoft has actually paid.

## 9. OPEN ITEMS — WHAT THE OPERATOR MUST READ OR DECIDE

| # | Item | Owner | What it blocks |
| --- | --- | --- | --- |
| 1 | **Store Policies §10.8 and §10.13, in full** | operator (or a session with a route past the truncation) | whether the in-game store and any rail design is even permissible — **the top blocking read** |
| 2 | **The XBOX Game Store policy document** (the Publisher Guide's home of XR-041/XR-042) | operator | the cash-out and token rules **in their current wording** |
| 3 | **The revenue-share percentage** | operator, in the agreement behind the NDA | every input to §7's cap |
| 4 | **PC-first or console-first** | operator | the sequencing of workstreams A–I |
| 5 | **Partner Center + payout/tax profile setup** | operator | any payment at all (§6 — enrolment is incomplete until it validates) |
| 6 | **The ratings questionnaire** | operator + us | required for every submission; decides whether Japan is excluded |

## 10. HONEST LIMITS OF THIS PLAN

* **The Store's revenue share is unknown** and is therefore absent from every number here. **No estimate has been
  substituted for it** — an invented cut would silently corrupt the economy plan's §7 cap.
* **§10.8 / §10.13 are unread** (two fetch attempts, both truncating the page's middle); the one §10.13 extract is quoted
  **as a snippet**, not as text.
* The XR list cited is the **certification-tested subset**; the page itself notes a broader *"XBOX Requirements"* list
  exists that includes requirements **not** tested in certification.
* **Consoles are devkit-gated.** Nothing here should be read as "we can build for Xbox today": workstream **A** needs a
  managed programme, which needs open item **4** first.
* **"A browser client inside a Win32/GDK shell" is an assumption about workstream A, not a verified packaging path** — the
  GDK documentation is written for native engines and names Unity and Unreal explicitly.





