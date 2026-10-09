//go:build !js && !wasm

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
)

// TestNoteVocabularyIsWellFormed re-runs the init assertion explicitly, so a
// failure names the vocabulary instead of appearing as "panic in init".
func TestNoteVocabularyIsWellFormed(t *testing.T) {
	assertNoteVocabularyPolicy()
	if len(noteVocabulary) == 0 {
		t.Fatal("the catalogue is empty")
	}
}

// TestNotePrefixesAreUnique — two purposes sharing one prefix would let a
// payment made for one satisfy the other.
func TestNotePrefixesAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range NotePrefixes() {
		if seen[p] {
			t.Errorf("prefix %q is declared twice", p)
		}
		seen[p] = true
	}
}

// TestNotePrefixAmbiguitiesAreExactlyTheAcknowledgedSet pins the ambiguity set.
// It is EMPTY today, and that is the strong statement: no declared prefix is a
// proper prefix of another, so a reader keyed on one purpose's prefix can never
// match a different purpose's note. Adding a prefix that breaks this fails init
// (and this test) instead of being discovered in production.
func TestNotePrefixAmbiguitiesAreExactlyTheAcknowledgedSet(t *testing.T) {
	got := AmbiguousNotePrefixPairs()
	if len(got) != len(notePrefixAmbiguitiesAcknowledged) {
		t.Fatalf("ambiguous prefix pairs = %v; acknowledged = %v", got, notePrefixAmbiguitiesAcknowledged)
	}
	for i, pair := range got {
		want := notePrefixAmbiguitiesAcknowledged[i]
		if pair[0] != want[0] || pair[1] != want[1] {
			t.Errorf("ambiguous pair %d = %v; want %v", i, pair, want)
		}
	}
	// A stale acknowledgement would mask a future ambiguity behind a count that
	// no longer matches, so every acknowledged pair must still BE a pair.
	for _, want := range notePrefixAmbiguitiesAcknowledged {
		if !strings.HasPrefix(want[1], want[0]) {
			t.Errorf("the acknowledged pair (%q, %q) is not a prefix pair; remove the acknowledgement", want[0], want[1])
		}
	}
}

// TestDeclaredPrefixesAreExactMatchOnly — IsDeclaredNotePrefix must be an exact
// test, never a prefix-of test.
func TestDeclaredPrefixesAreExactMatchOnly(t *testing.T) {
	for _, p := range NotePrefixes() {
		if !IsDeclaredNotePrefix(p) {
			t.Errorf("%q is not exact-match resolvable", p)
		}
	}
	if IsDeclaredNotePrefix(NotePrefixOnboard + "SOMETHING_ELSE") {
		t.Error("a longer note must not satisfy the shorter declaration")
	}
	if IsDeclaredNotePrefix(strings.TrimSuffix(NotePrefixWin, ":")) {
		t.Error("a prefix missing its ':' must not be declared")
	}
	if IsDeclaredNotePrefix("") {
		t.Error("the empty string must never be a declared prefix (HasPrefix(x, \"\") is always true)")
	}
}

