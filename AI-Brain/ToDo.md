# Arena Development: Launch Roadmap

## Pillar 1: Production Hardening (Active)
- [x] **Live Stress Test**: Execute 16-player tournament bracket simulation under concurrent load to verify pot distribution and kickbacks.
- [x] **Secret Security**: Finalize migration of `FAUCET_MNEMONIC` and `ADMIN_WALLETS` from `.env` to Render Environment Secrets.
- [x] **RPC Resilience**: Perform 24-hour health check on LlamaRPC and Nodly public endpoints.
- [x] **CORS Hardening**: Implement strict `CheckOrigin` filtering for WebSocket connections.

## Pillar 2: UI/UX & Immersion (Final Polish)
- [x] **Mobile Responsiveness**: Standardize `$panel-width` scaling for small screens in `_variables.scss`.
- [x] **Atmospheric Shifting**: Trigger red-tint background CSS variables during criminal "Underworld" phases.
- [x] **Visual Feedback**: Add loading shimmer states (`.animate-shimmer`) for cross-chain metadata retrieval.
- [x] **Narrative Depth**: Integrate typewriter effects for NPC taunts in the global chat.

## Pillar 3: Administrative Automation
- [x] **Season Cycle Tool**: Implement an admin command to manually trigger season rollover for testing archival receipts.
- [x] **Audit Export**: Build a tool to export `admin_audit.log` into CSV for hackathon reporting.
- [x] **Admin UI**: Add 'Season Rollover' and 'Export Audit' buttons to the Admin Panel UI.
- [x] **Metadata Expansion**: Implement ARC-19 and ARC-69 discovery logic in `oracle_service.go`.

## Completed & Hardened (Reference)
*   [x] Milestone 1: Domain-Driven Refactor (Battle, Economy, Oracle).
*   [x] Milestone 2: Industrial Loop (Fee Rerouting & Card Leasing).
*   [x] Milestone 3: Competitive Archival (Receipt-backed Brackets).
*   [x] Milestone 4: Production Resilience (429 Retries & Persistent Volumes).

---

## Next Session Initialization Prompt
> "I am starting a new development session for Virtualbabes Arena. We have just completed the Hackathon Main Branch deployment (Task 150 in A.I_memory.md). 
> 
> Please review the Pillar 1 and Pillar 2 priorities in ToDo.md. We need to begin with the 16-player tournament stress test using the `npm run test:stress` configuration. Let's start by auditing the `simulateTournament` logic to ensure the treasury kickbacks are logging correctly to our local `./test_data` audit file."