//go:build !js && !wasm

package main

import (
	"strings"
)

// â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•
// NOTE VOCABULARY â€” THE SINGLE OWNER OF EVERY ON-CHAIN NOTE PREFIX.
//
// A "note" is the free-text field an AVM transaction carries. This project uses
// notes for two very different things:
//
//  1. MONEY DOORS (client â†’ vault). The client pays the vault and binds the
//     payment to ONE purpose by prefixing the note. The server then proves the
//     payment with `VerifyBuyInTransaction`, which requires an exact purpose
//     prefix match. A prefix that DRIFTS does not raise an error â€” it silently
//     fails verification, or worse, lets a payment for one purpose be replayed
//     against another.
//
//  2. STATE CHECKPOINTS (vault â†’ vault) and AUDIT LOGS (vault â†’ chain). These
//     are WRITTEN by the vault and read back by narrow, vault-scoped readers.
//
// OWNERSHIP RULE (binding): no other file in this repository may spell a note
// prefix as a string literal. Server code references the constants declared
// here; the browser reads the served vocabulary
// (`GET /api/notes/vocabulary`) instead of re-typing a prefix. A note prefix is
// therefore declared in exactly ONE place, so the client and the server can
// never disagree about it.
//
// FAIL-LOUD: `assertNoteVocabularyPolicy()` runs at `init()` and panics if a
// prefix is empty, duplicated, malformed, or newly ambiguous (a proper prefix
// of another declared prefix). The build cannot run silently with the
// vocabulary broken â€” the same discipline Â§10.1 uses for the card exclusion.
//
// TAMPER-EVIDENCE, NOT PREVENTION: a user may put ANY text in a note. Nothing
// here tries to stop that. What keeps the model sound is that every
// authoritative reader is VAULT/Faucet-SCOPED â€” it requires the transfer to be
// `from = vault` and/or `to = vault` â€” so a note a player authored is never
// read as an instruction. A forged note is inert, not dangerous.
// â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•

// Note scopes. A scope states WHO may legitimately author the note.
const (
	// NoteScopeMoneyDoor: authored by a CLIENT and consumed by the vault at a
	// money door. Verified by VerifyBuyInTransaction against the exact prefix.
	NoteScopeMoneyDoor = "money_door"
	// NoteScopeCheckpoint: authored by the VAULT and read back by the vault to
	// reconstruct state. Never authored by a client.
	NoteScopeCheckpoint = "checkpoint"
	// NoteScopeAuditLog: authored by the VAULT as an immutable audit trail.
	// Nothing reads it; it exists so the chain remembers.
	NoteScopeAuditLog = "audit_log"
)

// Note directions. A direction states which way the value moved.
const (
	NoteDirectionClientToVault = "client_to_vault"
	NoteDirectionVaultToVault  = "vault_to_vault"
	NoteDirectionVaultToChain  = "vault_to_chain"
)

// â”€â”€ MONEY DOORS (client â†’ vault) â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
// Every one of these is passed as the final argument of a
// `VerifyBuyInTransaction` call. The trailing colon is REQUIRED: without it a
// shorter purpose would match a longer one.
const (
	NotePrefixCourthouseFine    = "COURTHOUSE_FINE:"
	NotePrefixBailPayment       = "BAIL_PAYMENT:"
	NotePrefixRepayLoan         = "REPAY_LOAN:"
	NotePrefixCounterfeitDetect = "COUNTERFEIT_DETECT:"
	// NotePrefixFoundClub / JoinClub / ClaimDistrict carry a bound, dynamic part
	// (see the Note* builders below) so a payment cannot be replayed against a
	// different organization, club or district.
	NotePrefixFoundClub            = "FOUND_CLUB:"
	NotePrefixJoinClub             = "JOIN_CLUB:"
	NotePrefixClaimDistrict        = "CLAIM_DISTRICT:"
	NotePrefixTournamentBuyIn      = "VBT_TOURN_BUYIN:"
	NotePrefixArenaTournamentBuyIn = "ARENA_TOURN_BUYIN:"
)

