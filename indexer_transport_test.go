//go:build !js && !wasm

package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// ════════════════════════════════════════════════════════════════════════════
// THE INDEXER TRANSPORT — failover semantics (Problems.md §15)
//
// Every chain read in this build goes through indexerGet. It used to exist
// twice (`OracleService.IndexerRequest` and `Lobby.indexerRequest`), each copy
// returning the FIRST response that was not a 429 or a 5xx — including a 404.
//
// A base that does not serve a path answers 404 for EVERY request to it, so the
// first base's 404 ended the chain and no later base was ever asked. That is the
// difference between "this base does not route /arc200/*" and "the resource
// does not exist", and the two must not be conflated: the Voi base this build is
// configured with answers 200 for /v2/accounts and 404 for every /arc200/* path
// on the SAME host (measured live), while all thirteen ARC-200 readers ask that
// same base.
//
// These tests pin the contract:
//   - a 404 advances to the NEXT base and is not retried on the same one;
//   - the most recent 404 is still RETURNED (not turned into an error) when
//     every base lacks the path, because two callers read a 404 as "not found
//     yet" rather than as a failure;
//   - 5xx/429 keep the original retry-and-fail behaviour;
//   - an empty base list answers with an honest error instead of a nil wrap.
// ════════════════════════════════════════════════════════════════════════════

