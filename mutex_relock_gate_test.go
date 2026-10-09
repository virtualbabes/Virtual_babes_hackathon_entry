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
// THE SAME-MUTEX RE-LOCK GATE — no function may take the same mutex twice
// without releasing it.
// ============================================================================
//
// WHY THIS EXISTS. `sync.Mutex` and `sync.RWMutex` are NOT re-entrant. A goroutine that acquires a
// mutex it already holds blocks FOR EVER, and because this tree's request paths hold the lobby WRITE
// lock across their bodies, "for ever" means the WHOLE PROCESS: every request, every WebSocket, every
// ticker. Nothing in the tree measured this shape until now.
//
// THE GATES THIS ONE COMPLEMENTS, and the seam between them is exact:
//   - `selflock_gate_test.go` owns the CALL shape — a function that calls a helper which takes the
//     lobby lock itself. Its registry is derived from `*Lobby` RECEIVERS ONLY, which is exactly why
//     it is structurally blind to any other type's mutex.
//   - `map_race_gate_test.go` owns MAP ACCESS without a lock.
//   - THIS gate owns the DIRECT shape, on ANY receiver type: a body that acquires a mutex which,
//     on that path, it is already holding. That is the shape two LIVE doors carried —
//     `handleInvestEntity` and `HandleDirectInvest` each took `EntityMarketNode.Mu` for writing and
//     then re-took it for reading on the success path, so EVERY successful direct investment froze
//     the process. Both are fixed; this gate is why the next one is a build failure rather than
//     another reading exercise.
//
// WHAT IT MEASURES, in one line: within ONE function body, is a mutex-typed expression acquired at a
// point where that same expression is DEFINITELY already held?
//
// THE PROTECTED SET IS DERIVED, NOT CURATED: every struct field whose type is `sync.Mutex` or
// `sync.RWMutex`, parsed from the struct declarations themselves, so a mutex added tomorrow is
// measured tomorrow. A curated list goes stale on the first new field, and a stale gate reads exactly
// like a clean one. If the derivation finds ZERO mutex fields the gate FAILS, because a detector that
// matches nothing reports a clean repository for ever.
//
// THE ANALYSIS IS MUST-HELD AND ONE CALL EDGE DEEP — stated, not implied:
//   - MUST-HELD. `if`/`else`, `switch`/`select` clauses and loops are JOINED: a mutex counts as held
//     after a branch only if EVERY path through it acquired it. A loop may run zero times, so it can
//     only carry out what was already held. That is what removed the flat-source-order false
//     positives which made the sibling gate's first version unusable; its mirror image is a FALSE
//     NEGATIVE (a re-lock on a branch-only-held mutex), which is the safe direction.
//   - A `defer`red RELEASE IS NOT A RELEASE AT ITS SOURCE LINE. `x.Mu.Lock(); defer x.Mu.Unlock();
//     ...; x.Mu.RLock()` really does deadlock, so the deferred unlock clears the state only at
//     function exit. This is what makes the dominant idiom of this tree visible.
//   - A `go` STATEMENT STARTS A NEW GOROUTINE, WHICH DOES NOT INHERIT THE LOCK: a function literal
//     under `go` is analysed from an EMPTY held-set, so it cannot become a false positive. A closure
//     that is NOT under `go` shares the goroutine, so it IS analysed from the current held-set.
//   - An expression this gate cannot render to a stable key (`exprKey` returns "") is SKIPPED, never
//     guessed: a reported `x.Mu` that is not the same mutex would make the gate noise, and "prove it
//     is safe" is the work.
//
// PASS 2 — ONE CALL EDGE — closes the limit pass 1 had stated for itself. A body that holds `x.Mu` and
// calls a method which takes `x.Mu` deadlocks exactly like a direct re-lock, and no reading of a single
// body can see it: this is the shape `CalculateTotalPortfolioValueLocked` forbids in a comment. The callee
// registry is DERIVED from the tree, the RECEIVER EXPRESSION AT THE CALL SITE is the identity, and:
//   - `*Lobby` receivers are EXCLUDED, because `selflock_gate_test.go` owns that shape — a shape owned by
//     two gates is how one of them starts lying;
//   - a method name declared by two types with DIFFERENT acquired-field sets is SKIPPED, never guessed;
//   - a `go` callee is NOT an edge (a new goroutine does not inherit the lock) although its ARGUMENTS are;
//     a `defer`red callee is NOT analysed (LIFO makes the held-set at the `defer` statement the wrong
//     question) and a call in a `range` EXPRESSION is not walked — both are stated false negatives;
//   - DEPTH 1 ONLY: a chain deeper than one edge is the next step if a measured instance ever appears.
//   - THE RACE DETECTOR IS UNAVAILABLE HERE (`go test -race` needs cgo and this host has no C
//     toolchain), so this is a STATIC census of a shape that is a deadlock BY CONSTRUCTION, not a
//     race proof.
// ============================================================================

// mutexSource is one file handed to the detector. It is a SOURCE rather than a path so the controls
// below can drive the detector with synthetic code.
type mutexSource struct {
	name string
	src  []byte
}

// relockEntry is one function that acquires a mutex it is already holding — directly, or across the
// single CALL EDGE named by `via`.
type relockEntry struct {
	key   string // "Type.Method" or "FuncName"
	expr  string // the expression re-acquired, e.g. "node.Mu"
	first int    // line of the first acquisition
	again int    // line of the re-acquisition
	via   string // "" for a direct re-lock; else the callee that takes the mutex (the WITNESS edge)
}

