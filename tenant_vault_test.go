//go:build !js && !wasm

package main

import (
	"strings"
	"testing"
)

func testLobbyWithPlayer(wallet string) *Lobby {
	l := &Lobby{playerBalances: map[string]uint64{}, leaderboard: map[string]PlayerStats{}}
	if wallet != "" {
		l.playerBalances[wallet] = 1
	}
	return l
}

// TestTheTenantVaultIsVaultOnly is the SECURITY pin: a wallet the engine knows as a PLAYER cannot be
// a tenant vault, and the answer does not depend on spelling.
func TestTheTenantVaultIsVaultOnly(t *testing.T) {
	reg := NewTenantVaultRegistry()
	l := testLobbyWithPlayer("0xPLAYER")
	if _, err := reg.Register(l, "0xtenant", "0xPLAYER", ""); err == nil {
		t.Error("a registered PLAYER wallet must be REFUSED as a vault: a vault that also plays proves nothing")
	} else if !strings.Contains(err.Error(), "PLAYER wallet") {
		t.Errorf("the refusal must say WHY; got %v", err)
	}
	// Case-insensitive, because the engine stores whatever spelling it was handed.
	if _, err := reg.Register(l, "0xtenant", "0xplayer", ""); err == nil {
		t.Error("the player check must be CASE-INSENSITIVE")
	}
	// The same wallet as the tenant is refused: the app can never sign for the vault.
	if _, err := reg.Register(l, "0xsame", "0xsame", ""); err == nil {
		t.Error("the vault must differ from the tenant wallet")
	}
	if _, err := reg.Register(l, "", "0xvault", ""); err == nil {
		t.Error("a vault with no tenant must be refused")
	}
	if _, err := reg.Register(l, "0xtenant", "   ", ""); err == nil {
		t.Error("a blank vault must be refused")
	}
	if _, err := reg.Register(nil, "0xtenant", "0xvault", ""); err == nil {
		t.Error("a nil lobby must be refused rather than panicking")
	}
	// THE ALLOWED PATH: state proposed, Proven false - the proof is Stage A build 2.
	v, err := reg.Register(l, "0xTenant", "0xVaultOnly", "world one")
	if err != nil {
		t.Fatalf("a vault-only address must register; got %v", err)
	}
	if v.State != tenantVaultStateProposed || v.Proven {
		t.Errorf("a registration is PROPOSED and NOT proven until the verifier lands; got state=%q proven=%v", v.State, v.Proven)
	}
	if v.Network != "Voi" {
		t.Errorf("the vault network is Voi (the sole base chain); got %q", v.Network)
	}
	// A duplicate is refused REGARDLESS of spelling.
	if _, err := reg.Register(l, "0xother", "0xVaultOnly", ""); err == nil {
		t.Error("a duplicate vault must be refused case-insensitively")
	}
	if _, err := reg.Register(l, "0xother", "0xvaultonly", ""); err == nil {
		t.Error("a duplicate vault must be refused in ANY spelling")
	}
	// The snapshot is COPIES: mutating one cannot change the registry.
	snap := reg.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("snapshot size = %d; want 1", len(snap))
	}
	snap[0].Label = "mutated"
	if again := reg.Snapshot(); again[0].Label == "mutated" {
		t.Error("Snapshot must return COPIES, not live pointers")
	}
}
