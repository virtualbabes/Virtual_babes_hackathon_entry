//go:build !js && !wasm

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// ============================================================================
// THE SELF-LOCK GATE — no function may reach a lobby helper that takes the
// lobby lock ITSELF while it is already holding that lock.
// ============================================================================
//
// WHY THIS EXISTS. `l.mutex` is a `sync.RWMutex`, which is NOT re-entrant. A function that holds it
// and then calls a helper that takes it again does not slow down, does not error, and does not
// return: it DEADLOCKS, and because the write lock is never released it freezes every request and
// every WebSocket in the process — not merely the request that happened to be running.
//
// SEVEN instances of this class have been found in this repository, every one of them BY READING:
// `handleBailCard`, `HandleRepayLoan`, `HandleDetectCounterfeit`, `use_item` ▸ `legal_pardon`,
// `handleCyberIntercept`, `handleKidnapRequest` and the `sendToClient` family below. Reading is not a
// gate, and most of them sat behind DEAD arithmetic or an unexercised branch, so the code simply
// never ran and no behaviour test could have caught them. This gate measures the SHAPE instead.
//
// WHAT IT MEASURES, in one line: within ONE function body, is a call to a SELF-LOCKING lobby helper
// reachable at a source position where the lobby lock is DEFINITELY held?
//
// THE REGISTRY IS DERIVED, NOT CURATED. The first pass asks the TREE which `*Lobby` methods take
// `l.mutex` themselves; the second pass reports every call to one of them from under the lock. A
// helper added tomorrow is therefore measured tomorrow — a hand-written list would have gone stale on
// the first new service, and a stale gate reads exactly like a clean one.
//
// FOUR DECISIONS CARRY THE WEIGHT:
//
//   - MUST, NOT MAY. The lock counts as held only when it is held on EVERY path to the call
//     (`if`, `else`, `switch`/`select` clauses and loops are joined accordingly). The previous
//     version of this gate walked flat source order, and a `l.mutex.Lock()` inside ONE `switch`
//     case then poisoned every LATER case: 40 of its 44 reports were that single false positive
//     (`handleGameProtocol`'s `use_item` case), which is precisely how a gate gets switched off.
//   - A DEFERRED RELEASE releases at FUNCTION EXIT, so it does NOT clear the held state where it is
//     written. That is what makes the dominant spelling of this defect — `Lock(); defer Unlock();
//     …call` — visible instead of invisible.
//   - THE RECEIVER'S TYPE DECIDES. Only an identifier typed `*Lobby` in that function (its receiver
//     or a parameter) can touch the lobby mutex or call a lobby method, so
//     `stats.CareerXP.TrackCareerXP(…)` and any other type's `mutex` field are not even candidates.
//     Measured support: all 557 Lobby methods in this tree use receiver `l`, and NO type embeds
//     `*Lobby` (a type that did would need the environment below extending — that absence is what
//     this assumption rests on).
//   - `fallthrough` IS MODELLED, because the tree uses it twice (`resilience_utils.go`): the state
//     flows INTO the next clause instead of out of the switch.
//
// HONEST LIMITS (stated, not implied):
//   - THE ANALYSIS IS INTRA-PROCEDURAL. It decides whether ONE function body holds the lock at a call
//     site. A function that calls a self-locking helper WITHOUT holding the lock is not a violation,
//     even when one of ITS callers holds the lock — that second function would need its own `...Locked`
//     sibling, and finding it from here needs a call graph this gate does not build. So the guarantee is
//     precise: no function reaches a self-locking helper while holding the lock ITSELF, which is the
//     shape of every one of the seven instances found in this repository.
//   - A FUNCTION LITERAL (`func(){…}()`, `go func(){…}()`) is its own scope and is not descended into:
//     a goroutine's lock discipline cannot be decided statically. A `go` STATEMENT is modelled exactly
//     as far as Go's semantics allow — its CALLEE is NOT a call site (a new goroutine does not inherit
//     the lock, so it cannot deadlock; it only waits), while its ARGUMENTS still are (`go f(x)`
//     evaluates `x` in the calling goroutine). Both halves are pinned by a control below.
//   - A DEFERRED call is reported in the state at its `defer` STATEMENT, and a deferred RELEASE does not
//     clear that state. For the spelling this tree uses (`Lock(); defer Unlock(); …; defer g()`) the
//     deferred call does run BEFORE the deferred unlock, so the report is correct; the reverse
//     registration order would be safe and is NOT modelled. Measured: the tree contains no `defer` of a
//     self-locking helper at all, so this limit currently decides nothing.
//   - A `break`/`goto` that CARRIES a freshly taken lock out of a loop or switch is not modelled, so
//     a call after such a jump is NOT reported (the gate errs toward silence there, never toward noise).
//   - Assignments through an untyped identifier (`x := l.something()`) leave the type unknown, so a
//     call made through `x` is skipped rather than guessed.
//   - `_test.go` files are NOT scanned: a test that holds the lock on purpose is TESTING the lock.
//   - A READ lock held while the callee takes a READ lock is a VIOLATION, exactly like the write
//     case. Go's RWMutex blocks a recursive read lock the moment a writer queues, and this goroutine
//     never releases the first one, so the process freezes — and the write lock is taken by
//     matchmaking, club, market, treasury and admin paths constantly, so "a writer is already
//     waiting" is the normal state under load rather than a corner case. It was a NOTE until the
//     pass below measured THREE LIVE SITES of it (a GET handler, a per-club loop and a per-player
//     loop); all three are fixed, and the shape is fatal here so it cannot come back.
// ============================================================================

// heldKind is the lobby lock state that is DEFINITELY held at a source position.
type heldKind int

const (
	heldNone heldKind = iota
	heldRead
	heldWrite
)

func (h heldKind) String() string {
	switch h {
	case heldRead:
		return "RLock"
	case heldWrite:
		return "Lock"
	}
	return "no lock"
}

// joinMust keeps a lock state only when BOTH paths reaching a point agree on it.
func joinMust(a, b heldKind) heldKind {
	if a == b {
		return a
	}
	return heldNone
}

// lobbyHelper is ONE Lobby method that takes the lobby lock ITSELF.
type lobbyHelper struct {
	name    string
	write   bool   // it takes the write lock
	read    bool   // it takes the read lock
	sibling string // the `...Locked` form beside it, when this tree has one
}

func (h *lobbyHelper) locks() string {
	switch {
	case h.write && h.read:
		return "Lock and RLock"
	case h.write:
		return "Lock"
	}
	return "RLock"
}

// deadlocksUnder answers the only question that matters at a call site: can a callee that takes the
// lobby lock ITSELF ever return when the SAME goroutine already holds it?
//
// The answer is NO for every combination, which is why this is not `h.write || held == heldWrite`:
//
//	held write + wants write/read → the write lock is never released
//	held read  + wants write      → Lock waits for the read lock this goroutine never releases
//	held read  + wants read       → Go's RWMutex blocks a second RLock the moment a writer queues,
//	                                and the first RLock is still held, so the WRITER is what
//	                                decides; the write lock is taken by matchmaking, club, market,
//	                                treasury and admin paths constantly, so the queue is the normal
//	                                state under load rather than a corner case.
//
// The read+read case used to be a NOTE here. The transitive pass measured THREE live sites of it (a
// GET handler, a per-club loop and a per-player loop), which is why it is now a failure — and why
// `TestSelfLockGateDetectsAViolation` carries a `badRecursiveRead` shape, so a future edit that
// softens this back to a note fails the build instead of quietly widening the hole.
func (h *lobbyHelper) deadlocksUnder(held heldKind) bool {
	return held != heldNone
}

