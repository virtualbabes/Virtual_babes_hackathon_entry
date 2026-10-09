//go:build !js && !wasm

package main

// placeholder_assets.go
// §23.5.1 / §10.6 — THE NPC-HELPER PLACEHOLDER ART CATALOGUE.
//
// Brendan's rule this file implements: "the placeholder images should be usable free for theming
// and should be available to all users as bonded assets available for purchase, they should also
// receive the first needed images as bonded assets free".
//
// So there are exactly TWO acquisition paths for placeholder art, and this file owns both:
//
//	FREE STARTER PACK — the six frames the app's own slideshows show (three on app boot / main
//	                    menu, three as the World Dashboard background). Every wallet is GRANTED
//	                    them as bonded assets, once, at no cost, so a brand-new player can
//	                    re-theme what they are looking at without buying anything.
//	SHOP              — the rest of the pack, priced by the SERVER (PlaceholderSkuPriceMicro) and
//	                    charged through the ONE bonded-asset fee door
//	                    (chargeBondedAssetFeeLocked → token sink), so no client prices its own
//	                    purchase and the Industrial Loop still reconciles.
//
// WHY A GRANT AND A PURCHASE ARE NOT CAPACITY-GATED (§23.5.1): the cap limits how many bonded
// assets a wallet may CREATE (one per card/deck NFT it holds). A grant and a shop purchase are
// ACQUISITIONS — the asset is created by the house, not by the player — and Brendan's own rule
// requires the pack to be "available to ALL users", which a card-holder-only cap would break for
// every wallet that holds no cards. The served payload states this, and the wallet's own creation
// budget is still reported beside it so nothing is hidden.
//
// THE ART ITSELF IS THE DISK. The catalogue is built by scanning the served pack
// (Public/Assets/Images/NPC-helpers/<Character>/<Character>-NNN.png), so the file list IS the SKU
// list: no hand-maintained frame table can drift from the art. Every SKU's media declaration
// carries REAL values read from the real file — byte size, PNG IHDR dimensions and the sha256 of
// the bytes — and ValidateBondedMedia (the same boundary every user-declared media passes) must
// accept it before the SKU may be granted or sold. A frame that cannot satisfy the policy is
// REPORTED as unavailable with its reason and is never minted, never granted and never sold.

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const (
	// PlaceholderSkuPriceMicro is the price of ONE catalogue frame, in uint64 micro-$VBV. It is
	// served, and the purchase handler reads the price from HERE — never from a request body.
	PlaceholderSkuPriceMicro uint64 = 100 * 1_000_000

	// placeholderArtRoot is the served pack, relative to the process working directory.
	placeholderArtRoot = "Public/Assets/Images/NPC-helpers"

	// placeholderSkuURIPrefix is the SAME-ORIGIN form a catalogue asset's media uses: this process
	// serves the pack itself, so no host is invented and no external gateway is required.
	placeholderSkuURIPrefix = "/Assets/Images/NPC-helpers/"

	// PlaceholderSourceStarter marks an asset granted by the free starter pack.
	PlaceholderSourceStarter = "starter"
	// PlaceholderSourceShop marks an asset bought from the catalogue.
	PlaceholderSourceShop = "shop"
)

// placeholderStarterSKUs is the FREE starter pack: exactly the six frames the app's own slideshows
// display (app boot / main menu ×3, World Dashboard background ×3 — one image of each NPC helper
// each). slide_theming.go declares the slots that read them, and a test pins that every slot's
// default SKU is in this list, so "the first needed images" can never drift apart from "the images
// the slides actually need".
var placeholderStarterSKUs = []string{
	"npc-anya-001",          // boot / main menu slide 1
	"npc-crypto-seraph-002", // boot / main menu slide 2
	"npc-vbabes-046",        // boot / main menu slide 3
	"npc-vbabes-054",        // World Dashboard slide 1
	"npc-crypto-seraph-004", // World Dashboard slide 2
	"npc-anya-041",          // World Dashboard slide 3
}

