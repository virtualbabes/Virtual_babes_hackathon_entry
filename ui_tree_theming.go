//go:build !js && !wasm

package main

// ui_tree_theming.go
// §10.7 — ONE UNIQUE PLACEHOLDER FRAME PER UI TREE (the World Dashboard's sub-UI trees).
//
// Brendan's rule: "they would need at least 1 unique image from the placeholders for each, crypto
// seraph should be in rewards and achievements".
//
// So every UI tree — each World Dashboard CATEGORY and each FEATURE inside it — is assigned one pack
// frame of its own, so a sub-UI tree is visually distinguishable from its siblings instead of every
// panel wearing the same picture. Two frames are PINNED by name: `rewards` and `achievements` are
// Crypto-seraph, because that is the character the rule asks for there.
//
// WHAT IS GUARANTEED, AND HOW:
//   - UNIQUENESS IS ARITHMETIC, NOT HOPE. The pack holds 117 frames and the dashboard declares 78
//     tree ids (11 categories + 67 features), of which `governance`, `faith` and `creator` are BOTH
//     a category and a feature — so the served assignment is 75 distinct trees. A collision-free
//     assignment is possible; the pool is ordered deterministically
//     (round-robin across the three characters, so neighbouring trees differ in character too) and
//     each tree takes the next unused frame. `unique` is COMPUTED from what was handed out, and a
//     pool too small to cover the tree count is REPORTED rather than silently repeating.
//   - THE ASSIGNMENT IS SERVER-OWNED AND DETERMINISTIC. Same pack + same tree list ⇒ same map, on
//     every machine, with no RNG. A client may name EXTRA trees it has (the dashboard owns its own
//     taxonomy) and those are appended after the canonical set; it can never re-order the canonical
//     assignment.
//   - THE ART IS THE LIGHT RENDITION WHEN IT EXISTS (placeholder_derivatives.go), with the shipped
//     frame named beside it, so a panel background does not drag a multi-megabyte file over the
//     wire. `derived:false` says the rendition is not ready yet.
//
// This is the app's OWN chrome art, not a viewer preference: it is not keyed by wallet, it is not a
// bonding target, and it never writes `registry.Bindings` — a player's own art reaches these screens
// through the slides / card-view layers, which have their own (viewer-scoped) records.

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// uiTreePin declares the trees whose character is fixed by the rule (currently the Crypto-seraph
// pair). A pinned tree's frame is removed from the pool first, so no other tree can take it.
type uiTreePin struct {
	TreeID string
	SKU    string
}

var uiTreePins = []uiTreePin{
	{TreeID: "rewards", SKU: "npc-crypto-seraph-003"},
	{TreeID: "achievements", SKU: "npc-crypto-seraph-004"},
}

// uiTreeCanonicalOrder is the World Dashboard's tree list as the SERVER knows it: every category id
// followed by that category's feature ids, in the order the dashboard declares them. It exists so
// the assignment is a server guarantee; a client that has grown a tree the server does not know
// names it in `ids=` and receives an assignment for it too (appended, canonical order untouched).
var uiTreeCanonicalOrder = []string{
	// Tier 1 — categories
	"player", "careers", "assets", "economy", "governance", "play", "world", "faith", "competition", "creator", "system",
	// Player Hub
	"portfolio", "theme", "identity", "stats", "achievements", "deck", "card_titles", "card_progression", "mood", "wagers",
	// Careers & Factions
	"career", "criminality", "justice", "contracts", "counterfeit", "faction",
	// Assets
	"items", "equipment", "pets", "vehicles", "citizens", "children", "branding", "pet_arena", "vehicle_arena", "clubs", "orphan",
	// Economy & Trade
	"markets", "dividends", "loans", "blackmarket", "bridge", "ads",
	// Governance
	"governance", "governor", "territory", "regions", "compliance", "leaderboard",
	// Play & Board
	"game", "create_match", "match", "multiplayer", "game_modes", "campaign", "tutorial", "locations", "npc_taunts", "tea_house", "zen_garden",
	// World & Events
	"season", "events", "tournament", "replay", "treasure", "worldcontent", "industrial",
	// Faith & Church
	"faith", "religion", "church",
	// Competition
	"rivalry", "rewards",
	// Creator Economy
	"creator", "launches",
	// System & Ops
	"infrastructure", "gamingos", "devhub", "localmodel", "maintenance", "systemmsg", "report", "admin",
	// The five operations consoles (community/extended/security/system/utilities). Each was composed
	// but rendered nothing — its root container had no markup, so init() returned on the null-check.
	// They became reachable when the roots were self-mounted and the dashboard gained their leaves.
	"community", "exconsole", "secconsole", "sysconsole", "utconsole",
}

