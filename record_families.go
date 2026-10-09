//go:build !js && !wasm

package main

// â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•
// THE RECORD SET â€” ONE owner for what the chain remembers.
//
// Brendan's rule (2026-09-19): **A is the source, and B and C must be
// reconstructible FROM A.** The design question per piece of state is therefore
// not "give it a family" but "is it primary, derived, or ephemeral":
//
//	primary   â€” a fact the world cannot rebuild from anything else. MUST be in A.
//	derived   â€” rebuildable from A. NO family; it is rebuilt on boot.
//	ephemeral â€” meaningless after a restart, by nature. Named here so silence is
//	            never mistaken for coverage.
//
// This ONE declaration has three consumers: the seed manifest (what the seed
// writes), the reader (which families it pulls and reassembles in order), and the
// Dev/Game Hub wizard (which families it may truthfully promise). None of them
// re-declares the set.
//
// A family whose prefix is EMPTY is not yet declarable: note_vocabulary.go is the
// only place a prefix may be declared and this file must not invent one. Such a
// family is PENDING with an explicit blocker, so a hole in the record set is
// visible instead of looking complete.
// â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•

// RecordClass says what must happen to a piece of state.
type RecordClass string

const (
	RecordClassPrimary   RecordClass = "primary"
	RecordClassDerived   RecordClass = "derived"
	RecordClassEphemeral RecordClass = "ephemeral"
)

// RecordFamily is ONE record family: the note prefix it is written under, and the
// state it carries.
type RecordFamily struct {
	Key     string   // stable key used by the seed manifest and the reader
	Prefix  string   // the vocabulary constant, or "" while it is not yet declared
	Carries []string // the state this family makes restorable
	Restore string   // how the state comes back
	Pending string   // a blocker, when the family cannot be written yet
}