// PlaceholderSku is one frame of the pack, as the server knows it. `path` is unexported on
// purpose: it is the local filesystem location used to read bytes/dimensions/hash, and it must
// never reach a client payload.
type PlaceholderSku struct {
	SKU         string `json:"sku"`
	Character   string `json:"character"`
	Frame       int    `json:"frame"`
	Name        string `json:"name"`
	URI         string `json:"uri"`
	MimeType    string `json:"mime_type"`
	Bytes       uint64 `json:"bytes"`
	Width       uint64 `json:"width"`
	Height      uint64 `json:"height"`
	PriceMicro  uint64 `json:"price_micro"`
	Starter     bool   `json:"starter"`
	Purchasable bool   `json:"purchasable"`
	Unavailable string `json:"unavailable,omitempty"`

	path string
}

// placeholderSkuFileRe matches the RENAMED, URL-safe file scheme "<Prefix>-NNN.png". The old
// shipped names ("Anya (1).png") contained spaces and parentheses and are deliberately NOT
// accepted: a path that needs escaping is a path that breaks in shells, globs and manifests.
var placeholderSkuFileRe = regexp.MustCompile(`^([A-Za-z0-9]+(?:-[A-Za-z0-9]+)*)-(\d{1,4})\.png$`)

// placeholderSkuID is the canonical SKU id: character + zero-padded frame, lowercase.
func placeholderSkuID(character string, frame int) string {
	return "npc-" + strings.ToLower(character) + "-" + fmt.Sprintf("%03d", frame)
}

func placeholderIsStarterSKU(sku string) bool {
	for _, s := range placeholderStarterSKUs {
		if s == sku {
			return true
		}
	}
	return false
}

var (
	placeholderCatalogOnce sync.Once
	placeholderCatalogList []PlaceholderSku
	placeholderCatalogIdx  map[string]PlaceholderSku
	placeholderCatalogErr  error

	placeholderHashMu sync.Mutex
	placeholderHashes = make(map[string]placeholderHashEntry)
)

// placeholderHashEntry is a cached digest together with the file identity it was computed from, so
// a frame replaced on disk is re-hashed rather than answered with a stale hash.
type placeholderHashEntry struct {
	sum     string
	size    int64
	modNano int64
}

// scanPlaceholderPack reads a pack root into SKUs: the FILE LIST IS THE SKU LIST, so no
// hand-maintained frame table can drift from the art. It is shared by the served catalogue and by
// the derivative pass (placeholder_derivatives.go), so both agree on what a SKU is, and a test can
// point it at a temporary pack.
func scanPlaceholderPack(root string) ([]PlaceholderSku, error) {
	list := make([]PlaceholderSku, 0, 128)
	dirs, err := os.ReadDir(root)
	if err != nil {
		return list, fmt.Errorf("placeholder pack not readable at %s: %w", root, err)
	}
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		character := d.Name()
		files, err := os.ReadDir(filepath.Join(root, character))
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			m := placeholderSkuFileRe.FindStringSubmatch(f.Name())
			if m == nil || !strings.EqualFold(m[1], character) {
				continue // a stray or legacy-named file is not a SKU
			}
			frame, err := strconv.Atoi(m[2])
			if err != nil {
				continue
			}
			info, err := f.Info()
			if err != nil {
				continue
			}
			rec := PlaceholderSku{
				SKU:        placeholderSkuID(character, frame),
				Character:  character,
				Frame:      frame,
				Name:       fmt.Sprintf("%s frame %d", character, frame),
				URI:        placeholderSkuURIPrefix + character + "/" + f.Name(),
				MimeType:   "image/png",
				Bytes:      uint64(info.Size()),
				PriceMicro: PlaceholderSkuPriceMicro,
				path:       filepath.Join(root, character, f.Name()),
			}
			rec.Starter = placeholderIsStarterSKU(rec.SKU)
			w, h, derr := readPNGDimensions(rec.path)
			rec.Width, rec.Height = w, h
			switch {
			case derr != nil:
				rec.Unavailable = "art could not be read as a PNG: " + derr.Error()
			case rec.Bytes == 0:
				rec.Unavailable = "art file is empty"
			case rec.Bytes > BondedMediaMaxBytes:
				rec.Unavailable = fmt.Sprintf("art is %d bytes, above the %d byte bonded-media cap",
					rec.Bytes, BondedMediaMaxBytes)
			case w == 0 || h == 0:
				rec.Unavailable = "art declares no pixel dimensions"
			case w > BondedMediaMaxDimension || h > BondedMediaMaxDimension:
				rec.Unavailable = fmt.Sprintf("art is %dx%d px, above the %d px per-side cap", w, h, BondedMediaMaxDimension)
			}
			rec.Purchasable = rec.Unavailable == ""
			if rec.Starter && !rec.Purchasable {
				// A starter frame that cannot satisfy the media policy must never be granted, and
				// the reason must be loud: the slideshow would otherwise point at art the chest
				// can never hold.
				rec.Unavailable = "STARTER ART UNAVAILABLE: " + rec.Unavailable
			}
			list = append(list, rec)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].SKU < list[j].SKU })
	return list, nil
}

