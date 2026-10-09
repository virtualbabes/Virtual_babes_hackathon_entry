//go:build !js && !wasm

package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// ════════════════════════════════════════════════════════════════════════════
// THE RECORD ENVELOPE — ONE owner of how a record is CARRIED, and of how it is
// put back together.
//
// WHY A SEQUENCE OF NOTES AND NOT ONE NOTE. The AVM caps a transaction NOTE at
// 1024 bytes. The previous writer built `prefix + base64(gzip(json))` in ONE
// note, and the state a family carries is a whole map (the leaderboard is every
// player's PlayerStats), so the transaction it built COULD NOT HAVE BEEN
// ACCEPTED: the record design was not merely inefficient, it was unbounded by
// construction and had never been exercised (the dispatch gate is OFF). A record
// is therefore a SEQUENCE of chunk notes.
//
// WHY A SELF-NONCE, AND WHY A SEQUENCE RATHER THAN A CLOCK. The nonce is a
// monotonic per-family SEQUENCE, because a sequence makes a MISSING update
// DETECTABLE — a nonce's chunks are contiguous and the nonces themselves are
// ordered — while a timestamp makes a gap look like a quiet period. The reader
// therefore selects the newest COMPLETE nonce and REPORTS every incomplete one
// instead of quietly reading stale state.
//
// THE NOTE GRAMMAR (strict, fail-closed):
//
//	<prefix>R1.<nonce>.<chunk>.<total>.<base64(gzip(data))>
//
// `.` is NOT in the base64 alphabet, so the body can never contain a separator
// and the header can never be confused with the payload.
//
// THE PAYLOAD BODY IS UNCHANGED, which is what lets the battle-tested caps in
// snapshot_guard.go (8 MiB base64 / 64 MiB decoded) keep applying at the same
// boundary: a chunk body is exactly what the old single-note shape carried.
//
// A NOTE WITHOUT THE ENVELOPE IS STILL READ — as ONE chunk of nonce 0, FLAGGED as
// legacy. Nothing has ever been written in the old shape on chain, but a reader
// that refused it would discard state if one existed, and a reader that accepted
// it SILENTLY would hide that the writer's format had changed. So it is accepted
// AND flagged.
// ════════════════════════════════════════════════════════════════════════════

const (
	// avmNoteMaxBytes is the AVM's hard cap on a transaction note. Every chunk
	// note MUST fit inside it or the transaction is rejected by the chain.
	avmNoteMaxBytes = 1024
	// recordEnvelopeTag identifies the format AND its version (R + version), so a
	// future format is REFUSED BY NAME instead of being mis-parsed as this one.
	recordEnvelopeTag = "R1"
	// recordMaxChunks bounds ONE record. Each chunk is a paid transaction, so a
	// record needing more than this is a design error rather than a large state.
	recordMaxChunks = 8192
	// recordGapReportLimit caps how many missing chunk indices one gap line names.
	recordGapReportLimit = 8
)

// recordNote is ONE note as the reader understands it: the envelope's header
// fields plus the payload body, still in its encoded form.
type recordNote struct {
	TransactionID string
	Nonce         uint64
	Chunk         int
	Total         int
	Body          string // base64(gzip(data)) — decoded by decodeSnapshotPayload
	Timestamp     int64
	Legacy        bool
}

// recordSelection is ONE COMPLETE record: every chunk of one nonce, in order.
type recordSelection struct {
	Nonce     uint64
	Chunks    int
	Body      string
	Timestamp int64
	TxIDs     []string
	Legacy    bool
}

// recordGap is an INCOMPLETE nonce — a record whose chunks are not all present.
// A gap is a MEASUREMENT, so it is reported rather than smoothed over.
type recordGap struct {
	Nonce   uint64
	Total   int
	Have    int
	Missing []int
}

