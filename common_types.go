package main

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Matchmaking Logic
type QueueEntry struct {
	ClientID   string    `json:"client_id"`
	Wallet     string    `json:"wallet"`
	Reputation int       `json:"reputation"`
	DeckRating string    `json:"deck_rating"`
	JoinedAt   time.Time `json:"joined_at"`
}

// NonceData stores the nonce value and its creation time for expiration logic.
type NonceData struct {
	Value     string
	CreatedAt time.Time
}

// RateBucket implements the Leaky Bucket state for a single entity (IP).
type RateBucket struct {
	Tokens     float64
	LastUpdate time.Time
}

// HoldingBonus defines a multiplier for a specific reward if a player holds a certain asset.
type HoldingBonus struct {
	HoldingAssetID string  `json:"holding_asset_id"` // The NFT/Token required to be held
	Network        string  `json:"network"`          // Chain to check (VOI or ALGO)
	Multiplier     float64 `json:"multiplier"`       // Reward boost (e.g., 1.1 for 10% bonus)
	MinAmount      uint64  `json:"min_amount"`       // Minimum micro-units required to qualify
}

// Club represents a player-owned organization with specialized shops.
type Club struct {
	ID              string               `json:"id"`
	Name            string               `json:"name"`
	OwnerWallet     string               `json:"owner_wallet"`
	Type            string               `json:"type"`        // Elemental, Tactical, Vitality
	Territories     []string             `json:"territories"` // Supports multiple districts
	RegionName      string               `json:"region_name,omitempty"`
	Treasury        float64              `json:"treasury"`
	Commission      float64              `json:"commission_rate"` // e.g., 0.05 for 5%
	Inventory       map[string]int       `json:"inventory"`       // ItemID -> Quantity
	Staff           map[string]string    `json:"staff"`           // Wallet -> Role (Manager, Security, Clerk)
	ActiveBuffs     map[string]string    `json:"active_buffs"`
	BuffExpirations map[string]time.Time `json:"buff_expirations"` // Key -> Expiration Timestamp
	Members         map[string]time.Time `json:"members"`          // Wallet -> Join Timestamp
	Leases          map[string]*Lease    `json:"leases"`           // LeaseID -> Lease (cards available for rent)
	Mojo            int                  `json:"club_mojo"`        // Unlocks higher tier items
	Jail            map[int]ServerCard   `json:"jail"`             // CardID -> ServerCard (captured cards)
	LastActivity    time.Time            `json:"last_activity"`    // For Mojo decay tracking
	LastHeistAt     time.Time            `json:"last_heist_at"`    // Hook for UI .under-attack status
	CreatedAt       time.Time            `json:"created_at"`
}

// Lease represents a card available for temporary use within a club.
type Lease struct {
	ID            string    `json:"id"`
	LenderWallet  string    `json:"lender_wallet"`
	CardID        int       `json:"card_id"`
	CardName      string    `json:"card_name"`
	Price         float64   `json:"price"` // Base units of $VBV
	DurationHours int       `json:"duration_hours"`
	ExpiresAt     time.Time `json:"expires_at,omitempty"` // Set once taken
	Borrower      string    `json:"borrower_wallet,omitempty"`
	ClubID        string    `json:"club_id"`
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
	Action string          `json:"action"` // "invite", "accept", "decline", "sync_back"
	Deck   []int           `json:"deck,omitempty"`
	Avatar string          `json:"avatar,omitempty"`
	Gloat  string          `json:"gloat,omitempty"`
	Rules  map[string]bool `json:"rules,omitempty"`
	Wanted int             `json:"wanted_level,omitempty"`
}

// MoveData synchronizes gameplay actions between two human players.
type MoveData struct {
	GridIndex int    `json:"grid_index"`
	CardID    int    `json:"card_id"`
	Power     [4]int `json:"power"`
}

// ReportGloatData captures information about a reported gloat message.
type ReportGloatData struct {
	OpponentClientID string `json:"opponent_client_id"`
	GloatText        string `json:"gloat_text"`
}