// TestBoundPrefixBuildersBindTheirTarget — a bound prefix must carry its target,
// so a payment cannot be replayed against a different club/district/tournament.
func TestBoundPrefixBuildersBindTheirTarget(t *testing.T) {
	type c struct{ name, got, base string }
	for _, tc := range []c{
		{"found_club", NoteFoundClub("MYCLUB"), NotePrefixFoundClub},
		{"join_club", NoteJoinClub("CLUB-1"), NotePrefixJoinClub},
		{"claim_district", NoteClaimDistrict("CLUB-1", "arena_center"), NotePrefixClaimDistrict},
		{"tournament_buyin", NoteTournamentBuyIn("TOURN-9"), NotePrefixTournamentBuyIn},
		{"arena_tournament_buyin", NoteArenaTournamentBuyIn("TOURN-9"), NotePrefixArenaTournamentBuyIn},
	} {
		if !strings.HasPrefix(tc.got, tc.base) {
			t.Errorf("%s: %q does not start with the declared base %q", tc.name, tc.got, tc.base)
		}
		if len(tc.got) <= len(tc.base) {
			t.Errorf("%s: %q carries no bound part", tc.name, tc.got)
		}
		if !IsMoneyDoorPurpose(tc.base) {
			t.Errorf("%s: %q is not registered as a money door", tc.name, tc.base)
		}
	}

	// Distinct targets MUST produce distinct prefixes.
	if NoteJoinClub("A") == NoteJoinClub("B") {
		t.Error("join_club does not bind its club id: two clubs produce the same prefix")
	}
	if NoteClaimDistrict("A", "d1") == NoteClaimDistrict("B", "d1") {
		t.Error("claim_district does not bind its club id")
	}
	if NoteClaimDistrict("A", "d1") == NoteClaimDistrict("A", "d2") {
		t.Error("claim_district does not bind its district")
	}

	// An EMPTY bound part returns "" so the door fails closed at the boundary.
	for _, got := range []string{
		NoteFoundClub(""), NoteJoinClub(""), NoteClaimDistrict("", "d"),
		NoteClaimDistrict("c", ""), NoteTournamentBuyIn(""),
	} {
		if got != "" {
			t.Errorf("an empty bound part must return \"\", got %q", got)
		}
	}

	// An undeclared base PANICS: a first-party wiring error, unreachable from a
	// request. Proving it here keeps the guard honest.
	func() {
		defer func() {
			if recover() == nil {
				t.Error("notePrefixBuild did not panic on an undeclared base prefix")
			}
		}()
		_ = notePrefixBuild("NOT_A_DECLARED_PREFIX:", "x")
	}()
}

// ---------------------------------------------------------------------------
// PARITY: the source must USE the vocabulary, never re-declare it
// ---------------------------------------------------------------------------

// TestEveryVerifyCallSiteNamesADeclaredPurpose is the money-door parity test.
// Every VerifyBuyInTransaction call must pass either a declared constant by
// name, or a variable THIS FILE builds from the vocabulary. A re-typed literal
// is exactly the drift that fails verification silently.
func TestEveryVerifyCallSiteNamesADeclaredPurpose(t *testing.T) {
	files := firstPartyGoSources(t, "note_vocabulary.go", "note_vocabulary_test.go")
	consts := declaredNotePrefixConstNames()
	found := 0

	for name, src := range files {
		for i, line := range strings.Split(src, "\n") {
			if !strings.Contains(line, "VerifyBuyInTransaction(") {
				continue
			}
			if strings.Contains(line, "func ") || strings.Contains(line, "//") {
				continue // the declaration, or a comment about it
			}
			idx := strings.Index(line, "VerifyBuyInTransaction(")
			rest := line[idx+len("VerifyBuyInTransaction("):]
			end := strings.LastIndex(rest, ")")
			if end < 0 {
				t.Errorf("%s:%d: cannot find the end of the call: %s", name, i+1, strings.TrimSpace(line))
				continue
			}
			args := splitTopLevelArgs(rest[:end])
			if len(args) < 8 {
				t.Errorf("%s:%d: expected 8 arguments, got %d", name, i+1, len(args))
				continue
			}
			last := strings.TrimSpace(args[len(args)-1])
			found++

			if strings.HasPrefix(last, "\"") {
				t.Errorf("%s:%d: the purpose prefix is a LITERAL (%s) - it must come from note_vocabulary.go", name, i+1, last)
				continue
			}
			if _, isConst := consts[last]; isConst {
				continue
			}
			if !fileBuildsPrefixFromVocabulary(src, last) {
				t.Errorf("%s:%d: %q is neither a declared NotePrefix constant nor a variable built from the vocabulary", name, i+1, last)
			}
		}
	}

	if found < 8 {
		t.Fatalf("found only %d VerifyBuyInTransaction call sites; the scan expects at least 8 (is the scan stale?)", found)
	}
}

