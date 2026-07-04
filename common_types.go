//go:build !js && !wasm

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
const MaxAdminNotificationAmountMicro = 5000 * 1000000 // Cap admin notifications at 5,000 VBV
const MaxAdminRewardAmountMicro = 5000 * 1000000       // Cap admin-set reward amounts at 5,000 VBV
const MaxGovPayoutMicro = 2000 * 1000000               // 2,000 $VBV Cap for Dividends

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
	TreasuryMicro           uint64               `json:"treasury_micro"` // PILLAR 2: Integer Supremacy
	Treasury                float64              `json:"treasury"`       // PILLAR 3: Floating point for display UI ONLY
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
	MojoStartOf24hWindow    int                  `json:"mojo_start_of_24h_window"`  // Mojo value at the start of the current 24h window
	MojoWindowStartTime     time.Time            `json:"mojo_window_start_time"`    // Timestamp when the current 24h window began
	LastBailoutAt           time.Time            `json:"last_bailout_at,omitempty"` // PILLAR 1: stimulus tracking
	LastHeistAt             time.Time            `json:"last_heist_at"`
	CreatedAt               time.Time            `json:"created_at"`
	AlliedClubID            string               `json:"allied_club_id,omitempty"`             // ID of the allied club
	AllianceInviteID        string               `json:"alliance_invite_id,omitempty"`         // ID of club that sent an invitation
	TaxHavenExpiresAt       time.Time            `json:"tax_haven_expires_at,omitempty"`       // PILLAR 1: Political Influence
	AllianceInviteExpiresAt time.Time            `json:"alliance_invite_expires_at,omitempty"` // PILLAR 1: Invitation TTL
	MutationSuccesses       int                  `json:"mutation_successes"`
	MutationFailures        int                  `json:"mutation_failures"`            // PILLAR 6: Mutation failures
	MutationHistory         []string             `json:"mutation_history"`             // Tracks mutation events
	CommissionHistory       []CommissionEvent    `json:"commission_history,omitempty"` // PILLAR 1: Alliance dividend logs
}

// CommissionEvent records a dividend payout from an alliance partner.
type CommissionEvent struct {
	Timestamp  int64   `json:"timestamp"`
	SourceClub string  `json:"source_club"`
	Type       string  `json:"type"`   // "VECTOR", "MOOD", "LOYALTY"
	Amount     float64 `json:"amount"` // Base $VBV
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
	Chain     string    `json:"chain"` // e.g., "ETH", "POLY", "SOL"
	Verified  bool      `json:"verified"`
	Timestamp time.Time `json:"timestamp"` // When it was linked/verified
}

// WalletLinkInfo stores the primary AVM wallet and its linked non-AVM wallets.
type WalletLinkInfo struct {
	PrimaryAVMWallet string         `json:"primary_avm_wallet"`
	Linked           []LinkedWallet `json:"linked_wallets"`
}

// HoldingBonus defines a multiplier for a specific reward if a player holds a certain asset.
type HoldingBonus struct {
	HoldingAssetID string  `json:"holding_asset_id"` // The NFT/Token required to be held
	Network        string  `json:"network"`          // Chain to check (VOI or ALGO)
	Multiplier     float64 `json:"multiplier"`       // Reward boost (e.g., 1.1 for 10% bonus)
	MinAmount      uint64  `json:"min_amount"`       // Minimum micro-units required to qualify
}

// FaceplateStats defines the RPG modifiers provided by cosmetic items.
type FaceplateStats struct {
	MojoBonus    int
	CunningBonus int
}

// FaceplateRegistry maps legacy cosmetic IDs to functional social simulation bonuses.
var FaceplateRegistry = map[string]FaceplateStats{ // PILLAR 5: Isomorphic Parity. Shared registry.
	"faceplate_neon_vibe":   {MojoBonus: 15, CunningBonus: 5},
	"faceplate_shadow":      {MojoBonus: 5, CunningBonus: 20},
	"faceplate_governor":    {MojoBonus: 50, CunningBonus: 10},
	"faceplate_placeholder": {MojoBonus: 0, CunningBonus: 0},
}

