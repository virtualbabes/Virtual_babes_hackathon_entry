//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newBrandingTestLobby builds a lobby with a live bonded-asset registry plus one of every
// brandable entity kind, owned by OWNER, so the branding authority can be exercised end to end.
func newBrandingTestLobby(t *testing.T) *Lobby {
	t.Helper()
	l, _ := newTestLobbyForBonded(t)
	l.bondedAssets = NewBondedAssetRegistry()
	l.clubs = make(map[string]*Club)
	l.worldContent = make(map[string]*WorldContentNFT)
	l.itemRegistry = NewItemRegistry()
	l.aiEngine = &AICitizenEngine{citizens: make(map[string]*AICitizen)}

	l.pets["PET-1"] = &PetNFT{PetID: "PET-1", Owner: "OWNER", Name: "Rex", Certified: true}
	l.pets["PET-2"] = &PetNFT{PetID: "PET-2", Owner: "RIVAL", Name: "Enemy"}
	l.vehicles["VEH-1"] = &VehicleNFT{VehicleID: "VEH-1", Owner: "OWNER", Name: "Runner"}
	l.worldContent["WC-1"] = &WorldContentNFT{ContentID: "WC-1", Creator: "OWNER", Kind: "SCENERY"}
	l.clubs["CLUB-1"] = &Club{ID: "CLUB-1", Name: "Neon Room", OwnerWallet: "OWNER"}
	l.clubs["CLUB-2"] = &Club{ID: "CLUB-2", Name: "Rival Room", OwnerWallet: "RIVAL"}
	l.aiEngine.citizens["0xai1"] = &AICitizen{Wallet: "0xai1", Name: "Shadow_01", OriginWallet: "OWNER"}
	l.aiEngine.citizens["0xai2"] = &AICitizen{Wallet: "0xai2", Name: "Rival_Bot", OriginWallet: "RIVAL"}
	faithChurchEngine.mu.Lock()
	if faithChurchEngine.churches == nil {
		faithChurchEngine.churches = make(map[string]*Church)
	}
	faithChurchEngine.churches["CH-1"] = &Church{ID: "CH-1", Name: "Order of Neon", Owner: "OWNER"}
	faithChurchEngine.churches["CH-2"] = &Church{ID: "CH-2", Name: "Rival Faith", Owner: "RIVAL"}
	faithChurchEngine.mu.Unlock()
	return l
}

// validTestMedia is a well-formed media declaration.
func validTestMedia() *BondedMedia {
	return &BondedMedia{
		URI:      "ipfs://bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi",
		MimeType: "image/png",
		HashHex:  strings.Repeat("a", 64),
		Bytes:    1024,
		Width:    512,
		Height:   512,
	}
}

// TestBondedBrandingPolicy_CardsAreExcluded pins the cardinal rule: no card-shaped kind is ever
// brandable, every card alias is listed as blocked, and the startup guard actually fires.
func TestBondedBrandingPolicy_CardsAreExcluded(t *testing.T) {
	cardKinds := []string{
		"card", "cards", "Card", "DECK_CARD", "deck-card", "deck", "decks", "hand", "hand_card",
		"religious_leader_card", "Religious-Leader-Card", "leader_card", "card_deck", "team card",
		"deck_cards", "hand cards",
	}
	for _, k := range cardKinds {
		if !IsCardTargetKind(k) {
			t.Errorf("IsCardTargetKind(%q) = false; every card surface must be excluded", k)
		}
		if IsBindableTargetKind(k) {
			t.Errorf("IsBindableTargetKind(%q) = true; a card can never be bondable", k)
		}
	}
	// Non-card names must not be caught by the exclusion.
	for _, k := range []string{TargetPet, TargetClub, TargetShop, "world-content", "BOT"} {
		if IsCardTargetKind(k) {
			t.Errorf("IsCardTargetKind(%q) = true; this is not a card surface", k)
		}
	}
	// The served policy states the rule for the client.
	p := BondedBrandingPolicy()
	if p["cards_excluded"] != true {
		t.Error("served policy must declare cards_excluded: true")
	}
	if _, ok := p["blocked_target_kinds"].([]string); !ok {
		t.Error("served policy must carry the blocked kind list")
	}
	// The startup guard fires if a card kind is ever added to the bondable catalogue.
	orig := bondableTargetKindDescriptions
	defer func() { bondableTargetKindDescriptions = orig }()
	bondableTargetKindDescriptions = append(append([]bondedTargetKind{}, orig...),
		bondedTargetKind{Kind: "deck_card", Label: "Cards"})
	defer func() {
		if recover() == nil {
			t.Fatal("assertBondedBrandingPolicy must panic when a card kind is listed as bondable")
		}
	}()
	assertBondedBrandingPolicy()
}

