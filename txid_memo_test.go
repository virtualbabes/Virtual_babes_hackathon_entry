//go:build !js && !wasm

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ════════════════════════════════════════════════════════════════════════════
// THE UNIFORM TXID MEMO (txid_memo.go) — one idempotency door for every money
// door, and the two deadlock regressions it had to fix to be placeable.
// ════════════════════════════════════════════════════════════════════════════

func TestTxIDMemoClaimsCommitsAndReleases(t *testing.T) {
	l := &Lobby{}

	// 1. A fresh id is claimable.
	key, reason := l.claimTxID("TX-A")
	if key != "TX-A" || reason != "" {
		t.Fatalf("claimTxID(TX-A) = (%q, %q); want the claim", key, reason)
	}

	// 2. While it is in flight a second submit is refused — this is the window
	//    that used to be open (check and record lived in separate lock windows).
	if _, reason := l.claimTxID("TX-A"); reason != TxIDInFlight {
		t.Errorf("second claim reason = %q; want %q", reason, TxIDInFlight)
	}

	// 3. A release frees it: verification failed, so nothing was applied.
	l.releaseTxID(key)
	if key2, reason := l.claimTxID("TX-A"); key2 != "TX-A" || reason != "" {
		t.Errorf("after release the id must be claimable again, got (%q, %q)", key2, reason)
	}

	// 4. A commit consumes it permanently, recording the ON-CHAIN time.
	onchain := time.Unix(1700000000, 0)
	l.commitTxID(key, onchain)
	if _, reason := l.claimTxID("TX-A"); reason != TxIDAlreadyUtilized {
		t.Errorf("a committed id must be refused with %q, got %q", TxIDAlreadyUtilized, reason)
	}
	if got := l.registeredTxIDs["TX-A"]; !got.Equal(onchain) {
		t.Errorf("committed time = %v; want %v", got, onchain)
	}
	if _, inFlight := l.pendingTxIDs["TX-A"]; inFlight {
		t.Error("a committed id must not remain in flight")
	}

	// 5. Committing is idempotent and never duplicates.
	l.commitTxID(key, onchain)
	if stats := l.txidMemoStats(); stats.Committed != 1 || stats.InFlight != 0 {
		t.Errorf("memo stats = %+v; want 1 committed, 0 in flight", stats)
	}
}

func TestTxIDMemoFailsClosed(t *testing.T) {
	l := &Lobby{}

	// An empty id must be REFUSED, never skipped: skipping would make a door
	// replayable, and every empty id would share one memo slot.
	for _, txid := range []string{"", "   ", "\t"} {
		if key, reason := l.claimTxID(txid); key != "" || reason != TxIDRequired {
			t.Errorf("claimTxID(%q) = (%q, %q); want (\"\", %q)", txid, key, reason, TxIDRequired)
		}
	}

	// release/commit on a non-id are no-ops, not panics.
	l.releaseTxID("")
	l.commitTxID("", time.Time{})
	if stats := l.txidMemoStats(); stats.Committed != 0 {
		t.Errorf("an empty id was recorded: %+v", stats)
	}

	// A zero time commits as NOW, never as the zero instant.
	l.commitTxID("TX-Z", time.Time{})
	if got := l.registeredTxIDs["TX-Z"]; got.IsZero() {
		t.Error("commitTxID recorded the zero time instead of now")
	}
}

func TestTxIDMemoLockedVariants(t *testing.T) {
	l := &Lobby{}

	// The Locked forms are used INSIDE an existing critical section (a door that
	// wants the memo write in the same lock as the money write). Calling the
	// self-locking form there would deadlock, so both exist deliberately.
	l.mutex.Lock()
	l.pendingTxIDs = map[string]time.Time{"TX-L": time.Now()}
	l.commitTxIDLocked("TX-L", time.Unix(1700000001, 0))
	if _, inFlight := l.pendingTxIDs["TX-L"]; inFlight {
		t.Error("commitTxIDLocked left the id in flight")
	}
	if got := l.registeredTxIDs["TX-L"]; !got.Equal(time.Unix(1700000001, 0)) {
		t.Errorf("commitTxIDLocked recorded %v", got)
	}

	l.pendingTxIDs["TX-M"] = time.Now()
	l.releaseTxIDLocked("TX-M")
	if _, inFlight := l.pendingTxIDs["TX-M"]; inFlight {
		t.Error("releaseTxIDLocked did not drop the reservation")
	}
	l.mutex.Unlock()

	// isTxIDConsumed sees both sets.
	if !l.isTxIDConsumed("TX-L") {
		t.Error("a committed id must read as consumed")
	}
	l.mutex.Lock()
	l.pendingTxIDs["TX-N"] = time.Now()
	l.mutex.Unlock()
	if !l.isTxIDConsumed("TX-N") {
		t.Error("an in-flight id must read as consumed")
	}
	if l.isTxIDConsumed("TX-NOPE") {
		t.Error("an unknown id must not read as consumed")
	}
	if l.isTxIDConsumed("") {
		t.Error("the empty id must not read as consumed")
	}
}