// recordHeaderLen is the exact byte length of `R1.<nonce>.<chunk>.<total>.`.
func recordHeaderLen(nonce uint64, chunk, total int) int {
	return len(recordEnvelopeTag) + 1 + len(strconv.FormatUint(nonce, 10)) + 1 +
		len(strconv.Itoa(chunk)) + 1 + len(strconv.Itoa(total)) + 1
}

// recordChunkCapacity is how many base64 characters ONE chunk may carry. It is
// computed for the HIGHEST chunk index, because the last index has the most
// digits and that is the worst case — sizing it for chunk 0 would push the last
// chunk out of the cap.
func recordChunkCapacity(prefix string, nonce uint64, total int) int {
	worst := total - 1
	if worst < 0 {
		worst = 0
	}
	return avmNoteMaxBytes - len(prefix) - recordHeaderLen(nonce, worst, total)
}

// encodeRecordNotes splits ONE record into the smallest number of chunk notes
// that each fit inside the AVM cap.
//
// The arithmetic is NOT trusted: every produced note is measured against the cap,
// and the chunks are reassembled and compared with the base64 that was
// compressed. A split that does not cover its payload is refused.
func encodeRecordNotes(prefix string, nonce uint64, data []byte) ([]string, error) {
	if prefix == "" {
		return nil, fmt.Errorf("a record needs a family prefix")
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("%s record carries no bytes", prefix)
	}

	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write(data); err != nil {
		return nil, fmt.Errorf("%s record compression failed: %w", prefix, err)
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("%s record compression failed: %w", prefix, err)
	}
	body := base64.StdEncoding.EncodeToString(b.Bytes())

	// Fixed point: the chunk count sets the header (digits of `total`), which sets
	// the capacity, which sets the chunk count.
	total := 1
	for i := 0; i < 32; i++ {
		chunkCap := recordChunkCapacity(prefix, nonce, total)
		if chunkCap <= 0 {
			return nil, fmt.Errorf("the prefix %s leaves no room in a %d byte note", prefix, avmNoteMaxBytes)
		}
		need := (len(body) + chunkCap - 1) / chunkCap
		if need < 1 {
			need = 1
		}
		if need > recordMaxChunks {
			return nil, fmt.Errorf("%s record needs %d chunks, above the %d chunk cap", prefix, need, recordMaxChunks)
		}
		if need == total {
			break
		}
		if i == 31 {
			return nil, fmt.Errorf("%s record chunk count did not converge", prefix)
		}
		total = need
	}

	chunkCap := recordChunkCapacity(prefix, nonce, total)
	if chunkCap <= 0 {
		return nil, fmt.Errorf("the prefix %s leaves no room in a %d byte note", prefix, avmNoteMaxBytes)
	}

	notes := make([]string, 0, total)
	bodies := make([]string, 0, total)
	for i := 0; i < total; i++ {
		start := i * chunkCap
		end := start + chunkCap
		if end > len(body) {
			end = len(body)
		}
		part := body[start:end]
		note := prefix + recordEnvelopeTag + "." + strconv.FormatUint(nonce, 10) + "." +
			strconv.Itoa(i) + "." + strconv.Itoa(total) + "." + part
		if len(note) > avmNoteMaxBytes {
			return nil, fmt.Errorf("%s record chunk %d of %d is %d bytes, above the %d byte note cap",
				prefix, i+1, total, len(note), avmNoteMaxBytes)
		}
		notes = append(notes, note)
		bodies = append(bodies, part)
	}

	// PROOF OF COVERAGE, not an assumption: the chunks must reassemble into the
	// exact base64 that was compressed.
	if strings.Join(bodies, "") != body {
		return nil, fmt.Errorf("%s record chunks do not cover their payload (%d chunks)", prefix, total)
	}
	return notes, nil
}

