//go:build !js && !wasm

package main

// bonded_branding.go
// §23.5 — BONDED ASSETS ARE THE ECOSYSTEM-WIDE BRANDING LAYER.
//
// A bonded asset is a player-owned, player-named NFT carrying the player's OWN media. It may
// be worn by any entity the player owns across the civilization: an item, a theme, a bot (AI
// citizen / child bot), a pet, a vehicle, a church, a club, a shop, or deployed world content
// — i.e. anything a user can tie an image to, for branding or theming.
//
// THE CARD EXCLUSION IS BINDING. Deck cards, hand cards, religious-leader cards and ANY OTHER
// CARD TYPE can never be bonded-asset branded. This is enforced structurally:
//   * IsCardTargetKind refuses every target kind that names a card (and every explicit card
//     alias) BEFORE any other check, so a card request fails closed;
//   * the bondable-kind allowlist may not contain a card kind — assertBondedBrandingPolicy
//     runs at init() and panics at startup if one is ever added;
//   * IsCardAssetType refuses a card-typed bonded asset at mint, so a card cannot even become
//     the branding asset.
//
// Ownership discipline: a player may only brand an entity they own, and only with an asset
// they both own AND hold (§27.8) that is legitimate (§27.7.3 — a black-market-adopted asset
// may not masquerade as legitimate branding). Every refusal happens before state is written,
// so a refused brand changes nothing.
//
// Integer discipline: media carries no float — bytes and pixel dimensions are uint64 and the
// hash is an opaque hex string (Architecture Ledger).

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// ── Bondable target kinds ─────────────────────────────────────────────────────
const (
	TargetItem         = "item"          // §14 built item / equipment (creator-owned)
	TargetTheme        = "theme"         // §27 wallet theme vector / UI skin set
	TargetBot          = "bot"           // §15/§24 AI citizen or child bot (owner = OriginWallet)
	TargetPet          = "pet"           // §26.4 companion
	TargetVehicle      = "vehicle"       // §25.6 vehicle
	TargetChurch       = "church"        // §32 faith church
	TargetClub         = "club"          // club / organization
	TargetShop         = "shop"          // a club's shopfront (target id is the club id)
	TargetWorldContent = "world_content" // §25.7 deployed native world entity
)

// bondedTargetKind is one served (kind, label) pair of the brandable catalogue.
type bondedTargetKind struct {
	Kind  string `json:"kind"`
	Label string `json:"label"`
}

// bondableTargetKindDescriptions is the SERVED catalogue: the UI renders exactly this, so the
// client never re-declares which entities are brandable.
var bondableTargetKindDescriptions = []bondedTargetKind{
	{TargetItem, "Items & Equipment"},
	{TargetTheme, "Themes & Skins"},
	{TargetBot, "AI Citizens & Child Bots"},
	{TargetPet, "Companions"},
	{TargetVehicle, "Vehicles"},
	{TargetChurch, "Churches"},
	{TargetClub, "Clubs"},
	{TargetShop, "Club Shops"},
	{TargetWorldContent, "World Content"},
}

// cardTargetKinds are the CARD aliases that do not literally contain the substring "card".
// Anything else card-shaped is caught by the substring rule in IsCardTargetKind, which is what
// makes the exclusion cover "any other cards" structurally rather than by enumeration.
var cardTargetKinds = []string{
	"card", "cards", "deck", "decks", "deck_card", "hand", "hands", "hand_card",
	"leader_card", "religious_leader_card", "card_deck", "card_hand",
}

// IsCardTargetKind reports whether a target kind names a card. The scan is deliberately broad
// (any kind containing "card", plus the deck/hand aliases) so a future card surface is refused
// by default instead of leaking in as a bondable target.
func IsCardTargetKind(kind string) bool {
	k := normalizeTargetKind(kind)
	if k == "" {
		return false
	}
	if strings.Contains(k, "card") {
		return true
	}
	for _, alias := range cardTargetKinds {
		if k == alias {
			return true
		}
	}
	return false
}

// IsBindableTargetKind reports whether a kind is on the allowlist. Fail-closed: unknown kinds
// are refused, and a card kind can never be on the list (asserted at init).
func IsBindableTargetKind(kind string) bool {
	k := normalizeTargetKind(kind)
	for _, d := range bondableTargetKindDescriptions {
		if d.Kind == k {
			return true
		}
	}
	return false
}

// normalizeTargetKind canonicalises a client-supplied kind: trimmed, lowercase, with '-'/' '
// folded to '_' so "World-Content" and "world content" are one kind.
func normalizeTargetKind(kind string) string {
	k := strings.ToLower(strings.TrimSpace(kind))
	k = strings.ReplaceAll(k, "-", "_")
	k = strings.ReplaceAll(k, " ", "_")
	return k
}

