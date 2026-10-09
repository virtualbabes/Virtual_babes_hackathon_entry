# README EXPANSION PLAN — prepared 2026-09-20

> **Owner:** the preparation for expanding `README.md`. Kept as a FILE, not as chat context, because the operator will
> compact the session before directing the update.
> **ART EXPANSION (second pass, 2026-09-20):** on the operator's direction the art half now shows the game's actual
> artwork — **the six slideshow backgrounds** (captioned with the real slot defaults from `slide_theming.go`:
> `npc-anya-001`, `npc-crypto-seraph-002`, `npc-vbabes-046` for the menu; `npc-vbabes-054`,
> `npc-crypto-seraph-004`, `npc-anya-041` for the dashboard) · **the NPC cast** (117 frames: Anya 67 ·
> Vbabes 46 · Crypto-seraph 4, which are the free starter pack plus the purchasable pack) · **the tiered cast
> portraits** (9 characters across **Boss · Mini-Boss · Witch · Lady · cute**, each with a `.webp` still and its
> looping clips — **21** in total) · **the 9 NPC animation clips** in `Public/Assets/Videos/NPC/` · and **the video
> library** (`Learning-media`: the architecture `.pptx` + the arena guide, endgame blueprint, pyramid of power and
> architecture walkthrough). **Measured before embedding:** every asset is **git-tracked** (`git ls-files` — the
> first README pass's census checked existence only, which is not enough: an untracked file is broken for every
> reader but one), the slideshow defaults came from their owner file, and the frames were chosen from the
> **measured smallest** end of the pack because the originals run **721 KB – 8.3 MB**. Result: **58 embedded images,
> 49.5 MB of image weight** (the 8.3 MB frame is deliberately NOT embedded). **Open decision:** if that weight is
> too much for the GitHub page, the correct fix is a small **committed** thumbnail set (a tracked
> `Public/Assets/Images/README/`), never a link into `Public/Assets/Generated/**`, which is gitignored output.
>
> **README v4 (same session):** on the operator's direction the investor half was **re-aimed at the player** — the
> architecture/token/roadmap pitch was replaced by **`# 💠 Why This Game Is Worth Your Investment`** (what your hours
> become · earning by playing · real scarcity · depth · why enter now · what this is not · CTA). The technical content
> it carried was **relocated into the developer half**, not deleted: the value-flow diagram is now `### How value
> flows`, the chain/escrow facts are an **Architecture** row, and the roadmap already lives in *Where this is going*.
>
> **ART WEIGHT — FIXED 2026-09-20 (recommendation 2 executed):** **43.5 MB → 11.7 MB**. The 12 NPC-pack frames the
> README showed as **721 KB – 8 MB originals** (28 MB of the page) now display the **app's own light renditions** —
> copied out of `Public/Assets/Generated/` into a committed `Public/Assets/Images/README/` (6 × 512 px `slide` for the
> backgrounds, 6 × 128 px `thumb` for the cast sample) — and the 4 arena textures were resized **1024 → 360 px**.
> **The set is 16 files / 3.19 MB, referenced exactly, and NOT gitignored — but it MUST be committed alongside
> `README.md`, or those links break for anyone who clones** (git currently reports `?? Public/Assets/Images/README/`).
> **Residual weight: ~4.3 MB of `.webp` Effects and ~1.9 MB of portraits.** WebP has no encoder on this host, so a
> further trim needs a WebP-capable tool (the existing headless-Chrome tooling could do it); the alternative is simply
> showing fewer effects.
> **Status:** **EXECUTED 2026-09-20** — the README was expanded 189 → **402 lines**: a user-facing half first
> (What this is · Player's View · first hour · core game · how everything stacks up · endgame positions · the eleven
> areas · earn/own/brand · living world · honest Beta limits) and the developer half beneath a `# 🔧 For Developers`
> divider (Architecture · what is live and what is not · Development Setup · the seven gates · repository layout ·
> documentation · where this is going · License). Operator's directions: keep **Public Beta**, and lead with the user
> half. Verified: **0 broken relative links · 0 `.clinerules` references · mojibake 0 · all 38 images intact.**
> **Two corrections to this plan made during execution:** the taxonomy is **11 areas / 77 surfaces** (not 63 — the
> owner's file wins), and the blueprint is a **`.pptx`**. **Two extra defects found:** the old blueprint link used a
> **root-absolute path** (`/Public/…`), which GitHub resolves against the domain root and 404s — every relative link
> is now slash-less; and the emoji headings contain **U+202F narrow spaces**, so the editor's literal anchors could
> not match them (they were edited byte-safely through the file API instead).


## 1. WHY THIS README MATTERS MORE THAN USUAL HERE

`README.md` is **tracked and travels with every push**. `.clinerules/` (Keys, constitution, handoff) is
**deliberately local** and does not travel. For anyone who clones this repository, **the README is the only document
they receive** that describes the app — which makes every false statement in it expensive.

## 2. CURRENT STATE (measured 2026-09-20)

**189 lines · 8,763 bytes · mtime 2026-09-04** — before the full-corpus sweep, before the console target compiled,
before the workflow rewrite, before the aspect derivation closed. **All 189 lines have now been read.**
Headings: title · Executive Summary · The Industrial Seal · Tactical Visual Showcase · Featured Collection · Elite
Cosmetics & Effects · Icons · Items · Dynamic Arena Textures · Engineering Status · Infrastructure (Phase 1) ·
Development Setup (Prerequisites · Quick Start · Build Targets · Console Build) · The Vision (Storefront) ·
Supplemental Blueprint (PDF). Voice: product/marketing + a technical half. **The visual catalogue is real and correct
(all 38 image links verified to exist); the technical half is not.**

## 3. MEASURED DEFECTS — false statements, not preferences

| Line | It says | Measured reality |
| --- | --- | --- |
| 104 | "Confirmed live via particles.js + game.js" | the sweep measured **8** `??`-precedence sites in `particles.js` and **4** mangled `initwindow.AudioContext` lines in `game.js` |
| 107 | `- [x]` "Dividends — Logic implemented, **testing required**" | a tick contradicting its own text |
| 110 | `- [x]` "70+ tracks, **integration pending**" | same contradiction |
| 108 | "Factional Sovereignty — **JusticeService** + CourthouseService" | **`JusticeService` exists nowhere**; owners are `justice_handlers.go` + `courthouse_service.go` |
| 116 | "**Compliance Engine** — KYC/AML audit trails" ✅ | the doors are **UNAUTHENTICATED** (any caller, any wallet, any severity); the engine is a package global with no snapshot |
| 126 | "Build Verification ✅ — **Go build completes without errors**" | a bare `go build` is the **FALSE GREEN** (the split architecture excludes every `!js && !wasm` file) |
| 127 | "Dev Server ✅ **`launch_dev_server.ps1`**" | **retired and deleted**; live: `tools/server/dev_server.ps1` |
| 127 | "$**VBB**-gate" | $VBV |
| 134 | "Go **1.23+**" | the active toolchain is **1.25** |
| 140 | `cd Z:\Crypto_Draught\NFT-Seduction` | an absolute machine path in a "clone" step |
| 151 | `… -File **launch_dev_server.ps1**` | **deleted file** again |
| 171-172 | `build_console.sh` / `.ps1` | they live in **`tools/build/`**; and until 2026-09-20 the target **did not compile** |
| 189 | `[📘 …](Public/Assets/Learning-media/Virtualbabes_Arena_Architecture.pdf)` | **that path does not exist** (the DIRECTORY exists with 5 files — so the PDF is likely misnamed/missing) → the README's only blueprint link is broken |

**Passed, not failed — and worth keeping:** all **38** `src=` image paths were census-checked and **every one exists**.
### 3b. THE BROKEN BLUEPRINT LINK, RESOLVED BY MEASUREMENT

`Public/Assets/Learning-media/` contains **5 files and no PDF**:

| File | Size |
| --- | --- |
| `Virtualbabes_Arena_Architecture.pptx` | 27.3 MB |
| `Virtualbabes_Arena_Guide.mp4` | 41.9 MB |
| `Virtualbabes_Architecture.mp4` | 53.5 MB |
| `Endgame_Blueprint.mp4` | 53.0 MB |
| `The_Pyramid_of_Power.mp4` | 73.8 MB |

So the README's `…Architecture.pdf` link should become the **`.pptx`** (or a PDF exported from it). The four videos are
a ready-made media library — including **"Endgame Blueprint"** and **"The Pyramid of Power"**, which speak directly to
the endgame section proposed below. Also unused today: `arena_floor_tournament_semi_final.png`.

## 4. WHAT IS MISSING ENTIRELY (counted, not guessed)

Occurrences in the current README: **`verify:` 0 · RAG 0 · App-Aspect-Index 0 · World Dashboard 0 · Portfolio 0**;
`go build` 1; `console` 5. The README therefore never mentions: the **seven verification gates** · the **four build
targets** · the **documentation spine** (39-aspect index + Player View, the RAG corpus, the coverage ledger) · **what a
user actually navigates** (World Dashboard, Portfolio) · the **39 aspects** · the **ledger rules** · the **console's
own rules**.

## 5. CONTENT THAT ALREADY EXISTS AND CAN BE REUSED (sourced, not invented)

1. **Player View** — `App-Aspect-Index.md` → *"Player View: What the App Offers You"*: the premise; wallet → faucet →
   World Dashboard; the **11 categories with 63 features**; the Portfolio's **19 read-only categories**; how you earn;
   identity/branding; how you are measured; the 3D world; the console rule; honest limits.
2. **The core game** — **nine tiles**, each with its own **element and mood** (*ported from Triple Triad*), and the
   measured power stack: **Coalition +10% · Regional +5% · household +1% per 60 stat points (cap +10%) · Wanted
   mitigated by Cunning (2 per point) · fatigue mitigated by Nurturing · +25 loyalty · religious-leader blessing**;
   **SUDDEN DEATH** redistribution; **Standard / Bounty / Tournament**; spectate · frame-by-frame replay · wagers.
3. **The endgame positions** — career **tier 4** (top seats include Justice Commissioner, Underworld Boss) ·
   **Governor requires owning a region** · **one of the 24 religions** (buy-in while slots last, then **buyout only**) ·
   **LEGENDARY at 1000** on a six-rung ladder (EMERGING 50 · RISING 100 · ESTABLISHED 250 · HEROIC 500) · tournament
   champion · the six-axis household feeding the 3D overlay (`floor(statSum/50)`) · region control and the matrices ·
   the collection summits · the flywheel itself.

4. **Economy & ledger rules** — `uint64` micro-units only, **no float on any ledger path**; token-sink router +
   reconciliation; faucet/vault; the **reward-token registry**; the record rail (**22 primary families**);
   `RECORDS_DISPATCH` **OFF** by standing decision.
5. **Ownership & branding** — bonded assets worn across the ecosystem; **cards never brandable**; viewer-scoped card
   display; the free starter pack + the purchasable NPC pack (**117 frames**); the player-to-player market.
6. **Architecture** — split WASM (Go authoritative, WASM mirror); bit-exact board hash + replay recovery; loopback
   console; **secrets ENV-ONLY**; **no cloud models**; AI citizens own wallets.
7. **The seven gates** — aspects · routes-parity · reachability · routes · duplicates · handlers · api. Each fails
   closed, each carries a `--selftest`, and every one is currently invisible to a reader.
8. **The console model** — virtual mirror · **no admin entry point** · Nautilus **manual** · **the DLC→USDC rail ships
   with its cap**; the console registers **268 routes** and its five chain rails are excluded by ruling.
9. **The documentation spine** — `App-Aspect-Index.md` (1060) · `RAG/17_aspects_flow.md` (584) ·
   `RAG/19_coverage_ledger.md` (867) · `ToDo.md` (453) · `Problems.md` (3383) · `MASTER-PLAN.md` (519) ·
   `Rivalry-Matrix.md` (347) — all tracked, so all of them travel with a push.

## 6. ASSET INVENTORY A README CAN DRAW ON (all verified present)

`Public/Assets/Images` **191** files — Cards 16 · **portraits 30** · NPC-helpers **117** · Effects 18 · icons 4 ·
Cosmetics 3 · Items 3 — plus `Textures` 5 · `Audio` **85** · `Learning-media` 5.
**Rule: never link `Public/Assets/Generated/**`** — 235 derived files, **gitignored**, i.e. broken images after a clone.

## 7. OPEN QUESTIONS FOR BRENDAN (the update must not guess these)

1. **The status claim.** The badge says **Public Beta** while the standing constraints are `RECORDS_DISPATCH` **OFF**
   and *no live/on-chain testing until the UI is complete*. Keep it, or restate (Private Beta / in development)?
2. **How public should the internals be?** The gates, the workflow and `Problems.md` are quality machinery. Does the
   public README state the engineering discipline and a **known-limitations** section, or stay product-only?
3. **The media library.** ~223 MB of video + a 27 MB PPTX: link them from the README, or keep the repository light?
4. **The blueprint.** Point the link at the **PPTX**, or export a PDF first?

## 8. EXECUTION RULES FOR THE UPDATE

- **Preserve the product voice and the entire visual catalogue** (all 38 links verified); correct only what is false.
- **Every technical claim sourced** from §5 or measured in-session — never from memory.
- **Never link a path that does not exist, and never a gitignored one.**
- **Point only at what travels:** `AI-Brain/**` and root documents. **Never at `.clinerules/**`** — it is local by
  design, so a README link into it is broken for every reader but one.
- **Re-run the link census after editing** (the same src/href check used to prepare this plan).

## 9. ACCEPTANCE FOR THE UPDATE

Zero broken links · zero false statements from §3 · the **four build targets** documented · the **seven gates**
documented · the **console rules** stated · **no `.clinerules/**` path referenced** · the visual catalogue untouched.

it** — those images are absent for anyone who clones.

