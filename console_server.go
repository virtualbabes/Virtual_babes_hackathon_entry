//go:build console

// Console-native build (Xbox / PlayStation / Switch devkit / Windows console / Steam Deck).
//
// Â§32.1 â€” "Console-powered AI runs as IN-GAME local logic, NOT an online AI except what it pushes
// through the world." This build compiles the FULL authoritative simulation (the same Lobby used
// by the Linux server) and runs it NATIVELY on console hardware. The console IS its own local
// authority: it binds to LOOPBACK only, so the sim never exposes a public surface â€” it only
// "pushes through the world" (the existing WS/HTTP routes) to a local renderer or to the hosted
// world for cross-save. No cloud model is used (constitutional: NO cloud models).
//
// Build:
//   GOOS=windows GOARCH=amd64 -tags console go build -o nft-seduction-console.exe .
//   GOOS=linux   GOARCH=amd64 -tags console go build -o nft-seduction-console .
// (see build_console.ps1 / build_console.sh)

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("[CONSOLE] No .env file found; relying on platform-injected env.")
	}
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("[CONSOLE] Initializing native console authority (loopback-only sim)...")

	// Kill any existing server process on this port
	killExistingServer("8090")

	lobby, err := newLobby()
	if err != nil {
		log.Fatalf("[CONSOLE FATAL] Failed to initialize Arena Lobby: %v", err)
	}

	// Start the main event loop (authoritative sim).
	go lobby.run()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("[CONSOLE] Shutdown signal received. Sealing state...")
		lobby.executeGracefulShutdown()
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(lobby, w, r)
	})
	// Mirror the primary API surface so a local renderer / companion app can drive the sim.
	registerConsoleRoutes(mux, lobby)

	// LOOPBACK ONLY â€” the console is a local authority, not a public server.
	addr := "127.0.0.1:8090"
	log.Printf("[CONSOLE] Native authority listening on %s (loopback only)", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("[CONSOLE FATAL] ListenAndServe: %v", err)
	}
}

