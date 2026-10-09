//go:build !js && !wasm

package main

import (
	"reflect"
	"sort"
	"testing"
	"time"
)

// ════════════════════════════════════════════════════════════════════════════
// COURTHOUSE RIVAL PAIRING — the lock contract (Problems.md §15)
//
// Two defects lived in this pairing logic, one in each direction:
//
//  1. The RESET handler (HandleCourthouseReset) ranged over l.leaderboard to
//     find the pairing with NO lock held while other goroutines wrote it — a
//     `fatal error: concurrent map iteration and map write`, which is not
//     recoverable and takes the process down.
//
//  2. The PARDON path runs with the lobby WRITE lock already held (`use_item`
//     holds it across applyItemEffect) and then awarded XP through the
//     SELF-LOCKING TrackCareerXP — an unconditional self-deadlock on a
//     sync.RWMutex that freezes every request and every WebSocket in the
//     process. The same defect class as the three money doors.
//
// The fix is ONE owner with TWO forms: courthouseRivalAwardsLocked resolves the pairings from a
// lock-held snapshot (caller holds the lock), courthouseRivalAwards takes the read lock itself
// (caller holds nothing), and every award is applied through the LOCK-HELD XP helper. Both forms
// resolve through ResolveRivalXPAward, so the CAREER matrix and the PATH matrix can never drift
// apart — the path matrix's direct justice↔criminal bonus had been declared, served and pinned by
// a test while being awarded in ZERO places.
// ════════════════════════════════════════════════════════════════════════════

const (
	rivalJudgeWallet  = "0xjudge"
	rivalTargetWallet = "0xtarget"
	rivalWalletA      = "0xrival-a"
	rivalWalletB      = "0xrival-b"
)

// rivalScanLobby builds the minimum a courthouse resolution point needs. The
// audit log is redirected into a temp dir so a test never writes into the repo.
//
// The judge is given the Tax Auditor job so the pairing scan has something to
// find when the pardon path runs (the "restricted to Judges" gate lives in
// applyItemEffect, not here).
func rivalScanLobby(t *testing.T) *Lobby {
	t.Helper()
	return &Lobby{
		DataDir: t.TempDir(),
		leaderboard: map[string]PlayerStats{
			rivalJudgeWallet:  {Wallet: rivalJudgeWallet, JobRole: "Tax Auditor", CareerXP: &CareerXP{}},
			rivalTargetWallet: {Wallet: rivalTargetWallet, JobRole: "JusticeCommissioner", WantedLevel: 5, CareerXP: &CareerXP{}},
			rivalWalletA:      {Wallet: rivalWalletA, JobRole: "JusticeCommissioner", CareerXP: &CareerXP{}},
			rivalWalletB:      {Wallet: rivalWalletB, JobRole: "JusticeCommissioner", CareerXP: &CareerXP{}},
		},
		clubs: map[string]*Club{},
	}
}

func TestRivalScanFindsTheTaxAuditorPairings(t *testing.T) {
	l := rivalScanLobby(t)

	// A wallet that does NOT hold the actor's role resolves nothing: the pairing is
	// TAX-AUDITOR-side, and the role is supplied by the resolution point rather than read off the
	// wallet being fined.
	l.mutex.RLock()
	others := l.courthouseRivalAwardsLocked(rivalTargetWallet, "Tax Auditor", CourthouseFineCareerXP)
	l.mutex.RUnlock()
	if len(others) != 0 {
		t.Errorf("a wallet that is not a Tax Auditor yielded %d awards; want 0", len(others))
	}

	l.mutex.RLock()
	awards := l.courthouseRivalAwardsLocked(rivalJudgeWallet, "Tax Auditor", CourthouseFineCareerXP)
	l.mutex.RUnlock()

	if len(awards) != 3 {
		t.Fatalf("awards = %d; want 3 (the target plus two JusticeCommissioners)", len(awards))
	}
	wallets := make([]string, 0, len(awards))
	for _, a := range awards {
		if a.Role != "JusticeCommissioner" {
			t.Errorf("award for %s has peer role %q; want JusticeCommissioner", a.Wallet, a.Role)
		}
		// A zero-bonus award is filtered out by the owner, so every row here PAYS — and it must
		// beat the base it was resolved from, or the pairing is a decoration.
		if a.Award.Bonus == 0 || a.Award.AttackerXP <= CourthouseFineCareerXP {
			t.Errorf("award for %s does not beat the base: %+v", a.Wallet, a.Award)
		}
		if !a.Award.IsRival {
			t.Errorf("award for %s lost its career-matrix pair: %+v", a.Wallet, a.Award)
		}
		wallets = append(wallets, a.Wallet)
	}
	sort.Strings(wallets)
	want := []string{rivalTargetWallet, rivalWalletA, rivalWalletB}
	sort.Strings(want)
	for i := range want {
		if wallets[i] != want[i] {
			t.Errorf("award wallets = %v; want %v", wallets, want)
			break
		}
	}
}

