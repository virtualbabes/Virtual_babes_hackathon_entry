//go:build !js && !wasm

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// ============================================================================
// THE SELF-LOCK GATE — a career award must never reach for a lock its caller
// already holds.
// ============================================================================
//
// WHY THIS EXISTS. `l.TrackCareerXP` takes `l.mutex.Lock()` ITSELF. A function that already holds
// that lock and then calls it is an UNCONDITIONAL SELF-DEADLOCK on a `sync.RWMutex`, and because
// the write lock is never released it freezes every request and every WebSocket in the process —
// not merely the one request that happened to be running.
//
// SIX instances of this class have been found in this repository, every one of them by READING:
// `handleBailCard`, `HandleRepayLoan`, `HandleDetectCounterfeit`, `use_item` ▸ `legal_pardon`,
// `handleCyberIntercept` and `handleKidnapRequest`. Reading is not a gate, and the last three sat
// behind DEAD arithmetic that hid them — the guarded code simply never ran — so no behaviour test
// could have caught them. This gate measures the shape instead of the behaviour.
//
// WHAT IT MEASURES, in one line: within ONE function body, is a call to a SELF-LOCKING career-award
// helper reachable at a source position where the lobby lock is already held?
//
// HONEST LIMITS (stated, not implied):
//   - SOURCE ORDER, not control flow. A lock taken in an `if` branch and an award in the `else` is
//     reported; a lock released on one path but not another is not modelled. The gate errs toward
//     REPORTING, and every real instance of this class here has been a straight line lock → award.
//   - A DEFERRED unlock releases at FUNCTION EXIT, so it does not clear the held state at its
//     source position. That is deliberate: it is what makes `Lock(); defer Unlock(); …award` — the
//     dominant spelling of this defect — visible.
//   - A FUNCTION LITERAL (`func(){…}()` or `go func(){…}()`) is its own scope and does not inherit
//     the enclosing lock state: a goroutine's lock discipline cannot be decided statically.
//   - `_test.go` files are NOT scanned — a test that holds the lock on purpose is TESTING the lock,
//     so scanning them would report the very file that proves the gate works.
//   - Only a PLAIN IDENTIFIER receiver is the Lobby helper (`l.TrackCareerXP`). The CareerXP method
//     `stats.CareerXP.TrackCareerXP(…)` takes NO lock and must not be reported; that discrimination
//     is pinned by the negative control, because a gate that cries wolf gets deleted.
// ============================================================================

// selfLockingCareerAwardFuncs names the career-award helpers that take the lobby lock THEMSELVES.
// A caller that already holds that lock must use the `…Locked` form beside it instead.
//
// The value is the REASON it is dangerous, so a failure explains itself without anyone having to go
// and look the helper up. Add a name here the moment a new self-locking award helper appears.
var selfLockingCareerAwardFuncs = map[string]string{
	"TrackCareerXP": "it takes l.mutex.Lock() itself (lobby_manager.go) — a caller holding the lock must call trackCareerXPLocked",
}

// careerLockCall is ONE lock acquisition, lock release, or self-locking award, in source order.
type careerLockCall struct {
	pos    token.Pos
	holds  bool   // true = ACQUIRES the lobby lock; false = releases it, or is an award
	award  string // non-empty = a self-locking award helper, carrying the reason
	detail string // how it is spelled, for the failure message
}

