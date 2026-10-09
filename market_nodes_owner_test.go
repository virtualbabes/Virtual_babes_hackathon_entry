//go:build !js && !wasm

package main

import (
	"testing"
	"time"
)

// ============================================================================
// THE MARKET-NODE MAP'S OWNER LOCK — pinned BEHAVIOURALLY.
// ============================================================================
//
// WHY A BEHAVIOURAL PIN, ON TOP OF THE STATIC GATE. `market_nodes_alias_gate_test.go` proves which lock a
// source position holds; it cannot prove the code RUNS, and it structurally cannot see a callee that
// takes ANOTHER object's mutex without naming it. These three tests drive the real functions instead:
//
//   - the SNAPSHOT helper must RELEASE the owner lock before it returns (a leak would make the next write
//     lock unobtainable for ever — a re-entrant acquisition raises no error, so only a watchdog sees it);
//   - the hourly dividend daemon must COMPLETE while the CALLER holds the lobby lock (it used to take it
//     itself, and it also used to read the map before taking ANY lock);
//   - the INSERT must QUEUE for the owner lock — the DISCRIMINATOR, because the body this replaced created
//     a map entry with no router lock at all and therefore completed instantly under a held READ lock.

func newMarketNodeOwnerFixture(t *testing.T) *Lobby {
	t.Helper()
	faucet, admin := uint64(0), uint64(0)
	router := NewTokenSinkRouter(&faucet, &admin)
	router.MarketNodes["0xentity-a"] = &EntityMarketNode{
		EntityID:          "0xentity-a",
		TotalSharesIssued: 1000,
		ReserveBalance:    24_000_000, // exactly 1000 micro-units of hourly yield at 1/24000
	}
	router.MarketNodes["0xentity-frozen"] = &EntityMarketNode{
		EntityID:          "0xentity-frozen",
		TotalSharesIssued: 1000,
		ReserveBalance:    24_000_000,
		IsDividendFrozen:  true,
	}
	l := &Lobby{
		tokenSinkRouter: router,
		broadcast:       make(chan []byte, 8),
		leaderboard:     map[string]PlayerStats{},
	}
	// The alias IS the subject: the lobby field must name the router's map, which is what `server.go` does
	// at boot. Asserted so this fixture cannot quietly drift away from the real shape.
	router.Mu.Lock()
	l.marketNodes = router.MarketNodes
	router.Mu.Unlock()
	if len(l.marketNodes) != 2 {
		t.Fatalf("fixture: the lobby field and the router's map must be ONE map, got %d entries", len(l.marketNodes))
	}
	return l
}

// TestMarketNodeRefsReleasesTheOwnerLock pins the contract that makes the helper safe to use from a caller
// that then takes a NODE lock: the owner lock is taken inside it and RELEASED before it returns.
func TestMarketNodeRefsReleasesTheOwnerLock(t *testing.T) {
	l := newMarketNodeOwnerFixture(t)

	refs := l.tokenSinkRouter.marketNodeRefs()
	if len(refs) != 2 {
		t.Fatalf("snapshot returned %d nodes, want 2", len(refs))
	}
	for _, ref := range refs {
		if ref.Node == nil || ref.EntityID == "" {
			t.Fatalf("snapshot entry is incomplete: %+v", ref)
		}
	}

	// THE DISCRIMINATOR. If `marketNodeRefs` leaked its read lock, this WRITE lock is never granted (a
	// non-re-entrant `sync.RWMutex`, same goroutine) — and a re-entrant acquisition raises no error, so
	// only a watchdog can see it.
	done := make(chan struct{})
	go func() {
		defer close(done)
		l.tokenSinkRouter.Mu.Lock()
		l.tokenSinkRouter.Mu.Unlock()
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("marketNodeRefs leaked the owner lock: the write lock was never granted, so the map's " +
			"owner is held by a function that has already returned")
	}
}

// TestHourlyEntityRevenueDistributionRunsWhileTheLobbyLockIsHeld pins the daemon's lock SCOPE: it takes no
// lobby lock (it used to hold it for its whole body) and it reads the map ONLY through the owner-locked
// snapshot helper. It also pins the INTEGER yield and the frozen-node skip.
func TestHourlyEntityRevenueDistributionRunsWhileTheLobbyLockIsHeld(t *testing.T) {
	l := newMarketNodeOwnerFixture(t)

	watchdog(t, "ProcessHourlyEntityRevenueDistribution with the lobby write lock held", func() {
		l.mutex.Lock()
		defer l.mutex.Unlock()
		l.ProcessHourlyEntityRevenueDistribution()
	})
	assertLobbyLockIsFree(t, l, "ProcessHourlyEntityRevenueDistribution")

	// 0.1% per day across 24 hours is 1/24000 of the reserve, FLOORED — the integer form that replaced a
	// float multiply inside the ledger.
	if got := l.tokenSinkRouter.MarketNodes["0xentity-a"].DividendPoolMicro; got != 1000 {
		t.Errorf("0xentity-a pool = %d, want the exact integer yield 1000 (24,000,000 / 24000)", got)
	}
	if got := l.tokenSinkRouter.MarketNodes["0xentity-frozen"].DividendPoolMicro; got != 0 {
		t.Errorf("a FROZEN entity received %d; the freeze check must skip it", got)
	}
}

// TestTheNodeInsertQueuesForTheOwnerLock is the DISCRIMINATOR for the WRITER. The insert used to create a
// map entry holding `l.mutex` only, so it completed instantly while any router-lock holder was active —
// which is precisely how a walk could die with `fatal error: concurrent map iteration and map write`.
// Holding the owner's READ lock must therefore BLOCK the insert.
func TestTheNodeInsertQueuesForTheOwnerLock(t *testing.T) {
	l := newMarketNodeOwnerFixture(t)
	const fresh = "0xentity-fresh"

	l.tokenSinkRouter.Mu.RLock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		l.mutex.Lock()
		_ = l.getOrCreateMarketNodeLocked(fresh)
		l.mutex.Unlock()
	}()
	select {
	case <-done:
		l.tokenSinkRouter.Mu.RUnlock()
		t.Fatalf("the insert COMPLETED while the owner's write lock was unavailable: it created a map " +
			"entry without the owner lock, which is the defect this pins")
	case <-time.After(300 * time.Millisecond):
		// EXPECTED: it is queued for the owner's write lock.
	}
	l.tokenSinkRouter.Mu.RUnlock()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("the insert never completed after the owner lock was released: it is deadlocked")
	}
	if _, ok := l.tokenSinkRouter.MarketNodes[fresh]; !ok {
		t.Errorf("the insert did not publish the new node to the ROUTER's map (the authority)")
	}
	if _, ok := l.marketNodes[fresh]; !ok {
		t.Errorf("the lobby field no longer names the router's map: the alias has been broken")
	}
}