// buildPlaceholderCatalogue scans the served pack once. A read error is recorded (and served) so
// an empty catalogue is never presented as "this game has no art".
func buildPlaceholderCatalogue() {
	list, err := scanPlaceholderPack(placeholderArtRoot)
	placeholderCatalogErr = err
	idx := make(map[string]PlaceholderSku, len(list))
	for _, s := range list {
		idx[s.SKU] = s
	}
	placeholderCatalogList = list
	placeholderCatalogIdx = idx
}

func placeholderCatalogueIndex() map[string]PlaceholderSku {
	placeholderCatalogOnce.Do(buildPlaceholderCatalogue)
	return placeholderCatalogIdx
}

// PlaceholderCatalogue returns a copy of every SKU in the pack, in deterministic SKU order.
func PlaceholderCatalogue() []PlaceholderSku {
	placeholderCatalogOnce.Do(buildPlaceholderCatalogue)
	return append([]PlaceholderSku{}, placeholderCatalogList...)
}

// PlaceholderCatalogueError reports a pack that could not be read at all (nil when it was).
func PlaceholderCatalogueError() error {
	placeholderCatalogOnce.Do(buildPlaceholderCatalogue)
	return placeholderCatalogErr
}

// placeholderSku looks a SKU up, case-insensitively (ids are lowercase-canonical).
func placeholderSku(sku string) (PlaceholderSku, bool) {
	s, ok := placeholderCatalogueIndex()[strings.ToLower(strings.TrimSpace(sku))]
	return s, ok
}

// readPNGDimensions reads the real pixel size from the IHDR chunk — 24 bytes of the file, not a
// full decode, and never a guess.
func readPNGDimensions(path string) (uint64, uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	var hdr [24]byte
	if _, err := io.ReadFull(f, hdr[:]); err != nil {
		return 0, 0, fmt.Errorf("file is shorter than a PNG header")
	}
	if hdr[0] != 0x89 || string(hdr[1:4]) != "PNG" {
		return 0, 0, fmt.Errorf("not a PNG")
	}
	if string(hdr[12:16]) != "IHDR" {
		return 0, 0, fmt.Errorf("PNG has no IHDR chunk")
	}
	return uint64(binary.BigEndian.Uint32(hdr[16:20])), uint64(binary.BigEndian.Uint32(hdr[20:24])), nil
}

