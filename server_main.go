//go:build !js && !wasm && !console

// Default hosted-server entrypoint. Split out from server.go so the console build
// (console_server.go, tag `console`) can supply its own loopback-only main() while
// reusing all the helpers (newLobby, serveWs, getDataPath, extractWalletFromRequest, ...)
// that live in server.go.

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
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

// htmlEsc escapes a string for safe embedding in HTML (diagnostics page).
func htmlEsc(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;")
	return r.Replace(s)
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

	mux.HandleFunc("/api/reward", lobby.rateLimiter.WithRateLimit(lobby.handleReward, "economy-tight"))
	mux.HandleFunc("/api/match/wager", lobby.rateLimiter.WithRateLimit(lobby.handleSpectatorWager, "economy-tight"))
	mux.HandleFunc("/api/match/active", lobby.rateLimiter.WithRateLimit(lobby.handleActiveMatches, "economy-tight"))
	// THE NOTE VOCABULARY (note_vocabulary.go): read-only, so a client learns the
	// prefixes it may write instead of re-declaring them.
	mux.HandleFunc("/api/notes/vocabulary", lobby.rateLimiter.WithRateLimit(handleNoteVocabulary, "standard"))
	mux.HandleFunc("/api/loans/take", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) { lobby.loanService.HandleTakeLoan(lobby, w, r) }, "economy-tight"))
	mux.HandleFunc("/api/loans/repay", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) { lobby.loanService.HandleRepayLoan(lobby, w, r) }, "economy-tight"))
	mux.HandleFunc("/api/black-market/sell-tokens", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		lobby.blackMarketService.HandleSellMarketTokens(lobby, w, r)
	}, "economy-tight"))

	mux.HandleFunc("/api/leaderboard", lobby.rateLimiter.WithRateLimit(lobby.handleLeaderboard, "core-economy"))
	mux.HandleFunc("/api/career/progress", lobby.rateLimiter.WithRateLimit(lobby.HandleGetCareerProgress, "core-economy"))
	// ── CAREER PATH · CIVIL RANK · PROMOTION (career_path.go / career_path_handlers.go) ──
	// The path is chosen ONCE, when the player opens a region with 2+ territories. Everything these
	// doors return — the civil rank, the eligibility, whether a promotion is earned, whether a
	// career is demoted — is DERIVED from engine state, never accepted from the caller.
	mux.HandleFunc("/api/career/path", lobby.rateLimiter.WithRateLimit(lobby.handleCareerPath, "wallet-default"))
	mux.HandleFunc("/api/career/path/choose", lobby.rateLimiter.WithRateLimit(lobby.handleCareerPathChoose, "economy-tight"))
	mux.HandleFunc("/api/career/promote", lobby.rateLimiter.WithRateLimit(lobby.handleCareerPromote, "economy-tight"))
	// The EMPLOYER's half of the rule: an employer may SEE which staff can upgrade and may REQUEST
	// that they do — the upgrade itself stays with the staff member (`career_path.go` §7.5). Refuses
	// an upgrade the level cap has not opened, quoting the EMPLOYEE's own missing gate.
	mux.HandleFunc("/api/career/staff/request", lobby.rateLimiter.WithRateLimit(lobby.handleCareerStaffUpgradeRequest, "economy-tight"))
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

	mux.HandleFunc("/api/underworld/contracts", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		lobby.blackMarketService.HandleGetUnderworldContracts(lobby, w, r)
	}, "underworld"))
	mux.HandleFunc("/api/justice/dashboard", lobby.rateLimiter.WithRateLimit(lobby.HandleGetJusticeDashboard, "underworld"))
	mux.HandleFunc("/api/justice/use-truth-serum", lobby.rateLimiter.WithRateLimit(lobby.handleUseTruthSerum, "economy-tight"))
	mux.HandleFunc("/api/justice/capture-bounty", lobby.rateLimiter.WithRateLimit(lobby.HandleCaptureBounty, "underworld"))
	mux.HandleFunc("/api/justice/bounty-board", lobby.rateLimiter.WithRateLimit(lobby.HandleGetJusticeDashboard, "underworld"))
	mux.HandleFunc("/api/justice/award-card", lobby.rateLimiter.WithRateLimit(lobby.handleAwardJusticeCard, "economy-tight"))
	mux.HandleFunc("/api/justice/use-rep-shield", lobby.rateLimiter.WithRateLimit(lobby.handleApplyRepShield, "economy-tight"))
	mux.HandleFunc("/api/rewards", lobby.rateLimiter.WithRateLimit(lobby.handleRewards, "wallet-default"))
	mux.HandleFunc("/api/dividends", lobby.rateLimiter.WithRateLimit(lobby.handleDividends, "wallet-default"))
	mux.HandleFunc("/api/criminality/cyber-intercept", lobby.rateLimiter.WithRateLimit(lobby.handleCyberInterceptWrapper, "underworld"))
	mux.HandleFunc("/api/contracts/list", lobby.rateLimiter.WithRateLimit(lobby.handleGetAvailableContracts, "underworld"))
	mux.HandleFunc("/api/contracts/assign", lobby.rateLimiter.WithRateLimit(lobby.handleAssignContract, "economy-tight"))

	mux.HandleFunc("/api/identity/link", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) { lobby.identityBridge.HandleLink(lobby, w, r) }, "wallet-default"))
	mux.HandleFunc("/api/identity/unlink", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) { lobby.identityBridge.HandleUnlink(lobby, w, r) }, "wallet-default"))
	mux.HandleFunc("/api/identity/resolve", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) { lobby.identityBridge.HandleResolve(lobby, w, r) }, "wallet-default"))
	mux.HandleFunc("/api/identity/snapshot", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) { lobby.identityBridge.HandleSnapshot(lobby, w, r) }, "wallet-default"))

	mux.HandleFunc("/api/theme/vector", lobby.rateLimiter.WithRateLimit(lobby.handleThemeVector, "wallet-default"))
	mux.HandleFunc("/api/market/weather", lobby.rateLimiter.WithRateLimit(lobby.handleMarketWeather, "wallet-default"))
	mux.HandleFunc("/api/rivalry/world-dynamics", lobby.rateLimiter.WithRateLimit(lobby.handleWorldDynamics, "wallet-default"))
	mux.HandleFunc("/api/assets/mint", lobby.rateLimiter.WithRateLimit(lobby.handleMintBondedAsset, "economy-tight"))
	mux.HandleFunc("/api/assets", lobby.rateLimiter.WithRateLimit(lobby.handleListBondedAssets, "wallet-default"))
	mux.HandleFunc("/api/assets/burn", lobby.rateLimiter.WithRateLimit(lobby.handleBurnBondedAsset, "economy-tight"))
	mux.HandleFunc("/api/assets/modify", lobby.rateLimiter.WithRateLimit(lobby.handleModifyBondedAsset, "economy-tight"))
	mux.HandleFunc("/api/assets/transfer", lobby.rateLimiter.WithRateLimit(lobby.handleTransferBondedAsset, "economy-tight"))
	// §23.5 ECOSPHERE BRANDING: a bonded asset may be worn by any entity the wallet owns.
	// Cards are excluded structurally (bonded_branding.go) — these routes refuse a card kind.
	mux.HandleFunc("/api/assets/bind", lobby.rateLimiter.WithRateLimit(lobby.handleBindBondedAssetTarget, "wallet-default"))
	mux.HandleFunc("/api/assets/unbind", lobby.rateLimiter.WithRateLimit(lobby.handleUnbindBondedAssetTarget, "wallet-default"))
	mux.HandleFunc("/api/assets/targets", lobby.rateLimiter.WithRateLimit(lobby.handleBondedTargets, "wallet-default"))
	mux.HandleFunc("/api/assets/branding", lobby.rateLimiter.WithRateLimit(lobby.handleBondedBranding, "wallet-default"))
	// §23.5.5 viewer-scoped card display: "player 1 can change how they see player 2's cards".
	// A VIEW preference keyed by the viewer's own wallet — it never writes a branding binding,
	// so cards stay excluded (§10.1). GET reads, POST sets, /clear removes.
	mux.HandleFunc("/api/assets/card-view", lobby.rateLimiter.WithRateLimit(lobby.handleCardView, "wallet-default"))
	mux.HandleFunc("/api/assets/card-view/clear", lobby.rateLimiter.WithRateLimit(lobby.handleClearCardView, "wallet-default"))
	// §23.5.1 placeholder art catalogue (placeholder_assets.go): the free starter pack every wallet
	// is granted, plus the shop the rest of the pack is purchased from.
	mux.HandleFunc("/api/assets/catalogue", lobby.rateLimiter.WithRateLimit(lobby.handlePlaceholderCatalogue, "wallet-default"))
	mux.HandleFunc("/api/assets/starter", lobby.rateLimiter.WithRateLimit(lobby.handleStarterAssets, "economy-tight"))
	mux.HandleFunc("/api/assets/purchase", lobby.rateLimiter.WithRateLimit(lobby.handleBuyPlaceholderSku, "economy-tight"))
	// §10.6 viewer-scoped slide theming (app boot / main menu + World Dashboard backgrounds).
	mux.HandleFunc("/api/assets/slide-theme", lobby.rateLimiter.WithRateLimit(lobby.handleSlideTheme, "wallet-default"))
	mux.HandleFunc("/api/assets/slide-theme/clear", lobby.rateLimiter.WithRateLimit(lobby.handleClearSlideTheme, "wallet-default"))
	// §10.7 UI-tree art: one unique placeholder frame per World Dashboard tree (category and
	// feature), with rewards + achievements pinned to Crypto-seraph. PUBLIC: it is the app's own
	// chrome art, not a viewer preference, and it reads no player state.
	mux.HandleFunc("/api/assets/ui-trees", lobby.rateLimiter.WithRateLimit(lobby.handleUiTreeArt, "wallet-default"))
	// §10.8 PLAYER-TO-PLAYER BONDED-ASSET MARKET (bonded_market_service.go): a wallet may offer its
	// own art for sale; the price is the listing's and the house fee goes to the sink.
	mux.HandleFunc("/api/assets/market", lobby.rateLimiter.WithRateLimit(lobby.handleBondedMarket, "wallet-default"))
	mux.HandleFunc("/api/assets/market/list", lobby.rateLimiter.WithRateLimit(lobby.handleBondedMarketList, "economy-tight"))
	mux.HandleFunc("/api/assets/market/cancel", lobby.rateLimiter.WithRateLimit(lobby.handleBondedMarketCancel, "wallet-default"))
	mux.HandleFunc("/api/assets/market/buy", lobby.rateLimiter.WithRateLimit(lobby.handleBondedMarketBuy, "economy-tight"))
	mux.HandleFunc("/api/theme/bind", lobby.rateLimiter.WithRateLimit(lobby.handleBindThemeAsset, "wallet-default"))
	mux.HandleFunc("/api/theme/lock", lobby.rateLimiter.WithRateLimit(lobby.handleLockThemeAsset, "wallet-default"))
	mux.HandleFunc("/api/local-model/promote", lobby.rateLimiter.WithRateLimit(lobby.handlePromoteLocalModel, "wallet-default"))
	mux.HandleFunc("/api/local-model/status", lobby.rateLimiter.WithRateLimit(lobby.handleLocalModelStatus, "wallet-default"))
	// §30 Pet World: entity event + stat + power-overlay endpoints.
	mux.HandleFunc("/api/entity-events/regions", lobby.rateLimiter.WithRateLimit(lobby.handleEntityRegions, "wallet-default"))
	mux.HandleFunc("/api/entity-event/resolve", lobby.rateLimiter.WithRateLimit(lobby.handleEntityEventResolve, "economy-tight"))
	mux.HandleFunc("/api/entity-event/host", lobby.rateLimiter.WithRateLimit(lobby.handleEntityEventHost, "wallet-default"))
	// §31 Synergy / Orphan / Combined-events endpoints.
	mux.HandleFunc("/api/owner/combined-stats", lobby.rateLimiter.WithRateLimit(lobby.handleCombineOwnerStats, "wallet-default"))
	mux.HandleFunc("/api/orphan/adopt", lobby.rateLimiter.WithRateLimit(lobby.handleAdoptEntity, "wallet-default"))
	mux.HandleFunc("/api/orphan/reclaim", lobby.rateLimiter.WithRateLimit(lobby.handleReclaimEntity, "wallet-default"))
	mux.HandleFunc("/api/orphan/status", lobby.rateLimiter.WithRateLimit(lobby.handleOrphanStatus, "wallet-default"))
	// §32 Faith + religious card-battle gambit endpoints.
	mux.HandleFunc("/api/faith/coherence", lobby.rateLimiter.WithRateLimit(lobby.handleFaithCoherence, "wallet-default"))
	mux.HandleFunc("/api/faith/war-gambit", lobby.rateLimiter.WithRateLimit(lobby.handleFaithWarGambit, "wallet-default"))

	// Faucet Dashboard — dynamic scaling + anti-whale AMM visualizer
	mux.HandleFunc("/api/faucet/status", lobby.rateLimiter.WithRateLimit(lobby.handleFaucetStatus, "wallet-default"))
	mux.HandleFunc("/api/faucet/vault-balance", lobby.rateLimiter.WithRateLimit(lobby.handleFaucetVaultBalance, "wallet-default"))
	mux.HandleFunc("/api/faucet/claim", lobby.rateLimiter.WithRateLimit(lobby.handleFaucetClaim, "economy-tight"))
	// Underworld placeholders (graceful empty lists until full tracking implemented)
	mux.HandleFunc("/api/underworld/heists", lobby.rateLimiter.WithRateLimit(lobby.handleUnderworldHeists, "underworld"))
	mux.HandleFunc("/api/underworld/kidnaps", lobby.rateLimiter.WithRateLimit(lobby.handleUnderworldKidnaps, "underworld"))

	// Entity Market + Pet Battle + Rivalry + Children Bots
	mux.HandleFunc("/api/entity/market/list", lobby.rateLimiter.WithRateLimit(lobby.handleEntityMarketList, "wallet-default"))
	mux.HandleFunc("/api/entity/market/create", lobby.rateLimiter.WithRateLimit(lobby.handleEntityMarketCreate, "wallet-default"))
	mux.HandleFunc("/api/entity/market/purchase", lobby.rateLimiter.WithRateLimit(lobby.handleEntityMarketPurchase, "wallet-default"))
	mux.HandleFunc("/api/pet-battle/list", lobby.rateLimiter.WithRateLimit(lobby.handlePetBattleList, "wallet-default"))
	mux.HandleFunc("/api/pet-battle/challenge", lobby.rateLimiter.WithRateLimit(lobby.handlePetBattleChallenge, "wallet-default"))
	mux.HandleFunc("/api/pet-battle/resolve", lobby.rateLimiter.WithRateLimit(lobby.handlePetBattleResolve, "wallet-default"))
	mux.HandleFunc("/api/rivalry/factions", lobby.rateLimiter.WithRateLimit(lobby.handleRivalryFactions, "wallet-default"))
	mux.HandleFunc("/api/rivalry/join", lobby.rateLimiter.WithRateLimit(lobby.handleRivalryJoin, "wallet-default"))
	mux.HandleFunc("/api/children-bots", lobby.rateLimiter.WithRateLimit(lobby.handleChildrenBots, "wallet-default"))

	// Religion Governance
	mux.HandleFunc("/api/faith/religions", lobby.rateLimiter.WithRateLimit(lobby.handleFaithReligions, "wallet-default"))
	mux.HandleFunc("/api/faith/religion/buy", lobby.rateLimiter.WithRateLimit(lobby.handleFaithReligionBuy, "wallet-default"))
	mux.HandleFunc("/api/faith/religion/buyout", lobby.rateLimiter.WithRateLimit(lobby.handleFaithReligionBuyout, "wallet-default"))
	mux.HandleFunc("/api/faith/religion/join", lobby.rateLimiter.WithRateLimit(lobby.handleFaithReligionJoin, "wallet-default"))
	mux.HandleFunc("/api/faith/religion/ritual", lobby.rateLimiter.WithRateLimit(lobby.handleFaithReligionRitual, "wallet-default"))
	mux.HandleFunc("/api/faith/religion/rivalry", lobby.rateLimiter.WithRateLimit(lobby.handleFaithReligionRivalry, "wallet-default"))
	// NOTE: /api/faith/coherence and /api/faith/war-gambit are already registered
	// in the §32 Faith + gambit block above — do NOT re-register (Go 1.22+ ServeMux panics).
	mux.HandleFunc("/api/faith/high-tier", lobby.rateLimiter.WithRateLimit(lobby.handleFaithHighTier, "wallet-default"))

	// Persistent Identity System
	mux.HandleFunc("/api/identity/profile", lobby.rateLimiter.WithRateLimit(lobby.handleIdentityProfile, "wallet-default"))
	mux.HandleFunc("/api/identity/events", lobby.rateLimiter.WithRateLimit(lobby.handleIdentityEvents, "wallet-default"))
	mux.HandleFunc("/api/identity/leaderboard", lobby.rateLimiter.WithRateLimit(lobby.handleIdentityLeaderboard, "wallet-default"))
	mux.HandleFunc("/api/identity/record", lobby.rateLimiter.WithRateLimit(lobby.handleIdentityRecord, "wallet-default"))

	// Player Progression (constellation unlock states)
	mux.HandleFunc("/api/player/progression", lobby.rateLimiter.WithRateLimit(lobby.handlePlayerProgression, "wallet-default"))

	// Players Constellation Browser
	mux.HandleFunc("/api/players/constellation", lobby.rateLimiter.WithRateLimit(lobby.handleGetPlayersConstellation, "default"))

	// Player Token Balances (NUGGET/UNIT)
	mux.HandleFunc("/api/player/tokens", lobby.rateLimiter.WithRateLimit(lobby.handleGetPlayerTokens, "wallet-default"))

	// Player Association Export (§3 READ-ONLY PORTFOLIO) — the wallet associations the
	// engine holds on PlayerStats that no other read-only route exposes.
	mux.HandleFunc("/api/player/associations", lobby.rateLimiter.WithRateLimit(lobby.handlePlayerAssociations, "wallet-default"))

	// Entity Shares System
	mux.HandleFunc("/api/shares/issue", lobby.rateLimiter.WithRateLimit(lobby.handleEntitySharesIssue, "wallet-default"))
	mux.HandleFunc("/api/shares/buy", lobby.rateLimiter.WithRateLimit(lobby.handleEntitySharesBuy, "wallet-default"))
	mux.HandleFunc("/api/shares/tokens", lobby.rateLimiter.WithRateLimit(lobby.handleEntitySharesTokens, "wallet-default"))
	mux.HandleFunc("/api/shares/holdings", lobby.rateLimiter.WithRateLimit(lobby.handleEntitySharesHoldings, "wallet-default"))

	// Infrastructure Leasing System
	mux.HandleFunc("/api/lease/create", lobby.rateLimiter.WithRateLimit(lobby.handleLeaseCreate, "wallet-default"))
	mux.HandleFunc("/api/lease/list", lobby.rateLimiter.WithRateLimit(lobby.handleLeaseList, "wallet-default"))
	mux.HandleFunc("/api/lease/available", lobby.rateLimiter.WithRateLimit(lobby.handleLeaseAvailable, "wallet-default"))

	// Governance System
	mux.HandleFunc("/api/governance/weight", lobby.rateLimiter.WithRateLimit(lobby.handleGovernanceWeight, "wallet-default"))
	mux.HandleFunc("/api/governance/register", lobby.rateLimiter.WithRateLimit(lobby.handleGovernanceRegister, "wallet-default"))
	mux.HandleFunc("/api/governance/vote", lobby.rateLimiter.WithRateLimit(lobby.handleGovernanceVote, "wallet-default"))
	mux.HandleFunc("/api/governance/close", lobby.rateLimiter.WithRateLimit(lobby.handleGovernanceClose, "wallet-default"))
	mux.HandleFunc("/api/governance/governor", lobby.rateLimiter.WithRateLimit(lobby.handleGovernanceGovernor, "wallet-default"))
	mux.HandleFunc("/api/governance/election", lobby.rateLimiter.WithRateLimit(lobby.handleGovernanceElection, "wallet-default"))
	mux.HandleFunc("/api/governance/leaderboard", lobby.rateLimiter.WithRateLimit(lobby.handleGovernanceLeaderboard, "wallet-default"))

	// Bridge Router System
	mux.HandleFunc("/api/bridge/asset", lobby.rateLimiter.WithRateLimit(lobby.handleBridgeAsset, "wallet-default"))
	mux.HandleFunc("/api/bridge/confirm", lobby.rateLimiter.WithRateLimit(lobby.handleBridgeConfirm, "wallet-default"))
	mux.HandleFunc("/api/bridge/assets", lobby.rateLimiter.WithRateLimit(lobby.handleBridgeAssets, "wallet-default"))
	mux.HandleFunc("/api/bridge/txs", lobby.rateLimiter.WithRateLimit(lobby.handleBridgeTxs, "wallet-default"))
	mux.HandleFunc("/api/bridge/summary", lobby.rateLimiter.WithRateLimit(lobby.handleBridgeSummary, "wallet-default"))

	// Creator Economy System
	mux.HandleFunc("/api/creator/dlc/create", lobby.rateLimiter.WithRateLimit(lobby.handleCreatorDLCCreate, "wallet-default"))
	mux.HandleFunc("/api/creator/dlc/purchase", lobby.rateLimiter.WithRateLimit(lobby.handleCreatorDLCPurchase, "wallet-default"))
	mux.HandleFunc("/api/creator/dlcs", lobby.rateLimiter.WithRateLimit(lobby.handleCreatorDLCs, "wallet-default"))
	mux.HandleFunc("/api/creator/royalties", lobby.rateLimiter.WithRateLimit(lobby.handleCreatorRoyalties, "wallet-default"))
	mux.HandleFunc("/api/creator/event/create", lobby.rateLimiter.WithRateLimit(lobby.handleCreatorEventCreate, "wallet-default"))
	mux.HandleFunc("/api/creator/events", lobby.rateLimiter.WithRateLimit(lobby.handleCreatorEvents, "wallet-default"))
	mux.HandleFunc("/api/creator/event/attend", lobby.rateLimiter.WithRateLimit(lobby.handleCreatorEventAttend, "wallet-default"))
	mux.HandleFunc("/api/creator/sub/create", lobby.rateLimiter.WithRateLimit(lobby.handleCreatorSubCreate, "wallet-default"))
	mux.HandleFunc("/api/creator/subs", lobby.rateLimiter.WithRateLimit(lobby.handleCreatorSubs, "wallet-default"))

	// Faith Church Storefront
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

	// Launchpad System
	mux.HandleFunc("/api/launch/create", lobby.rateLimiter.WithRateLimit(lobby.handleLaunchCreate, "wallet-default"))
	mux.HandleFunc("/api/launch/back", lobby.rateLimiter.WithRateLimit(lobby.handleLaunchBack, "wallet-default"))
	mux.HandleFunc("/api/launch/activate", lobby.rateLimiter.WithRateLimit(lobby.handleLaunchActivate, "wallet-default"))
	mux.HandleFunc("/api/launch/integrate", lobby.rateLimiter.WithRateLimit(lobby.handleLaunchIntegrate, "wallet-default"))
	mux.HandleFunc("/api/launches", lobby.rateLimiter.WithRateLimit(lobby.handleLaunches, "wallet-default"))
	mux.HandleFunc("/api/launch/get", lobby.rateLimiter.WithRateLimit(lobby.handleLaunchGet, "wallet-default"))
	mux.HandleFunc("/api/launch/creator", lobby.rateLimiter.WithRateLimit(lobby.handleLaunchesByCreator, "wallet-default"))

	// === New API Routes (ported from Triple Triad) ===
	mux.HandleFunc("/api/bounty/active", lobby.rateLimiter.WithRateLimit(lobby.HandleGetBountyActive, "underworld"))
	// NOTE: /api/justice/missions is registered below in the High-Priority block
	// (HandleGetJusticeMissionsHTTP) — do NOT re-register here (Go 1.22+ ServeMux panics).
	mux.HandleFunc("/api/church/members", lobby.rateLimiter.WithRateLimit(lobby.HandleGetChurchMembers, "wallet-default"))
	mux.HandleFunc("/api/shop/purchase", lobby.rateLimiter.WithRateLimit(lobby.HandleShopPurchase, "economy-tight"))
	mux.HandleFunc("/api/clubs", lobby.rateLimiter.WithRateLimit(lobby.HandleGetClubs, "wallet-default"))
	mux.HandleFunc("/api/creator/store/creator", lobby.rateLimiter.WithRateLimit(lobby.HandleGetCreatorStore, "wallet-default"))
	mux.HandleFunc("/api/player/profile", lobby.rateLimiter.WithRateLimit(lobby.HandleGetPlayerProfile, "wallet-default"))
	mux.HandleFunc("/api/rumors", lobby.rateLimiter.WithRateLimit(lobby.HandleGetRumors, "wallet-default"))
	mux.HandleFunc("/api/envoi-name", lobby.rateLimiter.WithRateLimit(lobby.handleEnvoiName, "wallet-default"))
	// === End New Routes ===

	// === High Priority API Routes (back existing UIs) ===
	mux.HandleFunc("/api/titles", lobby.rateLimiter.WithRateLimit(lobby.HandleGetCardTitles, "wallet-default"))
	mux.HandleFunc("/api/titles/equip", lobby.rateLimiter.WithRateLimit(lobby.HandleEquipCardTitle, "wallet-default"))
	mux.HandleFunc("/api/wagers", lobby.rateLimiter.WithRateLimit(lobby.HandleGetWagers, "wallet-default"))
	mux.HandleFunc("/api/wagers/resolve", lobby.rateLimiter.WithRateLimit(lobby.HandleResolveWager, "economy-tight"))
	mux.HandleFunc("/api/justice/missions", lobby.rateLimiter.WithRateLimit(lobby.HandleGetJusticeMissionsHTTP, "underworld"))
	mux.HandleFunc("/api/justice/missions/accept", lobby.rateLimiter.WithRateLimit(lobby.HandleAcceptJusticeMissionHTTP, "underworld"))
	mux.HandleFunc("/api/tea/recipes", lobby.rateLimiter.WithRateLimit(lobby.HandleGetTeaRecipes, "wallet-default"))
	mux.HandleFunc("/api/tea/brew", lobby.rateLimiter.WithRateLimit(lobby.HandleBrewTea, "economy-tight"))
	mux.HandleFunc("/api/garden", lobby.rateLimiter.WithRateLimit(lobby.HandleGetGarden, "wallet-default"))
	mux.HandleFunc("/api/garden/add", lobby.rateLimiter.WithRateLimit(lobby.HandleAddGardenElement, "economy-tight"))
	mux.HandleFunc("/api/garden/meditate", lobby.rateLimiter.WithRateLimit(lobby.HandleMeditate, "wallet-default"))
	mux.HandleFunc("/api/moods", lobby.rateLimiter.WithRateLimit(lobby.HandleGetMoods, "wallet-default"))
	mux.HandleFunc("/api/moods/adjust", lobby.rateLimiter.WithRateLimit(lobby.HandleAdjustMood, "wallet-default"))
	// === End High Priority Routes ===

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

	// Client Error Feed
	// (diagnostics page registered later at line ~1397)

	// Replay Engine
	mux.HandleFunc("/api/replay/state", lobby.rateLimiter.WithRateLimit(lobby.handleReplayState, "wallet-default"))
	mux.HandleFunc("/api/replay/latest", lobby.rateLimiter.WithRateLimit(lobby.handleReplayLatest, "wallet-default"))
	mux.HandleFunc("/api/replay/frames", lobby.rateLimiter.WithRateLimit(lobby.handleReplayFrames, "wallet-default"))
	mux.HandleFunc("/api/replay/player", lobby.rateLimiter.WithRateLimit(lobby.handleReplayPlayer, "wallet-default"))
	mux.HandleFunc("/api/replay/capture", lobby.rateLimiter.WithRateLimit(lobby.handleReplayCapture, "wallet-default"))
	mux.HandleFunc("/api/replay/start", lobby.rateLimiter.WithRateLimit(lobby.handleReplayStart, "wallet-default"))
	mux.HandleFunc("/api/replay/stop", lobby.rateLimiter.WithRateLimit(lobby.handleReplayStop, "wallet-default"))

	// Stat Overlay System
	mux.HandleFunc("/api/stat-overlay", lobby.rateLimiter.WithRateLimit(lobby.handleStatOverlayGet, "wallet-default"))
	mux.HandleFunc("/api/stat-overlay/region", lobby.rateLimiter.WithRateLimit(lobby.handleStatOverlayRegion, "wallet-default"))
	mux.HandleFunc("/api/stat-overlay/owner", lobby.rateLimiter.WithRateLimit(lobby.handleStatOverlayOwner, "wallet-default"))
	mux.HandleFunc("/api/stat-overlay/leaderboard", lobby.rateLimiter.WithRateLimit(lobby.handleStatOverlayLeaderboard, "wallet-default"))

	// Industrial Loop Engine
	mux.HandleFunc("/api/industrial-loop/metrics", lobby.rateLimiter.WithRateLimit(lobby.handleIndustrialLoopMetrics, "wallet-default"))
	mux.HandleFunc("/api/industrial-loop/health", lobby.rateLimiter.WithRateLimit(lobby.handleIndustrialLoopHealth, "wallet-default"))
	mux.HandleFunc("/api/industrial-loop/record", lobby.rateLimiter.WithRateLimit(lobby.handleIndustrialLoopRecord, "wallet-default"))

	mux.HandleFunc("/api/report-player", lobby.rateLimiter.WithRateLimit(lobby.handlePlayerReport, "default"))
	mux.HandleFunc("/api/re-sync-stats", lobby.rateLimiter.WithRateLimit(lobby.handleReSyncStats, "wallet-default"))
	mux.HandleFunc("/api/season/history", lobby.rateLimiter.WithRateLimit(lobby.handleSeasonHistory, "wallet-default"))

	mux.HandleFunc("/api/tournament/register", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		lobby.tournamentService.HandleTournamentRegister(lobby, w, r)
	}, "standard"))
	mux.HandleFunc("/api/tournament/history", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		lobby.tournamentService.HandleTournamentHistory(lobby, w, r)
	}, "wallet-default"))

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
	mux.HandleFunc("/api/admin/networks", lobby.rateLimiter.WithRateLimit(lobby.handleAdminNetworks, "wallet-default")) // READ-ONLY registry authority view (redacted; secret presence only)
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
	// Workstream C: THE CONSOLE ENTITLEMENT PATH. The fulfilment WRITER is registered on the PRIMARY
	// server only, matching the voucher redemption gateway beside it: this is the door the PLATFORM calls,
	// it grants the entitlement the platform's money already paid for, and the console build holds 0 chain
	// rails by ruling — a console-local grant would be a virtual-mirror record that never reaches the
	// account. The asymmetry is deliberate, and the served contract states it.
	mux.HandleFunc("/api/console/entitlement", lobby.rateLimiter.WithRateLimit(lobby.handleConsoleEntitlementGrant, "wallet-default"))
	mux.HandleFunc("/api/console/entitlements", lobby.rateLimiter.WithRateLimit(lobby.handleConsoleEntitlements, "wallet-default"))

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

	mux.HandleFunc("/api/courthouse/reset", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		lobby.courthouseService.HandleCourthouseReset(lobby, w, r)
	}, "wallet-default"))
	mux.HandleFunc("/api/loans", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) { lobby.loanService.HandleGetLoans(lobby, w, r) }, "standard"))
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
	mux.HandleFunc("/api/bridge/onboard", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) { lobby.onboardingService.HandleVoiOnboarding(lobby, w, r) }, "economy-tight"))

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
	mux.HandleFunc("/api/creator/store/product/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		productID := ""
		if len(parts) >= 5 {
			productID = parts[4]
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
	})
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
	})
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
	})
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

	// ── §26.4 Pets ──
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
	// §26.4.2 grooming ladder — the counterpart to §25.6.1 vehicle parts. A groom is a
	// PURCHASED account upgrade: the price is computed server-side and sink-routed.
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

	// ── §25.6 Vehicles ──
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
	// §25.6.1 vehicle upgrade ladder — the counterpart to §26.4 pet breeding. A part is
	// fitted for a deterministic, sink-routed fee; every fee-path change is economy-tight.
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
	// Deploy parks a vehicle in a region so it feeds the §30 region power overlay.
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

	// ── §25.7 World Content ──
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

	// Domestic AI System (§27.7.1)
	mux.HandleFunc("/api/ai/citizens/marry", lobby.rateLimiter.WithRateLimit(lobby.handleMarryAI, "wallet-default"))
	mux.HandleFunc("/api/ai/citizens/breed", lobby.rateLimiter.WithRateLimit(lobby.handleBreedAI, "wallet-default"))
	mux.HandleFunc("/api/ai/citizens/adopt-pet", lobby.rateLimiter.WithRateLimit(lobby.handleAdoptPetAI, "wallet-default"))
	mux.HandleFunc("/api/ai/citizens/progress", lobby.rateLimiter.WithRateLimit(lobby.handleAIProgression, "wallet-default"))

	// KEY 3.5 Slice 2 — Item Shop Archetype System (v4 §14)
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
	mux.HandleFunc("/api/creator/store/resell", lobby.rateLimiter.WithRateLimit(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			wallet := extractWalletFromRequest(r)
			if wallet == "" {
				http.Error(w, "wallet required", http.StatusBadRequest)
				return
			}
			var req struct {
				ProductID         string `json:"product_id"`
				SalePriceMicroVBV uint64 `json:"sale_price_micro_vbv"`
				BuyerWallet       string `json:"buyer_wallet"`
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
			product, err := cs.GetProduct(req.ProductID)
			if err != nil || product.CreatorWallet == "" {
				http.Error(w, "product not found or invalid", http.StatusBadRequest)
				return
			}
			if req.SalePriceMicroVBV == 0 {
				http.Error(w, "sale price must be greater than zero", http.StatusBadRequest)
				return
			}
			tx, err := cs.ProcessSecondarySale(req.ProductID, req.BuyerWallet, wallet, req.SalePriceMicroVBV)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if lobby.tokenSinkRouter != nil {
				matrix := RevenueSplitMatrix{
					FaucetShare:     0.5,
					ClubShare:       0.3,
					GovernanceShare: 0.2,
					CreatorRoyalty:  DefaultRoyaltyRate,
				}
				if err := lobby.tokenSinkRouter.RouteCriminalTax("CREATOR_SECONDARY_SALE", req.SalePriceMicroVBV, matrix, 0, ""); err != nil {
					log.Printf("[ROYALTY_ROUTE] WARNING: economy split failed for secondary sale %s: %v", tx.ID, err)
				}
			}
			payload := fmt.Sprintf(`{"product_id":"%s","creator_wallet":"%s","seller_wallet":"%s","amount_micro_vbv":%d,"royalty_paid_micro_vbv":%d,"timestamp":"%s"}`,
				tx.ProductID, tx.CreatorWallet, tx.SellerWallet, tx.AmountMicroVBV, tx.RoyaltyPaidMicroVBV, tx.Timestamp.Format(time.RFC3339))
			lobby.mutex.Lock()
			if cid := lobby.getClientIDFromWalletLocked(wallet); cid != "" {
				lobby.sendToClientLocked(cid, Envelope{Type: "creator_royalty_paid", Payload: json.RawMessage(payload)})
			}
			if req.BuyerWallet != wallet && lobby.getClientIDFromWalletLocked(req.BuyerWallet) != "" {
				bid := lobby.getClientIDFromWalletLocked(req.BuyerWallet)
				lobby.sendToClientLocked(bid, Envelope{Type: "creator_royalty_paid", Payload: json.RawMessage(payload)})
			}
			if cid := lobby.getClientIDFromWalletLocked(tx.CreatorWallet); cid != "" {
				lobby.sendToClientLocked(cid, Envelope{Type: "creator_royalty_received", Payload: json.RawMessage(payload)})
			}
			lobby.mutex.Unlock()
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":                    "secondary_sale_complete",
				"royalty_transaction_id":    tx.ID,
				"creator_royalty_micro_vbv": tx.RoyaltyPaidMicroVBV,
				"seller_net_proceeds":       req.SalePriceMicroVBV - tx.RoyaltyPaidMicroVBV,
			})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, "economy_tight"))

	// ── Dev diagnostics: client-side error capture (so the dev doesn't paste console errors) ──
	// In-memory ring buffer of client-reported JS errors (dev only; not persisted).
	type clientErr struct {
		Ts     string `json:"ts"`
		Msg    string `json:"msg"`
		Src    string `json:"src"`
		Line   int    `json:"line"`
		Col    int    `json:"col"`
		Stack  string `json:"stack"`
		Wallet string `json:"wallet"`
		Count  int    `json:"count"`
	}
	var clientErrMu sync.Mutex
	clientErrLog := make([]clientErr, 0, 200)

	mux.HandleFunc("/api/client-error", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var e clientErr
		if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if e.Ts == "" {
			e.Ts = time.Now().Format(time.RFC3339)
		}
		clientErrMu.Lock()
		// De-dupe burst: if the last entry is identical (Msg+Src+Line), fold it into a
		// repeat counter instead of appending (prevents one resize-crash from burying
		// every other distinct error in the ring buffer).
		if n := len(clientErrLog); n > 0 {
			last := &clientErrLog[n-1]
			if last.Msg == e.Msg && last.Src == e.Src && last.Line == e.Line {
				last.Count++
				last.Ts = e.Ts
				clientErrMu.Unlock()
				log.Printf("[CLIENT-ERR] %s | %s @ %s:%d (repeat x%d)", e.Wallet, e.Msg, e.Src, e.Line, last.Count)
				writeJSON(w, map[string]interface{}{"success": true})
				return
			}
		}
		e.Count = 1
		clientErrLog = append(clientErrLog, e)
		if len(clientErrLog) > 200 {
			clientErrLog = clientErrLog[len(clientErrLog)-200:]
		}
		clientErrMu.Unlock()
		log.Printf("[CLIENT-ERR] %s | %s @ %s:%d", e.Wallet, e.Msg, e.Src, e.Line)
		writeJSON(w, map[string]interface{}{"success": true})
	})

	// Diagnostics page: live view of captured client errors (open http://localhost:8090/diagnostics)
	mux.HandleFunc("/diagnostics", func(w http.ResponseWriter, r *http.Request) {
		clientErrMu.Lock()
		logged := make([]clientErr, len(clientErrLog))
		copy(logged, clientErrLog)
		clientErrMu.Unlock()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		fmt.Fprintf(w, `<!doctype html><html><head><meta charset="utf-8"><title>Dev Diagnostics</title>
<style>body{background:#0a0e14;color:#cfe;font:14px monospace;padding:20px}
h1{color:#0fc}h2{color:#0cf} .e{border:1px solid #234;margin:8px 0;padding:8px;border-radius:6px;background:#11182a}
.msg{color:#f88}.meta{color:#8ab}.ok{color:#0f8}</style></head>
<body><h1>NFT-Seduction Dev Diagnostics</h1>
<button onclick="location.reload()" style="padding:6px 12px">Refresh</button>
<p>Server time: %s</p>
<h2>Captured client errors (%d)</h2>
<div id="errs">`, time.Now().Format(time.RFC3339), len(logged))
		if len(logged) == 0 {
			fmt.Fprintf(w, `<p class="ok">No client errors captured yet. The client reports errors automatically via /api/client-error.</p>`)
		}
		for _, e := range logged {
			rep := ""
			if e.Count > 1 {
				rep = fmt.Sprintf(" (×%d)", e.Count)
			}
			fmt.Fprintf(w, `<div class="e"><div class="msg">%s</div><div class="meta">%s @ %s:%d%s (wallet: %s)</div><pre>%s</pre></div>`,
				htmlEsc(e.Msg), htmlEsc(e.Ts), htmlEsc(e.Src), e.Line, rep, htmlEsc(e.Wallet), htmlEsc(e.Stack))
		}
		fmt.Fprintf(w, `</div>
<p style="color:#678">Tip: errors are captured automatically by the client (window.onerror + unhandledrejection). No need to paste console output.</p>
</body></html>`)
	})

	// Static Asset Serving (WASM and UI)
	fs := http.FileServer(http.Dir("./Public"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Disable caching for all static assets so clients always revalidate (no stale
		// ES-module / three.js caches on localhost — fixes repeated "updateStaffTrainingVisuals
		// already declared" / "three.core.js 404" errors from cached old builds).
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		fs.ServeHTTP(w, r)
	})

	// Dashboard & Spectate (served from repo root)
	mux.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		http.ServeFile(w, r, "./dashboard.html")
	})
	mux.HandleFunc("/world", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		http.ServeFile(w, r, "./Public/world.html")
	})
	mux.HandleFunc("/spectate", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		http.ServeFile(w, r, "./spectate.html")
	})

	// Manuals & Tutorials (served from AI-Brain)
	mux.HandleFunc("/manuals/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		http.ServeFile(w, r, "./AI-Brain/Manuals/"+r.URL.Path[len("/manuals/"):])
	})
	mux.HandleFunc("/tutorials/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		http.ServeFile(w, r, "./AI-Brain/Tutorials/"+r.URL.Path[len("/tutorials/"):])
	})
	mux.HandleFunc("/watch", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		http.ServeFile(w, r, "./watch-feed.html")
	})
	mux.HandleFunc("/split", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		http.ServeFile(w, r, "./split-view.html")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8088"
	}

	// Kill any existing server process on this port
	killExistingServer(port)

	fmt.Printf(" VOICONOMY ARENA SERVER ONLINE: PORT %s\n", port)
	fmt.Println(" WebSocket Switchboard & API Ready               ")
	fmt.Println(" Rate Limiter Active: 7-tier wallet-based system  ")
	fmt.Println("-------------------------------------------------")

	// Graceful shutdown on Ctrl+C
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		fmt.Println("\n[SERVER] Shutdown signal received. Cleaning up...")
		os.Exit(0)
	}()

	if err := http.ListenAndServe(":"+port, corsMiddleware()(mux)); err != nil {
		log.Fatalf("[FATAL] Server startup failed: %v", err)
	}
}
