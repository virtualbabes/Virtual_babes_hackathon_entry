//go:build !js && !wasm

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
	"testing"
)

// ============================================================================
// THE ALIASED-MAP GATE — one map, ONE owner mutex.
// ============================================================================
//
// WHY THIS EXISTS. Three lock gates already stand in this tree and NONE of them can see this class:
// `selflock_gate_test.go` and `mutex_relock_gate_test.go` model ONE mutex, and
// `lock_order_gate_test.go` models the ORDER in which two mutexes are acquired. ALIASING is a third
// shape — a field that is a second NAME for another field's map. `Lobby.marketNodes` and
// `TokenSinkRouter.MarketNodes` are THE SAME MAP (assigned to one another in `server.go`, in
// `Lobby.getOrCreateMarketNodeLocked`, and on restore in `lobby_manager.go`), so a spelling is not a
// lock — and neither is "I took the lock I always take".
//
// THE FAILURE IS WORSE THAN A DEADLOCK. A self-lock or an ABBA inversion leaves the process alive and
// diagnosable; a concurrent map access is `fatal error: concurrent map read and map write` /
// `concurrent map iteration and map write` — not an error value, not catchable with `recover`, and it
// takes every connection down with it.
//
// MEASURED WHEN THIS GATE WAS WRITTEN — every finding was live:
//   - the only runtime WRITER (the insert in `getOrCreateMarketNodeLocked`) held `{Lobby.mutex}` alone;
//   - the 15-minute persistence worker, the dividends-history route and the revenue-distribution walk
//     each read or iterated the SAME map holding `{TokenSinkRouter.Mu}` and NO lobby lock;
//   - the market-vitality range (`theme_engine.go`) and the hourly dividend daemon's `len(...)` held NO
//     map lock at all.
// Disjoint lock sets over ONE map: the writer could land inside every one of those walks. They now all
// hold the owner mutex, and `marketNodesOwnerMutex` is the ONE lock the subject is measured against.
//
// THE SUBJECT IS DERIVED, NOT CURATED. An alias pair comes from an ASSIGNMENT between two field
// selectors of the SAME declared map type, and mutex identities are canonicalised through the struct
// declarations — the sibling gate's machinery REUSED rather than copied (`loDeriveFields`, `loVarsFor`,
// `loCanonFieldOf`, the MUST-held walker and its `...Locked` depth-1 seeding). A field that is a map of
// the same type but is never assigned from another field (`mnaLobby.solo` in the control;
// `RouterSnapshot.MarketNodes`, a VALUE map, in the real tree) is outside the subject by construction.
//
// LIMITS, stated rather than implied:
//   - MUST-HELD, with the sibling gate's semantics (branches joined, a `defer`red release not clearing at
//     its source line, a `go` literal starting from an empty set).
//   - An expression whose receiver type cannot be resolved WITHOUT GUESSING is skipped and COUNTED
//     (`a.skipped`, printed on every run — the blind spot is visible, never assumed away). A map reached
//     through a LOCAL (`m := l.marketNodes`) does not resolve to the field's identity and is therefore
//     skipped, never assumed safe.
//   - `...Locked` callees are walked once per call site under that call site's held set, DEPTH 1.
//   - It is a STATIC census: it says which lock a source position holds, not that the code runs.

// The pinned identities and the ONE owner mutex. Every pin is ASSERTED by the gate below, so a rename or
// a retype FAILS here and asks for the rule to be re-derived, rather than quietly measuring nothing.
const (
	marketNodesLobbyField  = "Lobby.marketNodes"
	marketNodesRouterField = "TokenSinkRouter.MarketNodes"
	marketNodesOwnerMutex  = "TokenSinkRouter.Mu"
	marketNodesMapType     = "map[string]*EntityMarketNode"

	// marketNodesAccessFloor is a DRIFT FLOOR, not a target: the derived census must keep seeing at
	// least this many DISTINCT access sites to the aliased map, or the walker stopped reaching the code
	// it was written for. The measured value is 30, printed in the census on every run; the floor is set
	// below it so legitimate consolidation does not trip the gate, but a collapse does.
	marketNodesAccessFloor = 24
)

// loAccess is one recorded access to an aliased map field.
type loAccess struct {
	Key     string   // canonical "<Type>.<Field>" identity of the field that was accessed
	Where   string   // file:line
	Func    string   // the enclosing function, so a finding NAMES where it lives
	HeldSet []string // every mutex definitely held at that position (canonical identities, sorted)
}

// loFieldTypeOf returns the DECLARED type of a canonical "<Type>.<Field>" identity, or "".
func loFieldTypeOf(a *loAnalysis, key string) string {
	i := strings.LastIndex(key, ".")
	if i <= 0 {
		return ""
	}
	return a.fieldTypes[key[:i]][key[i+1:]]
}

