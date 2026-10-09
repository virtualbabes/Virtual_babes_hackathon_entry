# APPLE / APP STORE — PLATFORM PLAN (prepared 2026-10-08)

> ## ⚠️ PRIORITY: LOW — NON-ESSENTIAL — DROPPABLE
> **This is deliberately the lowest-priority platform in the set.** It exists so a future session can execute iOS *if and
> only if* it chooses to, and so that **nothing else in the repository becomes dependent on it**. **§0 is the drop test and
> is the most important section in this file.**

> **Owner:** this file owns everything Apple-platform (iPhone, iPad, the App Store).
> **Companions:** `CONSOLE-DLC-STORE-PLAN.md` (the economy, and the platform index §10.9); `PLATFORM-ANDROID-PLAN.md`
> (**the primary mobile target**); the three console plans.
> **Provenance:** every fact below was fetched from an **Apple-hosted** page and quoted verbatim on **2026-10-08**. Where a
> guideline's text could not be retrieved it is named **UNREAD, NOT ABSENT** (§5) and **no behaviour is inferred from it**.

## 0. THE DROP TEST — what happens if we never ship on iOS

Answer: **nothing.** That is measured against the other plans, not asserted:

| If iOS is dropped… | Effect |
| --- | --- |
| **Android (the primary mobile target)** | **No change.** Its store work is justified entirely by its own policies; nothing in it cites Apple. |
| **The entitlement path** (`ConsoleAssetReceipt` writer — workstream **C** in every platform plan, and **BUILT 2026-10-08**) | **No change** — justified by Android **and** the three console platforms, and it was built **without** Apple. Apple would be a **fourth consumer** of it, never a justification for it. |
| **The store catalogue + prices** (one owner) | **No change.** |
| **The $99/year membership, the rating filing, the review cycle** | **Saved**, and no engineering time is spent on a store we may never enter. |
| **iPhone users** | **Reached anyway** — through the web/PWA build, which Apple's own guidelines name: *"For everything else there is always the open Internet. If the App Store model and guidelines … are not best for your app or business idea that's okay, we provide Safari for a great web experience too."* |

**Two rules follow, and they are why this section exists:**

1. **No design decision may be made because of Apple.** If an economy rule, ledger behaviour, feature or UI element exists
   *solely* to satisfy App Review, that is a defect in the plan, not a requirement. The game's design is set by the
   constitution and the game.
2. **Nothing may be built iOS-first or iOS-only.** Every shared component must be justified by at least one *other*
   platform — which is how §6's table is already written.

## 1. PROGRAM, ELIGIBILITY, COST (public)

* **Apple Developer Program: *"$99 annual membership"*** — verbatim from `developer.apple.com/programs/`. It includes
  TestFlight (*"invite up to 10,000 external users"*), distribution in **175 regions**, and *"Apple handles worldwide
  payment processing"*.
* A **free** Apple developer account exists — *"A free Apple developer account gives you access to tools like beta versions
  of Xcode and operating systems"* — but distribution needs the membership.
* **Apple Developer Enterprise Program (*"$299/year"*)** exists for private internal distribution and is **not applicable**.
* Enrolment is *"as an individual or organization"*, and on this platform that choice is **load-bearing**, because of
  guideline **3.1.5(i)** (§3).

## 2. THE MONEY — WHAT APPLE ACTUALLY PUBLISHES (all verbatim)

Sources: `developer.apple.com/app-store/review/guidelines/` and `developer.apple.com/app-store/small-business-program/`.

* **The in-app-purchase rule is stated inside the guidelines' own cross-references, verbatim:** *"Consumer, single user, or
  family sales must use in-app purchase."* (guideline **3.1.3(c)**), following **3.1.3 Other Purchase Methods: "The
  following apps may use purchase methods other than in-app purchase."**
* **The carve-out with the most relevance to this product, verbatim** — *"**3.1.3(f) Free Stand-alone Apps:** Free apps
  acting as a stand-alone companion to a paid web based tool (i.e. VoIP, Cloud Storage, Email Services, Web Hosting) do not
  need to use in-app purchase, provided there is no purchasing inside the app, or calls to action for purchase outside of
  the app."*
  **Read carefully:** an iOS *thin client over the PC/web product*, with **no purchasing inside it and no call to action**,
  is the one published shape that does not require IAP. The parenthetical list names VoIP/storage/email/hosting, so relying
  on it for a game is an **interpretation, not a published entitlement** — and it must never be the reason a design choice
  is made (§0 rule 1).
* **3.1.3(e), verbatim:** *"If your app enables people to purchase physical goods or services that will be consumed outside
  of the app, you must use purchase methods other than in-app purchase to collect those payments, such as Apple Pay or
  traditional credit card entry."*
* **3.1.4 Hardware-Specific Content, verbatim (excerpt):** functionality may be unlocked without IAP where it depends on
  specific hardware, *"provided that an in-app purchase option is available as well. You may not, however, require users to
  purchase unrelated products or engage in advertising or marketing activities to unlock app functionality."*
