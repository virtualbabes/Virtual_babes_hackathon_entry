//go:build !js && !wasm

package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/algorand/go-algorand-sdk/v2/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// TestVerifyAVMSignatureAcceptsAndRefuses is the NEGATIVE CONTROL for the AVM primitive: a real
// signature verifies, and every way of lying about it is refused.
func TestVerifyAVMSignatureAcceptsAndRefuses(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	var addr types.Address
	copy(addr[:], pub)
	message := "Algorand Signed Message:\nVirtualbabes Arena Tenant Vault:nonce-1"
	sig := ed25519.Sign(priv, []byte(message))
	b64 := base64.StdEncoding.EncodeToString(sig)
	if err := verifyAVMSignature(addr.String(), message, b64); err != nil {
		t.Fatalf("a REAL AVM signature must verify; got %v", err)
	}
	if err := verifyAVMSignature(addr.String(), message+"tampered", b64); err == nil {
		t.Error("a TAMPERED message must be refused")
	}
	if err := verifyAVMSignature("0xnotanalgorandaddress", message, b64); err == nil {
		t.Error("a malformed AVM address must be refused, never guessed")
	}
	if err := verifyAVMSignature(addr.String(), message, "!!!not-base64!!!"); err == nil {
		t.Error("a signature that is not base64 must be refused")
	}
	_, otherPriv, _ := ed25519.GenerateKey(rand.Reader)
	otherSig := base64.StdEncoding.EncodeToString(ed25519.Sign(otherPriv, []byte(message)))
	if err := verifyAVMSignature(addr.String(), message, otherSig); err == nil {
		t.Error("ANOTHER key signature must be refused for this address")
	}
}

// TestVerifyEVMSignatureAcceptsAndRefuses is the NEGATIVE CONTROL for the EVM primitive.
func TestVerifyEVMSignatureAcceptsAndRefuses(t *testing.T) {
	key, err := ethcrypto.GenerateKey()
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	wallet := ethcrypto.PubkeyToAddress(key.PublicKey).Hex()
	nonce := "nonce-abc"
	hash := ethcrypto.Keccak256(evmPersonalSignMessage(nonce))
	sig, err := ethcrypto.Sign(hash, key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	hexSig := hex.EncodeToString(sig)
	if err := verifyEVMSignature(wallet, nonce, hexSig); err != nil {
		t.Fatalf("a REAL EVM signature must verify; got %v", err)
	}
	if err := verifyEVMSignature(wallet, nonce+"x", hexSig); err == nil {
		t.Error("a DIFFERENT nonce must be refused")
	}
	if err := verifyEVMSignature(wallet, nonce, hex.EncodeToString(sig[:64])); err == nil {
		t.Error("a 64-byte signature must be refused")
	}
	other, _ := ethcrypto.GenerateKey()
	if err := verifyEVMSignature(ethcrypto.PubkeyToAddress(other.PublicKey).Hex(), nonce, hexSig); err == nil {
		t.Error("ANOTHER wallet signature must be refused")
	}
}