func (h *lobbyHelper) reason() string {
	if h.sibling != "" {
		return fmt.Sprintf("it takes the lobby lock ITSELF (%s), so a caller that already holds it must call %s()", h.locks(), h.sibling)
	}
	return fmt.Sprintf("it takes the lobby lock ITSELF (%s) and has NO lock-held form, so a caller that already holds it must DROP the lock first", h.locks())
}

// reasonAt names the mechanism AT THE CALL SITE, because a recursive WRITE lock and a recursive READ
// lock fail identically but for different reasons, and a message that hides which one it is makes
// the first reader go and work it out.
func (h *lobbyHelper) reasonAt(held heldKind) string {
	if held == heldRead && !h.write {
		return "it takes RLock, and a recursive READ lock blocks for ever the moment a writer queues on the same sync.RWMutex (Go prohibits recursive read-locking) — " + h.reason()
	}
	return h.reason()
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func receiverTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	t := recv.List[0].Type
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	if id, ok := t.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

func funcLabel(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name + "()"
	}
	return receiverTypeName(fn.Recv) + "." + fn.Name.Name + "()"
}

// lobbyTypedIdents returns the identifiers in ONE function that ARE a *Lobby: its receiver and its
// parameters of that type. This is the gate's whole soundness story — a call is only a candidate
// when the receiver of the call is one of these.
func lobbyTypedIdents(fn *ast.FuncDecl) map[string]bool {
	names := map[string]bool{}
	add := func(fl *ast.FieldList) {
		if fl == nil {
			return
		}
		for _, f := range fl.List {
			inner := f.Type
			if star, ok := inner.(*ast.StarExpr); ok {
				inner = star.X
			}
			id, isIdent := inner.(*ast.Ident)
			if !isIdent || id.Name != "Lobby" {
				continue
			}
			for _, n := range f.Names {
				names[n.Name] = true
			}
		}
	}
	add(fn.Recv)
	add(fn.Type.Params)
	return names
}

// lobbyTree is the whole measurement of the tree: the derived helpers, the parsed files, and the
// list of non-test sources actually scanned.
type lobbyTree struct {
	fset    *token.FileSet
	helpers map[string]*lobbyHelper
	parsed  map[string]*ast.File
	scanned []string

	// The INTERPROCEDURAL index, built beside the helpers and read only by the transitive pass:
	// every declared body by key, its file, the method/function key sets, and the unique-name index.
	decls   map[string]*ast.FuncDecl
	files   map[string]string
	methods map[string]bool
	funcs   map[string]bool
	byName  map[string][]string
}

// funcDeclKey is the identity of a declared body: "Type.Method" for a method, "Name" for a function.
// It is deliberately distinct from funcLabel, which is the HUMAN label used in messages.
func funcDeclKey(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	return receiverTypeName(fn.Recv) + "." + fn.Name.Name
}

// deriveHelpers answers the gate's question about the TREE: which `*Lobby` methods take the lobby
// lock THEMSELVES. It is separate from the scan so the controls below can drive it with synthetic
// sources rather than trusting it.
func deriveHelpers(fset *token.FileSet, parsed map[string]*ast.File, scanned []string) map[string]*lobbyHelper {
	helpers := map[string]*lobbyHelper{}
	declared := map[string]bool{}
	for _, name := range scanned {
		file := parsed[name]
		if file == nil {
			continue
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || receiverTypeName(fn.Recv) != "Lobby" {
				continue
			}
			declared[fn.Name.Name] = true
			a := &lockAuditor{
				fset: fset, filename: name, funcName: funcLabel(fn),
				lobbyNames: lobbyTypedIdents(fn),
			}
			a.block(fn.Body.List, heldNone)
			if !a.acquiredRead && !a.acquiredWrite {
				continue
			}
			h := helpers[fn.Name.Name]
			if h == nil {
				h = &lobbyHelper{name: fn.Name.Name}
				helpers[fn.Name.Name] = h
			}
			h.read = h.read || a.acquiredRead
			h.write = h.write || a.acquiredWrite
		}
	}
	for _, h := range helpers {
		if sib := lowerFirst(h.name) + "Locked"; declared[sib] {
			h.sibling = sib
		}
	}
	return helpers
}

// lockAuditor walks ONE function body, tracking the lobby lock that is DEFINITELY held, and reports
// every call that reaches a self-locking Lobby helper while it is.
//
// It is used for BOTH passes: with `registry` nil it only records what the function acquires (the
// derivation), and with a registry it reports the violations (the scan).
type lockAuditor struct {
	fset       *token.FileSet
	filename   string
	funcName   string
	lobbyNames map[string]bool
	registry   map[string]*lobbyHelper

	// derivation
	acquiredRead  bool
	acquiredWrite bool

	// reporting (nil while deriving)
	violations *[]string

	// INTERPROCEDURAL recording (nil everywhere the intra-procedural gate runs, which is what keeps
	// that gate's behaviour unchanged). When `sites` is set, every resolved CALL EDGE is recorded
	// with the lock state reached there, so the same walk can be replayed under an ENTRY state and
	// the held lock propagated across the edge.
	methods map[string]bool     // "Type.Method" keys declared in the tree
	funcs   map[string]bool     // package-level function names
	byName  map[string][]string // method name -> declaring keys (the unique-name rule)
	sites   *[]selfLockSite
}

// selfLockSite is ONE resolved call edge with the lobby lock that is DEFINITELY held there. The
// line is recorded so a finding can name the exact call site, and the index within a body's slice
// is stable across entry states (the walk is deterministic), which is what lets the transitive pass
// compare "held only because of the entry state" against the intra-procedural baseline.
type selfLockSite struct {
	callee string // "Lobby.Method" or "funcName"; "" when it cannot be resolved
	line   int
	held   heldKind
	helper string // the self-locking helper this calls, when it is one
}

// recordSite appends ONE edge when the interprocedural pass asked for recording.
func (a *lockAuditor) recordSite(callee string, c *ast.CallExpr, in heldKind) {
	if a.sites == nil {
		return
	}
	s := selfLockSite{callee: callee, line: a.fset.Position(c.Pos()).Line, held: in}
	if strings.HasPrefix(callee, "Lobby.") {
		if h := a.registry[strings.TrimPrefix(callee, "Lobby.")]; h != nil {
			s.helper = h.name
		}
	}
	*a.sites = append(*a.sites, s)
}