// mnaContainerOf returns the CONTAINER (struct type) of a canonical "<Type>.<Field>" identity.
func mnaContainerOf(key string) string {
	i := strings.LastIndex(key, ".")
	if i <= 0 {
		return ""
	}
	return key[:i]
}

// loDeriveAliases proves which fields are SECOND NAMES for another field's map, from the tree itself:
// an assignment whose two sides are field selectors of the SAME declared map type binds those two
// identities to ONE map. The pairs are recorded as EVIDENCE (a gate that cannot show its work is a gate
// nobody can adjudicate), and only their keys enter the watch set.
func loDeriveAliases(a *loAnalysis, sources []mutexSource) error {
	for _, s := range sources {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, s.name, s.src, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", s.name, err)
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			vars := loVarsFor(fn)
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				as, ok := n.(*ast.AssignStmt)
				if !ok || len(as.Lhs) != len(as.Rhs) {
					return true
				}
				for i := range as.Lhs {
					lsel, ok1 := as.Lhs[i].(*ast.SelectorExpr)
					rsel, ok2 := as.Rhs[i].(*ast.SelectorExpr)
					if !ok1 || !ok2 {
						continue
					}
					lk, ok3 := loCanonFieldOf(a, vars, lsel)
					rk, ok4 := loCanonFieldOf(a, vars, rsel)
					if !ok3 || !ok4 || lk == rk {
						continue
					}
					lft, rft := loFieldTypeOf(a, lk), loFieldTypeOf(a, rk)
					if lft == "" || lft != rft || !strings.HasPrefix(lft, "map[") {
						continue // a shape is not a map: both sides must be the SAME declared type
					}
					// CROSS-CONTAINER ONLY. `l.a = l.b` binds two fields of the SAME object, and demanding
					// ANOTHER object's mutex for those would be nonsense — the control caught exactly this
					// class on its first run (`l.nodes = l.solo` pulled an unrelated field into the watch
					// set). What makes two SPELLINGS share ONE map is an assignment that crosses the
					// containers, which is precisely the shape in `server.go` / `market_service.go` /
					// `lobby_manager.go` that this gate exists for.
					if mnaContainerOf(lk) == mnaContainerOf(rk) {
						continue
					}
					a.aliasFields[lk] = true
					a.aliasFields[rk] = true
					a.aliasPairs = append(a.aliasPairs, fmt.Sprintf("%s:%d %s = %s (%s)",
						s.name, fset.Position(as.Pos()).Line, lk, rk, lft))
				}
				return true
			})
		}
	}
	sort.Strings(a.aliasPairs)
	return nil
}

// aliasedMapFindings is the ONE predicate, driven by the real-tree gate AND by its control, so the
// control cannot end up agreeing with a re-implementation that measures something else. It returns
// every recorded access to a WATCHED alias field (i.e. one whose declared type is mapType) that does
// not hold ownerMutex, the watch set, and how many watched accesses were examined.
func aliasedMapFindings(a *loAnalysis, mapType, ownerMutex string) (findings []loAccess, watched map[string]bool, accesses int) {
	watched = map[string]bool{}
	for k := range a.aliasFields {
		if loFieldTypeOf(a, k) == mapType {
			watched[k] = true
		}
	}
	for _, acc := range a.accesses {
		if !watched[acc.Key] {
			continue
		}
		accesses++
		if !mnaHeldContains(acc.HeldSet, ownerMutex) {
			findings = append(findings, acc)
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Where != findings[j].Where {
			return findings[i].Where < findings[j].Where
		}
		return findings[i].Key < findings[j].Key
	})
	return findings, watched, accesses
}

func mnaHeldContains(held []string, want string) bool {
	for _, h := range held {
		if h == want {
			return true
		}
	}
	return false
}

func mnaKeysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ============================================================================
// THE GATE, ON THE REAL TREE.
// ============================================================================

