//go:build !js && !wasm

package main

// card_view_skins_test.go
// Pins the viewer-scoped card display layer (§23.5.5). The load-bearing assertions are:
//   1. the layer's own vocabulary never names a card (so "no bonded-branding key names a card"
//      stays mechanically true);
//   2. a card-shaped request fails closed, before anything is written;
//   3. the layer CANNOT write branding state — a viewer changing how they see another player's
//      cards creates no binding, touches no card, and leaves the §10.1 card exclusion intact.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Card-view fixture wallets. Wallets are lowercase-canonical throughout the engine.
const (
	cardViewOwner = "cv-owner"
	cardViewRival = "cv-rival"
)

// TestCardViewPolicyVocabularyIsCardFree pins the schema rule: nothing this layer can STORE names
// a card, and there is no asset-free theming mode. Every mode passes the branding rule's substring
// scan, and every served scope is allowlisted and is not a card target alias. The scope vocabulary
// legitimately says "cards" to describe WHICH cards are in view — which is why the substring scan is
// not applied to it (see the file header) — but an entity-naming spelling is still refused.
func TestCardViewPolicyVocabularyIsCardFree(t *testing.T) {
	pol := CardViewPolicy()
	if pol["viewer_scoped"] != true {
		t.Error("the card-view layer must be SERVED as viewer_scoped")
	}
	if pol["cards_excluded"] != true {
		t.Error("the card-view layer must state that cards are excluded")
	}
	if pol["default_scope"] != CardViewScopeForeign {
		t.Errorf("default_scope = %v; want %s (the layer defaults to other players' cards)", pol["default_scope"], CardViewScopeForeign)
	}
	if len(cardViewModes) != 2 || len(cardViewScopes) < 3 {
		t.Fatalf("vocabulary wrong: modes=%d (engine+asset only) scopes=%d", len(cardViewModes), len(cardViewScopes))
	}
	if pol["asset_required"] != true {
		t.Error("the policy must state that a bonded asset is REQUIRED - there is no asset-free theming mode")
	}
	for _, d := range cardViewModes {
		if IsCardTargetKind(d.Kind) {
			t.Errorf("mode %q is card-shaped; mode keys must stay card-free", d.Kind)
		}
		if !isCardViewMode(d.Kind) || normalizeCardViewMode(d.Kind) != d.Kind {
			t.Errorf("mode %q is not canonical/allowlisted", d.Kind)
		}
	}
	for _, d := range cardViewScopes {
		if !isCardViewScope(d.Kind) {
			t.Errorf("scope %q is not allowlisted", d.Kind)
		}
		if cardViewRefusesCardAlias(d.Kind) {
			t.Errorf("scope %q is a card target alias; the audience vocabulary must not name a card entity", d.Kind)
		}
	}
	// The alias trap must not misfire on the audience vocabulary, and must catch entity spellings.
	if cardViewRefusesCardAlias(CardViewScopeForeign) || cardViewRefusesCardAlias(CardViewScopeOwn) || cardViewRefusesCardAlias(CardViewScopeAll) {
		t.Error("the card alias trap misfires on the audience vocabulary — every scope would be unusable")
	}
	for _, alias := range []string{"card", "Deck-Card", "hand", "hand_card", "leader_card", "religious_leader_card", "card_deck", "deck"} {
		if !cardViewRefusesCardAlias(alias) {
			t.Errorf("cardViewRefusesCardAlias(%q) = false; a card entity spelling must be refused", alias)
		}
	}
}

