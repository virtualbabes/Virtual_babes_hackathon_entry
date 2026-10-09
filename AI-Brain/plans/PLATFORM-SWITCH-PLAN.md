# NINTENDO SWITCH — PLATFORM PLAN (prepared 2026-10-08)

> **Owner:** this file owns everything Nintendo-platform.
> **Companion:** `CONSOLE-DLC-STORE-PLAN.md` owns the economy model (§7 cap, §8 top-up, §10 research).
> **Provenance:** Nintendo publishes its *process* and *eligibility* openly, and its *money terms* not at all. Every fact
> below was fetched; where nothing is published, that is stated as a **measured absence**, never filled in.

## 1. WHAT IS PUBLIC vs GATED (measured)

| Fetched | Result |
| --- | --- |
| `developer.nintendo.com/` | public: *"Self-Publish your Game — Once your game is complete, you can self-publish it on the Nintendo eShop with the price and release date entirely up to you."* |
| `/the-process` | public: the six steps, ending *"sign a **publishing agreement**, obtain an age rating, and submit your game for review"* — **no figure published** |
| `/faq` | public: *"Registering for the portal and downloading the tools is **completely free**"*; *"the price, release date and content are all set by you"*; an IARC rating *"for no fee"*; the only cost named is **development hardware** |
| `/register` | public: the full registration shape (§2) |
| `/tools` | public but thin: **Unity** named; the middleware list itself is JS-driven and returned almost nothing |
| `/news` | **HTTP 404** |

**Conclusion:** Nintendo publishes **who may register, that it costs nothing but hardware, and that the developer sets the
price** — and **nothing about revenue share, payout schedule, thresholds or royalty statements.** Those sit behind
*"accept the Non-Disclosure Agreement and Terms of Service"* and, for Switch specifically, behind a **separate
application**.

## 2. ELIGIBILITY AND REGISTRATION (all public, all actionable today)

* **Individuals are welcome:** *"You can register even if you are an individual and do not represent a company."*
* **Legal age:** *"You must be the legal age of majority in your country of residence in order to submit a developer
  application to Nintendo."*
* **No experience or business address required:** *"No prior development experience is required"*; *"we accept home
  offices, you don't need a business address."*
* **Company vs individual:** a company applicant becomes the organisation administrator and may add users; an individual
  *"will not be able to add other users to your organization. Other than this limitation on user management, there is no
  difference from registering as a company."* · *"Please do not register the same organization multiple times."*
* **Development environments:** *"Resources are provided for multiple development environments including **Unity and native
  C++** software development."*
* **Switch access is a SECOND gate:** *"If you are interested in Nintendo Switch development and publishing resources you
  may apply for access by filling out the **Nintendo Switch Access Request** form that you will find in the GETTING STARTED
  section after registering… Please enter your development experience history and information on your planned project."*
* **Closed platforms:** *"New titles and patches cannot be released for Nintendo 3DS and Wii U."*
* **Tooling reality:** the portal is *"tested with the latest Google Chrome and Mozilla Firefox installed in the Windows
  environment"* — a Windows-based workflow.

## 3. THE IN-ACCOUNT READS (the entire money side lives here)

| # | Read this | The question it answers |
| --- | --- | --- |
| 1 | the **publishing agreement** (named in `/the-process` as a release step) | the revenue share, the royalty statement, and when it is paid |
| 2 | the **eShop in-game purchase / add-on rules** | whether DLC and in-game stores must use Nintendo's commerce, and whether any external payment is permitted |
| 3 | the **technical requirements checklist** | suspend/resume, connectivity loss, account linking, save data, handheld-vs-docked parity |
| 4 | the **content / gambling / real-money rules** | whether winnings may convert to real-world value — the Xbox answer is a flat prohibition (**XR-042 "No Real-Money Cash-Out"**), and Nintendo's must be **read**, not assumed |
| 5 | the **age-rating requirements** | IARC is already confirmed for digital-only titles at no fee; the open question is any **per-region blocker**, like Xbox's Japan exclusion for 18+ |
| 6 | the **payment / tax setup** | thresholds, currency, tax forms |

