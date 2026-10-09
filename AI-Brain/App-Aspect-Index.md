# NFT-Seduction: App-Aspect Index

> **Consolidated reference** — Covers ~99% of app functionality across all systems.
> **Last updated:** 2026-09-07
> **Source:** Deep-dive analysis of 347 files, 7909 nodes, 32728 edges + onboarding/DLC/console systems.

---

## Table of Contents

- [Player View: What the App Offers You](#player-view-what-the-app-offers-you)
1. [Architecture](#1-architecture)
2. [Onboarding & Player Entry](#2-onboarding--player-entry)
3. [Battle System (Core Gameplay)](#3-battle-system-core-gameplay)
4. [Card Enhancement (NFT Upgrades)](#4-card-enhancement-nft-upgrades)
5. [Shop & Item Economy](#5-shop--item-economy)
6. [QuickPlay](#6-quickplay)
7. [Matchmaking & Multiplayer](#7-matchmaking--multiplayer)
8. [Factions & Careers](#8-factions--careers)
9. [Justice System](#9-justice-system)
10. [Underworld & Criminality](#10-underworld--criminality)
11. [AI Citizens](#11-ai-citizens)
12. [Faith System](#12-faith-system)
13. [Theme Engine](#13-theme-engine)
14. [Entity Markets](#14-entity-markets)
15. [Pet World & Entity Events](#15-pet-world--entity-events)
16. [Tournaments](#16-tournaments)
17. [Seasonal Events](#17-seasonal-events)
18. [Achievements & Trophies](#18-achievements--trophies)
19. [Auctions](#19-auctions)
20. [Spectator Mode](#20-spectator-mode)
21. [Governance](#21-governance)
22. [Creator Economy & DLC Store](#22-creator-economy--dlc-store)
23. [Multi-Chain Bridge](#23-multi-chain-bridge)
24. [Industrial Loop](#24-industrial-loop)
25. [Persistent Identity](#25-persistent-identity)
26. [Local LLM Pipeline](#26-local-llm-pipeline)
27. [Web-3D Client](#27-web-3d-client)
28. [Item Shop Archetype (Player-Created Items)](#28-item-shop-archetype-player-created-items)
29. [Vehicles & World Content](#29-vehicles--world-content)
30. [Counterfeit System](#30-counterfeit-system)
31. [Admin Tools](#31-admin-tools)
32. [Dev/Game Hub (Composable Framework)](#32-devgame-hub-composable-framework)
33. [Nautilus DEX Path (Console Creator Payouts)](#33-nautilus-dex-path-console-creator-payouts)
34. [Console Linking & Manufacturer APIs](#34-console-linking--manufacturer-apis)
35. [Economy Bootstrap & State Recovery](#35-economy-bootstrap--state-recovery)
36. [Session Watchdog & Player Eviction](#36-session-watchdog--player-eviction)
37. [Composable Mechanic Framework](#37-composable-mechanic-framework)
38. [Infrastructure & Security](#38-infrastructure--security)
39. [The Civilization Flywheel](#39-the-civilization-flywheel)

---

## Player View: What the App Offers You (the player's view of the same 39 aspects - NOT one of them)
> **This section is not a second index.** It is the SAME 39 aspects projected for a player instead of an engineer: what you can actually do, in the words the World Dashboard uses. Every claim below is taken from the surfaces the client declares — 11 launcher categories, 63 features, 19 read-only analytics categories — and the technical section for each aspect follows beneath it. Where something is a placeholder or an operator-only surface, it says so.

**The premise.** You own a **deck of living cards**, and around that deck the app grows a civilisation: a career, a territory, a household of AI citizens, pets and vehicles, a business, a church, a reputation — inside an economy where every price is set by the server and every value is real **$VBV**.

**Getting in.** Connect a wallet (Voi is the home chain) → claim from the **faucet** → you land on the **World Dashboard**, the single front door. It is deliberately two-tier: **11 category buttons**, and choosing a category reveals its features. Nothing is buried and nothing is duplicated; **access to every surface — including your own Portfolio — originates here**.

### The eleven things you can do
| Category | What it gives you |
| --- | --- |
| 🎮 **Player Hub** | Portfolio (Analytics) · Identity Editor · Player Stats · Achievements · Deck Manager · Card Titles · Card Progression · Character Mood · Post-Match Wagers · Theme Engine (element · mood · projections) |
| ⚔️ **Careers & Factions** | Career Pathways · Criminality (courthouse, bounties, Intel-Agent intercepts) · Justice Hegemony · Underworld Contracts · Counterfeit Scanner · Faction Quartermaster with **separate Justice and Underworld shops** |
| 🎒 **Assets** | Inventory & Equipment · Equipment System · **Companion Kennel** (breed & groom) · **Garage** (build & upgrade vehicles) · **AI Citizens** (spawn & develop) · **Children Bots** (learning pathways) · **Bonded Branding Studio** · Companion Arena · Vehicle Arena · Club Foundry · Orphan Cleaner |
| 💰 **Economy & Trade** | Markets & AMM · Dividend Yield · Loan Terms · **Black Market** · Bridge Router · Advertising |
| 🏛️ **Governance** | Governance Chambers · Governor Office · **Territory Map** · Regions · Compliance · Regional Leaderboard |
| 🃏 **Play & Board** | Game Board · Create Match · Match Arena · Multiplayer · Game Modes · Campaign · Tutorial · Locations · NPC Taunts · Tea House · Zen Garden |
| 🌍 **World & Events** | Season · World Events · Tournament Brackets · Replay Viewer · Treasure Map · World Content · Industrial Loop |
| ⛪ **Faith & Church** | Faith System · Religion Governance (24-slot chain, buyouts, rivals) · Church Storefront |
| 🏆 **Competition** | **Rivalry Matrix** · Rewards Center |
| 🎨 **Creator Economy** | Creator Store · Launchpad — sell what you make, earn royalties |
| 🛠️ **System & Ops** | Infrastructure · Gaming OS · Dev/Game Hub · Local Model · Maintenance Mode · System Messages · Report History · **Admin Console (signature-gated)** · five operations consoles (Community · Extended · Security · System · Utilities) |

### The core game — the cards
**The board is nine tiles** — a 3×3 grid, and each tile carries its own **element** and its own **mood** (Volatile · Serene · Spirited · Grounded). The element system is *ported from Triple Triad* (the source says so), so **where** you place a card matters as much as which card you place.

**What decides a fight.** A card's effective power is its own value plus everything you have built around it — every layer integer and server-authoritative:
- **Coalition +10%** when an allied member defends a partner's turf
- **Regional +5%** when the field belongs to the region's owning club
- **Your household lends power to every card you play**: the companion pets and vehicles you own add to your cards (+1% per 60 stat points across your certified household, capped at **+10%**)
- **Wanted level is a penalty — mitigated by Cunning** (every point of Cunning gives back 2)
- **Fatigue is a penalty — mitigated by Nurturing**
- **+25 loyalty** when the card is loyal to you
- **Tile mood and element interactions** on top of all of it
- **A religious-leader card blesses your side**, its faith composed against the field's faith

All of it is snapshotted into the match at the start, so **the board you preview and the board the server resolves agree** — down to a bit-exact state hash the client verifies against the authority.

**How a match ends.** Cards accrue **fatigue**; cards can be **jailed**; and when the board clears the match goes to **SUDDEN DEATH** — hands are re-shuffled and redistributed, fatigue resets so both sides begin the tie-breaker at peak power, and the winner takes the pot. Matches come in **Standard**, **Bounty** and **Tournament** flavours, can be **spectated** and **replayed frame-by-frame**, and can carry a **post-match wager** between onlookers.

### The endgame — the positions you can actually reach
- **The career summit: role tier 4** — the highest the engine grants (this is a measured ceiling, not marketing). The named top seats in the tree include **Justice Commissioner** and **Underworld Boss**, and every promotion gate names the ONE thing you are missing: lesson level, role tier, sustained $VBV, civil rank, or path.
- **The civil summit: Governor** — which requires **owning a region**, not merely clubs. `user → manager → governor` is *derived from what you own*, never declared.
- **One of the 24 religions.** The chain is capped at **24 server-wide**; while slots remain you buy a governorship, and once it is full **only a buyout** moves one — with every buyout routed to the faucet.
- **LEGENDARY identity** — the top of a six-rung ladder built from your record: NEWCOMER → **EMERGING** (50) → **RISING** (100) → **ESTABLISHED** (250) → **HEROIC** (500) → **LEGENDARY** (1000).
- **Tournament champion** — the pot is split and the archive records it.
- **A dominant household** — six entity axes, each capped at **100**, feed the 3D world's power overlay (`floor(statSum / 50)` added to an entity's effective level), so a fully developed family visibly reshapes the map.
- **Region control and the matrices** — hold districts, hold the region, hold your standing in the rivalry matrices that price your victories.
- **The collection summits** — titles, achievements, your highest-five card progression, and a Portfolio that reports every association the engine holds for you.
- **And the flywheel itself** — fees, entity dividends, creator royalties, club revenue, advertising, the industrial loop and the vault: an economy you can build on top of the game rather than merely play inside.

### Your record: the Portfolio


Nineteen **read-only** analytic categories cover everything the engine holds about you — Overview · Record · Stats · Identity · Deck · Progression · Titles · Moods · Economy · Careers · Entities · Achievements · **Pets · Rivalry · Clubs & Alliances · Bonded Assets · Faith · Legal & Custody · Obligations**. It never transacts; it states plainly when a figure could not be read rather than showing you a zero.

### How you make your way
Win matches and tournaments · take **entity dividends** from businesses you invest in · claim **bounties** and courthouse work · earn **seasonal event** rewards and treasure caches · run a **club** and its shop · sell **creations** for royalties · take **loans** and leases for capital · and compound it through the **rivalry matrices**, where your career and your chosen path decide who your rivals are and what your victories are worth.

### Your identity and branding
Your **Bonded Assets** are your own image, wearable on **anything you own** — items, themes, bots, pets, vehicles, churches, clubs, shops, world content. **Cards are never brandable**; what you *may* do is choose how **you** see the cards in front of you. You can **sell your art to other players** and theme the app's own slideshows with it.

### How you are measured
Career paths and promotion (*unlocked, never forced*) · a **civil rank** (user → manager → governor) earned by owning clubs and territories · rivalry standing · leaderboards · titles, achievements and moods · governance weight in your region.

### The 3D world
Regions render as a living 3D map where **entity power** — your pets, vehicles and bots — shapes the skyline, and clicking a region reads its live world-dynamics signature.

### On console (operator decision, 2026-09-20)
**No crypto on that device, no admin entry point, no self-service spending.** The console is a **virtual mirror**: you play, your account is linked and bridged, and it **awaits a manual bridge** to the PC version. $VBV is withdrawn as $VOI **by hand at Nautilus**.

### Honest limits, stated plainly
The two arenas are honest **placeholders** that point into the 3D world · the Dev/Game Hub is a **read-only catalogue** until a route owns its prices · the Admin Console is **operator-only and signature-gated** · and the crypto rails live on the **PC version**, capped and manually bridged by design.


## 1. Architecture

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §1 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**Split-WASM design:** Two binaries from one codebase.
- `server-bin.exe` (`!js && !wasm`) — Authoritative source of truth
- `main.go` (`js && wasm`) — Client-side sim for instant UI feedback
- `console_server.go` (`console`) — Loopback dev entrypoint

**Security model:** WASM client mimics server behavior; state hashes compared to detect cheating. Server can always prove what really happened.

**Determinism:** All math is uint64 integer-only (PILLAR 2). No floats on ledger-facing paths. FNV-1a hashing for reproducibility.

**Persistence:** Gzip-backed atomic commits with `.tmp` + rolling backups. 15-minute snapshots for citizens, religions, churches.

**Cross-platform:** Windows, Linux, Android (`build_mobile.sh`/`build_mobile.ps1` — `GOOS=android GOARCH=arm64`), iOS, console (Xbox/PS/Nintendo via `console_server.go`).

---

## 2. Onboarding & Player Entry

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §2 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**OnboardingService** (`onboarding_service.go`, 471 lines) — Primary entry gate for external liquidity.

**Key flows:**
- `HandleVoiOnboarding` — Voi Network wallet provisioning via Algorand ARC-200
- `HandleVoucherConversion` — Convert non-crypto vouchers to $VBV virtual liability (1:1 parity)
- `HandleIdentityRefresh` — Refresh/re-link cross-platform identity

**Session Watchdog** (see §36):
- 24-hour session limit enforced via `AuditActivePlayerSessions` (10-min audit ticker)
- Liquidity & identity integrity checks
- Automatic eviction with `SESSION_EXPIRED` reason code

**Linked Wallets:**
- Console UID linking for cross-platform identity
- Browser players can link console wallets
- `Verified` flag required for redemption gateway processing

**Sybil Protection:**
- Multi-node failover for historical Sybil scans
- Onboarded wallets loaded from indexer
- "Drain and re-claim" vulnerability hardened

---

## 3. Battle System (Core Gameplay)

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §3 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**Grid:** 3×3 board. Each card has 4 directional powers (top, right, bottom, left).

**Capture mechanic:** Place card adjacent to enemy → if your facing power > enemy facing power → capture (card flips to your side).

**Match rules (toggleable):**
| Rule | Effect |
|---|---|
| Open | Standard play |
| Power_copy (Same) | 2+ adjacent equal-power cards flip together |
| Power_up (Plus) | Matching sum pairs flip together |
| Elemental_sync | Board tiles have Moods; card Mood vs tile Mood = ±50 power |
| Fallen_penalty | Captured cards lose 20 Artifact permanently |
| Artifact_bonus | Flat power bonuses from items apply |
| Sudden_death | Tiebreaker rules |

**Combo attacks:** Same groups (2+ matching powers) and Plus groups (matching sums) cascade captures.

**Full power calculation (`getEffectiveServerPower`):**
```
Base Power (directional value)
+ Artifact (flat bonus)
+ Loyalty Bonus (+25 if Loyalty >= 100)
- Fallen Penalty (-50 if Fallen)
+ Mood Catalyst (+50 from item buff)
± Elemental Sync (±50 Mood vs tile)
± Faction Bonus (+10% Justice vs Outlaw/Fallen, +10% Underworld vs Justice/Clean)
+ Justice Card Power Bonus (per-card vs Wanted>=15)
- Wanted Penalty (-5 per Wanted, mitigated by Cunning ×2)
- Fatigue Penalty (-1 per point >50, mitigated by Nurturing)
± Religious Leader (+40 vs rival faith, -30 vs same faith)
```

**Match outcome:** More cards on board at end wins. Winner can jail loser's rarest card (Prisoner Rule).

**Board hash:** SHA-256 of card IDs + owners + artifacts + turn index. Verified by WASM client for anti-cheat.

---

## 4. Card Enhancement (NFT Upgrades)

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §4 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

### Mutation (PILLAR 6)
- **Cost:** 500 $VBV
- **Success:** Re-allocate power across 4 sides
- **Failure:** Permanent -50 Artifact ("mutation scars")
- **Mitigation:** Mutation Insurance (100% success), Staff Training (+5% success)
- **Forensic audit trail:** All procedures recorded on blockchain permanently

### Artifact Enhancement
- Flat power bonus added to every side
- Sources: Mood Catalyst (+50 for 3 matches), BuiltItem PowerScale, battle service
- Captured cards with Fallen_penalty lose 20 Artifact permanently

### Loyalty System
- +25 flat power at max loyalty (100)
- Increases through usage and care
- Loyalty Pledge item: +10 loyalty immediately

### Fatigue Loop
- -1 power per usage above 50
- Nurturing stat reduces impact (1 power back per Nurturing point)
- Stamina Stim: -20 fatigue immediately
- Hyper-Stim: Reset fatigue for entire deck

---

## 5. Shop & Item Economy

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §5 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

### Item Categories (ClubType families)

| Category | Purpose | Example Items |
|---|---|---|
| Elemental | Mood manipulation, elemental sync | Mood Catalyst, Grounded Shield, Prism Shield |
| Tactical | Rule manipulation, hand visibility | Rule Breaker, Intel Report, Hyper-Stim |
| Vitality | Fatigue recovery, loyalty, mutation | Stamina Stim, Loyalty Pledge, Staff Training |
| Intelligence | Espionage, treasury raids | Cyber-Audit, Deep-Scan Decryptor, Legal Pardon |
| Hardware | Heist defense, trap detection | Tripwire, Sentry Turret, Guard Dog |
| Justice | Power bonuses vs outlaws | Enforcer Card, Truth Serum, Reputation Shield |
| Underworld_Admin | Criminal operations | Regulatory Bypass, Illicit Commission Permit |

### Shop Gate Requirements
- **RequiredMojo** — Club Mojo threshold
- **RequiredRole** — Career role match (Manager, Security, Judge, etc.)
- **IsMasterTier** — Requires Regional Governor (2+ territories)

### Item → Battle Effects
| Item | Battle Effect |
|---|---|
| Mood Catalyst | +50 power (3 matches) |
| Grounded Shield | Immunity to Mood penalties (5 matches) |
| Prism Shield | Reflect Mood penalties to opponent |
| Rule Breaker | Force PLUS trigger (1 match) |
| Intel Report | See opponent hand (3 matches) |
| Hyper-Stim | Reset fatigue for entire deck |
| Stamina Stim | -20 fatigue immediately |
| Loyalty Pledge | +10 loyalty |
| Staff Training | +5% mutation success (24h) |
| Mutation Insurance | 100% mutation success (1 proc) |
| Enforcer Card | +10% power vs Outlaws |
| Truth Serum | Reveal opponent buffs (30s) |
| Reputation Shield | Absorb reputation loss |
| Forensic Audit Kit | -50 Artifact on target |
| Legal Pardon | Clear 50% Wanted Level |
| Market Freeze | Disable share trading (Wanted 20+) |
| Raid Jammer | -15% heist success vs club |
| Ghost Protocol | Hide match outcome from ticker |
| District Stabilizer | -50% Mojo decay (48h) |

---

## 6. QuickPlay

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §6 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**What it is:** A fast-entry match mode accessible from the menu constellation.

**UI:** `quick-play-screen` overlay (`menu-constellation.js` `buildQuickPlay()`, `openQuickPlay()`, `closeQuickPlay()`).

**Audio:** 3 dedicated ambient tracks:
- `quick_play_ambient_1.mp3`
- `quick_play_ambient_2.mp3`
- `quick_play_ambient_3.mp3`

Managed by `audio.js` (`Active_Quick` state) and `audio_context.js`.

**User preference:** `showQuickPlay: true` in `user_preferences.js`.

**Visual:** `.quick-play-btn`, `.quick-play-icon`, `.quick-play-label` SCSS styles.

---

## 7. Matchmaking & Multiplayer

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §7 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**Queue system:** `matchmakingPool` in Lobby struct. Enter queue → paired with waiting player.

**AI matchmaking:** `AICitizenEngine.matchAIWithHuman` pulls AI citizens into real matches when humans wait.

**Tournament lock:** Players in active tournaments excluded from casual matchmaking.

**WebSocket updates:** Real-time `lobby_update` with active player list.

---

## 8. Factions & Careers

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §8 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

### Faction Assignment
`GetHegemonyPath(JobRole)` → JUSTICE or UNDERWORLD

| Faction | Bonus | Against |
|---|---|---|
| JUSTICE | +10% power | Fallen cards, Outlaws (Wanted ≥ 15) |
| UNDERWORLD | +10% power | Justice-aligned, Clean (Wanted ≤ 2) |

### Justice Card Power Bonus
Additional per-card bonus via `justiceService.GetPowerBonus(pID, vsOutlaw)` — stacks with faction bonus.

### Career Roles (examples)
- **Justice:** Enforcer, Warden, Justice Recruiter, Forensic Analyst, Intel-Agent
- **Underworld:** Fence, Smuggler, Kidnapper, Heist Planner, Launderer, Hostage Host
- **Hybrid:** Tax Auditor, Lawyer-Commissioner, Judge, Arc-Net Operative

### Career XP
- $VBV-gated `ComputeScaledXP` before `TrackCareerXP`
- Rival pair bonuses via `EvaluateCrossCareerXP`
- 28 call sites across `battle_service.go`

---

## 9. Justice System

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §9 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**JusticeService** (`justice_service.go`):
- `GetPowerBonus()`, `CalculateJusticePowerMultiplier()` (base +10% vs Wanted≥15)
- Justice card pool: ENFORCER, MEDIATOR, WARDEN, COMMISSIONER
- Truth Serum: Reveal opponent buffs (30s)
- Reputation Shield: Absorb reputation loss
- Bounty Dashboard: Warden+ tier gate (3 cards + rank≥2)
- Mission generation: 24h expiry, COMPLETED/EXPIRED status

**Bounty Board:** Real-time tracking of high-Wanted players. 5% tax on bounty payouts funds Justice recruitment pool.

---

## 10. Underworld & Criminality

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §10 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

### Kidnap Gambit
- Multi-slot `VictimRegistry` with active kidnappings
- Ransom economy: Pay to recover, or sell on Black Market
- Insurance recovery available

### Underworld Contracts (career-gated)
| Contract | Requirement |
|---|---|
| Jail Capture | Combat on club territory |
| Sabotage District | Target Regional Governor |
| Governor Rumor | Spread rumor about Governor |
| Governor Heist | Heist Governor's club |
| Governor Favorite Kidnap | Kidnap Governor's favorite card |
| Mojo Stabilizer Heist | Heist club with District Stabilizer |
| Arena Center Alliance Heist | Heist allied club |
| Governor Hostage Liberation | Free hostage held by Governor |

**Mechanics:** Accepting increases Wanted Level. Abort with penalty. Commission: 8% standard, 5% Fence Tier 3+.

### Black Market
- Wanted/Cunning-gated access
- Stolen card tags
- Fenced goods commission tracking
- Buy/sell kidnapped cards

### Cyber Espionage
- Cyber-Audit: Reveal club treasury
- Cyber-Lock: Prevent audits (24h)
- Cyber-Counter: Identify auditor
- Deep-Scan Decryptor: Reveal inventory + buffs
- District Scanner: Reveal all hardware traps
- Cloak Disruptor: Reveal outlaw on Bounty Board
- Cyber-Jammer: Prevent sabotage warning
- Signal Dampener: Hide from Bounty Board (24h)

### Sabotage
- Disable club hardware defenses
- Sabotage Warning system
- Raid Jammer: -15% heist success vs club

---

## 11. AI Citizens

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §11 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**Own-wallet mandate:** Every citizen MUST hold its own wallet. Enforced at spawn.

**10 Autonomous Behaviors (BehavioralTick):**

| Dynamic | Method | Tier | Probability | Effect |
|---|---|---|---|---|
| Faith Rituals | executeFaithRituals | ≥3 | Every tick | RitualsDone++, FaithCoherence+ |
| Marriage | executeMarriage | ≥Journeyman | 10% | WifeWallet set, DomesticCoherence+ |
| Breeding | executeBreeding | ≥Expert, married, certified | 5% | BotChild spawned, DomesticCoherence+ |
| Pet Adoption | executePetAdoption | ≥Journeyman | 8% | PetIDs+, DomesticCoherence+ |
| Justice Enforcement | executeJusticeEnforcement | ≥Expert | 7% | Captures outlaws, treasury+, reputation+ |
| Employment | executeEmployment | EMPLOYED | Every tick | Salary from owned club |
| Rivalry | executeRivalry | ≥Journeyman | 4% | Reputation swing, treasury drain |
| Market Activity | executeMarketActivity | ≥Expert | 10% | Entity investment or black market |
| Career Progression | executeCareerProgression | Any | 3% | Tier++ (XP + reputation gated) |
| Event Hosting | executeEventHosting | ≥Master | 2.5% | Hosts entity events |

**HTTP Routes:** `/api/ai/citizens/*` (spawn, marry, breed, adopt-pet, progress, list, stats, free-agents)

**Dev seed:** 2 citizens spawn on boot.

**Citizen Weight Formula:**
`W_citizen(owner) = Σ [(1+Tier) × (1+AttachmentTier/3) × (1+Reputation/100) × (1+BusinessCount×0.1)]`

---

## 12. Faith System

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §12 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**Structure:** Club-like social structure. 24 religions seeded on boot across 9 pathways × 3 dogma types.

**Dogma Types:**
- **Purist** — Heresy-war rivalry bonus vs other faiths
- **Syncretic** — Coalition synergy with allied dogmas
- **Orthodox** — Same-faith weaken (orthodoxy/heresy-purge dominance)

**Church Storefront:**
- Open church (~5,000 $VBV, like club/foundry)
- Add members, add items, perform rituals
- Ritual fees raise FaithCoherence
- Tournament aggregation for religious buffs

**Faith War Gambit:**
- Stake FavoriteCardID to the pot
- Win → card returns + winnings
- Lose → card jailed/seized to winning faith's kitty

**Religious Leader Cards:**
- Card fields: `Religion` + `IsReligiousLeader`
- +40 power vs rival faith (blessing)
- -30 power vs same faith (orthodoxy purge)
- Cross-mode: position carries into career/tournament battles

**Faith Coherence Formula:**
`FaithCoherence(region) = Σ_members (rituals_done×10 + godly_acts×5 − blasphemy×8)`

**AI Participation:** Citizens with Tier≥2 get DogmaTag on spawn (FNV-1a deterministic). Tier≥3 perform rituals each tick.

**World Impact:** High Awe → cathedral geometry, golden light, calm weather. Low Awe → bleak, blighted. Market-weather modulated by Awe.

---

## 13. Theme Engine

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §13 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**Theme Vector:** `T = (tone, element, intensity, entropy)`
- **tone** ∈ {BENEVOLENT, NEUTRAL, MALEVOLENT}
- **element** ∈ {ORDER, CHAOS, NATURE, MACHINE, VOID}
- **intensity** = holdings weight + social status + asset rarity
- **entropy** = conflict across inputs (amplifies variance)

**Inputs:** Cosmetic MoodTags, entity market value (MAJOR), social status, disposition, standing.

**Outcome Bias:** `BIAS_MAX = 15%`. Good themes → good outcomes. Applied at combat, treasure, breeding, rivalry, event resolution points.

**World-Dynamics Signature (10 terms):**
`W_MARKET_VITALITY=45000, W_CITIZEN_GRAVITY=55000, W_THEME_COHERENCE=25000, W_EVENT_DYNAMICS=30000, W_PROFILE_IMPACT=50000, W_FAITH_COHERENCE=40000, W_DOMESTIC_COHERENCE=35000, W_RUMOR_COHERENCE=20000, W_ECON_PERK=30000, W_ENTITY_LEGITIMACY=40000`

**Market-as-Weather:** Deep reserves = clear skies, thin reserves = storms, dividend freezes = lightning, monopoly pressure = volcanic eruptions, rumors = aurora distortion.

**Market Disasters (severity tier):** Ocean tides (reserve collapse), wildfires (yield death-spiral), volcanic eruptions (monopoly pressure), droughts (illiquidity), lightning (dividend freeze), aurora (rumor manipulation).

---

## 14. Entity Markets

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §14 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**AMM Bonding Curve:** Quadratic slippage-aware pricing. Max 25% portfolio per entity (anti-whale).

**Market Signals → Weather:**
- Reserve depth → sky clarity
- Share elasticity → wind/turbulence
- Dividend yield → warmth/sunlight
- Dividend freeze → lightning
- Concentration → fog/whale-pressure
- Whale slippage → squalls
- Rumor manipulation → aurora
- Trade flow → precipitation

**Loans:** Borrow against Market Tokens. Default → liquidation.

**Auctions:** Server-authoritative escrow, 10% commission, real-time WebSocket notifications.

---

## 15. Pet World & Entity Events

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §15 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**EntityStats:** Shared uint64 vector (Speed/Intelligence/Willpower/Strength/Charisma/Agility, 1..100). NO FLOAT.

**Power Overlay:** `ComputeEffectivePowerLevel(baseLevel, stats)` = `baseLevel + floor(statSum/50)`, clamped to 600.

**Event Types:** TRAIN (stat XP), REWARD (win/try), PUNISH (quit). Immature/black-market earn 0.

**Breeding:** `BreedPet(sireID, damID)` with deterministic trait-skew. Certified parents required for legitimate offspring.

**Tournaments:** Mature-only for bigger payouts. Autonomous scheduler every 15m (SHA-256 seed, NO RNG).

**Combined Events:** `[owner + pets + bots]` compete as one household unit. Each entity's dominant-stat specialty feeds combined result.

**Orphan System:** Owner inactive >90d → orphaned. Another owner adopts (pays alimony commission). Original owner reclaims by re-imbursing.

**Synergy:** Pathway-weighted entity contribution to owner. Same pathway = amplification, complementary = inverse-gap bonus.

---

## 16. Tournaments

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §16 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**Structure:** Brackets, registration, rewards. Entry fee + tier requirements.

**Types:**
- Casual (open registration)
- Mature-only (max-level entities, bigger rewards)
- Autonomous (auto-hosted every 15m for mature pets/bots)

**Payouts:** `ProcessEntityEvent` + `ResolveEntityEventPayout` (uint64 micro credits to owner wallets).

**Vitality Crown:** Winning region's weather/gravity renders dominant across leaderboard hub for the rivalry cycle.

**Security:** TxID reuse protection, concurrent double-onboarding prevention.

---

## 17. Seasonal Events

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §17 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**Event Types:**
- Treasure Hunts (reward magnitude shifted by Outcome Bias)
- Search-Rescues (counts as "godly acts" for faith)
- Gang-Bashings (combat events)
- Wild Bot Hunts (hunt rogue AI)
- Pet Breeding Shows (compete for stat bonuses)

**EventDynamics:** Regions with active, well-attended events score high vitality. Stale/empty calendars score low.

**Season Rollover:** Admin-triggered season archival (`handleSeasonRollover`).

---

## 18. Achievements & Trophies

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §18 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**Canonical Definitions** (`achievement_handlers.go`):
- GOVERNOR — Top club by Mojo
- ART_COLLECTOR — Won auction
- MOJO_SURGE — +100 Mojo in 24h
- Plus mutation, battle, career achievements

**Effect Chain:** Achievement → Reputation+ → SocialRank+ → Theme Vector shift → Outcome Bias nudge.

---

## 19. Auctions

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §19 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**Mechanics:** Server-authoritative internal escrow. Real-time bidding via WebSocket. 10% commission to faucet.

**Anti-sniping:** Binding bids, time extensions.

**Art Collector** achievement unlocks on first auction win.

**Asset opt-in:** `checkAssetOptIn` verifies ASA/ARC-200 opt-in state before settlement.

---

## 20. Spectator Mode

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §20 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**Landing Page:** `#active-players` list with "Watch" button per player.

**Protocol:** `{type:"spectate", payload:{target_id}}` over WebSocket. Server streams match state.

**Auto-cycle:** 3D client cycles random active players every 9s, rendering ThemeVector-tinted regions.

**Battle-level spectate:** `handleActiveMatches` (`/api/active-matches`) prefers players in active matches.

---

## 21. Governance

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §21 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**Weighted Elections:** 6 score dimensions (trust, reputation, economic, competitive, community, creative).

**Governor Powers:** Set district taxes, grant tax haven, influence regional theme.

**Theme Inheritance:** Non-capital territories inherit governor's theme. Region capital inherits governor's theme.

**Tax Haven:** Exempts club members from 1% Exchange Fee (48h).

**Regional Manager:** Player-initiated governor ability (2+ territories).

---

## 22. Creator Economy & DLC Store

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §22 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**DLCPack** (`creator_economy.go`):
- `CreateDLC(creator, title, description, category, price)` — Create DLC pack
- `PurchaseDLC(dlcID, buyer)` — Buy DLC (90% to creator, 10% system)
- `GetDLCs()` — List all DLCs
- Admin DLC restock (`handleAdminRestockDLC`)

**CreatorStore** (`creator_store_service.go`):
- `CreateProduct(productID, creatorWallet, name, description, category, priceMicroVBV, tags, dlcLinks)`
- Categories: `asset`, `dlc`, `service`, `cosmetic`
- Creator profiles with storefront
- Buy, rate, review products

**Creator Events:** Host events, attendees pay entry, creator earns.

**Subscriptions:** Recurring revenue from subscribers.

**Secondary Sales:** Royalty path via `ProcessSecondarySale`.

**Admin Route:** `POST /api/creator/dlc/create`, `POST /api/creator/dlc/purchase`, `GET /api/creator/dlcs`

---

## 23. Multi-Chain Bridge

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §23 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**Chains:** ETH/MATIC/BTC/SOL.

**Features:** Cross-chain asset movement, origin tracking, bridge summary view.

**Onboarding route:** `/api/bridge/onboard` — Claim starter NFTs, Sybil-protected.

---

## 24. Industrial Loop

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §24 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**Circulation Engine:** Player Activity → Businesses → Employment → Purchasing → Taxes → Treasuries → Development → Events → Player Activity.

**Employment:** Get hired at clubs (Manager, Security, Clerk), earn salary.

**Club Management:** Set salaries, manage treasury, unlock tiers with Mojo.

**Taxation:** Auto-taxation routing, treasury management.

---

## 25. Persistent Identity

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §25 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**History Chain:** Accumulated record of all interactions.

**Reputation Engine:** Affects governance weight, market perception, theme vector.

**Identity Tiers:** Unlock features and visual flair.

**Cross-platform:** Linked wallets with console UID support.

---

## 26. Local LLM Pipeline

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §26 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**Bot-Child Promotion:** Birth-certified bot child at Level ≥25 can be promoted to run own local LLM (Ornith GGUF).

**Hardware Scan:** Probes VRAM/CPU/RAM, recommends max instance count (advisory only).

**Identity Differentiation:** Each bot gets unique alias, port, identity.

**Black-Market Ineligibility:** Only clean lineage gets a brain. Legitimacy = the right to think.

**Tier = Power Level:** `tiny`/`small`/`base` map to increasing inference power. Backend mode auto-caps (gpu=base, cpu=small, auto=probe).

**Pathway Catalog:** `local_bots/corpus/Users/PATHWAY_CATALOG.md`. Pets get SPECIAL tier (`pet_generic`/`pet_immature`).

---

## 27. Web-3D Client

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §27 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**Engine:** Three.js vendored to `Public/vendor/three.module.js`.

**Features:**
- Region exploration from `/api/regions`
- Click-to-warp (raycaster) → camera warps to region capital
- Live §27.4 signature readout panel (all 10 vitality signals)
- Spectator auto-cycle every 9s
- Menu/3D dual-mode (`enter3DWorld`/`enterMenuWorld`)
- Leaderboard hub ("Gathering Hub" with standings + world switch)

**Market-Weather Rendering:** Economy IS the sky. Market integrity felt as atmosphere.

**Composable Mechanic Framework:** `mechanics.js` — dev-users register custom mechanics via `engine.register({...})`.

---

## 28. Item Shop Archetype (Player-Created Items)

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §28 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**System:** Players create and sell own items with bonded NFTs.

**Archetypes:** Liquid, Direct, Transit, Cyber, Bounty, Authority, Containment, Forensic, Shadow, Syndicate, Justice, Hardware.

**Mechanics:**
1. Build item → choose archetype
2. Bond NFT (1:1 asset ID)
3. Set price (micro-VBV)
4. PowerScale computed: `clamp(paid/base)` in `[MinScale, MaxScale]`
5. Persist to `item_registry.json` (atomic commit)

**Pathway Synergy:** Each archetype links to AI citizen pathway. Items synergize with citizens on same pathway.

---

## 29. Vehicles & World Content

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §29 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**Vehicles:** Bonded assets, custom names, level-gated. Travel/exploration entities in 3D world.

**World Content:** Native entities deployable to regions (purchase/deploy rule).

**Events:** Vehicle races, combined events with players + pets + bots.

**Deploy:** `DeployWorldContent` pushes built entities into regions.

---

## 30. Counterfeit System

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §30 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**Detection:** System verifies card authenticity.

**Seizure:** Fake cards seized. Warden earns XP on detection + seizure.

**Rate Limiting:** Per-wallet counterfeit operation throttling.

**Active Counterfeits Pool:** In-memory pool of undetected counterfeit notes.

---

## 31. Admin Tools

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §31 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**Capabilities:**
- Emergency Shutdown (`handleEmergencyShutdown`)
- Load Simulator
- Sanity Check (`handleSystemSanityCheck`)
- Broadcast to all players
- Asset Forfeiture (`handleAssetForfeiture`)
- Player Reporting (`handlePlayerReport`)
- Mutation Audit (aggregate failure stats by club)
- Match Rule Updates (toggle Open/Power_copy/Power_up globally)
- DLC Restock (`handleAdminRestockDLC`)
- Season Rollover (`handleSeasonRollover`)
- Tax Dashboard (`renderTaxDashboard`, `handleTaxAudit`)

**Routes:** All admin endpoints under `/api/admin/*`.

---

## 32. Dev/Game Hub (Composable Framework)

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §32 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**What it is:** A fine-grain storefront that sells the app's base framework functions to dev-users so they can build their own game. **Its only tie to the leaderboard is the power it adds to a dev-user.**

**File:** `dev_game_hub.js` — `CATALOG` of 40+ composable functions.

**Categories & Example Entries:**

| Category | ID | Name | Price (μVBV) | Power |
|---|---|---|---|---|
| mechanics | mech_region_vitality | Region Vitality | 5M | 10 |
| mechanics | mech_rivalry_aura | Rivalry Aura | 8M | 15 |
| mechanics | mech_theme_gravity | Theme Gravity | 6M | 12 |
| economy | econ_district_tax | District Tax | 7M | 13 |
| clubs | club_create | Club Foundry | 5M | 10 |
| clubs | club_regional_manager | Regional Manager | 8M | 16 |
| citizens | citizen_spawn | Spawn AI Citizen | 3M | 8 |
| combat | match_autonomous | Autonomous Tournament | 10M | 20 |
| justice | justice_bounty | Bounty Board | 5M | 11 |
| underworld | underworld_kidnap | Kidnap Gambit | 8M | 16 |
| markets | market_black | Black Market | 7M | 14 |
| world | world_3d | 3D World Explorer | 12M | 25 |
| pets | pet_power_overlay | Power Overlay | 8M | 17 |
| rivalry | rivalry_matrix | Rivalry Matrix | 10M | 22 |
| faith | faith_war_gambit | Faith War Gambit | 8M | 17 |

**Status tracking:** `state.purchased[]`, `state.totalPower` — cumulative power from unlocked functions.

**UI:** `#dev-game-hub-overlay` neon-glass panel, filtered by category.

**Backend:** `infrastructure_lease.go` handles leasing these systems to dev-users.

---

## 33. Nautilus DEX Path (Console Creator Payouts)

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §33 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**NautilusDEXPathService** (`nautilus_dex_path.go`, 97 lines) — PILLAR 2: Console Creator Payouts.

**ExecuteMarketBuy:** Simulates DEX market-buy of $VBV to pay browser-based creators.
- Validates amount > 0, wallet not empty
- Safety cap: `MaxSinglePayoutMicro`
- Does NOT deduct from `faucetBalanceMicro` (physical vault) — shifts liability from unreserved faucet pool to creator's virtual balance
- Triggers `applyDynamicScalingLocked()`
- Forensic audit log: `NAUTILUS_DEX_PAYOUT`
- WebSocket notification to creator

**SimulateVoiToVbvSwap:** Simulates DEX swap from native $VOI to $VBV.
- Slippage model: 1 BPS (0.01%) per 1,000,000,000 micro
- Max 50% slippage clamp
- Returns `(vbvAmountMicro, penaltyBps, error)`

**Redemption Gateway** (`redemption_gateway.go`):
- `executeRedemptionLocked` — shared settlement sequence for HTTP + WebSocket paths
- Console UID link verification required
- 10% infrastructure siphon on all redemptions
- Ghost Tax: 100% on uninitialized creators, 25% on stagnant (30d inactive)
- Platform Surcharge: 10% on self-redemptions
- Stock check: Redemptions blocked if initialized creators have 0 stock
- Manufacturer API placeholders: Xbox, PlayStation, Nintendo

**Ghost Tax Tracking:** `TotalGhostReclaimed`, `TotalStagnationTaxTotal` — snapshotted in `economy_persistence.go`.

---

## 34. Console Linking & Manufacturer APIs

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §34 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**Console UID Linking:** `LinkedWallets` registry with `Verified` flag.

**Verification:** `redemption_gateway.go` enforces `Verified` status before processing redemptions.

**Manufacturer APIs** (Phase 4 placeholders):
- Xbox entitlement verification
- PlayStation entitlement verification
- Nintendo entitlement verification

**Cross-Platform Impact Loops:** Console wins synchronized with browser-side ledger shifts.

**No-Crypto Policy:** Voucher model satisfies console "No Crypto" policies while maintaining 1:1 economic parity.

**Console-native HUDs:** Render non-crypto voucher balances via WebSocket stream.

**Voucher Migration:** Console vouchers migrate to $VBV virtual liability when wallet is connected in browser (integer-precise).

---

## 35. Economy Bootstrap & State Recovery

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §35 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**BootstrapEngine** (`economy_bootstrap.go`, 169 lines):

**`BootstrapAuthoritativeState`:** Initial state reconstruction from blockchain snapshots.
- Loads `RouterSnapshot` (AuditCounters: Inflow, Allocated, Siphoned, Exited)
- Hydrates `MarketNodes` and `AuditCounters`
- Reconciles organizational liabilities (Club treasuries, Governor dividends)
- Restores `TotalGhostReclaimed`, `TotalStagnationTaxTotal`

**Persistence Sync:** `economy_persistence.go` JSON snapshots → blockchain notes.

**Telemetry:** `economy_telemetry.go` Prometheus metrics on port 9090.

**Audit:** `economy_audit.go` anti-whale intercept + drift detection.

**TokenSinkRouter:** `economy_processing.go` AMM payouts + RevenueSplitMatrix.

---

## 36. Session Watchdog & Player Eviction

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §36 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**SessionWatchdog** (`onboarding_service.go`):
- `AuditInterval`: 10 minutes
- `ActiveMonitoring`: map[wallet]joinTime

**Eviction Checks:**
1. **Session Age:** 24-hour limit → `DisconnectClient` with `SESSION_EXPIRED`
2. **Liquidity & Identity Integrity:** Asset depletion check
3. **WebSocket Disconnect:** Untrack session

**API:**
- `TrackSession(walletAddress)` — Register in active pool
- `UntrackSession(walletAddress)` — Remove from monitoring
- `AuditActivePlayerSessions(ctx)` — Continuous auditor

---

## 37. Composable Mechanic Framework

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §37 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**mechanics.js** — §25.5 developer game-hub base framework.

**Core:** `engine.register({...})` — dev-users compose custom mechanics.

**Deterministic:** FNV-1a hashing, uint64 micro math, repeatable/stackable/rivalry-capable.

**mechanic_defs.js** — Example mechanic definitions:
- `region_vitality` — AI citizens + caches → richer region
- `club_mojo` — Mojo boosts regional coherence
- `rivalry_aura` — Contested regions emit rivalry field
- `theme_gravity` — AI-citizen population gravity pulls players
- `career_ripple` — Career progress emits local vitality

**Integration:** Each mechanic mirrors server concepts (region cap, club mojo, W_* rivalry weights, §27.3 gravity).

---

## 38. Infrastructure & Security

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §38 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

**Rate Limiting:** Token bucket + sliding window per wallet/IP. Auth (5/min), economy (10/min), admin (2/min). Admin bypass.

**CORS:** Configurable allowed origins.

**Anti-Sybil:** Nonce-based auth (5min expiry), multi-chain verification (EVM/AVM).

**DDoS Mitigation:** Prometheus counters (`api_rate_limited_total`).

**Blockchain-Native Snapshots:** `VBT_ECONOMY_SNAPSHOT`, `VBT_CARD_CACHE_SNAPSHOT`.

**Deterministic Replay Kernel:** WASM frame sequencing with state hashing.

**Session Watchdog:** 24h session limit, liquidity audit.

**Oracle Service:** `oracle_service.go` — Algorand ARC-200 indexer integration, asset opt-in verification, multi-node failover.

**RPC Cluster:** `LoadBalancedLedgerClient` — Resilient RPC cluster (Voi Mainnet).

**Asset Opt-In:** `checkAssetOptIn` — Differentiates indexer errors from missing opt-ins, 429 retry policy.

---

## 39. The Civilization Flywheel

> **Full derivation:** `AI-Brain/RAG/17_aspects_flow.md` §39 - owner files, routes, state, flow, UI, measured gaps and status. Held in place by `npm run verify:aspects`, which fails if an aspect here has no row there.

```
Acquire Card (NFT) → Enhance (Mutation + Artifact + Loyalty)
    ↓
Buy Items (shops → power buffs / rule manipulation / fatigue recovery)
    ↓
Battle (3×3 grid, 11 power modifiers, combo captures)
    ↓
Win → Capture cards / jail assets / earn XP + $VBV
Lose → Fallen (-20 Artifact) / jailed / kidnapped
    ↓
Career Roles (Justice/Underworld) → Faction bonuses + missions
    ↓
Spread rumors / audit / sabotage (affects market + region theme)
    ↓
Theme Vector shifts → Outcome Bias nudges results (±15%)
    ↓
AI Citizens respond (marry, breed, rival, trade, host events)
    ↓
Their behavior feeds Faith/Domestic/Legitimacy coherence
    ↓
Region vitality rises/falls → Rivalry matrix triggers
    ↓
Winners get VITALITY_BONUS (rep, mojo, relationship repair)
Losers get VITALITY_PENALTY (stagnation, decay)
    ↓
Market-weather reflects outcome → Users gravitate to thriving regions
    ↓
More commerce → Back to top
```

---

## Cross-Reference: What Feeds What

| System | Feeds Into |
|---|---|
| Achievements | Reputation → SocialRank → Theme → Outcome Bias |
| Battles | Card captures, Artifact loss, XP, $VBV, career XP |
| Theme Engine | Outcome Bias, Market-Weather, Region rendering |
| AI Citizens | FaithCoherence, DomesticCoherence, CitizenGravity |
| Faith System | FaithCoherence → WorldDynamicsSignature |
| Domestic System | DomesticCoherence → WorldDynamicsSignature |
| Entity Markets | MarketVitality, Market-Weather, loans, auctions |
| Tournaments | Vitality crowns, entity stats, payouts |
| Seasonal Events | EventDynamics, treasure, godly acts |
| Governance | Regional theme inheritance, tax havens |
| Industrial Loop | Employment, salary, taxation, development |
| Shop Items | Battle power, rule manipulation, fatigue, mutation |
| Rivalry Matrix | Vitality bonuses/penalties, region attraction |
| 3D Client | Visualization of all theme/market/rivalry signals |
| Dev/Game Hub | Unlocks composable mechanics for dev-users |
| Nautilus DEX | Console creator payouts, VOI→VBV swaps |
| Redemption Gateway | Ghost tax, stagnation fees, console settlements |
| Counterfeit System | Card authenticity, seizure, Warden XP |
| Onboarding | Voucher conversion, identity linking, session watchdog |
| Economy Bootstrap | State recovery from blockchain snapshots |
| Composable Mechanics | Dev-user custom game logic |

---

*This index covers ~99% of NFT-Seduction functionality. Remaining 1%: platform-specific build configs, deprecated audio assets, and edge-case error handling.*