// â”€â”€ STATE CHECKPOINTS (vault â†’ vault) â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
const (
	NotePrefixStateSnapshot     = "VBT_STATE_SNAPSHOT:"
	NotePrefixEconomySnapshot   = "VBT_ECONOMY_SNAPSHOT:"
	NotePrefixRegTxSnapshot     = "VBT_REG_TX_SNAPSHOT:"
	NotePrefixLinkSnapshot      = "VBT_LINK_SNAPSHOT:"
	NotePrefixOnboardSnapshot   = "VBT_ONBOARD_SNAPSHOT:"
	NotePrefixCardCacheSnapshot = "VBT_CARD_CACHE_SNAPSHOT:"

	// State families added 2026-09-19 (plan Â§13): the primary facts that were
	// process-only before â€” clubs (with their territories and inventories), the
	// loan book, and the open auctions. Declared HERE because this file is the only
	// owner of a prefix; `record_families.go` is the only owner of which state each
	// family carries.
	NotePrefixClubSnapshot    = "VBT_CLUB_SNAPSHOT:"
	NotePrefixLoanSnapshot    = "VBT_LOAN_SNAPSHOT:"
	NotePrefixAuctionSnapshot = "VBT_AUCTION_SNAPSHOT:"
	// State families declared 2026-09-19 (plan Â§13 C â€” the thirteen remaining
	// primary facts). Each is a VAULT-authored checkpoint: the vault writes it and
	// the vault reads it back for reconstruction, so no client can author one.
	NotePrefixEntityMarketSnapshot  = "VBT_ENTITY_MARKET_SNAPSHOT:"
	NotePrefixPetSnapshot           = "VBT_PET_SNAPSHOT:"
	NotePrefixVehicleSnapshot       = "VBT_VEHICLE_SNAPSHOT:"
	NotePrefixWorldContentSnapshot  = "VBT_WORLD_CONTENT_SNAPSHOT:"
	NotePrefixRivalrySnapshot       = "VBT_RIVALRY_SNAPSHOT:"
	NotePrefixFencedListingSnapshot = "VBT_FENCED_LISTING_SNAPSHOT:"
	NotePrefixMatchHistorySnapshot  = "VBT_MATCH_HISTORY_SNAPSHOT:"
	NotePrefixAICitizenSnapshot     = "VBT_AI_CITIZEN_SNAPSHOT:"
	NotePrefixBondedAssetSnapshot   = "VBT_BONDED_ASSET_SNAPSHOT:"
	NotePrefixItemSnapshot          = "VBT_ITEM_SNAPSHOT:"
	NotePrefixFaithSnapshot         = "VBT_FAITH_SNAPSHOT:"
	NotePrefixLocalModelSnapshot    = "VBT_LOCAL_MODEL_SNAPSHOT:"
	NotePrefixSeasonArchiveSnapshot = "VBT_SEASON_ARCHIVE_SNAPSHOT:"

	// A6 (Stage A): the TENANT VAULT registry is new PRIMARY state - vault-only addresses leased to a
	// tenant own bot, with their state and proof. Declared with the registry, never before it.
	NotePrefixTenantVaultSnapshot = "VBT_TENANT_VAULT_SNAPSHOT:"

	// THE BATCH ENVELOPE (A5, plan 15). This is NOT a state family: it is the transport envelope that
	// carries SEVERAL families in ONE record per cadence, chunked sequentially when too large. It is
	// declared HERE because this file is the only owner of a prefix, and it is deliberately absent from
	// PrimaryRecordFamilies because a batch owns no fact.
	NotePrefixBatchSnapshot = "VBT_BATCH_SNAPSHOT:"
	// Workstream C (2026-10-08): the CONSOLE ENTITLEMENT registry is new PRIMARY state - the DLC
	// purchases and leases a platform sale granted, keyed by the platform purchase id. Declared HERE,
	// beside the registry that owns it (the same rule the tenant vault family follows).
	NotePrefixConsoleEntitlementSnapshot = "VBT_CONSOLE_ENTITLEMENT_SNAPSHOT:"
)

