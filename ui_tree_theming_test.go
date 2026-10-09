package main

// ui_tree_theming_test.go
// §10.7 — proof for the UI-tree art assignment: every tree gets ONE frame of its own, the two
// pinned trees are Crypto-seraph, the assignment is deterministic, the client's own taxonomy is
// covered (so the two lists cannot drift), and a pool too small is REPORTED rather than hidden.

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func uiTreeByID(t *testing.T, m UiTreeArtMap, id string) UiTreeArt {
	t.Helper()
	for _, a := range m.Trees {
		if a.TreeID == id {
			return a
		}
	}
	t.Fatalf("no art assigned to tree %q", id)
	return UiTreeArt{}
}

// TestUiTreeArtIsUniqueAndPinsCryptoSeraph pins the rule Brendan asked for: one unique frame per UI
// tree, and Crypto-seraph in rewards and achievements.
func TestUiTreeArtIsUniqueAndPinsCryptoSeraph(t *testing.T) {
	m := UiTreeArtFor(nil)
	if m.Count == 0 || len(m.Trees) != m.Count {
		t.Fatalf("expected a non-empty assignment, got count=%d trees=%d", m.Count, len(m.Trees))
	}
	if !m.Unique {
		t.Fatalf("every tree must hold a frame of its own (pool=%d, count=%d, unique=%v)", m.Pool, m.Count, m.Unique)
	}
	seen := make(map[string]string, m.Count)
	for _, a := range m.Trees {
		if a.SKU == "" || a.URI == "" {
			t.Fatalf("tree %q has no art: %+v", a.TreeID, a)
		}
		if a.SourceURI == "" {
			t.Fatalf("tree %q must name the shipped frame it came from", a.TreeID)
		}
		if prev, dup := seen[a.SKU]; dup {
			t.Fatalf("frames must be unique: %q and %q both hold %s", prev, a.TreeID, a.SKU)
		}
		seen[a.SKU] = a.TreeID
		// The pool only ever contains frames the media policy accepts, so the art a panel paints
		// is always art the chest could hold too.
		if s, ok := placeholderSku(a.SKU); !ok || !s.Purchasable {
			t.Fatalf("tree %q was given frame %q, which the pack does not offer", a.TreeID, a.SKU)
		}
	}

	rewards := uiTreeByID(t, m, "rewards")
	if !rewards.Pinned || !strings.EqualFold(rewards.Character, "Crypto-seraph") {
		t.Fatalf("rewards must be Crypto-seraph and pinned, got character=%q pinned=%v", rewards.Character, rewards.Pinned)
	}
	achievements := uiTreeByID(t, m, "achievements")
	if !achievements.Pinned || !strings.EqualFold(achievements.Character, "Crypto-seraph") {
		t.Fatalf("achievements must be Crypto-seraph and pinned, got character=%q pinned=%v", achievements.Character, achievements.Pinned)
	}
	if rewards.SKU == achievements.SKU {
		t.Fatal("the two pinned trees must still hold DIFFERENT frames of that character")
	}

	// The pins are published, and the served character comes from the catalogue (never a
	// hard-coded word that could disagree with the art).
	pins := uiTreePinViews()
	if len(pins) != len(uiTreePins) {
		t.Fatalf("expected %d published pins, got %d", len(uiTreePins), len(pins))
	}
	for _, p := range pins {
		if !strings.EqualFold(p["character"], "Crypto-seraph") {
			t.Fatalf("a published pin names character %q", p["character"])
		}
	}
}

// TestUiTreeArtIsDeterministicAndAcceptsClientTrees proves the assignment is stable across calls (no
// RNG) and that a tree the server does not know yet can still be given art — appended after the
// canonical set, so a client can never re-order the canonical assignment.
func TestUiTreeArtIsDeterministicAndAcceptsClientTrees(t *testing.T) {
	first := UiTreeArtFor(nil)
	second := UiTreeArtFor(nil)
	if first.Count != second.Count {
		t.Fatalf("assignment is not deterministic: %d then %d trees", first.Count, second.Count)
	}
	for i := range first.Trees {
		if first.Trees[i] != second.Trees[i] {
			t.Fatalf("assignment differs at index %d: %+v vs %+v", i, first.Trees[i], second.Trees[i])
		}
	}

	extended := UiTreeArtFor([]string{"a_tree_the_server_has_never_heard_of"})
	if extended.Count != first.Count+1 {
		t.Fatalf("an extra tree must be added: %d -> %d", first.Count, extended.Count)
	}
	// Canonical order is untouched: the extra tree is LAST.
	for i := range first.Trees {
		if extended.Trees[i] != first.Trees[i] {
			t.Fatalf("a client-declared tree must not re-order the canonical assignment (index %d changed)", i)
		}
	}
	extra := extended.Trees[len(extended.Trees)-1]
	if extra.TreeID != "a_tree_the_server_has_never_heard_of" || extra.URI == "" {
		t.Fatalf("the extra tree must be appended with art of its own, got %+v", extra)
	}
	if !extended.Unique {
		t.Fatal("adding one tree must not break uniqueness while the pool has spare frames")
	}
	// A DUPLICATE extra id must not consume a second frame.
	dupe := UiTreeArtFor([]string{"rewards"})
	if dupe.Count != first.Count {
		t.Fatalf("a duplicate id must not add a tree: %d -> %d", first.Count, dupe.Count)
	}
}