// GetEffectiveCunning returns base cunning plus cosmetic bonuses.
func (p PlayerStats) GetEffectiveCunning() int {
	eff := p.Cunning
	if p.EquippedFaceplate != "" {
		if fp, exists := FaceplateRegistry[p.EquippedFaceplate]; exists {
			eff += fp.CunningBonus
		}
	}
	// Infamy Penalty: Every 5 levels of Wanted Level reduces effective Cunning by 1
	penalty := p.WantedLevel / 5
	eff -= penalty
	if eff < 0 {
		eff = 0
	}
	return eff
}

// GetEffectiveMojo returns base mojo plus cosmetic bonuses.
func (p PlayerStats) GetEffectiveMojo() int {
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
	TargetCardID    int    `json:"target_card_id,omitempty"`    // For card-specific buffs (e.g., Stim, Pledge)
	TargetGridIndex int    `json:"target_grid_index,omitempty"` // For board-specific buffs (e.g., Mood Catalyst)
	TargetClubID    string `json:"target_club_id,omitempty"`    // For organizational intelligence (e.g., Cyber-Audit)
	TargetWallet    string `json:"target_wallet,omitempty"`     // PILLAR 3: Targeted Intelligence (e.g., Cloak Disruptor)
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
	Type    string          `json:"type"`    // "lobby_update", "challenge", "move", "chat", "identity", "vault_update", "rules_update", "rewards_update", "maintenance_update", "ping", "pong", "report_gloat", "admin_notification"
	FromID  string          `json:"from_id"` // Sender ID
	ToID    string          `json:"to_id,omitempty"`
	Payload json.RawMessage `json:"payload"` // Flexible JSON content
}

// ChallengeData handles the matchmaking handshake.
type ChallengeData struct {
	Action    string          `json:"action"` // "invite", "accept", "decline", "sync_back"
	Deck      []int           `json:"deck,omitempty"`
	Avatar    string          `json:"avatar,omitempty"`
	Gloat     string          `json:"gloat,omitempty"`
	Rules     map[string]bool `json:"rules,omitempty"`
	Wanted    int             `json:"wanted_level,omitempty"`
	Faceplate string          `json:"faceplate,omitempty"` // PILLAR 4: Handshake sync
}

// MoveData synchronizes gameplay actions between two human players.
type MoveData struct {
	GridIndex   int    `json:"grid_index"`
	CardID      int    `json:"card_id"`
	Power       [4]int `json:"power"`
	PlayerIndex int    `json:"player_index"` // 0 or 1
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
	AssetID        string            `json:"asset_id"` // The primary game asset ID on this network
	AppID          string            `json:"app_id"`   // The main game smart contract ID on this network
	ChainID        string            `json:"chain_id"` // WalletConnect / CAIP-2 Chain ID
	PowerDivisor   float64           `json:"power_divisor"`
	PowerBase      int               `json:"power_base"`
	IPFSGatewayURL string            `json:"ipfs_gateway_url"` // Custom IPFS gateway for metadata
	IPFSAPIKey     string            `json:"ipfs_api_key"`     // API key for authenticated gateways
	IPFSHeaders    map[string]string `json:"ipfs_headers"`     // Custom headers for IPFS requests
}

