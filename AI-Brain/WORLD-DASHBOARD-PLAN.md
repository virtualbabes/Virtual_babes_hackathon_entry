# World Dashboard — Complete Tab & Feature Plan

> **Purpose:** Define every user-facing feature in the game, organized by unlock order (player journey).  
> **Each feature** = a sub-function in a World Dashboard tab, starable to Constellation Hub.  
> **Source:** Grepped all `.go` handlers, JS panels, and asset registry.

---

## Player Journey (Unlock Order)

### Phase 0 — Onboarding (Wallet Connect → First Login)
| Tab | Feature | API | Description |
|-----|---------|-----|-------------|
| `onboarding` | Wallet Link | `/api/identity/link` | Connect wallet, link cross-chain |
| `onboarding` | Starter Pack | `/api/bridge/onboard` | Claim starter NFTs, Sybil-protected |
| `onboarding` | Tutorial | *(video player)* | Tutorial videos, game intro |
| `onboarding` | Avatar Setup | `register_avatar` WS | Select NFT avatar, set gloat |

### Phase 1 — Core Loop (Battles & Earnings)
| Tab | Feature | API | Description |
|-----|---------|-----|-------------|
| `battle` | Quick Play | `join_queue` WS | Matchmaking pool, ranked match |
| `battle` | Deck Manager | `GetGameState` WASM | Build/edit 5-card deck |
| `battle` | Battle History | `/api/replay/latest` | Replays, frame-by-frame |
| `battle` | Spectate | `sendSpectate` WS | Watch live matches |
| `rewards` | Faucet Claim | `/api/reward` | Claim $VBV winnings |
| `rewards` | Vault Status | `/api/faucet/status` | Faucet balance, dynamic scaling |
| `rewards` | Rewards Config | `/api/reward/add`, `/api/reward/remove` | Admin: configure reward assets |

### Phase 2 — Economy (Earn & Trade)
| Tab | Feature | API | Description |
|-----|---------|-----|-------------|
| `markets` | Entity Market | `/api/entity/market/list` | Buy/sell NPC/player shares |
| `markets` | Auctions | `/api/auctions` | Art Gallery, card bundle bidding |
| `markets` | Black Market | `/api/black-market/buy` | Stolen/g liquidated assets |
| `markets` | Entity Shares | `/api/shares/issue`, `/api/shares/buy` | Issue & trade share tokens |
| `loans` | Loan List | `/api/loans` | Active loans, collateral |
| `loans` | Take Loan | `/api/loans/take` | Borrow $VBV against NFT collateral |
| `loans` | Repay | `/api/loans/repay` | Return loan + interest |
| `investments` | Portfolio | `/api/invest/portfolio` | Entity investments, dividends |
| `investments` | Claim Dividends | `/api/claim/dividends` | Collect investment yield |

### Phase 3 — Social & Status
| Tab | Feature | API | Description |
|-----|---------|-----|-------------|
| `leaderboard` | Global Rank | `/api/leaderboard` | Top players by reputation |
| `leaderboard` | Season History | `/api/season/history` | Past seasons, champions |
| `leaderboard` | Hall of Fame | `window.ToggleLeaderboard()` | Tournament archives |
| `identity` | Profile | `/api/identity/profile` | Reputation, Mojo, Social Rank |
| `identity` | Achievements | `/api/achievements` | Trophy badges, unlock progress |
| `identity` | Stats Overlay | `/api/stat-overlay` | Region stats, owner stats |
| `identity` | Compliance | `/api/compliance/records` | KYC/AML audit records |
| `career` | Career Progress | `/api/career/progress` | Current path, XP, level |
| `career` | Employment | `handleHirePlayer` WS | Get hired by clubs |
| `career` | Salary | `handleSetSalary` WS | Daily salary dispenser |

### Phase 4 — Clubs & Territory (Guild System)
| Tab | Feature | API | Description |
|-----|---------|-----|-------------|
| `clubs` | Club List | `/api/clubs` (WS state) | Active clubs, members |
| `clubs` | Found Club | `handleCreateClub` WS | Create new club (5000 VBV) |
| `clubs` | Join Club | `handleJoinClub` WS | Apply to join |
| `clubs` | Club Management | `handleRestockInventory` WS | Manage staff, inventory |
| `territory` | World Map | `openTerritoryMapOverlay()` | 3D tactical district view |
| `territory` | Purchase District | `handlePurchaseTerritory` WS | Buy territory (2500 VBV) |
| `territory` | Regional Governor | *(auto at 2+ districts)* | Tax districts, +5% power |
| `territory` | Alliances | `sendAllianceInvite` WS | Defensive coalitions |

