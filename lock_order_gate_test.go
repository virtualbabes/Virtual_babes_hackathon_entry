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
// THE CROSS-MUTEX LOCK-ORDER GATE — no pair of mutexes may be acquired in both
// orders (A->B and B->A anywhere in the tree).
// ============================================================================
//
// WHY THIS EXISTS. Every other lock gate in this tree models ONE mutex: `selflock_gate_test.go` asks
// whether a function calls a helper that takes the LOCK IT ALREADY HOLDS, and
// `mutex_relock_gate_test.go` asks whether a body acquires a mutex it already holds. Both are blind
// to the deadlock that needs TWO locks, and this tree has nine gates and (until now) zero covering it.
//
// THE SHAPE. `sync.Mutex`/`sync.RWMutex` are not re-entrant, so two goroutines that take A then B and
// B then A block FOR EVER. It is worse than the single-mutex class in one specific way: a QUEUED
// WRITER on an `RWMutex` blocks every later reader, so a reader-reader pair — which looks harmless in
// isolation — is fatal as soon as any third goroutine queues a write. Request paths in this tree hold
// the lobby WRITE lock across their bodies, so a queued lobby writer is the NORMAL state, not a corner
// case. That is the same reasoning that turned `mutex_relock_gate_test.go`'s read+read "notes" into
// failures, applied to a second mutex.
//
// MEASURED ON ITS FIRST RUN: exactly TWO inversions, both live.
//   - `AICitizenEngine.mu` <-> `Lobby.mutex`: the behavioural tick held `ace.mu` for its whole body and
//     took the lobby lock, while `runAutonomousTournament` (a 15-minute daemon) and the Zen-Garden
//     doors held the lobby lock and reached into the engine.
//   - `Lobby.mutex` <-> `TokenSinkRouter.Mu`: the persistence worker held the router lock and took the
//     lobby lock, while 34 fee-routing sites take the lobby lock and then `RouteCriminalTax` (which
//     takes the router lock for WRITING — the side that decides the deadlock).
//
// Both are fixed; the baseline is EMPTY, and it stays that way by measurement rather than by trust.
//
// WHAT IT MEASURES, in one line: build the directed graph of ORDER edges (holding X, then acquiring Y)
// over the whole tree, and report every pair that appears in both directions.
//
// THE SUBJECT IS DERIVED, NOT CURATED: struct fields are read from the declarations, and each mutex
// identity is canonicalised to "<StructType>.<Field>" THROUGH THOSE DECLARATIONS. Canonicalisation is
// not a nicety — without it `ace.mu` and `l.aiEngine.mu` are two unrelated strings and the whole
// analysis is meaningless. If the derivation finds ZERO mutex fields the gate FAILS, because a
// detector that matches nothing reports a clean repository for ever.
//
// THE ANALYSIS IS MUST-HELD AND ONE CALL EDGE DEEP, and its limits are stated rather than implied:
//   - MUST-HELD. `if`/`else`, `switch`/`select` clauses and loops are JOINED; a loop may run zero
//     times, so it can only carry out what was already held.
//   - A `defer`red RELEASE does not clear the held state at its source line (it happens at exit).
//   - A `go` literal is a NEW GOROUTINE and starts from an EMPTY held set; a `defer`red CALL is not
//     analysed as an edge at all (LIFO — it runs after the body).
//   - ONE CALL EDGE. A callee's acquired receiver fields are resolved against the CALL-SITE receiver
//     expression, so `l.aiEngine.GetAllCitizens()` becomes `AICitizenEngine.mu` only because
//     `Lobby.aiEngine` is declared `*AICitizenEngine`. A method name declared by two receivers with
//     DIFFERENT takes is SKIPPED, never guessed; a method this pass has no take for is not an edge.
//   - Anything whose type cannot be resolved from a receiver/param/local declaration WITHOUT GUESSING
//     is SKIPPED AND COUNTED. Today that census is nonzero, and the number is printed on every run so
//     the blind spot is visible instead of assumed away.
//   - It is an ORDER graph, not a proof of reachability: a reported pair names two ORDER edges and
//     their source positions, which is what a reader needs to adjudicate it.
//
// loAnalysis holds one full derivation + walk over a set of sources. It is a VALUE rather than
// package state so the real tree and the synthetic control cannot contaminate each other.
type loAnalysis struct {
	fieldTypes  map[string]map[string]string // struct type -> field -> type string
	mutexNames  map[string]bool              // field NAMES whose type is a sync mutex
	methodTakes map[string][]string          // method name -> fields acquired through its own receiver
	ambiguous   map[string]bool              // method names declared by 2+ receivers with different takes
	edges       []loEdge
	skipped     int
	files       int
	structs     int
	seeded      int                         // `...Locked` callees walked under their caller's held set
	seeds       map[string][]map[string]int // callee name -> the held sets seen at its call sites

	// THE ALIASED-MAP SUBJECT, measured by `market_nodes_alias_gate_test.go`: every access to a field
	// that an assignment in this tree has PROVEN to be a second name for another field's map, with the
	// mutexes definitely held at that access — plus the evidence (the alias assignment sites themselves).
	accesses    []loAccess
	aliasFields map[string]bool
	aliasPairs  []string
}

