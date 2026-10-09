package main

// bonded_market_test.go
// §10.8 — proof for the player-to-player bonded-asset market: the money moves EXACTLY (debit ==
// credit + fee, and the fee really reaches the sink), ownership moves with it, every refusal moves
// nothing at all, a listing can never outlive the ownership it was written under, the price is never
// client-supplied, and a sale survives a restart.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// marketTestMedia is a legal media declaration for a test asset (the mint path does not validate, but
// the market REQUIRES art, so the test asset has to carry some).
func marketTestMedia() *BondedMedia {
	return &BondedMedia{
		URI:      "/Assets/Images/NPC-helpers/Anya/Anya-001.png",
		MimeType: "image/png",
		HashHex:  strings.Repeat("ab", 32),
		Bytes:    1024, Width: 64, Height: 64,
	}
}

// marketTestSellerAsset gives `seller` one bonded asset with art and returns its id.
func marketTestSellerAsset(t *testing.T, l *Lobby, seller string) string {
	t.Helper()
	if l.bondedAssets == nil {
		// The shared bonded-asset test lobby deliberately does not build the registry (other tests
		// install their own); a market test needs one, so it is created here if it is missing.
		l.bondedAssets = NewBondedAssetRegistry()
	}
	a, err := l.bondedAssets.MintBondedAssetWithMedia(seller, seller, seller, AssetBackground, "Test Art", MoodNeutral, 0, marketTestMedia())
	if err != nil {
		t.Fatalf("mint test asset: %v", err)
	}
	return a.AssetID
}

// TestBondedMarketSaleMovesOwnershipAndMoneyExactly is the core trade: the buyer pays the listing's
// price, the seller receives the price minus the house fee, the fee reaches the sink, and the asset
// changes hands.
func TestBondedMarketSaleMovesOwnershipAndMoneyExactly(t *testing.T) {
	l, faucet := newTestLobbyForBonded(t)
	const seller, buyer = "seller", "buyer"
	l.playerBalances[seller] = 0
	l.playerBalances[buyer] = 50_000 * 1_000_000
	assetID := marketTestSellerAsset(t, l, seller)

	const price uint64 = 10_000 * 1_000_000 // 10,000 $VBV
	listing, err := l.ListBondedAssetForSale(seller, assetID, price)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if listing.Status != BondedListingActive {
		t.Fatalf("a new listing must be active, got %q", listing.Status)
	}
	wantFee := price * uint64(BondedMarketFeeBps) / 10_000
	if wantFee != 250*1_000_000 {
		t.Fatalf("the fee for %d must be %d micro; got %d", price, 250*1_000_000, wantFee)
	}
	buyerBefore := l.playerBalances[buyer]

	sold, asset, err := l.BuyBondedListing(buyer, listing.ListingID)
	if err != nil {
		t.Fatalf("buy: %v", err)
	}
	if sold.Status != BondedListingSold || sold.BuyerWallet != buyer {
		t.Fatalf("listing must be SOLD to the buyer, got %+v", sold)
	}
	// MONEY: exact debit, exact credit, exact fee, and the split reconciles.
	if spent := buyerBefore - l.playerBalances[buyer]; spent != price {
		t.Fatalf("the buyer paid %d micro, want %d", spent, price)
	}
	net := price - wantFee
	if l.playerBalances[seller] != net {
		t.Fatalf("the seller received %d micro, want %d", l.playerBalances[seller], net)
	}
	if *faucet != wantFee {
		t.Fatalf("the sink received %d micro, want %d (a fee must never be burned silently)", *faucet, wantFee)
	}
	if l.playerBalances[buyer]+l.playerBalances[seller]+*faucet != buyerBefore {
		t.Fatalf("the trade must reconcile exactly: %d + %d + %d != %d",
			l.playerBalances[buyer], l.playerBalances[seller], *faucet, buyerBefore)
	}
	if sold.FeeMicro != wantFee || sold.NetMicro != net {
		t.Fatalf("the listing must record the ACTUAL amounts moved: fee=%d net=%d", sold.FeeMicro, sold.NetMicro)
	}
	// OWNERSHIP moved with the money.
	if asset == nil || asset.OwnerWallet != buyer || asset.HolderWallet != buyer {
		t.Fatalf("the asset must belong to the buyer (owner+holder), got %+v", asset)
	}

	// A SOLD listing is dead: a second buy refuses and moves nothing.
	bal, sel, faucetAfter := l.playerBalances[buyer], l.playerBalances[seller], *faucet
	if _, _, err := l.BuyBondedListing(buyer, listing.ListingID); err == nil {
		t.Fatal("a sold listing must not be buyable again")
	} else if !strings.Contains(err.Error(), "sold") {
		t.Fatalf("the refusal must name the status, got %q", err.Error())
	}
	if l.playerBalances[buyer] != bal || l.playerBalances[seller] != sel || *faucet != faucetAfter {
		t.Fatal("a refused second buy must move no money")
	}
}

