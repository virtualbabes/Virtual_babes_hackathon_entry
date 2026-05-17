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
- [x] **Metadata Expansion**: Implement ARC-19, ARC-69, and Dispatcher logic in `oracle_service.go`.

## Pillar 4: Live Deployment & Monitoring (Next)
- [x] **Health Check Hardening**: Audit `handleHealthCheck` in `handlers_public.go` to verify RPC connectivity and Faucet liquidity for Render monitoring.
- [x] **Post-Deployment Audit**: Verify WASM Engine status ('ACTIVE') and Ledger aggregation ('Liquid Balance') on the live site.

## Pillar 5: Build Stabilization & Logic Scrub (Pending)

## Pillar 5: Build Stabilization & Logic Scrub (Active)
- [x] **Backend vs WASM Isolation**: Applied build tags to isolate server logic.
- [x] **Resolve Redeclarations**: Fixed `GlobalSentiment` and `main()` conflicts.
- [x] **Syntax Repair**: Repaired function boundaries in `oracle_service.go` and JS syntax in `admin.js`.
- [ ] **Pluralization Audit**: Update singular `NodeURL` to `NodeURLs` in `economy_service.go` and `handlers_admin.go`.
- [ ] **Ledger Synchronization**: Replace `l.rewards` with `l.playerBalances` in `handlers_admin.go` and `lobby_manager.go`.
- [ ] **Syntax Repair**:
    - `oracle_service.go`: Fix expected `;` found `{` at line 703.
    - `tournament_manager.go`: Fix expected statement found `)` at line 649.
    - `Public/app.js`: Fix missing braces and malformed logic around line 586/2173.
    - `Public/js/admin.js`: Repair malformed `try-catch` blocks and missing statements (line 205-369).
    - `Public/js/wallet.js`: Fix missing closing brace at line 276.
- [ ] **Compiler Compliance**: Resolve unused `pIdx` in `battle_service.go` and unused `p` in `main.go`. Fix `:=` shadowing in `achievement_service.go` and `club_service.go`.
- [ ] **Data Model Restoration**: Restore missing `Wallet` field to `PlayerStats` struct and fix `jsMap` references in `main.go`.

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