// loDeclRef is enough to walk a function a second time, under a held set supplied by a CALL SITE.
type loDeclRef struct {
	fset *token.FileSet
	file string
	fn   *ast.FuncDecl
}

// loVarsFor resolves a function's receiver and parameter NAMES to type strings.
func loVarsFor(fn *ast.FuncDecl) map[string]string {
	vars := map[string]string{}
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		f0 := fn.Recv.List[0]
		if len(f0.Names) > 0 {
			vars[f0.Names[0].Name] = loTypeString(f0.Type)
		}
	}
	if fn.Type.Params != nil {
		for _, p := range fn.Type.Params.List {
			for _, n := range p.Names {
				vars[n.Name] = loTypeString(p.Type)
			}
		}
	}
	return vars
}

type loEdge struct {
	from, to string
	where    string
}

func loSyncMutexType(e ast.Expr) bool {
	for {
		switch x := e.(type) {
		case *ast.StarExpr:
			e = x.X
		case *ast.ParenExpr:
			e = x.X
		case *ast.SelectorExpr:
			b, ok := x.X.(*ast.Ident)
			return ok && b.Name == "sync" && (x.Sel.Name == "Mutex" || x.Sel.Name == "RWMutex")
		default:
			return false
		}
	}
}

// loTypeString renders a type expression; "" means NOT rendered, and it is never guessed at.
func loTypeString(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.StarExpr:
		return "*" + loTypeString(x.X)
	case *ast.SelectorExpr:
		return loTypeString(x.X) + "." + x.Sel.Name
	case *ast.ParenExpr:
		return loTypeString(x.X)
	case *ast.ArrayType:
		return "[]" + loTypeString(x.Elt)
	case *ast.MapType:
		return "map[" + loTypeString(x.Key) + "]" + loTypeString(x.Value)
	case *ast.ChanType:
		return "chan " + loTypeString(x.Value)
	case *ast.FuncType:
		return "func"
	case *ast.InterfaceType:
		return "interface{}"
	}
	return ""
}

func loBaseType(t string) string { return strings.TrimPrefix(strings.TrimPrefix(t, "*"), "*") }

// loPathOf renders `a.b.c` into root `a` + ["b","c"]; anything else is not rendered.
func loPathOf(e ast.Expr) (string, []string, bool) {
	var parts []string
	cur := e
	for {
		switch x := cur.(type) {
		case *ast.ParenExpr:
			cur = x.X
		case *ast.SelectorExpr:
			parts = append([]string{x.Sel.Name}, parts...)
			cur = x.X
		case *ast.Ident:
			if len(parts) == 0 {
				return "", nil, false
			}
			return x.Name, parts, true
		default:
			return "", nil, false
		}
	}
}

// loDeriveFields reads every struct field of every declaration, keeping the FIELD TYPES (which is what
// makes canonicalisation possible) and the mutex-typed NAMES (which is what makes a lock operation
// recognisable). Done first, because both later passes depend on it.
func loDeriveFields(a *loAnalysis, sources []mutexSource) error {
	for _, s := range sources {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, s.name, s.src, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", s.name, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok || st.Fields == nil {
				return true
			}
			a.structs++
			m := a.fieldTypes[ts.Name.Name]
			if m == nil {
				m = map[string]string{}
				a.fieldTypes[ts.Name.Name] = m
			}
			for _, fl := range st.Fields.List {
				t := loTypeString(fl.Type)
				if loSyncMutexType(fl.Type) {
					for _, nm := range fl.Names {
						a.mutexNames[nm.Name] = true
					}
				}
				if len(fl.Names) == 0 {
					m[loBaseType(t)] = t // embedded field: its name is its base type
					continue
				}
				for _, nm := range fl.Names {
					m[nm.Name] = t
				}
			}
			return true
		})
	}
	return nil
}

