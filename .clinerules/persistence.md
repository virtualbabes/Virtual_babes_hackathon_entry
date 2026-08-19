Version: 2.1
Status: Stable
Last Amended: 2026-06-20

# Persistence & Recovery

> **Protect player history, economic integrity, and civilization continuity.**
>
> Recovery restores the civilization—it never redefines it.

---

# AUTHORITATIVE STATE

The blockchain is the authoritative ledger.

Persistent state must always be reconstructible from blockchain records.

Local storage (`DATA_DIR`) exists only for:

* Performance
* Startup
* Diagnostics
* Recovery
* Forensics

Caches are never authoritative.

---

# RECOVERY

Assume:

* Servers fail
* Storage is lost
* Nodes migrate
* Infrastructure changes

Recovery restores the existing civilization.

Never create a new one.

---

# SNAPSHOTS

Serialize shared mutable state only while holding the appropriate global mutex.

Examples:

* `VBT_ECONOMY_SNAPSHOT`
* Player registries
* Shared economic state

Never serialize mutable shared state outside synchronization.

Prevent:

* Concurrent map access
* Partial serialization
* Inconsistent recovery

---

# ANTI-MANIPULATION

Protected progression must survive reconnects.

Persist critical progression atomically to authoritative storage.

Client refreshes must never:

* Reset progression
* Duplicate rewards
* Bypass validation
* Circumvent gameplay

Persistence protects fairness.

---

# DETERMINISTIC RECOVERY

Identical authoritative inputs must always produce identical state.

Recovery must never depend on:

* Timing
* Execution order
* Client state
* Browser state
* Temporary caches

Recovery is deterministic.

---

# CIVILIZATION CONTINUITY

Protect long-term player history, including:

* Ownership
* Reputation
* Economic activity
* Organizations
* Businesses
* Political state
* Progression

Implementation evolves.

Player history endures.

---

# FINAL PRINCIPLE

Caches accelerate.

Persistence protects.

The blockchain remembers.

The civilization continues.