// PrimaryRecordFamilies is Class 1 â€” the facts A must carry.
//
// Every family names a prefix, and the prefixes live in note_vocabulary.go (the
// sole owner). Nine existed before 2026-09-19; the thirteen declared on 2026-09-19
// from the plan's Â§13 C list complete the set, so the census reports 22 of 22
// declared and ZERO pending. A family may still be added PENDING (no prefix, with
// a stated blocker): the boot assertion refuses a family that is silent about why
// it cannot be written, so a hole in the record set stays COUNTED, never hidden.
var PrimaryRecordFamilies = []RecordFamily{
	{Key: "tenant_vaults", Prefix: NotePrefixTenantVaultSnapshot, Carries: []string{"tenant vaults (vault-only addresses leased to a tenant own bot, with their state and proof)"}, Restore: "rebuild the tenant vault registry"},
	{Key: "console_entitlements", Prefix: NotePrefixConsoleEntitlementSnapshot, Carries: []string{"console entitlements (the DLC purchases and leases granted from a VERIFIED platform purchase, keyed by the platform purchase id)"}, Restore: "rebuild granted console entitlements, one per platform purchase"},
	{Key: "leaderboard", Prefix: NotePrefixStateSnapshot, Carries: []string{"leaderboard"}, Restore: "newest snapshot wins"},
	// MEASURED 2026-09-19 (A4): this family WRITES far more than the ledger used to name. The
	// payload (lobby_manager.go saveEconomyState) carries these nine facts, and nothing else in
	// the tree names them. It also MARSHALS `matchHistory`, `marketNodes` and `onboardedWallets`,
	// whose OWNING families are `match_history`, `entity_market` and `onboarded_wallets` â€” a
	// batch (A5) puts several families in one record, so a mirror here is a TRANSPORT fact, not
	// a second owner (Problems.md Â§37 B1/B2).
	{Key: "economy", Prefix: NotePrefixEconomySnapshot,
		Carries: []string{"playerBalances", "playerLiquidityTiers", "activeKidnappings", "live tournament state", "seasonNumber/seasonStart", "paidParticipants", "token-sink audit counters", "initialRewards", "rewardTokens"},
		Restore: "newest snapshot wins"},
	{Key: "registered_tx", Prefix: NotePrefixRegTxSnapshot, Carries: []string{"registeredTxIDs"}, Restore: "newest snapshot wins"},
	{Key: "linked_wallets", Prefix: NotePrefixLinkSnapshot, Carries: []string{"linkedWallets", "LinkedPlatforms"}, Restore: "newest snapshot wins"},
	{Key: "onboarded_wallets", Prefix: NotePrefixOnboardSnapshot, Carries: []string{"onboardedWallets"}, Restore: "newest snapshot wins"},
	{Key: "card_cache", Prefix: NotePrefixCardCacheSnapshot, Carries: []string{"persistentCardCache"}, Restore: "newest snapshot wins, else refetch"},

	// â”€â”€ DECLARED 2026-09-19 (plan Â§13 C) â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
	// The sixteen primary facts that were process-only or file-only before this
	// pass â€” clubs, loans and auctions, then the thirteen Â§13 C families. Each now
	// names its prefix, so the census reads 22 declared / 0 pending, and the reader
	// iterates this slice rather than re-declaring the set.
	// treasuryAverages is PRIMARY, not derived: processTreasuryAnalytics (lobby_manager.go) keeps
	// a rolling EMA per club and item_service.go reads it. An EMA cannot be recomputed from the
	// current treasury â€” a rebuild would have to INVENT history â€” and it is club state, so it
	// lives here.
	{Key: "clubs", Prefix: NotePrefixClubSnapshot, Carries: []string{"clubs", "territories", "club inventories", "staff", "mojo", "club treasury", "treasuryAverages"},
		Restore: "rebuild clubs, then their territories and inventories"},
	{Key: "loans", Prefix: NotePrefixLoanSnapshot, Carries: []string{"loans"}, Restore: "rebuild the loan book"},
	{Key: "auctions", Prefix: NotePrefixAuctionSnapshot, Carries: []string{"auctions"}, Restore: "rebuild open auctions"},
	{Key: "entity_market", Prefix: NotePrefixEntityMarketSnapshot, Carries: []string{"marketNodes", "playerDirectInvestments", "playerInvestmentRecords"},
		Restore: "rebuild nodes, reserves and holdings"},
	{Key: "pets", Prefix: NotePrefixPetSnapshot, Carries: []string{"pets", "Grooming"}, Restore: "rebuild owned pets and their ladders"},
	{Key: "vehicles", Prefix: NotePrefixVehicleSnapshot, Carries: []string{"vehicles"}, Restore: "rebuild owned vehicles"},
	{Key: "world_content", Prefix: NotePrefixWorldContentSnapshot, Carries: []string{"worldContent"}, Restore: "rebuild authored and deployed content"},
	{Key: "rivalries", Prefix: NotePrefixRivalrySnapshot, Carries: []string{"Rivalries"}, Restore: "rebuild the rivalry graph"},
	{Key: "fenced_listings", Prefix: NotePrefixFencedListingSnapshot, Carries: []string{"fencedListings"}, Restore: "rebuild the fence"},
	{Key: "match_history", Prefix: NotePrefixMatchHistorySnapshot, Carries: []string{"matchHistory"}, Restore: "rebuild the match record"},
	{Key: "ai_citizens", Prefix: NotePrefixAICitizenSnapshot, Carries: []string{"AI citizens (ai_citizens.json)"}, Restore: "rebuild citizens and their bonds"},
	{Key: "bonded_assets", Prefix: NotePrefixBondedAssetSnapshot, Carries: []string{"bonded assets, bindings, listings, slide themes, card views"},
		Restore: "rebuild the registry and its indexes"},
	{Key: "items", Prefix: NotePrefixItemSnapshot, Carries: []string{"built items (item_registry.json)"}, Restore: "rebuild the item registry"},
	{Key: "faith", Prefix: NotePrefixFaithSnapshot, Carries: []string{"churches", "religions"}, Restore: "rebuild churches then religions"},
	{Key: "local_models", Prefix: NotePrefixLocalModelSnapshot, Carries: []string{"bot promotions (local_model_promotions.json)"}, Restore: "rebuild promotion records"},
}

// DerivedState is Class 2 â€” NO family. Each row names a source that EXISTS, because a derived
// row whose source does not exist is a fact persisted NOWHERE.
//
// THE A4 VERDICT (2026-09-19; operator rule: MEASURE THE READERS). Four rows named a source
// that does not exist and were RETIRED or RECLASSIFIED, each with its measurement:
//
//	treasuryAverages        MEASURED: processTreasuryAnalytics WRITES it (a rolling EMA per
//	                        club, deliberately path-dependent) and item_service.go READS it, so
//	                        it is PRIMARY â€” now carried by the `clubs` family above.
//	holdingBonuses          MEASURED: no writer and no reader anywhere in the tree. The FIELD
//	                        was RETIRED (backend_types.go), with its initialiser and this row.
//	rewardStack/rewardTokens MEASURED: WRITTEN by the economy snapshot, and an admin-created
//	                        reward token exists nowhere else, so they are PRIMARY â€” now carried
//	                        by the `economy` family above.
//	CollectorMap            MEASURED: WRITTEN by the cyber-audit (handlers_criminality.go) and
//	                        READ by nothing. The FIELD was RETIRED, with its writes and its row.
var DerivedState = []struct{ State, RebuiltFrom string }{
	// MEASURED, and the previous source ("clubs plus their territories") was not what the code
	// does: a district metric is rebuilt at BOOT from the persisted district snapshot â€”
	// economy_bootstrap.go fills Router.RegionalDistricts from snapshot.Districts â€” and that
	// snapshot is written by economy_persistence.go.
	{"RegionalDistricts", "the router snapshot at boot (economy_bootstrap.go), persisted by economy_persistence.go"},
	{"card_cache (local file)", "the card-cache family, else refetch from the indexer"},
	{"rewardStack", "initialRewards (rebuilt by applyDynamicScalingLocked from initial_rewards in the economy record, at the dynamic liquidity ratio)"},
	{"season archive index", "the CHAIN ITSELF: handleSeasonHistory scans transactions whose note carries VBT_SEASON_ARCHIVE: and dedupes by season (oracle_service.go:1954-1979), so there is no in-memory archive to persist"},
	{"derivatives manifest", "regenerated from the shipped placeholder pack"},
	{"economy_state_authoritative (local file)", "the economy family"},
	{"linked_wallets (local file)", "the linked-wallets family"},
}