// ServerCard mirrors the client Card for verification logic.
type ServerCard struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Power         [4]int    `json:"power"`
	Image         string    `json:"image"`
	Rarity        float64   `json:"rarity"` // Power multiplier based on supply
	Owner         int       `json:"owner"`
	Artifact      int       `json:"artifact"`
	Fatigue       int       `json:"fatigue"`        // 0-100
	Loyalty       int       `json:"loyalty"`        // 0-100
	LastUpdated   time.Time `json:"last_updated"`   // TTL tracking for cache refresh
	MetadataValid bool      `json:"metadata_valid"` // Indicates if metadata was successfully parsed
	Mood          string    `json:"mood"`           // Volatile, Serene, Spirited, Grounded
	Fallen        bool      `json:"fallen"`         // Pillar 7: Underworld status
	Scars         []string  `json:"scars"`          // PILLAR 6: Procedure failure history
	EquippedItems []string  `json:"equipped_items"` // PILLAR 5: Hardware trap/item overlay indicators
}

// TournamentSummary represents a finalized tournament for archival.
type TournamentSummary struct {
	ID               string            `json:"id"`
	Timestamp        time.Time         `json:"timestamp"`
	PotMicro         uint64            `json:"pot_micro"` // PILLAR 2: Integer Supremacy
	Winner           string            `json:"winner"`
	IsVerified       bool              `json:"is_verified"`            // Indicates successful blockchain reconstruction
	ReceiptsVerified bool              `json:"receipts_verified"`      // Indicates VBT_WIN receipts were found for all matches
	PayoutsHash      string            `json:"payouts_hash,omitempty"` // SHA256 of reward Group IDs
	Checksum         string            `json:"checksum,omitempty"`     // SHA256 of full match data
	Links            []string          `json:"links,omitempty"`        // TxIDs for additional match data
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
	P1Wallet          string          `json:"p1_wallet"` // Snapshotted for penalty calculation stability
	P2Wallet          string          `json:"p2_wallet"`
	TournamentID      string          `json:"tournament_id,omitempty"` // Instance ID of the tournament
	TournamentMatchID string          `json:"match_id,omitempty"`      // PILLAR 3: Standardized match indexing
	P1Deck            []int           `json:"p1_deck"`                 // Card IDs in P1's deck
	P1Avatar          string          `json:"p1_avatar"`
	P1Gloat           string          `json:"p1_gloat"`
	P2Deck            []int           `json:"p2_deck"` // Card IDs in P2's deck
	P2Avatar          string          `json:"p2_avatar"`
	StartTime         time.Time       `json:"start_time"`
	MatchRating       string          `json:"match_rating"`
	BoardMoods        [9]string       `json:"board_moods"` // Moods assigned to specific tiles
	P2Gloat           string          `json:"p2_gloat"`
	Board             [9]*ServerCard  `json:"board"`
	Rules             map[string]bool `json:"rules"`
	IsFinished        bool            `json:"is_finished"`
	Spectators        []string        `json:"spectators"`  // Client IDs spectating this match
	Multiplayer       bool            `json:"multiplayer"` // Explicitly flag as multiplayer to prevent local AI loops
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
	CapturedCards     []CapturedCardInfo        `json:"captured_cards,omitempty"` // Tracking for jailing
	Round             int                       `json:"round"`                    // Match round (isolation for Sudden Death)
	TerritoryID       string                    `json:"territory_id,omitempty"`   // The territory where the match is played
	ActiveItemBuffs   map[string]map[string]int `json:"active_item_buffs"`        // PlayerID -> ItemID -> MatchesRemaining
	IsBountyMatch     bool
	WagersMicro       uint64         `json:"wagers_micro"` // PILLAR 2: Spectator wagering pool
	BoardStateHash    BoardStateHash // PILLAR 4: Deterministic Sync.
}

// AuthoritativeFrame represents the full authoritative state update sent to the client.
type AuthoritativeFrame struct {
	SequenceID uint64         `json:"sequence_id"`
	MoveIntent MoveData       `json:"move_intent"` // The actual move data
	StateHash  BoardStateHash `json:"state_hash"`  // Hash of the board state after the move
}

// MutationEvent records a single gene-editing action for forensic auditing.
type MutationEvent struct {
	Timestamp int64  `json:"timestamp"`
	Type      string `json:"type"` // "VECTOR", "MOOD", "LOYALTY", "SCAR"
	CardID    int    `json:"card_id"`
	Details   string `json:"details"`
}