### Phase 5 — Justice & Crime (PvP Reputation)
| Tab | Feature | API | Description |
|-----|---------|-----|-------------|
| `justice` | Bounty Board | `/api/justice/bounty-board` | High-Wanted targets |
| `justice` | Capture Bounty | `/api/justice/capture-bounty` | Jail wanted players |
| `justice` | Justice Dashboard | `/api/justice/dashboard` | Warden+ access, mission gen |
| `justice` | Truth Serum | `/api/justice/use-truth-serum` | Reveal opponent buffs |
| `justice` | Rep Shield | `/api/justice/use-rep-shield` | Shield reputation loss |
| `justice` | Award Card | `/api/justice/award-card` | Justice card redemption |
| `underworld` | Contracts | `/api/contracts/list` | Underworld missions |
| `underworld` | Assign Contract | `/api/contracts/assign` | Take a heist/kidnap contract |
| `underworld` | Counterfeit Detect | `/api/counterfeit/detect` | Identify fakes |
| `underworld` | Counterfeit Generate | `/api/counterfeit/generate` | Create fakes (criminal) |
| `underworld` | Cyber Intercept | `/api/criminality/cyber-intercept` | Hack rival clubs |
| `underworld` | Kidnap Gambit | `handleKidnapRequest` WS | Hostage mechanics |
| `underworld` | Ransom | `handlePayRansom` WS | Pay/receive ransom |
| `underworld` | Fence Goods | `/api/black-market/fence-goods` | Sell stolen items |
| `underworld` | Sell Tokens | `/api/black-market/sell-tokens` | Liquidate market tokens |

### Phase 6 — Civilization (Pets, Vehicles, World)
| Tab | Feature | API | Description |
|-----|---------|-----|-------------|
| `pets` | Pet List | `/api/pets` | Owned companions |
| `pets` | Spawn Pet | `/api/pets/spawn` | Adopt new pet |
| `pets` | Breed Pet | `/api/pets/breed` | Trait-skew breeding |
| `pets` | Pet Battle | `/api/pet-battle/challenge` | Pet PvP arena |
| `vehicles` | Vehicle List | `/api/vehicles` | Owned vehicles |
| `vehicles` | Spawn Vehicle | `/api/vehicles/spawn` | Create vehicle |
| `world` | World Content | `/api/world-content` | NFT-placed world items |
| `world` | Create Content | `/api/world-content/create` | Design world item |
| `world` | Deploy Content | `/api/world-content/deploy` | Place in 3D world |
| `regions` | Region List | `/api/regions` | Discovered regions |
| `regions` | Region Details | `/api/stat-overlay/region` | Region vitality, alerts |
| `treasure` | Spawn Treasure | `/api/treasure/spawn` | Create treasure cache |
| `treasure` | Claim Treasure | `/api/treasure/claim` | Open discovered cache |

### Phase 7 — AI & Theme (Advanced)
| Tab | Feature | API | Description |
|-----|---------|-----|-------------|
| `ai-citizens` | AI List | `/api/ai/citizens/list` | All AI citizens |
| `ai-citizens` | Spawn AI | `/api/ai/citizens/spawn` | Create own AI citizen |
| `ai-citizens` | Free Agents | `/api/ai/citizens/free-agents` | Job-seeking AIs |
| `ai-citizens` | AI Business | `/api/ai/citizens/business/spawn` | Spawn AI-run shop |
| `ai-citizens` | AI Challenge | `/api/ai/citizens/challenge` | Challenge AI to bout |
| `ai-citizens` | Adopt Orphan | `/api/orphan/adopt` | Adopt orphaned AI child |
| `theme` | Theme Vector | `/api/theme/vector` | Tone, element, intensity |
| `theme` | Theme Lock | `/api/theme/lock` | Lock asset to slot |
| `theme` | Theme Bind | `/api/theme/bind` | Bind asset as cosmetic |
| `theme` | World Dynamics | `/api/rivalry/world-dynamics` | 10-signal region vitality |
| `theme` | Market Weather | `/api/market/weather` | Sentiment climate |
| `faith` | Faith Coherence | `/api/faith/coherence` | Regional faith score |
| `faith` | Faith War Gambit | `/api/faith/war-gambit` | Stake card in faith battle |
| `faith` | Religions | `/api/faith/religions` | List faith systems |
| `faith` | Church Storefront | `/api/church/*` | Church items, rituals |

