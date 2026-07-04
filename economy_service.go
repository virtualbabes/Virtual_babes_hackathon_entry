//go:build !js && !wasm

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/mnemonic"
	"github.com/algorand/go-algorand-sdk/v2/transaction"
	"github.com/algorand/go-algorand-sdk/v2/types"
)

func (l *Lobby) applyDynamicScaling() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.applyDynamicScalingLocked()
}

// applyDynamicScalingLocked contains the core logic for reward scaling and assumes the mutex is held.
func (l *Lobby) applyDynamicScalingLocked() {
	// Scaling is based on how full the faucet is relative to its target maximum
	if l.maxFaucetCapacity <= 0 {
		return
	}

	// PILLAR 2: Usable Liquidity.
	// The authoritative unreserved liquidity equals the physical vault balance minus the
	// 1.0 unit gas floor, all outstanding virtual reward liabilities (playerBalances),
	// Club treasuries, the currently committed tournament pot, and any reserved
	// pending payouts. This ensures the reward ratio correctly accounts for
	// committed funds during high-concurrency events like tournament finalization.
	var totalLiabilitiesMicro uint64
	for _, bal := range l.playerBalances {
		totalLiabilitiesMicro += bal
	}

	// PILLAR 2: Phase 4 Expansion. Include non-crypto vouchers in total liabilities
	// to ensure reward scaling remains conservative and solvent.
	for _, stats := range l.leaderboard {
		totalLiabilitiesMicro += stats.ArenaVouchers
		for _, bounty := range stats.RecoveryBounties {
			totalLiabilitiesMicro += bounty
		}
		totalLiabilitiesMicro += stats.BountyHunterBondMicro
	}
	totalLiabilities := float64(totalLiabilitiesMicro) / 1000000.0

	totalClubReserves := 0.0
	for _, club := range l.clubs {
		totalClubReserves += club.Treasury
	}

	tournamentCommitment := float64(l.tournamentPotBonusMicro) / 1000000.0 // PILLAR 2: Integer Supremacy
	if l.tournament.Active {
		tournamentCommitment += float64(l.tournament.PotMicro) / 1000000.0 // PILLAR 2: Integer Supremacy
	}

	usableBalance := l.faucetBalance - 1.0 - totalLiabilities - totalClubReserves - tournamentCommitment - (float64(l.pendingTournamentPayoutsMicro) / 1000000.0) // PILLAR 2: Integer Supremacy
	if usableBalance < 0 {
		usableBalance = 0
	}

	ratio := usableBalance / l.maxFaucetCapacity
	if ratio > 1.0 {
		ratio = 1.0
	}
	if ratio < 0.1 {
		ratio = 0.1
	}

	// 1. Scale the primary base reward (for internal tracking/legacy logic)
	l.baseReward = uint64(float64(l.initialBaseReward) * ratio)

	// 2. Iterate through the entire reward stack and scale based on unscaled initial values.
	// We clear the stack first to ensure assets removed from the template are purged.
	l.rewardStack = make(map[string]uint64)
	var rewardSum uint64
	for assetID, initialAmt := range l.initialRewards {
		scaledAmt := uint64(float64(initialAmt) * ratio)
		l.rewardStack[assetID] = scaledAmt
		rewardSum += scaledAmt
	}

	// PILLAR 2: Safety Cap Alignment.
	// Ensure the sum of un-boosted base rewards does not exceed 50% of the MaxSinglePayoutMicro.
	// This reserves headroom for Reputation multipliers, Bounties, and Virtual Balances (Salaries/Heists).
	const rewardSafetyLimit = MaxSinglePayoutMicro / 2
	if rewardSum > rewardSafetyLimit {
		log.Printf("[ECONOMY WARNING] Aggregated base rewards (%d) exceed safety limit (%d). Clamping stack to preserve headroom.\n", rewardSum, rewardSafetyLimit)
		clampRatio := float64(rewardSafetyLimit) / float64(rewardSum)
		for assetID := range l.rewardStack {
			l.rewardStack[assetID] = uint64(float64(l.rewardStack[assetID]) * clampRatio)
		}
		// Sync the legacy baseReward field
		l.baseReward = l.rewardStack[l.rewardAssetID]
	}

	l.RewardRatio = ratio // PILLAR 2: Persist ratio for UI transparency

	log.Printf("[ECONOMY] Dynamic Scaling Applied (Ratio: %.2f). Faucet Capacity: %.2f units.\n", ratio, l.faucetBalance)
}

