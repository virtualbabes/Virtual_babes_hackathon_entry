//go:build !js && !wasm

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestPowerScaleForPrice verifies the §14.3 dynamic-power bounds.
func TestPowerScaleForPrice(t *testing.T) {
	cases := []struct {
		paid, base uint64
		wantMin    bool
		wantMax    bool
	}{
		{0, 100, true, false},                         // zero paid → floor
		{50_000_000, 100_000_000, true, false},       // 0.5x → floor
		{100_000_000, 100_000_000, false, false},     // 1.0x → mid
		{300_000_000, 100_000_000, false, true},      // 3.0x → ceiling
		{900_000_000, 100_000_000, false, true},      // 9.0x → clamped ceiling
	}
	for _, c := range cases {
		got := PowerScaleForPrice(c.paid, c.base)
		if c.wantMin && got < 0.5-1e-9 {
			t.Errorf("paid=%d base=%d: expected floor >=0.5, got %.2f", c.paid, c.base, got)
		}
		if c.wantMax && got > 3.0+1e-9 {
			t.Errorf("paid=%d base=%d: expected ceiling <=3.0, got %.2f", c.paid, c.base, got)
		}
		if !c.wantMin && !c.wantMax && (got < 0.99 || got > 1.01) {
			t.Errorf("paid=%d base=%d: expected ~1.0, got %.2f", c.paid, c.base, got)
		}
	}
}

// TestItemRegistryBuildAndPersist verifies build → persist → reload round-trip and collection lookup.
func TestItemRegistryBuildAndPersist(t *testing.T) {
	dir := t.TempDir()
	l := &Lobby{DataDir: dir}
	ir := NewItemRegistry()

	item, err := ir.BuildItem(l, BuildItemRequest{
		Wallet:     "WALLET_A",
		BaseItemID: "mood_catalyst",
		PaidMicro:  100_000_000, // 100 $VBV → scale 1.0
	})
	if err != nil {
		t.Fatalf("BuildItem failed: %v", err)
	}
	if item.PowerScale < 0.99 || item.PowerScale > 1.01 {
		t.Errorf("expected power scale ~1.0, got %.2f", item.PowerScale)
	}
	if item.BondedNFT == "" {
		t.Error("expected a synthesized bonded NFT ID")
	}

	// Persist and reload.
	if err := ir.Save(l); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	savedPath := filepath.Join(dir, itemRegistryFile)
	if _, err := os.Stat(savedPath); err != nil {
		t.Fatalf("registry file not written: %v", err)
	}

	ir2 := NewItemRegistry()
	if err := ir2.Load(l); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	reg := ir2.GetRegistry()
	if len(reg) != 1 {
		t.Fatalf("expected 1 item after reload, got %d", len(reg))
	}
	if reg[0].CreatorWallet != "WALLET_A" {
		t.Errorf("creator wallet not preserved: %s", reg[0].CreatorWallet)
	}
	if reg[0].PaidMicro != 100_000_000 {
		t.Errorf("paid micro not preserved: %d", reg[0].PaidMicro)
	}

	// Collection lookup by wallet.
	coll := ir2.GetCollection("WALLET_A")
	if len(coll) != 1 {
		t.Fatalf("expected 1 item in collection, got %d", len(coll))
	}

	// Marshal round-trip sanity.
	if _, err := json.Marshal(reg[0]); err != nil {
		t.Errorf("item not JSON-serializable: %v", err)
	}
}