* **Subscriptions, verbatim:** *"Subscriptions may include consumable credits, gems, in-game currencies, etc."*;
  *"those offering subscriptions should allow a user to get what they've paid for without performing additional tasks"*;
  *"Before asking a customer to subscribe, you should clearly describe what the user will get for the price."*; and
  *"Apps that attempt to scam users will be removed from the App Store… you may be removed from the Apple Developer
  Program."*
* **Commission, verbatim:** the Small Business Program *"features a reduced commission rate of **15%** on paid apps and
  Apple In-App Purchases"* for *"Existing developers who made up to **1 million USD** in proceeds in the prior calendar
  year… as well as developers new to the App Store"*; above that threshold *"the standard commission rate will apply to
  future sales"*, and if proceeds later fall below it *"they can re-qualify for the 15% commission the year after"*.
  On the EU's alternative terms: *"Apple will offer a further reduced commission of **10%**."*
* **Honest gap:** that page names *"the standard commission rate"* but **does not print the number**, so **no standard-rate
  figure is asserted in this file** — this plan records only what a page actually says.

## 3. CRYPTO — GUIDELINE 3.1.5, VERBATIM (the clause that decides most of this platform)

> **3.1.5 Cryptocurrencies:** **(i) Wallets:** *"Apps may facilitate virtual currency storage, provided they are offered by
> developers enrolled as an **organization**."* **(ii) Mining:** *"Apps may not mine for cryptocurrencies unless the
> processing is performed off device (e.g. cloud-based mining)."* **(iii) Exchanges:** *"Apps may facilitate transactions or
> transmissions of cryptocurrency on an approved exchange, provided they are offered only in countries or regions where the
> app has appropriate licensing and permissions to provide a cryptocurrency exchange."* **(iv) Initial Coin Offerings:**
> *"Apps facilitating Initial Coin Offerings ("ICOs"), cryptocurrency futures trading, and other crypto-securities or
> quasi-securities trading must come from established banks, securities firms, futures commission merchants ("FCM"), or
> other approved financial institutions and must comply with al…"* *(the sentence is cut at exactly that point in the
> retrieved text — recorded as a cut, not completed by inference)*.

**What this clause alone decides:**

1. **A wallet-bearing build cannot ship from an individual developer account** — clause (i) requires *organization*
   enrolment. On this platform the enrolment type is therefore a product decision, not paperwork.
2. **Clause (iii) is a per-region licensing obligation** for anything exchange-shaped, and **clause (iv) is a
   disqualification**: we are not a bank, a securities firm, an FCM, or an approved financial institution, so **no
   token-sale or quasi-security surface may exist in an Apple build**.
3. **The safe shape is the one the console plans already chose:** a thin client whose value moves in the **virtual liability
   ledger**, with the chain-facing half living on PC/web. That is now the third independent policy to point at the same
   design.

## 4. RATINGS — THE TWO REGIONAL CLAUSES THAT BITE THIS PRODUCT (verbatim)

Source: `developer.apple.com/help/app-store-connect/reference/age-ratings/`. Apple issues a **global** rating plus
**regional overrides**, and two of those overrides are triggered by content this game already has:

* **Republic of Korea (GRAC), verbatim:** *"apps in the **Games or Entertainment** categories (primary or secondary) and/or
  apps with **Frequent/Intense instances of Simulated Gambling** will display an additional regional rating along with
  their Apple global age rating"*, and *"The GRAC may issue a **KR-15** regional rating with an updated pictogram, or text
  that indicates **KR-19** regional age rating for some apps."* Apple then messages the developer to override and re-submit.
* **France, verbatim:** *"apps with a **17+** Apple global age rating will display an additional regional rating of **18+**
  on the App Store in France."*
* **Brazil:** the page distinguishes **MJSP official** ratings/descriptors from **self-rated** ones, and updates regional
  ratings on the developer's behalf where the MJSP has issued one.

**Why this matters more than it looks:** *"Frequent/Intense instances of Simulated Gambling"* is a descriptor our wagered
matches, spectator wagers and tournaments plausibly attract, and *"Games"* is a category we would be in — so **Korea's
regional regime applies by default**, and France escalates **any** 17+ rating by one band. Both are rating consequences
rather than prohibitions, but neither is discretionary, and neither is avoidable by intent.

**Honest gap:** the page's **global value table and questionnaire** fell inside the un-displayed middle of the fetched page.
The values evidenced in the retrieved text are **4+, 9+, 12+ and 17+** (the Korea mapping table pairs *"4+ / 9+ / 12+*"*,
and the France clause names *"17+"*); the questionnaire's content is **not** read here.

## 5. THE UNREAD GUIDELINES (the gates on this platform — named, never filled in)

Apple's guidelines are a single large page; its **middle did not survive retrieval** on either the direct fetch or the
text-mirror fetch, so the sections below are **UNREAD, NOT ABSENT**, and **nothing in this plan is inferred from them**:

