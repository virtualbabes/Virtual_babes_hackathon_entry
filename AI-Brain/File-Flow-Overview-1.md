## [AUTHORITATIVE CORE: END-GAME ARCHITECTURE] 

### 📑 DOCUMENT STATUS: AUTHORITATIVE BLUEPRINT (V2.1)
This document serves as the supreme architectural reference. All logic implementations and file flows must align with the topology defined herein. 
*   **Last Updated:** 23/06/2026
*   **Pending Tasks:** Consult `ToDo.md`
*   **Phase 2 Career Wiring Complete:** Underworld #3-10 + Justice D1-D6 XP triggers verified; Justice D2-TaxAuditor resolution bonus pending
*   **Implementation History:** Search `A.I_memory.md`

---

## 📑 CANONICAL INDEX
1. Tactical Synchronization Snapshot
2. Synergy Architecture
3. Client-Side Interaction Flow
4. Authoritative Core Topology
5. UI File System Flow
6. SCSS Architecture
7. Operational Integrity: File Registry
8. Core Economic Sequences
9. Future Cross-Platform Console Expansion Map

---

## 1. Tactical Synchronization Snapshot
* `JS sends a move to the Backend via WebSocket.`
* `Backend validates the move, increments the Match SequenceID, and computes a BoardStateHash.`
* `Backend calculates Factional Combat Boosts (+10%) based on Signature Alignment (Justice vs Outlaw).`
* `Backend constructs an AuthoritativeFrame (Move + Sequence + Hash) and commits to sh.HistoricalFrames.`
* `Backend broadcasts the AuthoritativeFrame JSON to all match participants and spectators (handled in lobby_manager.go).`
* `JS feeds the frame into window.SyncMove (Go WASM).`
* `WASM Engine predicts UI changes instantly (Optimistic UI) based on local calculations.`
* `WASM Engine verifies Sequence continuity and StateHash parity; if a mismatch is found, it triggers Replay Recovery.`
* `WASM Engine updates local state, triggers checkCaptures, and calls window.syncUI("combat") automatically.`
* `The SCSS layer triggers a .flip-capture animation; if a Factional Boost is active, the card displays a pulsing neon cyan (Justice) or green (Underworld) glow.`
* `If server audit fails, UI smoothly rolls back to reflect authoritative state.`

## 2. High-Level Overview: Synergy Architecture
The Virtualbabes Arena is a production-hardened social economic simulation. It utilizes a Dual-Target build strategy to enforce bit-perfect parity between the Go Backend and the Deterministic WASM Game Engine. 
**Note:** All paths below are relative to the project root.

### 2.1 Backend Topology (Go - `./*.go`)
*   **authoritative Core**: `./server.go` initializes the event loop and management kernels.
*   **Service Delegation**: Specialized logic is isolated in `./battle_service.go`, `./club_service.go`, `./faucet_service.go`, and `./onboarding_service.go`.
 *   **Economic Integrity (Pillar 2 - Complete)**: `./economy_service.go` (core engine), `./economy_bootstrap.go` (blockchain reconstruction), `./economy_processing.go` (TokenSinkRouter + AMM payouts + RevenueSplitMatrix), `./economy_persistence.go` (JSON snapshots → blockchain notes), `./economy_audit.go` (anti-whale intercept + drift detection), `./economy_telemetry.go` (Prometheus metrics :9090).
*   **Replay Resilience**: `./lobby_manager.go` coordinates with `./backend_types.go` to manage match sequencing and connection quarantine.
*   **Economy Persistence Layer**: `./economy_bootstrap.go` (state bootstrapping) and `./economy_persistence.go` (atomic snapshots).
*   **Shop Registry**: `./shop_registry.go` provides the canonical item catalog for all district shops.
*   **Public Handlers**: `./handlers_public.go` serves public-facing data endpoints for unauthenticated queries.
*   **Rival Engine**: `./rival_career_engine.go` drives NPC career progression; `./rivalry_handlers.go` exposes rivalry state via HTTP/WebSocket.
*   **DEX Integration**: `./nautilus_dex_path.go` defines Nautilus DEX routing for cross-chain liquidity swaps.
*   **Game Layers**: Supports the emerging **Underworld vs. Justice Hegemony** through expanded roles and specialized economic interactions.
*   **Resiliency Layer**: `./resilience_utils.go` provides load-balanced RPC cluster management.

### 2.2 Game Engine (Go WASM - `./main.go`)
*   **Deterministic Logic**: Core rules and AI evaluation execute in `./Public/main.wasm`.
*   **Resilience Kernel**: Implements the `ClientReplayEngine` and `RedirectManager` for frame catch-up and interaction locking.
*   **IPC Bridge**: `./bridge_service.go` defines the explicit IPC (Inter-Process Communication) translation registry between Go WASM and Browser JavaScript (Pillar 4).
*   **Card Archetypes**: Supports new **Justice Cards** and **Underworld Cards** with specialized combat mechanics.
### 2.3 Frontend Orchestration (JS/SCSS - `./Public/`)
*   **Modular Authority**: `./Public/app.js` delegates UI rendering to specialized modules in `./Public/js/`.
*   **Visual Fidelity**: `./Public/styles.css` is pre-processed from the atomic SCSS structure in `./Public/src/scss/`.

## 3. Detailed Client-Side Interaction Flow

### 3.1 Public-JS Mermaid Map
graph TD
    subgraph Browser Environment
        HTML(./Public/index.html) --> AppJS(./Public/app.js)
        AppJS -- Loads & Runs --> WASM_EXEC(./Public/wasm_exec.js)
        WASM_EXEC -- Provides Runtime --> WASM_ENGINE(./Public/main.wasm - Go WASM Engine)
    end

    subgraph Client-Side JavaScript Modules
        AppJS -- Coordinates --> Config(./Public/js/config.js)
        AppJS -- Coordinates --> Network(./Public/js/network.js)
        AppJS -- Coordinates --> Wallet(./Public/js/wallet.js)
        AppJS -- Coordinates --> UI(./Public/js/ui.js)
        AppJS -- Coordinates --> Game(./Public/js/game.js)
        AppJS -- Coordinates --> Deck(./Public/js/deck.js)
        AppJS -- Coordinates --> Economy(./Public/js/economy.js)
        AppJS -- Coordinates --> Criminality(./Public/js/criminality.js)
        AppJS -- Coordinates --> Admin(./Public/js/admin.js)
        AppJS -- Coordinates --> Leaderboard(./Public/js/leaderboard.js)
        AppJS -- Coordinates --> Audio(./Public/js/audio.js)
        AppJS -- Coordinates --> Particles(./Public/js/particles.js)
        AppJS -- Coordinates --> Utils(./Public/js/utils.js)
        AppJS -- Beacon Push/Pull --> Storage[(LocalStorage)]
        AppJS -- Coordinates --> CollectiveAI(./Public/collective-intelligence.js)

        Game -- Triggers Taunt --> CollectiveAI
        UI -- Render Taunt --> CollectiveAI
        Network -- Uses --> Config
        Wallet -- Uses --> Config
        UI -- Uses --> Config
        Game -- Uses --> Config
        Deck -- Uses --> Config
        Economy -- Uses --> Config
        Criminality -- Uses --> Config
        Admin -- Uses --> Config
        Leaderboard -- Uses --> Config
        Utils -- Uses --> Config

        Wallet -- Sends WS messages --> Network
        Admin -- Sends WS messages --> Network
        Game -- Sends WS messages --> Network
        Deck -- Sends WS messages --> Network
        Economy -- Sends WS messages --> Network
        Criminality -- Sends WS messages --> Network
        Leaderboard -- Sends WS messages --> Network

        Network -- Dispatches messages to --> Wallet
        Network -- Dispatches messages to --> UI
        Network -- Dispatches messages to --> Game
        Network -- Dispatches messages to --> Admin
        Network -- Dispatches messages to --> Economy
        Network -- Dispatches messages to --> Criminality
        Network -- Dispatches messages to --> Leaderboard

        Wallet -- Calls --> Utils
        Admin -- Calls --> Utils
        Economy -- Calls --> Utils
        Criminality -- Calls --> Utils
        Leaderboard -- Calls --> Utils
        Game -- Calls --> Utils

        Game -- Calls --> CollectiveAI

        UI -- Calls --> Audio
        AppJS -- Calls --> Audio

        Game -- Calls --> Particles
    end

    subgraph Go Backend
        Backend(Go Backend Server)
    end

    subgraph WASM Interface
        BRIDGE_W(./bridge_service.go)
    end

    subgraph WASM Replay Kernel
        REPLAY(ClientReplayEngine)
        REPLAY -- Frame Gap --> REDIRECT(RedirectManager)
        REDIRECT -- Freeze UI --> UI
    end

    AppJS -- Primes Engine --> WASM_ENGINE
    AppJS -- Calls Hook --> BRIDGE_W
    BRIDGE_W -- Invokes Logic --> WASM_ENGINE

    WASM_ENGINE -- window.syncUI --> AppJS
    WASM_ENGINE -- triggerCaptureParticles --> Particles
    AppJS -- window.GetGameState --> WASM_ENGINE
    
    Network -- Auth Frame --> REPLAY
    REPLAY -- Catch-up Move --> WASM_ENGINE

    Network -- WebSocket --> Backend
    Admin -- HTTP API --> Backend