// ---------------------------------------------------------------------------
// DEADLOCK REGRESSIONS (Problems.md §15)
//
// VerifyBuyInTransaction takes l.mutex.RLock() internally. A money door that
// holds the WRITE lock across that call deadlocks the ENTIRE server — not just
// that request. Two doors did exactly that (handleBailCard, which additionally
// took a read lock while holding the write lock, and HandleRepayLoan). These
// tests fail by TIMEOUT rather than by assertion, because a deadlock raises no
// error at all; it just never returns.
// ---------------------------------------------------------------------------

const testPayerWallet = "0xpayer"

func runWithWatchdog(t *testing.T, name string, d time.Duration, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("%s did not return within %s: it is holding the lobby write lock across an oracle call. "+
			"VerifyBuyInTransaction takes l.mutex.RLock(), so a caller holding the write lock deadlocks the whole server.", name, d)
	}
}

// assertLobbyLockIsFree proves the write lock is not still held. A deadlock
// leaves no error behind — only a lock nobody will ever release.
func assertLobbyLockIsFree(t *testing.T, l *Lobby, what string) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		l.mutex.Lock()
		l.mutex.Unlock()
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("%s: the lobby write lock is still held after the door returned (deadlock)", what)
	}
}

// unreachableIndexerLobby builds the minimum a money door needs, with an indexer
// endpoint that REFUSES connections so the oracle round-trip fails fast.
func unreachableIndexerLobby() *Lobby {
	l := &Lobby{
		wallets:         map[string]string{"c1": testPayerWallet},
		clients:         map[string]*Client{},
		clubs:           map[string]*Club{},
		leaderboard:     map[string]PlayerStats{},
		loans:           map[string]*Loan{},
		registeredTxIDs: map[string]time.Time{},
		pendingTxIDs:    map[string]time.Time{},
		vaultAddress:    "VAULTADDRESS",
		oracleService:   &OracleService{},
		availableNetworks: map[string]NetworkConfig{
			"Voi Mainnet": {
				NetworkName: "Voi Mainnet",
				AssetID:     "1",
				IndexerURLs: []string{"http://127.0.0.1:1"},
			},
		},
	}
	// A jailed card the payer owns, so the bail door passes validation.
	l.clubs["CLUB-1"] = &Club{ID: "CLUB-1", Name: "Test Club", Jail: map[int]ServerCard{7: {}}}
	l.leaderboard[testPayerWallet] = PlayerStats{JailedCards: map[int]string{7: "CLUB-1"}}
	return l
}

func TestBailDoorDoesNotHoldTheWriteLockAcrossTheOracle(t *testing.T) {
	l := unreachableIndexerLobby()
	env := &Envelope{
		Type:    "bail_card",
		FromID:  "c1",
		Payload: json.RawMessage(`{"card_id":7,"club_id":"CLUB-1","txid":"TX-BAIL-1","network":"VOI"}`),
	}

	runWithWatchdog(t, "handleBailCard", 20*time.Second, func() { l.handleBailCard(env) })
	assertLobbyLockIsFree(t, l, "handleBailCard")

	// Verification could not succeed against an unreachable indexer, so NOTHING
	// may have been applied.
	if _, stillJailed := l.clubs["CLUB-1"].Jail[7]; !stillJailed {
		t.Error("a bail whose verification failed must not release the card")
	}
	if l.isTxIDConsumed("TX-BAIL-1") {
		t.Error("a bail whose verification FAILED must not consume the transaction id")
	}
	if stats := l.txidMemoStats(); stats.InFlight != 0 {
		t.Errorf("the in-flight reservation leaked: %+v", stats)
	}
}

func TestRepayLoanDoorDoesNotHoldTheWriteLockAcrossTheOracle(t *testing.T) {
	l := unreachableIndexerLobby()
	l.loans["L1"] = &Loan{
		ID: "L1", BorrowerWallet: testPayerWallet, Status: "active",
		LoanAmount: 900000, RepaymentAmount: 1000000,
	}

	s := &LoanService{}
	body := `{"loan_id":"L1","wallet":"0xpayer","txid":"TX-LOAN-1","network":"VOI"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/loans/repay", strings.NewReader(body))

	runWithWatchdog(t, "HandleRepayLoan", 20*time.Second, func() { s.HandleRepayLoan(l, rec, req) })
	assertLobbyLockIsFree(t, l, "HandleRepayLoan")

	if rec.Code != http.StatusPaymentRequired {
		t.Errorf("status = %d; want %d (the indexer is unreachable, so verification must fail)", rec.Code, http.StatusPaymentRequired)
	}
	if _, stillThere := l.loans["L1"]; !stillThere {
		t.Error("a repayment whose verification failed must not delete the loan")
	}
	if l.isTxIDConsumed("TX-LOAN-1") {
		t.Error("a repayment whose verification FAILED must not consume the transaction id")
	}
}
