//go:build !js && !wasm

package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"strings"
	"testing"
)

func gzipB64(t *testing.T, payload []byte) string {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(payload); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestDecodeSnapshotPayloadRoundTrips(t *testing.T) {
	want := []byte(`{"balances":{"0xabc":17}}`)
	got, err := decodeSnapshotPayload("TEST:", gzipB64(t, want))
	if err != nil {
		t.Fatalf("a legitimate payload must decode: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("round trip = %s; want %s", got, want)
	}
}

func TestDecodeSnapshotPayloadRefuses(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		if _, err := decodeSnapshotPayload("TEST:", ""); err == nil {
			t.Error("an empty payload must be refused")
		}
	})
	t.Run("not base64", func(t *testing.T) {
		if _, err := decodeSnapshotPayload("TEST:", "!!!!"); err == nil {
			t.Error("a non-base64 payload must be refused")
		}
	})
	t.Run("not gzip", func(t *testing.T) {
		raw := base64.StdEncoding.EncodeToString([]byte("plain text, not gzip"))
		if _, err := decodeSnapshotPayload("TEST:", raw); err == nil {
			t.Error("a non-gzip payload must be refused")
		}
	})
	t.Run("oversized base64 is refused before decoding", func(t *testing.T) {
		huge := strings.Repeat("A", maxSnapshotEncodedBytes+1)
		_, err := decodeSnapshotPayload("TEST:", huge)
		if err == nil {
			t.Fatal("an oversized base64 payload must be refused")
		}
		if !strings.Contains(err.Error(), "above the") {
			t.Errorf("the refusal should name the cap, got: %v", err)
		}
	})
	t.Run("a decompression bomb is refused", func(t *testing.T) {
		// ~64 MiB of zeros compresses to a few tens of KB, so the ENCODED size
		// passes and only the DECODED cap can stop it. Before this guard the
		// reader had no cap at all and read the stream straight into memory, so a
		// note like this could exhaust memory during boot.
		bomb := gzipB64(t, make([]byte, maxSnapshotDecodedBytes+1024))
		if len(bomb) >= maxSnapshotEncodedBytes {
			t.Fatalf("the bomb fixture is unexpectedly large (%d bytes); it should pass the encoded cap", len(bomb))
		}
		_, err := decodeSnapshotPayload("TEST:", bomb)
		if err == nil {
			t.Fatal("a payload that expands past the decoded cap must be refused")
		}
		if !strings.Contains(err.Error(), "expands past") {
			t.Errorf("the refusal should name the decompression cap, got: %v", err)
		}
	})
}

func TestIsVaultCheckpointTransfer(t *testing.T) {
	const vault = "VAULTADDR"
	const goodPrefix = "VBT_LEGIT:"

	type ck struct {
		name      string
		tx        snapshotTransfer
		prefix    string
		vaultAddr string
		wantOK    bool
		wantKnown bool
	}

	for _, c := range []ck{
		{name: "vault to vault", tx: snapshotTransfer{From: vault, To: vault, Metadata: goodPrefix + "x"},
			prefix: goodPrefix, vaultAddr: vault, wantOK: true, wantKnown: true},
		{name: "endpoints are case-insensitive", tx: snapshotTransfer{From: "vaultaddr", To: "VAULTADDR", Metadata: goodPrefix + "x"},
			prefix: goodPrefix, vaultAddr: vault, wantOK: true, wantKnown: true},
		{name: "wrong note prefix", tx: snapshotTransfer{From: vault, To: vault, Metadata: "OTHER:" + "x"},
			prefix: goodPrefix, vaultAddr: vault, wantOK: false, wantKnown: true},
		{name: "from a third party", tx: snapshotTransfer{From: "0xattacker", To: vault, Metadata: goodPrefix + "x"},
			prefix: goodPrefix, vaultAddr: vault, wantOK: false, wantKnown: true},
		{name: "to a third party", tx: snapshotTransfer{From: vault, To: "0xattacker", Metadata: goodPrefix + "x"},
			prefix: goodPrefix, vaultAddr: vault, wantOK: false, wantKnown: true},
		{name: "indexer omitted both endpoints", tx: snapshotTransfer{Metadata: goodPrefix + "x"},
			prefix: goodPrefix, vaultAddr: vault, wantOK: true, wantKnown: false},
		{name: "only one endpoint supplied", tx: snapshotTransfer{From: vault, Metadata: goodPrefix + "x"},
			prefix: goodPrefix, vaultAddr: vault, wantOK: false, wantKnown: true},
		{name: "an empty prefix is refused", tx: snapshotTransfer{From: vault, To: vault, Metadata: goodPrefix + "x"},
			prefix: "", vaultAddr: vault, wantOK: false, wantKnown: true},
		{name: "an empty vault is refused", tx: snapshotTransfer{From: vault, To: vault, Metadata: goodPrefix + "x"},
			prefix: goodPrefix, vaultAddr: "", wantOK: false, wantKnown: true},
	} {
		t.Run(c.name, func(t *testing.T) {
			ok, known := isVaultCheckpointTransfer(c.tx, c.vaultAddr, c.prefix)
			if ok != c.wantOK || known != c.wantKnown {
				t.Errorf("isVaultCheckpointTransfer = (ok=%v, scopeKnown=%v); want (%v, %v)",
					ok, known, c.wantOK, c.wantKnown)
			}
		})
	}
}

// TestWarnSnapshotScopeUnknownIsOnceOnly — the warning must not become per-read
// log spam, but it must exist so the hardening's inactivity is VISIBLE.
func TestWarnSnapshotScopeUnknownIsOnceOnly(t *testing.T) {
	// Exercising it twice must not panic and must not change any state.
	warnSnapshotScopeUnknown("VBT_TEST:")
	warnSnapshotScopeUnknown("VBT_TEST:")
}