**Record each answer here with its date.** Nintendo's own portal is the only source for any of it.

## 4. WHAT IS KNOWN THAT ALREADY SHAPES THE BUILD

1. **Pricing is ours** — *"the price, release date and content are all set by you"* — so the economy plan's §8 top-up and
   DLC ladder are ours to set, subject only to read #1's cut.
2. **The named development environments are "Unity and native C++", and this project's game engine is Go/WASM**
   (`main.go`, `Public/main.wasm`). **That is a porting question, not a detail:** Nintendo names neither Go nor WebAssembly,
   so the console client is either a Unity/native-C++ re-host of the game rules or a re-implementation that calls our Go
   server for authority. **This is the largest single engineering fact in this plan.** The server side (economy, records,
   authority) ports as-is because it is plain Go over HTTP/WS.
3. **The server stays ours** — nothing Nintendo publishes forbids a title talking to its own backend, but the
   connectivity-loss behaviour Microsoft tests (XR-074) is the industry norm and should be assumed until read #3 says
   otherwise.
4. **Our console segregation is the same asset here** — 0 chain rails, the voucher bridge PC-side, and value moving inside
   the *virtual liability ledger*. If Nintendo's read #2 or #4 permits less than Microsoft's, ours is already the minimal
   design.

## 5. WORKSTREAMS — WHAT IS NINTENDO-SPECIFIC vs SHARED

| # | Workstream | Nintendo-specific? |
| --- | --- | --- |
| **A** | **Client re-host** — Unity or native C++ implementing the game rules, or a thin client over our Go server | **YES — the largest item.** Unlike Xbox there is no "ship it as Win32" shortcut (Unity is a named path) |
| **B** | **In-game purchases on Nintendo's commerce** | **YES** — mechanism is in-portal (read #2) |
| **C** | **`ConsoleAssetReceipt` writer** — entitlement from a verified platform purchase | ✅ **BUILT 2026-10-08** (`console_entitlement.go`) — **one implementation** for every platform, so Switch is a consumer, never a second owner |
| **D** | **Account link / unlink / SSO** | **shared shape**, platform-specific API |
| **E** | **Save data + handheld/docked parity** | **YES** — Nintendo's own dual-mode parity case |
| **F** | **Connectivity-loss resilience** | shared (the industry norm) |
| **G** | **Age-rating submission** | shared, via IARC |
| **H** | **The store itself** (catalogue, prices, availability) | shared with the economy plan's §8 top-up catalogue — **one owner, not three** |

**The rule this table exists to enforce:** *one* entitlement path (**C**) and *one* catalogue (**H**) serve all three
platforms. Only **A**, **B**, **D** and **E** are genuinely per-platform, and duplicating C or H per platform would create
three owners of one fact.

## 6. ACCEPTANCE GATES

1. **Registration and the Switch Access Request are the first real milestones** — both are public, free and actionable
   today, and nothing else on this platform is schedulable until they are complete.
2. **No money path is designed before reads #1, #2 and #4.** The cut, the permitted payment rails and the real-money rule
   are all unknown, and Nintendo's answer may be stricter than Microsoft's published prohibition.
3. Any code that lands meets the repository's own gates: the seven `verify:*` gates, the full `go test .`, all three builds
   rc 0, and `main.wasm` proved untouched wherever the client is uninvolved.
4. **A port is proven by running on the platform, not by compiling** — a Unity/native-C++ re-host has no meaningful local
   proxy.

## 7. HONEST LIMITS

* **Nintendo publishes no money terms at all.** The cut, the schedule and the royalty statement are unknowns, and none of
  them is guessed here.
* The **middleware list is JavaScript-rendered** and could not be read; only **Unity** was retrievable, plus the
  *"Unity and native C++"* statement on `/register`.
* **Switch 2's developer terms are not public** beyond the existence of a separate access request.
* **The Go/WASM engine is not a named Nintendo development environment** — this is a porting project, and §5 row **A** says
  so plainly rather than implying the existing client runs.


