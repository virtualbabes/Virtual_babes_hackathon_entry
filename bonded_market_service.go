//go:build !js && !wasm

package main

// bonded_market_service.go
// §10.8 — THE PLAYER-TO-PLAYER BONDED-ASSET MARKET.
//
// A wallet that owns a bonded asset may offer it to another wallet for a price it names. Until now
// the only ways to ACQUIRE one were to create it (capped by the card/deck NFT supply, §23.5.1) or to
// receive it as a gift (`POST /api/assets/transfer`, which names the recipient and takes no money),
// so a player could never SELL art they had made. This file adds that trade.
//
// WHAT MAKES THE TRADE HONEST:
//   - THE MONEY IS EXACT AND INTEGER. The buyer pays `price`; the seller receives `price − fee`; the
//     house fee is `price × fee_bps / 10_000` (floor, in uint64) and is routed through the ONE
//     bonded-asset sink door (asset_life_engine.go). The split is exact by construction, so nothing
//     is minted, nothing is burned and nothing is lost to rounding — the debit equals the credit plus
//     the fee, and that is ASSERTED before any write.
//   - THE PRICE IS THE LISTING'S. A buy body may carry ONLY a listing id (`DisallowUnknownFields`),
//     so a client cannot name a price, a seller or an asset.
//   - §27.8 IS RE-PROVED ON EVERY READ AND EVERY WRITE. A listing is only honoured while the seller
//     still OWNS AND HOLDS the asset; one that changed hands or was burned is served as STALE with
//     the reason, and `TransferOwnership` itself cancels the active listings for the asset, so a
//     stale listing can never be bought.
//   - THE FEE DOES NOT GATE THE ASSET. Buying from another player is an ACQUISITION, exactly like the
//     placeholder shop, so §23.5.1's creation cap does not apply — a wallet with no cards may still
//     buy art from a wallet that has some. That is stated in the served rule.
//   - A WALLET CANNOT BUY ITS OWN LISTING, and a listing that is sold or cancelled is never buyable
//     again (the status is the gate, and only an ACTIVE listing can be bought).
//
// Storage lives on BondedAssetRegistry (it is the snapshot owner) and every listing accessor below is
// `...Locked` on purpose: it assumes `r.mu` is held. The authority (Lobby methods) takes `l.mutex`
// FIRST and calls into the registry second — the same nesting the placeholder shop already uses.

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	// BondedMarketFeeBps is the house's cut of a player-to-player sale, in basis points (250 = 2.5%).
	// It is SERVED, and the authority reads it from here — never from a request.
	BondedMarketFeeBps = 250

	// MinBondedListingMicro / MaxBondedListingMicro bound a price so the integer fee arithmetic cannot
	// overflow and a listing cannot be priced at zero (a gift has its own route: /api/assets/transfer).
	MinBondedListingMicro uint64 = 1
	MaxBondedListingMicro uint64 = 1_000_000 * 1_000_000 // 1,000,000 $VBV

	// Listing statuses. Only "active" may be bought.
	BondedListingActive    = "active"
	BondedListingSold      = "sold"
	BondedListingCancelled = "cancelled"
)

