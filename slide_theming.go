//go:build !js && !wasm

package main

// slide_theming.go
// §10.6 — VIEWER-SCOPED SLIDE THEMING (app boot / main menu + World Dashboard backgrounds).
//
// "the slide shows can have each image re-themed by a user with bonded assets"
//
// This layer has exactly one subject: THE WALLET DOING THE LOOKING. A record is keyed by the
// viewer's own wallet and names only an asset that viewer owns and holds, so there is no field in
// which another player's entity could be named. The absence of a target field is the guarantee,
// not a convention:
//
//   - NO CARD IS EVER INVOLVED. The vocabulary is six slide slots and nothing else; the layer
//     writes no ThemeBinding.TargetKind/TargetID and NEVER writes registry.Bindings, so it cannot
//     brand anything — the only branding writer remains BindBondedAssetToTarget, which refuses
//     every card kind before any lookup (§10.1 stays structural).
//   - A THEME REQUIRES A BONDED ASSET. Either a slot names one of the viewer's own assets, or the
//     slot is OFF and the pack's default frame is shown. There is no asset-free re-theme, so a
//     slide cannot be painted with art the player does not own.
//   - §27.8 IS RE-PROVED ON EVERY READ (owner == holder == viewer) and a black-market asset
//     (§27.7.3) is refused, so an asset that was SOLD or transferred stops being worn the next
//     time the viewer's own client reads its surface — a stale record can never outlive ownership.
//
// The DEFAULT art of every slot is the placeholder catalogue (placeholder_assets.go), which is also
// what the free starter pack grants, so a brand-new wallet already owns the art its slides show.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ── Slot vocabulary (the whole of it — served, so no client re-declares it) ───
const (
	SlideSlotMenu1      = "menu_slide_1"
	SlideSlotMenu2      = "menu_slide_2"
	SlideSlotMenu3      = "menu_slide_3"
	SlideSlotDashboard1 = "dashboard_slide_1"
	SlideSlotDashboard2 = "dashboard_slide_2"
	SlideSlotDashboard3 = "dashboard_slide_3"
)

type slideSlotDef struct {
	ID         string
	Label      string
	Show       string // "menu" (app boot / main menu) or "dashboard"
	DefaultSKU string
}

// slideSlotDefs is the ONE declaration of the six slides and the pack frame each shows by default.
// It is deliberately card-free vocabulary (the word "card" appears nowhere), and a test pins that.
var slideSlotDefs = []slideSlotDef{
	{SlideSlotMenu1, "App boot / main menu — slide 1", "menu", "npc-anya-001"},
	{SlideSlotMenu2, "App boot / main menu — slide 2", "menu", "npc-crypto-seraph-002"},
	{SlideSlotMenu3, "App boot / main menu — slide 3", "menu", "npc-vbabes-046"},
	{SlideSlotDashboard1, "World Dashboard — slide 1", "dashboard", "npc-vbabes-054"},
	{SlideSlotDashboard2, "World Dashboard — slide 2", "dashboard", "npc-crypto-seraph-004"},
	{SlideSlotDashboard3, "World Dashboard — slide 3", "dashboard", "npc-anya-041"},
}

func isSlideSlot(slot string) bool {
	for _, d := range slideSlotDefs {
		if d.ID == slot {
			return true
		}
	}
	return false
}

func slideSlotDefFor(slot string) (slideSlotDef, bool) {
	for _, d := range slideSlotDefs {
		if d.ID == slot {
			return d, true
		}
	}
	return slideSlotDef{}, false
}

// SlideThemeSlotView is the READ payload for one slot: the default pack art joined with whatever
// the viewer has chosen, so a renderer needs no second lookup and no guess.
type SlideThemeSlotView struct {
	Slot       string       `json:"slot"`
	Label      string       `json:"label"`
	Show       string       `json:"show"`
	DefaultSKU string       `json:"default_sku"`
	DefaultURI string       `json:"default_uri"`
	// DefaultSourceURI is the pack ORIGINAL the default art was derived from; DefaultDerived says
	// whether DefaultURI is a LIGHT rendition of it (placeholder_derivatives.go) or the original
	// itself. Both are stated, so a client never implies light art IS the shipped art.
	DefaultSourceURI string `json:"default_source_uri,omitempty"`
	DefaultDerived   bool   `json:"default_derived,omitempty"`
	DefaultNote      string `json:"default_note,omitempty"`
	Active     bool         `json:"active"`
	AssetID    string       `json:"asset_id,omitempty"`
	AssetName  string       `json:"asset_name,omitempty"`
	Media      *BondedMedia `json:"media,omitempty"`
	// ArtURI is what to paint NOW: the viewer's own art when the slot is active, otherwise the
	// pack default. ArtKind says which of the two it is, so the client never has to infer it.
	ArtURI  string `json:"art_uri"`
	ArtKind string `json:"art_kind"` // "asset" | "default" | "" (neither could be resolved)
}