// TestBondedMarketFeeIsExactIntegerArithmetic pins the split arithmetic, including the floor case
// where the remainder stays with the SELLER (nothing is lost and nothing is minted).
func TestBondedMarketFeeIsExactIntegerArithmetic(t *testing.T) {
	cases := []struct {
		price, fee uint64
	}{
		{1, 0},        // a 1-micro price cannot fund a 2.5% fee; the seller keeps the whole micro
		{399, 9},      // floor(399 * 250 / 10000) = 9
		{10_000, 250}, // exactly 2.5%
		{1_000_000, 25_000},
	}
	for _, c := range cases {
		if got := bondMarketFee(c.price); got != c.fee {
			t.Fatalf("fee of %d = %d, want %d", c.price, got, c.fee)
		}
		if bondMarketFee(c.price)+(c.price-bondMarketFee(c.price)) != c.price {
			t.Fatalf("the split of %d does not reconcile", c.price)
		}
	}
	if BondedMarketFeeBps != 250 {
		t.Fatalf("the served fee is %d bps; this test documents 250 (2.5%%)", BondedMarketFeeBps)
	}
}

// TestBondedMarketRefusalsMoveNothing walks the whole refusal set. Every case must leave balances,
// ownership and the sink EXACTLY as they were — a refused sale changes nothing.
func TestBondedMarketRefusalsMoveNothing(t *testing.T) {
	l, faucet := newTestLobbyForBonded(t)
	const seller, buyer, outsider = "seller", "buyer", "outsider"
	l.playerBalances[seller] = 0
	l.playerBalances[buyer] = 5_000 * 1_000_000
	l.playerBalances[outsider] = 5_000 * 1_000_000
	assetID := marketTestSellerAsset(t, l, seller)

	snapshot := func() (uint64, uint64, uint64, string) {
		a, _ := l.bondedAssets.Get(assetID)
		owner := ""
		if a != nil {
			owner = a.OwnerWallet
		}
		return l.playerBalances[buyer], l.playerBalances[outsider], *faucet, owner
	}
	expectUnchanged := func(label string, before ...interface{}) {
		t.Helper()
		b, o, f, owner := snapshot()
		if b != before[0].(uint64) || o != before[1].(uint64) || f != before[2].(uint64) || owner != before[3].(string) {
			t.Fatalf("%s moved state: buyer=%d outsider=%d sink=%d owner=%s", label, b, o, f, owner)
		}
	}

	// 1. LISTING refusals.
	b1, o1, f1, w1 := snapshot()
	if _, err := l.ListBondedAssetForSale(outsider, assetID, 1_000); err == nil {
		t.Fatal("a wallet must not list an asset it does not own")
	}
	expectUnchanged("listing another wallet's asset", b1, o1, f1, w1)
	if _, err := l.ListBondedAssetForSale(seller, assetID, 0); err == nil {
		t.Fatal("a zero price must be refused (a gift has its own route)")
	}
	expectUnchanged("listing at zero", b1, o1, f1, w1)
	if _, err := l.ListBondedAssetForSale(seller, assetID, MaxBondedListingMicro+1); err == nil {
		t.Fatal("a price above the ceiling must be refused")
	}
	expectUnchanged("listing above the ceiling", b1, o1, f1, w1)
	if _, err := l.ListBondedAssetForSale(seller, "BA-nonexistent", 1_000); err == nil {
		t.Fatal("listing an unknown asset must be refused")
	}
	expectUnchanged("listing an unknown asset", b1, o1, f1, w1)

	listing, err := l.ListBondedAssetForSale(seller, assetID, 1_000_000)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	b2, o2, f2, w2 := snapshot()
	if _, err := l.ListBondedAssetForSale(seller, assetID, 2_000_000); err == nil {
		t.Fatal("the same asset must not be offered twice at once")
	}
	expectUnchanged("double listing", b2, o2, f2, w2)

	// 2. BUY refusals.
	if _, _, err := l.BuyBondedListing(seller, listing.ListingID); err == nil {
		t.Fatal("a wallet must not buy its own listing")
	}
	expectUnchanged("buying own listing", b2, o2, f2, w2)
	if _, _, err := l.BuyBondedListing(buyer, "BML-nope"); err == nil {
		t.Fatal("an unknown listing must be refused")
	}
	expectUnchanged("buying an unknown listing", b2, o2, f2, w2)

	// A broke buyer is refused BY BALANCE, and nothing moves.
	l.playerBalances["broke"] = 10
	f3 := *faucet
	if _, _, err := l.BuyBondedListing("broke", listing.ListingID); err == nil {
		t.Fatal("an underfunded buyer must be refused")
	} else if !strings.Contains(err.Error(), "insufficient balance") {
		t.Fatalf("the refusal must state the balance problem, got %q", err.Error())
	}
	if *faucet != f3 {
		t.Fatal("a refused buy must not touch the sink")
	}
	if a, _ := l.bondedAssets.Get(assetID); a.OwnerWallet != seller {
		t.Fatal("a refused buy must not move ownership")
	}

	// 3. CANCEL refusals.
	if _, err := l.CancelBondedListing(outsider, listing.ListingID); err == nil {
		t.Fatal("only the seller may cancel")
	}
	expectUnchanged("cancelling as a non-seller", b2, o2, f2, w2)
	if _, err := l.CancelBondedListing(seller, "BML-nope"); err == nil {
		t.Fatal("cancelling an unknown listing must be refused")
	}
	expectUnchanged("cancelling an unknown listing", b2, o2, f2, w2)

	// 4. A cancelled listing is dead, and cancelling twice refuses.
	if _, err := l.CancelBondedListing(seller, listing.ListingID); err != nil {
		t.Fatalf("the seller must be able to cancel: %v", err)
	}
	b4, o4, f4, w4 := snapshot()
	if _, err := l.CancelBondedListing(seller, listing.ListingID); err == nil {
		t.Fatal("a cancelled listing must not be cancellable again")
	}
	expectUnchanged("double cancel", b4, o4, f4, w4)
	if _, _, err := l.BuyBondedListing(buyer, listing.ListingID); err == nil {
		t.Fatal("a cancelled listing must not be buyable")
	}
	expectUnchanged("buying a cancelled listing", b4, o4, f4, w4)
}

