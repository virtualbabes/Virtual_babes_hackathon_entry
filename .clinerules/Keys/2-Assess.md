Version: 4.0
Status: Current
Last Amended: 2026-09-20

# KEY 2 — ASSESS

> Evaluate the synchronized repository. Assess, do not recommend, do not implement.

---

# OBJECTIVE

Establish repository understanding against **measured state**, and identify what is worth doing next.

---

# ASSESS AGAINST THE SWEEP (not against memory)

1. **`AI-Brain/RAG/17_aspects_flow.md`** — the 39 aspect rows. Each carries OWNER FILES · ROUTES · STATE · FLOW ·
   UI · **GAPS** · STATUS. This is the per-aspect assessment; read the row before forming an opinion on its aspect.
2. **`AI-Brain/App-Aspect-Index.md`** — the index, and its **Player View** section: assess player value from what the
   app actually offers a player, not from an internal feeling about the code.
3. `AI-Brain/Problems.md` — actionable defects (each one is a gap a row already names).
4. `AI-Brain/ToDo.md` — plans and their order.
5. `AI-Brain/MASTER-PLAN.md` §0 — progress table. **Its §6 rows are stale; the aspect rows outrank them.**

> **Referenced ≠ reachable** (the lesson of 2026-09-15 d), and **a green reading is not a measurement**. If a document
> claims something, name the file and line that proves it — or record the document itself as a finding, because this
> repository has repeatedly found docs and comments that name files and functions **that do not exist**.

---

# THE FOUR LENSES (with this repository's known defect classes attached)

### Vision
Alignment with the Design Constitution; drift; anything that would make a player-facing promise the engine cannot keep.

### Architecture — check the classes the sweep proved are live here
- **Money doors that move no money** (`BuyShares`, `BackLaunch`, `ConfirmBridge`): a price tag with no debit or credit.
- **Unauthenticated doors** (compliance record/resolve/escalate): any caller naming any wallet or severity.
- **Client-declared authority** (industrial-loop phase/amount, ritual cost, governance weight, identity impact):
  a metric the world is priced from, accepted from the body.
- **Package-level globals owning recorded state** outside the Lobby and outside every record family (eleven found).
- **Floats on ledger paths** (fee splits, prices, XP): the Ledger permits floats for display only.
- **Signedness and index arithmetic** (`uint64(signed)`; `wallet[len-8:]`) — inversions and request-reachable panics.
- **Ownership**: one owner per fact; the aliased map (`marketNodes` IS the router's) is the precedent.
- Split-WASM discipline: a server file carries `//go:build !js && !wasm`; the console entrypoint is `//go:build console`.

### Development
Implementation progress vs the aspect rows; open gaps; documentation drift; and the **frontend wiring** facts the gates
already measure (dead handlers, orphan modules, unrouted features, overlays that cannot render).

### Player Value
Agency · emergent gameplay · long-term engagement · Year-Five value. Judge a mechanic by whether it interacts.

---

# METHOD (this is what makes an assessment credible)

- **Measure before asserting.** Run the gates; open the file:line. Never generalise from one sighting.
- **Prefer the row to the recollection.** If a row and a document disagree, the row wins until measured; if the code and
  both disagree, the code wins and the row is corrected in the same unit.
- **Report what is NOT known.** "Not measured" is a legitimate finding; a confident sentence covering unread code is not.
- **Count, then conclude.** A census (`grep`/gate) beats an impression; a truncated `-First N` is not a census.

---

# FINDINGS

Record: major strengths · major concerns · highest risks · highest opportunities — **prioritised by leverage, not by
count**, and each with the file/line or gate output that supports it.

---

# REPORT

Report: repository health · vision alignment · strengths · concerns · risks · opportunities.
Flow to **KEY 3 — RECOMMEND**. Propose no solutions here.