// rootIdentLocal walks a selector/index/star/paren chain to the identifier it starts from
// (`l.svc.x` -> `l`) and returns nil for a chain rooted in a CALL, which parsing cannot type.
func rootIdentLocal(e ast.Expr) *ast.Ident {
	for {
		switch x := e.(type) {
		case *ast.Ident:
			return x
		case *ast.SelectorExpr:
			e = x.X
		case *ast.StarExpr:
			e = x.X
		case *ast.IndexExpr:
			e = x.X
		case *ast.ParenExpr:
			e = x.X
		default:
			return nil
		}
	}
}

func (a *lockAuditor) block(list []ast.Stmt, in heldKind) heldKind {
	cur := in
	for _, s := range list {
		cur = a.stmt(s, cur)
	}
	return cur
}

func (a *lockAuditor) stmt(s ast.Stmt, in heldKind) heldKind {
	switch n := s.(type) {
	case *ast.BlockStmt:
		return a.block(n.List, in)
	case *ast.IfStmt:
		cur := in
		if n.Init != nil {
			cur = a.stmt(n.Init, cur)
		}
		// The CONDITION is a call site like any other (`if !l.isWalletRegistered(w)`), and it is
		// evaluated on BOTH paths, so it is walked here. It cannot change the held state: neither
		// Lock() nor RLock() returns a value, so neither can appear in a condition.
		a.simple(n.Cond, cur)
		// The two branches are exclusive, so the state afterwards must hold on BOTH of them.
		thenOut := a.block(n.Body.List, cur)
		elseOut := cur
		if n.Else != nil {
			elseOut = a.stmt(n.Else, cur)
		}
		return joinMust(thenOut, elseOut)
	case *ast.SwitchStmt:
		cur := in
		if n.Init != nil {
			cur = a.stmt(n.Init, cur)
		}
		if n.Tag != nil {
			a.simple(n.Tag, cur)
		}
		return a.clauses(n.Body.List, cur)
	case *ast.TypeSwitchStmt:
		cur := in
		if n.Init != nil {
			cur = a.stmt(n.Init, cur)
		}
		if n.Assign != nil {
			cur = a.stmt(n.Assign, cur)
		}
		return a.clauses(n.Body.List, cur)
	case *ast.SelectStmt:
		return a.clauses(n.Body.List, in)
	case *ast.ForStmt:
		cur := in
		if n.Init != nil {
			cur = a.stmt(n.Init, cur)
		}
		if n.Cond != nil {
			a.simple(n.Cond, cur)
		}
		if n.Post != nil {
			a.simple(n.Post, cur)
		}
		// A loop need not run, so it never leaves the lock DEFINITELY held; the body is still walked
		// with the state it starts from, which is what makes a violation inside the body visible.
		a.block(n.Body.List, cur)
		return in
	case *ast.RangeStmt:
		a.simple(n.X, in)
		a.block(n.Body.List, in)
		return in
	case *ast.LabeledStmt:
		return a.stmt(n.Stmt, in)
	case *ast.DeferStmt:
		// A DEFERRED call runs at FUNCTION EXIT, so a deferred RELEASE does not clear the held state
		// here (this is what makes `Lock(); defer Unlock(); …call` visible), while a deferred CALL to
		// a self-locking helper is still a call that will run while the lock may be held.
		var calls []*ast.CallExpr
		collectCalls(n.Call, &calls)
		for _, c := range calls {
			if a.isLobbyLockRelease(c) {
				continue
			}
			a.call(c, in)
		}
		return in
	case *ast.GoStmt:
		// A `go` STATEMENT STARTS A NEW GOROUTINE, AND A GOROUTINE DOES NOT INHERIT THE LOCK. Its
		// callee therefore runs with NO lock held, so `go l.sendNoteTx(…)` under `Lock()` cannot
		// deadlock — it blocks until the parent releases, which it does. Treating the callee as a
		// call site here was a FALSE POSITIVE on real, correct code (market_service.go, twice), and a
		// gate that reports correct code is a gate that gets switched off.
		//
		// ITS ARGUMENTS ARE A DIFFERENT MATTER: `go f(x)` evaluates `x` in the CALLING goroutine, so a
		// self-locking helper reached from an argument IS still under the held lock. Both halves of
		// that distinction are pinned by a control below (`badInGoArgument` / `fineInNewGoroutine`).
		a.goCall(n.Call, in)
		return in
	}
	return a.simple(s, in)
}

// goCall walks the parts of a `go f(…)` statement that RUN IN THE CALLING GOROUTINE — every argument,
// and the receiver expression of the callee — and deliberately NOT the callee itself, which is the one
// part that runs in the new goroutine. It is the mirror image of collectCalls, descending the same two
// places for the opposite reason.
func (a *lockAuditor) goCall(c *ast.CallExpr, in heldKind) {
	for _, arg := range c.Args {
		a.simple(arg, in)
	}
	if sel, ok := c.Fun.(*ast.SelectorExpr); ok {
		a.simple(sel.X, in)
	}
}

// isLobbyLockRelease recognises `l.mutex.Unlock()` / `l.mutex.RUnlock()` on a *Lobby identifier.
func (a *lockAuditor) isLobbyLockRelease(c *ast.CallExpr) bool {
	sel, ok := c.Fun.(*ast.SelectorExpr)
	if !ok || (sel.Sel.Name != "Unlock" && sel.Sel.Name != "RUnlock") {
		return false
	}
	inner, ok := sel.X.(*ast.SelectorExpr)
	if !ok || inner.Sel.Name != "mutex" {
		return false
	}
	base, ok := inner.X.(*ast.Ident)
	return ok && a.lobbyNames[base.Name]
}

// clauses walks the clauses of a switch/type-switch/select. Each clause is an EXCLUSIVE path, so a
// lock taken inside one is not held inside the next: every clause starts from the state that reaches
// the switch. The state after the switch must therefore hold on EVERY clause (and on the pre-state
// itself when there is no `default`, because then no clause may run at all).
func (a *lockAuditor) clauses(list []ast.Stmt, in heldKind) heldKind {
	var outs []heldKind
	hasDefault := false
	carry := false
	prevOut := heldNone
	for _, c := range list {
		var body []ast.Stmt
		switch clause := c.(type) {
		case *ast.CaseClause:
			body = clause.Body
			if clause.List == nil {
				hasDefault = true
			}
		case *ast.CommClause:
			body = clause.Body
			if clause.Comm == nil {
				hasDefault = true
			} else {
				// The channel expression of a `case` is evaluated while choosing a clause.
				a.simple(clause.Comm, in)
			}
		default:
			continue
		}
		start := in
		if carry {
			start = joinMust(in, prevOut)
		}
		carry = false
		out := a.block(body, start)
		if n := len(body); n > 0 {
			if br, ok := body[n-1].(*ast.BranchStmt); ok && br.Tok == token.FALLTHROUGH {
				// `fallthrough` does NOT exit the switch: the state flows INTO the next clause.
				prevOut, carry = out, true
				continue
			}
		}
		outs = append(outs, out)
	}
	if !carry && !hasDefault {
		outs = append(outs, in)
	}
	if len(outs) == 0 {
		return in
	}
	res := outs[0]
	for _, o := range outs[1:] {
		res = joinMust(res, o)
	}
	return res
}