// TestCardViewRefusesCardShapedRequests proves the fail-closed order: a card-shaped request is
// refused with the cards-excluded error and writes nothing at all.
func TestCardViewRefusesCardShapedRequests(t *testing.T) {
	l := newBrandingTestLobby(t)
	w := cardViewOwner
	cardShaped := []CardViewRequest{
		{Mode: "deck_card", Scope: CardViewScopeForeign},
		{Mode: "hand_card", Scope: CardViewScopeForeign},
		{Mode: "religious_leader_card", Scope: CardViewScopeForeign},
		{Mode: CardViewModeAsset, AssetID: "deck_card"},
		{Mode: CardViewModeAsset, AssetID: "leader_card"},
	}
	for _, req := range cardShaped {
		_, err := l.SetCardViewForWallet(w, req)
		if err == nil || !strings.Contains(err.Error(), "cards are excluded") {
			t.Errorf("request %+v: err = %v; want the cards-excluded refusal", req, err)
		}
	}
	if n := l.bondedAssets.CardViewCount(); n != 0 {
		t.Errorf("a refused card-shaped request wrote %d card views", n)
	}
	// Unknown-but-not-card values are refused too, and for the RIGHT reason.
	if _, err := l.SetCardViewForWallet(w, CardViewRequest{Mode: "hologram"}); err == nil || !strings.Contains(err.Error(), "unknown card-view mode") {
		t.Errorf("unknown mode: err = %v", err)
	}
	if _, err := l.SetCardViewForWallet(w, CardViewRequest{Mode: CardViewModeAsset, Scope: "everyone"}); err == nil || !strings.Contains(err.Error(), "unknown card-view scope") {
		t.Errorf("unknown scope: err = %v", err)
	}
	// ASSET-FREE THEMING IS GONE. The retired `placeholder` mode is not a mode at all any more, so
	// asking for it is refused as an unknown mode instead of granting free art.
	if _, err := l.SetCardViewForWallet(w, CardViewRequest{Mode: "placeholder"}); err == nil || !strings.Contains(err.Error(), "unknown card-view mode") {
		t.Errorf("the retired asset-free mode must be refused: err = %v", err)
	}
	// ... and asset mode with no asset says what is required, rather than failing obscurely.
	if _, err := l.SetCardViewForWallet(w, CardViewRequest{Mode: CardViewModeAsset}); err == nil || !strings.Contains(err.Error(), "a bonded asset is required") {
		t.Errorf("asset mode with no asset: err = %v; want the asset-required refusal", err)
	}
	if n := l.bondedAssets.CardViewCount(); n != 0 {
		t.Errorf("refused requests wrote %d card views", n)
	}
}

