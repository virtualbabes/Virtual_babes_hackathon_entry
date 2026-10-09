//go:build !js && !wasm

package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
)

// ════════════════════════════════════════════════════════════════════════════
// SNAPSHOT READ GUARD — the ONE place a state checkpoint is decoded.
//
// Three readers reconstruct state from on-chain checkpoint notes
// (loadLeaderboard, loadEconomyState, loadBlockchainStateSnapshotLocked). Each
// decoded the indexer payload itself, so each had the same two defects:
//
//  1. UNBOUNDED DECOMPRESSION. `decompressedData.ReadFrom(gzr)` had no cap, so a
//     crafted or corrupt note could expand without limit and exhaust memory
//     during BOOT — before any rate limiter, wallet check or circuit breaker
//     exists. Decompression is now bounded at BOTH stages.
//
//  2. THE VAULT SCOPING WAS ONLY IN THE QUERY STRING. The reader trusted
//     `from=<vault>&to=<vault>` in the URL and then matched on the note prefix
//     alone. A query string is not an authorisation decision: if the indexer
//     ignored a parameter, or a node returned a broader result set, a transfer
//     a THIRD PARTY authored could have been parsed as engine state. The
//     scoping is now asserted in code.
//
// A snapshot is only ever written vault → vault, so that is what is required.
// The assertion is applied WHEN THE INDEXER SUPPLIES the fields. It is
// deliberately not a hard requirement that `from`/`to` be present, because
// whether this indexer's `/arc200/transfers` projection always carries them
// could NOT be confirmed against a live response in this session (the endpoint
// 404s for every contract id available here). Requiring a field that might be
// absent would turn a hardening pass into a boot-time data-loss risk. Instead:
// absent fields are LOGGED ONCE, so the protection's inactivity is visible
// rather than assumed. See AI-Brain/Problems.md §15.
// ════════════════════════════════════════════════════════════════════════════

const (
	// maxSnapshotEncodedBytes caps the base64 text lifted from the note. A note
	// is a few KB in practice; 8 MiB is already far beyond any legitimate value.
	maxSnapshotEncodedBytes = 8 << 20
	// maxSnapshotDecodedBytes caps the gunzipped JSON. The real checkpoint is a
	// whole leaderboard/economy map; 64 MiB is generous while still bounded.
	maxSnapshotDecodedBytes = 64 << 20
)

// snapshotTransfer is the projection of an indexer transfer a checkpoint reader
// needs. From/To are decoded ONLY so the vault scoping can be asserted in code.
type snapshotTransfer struct {
	TransactionID string `json:"transactionId"`
	From          string `json:"from"`
	To            string `json:"to"`
	Metadata      string `json:"metadata"`
	Timestamp     int64  `json:"timestamp"`
}

// snapshotScopeWarnOnce keeps the "indexer omitted from/to" warning to one line
// per process instead of one per snapshot read.
var snapshotScopeWarnOnce sync.Once

// decodeSnapshotPayload base64-decodes and gunzips a checkpoint note with hard
// caps on BOTH stages. Every failure returns its reason; no partial payload is
// ever returned and no art/state is invented.
func decodeSnapshotPayload(prefix, encoded string) ([]byte, error) {
	if encoded == "" {
		return nil, fmt.Errorf("%s snapshot payload is empty", prefix)
	}
	if len(encoded) > maxSnapshotEncodedBytes {
		return nil, fmt.Errorf("%s snapshot is %d bytes of base64, above the %d byte cap",
			prefix, len(encoded), maxSnapshotEncodedBytes)
	}

	decodedBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("%s snapshot base64 decode failed: %w", prefix, err)
	}

	gzr, err := gzip.NewReader(bytes.NewReader(decodedBytes))
	if err != nil {
		return nil, fmt.Errorf("%s snapshot gzip reader failed: %w", prefix, err)
	}
	defer gzr.Close()

	// LimitReader at cap+1 distinguishes "exactly at the cap" from "over it".
	var buf bytes.Buffer
	n, err := buf.ReadFrom(io.LimitReader(gzr, maxSnapshotDecodedBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%s snapshot decompression failed: %w", prefix, err)
	}
	if n > maxSnapshotDecodedBytes {
		return nil, fmt.Errorf("%s snapshot expands past the %d byte cap and was refused (unbounded decompression)",
			prefix, maxSnapshotDecodedBytes)
	}
	return buf.Bytes(), nil
}

