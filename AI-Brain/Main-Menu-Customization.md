# NFT-Seduction: Main Menu Customization System

> **Purpose:** Fully customizable Main Menu — button shapes, sizes, grid layouts, and positioning.
> **Design Goal:** Controller-friendly navigation (D-pad friendly) with visual flexibility.
> **Last updated:** 2026-09-07

---

## 1. Button Shape System

### 1.1 Shape Catalog

All shapes use CSS `clip-path` for rendering. Each shape has a clear **directional axis** for controller navigation.

| ID | Name | Clip-Path | Controller Axis | Visual |
|---|---|---|---|---|
| `circle` | Circle | `circle(50% at 50% 50%)` | Omnidirectional | ● |
| `oval` | Oval | `ellipse(50% 35% at 50% 50%)` | Horizontal | ⬭ |
| `star` | Star (5-point) | `polygon()` 10-point star | Top/bottom points | ★ |
| `triangle` | Triangle | `polygon(50% 0%, 0% 100%, 100% 100%)` | Top/base vertices | ▲ |
| `oblong` | Oblong | `border-radius: 25px` | Horizontal | ▬ |
| `hexagon` | Hexagon | `polygon()` 6-point | Left/right vertices | ⬡ |
| `octagon` | Octagon | `clip-path: polygon(30% 0%, 70% 0%, 100% 30%, 100% 70%, 70% 100%, 30% 100%, 0% 70%, 0% 30%)` | Flat edges | ⯃ |
| `rectangle` | Rectangle | `border-radius: 6px` | Horizontal | ■ |
| `diamond` | Diamond | `polygon(50% 0%, 100% 50%, 50% 100%, 0% 50%)` | Vertices | ◆ |
| `pentagon` | Pentagon | `polygon()` 5-point | Top vertex | ⬠ |
| `shield` | Shield | `polygon(50% 0%, 100% 20%, 100% 70%, 50% 100%, 0% 70%, 0% 20%)` | Top/bottom | 🛡 |
| `arrow-right` | Arrow Right | `polygon()` arrow shape | Left→Right | ▶ |
| `arrow-left` | Arrow Left | `polygon()` arrow shape | Right→Left | ◀ |
| `cross` | Plus/Cross | `polygon()` plus shape | Up/down/left/right | ✚ |
| `parallelogram` | Parallelogram | `polygon(25% 0%, 100% 0%, 75% 100%, 0% 100%)` | Diagonal | ▱ |

### 1.2 Shape Hit Areas

For controller navigation, each shape exposes **logical hit directions**:

```
        Up
         ▲
Left ◄ ─●─ ► Right
         ▼
       Down
```

- **Circle/Oval** — All 4 directions valid
- **Triangle** — Up points to apex, Down connects to base vertices
- **Star** — Up points to top star point, Down to bottom star point
- **Hexagon/Octagon** — Cardinal directions map to flat edges
- **Diamond** — Diagonal directions map to vertices
- **Arrow-Right** → Left = back, Right = select, Up/Down = sibling buttons

### 1.3 Clip-Path Definitions (CSS)

```css
/* === SHAPE LIBRARY === */
.shape-circle    { clip-path: circle(50% at 50% 50%); }
.shape-oval      { clip-path: ellipse(50% 35% at 50% 50%); }
.shape-star      { clip-path: polygon(50% 0%, 61% 35%, 98% 35%, 68% 57%, 79% 91%, 50% 70%, 21% 91%, 32% 57%, 2% 35%, 39% 35%); }
.shape-triangle  { clip-path: polygon(50% 0%, 0% 100%, 100% 100%); }
.shape-oblong    { clip-path: inset(0% 0% 0% 0% round 25px); } /* uses border-radius */
.shape-hexagon   { clip-path: polygon(25% 0%, 75% 0%, 100% 50%, 75% 100%, 25% 100%, 0% 50%); }
.shape-octagon   { clip-path: polygon(30% 0%, 70% 0%, 100% 30%, 100% 70%, 70% 100%, 30% 100%, 0% 70%, 0% 30%); }
.shape-rectangle { clip-path: inset(0% 0% 0% 0% round 6px); }
.shape-diamond   { clip-path: polygon(50% 0%, 100% 50%, 50% 100%, 0% 50%); }
.shape-pentagon  { clip-path: polygon(50% 0%, 100% 38%, 82% 100%, 18% 100%, 0% 38%); }
.shape-shield    { clip-path: polygon(50% 0%, 100% 20%, 100% 65%, 50% 100%, 0% 65%, 0% 20%); }
.shape-arrow-r   { clip-path: polygon(0% 50%, 50% 0%, 50% 35%, 100% 35%, 100% 65%, 50% 65%, 50% 100%); }
.shape-arrow-l   { clip-path: polygon(50% 0%, 50% 35%, 0% 35%, 0% 65%, 50% 65%, 50% 100%, 100% 50%); }
.shape-cross     { clip-path: polygon(35% 0%, 65% 0%, 65% 35%, 100% 35%, 100% 65%, 65% 65%, 65% 100%, 35% 100%, 35% 65%, 0% 65%, 0% 35%, 35% 35%); }
.shape-parallelogram { clip-path: polygon(25% 0%, 100% 0%, 75% 100%, 0% 100%); }
```

