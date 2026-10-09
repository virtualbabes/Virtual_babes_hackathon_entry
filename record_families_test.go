//go:build !js && !wasm

package main

import (
	"os"
	"strings"
	"testing"
)

// TestRecordSetShape pins the counted shape of the record set. A change to the set
// must be a DELIBERATE change to this number, not a silent one.
func TestRecordSetShape(t *testing.T) {
	c := RecordSetCensus()
	if c["primary_declared"] != 23 {
		t.Errorf("declared primary families = %d; want 23 (9 pre-existing + clubs, loans and auctions + the thirteen declared 2026-09-19 from plan ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â§13 C)", c["primary_declared"])
	}
	if c["primary_pending"] != 0 {
		t.Errorf("pending primary families = %d; want 0 ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â every ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â§13 C primary fact now names a prefix in note_vocabulary.go", c["primary_pending"])
	}
	if c["derived"] == 0 || c["ephemeral"] == 0 {
		t.Error("the derived and ephemeral classes must both be populated: an empty class reads as coverage")
	}
}

// TestDeclaredFamiliesUseTheVocabularyOwner proves each declared family points at
// the prefix note_vocabulary.go actually owns ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â a family that named the wrong
// prefix would read and write a different record set than it claims.
func TestDeclaredFamiliesUseTheVocabularyOwner(t *testing.T) {
	want := map[string]string{
		"tenant_vaults":     NotePrefixTenantVaultSnapshot,
		"leaderboard":       NotePrefixStateSnapshot,
		"economy":           NotePrefixEconomySnapshot,
		"registered_tx":     NotePrefixRegTxSnapshot,
		"linked_wallets":    NotePrefixLinkSnapshot,
		"onboarded_wallets": NotePrefixOnboardSnapshot,
		"card_cache":        NotePrefixCardCacheSnapshot,
		// Declared 2026-09-19 from the pending list (plan ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â§13 C).
		"clubs":    NotePrefixClubSnapshot,
		"loans":    NotePrefixLoanSnapshot,
		"auctions": NotePrefixAuctionSnapshot,
		// The thirteen declared 2026-09-19 (plan ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â§13 C, the remaining set).
		"entity_market":   NotePrefixEntityMarketSnapshot,
		"pets":            NotePrefixPetSnapshot,
		"vehicles":        NotePrefixVehicleSnapshot,
		"world_content":   NotePrefixWorldContentSnapshot,
		"rivalries":       NotePrefixRivalrySnapshot,
		"fenced_listings": NotePrefixFencedListingSnapshot,
		"match_history":   NotePrefixMatchHistorySnapshot,
		"ai_citizens":     NotePrefixAICitizenSnapshot,
		"bonded_assets":   NotePrefixBondedAssetSnapshot,
		"items":           NotePrefixItemSnapshot,
		"faith":           NotePrefixFaithSnapshot,
		// Workstream C (2026-10-08): the console entitlement registry, keyed by the platform purchase id.
		"console_entitlements": NotePrefixConsoleEntitlementSnapshot,

		"local_models": NotePrefixLocalModelSnapshot,
	}
	got := map[string]string{}
	for _, f := range PrimaryRecordFamilies {
		if f.Prefix != "" {
			got[f.Key] = f.Prefix
		}
	}
	if len(got) != len(want) {
		t.Fatalf("declared families = %d; want %d", len(got), len(want))
	}
	for k, p := range want {
		if got[k] != p {
			t.Errorf("family %s = %q; want the vocabulary constant %q", k, got[k], p)
		}
	}
}