// NetworkConfig holds the configuration details for a specific blockchain network.
type NetworkConfig struct {
	NetworkName  string  `json:"network_name"`
	ExplorerURL  string  `json:"explorer_url"`
	IndexerURL   string  `json:"indexer_url"`
	NodeURL      string  `json:"node_url"`
	FaucetURL    string  `json:"faucet_url"`
	AssetID      string  `json:"asset_id"` // The primary game asset ID on this network
	AppID        string  `json:"app_id"`   // The main game smart contract ID on this network
	ChainID      string  `json:"chain_id"` // WalletConnect / CAIP-2 Chain ID
	PowerDivisor float64 `json:"power_divisor"`
	PowerBase    int     `json:"power_base"`
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
	TournamentID      string          `json:"tournament_id,omitempty"`       // Instance ID of the tournament
	TournamentMatchID string          `json:"tournament_match_id,omitempty"` // Link to tournament bracket
	P1Deck            []int           `json:"p1_deck"`                       // Card IDs in P1's deck
	P1Avatar          string          `json:"p1_avatar"`
	P1Gloat           string          `json:"p1_gloat"`
	P2Deck            []int           `json:"p2_deck"` // Card IDs in P2's deck
	P2Avatar          string          `json:"p2_avatar"`
	BoardMoods        [9]string       `json:"board_moods"` // Moods assigned to specific tiles
	P2Gloat           string          `json:"p2_gloat"`
	Board             [9]*ServerCard  `json:"board"`
	Rules             map[string]bool `json:"rules"`
	IsFinished        bool            `json:"is_finished"`
	Spectators        []string        `json:"spectators"` // Client IDs spectating this match
	P1WantedLevel     int             `json:"p1_wanted_level"`
	P2WantedLevel     int             `json:"p2_wanted_level"`
	P1Cunning         int             `json:"p1_cunning"`
	P1Nurturing       int             `json:"p1_nurturing"`
	P2Cunning         int             `json:"p2_cunning"`
	P2Nurturing       int             `json:"p2_nurturing"`
	P1RegionalBoost   bool            `json:"p1_regional_boost"`
	P2RegionalBoost   bool            `json:"p2_regional_boost"`
	FinalScores       [2]int
	CapturedCards     []CapturedCardInfo        `json:"captured_cards,omitempty"` // Tracking for jailing
	Round             int                       `json:"round"`                    // Match round (isolation for Sudden Death)
	TerritoryID       string                    `json:"territory_id,omitempty"`   // The territory where the match is played
	ActiveItemBuffs   map[string]map[string]int `json:"active_item_buffs"`        // PlayerID -> ItemID -> MatchesRemaining
	IsBountyMatch     bool
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
	BountyReward      float64                   `json:"bounty_reward,omitempty"`
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
	Wins              int                `json:"wins"`
	DNFs              int                `json:"dnfs"`
	DisconnectStreak  int                `json:"disconnect_streak"`
	BanExpires        time.Time          `json:"ban_expires"`
	GloatBannedUntil  time.Time          `json:"gloat_banned_until"`
	EquippedFaceplate string             `json:"equipped_faceplate"`
	Reputation        int                `json:"reputation"`
	Mojo              int                `json:"mojo"`                // Social standing for Club unlocks
	SocialRank        string             `json:"social_rank"`         // e.g., "Nobody", "Regular", "Icon"
	JobRole           string             `json:"job_role"`            // Manager, Security, Clerk, Freelancer
	EmployerClubID    string             `json:"employer_id"`         // The club currently paying this user
	Salary            uint64             `json:"salary"`              // Micro-units of $VBV per payment cycle
	AuctionsWon       int                `json:"auctions_won"`        // Total Art Gallery victories
	LastSalaryPayment time.Time          `json:"last_salary_payment"` // Timestamp of last payment
	Inventory         map[string]int     `json:"inventory"`           // ItemID -> Quantity
	History           []MatchHistory     `json:"match_history"`       // Historical records from blockchain
	MarketTokens      uint64             `json:"market_tokens"`       // Equity from liquidated loans
	Relationships     map[string]int     `json:"relationships"`       // Character Name -> Score (0-100)
	BestRating        string             `json:"best_rating"`
	Achievements      []string           `json:"achievements"`   // List of unlocked IDs
	Portfolio         map[string]float64 `json:"portfolio"`      // EntityID -> Shares
	WantedLevel       int                `json:"wanted_level"`   // Risk factor for heists
	HeistAttempts     int                `json:"heist_attempts"` // Number of times player attempted a heist
	Cunning           int                `json:"cunning"`        // Success modifier for criminal actions
	Nurturing         int                `json:"nurturing"`      // Success modifier for garden/donations
	JailedCards       map[int]string     `json:"jailed_cards"`   // CardID -> ClubID (cards currently in jail)
	// New fields for Kidnap Gambit
	FavoriteCardID   int            `json:"favorite_card_id"`   // The card ID the player has designated as their favorite
	KidnappedCards   map[int]string `json:"kidnapped_cards"`    // CardID -> VictimWallet (cards player has kidnapped)
	HeldHostageCards map[int]string `json:"held_hostage_cards"` // CardID -> KidnapperWallet (cards player has lost to kidnapping)
	// New fields for Collective NPC Intelligence
	RumorCount     int                 `json:"rumor_count"`     // Number of rumors spread by this player
	Aggressiveness float64             `json:"aggressiveness"`  // 0-1 scale of aggressive play
	RiskTolerance  float64             `json:"risk_tolerance"`  // 0-1 scale of risk-taking
	PreferredRules map[string]int      `json:"preferred_rules"` // Rule name -> usage count
	Moods          map[string]int      `json:"moods"`           // Mood -> count (e.g., "aggressive", "defensive")
	Playstyle      PlaystyleTendencies `json:"playstyle"`       // Observed playstyle tendencies
}