// parseRecordNote reads ONE note into the envelope's fields.
//
// A note that carries the prefix but NOT the envelope is accepted as one chunk of
// nonce 0 and flagged legacy (see the file header). Anything else that cannot be
// read is REFUSED with its reason — never guessed at.
func parseRecordNote(prefix, note, txID string, ts int64) (recordNote, error) {
	if prefix == "" {
		return recordNote{}, fmt.Errorf("a record read needs the family prefix")
	}
	if !strings.HasPrefix(note, prefix) {
		return recordNote{}, fmt.Errorf("the note does not carry the family prefix %s", prefix)
	}
	rest := strings.TrimPrefix(note, prefix)
	if rest == "" {
		return recordNote{}, fmt.Errorf("%s note carries no payload", prefix)
	}

	parts := strings.SplitN(rest, ".", 5)
	if len(parts) != 5 {
		return recordNote{TransactionID: txID, Nonce: 0, Chunk: 0, Total: 1,
			Body: rest, Timestamp: ts, Legacy: true}, nil
	}
	if parts[0] != recordEnvelopeTag {
		return recordNote{}, fmt.Errorf("unknown record envelope %q (this build writes %s, %d bytes per note)",
			parts[0], recordEnvelopeTag, avmNoteMaxBytes)
	}

	nonce, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return recordNote{}, fmt.Errorf("%s record nonce %q is not a number", prefix, parts[1])
	}
	chunk, err := strconv.Atoi(parts[2])
	if err != nil {
		return recordNote{}, fmt.Errorf("%s record chunk %q is not a number", prefix, parts[2])
	}
	total, err := strconv.Atoi(parts[3])
	if err != nil {
		return recordNote{}, fmt.Errorf("%s record total %q is not a number", prefix, parts[3])
	}
	if total < 1 || chunk < 0 || chunk >= total {
		return recordNote{}, fmt.Errorf("%s record chunk %d of %d is out of range", prefix, chunk, total)
	}
	if parts[4] == "" {
		return recordNote{}, fmt.Errorf("%s record chunk %d of %d carries no payload", prefix, chunk, total)
	}

	return recordNote{TransactionID: txID, Nonce: nonce, Chunk: chunk, Total: total,
		Body: parts[4], Timestamp: ts}, nil
}

// ── ASSEMBLY ───────────────────────────────────────────────────────────────

// assembleRecords turns every note read for ONE family into (a) its COMPLETE
// records, newest nonce first, and (b) its INCOMPLETE nonces, reported as gaps.
//
// A nonce whose chunks disagree about `total`, or that repeats a chunk index, is
// treated as INCOMPLETE: two different claims about one record is not a record.
func assembleRecords(notes []recordNote) (complete []recordSelection, gaps []recordGap) {
	byNonce := map[uint64][]recordNote{}
	for _, n := range notes {
		if n.Total < 1 {
			continue // a refused note; the reader already logged its reason
		}
		byNonce[n.Nonce] = append(byNonce[n.Nonce], n)
	}

	nonces := make([]uint64, 0, len(byNonce))
	for nonce := range byNonce {
		nonces = append(nonces, nonce)
	}
	sort.Slice(nonces, func(i, j int) bool { return nonces[i] > nonces[j] })

	for _, nonce := range nonces {
		chunks := byNonce[nonce]
		if sel, ok := completeRecordFromChunks(nonce, chunks); ok {
			complete = append(complete, sel)
			continue
		}
		gaps = append(gaps, gapFromChunks(nonce, chunks))
	}
	return complete, gaps
}

