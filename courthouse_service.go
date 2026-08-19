//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// CourthouseService encapsulates logic for legal systems and infamy management.
// PILLAR 5: Stateless Service Design.
type CourthouseService struct{}

// HandleCourthouseReset allows players to pay a $VBV fine to reset their Wanted Level.
// The fine is calculated as 100 $VBV per Wanted Level point.
func (s *CourthouseService) HandleCourthouseReset(l *Lobby, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Wallet  string `json:"wallet"`
		TxID    string `json:"txid"`
		Network string `json:"network"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Wallet == "" || req.TxID == "" {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	targetWallet := strings.ToLower(req.Wallet)

	l.mutex.RLock()
	stats, exists := l.leaderboard[targetWallet]
	voiConfig, voiOk := l.availableNetworks["Voi Mainnet"]
	avoiAssetID := l.avoiAssetID
	vaultAddr := l.vaultAddress
	l.mutex.RUnlock()

	if !voiOk {
		http.Error(w, "Voi network configuration missing", http.StatusInternalServerError)
		return
	}

	if !exists || stats.WantedLevel <= 0 {
		http.Error(w, "No active wanted level to reset", http.StatusBadRequest)
		return
	}

	// Cost calculation: 100 $VBV per Wanted Level point
	costBase := float64(stats.WantedLevel * 100)
	costMicro := uint64(costBase * 1000000)

	assetID := voiConfig.AssetID
	if assetID == "" {
		assetID = voiConfig.AppID
	}
	verifyNet := "Voi"
	if req.Network == "ALGO" {
		assetID = avoiAssetID
		verifyNet = "Algorand"
	}

	// PILLAR 3: Specific Purpose Verification for courthouse fines
	verified, _, err := l.verifyBuyInTransaction(verifyNet, req.TxID, costMicro, assetID, targetWallet, vaultAddr, "COURTHOUSE_FINE:")
	if err != nil || !verified {
		log.Printf("[COURTHOUSE] Verification failed for %s. Error: %v\n", targetWallet, err)
		http.Error(w, "Fine payment verification failed or insufficient amount", http.StatusPaymentRequired)
		return
	}

	// Career XP: Tax Auditor (Justice D2) gains XP on fine payment processing
	l.TrackCareerXP(targetWallet, "Tax Auditor", 15)

	// PILLAR 8 Task 5002: TaxAuditor↔JusticeCommissioner rival pair hook — EvaluateCrossCareerXP at courthouse resolution point #1
	if l.leaderboard[targetWallet] != nil && (l.leaderboard[targetWallet].JobRole == "TaxAuditor" || CareerHasRole(l.leaderboard[targetWallet].CareerXP, "TaxAuditor")) {
		for oppWallet, oppStats := range l.leaderboard {
			if oppWallet == targetWallet {
				continue
			}
			if oppStats.CareerXP != nil && (oppStats.JobRole == "JusticeCommissioner" || CareerHasRole(oppStats.CareerXP, "JusticeCommissioner")) {
				rivalXP, _, isRival := EvaluateCrossCareerXP("TaxAuditor", oppStats.JobRole, 15, l.leaderboard[targetWallet], oppStats)
				if isRival && rivalXP > 15 {
					l.TrackCareerXP(targetWallet, "Tax Auditor", int(rivalXP-15))
					log.Printf("[COURTHOUSE] RIVAL_BONUS: TaxAuditor↔JusticeCommissioner — +%d XP bonus for %s (rival target: %s)", rivalXP-15, targetWallet, oppWallet)
				}
			}
		}
	}

	// Update Player Stats and Vault balance
	l.mutex.Lock()
	stats.WantedLevel = 0
	l.leaderboard[targetWallet] = stats

	// PILLAR 2: Industrial Loop.
	// Delegate redistribution to the specialized service to ensure Mojo and Taxes 
	// are handled atomically within the reconciliation kernel.
	l.clubService.DistributeCourthouseFineMicroToClubsLocked(l, costMicro)

	l.logAdminAuditLocked("COURTHOUSE_RESET", targetWallet, fmt.Sprintf("Paid %.2f $VBV fine", costBase))
	l.mutex.Unlock()

	go l.achievementService.UnlockAchievement(l, targetWallet, "REHABILITATED")

	// Update all clients with the new social standing
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":           "success",
		"message":          "Wanted level cleared. The Arena recognizes your clean slate.",
		"new_wanted_level": 0,
		"fine_paid":        costBase,
	})
}

/**
 * ApplyLegalPardonLocked executes the 50% Wanted Level reduction logic.
 * PILLAR 3: Justice Layer.
 */
func (s *CourthouseService) ApplyLegalPardonLocked(l *Lobby, judgeWallet, targetWallet string, item ShopItem) error {
	// Career XP: Lawyer-Commissioner (Underworld #5) gains XP on legal pardon execution
	l.TrackCareerXP(judgeWallet, "Lawyer-Commissioner", 30)

	// PILLAR 8 Task 5002: TaxAuditor↔JusticeCommissioner rival pair hook — EvaluateCrossCareerXP at courthouse resolution point #2 (legal pardon)
	if l.leaderboard[judgeWallet] != nil && (l.leaderboard[judgeWallet].JobRole == "TaxAuditor" || CareerHasRole(l.leaderboard[judgeWallet].CareerXP, "TaxAuditor")) {
		tStatsPardon, existsP := l.leaderboard[targetWallet]
		if existsP && tStatsPardon.CareerXP != nil && (tStatsPardon.JobRole == "JusticeCommissioner" || CareerHasRole(tStatsPardon.CareerXP, "JusticeCommissioner")) {
			rivalXP, _, isRival := EvaluateCrossCareerXP("TaxAuditor", tStatsPardon.JobRole, 30, l.leaderboard[judgeWallet], &tStatsPardon)
			if isRival && rivalXP > 30 {
				l.TrackCareerXP(judgeWallet, "Tax Auditor", int(rivalXP-30))
				log.Printf("[COURTHOUSE] RIVAL_BONUS: TaxAuditor↔JusticeCommissioner (pardon) — +%d XP bonus for %s vs JusticeCommissioner target %s", rivalXP-30, judgeWallet, targetWallet)
			}
		}
	}

	tStats, exists := l.leaderboard[targetWallet]
	if !exists {
		return fmt.Errorf("target signature not found in sector")
	}

	if tStats.WantedLevel <= 0 {
		return fmt.Errorf("target has no active infamy to pardon")
	}

	reduction := tStats.WantedLevel / 2
	if reduction < 1 { reduction = 1 }
	tStats.WantedLevel -= reduction
	tStats.Reputation = l.CalculateReputation(tStats)
	l.leaderboard[targetWallet] = tStats

	// PILLAR 1: Infrastructure Prestige. Gain Mojo for the organization.
	jStats := l.leaderboard[judgeWallet]
	if jStats.EmployerClubID != "" {
		if club, ok := l.clubs[jStats.EmployerClubID]; ok {
			club.Mojo += item.MojoBonus
			l.achievementService.CheckMojoSurgeAchievementLocked(l, club.ID)
		}
	}

	l.logAdminAuditLocked("LEGAL_PARDON_USED", judgeWallet, fmt.Sprintf("Target: %s, Reduced: %d", targetWallet, reduction))
	return nil
}
