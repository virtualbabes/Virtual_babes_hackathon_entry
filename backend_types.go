//go:build !js && !wasm

package main

import (
	"context"
	"math"
	"sync"

	"log"
	"time"

	"github.com/gorilla/websocket"
)

// SessionState represents the network status of a player's session.
type SessionState string

const (
	StateConnected         SessionState = "CONNECTED"
	StatePendingDisconnect SessionState = "PENDING_DISCONNECT"
)

// PlayerSession tracks the quarantine state of a disconnected user.
type PlayerSession struct {
	WalletAddress   string
	CurrentState    SessionState
	CancelTimer     context.CancelFunc // Callback function to halt the eviction sequence
	LastActiveFrame uint64
}

// GracePeriodMatrix manages connection drop events and automated state restoration.
// PILLAR 4: Network Resiliency.
type GracePeriodMatrix struct {
	Mu              sync.Mutex
	ActiveSessions  map[string]*PlayerSession
	DisconnectGrace time.Duration
	EvictionWorker  func(wallet string) // Core system hook to drop player rank / trigger match forfeit
}

// FrameDelta captures a single state transition for recovery logs.
type FrameDelta struct {
	SequenceID uint64         `json:"sequence_id"`
	MoveIntent []byte         `json:"move_intent"`
	StateHash  BoardStateHash `json:"state_hash"`
}

// SyncHandshaker manages the chronological verification of match frames.
type SyncHandshaker struct {
	Mu               sync.RWMutex
	CurrentSequence  uint64
	HistoricalFrames map[uint64]FrameDelta
	LastVerifiedHash BoardStateHash
}

// RegionalGovernanceMetric tracks dividends for governors.
// PILLAR 2: uint64 Precision.
type RegionalGovernanceMetric struct {
	GovernorAddress      string `json:"governor_address"`
	DistrictDividendPool uint64 `json:"district_dividend_pool"`
	CustomTaxRate        float64 `json:"custom_tax_rate"` // PILLAR 1: Political Influence (0.0 to 0.20)
}

// ClubTreasuryNode represents a localized organization vault.
type ClubTreasuryNode struct {
	ClubID          uint64 `json:"club_id"`
	TreasuryBalance uint64 `json:"treasury_balance"`
}

// TokenSinkRouter manages the atomic distribution of capital flows.
// PILLAR 2: Ledger circularity.
type TokenSinkRouter struct {
	Mu                sync.RWMutex
	GlobalFaucetPool  *uint64 // Reference to the system rewards reservoir
	AdminMaintenancePool *uint64 // PILLAR 2: Infrastructure Siphon (Section 11)
	ActiveClubs       map[uint64]*ClubTreasuryNode
	MarketNodes       map[string]*EntityMarketNode // PILLAR 2: AMM Persistence
	RegionalDistricts map[string]*RegionalGovernanceMetric
	Audit             *TokenSinkAuditReporter // PILLAR 2: Invariant Monitoring
	SiphonNotifier    func(string)            // PILLAR 2: Infrastructure Funding Alerts
}

// EntityMarketNode implements an Automated Market Maker (AMM) for entity shares.
// PILLAR 2: Dynamic Supply-Elastic Pricing.
type EntityMarketNode struct {
	Mu                sync.RWMutex `json:"-"`
	EntityID          string       `json:"entity_id"`
	TotalSharesIssued uint64       `json:"total_shares_issued"`
	ReserveBalance    uint64       `json:"reserve_balance"` // Micro-VBV
	ReserveRatio      float64      `json:"reserve_ratio"`   // e.g., 0.33
	DividendPoolMicro uint64       `json:"dividend_pool_micro"` // PILLAR 1: Yield-Bearing Assets
	CumulativeYieldPerShare uint64 `json:"cumulative_yield_per_share"` // PILLAR 2: Integer Supremacy
	IsDividendFrozen  bool         `json:"is_dividend_frozen"` // PILLAR 3: Justice Counter-play
}

// VerificationHook handles cryptographic validation of market commands.
// PILLAR 3: Switchboard Security.
type VerificationHook struct {
	Mu             sync.RWMutex
	ActiveNonces   map[uint64]time.Time // Valid nonces + TTL
	ConsumedNonces map[uint64]bool      // Replay protection
}

// EvictionPayload represents the reason for a session termination.
type EvictionPayload struct {
	WalletAddress string `json:"wallet_address"`
	ReasonCode    string `json:"reason_code"` // e.g., "SESSION_EXPIRED" or "INSUFFICIENT_LIQUIDITY"
}