// loDeriveMethodTakes builds the ONE-CALL-EDGE registry from the tree: every method whose body takes a
// mutex field THROUGH ITS OWN RECEIVER. A name declared by two receivers with different takes is
// marked ambiguous and skipped, never guessed.
func loDeriveMethodTakes(a *loAnalysis, sources []mutexSource) error {
	for _, s := range sources {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, s.name, s.src, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", s.name, err)
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Recv == nil || len(fn.Recv.List) == 0 {
				continue
			}
			recvType := loBaseType(loTypeString(fn.Recv.List[0].Type))
			names := fn.Recv.List[0].Names
			if len(names) == 0 {
				continue
			}
			recv := names[0].Name
			takes := map[string]bool{}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				cc, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				se, ok := cc.Fun.(*ast.SelectorExpr)
				if !ok || (se.Sel.Name != "Lock" && se.Sel.Name != "RLock") {
					return true
				}
				inner, ok := se.X.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				id, ok := inner.X.(*ast.Ident)
				if !ok || id.Name != recv {
					return true
				}
				ft := a.fieldTypes[recvType][inner.Sel.Name]
				if ft == "sync.Mutex" || ft == "sync.RWMutex" {
					takes[inner.Sel.Name] = true
				}
				return true
			})
			var fields []string
			for k := range takes {
				fields = append(fields, k)
			}
			sort.Strings(fields)
			if len(fields) == 0 {
				continue
			}
			name := fn.Name.Name
			if prev, seen := a.methodTakes[name]; seen {
				if strings.Join(prev, ",") != strings.Join(fields, ",") {
					a.ambiguous[name] = true
				}
				continue
			}
			a.methodTakes[name] = fields
		}
	}
	return nil
}

// loWalker walks ONE function body threading the set of DEFINITELY-held mutex identities and recording
// every ORDER edge it implies. Held state is shared verbatim with the sibling gate (a `defer`red
// release does not clear at its source; branches join; a `go` literal starts empty).
type loWalker struct {
	a    *loAnalysis
	fset *token.FileSet
	file string
	fn   string // the enclosing function, so a recorded access can NAME where it lives
	vars map[string]string

	// standalone marks the TOP-LEVEL pass. A `...Locked` body takes no lock of its own BECAUSE its
	// caller holds one, so measuring it from an empty held set reports every line in it; it is measured
	// from its CALL SITES instead (the seeded pass), which is the sibling gate's own convention.
	standalone bool
}

func (w *loWalker) line(n ast.Node) int { return w.fset.Position(n.Pos()).Line }

// loObjType resolves an object expression to its STRUCT TYPE by walking the declarations.
func (w *loWalker) loObjType(e ast.Expr) (string, bool) {
	root, path, ok := loPathOf(e)
	if !ok {
		return "", false
	}
	cur, ok := w.vars[root]
	if !ok {
		return "", false
	}
	cur = loBaseType(cur)
	for _, f := range path {
		ft, ok := w.a.fieldTypes[cur][f]
		if !ok {
			return "", false
		}
		cur = loBaseType(ft)
	}
	return cur, true
}

// loCanonMutex canonicalises a mutex expression to "<StructType>.<Field>". This is the step that makes
// `ace.mu` and `l.aiEngine.mu` ONE identity instead of two unrelated strings.
func (w *loWalker) loCanonMutex(e ast.Expr) (string, bool) {
	key, ok := w.loCanonField(e)
	if !ok {
		return "", false
	}
	// `key` is "<Type>.<Field>" by construction, so the last segment names the field to type-check.
	if i := strings.LastIndex(key, "."); i > 0 {
		if ft := w.a.fieldTypes[key[:i]][key[i+1:]]; ft == "sync.Mutex" || ft == "sync.RWMutex" {
			return key, true
		}
	}
	return "", false
}

// loCanonField canonicalises ANY field expression to "<StructType>.<Field>" — the resolution
// `loCanonMutex` needs, without requiring the field to be a mutex. It is what lets the aliased-map gate
// tell `l.marketNodes` (Lobby) from `l.tokenSinkRouter.MarketNodes` (TokenSinkRouter) from
// `snapshot.MarketNodes` (a LOCAL RouterSnapshot copy, which must NOT be reported).
func (w *loWalker) loCanonField(e ast.Expr) (string, bool) {
	return loCanonFieldOf(w.a, w.vars, e)
}

