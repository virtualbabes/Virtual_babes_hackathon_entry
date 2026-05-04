*****NFT Seduction: Faucet & Tournament Platform*****

# VIRTUALBABES ARENA: ARCHITECTURE & CONTEXT MASTER DOCUMENT

## 1. OVERALL ARCHITECTURE & TECHNOLOGY STACK
Virtualbabes Arena is a blockchain-integrated card game platform blending real-time multiplayer mechanics with decentralized economics.

* **Backend (Go):** High-performance server (`server.go`) using WebSockets for real-time communication, HTTP APIs for RESTful endpoints, and blockchain indexers for on-chain verification. State is managed in-memory and via blockchain persistence (no traditional DB).
* **Game Engine (Go WASM):** Core logic compiled to WebAssembly (`main.go`, `main.wasm`) for deterministic gameplay (Triple Triad-inspired rules). Ensures tamper-proof client-side calculations.
* **Frontend (JavaScript + SCSS):** Responsive UI (`app.js`, `index.html`) utilizing WalletConnect v2 for multi-wallet support (Nautilus, Kibisis, Pera). Styled with a "neon-glass" aesthetic.
* **Blockchain Integration:** Primary support for Voi (ARC-200 tokens/NFTs) and Algorand. Uses indexers for metadata, verification, and state reconstruction.
* **Security Model:** "Switchboard Pattern" (server-side signing for payouts; client-side nonce proofs to prevent replay attacks). Zero client-side private key exposure.
* **Deployment:** GitHub + Render (hosting), Carrd.co (landing/status), Docker-ready environments.

---

## 2. CORE COMPONENTS BREAKDOWN

### Backend Services (`*.go`)
* `server.go`: Central hub. Initializes state, handles WS upgrades, HTTP routes, rate-limiting, and concurrent clients via goroutines/mutexes.
* `lobby_manager.go`: Real-time state manager (clients, matches, tournaments). Broadcasts updates and enforces rules.
* `tournament_manager.go`: Handles 8/16-player brackets, verifies on-chain buy-ins, advances rounds, and archives results as blockchain notes.
* `battle_service.go`: Oversees match logic, PvP validation, and server-authoritative win calculations.
* `economy_service.go`: Manages $VBV token faucet, loans, auctions, black markets, and collateral liquidation.
* `achievement_service.go`: Tracks/unlocks trophies; updates leaderboards.
* `courthouse_service.go`: Manages Wanted Level resets via $VBV fines; distributes fines to club treasuries.
* `handlers_admin.go`: Admin panel (ARC-14 signature auth) for network config, bans, and broadcasts.
* `handlers_public.go`: Public APIs for Carrd integration (leaderboards, status).
* `bridge_service.go`: Sybil-protected Algorand-to-Voi onboarding ("Starter Pack" claims).
* `shop_registry.go`: Defines consumable and security upgrade items.
* `common_types.go`: Shared structs (`Player`, `TournamentState`, `Card`).
* `career.go`: Framework for club employment roles (Manager, Security, Clerk).

### Frontend Architecture (`Public/`)
* `main.go` (WASM): Game state (`Engine` struct), AI logic (weighted scoring for Hard Mode), deck building, rule enforcement, and asset pooling.
* `app.js`: UI state, WS connections, WalletConnect, audio controls, tournament spectating, and indexer asset resolution.
* `index.html`: Entry point. Embeds WASM, sets up SDKs (AlgoSDK, WalletConnect).
* `styles.css` / `src/scss/`: Modular SCSS structure utilizing `_variables.scss` for neon-cyan/purple palettes, `_neon-glass.scss` for glassmorphism, and feature-specific styling (`_criminality.scss`, `_territory.scss`).

---

## 3. CROSS-CHAIN FUNCTIONALITY & ORACLE SERVICE
Managed via `oracle_service.go` and Wallet Linking.

### Supported Networks
* **Primary (Full Tx Support):** Voi (Main network, $VBV, Tournaments), Algorand (Secondary, $AVoi, Bridging).
* **Metadata-Only (NFT Discovery):** Ethereum (ERC-721/1155), Polygon, Solana (Metaplex DAS), Bitcoin (Ordinals), Flow, WAX.

### Mechanisms
* **Wallet Linking:** Non-AVM wallets link to the primary AVM wallet via server-side verification.
* **NFT Discovery:** Oracle queries linked wallets across chains (ARC-72, Etherscan, RPCs) to fetch and cache metadata.
* **Power Scaling:** Base power boosts are applied to cross-chain NFTs to balance gameplay (e.g., ETH +100, SOL +75).
* **Transactions:** Buys-ins utilize $VBV or $AVoi. No direct cross-chain asset swaps; utility is derived from metadata aggregation.

---

## 4. INTERACTIVE IMMERSIVE SOCIAL LAYERS

### Current State (Production-Ready Beta)
1. **Lobby & Real-Time Interaction:** Shared hub for chat, spectating, and challenges. Builds rivalries via reputation-based matchmaking.
2. **Reputation & Wanted System:** Heists and DNFs increase Wanted Levels, resulting in matchmaking penalties or courthouse fines (redistributed to Clubs).
3. **Clubs & Territories:** Guild system where clubs control districts, earning revenue from shop turnover and providing member buffs (+5% power).
4. **Tournaments:** High-stakes, verifiable on-chain bracket events driving community engagement and leaderboards.
5. **Economy:** $VBV faucet, Entity Market (player/NPC stocks), loans, and auctions.
6. **Achievements:** Broadcasted trophy unlocks that influence share price multipliers.

### Expansion Roadmap: Beta vs. Future Simulation
*Transitioning from a Tactical Card Game to a Social Economic Simulation.*

| Pillar / Aspect | Current Game (Beta) | Expansion Plan (Future) | Key Upgrades |
| :--- | :--- | :--- | :--- |
| **Vision** | Tactical NFT card game with tournaments & basic economy. | Full social simulation: combat + investment + politics + criminality. | Shifts to a "living world" with RPG elements, fatigue, and performative assets. |
| **1. Industrial/Trust** | Basic clubs/territories, governors, shop revenue. | Player employment (Managers/Clerks), regional alliances, Mojo unlocks. | Deepens trust mechanics, service records, and industrial staffing. |
| **2. High-Finance** | Entity market (shares), basic loans/auctions. | Rumor mill (price manipulation), second-hand stores, art galleries. | Introduces risk (defaulted collateral), strategic auctions, and market manipulation. |
| **3. Criminality** | Wanted levels, heists, courthouse fines. | Kidnapping gambits (hostages), collective NPC intelligence/taunts, insurance. | Shifts from purely punitive to high-risk gambits and narrative-driven NPC reactions. |
| **4. Social Flex** | Achievements, lobby chat, gloating. | Badge multipliers, 1-click X/Twitter sharing, enhanced portfolios. | Makes performance liquid; ties social status directly to economic value and virality. |
| **UI/UX** | Neon-glass theme, basic overlays. | Particle effects, shader backgrounds, holographic cards. | Upgrades static polish to dynamic, immersive visuals. |
