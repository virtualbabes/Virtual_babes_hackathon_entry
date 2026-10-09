//go:build !js && !wasm

package main

// card_view_skins.go
// §23.5.5 — THE VIEWER-SCOPED CARD DISPLAY LAYER.
//
// "Player 1 can change how they see player 2's cards."
//
// This layer has exactly one subject: THE WALLET DOING THE LOOKING. A record is keyed by the
// viewer's own wallet and names only an asset that viewer owns and holds, so there is no field
// anywhere in which another player's card — or any other player's entity — could be named. The
// absence of a target field is the guarantee, not a convention:
//
//   - THE CARD EXCLUSION STAYS INTACT (§10.1). This file NEVER writes registry.Bindings. The only
//     branding writer remains bonded_branding.go's BindBondedAssetToTarget, which refuses every
//     card kind before any lookup. A card cannot become a branding target "through" this feature,
//     because this feature cannot write branding at all.
//   - A viewer cannot mutate another player's state: nothing outside the viewer's own wallet is
//     addressable here. The match authority, the replay, and the opponent's own client never see
//     this record — the read route is wallet-scoped and serves only the caller's own choice.
//   - Ownership rules are honoured exactly as on every other bonded-asset surface: §27.8
//     owner==holder==caller, §27.7.3 legitimacy (no black-market art), plus the card asset-type
//     refusal as defence in depth.
//
// Fail-closed discipline (mirrors the bind path's "refuse before any lookup"):
//
//  1. The entity-naming field is card-trapped FIRST: `mode` takes the full IsCardTargetKind
//     substring scan, and ANY field equal to a card TARGET ALIAS (deck_card,
//     hand, leader_card, …) is refused with the cards-excluded error.
//  2. `scope` is the one field the substring scan must NOT touch, and this is worth stating
//     plainly: the audience vocabulary legitimately contains the word "card", because it says
//     WHICH cards are in view (foreign_cards / own_cards / all_cards). A substring trap here
//     would refuse this layer's own valid values. Scope is therefore exhaustively allowlisted,
//     while the alias check above still refuses an entity-naming spelling such as "deck_card".
//  3. A BONDED ASSET IS REQUIRED. Modes are `engine` (off) and `asset` (wear one of your own):
//     there is no asset-free theming mode, so this layer can never be used to theme another
//     player's cards with art the player does not own. The whole vocabulary is card-free (two
//     modes, three scopes), so "no bonded-branding key names a card" stays mechanically true and
//     is pinned by a test.
//  4. The body is decoded with DisallowUnknownFields(), so a payload carrying target_kind /
//     target_id / card_id cannot even be parsed into a card-view request.
//
//  5. §23.5.1 CAPACITY. The asset a viewer wears must ALREADY EXIST (created, bought or
//     transferred in): capacity to create one equals the wallet's card/deck NFT supply
//     (BondedAssetCapacityForWallet). The read path serves the viewer's own budget, so a player is
//     never refused without being told why.
//
// Integer discipline (Architecture Ledger): no money and no float live here — the only numeric
// value is a timestamp.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ── Scope / mode / slot vocabulary (all SERVED, so no client re-declares it) ──
const (
	// CardViewScopeForeign (default) skins only cards that are NOT the viewer's own.
	CardViewScopeForeign = "foreign_cards"
	// CardViewScopeOwn skins the viewer's own cards only.
	CardViewScopeOwn = "own_cards"
	// CardViewScopeAll skins every card the viewer sees.
	CardViewScopeAll = "all_cards"

	// CardViewModeEngine leaves the engine's own card art alone (default; clears any record).
	CardViewModeEngine = "engine"
	// CardViewModeAsset wears one of the viewer's OWN bonded assets on those cards. It is the only
	// theming mode: an asset from the viewer's chest is REQUIRED (§23.5.1), so a viewer can never
	// paint another player's cards with art they do not own.
	CardViewModeAsset = "asset"
)

var cardViewScopes = []bondedTargetKind{
	{CardViewScopeForeign, "Other players' cards (default)"},
	{CardViewScopeOwn, "My own cards"},
	{CardViewScopeAll, "Every card I see"},
}

// cardViewModes is the ENTIRE theming vocabulary. There is no asset-free mode: a viewer either
// leaves the engine art alone (default) or wears one bonded asset they own - nothing else can paint
// a card face.
var cardViewModes = []bondedTargetKind{
	{CardViewModeEngine, "Engine art (off)"},
	{CardViewModeAsset, "One of my bonded assets (required)"},
}