// ── Card exclusion for the ASSET side ─────────────────────────────────────────
// Card asset types live at/above AssetCard. They can never be minted as a bonded asset.
const (
	AssetCard AssetType = 100 // first card asset-type id: cards are excluded from bonding
)

// IsCardAssetType reports whether an asset type names a card.
func IsCardAssetType(t AssetType) bool { return t >= AssetCard }

// IsValidAssetType reports whether an asset type is a real, bondable cosmetic type.
// Out-of-range types are refused rather than stored as nonsense (the enum is the policy).
func IsValidAssetType(t AssetType) bool {
	return t >= AssetSkin && t <= AssetCustom
}

// assertBondedBrandingPolicy is the startup guard for the cardinal rule. It panics the server
// at boot if any bondable kind is card-shaped or if the card alias list is empty — i.e. if a
// future edit ever tries to make cards brandable, the build cannot run silently.
func assertBondedBrandingPolicy() {
	if len(bondableTargetKindDescriptions) == 0 {
		panic("bonded branding: bondable target kind list is empty")
	}
	if len(cardTargetKinds) == 0 {
		panic("bonded branding: card exclusion list is empty")
	}
	for _, d := range bondableTargetKindDescriptions {
		if IsCardTargetKind(d.Kind) {
			panic(fmt.Sprintf("bonded branding: target kind %q is card-shaped but listed as bondable; cards are excluded", d.Kind))
		}
		if d.Kind != normalizeTargetKind(d.Kind) {
			panic(fmt.Sprintf("bonded branding: target kind %q is not canonical; use %q", d.Kind, normalizeTargetKind(d.Kind)))
		}
	}
	for _, card := range cardTargetKinds {
		if IsBindableTargetKind(card) {
			panic(fmt.Sprintf("bonded branding: card alias %q appears in the bondable list", card))
		}
	}
}

func init() { assertBondedBrandingPolicy() }

// ── Media policy (§23.5) ──────────────────────────────────────────────────────
const (
	// BondedMediaMaxBytes caps a declared media payload (8 MiB).
	BondedMediaMaxBytes uint64 = 8 << 20
	// BondedMediaMaxDimension caps a declared pixel dimension per side.
	BondedMediaMaxDimension uint64 = 8192
	// BondedMediaUriMaxLen caps the declared URI length.
	BondedMediaUriMaxLen = 1024
)

// BondedMediaLimits is the SERVED media policy (no client-side re-declaration).
type BondedMediaLimits struct {
	MaxBytes           uint64   `json:"max_bytes"`
	MaxDimension       uint64   `json:"max_dimension"`
	MaxUriLength       int      `json:"max_uri_length"`
	AllowedSchemes     []string `json:"allowed_schemes"`
	AllowedLocalPrefix string   `json:"allowed_local_prefix"`
	AllowedMimeTypes   []string `json:"allowed_mime_types"`
	RequiresSha256Hex  bool     `json:"requires_sha256_hex"`
}

// bondedMediaAllowedSchemes are the only URI schemes accepted. `data:` is refused on purpose
// (an unbounded inline blob would bloat every payload carrying the asset) and
// `javascript:`/`file:`/`blob:` are refused as unsafe.
//
// BondedMediaLocalPrefix is the ONE accepted scheme-less form: a same-origin path under the art
// this process serves itself. It exists because the local NPC-helper placeholder pack (and the
// catalogue frames minted from it, placeholder_assets.go) are our own art, not remote media — the
// client has always painted `/Assets/…` (card_view_skins.js), so the policy had to admit the form
// it was already rendering rather than force the art behind an invented gateway.
const BondedMediaLocalPrefix = "/Assets/"

var bondedMediaAllowedSchemes = []string{"https", "http", "ipfs", "ar"}

// bondedMediaAllowedMimes are the accepted media types.
var bondedMediaAllowedMimes = []string{
	"image/png", "image/jpeg", "image/webp", "image/gif", "image/avif", "image/svg+xml",
	"audio/mpeg", "audio/ogg", "audio/wav",
	"video/mp4", "video/webm",
}

// BondedMediaLimitsForClient returns the served limits.
func BondedMediaLimitsForClient() BondedMediaLimits {
	return BondedMediaLimits{
		MaxBytes:           BondedMediaMaxBytes,
		MaxDimension:       BondedMediaMaxDimension,
		MaxUriLength:       BondedMediaUriMaxLen,
		AllowedSchemes:     append([]string{}, bondedMediaAllowedSchemes...),
		AllowedLocalPrefix: BondedMediaLocalPrefix,
		AllowedMimeTypes:   append([]string{}, bondedMediaAllowedMimes...),
		RequiresSha256Hex:  true,
	}
}

