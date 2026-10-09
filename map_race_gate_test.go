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
// THE UNFENCED-MAP GATE — no lobby-owning function may touch a lobby-owned map
// where its OWN lock cannot possibly be held.
// ============================================================================
//
// WHY THIS EXISTS. Every map field of `Lobby` is shared mutable state: `l.leaderboard` alone is
// written by matchmaking, the career door and the economy tickers while HTTP request goroutines
// read it. Go's runtime turns a concurrent map access into `fatal error: concurrent map read and
// map write` / `concurrent map iteration and map write`, which is NOT recoverable and takes the
// WHOLE PROCESS down — every player, every WebSocket. It is not an error value a handler can
// return, and it cannot be caught by `recover`.
//
// THIS GATE WAS WRITTEN AFTER MEASURING, and its first run found FIVE REGISTERED ROUTES reading
// lobby maps with no lock at all (`/api/player/profile`, `/api/invest/portfolio`,
// `/api/invest/dividends/history`, `/api/contracts/list`, `/api/contracts/assign`) — two of which
// also WROTE. All five are fixed; the baseline below is what remains, and every entry states
// whether a lock is still OWED.
//
// WHAT IT MEASURES, in one line: within ONE function body, does the function touch a map field of
// `Lobby` when nothing in that body has acquired the lobby lock yet?
//
// THE PROTECTED SET IS DERIVED, NOT CURATED. It is every field of `type Lobby struct` whose type
// is a `map[…]…`, parsed from the struct itself, so a map added tomorrow is measured tomorrow. A
// curated list would go stale on the first new field, and a stale gate reads exactly like a clean
// one. If the derivation finds ZERO maps the gate FAILS, because a detector that matches nothing
// reports a clean repository for ever.
//
// THE ANALYSIS IS ONE-HOP AND POSITIONAL — stated, not implied:
//   - POSITIONAL. "No lock of its own" means the body acquires the lock NOWHERE (or only after the
//     access). That is sound — a lock can only be held if an earlier statement took it — and it
//     needs no control-flow modelling, so it has no branch-shaped false positives. Its mirror
//     image is a FALSE NEGATIVE (an access after a lock taken only inside an `if` is not reported),
//     which is the safe direction for a gate about an unfenced access.
//   - ONE HOP. A function is judged by its OWN body and its DIRECT callers. `...Locked` is honoured
//     as this tree's ONE spelling for "the caller holds it", so a helper called only from
//     `...Locked` bodies reads as CONTRACT. A helper reached through a longer chain stays
//     UNFENCED — the conservative default, because "prove it is safe" is the work.
//   - THE RACE DETECTOR IS NOT AVAILABLE HERE (`go test -race` requires cgo and this host has no C
//     toolchain), so this is a STATIC gate: a census of a shape, not a proof of a race.
//   - It says NOTHING about whether the locking that IS present is correct. The recursive-lock
//     class belongs to the sibling gate in `selflock_gate_test.go`, which owns it alone.
// ============================================================================

// protectedLobbyMaps reads the map-typed fields of `type Lobby struct`.
func protectedLobbyMaps(dir string) (map[string]bool, error) {
	src := filepath.Join(dir, "backend_types.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, src, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", src, err)
	}
	out := map[string]bool{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != "Lobby" {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			for _, f := range st.Fields.List {
				if _, isMap := f.Type.(*ast.MapType); isMap {
					for _, n := range f.Names {
						out[n.Name] = true
					}
				}
			}
		}
	}
	return out, nil
}

// lobbyMapSource is ONE file's source, keyed by its name, so the detector can be driven by a
// synthetic file as well as by the real tree.
type lobbyMapSource struct {
	name string
	src  []byte
}

// unfencedMapEntry is one function that touches a protected map with no lock of its own.
type unfencedMapEntry struct {
	key       string // "Type.Method" or "FuncName"
	fields    []string
	callers   []string
	exported  bool
	firstLine int
}