// loCanonFieldOf is the free form, resolving against SUPPLIED vars — which is what the alias derivation
// needs, because it runs before any walk exists.
func loCanonFieldOf(a *loAnalysis, vars map[string]string, e ast.Expr) (string, bool) {
	root, path, ok := loPathOf(e)
	if !ok || len(path) == 0 {
		return "", false
	}
	cur, ok := vars[root]
	if !ok {
		return "", false
	}
	cur = loBaseType(cur)
	for i, f := range path {
		ft, ok := a.fieldTypes[cur][f]
		if !ok {
			return "", false
		}
		if i == len(path)-1 {
			return cur + "." + f, true
		}
		cur = loBaseType(ft)
	}
	return "", false
}

func (w *loWalker) note(from, to string, n ast.Node) {
	if from == "" || to == "" || from == to {
		return
	}
	w.a.edges = append(w.a.edges, loEdge{from: from, to: to,
		where: fmt.Sprintf("%s:%d", w.file, w.line(n))})
}

// order records the ORDER edges implied by holding `held` and then acquiring `key`.
func (w *loWalker) order(held map[string]int, key string, n ast.Node) {
	for h := range held {
		w.note(h, key, n)
	}
}

func (w *loWalker) block(list []ast.Stmt, in map[string]int) map[string]int {
	cur := cloneHeld(in)
	for _, s := range list {
		cur = w.stmt(s, cur)
	}
	return cur
}

func (w *loWalker) stmt(s ast.Stmt, in map[string]int) map[string]int {
	switch x := s.(type) {
	case nil:
		return in
	case *ast.BlockStmt:
		return w.block(x.List, in)
	case *ast.ExprStmt:
		if c, ok := x.X.(*ast.CallExpr); ok {
			return w.call(c, in, false)
		}
		w.walkExpr(x.X, in)
		return in
	case *ast.DeferStmt:
		// A deferred CALL runs after the body (LIFO), so it is not an order edge here; a deferred
		// RELEASE likewise does not clear the held state at this line.
		return w.call(x.Call, in, true)
	case *ast.GoStmt:
		// A new goroutine does NOT inherit the locks: a literal under `go` starts from EMPTY.
		if lit, ok := x.Call.Fun.(*ast.FuncLit); ok {
			_ = w.block(lit.Body.List, map[string]int{})
			return in
		}
		return w.call(x.Call, in, false)
	case *ast.AssignStmt:
		for _, r := range x.Rhs {
			w.walkExpr(r, in)
			if lit, ok := r.(*ast.FuncLit); ok {
				_ = w.block(lit.Body.List, cloneHeld(in))
			}
		}
		// The LEFT side is measured too: it can WRITE the map (`l.marketNodes[id] = …`) or rebind the
		// field (`l.marketNodes = …`), and an unfenced write is the very defect this subject exists for.
		for _, l := range x.Lhs {
			switch lt := l.(type) {
			case *ast.SelectorExpr:
				w.recordAliasAccess(lt, in)
			case *ast.IndexExpr:
				if se, ok := lt.X.(*ast.SelectorExpr); ok {
					w.recordAliasAccess(se, in)
				}
			}
		}
		if len(x.Lhs) == len(x.Rhs) {
			for i, l := range x.Lhs {
				id, ok := l.(*ast.Ident)
				if !ok {
					continue
				}
				if t := loTypeString(x.Rhs[i]); t != "" {
					w.vars[id.Name] = t
				}
			}
		}
		return in
	case *ast.DeclStmt:
		if gd, ok := x.Decl.(*ast.GenDecl); ok {
			for _, sp := range gd.Specs {
				if vs, ok := sp.(*ast.ValueSpec); ok {
					t := loTypeString(vs.Type)
					for _, n := range vs.Names {
						if t != "" {
							w.vars[n.Name] = t
						}
					}
				}
			}
		}
		return in
	case *ast.ReturnStmt:
		for _, r := range x.Results {
			w.walkExpr(r, in)
		}
		return in
	case *ast.IfStmt:
		if x.Init != nil {
			in = w.stmt(x.Init, in)
		}
		w.walkExpr(x.Cond, in)
		a := w.block(x.Body.List, in)
		b := in
		if x.Else != nil {
			b = w.stmt(x.Else, in)
		}
		return intersectHeld(a, b)
	case *ast.ForStmt:
		if x.Init != nil {
			in = w.stmt(x.Init, in)
		}
		w.walkExpr(x.Cond, in)
		body := w.block(x.Body.List, in)
		if x.Post != nil {
			body = w.stmt(x.Post, body)
		}
		return intersectHeld(in, body)
	case *ast.RangeStmt:
		w.walkExpr(x.X, in)
		return intersectHeld(in, w.block(x.Body.List, in))
	case *ast.SwitchStmt:
		if x.Init != nil {
			in = w.stmt(x.Init, in)
		}
		w.walkExpr(x.Tag, in)
		return w.clauses(x.Body.List, in)
	case *ast.TypeSwitchStmt:
		if x.Init != nil {
			in = w.stmt(x.Init, in)
		}
		if x.Assign != nil {
			in = w.stmt(x.Assign, in)
		}
		return w.clauses(x.Body.List, in)
	case *ast.SelectStmt:
		return w.clauses(x.Body.List, in)
	case *ast.LabeledStmt:
		return w.stmt(x.Stmt, in)
	}
	return in
}