// TestCardViewRequiresOwnedHeldLegitimateAsset exercises the ownership / legitimacy gates.
func TestCardViewRequiresOwnedHeldLegitimateAsset(t *testing.T) {
	l := newBrandingTestLobby(t)
	owner := cardViewOwner
	mine, err := l.bondedAssets.MintBondedAssetWithMedia(owner, owner, owner, AssetSkin, "Mine", MoodNeutral, 0, validTestMedia())
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	surface, err := l.SetCardViewForWallet(owner, CardViewRequest{Mode: CardViewModeAsset, AssetID: mine.AssetID})
	if err != nil {
		t.Fatalf("wearing my own asset was refused: %v", err)
	}
	if !surface.Active || surface.AssetID != mine.AssetID || surface.AssetName != "Mine" || surface.Media == nil {
		t.Errorf("surface = %+v; want the asset joined with its media", surface)
	}
	if surface.Scope != CardViewScopeForeign {
		t.Errorf("scope defaulted to %q; want %s", surface.Scope, CardViewScopeForeign)
	}

	theirs, err := l.bondedAssets.MintBondedAssetWithMedia(cardViewRival, cardViewRival, cardViewRival, AssetSkin, "Theirs", MoodNeutral, 0, nil)
	if err != nil {
		t.Fatalf("mint rival: %v", err)
	}
	if _, err := l.SetCardViewForWallet(owner, CardViewRequest{Mode: CardViewModeAsset, AssetID: theirs.AssetID}); err == nil || !strings.Contains(err.Error(), "owner/holder") {
		t.Errorf("wearing a rival's asset: err = %v; want the §27.8 owner/holder refusal", err)
	}

	adopted, err := l.bondedAssets.MintBondedAssetWithMedia(owner, owner, owner, AssetSkin, "Adopted", MoodNeutral, 0, nil)
	if err != nil {
		t.Fatalf("mint adopted: %v", err)
	}
	l.bondedAssets.Assets[adopted.AssetID].BlackMarketAdopted = true
	if _, err := l.SetCardViewForWallet(owner, CardViewRequest{Mode: CardViewModeAsset, AssetID: adopted.AssetID}); err == nil || !strings.Contains(err.Error(), "black-market") {
		t.Errorf("wearing off-ledger art: err = %v; want the §27.7.3 refusal", err)
	}

	// Defence in depth: the mint path cannot produce a card-typed asset (IsCardAssetType), so this
	// state is injected directly to prove the guard holds if that invariant is ever broken.
	l.bondedAssets.Assets["BA-cardtyped"] = &BondedAsset{
		AssetID: "BA-cardtyped", AssetType: AssetCard, Name: "Card", Certified: true,
		OwnerWallet: owner, HolderWallet: owner,
	}
	if _, err := l.SetCardViewForWallet(owner, CardViewRequest{Mode: CardViewModeAsset, AssetID: "BA-cardtyped"}); err == nil || !strings.Contains(err.Error(), "card-typed") {
		t.Errorf("wearing a card-typed asset: err = %v; want the card-typed refusal", err)
	}

	if _, err := l.SetCardViewForWallet(owner, CardViewRequest{Mode: CardViewModeAsset}); err == nil || !strings.Contains(err.Error(), "a bonded asset is required") {
		t.Errorf("asset mode without an asset: err = %v", err)
	}
	if _, err := l.SetCardViewForWallet(owner, CardViewRequest{Mode: CardViewModeAsset, AssetID: "BA-missing"}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("unknown asset: err = %v", err)
	}
	if _, err := l.SetCardViewForWallet("", CardViewRequest{Mode: CardViewModeAsset, AssetID: mine.AssetID}); err == nil || !strings.Contains(err.Error(), "wallet required") {
		t.Errorf("no wallet: err = %v", err)
	}
}

// TestCardViewNeverWritesBrandingState is the core safety test for "player 1 can change how they
// see player 2's cards": doing so must create NO branding binding, must not make any card a
// target, and must leave the card exclusion untouched. The guarantee is structural — this feature
// writes only registry.CardViews, never registry.Bindings.
func TestCardViewNeverWritesBrandingState(t *testing.T) {
	l := newBrandingTestLobby(t)
	owner := cardViewOwner
	a, err := l.bondedAssets.MintBondedAssetWithMedia(owner, owner, owner, AssetSkin, "Eyes", MoodNeutral, 0, validTestMedia())
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if _, err := l.SetCardViewForWallet(owner, CardViewRequest{Mode: CardViewModeAsset, AssetID: a.AssetID, Scope: CardViewScopeAll}); err != nil {
		t.Fatalf("asset-mode card view: %v", err)
	}
	// A second set REPLACES the first (one preference per viewer) and still writes no branding.
	if _, err := l.SetCardViewForWallet(owner, CardViewRequest{Mode: CardViewModeAsset, AssetID: a.AssetID, Scope: CardViewScopeForeign}); err != nil {
		t.Fatalf("second asset-mode card view: %v", err)
	}

	if n := len(l.bondedAssets.Bindings); n != 0 {
		t.Errorf("the card-view layer wrote %d branding bindings; it must never write branding", n)
	}
	if got := l.bondedAssets.BindingsForAsset(a.AssetID); len(got) != 0 {
		t.Errorf("the viewer's asset gained %d bindings from a card view", len(got))
	}
	for _, kind := range []string{"card", "cards", "deck", "deck_card", "hand", "hand_card", "leader_card", "religious_leader_card", "card_deck"} {
		if got := l.BondedBrandingForTarget(kind, "1"); len(got) != 0 {
			t.Errorf("BondedBrandingForTarget(%q) returned %d brandings; cards carry none", kind, len(got))
		}
		if got := l.BondedTargetsForWallet(owner, kind); len(got) != 0 {
			t.Errorf("BondedTargetsForWallet(%q) enumerated %d targets; cards are never enumerated", kind, len(got))
		}
		if _, err := l.BindBondedAssetToTarget(owner, a.AssetID, kind, "1"); err == nil || !strings.Contains(err.Error(), "cards are excluded") {
			t.Errorf("BindBondedAssetToTarget(%q): err = %v; the card exclusion must still refuse it", kind, err)
		}
	}
	if n := l.bondedAssets.CardViewCount(); n != 1 {
		t.Errorf("card views = %d; the second set replaces the first (one preference per viewer)", n)
	}
}