// isSameOriginAssetsPath reports whether a scheme-less URI is a path to art THIS process serves.
// It is deliberately narrow: the art prefix, no traversal, no protocol-relative "//", and only
// unreserved URL characters (so a path that needs escaping — a space, a parenthesis, a query, a
// fragment — is refused rather than silently mis-resolved). The shipped pack was renamed to
// "<Prefix>-NNN.png" precisely so every frame satisfies this.
func isSameOriginAssetsPath(u string) bool {
	if !strings.HasPrefix(u, BondedMediaLocalPrefix) {
		return false
	}
	if strings.Contains(u, "..") || strings.Contains(u, "//") {
		return false
	}
	for _, c := range u {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case c == '/' || c == '-' || c == '_' || c == '.' || c == '+' || c == '~':
		default:
			return false
		}
	}
	return true
}

// isLowerHex reports whether s is exactly n lowercase hex characters.
func isLowerHex(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// ValidateBondedMedia enforces the media policy. A nil media is valid (an asset may be minted
// unpainted and given art later). It CANONICALISES the hash to lowercase and trims URI/mime
// whitespace, so two spellings of one declaration cannot produce two stored forms. The bytes
// are never fetched: HashHex is the declared integrity reference a client re-checks after
// retrieval — the server validates shape, it does not claim to have verified the bytes.
func ValidateBondedMedia(media *BondedMedia) error {
	if media == nil {
		return nil
	}
	media.URI = strings.TrimSpace(media.URI)
	media.MimeType = strings.ToLower(strings.TrimSpace(media.MimeType))
	media.HashHex = strings.ToLower(strings.TrimSpace(media.HashHex))

	if media.URI == "" {
		return fmt.Errorf("media.uri is required")
	}
	if len(media.URI) > BondedMediaUriMaxLen {
		return fmt.Errorf("media.uri exceeds %d characters", BondedMediaUriMaxLen)
	}
	u, err := url.Parse(media.URI)
	if err != nil {
		return fmt.Errorf("media.uri is not a valid URI")
	}
	scheme := strings.ToLower(u.Scheme)
	allowedScheme := false
	sameOrigin := false
	// The ONE scheme-less form: art THIS process serves (the NPC-helper pack). It is judged by its
	// own narrow rule BEFORE the scheme list, so a same-origin path cannot be smuggled in as a
	// scheme and a remote URL cannot be smuggled in as a local path.
	if scheme == "" && strings.HasPrefix(media.URI, "/") {
		if !isSameOriginAssetsPath(media.URI) {
			return fmt.Errorf("media.uri %q is not a usable same-origin path (it must start with %s, carry no '..' or '//', and use unreserved URL characters only)",
				media.URI, BondedMediaLocalPrefix)
		}
		allowedScheme, sameOrigin = true, true
	}
	if !allowedScheme {
		for _, s := range bondedMediaAllowedSchemes {
			if scheme == s {
				allowedScheme = true
				break
			}
		}
	}
	if !allowedScheme {
		return fmt.Errorf("media.uri scheme %q is not allowed (%s, or a same-origin %s path)",
			u.Scheme, strings.Join(bondedMediaAllowedSchemes, ", "), BondedMediaLocalPrefix)
	}
	if u.User != nil {
		return fmt.Errorf("media.uri must not carry credentials")
	}
	if !sameOrigin && (scheme == "https" || scheme == "http") {
		if u.Host == "" {
			return fmt.Errorf("media.uri must include a host")
		}
	}

	if media.MimeType == "" {
		return fmt.Errorf("media.mime_type is required")
	}
	allowedMime := false
	for _, m := range bondedMediaAllowedMimes {
		if media.MimeType == m {
			allowedMime = true
			break
		}
	}
	if !allowedMime {
		return fmt.Errorf("media.mime_type %q is not allowed", media.MimeType)
	}
	if !isLowerHex(media.HashHex, 64) {
		return fmt.Errorf("media.hash_hex must be the sha256 of the bytes (64 lowercase hex characters)")
	}
	if media.Bytes == 0 {
		return fmt.Errorf("media.bytes must be greater than zero")
	}
	if media.Bytes > BondedMediaMaxBytes {
		return fmt.Errorf("media.bytes %d exceeds the %d byte limit", media.Bytes, BondedMediaMaxBytes)
	}
	if media.Width > BondedMediaMaxDimension || media.Height > BondedMediaMaxDimension {
		return fmt.Errorf("media dimensions exceed the %d pixel per-side limit", BondedMediaMaxDimension)
	}
	if strings.HasPrefix(media.MimeType, "image/") && (media.Width == 0 || media.Height == 0) {
		return fmt.Errorf("image media must declare width and height")
	}
	return nil
}

// ── Target ownership resolution ───────────────────────────────────────────────
// Each case takes exactly ONE engine lock and returns; locks are never nested, so branding
// can never deadlock against a service that holds its own lock while calling into the Lobby.
// A target that does not exist resolves to "" and the caller reports it as a refusal — never
// as ownership.

// canonicalTargetID normalises a target id for the kinds whose id IS a wallet: wallets are
// lowercase-canonical throughout this engine (getWalletFromRequest lowercases the caller), so
// "0xABC" and "0xabc" must resolve to ONE target rather than two — otherwise a client whose
// casing differs is wrongly told it does not own its own theme/bot.
func canonicalTargetID(kind, targetID string) string {
	id := strings.TrimSpace(targetID)
	switch kind {
	case TargetTheme, TargetBot:
		return strings.ToLower(id)
	}
	return id
}

// bondedTargetOwner resolves the wallet owning a brandable entity ("" when it does not exist).
func (l *Lobby) bondedTargetOwner(kind, targetID string) string {
	targetID = canonicalTargetID(kind, targetID)
	if targetID == "" {
		return ""
	}
	switch kind {
	case TargetTheme:
		// A theme IS the wallet's own §27 theme vector, so the wallet is its owner.
		return targetID
	case TargetPet:
		l.mutex.RLock()
		defer l.mutex.RUnlock()
		if p, ok := l.pets[targetID]; ok {
			return p.Owner
		}
	case TargetVehicle:
		l.mutex.RLock()
		defer l.mutex.RUnlock()
		if v, ok := l.vehicles[targetID]; ok {
			return v.Owner
		}
	case TargetWorldContent:
		l.mutex.RLock()
		defer l.mutex.RUnlock()
		if c, ok := l.worldContent[targetID]; ok {
			return c.Creator
		}
	case TargetClub, TargetShop:
		l.mutex.RLock()
		defer l.mutex.RUnlock()
		if c, ok := l.clubs[targetID]; ok {
			return c.OwnerWallet
		}
	case TargetItem:
		if l.itemRegistry == nil {
			return ""
		}
		for _, it := range l.itemRegistry.GetRegistry() {
			if it.ItemID == targetID {
				return it.CreatorWallet
			}
		}
	case TargetBot:
		if l.aiEngine == nil {
			return ""
		}
		l.aiEngine.mu.RLock()
		defer l.aiEngine.mu.RUnlock()
		if c, ok := l.aiEngine.citizens[targetID]; ok {
			if c.OriginWallet != "" {
				return c.OriginWallet
			}
			return c.OwnerWallet
		}
	case TargetChurch:
		faithChurchEngine.mu.RLock()
		defer faithChurchEngine.mu.RUnlock()
		if c, ok := faithChurchEngine.churches[targetID]; ok {
			return c.Owner
		}
	}
	return ""
}

// ── Branding views (what the API serves) ──────────────────────────────────────

// BondedBrandingView is one branded asset worn by a target, joined with the asset's media.
type BondedBrandingView struct {
	AssetID   string       `json:"asset_id"`
	AssetName string       `json:"asset_name"`
	Slot      string       `json:"slot,omitempty"`
	Locked    bool         `json:"locked"`
	Media     *BondedMedia `json:"media,omitempty"`
}

// BondedTargetView is one entity the wallet may brand, with the branding it already wears.
type BondedTargetView struct {
	Kind      string               `json:"kind"`
	TargetID  string               `json:"target_id"`
	Name      string               `json:"name"`
	KindLabel string               `json:"kind_label"`
	Branding  []BondedBrandingView `json:"branding"`
}

// kindLabel returns the served human label for a kind.
func kindLabel(kind string) string {
	for _, d := range bondableTargetKindDescriptions {
		if d.Kind == kind {
			return d.Label
		}
	}
	return kind
}

// ── Lobby binding authority ───────────────────────────────────────────────────

// BindBondedAssetToTarget is the single authority for branding an entity. It fails closed in
// this order: cards (never), unknown kinds (never), a missing/foreign/off-ledger asset (never),
// a target the caller does not own (never) — and only then writes the binding.
func (l *Lobby) BindBondedAssetToTarget(wallet, assetID, kind, targetID string) (*ThemeBinding, error) {
	// The card exclusion is checked FIRST, before any lookup, so no card request can touch
	// state even if a future edit muddles the allowlist.
	if IsCardTargetKind(kind) {
		return nil, fmt.Errorf("cards are excluded from bonded assets (%q refused)", kind)
	}
	k := normalizeTargetKind(kind)
	if !IsBindableTargetKind(k) {
		return nil, fmt.Errorf("target_kind %q is not brandable", kind)
	}
	if wallet == "" {
		return nil, fmt.Errorf("wallet required")
	}
	if assetID == "" {
		return nil, fmt.Errorf("asset_id required")
	}
	if l.bondedAssets == nil {
		return nil, fmt.Errorf("registry not initialized")
	}
	a, ok := l.bondedAssets.Get(assetID)
	if !ok {
		return nil, fmt.Errorf("asset %s not found", assetID)
	}
	// §27.8: only the in-game owner/holder (both) may bind or re-bind an asset.
	if a.OwnerWallet != wallet || a.HolderWallet != wallet {
		return nil, fmt.Errorf("asset %s may only be bound by its in-game owner/holder (both must match)", assetID)
	}
	// §27.7.3 legitimacy: off-ledger (black-market-adopted) art may not brand a legitimate
	// entity. Certified is stamped by the legitimate mint path; an asset minted before the
	// flag existed is not retro-refused, but an adopted one never brands.
	if a.BlackMarketAdopted {
		return nil, fmt.Errorf("asset %s is black-market-adopted and may not brand an entity (§27.7.3)", assetID)
	}
	targetID = canonicalTargetID(k, targetID)
	owner := l.bondedTargetOwner(k, targetID)
	if owner == "" {
		return nil, fmt.Errorf("target %s/%s not found", k, targetID)
	}
	if owner != wallet {
		return nil, fmt.Errorf("you do not own that %s", k)
	}
	return l.bondedAssets.BindAssetTarget(assetID, k, targetID)
}

// UnbindBondedAssetTarget removes one of the caller's own bindings (§27.8 owner/holder). It is
// idempotent: removing a binding that is not there reports removed=false, not an error.
func (l *Lobby) UnbindBondedAssetTarget(wallet, assetID, kind, targetID string) (bool, error) {
	if IsCardTargetKind(kind) {
		return false, fmt.Errorf("cards are excluded from bonded assets (%q refused)", kind)
	}
	k := normalizeTargetKind(kind)
	if !IsBindableTargetKind(k) {
		return false, fmt.Errorf("target_kind %q is not brandable", kind)
	}
	if wallet == "" {
		return false, fmt.Errorf("wallet required")
	}
	if l.bondedAssets == nil {
		return false, fmt.Errorf("registry not initialized")
	}
	a, ok := l.bondedAssets.Get(assetID)
	if !ok {
		return false, fmt.Errorf("asset %s not found", assetID)
	}
	if a.OwnerWallet != wallet || a.HolderWallet != wallet {
		return false, fmt.Errorf("asset %s may only be unbound by its in-game owner/holder (both must match)", assetID)
	}
	return l.bondedAssets.UnbindAssetTarget(assetID, k, canonicalTargetID(k, targetID))
}

// BondedBrandingForTarget is the READ path every renderer uses: "what art does this entity
// wear?". It needs no wallet because branding is public, cosmetic data (like an NFT image). A
// card target returns nothing — a card can never carry bonded branding.
func (l *Lobby) BondedBrandingForTarget(kind, targetID string) []BondedBrandingView {
	out := make([]BondedBrandingView, 0)
	if IsCardTargetKind(kind) || l.bondedAssets == nil {
		return out
	}
	k := normalizeTargetKind(kind)
	if !IsBindableTargetKind(k) || targetID == "" {
		return out
	}
	targetID = canonicalTargetID(k, targetID)
	for _, b := range l.bondedAssets.BindingsForTarget(k, targetID) {
		v := BondedBrandingView{AssetID: b.AssetID, Slot: b.Slot, Locked: b.Locked}
		if a, ok := l.bondedAssets.Get(b.AssetID); ok {
			v.AssetName = a.Name
			v.Media = a.Media
		}
		out = append(out, v)
	}
	return out
}

// BondedAssetTypes is the SERVED asset-type catalogue, so the client renders the real enum
// instead of re-declaring it.
var bondedAssetTypeCatalogue = []bondedTargetKind{
	{Kind: "0", Label: "Skin"},
	{Kind: "1", Label: "Background"},
	{Kind: "2", Label: "Board"},
	{Kind: "3", Label: "Button"},
	{Kind: "4", Label: "Appearance"},
	{Kind: "5", Label: "Audio"},
	{Kind: "6", Label: "Custom"},
}

// BondedBrandingPolicy is the SERVED rule set: which kinds may be branded, which are excluded,
// and what media is acceptable. The client renders this instead of hard-coding policy.
func BondedBrandingPolicy() map[string]interface{} {
	return map[string]interface{}{
		"bindable_target_kinds": bondableTargetKindDescriptions,
		"blocked_target_kinds":  append([]string{}, cardTargetKinds...),
		"asset_types":           bondedAssetTypeCatalogue,
		"cards_excluded":        true,
		"rule": "Bonded assets brand every entity you own — items, themes, bots, pets, vehicles, " +
			"churches, clubs, shops, world content. Cards (deck cards, hand cards, " +
			"religious-leader cards and any other card type) are excluded.",
		"legitimacy": "Only your own in-game owner/holder assets may brand an entity; " +
			"black-market-adopted assets are refused (§27.7.3).",
		"media": BondedMediaLimitsForClient(),
		"capacity_rule": "A wallet may create as many bonded assets as the card/deck NFTs it holds: " +
			"capacity equals your card/deck NFT supply.",
		"creation_requires": "a bonded asset must be created (or acquired) before it can be worn " +
			"anywhere - including the viewer-scoped card display, which requires one of your own assets",
	}
}

// -- §23.5.1 BONDED-ASSET CAPACITY: one bonded asset per card/deck NFT ------------------
//
// "a player can create many bonded assets equal to there NFT supply - cards/deck."
//
// The basis is the wallet's card/deck NFT supply AS THE ENGINE RECORDS IT: the `CARD-<id>` keys of
// PlayerStats.Inventory. That is the same authoritative bookkeeping the auction, loan, black-market
// and jail escrow paths maintain through TransferBundleItems, so the number is engine-owned rather
// than client-declared (a client can never raise its own cap).
//
// When wallet-NFT sourcing lands (verified on-chain holdings), CardNFTSupplyForWallet becomes the
// one function that changes: no caller, no route, no policy text and no client moves.
//
// The cap gates CREATION only. It never retro-invalidates an asset that already exists, and it never
// blocks a transfer, so a wallet that receives assets may legitimately sit above its capacity until
// it burns or sends one away.

// BondedAssetCapacityBasisEngineCards names, in the served payload, WHERE the limit came from -
// so a player is never told "no" without being told why.
const BondedAssetCapacityBasisEngineCards = "card/deck NFTs held by this wallet (engine record: PlayerStats.Inventory CARD-*)"

// BondedAssetCapacity is the served creation budget for one wallet.
type BondedAssetCapacity struct {
	Basis     string `json:"basis"`
	Limit     int    `json:"limit"`     // card/deck NFT supply (integer)
	Used      int    `json:"used"`      // bonded assets currently owned
	Remaining int    `json:"remaining"` // max(0, limit-used)
	Rule      string `json:"rule"`
}

// CardNFTSupplyForWallet counts the card/deck NFTs the engine records for a wallet: every
// PlayerStats.Inventory key prefixed `CARD-`, with its quantity summed (two copies of a card are
// two NFTs of supply). Quantities <= 0 are ignored, and a wallet the engine has no record for has
// a supply of 0 - which is a real answer, not an error.
func (l *Lobby) CardNFTSupplyForWallet(wallet string) int {
	raw := strings.TrimSpace(wallet)
	if raw == "" {
		return 0
	}
	l.mutex.RLock()
	stats, ok := l.leaderboard[strings.ToLower(raw)]
	if !ok {
		// Wallets are lowercase-canonical, but a record stored under another spelling is the SAME
		// wallet: match it case-insensitively rather than telling a player they own no cards.
		for k, v := range l.leaderboard {
			if strings.EqualFold(k, raw) {
				stats, ok = v, true
				break
			}
		}
	}
	total := 0
	if ok && stats.Inventory != nil {
		for itemID, qty := range stats.Inventory {
			if qty > 0 && strings.HasPrefix(itemID, "CARD-") {
				total += qty
			}
		}
	}
	l.mutex.RUnlock()
	return total
}

// BondedAssetCapacityForWallet reports the wallet's creation budget. Exactly one engine lock is
// taken and released for the supply, then exactly one registry lock for the used count - never
// nested, so capacity can never deadlock against either side.
func (l *Lobby) BondedAssetCapacityForWallet(wallet string) BondedAssetCapacity {
	w := strings.ToLower(strings.TrimSpace(wallet))
	cap := BondedAssetCapacity{
		Basis: BondedAssetCapacityBasisEngineCards,
		Limit: l.CardNFTSupplyForWallet(w),
		Rule:  "Capacity equals your card/deck NFT supply: one bonded asset per card you hold.",
	}
	if l.bondedAssets != nil && w != "" {
		cap.Used = l.bondedAssets.CountOwnedBy(w)
	}
	cap.Remaining = cap.Limit - cap.Used
	if cap.Remaining < 0 {
		// A gifted asset can put a wallet above its own capacity; report the truth and clamp only
		// the remaining budget, never the real ownership count.
		cap.Remaining = 0
	}
	return cap
}

// assertBondedAssetCapacity is the mint gate (§23.5.1). It refuses BEFORE anything is written, and
// its message states the rule and the numbers so the player knows what to do about it.
func (l *Lobby) assertBondedAssetCapacity(wallet string) error {
	c := l.BondedAssetCapacityForWallet(wallet)
	if c.Limit <= 0 {
		return fmt.Errorf("bonded assets are capped at your card/deck NFT supply and the engine records "+
			"none for this wallet (%d/%d) - acquire cards or decks first", c.Used, c.Limit)
	}
	if c.Used >= c.Limit {
		return fmt.Errorf("bonded-asset capacity reached (%d/%d: capacity equals your card/deck NFT supply)",
			c.Used, c.Limit)
	}
	return nil
}

// ── Target enumeration (what the wallet may brand) ────────────────────────────

// bondedTargetsOfKind collects the wallet's owned entities of ONE kind. Each case takes exactly
// one engine lock, copies out (id, name) pairs, and releases it BEFORE any view is built — so no
// two locks are ever held at once and the registry is never read under an engine lock.
func (l *Lobby) bondedTargetsOfKind(wallet, kind string) []BondedTargetView {
	type pair struct{ id, name string }
	pairs := make([]pair, 0)

	switch kind {
	case TargetTheme:
		// The wallet's own §27 theme is always brandable by that wallet.
		pairs = append(pairs, pair{wallet, "Your theme"})
	case TargetPet:
		l.mutex.RLock()
		for _, p := range l.pets {
			if p.Owner == wallet {
				pairs = append(pairs, pair{p.PetID, p.Name})
			}
		}
		l.mutex.RUnlock()
	case TargetVehicle:
		l.mutex.RLock()
		for _, v := range l.vehicles {
			if v.Owner == wallet {
				pairs = append(pairs, pair{v.VehicleID, v.Name})
			}
		}
		l.mutex.RUnlock()
	case TargetWorldContent:
		l.mutex.RLock()
		for _, c := range l.worldContent {
			if c.Creator == wallet {
				name := c.Kind
				if name == "" {
					name = c.ContentID
				}
				pairs = append(pairs, pair{c.ContentID, name})
			}
		}
		l.mutex.RUnlock()
	case TargetClub, TargetShop:
		l.mutex.RLock()
		for _, c := range l.clubs {
			if c.OwnerWallet == wallet {
				name := c.Name
				if name == "" {
					name = c.ID
				}
				pairs = append(pairs, pair{c.ID, name})
			}
		}
		l.mutex.RUnlock()
	case TargetItem:
		if l.itemRegistry != nil {
			for _, it := range l.itemRegistry.GetCollection(wallet) {
				pairs = append(pairs, pair{it.ItemID, it.Name})
			}
		}
	case TargetBot:
		if l.aiEngine != nil {
			l.aiEngine.mu.RLock()
			for _, c := range l.aiEngine.citizens {
				owner := c.OriginWallet
				if owner == "" {
					owner = c.OwnerWallet
				}
				if owner == wallet {
					pairs = append(pairs, pair{c.Wallet, c.Name})
				}
			}
			l.aiEngine.mu.RUnlock()
		}
	case TargetChurch:
		faithChurchEngine.mu.RLock()
		for _, c := range faithChurchEngine.churches {
			if c.Owner == wallet {
				pairs = append(pairs, pair{c.ID, c.Name})
			}
		}
		faithChurchEngine.mu.RUnlock()
	}

	out := make([]BondedTargetView, 0, len(pairs))
	for _, pr := range pairs {
		out = append(out, BondedTargetView{
			Kind:      kind,
			TargetID:  pr.id,
			Name:      pr.name,
			KindLabel: kindLabel(kind),
			Branding:  l.BondedBrandingForTarget(kind, pr.id),
		})
	}
	// Deterministic order: the API answer must not depend on map iteration order.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].TargetID < out[j].TargetID
	})
	return out
}

