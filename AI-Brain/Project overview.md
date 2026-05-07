1. Architectural Integrity & Modularity
The project has undergone significant refactoring to move away from a monolithic structure. The architecture is now defined by Separation of Concerns:

Domain-Specific Services: Logic is encapsulated in dedicated files such as battle_service.go, club_service.go, economy_processing.go, and loan_service.go.
WASM Engine Determinism: The main.go file (compiled to main.wasm) ensures that core rules like "Same," "Plus," and "Combo" are identical for both the client and server, preventing tactical exploits.
The Switchboard Pattern: A robust security model where the server handles private keys (Faucet/Admin) to sign transactions, while the client provides nonces as "proof of intent," ensuring no private key exposure.
2. The Industrial & Economic Loop
A core highlight of the development is the implementation of a circular economy, referred to in the documents as the Industrial Loop:

Dynamic Scaling: Payouts from the faucet automatically scale based on vault liquidity, preventing depletion.
Revenue Rerouting: Instead of "burning" tokens, economic sinks like Courthouse Fines, Auction Commissions, and Heist Fence Fees are redistributed back into Club Treasuries or the Faucet pool.
Industrial Leases: A sophisticated rental market where Club members can lease high-value tactical cards, with automated revenue sharing between the Lender, the Club, and the Faucet.
3. Social Hierarchy & RPG Mechanics
The simulation introduces "Social Standing" as a tangible asset:

Clubs & Governance: Fully functional territory map and club creation system with regional governor status.
Employment System: Functional careers with Mojo-based tiers and automatic salary distributions.
Dynamic Attributes: RPG-style stats—Mojo (Social Rank), Cunning (Stealth/Success), and Nurturing (Fatigue mitigation)—are now wired into the combat power calculations and criminal success rates.
Cosmetics with Utility: Faceplate registry provides functional bonuses to Mojo and Cunning, influencing social rank and heist success.
4. Criminality, Market & Social
The "High-Stakes" aspect of the simulation is driven by:

Kidnap Gambit: A high-risk mechanic where cards can be taken hostage for ransom or insurance recovery.
Entity Market: A trading platform where players buy/sell shares in NPCs and other players, influenced by a real-time Rumor Mill and market sentiment.
Black Market: Liquidation of collateral from defaulted loans, creating a high-risk secondary market for "stolen" assets.
Social Hub: A unified interface for managing Alliances, Career paths, and Trophies (Valor).
5. Security & Launch Readiness
The project is currently in a Production-Ready Beta state. Recent audits have hardened the following:

Sybil Protection: Algorand-to-Voi onboarding is gated by historical indexer checks to prevent drain-and-claim exploits.
Spectator Sync: Live spectating is fully synchronized, including authoritative board moods and penalty snapshots.
Tournament Automation: Bracket management for 8/16-player events is fully functional, with on-chain archival of results.
Orphan Resolution: Significant effort has been spent consolidating "orphaned" logic (e.g., payout address validation and loan processing) into the new service-oriented architecture.
6. Technical Debt & To-Do
The primary focus before the full Mainnet launch remains:

Secret Management: Securely wiring FAUCET_MNEMONIC and ADMIN_WALLETS into the production environment.
Mainnet Finalization: Confirming Node/Indexer stability and finalizing the WalletConnect Project ID configuration.
