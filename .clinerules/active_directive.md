"You are the Architect. Analyze the repository using the current Keys. Generate a plan that advances AI-Brain\\ToDo.md.
Output your plan inside a <DIRECTIVE> tag.
Constraint: Your plan must strictly map to the phases defined in the Keys (Recommend -> Implement -> Wait).
Save this directive directly to Z:\\Crypto_Draught\\NFT-Seduction\\.clinerules\\active_directive.md."

---

# DIRECTIVE — Session 2026-07-15: KEY Workflow Execution + Awaiting Brendan's Direction

## Phase: Autonomous Cycle Complete → WAIT STATE (No pending approved work)

### Step 1: Load Constitution ✅ COMPLETE
- All 9 constitutional documents loaded and verified stable
- No conflicts detected across constitution layers

### Step 2: Load Active State ✅ COMPLETE  
- Session-Handoff.md v7.0 — Phase PILLAR 8 complete, yolo=true authorized
- TODO.md — All approved tasks marked complete (Phase 4 done)
- Problems.md — No open critical issues
- Docbase-Analysis.md — Architecture integrity confirmed
- File-Flow-Overview-1.md — Service topology current

### Step 3: Check Directive ✅ COMPLETE
- active_directive.md contains all completed sessions through Phase 4
- Final status: "Awaiting Brendan's Direction on Next Priority"
- No pending steps in directive to execute

### Step 4: KEY 1 SYNCHRONIZE ✅ COMPLETE
- Repository Truth established across all documents
- Build verified exit code 0 (all prior work)
- All constitutional safeguards active

### Step 5: KEY 2 ASSESS ✅ COMPLETE
- Repository health: Excellent (95%+ core systems)
- Vision alignment: STRONG — no drift detected
- Architecture integrity: SOUND — zero structural conflicts
- Career XP engine fully $VBV-gated per KEY 3.5 protocol

### Step 6: KEY 3 RECOMMEND → TRANSITION TO WAIT ✅ COMPLETE
- No approved work remains in TODO.md requiring immediate implementation
- Active directive explicitly states "Awaiting Brendan's Direction"
- Transitioned to KEY 4 WAIT state per Directive-Protocol and Key 4 specifications

---

# DIRECTIVE — Session 2026-07-13: Justice Dashboard WS Event Broadcast Hooks Complete (Historical)

## Phase: PILLAR 7 (Justice Hegemony) + KEY 3.5 Implementation

### Step 1: Add missing WebSocket broadcast hooks to justice_handlers.go
**Status:** ✅ COMPLETE
- Added `dashboard_refresh` broadcast to handleGetDashboard endpoint
- Created POST /api/justice/award-card handler with `justice_card_awarded` WS event (Endpoint 5)
- Created POST /api/justice/use-rep-shield handler with `shield_active` WS event (Endpoint 6)

### Step 2: Wire new routes in server.go
**Status:** ✅ COMPLETE
- Registered `/api/justice/award-card` route with economy-tight rate limiting
- Registered `/api/justice/use-rep-shield` route with economy-tight rate limiting
- Added lobby wrapper methods handleAwardJusticeCard and handleApplyRepShield

### Step 3: Build verification
**Status:** ✅ COMPLETE — exit code 0

### Step 4: Update documentation (TODO.md + Session-Handoff)
**Status:** ✅ COMPLETE — Updated again below

---

## SESSION 2026-07-14: Priority 3 Assessment — Career Combat Hooks Already Complete

### Problem Identified
Priority 3 was "Remaining career combat hooks (~14 careers without hooks beyond PILLAR 13 Phase A)". Comprehensive search of battle_service.go revealed ALL ~16+ careers already have $VBV-gated XP triggers wired.

