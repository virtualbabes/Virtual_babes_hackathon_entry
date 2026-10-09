//go:build !js && !wasm

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// killExistingServer kills any process listening on the given port.
//
// WHY IT LIVES IN server.go RATHER THAN AN ENTRYPOINT: it has TWO callers on OPPOSITE sides of a build
// tag — server_main.go (tags: !js && !wasm && !console) and the console entrypoint console_server.go
// (tag: console). While it sat in server_main.go the console target could not resolve it, so that target
// did not compile at all. server.go's tag (!js && !wasm) is the one BOTH targets satisfy, and this file
// is already the home of the helpers both entrypoints share (newLobby, serveWs, getDataPath — the
// console entrypoint's own header says so).
//
// HONEST LIMIT: the body is Windows-shaped (`netstat -ano` + `taskkill`). On a Linux console build the
// first exec fails and the function RETURNS — port hygiene is skipped, and the bind then fails loudly
// rather than silently. The move does not change that; it is written down here because it was previously
// implied by the primary entrypoint's build tag rather than stated.
func killExistingServer(port string) {
	// Use netstat to find the PID listening on the port
	cmd := exec.Command("netstat", "-ano")
	output, err := cmd.Output()
	if err != nil {
		return
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "LISTENING") {
			continue
		}
		if !strings.Contains(line, ":"+port) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		pid := fields[len(fields)-1]
		if pid == "0" {
			continue
		}
		// Don't kill ourselves
		if pid == strconv.Itoa(os.Getpid()) {
			continue
		}
		// Kill the process
		killCmd := exec.Command("taskkill", "/F", "/PID", pid)
		killCmd.Run()
	}
}

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
		pendingTxIDs:            make(map[string]time.Time), // txid_memo.go: in-flight reservations (never persisted)
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
		creatorStore:            &CreatorStore{},
		seasonEngine:            &SeasonalEventEngine{},
		entityInvestmentService: &EntityInvestmentService{},
		loanService:             &LoanService{},
		auctionService:          &AuctionService{},
		blackMarketService:      &BlackMarketService{},
		counterfeitService:      &CounterfeitService{},
		narrativeService:        &NarrativeService{},
		nautilusDEXPathService:  &NautilusDEXPathService{}, // PILLAR 2: Console Creator Payouts
		playerService:           &PlayerService{},
		justiceService:          &JusticeService{},                                            // PILLAR 7: Justice Hegemony Path
		justiceHandlers:         &JusticeHandlers{service: nil, playerSvc: &PlayerService{}},  // PILLAR 7: HTTP presentation layer (initialized below)
		evidencePool:            &EvidencePool{ActiveRecords: make(map[string]*RaidEvidence)}, // PILLAR 13: Forensic evidence pool
		identityBridge:          NewIdentityBridge(),                                          // PILLAR 7-E: Cross-platform identity ownership
		contractEngine:          NewContractEngine(),                                          // PILLAR 3: Underworld Contracts dynamic engine
		matchHandshakers:        make(map[string]*SyncHandshaker),

		linkedWallets:       make(map[string]WalletLinkInfo),
		loans:               make(map[string]*Loan),
		rumors:              make(map[string]*Rumor),
		auctions:            make(map[string]*Auction),
		rewardStack:         make(map[string]uint64),
		playerBalances:      make(map[string]uint64),
		initialRewards:      make(map[string]uint64),
		register:            make(chan *Client, 256),
		unregister:          make(chan *Client, 256),
		broadcast:           make(chan []byte, 256),
		onboardedWallets:    make(map[string]bool),         // Initialize the new map
		fencedListings:      make(map[string]FenceListing), // P2-B3: Fenced Goods Marketplace
		onboardingSemaphore: make(chan struct{}, 5),        // Limit concurrent bridge operations
		oracleSemaphore:     make(chan struct{}, 10),       // Limit concurrent indexer queries
		envoiCache:          make(map[string]string),
		lastSeenDistricts:   make(map[string]string),
		lastActive:          make(map[string]time.Time),
		treasuryAverages:    make(map[string]float64),
		// Workstream C: the platform purchase/lease GRANTS. Constructed WITH the Lobby, so no door can be
		// reached before its owner exists.
		consoleEntitlements: NewConsoleEntitlementRegistry(),

		tenantVaults:        NewTenantVaultRegistry(),
		treasuryCrashed:     make(map[string]bool),
		vaultAddress:        os.Getenv("VAULT_ADDRESS"),
		WCProjectID:         wcID,                  // Load WalletConnect Project ID (AppID)
		DataDir:             os.Getenv("DATA_DIR"), // Persistent volume path
		maxFaucetCapacity:   10000.0,
		adminFocusNetwork:   "Voi Mainnet",
		maintenancePriority: "info",
	}

	// Initialize reward configuration.
	//
	// BASE_REWARD is read as EXACT integer micro units (reward_registry.go): the previous
	// strconv.ParseUint silently turned the DOCUMENTED spelling "5.0" into 0, i.e. a dead payout
	// path. BASE_REWARD_MICRO is the precise form; BASE_REWARD is accepted in whole units.
	baseRewardMicro := resolveEnvBaseRewardMicro()
	l.baseReward = baseRewardMicro
	l.initialBaseReward = l.baseReward
	l.rewardAssetID = os.Getenv("REWARD_ASSET_ID")
	l.avoiAssetID = os.Getenv("AVOI_ASSET_ID")
	if l.rewardAssetID == "" {
		log.Println("[REWARD CONFIG WARNING] REWARD_ASSET_ID is not set, so no primary reward token can be registered: rewards have no asset to be paid in.")
	}
	// Seed the registry's env-owned primary entry and make the template agree with it.
	l.reconcileRewardRegistryLocked()

	// PILLAR 2: Economic Engine Initialization
	l.tokenSinkRouter = NewTokenSinkRouter(&l.faucetBalanceMicro, &l.AdminMaintenancePool)

	// PILLAR 2: Authoritative Map Linking.
	// Ensure the Lobby and the Router share the same AMM memory space
	// so that trades are correctly captured in authoritative snapshots. Taken under the map's ONE owner
	// mutex (the router's): the lobby field is a second NAME for the same map, and the alias gate
	// (`market_nodes_alias_gate_test.go`) requires every access to it to hold that mutex.
	l.tokenSinkRouter.Mu.Lock()
	l.marketNodes = l.tokenSinkRouter.MarketNodes
	l.tokenSinkRouter.Mu.Unlock()

	// PILLAR 2: Siphon Alert Wiring.
	// Connect the economic router's siphon hook to the admin broadcast system
	// to enable real-time infrastructure funding alerts.
	l.tokenSinkRouter.SiphonNotifier = l.broadcastToAdmins

	l.payoutScheduler = NewPayoutScheduler(l.tokenSinkRouter, l, 24*time.Hour) // Daily Governor payouts

	// PILLAR 2: Authoritative State Reconstruction (Local Disk Fallback).
	// Hydrate the economic router with organizational treasuries and AMM reserves before
	// initiating the deeper blockchain-native reconstruction sequence.
	bootstrap := NewBootstrapEngine(l.tokenSinkRouter, l, l.DataDir)
	diskRecovered, err := bootstrap.BootstrapAuthoritativeState()
	if err != nil {
		log.Printf("[BOOTSTRAP WARNING] Local state recovery failed: %v. Engine will rely on ledger synchronization.\n", err)
	}

	// PILLAR 4: Periodic Economic Persistence.
	// Initialize the background worker to snapshot the Token-Sink router every 15 minutes.
	persistenceWorker := NewPersistenceSyncWorker(l.tokenSinkRouter, l, l.DataDir, 15*time.Minute)
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
	l.aiEngine.lobby = l
	l.aiEngine.StartBehavioralLoop()
	// P7-D "Remember": rehydrate the persistent AI civilization across restarts.
	if err := l.aiEngine.LoadCitizens(); err != nil {
		log.Printf("[newLobby] AI citizen rehydration skipped: %v", err)
	}
	// §15 v3 dev seed: ensure at least 2 citizens exist for domestic/faith testing
	if len(l.aiEngine.GetAllCitizens()) == 0 {
		if _, err := l.aiEngine.SpawnAI(l, "DEV_SEED_OWNER", "Base", "Seed_Citizen_1"); err == nil {
			l.aiEngine.SpawnAI(l, "DEV_SEED_OWNER", "Base", "Seed_Citizen_2")
			log.Printf("[DEV] Seeded 2 test AI citizens for domestic/faith system testing")
		}
	}

	// KEY 3.5 Slice 2: Item Shop archetype registry (v4 §14) — persisted, mutable.
	l.itemRegistry = NewItemRegistry()
	if err := l.itemRegistry.Load(l); err != nil {
		log.Printf("[newLobby] Item registry rehydration skipped: %v", err)
	}

	// Initialize justiceHandlers with reference to the economy service (set after bootstrap)
	if l.justiceService != nil && l.playerService != nil {
		l.justiceHandlers.service = l.justiceService
	}

	// PILLAR 1: Seasonal Event Engine — Industrial Loop event lifecycle management (P7-B Task 7101)
	l.seasonEngine = &SeasonalEventEngine{
		ActiveEvents:      make(map[string]*SeasonEvent),
		CurrentRewardPool: make(map[string]*SeasonRewardPool),
	}
	l.InitRivalryEngine() // §25.10
	l.InitAssetLife()     // §26.4 pets / §25.6 vehicles / §25.7 world-content
	l.InitThemeEngine()   // §27 / KEY 3.5: ThemeEngine (ThemeVector + World-Dynamics Signature)
	l.bondedAssets = NewBondedAssetRegistry() // §23.5: Bonded Asset Registry
	if err := l.bondedAssets.Load(l); err != nil {
		log.Printf("[BondedAsset] Warning: failed to load bonded assets: %v", err)
	}
	// §10.6/§10.7 DERIVATIVES: derive the light renditions of the placeholder pack in the
	// BACKGROUND (slideshow frames first). Started here so BOTH binaries get it, and started
	// without being awaited: boot must never wait on 117 PNG decodes, and every reader falls back
	// to the original pack art (reported as pending) until a rendition exists.
	StartPlaceholderDerivativeWarmup()
	l.localModelPromotions = NewLocalModelPromotionRegistry() // §24.5/§24.6: Local-Model Promotion bridge
	// A6 / Stage A: the TENANT VAULT registry loads BESIDE the other registries, because a registry that
	// is constructed but never read back is a file written and never used.
	if err := l.tenantVaults.Load(l); err != nil {
		log.Printf("[TenantVault] Failed to load the tenant vault registry: %v\n", err)
	}
	// Workstream C: the console entitlement registry loads BESIDE the other registries, for the same
	// reason — a registry that is constructed and never read back is a file written and never used.
	if err := l.consoleEntitlements.Load(l); err != nil {
		log.Printf("[ConsoleEntitlement] Failed to load the console entitlement registry: %v\n", err)
	}
	// The platform verifiers come from the ENVIRONMENT (secrets are env-only) and the outcome is STATED:
	// with none configured the door refuses by design, and boot should say so rather than let an operator
	// read a silent zero.
	if n := syncConsoleVerifiersFromEnv(); n > 0 {
		log.Printf("[ConsoleEntitlement] %d platform purchase verifier(s) registered from the environment\n", n)
	} else {
		log.Printf("[ConsoleEntitlement] No platform purchase verifier is configured (set CONSOLE_VERIFIER_URL_<PLATFORM> and CONSOLE_VERIFIER_SECRET_<PLATFORM>): the entitlement door refuses with its blockers\n")
	}

	if err := l.localModelPromotions.Load(l); err != nil {
		log.Printf("[LocalModel] Warning: failed to load promotions: %v", err)
	}
	l.entityEvents = NewEntityEventEngine() // §30 Pet World: entity event + stat + power-overlay engine

	// §32 Faith: seed the 24 religions on first boot (faucet-owned, ungoverned)
	if loaded := religionGov.Load(l); loaded > 0 {
		log.Printf("[Faith] Loaded %d religions from disk", loaded)
	} else {
		religionGov.InitializeFaithReligions(l)
		log.Printf("[Faith] Seeded %d faucet-owned religions", len(religionGov.Religions))
	}

	// §32 Faith: wire global lobby ref for faith_church.go to create Club{Type:"Faith"}
	SetGlobalLobbyRef(l)

	// §32 Faith: load persisted churches
	if loaded := faithChurchEngine.Load(l); loaded > 0 {
		log.Printf("[Faith] Loaded %d churches from disk", loaded)
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
	l.loadLeaderboard()                                   // Reconstruct Playstyles before sync
	chainRecovered := l.loadEconomyState()                // Reconstruct Virtual Balances
	go l.oracleService.LoadOnboardedWalletsFromIndexer(l) // Reconstruct Sybil protection state

	// PILLAR 4: Resilient Ledger Client Initialization (Primary Network)
	primaryNetwork := os.Getenv("DEFAULT_NETWORK")
	if primaryNetwork == "" {
	    primaryNetwork = "VOI"
	}
	l.adminFocusNetwork = primaryNetwork

	if primaryNetwork == "VOI" {
	    // Primary: Voi Mainnet
	    if voiCfg, ok := l.availableNetworks["Voi Mainnet"]; ok {
	        lb, err := NewLoadBalancedClient(voiCfg.NodeURLs, voiCfg.AlgodToken)
	        if err == nil {
	            l.ledgerClient = lb
	            l.ledgerClient.SetProductionMode(true)
	            go l.ledgerClient.RunHealthMonitor(ctx)
	            fmt.Printf("[MultiChain] Voi Mainnet healthy (primary network)\n")
	        }
	    }

	    // Secondary: Algorand Mainnet (metadata only, optional)
	    if algoCfg, ok := l.availableNetworks["Algorand Mainnet"]; ok && len(algoCfg.NodeURLs) > 0 {
	        algoAssetID := l.avoiAssetID // Use Algorand-specific asset, NOT VBV
	        algoCluster, err := NewLoadBalancedClient(algoCfg.NodeURLs, algoCfg.AlgodToken)
	        if err == nil {
	            l.algorandMainnetClient = algoCluster
	            l.multiChainRouter = NewMultiChainRouter(l.ledgerClient, algoCluster, l.ethClient)
	            if algoAssetID != "" {
	                l.multiChainRouter.AlgorandMainnetAsset = algoAssetID
	            }
	            fmt.Println("[MultiChain] Algorand Mainnet initialized (secondary)")
	        }
	    }
	} else {
	    // Primary: Algorand Mainnet
	    if algoCfg, ok := l.availableNetworks["Algorand Mainnet"]; ok && len(algoCfg.NodeURLs) > 0 {
	        algoCluster, err := NewLoadBalancedClient(algoCfg.NodeURLs, algoCfg.AlgodToken)
	        if err == nil {
	            l.ledgerClient = algoCluster
	            l.algorandMainnetClient = algoCluster
	            go algoCluster.RunHealthMonitor(ctx)
	            l.multiChainRouter = NewMultiChainRouter(l.ledgerClient, algoCluster, l.ethClient)
	            if algoCfg.AssetID != "" {
	                l.multiChainRouter.AlgorandMainnetAsset = algoCfg.AssetID
	            }
	            fmt.Printf("[MultiChain] Algorand Mainnet healthy (primary network)\n")
	        }
	    }

	    // Secondary: Voi Mainnet
	    if voiCfg, ok := l.availableNetworks["Voi Mainnet"]; ok {
	        voiClient, err := NewLoadBalancedClient(voiCfg.NodeURLs, voiCfg.AlgodToken)
	        if err == nil {
	            if l.multiChainRouter == nil {
	                l.multiChainRouter = NewMultiChainRouter(voiClient, nil, l.ethClient)
	            }
	            fmt.Println("[MultiChain] Voi Mainnet initialized (secondary)")
	        }
	    }
	}

	// ── THE VOI APP ID IS APPLIED AFTER THE ROUTER EXISTS ──────────────────────
	//
	// This assignment used to live INSIDE the Voi branch above, which runs BEFORE
	// the Algorand branch that CONSTRUCTS l.multiChainRouter — so it wrote through
	// a NIL pointer. It never fired only because networks.json shipped an EMPTY
	// asset_id: filling it in (2026-09-17) panicked the boot with a nil dereference
	// at that exact line. Moving it here is the fix, and the nil check stops a
	// registry with no reachable chain from crashing the process.
	if voiCfg, ok := l.availableNetworks["Voi Mainnet"]; ok && voiCfg.AssetID != "" {
		if voiAppID, parseErr := strconv.ParseUint(voiCfg.AppID, 10, 64); parseErr == nil {
			if l.multiChainRouter != nil {
				l.multiChainRouter.VoiAssetID = voiAppID
				log.Printf("[MultiChain] Voi app id %d applied (asset_id %s).\n", voiAppID, voiCfg.AssetID)
			} else {
				log.Println("[MultiChain] WARNING: the Voi app id could not be applied: the multi-chain router was not constructed (check the Algorand entry's node_urls).")
			}
		} else {
			log.Printf("[MultiChain] WARNING: Voi app_id %q is not a decimal id, so the Voi app-call path stays disabled.\n", voiCfg.AppID)
		}
	}

	go l.loadRegistrationsFromIndexer() // Reconstruct tournament registration state

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
			// NOTE: the $VBV pool balance is no longer synced proactively at boot.
			// It is fetched on-demand (throttled) when a user opens the Faucet Dashboard
			// via Lobby.refreshVaultStateIfStale(), so the server does not probe every
			// cluster node unprompted (which got us blacklisted by provider endpoints).
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
	// Local-node API tokens (for authenticated algod/Voi nodes). Empty = no token (public RPC).
	voiToken := os.Getenv("ALGOD_TOKEN_VOI")
	algoToken := os.Getenv("ALGOD_TOKEN_ALGO")
	// IPFS_API_KEY is the ONE env form of the operator's IPFS gateway credential. It is
	// deliberately NOT persisted to the registry file (networks.json is GIT-TRACKED):
	// see network_registry_view.go for the projection both doors use.
	ipfsKey := os.Getenv("IPFS_API_KEY")
	data, err := os.ReadFile("networks.json")
	if err != nil {
		log.Println("[CONFIG] networks.json not found, using defaults.")
		l.availableNetworks["Voi Mainnet"] = NetworkConfig{
			NetworkName:    "Voi Mainnet",
			IndexerURLs:    []string{"https://mainnet-idx.voi.nodely.dev"},
			NodeURLs:       []string{"https://mainnet-api.voi.nodely.dev"},
			ExplorerURL:    "https://block.voi.network/explorer",
			AppID:          l.rewardAssetID,
			AssetID:        l.rewardAssetID,
			ChainID:        "algorand:r20fSQI8gWe_kFZziNonSPCXLwcQmH_n",
			PowerDivisor:   1000000,
			PowerBase:      50,
			IPFSGatewayURL: "https://ipfs.io/ipfs/", // Default public gateway
			AlgodToken:     voiToken,
		}
		l.availableNetworks["Algorand Mainnet"] = NetworkConfig{
			NetworkName:  "Algorand Mainnet",
			IndexerURLs:  []string{"https://mainnet-idx.algonode.cloud", "https://mainnet-idx.algonodly.io"},
			NodeURLs:     []string{"https://mainnet-api.algonode.cloud", "https://mainnet-api.algonodly.io"},
			ExplorerURL:  "https://allo.info",
			AppID:        "0",             // No game app on Algo, assets only
			AssetID:      l.avoiAssetID,   // Use Algorand-specific asset (AVoi), NOT VBV
			ChainID:      "algorand:wGHE2Pwdvd7S12BL5FaOP20EGYesN73k",
			PowerDivisor: 1000000,
			PowerBase:    50,
			AlgodToken:   algoToken,
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
		l.threadEnvSecrets(voiToken, algoToken, ipfsKey)
		l.saveNetworkConfigs()
		return
	}
	// A registry file exists: parse it. A malformed file must NOT silently answer with an
	// empty registry (which would look exactly like "no networks are configured"), so the
	// decode failure is reported loudly. Whatever parsed is still kept and used.
	l.mutex.Lock()
	if err := json.Unmarshal(data, &l.availableNetworks); err != nil {
		log.Printf("[CONFIG ERROR] networks.json could not be parsed (%v). The registry may be incomplete: repair or delete it with the server stopped, then restart.\n", err)
	}
	l.mutex.Unlock()
	l.threadEnvSecrets(voiToken, algoToken, ipfsKey)
}

// threadEnvSecrets applies the ENVIRONMENT-owned secret material to the loaded registry.
// Called by BOTH branches of loadNetworkConfigs, so a token reaches the map exactly once
// whether or not a registry file existed, and no secret has to be stored on disk to survive a
// restart. Owner of this rule: network_registry_view.go.
func (l *Lobby) threadEnvSecrets(voiToken, algoToken, ipfsKey string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if cfg, ok := l.availableNetworks["Voi Mainnet"]; ok {
		cfg.AlgodToken = voiToken
		if ipfsKey != "" && cfg.IPFSAPIKey == "" {
			cfg.IPFSAPIKey = ipfsKey
		}
		l.availableNetworks["Voi Mainnet"] = cfg
	}
	if cfg, ok := l.availableNetworks["Algorand Mainnet"]; ok {
		cfg.AlgodToken = algoToken
		if ipfsKey != "" && cfg.IPFSAPIKey == "" {
			cfg.IPFSAPIKey = ipfsKey
		}
		l.availableNetworks["Algorand Mainnet"] = cfg
	}
	// The operator's IPFS credential is a server-wide gateway credential, so it applies to any
	// OTHER network that was not given one (metadata lookups can run on any chain).
	if ipfsKey != "" {
		for name, cfg := range l.availableNetworks {
			if cfg.IPFSAPIKey == "" {
				cfg.IPFSAPIKey = ipfsKey
				l.availableNetworks[name] = cfg
			}
		}
	}
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

		// THE BALANCE IS READ FROM THE AUTHORITATIVE LEDGER MAP, IN INTEGER MICRO-UNITS.
		//
		// This line used to be `uint64(float64(stats.VBVBalance) * 1_000_000)`. MEASURED: the field
		// `VBVBalance` is ASSIGNED NOWHERE in this repository — `common_types.go` even labels it
		// "authoritative = playerBalances map on Lobby" — so EVERY sample was 0, every
		// `AvgSustainedMicro` was 0, and the $VBV-sustained career gate could never be met by any
		// player. The float multiply was an Architecture-Ledger violation on top of that. The
		// authoritative map is already in micro-units (`career.go` credits `netSalaryMicro` into
		// it; `theme_engine.go` compares it against `2500*1000000`), so it is read directly — at
		// the key resolved case-insensitively, because the stored spelling is whatever the engine
		// was handed.
		microBalance := l.playerBalances[l.balanceKeyLocked(wallet)]

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

		// THE CAREER LIFECYCLE. `career_path.go` owns the RULES (warn -> grace -> demote, and the
		// WITHDRAWAL of the warning when the balance recovers). This daemon is what samples the
		// balance, so this is where the lifecycle runs. TWO DEFECTS CLOSE HERE AT ONCE:
		//
		//  1. Nothing was ever DEMOTED. The old code only ever set a warning clock, and its
		//     `gatePass` test was false for a player whose balance was FINE (see the gate fix in
		//     `CheckCareerTierGate`), so the "career tier demoted" event could fire for a funded
		//     player while a genuinely unfunded one was never demoted at all.
		//  2. The broadcast condition was `lwb == walletLower || cid != ""`, whose second clause is
		//     ALWAYS TRUE, so every connected client received every other player's career demotion.
		//     A wallet, a role and a balance shortfall are not public facts. Events are now
		//     addressed to the ONE wallet they concern; an offline player misses nothing, because
		//     the warning is stored on the record and the panel renders it on the next read.
		l.leaderboard[wallet] = stats
		for _, ev := range l.EvaluateCareerStandingLocked(wallet) {
			l.NotifyCareerStandingLocked(ev)
		}

		// THE UNLOCK ADVICE (`career_path.go` §7.5). "Notified, not forced": a career that has just
		// become UNLOCKED produces ONE notice to THIS wallet and NOTHING ELSE — no promotion, no
		// queue, no deadline. It runs beside the standing events because this daemon is what samples
		// the balance, and the sustained balance is one of the gates the unlock is derived from.
		for _, ev := range l.CareerUnlockNoticesLocked(wallet) {
			l.NotifyCareerUnlockLocked(ev)
		}
		sampledCount++
	}

	log.Printf("[PILLAR 13] Liquidity sampling complete at %s: %d players sampled\n", now.Format("2006-01-02 15:04:05"), sampledCount)
}

