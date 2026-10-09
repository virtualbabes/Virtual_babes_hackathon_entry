# Remaining Features — Synergy vs Rivalry Analysis

> **Date:** 2026-09-07
> **Purpose:** Compare remaining Triple Triad features against existing NFT-Seduction systems
> **Verdict:** ✅ SYNERGIZE | ❌ POINTLESS | ⚠️ RIVAL

---

## Feature-by-Feature Analysis

### 1. Card Titles/Perks (Triple Triad: card-titles.js + 3 more)

| NFT-Seduction Has | Overlap? | Synergy? |
|-------------------|----------|----------|
| Achievement Progress (achievement_progress.js) | No | ✅ **Connects** — Card perks = micro-achievements |
| Identity Editor (identity_editor.js) | No | ✅ **Displays** — Titles shown in player profile |
| Career Pathways (pathway_avenues.js) | No | ✅ **Unlocks** — Perks unlock at career tiers |
| Deck Manager (deck_manager.js) | No | ✅ **Shows** — Card details display titles |

**Verdict: ✅ SYNERGIZE — Adds depth to existing achievement/identity/deck systems**

---

### 2. Post-Match Wagers (Triple Triad: post-game-wagers.js)

| NFT-Seduction Has | Overlap? | Synergy? |
|-------------------|----------|----------|
| Economy Service (economy_service.go) | No | ✅ **Core** — Wagers flow through economy |
| Black Market (black_market.js) | No | ✅ **Tracks** — Gambling history on profile |
| Match Creator (match_creator.js) | Partial | ✅ **Extends** — Wager input exists, post-match is different |
| AMM Chart (amm_chart.js) | No | ✅ **Bets** — Prediction markets on match outcomes |

**Verdict: ✅ SYNERGIZE — Natural extension of match flow + economy + black market**

---

### 3. Character Mood Display (Triple Triad: character-mood.js)

| NFT-Seduction Has | Overlap? | Synergy? |
|-------------------|----------|----------|
| AI Citizens (ai_citizen_engine.go) | No | ✅ **Shows** — NPC morale affects behavior |
| Faith System (faith_church.go) | No | ✅ **Religious** — Devotion level = mood |
| Justice Dashboard (justice_dashboard.js) | No | ✅ **Wanted** — Criminal mood = aggression |
| Theme Engine (theme_engine.go) | Partial | ✅ **Mood** — Already computes mood effects |
| Card Mood (existing) | Partial | ⚠️ **Overlap** — Card mood exists, NPC mood is different |

**Verdict: ✅ SYNERGIZE (conditional) — NPC mood ≠ card mood. Connects AI citizens, faith, justice**

---

### 4. Meme Coin Market (Triple Triad: meme-coin-market.js)

| NFT-Seduction Has | Overlap? | Synergy? |
|-------------------|----------|----------|
| AMM Chart (amm_chart.js) | ⚠️ **YES** | ❌ Duplicate — Bonding curve + token trading already exist |
| Token Trader (economy_service.go) | ⚠️ **YES** | ❌ Duplicate — Token swaps handled |
| Black Market (black_market.js) | ⚠️ **YES** | ❌ Duplicate — Underground trading exists |
| Entity Market (entity_market.go) | ⚠️ **YES** | ❌ Duplicate — Public listings exist |

**Verdict: ❌ POINTLESS — Full overlap with AMM Chart + Economy + Entity Market**

---

### 5. Silk Road Trading (Triple Triad: silk-road.js)

| NFT-Seduction Has | Overlap? | Synergy? |
|-------------------|----------|----------|
| Nautilus DEX (nautilus_dex_path.go) | Partial | ⚠️ **Flavor** — Cross-chain trading exists |
| Entity Market (entity_market.go) | Partial | ⚠️ **Flavor** — Rare items exist |
| Bridge Router (bridge_router.go) | Partial | ✅ **Cross-chain** — Fits the bridge narrative |

**Verdict: ⚠️ RIVAL (partial) — Nautilus DEX covers cross-chain. Silk Road = flavor rename only.**

---

### 6. Swap Meet (Triple Triad: swap-meet.js)

| NFT-Seduction Has | Overlap? | Synergy? |
|-------------------|----------|----------|
| Entity Market (entity_market.go) | ⚠️ **YES** | ❌ Duplicate — Public listings + trade history |
| Black Market (black_market.js) | ⚠️ **YES** | ❌ Duplicate — Underground P2P trading |
| AMM Chart (amm_chart.js) | ⚠️ **YES** | ❌ Duplicate — Token swaps |

**Verdict: ❌ POINTLESS — Full overlap with Entity Market + Black Market + AMM**

---

### 7. Tea House (Triple Triad: tea-house.js)

