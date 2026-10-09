//go:build !js && !wasm

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// slideSlotView finds one slot in a surface (test convenience).
func slideSlotView(t *testing.T, s SlideThemeSurface, slot string) SlideThemeSlotView {
	t.Helper()
	for _, v := range s.Slots {
		if v.Slot == slot {
			return v
		}
	}
	t.Fatalf("slot %s is missing from the surface", slot)
	return SlideThemeSlotView{}
}

// assertNoCardKeys walks a served payload and fails if any KEY names a card. Values may say
// "cards are excluded" (that is the rule being stated); a KEY never may.
func assertNoCardKeys(t *testing.T, prefix string, v interface{}) {
	t.Helper()
	switch m := v.(type) {
	case map[string]interface{}:
		for k, sub := range m {
			if strings.Contains(strings.ToLower(k), "card") {
				t.Errorf("served key %s%s names a card", prefix, k)
			}
			assertNoCardKeys(t, prefix+k+".", sub)
		}
	case []interface{}:
		for i, sub := range m {
			assertNoCardKeys(t, prefix+"["+string(rune('0'+i))+"]", sub)
		}
	}
}

// TestSlideTheming_VocabularyIsCardFreeAndComplete pins the slide vocabulary: six slots, no card in
// any of them, and every slot defaulting to a frame the FREE starter pack grants — because "the
// first needed images" and "the images the slides need" must never drift apart.
func TestSlideTheming_VocabularyIsCardFreeAndComplete(t *testing.T) {
	if len(slideSlotDefs) != 6 {
		t.Fatalf("there are %d slide slots; the app shows three per show on two shows", len(slideSlotDefs))
	}
	menus, dash := 0, 0
	for _, d := range slideSlotDefs {
		if strings.Contains(strings.ToLower(d.ID+" "+d.Label+" "+d.Show), "card") {
			t.Errorf("slide slot %q names a card", d.ID)
		}
		if !isSlideSlot(d.ID) {
			t.Errorf("slot %s is not recognised by isSlideSlot", d.ID)
		}
		switch d.Show {
		case "menu":
			menus++
		case "dashboard":
			dash++
		default:
			t.Errorf("slot %s has an unknown show %q", d.ID, d.Show)
		}
		if !placeholderIsStarterSKU(d.DefaultSKU) {
			t.Errorf("slot %s defaults to %s, which the free starter pack does not grant (the slides "+
				"must show art the player already owns)", d.ID, d.DefaultSKU)
		}
		if s, ok := placeholderSku(d.DefaultSKU); !ok || !s.Purchasable {
			t.Errorf("slot %s default %s is not a usable pack frame", d.ID, d.DefaultSKU)
		}
	}
	if menus != 3 || dash != 3 {
		t.Errorf("slide shows are menu=%d dashboard=%d; want 3 and 3", menus, dash)
	}
	assertNoCardKeys(t, "", SlideThemingPolicy())
}