// TestCardViewIsViewerScoped proves one viewer's choice cannot be read or altered by another.
func TestCardViewIsViewerScoped(t *testing.T) {
	l := newBrandingTestLobby(t)
	ownAsset, err := l.bondedAssets.MintBondedAssetWithMedia(cardViewOwner, cardViewOwner, cardViewOwner, AssetSkin, "Mine", MoodNeutral, 0, validTestMedia())
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	rivalAsset, err := l.bondedAssets.MintBondedAssetWithMedia(cardViewRival, cardViewRival, cardViewRival, AssetSkin, "Theirs", MoodNeutral, 0, validTestMedia())
	if err != nil {
		t.Fatalf("mint rival: %v", err)
	}
	if _, err := l.SetCardViewForWallet(cardViewOwner, CardViewRequest{Mode: CardViewModeAsset, AssetID: ownAsset.AssetID, Scope: CardViewScopeOwn}); err != nil {
		t.Fatalf("set: %v", err)
	}
	mine := l.CardViewSurfaceForWallet(cardViewOwner)
	if !mine.Active || mine.AssetID != ownAsset.AssetID || mine.Scope != CardViewScopeOwn || mine.ViewerWallet != cardViewOwner {
		t.Errorf("owner surface = %+v", mine)
	}
	other := l.CardViewSurfaceForWallet(cardViewRival)
	if other.Active {
		t.Errorf("a second viewer saw the first viewer's card display: %+v (viewer scoping is broken)", other)
	}
	if other.Mode != CardViewModeEngine || other.Scope != CardViewScopeForeign {
		t.Errorf("a viewer with no record must read as engine/foreign, got %+v", other)
	}
	if _, err := l.SetCardViewForWallet(cardViewRival, CardViewRequest{Mode: CardViewModeAsset, AssetID: rivalAsset.AssetID, Scope: CardViewScopeAll}); err != nil {
		t.Fatalf("second viewer set: %v", err)
	}
	if again := l.CardViewSurfaceForWallet(cardViewOwner); again.AssetID != ownAsset.AssetID || again.Scope != CardViewScopeOwn {
		t.Errorf("another viewer's write changed the first viewer's choice: %+v", again)
	}
	if n := l.bondedAssets.CardViewCount(); n != 2 {
		t.Errorf("card views = %d; want 2 (one per viewer)", n)
	}
	// An uppercase spelling is the SAME viewer: wallets are lowercase-canonical.
	if _, err := l.SetCardViewForWallet(strings.ToUpper(cardViewOwner), CardViewRequest{Mode: CardViewModeAsset, AssetID: ownAsset.AssetID, Scope: CardViewScopeAll}); err != nil {
		t.Fatalf("uppercase set: %v", err)
	}
	if n := l.bondedAssets.CardViewCount(); n != 2 {
		t.Errorf("an uppercase wallet spelling created a duplicate record: %d", n)
	}
	if cleared, err := l.ClearCardViewForWallet(cardViewOwner); err != nil || !cleared {
		t.Errorf("clear: cleared=%v err=%v; want a real removal", cleared, err)
	}
	if cleared, _ := l.ClearCardViewForWallet(cardViewOwner); cleared {
		t.Error("a second clear reported cleared=true; clearing is idempotent")
	}
	if s := l.CardViewSurfaceForWallet(cardViewOwner); s.Active {
		t.Errorf("after clear the surface is still active: %+v", s)
	}
}

