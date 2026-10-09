//go:build !js && !wasm

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// Ã¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢Â
// CHECKPOINT READ Ã¢â‚¬â€ the ONE place a state checkpoint is FETCHED from the chain.
//
// Why this file exists (measured 2026-09-17, Problems.md Ã‚Â§36). The three
// checkpoint readers asked the ARC-200 transfer registry for:
//
//	/arc200/transfers?contractId=<token>&from=<vault>&to=<vault>&note_prefix=X
//
// which is a request for a VAULT-TO-VAULT transfer carrying a note. Three
// independent measured facts make that unfetchable on Voi:
//
//  1. `from=<vault>&to=<vault>` is a SELF-TRANSFER. Measured:
//     /arc200/transfers?contractId=40227315&from=<vault> answers {"transfers":[]}
//     Ã¢â‚¬â€ the vault has never SENT $VBV. The writer is an APP-CALL signed BY the
//     vault, which is not a transfer at all, so the query could only ever return
//     nothing.
//  2. The transfer registry carries NO note field. Measured across all 232
//     records for the $VBV contract: the keys are exactly transactionId,
//     contractId, timestamp, round, sender, receiver, amount. The reader matched
//     `metadata` Ã¢â‚¬â€ a key this API never sends Ã¢â‚¬â€ so the prefix test could never
//     fire.
//  3. Its parameter is `note-prefix`, not `note_prefix`, and the value must be
//     the BASE64 of the note bytes. The indexer says so itself: plain text is
//     refused with "invalid input: unable to parse base64 data: 'note-prefix'".
//
// A note is a property of a TRANSACTION, so the note-prefix search belongs to
// the standard indexer's account-transaction endpoint:
//
//	/v2/accounts/<vault>/transactions?note-prefix=<base64>&limit=N
//
// whose transactions carry `id`, `note` (base64 in JSON, decoded into []byte by
// Go), `sender`, `round-time` and `tx-type`. Voi serves /v2/* on a DIFFERENT host
// from /arc200/*, and indexerGet advances to the next base on a 404, so the
// network registry must list both Ã¢â‚¬â€ pinned by networks_config_test.go.
//
// WHAT IS ASSERTED IN CODE, NEVER ASSUMED FROM THE QUERY STRING: the
// transaction's SENDER must be the vault. A query string is not an authorisation
// decision Ã¢â‚¬â€ if a base ignored the parameter, a third party's note could
// otherwise be parsed as engine state.
// Ã¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢Â

// checkpointReadLimit bounds one checkpoint read. A checkpoint is written every
// few minutes at most, so the newest 100 in a note-prefix family is far more than
// a single restore needs.
const checkpointReadLimit = 100

// indexerCheckpointTx is the standard-indexer projection of a checkpoint
// transaction.
type indexerCheckpointTx struct {
	ID        string `json:"id"`
	Note      []byte `json:"note"`
	Sender    string `json:"sender"`
	RoundTime int64  `json:"round-time"`
	TxType    string `json:"tx-type"`
}

// isVaultAuthoredCheckpoint asserts that the transaction carrying a checkpoint
// note was SIGNED BY THE VAULT. An app-call has no receiver, so the sender is the
// only participant that CAN be asserted Ã¢â‚¬â€ and it is asserted.
func isVaultAuthoredCheckpoint(sender, vaultAddr string) bool {
	if strings.TrimSpace(vaultAddr) == "" || strings.TrimSpace(sender) == "" {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(sender), strings.TrimSpace(vaultAddr))
}