// measureUnfencedLobbyMaps reports every lobby-owning function that touches a protected map where
// its own body cannot hold the lock. It takes SOURCES rather than a directory so the controls below
// can drive it with synthetic code, and it returns the lock-acquisition census BESIDE the entries so
// the verdicts can be computed from the same measurement.
func measureUnfencedLobbyMaps(sources []lobbyMapSource, protected map[string]bool) ([]unfencedMapEntry, map[string]bool, error) {
	type info struct {
		key      string
		hasLock  bool
		fields   map[string]bool
		callees  map[string]bool
		callers  []string
		exported bool
		line     int
	}
	infos := map[string]*info{}

	for _, s := range sources {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, s.name, s.src, 0)
		if err != nil {
			return nil, nil, fmt.Errorf("parse %s: %w", s.name, err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			names := lobbyTypedIdents(fn)
			if len(names) == 0 {
				continue
			}
			key := funcDeclKey(fn)
			e := &info{key: key, fields: map[string]bool{}, callees: map[string]bool{},
				exported: ast.IsExported(fn.Name.Name), line: fset.Position(fn.Pos()).Line}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.CallExpr:
					if sel, isSel := x.Fun.(*ast.SelectorExpr); isSel {
						if inner, isInner := sel.X.(*ast.SelectorExpr); isInner && inner.Sel.Name == "mutex" {
							if base, isIdent := inner.X.(*ast.Ident); isIdent && names[base.Name] &&
								(sel.Sel.Name == "Lock" || sel.Sel.Name == "RLock") {
								e.hasLock = true
							}
						}
						if base, isIdent := sel.X.(*ast.Ident); isIdent && names[base.Name] {
							e.callees["Lobby."+sel.Sel.Name] = true
							e.callees[sel.Sel.Name] = true
						} else if root := rootIdentLocal(sel.X); root != nil && names[root.Name] {
							e.callees[sel.Sel.Name] = true
						}
					} else if id, isIdent := x.Fun.(*ast.Ident); isIdent {
						e.callees[id.Name] = true
					}
				case *ast.SelectorExpr:
					if base, isIdent := x.X.(*ast.Ident); isIdent && names[base.Name] && protected[x.Sel.Name] {
						e.fields[x.Sel.Name] = true
					}
				}
				return true
			})
			infos[key] = e
		}
	}

	hasLock := map[string]bool{}
	for _, e := range infos {
		if e.hasLock {
			hasLock[e.key] = true
		}
	}
	for _, e := range infos {
		for callee := range e.callees {
			if c, ok := infos[callee]; ok {
				c.callers = append(c.callers, e.key)
			}
		}
	}

	var out []unfencedMapEntry
	for key, e := range infos {
		if len(e.fields) == 0 || e.hasLock || strings.HasSuffix(key, "Locked") {
			continue
		}
		fields := make([]string, 0, len(e.fields))
		for f := range e.fields {
			fields = append(fields, f)
		}
		sort.Strings(fields)
		// A call may be recorded under more than one candidate key (`Lobby.X` and `X`), so the
		// caller list is de-duplicated before it is published — a caller named twice reads like two
		// callers and would make the evidence look stronger than it is.
		seen := map[string]bool{}
		callers := make([]string, 0, len(e.callers))
		for _, c := range e.callers {
			if !seen[c] {
				seen[c] = true
				callers = append(callers, c)
			}
		}
		sort.Strings(callers)
		out = append(out, unfencedMapEntry{key: key, fields: fields, callers: callers,
			exported: e.exported, firstLine: e.line})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].key < out[j].key })
	return out, hasLock, nil
}

// entryHolds propagates "this body is entered with the lobby lock held" to a FIXPOINT over the
// caller graph, so a helper reached ONLY through lock-holding helpers is not FALSELY reported as
// owing a lock. The conservative direction is preserved: a body with no caller, or with ANY caller
// that cannot be proved to hold, stays UNFENCED. A cycle proves nothing, so it credits nothing.
func entryHolds(key string, byKey map[string]unfencedMapEntry, hasLock, memo, visiting map[string]bool) bool {
	if v, done := memo[key]; done {
		return v
	}
	if visiting[key] {
		return false
	}
	e, isEntry := byKey[key]
	if !isEntry {
		// Its body is not an entry, so it either acquires the lock itself or declares the contract.
		return hasLock[key] || strings.HasSuffix(key, "Locked")
	}
	visiting[key] = true
	ok := len(e.callers) > 0
	for _, c := range e.callers {
		if !ok {
			break
		}
		if !entryHolds(c, byKey, hasLock, memo, visiting) {
			ok = false
		}
	}
	delete(visiting, key)
	memo[key] = ok
	return ok
}