| NFT-Seduction Has | Overlap? | Synergy? |
|-------------------|----------|----------|
| Club Foundry (club_service.js) | No | ✅ **Club type** — Tea House = social club |
| Social Hub (relationship_manager.js) | No | ✅ **NPC hub** — Social interactions |
| Lounge (game_locations.js) | Partial | ⚠️ **Similar** — Both are casual social spaces |

**Verdict: ✅ SYNERGIZE — New club type for Club Foundry. Adds social flavor.**

---

### 8. Zen Garden / Sanctuary (Triple Triad: zen-garden.js, sanctuary.js)

| NFT-Seduction Has | Overlap? | Synergy? |
|-------------------|----------|----------|
| Dividend Yield (dividend_yield.js) | Partial | ✅ **Visual** — Garden theme already exists |
| Faith Church (faith_storefront.js) | Partial | ✅ **Sanctuary** — Religious meditation space |
| Achievement Progress | No | ✅ **Reward** — Zen garden = achievement visualization |

**Verdict: ✅ SYNERGIZE (conditional) — Visual enhancement to Dividend Yield, Faith system**

---

### 9. 11 Missing API Routes

| Route | UI Tab Exists? | Impact |
|-------|---------------|--------|
| `/api/active-matches` | ❌ No | No UI to call it |
| `/api/bounty/active` | ✅ Yes (justice_dashboard) | Returns 404 — bounty board broken |
| `/api/children-bots/owner` | ❌ No | No UI to call it |
| `/api/church/members` | ✅ Yes (church_storefront) | Returns 404 — church broken |
| `/api/clubs` | ✅ Yes (club_foundry) | Returns 404 — clubs broken |
| `/api/creator/store/creator` | ✅ Yes (creator_studio) | Returns 404 — creator broken |
| `/api/justice/missions` | ✅ Yes (justice_dashboard) | Returns 404 — missions broken |
| `/api/player/profile` | ✅ Yes (identity_editor) | Returns 404 — profile broken |
| `/api/rumors` | ✅ Yes (underworld) | Returns 404 — rumors broken |
| `/api/shop/purchase` | ✅ Yes (church_storefront) | Returns 404 — shop broken |
| `/api/active-matches` | ❌ No | No UI to call it |

**Verdict: 🔴 CRITICAL — 8/11 routes back EXISTING UI tabs. Without them, tabs show 404 errors.**

---

### 10. 3 Empty Go Stubs

| Stub | Called From | Impact |
|------|------------|--------|
| `syncStatsFromBlockchain()` | Match end | Stats never reconcile from chain |
| `loadRegistrationsFromIndexer()` | Tournament start | Tournament entries never load |
| `checkAssetOptIn()` | Asset transfer | **Always returns true** — security bypass |

**Verdict: 🔴 CRITICAL — Breaks blockchain sync, tournament loading, asset security**

---

## Summary Matrix

| Feature | Verdict | Action |
|---------|---------|--------|
| Card Titles/Perks | ✅ SYNERGIZE | Port |
| Post-Match Wagers | ✅ SYNERGIZE | Port |
| Character Mood Display | ✅ SYNERGIZE | Port (NPC mood, not card mood) |
| Meme Coin Market | ❌ POINTLESS | Skip |
| Silk Road Trading | ⚠️ RIVAL | Skip (Nautilus DEX covers it) |
| Swap Meet | ❌ POINTLESS | Skip |
| Tea House | ✅ SYNERGIZE | Port (new club type) |
| Zen Garden/Sanctuary | ✅ SYNERGIZE | Port (visual enhancement) |
| 11 Missing API Routes | 🔴 CRITICAL | Implement 8 that back existing UI |
| 3 Empty Go Stubs | 🔴 CRITICAL | Implement all 3 |

---

## Recommended Priority

### Tier 1: CRITICAL (Fix Existing)
1. Implement 8 missing API routes that back existing UI tabs
2. Implement 3 empty Go stubs (blockchain sync, tournament loading, asset security)

### Tier 2: HIGH SYNERGY (Add Value)
3. Card Titles/Perks — connects to achievements, identity, deck manager
4. Post-Match Wagers — connects to economy, black market, match flow

### Tier 3: MEDIUM SYNERGY (Add Flavor)
5. Character Mood Display — connects to AI citizens, faith, justice
6. Tea House — new club type for Club Foundry
7. Zen Garden/Sanctuary — visual enhancement to dividend yield

### SKIP: Redundant
- Meme Coin Market → AMM Chart already exists
- Silk Road Trading → Nautilus DEX already exists
- Swap Meet → Entity Market already exists

---

## Key Insight

The **most critical work** isn't new features — it's **fixing the 8 API routes + 3 Go stubs** that break EXISTING UI. The tabs are built but non-functional without backend support.

After that, Card Titles + Wagers add the most value with least effort.