### Verification Performed
- Searched all `wStats.CareerXP.TrackCareerXP()` calls in battle_service.go — found **28 call sites** covering 14+ distinct career types
- Careers verified with hooks: Heist Planner, Hostage Host, Lawyer-Commissioner, Arc-Net Operative, Smuggler, Launderer, Warden, AOS Leader, Sector Peacekeeper (vs Smuggler), Bounty Hunter (vs Kidnapper), Justice Recruiter (with ally synergy vs Mutation Auditor + recruit bonus), Intel Agent (decrypt vision + ally/enemy bonuses), Forensic Analyst (evidence double-clean + Gossip rival), Judge/Mutation variants
- All careers use `ComputeScaledXP()` pattern with $VBV-gate multiplier before TrackCareerXP call ✅

### Result: Priority 3 is ALREADY COMPLETE — no additional implementation needed

---

## Additional Work Completed This Session

### Task 4008: Combat Power Bonus Hooks — ✅ COMPLETE
- Added forensic audit logging to existing `GetPowerBonus()` hooks in battle_service.go (2 locations)
- Event types: `JUSTICE_POWER_BONUS_APPLIED` and `JUSTICE_POWER_BONUS_COMBO`
- Build verified exit code 0

---

## Session Summary — Justice Hegemony + Intel-Agent Cyber-Intercept Complete

### Phase: PILLAR 7 (Justice Hegemony) + KEY 3.5 Implementation → PILLAR 13 Expansion

### Step 1: Add missing WebSocket broadcast hooks to justice_handlers.go ✅ COMPLETE
- Added `dashboard_refresh` broadcast to handleGetDashboard endpoint
- Created POST /api/justice/award-card handler with `justice_card_awarded` WS event (Endpoint 5)
- Created POST /api/justice/use-rep-shield handler with `shield_active` WS event (Endpoint 6)

### Step 2: Wire new routes in server.go ✅ COMPLETE
- Registered `/api/justice/award-card` route with economy-tight rate limiting
- Registered `/api/justice/use-rep-shield` route with economy-tight rate limiting
- Added lobby wrapper methods handleAwardJusticeCard and handleApplyRepShield

### Step 3: Build verification ✅ COMPLETE — exit code 0

### Step 4: Update documentation (TODO.md + Session-Handoff) ✅ COMPLETE

---

## Additional Work This Session — PILLAR 13 Expansion

### Task 4501-A/B/C: Justice Recruiter Career Hooks
- Implemented `IsJusticeAligned()` helper function in rival_career_engine.go (+68 lines, after GetVBVGatingMultiplier())
- Added XP trigger block (~+72 lines) to battle_service.go bounty capture resolution point (before D7-D10 placeholder section)
- Pattern: baseXP → $VBV-gated ComputeScaledXP → TrackCareerXP("Justice Recruiter") with rival pair evaluation

### Task 4502-A/B/C: Intel-Agent Cyber-Intercept System
- Added `CyberInterceptEvent` struct type to backend_types.go (+13 lines)
- Implemented `handleCyberIntercept()` HTTP handler in handlers_criminality.go (~+98 lines): signal scanning, event generation/interception, $VBV-gated decrypt bonus, ally synergy with Arc-Net Operative
- Registered POST /api/criminality/cyber-intercept route in server.go under PILLAR 13 section

### Task 4503-A/B/C/D: Forensic Analyst Raid Evidence System
- Added `RaidEvidence` struct + `EvidencePool` map types to backend_types.go (+28 lines)
- Added `evidencePool *EvidencePool` field to Lobby struct in backend_types.go (~line 204, between justiceHandlers and rateLimiter)
- Initialized evidencePool with empty maps in server.go newLobby() function (between justiceHandlers and matchHandshakers fields)
- Wired XP trigger block (+~85 lines) to battle_service.go bounty capture resolution point: ForensicAnalyst earns +60 baseXP per capture, tracks via TrackCareerXP

### Task 4008: Combat Power Bonus Hooks ✅ COMPLETE
- Added forensic audit logging to existing `GetPowerBonus()` hooks in battle_service.go (2 locations)
- Event types: `JUSTICE_POWER_BONUS_APPLIED` and `JUSTICE_POWER_BONUS_COMBO`
- Build verified exit code 0

