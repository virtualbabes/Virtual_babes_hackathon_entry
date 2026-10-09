//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AssetType enumerates the bonded-asset categories (Ã‚Â§23.5 / Ã‚Â§27.7).
type AssetType int

const (
	AssetSkin       AssetType = iota // 0
	AssetBackground                  // 1
	AssetBoard                       // 2
	AssetButton                      // 3
	AssetAppearance                  // 4
	AssetAudio                       // 5
	AssetCustom                      // 6
)

// MoodTag enum (canonical lock, Ã‚Â§23.5): 0=NEUTRAL (Go zero-default, no bias),
// 1=BENEVOLENT (+bias), 2=MALEVOLENT (-bias). NO confidence field (PILLAR 2 deterministic).
const (
	MoodNeutral    = 0
	MoodBenevolent = 1
	MoodMalevolent = 2
)

// BondedAsset is a player-created, user-named cosmetic NFT bound to the game hub
// lease (Ã‚Â§23.5). It is the registry entity that Ã‚Â§27.6 (ThemeBinding lock) and
// Ã‚Â§27.8 (owner+holder transfer guard) operate on.
type BondedAsset struct {
	AssetID       string    `json:"asset_id"` // "BA-" + uuid
	AssetType     AssetType `json:"asset_type"`
	Name          string    `json:"name"` // user-customizable at creation (design rule)
	CreatorWallet string    `json:"creator_wallet"`
	OwnerWallet   string    `json:"owner_wallet"`
	HolderWallet  string    `json:"holder_wallet"` // may equal OwnerWallet; Ã‚Â§27.8 needs both
	MoodTag       int       `json:"mood_tag"`      // 0/1/2
	RoyaltyBps    int       `json:"royalty_bps"`   // <= 1000
	CreatedAt     time.Time `json:"created_at"`

	// Ã‚Â§27.7.3 scaffold (populated later, not enforced here):
	BirthCertID        string `json:"birth_cert_id,omitempty"`
	Certified          bool   `json:"certified,omitempty"`
	BlackMarketAdopted bool   `json:"black_market_adopted,omitempty"`

	// Media is the USER-SUPPLIED image/audio/video the asset represents (Â§23.5 branding).
	// It is what makes a bonded asset a branding layer: a player ties their own art to an
	// entity they own (item, theme, bot, pet, vehicle, church, club, shop, world content).
	// The payload is integer-only plus opaque strings â€” no float ever reaches the ledger.
	Media *BondedMedia `json:"media,omitempty"`

	// Sku / Source identify a HOUSE-CATALOGUE asset (placeholder_assets.go): Sku is the pack frame
	// it was minted from, Source records HOW it was acquired ("starter" free grant or "shop"
	// purchase). Both are json-tagged so they survive Save/Load, and both are empty for a normal
	// player-minted asset â€” so a client can always tell a house frame from a player creation.
	Sku    string `json:"sku,omitempty"`
	Source string `json:"source,omitempty"`
}

// ThemeBinding is the Ã‚Â§27.6 lock-state scaffold. Lock enforcement deferred to Ã‚Â§27.6 slice.
type ThemeBinding struct {
	AssetID string `json:"asset_id"`
	Slot    string `json:"slot"` // "skin", "background", ...
	Locked  bool   `json:"locked"`

	// Target binding (Â§23.5 ecosphere branding). A binding may address an OWNED ENTITY
	// instead of a UI slot: TargetKind is one of the bondable kinds owned by
	// bonded_branding.go, TargetID is that entity's engine id. Both empty = a legacy
	// UI-slot binding, which keeps every pre-existing binding working unchanged.
	TargetKind string `json:"target_kind,omitempty"`
	TargetID   string `json:"target_id,omitempty"`
}

// BondedMedia is the declared media payload of a bonded asset (Â§23.5).
//
// Honesty contract: the server validates the SHAPE (scheme, mime, hash format, integer
// size/dimensions) and can never fetch the bytes, so HashHex is a DECLARED integrity
// reference â€” any client that retrieves the URI re-hashes the bytes and compares, which is
// what makes a claimed image verifiable rather than trusted. Integer fields only (no float).
type BondedMedia struct {
	URI      string `json:"uri"`       // https:// | ipfs:// | ar:// (content-addressed preferred)
	MimeType string `json:"mime_type"` // image/* | audio/* | video/* allowlist
	HashHex  string `json:"hash_hex"`  // sha256 of the bytes, 64 lowercase hex chars
	Bytes    uint64 `json:"bytes"`     // declared size in bytes (integer)
	Width    uint64 `json:"width"`     // pixels (0 for non-image media)
	Height   uint64 `json:"height"`    // pixels (0 for non-image media)
}