// classifyLobbyLockCall decides whether ONE call expression touches the lobby lock or reaches a
// self-locking award helper.
//
// `holds` is true for BOTH `Lock` and `RLock`: a goroutine holding the read lock and then asking
// for the write lock deadlocks exactly as hard, so both are "lock held" state.
func classifyLobbyLockCall(call *ast.CallExpr) (holds bool, award, detail string, ok bool) {
	sel, isSelector := call.Fun.(*ast.SelectorExpr)
	if !isSelector {
		return false, "", "", false
	}
	switch sel.Sel.Name {
	case "Lock", "RLock", "Unlock", "RUnlock":
		// The receiver must BE the lobby mutex field, so an unrelated `mu.Lock()` or
		// `wg.Lock()` is never mistaken for it.
		inner, isMutexField := sel.X.(*ast.SelectorExpr)
		if !isMutexField || inner.Sel.Name != "mutex" {
			return false, "", "", false
		}
		acquires := sel.Sel.Name == "Lock" || sel.Sel.Name == "RLock"
		return acquires, "", sel.Sel.Name + "() on the lobby mutex", true
	}

	reason, isAward := selfLockingCareerAwardFuncs[sel.Sel.Name]
	if !isAward {
		return false, "", "", false
	}
	// ONLY a plain identifier receiver is the Lobby helper. `stats.CareerXP.TrackCareerXP(…)` is
	// the CareerXP METHOD, which takes no lock at all — reporting it would be a false positive on
	// correct code, and a gate with false positives is a gate that gets switched off.
	if _, isPlainReceiver := sel.X.(*ast.Ident); !isPlainReceiver {
		return false, "", "", false
	}
	return false, reason, sel.Sel.Name + "() on a Lobby receiver", true
}

// careerLockEvents collects, IN SOURCE ORDER, every lock acquisition/release and every self-locking
// award reachable inside ONE function body.
//
// TWO decisions carry the weight:
//
//   - A DEFERRED release is SKIPPED, because `defer l.mutex.Unlock()` releases at FUNCTION EXIT and
//     not at its source position. Treating it as a release where it is written would hide the
//     dominant spelling of this defect — `Lock(); defer Unlock(); … l.TrackCareerXP(…)` — which is
//     precisely the shape that froze the process.
//   - A FUNCTION LITERAL is its own scope and is not descended into (see the limits at the head).
func careerLockEvents(body *ast.BlockStmt) []careerLockCall {
	var events []careerLockCall
	if body == nil {
		return events
	}
	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncLit:
			return false // a closure's lock discipline cannot be decided statically
		case *ast.DeferStmt:
			if holds, award, _, ok := classifyLobbyLockCall(node.Call); ok && !holds && award == "" {
				return false // a deferred RELEASE happens at return, not here
			}
			return true
		case *ast.CallExpr:
			if holds, award, detail, ok := classifyLobbyLockCall(node); ok {
				events = append(events, careerLockCall{pos: node.Pos(), holds: holds, award: award, detail: detail})
			}
		}
		return true
	})
	sort.Slice(events, func(i, j int) bool { return events[i].pos < events[j].pos })
	return events
}

// careerAwardLockViolations reports every function in ONE parsed source file that reaches a
// self-locking career-award helper while the lobby lock is held.
//
// It takes SOURCE BYTES, not a filename, so this gate's own detector can be driven with a synthetic
// file — the negative control below. Without that the gate would be an untested assertion about
// itself: a detector that silently matched nothing would report a clean repository for ever.
func careerAwardLockViolations(filename string, src []byte) ([]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, decl := range file.Decls {
		fn, isFunc := decl.(*ast.FuncDecl)
		if !isFunc || fn.Body == nil {
			continue
		}
		heldDetail, heldLine := "", 0
		for _, ev := range careerLockEvents(fn.Body) {
			if ev.award != "" {
				if heldDetail != "" {
					out = append(out, fmt.Sprintf(
						"%s: %s() reaches %s at line %d while the lobby lock is held (%s at line %d) — %s",
						filename, fn.Name.Name, ev.detail, fset.Position(ev.pos).Line,
						heldDetail, heldLine, ev.award))
				}
				continue
			}
			if ev.holds {
				heldDetail, heldLine = ev.detail, fset.Position(ev.pos).Line
			} else {
				heldDetail, heldLine = "", 0
			}
		}
	}
	return out, nil
}