// placeholderSkuHash returns the sha256 of the art bytes, computed once per SKU and CACHED AGAINST
// THE FILE'S IDENTITY (size + modification time). This is the one place the server can honestly say
// the declared hash IS the art's hash rather than a claim somebody typed: the bytes are local, so
// they are actually hashed.
//
// The identity check is what makes the cache safe: a frame replaced on disk gets re-hashed instead
// of answering with a stale digest, which is what lets the derivative manifest notice that art
// changed (placeholder_derivatives.go). The known limit is the filesystem's timestamp resolution —
// a replacement that keeps BOTH the exact byte size and the exact modification time is not
// detectable by this cheap check.
func placeholderSkuHash(s PlaceholderSku) (string, error) {
	size, modNano := int64(-1), int64(-1)
	if info, statErr := os.Stat(s.path); statErr == nil {
		size, modNano = info.Size(), info.ModTime().UnixNano()
	}
	placeholderHashMu.Lock()
	if e, ok := placeholderHashes[s.SKU]; ok && size >= 0 && e.size == size && e.modNano == modNano {
		placeholderHashMu.Unlock()
		return e.sum, nil
	}
	placeholderHashMu.Unlock()

	f, err := os.Open(s.path)
	if err != nil {
		return "", fmt.Errorf("could not read placeholder art %s: %w", s.SKU, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("could not hash placeholder art %s: %w", s.SKU, err)
	}
	sum := hex.EncodeToString(h.Sum(nil))
	placeholderHashMu.Lock()
	placeholderHashes[s.SKU] = placeholderHashEntry{sum: sum, size: size, modNano: modNano}
	placeholderHashMu.Unlock()
	return sum, nil
}

// placeholderSkuMedia builds the media declaration for one SKU and puts it through the SAME
// ValidateBondedMedia boundary every user-declared payload passes — so a catalogue asset can never
// hold media a player could not. A SKU whose art fails the policy is refused, never adjusted.
func placeholderSkuMedia(s PlaceholderSku) (*BondedMedia, error) {
	if !s.Purchasable {
		return nil, fmt.Errorf("placeholder frame %s is not available as a bonded asset: %s", s.SKU, s.Unavailable)
	}
	sum, err := placeholderSkuHash(s)
	if err != nil {
		return nil, err
	}
	media := &BondedMedia{
		URI: s.URI, MimeType: s.MimeType, HashHex: sum,
		Bytes: s.Bytes, Width: s.Width, Height: s.Height,
	}
	if err := ValidateBondedMedia(media); err != nil {
		return nil, fmt.Errorf("placeholder frame %s does not satisfy the media policy: %w", s.SKU, err)
	}
	return media, nil
}

// ── Acquisition: the free starter pack and the shop ──────────────────────────

// StarterGrantResult reports exactly what a provisioning call did, including what it could NOT do
// — so a player is never told "you have the starter pack" when a frame was skipped.
type StarterGrantResult struct {
	Granted      []*BondedAsset      `json:"granted"`
	AlreadyOwned []*BondedAsset      `json:"already_owned"`
	Skipped      []string            `json:"skipped"`
	StarterSKUs  []string            `json:"starter_skus"`
	Capacity     BondedAssetCapacity `json:"capacity"`
}

// EnsureStarterAssets grants the six slideshow frames to a wallet, once and free. It is IDEMPOTENT
// by SKU ownership rather than by a separate "already granted" flag: the assets themselves ARE the
// record, so the grant needs no extra persisted state, cannot double-mint on a double click, and
// behaves honestly for a wallet that gave a frame away (it can receive it again — the gift was
// theirs to make).
//
// No capacity gate: see the file header — a GRANT is an acquisition, not a creation, and the pack
// must be available to every wallet, including one that holds no cards.
func (l *Lobby) EnsureStarterAssets(wallet string) (*StarterGrantResult, error) {
	w := strings.ToLower(strings.TrimSpace(wallet))
	if w == "" {
		return nil, fmt.Errorf("wallet required")
	}
	if l.bondedAssets == nil {
		return nil, fmt.Errorf("registry not initialized")
	}
	res := &StarterGrantResult{
		Granted:      make([]*BondedAsset, 0, len(placeholderStarterSKUs)),
		AlreadyOwned: make([]*BondedAsset, 0, len(placeholderStarterSKUs)),
		Skipped:      make([]string, 0),
		StarterSKUs:  append([]string{}, placeholderStarterSKUs...),
	}
	for _, sku := range placeholderStarterSKUs {
		s, ok := placeholderSku(sku)
		if !ok {
			res.Skipped = append(res.Skipped, sku+": not present in the shipped art pack")
			continue
		}
		if !s.Purchasable {
			res.Skipped = append(res.Skipped, sku+": "+s.Unavailable)
			continue
		}
		if existing, owned := l.bondedAssets.OwnedSku(w, sku); owned {
			res.AlreadyOwned = append(res.AlreadyOwned, existing)
			continue
		}
		media, err := placeholderSkuMedia(s)
		if err != nil {
			res.Skipped = append(res.Skipped, sku+": "+err.Error())
			continue
		}
		a, err := l.bondedAssets.MintCatalogueAsset(w, AssetBackground, s.Name, sku, PlaceholderSourceStarter, media)
		if err != nil {
			res.Skipped = append(res.Skipped, sku+": "+err.Error())
			continue
		}
		res.Granted = append(res.Granted, a)
	}
	res.Capacity = l.BondedAssetCapacityForWallet(w)
	if len(res.Granted) > 0 {
		l.logAdminAudit("STARTER_ASSETS_GRANTED", w, fmt.Sprintf(
			"granted %d free starter frame(s) (capacity is unaffected: a grant is acquisition, not creation)",
			len(res.Granted)))
	}
	return res, nil
}

// BuyPlaceholderSku sells ONE catalogue frame at the SERVED price. The price comes from
// PlaceholderSkuPriceMicro, never from the request, and the fee goes through the single
// bonded-asset fee door so the sink accounting still reconciles (§25.6.1/§26.4.2 pattern).
//
// ORDERING (why there is no refund path): the balance is checked first, then the asset is minted,
// then the fee is charged. If the fee door somehow refuses after the mint, the mint is ROLLED BACK
// by burning it, so money can never be taken for an asset that was not delivered and an asset can
// never be delivered unpaid. The registry insert itself cannot fail once the media has validated,
// so the rollback is a belt-and-braces path, not the normal one.
//
// A repeat purchase of the same SKU is allowed on purpose: a player may want a second copy to
// transfer or gift, and the price is the limiter — this is not a one-per-wallet rule.
func (l *Lobby) BuyPlaceholderSku(wallet, sku string) (*BondedAsset, error) {
	w := strings.ToLower(strings.TrimSpace(wallet))
	if w == "" {
		return nil, fmt.Errorf("wallet required")
	}
	if l.bondedAssets == nil {
		return nil, fmt.Errorf("registry not initialized")
	}
	s, ok := placeholderSku(sku)
	if !ok {
		return nil, fmt.Errorf("unknown placeholder frame %q", strings.TrimSpace(sku))
	}
	// The media (and the sha256 of the real bytes) is built BEFORE any lock or charge, so a SKU
	// whose art cannot satisfy the policy is refused with nothing spent.
	media, err := placeholderSkuMedia(s)
	if err != nil {
		return nil, err
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	// The engine keys balances by the spelling it was given — in production that is the lowercase
	// request wallet, but this method must not tell a funded caller it is broke because it spoke a
	// different casing. Resolve the existing key (lock held) and debit THAT entry.
	key := l.balanceKeyLocked(w)
	if l.playerBalances[key] < PlaceholderSkuPriceMicro {
		return nil, fmt.Errorf("insufficient balance: %d micro-$VBV required to buy %s",
			PlaceholderSkuPriceMicro, s.SKU)
	}
	a, err := l.bondedAssets.MintCatalogueAsset(w, AssetBackground, s.Name, s.SKU, PlaceholderSourceShop, media)
	if err != nil {
		return nil, err
	}
	if err := l.chargeBondedAssetFeeLocked(key, "PLACEHOLDER_SKU_FEE", PlaceholderSkuPriceMicro); err != nil {
		_ = l.bondedAssets.Burn(a.AssetID, w)
		return nil, err
	}
	l.logAdminAuditLocked("PLACEHOLDER_SKU_PURCHASED", w, fmt.Sprintf("%s (%s) for %d micro-$VBV",
		s.SKU, s.URI, PlaceholderSkuPriceMicro))
	return a, nil
}

// balanceKeyLocked resolves the playerBalances entry for a wallet spelling, case-insensitively, so
// a caller that speaks another casing reaches the SAME balance. Caller must hold l.mutex (the map
// is read directly). A wallet with no record yet answers with the wallet as given — a real zero.
func (l *Lobby) balanceKeyLocked(wallet string) string {
	if _, ok := l.playerBalances[wallet]; ok {
		return wallet
	}
	for k := range l.playerBalances {
		if strings.EqualFold(k, wallet) {
			return k
		}
	}
	return wallet
}

// ── HTTP ─────────────────────────────────────────────────────────────────────

// handlePlaceholderCatalogue serves the whole pack: every SKU, its price, whether it is in the free
// starter set, and the reason any frame is unavailable. Public — the catalogue is the same for
// every player, and a wallet-less visitor can see what a purchase would buy.
func (l *Lobby) handlePlaceholderCatalogue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, map[string]interface{}{"success": false, "error": "method not allowed"})
		return
	}
	list := PlaceholderCatalogue()
	available := 0
	for _, s := range list {
		if s.Purchasable {
			available++
		}
	}
	payload := map[string]interface{}{
		"success":      true,
		"count":        len(list),
		"purchasable":  available,
		"price_micro":  PlaceholderSkuPriceMicro,
		"starter_skus": append([]string{}, placeholderStarterSKUs...),
		"skus":         PlaceholderCatalogueViews(),
		// §10.6 derivatives: the progress of the light-art pass, so a client can tell whether the
		// art it is about to fetch is the light rendition or still the multi-megabyte original.
		"derivative_status": PlaceholderDerivativeStatusForClient(),
		"media":             BondedMediaLimitsForClient(),
		"rule": "The placeholder pack is usable free for theming: every wallet is granted the six " +
			"frames the app's own slideshows show (three on boot / main menu, three as the World " +
			"Dashboard background). The rest of the pack is available to all users as bonded " +
			"assets for purchase at the served price. Capacity (§23.5.1) caps CREATION only: a " +
			"grant and a purchase are acquisitions, so the pack is reachable by every wallet.",
	}
	if err := PlaceholderCatalogueError(); err != nil {
		payload["pack_error"] = err.Error()
	}
	writeJSON(w, payload)
}