// whyUnfenced states the mechanism for ONE entry and the verdict it implies. UNFENCED is the
// default: an entry leaves the baseline by being FIXED, never by being re-worded. The CONTRACT
// verdict is TRANSITIVE (see entryHolds), because "every caller holds" is a property of the whole
// caller chain, not of the one hop this measurement happens to see.
func whyUnfenced(e unfencedMapEntry, hasLock map[string]bool, byKey map[string]unfencedMapEntry) string {
	switch {
	case len(e.callers) == 0:
		return "UNFENCED — no caller in the tree (dead, boot-time, or a public door)"
	case e.exported:
		return "UNFENCED — exported door: its callers are unconstrained"
	}
	memo, visiting := map[string]bool{}, map[string]bool{}
	var unlocked []string
	for _, c := range e.callers {
		if !entryHolds(c, byKey, hasLock, memo, visiting) {
			unlocked = append(unlocked, c)
		}
	}
	if len(unlocked) == 0 {
		return "CONTRACT — every caller holds the lobby lock, transitively"
	}
	return "UNFENCED — reached from bodies that do not hold the lock: " + strings.Join(unlocked, ", ")
}

// entriesByKey indexes the measurement so the verdict can walk the caller chain.
func entriesByKey(entries []unfencedMapEntry) map[string]unfencedMapEntry {
	out := make(map[string]unfencedMapEntry, len(entries))
	for _, e := range entries {
		out[e.key] = e
	}
	return out
}