### Phase 8 — Creator & Launchpad (Meta)
| Tab | Feature | API | Description |
|-----|---------|-----|-------------|
| `creator` | Creator Store | `/api/creator/store/products` | Player-owned DLC shops |
| `creator` | DLC Create | `/api/creator/dlc/create` | Create DLC package |
| `creator` | DLC Purchase | `/api/creator/dlc/purchase` | Buy player DLC |
| `creator` | Creator Events | `/api/creator/events` | Host in-game events |
| `creator` | Subscriptions | `/api/creator/subs` | Recurring patronage |
| `creator` | Royalties | `/api/creator/royalties` | Creator earnings |
| `launchpad` | Launch List | `/api/launches` | Active launch projects |
| `launchpad` | Back Launch | `/api/launch/back` | Fund a creator project |
| `launchpad` | Activate | `/api/launch/activate` | Go live with funding |
| `launchpad` | Integrate | `/api/launch/integrate` | Merge into ecosystem |

### Phase 9 — Infrastructure & Meta
| Tab | Feature | API | Description |
|-----|---------|-----|-------------|
| `infrastructure` | OS Modules | `/api/os/modules` | Gaming OS modules |
| `infrastructure` | OS Lease | `/api/os/lease` | Lease infrastructure |
| `infrastructure` | Lease Create | `/api/lease/create` | New infrastructure lease |
| `infrastructure` | Industrial Loop | `/api/industrial-loop/metrics` | Economic circulation health |
| `gamingos` | Gaming OS Summary | `/api/os/summary` | Civilization-as-a-service |
| `advertising` | Ad Create | `/api/ad/create` | Buy ad campaigns |
| `advertising` | Ad Stats | `/api/ad/stats` | Impressions, clicks |
| `governance` | Gov Weight | `/api/governance/weight` | Voting power |
| `governance` | Register | `/api/governance/register` | Register as voter |
| `governance` | Vote | `/api/governance/vote` | Cast vote |
| `governance` | Election | `/api/governance/election` | Governor elections |
| `compliance` | Records | `/api/compliance/records` | KYC/AML audit trail |
| `compliance` | Resolve | `/api/compliance/resolve` | Mark issue resolved |
| `compliance` | Escalate | `/api/compliance/escalate` | Escalate to admin |
| `religion-gov` | Religion Buy | `/api/faith/religion/buy` | Purchase religion slot |
| `religion-gov` | Ritual | `/api/faith/religion/ritual` | Perform faith ritual |

### Phase 10 — Tournaments & Spectate
| Tab | Feature | API | Description |
|-----|---------|-----|-------------|
| `tournament` | Register | `/api/tournament/register` | Enter bracket event |
| `tournament` | History | `/api/tournament/history` | Past brackets |
| `tournament` | Bracket View | `openTournamentBracket()` | Live bracket viz |
| `tournament` | Spectate | `/api/match/wager` | Bet on matches |
| `spectate` | Live Spectate | `sendSpectate` WS | Watch in-progress |
| `spectate` | Watch Feed | `watch-feed.html` | External portal |

---

## Total Counts

| Category | Sub-features |
|----------|--------------|
| Onboarding | 4 |
| Battle & Rewards | 7 |
| Economy & Markets | 10 |
| Social & Status | 10 |
| Clubs & Territory | 7 |
| Justice & Crime | 15 |
| Civilization (Pets/Vehicles/World) | 13 |
| AI & Theme | 15 |
| Creator & Launchpad | 10 |
| Infrastructure & Meta | 13 |
| Tournaments | 5 |
| **TOTAL** | **~109 starable features** |

---

## Notes

1. **All tabs stay unlocked for development** — no gating logic yet. The unlock order is the *design intent* for future progression gating.
2. **Each sub-feature** has a ☆ star button. Clicking adds it to the Constellation Hub with the tab name as context.
3. **Bound assets** (NFT theming) attach at the button level via `UserPreferences.bindAsset(tab, sub, assetId)`.
4. **World Dashboard** is the single source of truth for all game functions.
5. **Constellation Hub** is the user-curated shortcut layer on top.

---

## Implementation Status

- [x] Data model (`user_preferences.js`)
- [x] World Dashboard (skeleton with Career tab fully wired)
- [x] Constellation Hub (grid with animations)
- [ ] All remaining tabs wired (95 features remaining)
- [ ] Bound asset metadata endpoint
- [ ] Drag-and-drop reordering in Constellation Hub
- [ ] Import/export UI for preferences
- [ ] Progression gating logic

---

*Last Updated: 2026-09-04 — Plan v1.0*