---

## 2. Button Size System

### 2.1 Size Tiers

| ID | Name | Min Width | Min Height | Font Size | Icon Size | Use Case |
|---|---|---|---|---|---|---|
| `xs` | Tiny | 32px | 32px | 8px | 14px | Status indicators, sub-actions |
| `sm` | Small | 48px | 48px | 9px | 18px | Compact menus, many items |
| `md` | Medium (default) | 72px | 72px | 11px | 24px | Standard favorites |
| `lg` | Large | 96px | 96px | 13px | 32px | Primary actions |
| `xl` | Extra Large | 128px | 128px | 15px | 40px | Hero buttons |
| `xxl` | Huge | 160px | 160px | 18px | 48px | Splash/featured actions

### 2.2 Size Scaling with Shape

- **Circle**: Diameter = size value
- **Triangle**: Height = size value, width auto
- **Hexagon**: Width = size value, height = size × 0.866
- **Star**: Bounding box = size value
- **Oblong**: Width = size × 1.5, Height = size × 0.6

### 2.3 Responsive Sizing

```
Viewport < 600px   → size *= 0.7
600-1024px         → size *= 1.0 (default)
1024-1920px        → size *= 1.2
> 1920px            → size *= 1.4
```

---

## 3. Grid Layout System

### 3.1 Grid Catalog

| ID | Name | Arrangement | Controller Flow | Best For |
|---|---|---|---|---|
| `grid` | Standard Grid | Row-major grid | Left/Right/Up/Down | 4-12 items |
| `circle` | Radial Wheel | Items on circle circumference | Clockwise/counter-clockwise + center | 5-8 items, circular controller feel |
| `triangle` | Pyramid | 1-2-3-4... triangle | Layer by layer | Hierarchical menus (career trees) |
| `cross` | Plus/Cross | Center + 4 arms | Cardinal directions | 4-directional control schemes |
| `arc` | Arc/Semicircle | Items in semicircle | Left/right sweep | 3-5 items, dashboards |
| `diamond` | Diamond | Items in diamond pattern | Diagonal nav | 4-8 items |
| `linear-h` | Horizontal Line | Single row | Left/Right | Few items, top-bar |
| `linear-v` | Vertical Line | Single column | Up/Down | Few items, side-bar |
| `freeform` | Free Position | Absolute coordinates | Manual tab order | Custom layouts, controller macros |

### 3.2 Grid Positioning

| Property | Values | Description |
|---|---|---|
| `gridType` | grid/circle/triangle/cross/arc/diamond/linear-h/linear-v/freeform | Layout algorithm |
| `gridX` | `center` / `left` / `right` / `%` | Horizontal anchor |
| `gridY` | `center` / `top` / `bottom` / `%` | Vertical anchor |
| `gridW` | `auto` / `%` / `px` | Grid container width |
| `gridH` | `auto` / `%` / `px` | Grid container height |
| `gridGap` | `px` / `rem` | Space between items |
| `gridCols` | number | Columns (grid type only) |
| `gridRows` | number | Rows (grid type only) |
| `gridRadius` | `%` / `px` | Radius for circle/arc types |
| `gridRotation` | `deg` | Rotation for circle/arc types |