// BondedAssetRegistry owns all bonded assets + theme bindings for a lobby (Ã‚Â§23.5).
type BondedAssetRegistry struct {
	mu       sync.RWMutex
	Assets   map[string]*BondedAsset  `json:"assets"`
	Bindings map[string]*ThemeBinding `json:"bindings"`

	// CardViews is the VIEWER-SCOPED card display layer (card_view_skins.go, Â§10.5).
	//
	// It is keyed by the VIEWER's own wallet and each record names only the viewer's own asset,
	// so there is no field in which another player's card or entity could ever be named: a card
	// can never become a branding target here. The field is EXPORTED and carries a json tag so
	// it survives Save/Load â€” unexported fields silently vanish from the snapshot, which is
	// exactly how the pre-existing assets/bindings persistence bug happened.
	CardViews map[string]*CardView `json:"card_views,omitempty"`

	// SlideThemes is the VIEWER-SCOPED SLIDE THEMING layer (slide_theming.go, Â§10.6) for the app
	// boot / main menu and World Dashboard backgrounds. Same discipline as CardViews â€” keyed by the
	// VIEWER's own wallet, naming only the viewer's own asset, exported + json-tagged so it
	// survives Save/Load. Unexported storage is how the pre-existing assets/bindings bug happened,
	// so a new map is exported the moment it exists.
	SlideThemes map[string]*SlideTheme `json:"slide_themes,omitempty"`

	// Listings is the PLAYER-TO-PLAYER BONDED-ASSET MARKET (bonded_market_service.go). Storage lives
	// here because this struct is the snapshot owner (Save/Load/Burn), while the market file owns
	// every rule about who may list, cancel and buy. Exported + json-tagged for the same reason as
	// CardViews/SlideThemes: unexported storage is how the pre-existing persistence bug happened.
	Listings map[string]*BondedListing `json:"listings,omitempty"`
}

const bondedAssetsFile = "bonded_assets.json"

// bondedAssetsFileMu serialises WRITERS of that one file. Save writes "<file>.tmp" and then renames,
// so two saves running at once could interleave on the same temp path â€” one rename would fail, or a
// half-written temp could be published. Until now the 15-minute persistence worker was the only
// writer, which made that unlikely; the market now writes through after every list/cancel/sale, so it
// is guarded here. The mutex belongs to the FILE, not to one registry instance, because the file is
// the thing two writers would collide on.
var bondedAssetsFileMu sync.Mutex

// NewBondedAssetRegistry returns an empty registry.
func NewBondedAssetRegistry() *BondedAssetRegistry {
	return &BondedAssetRegistry{
		Assets:      make(map[string]*BondedAsset),
		Bindings:    make(map[string]*ThemeBinding),
		CardViews:   make(map[string]*CardView),
		SlideThemes: make(map[string]*SlideTheme),
		Listings:    make(map[string]*BondedListing),
	}
}

// MintBondedAsset creates a new bonded asset. Validates royalty + mood + name.
func (r *BondedAssetRegistry) MintBondedAsset(creator, owner, holder string, atype AssetType, name string, mood, royaltyBps int) (*BondedAsset, error) {
	return r.MintBondedAssetWithMedia(creator, owner, holder, atype, name, mood, royaltyBps, nil)
}

// MintBondedAssetWithMedia is the Â§23.5 branding mint: the caller supplies the media the
// asset claims. The registry does not re-validate media (ValidateBondedMedia in
// bonded_branding.go owns that policy) so the storage layer stays policy-free, but it does
// refuse a card asset type via IsCardAssetType â€” a card can never become a bonded asset.
func (r *BondedAssetRegistry) MintBondedAssetWithMedia(creator, owner, holder string, atype AssetType, name string, mood, royaltyBps int, media *BondedMedia) (*BondedAsset, error) {
	if IsCardAssetType(atype) {
		return nil, fmt.Errorf("asset_type %d is a card type; cards are excluded from bonded assets", int(atype))
	}
	if !IsValidAssetType(atype) {
		return nil, fmt.Errorf("asset_type %d is not a valid bonded asset type (0..%d)", int(atype), int(AssetCustom))
	}
	if name == "" {
		return nil, fmt.Errorf("asset name required")
	}
	if mood < MoodNeutral || mood > MoodMalevolent {
		return nil, fmt.Errorf("invalid mood_tag %d (must be 0..2)", mood)
	}
	if royaltyBps < 0 || royaltyBps > 1000 {
		return nil, fmt.Errorf("royalty_bps %d out of range [0,1000]", royaltyBps)
	}
	if owner == "" || holder == "" {
		return nil, fmt.Errorf("owner and holder wallets required")
	}
	a := &BondedAsset{
		AssetID:       "BA-" + uuid.NewString(),
		AssetType:     atype,
		Name:          name,
		CreatorWallet: creator,
		OwnerWallet:   owner,
		HolderWallet:  holder,
		MoodTag:       mood,
		RoyaltyBps:    royaltyBps,
		Media:         media,
		Certified:     true, // Â§27.7.3: the own-wallet mint path IS the legitimate spawn path
		CreatedAt:     time.Now(),
	}
	r.mu.Lock()
	r.Assets[a.AssetID] = a
	r.mu.Unlock()
	return a, nil
}