// SlideThemeSurface is the whole viewer-scoped payload.
type SlideThemeSurface struct {
	ViewerWallet string               `json:"viewer_wallet"`
	Slots        []SlideThemeSlotView `json:"slots"`
	ActiveCount  int                  `json:"active_count"`
	// StaleSlots names any slot whose stored art can no longer be worn (sold, transferred, burned,
	// black-market-adopted, card-typed or artless). The slot renders its pack default AND the reason
	// is stated, so a reverted slide is never a silent mystery.
	StaleSlots []string `json:"stale_slots,omitempty"`
}

// SlideThemingPolicy is SERVED: slot list, default art, and the rule in words.
func SlideThemingPolicy() map[string]interface{} {
	slots := make([]map[string]interface{}, 0, len(slideSlotDefs))
	for _, d := range slideSlotDefs {
		entry := map[string]interface{}{
			"slot":        d.ID,
			"label":       d.Label,
			"show":        d.Show,
			"default_sku": d.DefaultSKU,
		}
		if s, ok := placeholderSku(d.DefaultSKU); ok {
			entry["default_uri"] = s.URI
			entry["default_available"] = s.Purchasable
			if !s.Purchasable {
				entry["default_note"] = s.Unavailable
			}
			// The light rendition is named beside the original, so a client can paint the small
			// copy and still say which shipped frame it came from.
			entry["default_source_uri"] = s.URI
			if light, derived := PlaceholderDerivativeURIForSKU(s.SKU, PlaceholderDerivativeSlideTier); derived {
				entry["default_uri"] = light
				entry["default_derived"] = true
			} else if rec, recOK := PlaceholderDerivativeRecordFor(s.SKU); recOK && rec.Slide != nil && rec.Slide.Passthrough {
				entry["default_derived"] = false
				entry["derivative_note"] = "the original frame is already at or below the light rendition width, so no smaller copy exists"
			} else {
				entry["default_derived"] = false
				entry["derivative_note"] = "light rendition pending: the original frame is shown until the server has derived it"
			}
		} else {
			entry["default_available"] = false
			entry["default_note"] = "the shipped art pack does not contain " + d.DefaultSKU
		}
		slots = append(slots, entry)
	}
	return map[string]interface{}{
		"slots":          slots,
		"show":           []string{"menu", "dashboard"},
		// §10.6 DERIVATIVES: whether the light renditions exist yet. A client can paint the small
		// copy and re-read once when the pass finishes, instead of fetching the originals forever.
		"derivatives": PlaceholderDerivativeStatusForClient(),
		"asset_required": true,
		"rule": "Re-theming a slide requires one of YOUR OWN bonded assets — create one, buy one " +
			"from the placeholder catalogue, or use the frames the free starter pack already " +
			"granted you (the six the slides show by default). A slide with no asset of yours " +
			"simply shows the pack default.",
		"guarantees": "This is a VIEW of your own screens: the record is keyed by YOUR wallet, names " +
			"only YOUR asset (there is no target field), and is re-checked against ownership and " +
			"holding on every read — so art you sold or transferred stops being worn, and branding " +
			"is never written by this feature.",
		"capacity": "Creating a bonded asset is capped at your card/deck NFT supply (§23.5.1); the " +
			"starter pack is granted free and catalogue frames are purchased, so theming is " +
			"reachable without spending a creation slot.",
	}
}

// SlideThemeRequest is the ONLY accepted write body. It decodes with DisallowUnknownFields(), so a
// payload carrying target_kind / target_id / entity / card_id cannot even be parsed: this layer
// has no way to be told to theme anything but the caller's own slides.
type SlideThemeRequest struct {
	Slot    string `json:"slot"`
	AssetID string `json:"asset_id"`
}