// â”€â”€ AUDIT LOGS (vault â†’ chain) â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
const (
	NotePrefixOnboard         = "VBT_ONBOARD:"
	NotePrefixWin             = "VBT_WIN:"
	NotePrefixDNF             = "VBT_DNF:"
	NotePrefixGovDividend     = "VBT_GOV_DIVIDEND:"
	NotePrefixAlgoDividend    = "VBT_ALGO_DIV:"
	NotePrefixVoiDividend     = "VBT_VOI_DIV:"
	NotePrefixTournSummary    = "VBT_TOURN_SUMM:"
	NotePrefixTournData       = "VBT_TOURN_DATA:"
	NotePrefixTournPayout     = "VBT_TOURN_PAYOUT:"
	NotePrefixLoanPayback     = "VBT_LOAN_PAYBACK:"
	NotePrefixLoanLiquidate   = "VBT_LOAN_LIQUIDATE:"
	NotePrefixLoanAutoPull    = "VBT_LOAN_AUTO_PULL:"
	NotePrefixAuctionSettle   = "VBT_AUCTION_SETTLE:"
	NotePrefixAuctionPull     = "VBT_AUCTION_PULL:"
	NotePrefixHeistLog        = "VBT_HEIST_LOG:"
	NotePrefixSabotageLog     = "VBT_SABOTAGE_LOG:"
	NotePrefixLeaseTake       = "VBT_LEASE_TAKE:"
	NotePrefixLeaseReturn     = "VBT_LEASE_RETURN:"
	NotePrefixRansomLog       = "VBT_RANSOM_LOG:"
	NotePrefixBailLog         = "VBT_BAIL_LOG:"
	NotePrefixInsuranceReturn = "VBT_INSURANCE_RETURN:"
	NotePrefixShareTrade      = "VBT_SHARE_TRADE:"
	NotePrefixSeasonArchive   = "VBT_SEASON_ARCHIVE:"
)

// NotePurpose catalogues one prefix: what it is called, what it is, who authors
// it, which way the value moved, and what it means.
type NotePurpose struct {
	Key         string `json:"key"`
	Prefix      string `json:"prefix"`
	Scope       string `json:"scope"`
	Direction   string `json:"direction"`
	Description string `json:"description"`
}