// MintCatalogueAsset mints a HOUSE-CATALOGUE asset (a placeholder pack frame) and stamps the SKU it
// came from plus how it was acquired. The stamp is what makes the free starter grant idempotent
// (the assets themselves are the record) and makes a purchase auditable, and it lets a client show
// "from the pack" instead of pretending a house frame is a player creation.
func (r *BondedAssetRegistry) MintCatalogueAsset(owner string, atype AssetType, name, sku, source string, media *BondedMedia) (*BondedAsset, error) {
	a, err := r.MintBondedAssetWithMedia(owner, owner, owner, atype, name, MoodNeutral, 0, media)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	a.Sku = sku
	a.Source = source
	r.mu.Unlock()
	return a, nil
}

// walletMatches reports whether a STORED wallet spelling and a RESOLVED request wallet are the same
// wallet. A resolved request wallet is lowercased on the way in (getWalletFromRequest) while a stored
// spelling is whatever the engine was handed (a mint, a free grant, a purchase, or a transfer to a
// pasted address), so a case-sensitive compare makes a wallet invisible to itself. That is the same
// defect already fixed three times in this subsystem (balance lookup, card-view viewer key,
// wallet-keyed branding targets), so the rule lives here once and every reader uses it.
//
// An empty resolved wallet matches nothing: an anonymous read must never be treated as the owner of
// an asset whose stored wallet is also empty.
func walletMatches(stored, resolved string) bool {
	if strings.TrimSpace(resolved) == "" {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(stored), strings.TrimSpace(resolved))
}