// TestRivalScanReturnsCopiesNotLiveMapValues is what makes it safe to APPLY the award after
// releasing the lock: nothing the resolution returns may alias the map.
//
// It pins BOTH halves of that contract — an award already resolved is a value copy that a later
// write cannot change, AND the resolution itself reads the LIVE map, which is precisely why it can
// only ever run while the lock is held.
func TestRivalScanReturnsCopiesNotLiveMapValues(t *testing.T) {
	l := rivalScanLobby(t)

	l.mutex.RLock()
	taken := l.courthouseRivalAwardsLocked(rivalJudgeWallet, "Tax Auditor", CourthouseFineCareerXP)
	l.mutex.RUnlock()
	if len(taken) == 0 {
		t.Fatal("the scan found no pairings to resolve")
	}
	before := make([]courthouseRivalAward, len(taken))
	copy(before, taken)

	// A writer now changes one rival's stored stats, exactly as another request would while an
	// already-resolved award is being applied.
	l.mutex.Lock()
	mutated := l.leaderboard[rivalWalletA]
	mutated.Wins = 999
	mutated.JobRole = "Freelancer" // declares no pair with a Tax Auditor
	l.leaderboard[rivalWalletA] = mutated
	l.mutex.Unlock()

	if !reflect.DeepEqual(taken, before) {
		t.Error("a write to the leaderboard changed an award that was already resolved — the resolution aliased live map state")
	}

	// The same call now finds one FEWER pairing: the resolution reads the map it is holding the
	// lock for, so a scan run outside the lock is a scan over a map another goroutine is writing.
	l.mutex.RLock()
	after := l.courthouseRivalAwardsLocked(rivalJudgeWallet, "Tax Auditor", CourthouseFineCareerXP)
	l.mutex.RUnlock()
	if len(after) != len(before)-1 {
		t.Errorf("awards after the peer's role changed = %d; want %d — the resolution must read the live map, not a cache", len(after), len(before)-1)
	}
}

// TestRivalBonusScanLeavesTheLobbyLockFree pins the unlocked-facing form used by
// the request handler: it must take and release the read lock itself.
func TestRivalBonusScanLeavesTheLobbyLockFree(t *testing.T) {
	l := rivalScanLobby(t)
	_ = l.courthouseRivalAwards(rivalJudgeWallet, "Tax Auditor", CourthouseFineCareerXP)
	assertLobbyLockIsFree(t, l, "courthouseRivalAwards")
}

// TestLegalPardonCompletesWhileTheItemUseLockIsHeld is the deadlock regression.
//
// `use_item` (lobby_manager.go) holds the write lock across applyItemEffect,
// which routes legal_pardon into ApplyLegalPardonLocked. That function awarded
// XP through the self-locking TrackCareerXP, so it blocked on a lock it already
// held — and because the write lock is never released, the whole process froze.
//
// This test fails by TIMEOUT rather than by assertion, because a deadlock raises
// no error at all: it just never returns.
func TestLegalPardonCompletesWhileTheItemUseLockIsHeld(t *testing.T) {
	l := rivalScanLobby(t)
	cs := &CourthouseService{}
	item := ShopItem{ID: "legal_pardon", Name: "Legal Pardon"}

	done := make(chan error, 1)
	go func() {
		l.mutex.Lock()
		defer l.mutex.Unlock()
		done <- cs.ApplyLegalPardonLocked(l, rivalJudgeWallet, rivalTargetWallet, item)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("the pardon was refused: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ApplyLegalPardonLocked did not return within 5s while the lobby write lock was held: " +
			"it re-acquired the lock (self-deadlock), which freezes every request and every WebSocket in the process")
	}

	assertLobbyLockIsFree(t, l, "ApplyLegalPardonLocked")

	l.mutex.RLock()
	target := l.leaderboard[rivalTargetWallet]
	judge := l.leaderboard[rivalJudgeWallet]
	l.mutex.RUnlock()

	if target.WantedLevel != 3 {
		t.Errorf("target WantedLevel = %d; want 3 (5 minus the 2-point half-reduction)", target.WantedLevel)
	}
	if judge.CareerXP == nil || judge.CareerXP.RoleXP["Lawyer-Commissioner"] != 30 {
		t.Errorf("the judge's Lawyer-Commissioner XP was not awarded while the lock was held: %+v", judge.CareerXP)
	}
	// AND THE RIVAL AWARD IS PAID ON THE LOCKED PATH TOO. The pardon is resolution point #2, and it
	// resolves BOTH matrices (the career pair and the direct justice↔criminal path bonus) through
	// the SAME owner as the fine reset. A zero here means the award is declared, served and pinned
	// by a test — and paid nowhere, which is exactly the defect this path used to have.
	if judge.CareerXP == nil || judge.CareerXP.RoleXP["Tax Auditor"] == 0 {
		t.Errorf("the courthouse rival award was not paid while the lock was held: %+v", judge.CareerXP)
	}
}
