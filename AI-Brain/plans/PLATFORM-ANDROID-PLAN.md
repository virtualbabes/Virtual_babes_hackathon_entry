# GOOGLE PLAY / ANDROID — PLATFORM PLAN (prepared 2026-10-08)

> **Owner:** this file owns everything Google-Play / Android.
> **Companions:** `CONSOLE-DLC-STORE-PLAN.md` owns the *economy* (§7 solvency cap, §8 top-up) and the platform index (§10.9);
> `PLATFORM-APPLE-PLAN.md` owns iOS and is deliberately droppable.
> **Provenance:** every fact below was fetched from a **Google-hosted** URL and quoted verbatim, with its source named
> beside it, on **2026-10-08**. Where a policy could not be retrieved it is named **UNREAD, NOT ABSENT** — never filled in.

## 1. THE DECISIVE FINDING FIRST — Android is the most economically legible platform this product has

Three things are public on Android that are public on **no console**, and all three change what can be *scheduled*:

1. **The service fee is published in full, with rollout dates** (Google's *Service fees* article — §5).
2. **An external-web-link route exists and is priced**: *"**20% for external web links**"* (*"15% for external web links"*
   when participating in the Play Games Level Up / Apps Experience programs).
3. **The fee is being split by install age** — *"new"* vs *"existing"* installs — from **30 June 2026** (EEA, UK, US) and
   **30 September 2026** (Australia, Japan).

So on Android the economics can be **modelled from published numbers**, which is the exact opposite of the console position
recorded in §10.6 of the console plan (*unknown is not zero*).

The second finding is harder, and it is a **design constraint**: Google **requires** its own billing for *"virtual
currencies"*, and **forbids** it for *"peer-to-peer payments"* and *"content that facilitates online gambling"*. Our
player-to-player market, auctions and wagered matches sit precisely in that gap. §3 and §4 state the structural answer.

## 2. PROGRAM, ELIGIBILITY, COST (all public)

Source: `support.google.com/googleplay/android-developer/answer/6112435` (*Get started with Play Console*).

* **US$25 one-time registration fee**, verbatim: *"There is a US$25 one-time registration fee."* Accepted cards:
  MasterCard, Visa, American Express, Discover **(the U.S. only)**, Visa Electron (outside the U.S. only) —
  *"Prepaid cards are not accepted."*
* **18+:** *"You must be at least 18 years of age to sign up for a Play Console account."*
* **Two account types:** Personal and Organization. The choice has consequences:
  * *"Developers with personal accounts created after November 13, 2023, must meet specific testing requirements before
    they can make their app available on Google Play"* — i.e. a **closed-testing phase is mandatory** on that path;
  * *"Starting in early 2024, developers with new personal accounts will be required to verify that they have access to an
    Android device using the Play Console mobile app before they can make their app available."*
* **Identity verification:** *"you may be asked for a valid government ID and a credit card, both under your legal name. If
  this information is determined to be invalid, your registration fee will not be refunded."*
* **THE OPERATOR'S OWN ACCOUNT (given directly, 2026-10-08):** he **already holds a Google Cloud and a developer
  account**, so the fee, the 18+ gate and identity verification are **behind us rather than ahead**. **One question
  remains, and it is not paperwork:** whether that Play Console developer account is **Personal or Organization** — a
  Personal account created after 13 Nov 2023 carries the mandatory **closed-testing phase** and the **Android-device
  verification** before production, and an Organization account does not. That one fact decides whether a testing cycle
  must be scheduled into the launch.

**The Developer Distribution Agreement** (effective **15 September 2025**) fixes the commercial shape:
Google acts *"solely at Your direction"* (§2.1); money flows through a **Payments Profile** held by a **Payment
Processor** — *"a Google-affiliated entity providing services that enable Developers to receive payments for Products sold
via Google Play"* — and **Taxes** are defined to include *"Transaction Taxes and Withholding Taxes"*, with each party
responsible for its own net income taxes. The DDA's **payment-schedule clause** (the section stating when funds are
remitted, and against what minimum) fell inside the un-displayed middle of the fetched document and is therefore recorded
in §6 as an **unread DDA/in-account read**, not as absent.

**Distribution outside Play is explicitly blessed**, which no console allows: *"Can I distribute my app on other Android
app stores or my website? **Yes**, you can distribute your app however you like… all without using Google Play's billing
system."* **Android is therefore the one platform where the store is optional** — a direct APK and the web/PWA build are
policy-permitted routes, not workarounds.

## 3. HOW DIGITAL GOODS MUST BE SOLD (Payments policy, verbatim)

Source: `support.google.com/googleplay/android-developer/answer/10281818` — *Understanding Google Play's Payments policy*.