// simple walks the calls inside ONE expression or statement in the order they RUN (arguments first),
// which is the only place two calls in one statement could matter.
func (a *lockAuditor) simple(node ast.Node, in heldKind) heldKind {
	var calls []*ast.CallExpr
	collectCalls(node, &calls)
	cur := in
	for _, c := range calls {
		cur = a.call(c, cur)
	}
	return cur
}

// collectCalls emits the calls inside ONE node ARGUMENTS FIRST, and never descends into a function
// literal (its lock discipline is its own).
func collectCalls(node ast.Node, out *[]*ast.CallExpr) {
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncLit:
			return false
		case *ast.CallExpr:
			for _, arg := range x.Args {
				collectCalls(arg, out)
			}
			if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
				collectCalls(sel.X, out)
			}
			*out = append(*out, x)
			return false
		}
		return true
	})
}

// call applies ONE call to the held-lock state, and reports it when it reaches a self-locking Lobby
// helper from under the lock.
func (a *lockAuditor) call(c *ast.CallExpr, in heldKind) heldKind {
	sel, ok := c.Fun.(*ast.SelectorExpr)
	if !ok {
		// A BARE identifier that names a package function is an edge too: that is how a free
		// function taking `l *Lobby` sits between a lock holder and a self-locking helper.
		if id, isIdent := c.Fun.(*ast.Ident); isIdent && a.funcs != nil && a.funcs[id.Name] {
			a.recordSite(id.Name, c, in)
		}
		return in
	}

	// 1. The lobby mutex itself: `l.mutex.Lock()` and friends, on a *Lobby identifier.
	if inner, isSel := sel.X.(*ast.SelectorExpr); isSel && inner.Sel.Name == "mutex" {
		base, isIdent := inner.X.(*ast.Ident)
		if !isIdent || !a.lobbyNames[base.Name] {
			return in
		}
		switch sel.Sel.Name {
		case "Lock":
			a.acquiredWrite = true
			return heldWrite
		case "RLock":
			a.acquiredRead = true
			return heldRead
		case "Unlock", "RUnlock":
			return heldNone
		}
		return in
	}

	// 2. A call to a Lobby method, on a *Lobby identifier.
	base, isIdent := sel.X.(*ast.Ident)
	if !isIdent || !a.lobbyNames[base.Name] {
		// 2b. A call THROUGH something the lobby holds (`l.auctionService.ProcessAuctions(l)`): the
		// receiver chain is rooted at a *Lobby identifier, so this is a lobby-internal call one
		// FIELD deep. Parsing alone cannot type `l.auctionService`, so it is resolved by UNIQUE
		// method name and left unresolved when ambiguous — guessing would invent edges, and a gate
		// that reports edges it invented is a gate that gets switched off.
		if a.byName != nil {
			if root := rootIdentLocal(sel.X); root != nil && a.lobbyNames[root.Name] {
				if keys := a.byName[sel.Sel.Name]; len(keys) == 1 {
					a.recordSite(keys[0], c, in)
				}
			}
		}
		return in
	}
	a.recordSite("Lobby."+sel.Sel.Name, c, in)
	h := a.registry[sel.Sel.Name]
	if h == nil || in == heldNone {
		return in
	}
	where := fmt.Sprintf("%s:%d: %s holds %s", a.filename, a.fset.Position(c.Pos()).Line, a.funcName, in)
	if h.deadlocksUnder(in) && a.violations != nil {
		*a.violations = append(*a.violations, fmt.Sprintf(
			"%s and calls %s() — %s. A sync.RWMutex is not re-entrant, so this call can never return and the lock is never released.",
			where, h.name, h.reasonAt(in)))
	}
	return in
}

// lobbySelfLockViolations reports every call in ONE parsed source file that reaches a self-locking
// Lobby helper while the lobby lock is definitely held.
//
// It takes SOURCE BYTES, not a filename, so the gate's own detector can be driven with a synthetic
// file. Without that, the gate would be an untested assertion about itself.
func lobbySelfLockViolations(filename string, src []byte, registry map[string]*lobbyHelper) ([]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	return lobbySelfLockViolationsInFile(fset, filename, file, registry), nil
}

func lobbySelfLockViolationsInFile(fset *token.FileSet, filename string, file *ast.File, registry map[string]*lobbyHelper) []string {
	var out []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		a := &lockAuditor{
			fset: fset, filename: filename, funcName: funcLabel(fn),
			lobbyNames: lobbyTypedIdents(fn), registry: registry,
			violations: &out,
		}
		a.block(fn.Body.List, heldNone)
	}
	return out
}

// loadLobbyTree parses every non-test Go source in the PACKAGE ROOT — the tree the gate claims to
// cover, and the tree every instance of this class was found in — then derives the helpers from it.
//
// Note the build tag: this whole file is `!js && !wasm`, because a client build has no lobby lock at
// all. That is also why the gate cannot run in the WASM target.
func loadLobbyTree(t *testing.T) *lobbyTree {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob *.go: %v", err)
	}
	if len(files) == 0 {
		// A gate that looked at nothing reports nothing. Never let that read as a pass.
		t.Fatal("no Go sources found in the working directory — this gate must never pass vacuously")
	}
	fset := token.NewFileSet()
	tree := &lobbyTree{fset: fset, helpers: map[string]*lobbyHelper{}, parsed: map[string]*ast.File{}}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue // a test that holds the lock on purpose is TESTING the lock
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		file, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		tree.parsed[name] = file
		tree.scanned = append(tree.scanned, name)
	}
	sort.Strings(tree.scanned)
	indexLobbyTree(tree)
	return tree
}

// indexLobbyTree derives the helper registry and indexes every declared body. BOTH passes read the
// SAME parse, so the registry the transitive pass propagates over is the one the gate reports from —
// two registries would be two answers to one question.
//
// A key declared twice (a build-tagged twin, `js/wasm` beside the server) keeps the LAST file in
// `scanned` order; that is stated here because a silently-chosen twin is how a gate goes blind.
func indexLobbyTree(tree *lobbyTree) {
	tree.helpers = deriveHelpers(tree.fset, tree.parsed, tree.scanned)
	tree.decls = map[string]*ast.FuncDecl{}
	tree.files = map[string]string{}
	tree.methods = map[string]bool{}
	tree.funcs = map[string]bool{}
	tree.byName = map[string][]string{}
	for _, name := range tree.scanned {
		file := tree.parsed[name]
		if file == nil {
			continue
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			key := funcDeclKey(fn)
			tree.decls[key] = fn
			tree.files[key] = name
			if fn.Recv == nil {
				tree.funcs[fn.Name.Name] = true
				continue
			}
			tree.methods[key] = true
			tree.byName[fn.Name.Name] = append(tree.byName[fn.Name.Name], key)
		}
	}
}

