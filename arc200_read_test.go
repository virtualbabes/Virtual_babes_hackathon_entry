//go:build !js && !wasm

package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// These tests pin the Voi ARC-200 READ paths against the wire shapes MEASURED on
// 2026-09-17 (Problems.md §36). Every fixture is copied from a real response:
//
//	/arc200/balances?accountId=<vault>&contractId=40227315
//	  -> {"balances":[{"name":"Virtual Babes VOiconomy","symbol":"VBV",
//	       "balance":"7000000000","decimals":6,"accountId":"<vault>",
//	       "contractId":40227315}],"total-count":1,"current-round":22767990}
//
//	/arc200/transfers?contractId=40227315
//	  -> {"transfers":[{"transactionId":"...","contractId":40227315,
//	       "timestamp":1776525958,"round":17969806,"sender":"...",
//	       "receiver":"...","amount":"7000000000"}]}
//	     ^ sender/receiver/amount — NO from, NO to, NO note, NO metadata.
const (
	testVaultAddr    = "2A3NWJMYQ7AWJ5KIMYJKWEZS37FWND3PEXT3XPS6QONL6MDRJO257C7WEI"
	testVbvContract  = "40227315"
	arc200TestPayer  = "CVVDSX3ZNMY437AD2KY5FB4Q3HQRKVOWAQ7WC53APHJIAU6DVMXJG7HZCI"
	testTransferTxID = "VKOZHCN5Q6A2JLDQ3EZRI2HREH3EP7PELLIGXPCK5HLW6AHEIOBQ"
)

func TestParseARC200AmountRangeChecks(t *testing.T) {
	ok := []struct {
		in   string
		want uint64
	}{
		{"7000000000", 7000000000},
		{"0", 0},
		{"18446744073709551615", 18446744073709551615}, // MaxUint64, 20 digits
	}
	for _, c := range ok {
		got, err := parseARC200Amount(c.in)
		if err != nil || got != c.want {
			t.Errorf("parseARC200Amount(%q) = (%d, %v); want (%d, nil)", c.in, got, err, c.want)
		}
	}

	bad := []struct {
		in     string
		reason string
	}{
		{"", "empty"},
		{"   ", "empty"},
		{"1.5", "decimal integer"},
		{"0x10", "decimal integer"},
		// The 2^256-1 mint/burn sentinel this indexer DOES emit in transfer
		// records. Without the range check it would enter the ledger as a
		// fabricated balance.
		{"115792089237316195423570985008687907853269984665640564039457584007913129639935", "uint64"},
		{"18446744073709551616", "range"},
	}
	for _, c := range bad {
		got, err := parseARC200Amount(c.in)
		if err == nil {
			t.Errorf("parseARC200Amount(%q) = (%d, nil); want a refusal (%s)", c.in, got, c.reason)
			continue
		}
		if !strings.Contains(strings.ToLower(err.Error()), c.reason) {
			t.Errorf("parseARC200Amount(%q) refusal %q should mention %q", c.in, err.Error(), c.reason)
		}
	}
}

// balanceServer serves the measured /arc200/balances shape with a configurable row.
func balanceServer(t *testing.T, row map[string]any, status int) (*httptest.Server, *string) {
	t.Helper()
	var gotQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		if r.URL.Path != "/arc200/balances" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		rows := []map[string]any{}
		if row != nil {
			rows = append(rows, row)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"balances": rows, "total-count": len(rows), "current-round": 22767990})
	}))
	t.Cleanup(ts.Close)
	return ts, &gotQuery
}

func TestFetchARC200BalanceReadsTheMeasuredShape(t *testing.T) {
	row := map[string]any{
		"name": "Virtual Babes VOiconomy", "symbol": "VBV", "balance": "7000000000",
		"decimals": 6, "accountId": testVaultAddr, "contractId": 40227315,
	}
	ts, gotQuery := balanceServer(t, row, http.StatusOK)

	bal, symbol, err := (&OracleService{}).FetchARC200Balance(NetworkConfig{IndexerURLs: []string{ts.URL}}, testVaultAddr, testVbvContract)
	if err != nil {
		t.Fatalf("the measured balance shape must decode: %v", err)
	}
	if bal != 7_000_000_000 {
		t.Errorf("balance = %d; want 7000000000 (7,000.000000 $VBV)", bal)
	}
	if symbol != "VBV" {
		t.Errorf("symbol = %q; want VBV", symbol)
	}
	if !strings.Contains(*gotQuery, "accountId="+testVaultAddr) || !strings.Contains(*gotQuery, "contractId="+testVbvContract) {
		t.Errorf("the request must name the holder and the contract, got query %q", *gotQuery)
	}
}