// ============================================================================
// THE BASELINE — a WORKLIST, not an excuse.
// ============================================================================
//
// Every entry was MEASURED on this tree. The verdict is PINNED, so the gate fails on THREE things
// and not merely on a new name:
//
//	NEW     — a function that is unfenced and appears in no line of this baseline
//	STALE   — a baselined function that is no longer unfenced. Its fix must DELETE this line, so the
//	          baseline can never quietly legitimise rot (the `verify:reachability` discipline).
//	FLIPPED — a CONTRACT entry that has become UNFENCED, i.e. somebody gave a lock-holding helper a
//	          caller that holds nothing.
//
// CONTRACT means EVERY caller either acquires the lobby lock or declares it with the `Locked`
// suffix. UNFENCED means a lock is still OWED: the function has no caller at all (dead, boot-time,
// or a future door), or it is exported so its callers are unconstrained, or a caller was measured
// holding nothing. UNFENCED is the default because "prove it is safe" is the work.
var unfencedLobbyMapBaseline = map[string]string{
	"AchievementService.TransferBundleItems":     "UNFENCED",
	"AuctionService.TransferBundleItems":         "UNFENCED",
	"ContractEngine.HandleCompleteContract":      "UNFENCED",
	"EntityEventEngine.ProcessEntityEvent":       "UNFENCED",
	"EntityEventEngine.ResolveEntityEventPayout": "UNFENCED",
	"EntityEventEngine.ownerWalletOf":            "UNFENCED",
	"EntityMarket.CreateListing":                 "UNFENCED",
	"EntityMarket.PurchaseListing":               "UNFENCED",
	"LoanService.TransferBundleItems":            "UNFENCED",
	"Lobby.CalculateReputation":                  "UNFENCED",
	// REMOVED 2026-09-16: `Lobby.CalculateTotalPortfolioValue` now takes the lobby read lock itself
	// and delegates to `CalculateTotalPortfolioValueLocked`, so it is fenced. Its three lock-holding
	// callers moved to the `...Locked` form, which is what keeps the pair honest.
	"Lobby.InitializeEntityInvestmentSystem":    "UNFENCED",
	"Lobby.applyCyberAudit":                     "CONTRACT",
	"Lobby.applyDistrictScanner":                "CONTRACT",
	"Lobby.applyItemEffectToMatch":              "CONTRACT",
	"Lobby.ensurePlayerStatsMapsInitialized":    "UNFENCED",
	"Lobby.getClubByTerritoryID":                "UNFENCED",
	"Lobby.handleAdoptEntity":                   "UNFENCED",
	"Lobby.handleCombineOwnerStats":             "UNFENCED",
	"Lobby.handleFaithReligionRitual":           "UNFENCED",
	"Lobby.handleFaithWarGambit":                "UNFENCED",
	"Lobby.handleReclaimEntity":                 "UNFENCED",
	"Lobby.processItemBuffExpiration":           "CONTRACT",
	"PetBattleArena.ChallengePet":               "UNFENCED",
	"PetBattleArena.ResolveBattle":              "UNFENCED",
	"ReligionGovernance.BuyGovernorship":        "UNFENCED",
	"ReligionGovernance.BuyoutReligion":         "UNFENCED",
	"TournamentService.DetermineTop5":           "UNFENCED",
	"TournamentService.ProcessTournamentResult": "UNFENCED",
	"Lobby.applyItemEffect":                     "CONTRACT",
	"Lobby.findRarestCardInInventory":           "CONTRACT",
	"Lobby.incrementDNF":                        "CONTRACT",
	"Lobby.initiatePairedMatch":                 "CONTRACT",
	"Lobby.resolveCareerCaller":                 "CONTRACT",
	"Lobby.serverCheckCaptures":                 "CONTRACT",
	"Lobby.updateLeaderboard":                   "CONTRACT",
	"Lobby.verifyWinner":                        "CONTRACT",
	// Found by THIS gate's own derivation, which reads all 47 map fields of `Lobby` rather than the
	// hand-written subset the first measurement used: these five touch a map the earlier census did
	// not know was protected (`inventory`, `persistentCardCache`, `lastActive`, `onboardedWallets`,
	// `availableNetworks`).
	"Lobby.applyCapturePenalty":     "CONTRACT",
	"Lobby.handleOrphanStatus":      "UNFENCED",
	"Lobby.initiateSuddenDeath":     "CONTRACT",
	"Lobby.saveOnboardedWallets":    "CONTRACT",
	"Lobby.savePersistentCardCache": "CONTRACT",
	// TournamentService.RecordTournamentOnChain was baselined UNFENCED and the line was
	// DELETED on 2026-09-19 (Problems.md §39): the tournament archive no longer reads
	// `l.availableNetworks` — a protected map — while holding no lock. It hands its payload
	// to the ONE record writer, which needs no config. The gate FAILED on the stale line and
	// was right to: a baseline that outlives its reason legitimises rot.
}

// realLobbyMapSources reads every non-test Go file in the package directory.
func realLobbyMapSources(t *testing.T) []lobbyMapSource {
	t.Helper()
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob sources: %v", err)
	}
	var out []lobbyMapSource
	for _, p := range paths {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		src, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		out = append(out, lobbyMapSource{name: p, src: src})
	}
	if len(out) == 0 {
		t.Fatal("no non-test sources found — the gate would pass vacuously")
	}
	return out
}

// verdictOf reduces a computed reason to the verdict the baseline pins.
func verdictOf(reason string) string {
	if strings.HasPrefix(reason, "CONTRACT") {
		return "CONTRACT"
	}
	return "UNFENCED"
}

