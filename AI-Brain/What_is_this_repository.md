# VIRTUALBABES ARENA: ARCHITECTURE & CONTEXT MASTER DOCUMENT

## 1. OVERALL ARCHITECTURE & TECHNOLOGY STACK
Virtualbabes Arena is a blockchain-integrated card game platform blending real-time multiplayer mechanics with decentralized economics.

* **Backend (Go):** High-performance server (`server.go`) using WebSockets for real-time communication, HTTP APIs for RESTful endpoints, and blockchain indexers for on-chain verification. State is managed in-memory and via blockchain persistence (no traditional DB).
* **Game Engine (Go WASM):** Core logic compiled to WebAssembly (`main.go`, `main.wasm`) for deterministic gameplay (Triple Triad-inspired rules). Ensures tamper-proof client-side calculations.
* **Frontend (JavaScript + SCSS):** Modular Single-Page Application (`app.js`) utilizing WalletConnect v2 for multi-wallet support. Styled with a "neon-glass" aesthetic via modular SCSS.
* **Blockchain Integration:** Primary support for Voi (ARC-200 tokens/NFTs) and Algorand. Uses indexers for metadata, verification, and state reconstruction.
* **Security Model:** "Switchboard Pattern" (server-side signing for payouts; client-side nonce proofs to prevent replay attacks). Zero client-side private key exposure. Critical state is reconstructed from authenticated blockchain receipts.
* **Deployment:** GitHub + Render (hosting), Carrd.co (landing/status), Docker-ready environments.

---

## 2. PROJECT DOCUMENTATION
* `AI-Brain/What_is_this_repository.md`: The definitive guide to the system architecture, file roles, and the real-time interaction flow between the Go Backend, WASM Engine, and Frontend.

---

## 3. CORE COMPONENTS BREAKDOWN

### Backend Services (`*.go`)
* `server.go`: Central hub. Initializes state, handles WS upgrades, HTTP routes, rate-limiting, and concurrent clients via goroutines/mutexes.
* `lobby_manager.go`: Real-time state manager (clients, matches, tournaments). Broadcasts updates and enforces rules.
* `tournament_manager.go`: Handles 8/16-player brackets, verifies on-chain buy-ins, advances rounds, and archives results as blockchain notes.
* `battle_service.go`: Oversees match logic, PvP validation, and server-authoritative win calculations.
* `economy_service.go`: Manages global economic health, dynamic reward scaling, and on-chain note records.
* `economy_processing.go`: Orchestrates temporal loops for loan defaults and auction settlements.
* `faucet_service.go`: Handles secure $VBV reward payouts via the Switchboard Pattern and signature verification.
* `loan_service.go`: Manages the Second-Hand Store, collateralized escrow, and repayment logic.
* `auction_service.go`: Operates the Art Gallery (Auctions) and settlement distributions.
* `black_market_service.go`: Manages the liquidation and sale of defaulted collateral.
* **Industrial Loop**: Implementation of a circular economy where all protocol fees (Courthouse, Auction, Heist) return to the Faucet or Club Treasuries.
* `item_service.go`: Centralizes application of tactical buffs, vitality stims, and deployable hardware traps.
* `achievement_service.go`: Tracks/unlocks trophies; updates leaderboards.
* `courthouse_service.go`: Manages Wanted Level resets via $VBV fines; distributes fines to club treasuries.
* `handlers_admin.go`: Admin panel (ARC-14 signature auth) for network config, bans, and broadcasts.
* `handlers_public.go`: Public APIs for Carrd integration (leaderboards, status).
* `handlers_criminality.go`: Endpoint management for kidnapping gambits and heist result processing.
* `handlers_rumor.go`: Manages the intake and propagation of market-moving rumors.
* `onboarding_service.go`: Manages Sybil-protected Algorand-to-Voi onboarding ("Starter Pack" claims).
* `bridge_service.go`: Placeholder for future cross-chain asset interoperability.
* `shop_registry.go`: Defines consumable and security upgrade items.
* `common_types.go`: Centralized definitions for shared data structures (Player, Match, Club, etc.).
* `employment_service.go`: Manages hiring, staffing roles, and salary configurations.
* `career.go`: Orchestrates the daily automated salary dispenser for club employees.
* `market_service.go`: Manages the Entity Market (share trading) and observes meta-sentiments.

### Frontend Architecture (`Public/`)
* `main.go` (WASM): Game state (`Engine` struct), AI logic (weighted scoring for Hard Mode), deck building, rule enforcement, and asset pooling.
* **The Real-Time Loop**: The Backend broadcasts state updates over WebSockets; `network.js` receives these and triggers `app.js:syncUI`. `syncUI` reads the current snapshot from the WASM `GetGameState` and re-renders the DOM.
* **JS/WASM Bridge**: WASM exposes functions to the JS `window` object (e.g., `window.PlaceCard`, `window.SyncMove`) while JS provides audio and particle triggers back to the engine.
* `app.js`: Central orchestrator for UI state, WS management, and module coordination.
* **Modular Feature Scripts (`Public/js/`)**:
    * `config.js`: Centralized environment constants and dynamic blockchain asset configuration.
    * `network.js`: Manages the WebSocket connection, watchdog timers, and message dispatching.
    * `wallet.js`: Handles multi-provider wallet integration (WalletConnect, Nautilus, Kibisis).
    * `game.js`: Primary interface for combat input, matchmaking, and real-time chat coordination.
    * `deck.js`: Manages card inventory, deck construction, and interactive avatar processing.
    * `economy.js`: Implements the high-finance layer (Art Gallery, Leases, and Entity Trading).
    * `criminality.js`: Logic and UI for Heists, Kidnapping Gambits, and the Rumor Mill.
    * `leaderboard.js`: Processes tournament brackets and Hall of Fame seasonal history.
    * `admin.js`: Orchestrates administrative audits and system controls via signature auth.
    * `ui.js`: Core layout utilities, toast notifications, card rendering, and tactical tooltips.
    * `audio.js`: Global volume management and spatial audio sync with the WASM engine.
    * `particles.js`: Canvas-based visual effects system for card captures and victories.
    * `utils.js`: Generic caching logic for cross-chain Envoi names and asset symbols.