// TestEveryFactHasExactlyOneOwner ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â a fact carried by two families has two owners,
// which is how a record set drifts. And every family key is pinned BY NAME, so a
// hole can never be closed by quietly deleting a row.
func TestEveryFactHasExactlyOneOwner(t *testing.T) {
	seen := map[string]string{}
	for _, f := range PrimaryRecordFamilies {
		for _, state := range f.Carries {
			if other, dup := seen[state]; dup {
				t.Errorf("%q is carried by BOTH %s and %s ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â one fact, one owner", state, other, f.Key)
			}
			seen[state] = f.Key
		}
	}

	// EVERY family must now name a prefix: the ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â§13 C list was closed on 2026-09-19,
	// so a family that still carries a blocker is a HOLE and must fail here rather
	// than read as coverage. The keys are named one by one, so a gap can never be
	// closed by quietly deleting a row.
	pending := map[string]bool{}
	for _, f := range PrimaryRecordFamilies {
		if f.Prefix == "" {
			pending[f.Key] = true
		}
	}
	for _, key := range []string{
		"leaderboard", "economy", "registered_tx", "linked_wallets", "onboarded_wallets",
		"card_cache", "clubs", "loans", "auctions", "entity_market", "pets", "vehicles",
		"world_content", "rivalries", "fenced_listings", "match_history", "ai_citizens",
		"bonded_assets", "items", "faith", "local_models", "season_archive",
	} {
		if pending[key] {
			t.Errorf("primary fact %q carries no prefix: the ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â§13 C set was closed on 2026-09-19, so this is a NEW hole ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â declare it in note_vocabulary.go or state its blocker deliberately", key)
		}
	}
	if len(pending) != 0 {
		t.Errorf("pending families = %d; want 0 ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â every declared primary fact names a prefix", len(pending))
	}
}

// TestDerivedStateNamesItsSource ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â a derived item that names no source would be
// persisted nowhere, which is the exact failure this class exists to prevent.
func TestDerivedStateNamesItsSource(t *testing.T) {
	for _, d := range DerivedState {
		if d.State == "" || d.RebuiltFrom == "" {
			t.Errorf("derived state %+v names no source", d)
		}
	}
	if len(EphemeralState) == 0 {
		t.Error("the ephemeral list is empty: 'not recorded' must be a decision that is written down")
	}
}

// TestRecordSetDialectHolds re-runs the boot assertion inside a test, so a
// contradiction fails `go test` rather than only the boot.
func TestRecordSetDialectHolds(t *testing.T) {
	assertRecordSetDialect()
}

// TestTheRetiredDerivedRowsStayRetired ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â A4 retired four rows because their named rebuild
// source does not exist (Problems.md ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â§40). A derived row whose source does not exist is a fact
// persisted NOWHERE, so the retirement is pinned: if one returns, it returns deliberately.
func TestTheRetiredDerivedRowsStayRetired(t *testing.T) {
	derived := map[string]bool{}
	for _, d := range DerivedState {
		derived[d.State] = true
	}
	for _, gone := range []string{"treasuryAverages", "holdingBonuses", "CollectorMap"} {
		if derived[gone] {
			t.Errorf("%q is derived again ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â A4 retired it WITH its measurement; re-adding it silently names a source that does not exist", gone)
		}
	}
	// Two of the four were RECLASSIFIED, and must be carried by the family that owns them.
	carriedBy := map[string]string{}
	for _, f := range PrimaryRecordFamilies {
		for _, state := range f.Carries {
			carriedBy[state] = f.Key
		}
	}
	if got := carriedBy["treasuryAverages"]; got != "clubs" {
		t.Errorf("treasuryAverages is PRIMARY (a path-dependent EMA with a writer and a reader) and must be carried by the clubs family; got %q", got)
	}
	for _, state := range []string{"initialRewards", "rewardTokens"} {
		if got := carriedBy[state]; got != "economy" {
			t.Errorf("%s must be carried by the economy family: rewardTokens is PRIMARY (an admin-created token exists nowhere else) and initialRewards is the payout template the scaled stack is rebuilt from; got %q", state, got)
		}
	}
}

// TestRetiredFieldsAreGoneFromTheSource ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â the two FIELDS A4 retired must not return to the type
// that held them: a field nothing writes and nothing reads is state persisted nowhere
// (holdingBonuses), and a write-only map has no consumer (CollectorMap).
func TestRetiredFieldsAreGoneFromTheSource(t *testing.T) {
	raw, err := os.ReadFile("backend_types.go")
	if err != nil {
		t.Fatalf("read backend_types.go: %v", err)
	}
	src := string(raw)
	for _, field := range []string{"holdingBonuses", "CollectorMap"} {
		if strings.Contains(src, field) {
			t.Errorf("%s is declared again in backend_types.go; A4 retired it as state nothing consumes", field)
		}
	}
	if !strings.Contains(src, "ActiveRecords map[string]*RaidEvidence") {
		t.Error("the evidence pool ActiveRecords (the LIVE store) must survive the retirement")
	}
}