func (e relockEntry) String() string {
	where := ""
	if e.via != "" {
		where = fmt.Sprintf(" [via the call to %s]", e.via)
	}
	return fmt.Sprintf("%s re-locks %s at line %d (already held since line %d)%s", e.key, e.expr, e.again, e.first, where)
}

// isSyncMutexType reports whether a field type is a Go mutex (or a pointer to one).
func isSyncMutexType(e ast.Expr) bool {
	for {
		switch x := e.(type) {
		case *ast.StarExpr:
			e = x.X
		case *ast.ParenExpr:
			e = x.X
		case *ast.SelectorExpr:
			base, ok := x.X.(*ast.Ident)
			return ok && base.Name == "sync" && (x.Sel.Name == "Mutex" || x.Sel.Name == "RWMutex")
		default:
			return false
		}
	}
}

// derivedMutexFields reads every struct field that IS a mutex, from the declarations rather than from
// a list, and reports the census so the gate can prove it is looking at something.
func derivedMutexFields(dir string) (map[string]bool, int, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return nil, 0, err
	}
	fields := map[string]bool{}
	structs := 0
	for _, p := range paths {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			return nil, 0, fmt.Errorf("parse %s: %w", p, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok || st.Fields == nil {
				return true
			}
			structs++
			for _, f := range st.Fields.List {
				if f.Type == nil || !isSyncMutexType(f.Type) {
					continue
				}
				for _, name := range f.Names {
					if name.Name != "" && name.Name != "_" {
						fields[name.Name] = true
					}
				}
			}
			return true
		})
	}
	return fields, structs, nil
}

// exprKey renders an expression into a stable key, or "" when it cannot be rendered without guessing.
// Only ident / selector / index chains are rendered and an index is rendered as `[]` (the ELEMENT is
// taken from the same collection, so the name identifies the mutex); anything else is skipped.
func exprKey(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.ParenExpr:
		return exprKey(x.X)
	case *ast.SelectorExpr:
		base := exprKey(x.X)
		if base == "" {
			return ""
		}
		return base + "." + x.Sel.Name
	case *ast.IndexExpr:
		base := exprKey(x.X)
		if base == "" {
			return ""
		}
		return base + "[]"
	case *ast.StarExpr:
		base := exprKey(x.X)
		if base == "" {
			return ""
		}
		return "*" + base
	default:
		return ""
	}
}

// mutexTarget resolves a METHOD CALL's receiver to "<expr>.<mutexField>". For `n.Mu.RLock()` the call
// is the selector `n.Mu . RLock`, so the RECEIVER is `n.Mu`, the mutex field is `Mu` and the operand is
// `n` — this function returns "n.Mu". It returns "" when the call is not a lock operation on a derived
// mutex field, which SKIPS rather than guesses: a reported `x.Mu` that is not the same mutex would make
// the gate noise, and a noisy gate gets switched off.
func mutexTarget(call *ast.SelectorExpr, mutexFields map[string]bool) string {
	recv, ok := call.X.(*ast.SelectorExpr)
	if !ok || !mutexFields[recv.Sel.Name] {
		return ""
	}
	base := exprKey(recv.X)
	if base == "" {
		return ""
	}
	return base + "." + recv.Sel.Name
}

