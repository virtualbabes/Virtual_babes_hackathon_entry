package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

// Global constants for resilience
const indexerTimeout = 10 * time.Second

// getDataPath constructs a full path for persistent files using the DataDir field.
func (l *Lobby) getDataPath(filename string) string {
	if l.DataDir == "" {
		return filename
	}
	return filepath.Join(l.DataDir, filename)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Adjust for production security
	},
}

type GlobalSentiment struct {
	AvgAggressiveness float64
	AvgRiskTolerance  float64
	DominantRules     map[string]float64
	UpdatedAt         time.Time
}

// newLobby creates and returns a new Lobby instance, initializing all shared state.
func newLobby() (*Lobby, error) {
	seasonStart := time.Now()
	seasonNum := 1
	var loadedRewards map[string]uint64

	// Load Season Metadata
	if sData, err := os.ReadFile("season.json"); err == nil {
		var meta struct {
			Start          time.Time         `json:"start"`
			Num            int               `json:"num"`
			InitialRewards map[string]uint64 `json:"initial_rewards"`
		}
		if json.Unmarshal(sData, &meta) == nil {
			seasonStart = meta.Start
			seasonNum = meta.Num
			loadedRewards = meta.InitialRewards
		}
	}

	l := &Lobby{
		clients:                 make(map[string]*Client),
		matches:                 make(map[string]*MatchState),
		inventory:               make(map[int]ServerCard),
		persistentCardCache:     make(map[int]ServerCard),
		wallets:                 make(map[string]string),
		leaderboard:             make(map[string]PlayerStats),
		matchHistory:            make(map[string]MatchHistory),
		nonces:                  make(map[string]NonceData),
		rateLimits:              make(map[string]time.Time),
		httpRateLimits:          make(map[string]*RateBucket),
		bannedAvatars:           make(map[string]time.Time),
		registeredTxIDs:         make(map[string]time.Time),
		processingRewards:       make(map[string]time.Time),
		processingOnboarding:    make(map[string]time.Time),
		processingRegistrations: make(map[string]time.Time),
		activeKidnappings:       make(map[int]KidnapState),
		availableNetworks:       make(map[string]NetworkConfig),
		linkedWallets:           make(map[string]WalletLinkInfo),
		loans:                   make(map[string]*Loan),
		rumors:                  make(map[string]*Rumor),
		auctions:                make(map[string]*Auction),
		rewards:                 make(map[string]uint64),
		initialRewards:          make(map[string]uint64),
		holdingBonuses:          make(map[string][]HoldingBonus),
		register:                make(chan *Client),
		unregister:              make(chan *Client),
		broadcast:               make(chan []byte),
		onboardedWallets:        make(map[string]bool),   // Initialize the new map
		onboardingSemaphore:     make(chan struct{}, 5),  // Limit concurrent bridge operations
		oracleSemaphore:         make(chan struct{}, 10), // Limit concurrent indexer queries
		envoiCache:              make(map[string]string),
		vaultAddress:            os.Getenv("VAULT_ADDRESS"),
		WCProjectID:             os.Getenv("WC_PROJECT_ID"), // Load WalletConnect Project ID
		DataDir:                 os.Getenv("DATA_DIR"),      // Persistent volume path
		maxFaucetCapacity:       10000.0,
		adminFocusNetwork:       "Voi Mainnet",
	}

	// Initialize reward configuration
	baseReward, _ := strconv.ParseUint(os.Getenv("BASE_REWARD"), 10, 64)
	l.baseReward = baseReward * 1000000
	l.initialBaseReward = l.baseReward
	l.rewardAssetID = os.Getenv("REWARD_ASSET_ID")
	l.avoiAssetID = os.Getenv("AVOI_ASSET_ID")

	l.seasonStart = seasonStart
	l.seasonNumber = seasonNum

	// Restore unscaled reward targets from disk if they exist
	if loadedRewards != nil {
		for k, v := range loadedRewards {
			l.initialRewards[k] = v
		}
	}

	l.loadNetworkConfigs()
	l.loadRegisteredTxIDs()
	l.loadLinkedWallets()
	go l.loadOnboardedWalletsFromIndexer() // Reconstruct Sybil protection state
	go l.loadRegistrationsFromIndexer()    // Reconstruct tournament registration state

	// Load Persistent Card Cache
	if data, err := os.ReadFile(l.getDataPath("card_cache.json")); err == nil {
		l.mutex.Lock()
		json.Unmarshal(data, &l.persistentCardCache)
		for id, card := range l.persistentCardCache {
			l.inventory[id] = card
		}
		l.mutex.Unlock()
		log.Printf("[CACHE] Loaded %d cards from persistent storage.\n", len(l.persistentCardCache))
	}

	return l, nil
}

