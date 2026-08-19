Version: 1.0
Status: Stable
Last Amended: 2026-06-20

# Decision Heuristics

> **Purpose**
>
> Guide professional engineering judgement when multiple valid solutions exist.
>
> These heuristics complement the Repository Constitution. They do not override it.

---

# PRINCIPLE

When several implementations are technically correct:

Choose the one that creates the greatest long-term value.

Optimize for **Year Five**, not immediate convenience.

---

# PREFERENCE ORDER

Prefer implementations that increase:

1. Repository truth
2. Vision alignment
3. Player agency
4. System interconnectedness
5. Determinism
6. Architectural cohesion
7. Maintainability
8. Reusability
9. Verification confidence
10. Future development velocity

---

# ARCHITECTURE

Prefer:

- Extend existing systems
- Composition over duplication
- Single responsibility
- Clear ownership
- Explicit interfaces
- Deterministic behaviour
- Fewer public APIs
- Lower coupling
- Higher cohesion

Avoid:

- Parallel implementations
- God services
- Hidden ownership
- Circular dependencies
- Duplicate business logic

---

# IMPLEMENTATION

Prefer implementations that:

- Solve the root problem
- Remove technical debt
- Reduce future complexity
- Reuse existing architecture
- Minimize surface area
- Preserve compatibility
- Require minimal documentation changes

Do not optimize for the fewest lines of code.

Optimize for clarity and longevity.

---

# PLAYER VALUE

When priorities conflict, prefer work that increases:

- Meaningful interaction
- Emergent gameplay
- Player freedom
- Discovery
- Long-term engagement
- Living-world depth

Avoid mechanics that exist in isolation.

---

# DOCUMENTATION

Prefer:

- Updating existing documents
- One source of truth
- Concise explanations
- Clear ownership

Avoid:

- Duplicate documentation
- Conflicting guidance
- Historical clutter

---

# VERIFICATION

Prefer evidence over assumption.

Run the smallest verification that provides sufficient confidence.

Increase verification as implementation scope increases.

---

# DECISION TEST

Before finalizing an implementation ask:

- Is there a simpler solution?
- Can an existing system be extended?
- Does this reduce future maintenance?
- Does ownership remain obvious?
- Does this improve the repository?
- Will this still be the preferred design in Year Five?

If not, reconsider.

---

# FINAL PRINCIPLE

Good engineering solves today's problem.

Great engineering makes tomorrow's problems easier.

Prefer decisions that compound.