// TestUiTreeArtReportsAnExhaustedPool proves the honest failure path: when the pack cannot give a
// tree its own frame, the tree is REPORTED as unserved — never silently handed a repeat, and never
// counted as part of a "unique" assignment.
func TestUiTreeArtReportsAnExhaustedPool(t *testing.T) {
	extras := make([]string, 0, 200)
	for i := 0; i < 200; i++ {
		extras = append(extras, "synthetic_tree_"+itoaTest(i))
	}
	m := UiTreeArtFor(extras)
	if len(m.Unknown) == 0 {
		t.Fatal("trees the pool cannot serve must be reported, not hidden behind a silent repeat")
	}
	for _, u := range m.Unknown {
		if !strings.Contains(u, "no unused frame left") {
			t.Fatalf("the refusal must state the reason, got %q", u)
		}
	}
	// Everything that WAS served is still unique (the refusal path never duplicates a frame). The
	// servable ceiling is the pool PLUS the pinned frames, because pins are taken out of the pool
	// and handed to their own tree.
	if !m.Unique {
		t.Fatal("every served tree must still hold a frame of its own")
	}
	maxServable := m.Pool + len(uiTreePins)
	if m.Count > maxServable {
		t.Fatalf("served %d trees from a pool of %d frames (+%d pinned)", m.Count, m.Pool, len(uiTreePins))
	}
	if m.Count != maxServable {
		t.Fatalf("an over-subscribed assignment should use every available frame: served %d of %d", m.Count, maxServable)
	}
}

func itoaTest(n int) string {
	if n == 0 {
		return "0"
	}
	out := ""
	for n > 0 {
		out = string(rune('0'+n%10)) + out
		n /= 10
	}
	return out
}

// TestUiTreeArtCoversTheDashboardsOwnTaxonomy is the DRIFT test: every category and feature the
// World Dashboard declares must be covered by the server's canonical tree list, so a new tab cannot
// silently ship without art.
func TestUiTreeArtCoversTheDashboardsOwnTaxonomy(t *testing.T) {
	raw, err := os.ReadFile("Public/js/world_dashboard.js")
	if err != nil {
		t.Skipf("the dashboard source is unavailable (%v); drift cannot be checked here", err)
	}
	src := string(raw)
	start := strings.Index(src, "const WD_CATEGORIES = [")
	if start < 0 {
		t.Skip("WD_CATEGORIES was not found; the dashboard taxonomy may have been renamed")
	}
	end := strings.Index(src[start:], "\n    ];")
	if end < 0 {
		t.Skip("the end of WD_CATEGORIES was not found")
	}
	block := src[start : start+end]

	known := make(map[string]bool, len(uiTreeCanonicalOrder))
	for _, id := range uiTreeCanonicalOrder {
		known[id] = true
	}
	catRe := regexp.MustCompile(`\bid:\s*'([a-z0-9_]+)'`)
	tabRe := regexp.MustCompile(`\btab:\s*'([a-z0-9_]+)'`)
	checked := 0
	declared := make(map[string]bool)
	for _, m := range catRe.FindAllStringSubmatch(block, -1) {
		checked++
		declared[m[1]] = true
		if !known[m[1]] {
			t.Fatalf("the dashboard declares the category %q, which the server's UI-tree list does not cover", m[1])
		}
	}
	for _, m := range tabRe.FindAllStringSubmatch(block, -1) {
		checked++
		declared[m[1]] = true
		if !known[m[1]] {
			t.Fatalf("the dashboard declares the feature %q, which the server's UI-tree list does not cover", m[1])
		}
	}
	if checked == 0 {
		t.Fatal("no dashboard tree id could be read, so drift was not actually checked")
	}
	// The reverse direction: a canonical id the dashboard does NOT declare is a stale server entry.
	// Reported rather than failed — the server list may legitimately run ahead of a staged tab — but
	// the count must be surfaced instead of assumed to be zero.
	stale := make([]string, 0)
	for _, id := range uiTreeCanonicalOrder {
		if !declared[id] {
			stale = append(stale, id)
		}
	}
	if len(stale) > 0 {
		t.Logf("server UI-tree entries not declared by the dashboard: %v", stale)
	}
}