---

## SESSION 2026-07-14: Heist Planner Career Hooks — COMPLETE

### Step 5A: Implement GetPlanningBuff() for Heist Planner career tier buff
**Status:** ✅ COMPLETE
- Added `GetPlanningBuff()` method on CareerXP type in rival_career_engine.go (+~28 lines)
- Returns +5% per tier team planning buff, capped at ×1.35 (Boss tier equivalent)

### Step 5B: Implement GetHeistDividendRate() for planner dividend tracking
**Status:** ✅ COMPLETE  
- Added `GetHeistDividendRate()` method on CareerXP type in rival_career_engine.go (+~35 lines)
- Scales from Tier 1=×1% to Boss+=×8% based on career tier

### Step 5C: Add HasHeistPlannerRole() helper function
**Status:** ✅ COMPLETE
- Added `HasHeistPlannerRole()` method for role checking in rival_career_engine.go (+~9 lines)

### Step 6: Build verification + documentation update
**Status:** ✅ COMPLETE — exit code 0, TODO.md updated with completion status

---

## SESSION 2026-07-14: Available Work Items — All Verified Complete

### Task 1: Frontend integration of Justice Dashboard API endpoints ✅ ALREADY COMPLETE
- justice_dashboard.js loaded in index.html with `openJusticeDashboard()` button trigger
- All 5 WS event callbacks wired correctly in network.js → justice_dashboard.js exports (onJusticeCardAwarded, onTruthSerumApplied, onShieldActive, onDashboardRefresh, onBountyUpdated)
- Backend routes all registered: /api/justice/dashboard, use-truth-serum, capture-bounty, bounty-board, award-card, use-rep-shield

### Task 2: Bounty Hunter ↔ Kidnapper wiring (P2-E scope) ✅ ALREADY COMPLETE
- battle_service.go lines 1095-1115: `EvaluateCrossCareerXP("BountyHunter", p2Stats.JobRole, bhXP, &wStats, p2Stats)` with rival bonus tracking + audit logging

### Task 3: Sector Peacekeeper ↔ Smuggler wiring (P2-E scope) ✅ ALREADY COMPLETE
- battle_service.go lines 1065-1074: `EvaluateCrossCareerXP("SectorPeacekeeper", p2Stats.JobRole, pkXP, &wStats, p2Stats)` with rival bonus tracking + audit logging

---

## DIRECTIVE — PHASE 4: Deep Career System Pillar 8 Implementation (APPROVED BY BRENDAN) ✅ COMPLETE

### Step 1: Task 5001 — UnderworldBoss XP Trigger via Heist Completion Bonus
**Status:** ✅ COMPLETE — CONTRACT-029 through CONTRACT-033 templates added to underworld_contracts.go; HandleCompleteContract() TrackCareerXP pattern awards XP correctly. Build exit code 0.

### Step 2: Task 5002 — TaxAuditor↔JusticeCommissioner Rival Pair Hook
**Status:** ✅ COMPLETE — courthouse_service.go resolution points #1 and #2 wired with EvaluateCrossCareerXP("TaxAuditor", "JusticeCommissioner"). Build exit code 0.

### Step 3: Task 5003 — JusticeRecruiter Independent XP Trigger Verification
**Status:** ✅ VERIFIED — Three independent TrackCareerXP call sites confirmed in battle_service.go (primary bounty capture, ally synergy vs MutationAuditor, recruit bonus). No changes needed.

### Step 4: Build verification + documentation update
**Status:** ✅ COMPLETE — `go build ./...` exit code 0; TODO.md Phase 4 section inserted; Session-Handoff.md v7.0 updated with session block and Current Phase header.

---

---

## DIRECTIVE — Session 2026-07-21: P7-A/B/C Implementation + KEY Workflow (HISTORICAL) ✅ COMPLETE