// TestCardViewPersistsAndEngineModeClears pins both the snapshot round-trip (a display choice must
// survive a restart — Persistence mandate) and the off switch.
func TestCardViewPersistsAndEngineModeClears(t *testing.T) {
	l := newBrandingTestLobby(t)
	owner := cardViewOwner
	a, err := l.bondedAssets.MintBondedAssetWithMedia(owner, owner, owner, AssetSkin, "Persisted", MoodNeutral, 0, validTestMedia())
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if _, err := l.SetCardViewForWallet(owner, CardViewRequest{Mode: CardViewModeAsset, AssetID: a.AssetID, Scope: CardViewScopeOwn}); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := l.bondedAssets.Save(l); err != nil {
		t.Fatalf("save: %v", err)
	}
	// Reload into a FRESH registry (the restart case).
	l.bondedAssets = NewBondedAssetRegistry()
	if err := l.bondedAssets.Load(l); err != nil {
		t.Fatalf("load: %v", err)
	}
	restored := l.CardViewSurfaceForWallet(owner)
	if !restored.Active || restored.AssetID != a.AssetID || restored.Scope != CardViewScopeOwn {
		t.Errorf("card view did not survive Save/Load: %+v", restored)
	}
	if restored.Media == nil {
		t.Error("the restored view did not resolve its asset's media")
	}

	view, err := l.SetCardViewForWallet(owner, CardViewRequest{Mode: CardViewModeEngine})
	if err != nil {
		t.Fatalf("engine mode: %v", err)
	}
	if view.Active {
		t.Errorf("engine mode must switch the override OFF, got %+v", view)
	}
	if n := l.bondedAssets.CardViewCount(); n != 0 {
		t.Errorf("engine mode left %d records behind", n)
	}
	if _, ok := l.bondedAssets.GetCardView(owner); ok {
		t.Error("engine mode did not clear the stored record")
	}
}

// TestCardViewBurnDropsDanglingView: a burned asset must not linger as a viewer's skin.
func TestCardViewBurnDropsDanglingView(t *testing.T) {
	l := newBrandingTestLobby(t)
	owner := cardViewOwner
	a, err := l.bondedAssets.MintBondedAssetWithMedia(owner, owner, owner, AssetSkin, "Doomed", MoodNeutral, 0, validTestMedia())
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if _, err := l.SetCardViewForWallet(owner, CardViewRequest{Mode: CardViewModeAsset, AssetID: a.AssetID}); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := l.bondedAssets.Burn(a.AssetID, owner); err != nil {
		t.Fatalf("burn: %v", err)
	}
	if _, ok := l.bondedAssets.GetCardView(owner); ok {
		t.Error("a burned asset's card view survived; the viewer would render art that no longer exists")
	}
	if s := l.CardViewSurfaceForWallet(owner); s.Active {
		t.Errorf("surface still active after the asset was burned: %+v", s)
	}
}