// noteVocabulary is THE catalogue. Every NotePrefix* constant declared above
// appears here exactly once (asserted at init and by note_vocabulary_test.go).
var noteVocabulary = []NotePurpose{
	// â”€â”€ state families added 2026-09-19 (plan Â§13) â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
	{Key: "tenant_vault_state", Prefix: NotePrefixTenantVaultSnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "Tenant vaults: vault-only addresses leased to a tenant own bot, with their state and proof."},
	{Key: "console_entitlement_state", Prefix: NotePrefixConsoleEntitlementSnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "Console entitlements: the DLC purchases and leases granted from a VERIFIED platform purchase, keyed by the platform purchase id."},
	{Key: "record_batch", Prefix: NotePrefixBatchSnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "THE BATCH ENVELOPE (A5): one record per cadence carrying several families, chunked sequentially when too large. It is TRANSPORT, not state - it introduces no family and no new owner of any fact."},
	{Key: "club_state", Prefix: NotePrefixClubSnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "Club state: clubs, their territories, inventories, staff, mojo and club treasury."},
	{Key: "loan_book", Prefix: NotePrefixLoanSnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "The loan book: principal, interest, collateral and due dates."},
	{Key: "auction_book", Prefix: NotePrefixAuctionSnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "Open auctions: lots, reserves, current bids and settlement state."},
	// â”€â”€ state families declared 2026-09-19 (plan Â§13 C â€” the thirteen remaining
	//    primary facts). Each is a VAULT-authored checkpoint: the vault writes it
	//    and the vault reads it back, so no client can author one.
	{Key: "entity_market_state", Prefix: NotePrefixEntityMarketSnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "Entity market state: AMM nodes with their reserves, direct investments and per-wallet investment records."},
	{Key: "pet_state", Prefix: NotePrefixPetSnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "Companion pets: lineage, traits, stats, level, maturity, provenance and grooming ladders."},
	{Key: "vehicle_state", Prefix: NotePrefixVehicleSnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "Vehicles: class, stat vector, build level, fitted parts, region and provenance."},
	{Key: "world_content_state", Prefix: NotePrefixWorldContentSnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "Authored and deployed world content: kind, region, placement and provenance."},
	{Key: "rivalry_state", Prefix: NotePrefixRivalrySnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "The rivalry graph: active rivalries, their sides, scores and expiry."},
	{Key: "fenced_listing_state", Prefix: NotePrefixFencedListingSnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "The fence: fenced goods listings, their asking prices and their adoption state."},
	{Key: "match_history_state", Prefix: NotePrefixMatchHistorySnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "Match history: completed matches, their participants, scores and settlement state."},
	{Key: "ai_citizen_state", Prefix: NotePrefixAICitizenSnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "AI citizens: identity, their own wallet, bonds, stats and reputation."},
	{Key: "bonded_asset_state", Prefix: NotePrefixBondedAssetSnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "Bonded assets: the registry, its bindings, its listings and the viewer-scoped themes."},
	{Key: "item_state", Prefix: NotePrefixItemSnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "Built items: the item registry â€” kind, stats, owner and provenance."},
	{Key: "faith_state", Prefix: NotePrefixFaithSnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "Faith: churches with their owning wallets, and the religions with their dogmas."},
	{Key: "local_model_state", Prefix: NotePrefixLocalModelSnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "Local-model promotion records for bots: pathway, build state and calibration."},
	{Key: "season_archive_state", Prefix: NotePrefixSeasonArchiveSnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "Season archives: the rolled-over season ledger and its reward pools."},

	// â”€â”€ money doors (client â†’ vault) â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
	{Key: "courthouse_fine", Prefix: NotePrefixCourthouseFine, Scope: NoteScopeMoneyDoor, Direction: NoteDirectionClientToVault,
		Description: "Fine paid to clear a Wanted Level (100 $VBV per Wanted point)."},
	{Key: "bail_payment", Prefix: NotePrefixBailPayment, Scope: NoteScopeMoneyDoor, Direction: NoteDirectionClientToVault,
		Description: "Bail paid to release a jailed card from a club's jail."},
	{Key: "repay_loan", Prefix: NotePrefixRepayLoan, Scope: NoteScopeMoneyDoor, Direction: NoteDirectionClientToVault,
		Description: "Loan repayment (principal + interest) returning the collateral bundle."},
	{Key: "counterfeit_detect", Prefix: NotePrefixCounterfeitDetect, Scope: NoteScopeMoneyDoor, Direction: NoteDirectionClientToVault,
		Description: "Detection fee paid by a Justice career to sweep a sector for counterfeit notes."},
	{Key: "found_club", Prefix: NotePrefixFoundClub, Scope: NoteScopeMoneyDoor, Direction: NoteDirectionClientToVault,
		Description: "Club foundry fee. Bound to the organization name (see NoteFoundClub)."},
	{Key: "join_club", Prefix: NotePrefixJoinClub, Scope: NoteScopeMoneyDoor, Direction: NoteDirectionClientToVault,
		Description: "Club entry fee. Bound to the club id (see NoteJoinClub)."},
	{Key: "claim_district", Prefix: NotePrefixClaimDistrict, Scope: NoteScopeMoneyDoor, Direction: NoteDirectionClientToVault,
		Description: "Territory purchase. Bound to club id AND district (see NoteClaimDistrict)."},
	{Key: "tournament_buyin", Prefix: NotePrefixTournamentBuyIn, Scope: NoteScopeMoneyDoor, Direction: NoteDirectionClientToVault,
		Description: "Tournament buy-in. Bound to the tournament id (see NoteTournamentBuyIn)."},
	{Key: "arena_tournament_buyin", Prefix: NotePrefixArenaTournamentBuyIn, Scope: NoteScopeMoneyDoor, Direction: NoteDirectionClientToVault,
		Description: "Arena tournament buy-in prefix, read by the registration reconstruction scan."},

	// â”€â”€ state checkpoints (vault â†’ vault) â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
	{Key: "state_snapshot", Prefix: NotePrefixStateSnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "Whole-leaderboard checkpoint (gzip+base64) for boot reconstruction."},
	{Key: "economy_snapshot", Prefix: NotePrefixEconomySnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "Economy checkpoint: player balances and active kidnappings."},
	{Key: "reg_tx_snapshot", Prefix: NotePrefixRegTxSnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "Checkpoint of the consumed transaction-id memo."},
	{Key: "link_snapshot", Prefix: NotePrefixLinkSnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "Checkpoint of the cross-platform linked-wallet registry."},
	{Key: "onboard_snapshot", Prefix: NotePrefixOnboardSnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "Checkpoint of the Sybil/onboarding registry of funded wallets."},
	{Key: "card_cache_snapshot", Prefix: NotePrefixCardCacheSnapshot, Scope: NoteScopeCheckpoint, Direction: NoteDirectionVaultToVault,
		Description: "Checkpoint of the persistent card cache."},

	// â”€â”€ audit logs (vault â†’ chain) â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
	{Key: "onboard", Prefix: NotePrefixOnboard, Scope: NoteScopeAuditLog, Direction: NoteDirectionVaultToChain,
		Description: "Onboarding gas/token transfer marker."},
	{Key: "win", Prefix: NotePrefixWin, Scope: NoteScopeAuditLog, Direction: NoteDirectionVaultToChain,
		Description: "Immutable win record (opponent, scores, tournament/match ids)."},
	{Key: "dnf", Prefix: NotePrefixDNF, Scope: NoteScopeAuditLog, Direction: NoteDirectionVaultToChain,
		Description: "Immutable did-not-finish record."},
	{Key: "gov_dividend", Prefix: NotePrefixGovDividend, Scope: NoteScopeAuditLog, Direction: NoteDirectionVaultToChain,
		Description: "Governance dividend payout record."},
	{Key: "algo_dividend", Prefix: NotePrefixAlgoDividend, Scope: NoteScopeAuditLog, Direction: NoteDirectionVaultToChain,
		Description: "Algorand-side dividend payout record â€” the plug-in chain's fallback payout rail. MEASURED 2026-09-19: this purpose was written as the bare literal \"VBET_ALGO_DIV\" (no colon, no catalogue entry) by the multi-chain router, which is why it is declared here."},
	{Key: "voi_dividend", Prefix: NotePrefixVoiDividend, Scope: NoteScopeAuditLog, Direction: NoteDirectionVaultToChain,
		Description: "Voi-side dividend payout record from the multi-chain router. MEASURED 2026-09-19: written as the bare literal \"VBET_VOI_DIV\" (no colon, no catalogue entry). The router's OUTBOUND rails are also a MODEL CONFLICT with plan Â§1 ('payouts only in $VBV on Voi; there is no other outbound rail') â€” recorded for the operator, not changed here."},
	{Key: "tourn_summary", Prefix: NotePrefixTournSummary, Scope: NoteScopeAuditLog, Direction: NoteDirectionVaultToChain,
		Description: "Tournament summary archive."},
	{Key: "tourn_data", Prefix: NotePrefixTournData, Scope: NoteScopeAuditLog, Direction: NoteDirectionVaultToChain,
		Description: "Tournament data chunk archive."},
	{Key: "tourn_payout", Prefix: NotePrefixTournPayout, Scope: NoteScopeAuditLog, Direction: NoteDirectionVaultToChain,
		Description: "Tournament payout record (tid, rank, pot share)."},
	{Key: "loan_payback", Prefix: NotePrefixLoanPayback, Scope: NoteScopeAuditLog, Direction: NoteDirectionVaultToChain,
		Description: "Loan repayment financial-proof record."},
	{Key: "loan_liquidate", Prefix: NotePrefixLoanLiquidate, Scope: NoteScopeAuditLog, Direction: NoteDirectionVaultToChain,
		Description: "Loan collateral liquidation record."},
	{Key: "loan_auto_pull", Prefix: NotePrefixLoanAutoPull, Scope: NoteScopeAuditLog, Direction: NoteDirectionVaultToChain,
		Description: "Automatic collateral pull marker."},
	{Key: "auction_settle", Prefix: NotePrefixAuctionSettle, Scope: NoteScopeAuditLog, Direction: NoteDirectionVaultToChain,
		Description: "Auction settlement record."},
	{Key: "auction_pull", Prefix: NotePrefixAuctionPull, Scope: NoteScopeAuditLog, Direction: NoteDirectionVaultToChain,
		Description: "Auction consignment pull marker."},
	{Key: "heist_log", Prefix: NotePrefixHeistLog, Scope: NoteScopeAuditLog, Direction: NoteDirectionVaultToChain,
		Description: "Heist outcome record."},
	{Key: "sabotage_log", Prefix: NotePrefixSabotageLog, Scope: NoteScopeAuditLog, Direction: NoteDirectionVaultToChain,
		Description: "Regional sabotage outcome record."},
	{Key: "lease_take", Prefix: NotePrefixLeaseTake, Scope: NoteScopeAuditLog, Direction: NoteDirectionVaultToChain,
		Description: "Infrastructure lease take-up record."},
	{Key: "lease_return", Prefix: NotePrefixLeaseReturn, Scope: NoteScopeAuditLog, Direction: NoteDirectionVaultToChain,
		Description: "Infrastructure lease return record."},
	{Key: "ransom_log", Prefix: NotePrefixRansomLog, Scope: NoteScopeAuditLog, Direction: NoteDirectionVaultToChain,
		Description: "Kidnap ransom payment record."},
	{Key: "bail_log", Prefix: NotePrefixBailLog, Scope: NoteScopeAuditLog, Direction: NoteDirectionVaultToChain,
		Description: "Card bail settlement record."},
	{Key: "insurance_return", Prefix: NotePrefixInsuranceReturn, Scope: NoteScopeAuditLog, Direction: NoteDirectionVaultToChain,
		Description: "Insurance recovery return record."},
	{Key: "share_trade", Prefix: NotePrefixShareTrade, Scope: NoteScopeAuditLog, Direction: NoteDirectionVaultToChain,
		Description: "Entity share trade record."},
	{Key: "season_archive", Prefix: NotePrefixSeasonArchive, Scope: NoteScopeAuditLog, Direction: NoteDirectionVaultToChain,
		Description: "Season rollover archive."},
}