// BondedTargetsForWallet returns every entity the wallet may brand, by kind. An empty kindFilter
// returns all allowed kinds. A card kind is refused (empty result), never enumerated.
func (l *Lobby) BondedTargetsForWallet(wallet, kindFilter string) []BondedTargetView {
	out := make([]BondedTargetView, 0)
	if wallet == "" || IsCardTargetKind(kindFilter) {
		return out
	}
	if kindFilter != "" {
		k := normalizeTargetKind(kindFilter)
		if !IsBindableTargetKind(k) {
			return out
		}
		return l.bondedTargetsOfKind(wallet, k)
	}
	for _, d := range bondableTargetKindDescriptions {
		out = append(out, l.bondedTargetsOfKind(wallet, d.Kind)...)
	}
	return out
}

// ── HTTP surface ──────────────────────────────────────────────────────────────

// bondedTargetRequest is the shared body for bind/unbind.
type bondedTargetRequest struct {
	AssetID    string `json:"asset_id"`
	TargetKind string `json:"target_kind"`
	TargetID   string `json:"target_id"`
}

// handleBindBondedAssetTarget brands an owned entity with one of the caller's bonded assets.
func (l *Lobby) handleBindBondedAssetTarget(w http.ResponseWriter, r *http.Request) {
	if l.bondedAssets == nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "registry not initialized"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	var req bondedTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}
	b, err := l.BindBondedAssetToTarget(wallet, req.AssetID, req.TargetKind, req.TargetID)
	if err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{
		"success":     true,
		"asset_id":    b.AssetID,
		"target_kind": b.TargetKind,
		"target_id":   b.TargetID,
		"branding":    l.BondedBrandingForTarget(b.TargetKind, b.TargetID),
	})
}

