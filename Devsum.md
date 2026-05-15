Virtualbabes Arena is an experimental project that aims to evolve the tactical card battler genre into a complex **Social Economic Simulation**. Built specifically for the Voi network, the platform integrates real-time multiplayer combat with deep organizational management, high-finance markets, and high-stakes criminality.

**Current Status:** Production-Ready Beta. The codebase has undergone a significant hardening phase focused on architectural modularity, economic circularity, and cryptographic verification.

---

## 2. Technical Architecture
The system has been refactored from a monolith into a **Domain-Driven Service Architecture** to ensure maintainability and reduce mutex contention:
*   **Modular Services**: Specialized logic is encapsulated in `battle_service.go`, `club_service.go`, `economy_service.go`, `employment_service.go`, and `oracle_service.go`.
*   **Authoritative Backend (Go):** Manages real-time state via WebSockets and enforces rule sets verified by on-chain data.
*   **Deterministic Game Engine (Go WASM):** The core combat logic is written in Go and compiled to WebAssembly. This allows the same code to run in the browser and on the server, ensuring identical rule enforcement (Triple Triad-inspired) for both players and spectators.
*   **Modular Frontend (JS/SCSS):** A comprehensive cleanup of `Public/app.js` has enforced strict modular authority, delegating UI and feature logic to specialized domain files (e.g., `economy.js`, `criminality.js`).
*   **Switchboard Security Pattern:** Implements server-side signing for rewards and administrative actions while maintaining zero client-side private key exposure. Proof of intent is established via cryptographically signed nonces.

---

## 3. The Arena Simulation Pillars

### A. The Industrial Loop (Circular Economy)
The ecosystem features a complete circular economy where protocol fees are intelligently redistributed:
*   **Faucet Logic:** Rewards are dispensed for victories, with dynamic scaling based on current vault liquidity.
*   **Industrial Leases:** A fully functional rental market for tactical assets with automated revenue splits between Lenders, Clubs, and the Faucet.
*   **Employment & Salaries**: Club owners hire players into specialized roles (Manager, Security, Clerk) with automated daily salary distributions.
*   **Revenue Rerouting:** Instead of burning tokens, fees from the Art Gallery (Auctions), Courthouse (Wanted Level resets), and Heists are redistributed back into Club Treasuries and the Faucet pool.

### B. High-Finance & Market Layer
*   **Entity Market:** Real-time fractional equity trading in players and NPCs. Pricing is driven by a dynamic Reputation system that accounts for wins, achievements, and "Rumor Mill" manipulation.
*   **On-Chain Audibility**: High-value share trades are recorded on the blockchain via `VBT_SHARE_TRADE:` notes, ensuring an immutable audit trail.
*   **Art Gallery (Auctions):** Server-side internal escrow system for listing and bidding on card bundles with automated commission re-routing.
*   **Second-Hand Store (Loans):** Card-collateralized lending with automated liquidation paths into the Black Market upon default.

### C. Criminality & Intelligence
*   **Tactical Heists:** A risk-based looting system where players rob Club treasuries, countered by deployable hardware (Sentry Turrets, Bio-Guard Dogs). Success is derived from player Cunning vs. Club Security Level.
*   **Kidnap Gambits:** Seize opponent cards during heists for $VBV ransom or wait for an automated "Insurance Recovery" cycle.
*   **NPC Intelligence:** An observation loop that evaluates player playstyle (Risk/Aggressiveness) to trigger contextual NPC taunts in the global lobby.

---

## 4. Resilience & Financial Proof
*   **ARC-200/ARC-72 Support:** Native integration for Voi tokens and NFT standards.
*   **Global Result Recovery**: The Oracle reconstructs player leaderboards, match history, and tournament registrations directly from on-chain notes, ensuring state persistence across server restarts.
*   **Receipt-Backed Brackets**: Automated 8/16-player tournaments feature deep verification, cross-referencing bracket results with `VBT_WIN` payout receipts and deterministic `PayoutsHash` proofs.
*   **Bound Verification**: Critical economic events (Courthouse resets, Bail, Loan repayments) utilize purpose-specific note prefixes and timestamps to prevent transaction replay exploits.
*   **Multi-Chain Oracle:** NFT metadata discovery across Algorand, EVM, and Solana with power-scaling normalization to balance the competitive meta.
*   **RPC Reliability**: Standardized 3-attempt retry policies with backoff for all external Indexer/Node calls (HTTP 429).

---

## 5. Current Development State
*   **Feature Status:** All core services (Clubs, Markets, Criminality, Tournaments) are fully functional and hardened.
*   **Testing Infrastructure:** Includes a dedicated `test:stress` suite and a startup hook for 16-player tournament simulations to verify economic stability under load.
*   **Persistence**: Implemented `DATA_DIR` support for Render persistent volumes to secure JSON caches and audit logs.
*   **Deployment:** Docker-ready and optimized for Render (Backend) and GitHub Pages (Static Assets).

---

## 6. Project Vision
To create a "Living World" on the Voi network where tactical skill is just one component of success. The ultimate goal is a simulation where organizational politics, market manipulation, and territorial control are as important as the cards in your hand.

---
*Submitted for the Voi Hackathon.*

* `Solo_Developer: "Zap" of Virtualbabes.voi`
* `X:"Sleeper_world_changer @vbabesalgo" / AKA:"BM"`
* `Inspiration From and thanks to Dave, Nic, FF-series, DR`
* `Open_source_sound, AI generated Images`
* `Developer Owned and created Code`

---

## 7. Licensing
This codebase is proprietary and is provided for read-only access. Any use, reproduction, or distribution requires explicit written permission. Open-source sound assets are an exception and are subject to their respective licenses. For full details, please refer to the `LICENSE` file in the root directory.