* *"Google Play's billing system **is required** for developers offering in-app purchases of digital goods and services
  distributed on Google Play."*
* **Requiring GPB:** *"Digital items (such as **virtual currencies**, extra lives, additional playtime, add-on items,
  characters, or avatars)"*, subscription services, *"App functionality or content (such as an ad-free version of an app or
  new features not available in the free version)"*, *"Cloud software and services"*.
* **Not supported:** *"Purchases or rentals of physical goods"*, *"Purchases of physical services"*, *"Payment of a credit
  card or utility bill"*.
* **The prohibition that binds this product, verbatim:** *"Google Play's billing system **must not be used for
  peer-to-peer payments, content that facilitates online gambling**, or any product category deemed unacceptable under
  Google's Payments Center Content Policies."*
* **Tokenized assets:** the article's closing paragraph reads *"…Google Play's Payments policy applies and you must use
  Google Play's billing system for such transactions. Refer to our **Blockchain-based content policy** for more details on
  tokenized digital assets."* (That policy's text is **UNREAD** — see §6.)

### The store-legal shape this forces (a design constraint, not a tax)

| Our system | On Android it may be | It may NOT be |
| --- | --- | --- |
| $VBV purchase / top-up (a *virtual currency*) | sold through **GPB** | sold through our own rail **inside** the app |
| player-to-player market, auctions | a spend of a **balance the player already owns** | a *"peer-to-peer payment"* |
| wagered matches, spectator wagers | an in-game wager of virtual currency with **no real-world value** | *"content that facilitates online gambling"* |

**The dividing line is real-world convertibility.** The moment the mobile build can cash out, those same flows become both
(a) barred from GPB and (b) a **real-money gambling/gaming app** under §4 — a dead end, not a fee. **The console plans'
segregation is therefore the mobile design too, and it is now backed by two independent policies rather than one.**

## 4. REAL-MONEY GAMBLING, GAMES, AND CONTESTS (verbatim)

Source: `support.google.com/googleplay/android-developer/answer/9877032`.

* **Summary, verbatim:** *"While Google Play generally prohibits apps from facilitating real-money gambling and gaming, an
  exception is made for those that are licensed and approved by Google. **All approved apps must be free to download, must
  not use In-app Billing, and must prevent access for minors and unauthorized locations.**"*
* **Permitted product types** (each requiring a licence): *"Online Casino games · Sports Betting · Horse Racing (where
  regulated and licensed separately from Sports Betting) · Lotteries · Daily Fantasy Sports"*.
* **The eight requirements, verbatim:** complete Google's application process · comply with local law and industry
  standards per country · *"Developer must have a valid gambling license for each country or state/territory in which the
  app is distributed"* · *"must not offer a type of gambling product that exceeds the scope of its gambling license"* ·
  *"App must prevent under-age users from using the app"* · *"App must prevent access and use from countries, states/
  territories, or the geographic areas not covered by the developer-provided gambling license"* · *"App must NOT be
  purchasable as a paid app on Google Play, nor use Google Play In-app Billing"* · *"App must be free to download and
  install from the Google Play store"* · *"App must be rated AO (Adult Only) or IARC equivalent"* · *"App and its app
  listing must clearly display information about responsible gambling."*
* **Don'ts, verbatim (excerpt):** *"Don't let minors access the app or incorrectly age-rate your app"*; *"Don't charge users
  to download a real-money gambling or gaming app from Google Play"*; *"Don't use Google Play's In-app Billing for payments
  within your app."*
* The article also carries **"Other Real-Money Games, Contests, and Tournament Apps"** — a second acceptance process with
  US state/territory licensing rules for daily fantasy sports (part of that section fell outside the fetched window, and is
  recorded as such).
* Google additionally publishes **"Country/region allowances for gambling apps"** — the per-territory list that must be
  read before any wagering surface is designed.

**Two consequences, and neither is optional.**

1. **A classification as real-money gambling/gaming makes the app FREE WITH NO IAP** — which is *incompatible* with §3's
   requirement to sell the virtual currency through GPB. The two policies cannot both be satisfied, so **the
   classification must be avoided by structure, never argued**.
2. **"AO (Adult Only) or IARC equivalent"** is a rating mainstream storefronts will not carry in practice, so a
   classification error is not a fine — it is **removal**.

**Nothing here changes the game.** It is a statement that the **mobile build must contain no real-value exit**, which is the
same conclusion the console plans reached from Microsoft's published prohibition — now reached independently, twice.

## 5. SERVICE FEES (verbatim, with the rollout dates)

Source: `support.google.com/googleplay/android-developer/answer/112622` — *Service fees*.

* Framing, verbatim: *"97% of developers distribute their apps and take advantage of all Google Play has to offer at no
  charge. Of those developers that are subject to a service fee, 99% are eligible for a fee of 15% or less by
  participating in different programs offered by Google Play."*
