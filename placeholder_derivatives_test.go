package main

// placeholder_derivatives_test.go
// §10.6 — proof for the light-art pass. Every claim the file header makes is asserted here with
// real bytes on disk: the filter is deterministic and integer-exact, the measured facts match the
// file that was written, a pass is idempotent until a source changes, a refusal is reported rather
// than papered over, and a derived rendition is a legal media URI under §10.3.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// ── helpers ──────────────────────────────────────────────────────────────────────────────────

func derivWritePNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		t.Fatalf("encode %s: %v", path, err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close %s: %v", path, err)
	}
}

func derivSolid(w, h int, c color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

// derivBuildPack writes a synthetic pack (unique character name per test, so the global SKU hash
// cache can never be shared with the real pack or another test) and returns the scanned SKUs.
func derivBuildPack(t *testing.T, character string, frames []int, w, h int, c color.RGBA) (string, []PlaceholderSku) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "pack")
	for _, fr := range frames {
		derivWritePNG(t, filepath.Join(root, character, fmt.Sprintf("%s-%03d.png", character, fr)), derivSolid(w, h, c))
	}
	skus, err := scanPlaceholderPack(root)
	if err != nil {
		t.Fatalf("scan of the synthetic pack failed: %v", err)
	}
	if len(skus) != len(frames) {
		t.Fatalf("expected %d SKUs from the synthetic pack, got %d", len(frames), len(skus))
	}
	return root, skus
}

func derivTarget(root string) placeholderDerivativeTarget {
	out := filepath.Join(filepath.Dir(root), "out")
	return placeholderDerivativeTarget{
		ArtRoot:      root,
		OutRoot:      out,
		URIPrefix:    "/Assets/Generated/test-pack/",
		ManifestPath: filepath.Join(out, placeholderDerivativeManifestFile),
	}
}

func derivFileSHA(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// ── the filter ───────────────────────────────────────────────────────────────────────────────

// TestIntegerBoxDownscaleIsExactAndAlphaWeighted pins the two properties the pass depends on: the
// integer box average is exact and reproducible, and a transparent pixel does NOT darken the
// colour around it (which is what an un-weighted average would do).
func TestIntegerBoxDownscaleIsExactAndAlphaWeighted(t *testing.T) {
	// A solid 4x4 → 2x2 must stay exactly the same colour and full alpha.
	solid := integerBoxDownscale(derivSolid(4, 4, color.RGBA{R: 128, G: 64, B: 32, A: 255}), 2)
	if solid == nil {
		t.Fatal("a 4x4 image at 2 px wide should downscale, not passthrough")
	}
	if got := solid.Bounds(); got.Dx() != 2 || got.Dy() != 2 {
		t.Fatalf("expected a 2x2 result, got %dx%d", got.Dx(), got.Dy())
	}
	r, g, b, a := solid.At(0, 0).RGBA()
	if r>>8 != 128 || g>>8 != 64 || b>>8 != 32 || a>>8 != 255 {
		t.Fatalf("a solid block must average to itself: got %d,%d,%d,%d", r>>8, g>>8, b>>8, a>>8)
	}

	// 3 opaque white pixels + 1 fully transparent black: the coverage-weighted average keeps the
	// colour WHITE (255) and reduces only the alpha. An un-weighted average would give a grey
	// ~191 colour, i.e. the dark halo this filter exists to avoid.
	mixed := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			col := color.RGBA{R: 255, G: 255, B: 255, A: 255}
			if x == 0 && y == 0 {
				col = color.RGBA{A: 0}
			}
			mixed.Set(x, y, col)
		}
	}
	out := integerBoxDownscale(mixed, 2)
	if out == nil {
		t.Fatal("expected a downscale of the mixed image")
	}
	r2, g2, b2, a2 := out.At(0, 0).RGBA()
	if r2>>8 != 255 || g2>>8 != 255 || b2>>8 != 255 {
		t.Fatalf("transparent pixels must not darken the colour (halo guard): got %d,%d,%d", r2>>8, g2>>8, b2>>8)
	}
	if a2>>8 != 191 {
		t.Fatalf("expected the average alpha of 3/4 coverage to be 191, got %d", a2>>8)
	}

	// Determinism: the SAME input must produce the SAME bytes on a second call.
	again := integerBoxDownscale(mixed, 2)
	if !bytesEqual(out, again) {
		t.Fatal("the box filter is not deterministic for identical input")
	}

	// No downscale needed → nil, and the caller uses the source (never a duplicate file).
	if integerBoxDownscale(derivSolid(64, 64, color.RGBA{A: 255}), 512) != nil {
		t.Fatal("an image at or below the tier width must not be resampled")
	}
}