// countingIndexer is a base URL that always answers the same status and counts
// how many times it was asked.
func countingIndexer(t *testing.T, status int, body string) (*httptest.Server, *int32) {
	t.Helper()
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func TestIndexerReturnsTheNotFoundAnswerWhenNoBaseServesThePath(t *testing.T) {
	only, hits := countingIndexer(t, http.StatusNotFound, `{"message":"Not Found"}`)

	resp, err := indexerGet([]string{only.URL}, "/arc200/transfers?contractId=1")
	if err != nil {
		t.Fatalf("a 404 from every base must be RETURNED as a response (two callers read it as \"not found yet\"), got error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d; want %d", resp.StatusCode, http.StatusNotFound)
	}
	// The retained response is the real one from the last base, so its body must
	// still be readable: a caller that inspects a 404 body must not be handed a
	// closed stream.
	if _, readErr := io.ReadAll(resp.Body); readErr != nil {
		t.Errorf("the retained 404 body must stay readable: %v", readErr)
	}
	if n := atomic.LoadInt32(hits); n != 1 {
		t.Errorf("the single base was asked %d times; want 1 (the old code did not retry a 404 either)", n)
	}
}

func TestIndexerRetriesServerErrorsAndFailsAfterEveryBase(t *testing.T) {
	broken, hits := countingIndexer(t, http.StatusInternalServerError, `{"message":"boom"}`)

	resp, err := indexerGet([]string{broken.URL}, "/v2/accounts")
	if err == nil {
		t.Fatalf("every base answered 5xx; want an error, got status %d", resp.StatusCode)
	}
	if resp != nil {
		t.Error("a cluster that failed everywhere must not return a response")
	}
	if n := atomic.LoadInt32(hits); n != 3 {
		t.Errorf("attempts = %d; want 3 (the retry budget per base is unchanged)", n)
	}
}

func TestIndexerSaysSoWhenNoBaseIsConfigured(t *testing.T) {
	resp, err := indexerGet(nil, "/v2/accounts")
	if err == nil {
		t.Fatalf("no configured base must be an error, got status %d", resp.StatusCode)
	}
	if !strings.Contains(err.Error(), "no indexer base") {
		t.Errorf("error = %q; want it to name the real cause (no configured base), not a wrapped nil", err.Error())
	}
}

// TestCheckpointReaderAndOracleShareOneTransport proves the two entry points
// cannot drift: the checkpoint readers (Lobby.indexerRequest) and the oracle
// readers (OracleService.IndexerRequest) must resolve a path the same way.
func TestCheckpointReaderAndOracleShareOneTransport(t *testing.T) {
	stale, _ := countingIndexer(t, http.StatusNotFound, `{"message":"Not Found"}`)
	live, liveHits := countingIndexer(t, http.StatusOK, `{"transfers":[]}`)
	cfg := NetworkConfig{NetworkName: "Voi Mainnet", IndexerURLs: []string{stale.URL, live.URL}}

	l := &Lobby{}
	resp, err := l.indexerRequest(cfg, "/arc200/transfers?note_prefix=VBT_STATE_SNAPSHOT:")
	if err != nil {
		t.Fatalf("Lobby.indexerRequest: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Lobby.indexerRequest status = %d; want %d (it must fail over like the oracle path)", resp.StatusCode, http.StatusOK)
	}
	resp.Body.Close()

	s := &OracleService{}
	resp2, err := s.IndexerRequest(nil, cfg, "/arc200/transfers?note_prefix=VBT_STATE_SNAPSHOT:")
	if err != nil {
		t.Fatalf("OracleService.IndexerRequest: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("OracleService.IndexerRequest status = %d; want %d", resp2.StatusCode, http.StatusOK)
	}

	if n := atomic.LoadInt32(liveHits); n != 2 {
		t.Errorf("the serving base was asked %d times; want 2 (once per entry point)", n)
	}
}

func TestIndexerAdvancesPastABaseThatDoesNotServeThePath(t *testing.T) {
	stale, staleHits := countingIndexer(t, http.StatusNotFound, `{"message":"Not Found"}`)
	live, liveHits := countingIndexer(t, http.StatusOK, `{"transfers":[]}`)

	resp, err := indexerGet([]string{stale.URL, live.URL}, "/arc200/transfers?contractId=1")
	if err != nil {
		t.Fatalf("an unserved path on the first base must not end the failover chain: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d; want %d from the base that DOES serve the path", resp.StatusCode, http.StatusOK)
	}
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		t.Fatalf("the answer from the serving base must be readable: %v", readErr)
	}
	if !strings.Contains(string(body), "transfers") {
		t.Errorf("body = %q; want the serving base's payload", string(body))
	}

	if n := atomic.LoadInt32(staleHits); n != 1 {
		t.Errorf("the base that does not serve the path was asked %d times; want 1 (a routing answer must not be retried)", n)
	}
	if n := atomic.LoadInt32(liveHits); n != 1 {
		t.Errorf("the serving base was asked %d times; want 1", n)
	}
}

// TestIndexerGetReadsABodyLargerThanTheSocketBuffer — the response is handed to a
// caller that decodes it AFTER indexerGet returns, so its body must survive the
// attempt context's cancellation.
//
// MEASURED DEFECT (2026-09-19): `attemptCancel()` ran IMMEDIATELY after `Do`, so a
// Body still bound to the cancelled context answered `context canceled` for anything
// the socket had not already buffered. Every earlier test used a tiny body, so the
// transport looked correct — but the record reader returns tens of KB, so it could
// not read a real record at all. This test fails before the fix and passes after it.
func TestIndexerGetReadsABodyLargerThanTheSocketBuffer(t *testing.T) {
	// ~128 KB of low-redundancy bytes: far above any socket buffer, and it stays large
	// on the wire because there is no HTTP-layer compression to hide it.
	var sb strings.Builder
	seed := uint64(88172645463325252)
	for i := 0; i < 8192; i++ {
		seed = seed*6364136223846793005 + 1442695040888963407
		fmt.Fprintf(&sb, "%016x", seed)
	}
	big := sb.String()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"blob":"` + big + `"}`))
	}))
	t.Cleanup(ts.Close)

	resp, err := indexerGet([]string{ts.URL}, "/v2/blob")
	if err != nil {
		t.Fatalf("indexerGet: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("a %d byte response must be readable AFTER indexerGet returns: %v", len(big), err)
	}
	if !strings.Contains(string(body), big[:64]) {
		t.Error("the buffered body must carry what the server sent")
	}
}