* Scope: *"Apps and in-app products sold through Google Play's billing system or an Alternative Billing System (as defined
  below) in accordance with the Payments policy are subject to a service fee."*
* **The 2026 restructure.** For *"transactions with users in Australia, the European Economic Area, Japan, United Kingdom,
  or United States"*, the fee *"will be determined by whether the transacting user's install is 'new' or 'existing', with
  the regional rollout date (**June 30, 2026 for the EEA, UK, and US; September 30, 2026 for Australia and Japan**) serving
  as the effective date"* — *"New Installs"* meaning a first install/update **on or after** that date, *"Existing Installs"*
  **before** it.
* **The cells, exactly as the article renders them.** Its table **flattened** in the retrieved text, so the column alignment
  is **ambiguous and is NOT reconstructed here**:
  * *"First $1M (USD) of annual earnings"* · *"10% + 5% billing fee *"* · *"Standard 10% + 5% billing fee *"* ·
    *"20% + 5% billing fee *"* · *"25% + 5% billing fee * OR; 20% for external web links"*
  * *"When participating in the Play Games Level Up or Apps Experience programs"* · *"10% + 5% billing fee *"* ·
    *"15% + 5% billing fee *"* · *"20% + 5% billing fee * OR; 15% for external web links"*
  * *"\* Billing fee applies when a user completes a purchase of an in-app digital service or feature using Google Play
    Billing or for the initial purchase of a paid app or game"*
  * The source table's column order is: **Auto-renewing subscriptions (new & existing installs) · Other transactions (new
    installs) · Other transactions (existing installs)**.
* **Every other market, until the restructure rolls out globally:** *"15% for the first $1M (USD) revenue earned by the
  developer each year"*, *"30% for earnings in excess of $1M (USD)"*; **subscriptions 15%** *"regardless of revenue earned
  by the developer each year"*.
* **Alternative billing systems** (South Korea, India): the fee on those transactions *"is equal to the service fee
  applicable for transactions via Google Play's billing system **reduced by 4%**"*.
* **EEA external offers:** *"developers may direct users in the EEA outside of their app to promote offers of digital
  features and services, in accordance with program requirements"*, and *"can now offer their users in the EEA an
  alternative to Google Play's billing system within their apps, subject to program requirements."*

**The honest limit, stated rather than papered over:** the article itself warns *"there isn't a single service fee"*. The
exact cell that would apply to a **$VBV top-up** in the US after 30 June 2026 cannot be selected from the flattened text
without guessing, so it is **not selected here** — the two candidate cells (*"20% + 5% … OR; 20% for external web links"*
for new installs, *"25% + 5% … OR; 20% for external web links"* for existing installs) are recorded verbatim and the choice
is an **in-account read** (§6). **This is the one platform where the operator can read the real number before committing.**

## 6. THE UNREAD POLICIES AND IN-ACCOUNT READS (named, never filled in)

Google publishes a **policy index** at `play.google/developer-content-policy/` whose **exact titles** are known (retrieved
verbatim). The *bodies* were then attempted and **failed**: two URLs guessed from the index returned the Play storefront,
two 404'd, and the Policy Center publishes no discoverable per-policy URL pattern. So:

| # | Policy / read (title as Google's index gives it) | Status and why it matters |
| --- | --- | --- |
| 1 | **Blockchain-based Content** | **UNREAD** — this is the policy the Payments article itself points at for *"tokenized digital assets"*; it decides whether our tokens/NFTs may be sold on the store at all |
| 2 | **Functionality and User Experience** (+ **Spam**) | **UNREAD** — the wrapper/WebView "minimum functionality" question lives here, and it decides §7's route (b) |
| 3 | **Content Ratings** (+ **Age-Restricted Content and Functionality**) | **UNREAD** — the IARC path for Play, and therefore our age framing |
| 4 | **Target API Level** · **Use of SDKs** · **In Apps SDK Requirements** | **UNREAD** — annual obligations, recurring calendar cost |
| 5 | **Understanding Google Play's Cryptocurrency Exchanges and Software Wallets Policy** | **UNREAD** |
| 6 | **Country/region allowances for gambling apps** | **UNREAD** — required reading before **any** wagering surface is designed |
| 7 | **Financial Services** · **User Generated Content** · **AI-Generated Content** · **Requirements for apps with incidental dating or matchmaking features** · **Understanding moderation requirements and incidental sexual content in UGC apps** | **UNREAD** — four of these plausibly touch this product directly (UGC, AI citizens, adult themes) |
| 8 | the **DDA payment-schedule clause** (when funds remit, against what minimum, currency conversion) | **UNREAD** — it fell inside the document's un-displayed middle |
| 9 | **Play Console → Payments profile** (fee choices, tax documents, payout minimums) | **in-account** |
| 10 | the **policy-update stream**: the index carries *"April 15, 2026: We're updating our policies"* and a *PolicyBytes April 2026* item, and lists **"Changes to Google Play for upcoming app store bills for users in applicable US states"** and **"Changes to the external offers program for users in the EEA"** | **UNREAD** — these two decide *where* §5's external-link route is actually available |

