# AI SYSTEM PROTOCOL & BEHAVIOR RULES (V2.0)

## 1. CORE MISSION
Maintain the absolute integrity of the **Virtualbabes Arena Social Economic Simulation**. Every change must respect the **Industrial Loop** (Ledger circularity) and **Dual-Target Build Synergy** (Go/WASM parity).

## 2. LEDGER & PRECISION PROTOCOL
*   **Integer Supremacy**: All virtual balance adjustments, taxes, and fees MUST utilize micro-unit integer math (`uint64`). Floating point (`float64`) is reserved for display UI or final scaling ratios ONLY.
*   **Industrial Seal**: No funds may be "burned" or "created" within virtual transfers. Remainders must be routed to the Faucet or specific Club Treasuries.

## 3. ARCHITECTURAL BOUNDARIES
*   **Modular Authority**: The Frontend orchestrator (`app.js`) MUST delegate domain logic to specialized modules. Never duplicate logic between `economy.js` and `criminality.js`.
*   **Switchboard Security**: Private keys remain server-side. Clients provide signatures of unique nonces provided by the server.

## 4. PERSISTENCE & RECOVERY
*   **Blockchain-Native**: AUTHORITATIVE state must be reconstructible from blockchain notes. Local caches (DATA_DIR) are for forensics and performance, not ground truth.
*   **State Snapshots**: All state-save operations (`VBT_ECONOMY_SNAPSHOT`, etc.) must perform JSON marshaling WITHIN the global mutex lock to prevent concurrent map access panics.

## 5. RE-ALIGNMENT & AUDIT
*   **Context over Guessing**: Utilize `DIR.md` and `File-Flow-Overview-1.md` to establish accurate topology. 
*   **Focused Commits**: Restrict logic changes to one file at a time to prevent context drift.
*   **Audit Trail**: Every significant change or audit MUST be logged in `AI-Brain/A.I_memory.md` with a unique task ID.

## 6. PROMPT DISCIPLINE
*   Always provide two relevant, brief prompt suggestions for the next logical step in development or hardening.