func bytesEqual(a, b image.Image) bool {
	ab, bb := a.Bounds(), b.Bounds()
	if ab != bb {
		return false
	}
	for y := ab.Min.Y; y < ab.Max.Y; y++ {
		for x := ab.Min.X; x < ab.Max.X; x++ {
			if a.At(x, y) != b.At(x, y) {
				return false
			}
		}
	}
	return true
}

// ── derivation ───────────────────────────────────────────────────────────────────────────────

// TestDerivePNGToWidthMeasuresWhatItWrote proves the facts a payload carries are MEASURED: the
// byte size and sha256 reported for a rendition are the real ones of the file on disk, the pixel
// size is the encoded image's, and the aspect ratio is preserved by integer math.
func TestDerivePNGToWidthMeasuresWhatItWrote(t *testing.T) {
	root, skus := derivBuildPack(t, "DerivA", []int{1}, 1024, 512, color.RGBA{R: 10, G: 200, B: 90, A: 255})
	_ = root
	s := skus[0]
	out := filepath.Join(t.TempDir(), "out", "DerivA-001-slide.png")

	d, err := derivePNGToWidth(s.path, out, "/Assets/Generated/test-pack/DerivA-001-slide.png",
		PlaceholderDerivativeSlideTier, 512, s.URI)
	if err != nil {
		t.Fatalf("derivation failed: %v", err)
	}
	if d.Passthrough {
		t.Fatal("a 1024 px wide source at 512 px must actually be derived, not passed through")
	}
	if d.Width != 512 || d.Height != 256 {
		t.Fatalf("expected a 512x256 rendition (aspect preserved by integer math), got %dx%d", d.Width, d.Height)
	}
	if d.Bytes != uint64(derivStatSize(t, out)) {
		t.Fatalf("declared bytes %d do not match the file on disk (%d)", d.Bytes, derivStatSize(t, out))
	}
	if d.HashHex != derivFileSHA(t, out) {
		t.Fatal("declared hash_hex is not the sha256 of the bytes that were written")
	}
	if d.URI != "/Assets/Generated/test-pack/DerivA-001-slide.png" {
		t.Fatalf("unexpected URI %q", d.URI)
	}
	// The written file really is a PNG at the declared size.
	f, err := os.Open(out)
	if err != nil {
		t.Fatalf("open rendition: %v", err)
	}
	img, err := png.Decode(f)
	_ = f.Close()
	if err != nil {
		t.Fatalf("the rendition is not a decodable PNG: %v", err)
	}
	if b := img.Bounds(); uint64(b.Dx()) != d.Width || uint64(b.Dy()) != d.Height {
		t.Fatalf("decoded %dx%d but declared %dx%d", b.Dx(), b.Dy(), d.Width, d.Height)
	}
	// DERIVED ART IS SMALLER — the entire point of the pass. A rendition that is not smaller would
	// be a lie about the saving, so it is asserted rather than assumed.
	if d.Bytes >= s.Bytes {
		t.Fatalf("the rendition (%d bytes) is not smaller than the original (%d bytes)", d.Bytes, s.Bytes)
	}

	// A source already at or below the tier width is a PASSTHROUGH naming the original: no second
	// file, and the facts come from the source itself.
	noDup := filepath.Join(t.TempDir(), "out", "DerivA-001-thumb-would-be.png")
	pass, err := derivePNGToWidth(s.path, noDup, "urn:should-not-be-used", PlaceholderDerivativeThumbTier, 4096, s.URI)
	if err != nil {
		t.Fatalf("passthrough derivation failed: %v", err)
	}
	if !pass.Passthrough || pass.URI != s.URI {
		t.Fatalf("expected a passthrough naming the source, got passthrough=%v uri=%q", pass.Passthrough, pass.URI)
	}
	if _, statErr := os.Stat(noDup); statErr == nil {
		t.Fatal("a passthrough must not write a duplicate file")
	}
	if pass.HashHex != derivFileSHA(t, s.path) {
		t.Fatal("a passthrough must carry the SOURCE's real hash")
	}
}

// ── the pass ─────────────────────────────────────────────────────────────────────────────────

