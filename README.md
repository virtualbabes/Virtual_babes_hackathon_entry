##### Developer Branch:

#### NFT Seduction: Faucet & Tournament Platform ####

### VIRTUALBABES ARENA: ARCHITECTURE & CONTEXT MASTER DOCUMENT
Current Development Status: Production-Ready Beta
All core systems are implemented, security-hardened, and UI-unified. 
The Go server serves static frontend assets from ./Public.
The runtime expects the generated artifact Public/main.wasm and Public/wasm_exec.js alongside HTML/CSS/JS.

## Project Goal
To evolve the classic tactical card battler into a high-stakes Social Economic Simulation. 
The platform rewards not just combat skill, but strategic investment, political maneuvering within Card Clubs, and the management of "Social Standing" (Reputation and Mojo).
It operates on both Voi and Algorand networks, ensuring secure transactions and persistent data storage via indexer receipts.


## 1. OVERALL ARCHITECTURE & TECHNOLOGY STACK ##

# Virtualbabes Arena is a blockchain-integrated card game platform blending real-time multiplayer mechanics with decentralized economics.

*Backend (Go): High-performance server (server.go) using WebSockets for real-time communication, HTTP APIs for RESTful endpoints, and blockchain indexers for on-chain verification.

*State is managed in-memory and via blockchain persistence (no traditional DB).

*Game Engine (Go WASM): Core logic compiled to WebAssembly (main.go, main.wasm) for deterministic gameplay (Triple Triad-inspired rules).

*Ensures tamper-proof client-side calculations.

*Frontend (JavaScript + SCSS): Responsive UI (app.js, index.html) utilizing WalletConnect v2 for multi-wallet support (Nautilus, Kibisis, Pera).

*Styled with a "neon-glass" aesthetic.

*Blockchain Integration: Primary support for Voi (ARC-200 tokens/NFTs) and Algorand.

*Uses indexers for metadata, verification, and state reconstruction.

*Security Model: "Switchboard Pattern" (server-side signing for payouts; client-side nonce proofs to prevent replay attacks). 

*Zero client-side private key exposure. 

*All critical game state (leaderboards, match history, tournament archives, DNF penalties) is reconstructed by reading authenticated data and receipts directly from blockchain indexers.
*Deployment: GitHub + Render (hosting), Carrd.co (landing/status), Docker-ready environments.


## 2. CORE COMPONENTS BREAKDOWN ##

*Backend Services (*.go) - Orchestrated by lobby_manager.go

*server.go: Central hub. Initializes state, handles WS upgrades, HTTP routes, rate-limiting, and concurrent clients via goroutines/mutexes.

*lobby_manager.go: Real-time state manager (clients, matches, tournaments). Orchestrates updates and enforces rules.

*tournament_manager.go: Handles 8/16-player brackets, verifies on-chain buy-ins, advances rounds, and archives results as blockchain notes.

*battle_service.go: Oversees match logic, PvP validation, and server-authoritative win calculations, including item effect application.

*economy_service.go: Manages dynamic faucet scaling, and records on-chain notes for season archives and DNF penalties.

*faucet_service.go: Handles secure $VBV token faucet payouts using the "Switchboard Pattern" and nonce verification.

*loan_service.go: Manages collateralized loans, repayment, and default processing.

*auction_service.go: Manages item auctions, bidding, and settlement.

*black_market_service.go: Handles liquidated collateral from defaulted loans.

*achievement_service.go: Tracks/unlocks trophies; updates leaderboards.

*courthouse_service.go: Manages Wanted Level resets via $VBV fines; distributes fines to club treasuries.

*handlers_admin.go: Admin panel (ARC-14 signature auth) for network config, bans, broadcasts, and system controls.

*handlers_public.go: Public APIs for leaderboard, status, and general information.

*handlers_criminality.go: Manages criminal actions like kidnapping gambits and heist results.

*handlers_rumor.go: Handles rumor spreading for market manipulation.

*onboarding_service.go: Manages Sybil-protected Algorand-to-Voi onboarding ("Starter Pack" claims).

*bridge_service.go: Reserved for future multi-chain bridge services (currently a placeholder).

*shop_registry.go: Defines consumable and security upgrade items.

*item_service.go: Centralizes logic for applying item effects (buffs, traps).

*common_types.go: Shared structs (Player, TournamentState, Card, Club, Loan, Auction, Rumor, etc.).

