//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRecordEnvelopeRoundTrips — the record a family carries is a whole map, so the
// envelope exists to make that possible at all. This proves the BYTES survive.
func TestRecordEnvelopeRoundTrips(t *testing.T) {
	want := []byte(`{"balances":{"0xabc":17},"note":"one chunk is enough"}`)

	notes, err := encodeRecordNotes(NotePrefixEconomySnapshot, 1, want)
	if err != nil {
		t.Fatalf("the encoder must accept a small record: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("a small record must need ONE note, got %d", len(notes))
	}
	if !strings.HasPrefix(notes[0], NotePrefixEconomySnapshot+recordEnvelopeTag+".") {
		t.Fatalf("the note must carry the family prefix AND the envelope tag, got %q", notes[0])
	}

	rn, err := parseRecordNote(NotePrefixEconomySnapshot, notes[0], "TX1", 1700)
	if err != nil {
		t.Fatalf("the note must parse: %v", err)
	}
	if rn.Legacy || rn.Nonce != 1 || rn.Chunk != 0 || rn.Total != 1 {
		t.Fatalf("parsed envelope = %+v; want nonce 1, chunk 0 of 1, not legacy", rn)
	}

	complete, gaps := assembleRecords([]recordNote{rn})
	if len(gaps) != 0 {
		t.Errorf("a complete record must report NO gap, got %v", gaps)
	}
	if len(complete) != 1 {
		t.Fatalf("one complete record expected, got %d", len(complete))
	}
	got, err := decodeSnapshotPayload(NotePrefixEconomySnapshot, complete[0].Body)
	if err != nil {
		t.Fatalf("the reassembled body must decode: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("the round trip changed the bytes:\n got %s\nwant %s", got, want)
	}
}

// TestRecordEnvelopeRespectsTheAVMNoteCap — the cap is not a preference. A record far
// larger than one note MUST be split, and EVERY note must fit inside the cap. This is
// the defect the envelope exists for: the previous single-note writer built a
// transaction the chain could not have accepted.
func TestRecordEnvelopeRespectsTheAVMNoteCap(t *testing.T) {
	// A realistic family state: tens of KB that does not compress to nothing.
	payload := recordTestPayload(2000)

	notes, err := encodeRecordNotes(NotePrefixEntityMarketSnapshot, 42, payload)
	if err != nil {
		t.Fatalf("the encoder must accept a realistic record: %v", err)
	}
	if len(notes) < 2 {
		t.Fatalf("a %d byte record must need MORE than one note, got %d", len(payload), len(notes))
	}
	for i, note := range notes {
		if len(note) > avmNoteMaxBytes {
			t.Errorf("note %d is %d bytes, above the %d byte AVM cap", i, len(note), avmNoteMaxBytes)
		}
	}

	// Reassemble EXACTLY as the reader does.
	parsed := make([]recordNote, 0, len(notes))
	for i, note := range notes {
		rn, perr := parseRecordNote(NotePrefixEntityMarketSnapshot, note, fmt.Sprintf("TX%d", i), int64(1000+i))
		if perr != nil {
			t.Fatalf("chunk %d must parse: %v", i, perr)
		}
		parsed = append(parsed, rn)
	}
	complete, gaps := assembleRecords(parsed)
	if len(gaps) != 0 || len(complete) != 1 {
		t.Fatalf("a complete multi-chunk record must reassemble: gaps=%v complete=%d", gaps, len(complete))
	}
	if complete[0].Chunks != len(notes) {
		t.Errorf("the selected record reports %d chunks; want %d", complete[0].Chunks, len(notes))
	}
	got, err := decodeSnapshotPayload(NotePrefixEntityMarketSnapshot, complete[0].Body)
	if err != nil {
		t.Fatalf("the reassembled body must decode: %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("the reassembled record differs from what was encoded (%d vs %d bytes)", len(got), len(payload))
	}
}

// TestRecordEnvelopeRefusesRatherThanGuesses — every unusable note names its reason;
// none is silently accepted and none is silently dropped.
func TestRecordEnvelopeRefusesRatherThanGuesses(t *testing.T) {
	prefix := NotePrefixPetSnapshot
	for _, tc := range []struct{ name, note string }{
		{"another family's note", "SOMETHING_ELSE" + recordEnvelopeTag + ".1.0.1.AAAA"},
		{"the prefix and nothing else", prefix},
		{"a FUTURE envelope", prefix + "R2.1.0.1.AAAA"},
		{"a non-numeric nonce", prefix + recordEnvelopeTag + ".x.0.1.AAAA"},
		{"a non-numeric chunk", prefix + recordEnvelopeTag + ".1.x.1.AAAA"},
		{"a non-numeric total", prefix + recordEnvelopeTag + ".1.0.x.AAAA"},
		{"a chunk beyond the total", prefix + recordEnvelopeTag + ".1.3.2.AAAA"},
		{"an empty payload", prefix + recordEnvelopeTag + ".1.0.1."},
	} {
		if _, err := parseRecordNote(prefix, tc.note, "TX", 1); err == nil {
			t.Errorf("%s must be REFUSED, not accepted", tc.name)
		}
	}

	if _, err := encodeRecordNotes("", 1, []byte("x")); err == nil {
		t.Error("a record with no family prefix must be refused")
	}
	if _, err := encodeRecordNotes(prefix, 1, nil); err == nil {
		t.Error("a record carrying no bytes must be refused")
	}
	if _, err := encodeRecordNotes(strings.Repeat("VBT_LONG_", 120), 1, []byte("x")); err == nil {
		t.Error("a prefix that leaves no room in a note must be refused, not truncated")
	}
}

// TestRecordNonceIsASequenceTheReaderCanContinue — a SEQUENCE makes a missing update
// detectable; a clock does not. The seed may only ever RAISE the sequence.
func TestRecordNonceIsASequenceTheReaderCanContinue(t *testing.T) {
	recordNonceReset()
	if got := recordNonceNext("pets"); got != 1 {
		t.Errorf("the first nonce = %d; want 1", got)
	}
	if got := recordNonceNext("pets"); got != 2 {
		t.Errorf("the second nonce = %d; want 2", got)
	}
	// Each family has its OWN sequence: one family's history must not shift another's.
	if got := recordNonceNext("vehicles"); got != 1 {
		t.Errorf("another family must start at 1, got %d", got)
	}

	recordNonceSeed("pets", 9)
	if got := recordNonceCurrentValue("pets"); got != 9 {
		t.Errorf("the seed must raise the sequence to 9, got %d", got)
	}
	if got := recordNonceNext("pets"); got != 10 {
		t.Errorf("the write after a seed must be 10, got %d", got)
	}
	// A LOWER seed must NOT rewind: that would rewrite a nonce the chain already holds.
	recordNonceSeed("pets", 3)
	if got := recordNonceCurrentValue("pets"); got != 10 {
		t.Errorf("a lower seed must not rewind the sequence, got %d", got)
	}
	recordNonceReset()
}

// TestRecordOrderComesFromTheNonceNotTheClock — the discriminator for the whole
// design. The OLDER nonce is given the LATER block time, so a reader that ordered
// by time would restore the stale record. A sequence cannot be fooled that way.
func TestRecordOrderComesFromTheNonceNotTheClock(t *testing.T) {
	prefix := NotePrefixFaithSnapshot
	older := []byte(`{"churches":["old"]}`)
	newer := []byte(`{"churches":["new"]}`)

	olderNotes, err := encodeRecordNotes(prefix, 1, older)
	if err != nil {
		t.Fatalf("encode older: %v", err)
	}
	newerNotes, err := encodeRecordNotes(prefix, 2, newer)
	if err != nil {
		t.Fatalf("encode newer: %v", err)
	}

	oldNote, _ := parseRecordNote(prefix, olderNotes[0], "TXOLD", 9999) // LATER on chain
	newNote, _ := parseRecordNote(prefix, newerNotes[0], "TXNEW", 10)   // EARLIER on chain

	complete, gaps := assembleRecords([]recordNote{oldNote, newNote})
	if len(gaps) != 0 {
		t.Fatalf("both records are complete, so no gap is expected: %v", gaps)
	}
	if len(complete) != 2 {
		t.Fatalf("both nonces are complete, got %d", len(complete))
	}
	if complete[0].Nonce != 2 {
		t.Fatalf("the NEWEST NONCE must be selected first, got nonce %d", complete[0].Nonce)
	}
	got, err := decodeSnapshotPayload(prefix, complete[0].Body)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(got) != string(newer) {
		t.Errorf("the selected record = %s; want the nonce-2 record %s", got, newer)
	}
}

// TestRecordGapIsDetectedAndReported — a partial write must never be read as a whole
// record. The newest COMPLETE record wins and the gap is NAMED.
func TestRecordGapIsDetectedAndReported(t *testing.T) {
	prefix := NotePrefixRivalrySnapshot
	big, err := encodeRecordNotes(prefix, 5, recordTestPayload(300))
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if len(big) < 2 {
		t.Fatalf("this test needs a multi-chunk record, got %d chunk(s)", len(big))
	}
	partial := big[:len(big)-1] // the LAST chunk never landed

	older, err := encodeRecordNotes(prefix, 4, []byte(`{"rivalries":[]}`))
	if err != nil {
		t.Fatalf("encode older: %v", err)
	}

	notes := []recordNote{}
	for i, note := range partial {
		rn, perr := parseRecordNote(prefix, note, fmt.Sprintf("PART%d", i), 900)
		if perr != nil {
			t.Fatalf("parse: %v", perr)
		}
		notes = append(notes, rn)
	}
	rn, perr := parseRecordNote(prefix, older[0], "OLD", 100)
	if perr != nil {
		t.Fatalf("parse older: %v", perr)
	}
	notes = append(notes, rn)

	complete, gaps := assembleRecords(notes)
	if len(complete) != 1 {
		t.Fatalf("exactly the ONE complete record must be selectable, got %d", len(complete))
	}
	if complete[0].Nonce != 4 {
		t.Errorf("the newest COMPLETE nonce is 4, got %d", complete[0].Nonce)
	}
	if len(gaps) != 1 || gaps[0].Nonce != 5 {
		t.Fatalf("the incomplete nonce 5 must be reported as a gap, got %v", gaps)
	}
	if len(gaps[0].Missing) == 0 {
		t.Error("a gap must NAME what is missing")
	}
	summary := recordGapSummary(prefix, gaps[0])
	if !strings.Contains(summary, "INCOMPLETE") || !strings.Contains(summary, "nonce 5") {
		t.Errorf("the gap must be explainable in one line, got %q", summary)
	}
}

// TestRecordWithDisagreeingTotalsIsNotARecord — two chunks claiming different totals
// are two different claims about one record, which is not a record.
func TestRecordWithDisagreeingTotalsIsNotARecord(t *testing.T) {
	complete, gaps := assembleRecords([]recordNote{
		{TransactionID: "A", Nonce: 7, Chunk: 0, Total: 2, Body: "AAAA", Timestamp: 5},
		{TransactionID: "B", Nonce: 7, Chunk: 1, Total: 3, Body: "BBBB", Timestamp: 6},
	})
	if len(complete) != 0 {
		t.Errorf("disagreeing totals must NOT assemble into a record, got %d", len(complete))
	}
	if len(gaps) != 1 {
		t.Fatalf("the nonce must be reported as incomplete, got %v", gaps)
	}
	if gaps[0].Total != 3 || gaps[0].Have != 2 {
		t.Errorf("gap = %+v; want the larger total (3) and both chunks counted", gaps[0])
	}
}

// TestLegacyNoteIsReadAndFlagged — the PRE-ENVELOPE shape (prefix + base64) is a real
// note shape this code once wrote. It must still be READ, and it must be FLAGGED: a
// reader that dropped it would lose state, and one that accepted it silently would
// hide that the writer's format had changed.
func TestLegacyNoteIsReadAndFlagged(t *testing.T) {
	prefix := NotePrefixItemSnapshot
	want := []byte(`{"items":["a-built-item"]}`)

	notes, err := encodeRecordNotes(prefix, 3, want)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	// Rebuild the OLD shape from a real body: drop the envelope header entirely.
	parts := strings.SplitN(strings.TrimPrefix(notes[0], prefix), ".", 5)
	if len(parts) != 5 {
		t.Fatalf("this test needs an enveloped note, got %q", notes[0])
	}
	legacy := prefix + parts[4]

	rn, err := parseRecordNote(prefix, legacy, "TXLEGACY", 5)
	if err != nil {
		t.Fatalf("a pre-envelope note must still parse: %v", err)
	}
	if !rn.Legacy {
		t.Error("a pre-envelope note must be FLAGGED as legacy")
	}
	if rn.Nonce != 0 || rn.Chunk != 0 || rn.Total != 1 {
		t.Errorf("a legacy note is one chunk of nonce 0, got %+v", rn)
	}

	complete, gaps := assembleRecords([]recordNote{rn})
	if len(gaps) != 0 || len(complete) != 1 {
		t.Fatalf("a legacy note alone must be a complete record: gaps=%v complete=%d", gaps, len(complete))
	}
	if !complete[0].Legacy {
		t.Error("the selection must carry the legacy flag through")
	}
	got, err := decodeSnapshotPayload(prefix, complete[0].Body)
	if err != nil {
		t.Fatalf("the legacy body must decode: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("legacy round trip = %s; want %s", got, want)
	}
}

// TestReadRecordFamilyReassemblesChunksFromTheIndexer — the END-TO-END proof: the real
// transport, the real vault assertion, the real envelope, the real reassembly. It uses
// a family declared only on 2026-09-19, so it also proves a new family needs no reader
// of its own.
func TestReadRecordFamilyReassemblesChunksFromTheIndexer(t *testing.T) {
	recordNonceReset()
	const family = "ai_citizens"
	prefix := NotePrefixAICitizenSnapshot

	// A deterministic, LOW-REDUNDANCY payload: a family's state is not a text file, and
	// a highly compressible payload would make this test's chunk count an accident.
	payload := recordTestPayload(240)

	notes, err := encodeRecordNotes(prefix, 9, payload)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if len(notes) < 2 {
		t.Fatalf("this test needs a multi-chunk record, got %d chunk(s)", len(notes))
	}

	transactions := make([]map[string]any, 0, len(notes)+1)
	for i, note := range notes {
		transactions = append(transactions, map[string]any{
			"id": fmt.Sprintf("TX%d", i), "note": []byte(note),
			"sender": testVaultAddr, "round-time": 2000 + i, "tx-type": "appl",
		})
	}
	// A THIRD PARTY writing the same prefix and the LATEST block time must be ignored.
	transactions = append(transactions, map[string]any{
		"id": "TXPARTY", "note": []byte(notes[0]),
		"sender": arc200TestPayer, "round-time": 99999, "tx-type": "appl",
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"current-round": 1,
			"transactions":  transactions,
		})
	}))
	t.Cleanup(ts.Close)

	l := &Lobby{}
	got, at, found, err := l.readRecordFamily(NetworkConfig{IndexerURLs: []string{ts.URL}}, testVaultAddr, prefix)
	if err != nil {
		t.Fatalf("the read must succeed: %v", err)
	}
	if !found {
		t.Fatal("the record must be found")
	}
	if string(got) != string(payload) {
		t.Errorf("the reassembled record differs from what was written (%d vs %d bytes)", len(got), len(payload))
	}
	if want := int64(2000 + len(notes) - 1); at != want {
		t.Errorf("block time = %d; want the newest CHUNK's time %d (the third party's 99999 must be ignored)", at, want)
	}
	// The reader CONTINUES the writer's sequence, so a restart cannot reuse a nonce.
	if seed := recordNonceCurrentValue(family); seed != 9 {
		t.Errorf("the nonce sequence was seeded to %d; want 9", seed)
	}
	if next := recordNonceNext(family); next != 10 {
		t.Errorf("the next write must be nonce 10, got %d", next)
	}
	recordNonceReset()
}

// TestReadRecordFamilyReportsNothingFoundHonestly — a fresh world has no records, and
// that is NOT an error. An empty read must say `found=false` rather than fabricate.
func TestReadRecordFamilyReportsNothingFoundHonestly(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"current-round": 1, "transactions": []map[string]any{}})
	}))
	t.Cleanup(ts.Close)

	l := &Lobby{}
	got, at, found, err := l.readRecordFamily(NetworkConfig{IndexerURLs: []string{ts.URL}}, testVaultAddr, NotePrefixVehicleSnapshot)
	if err != nil {
		t.Fatalf("an empty chain is not an error: %v", err)
	}
	if found || got != nil || at != 0 {
		t.Errorf("an empty read = (data=%v, at=%d, found=%v); want (nil, 0, false)", got, at, found)
	}
}

// recordTestPayload builds a deterministic, LOW-REDUNDANCY payload of n records. A
// compressible payload would make a chunk-count assertion an accident rather than a
// measurement — 3,000 identical bytes gzip to a few dozen, which is ONE chunk.
func recordTestPayload(n int) []byte {
	seed := uint64(88172645463325252)
	var sb strings.Builder
	for i := 0; i < n; i++ {
		seed = seed*6364136223846793005 + 1442695040888963407
		fmt.Fprintf(&sb, "%016x", seed)
	}
	return []byte(sb.String())
}
