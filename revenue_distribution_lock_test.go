//go:build !js && !wasm

package main

import (
	"math"
	"testing"
	"time"
)

// ============================================================================
// ProcessEntityRevenueDistribution — a DORMANT function pinned for its LOCK ORDER.
// ============================================================================
//
// WHY THIS EXISTS. The function had ZERO callers and carried THREE defects at once: it held `l.mutex` for
// its whole body, it took `tokenSinkRouter.Mu` and then mutated `node.DividendPoolMicro` with NO node lock
// at all, and it divided with a float inside the ledger. Dormant code is where a lock bug hides longest,
// so it is pinned by BEHAVIOUR rather than by its own comment — and a re-entrant lock acquisition raises
// NO error (the goroutine simply never returns), so every lock assertion here runs under the watchdog
// that `entity_investment_lock_test.go` already owns.

// newRevenueDistributionFixture builds the smallest lobby the function can run against: a router with
// three nodes, one of which carries NO reserve (the zero-divisor case the original guarded with a +1
// seed, which the rewrite preserves).
func newRevenueDistributionFixture(t *testing.T) *Lobby {
	t.Helper()
	faucet, admin := uint64(0), uint64(0)
	router := NewTokenSinkRouter(&faucet, &admin)
	for id, reserve := range map[string]uint64{
		"0xentity-a": 3_000_000_000,
		"0xentity-b": 1_000_000_000,
		"0xentity-c": 0,
	} {
		router.MarketNodes[id] = &EntityMarketNode{
			EntityID:          id,
			TotalSharesIssued: 1000,
			ReserveBalance:    reserve,
		}
	}
	return &Lobby{tokenSinkRouter: router}
}

// TestEntityRevenueDistributionRunsUnderAHeldLobbyLock pins the LOBBY half of the contract: the body this
// replaced took `l.mutex` ITSELF, so calling it from a context that already held the lobby write lock
// deadlocked the WHOLE process (every request, every WebSocket) rather than merely this call.
func TestEntityRevenueDistributionRunsUnderAHeldLobbyLock(t *testing.T) {
	l := newRevenueDistributionFixture(t)
	const revenue = uint64(7_000_000_000)

	watchdog(t, "ProcessEntityRevenueDistribution with the lobby write lock held", func() {
		l.mutex.Lock()
		defer l.mutex.Unlock()
		l.ProcessEntityRevenueDistribution(revenue, "TEST")
	})
	// It takes no lobby lock of its own, so it must leave the caller's lock exactly as it found it.
	assertLobbyLockIsFree(t, l, "ProcessEntityRevenueDistribution")

	total := uint64(1) // the denominator's seed, computed the same way the function computes it
	for _, node := range l.tokenSinkRouter.MarketNodes {
		total += node.ReserveBalance
	}
	var distributed uint64
	for id, node := range l.tokenSinkRouter.MarketNodes {
		want := proportionalMicro(revenue, node.ReserveBalance, total)
		if node.DividendPoolMicro != want {
			t.Errorf("entity %s pool = %d, want the exact integer split %d", id, node.DividendPoolMicro, want)
		}
		distributed += node.DividendPoolMicro
	}
	if distributed == 0 {
		t.Fatalf("the split distributed nothing at all — the arithmetic is not reaching the pools")
	}
	if distributed > revenue {
		t.Errorf("distributed %d exceeds the %d routed in: a proportional share of the whole must never "+
			"mint beyond it", distributed, revenue)
	}
	if got := l.tokenSinkRouter.MarketNodes["0xentity-c"].DividendPoolMicro; got != 0 {
		t.Errorf("a node with no reserve received %d; it must receive 0 rather than a division by zero", got)
	}
}

// TestEntityRevenueDistributionHonoursTheNodeWriteLock is the DISCRIMINATOR. It holds one node's READ lock
// and asserts the function canNOT finish, because the function must queue for that node's WRITE lock to
// apply the share. The body this replaced wrote `DividendPoolMicro` with no lock at all, so it completed
// happily under a read lock — i.e. this test fails against the code it was written for.
func TestEntityRevenueDistributionHonoursTheNodeWriteLock(t *testing.T) {
	l := newRevenueDistributionFixture(t)
	node := l.tokenSinkRouter.MarketNodes["0xentity-a"]

	node.Mu.RLock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		l.ProcessEntityRevenueDistribution(2_000_000_000, "TEST")
	}()
	select {
	case <-done:
		node.Mu.RUnlock()
		t.Fatalf("the function COMPLETED while another goroutine held entity-a's read lock: it applied " +
			"its write without the node's write lock, which is the defect this pins")
	case <-time.After(300 * time.Millisecond):
		// It is queued for the write lock, which is exactly what it must do.
	}
	node.Mu.RUnlock()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("the function never completed after the node lock was released")
	}
	if got := node.DividendPoolMicro; got == 0 {
		t.Errorf("entity-a's pool is still 0 after the distribution: the write never happened")
	}
}

// TestProportionalMicroIsExactAndCannotOverflow pins the arithmetic owner: integer only, FLOORED, and
// correct at the values where the float expression it replaced cannot even represent the result.
func TestProportionalMicroIsExactAndCannotOverflow(t *testing.T) {
	cases := []struct {
		name                      string
		amount, part, whole, want uint64
	}{
		{"an exact quarter", 1000, 1, 4, 250},
		{"floors rather than rounds", 1000, 1, 3, 333},
		{"three quarters", 1000, 3, 4, 750},
		{"a zero amount", 0, 5, 10, 0},
		{"a zero share", 1000, 0, 10, 0},
		{"a zero whole must not divide", 1000, 5, 0, 0},
		{"the whole is the whole", 1000, 10, 10, 1000},
		{"more than the whole is CLAMPED, never wrapped", 1000, 11, 10, 1000},
		{"the float expression cannot represent this one",
			math.MaxUint64, math.MaxUint64 - 1, math.MaxUint64, math.MaxUint64 - 1},
	}
	for _, c := range cases {
		if got := proportionalMicro(c.amount, c.part, c.whole); got != c.want {
			t.Errorf("proportionalMicro(%d, %d, %d) = %d, want %d (%s)",
				c.amount, c.part, c.whole, got, c.want, c.name)
		}
	}
}