func TestFetchARC200BalanceRefusals(t *testing.T) {
	cases := []struct {
		name      string
		row       map[string]any
		status    int
		accountID string
		contract  string
		wantIn    string
	}{
		{
			name:   "another contract's row is not this token",
			row:    map[string]any{"balance": "7000000000", "accountId": testVaultAddr, "contractId": 49001530},
			status: http.StatusOK, accountID: testVaultAddr, contract: testVbvContract,
			wantIn: "holds no balance row",
		},
		{
			name:   "another account's row is not this holder",
			row:    map[string]any{"balance": "7000000000", "accountId": arc200TestPayer, "contractId": 40227315},
			status: http.StatusOK, accountID: testVaultAddr, contract: testVbvContract,
			wantIn: "holds no balance row",
		},
		{
			name:   "a uint256 sentinel is refused, never truncated",
			row:    map[string]any{"balance": "115792089237316195423570985008687907853269984665640564039457584007913129639935", "accountId": testVaultAddr, "contractId": 40227315},
			status: http.StatusOK, accountID: testVaultAddr, contract: testVbvContract,
			wantIn: "uint64",
		},
		{
			name:   "a base that does not serve the ARC-200 path says so",
			row:    nil,
			status: http.StatusNotFound, accountID: testVaultAddr, contract: testVbvContract,
			wantIn: "no configured indexer base serves",
		},
		{
			name:   "an empty account is refused",
			row:    nil,
			status: http.StatusOK, accountID: "", contract: testVbvContract,
			wantIn: "account id is required",
		},
		{
			name:   "the zero contract can never resolve a token",
			row:    nil,
			status: http.StatusOK, accountID: testVaultAddr, contract: "0",
			wantIn: "non-empty contract id",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ts, _ := balanceServer(t, c.row, c.status)
			_, _, err := (&OracleService{}).FetchARC200Balance(NetworkConfig{IndexerURLs: []string{ts.URL}}, c.accountID, c.contract)
			if err == nil {
				t.Fatalf("want a refusal mentioning %q, got success", c.wantIn)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(c.wantIn)) {
				t.Errorf("refusal %q should mention %q", err.Error(), c.wantIn)
			}
		})
	}
}