func cloneHeld(in map[string]int) map[string]int {
	out := make(map[string]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// intersectHeld keeps only the mutexes held on EVERY path (MUST-held).
func intersectHeld(a, b map[string]int) map[string]int {
	out := map[string]int{}
	for k, v := range a {
		if _, ok := b[k]; ok {
			out[k] = v
		}
	}
	return out
}

// relockWalk walks ONE function body threading the set of DEFINITELY-held mutexes.
type relockWalk struct {
	fset        *token.FileSet
	fn          string
	mutexFields map[string]bool
	out         *[]relockEntry
	// acquired collects every mutex expression this body ACQUIRES (expr -> the line it first does so).
	// It is set only while the callee registry is being derived, and it is what makes that registry
	// DERIVED from the tree rather than a list somebody has to remember to maintain.
	acquired map[string]int
	// methods is pass 2's registry, keyed by METHOD NAME. nil disables the edge pass, so the registry
	// walk itself cannot recurse into itself.
	methods map[string][]methodMutexTake
}

func (w *relockWalk) report(expr string, first, line int, via string) {
	if w.out == nil {
		return // a registry-only walk, which reports nothing (see deriveMethodMutexTakes)
	}
	*w.out = append(*w.out, relockEntry{key: w.fn, expr: expr, first: first, again: line, via: via})
}

func (w *relockWalk) line(n ast.Node) int { return w.fset.Position(n.Pos()).Line }

// lockCall records one lock/unlock operation. `deferred` marks a call under `defer`, where a RELEASE
// does not happen at this line (it happens at function exit) while an acquisition still does.
func (w *relockWalk) lockCall(c *ast.CallExpr, in map[string]int, deferred bool) map[string]int {
	sel, ok := c.Fun.(*ast.SelectorExpr)
	if !ok {
		return in
	}
	expr := mutexTarget(sel, w.mutexFields)
	if expr == "" {
		return in
	}
	switch sel.Sel.Name {
	case "Lock", "RLock":
		if deferred {
			// `defer x.Mu.Lock()` acquires at return, not here: nothing is held for this body.
			return in
		}
		if first, held := in[expr]; held {
			w.report(expr, first, w.line(c), "")
			return in
		}
		in[expr] = w.line(c)
		if w.acquired != nil {
			w.acquired[expr] = w.line(c)
		}
	case "Unlock", "RUnlock":
		if deferred {
			// A release that happens at function exit keeps the lock held for the rest of the body:
			// this single rule is what makes `Lock(); defer Unlock(); ...; RLock()` visible.
			return in
		}
		delete(in, expr)
	}
	return in
}

// block walks a statement list in order.
func (w *relockWalk) block(list []ast.Stmt, in map[string]int) map[string]int {
	cur := cloneHeld(in)
	for _, s := range list {
		cur = w.stmt(s, cur)
	}
	return cur
}

func (w *relockWalk) stmt(s ast.Stmt, in map[string]int) map[string]int {
	switch x := s.(type) {
	case nil:
		return in
	case *ast.BlockStmt:
		return w.block(x.List, in)
	case *ast.ExprStmt:
		w.exprForLits(x.X, in)
		if c, ok := x.X.(*ast.CallExpr); ok {
			return w.lockCall(c, in, false)
		}
		return in
	case *ast.DeferStmt:
		w.exprForLits(x.Call.Fun, in)
		return w.lockCall(x.Call, in, true)
	case *ast.GoStmt:
		// A new goroutine does NOT inherit the lock: a literal under `go` starts from nothing.
		if lit, ok := x.Call.Fun.(*ast.FuncLit); ok {
			_ = w.block(lit.Body.List, map[string]int{})
			return in
		}
		// Arguments are evaluated in THIS goroutine, so they keep the current state.
		for _, a := range x.Call.Args {
			w.exprForLits(a, in)
		}
		return in
	case *ast.IfStmt:
		base := in
		if x.Init != nil {
			base = w.stmt(x.Init, base)
		}
		w.exprForLits(x.Cond, base)
		thenOut := w.block(x.Body.List, base)
		elseOut := base
		if x.Else != nil {
			elseOut = w.stmt(x.Else, base)
		}
		return intersectHeld(thenOut, elseOut)
	case *ast.SwitchStmt:
		base := in
		if x.Init != nil {
			base = w.stmt(x.Init, base)
		}
		return w.clauses(x.Body.List, base)
	case *ast.TypeSwitchStmt:
		base := in
		if x.Init != nil {
			base = w.stmt(x.Init, base)
		}
		if x.Assign != nil {
			base = w.stmt(x.Assign, base)
		}
		return w.clauses(x.Body.List, base)
	case *ast.SelectStmt:
		return w.clauses(x.Body.List, in)
	case *ast.ForStmt:
		// A loop may run zero times: only what was held BEFORE it survives it.
		bodyIn := in
		if x.Init != nil {
			bodyIn = w.stmt(x.Init, bodyIn)
		}
		if x.Cond != nil {
			w.exprForLits(x.Cond, bodyIn)
		}
		_ = w.block(x.Body.List, bodyIn)
		after := cloneHeld(in)
		if x.Post != nil {
			after = w.stmt(x.Post, after)
		}
		return after
	case *ast.RangeStmt:
		_ = w.block(x.Body.List, in)
		return in
	case *ast.LabeledStmt:
		return w.stmt(x.Stmt, in)
	}
	// Anything else (assignments, returns, declarations) can still CONTAIN a closure or a lock call in
	// an expression position, so its sub-expressions are walked for closures — which is how a lock
	// taken inside a literal is attributed to this body rather than lost.
	w.exprForLits(stmtExpressions(s), in)
	return in
}

// clauses joins every clause of a switch/select. Clauses are exclusive, so a mutex counts as held
// afterwards only if EVERY clause — and, when there is no `default`, the no-clause path — held it.
func (w *relockWalk) clauses(list []ast.Stmt, in map[string]int) map[string]int {
	acc := cloneHeld(in)
	first := true
	sawDefault := false
	for _, s := range list {
		var body []ast.Stmt
		var isDefault bool
		switch c := s.(type) {
		case *ast.CaseClause:
			body, isDefault = c.Body, len(c.List) == 0
		case *ast.CommClause:
			body, isDefault = c.Body, c.Comm == nil
		default:
			continue
		}
		if isDefault {
			sawDefault = true
		}
		out := w.block(body, in)
		if first {
			acc, first = out, false
		} else {
			acc = intersectHeld(acc, out)
		}
	}
	if !sawDefault {
		acc = intersectHeld(acc, in)
	}
	return acc
}

// stmtExpressions returns the statement itself, which `ast.Inspect` then walks for closures. It
// exists so the default branch above reads as a decision rather than as a type switch with no default.
func stmtExpressions(s ast.Stmt) ast.Node { return s }

// exprForLits analyses a closure found in an expression from the CURRENT held-set: a closure that is
// not under `go` runs in this goroutine, so it shares the lock and can deadlock on it.
func (w *relockWalk) exprForLits(n ast.Node, in map[string]int) {
	if n == nil {
		return
	}
	ast.Inspect(n, func(node ast.Node) bool {
		// PASS 2 — every call in an EXPRESSION position is a candidate edge: the callee runs in THIS
		// goroutine (its arguments certainly do), so a method that takes a mutex this body holds
		// re-locks it.
		if call, ok := node.(*ast.CallExpr); ok {
			w.callEdge(call, in)
		}
		lit, ok := node.(*ast.FuncLit)
		if !ok {
			return true
		}
		_ = w.block(lit.Body.List, cloneHeld(in))
		return false // its body is walked once, as its own scope
	})
}

// ============================================================================
// PASS 2 — THE SAME RE-LOCK ACROSS ONE CALL EDGE.
// ============================================================================
//
// The limit pass 1 stated for itself was "the analysis is intra-procedural", and this tree has already
// paid for that limit: `CalculateTotalPortfolioValueLocked`'s second contract clause ("NO node lock may
// be held") is enforced by a comment, a test and a hoist rather than by a gate, and the shape it forbids
// — a body that holds `x.Mu` and calls a method which takes `x.Mu` — is statically reachable. A callee
// that takes the lock its caller already holds cannot return, so it deadlocks exactly like a direct
// re-lock; it is merely harder to see, which is why it needs a machine rather than a reader.

// mutexAcq is one mutex field a method acquires through its own receiver, with the line it first does so.
type mutexAcq struct {
	field string
	line  int
}

// methodMutexTake is what ONE method does to its receiver's mutex.
type methodMutexTake struct {
	recvType string
	fields   []mutexAcq
}

// signature renders the acquired FIELD NAMES in a stable order, which is all that is needed to decide
// whether two same-named methods can be treated as one.
func (m methodMutexTake) signature() string {
	names := make([]string, 0, len(m.fields))
	for _, f := range m.fields {
		names = append(names, f.field)
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}

// deriveMethodMutexTakes builds pass 2's registry FROM THE TREE: every method whose body acquires a mutex
// field through its own receiver, keyed by method name. `*Lobby` receivers are excluded ON PURPOSE —
// `selflock_gate_test.go` owns that shape, and a shape owned by two gates is how one of them starts
// lying. It returns the registry, the number of definitions derived (the liveness census) and the number
// of method names SKIPPED for being declared by two types with different acquired-field sets.
func deriveMethodMutexTakes(sources []mutexSource, mutexFields map[string]bool) (map[string][]methodMutexTake, int, int, error) {
	byName := map[string][]methodMutexTake{}
	defined := 0
	for _, s := range sources {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, s.name, s.src, 0)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("parse %s: %w", s.name, err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Recv == nil || len(fn.Recv.List) == 0 {
				continue
			}
			recvType := relockReceiverType(fn.Recv.List[0].Type)
			if recvType == "" || recvType == "Lobby" {
				continue
			}
			names := fn.Recv.List[0].Names
			if len(names) == 0 {
				continue
			}
			recvName := names[0].Name

			// `out` is nil, so the walk REPORTS nothing: it only collects what this method acquires.
			acquired := map[string]int{}
			w := &relockWalk{fset: fset, fn: funcDeclKey(fn), mutexFields: mutexFields, acquired: acquired}
			_ = w.block(fn.Body.List, map[string]int{})

			var fields []mutexAcq
			for expr, line := range acquired {
				if !strings.HasPrefix(expr, recvName+".") {
					continue // some other object's mutex: not this receiver's own field
				}
				field := expr[len(recvName)+1:]
				if field == "" || strings.Contains(field, ".") {
					continue // `x.Mu` is the shape this pass models; a nested path is not guessed at
				}
				fields = append(fields, mutexAcq{field: field, line: line})
			}
			if len(fields) == 0 {
				continue
			}
			sort.Slice(fields, func(i, j int) bool { return fields[i].field < fields[j].field })
			byName[fn.Name.Name] = append(byName[fn.Name.Name], methodMutexTake{recvType: recvType, fields: fields})
			defined++
		}
	}

	// A method name declared by two types that do NOT acquire the same fields cannot be resolved BY NAME,
	// so it is SKIPPED rather than guessed: a reported `x.Mu` that is not the mutex actually in hand would
	// make the gate noise, and a noisy gate gets switched off.
	ambiguous := 0
	for name, takes := range byName {
		if len(takes) < 2 {
			continue
		}
		first := takes[0].signature()
		for _, t := range takes[1:] {
			if t.signature() != first {
				delete(byName, name)
				ambiguous++
				break
			}
		}
	}
	return byName, defined, ambiguous, nil
}

// relockReceiverType renders a method receiver's type name, unwrapping the pointer. It is deliberately
// its own function rather than the sibling gate's: the gates must be able to be fixed independently, and
// a shared helper is how one gate's change silently becomes another's.
func relockReceiverType(e ast.Expr) string {
	for {
		switch x := e.(type) {
		case *ast.StarExpr:
			e = x.X
		case *ast.ParenExpr:
			e = x.X
		case *ast.Ident:
			return x.Name
		default:
			return ""
		}
	}
}

// callEdge resolves ONE call against pass 2's registry. THE RECEIVER EXPRESSION AT THE CALL SITE IS THE
// IDENTITY: `x.B()` re-locks `x.Mu` only when `B` acquires `Mu` AND the caller holds exactly `x.Mu`. A
// call on a different object is a different mutex, not a finding — that rule is what keeps this pass from
// reporting correct code.
func (w *relockWalk) callEdge(call *ast.CallExpr, in map[string]int) {
	if w.methods == nil {
		return
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	takes, ok := w.methods[sel.Sel.Name]
	if !ok {
		return
	}
	recv := exprKey(sel.X)
	if recv == "" {
		return // an expression this pass cannot render — skipped, never guessed
	}
	for _, t := range takes {
		for _, f := range t.fields {
			expr := recv + "." + f.field
			if first, held := in[expr]; held {
				w.report(expr, first, w.line(call), fmt.Sprintf("%s.%s()", recv, sel.Sel.Name))
			}
		}
	}
}

// measureRelocksAcrossOneCallEdge is pass 2: every body walked ONCE against the derived registry, so a
// body that holds `x.Mu` and calls a method taking `x.Mu` is reported WITH ITS WITNESS EDGE.
func measureRelocksAcrossOneCallEdge(sources []mutexSource, mutexFields map[string]bool) ([]relockEntry, int, int, error) {
	methods, defined, ambiguous, err := deriveMethodMutexTakes(sources, mutexFields)
	if err != nil {
		return nil, 0, 0, err
	}
	var out []relockEntry
	seen := map[string]bool{}
	for _, s := range sources {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, s.name, s.src, 0)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("parse %s: %w", s.name, err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			// One call can be reached from two expression positions of one statement (the statement, and
			// its own sub-expression), so findings are de-duplicated by everything a reader needs.
			var body []relockEntry
			w := &relockWalk{fset: fset, fn: funcDeclKey(fn), mutexFields: mutexFields, out: &body, methods: methods}
			_ = w.block(fn.Body.List, map[string]int{})
			for _, e := range body {
				k := fmt.Sprintf("%s|%s|%s|%d", e.key, e.expr, e.via, e.again)
				if seen[k] {
					continue
				}
				seen[k] = true
				out = append(out, e)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].key != out[j].key {
			return out[i].key < out[j].key
		}
		return out[i].again < out[j].again
	})
	return out, defined, ambiguous, nil
}

// measureSameMutexRelocks reports every function that re-acquires a mutex it already holds.
func measureSameMutexRelocks(sources []mutexSource, mutexFields map[string]bool) ([]relockEntry, error) {
	var out []relockEntry
	for _, s := range sources {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, s.name, s.src, 0)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", s.name, err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			w := &relockWalk{fset: fset, fn: funcDeclKey(fn), mutexFields: mutexFields, out: &out}
			_ = w.block(fn.Body.List, map[string]int{})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].key != out[j].key {
			return out[i].key < out[j].key
		}
		return out[i].again < out[j].again
	})
	return out, nil
}