// handleStarterAssets provisions the free starter pack. POST-only (a read must never write state)
// and idempotent, so a client may safely call it whenever a wallet connects.
func (l *Lobby) handleStarterAssets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, map[string]interface{}{"success": false, "error": "method not allowed"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	res, err := l.EnsureStarterAssets(wallet)
	if err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{
		"success":       true,
		"wallet":        wallet,
		"granted":       res.Granted,
		"already_owned": res.AlreadyOwned,
		"skipped":       res.Skipped,
		"starter_skus":  res.StarterSKUs,
		"capacity":      res.Capacity,
		"note": "the starter pack is granted free and is NOT charged against your bonded-asset " +
			"creation budget: capacity (§23.5.1) caps creation, and a grant is acquisition.",
	})
}

// handleBuyPlaceholderSku sells one frame. The body may carry ONLY a sku — an unknown field (such
// as a client-declared price) cannot even be parsed, so the price is never client-supplied.
func (l *Lobby) handleBuyPlaceholderSku(w http.ResponseWriter, r *http.Request) {
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
		SKU string `json:"sku"`
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, 4<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "invalid purchase body: only sku is accepted (a client never declares a price)",
		})
		return
	}
	if strings.TrimSpace(req.SKU) == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "sku required"})
		return
	}
	a, err := l.BuyPlaceholderSku(wallet, req.SKU)
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"success":     false,
			"error":       err.Error(),
			"price_micro": PlaceholderSkuPriceMicro,
			"capacity":    l.BondedAssetCapacityForWallet(wallet),
		})
		return
	}
	writeJSON(w, map[string]interface{}{
		"success":     true,
		"wallet":      wallet,
		"asset":       a,
		"sku":         a.Sku,
		"price_micro": PlaceholderSkuPriceMicro,
		"capacity":    l.BondedAssetCapacityForWallet(wallet),
	})
}