// voiVerificationServer serves the TWO endpoints the Voi verification branch now
// requires: the ARC-200 transfer registry (the payment) and the standard indexer
// transaction (the note).
func voiVerificationServer(t *testing.T, transfer map[string]any, note []byte, noteStatus int) (*httptest.Server, *string) {
	t.Helper()
	var gotTransferQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/arc200/transfers":
			gotTransferQuery = r.URL.RawQuery
			rows := []map[string]any{}
			if transfer != nil {
				rows = append(rows, transfer)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"transfers": rows})
		case strings.HasPrefix(r.URL.Path, "/v2/transactions/"):
			if noteStatus != http.StatusOK {
				w.WriteHeader(noteStatus)
				return
			}
			// Go encodes []byte as base64, exactly as the indexer does.
			_ = json.NewEncoder(w).Encode(map[string]any{"transaction": map[string]any{"note": note}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(ts.Close)
	return ts, &gotTransferQuery
}

func voiVerifyLobby(ts *httptest.Server) *Lobby {
	return &Lobby{
		vaultAddress: testVaultAddr,
		availableNetworks: map[string]NetworkConfig{
			"Voi Mainnet": {NetworkName: "Voi Mainnet", AssetID: testVbvContract, AppID: testVbvContract, IndexerURLs: []string{ts.URL}},
		},
	}
}

// measuredTransfer is the exact /arc200/transfers row shape, with overrides.
func measuredTransfer(over map[string]any) map[string]any {
	row := map[string]any{
		"transactionId": testTransferTxID,
		"contractId":    40227315,
		"timestamp":     int64(1776525958),
		"sender":        arc200TestPayer,
		"receiver":      testVaultAddr,
		"amount":        "7000000000",
	}
	for k, v := range over {
		if v == nil {
			delete(row, k)
			continue
		}
		row[k] = v
	}
	return row
}

func TestVoiVerificationUsesTheMeasuredWireShape(t *testing.T) {
	svc := &OracleService{}

	t.Run("a matching transfer with the matching note verifies", func(t *testing.T) {
		ts, gotQuery := voiVerificationServer(t, measuredTransfer(nil), []byte(NotePrefixArenaTournamentBuyIn+`{"ok":true}`), http.StatusOK)
		ok, when, err := svc.VerifyBuyInTransaction(voiVerifyLobby(ts), "VOI", testTransferTxID, 100, testVbvContract, arc200TestPayer, testVaultAddr, NotePrefixArenaTournamentBuyIn)
		if err != nil {
			t.Fatalf("want a verified payment, got error %v", err)
		}
		if !ok {
			t.Fatal("the measured transfer + note must verify")
		}
		if when != 1776525958 {
			t.Errorf("the on-chain timestamp = %d; want 1776525958", when)
		}
		// The lookup must be scoped by SENDER and CONTRACT: the old code asked
		// ?transactionId=, a filter this API IGNORES (measured).
		if !strings.Contains(*gotQuery, "from="+arc200TestPayer) || !strings.Contains(*gotQuery, "contractId="+testVbvContract) {
			t.Errorf("the transfer lookup must be scoped by sender+contract, got %q", *gotQuery)
		}
		if strings.Contains(*gotQuery, "transactionId=") {
			t.Errorf("the lookup must not rely on the ignored transactionId filter, got %q", *gotQuery)
		}
	})

	cases := []struct {
		name     string
		transfer map[string]any
		note     []byte
		noteStat int
		wantErr  string
	}{
		{name: "a note carrying another purpose", transfer: measuredTransfer(nil), note: []byte("VBT_OTHER_PURPOSE:{}"), noteStat: http.StatusOK},
		{name: "a transfer with no note at all", transfer: measuredTransfer(nil), note: nil, noteStat: http.StatusOK},
		{name: "a transaction the standard indexer has not seen", transfer: measuredTransfer(nil), note: nil, noteStat: http.StatusNotFound, wantErr: "not indexed"},
		{name: "an unrelated transaction id", transfer: measuredTransfer(map[string]any{"transactionId": "SOMEONEELSESTX"}), note: []byte(NotePrefixArenaTournamentBuyIn), noteStat: http.StatusOK},
		{name: "a payment to somebody else", transfer: measuredTransfer(map[string]any{"receiver": arc200TestPayer}), note: []byte(NotePrefixArenaTournamentBuyIn), noteStat: http.StatusOK},
		{name: "a payment from somebody else", transfer: measuredTransfer(map[string]any{"sender": "SOMEONEELSE"}), note: []byte(NotePrefixArenaTournamentBuyIn), noteStat: http.StatusOK},
		{name: "an amount below the price", transfer: measuredTransfer(map[string]any{"amount": "1"}), note: []byte(NotePrefixArenaTournamentBuyIn), noteStat: http.StatusOK},
		{name: "the uint256 sentinel is not a balance", transfer: measuredTransfer(map[string]any{"amount": "115792089237316195423570985008687907853269984665640564039457584007913129639935"}), note: []byte(NotePrefixArenaTournamentBuyIn), noteStat: http.StatusOK},
		{name: "a different token contract", transfer: measuredTransfer(map[string]any{"contractId": 49001530}), note: []byte(NotePrefixArenaTournamentBuyIn), noteStat: http.StatusOK},
		{name: "no transfer at all", transfer: nil, note: []byte(NotePrefixArenaTournamentBuyIn), noteStat: http.StatusOK},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ts, _ := voiVerificationServer(t, c.transfer, c.note, c.noteStat)
			ok, _, err := svc.VerifyBuyInTransaction(voiVerifyLobby(ts), "VOI", testTransferTxID, 100, testVbvContract, arc200TestPayer, testVaultAddr, NotePrefixArenaTournamentBuyIn)
			if c.wantErr != "" {
				if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(c.wantErr)) {
					t.Fatalf("want an error mentioning %q, got (ok=%v, err=%v)", c.wantErr, ok, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("a mismatch must be a plain NOT-VERIFIED, not an error: %v", err)
			}
			if ok {
				t.Fatal("this must NOT verify")
			}
		})
	}
}