// UiTreeArt is one tree's assigned frame.
type UiTreeArt struct {
	TreeID string `json:"tree_id"`
	SKU    string `json:"sku"`
	// URI is what to paint: the LIGHT thumb rendition when one exists, otherwise the shipped frame.
	URI string `json:"uri"`
	// SourceURI is the shipped frame this art belongs to, so a light copy is never mistaken for a
	// different picture.
	SourceURI string `json:"source_uri"`
	Derived   bool   `json:"derived"`
	// Character is the NPC helper the frame belongs to (the rule names characters, so the answer to
	// "which character is in rewards?" is IN the payload).
	Character string `json:"character"`
	Frame     int    `json:"frame"`
	// Pinned marks a tree whose character the rule fixes.
	Pinned bool `json:"pinned,omitempty"`
	// SharedBy is >1 only when the pool was too small, and then the tree is reported as sharing.
	SharedBy int `json:"shared_by,omitempty"`
}

// UiTreeArtMap is the whole served assignment.
type UiTreeArtMap struct {
	Trees   []UiTreeArt `json:"trees"`
	Count   int         `json:"count"`
	Pool    int         `json:"pool"`
	Unique  bool        `json:"unique"`
	Unknown []string    `json:"unknown_ids,omitempty"`
	Rule    string      `json:"rule"`
}

// uiTreePoolOrder returns the frame order trees draw from: one frame of each character in turn, so
// neighbouring trees differ in character as well as in frame. Pinned frames are removed first (they
// are given to their tree by name), and a frame the media policy refuses is never handed out.
func uiTreePoolOrder(skus []PlaceholderSku) []PlaceholderSku {
	taken := make(map[string]bool, len(uiTreePins))
	for _, p := range uiTreePins {
		taken[p.SKU] = true
	}
	byCharacter := make(map[string][]PlaceholderSku)
	characters := make([]string, 0, 4)
	for _, s := range skus {
		if taken[s.SKU] || !s.Purchasable {
			continue
		}
		if _, seen := byCharacter[s.Character]; !seen {
			characters = append(characters, s.Character)
		}
		byCharacter[s.Character] = append(byCharacter[s.Character], s)
	}
	sort.Strings(characters)
	out := make([]PlaceholderSku, 0, len(skus))
	for {
		progressed := false
		for _, c := range characters {
			list := byCharacter[c]
			if len(list) == 0 {
				continue
			}
			out = append(out, list[0])
			byCharacter[c] = list[1:]
			progressed = true
		}
		if !progressed {
			break
		}
	}
	return out
}

// uiTreeArtFromSku joins a tree with the frame it was given, preferring the LIGHT rendition.
func uiTreeArtFromSku(treeID string, s PlaceholderSku, pinned bool) UiTreeArt {
	a := UiTreeArt{
		TreeID: treeID, SKU: s.SKU, URI: s.URI, SourceURI: s.URI,
		Character: s.Character, Frame: s.Frame, Pinned: pinned,
	}
	if light, derived := PlaceholderDerivativeURIForSKU(s.SKU, PlaceholderDerivativeThumbTier); derived {
		a.URI = light
		a.Derived = true
	}
	return a
}

// UiTreeArtFor builds the assignment. `extraIDs` are client-declared trees (in the order given),
// appended after the canonical set; anything the pool cannot cover individually is reported.
func UiTreeArtFor(extraIDs []string) UiTreeArtMap {
	skus := PlaceholderCatalogue()
	index := make(map[string]PlaceholderSku, len(skus))
	for _, s := range skus {
		index[s.SKU] = s
	}
	order := make([]string, 0, len(uiTreeCanonicalOrder)+len(extraIDs))
	seen := make(map[string]bool, len(uiTreeCanonicalOrder)+len(extraIDs))
	for _, id := range uiTreeCanonicalOrder {
		if !seen[id] {
			seen[id] = true
			order = append(order, id)
		}
	}
	pool := uiTreePoolOrder(skus)
	out := UiTreeArtMap{
		Trees: make([]UiTreeArt, 0, len(order)),
		Pool:  len(pool),
		Rule: "Every World Dashboard tree (category and feature) is assigned one placeholder frame " +
			"of its own, so a sub-UI tree is visually distinguishable from its siblings. The " +
			"assignment is deterministic and server-owned; `rewards` and `achievements` are pinned " +
			"to Crypto-seraph. A panel paints the light thumb rendition when the server has derived " +
			"it and the shipped frame otherwise. This is the app's own chrome art: it is not " +
			"wallet-keyed and it never writes a branding binding.",
	}
	assigned := make(map[string]UiTreeArt, len(order))
	next := 0
	for _, id := range order {
		var s PlaceholderSku
		ok := false
		pinned := false
		for _, p := range uiTreePins {
			if p.TreeID == id {
				s, ok = index[p.SKU]
				pinned = true
				break
			}
		}
		if !ok && !pinned {
			if next < len(pool) {
				s, ok = pool[next], true
				next++
			}
		}
		if !ok {
			// The pool cannot honour the rule here. The tree still renders, and it SAYS it shares
			// another tree's frame instead of pretending the assignment is unique.
			if len(pool) == 0 {
				out.Unknown = append(out.Unknown, id+": the pack has no frame this policy accepts")
				continue
			}
			shared := pool[len(pool)-1]
			a := uiTreeArtFromSku(id, shared, false)
			a.SharedBy = 2
			assigned[id] = a
			continue
		}
		assigned[id] = uiTreeArtFromSku(id, s, pinned)
	}
	// Append client-declared trees the server did not know, so a grown dashboard still gets art.
	for _, raw := range extraIDs {
		key := strings.ToLower(strings.TrimSpace(raw))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		if next >= len(pool) {
			out.Unknown = append(out.Unknown, key+": the pack has no unused frame left for it")
			continue
		}
		assigned[key] = uiTreeArtFromSku(key, pool[next], false)
		next++
		order = append(order, key)
	}
	for _, id := range order {
		if a, ok := assigned[id]; ok {
			out.Trees = append(out.Trees, a)
		}
	}
	out.Count = len(out.Trees)
	// UNIQUENESS IS COMPUTED from what was actually handed out, never asserted.
	used := make(map[string]int, out.Count)
	for _, a := range out.Trees {
		used[a.SKU]++
	}
	out.Unique = true
	for _, n := range used {
		if n > 1 {
			out.Unique = false
		}
	}
	return out
}