// OwnedSku reports the wallet's asset for one SKU, if it holds one. "Own OR hold" is the chest's
// own rule (an asset is shown to both its owner and its holder), so a granted frame still counts
// while the wallet holds it, and a wallet that gave its frame away can be re-granted on return.
func (r *BondedAssetRegistry) OwnedSku(wallet, sku string) (*BondedAsset, bool) {
	if strings.TrimSpace(wallet) == "" || sku == "" {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, a := range r.Assets {
		if a == nil || a.Sku != sku {
			continue
		}
		if walletMatches(a.OwnerWallet, wallet) || walletMatches(a.HolderWallet, wallet) {
			cp := *a
			return &cp, true
		}
	}
	return nil, false
}

// Get returns a bonded asset by ID.
func (r *BondedAssetRegistry) Get(assetID string) (*BondedAsset, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.Assets[assetID]
	return a, ok
}

// CountOwnedBy reports how many bonded assets a wallet currently OWNS (Â§23.5.1 capacity). A burned
// asset is deleted from the map, so burning frees a capacity slot; a transferred asset moves to the
// recipient, so sending one away frees a slot too.
func (r *BondedAssetRegistry) CountOwnedBy(wallet string) int {
	if strings.TrimSpace(wallet) == "" {
		return 0
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	n := 0
	for _, a := range r.Assets {
		if a != nil && walletMatches(a.OwnerWallet, wallet) {
			n++
		}
	}
	return n
}

// SetBondedAssetMedia replaces the asset's declared media (Â§27.8: owner==holder only).
// Passing nil clears the media. Media policy (scheme/mime/hash/size) belongs to
// ValidateBondedMedia in bonded_branding.go and must be applied by the caller first.
func (r *BondedAssetRegistry) SetBondedAssetMedia(assetID, callerWallet string, media *BondedMedia) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, err := r.guardOwnerHolder(assetID, callerWallet)
	if err != nil {
		return err
	}
	a.Media = media
	return nil
}

// â”€â”€ Â§23.5 ECOSPHERE TARGET BINDINGS â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
// A binding may address an owned ENTITY (item/theme/bot/pet/vehicle/church/club/shop/
// world-content) rather than a UI slot. The registry stores and returns them; the
// AUTHORITY rules (which kinds exist, that no kind is a card, and that the caller owns
// the target) live in bonded_branding.go, which calls these methods only after it has
// proven them. Binding key is "<assetID>:<kind>:<targetID>" so a target may carry more
// than one bonded asset while each (asset,target) pair stays unique.

// BindAssetTarget writes (or refreshes) a target binding. Idempotent: re-binding the same
// asset to the same target updates the existing record instead of duplicating it.
func (r *BondedAssetRegistry) BindAssetTarget(assetID, kind, targetID string) (*ThemeBinding, error) {
	if kind == "" || targetID == "" {
		return nil, fmt.Errorf("target_kind and target_id are required")
	}
	if _, ok := r.Get(assetID); !ok {
		return nil, fmt.Errorf("asset %s not found", assetID)
	}
	b := &ThemeBinding{AssetID: assetID, Slot: kind, TargetKind: kind, TargetID: targetID}
	r.mu.Lock()
	r.Bindings[assetID+":"+kind+":"+targetID] = b
	r.mu.Unlock()
	return b, nil
}

// UnbindAssetTarget removes one target binding. Removing a binding that does not exist is
// not an error (idempotent unbind), but it reports whether anything was removed.
func (r *BondedAssetRegistry) UnbindAssetTarget(assetID, kind, targetID string) (bool, error) {
	if kind == "" || targetID == "" {
		return false, fmt.Errorf("target_kind and target_id are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := assetID + ":" + kind + ":" + targetID
	if _, ok := r.Bindings[key]; !ok {
		return false, nil
	}
	delete(r.Bindings, key)
	return true, nil
}

// sortThemeBindings orders bindings by the thing they address, with the asset id as a tie-break, so
// the order never depends on map iteration: two reads of an unchanged registry list the same
// bindings in the same order. Every caller of the binding readers is a RENDERER (the studio's
// per-entity rows, the public branding read path, the chest's binding list), and a list that
// reshuffles on refresh makes the UI look like it changed when nothing did.
func sortThemeBindings(list []*ThemeBinding) {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0; j-- {
			a, b := list[j], list[j-1]
			less := a.TargetKind < b.TargetKind ||
				(a.TargetKind == b.TargetKind && a.TargetID < b.TargetID) ||
				(a.TargetKind == b.TargetKind && a.TargetID == b.TargetID && a.AssetID < b.AssetID)
			if less {
				list[j], list[j-1] = list[j-1], list[j]
				continue
			}
			break
		}
	}
}

// BindingsForAsset returns every binding that references an asset (copy-safe snapshot) in a
// deterministic order.
func (r *BondedAssetRegistry) BindingsForAsset(assetID string) []*ThemeBinding {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*ThemeBinding, 0)
	for _, b := range r.Bindings {
		if b.AssetID == assetID {
			cp := *b
			out = append(out, &cp)
		}
	}
	sortThemeBindings(out)
	return out
}

// BindingsForTarget returns every binding addressing one entity (copy-safe snapshot) in a
// deterministic order, so a renderer can ask "what branding does this pet/club/church wear?"
// without scanning.
func (r *BondedAssetRegistry) BindingsForTarget(kind, targetID string) []*ThemeBinding {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*ThemeBinding, 0)
	for _, b := range r.Bindings {
		if b.TargetKind == kind && b.TargetID == targetID {
			cp := *b
			out = append(out, &cp)
		}
	}
	sortThemeBindings(out)
	return out
}

// Ã‚Â§27.8 enforcement: an asset may only be modified/transferred/burned/locked by
// the in-game owner/holder, and BOTH must be the same wallet (owner==holder).
// This is the single canonical guard the user's rule requires:
//
//	"assets sold can only be modified by the in game owner/holder (must be both)".
//
// Caller must pass the resolved request wallet. Returns the asset or an error.
func (r *BondedAssetRegistry) guardOwnerHolder(assetID, callerWallet string) (*BondedAsset, error) {
	a, ok := r.Assets[assetID]
	if !ok {
		return nil, fmt.Errorf("asset %s not found", assetID)
	}
	if a.OwnerWallet != callerWallet || a.HolderWallet != callerWallet {
		return nil, fmt.Errorf("asset %s may only be modified by its in-game owner/holder (both must match); caller is not the owner/holder", assetID)
	}
	return a, nil
}