// TestNoNotePrefixLiteralOutsideTheOwner enforces the single-owner rule
// mechanically: no Go file other than note_vocabulary.go may spell a prefix.
func TestNoNotePrefixLiteralOutsideTheOwner(t *testing.T) {
	owner, err := os.ReadFile("note_vocabulary.go")
	if err != nil {
		t.Fatalf("read note_vocabulary.go: %v", err)
	}
	// Prove the scan is really reading source before trusting a clean result.
	if !strings.Contains(string(owner), "\""+NotePrefixWin) {
		t.Fatal("the scan cannot see note_vocabulary.go's literals; the check is meaningless")
	}

	files := firstPartyGoSources(t, "note_vocabulary.go", "note_vocabulary_test.go")
	for name, src := range files {
		for _, p := range NotePrefixes() {
			if strings.Contains(src, "\""+p) {
				t.Errorf("%s: declares the note prefix %q as a literal; note_vocabulary.go is its only owner", name, p)
			}
		}
	}
}

// TestClientDeclaresNoNotePrefix — the browser must FETCH the prefixes it may
// write, never carry its own copy.
func TestClientDeclaresNoNotePrefix(t *testing.T) {
	moneyDoors := notePrefixesByScope(NoteScopeMoneyDoor)
	if len(moneyDoors) < 8 {
		t.Fatalf("only %d money-door purposes declared; expected at least 8", len(moneyDoors))
	}

	dir := filepath.Join("Public", "js")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	scanned := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".js") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		scanned++
		for _, p := range moneyDoors {
			if strings.Contains(string(data), p) {
				t.Errorf("Public/js/%s contains the note prefix %q; the client must read it from /api/notes/vocabulary", e.Name(), p)
			}
		}
	}
	if scanned == 0 {
		t.Fatal("no client JS was scanned")
	}
}

// TestEveryDeclaredConstantIsCatalogued — a constant that is not in the served
// catalogue is drift: the client could never learn it.
func TestEveryDeclaredConstantIsCatalogued(t *testing.T) {
	consts := declaredNotePrefixConstNames()
	if len(consts) == 0 {
		t.Fatal("no NotePrefix constants were found (is the regex stale?)")
	}
	catalogued := map[string]bool{}
	for _, p := range noteVocabulary {
		catalogued[p.Prefix] = true
	}
	for name, prefix := range consts {
		if !catalogued[prefix] {
			t.Errorf("constant %s (%q) is NOT in the served catalogue", name, prefix)
		}
	}
	if len(catalogued) != len(consts) {
		t.Errorf("the catalogue holds %d prefixes but %d constants are declared", len(catalogued), len(consts))
	}
}

// TestServedVocabularyPayloadShape — the payload is the client's only view.
func TestServedVocabularyPayloadShape(t *testing.T) {
	payload := NoteVocabularyPayload()

	purposes, ok := payload["purposes"].([]map[string]string)
	if !ok || len(purposes) == 0 {
		t.Fatalf("payload has no purposes: %T", payload["purposes"])
	}
	for _, p := range purposes {
		for _, field := range []string{"key", "prefix", "scope", "direction", "description"} {
			if p[field] == "" {
				t.Errorf("purpose %v is missing %q", p, field)
			}
		}
	}

	counts, ok := payload["counts"].(map[string]int)
	if !ok {
		t.Fatalf("payload has no counts: %T", payload["counts"])
	}
	if counts["total"] != len(purposes) {
		t.Errorf("counts.total = %d but %d purposes were served", counts["total"], len(purposes))
	}
	if counts["money_door"] != len(notePrefixesByScope(NoteScopeMoneyDoor)) {
		t.Errorf("counts.money_door = %d but %d are declared", counts["money_door"], len(notePrefixesByScope(NoteScopeMoneyDoor)))
	}
	if counts["money_door"]+counts["checkpoint"]+counts["audit_log"] != counts["total"] {
		t.Errorf("the scope counts do not sum to the total: %v", counts)
	}

	rules, ok := payload["rules"].(map[string]interface{})
	if !ok {
		t.Fatalf("payload has no rules: %T", payload["rules"])
	}
	if rules["client_may_name_money_door_only"] != true {
		t.Error("the payload must state that a client may name money-door purposes only")
	}
	if rules["empty_prefix_is_refused"] != true {
		t.Error("the payload must state that an empty prefix is refused")
	}
}