// registerConsoleRoutes exposes the same handlers the hosted server uses, so console + cloud
// share one code path. Console simply does not forward to the public internet.
func registerConsoleRoutes(mux *http.ServeMux, lobby *Lobby) {
	// Entity events + stats
	mux.HandleFunc("/api/entity-events/regions", lobby.rateLimiter.WithRateLimit(lobby.handleEntityRegions, "wallet-default"))
	mux.HandleFunc("/api/entity-event/resolve", lobby.rateLimiter.WithRateLimit(lobby.handleEntityEventResolve, "economy-tight"))
	mux.HandleFunc("/api/entity-event/host", lobby.rateLimiter.WithRateLimit(lobby.handleEntityEventHost, "wallet-default"))
	mux.HandleFunc("/api/owner/combined-stats", lobby.rateLimiter.WithRateLimit(lobby.handleCombineOwnerStats, "wallet-default"))
	mux.HandleFunc("/api/orphan/status", lobby.rateLimiter.WithRateLimit(lobby.handleOrphanStatus, "wallet-default"))
	// THE NOTE VOCABULARY (note_vocabulary.go): read-only, so a client learns the
	// prefixes it may write instead of re-declaring them.
	mux.HandleFunc("/api/notes/vocabulary", lobby.rateLimiter.WithRateLimit(handleNoteVocabulary, "standard"))

	// ---- PARITY BLOCK 2 (2026-09-20): the routes registered with an INLINE CLOSURE, copied as WHOLE BLOCKS ----
	// A strict single-line extractor saw only 250 of server_main.go 326 registrations; the other 76 use a
	// multi-line closure, which is why the named families were absent from the first parity block. These 56
	// are copied VERBATIM (path, closure body, rate-limit tier) by script, never retyped. Before copying,
	// every body was scanned for symbols declared ONLY in server_main.go -- measured to be exactly htmlEsc
	// and main, so REFUSED=0; a body referencing either would have been refused rather than patched.
	mux.HandleFunc("/api/faction/shop/", func(next http.HandlerFunc) http.HandlerFunc {
		return lobby.rateLimiter.WithRateLimit(next, "core-economy")
	}(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/buy") {
			lobby.HandleBuyFactionItem(w, r)
		} else {
			lobby.HandleGetFactionShop(w, r)
		}
	}))

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

	mux.HandleFunc("/api/underworld/contracts", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		lobby.blackMarketService.HandleGetUnderworldContracts(lobby, w, r)
	}, "underworld"))

	mux.HandleFunc("/api/tournament/register", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		lobby.tournamentService.HandleTournamentRegister(lobby, w, r)
	}, "standard"))

	mux.HandleFunc("/api/tournament/history", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		lobby.tournamentService.HandleTournamentHistory(lobby, w, r)
	}, "wallet-default"))

	mux.HandleFunc("/api/courthouse/reset", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		lobby.courthouseService.HandleCourthouseReset(lobby, w, r)
	}, "wallet-default"))

	mux.HandleFunc("/api/black-market/buy", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		lobby.blackMarketService.HandleBuyBlackMarket(lobby, w, r)
	}, "economy-tight"))

	mux.HandleFunc("/api/counterfeit/generate", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			lobby.counterfeitService.HandleGenerateCounterfeit(lobby, w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "economy-tight"))

	mux.HandleFunc("/api/counterfeit/detect", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			lobby.counterfeitService.HandleDetectCounterfeit(lobby, w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "economy-tight"))

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

	mux.HandleFunc("/api/black-market/buy-stolen", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		lobby.blackMarketService.HandleBuyFencedGood(lobby, w, r)
	}, "economy-tight"))

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
				ProductID     string   `json:"product_id"`
				Name          string   `json:"name"`
				Description   string   `json:"description"`
				Category      string   `json:"category"`
				PriceMicroVBV uint64   `json:"price_micro_vbv"`
				Tags          []string `json:"tags,omitempty"`
				DLCLinks      []string `json:"dlc_links,omitempty"`
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

	mux.HandleFunc("/api/creator/store/profile/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		walletAddr := ""
		if len(parts) >= 5 {
			walletAddr = parts[4]
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
	})
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
				Rating    uint64 `json:"rating"`
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
	})
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
	})

	mux.HandleFunc("/api/ai/citizens/spawn", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			wallet := extractWalletFromRequest(r)
			if wallet == "" {
				http.Error(w, "wallet required", http.StatusBadRequest)
				return
			}
			citizen, err := lobby.aiEngine.SpawnAI(lobby, wallet, "", "")
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
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": stats})
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
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": citizens})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "standard"))

	mux.HandleFunc("/api/ai/citizens/adopt", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var req struct {
				Wallet  string `json:"wallet"`
				Citizen string `json:"citizen_wallet"`
				Tier    int    `json:"attachment_tier"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Wallet == "" || req.Citizen == "" {
				http.Error(w, "wallet + citizen_wallet required", http.StatusBadRequest)
				return
			}
			c, ok := lobby.aiEngine.GetCitizen(req.Citizen)
			if !ok {
				http.Error(w, "citizen not found", http.StatusNotFound)
				return
			}
			c.AttachOwnership(req.Wallet, req.Tier)
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": c})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "economy-tight"))

	mux.HandleFunc("/api/ai/citizens/release", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var req struct {
				Citizen string `json:"citizen_wallet"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Citizen == "" {
				http.Error(w, "citizen_wallet required", http.StatusBadRequest)
				return
			}
			c, ok := lobby.aiEngine.GetCitizen(req.Citizen)
			if !ok {
				http.Error(w, "citizen not found", http.StatusNotFound)
				return
			}
			c.DetachOwnership()
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": c})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "economy-tight"))

	mux.HandleFunc("/api/ai/citizens/challenge", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var req struct {
				Citizen    string `json:"citizen_wallet"`
				Challenger string `json:"challenger_wallet"`
				Bond       uint64 `json:"bond_micro"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Citizen == "" || req.Challenger == "" {
				http.Error(w, "citizen_wallet + challenger_wallet + bond_micro required", http.StatusBadRequest)
				return
			}
			lobby.mutex.Lock()
			challengerBal := lobby.playerBalances[strings.ToLower(req.Challenger)]
			if challengerBal < req.Bond {
				lobby.mutex.Unlock()
				http.Error(w, "challenger has insufficient balance for bond", http.StatusPaymentRequired)
				return
			}
			lobby.playerBalances[strings.ToLower(req.Challenger)] -= req.Bond
			lobby.mutex.Unlock()
			if lobby.tokenSinkRouter != nil {
				_ = lobby.tokenSinkRouter.RouteCriminalTax("AI_CHALLENGE_BOND_STAKE", req.Bond, RevenueSplitMatrix{FaucetShare: 1.0, ClubShare: 0.0, GovernanceShare: 0.0}, 0, "")
			}
			c, ok := lobby.aiEngine.GetCitizen(req.Citizen)
			if !ok {
				lobby.mutex.Lock()
				lobby.playerBalances[strings.ToLower(req.Challenger)] += req.Bond
				lobby.mutex.Unlock()
				http.Error(w, "citizen not found", http.StatusNotFound)
				return
			}
			c.ChallengerWallet = strings.ToLower(req.Challenger)
			c.ChallengeBond = req.Bond
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": c})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "economy-tight"))

	mux.HandleFunc("/api/treasure/spawn", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var req struct {
				Region string `json:"region"`
				// Coordinates must carry ONE key each: a grouped `X, Y, Z float64 `json:"x,y,z"``
				// gave all three fields the same key, so the payload could never populate them.
				X           float64 `json:"x"`
				Y           float64 `json:"y"`
				Z           float64 `json:"z"`
				RewardMicro uint64  `json:"reward_micro"`
				RewardKind  string  `json:"reward_kind"`
				RewardRef   string  `json:"reward_ref"`
				BondMicro   uint64  `json:"bond_micro"`
				Hidden      bool    `json:"hidden"`
				TTLSeconds  int64   `json:"ttl_seconds"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid body", http.StatusBadRequest)
				return
			}
			wallet := extractWalletFromRequest(r)
			c := lobby.SpawnTreasureCache(req.Region, req.X, req.Y, req.Z, req.RewardMicro, req.RewardKind, req.RewardRef, req.BondMicro, req.Hidden, wallet, time.Duration(req.TTLSeconds)*time.Second)
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": c})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "economy-tight"))

	mux.HandleFunc("/api/treasure/claim", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var req struct {
				CacheID string `json:"cache_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CacheID == "" {
				http.Error(w, "cache_id required", http.StatusBadRequest)
				return
			}
			wallet := lobby.getWalletFromRequest(r)
			payout, err := lobby.ClaimTreasureCache(req.CacheID, wallet)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "payout_micro": payout})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "standard"))

	mux.HandleFunc("/api/events/create", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var req struct {
				Type        string `json:"type"`
				Title       string `json:"title"`
				Description string `json:"description"`
				Region      string `json:"region"`
				RewardMicro uint64 `json:"reward_micro"`
				EntryMicro  uint64 `json:"entry_micro"`
				BondedNFT   string `json:"bonded_nft"`
				RoyaltyBps  uint64 `json:"royalty_bps"`
				TTLSeconds  int64  `json:"ttl_seconds"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Type == "" || req.Title == "" {
				http.Error(w, "type + title required", http.StatusBadRequest)
				return
			}
			wallet := lobby.getWalletFromRequest(r)
			e, err := lobby.CreateUserEvent(UserEventType(req.Type), req.Title, req.Description, req.Region, req.RewardMicro, req.EntryMicro, req.BondedNFT, wallet, req.RoyaltyBps, time.Duration(req.TTLSeconds)*time.Second)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": e})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "economy-tight"))

	mux.HandleFunc("/api/events/enter", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var req struct {
				EventID string `json:"event_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.EventID == "" {
				http.Error(w, "event_id required", http.StatusBadRequest)
				return
			}
			wallet := lobby.getWalletFromRequest(r)
			if err := lobby.EnterUserEvent(req.EventID, wallet); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "standard"))

	mux.HandleFunc("/api/regions", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": lobby.GetRegionViews()})
	}, "standard"))

	mux.HandleFunc("/api/rivalry/recompute", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		lobby.RecomputeAllSignatures()
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "count": len(lobby.rivalryEngine.Signatures)})
	}, "admin-tight"))

	mux.HandleFunc("/api/rivalry/detect", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		created := lobby.DetectRivalries()
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "created": len(created), "data": created})
	}, "admin-tight"))

	mux.HandleFunc("/api/rivalry/list", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		lobby.rivalryEngine.Mu.RLock()
		rivalries := make([]*RegionRivalry, 0, len(lobby.rivalryEngine.Rivalries))
		for _, rv := range lobby.rivalryEngine.Rivalries {
			rivalries = append(rivalries, rv)
		}
		lobby.rivalryEngine.Mu.RUnlock()
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": rivalries})
	}, "standard"))

	mux.HandleFunc("/api/rivalry/resolve", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var req struct {
				RivalryID string `json:"rivalry_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RivalryID == "" {
				http.Error(w, "rivalry_id required", http.StatusBadRequest)
				return
			}
			rv, err := lobby.ResolveRivalry(req.RivalryID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": rv})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "admin-tight"))

	mux.HandleFunc("/api/pets/spawn", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Name   string `json:"name"`
			Traits uint64 `json:"traits"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		wallet := extractWalletFromRequest(r)
		p, err := lobby.SpawnPet(wallet, req.Name, req.Traits)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": p})
	}, "economy-tight"))

	mux.HandleFunc("/api/pets/breed", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// NOTE: this handler deliberately reads NO fee from the body. The breeding price is
		// server-authoritative (PetBreedFeeMicro) — a client must never price its own upgrade.
		var req struct {
			Name   string `json:"name"`
			SireID string `json:"sire_id"`
			DamID  string `json:"dam_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		wallet := extractWalletFromRequest(r)
		off, err := lobby.BreedPet(wallet, req.Name, req.SireID, req.DamID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":   true,
			"data":      off,
			"fee_micro": PetBreedFeeMicro,
		})
	}, "economy-tight"))

	mux.HandleFunc("/api/pets", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		wallet := extractWalletFromRequest(r)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    lobby.GetPets(wallet),
			// §26.4.2 purchase + progression calibration, served so no client re-declares a price.
			"spawn_fee_micro":  PetSpawnFeeMicro,
			"breed_fee_micro":  PetBreedFeeMicro,
			"groom_base_micro": PetGroomBaseMicro,
			"groom_max_level":  PetGroomMaxLevel,
			"groom_stat_gain":  PetGroomStatGain,
			"groom_axes":       PetGroomTableView(),
			"maturity_ms":      30 * 24 * 60 * 60 * 1000,
		})
	}, "standard"))

	mux.HandleFunc("/api/pets/groom", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			PetID string `json:"pet_id"`
			Focus string `json:"focus"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		wallet := extractWalletFromRequest(r)
		p, fee, applied, err := lobby.GroomPet(wallet, req.PetID, req.Focus)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":   true,
			"data":      p,
			"fee_micro": fee,
			"stat_gain": applied,
			"focus":     strings.ToUpper(strings.TrimSpace(req.Focus)),
			"note":      "fee routed to the faucet sink (§26.4.2 Industrial Loop)",
		})
	}, "economy-tight"))

	mux.HandleFunc("/api/vehicles/spawn", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// NOTE: `min_level` is NOT read. The vehicle class owns the Level gate (§24.5) and the
		// price (§25.6.1), so a client can neither choose its own gate nor its own fee.
		var req struct {
			Name string `json:"name"`
			Kind string `json:"kind"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		wallet := extractWalletFromRequest(r)
		v, err := lobby.SpawnVehicle(wallet, req.Name, req.Kind, 0)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": v})
	}, "economy-tight"))

	mux.HandleFunc("/api/vehicles", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		wallet := extractWalletFromRequest(r)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    lobby.GetVehicles(wallet),
			// §25.6.1 calibration, served so the client never re-declares it.
			"parts":           VehiclePartTableView(),
			"part_max_level":  VehiclePartMaxLevel,
			"base_cost_micro": VehicleUpgradeBaseMicro,
			"stat_gain":       VehicleStatGain,
			"break_in_ms":     VehicleBreakInMs,
			// §25.6.1 purchase calibration: the class decides price, Level gate and stat floor.
			"spawn_fees": VehicleKindTableView(),
		})
	}, "standard"))

	mux.HandleFunc("/api/vehicles/upgrade", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			VehicleID string `json:"vehicle_id"`
			Part      string `json:"part"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		wallet := extractWalletFromRequest(r)
		v, fee, applied, err := lobby.UpgradeVehicle(wallet, req.VehicleID, req.Part)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":   true,
			"data":      v,
			"fee_micro": fee,
			"stat_gain": applied,
			"part":      strings.ToUpper(strings.TrimSpace(req.Part)),
			"note":      "fee routed to the faucet sink (§25.6.1 Industrial Loop)",
		})
	}, "economy-tight"))

	mux.HandleFunc("/api/vehicles/deploy", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			VehicleID string `json:"vehicle_id"`
			Region    string `json:"region"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		wallet := extractWalletFromRequest(r)
		v, err := lobby.DeployVehicle(wallet, req.VehicleID, req.Region)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": v})
	}, "wallet-default"))

	mux.HandleFunc("/api/world-content/create", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Kind string `json:"kind"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		wallet := extractWalletFromRequest(r)
		c, err := lobby.CreateWorldContent(wallet, req.Kind)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": c})
	}, "economy-tight"))

	mux.HandleFunc("/api/world-content/deploy", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			ContentID string `json:"content_id"`
			Region    string `json:"region"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := lobby.DeployWorldContent(req.ContentID, req.Region); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}, "economy-tight"))

	mux.HandleFunc("/api/world-content", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		region := r.URL.Query().Get("region")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": lobby.GetWorldContent(region)})
	}, "standard"))

	mux.HandleFunc("/api/ai/citizens/free-agents", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			agents := lobby.aiEngine.GetFreeAgents()
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": agents})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "standard"))

	mux.HandleFunc("/api/items/archetypes", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": GetItemArchetypes()})
	}, "standard"))

	mux.HandleFunc("/api/items/registry", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": lobby.itemRegistry.GetRegistry()})
	}, "standard"))

	mux.HandleFunc("/api/items/collection", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		wallet := r.URL.Query().Get("wallet")
		if wallet == "" {
			http.Error(w, "wallet required", http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": lobby.itemRegistry.GetCollection(wallet)})
	}, "standard"))

	mux.HandleFunc("/api/items/build", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var req BuildItemRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			item, err := lobby.itemRegistry.BuildItem(lobby, req)
			if err != nil {
				http.Error(w, fmt.Sprintf("build failed: %v", err), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": item})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "economy-tight"))

	mux.HandleFunc("/api/items/bind-nft", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var req struct {
				ItemID    string `json:"item_id"`
				BondedNFT string `json:"bonded_nft"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			item, err := lobby.itemRegistry.BindNFT(req.ItemID, req.BondedNFT)
			if err != nil {
				http.Error(w, fmt.Sprintf("bind failed: %v", err), http.StatusBadRequest)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": item})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "economy-tight"))

	// ---- PARITY BLOCK (2026-09-20): the game-facing routes this loopback authority was missing ---- 	// Every line below is COPIED VERBATIM from server_main.go -- path AND handler expression AND rate-limit 	// tier -- by script, never retyped, so the two tables cannot drift in spelling or in gate. DELIBERATELY 	// NOT mirrored, and therefore still baselined by the parity gate: the OPERATOR family (the admin doors), 	// the served HTML roots and the diagnostics page. A console is its own local authority bound to loopback; 	// it serves the GAME, not the hosted operator console, and mirroring those would claim a parity that is 	// not wanted. Path literals are absent from these comments on purpose: the parity gate raw-scans comments.
	mux.HandleFunc("/api/achievement/unlock", lobby.rateLimiter.WithRateLimit(lobby.handleUnlockAchievement, "achievement"))
	mux.HandleFunc("/api/achievements", lobby.rateLimiter.WithRateLimit(lobby.handleGetAchievements, "achievement"))
	mux.HandleFunc("/api/achievement-stats", lobby.rateLimiter.WithRateLimit(lobby.handleGetAchievementStats, "achievement"))
	mux.HandleFunc("/api/ai/citizens/adopt-pet", lobby.rateLimiter.WithRateLimit(lobby.handleAdoptPetAI, "wallet-default"))
	mux.HandleFunc("/api/ai/citizens/breed", lobby.rateLimiter.WithRateLimit(lobby.handleBreedAI, "wallet-default"))
	mux.HandleFunc("/api/ai/citizens/marry", lobby.rateLimiter.WithRateLimit(lobby.handleMarryAI, "wallet-default"))
	mux.HandleFunc("/api/ai/citizens/progress", lobby.rateLimiter.WithRateLimit(lobby.handleAIProgression, "wallet-default"))
	mux.HandleFunc("/api/bounty/active", lobby.rateLimiter.WithRateLimit(lobby.HandleGetBountyActive, "underworld"))
	// ---- CHAIN-RAIL EXCLUSION (operator ruling 2026-09-20): crypto is PC-side, the console mirrors VIRTUAL ----
	// Five routes were REMOVED here after every handler body was CENSUSED (from its `func` line to the next) for
	// on-chain primitives. Removed because the body SIGNS or SWAPS: the onboarding bridge (5 chain calls),
	// the redemption gateway (2), loan repayment (3) -- and with it LOAN TAKING, because a loan that cannot be
	// repaid on this device is a trap -- plus the faucet claim, which moves nothing and answers success.
	// KEPT and classified VIRTUAL (0 chain calls, measured): shop purchase, contract assignment, and the two
	// READS (loan list, bridge transactions). Path literals are absent from this comment on purpose: the parity
	// gate raw-scans comments and would read a quoted path as a registration.
	mux.HandleFunc("/api/bridge/txs", lobby.rateLimiter.WithRateLimit(lobby.handleBridgeTxs, "wallet-default"))
	mux.HandleFunc("/api/card-details", lobby.rateLimiter.WithRateLimit(lobby.handleGetCardDetails, "standard"))
	mux.HandleFunc("/api/card-stats", lobby.rateLimiter.WithRateLimit(lobby.handleCardStats, "standard"))
	mux.HandleFunc("/api/career/progress", lobby.rateLimiter.WithRateLimit(lobby.HandleGetCareerProgress, "core-economy"))
	mux.HandleFunc("/api/church/members", lobby.rateLimiter.WithRateLimit(lobby.HandleGetChurchMembers, "wallet-default"))
	mux.HandleFunc("/api/clubs", lobby.rateLimiter.WithRateLimit(lobby.HandleGetClubs, "wallet-default"))
	mux.HandleFunc("/api/contracts/assign", lobby.rateLimiter.WithRateLimit(lobby.handleAssignContract, "economy-tight"))
	mux.HandleFunc("/api/contracts/list", lobby.rateLimiter.WithRateLimit(lobby.handleGetAvailableContracts, "underworld"))
	mux.HandleFunc("/api/creator/store/creator", lobby.rateLimiter.WithRateLimit(lobby.HandleGetCreatorStore, "wallet-default"))
	mux.HandleFunc("/api/criminality/cyber-intercept", lobby.rateLimiter.WithRateLimit(lobby.handleCyberInterceptWrapper, "underworld"))
	mux.HandleFunc("/api/envoi-name", lobby.rateLimiter.WithRateLimit(lobby.handleEnvoiName, "wallet-default"))
	mux.HandleFunc("/api/faucet/status", lobby.rateLimiter.WithRateLimit(lobby.handleFaucetStatus, "wallet-default"))
	mux.HandleFunc("/api/faucet/vault-balance", lobby.rateLimiter.WithRateLimit(lobby.handleFaucetVaultBalance, "wallet-default"))
	mux.HandleFunc("/api/garden", lobby.rateLimiter.WithRateLimit(lobby.HandleGetGarden, "wallet-default"))
	mux.HandleFunc("/api/garden/add", lobby.rateLimiter.WithRateLimit(lobby.HandleAddGardenElement, "economy-tight"))
	mux.HandleFunc("/api/garden/meditate", lobby.rateLimiter.WithRateLimit(lobby.HandleMeditate, "wallet-default"))
	mux.HandleFunc("/api/governance/close", lobby.rateLimiter.WithRateLimit(lobby.handleGovernanceClose, "wallet-default"))
	mux.HandleFunc("/api/identity/link", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) { lobby.identityBridge.HandleLink(lobby, w, r) }, "wallet-default"))
	mux.HandleFunc("/api/identity/record", lobby.rateLimiter.WithRateLimit(lobby.handleIdentityRecord, "wallet-default"))
	mux.HandleFunc("/api/identity/resolve", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) { lobby.identityBridge.HandleResolve(lobby, w, r) }, "wallet-default"))
	mux.HandleFunc("/api/identity/snapshot", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) { lobby.identityBridge.HandleSnapshot(lobby, w, r) }, "wallet-default"))
	mux.HandleFunc("/api/identity/unlink", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) { lobby.identityBridge.HandleUnlink(lobby, w, r) }, "wallet-default"))
	mux.HandleFunc("/api/industrial-loop/record", lobby.rateLimiter.WithRateLimit(lobby.handleIndustrialLoopRecord, "wallet-default"))
	mux.HandleFunc("/api/justice/award-card", lobby.rateLimiter.WithRateLimit(lobby.handleAwardJusticeCard, "economy-tight"))
	mux.HandleFunc("/api/justice/bounty-board", lobby.rateLimiter.WithRateLimit(lobby.HandleGetJusticeDashboard, "underworld"))
	mux.HandleFunc("/api/justice/capture-bounty", lobby.rateLimiter.WithRateLimit(lobby.HandleCaptureBounty, "underworld"))
	mux.HandleFunc("/api/justice/dashboard", lobby.rateLimiter.WithRateLimit(lobby.HandleGetJusticeDashboard, "underworld"))
	mux.HandleFunc("/api/justice/missions", lobby.rateLimiter.WithRateLimit(lobby.HandleGetJusticeMissionsHTTP, "underworld"))
	mux.HandleFunc("/api/justice/missions/accept", lobby.rateLimiter.WithRateLimit(lobby.HandleAcceptJusticeMissionHTTP, "underworld"))
	mux.HandleFunc("/api/justice/use-rep-shield", lobby.rateLimiter.WithRateLimit(lobby.handleApplyRepShield, "economy-tight"))
	mux.HandleFunc("/api/justice/use-truth-serum", lobby.rateLimiter.WithRateLimit(lobby.handleUseTruthSerum, "economy-tight"))
	mux.HandleFunc("/api/leaderboard", lobby.rateLimiter.WithRateLimit(lobby.handleLeaderboard, "core-economy"))
	mux.HandleFunc("/api/loans", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) { lobby.loanService.HandleGetLoans(lobby, w, r) }, "standard"))
	mux.HandleFunc("/api/local-model/promote", lobby.rateLimiter.WithRateLimit(lobby.handlePromoteLocalModel, "wallet-default"))
	mux.HandleFunc("/api/market/weather", lobby.rateLimiter.WithRateLimit(lobby.handleMarketWeather, "wallet-default"))
	mux.HandleFunc("/api/match/wager", lobby.rateLimiter.WithRateLimit(lobby.handleSpectatorWager, "economy-tight"))
	mux.HandleFunc("/api/moods", lobby.rateLimiter.WithRateLimit(lobby.HandleGetMoods, "wallet-default"))
	mux.HandleFunc("/api/moods/adjust", lobby.rateLimiter.WithRateLimit(lobby.HandleAdjustMood, "wallet-default"))
	mux.HandleFunc("/api/orphan/adopt", lobby.rateLimiter.WithRateLimit(lobby.handleAdoptEntity, "wallet-default"))
	mux.HandleFunc("/api/orphan/reclaim", lobby.rateLimiter.WithRateLimit(lobby.handleReclaimEntity, "wallet-default"))
	mux.HandleFunc("/api/player/profile", lobby.rateLimiter.WithRateLimit(lobby.HandleGetPlayerProfile, "wallet-default"))
	mux.HandleFunc("/api/player/progression", lobby.rateLimiter.WithRateLimit(lobby.handlePlayerProgression, "wallet-default"))
	mux.HandleFunc("/api/player/tokens", lobby.rateLimiter.WithRateLimit(lobby.handleGetPlayerTokens, "wallet-default"))
	mux.HandleFunc("/api/players/constellation", lobby.rateLimiter.WithRateLimit(lobby.handleGetPlayersConstellation, "default"))
	mux.HandleFunc("/api/rivalry/action", lobby.rateLimiter.WithRateLimit(lobby.HandleRivalryAction, "standard"))
	mux.HandleFunc("/api/rivalry/factions", lobby.rateLimiter.WithRateLimit(lobby.handleRivalryFactions, "wallet-default"))
	mux.HandleFunc("/api/rivalry/join", lobby.rateLimiter.WithRateLimit(lobby.handleRivalryJoin, "wallet-default"))
	mux.HandleFunc("/api/rivalry/request", lobby.rateLimiter.WithRateLimit(lobby.HandleRivalryRequest, "standard"))
	mux.HandleFunc("/api/rivalry/state", lobby.rateLimiter.WithRateLimit(lobby.HandleGetRivalryState, "standard"))
	mux.HandleFunc("/api/rivalry/world-dynamics", lobby.rateLimiter.WithRateLimit(lobby.handleWorldDynamics, "wallet-default"))
	mux.HandleFunc("/api/rumors", lobby.rateLimiter.WithRateLimit(lobby.HandleGetRumors, "wallet-default"))
	mux.HandleFunc("/api/season/history", lobby.rateLimiter.WithRateLimit(lobby.handleSeasonHistory, "wallet-default"))
	mux.HandleFunc("/api/shop/purchase", lobby.rateLimiter.WithRateLimit(lobby.HandleShopPurchase, "economy-tight"))
	mux.HandleFunc("/api/tea/brew", lobby.rateLimiter.WithRateLimit(lobby.HandleBrewTea, "economy-tight"))
	mux.HandleFunc("/api/tea/recipes", lobby.rateLimiter.WithRateLimit(lobby.HandleGetTeaRecipes, "wallet-default"))
	mux.HandleFunc("/api/titles", lobby.rateLimiter.WithRateLimit(lobby.HandleGetCardTitles, "wallet-default"))
	mux.HandleFunc("/api/titles/equip", lobby.rateLimiter.WithRateLimit(lobby.HandleEquipCardTitle, "wallet-default"))
	mux.HandleFunc("/api/underworld/heists", lobby.rateLimiter.WithRateLimit(lobby.handleUnderworldHeists, "underworld"))
	mux.HandleFunc("/api/underworld/kidnaps", lobby.rateLimiter.WithRateLimit(lobby.handleUnderworldKidnaps, "underworld"))
	mux.HandleFunc("/api/wagers", lobby.rateLimiter.WithRateLimit(lobby.HandleGetWagers, "wallet-default"))
	mux.HandleFunc("/api/wagers/resolve", lobby.rateLimiter.WithRateLimit(lobby.HandleResolveWager, "economy-tight"))
	// §23.5 Bonded assets + ECOSPHERE BRANDING (parity with server_main.go). A bonded asset
	// may be worn by any entity the wallet owns; cards are excluded structurally.
	mux.HandleFunc("/api/assets/mint", lobby.rateLimiter.WithRateLimit(lobby.handleMintBondedAsset, "economy-tight"))
	mux.HandleFunc("/api/assets", lobby.rateLimiter.WithRateLimit(lobby.handleListBondedAssets, "wallet-default"))
	mux.HandleFunc("/api/assets/burn", lobby.rateLimiter.WithRateLimit(lobby.handleBurnBondedAsset, "economy-tight"))
	mux.HandleFunc("/api/assets/modify", lobby.rateLimiter.WithRateLimit(lobby.handleModifyBondedAsset, "economy-tight"))
	mux.HandleFunc("/api/assets/transfer", lobby.rateLimiter.WithRateLimit(lobby.handleTransferBondedAsset, "economy-tight"))
	mux.HandleFunc("/api/assets/bind", lobby.rateLimiter.WithRateLimit(lobby.handleBindBondedAssetTarget, "wallet-default"))
	mux.HandleFunc("/api/assets/unbind", lobby.rateLimiter.WithRateLimit(lobby.handleUnbindBondedAssetTarget, "wallet-default"))
	mux.HandleFunc("/api/assets/targets", lobby.rateLimiter.WithRateLimit(lobby.handleBondedTargets, "wallet-default"))
	mux.HandleFunc("/api/assets/branding", lobby.rateLimiter.WithRateLimit(lobby.handleBondedBranding, "wallet-default"))
	// §23.5.5 viewer-scoped card display (parity with server_main.go). A VIEW preference keyed by
	// the viewer's own wallet; it never writes a branding binding, so cards stay excluded (§10.1).
	mux.HandleFunc("/api/assets/card-view", lobby.rateLimiter.WithRateLimit(lobby.handleCardView, "wallet-default"))
	mux.HandleFunc("/api/assets/card-view/clear", lobby.rateLimiter.WithRateLimit(lobby.handleClearCardView, "wallet-default"))
	mux.HandleFunc("/api/assets/catalogue", lobby.rateLimiter.WithRateLimit(lobby.handlePlaceholderCatalogue, "wallet-default"))
	mux.HandleFunc("/api/assets/starter", lobby.rateLimiter.WithRateLimit(lobby.handleStarterAssets, "economy-tight"))
	mux.HandleFunc("/api/assets/purchase", lobby.rateLimiter.WithRateLimit(lobby.handleBuyPlaceholderSku, "economy-tight"))
	mux.HandleFunc("/api/assets/slide-theme", lobby.rateLimiter.WithRateLimit(lobby.handleSlideTheme, "wallet-default"))
	mux.HandleFunc("/api/assets/slide-theme/clear", lobby.rateLimiter.WithRateLimit(lobby.handleClearSlideTheme, "wallet-default"))
	mux.HandleFunc("/api/assets/ui-trees", lobby.rateLimiter.WithRateLimit(lobby.handleUiTreeArt, "wallet-default"))
	// §10.8 player-to-player bonded-asset market (parity with server_main.go).
	mux.HandleFunc("/api/assets/market", lobby.rateLimiter.WithRateLimit(lobby.handleBondedMarket, "wallet-default"))
	mux.HandleFunc("/api/assets/market/list", lobby.rateLimiter.WithRateLimit(lobby.handleBondedMarketList, "economy-tight"))
	mux.HandleFunc("/api/assets/market/cancel", lobby.rateLimiter.WithRateLimit(lobby.handleBondedMarketCancel, "wallet-default"))
	mux.HandleFunc("/api/assets/market/buy", lobby.rateLimiter.WithRateLimit(lobby.handleBondedMarketBuy, "economy-tight"))
	mux.HandleFunc("/api/theme/bind", lobby.rateLimiter.WithRateLimit(lobby.handleBindThemeAsset, "wallet-default"))
	mux.HandleFunc("/api/theme/lock", lobby.rateLimiter.WithRateLimit(lobby.handleLockThemeAsset, "wallet-default"))
	mux.HandleFunc("/api/theme/vector", lobby.rateLimiter.WithRateLimit(lobby.handleThemeVector, "wallet-default"))

	// Faith + religion
	mux.HandleFunc("/api/faith/coherence", lobby.rateLimiter.WithRateLimit(lobby.handleFaithCoherence, "wallet-default"))
	mux.HandleFunc("/api/faith/war-gambit", lobby.rateLimiter.WithRateLimit(lobby.handleFaithWarGambit, "wallet-default"))
	mux.HandleFunc("/api/faith/religions", lobby.rateLimiter.WithRateLimit(lobby.handleFaithReligions, "wallet-default"))
	mux.HandleFunc("/api/faith/religion/buy", lobby.rateLimiter.WithRateLimit(lobby.handleFaithReligionBuy, "wallet-default"))
	mux.HandleFunc("/api/faith/religion/buyout", lobby.rateLimiter.WithRateLimit(lobby.handleFaithReligionBuyout, "wallet-default"))
	mux.HandleFunc("/api/faith/religion/join", lobby.rateLimiter.WithRateLimit(lobby.handleFaithReligionJoin, "wallet-default"))
	mux.HandleFunc("/api/faith/religion/ritual", lobby.rateLimiter.WithRateLimit(lobby.handleFaithReligionRitual, "wallet-default"))
	mux.HandleFunc("/api/faith/religion/rivalry", lobby.rateLimiter.WithRateLimit(lobby.handleFaithReligionRivalry, "wallet-default"))
	mux.HandleFunc("/api/faith/high-tier", lobby.rateLimiter.WithRateLimit(lobby.handleFaithHighTier, "wallet-default"))
	// (A fabricated faith-conversion route was REMOVED from this table: the handler it named exists in NO
	// .go file, and the primary server serves no such path — so the line could never compile, and
	// therefore never served a single request. See Problems.md and `npm run verify:routes-parity`. The
	// path string is deliberately NOT repeated in this comment: the parity gate's raw scan matches a path
	// inside a comment, so quoting it would resurrect the very console-only route this line removed.)

	// Local model + AI citizens
	mux.HandleFunc("/api/local-model/status", lobby.rateLimiter.WithRateLimit(lobby.handleLocalModelStatus, "wallet-default"))

	// Stat overlay
	mux.HandleFunc("/api/stat-overlay", lobby.rateLimiter.WithRateLimit(lobby.handleStatOverlayGet, "wallet-default"))
	mux.HandleFunc("/api/stat-overlay/region", lobby.rateLimiter.WithRateLimit(lobby.handleStatOverlayRegion, "wallet-default"))
	mux.HandleFunc("/api/stat-overlay/owner", lobby.rateLimiter.WithRateLimit(lobby.handleStatOverlayOwner, "wallet-default"))
	mux.HandleFunc("/api/stat-overlay/leaderboard", lobby.rateLimiter.WithRateLimit(lobby.handleStatOverlayLeaderboard, "wallet-default"))

	// Industrial loop
	mux.HandleFunc("/api/industrial-loop/metrics", lobby.rateLimiter.WithRateLimit(lobby.handleIndustrialLoopMetrics, "wallet-default"))
	mux.HandleFunc("/api/industrial-loop/health", lobby.rateLimiter.WithRateLimit(lobby.handleIndustrialLoopHealth, "wallet-default"))

	// Persistent identity
	mux.HandleFunc("/api/identity/profile", lobby.rateLimiter.WithRateLimit(lobby.handleIdentityProfile, "wallet-default"))
	mux.HandleFunc("/api/identity/events", lobby.rateLimiter.WithRateLimit(lobby.handleIdentityEvents, "wallet-default"))
	mux.HandleFunc("/api/identity/leaderboard", lobby.rateLimiter.WithRateLimit(lobby.handleIdentityLeaderboard, "wallet-default"))

	// Entity shares
	mux.HandleFunc("/api/shares/issue", lobby.rateLimiter.WithRateLimit(lobby.handleEntitySharesIssue, "wallet-default"))
	mux.HandleFunc("/api/shares/buy", lobby.rateLimiter.WithRateLimit(lobby.handleEntitySharesBuy, "wallet-default"))
	mux.HandleFunc("/api/shares/tokens", lobby.rateLimiter.WithRateLimit(lobby.handleEntitySharesTokens, "wallet-default"))
	mux.HandleFunc("/api/shares/holdings", lobby.rateLimiter.WithRateLimit(lobby.handleEntitySharesHoldings, "wallet-default"))

	// Infrastructure leasing
	mux.HandleFunc("/api/lease/create", lobby.rateLimiter.WithRateLimit(lobby.handleLeaseCreate, "wallet-default"))
	// Workstream C, READ-ONLY HALF: a console build may SEE the entitlements granted to its accounts and
	// the served contract that explains them, without transacting — the same ruling that lets it read loans
	// and bridge history while it cannot move value. The WRITER is primary-only, deliberately.
	mux.HandleFunc("/api/console/entitlements", lobby.rateLimiter.WithRateLimit(lobby.handleConsoleEntitlements, "wallet-default"))

	mux.HandleFunc("/api/lease/list", lobby.rateLimiter.WithRateLimit(lobby.handleLeaseList, "wallet-default"))
	mux.HandleFunc("/api/lease/available", lobby.rateLimiter.WithRateLimit(lobby.handleLeaseAvailable, "wallet-default"))

	// Governance
	mux.HandleFunc("/api/governance/weight", lobby.rateLimiter.WithRateLimit(lobby.handleGovernanceWeight, "wallet-default"))
	mux.HandleFunc("/api/governance/register", lobby.rateLimiter.WithRateLimit(lobby.handleGovernanceRegister, "wallet-default"))
	mux.HandleFunc("/api/governance/vote", lobby.rateLimiter.WithRateLimit(lobby.handleGovernanceVote, "wallet-default"))
	mux.HandleFunc("/api/governance/governor", lobby.rateLimiter.WithRateLimit(lobby.handleGovernanceGovernor, "wallet-default"))
	mux.HandleFunc("/api/governance/election", lobby.rateLimiter.WithRateLimit(lobby.handleGovernanceElection, "wallet-default"))
	mux.HandleFunc("/api/governance/leaderboard", lobby.rateLimiter.WithRateLimit(lobby.handleGovernanceLeaderboard, "wallet-default"))

	// World Dashboard data endpoints
	mux.HandleFunc("/api/rewards", lobby.rateLimiter.WithRateLimit(lobby.handleRewards, "wallet-default"))
	mux.HandleFunc("/api/dividends", lobby.rateLimiter.WithRateLimit(lobby.handleDividends, "wallet-default"))

	// Player association export (§3 READ-ONLY PORTFOLIO): the wallet associations the
	// engine holds on PlayerStats that no other read-only route exposes.
	mux.HandleFunc("/api/player/associations", lobby.rateLimiter.WithRateLimit(lobby.handlePlayerAssociations, "wallet-default"))

	// Bridge router
	mux.HandleFunc("/api/bridge/asset", lobby.rateLimiter.WithRateLimit(lobby.handleBridgeAsset, "wallet-default"))
	mux.HandleFunc("/api/bridge/confirm", lobby.rateLimiter.WithRateLimit(lobby.handleBridgeConfirm, "wallet-default"))
	mux.HandleFunc("/api/bridge/assets", lobby.rateLimiter.WithRateLimit(lobby.handleBridgeAssets, "wallet-default"))
	mux.HandleFunc("/api/bridge/summary", lobby.rateLimiter.WithRateLimit(lobby.handleBridgeSummary, "wallet-default"))

	// Creator economy
	mux.HandleFunc("/api/creator/dlc/create", lobby.rateLimiter.WithRateLimit(lobby.handleCreatorDLCCreate, "wallet-default"))
	mux.HandleFunc("/api/creator/dlc/purchase", lobby.rateLimiter.WithRateLimit(lobby.handleCreatorDLCPurchase, "wallet-default"))
	mux.HandleFunc("/api/creator/dlcs", lobby.rateLimiter.WithRateLimit(lobby.handleCreatorDLCs, "wallet-default"))
	mux.HandleFunc("/api/creator/royalties", lobby.rateLimiter.WithRateLimit(lobby.handleCreatorRoyalties, "wallet-default"))
	mux.HandleFunc("/api/creator/event/create", lobby.rateLimiter.WithRateLimit(lobby.handleCreatorEventCreate, "wallet-default"))
	mux.HandleFunc("/api/creator/events", lobby.rateLimiter.WithRateLimit(lobby.handleCreatorEvents, "wallet-default"))
	mux.HandleFunc("/api/creator/event/attend", lobby.rateLimiter.WithRateLimit(lobby.handleCreatorEventAttend, "wallet-default"))
	mux.HandleFunc("/api/creator/sub/create", lobby.rateLimiter.WithRateLimit(lobby.handleCreatorSubCreate, "wallet-default"))
	mux.HandleFunc("/api/creator/subs", lobby.rateLimiter.WithRateLimit(lobby.handleCreatorSubs, "wallet-default"))

	// Faith church
	mux.HandleFunc("/api/church/open", lobby.rateLimiter.WithRateLimit(lobby.handleChurchOpen, "wallet-default"))
	mux.HandleFunc("/api/church/get", lobby.rateLimiter.WithRateLimit(lobby.handleChurchGet, "wallet-default"))
	mux.HandleFunc("/api/church/owner", lobby.rateLimiter.WithRateLimit(lobby.handleChurchesByOwner, "wallet-default"))
	mux.HandleFunc("/api/church/region", lobby.rateLimiter.WithRateLimit(lobby.handleChurchesByRegion, "wallet-default"))
	mux.HandleFunc("/api/church/add-member", lobby.rateLimiter.WithRateLimit(lobby.handleChurchAddMember, "wallet-default"))
	mux.HandleFunc("/api/church/remove-member", lobby.rateLimiter.WithRateLimit(lobby.handleChurchRemoveMember, "wallet-default"))
	mux.HandleFunc("/api/church/add-item", lobby.rateLimiter.WithRateLimit(lobby.handleChurchAddItem, "wallet-default"))
	mux.HandleFunc("/api/church/items", lobby.rateLimiter.WithRateLimit(lobby.handleChurchItems, "wallet-default"))
	mux.HandleFunc("/api/church/ritual", lobby.rateLimiter.WithRateLimit(lobby.handleChurchRitual, "wallet-default"))
	mux.HandleFunc("/api/church/rituals", lobby.rateLimiter.WithRateLimit(lobby.handleChurchRituals, "wallet-default"))
	mux.HandleFunc("/api/church/leaderboard", lobby.rateLimiter.WithRateLimit(lobby.handleChurchLeaderboard, "wallet-default"))

	// Entity market + pet battle
	mux.HandleFunc("/api/entity/market/list", lobby.rateLimiter.WithRateLimit(lobby.handleEntityMarketList, "wallet-default"))
	mux.HandleFunc("/api/entity/market/create", lobby.rateLimiter.WithRateLimit(lobby.handleEntityMarketCreate, "wallet-default"))
	mux.HandleFunc("/api/entity/market/purchase", lobby.rateLimiter.WithRateLimit(lobby.handleEntityMarketPurchase, "wallet-default"))
	mux.HandleFunc("/api/match/active", lobby.rateLimiter.WithRateLimit(lobby.handleActiveMatches, "wallet-default"))
	mux.HandleFunc("/api/pet-battle/list", lobby.rateLimiter.WithRateLimit(lobby.handlePetBattleList, "wallet-default"))
	mux.HandleFunc("/api/pet-battle/challenge", lobby.rateLimiter.WithRateLimit(lobby.handlePetBattleChallenge, "wallet-default"))
	mux.HandleFunc("/api/pet-battle/resolve", lobby.rateLimiter.WithRateLimit(lobby.handlePetBattleResolve, "wallet-default"))

	// Children bots
	mux.HandleFunc("/api/children-bots", lobby.rateLimiter.WithRateLimit(lobby.handleChildrenBots, "wallet-default"))

	// Admin
	// ALIGNED TO THE PRIMARY SERVER'S OWN PATHS for these two capabilities. The old console-only spellings
	// had NO caller anywhere in Public/** while the primary paths are called by admin.js and
	// world_dashboard.js, so the console table now names the same door the clients already use. The old
	// spellings are deliberately not repeated here (the parity gate's raw scan matches paths in comments).
	mux.HandleFunc("/api/system-message", lobby.rateLimiter.WithRateLimit(lobby.handleSystemMessage, "wallet-default"))
	mux.HandleFunc("/api/maintenance-mode", lobby.rateLimiter.WithRateLimit(lobby.handleMaintenanceMode, "wallet-default"))
	// READ-ONLY registry authority view: redacted configs + which networks hold secret material
	// in the running process (registered here for parity with server_main.go).
	mux.HandleFunc("/api/admin/networks", lobby.rateLimiter.WithRateLimit(lobby.handleAdminNetworks, "wallet-default"))

	// Launchpad System
	mux.HandleFunc("/api/launch/create", lobby.rateLimiter.WithRateLimit(lobby.handleLaunchCreate, "wallet-default"))
	mux.HandleFunc("/api/launch/back", lobby.rateLimiter.WithRateLimit(lobby.handleLaunchBack, "wallet-default"))
	mux.HandleFunc("/api/launch/activate", lobby.rateLimiter.WithRateLimit(lobby.handleLaunchActivate, "wallet-default"))
	mux.HandleFunc("/api/launch/integrate", lobby.rateLimiter.WithRateLimit(lobby.handleLaunchIntegrate, "wallet-default"))
	mux.HandleFunc("/api/launches", lobby.rateLimiter.WithRateLimit(lobby.handleLaunches, "wallet-default"))
	mux.HandleFunc("/api/launch/get", lobby.rateLimiter.WithRateLimit(lobby.handleLaunchGet, "wallet-default"))
	mux.HandleFunc("/api/launch/creator", lobby.rateLimiter.WithRateLimit(lobby.handleLaunchesByCreator, "wallet-default"))

	// Advertising System
	mux.HandleFunc("/api/ad/create", lobby.rateLimiter.WithRateLimit(lobby.handleAdCreate, "wallet-default"))
	mux.HandleFunc("/api/ad/activate", lobby.rateLimiter.WithRateLimit(lobby.handleAdActivate, "wallet-default"))
	mux.HandleFunc("/api/ad/pause", lobby.rateLimiter.WithRateLimit(lobby.handleAdPause, "wallet-default"))
	mux.HandleFunc("/api/ad/impression", lobby.rateLimiter.WithRateLimit(lobby.handleAdImpression, "wallet-default"))
	mux.HandleFunc("/api/ad/click", lobby.rateLimiter.WithRateLimit(lobby.handleAdClick, "wallet-default"))
	mux.HandleFunc("/api/ads", lobby.rateLimiter.WithRateLimit(lobby.handleAds, "wallet-default"))
	mux.HandleFunc("/api/ads/advertiser", lobby.rateLimiter.WithRateLimit(lobby.handleAdsByAdvertiser, "wallet-default"))
	mux.HandleFunc("/api/ads/region", lobby.rateLimiter.WithRateLimit(lobby.handleAdsForRegion, "wallet-default"))
	mux.HandleFunc("/api/ad/stats", lobby.rateLimiter.WithRateLimit(lobby.handleAdStats, "wallet-default"))

	// Gaming OS System
	mux.HandleFunc("/api/os/modules", lobby.rateLimiter.WithRateLimit(lobby.handleOSModules, "wallet-default"))
	mux.HandleFunc("/api/os/module/register", lobby.rateLimiter.WithRateLimit(lobby.handleOSModuleRegister, "wallet-default"))
	mux.HandleFunc("/api/os/lease", lobby.rateLimiter.WithRateLimit(lobby.handleOSLease, "wallet-default"))
	mux.HandleFunc("/api/os/leases", lobby.rateLimiter.WithRateLimit(lobby.handleOSLeases, "wallet-default"))
	mux.HandleFunc("/api/os/summary", lobby.rateLimiter.WithRateLimit(lobby.handleOSSummary, "wallet-default"))

	// Compliance System
	mux.HandleFunc("/api/compliance/record", lobby.rateLimiter.WithRateLimit(lobby.handleComplianceRecord, "wallet-default"))
	mux.HandleFunc("/api/compliance/resolve", lobby.rateLimiter.WithRateLimit(lobby.handleComplianceResolve, "wallet-default"))
	mux.HandleFunc("/api/compliance/escalate", lobby.rateLimiter.WithRateLimit(lobby.handleComplianceEscalate, "wallet-default"))
	mux.HandleFunc("/api/compliance/records", lobby.rateLimiter.WithRateLimit(lobby.handleComplianceRecords, "wallet-default"))
	mux.HandleFunc("/api/compliance/wallet", lobby.rateLimiter.WithRateLimit(lobby.handleComplianceByWallet, "wallet-default"))
	mux.HandleFunc("/api/compliance/summary", lobby.rateLimiter.WithRateLimit(lobby.handleComplianceSummary, "wallet-default"))

	// Replay Engine
	mux.HandleFunc("/api/replay/state", lobby.rateLimiter.WithRateLimit(lobby.handleReplayState, "wallet-default"))
	mux.HandleFunc("/api/replay/latest", lobby.rateLimiter.WithRateLimit(lobby.handleReplayLatest, "wallet-default"))
	mux.HandleFunc("/api/replay/frames", lobby.rateLimiter.WithRateLimit(lobby.handleReplayFrames, "wallet-default"))
	mux.HandleFunc("/api/replay/player", lobby.rateLimiter.WithRateLimit(lobby.handleReplayPlayer, "wallet-default"))
	mux.HandleFunc("/api/replay/capture", lobby.rateLimiter.WithRateLimit(lobby.handleReplayCapture, "wallet-default"))
	mux.HandleFunc("/api/replay/start", lobby.rateLimiter.WithRateLimit(lobby.handleReplayStart, "wallet-default"))
	mux.HandleFunc("/api/replay/stop", lobby.rateLimiter.WithRateLimit(lobby.handleReplayStop, "wallet-default"))

	// ── CAREER PATH · CIVIL RANK · PROMOTION (career_path.go / career_path_handlers.go) ──
	// Parity with server_main.go. The path choice and the promotion are the only two things a
	// client may name; everything else on these routes is derived from engine state.
	mux.HandleFunc("/api/career/path", lobby.rateLimiter.WithRateLimit(lobby.handleCareerPath, "wallet-default"))
	mux.HandleFunc("/api/career/path/choose", lobby.rateLimiter.WithRateLimit(lobby.handleCareerPathChoose, "economy-tight"))
	mux.HandleFunc("/api/career/promote", lobby.rateLimiter.WithRateLimit(lobby.handleCareerPromote, "economy-tight"))
	// Parity with server_main.go: the employer may SEE and REQUEST a staff upgrade; the upgrade
	// itself stays with the staff member (`career_path.go` §7.5).
	mux.HandleFunc("/api/career/staff/request", lobby.rateLimiter.WithRateLimit(lobby.handleCareerStaffUpgradeRequest, "economy-tight"))
}