## 4. Authoritative Core Topology (Go - `./*.go`)
graph TD
    subgraph Entry & Routing
        SVR(./server.go)
        ONBOARD(./onboarding_service.go)
    end

    subgraph Core Orchestration
        LOBBY(./lobby_manager.go) %% The Mutex Gate
        TYPES(./common_types.go)
        REG(./shop_registry.go)
        BOOT(./economy_bootstrap.go)
        RESIL(./resilience_utils.go)
    end

    subgraph Reconciliation & Metrics
        AUDIT(./economy_audit.go)
        TELEM(./economy_telemetry.go)
        ROUTER_P(./economy_processing.go)
    end

    subgraph Battle Systems
        BATTLE(./battle_service.go) %% Authoritative PvP Validation
        ITEM(./item_service.go)
        PLAYER_S(./player_service.go)
        ACHIEVE(./achievement_service.go)
    end

    subgraph Industrial Economy
        ECON_S(./economy_service.go)
        FAUCET(./faucet_service.go)
        MARKET(./market_service.go)
        AUCTION(./auction_service.go)
        LOAN(./loan_service.go) %% Collateralized Debt Market
        CAREER(./career.go)
        BLACK(./black_market_service.go)
    end

    subgraph Criminality & Social
        CLUB(./club_service.go)
        CRIM(./handlers_criminality.go)
        RUMOR(./handlers_rumor.go)
        COURT(./courthouse_service.go)
        EMPLOY(./employment_service.go) %% Staffing & Salaries
        NARRATIVE(./narrative_service.go)
    end

    subgraph Infrastructure
        ORACLE(./oracle_service.go)
        ADMIN(./handlers_admin.go)
        TOURN(./tournament_manager.go)
    end

    SVR -- Initializes --> LOBBY
    SVR -- Payout API --> ONBOARD %% Sybil-Protected Bridge
    SVR -- Admin API --> ADMIN
    SVR -- Faucet API --> FAUCET

    SVR -- Hydrate --> BOOT
    BOOT -- Restore --> LOBBY
    LOBBY -- Game Loop --> BATTLE
    LOBBY -- Node Failover --> RESIL
    LOBBY -- Tournament Loop --> TOURN
    LOBBY -- WS Protocol --> CLUB
    LOBBY -- WS Protocol --> CRIM
    LOBBY -- Item Effects --> ITEM
    LOBBY -- Market Liquidation --> BLACK
    LOBBY -- Salary Dispenser --> CAREER
    LOBBY -- Behavioral Narrative --> NARRATIVE

    LOBBY -- Track --> ROUTER_P
    ROUTER_P -- Invariant --> AUDIT
    AUDIT -- Scrape --> TELEM

    BATTLE -- AuthoritativeFrame Sync --> WASM_ENGINE
    BATTLE -- SCAR Persistence --> TYPES
    BATTLE -- Effective Stats --> PLAYER_S
    
    BLACK -- Fencing Payouts --> MARKET
    ECON_S -- Dynamic Scaling --> FAUCET
    ROUTER_P -- Atomic Routing --> LOBBY

    FAUCET -- Signature Auth --> ORACLE
    
    CLUB -- Revenue --> ECON_P
    CLUB -- Staffing --> EMPLOY
    CRIM -- Fines --> COURT
    COURT -- Redistribution --> CLUB
    
    ORACLE -- Indexer Data --> TOURN
    ORACLE -- NFT Metadata --> TYPES
    ONBOARD -- Verifies Holder --> ORACLE
    
    ADMIN -- Asset Recovery --> CLUB
    ADMIN -- Community Monitoring --> LOBBY
    MARKET -- Context --> NARRATIVE
    CRIM -- Underworld Contracts --> BLACK

    %% New Game Layers Integration (Architectural Flow)
    CRIM -- Underworld Roles --> BLACK
    CRIM -- Bounty Hunting --> ADMIN
    ITEM -- Specialized Gear --> CRIM
    ITEM -- Specialized Gear --> ADMIN
    ECON_S -- Commission Setting --> ADMIN
    COURT -- Judicial Authority --> ADMIN
    FAUCET -- Recruitment Funds --> ADMIN
    PLAYER_S -- Defensive Buffs --> BATTLE

## 7. Redundant & Protected Files

### Redundant Artifacts
The following entries in `./AI-Brain/DIR.md` are metadata artifacts and do not represent physical files:
*   `distribution validation`: Notes regarding `./club_service_test.go`.
*   `stress tests`: Notes regarding `./market_service_test.go`.
*   `A: Single_player_Fanfare_Characters` (and other Headers): Organizational markers.

### Protected Placeholders
These files may appear orphaned but are required for future architectural phases:
*   `./bridge_service.go`: **WASM IPC Bridge (Active — Complete)**. Full WASM-to-Browser translation registry with 85 registered hooks (`js.Global().Set`), connects Go WASM to browser JS, exports: connectWallet, GetGameState, PlaceCard, SyncFullProfile. No longer empty — migrated from placeholder status per Docbase-Analysis.md code audit.
*   `./isPerishable` flag: **Phase 5 - Consumable Mechanics**. A logical hook within organization revenue splits.

### Standard Configurations
*   `./jsconfig.json`: Required for JS Module path resolution in the IDE.
*   `./.gitattributes`: Enforces LF line endings for cross-platform Dual-Target build stability.
*   `./render.yaml`: Authoritative blueprint for persistent volume mounting on the hosting provider.


## 5. UI-File-sys-Flow

The UI of Virtualbabes Arena is built with a strong emphasis on a "neon-glass" aesthetic, dynamic content, and responsiveness. It leverages a combination of static assets (images, videos), structural HTML, and a highly modular SCSS architecture to deliver an immersive user experience.

