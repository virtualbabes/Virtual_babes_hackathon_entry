Version: 2.1
Status: Stable
Last Amended: 2026-06-22

# KEY 4 — WAIT

> Preserve repository awareness while awaiting further instruction or continuing autonomous execution.

---

# OBJECTIVE

The current phase has concluded.

Remain synchronized.

Maintain Repository Truth.

Await Brendan's instruction or continue the approved autonomous workflow.

---

# WAIT STATE

If the current handoff explicitly records a hold or pause:

* Treat the current recommendation as pending.
* Do not assume approval.
* Do not infer intent.
* Do not begin implementation.
* Clarify when uncertain.

If the current handoff explicitly records `yolo=true` authorization, or if there is no explicit hold and `AI-Brain\ToDo.md` contains active tasks:

* Review `.clinerules\Session-Handoff.md`.
* Review `AI-Brain\ToDo.md` for active tasks.
* Determine whether approved work remains.
* If approved work remains, proceed immediately to KEY 3 — RECOMMEND or KEY 3.5 — IMPLEMENT as appropriate.
* Do not reassess unless required.

---

# AUTONOMOUS FLOW

If the current handoff explicitly records `yolo=true` authorization, or if there is no explicit hold and `AI-Brain\ToDo.md` contains active tasks:

If approved recommendations remain:

Return to:

* `.clinerules\Keys\3-Recommend.md`

If no approved recommendations remain:

Continue the repository workflow:

* `.clinerules\Keys\1-Synchronize.md`
* `.clinerules\Keys\2-Assess.md`
* `.clinerules\Keys\3-Recommend.md`
* `.clinerules\Keys\3.5-Implement.md`
* `.clinerules\Keys\4-Wait.md`

* If `.clinerules\Session-Handoff.md` records `yolo=true` authorization or `AI-Brain\ToDo.md` contains active open tasks, return immediately to KEY 1 and continue the approved autonomous cycle.

---

# PRESERVE

Maintain awareness of:

* Repository state
* Current recommendation
* Active priorities
* Known blockers

Avoid unnecessary re-analysis.

Repository Truth remains authoritative.

---

# REASSESS

If Brendan provides:

* New requirements
* Repository changes
* Documentation updates
* Implementation results
* Architectural feedback

Determine whether Repository Truth may have changed.

If so:

Return to **KEY 1 — SYNCHRONIZE**.

---

# COMMUNICATION

Remain concise.

Respond only to:

* New instructions
* Requested clarification

Do not repeat completed analysis or recommendations.

---

# TRANSITION

If the current session handoff explicitly records a hold or pause:

Await Brendan's instruction.

If the current session handoff explicitly records `yolo=true` authorization, or if no explicit hold exists and `AI-Brain\ToDo.md` contains active tasks:

Continue the autonomous workflow until:

* No approved work remains.
* Repository Truth becomes uncertain.
* Human judgement is required.
* Verification fails.
* Constitutional conflict is encountered.

---

> Disciplined waiting preserves momentum.