// TestNoNewUnfencedLobbyMapAccess IS the gate.
func TestNoNewUnfencedLobbyMapAccess(t *testing.T) {
	protected, err := protectedLobbyMaps(".")
	if err != nil {
		t.Fatalf("the protected map set must be derived from `type Lobby struct`: %v", err)
	}
	if len(protected) == 0 {
		t.Fatal("the derivation found ZERO map fields on Lobby — a detector that matches nothing would " +
			"report a clean repository for ever")
	}

	entries, hasLock, err := measureUnfencedLobbyMaps(realLobbyMapSources(t), protected)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("the measurement returned NO unfenced entries at all, which cannot be true of this tree " +
			"(and would make the baseline below read as entirely stale) — the detector is broken")
	}

	seen := map[string]bool{}
	byKey := entriesByKey(entries)
	var newOnes, flipped, report []string
	for _, e := range entries {
		reason := whyUnfenced(e, hasLock, byKey)
		verdict := verdictOf(reason)
		seen[e.key] = true
		report = append(report, fmt.Sprintf("%s  [%s]  %s", e.key, verdict, reason))
		base, baselined := unfencedLobbyMapBaseline[e.key]
		switch {
		case !baselined:
			newOnes = append(newOnes, fmt.Sprintf("%s — %s (fields: %s)",
				e.key, reason, strings.Join(e.fields, ",")))
		case base != verdict:
			flipped = append(flipped, fmt.Sprintf("%s: baselined %s, measured %s — %s",
				e.key, base, verdict, reason))
		}
	}
	var stale []string
	for key := range unfencedLobbyMapBaseline {
		if !seen[key] {
			stale = append(stale, key)
		}
	}
	sort.Strings(newOnes)
	sort.Strings(flipped)
	sort.Strings(stale)

	t.Logf("unfenced-map gate: %d protected maps derived from Lobby; %d functions touch one with no lock "+
		"of their own; %d baselined", len(protected), len(entries), len(unfencedLobbyMapBaseline))

	if len(newOnes) > 0 {
		t.Errorf("NEW unfenced lobby-map access. A concurrent map access is `fatal error: concurrent map "+
			"read and map write` — NOT recoverable, and it takes the whole process down (every player, every "+
			"WebSocket), not merely the request. Take the lobby lock around the access, or add a `...Locked` "+
			"sibling and call it from a body that already holds it:\n  %s", strings.Join(newOnes, "\n  "))
	}
	if len(flipped) > 0 {
		t.Errorf("a baselined CONTRACT entry is now UNFENCED — a helper that used to be reached only from "+
			"lock-holding bodies has gained a caller that holds nothing:\n  %s", strings.Join(flipped, "\n  "))
	}
	if len(stale) > 0 {
		t.Errorf("STALE baseline entries: these no longer touch a protected map without a lock of their "+
			"own. That is a FIX, so DELETE its line — a baseline that outlives its reason reads like a clean "+
			"repository while legitimising rot:\n  %s", strings.Join(stale, "\n  "))
	}
	if len(newOnes) > 0 || len(flipped) > 0 || len(stale) > 0 {
		t.Logf("census of this run:\n  %s", strings.Join(report, "\n  "))
	}
}

