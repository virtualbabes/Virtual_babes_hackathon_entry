//go:build !js && !wasm

package main

// -- THE BATCH ENVELOPE (A5, plan 15) --
//
// ONE record per cadence carrying SEVERAL families, chunked sequentially when too large. A batch is
// TRANSPORT: it owns no fact, so it carries a family KEY and that family RAW JSON, and never a
// reinterpretation of either. A key no family declares is REFUSED here, because batching it would be a
// second owner of a fact. The envelope itself (gzip, the 1024 byte cap, the chunk count) is owned by
// record_envelope.go; this file owns only the PAYLOAD inside it.

import (
	
"encoding/json"
	"log"
	
"errors"
	
"fmt"
)

// recordBatchPayload is the gzipped payload. The NONCE is the batch sequence, and the per-family
// sequences travel WITH it, so a missing batch is detectable while every family is still ordered
// against its own history rather than against a clock.
type recordBatchPayload struct {
	
Nonce     uint64                     `json:"nonce"`
	
Families  map[string]json.RawMessage `json:"families"`
	
Sequences map[string]uint64          `json:"family_sequences"`
}

// buildRecordBatchPayload refuses an empty batch, an unnamed family, a family that the record set does
// NOT declare, and a family with no bytes - each of which would otherwise restore as a silent empty.
func buildRecordBatchPayload(nonce uint64, families map[string]json.RawMessage, sequences map[string]uint64) ([]byte, error) {
	
if len(families) == 0 {
	
	
return nil, errors.New("record batch: refusing an empty batch, because a record with no families restores nothing and looks like progress")
	
}
	
for key, raw := range families {
	
	
if key == "" {
	
	
	
return nil, errors.New("record batch: a family with no key cannot be read back")
	
	
}
	
	
if !recordBatchFamilyIsDeclared(key) {
	
	
	
return nil, fmt.Errorf("record batch: family %q is not declared in PrimaryRecordFamilies, so batching it would create a second owner of a fact", key)
	
	
}
	
	
if len(raw) == 0 {
	
	
	
return nil, fmt.Errorf("record batch: family %q carries no bytes; an absent family and an empty one must never be the same answer on restore", key)
	
	
}
	
}
	
return json.Marshal(recordBatchPayload{Nonce: nonce, Families: families, Sequences: sequences})
}

// recordBatchFamily returns ONE family out of a batch payload. The bool answers a DIFFERENT question
// from the bytes: did the batch CARRY this family? A reader must report an absent family as absent,
// never as empty state.
func recordBatchFamily(payload []byte, familyKey string) (json.RawMessage, bool, error) {
	
var p recordBatchPayload
	
if err := json.Unmarshal(payload, &p); err != nil {
	
	
return nil, false, fmt.Errorf("record batch: malformed payload: %w", err)
	
}
	
raw, carried := p.Families[familyKey]
	
if !carried {
	
	
return nil, false, nil
	
}
	
if len(raw) == 0 {
	
	
return nil, true, fmt.Errorf("record batch: family %q was carried with no bytes", familyKey)
	
}
	
return raw, true, nil
}

// recordBatchSequences reports the per-family sequences the batch recorded, so a reader can order a
// family against its own history instead of trusting a clock.
func recordBatchSequences(payload []byte) (map[string]uint64, error) {
	
var p recordBatchPayload
	
if err := json.Unmarshal(payload, &p); err != nil {
	
	
return nil, fmt.Errorf("record batch: malformed payload: %w", err)
	
}
	
return p.Sequences, nil
}

func recordBatchFamilyIsDeclared(key string) bool {
	
for _, f := range PrimaryRecordFamilies {
	
	
if f.Key == key {
	
	
	
return true
	
	
}
	
}
	
return false
}

// saveRecordBatchLocked writes ONE record carrying SEVERAL families (A5, plan 15): ONE fee per
// cadence instead of one per family.
//
// WHY IT GOES THROUGH THE AUDIT DOOR: the batch prefix is a DECLARED NOTE PURPOSE but NOT a state
// family, and `dispatchBlockchainSnapshot` must keep REFUSING anything that is not a family (a record
// outside the family set could never be read back). `sendAuditNoteStream` accepts any declared prefix
// and gives the batch its OWN nonce sequence, which is exactly what a transport envelope needs.
//
// Each family is marshalled ONCE, the family OWN sequence travels INSIDE the payload, and an
// undeclared family key is refused by the payload owner rather than written into a record no reader
// could resolve.
func (l *Lobby) saveRecordBatchLocked(families map[string]any) {
	if len(families) == 0 {
		return
	}
	raw := make(map[string]json.RawMessage, len(families))
	seqs := make(map[string]uint64, len(families))
	for key, state := range families {
		data, err := json.Marshal(state)
		if err != nil {
			log.Printf("[CACHE ERROR] Record batch: family %s could not be marshalled (%v); the batch is NOT written rather than written incomplete.\n", key, err)
			return
		}
		raw[key] = json.RawMessage(data)
		seqs[key] = recordNonceNext(key)
	}
	nonce := recordNonceNext(NotePrefixBatchSnapshot)
	payload, err := buildRecordBatchPayload(nonce, raw, seqs)
	if err != nil {
		log.Printf("[CACHE ERROR] Record batch refused: %v\n", err)
		return
	}
	l.sendAuditNoteStream(NotePrefixBatchSnapshot, payload)
}
