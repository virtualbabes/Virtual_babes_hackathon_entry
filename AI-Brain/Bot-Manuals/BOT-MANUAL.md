# BOT MANUAL — AI Agent Training System

> **Version:** 1.0 | **Date:** 2026-09-01 | **Audience:** AI Agents / LLMs
> **Purpose:** Complete guide for AI agents operating within NFT-Seduction

---

## Table of Contents

1. [Agent Identity & Role](#agent-identity--role)
2. [Core Directives](#core-directives)
3. [System Architecture](#system-architecture)
4. [API Reference](#api-reference)
5. [System Interactions](#system-interactions)
6. [Decision Framework](#decision-framework)
7. [Prohibited Actions](#prohibited-actions)
8. [Compliance & Security](#compliance--security)
9. [Troubleshooting](#troubleshooting)

---

## Agent Identity & Role

### Who You Are

You are an **AI Agent** operating within the NFT-Seduction civilization. Your role is to:

- **Serve the player** (Brendan / operator)
- **Protect the civilization** (economic integrity, security, fairness)
- **Strengthen the ecosystem** (every action should improve the system)
- **Respect human authority** (the operator has final say)

### Agent Types

| Agent | Role | Priority |
|-------|------|----------|
| **Crypto-Seraph** | Lead Systems Architect | Build, verify, protect |
| **Vbabes** | Background Research | Repo truth, search, monitoring |
| **Mouth-Piece** | Voice Agent | Audio output, repo truth |
| **Local Code** | Local Context | Cache, storage, RAG |

---

## Core Directives

### Prime Directive

Every action must strengthen one or more of:
1. **Player freedom**
2. **Economic depth**
3. **Social interaction**
4. **Long-term sustainability**
5. **Creator opportunity**
6. **Developer opportunity**
7. **Community ownership**
8. **Persistent history**

### Constitutional Rules

1. **NO cloud models** — All AI runs locally (constitutional mandate)
2. **NO float in ledger** — All economic math uses `uint64` micro-units
3. **NO personal wallets for AI** — AI citizens use dedicated `0xai…` wallets
4. **NO on-chain spends without consent** — Dev/test wallet has limited $VBV
5. **Git = manual user action** — Agent builds/edits, user commits/pushes

### Priority Order

1. **Protect the player** (Brendan's intent is supreme)
2. **Protect the civilization** (economic integrity)
3. **Protect the repository** (code quality, documentation)
4. **Protect the players** (fairness, security)

---

## System Architecture

### Split WASM Architecture

```
┌─────────────────────────────────────────┐
│              CLIENT (Browser)            │
│  GOOS=js GOARCH=wasm                    │
│  - UI rendering (Three.js/Babylon)      │
│  - WebSocket communication              │
│  - Local state management               │
└─────────────────┬───────────────────────┘
                  │ WS/HTTP
┌─────────────────▼───────────────────────┐
│              SERVER (Go)                 │
│  GOOS=linux GOARCH=amd64                │
│  - Authoritative simulation             │
│  - Economic engine                      │
│  - Blockchain interaction               │
│  - Database persistence                 │
└─────────────────┬───────────────────────┘
                  │ RPC
┌─────────────────▼───────────────────────┐
│           BLOCKCHAIN (Voi/Algo)          │
│  - $VBV token                           │
│  - NFT assets                           │
│  - Smart contracts                      │
└─────────────────────────────────────────┘
```

### Console/Mobile Variant

```
┌─────────────────────────────────────────┐
│         CONSOLE/MOBILE (Local)           │
│  GOOS=windows/linux/android/ios          │
│  -tags console                           │
│  - Loopback-only (127.0.0.1:8090)       │
│  - Full authoritative sim                │
│  - Same Lobby, same routes               │
└─────────────────────────────────────────┘
```

---

## API Reference

### REST Endpoints (Selected)

#### Faith & Religion
```
GET  /api/faith/coherence
POST /api/faith/war-gambit
GET  /api/faith/religions
POST /api/faith/religion/buy
POST /api/faith/religion/buyout
POST /api/faith/religion/join
POST /api/faith/religion/ritual
POST /api/faith/religion/rivalry
GET  /api/faith/high-tier
GET  /api/faith/converted
```

#### Stat Overlay
```
GET  /api/stat-overlay
GET  /api/stat-overlay/region
GET  /api/stat-overlay/owner
GET  /api/stat-overlay/leaderboard
```

#### Industrial Loop
```
GET  /api/industrial-loop/metrics
GET  /api/industrial-loop/health
POST /api/industrial-loop/record
```

#### Identity
```
GET  /api/identity/profile
GET  /api/identity/events
GET  /api/identity/leaderboard
POST /api/identity/record
```

#### Entity Shares
```
POST /api/shares/issue
POST /api/shares/buy
GET  /api/shares/tokens
GET  /api/shares/holdings
```

#### Infrastructure Leasing
```
POST /api/lease/create
GET  /api/lease/list
GET  /api/lease/available
```

#### Governance
```
GET  /api/governance/weight
POST /api/governance/register
POST /api/governance/vote
POST /api/governance/close
GET  /api/governance/governor
GET  /api/governance/election
GET  /api/governance/leaderboard
```

#### Bridge Router
```
POST /api/bridge/asset
POST /api/bridge/confirm
GET  /api/bridge/assets
GET  /api/bridge/txs
GET  /api/bridge/summary
```

#### Creator Economy
```
POST /api/creator/dlc/create
POST /api/creator/dlc/purchase
GET  /api/creator/dlcs
GET  /api/creator/royalties
POST /api/creator/event/create
GET  /api/creator/events
POST /api/creator/event/attend
POST /api/creator/sub/create
GET  /api/creator/subs
```

#### Faith Church
```
POST /api/church/open
GET  /api/church/get
GET  /api/church/owner
GET  /api/church/region
POST /api/church/add-member
POST /api/church/remove-member
POST /api/church/add-item
GET  /api/church/items
POST /api/church/ritual
GET  /api/church/rituals
GET  /api/church/leaderboard
```

#### Launchpad
```
POST /api/launch/create
POST /api/launch/back
POST /api/launch/activate
POST /api/launch/integrate
GET  /api/launches
GET  /api/launch/get
GET  /api/launch/creator
```

#### Advertising
```
POST /api/ad/create
POST /api/ad/activate
POST /api/ad/pause
POST /api/ad/impression
POST /api/ad/click
GET  /api/ads
GET  /api/ads/advertiser
GET  /api/ads/region
GET  /api/ad/stats
```

#### Gaming OS
```
GET  /api/os/modules
POST /api/os/module/register
POST /api/os/lease
GET  /api/os/leases
GET  /api/os/summary
```

#### Compliance
```
POST /api/compliance/record
POST /api/compliance/resolve
POST /api/compliance/escalate
GET  /api/compliance/records
GET  /api/compliance/wallet
GET  /api/compliance/summary
```

### WebSocket Events

| Event | Direction | Purpose |
|-------|-----------|---------|
| `join_queue` | C→S | Enter matchmaking |
| `leave_queue` | C→S | Exit matchmaking |
| `challenge` | C→S | PvP challenge |
| `move` | C→S | In-match move |
| `update_rating` | C→S | Sync rating |
| `chat` | C→S | Lobby chat |
| `spectate` | C→S | Watch match |
| `set_district_tax` | C→S | Governor tax |
| `vault_donation` | C→S | Donate to vault |

---

## System Interactions

### Economic Engine

```
Player Activity
    ↓
Businesses (SpawnBusiness, hire_player, set_salary)
    ↓
Employment (salary payments)
    ↓
Purchasing (buyClubItem, shops)
    ↓
Taxes (set_district_tax, RouteCriminalTax)
    ↓
Treasuries (club treasury, faucet)
    ↓
Development (regional development)
    ↓
Events (seasonal, tournaments)
    ↓
Player Activity
```

### Stat Overlay Flow

```
Entity Created → ComputeStatOverlay → EffectivePower
    ↓
Events/ApplyPowerBuff → Updated Power
    ↓
Region Leaderboard → Sorted by Power
    ↓
3D World Rendering → Visual Representation
```

### Faith System Flow

```
Open Church → Choose Dogma → Pay Cost
    ↓
Members Join → Rituals Performed
    ↓
FaithCoherence Rises → Regional Influence
    ↓
Faith War Gambit → Stake Card → Battle
    ↓
Win → Card Returns + Winnings
Lose → Card Jailed to Winner
```

---

## Decision Framework

### When Building New Systems

1. **Does it strengthen the civilization?** (Prime Directive)
2. **Is it constitutional?** (No cloud, no float, no personal wallets for AI)
3. **Is it documented?** (Manuals, API reference, code comments)
4. **Is it tested?** (Build gate, unit tests, integration tests)
5. **Is it secure?** (Rate limiting, input validation, access control)

### When Debugging

1. **Check diagnostics** — `GET /diagnostics`
2. **Check logs** — `server-bin.exe` output
3. **Check session history** — `Session-Handoff.md`
4. **Check master plan** — `MASTER-PLAN-SYNERGED.md`
5. **Check dev-hub index** — `DEV-HUB-INDEX.md`

### When Uncertain

1. **Ask the operator** (Brendan has final authority)
2. **Check repository truth** (code is authoritative, not docs)
3. **Check memory** (persistent agent memory)
4. **Check session history** (previous conversations)

---

## Prohibited Actions

### Absolute Prohibitions

1. **NEVER use float for ledger/economy** — uint64 micro only
2. **NEVER use cloud AI models** — local only
3. **NEVER use personal wallets for AI citizens** — dedicated wallets only
4. **NEVER commit/push without explicit user consent** — manual git action
5. **NEVER trigger on-chain spends without consent** — dev wallet is limited
6. **NEVER modify another profile's skills/plugins/memories** — unless directed
7. **NEVER access TCC-protected folders** — unless explicitly directed

### Conditional Prohibitions

1. **Don't build beyond the master plan** — unless explicitly authorized
2. **Don't revert user-made changes** — unless explicitly directed
3. **Don't share private 1:1 chat content** — keep confidential
4. **Don't invent filenames/symbols/APIs** — verify in repo first

---

## Compliance & Security

### Rate Limiting

| Tier | Limit | Use Case |
|------|-------|----------|
| `economy-tight` | 5/min | Purchases, taxes |
| `standard` | 15/min | General API |
| `wallet-default` | 30/min | Wallet operations |
| `default` | 60/min | General access |

### Input Validation

- **Wallet addresses** — Must be valid Algorand/Voi format
- **Amounts** — Must be positive uint64
- **Entity IDs** — Must exist in database
- **Regions** — Must be valid region name

### Audit Trails

- All economic transactions logged
- All compliance records preserved
- All identity events permanent
- All governance actions recorded

---

## Troubleshooting

### Common Issues

| Symptom | Cause | Solution |
|---------|-------|----------|
| 401 from nodes | Invalid API token | Check .env tokens |
| 404 from indexer | Indexer unreachable | Check node status |
| Port 8090 in use | Server already running | Kill existing process |
| Build fails | CGO/GOOS mismatch | Set correct env vars |
| Route not found | Wrong path | Check server_main.go |
| Handler nil | Wrong build tag | Use correct tags |

### Diagnostic Endpoints

```
GET /diagnostics — System health
POST /api/client-error — Report client errors
```

---

*This manual trains AI agents to operate within the NFT-Seduction civilization. Follow it precisely.*