// TestBondedBrandingPolicy_CardAssetTypeRefused pins the asset-side exclusion.
func TestBondedBrandingPolicy_CardAssetTypeRefused(t *testing.T) {
	if !IsCardAssetType(AssetCard) {
		t.Fatal("AssetCard must classify as a card type")
	}
	if IsValidAssetType(AssetCard) {
		t.Fatal("a card asset type must not be a valid bonded cosmetic type")
	}
	if !IsValidAssetType(AssetSkin) || !IsValidAssetType(AssetCustom) {
		t.Fatal("the real cosmetic enum range must stay valid")
	}
	r := NewBondedAssetRegistry()
	if _, err := r.MintBondedAssetWithMedia("OWNER", "OWNER", "OWNER", AssetCard, "Deck Art", MoodNeutral, 0, validTestMedia()); err == nil {
		t.Fatal("minting a card-typed bonded asset must be refused")
	}
	if _, err := r.MintBondedAssetWithMedia("OWNER", "OWNER", "OWNER", AssetType(999), "Nonsense", MoodNeutral, 0, nil); err == nil {
		t.Fatal("minting an out-of-range asset type must be refused")
	}
}

// TestBondedMediaValidation pins the media policy: shape is enforced, unsafe/oversized payloads
// are refused, and a declaration is canonicalised so one image has one stored form.
func TestBondedMediaValidation(t *testing.T) {
	if err := ValidateBondedMedia(nil); err != nil {
		t.Errorf("nil media must be allowed (an asset may be minted unpainted): %v", err)
	}
	if err := ValidateBondedMedia(validTestMedia()); err != nil {
		t.Fatalf("a well-formed declaration must validate: %v", err)
	}

	// Canonicalisation: uppercase hash + padded strings collapse to one stored form.
	m := validTestMedia()
	m.HashHex = "  " + strings.ToUpper(strings.Repeat("B", 64)) + "  "
	m.MimeType = " IMAGE/WEBP "
	if err := ValidateBondedMedia(m); err != nil {
		t.Fatalf("canonicalisable media must validate: %v", err)
	}
	if m.HashHex != strings.Repeat("b", 64) || m.MimeType != "image/webp" {
		t.Errorf("media was not canonicalised: hash=%q mime=%q", m.HashHex, m.MimeType)
	}

	// Audio/video need no pixel dimensions.
	audio := &BondedMedia{URI: "https://cdn.example/theme.mp3", MimeType: "audio/mpeg", HashHex: strings.Repeat("c", 64), Bytes: 4096}
	if err := ValidateBondedMedia(audio); err != nil {
		t.Errorf("audio without dimensions must validate: %v", err)
	}

	bad := map[string]func(*BondedMedia){
		"missing uri":        func(m *BondedMedia) { m.URI = "" },
		"data uri":           func(m *BondedMedia) { m.URI = "data:image/png;base64,AAAA" },
		"javascript uri":     func(m *BondedMedia) { m.URI = "javascript:alert(1)" },
		"file uri":           func(m *BondedMedia) { m.URI = "file:///etc/passwd" },
		"hostless https":     func(m *BondedMedia) { m.URI = "https:///x.png" },
		"uri credentials":    func(m *BondedMedia) { m.URI = "https://user:pass@cdn.example/x.png" },
		"bad mime":           func(m *BondedMedia) { m.MimeType = "text/html" },
		"short hash":         func(m *BondedMedia) { m.HashHex = strings.Repeat("a", 63) },
		"non-hex hash":       func(m *BondedMedia) { m.HashHex = strings.Repeat("z", 64) },
		"zero bytes":         func(m *BondedMedia) { m.Bytes = 0 },
		"oversize":           func(m *BondedMedia) { m.Bytes = BondedMediaMaxBytes + 1 },
		"oversize pixel":     func(m *BondedMedia) { m.Width = BondedMediaMaxDimension + 1 },
		"image no dimension": func(m *BondedMedia) { m.Width, m.Height = 0, 0 },
		"overlong uri":       func(m *BondedMedia) { m.URI = "https://cdn.example/" + strings.Repeat("a", BondedMediaUriMaxLen) },
	}
	for name, mutate := range bad {
		m := validTestMedia()
		mutate(m)
		if err := ValidateBondedMedia(m); err == nil {
			t.Errorf("%s: must be refused", name)
		}
	}
}

