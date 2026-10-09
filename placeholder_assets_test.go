//go:build !js && !wasm

package main

import (
	"strings"
	"testing"
)

// TestPlaceholderCatalogue_IsBuiltFromTheShippedPack pins that the file list IS the SKU list: the
// catalogue is read from the served art, its ids and URIs are URL-safe, and every frame that is
// offered for sale satisfies the media policy it will be judged by.
func TestPlaceholderCatalogue_IsBuiltFromTheShippedPack(t *testing.T) {
	if err := PlaceholderCatalogueError(); err != nil {
		t.Fatalf("the pack could not be read: %v", err)
	}
	list := PlaceholderCatalogue()
	if len(list) < 100 {
		t.Fatalf("the catalogue has %d frames; the shipped NPC-helper pack has ~117", len(list))
	}
	prev := ""
	for _, s := range list {
		if s.SKU <= prev {
			t.Errorf("catalogue is not in ascending SKU order at %s", s.SKU)
		}
		prev = s.SKU
		if !strings.HasPrefix(s.SKU, "npc-") {
			t.Errorf("sku %q is not a pack id", s.SKU)
		}
		if !strings.HasPrefix(s.URI, placeholderSkuURIPrefix) {
			t.Errorf("sku %s uri %q must be a same-origin pack path", s.SKU, s.URI)
		}
		if strings.ContainsAny(s.URI, " ()?#") {
			t.Errorf("sku %s uri %q needs escaping: the pack must be URL-safe", s.SKU, s.URI)
		}
		if s.Bytes == 0 {
			t.Errorf("sku %s declares no byte size", s.SKU)
		}
		if s.Purchasable {
			media, err := placeholderSkuMedia(s)
			if err != nil {
				t.Errorf("purchasable sku %s could not build media: %v", s.SKU, err)
				continue
			}
			if err := ValidateBondedMedia(media); err != nil {
				t.Errorf("purchasable sku %s fails the very policy it is sold under: %v", s.SKU, err)
			}
			if len(media.HashHex) != 64 {
				t.Errorf("sku %s hash is not a sha256 hex (%d chars)", s.SKU, len(media.HashHex))
			}
		} else if s.Unavailable == "" {
			t.Errorf("sku %s is unavailable but states no reason", s.SKU)
		}
	}

	// The starter pack must be real, flagged, and reachable — otherwise the free path is a lie.
	for _, sku := range placeholderStarterSKUs {
		s, ok := placeholderSku(sku)
		if !ok {
			t.Fatalf("starter frame %s is missing from the shipped pack", sku)
		}
		if !s.Starter {
			t.Errorf("%s is in the starter list but not flagged starter", sku)
		}
		if !s.Purchasable {
			t.Errorf("starter frame %s cannot be granted: %s", sku, s.Unavailable)
		}
	}

	// A frame too large for the media policy is REPORTED, never silently sold. (If the art is ever
	// optimised this becomes a note rather than a failure — the point is the reporting, not the size.)
	if s, ok := placeholderSku("npc-crypto-seraph-001"); ok && !s.Purchasable {
		if !strings.Contains(s.Unavailable, "bytes") {
			t.Errorf("expected an oversize reason for %s, got %q", s.SKU, s.Unavailable)
		}
	}
}

// TestStarterGrant_IsFreeIdempotentAndUncapped pins the free tier: a wallet that holds NO cards and
// NO balance still receives every starter frame, once, and the grant never becomes a creation.
func TestStarterGrant_IsFreeIdempotentAndUncapped(t *testing.T) {
	l, faucet := newTestLobbyForBonded(t)
	l.bondedAssets = NewBondedAssetRegistry()
	const owner = "OWNER" // no cards, no $VBV

	res, err := l.EnsureStarterAssets(owner)
	if err != nil {
		t.Fatalf("EnsureStarterAssets: %v", err)
	}
	if len(res.Granted) != len(placeholderStarterSKUs) {
		t.Fatalf("granted %d starter frames, want %d (skipped: %v)", len(res.Granted), len(placeholderStarterSKUs), res.Skipped)
	}
	if len(res.Skipped) != 0 {
		t.Errorf("nothing should be skipped: %v", res.Skipped)
	}
	if l.playerBalances[owner] != 0 {
		t.Errorf("a grant must cost nothing, balance is %d", l.playerBalances[owner])
	}
	if *faucet != 0 {
		t.Errorf("a grant must route no fee, the sink received %d", *faucet)
	}
	for _, a := range res.Granted {
		if a.Source != PlaceholderSourceStarter || a.Sku == "" {
			t.Errorf("granted asset %s is not stamped as a starter frame (sku=%q source=%q)", a.AssetID, a.Sku, a.Source)
		}
		if !a.Certified {
			t.Errorf("granted asset %s must be certified (§27.7.3)", a.AssetID)
		}
		if err := ValidateBondedMedia(a.Media); err != nil {
			t.Errorf("granted asset %s holds policy-invalid media: %v", a.AssetID, err)
		}
	}

	// Idempotent: the assets themselves are the record, so a second call grants nothing new.
	again, err := l.EnsureStarterAssets(owner)
	if err != nil {
		t.Fatalf("second EnsureStarterAssets: %v", err)
	}
	if len(again.Granted) != 0 || len(again.AlreadyOwned) != len(placeholderStarterSKUs) {
		t.Errorf("second call granted %d / already %d; want 0 / %d",
			len(again.Granted), len(again.AlreadyOwned), len(placeholderStarterSKUs))
	}
	if n := l.bondedAssets.CountOwnedBy(owner); n != len(placeholderStarterSKUs) {
		t.Errorf("ownership is %d after two calls; want %d", n, len(placeholderStarterSKUs))
	}

	// The grant is ACQUISITION: capacity still reports limit 0 (no card/deck NFTs), and CREATION is
	// still refused — the free tier must not have quietly raised the creation cap.
	cap := l.BondedAssetCapacityForWallet(owner)
	if cap.Limit != 0 || cap.Used != len(placeholderStarterSKUs) {
		t.Errorf("capacity after a grant: used=%d limit=%d; want used=%d limit=0",
			cap.Used, cap.Limit, len(placeholderStarterSKUs))
	}
	if err := l.assertBondedAssetCapacity(owner); err == nil {
		t.Error("a wallet with no card/deck NFTs must still be unable to CREATE a bonded asset")
	}
}