*career.go: Manages the daily salary dispenser for club employees.

*employment_service.go: Handles club employment roles (Manager, Security, Clerk) and salary setting.

*market_service.go: Manages entity stock trading and global sentiment analysis.

*Game Engine (Go WASM)

*main.go (WASM): Core game logic (Engine struct), AI logic (weighted scoring for Hard Mode), deck building, rule enforcement, and asset pooling. Communicates with the server via WebSockets and exposes functions to JavaScript.

*Frontend Architecture (Public/)

*app.js: UI state, WebSocket connections, WalletConnect integration, audio controls, tournament spectating, and indexer asset resolution. Orchestrates client-side interactions.

*index.html: Entry point. Embeds WASM, sets up SDKs (AlgoSDK, WalletConnect), and defines the base UI structure.

*styles.css / src/scss/: Modular SCSS structure utilizing _variables.scss for neon-cyan/purple palettes, _neon-glass.scss for glassmorphism, and feature-specific styling (_criminality.scss, _territory.scss).


## 3. CROSS-CHAIN FUNCTIONALITY & ORACLE SERVICE ##
*Managed via oracle_service.go and Wallet Linking.

# Supported Networks (Dynamic via networks.json)
*Primary (Full Tx Support): Voi (Main network, $VBV, Tournaments), Algorand (Secondary, $AVoi, Bridging).
*Metadata-Only (NFT Discovery): Ethereum (ERC-721/1155), Polygon, Solana (Metaplex DAS), Bitcoin (Ordinals), Flow, WAX.

# Mechanisms
*Wallet Linking: Non-AVM wallets link to the primary AVM wallet via server-side verification.
*NFT Discovery: Oracle queries linked wallets across chains (ARC-72, Etherscan, RPCs) to fetch and cache metadata.
*Power Scaling: Base power boosts are applied to cross-chain NFTs to balance gameplay (e.g., ETH +100, SOL +75).
*Transactions: Buy-ins utilize $VBV or $AVoi. No direct cross-chain asset swaps; utility is derived from metadata aggregation.


## 4. INTERACTIVE IMMERSIVE SOCIAL LAYERS ##

# Core Features
1. Engaging NFT Seduction Gameplay:

*Classic NFT Seduction rules with "Same," "Plus," and "Combo" mechanics.
*Dynamic AI (PvE) with adjustable difficulty and "thinking" simulations.
*Player-vs-Player (PvP) matchmaking with reputation-based heuristics.
*NFT-based cards with rarity, power ratings, visual tiers, and Elemental Moods.

2. The Industrial & Trust Layer (Social Hierarchy):
*Club Founding & Governance: Establish clubs, own territories, and expand into Regions.
*Employment & Careers: Owners hire players as Managers, Security, or Clerks with daily salaries paid from Club Treasuries.
*Courthouse Rerouting: Fines are no longer burned but distributed as revenue to active Club Treasuries.

3. High-Finance & Market Layer:
*Entity Market: Trade shares in players and NPCs using $VBV reward balances.
*Rumor Mill: Spread rumors to manipulate market sentiment and share prices via multipliers.
*Black Market: Buy liquidated collateral from defaulted loans at a discount (carrying "Stolen" tags).
*Dynamic Ticker: Real-time market data reflecting performance, rumors, and achievements.

4. Criminality & Intelligence:

*Heists & Security: Loot Club Treasuries, countered by deployable hardware (Sentry Turrets, Guard Dogs, Tripwires).
*Kidnap Gambit: High-stakes captures of opponent cards for $VBV ransom or wait for Insurance Recovery.
*Bounty Board: Hunters track high-infamy "Outlaws" for specialized VBV bounties.

5. Decentralized Economy & Faucet System:

*Multi-Chain Support: Operates on Voi Mainnet (primary) and Algorand Mainnet (secondary for $AVoi buy-ins).
*$VBV Token Rewards: Players earn $VBV (ARC-200) for victories, dispensed via a secure server-side faucet.
*NFT Value Integration: In-game card power ratings are dynamically derived from on-chain NFT metadata (mint round, sales history, rarity).
*Balanced Faucet Vault: Designed to prevent depletion through careful reward mechanics and future dynamic scaling.

6. Rewarding Tournament System:

*Automated, bracket-based tournaments (8 or 16 players).
*Buy-in mechanics (with $VBV or $AVoi) for prize pool contribution.
*"Elite Privilege" free passes for top-ranked players.
*Rank-based tiered payouts for Top 5 finishers.
*On-Chain Archiving: Tournament results are permanently recorded on the blockchain as linked transaction notes for verifiable history.

