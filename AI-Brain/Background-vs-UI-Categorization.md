# NFT-Seduction: Background vs UI-Interactive Categorization

> **Purpose:** Classify every system by runtime behavior — what runs autonomously vs. what requires active player engagement vs. what is hybrid.
> **Source:** App-Aspect-Index.md (39 sections)
> **Last updated:** 2026-09-07

---

## How to Read This Document

| Category | Meaning |
|---|---|
| 🟢 **Background** | Runs autonomously on server tick loops, timers, or event triggers. No user input needed. Consumes server resources continuously. |
| 🔵 **UI-Interactive** | Requires the user to actively click, navigate, or input data. Driven by user sessions. Real-time UI updates expected. |
| 🟡 **Both** | Has significant autonomous background processing AND direct user interaction paths. Usually the background part feeds data the user sees/acts on. |

---

## 🟢 Background Systems (24)

These run 24/7 on server tick loops, timers, or event-driven pipelines. No player needs to be logged in.

### Core Engine
| System | What It Does Autonomously | Tick/Timer |
|---|---|---|
| Theme Engine | Computes Theme Vectors, Outcome Bias, World-Dynamics Signature from live data | Continuous |
| Industrial Loop | Circulates value: Activity → Business → Employment → Purchasing → Taxes → Development → Events | 15-min economy tick |
| Economy Bootstrap | Reconstructs authoritative state from blockchain snapshots on boot | On startup |
| Civilization Flywheel | Emergent loop of all systems feeding each other | Continuous |

### AI & NPC Systems
| System | What It Does Autonomously | Tick/Timer |
|---|---|---|
| AI Citizens | BehavioralTick loop: faith rituals, marriage, breeding, pet adoption, justice enforcement, employment, rivalry, market activity, career progression, event hosting | Per-citizen tick (probability-gated) |
| Faith System | Religious leader power effects, faith war gambits, coherence decay/growth | Per-tick + event-driven |
| Domestic System | BondGraph coherence decay, child aging, pet stat drift | Per-tick |
| Local LLM Pipeline | Bot-child inference (if promoted), hardware monitoring | Continuous (if active) |

### Market & Economy
| System | What It Does Autonomously | Tick/Timer |
|---|---|---|
| Entity Markets | AMM bonding curve pricing, market-weather rendering, disaster triggers, loan interest accrual | Real-time (reserve shifts) |
| Nautilus DEX Path | Console creator payout processing, VOI→VBV swap simulation, dynamic scaling | Event-driven |
| Counterfeit System | Detection sweeps, seizure processing, rate limiting | Periodic scan |

### Events & Tournaments
| System | What It Does Autonomously | Tick/Timer |
|---|---|---|
| Seasonal Events | Event lifecycle: spawn → active → resolve → reward distribution | Season timer |
| Tournaments | Autonomous scheduler hosts brackets every 15m for mature entities | 15-min timer |
| Achievement Tracking | Monitors player stats, auto-awards achievements | Event-driven |

### Governance & Security
| System | What It Does Autonomously | Tick/Timer |
|---|---|---|
| Governance | Weighted election scoring, theme inheritance propagation, tax haven expiry | Election cycle |
| Session Watchdog | 24h session limit enforcement, liquidity audit, eviction | 10-min audit tick |
| Infrastructure & Security | Rate limiting, anti-sybil verification, DDoS mitigation, Prometheus counters | Continuous |

### Persistence & Recovery
| System | What It Does Autonomously | Tick/Timer |
|---|---|---|
| Economy Persistence | Gzip-backed atomic commits, .tmp rolling backups, 15-min snapshots | 15-min timer |
| Asset Opt-In | ARC-200 indexer polling, multi-node failover | On-demand + periodic |

### World Rendering
| System | What It Does Autonomously | Tick/Timer |
|---|---|---|
| Web-3D Client | Spectator auto-cycle (9s), market-weather rendering, region vitality glow | Client-side loop |
| Composable Mechanics | Dev-registered custom logic runs deterministically | Event-driven |

---

## 🔵 UI-Interactive Systems (7)

These require the user to be actively engaged. They don't do anything meaningful without player input.

### Player Onboarding & Identity
| System | User Interaction | Input Required |
|---|---|---|
| Onboarding | Voi wallet provisioning, voucher conversion, identity linking | Wallet signature + voucher code |
| Console Linking | Link console UID to browser wallet | Console ID + verification |
| Multi-Chain Bridge | Cross-chain asset movement | Wallet signature on both chains |

