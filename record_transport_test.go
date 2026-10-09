//go:build !js && !wasm

package main

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/algorand/go-algorand-sdk/v2/types"
)

// TestRecordTravelsOnAnARC200Transfer pins Brendan's rule (2026-09-19): a record is
// "a note in the arc200 transaction from faucet vault interaction … not an app
// call". The transport it replaces was `MakeApplicationNoOpTx(appID, nil, …)` — a
// bare app-call with no token movement — which is why the vault has never landed a
// single record on Voi.
func TestRecordTravelsOnAnARC200Transfer(t *testing.T) {
	vault, err := types.DecodeAddress("2A3NWJMYQ7AWJ5KIMYJKWEZS37FWND3PEXT3XPS6QONL6MDRJO257C7WEI")
	if err != nil {
		t.Fatalf("vault address: %v", err)
	}
	sp := types.SuggestedParams{Fee: 1000, FirstRoundValid: 1, LastRoundValid: 1000}
	// Built from the vocabulary OWNER rather than spelled here: note_vocabulary.go is
	// the only place a prefix may be declared (pinned by TestNoNotePrefixLiteralOutsideTheOwner).
	note := NotePrefixStateSnapshot + "AAAA"

	txn, err := buildRecordTransferTx(40227315, vault, note, sp)
	if err != nil {
		t.Fatalf("a record transaction must build: %v", err)
	}

	if txn.Type != types.ApplicationCallTx {
		t.Errorf("tx type = %v; want an application call (an ARC-200 transfer IS an app call)", txn.Type)
	}
	if txn.Sender != vault {
		t.Error("the record must be SENT BY the vault")
	}
	if string(txn.Note) != note {
		t.Errorf("note = %q; want %q", txn.Note, note)
	}
	// The application fields are EMBEDDED in types.Transaction (not a pointer), so a
	// transaction carrying no app call would report a zero ApplicationID.
	if txn.ApplicationID != 40227315 {
		t.Errorf("application id = %d; want the $VBV ARC-200 contract 40227315", txn.ApplicationID)
	}
	args := txn.ApplicationArgs
	if len(args) != 3 {
		t.Fatalf("application args = %d; want 3 (selector, recipient, amount)", len(args))
	}
	if !bytes.Equal(args[0], []byte{0x2b, 0x42, 0x6d, 0xec}) {
		t.Errorf("arg[0] = %x; want the ARC-200 transfer(address,uint256) selector 2b426dec", args[0])
	}
	if !bytes.Equal(args[1], vault[:]) {
		t.Error("arg[1] must be the vault itself: a record moves nothing net")
	}
	if new(big.Int).SetBytes(args[2]).Uint64() != recordNoteAmountMicro {
		t.Errorf("arg[2] amount = %d; want %d (one micro-unit: a zero transfer may be rejected by the app, and a rejected record is silent state loss)",
			new(big.Int).SetBytes(args[2]).Uint64(), recordNoteAmountMicro)
	}
}

// TestRecordDispatchIsGated documents that mass dispatch stays OFF until it is
// switched on, so no daemon spends fees while the save set is still incomplete.
func TestRecordDispatchIsGated(t *testing.T) {
	t.Setenv("RECORDS_DISPATCH", "")
	if recordsDispatchEnabled() {
		t.Fatal("record dispatch must default to OFF (nothing seeds before the save set is complete)")
	}
	for _, v := range []string{"1", "true", "TRUE", " yes ", "on"} {
		t.Setenv("RECORDS_DISPATCH", v)
		if !recordsDispatchEnabled() {
			t.Errorf("%q must enable dispatch", v)
		}
	}
	for _, v := range []string{"0", "false", "off", "no", "maybe"} {
		t.Setenv("RECORDS_DISPATCH", v)
		if recordsDispatchEnabled() {
			t.Errorf("%q must NOT enable dispatch", v)
		}
	}

	// And the gated door must REFUSE rather than pretend a record was written.
	l := &Lobby{}
	if _, err := l.sendNoteTx(NotePrefixStateSnapshot + "gated"); err == nil {
		t.Error("a disabled dispatch must return an error, never a silent success")
	}
}