// TestNoSelfLockingLobbyHelper is the gate itself: no function in this package may reach a lobby
// helper that takes the lobby lock ITSELF while it is already holding that lock.
//
// It scans the PACKAGE ROOT, because that is where every instance of this class was found, and
// because a source gate built on a hand-written file list goes stale the first time a file is added.
func TestNoSelfLockingLobbyHelper(t *testing.T) {
	tree := loadLobbyTree(t)
	if len(tree.scanned) < 50 {
		t.Errorf("scanned only %d non-test Go files; the package root holds far more, so this gate is not looking at the tree it claims to", len(tree.scanned))
	}

	// The DERIVATION must see the helpers that are known to take the lock. A derivation that
	// silently returned nothing would report a clean repository for ever, and nobody would know.
	for _, must := range []string{"TrackCareerXP", "sendToClient", "broadcastToAdmins", "logAdminAudit", "isJusticeAligned", "isWalletRegistered"} {
		h := tree.helpers[must]
		if h == nil {
			t.Errorf("the derivation did not see Lobby.%s() take the lobby lock — the gate is blind to it", must)
			continue
		}
		if h.sibling == "" {
			t.Logf("note: Lobby.%s() is self-locking (%s) with no `...Locked` form; a caller holding the lock must drop it first", must, h.locks())
		}
	}

	var violations []string
	for _, name := range tree.scanned {
		violations = append(violations, lobbySelfLockViolationsInFile(tree.fset, name, tree.parsed[name], tree.helpers)...)
	}
	sort.Strings(violations)
	for _, v := range violations {
		t.Errorf("SELF-LOCK: %s", v)
	}
	t.Logf("self-lock gate: %d non-test Go files scanned, %d self-locking Lobby helpers, %d violations",
		len(tree.scanned), len(tree.helpers), len(violations))
}

// TestSelfLockDerivationSeesTheRightMethods is the DERIVATION's own control. Detection is only as
// good as the registry it is given, so the registry must be measured too: it must recognise a method
// that takes the lock, must NOT mistake a `...Locked` form for one, and must connect a helper to its
// sibling so the failure message can name the fix.
func TestSelfLockDerivationSeesTheRightMethods(t *testing.T) {
	const synthetic = `package main

func (l *Lobby) selfLocking() {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	_ = l.leaderboard
}

func (l *Lobby) selfLockingLocked() {
	_ = l.leaderboard
}

func (l *Lobby) takesNothing() {
	_ = Envelope{}
}

func (s *Service) notTheLobby(l *Lobby) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "derive_synthetic.go", []byte(synthetic), parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("the derivation control must parse a well-formed file: %v", err)
	}
	helpers := deriveHelpers(fset, map[string]*ast.File{"derive_synthetic.go": file}, []string{"derive_synthetic.go"})

	h := helpers["selfLocking"]
	if h == nil {
		t.Fatalf("the derivation MISSED Lobby.selfLocking(), which takes the read lock — it cannot see the shape it exists for")
	}
	if !h.read || h.write {
		t.Errorf("Lobby.selfLocking() takes RLock and no Lock; derived read=%v write=%v", h.read, h.write)
	}
	if h.sibling != "selfLockingLocked" {
		t.Errorf("the derivation must connect the helper to its `...Locked` sibling so the failure names the fix; got %q", h.sibling)
	}
	for _, mustNot := range []string{"selfLockingLocked", "takesNothing", "notTheLobby"} {
		if _, ok := helpers[mustNot]; ok {
			t.Errorf("%s was derived as a self-locking Lobby helper, which it is not — a registry that cries wolf makes the gate noise", mustNot)
		}
	}
}

// TestSelfLockGateDetectsAViolation is the NEGATIVE CONTROL, and it is the part that makes the gate
// above mean anything: a detector whose matcher silently failed would report a clean repository for
// ever, and the clean report is exactly what the real scan asserts about the whole tree.
//
// The synthetic file below carries THIRTEEN shapes. EIGHT must be reported — including a RECURSIVE
// READ lock, which was a NOTE until three live sites of it were measured — and FIVE must not: they
// are the false positives that would make this gate unusable, and THREE of them are the specific
// shapes that broke the previous flat-source-order version of this gate.
func TestSelfLockGateDetectsAViolation(t *testing.T) {
	const synthetic = `package main

func (l *Lobby) badDeferredUnlock() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.sendToClient("0xbad", Envelope{})
}

func (l *Lobby) badReadThenWrite() {
	l.mutex.RLock()
	_ = l.leaderboard
	l.claimDividends("0xbad")
	l.mutex.RUnlock()
}

// A RECURSIVE READ lock: a read lock is held and the callee only wants a read lock. Go's RWMutex
// blocks that for ever the moment a writer queues, so it is a deadlock like any other here.
func (l *Lobby) badRecursiveRead() {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	l.sendToClient("0xbad", Envelope{})
}

func (l *Lobby) badInOneSwitchCase() {
	switch "x" {
	case "a":
		l.mutex.Lock()
		defer l.mutex.Unlock()
		l.sendToClient("0xbad", Envelope{})
	case "b":
		_ = l.leaderboard
	}
}

func (l *Lobby) badInBothBranches() {
	if true {
		l.mutex.Lock()
		defer l.mutex.Unlock()
	} else {
		l.mutex.Lock()
		defer l.mutex.Unlock()
	}
	l.sendToClient("0xbad", Envelope{})
}

func (bs *Service) badThroughALobbyParameter(l *Lobby) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.sendToClient("0xbad", Envelope{})
}

func (l *Lobby) badInIfCondition() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.isWalletRegistered("0xbad") {
		_ = l.leaderboard
	}
}

// The ARGUMENT of a "go" statement runs in the CALLING goroutine, so a self-locking helper reached from
// it is still under the held lock. (envelopeOf is not a Lobby method and is not expected to report.)
func (l *Lobby) badInGoArgument() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	go l.sendToClientLocked("0xbad", l.envelopeOf(l.isWalletRegistered("0xbad")))
}

func (l *Lobby) fineSiblingForm() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.sendToClientLocked("0xok", Envelope{})
}

func (l *Lobby) fineOtherCaseDoesNotHold() {
	switch "x" {
	case "a":
		l.mutex.Lock()
		defer l.mutex.Unlock()
		l.sendToClientLocked("0xok", Envelope{})
	case "b":
		l.sendToClient("0xok", Envelope{})
	}
}

// A NEW GOROUTINE DOES NOT INHERIT THE LOCK. A "go l.sendToClient(...)" written under Lock() blocks until
// the parent releases (which it does, at function exit), so it cannot deadlock. This is the exact shape
// the real tree uses twice (go l.sendNoteTx(...) in market_service.go), and reporting it was a false
// positive on correct code.
func (l *Lobby) fineInNewGoroutine() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	go l.sendToClient("0xok", Envelope{})
}

func (l *Lobby) fineReleasedFirst() {
	l.mutex.Lock()
	l.mutex.Unlock()
	l.sendToClient("0xok", Envelope{})
}