| # | Read | Status |
| --- | --- | --- |
| 1 | **Guideline 3.1.1 (In-App Purchase)** — including its NFT / cryptocurrency clause | **UNREAD** — the single most-cited rule for NFT-capable apps |
| 2 | **Guideline 3.1.3(b) Multiplatform Services** | **UNREAD** — only the 3.1.3 intro sentence (*"The following apps may use purchase methods other than in-app purchase."*) was retrieved |
| 3 | **Guideline 4.2 Minimum Functionality** | **UNREAD** — this is the **wrapper/repackage risk** for a shell over our web build |
| 4 | **Guideline 5.3 Gaming, Gambling, and Lotteries** | **UNREAD** — the free-app / geo-restriction / licence questions for any wagering surface |
| 5 | the **payment schedule and thresholds** | **UNREAD** — `developer.apple.com/support/storekit-external-purchase-link/` returned **HTTP 404** (recorded rather than worked around), and no payments-timing page was located |
| 6 | **App Store Connect → Agreements, Tax, and Banking / Payments and Financial Reports** | **in-account** |
| 7 | **Notarization / alternative marketplaces / Web Distribution** (named in the guidelines' introduction — *"In some markets and on certain platforms, developers can also distribute notarized apps from alternative app marketplaces and directly from their website"*) | **UNREAD** — noted only because it is an existing published route, not because this plan relies on it |

**Gates 1, 3 and 4 are the ones that must be read before a single line of iOS work begins** — and, per §0, **none of them
is allowed to change a design decision** elsewhere in the product.

## 6. IF (AND ONLY IF) WE EVER DO IT — WORKSTREAMS, AND WHICH ONE JUSTIFIES THEM

| # | Workstream | Apple-specific? | Justified by *another* platform? |
| --- | --- | --- | --- |
| **A** | **Client delivery** — a thin client over the web build, or a Swift/SwiftUI re-host | **YES** | **partly** — the thin-shell question is shared with Android's route (b); a Swift re-host would be Apple-only and needs its own reason |
| **B** | **StoreKit / IAP integration** ($VBV top-up) | **YES** | **no** — and only if the build sells anything at all (§2's carve-out) |
| **C** | ✅ **`ConsoleAssetReceipt` writer — BUILT 2026-10-08** | **shared** | **YES** — Android **and** all three console plans; iOS would be a fourth consumer of one implementation |
| **D** | **Account link / sign-in** | shared shape, Apple-specific API | yes |
| **E** | **Age-rating questionnaire** | shared | yes — the same questionnaire is filed with Google's IARC path |
| **F** | **Enrolment as an *organization*** if the build carries a wallet (3.1.5(i)) | **YES** | n/a — a precondition, not engineering |
| **G** | **Store catalogue + prices** | shared (economy plan §8) | **YES** |

**The rule this table enforces** (and the reason §0 exists): **no row may be started *because* Apple exists.** **C**, **D**,
**E** and **G** are already justified without it; only **A**, **B** and **F** would come into being solely for iOS.

## 7. ACCEPTANCE GATES

1. **§5 gates 1, 3 and 4 are read first**, in that order, and their answers are recorded in this file **as facts with their
   source** — before any iOS code is written.
2. **An explicit operator decision to enter the App Store**, recorded in the session handoff — because §0 makes this plan
   **opt-in**, and dropping it is a legitimate, expected outcome.
3. **The enrolment type is decided before the build shape**, since a wallet-bearing build requires *organization*
   enrolment (3.1.5(i)).
4. If a build sells nothing and links nowhere, the **3.1.3(f)** carve-out is named **as the interpretation it relies on**,
   in plain words — never left implicit.
5. Any code that lands meets the repository's own gates: the seven `verify:*` gates, the full `go test .`, native +
   `linux/amd64` + `js/wasm` rc 0 — iOS work is no exception for the parts that live in this repository.

## 8. HONEST LIMITS

* **Five guideline sections in §5 are unread**, and **three of them are gates** (3.1.1, 4.2, 5.3).
* **No standard commission figure is asserted**, because the page that names *"the standard commission rate"* does not print
  it. Only the **15%** and **10%** figures are quoted, and they are quoted from a page that states them.
* **One URL 404'd** (`developer.apple.com/support/storekit-external-purchase-link/`) — recorded as a failed read, **not**
  substituted with a guess about the US external-purchase-link route.
* **Search engines contributed nothing:** DuckDuckGo answered with a bot challenge and Bing returned unrelated consumer
  results even with a `site:developer.apple.com` filter — so every Apple fact here comes from a **direct Apple fetch**.
* **Truncation is this platform's chief research obstacle:** the guidelines are one very large page whose middle is lost on
  retrieval, which is exactly why so much of this file says **UNREAD** rather than filling gaps in.
* **No code was written or changed by this research.** This file is a record; it fixes no software, and by §0 it should
  never be the reason any software changes.