// completeRecordFromChunks rebuilds ONE record, or reports that it cannot: every
// chunk index 0..total-1 present exactly once, all agreeing on `total`.
func completeRecordFromChunks(nonce uint64, chunks []recordNote) (recordSelection, bool) {
	if len(chunks) == 0 {
		return recordSelection{}, false
	}
	total := chunks[0].Total
	for _, c := range chunks {
		if c.Total != total {
			return recordSelection{}, false
		}
	}
	if len(chunks) != total {
		return recordSelection{}, false
	}

	ordered := make([]recordNote, total)
	seen := make([]bool, total)
	legacy := 0
	for _, c := range chunks {
		if c.Chunk < 0 || c.Chunk >= total || seen[c.Chunk] {
			return recordSelection{}, false
		}
		seen[c.Chunk] = true
		ordered[c.Chunk] = c
		if c.Legacy {
			legacy++
		}
	}
	for _, ok := range seen {
		if !ok {
			return recordSelection{}, false
		}
	}

	var sb strings.Builder
	var ts int64
	txids := make([]string, 0, total)
	for _, c := range ordered {
		sb.WriteString(c.Body)
		if c.Timestamp > ts {
			ts = c.Timestamp
		}
		if c.TransactionID != "" {
			txids = append(txids, c.TransactionID)
		}
	}
	return recordSelection{
		Nonce:     nonce,
		Chunks:    total,
		Body:      sb.String(),
		Timestamp: ts,
		TxIDs:     txids,
		Legacy:    legacy == total,
	}, true
}

// gapFromChunks names what a nonce is missing.
func gapFromChunks(nonce uint64, chunks []recordNote) recordGap {
	total := 0
	for _, c := range chunks {
		if c.Total > total {
			total = c.Total
		}
	}
	present := map[int]bool{}
	for _, c := range chunks {
		present[c.Chunk] = true
	}
	missing := make([]int, 0, total)
	for i := 0; i < total; i++ {
		if !present[i] {
			missing = append(missing, i)
		}
	}
	return recordGap{Nonce: nonce, Total: total, Have: len(present), Missing: missing}
}

// recordGapSummary states a gap in one line. The missing indices are capped so a
// badly broken log cannot flood the boot log.
func recordGapSummary(prefix string, g recordGap) string {
	shown := g.Missing
	suffix := ""
	if len(shown) > recordGapReportLimit {
		suffix = fmt.Sprintf(" (+%d more)", len(shown)-recordGapReportLimit)
		shown = shown[:recordGapReportLimit]
	}
	return fmt.Sprintf("%s record nonce %d is INCOMPLETE: %d of %d chunk(s) present, missing %v%s",
		prefix, g.Nonce, g.Have, g.Total, shown, suffix)
}

// ── THE WRITER'S NONCE SOURCE ──────────────────────────────────────────────

// One monotonic SEQUENCE per family, seeded by the reader from the highest nonce
// it found on chain, so a restart continues the sequence instead of restarting it
// and colliding with its own history.
//
// LIMIT, stated rather than implied: the seed is per-PROCESS. No record has ever
// been written on chain (the dispatch gate is OFF and the vault has sent no
// app-call), so there is no history to continue yet. When the seed is planted (D1)
// the highest nonce per family becomes part of the restored state, and this note
// must be revised WITH that state rather than left to drift.
var (
	recordNonceMu      sync.Mutex
	recordNonceCurrent = map[string]uint64{}
)

// recordNonceNext returns the next nonce for a family and advances the sequence.
func recordNonceNext(familyKey string) uint64 {
	recordNonceMu.Lock()
	defer recordNonceMu.Unlock()
	recordNonceCurrent[familyKey]++
	return recordNonceCurrent[familyKey]
}

// recordNonceSeed raises a family's sequence to at least `seen`, so a reader that
// found nonce N on chain makes the next write N+1.
func recordNonceSeed(familyKey string, seen uint64) {
	recordNonceMu.Lock()
	defer recordNonceMu.Unlock()
	if seen > recordNonceCurrent[familyKey] {
		recordNonceCurrent[familyKey] = seen
	}
}

// recordNonceCurrentValue reports a family's sequence without advancing it.
func recordNonceCurrentValue(familyKey string) uint64 {
	recordNonceMu.Lock()
	defer recordNonceMu.Unlock()
	return recordNonceCurrent[familyKey]
}

// recordNonceReset clears every sequence. It exists for TESTS: a package-level
// sequence that cannot be reset makes every test that writes a record depend on
// the order of the tests before it.
func recordNonceReset() {
	recordNonceMu.Lock()
	defer recordNonceMu.Unlock()
	recordNonceCurrent = map[string]uint64{}
}
