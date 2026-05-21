# Virtualbabes Arena: Technical Summary
**Virtualbabes Arena** is a first-of-its-kind **Social Economic Simulation** built on the Voi Network. It transcends the classic tactical card battler by integrating real-time multiplayer combat into a living ecosystem of fractional equity markets, organizational governance, and high-stakes criminality.

**Current Status:** Production-Ready / Build-Stabilized (Hackathon Phase Complete).
The platform has successfully transitioned from a monolithic architecture into a high-performance, domain-separated service model. It features a unique dual-target build system for absolute synergy between server-side authoritative logic and client-side deterministic execution.

---

## 2. Technical Architecture
The system utilizes a **Dual-Target Build Synergy Architecture** to ensure 100% type safety and mathematical consistency:
*   **Pure Data Bridge (`common_types.go`)**: A WASM-friendly schema for shared objects (Clubs, MatchHistory, Stats), enabling identical state interpretation between Go Server and WASM.
*   **Server State Container (`backend_types.go`)**: Isolates network-heavy orchestration (Lobby/Client) via build tags (`//go:build !js || !wasm`) to prevent dependency leakage into the browser engine.
*   **Modular Services**: Specialized domain logic is encapsulated in dedicated service files (Battle, Club, Economy, Oracle).
*   **Build Integrity**: Standardized build tags and structural method isolation ensure concurrent stability across the service architecture.
*   **Authoritative Backend (Go)**: Manages real-time state via WebSockets and enforces rules verified by on-chain data.
*   **Deterministic Game Engine (Go WASM)**: Core combat logic compiled to WASM for tamper-proof client-side calculations and spectator fidelity.
*   **Modular Frontend (JS/SCSS):** A comprehensive cleanup of `Public/app.js` has enforced strict modular authority, delegating UI and feature logic to specialized domain files (e.g., `economy.js`, `criminality.js`).
*   **The Switchboard Pattern:** A robust security model where the server manages high-value keys (Faucet/Admin) to sign rewards, while clients provide cryptographically signed **nonces** as "proof of intent." This enables secure, gasless-feel interactions with zero private key exposure.

---

## 3. The Arena Simulation Pillars

### A. The Industrial Loop (Circular Economy)
The ecosystem features a complete circular economy where protocol fees are intelligently redistributed:
*   **Dynamic Scaling:** Reward payouts automatically scale based on real-time vault liquidity and a **1.0 VOI Gas Floor**, ensuring long-term economic sustainability and operational uptime.
*   **Revenue Rerouting:** Instead of "burning" tokens, economic sinks (Courthouse Fines, Auction Commissions, Heist Fence Fees) are redistributed into Club Treasuries or back to the Faucet.
*   **Integer Supremacy:** Absolute ledger integrity is maintained via `uint64` micro-unit integer math for all virtual balance adjustments, eliminating floating-point "dust drift."
*   **Industrial Leases:** A sophisticated card rental market with automated, micro-unit precise revenue splits between the Lender, the Club, and the Arena Faucet.
*   **Employment & Salaries:** Functional careers where Club owners hire players into specialized roles (Manager, Security, Clerk) with automated daily salary distributions from Club reserves.

### B. High-Finance & Market Layer
*   **Fractional Equity Trading:** Players can buy and sell "shares" in themselves or rivals. Prices are driven by combat performance, social standing, and "Rumor Mill" sentiment manipulation.
*   **Art Gallery (Auctions):** An internal escrow system for listing and bidding on multi-asset bundles (Card + Weapon + Faceplate) with automated settlement logic.
*   **Second-Hand Store (Loans):** Collateralized lending where players use Soul-Bonded cards for liquidity. Defaulted loans are liquidated into the **Black Market**, creating a high-risk secondary economy for "stolen" assets.
*   **On-Chain Audit Trail:** High-value economic events are recorded immutably via transaction notes (`VBT_SHARE_TRADE`, `VBT_LOAN_LIQUIDATE`), providing a forensic ledger of the simulation's growth.

### C. Criminality & Intelligence
*   **Tactical Heists:** A risk-based system to loot Club treasuries. Success is determined by player **Cunning** vs. Club **Security staff** and deployable hardware (Laser Tripwires, Guard Dogs).
*   **Kidnap Gambits:** Elite heisters can take a card "hostage," forcing the victim to pay a $VBV ransom or wait for a 48-hour **Insurance Recovery** cycle.
*   **Narrative Intelligence:** A server-side observation loopPorted from `collective-consciousness.js` that evaluates player traits (Risk/Aggressiveness) to trigger contextual NPC taunts in the global lobby.
*   **Bounty Board:** Real-time tracking of high-Wanted players, broadcasting their last seen district to the lobby.
*   **Ghost Protocol:** Players can pay to temporarily scramble their signal, hiding from the Bounty Board.
*   **Cyber-Audit:** Tactical item allowing players to reveal a target club's treasury average and crash status.
*   **Cyber-Counter:** Hardware trap that identifies players using Cyber-Audits against the club.
*   **Cyber-Lock:** Hardware trap that prevents all Cyber-Audits against the club for 24 hours.
*   **Player Reporting:** System for players to report malicious activity, automatically logging and notifying admins.
*   **Asset Forfeiture:** Administrative protocol for manual card recovery from club jails.

---

## 4. Resilience & Financial Proof
*   **Multi-Standard Oracle:** Native discovery and metadata resolution for **ARC-72**, **ARC-19**, and **ARC-69** standards, plus cross-chain support for **Ethereum**, **Solana**, and **Polygon**.
*   **Database-less Architecture:** Utilizing **Global Result Recovery**, the server reconstructs the entire leaderboard, tournament registration, and match history directly from blockchain notes upon startup.
*   **Deep Financial Verification:** Automated tournaments and high-value heists utilize a deterministic `PayoutsHash` (Base64 aligned with Indexer standards) to cryptographically prove that winners were paid on-chain.
*   **Production Resilience:** Standardized RPC failover cycling and 429 retry policies ensure 100% uptime even during heavy indexer load or network congestion.

---

## 5. Current Development State
*   **Infrastructure:** Docker-ready environment with Render-optimized persistence for audit logs and behavioral caches.
*   **Testing:** Verified Dual-Target build stability for both Linux and WASM targets. Successfully passed 16-player high-concurrency stress tests.
*   **Automation:** Admin suite finalized with automated season rollover and audit log CSV exporting.

---

## 6. Project Vision
To create a "Living World" on the Voi network where tactical skill is just one component of success. The ultimate goal is a simulation where organizational politics, market manipulation, and territorial control are as important as the cards in your hand.

---
*Submitted for the Voi Hackathon.*

*   **Solo Developer:** "Zap" of Virtualbabes.voi
*   **X:** @vbabesalgo / AKA: "BM"
*   **Inspiration:** Thanks to Dave, Nic, FF-series, and DR.
* `Open_source_sound, AI generated Images`
* `Developer Owned and created Code`

---

## 7. Licensing
This codebase is proprietary and is provided for read-only access. Any use, reproduction, or distribution requires explicit written permission. Open-source sound assets are an exception and are subject to their respective licenses. For full details, please refer to the `LICENSE` file in the root directory.