// loadNetworkConfigs loads network configurations from the local JSON store.
func (l *Lobby) loadNetworkConfigs() {
	data, err := os.ReadFile("networks.json")
	if err != nil {
		log.Println("[CONFIG] networks.json not found, using defaults.")
		l.availableNetworks["Voi Mainnet"] = NetworkConfig{
			NetworkName:  "Voi Mainnet",
			IndexerURL:   "https://mainnet-idx.voi.nodly.io",
			NodeURL:      "https://mainnet-api.voi.nodly.io",
			ExplorerURL:  "https://block.voi.network",
			AppID:        l.rewardAssetID,
			AssetID:      l.rewardAssetID,
			ChainID:      "algorand:wGHE2Pwd1-YdV4EuJFy9u6C24-L-2B05",
			PowerDivisor: 1000000,
			PowerBase:    50,
		}
		l.availableNetworks["Algorand Mainnet"] = NetworkConfig{
			NetworkName:  "Algorand Mainnet",
			IndexerURL:   "https://mainnet-idx.algonode.cloud",
			NodeURL:      "https://mainnet-api.algonode.cloud",
			ExplorerURL:  "https://explorer.perawallet.app",
			AppID:        "0",             // No game app on Algo, assets only
			AssetID:      l.rewardAssetID, // Placeholder or specific mapping
			ChainID:      "algorand:mainnet-v1.0",
			PowerDivisor: 1000000,
			PowerBase:    50,
		}
		// Other chains added as Metadata sources only - No transaction capability implied
		l.availableNetworks["Ethereum"] = NetworkConfig{
			NetworkName:  "Ethereum",
			IndexerURL:   "https://api.etherscan.io",
			NodeURL:      "https://mainnet.infura.io/v3/your-project-id",
			ExplorerURL:  "https://etherscan.io",
			ChainID:      "eip155:1",
			PowerDivisor: 1e18, // standard ETH decimals
			PowerBase:    100,
		}
		l.availableNetworks["Solana"] = NetworkConfig{
			NetworkName:  "Solana",
			IndexerURL:   "https://api.mainnet-beta.solana.com",
			NodeURL:      "https://api.mainnet-beta.solana.com",
			ExplorerURL:  "https://solscan.io",
			ChainID:      "solana:5eykt4UsFvXYfy2khQbSsLurFBXY",
			PowerDivisor: 1e9, // standard SOL decimals
			PowerBase:    75,
		}
		l.availableNetworks["Polygon"] = NetworkConfig{
			NetworkName:  "Polygon",
			IndexerURL:   "https://api.polygonscan.com",
			NodeURL:      "https://polygon-mainnet.infura.io/v3/your-project-id",
			ExplorerURL:  "https://polygonscan.com",
			ChainID:      "eip155:137",
			PowerDivisor: 1e18,
			PowerBase:    40,
		}
		l.availableNetworks["Bitcoin"] = NetworkConfig{
			NetworkName:  "Bitcoin",
			IndexerURL:   "https://ordinals.com",
			NodeURL:      "https://ordinals.com",
			ExplorerURL:  "https://ordiscan.com",
			ChainID:      "bip122:000000000019d6689c085ae165831e93",
			PowerDivisor: 1, // Ordinals are individual inscriptions
			PowerBase:    200,
		}
		l.availableNetworks["Flow"] = NetworkConfig{
			NetworkName:  "Flow",
			IndexerURL:   "https://rest-mainnet.onflow.org",
			NodeURL:      "https://access-mainnet-beta.onflow.org",
			ExplorerURL:  "https://flowscan.org",
			ChainID:      "flow:mainnet",
			PowerDivisor: 1e8,
			PowerBase:    60,
		}
		l.availableNetworks["WAX"] = NetworkConfig{
			NetworkName:  "WAX",
			IndexerURL:   "https://wax.api.atomicassets.io",
			NodeURL:      "https://wax.greymass.com",
			ExplorerURL:  "https://wax.bloks.io",
			ChainID:      "wax:1064487b3cd1a897ce03ae5b6a865651",
			PowerDivisor: 1e8,
			PowerBase:    30,
		}
		l.saveNetworkConfigs()
		return
	}
	l.mutex.Lock()
	json.Unmarshal(data, &l.availableNetworks)
	l.mutex.Unlock()
}