// TestCardViewLegacyAndDanglingRecordsReadAsOff pins the migration + integrity contract: a record
// this build can no longer honour reads as OFF (never as "active with no art"), so nothing can fake
// a theme, and a sold/transferred asset stops being worn without any client involvement.
func TestCardViewLegacyAndDanglingRecordsReadAsOff(t *testing.T) {
	l := newBrandingTestLobby(t)
	// 1. A record written by an earlier build, whose asset-free mode no longer exists.
	if err := l.bondedAssets.SetCardView(&CardView{ViewerWallet: cardViewOwner, Mode: "placeholder", Scope: CardViewScopeForeign}); err != nil {
		t.Fatalf("store legacy record: %v", err)
	}
	if s := l.CardViewSurfaceForWallet(cardViewOwner); s.Active {
		t.Errorf("a retired-mode record read as active: %+v", s)
	}
	// 2. An asset-mode record whose asset no longer resolves.
	if err := l.bondedAssets.SetCardView(&CardView{ViewerWallet: cardViewOwner, Mode: CardViewModeAsset, Scope: CardViewScopeForeign, AssetID: "BA-gone"}); err != nil {
		t.Fatalf("store dangling record: %v", err)
	}
	if s := l.CardViewSurfaceForWallet(cardViewOwner); s.Active {
		t.Errorf("a dangling record read as active: %+v", s)
	}
	// 3. A viewer who SELLS their art stops wearing it: ownership is re-proved on every read.
	sold, err := l.bondedAssets.MintBondedAssetWithMedia(cardViewOwner, cardViewOwner, cardViewOwner, AssetSkin, "Sold", MoodNeutral, 0, validTestMedia())
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if _, err := l.SetCardViewForWallet(cardViewOwner, CardViewRequest{Mode: CardViewModeAsset, AssetID: sold.AssetID}); err != nil {
		t.Fatalf("set: %v", err)
	}
	if s := l.CardViewSurfaceForWallet(cardViewOwner); !s.Active {
		t.Fatalf("the viewer's own asset did not activate the view: %+v", s)
	}
	if err := l.bondedAssets.TransferOwnership(sold.AssetID, cardViewOwner, cardViewRival); err != nil {
		t.Fatalf("transfer: %v", err)
	}
	if s := l.CardViewSurfaceForWallet(cardViewOwner); s.Active {
		t.Errorf("a sold asset is still being worn: %+v (the read path must re-prove ownership)", s)
	}
}