// ── HTTP ─────────────────────────────────────────────────────────────────────────────────────

// handleUiTreeArt serves the assignment. It is PUBLIC and wallet-independent: this is the app's own
// chrome, so no ownership check applies and no player state is read. `ids=` lets the dashboard name
// trees the server does not know yet (its taxonomy is its own), appended after the canonical set.
func (l *Lobby) handleUiTreeArt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, map[string]interface{}{"success": false, "error": "method not allowed"})
		return
	}
	extra := make([]string, 0)
	if raw := strings.TrimSpace(r.URL.Query().Get("ids")); raw != "" {
		for _, id := range strings.Split(raw, ",") {
			if v := strings.TrimSpace(id); v != "" {
				extra = append(extra, v)
			}
		}
	}
	m := UiTreeArtFor(extra)
	if err := uiTreeArtAsserted(); err != nil {
		writeJSON(w, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
			"trees":   m.Trees,
			"count":   m.Count,
			"pool":    m.Pool,
		})
		return
	}
	writeJSON(w, map[string]interface{}{
		"success":      true,
		"trees":        m.Trees,
		"count":        m.Count,
		"unique":       m.Unique,
		"pool":         m.Pool,
		"unknown_ids":  m.Unknown,
		"pinned_trees": uiTreePinViews(),
		"derivatives":  PlaceholderDerivativeStatusForClient(),
		"rule":         m.Rule,
	})
}

// uiTreePinViews lists the pinned trees (so a client can assert the rule without re-declaring which
// character belongs where).
func uiTreePinViews() []map[string]string {
	out := make([]map[string]string, 0, len(uiTreePins))
	for _, p := range uiTreePins {
		out = append(out, map[string]string{"tree_id": p.TreeID, "sku": p.SKU, "character": pinCharacterName(p.SKU)})
	}
	return out
}

// pinCharacterName answers the character a pinned frame belongs to, from the catalogue (never from
// a hard-coded word, so the served answer cannot disagree with the art).
func pinCharacterName(sku string) string {
	if s, ok := placeholderSku(sku); ok {
		return s.Character
	}
	return ""
}

// uiTreeArtAsserted fails LOUDLY when the declared pins can never be honoured — an empty pack or a
// missing pinned frame would otherwise be served as an assignment that is quietly wrong.
func uiTreeArtAsserted() error {
	idx := placeholderCatalogueIndex()
	if len(idx) == 0 {
		return fmt.Errorf("the placeholder pack is empty, so no UI tree can be given art")
	}
	for _, p := range uiTreePins {
		s, ok := idx[p.SKU]
		if !ok {
			return fmt.Errorf("pinned UI tree %q names frame %q, which is not in the shipped pack", p.TreeID, p.SKU)
		}
		// A pinned frame must also be one the media policy ACCEPTS: the pool only ever hands out
		// frames the chest could hold, and a pin must not be the exception that quietly hands out
		// art the catalogue refuses.
		if !s.Purchasable {
			return fmt.Errorf("pinned UI tree %q names frame %q, which the media policy refuses: %s", p.TreeID, p.SKU, s.Unavailable)
		}
	}
	if len(uiTreePoolOrder(PlaceholderCatalogue())) == 0 {
		return fmt.Errorf("no purchasable frame is available for the UI tree pool")
	}
	return nil
}