// indexerCheckpointTransfers returns the vault's OWN checkpoint transactions
// carrying `prefix`, projected into the transfer shape the readers already
// consume (and which isVaultCheckpointTransfer re-checks).
//
// `From` and `To` are both the transaction's own reported sender: the checkpoint
// is authored by the vault and an app-call has no receiver, so the sole
// participant is the vault. The address is never invented Ã¢â‚¬â€ it is the one the
// indexer reported, after the assertion above.
func (l *Lobby) indexerCheckpointTransfers(cfg NetworkConfig, vaultAddr, prefix string) ([]snapshotTransfer, error) {
	if strings.TrimSpace(vaultAddr) == "" || strings.TrimSpace(prefix) == "" {
		return nil, fmt.Errorf("a checkpoint read requires both the vault address and a purpose prefix")
	}
	if len(cfg.IndexerURLs) == 0 {
		return nil, fmt.Errorf("a checkpoint read requires an indexer base; the network registry entry carries none")
	}

	// The indexer's note-prefix search takes the BASE64 of the note bytes.
	encodedPrefix := base64.StdEncoding.EncodeToString([]byte(prefix))
	path := fmt.Sprintf("/v2/accounts/%s/transactions?limit=%d&note-prefix=%s",
		url.QueryEscape(vaultAddr), checkpointReadLimit, url.QueryEscape(encodedPrefix))

	resp, err := l.indexerRequest(cfg, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no configured indexer base serves the note-prefix checkpoint path %s; the registry must include the standard indexer host", path)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the checkpoint read for %s returned HTTP %d", prefix, resp.StatusCode)
	}

	var res struct {
		Transactions []indexerCheckpointTx `json:"transactions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("the checkpoint response for %s could not be decoded: %w", prefix, err)
	}

	out := make([]snapshotTransfer, 0, len(res.Transactions))
	for _, tx := range res.Transactions {
		if !isVaultAuthoredCheckpoint(tx.Sender, vaultAddr) {
			continue // a third party cannot write engine state
		}
		out = append(out, snapshotTransfer{
			TransactionID: tx.ID,
			From:          tx.Sender,
			To:            tx.Sender,
			Metadata:      string(tx.Note),
			Timestamp:     tx.RoundTime,
		})
	}
	return out, nil
}

// Ã¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢Â
// readRecordFamily Ã¢â‚¬â€ THE reader. Every family is read HERE; no caller re-implements
// the loop, and no caller re-declares which families exist.
//
// What it does that the three per-family readers could not:
//
//  1. It reads the ENVELOPE (record_envelope.go), so a record larger than the AVM's
//     1024 byte note cap is reassembled from its chunks IN ORDER rather than being
//     impossible to write in the first place.
//  2. It orders by the SELF-NONCE, not by block time: the newest COMPLETE nonce
//     wins, and every INCOMPLETE nonce is REPORTED as a gap. Block time cannot
//     detect a missing update; a sequence can.
//  3. A newest record that cannot be DECODED is reported and the next complete one
//     is used, so corruption costs availability but never truthfulness Ã¢â‚¬â€ and it is
//     never silent.
//
// Return contract: (data, blockTime, found, err). `found=false` with err=nil means
// the chain holds no record for this family, which is NOT an error Ã¢â‚¬â€ it is the
// honest empty state a fresh world has. An err means the READ failed, which the
// callers report through warnCheckpointReadUnavailable.
// Ã¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢ÂÃ¢â€¢Â
// readRecordSelections reassembles ONE prefix into its COMPLETE records, newest nonce first. It is the
// shared body of the family reader and the batch reader (A5), so the envelope is parsed in exactly ONE
// place and the gap reporting cannot drift between them.
func (l *Lobby) readRecordSelections(cfg NetworkConfig, vaultAddr, prefix string) ([]recordSelection, error) {
	
transfers, err := l.indexerCheckpointTransfers(cfg, vaultAddr, prefix)
	
if err != nil {
	
	
return nil, err
	
}
	
notes := make([]recordNote, 0, len(transfers))
	
for _, tx := range transfers {
	
	
// SNAPSHOT READ GUARD (snapshot_guard.go): the vault scoping is asserted IN
	
	
// CODE and the gunzip is bounded. A query string is not an authorisation
	
	
// decision, so the note prefix alone is never enough.
	
	
ok, scopeKnown := isVaultCheckpointTransfer(tx, vaultAddr, prefix)
	
	
if !ok {
	
	
	
continue
	
	
}
	
	
if !scopeKnown {
	
	
	
warnSnapshotScopeUnknown(prefix)
	
	
}
	
	
rn, perr := parseRecordNote(prefix, tx.Metadata, tx.TransactionID, tx.Timestamp)
	
	
if perr != nil {
	
	
	
log.Printf("[CACHE ERROR] Rejected %s record from TxID %s: %v\n", prefix, tx.TransactionID, perr)
	
	
	
continue
	
	
}
	
	
notes = append(notes, rn)
	
}
	
complete, gaps := assembleRecords(notes)
	
for _, g := range gaps {
	
	
log.Printf("[CACHE] WARNING: a gap in the record log - %s. The newest COMPLETE record is used instead; "+
	
	
	
"a chunk may still be in flight, or its transaction failed.\n", recordGapSummary(prefix, g))
	
}
	
return complete, nil
}

// readRecordFamily is THE reader, and its CONTRACT IS UNCHANGED: (data, blockTime, found, err). It
// reports WHICH record answered through readRecordFamilyWithSource.
func (l *Lobby) readRecordFamily(cfg NetworkConfig, vaultAddr, prefix string) ([]byte, int64, bool, error) {
	
data, ts, found, _, err := l.readRecordFamilyWithSource(cfg, vaultAddr, prefix)
	
return data, ts, found, err
}

// readRecordFamilyWithSource reads a family and NAMES THE RECORD THAT ANSWERED.
//
// A batch (A5, plan 15) is a FALLBACK, and a fallback that does not name its source hides a broken
// writer: a family that stopped being written under its own prefix would keep reading correctly out
// of an old batch for ever, and nothing would say so.
//
// The winner is decided by recordSourceOrdering (record_batch_read.go), which orders by NONCE
// SEQUENCE and never by a clock - the batch and the family record are written by the SAME vault, so
// their times can be equal, or a batch can be later while carrying OLDER state for that family.
func (l *Lobby) readRecordFamilyWithSource(cfg NetworkConfig, vaultAddr, prefix string) ([]byte, int64, bool, string, error) {
	
familySelections, err := l.readRecordSelections(cfg, vaultAddr, prefix)
	
if err != nil {
	
	
return nil, 0, false, RecordSourceNone, err
	
}

	
// THE FAMILY SIDE: the newest COMPLETE record that DECODES, exactly as before. A corrupted newest
	
// record costs availability, never truthfulness, and it is never silent.
	
var (
	
	
familySel   *recordSelection
	
	
familyData  []byte
	
	
familyNonce uint64
	
	
lastErr     error
	
)
	
for i := range familySelections {
	
	
sel := &familySelections[i]
	
	
data, derr := decodeSnapshotPayload(prefix, sel.Body)
	
	
if derr != nil {
	
	
	
lastErr = derr
	
	
	
log.Printf("[CACHE ERROR] The %s record nonce %d (TxIDs %v) could not be decoded: %v\n",
	
	
	
	
prefix, sel.Nonce, sel.TxIDs, derr)
	
	
	
continue
	
	
}
	
	
familySel, familyData, familyNonce = sel, data, sel.Nonce
	
	
break
	
}

	
// THE BATCH SIDE, read ALWAYS rather than only on failure: a batch may be NEWER than the newest
	
// record written under this prefix, and only BOTH sequences can decide that.
	
var (
	
	
batchSel     *recordSelection
	
	
batchPayload []byte
	
	
batchData    json.RawMessage
	
	
batchNonce   uint64
	
	
batchOK      bool
	
)
	
if familyKey, declared := RecordFamilyKeyForPrefix(prefix); declared {
	
	
selections, berr := l.readRecordSelections(cfg, vaultAddr, NotePrefixBatchSnapshot)
	
	
if berr != nil {
	
	
	
log.Printf("[CACHE] The batch envelope could not be read while resolving %s: %v. The family record answers.\n", familyKey, berr)
	
	
} else {
	
	
	
for i := range selections {
	
	
	
	
sel := &selections[i]
	
	
	
	
payload, derr := decodeSnapshotPayload(NotePrefixBatchSnapshot, sel.Body)
	
	
	
	
if derr != nil {
	
	
	
	
	
log.Printf("[CACHE ERROR] The batch nonce %d (TxIDs %v) could not be decoded: %v\n", sel.Nonce, sel.TxIDs, derr)
	
	
	
	
	
continue
	
	
	
	
}
	
	
	
	
raw, carried, cerr := recordBatchFamily(payload, familyKey)
	
	
	
	
if cerr != nil {
	
	
	
	
	
log.Printf("[CACHE ERROR] The batch nonce %d does not resolve family %s: %v\n", sel.Nonce, familyKey, cerr)
	
	
	
	
	
continue
	
	
	
	
}
	
	
	
	
if !carried {
	
	
	
	
	
continue // this batch does not carry the family; an OLDER one may
	
	
	
	
}
	
	
	
	
batchSel, batchPayload, batchData, batchNonce, batchOK = sel, payload, raw, sel.Nonce, true
	
	
	
	
break
	
	
	
}
	
	
	
if batchOK {
	
	
	
	
// The batch records each family OWN sequence, so the writer continues ABOVE it: without
	
	
	
	
// this a restart could reuse a sequence the batch already carries.
	
	
	
	
if seqs, serr := recordBatchSequences(batchPayload); serr == nil {
	
	
	
	
	
if seen, has := seqs[familyKey]; has {
	
	
	
	
	
	
recordNonceSeed(familyKey, seen)
	
	
	
	
	
}
	
	
	
	
}
	
	
	
}
	
	
}
	
}

	
source, use := recordSourceOrdering(familyNonce, familySel != nil, batchNonce, batchOK)
	
if !use {
	
	
if lastErr != nil {
	
	
	
return nil, 0, false, RecordSourceNone, fmt.Errorf("every complete %s record failed to decode; the newest refusal was: %w", prefix, lastErr)
	
	
}
	
	
return nil, 0, false, RecordSourceNone, nil
	
}
	
if source == RecordSourceBatch && batchSel != nil {
	
	
log.Printf("[CACHE] %s resolved FROM THE BATCH (nonce %d): the batch is newer than the newest record written under the family prefix.\n",
	
	
	
prefix, batchSel.Nonce)
	
	
return []byte(batchData), batchSel.Timestamp, true, RecordSourceBatch, nil
	
}

	
if familyKey, ok := RecordFamilyKeyForPrefix(prefix); ok {
	
	
recordNonceSeed(familyKey, familyNonce)
	
}
	
if familySel.Legacy {
	
	
log.Printf("[CACHE] %s record nonce 0 is in the PRE-ENVELOPE shape; it was read and flagged "+
	
	
	
"(this build writes chunked %s notes).\n", prefix, recordEnvelopeTag)
	
}
	
return familyData, familySel.Timestamp, true, RecordSourceFamilyRecord, nil
}