func (l *Lobby) fineNotALobbyReceiver(x *SomethingElse) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	x.sendToClient("0xok", Envelope{})
}
`
	// The registry the real gate DERIVES; here it is declared, so the control measures the DETECTOR.
	registry := map[string]*lobbyHelper{
		"sendToClient":       {name: "sendToClient", read: true, sibling: "sendToClientLocked"},
		"claimDividends":     {name: "claimDividends", write: true},
		"isWalletRegistered": {name: "isWalletRegistered", read: true},
	}

	violations, err := lobbySelfLockViolations("synthetic.go", []byte(synthetic), registry)
	if err != nil {
		t.Fatalf("the detector must parse a well-formed file: %v", err)
	}
	report := strings.Join(violations, "\n")

	if len(violations) != 8 {
		t.Fatalf("expected exactly 8 violations (deferred unlock, read-then-write, recursive read, one switch case, both branches, a *Lobby parameter, an if-condition, a `go` argument); got %d:\n%s", len(violations), report)
	}
	for _, want := range []string{"badDeferredUnlock", "badReadThenWrite", "badRecursiveRead", "badInOneSwitchCase", "badInBothBranches", "badThroughALobbyParameter", "badInIfCondition", "badInGoArgument"} {
		if !strings.Contains(report, want) {
			t.Errorf("the detector MISSED %s() — it cannot see the shape it exists for:\n%s", want, report)
		}
	}
	for _, mustNot := range []string{"fineSiblingForm", "fineOtherCaseDoesNotHold", "fineReleasedFirst", "fineNotALobbyReceiver", "fineInNewGoroutine"} {
		if strings.Contains(report, mustNot) {
			t.Errorf("the detector reported %s(), which is correct code — a gate that cries wolf gets switched off:\n%s", mustNot, report)
		}
	}
	if strings.Contains(report, "case \"b\"") {
		t.Errorf("a clause that never holds the lock was reported — clause exclusivity is broken again:\n%s", report)
	}

	// The failure message must NAME the fix, or the first reader has to go and find it.
	if !strings.Contains(report, "sendToClientLocked") {
		t.Errorf("a violation naming a helper with a `...Locked` sibling must name that sibling:\n%s", report)
	}

	// It must also be able to report NOTHING: a gate that always fires is as useless as one that
	// never does, and the clean case is what the real scan asserts about the whole tree.
	clean, err := lobbySelfLockViolations("clean.go", []byte("package main\n\nfunc (l *Lobby) ok() {}\n"), registry)
	if err != nil {
		t.Fatalf("parse clean.go: %v", err)
	}
	if len(clean) != 0 {
		t.Errorf("a clean file reported %d violations: %v", len(clean), clean)
	}
}

// TestTheRecommendedLockedFormsRunWhileTheWriteLockIsHeld is the BEHAVIOURAL half of this gate. The
// failure message above tells a caller to use the `...Locked` sibling, and TWO of those siblings did not
// exist until this change (`broadcastToAdminsLocked`, `isWalletRegisteredLocked`) — so the
// recommendation has to be PROVED usable rather than merely declared, or the gate would be advising a
// fix that is not there.
//
// It fails by TIMEOUT rather than by assertion, because a self-lock raises no error at all: it simply
// never returns, and the write lock is never released.
// ============================================================================
// THE TRANSITIVE PASS — the blind spot the gate above states it has.
// ============================================================================
//
// WHY IT EXISTS. The gate above decides whether ONE body holds the lock at a call site, so it is
// satisfied by any function that is merely CALLED under the lock:
//
//	F holds l.mutex  ->  calls G   (G takes no lock itself)     <- invisible to the gate above
//	                 ->  G calls H (H takes l.mutex itself)     <- and this is the freeze
//
// `G` never acquires the lock, so the rule "no function reaches a self-locking helper while holding
// the lock ITSELF" holds for G, and F's call to G is not a helper call at all. This pass propagates
// the held lock ACROSS the call edge instead: every body is replayed under an ENTRY state of "the
// caller holds it", and whatever that reaches is asked the same question.
//
// IT WAS NOT BUILT ON SUSPICION. Its first run over this tree found the deadlock inside
// `Lobby.initiatePairedMatch`, reached from `Lobby.processMatchmaking`'s write lock — the matchmaking
// pairing path — which therefore froze the whole process on EVERY successful pairing. The calls were
// the self-locking `sendToClient`, and the fix uses its `...Locked` sibling. That is what a blind
// spot costs, and it is why this pass is committed rather than merely measured once.
//
// HONEST LIMITS, stated rather than implied:
//   - Parsing carries no type information, so a call through a service the lobby HOLDS
//     (`l.auctionService.ProcessAuctions(l)`) is resolved only when that method name is UNIQUE in the
//     package; an ambiguous name is left unresolved rather than guessed at.
//   - A method on another type that takes a `*Lobby` parameter IS analysed, but it is only REACHED
//     when a body calls it and that call resolves under the rule above.
//   - The walk's own limits (a `break`/`goto` carrying a freshly taken lock, an untyped local, a
//     function literal's own scope) apply here exactly as they do above, where they are stated.
//   - Paths are JOINED, not tracked: a body reached under the lock on ONE path only is still
//     reported, so every finding names its WITNESS — the single edge that supplies the lock — for
//     the reader to check instead of re-deriving the chain.
// ============================================================================

// transitiveSelfLockAudit owns the fixpoint over call edges.
type transitiveSelfLockAudit struct {
	tree    *lobbyTree
	keys    []string
	cache   map[string]map[heldKind][]selfLockSite
	entries map[string]map[heldKind]bool
	witness map[string]string
	edges   int
}

func newTransitiveSelfLockAudit(tree *lobbyTree) *transitiveSelfLockAudit {
	x := &transitiveSelfLockAudit{
		tree: tree, cache: map[string]map[heldKind][]selfLockSite{},
		entries: map[string]map[heldKind]bool{}, witness: map[string]string{},
	}
	for key := range tree.decls {
		x.keys = append(x.keys, key)
	}
	sort.Strings(x.keys)
	return x
}

// sites replays ONE body under ONE entry state, caching the result. The walk is deterministic, so a
// site's INDEX is stable and can be compared with the same body's intra-procedural baseline — which
// is how a finding proves the lock arrived across an edge rather than being taken inside the body.
func (x *transitiveSelfLockAudit) sites(key string, entry heldKind) []selfLockSite {
	if byEntry, ok := x.cache[key]; ok {
		if got, ok := byEntry[entry]; ok {
			return got
		}
	}
	fn := x.tree.decls[key]
	if fn == nil {
		return nil
	}
	var out []selfLockSite
	a := &lockAuditor{
		fset: x.tree.fset, filename: x.tree.files[key], funcName: funcLabel(fn),
		lobbyNames: lobbyTypedIdents(fn), registry: x.tree.helpers,
		methods: x.tree.methods, funcs: x.tree.funcs, byName: x.tree.byName,
		sites: &out,
	}
	a.block(fn.Body.List, entry)
	if x.cache[key] == nil {
		x.cache[key] = map[heldKind][]selfLockSite{}
	}
	x.cache[key][entry] = out
	return out
}

// propagate is the fixpoint: which bodies can be ENTERED while the lobby lock is definitely held.
// A body is entered under the lock when some call site THAT HOLDS IT names that body as the callee.
// Entries grow monotonically and are bounded by (bodies × 2 kinds), so this terminates.
func (x *transitiveSelfLockAudit) propagate() {
	add := func(key string, k heldKind, w string) bool {
		if k == heldNone {
			return false
		}
		if x.entries[key] == nil {
			x.entries[key] = map[heldKind]bool{}
		}
		if x.entries[key][k] {
			return false
		}
		x.entries[key][k] = true
		if _, seen := x.witness[key]; !seen {
			x.witness[key] = w
		}
		return true
	}
	for changed := true; changed; {
		changed = false
		for _, key := range x.keys {
			states := []heldKind{heldNone}
			for k := range x.entries[key] {
				states = append(states, k)
			}
			for _, entry := range states {
				for _, s := range x.sites(key, entry) {
					if s.callee == "" || s.held == heldNone {
						continue
					}
					if _, known := x.tree.decls[s.callee]; !known {
						continue // a call the resolver could not place: skipped, never guessed
					}
					x.edges++
					w := fmt.Sprintf("%s:%d  %s() calls %s() while holding %s",
						x.tree.files[key], s.line, key, s.callee, s.held)
					if add(s.callee, s.held, w) {
						changed = true
					}
				}
			}
		}
	}
}

// transitiveSelfLockReport is what the pass found. ONE bucket, because every shape it reports is a
// deadlock: the lock is held and the callee takes it again on a non-re-entrant sync.RWMutex.
type transitiveSelfLockReport struct {
	violations []string
	stats      string
}

func (x *transitiveSelfLockAudit) report() transitiveSelfLockReport {
	var out transitiveSelfLockReport
	var violations []string
	intra, legit, reachable := 0, 0, 0
	seen := map[string]bool{}
	for _, key := range x.keys {
		base := x.sites(key, heldNone)
		for _, s := range base {
			if s.helper == "" {
				continue
			}
			if s.held == heldNone {
				legit++
				continue
			}
			intra++ // intra-procedural: the gate above already names it, and one owner per finding
		}
		var kinds []heldKind
		for k := range x.entries[key] {
			kinds = append(kinds, k)
		}
		sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })
		if len(kinds) > 0 {
			reachable++
		}
		for _, entry := range kinds {
			for i, s := range x.sites(key, entry) {
				if s.helper == "" || s.held == heldNone {
					continue
				}
				if i < len(base) && base[i].held != heldNone {
					continue // the body takes the lock itself: not this pass's finding
				}
				id := fmt.Sprintf("%s:%d:%s", key, s.line, s.helper)
				if seen[id] {
					continue
				}
				seen[id] = true
				h := x.tree.helpers[s.helper]
				violations = append(violations, fmt.Sprintf(
					"%s:%d: %s() is entered with the lobby lock held (%s) and calls %s() with %s still held — %s. Witness: %s",
					x.tree.files[key], s.line, key, entry, s.helper, s.held, h.reasonAt(s.held), x.witness[key]))
			}
		}
	}
	sort.Strings(violations)
	out.violations = violations
	out.stats = fmt.Sprintf(
		"transitive pass: %d bodies, %d self-locking helpers, %d edges propagated with the lock held, %d bodies enterable under the lock, %d findings reached with the lock across an edge, %d intra-procedural (owned by the gate above), %d correct calls (no lock held). Calls the resolver cannot place are simply not edges and are counted NOWHERE, because a field that always answers 0 is a false statement.",
		len(x.keys), len(x.tree.helpers), x.edges, reachable, len(violations), intra, legit)
	return out
}

// syntheticLobbyTree parses ONE source into a lobbyTree so the transitive pass can be driven against
// shapes chosen on purpose. Without it, the pass would be an untested assertion about itself.
func syntheticLobbyTree(t *testing.T, name, src string) *lobbyTree {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	tree := &lobbyTree{
		fset: fset, helpers: map[string]*lobbyHelper{},
		parsed: map[string]*ast.File{name: file}, scanned: []string{name},
	}
	indexLobbyTree(tree)
	return tree
}

// TestNoTransitiveSelfLockingLobbyHelper is the SECOND gate: no body may be ENTERED with the lobby
// lock held and reach a self-locking lobby helper, however many calls away that helper is.
func TestNoTransitiveSelfLockingLobbyHelper(t *testing.T) {
	tree := loadLobbyTree(t)
	if len(tree.scanned) < 50 {
		t.Errorf("scanned only %d non-test Go files; this pass is not looking at the tree it claims to", len(tree.scanned))
	}
	if len(tree.helpers) < 100 {
		t.Errorf("derived only %d self-locking helpers — the propagation would run over a registry that is not the gate's", len(tree.helpers))
	}

	x := newTransitiveSelfLockAudit(tree)
	x.propagate()
	rep := x.report()

	// A fixpoint over zero edges reports a clean tree for ever. Never let that read as a pass.
	if x.edges == 0 {
		t.Errorf("the transitive pass propagated ZERO call edges — it cannot have measured anything")
	}
	for _, v := range rep.violations {
		t.Errorf("TRANSITIVE SELF-LOCK: %s", v)
	}
	t.Log(rep.stats)
}

// TestTransitiveSelfLockDetectsAViolation is the pass's own control. It carries the shapes that MUST
// be reported (a method hop, a free-function hop, a hop through a service the lobby holds) and the
// shapes that MUST NOT (a correct caller, a call made after the lock is released, a `go` statement,
// an ambiguous held-service name, and the intra-procedural case that belongs to the gate above).
func TestTransitiveSelfLockDetectsAViolation(t *testing.T) {
	const src = `package main