// PlaystyleTendencies captures observed player behaviors for Collective Intelligence.
type PlaystyleTendencies struct {
	Aggressiveness     float64            `json:"aggressiveness"`       // 0.0 - 1.0, higher means more aggressive
	RiskTolerance      float64            `json:"risk_tolerance"`       // 0.0 - 1.0, higher means more risky
	PreferredRules     map[string]float64 `json:"preferred_rules"`      // RuleName -> Weighted Preference Score
	PreferredCardMoods map[string]float64 `json:"preferred_card_moods"` // Mood -> Weighted Preference Score
	FavoriteCardID     int                `json:"favorite_card_id"`     // The card ID set as favorite
	PreferredItems     map[string]float64 `json:"preferred_items"`      // ItemID -> Weighted Usage Score
}

// CardBundle represents a set of items listed together in an auction.
type CardBundle struct {
	CardID      int    `json:"card_id"`
	WeaponID    string `json:"weapon_id,omitempty"`
	FaceplateID string `json:"faceplate_id,omitempty"`
}

// Auction represents a live listing in the Art Gallery.
type Auction struct {
	ID                string     `json:"id"`
	SellerWallet      string     `json:"seller_wallet"`
	SellerName        string     `json:"seller_name"` // Pre-resolved Envoi name
	Bundle            CardBundle `json:"bundle"`
	CurrentBid        uint64     `json:"current_bid"` // Micro-units of $VBV
	HighestBidder     string     `json:"highest_bidder"`
	HighestBidderName string     `json:"highest_bidder_name"` // Pre-resolved Envoi name
	EndsAt            time.Time  `json:"ends_at"`
	TerritoryID       string     `json:"territory_id"` // For commission distribution
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
	ID          string `json:"id"`
	P1          string `json:"p1"` // Wallet Address
	P2          string `json:"p2"` // Wallet Address
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
	Pot          float64           `json:"pot"`
	BuyInAmount  float64           `json:"buy_in_amount"`
	IsBuyInMode  bool              `json:"is_buy_in_mode"`
	OpenTime     time.Time         `json:"open_time"` // Registration window start
	Winner       string            `json:"winner"`
}