// SessionWatchdog monitors active player eligibility throughout their session.
// PILLAR 3: Continuous Verification.
type SessionWatchdog struct {
	Mu               sync.Mutex
	AuditInterval    time.Duration
	ActiveMonitoring map[string]time.Time // Key: Wallet Address -> Join Time
}

// HostageSituation represents a unique criminal capture event.
// PILLAR 3: Multi-Slot Attacker Isolation.
type HostageSituation struct {
	AttackerAddress string `json:"attacker_address"`
	AssetID         uint64 `json:"asset_id"`
	RansomAmount    uint64 `json:"ransom_amount"`   // PILLAR 2: uint64 Precision
	ExpirationTime  int64  `json:"expiration_time"` // Unix Timestamp (48h Rule)
}

// VictimRegistry tracks active kidnappings indexed by victim and attacker.
// This eliminates the "Immunity Exploit" by allowing multiple attackers per victim.
type VictimRegistry struct {
	Mu sync.RWMutex `json:"-"`
	// ActiveKidnaps maps Victim Address -> Attacker Address -> Situation details.
	// PILLAR 3: Modular Authority (Fine-grained locking).
	ActiveKidnaps map[string]map[string]HostageSituation `json:"active_kidnaps"`
}

// CyberInterceptEvent represents a cyber-interception opportunity detected by Intel-Agent.
// PILLAR 13: Justice Hegemony — Intel-Agent combat hooks base type.
type CyberInterceptEvent struct {
	EventID        string    `json:"event_id"`         // UUID v4 unique identifier
	SourceWallet   string    `json:"source_wallet"`    // Originating wallet (suspect signal source)
	TargetWallet   string    `json:"target_wallet"`    // Intercept target wallet
	SignalStrength int       `json:"signal_strength"`  // 1-100 signal quality score
	DecryptBonus   float64   `json:"decrypt_bonus"`    // Multiplier from Intel-Agent career tier
	CreatedAt      time.Time `json:"created_at"`       // Event creation timestamp
	ExpiresAt      time.Time `json:"expires_at"`       // TTL (e.g., 30 minutes)
	Intercepted    bool      `json:"intercepted"`      // Whether successfully intercepted by Intel-Agent
}

// RaidEvidence represents forensic data collected during a criminal raid or cyber-intercept.
// PILLAR 13: Justice Hegemony — Forensic Analyst combat hooks base type.
type RaidEvidence struct {
	EvidenceID   string    `json:"evidence_id"`      // UUID v4 unique identifier
	SourceWallet string    `json:"source_wallet"`    // Collector wallet (ForensicAnalyst)
	TargetWallet string    `json:"target_wallet"`    // Suspect target wallet
	CrimeType    string    `json:"crime_type"`       // e.g., "KIDNAP", "LAUNDERING", "COUNTERFEIT"
	Confidence   float64   `json:"confidence"`       // 0.0-1.0 confidence score (evidence accuracy)
	CollectedAt  time.Time `json:"collected_at"`     // Collection timestamp
	Expired      bool      `json:"expired"`          // Whether evidence has degraded beyond use
}

// SeasonEventPhase represents the lifecycle stage of a seasonal event.
type SeasonEventPhase string

const (
	SeasonAnnouncement Phase = "ANNOUNCEMENT" // Event announced, players can register
	SeasonActive       Phase = "ACTIVE"        // Event is live with multiplier effects
	SeasonResolution   Phase = "RESOLUTION"    // Event ended, computing payouts
	SeasonTreasuryPayout  Phase = "TREASURY_PAYOUT" // Rewards distributed to participants
)

// SeasonEventType defines the category of seasonal event.
type SeasonEventType string

const (
	SeasonHarvestFest  SeasonEventType = "HARVEST_FEST"   // Resource gathering multiplier week
	SeasonShadowAuction SeasonEventType = "SHADOW_AUCTION" // Criminality rewards doubled
	SeasonTerritoryWar SeasonEventType = "TERRITORY_WAR"   // Club territory bonuses ×2
	SeasonCryptoRaid   SeasonEventType = "CRYPTO_RAID"     // Bounty capture XP bonus week
	SeasonDiamondWeek  SeasonEventType = "DIAMOND_WEEK"    // All rewards +50% for Diamond+ reputation players
)