// saveSeasonMetadataLocked persists the current season state and reward configuration to disk.
// This function assumes the Lobby mutex is already held.
func (l *Lobby) saveSeasonMetadataLocked() {
	// PILLAR 6: Blockchain Persistence.
	// Local file persistence is deprecated. Economy metadata is now
	// part of the authoritative VBT_ECONOMY_SNAPSHOT.
	// We trigger the snapshot logic immediately using the current locked state.
	state := struct {
		Balances     map[string]uint64       `json:"balances"`
		Kidnappings  map[int]KidnapState     `json:"active_kidnappings"`
		MatchHistory map[string]MatchHistory `json:"match_history"`
		Tournament   TournamentState         `json:"tournament"`
		SeasonNum    int                     `json:"season_num"`
		SeasonStart  time.Time               `json:"season_start"`
		MarketNodes  map[string]EntityMarketNode `json:"market_nodes"` // PILLAR 2: AMM State
		Rewards      map[string]uint64       `json:"initial_rewards"`
	}{
		TournamentPotBonusMicro: l.tournamentPotBonusMicro, // PILLAR 2: Integer Supremacy
		Balances:     l.playerBalances,
		Kidnappings:  l.activeKidnappings,
		MatchHistory: l.matchHistory,
		Tournament:   l.tournament,
		SeasonNum:    l.seasonNumber,
		SeasonStart:  l.seasonStart,
		MarketNodes:  make(map[string]EntityMarketNode),
		Rewards:      l.initialRewards,
		TournamentPotBonusMicro: l.tournamentPotBonusMicro, // PILLAR 2: Integer Supremacy
	}

	// Deep copy AMM state for snapshotting
	for id, node := range l.marketNodes {
		if node != nil {
			state.MarketNodes[id] = *node
		}
	}

	// PILLAR 2: Ledger Integrity.
	// Since we are already holding the mutex, we dispatch the snapshot note
	// via the internal helper which is goroutine-safe.
	l.saveBlockchainStateSnapshotLocked("VBT_ECONOMY_SNAPSHOT:", state)
	log.Printf("[ECONOMY] Season metadata (%d) queued for blockchain persistence.\n", l.seasonNumber)
}