## 7. WHAT OUR ARCHITECTURE ALREADY GETS RIGHT, AND THE ONE MOBILE QUESTION

1. **The server is untouched.** Android's rules govern *commerce and content*, not Go: the economy, the record set, the
   lock gates and the authority all port as-is. **No workstream in this plan changes the ledger.**
2. **The segregation is already built, and it is now double-justified.** `console_server.go` has **0** signing calls and
   **0 of 5** chain rails; the same virtual-ledger discipline is what keeps a mobile build on the right side of §3 and §4.
3. **The one engineering question is the client shell** — and it is the *same* question the Switch plan names: our client is
   **Go/WASM in a browser**. Android can host that (a WebView / Trusted Web Activity shell over the existing `index.html` +
   `main.wasm`), **but whether such a shell satisfies Google's Functionality and User Experience policy is UNREAD** (§6
   row 2). Two honest routes, in order:

   * **(a) ship the web/PWA build as the Android deliverable** — no store, no fee, no rating, no policy review, blessed by
     the Payments article's own answer (*"you can distribute your app however you like"*), and it reuses everything this
     repository already runs;
   * **(b) ship a store build** — only **after** §6 row 2 is read, because a submission judged a repackaged website is a
     rejection, and entering the store also brings §3 and §4 fully into force.

## 8. WORKSTREAMS — SHARED vs ANDROID-SPECIFIC

| # | Workstream | Android-specific? |
| --- | --- | --- |
| **A** | **Client delivery** — TWA/WebView shell over the WASM build, or the PWA itself | **YES** — and **optional** if route (a) is chosen |
| **B** | **Play billing integration** ($VBV top-up / DLC through GPB) | **YES** |
| **C** | **`ConsoleAssetReceipt` writer** — entitlement granted from a verified platform purchase | ✅ **BUILT 2026-10-08** (`console_entitlement.go`): **one** implementation for every platform, so Play is a **consumer** of it and never a second owner |
| **D** | **Account link / sign-in** | shared shape, Play-specific API |
| **E** | **Content-rating submission** (IARC questionnaire, filed per store) | shared questionnaire |
| **F** | **Target API level · Data safety · SDK declarations** | **YES** — recurring Play-only obligations |
| **G** | **The store catalogue + prices** | shared with the economy plan's §8 — **one owner, not four** |

**The rule this table enforces:** *one* entitlement path (**C**) and *one* catalogue (**G**) serve every platform. Only
**A**, **B**, **D** and **F** are genuinely Play-specific, and duplicating **C** or **G** per platform would create several
owners of one fact — the failure this repository's gates exist to prevent.

## 9. ACCEPTANCE GATES

1. **Registration is the first real milestone** (US$25, 18+), and if a **personal** account is used then the mandated
   closed-testing phase and the device-verification requirement are **scheduled in advance**, not discovered mid-launch.
2. **No mobile commerce is designed before §6 rows 1–3 are read.** Row 1 decides whether tokenised assets may be sold on the
   store at all; row 3 decides the age framing the whole mobile build inherits.
3. **No wagering surface is designed before §6 row 6** (Google's per-territory gambling allowances).
4. Any code that lands meets the repository's own gates: the seven `verify:*` gates, the full `go test .`, native +
   `linux/amd64` + `js/wasm` rc 0, and `main.wasm` proved untouched wherever the client is uninvolved.
5. **A shell build is proven by running on a device**, not by compiling — the same rule the Switch plan states.

## 10. HONEST LIMITS

* **No estimate appears in this plan.** Every percentage above is quoted from Google's own article, with its date; the
  one cell that matters financially is recorded verbatim and deliberately **not selected** (§5).
* **The fee table's column alignment could not be recovered** from the retrieved text, so the applicable cell for a top-up
  is an in-account read, not a guess.
* **Ten items in §6 are unread**, and three of them (blockchain-based content, functionality/user experience, gambling
  allowances) are load-bearing for design.
* **Google's own search page rendered nothing** and the Policy Center's per-policy pages follow no discoverable URL pattern;
  the policy *titles* are exact, the *bodies* are not yet read.
* **Play's policies change frequently** (its index advertises an April 2026 update), so every fact here carries its
  fetch date and must be re-read before any submission.
* **No code was written or changed by this research.** The file is a record; it fixes no software.
