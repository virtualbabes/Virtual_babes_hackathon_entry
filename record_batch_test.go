//go:build !js && !wasm

package main

import (
	
"encoding/json"
	
"testing"
)

// TestRecordBatchRoundTripsAndDistinguishesAbsentFromEmpty proves the payload carries several families
// in ONE record, keeps their own sequences, and - the part that decides correctness - answers ABSENT
// for a family it did not carry instead of looking like empty state.
func TestRecordBatchRoundTripsAndDistinguishesAbsentFromEmpty(t *testing.T) {
	
families := map[string]json.RawMessage{
	
	
"economy": json.RawMessage(`{"balances":{"0xa":1}}`),
	
	
"clubs":   json.RawMessage(`[{"id":"club-1"}]`),
	
}
	
seqs := map[string]uint64{"economy": 7, "clubs": 3}
	
payload, err := buildRecordBatchPayload(42, families, seqs)
	
if err != nil {
	
	
t.Fatalf("build: %v", err)
	
}
	
for key, want := range families {
	
	
got, carried, err := recordBatchFamily(payload, key)
	
	
if err != nil || !carried {
	
	
	
t.Fatalf("family %s: carried=%v err=%v; want carried", key, carried, err)
	
	
}
	
	
if string(got) != string(want) {
	
	
	
t.Errorf("family %s round-tripped to %s; want %s", key, got, want)
	
	
}
	
}
	
gotSeqs, err := recordBatchSequences(payload)
	
if err != nil {
	
	
t.Fatalf("sequences: %v", err)
	
}
	
for key, want := range seqs {
	
	
if gotSeqs[key] != want {
	
	
	
t.Errorf("family %s sequence = %d; want %d", key, gotSeqs[key], want)
	
	
}
	
}
	
// THE DISCRIMINATOR: a family the batch did NOT carry is carried=false, no error, no bytes.
	
if raw, carried, err := recordBatchFamily(payload, "pets"); carried || err != nil || raw != nil {
	
	
t.Errorf("an uncarried family must be carried=false with no error and no bytes; got carried=%v err=%v raw=%v", carried, err, raw)
	
}
}

// TestRecordBatchRefusesWhatWouldRestoreAsSilentEmpty pins every refusal that protects the restore.
func TestRecordBatchRefusesWhatWouldRestoreAsSilentEmpty(t *testing.T) {
	
good := map[string]json.RawMessage{"economy": json.RawMessage(`{}`)}
	
if _, err := buildRecordBatchPayload(1, good, nil); err != nil {
	
	
t.Errorf("a declared family WITH bytes must be accepted; got %v", err)
	
}
	
if _, err := buildRecordBatchPayload(1, map[string]json.RawMessage{}, nil); err == nil {
	
	
t.Error("an empty batch must be refused: it restores nothing and looks like progress")
	
}
	
if _, err := buildRecordBatchPayload(1, map[string]json.RawMessage{"": json.RawMessage(`{}`)}, nil); err == nil {
	
	
t.Error("a family with no key must be refused")
	
}
	
if _, err := buildRecordBatchPayload(1, map[string]json.RawMessage{"not_a_family": json.RawMessage(`{}`)}, nil); err == nil {
	
	
t.Error("a family the record set does not DECLARE must be refused: batching it creates a second owner of a fact")
	
}
	
if _, err := buildRecordBatchPayload(1, map[string]json.RawMessage{"economy": json.RawMessage{}}, nil); err == nil {
	
	
t.Error("a family with no bytes must be refused: absent and empty must never share an answer")
	
}
	
if _, _, err := recordBatchFamily([]byte("{not json"), "economy"); err == nil {
	
	
t.Error("a malformed payload must be an error, never an absent family")
	
}
}