// isVaultCheckpointTransfer asserts the vault scoping IN CODE.
//
//   - prefix and vaultAddr must both be non-empty (fail closed);
//   - the note must carry the requested prefix (exact prefix test, never "");
//   - WHEN the indexer supplies From/To, BOTH must be the vault.
//
// Returns (ok, scopeKnown). scopeKnown=false means the indexer omitted the
// endpoints, so the prefix match is all that was checked — the caller may warn.
func isVaultCheckpointTransfer(tx snapshotTransfer, vaultAddr, prefix string) (ok bool, scopeKnown bool) {
	if vaultAddr == "" || prefix == "" {
		return false, true
	}
	if !strings.HasPrefix(tx.Metadata, prefix) {
		return false, true
	}

	from := strings.TrimSpace(tx.From)
	to := strings.TrimSpace(tx.To)
	if from == "" && to == "" {
		return true, false
	}
	if !strings.EqualFold(from, vaultAddr) || !strings.EqualFold(to, vaultAddr) {
		return false, true
	}
	return true, true
}

// warnSnapshotScopeUnknown records — once — that a checkpoint read could not
// verify the transfer's endpoints.
func warnSnapshotScopeUnknown(prefix string) {
	snapshotScopeWarnOnce.Do(func() {
		log.Printf("[CACHE] WARNING: the indexer response for %s carried no from/to, so the vault scoping of "+
			"checkpoint transfers could only be checked against the query parameters. The note-prefix match still "+
			"applies. Hardening is inactive for this shape (Problems.md §15).\n", prefix)
	})
}

// checkpointTransportWarnOnce keeps the "the checkpoint read path returned
// nothing" explanation to one line per process.
var checkpointTransportWarnOnce sync.Once

// warnCheckpointReadUnavailable explains WHY a checkpoint read produced nothing.
//
// Every checkpoint reader asks the indexer for `/arc200/transfers` — the ARC-200
// TRANSFER registry — while the checkpoint WRITER dispatches an
// ApplicationNoOp APP-CALL (economy_service.go sendNoteTx). That is a real
// shape difference, and it is not the only candidate: the Voi base this build is
// configured with answers 404 for EVERY `/arc200/*` path while answering 200 for
// `/v2/accounts` and `/v2/transactions` on the same host (measured live), so the
// read path may simply not be routed by any base the config lists.
//
// Both are named as candidates and NEITHER is asserted, because either one alone
// prevents reconstruction and this session could not establish on-chain which is
// true. What this function refuses to do is let the failure stay silent and be
// mistaken for "no snapshots exist". See AI-Brain/Problems.md §15.
func warnCheckpointReadUnavailable(prefix, detail string, bases []string) {
	checkpointTransportWarnOnce.Do(func() {
		log.Printf("[CACHE] WARNING: checkpoint read for %s returned nothing usable (%s). Two candidate causes, "+
			"NEITHER asserted: (a) no configured indexer serves the ARC-200 read path — the living check is that "+
			"/arc200/transfers answers 404 while /v2/accounts answers 200 on the same base; or (b) the checkpoint "+
			"WRITER dispatches an ApplicationNoOp app-call while this reader asks the ARC-200 TRANSFER registry, so "+
			"the event may never appear there. Configured base(s): %s. On-chain state reconstruction cannot fire "+
			"until one of these is resolved (Problems.md §15).\n",
			prefix, detail, strings.Join(bases, ", "))
	})
}