// SeasonEvent represents a seasonal event managed by the Industrial Loop.
// PILLAR 1: Sacred Industrial Loop — Event-driven value circulation per vision lines 107-159.
type SeasonEvent struct {
	EventID         string          `json:"event_id"`           // UUID v4 unique identifier
	Type            SeasonEventType `json:"type"`               // Category of event
	Title           string          `json:"title"`              // Display title (e.g., "Harvest Fest 2026-Q3")
	Description     string          `json:"description"`        // Event lore and mechanics description
	StartTime       time.Time       `json:"start_time"`         // Announcement start timestamp
	EndTime         time.Time       `json:"end_time"`           // Resolution trigger timestamp
	Multiplier       float64        `json:"multiplier"`         // Base reward multiplier (e.g., 1.5 = +50%)
	TreasuryBudget   uint64         `json:"treasury_budget_micro"` // Treasury allocation in micro-VBV
	ActiveParticipants map[string]time.Time `json:"-"`            // Key: wallet -> registration time (not serialized)
	Status          SeasonEventPhase `json:"status"`           // Current lifecycle phase
	CreatedBy       string          `json:"created_by"`         // Admin wallet that created event
}

// SeasonRewardPool tracks treasury allocation for a seasonal event.
type SeasonRewardPool struct {
	TotalAllocated uint64            `json:"total_allocated_micro"` // Treasury budget in micro-VBV
	Distributed    uint64            `json:"distributed_micro"`     // Amount already paid out
	PerParticipant map[string]uint64 `json:"-"`                     // Key: wallet -> reward amount (not serialized)
}