// TransferOwnership is the Ã‚Â§27.8 transfer point. Requires caller==owner==holder
// (the in-game owner/holder, per the user's rule), then moves both ownership and
// holding to newOwner. The auction/transfer flow calls this.
func (r *BondedAssetRegistry) TransferOwnership(assetID, callerWallet, newOwner string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, err := r.guardOwnerHolder(assetID, callerWallet)
	if err != nil {
		return err
	}
	if newOwner == "" {
		return fmt.Errorf("newOwner required")
	}
	a.OwnerWallet = newOwner
	a.HolderWallet = newOwner
	// A LISTING must not outlive the ownership it was written under: the previous owner's active
	// listings for this asset are cancelled here (the market's own sale marks its listing SOLD
	// BEFORE calling this, so a completed sale is never rewritten as a cancellation).
	r.cancelActiveListingsForAssetLocked(assetID, "the asset changed hands")
	return nil
}

// ModifyBondedAsset is the Ã‚Â§27.8 mutation point for in-game cosmetic edits
// (name / mood / royalty), per the user's rule that only the owner/holder (both)
// may modify a sold asset. Only provided fields are changed.
func (r *BondedAssetRegistry) ModifyBondedAsset(assetID, callerWallet string, name *string, mood *int, royaltyBps *int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, err := r.guardOwnerHolder(assetID, callerWallet)
	if err != nil {
		return err
	}
	if name != nil {
		if *name == "" {
			return fmt.Errorf("asset name required")
		}
		a.Name = *name
	}
	if mood != nil {
		if *mood < MoodNeutral || *mood > MoodMalevolent {
			return fmt.Errorf("invalid mood_tag %d (must be 0..2)", *mood)
		}
		a.MoodTag = *mood
	}
	if royaltyBps != nil {
		if *royaltyBps < 0 || *royaltyBps > 1000 {
			return fmt.Errorf("royalty_bps %d out of range [0,1000]", *royaltyBps)
		}
		a.RoyaltyBps = *royaltyBps
	}
	return nil
}

// Burn removes an asset. Requires caller==owner==holder (Ã‚Â§27.8).
func (r *BondedAssetRegistry) Burn(assetID, callerWallet string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, err := r.guardOwnerHolder(assetID, callerWallet); err != nil {
		return err
	}
	delete(r.Assets, assetID)
	// Drop any theme bindings for this asset so MoodTagForWallet stops counting it.
	for k := range r.Bindings {
		if len(k) >= len(assetID) && k[:len(assetID)] == assetID {
			delete(r.Bindings, k)
		}
	}
	// A burned asset must not survive as a viewer's card skin: drop those views (the viewer
	// falls back to the engine's own card art) rather than rendering art that no longer exists.
	for viewer, cv := range r.CardViews {
		if cv != nil && cv.AssetID == assetID {
			delete(r.CardViews, viewer)
		}
	}
	// Same rule for a themed slide: drop the record so the slide falls back to the pack default
	// instead of pointing at art that no longer exists.
	for k, st := range r.SlideThemes {
		if st != nil && st.AssetID == assetID {
			delete(r.SlideThemes, k)
		}
	}
	// And a market listing for art that no longer exists must not remain buyable.
	r.cancelActiveListingsForAssetLocked(assetID, "the asset was burned")
	return nil
}

// BindThemeAsset creates a theme binding (Ã‚Â§27.6 scaffold; Locked=false until locked).
func (r *BondedAssetRegistry) BindThemeAsset(assetID, slot string) (*ThemeBinding, error) {
	if _, ok := r.Get(assetID); !ok {
		return nil, fmt.Errorf("asset %s not found", assetID)
	}
	// ONE OWNER FOR ONE KEY SPACE. A theme slot binding is keyed `<asset>:<slot>` (TWO parts) while
	// an entity target binding is keyed `<asset>:<kind>:<target_id>` (THREE parts) â€” the same map,
	// written by two doors through the same `Slot` field. The shapes only stay disjoint while a slot
	// label cannot contain a colon: a slot spelled `"item:someid"` would address a target binding's
	// key, so a later LockThemeAsset/IsThemeLocked would read and mutate the WRONG record. The rule
	// is enforced here, at the door that owns the slot vocabulary, instead of being left to luck.
	if slot == "" {
		return nil, fmt.Errorf("slot is required")
	}
	if strings.Contains(slot, ":") {
		return nil, fmt.Errorf("a theme slot may not contain ':' â€” that key shape belongs to an entity target binding")
	}
	b := &ThemeBinding{AssetID: assetID, Slot: slot, Locked: false}
	r.mu.Lock()
	r.Bindings[assetID+":"+slot] = b
	r.mu.Unlock()
	return b, nil
}

