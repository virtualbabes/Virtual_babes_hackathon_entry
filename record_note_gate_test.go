//go:build !js && !wasm

package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// ════════════════════════════════════════════════════════════════════════════
// THE NOTE-LITERAL GATE — a note is DECLARED, never spelled.
//
// WHY IT EXISTS (measured 2026-09-19, Problems.md §39): the Algorand payout rail carried
// `[]byte("VBET_ALGO_DIV")` — an UNDECLARED prefix with no trailing colon — on a real
// value-moving transaction. `TestNoNotePrefixLiteralOutsideTheOwner` could not see it:
// that test scans for the DECLARED prefixes, so an UNDECLARED literal is invisible to it
// by construction. This gate keys on the SHAPE instead — a byte-slice string literal
// passed to a note-carrying transaction builder — so it catches what the other cannot.
//
// A `_test.go` file is skipped on purpose: a fixture may legitimately spell a literal to
// drive a refusal path. Production code may not.
// ════════════════════════════════════════════════════════════════════════════

// noteCarryingBuilders are the calls that attach a NOTE to an on-chain transaction.
var noteCarryingBuilders = []string{"MakePaymentTxn(", "MakeApplicationNoOpTx(", "AppendNote("}

// noteLiteralGateErrors scans one source for a transaction builder whose note argument is
// a bare string literal. Returns one line per offending call site.
func noteLiteralGateErrors(name, src string) []string {
	var errs []string
	for _, builder := range noteCarryingBuilders {
		for from := 0; ; {
			i := strings.Index(src[from:], builder)
			if i < 0 {
				break
			}
			open := from + i + len(builder) - 1
			args, end, ok := balancedCallArgs(src, open)
			if !ok {
				from = open + 1 // an unparsable call is skipped, never guessed at
				continue
			}
			if strings.Contains(args, `[]byte("`) {
				errs = append(errs, fmt.Sprintf(
					"%s: a note is a STRING LITERAL inside %s — declare it in note_vocabulary.go and reference the constant",
					name, strings.TrimSuffix(builder, "(")))
			}
			from = end
		}
	}
	return errs
}

// balancedCallArgs returns the text inside the parentheses that start at `open`, plus the
// index just past the closing parenthesis. String literals and escapes are skipped, so a
// ')' inside a note payload cannot end the call early.
func balancedCallArgs(src string, open int) (string, int, bool) {
	if open < 0 || open >= len(src) || src[open] != '(' {
		return "", open, false
	}
	depth := 0
	inString := false
	escaped := false
	for i := open; i < len(src); i++ {
		c := src[i]
		if inString {
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return src[open+1 : i], i + 1, true
			}
		}
	}
	return "", open, false
}

// TestNoTransactionNoteIsAStringLiteral — every note on chain comes from the vocabulary
// owner, in production code.
func TestNoTransactionNoteIsAStringLiteral(t *testing.T) {
	files := firstPartyGoSources(t, "note_vocabulary.go", "note_vocabulary_test.go", "record_note_gate_test.go")
	scanned := 0
	var found []string
	for name, src := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue // a fixture may spell a literal to drive a refusal path
		}
		scanned++
		found = append(found, noteLiteralGateErrors(name, src)...)
	}
	if scanned == 0 {
		t.Fatal("no production Go source was scanned; the gate is meaningless")
	}
	if len(found) > 0 {
		t.Errorf("a note must be DECLARED, never a literal:\n  %s", strings.Join(found, "\n  "))
	}

	// A detector that matches nothing reports a clean repository for ever, so it is
	// driven against shapes that MUST report and shapes that MUST NOT.
	for name, src := range map[string]string{
		"a payment with a literal note":   `th, _ := transaction.MakePaymentTxn(a, b, 1, []byte("VBET_ALGO_DIV"), "", sp)`,
		"an app-call with a literal note": `th, _ := transaction.MakeApplicationNoOpTx(id, args, nil, nil, nil, sp, snd, []byte("BAD_NOTE:"), types.Digest{}, [32]byte{}, types.Address{})`,
	} {
		if len(noteLiteralGateErrors("synthetic.go", src)) == 0 {
			t.Errorf("%s must be REPORTED, not accepted", name)
		}
	}
	for name, src := range map[string]string{
		"a declared constant":       `th, _ := transaction.MakePaymentTxn(a, b, 1, []byte(NotePrefixAlgoDividend), "", sp)`,
		"a built prefix":            `th, _ := transaction.MakePaymentTxn(a, b, 1, []byte(fmt.Sprintf(NotePrefixWin+"%s", x)), "", sp)`,
		"a brace byte slice":        `th, _ := transaction.MakeApplicationNoOpTx(id, [][]byte{{0x2b}}, nil, nil, nil, sp, snd, note, types.Digest{}, [32]byte{}, types.Address{})`,
		"a literal, but not a note": `th, _ := transaction.MakePaymentTxn("ADDR", "ADDR", 1, note, "", sp)`,
	} {
		if errs := noteLiteralGateErrors("synthetic.go", src); len(errs) > 0 {
			t.Errorf("%s must NOT be reported, got %v", name, errs)
		}
	}
}

