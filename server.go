//go:build !js && !wasm

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

// getDataPath constructs a full path for persistent files using the DataDir field.
func (l *Lobby) getDataPath(filename string) string {
	if l.DataDir == "" {
		return filename
	}
	return filepath.Join(l.DataDir, filename)
}

// corsMiddleware returns a middleware that enforces strict CORS for production domains.
func corsMiddleware() func(http.Handler) http.Handler {
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	allowAll := allowedOrigins == "" || strings.TrimSpace(strings.ToLower(allowedOrigins)) == "*"

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				origin := r.Header.Get("Origin")
				for _, o := range strings.Split(allowedOrigins, ",") {
					o = strings.TrimSpace(o)
					if o == origin {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						w.Header().Set("Vary", "Origin")
						break
					}
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		allowed := os.Getenv("ALLOWED_ORIGINS")
		if allowed == "" || strings.TrimSpace(strings.ToLower(allowed)) == "*" {
			return true // Permissive in dev or if explicitly wildcarded
		}
		origin := r.Header.Get("Origin")
		for _, o := range strings.Split(allowed, ",") {
			if strings.TrimSpace(o) == origin {
				return true
			}
		}
		log.Printf("[SECURITY] WebSocket connection rejected from unauthorized origin: %s", origin)
		return false
	},
}