type Service struct{}
type Other struct{}

func (l *Lobby) helperH() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
}

func (l *Lobby) helperHLocked() {}

func (l *Lobby) innerMid() {
	l.helperH()
}

func freeMid(l *Lobby) {
	l.helperH()
}

func (l *Lobby) directHolder() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.helperH()
}

func (l *Lobby) transitiveViaMethod() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.innerMid()
}

func (l *Lobby) transitiveViaFreeFunction() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	freeMid(l)
}

func (s *Service) Serve(l *Lobby) {
	l.helperH()
}

func (l *Lobby) transitiveViaHeldService() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.svc.Serve(l)
}

func (l *Lobby) safeCaller() {
	l.helperH()
}

func (l *Lobby) releasedBeforeCall() {
	l.mutex.Lock()
	l.mutex.Unlock()
	l.helperH()
}

func (l *Lobby) goUnderLock() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	go l.helperH()
}

func (s *Service) Dup(l *Lobby) { l.helperH() }
func (o *Other) Dup(l *Lobby)   { l.helperH() }

func (l *Lobby) ambiguousHeldService() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.svc.Dup(l)
}

// A RECURSIVE READ lock reached across an EDGE (the whole point of the transitive pass): a read lock
// is taken by transitiveReadViaMethod, the mid function takes no lock of its own, and the helper it
// reaches only wants a read lock. All three layers are needed for the pass to see it.
func (l *Lobby) readHelperH() {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
}

func (l *Lobby) readHelperHLocked() {}

func (l *Lobby) innerReadMid() {
	l.readHelperH()
}