// clauses joins every case clause (MUST-held).
func (w *loWalker) clauses(list []ast.Stmt, in map[string]int) map[string]int {
	out := map[string]int{}
	first := true
	for _, c := range list {
		var body []ast.Stmt
		switch cc := c.(type) {
		case *ast.CaseClause:
			body = cc.Body
		case *ast.CommClause:
			body = cc.Body
		default:
			continue
		}
		cur := w.block(body, in)
		if first {
			out = cur
			first = false
		} else {
			out = intersectHeld(out, cur)
		}
	}
	if first {
		return in
	}
	return out
}

// recordAliasAccess records ONE access to an ALIASED map field, with the mutexes definitely held there.
// It records nothing unless the field has been PROVEN (by an assignment in the tree) to be a second name
// for another field's map, so the gate's subject is DERIVED rather than typed out by hand — and a field
// that is not part of an alias pair (`snapshot.MarketNodes`, a local RouterSnapshot copy) is invisible
// to it by construction.
func (w *loWalker) recordAliasAccess(e ast.Expr, in map[string]int) {
	if len(w.a.aliasFields) == 0 {
		return
	}
	// A `...Locked` body takes no lock of its own BECAUSE its caller holds one, so an access inside it is
	// measured from its CALL SITES (the seeded pass), never from an empty held set — the same convention
	// the sibling gate's one-call-edge pass uses.
	if w.standalone && strings.HasSuffix(w.fn, "Locked") {
		return
	}
	key, ok := w.loCanonField(e)
	if !ok || !w.a.aliasFields[key] {
		return
	}
	held := make([]string, 0, len(in))
	for k := range in {
		held = append(held, k)
	}
	sort.Strings(held)
	w.a.accesses = append(w.a.accesses, loAccess{
		Key:     key,
		Where:   fmt.Sprintf("%s:%d", w.file, w.line(e)),
		Func:    w.fn,
		HeldSet: held,
	})
}

func (w *loWalker) walkExpr(e ast.Expr, in map[string]int) {
	if e == nil {
		return // `for {}` has no Cond, and a bare `return` has no Results: nil is not a shape
	}
	ast.Inspect(e, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.FuncLit:
			_ = w.block(t.Body.List, cloneHeld(in))
			return false
		case *ast.SelectorExpr:
			// A map access through either spelling of an aliased field, measured with the held set here.
			w.recordAliasAccess(t, in)
			return true
		case *ast.CallExpr:
			_ = w.call(t, in, false)
			return false
		}
		return true
	})
}

// call handles one call site: lock operations (intra-procedural order) and callee acquisitions (the
// ONE call edge), recording an edge for every mutex held at that point.
func (w *loWalker) call(c *ast.CallExpr, in map[string]int, deferred bool) map[string]int {
	sel, isSel := c.Fun.(*ast.SelectorExpr)
	if isSel {
		switch sel.Sel.Name {
		case "Lock", "RLock":
			if key, ok := w.loCanonMutex(sel.X); ok {
				if !deferred {
					w.order(in, key, c)
					if _, held := in[key]; !held {
						in[key] = w.line(c)
					}
				}
			} else {
				w.a.skipped++
			}
		case "Unlock", "RUnlock":
			if key, ok := w.loCanonMutex(sel.X); ok {
				if !deferred {
					delete(in, key)
				}
			} else {
				w.a.skipped++
			}
		}
		// THE ONE CALL EDGE: resolve the callee's acquired receiver fields against the CALL-SITE
		// receiver expression. An ambiguous name is skipped, never guessed.
		if fields, ok := w.a.methodTakes[sel.Sel.Name]; ok && !w.a.ambiguous[sel.Sel.Name] && len(in) > 0 {
			if recvType, ok := w.loObjType(sel.X); ok {
				for _, f := range fields {
					w.order(in, recvType+"."+f, c)
				}
			} else {
				w.a.skipped++
			}
		}
		// ONE-HOP SEED: a `...Locked` callee takes no lock of its own BECAUSE its caller already holds
		// one — the tree's own naming contract. Record the caller's held set so it can be walked under
		// it below; without this, MOVING a locked block into such a helper (exactly what the two fixes
		// in §34 did) would make the shape invisible and the gate would silently stop measuring it.
		if strings.HasSuffix(sel.Sel.Name, "Locked") && len(in) > 0 {
			w.a.seeds[sel.Sel.Name] = append(w.a.seeds[sel.Sel.Name], cloneHeld(in))
		}
	}
	for _, a := range c.Args {
		w.walkExpr(a, in)
	}
	return in
}