#### A: Card_images (`Public\Assets\Images\Cards\*.webp`)

*   **Purpose**: These `.webp` image files serve as the visual representation of the collectible game cards. Each file corresponds to a unique card character in the game, displaying their artwork.
*   **Flow**: These images are loaded by the browser as `<img>` tags or as `background-image` properties for HTML elements. Their specific paths are determined dynamically by the client-side JavaScript (`deck.js`, `game.js`) based on card IDs or metadata received from the Go WASM engine or the backend.
*   **Hierarchy**: These are low-level static visual assets, forming the core visual identity of the game's primary interactive elements (the cards).
*   **Synergy**:
    *   **`Public/app.js`, `Public/js/game.js`, `Public/js/deck.js`**: These JavaScript modules are responsible for dynamically creating and updating the HTML elements that display these card images (e.g., in the player's hand, on the game board, or in the deck manager). They construct the image `src` attributes using these file paths.
    *   **`Public/main.wasm` (Go WASM Engine)**: The WASM engine holds the game state, including which cards are in a player's hand or on the board. It provides card metadata (like card ID) to JavaScript, which then maps to the correct image file.
    *   **`Public/src/scss/components/_cards.scss`**: This SCSS file defines the visual styling for how these card images are presented, including their dimensions, borders, shadows, and animations (e.g., `.playing-card`, `.card-mini`).
    *   **Backend (`oracle_service.go`)**: Fetches and provides metadata for these cards (including their image URLs) from blockchain indexers, ensuring that the client displays authenticated assets.

#### B: Fan_fare_Avatars (`Public\Assets\Images\portraits\*\*.mp4`, `*.webp`, `*.png`)

*   **Purpose**: This collection provides visual assets for player avatars and NPC portraits. It includes both static (`.webp`, `.png`) and animated (`.mp4`) formats to offer dynamic and expressive character representations. The different subdirectories (`Boss`, `cute`, `Lady`, `Mini-Boss`, `Witch`) categorize avatars by character type or role.
*   **Flow**:
    *   Static images (`.webp`, `.png`): Loaded into `<img>` tags for display in various UI components (e.g., player profiles, leaderboards).
    *   Animated videos (`.mp4`): Loaded into `<video>` tags, typically configured for looping and autoplay, to provide dynamic flair for key characters or player selections.
    *   The `deck.js` module specifically handles the selection, preview, and cropping of these avatars during player setup.
*   **Hierarchy**: These are static/animated visual assets representing player and NPC identities, used across various UI screens.
*   **Synergy**:
    *   **`Public/app.js`, `Public/js/ui.js`, `Public/js/deck.js`**: These modules dynamically render avatars in elements like `#p1-avatar`, `#p2-avatar`, and the avatar selection grid in the setup overlay. `deck.js` manages the interactive cropping and selection process, potentially sending the chosen avatar URL to the backend.
    *   **`Public/main.wasm` (Go WASM Engine)**: Stores the player's selected avatar URL as part of their profile, which JavaScript retrieves for display.
    *   **`Public/src/scss/layouts/_dashboard.scss` (specifically `.avatar-frame`)**: Styles the display of avatars, including their circular frames, borders, and sizes.
    *   **Backend (`oracle_service.go`, `deck.go` - if avatar registration is a backend call)**: Stores and retrieves the selected avatar URLs, and may handle the storage of custom-cropped avatars.

#### C: Textures (`Public\Assets\Textures\*.png`)

*   **Purpose**: These `.png` files provide background textures for the game arena, allowing the visual theme of the battleground to change dynamically based on the match context (e.g., standard, challenge, tournament).
*   **Flow**: These images are typically set as `background-image` properties for specific HTML elements (e.g., the game board container). The choice of texture is dynamic, based on the current game mode.
*   **Hierarchy**: Background visual assets, providing environmental context.
*   **Synergy**:
    *   **`Public/app.js`, `Public/js/ui.js`**: The `ui.js` module (specifically the `updateDynamicArenaFloor` function, as indicated by its import in `ui.js`) is responsible for dynamically changing the `background-image` of the game board element in `index.html` based on the match type.
    *   **`Public/src/scss/layouts/_dashboard.scss`**: May define base styling for the arena floor element, which is then overridden or augmented by JavaScript to apply specific textures.

#### D: UI_filesys

This category encompasses the core structural and styling files that define the entire frontend user interface.

*   **`Public\index.html`**
    *   **Purpose**: This is the single entry point for the Virtualbabes Arena web application. It defines the fundamental HTML structure, loads all essential scripts (WASM runtime, main JavaScript application, blockchain SDKs), and links the primary stylesheet. It contains static UI elements and placeholders (`div`s with IDs) where dynamic content will be injected by JavaScript.
    *   **Flow**: The browser first loads this file. It then sequentially loads `styles.css`, `wasm_exec.js`, `app.js`, and various external SDKs (Buffer, Algorand SDK, WalletConnect). It also contains inline `onclick` event handlers that trigger functions defined in `app.js`.
    *   **Hierarchy**: The root of the entire client-side application's DOM structure. All other UI components and scripts are loaded into or interact with elements defined here.
    *   **Synergy**:
        *   **`Public/app.js`**: The primary JavaScript file that manipulates the DOM elements defined in `index.html`. It populates dynamic content, attaches event listeners, and controls the visibility of various sections and overlays.
        *   **`Public/styles.css`**: Provides the visual styling for all elements within `index.html`.
        *   **`Public/wasm_exec.js`**: Loaded by `index.html` to enable the execution of the Go WASM game engine.
        *   **`Public/js/wallet.js`, `Public/js/leaderboard.js`, `Public/js/deck.js`, `Public/js/economy.js`, `Public/js/criminality.js`, `Public/js/admin.js`, `Public/js/ui.js`**: These modules contain functions that directly interact with specific DOM elements (e.g., buttons, input fields, display areas) defined in `index.html` to render data, handle user input, and manage UI state.
        *   **`Public/Assets/Images/portraits/*.svg` (inline in `wallet-selector-overlay`)**: The SVG data for wallet icons is directly embedded in the HTML, providing immediate visual feedback for wallet options.

*   **`Public\styles.css`**
    *   **Purpose**: This is the compiled CSS file that applies all the visual styling to the `index.html`. It's generated from the SCSS source files.
    *   **Flow**: Loaded by `index.html` early in the page load process, ensuring that styles are applied before JavaScript renders dynamic content.
    *   **Hierarchy**: The final output of the SCSS pre-processing, directly consumed by the browser.
    *   **Synergy**:
        *   **`Public/index.html`**: The target for all its styles.
        *   **`Public/src/scss/*.scss`**: Its source code. Any changes to SCSS files are compiled into this `styles.css`.
        *   **`Public/app.js`, `Public/js/ui.js`**: These JavaScript files might dynamically add or remove CSS classes (e.g., `hidden`, `active`, `error`, `success`) to HTML elements, which then trigger styles defined in `styles.css`.

*   **`Public\src\scss\main.scss`**
    *   **Purpose**: This is the main entry point for the SCSS compilation process. It imports all other SCSS partials, organizing them into a logical structure.
    *   **Flow**: A SCSS pre-processor reads `main.scss`, resolves all `@import` statements, and compiles the entire stylesheet into a single `Public/styles.css` file.
    *   **Hierarchy**: The root of the SCSS architecture, defining the order in which styles are processed.
    *   **Synergy**:
        *   **All other `Public/src/scss/*.scss` files**: It imports them, bringing all styling rules together.
        *   **Build process (e.g., `npm run build-css` or similar script)**: This file is the input for the SCSS compiler.

*   **`Public\src\scss\base\_reset.scss`**
    *   **Purpose**: Provides a CSS reset to ensure consistent styling across different browsers and establishes fundamental base styles for common HTML elements (e.g., `body`, `h1-h6`, `p`, `a`, `ul`, `ol`, `button`, `img`, `table`). It also defines custom scrollbar styles for Webkit browsers.
    *   **Flow**: Imported by `main.scss` early in the compilation process to apply foundational styles before more specific component or layout styles.
    *   **Hierarchy**: Base-level styling, affecting the entire document.
    *   **Synergy**:
        *   **`Public/index.html`**: Sets the default appearance for all raw HTML elements.
        *   **`Public/src/scss/base/_variables.scss`**: Utilizes variables like `$font-body`, `$color-text-main`, `$spacing-md`, `$border-radius-md` for consistent theming.

*   **`Public\src\scss\base\_typography.scss`**
    *   **Purpose**: Defines specific typographic styles, including font families, sizes, weights, colors, and text transformations, with a focus on the "neon-glass" aesthetic. It includes utility classes for common text styles and responsive adjustments.
    *   **Flow**: Imported by `main.scss` after `_reset.scss` to apply specific text styling rules.
    *   **Hierarchy**: Base-level styling, focusing on text presentation.
    *   **Synergy**:
        *   **`Public/index.html`**: Styles headings, paragraphs, and other text content.
        *   **`Public/src/scss/base/_variables.scss`**: Heavily relies on variables for `$font-heading`, `$font-body`, `$color-neon-cyan`, `$font-size-xl`, etc.
        *   **`Public/app.js`, `Public/js/ui.js`**: JavaScript might dynamically add utility classes (e.g., `text-neon-green`, `font-bold`) to text elements.

*   **`Public\src\scss\base\_variables.scss`**
    *   **Purpose**: Acts as the central repository for all design tokens and global constants, including color palettes, font definitions, spacing scales, border radii, shadows, z-indices, transitions, and breakpoints. It also defines component-specific variables like avatar and card sizes.
    *   **Flow**: Imported by almost all other SCSS files. It must be imported first within any file that uses its variables.
    *   **Hierarchy**: The absolute foundation of the visual design system.
    *   **Synergy**:
        *   **All other `Public/src/scss/*.scss` files**: Provides consistent values for styling throughout the application.
        *   **`Public/app.js`**: Dynamically sets CSS variables like `--arena-mood-color` based on game state, which are then used in SCSS.

*   **`Public\src\scss\components\_buttons.scss`**
    *   **Purpose**: Defines the styling for all interactive buttons in the application, including base styles, primary glowing buttons, outline buttons, secondary buttons, and specific styles for success, danger, and warning actions. It also includes size variants and styles for wallet connection options.
    *   **Flow**: Imported by `main.scss` to apply styling to button elements.
    *   **Hierarchy**: Component-level styling, specific to buttons.
    *   **Synergy**:
        *   **`Public/index.html`**: Buttons defined in the HTML (`<button>`) will automatically pick up these styles.
        *   **`Public/app.js`, `Public/js/ui.js`, `Public/js/wallet.js`**: JavaScript controls button states (e.g., `disabled`, adding/removing classes like `active`, `loading`) which are styled here.
        *   **`Public/src/scss/base/_variables.scss`**: Uses color variables (`$color-neon-purple`, `$color-error-red`), spacing (`$spacing-md`), and border radii (`$border-radius-md`).
        *   **`Public/src/scss/themes/_neon-glass.scss`**: The `.wallet-option` uses the `neon-glass-panel` mixin.

*   **`Public\src\scss\components\_cards.scss`**
    *   **Purpose**: Styles the visual presentation of game cards, including their dimensions, appearance, rarity indicators, type icons, debuff badges, and interactive states (hover, selected, disabled). It also defines styles for card grids and tooltips.
    *   **Flow**: Imported by `main.scss` to style card elements.
    *   **Hierarchy**: Component-level styling, specific to game cards.
    *   **Synergy**:
        *   **`Public/app.js`, `Public/js/game.js`, `Public/js/deck.js`**: These JavaScript modules dynamically generate the HTML for cards using `renderCardHTML` and apply classes (e.g., `selected-card`, `disabled`, `common`, `rare`, `epic`, `legendary`) that are styled here.
        *   **`Public/main.wasm` (Go WASM Engine)**: Provides card data (power, rarity, mood, artifact) that JavaScript uses to determine which classes to apply.
        *   **`Public/src/scss/base/_variables.scss`**: Uses `$card-width`, `$card-height`, `$color-neon-cyan`, `$glass-blur`, etc.
        *   **`Public/src/scss/utilities/_animations.scss`**: Defines keyframe animations (`card-enter`, `card-exit`, `card-flip`) used for card transitions.

*   **`Public\src\scss\components\_overlays.scss`**
    *   **Purpose**: Provides generic and specific styling for all modal overlays and pop-up windows in the application (e.g., settings, wallet selector, deck manager, admin panel, tournament bracket, match preview, kidnap gambit, Hall of Fame). It includes base overlay styles, content containers, headers, bodies, and footers.
    *   **Flow**: Imported by `main.scss` to style overlay elements.
    *   **Hierarchy**: Component-level styling, specific to overlays.
    *   **Synergy**:
        *   **`Public/index.html`**: Defines the base HTML structure for all overlays (e.g., `<div class="overlay hidden">`).
        *   **`Public/app.js`, `Public/js/ui.js`, and other feature-specific JS files**: JavaScript controls the visibility of these overlays by adding/removing the `hidden` class. It also dynamically populates their content.
        *   **`Public/src/scss/base/_variables.scss`**: Uses `$z-index-modal`, `$glass-blur`, `$spacing-lg`, `$border-radius-xl`, etc.
        *   **`Public/src/scss/themes/_neon-glass.scss`**: Heavily uses the `neon-glass-panel` mixin for the distinctive UI aesthetic.
        *   **`Public/src/scss/utilities/_animations.scss`**: Defines animations like `animate-modal` for overlay transitions.

*   **`Public\src\scss\features\_criminality.scss`**
    *   **Purpose**: Styles the UI elements related to the game's criminality features, such as the criminality panel, heist actions, target selection, risk assessment, and results display. It emphasizes a red/orange color palette to convey danger and warning.
    *   **Flow**: Imported by `main.scss` to style criminality-specific UI.
    *   **Hierarchy**: Feature-specific styling.
    *   **Synergy**:
        *   **`Public/js/criminality.js`**: This JavaScript module dynamically generates and manipulates the HTML for criminality features, applying classes and IDs that are styled here.
        *   **`Public/app.js`**: Calls functions in `criminality.js` to open and manage these overlays.
        *   **`Public/src/scss/base/_variables.scss`**: Uses `$color-error-red`, `$color-warning-orange`, `$color-neon-purple`, etc.
        *   **`Public/src/scss/themes/_neon-glass.scss`**: Applies the `neon-glass-panel` mixin.
        *   **`Public/src/scss/utilities/_animations.scss`**: Defines animations like `progress-shine` and `risk-pulse`.

*   **`Public\src\scss\features\_economy.scss`**
    *   **Purpose**: Styles the UI elements for economic features, including the economy panel, market ticker, auction gallery, second-hand store (loans), and black market. It uses green and cyan tones to represent wealth and digital interfaces.
    *   **Flow**: Imported by `main.scss` to style economy-specific UI.
    *   **Hierarchy**: Feature-specific styling.
    *   **Synergy**:
        *   **`Public/js/economy.js`**: This JavaScript module dynamically generates and manipulates the HTML for economic features, applying classes and IDs that are styled here.
        *   **`Public/app.js`**: Calls functions in `economy.js` to open and manage these overlays.
        *   **`Public/src/scss/base/_variables.scss`**: Uses `$color-neon-green`, `$color-neon-cyan`, `$color-neon-purple`, etc.
        *   **`Public/src/scss/themes/_neon-glass.scss`**: Applies the `neon-glass-panel` mixin.
        *   **`Public/src/scss/utilities/_animations.scss`**: Defines animations like `ticker-scroll`.

*   **`Public\src\scss\features\_shops.scss`**
    *   **Purpose**: Styles the UI for district shops, including shop panels, categories, item grids, filters, shopping cart, and special offers. It often uses purple and cyan tones.
    *   **Flow**: Imported by `main.scss` to style shop-specific UI.
    *   **Hierarchy**: Feature-specific styling.
    *   **Synergy**:
        *   **`Public/js/economy.js`**: This JavaScript module (specifically `openShopsOverlay`, `switchShopCategory`, `buyClubItem`) dynamically generates and manipulates the HTML for shop features, applying classes and IDs that are styled here.
        *   **`Public/app.js`**: Calls functions in `economy.js` to open and manage these overlays.
        *   **`Public/src/scss/base/_variables.scss`**: Uses `$color-neon-purple`, `$color-neon-cyan`, `$color-gold`, etc.
        *   **`Public/src/scss/themes/_neon-glass.scss`**: Applies the `neon-glass-panel` mixin.

*   **`Public\src\scss\features\_social.scss`**
    *   **Purpose**: Styles the UI for social features, including the social panel, achievement system (trophies), career paths, entity portfolio, and social network connections. It uses a mix of gold, blue, and pink tones.
    *   **Flow**: Imported by `main.scss` to style social-specific UI.
    *   **Hierarchy**: Feature-specific styling.
    *   **Synergy**:
        *   **`Public/js/criminality.js` (for `openSocialPanelOverlay`, `switchSocialTab`)**: This JavaScript module dynamically generates and manipulates the HTML for social features, applying classes and IDs that are styled here.
        *   **`Public/app.js`**: Calls functions in `criminality.js` to open and manage these overlays.
        *   **`Public/src/scss/base/_variables.scss`**: Uses `$color-gold`, `$color-neon-blue`, `$color-neon-pink`, etc.
        *   **`Public/src/scss/themes/_neon-glass.scss`**: Applies the `neon-glass-panel` mixin.
        *   **`Public/src/scss/utilities/_animations.scss`**: Defines animations like `badge-glow`.

*   **`Public\src\scss\features\_territory.scss`**
    *   **Purpose**: Styles the UI for territory management, including the territory panel, 3D world map, districts, club foundry, regional governor status, and conflicts. It uses purple, cyan, and blue tones for a futuristic, strategic feel.
    *   **Flow**: Imported by `main.scss` to style territory-specific UI.
    *   **Hierarchy**: Feature-specific styling.
    *   **Synergy**:
        *   **`Public/app.js` (for `openTerritoryMapOverlay`, `adjustMapZoom`)**: This JavaScript module dynamically generates and manipulates the HTML for territory features, applying classes and IDs that are styled here.
        *   **`Public/js/economy.js` (for `openClubFoundry`, `submitClubFoundry`, `openTerritoryView`)**: These functions interact with the territory UI.
        *   **`Public/src/scss/base/_variables.scss`**: Uses `$color-neon-purple`, `$color-neon-cyan`, `$color-neon-blue`, etc.
        *   **`Public/src/scss/themes/_neon-glass.scss`**: Applies the `neon-glass-panel` mixin.
        *   **`Public/src/scss/utilities/_animations.scss`**: Defines animations like `contested-pulse`.

*   **`Public\src\scss\layouts\_dashboard.scss`**
    *   **Purpose**: Defines the layout and styling for the main game dashboard, including the overall container, columns, player lists, chat interface, matchmaking box, cooldown displays, tournament banners, action bars, and match history.
    *   **Flow**: Imported by `main.scss` to structure the primary game screen.
    *   **Hierarchy**: Layout-level styling, organizing major UI sections.
    *   **Synergy**:
        *   **`Public/index.html`**: Provides the structural `div`s (e.g., `.dashboard`, `.column`) that these styles target.
        *   **`Public/app.js`, `Public/js/game.js`, `Public/js/ui.js`**: JavaScript dynamically populates content within these layout elements (e.g., player names, chat messages, history items) and controls their visibility.
        *   **`Public/src/scss/base/_variables.scss`**: Uses `$spacing-lg`, `$border-radius-lg`, `$color-neon-cyan`, etc.
        *   **`Public/src/scss/themes/_neon-glass.scss`**: Applies the `neon-glass-panel` mixin to various dashboard elements.

*   **`Public\src\scss\layouts\_main-layout.scss`**
    *   **Purpose**: Defines the overarching layout for the entire application, including the main game container and the top navigation bar.
    *   **Flow**: Imported by `main.scss` to establish the highest-level structural styles.
    *   **Hierarchy**: Global layout styling.
    *   **Synergy**:
        *   **`Public/index.html`**: Provides the root layout elements (`.main-game-container`, `.top-bar`).
        *   **`Public/app.js`, `Public/js/ui.js`**: JavaScript controls elements within this layout (e.g., wallet connection button, maintenance bar visibility).
        *   **`Public/src/scss/base/_variables.scss`**: Uses `$spacing-md`, `$glass-border-color`, `$glass-blur`, etc.
        *   **`Public/src/scss/themes/_neon-glass.scss`**: Applies the `neon-glass-panel` mixin to elements like status widgets.

*   **`Public\src\scss\themes\_neon-glass.scss`**
    *   **Purpose**: Defines the core "neon-glass" aesthetic through a reusable mixin (`neon-glass-panel`) and applies it to base elements. It also includes a mixin for neon text glow.
    *   **Flow**: Imported by `main.scss` and then explicitly `@include`d by other component and feature SCSS files to apply the glassmorphism effect.
    *   **Hierarchy**: Thematic styling, providing a consistent visual language.
    *   **Synergy**:
        *   **`Public/src/scss/base/_variables.scss`**: Relies entirely on variables like `$glass-bg-color`, `$glass-border-color`, `$glass-blur`, `$glass-shadow` to define the glassmorphism properties.
        *   **All other component/feature SCSS files**: Consumes the `neon-glass-panel` mixin to apply the theme.

*   **`Public\src\scss\utilities\_animations.scss`**
    *   **Purpose**: Provides a comprehensive set of CSS animations and keyframe definitions for various UI effects (fade, slide, scale, bounce, pulse, glow, shimmer, float, spin). It also includes utility classes for applying these animations and controlling their properties (duration, delay, fill mode).
    *   **Flow**: Imported by `main.scss` to make animation classes available globally.
    *   **Hierarchy**: Utility-level styling, providing reusable animation effects.
    *   **Synergy**:
        *   **`Public/app.js`, `Public/js/ui.js`, `Public/js/particles.js`**: JavaScript dynamically adds/removes animation classes to trigger visual effects (e.g., `animate-modal`, `animate-capture-burst`).
        *   **`Public/src/scss/base/_variables.scss`**: Uses `$transition-base`, `$color-neon-cyan`, etc. for animation properties.

*   **`Public\src\scss\utilities\_spacing.scss`**
    *   **Purpose**: Provides a utility-first approach for common layout and spacing properties, including display types, flexbox, grid, margin, padding, gap, width, height, position, z-index, overflow, text alignment, font properties, opacity, visibility, borders, shadows, cursors, object-fit, transforms, and transitions. It also includes responsive utilities.
    *   **Flow**: Imported by `main.scss` to provide a wide range of atomic utility classes.
    *   **Hierarchy**: Utility-level styling, offering granular control over layout and appearance.
    *   **Synergy**:
        *   **`Public/index.html`**: HTML elements are directly annotated with these utility classes (e.g., `flex-row`, `gap-15`, `mb-20`, `w-full`) to define their layout and spacing.
        *   **`Public/src/scss/base/_variables.scss`**: Relies on the `$spacing-scale` and other variables for consistent sizing.

## 6. UI-Map
graph TD
    subgraph "SCSS Source (Modular Partials)"
        VAR[base/_variables.scss] --> BASE
        VAR --> COMP
        VAR --> FEAT
        VAR --> LAY
        VAR --> UTIL
        VAR --> THEME

        subgraph "Base Layer"
            BASE(./Public/src/scss/base/_reset.scss<br/>./Public/src/scss/base/_typography.scss)
        end

        subgraph "Thematic Mixins"
            THEME(./Public/src/scss/themes/_neon-glass.scss)
        end

        subgraph "Component Layer"
            COMP(./Public/src/scss/components/_buttons.scss<br/>./Public/src/scss/components/_cards.scss<br/>./Public/src/scss/components/_overlays.scss)
        end

        subgraph "Feature Specifics"
            FEAT(./Public/src/scss/features/_criminality.scss<br/>./Public/src/scss/features/_economy.scss<br/>./Public/src/scss/features/_shops.scss<br/>./Public/src/scss/features/_social.scss<br/>./Public/src/scss/features/_territory.scss)
        end

        subgraph "Structural Layouts"
            LAY(./Public/src/scss/layouts/_dashboard.scss<br/>./Public/src/scss/layouts/_main-layout.scss)
        end

        subgraph "Utilities & Anim"
            UTIL(./Public/src/scss/utilities/_animations.scss<br/>./Public/src/scss/utilities/_spacing.scss)
        end

        THEME -.->|@mixin neon-glass-panel| COMP
        THEME -.->|@mixin neon-glass-panel| FEAT
        THEME -.->|@mixin neon-glass-panel| LAY
    end

    %% Aggregation
    BASE --> MAIN(./Public/src/scss/main.scss)
    COMP --> MAIN
    FEAT --> MAIN
    LAY --> MAIN
    THEME --> MAIN
    UTIL --> MAIN

    MAIN -- "Sass Compiler" --> CSS(./Public/styles.css)

## 6. Detailed Backend Service Topology
The backend utilizes an **Authoritative Core with Stateless Service Delegation** model:

*   **Switchboard**: `./server.go` / `./Public/js/network.js`.
*   **Persistence**: `./economy_bootstrap.go` (Recovery) and `./economy_persistence.go` (Snapshots).
*   **Orchestration**: `./lobby_manager.go` (The Mutex Gate).
*   **Specialized Systems**:
    *   `./battle_service.go`: Authoritative combat validation.
    *   `./club_service.go`: Organizational management.
    *   `./economy_processing.go`: The **Token-Sink Router** (Atomic Fee Redistribution).
    *   `./economy_audit.go`: The **Reconciliation Kernel**.
    *   `./economy_telemetry.go`: The **Prometheus Exporter**.
*   **Isomorphic Bridge**: `./common_types.go` (Shared Data) and `./backend_types.go` (Server Containers).

## 7. Operational Integrity: File Registry

### 7.1 Redundant Artifacts
| Entry | Status | Reason |
| :--- | :--- | :--- |
| `distribution validation` | Metadata | Notes regarding `./club_service_test.go` logic. |
| `stress tests` | Metadata | Notes regarding `./market_service_test.go` logic. |
| `A: Single_player_Fanfare_Characters` | Metadata | Header artifact in `./AI-Brain/DIR.md`. |

### 7.2 Strategic Kernels
| File / Placeholder | Role | Reason |
| :--- | :--- | :--- |
| `./bridge_service.go` | Integration | Authoritative IPC Registry for WASM (Pillar 4). |
| `./season.json` | Fallback | Local state root backup if blockchain reconstruction fails. |
| `./jsconfig.json` | Config | Required for JS Module path resolution in the IDE. |
| `./render.yaml` | Config | Authoritative blueprint for persistent volume mounting. |
| `redemption_gateway.go` | Integration | Active Entitlement Gateway for Phase 4 Console Expansion. |
| `./Dockerfile` | Deployment | Containerization blueprint for production build. |
| `./entrypoint.sh` | Deployment | Container startup script (env injection, health checks). |
| `./.dockerignore` | Deployment | Excludes artifacts from Docker build context. |
| `./deploy-wasm.yml` | CI/CD | WASM rebuild & deploy pipeline configuration. |
| `./networks.json` | Config | Network endpoint registry (chain RPCs, testnet/mainnet). |
| `./.env.example` | Config | Environment variable template for local/production setup. |
| `./ai_exclude` | DevTool | File patterns excluded from AI indexing/exploration. |

## 8. Core Economic Sequences

### 8.1 Observability & Integrity
1. `./economy_service.go` triggers a tax event via the **Token-Sink Router**.
2. `./economy_processing.go` calculates atomic splits.
3. `./economy_audit.go` performs an invariant check (**Inflow == Outflow + Remainder**).
4. Discrepancies raise a security exception and block the transaction.
5. Vetted data is forwarded to `./economy_telemetry.go` for Prometheus scraping (9090).

### 8.2 Market Volatility
1. `./Public/js/criminality.js` triggers `spreadRumor` via WebSocket.
2. `./handlers_rumor.go` validates the fee and acquires the `Lobby` lock.
3. Reputation is recalculated for the target, forcing a price update in `./market_service.go`.
4. `./market_service.go` applies the rumor strength to the AMM bonding curve.
5. `./lobby_manager.go` eventually clears the effect via the `processRumors` loop.

<!-- ========================================================================= -->
<!--                    [UNIMPLEMENTED CONSOLE EXPANSION SPEC]                  -->
<!-- ========================================================================= -->

## 9. Future Cross-Platform Console Expansion Map (Store-Mirrored DLC Model)
This module outlines the architectural blueprints for the upcoming console deployment phase. To satisfy platform policies, the console runtime is **strictly NON-crypto active**.

### 9.1 The Entitlement & Redemption Pipeline
Console players redeem **earned Arena Vouchers** for in-game DLC. The console store UI functions exclusively as a display/presentation layer for entitlements granted by the Go Backend.

[ Console Hardware Client ]
│
▼ (1. Redemption Event: Voucher → DLC Blueprint)
[ redemption_gateway.go ]
│
├──► [ 2. Backend Entitlement Validation ]
│    ├── Check ArenaVoucher Balance
│    └── Verify Product Entitlement with Manufacturer API (Xbox/PSN)
│
▼ (3. Deterministic Settlement)
[ economy_processing.go / economy_audit.go ]
│
├───► [ 4A. Browser Creator Payout ] ──► Increments $VBV (Liability Shift)
├───► [ 4B. Vault Reconciliation ] ────► Physical VBV returns to Faucet
└───► [ 4C. Voucher Deduction ] ───────► Decrements ArenaVouchers
│
▼ (5. Final Entitlement Grant)
[ Console Store API ] ──► Marks DLC as "Purchased/Unlocked" (0.00 Price)

### 9.2 Browser Players as DLC Suppliers
1. **Minting**: Creator mints a non-crypto asset blueprint on the Voi Network.
2. **Registration**: Server indexes the blueprint and mirrors it to the Console Hub.
3. **Market Buy**: When a console player redeems vouchers, the server executes a market-buy of $VBV via the **Nautilus DEX Path** (Placeholder: Future specialized service) to pay the creator.

### 9.3 Cross-Platform Operational Directives
1. **Compliance Invariant**: The console store UI must never display a fiat price for items redeemable via vouchers. Items are marked as "Redeemable with Arena Vouchers."
2. **Non-Crypto Client**: Consoles are prohibited from blockchain interaction. All cryptographic signing and DEX swaps are performed by the **Admin Node**.
3. **State Isolation**: Consoles operate exclusively via `api/v1/`. Direct interaction with `./bridge_service.go` is prohibited.
4. **Deterministic Timelocks**: If an asset is leased, `./loan_service.go` maintains a server-side ticker. Upon expiry, the backend broadcasts an invalidation packet to lock the local DLC profile.
5. **Liquidity Preservation (USDC Siphon)**: The excessive accumulation sub-routine monitors faucet liquidity thresholds via `./economy_telemetry.go`. If optimal reserves are achieved, the siphon extracts a strict, predetermined maintenance cut from the incoming transaction payload directly into the Admin USDC operational ledger before routing liquidity onto Nautilus DEX pools.

### 9.4 Combat Sync Loop: Cross-Platform Parity

**Scenario: Console Player (CP) defeats Browser Player (BP)**
1. [ Authoritative Core ] identifies CP as Winner.
2. [ BP Wallet ] → Virtual Deduction (-X VBV) → [ Global Faucet ].
3. [ CP Profile ] → Virtual Increment (+X Arena Vouchers) → [ Server State ].
4. [ CP Hardware ] receives `update_vouchers` WebSocket frame; DLC becomes "Redeemable."

**Scenario: Browser Player (BP) defeats Console Player (CP)**
1. [ CP Profile ] → Virtual Deduction (-X Arena Credits) → [ Server State ].
2. [ Global Faucet ] → Virtual Increment (+X VBV) → [ BP Wallet ].
3. [ BP Wallet ] can now "Dispatch" those winnings on-chain via browser.

### 9.5 Authoritative Service Lifecycle
graph LR
    Init[INIT: BOOT & Hydrate] --> Run[RUN: Active Event Loop]
    Run --> Snap[SNAPSHOT: Blockchain Archival]
    Snap --> Run
    Run --> SIG[SIGTERM: Integrity Audit]
    SIG --> Exit[EXIT: Final Snapshot]

### 9.6 Non-Custodial Automated Settlement (End-Game)
1. Player provides an ARC-200 spend allowance via `./Public/js/wallet.js`.
2. `./oracle_service.go` verifies the allowance via the Indexer registry.
3. Background workers in `./loan_service.go` or `./auction_service.go` detect settlement triggers (Expiry/Win).
4. Specialized services execute `pullApprovedTokens` via the Go SDK `transferFrom` protocol.
5. `./economy_audit.go` captures the inflow and synchronizes the virtual liability ledger.

### 9.7 Multi-Chain Asset Lifecycle (End-Game)
1. `./onboarding_service.go` acts as the primary entry gate for external liquidity (Algorand Bridge).
2. Cross-chain discovery triggers metadata ingestion via the `./oracle_service.go` `MetadataDispatcher`.
3. `./lobby_manager.go` dispatches compressed `VBT_STATE_SNAPSHOT` notes to ensure archival parity.
4. The Arena Vault executes authenticated payouts across supported networks using the Switchboard Pattern.

## 10. Game Layers: Underworld vs. Justice Hegemony (Architectural View)
This section outlines how the Arena's core services are leveraged and expanded to support the specialized career paths and dynamic conflict between the Underworld and Justice factions.

### 10.1 Underworld Layer (Criminal Operations)
*   **`./black_market_service.go`**: Expanded to support "Fence" role (selling illicit goods) and "Underworld Boss" PvE encounters (exclusive item/card drops).
*   **`./handlers_criminality.go`**: Core logic for "Kidnapper" (enhanced hostage events), "Hostage Host" (team-based asset hoarding), and "Launderer" (Wanted Level reduction for a fee).
    *   **Career XP Triggers:** Kidnapper (+80 XP + raid bonus), Hostage Host (+50 XP first capture), Racketeer (+40 XP extortion), Smuggler (conditional transfer XP)
*   **`./market_service.go`**: Implements **Dutch Auction** logic for liquidated assets and **Dividend Freezes** for Tax Auditors.
*   **`./item_service.go`**: Implements "Arc-Net-Spy" (inventory/match history reveal) and "Signal Dampeners" (Ghost Protocol stealth buffs).
*   **`./narrative_service.go`**: Leveraged by "Gossip" role for enhanced rumor propagation and "Arc-Net Operative" for intelligence gathering.
*   **`./player_service.go`**: Calculates "Smuggler" bypass mechanics and "Heist Planner" success buffs.
    *   **Career XP Triggers:** Heist Planner (rivalry-based via EvaluateCrossCareerXP)
*   **`./economy_service.go`**: Manages "Lawyer-Commissioner" (illicit commission rates) and "Underworld Contracts" (new Faucet sinks).
    *   **Career XP Triggers:** Lawyer-Commissioner (+30 XP on release in courthouse_service.go)
*   **`./counterfeit_service.go`** (new): Created for Counterfeiter career with TrackCareerXP hooks.
    *   **Career XP Triggers:** Counterfeiter (per note production)

### 10.2 Justice Layer (Arena Law Enforcement)
*   **`./handlers_criminality.go`**: Core logic for "Bounty Hunter" (tracking high-Wanted targets) and "Armed-Offender-Squad (AOS)" (team-based recovery missions).
    *   **Career XP Triggers:** Bounty Hunter (+60 + targetWanted/5 per capture in battle_service.go:660), AOS Leader (+60 + teamSize×10 per raid)
*   **`./item_service.go`**: Implements "Truth Serum" (buff/debuff reveal) and "Reputation Shield" (penalty mitigation).
*   **`./handlers_admin.go`**: Provides the "Justice Tier Bounty Center Dashboard" (enhanced tracking, mission dispatch) and "Reputation Enforcement" (player flagging).
*   **`./courthouse_service.go`**: Expanded for "Judge" (rehabilitation fees, court ownership) and "Warden" (bail rates, prisoner management).
    *   **Career XP Triggers:** Warden (+40 capture + 25 detection + 15 ransom + 20 release), Tax Auditor (+15 fine processing — resolution bonus pending)
*   **`./economy_processing.go`**: Implements **Corporate Bailouts** (5,000 $VBV stimulus) and **Administrative USDC Siphons**.
*   **`./faucet_service.go`**: Manages "Justice Recruitment Packs" and "Bounty Hunter Licenses" (new Faucet sinks).
*   **`./economy_service.go`**: Manages "Justice Commissioner" (pro-social commission rates) and "Tax Auditor" (shadow funds recovery).
    *   **Career XP Triggers:** Justice Commissioner (rivalry-based via EvaluateCrossCareerXP), Tax Auditor (+15 processing — resolution bonus 30-80 pending)
*   **`./player_service.go`**: Calculates "Sector Peacekeeper" defensive buffs.
    *   **Career XP Triggers:** Sector Peacekeeper (+20 kidnap + 15 release + 15 detection)
*   **`./achievement_service.go`**: Tracks progression for "Bounty Hunter Mastery" and "Courthouse Advocate" achievements.
*   **`./black_market_service.go`**: Provides "Underworld Contracts" for criminal players.

### 10.3 Cross-Layer Career Engine
*   **`./rival_career_engine.go`**: Authority on career XP evaluation via `EvaluateCrossCareerXP`. All Justice D7-D10 careers use rivalry pairs for XP:
    *   Intel-Agent ↔ Arc-Net Operative (ally intel sharing)
    *   Justice Recruiter ↔ Mutation Log Auditor (ally mutation data)
    *   Justice Commissioner ↔ Tax Auditor (synergy override alliance)
*   **`./rivalry_handlers.go`**: Exposes rivalry state via HTTP/WebSocket.
*   **`./shop_registry.go`**: Contains item definitions for all career types including D7-D10 specialized gear.

### 10.4 Cross-Layer Interactions & UI
*   **`./Public/js/criminality.js`**: Orchestrates UI for Bounty Board, Heist Planning, Kidnap Selection, and new Justice/Underworld dashboards.
*   **`./Public/js/economy.js`**: Manages Black Market, Art Gallery, and new specialized shops for Justice/Underworld items.
*   **`./Public/js/ui.js`**: Renders specialized card archetypes (Justice/Underworld Cards), item overlays, and dynamic UI feedback for faction-specific events.
*   **`./Public/src/scss/`**: Provides distinct visual themes (e.g., neon-cyan for Justice, error-red for Underworld) for faction-aligned UI elements, cards, and dashboards.
*   **`./main.go` (WASM)**: Ensures deterministic combat logic for "Fallen Cards vs. Justice Cards" battles, applying faction-specific power bonuses and penalties.

### 10.3 Cross-Layer Interactions & UI
*   **`./Public/js/criminality.js`**: Orchestrates UI for Bounty Board, Heist Planning, Kidnap Selection, and new Justice/Underworld dashboards.
*   **`./Public/js/economy.js`**: Manages Black Market, Art Gallery, and new specialized shops for Justice/Underworld items.
*   **`./Public/js/ui.js`**: Renders specialized card archetypes (Justice/Underworld Cards), item overlays, and dynamic UI feedback for faction-specific events.
*   **`./Public/src/scss/`**: Provides distinct visual themes (e.g., neon-cyan for Justice, error-red for Underworld) for faction-aligned UI elements, cards, and dashboards.
*   **`./main.go` (WASM)**: Ensures deterministic combat logic for "Fallen Cards vs. Justice Cards" battles, applying faction-specific power bonuses and penalties.

## 11. Service Wiring Matrix (Verification Snapshot — 2026-06-25)

### 11.1 Instantiation Status (Lobby Struct Fields)

The following services are instantiated in `newLobby()`:

| # | Service | Type | File | Status |
|---|---------|------|------|--------|
| 1 | ClubService | `*ClubService` | `club_service.go` | ✅ Wired (HTTP routes via dispatch, WS paths) |
| 2 | CareerService | `*CareerService` | `career.go` | ✅ Background-only (`StartSalaryDispenser`) |
| 3 | CourthouseService | `*CourthouseService` | `courthouse_service.go` | ✅ Wired (HTTP reset route) |
| 4 | OnboardingService | `*OnboardingService` | `onboarding_service.go` | ✅ Wired (HTTP `/api/bridge/onboard`) |
| 5 | AchievementService | `*AchievementService` | `achievement_service.go` | ✅ Wired (HTTP GET/POST/UNLOCK routes) |
| 6 | OracleService | `*OracleService` | `oracle_service.go` | ℹ️ Helper only (dependency injection, no direct HTTP routes) |
| 7 | TournamentService | `*TournamentService` | `tournament_manager.go` | ✅ Wired (HTTP register/history/admin/start) |
| 8 | LoanService | `*LoanService` | `loan_service.go` | ✅ Wired (HTTP TAKE/REPAY/list routes) |
| 9 | AuctionService | `*AuctionService` | `auction_service.go` | ✅ Wired (HTTP GET/POST routes + game loop `ProcessAuctions`) |
| 10 | BlackMarketService | `*BlackMarketService` | `black_market_service.go` | ✅ Wired (5+ HTTP routes) |
| 11 | PlayerService | — | `player_service.go` | ✅ Helper methods on Lobby, used throughout |
| 12 | NarrativeService | `*NarrativeService` | `narrative_service.go` | ⚠️ ORPHANED (no HTTP routes, no WS dispatchers) |
| 13 | NautilusDEXPathService | `*NautilusDEXPathService` | `nautilus_dex_path.go` | ⚠️ PLACEHOLDER (comment: "PILLAR 2: Console Creator Payouts") |
| 14 | JusticeService | `*JusticeService` | `justice_service.go` | ℹ️ Partially wired (`/api/justice/missions` route exists) |

### 11.2 Career Engine (Not instantiated in Lobby)

| Service | File | Status |
|---------|------|--------|
| RivalCareerEngine | `rival_career_engine.go` | ✅ Authority on career XP evaluation, rivalry pairs wired |
| CounterfeitService | `counterfeit_service.go` | ⚠️ Fully orphaned (313 lines: `HandleGenerateCounterfeit`, `HandleDetectCounterfeit`, `SeizeCounterfeitNoteLocked`, `CleanupExpiredCounterfeits`; zero callers) |

### 11.3 Employment Layer (Methods on Lobby, not a service struct)

| Method | File | WS Dispatch | Status |
|--------|------|-------------|--------|
| `handleHirePlayer` | `employment_service.go:14` | ✅ `"hire_player"` case in lobby_manager.go | ✅ Partially wired |
| `handleSetSalary` | `employment_service.go:88` | ❌ No WS dispatch found | ⚠️ ORPHANED (method exists, zero callers) |
| `HandleLaunderCapital` | `employment_service.go:154` | ✅ `"launder_capital"` case in lobby_manager.go | ✅ Partially wired |

### 11.4 Wiring Summary

| Status | Count | Services |
|--------|-------|----------|
| ✅ Active & Wired | 8 | ClubService, CareerService (bg), CourthouseService, OnboardingService, AchievementService, TournamentService, LoanService, AuctionService, BlackMarketService |
| ℹ️ Helper/Utility | 2 | OracleService, PlayerService |
| ⚠️ Partially Wired | 1 | EmploymentLayer (hire_player/launder_capital yes; set_salary no), JusticeService (only missions) |
| ⚠️ Orphaned | 2 | NarrativeService (708 lines), CounterfeitService (313 lines) |
| ℹ️ Placeholder | 1 | NautilusDEXPathService |

### 11.5 Recommended Actions

1. **CounterfeitService**: Wire `HandleGenerateCounterfeit` and `HandleDetectCounterfeit` to HTTP routes in `server.go`, or document as planned/deprecated.
2. **handleSetSalary**: Add `"set_salary"` case to lobby_manager.go WS dispatch, or remove method.
3. **NarrativeService**: Audit if it should be wired, migrated to career engine, or documented as completed/deprecated.
4. **JusticeService**: Verify endpoint coverage per documented Justice D1-D10 careers.

---

> **NOTE**: Duplicate content removed (lines 723-768 originally repeated sections 9.2-9.7 with incorrect numbering). Clean end of file at line 690.