// TestNoteVocabularyRouteIsRegisteredInBothServers — this repo has TWO servers
// (server_main.go and console_server.go) and a route added to only one of them
// is a real, repeated defect class here. The HTTP boundary is also exercised, so
// "registered" and "actually serves the payload" are both proven.
func TestNoteVocabularyRouteIsRegisteredInBothServers(t *testing.T) {
	for _, f := range []string{"server_main.go", "console_server.go"} {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		if !strings.Contains(string(data), "\"/api/notes/vocabulary\"") {
			t.Errorf("%s does not register /api/notes/vocabulary", f)
		}
	}

	// GET must serve the client's only view of the vocabulary.
	rec := httptest.NewRecorder()
	handleNoteVocabulary(rec, httptest.NewRequest(http.MethodGet, "/api/notes/vocabulary", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d; want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q; want JSON", ct)
	}

	var served struct {
		Success  bool `json:"success"`
		Purposes []struct {
			Key    string `json:"key"`
			Prefix string `json:"prefix"`
			Scope  string `json:"scope"`
		} `json:"purposes"`
		Counts map[string]int `json:"counts"`
		Rules  map[string]any `json:"rules"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &served); err != nil {
		t.Fatalf("the served payload is not the expected JSON: %v", err)
	}
	if !served.Success || len(served.Purposes) == 0 {
		t.Fatalf("served payload is empty: %s", rec.Body.String())
	}
	if served.Counts["money_door"] != len(notePrefixesByScope(NoteScopeMoneyDoor)) {
		t.Errorf("served money_door count = %d; want %d", served.Counts["money_door"], len(notePrefixesByScope(NoteScopeMoneyDoor)))
	}
	// Every money door the CLIENT is told about must be one this server declares.
	moneyDoors := map[string]bool{}
	for _, p := range notePrefixesByScope(NoteScopeMoneyDoor) {
		moneyDoors[p] = true
	}
	for _, p := range served.Purposes {
		if p.Scope == NoteScopeMoneyDoor && !moneyDoors[p.Prefix] {
			t.Errorf("the server served money door %q which it does not declare", p.Prefix)
		}
		if !IsDeclaredNotePrefix(p.Prefix) {
			t.Errorf("the server served undeclared prefix %q", p.Prefix)
		}
	}
	if served.Rules["client_may_name_money_door_only"] != true {
		t.Error("the served rules must state that a client may name money-door purposes only")
	}

	// A write must be refused: this is a read-only vocabulary.
	recPost := httptest.NewRecorder()
	handleNoteVocabulary(recPost, httptest.NewRequest(http.MethodPost, "/api/notes/vocabulary", nil))
	if recPost.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST status = %d; want %d", recPost.Code, http.StatusMethodNotAllowed)
	}
}

// TestOracleRefusesAnEmptyPurposePrefix — the boundary that makes the binding
// rule unbypassable. It must refuse BEFORE any network work.
func TestOracleRefusesAnEmptyPurposePrefix(t *testing.T) {
	s := &OracleService{}
	l := &Lobby{availableNetworks: map[string]NetworkConfig{}}

	if ok, _, err := s.VerifyBuyInTransaction(l, "Voi", "TX", 1, "1", "0xa", "VAULT", ""); ok || err == nil {
		t.Error("an EMPTY purpose prefix must be refused (HasPrefix(x, \"\") is always true)")
	}
	if ok, _, err := s.VerifyBuyInTransaction(l, "Voi", "", 1, "1", "0xa", "VAULT", NotePrefixWin); ok || err == nil {
		t.Error("an EMPTY transaction id must be refused (it cannot be memoised)")
	}
	if ok, _, err := s.VerifyBuyInTransaction(l, "Voi", "   ", 1, "1", "0xa", "VAULT", NotePrefixWin); ok || err == nil {
		t.Error("a blank transaction id must be refused")
	}
	// A well-formed request for an unknown network is refused by the config
	// lookup, which proves the guards did not swallow legitimate calls.
	if ok, _, err := s.VerifyBuyInTransaction(l, "Voi", "TX", 1, "1", "0xa", "VAULT", NotePrefixWin); ok || err == nil {
		t.Error("an unknown network must be refused")
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func firstPartyGoSources(t *testing.T, skip ...string) map[string]string {
	t.Helper()
	skipSet := map[string]bool{}
	for _, s := range skip {
		skipSet[s] = true
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read repo root: %v", err)
	}
	out := map[string]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || skipSet[e.Name()] {
			continue
		}
		data, err := os.ReadFile(e.Name())
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		out[e.Name()] = string(data)
	}
	if len(out) == 0 {
		t.Fatal("no Go sources were found")
	}
	return out
}

// splitTopLevelArgs splits a call's argument list on commas that are NOT inside
// brackets, so uint64(5000*divisor) stays one argument.
func splitTopLevelArgs(s string) []string {
	var out []string
	depth := 0
	start := 0
	for i, r := range s {
		switch r {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, s[start:i])
				start = i + 1
			}
		}
	}
	if strings.TrimSpace(s[start:]) != "" {
		out = append(out, s[start:])
	}
	return out
}

var notePrefixConstRe = regexp.MustCompile(`(?m)^\s*(NotePrefix[A-Za-z0-9]*)\s*=\s*"([^"]*)"`)

var (
	notePrefixConstNamesOnce sync.Once
	notePrefixConstNames     map[string]string
)

// declaredNotePrefixConstNames reads the constant names and literal values from
// the OWNER file, so "the constant exists" is proven from source rather than
// assumed from a hand-maintained list.
func declaredNotePrefixConstNames() map[string]string {
	notePrefixConstNamesOnce.Do(func() {
		notePrefixConstNames = map[string]string{}
		data, err := os.ReadFile("note_vocabulary.go")
		if err != nil {
			return
		}
		for _, m := range notePrefixConstRe.FindAllStringSubmatch(string(data), -1) {
			notePrefixConstNames[m[1]] = m[2]
		}
	})
	return notePrefixConstNames
}

// fileBuildsPrefixFromVocabulary reports whether `varName` is assigned from
// something named "Note..." in this file (a NotePrefix constant or a Note*
// builder), which is the only acceptable way to obtain a prefix.
func fileBuildsPrefixFromVocabulary(src, varName string) bool {
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, varName+" :=") &&
			!strings.HasPrefix(trimmed, varName+" =") &&
			!strings.HasPrefix(trimmed, "var "+varName) {
			continue
		}
		rhs := trimmed
		if i := strings.Index(trimmed, ":="); i >= 0 {
			rhs = trimmed[i+2:]
		} else if i := strings.Index(trimmed, "="); i >= 0 {
			rhs = trimmed[i+1:]
		}
		if strings.Contains(rhs, "NotePrefix") || strings.Contains(rhs, "NoteFoundClub") ||
			strings.Contains(rhs, "NoteJoinClub") || strings.Contains(rhs, "NoteClaimDistrict") ||
			strings.Contains(rhs, "NoteTournamentBuyIn") || strings.Contains(rhs, "NoteArenaTournamentBuyIn") {
			return true
		}
	}
	return false
}