// TestBondedBrandingBindAuthority pins every refusal path of the binding authority, in order:
// cards, unknown kinds, missing/foreign/off-ledger assets, foreign targets — then proves a
// legitimate bind works, is idempotent, is readable, and unbinds.
func TestBondedBrandingBindAuthority(t *testing.T) {
	l := newBrandingTestLobby(t)
	const owner = "OWNER"

	a, err := l.bondedAssets.MintBondedAssetWithMedia(owner, owner, owner, AssetSkin, "Neon Skin", MoodBenevolent, 0, validTestMedia())
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if !a.Certified {
		t.Error("the own-wallet mint path must certify the asset (§27.7.3)")
	}
	if a.Media == nil || a.Media.Width != 512 {
		t.Errorf("minted media not stored: %+v", a.Media)
	}

	// 1. Cards are refused first — with a valid asset and a valid target in hand.
	for _, card := range []string{"card", "deck", "hand_card", "religious_leader_card", "Deck-Card"} {
		if _, err := l.BindBondedAssetToTarget(owner, a.AssetID, card, "PET-1"); err == nil {
			t.Errorf("binding to card kind %q must be refused", card)
		}
		if _, err := l.UnbindBondedAssetTarget(owner, a.AssetID, card, "PET-1"); err == nil {
			t.Errorf("unbinding card kind %q must be refused", card)
		}
	}

	// 2. Unknown (non-card) kind refused.
	if _, err := l.BindBondedAssetToTarget(owner, a.AssetID, "spaceship", "PET-1"); err == nil {
		t.Error("an unknown target kind must be refused")
	}

	// 3. Missing / foreign asset refused.
	if _, err := l.BindBondedAssetToTarget(owner, "BA-nope", TargetPet, "PET-1"); err == nil {
		t.Error("a missing asset must be refused")
	}
	other, err := l.bondedAssets.MintBondedAssetWithMedia("RIVAL", "RIVAL", "RIVAL", AssetSkin, "Rival Skin", MoodNeutral, 0, nil)
	if err != nil {
		t.Fatalf("rival mint: %v", err)
	}
	if _, err := l.BindBondedAssetToTarget(owner, other.AssetID, TargetPet, "PET-1"); err == nil {
		t.Error("binding someone else's asset must be refused (§27.8)")
	}

	// 4. Off-ledger asset refused (§27.7.3).
	adopted, err := l.bondedAssets.MintBondedAssetWithMedia(owner, owner, owner, AssetSkin, "Adopted", MoodNeutral, 0, nil)
	if err != nil {
		t.Fatalf("adopted mint: %v", err)
	}
	adopted.BlackMarketAdopted = true
	if _, err := l.BindBondedAssetToTarget(owner, adopted.AssetID, TargetPet, "PET-1"); err == nil {
		t.Error("a black-market-adopted asset must not brand an entity (§27.7.3)")
	}

	// 5. Targets the caller does not own (or that do not exist) are refused.
	if _, err := l.BindBondedAssetToTarget(owner, a.AssetID, TargetPet, "PET-2"); err == nil {
		t.Error("branding a rival's companion must be refused")
	}
	if _, err := l.BindBondedAssetToTarget(owner, a.AssetID, TargetPet, "PET-NOPE"); err == nil {
		t.Error("branding a non-existent target must be refused")
	}
	if _, err := l.BindBondedAssetToTarget(owner, a.AssetID, TargetClub, "CLUB-2"); err == nil {
		t.Error("branding a rival's club must be refused")
	}

	// 6. A legitimate bind succeeds, carries the art, and is idempotent.
	if _, err := l.BindBondedAssetToTarget(owner, a.AssetID, "PET", "PET-1"); err != nil {
		t.Fatalf("binding an own companion must succeed: %v", err)
	}
	if _, err := l.BindBondedAssetToTarget(owner, a.AssetID, TargetPet, "PET-1"); err != nil {
		t.Fatalf("re-binding must be idempotent, not an error: %v", err)
	}
	got := l.BondedBrandingForTarget(TargetPet, "PET-1")
	if len(got) != 1 {
		t.Fatalf("idempotent re-bind produced %d brandings, want 1", len(got))
	}
	if got[0].AssetID != a.AssetID || got[0].AssetName != "Neon Skin" || got[0].Media == nil {
		t.Errorf("branding read did not join the asset + media: %+v", got[0])
	}

	// 7. The read path is card-free: a card target can never report branding.
	if out := l.BondedBrandingForTarget("deck_card", "PET-1"); len(out) != 0 {
		t.Error("a card target must never report bonded branding")
	}

	// 8. Unbind is idempotent and removes the branding.
	removed, err := l.UnbindBondedAssetTarget(owner, a.AssetID, TargetPet, "PET-1")
	if err != nil || !removed {
		t.Fatalf("unbind: removed=%v err=%v", removed, err)
	}
	if len(l.BondedBrandingForTarget(TargetPet, "PET-1")) != 0 {
		t.Error("branding survived unbind")
	}
	removed, err = l.UnbindBondedAssetTarget(owner, a.AssetID, TargetPet, "PET-1")
	if err != nil || removed {
		t.Errorf("second unbind must be a no-op: removed=%v err=%v", removed, err)
	}
}