// TestSlideTheming_RequiresOneOfYourOwnAssets exercises every refusal class, then the allowed path,
// and pins that theming never writes a branding binding (the §10.1 card exclusion stays structural).
func TestSlideTheming_RequiresOneOfYourOwnAssets(t *testing.T) {
	l := newBrandingTestLobby(t)
	const owner = "OWNER"

	own, err := l.bondedAssets.MintBondedAssetWithMedia(owner, owner, owner, AssetBackground, "Mine", MoodNeutral, 0, validTestMedia())
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	rival, err := l.bondedAssets.MintBondedAssetWithMedia("RIVAL", "RIVAL", "RIVAL", AssetBackground, "Theirs", MoodNeutral, 0, validTestMedia())
	if err != nil {
		t.Fatalf("mint rival: %v", err)
	}
	artless, err := l.bondedAssets.MintBondedAssetWithMedia(owner, owner, owner, AssetBackground, "Blank", MoodNeutral, 0, nil)
	if err != nil {
		t.Fatalf("mint artless: %v", err)
	}

	// Direct inserts for the states the mint door refuses on purpose (card-typed) and the states a
	// real engine can produce (black-market adoption).
	cardish := &BondedAsset{AssetID: "BA-card-typed", AssetType: AssetCard, Name: "Card", OwnerWallet: owner, HolderWallet: owner, Media: validTestMedia()}
	market := &BondedAsset{AssetID: "BA-black-market", AssetType: AssetBackground, Name: "Stolen", OwnerWallet: owner, HolderWallet: owner, BlackMarketAdopted: true, Media: validTestMedia()}
	l.bondedAssets.mu.Lock()
	l.bondedAssets.Assets[cardish.AssetID] = cardish
	l.bondedAssets.Assets[market.AssetID] = market
	l.bondedAssets.mu.Unlock()

	refusals := []struct {
		what string
		req  SlideThemeRequest
	}{
		{"no asset at all", SlideThemeRequest{Slot: SlideSlotMenu1}},
		{"an unknown slot", SlideThemeRequest{Slot: "card_slot_1", AssetID: own.AssetID}},
		{"an asset that does not exist", SlideThemeRequest{Slot: SlideSlotMenu1, AssetID: "BA-nope"}},
		{"another wallet's asset", SlideThemeRequest{Slot: SlideSlotMenu1, AssetID: rival.AssetID}},
		{"a card-typed asset", SlideThemeRequest{Slot: SlideSlotMenu1, AssetID: cardish.AssetID}},
		{"a black-market asset", SlideThemeRequest{Slot: SlideSlotMenu1, AssetID: market.AssetID}},
		{"an asset with no art", SlideThemeRequest{Slot: SlideSlotMenu1, AssetID: artless.AssetID}},
	}
	for _, r := range refusals {
		if _, err := l.SetSlideThemeForWallet(owner, r.req); err == nil {
			t.Errorf("%s must be refused", r.what)
		}
	}
	if n := l.bondedAssets.SlideThemeCount(); n != 0 {
		t.Fatalf("%d slide records were written by refused requests; a refusal must change nothing", n)
	}

	// The allowed path: one of the wallet's own, painted on its own screen.
	bindingsBefore := len(l.bondedAssets.Bindings)
	surface, err := l.SetSlideThemeForWallet(owner, SlideThemeRequest{Slot: SlideSlotMenu1, AssetID: own.AssetID})
	if err != nil {
		t.Fatalf("setting a slide theme with an owned asset must succeed: %v", err)
	}
	if surface.ActiveCount != 1 {
		t.Errorf("active_count = %d; want 1", surface.ActiveCount)
	}
	v := slideSlotView(t, *surface, SlideSlotMenu1)
	if !v.Active || v.ArtURI != own.Media.URI || v.ArtKind != "asset" || v.AssetID != own.AssetID {
		t.Errorf("the themed slot reads active=%v art=%q kind=%q asset=%q", v.Active, v.ArtURI, v.ArtKind, v.AssetID)
	}
	// The other five stay on the pack default, with real art resolved from the catalogue.
	other := slideSlotView(t, *surface, SlideSlotDashboard1)
	if other.Active || other.ArtKind != "default" || other.ArtURI == "" {
		t.Errorf("an unthemed slot must show the pack default: %+v", other)
	}
	// NEVER writes branding: the only branding writer remains BindBondedAssetToTarget.
	if len(l.bondedAssets.Bindings) != bindingsBefore {
		t.Error("slide theming wrote a branding binding; it must never brand anything")
	}
}