// TestBondedMarketListingCannotOutliveOwnership pins the §27.8 re-proof: a listing that changed hands
// or was burned is STALE (served with the reason) and is CANCELLED when a buyer tries it — even a
// record written by an older build, which is simulated here by writing an active listing directly.
func TestBondedMarketListingCannotOutliveOwnership(t *testing.T) {
	l, faucet := newTestLobbyForBonded(t)
	const seller, buyer = "seller", "buyer"
	l.playerBalances[buyer] = 10_000 * 1_000_000
	assetID := marketTestSellerAsset(t, l, seller)

	// A legacy/stale record: ACTIVE, but its seller is not the asset's owner.
	l.bondedAssets.putListingLocked(&BondedListing{
		ListingID: "BML-legacy", AssetID: assetID, SellerWallet: "ghost",
		PriceMicro: 1_000_000, Status: BondedListingActive, CreatedAt: time.Now(),
	})

	view := l.BondedMarketView(buyer)
	if view["stale_count"].(int) != 1 {
		t.Fatalf("a listing whose seller no longer owns the asset must be reported stale, got %v", view["stale_count"])
	}
	row := view["listings"].([]map[string]interface{})[0]
	if _, ok := row["stale_reason"]; !ok {
		t.Fatal("a stale listing must carry the reason in the payload")
	}

	bal, f := l.playerBalances[buyer], *faucet
	if _, _, err := l.BuyBondedListing(buyer, "BML-legacy"); err == nil {
		t.Fatal("a stale listing must not be buyable")
	} else if !strings.Contains(err.Error(), "no longer owns") {
		t.Fatalf("the refusal must state the ownership problem, got %q", err.Error())
	}
	if l.playerBalances[buyer] != bal || *faucet != f {
		t.Fatal("a refused stale buy must move no money")
	}
	// Buying it CANCELLED it, so it stops being offered.
	if after, _ := l.bondedAssets.Listing("BML-legacy"); after.Status != BondedListingCancelled {
		t.Fatalf("a stale listing a buyer tried must be cancelled, got %q", after.Status)
	}

	// A real transfer cancels the seller's active listing for that asset.
	listing, err := l.ListBondedAssetForSale(seller, assetID, 1_000_000)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if err := l.bondedAssets.TransferOwnership(assetID, seller, buyer); err != nil {
		t.Fatalf("transfer: %v", err)
	}
	after, _ := l.bondedAssets.Listing(listing.ListingID)
	if after.Status != BondedListingCancelled {
		t.Fatalf("a listing must not survive a transfer: got %q", after.Status)
	}
	if !strings.Contains(after.Note, "changed hands") {
		t.Fatalf("the cancellation must state the reason, got %q", after.Note)
	}
}

