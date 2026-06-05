# Arena Development: Launch Roadmap

## Pillar 0: Build Synergy & Industrial Seal (Complete)
- [x] **Unified Data Schema**: Consolidated all shared data structs (`Club`, `MatchHistory`, etc.) in `common_types.go`.
- [x] **Path Normalization**: Updated `DIR.md` to use standard relative forward-slash pathing.
- [x] **Target Isolation**: Applied `//go:build !js || !wasm` only to network-heavy orchestration (Lobby/Client) to prevent WASM bloat.
- [x] **Syntax Forensic**: Resolved structural corruption in `oracle_service.go`, `common_types.go`, and `Public/js/admin.js`.
- [x] **Index Synchronization**: Updated `DIR.md` with missing infrastructure (`render.yaml`) and isolation (`backend_types.go`) files.
- [x] **Dual-Target Build Verification**: Verified clean compilation for both WASM and Server targets.
- [x] **Render Volume Verification**: Verified the project's design correctly supports `DATA_DIR` persistence for `admin_audit.log` and other state files, contingent on Render's volume mounting configuration.

## Pillar 0.1: Ancillary System Hardening (NEW/PENDING)
- [x] **Achievement Audit**: Hardened identity validation and verified reputation ripple.
- [x] **Courthouse Audit**: Refactored fine redistribution to utilize micro-unit precision and Governor tax compliance.
- [x] **Employment Audit**: Verified dissolution safety and salary scaling in `career.go` and `employment_service.go`.
- [x] **Onboarding Failover**: Verified `loadOnboardedWalletsFromIndexer` supports multi-node failover via `l.indexerRequest`; improved logging for clarity.
- [ ] **Client Beacon Recovery**: Implement browser-side state caching to optimize startup latency.

## Pillar 1: Production Hardening (Active)
- [x] **Live Stress Test**: Execute 16-player tournament bracket simulation under concurrent load to verify pot distribution and kickbacks.
- [x] **Secret Security**: Finalized migration of `FAUCET_MNEMONIC`, `ADMIN_WALLETS`, and `AppID` to Render Environment Secrets.
- [x] **RPC Resilience**: Perform 24-hour health check on LlamaRPC and Nodly public endpoints.
- [x] **CORS Hardening**: Implement strict `CheckOrigin` filtering for WebSocket connections.

## Pillar 2: UI/UX & Immersion (Final Polish)
- [x] **Mobile Responsiveness**: Standardize `$panel-width` scaling for small screens in `_variables.scss`.
- [x] **Atmospheric Shifting**: Trigger red-tint background CSS variables during criminal "Underworld" phases.
- [x] **Visual Feedback**: Add loading shimmer states (`.animate-shimmer`) for cross-chain metadata retrieval.
- [x] **Narrative Depth**: Integrate typewriter effects for NPC taunts in the global chat.
- [x] **Bounty Board**: Implement aggregator for real-time tracking of high-Wanted players.
- [x] **Ghost Protocol**: Implement item effect to scramble Bounty Board intelligence.

## Pillar 3: Administrative Automation
- [x] **Season Cycle Tool**: Implement an admin command to manually trigger season rollover for testing archival receipts.
- [x] **Audit Export**: Build a tool to export `admin_audit.log` into CSV for hackathon reporting.
- [x] **Admin UI**: Add 'Season Rollover' and 'Export Audit' buttons to the Admin Panel UI.
- [x] **Metadata Expansion**: Implemented ARC-19, ARC-69, and Dispatcher logic in `oracle_service.go`.
- [x] **Onboarding Stability**: Resolved nilness analyzer error in `onboarding_service.go`. (Task 621)
- [x] **Mojo Decay Stress Test**: Implemented simulation and integrated `test:mojo` script into `package.json`. (Task 557)
- [x] **Asset Forfeiture**: Finalized manual card recovery protocol, API registration, and Admin UI controls.
- [x] **Player Reporting**: Finalized reporting protocol and UI integration. (Task 517)
- [x] **District Stabilizer**: Implemented activation logic for Mojo decay mitigation. (Task 508)

## Pillar 3: Criminality & Intelligence (Continued)
- [x] **Cyber-Audit Dispatch**: Modified `network.js` to dispatch 'Cyber-Audit' notifications to `app.js`. (Task 510) (Task 510)
- [x] **Mojo Decay Stress Test Execution**: Executed the Mojo Decay stress test and monitored terminal output. (Task 583)
- [x] **Intelligence Bonus Audit**: Hardened `CalculateReputation` to ensure audits against allied clubs are excluded from bonuses. (Task 607)
- [x] **Corporate Espionage Achievement Audit**: Hardened `checkCorporateEspionageAchievementLocked` to require 5 RIVAL audits. (Task 608)

## Pillar 1: Infrastructure & Trust (Continued)
- [x] **Mojo Surge Tracking**: Added `MojoStartOf24hWindow` and `MojoWindowStartTime` to `Club` struct. (Task 513)
- [x] **Mojo Surge Integration**: Wired achievement trigger into `item_service.go`. (Task 515)
- [x] **Mojo Surge Audit**: Hardened 24-hour window reset logic to handle manual system clock shifts backward. (Task 610)
- [x] **Mojo Decay Audit**: Hardened `processMojoDecayLocked` to reset surge window baseline during decay events. (Task 612)

## Pillar 4: Live Deployment & Monitoring (Next)

- [x] **High-Availability Health Check**: Final audit of `handleHealthCheck` to ensure multi-node cycling is fully active in production.

## Completed & Hardened (Reference)
- [x] Milestone 1: Domain-Driven Refactor (Battle, Economy, Oracle).
- [x] Milestone 2: Industrial Loop (Fee Rerouting & Card Leasing).
- [x] Milestone 3: Competitive Archival (Receipt-backed Brackets).
- [x] Milestone 4: Production Resilience (429 Retries & Persistent Volumes).
- [x] Milestone 5: Intelligence & Administrative Layers (Pillar 3).
- [x] Milestone 6: Comprehensive Documentation Alignment.
- [x] Milestone 7: Blockchain-Native Persistence & Ledger Verification.
- [x] Milestone 8: Authoritative Hackathon Entry (Updates-land-here). **(FINALIZED)**
- [ ] Milestone 9: High-Fidelity Documentation Alignment Pass. (Current)

## Post-Hackathon Mainnet Scaling Priorities
- [ ] **Phase 4: Direct Cross-Chain Asset Bridging**: Implement `bridge_service.go` for direct asset transfers.
- [ ] **Advanced NPC Dynamics**: Further integrate Collective Intelligence for more nuanced NPC interactions and market responses.
- [ ] **Regional Warfare & Territory Sabotage**: Deepen club-level conflict with strategic territory control and direct sabotage mechanics.
- [ ] **Specialized Gene-Editing**: Introduce advanced card attribute mutation and customization.