// TestSlideTheming_AStaleRecordLosesItsArt pins that ownership is re-proved on EVERY read: once the
// asset is sold or burned, the slide falls back to the pack default and SAYS why.
func TestSlideTheming_AStaleRecordLosesItsArt(t *testing.T) {
	l := newBrandingTestLobby(t)
	const owner = "OWNER"

	sold, err := l.bondedAssets.MintBondedAssetWithMedia(owner, owner, owner, AssetBackground, "Sold", MoodNeutral, 0, validTestMedia())
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	burned, err := l.bondedAssets.MintBondedAssetWithMedia(owner, owner, owner, AssetBackground, "Burned", MoodNeutral, 0, validTestMedia())
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if _, err := l.SetSlideThemeForWallet(owner, SlideThemeRequest{Slot: SlideSlotDashboard2, AssetID: sold.AssetID}); err != nil {
		t.Fatalf("set: %v", err)
	}
	if _, err := l.SetSlideThemeForWallet(owner, SlideThemeRequest{Slot: SlideSlotDashboard3, AssetID: burned.AssetID}); err != nil {
		t.Fatalf("set: %v", err)
	}

	if err := l.bondedAssets.TransferOwnership(sold.AssetID, owner, "NEWOWNER"); err != nil {
		t.Fatalf("transfer: %v", err)
	}
	if err := l.bondedAssets.Burn(burned.AssetID, owner); err != nil {
		t.Fatalf("burn: %v", err)
	}

	surface := l.SlideThemeSurfaceForWallet(owner)
	if surface.ActiveCount != 0 {
		t.Errorf("a sold/burned asset is still being worn (%d active)", surface.ActiveCount)
	}
	if v := slideSlotView(t, surface, SlideSlotDashboard2); v.Active || v.ArtKind != "default" {
		t.Errorf("the sold slot must fall back to the pack default: %+v", v)
	}
	if v := slideSlotView(t, surface, SlideSlotDashboard3); v.Active || v.ArtKind != "default" {
		t.Errorf("the burned slot must fall back to the pack default: %+v", v)
	}
	joined := strings.Join(surface.StaleSlots, " | ")
	if !strings.Contains(joined, "no longer owned and held") {
		t.Errorf("the sold slot must state why it reverted; stale=%v", surface.StaleSlots)
	}
	// A BURN drops the record outright (the asset does not exist), so only the sold slot is reported
	// as stale — the burned slot simply shows the pack default with nothing to explain.
	if len(surface.StaleSlots) != 1 {
		t.Errorf("stale slots = %v; want only the sold asset", surface.StaleSlots)
	}
}

// TestSlideTheming_SurvivesSaveAndLoad pins persistence: the map is exported and json-tagged, the
// exact bug class the registry already suffered once (unexported fields vanish from the snapshot).
func TestSlideTheming_SurvivesSaveAndLoad(t *testing.T) {
	l := newBrandingTestLobby(t)
	const owner = "OWNER"
	own, err := l.bondedAssets.MintBondedAssetWithMedia(owner, owner, owner, AssetBackground, "Mine", MoodNeutral, 0, validTestMedia())
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if _, err := l.SetSlideThemeForWallet(owner, SlideThemeRequest{Slot: SlideSlotMenu2, AssetID: own.AssetID}); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := l.bondedAssets.Save(l); err != nil {
		t.Fatalf("save: %v", err)
	}
	fresh := NewBondedAssetRegistry()
	if err := fresh.Load(l); err != nil {
		t.Fatalf("load: %v", err)
	}
	if fresh.SlideThemeCount() != 1 {
		t.Fatalf("loaded %d slide records; want 1 (unexported storage would silently drop it)", fresh.SlideThemeCount())
	}
	st, ok := fresh.GetSlideTheme(owner, SlideSlotMenu2)
	if !ok || st.AssetID != own.AssetID || !strings.EqualFold(st.ViewerWallet, owner) {
		t.Errorf("the slide record did not survive the round trip: %+v", st)
	}
	// The lookup itself is case-insensitive: the stored wallet is canonical (lowercase), so a caller
	// that speaks another casing must still reach its own record rather than a mystery empty slot.
	if _, ok := fresh.GetSlideTheme(strings.ToUpper(owner), SlideSlotMenu2); !ok {
		t.Error("GetSlideTheme must canonicalise the viewer, or a differently-cased caller loses its own theme")
	}
}