* `collective-intelligence.js`: Observes server-evaluated playstyle tendencies to generate contextual NPC taunts and chat narrative.
* `index.html`: Entry point. Embeds WASM, sets up SDKs (AlgoSDK, WalletConnect).
* **Modular SCSS Architecture (`Public/src/scss/`)**:
    * `main.scss`: The primary manifest that aggregates all partials into a single production stylesheet.
    * **Base Layer**:
        * `base/_variables.scss`: Central repository for design tokens (colors, fonts, spacing, z-index).
        * `base/_reset.scss`: Cross-browser CSS reset and mobile normalization.
        * `base/_typography.scss`: Orbitron/Rajdhani font scaling and neon text effects.
        * `base/_dashboard.scss`: Status indicators, badges, and progress bar foundations.
    * **Thematic Mixins**:
        * `themes/_neon-glass.scss`: Core glassmorphism mixins and standard neon border glows.
    * **Component Layer**:
        * `components/_buttons.scss`: Tactile :active states and selection glow logic.
        * `components/_cards.scss`: Rarity-based styling and grid-system for playing cards and miniatures.
        * `components/_overlays.scss`: Modal architecture for settings, deck management, and tournament brackets.
    * **Feature Modules**:
        * `features/_territory.scss`: Styles for the 3D territory map and Regional Governor status.
        * `features/_social.scss`: Achievement badge animations and Alliance Hub layout.
        * `features/_shops.scss`: Grid layout and category filters for District hardware stores.
        * `features/_economy.scss`: Market Ticker motion logic, Auction Gallery, and Black Market aesthetics.
        * `features/_criminality.scss`: Tactical heist planning grids and the high-risk Risk Meter.
    * **Structural Layouts**:
        * `layouts/_main-layout.scss`: Global navigation, status widgets, and the primary game container.
        * `layouts/_dashboard.scss`: Lobby chat containers, player lists, and 3x3 battle grid spacing.
    * **Utilities & Animations**:
        * `utilities/_spacing.scss`: Atomic utility classes for layout (flex, grid, margin, padding).
        * `utilities/_animations.scss`: Keyframe library for card flips, capture bursts, and screen shakes.

---

## 4. CROSS-CHAIN FUNCTIONALITY & ORACLE SERVICE
Managed via `oracle_service.go` and Wallet Linking.

### Supported Networks
* **Primary (Full Tx Support):** Voi (Main network, $VBV, Tournaments). Native NFT Standards: ARC-72 (Smart Contract), ARC-69 (Notes), ARC-19 (Reserve CID).
* **Secondary/Bridge:** Algorand ($AVoi).
* **Metadata-Only (NFT Discovery):** Ethereum (ERC-721/1155), Polygon, Solana (Metaplex DAS), Bitcoin (Ordinals), Flow, WAX.

### Mechanisms
* **Wallet Linking:** Non-AVM wallets link to the primary AVM wallet via server-side verification.
* **NFT Discovery:** Oracle queries linked wallets across chains (ARC-72, Etherscan, RPCs) to fetch and cache metadata.
* **Power Scaling:** Base power boosts are applied to cross-chain NFTs to balance gameplay (e.g., ETH +100, SOL +75).
* **Transactions:** Buys-ins utilize $VBV or $AVoi. No direct cross-chain asset swaps; utility is derived from metadata aggregation.

---

## 5. INTERACTIVE IMMERSIVE SOCIAL LAYERS

### Current State (Production-Ready Beta)
1. **Lobby & Real-Time Interaction:** Shared hub for chat, spectating, and challenges. Builds rivalries via reputation-based matchmaking.
2. **Reputation & Wanted System:** Heists and DNFs increase Wanted Levels, resulting in matchmaking penalties or courthouse fines (redistributed to Clubs).
3. **Clubs & Territories:** Guild system where clubs control districts, earning revenue from shop turnover and providing member buffs (+5% power).
4. **Tournaments:** High-stakes, verifiable on-chain bracket events driving community engagement and leaderboards.
5. **Economy:** $VBV faucet, Entity Market (player/NPC stocks), loans, and auctions.
6. **Achievements:** Broadcasted trophy unlocks that influence share price multipliers.

### High-Stakes Social Simulation (Beta Baseline)
*The Arena has evolved from a tactical battler into a full-scale social simulation:*

| Pillar / Aspect | Current Game (Beta) | Expansion Plan (Future) | Key Upgrades |
| :--- | :--- | :--- | :--- |
| **1. Industrial/Trust** | **COMPLETE** | Regional Alliances & Warfare | Employment records and salary loops are live. |
| **2. High-Finance** | **COMPLETE** | Cross-Asset Derivatives | Rumor Mill and Collateralized Loans are live. |
| **3. Criminality** | **COMPLETE** | Syndicate Governance | Kidnap Gambits and Hardware Traps are live. |
| **4. Deep RPG** | **COMPLETE** | Specialized Gene-Editing | Fatigue/Loyalty and Elemental Moods are live. |
| **5. Social Flex** | **COMPLETE** | Virality Multipliers | Trophy Badges and X-Sharing are live. |
| **UI/UX** | **HARDENED** | AR/VR Interaction Layers | Neon-Glass theme and 3D map are live. |