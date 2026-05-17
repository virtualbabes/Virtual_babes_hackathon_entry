# Virtualbabes Arena: Technical Summary
**Virtualbabes Arena** is a first-of-its-kind **Social Economic Simulation** built on the Voi Network. It transcends the classic tactical card battler by integrating real-time multiplayer combat into a living ecosystem of fractional equity markets, organizational governance, and high-stakes criminality.

**Current Status:** Production-Ready Hardened Beta. 
The platform has successfully transitioned from a monolithic architecture into a high-performance, domain-separated service model, fully audited for economic circularity and cryptographic financial proof.

---

## 2. Technical Architecture
The system has been refactored from a monolith into a **Domain-Driven Service Architecture** to ensure maintainability and reduce mutex contention:
*   **Modular Services**: Specialized logic is encapsulated in `battle_service.go`, `club_service.go`, `economy_service.go`, `employment_service.go`, and `oracle_service.go`.
*   **Authoritative Backend (Go):** Manages real-time state via WebSockets and enforces rule sets verified by on-chain data.
*   **Deterministic Game Engine (Go WASM):** The core combat logic (Triple Triad-inspired) is compiled to WASM. This ensures **mathematical parity** between the client and server, preventing tactical exploits and ensuring spectators see the identical board state as combatants.
*   **Modular Frontend (JS/SCSS):** A comprehensive cleanup of `Public/app.js` has enforced strict modular authority, delegating UI and feature logic to specialized domain files (e.g., `economy.js`, `criminality.js`).
*   **The Switchboard Pattern:** A robust security model where the server manages high-value keys (Faucet/Admin) to sign rewards, while clients provide cryptographically signed **nonces** as "proof of intent." This enables secure, gasless-feel interactions with zero private key exposure.

---

## 3. The Arena Simulation Pillars

### A. The Industrial Loop (Circular Economy)
The ecosystem features a complete circular economy where protocol fees are intelligently redistributed:
*   **Dynamic Scaling:** Reward payouts automatically scale based on real-time vault liquidity, ensuring long-term economic sustainability.
*   **Revenue Rerouting:** Instead of "burning" tokens, economic sinks (Courthouse Fines, Auction Commissions, Heist Fence Fees) are redistributed into Club Treasuries or back to the Faucet.
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

---

## 4. Resilience & Financial Proof
*   **Multi-Standard Oracle:** Native discovery and metadata resolution for **ARC-72**, **ARC-19**, and **ARC-69** standards, plus cross-chain support for **Ethereum**, **Solana**, and **Polygon**.
*   **Database-less Architecture:** Utilizing **Global Result Recovery**, the server reconstructs the entire leaderboard, tournament registration, and match history directly from blockchain notes upon startup.
*   **Receipt-Backed Brackets:** Automated 16-player tournaments utilize a deterministic `PayoutsHash` to cryptographically prove that all winners were paid on-chain.
*   **Production Resilience:** Standardized RPC failover cycling and 429 retry policies ensure 100% uptime even during heavy indexer load or network congestion.

---

## 5. Current Development State
*   **Infrastructure:** Docker-ready environment with Render-optimized persistence for audit logs and behavioral caches.
*   **Testing:** Successfully passed 16-player high-concurrency stress tests, verifying tournament bracket integrity and treasury kickback accuracy.
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