// realMutexSources reads every non-test Go file in the package directory. It is deliberately its own
// reader rather than a reuse of the sibling gate's: the three gates must be able to be reasoned about
// (and fixed) independently, and a shared loader is how one gate's change silently becomes another's.
func realMutexSources(t *testing.T) []mutexSource {
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

// ============================================================================
// THE BASELINE — a WORKLIST, not an excuse.
// ============================================================================
//
// The key is "Type.Method  expr" — the FUNCTION and the MUTEX expression — deliberately WITHOUT the
// line numbers, so an unrelated edit elsewhere cannot invalidate a line and report a false STALE. The
// line numbers travel in the failure message instead, which is where a reader needs them.
//
// Every entry is a MEASURED, still-unfixed re-lock. A re-lock is a deadlock by construction, so unlike
// the map gate there is no "CONTRACT" verdict to earn: the only two states are FIXED (the line is
// deleted, and the gate FAILS until it is) and KNOWN-OPEN (recorded here with its reason).
var sameMutexRelockBaseline = map[string]string{}

// sameMutexRelockViaCallBaseline is PASS 2's worklist, keyed "Type.Method  expr  via Callee()" — again
// WITHOUT line numbers, so an unrelated edit elsewhere cannot produce a false STALE.
var sameMutexRelockViaCallBaseline = map[string]string{}

// TestNoSameMutexRelock IS the gate.
func TestNoSameMutexRelock(t *testing.T) {
	mutexFields, structs, err := derivedMutexFields(".")
	if err != nil {
		t.Fatalf("the mutex set must be derived from the struct declarations: %v", err)
	}
	if len(mutexFields) == 0 {
		t.Fatal("the derivation found ZERO mutex-typed struct fields — a detector that matches nothing " +
			"would report a clean repository for ever")
	}

	entries, err := measureSameMutexRelocks(realMutexSources(t), mutexFields)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}

	// THE DETECTOR'S OWN LIVENESS CHECK, inside the gate. A tree with ZERO re-locks is exactly what a
	// BLIND detector reports, so the clean report below cannot be trusted on its own — the gate must
	// prove, on every run, that it can still see the shape it exists for. (The first version of this
	// detector reported 0 for EVERY input; see the file header. That is not a hypothetical.)
	const liveness = `package main

import "sync"

type Live struct{ Mu sync.RWMutex }

func (v *Live) mustBeSeen() {
	v.Mu.Lock()
	defer v.Mu.Unlock()
	v.Mu.RLock()
}
`
	probe, err := measureSameMutexRelocks([]mutexSource{{name: "liveness.go", src: []byte(liveness)}},
		map[string]bool{"Mu": true})
	if err != nil {
		t.Fatalf("liveness probe must parse: %v", err)
	}
	if len(probe) == 0 {
		t.Fatal("THE DETECTOR IS BLIND: it reported nothing for a function that takes the same mutex twice, " +
			"so its clean report about the tree means nothing. A gate whose matcher silently failed reports " +
			"a clean repository for ever.")
	}

	seen := map[string]bool{}
	var report, newOnes []string
	for _, e := range entries {
		key := fmt.Sprintf("%s  %s", e.key, e.expr)
		seen[key] = true
		report = append(report, e.String())
		if _, ok := sameMutexRelockBaseline[key]; !ok {
			newOnes = append(newOnes, e.String())
		}
	}
	var stale []string
	for k := range sameMutexRelockBaseline {
		if !seen[k] {
			stale = append(stale, k)
		}
	}
	sort.Strings(newOnes)
	sort.Strings(stale)
	sort.Strings(report)

	t.Logf("same-mutex re-lock gate: %d mutex fields derived from %d struct declarations; %d re-locks in "+
		"the tree; %d baselined", len(mutexFields), structs, len(entries), len(sameMutexRelockBaseline))

	if len(newOnes) > 0 {
		t.Errorf("SAME-MUTEX RE-LOCK. `sync.RWMutex` and `sync.Mutex` are NOT re-entrant: a goroutine "+
			"that takes a mutex it already holds blocks FOR EVER, and because this tree holds the lobby "+
			"write lock across its request bodies, that freezes the WHOLE process — every request, every "+
			"WebSocket, not merely the calling one. Take the lock ONCE per function and read the fields "+
			"inside it, move the nested acquisition outside the held section, or add a `...Locked` "+
			"sibling for the inner step:\n  %s", strings.Join(newOnes, "\n  "))
	}
	if len(stale) > 0 {
		t.Errorf("STALE baseline entries: these no longer re-lock a held mutex. That is a FIX, so DELETE "+
			"its line — a baseline that outlives its reason reads like a clean repository while "+
			"legitimising rot:\n  %s", strings.Join(stale, "\n  "))
	}
	if len(newOnes) > 0 || len(stale) > 0 {
		t.Logf("census of this run:\n  %s", strings.Join(report, "\n  "))
	}

	// ------------------------------------------------------------------------
	// PASS 2 — the same question asked across ONE CALL EDGE.
	// ------------------------------------------------------------------------
	viaEntries, methodTakes, ambiguous, err := measureRelocksAcrossOneCallEdge(realMutexSources(t), mutexFields)
	if err != nil {
		t.Fatalf("pass 2 measure: %v", err)
	}
	// A clean edge report means NOTHING unless the registry is non-empty: a derivation that matched no
	// method would find no edge for ever, which is precisely how the first version of the pass-1 detector
	// behaved (0 findings for EVERY input).
	if methodTakes == 0 {
		t.Fatal("pass 2 derived ZERO method takes — the registry is empty, so a clean edge report would " +
			"be the report of a blind pass rather than of a clean tree")
	}
	seenVia := map[string]bool{}
	var viaReport, newVia, staleVia []string
	for _, e := range viaEntries {
		key := fmt.Sprintf("%s  %s  via %s", e.key, e.expr, e.via)
		seenVia[key] = true
		viaReport = append(viaReport, e.String())
		if _, ok := sameMutexRelockViaCallBaseline[key]; !ok {
			newVia = append(newVia, e.String())
		}
	}
	for k := range sameMutexRelockViaCallBaseline {
		if !seenVia[k] {
			staleVia = append(staleVia, k)
		}
	}
	sort.Strings(newVia)
	sort.Strings(staleVia)
	sort.Strings(viaReport)

	t.Logf("same-mutex re-lock gate, pass 2 (one call edge): %d method takes derived, %d ambiguous method "+
		"names skipped, %d findings, %d baselined", methodTakes, ambiguous, len(viaEntries),
		len(sameMutexRelockViaCallBaseline))

	if len(newVia) > 0 {
		t.Errorf("SAME-MUTEX RE-LOCK ACROSS A CALL EDGE. A body that holds `x.Mu` and calls a method which "+
			"takes `x.Mu` itself cannot return, so it deadlocks exactly like a direct re-lock — and no "+
			"reading of one body can see it. Move the nested acquisition outside the held section, or call "+
			"the `...Locked` sibling of the callee:\n  %s", strings.Join(newVia, "\n  "))
	}
	if len(staleVia) > 0 {
		t.Errorf("STALE pass-2 baseline entries: these no longer re-lock across an edge, so that is a FIX "+
			"and the line must be DELETED — a baseline that outlives its reason reads like a clean "+
			"repository while legitimising rot:\n  %s", strings.Join(staleVia, "\n  "))
	}
	if len(newVia) > 0 || len(staleVia) > 0 {
		t.Logf("pass-2 census of this run:\n  %s", strings.Join(viaReport, "\n  "))
	}
}