// LockThemeAsset flips a binding to Locked=true (Ã‚Â§27.6 enforcement state). A locked
// asset is removed from play (cannot contribute its cosmetic MoodTag to the theme)
// unless re-bound/activated through the entity market. Requires owner==holder
// (the in-game owner/holder, per Ã‚Â§27.8 Ã¢â‚¬â€ only they may lock/modify).
func (r *BondedAssetRegistry) LockThemeAsset(assetID, slot, callerWallet string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.Assets[assetID]
	if !ok {
		return fmt.Errorf("asset %s not found", assetID)
	}
	if a.OwnerWallet != callerWallet || a.HolderWallet != callerWallet {
		return fmt.Errorf("theme lock requires owner==holder; caller is not the in-game owner/holder")
	}
	key := assetID + ":" + slot
	b, ok := r.Bindings[key]
	if !ok {
		b = &ThemeBinding{AssetID: assetID, Slot: slot}
	}
	b.Locked = true
	r.Bindings[key] = b
	return nil
}

// IsThemeLocked reports whether a binding is in the Ã‚Â§27.6 locked (out-of-play) state.
func (r *BondedAssetRegistry) IsThemeLocked(assetID, slot string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if b, ok := r.Bindings[assetID+":"+slot]; ok {
		return b.Locked
	}
	return false
}

// MoodTagForWallet aggregates the cosmetic MoodTag of the wallet's non-locked bound
// assets into a single theme MoodTag (Ã‚Â§27.1 cosmetic term). Locked assets are excluded
// (Ã‚Â§27.6: removed from play). Returns (moodTag, true) if at least one bound asset
// contributes; otherwise (0, false) and the theme stays neutral/deferred.
func (r *BondedAssetRegistry) MoodTagForWallet(wallet string) (int, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	sum := 0
	count := 0
	for _, a := range r.Assets {
		if !walletMatches(a.OwnerWallet, wallet) && !walletMatches(a.HolderWallet, wallet) {
			continue
		}
		// Skip if this asset is locked in any slot.
		locked := false
		for k, b := range r.Bindings {
			if len(k) >= len(a.AssetID) && k[:len(a.AssetID)] == a.AssetID && b.Locked {
				locked = true
				break
			}
		}
		if locked {
			continue
		}
		// Map 0/1/2 -> -1/0/+1 so neutrality is the center of mass.
		switch a.MoodTag {
		case MoodBenevolent:
			sum += 1
		case MoodMalevolent:
			sum -= 1
		}
		count++
	}
	if count == 0 {
		return 0, false
	}
	switch {
	case sum > 0:
		return MoodBenevolent, true
	case sum < 0:
		return MoodMalevolent, true
	default:
		return MoodNeutral, true
	}
}

// Save atomically persists the registry (mirrors ItemRegistry.Save item_shop_archetype.go:230).
func (r *BondedAssetRegistry) Save(l *Lobby) error {
	if l == nil {
		return fmt.Errorf("lobby not initialized")
	}
	r.mu.RLock()
	snapshot := &BondedAssetRegistry{
		Assets:      make(map[string]*BondedAsset, len(r.Assets)),
		Bindings:    make(map[string]*ThemeBinding, len(r.Bindings)),
		CardViews:   make(map[string]*CardView, len(r.CardViews)),
		SlideThemes: make(map[string]*SlideTheme, len(r.SlideThemes)),
		Listings:    make(map[string]*BondedListing, len(r.Listings)),
	}
	for k, v := range r.Assets {
		cp := *v
		snapshot.Assets[k] = &cp
	}
	for k, v := range r.Bindings {
		cp := *v
		snapshot.Bindings[k] = &cp
	}
	for k, v := range r.CardViews {
		cp := *v
		snapshot.CardViews[k] = &cp
	}
	for k, v := range r.SlideThemes {
		cp := *v
		snapshot.SlideThemes[k] = &cp
	}
	for k, v := range r.Listings {
		cp := *v
		snapshot.Listings[k] = &cp
	}
	r.mu.RUnlock()

	// One writer at a time for this file (bondedAssetsFileMu). r.mu is already released above, so this
	// can never be taken in the opposite order to it.
	// THE RECORD IS A TRANSPORT MIRROR of the same payload the file write produces: the snapshot above
	// was taken under the REGISTRY OWN lock and released, so the record never sees a live map.
	l.saveBlockchainStateSnapshotLocked(NotePrefixBondedAssetSnapshot, snapshot)

	bondedAssetsFileMu.Lock()
	defer bondedAssetsFileMu.Unlock()
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal bonded assets: %w", err)
	}
	targetPath := l.getDataPath(bondedAssetsFile)
	tempPath := targetPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write bonded assets temp: %w", err)
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to commit bonded assets: %w", err)
	}
	log.Printf("[BondedAsset] Persisted %d bonded assets (%d bindings, %d viewer card views, %d viewer slide themes, %d market listings) to %s",
		len(snapshot.Assets), len(snapshot.Bindings), len(snapshot.CardViews), len(snapshot.SlideThemes), len(snapshot.Listings), targetPath)
	return nil
}

