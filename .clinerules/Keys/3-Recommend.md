Version: 4.0
Status: Current
Last Amended: 2026-09-20

# KEY 3 — RECOMMEND

> Recommend the highest-leverage work, from measured state, with its own falsification pre-committed.

---

# OBJECTIVE

Select the highest-leverage implementation. Exercise independent engineering judgement. **Do not implement.**
Optimise for Year Five: systems over features, interconnectedness over expansion.

---

# EVALUATE

Prefer the unit that most improves **player value · architectural leverage · repository integrity ·
interconnectedness** — and prefer the one that turns an UNMEASURED region into a measured one, because a measured gap
can be fixed while an unmeasured one cannot be seen.

# EVERY RECOMMENDATION CARRIES A PLAN PROTOCOL (binding — this repository learned it the hard way)

State, BEFORE any code is written:

1. **The surfaces it touches**, each CITED from a read (file:line) — never from memory. A recommendation that asserts
   unread code as known has already failed here once.
2. **The ONE unknown that decides its shape**, and the read that resolves it (Phase 0). Never treat a
   design-determining unknown as an implementation detail.
3. **The test that MUST FAIL if the claim is false.** If nothing can fail, the recommendation proves nothing.
4. **The acceptance gate**: which of the existing gates/tests must still pass, and which new one it earns.
5. **Its failure modes**, each with a pre-committed counter (e.g. "a stub that does nothing is refused"; "the fix must
   do what the primary server does, not merely compile").

# VALIDATE — REJECT A RECOMMENDATION THAT

- duplicates an owner (`npm run verify:duplicates` is the judge) or introduces a parallel implementation;
- increases architectural debt, reduces future flexibility, or creates an isolated mechanic;
- conflicts with the Constitution: **no float on ledger paths** (uint64 micro only) · **no cloud models** (local
  Ollama/llama.cpp only) · **AI citizens own wallets** · contained overlays · **secrets ENV-ONLY**;
- **would create a FALSE CLAIM** — the repository's most expensive defect class: a served field nothing populates, a
  document naming a file that does not exist, a gate that cannot fail, a comment stating a rule the code does not
  implement. If the honest options are "build it properly" or "say it is not built", choose one of those;

  --- and never "wire it up so it looks done".

# STANDING OPERATOR CONSTRAINTS (2026-09-20 — binding, do not re-litigate)

- `RECORDS_DISPATCH` stays **OFF**; **no live/on-chain testing** until the UI is complete.
- `$NUGGET` / `$UNIT` ids are PLACEHOLDERS (operator-authorised) and a money door **REFUSES to move real value** while
  they are; `$VBV`/`$VOI` are named `40227315`.
- **The console has NO admin entry point and no crypto rail**: it is a virtual mirror awaiting a **manual bridge**.
- **The Nautilus DEX path is executed MANUALLY** ($VBV withdrawn as $VOI by the user).
- Console DLC buys land as **USDC in the admin wallet**; the admin tops the faucet up by hand — and
  **a CAP/LOCK is a PRECONDITION to that rail: it ships with the cap or it does not ship.**
- **GIT PUSH is Brendan's.** The agent commits nothing and pushes nothing.

# APPROVAL / DIRECTIVE GATE

- `yolo=true` (or no hold and active tasks in `AI-Brain/ToDo.md`) → authorised → **KEY 3.5**.
- Hold/pause recorded → await approval; do not implement.
- Before KEY 3.5, confirm `active_directive.md` authorises the scope; if it predates the work it MUST be replaced.

# REPORT

**Recommendation Ready** — the recommendation, why now, benefits, risks, systems affected, documents affected, its plan
protocol answers, and the acceptance gate. Then **exactly two next-step recommendations** (system-protocol OUTPUT rule).

> Architects create leverage, not activity.
