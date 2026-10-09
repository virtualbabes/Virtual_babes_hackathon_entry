//go:build !js && !wasm

package main

import "testing"

func TestValidateIdentityInputNormalizesAndRequiresFields(t *testing.T) {
	primary, address, chain, platform, err := validateIdentityInput("  AVM-1 ", " Wallet-1 ", " ETH ", " Console ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if primary != "avm-1" || address != "wallet-1" || chain != "eth" || platform != "console" {
		t.Fatalf("unexpected normalized identity: %q %q %q %q", primary, address, chain, platform)
	}
	if _, _, _, _, err := validateIdentityInput("primary", "address", "chain", ""); err == nil {
		t.Fatal("expected missing platform to fail")
	}
}
