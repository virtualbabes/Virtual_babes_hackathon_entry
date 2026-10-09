# PLAYSTATION (SONY) — PLATFORM PLAN (prepared 2026-10-08)

> **Owner:** this file owns everything PlayStation-platform.
> **Companion:** `CONSOLE-DLC-STORE-PLAN.md` owns the economy model (§7 cap, §8 top-up, §10 research).
> **⚠️ THIS IS THE LEAST INFORMED OF THE THREE PLANS, AND THAT IS A MEASURED FACT, NOT AN OMISSION.** Sony publishes
> nothing to a non-partner. Everything below either (a) is a measured statement about the gate, or (b) is an
> **in-account read**, listed precisely so the first session with access is mechanical rather than exploratory.

## 1. THE GATE, MEASURED FIVE WAYS

| Path fetched | Result |
| --- | --- |
| `partners.playstation.net/` | **4,539-byte shell** — only the words *"PlayStation® Partners"* |
| `partners.playstation.net/support` | **identical 4,539-byte shell** |
| `partners.playstation.net/login` | **identical 4,539-byte shell** |
| `playstation.com/en-us/developers/` | **HTTP 404** |
| `sonyinteractive.com/en/` (corporate) | public — but a **newsroom / careers / impact** site with **no developer or partner section at all** |

**Conclusion, stated exactly:** there is **no public PlayStation developer information** — no submission steps, no
technical requirements, no commerce model, no revenue share, no payout terms. Anything asserted about PlayStation's cut
would be a guess, and **none is asserted here.**

## 2. WHAT THIS PLAN CAN THEREFORE BE

* **It cannot** list the requirements: they are behind a partner login, and a plan that invented them would be worse than
  no plan — it would be acted on.
* **It can** (i) fix the platform-independent entry work, (ii) name the exact in-account reads in the order they matter,
  and (iii) state the constraints that follow from what **is** known — the industry-shared rules, and our own boundaries.

## 3. THE ENTRY (platform-independent, and it happens once for all three)

1. **Apply to PlayStation Partners** and complete the NDA/agreement chain. Until that is done **no PlayStation work is
   schedulable** and no devkit exists (the portal's gate is the evidence).
2. **Prepare the concept/pitch material ONCE.** All three platforms gate on a submission of *something* — Xbox on concept
   approval for managed programmes, Nintendo on a Switch Access Request describing *"your development experience history
   and information on your planned project"*, Sony on whatever the partner agreement requires. **One pitch, three
   audiences**; the README's player half is the closest material we already have.
3. **Make the title ratings-ready** (§5) — shared with the other two platforms, and it can be done now.

## 4. THE IN-ACCOUNT READS, IN THE ORDER THEY DECIDE THINGS

Each is a **question with a named owner**, so an answer can be recorded as a fact the moment it is seen:

| # | Read this | The question it answers | Why it blocks |
| --- | --- | --- | --- |
| 1 | the **partner agreement / royalty terms** | the revenue share, and when it is paid | §7's cap and §8's top-up cannot promise a payout against an unknown cut |
| 2 | the **commerce / PS Store guideline** for in-game purchases | **whether an in-game store must use PlayStation's commerce**, and whether any external payment is permitted | decides whether our DLC store is a platform front-end (the Xbox answer) or something else |
| 3 | the **TCR / technical requirements checklist** | suspend/resume, connectivity loss, account linking, trophies, parity | XR-074 / XR-013 / XR-130 show how much of the Xbox list is industry-shared; Sony's must be **read**, not assumed |
| 4 | the **content / gambling / real-money policy** | whether winnings may convert to real-world value | Microsoft §11.14 prohibits it in the US and Xbox carries **XR-042 "No Real-Money Cash-Out"**; Sony's rule must be read before any payout path is designed |
| 5 | the **age-rating requirements** | whether IARC is used for digital-only, and any per-region blockers | our adult theme makes regional blocking plausible (Xbox: CERO Z / IARC 18+ **cannot release in Japan at all**) |
| 6 | the **payout and tax profile** screens | minimum payout, currency, tax forms, settlement day | the same precondition Microsoft makes public: enrolment is incomplete until the payout/tax profile validates |

**Record each answer in this file with the date it was read.** An unrecorded in-account fact is lost with the session.

## 5. RATINGS (shared with the other two platforms — and already actionable)

* **IARC** is *"a streamlined global age rating system for digitally delivered games and apps"*, **free at the point of
  use**, obtained by questionnaire (`globalratings.com`). Corroborated twice: Xbox's Game Publishing Guide — *"Participation
  is free at the point of use and publishers complete a questionnaire in Partner Center to generate a rating"* — and
  Nintendo's FAQ, which confirms IARC for digital-only titles.
* **Participating rating authorities (verbatim, `globalratings.com/participants`):** Australian Classification Board ·
  Classificação Indicativa (Brazil) · Digital Game Self-regulation Committee (Taiwan) · **ESRB** (US & Canada) · General
  Authority for Media Regulation (Kingdom of Saudi Arabia) · **GRAC** (Republic of Korea) · Indonesia Game Rating System ·
  **PEGI** (UK & Europe) · **USK** (Germany).
* **Honest limit:** `globalratings.com/storefronts` renders its list in JavaScript and returned **only its heading**, so
  **Sony's IARC participation is NOT verified from that page** — Nintendo's own FAQ and Microsoft's docs confirm it for
  their platforms. **For PlayStation, verify in-account** (read #5).
* **Product-shaped risk, recorded now:** Xbox documents that *"Any free product (including betas or demos) rated as CERO Z
  or IARC 18+ is not allowed to release on XBOX in Japan regardless of the distribution method"*, and that ratings consider
  *"functionality"* — explicitly including *"loot boxes, reward systems for user logins or other pressures to play"* — and
  **user-generated content**. This product has all three. **Expect an adult rating and expect regional exclusions**; plan
  for them rather than discovering them at submission.

## 6. WHAT OUR OWN ARCHITECTURE ALREADY GETS RIGHT FOR THIS PLATFORM

Independent of what Sony publishes, three of our boundaries already point the same way the industry's rules do:

1. **The console entrypoint is a virtual mirror with no chain rails** — `console_server.go` holds **0** signing calls,
   **0** application-call constructions, **0** raw-transaction submissions, **0 of 5** chain-rail routes, serves no HTML
   roots and binds loopback only.
2. **The voucher bridge is PC-side by construction** — `HandleVoucherConversion` lives in `onboarding_service.go`, a file
   the console build excludes by ruling.
3. **Nothing is minted to pay a player.** Player-to-player value moves inside the *virtual liability ledger*
   (`auction_service.go:240` shows the physical vault unchanged as funds move between virtual accounts).

Those three are the reason a console build is defensible at all; **read #4 is what confirms whether they are also
sufficient.**