// TestDerivativePassIsIdempotentAndReDerivesChangedSources proves the pass writes each rendition
// once, does NOTHING on a second run, and re-derives a frame whose source bytes changed (so a
// manifest can never serve stale art).
func TestDerivativePassIsIdempotentAndReDerivesChangedSources(t *testing.T) {
	root, skus := derivBuildPack(t, "DerivB", []int{1, 2}, 900, 450, color.RGBA{R: 20, G: 30, B: 40, A: 255})
	tgt := derivTarget(root)

	first := runPlaceholderDerivativePass(tgt, skus, nil)
	if first.Failed != 0 {
		t.Fatalf("pass reported %d failures: %v", first.Failed, first.Errors)
	}
	if first.Derived != len(skus)*2 {
		t.Fatalf("expected %d renditions on the first pass, got %d", len(skus)*2, first.Derived)
	}
	slidePath, ok := tgt.pathForURI(tgt.uriFor(skus[0], PlaceholderDerivativeSlideTier))
	if !ok {
		t.Fatal("the target could not resolve its own rendition URI back to a path")
	}
	if _, err := os.Stat(slidePath); err != nil {
		t.Fatalf("the pass reported a rendition it did not write: %v", err)
	}
	if _, err := os.Stat(tgt.ManifestPath); err != nil {
		t.Fatalf("the pass did not commit its manifest: %v", err)
	}
	firstHash := derivFileSHA(t, slidePath)

	// The pass mutates the PROCESS-WIDE index for this root; clear it so the second run really
	// re-reads the manifest from disk (otherwise the test would only prove the cache works).
	forgetDerivativeManifestForTest(tgt.OutRoot)
	second := runPlaceholderDerivativePass(tgt, skus, nil)
	if second.Derived != 0 || second.Reused != len(skus) {
		t.Fatalf("a second pass must reuse every frame: derived=%d reused=%d", second.Derived, second.Reused)
	}
	if derivFileSHA(t, slidePath) != firstHash {
		t.Fatal("a reused frame must not be rewritten (its bytes changed)")
	}

	// CHANGE A SOURCE: the pass must notice by hash and re-derive rather than serve stale art.
	derivWritePNG(t, skus[0].path, derivSolid(600, 600, color.RGBA{R: 250, G: 20, B: 20, A: 255}))
	rescanned, err := scanPlaceholderPack(root)
	if err != nil {
		t.Fatalf("rescan: %v", err)
	}
	forgetDerivativeManifestForTest(tgt.OutRoot)
	third := runPlaceholderDerivativePass(tgt, rescanned, nil)
	if third.Derived == 0 {
		t.Fatal("a changed source must be re-derived, not reused")
	}
	if derivFileSHA(t, slidePath) == firstHash {
		t.Fatal("the rendition was reported as re-derived but its bytes are unchanged")
	}
}

func derivStatSize(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return info.Size()
}

// forgetDerivativeManifestForTest drops the process-wide index for a root, so a test can prove the
// pass re-reads what it committed to disk instead of trusting its own cache.
func forgetDerivativeManifestForTest(root string) {
	placeholderDerivManifestMu.Lock()
	delete(placeholderDerivManifests, root)
	placeholderDerivManifestMu.Unlock()
}

// TestDerivativePassReportsRefusalWithoutInventingArt feeds the pass one real frame and one file
// that is not a PNG: the refusal must be REPORTED with a reason, no rendition may be invented for
// it, and the healthy frame must still be derived.
func TestDerivativePassReportsRefusalWithoutInventingArt(t *testing.T) {
	root := filepath.Join(t.TempDir(), "pack")
	derivWritePNG(t, filepath.Join(root, "DerivC", "DerivC-001.png"), derivSolid(800, 400, color.RGBA{A: 255}))
	if err := os.WriteFile(filepath.Join(root, "DerivC", "DerivC-002.png"), []byte("this is not a PNG at all"), 0644); err != nil {
		t.Fatalf("write the junk frame: %v", err)
	}
	skus, err := scanPlaceholderPack(root)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(skus) != 2 {
		t.Fatalf("both frames must be catalogued (one is merely unavailable), got %d", len(skus))
	}
	tgt := placeholderDerivativeTarget{
		ArtRoot: root, OutRoot: filepath.Join(t.TempDir(), "out"),
		URIPrefix: "/Assets/Generated/test-pack/",
	}
	tgt.ManifestPath = filepath.Join(tgt.OutRoot, placeholderDerivativeManifestFile)

	res := runPlaceholderDerivativePass(tgt, skus, nil)
	if res.Failed != 1 {
		t.Fatalf("expected exactly one reported failure, got %d (%v)", res.Failed, res.Errors)
	}
	if len(res.Errors) == 0 {
		t.Fatal("a refusal must carry a reason")
	}
	if res.Derived != 2 {
		t.Fatalf("the healthy frame must still be derived (2 renditions), got %d", res.Derived)
	}
	for _, tier := range []string{PlaceholderDerivativeSlideTier, PlaceholderDerivativeThumbTier} {
		if _, statErr := os.Stat(tgt.outPathFor(skus[1], tier)); statErr == nil {
			t.Fatalf("a refused frame must not have a %s rendition", tier)
		}
	}
	m := loadDerivativeManifest(tgt)
	rec := m.Records[skus[1].SKU]
	if rec == nil || rec.Failed == "" {
		t.Fatal("the refused frame must be recorded with its reason")
	}
	if rec.Slide != nil || rec.Thumb != nil {
		t.Fatal("a refused frame must not claim a rendition")
	}
	forgetDerivativeManifestForTest(tgt.OutRoot)
}