7. Secure & Verifiable Transactions:

*Faucet Payouts (Server-Side): All $VBV/VOI dispensing transactions from the faucet's vault are constructed, signed, and broadcasted securely by the backend (server.go). The faucet's private keys (mnemonic) are *never exposed to the client. Client-side requests for rewards use a "reverse sign" nonce verification pattern, where the user proves intent without handling the actual fund transfer.
*Tournament Buy-ins (Client-Side & Verified): Users initiate buy-in transactions from their own wallets to the faucet's vault. The server.go backend then on-chain verifies these transactions via indexers to *confirm payment before registering the participant.
*On-Chain Data for Persistence: All critical game state (leaderboards, match history, tournament archives, DNF penalties) is reconstructed by reading authenticated data and receipts directly from blockchain indexers.
*Historical Sybil Protection: Onboarding starter packs are protected by Indexer-based historical checks to prevent repeated claims per wallet identity.

8. User Experience & Infrastructure:

*Responsive UI with WalletConnect v2, Nautilus, and Kibisis wallet integration.
*Go WASM engine for core game logic, ensuring deterministic gameplay and portability.
*Real-time lobby updates, chat, matchmaking, and live spectating via WebSockets.
*Admin controls for managing rewards, rules, maintenance, and player bans.
*Asset opt-in checks to prevent failed reward payouts.

9. Deep RPG Mechanics:

*The Fatigue/Loyalty Loop: Manage card persistence; overused cards lose power, while soul-bonded cards gain bonuses.
*Elemental Synthesis: Tactical depth via card Moods interacting with Tile Moods on the battle board.
*Social Flex: Integrated X/Twitter sharing for match results and achievement broadcasts.

##  5.  Architectural Highlights ##

*Go Backend (server.go): Manages real-time lobby state, matchmaking, admin APIs, secure faucet operations, and blockchain interactions (signing and broadcasting faucet-controlled transactions, querying indexers).

*Go WASM Engine (main.go): Encapsulates core game logic (NFT Seduction rules, AI, card evaluation), providing a deterministic and performant gameplay experience within the browser. Exposes a JS bridge for UI interactions.

*Frontend (Public/app.js, Public/index.html, Public/styles.css): Handles UI rendering, wallet connections, user input, and communicates with the backend via WebSockets and REST APIs.

##  6. Current Status ##
The repository is in a production-ready beta state. Security primitives, cross-chain inventory, and automated tournament brackets are fully functional. The UI is unified under a cohesive glassmorphism theme.

## Near-Term Focus
Securely wire FAUCET_MNEMONIC and ADMIN_WALLETS for Mainnet launch.
Finalize WC_PROJECT_ID configuration.
Verify Mainnet Node/Indexer stability in networks.json.
Perform 16-player tournament stress tests.
Expansion Roadmap: Beta vs. Future Simulation
Transitioning from a Tactical Card Game to a Social Economic Simulation.

##   7. Pillar / Aspect	Current Game (Beta)	##

1. Industrial/Trust	Basic clubs/territories, governors, shop revenue.	Player employment (Managers/Clerks), regional alliances, Mojo unlocks.	Deepens trust mechanics, service records, and industrial staffing.

2. High-Finance	Entity market (shares), basic loans/auctions.	Rumor mill (price manipulation), second-hand stores, art galleries.	Introduces risk (defaulted collateral), strategic auctions, and market manipulation.

3. Criminality	Wanted levels, heists, courthouse fines.	Kidnapping gambits (hostages), collective NPC intelligence/taunts, insurance.	Shifts from purely punitive to high-risk gambits and narrative-driven NPC reactions.
 
4. Social Flex	Achievements, lobby chat, gloating.	Badge multipliers, 1-click X/Twitter sharing, enhanced portfolios. Makes performance liquid; ties social status directly to economic value and virality.

5. UI/UX	Neon-glass theme, basic overlays.	Particle effects, shader backgrounds, holographic cards.	Upgrades static polish to dynamic, immersive visuals.

## Expansion Plan (Future)	Key Upgrades

1. Vision	Tactical NFT card game with tournaments & basic economy.

2. Full social simulation: combat + investment + politics + criminality.	

3. Shifts to a "living world" with RPG elements, fatigue, and performative assets.