// SeasonalEventEngine manages the lifecycle of seasonal events.
type SeasonalEventEngine struct {
	Mu              sync.RWMutex               `json:"-"`
	ActiveEvents    map[string]*SeasonEvent     // Key: EventID -> *SeasonEvent
	CurrentRewardPool map[string]*SeasonRewardPool // Key: EventID -> *SeasonRewardPool
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

// Lobby manages the central state of the arena.
type Lobby struct {
	clients                  map[string]*Client
	matches                  map[string]*MatchState
	tournamentPotBonusMicro  uint64 // PILLAR 2: Integer Supremacy
	pendingTournamentPayoutsMicro uint64 // PILLAR 2: Integer Supremacy
	inventory                map[int]ServerCard
	persistentCardCache      map[int]ServerCard
	tournamentPotBonus       float64
	pendingTournamentPayouts float64
	tournamentCache          map[string]*interface{}
	paidParticipants         []string
	matchmakingPool          []QueueEntry
	bannedAvatars            map[string]time.Time
	registeredTxIDs          map[string]time.Time
	processingRewards        map[string]time.Time
	processingOnboarding     map[string]time.Time
	processingRegistrations  map[string]time.Time
	activeKidnappings        map[int]KidnapState          // Legacy card-based tracking
	victimRegistry           *VictimRegistry              // PILLAR 3: Improved multi-attacker logic
	marketNodes              map[string]*EntityMarketNode // PILLAR 2: AMM State
	tokenSinkRouter          *TokenSinkRouter             // PILLAR 2: Economic flow control
	verificationHook         *VerificationHook            // PILLAR 3: Crypto Gate
	sessionWatchdog          *SessionWatchdog             // PILLAR 3: Active session auditor
	payoutScheduler          *PayoutScheduler             // PILLAR 2: Governor distributions
	clubService              *ClubService                 // PILLAR 5: Organization logic
	careerService            *CareerService               // PILLAR 5: Employment logic
	courthouseService        *CourthouseService           // PILLAR 5: Legal logic
	onboardingService        *OnboardingService           // PILLAR 5: New player onboarding logic
	achievementService       *AchievementService          // PILLAR 5: Trophy logic
	oracleService            *OracleService               // PILLAR 5: Blockchain interaction logic
	tournamentService        *TournamentService           // PILLAR 5: Competitive logic
	loanService              *LoanService                 // PILLAR 5: Lending logic
	auctionService           *AuctionService              // PILLAR 5: Art Gallery logic
	blackMarketService       *BlackMarketService          // PILLAR 5: Underworld logic
	narrativeService         *NarrativeService            // PILLAR 5: Story & Atmosphere logic
	nautilusDEXPathService   *NautilusDEXPathService      // PILLAR 2: Console Creator Payouts
	playerService            *PlayerService               // PILLAR 5: Player attribute logic
	justiceService           *JusticeService              // PILLAR 7: Justice Hegemony Path
	justiceHandlers          *JusticeHandlers             // PILLAR 7: HTTP presentation layer for Justice Dashboard
	entityInvestmentService  *EntityInvestmentService     // PILLAR 2 / Phase 7-A: Entity Investment Layer (Player-to-Player Share Allocation)
	evidencePool             *EvidencePool                // PILLAR 13: Forensic evidence pool for Raid events
	seasonEngine             *SeasonalEventEngine         // PILLAR 1: Seasonal event lifecycle management (Industrial Loop)
	creatorStore             *CreatorStore                // PILLAR 7-C: Creator storefront and royalty system
	aiEngine                 *AICitizenEngine             // PILLAR 7-D: AI Autonomous Economy (citizen lifecycle)
	contractEngine           *ContractEngine              // PILLAR 3: Underworld Contracts dynamic engine
	rateLimiter              *RateLimiterService          // PILLAR 1-C: Rate Limiting & DDoS Mitigation
	playerDirectInvestments  map[string]map[string]uint64 // Key: wallet -> map[entity_id]amountMicro (P7-A Entity Investment Layer)
	dividendTracker          *EntityDividendTracker       // P7-A: Per-entity dividend distribution state tracking
	fencedListings           map[string]FenceListing      // P2-B3: Fenced Goods Marketplace listings
	fencedListingsMu         sync.RWMutex                 // Protects fencedListings map
	counterfeitRateLimit     map[string]*TokenBucket      // Per-wallet counterfeit operation rate limiting
	counterfeitRateLimitMu   sync.RWMutex                 // Protects counterfeitRateLimit map
	gracePeriodMatrix        *GracePeriodMatrix           // PILLAR 4: Connection quarantine
	ledgerClient             *LoadBalancedLedgerClient    // PILLAR 4: Resilient RPC Cluster (Voi Mainnet)
	algorandMainnetClient    *algod.Client               // PILLAR-B: Algorand Mainnet transaction client
	multiChainRouter         *MultiChainRouter            // PILLAR-B: Multi-chain transaction routing
	telemetry                *TelemetryLogger             // PILLAR 4: System Observability
	matchHandshakers         map[string]*SyncHandshaker   // PILLAR 4: Frame sequence matching
	wallets                  map[string]string
	clubs                    map[string]*Club
	blackMarket              []Loan
	rumors                   map[string]*Rumor
	loans                    map[string]*Loan
	auctions                 map[string]*Auction
	leaderboard              map[string]PlayerStats
	matchHistory             map[string]MatchHistory
	linkedWallets            map[string]WalletLinkInfo
	vaultAddress             string
	faucetBalanceMicro       uint64 // PILLAR 2: Source of truth for integer accounting
	CorporateTaxTotal        uint64 // Session total micro-units (Task 898)
	CorporateTaxCount        uint64 // Session total contributing contracts (Task 912)
	LuxuryTaxTotal           uint64 // Session total micro-units (Task 898)
	LuxuryTaxCount           uint64 // Session total Master Tier item sales (Task 913)
	SabotageSurchargeTotal   uint64 // Session total distributed to Governors (Task 915)
	GovernorSurchargeTotal   uint64 // Session total collected by capital owner (Task 917)
	PlatformTaxTotal         uint64 // PILLAR 2: Session total from self-redemptions
	AdminMaintenancePool     uint64 // PILLAR 2: Infrastructure Siphon (Section 11)
	faucetBalance            float64
	rewardStack              map[string]uint64
	playerBalances           map[string]uint64
	initialRewards           map[string]uint64
	holdingBonuses           map[string][]HoldingBonus
	initialBaseReward        uint64
	seasonStart              time.Time
	seasonNumber             int
	maxFaucetCapacity        float64
	rewardAssetID            string
	avoiAssetID              string
	baseReward               uint64
	nonces                   map[string]NonceData
	availableNetworks        map[string]NetworkConfig
	adminFocusNetwork        string
	maintenanceMode          bool
	maintenanceTime          time.Time
	maintenancePriority      string
	rateLimits               map[string]time.Time
	httpRateLimits           map[string]*RateBucket
	tournament               TournamentState
	globalSentiment          GlobalSentiment
	register                 chan *Client
	unregister               chan *Client
	broadcast                chan []byte
	onboardedWallets         map[string]bool
	onboardingSemaphore      chan struct{}
	oracleSemaphore          chan struct{}
	envoiCache               map[string]string
	envoiMutex               sync.RWMutex
	lastSeenDistricts        map[string]string
	treasuryAverages         map[string]float64
	treasuryCrashed          map[string]bool
	SybilSyncComplete        bool
	WCProjectID              string
	DataDir                  string
	RewardRatio              float64 // PILLAR 2: Current scaling ratio for transparency
	mutex                    sync.RWMutex
}

func (l *Lobby) GetLevelLabelForDisplay(power int) string {
	if power <= 0 {
		return "Z"
	}
	alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	idx := (power - 1) / 100
	if idx > 25 {
		idx = 25
	}
	return string(alphabet[25-idx])
}

// processMojoDecayLocked is the internal implementation that assumes the lock is held.
// PILLAR 1: Industrial Loop.
func (l *Lobby) processMojoDecayLocked() {
	now := time.Now()
	stagnationThreshold := 48 * time.Hour
	decayOccurred := false
	decayedClubIDs := make(map[string]bool)

	for _, club := range l.clubs {
		if club.Mojo <= 0 {
			continue
		}

		// PILLAR 1: Infrastructure Prestige.
		// Check for active MOJO_STABILIZER buff.
		isMojoStabilizerActive := false
		if expiry, exists := club.BuffExpirations["MOJO_STABILIZER"]; exists {
			if now.Before(expiry) {
				isMojoStabilizerActive = true
			} else {
				// Buff expired, remove it
				delete(club.ActiveBuffs, "MOJO_STABILIZER")
				delete(club.BuffExpirations, "MOJO_STABILIZER")
				log.Printf("[INDUSTRIAL] MOJO_STABILIZER expired for club %s\n", club.Name)
			}
		}

		if now.Sub(club.LastActivity) > stagnationThreshold {
			// PILLAR 1: Dynamic Decay Scaling.
			// Larger clubs lose more Mojo to maintain competitive churn.

			// PILLAR 1: Alliance-Aware Ranking.
			// Use the authoritative helper to ensure Regional Governor status
			// correctly accounts for combined territory counts in alliances.
			isRegion := l.isClubRegionalLocked(club)
			decayRate := 0.02
			minDecay := 5

			if isRegion {
				// PILLAR 1: Regional Governor Accountability.
				// Established regions suffer 2.5x higher decay to prevent sector stagnation.
				decayRate = 0.05
				minDecay = 15
			}

			// PILLAR 1: Inactive Member Scaling.
			// Larger clubs lose Mojo faster when stagnant to reflect organizational overhead.
			// Add 0.2% to the decay rate for every member (e.g. 50 members = +10% rate).
			decayRate += float64(len(club.Members)) * 0.002

			// PILLAR 1: District Stabilizer Effect.
			// Reduce decay rate by 50% if MOJO_STABILIZER is active.
			if isMojoStabilizerActive {
				decayRate *= 0.50
				minDecay = int(float64(minDecay) * 0.50) // Also reduce minimum decay
				log.Printf("[INDUSTRIAL] Club %s Mojo decay reduced by 50%% due to active MOJO_STABILIZER.\n", club.Name)
			}
			decayAmount := int(float64(club.Mojo)*decayRate + 0.5)
			if decayAmount < minDecay {
				decayAmount = minDecay
			}

			club.Mojo -= decayAmount
			if club.Mojo < 0 {
				club.Mojo = 0
			}
			decayOccurred = true
			decayedClubIDs[club.ID] = true
			log.Printf("[INDUSTRIAL] Club %s suffered Mojo decay (isRegion: %v). New Mojo: %d\n", club.Name, isRegion, club.Mojo)

			// Reset clock to 'now' so decay is periodic (e.g., every 48h) rather than continuous
			club.LastActivity = now

			// PILLAR 1: Mojo Surge Integrity.
			// Reset the 24-hour surge window baseline upon decay.
			// This prevents clubs from triggering a "Surge" by simply recovering
			// from a period of stagnation.
			club.MojoStartOf24hWindow = club.Mojo
			club.MojoWindowStartTime = now
		}
	}

	// PILLAR 1: Performance-Optimized Reputation Ripple.
	// If any clubs decayed, perform a single pass over the leaderboard to update
	// employee standings, preventing O(N^2) complexity.
	if len(decayedClubIDs) > 0 {
		for wallet, stats := range l.leaderboard {
			if decayedClubIDs[stats.EmployerClubID] {
				stats.Reputation = l.CalculateReputation(stats)
				l.leaderboard[wallet] = stats
			}
		}
	}

	if decayOccurred {
		// Trigger global sync so UI reflects the Mojo loss
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	}
}
