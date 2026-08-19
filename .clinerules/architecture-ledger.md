Version: 2.1
Status: Stable
Last Amended: 2026-06-20

# Architecture Ledger

> **Immutable engineering constraints.**
>
> Preserve deterministic simulation, economic integrity, and architectural consistency.

---

# LEDGER

## Deterministic Finance

* `uint64` integer micro-units only.
* Floating point prohibited for:
  * Balances
  * Transfers
  * Taxes
  * Rewards
  * Debts
  * Interest
* Floating point permitted for UI presentation only.

## Industrial Loop

Every transaction must reconcile.

* No silent minting.
* No silent burning.
* Route remainders to a deterministic sink:
  * Faucet
  * Treasury
  * Approved system sink

---

# DETERMINISM

Simulation must produce identical results across:

* Go Server
* WASM
* Browsers
* Platforms

Go is authoritative.

WASM mirrors gameplay.

No simulation drift.

---

# ARCHITECTURE

## Domain Ownership

One domain.

One responsibility.

One owner.

Business logic belongs to its domain.

Cross-domain interaction occurs through explicit interfaces.

The orchestrator coordinates.

Modules own behaviour.

---

## Frontend

Frontend responsibilities:

* Presentation
* Interaction
* Coordination

Business rules belong to domain modules.

---

## State Ownership

Each mutable state has one authoritative owner.

* No mirrored state.
* No duplicate synchronization.
* Inventory is authoritative.
* Reference assets.
* Never duplicate ownership.

---

## Services

* Single responsibility.
* No God services.
* No circular dependencies.
* Prefer composition.
* Extend before creating.

---

# SECURITY

Authority remains server-side.

Server:

* Private keys
* Validation
* Authorization

Client:

* Nonce
* Challenge
* Signature

Trust cryptographic proof only.

---

# EVOLUTION

Prefer:

* Extending existing systems
* Reducing technical debt
* Increasing interconnectedness

Avoid:

* Parallel implementations
* Duplicate logic
* Architectural drift

---

# SUCCESS METRICS

Architecture should continuously improve:

* Determinism
* Integrity
* Maintainability
* Interconnectedness
* Five-Year extensibility

Protect the civilization.