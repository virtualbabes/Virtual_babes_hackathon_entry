//go:build js && wasm

package main

import (
	"encoding/json"
	"time"
)

type BoardStateHash [32]byte

// Global constants for resilience
const indexerTimeout = 10 * time.Second

// PILLAR 2: Economic Safety Caps.
const MaxSinglePayoutMicro = 1000 * 1000000
const MaxAdminNotificationAmountMicro = 5000 * 1000000
const MaxAdminRewardAmountMicro = 5000 * 1000000
const MaxGovPayoutMicro = 2000 * 1000000

// GlobalSentiment aggregates playstyle data for NPC commentary
type GlobalSentiment struct {
	AvgAggressiveness float64            `json:"avg_aggressiveness"`
	AvgRiskTolerance  float64            `json:"avg_risk_tolerance"`
	DominantRules     map[string]float64 `json:"dominant_rules"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

// Club represents a player-owned organization with specialized shops.
type Club struct {
	ID                      string               `json:"id"`
	Name                    string               `json:"name"`
	OwnerWallet             string               `json:"owner_wallet"`
	Type                    string               `json:"type"`
	Territories             []string             `json:"territories"`
	RegionName              string               `json:"region_name,omitempty"`
	TreasuryMicro           uint64               `json:"treasury_micro"`
	Treasury                float64              `json:"treasury"`
	Commission              float64              `json:"commission_rate"`
	Inventory               map[string]int       `json:"inventory"`
	Staff                   map[string]string    `json:"staff"`
	ActiveBuffs             map[string]string    `json:"active_buffs"`
	BuffExpirations         map[string]time.Time `json:"buff_expirations"`
	Members                 map[string]time.Time `json:"members"`
	Leases                  map[string]*Lease    `json:"leases"`
	Mojo                    int                  `json:"club_mojo"`
	Jail                    map[int]ServerCard   `json:"jail"`
	LastActivity            time.Time            `json:"last_activity"`
	MojoStartOf24hWindow    int                  `json:"mojo_start_of_24h_window"`
	MojoWindowStartTime     time.Time            `json:"mojo_window_start_time"`
	LastBailoutAt           time.Time            `json:"last_bailout_at,omitempty"`
	LastHeistAt             time.Time            `json:"last_heist_at"`
	CreatedAt               time.Time            `json:"created_at"`
	AlliedClubID            string               `json:"allied_club_id,omitempty"`
	AllianceInviteID        string               `json:"alliance_invite_id,omitempty"`
	TaxHavenExpiresAt       time.Time            `json:"tax_haven_expires_at,omitempty"`
	AllianceInviteExpiresAt time.Time            `json:"alliance_invite_expires_at,omitempty"`
	MutationSuccesses       int                  `json:"mutation_successes"`
	MutationFailures        int                  `json:"mutation_failures"`
	MutationHistory         []string             `json:"mutation_history"`
	CommissionHistory       []CommissionEvent    `json:"commission_history,omitempty"`
}

// CommissionEvent records a dividend payout from an alliance partner.
type CommissionEvent struct {
	Timestamp  int64   `json:"timestamp"`
	SourceClub string  `json:"source_club"`
	Type       string  `json:"type"`
	Amount     float64 `json:"amount"`
}

// Lease represents a card available for temporary use within a club.
type Lease struct {
	ID            string    `json:"id"`
	LenderWallet  string    `json:"lender_wallet"`
	CardID        int       `json:"card_id"`
	CardName      string    `json:"card_name"`
	Price         float64   `json:"price"`
	DurationHours int       `json:"duration_hours"`
	ExpiresAt     time.Time `json:"expires_at,omitempty"`
	Borrower      string    `json:"borrower_wallet,omitempty"`
	ClubID        string    `json:"club_id"`
}

// Matchmaking Logic
type QueueEntry struct {
	ClientID   string    `json:"client_id"`
	Wallet     string    `json:"wallet"`
	Reputation int       `json:"reputation"`
	DeckRating string    `json:"deck_rating"`
	JoinedAt   time.Time `json:"joined_at"`
}

// LinkedWallet represents a non-AVM wallet linked to a primary AVM wallet.
type LinkedWallet struct {
	Address   string    `json:"address"`
	Chain     string    `json:"chain"`
	Verified  bool      `json:"verified"`
	Timestamp time.Time `json:"timestamp"`
}

// WalletLinkInfo stores the primary AVM wallet and its linked non-AVM wallets.
type WalletLinkInfo struct {
	PrimaryAVMWallet string         `json:"primary_avm_wallet"`
	Linked           []LinkedWallet `json:"linked_wallets"`
}

// HoldingBonus defines a multiplier for a specific reward if a player holds a certain asset.
type HoldingBonus struct {
	HoldingAssetID string  `json:"holding_asset_id"`
	Network        string  `json:"network"`
	Multiplier     float64 `json:"multiplier"`
	MinAmount      uint64  `json:"min_amount"`
}

// FaceplateStats defines the RPG modifiers provided by cosmetic items.
type FaceplateStats struct {
	MojoBonus    int
	CunningBonus int
}

// FaceplateRegistry maps legacy cosmetic IDs to functional social simulation bonuses.
var FaceplateRegistry = map[string]FaceplateStats{
	"faceplate_neon_vibe":   {MojoBonus: 15, CunningBonus: 5},
	"faceplate_shadow":      {MojoBonus: 5, CunningBonus: 20},
	"faceplate_governor":    {MojoBonus: 50, CunningBonus: 10},
	"faceplate_placeholder": {MojoBonus: 0, CunningBonus: 0},
}

// GetEffectiveMojo returns base mojo plus cosmetic bonuses.
func (p Player) GetEffectiveMojo() int {
	if p.EquippedFaceplate != "" {
		if fp, exists := FaceplateRegistry[p.EquippedFaceplate]; exists {
			return p.Mojo + fp.MojoBonus
		}
	}
	return p.Mojo
}

// UseItemData defines the payload for the "use_item" WebSocket message.
type UseItemData struct {
	ItemID          string `json:"item_id"`
	TargetCardID    int    `json:"target_card_id,omitempty"`
	TargetGridIndex int    `json:"target_grid_index,omitempty"`
	TargetClubID    string `json:"target_club_id,omitempty"`
	TargetWallet    string `json:"target_wallet,omitempty"`
}

// BailCardData defines the payload for the "bail_card" WebSocket message.
type BailCardData struct {
	CardID  int    `json:"card_id"`
	ClubID  string `json:"club_id"`
	TxID    string `json:"txid"`
	Network string `json:"network"`
}

// Envelope is the standard wrapper for all messages.
type Envelope struct {
	Type    string          `json:"type"`
	FromID  string          `json:"from_id"`
	ToID    string          `json:"to_id,omitempty"`
	Payload json.RawMessage `json:"payload"`
}

// ChallengeData handles the matchmaking handshake.
type ChallengeData struct {
	Action    string          `json:"action"`
	Deck      []int           `json:"deck,omitempty"`
	Avatar    string          `json:"avatar,omitempty"`
	Gloat     string          `json:"gloat,omitempty"`
	Rules     map[string]bool `json:"rules,omitempty"`
	Wanted    int             `json:"wanted_level,omitempty"`
	Faceplate string          `json:"faceplate,omitempty"`
}

// MoveData synchronizes gameplay actions between two human players.
type MoveData struct {
	GridIndex   int    `json:"grid_index"`
	CardID      int    `json:"card_id"`
	Power       [4]int `json:"power"`
	PlayerIndex int    `json:"player_index"`
}

// ReportGloatData captures information about a reported gloat message.
type ReportGloatData struct {
	OpponentClientID string `json:"opponent_client_id"`
	GloatText        string `json:"gloat_text"`
}

// NetworkConfig holds the configuration details for a specific blockchain network.
type NetworkConfig struct {
	NetworkName    string            `json:"network_name"`
	ExplorerURL    string            `json:"explorer_url"`
	IndexerURLs    []string          `json:"indexer_urls"`
	NodeURLs       []string          `json:"node_urls"`
	FaucetURL      string            `json:"faucet_url"`
	AssetID        string            `json:"asset_id"`
	AppID          string            `json:"app_id"`
	ChainID        string            `json:"chain_id"`
	PowerDivisor   float64           `json:"power_divisor"`
	PowerBase      int               `json:"power_base"`
	IPFSGatewayURL string            `json:"ipfs_gateway_url"`
	IPFSAPIKey     string            `json:"ipfs_api_key"`
	IPFSHeaders    map[string]string `json:"ipfs_headers"`
}

// ServerCard mirrors the client Card for verification logic.
type ServerCard struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Power         [4]int    `json:"power"`
	Image         string    `json:"image"`
	Rarity        float64   `json:"rarity"`
	Owner         int       `json:"owner"`
	Artifact      int       `json:"artifact"`
	Fatigue       int       `json:"fatigue"`
	Loyalty       int       `json:"loyalty"`
	LastUpdated   time.Time `json:"last_updated"`
	MetadataValid bool      `json:"metadata_valid"`
	Mood          string    `json:"mood"`
	Fallen        bool      `json:"fallen"`
	Scars         []string  `json:"scars"`
	EquippedItems []string  `json:"equipped_items"`
}

// TournamentSummary represents a finalized tournament for archival.
type TournamentSummary struct {
	ID               string            `json:"id"`
	Timestamp        time.Time         `json:"timestamp"`
	PotMicro         uint64            `json:"pot_micro"`
	Winner           string            `json:"winner"`
	IsVerified       bool              `json:"is_verified"`
	ReceiptsVerified bool              `json:"receipts_verified"`
	PayoutsHash      string            `json:"payouts_hash,omitempty"`
	Checksum         string            `json:"checksum,omitempty"`
	Links            []string          `json:"links,omitempty"`
	Matches          []TournamentMatch `json:"matches"`
}

type MetadataAttribute struct {
	TraitType string      `json:"trait_type"`
	Value     interface{} `json:"value"`
}

type ARC72Metadata struct {
	Name       string              `json:"name"`
	Image      string              `json:"image"`
	Attributes []MetadataAttribute `json:"attributes"`
}

// MatchState tracks an ongoing game on the server for win verification.
type MatchState struct {
	P1ID              string          `json:"p1_id"`
	P2ID              string          `json:"p2_id"`
	P1Wallet          string          `json:"p1_wallet"`
	P2Wallet          string          `json:"p2_wallet"`
	TournamentID      string          `json:"tournament_id,omitempty"`
	TournamentMatchID string          `json:"match_id,omitempty"`
	P1Deck            []int           `json:"p1_deck"`
	P1Avatar          string          `json:"p1_avatar"`
	P1Gloat           string          `json:"p1_gloat"`
	P2Deck            []int           `json:"p2_deck"`
	P2Avatar          string          `json:"p2_avatar"`
	StartTime         time.Time       `json:"start_time"`
	MatchRating       string          `json:"match_rating"`
	BoardMoods        [9]string       `json:"board_moods"`
	P2Gloat           string          `json:"p2_gloat"`
	Board             [9]*ServerCard  `json:"board"`
	Rules             map[string]bool `json:"rules"`
	IsFinished        bool            `json:"is_finished"`
	Spectators        []string        `json:"spectators"`
	Multiplayer       bool            `json:"multiplayer"`
	P1WantedLevel     int             `json:"p1_wanted_level"`
	P2WantedLevel     int             `json:"p2_wanted_level"`
	P1Cunning         int             `json:"p1_cunning"`
	P1Nurturing       int             `json:"p1_nurturing"`
	P2Cunning         int             `json:"p2_cunning"`
	P2Nurturing       int             `json:"p2_nurturing"`
	P1RegionalBoost   bool            `json:"p1_regional_boost"`
	P2RegionalBoost   bool            `json:"p2_regional_boost"`
	P1CoalitionBoost  bool            `json:"p1_coalition_boost"`
	P2CoalitionBoost  bool            `json:"p2_coalition_boost"`
	FinalScores       [2]int
	CapturedCards     []CapturedCardInfo        `json:"captured_cards,omitempty"`
	Round             int                       `json:"round"`
	TerritoryID       string                    `json:"territory_id,omitempty"`
	ActiveItemBuffs   map[string]map[string]int `json:"active_item_buffs"`
	IsBountyMatch     bool
	WagersMicro       uint64         `json:"wagers_micro"`
	BoardStateHash    BoardStateHash
}

// AuthoritativeFrame represents the full authoritative state update sent to the client.
type AuthoritativeFrame struct {
	SequenceID uint64         `json:"sequence_id"`
	MoveIntent MoveData       `json:"move_intent"`
	StateHash  BoardStateHash `json:"state_hash"`
}

// MutationEvent records a single gene-editing action for forensic auditing.
type MutationEvent struct {
	Timestamp int64  `json:"timestamp"`
	Type      string `json:"type"`
	CardID    int    `json:"card_id"`
	Details   string `json:"details"`
}

// CapturedCardInfo tracks details of a card that was flipped during a match.
type CapturedCardInfo struct {
	CardID                int
	OriginalOwnerWallet   string
	CapturingPlayerWallet string
	CaptureType           string
	GridIndex             int
	Round                 int
}

// MatchHistory stores the result of a completed game for reward verification.
type MatchHistory struct {
	WinnerID          string                    `json:"winner_id"`
	Opponent          string                    `json:"opponent_wallet"`
	TournamentID      string                    `json:"tournament_id,omitempty"`
	TournamentMatchID string                    `json:"match_id,omitempty"`
	ReceiptTxID       string                    `json:"receipt_txid,omitempty"`
	Scores            [2]int                    `json:"scores"`
	Timestamp         time.Time                 `json:"timestamp"`
	WinnerIndex       int                       `json:"winner_index"`
	IsBountyMatch     bool                      `json:"is_bounty_match,omitempty"`
	BountyRewardMicro uint64                    `json:"bounty_reward_micro,omitempty"`
	P1WantedLevel     int                       `json:"p1_wanted_level"`
	P2WantedLevel     int                       `json:"p2_wanted_level"`
	P1Cunning         int                       `json:"p1_cunning"`
	P2Cunning         int                       `json:"p2_cunning"`
	P1Nurturing       int                       `json:"p1_nurturing"`
	P2Nurturing       int                       `json:"p2_nurturing"`
	ActiveItemBuffs   map[string]map[string]int `json:"active_item_buffs,omitempty"`
	CapturedCards     []CapturedCardInfo        `json:"captured_cards,omitempty"`
}

// PlayerStats tracks the performance and reliability of a player.
type PlayerStats struct {
	Wallet                       string              `json:"wallet"`
	Wins                         int                 `json:"wins"`
	DNFs                         int                 `json:"dnfs"`
	DisconnectStreak             int                 `json:"disconnect_streak"`
	BanExpires                   time.Time           `json:"ban_expires"`
	GloatBannedUntil             time.Time           `json:"gloat_banned_until"`
	EquippedFaceplate            string              `json:"equipped_faceplate"`
	Reputation                   int                 `json:"reputation"`
	Mojo                         int                 `json:"mojo"`
	SocialRank                   string              `json:"social_rank"`
	JobRole                      string              `json:"job_role"`
	EmployerClubID               string              `json:"employer_id"`
	Salary                       uint64              `json:"salary"`
	AuctionsWon                  int                 `json:"auctions_won"`
	LastSalaryPayment            time.Time           `json:"last_salary_payment"`
	Inventory                    map[string]int      `json:"inventory"`
	History                      []MatchHistory      `json:"match_history"`
	MarketTokens                 uint64              `json:"market_tokens"`
	Relationships                map[string]int      `json:"relationships"`
	BestRating                   string              `json:"best_rating"`
	Achievements                 []string            `json:"achievements"`
	Portfolio                    map[string]uint64   `json:"portfolio"`
	WantedLevel                  int                 `json:"wanted_level"`
	HeistAttempts                int                 `json:"heist_attempts"`
	Cunning                      int                 `json:"cunning"`
	Nurturing                    int                 `json:"nurturing"`
	JailedCards                  map[int]string      `json:"jailed_cards"`
	CapturedOutlaws              map[string]bool     `json:"captured_outlaws"`
	FavoriteCardID               int                 `json:"favorite_card_id"`
	KidnappedCards               map[int]string      `json:"kidnapped_cards"`
	HeldHostageCards             map[int]string      `json:"held_hostage_cards"`
	LastClaimedYield             map[string]uint64   `json:"last_claimed_yield"`
	GhostProtocolExpiresAt       time.Time           `json:"ghost_protocol_expires_at"`
	LastDeepScanAt               time.Time           `json:"last_deep_scan_at"`
	DistrictScannerExpiresAt     time.Time           `json:"district_scanner_expires_at"`
	DisruptorCooldownAt          time.Time           `json:"disruptor_cooldown_at"`
	CloakDisruptedUntil          time.Time           `json:"cloak_disrupted_until"`
	HasCyberJammer               bool                `json:"has_cyber_jammer"`
	HeistAlarmsJammerCount       int                 `json:"heist_alarms_jammer_count"`
	HasMutationInsurance         bool                `json:"has_mutation_insurance"`
	AuditedClubs                 map[string]bool     `json:"audited_clubs"`
	RumorCount                   int                 `json:"rumor_count"`
	Aggressiveness               float64             `json:"aggressiveness"`
	RiskTolerance                float64             `json:"risk_tolerance"`
	PreferredRules               map[string]int      `json:"preferred_rules"`
	Moods                        map[string]int      `json:"moods"`
	Playstyle                    PlaystyleTendencies `json:"playstyle"`
	ActiveItemBuffs              map[string]int      `json:"active_item_buffs"`
	TotalDonated                 uint64              `json:"total_donated"`
	ReparationsReceivedCount     int                 `json:"reparations_received_count"`
	IsMojoStabilizerActive       bool                `json:"is_mojo_stabilizer_active"`
	MojoDecayRate                float64             `json:"mojo_decay_rate"`
	RaidInsuranceExpiresAt       time.Time           `json:"raid_insurance_expires_at"`
	RaidInsuranceClaimsRemaining int                 `json:"raid_insurance_claims_remaining"`
	BountyHunterBondMicro        uint64              `json:"bounty_hunter_bond_micro"`
	BountyHunterLicenseExpiresAt time.Time           `json:"bounty_hunter_license_expires_at"`
	ArenaVouchers                uint64              `json:"arena_vouchers"`
	ActiveUnderworldContractID   string              `json:"active_underworld_contract_id"`
	RecoveryChallengeCardID      int                 `json:"recovery_challenge_card_id"`
	RecoveryChallengeWins        int                 `json:"recovery_challenge_wins"`
	RecoveryBounties             map[int]uint64      `json:"recovery_bounties"`
	MarketFrozenUntil            time.Time           `json:"market_frozen_until"`
	MutationHistory              []MutationEvent     `json:"mutation_history"`
	CareerXP                     *CareerXP           `json:"career_xp,omitempty"`
	Rivalry                      *RivalryState       `json:"rivalry,omitempty"`
	BountyLicenseActive          bool                `json:"bounty_license_active"`
	ArcNetActive                 bool                `json:"arc_net_active,omitempty"`
	LiquiditySamples             []uint64            `json:"liquidity_samples,omitempty"`
	LiquidityWindowMin           int                 `json:"liquidity_window_min"`
	DemotionWarningAt            time.Time           `json:"demotion_warning_at,omitempty"`
	AvgSustainedMicro            uint64              `json:"avg_sustained_micro"`
	ActiveJusticeMissionID       string              `json:"active_justice_mission_id,omitempty"`
	Buffs                        map[string]bool     `json:"buffs,omitempty"`
}

// ItemDef is the base definition for all purchasable shop items.
type ItemDef struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	CostMicro     uint64 `json:"cost_micro"`
	Category      string `json:"category"`
	MaxStack      int    `json:"max_stack"`
	Recurring     bool   `json:"recurring"`
	RecurringDays int    `json:"recurring_days"`
}

// CareerXP tracks a player's progression across career paths.
type CareerXP struct {
	CurrentPath     string            `json:"current_path"`
	Level           int               `json:"level"`
	XP              uint64            `json:"xp"`
	Bonuses         map[string]float64 `json:"bonuses"`
	LastPromotionAt time.Time         `json:"last_promotion_at"`
}

// RivalryState tracks competitive relationships between players.
type RivalryState struct {
	OpponentWallet   string    `json:"opponent_wallet"`
	Status           string    `json:"status"`
	ConsecutiveWins  int       `json:"consecutive_wins"`
	LastRivalryAt    time.Time `json:"last_rivalry_at"`
	BetMicro         uint64    `json:"bet_micro"`
	HonorPoints      int       `json:"honor_points"`
}

// JusticeCategory and UnderworldCategory are shop category constants.
const (
	JusticeCategory    = "JUSTICE"
	UnderworldCategory = "UNDERWORLD"
)

// PlaystyleTendencies captures observed player behaviors for Collective Intelligence.
type PlaystyleTendencies struct {
	Aggressiveness     float64            `json:"aggressiveness"`
	RiskTolerance      float64            `json:"risk_tolerance"`
	PreferredRules     map[string]float64 `json:"preferred_rules"`
	PreferredCardMoods map[string]float64 `json:"preferred_card_moods"`
	FavoriteCardID     int                `json:"favorite_card_id"`
	PreferredItems     map[string]float64 `json:"preferred_items"`
}

// ConsolePlatform defines supported hardware ecosystems for the Phase 4 expansion.
type ConsolePlatform string

const (
	PlatformXbox        ConsolePlatform = "XBOX_SERIES_X"
	PlatformPlayStation ConsolePlatform = "PLAYSTATION_5"
	PlatformNintendo    ConsolePlatform = "NINTENDO_SWITCH_NEXT"
)

// ConsoleAssetReceipt represents a secure purchase or lease confirmation from an external platform.
type ConsoleAssetReceipt struct {
	PlatformRef    ConsolePlatform `json:"platform_ref"`
	ConsoleUID     string          `json:"console_user_uid"`
	DlcProductCode string          `json:"dlc_product_code"`
	IsLeaseAction  bool            `json:"is_lease_action"`
	LeaseDuration  int64           `json:"lease_duration_blocks"`
}

// CardBundle represents a set of items listed together in an auction.
type CardBundle struct {
	CardID      int    `json:"card_id"`
	WeaponID    string `json:"weapon_id,omitempty"`
	FaceplateID string `json:"faceplate_id,omitempty"`
}

// Auction represents a live listing in the Art Gallery.
type Auction struct {
	ID                   string     `json:"id"`
	SellerWallet         string     `json:"seller_wallet"`
	SellerName           string     `json:"seller_name"`
	Bundle               CardBundle `json:"bundle"`
	CurrentBid           uint64     `json:"current_bid"`
	HighestBidder        string     `json:"highest_bidder"`
	HighestBidderName    string     `json:"highest_bidder_name"`
	EndsAt               time.Time  `json:"ends_at"`
	TerritoryID          string     `json:"territory_id"`
	HighestBidIsApproved bool       `json:"highest_bid_is_approved"`
}

// Loan represents a collateralized loan from the Second-Hand Store.
type Loan struct {
	ID               string     `json:"id"`
	BorrowerWallet   string     `json:"borrower_wallet"`
	BorrowerName     string     `json:"borrower_name"`
	CollateralBundle CardBundle `json:"collateral_bundle"`
	LoanAmount       uint64     `json:"loan_amount"`
	RepaymentAmount  uint64     `json:"repayment_amount"`
	DueAt            time.Time  `json:"due_at"`
	Status           string     `json:"status"`
	TerritoryID      string     `json:"territory_id"`
}

// Rumor represents an active rumor affecting an entity's share price.
type Rumor struct {
	ID             string    `json:"id"`
	SpreaderWallet string    `json:"spreader_wallet"`
	TargetWallet   string    `json:"target_wallet"`
	Type           string    `json:"type"`
	Strength       float64   `json:"strength"`
	ExpiresAt      time.Time `json:"expires_at"`
}

// KidnapState tracks the details of an active kidnapping for recovery logic.
type KidnapState struct {
	VictimWallet string    `json:"victim_wallet"`
	PerpWallet   string    `json:"perp_wallet"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// TournamentMatch represents a single duel within the bracket.
type TournamentMatch struct {
	ID          string `json:"match_id"`
	P1          string `json:"p1"`
	P2          string `json:"p2"`
	Winner      string `json:"winner,omitempty"`
	Round       int    `json:"round"`
	ReceiptTxID string `json:"receipt_txid,omitempty"`
}

// TournamentState tracks the progress of an automated event.
type TournamentState struct {
	Active       bool              `json:"active"`
	ID           string            `json:"id"`
	Matches      []TournamentMatch `json:"matches"`
	CurrentRound int               `json:"current_round"`
	Participants []string          `json:"participants"`
	PotMicro     uint64            `json:"pot_micro"`
	BuyInMicro   uint64            `json:"buy_in_micro"`
	IsBuyInMode  bool              `json:"is_buy_in_mode"`
	OpenTime     time.Time         `json:"open_time"`
	Winner       string            `json:"winner"`
}
