//go:build !js && !wasm

package main

import (
	"strings"
	"time"
)

// ════════════════════════════════════════════════════════════════════════════
// UNIFORM TRANSACTION-ID MEMO — the ONE idempotency door.
//
// Every money door must answer the same question the same way: "has this
// on-chain transaction already been consumed?" Before this file the answer was
// consulted at only 4 of the 8 doors that call VerifyBuyInTransaction, and even
// there the check and the record lived in TWO separate lock windows with the
// (slow) indexer lookup in between — so two concurrent double-submits could both
// pass verification and both apply (double career XP, a double treasury credit,
// a doubled inventory count).
//
// The contract implemented here:
//
//  1. claimTxID   — atomically RESERVE the id BEFORE chain verification,
//                   checking the committed set AND the in-flight set.
//  2. verify      — the (slow) indexer lookup, with NO lobby lock held.
//  3. commitTxID  — on SUCCESS ONLY, move the reservation into the committed set
//                   (the set that is snapshotted), recording the on-chain time.
//  4. releaseTxID — on failure, drop the reservation, so a genuine retry (a
//                   freshly signed transaction with a NEW id) is never blocked.
//
// LOCK DISCIPLINE (binding): claimTxID/releaseTxID/commitTxID each take
// `l.mutex` themselves, so a caller must NOT hold it. In particular no money
// door may hold the lobby write lock across VerifyBuyInTransaction —
// VerifyBuyInTransaction takes `l.mutex.RLock()` internally, so a caller holding
// the write lock deadlocks the WHOLE SERVER (see Problems.md §15).
//
// PERSISTENCE: only `registeredTxIDs` is persisted. A reservation is a fact
// about THIS process, so a restart mid-verification leaves no trace — and
// nothing was applied. "Record only on successful apply" therefore holds across
// restarts without any extra state.
//
// FAIL CLOSED: an empty transaction id is REFUSED rather than skipped. Skipping
// it would make an unclaimed door replayable; and because
// `strings.HasPrefix(note, "")` is always true, an empty purpose prefix would
// break purpose binding entirely (guarded separately in VerifyBuyInTransaction).
// ════════════════════════════════════════════════════════════════════════════

// TxIDAlreadyUtilized is the ONE refusal message a door returns when a
// transaction id has already been consumed. Doors render this rather than
// inventing their own wording.
const TxIDAlreadyUtilized = "Transaction ID already utilized"

// TxIDInFlight is returned while another request holds the reservation.
const TxIDInFlight = "Transaction ID is already being processed"

// TxIDRequired is returned for an empty id (fail closed, never skip).
const TxIDRequired = "Transaction ID is required"

// TxIDMemoStats reports the two sets. Read-only; used by tests and diagnostics.
type TxIDMemoStats struct {
	Committed int `json:"committed"`
	InFlight  int `json:"in_flight"`
}

// claimTxID atomically reserves txid for the caller.
//
// Returns (txidKey, "") when the caller owns the reservation, or
// ("", reason) when it does not. A non-empty first return value MUST be paired
// with exactly one releaseTxID (failure) or commitTxID (success).
func (l *Lobby) claimTxID(txid string) (string, string) {
	txid = strings.TrimSpace(txid)
	if txid == "" {
		return "", TxIDRequired
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.registeredTxIDs == nil {
		l.registeredTxIDs = make(map[string]time.Time)
	}
	if l.pendingTxIDs == nil {
		l.pendingTxIDs = make(map[string]time.Time)
	}

	if _, used := l.registeredTxIDs[txid]; used {
		return "", TxIDAlreadyUtilized
	}
	if _, inFlight := l.pendingTxIDs[txid]; inFlight {
		return "", TxIDInFlight
	}

	l.pendingTxIDs[txid] = time.Now()
	return txid, ""
}

// releaseTxID drops a reservation after a FAILED verification or a refused
// apply. Nothing was written, so nothing is recorded — and the same id may be
// retried.
func (l *Lobby) releaseTxID(txid string) {
	if txid == "" {
		return
	}
	l.mutex.Lock()
	defer l.mutex.Unlock()
	delete(l.pendingTxIDs, txid)
}

// commitTxID records a successfully applied money door. `when` is the on-chain
// transaction time; a zero value is recorded as now.
//
// Committing is idempotent: re-committing the same id overwrites the timestamp
// rather than creating a second entry.
func (l *Lobby) commitTxID(txid string, when time.Time) {
	if txid == "" {
		return
	}
	if when.IsZero() {
		when = time.Now()
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.registeredTxIDs == nil {
		l.registeredTxIDs = make(map[string]time.Time)
	}
	delete(l.pendingTxIDs, txid)
	l.registeredTxIDs[txid] = when
}

// isTxIDConsumed reports whether an id is committed or in flight. Read-only
// helper for diagnostics; doors use claimTxID, never this.
func (l *Lobby) isTxIDConsumed(txid string) bool {
	if txid == "" {
		return false
	}
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	if _, used := l.registeredTxIDs[txid]; used {
		return true
	}
	if _, inFlight := l.pendingTxIDs[txid]; inFlight {
		return true
	}
	return false
}

// ── LOCKED VARIANTS ────────────────────────────────────────────────────────
// For a door that already holds the write lock and wants the memo write to be
// part of the SAME critical section that applies the money. Calling the
// self-locking form there would deadlock, so the discipline is explicit:
// `commitTxIDLocked` / `releaseTxIDLocked` REQUIRE l.mutex to be held.

// commitTxIDLocked records a successfully applied door. Caller MUST hold l.mutex.
func (l *Lobby) commitTxIDLocked(txid string, when time.Time) {
	if txid == "" {
		return
	}
	if when.IsZero() {
		when = time.Now()
	}
	if l.registeredTxIDs == nil {
		l.registeredTxIDs = make(map[string]time.Time)
	}
	delete(l.pendingTxIDs, txid)
	l.registeredTxIDs[txid] = when
}

// releaseTxIDLocked drops a reservation after a refused apply.
// Caller MUST hold l.mutex.
func (l *Lobby) releaseTxIDLocked(txid string) {
	if txid == "" {
		return
	}
	delete(l.pendingTxIDs, txid)
}

// txidMemoStats snapshots the memo for tests and diagnostics.
func (l *Lobby) txidMemoStats() TxIDMemoStats {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return TxIDMemoStats{Committed: len(l.registeredTxIDs), InFlight: len(l.pendingTxIDs)}
}