// loAnalyse runs the whole measurement over a set of sources and returns the analysis as a VALUE.
func loAnalyse(sources []mutexSource) (loAnalysis, error) {
	a := loAnalysis{
		fieldTypes:  map[string]map[string]string{},
		mutexNames:  map[string]bool{},
		methodTakes: map[string][]string{},
		ambiguous:   map[string]bool{},
		seeds:       map[string][]map[string]int{},
		aliasFields: map[string]bool{},
		files:       len(sources),
	}
	if err := loDeriveFields(&a, sources); err != nil {
		return a, err
	}
	if err := loDeriveMethodTakes(&a, sources); err != nil {
		return a, err
	}
	// BEFORE the walk, because the walk records accesses only to fields this pass has PROVEN aliased.
	if err := loDeriveAliases(&a, sources); err != nil {
		return a, err
	}
	decls := map[string]loDeclRef{}
	for _, s := range sources {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, s.name, s.src, 0)
		if err != nil {
			return a, fmt.Errorf("parse %s: %w", s.name, err)
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			decls[fn.Name.Name] = loDeclRef{fset: fset, file: s.name, fn: fn}
			w := &loWalker{a: &a, fset: fset, file: s.name, fn: fn.Name.Name, vars: loVarsFor(fn), standalone: true}
			_ = w.block(fn.Body.List, map[string]int{})
		}
	}
	// ONE-HOP PROPAGATION INTO `...Locked` CALLEES. The tree's convention (Problems §28-§33) is that a
	// `...Locked` function takes no lock of its own BECAUSE the caller holds it. Walking those bodies
	// under the held set seen at their CALL SITES is what keeps a fix that MOVES a locked block into
	// such a helper visible. DEPTH 1: a seed found inside a seeded walk is not re-propagated.
	seedNames := make([]string, 0, len(a.seeds))
	for name := range a.seeds {
		seedNames = append(seedNames, name)
	}
	sort.Strings(seedNames) // deterministic walk order, so the edge list a reader sees is stable
	for _, name := range seedNames {
		ref, ok := decls[name]
		if !ok || ref.fn == nil || ref.fn.Body == nil {
			continue
		}
		for _, held := range a.seeds[name] {
			a.seeded++
			w := &loWalker{a: &a, fset: ref.fset, file: ref.file, fn: name, vars: loVarsFor(ref.fn)}
			_ = w.block(ref.fn.Body.List, cloneHeld(held))
		}
	}

	// Deduplicate so one line is one edge, in a stable order (a report a reader can diff).
	seen := map[string]bool{}
	var uniq []loEdge
	for _, e := range a.edges {
		k := e.from + " -> " + e.to + " @" + e.where
		if seen[k] {
			continue
		}
		seen[k] = true
		uniq = append(uniq, e)
	}
	sort.Slice(uniq, func(i, j int) bool { return uniq[i].where < uniq[j].where })
	a.edges = uniq
	return a, nil
}

// loInversions returns the pairs acquired in BOTH orders. The key is the A->B direction; the reverse
// direction is looked up in the same map, and only one of the two directions is reported.
func loInversions(edges []loEdge) ([]string, map[string][]loEdge) {
	pairs := map[string][]loEdge{}
	for _, e := range edges {
		pairs[e.from+" -> "+e.to] = append(pairs[e.from+" -> "+e.to], e)
	}
	var out []string
	for k := range pairs {
		parts := strings.Split(k, " -> ")
		if len(parts) != 2 {
			continue
		}
		if _, ok := pairs[parts[1]+" -> "+parts[0]]; ok && parts[0] < parts[1] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out, pairs
}

// loSources reads every non-test Go file in the package directory. Its own reader, deliberately: the
// sibling gate's comment states that each gate must be independently reason-able, and a shared loader
// is how one gate's change silently becomes another's.
func loSources(t *testing.T) []mutexSource {
	t.Helper()
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob sources: %v", err)
	}
	var out []mutexSource
	for _, p := range paths {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		src, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		out = append(out, mutexSource{name: p, src: src})
	}
	if len(out) == 0 {
		t.Fatal("no non-test sources found — the gate would pass vacuously")
	}
	return out
}