// saveNetworkConfigs persists the current network configurations to disk.
//
// IT NEVER WRITES A SECRET. networks.json is GIT-TRACKED, so an env-threaded algod token or
// an IPFS api key reaching it would be a committed credential. The redaction lives in ONE
// place (network_registry_view.go), which both this writer and the client broadcast use — so
// the file and the wire cannot disagree about what is public.
func (l *Lobby) saveNetworkConfigs() {
	l.mutex.RLock()
	persisted := publicNetworkConfigsForDisk(l.availableNetworks)
	tokens, ipfsKeys, headerSets, secretNetworks := withheldSecretSummary(l.availableNetworks)
	l.mutex.RUnlock()

	data, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		log.Printf("[CONFIG ERROR] cannot serialise the network registry (%v); networks.json left unchanged.\n", err)
		return
	}
	if err := os.WriteFile("networks.json", data, 0644); err != nil {
		log.Printf("[CONFIG ERROR] cannot write networks.json (%v); the registry change is NOT persisted.\n", err)
		return
	}
	// Name what was NOT persisted, so "the file carries no token" can never be read as
	// "the server holds none".
	warnWithheldSecrets(tokens, ipfsKeys, headerSets, secretNetworks)
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
	// One envelope for every outcome. The two UIs that consume this route read
	// `res.contracts`, so a bare array (the old wallet-bound response) silently
	// rendered as "no contracts available". Empty is [] — never null.
	w.Header().Set("Content-Type", "application/json")
	if wallet == "" {
		// Contract eligibility is evaluated per wallet, so there is no global list
		// to return. Say so explicitly rather than implying none exist.
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":         true,
			"contracts":       make([]DynamicContract, 0),
			"wallet_required": true,
			"note":            "contract eligibility is evaluated per wallet — connect a wallet to list the contracts it qualifies for",
		})
		return
	}
	contracts := l.contractEngine.HandleGetAvailableContracts(l, wallet)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"contracts": nonNilSlice(contracts),
	})
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

// extractWalletFromRequest recovers the acting wallet from an HTTP request.
// Reconciled helper: checks the X-Wallet header, then ?wallet=/?address= query params.
func extractWalletFromRequest(r *http.Request) string {
	if w := r.Header.Get("X-Wallet"); w != "" {
		return w
	}
	if w := r.URL.Query().Get("wallet"); w != "" {
		return w
	}
	if w := r.URL.Query().Get("address"); w != "" {
		return w
	}
	return ""
}