// handleUnbindBondedAssetTarget removes one of the caller's own brandings.
func (l *Lobby) handleUnbindBondedAssetTarget(w http.ResponseWriter, r *http.Request) {
	if l.bondedAssets == nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "registry not initialized"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	var req bondedTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}
	removed, err := l.UnbindBondedAssetTarget(wallet, req.AssetID, req.TargetKind, req.TargetID)
	if err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{
		"success":     true,
		"removed":     removed,
		"target_kind": normalizeTargetKind(req.TargetKind),
		"target_id":   req.TargetID,
	})
}

// handleBondedTargets lists every entity the wallet may brand, with the art it already wears.
// A card kind filter is refused explicitly rather than silently returning nothing, so a client
// that asks for cards is told the rule. With no wallet the response is still an honest policy
// read (wallet_required:true + no targets) so a disconnected visitor can read the rules and see
// which kinds exist — the policy is public; only the entity LIST is wallet-scoped.
func (l *Lobby) handleBondedTargets(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	kindFilter := strings.TrimSpace(r.URL.Query().Get("kind"))
	if IsCardTargetKind(kindFilter) {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "cards are excluded from bonded assets; no card entity can be branded",
		})
		return
	}
	if kindFilter != "" && !IsBindableTargetKind(kindFilter) {
		writeJSON(w, map[string]interface{}{"success": false, "error": "unknown target_kind " + kindFilter})
		return
	}
	if wallet == "" {
		writeJSON(w, map[string]interface{}{
			"success":         true,
			"wallet":          "",
			"wallet_required": true,
			"note":            "connect a wallet to list the entities you may brand",
			"targets":         []BondedTargetView{},
			"policy":          BondedBrandingPolicy(),
		})
		return
	}
	writeJSON(w, map[string]interface{}{
		"success": true,
		"wallet":  wallet,
		"targets": l.BondedTargetsForWallet(wallet, kindFilter),
		"policy":  BondedBrandingPolicy(),
	})
}

// handleBondedBranding is the public read path: the branding a target wears. No wallet is
// needed (branding is cosmetic, like an NFT image). Cards answer 400: they are excluded.
func (l *Lobby) handleBondedBranding(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	kind := q.Get("kind")
	if kind == "" {
		kind = q.Get("target_kind")
	}
	targetID := q.Get("target_id")
	if targetID == "" {
		targetID = q.Get("target-id")
	}
	if IsCardTargetKind(kind) {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "cards are excluded from bonded assets; a card carries no branding",
		})
		return
	}
	if kind == "" || targetID == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "kind and target_id are required"})
		return
	}
	k := normalizeTargetKind(kind)
	if !IsBindableTargetKind(k) {
		writeJSON(w, map[string]interface{}{"success": false, "error": "unknown target_kind " + kind})
		return
	}
	writeJSON(w, map[string]interface{}{
		"success":   true,
		"kind":      k,
		"target_id": targetID,
		"owner":     l.bondedTargetOwner(k, targetID),
		"branding":  l.BondedBrandingForTarget(k, targetID),
	})
}
