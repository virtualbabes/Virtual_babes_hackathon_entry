//go:build !js && !wasm

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/client/v2/algod"
)

// handleNoteVocabulary serves the READ-ONLY note-purpose vocabulary
// (note_vocabulary.go is its single owner).
//
// A browser reads the prefixes it is allowed to write from here instead of
// re-declaring them, so a client and the server can never disagree about a note
// prefix — a disagreement that would NOT raise an error, only silently fail
// verification. The payload states the rules (client-may-name-money-door-only,
// exact prefix match, empty prefix refused) so a surface renders the rule rather
// than inventing one.
func handleNoteVocabulary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(NoteVocabularyPayload())
}

// handleSpectatorWager processes spectator bets on ongoing matches.
// PILLAR 2: Industrial Loop (Spectator Siphon).
func (l *Lobby) handleSpectatorWager(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SpectatorWallet string `json:"spectator_wallet"`
		MatchID         string `json:"match_id"` // This should be the P1ID of the match
		BetOnWallet     string `json:"bet_on_wallet"`
		WagerMicro      uint64 `json:"wager_micro"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SpectatorWallet == "" || req.MatchID == "" || req.BetOnWallet == "" || req.WagerMicro == 0 {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	spectatorWallet := strings.ToLower(req.SpectatorWallet)
	betOnWallet := strings.ToLower(req.BetOnWallet)

	l.mutex.Lock()
	defer l.mutex.Unlock()

	// 1. Validate Spectator's Balance
	if l.playerBalances[spectatorWallet] < req.WagerMicro {
		http.Error(w, "Insufficient rewards for wager", http.StatusPaymentRequired)
		return
	}

	// 2. Find the Match and Validate Participants
	match, ok := l.matches[req.MatchID] // MatchState is keyed by P1ID
	if !ok {
		http.Error(w, "Match not found or no longer active", http.StatusNotFound)
		return
	}

	// Ensure the match is still active and not finished
	if match.IsFinished {
		http.Error(w, "Cannot place wager on a finished match", http.StatusForbidden)
		return
	}

	// Ensure the spectator is actually spectating this match
	isSpectator := false
	for _, sID := range match.Spectators {
		if sWallet, ok := l.wallets[sID]; ok && strings.EqualFold(sWallet, spectatorWallet) {
			isSpectator = true
			break
		}
	}
	if !isSpectator {
		http.Error(w, "You are not spectating this match", http.StatusForbidden)
		return
	}
	// PILLAR 2: Fair Play Enforcement. Prevent players from betting on their own matches.
	if strings.EqualFold(match.P1Wallet, spectatorWallet) || strings.EqualFold(match.P2Wallet, spectatorWallet) {
		http.Error(w, "You cannot place a wager on your own match.", http.StatusForbidden)
		return
	}

	// Ensure betOnWallet is a valid participant in the match
	if !strings.EqualFold(match.P1Wallet, betOnWallet) && !strings.EqualFold(match.P2Wallet, betOnWallet) {
		http.Error(w, "Invalid player to bet on", http.StatusBadRequest)
		return
	}

	// 3. Deduct Wager from Spectator and Add to Match Pool
	l.playerBalances[spectatorWallet] -= req.WagerMicro
	match.WagersMicro += req.WagerMicro

	// 4. Log the event
	l.logAdminAuditLocked("SPECTATOR_WAGER", spectatorWallet, fmt.Sprintf("Match: %s, BetOn: %s, Amount: %d micro-VBV", req.MatchID, betOnWallet, req.WagerMicro))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "message": "Wager placed successfully"})
}

func (l *Lobby) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	type entry struct {
		Wins             int       `json:"wins"`
		DNFs             int       `json:"dnfs"`
		DisconnectStreak int       `json:"disconnect_streak"`
		Reputation       int       `json:"reputation"`
		BestRating       string    `json:"best_rating"`
		BanExpires       time.Time `json:"ban_expires"`
		Wallet           string    `json:"wallet"`
		TotalDonated     uint64    `json:"total_donated"`
		ReparationsReceivedCount int `json:"reparations_received_count"`
	}
	// Non-nil so an empty civilization encodes as [] rather than null — a reader
	// must never see "null" where the honest answer is "no players yet".
	list := make([]entry, 0)
	l.mutex.RLock()
	for w, stats := range l.leaderboard {
		list = append(list, entry{
			Wins: stats.Wins, DNFs: stats.DNFs, DisconnectStreak: stats.DisconnectStreak,
			Reputation: stats.Reputation, BestRating: stats.BestRating,
			BanExpires: stats.BanExpires, Wallet: w,
			TotalDonated: stats.TotalDonated,
			ReparationsReceivedCount: stats.ReparationsReceivedCount,
		})
	}
	l.mutex.RUnlock()

	sort.Slice(list, func(i, j int) bool { return list[i].Wins > list[j].Wins })
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// handleGetPlayersConstellation returns all active players with their constellation progression states.
// Used by constellation_spectate.js to render the multi-player constellation browser.
func (l *Lobby) handleGetPlayersConstellation(w http.ResponseWriter, r *http.Request) {
	type playerConstellation struct {
		Wallet      string            `json:"wallet"`
		Name        string            `json:"name"`
		Features    map[string]string `json:"features"`
		CareerTier  int               `json:"career_tier"`
		Region      string            `json:"region"`
		Territories int               `json:"territories"`
		IsGovernor  bool              `json:"is_governor"`
	}

	l.mutex.RLock()
	players := make([]playerConstellation, 0, len(l.leaderboard))
	for wallet, stats := range l.leaderboard {
		// Find player's club + territory count
		var playerClub *Club
		ownedTerritories := 0
		for _, club := range l.clubs {
			if strings.EqualFold(club.OwnerWallet, wallet) {
				playerClub = club
				ownedTerritories = len(club.Territories)
				break
			}
		}

		isGovernor := false
		if playerClub != nil && l.clubService != nil {
			isGovernor = l.clubService.IsClubRegionalLocked(l, playerClub)
		}

		regionName := "Base"
		if playerClub != nil && playerClub.RegionName != "" {
			regionName = playerClub.RegionName
		}

		careerTier := 0
		if stats.CareerXP != nil && stats.JobRole != "" {
			careerTier = stats.CareerXP.GetCareerTier(stats.JobRole)
		}

		// Build feature map from progression
		features := map[string]string{
			"shops":       "dormant",
			"career":      "dormant",
			"clubs":       "dormant",
			"territory":   "dormant",
			"governor":    "dormant",
			"rivalry":     "dormant",
			"faith":       "dormant",
			"stats":       "dormant",
			"regional":    "dormant",
			"tournament":  "dormant",
			"entity":      "dormant",
			"children":    "dormant",
			"creator":     "dormant",
			"governance":   "dormant",
			"leasing":     "dormant",
			"identity":    "dormant",
		}

		// Unlock features based on progression
		if stats.Wins > 0 {
			features["career"] = "alive"
		}
		if playerClub != nil {
			features["clubs"] = "alive"
			features["territory"] = "alive"
		}
		if isGovernor {
			features["governor"] = "alive"
			features["regional"] = "alive"
			features["governance"] = "alive"
		}
		if stats.Reputation > 50 {
			features["stats"] = "alive"
		}
		if stats.Achievements != nil && len(stats.Achievements) > 0 {
			features["tournament"] = "alive"
		}

		players = append(players, playerConstellation{
			Wallet:      wallet,
			Name:        stats.Name,
			Features:    features,
			CareerTier:  careerTier,
			Region:      regionName,
			Territories: ownedTerritories,
			IsGovernor:  isGovernor,
		})
	}
	l.mutex.RUnlock()

	writeJSON(w, map[string]interface{}{"success": true, "players": players})
}

// handleGetPlayerTokens returns NUGGET/UNIT token balances for a wallet.
// Used by constellation_spectate.js for the token balance widget.
func (l *Lobby) handleGetPlayerTokens(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}

	l.mutex.RLock()
	balance := l.playerBalances[wallet]
	l.mutex.RUnlock()

	writeJSON(w, map[string]interface{}{
		"success": true,
		"wallet":  wallet,
		"balance": balance,
		// NUGGET/UNIT placeholders — these require ASA IDs from env vars
		"nugget_balance": nil,
		"unit_balance":   nil,
	})
}

// ============================================================================
// §3 READ-ONLY PORTFOLIO — engine association export
// ============================================================================
// handlePlayerAssociations exports the wallet associations the engine holds on
// PlayerStats that NO other read-only route exposes: the alliance graph, the
// warrant network, the custodial maps (jailed / kidnapped / held-hostage /
// captured outlaws), buff and debuff flags, sector tiles, character
// relationships, sustained-liquidity samples, cooldown timers and the forensic
// mutation record.
//
// Contract (binding):
//   - READ-ONLY. It takes a read lock only and never mutates state.
//   - Money is uint64 micro-$VBV. Engine ratios that are STORED as float64
//     (aggressiveness, risk tolerance, mojo decay, playstyle weights) are
//     converted ONCE at this HTTP boundary into integer parts-per-million so the
//     payload carries no float (Architecture Ledger). The converted value is
//     presented only — never stored, summed or compared.
//   - Absent maps/slices are emitted as {} / [] so a reader can never confuse
//     "not exported" with "empty".
//   - A wallet the engine holds no record for is reported as present=false with a
//     reason. That is NOT the same statement as "this wallet has no alliances",
//     and must not be rendered as one.
func (l *Lobby) handlePlayerAssociations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}

	l.mutex.RLock()
	stats, exists := l.leaderboard[wallet]
	balance := l.playerBalances[wallet]
	l.mutex.RUnlock()

	if !exists {
		// Honest absence: the engine has no PlayerStats row for this wallet yet.
		writeJSON(w, map[string]interface{}{
			"success": true,
			"wallet":  wallet,
			"present": false,
			"note":    "the engine holds no PlayerStats record for this wallet yet, so no associations are reported",
		})
		return
	}

	// Most recent sustained-liquidity samples only; the full count is reported
	// separately so the payload stays bounded without hiding the truth.
	const liquidityTail = 64
	samples := stats.LiquiditySamples
	if len(samples) > liquidityTail {
		samples = samples[len(samples)-liquidityTail:]
	}

	writeJSON(w, map[string]interface{}{
		"success": true,
		"wallet":  wallet,
		"present": true,


		// ── Identity + ledger mirrors (authoritative balance = playerBalances) ──
		"name":                         stats.Name,
		"id":                           stats.ID,
		"last_activity":                stats.LastActivity,
		"equipped_faceplate":           stats.EquippedFaceplate,
		"career":                       stats.Career,
		"career_tier":                  stats.CareerTier,
		"career_level":                 nonNilMap(stats.CareerLevel),
		"vbv_balance_micro":            balance,
		"credits":                      stats.Credits,
		"market_tokens_micro":          stats.MarketTokens,
		"arena_vouchers":               stats.ArenaVouchers,
		"total_donated_micro":          stats.TotalDonated,
		"total_dividend_claimed_micro": stats.TotalDividendClaimedMicro,
		"bounty_hunter_bond_micro":     stats.BountyHunterBondMicro,

		// ── Alliances (no other endpoint exports these) ──
		"alliances":          nonNilMap(stats.Alliances),
		"active_alliance_id": stats.ActiveAllianceID,

		// ── Warrant network + risk record ──
		"wanted":               nonNilMap(stats.Wanted),
		"wanted_level":         stats.WantedLevel,
		"heist_attempts":       stats.HeistAttempts,
		"active_hostage_count": stats.ActiveHostageCount,
		"captured_outlaws":     nonNilMap(stats.CapturedOutlaws),
		"recovery_bounties":    nonNilMap(stats.RecoveryBounties),
		"recovery_card_id":     stats.RecoveryChallengeCardID,
		"recovery_wins":        stats.RecoveryChallengeWins,
		"market_frozen_until":  stats.MarketFrozenUntil,
		"ban_expires":          stats.BanExpires,
		"gloat_banned_until":   stats.GloatBannedUntil,
		"disconnect_streak":    stats.DisconnectStreak,
		"audited_clubs":        nonNilMap(stats.AuditedClubs),
		"rumor_count":          stats.RumorCount,
		"reparations_received": stats.ReparationsReceivedCount,

		// ── Custody maps (cards currently held by another party) ──
		"jailed_cards":       nonNilMap(stats.JailedCards),
		"kidnapped_cards":    nonNilMap(stats.KidnappedCards),
		"held_hostage_cards": nonNilMap(stats.HeldHostageCards),

		// ── Buff + debuff flags ──
		"buffs":                      nonNilMap(stats.Buffs),
		"active_buffs":               nonNilMap(stats.ActiveBuffs),
		"active_item_buffs":          nonNilMap(stats.ActiveItemBuffs),
		"mojo_stabilizer_active":     stats.IsMojoStabilizerActive,
		"has_cyber_jammer":           stats.HasCyberJammer,
		"heist_alarms_jammer_count":  stats.HeistAlarmsJammerCount,
		"has_mutation_insurance":     stats.HasMutationInsurance,
		"arc_net_active":             stats.ArcNetActive,
		"bounty_license_active":      stats.BountyLicenseActive,
		"bounty_license_expires_at":  stats.BountyHunterLicenseExpiresAt,
		"raid_insurance_expires_at":  stats.RaidInsuranceExpiresAt,
		"raid_insurance_claims_left": stats.RaidInsuranceClaimsRemaining,

		// ── Territory + social graph ──
		"sector_tiles":       nonNilMap(stats.SectorTiles),
		"relationships":      nonNilMap(stats.Relationships),
		"moods":              nonNilMap(stats.Moods),
		"preferred_rules":    nonNilMap(stats.PreferredRules),
		"inventory":          nonNilMap(stats.Inventory),
		"last_claimed_yield": nonNilMap(stats.LastClaimedYield),
		"favorite_card_id":   stats.FavoriteCardID,

		// ── Cooldown / timer state ──
		"ghost_protocol_expires_at":   stats.GhostProtocolExpiresAt,
		"last_deep_scan_at":           stats.LastDeepScanAt,
		"district_scanner_expires_at": stats.DistrictScannerExpiresAt,
		"disruptor_cooldown_at":       stats.DisruptorCooldownAt,
		"cloak_disrupted_until":       stats.CloakDisruptedUntil,
		"demotion_warning_at":         stats.DemotionWarningAt,
		"last_salary_payment":         stats.LastSalaryPayment,

		// ── Active assignments ──
		"active_underworld_contract_id": stats.ActiveUnderworldContractID,
		"active_justice_mission_id":     stats.ActiveJusticeMissionID,

		// ── Sustained liquidity (career-tier eligibility) ──
		"liquidity_samples":      nonNilSlice(samples),
		"liquidity_sample_count": len(stats.LiquiditySamples),
		"liquidity_window_min":   stats.LiquidityWindowMin,
		"avg_sustained_micro":    stats.AvgSustainedMicro,

		// ── Forensic record ──
		"mutation_history": nonNilSlice(stats.MutationHistory),

		// ── Playstyle as integer parts-per-million (boundary conversion) ──
		"aggressiveness_ppm":  ppmFromRatio(stats.Aggressiveness),
		"risk_tolerance_ppm":  ppmFromRatio(stats.RiskTolerance),
		"mojo_decay_rate_ppm": ppmFromRatio(stats.MojoDecayRate),
		"playstyle": map[string]interface{}{
			"aggressiveness_ppm":       ppmFromRatio(stats.Playstyle.Aggressiveness),
			"risk_tolerance_ppm":       ppmFromRatio(stats.Playstyle.RiskTolerance),
			"favorite_card_id":         stats.Playstyle.FavoriteCardID,
			"preferred_rules_ppm":      ppmMap(stats.Playstyle.PreferredRules),
			"preferred_card_moods_ppm": ppmMap(stats.Playstyle.PreferredCardMoods),
			"preferred_items_ppm":      ppmMap(stats.Playstyle.PreferredItems),
		},
	})
}

// ppmFromRatio converts a 0.0-1.0 engine ratio into integer parts-per-million.
// Boundary conversion only: results are presented to a client, never stored,
// summed or compared, so the ledger keeps its no-float invariant.
func ppmFromRatio(v float64) uint64 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 1_000_000
	}
	return uint64(v*1_000_000 + 0.5)
}

// ppmMap converts a float-valued weight map into an integer parts-per-million
// map (same boundary-only rule as ppmFromRatio).
func ppmMap(m map[string]float64) map[string]uint64 {
	out := make(map[string]uint64, len(m))
	for k, v := range m {
		out[k] = ppmFromRatio(v)
	}
	return out
}

// nonNilMap guarantees a JSON object ({}), never null, so a reader can never
// mistake "the engine did not export this" for "the wallet has none of these".
func nonNilMap[K comparable, V any](m map[K]V) map[K]V {
	if m == nil {
		return map[K]V{}
	}
	return m
}

// nonNilSlice guarantees a JSON array ([]), never null (same rule as nonNilMap).
func nonNilSlice[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}


// handlePublicStatus provides public-facing statistics for external sites (e.g., Carrd.co).
func (l *Lobby) handlePublicStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	l.mutex.RLock()
	defer l.mutex.RUnlock()

	status := struct {
		FaucetBalance float64   `json:"faucet_balance"`
		Maintenance   bool      `json:"maintenance_mode"`
		ActiveMatches int       `json:"active_matches"`
		TotalPlayers  int       `json:"total_players"`
		Timestamp     time.Time `json:"timestamp"`
	}{
		FaucetBalance: l.faucetBalance,
		Maintenance:   l.maintenanceMode,
		ActiveMatches: l.countUniqueMatchesLocked(), // Consistent load telemetry
		TotalPlayers:  len(l.clients),
		Timestamp:     time.Now(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (l *Lobby) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	// PILLAR 4: High-Fidelity Health Monitoring.
	// This endpoint allows Render's load balancer to verify that the server
	// is not just responsive, but has active connectivity and liquidity.
	l.mutex.RLock()
	voiConfig, ok := l.availableNetworks["Voi Mainnet"]
	balance := l.faucetBalance
	clientsCount := len(l.clients)
	nodeReport := l.getHealthReportLocked()
	l.mutex.RUnlock()

	isHealthy := true
	var errs []string

	// 1. Verify RPC Connectivity (Cycle through all available nodes with LlamaRPC failover)
	rpcResponded := false
	if ok && len(voiConfig.NodeURLs) > 0 {
		for _, url := range voiConfig.NodeURLs {
			client, _ := algod.MakeClient(url, voiConfig.AlgodToken)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			err := client.HealthCheck().Do(ctx)
			cancel()
			if err == nil {
				rpcResponded = true
				break
			}
			log.Printf("[HEALTH] Node unreachable: %s - %v\n", url, err)
		}
		if !rpcResponded {
			isHealthy = false
			errs = append(errs, "rpc_unreachable")
		}
	} else {
		isHealthy = false
		errs = append(errs, "config_missing")
	}

	// 2. Verify Faucet Liquidity (Gas Check)
	if balance < 1.0 {
		isHealthy = false
		errs = append(errs, "low_liquidity")
	}

	status := struct {
		Status      string            `json:"status"`
		Connections int               `json:"connections"`
		Vault       float64           `json:"vault_balance"`
		Nodes       []NodeHealthReport `json:"nodes,omitempty"`
		Errors      []string          `json:"errors,omitempty"`
	}{Status: "ok", Connections: clientsCount, Vault: balance, Nodes: nodeReport}

	if !isHealthy {
		status.Status = "unhealthy"
		status.Errors = errs
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// getHealthReportLocked returns the node health report. Must be called under l.mutex.RLock.
func (l *Lobby) getHealthReportLocked() []NodeHealthReport {
	if l.ledgerClient == nil {
		return nil
	}
	return l.ledgerClient.GetHealthReport()
}

// handleFaucetStatus returns the full faucet state for the Faucet Dashboard.
func (l *Lobby) handleFaucetStatus(w http.ResponseWriter, r *http.Request) {
	l.refreshVaultStateIfStale() // PILLAR 5: refresh $VBV pool state on-demand (throttled), not via background poll
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	// Compute usable balance (mirror of applyDynamicScalingLocked)
	var totalLiabilitiesMicro uint64
	for _, stats := range l.leaderboard {
		totalLiabilitiesMicro += stats.ArenaVouchers
		totalLiabilitiesMicro += stats.BountyHunterBondMicro
	}
	totalLiabilities := float64(totalLiabilitiesMicro) / 1000000.0

	var totalClubReserves float64
	for _, club := range l.clubs {
		totalClubReserves += club.Treasury
	}

	tournamentCommitment := float64(l.tournamentPotBonusMicro) / 1000000.0
	if l.tournament.Active {
		tournamentCommitment += float64(l.tournament.PotMicro) / 1000000.0
	}

	usableBalance := l.faucetBalance - 1.0 - totalLiabilities - totalClubReserves - tournamentCommitment - (float64(l.pendingTournamentPayoutsMicro) / 1000000.0)
	if usableBalance < 0 {
		usableBalance = 0
	}

	ratio := l.RewardRatio
	if ratio <= 0 {
		ratio = 1.0
	}

	// Collect AMM node data.
	//
	// SNAPSHOT UNDER THE MAP'S OWNER LOCK, RELEASE, THEN TAKE EACH NODE'S LOCK. The range here used to
	// read `l.marketNodes` under the LOBBY lock only, while the trade path inserts into THE SAME map
	// under the ROUTER's — so an insert could land inside this walk, and Go answers that with
	// `fatal error: concurrent map iteration and map write`: not an error value, not recoverable, and
	// it kills every connection. `marketNodeRefs` takes and releases the owner lock itself, and the
	// node locks are taken OUTSIDE it, so `tsr.Mu -> node.Mu` is never created.
	ammNodes := []map[string]interface{}{}
	for _, ref := range l.tokenSinkRouter.marketNodeRefs() {
		node := ref.Node
		node.Mu.RLock()
		ammNodes = append(ammNodes, map[string]interface{}{
			"entity_id":            node.EntityID,
			"reserve_balance":      node.ReserveBalance,
			"total_shares_issued":  node.TotalSharesIssued,
			"spot_price":           node.GetSpotPrice(),
		})
		node.Mu.RUnlock()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":               true,
		"faucet_balance_micro":  l.faucetBalanceMicro,
		"vault_address":          l.vaultAddress,
		"reward_asset_id":        l.rewardAssetID,
		"vault_balance_live":     l.vaultBalanceLive,
		"last_vault_sync_unix":   l.lastVaultSync.Unix(),
		"max_faucet_capacity":   l.maxFaucetCapacity,
		"usable_balance":        usableBalance,
		"total_liabilities_micro": totalLiabilitiesMicro,
		"reward_ratio":          ratio,
		"reward_stack":          l.rewardStack,
		"reward_tokens":         l.rewardTokenViewsLocked(), // the registry: what is payable, and why
		"amm_nodes":             ammNodes,
		"market_weather":        map[string]interface{}{},
	})
}

// handleFaucetVaultBalance returns the per-wallet vault balance for the Faucet Dashboard.
func (l *Lobby) handleFaucetVaultBalance(w http.ResponseWriter, r *http.Request) {
	l.refreshVaultStateIfStale() // PILLAR 5: refresh $VBV pool state on-demand (throttled), not via background poll
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	wallet := r.URL.Query().Get("wallet")
	if wallet == "" {
		wallet = r.URL.Query().Get("address")
	}

	var walletMicro uint64
	if wallet != "" {
		walletMicro = l.playerBalances[strings.ToLower(wallet)]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":              true,
		"wallet":               wallet,
		"vault_balance_micro":  walletMicro,
		"faucet_balance_micro": l.faucetBalanceMicro,
		"vault_address":          l.vaultAddress,
		"reward_asset_id":        l.rewardAssetID,
	})
}

// handleFaucetClaim processes a simplified faucet claim grant (placeholder).
func (l *Lobby) handleFaucetClaim(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Wallet string `json:"wallet"`
		Amount uint64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Wallet == "" {
		http.Error(w, "wallet required", http.StatusBadRequest)
		return
	}

	log.Printf("[FAUCET] Claim request for %s (amount %d micro) — placeholder, no on-chain settlement.", req.Wallet, req.Amount)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"status":  "placeholder",
		"amount":  0,
		"message": "Faucet claim is a placeholder. Real settlement via /api/reward.",
	})
}

// handleUnderworldHeists returns available underworld heists (placeholder).
func (l *Lobby) handleUnderworldHeists(w http.ResponseWriter, r *http.Request) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"heists":  []interface{}{},
	})
}

// handleUnderworldKidnaps returns active underworld kidnaps (placeholder).
func (l *Lobby) handleUnderworldKidnaps(w http.ResponseWriter, r *http.Request) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"kidnaps": []interface{}{},
	})
}

// handleLiveEndpoint provides the /live Kubernetes-compatible liveliness probe.
// This endpoint always returns 200 as long as the process is running, allowing
// Kubernetes/Render to detect if the server process is alive regardless of external dependencies.
func (l *Lobby) handleLiveEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now().Format(time.RFC3339),
		"uptime":    time.Since(l.seasonStart).Round(time.Second).String(),
	})
}

// handleReadyEndpoint provides the /ready Kubernetes-compatible readiness probe.
// This endpoint returns 200 only when all critical dependencies are operational,
// including RPC connectivity and faucet liquidity.
func (l *Lobby) handleReadyEndpoint(w http.ResponseWriter, r *http.Request) {
	l.mutex.RLock()
	voiConfig, ok := l.availableNetworks["Voi Mainnet"]
	balance := l.faucetBalance
	nodeReport := l.getHealthReportLocked()
	l.mutex.RUnlock()

	isReady := true
	var notReady []string

	// Check RPC connectivity
	if !ok || len(voiConfig.NodeURLs) == 0 {
		isReady = false
		notReady = append(notReady, "voi_config_missing")
	} else {
		for _, url := range voiConfig.NodeURLs {
			client, _ := algod.MakeClient(url, voiConfig.AlgodToken)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			err := client.HealthCheck().Do(ctx)
			cancel()
			if err == nil {
				break
			}
			notReady = append(notReady, fmt.Sprintf("node_down:%s", url))
			isReady = false
		}
	}

	// Check faucet liquidity
	if balance < 1.0 {
		isReady = false
		notReady = append(notReady, "low_liquidity")
	}

	statusCode := http.StatusOK
	if !isReady {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      map[string]bool{"ready": isReady},
		"nodes":       nodeReport,
		"vault_balance": balance,
		"not_ready":   notReady,
	})
}

func (l *Lobby) handleCardStats(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	card, err := l.getVerifiedCard("", id, "Voi Mainnet")
	if err != nil {
		http.Error(w, "Card verification failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(card)
}

func (l *Lobby) handleGetCardDetails(w http.ResponseWriter, r *http.Request) {
	idsStr := r.URL.Query().Get("ids")
	network := r.URL.Query().Get("network")
	wallet := r.URL.Query().Get("wallet")
	if network == "" {
		network = "Voi Mainnet"
	}

	var ids []int
	for _, s := range strings.Split(idsStr, ",") {
		if id, err := strconv.Atoi(s); err == nil {
			ids = append(ids, id)
		}
	}

	cards, err := l.getVerifiedCards(wallet, ids, network)
	if err != nil {
		http.Error(w, "Metadata retrieval failed", http.StatusInternalServerError)
		return
	}
	// Non-nil so an empty result set encodes as [] rather than null.
	results := make([]ServerCard, 0)
	for _, id := range ids {
		if c, ok := cards[id]; ok {
			results = append(results, c)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (l *Lobby) getVerifiedCard(wallet string, tokenID int, networkName string) (ServerCard, error) {
	cards, err := l.getVerifiedCards(wallet, []int{tokenID}, networkName)
	if err != nil || len(cards) == 0 {
		return ServerCard{}, err
	}
	return cards[tokenID], nil
}

// handleActiveMatches returns a list of ongoing matches for the spectator portals.
func (l *Lobby) handleActiveMatches(w http.ResponseWriter, r *http.Request) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	type matchSummary struct {
		ID         string    `json:"id"`
		P1         string    `json:"p1_id"`
		P2         string    `json:"p2_id"`
		Rating     string    `json:"rating"`
		Territory  string    `json:"territory"`
		Spectators int       `json:"spectator_count"`
		StartTime  time.Time `json:"start_time"`
	}

	active := []matchSummary{} // Ensure empty array instead of null in JSON
	seen := make(map[*MatchState]bool)

	for _, m := range l.matches {
		// PILLAR 4: Broadcasting Accuracy.
		// Only show matches that have been paired (P2ID present) and are actively in combat.
		if seen[m] || m.IsFinished || m.P2ID == "" {
			continue
		}

		// Use P1's ID as the primary Match ID for routing
		active = append(active, matchSummary{
			ID:         m.P1ID,
			P1:         m.P1ID,
			P2:         m.P2ID,
			Rating:     m.MatchRating, // Correctly uses snapshot to survive Sudden Death
			StartTime:  m.StartTime,
			Territory:  m.TerritoryID,
			Spectators: len(m.Spectators),
		})
		seen[m] = true
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":   len(active),
		"matches": active,
	})
}

// handleEnvoiName resolves a Voi wallet address to its Envoi (.voi) human-readable name.
// GET /api/envoi-name?address=XXXX -> {"address":"XXXX","name":"something.voi"}
// Delegates to OracleService.ResolveEnvoiName (Envoi API primary, indexer scan + truncation fallback).
func (l *Lobby) handleEnvoiName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	address := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("address")))
	if address == "" {
		http.Error(w, "address required", http.StatusBadRequest)
		return
	}
	name := l.oracleService.ResolveEnvoiName(l, address)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"address": address, "name": name})
}