func (l *Lobby) transitiveReadViaMethod() {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	l.innerReadMid()
}
`
	tree := syntheticLobbyTree(t, "transitive_probe.go", src)
	if tree.helpers["helperH"] == nil {
		t.Fatalf("the registry did not see helperH() take the lock, so this control would pass vacuously")
	}
	x := newTransitiveSelfLockAudit(tree)
	x.propagate()
	report := strings.Join(x.report().violations, "\n")

	for _, want := range []string{"Lobby.innerMid()", "freeMid()", "Service.Serve()", "Lobby.innerReadMid()"} {
		if !strings.Contains(report, want) {
			t.Errorf("the transitive pass did NOT report %s, which is entered with the lock held and reaches a self-locking helper — the blind spot is still blind:\n%s", want, report)
		}
	}

	mustNotReport := []string{
		"Lobby.safeCaller()",           // calls the helper with no lock held: the correct form
		"Lobby.releasedBeforeCall()",   // the lock is released before the call
		"Lobby.goUnderLock()",          // a goroutine does not inherit the lock
		"Lobby.ambiguousHeldService()", // Dup is declared twice, so the name is NOT guessed at
		"Lobby.directHolder()",         // intra-procedural: owned by the gate above
	}
	for _, mustNot := range mustNotReport {
		if strings.Contains(report, mustNot) {
			t.Errorf("the transitive pass reported %s, which must stay out of its scope — a pass that reports correct code, or a sibling gate's case, gets switched off:\n%s", mustNot, report)
		}
	}

	// The finding must NAME the fix, exactly as the gate above does, or the first reader has to go
	// and find the sibling themselves ... and it must name its WITNESS, the edge that supplies the lock.
	if !strings.Contains(report, "Locked") {
		t.Errorf("a violation must name the `...Locked` form to call instead:\n%s", report)
	}
	if !strings.Contains(report, "Witness:") {
		t.Errorf("a transitive finding must name the edge that supplies the lock:\n%s", report)
	}

	// It must also be able to report NOTHING: a pass that always fires is as useless as one that
	// never does, and the clean case is what the scan of the whole tree asserts about this package.
	clean := syntheticLobbyTree(t, "clean.go", "package main\n\nfunc (l *Lobby) ok() { l.helperH() }\n")
	cx := newTransitiveSelfLockAudit(clean)
	cx.propagate()
	if got := cx.report().violations; len(got) != 0 {
		t.Errorf("a clean tree reported %d transitive violations: %v", len(got), got)
	}
}

// TestTheMatchmakingPairingPathCompletesUnderTheWriteLock is the BEHAVIOURAL half of the transitive
// gate, pinned to the one site that pass found. `Lobby.processMatchmaking` holds the WRITE lock for
// its whole body and calls `Lobby.initiatePairedMatch`, which sent through the SELF-LOCKING
// `sendToClient` — an unconditional freeze, because an `sync.RWMutex` is not re-entrant and the write
// lock is never released. It fails by TIMEOUT rather than by assertion, because a self-lock raises no
// error at all: it simply never returns.
func TestTheMatchmakingPairingPathCompletesUnderTheWriteLock(t *testing.T) {
	l := &Lobby{
		// REAL, BUFFERED client channels: the pairing path also pushes the challenge envelopes with a
		// plain `c.send <- msg`, so a nil channel would hang this test for a fixture reason rather
		// than measure the lock discipline it exists to measure.
		clients: map[string]*Client{
			"c1": {send: make(chan []byte, 4)},
			"c2": {send: make(chan []byte, 4)},
		},
		matches:           map[string]*MatchState{"c1": {}, "c2": {}},
		wallets:           map[string]string{"c1": "0xp1", "c2": "0xp2"},
		leaderboard:       map[string]PlayerStats{"0xp1": {}, "0xp2": {}},
		matchHandshakers:  map[string]*SyncHandshaker{},
		lastSeenDistricts: map[string]string{},
		lastActive:        map[string]time.Time{},
		playerBalances:    map[string]uint64{},
	}

	runWithWatchdog(t, "the matchmaking pairing path under the write lock", 5*time.Second, func() {
		l.mutex.Lock()
		defer l.mutex.Unlock()
		if !l.initiatePairedMatch("c1", "c2", "") {
			t.Error("a pairing of two seeded clients reported failure")
		}
		if l.matches["c1"] == nil || l.matches["c1"] != l.matches["c2"] {
			t.Error("both clients must end up in the SAME match state")
		}
	})
}

func TestTheRecommendedLockedFormsRunWhileTheWriteLockIsHeld(t *testing.T) {
	l := &Lobby{clients: map[string]*Client{}, onboardedWallets: map[string]bool{}}

	runWithWatchdog(t, "the `...Locked` forms under the write lock", 5*time.Second, func() {
		l.mutex.Lock()
		defer l.mutex.Unlock()
		// Each of these is the lock-held form. Called under the WRITE lock, the self-locking form of
		// either one would take the lobby lock again and never return.
		l.broadcastToAdminsLocked("self-lock gate probe")
		if l.isWalletRegisteredLocked("0xprobe") {
			t.Error("an unregistered probe wallet reported as registered")
		}
	})
}

// TestTheCareerProgressDoorCompletesAndWritesUnderTheWriteLock is the BEHAVIOURAL half of the THIRD
// fixed site. `HandleGetCareerProgress` used to hold a READ lock while it (a) wrote `l.leaderboard` —
// and that map holds VALUES, so the write-back was a concurrent map write against every other reader
// — and then (b) called the self-locking `GetCareerProgress`. The fix takes the WRITE lock once,
// builds the payload with the lock-held `careerProgressLocked`, and releases before it encodes.
//
// Both halves are asserted: the door answers, and the record it lazily initialised IS the one that
// was served (a write that landed on a copy would leave the stored record untouched).
func TestTheCareerProgressDoorCompletesAndWritesUnderTheWriteLock(t *testing.T) {
	l := &Lobby{leaderboard: map[string]PlayerStats{
		"0xcareer": {CareerLevel: map[string]int{"Judge": 2}},
	}}

	runWithWatchdog(t, "the career-progress door", 5*time.Second, func() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/career/progress?wallet=0xcareer", nil)
		l.HandleGetCareerProgress(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("the career-progress door answered %d: %s", rec.Code, rec.Body.String())
			return
		}
		if !strings.Contains(rec.Body.String(), `"career_level"`) {
			t.Errorf("the served payload must carry the career level: %s", rec.Body.String())
		}
		if l.leaderboard["0xcareer"].CareerXP == nil {
			t.Error("the door must have initialised CareerXP on the STORED record, not on a copy")
		}
	})

	// The lock-held builder is usable while the WRITE lock is held — the contract the fix relies on,
	// and the one the gate now enforces. A self-locking callee here would never return.
	runWithWatchdog(t, "careerProgressLocked under the write lock", 5*time.Second, func() {
		l.mutex.Lock()
		defer l.mutex.Unlock()
		if got := careerProgressLocked("0xcareer", l.leaderboard["0xcareer"]); got["career_level"] == nil {
			t.Error("the lock-held builder must carry the career level")
		}
	})
}
