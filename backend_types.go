//go:build !js && !wasm

package main

import (
	"sync"

	"log"
	"time"

	"github.com/gorilla/websocket"
)

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
	activeKidnappings        map[int]KidnapState
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
