Developer_Branch2:  NFT Seduction: Faucet & Tournament Platform
VIRTUALBABES ARENA: SOCIAL ECONOMIC SIMULATION
Current Development Status: Production-Ready Hardened Beta

Project Goal
To evolve the classic tactical card battler into a high-stakes Social Economic Simulation. The platform rewards not just combat skill, but strategic investment, political maneuvering within Card Clubs, and the management of "Social Standing" (Reputation and Mojo). Built for the Voi Network, the Arena features a circular economy where every protocol fee is redistributed to players and organizations.

1. OVERALL ARCHITECTURE & TECHNOLOGY STACK
The project utilizes a **Modular Domain-Driven Service Architecture** to maintain state and enforce authoritative rules:

*   **Authoritative Backend (Go):** High-performance server managing WebSocket communication and domain-separated services (Battle, Economy, Club, Oracle). Handles state in-memory with high-fidelity blockchain verification.
*   **Deterministic Game Engine (Go WASM):** Core combat logic (Triple Triad-inspired) compiled to WebAssembly, ensuring identical rule enforcement between client and server to prevent tactical exploits.
*   **Modular Frontend (JS/SCSS):** Single-Page Application (SPA) orchestrating UI, WebSockets, and WASM interactions with a high-fidelity "Neon-Glass" aesthetic and fully responsive 3D territory map.
*   **Security Model:** "Switchboard Pattern" (server-side signing for rewards; client-side nonce proofs) ensuring zero private key exposure.
*   **Resilience:** Standardized 429 retry policies with backoff for all RPC/Indexer calls and `DATA_DIR` persistence for Render volumes.

2. CORE COMPONENTS BREAKDOWN
The backend is decomposed into specialized services to reduce mutex contention:

*   **Tournament Manager:** Handles 8/16-player brackets, verifies on-chain buy-ins, and archives results as blockchain notes with deterministic `PayoutsHash` financial proofs.
*   **Club Service:** Manages organizational founding, territory acquisition, and the "Industrial Loop" (Leases, Mojo, Shop Turnover).
*   **Battle Service:** Server-authoritative move validation, capture calculations, and Sudden Death resolution.
*   **Economy & Faucet:** Dynamic scaling of rewards based on vault liquidity and secure signature-based payouts.
*   **Oracle Service:** Features an intelligent `MetadataDispatcher` for auto-discovery of ARC-72, ARC-19, and ARC-69 NFT standards.
*   **Criminality Handlers:** Tactical Heists, Kidnap Gambits, and Bounty Hunter payouts.

3. SIMULATION PILLARS
A. The Industrial Loop (Circular Economy)
*   Clubs & Territories: Establish clubs, own territories, and expand into Regions.
*   Employment: Club owners hire players into specialized roles (Manager, Security, Clerk) with automated daily salaries.
*   Revenue Rerouting: Fees from Auctions, Courthouse Fines, and Heists are redistributed back into Club Treasuries and the Faucet.
*   Governor's Tax: A 5% tax on all tournament pools is automatically routed to the club controlling the Arena Center.
*   Treasury Kickbacks: Clubs earn 1-5% (scaled by Mojo) from member tournament registration fees.

B. High-Finance & Market Layer
*   Entity Market: Trade shares in players and NPCs; pricing is influenced by combat performance, scaled Reputation, and "Rumor Mill" manipulation.
*   Art Gallery (Auctions): Internal escrow system for listing and bidding on card bundles with automated commission routing.
*   Black Market: Discounted acquisition of defaulted collateral from the Loan system, carrying infamy penalties.

C. Criminality & Intelligence
*   Tactical Heists: Risk-based looting of Club treasuries, countered by deployable hardware (Sentry Turrets, Bio-Guard Dogs) managing by specialized Security staff.
*   Kidnap Gambits: High-stakes card hostage situations with Ransom or Insurance Recovery cycles.
*   NPC Intelligence: Narrative taunts triggered by the server's evaluation of player playstyle (Risk/Aggressiveness).
*   Elemental Synthesis: Tactical power boosts derived from card/tile mood alignment.

4. CROSS-CHAIN FUNCTIONALITY & ORACLE SERVICE
*Managed via oracle_service.go and Wallet Linking.

Supported Networks (Dynamic via networks.json)
*   **Primary (Full Tx Support):** Voi (Main network, $VBV, Tournaments). Native Standards: ARC-72, ARC-19, ARC-69.
*   **Secondary/Bridge:** Algorand ($AVoi).
*   **Metadata-Only (NFT Discovery):** Ethereum (ERC-721/1155), Polygon, Solana (Metaplex DAS), Bitcoin (Ordinals), Flow, WAX.

Mechanisms
*   **Wallet Linking:** Non-AVM wallets link to the primary AVM wallet via server-side verification. 
*   **NFT Discovery:** Oracle queries linked wallets across chains (Etherscan, Solana DAS, RPCs) to fetch and cache metadata. 
*   **Power Scaling:** Base power boosts are applied to cross-chain NFTs to balance gameplay (e.g., ETH +100, SOL +75). 
*   **Transactions:** Buy-ins utilize $VBV or $AVoi. No direct cross-chain asset swaps; utility is derived from metadata aggregation.

5. ADMINISTRATION & AUTOMATION
The Arena features a professional administrative suite for ecosystem maintenance:
*   **Manual Season Rollover:** Secure archival of Hall of Fame standings to the blockchain.
*   **Audit Log Exporter:** Convert JSON-line administrative logs into CSV for regulatory reporting.
*   **Global Moderation:** Real-time ban management for gloat messages and profile avatars.
