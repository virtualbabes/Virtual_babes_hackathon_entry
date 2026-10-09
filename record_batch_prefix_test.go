//go:build !js && !wasm

package main

import "testing"

// TestTheBatchEnvelopeIsNotARecordFamily pins the A5 design guarantee: the batch envelope is
// TRANSPORT. If it ever became a family it would be a SECOND OWNER of every fact it carries - the
// exact defect the record set exists to prevent - so the batch resolver accepts it while the family
// resolver still REFUSES it, and that pair is pinned here.
func TestTheBatchEnvelopeIsNotARecordFamily(t *testing.T) {
	
if !RecordPrefixIsBatch(NotePrefixBatchSnapshot) {
	
	
t.Fatal("the batch envelope prefix must be accepted by the batch resolver, or the one cadence could never write a batch")
	
}
	
// Negative controls: a family prefix is NOT a batch, and an undeclared prefix is neither.
	
if RecordPrefixIsBatch(NotePrefixEconomySnapshot) {
	
	
t.Error("a state family prefix must never be treated as the batch envelope")
	
}
	
if RecordPrefixIsBatch("VBT_NOT_DECLARED:") {
	
	
t.Error("an undeclared prefix must not be accepted as the batch envelope")
	
}
	
if _, ok := RecordFamilyKeyForPrefix(NotePrefixBatchSnapshot); ok {
	
	
t.Error("the batch envelope resolved to a FAMILY: a batch owns no fact, so it must stay outside PrimaryRecordFamilies")
	
}
	
if _, ok := RecordFamilyKeyForPrefix(NotePrefixEconomySnapshot); !ok {
	
	
t.Error("a declared family must still resolve to its key: the refusal must not have widened")
	
}
	
if _, ok := RecordFamilyKeyForPrefix("VBT_NOT_DECLARED:"); ok {
	
	
t.Error("an undeclared prefix must still be REFUSED by the family resolver")
	
}
}
