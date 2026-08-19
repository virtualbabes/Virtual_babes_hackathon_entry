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
		justiceService:          &JusticeService{},   // PILLAR 7: Justice Hegemony Path
		justiceHandlers:         &JusticeHandlers{service: nil, playerSvc: &PlayerService{}}, // PILLAR 7: HTTP presentation layer (initialized below)
		evidencePool:            &EvidencePool{ActiveRecords: make(map[string]*RaidEvidence), CollectorMap: make(map[string][]string)}, // PILLAR 13: Forensic evidence pool
		contractEngine:          NewContractEngine(),                                                                                     // PILLAR 3: Underworld Contracts dynamic engine
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

// PILLAR 13: $VBV-Sustained Liquidity Sampling Daemon (24h window for tier gating)
go l.StartLiquiditySamplingDaemon(ctx)

// PILLAR 2: Counterfeiter Rate Limiting — Initialize the per-wallet map.
	l.counterfeitRateLimit = make(map[string]*TokenBucket)

	// PILLAR 7-C: Creator Storefront initialization
	l.creatorStore = NewCreatorStore()

	// PILLAR 7-D: AI Autonomous Economy — AICitizenEngine initialization
	l.aiEngine = NewAICitizenEngine()

	// Initialize justiceHandlers with reference to the economy service (set after bootstrap)
	if l.justiceService != nil && l.playerService != nil {
		l.justiceHandlers.service = l.justiceService
	}

	// PILLAR 1: Seasonal Event Engine — Industrial Loop event lifecycle management (P7-B Task 7101)
	l.seasonEngine = &SeasonalEventEngine{
		ActiveEvents:      make(map[string]*SeasonEvent),
		CurrentRewardPool: make(map[string]*SeasonRewardPool),
	}

	// PILLAR 2 / Phase 7-A: Entity Investment Layer initialization
	l.entityInvestmentService = NewEntityInvestmentService()
	l.dividendTracker = &EntityDividendTracker{
		EntityPools:      make(map[string]*uint64),
		LastDistribution: make(map[string]time.Time),
	}

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

// StartLiquiditySamplingDaemon runs a 24h ticker that samples each player's $VBV balance.
// PILLAR 13: Collects sustained balance history for tier gating validation.
// Runs every 24 hours to capture point-in-time balance across the player base.
func (l *Lobby) StartLiquiditySamplingDaemon(ctx context.Context) {
ticker := time.NewTicker(24 * time.Hour)
defer ticker.Stop()

log.Println("[PILLAR 13] $VBV-Sustained Liquidity Sampling Daemon started (24h window)")

for {
select {
case <-ctx.Done():
log.Println("[PILLAR 13] Liquidity sampling daemon shutting down")
return
case <-ticker.C:
l.CollectLiquiditySamples()
}
}
}