// Load rehydrates the registry from disk (mirrors ItemRegistry.Load item_shop_archetype.go:260).
func (r *BondedAssetRegistry) Load(l *Lobby) error {
	if l == nil {
		return fmt.Errorf("lobby not initialized")
	}
	targetPath := l.getDataPath(bondedAssetsFile)
	data, err := os.ReadFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("[BondedAsset] No existing registry at %s (fresh start)", targetPath)
			return nil
		}
		return fmt.Errorf("failed to read bonded assets: %w", err)
	}
	var snapshot BondedAssetRegistry
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return fmt.Errorf("failed to unmarshal bonded assets: %w", err)
	}
	r.mu.Lock()
	if snapshot.Assets != nil {
		r.Assets = snapshot.Assets
	}
	if snapshot.Bindings != nil {
		r.Bindings = snapshot.Bindings
	}
	if snapshot.CardViews != nil {
		r.CardViews = snapshot.CardViews
	} else {
		r.CardViews = make(map[string]*CardView)
	}
	if snapshot.SlideThemes != nil {
		r.SlideThemes = snapshot.SlideThemes
	} else {
		r.SlideThemes = make(map[string]*SlideTheme)
	}
	if snapshot.Listings != nil {
		r.Listings = snapshot.Listings
	} else {
		r.Listings = make(map[string]*BondedListing)
	}
	r.mu.Unlock()
	log.Printf("[BondedAsset] Loaded %d bonded assets (%d viewer card views, %d viewer slide themes, %d market listings) from %s",
		len(r.Assets), len(r.CardViews), len(r.SlideThemes), len(r.Listings), targetPath)
	return nil
}

// --- HTTP handlers (mirror theme_engine.go handler idiom) ---

func (l *Lobby) handleMintBondedAsset(w http.ResponseWriter, r *http.Request) {
	if l.bondedAssets == nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "registry not initialized"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	// Â§23.5.1 capacity: creation is capped at the wallet's card/deck NFT supply, so the gate runs
	// BEFORE the body is even read and before anything can be written.
	if err := l.assertBondedAssetCapacity(wallet); err != nil {
		writeJSON(w, map[string]interface{}{
			"success":  false,
			"error":    err.Error(),
			"capacity": l.BondedAssetCapacityForWallet(wallet),
		})
		return
	}
	var req struct {
		AssetType  int          `json:"asset_type"`
		Name       string       `json:"name"`
		MoodTag    int          `json:"mood_tag"`
		RoyaltyBps int          `json:"royalty_bps"`
		Media      *BondedMedia `json:"media"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}
	// Â§23.5 media policy is enforced at the boundary: a malformed or unsafe media
	// declaration is refused outright rather than stored and trusted.
	if err := ValidateBondedMedia(req.Media); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	a, err := l.bondedAssets.MintBondedAssetWithMedia(wallet, wallet, wallet, AssetType(req.AssetType), req.Name, req.MoodTag, req.RoyaltyBps, req.Media)
	if err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{
		"success":  true,
		"asset_id": a.AssetID,
		"media":    a.Media,
		// The new budget, so a client never has to guess how many are left (Â§23.5.1).
		"capacity": l.BondedAssetCapacityForWallet(wallet),
	})
}

// sortBondedAssets orders a chest newest first with the asset id as a tie-break â€” the same rule the
// market listings use. `Assets` is a map, so without this the chest's order (and therefore the
// studio's asset rows and the market's "asset to sell" dropdown) reshuffled on every refresh even
// though nothing had changed.
func sortBondedAssets(list []*BondedAsset) {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0; j-- {
			older := list[j].CreatedAt.Before(list[j-1].CreatedAt)
			sameMoment := list[j].CreatedAt.Equal(list[j-1].CreatedAt) && list[j].AssetID < list[j-1].AssetID
			if older || sameMoment {
				list[j], list[j-1] = list[j-1], list[j]
				continue
			}
			break
		}
	}
}

func (l *Lobby) handleListBondedAssets(w http.ResponseWriter, r *http.Request) {
	if l.bondedAssets == nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "registry not initialized"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	l.bondedAssets.mu.RLock()
	out := make([]*BondedAsset, 0, len(l.bondedAssets.Assets))
	bindings := make([]*ThemeBinding, 0)
	for _, a := range l.bondedAssets.Assets {
		if walletMatches(a.OwnerWallet, wallet) || walletMatches(a.HolderWallet, wallet) {
			out = append(out, a)
			for _, b := range l.bondedAssets.Bindings {
				if b.AssetID == a.AssetID {
					bindings = append(bindings, b)
				}
			}
		}
	}
	l.bondedAssets.mu.RUnlock()
	// Deterministic order: the chest and its bindings read identically twice in a row, so a refresh
	// never looks like a change (Â§ determinism; mirrors sortBondedListings on the market).
	sortBondedAssets(out)
	sortThemeBindings(bindings)
	writeJSON(w, map[string]interface{}{
		"success":  true,
		"wallet":   wallet,
		"assets":   out,
		"bindings": bindings,
		// Â§23.5.1 the chest's creation budget: capacity equals the wallet's card/deck NFT supply.
		"capacity": l.BondedAssetCapacityForWallet(wallet),
		// Served policy: the client never re-declares which kinds are bondable or what
		// media is acceptable (Â§23.5). `blocked_target_kinds` is the card exclusion rule.
		"policy": BondedBrandingPolicy(),
	})
}

func (l *Lobby) handleBurnBondedAsset(w http.ResponseWriter, r *http.Request) {
	if l.bondedAssets == nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "registry not initialized"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	var req struct {
		AssetID string `json:"asset_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}
	a, ok := l.bondedAssets.Get(req.AssetID)
	if !ok {
		writeJSON(w, map[string]interface{}{"success": false, "error": "asset not found"})
		return
	}
	if a.OwnerWallet != wallet || a.HolderWallet != wallet {
		writeJSON(w, map[string]interface{}{"success": false, "error": "only the owner/holder may burn this asset"})
		return
	}
	if err := l.bondedAssets.Burn(req.AssetID, wallet); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true})
}

