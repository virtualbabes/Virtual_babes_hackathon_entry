# RIVALRY MANUAL - NFT-Seduction

> **Version:** 2.0 | **Date:** 2026-09-15 | **Audience:** Players & AI Agents
> **Owner of record: `AI-Brain/Rivalry-Matrix.md`.** This file is a player-facing SUMMARY of it.
>
> **WHAT CHANGED IN v2.0.** Version 1.0 described career types (`Warrior`, `Mage`, `Rogue`,
> `Healer`, `Engineer`, `Breeder`) that exist NOWHERE in the repository, "rivalry factions"
> (Vehicle Builders vs Breeders) that are not a system, and **five API endpoints that are
> registered nowhere**. All of it is replaced below. For the authoritative tables, weights,
> gates and endpoint list, read `AI-Brain/Rivalry-Matrix.md`.

---

## 1. WHAT RIVALRY IS

Rivalry is not one system. There are **THREE matrices**, each owned by its own code, each met in a
different place:

| Matrix | Scores | Where you meet it |
| --- | --- | --- |
| **Path** | your alignment against another player's: `justice` vs `criminal` is the one DIRECT rivalry | the career panel's path matrix; the direct-rival XP bonus |
| **Career** | role vs role: 13 declared pairs of careers | every cross-career interaction (a fight, the courthouse, a contract, a heist) |
| **Region** | territory and region strength, and living-world vitality | `/api/regions`, the rivalry viewer, the 3D world, the leaderboard hub |

A UI always shows WHICH matrix a number came from, because the server serves the stage id with it.

---

## 2. THE TWO SIDES (and the neutral)

There are two rival SIDES, not two "factions":

| Side | Direct rivalry | Careers |
| --- | --- | --- |
| `justice` | vs `criminal` | Bounty Hunter, Forensic Analyst, Tax Auditor, Warden, Sector Peacekeeper, Intel-Agent, Justice Recruiter, AOS Leader, Justice Commissioner, Mutation Log Auditor |
| `criminal` | vs `justice` | Kidnapper, Gossip, Launderer, Heist Planner, Smuggler, Arc-Net Operative, Fence, Saboteur, Judge, Underworld Boss |
| `neutral` | none (interprets) | any of the above, at the cost of the direct-rival bonus |

| You | Them | Relation | Bonus |
| --- | --- | --- | --- |
| justice | criminal | **direct** | +10% |
| criminal | justice | **direct** | +10% |
| justice | justice | shared (ally) | - |
| criminal | criminal | shared (ally) | - |
| neutral | neutral | **interpreted** | - |
| neutral | either side | **interpreted** | - |

**The neutral is a real choice, not a free upgrade.** Neutral lets you promote into EITHER side's
careers; in exchange you hold no direct enemy, so you never earn the direct-rival bonus. You read
the matrices instead of sitting inside one.

The FACTIONS that do exist in this repository are the **faiths** (24 seeded religions) and the
twelve **AI-citizen pathways**: `P-Shadow`, `P-Lockdown`, `P-Ledger`, `P-Syndicate`, `P-Tax`,
`P-Peace`, `P-Intel`, `P-Justice`, `P-AOS`, `P-Commissioner`, `P-Forensic`, `P-Boss`.

---

## 3. YOUR CAREER PATH

### 3.1 The choice - made once

You choose `justice`, `criminal` or `neutral` **once**, when you **open a region while holding at
least 2 territories**. The region step is the engine's own: opening a region already requires two
territories, so the gate cannot disagree with itself. A second choice is refused - an alignment you
can flip on demand carries no weight in any matrix.

### 3.2 Promotion - the level-cap unlock

A career is promoted when ALL of these hold. A refusal always names the ONE you are missing:

| Gate | What it means |
| --- | --- |
| the path | you hold a path, and it allows that career (a neutral may take either side's) |
| the level cap | your career level has reached the career's declared cap |
| the role tier | your XP in that career has reached the declared tier |
| the sustained $VBV | your sustained balance reaches the career's $VBV rung (5,000 up to 2,000,000) |
| the civil rank | you hold the organisation the career demands (see section 4) |

### 3.3 Demotion - when the $VBV drains away

Cashing out takes the career with it, and never without warning:

1. **WARNING** - your sustained $VBV falls below what your career requires. NOTHING is taken. You
   are told the exact shortfall and the deadline (7 days).
2. **CLEARED** - earn it back inside the window and the warning is withdrawn. Your career stands.
3. **DEMOTED** - the window closes with the shortfall still open and the career is removed, with the
   exact arithmetic recorded.

A career that has no declared gate is never judged, so it can never be taken away silently.

---

## 4. THE CIVIL RANK - the second ladder

This ladder is DERIVED from what you own. No client can declare it.

| Rank | How you get it | Why the gate exists |
| --- | --- | --- |
| `user` | no territory | a career that only does the work needs nothing behind it |
| `manager` | hold at least one territory (an alliance's territories count) | a career that directs an organisation's operations needs one |
| `governor` | OPEN A REGION on one of your clubs | a career whose authority IS a region needs a region |

A high-level career needs active user management, which is why the top careers are governor-gated.
This is not new: the pathway constant for the Syndicate/Boss line has always been annotated
"governor-gated" in the code.

---

## 5. CAREER RIVALRIES (the career matrix)

| Pair | Delta | Kind |
| --- | --- | --- |
| BountyHunter - Kidnapper | -15 | enemy |
| ForensicAnalyst - Gossip | -10 | enemy |
| TaxAuditor - Launderer | -10 | enemy |
| Warden - HeistPlanner | -10 | enemy |
| SectorPeacekeeper - Smuggler | -10 | enemy |
| IntelAgent - ArcNetOperative | -10 | enemy |
| MutationLogAuditor - Kidnapper | 0 (declared antagonistic) | enemy |
| JusticeRecruiter - BountyHunter | +8 | ally |
| Launderer - Fence | +5 | ally |
| HeistPlanner - Kidnapper | +12 | ally |
| AOS - SectorPeacekeeper | +6 | ally |
| TaxAuditor - JusticeCommissioner | +7 | ally |
| JusticeRecruiter - MutationLogAuditor | 0 | ally |

A NEGATIVE delta is antagonistic (the attacker gains). A POSITIVE delta is synergistic. Spellings
are compared BY ROLE, so `IntelAgent`, `Int.Agent` and `Intel-Agent` are one career.

---

## 6. REGIONAL RIVALRIES (the region matrix)

A region/territory is scored from the assets deployed in it - **employed players weigh heaviest**,
then AI-citizen pride, then model citizens, vehicles, pets, world content, items and events. Two
regions whose profiles overlap by 40% or more (the collision threshold) auto-declare a rivalry.

Winning one pays a **4-part prize**: a showcase buff, a resident loot cache, governor reputation and
a worker bonus. The winning region's vitality then renders dominant across the leaderboard hub for
the cycle, which attracts more citizens - the loop closes.

**Governor powers that actually exist:** open the region ability (2+ territories, owner only),
collect the rivalry prize, and the region's standing feeds your reputation. There is no
"define a rivalry set" command in the repository - see the endpoint list in section 10.

---

## 7. FAITH RIVALRIES

- 24 religions are seeded on boot, across 9 pathways x 3 dogmas, owned by the faucet until bought.
- A **buyout** costs 10% over the last paid price; all members convert and the previous governor
  receives nothing (the price goes to the faucet).
- **FaithPower** = members x 100 + rituals x 10 + coherence. Rituals add coherence for a fee.
- A **Faith War Gambit** stakes a favourite card; the loser's card is jailed to the winner's kitty.
- Regions carry a FaithCoherence signal that feeds the region matrix.

---

## 8. ENTITIES AND PET BATTLES

Entity power (companions, vehicles, bots) reaches its owner **through the deck cards**: a capped
integer percentage of the household's stat total is added to every card the owner plays. The
percentage is snapshotted at match start, so the server and the engine apply one identical constant.

Arenas (pet bouts, vehicle races and dogfights) are **3D-world destinations**, not a card-game
subsystem. The arena panels are honest placeholders that report the real roster power and point at
the 3D world; nothing is simulated client-side.

---

## 9. STRATEGY

**For a player**
1. Take a territory, then a second, then open your region - that is what unlocks your path choice.
2. Pick your path deliberately: the two sides get a direct-rival bonus, the neutral gets breadth.
3. Work one career up rather than several: each promotion needs a level cap, a tier, a sustained
   balance AND a civil rank.
4. Keep the sustained balance up. Cashing out below your career's rung starts a 7-day clock.

**For a governor**
1. A region's score is mostly PEOPLE - employed players outweigh every asset class.
2. Open your region early; the rank gates the careers that manage others.
3. Publish work: a club with staff and salaries is what makes your region score.

**For an AI citizen**
1. Pathway fit decides which contracts and behaviours you gravitate to.
2. Bonding (marriage, children, pets) raises regional domestic coherence, which raises the score.

---

## 10. API REFERENCE (every route below is registered)

```
GET  /api/career/path                  the whole career system for a wallet
POST /api/career/path/choose           {path}   the one choice
POST /api/career/promote               {role}   the level-cap promotion
GET  /api/career/progress              the legacy progression snapshot

GET  /api/rivalry/world-dynamics       the region world-dynamics signature
GET  /api/rivalry/factions
GET  /api/rivalry/state?wallet=
GET  /api/rivalry/list
POST /api/rivalry/recompute | /detect | /resolve | /request | /action

GET  /api/regions                      region views (a bare list; no per-region path)

GET  /api/pet-battle/list
POST /api/pet-battle/challenge | /resolve

GET  /api/faith/coherence | /religions | /religion/rivalry
POST /api/faith/war-gambit | /religion/buy | /religion/buyout | /religion/join | /religion/ritual
```

---

*Pick a side, hold a rank, keep the balance. The matrices do the rest.*