// TestSlideTheming_ClearIsIdempotent pins the reset path.
func TestSlideTheming_ClearIsIdempotent(t *testing.T) {
	l := newBrandingTestLobby(t)
	const owner = "OWNER"
	a, _ := l.bondedAssets.MintBondedAssetWithMedia(owner, owner, owner, AssetBackground, "One", MoodNeutral, 0, validTestMedia())
	b, _ := l.bondedAssets.MintBondedAssetWithMedia(owner, owner, owner, AssetBackground, "Two", MoodNeutral, 0, validTestMedia())
	if _, err := l.SetSlideThemeForWallet(owner, SlideThemeRequest{Slot: SlideSlotMenu1, AssetID: a.AssetID}); err != nil {
		t.Fatalf("set: %v", err)
	}
	if _, err := l.SetSlideThemeForWallet(owner, SlideThemeRequest{Slot: SlideSlotMenu3, AssetID: b.AssetID}); err != nil {
		t.Fatalf("set: %v", err)
	}
	removed, surface, err := l.ClearSlideThemeForWallet(owner, SlideSlotMenu1)
	if err != nil || removed != 1 {
		t.Fatalf("clear one: removed=%d err=%v", removed, err)
	}
	if surface.ActiveCount != 1 {
		t.Errorf("clearing one slot left %d active; want 1", surface.ActiveCount)
	}
	if removed, _, _ := l.ClearSlideThemeForWallet(owner, SlideSlotMenu1); removed != 0 {
		t.Errorf("clearing a slot that is already clear removed %d", removed)
	}
	removed, _, err = l.ClearSlideThemeForWallet(owner, "")
	if err != nil || removed != 1 {
		t.Fatalf("clear all: removed=%d err=%v", removed, err)
	}
	if n := l.bondedAssets.SlideThemeCount(); n != 0 {
		t.Errorf("%d slide records survive a full clear", n)
	}
	if _, _, err := l.ClearSlideThemeForWallet(owner, "not_a_slide"); err == nil {
		t.Error("an unknown slot must be refused")
	}
}

// TestSlideTheming_HTTPBoundary pins the transport rules: a body naming a target cannot even be
// parsed, a wallet-less read still serves the vocabulary and the pack art, and an unthemed request
// is refused with the rule stated.
func TestSlideTheming_HTTPBoundary(t *testing.T) {
	l := newBrandingTestLobby(t)

	// 1. A payload that tries to name a card/target is refused at the decoder.
	body := strings.NewReader(`{"slot":"menu_slide_1","asset_id":"BA-1","target_kind":"card","target_id":"DECK-1"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/assets/slide-theme?wallet=OWNER", body)
	w := httptest.NewRecorder()
	l.handleSlideTheme(w, r)
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response was not JSON: %v", err)
	}
	if resp["success"] == true {
		t.Error("a body naming a target must be refused")
	}
	if n := l.bondedAssets.SlideThemeCount(); n != 0 {
		t.Errorf("a refused request wrote %d records", n)
	}

	// 2. A wallet-less read still serves the rules and real pack art (never an empty control).
	r2 := httptest.NewRequest(http.MethodGet, "/api/assets/slide-theme", nil)
	w2 := httptest.NewRecorder()
	l.handleSlideTheme(w2, r2)
	resp = map[string]interface{}{}
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("wallet-less response was not JSON: %v", err)
	}
	if resp["wallet_required"] != true {
		t.Errorf("a wallet-less read must say so; got %v", resp["wallet_required"])
	}
	surf, _ := resp["surface"].(map[string]interface{})
	slots, _ := surf["slots"].([]interface{})
	if len(slots) != 6 {
		t.Fatalf("a wallet-less read served %d slots; want 6", len(slots))
	}
	first, _ := slots[0].(map[string]interface{})
	if first["art_kind"] != "default" || first["art_uri"] == "" {
		t.Errorf("a wallet-less read must still name the pack art: %v", first)
	}

	// 3. An unthemed POST is refused with a message a player can act on.
	r3 := httptest.NewRequest(http.MethodPost, "/api/assets/slide-theme?wallet=OWNER", strings.NewReader(`{"slot":"menu_slide_1"}`))
	w3 := httptest.NewRecorder()
	l.handleSlideTheme(w3, r3)
	if !strings.Contains(w3.Body.String(), "bonded asset is required") {
		t.Errorf("the refusal should state the rule; got %s", w3.Body.String())
	}
}