// TestNoSelfLockingCareerAward is the gate itself: NO production Go file in this package may reach
// a self-locking career-award helper while it holds the lobby lock.
//
// It scans the PACKAGE ROOT, because that is where every one of the six instances of this class was
// found (`handlers_criminality.go`, `courthouse_service.go`, `counterfeit_service.go`,
// `loan_service.go`), and because a source gate built on a hand-written file list goes stale the
// first time a file is added.
func TestNoSelfLockingCareerAward(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob *.go: %v", err)
	}
	if len(files) == 0 {
		// A gate that looked at nothing reports nothing. Never let that read as a pass.
		t.Fatal("no Go sources found in the working directory — this gate must never pass vacuously")
	}

	scanned := 0
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue // a test that holds the lock on purpose is TESTING the lock
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		scanned++
		violations, err := careerAwardLockViolations(name, src)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, v := range violations {
			t.Errorf("SELF-LOCKING CAREER AWARD: %s", v)
		}
	}

	// The tree must be the tree this gate claims to cover: the package root holds far more than
	// fifty non-test Go files, and a scan of a couple of them would be a clean report about nothing.
	if scanned < 50 {
		t.Errorf("scanned only %d non-test Go files; the package root holds far more, so this gate is not looking at the tree it claims to", scanned)
	}
	t.Logf("self-lock gate: %d non-test Go files scanned, 0 self-locking career awards", scanned)
}

// TestCareerAwardLockGateDetectsAViolation is the NEGATIVE CONTROL, and it is the part that makes
// the gate above mean anything: a detector whose matcher silently failed would report a clean
// repository for ever, and nobody would know.
//
// The synthetic file below contains FIVE shapes. Only the first TWO may be reported.
func TestCareerAwardLockGateDetectsAViolation(t *testing.T) {
	const synthetic = `package main

func (l *Lobby) badDeferredUnlock() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.TrackCareerXP("0xbad", "Tax Auditor", 15)
}

func (l *Lobby) badReadLockHeld() {
	l.mutex.RLock()
	_ = l.leaderboard
	l.TrackCareerXP("0xbad", "Fence", 5)
	l.mutex.RUnlock()
}

func (l *Lobby) fineLockedForm() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.trackCareerXPLocked("0xok", "Fence", 5)
}

func (l *Lobby) fineCareerXPMethodUnderLock() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	stats := l.leaderboard["0xok"]
	stats.CareerXP.TrackCareerXP("Fence", 5)
}

func (l *Lobby) fineReleaseThenAward() {
	l.mutex.Lock()
	l.mutex.Unlock()
	l.TrackCareerXP("0xok", "Fence", 5)
}
`

	violations, err := careerAwardLockViolations("synthetic.go", []byte(synthetic))
	if err != nil {
		t.Fatalf("the detector must parse a well-formed file: %v", err)
	}
	report := strings.Join(violations, "\n")

	if len(violations) != 2 {
		t.Fatalf("expected exactly 2 violations (the deferred-unlock and read-lock forms); got %d:\n%s", len(violations), report)
	}
	for _, want := range []string{"badDeferredUnlock", "badReadLockHeld"} {
		if !strings.Contains(report, want) {
			t.Errorf("the detector MISSED %s() — it cannot see the shape it exists for:\n%s", want, report)
		}
	}
	for _, mustNot := range []string{"fineLockedForm", "fineCareerXPMethodUnderLock", "fineReleaseThenAward"} {
		if strings.Contains(report, mustNot) {
			t.Errorf("the detector reported %s(), which is correct code — a gate that cries wolf gets switched off:\n%s", mustNot, report)
		}
	}

	// It must also be able to report NOTHING: a gate that always fires is as useless as one that
	// never does, and the clean case is what the real scan asserts about the whole tree.
	clean, err := careerAwardLockViolations("clean.go", []byte("package main\n\nfunc (l *Lobby) ok() {}\n"))
	if err != nil {
		t.Fatalf("parse clean.go: %v", err)
	}
	if len(clean) != 0 {
		t.Errorf("a clean file reported %d violations: %v", len(clean), clean)
	}
}
