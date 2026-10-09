# NFT-Seduction: World Dashboard — Categorization Summary

> **Purpose:** How the 43 world-dashboard tabs are grouped into 12 categories.
> **Last updated:** 2026-09-07
> **Source:** `world_dashboard.js` + `Background-vs-UI-Categorization.md`

---

## The 12 Categories

| # | Icon | Name | Tab Count | Tab IDs |
|---|---|---|---|---|
| 1 | 🎮 | Player Hub | 4 | identity, career, achievements, stats |
| 2 | 🎒 | Assets | 4 | items, pets, vehicles, clubs |
| 3 | 💰 | Economy & Trade | 4 | markets, dividends, loans, blackmarket |
| 4 | ⚖️ | Faction | 3 | justice, counterfeit, contracts |
| 5 | 🏛️ | Governance | 6 | governance, governor, territory, regions, compliance, leaderboard |
| 6 | 🌍 | World & Events | 9 | season, events, tournament, replay, treasure, worldcontent, industrial, maintenance, systemmsg |
| 7 | ⛪ | Faith & Church | 2 | faith, church |
| 8 | 🐾 | Orphans | 1 | orphan |
| 9 | 🎨 | Creator Economy | 3 | creator, launches, ads |
| 10 | 🤖 | Infrastructure | 2 | gamingos, localmodel |
| 11 | 🏆 | Competition | 3 | rivalry, match, rewards |
| 12 | 🛡️ | Moderation | 1 | report |
| **Total** | | | **43** | |

---

## What Changed (vs. Flat 43-Tab List)

### Merges
- **`gov` → `governance`** — eliminated duplicate tab; the `gov` route now routes to the `governance` tab

### Moves
| Tab | Old Position | New Position | Rationale |
|---|---|---|---|
| contracts | Economy | **Faction** | Underworld content, not general economy |
| counterfeit | Economy | **Faction** | Criminal justice activity |
| match | World & Events | **Competition** | PvP content, not passive world |
| rewards | World & Events | **Competition** | PvP reward tracking |
| replay | World & Events | **World & Events** (kept) | Match history is world activity |
| leaderboard | World & Events | **Governance** | Political ranking, not world events |
| territory | Governance | **Governance** (kept) | Regional control = governance |

### New Categories (didn't exist before)
- **🎒 Assets** — items, pets, vehicles, clubs (were scattered under "Personal" or uncategorized)
- **⚖️ Faction** — justice, counterfeit, contracts (were split between "Economy" and "Criminal")
- **⛪ Faith & Church** — faith, church (were standalone)
- **🐾 Orphans** — orphan (was standalone)
- **🎨 Creator Economy** — creator, launches, ads (were uncategorized)
- **🤖 Infrastructure** — gamingos, localmodel (were uncategorized)
- **🏆 Competition** — rivalry, match, rewards (were split across categories)
- **🛡️ Moderation** — report (was standalone)

---

## Background vs UI-Interactive by Category

### 🟢 Background (runs without user input)
| Category | Background Tabs |
|---|---|
| 🎮 Player Hub | identity, stats |
| 🎒 Assets | items, pets, vehicles |
| 💰 Economy | — |
| ⚖️ Faction | justice |
| 🏛️ Governance | territory, regions, compliance, leaderboard |
| 🌍 World | season, industrial, maintenance, systemmsg |
| ⛪ Faith | — |
| 🐾 Orphans | — |
| 🎨 Creator | launches, ads |
| 🤖 Infrastructure | gamingos |
| 🏆 Competition | rivalry, rewards |
| 🛡️ Moderation | — |

### 🔵 UI-Interactive (requires user action)
| Category | UI-Interactive Tabs |
|---|---|
| 🎮 Player Hub | — |
| 🎒 Assets | — |
| 💰 Economy | — |
| ⚖️ Faction | — |
| 🏛️ Governance | — |
| 🌍 World | replay |
| ⛪ Faith | — |
| 🐾 Orphans | — |
| 🎨 Creator | — |
| 🤖 Infrastructure | — |
| 🏆 Competition | — |
| 🛡️ Moderation | report |

### 🟡 Both (background + user interaction)
| Category | Both Tabs |
|---|---|
| 🎮 Player Hub | career, achievements |
| 🎒 Assets | clubs |
| 💰 Economy | markets, dividends, loans, blackmarket |
| ⚖️ Faction | counterfeit, contracts |
| 🏛️ Governance | governance, governor |
| 🌍 World | events, tournament, treasure, worldcontent |
| ⛪ Faith | faith, church |
| 🐾 Orphans | orphan |
| 🎨 Creator | creator |
| 🤖 Infrastructure | localmodel |
| 🏆 Competition | match |
| 🛡️ Moderation | — |

---

## Tab Starring & User Preferences

Each tab has a ☆/★ star toggle. Starred tabs appear in the **Constellation Hub** main screen grid for quick access. The `tabLabel()` function maps tab IDs to display names for the starred-items panel.

---

## Implementation Notes

- **Tab bar:** Built dynamically from `WD_CATEGORIES` array — add a new category by adding one object
- **Adding a tab:** Add to the `tabs` array in the appropriate category + add a `load<Tab>()` function + add `TAB_LABELS` entry
- **Routing:** `switchWDTab()` handles `gov` → `governance` merge automatically
- **Panels:** All 43 `<div id="wd-panel-*">` elements exist in the HTML; tabs just toggle `.hidden` class
- **CSS:** `.wd-cat` is `flex: 1 1 100%` so each category gets its own row; `.wd-cat-tabs` is indented 20px

---

*Next: Build the actual UI screens for each category — starting with the highest-traffic ones (Player Hub, Economy, Faction).*