func TestNoUnfencedAliasedMapAccess(t *testing.T) {
	sources := loSources(t)
	a, err := loAnalyse(sources)
	if err != nil {
		t.Fatalf("analysis failed: %v", err)
	}

	// 1. THE PIN. A gate whose subject changed under it reports a clean repository for ever, so a
	//    rename, a retype or a re-declared owner mutex FAILS HERE — where the rule must be re-derived —
	//    instead of passing vacuously.
	if got := a.fieldTypes["Lobby"]["marketNodes"]; got != marketNodesMapType {
		t.Fatalf("pin: `Lobby.marketNodes` is declared %q, want %q. The aliased-map rule is derived from\n"+
			"    that type; re-derive the rule before trusting this gate.", got, marketNodesMapType)
	}
	if got := a.fieldTypes["TokenSinkRouter"]["MarketNodes"]; got != marketNodesMapType {
		t.Fatalf("pin: `TokenSinkRouter.MarketNodes` is declared %q, want %q — re-derive the rule.", got, marketNodesMapType)
	}
	if ft := a.fieldTypes["TokenSinkRouter"]["Mu"]; ft != "sync.RWMutex" && ft != "sync.Mutex" {
		t.Fatalf("pin: the owner mutex `%s` is no longer a sync mutex (declared %q).", marketNodesOwnerMutex, ft)
	}

	findings, watched, accesses := aliasedMapFindings(&a, marketNodesMapType, marketNodesOwnerMutex)

	// 2. THE DERIVATION MUST BE NON-EMPTY IN BOTH DIRECTIONS, and must still contain the alias: if the
	//    two fields stopped being the same map, this gate's rule describes nothing.
	if !watched[marketNodesLobbyField] || !watched[marketNodesRouterField] {
		t.Fatalf("derivation found NO alias binding %s and %s (watch set: %v, evidence: %v).\n"+
			"    The rule is: one map, one owner mutex — if the alias is gone, revisit this gate rather\n"+
			"    than deleting it, because the two fields would then need TWO owners or none.",
			marketNodesLobbyField, marketNodesRouterField, mnaKeysOf(watched), a.aliasPairs)
	}
	if accesses == 0 {
		t.Fatal("the walk recorded ZERO accesses to the aliased map — a blind pass reports a clean tree for ever")
	}

	// 3. THE MEASUREMENT. Every access, from EITHER spelling, must hold the owner mutex.
	for _, f := range findings {
		t.Errorf("UNFENCED ALIASED-MAP ACCESS in `%s` at %s — holds {%s}, never `%s`.\n"+
			"    %s and %s are THE SAME map, so a spelling is not a lock: this access can run concurrently\n"+
			"    with an insert on the trade path, and Go answers that with `fatal error: concurrent map read\n"+
			"    and map write` / `concurrent map iteration and map write` — NOT an error value, NOT catchable\n"+
			"    with `recover`, and it takes every connection down with it.\n"+
			"    Fix it by holding `tsr.Mu` (write) around the access, or by walking `tsr.marketNodeRefs()` —\n"+
			"    a value snapshot that RELEASES the owner lock before returning, so a node lock taken\n"+
			"    afterwards cannot invert the `node.Mu -> tsr.Mu` order the investment doors already use.",
			f.Func, f.Where, strings.Join(f.HeldSet, ", "), marketNodesOwnerMutex,
			marketNodesLobbyField, marketNodesRouterField)
	}

	// 4. THE CENSUS — printed on every run, including a clean one, so a reader can see what was measured.
	distinct := map[string]bool{}
	for _, acc := range a.accesses {
		if watched[acc.Key] {
			distinct[acc.Key+" @"+acc.Where] = true
		}
	}
	t.Logf("census: %d files, %d structs, %d alias pair(s) %v, %d recorded accesses (%d distinct site(s)) to the aliased map, %d expressions skipped (never guessed), %d UNFENCED",
		a.files, a.structs, len(a.aliasPairs), a.aliasPairs, accesses, len(distinct), a.skipped, len(findings))

	// 5. THE DRIFT FLOOR: a walker that stops reaching this code must FAIL, not report clean.
	if len(distinct) < marketNodesAccessFloor {
		t.Errorf("the census fell to %d distinct access site(s), below the floor of %d — the walker (or the\n"+
			"    code) changed shape, so re-measure before trusting a clean result", len(distinct), marketNodesAccessFloor)
	}
}

