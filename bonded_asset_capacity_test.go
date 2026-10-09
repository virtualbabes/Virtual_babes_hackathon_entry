//go:build !js && !wasm

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// bonded_asset_capacity_test.go
// Pins §23.5.1: "a player can create many bonded assets equal to their NFT supply - cards/deck".
//
// The load-bearing assertions are:
//  1. the capacity is the wallet's card/deck NFT supply AS THE ENGINE RECORDS IT (the CARD-* keys of
//     PlayerStats.Inventory - the same ledger the auction/loan/black-market/jail escrow maintains);
//  2. the MINT DOOR enforces it, and a refusal writes nothing;
//  3. the cap is on CREATION: an asset that already exists is never retro-invalidated, and a wallet
//     that receives a gift is reported truthfully (over capacity) rather than losing assets;
//  4. the budget and its BASIS are served, so a player is never refused without being told why.

// capOwner is lowercase, mirroring the engine's canonical wallet spelling.
const capOwner = "owner"
const capRival = "rival"

// seedCardLedger gives a wallet card/deck NFTs in the engine's own ledger.
func seedCardLedger(l *Lobby, wallet string, entries map[string]int) {
	s := l.leaderboard[wallet]
	if s.Inventory == nil {
		s.Inventory = make(map[string]int)
	}
	for k, v := range entries {
		s.Inventory[k] = v
	}
	l.leaderboard[wallet] = s
}

// mintCapAsset mints directly through the registry (the storage layer), which is deliberately NOT
// where the capacity rule lives - the mint HTTP door is - so a test can set up over-cap states.
func mintCapAsset(t *testing.T, l *Lobby, owner, name string) *BondedAsset {
	t.Helper()
	a, err := l.bondedAssets.MintBondedAssetWithMedia(owner, owner, owner, AssetSkin, name, MoodNeutral, 0, nil)
	if err != nil {
		t.Fatalf("registry mint: %v", err)
	}
	return a
}

// TestBondedAssetCapacityIsTheCardSupply pins the basis and the arithmetic.
func TestBondedAssetCapacityIsTheCardSupply(t *testing.T) {
	l := newBrandingTestLobby(t)

	// A wallet the engine has no card record for has a capacity of ZERO - a real answer, not an error.
	if c := l.BondedAssetCapacityForWallet(capOwner); c.Limit != 0 || c.Used != 0 || c.Remaining != 0 || c.Basis == "" {
		t.Errorf("no card record: %+v; want limit 0 with a stated basis", c)
	}

	// Only CARD-* keys are supply: weapons and faceplates are not cards, and quantities are SUMMED
	// (two copies of a card are two NFTs).
	seedCardLedger(l, capOwner, map[string]int{
		"CARD-1": 1, "CARD-2": 3, "WEAPON-RAILGUN": 5, "FACEPLATE-X": 2,
	})
	if got := l.CardNFTSupplyForWallet(capOwner); got != 4 {
		t.Errorf("supply = %d; want 4 (1+3 card NFTs; weapons/faceplates are not cards)", got)
	}
	// An uppercase spelling is the SAME wallet (wallets are canonical), so the cap does not vanish.
	if got := l.CardNFTSupplyForWallet(strings.ToUpper(capOwner)); got != 4 {
		t.Errorf("uppercase supply = %d; want 4 (one wallet, one spelling rule)", got)
	}
	// A zero/negative quantity is not supply.
	seedCardLedger(l, capOwner, map[string]int{"CARD-EXPIRED": 0})
	if got := l.CardNFTSupplyForWallet(capOwner); got != 4 {
		t.Errorf("a zero-quantity card counted as supply: %d", got)
	}

	c := l.BondedAssetCapacityForWallet(capOwner)
	if c.Limit != 4 || c.Used != 0 || c.Remaining != 4 {
		t.Errorf("capacity = %+v; want limit 4 used 0 remaining 4", c)
	}

	// Owned assets consume the budget, one per asset.
	mintCapAsset(t, l, capOwner, "One")
	mintCapAsset(t, l, capOwner, "Two")
	c = l.BondedAssetCapacityForWallet(capOwner)
	if c.Limit != 4 || c.Used != 2 || c.Remaining != 2 {
		t.Errorf("capacity = %+v; want used 2 remaining 2", c)
	}

	// A GIFTED asset is reported truthfully: ownership above the cap is stated, never hidden, and the
	// remaining budget is clamped at zero rather than going negative.
	seedCardLedger(l, capRival, map[string]int{"CARD-9": 1})
	gift := mintCapAsset(t, l, capRival, "Gift")
	if err := l.bondedAssets.TransferOwnership(gift.AssetID, capRival, capOwner); err != nil {
		t.Fatalf("transfer: %v", err)
	}
	c = l.BondedAssetCapacityForWallet(capOwner)
	if c.Used != 3 || c.Remaining != 1 {
		t.Errorf("after a gift: %+v; want used 3 remaining 1 (limit 4)", c)
	}
	// Burn frees the slot again (a burned asset is deleted from the registry).
	if err := l.bondedAssets.Burn(gift.AssetID, capOwner); err != nil {
		t.Fatalf("burn: %v", err)
	}
	if c = l.BondedAssetCapacityForWallet(capOwner); c.Used != 2 || c.Remaining != 2 {
		t.Errorf("after a burn: %+v; want the slot freed", c)
	}
}