// CapturedCardInfo tracks details of a card that was flipped during a match.
type CapturedCardInfo struct {
	CardID                int
	OriginalOwnerWallet   string // Wallet of the player who originally owned the card
	CapturingPlayerWallet string // Wallet of the player who captured the card
	CaptureType           string // "BASIC", "SAME", "POWER_UP", "COMBO"
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
	WinnerIndex       int                       `json:"winner_index"` // 0 for P1, 1 for P2
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
	Mojo                         int                 `json:"mojo"`                // Social standing for Club unlocks
	SocialRank                   string              `json:"social_rank"`         // e.g., "Nobody", "Regular", "Icon"
	JobRole                      string              `json:"job_role"`            // Manager, Security, Clerk, Freelancer
	EmployerClubID               string              `json:"employer_id"`         // PILLAR 3: Standardized Org ID
	Salary                       uint64              `json:"salary"`              // Micro-units of $VBV per payment cycle
	AuctionsWon                  int                 `json:"auctions_won"`        // Total Art Gallery victories
	LastSalaryPayment            time.Time           `json:"last_salary_payment"` // Timestamp of last payment
	Inventory                    map[string]int      `json:"inventory"`           // ItemID -> Quantity
	History                      []MatchHistory      `json:"match_history"`       // Historical records from blockchain
	MarketTokens                 uint64              `json:"market_tokens"`       // Equity from liquidated loans
	Relationships                map[string]int      `json:"relationships"`       // Character Name -> Score (0-100)
	BestRating                   string              `json:"best_rating"`
	Achievements                 []string            `json:"achievements"`     // List of unlocked IDs
	Portfolio                    map[string]uint64   `json:"portfolio"`        // EntityID -> Shares (in micro-shares)
	WantedLevel                  int                 `json:"wanted_level"`     // Risk factor for heists
	HeistAttempts                int                 `json:"heist_attempts"`   // Number of times player attempted a heist
	Cunning                      int                 `json:"cunning"`          // Success modifier for criminal actions
	Nurturing                    int                 `json:"nurturing"`        // Success modifier for garden/donations
	JailedCards                  map[int]string      `json:"jailed_cards"`     // CardID -> ClubID (cards currently in jail)
	CapturedOutlaws              map[string]bool     `json:"captured_outlaws"` // PILLAR 3: Unique bounty tracking
	FavoriteCardID               int                 `json:"favorite_card_id"` // The card ID designated as favorite
	KidnappedCards               map[int]string      `json:"kidnapped_cards"`  // CardID -> VictimWallet
	HeldHostageCards             map[int]string      `json:"held_hostage_cards"`
	LastClaimedYield             map[string]uint64   `json:"last_claimed_yield"`          // PILLAR 1: Dividend tracking
	GhostProtocolExpiresAt       time.Time           `json:"ghost_protocol_expires_at"`   // Signal scrambling duration
	LastDeepScanAt               time.Time           `json:"last_deep_scan_at"`           // PILLAR 3: Intel-Agent cooldown
	DistrictScannerExpiresAt     time.Time           `json:"district_scanner_expires_at"` // Active scanning duration
	DisruptorCooldownAt          time.Time           `json:"disruptor_cooldown_at"`       // PILLAR 3: Hunter item cooldown
	CloakDisruptedUntil          time.Time           `json:"cloak_disrupted_until"`       // PILLAR 3: Temporarily revealed by hunter
	HasCyberJammer               bool                `json:"has_cyber_jammer"`            // Suppresses Sabotage Warning
	HeistAlarmsJammerCount       int                 `json:"heist_alarms_jammer_count"`   // Count of successful jammer uses
	HasMutationInsurance         bool                `json:"has_mutation_insurance"`      // PILLAR 6: 100% Mutation Success
	AuditedClubs                 map[string]bool     `json:"audited_clubs"`               // Unique clubs targeted by Cyber-Audit
	RumorCount                   int                 `json:"rumor_count"`                 // Number of rumors spread by this player
	Aggressiveness               float64             `json:"aggressiveness"`              // 0-1 scale of aggressive play
	RiskTolerance                float64             `json:"risk_tolerance"`              // 0-1 scale of risk-taking
	PreferredRules               map[string]int      `json:"preferred_rules"`             // Rule name -> usage count
	Moods                        map[string]int      `json:"moods"`                       // Mood -> count
	Playstyle                    PlaystyleTendencies `json:"playstyle"`                   // Observed playstyle tendencies
	ActiveItemBuffs              map[string]int      `json:"active_item_buffs"`           // ItemID -> MatchesRemaining
	TotalDonated                 uint64              `json:"total_donated"`               // PILLAR 1: Philanthropy tracking
	ReparationsReceivedCount     int                 `json:"reparations_received_count"`
	IsMojoStabilizerActive       bool                `json:"is_mojo_stabilizer_active"`
	MojoDecayRate                float64             `json:"mojo_decay_rate"`
	RaidInsuranceExpiresAt       time.Time           `json:"raid_insurance_expires_at"`
	RaidInsuranceClaimsRemaining int                 `json:"raid_insurance_claims_remaining"`
	BountyHunterBondMicro        uint64              `json:"bounty_hunter_bond_micro"` // PILLAR 1: Locked Deposit
	BountyHunterLicenseExpiresAt time.Time           `json:"bounty_hunter_license_expires_at"`
	ArenaVouchers                uint64              `json:"arena_vouchers"` // PILLAR 2: Console reward representation
	ActiveUnderworldContractID   string              `json:"active_underworld_contract_id"`
	RecoveryChallengeCardID      int                 `json:"recovery_challenge_card_id"`
	RecoveryChallengeWins        int                 `json:"recovery_challenge_wins"`
	RecoveryBounties             map[int]uint64      `json:"recovery_bounties"`   // CardID -> MicroVBV reward
	MarketFrozenUntil            time.Time           `json:"market_frozen_until"` // PILLAR 3: Justice Layer

	// MutationHistory captures forensic mutation events for deterministic replay & achievements.
	MutationHistory []MutationEvent `json:"mutation_history"`

	// Pillar 12-13: Career progression and rivalry system fields
	CareerXP        *CareerXP         `json:"career_xp,omitempty"`       // XP tracking across career roles
	Rivalry         *RivalryState     `json:"rivalry,omitempty"`          // Rivalry engine state (solo hunter, info broker)
	BountyLicenseActive bool           `json:"bounty_license_active"`      // Pillar 12: Active bounty hunter license flag
	ArcNetActive      bool            `json:"arc_net_active,omitempty"`   // Pillar 12: Active Arc-Net spy vision

	// PILLAR 13: $VBV Sustained Liquidity Tracking for Career XP
	// Tracks average balance over time to determine career tier eligibility
	LiquiditySamples    []uint64        `json:"liquidity_samples,omitempty"` // Recent balance snapshots (micro-$VBV)
	LiquidityWindowMin  int             `json:"liquidity_window_min"`         // Sliding window duration in minutes
	DemotionWarningAt   time.Time       `json:"demotion_warning_at,omitempty"` // When demotion warning was issued (0 = none)
	AvgSustainedMicro   uint64          `json:"avg_sustained_micro"`          // Computed from samples (micro-$VBV)

	// Justice Layer active mission tracking
	ActiveJusticeMissionID string `json:"active_justice_mission_id,omitempty"` // Current Justice Layer mission ID

	// Buff tracking for various game mechanics (e.g., fenced rate, rep shield)
	Buffs map[string]bool `json:"buffs,omitempty"` // Active buff flags
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

// JusticeCategory and UnderworldCategory are shop category constants.
const (
	JusticeCategory     = "JUSTICE"
	UnderworldCategory  = "UNDERWORLD"
)

// PlaystyleTendencies captures observed player behaviors for Collective Intelligence.
type PlaystyleTendencies struct {
	Aggressiveness     float64            `json:"aggressiveness"`       // 0.0 - 1.0, higher means more aggressive
	RiskTolerance      float64            `json:"risk_tolerance"`       // 0.0 - 1.0, higher means more risky
	PreferredRules     map[string]float64 `json:"preferred_rules"`      // RuleName -> Weighted Preference Score
	PreferredCardMoods map[string]float64 `json:"preferred_card_moods"` // Mood -> Weighted Preference Score
	FavoriteCardID     int                `json:"favorite_card_id"`     // The card ID set as favorite
	PreferredItems     map[string]float64 `json:"preferred_items"`      // ItemID -> Weighted Usage Score
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
	SellerName           string     `json:"seller_name"` // Pre-resolved Envoi name
	Bundle               CardBundle `json:"bundle"`
	CurrentBid           uint64     `json:"current_bid"` // Micro-units of $VBV
	HighestBidder        string     `json:"highest_bidder"`
	HighestBidderName    string     `json:"highest_bidder_name"` // Pre-resolved Envoi name
	EndsAt               time.Time  `json:"ends_at"`
	TerritoryID          string     `json:"territory_id"`            // For commission distribution
	HighestBidIsApproved bool       `json:"highest_bid_is_approved"` // PILLAR 2: Non-custodial tracking
}

// Loan represents a collateralized loan from the Second-Hand Store.
type Loan struct {
	ID               string     `json:"id"`
	BorrowerWallet   string     `json:"borrower_wallet"`
	BorrowerName     string     `json:"borrower_name"` // Pre-resolved Envoi name
	CollateralBundle CardBundle `json:"collateral_bundle"`
	LoanAmount       uint64     `json:"loan_amount"`      // Micro-units of $VBV
	RepaymentAmount  uint64     `json:"repayment_amount"` // LoanAmount + Interest
	DueAt            time.Time  `json:"due_at"`
	Status           string     `json:"status"`       // "active", "repaid", "defaulted"
	TerritoryID      string     `json:"territory_id"` // For commission distribution (Second-Hand Store)
}

// Rumor represents an active rumor affecting an entity's share price.
type Rumor struct {
	ID             string    `json:"id"`
	SpreaderWallet string    `json:"spreader_wallet"`
	TargetWallet   string    `json:"target_wallet"`
	Type           string    `json:"type"`     // "positive", "negative"
	Strength       float64   `json:"strength"` // Multiplier (e.g., 1.1 for +10%, 0.9 for -10%)
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
	ID          string `json:"match_id"` // PILLAR 3: Standardized match indexing
	P1          string `json:"p1"`       // Wallet Address
	P2          string `json:"p2"`       // Wallet Address
	Winner      string `json:"winner,omitempty"`
	Round       int    `json:"round"`
	ReceiptTxID string `json:"receipt_txid,omitempty"` // On-chain VBT_WIN receipt ID
}

// TournamentState tracks the progress of an automated event.
type TournamentState struct {
	Active       bool              `json:"active"`
	ID           string            `json:"id"`
	Matches      []TournamentMatch `json:"matches"`
	CurrentRound int               `json:"current_round"`
	Participants []string          `json:"participants"`
	PotMicro     uint64            `json:"pot_micro"`    // PILLAR 2: Integer Supremacy
	BuyInMicro   uint64            `json:"buy_in_micro"` // PILLAR 2: Integer Supremacy
	IsBuyInMode  bool              `json:"is_buy_in_mode"`
	OpenTime     time.Time         `json:"open_time"` // Registration window start
	Winner       string            `json:"winner"`
}