// TestMutexRelockDerivationSeesTheMutexesTheDefectUsed is the DERIVATION's own control. A derivation
// that silently returned an empty set would report a clean repository for ever, and the clean report is
// exactly what the gate above asserts about the whole tree.
func TestMutexRelockDerivationSeesTheMutexesTheDefectUsed(t *testing.T) {
	fields, structs, err := derivedMutexFields(".")
	if err != nil {
		t.Fatalf("derivation: %v", err)
	}
	if structs < 100 {
		t.Errorf("only %d struct declarations were read; this package holds far more, so the derivation "+
			"is not looking at the tree it claims to", structs)
	}
	// `Mu` is EntityMarketNode's (where the two live doors deadlocked) and `mutex` is Lobby's. If the
	// derivation cannot see those, it cannot see the shape this gate exists for.
	for _, must := range []string{"Mu", "mutex"} {
		if !fields[must] {
			t.Errorf("the derivation did not see a mutex field named %q — the gate is blind to it", must)
		}
	}
	names := make([]string, 0, len(fields))
	for n := range fields {
		names = append(names, n)
	}
	sort.Strings(names)
	t.Logf("derived mutex field names (%d): %s", len(names), strings.Join(names, ", "))
}

// TestSyncMutexTypeRuleIsNotOverBroad pins the type rule itself: a matcher that accepted `sync.Map` or
// another package's `RWMutex` would report a mutex that is not one, and a matcher that missed a
// POINTER to a mutex would miss a real one.
func TestSyncMutexTypeRuleIsNotOverBroad(t *testing.T) {
	const src = `package main

import "sync"

type S struct {
	A sync.RWMutex
	B sync.Mutex
	C *sync.Mutex
	D sync.Map
	E int
	F other.RWMutex
	G chan int
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sync_types.go", []byte(src), 0)
	if err != nil {
		t.Fatalf("control source must parse: %v", err)
	}
	got := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		st, ok := n.(*ast.StructType)
		if !ok || st.Fields == nil {
			return true
		}
		for _, f := range st.Fields.List {
			for _, name := range f.Names {
				got[name.Name] = isSyncMutexType(f.Type)
			}
		}
		return true
	})
	want := map[string]bool{"A": true, "B": true, "C": true, "D": false, "E": false, "F": false, "G": false}
	for name, expect := range want {
		if got[name] != expect {
			t.Errorf("isSyncMutexType(%s) = %v, want %v", name, got[name], expect)
		}
	}
}

// TestSameMutexRelockGateDetectsAViolation is the NEGATIVE CONTROL, and it is what gives the gate above
// its meaning: a detector whose matcher silently failed would report a clean repository for ever, and a
// clean report is exactly what the real scan asserts.
//
// The synthetic file carries FIFTEEN functions. FIVE must be reported — including the EXACT shape the
// two live doors carried — and TEN must not, because those are the false positives that would make the
// gate unusable, and one of them is the shape a new goroutine creates.
func TestSameMutexRelockGateDetectsAViolation(t *testing.T) {
	const src = `package main

import "sync"

type Node struct {
	Mu    sync.RWMutex
	Thing Widget
}

type Widget struct{}

func (w *Widget) Lock() {}

// MUST REPORT 1 — the exact shape the two direct-invest doors carried: the write lock is taken for the
// whole body (deferred release) and then re-taken for reading.
func (n *Node) badWriteThenRead() {
	n.Mu.Lock()
	defer n.Mu.Unlock()
	n.Mu.RLock()
}

// MUST REPORT 2 — a recursive READ lock. The writer that queues decides it, so it is a deadlock.
func (n *Node) badRecursiveRead() {
	n.Mu.RLock()
	defer n.Mu.RUnlock()
	n.Mu.RLock()
}

// MUST REPORT 3 — a read lock ESCALATED to a write lock inside one body.
func (n *Node) badReadThenWrite() {
	n.Mu.RLock()
	n.Mu.Lock()
}

// MUST REPORT 4 — BOTH branches take it, so it is held on every path reaching the re-lock.
func (n *Node) badInBothBranches(b bool) {
	if b {
		n.Mu.Lock()
		defer n.Mu.Unlock()
	} else {
		n.Mu.Lock()
		defer n.Mu.Unlock()
	}
	n.Mu.RLock()
}

// MUST REPORT 5 — taken before the loop, re-taken inside it.
func (n *Node) badInsideLoop() {
	n.Mu.Lock()
	defer n.Mu.Unlock()
	for i := 0; i < 3; i++ {
		n.Mu.RLock()
	}
}

// MUST NOT 1 — a proper release and re-acquire: the shape a correct tree is made of.
func (n *Node) fineSequential() {
	n.Mu.Lock()
	n.Mu.Unlock()
	n.Mu.Lock()
	n.Mu.Unlock()
}

// MUST NOT 2 — two DIFFERENT objects: the EXPRESSION is the identity, not the field name.
func (a *Node) fineTwoObjects(b *Node) {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	b.Mu.Lock()
	defer b.Mu.Unlock()
}

// MUST NOT 3 — a NEW GOROUTINE DOES NOT INHERIT THE LOCK: the literal cannot deadlock on a mutex its
// parent holds; it only waits until the parent releases, which it does at function exit.
func (n *Node) fineInNewGoroutine() {
	n.Mu.Lock()
	defer n.Mu.Unlock()
	go func() {
		n.Mu.Lock()
		defer n.Mu.Unlock()
	}()
}

// MUST NOT 4 — a Lock() method on a type the derivation did NOT call a mutex.
func (n *Node) fineCustomLocker() {
	n.Thing.Lock()
	n.Thing.Lock()
}

// MUST NOT 5 — held on ONE branch only (no else): MUST-held keeps nothing, so the re-lock after the
// branch is NOT reported. That is this gate's documented false-NEGATIVE direction.
func (n *Node) fineOneBranchOnly(b bool) {
	if b {
		n.Mu.Lock()
		defer n.Mu.Unlock()
	}
	n.Mu.RLock()
}
`
	entries, err := measureSameMutexRelocks(
		[]mutexSource{{name: "relock_synthetic.go", src: []byte(src)}},
		map[string]bool{"Mu": true})
	if err != nil {
		t.Fatalf("the control source must parse and measure: %v", err)
	}

	reported := map[string]bool{}
	for _, e := range entries {
		reported[e.key] = true
	}
	must := []string{"badWriteThenRead", "badRecursiveRead", "badReadThenWrite", "badInBothBranches", "badInsideLoop"}
	for _, name := range must {
		found := false
		for key := range reported {
			if strings.HasSuffix(key, "."+name) {
				found = true
			}
		}
		if !found {
			t.Errorf("the detector MISSED %s, which re-locks a mutex it is already holding — the gate "+
				"cannot see the shape it exists for (found: %v)", name, entries)
		}
	}
	mustNot := []string{"fineSequential", "fineTwoObjects", "fineInNewGoroutine", "fineCustomLocker", "fineOneBranchOnly"}
	for _, name := range mustNot {
		for key := range reported {
			if strings.HasSuffix(key, "."+name) {
				t.Errorf("the detector reported %s, which does NOT re-lock a held mutex — a gate that "+
					"reports correct code gets switched off", key)
			}
		}
	}
	t.Logf("re-lock control: %d findings across %d shapes (5 must report, 5 must not)", len(entries), len(must)+len(mustNot))
}

// TestSameMutexRelockGateDetectsARelockAcrossACallEdge is PASS 2's negative control — the pass's own
// proof that it can see the shape it exists for. A detector that reports nothing because its index is
// empty looks EXACTLY like a clean repository, so this control drives the same registry derivation and
// the same edge resolver with a synthetic file of six shapes: TWO must be reported, FOUR must not.
func TestSameMutexRelockGateDetectsARelockAcrossACallEdge(t *testing.T) {
	const src = `package main

import "sync"

type Node struct {
	Mu    sync.RWMutex
	Other sync.RWMutex
}

// MUST REPORT 1 — the caller holds node.Mu and calls a method that takes node.Mu itself.
func (n *Node) badViaMethod() {
	n.Mu.Lock()
	defer n.Mu.Unlock()
	n.inner()
}

func (n *Node) inner() {
	n.Mu.RLock()
	defer n.Mu.RUnlock()
}

// MUST REPORT 2 — the same shape on a second mutex, reached through a READ lock.
func (n *Node) badViaOtherHandler() {
	n.Other.RLock()
	defer n.Other.RUnlock()
	n.readOther()
}

func (n *Node) readOther() {
	n.Other.RLock()
}

// MUST NOT 1 — a DIFFERENT object: the expression is the identity, not the field name.
func (a *Node) fineOtherObject(b *Node) {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	b.inner()
}

// MUST NOT 2 — a new goroutine does NOT inherit the lock, so its callee cannot deadlock on it.
func (n *Node) fineViaGoroutine() {
	n.Mu.Lock()
	defer n.Mu.Unlock()
	go n.inner()
}

// MUST NOT 3 — the call happens AFTER the lock is released.
func (n *Node) fineAfterRelease() {
	n.Mu.Lock()
	n.Mu.Unlock()
	n.inner()
}

// MUST NOT 4 — the callee takes a DIFFERENT mutex than the caller holds.
func (n *Node) fineDifferentField() {
	n.Mu.Lock()
	defer n.Mu.Unlock()
	n.readOther()
}
`
	entries, defined, ambiguous, err := measureRelocksAcrossOneCallEdge(
		[]mutexSource{{name: "relock_edge_synthetic.go", src: []byte(src)}},
		map[string]bool{"Mu": true, "Other": true})
	if err != nil {
		t.Fatalf("the control source must parse and measure: %v", err)
	}
	if defined == 0 {
		t.Fatal("the control derived ZERO method takes — the registry builder is blind, so its silence " +
			"about the real tree would mean nothing")
	}
	if ambiguous != 0 {
		t.Errorf("these method names are unambiguous by construction, but %d were skipped as ambiguous: "+
			"the ambiguity rule is too eager", ambiguous)
	}

	reported := map[string]bool{}
	for _, e := range entries {
		if e.via == "" {
			t.Errorf("pass 2 reported %q with NO witness edge: every finding of this pass must name the "+
				"callee it came through", e)
		}
		reported[e.key] = true
	}
	for _, name := range []string{"badViaMethod", "badViaOtherHandler"} {
		found := false
		for key := range reported {
			if strings.HasSuffix(key, "."+name) {
				found = true
			}
		}
		if !found {
			t.Errorf("pass 2 MISSED %s, which calls a method that takes the mutex it already holds — the "+
				"pass cannot see the shape it exists for (found: %v)", name, entries)
		}
	}
	for _, name := range []string{"fineOtherObject", "fineViaGoroutine", "fineAfterRelease", "fineDifferentField"} {
		for key := range reported {
			if strings.HasSuffix(key, "."+name) {
				t.Errorf("pass 2 reported %s, which does NOT re-lock across its call — a gate that reports "+
					"correct code gets switched off", key)
			}
		}
	}
	t.Logf("pass-2 control: %d method takes derived, %d findings (2 must report, 4 must not)",
		defined, len(entries))
}