// â”€â”€ lookup helpers â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

// NotePrefixes returns every declared prefix in catalogue order.
func NotePrefixes() []string {
	out := make([]string, 0, len(noteVocabulary))
	for _, p := range noteVocabulary {
		out = append(out, p.Prefix)
	}
	return out
}

// NotePrefixForKey resolves a catalogue key (e.g. "courthouse_fine") to its
// prefix. The client echoes keys, never prefixes, so a stale client cannot
// smuggle in a prefix the server does not know.
func NotePrefixForKey(key string) (string, bool) {
	for _, p := range noteVocabulary {
		if p.Key == key {
			return p.Prefix, true
		}
	}
	return "", false
}

// IsDeclaredNotePrefix reports whether p is exactly one of the declared
// prefixes. NOT a prefix-of test: `VBT_ONBOARD:` must not answer true for
// `VBT_ONBOARD_SNAPSHOT:`.
func IsDeclaredNotePrefix(p string) bool {
	for _, declared := range noteVocabulary {
		if declared.Prefix == p {
			return true
		}
	}
	return false
}

// IsMoneyDoorPurpose reports whether p is a client â†’ vault purpose. This is the
// set a browser is allowed to name.
func IsMoneyDoorPurpose(p string) bool {
	for _, declared := range noteVocabulary {
		if declared.Prefix == p && declared.Scope == NoteScopeMoneyDoor {
			return true
		}
	}
	return false
}

