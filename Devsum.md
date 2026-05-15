# Virtualbabes Arena: Voi Hackathon Development Summary

## 1. Executive Summary
Virtualbabes Arena is an experimental project that aims to evolve the tactical card battler genre into a complex **Social Economic Simulation**. Built specifically for the Voi network, the platform integrates real-time multiplayer combat with deep organizational management, high-finance markets, and high-stakes criminality.

**Disclaimer:** This repository represents a functional logic prototype. While the core features are implemented, the system has undergone **NO formal testing**, QA, or production-grade auditing. It is an experimental codebase submitted for the Voi Hackathon.

---

## 2. Technical Architecture
The project utilizes a distributed service architecture to manage state and logic:

*   **Authoritative Backend (Go):** A high-performance server managing WebSocket communication and domain-separated services (Battle, Economy, Club, Oracle). It handles all critical state in-memory with blockchain-based verification.
*   **Deterministic Game Engine (Go WASM):** The core combat logic is written in Go and compiled to WebAssembly. This allows the same code to run in the browser and on the server, ensuring identical rule enforcement (Triple Triad-inspired) for both players and spectators.
*   **Modular Frontend (JavaScript + SCSS):** A Single-Page Application (SPA) that orchestrates the UI, WebSockets, and WASM interactions. It uses a "Neon-Glass" modular SCSS system for a high-fidelity aesthetic.
*   **Switchboard Security Pattern:** Implements server-side signing for rewards while maintaining zero client-side private key exposure. Intent is proven via client-signed nonces.

---

## 3. Implemented Simulation Pillars (Implemented Logic)

### A. The Industrial Loop (Circular Economy)
The project implements a circular economic model where $VBV flows through various sinks and sources:
*   **Faucet Logic:** Rewards are dispensed for victories, with dynamic scaling based on current vault liquidity.
*   **Club Infrastructure:** Players can found Clubs, claim territories, and open hardware shops.
*   **Revenue Rerouting:** Instead of burning tokens, fees from the Art Gallery (Auctions), Courthouse (Wanted Level resets), and Heists are redistributed back into Club Treasuries and the Faucet pool.

### B. High-Finance & Market Layer
*   **Entity Market:** Logic for trading fractional equity in players and NPCs, with pricing influenced by wins and "Rumor Mill" manipulation.
*   **Art Gallery (Auctions):** Server-side internal escrow system for listing and bidding on card bundles.
*   **Second-Hand Store (Loans):** Logic for using cards as collateral for immediate liquidity, with automated liquidation paths into the Black Market upon default.

### C. Criminality & Intelligence
*   **Tactical Heists:** A risk-based looting system where players can attempt to rob Club treasuries, countered by deployable security hardware (Sentry Turrets, Bio-Guard Dogs).
*   **Kidnap Gambits:** Prototype logic allowing players to seize opponent cards during heists for ransom or wait for an automated "Insurance Recovery" cycle.
*   **NPC Intelligence:** An observation loop that evaluates player playstyle (Risk/Aggressiveness) to trigger contextual NPC taunts in the lobby.

---

## 4. Voi Blockchain Integration
*   **ARC-200/ARC-72 Support:** Designed to handle Voi native tokens and NFT standards for cards and cosmetics.
*   **Indexer-Driven State:** The system is designed to reconstruct leaderboards, match history, and tournament archives by reading authenticated transaction receipts directly from blockchain indexers.
*   **Multi-Chain Oracle:** Implementation logic for cross-chain NFT metadata discovery (Algorand, EVM, Solana) with power-scaling normalization.

---

## 5. Current Development State
*   **Feature Status:** Implementations for Clubs, Employment, Market Trading, Auctions, Heists, and 16-player Tournament Brackets are present in the code.
*   **Testing Status:** **None.** No unit tests, integration tests, or security audits have been performed.
*   **Deployment:** The project includes a Docker-ready environment and is structured for deployment on Render (Backend) and GitHub (Static Assets).

---

## 6. Project Vision
To create a "Living World" on the Voi network where tactical skill is just one component of success. The ultimate goal is a simulation where organizational politics, market manipulation, and territorial control are as important as the cards in your hand.

---
*Submitted for the Voi Hackathon.*

* `Solo_Developer: "Zap" of Virtualbabes.voi / X:"Sleeper_world_changer @vbabesalgo" / AKA:"BM"`
* `Inspiration From and thanks to Dave, Nic, FF-series, DR`
* `Open_source_sound, AI generated Images`
* `Developer Owned and created Code`

---

## 7. Licensing
This codebase is proprietary and is provided for read-only access. Any use, reproduction, or distribution requires explicit written permission. Open-source sound assets are an exception and are subject to their respective licenses. For full details, please refer to the `LICENSE` file in the root directory.