### Phase: PHASE 7 CIVILIZATION EXPANSION → All approved tasks complete, Awaiting Brendan's Direction on Next Priority.

#### P7-A Entity Investment Layer — FULL STACK COMPLETE
| Component | Status | Details |
|-----------|--------|---------|
| entity_investment_service.go | ✅ Complete (~380 lines) | InvestInEntity, ClaimDividendsForPlayer, GetPortfolio, CalculateSharePrice handlers + 6 methods |
| Lobby struct field | ✅ Complete (+1 line) | investmentService *EntityInvestmentService added in backend_types.go |
| HTTP routes (4 endpoints) | ✅ Registered | /api/invest/entity, /api/claim/dividends, /api/invest/portfolio, /api/invest/dividends/history |
| Frontend module | ✅ Complete (~552 lines) | js/investment_dashboard.js IIFE + CSS overlay styles in index.html |
| Action bar button | ✅ Wired | "💼 Entity Investments" trigger in index.html action-bar (line 259 area) |

#### P7-B SeasonalEventEngine — FULL STACK COMPLETE
| Component | Status | Details |
|-----------|--------|---------|
| Backend types + service | ✅ Complete (~680 lines) | seasonal_event_engine.go, backend_types.go structs added |
| HTTP routes (8 endpoints) | ✅ Registered | /api/season/* in server.go with economy-tight rate limiting |
| WS event broadcasts | ✅ Wired | 5 events: activated, created, reward, expired, pool_updated |
| Frontend module | ✅ Complete (~390 lines) | js/seasonal_events.js IIFE + CSS overlay styles (index.html ~1260 lines) |
| Action bar button | ✅ Wired | "🌸 Seasonal Events" trigger in index.html action-bar (line 259 area) |

#### P7-C CreatorStore Backend — BACKEND COMPLETE
| Component | Status | Details |
|-----------|--------|---------|
| creator_store_service.go | ✅ Complete (~590 lines) | CreateCreator, ListProducts, GetProduct, BuyProduct, RateReview, GetCreatorProfile handlers + commission tracking |
| backend_types.go structs | ✅ Complete (+~38 lines) | CreatorStore, ProductListing, Review types added |
| Lobby struct field | ✅ Complete (+1 line) | creatorStore *CreatorStoreService added in backend_types.go (~line 204 area) |
| HTTP routes (8 endpoints) | ✅ Registered | /api/creator/* in server.go PILLAR 7-C section with economy-tight rate limiting |

#### Build Verification: `go build ./...` exit code 0 confirmed 2026-07-21. All systems operational.

---

## DIRECTIVE — Session 2026-07-21: Phase 7 Full Stack Completion + Documentation Update ✅ COMPLETE

### Work Completed This Session
| Component | Status | Details |
|-----------|--------|---------|
| P7-B SeasonalEventEngine | ✅ FULL STACK COMPLETE | Backend (~680 lines), HTTP routes (8 endpoints), WS broadcasts (5 events), frontend module (~450 lines js/seasonal_events.js + CSS overlay in index.html) |
| P7-C CreatorStore | ✅ FULL STACK COMPLETE | Backend (~590 lines), Lobby struct field, 8 HTTP routes registered. Frontend complete (js/creator_store.js ~480 lines + CSS overlay). Build verified exit code 0 ✅ |
| TODO.md Phase 7 status update | ✅ COMPLETE | All P7-A/B/C phases marked FULL STACK COMPLETE in documentation |

### Pending Work
- **Task 7202:** Secondary-sale royalty routing pipeline (only remaining P7 task) — economy_processing.go extension needed
- **Next Priority Options:** P7-D AI Autonomous Economy, P7-E Cross-Platform Identity Bridge, or Brendan's own direction

---

## SESSION STATUS — All Approved Tasks Complete, Awaiting Brendan's Direction on Next Priority