// â”€â”€ bound prefix builders â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
// The dynamic part of a bound prefix is ALWAYS derived from a declared base, so
// a caller cannot invent a purpose.
//
// notePrefixBuild PANICS on an undeclared base: that is a first-party wiring
// error (the same class the Â§10.1 card-exclusion init panic guards) and it is
// unreachable from a request. An EMPTY bound part returns "" instead of
// panicking â€” a data error must not take the process down. "" then fails closed
// at the boundary, because VerifyBuyInTransaction refuses an empty prefix.

func notePrefixBuild(base string, parts ...string) string {
	if !IsDeclaredNotePrefix(base) {
		panic("note vocabulary: undeclared base prefix " + base +
			" (declare it in note_vocabulary.go; a drifted prefix silently fails verification)")
	}
	var b strings.Builder
	b.WriteString(base)
	for _, part := range parts {
		if part == "" {
			// An unbound purpose could be replayed against a different target.
			// Return the empty string so the door fails closed at the boundary.
			return ""
		}
		b.WriteString(part)
		b.WriteString(":")
	}
	return b.String()
}

// NoteFoundClub binds the foundry fee to the organization name.
func NoteFoundClub(name string) string { return notePrefixBuild(NotePrefixFoundClub, name) }

// NoteJoinClub binds the entry fee to the club id.
func NoteJoinClub(clubID string) string { return notePrefixBuild(NotePrefixJoinClub, clubID) }