// lockOrderBaseline is a WORKLIST, not an excuse: every entry names a pair known to invert, with the
// reason it has not been fixed. It FAILS in both directions — a NEW inversion fails, and a BASELINED
// inversion that is gone also fails, because a baseline that never shrinks is how a gate stops
// measuring anything. It is currently EMPTY: the two inversions this gate found on its first run are
// fixed in the code, not written down here.
var lockOrderBaseline = map[string]string{}

// loPairInverts reports whether two mutex identities are acquired in BOTH orders anywhere in the graph.
func loPairInverts(edges []loEdge, a, b string) bool {
	ab, ba := false, false
	for _, e := range edges {
		if e.from == a && e.to == b {
			ab = true
		}
		if e.from == b && e.to == a {
			ba = true
		}
	}
	return ab && ba
}

// ============================================================================
// THE GATE, ON THE REAL TREE.
// ============================================================================

func TestNoCrossMutexLockOrderInversion(t *testing.T) {
	sources := loSources(t)
	a, err := loAnalyse(sources)
	if err != nil {
		t.Fatalf("analysis failed: %v", err)
	}

	// A blind pass reports a clean repository for ever, so the derivations must be PROVEN non-empty.
	if len(a.mutexNames) == 0 {
		t.Fatal("derivation found ZERO mutex-typed fields — the gate would pass vacuously")
	}
	if a.structs == 0 {
		t.Fatal("derivation found ZERO struct declarations — the gate would pass vacuously")
	}
	if len(a.methodTakes) == 0 {
		t.Fatal("derivation found ZERO methods taking a mutex — the ONE-CALL-EDGE pass would never fire")
	}
	if len(a.edges) == 0 {
		t.Fatal("walk produced ZERO order edges — nothing is being measured")
	}

	inversions, pairs := loInversions(a.edges)
	t.Logf("census: %d files, %d structs, %d mutex field names, %d method takes, %d ambiguous names skipped, %d `...Locked` bodies seeded from call sites, %d order edges, %d expressions skipped (never guessed), %d inversions",
		a.files, a.structs, len(a.mutexNames), len(a.methodTakes), len(a.ambiguous), a.seeded, len(a.edges), a.skipped, len(inversions))

	// 1. A pair acquired in BOTH orders is an ABBA deadlock. Every one must be fixed or baselined.
	for _, inv := range inversions {
		parts := strings.Split(inv, " -> ")
		if len(parts) != 2 {
			continue
		}
		rev := parts[1] + " -> " + parts[0]
		var witness strings.Builder
		for _, e := range pairs[inv] {
			fmt.Fprintf(&witness, "\n      A->B  %s", e.where)
		}
		for _, e := range pairs[rev] {
			fmt.Fprintf(&witness, "\n      B->A  %s", e.where)
		}
		if reason, ok := lockOrderBaseline[inv]; ok {
			t.Logf("BASELINED inversion %s (%s)%s", inv, reason, witness.String())
			continue
		}
		t.Errorf("CROSS-MUTEX LOCK-ORDER INVERSION: %s   <==>   %s%s\n"+
			"    Two goroutines can take these in opposite orders and never return. Fix it the way the\n"+
			"    two originals were fixed — snapshot under one lock, RELEASE, then take the other (or give\n"+
			"    the callee a `...Locked` form) — or add a baseline entry WITH the reason it is not fixed.",
			inv, rev, witness.String())
	}

	// 2. A baselined inversion that no longer exists must fail too: a baseline that never shrinks is
	//    how a gate quietly stops measuring anything.
	present := map[string]bool{}
	for _, inv := range inversions {
		present[inv] = true
	}
	for k, reason := range lockOrderBaseline {
		if !present[k] {
			t.Errorf("STALE baseline entry %q (%s) — it is no longer an inversion, so delete the line", k, reason)
		}
	}

	// 3. REGRESSION PIN, named. If either pair returns, the failure says WHICH pair and why it matters,
	//    rather than only printing an unexpected string.
	for _, pair := range [][2]string{
		{"AICitizenEngine.mu", "Lobby.mutex"},
		{"Lobby.mutex", "TokenSinkRouter.Mu"},
	} {
		if loPairInverts(a.edges, pair[0], pair[1]) {
			t.Errorf("REGRESSION: %s and %s are acquired in BOTH orders again.\n"+
				"    The first was live via `BehavioralTick` (holds `ace.mu`, takes `lobby.mutex`) against\n"+
				"    `runAutonomousTournament` and five Zen-Garden doors (hold `lobby.mutex`, reach the engine);\n"+
				"    the second via the persistence worker (holds the router lock, takes the lobby lock) against\n"+
				"    34 fee-routing sites whose `RouteCriminalTax` takes the router lock for WRITING.",
				pair[0], pair[1])
		}
	}
}