// TestTournamentArchiveWritesThroughTheRecordWriter — the archive's notes must be WRITTEN.
// The previous shape issued an HTTP GET whose URL PATH was the note, so the archive existed
// nowhere, a nonsense request fired per chunk, and `Links` — documented as "TxIDs for
// additional match data" — carried those URLs. This pin fails if the stub ever returns.
func TestTournamentArchiveWritesThroughTheRecordWriter(t *testing.T) {
	raw, err := os.ReadFile("tournament_manager.go")
	if err != nil {
		t.Fatalf("read tournament_manager.go: %v", err)
	}
	body := goFunctionBody(string(raw), "func (s *TournamentService) RecordTournamentOnChain(")
	if body == "" {
		t.Fatal("RecordTournamentOnChain was not found — this pin would be meaningless")
	}
	if strings.Contains(body, "IndexerRequest") {
		t.Error("the archive must NOT 'write' a note with an indexer GET (a GET cannot carry a note)")
	}
	if !strings.Contains(body, "sendAuditNoteStream") {
		t.Error("the archive must write through the ONE record writer")
	}
	if !strings.Contains(body, "summary.Links = nil") {
		t.Error("Links is documented as TxIDs; it must be left empty rather than filled with a URL")
	}
}

// goFunctionBody returns the brace-balanced body of the function whose header starts with
// `header`, or "" when the header is absent. Braces inside strings, raw strings and
// comments are skipped, so a JSON payload cannot end the body early.
func goFunctionBody(src, header string) string {
	start := strings.Index(src, header)
	if start < 0 {
		return ""
	}
	open := strings.Index(src[start:], "{")
	if open < 0 {
		return ""
	}
	open += start

	depth := 0
	inString, inRaw, inLineComment, inBlockComment, escaped := false, false, false, false, false
	for i := open; i < len(src); i++ {
		c := src[i]
		switch {
		case inLineComment:
			if c == '\n' {
				inLineComment = false
			}
			continue
		case inBlockComment:
			if c == '*' && i+1 < len(src) && src[i+1] == '/' {
				inBlockComment = false
				i++
			}
			continue
		case inString:
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		case inRaw:
			if c == '`' {
				inRaw = false
			}
			continue
		}
		switch {
		case c == '/' && i+1 < len(src) && src[i+1] == '/':
			inLineComment = true
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			inBlockComment = true
		case c == '"':
			inString = true
		case c == '`':
			inRaw = true
		case c == '{':
			depth++
		case c == '}':
			depth--
			if depth == 0 {
				return src[open : i+1]
			}
		}
	}
	return ""
}

// TestAuditNoteStreamRefusesAnUndeclaredPurpose — the audit door admits only a DECLARED
// purpose, and a refusal must not consume a nonce: a refused write is not a write.
func TestAuditNoteStreamRefusesAnUndeclaredPurpose(t *testing.T) {
	const bogus = "VBT_NOT_DECLARED:"
	if IsDeclaredNotePrefix(bogus) {
		t.Fatalf("%s must NOT be declared for this test to mean anything", bogus)
	}
	if !IsDeclaredNotePrefix(NotePrefixTournData) {
		t.Errorf("%s must be declared — the tournament archive writes it", NotePrefixTournData)
	}
	if !IsDeclaredNotePrefix(NotePrefixAlgoDividend) {
		t.Errorf("%s must be declared — a value-moving payout carries it", NotePrefixAlgoDividend)
	}

	recordNonceReset()
	l := &Lobby{}
	l.sendAuditNoteStream(bogus, []byte(`{"x":1}`))
	if got := recordNonceCurrentValue(bogus); got != 0 {
		t.Errorf("a REFUSED purpose must not consume a nonce, got %d", got)
	}
	recordNonceReset()
}