// NoteClaimDistrict binds the territory purchase to club id AND district.
func NoteClaimDistrict(clubID, territoryID string) string {
	return notePrefixBuild(NotePrefixClaimDistrict, clubID, territoryID)
}

// NoteTournamentBuyIn binds the buy-in to the running tournament id.
func NoteTournamentBuyIn(tournamentID string) string {
	return notePrefixBuild(NotePrefixTournamentBuyIn, tournamentID)
}

// NoteArenaTournamentBuyIn binds the legacy/arena buy-in to a tournament id.
func NoteArenaTournamentBuyIn(tournamentID string) string {
	return notePrefixBuild(NotePrefixArenaTournamentBuyIn, tournamentID)
}

// â”€â”€ served vocabulary â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

// NoteVocabularyPayload is the READ-ONLY payload served at
// `GET /api/notes/vocabulary`. The browser uses it to learn the prefixes it is
// allowed to write; it never re-declares one. `scope`/`direction` let a surface
// render only the purposes a client may name.
func NoteVocabularyPayload() map[string]interface{} {
	purposes := make([]map[string]string, 0, len(noteVocabulary))
	moneyDoor := make([]string, 0, len(noteVocabulary))
	for _, p := range noteVocabulary {
		purposes = append(purposes, map[string]string{
			"key":         p.Key,
			"prefix":      p.Prefix,
			"scope":       p.Scope,
			"direction":   p.Direction,
			"description": p.Description,
		})
		if p.Scope == NoteScopeMoneyDoor {
			moneyDoor = append(moneyDoor, p.Prefix)
		}
	}
	return map[string]interface{}{
		"success":  true,
		"version":  1,
		"purposes": purposes,
		"counts": map[string]int{
			"total":      len(purposes),
			"money_door": len(moneyDoor),
			"checkpoint": len(notePrefixesByScope(NoteScopeCheckpoint)),
			"audit_log":  len(notePrefixesByScope(NoteScopeAuditLog)),
		},
		// Stated rules â€” a surface renders the rule, it does not invent one.
		"rules": map[string]interface{}{
			"client_may_name_money_door_only": true,
			"prefixes_are_exact_match":        true,
			"empty_prefix_is_refused":         true,
			"single_owner":                    "Go: note_vocabulary.go (this payload is its only client view)",
		},
	}
}

func notePrefixesByScope(scope string) []string {
	out := make([]string, 0, len(noteVocabulary))
	for _, p := range noteVocabulary {
		if p.Scope == scope {
			out = append(out, p.Prefix)
		}
	}
	return out
}

// â”€â”€ integrity â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

// notePrefixAmbiguitiesAcknowledged lists the prefix-of pairs this project has
// deliberately accepted. It is CURRENTLY EMPTY, and that is the strong claim:
// no declared prefix is a proper prefix of another, so a reader keyed on one
// purpose's prefix can never match a different purpose's note.
//
// A pair is only needed if a prefix must remain ambiguous; a NEW one panics at
// init and fails note_vocabulary_test.go, so the situation is discovered at
// build time rather than in production.
var notePrefixAmbiguitiesAcknowledged = [][2]string{}