### 3.3 Layout Diagrams

#### Grid (Standard)
```
┌──────┬──────┬──────┐
│  1   │  2   │  3   │
├──────┼──────┼──────┤
│  4   │  5   │  6   │
├──────┼──────┼──────┤
│  7   │  8   │  9   │
└──────┴──────┴──────┘
Controller: 2D directional
```

#### Circle (Radial Wheel)
```
        [1]
    [8]     [2]
  [7]    ★    [3]
    [6]     [4]
        [5]
Controller: Left/Right = rotate, Up = select, Down = back
```

#### Triangle (Pyramid)
```
        [1]
      [2] [3]
    [4] [5] [6]
  [7] [8] [9] [10]
Controller: Down = next layer, Up = prev layer, Left/Right = sibling
```

#### Cross (Plus)
```
         [2]
          │
    [1]── ★ ──[3]
          │
         [4]
Controller: Pure cardinal (D-pad perfect)
```

#### Arc (Dashboard)
```
  [1]────[2]────[3]────[4]
         ╲    ╱
          ╲  ╱
           ★
Controller: Left/Right sweep
```

---

## 4. Customization Flow

### 4.1 User Customization UI

```
┌──────────────────────────────────────────┐
│       MENU CUSTOMIZATION PANEL            │
├──────────────────────────────────────────┤
│  Button Shape: [Circle ▼]  Size: [MD ▼] │
│                                          │
│  Grid Type:    [Grid ▼]   Pos: [Center] │
│  Grid Gap:     [12px]     Cols: [3]     │
│                                          │
│  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐    │
│  │ 🌍   │ │ ⚡   │ │ 👤   │ │ 📚   │    │
│  │World │ │Quick │ │Profil│ │Tutori│    │
│  └──────┘ └──────┘ └──────┘ └──────┘    │
│                                          │
│  [Preview]  [Save]  [Reset Defaults]     │
└──────────────────────────────────────────┘
```

### 4.2 Per-Button Override

Each button can override the global defaults:

```json
{
  "buttons": [
    { "id": "wd", "label": "World Dashboard", "icon": "🌍", "shape": "circle", "size": "lg", "color": "#00bcd4" },
    { "id": "qp", "label": "Quick Play", "icon": "⚡", "shape": "star", "size": "md", "color": "#ff4b4b" },
    { "id": "pf", "label": "Profile", "icon": "👤", "shape": "hexagon", "size": "sm", "color": "#6366f1" },
    { "id": "tut", "label": "Tutorial", "icon": "📚", "shape": "oval", "size": "xs", "color": "#9c27b0" }
  ]
}
```

### 4.3 Preset Layouts

| Preset | Grid | Button Shape | Size | Use Case |
|---|---|---|---|---|
| Default | 2×2 Grid | Circle | md | Standard |
| Compact | 4×1 Linear-H | Oblong | sm | Mobile/Tablet |
| Wheel | Radial Circle | Circle | md | Controller |
| Dashboard | Arc | Rectangle | lg | Desktop |
| Pyramid | Triangle | Star | md | Progression Showcase |
| Cross | Cross | Circle | lg | Minimalist |
| Freeform | Free | Mixed | Mixed | Power Users |

---

## 5. Controller Integration

### 5.1 Navigation Map (per grid type)

| Grid | Up | Down | Left | Right | Select | Back |
|---|---|---|---|---|---|---|
| Grid | row-1 | row+1 | col-1 | col+1 | A/Enter | B/Esc |
| Circle | center | outer | CW | CCW | A | B |
| Triangle | prev layer | next layer | sibling- | sibling+ | A | B |
| Cross | arm-up | arm-down | arm-left | arm-right | A | B |
| Arc | - | - | prev | next | A | B |
| Linear-H | - | - | prev | next | A | B |
| Linear-V | prev | next | - | - | A | B |

### 5.2 Focus Ring

```css
.menu-btn:focus-visible {
  outline: 3px solid var(--pathway-glow);
  outline-offset: 4px;
  transform: scale(1.1);
  z-index: 100;
}

/* Controller focus uses different style than keyboard */
.menu-btn.controller-focus {
  animation: focus-pulse 1s infinite;
  box-shadow: 0 0 20px var(--pathway-glow);
}
```

