Version: 3.0
Status: Stable
Last Amended: 2026-06-21

# Repository Truth

> **The repository is the product.**
>
> Documentation explains it.
>
> Repository Truth prevails.

---

# PURPOSE

Establish repository truth whenever implementation, documentation, or historical context disagree.

The objective is one consistent understanding of the project.

---

# TRUTH HIERARCHY

Determine project state in this order:

1. Repository implementation
2. Repository Constitution (`.clinerules`)
3. `AI-Brain/Session-Handoff.md`
4. `AI-Brain/Docbase-Analysis.md`
5. README.md
6. Current documentation
7. Read last 50 lines `AI-Brain\A.I_memory.md`
8. Conversational context

Higher authority overrides lower authority.

---

# CONFLICT RESOLUTION

```
Conflict
    │
    ▼
Investigate
    │
    ▼
Implementation intentional?
    │
 ┌──┴──┐
 │     │
Yes    No
 │     │
 ▼     ▼
Update Correct
Docs   Code
    │
    ▼
Restore Repository Truth
```

Never assume code or documentation is correct.

Verify first.

---

# INVESTIGATION

Use only the evidence required to resolve the conflict.

Possible sources:

* Repository implementation
* Constitution
* Session-Handoff
* Docbase Analysis
* Active documentation
* Project Vision

If uncertainty remains:

Pause.

Request clarification before structural changes.

---

# DRIFT

Documentation and implementation are equally fallible.

Determine:

* Was the implementation intentional?
* Is documentation outdated?
* Does current behaviour support the approved vision?
* Would changing it introduce regression?

Only then recommend corrective action.

---

# AUDIT

During analysis identify:

* Documentation drift
* Duplicate implementations
* Orphaned systems
* Ownership conflicts
* Abandoned architecture

Report findings before implementation.

---

# SESSION SYNCHRONIZATION

Every session begins by reconciling repository state.

Never rely on previous conversational context.

Verify first.

Continue second.

Avoid repeating completed work.

---

# PRINCIPLES

* Investigate before concluding.
* Explain before changing.
* Verify before implementing.
* Maintain one shared understanding.

---

# FINAL PRINCIPLE

Repository Truth is the objective.

Code builds the civilization.

Documentation explains it.

Both evolve together.