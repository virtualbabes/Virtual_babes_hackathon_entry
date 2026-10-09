//go:build !js && !wasm

package main

// -- ONE OWNER OF THE SIGNATURE PRIMITIVES (Stage A / B2) ------------------
//
// WHY THIS FILE EXISTS (measured 2026-09-19): the SAME claim - this wallet signed this message - was
// verified by THREE independent code paths (the admin gate, the wallet-link handler, the oracle), and
// the EVM half was duplicated VERBATIM between two of them (personal_sign, 65 bytes, the 27/28
// recovery-byte normalisation, SigToPub, a case-insensitive address compare). A divergence between two
// copies of a signature check means ACCEPTING AN INVALID SIGNATURE, so the primitive gets ONE owner.
//
// WHAT STAYS LOCAL, DELIBERATELY: each context keeps its OWN message construction and its OWN nonce
// policy, because those are protocol facts rather than duplication. This file owns only the part that
// must never differ: decoding, length validation, verification, and the refusal itself.

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/algorand/go-algorand-sdk/v2/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// evmPersonalSignMessage builds the message an EVM wallet signs for a nonce: the RAW nonce, which the
// wallet prefixes with the standard personal_sign header.
func evmPersonalSignMessage(nonce string) []byte {
	return []byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(nonce), nonce))
}

// verifyEVMSignature is the ONE EVM check: recover the signer and compare it, CASE-INSENSITIVELY, with
// the wallet it claims to be. The 27/28 recovery byte is normalised, because some wallets add it.
func verifyEVMSignature(wallet, nonce, signatureHex string) error {
	messageHash := ethcrypto.Keccak256(evmPersonalSignMessage(nonce))
	signatureBytes, err := hex.DecodeString(strings.TrimPrefix(signatureHex, "0x"))
	if err != nil {
		return fmt.Errorf("invalid EVM signature format: %w", err)
	}
	if len(signatureBytes) != 65 {
		return fmt.Errorf("invalid EVM signature length: %d (want 65)", len(signatureBytes))
	}
	if signatureBytes[64] == 27 || signatureBytes[64] == 28 {
		signatureBytes[64] -= 27
	}
	pubKey, err := ethcrypto.SigToPub(messageHash, signatureBytes)
	if err != nil {
		return fmt.Errorf("EVM signature recovery failed: %w", err)
	}
	recovered := ethcrypto.PubkeyToAddress(*pubKey).Hex()
	if !strings.EqualFold(recovered, wallet) {
		return fmt.Errorf("EVM signature mismatch: recovered %s, expected %s", recovered, wallet)
	}
	return nil
}

// verifyAVMSignature is the ONE AVM (ARC-14) check: the address is DECODED to its 32-byte public key -
// a malformed address is a REFUSAL, never a guess - and the base64 signature is verified against the
// message the CALLER constructed. This is the check the tenant-vault proof and any AVM link must use.
func verifyAVMSignature(wallet, message, signatureBase64 string) error {
	addr, err := types.DecodeAddress(wallet)
	if err != nil {
		return fmt.Errorf("invalid AVM address %q: %w", wallet, err)
	}
	sigBytes, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return fmt.Errorf("invalid AVM signature encoding: %w", err)
	}
	// The EXPLICIT primitive (crypto/ed25519), matching oracle_service.go:122, rather than an SDK
	// helper whose domain handling is not visible here: this is the check every AVM door shares.
	if len(sigBytes) != ed25519.SignatureSize {
		return fmt.Errorf("invalid AVM signature length: %d (want %d)", len(sigBytes), ed25519.SignatureSize)
	}
	if !ed25519.Verify(ed25519.PublicKey(addr[:]), []byte(message), sigBytes) {
		return fmt.Errorf("AVM signature does not match %s", wallet)
	}
	return nil
}