// CollectLiquiditySamples iterates all active players and samples their current VBV balance.
// PILLAR 13: Deterministic sampling — each player's CareerXP receives the micro-balance snapshot.
func (l *Lobby) CollectLiquiditySamples() {
l.mutex.Lock()
defer l.mutex.Unlock()

now := time.Now()
sampledCount := 0

for wallet, stats := range l.leaderboard {
// Initialize CareerXP if needed
if stats.CareerXP == nil {
stats.CareerXP = &CareerXP{
RoleXP:           make(map[string]uint64),
LiquiditySamples: []uint64{},
}
l.leaderboard[wallet] = stats
}

// Convert balance to micro-units (assuming VBVBalance is already in display units)
microBalance := uint64(float64(stats.VBVBalance) * 1_000_000)

// Append sample to sliding window (keep last 14 samples for 2-week history)
stats.CareerXP.LiquiditySamples = append(stats.CareerXP.LiquiditySamples, microBalance)
if len(stats.CareerXP.LiquiditySamples) > 14 {
stats.CareerXP.LiquiditySamples = stats.CareerXP.LiquiditySamples[len(stats.CareerXP.LiquiditySamples)-14:]
}

// Compute and store average sustained balance
if len(stats.CareerXP.LiquiditySamples) > 0 {
sum := uint64(0)
for _, sample := range stats.CareerXP.LiquiditySamples {
sum += sample
}
	stats.CareerXP.AvgSustainedMicro = sum / uint64(len(stats.CareerXP.LiquiditySamples))
}

// PILLAR 13: Demotion grace period check — wire CheckCareerTierGate into liquidity sampling flow
if stats.CareerXP != nil && len(stats.CareerXP.LiquiditySamples) > 0 {
	// Evaluate against Apprentice tier (minimum gate for any career progression)
	gatePass, currentTier, requiredMicro, isDemotionWarning := stats.CareerXP.CheckCareerTierGate(VBVTierApprentice)

	if isDemotionWarning && !stats.CareerXP.DemotionWarningAt.IsZero() {
		// Grace period expired — broadcast demotion warning to client
		walletLower := strings.ToLower(wallet)
		for cid, lwb := range l.clientWallets {
			if lwb == walletLower || cid != "" {
				l.sendToClientLocked(cid, Envelope{Type: "career_tier_demoted", Payload: json.RawMessage(fmt.Sprintf(`{"wallet":"%s","currentTier":%d,"requiredMicro":%d}` , wallet, currentTier, requiredMicro))})
			}
		}
	} else if !gatePass && stats.CareerXP.DemotionWarningAt.IsZero() {
		// First time falling below threshold — issue 7-day warning
		stats.CareerXP.DemotionWarningAt = now
		l.leaderboard[wallet] = stats
	}

l.leaderboard[wallet] = stats
sampledCount++
}

log.Printf("[PILLAR 13] Liquidity sampling complete at %s: %d players sampled\n", now.Format("2006-01-02 15:04:05"), sampledCount)
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
	// ========================================================================
	// PILLAR 7: Justice Hegemony Dashboard — HTTP API Routes (KEY 3.5)
	// ========================================================================
	mux.HandleFunc("/api/justice/dashboard", lobby.rateLimiter.WithRateLimit(lobby.HandleGetJusticeDashboard, "underworld"))
	mux.HandleFunc("/api/justice/use-truth-serum", lobby.rateLimiter.WithRateLimit(lobby.handleUseTruthSerum, "economy-tight"))
	mux.HandleFunc("/api/justice/capture-bounty", lobby.rateLimiter.WithRateLimit(lobby.HandleCaptureBounty, "underworld"))
	// Alias bounty-board to dashboard for frontend compatibility
	mux.HandleFunc("/api/justice/bounty-board", lobby.rateLimiter.WithRateLimit(lobby.HandleGetJusticeDashboard, "underworld"))

	// P2-A: Justice card award + reputation shield endpoints (WS events: justice_card_awarded, shield_active)
	mux.HandleFunc("/api/justice/award-card", lobby.rateLimiter.WithRateLimit(lobby.handleAwardJusticeCard, "economy-tight"))
	mux.HandleFunc("/api/justice/use-rep-shield", lobby.rateLimiter.WithRateLimit(lobby.handleApplyRepShield, "economy-tight"))

	// PILLAR 13: Intel-Agent cyber-intercept endpoint (WS event: cyber_intercept_event)
	mux.HandleFunc("/api/criminality/cyber-intercept", lobby.rateLimiter.WithRateLimit(lobby.handleCyberInterceptWrapper, "underworld"))

	// ========================================================================
	// PILLAR 13 / UNDERWORLD CONTRACTS — Dynamic Contract Engine routes (KEY 3.5)
	// ========================================================================
	mux.HandleFunc("/api/contracts/list", lobby.rateLimiter.WithRateLimit(lobby.handleGetAvailableContracts, "underworld"))
	mux.HandleFunc("/api/contracts/assign", lobby.rateLimiter.WithRateLimit(lobby.handleAssignContract, "economy-tight"))

	// END OF UNDERWORLD CONTRACTS routes
	// ========================================================================

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

	// ========================================================================
	// PILLAR 2: Entity Investment Layer — Player-to-Player Share Allocation + Dividend Distribution (Task 7001)
	// ========================================================================
	mux.HandleFunc("/api/invest/entity", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			lobby.entityInvestmentService.HandleDirectInvest(lobby, w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "economy-tight"))

	mux.HandleFunc("/api/claim/dividends", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			lobby.entityInvestmentService.HandleClaimDividend(lobby, w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "economy-tight"))

	mux.HandleFunc("/api/invest/portfolio", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			lobby.entityInvestmentService.HandleGetPortfolio(lobby, w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "standard"))

	mux.HandleFunc("/api/invest/dividends/history", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			lobby.entityInvestmentService.HandleDividendHistory(lobby, w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "standard"))

	// ========================================================================
	// PILLAR-1 / Task 7102: Seasonal Event Engine — HTTP routes for event lifecycle management (P7-B Industrial Loop)
	// ========================================================================
	mux.HandleFunc("/api/season/events", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			lobby.seasonEngine.HandleListSeasonEvents(lobby, w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "standard"))

	mux.HandleFunc("/api/season/events/join", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			lobby.seasonEngine.HandleJoinSeasonEvent(lobby, w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "economy-tight"))

	mux.HandleFunc("/api/season/events/reward", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			lobby.seasonEngine.HandleClaimSeasonReward(lobby, w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "economy-tight"))

	mux.HandleFunc("/api/season/status", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			lobby.seasonEngine.HandleSeasonStatus(lobby, w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "standard"))

	mux.HandleFunc("/api/season/admin/create-event", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			lobby.seasonEngine.HandleAdminCreateEvent(lobby, w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "wallet-default"))

	mux.HandleFunc("/api/season/admin/end-event", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			lobby.seasonEngine.HandleAdminEndEvent(lobby, w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "wallet-default"))

	mux.HandleFunc("/api/season/admin/update-reward-pool", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			lobby.seasonEngine.HandleAdminUpdateRewardPool(lobby, w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "wallet-default"))

	// ========================================================================
	// PILLAR 7-C: Creator Storefront — HTTP routes for creator economy (P7-C Vision Lines 412-476)
	// ========================================================================
	mux.HandleFunc("/api/creator/store/products", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			category := r.URL.Query().Get("category")
			creatorWallet := r.URL.Query().Get("creator_wallet")
			cs := lobby.getCreatorStore()
			if cs == nil {
				http.Error(w, "creator store not initialized", http.StatusServiceUnavailable)
				return
			}
			products := cs.ListProducts(category, creatorWallet)
			json.NewEncoder(w).Encode(products)
		case http.MethodPost:
			wallet := extractWalletFromRequest(r)
			if wallet == "" {
				http.Error(w, "wallet required", http.StatusBadRequest)
				return
			}
			cs := lobby.getCreatorStore()
			if cs == nil {
				http.Error(w, "creator store not initialized", http.StatusServiceUnavailable)
				return
			}
			var req struct {
				ProductID      string   `json:"product_id"`
				Name           string   `json:"name"`
				Description    string   `json:"description"`
				Category       string   `json:"category"` // asset, dlc, service, cosmetic
				PriceMicroVBV  uint64   `json:"price_micro_vbv"`
				Tags           []string `json:"tags,omitempty"`
				DLCLinks       []string `json:"dlc_links,omitempty"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			err := cs.CreateProduct(req.ProductID, wallet, req.Name, req.Description, req.Category, req.PriceMicroVBV, req.Tags, req.DLCLinks)
			if err != nil {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"status": "product created"})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "economy-tight"))

	mux.HandleFunc("/api/creator/store/product/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		productID := ""
		if len(parts) >= 5 {
			productID = parts[4] // /api/creator/store/product/{id}
		}
		switch r.Method {
		case http.MethodGet:
			cs := lobby.getCreatorStore()
			if cs == nil {
				http.Error(w, "creator store not initialized", http.StatusServiceUnavailable)
				return
			}
			product, err := cs.GetProduct(productID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(product)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "standard"))

	mux.HandleFunc("/api/creator/store/purchase/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		productID := ""
		if len(parts) >= 5 {
			productID = parts[4]
		}
		switch r.Method {
		case http.MethodPost:
			wallet := extractWalletFromRequest(r)
			if wallet == "" {
				http.Error(w, "wallet required", http.StatusBadRequest)
				return
			}
			cs := lobby.getCreatorStore()
			if cs == nil {
				http.Error(w, "creator store not initialized", http.StatusServiceUnavailable)
				return
			}
			tx, err := cs.PurchaseProduct(productID, wallet)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			json.NewEncoder(w).Encode(tx)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "economy-tight"))

	mux.HandleFunc("/api/creator/store/profile/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		walletAddr := ""
		if len(parts) >= 5 {
			walletAddr = parts[4] // /api/creator/store/profile/{wallet}
		}
		switch r.Method {
		case http.MethodGet:
			cs := lobby.getCreatorStore()
			if cs == nil {
				http.Error(w, "creator store not initialized", http.StatusServiceUnavailable)
				return
			}
			profile, err := cs.GetCreatorProfile(walletAddr)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(profile)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "standard"))

	mux.HandleFunc("/api/creator/store/rate/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			wallet := extractWalletFromRequest(r)
			if wallet == "" {
				http.Error(w, "wallet required", http.StatusBadRequest)
				return
			}
			cs := lobby.getCreatorStore()
			if cs == nil {
				http.Error(w, "creator store not initialized", http.StatusServiceUnavailable)
				return
			}
			var req struct {
				ProductID string `json:"product_id"`
				Rating    uint64 `json:"rating"` // 1-5 scale
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			err := cs.RateProduct(req.ProductID, wallet, req.Rating)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"status": "rating submitted"})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "standard"))

	mux.HandleFunc("/api/creator/store/royalty-history", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			productID := r.URL.Query().Get("product_id")
			creatorWallet := r.URL.Query().Get("creator_wallet")
			limitStr := r.URL.Query().Get("limit")
			limit := 50
			if limitStr != "" {
				if lim, err := strconv.Atoi(limitStr); err == nil && lim > 0 && lim <= 100 {
					limit = lim
				}
			}
			cs := lobby.getCreatorStore()
			if cs == nil {
				http.Error(w, "creator store not initialized", http.StatusServiceUnavailable)
				return
			}
			history := cs.GetRoyaltyHistory(productID, creatorWallet, limit)
			json.NewEncoder(w).Encode(history)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "standard"))

	mux.HandleFunc("/api/creator/store/deactivate/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		productID := ""
		if len(parts) >= 5 {
			productID = parts[4]
		}
		switch r.Method {
		case http.MethodPost:
			wallet := extractWalletFromRequest(r)
			if wallet == "" {
				http.Error(w, "wallet required", http.StatusBadRequest)
				return
			}
			cs := lobby.getCreatorStore()
			if cs == nil {
				http.Error(w, "creator store not initialized", http.StatusServiceUnavailable)
				return
			}
			err := cs.DeactivateProduct(productID, wallet)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "product deactivated"})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "standard"))

	mux.HandleFunc("/api/creator/store/reactivate/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		productID := ""
		if len(parts) >= 5 {
			productID = parts[4]
		}
		switch r.Method {
		case http.MethodPost:
			wallet := extractWalletFromRequest(r)
			if wallet == "" {
				http.Error(w, "wallet required", http.StatusBadRequest)
				return
			}
			cs := lobby.getCreatorStore()
			if cs == nil {
				http.Error(w, "creator store not initialized", http.StatusServiceUnavailable)
				return
			}
			err := cs.ReactivateProduct(productID, wallet)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "product activated"})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "standard"))

	// ============================================================================
	// PILLAR 7-D: AI Autonomous Economy — HTTP routes for citizen lifecycle (Task 7301)
	// ===========================================================================

	mux.HandleFunc("/api/ai/citizens/spawn", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			wallet := extractWalletFromRequest(r)
			if wallet == "" {
				http.Error(w, "wallet required", http.StatusBadRequest)
				return
			}
			citizen, err := lobby.aiEngine.SpawnAI(lobby)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to spawn AI: %v", err), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data": map[string]interface{}{
					"wallet":   citizen.Wallet,
					"name":     citizen.Name,
					"career":   citizen.Career,
					"tier":     citizen.Tier,
					"treasury": citizen.Treasury,
				},
			})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "economy-tight"))

	mux.HandleFunc("/api/ai/citizens/stats", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			stats := lobby.aiEngine.GetAIStats()
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    stats,
			})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "standard"))

	mux.HandleFunc("/api/ai/citizens/business/spawn", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			wallet := extractWalletFromRequest(r)
			if wallet == "" {
				http.Error(w, "wallet required", http.StatusBadRequest)
				return
			}
			err := lobby.aiEngine.SpawnBusiness(wallet)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to spawn business: %v", err), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "business spawned"})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "economy-tight"))

	mux.HandleFunc("/api/ai/citizens/list", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			citizens := lobby.aiEngine.GetAllCitizens()
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    citizens,
			})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "standard"))

	// Player secondary-sale with automatic creator royalty calculation + economy routing.
	// ============================================================================

	mux.HandleFunc("/api/creator/store/resell", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			wallet := extractWalletFromRequest(r)
			if wallet == "" {
				http.Error(w, "wallet required", http.StatusBadRequest)
				return
			}

			var req struct {
				ProductID          string `json:"product_id"`
				SalePriceMicroVBV  uint64 `json:"sale_price_micro_vbv"`
				BuyerWallet        string `json:"buyer_wallet"` // wallet of the buyer (not sender)
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}

			cs := lobby.getCreatorStore()
			if cs == nil {
				http.Error(w, "creator store not initialized", http.StatusServiceUnavailable)
				return
			}

			// Validate product exists and get creator wallet for ownership check
			product, err := cs.GetProduct(req.ProductID)
			if err != nil || product.CreatorWallet == "" {
				http.Error(w, "product not found or invalid", http.StatusBadRequest)
				return
			}

			// Validate sale price is positive (uint64 ensures non-negative; check > 0)
			if req.SalePriceMicroVBV == 0 {
				http.Error(w, "sale price must be greater than zero", http.StatusBadRequest)
				return
			}

			// Process secondary sale — calculates 10% royalty to creator automatically
			tx, err := cs.ProcessSecondarySale(req.ProductID, req.BuyerWallet, wallet, req.SalePriceMicroVBV)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Route economy-wide revenue split with CreatorRoyalty field set for secondary-sale context.
			// This seeds the royalty tracking in TokenSinkRouter audit trail (actual creator wallet routing
			// is handled by ProcessSecondarySale above). The 10% royalty amount flows to creator automatically.
			if lobby.tokenSinkRouter != nil {
				matrix := RevenueSplitMatrix{
					FaucetShare:     0.5, // 50% to faucet (standard sink)
					ClubShare:       0.3, // 30% to club treasury
					GovernanceShare: 0.2, // 20% to governor/district
					CreatorRoyalty:  DefaultRoyaltyRate, // 10% creator royalty on secondary sale
				}
				if err := lobby.tokenSinkRouter.RouteCriminalTax(
					"CREATOR_SECONDARY_SALE",
					req.SalePriceMicroVBV,
					matrix,
					0,   // no specific club (economy-wide routing)
					"",  // no specific district
				); err != nil {
					log.Printf("[ROYALTY_ROUTE] WARNING: economy split failed for secondary sale %s: %v", tx.ID, err)
					// Non-fatal — royalty already credited to creator via ProcessSecondarySale above
				}
			}

			// WS broadcast to all connected clients for real-time royalty tracking + frontend sync.
			payload := fmt.Sprintf(`{"product_id":"%s","creator_wallet":"%s","seller_wallet":"%s","amount_micro_vbv":%d,"royalty_paid_micro_vbv":%d,"timestamp":"%s"}`,
				tx.ProductID, tx.CreatorWallet, tx.SellerWallet, tx.AmountMicroVBV, tx.RoyaltyPaidMicroVBV, tx.Timestamp.Format(time.RFC3339))

			lobby.mutex.Lock()
			if cid := lobby.getClientIDFromWalletLocked(wallet); cid != "" {
				lobby.sendToClientLocked(cid, Envelope{Type: "creator_royalty_paid", Payload: json.RawMessage(payload)})
			}
			// Also broadcast to buyer for portfolio sync
			if req.BuyerWallet != wallet && lobby.getClientIDFromWalletLocked(req.BuyerWallet) != "" {
				bid := lobby.getClientIDFromWalletLocked(req.BuyerWallet)
				lobby.sendToClientLocked(bid, Envelope{Type: "creator_royalty_paid", Payload: json.RawMessage(payload)})
			}
			// Broadcast to creator so they see royalty earned in real-time
			if cid := lobby.getClientIDFromWalletLocked(tx.CreatorWallet); cid != "" {
				lobby.sendToClientLocked(cid, Envelope{Type: "creator_royalty_received", Payload: json.RawMessage(payload)})
			}
			lobby.mutex.Unlock()

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":                "secondary_sale_complete",
				"royalty_transaction_id": tx.ID,
				"creator_royalty_micro_vbv": tx.RoyaltyPaidMicroVBV,
				"seller_net_proceeds":   req.SalePriceMicroVBV - tx.RoyaltyPaidMicroVBV,
			})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "economy_tight"))

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
}

// ============================================================================
// PILLAR 7: Justice Dashboard — Lobby wrapper methods (KEY 3.5)
// These delegate to the JusticeHandlers instance stored on the Lobby struct.
// ============================================================================

func (l *Lobby) HandleGetJusticeDashboard(w http.ResponseWriter, r *http.Request) {
	h := &JusticeHandlers{lobby: l}
	h.handleGetDashboard(w, r)
}

func (l *Lobby) handleUseTruthSerum(w http.ResponseWriter, r *http.Request) {
	h := &JusticeHandlers{lobby: l}
	h.handleUseTruthSerum(w, r)
}

func (l *Lobby) HandleCaptureBounty(w http.ResponseWriter, r *http.Request) {
	h := &JusticeHandlers{lobby: l}
	h.handleCaptureBounty(w, r)
}

func (l *Lobby) handleAwardJusticeCard(w http.ResponseWriter, r *http.Request) {
	h := &JusticeHandlers{lobby: l}
	h.handleAwardJusticeCard(w, r)
}

func (l *Lobby) handleApplyRepShield(w http.ResponseWriter, r *http.Request) {
	h := &JusticeHandlers{lobby: l}
	h.handleApplyRepShield(w, r)
}

// PILLAR 13: Intel-Agent Cyber-Intercept — Lobby wrapper method (KEY 3.5)
// ============================================================================

// getCreatorStore returns the CreatorStore instance on the Lobby.
func (l *Lobby) getCreatorStore() *CreatorStore {
	return l.creatorStore
}

func (l *Lobby) handleCyberInterceptWrapper(w http.ResponseWriter, r *http.Request) {
	l.handleCyberIntercept(w, r)
}

// Underworld Contracts — Lobby wrapper methods (KEY 3.5)

func (l *Lobby) handleGetAvailableContracts(w http.ResponseWriter, r *http.Request) {
	wallet := extractWalletFromRequest(r)
	if wallet == "" {
		http.Error(w, "wallet required", http.StatusBadRequest)
		return
	}
	contracts := l.contractEngine.HandleGetAvailableContracts(l, wallet)
	json.NewEncoder(w).Encode(contracts)
}

func (l *Lobby) handleAssignContract(w http.ResponseWriter, r *http.Request) {
	wallet := extractWalletFromRequest(r)
	if wallet == "" {
		http.Error(w, "wallet required", http.StatusBadRequest)
		return
	}

	var req struct {
		TemplateID string `json:"template_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	contract, err := l.contractEngine.HandleAssignContract(l, wallet, req.TemplateID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	json.NewEncoder(w).Encode(contract)
}