func (l *Lobby) handleBindThemeAsset(w http.ResponseWriter, r *http.Request) {
	if l.bondedAssets == nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "registry not initialized"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	var req struct {
		AssetID string `json:"asset_id"`
		Slot    string `json:"slot"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}
	b, err := l.bondedAssets.BindThemeAsset(req.AssetID, req.Slot)
	if err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{
		"success":  true,
		"asset_id": b.AssetID,
		"slot":     b.Slot,
		"locked":   b.Locked,
	})
}

func (l *Lobby) handleLockThemeAsset(w http.ResponseWriter, r *http.Request) {
	if l.bondedAssets == nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "registry not initialized"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	var req struct {
		AssetID string `json:"asset_id"`
		Slot    string `json:"slot"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}
	if err := l.bondedAssets.LockThemeAsset(req.AssetID, req.Slot, wallet); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{
		"success": true,
		"locked":  l.bondedAssets.IsThemeLocked(req.AssetID, req.Slot),
	})
}

// handleModifyBondedAsset is the Ã‚Â§27.8 HTTP entry point for in-game owner/holder
// cosmetic edits (name/mood/royalty). Enforces caller==owner==holder via
// BondedAssetRegistry.ModifyBondedAsset (the single canonical guard).
func (l *Lobby) handleModifyBondedAsset(w http.ResponseWriter, r *http.Request) {
	if l.bondedAssets == nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "registry not initialized"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	var req struct {
		AssetID    string  `json:"asset_id"`
		Name       *string `json:"name"`
		MoodTag    *int    `json:"mood_tag"`
		RoyaltyBps *int    `json:"royalty_bps"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}
	if err := l.bondedAssets.ModifyBondedAsset(req.AssetID, wallet, req.Name, req.MoodTag, req.RoyaltyBps); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true})
}

// handleTransferBondedAsset is the Ã‚Â§27.8 HTTP entry point for transferring a
// bonded asset. Enforces caller==owner==holder via BondedAssetRegistry.
// TransferOwnership (the single canonical guard) before moving ownership+holding.
func (l *Lobby) handleTransferBondedAsset(w http.ResponseWriter, r *http.Request) {
	if l.bondedAssets == nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "registry not initialized"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	var req struct {
		AssetID  string `json:"asset_id"`
		NewOwner string `json:"new_owner"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}
	if err := l.bondedAssets.TransferOwnership(req.AssetID, wallet, req.NewOwner); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true})
}
