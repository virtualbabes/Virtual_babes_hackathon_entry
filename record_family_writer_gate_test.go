//go:build !js && !wasm

package main

import (
	
"os"
	
"regexp"
	
"strings"
	
"testing"
)

// THE DECLARED-VS-WRITTEN CENSUS.
//
// A family that is DECLARED but never written is state persisted nowhere; a family with a READER
// and no writer is WORSE, because the restore path makes the code look complete. Measured
// 2026-09-19: of 22 declared families only FOUR had a writer, and `leaderboard` and
// `onboarded_wallets` were reader-only - both closed by giving them a writer on the economy
// cadence. The sixteen that are only a constant plus a family declaration are BASELINED here with
// the reason they cannot be written yet, so a hole stays COUNTED rather than hidden, and the
// baseline must SHRINK as A5 gives them writers (a baseline line that outlives its reason fails).
var recordFamilyWriterBaseline = map[string]string{
	

	

	
	
	
	
	
	
	
	
	
	
	
	
	
	
}

var prefixConstRe = regexp.MustCompile(`(?m)^\s*(NotePrefix\w+)\s*=\s*"([^"]+)"`)
var writerCallRe = regexp.MustCompile(`(?:save|write|encode|send)\w*\(`)
var readerCallRe = regexp.MustCompile(`(?:read|load|decode|dispatch|checkpoint)\w*\(`)

// classifyRecordLine names the shape of ONE line: the callee decides, because a prefix is always
// passed as an argument and the callers are named for what they do.
func classifyRecordLine(line string) string {
	
switch {
	
case writerCallRe.MatchString(line):
	
	
return "writer"
	
case readerCallRe.MatchString(line):
	
	
return "reader"
	
}
	
return "other"
}

// writerCensusFor counts the writer, reader and other lines that name ONE constant, outside the
// two files that DECLARE the vocabulary and the families (a declaration is not a writer).
func writerCensusFor(t *testing.T, constName, familyKey string) (int, int, int) {
	
t.Helper()
	
entries, err := os.ReadDir(".")
	
if err != nil {
	
	
t.Fatalf("read dir: %v", err)
	
}
	
w, r, o := 0, 0, 0
	
for _, e := range entries {
	
	
name := e.Name()
	
	
if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
	
	
	
continue
	
	
}
	
	
if name == "note_vocabulary.go" || name == "record_families.go" {
	
	
	
continue
	
	
}
	
	
raw, err := os.ReadFile(name)
		// The BATCH is a WRITER for the families it carries (A5): a MAP KEY named in a file that calls
		// saveRecordBatchLocked IS written, even though its own constant never appears on a writer line. A
		// Without this the cadence swap reports three LIVE families as unwritten - the safe direction is to
		// count them, because a FALSE negative here fails the build rather than hiding a hole. The key must be
		if strings.Contains(string(raw), "saveRecordBatchLocked(") {
			for _, line := range strings.Split(string(raw), "\n") {
				if strings.Contains(line, "\""+familyKey+"\":") {
					w++
				}
			}
		}
	
	
if err != nil {
	
	
	
t.Fatalf("read %s: %v", name, err)
	
	
}
	
	
for _, line := range strings.Split(string(raw), "\n") {
	
	
	
if !strings.Contains(line, constName) {
	
	
	
	
continue
	
	
	
}
	
	
	
switch classifyRecordLine(line) {
	
	
	
case "writer":
	
	
	
	
w++
	
	
	
case "reader":
	
	
	
	
r++
	
	
	
default:
	
	
	
	
o++
	
	
	
}
	
	
}
	
}
	
return w, r, o
}

// TestEveryDeclaredFamilyHasAWriterOrAStatedReason is the census. It fails CLOSED on a new hole and
// on a baseline line that has outlived its reason, and it fails if its OWN derivation finds
// nothing - a census that parses zero families reports a clean tree for ever.
func TestEveryDeclaredFamilyHasAWriterOrAStatedReason(t *testing.T) {
	
raw, err := os.ReadFile("note_vocabulary.go")
	
if err != nil {
	
	
t.Fatalf("read note_vocabulary.go: %v", err)
	
}
	
nameForPrefix := map[string]string{}
	
for _, m := range prefixConstRe.FindAllStringSubmatch(string(raw), -1) {
	
	
nameForPrefix[m[2]] = m[1]
	
}
	
if len(nameForPrefix) == 0 {
	
	
t.Fatal("the prefix parser found ZERO vocabulary constants")
	
}
	
if len(PrimaryRecordFamilies) == 0 {
	
	
t.Fatal("no declared families: the census has no subject")
	
}
	
familyKeys := map[string]bool{}
	
for _, f := range PrimaryRecordFamilies {
	
	
familyKeys[f.Key] = true
	
}
	
for key := range recordFamilyWriterBaseline {
	
	
if !familyKeys[key] {
	
	
	
t.Errorf("baseline entry %q is not a declared family: a baseline must not outlive the family it excuses", key)
	
	
}
	
}
	
writers, baselinedOpen, baselinedClosed := 0, 0, 0
	
for _, f := range PrimaryRecordFamilies {
	
	
constName, ok := nameForPrefix[f.Prefix]
	
	
if !ok {
	
	
	
t.Errorf("family %s names prefix %q, which no NotePrefix constant declares", f.Key, f.Prefix)
	
	
	
continue
	
	
}
	
	
w, r, _ := writerCensusFor(t, constName, f.Key)
	
	
reason, isBaselined := recordFamilyWriterBaseline[f.Key]
	
	
switch {
	
	
case w > 0:
	
	
	
writers++
	
	
	
if isBaselined {
	
	
	
	
t.Errorf("family %s HAS a writer (%d line(s)) and is still baselined as %q: delete the baseline line", f.Key, w, reason)
	
	
	
}
	
	
case !isBaselined:
	
	
	
t.Errorf("family %s has NO writer (reader lines=%d): state that is declared and never written is persisted nowhere - add a writer, or baseline it with the reason it cannot be written yet", f.Key, r)
	
	
default:
	
	
	
if r > 0 {
	
	
	
	
baselinedOpen++
	
	
	
} else {
	
	
	
	
baselinedClosed++
	
	
	
}
	
	
}
	
}
	
t.Logf("writer census: %d/%d families have a writer; %d baselined (%d with a reader already, %d declaration only)", writers, len(PrimaryRecordFamilies), len(recordFamilyWriterBaseline), baselinedOpen, baselinedClosed)
	
if writers+len(recordFamilyWriterBaseline) != len(PrimaryRecordFamilies) {
	
	
t.Errorf("the census accounts for %d of %d families: every family must be either written or baselined", writers+len(recordFamilyWriterBaseline), len(PrimaryRecordFamilies))
	
}
}

// TestTheWriterCensusClassifierReportsBothShapes is the control: a census that classifies nothing
// reports a clean tree, so the classifier is driven with one line of each shape.
func TestTheWriterCensusClassifierReportsBothShapes(t *testing.T) {
	
cases := []struct{ line, want string }{
	
	
{"l.saveBlockchainStateSnapshotLocked(NotePrefixX, state)", "writer"},
	
	
{"l.readRecordFamily(cfg, vault, NotePrefixX)", "reader"},
	
	
{"var n = NotePrefixX", "other"},
	
	
{"l.sendAuditNoteStream(NotePrefixX, data)", "writer"},
	
}
	
for _, c := range cases {
	
	
if got := classifyRecordLine(c.line); got != c.want {
	
	
	
t.Errorf("classify(%q) = %q; want %q", c.line, got, c.want)
	
	
}
	
}
}