// ============================================================================
// THE CONTROL — a detector that cannot be SHOWN to fail is not a gate.
// ============================================================================
//
// Seven shapes in one synthetic file: THREE that MUST be reported (an unfenced walk that holds the
// WRONG lock, an unlocked walk, and an unfenced rebind) and FOUR that MUST NOT (a properly locked
// rebind, the same walk under the owner lock, a map reached through a legal SNAPSHOT helper, and a map
// of the same TYPE that no assignment ever binds to another field). It drives the SAME
// `aliasedMapFindings` predicate the real-tree gate uses, so the two cannot drift apart.
func TestAliasedMapGateDetectsAnUnfencedAccess(t *testing.T) {
	src := []byte(`package main

import "sync"

type mnaNode struct {
	Mu sync.RWMutex
	V  uint64
}

type mnaRouter struct {
	Mu    sync.RWMutex
	Nodes map[string]*mnaNode
}

type mnaLobby struct {
	mutex sync.RWMutex
	nodes map[string]*mnaNode
	solo  map[string]*mnaNode
	alt   map[string]*mnaNode
	other map[string]uint64
	rt    *mnaRouter
}

// THE ALIAS, and it holds the owner lock (as the real sites now do): MUST NOT be reported.
func (l *mnaLobby) fineRebind() {
	l.rt.Mu.Lock()
	l.nodes = l.rt.Nodes
	l.rt.Mu.Unlock()
}

// MUST REPORT: the WRONG lock (the lobby's) is held — the mistake that reads as safe.
func (l *mnaLobby) badUnfencedRead() {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	for _, n := range l.nodes {
		_ = n
	}
}

// MUST REPORT: a walk with NO map lock at all.
func (l *mnaLobby) badUnlockedRange() {
	for _, n := range l.nodes {
		_ = n
	}
}

// MUST REPORT: an unfenced REBIND — a write to the field, from the other spelling.
func (l *mnaLobby) badUnfencedRebind() {
	l.rt.Nodes = l.nodes
}

// MUST NOT REPORT: the owner lock is held.
func (l *mnaLobby) fineRouterRead() {
	l.rt.Mu.RLock()
	defer l.rt.Mu.RUnlock()
	for _, n := range l.rt.Nodes {
		_ = n
	}
}

// MUST NOT REPORT: a legal SNAPSHOT helper — the owner lock is taken and RELEASED inside it.
func (r *mnaRouter) fineSnapshot() []*mnaNode {
	r.Mu.RLock()
	defer r.Mu.RUnlock()
	out := make([]*mnaNode, 0, len(r.Nodes))
	for _, n := range r.Nodes {
		out = append(out, n)
	}
	return out
}

// MUST NOT REPORT: 'solo' and 'alt' are the same TYPE, and this binds them to EACH OTHER — a bind
// WITHIN one object, which is not the cross-container alias the rule describes — so neither is watched
// and 'other' is a different type entirely. Reading them needs no router lock.
func (l *mnaLobby) fineSameObjectBind() {
	l.solo = l.alt
	for _, n := range l.solo {
		_ = n
	}
	_ = l.other
}

func (l *mnaLobby) rtMap() map[string]uint64 { return nil }
`)
	a, err := loAnalyse([]mutexSource{{name: "mna_control.go", src: src}})
	if err != nil {
		t.Fatalf("control analysis failed: %v", err)
	}

	// The control's OWN pins: if its alias is not derived, the shapes below prove nothing at all.
	watched := map[string]bool{}
	for k := range a.aliasFields {
		if loFieldTypeOf(&a, k) == "map[string]*mnaNode" {
			watched[k] = true
		}
	}
	if !watched["mnaLobby.nodes"] || !watched["mnaRouter.Nodes"] {
		t.Fatalf("control: the alias `l.nodes = l.rt.Nodes` was NOT derived (derived: %v, evidence: %v)",
			mnaKeysOf(a.aliasFields), a.aliasPairs)
	}
	if watched["mnaLobby.solo"] || watched["mnaLobby.alt"] {
		t.Errorf("control: `solo`/`alt` entered the watch set although the only assignment binding them is "+
			"WITHIN one object — a cross-container bind is what proves two SPELLINGS share one map. "+
			"Evidence: %v", a.aliasPairs)
	}

	findings, _, accesses := aliasedMapFindings(&a, "map[string]*mnaNode", "mnaRouter.Mu")
	if accesses == 0 {
		t.Fatal("control: the walk recorded ZERO accesses — nothing is being measured")
	}
	got := map[string]int{}
	for _, f := range findings {
		got[f.Func]++
	}
	for _, want := range []string{"badUnfencedRead", "badUnlockedRange", "badUnfencedRebind"} {
		if got[want] == 0 {
			t.Errorf("control: `%s` MUST be reported and was not (findings by function: %v)", want, got)
		}
	}
	for _, notWant := range []string{"fineRebind", "fineRouterRead", "fineSnapshot", "fineSameObjectBind"} {
		if got[notWant] != 0 {
			t.Errorf("control: `%s` MUST NOT be reported and was %d time(s) (findings by function: %v)",
				notWant, got[notWant], got)
		}
	}
	if len(findings) != 4 {
		t.Errorf("control: want exactly 4 findings — one for the unfenced walk, one for the unlocked range, and "+
			"TWO for the unfenced rebind (the field WRITTEN and the field READ) — got %d: %+v",
			len(findings), findings)
	}
	if got["badUnfencedRebind"] != 2 {
		t.Errorf("control: the unfenced rebind writes one field and reads the other, so it is TWO access "+
			"sites; got %d (findings by function: %v)", got["badUnfencedRebind"], got)
	}
	for _, f := range findings {
		t.Logf("control finding: %s in `%s` holds {%s}", f.Where, f.Func, strings.Join(f.HeldSet, ", "))
	}
}