// TestBondedMarketPersistsAcrossRestart proves the listing storage is real: it survives Save/Load
// (exported + json-tagged), so a sale in progress is not lost by a restart.
func TestBondedMarketPersistsAcrossRestart(t *testing.T) {
	l, _ := newTestLobbyForBonded(t)
	const seller = "seller"
	assetID := marketTestSellerAsset(t, l, seller)
	listing, err := l.ListBondedAssetForSale(seller, assetID, 4_242_000)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if err := l.bondedAssets.Save(l); err != nil {
		t.Fatalf("save: %v", err)
	}
	// A restart: a brand-new registry loading the same data dir.
	l.bondedAssets = NewBondedAssetRegistry()
	if err := l.bondedAssets.Load(l); err != nil {
		t.Fatalf("load: %v", err)
	}
	back, ok := l.bondedAssets.Listing(listing.ListingID)
	if !ok {
		t.Fatal("the listing did not survive the restart (storage must be exported + json-tagged)")
	}
	if back.Status != BondedListingActive || back.PriceMicro != 4_242_000 || back.SellerWallet != seller {
		t.Fatalf("the restored listing is wrong: %+v", back)
	}
	if n := l.bondedAssets.activeListingsForAssetLocked(assetID); n != 1 {
		t.Fatalf("the restored market must still offer the asset once, got %d", n)
	}
	view := l.BondedMarketView(seller)
	if view["active_count"].(int) != 1 || view["mine_count"].(int) != 1 {
		t.Fatalf("the view after a restart is wrong: active=%v mine=%v", view["active_count"], view["mine_count"])
	}
}

// TestBondedMarketWritesThroughOnEveryMutation pins the market's durability promise: a list, a
// cancel and a SALE are each on disk when the call returns, WITHOUT anyone calling Save. The
// periodic persistence worker only runs every 15 minutes, so a restart inside that window used to
// resurrect the seller as the owner of art a buyer had already paid for.
func TestBondedMarketWritesThroughOnEveryMutation(t *testing.T) {
	l, _ := newTestLobbyForBonded(t)
	const seller, buyer = "seller", "buyer"
	l.playerBalances[buyer] = 100_000_000
	assetID := marketTestSellerAsset(t, l, seller)

	// The registry file must not exist before the first mutation, so the assertions below cannot be
	// satisfied by a file another test happened to leave behind (each test gets its own DataDir).
	file := l.getDataPath(bondedAssetsFile)
	if _, err := os.Stat(file); err == nil {
		t.Fatalf("the data dir already holds %s before any mutation", file)
	}

	listing, err := l.ListBondedAssetForSale(seller, assetID, 5_000_000)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	// A LISTING is user-visible state and is written through too.
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("a listing was not written through: %v", err)
	}

	if _, _, err := l.BuyBondedListing(buyer, listing.ListingID); err != nil {
		t.Fatalf("buy: %v", err)
	}
	// The SALE moved ownership and money; a fresh registry over the same dir must agree, with nobody
	// having called Save.
	l.bondedAssets = NewBondedAssetRegistry()
	if err := l.bondedAssets.Load(l); err != nil {
		t.Fatalf("load: %v", err)
	}
	asset, ok := l.bondedAssets.Get(assetID)
	if !ok {
		t.Fatal("the sold asset vanished from the written-through registry")
	}
	if !strings.EqualFold(asset.OwnerWallet, buyer) || !strings.EqualFold(asset.HolderWallet, buyer) {
		t.Fatalf("the written-through owner is %q/%q; want the buyer on both", asset.OwnerWallet, asset.HolderWallet)
	}
	back, ok := l.bondedAssets.Listing(listing.ListingID)
	if !ok || back.Status != BondedListingSold || !strings.EqualFold(back.BuyerWallet, buyer) {
		t.Fatalf("the sold listing did not survive write-through: %+v", back)
	}

	// A CANCELLATION writes through as well.
	second, err := l.ListBondedAssetForSale(buyer, assetID, 2_000_000)
	if err != nil {
		t.Fatalf("re-list: %v", err)
	}
	if _, err := l.CancelBondedListing(buyer, second.ListingID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	l.bondedAssets = NewBondedAssetRegistry()
	if err := l.bondedAssets.Load(l); err != nil {
		t.Fatalf("load after cancel: %v", err)
	}
	if got, ok := l.bondedAssets.Listing(second.ListingID); !ok || got.Status != BondedListingCancelled {
		t.Fatalf("the cancellation was not written through: %+v", got)
	}
}