// ============================================================================
// THE CONTROL — a detector that cannot be SHOWN to fail is not a gate.
// ============================================================================
//
// Four shapes that MUST NOT report and two that MUST, in one synthetic file. It also pins two
// properties the real-tree run cannot prove by itself: that the ONE-CALL-EDGE pass actually resolves a
// call to the callee's own mutex, and that two DIFFERENT receiver names for the same mutex field
// canonicalise to ONE identity (`o.inner.mu` and `other.mu`).
func TestLockOrderGateDetectsAnInversion(t *testing.T) {
	src := []byte(`package main

import "sync"

type loOuter struct {
	mutex sync.RWMutex
	inner *loInner
}

type loInner struct {
	mu sync.RWMutex
}

func (i *loInner) Touch() {
	i.mu.Lock()
	defer i.mu.Unlock()
}

// MUST REPORT: holds loOuter.mutex and reaches loInner.mu ACROSS THE CALL EDGE.
func (o *loOuter) badEdge() {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	o.inner.Touch()
}

// MUST REPORT the other direction: holds loInner.mu, then takes loOuter.mutex.
func (o *loOuter) badIntra() {
	o.inner.mu.Lock()
	defer o.inner.mu.Unlock()
	o.mutex.Lock()
	defer o.mutex.Unlock()
}

// MUST NOT REPORT: sequential — each order is RELEASED before the other is taken.
func (o *loOuter) fineSequential() {
	o.mutex.RLock()
	o.mutex.RUnlock()
	o.inner.mu.RLock()
	o.inner.mu.RUnlock()
}

// MUST NOT REPORT: a go literal is a NEW GOROUTINE and inherits no lock.
func (o *loOuter) fineInNewGoroutine() {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	go func() {
		o.inner.mu.Lock()
		defer o.inner.mu.Unlock()
	}()
}

// MUST NOT ADD A REVERSE EDGE, but MUST canonicalise: other.mu is the SAME mutex as o.inner.mu.
func (o *loOuter) fineOtherObject(other *loInner) {
	other.mu.Lock()
	defer other.mu.Unlock()
	o.mutex.Lock()
	defer o.mutex.Unlock()
}
`)
	a, err := loAnalyse([]mutexSource{{name: "lo_control.go", src: src}})
	if err != nil {
		t.Fatalf("control analysis failed: %v", err)
	}
	count := func(from, to string) int {
		n := 0
		for _, e := range a.edges {
			if e.from == from && e.to == to {
				n++
			}
		}
		return n
	}

	// The ONE-CALL-EDGE pass must resolve `o.inner.Touch()` to loInner.mu. If it were blind this would
	// be 0 and the inversion below would silently disappear — which is exactly the failure mode a
	// control exists to catch.
	if got := count("loOuter.mutex", "loInner.mu"); got != 1 {
		t.Errorf("call-edge resolution: want exactly 1 `loOuter.mutex -> loInner.mu` edge, got %d "+
			"(the callee's receiver field must be resolved at the CALL SITE)", got)
	}
	// badIntra + fineOtherObject. A third would mean a shape that releases (or a goroutine literal)
	// contributed an edge it must not.
	if got := count("loInner.mu", "loOuter.mutex"); got != 2 {
		t.Errorf("want exactly 2 `loInner.mu -> loOuter.mutex` edges (badIntra + fineOtherObject), got %d "+
			"— a sequential release or a `go` literal leaked an edge", got)
	}
	inversions, _ := loInversions(a.edges)
	if len(inversions) != 1 {
		t.Fatalf("want exactly 1 inversion in the control, got %d: %v", len(inversions), inversions)
	}
	if inversions[0] != "loInner.mu -> loOuter.mutex" {
		t.Errorf("want the inversion reported as `loInner.mu -> loOuter.mutex`, got %q", inversions[0])
	}
}
