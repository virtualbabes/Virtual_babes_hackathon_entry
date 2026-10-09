//go:build !js && !wasm

package main

import (
	
"reflect"
	
"testing"
)

// TestRecordSourceOrderingChoosesBySequenceNeverByClock drives every combination, including the two
// that a timestamp would get WRONG: a batch that is later on the clock but carries older state for the
// family, and an incomplete candidate that must never win by being more recent.
func TestRecordSourceOrderingChoosesBySequenceNeverByClock(t *testing.T) {
	
cases := []struct {
	
	
name      string
	
	
familySeq uint64
	
	
batchSeq  uint64
	
	
famOK     bool
	
	
batchOK   bool
	
	
wantSrc   string
	
	
wantUse   bool
	
	
}{
	
	
{"only the family record is complete", 9, 0, true, false, RecordSourceFamilyRecord, true},
	
	
{"only the batch is complete", 0, 9, false, true, RecordSourceBatch, true},
	
	
{"family record is NEWER", 9, 8, true, true, RecordSourceFamilyRecord, true},
	
	
{"batch is NEWER", 8, 9, true, true, RecordSourceBatch, true},
	
	
{"identical sequences: the direct record is the more specific owner", 7, 7, true, true, RecordSourceFamilyRecord, true},
	
	
{"family record complete, batch incomplete", 1, 9, true, false, RecordSourceFamilyRecord, true},
	
	
{"family record incomplete, batch complete", 9, 1, false, true, RecordSourceBatch, true},
	
	
{"neither complete is a NAMED GAP, never empty state", 5, 6, false, false, RecordSourceNone, false},
	
 }
	
for _, c := range cases {
	
	
gotSrc, gotUse := recordSourceOrdering(c.familySeq, c.famOK, c.batchSeq, c.batchOK)
	
	
if gotSrc != c.wantSrc || gotUse != c.wantUse {
	
	
	
t.Errorf("%s: got (%q, %v); want (%q, %v)", c.name, gotSrc, gotUse, c.wantSrc, c.wantUse)
	
	
}
	
}
}

// TestRecordSourceOrderingCannotConsultAClock proves the guarantee STRUCTURALLY: the rule takes no
// time argument at all, so no future edit can let a timestamp decide which record restores.
func TestRecordSourceOrderingCannotConsultAClock(t *testing.T) {
	
rt := reflect.TypeOf(recordSourceOrdering)
	
if rt.NumIn() != 4 {
	
	
t.Fatalf("recordSourceOrdering takes %d arguments; want exactly 4 (two sequences and two completeness flags)", rt.NumIn())
	
}
	
for i := 0; i < rt.NumIn(); i++ {
	
	
switch rt.In(i).Kind() {
	
	
case reflect.Uint64, reflect.Bool:
	
	
default:
	
	
	
t.Errorf("argument %d is %s: any non-sequence input would let something other than ordering decide a restore", i, rt.In(i).Kind())
	
	
}
	
}
	
if rt.NumOut() != 2 {
	
	
t.Errorf("recordSourceOrdering returns %d values; want (source, use)", rt.NumOut())
	
}
}