// CardView is the viewer-scoped record. Read the ABSENT fields as the contract: there is no
// target kind, no target id, no card id and no other player's wallet. A card cannot be named
// here, so it cannot be branded here.
type CardView struct {
	ViewerWallet string    `json:"viewer_wallet"`
	Mode         string    `json:"mode"`
	Scope        string    `json:"scope"`
	AssetID      string    `json:"asset_id,omitempty"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CardViewSurface is the READ payload: the record joined with the asset's media, so a renderer
// needs one call and never has to join two reads.
type CardViewSurface struct {
	ViewerWallet string       `json:"viewer_wallet"`
	Active       bool         `json:"active"`
	Mode         string       `json:"mode"`
	Scope        string       `json:"scope"`
	AssetID      string       `json:"asset_id,omitempty"`
	AssetName    string       `json:"asset_name,omitempty"`
	Media        *BondedMedia `json:"media,omitempty"`
	UpdatedAt    time.Time    `json:"updated_at,omitempty"`
}

// CardViewRequest is the accepted body. There is deliberately no target field: a request can
// express only the caller's own art choice - and it must name one of the caller's OWN assets.
type CardViewRequest struct {
	Mode    string `json:"mode"`
	Scope   string `json:"scope"`
	AssetID string `json:"asset_id"`
}

// CardViewPolicy is the SERVED rule set for this layer.
func CardViewPolicy() map[string]interface{} {
	return map[string]interface{}{
		"viewer_scoped":     true,
		"cards_excluded":    true,
		"scopes":            cardViewScopes,
		"default_scope":     CardViewScopeForeign,
		"modes":             cardViewModes,
		// There is no asset-free mode: theming is an asset you own, tied to what you see.
		"asset_required":    true,
		"capacity_rule": "Capacity equals your card/deck NFT supply: one bonded asset per " +
			"card/deck NFT you hold (served per wallet as `capacity`).",
		"rule": "This changes only how YOU see cards — nothing is written to the cards, to their " +
			"owner, or to the match. Cards are excluded from bonded assets, so this is a display " +
			"preference and never a card binding.",
		"legitimacy": "Only your own in-game owner/holder assets may be worn, and black-market-" +
			"adopted art is refused (§27.7.3).",
	}
}

// ── Validators ────────────────────────────────────────────────────────────────
func normalizeCardViewScope(s string) string { return normalizeTargetKind(s) }
func normalizeCardViewMode(m string) string  { return normalizeTargetKind(m) }

// canonicalViewerWallet keeps ONE spelling per viewer. Wallets are lowercase-canonical in the
// engine, so "0xAB.." and "0xab.." are the same EYES (mirrors canonicalTargetID on the branding
// path): without this, a viewer spelling their own wallet in uppercase would be told they do not
// own their own asset.
func canonicalViewerWallet(wallet string) string {
	return strings.ToLower(strings.TrimSpace(wallet))
}

func isCardViewScope(s string) bool {
	for _, d := range cardViewScopes {
		if d.Kind == s {
			return true
		}
	}
	return false
}

func isCardViewMode(m string) bool {
	for _, d := range cardViewModes {
		if d.Kind == m {
			return true
		}
	}
	return false
}

// cardViewRefusesCardAlias reports whether a value names a CARD TARGET (one of the explicit card
// aliases the branding rule blocks). The SUBSTRING scan is deliberately not used here: the scope
// vocabulary says "cards" to describe WHICH cards are in view, which the target-kind rule does not
// cover. Entity-naming spellings are still refused loudly.
func cardViewRefusesCardAlias(v string) bool {
	k := normalizeTargetKind(v)
	if k == "" {
		return false
	}
	for _, alias := range cardTargetKinds {
		if k == alias {
			return true
		}
	}
	return false
}

// ── Registry storage (viewer-keyed) ───────────────────────────────────────────

// SetCardView stores (or replaces) ONE viewer's own record.
func (r *BondedAssetRegistry) SetCardView(cv *CardView) error {
	if cv == nil || cv.ViewerWallet == "" {
		return fmt.Errorf("card view requires a viewer wallet")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.CardViews == nil {
		r.CardViews = make(map[string]*CardView)
	}
	cp := *cv
	cp.ViewerWallet = strings.ToLower(cv.ViewerWallet)
	r.CardViews[cp.ViewerWallet] = &cp
	return nil
}

// GetCardView returns the viewer's own record (copy-safe snapshot).
func (r *BondedAssetRegistry) GetCardView(viewerWallet string) (*CardView, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cv, ok := r.CardViews[strings.ToLower(viewerWallet)]
	if !ok || cv == nil {
		return nil, false
	}
	cp := *cv
	return &cp, true
}

// ClearCardView removes the viewer's own record, reporting whether one existed.
func (r *BondedAssetRegistry) ClearCardView(viewerWallet string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := strings.ToLower(viewerWallet)
	if _, ok := r.CardViews[k]; !ok {
		return false
	}
	delete(r.CardViews, k)
	return true
}

// CardViewCount reports how many viewers have a card-display record (diagnostics only).
func (r *BondedAssetRegistry) CardViewCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.CardViews)
}

// ── Lobby authority ───────────────────────────────────────────────────────────

// SetCardViewForWallet sets the caller's OWN card-display record. `engine` mode clears it, and
// `asset` mode requires one of the CALLER'S OWN bonded assets (§23.5.1: theming is an asset you
// hold, never art you do not own). Every refusal happens before anything is written, so a refused
// request changes nothing.
func (l *Lobby) SetCardViewForWallet(wallet string, req CardViewRequest) (*CardViewSurface, error) {
	if l.bondedAssets == nil {
		return nil, fmt.Errorf("registry not initialized")
	}
	// One spelling per viewer, so the ownership comparison and the stored record agree.
	wallet = canonicalViewerWallet(wallet)
	if wallet == "" {
		return nil, fmt.Errorf("wallet required")
	}
	// 1. Card-shaped ENTITIES fail closed before any lookup: the alias list covers every field, and
	// the full substring scan covers the one field whose vocabulary cannot legitimately say "cards"
	// (mode). This function never writes registry.Bindings.
	if cardViewRefusesCardAlias(req.Scope) || cardViewRefusesCardAlias(req.Mode) ||
		cardViewRefusesCardAlias(req.AssetID) || IsCardTargetKind(req.Mode) {
		return nil, fmt.Errorf("cards are excluded from bonded assets; a card-view request cannot name a card")
	}
	mode := normalizeCardViewMode(req.Mode)
	if mode == "" {
		mode = CardViewModeEngine
	}
	if !isCardViewMode(mode) {
		return nil, fmt.Errorf("unknown card-view mode %q", req.Mode)
	}
	scope := normalizeCardViewScope(req.Scope)
	if scope == "" {
		scope = CardViewScopeForeign
	}
	if !isCardViewScope(scope) {
		return nil, fmt.Errorf("unknown card-view scope %q", req.Scope)
	}
	if mode == CardViewModeEngine {
		if _, err := l.ClearCardViewForWallet(wallet); err != nil {
			return nil, err
		}
		surface := l.CardViewSurfaceForWallet(wallet)
		return &surface, nil
	}

	cv := &CardView{ViewerWallet: wallet, Mode: mode, Scope: scope, UpdatedAt: time.Now()}
	switch mode {
	case CardViewModeAsset:
		if req.AssetID == "" {
			// Theming requires a bonded asset (§23.5.1): there is no asset-free mode, so a viewer
			// must first create/acquire one and put it in their chest.
			return nil, fmt.Errorf("a bonded asset is required: create or acquire one in the bonded " +
				"assets chest, then tie it to your card view")
		}
		a, ok := l.bondedAssets.Get(req.AssetID)
		if !ok {
			return nil, fmt.Errorf("asset %s not found", req.AssetID)
		}
		// §27.8: only the in-game owner/holder (both) may wear their own art.
		if a.OwnerWallet != wallet || a.HolderWallet != wallet {
			return nil, fmt.Errorf("asset %s may only be worn by its in-game owner/holder (both must match)", req.AssetID)
		}
		// §27.7.3: off-ledger art may not masquerade as legitimate branding.
		if a.BlackMarketAdopted {
			return nil, fmt.Errorf("asset %s is black-market-adopted and may not be worn (§27.7.3)", req.AssetID)
		}
		// Defence in depth: the mint path already refuses card asset types (IsCardAssetType).
		if IsCardAssetType(a.AssetType) {
			return nil, fmt.Errorf("asset %s is a card-typed asset; cards are excluded from bonded assets", req.AssetID)
		}
		cv.AssetID = req.AssetID
	}
	if err := l.bondedAssets.SetCardView(cv); err != nil {
		return nil, err
	}
	surface := l.CardViewSurfaceForWallet(wallet)
	return &surface, nil
}

// ClearCardViewForWallet removes the caller's own record (idempotent).
func (l *Lobby) ClearCardViewForWallet(wallet string) (bool, error) {
	if l.bondedAssets == nil {
		return false, fmt.Errorf("registry not initialized")
	}
	wallet = canonicalViewerWallet(wallet)
	if wallet == "" {
		return false, fmt.Errorf("wallet required")
	}
	return l.bondedAssets.ClearCardView(wallet), nil
}

// CardViewSurfaceForWallet is the read path: the viewer's own record joined with its media.
// A viewer with no record answers Active:false (the engine's own art) — a real state, not "no data".
func (l *Lobby) CardViewSurfaceForWallet(wallet string) CardViewSurface {
	wallet = canonicalViewerWallet(wallet)
	out := CardViewSurface{
		ViewerWallet: wallet,
		Active:       false,
		Mode:         CardViewModeEngine,
		Scope:        CardViewScopeForeign,
	}
	if l.bondedAssets == nil || wallet == "" {
		return out
	}
	cv, ok := l.bondedAssets.GetCardView(wallet)
	if !ok {
		return out
	}
	// A record that is not one of TODAY'S modes cannot be honoured - for example the retired
	// asset-free `placeholder` record an earlier build could store. It reads as OFF rather than
	// claiming to be active with no art, and the next set replaces it.
	if cv.Mode == CardViewModeEngine || !isCardViewMode(cv.Mode) || cv.AssetID == "" {
		return out
	}
	out.Active = true
	out.Mode = cv.Mode
	out.Scope = cv.Scope
	out.AssetID = cv.AssetID
	out.UpdatedAt = cv.UpdatedAt
	if cv.AssetID != "" {
		a, ok := l.bondedAssets.Get(cv.AssetID)
		if !ok {
			// The asset no longer resolves (burned, or a corrupted record): show the engine art rather
			// than pretending to wear art that does not exist.
			return CardViewSurface{ViewerWallet: wallet, Active: false, Mode: CardViewModeEngine, Scope: cv.Scope}
		}
		// Ownership is re-proved on EVERY read (§27.8): a viewer who has SOLD, transferred or had their
		// art black-market-adopted must not keep wearing it. Case-insensitively, because wallets are
		// canonical but a stored spelling may differ.
		if !strings.EqualFold(a.OwnerWallet, wallet) || !strings.EqualFold(a.HolderWallet, wallet) || a.BlackMarketAdopted {
			return CardViewSurface{ViewerWallet: wallet, Active: false, Mode: CardViewModeEngine, Scope: cv.Scope}
		}
		out.AssetName = a.Name
		out.Media = a.Media
	}
	return out
}

// ── HTTP surface ──────────────────────────────────────────────────────────────

// handleCardView serves the caller's own card-display preference: GET reads, POST sets.
func (l *Lobby) handleCardView(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		l.cardViewRead(w, r)
	case http.MethodPost:
		l.cardViewSet(w, r)
	default:
		writeJSON(w, map[string]interface{}{"success": false, "error": "method not allowed"})
	}
}

func (l *Lobby) cardViewRead(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		// The RULES are public; only the preference is wallet-scoped — so a disconnected visitor
		// still learns the rule instead of staring at an unexplained empty control.
		writeJSON(w, map[string]interface{}{
			"success":         true,
			"wallet":          "",
			"wallet_required": true,
			"note":            "connect a wallet to read your own card-display preference",
			"view":            CardViewSurface{Active: false, Mode: CardViewModeEngine, Scope: CardViewScopeForeign},
			"capacity":        l.BondedAssetCapacityForWallet(""),
			"policy":          CardViewPolicy(),
		})
		return
	}
	writeJSON(w, map[string]interface{}{
		"success":  true,
		"wallet":   wallet,
		"view":     l.CardViewSurfaceForWallet(wallet),
		"capacity": l.BondedAssetCapacityForWallet(wallet),
		"policy":   CardViewPolicy(),
	})
}

func (l *Lobby) cardViewSet(w http.ResponseWriter, r *http.Request) {
	if l.bondedAssets == nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "registry not initialized"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	var req CardViewRequest
	dec := json.NewDecoder(io.LimitReader(r.Body, 8<<10))
	// Unknown fields are refused, so target_kind / target_id / card_id cannot even be expressed.
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "invalid card-view body: only mode, scope and asset_id are accepted (a card request cannot be expressed)",
		})
		return
	}
	view, err := l.SetCardViewForWallet(wallet, req)
	if err != nil {
		// A refusal carries the wallet's own creation budget, so "you have nothing to wear" is
		// never a dead end (§23.5.1).
		writeJSON(w, map[string]interface{}{
			"success":  false,
			"error":    err.Error(),
			"capacity": l.BondedAssetCapacityForWallet(wallet),
		})
		return
	}
	writeJSON(w, map[string]interface{}{
		"success":  true,
		"wallet":   wallet,
		"view":     view,
		"capacity": l.BondedAssetCapacityForWallet(wallet),
		"policy":   CardViewPolicy(),
	})
}

func (l *Lobby) handleClearCardView(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, map[string]interface{}{"success": false, "error": "method not allowed"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	cleared, err := l.ClearCardViewForWallet(wallet)
	if err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{
		"success": true,
		"wallet":  wallet,
		"cleared": cleared,
		"view":    l.CardViewSurfaceForWallet(wallet),
		"policy":  CardViewPolicy(),
	})
}