// newLobby creates and returns a new Lobby instance, initializing all shared state.
func newLobby() (*Lobby, error) {
	startBoot := time.Now()
	ctx := context.Background()

	seasonStart := time.Now()
	seasonNum := 1

	// PILLAR 5: Infrastructure Synergy.
	wcID := os.Getenv("WC_PROJECT_ID")
	if wcID == "" {
		wcID = os.Getenv("AppID") // Prioritize the user-added AppID from Render
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
		activeKidnappings:       make(map[int]KidnapState), // Legacy
		victimRegistry:          &VictimRegistry{ActiveKidnaps: make(map[string]map[string]HostageSituation)},
		verificationHook:        NewVerificationHook(),
		availableNetworks:       make(map[string]NetworkConfig),
		clubService:             &ClubService{},
		careerService:           &CareerService{},
		courthouseService:       &CourthouseService{},
		onboardingService:       &OnboardingService{},
		achievementService:      &AchievementService{},
		oracleService:           &OracleService{},
		tournamentService:       &TournamentService{},
		loanService:             &LoanService{},
		auctionService:          &AuctionService{},
		blackMarketService:      &BlackMarketService{},
		narrativeService:        &NarrativeService{},
		nautilusDEXPathService:  &NautilusDEXPathService{}, // PILLAR 2: Console Creator Payouts
		playerService:           &PlayerService{},
		justiceService:          &JusticeService{}, // PILLAR 7: Justice Hegemony Path
		matchHandshakers:        make(map[string]*SyncHandshaker),

		linkedWallets:           make(map[string]WalletLinkInfo),
		loans:                   make(map[string]*Loan),
		rumors:                  make(map[string]*Rumor),
		auctions:                make(map[string]*Auction),
		rewardStack:             make(map[string]uint64),
		playerBalances:          make(map[string]uint64),
		initialRewards:          make(map[string]uint64),
		holdingBonuses:          make(map[string][]HoldingBonus),
		register:                make(chan *Client),
		unregister:              make(chan *Client),
		broadcast:               make(chan []byte),
		onboardedWallets:        make(map[string]bool),   // Initialize the new map
		fencedListings:          make(map[string]FenceListing), // P2-B3: Fenced Goods Marketplace
		onboardingSemaphore:     make(chan struct{}, 5),  // Limit concurrent bridge operations
		oracleSemaphore:         make(chan struct{}, 10), // Limit concurrent indexer queries
		envoiCache:              make(map[string]string),
		lastSeenDistricts:       make(map[string]string),
		treasuryAverages:        make(map[string]float64),
		treasuryCrashed:         make(map[string]bool),
		vaultAddress:            os.Getenv("VAULT_ADDRESS"),
		WCProjectID:             wcID,                  // Load WalletConnect Project ID (AppID)
		DataDir:                 os.Getenv("DATA_DIR"), // Persistent volume path
		maxFaucetCapacity:       10000.0,
		adminFocusNetwork:       "Voi Mainnet",
		maintenancePriority:     "info",
	}

	// Initialize reward configuration
	baseReward, _ := strconv.ParseUint(os.Getenv("BASE_REWARD"), 10, 64)
	l.baseReward = baseReward * 1000000
	l.initialBaseReward = l.baseReward
	l.rewardAssetID = os.Getenv("REWARD_ASSET_ID")
	l.avoiAssetID = os.Getenv("AVOI_ASSET_ID")
	l.initialRewards[l.rewardAssetID] = l.initialBaseReward

	// PILLAR 2: Economic Engine Initialization
	l.tokenSinkRouter = NewTokenSinkRouter(&l.faucetBalanceMicro, &l.AdminMaintenancePool)

	// PILLAR 2: Authoritative Map Linking.
	// Ensure the Lobby and the Router share the same AMM memory space 
	// so that trades are correctly captured in authoritative snapshots.
	l.marketNodes = l.tokenSinkRouter.MarketNodes

	// PILLAR 2: Siphon Alert Wiring.
	// Connect the economic router's siphon hook to the admin broadcast system
	// to enable real-time infrastructure funding alerts.
	l.tokenSinkRouter.SiphonNotifier = l.broadcastToAdmins

	l.payoutScheduler = NewPayoutScheduler(l.tokenSinkRouter, l, 24*time.Hour) // Daily Governor payouts

	// PILLAR 2: Authoritative State Reconstruction (Local Disk Fallback).
	// Hydrate the economic router with organizational treasuries and AMM reserves before
	// initiating the deeper blockchain-native reconstruction sequence.
	bootstrap := NewBootstrapEngine(l.tokenSinkRouter, l.DataDir)
	diskRecovered, err := bootstrap.BootstrapAuthoritativeState()
	if err != nil {
		log.Printf("[BOOTSTRAP WARNING] Local state recovery failed: %v. Engine will rely on ledger synchronization.\n", err)
	}

	// PILLAR 4: Periodic Economic Persistence.
	// Initialize the background worker to snapshot the Token-Sink router every 15 minutes.
	persistenceWorker := NewPersistenceSyncWorker(l.tokenSinkRouter, l.DataDir, 15*time.Minute)
	persistenceWorker.StartSyncDaemon(ctx)

	// PILLAR 4: Telemetry Initialization
	l.telemetry = NewTelemetryLogger("9090")
	l.telemetry.StartTelemetryServer(ctx)

	// PILLAR 4: Observability Wiring (Kernel setup).
	// Note: Baseline reserve logging moved to end of boot sequence to prevent double-counting.
	if l.tokenSinkRouter != nil && l.tokenSinkRouter.Audit != nil {
		l.tokenSinkRouter.Audit.Telemetry = l.telemetry
	}

	// PILLAR 4: Resiliency Initialization
	l.gracePeriodMatrix = l.NewGracePeriodMatrix(60*time.Second, func(wallet string) {
		l.handleAuthoritativeForfeit(wallet)
	})

	// PILLAR 1-C: Rate Limiting Initialization.
	// Parse ADMIN_WALLETS env var into a slice for admin bypass.
	adminWalletsRaw := os.Getenv("ADMIN_WALLETS")
	var adminWalletList []string
	if adminWalletsRaw != "" {
		for _, w := range strings.Split(adminWalletsRaw, ",") {
			w = strings.TrimSpace(strings.ToLower(w))
			if w != "" {
				adminWalletList = append(adminWalletList, w)
			}
		}
	}
	l.rateLimiter = NewRateLimiterService(l, adminWalletList)
	go l.rateLimiter.CleanupStaleEntries(5 * time.Minute)

	// PILLAR 2: Counterfeiter Rate Limiting — Initialize the per-wallet map.
	l.counterfeitRateLimit = make(map[string]*TokenBucket)

	// Task 3103: Auto-assign per-wallet tiers for known economic/admin wallets.
	for _, w := range adminWalletList {
		l.rateLimiter.SetWalletQuota(w, "admin")
	}

	l.seasonStart = seasonStart
	l.seasonNumber = seasonNum

	l.loadNetworkConfigs()
	l.loadRegisteredTxIDs()
	l.loadLinkedWallets()
	l.loadLeaderboard()                    // Reconstruct Playstyles before sync
	chainRecovered := l.loadEconomyState() // Reconstruct Virtual Balances
	go l.oracleService.LoadOnboardedWalletsFromIndexer(l) // Reconstruct Sybil protection state

	// PILLAR 4: Resilient Ledger Client Initialization (Voi Mainnet)
	if voiCfg, ok := l.availableNetworks["Voi Mainnet"]; ok {
		lb, err := NewLoadBalancedClient(voiCfg.NodeURLs)
		if err == nil {
			l.ledgerClient = lb
			l.ledgerClient.SetProductionMode(true) // Hardened: 329 max rate limits, 5s sync lag, 15m cooldown
			go l.ledgerClient.RunHealthMonitor(ctx)

			// Parse Voi asset/app IDs from configuration
			if voiCfg.AssetID != "" {
				voiAppID, parseErr := strconv.ParseUint(voiCfg.AppID, 10, 64)
				if parseErr == nil {
					l.multiChainRouter.VoiAssetID = voiAppID
				}
			}
		}
	}

	// PILLAR-A: Ethereum Multi-Chain Client Initialization (trust anchor for ETH-based settlement)
	if ethNodes, ok := l.availableNetworks["Ethereum Mainnet"]; ok && len(ethNodes.NodeURLs) > 0 {
		log.Println("[MultiChain] Initializing Ethereum trust anchor...")

		// Extract node URLs with https:// prefix if missing
		var ethUrls []string
		for _, u := range ethNodes.NodeURLs {
			if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
				u = "https://" + u
			}
			ethUrls = append(ethUrls, u)
		}

		if len(ethUrls) > 0 {
			l.ethClient = NewEthereumClient(ctx, ethUrls)
			if l.ethClient != nil && l.ethClient.IsHealthy() {
				log.Printf("[MultiChain] Ethereum trust anchor healthy (vault: %s)\n", l.ethClient.GetVaultAddress())
			} else if l.ethClient != nil {
				log.Println("[MultiChain] WARNING: Ethereum client created but not healthy")
			} else {
				log.Println("[MultiChain] ERROR: Ethereum client initialization failed — NFT settlements will be unavailable")
			}
		}
	}

	// PILLAR-B: Algorand Mainnet Client Initialization
	if algoCfg, ok := l.availableNetworks["Algorand Mainnet"]; ok && len(algoCfg.NodeURLs) > 0 {
		bestClient, err := algod.MakeClient(algoCfg.NodeURLs[0], "")
		if err == nil {
			l.algorandMainnetClient = bestClient

			// Start health monitoring for Algorand Mainnet node
			go func() {
				ticker := time.NewTicker(30 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						nodeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
						status, err := bestClient.Status().Do(nodeCtx)
						cancel()
						if err == nil && status.LastRound > 0 {
							fmt.Printf("[MultiChain] Algorand Mainnet healthy (block: %d)\n", status.LastRound)
						} else if err != nil {
							fmt.Printf("[MultiChain] Algorand Mainnet health check failed: %v\n", err)
						}
					}
				}
			}()

			// Wire multi-chain router with Ethereum trust anchor + Algorand Mainnet
			if l.ethClient != nil && l.ethClient.IsHealthy() {
				log.Println("[MultiChain] Ethereum trust anchor confirmed — wiring into MultiChainRouter")
			} else if l.ethClient != nil {
				log.Println("[MultiChain] WARNING: Ethereum client exists but not healthy — MultiChainRouter will skip ETH layer")
			}

			if l.multiChainRouter == nil {
				l.multiChainRouter = NewMultiChainRouter(l.ledgerClient, bestClient, l.ethClient)
			} else {
				l.multiChainRouter.AlgorandMainnet = bestClient
			}

			// Parse Algorand Mainnet asset/app IDs from configuration
			if algoCfg.AssetID != "" && algoCfg.AppID != "" {
				algoAssetID, parseErr := strconv.ParseUint(algoCfg.AssetID, 10, 64)
				if parseErr == nil {
					l.multiChainRouter.AlgorandMainnetAsset = algoCfg.AssetID
				}
				algoAppID, parseErr := strconv.ParseUint(algoCfg.AppID, 10, 64)
				if parseErr == nil {
					l.multiChainRouter.AlgorandAppID = algoAppID
					fmt.Printf("[MultiChain] Algorand Mainnet wired (app_id: %d)\n", algoAppID)
				}
			} else {
				fmt.Println("[MultiChain] Algorand Mainnet wired (ARC-200 mode disabled — pure ALGO transfers only)")
			}

			fmt.Printf("[MultiChain] Algorand Mainnet initialized from: %s\n", algoCfg.NodeURLs[0])
		} else {
			fmt.Printf("[MultiChain] Failed to initialize Algorand Mainnet client: %v\n", err)
		}
	}

	go l.loadRegistrationsFromIndexer()    // Reconstruct tournament registration state

	// PILLAR 3: Continuous Verification.
	// Initialize the session watchdog to monitor player eligibility.
	l.StartWatchdogEngine(ctx)

	// Start the governor payout daemon
	l.payoutScheduler.StartPayoutEngine(ctx)

	// Start the salary dispenser daemon
	go l.careerService.StartSalaryDispenser(l)

	// PILLAR 6: Blockchain Persistence. Load persistent card cache from blockchain snapshots.
	l.oracleService.LoadPersistentCardCache(l)

	// PILLAR 2: Authoritative Baseline.
	// Log the starting reserves as vetted input ONLY if no previous state was found.
	// This prevents double-counting reserves that are already accounted for in recovered audit counters.
	if !diskRecovered && !chainRecovered {
		if l.tokenSinkRouter != nil && l.tokenSinkRouter.Audit != nil {
			// Synchronize physical balance from chain before baseline logging
			l.oracleService.CheckVaultBalanceOnChain(l)
			
			log.Printf("[ECONOMY] Genesis boot detected. Logging initial reserves for audit: %d micro-VBV\n", l.faucetBalanceMicro)
			l.tokenSinkRouter.Audit.LogInitialReserves(l.faucetBalanceMicro)
		}
	}

	// Record bootstrap metrics after hydration completes
	l.telemetry.RecordBootstrapMetrics(startBoot, true)

	return l, nil
}