// saveNetworkConfigs persists the current network configurations to disk.
func (l *Lobby) saveNetworkConfigs() {
	l.mutex.RLock()
	data, _ := json.MarshalIndent(l.availableNetworks, "", "  ")
	l.mutex.RUnlock()
	os.WriteFile("networks.json", data, 0644)
}

// distributeTournamentKickback handles the 1-5% payout to clubs based on member tournament fees.
// Ensures only players who were members at the time of tournament registration qualify.
func (l *Lobby) distributeTournamentKickback(playerWallet string, feeMicro uint64, registrationTime time.Time, network string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// PILLAR 3: Dynamic Precision Recovery.
	divisor := 1000000.0 // Default fallback
	if cfg, ok := l.availableNetworks[network+" Mainnet"]; ok && cfg.PowerDivisor > 0 {
		divisor = cfg.PowerDivisor
	}

	for _, club := range l.clubs {
		joinedAt, isMember := club.Members[strings.ToLower(playerWallet)]

		// Verify the player was a member at the time of tournament registration
		if isMember && joinedAt.Before(registrationTime) {
			// Base 1% kickback, scales with Club Mojo up to 5%
			rate := 0.01 + (float64(club.Mojo)/1000.0)*0.04
			if rate > 0.05 {
				rate = 0.05
			}

			kickback := (float64(feeMicro) / divisor) * rate
			club.Treasury += kickback
			club.LastActivity = time.Now()

			log.Printf("[REVENUE] Club %s received %.2f $VBV kickback from member %s registration (Rate: %.1f%%)\n",
				club.Name, kickback, playerWallet, rate*100)

			// A player can only benefit one club's treasury per registration
			return
		}
	}
}

// serveWs upgrades HTTP connections to WebSockets and registers clients in the Lobby.
func serveWs(lobby *Lobby, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS ERROR] Upgrade failed: %v\n", err)
		return
	}

	client := &Client{
		conn:  conn,
		send:  make(chan []byte, 256),
		id:    fmt.Sprintf("Player-%d", time.Now().UnixNano()%10000),
		lobby: lobby,
	}

	lobby.register <- client

	// Initial connection handshake: provide the client with their ID and server config
	// $Voi First: Identity always emphasizes Voi assets for payouts
	identityMsg := Envelope{
		Type:   "identity",
		ToID:   client.id,
		FromID: "SERVER",
		Payload: json.RawMessage(fmt.Sprintf(`{"vault":"%s","vbv":"%s","avoi":"%s","wc_project_id":"%s","primary_network": "Voi Mainnet"}`,
			lobby.vaultAddress,
			lobby.rewardAssetID,
			lobby.avoiAssetID,
			lobby.WCProjectID, // Include the WalletConnect Project ID
		)),
	}
	msg, _ := json.Marshal(identityMsg)
	client.send <- msg

	go client.writePump()
	go client.readPump()
}