// SlideTheme is ONE viewer's choice for ONE slide. It names only the VIEWER'S OWN asset — there is
// no target field, by design, so another player's slide, entity or card cannot be expressed here.
type SlideTheme struct {
	ViewerWallet string    `json:"viewer_wallet"`
	Slot         string    `json:"slot"`
	AssetID      string    `json:"asset_id"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ── Registry storage (mirrors the CardViews precedent, so persistence is uniform) ──
// Keyed "<viewer>:<slot>", with BOTH halves canonicalised: wallets are lowercase-canonical in this
// engine and slot ids are lowercase by declaration, so two spellings of one viewer/slot are ONE
// record rather than two. Every accessor below canonicalises, so no caller can get the key wrong.
func slideThemeKey(viewer, slot string) string {
	return canonicalViewerWallet(viewer) + ":" + strings.ToLower(strings.TrimSpace(slot))
}

// SetSlideTheme writes (or replaces) one viewer's slot record. The copy is stored, never the
// caller's pointer, so a caller cannot mutate stored state after the fact.
func (r *BondedAssetRegistry) SetSlideTheme(st *SlideTheme) error {
	if st == nil || st.ViewerWallet == "" || st.Slot == "" {
		return fmt.Errorf("viewer_wallet and slot are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.SlideThemes == nil {
		r.SlideThemes = make(map[string]*SlideTheme)
	}
	cp := *st
	cp.ViewerWallet = canonicalViewerWallet(cp.ViewerWallet)
	cp.Slot = strings.ToLower(strings.TrimSpace(cp.Slot))
	r.SlideThemes[slideThemeKey(cp.ViewerWallet, cp.Slot)] = &cp
	return nil
}

// GetSlideTheme reads one viewer's slot record (a copy, so nothing outside can mutate storage).
func (r *BondedAssetRegistry) GetSlideTheme(viewer, slot string) (*SlideTheme, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	st, ok := r.SlideThemes[slideThemeKey(viewer, slot)]
	if !ok || st == nil {
		return nil, false
	}
	cp := *st
	return &cp, true
}

// SlideThemesFor returns every slide this viewer has themed, keyed by slot.
func (r *BondedAssetRegistry) SlideThemesFor(viewer string) map[string]*SlideTheme {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]*SlideTheme)
	prefix := canonicalViewerWallet(viewer) + ":"
	for k, st := range r.SlideThemes {
		if st == nil || !strings.HasPrefix(k, prefix) {
			continue
		}
		cp := *st
		out[cp.Slot] = &cp
	}
	return out
}

// ClearSlideTheme removes one slot (or every slot when slot is empty) for a viewer, returning how
// many records were removed. Clearing what is not there is not an error (idempotent).
func (r *BondedAssetRegistry) ClearSlideTheme(viewer, slot string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	removed := 0
	if slot != "" {
		k := slideThemeKey(viewer, slot)
		if _, ok := r.SlideThemes[k]; ok {
			delete(r.SlideThemes, k)
			removed++
		}
		return removed
	}
	prefix := canonicalViewerWallet(viewer) + ":"
	for k := range r.SlideThemes {
		if strings.HasPrefix(k, prefix) {
			delete(r.SlideThemes, k)
			removed++
		}
	}
	return removed
}

// SlideThemeCount reports how many slide-theme records exist (diagnostics only).
func (r *BondedAssetRegistry) SlideThemeCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.SlideThemes)
}

// ── Authority ────────────────────────────────────────────────────────────────

// slideSlotIDs lists the slot ids for an error message (so a refusal teaches the vocabulary).
func slideSlotIDs() string {
	ids := make([]string, 0, len(slideSlotDefs))
	for _, d := range slideSlotDefs {
		ids = append(ids, d.ID)
	}
	return strings.Join(ids, ", ")
}

// slideSlotDefaultView builds the OFF view for one slot: the pack frame it shows by default.
func slideSlotDefaultView(d slideSlotDef) SlideThemeSlotView {
	v := SlideThemeSlotView{Slot: d.ID, Label: d.Label, Show: d.Show, DefaultSKU: d.DefaultSKU}
	if s, ok := placeholderSku(d.DefaultSKU); ok {
		v.DefaultURI = s.URI
		v.DefaultSourceURI = s.URI
		// §10.6: paint the LIGHT rendition when the server has derived one, so the boot screen
		// stops fetching multi-megabyte art. The original is still named, and a pending or
		// unnecessary derivation is stated rather than silently serving the heavy file forever.
		if light, derived := PlaceholderDerivativeURIForSKU(s.SKU, PlaceholderDerivativeSlideTier); derived {
			v.DefaultURI = light
			v.DefaultDerived = true
		} else if rec, recOK := PlaceholderDerivativeRecordFor(s.SKU); recOK && rec.Slide != nil && rec.Slide.Passthrough {
			v.DefaultNote = "the original frame is already at or below the light rendition width, so no smaller copy exists"
		} else {
			v.DefaultNote = "light rendition pending: the original frame is shown until the server has derived it"
		}
		v.ArtURI = v.DefaultURI
		v.ArtKind = "default"
	}
	return v
}

// SetSlideThemeForWallet is the ONE write path for slide theming. Every refusal happens BEFORE the
// record is written, so a refused re-theme changes nothing at all.
func (l *Lobby) SetSlideThemeForWallet(wallet string, req SlideThemeRequest) (*SlideThemeSurface, error) {
	if l.bondedAssets == nil {
		return nil, fmt.Errorf("registry not initialized")
	}
	w := canonicalViewerWallet(wallet)
	if w == "" {
		return nil, fmt.Errorf("wallet required")
	}
	slot := strings.ToLower(strings.TrimSpace(req.Slot))
	def, ok := slideSlotDefFor(slot)
	if !ok {
		return nil, fmt.Errorf("unknown slide slot %q (slots: %s)", req.Slot, slideSlotIDs())
	}
	assetID := strings.TrimSpace(req.AssetID)
	if assetID == "" {
		return nil, fmt.Errorf("a bonded asset is required to re-theme %s: use one you own — the "+
			"free starter pack already granted the frames these slides show — then pick it for "+
			"this slide", slot)
	}
	a, ok := l.bondedAssets.Get(assetID)
	if !ok || a == nil {
		return nil, fmt.Errorf("asset %s not found (it may have been burned)", assetID)
	}
	// Defence in depth: a card-typed asset must never be reachable from any theming surface (§10.1).
	if IsCardAssetType(a.AssetType) {
		return nil, fmt.Errorf("asset %s is card-typed and cards are excluded from theming", assetID)
	}
	// §27.8: only the in-game owner AND holder may wear the asset — and only on their own screens.
	if !strings.EqualFold(a.OwnerWallet, w) || !strings.EqualFold(a.HolderWallet, w) {
		return nil, fmt.Errorf("you may only wear an asset you own and hold (asset %s is owned or "+
			"held by another wallet)", assetID)
	}
	if a.BlackMarketAdopted {
		return nil, fmt.Errorf("a black-market-adopted asset cannot theme a screen the whole "+
			"civilization sees (§27.7.3): asset %s", assetID)
	}
	if a.Media == nil {
		return nil, fmt.Errorf("asset %s has no art yet, so there is nothing to paint on %s", assetID, slot)
	}
	if err := l.bondedAssets.SetSlideTheme(&SlideTheme{
		ViewerWallet: w, Slot: slot, AssetID: assetID, UpdatedAt: time.Now(),
	}); err != nil {
		return nil, err
	}
	surface := l.SlideThemeSurfaceForWallet(w)
	l.logAdminAudit("SLIDE_THEMED", w, fmt.Sprintf("%s (%s) <- %s", slot, def.Label, assetID))
	return &surface, nil
}

// ClearSlideThemeForWallet removes one slot, or every slot when slot is empty. Idempotent.
func (l *Lobby) ClearSlideThemeForWallet(wallet, slot string) (int, *SlideThemeSurface, error) {
	if l.bondedAssets == nil {
		return 0, nil, fmt.Errorf("registry not initialized")
	}
	w := canonicalViewerWallet(wallet)
	if w == "" {
		return 0, nil, fmt.Errorf("wallet required")
	}
	s := strings.ToLower(strings.TrimSpace(slot))
	if s != "" && !isSlideSlot(s) {
		return 0, nil, fmt.Errorf("unknown slide slot %q (slots: %s)", slot, slideSlotIDs())
	}
	removed := l.bondedAssets.ClearSlideTheme(w, s)
	surface := l.SlideThemeSurfaceForWallet(w)
	return removed, &surface, nil
}

// SlideThemeSurfaceForWallet is the read path: every slot, the viewer's own choice where one exists,
// and the pack default everywhere else. Ownership (§27.8), legitimacy (§27.7.3) and the card
// exclusion are RE-PROVED here on EVERY read, so a record can never outlive the ownership it was
// written under and a stale slot is reported rather than silently honoured.
func (l *Lobby) SlideThemeSurfaceForWallet(wallet string) SlideThemeSurface {
	w := canonicalViewerWallet(wallet)
	out := SlideThemeSurface{
		ViewerWallet: w,
		Slots:        make([]SlideThemeSlotView, 0, len(slideSlotDefs)),
		StaleSlots:   make([]string, 0),
	}
	chosen := map[string]*SlideTheme{}
	if l.bondedAssets != nil && w != "" {
		chosen = l.bondedAssets.SlideThemesFor(w)
	}
	for _, d := range slideSlotDefs {
		view := slideSlotDefaultView(d)
		st := chosen[d.ID]
		if st == nil || st.AssetID == "" {
			out.Slots = append(out.Slots, view)
			continue
		}
		a, ok := l.bondedAssets.Get(st.AssetID)
		switch {
		case !ok || a == nil:
			out.StaleSlots = append(out.StaleSlots, d.ID+": "+st.AssetID+" no longer exists (burned)")
		case !strings.EqualFold(a.OwnerWallet, w), !strings.EqualFold(a.HolderWallet, w):
			out.StaleSlots = append(out.StaleSlots, d.ID+": "+st.AssetID+" is no longer owned and held by you")
		case a.BlackMarketAdopted:
			out.StaleSlots = append(out.StaleSlots, d.ID+": "+st.AssetID+" is black-market-adopted")
		case IsCardAssetType(a.AssetType):
			out.StaleSlots = append(out.StaleSlots, d.ID+": "+st.AssetID+" is card-typed")
		case a.Media == nil:
			out.StaleSlots = append(out.StaleSlots, d.ID+": "+st.AssetID+" has no art")
		default:
			view.Active = true
			view.AssetID = a.AssetID
			view.AssetName = a.Name
			view.Media = a.Media
			view.ArtURI = a.Media.URI
			view.ArtKind = "asset"
			out.ActiveCount++
		}
		out.Slots = append(out.Slots, view)
	}
	return out
}

// ── HTTP ─────────────────────────────────────────────────────────────────────

// handleSlideTheme reads or writes the CALLER'S OWN slide theming. GET always answers (a visitor
// with no wallet still learns the vocabulary and the default art); POST requires a wallet.
func (l *Lobby) handleSlideTheme(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		l.slideThemeRead(w, r)
	case http.MethodPost:
		l.slideThemeSet(w, r)
	default:
		writeJSON(w, map[string]interface{}{"success": false, "error": "method not allowed"})
	}
}

func (l *Lobby) slideThemeRead(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		// The RULES and the default art are public; only the preference is wallet-scoped.
		writeJSON(w, map[string]interface{}{
			"success":         true,
			"wallet":          "",
			"wallet_required": true,
			"note":            "connect a wallet to read your own slide theming",
			"surface":         l.SlideThemeSurfaceForWallet(""),
			"policy":          SlideThemingPolicy(),
			"capacity":        l.BondedAssetCapacityForWallet(""),
		})
		return
	}
	writeJSON(w, map[string]interface{}{
		"success":  true,
		"wallet":   wallet,
		"surface":  l.SlideThemeSurfaceForWallet(wallet),
		"policy":   SlideThemingPolicy(),
		"capacity": l.BondedAssetCapacityForWallet(wallet),
	})
}

func (l *Lobby) slideThemeSet(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	var req SlideThemeRequest
	dec := json.NewDecoder(io.LimitReader(r.Body, 4<<10))
	// Unknown fields are refused, so target_kind / target_id / entity / card_id cannot be expressed.
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error": "invalid slide-theme body: only slot and asset_id are accepted (there is no " +
				"field in which another player's art, entity or card could be named)",
		})
		return
	}
	surface, err := l.SetSlideThemeForWallet(wallet, req)
	if err != nil {
		// A refusal carries the caller's own surface + budget, so "why can I not see this?" is
		// answered in the same payload the client already renders.
		writeJSON(w, map[string]interface{}{
			"success":  false,
			"error":    err.Error(),
			"surface":  l.SlideThemeSurfaceForWallet(wallet),
			"capacity": l.BondedAssetCapacityForWallet(wallet),
		})
		return
	}
	writeJSON(w, map[string]interface{}{
		"success":  true,
		"wallet":   wallet,
		"surface":  surface,
		"policy":   SlideThemingPolicy(),
		"capacity": l.BondedAssetCapacityForWallet(wallet),
	})
}

// handleClearSlideTheme takes a slot off (or every slot when the body is empty / names none).
func (l *Lobby) handleClearSlideTheme(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, map[string]interface{}{"success": false, "error": "method not allowed"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	var req struct {
		Slot string `json:"slot"`
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, 4<<10))
	dec.DisallowUnknownFields()
	// An empty body is the documented "reset every slide" call, so io.EOF is not an error here.
	if err := dec.Decode(&req); err != nil && err != io.EOF {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "invalid clear body: only slot is accepted (omit it to clear every slide)",
		})
		return
	}
	removed, surface, err := l.ClearSlideThemeForWallet(wallet, req.Slot)
	if err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{
		"success": true,
		"wallet":  wallet,
		"cleared": removed,
		"surface": surface,
		"policy":  SlideThemingPolicy(),
	})
}