// TestBondedBrandingTargetsEnumeration pins that a wallet sees exactly its own brandable
// entities across the ecosphere — never a rival's, never a card, and never a card filter.
func TestBondedBrandingTargetsEnumeration(t *testing.T) {
	l := newBrandingTestLobby(t)
	l.itemRegistry.items["ITEM-1"] = &BuiltItem{
		ItemID:        "ITEM-1",
		CreatorWallet: "OWNER",
		ShopItem:      ShopItem{Name: "Glow Trap"},
	}

	targets := l.BondedTargetsForWallet("OWNER", "")
	kinds := map[string]int{}
	for _, tv := range targets {
		kinds[tv.Kind]++
		if IsCardTargetKind(tv.Kind) {
			t.Errorf("a card kind %q was enumerated as brandable", tv.Kind)
		}
		if tv.KindLabel == "" {
			t.Errorf("target %s/%s has no served label", tv.Kind, tv.TargetID)
		}
	}
	for _, want := range []string{TargetItem, TargetTheme, TargetBot, TargetPet, TargetVehicle, TargetChurch, TargetClub, TargetShop, TargetWorldContent} {
		if kinds[want] == 0 {
			t.Errorf("kind %q was not enumerated for a wallet that owns one", want)
		}
	}
	for _, foreign := range []string{"PET-2", "CLUB-2", "CH-2", "0xai2"} {
		for _, tv := range targets {
			if tv.TargetID == foreign {
				t.Errorf("a rival's entity %s leaked into the wallet's target list", foreign)
			}
		}
	}

	// Kind filters: a real kind narrows, a card kind and an unknown kind yield nothing.
	if got := l.BondedTargetsForWallet("OWNER", "pet"); len(got) != 1 || got[0].TargetID != "PET-1" {
		t.Errorf("pet filter: got %+v want exactly PET-1", got)
	}
	if got := l.BondedTargetsForWallet("OWNER", "deck"); len(got) != 0 {
		t.Error("a card kind filter must never enumerate targets")
	}
	if got := l.BondedTargetsForWallet("OWNER", "spaceship"); len(got) != 0 {
		t.Error("an unknown kind filter must never enumerate targets")
	}
	// No wallet → nothing (never the whole civilization).
	if got := l.BondedTargetsForWallet("", ""); len(got) != 0 {
		t.Error("an empty wallet must enumerate nothing")
	}
	// The catalogue itself is card-free and stable-ordered.
	last := ""
	for _, tv := range l.BondedTargetsForWallet("OWNER", TargetPet) {
		if last != "" && tv.TargetID < last {
			t.Error("target enumeration is not deterministically ordered")
		}
		last = tv.TargetID
	}
}