func main() {
	// Load local .env file if it exists (primarily for development)
	if err := godotenv.Load(); err != nil {
		log.Println("[INFO] No .env file found; relying on platform-injected environment variables.")
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Mainnet Security Audit: Pre-validate critical secrets at startup to ensure stability
	mnemonicRaw := os.Getenv("FAUCET_MNEMONIC")
	if mnemonicRaw == "" {
		log.Println("[CRITICAL WARNING] FAUCET_MNEMONIC is missing! Reward payouts and onboarding features will be disabled.")
	} else if len(strings.Fields(mnemonicRaw)) != 25 {
		log.Println("[ERROR] FAUCET_MNEMONIC appears malformed (expected 25 words). Check your deployment configuration.")
	}

	if os.Getenv("ADMIN_WALLETS") == "" {
		log.Println("[WARNING] ADMIN_WALLETS is not configured. Administrative panel authentication will fail.")
	}

	lobby, err := newLobby()
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize Arena Lobby: %v", err)
	}

	// Start the main event loop (Defined in lobby_manager.go)
	go lobby.run()

	// --- ROUTING ---

	// WebSocket Entry Point
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(lobby, w, r)
	})

	http.HandleFunc("/api/leaderboard", lobby.handleLeaderboard)
	http.HandleFunc("/api/reward", lobby.handleReward) // Now in faucet_service.go
	http.HandleFunc("/api/status", lobby.handlePublicStatus)
	http.HandleFunc("/api/health", lobby.handleHealthCheck)
	http.HandleFunc("/api/card-stats", lobby.handleCardStats)
	http.HandleFunc("/api/card-details", lobby.handleGetCardDetails)
	http.HandleFunc("/api/re-sync-stats", lobby.handleReSyncStats)
	http.HandleFunc("/api/season/history", lobby.handleSeasonHistory)
	http.HandleFunc("/api/courthouse/reset", lobby.handleCourthouseReset)

	// Art Gallery / Auctions
	http.HandleFunc("/api/auctions", lobby.handleGetAuctions)
	http.HandleFunc("/api/auctions/create", lobby.handleCreateAuction)
	http.HandleFunc("/api/auctions/bid", lobby.handlePlaceBid)

	// Second-Hand Store / Loans
	http.HandleFunc("/api/loans", lobby.handleGetLoans)
	http.HandleFunc("/api/loans/take", lobby.handleTakeLoan)
	http.HandleFunc("/api/loans/repay", lobby.handleRepayLoan)

	// Underworld / Black Market
	http.HandleFunc("/api/black-market", lobby.handleGetBlackMarket)
	http.HandleFunc("/api/black-market/buy", lobby.handleBuyBlackMarket)
	http.HandleFunc("/api/black-market/sell-tokens", lobby.handleSellMarketTokens)

	// Onboarding (Handlers defined in onboarding_service.go)
	http.HandleFunc("/api/bridge/onboard", lobby.handleVoiOnboarding)

	// Tournament Management (Handlers defined in tournament_manager.go)
	http.HandleFunc("/api/tournament/register", lobby.handleTournamentRegister)
	http.HandleFunc("/api/tournament/history", lobby.handleTournamentHistory)

	// Admin Controls (Handlers defined in handlers_admin.go)
	http.HandleFunc("/api/refill-vault", lobby.handleRefillVault)
	http.HandleFunc("/api/update-rules", lobby.handleUpdateRules)
	http.HandleFunc("/api/system-message", lobby.handleSystemMessage)
	http.HandleFunc("/api/ban-player", lobby.handleBanPlayer)
	http.HandleFunc("/api/reset-stats", lobby.handleResetStats)
	http.HandleFunc("/api/maintenance-mode", lobby.handleMaintenanceMode)
	http.HandleFunc("/api/reward/add", lobby.handleAdminAddReward)
	http.HandleFunc("/api/reward/remove", lobby.handleAdminRemoveReward)
	http.HandleFunc("/api/admin/network/add", lobby.handleAddNetwork)
	http.HandleFunc("/api/admin/set-admin-focus-network", lobby.handleSetActiveNetwork)
	http.HandleFunc("/api/admin/update-power", lobby.handleUpdatePowerScaling)
	http.HandleFunc("/api/admin/logs", lobby.handleGetAdminLogs)
	http.HandleFunc("/api/admin/simulate-tournament", lobby.handleSimulateTournament)
	http.HandleFunc("/api/admin/gloat-ban", lobby.handleGloatBan)
	http.HandleFunc("/api/admin/avatar-ban", lobby.handleAvatarBan)
	http.HandleFunc("/api/admin/start-tournament", lobby.handleStartTournament)
	http.HandleFunc("/api/admin/open-registration", lobby.handleOpenRegistration)

	// Static Asset Serving (WASM and UI)
	fs := http.FileServer(http.Dir("./Public"))
	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for local development if needed
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Security: Prevent WASM caching for rapid development cycles
		if r.URL.Path == "/main.wasm" {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		}
		fs.ServeHTTP(w, r)
	}))

	// Server Startup
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	fmt.Println("-------------------------------------------------")
	fmt.Printf(" VOICONOMY ARENA SERVER ONLINE: PORT %s\n", port)
	fmt.Println(" WebSocket Switchboard & API Ready               ")
	fmt.Println("-------------------------------------------------")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("[FATAL] Server startup failed: %v", err)
	}
}
