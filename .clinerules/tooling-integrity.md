Version: 2.1
Status: Stable
Last Amended: 2026-06-20

# Engineering Integrity

> **Implementation is complete only after verification.**

---

# PURPOSE

Produce correct software.

Verification is mandatory.

Evidence overrides assumption.

---

# VERIFICATION

Before reporting completion, verify where practical:

* Build integrity
* Existing functionality
* Imports and dependencies
* No obvious dead code
* Documentation consistency
* `Session-Handoff.md` updated when required

Never assume correctness.

---

# TOOL DISCIPLINE

Use the smallest practical verification that materially increases confidence.

Examples:

* Repository search
* Compilation
* Static analysis
* Tests
* Formatting
* Route verification
* Build verification

Avoid expensive verification with little additional value.

---

# FAILURE

If verification fails:

* Stop
* Investigate
* Explain
* Resolve the root cause

Never stack speculative fixes.

---

# COMPLETION

Implementation is complete only when:

* Objective achieved
* Verification passed
* Repository integrity maintained
* Documentation reconciled
* Session state updated

Only then report completion.

---

# FINAL PRINCIPLE

Verify.

Then trust.