// TestCardViewHandlerCannotExpressACardRequest proves the HTTP boundary: a payload carrying
// target_kind / target_id / card_id is refused by the decoder, and a card-shaped VALUE is refused
// by the authority. A legitimate request succeeds and still writes no branding.
func TestCardViewHandlerCannotExpressACardRequest(t *testing.T) {
	l := newBrandingTestLobby(t)
	post := func(body string) map[string]interface{} {
		req := httptest.NewRequest(http.MethodPost, "/api/assets/card-view?wallet="+cardViewOwner, strings.NewReader(body))
		rec := httptest.NewRecorder()
		l.handleCardView(rec, req)
		var out map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("handler returned non-JSON for body %s: %v", body, err)
		}
		return out
	}
	refused := []string{
		`{"target_kind":"deck_card","target_id":"1"}`,
		`{"card_id":"card-7"}`,
		`{"mode":"deck_card"}`,
		`{"mode":"hand_card"}`,
		`{"mode":"placeholder"}`,
		`{"mode":"asset"}`,
	}
	for _, body := range refused {
		if out := post(body); out["success"] == true {
			t.Errorf("body %s was ACCEPTED at the HTTP boundary; a card request must fail closed (%v)", body, out)
		}
	}
	if n := l.bondedAssets.CardViewCount(); n != 0 {
		t.Errorf("refused HTTP requests wrote %d card views", n)
	}

	// A legitimate request wears one of the caller's OWN assets; the response also serves the
	// wallet's creation budget so a refusal can never be a dead end (§23.5.1).
	asset, err := l.bondedAssets.MintBondedAssetWithMedia(cardViewOwner, cardViewOwner, cardViewOwner, AssetSkin, "Eyes", MoodNeutral, 0, validTestMedia())
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	out := post(`{"mode":"asset","scope":"foreign_cards","asset_id":"` + asset.AssetID + `"}`)
	if out["success"] != true {
		t.Fatalf("a legitimate card-view request was refused: %v", out)
	}
	view, ok := out["view"].(map[string]interface{})
	if !ok {
		t.Fatalf("no view object in %v", out)
	}
	if view["active"] != true || view["asset_id"] != asset.AssetID || view["mode"] != "asset" || view["scope"] != "foreign_cards" {
		t.Errorf("view = %v", view)
	}
	if _, hasCap := out["capacity"].(map[string]interface{}); !hasCap {
		t.Errorf("the set response must serve the wallet's capacity budget: %v", out["capacity"])
	}
	if _, hasCardField := view["card_id"]; hasCardField {
		t.Error("the view payload exposed a card field")
	}
	if n := len(l.bondedAssets.Bindings); n != 0 {
		t.Errorf("the HTTP set path wrote %d branding bindings", n)
	}

	// GET reads it back for the SAME wallet only.
	readFor := func(w string) map[string]interface{} {
		req := httptest.NewRequest(http.MethodGet, "/api/assets/card-view?wallet="+w, nil)
		rec := httptest.NewRecorder()
		l.handleCardView(rec, req)
		var got map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("GET returned non-JSON: %v", err)
		}
		return got
	}
	mine := readFor(cardViewOwner)
	if v, _ := mine["view"].(map[string]interface{}); v == nil || v["active"] != true {
		t.Errorf("GET did not return the viewer's own record: %v", mine)
	}
	theirs := readFor(cardViewRival)
	if v, _ := theirs["view"].(map[string]interface{}); v == nil || v["active"] == true {
		t.Errorf("GET leaked another viewer's record: %v", theirs)
	}

	// No wallet: the RULES are still served (with wallet_required) rather than an empty answer.
	reqAnon := httptest.NewRequest(http.MethodGet, "/api/assets/card-view", nil)
	recAnon := httptest.NewRecorder()
	l.handleCardView(recAnon, reqAnon)
	var anon map[string]interface{}
	if err := json.Unmarshal(recAnon.Body.Bytes(), &anon); err != nil {
		t.Fatalf("anon GET returned non-JSON: %v", err)
	}
	if anon["success"] != true || anon["wallet_required"] != true {
		t.Errorf("anonymous read = %v; the policy must still be served", anon)
	}
	if pol, _ := anon["policy"].(map[string]interface{}); pol == nil || pol["cards_excluded"] != true || pol["viewer_scoped"] != true {
		t.Errorf("anonymous read did not serve the card-view policy: %v", anon["policy"])
	}

	// Method dispatch + the clear route.
	reqDel := httptest.NewRequest(http.MethodDelete, "/api/assets/card-view?wallet="+cardViewOwner, nil)
	recDel := httptest.NewRecorder()
	l.handleCardView(recDel, reqDel)
	if s := recDel.Body.String(); !strings.Contains(s, "method not allowed") {
		t.Errorf("DELETE on the card-view route = %s; want a method refusal", s)
	}
	reqClearGet := httptest.NewRequest(http.MethodGet, "/api/assets/card-view/clear?wallet="+cardViewOwner, nil)
	recClearGet := httptest.NewRecorder()
	l.handleClearCardView(recClearGet, reqClearGet)
	if s := recClearGet.Body.String(); !strings.Contains(s, "method not allowed") {
		t.Errorf("GET on the clear route = %s; clearing must be POST-only", s)
	}
	reqClear := httptest.NewRequest(http.MethodPost, "/api/assets/card-view/clear?wallet="+cardViewOwner, nil)
	recClear := httptest.NewRecorder()
	l.handleClearCardView(recClear, reqClear)
	var cleared map[string]interface{}
	if err := json.Unmarshal(recClear.Body.Bytes(), &cleared); err != nil {
		t.Fatalf("clear returned non-JSON: %v", err)
	}
	if cleared["success"] != true || cleared["cleared"] != true {
		t.Errorf("clear = %v; want cleared=true", cleared)
	}
	if n := l.bondedAssets.CardViewCount(); n != 0 {
		t.Errorf("clear left %d records", n)
	}
}