// AmbiguousNotePrefixPairs returns every ordered (shorter, longer) declared pair
// where shorter is a proper prefix of longer.
func AmbiguousNotePrefixPairs() [][2]string {
	pairs := make([][2]string, 0)
	for _, a := range noteVocabulary {
		for _, b := range noteVocabulary {
			if a.Prefix == b.Prefix {
				continue
			}
			if strings.HasPrefix(b.Prefix, a.Prefix) {
				pairs = append(pairs, [2]string{a.Prefix, b.Prefix})
			}
		}
	}
	return pairs
}

func isAcknowledgedNoteAmbiguity(short, long string) bool {
	for _, pair := range notePrefixAmbiguitiesAcknowledged {
		if pair[0] == short && pair[1] == long {
			return true
		}
	}
	return false
}

// assertNoteVocabularyPolicy panics if the vocabulary is unsafe to use. It runs
// at init(), so a broken vocabulary cannot reach production silently.
func assertNoteVocabularyPolicy() {
	if len(noteVocabulary) == 0 {
		panic("note vocabulary: the catalogue is empty")
	}

	seenKeys := make(map[string]bool, len(noteVocabulary))
	seenPrefixes := make(map[string]bool, len(noteVocabulary))

	for _, p := range noteVocabulary {
		if p.Key == "" || p.Prefix == "" || p.Description == "" || p.Scope == "" || p.Direction == "" {
			panic("note vocabulary: incomplete entry for key " + p.Key + " / prefix " + p.Prefix)
		}
		if seenKeys[p.Key] {
			panic("note vocabulary: duplicate key " + p.Key)
		}
		seenKeys[p.Key] = true

		if seenPrefixes[p.Prefix] {
			panic("note vocabulary: duplicate prefix " + p.Prefix +
				" (two purposes sharing one prefix means a payment for one can satisfy the other)")
		}
		seenPrefixes[p.Prefix] = true

		if !strings.HasSuffix(p.Prefix, ":") {
			panic("note vocabulary: prefix " + p.Prefix + " must end with ':' (else a shorter purpose matches a longer one)")
		}
		for _, r := range strings.TrimSuffix(p.Prefix, ":") {
			upper := r >= 'A' && r <= 'Z'
			digit := r >= '0' && r <= '9'
			if !upper && !digit && r != '_' {
				panic("note vocabulary: prefix " + p.Prefix + " must be upper-case A-Z, 0-9 and '_' only")
			}
		}

		switch p.Scope {
		case NoteScopeMoneyDoor, NoteScopeCheckpoint, NoteScopeAuditLog:
		default:
			panic("note vocabulary: prefix " + p.Prefix + " has unknown scope " + p.Scope)
		}
		switch p.Direction {
		case NoteDirectionClientToVault, NoteDirectionVaultToVault, NoteDirectionVaultToChain:
		default:
			panic("note vocabulary: prefix " + p.Prefix + " has unknown direction " + p.Direction)
		}
	}

	for _, pair := range AmbiguousNotePrefixPairs() {
		if !isAcknowledgedNoteAmbiguity(pair[0], pair[1]) {
			panic("note vocabulary: " + pair[0] + " is a proper prefix of " + pair[1] +
				" - a reader using the shorter form would also match the longer note. " +
				"Acknowledge it in notePrefixAmbiguitiesAcknowledged or rename one prefix.")
		}
	}

	// A money door is the only place a CLIENT names a purpose, so every declared
	// money-door prefix must be exact-match addressable by the client.
	for _, p := range noteVocabulary {
		if p.Scope == NoteScopeMoneyDoor && !IsMoneyDoorPurpose(p.Prefix) {
			panic("note vocabulary: money door " + p.Prefix + " is not resolvable as a money-door purpose")
		}
	}
}

func init() { assertNoteVocabularyPolicy() }