### 5.3 Dead Zone Padding

Buttons have **inner padding** for shape visual + **outer hit area** for controller:

```css
.menu-btn {
  /* Visual size */
  width: var(--btn-size);
  height: var(--btn-size);
  /* Hit area (larger than visual for controller) */
  padding: 4px;
  margin: 6px;
}
```

---

## 6. Implementation Architecture

### 6.1 Data Model

```javascript
// User Preferences extension
{
  "menuLayout": {
    "gridType": "grid",       // grid/circle/triangle/cross/arc/diamond/linear-h/linear-v/freeform
    "gridX": "center",
    "gridY": "center",
    "gridW": "auto",
    "gridH": "auto",
    "gridGap": "12px",
    "gridCols": 3,
    "gridRows": 0,            // 0 = auto
    "gridRadius": "40%",
    "gridRotation": "0deg",
    "defaultShape": "circle",
    "defaultSize": "md",
    "buttonOverrides": {
      "wd":   { "shape": "circle", "size": "lg" },
      "qp":   { "shape": "star", "size": "md" },
      "pf":   { "shape": "hexagon", "size": "sm" }
    }
  }
}
```

### 6.2 Render Pipeline

```
UserPreferences.getMenuLayout()
    ↓
computeGridPositions(gridType, items[], containerWidth, containerHeight)
    ↓
For each item:
  - shape = buttonOverrides[item.id]?.shape || defaultShape
  - size  = buttonOverrides[item.id]?.size  || defaultSize
  - applyCSS(button, shape, size, color, position)
    ↓
attachControllerHandlers() (when controller detected)
```

### 6.3 Grid Position Calculator

```javascript
function computeGridPositions(layout, items, containerW, containerH) {
  const positions = [];
  
  switch (layout.gridType) {
    case 'grid':
      // Row-major grid layout
      break;
    case 'circle':
      // Items on circumference
      const cx = containerW / 2;
      const cy = containerH / 2;
      const r = Math.min(containerW, containerH) * parseFloat(layout.gridRadius) / 100;
      items.forEach((item, i) => {
        const angle = (2 * Math.PI * i) / items.length - Math.PI / 2;
        positions.push({
          x: cx + r * Math.cos(angle) - item.w / 2,
          y: cy + r * Math.sin(angle) - item.h / 2,
        });
      });
      break;
    case 'triangle':
      // Pyramid: 1, 2, 3, 4... items per row
      break;
    case 'cross':
      // Center + 4 cardinal arms
      break;
    // ... etc
  }
  
  return positions;
}
```

---

## 7. Phase Plan

### Phase 2A: Button Shapes + Sizes
- [ ] Create `menu_customization.js` with shape library
- [ ] Create `menu_customization.scss` with clip-paths
- [ ] Add size tier CSS variables
- [ ] Add per-button override in UserPreferences
- [ ] Wire Main Menu to read shape/size from prefs

### Phase 2B: Grid Layouts
- [ ] Implement `computeGridPositions()` for all 9 grid types
- [ ] Add grid layout picker to customization panel
- [ ] Store grid config in UserPreferences
- [ ] Main Menu renders items at computed positions

### Phase 2C: Customization UI
- [ ] Build overlay panel for menu customization
- [ ] Shape picker (visual preview of each shape)
- [ ] Size slider/dropdown
- [ ] Grid type picker (diagram preview)
- [ ] Live preview of changes
- [ ] Save/Load/Reset buttons

### Phase 2D: Controller Support
- [ ] Detect gamepad connection
- [ ] Map D-pad to grid navigation
- [ ] Focus ring animation
- [ ] Select/Back button mapping
- [ ] Haptic feedback (where supported)

---

## 8. Key Design Principles

1. **Shape = affordance** — A star button FEELS different from a circle button
2. **Grid = navigation model** — Controller users think in directions, not coordinates
3. **Defaults first** — Most users won't customize; the default must be good
4. **Per-button override** — One hero button can be XL while the rest are MD
5. **Live preview** — No save-then-see; changes apply immediately with undo

---

*Next: Phase 2A implementation — shape library + size system + UserPreferences extension.*