// TestUnfencedMapGateDetectsAViolation is the gate's NEGATIVE CONTROL. The detector takes SOURCES,
// so it can be driven with synthetic code — without this, a detector that silently matched nothing
// would report a clean repository for ever, which is exactly how a gate rots.
func TestUnfencedMapGateDetectsAViolation(t *testing.T) {
	const src = `package main

type Other struct {
	mutex       int
	leaderboard map[string]int
}

// ---- MUST REPORT -----------------------------------------------------------------

// A read with no lock at all: the shape this gate exists for.
func (l *Lobby) badUnlockedRead(w string) {
	_ = l.leaderboard[w]
}

// A RANGE over a concurrent map is the one that reports as "concurrent map iteration and map write".
func (l *Lobby) badUnlockedRange() int {
	n := 0
	for _, v := range l.playerBalances {
		n += int(v)
	}
	return n
}

// A function taking a *Lobby PARAMETER is as much a lobby body as a method.
func freeFnBad(l *Lobby) int {
	return len(l.clubs)
}

// An EXPORTED door reads a protected map: its callers are unconstrained, so it stays UNFENCED even
// when every caller it happens to have HOLDS the lock.
func (l *Lobby) BadExportedDoor(w string) {
	_ = l.leaderboard[w]
}

func (l *Lobby) holderForDoorLocked() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.BadExportedDoor("x")
}

// A CHAIN: the parent takes no lock, so the child inherits that and is UNFENCED too.
func (l *Lobby) badChainParent() {
	_ = l.leaderboard["x"]
}

func (l *Lobby) badChainChild() {
	l.badChainParent()
}

// ---- MUST NOT REPORT -------------------------------------------------------------

func (l *Lobby) fineLockedRead(w string) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	_ = l.leaderboard[w]
}

func (l *Lobby) fineNamedContractLocked(w string) {
	_ = l.leaderboard[w]
}

func (l *Lobby) fineNoMap() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
}

// NOT A CANDIDATE AT ALL: the receiver is not a *Lobby, so its mutex and its map are its own.
func (o *Other) fineOtherMutex() {
	_ = o.mutex
	_ = o.leaderboard["x"]
}

// ---- THE CONTRACT CASE -----------------------------------------------------------

// Reached ONLY from the lock-holding sibling, so it is CONTRACT and not an owed lock.
func (l *Lobby) contractHelper() {
	_ = l.leaderboard["x"]
}

func (l *Lobby) holderLocked() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.contractHelper()
}
`
	protected := map[string]bool{"leaderboard": true, "playerBalances": true, "clubs": true}
	entries, hasLock, err := measureUnfencedLobbyMaps(
		[]lobbyMapSource{{name: "synthetic.go", src: []byte(src)}}, protected)
	if err != nil {
		t.Fatalf("the detector must parse a well-formed file: %v", err)
	}

	verdicts := map[string]string{}
	reasons := map[string]string{}
	byKey := entriesByKey(entries)
	report := ""
	for _, e := range entries {
		reason := whyUnfenced(e, hasLock, byKey)
		verdicts[e.key] = verdictOf(reason)
		reasons[e.key] = reason
		report += "\n  " + e.key + " [" + verdictOf(reason) + "] " + reason
	}

	for _, want := range []string{
		"Lobby.badUnlockedRead", "Lobby.badUnlockedRange", "freeFnBad",
		"Lobby.BadExportedDoor", "Lobby.badChainParent",
		"Lobby.contractHelper",
	} {
		if _, ok := verdicts[want]; !ok {
			t.Errorf("the detector MISSED %s, so it cannot see the shape it exists for:%s", want, report)
		}
	}
	for _, mustNot := range []string{
		"Lobby.fineLockedRead",          // its own RLock is taken first
		"Lobby.fineNamedContractLocked", // the NAME declares that the caller holds it
		"Lobby.fineNoMap",               // touches no protected map
		"Other.fineOtherMutex",          // not a *Lobby body: its mutex and map are its own
	} {
		if _, ok := verdicts[mustNot]; ok {
			t.Errorf("the detector reported %s, which is correct code — a gate that cries wolf gets switched off:%s",
				mustNot, report)
		}
	}

	// The CLAUSES must be distinguishable, or the baseline's verdicts mean nothing.
	if got := verdicts["Lobby.contractHelper"]; got != "CONTRACT" {
		t.Errorf("a helper reached ONLY from a `...Locked` body must read as CONTRACT; got %q:%s", got, report)
	}
	if got := verdicts["Lobby.BadExportedDoor"]; got != "UNFENCED" {
		t.Errorf("an exported door must stay UNFENCED whatever its current callers do; got %q:%s", got, report)
	}
	// A body that touches NO protected map is NOT an entry — only the body that touches it is — and
	// the parent's evidence must NAME the unlocked caller that made it UNFENCED. (This assertion's
	// first draft expected the CHILD to be reported, which the control itself caught: the detector
	// reports the ACCESS, not the reachability, and the reachability travels in the evidence.)
	if _, ok := verdicts["Lobby.badChainChild"]; ok {
		t.Errorf("a body that touches no protected map must not be an entry — only the body that touches "+
			"it is:%s", report)
	}
	if r := reasons["Lobby.badChainParent"]; !strings.Contains(r, "Lobby.badChainChild") {
		t.Errorf("the parent's evidence must NAME the unlocked caller that made it UNFENCED; got %q:%s", r, report)
	}

	// And it must be able to report NOTHING: a file with no lobby maps at all.
	clean, _, err := measureUnfencedLobbyMaps([]lobbyMapSource{
		{name: "clean.go", src: []byte("package main\n\nfunc (l *Lobby) ok() {}\n")}}, protected)
	if err != nil {
		t.Fatalf("parse clean.go: %v", err)
	}
	if len(clean) != 0 {
		t.Errorf("a clean file reported %d entries: %v", len(clean), clean)
	}
}