// TestDerivativePriorityServesTheSlideshowFramesFirst pins the ordering rule: the frames the slides
// show are derived before the rest of the pack, and everything else keeps catalogue order.
func TestDerivativePriorityServesTheSlideshowFramesFirst(t *testing.T) {
	skus := []PlaceholderSku{
		{SKU: "npc-a-001"}, {SKU: "npc-b-002"}, {SKU: "npc-c-003"}, {SKU: "npc-d-004"},
	}
	got := orderSkusForDerivation(skus, []string{"npc-c-003", "npc-a-001"})
	want := []string{"npc-c-003", "npc-a-001", "npc-b-002", "npc-d-004"}
	for i := range want {
		if got[i].SKU != want[i] {
			t.Fatalf("derivation order = %v, want %v", derivSkuIDs(got), want)
		}
	}
	// The priority list comes from the slide declaration, so it cannot drift from what the app's
	// own slideshows display.
	prio := placeholderDerivativePriority()
	if len(prio) == 0 {
		t.Fatal("the priority list must not be empty")
	}
	for _, sku := range placeholderStarterSKUs {
		found := false
		for _, p := range prio {
			if p == sku {
				found = true
			}
		}
		if !found {
			t.Fatalf("every starter frame must be derived early; %s is missing", sku)
		}
	}
}

func derivSkuIDs(skus []PlaceholderSku) []string {
	out := make([]string, 0, len(skus))
	for _, s := range skus {
		out = append(out, s.SKU)
	}
	return out
}

// TestDerivedRenditionIsALegalMediaURI proves a derived rendition passes the SAME §10.3 media
// policy every user-declared payload passes — so painting light art is never a policy exception.
func TestDerivedRenditionIsALegalMediaURI(t *testing.T) {
	root, skus := derivBuildPack(t, "DerivD", []int{7}, 1200, 600, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	tgt := derivTarget(root)
	res := runPlaceholderDerivativePass(tgt, skus, nil)
	if res.Failed != 0 || res.Derived == 0 {
		t.Fatalf("pass failed: %+v", res)
	}
	rec, ok := derivativeRecordForTarget(tgt, skus[0].SKU)
	if !ok || rec.Slide == nil || rec.Thumb == nil {
		t.Fatal("expected both renditions to be recorded")
	}
	for _, d := range []*PlaceholderDerivative{rec.Slide, rec.Thumb} {
		media := &BondedMedia{
			URI: d.URI, MimeType: d.MimeType, HashHex: d.HashHex,
			Bytes: d.Bytes, Width: d.Width, Height: d.Height,
		}
		if err := ValidateBondedMedia(media); err != nil {
			t.Fatalf("a derived rendition must satisfy the media policy: %v", err)
		}
	}
	if rec.Thumb.Width != PlaceholderDerivativeThumbWidth {
		t.Fatalf("the thumb tier must be %d px wide, got %d", PlaceholderDerivativeThumbWidth, rec.Thumb.Width)
	}
	if rec.Slide.Width != PlaceholderDerivativeSlideWidth {
		t.Fatalf("the slide tier must be %d px wide, got %d", PlaceholderDerivativeSlideWidth, rec.Slide.Width)
	}
	if rec.SourceHashHex != derivFileSHA(t, skus[0].path) {
		t.Fatal("the record must name the SOURCE hash it was derived from")
	}
	forgetDerivativeManifestForTest(tgt.OutRoot)
}
