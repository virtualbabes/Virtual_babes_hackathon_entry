//go:build !js && !wasm

package main

import (
	"sync"
	"testing"
	"time"
)

// ============================================================================
// THE BEHAVIOURAL PIN — a static census must never be the only evidence.
// ============================================================================
//
// `lock_order_gate_test.go` reads the source. This DRIVES the shape, because a deadlock raises no
// error at all: it simply never returns, so only a watchdog can see it.
//
// THE ARRANGEMENT (three goroutines, and every one of them is a shape that exists in production):
//   - a LOBBY-LOCK READER that reaches into the engine — `HandleGetGarden` and
//     `runAutonomousTournament` both do exactly this;
//   - a QUEUED LOBBY WRITER. It does nothing but queue, and it is the *reason* the pair is fatal: on
//     an `RWMutex` a waiting writer blocks every later reader, so the tick's missing read lock could
//     never be granted. Request paths in this tree queue this writer constantly;
//   - the TICK, which held `ace.mu` for its whole body.
//
// Before the fix the tick asked for the lobby lock while holding `ace.mu`, the reader held the lobby
// lock while asking for `ace.mu`, and neither could ever return.
func TestTheBehavioralTickCompletesWhileTheLobbyLockIsContested(t *testing.T) {
	l := &Lobby{clubs: map[string]*Club{}}
	ace := NewAICitizenEngine()
	ace.lobby = l
	l.aiEngine = ace
	ace.mu.Lock()
	// Tier 0 and no dogma: the property under test is the LOCK ORDER of the whole tick, not the ritual
	// branch, so the fixture keeps every random branch cheap. Status EMPLOYED is deliberate — it is the
	// branch that used to take the lobby lock from inside the engine lock.
	ace.citizens["AI-LO-1"] = &AICitizen{Wallet: "AI-LO-1", OriginWallet: "OWNER", Career: "Gossip", Tier: 0, Status: "EMPLOYED"}
	ace.mu.Unlock()

	readerHolding := make(chan struct{})
	askEngine := make(chan struct{})
	tickDone := make(chan struct{})

	var wg sync.WaitGroup

	wg.Add(1)
	go func() { // the lobby-lock reader that reaches INTO the engine
		defer wg.Done()
		l.mutex.RLock()
		close(readerHolding)
		<-askEngine
		_ = ace.GetAllCitizens()
		l.mutex.RUnlock()
	}()
	<-readerHolding

	wg.Add(1)
	go func() { // the QUEUED WRITER: from here a new l.mutex.RLock cannot be granted
		defer wg.Done()
		l.mutex.Lock()
		l.mutex.Unlock()
	}()
	// Head start, so the writer is queued before anything else asks for the lobby lock. A slow
	// scheduler only makes this test more LENIENT, never wrong.
	time.Sleep(150 * time.Millisecond)

	go func() {
		ace.BehavioralTick()
		close(tickDone)
	}()
	// Head start, so the tick is inside its engine-locked window when the reader asks for `ace.mu`.
	time.Sleep(150 * time.Millisecond)
	close(askEngine)

	select {
	case <-tickDone:
	case <-time.After(5 * time.Second):
		t.Fatal("DEADLOCK: a tick did not complete while a lobby-lock reader reached into the engine and a " +
			"lobby writer was queued. NOTHING reached from the tick's engine-locked window may acquire " +
			"`lobby.mutex` — copy what it needs BEFORE taking `ace.mu` and apply it after releasing " +
			"(see BehavioralTick / lobbySnapshotForTick).")
	}
	wg.Wait()
}

// TestApplyFaithRitualsTakesTheWRITELock pins the other half of the fix: the club update the tick
// collects must be applied under the WRITE lock, because the block it replaced incremented
// `club.RitualsDone` under a READ lock — a write under a read lock (Problems §32's class), which
// quietly loses an increment whenever two goroutines do it at once.
func TestApplyFaithRitualsTakesTheWRITELock(t *testing.T) {
	l := &Lobby{clubs: map[string]*Club{}}
	l.clubs["CLUB-FAITH"] = &Club{ID: "CLUB-FAITH", Type: "Faith", OwnerWallet: "OWNER", RitualsDone: 1}
	l.clubs["CLUB-OTHER"] = &Club{ID: "CLUB-OTHER", Type: "Commerce", OwnerWallet: "OWNER", RitualsDone: 0}
	ace := NewAICitizenEngine()
	ace.lobby = l

	// A reader holds the lobby lock; the apply must NOT get through.
	hold := make(chan struct{})
	release := make(chan struct{})
	go func() {
		l.mutex.RLock()
		close(hold)
		<-release
		l.mutex.RUnlock()
	}()
	<-hold

	applied := make(chan struct{})
	go func() {
		ace.applyFaithRituals(map[string]int{"OWNER": 2})
		close(applied)
	}()
	select {
	case <-applied:
		t.Error("applyFaithRituals completed while a READER held the lobby lock — it must take the WRITE " +
			"lock, because it MUTATES clubs")
	case <-time.After(250 * time.Millisecond):
		// Correct: a writer waits for the reader to finish.
	}
	close(release)
	select {
	case <-applied:
	case <-time.After(5 * time.Second):
		t.Fatal("applyFaithRituals never completed after the reader released the lobby lock")
	}

	// The counts are APPLIED, and only to the matching Faith club owned by that wallet.
	if got := l.clubs["CLUB-FAITH"].RitualsDone; got != 3 {
		t.Errorf("faith club RitualsDone: want 1+2=3, got %d", got)
	}
	if got := l.clubs["CLUB-OTHER"].RitualsDone; got != 0 {
		t.Errorf("a non-Faith club must not receive rituals, got %d", got)
	}
	// A no-op when nothing was collected: the tick must not need the lobby lock at all in that case.
	ace.applyFaithRituals(nil)
	if got := l.clubs["CLUB-FAITH"].RitualsDone; got != 3 {
		t.Errorf("applyFaithRituals(nil) changed state: got %d, want 3", got)
	}
}
