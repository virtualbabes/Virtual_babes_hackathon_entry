//go:build !js && !wasm

package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"
)

// ============================================================================
// THE DIRECT-INVEST DOORS — pinned BEHAVIOURALLY, because a deadlock is silent.
// ============================================================================
//
// WHY A WATCHDOG IS THE ONLY HONEST PIN. A re-entrant `sync.RWMutex` acquisition raises NO error: the
// goroutine simply never returns. A test that asserted on a returned error would have passed against
// the broken code for ever, so these tests fail by TIMEOUT and dump every goroutine's stack, which is
// where the mechanism becomes legible (`sync.RWMutex.RLock` waiting on the write lock the SAME
// goroutine is holding).
//
// THE PRECONDITION IS THE POINT. Both doors priced the portfolio while holding the target node's WRITE
// lock, and the portfolio walk takes each node's READ lock — so the seed below gives the investor
// shares in the SAME entity they then invest in. That is what turned a conditional hazard into an
// unconditional one on the success path, and the seed asserts it, so the test cannot quietly stop
// testing the shape it was written for.

const (
	relockTestEntity = "0xentity-relock"
	relockTestWallet = "0xinvestor-relock"
)

func newDirectInvestFixture(t *testing.T) *Lobby {
	t.Helper()

	faucet, admin := uint64(0), uint64(0)
	router := NewTokenSinkRouter(&faucet, &admin)
	router.MarketNodes[relockTestEntity] = &EntityMarketNode{
		EntityID:                relockTestEntity,
		TotalSharesIssued:       1000,
		ReserveBalance:          1_000_000_000,
		DividendPoolMicro:       500_000_000,
		CumulativeYieldPerShare: 1234,
	}

	l := &Lobby{
		clients:         map[string]*Client{},
		broadcast:       make(chan []byte, 8),
		tokenSinkRouter: router,
		leaderboard: map[string]PlayerStats{
			relockTestWallet: {
				// THE PRECONDITION: shares in the entity the door is about to lock.
				Portfolio: map[string]uint64{relockTestEntity: 500},
			},
		},
		playerBalances:          map[string]uint64{relockTestWallet: 100_000_000_000},
		playerDirectInvestments: map[string]map[string]uint64{},
		playerInvestmentRecords: map[string][]*EntityInvestmentRecord{},
	}
	// Both doors broadcast AFTER they commit. A nil channel would leak the sender goroutine for ever;
	// the test drains it instead, so the door's own tail cannot be mistaken for a hang.
	go func() {
		for range l.broadcast {
		}
	}()
	return l
}

// watchdog runs fn and fails with every goroutine's stack if it has not returned in time.
func watchdog(t *testing.T, what string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		buf := make([]byte, 1<<20)
		n := runtime.Stack(buf, true)
		t.Fatalf("%s did not return within 5s. A re-entrant lock acquisition raises NO error — it simply "+
			"never returns — so a watchdog is the only pin that can see it. Goroutines:\n%s", what, buf[:n])
	}
}

// TestTheWsDirectInvestDoorCompletesWithSharesInTheTargetEntity pins the WebSocket door.
func TestTheWsDirectInvestDoorCompletesWithSharesInTheTargetEntity(t *testing.T) {
	l := newDirectInvestFixture(t)
	if got := l.leaderboard[relockTestWallet].Portfolio[relockTestEntity]; got <= 0 {
		t.Fatalf("the fixture must give the investor shares in the target entity (got %v) — otherwise this "+
			"test no longer exercises the re-entrant read lock it exists for", got)
	}

	watchdog(t, "the WS direct-invest door", func() {
		// The WS door reads the WALLET from `env.FromID` ("Use wallet address as identifier"), so the
		// fixture passes the wallet — and `sendToClientLocked` no-ops for a client id it does not know,
		// which is exactly the delivery path a disconnected investor gets.
		l.handleInvestEntity(Envelope{FromID: relockTestWallet}, InvestmentData{
			EntityID:    relockTestEntity,
			AmountMicro: 2_000_000_000,
		})
	})

	// It must have reached the SUCCESS path, which is where the deadlock lived: the balance moved and
	// the entity's pool grew by the same integer amount.
	if got := l.playerBalances[relockTestWallet]; got != 98_000_000_000 {
		t.Errorf("balance after investing 2,000 $VBV = %d, want %d", got, 98_000_000_000)
	}
	if got := l.tokenSinkRouter.MarketNodes[relockTestEntity].DividendPoolMicro; got != 2_500_000_000 {
		t.Errorf("dividend pool after the investment = %d, want %d", got, 2_500_000_000)
	}
}

// TestTheHttpDirectInvestDoorCompletesWithSharesInTheTargetEntity pins the HTTP door, which is the one a
// browser actually reaches (`POST /api/invest/entity`).
func TestTheHttpDirectInvestDoorCompletesWithSharesInTheTargetEntity(t *testing.T) {
	l := newDirectInvestFixture(t)
	if got := l.leaderboard[relockTestWallet].Portfolio[relockTestEntity]; got <= 0 {
		t.Fatalf("the fixture must give the investor shares in the target entity (got %v)", got)
	}

	body, err := json.Marshal(map[string]any{
		"entity_id":    relockTestEntity,
		"amount_micro": uint64(2_000_000_000),
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest("POST", "/api/invest/entity?wallet="+relockTestWallet, bytes.NewReader(body))
	rec := httptest.NewRecorder()

	svc := NewEntityInvestmentService()
	watchdog(t, "the HTTP direct-invest door", func() {
		svc.HandleDirectInvest(l, rec, req)
	})

	if rec.Code != 200 {
		t.Fatalf("POST /api/invest/entity answered %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("the door must answer JSON: %v (%s)", err, rec.Body.String())
	}
	if payload["status"] != "invested" {
		t.Errorf("status = %v, want \"invested\" (body: %s)", payload["status"], rec.Body.String())
	}
	if got := l.playerBalances[relockTestWallet]; got != 98_000_000_000 {
		t.Errorf("balance after investing 2,000 $VBV = %d, want %d", got, 98_000_000_000)
	}
	if got := l.tokenSinkRouter.MarketNodes[relockTestEntity].DividendPoolMicro; got != 2_500_000_000 {
		t.Errorf("dividend pool after the investment = %d, want %d", got, 2_500_000_000)
	}
}

// TestPortfolioPricingLockPairHonoursItsContract pins the CONTRACT of the new pair, which is what stops
// a future caller re-creating the deadlock: the lock-held form must not touch the lobby lock itself (so
// it is safe to call while holding it), while the DOOR must take and release it.
func TestPortfolioPricingLockPairHonoursItsContract(t *testing.T) {
	l := newDirectInvestFixture(t)

	// The door takes the read lock and releases it: the lock must be free afterwards.
	_ = l.CalculateTotalPortfolioValue(relockTestWallet)
	assertLobbyLockIsFree(t, l, "CalculateTotalPortfolioValue")

	// The lock-held form runs while the WRITE lock is already held. A re-entrant acquisition here would
	// hang this test rather than fail it, so it runs under the watchdog.
	watchdog(t, "CalculateTotalPortfolioValueLocked under the held write lock", func() {
		l.mutex.Lock()
		defer l.mutex.Unlock()
		if got := l.CalculateTotalPortfolioValueLocked(relockTestWallet); got == 0 {
			t.Errorf("the portfolio value of a funded investor must not be 0")
		}
	})
}