// TestBondedAssetMintIsCappedAtTheCardSupply drives the real HTTP mint door.
func TestBondedAssetMintIsCappedAtTheCardSupply(t *testing.T) {
	l := newBrandingTestLobby(t)
	mint := func() map[string]interface{} {
		req := httptest.NewRequest(http.MethodPost, "/api/assets/mint?wallet="+capOwner,
			strings.NewReader(`{"name":"Neon Livery","asset_type":0}`))
		rec := httptest.NewRecorder()
		l.handleMintBondedAsset(rec, req)
		var out map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("mint returned non-JSON: %v", err)
		}
		return out
	}

	// Zero cards: the door refuses, states the rule, serves the budget, and writes nothing.
	out := mint()
	if out["success"] == true {
		t.Fatalf("a wallet with no card/deck NFTs was allowed to mint: %v", out)
	}
	if msg, _ := out["error"].(string); !strings.Contains(msg, "capped at your card/deck NFT supply") {
		t.Errorf("refusal = %q; want the capacity rule stated", msg)
	}
	if _, ok := out["capacity"].(map[string]interface{}); !ok {
		t.Errorf("a capacity refusal must serve the wallet's budget: %v", out["capacity"])
	}
	if n := len(l.bondedAssets.Assets); n != 0 {
		t.Fatalf("a refused mint created %d assets", n)
	}

	// Two cards = two bonded assets, exactly.
	seedCardLedger(l, capOwner, map[string]int{"CARD-1": 1, "CARD-2": 1})
	for i := 0; i < 2; i++ {
		if out := mint(); out["success"] != true {
			t.Fatalf("mint %d within capacity was refused: %v", i+1, out)
		}
	}
	out = mint()
	if out["success"] == true {
		t.Fatalf("minting past the card/deck supply was allowed: %v", out)
	}
	if msg, _ := out["error"].(string); !strings.Contains(msg, "capacity reached") {
		t.Errorf("over-cap refusal = %q", msg)
	}
	if n := len(l.bondedAssets.Assets); n != 2 {
		t.Errorf("assets = %d; want exactly 2 (the card supply)", n)
	}

	// A third card raises the cap by exactly one.
	seedCardLedger(l, capOwner, map[string]int{"CARD-3": 1})
	if out := mint(); out["success"] != true {
		t.Fatalf("a raised card supply did not raise the cap: %v", out)
	}
	if c := l.BondedAssetCapacityForWallet(capOwner); c.Used != c.Limit {
		t.Errorf("capacity = %+v; want the wallet at exactly its limit", c)
	}
}

// TestBondedAssetCapNeverInvalidatesExistingAssets: the rule gates CREATION only.
func TestBondedAssetCapNeverInvalidatesExistingAssets(t *testing.T) {
	l := newBrandingTestLobby(t)
	for _, n := range []string{"A", "B", "C"} {
		mintCapAsset(t, l, capOwner, n)
	}
	// The wallet's card record is absent (or lost): capacity is 0 and a new mint is refused...
	req := httptest.NewRequest(http.MethodPost, "/api/assets/mint?wallet="+capOwner,
		strings.NewReader(`{"name":"D","asset_type":0}`))
	rec := httptest.NewRecorder()
	l.handleMintBondedAsset(rec, req)
	if !strings.Contains(rec.Body.String(), "capped at your card/deck NFT supply") {
		t.Errorf("a new mint over a zero capacity was not refused: %s", rec.Body.String())
	}

	// ...while every asset that already exists is still listed, with the over-cap state REPORTED.
	listReq := httptest.NewRequest(http.MethodGet, "/api/assets?wallet="+capOwner, nil)
	listRec := httptest.NewRecorder()
	l.handleListBondedAssets(listRec, listReq)
	var chest map[string]interface{}
	if err := json.Unmarshal(listRec.Body.Bytes(), &chest); err != nil {
		t.Fatalf("chest returned non-JSON: %v", err)
	}
	assets, _ := chest["assets"].([]interface{})
	if len(assets) != 3 {
		t.Errorf("chest dropped assets: %d; the cap must never remove an existing asset", len(assets))
	}
	capObj, _ := chest["capacity"].(map[string]interface{})
	if capObj == nil || capObj["used"] != float64(3) || capObj["limit"] != float64(0) || capObj["remaining"] != float64(0) {
		t.Errorf("chest capacity = %v; want the truthful over-cap report (used 3, limit 0, remaining 0)", chest["capacity"])
	}
}

// TestCardSupplyMatchesAnyStoredSpelling: this fixture stores the wallet as "OWNER" while production
// keys are lowercase-canonical - either way the SAME wallet must find its own card supply.
func TestCardSupplyMatchesAnyStoredSpelling(t *testing.T) {
	l := newBrandingTestLobby(t)
	seedCardLedger(l, "OWNER", map[string]int{"CARD-7": 2})
	if got := l.CardNFTSupplyForWallet("owner"); got != 2 {
		t.Errorf("lowercase query for an uppercase-keyed record = %d; want 2", got)
	}
	if got := l.CardNFTSupplyForWallet("OWNER"); got != 2 {
		t.Errorf("uppercase query = %d; want 2", got)
	}
}

// TestBondedAssetCapacityRuleIsServed: the rule and its basis are served, never re-declared client-side.
func TestBondedAssetCapacityRuleIsServed(t *testing.T) {
	pol := BondedBrandingPolicy()
	if rule, _ := pol["capacity_rule"].(string); !strings.Contains(rule, "card/deck NFT supply") {
		t.Errorf("capacity_rule = %q; the chest must serve the creation rule", rule)
	}
	if req, _ := pol["creation_requires"].(string); !strings.Contains(req, "created") {
		t.Errorf("creation_requires = %q; the policy must state that an asset must exist first", req)
	}
	if cv := CardViewPolicy(); cv["asset_required"] != true {
		t.Errorf("the card-view policy must require a bonded asset: %v", cv["asset_required"])
	}
	if basis := BondedAssetCapacityBasisEngineCards; !strings.Contains(basis, "CARD-*") {
		t.Errorf("the basis must name the engine record it reads: %q", basis)
	}
}