func (l *Lobby) sendNoteTx(note string) (string, error) {
	l.mutex.RLock()
	voiConfig, _ := l.availableNetworks["Voi Mainnet"]
	l.mutex.RUnlock()

	pk, _ := mnemonic.ToPrivateKey(os.Getenv("FAUCET_MNEMONIC"))
	faucetAccount, _ := crypto.AccountFromPrivateKey(pk)
	senderAddr, _ := types.DecodeAddress(l.vaultAddress)
	appID, _ := strconv.ParseUint(voiConfig.AppID, 10, 64)

	var lastErr error
	for _, nodeURL := range voiConfig.NodeURLs {
		client, _ := algod.MakeClient(nodeURL, "")
		sp, err := client.SuggestedParams().Do(context.Background())
		if err != nil {
			lastErr = err
			continue
		}

		txn, _ := transaction.MakeApplicationNoOpTx(appID, nil, nil, nil, nil, sp, senderAddr, []byte(note), types.Digest{}, [32]byte{}, types.Address{})
		_, stxn, _ := crypto.SignTransaction(faucetAccount.PrivateKey, txn)
		txid, err := client.SendRawTransaction(stxn).Do(context.Background())
		if err == nil {
			return txid, nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("all nodes failed to dispatch note: %w", lastErr)
}

func (l *Lobby) recordWinOnChain(winnerWallet string, history MatchHistory) {
	log.Printf("[ORACLE] Win Logged: %s vs %s. Payout sequence initiated.\n", winnerWallet, history.Opponent)
}

// recordDNFOnChain persists a match disconnection to the blockchain.
// Metadata includes the leaver, the opponent, and the tournament context for reconstruction.
func (l *Lobby) recordDNFOnChain(wallet, opponent, tid string) {
	if wallet == "" {
		return
	}

	l.mutex.RLock()
	voiConfig, _ := l.availableNetworks["Voi Mainnet"]
	l.mutex.RUnlock()
	client, _ := algod.MakeClient(voiConfig.NodeURLs[0], "")
	pk, _ := mnemonic.ToPrivateKey(os.Getenv("FAUCET_MNEMONIC"))
	faucetAccount, _ := crypto.AccountFromPrivateKey(pk)
	sp, _ := client.SuggestedParams().Do(context.Background())
	senderAddr, _ := types.DecodeAddress(l.vaultAddress)

	meta := map[string]string{"leaver": wallet, "opp": opponent, "tid": tid}
	jsonData, _ := json.Marshal(meta)
	dnfNote := []byte(fmt.Sprintf("VBT_DNF:%s", string(jsonData)))

	appID, _ := strconv.ParseUint(voiConfig.AppID, 10, 64)
	// PILLAR 4: Historical Persistence. Send NoOp to vault with leaver context.
	txn, _ := transaction.MakeApplicationNoOpTx(appID, nil, []string{wallet}, nil, nil, sp, senderAddr, dnfNote, types.Digest{}, [32]byte{}, types.Address{})
	_, stxn, _ := crypto.SignTransaction(faucetAccount.PrivateKey, txn)
	client.SendRawTransaction(stxn).Do(context.Background())
}

func logWinAudit(recipient, network, txid, groupID string, amount uint64, history MatchHistory) {
	entry := struct {
		Timestamp string       `json:"timestamp"`
		Recipient string       `json:"recipient"`
		Network   string       `json:"network"`
		TxID      string       `json:"txid"`
		GroupID   string       `json:"group_id"`
		Amount    string       `json:"amount"`
		History   MatchHistory `json:"history"`
	}{
		Timestamp: time.Now().Format(time.RFC3339), Recipient: recipient, Network: network,
		TxID: txid, GroupID: groupID, Amount: fmt.Sprintf("%.1f $VBV", float64(amount)/1000000.0), History: history,
	}
	b, _ := json.Marshal(entry)
	f, _ := os.OpenFile("win_audit.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	f.Write(append(b, '\n'))
}

// CalculateReputation computes a player's social standing based on their performance and infamy.
func (l *Lobby) CalculateReputation(stats PlayerStats) int {
	// 1. Base Performance Score (Wins vs DNFs/Streak)
	// PILLAR 4: Competitive Balance.
	// Implemented diminishing returns for raw wins to prevent "grinding" from overwhelming
	// social markers (Achievements/Mojo) during long seasons or extreme simulations.
	winRep := 0
	if stats.Wins <= 100 {
		winRep = stats.Wins * 100
	} else if stats.Wins <= 500 {
		winRep = 10000 + (stats.Wins-100)*25
	} else {
		winRep = 20000 + (stats.Wins-500)*5
	}

	// PILLAR 4: Social Standing Integration.
	// Direct Mojo contribution ensures that personal social growth is a core component of Standing.
	// 1 Mojo point = 10 Reputation points (matching standard cosmetic scaling).
	rep := winRep + (stats.GetEffectiveMojo() * 10) - (stats.DNFs * 50) - (stats.DisconnectStreak * 15)

	// 2. Infamy Penalty
	rep -= (stats.WantedLevel * 20)

	// 2.1 Asset Impoundment Penalty: Cards in sector custody reduce social reach
	rep -= (len(stats.JailedCards) * 25)

	// 3. Achievement Weighting
	for _, id := range stats.Achievements {
		bonus := 50 // Standard achievement
		switch id {
		case "SOCIAL_TITAN":
			bonus = 300 // Seasonal peak mojo milestone
		case "TOURNAMENT_CHAMPION":
			bonus = 200 // Bracket dominance milestone
		case "GOVERNOR", "PATRON_OF_THE_ARTS":
			bonus = 250 // Regional influence milestone
		case "TREASURY_RECOVERY":
			bonus = 150 // Organizational resilience milestone
		case "ARENA_LEGEND":
			bonus = 150 // Career veteran milestone
		case "MOJO_SURGE", "PERFECT_GAME":
			bonus = 125 // High-impact performance milestones
		case "CORPORATE_ESPIONAGE", "ART_COLLECTOR":
			bonus = 100 // Specialized elite-tier achievements
		case "HEIST_SABOTEUR":
			bonus = 75 // Stealth and tactical mastery
		case "REHABILITATED", "OUTLAW_SLAYER", "EXECUTIVE_PAY", "CAREER_START":
			bonus = 75 // Specialized mid-tier achievements
		}
		rep += bonus
	}

	// 4. Marketability Multiplier (Aggressiveness & Risk rewarded as "Marketable Traits")
	// Instead of a flat bonus, playstyle now acts as a multiplier to scale with player performance.
	// Aggressiveness: Max +15%, Risk Tolerance: Max +10% (Total potential: 1.25x)

	// PILLAR 1: Corporate Synergy.
	// A player's personal brand (Playstyle) is amplified by the prestige (Mojo) of their employer.
	brandAmper := 1.0
	if stats.EmployerClubID != "" {
		if club, exists := l.clubs[stats.EmployerClubID]; exists {
			// PILLAR 1: Alliance Integration.
			// The synergy boost now accounts for the combined prestige of the coalition.
			// Allied partners contribute 50% of their Mojo to the employee's brand amplification.
			effMojo := float64(club.Mojo)
			if club.AlliedClubID != "" {
				if ally, ok := l.clubs[club.AlliedClubID]; ok {
					effMojo += float64(ally.Mojo) * 0.5
				}
			}

			brandAmper = 1.0 + (effMojo / 4000.0) // Max +25% synergy boost
			if brandAmper > 1.25 {
				brandAmper = 1.25
			}
		}
	}

	marketabilityMult := (1.0 + (stats.Playstyle.Aggressiveness * 0.15) + (stats.Playstyle.RiskTolerance * 0.10)) * brandAmper
	rep = int(float64(rep) * marketabilityMult)

	// 5. Employment Multiplier (Social Trust from high-Mojo Clubs)
	if stats.EmployerClubID != "" {
		// Note: Mutex expected to be held by caller (e.g. updateLeaderboard)
		club, exists := l.clubs[stats.EmployerClubID]
		if exists { // Check if the club actually exists
			// PILLAR 1: Alliance Integration.
			// The social trust multiplier now accounts for the prestige of the allied organization.
			effMojo := float64(club.Mojo)
			if club.AlliedClubID != "" {
				if ally, ok := l.clubs[club.AlliedClubID]; ok {
					effMojo += float64(ally.Mojo) * 0.5
				}
			}

			// Multiplier scales with effective Mojo: 1.0 to 1.5 (at 1000 effective Mojo)
			multiplier := 1.0 + (effMojo / 2000.0)
			if multiplier > 1.5 {
				multiplier = 1.5
			}

			// PILLAR 3: Sabotage Penalty.
			// Organizations with compromised defenses suffer a loss in social trust.
			// Working for a club that fails to secure its borders is a mark of professional shame.
			if expiry, active := club.BuffExpirations["SABOTAGE"]; active && time.Now().Before(expiry) {
				multiplier -= 0.20 // -20% standing penalty for working for a compromised club
			}

			// PILLAR 1: Regional Governor Administrative Bonus.
			// Club owners managing a region (2+ districts) receive a +10% bonus to reflect
			// their superior administrative influence.
			if strings.EqualFold(club.OwnerWallet, stats.Wallet) && l.clubService.IsClubRegionalLocked(l, club) {
				multiplier += 0.10
			}

			rep = int(float64(rep) * multiplier)
		}
	}

	// 6. Cosmetic Prestige Multiplier (Faceplates)
	// For Standard players, faceplates provide a flat Reputation boost to aid their climb.
	// For Diamond Tier (Rep >= 500) players, cosmetics provide a "Prestige Multiplier".
	if stats.EquippedFaceplate != "" {
		if fp, exists := FaceplateRegistry[stats.EquippedFaceplate]; exists {
			if rep >= 500 { // This is a direct check on 'rep', not stats.GetEffectiveMojo
				// Diamond Tier: 1 Mojo point = 0.5% prestige multiplier (Max +25% for Governor)
				prestigeMult := 1.0 + (float64(fp.MojoBonus) * 0.005)
				rep = int(float64(rep) * prestigeMult)
			} else {
				// Standard: Additive bonus (1 Mojo = 10 Reputation points)
				rep += (fp.MojoBonus * 10)
			}
		} // This is a direct check on 'rep', not stats.GetEffectiveMojo
	}

	// 7. Spreader Multiplier (Market Manipulation Reward)
	// Active participants in the Rumor Mill gain Standing for their social influence.
	// Hardening: Changed from a multiplier to an additive bonus to ensure players with
	// zero wins still gain visible "Standing" from rumor activity.
	if stats.RumorCount > 0 {
		spreaderBonus := int(float64(stats.RumorCount) * 10) // 10 reputation points per rumor spread
		if spreaderBonus > 100 {                             // Cap the bonus to prevent excessive reputation from rumors alone
			spreaderBonus = 100
		}
		rep += spreaderBonus
	}

	// 8. Intelligence Bonus (Corporate Espionage)
	// Players are rewarded for gathering economic intelligence on rival organizations.
	// This rewards the active use of Cyber-Audits even before the achievement is unlocked.
	if len(stats.AuditedClubs) > 0 {
		rivalAudits := 0
		for clubID := range stats.AuditedClubs {
			auditedClub, exists := l.clubs[clubID]
			if !exists {
				continue
			}

			// PILLAR 3: Espionage Integrity.
			// Do not reward intelligence gathered on your own organization or its allies.
			if l.isPlayerAffiliatedWithClubLocked(stats.Wallet, auditedClub) {
				continue
			}
			rivalAudits++
		}

		auditBonus := rivalAudits * 20 // 20 reputation points per unique rival club audited
		if auditBonus > 100 {          // Cap at 5 unique clubs (matching achievement threshold)
			auditBonus = 100
		}
		rep += auditBonus
	}

	// 9. Capital Presence Multiplier (Pillar 1)
	// Members of the organization controlling the Arena Center receive a 1.25x prestige
	// boost to reflect their proximity to the sector's administrative core.
	if stats.EmployerClubID != "" {
		capitalClub := l.getClubByTerritoryID("arena_center")
		if capitalClub != nil && capitalClub.ID == stats.EmployerClubID {
			rep = int(float64(rep) * 1.25)
		}
	}

	// PILLAR 3: Career Role Weighting. Apply Hegemony path multiplier.
	reputationWeighting := float64(l.playerService.GetReputationWeighting(stats.JobRole)) / 100.0
	rep = int(float64(rep) * reputationWeighting)

	// 10. Security Bonus (Pillar 1)
	// Regional Governors who have received 5 or more reparations (sabotage surcharges)
	// this session receive a 1.1x 'Security Bonus' to reflect organizational resilience.
	if stats.ReparationsReceivedCount >= 5 && stats.EmployerClubID != "" {
		if club, exists := l.clubs[stats.EmployerClubID]; exists && strings.EqualFold(club.OwnerWallet, stats.Wallet) {
			if l.clubService.IsClubRegionalLocked(l, club) {
				rep = int(float64(rep) * 1.1)
			}
		}
	}

	// PILLAR 1: House Authority.
	// The House Account (Vault) receives a 1.5x multiplier to its total Reputation
	// to ensure it maintains a top-tier rank as the sector's primary liquidity hub.
	if l.vaultAddress != "" && strings.EqualFold(stats.Wallet, l.vaultAddress) {
		rep = int(float64(rep) * 1.5)
	}

	if rep < 0 {
		return 0
	}
	return rep
}

// ApplyBountyHunterTaxLocked calculates and logs the 5% Justice Faction tax.
// PILLAR 1: Industrial Loop.
func (l *Lobby) ApplyBountyHunterTaxLocked(bountyMicro uint64) uint64 {
	if bountyMicro == 0 {
		return 0
	}
	// PILLAR 2: Integer Supremacy.
	taxMicro := (bountyMicro * 5) / 100
	netMicro := bountyMicro - taxMicro

	l.logAdminAuditLocked("BOUNTY_TAX_COLLECTED", "JUSTICE_POOL", fmt.Sprintf("Amount: %.2f $VBV", float64(taxMicro)/1000000.0))
	
	// Taxed funds remain in l.faucetBalanceMicro as a physical asset but are 
	// removed from the virtual reward liability for this match result.
	return netMicro
}

// handleSetDistrictTax allows Regional Governors to adjust the localized sales tax for their territories.
// PILLAR 1: Political Influence.
func (l *Lobby) handleSetDistrictTax(env *Envelope) {
	var data struct {
		TerritoryID string  `json:"territory_id"`
		NewRate     float64 `json:"new_rate"` // e.g., 0.08 for 8%
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	wallet, ok := l.wallets[env.FromID]
	if !ok {
		return
	}

	// 1. Identify Authority: Does the club owning this district have Governor status?
	owningClub := l.getClubByTerritoryID(data.TerritoryID)
	if owningClub == nil || !strings.EqualFold(owningClub.OwnerWallet, wallet) || !l.clubService.IsClubRegionalLocked(l, owningClub) {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ <b>POLITICAL ERROR:</b> You must be the Regional Governor of this district to adjust taxes."}`)})
		return
	}

	// 2. Regulatory Window: Enforce a 0% to 20% tax ceiling.
	if data.NewRate < 0 || data.NewRate > 0.20 {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ <b>POLITICAL ERROR:</b> District tax must be between 0% and 20%."}`)})
		return
	}

	// 2.5 Policy Shift: Calculate 1% 'Governor Surcharge' (PILLAR 1).
	// Governors pay a fee from their organization's treasury to the Arena Center owner
	// to enact localized tax policy changes.
	
	numericID, _ := strconv.ParseUint(strings.TrimPrefix(owningClub.ID, "CLUB-"), 10, 64)
	var surchargeMicro uint64

	// PILLAR 2: Authoritative Treasury Deduction.
	if l.tokenSinkRouter != nil {
		l.tokenSinkRouter.Mu.Lock()
		if node, ok := l.tokenSinkRouter.ActiveClubs[numericID]; ok {
			surchargeMicro = uint64(float64(node.TreasuryBalance) * 0.01 + 0.5)
			if node.TreasuryBalance >= surchargeMicro {
				node.TreasuryBalance -= surchargeMicro
			} else {
				surchargeMicro = node.TreasuryBalance
				node.TreasuryBalance = 0
			}
			// Sync the UI float from the authoritative integer node
			owningClub.Treasury = float64(node.TreasuryBalance) / 1000000.0
		}
		l.tokenSinkRouter.Mu.Unlock()
	}

	if surchargeMicro > 0 {
		arenaCenterClub := l.getClubByTerritoryID("arena_center")
		matrix := RevenueSplitMatrix{FaucetShare: 1.0, ClubShare: 0.0, GovernanceShare: 0.0}
		targetClubID := uint64(0)

		if arenaCenterClub != nil {
			matrix = RevenueSplitMatrix{FaucetShare: 0.0, ClubShare: 1.0, GovernanceShare: 0.0}
			targetClubID, _ = strconv.ParseUint(strings.TrimPrefix(arenaCenterClub.ID, "CLUB-"), 10, 64)
			arenaCenterClub.LastActivity = time.Now()
			l.GovernorSurchargeTotal += surchargeMicro
		}

		// PILLAR 2: Industrial Loop (Token-Sink Router migration).
		if l.tokenSinkRouter != nil {
			_ = l.tokenSinkRouter.RouteCriminalTax("GOV_SURCHARGE", surchargeMicro, matrix, targetClubID, "")

			// Sync recipients float if it's a club
			if targetClubID != 0 && arenaCenterClub != nil {
				if node, ok := l.tokenSinkRouter.ActiveClubs[targetClubID]; ok {
					arenaCenterClub.Treasury = float64(node.TreasuryBalance) / 1000000.0
				}
			}

			l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0
			l.applyDynamicScalingLocked()
		}
	}

	// 3. Update the TokenSinkRouter registry
	if l.tokenSinkRouter != nil {
		l.tokenSinkRouter.Mu.Lock()
		if metric, exists := l.tokenSinkRouter.RegionalDistricts[data.TerritoryID]; exists {
			metric.CustomTaxRate = data.NewRate
			l.tokenSinkRouter.RegionalDistricts[data.TerritoryID] = metric
		} else {
			// Initialize metric if it doesn't exist for this district yet
			l.tokenSinkRouter.RegionalDistricts[data.TerritoryID] = &RegionalGovernanceMetric{
				GovernorAddress: wallet,
				CustomTaxRate:   data.NewRate,
			}
		}
		l.tokenSinkRouter.Mu.Unlock()
	}

	l.logAdminAuditLocked("DISTRICT_TAX_ADJUSTED", wallet, fmt.Sprintf("District: %s, New Rate: %.1f%%", data.TerritoryID, data.NewRate*100))

	// 4. Public Proclamation: Notify the lobby of the policy change.
	proclamation := fmt.Sprintf("🏛️ <b>GOVERNANCE UPDATE:</b> Governor %s has set the sales tax for %s to <b>%.1f%%</b>.",
		template.HTMLEscapeString(l.ResolveEnvoiName(wallet)),
		strings.ReplaceAll(strings.ToUpper(data.TerritoryID), "_", " "),
		data.NewRate*100)

	// PILLAR 1: High-Intensity Policy Alert.
	if data.NewRate > 0.10 {
		proclamation = fmt.Sprintf("⚖️ <b>TAX POLICY ALERT:</b> Governor %s has implemented a high-intensity tax of <b>%.1f%%</b> in %s!",
			template.HTMLEscapeString(l.ResolveEnvoiName(wallet)),
			data.NewRate*100,
			strings.ReplaceAll(strings.ToUpper(data.TerritoryID), "_", " "))
	}

	payload, _ := json.Marshal(map[string]string{"text": proclamation})
	l.broadcast <- jsonListEnvelope("chat", payload)

	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"✅ <b>POLICY UPDATED:</b> Localized sales tax is now active."}`)})

	// Sync UI
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// CalculateSabotageCostLocked determines the dynamic cost of initiating a sabotage protocol.
// PILLAR 1: Political Influence (Reparation Surcharge).
func (l *Lobby) CalculateSabotageCostLocked(targetClub *Club) uint64 {
	// Base: 1,000 $VBV. Surcharge: 500 $VBV.
	const baseSabotageCostMicro = 1000 * 1000000
	const infiltrationSurchargeMicro = 500 * 1000000
	totalBaseMicro := baseSabotageCostMicro + infiltrationSurchargeMicro

	if targetClub == nil {
		return totalBaseMicro
	}

	// PILLAR 1: Reparation Surcharge.
	// Increase cost by 10% for every 5 reparations the owner has received.
	ownerWallet := strings.ToLower(targetClub.OwnerWallet)
	ownerStats, exists := l.leaderboard[ownerWallet]
	if !exists || ownerStats.ReparationsReceivedCount < 5 {
		return totalBaseMicro
	}

	// PILLAR 2: Integer Supremacy. 
	// Calculate surcharge: 10% increase for every 5 reparations. (5-9 -> 1.1x, 10-14 -> 1.2x)
	multiplierPercent := uint64(100 + (ownerStats.ReparationsReceivedCount/5)*10)
	finalCostMicro := (totalBaseMicro * multiplierPercent) / 100

	return finalCostMicro
}

/**
 * HandlePurchaseRaidInsurance allows a Hostage Host to secure a 24h protection buffer.
 * PILLAR 1: Industrial Loop (VBV Sink).
 */
func (l *Lobby) HandlePurchaseRaidInsurance(env *Envelope) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	wallet, ok := l.wallets[env.FromID]
	if !ok { return }
	stats := l.leaderboard[wallet]

	if stats.JobRole != "Hostage Host" {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Access Denied: Raid Insurance restricted to 'Hostage Host' career path."}`)})
		return
	}

	const insuranceCost = 3000 * 1000000
	if l.playerBalances[wallet] < insuranceCost {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Access Denied: Insufficient rewards for premium (3,000 $VBV required)."}`)})
		return
	}

	l.playerBalances[wallet] -= insuranceCost
	l.faucetBalanceMicro += insuranceCost
	l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0

	stats.RaidInsuranceExpiresAt = time.Now().Add(24 * time.Hour)
	stats.RaidInsuranceClaimsRemaining = 1
	l.leaderboard[wallet] = stats

	l.logAdminAuditLocked("RAID_INSURANCE_PURCHASED", wallet, "Premium Paid (3,000 $VBV)")
	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"🛡️ <b>INSURANCE ACTIVE:</b> One successful AOS Raid will be blocked for 24 hours."}`)})
	l.applyDynamicScalingLocked()
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

/**
 * HandlePurchaseBountyHunterBond allows a player to lock a 1,000 $VBV deposit.
 * PILLAR 1: Industrial Loop (Locked Capital).
 */
func (l *Lobby) HandlePurchaseBountyHunterBond(env *Envelope) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	wallet, ok := l.wallets[env.FromID]
	if !ok { return }
	stats := l.leaderboard[wallet]

	const bondCost = 1000 * 1000000
	if stats.BountyHunterBondMicro > 0 {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"⚠️ <b>BOND ACTIVE:</b> You already have a security deposit in the Justice pool."}`)})
		return
	}

	if l.playerBalances[wallet] < bondCost {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Access Denied: Insufficient rewards for bond (1,000 $VBV required)."}`)})
		return
	}

	// Execute Liability Shift: Liquid -> Locked
	l.playerBalances[wallet] -= bondCost
	stats.BountyHunterBondMicro = bondCost
	l.leaderboard[wallet] = stats

	l.logAdminAuditLocked("BOND_PURCHASED", wallet, "Bounty Hunter Bond (1,000 $VBV)")
	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"⚖️ <b>BOND SECURED:</b> 1,000 $VBV deposit locked. High-tier Justice missions unlocked."}`)})
	
	l.applyDynamicScalingLocked()
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

/**
 * HandleRefundBountyHunterBond allows a clean player to retrieve their deposit.
 * PILLAR 1: Industrial Loop (Capital Retrieval).
 */
func (l *Lobby) HandleRefundBountyHunterBond(env *Envelope) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	wallet, ok := l.wallets[env.FromID]
	if !ok { return }
	stats := l.leaderboard[wallet]

	if stats.BountyHunterBondMicro == 0 {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Refund Failed: No active bond found."}`)})
		return
	}

	// "Clean Retirement" check: Wanted Level must be low
	if stats.WantedLevel > 2 {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Refund Denied: Signature is currently flagged for infamy. Seek rehabilitation at the Courthouse first."}`)})
		return
	}

	// Check for active missions
	if stats.ActiveJusticeMissionID != "" {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Refund Denied: Active mission dossier in progress. Complete or abort it first."}`)})
		return
	}

	bondAmount := stats.BountyHunterBondMicro
	
	// Execute Liability Shift: Locked -> Liquid
	stats.BountyHunterBondMicro = 0
	l.playerBalances[wallet] += bondAmount
	l.leaderboard[wallet] = stats

	l.logAdminAuditLocked("BOND_REFUNDED", wallet, fmt.Sprintf("Refunded %.2f $VBV", float64(bondAmount)/1000000.0))
	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"⚖️ <b>BOND REFUNDED:</b> Security deposit returned to your liquid rewards. High-tier status revoked."}`)})
	
	l.applyDynamicScalingLocked()
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}