### Direct Player Actions
| System | User Interaction | Input Required |
|---|---|---|
| Battle System | Play 3×3 grid battles, place cards, capture enemies | Click-to-place, rule selection |
| Card Enhancement | Mutate cards, enhance artifacts, boost loyalty, reduce fatigue | Click + $VBV spend |
| Shop & Item Economy | Buy/sell items from player shops | Click-to-purchase |
| DLC Store | Create DLC packs, purchase from creators | File upload + $VBV spend |
| Creator Store | Create products, buy, rate, review | Form input + $VBV spend |
| Dev/Game Hub | Browse composable mechanics catalog, purchase functions | Click-to-buy |

### Quick Entry
| System | User Interaction | Input Required |
|---|---|---|
| QuickPlay | Enter fast-match queue from menu | Single click |

---

## 🟡 Background + UI-Interactive (8)

These are hybrid — they have significant autonomous processing AND direct user interaction paths.

### Matchmaking
- **Background:** Queue pooling, AI citizen injection into human matches, tournament-lock filtering
- **UI:** User enters queue, sees match found, accepts/declines
- **Both needed:** Background fills empty slots with AI; UI lets human players compete

### Factions & Careers
- **Background:** Career XP auto-tracks, faction bonuses apply to battles, role-gated access checks
- **UI:** User selects career path, accepts roles, views career tree
- **Both needed:** Background enforces role gates; UI lets user choose direction

### Justice System
- **Background:** Wanted Level decay, bounty board updates, mission expiry, power bonus calculation
- **UI:** User becomes Warden, claims bounty, generates missions, uses Truth Serum
- **Both needed:** Background tracks criminality; UI lets justice players act

### Underworld & Criminality
- **Background:** Wanted Level escalation, kidnap ransom economy, heist success calculation, Wanted decay
- **UI:** User accepts contracts, commits kidnaps, launders money, buys black market goods
- **Both needed:** Background processes crime consequences; UI lets criminals operate

### Auctions
- **Background:** Escrow holding, anti-sniping time extension, 10% commission routing, asset opt-in verification
- **UI:** User places bids, watches countdown, wins/loses
- **Both needed:** Background secures funds; UI provides bidding interface

### Spectator Mode
- **Background:** Auto-cycle through active players, streams match state over WebSocket
- **UI:** User clicks "Watch", selects target, views live battle
- **Both needed:** Background curates available spectate targets; UI lets user choose

### Governance
- **Background:** 6-dimension election scoring, theme inheritance, tax haven expiry, regional manager activation
- **UI:** User votes, runs for governor, sets district taxes, grants tax havens
- **Both needed:** Background computes election results; UI provides voting interface

### Vehicles & World Content
- **Background:** World content deployable to regions, vehicle bonding validation
- **UI:** User purchases vehicles, deploys world content, races
- **Both needed:** Background validates ownership/deployment; UI provides purchase interface

---

## Summary Table

| Category | Count | Server Load | Player Engagement |
|---|---|---|---|
| 🟢 Background | 24 | High (continuous) | None needed |
| 🔵 UI-Interactive | 7 | Low (on-demand) | Required |
| 🟡 Both | 8 | Medium (event-driven) | Optional but rewarded |

---

## Key Architectural Insight

**The 24 background systems ARE the game when no one is logged in.** Players return to a world that has evolved without them:
- AI citizens married, bred, and advanced careers
- Markets shifted and rendered new weather
- Tournaments completed with new vitality crowns
- Faith coherence rose or fell based on ritual activity
- Theme vectors drifted based on collective player behavior

**The 7 UI-interactive systems are the levers** players pull to inject their agency into that living world.

**The 8 hybrid systems are the bridges** — they let player actions feed the autonomous world, and the autonomous world surfaces back to the player through real-time UI updates.

This is why the **Split-WASM architecture** matters: the 24 background systems run on the authoritative server (`server-bin.exe`), while the 8 hybrid systems' UI components run in the WASM client (`main.go`) for instant feedback, with server validation preventing cheating.

---

*Next: We can now design the UI flow knowing exactly which screens need real-time WebSocket feeds (hybrid systems) vs. which can be request-response (UI-interactive) vs. which are read-only dashboards (background).*