// EphemeralState is Class 3 â€” named, deliberately NOT recorded.
var EphemeralState = []string{
	"live matches (a restart mid-match loses that match, and says so)",
	"rumors",
	"Caches",
	"HistoricalFrames",
	"rate-limiter buckets",
	"unconsumed nonces",
	"session/watchdog registrations",
}

// assertRecordSetDialect fails the BOOT if the record set contradicts itself, so a
// state can never be two things at once and a pending family can never look done.
func assertRecordSetDialect() {
	seen := make(map[string]bool, len(PrimaryRecordFamilies))
	primary := make(map[string]bool, len(PrimaryRecordFamilies))
	for _, f := range PrimaryRecordFamilies {
		if f.Key == "" {
			panic("record set: a family with no key cannot be referenced by the seed or the reader")
		}
		if seen[f.Key] {
			panic("record set: duplicate family key " + f.Key)
		}
		seen[f.Key] = true
		if len(f.Carries) == 0 {
			panic("record set: family " + f.Key + " carries nothing â€” a family that carries nothing is a lie")
		}
		if f.Restore == "" {
			panic("record set: family " + f.Key + " states no restore path")
		}
		if f.Prefix == "" && f.Pending == "" {
			panic("record set: family " + f.Key + " has no prefix AND no stated blocker â€” silence is not a reason")
		}
		if f.Prefix != "" && f.Pending != "" {
			panic("record set: family " + f.Key + " declares a prefix AND a blocker; one of them is stale")
		}
		for _, state := range f.Carries {
			primary[state] = true
		}
	}

	// A state cannot be both primary and derived: one of the two is wrong, and a
	// wrong class is how a fact ends up persisted nowhere.
	for _, d := range DerivedState {
		if primary[d.State] {
			panic("record set: " + d.State + " is declared BOTH primary and derived")
		}
		if d.RebuiltFrom == "" {
			panic("record set: derived state " + d.State + " names no source â€” it would be persisted nowhere")
		}
	}
	for _, e := range EphemeralState {
		if primary[e] {
			panic("record set: " + e + " is declared BOTH primary and ephemeral")
		}
	}
}

func init() { assertRecordSetDialect() }

// RecordSetCensus is the counted shape of the set, for the seed manifest, the
// reader and the wizard to report instead of guessing.
func RecordSetCensus() map[string]int {
	declared, pending := 0, 0
	for _, f := range PrimaryRecordFamilies {
		if f.Prefix == "" {
			pending++
			continue
		}
		declared++
	}
	return map[string]int{
		"primary_declared": declared,
		"primary_pending":  pending,
		"derived":          len(DerivedState),
		"ephemeral":        len(EphemeralState),
	}
}

// RecordFamilyKeyForPrefix resolves a note prefix to its family key using THIS
// declaration â€” the ONE owner of the set â€” so the record envelope never carries a
// second copy of the family list.
//
// ok=false is a REFUSAL, not a fallback: a prefix outside the declared set must not
// be written, because the reader iterates the set and a record outside it would be
// invisible to every reader that exists.
func RecordFamilyKeyForPrefix(prefix string) (string, bool) {
	for _, f := range PrimaryRecordFamilies {
		if f.Prefix != "" && f.Prefix == prefix {
			return f.Key, true
		}
	}
	return "", false
}

// RecordPrefixIsBatch reports whether a prefix is THE batch envelope (A5, plan 15): the transport
// that carries SEVERAL families in ONE record per cadence. A batch is transport, NOT state, so it
// has no family key - but the writer must ACCEPT it, or the one cadence could never write a batch,
// while everything else stays refused by RecordFamilyKeyForPrefix. That refusal is the point.
func RecordPrefixIsBatch(prefix string) bool {
	return prefix == NotePrefixBatchSnapshot
}