func TestCheckpointReadUsesBase64NotePrefixAndAssertsTheSender(t *testing.T) {
	payload := gzipB64(t, []byte(`{"balances":{"0xabc":17}}`))
	note := NotePrefixStateSnapshot + payload

	var gotPath, gotQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"current-round": 22768037,
			"transactions": []map[string]any{
				// The vault's own checkpoint (an app-call: no receiver).
				{"id": "TXVAULT", "note": []byte(note), "sender": testVaultAddr, "round-time": 1700, "tx-type": "appl"},
				// A third party writing the same prefix MUST be ignored.
				{"id": "TXTHIRDPARTY", "note": []byte(note), "sender": arc200TestPayer, "round-time": 1900, "tx-type": "appl"},
			},
		})
	}))
	t.Cleanup(ts.Close)

	l := &Lobby{}
	got, err := l.indexerCheckpointTransfers(NetworkConfig{IndexerURLs: []string{ts.URL}}, testVaultAddr, NotePrefixStateSnapshot)
	if err != nil {
		t.Fatalf("the measured transaction shape must decode: %v", err)
	}

	// The path is the standard indexer's account-transaction endpoint — the only
	// one that carries notes (the ARC-200 transfer registry carries none).
	if gotPath != "/v2/accounts/"+testVaultAddr+"/transactions" {
		t.Errorf("path = %q; want the account-transactions endpoint", gotPath)
	}

	// The value must be the BASE64 of the note bytes: the indexer refuses plain
	// text with "unable to parse base64 data". That is the measured defect.
	q, err := url.ParseQuery(gotQuery)
	if err != nil {
		t.Fatalf("the query must parse: %v", err)
	}
	raw := q.Get("note-prefix")
	if raw == "" {
		t.Fatalf("the read must send note-prefix, got query %q", gotQuery)
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("note-prefix %q must be base64: %v", raw, err)
	}
	if string(decoded) != NotePrefixStateSnapshot {
		t.Errorf("note-prefix decodes to %q; want %q", string(decoded), NotePrefixStateSnapshot)
	}
	if q.Get("limit") == "" {
		t.Errorf("the read must bound the result count, got query %q", gotQuery)
	}

	// Only the vault's own transaction survives the sender assertion.
	if len(got) != 1 {
		t.Fatalf("got %d candidate checkpoints; want 1 (the third party's must be skipped)", len(got))
	}
	if got[0].TransactionID != "TXVAULT" {
		t.Errorf("surviving tx = %q; want TXVAULT", got[0].TransactionID)
	}
	if got[0].From != testVaultAddr || got[0].To != testVaultAddr {
		t.Errorf("the projection must carry the asserted vault endpoint, got from=%q to=%q", got[0].From, got[0].To)
	}
	if got[0].Metadata != note {
		t.Errorf("the projection must carry the transaction's note, got %q", got[0].Metadata)
	}

	// …and the payload it carries still decodes through the guard.
	data, err := decodeSnapshotPayload(NotePrefixStateSnapshot, strings.TrimPrefix(got[0].Metadata, NotePrefixStateSnapshot))
	if err != nil {
		t.Fatalf("the projected checkpoint must decode: %v", err)
	}
	if string(data) != `{"balances":{"0xabc":17}}` {
		t.Errorf("decoded payload = %s", data)
	}
}

func TestCheckpointReadRefusesBadInputs(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(ts.Close)

	l := &Lobby{}
	cfg := NetworkConfig{IndexerURLs: []string{ts.URL}}

	_, err := l.indexerCheckpointTransfers(cfg, testVaultAddr, NotePrefixStateSnapshot)
	if err == nil || !strings.Contains(err.Error(), "no configured indexer base serves") {
		t.Fatalf("a 404 everywhere must be reported as an unserved path, got %v", err)
	}
	if _, err := l.indexerCheckpointTransfers(cfg, "", NotePrefixStateSnapshot); err == nil {
		t.Error("an empty vault address must be refused")
	}
	if _, err := l.indexerCheckpointTransfers(cfg, testVaultAddr, ""); err == nil {
		t.Error("an empty purpose prefix must be refused")
	}
	if _, err := l.indexerCheckpointTransfers(NetworkConfig{}, testVaultAddr, NotePrefixStateSnapshot); err == nil {
		t.Error("a config with no indexer base must be refused")
	}
	if isVaultAuthoredCheckpoint("", testVaultAddr) || isVaultAuthoredCheckpoint(testVaultAddr, "") {
		t.Error("an empty sender or vault cannot be asserted")
	}
	if !isVaultAuthoredCheckpoint(strings.ToLower(testVaultAddr), testVaultAddr) {
		t.Error("the sender assertion must be case-insensitive")
	}
}