// TestPlaceholderPurchase_ChargesTheServedPriceThroughTheSink pins the paid tier: the price is the
// server constant, it leaves the balance exactly, it lands in the sink, and a refusal moves no money.
func TestPlaceholderPurchase_ChargesTheServedPriceThroughTheSink(t *testing.T) {
	l, faucet := newTestLobbyForBonded(t)
	l.bondedAssets = NewBondedAssetRegistry()
	const owner = "OWNER"

	sku := ""
	for _, s := range PlaceholderCatalogue() {
		if s.Purchasable && !s.Starter {
			sku = s.SKU
			break
		}
	}
	if sku == "" {
		t.Skip("no purchasable non-starter frame in the shipped pack")
	}

	l.playerBalances[owner] = PlaceholderSkuPriceMicro
	beforeFaucet := *faucet
	a, err := l.BuyPlaceholderSku(owner, sku)
	if err != nil {
		t.Fatalf("BuyPlaceholderSku(%s): %v", sku, err)
	}
	if l.playerBalances[owner] != 0 {
		t.Errorf("balance is %d after buying; want the exact served price %d spent", l.playerBalances[owner], PlaceholderSkuPriceMicro)
	}
	if *faucet-beforeFaucet != PlaceholderSkuPriceMicro {
		t.Errorf("sink received %d; want %d (the fee must be routed, never burned)", *faucet-beforeFaucet, PlaceholderSkuPriceMicro)
	}
	if a.Source != PlaceholderSourceShop || a.Sku != sku {
		t.Errorf("bought asset stamp: sku=%q source=%q; want %q / %q", a.Sku, a.Source, sku, PlaceholderSourceShop)
	}
	if err := ValidateBondedMedia(a.Media); err != nil {
		t.Errorf("bought asset holds policy-invalid media: %v", err)
	}

	// An unknown frame is refused with nothing moved.
	l.playerBalances[owner] = 5 * PlaceholderSkuPriceMicro
	owned := l.bondedAssets.CountOwnedBy(owner)
	if _, err := l.BuyPlaceholderSku(owner, "npc-nobody-999"); err == nil {
		t.Error("an unknown frame must be refused")
	}
	if l.playerBalances[owner] != 5*PlaceholderSkuPriceMicro || l.bondedAssets.CountOwnedBy(owner) != owned {
		t.Error("an unknown frame must move no money and mint nothing")
	}

	// An unfunded purchase is refused before anything is written.
	l.playerBalances[owner] = PlaceholderSkuPriceMicro - 1
	if _, err := l.BuyPlaceholderSku(owner, sku); err == nil {
		t.Error("an unfunded purchase must be refused")
	}
	if l.bondedAssets.CountOwnedBy(owner) != owned {
		t.Error("an unfunded purchase must mint nothing")
	}

	// A frame the media policy cannot accept is refused, and the reason is the policy's.
	if s, ok := placeholderSku("npc-crypto-seraph-001"); ok && !s.Purchasable {
		l.playerBalances[owner] = 10 * PlaceholderSkuPriceMicro
		_, err := l.BuyPlaceholderSku(owner, "npc-crypto-seraph-001")
		if err == nil {
			t.Error("a frame that fails the media policy must be refused")
		} else if !strings.Contains(err.Error(), "bytes") {
			t.Errorf("the refusal should state the policy reason, got %v", err)
		}
		if l.playerBalances[owner] != 10*PlaceholderSkuPriceMicro {
			t.Error("a refused frame must move no money")
		}
	}
}

// TestBondedMediaPolicy_AcceptsSameOriginAssetsArt pins the ONE scheme-less form the policy admits
// (this process's own art) and that everything that could escape the origin is still refused.
func TestBondedMediaPolicy_AcceptsSameOriginAssetsArt(t *testing.T) {
	for _, u := range []string{
		"/Assets/Images/NPC-helpers/Anya/Anya-001.png",
		"/Assets/Images/NPC-helpers/Crypto-seraph/Crypto-seraph-004.png",
	} {
		m := validTestMedia()
		m.URI = u
		if err := ValidateBondedMedia(m); err != nil {
			t.Errorf("%s should be accepted as same-origin art: %v", u, err)
		}
	}
	for _, u := range []string{
		"/Assets/../secret.png", "/etc/passwd", "//evil.example/x.png",
		"/Assets/a b.png", "/Assets/a(1).png", "/Assets/x.png?y=1", "/Assets/x.png#frag",
		"data:image/png;base64,AAAA", "javascript:alert(1)", "file:///etc/passwd",
	} {
		m := validTestMedia()
		m.URI = u
		if err := ValidateBondedMedia(m); err == nil {
			t.Errorf("%s must be refused", u)
		}
	}
	// The served policy states the local form, so no client has to re-declare it.
	if lim := BondedMediaLimitsForClient(); lim.AllowedLocalPrefix != BondedMediaLocalPrefix {
		t.Errorf("served allowed_local_prefix = %q; want %q", lim.AllowedLocalPrefix, BondedMediaLocalPrefix)
	}
}