// BondedListing is one offer of one bonded asset at one price.
type BondedListing struct {
	ListingID    string    `json:"listing_id"`
	AssetID      string    `json:"asset_id"`
	SellerWallet string    `json:"seller_wallet"`
	PriceMicro   uint64    `json:"price_micro"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	// BuyerWallet / SoldAt / FeeMicro / NetMicro record a completed sale; the fee and the net are the
	// ACTUAL amounts moved, so the ledger arithmetic is auditable from the listing alone.
	BuyerWallet string    `json:"buyer_wallet,omitempty"`
	SoldAt      time.Time `json:"sold_at,omitempty"`
	FeeMicro    uint64    `json:"fee_micro,omitempty"`
	NetMicro    uint64    `json:"net_micro,omitempty"`
	CancelledAt time.Time `json:"cancelled_at,omitempty"`
	// Note states why a listing is no longer active.
	Note string `json:"note,omitempty"`
}

// bondMarketFee is the integer fee for a price. Floor division means the REMAINDER stays with the
// seller: the split is exact (buyer debit == seller credit + fee) and no rounding is silently lost.
func bondMarketFee(priceMicro uint64) uint64 {
	return priceMicro * uint64(BondedMarketFeeBps) / 10_000
}

// ── Listing storage (the registry owns it; every method below assumes r.mu is held) ─────────────

// putListingLocked writes (or replaces) a listing.
func (r *BondedAssetRegistry) putListingLocked(l *BondedListing) {
	if l == nil || l.ListingID == "" {
		return
	}
	if r.Listings == nil {
		r.Listings = make(map[string]*BondedListing)
	}
	cp := *l
	r.Listings[cp.ListingID] = &cp
}

// listingLocked reads one listing as a COPY, so nothing outside can mutate stored state.
func (r *BondedAssetRegistry) listingLocked(listingID string) (*BondedListing, bool) {
	if r.Listings == nil {
		return nil, false
	}
	l, ok := r.Listings[listingID]
	if !ok || l == nil {
		return nil, false
	}
	cp := *l
	return &cp, true
}

// ListingsSnapshot returns every listing as copies, newest first (deterministic order, so two reads
// of an unchanged market are identical).
func (r *BondedAssetRegistry) ListingsSnapshot() []BondedListing {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]BondedListing, 0, len(r.Listings))
	for _, l := range r.Listings {
		if l == nil {
			continue
		}
		out = append(out, *l)
	}
	sortBondedListings(out)
	return out
}

// sortBondedListings orders listings newest first with the id as a tie-break, so the order never
// depends on map iteration.
func sortBondedListings(list []BondedListing) {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0; j-- {
			older := list[j].CreatedAt.Before(list[j-1].CreatedAt)
			sameMoment := list[j].CreatedAt.Equal(list[j-1].CreatedAt) && list[j].ListingID < list[j-1].ListingID
			if older || sameMoment {
				list[j], list[j-1] = list[j-1], list[j]
				continue
			}
			break
		}
	}
}

// activeListingsForAssetLocked reports whether an ACTIVE listing already exists for an asset (one
// asset may be offered once at a time), and counts them.
func (r *BondedAssetRegistry) activeListingsForAssetLocked(assetID string) int {
	n := 0
	for _, l := range r.Listings {
		if l != nil && l.AssetID == assetID && l.Status == BondedListingActive {
			n++
		}
	}
	return n
}

// cancelActiveListingsForAssetLocked cancels every ACTIVE listing for an asset and returns how many
// it changed. Called by TransferOwnership and Burn (both hold r.mu), so a listing can never survive
// the ownership it was written under.
func (r *BondedAssetRegistry) cancelActiveListingsForAssetLocked(assetID, reason string) int {
	n := 0
	for _, l := range r.Listings {
		if l == nil || l.AssetID != assetID || l.Status != BondedListingActive {
			continue
		}
		l.Status = BondedListingCancelled
		l.CancelledAt = time.Now()
		l.Note = reason
		n++
	}
	return n
}

// NewBondedListingID mints the id used for a listing (uuid inside the asset's own prefix pattern).
func NewBondedListingID() string {
	return "BML-" + uuid.NewString()
}

// ── Authority (l.mutex is taken FIRST; registry locks are taken second) ───────────────────────

// bondMarketStaleReason reports why an ACTIVE listing can no longer be honoured, or "" when it can.
// §27.8 (owner == holder == seller) and the card exclusion are RE-PROVED here, so a listing that
// outlived its ownership is reported rather than bought.
func (l *Lobby) bondMarketStaleReason(listing BondedListing) string {
	if l.bondedAssets == nil {
		return "the bonded-asset registry is not initialised"
	}
	a, ok := l.bondedAssets.Get(listing.AssetID)
	if !ok || a == nil {
		return "the asset no longer exists (it was burned)"
	}
	if !strings.EqualFold(a.OwnerWallet, listing.SellerWallet) || !strings.EqualFold(a.HolderWallet, listing.SellerWallet) {
		return "the seller no longer owns and holds the asset"
	}
	if IsCardAssetType(a.AssetType) {
		return "the asset is card-typed and cards are excluded"
	}
	if a.Media == nil {
		return "the asset has no art"
	}
	return ""
}

// persistBondedRegistryAfterMarket writes the registry after a market mutation, following the
// established ItemRegistry precedent for a paid action ("Persist after mutation so restarts rehydrate
// the registry", item_shop_archetype.go). What it buys: without it, the registry is only snapshotted
// by the 15-minute persistence worker, so a restart inside that window resurrected the SELLER as the
// owner of art a buyer had already bought.
//
// HONEST SCOPE OF THE PROMISE. This makes the OWNERSHIP side durable when the call returns. The
// LEDGER has its own independent 15-minute snapshot (lobby_manager.go cacheSaveTicker), and there is
// no cross-file atomic commit, so a hard kill in the interval between the two writes can leave the
// money unsaved while the art has moved. That tear is AUDITABLE (the sale is written to the admin log
// with buyer, price, fee and net: BONDED_ASSET_SOLD) and it is not worse than the pre-existing
// behaviour, which could persist the money and LOSE the ownership. Closing it properly needs ONE
// atomic snapshot covering the ledger and the registry together, which is a persistence-architecture
// change recorded in AI-Brain/Problems.md rather than something this feature can honestly claim.
//
// It is SYNCHRONOUS on purpose: the trade holds l.mutex for the whole trade (ownership then money,
// one critical section), so this write happens under it - a bounded ~10 KB marshal + rename - and the
// alternative is worse for a money event, because "the response returned" would then mean "a
// goroutine is probably still writing it". Save serialises writers against the persistence worker.
//
// The mutation has ALREADY happened and the money has already moved, so a failed write is reported
// and never rolls the trade back (rolling back would be the worse lie: the buyer would keep the art
// and lose the money, or vice versa). Best-effort durability, honest logging.
func (l *Lobby) persistBondedRegistryAfterMarket(reason string) {
	if l == nil || l.bondedAssets == nil {
		return
	}
	if err := l.bondedAssets.Save(l); err != nil {
		log.Printf("[BondedMarket] persist after %s failed: %v", reason, err)
	}
}

// ListBondedAssetForSale offers one of the caller's own assets at a price the caller names. Every
// refusal happens before the listing is written, so a refused offer changes nothing.
func (l *Lobby) ListBondedAssetForSale(wallet, assetID string, priceMicro uint64) (*BondedListing, error) {
	if l.bondedAssets == nil {
		return nil, fmt.Errorf("registry not initialized")
	}
	w := canonicalViewerWallet(wallet)
	if w == "" {
		return nil, fmt.Errorf("wallet required")
	}
	assetID = strings.TrimSpace(assetID)
	if assetID == "" {
		return nil, fmt.Errorf("asset_id required")
	}
	if priceMicro < MinBondedListingMicro {
		return nil, fmt.Errorf("a listing must name a price of at least %d micro-$VBV (a gift uses /api/assets/transfer, which takes no money)", MinBondedListingMicro)
	}
	if priceMicro > MaxBondedListingMicro {
		return nil, fmt.Errorf("price_micro %d is above the %d micro-$VBV ceiling", priceMicro, MaxBondedListingMicro)
	}
	a, ok := l.bondedAssets.Get(assetID)
	if !ok || a == nil {
		return nil, fmt.Errorf("asset %s not found (it may have been burned)", assetID)
	}
	if IsCardAssetType(a.AssetType) {
		return nil, fmt.Errorf("asset %s is card-typed; cards are excluded from the market", assetID)
	}
	if a.Media == nil {
		return nil, fmt.Errorf("asset %s has no art, so there is nothing to sell", assetID)
	}
	if !strings.EqualFold(a.OwnerWallet, w) || !strings.EqualFold(a.HolderWallet, w) {
		return nil, fmt.Errorf("you may only offer an asset you own and hold (asset %s is owned or held by another wallet)", assetID)
	}
	l.bondedAssets.mu.Lock()
	if n := l.bondedAssets.activeListingsForAssetLocked(assetID); n > 0 {
		l.bondedAssets.mu.Unlock()
		return nil, fmt.Errorf("asset %s is already offered for sale; cancel that listing first", assetID)
	}
	listing := &BondedListing{
		ListingID: NewBondedListingID(), AssetID: assetID, SellerWallet: w,
		PriceMicro: priceMicro, Status: BondedListingActive, CreatedAt: time.Now(),
	}
	l.bondedAssets.putListingLocked(listing)
	l.bondedAssets.mu.Unlock()
	l.logAdminAudit("BONDED_ASSET_LISTED", w, fmt.Sprintf(
		"%s offered at %d micro-$VBV (house fee %d bps = %d micro)", assetID, priceMicro, BondedMarketFeeBps, bondMarketFee(priceMicro)))
	cp := *listing
	cp.Note = fmt.Sprintf("only the seller may cancel this listing; the buyer pays %d and the seller receives %d after the house fee", priceMicro, priceMicro-bondMarketFee(priceMicro))
	l.persistBondedRegistryAfterMarket("a listing")
	return &cp, nil
}

// CancelBondedListing withdraws an ACTIVE listing. Only the seller may cancel, and cancelling what is
// already sold or cancelled is refused with the status, so the caller learns what happened.
func (l *Lobby) CancelBondedListing(wallet, listingID string) (*BondedListing, error) {
	if l.bondedAssets == nil {
		return nil, fmt.Errorf("registry not initialized")
	}
	w := canonicalViewerWallet(wallet)
	if w == "" {
		return nil, fmt.Errorf("wallet required")
	}
	listingID = strings.TrimSpace(listingID)
	if listingID == "" {
		return nil, fmt.Errorf("listing_id required")
	}
	l.bondedAssets.mu.Lock()
	listing, ok := l.bondedAssets.listingLocked(listingID)
	if !ok {
		l.bondedAssets.mu.Unlock()
		return nil, fmt.Errorf("listing %s not found", listingID)
	}
	if listing.Status != BondedListingActive {
		l.bondedAssets.mu.Unlock()
		return nil, fmt.Errorf("listing %s is already %s and cannot be cancelled", listingID, listing.Status)
	}
	if !strings.EqualFold(listing.SellerWallet, w) {
		l.bondedAssets.mu.Unlock()
		return nil, fmt.Errorf("only the seller (%s) may cancel listing %s", listing.SellerWallet, listingID)
	}
	listing.Status = BondedListingCancelled
	listing.CancelledAt = time.Now()
	listing.Note = "cancelled by the seller"
	l.bondedAssets.putListingLocked(listing)
	l.bondedAssets.mu.Unlock()
	cp := *listing
	// Audited AFTER the registry lock is released, so the lock order here is never r.mu -> l.mutex
	// (the trade path takes l.mutex -> r.mu, and mixing the two orders deadlocks).
	l.logAdminAudit("BONDED_ASSET_LISTING_CANCELLED", w, fmt.Sprintf("%s for %s", listingID, listing.AssetID))
	l.persistBondedRegistryAfterMarket("a cancellation")
	return &cp, nil
}

// BuyBondedListing completes a trade. The price comes from the LISTING, never from the buyer.
//
// ORDERING (why nothing can be half-done): all of this happens under ONE l.mutex section.
//
//  1. re-read the listing and refuse unless it is still ACTIVE (a sold or cancelled listing is dead);
//  2. re-prove §27.8 (the seller must still own AND hold) — a stale listing is CANCELLED and refused;
//  3. verify the buyer is funded (case-insensitively resolved, the balanceKeyLocked rule);
//  4. move OWNERSHIP (the registry re-proves owner==holder==seller atomically), then
//  5. move the MONEY: debit buyer, credit seller `price − fee`, route `fee` to the sink, then
//  6. mark the listing SOLD with the buyer, the fee and the net that were ACTUALLY moved.
//
// Ownership moves BEFORE the money on purpose: if the guarded transfer refused, nothing has been
// spent yet, so there is no rollback path to get wrong. The money step is plain map arithmetic under
// the mutex with the balance already verified in step 3, so it cannot fail after the asset moved.
func (l *Lobby) BuyBondedListing(buyerWallet, listingID string) (*BondedListing, *BondedAsset, error) {
	if l.bondedAssets == nil {
		return nil, nil, fmt.Errorf("registry not initialized")
	}
	buyer := canonicalViewerWallet(buyerWallet)
	if buyer == "" {
		return nil, nil, fmt.Errorf("wallet required")
	}
	listingID = strings.TrimSpace(listingID)
	if listingID == "" {
		return nil, nil, fmt.Errorf("listing_id required")
	}
	l.mutex.Lock()
	defer l.mutex.Unlock()

	listing, ok := l.bondedAssets.Listing(listingID)
	if !ok {
		return nil, nil, fmt.Errorf("listing %s not found", listingID)
	}
	if listing.Status != BondedListingActive {
		return nil, nil, fmt.Errorf("listing %s is %s and cannot be bought", listingID, listing.Status)
	}
	if strings.EqualFold(listing.SellerWallet, buyer) {
		return nil, nil, fmt.Errorf("you cannot buy your own listing (%s)", listingID)
	}
	if reason := l.bondMarketStaleReason(*listing); reason != "" {
		// Cancel it so it stops being offered, and say why it could not be bought. The audit line is
		// written with the LOCKED form because l.mutex is already held here (logAdminAudit would
		// re-acquire it and deadlock).
		if n := l.bondedAssets.CancelListing(listingID, "stale: "+reason); n > 0 {
			l.logAdminAuditLocked("BONDED_ASSET_LISTING_STALE", listing.SellerWallet, fmt.Sprintf("%s: %s", listingID, reason))
		}
		return nil, nil, fmt.Errorf("listing %s can no longer be bought: %s", listingID, reason)
	}

	price := listing.PriceMicro
	if price < MinBondedListingMicro || price > MaxBondedListingMicro {
		return nil, nil, fmt.Errorf("listing %s carries a price (%d) outside the accepted range", listingID, price)
	}
	buyerKey := l.balanceKeyLocked(buyer)
	if l.playerBalances[buyerKey] < price {
		return nil, nil, fmt.Errorf("insufficient balance: %d micro-$VBV required for listing %s (you hold %d)",
			price, listingID, l.playerBalances[buyerKey])
	}
	sellerKey := l.balanceKeyLocked(listing.SellerWallet)
	fee := bondMarketFee(price)
	net := price - fee
	// The split must be exact: the debit equals the credit plus the house fee. That is guaranteed by
	// the floor division above, and asserted anyway so a future change cannot quietly unbalance it.
	if net+fee != price {
		return nil, nil, fmt.Errorf("listing %s would not reconcile (%d credit + %d fee != %d debit)", listingID, net, fee, price)
	}

	// 4+6. OWNERSHIP AND THE LISTING, ATOMICALLY. One registry call re-proves §27.8 for the seller,
	// moves the asset to the buyer and marks THIS listing sold together, so the ownership-cleanup rule
	// (a listing never outlives its ownership) cannot cancel the sale it is completing.
	if err := l.bondedAssets.CompleteSale(listing.AssetID, listing.SellerWallet, buyer, listingID, fee, net); err != nil {
		return nil, nil, fmt.Errorf("listing %s could not be completed: %v", listingID, err)
	}

	// 5. MONEY. One transfer, one fee, all integer micro-$VBV. The buyer's balance was verified above
	// under this same l.mutex, and every money mutation needs that mutex, so this cannot fail.
	l.playerBalances[buyerKey] -= price
	l.playerBalances[sellerKey] += net
	l.routeBondedFeeToSinkLocked("BONDED_ASSET_MARKET_FEE", fee)

	sold, _ := l.bondedAssets.Listing(listingID)
	asset, _ := l.bondedAssets.Get(listing.AssetID)
	// The LOCKED audit form: l.mutex is held for the whole trade (logAdminAudit would re-acquire it).
	l.logAdminAuditLocked("BONDED_ASSET_SOLD", buyer, fmt.Sprintf(
		"%s -> %s for %d micro-$VBV (seller %s received %d, house fee %d to the sink)",
		listing.AssetID, buyer, price, listing.SellerWallet, net, fee))
	if sold == nil {
		sold = listing
	}
	l.persistBondedRegistryAfterMarket("a sale")
	return sold, asset, nil
}

// ── Public registry accessors (each takes the registry lock itself) ────────────────────────────

// CompleteSale is the ONE atomic step of a market trade: it re-proves §27.8 for the seller, moves
// ownership to the buyer AND marks this listing sold, all under a single registry lock.
//
// It exists because the plain TransferOwnership path deliberately cancels the asset's active
// listings (a listing must not outlive the ownership it was written under) — which would cancel the
// very listing being completed. Doing both here, atomically, is what makes "sold" and "the asset
// changed hands" true together instead of one contradicting the other. Any OTHER active listing for
// the asset is still cancelled.
func (r *BondedAssetRegistry) CompleteSale(assetID, seller, buyer, listingID string, feeMicro, netMicro uint64) error {
	if strings.TrimSpace(buyer) == "" {
		return fmt.Errorf("buyer required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	a, err := r.guardOwnerHolder(assetID, seller)
	if err != nil {
		return err
	}
	listing, ok := r.listingLocked(strings.TrimSpace(listingID))
	if !ok {
		return fmt.Errorf("listing %s not found", listingID)
	}
	if listing.Status != BondedListingActive {
		return fmt.Errorf("listing %s is %s and cannot be completed", listingID, listing.Status)
	}
	if listing.AssetID != assetID {
		return fmt.Errorf("listing %s does not offer asset %s", listingID, assetID)
	}
	a.OwnerWallet = buyer
	a.HolderWallet = buyer
	listing.Status = BondedListingSold
	listing.BuyerWallet = buyer
	listing.SoldAt = time.Now()
	listing.FeeMicro = feeMicro
	listing.NetMicro = netMicro
	listing.Note = fmt.Sprintf("sold to %s: the house fee of %d micro reached the sink and the seller received %d", buyer, feeMicro, netMicro)
	r.putListingLocked(listing)
	// Any OTHER active listing for this asset is now stale by definition; cancel it so it cannot be
	// offered again.
	for _, other := range r.Listings {
		if other == nil || other.AssetID != assetID || other.ListingID == listing.ListingID || other.Status != BondedListingActive {
			continue
		}
		other.Status = BondedListingCancelled
		other.CancelledAt = time.Now()
		other.Note = "the asset was sold to another buyer"
	}
	return nil
}

// Listing reads one listing as a copy.
func (r *BondedAssetRegistry) Listing(listingID string) (*BondedListing, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.listingLocked(strings.TrimSpace(listingID))
}

// CancelListing marks an ACTIVE listing cancelled with a reason, returning how many changed (0 when
// it was already inactive).
func (r *BondedAssetRegistry) CancelListing(listingID, reason string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	listing, ok := r.listingLocked(strings.TrimSpace(listingID))
	if !ok || listing.Status != BondedListingActive {
		return 0
	}
	listing.Status = BondedListingCancelled
	listing.CancelledAt = time.Now()
	listing.Note = reason
	r.putListingLocked(listing)
	return 1
}

// ── Read view ────────────────────────────────────────────────────────────────────────────────

// BondedMarketView is the read payload: every listing with its state, the caller's own rows marked,
// the rules, and the served reason the §23.5.1 creation cap does not apply to a purchase.
func (l *Lobby) BondedMarketView(wallet string) map[string]interface{} {
	w := canonicalViewerWallet(wallet)
	active, mine, sold, cancelled, stale := 0, 0, 0, 0, 0
	all := l.bondedAssets.ListingsSnapshot()
	rows := make([]map[string]interface{}, 0, len(all))
	staleReasons := make([]string, 0)
	for _, listing := range all {
		row := map[string]interface{}{
			"listing_id":  listing.ListingID,
			"asset_id":    listing.AssetID,
			"seller":      listing.SellerWallet,
			"price_micro": listing.PriceMicro,
			"status":      listing.Status,
			"created_at":  listing.CreatedAt,
		}
		if listing.Status == BondedListingActive {
			active++
			// The asset's real art and the STALE reason are joined in, so the UI never has to guess
			// and a buyer is never offered something that cannot be delivered.
			if a, ok := l.bondedAssets.Get(listing.AssetID); ok && a != nil {
				row["asset_name"] = a.Name
				row["asset_type"] = a.AssetType
				row["sku"] = a.Sku
				row["source"] = a.Source
				row["media"] = a.Media
				row["owner_wallet"] = a.OwnerWallet
			}
			if reason := l.bondMarketStaleReason(listing); reason != "" {
				row["stale_reason"] = reason
				stale++
				staleReasons = append(staleReasons, listing.ListingID+": "+reason)
			}
			fee := bondMarketFee(listing.PriceMicro)
			row["fee_micro"] = fee
			row["net_micro"] = listing.PriceMicro - fee
		} else {
			switch listing.Status {
			case BondedListingSold:
				sold++
				row["buyer"] = listing.BuyerWallet
				row["sold_at"] = listing.SoldAt
				row["fee_micro"] = listing.FeeMicro
				row["net_micro"] = listing.NetMicro
			default:
				cancelled++
			}
			row["note"] = listing.Note
		}
		if w != "" && strings.EqualFold(listing.SellerWallet, w) {
			row["mine"] = true
			mine++
		}
		rows = append(rows, row)
	}
	return map[string]interface{}{
		"success":          true,
		"wallet":           w,
		"wallet_required":  w == "",
		"listings":         rows,
		"count":            len(rows),
		"active_count":     active,
		"mine_count":       mine,
		"sold_count":       sold,
		"cancelled_count":  cancelled,
		"stale_count":      stale,
		"stale":            staleReasons,
		"fee_bps":          BondedMarketFeeBps,
		"min_price_micro":  MinBondedListingMicro,
		"max_price_micro":  MaxBondedListingMicro,
		"capacity":         l.BondedAssetCapacityForWallet(w),
		"acquisition_rule": "Buying from another player is an ACQUISITION, exactly like the placeholder shop, so §23.5.1's creation cap does not limit it: a wallet the engine records no card/deck NFT for may still buy art from a wallet that has some. The cap limits how many bonded assets a wallet CREATES.",
		"rule":             fmt.Sprintf("Only the seller may cancel a listing. A listing is honoured only while the seller still owns AND holds the asset (§27.8) and the art is not card-typed (§10.1); one that outlived its ownership is reported as STALE with the reason and is cancelled when a buyer tries it. The price is the listing's — a buy body carries only a listing id. The house fee is %d bps of the price, routed to the faucet sink; the seller receives the rest, so the buyer's debit always equals the seller's credit plus the fee.", BondedMarketFeeBps),
	}
}

// ── HTTP ─────────────────────────────────────────────────────────────────────────────────────

// handleBondedMarket reads the market (GET). Public: a visitor may browse what is for sale, and a
// wallet additionally gets its own rows marked.
func (l *Lobby) handleBondedMarket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, map[string]interface{}{"success": false, "error": "method not allowed"})
		return
	}
	if l.bondedAssets == nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "registry not initialized"})
		return
	}
	writeJSON(w, l.BondedMarketView(l.getWalletFromRequest(r)))
}

// handleBondedMarketList offers one of the caller's own assets. The body may carry ONLY
// asset_id + price_micro (DisallowUnknownFields), so a client cannot name a seller or a status.
func (l *Lobby) handleBondedMarketList(w http.ResponseWriter, r *http.Request) {
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
		AssetID    string `json:"asset_id"`
		PriceMicro uint64 `json:"price_micro"`
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, 4<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "invalid listing body: only asset_id and price_micro are accepted (a client never names a seller, a status or a fee)",
		})
		return
	}
	listing, err := l.ListBondedAssetForSale(wallet, req.AssetID, req.PriceMicro)
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"success":     false,
			"error":       err.Error(),
			"fee_bps":     BondedMarketFeeBps,
			"capacity":    l.BondedAssetCapacityForWallet(wallet),
			"market_view": l.BondedMarketView(wallet),
		})
		return
	}
	writeJSON(w, map[string]interface{}{
		"success":  true,
		"wallet":   wallet,
		"listing":  listing,
		"fee_bps":  BondedMarketFeeBps,
		"capacity": l.BondedAssetCapacityForWallet(wallet),
	})
}

// handleBondedMarketCancel withdraws the caller's own listing.
func (l *Lobby) handleBondedMarketCancel(w http.ResponseWriter, r *http.Request) {
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
		ListingID string `json:"listing_id"`
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, 4<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "invalid cancel body: only listing_id is accepted",
		})
		return
	}
	listing, err := l.CancelBondedListing(wallet, req.ListingID)
	if err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{
		"success": true,
		"wallet":  wallet,
		"listing": listing,
	})
}

// handleBondedMarketBuy buys a listing. The body may carry ONLY a listing id — the price is the
// listing's, so it can never be named (let alone reduced) by the buyer.
func (l *Lobby) handleBondedMarketBuy(w http.ResponseWriter, r *http.Request) {
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
		ListingID string `json:"listing_id"`
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, 4<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   "invalid buy body: only listing_id is accepted (the price is the listing's, never the buyer's)",
		})
		return
	}
	listing, asset, err := l.BuyBondedListing(wallet, req.ListingID)
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"success":  false,
			"error":    err.Error(),
			"fee_bps":  BondedMarketFeeBps,
			"capacity": l.BondedAssetCapacityForWallet(wallet),
		})
		return
	}
	writeJSON(w, map[string]interface{}{
		"success":  true,
		"wallet":   wallet,
		"listing":  listing,
		"asset":    asset,
		"fee_bps":  BondedMarketFeeBps,
		"capacity": l.BondedAssetCapacityForWallet(wallet),
		"note": "the asset is yours now: it is in your chest and can be worn on anything you own " +
			"that bonding accepts (cards remain excluded).",
	})
}
