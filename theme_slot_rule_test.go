package main

import (
	"strings"
	"testing"
)

// TestThemeSlotKeySpaceIsDisjoint pins the §27.6 rule added at the domain owner.
//
// `BondedAssetRegistry.Bindings` is written by TWO doors through ONE `Slot` field:
//
//	BindThemeAsset(assetID, slot)        → key `<asset>:<slot>`            (TWO parts)
//	BindAssetTarget(assetID, kind, id)   → key `<asset>:<kind>:<targetID>` (THREE parts)
//
// The key shapes stay disjoint only while a slot label cannot contain a colon. A slot spelled
// `"item:someid"` addresses a TARGET binding's key, so a later LockThemeAsset / IsThemeLocked would
// read and mutate the wrong record — and it would look like a successful lock. The rule is enforced
// at the door that owns the slot vocabulary rather than left to the caller.
func TestThemeSlotKeySpaceIsDisjoint(t *testing.T) {
	r := NewBondedAssetRegistry()
	a, err := r.MintBondedAssetWithMedia("OWNER", "OWNER", "OWNER", AssetCustom, "Livery", MoodNeutral, 0, validTestMedia())
	if err != nil {
		t.Fatalf("minting the fixture asset must succeed: %v", err)
	}

	// An empty slot has no key at all: it would write `<asset>:` and be unaddressable afterwards.
	if _, err := r.BindThemeAsset(a.AssetID, ""); err == nil {
		t.Fatal("an empty slot must be refused")
	}
	// A colon would let a slot alias a target binding's key.
	if _, err := r.BindThemeAsset(a.AssetID, "item:someid"); err == nil {
		t.Fatal("a slot containing ':' must be refused")
	}
	if _, err := r.BindThemeAsset(a.AssetID, "a:b:c"); err == nil {
		t.Fatal("a slot containing ':' must be refused however many it has")
	}
	// Nothing above may have written.
	if _, ok := r.Bindings[a.AssetID+":"]; ok {
		t.Fatal("a refused slot must not write a binding")
	}
	if _, ok := r.Bindings[a.AssetID+":item:someid"]; ok {
		t.Fatal("a refused slot must not write a binding")
	}

	// A clean slot binds, is readable, and is lockable — the allowed path still works.
	b, err := r.BindThemeAsset(a.AssetID, "board_bg")
	if err != nil {
		t.Fatalf("a clean slot must bind: %v", err)
	}
	if b.Slot != "board_bg" || b.Locked {
		t.Fatalf("expected a fresh unlocked binding, got slot=%q locked=%v", b.Slot, b.Locked)
	}
	if r.IsThemeLocked(a.AssetID, "board_bg") {
		t.Fatal("a freshly bound slot must not read as locked")
	}
	if err := r.LockThemeAsset(a.AssetID, "board_bg", "OWNER"); err != nil {
		t.Fatalf("the owner must be able to lock its own binding: %v", err)
	}
	if !r.IsThemeLocked(a.AssetID, "board_bg") {
		t.Fatal("the lock must read back")
	}

	// The two shapes coexist in one map without aliasing: the target binding is a DIFFERENT record
	// and is not the one that reads as locked.
	if _, err := r.BindAssetTarget(a.AssetID, "board_bg", "PET-1"); err != nil {
		t.Fatalf("a target binding must still work: %v", err)
	}
	if _, ok := r.Bindings[a.AssetID+":board_bg"]; !ok {
		t.Fatal("the slot binding must occupy the two-part key")
	}
	if _, ok := r.Bindings[a.AssetID+":board_bg:PET-1"]; !ok {
		t.Fatal("the target binding must occupy the three-part key")
	}
}

// TestThemeSlotRuleSurvivesTheHTTPBoundary proves the rule is reachable from the client door: an
// unknown asset is refused FIRST (so a caller cannot probe the slot rule on a nonexistent asset),
// and once a real asset exists the colon is refused with a reason the client can display.
func TestThemeSlotRuleSurvivesTheHTTPBoundary(t *testing.T) {
	l := newBrandingTestLobby(t)
	a, err := l.bondedAssets.MintBondedAssetWithMedia("OWNER", "OWNER", "OWNER", AssetCustom, "Livery", MoodNeutral, 0, validTestMedia())
	if err != nil {
		t.Fatalf("mint: %v", err)
	}

	// A colon slot is refused by the domain owner, and the reason names the collision.
	if _, err := l.bondedAssets.BindThemeAsset(a.AssetID, "kind:target"); err == nil {
		t.Fatal("the colon slot must be refused")
	} else if !strings.Contains(err.Error(), ":") {
		t.Fatalf("the refusal should name the character that causes the collision, got %q", err.Error())
	}

	// The clean path through the same door: bind then lock, both readable.
	if _, err := l.bondedAssets.BindThemeAsset(a.AssetID, "menu_bg"); err != nil {
		t.Fatalf("clean bind: %v", err)
	}
	if err := l.bondedAssets.LockThemeAsset(a.AssetID, "menu_bg", "OWNER"); err != nil {
		t.Fatalf("lock: %v", err)
	}
	if !l.bondedAssets.IsThemeLocked(a.AssetID, "menu_bg") {
		t.Fatal("lock must read back through the lobby's registry")
	}
	// §27.6: a locked asset is excluded from the wallet's theme MoodTag.
	if _, contributed := l.bondedAssets.MoodTagForWallet("OWNER"); contributed {
		t.Fatal("a locked asset must not contribute its MoodTag to the theme")
	}
}

// TestMoodTagCountsAnUnlockedBoundAsset is the positive control for the test above: without the
// lock the same binding DOES contribute, so "contributed == false" there means the lock worked
// rather than that the feature never works.
func TestMoodTagCountsAnUnlockedBoundAsset(t *testing.T) {
	l := newBrandingTestLobby(t)
	a, err := l.bondedAssets.MintBondedAssetWithMedia("OWNER", "OWNER", "OWNER", AssetCustom, "Livery", MoodBenevolent, 0, validTestMedia())
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if _, err := l.bondedAssets.BindThemeAsset(a.AssetID, "menu_bg"); err != nil {
		t.Fatalf("clean bind: %v", err)
	}
	if _, contributed := l.bondedAssets.MoodTagForWallet("OWNER"); !contributed {
		t.Fatal("an UNLOCKED bound asset must contribute its MoodTag (the positive control)")
	}
}
