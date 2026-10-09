# Dynamic Game Types — Port Analysis (Triple Triad → NFT-Seduction)

> **Source:** `VBTT_DRAUGHT_BETA0.0.01/Tripple_Triad/` (Brendan's original repo)
> **Date:** 2026-09-07
> **Purpose:** Identify reusable dynamic game types, rules, and mechanics for NFT-Seduction card battles

---

## ✅ SAFE TO PORT — Created by Brendan, explicit authorization

---

## 1. Rules Engine (`rules-engine.js` — 264 lines)

### What It Does
Full capture rule system: Plus, Same, Combo, Prisoner rules with mood modifiers, dynamic stats, equipment bonuses

### Port to NFT-Seduction

| Rule | Triple Triad | NFT-Seduction Current | Action |
|------|-------------|----------------------|--------|
| **Plus** | `activeRules.includes('plus')` — sum of adjacent powers = capture | `rules["Power_up"]` exists | ✅ Rename/reuse |
| **Same** | `activeRules.includes('same')` — same value = capture group | `rules["Power_copy"]` exists | ✅ Rename/reuse |
| **Combo** | Chained flips from captured cards | `rules["Combo"]` exists | ✅ Already implemented |
| **Prisoner** | Captured cards get debuff | Not in NFT-Seduction | 🆕 **NEW RULE** — great addition! |
| **Mood Modifiers** | Rock-paper-scissors: Volatile→Serene→Spirited→Grounded→Volatile | `getEffectivePower()` has mood | ✅ Port the modifier system |
| **Equipment Bonuses** | Weapons add offense/defense boost | Not in NFT-Seduction | 🆕 **NEW FEATURE** |
| **Dynamic Stats** | Level-based scaling with mood distribution | Not in NFT-Seduction | 🆕 **NEW FEATURE** |

### Key Code Pattern (Port This)
```javascript
// Mood weaknesses system — RPS-style counter
const moodWeaknesses = {
    'Volatile': 'Serene',
    'Serene': 'Spirited',
    'Spirited': 'Grounded',
    'Grounded': 'Volatile'
};

// Card mood vs slot mood = ±1 power modifier
if (card.mood === slotMood) moodModifier = +1;
else if (moodWeaknesses[card.mood] === slotMood) moodModifier = -1;

// Apply to all 4 sides, clamp 1-10
modifiedStats[side] = Math.max(1, Math.min(10, baseStats[side] + moodModifier));
```

---

## 2. Campaign System (`campaign.js` — 797 lines)

### What It Does
Full campaign mode with lives, stages, linear unlocks, special campaigns

### Port to NFT-Seduction

| Feature | Triple Triad | NFT-Seduction Equivalent |
|---------|-------------|-------------------------|
| **Lives System** | 3 lives, lose all = restart campaign | Use for ranked/casual matches |
| **Stage Progression** | Linear unlock (complete 1 → unlock next) | Use for career path stages |
| **Special Campaigns** | Non-linear unlock (tax_overhaul, etc.) | Use for faction-specific campaigns |
| **Progress Persistence** | Server-side save/load | Already have WASM persistence |
| **Opponent Types** | Different AI per stage | Already have AI system |

### Key Code Pattern (Port This)
```javascript
// Default progress generation with merge from server
_generateDefaultProgress() {
    const defaultProgress = { lives: 3 };
    Object.keys(this.campaignData).forEach((key, index) => {
        defaultProgress[key] = { completed: 0, unlocked: index === 0 };
    });
    return defaultProgress;
}

// Merge server progress over local defaults
this.campaignProgress = { ...defaultProgress, ...progress };
```

---

## 3. Post-Game Modes (`post-game-modes.js` — 55 lines)

### What It Does
Multiple game mode resolutions: drunken brawl, fight club gauntlet, underworld job, kidnap gambit, invasion

### Port to NFT-Seduction — NEW GAME MODES

| Mode | Description | NFT-Seduction Equivalent |
|------|-------------|-------------------------|
| **drunken_brawl** | Casual brawl with reputation reward | Casual match mode |
| **fight_club_gauntlet** | Lose = lose your best card (prisoner rule) | Ranked with card risk |
| **underworld_job** | Heist-themed match with job data | Criminal pathway mission |
| **kidnap_gambit** | Kidnap-themed match with ransom | Kidnap system integration |
| **invasion** | Club vs club territory war | Club warfare mode |
| **campaign** | Standard campaign progression | Career mode |

### Key Code Pattern (Port This)
```javascript
// Mode-based resolution
resolve(data, winner) {
    if (data.gameMode === 'fight_club_gauntlet') {
        if (data.activeRules.includes('prisoner') && winner !== 'player') {
            // Lose your best card
            lostCard = data.playerHand.sort((a, b) => b.level - a.level)[0];
        }
    }
}
```

---

## 4. Dynamic Stats System (`dynamic-stats.js` — 55 lines)

### What It Does
Level-based stat scaling with mood-based stat distribution

### Port to NFT-Seduction

| Feature | Triple Triad | NFT-Seduction Equivalent |
|---------|-------------|-------------------------|
| **Level Scaling** | `baseStat * 1.1^(level-1)` | Card XP/leveling system |
| **Mood Distribution** | Different stats prioritized per mood | Mood already affects power |
| **Stat Cap** | Max 10 per side | Cap at UINT64 max |
| **Ace Stats** | 'A' = 10 * scaling | Special card tier system |

### Key Code Pattern (Port This)
```javascript
// Mood-based stat distribution
const mood_modifiers = {
    'Grounded': { top: 1, right: 1.2, bottom: 1.5, left: 1.2 },
    'Spirited': { top: 1.2, right: 1.5, bottom: 1, left: 1.2 },
    'Serene':   { top: 1.5, right: 1, bottom: 1.2, left: 1.2 },
    'Volatile': { top: 1.2, right: 1.2, bottom: 1, left: 1.5 },
};

// Distribute points by mood weights
const total_points = Math.floor(card.level * growth_rate * 4);
for (const stat in base_stats) {
    const weight = modifier[stat] / total_modifier_weight;
    card.stats[stat] = Math.max(1, base_stats[stat] + Math.round(total_points * weight));
}
```

---

## 5. Character Mood System (`character-mood.js` — 84 lines)

### What It Does
Mood states (Hostile → Loyal) with icons, colors, average mood tracking

### Port to NFT-Seduction

| Feature | Triple Triad | NFT-Seduction Equivalent |
|---------|-------------|-------------------------|
| **Mood States** | 5 states: Hostile, Annoyed, Neutral, Friendly, Loyal | NPCs/citizens already have mood |
| **Mood Icons** | 😠 😒 😐 😊 😍 | Use for NPC interactions |
| **Mood Colors** | Red, Yellow, Gray, Green, Blue | Region/area theming |
| **Average Mood** | Computed across all characters | City/world mood average |

### Key Code Pattern (Port This)
```javascript
// Mood state from numeric value
getMoodState(value) {
    if (value <= 20) return 'Hostile';
    if (value <= 40) return 'Annoyed';
    if (value <= 60) return 'Neutral';
    if (value <= 80) return 'Friendly';
    return 'Loyal';
}
```

---

## 6. Deck Builder (`deck-manager.js` + `card-loadout.js`)

### What It Does
Card selection, deck validation, loadout management

### Port to NFT-Seduction

| Feature | Triple Triad | NFT-Seduction Need |
|---------|-------------|-------------------|
| **Card Pool** | All owned cards | Inventory cards |
| **Deck Validation** | 5 cards required | 5 cards required |
| **Deck Rating** | Sum of card power | Sum of power (UINT64) |
| **Multiple Presets** | Save/load multiple decks | Deck presets per faction |
| **Loadout** | Equipment + deck | Weapons + deck |

---

## 7. Game Flow Controller (`game-flow-controller.js`)

### What It Does
Turn management, phase transitions, game state machine

### Port to NFT-Seduction
- Already have turn system in main.go
- Enhance with: phase timers, turn limits, sudden death

---

## 8. Card Progression (`card-training.js`, `card-fusion.js`, `card-upgrade.js`)

### What It Does
Cards level up, fuse, upgrade stats

### Port to NFT-Seduction

| Feature | Triple Triad | NFT-Seduction Equivalent |
|---------|-------------|-------------------------|
| **Training** | Cards gain XP from battles | Battle rewards → card XP |
| **Fusion** | Combine 2 cards → better card | Card crafting/synthesis |
| **Upgrade** | Level up → dynamic stats | Card leveling with stat growth |
| **Titles** | Card titles with perks | Card achievements/perks |

---

## 9. Multiple Game Locations (`arcade.js`, `casino.js`, `dive-bar.js`, `lounge.js`, etc.)

### What It Does
Different locations have different rules/styles

### Port to NFT-Seduction

| Location | Triple Triad Style | NFT-Seduction Equivalent |
|----------|-------------------|-------------------------|
| **Arcade** | Casual, quick matches | Quick play |
| **Casino** | High-stakes wagering | Wager matches |
| **Dive Bar** | Rough, no rules | Underground matches |
| **Lounge** | Premium, ranked | Ranked arena |
| **Underground Fight Club** | Prisoner rule active | Criminal pathway |

---

## 10. Board Themes (`game-board-themes.json` — empty but structured)

### What It Does
Different visual themes for game board

### Port to NFT-Seduction
- **Region-based themes:** Justice region = blue/red, Underworld = black/gold
- **Elemental themes:** Fire board = red glow, Water board = blue shimmer
- **Mood themes:** Wild board = chaotic animations, Calm board = smooth transitions

---

## Summary: What to Port

| Priority | Feature | Effort | Impact |
|----------|---------|--------|--------|
| 🔴 **P0** | Mood Modifiers (RPS system) | 1-2h | High — adds strategic depth |
| 🔴 **P0** | Prisoner Rule | 1h | High — risk/reward matches |
| 🔴 **P0** | Dynamic Stats (level scaling) | 2-3h | High — card progression |
| 🟡 **P1** | Campaign Mode | 3-4h | High — single-player content |
| 🟡 **P1** | Post-Game Modes | 2-3h | Medium — variety |
| 🟡 **P1** | Equipment Bonuses | 2-3h | Medium — card loadout |
| 🟢 **P2** | Character Mood Display | 1h | Low — NPC flavor |
| 🟢 **P2** | Card Titles/Perks | 2-3h | Medium — achievement system |
| 🟢 **P2** | Board Themes | 1-2h | Low — visual variety |

---

## Next Steps

1. **Approve port list** — which features to prioritize
2. **Start with Mood Modifiers + Prisoner Rule** — highest impact, lowest effort
3. **Then Campaign Mode** — single-player content
4. **Then Dynamic Stats** — card progression

---

*All code patterns extracted from Brendan's original Triple Triad repo. Explicit authorization to port.*