// TestBondedChestAndBindingsReadInADeterministicOrder pins the SERVED ORDER of the chest and its
// bindings. `Assets` and `Bindings` are maps, so before the sort every read handed back a different
// order: the studio's asset rows and the market's "asset to sell" dropdown reshuffled on every
// refresh even though nothing had changed, and anything that read chest[0] was a coin toss. This
// asserts both halves of the rule — the list is stable across repeated reads, and it is ordered
// newest-first with the asset id as the tie-break.
//
// The fixture stores its wallet as "OWNER" while the request resolves to "owner", so this test also
// pins the OTHER half of the same problem: the chest must find a wallet whose stored spelling differs
// from the resolved one, or a wallet cannot see the assets it owns.
func TestBondedChestAndBindingsReadInADeterministicOrder(t *testing.T) {
	l := newBrandingTestLobby(t)

	minted := make([]string, 0, 8)
	for i := 0; i < 8; i++ {
		a, err := l.bondedAssets.MintBondedAssetWithMedia("OWNER", "OWNER", "OWNER",
			AssetAppearance, fmt.Sprintf("Frame %d", i), MoodNeutral, 0, validTestMedia())
		if err != nil {
			t.Fatalf("mint %d: %v", i, err)
		}
		minted = append(minted, a.AssetID)
	}
	// Two bindings on ONE asset at two different targets, so the binding list has to be ordered too
	// (its map is keyed "<asset>:<kind>:<target>", which is not the order the UI shows).
	for _, k := range []string{TargetPet, TargetClub} {
		if _, err := l.bondedAssets.BindAssetTarget(minted[0], k, "TGT-1"); err != nil {
			t.Fatalf("bind %s: %v", k, err)
		}
	}

	read := func() ([]string, []string) {
		req := httptest.NewRequest(http.MethodGet, "/api/assets?wallet=OWNER", nil)
		rec := httptest.NewRecorder()
		l.handleListBondedAssets(rec, req)
		var chest struct {
			Assets []struct {
				AssetID   string    `json:"asset_id"`
				CreatedAt time.Time `json:"created_at"`
			} `json:"assets"`
			Bindings []struct {
				AssetID    string `json:"asset_id"`
				TargetKind string `json:"target_kind"`
				TargetID   string `json:"target_id"`
			} `json:"bindings"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &chest); err != nil {
			t.Fatalf("chest returned non-JSON: %v", err)
		}
		if len(chest.Assets) != len(minted) {
			t.Fatalf("chest = %d assets; want %d", len(chest.Assets), len(minted))
		}
		ids := make([]string, 0, len(chest.Assets))
		for i, a := range chest.Assets {
			ids = append(ids, a.AssetID)
			if i == 0 {
				continue
			}
			prev := chest.Assets[i-1]
			if prev.CreatedAt.Before(a.CreatedAt) {
				t.Fatalf("chest is not newest-first at %d: %s before %s", i, prev.AssetID, a.AssetID)
			}
			if prev.CreatedAt.Equal(a.CreatedAt) && prev.AssetID > a.AssetID {
				t.Fatalf("chest tie-break is not the asset id at %d: %s before %s", i, prev.AssetID, a.AssetID)
			}
		}
		binds := make([]string, 0, len(chest.Bindings))
		for _, b := range chest.Bindings {
			binds = append(binds, b.AssetID+":"+b.TargetKind+":"+b.TargetID)
		}
		return ids, binds
	}

	wantIDs, wantBinds := read()
	if len(wantBinds) != 2 {
		t.Fatalf("bindings = %d; want 2 (one asset worn on two targets)", len(wantBinds))
	}
	if wantBinds[0] > wantBinds[1] {
		t.Errorf("bindings are not ordered: %v", wantBinds)
	}
	// 25 reads: with eight assets a random order repeats by chance once in ~40,000 pairs, so a
	// missing sort cannot survive this loop.
	for i := 0; i < 25; i++ {
		gotIDs, gotBinds := read()
		if strings.Join(gotIDs, ",") != strings.Join(wantIDs, ",") {
			t.Fatalf("chest order changed between reads (read %d):\n got %v\nwant %v", i+1, gotIDs, wantIDs)
		}
		if strings.Join(gotBinds, ",") != strings.Join(wantBinds, ",") {
			t.Fatalf("binding order changed between reads (read %d):\n got %v\nwant %v", i+1, gotBinds, wantBinds)
		}
	}
}

// TestBondedBrandingWalletKeyedTargetsAreCanonical pins the fix for a real defect found by live
// probing: wallet-keyed targets (theme / bot) must resolve in ONE canonical form, so a client
// whose wallet casing differs from the engine's lowercase form still owns its own target.
func TestBondedBrandingWalletKeyedTargetsAreCanonical(t *testing.T) {
	l := newBrandingTestLobby(t)
	const lower = "0xowner"
	l.aiEngine.citizens["0xbot"] = &AICitizen{Wallet: "0xbot", Name: "Shadow", OriginWallet: lower}

	a, err := l.bondedAssets.MintBondedAssetWithMedia(lower, lower, lower, AssetSkin, "Skin", MoodNeutral, 0, nil)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}

	// A mixed-case request must resolve to the wallet's own theme.
	if _, err := l.BindBondedAssetToTarget(lower, a.AssetID, "Theme", "0XOWNER"); err != nil {
		t.Fatalf("a wallet-keyed target must resolve regardless of casing: %v", err)
	}
	if got := l.BondedBrandingForTarget("theme", "0XOWNER"); len(got) != 1 {
		t.Errorf("read with mixed case found %d brandings, want 1", len(got))
	}
	if got := l.BondedBrandingForTarget("theme", lower); len(got) != 1 {
		t.Errorf("read with canonical case found %d brandings, want 1", len(got))
	}

	// Bots are wallet-keyed too.
	if _, err := l.BindBondedAssetToTarget(lower, a.AssetID, "BOT", "0xBOT"); err != nil {
		t.Fatalf("bot bind: %v", err)
	}
	if got := l.BondedBrandingForTarget(TargetBot, "0xbot"); len(got) != 1 {
		t.Errorf("bot branding read: got %d want 1", len(got))
	}

	// Unbinding through the mixed-case form removes the ONE canonical binding.
	removed, err := l.UnbindBondedAssetTarget(lower, a.AssetID, "theme", "0XOWNER")
	if err != nil || !removed {
		t.Fatalf("mixed-case unbind: removed=%v err=%v", removed, err)
	}
	if got := l.BondedBrandingForTarget("theme", lower); len(got) != 0 {
		t.Error("mixed-case unbind left the canonical binding behind")
	}
}
