//go:build !js && !wasm

package main

// -- THE SOURCE RULE (A5 step 4, first half) --------------------------------
//
// A batch read is a FALLBACK, and a fallback that does not NAME its source hides a broken writer: a
// family that stopped being written directly would keep reading correctly out of an old batch for
// ever, and nothing would say so. The rule below therefore answers TWO things at once - which record
// wins, and which record answered.

const (
	
// RecordSourceFamilyRecord is a record written under the family OWN prefix.
	
RecordSourceFamilyRecord = "family_record"
	
// RecordSourceBatch is a family carried inside the batch envelope.
	
RecordSourceBatch = "batch"
	
// RecordSourceNone means NEITHER candidate was complete: a NAMED GAP, never an empty state.
	
RecordSourceNone = "none"
)

// recordSourceOrdering is the ONE rule that chooses between a family record and a batch record.
//
// ORDERING IS BY NONCE SEQUENCE, NEVER BY TIMESTAMP. The batch and the family record are written by
// the SAME vault, so their wall-clock times can be equal - or a batch can be later while carrying
// OLDER state for that family. A clock cannot tell those apart, and choosing wrongly restores older
// state over newer state without raising anything.
//
// The signature carries NO TIME ARGUMENT on purpose: that ABSENCE is the guarantee, and a test pins
// it with reflection so a clock cannot be reintroduced by a later edit.
//
// A tie is won by the FAMILY RECORD, because it is the more specific owner of that state: a tie means
// one cadence wrote both, and the direct record is the authoritative one.
func recordSourceOrdering(familySeq uint64, familyComplete bool, batchSeq uint64, batchComplete bool) (string, bool) {
	
switch {
	
case familyComplete && batchComplete:
	
	
if familySeq >= batchSeq {
	
	
	
return RecordSourceFamilyRecord, true
	
	
}
	
	
return RecordSourceBatch, true
	
case familyComplete:
	
	
return RecordSourceFamilyRecord, true
	
case batchComplete:
	
	
return RecordSourceBatch, true
	
}
	
return RecordSourceNone, false
}