// TestBondedMarketHTTPBoundaryCannotNameAPriceOrASeller proves the request bodies are narrow: unknown
// fields are refused by the decoder, so a client cannot supply a price, a seller or a status.
func TestBondedMarketHTTPBoundaryCannotNameAPriceOrASeller(t *testing.T) {
	l, _ := newTestLobbyForBonded(t)
	const seller, buyer = "seller", "buyer"
	l.playerBalances[buyer] = 10_000 * 1_000_000
	assetID := marketTestSellerAsset(t, l, seller)
	listing, err := l.ListBondedAssetForSale(seller, assetID, 1_000_000)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	post := func(handler http.HandlerFunc, wallet, body string) map[string]interface{} {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/api/assets/market?wallet="+wallet, strings.NewReader(body))
		rec := httptest.NewRecorder()
		handler(rec, req)
		out := map[string]interface{}{}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("handler did not answer JSON: %v", err)
		}
		return out
	}

	bal, owner := l.playerBalances[buyer], ""
	if a, _ := l.bondedAssets.Get(assetID); a != nil {
		owner = a.OwnerWallet
	}
	// A price in a BUY body cannot even be parsed.
	res := post(l.handleBondedMarketBuy, buyer, `{"listing_id":"`+listing.ListingID+`","price_micro":1}`)
	if ok, _ := res["success"].(bool); ok {
		t.Fatal("a buy body naming a price must be refused at the decoder")
	}
	if l.playerBalances[buyer] != bal {
		t.Fatal("a refused buy body must not move money")
	}
	// A seller in a LIST body cannot be parsed either.
	res = post(l.handleBondedMarketList, seller, `{"asset_id":"`+assetID+`","price_micro":2,"seller_wallet":"someone-else"}`)
	if ok, _ := res["success"].(bool); ok {
		t.Fatal("a listing body naming a seller must be refused at the decoder")
	}
	if a, _ := l.bondedAssets.Get(assetID); a.OwnerWallet != owner {
		t.Fatal("a refused listing body must not move ownership")
	}
	// A wrong METHOD on the read route is refused (a read must never write).
	req := httptest.NewRequest(http.MethodPost, "/api/assets/market", nil)
	rec := httptest.NewRecorder()
	l.handleBondedMarket(rec, req)
	if !strings.Contains(rec.Body.String(), "method not allowed") {
		t.Fatalf("POST to the market read route must be refused, got %s", rec.Body.String())
	}
	// The GET read carries the served rules.
	req = httptest.NewRequest(http.MethodGet, "/api/assets/market?wallet="+buyer, nil)
	rec = httptest.NewRecorder()
	l.handleBondedMarket(rec, req)
	body := rec.Body.String()
	for _, want := range []string{"fee_bps", "acquisition_rule", "min_price_micro", "max_price_micro", "capacity"} {
		if !strings.Contains(body, want) {
			t.Fatalf("the market read must serve %q, got %s", want, body)
		}
	}
}