// TournamentSummary represents a finalized tournament for archival.
type TournamentSummary struct {
	ID               string            `json:"id"`
	Timestamp        time.Time         `json:"timestamp"`
	Pot              float64           `json:"pot"`
	Winner           string            `json:"winner"`
	IsVerified       bool              `json:"is_verified"`            // Indicates successful blockchain reconstruction
	ReceiptsVerified bool              `json:"receipts_verified"`      // Indicates VBT_WIN receipts were found for all matches
	PayoutsHash      string            `json:"payouts_hash,omitempty"` // SHA256 of reward transaction IDs
	Checksum         string            `json:"checksum,omitempty"`     // SHA256 of full match data
	Links            []string          `json:"links,omitempty"`        // TxIDs for additional match data
	Matches          []TournamentMatch `json:"matches"`
}

// Client represents one connected WebSocket user.
type Client struct {
	conn              *websocket.Conn
	send              chan []byte
	id                string
	isAdmin           bool
	avatarURL         string
	gloat             string
	avatarBanNotice   string
	messageTimestamps []time.Time
	msgMutex          sync.Mutex
	lobby             *Lobby
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

// Lobby manages the central state of the arena.
type Lobby struct {
	clients                 map[string]*Client
	matches                 map[string]*MatchState
	inventory               map[int]ServerCard
	persistentCardCache     map[int]ServerCard
	tournamentPotBonus      float64
	tournamentCache         map[string]*interface{} // Using interface{} for element storage
	paidParticipants        []string
	matchmakingPool         []QueueEntry
	bannedAvatars           map[string]time.Time
	registeredTxIDs         map[string]time.Time
	processingRewards       map[string]time.Time
	processingOnboarding    map[string]time.Time
	processingRegistrations map[string]time.Time // Prevents concurrent registration hits for the same wallet
	activeKidnappings       map[int]KidnapState  // CardID -> State
	wallets                 map[string]string
	clubs                   map[string]*Club // Key: ClubID
	blackMarket             []Loan           // Defaulted loans available for purchase
	rumors                  map[string]*Rumor
	loans                   map[string]*Loan
	auctions                map[string]*Auction
	leaderboard             map[string]PlayerStats
	matchHistory            map[string]MatchHistory
	linkedWallets           map[string]WalletLinkInfo
	vaultAddress            string
	faucetBalance           float64
	rewards                 map[string]uint64
	initialRewards          map[string]uint64 // Unscaled base values for all assets in the reward stack
	holdingBonuses          map[string][]HoldingBonus
	initialBaseReward       uint64
	seasonStart             time.Time
	seasonNumber            int
	maxFaucetCapacity       float64
	rewardAssetID           string
	avoiAssetID             string
	baseReward              uint64
	nonces                  map[string]NonceData
	availableNetworks       map[string]NetworkConfig
	adminFocusNetwork       string
	maintenanceMode         bool
	maintenanceTime         time.Time
	rateLimits              map[string]time.Time
	httpRateLimits          map[string]*RateBucket
	tournament              TournamentState
	globalSentiment         GlobalSentiment
	register                chan *Client
	unregister              chan *Client
	broadcast               chan []byte
	onboardedWallets        map[string]bool // Tracks wallets that have received an onboarding pack
	onboardingSemaphore     chan struct{}
	oracleSemaphore         chan struct{}     // Throttles concurrent external indexer requests
	envoiCache              map[string]string // Wallet -> Envoi Name Cache
	envoiMutex              sync.RWMutex      // Dedicated lock for name resolution
	SybilSyncComplete       bool              // Indicates historical claim state is fully restored
	WCProjectID             string            // WalletConnect Project ID from environment variable
	DataDir                 string            // Path to persistent data directory for volumes
	mutex                   sync.RWMutex
}