// loadNetworkConfigs loads network configurations from the local JSON store.
func (l *Lobby) loadNetworkConfigs() {
	data, err := os.ReadFile("networks.json")
	if err != nil {
		log.Println("[CONFIG] networks.json not found, using defaults.")
		l.availableNetworks["Voi Mainnet"] = NetworkConfig{
			NetworkName:    "Voi Mainnet",
			IndexerURLs:    []string{"https://mainnet-idx.voi.nodly.io", "https://mainnet-idx.voi.network"},
			NodeURLs:       []string{"https://mainnet-api.voi.nodly.io", "https://mainnet-api.voi.network"},
			ExplorerURL:    "https://block.voi.network",
			AppID:          l.rewardAssetID,
			AssetID:        l.rewardAssetID,
			ChainID:        "algorand:wGHE2Pwd1-YdV4EuJFy9u6C24-L-2B05",
			PowerDivisor:   1000000,
			PowerBase:      50,
			IPFSGatewayURL: "https://ipfs.io/ipfs/", // Default public gateway
		}
		l.availableNetworks["Algorand Mainnet"] = NetworkConfig{
			NetworkName:  "Algorand Mainnet",
			IndexerURLs:  []string{"https://mainnet-idx.algonode.cloud", "https://mainnet-idx.algonodly.io"},
			NodeURLs:     []string{"https://mainnet-api.algonode.cloud", "https://mainnet-api.algonodly.io"},
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
			IndexerURLs:  []string{"https://api.etherscan.io"},
			NodeURLs:     []string{"https://eth.llamarpc.com"},
			ExplorerURL:  "https://etherscan.io",
			ChainID:      "eip155:1",
			PowerDivisor: 1e18, // standard ETH decimals
			PowerBase:    100,
		}
		l.availableNetworks["Solana"] = NetworkConfig{
			NetworkName:  "Solana",
			IndexerURLs:  []string{"https://api.mainnet-beta.solana.com"},
			NodeURLs:     []string{"https://api.mainnet-beta.solana.com"},
			ExplorerURL:  "https://solscan.io",
			ChainID:      "solana:5eykt4UsFvXYfy2khQbSsLurFBXY",
			PowerDivisor: 1e9, // standard SOL decimals
			PowerBase:    75,
		}
		l.availableNetworks["Polygon"] = NetworkConfig{
			NetworkName:  "Polygon",
			IndexerURLs:  []string{"https://api.polygonscan.com"},
			NodeURLs:     []string{"https://polygon.llamarpc.com"},
			ExplorerURL:  "https://polygonscan.com",
			ChainID:      "eip155:137",
			PowerDivisor: 1e18,
			PowerBase:    40,
		}
		l.availableNetworks["Bitcoin"] = NetworkConfig{
			NetworkName:  "Bitcoin",
			IndexerURLs:  []string{"https://ordinals.com"},
			NodeURLs:     []string{"https://ordinals.com"},
			ExplorerURL:  "https://ordiscan.com",
			ChainID:      "bip122:000000000019d6689c085ae165831e93",
			PowerDivisor: 1, // Ordinals are individual inscriptions
			PowerBase:    200,
		}
		l.availableNetworks["Flow"] = NetworkConfig{
			NetworkName:  "Flow",
			IndexerURLs:  []string{"https://rest-mainnet.onflow.org"},
			NodeURLs:     []string{"https://access-mainnet-beta.onflow.org"},
			ExplorerURL:  "https://flowscan.org",
			ChainID:      "flow:mainnet",
			PowerDivisor: 1e8,
			PowerBase:    60,
		}
		l.availableNetworks["WAX"] = NetworkConfig{
			NetworkName:  "WAX",
			IndexerURLs:  []string{"https://wax.api.atomicassets.io"},
			NodeURLs:     []string{"https://wax.greymass.com"},
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
	// PILLAR 1: Production Hardening. Check for either legacy or user-defined AppID.
	wcCheck := os.Getenv("WC_PROJECT_ID")
	if wcCheck == "" {
		wcCheck = os.Getenv("AppID")
	}

	secrets := []string{"FAUCET_MNEMONIC", "ADMIN_WALLETS", "VAULT_ADDRESS"}
	for _, s := range secrets {
		if os.Getenv(s) == "" {
			log.Printf("[SECURITY WARNING] Environment variable %s is missing. System functionality may be impaired.\n", s)
		}
	}

	if wcCheck == "" {
		log.Println("[SECURITY WARNING] WalletConnect Project ID (WC_PROJECT_ID or AppID) is missing. Mobile connectivity will FAIL.")
	}

	mnemonicRaw := os.Getenv("FAUCET_MNEMONIC")
	if mnemonicRaw != "" && len(strings.Fields(mnemonicRaw)) != 25 {
		log.Println("[CRITICAL ERROR] FAUCET_MNEMONIC is malformed (expected 25 words). Payouts will FAIL.")
	} else if mnemonicRaw != "" {
		log.Println("[INFO] FAUCET_MNEMONIC validated for length. Faucet Service active.")
	}

	lobby, err := newLobby()
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize Arena Lobby: %v", err)
	}

	// Start the main event loop (Defined in lobby_manager.go)
	go lobby.run()

	// PILLAR 4: Zero-Downtime Deployment & Graceful Shutdown.
	// Intercept SIGTERM (Render/Linux) and Interrupt (Local) to commit final state.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		fmt.Printf("\n [SYSTEM] Signal %v received. Sealing Arena state...\n", sig)

		// PILLAR 4: Shutdown Mutex Guard.
		// Harden the exit sequence by awaiting the Lobby mutex before triggering snapshots.
		// This ensures we don't attempt to archival while a high-load write is in progress.
		lobby.mutex.Lock()
		fmt.Println(" [SYSTEM] Mutex acquired. Initiating graceful archival...")
		lobby.mutex.Unlock()

		// Trigger the centralized Graceful Shutdown sequence (includes Integrity Audit)
		lobby.executeGracefulShutdown()
	}()

	// --- ROUTING ---
	mux := http.NewServeMux()

	// WebSocket Entry Point (CheckOrigin handles WS CORS — no rate limit for WS handshakes)
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(lobby, w, r)
	})

	// --- RATE-LIMITED ROUTING TIER ---
	// Apply per-wallet rate limiting middleware to all API routes.
	// Middleware chain: CORS → Rate Limit → Handler
	// Tier mapping (from ratelimiter.go):
	//   economy-tight   (3 req/min)  → high-risk payouts (reward, wager, loans, black-market)
	//   core-economy    (5 req/min)  → leaderboard, career, faction shop
	//   standard        (10 req/min) → card-stats, auctions, marketplace
	//   wallet-default  (30 req/min) → status, re-sync, admin controls
	//   achievement     (15 req/min) → trophy/achievement endpoints
	//   underworld      (8 req/min)  → underworld contracts, justice missions
	//   default         (20 req/min) → anything unmatched

	mux.HandleFunc("/api/reward", lobby.rateLimiter.WithRateLimit(lobby.handleReward, "economy-tight"))
	mux.HandleFunc("/api/match/wager", lobby.rateLimiter.WithRateLimit(lobby.handleSpectatorWager, "economy-tight"))
	mux.HandleFunc("/api/loans/take", lobby.rateLimiter.WithRateLimit(lobby.loanService.HandleTakeLoan, "economy-tight"))
	mux.HandleFunc("/api/loans/repay", lobby.rateLimiter.WithRateLimit(lobby.loanService.HandleRepayLoan, "economy-tight"))
	mux.HandleFunc("/api/black-market/sell-tokens", lobby.rateLimiter.WithRateLimit(lobby.blackMarketService.HandleSellMarketTokens, "economy-tight"))

	mux.HandleFunc("/api/leaderboard", lobby.rateLimiter.WithRateLimit(lobby.handleLeaderboard, "core-economy"))
	mux.HandleFunc("/api/career/progress", lobby.rateLimiter.WithRateLimit(lobby.HandleGetCareerProgress, "core-economy"))
	mux.HandleFunc("/api/faction/shop/", func(next http.HandlerFunc) http.HandlerFunc {
		return lobby.rateLimiter.WithRateLimit(next, "core-economy")
	}(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/buy") {
			lobby.HandleBuyFactionItem(w, r)
		} else {
			lobby.HandleGetFactionShop(w, r)
		}
	}))

	mux.HandleFunc("/api/card-stats", lobby.rateLimiter.WithRateLimit(lobby.handleCardStats, "standard"))
	mux.HandleFunc("/api/card-details", lobby.rateLimiter.WithRateLimit(lobby.handleGetCardDetails, "standard"))
	mux.HandleFunc("/api/auctions", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			lobby.auctionService.HandleGetAuctions(lobby, w, r)
		case http.MethodPost:
			lobby.auctionService.HandleCreateAuction(lobby, w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "standard"))

	mux.HandleFunc("/api/achievements", lobby.rateLimiter.WithRateLimit(lobby.handleGetAchievements, "achievement"))
	mux.HandleFunc("/api/achievement-stats", lobby.rateLimiter.WithRateLimit(lobby.handleGetAchievementStats, "achievement"))
	mux.HandleFunc("/api/achievement/unlock", lobby.rateLimiter.WithRateLimit(lobby.handleUnlockAchievement, "achievement"))

	mux.HandleFunc("/api/underworld/contracts", lobby.rateLimiter.WithRateLimit(lobby.blackMarketService.HandleGetUnderworldContracts, "underworld"))
	mux.HandleFunc("/api/justice/missions", lobby.rateLimiter.WithRateLimit(lobby.HandleGetJusticeMissions, "underworld"))

	mux.HandleFunc("/api/report-player", lobby.rateLimiter.WithRateLimit(lobby.handlePlayerReport, "default"))
	mux.HandleFunc("/api/re-sync-stats", lobby.rateLimiter.WithRateLimit(lobby.handleReSyncStats, "wallet-default"))
	mux.HandleFunc("/api/season/history", lobby.rateLimiter.WithRateLimit(lobby.handleSeasonHistory, "wallet-default"))

	// Tournament routes — standard rate for registration, wallet-default for history
	mux.HandleFunc("/api/tournament/register", lobby.rateLimiter.WithRateLimit(lobby.tournamentService.HandleTournamentRegister, "standard"))
	mux.HandleFunc("/api/tournament/history", lobby.rateLimiter.WithRateLimit(lobby.tournamentService.HandleTournamentHistory, "wallet-default"))

	// Admin controls — wallet-default (higher budget for admin operations)
	mux.HandleFunc("/api/refill-vault", lobby.rateLimiter.WithRateLimit(lobby.handleRefillVault, "wallet-default"))
	mux.HandleFunc("/api/update-rules", lobby.rateLimiter.WithRateLimit(lobby.handleUpdateRules, "wallet-default"))
	mux.HandleFunc("/api/system-message", lobby.rateLimiter.WithRateLimit(lobby.handleSystemMessage, "wallet-default"))
	mux.HandleFunc("/api/ban-player", lobby.rateLimiter.WithRateLimit(lobby.handleBanPlayer, "wallet-default"))
	mux.HandleFunc("/api/reset-stats", lobby.rateLimiter.WithRateLimit(lobby.handleResetStats, "wallet-default"))
	mux.HandleFunc("/api/maintenance-mode", lobby.rateLimiter.WithRateLimit(lobby.handleMaintenanceMode, "wallet-default"))
	mux.HandleFunc("/api/reward/add", lobby.rateLimiter.WithRateLimit(lobby.handleAdminAddReward, "wallet-default"))
	mux.HandleFunc("/api/reward/remove", lobby.rateLimiter.WithRateLimit(lobby.handleAdminRemoveReward, "wallet-default"))
	mux.HandleFunc("/api/reward/update-base", lobby.rateLimiter.WithRateLimit(lobby.handleUpdateBaseReward, "wallet-default"))
	mux.HandleFunc("/api/reward/update-asset", lobby.rateLimiter.WithRateLimit(lobby.handleUpdateRewardAsset, "wallet-default"))
	mux.HandleFunc("/api/admin/network/add", lobby.rateLimiter.WithRateLimit(lobby.handleAddNetwork, "wallet-default"))
	mux.HandleFunc("/api/admin/set-admin-focus-network", lobby.rateLimiter.WithRateLimit(lobby.handleSetActiveNetwork, "wallet-default"))
	mux.HandleFunc("/api/admin/update-power", lobby.rateLimiter.WithRateLimit(lobby.handleUpdatePowerScaling, "wallet-default"))
	mux.HandleFunc("/api/admin/logs", lobby.rateLimiter.WithRateLimit(lobby.handleGetAdminLogs, "wallet-default"))
	mux.HandleFunc("/api/admin/export-logs", lobby.rateLimiter.WithRateLimit(lobby.handleExportAuditLog, "wallet-default"))
	mux.HandleFunc("/api/admin/simulate-tournament", lobby.rateLimiter.WithRateLimit(lobby.handleSimulateTournament, "wallet-default"))
	mux.HandleFunc("/api/admin/season-rollover", lobby.rateLimiter.WithRateLimit(lobby.handleSeasonRollover, "wallet-default"))
	mux.HandleFunc("/api/admin/sanity-check", lobby.rateLimiter.WithRateLimit(lobby.handleSystemSanityCheck, "wallet-default"))
	mux.HandleFunc("/api/admin/emergency-shutdown", lobby.rateLimiter.WithRateLimit(lobby.handleEmergencyShutdown, "wallet-default"))
	mux.HandleFunc("/api/admin/simulate-mutation-failure", lobby.rateLimiter.WithRateLimit(lobby.handleSimulateMutationFailure, "wallet-default"))
	mux.HandleFunc("/api/admin/simulate-mutation-success", lobby.rateLimiter.WithRateLimit(lobby.handleSimulateMutationSuccess, "wallet-default"))
	mux.HandleFunc("/api/admin/simulate-load", lobby.rateLimiter.WithRateLimit(lobby.handleSimulateLoad, "wallet-default"))
	mux.HandleFunc("/api/admin/gloat-ban", lobby.rateLimiter.WithRateLimit(lobby.handleGloatBan, "wallet-default"))
	mux.HandleFunc("/api/admin/avatar-ban", lobby.rateLimiter.WithRateLimit(lobby.handleAvatarBan, "wallet-default"))
	mux.HandleFunc("/api/admin/commission-audit", lobby.rateLimiter.WithRateLimit(lobby.handleCommissionAudit, "wallet-default"))
	mux.HandleFunc("/api/admin/dlc-registry", lobby.rateLimiter.WithRateLimit(lobby.handleAdminGetDLCRegistry, "wallet-default"))
	mux.HandleFunc("/api/admin/dlc-registry/update", lobby.rateLimiter.WithRateLimit(lobby.handleAdminUpdateDLCRegistry, "wallet-default"))
	mux.HandleFunc("/api/admin/dlc-registry/restock", lobby.rateLimiter.WithRateLimit(lobby.handleAdminRestockDLC, "wallet-default"))
	mux.HandleFunc("/api/admin/mutation-audit", lobby.rateLimiter.WithRateLimit(lobby.handleMutationAudit, "wallet-default"))
	mux.HandleFunc("/api/admin/district-tax-audit", lobby.rateLimiter.WithRateLimit(lobby.handleDistrictTaxAudit, "wallet-default"))
	mux.HandleFunc("/api/v1/redemption_gateway", lobby.rateLimiter.WithRateLimit(lobby.handleRedemptionGateway, "wallet-default"))
	mux.HandleFunc("/api/admin/tax-audit", lobby.rateLimiter.WithRateLimit(lobby.handleTaxAudit, "wallet-default"))
	mux.HandleFunc("/api/admin/start-tournament", lobby.rateLimiter.WithRateLimit(lobby.handleStartTournament, "wallet-default"))
	mux.HandleFunc("/api/admin/open-registration", lobby.rateLimiter.WithRateLimit(lobby.handleOpenRegistration, "wallet-default"))
	mux.HandleFunc("/api/admin/asset-forfeiture", lobby.rateLimiter.WithRateLimit(lobby.handleAssetForfeiture, "wallet-default"))
	mux.HandleFunc("/api/admin/force-payout", lobby.rateLimiter.WithRateLimit(lobby.handleForcePayout, "wallet-default"))
	mux.HandleFunc("/api/admin/simulate-mojo-decay", lobby.rateLimiter.WithRateLimit(lobby.handleSimulateMojoDecay, "wallet-default"))
	mux.HandleFunc("/api/admin/ledger-audit", lobby.rateLimiter.WithRateLimit(lobby.handleLedgerAudit, "wallet-default"))

	// Rivalry system — standard rate
	mux.HandleFunc("/api/rivalry/request", lobby.rateLimiter.WithRateLimit(lobby.HandleRivalryRequest, "standard"))
	mux.HandleFunc("/api/rivalry/action", lobby.rateLimiter.WithRateLimit(lobby.HandleRivalryAction, "standard"))
	mux.HandleFunc("/api/rivalry/state", lobby.rateLimiter.WithRateLimit(lobby.HandleGetRivalryState, "standard"))

	// Courthouse reset — wallet-default (occasional admin use)
	mux.HandleFunc("/api/courthouse/reset", lobby.rateLimiter.WithRateLimit(lobby.courthouseService.HandleCourthouseReset, "wallet-default"))

	// Loan list endpoint — standard (read-heavy but low-risk)
	mux.HandleFunc("/api/loans", lobby.rateLimiter.WithRateLimit(lobby.loanService.HandleGetLoans, "standard"))

	// Black market buy — economy-tight (sensitive trade)
	mux.HandleFunc("/api/black-market/buy", lobby.rateLimiter.WithRateLimit(lobby.blackMarketService.HandleBuyBlackMarket, "economy-tight"))

	// P2-B3: Fenced Goods Marketplace routes
	mux.HandleFunc("/api/black-market/fence-goods", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			lobby.blackMarketService.HandleGetFencedGoods(lobby, w, r)
		case http.MethodPost:
			lobby.blackMarketService.HandleListFenceGoods(lobby, w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "economy-tight"))

	mux.HandleFunc("/api/black-market/buy-stolen", lobby.rateLimiter.WithRateLimit(lobby.blackMarketService.HandleBuyFencedGood, "economy-tight"))

	// Onboarding — economy-tight (sensitive wallet operation)
	mux.HandleFunc("/api/bridge/onboard", lobby.rateLimiter.WithRateLimit(lobby.onboardingService.HandleVoiOnboarding, "economy-tight"))

	// Static Asset Serving (WASM and UI)
	fs := http.FileServer(http.Dir("./Public"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Security: Prevent WASM caching for rapid development cycles
		if r.URL.Path == "/main.wasm" {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		}
		fs.ServeHTTP(w, r)
	})

	// Wrap all routes with the production-grade CORS middleware.
	fmt.Println("-------------------------------------------------")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8088"
	}
	fmt.Printf(" VOICONOMY ARENA SERVER ONLINE: PORT %s\n", port)
	fmt.Println(" WebSocket Switchboard & API Ready               ")
	fmt.Println(" Rate Limiter Active: 7-tier wallet-based system  ")
	fmt.Println("-------------------------------------------------")

	if err := http.ListenAndServe(":"+port, corsMiddleware()(mux)); err != nil {
		log.Fatalf("[FATAL] Server startup failed: %v", err)
	}
