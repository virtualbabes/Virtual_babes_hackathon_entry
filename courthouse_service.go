//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// CourthouseService encapsulates logic for legal systems and infamy management.
// PILLAR 5: Stateless Service Design.
type CourthouseService struct{}

// CourthouseFinePerWantedLevelMicro is the fine for ONE Wanted Level point: 100 $VBV.
// INTEGER micro-units, because it is a LEDGER price.
const CourthouseFinePerWantedLevelMicro = 100 * 1_000_000

const (
	// CourthouseFineCareerXP is the Tax Auditor's XP for processing a fine payment — and the BASE the
	// courthouse's rival resolution is priced from at that point.
	CourthouseFineCareerXP = 15
	// CourthousePardonCareerXP is the Tax Auditor's XP for executing a legal pardon — and the base at
	// that point.
	CourthousePardonCareerXP = 30
)

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

	// Cost calculation: 100 $VBV per Wanted Level point — INTEGER ONLY.
	//
	// This was `costBase := float64(stats.WantedLevel * 100)` then `uint64(costBase * 1000000)`: a
	// float multiply on a LEDGER amount, which the Architecture Ledger prohibits. The micro figure is
	// exact integer arithmetic now, and `fine_paid_micro` is what the response serves.
	costMicro := uint64(stats.WantedLevel) * CourthouseFinePerWantedLevelMicro

	assetID := voiConfig.AssetID
	if assetID == "" {
		assetID = voiConfig.AppID
	}
	verifyNet := "Voi"
	if req.Network == "ALGO" {
		assetID = avoiAssetID
		verifyNet = "Algorand"
	}

	// ── UNIFORM TXID GUARD (txid_memo.go) ────────────────────────────────────
	// Reserve the transaction id BEFORE the (slow) chain verification so two
	// concurrent double-submits cannot both pass, and commit it only once the
	// fine has actually been applied. No lobby lock is held across the oracle
	// call: VerifyBuyInTransaction takes l.mutex.RLock() internally.
	claimID, claimReason := l.claimTxID(req.TxID)
	if claimID == "" {
		http.Error(w, claimReason, http.StatusConflict)
		return
	}

	// PILLAR 3: Specific Purpose Verification for courthouse fines
	verified, _, err := l.oracleService.VerifyBuyInTransaction(l, verifyNet, req.TxID, costMicro, assetID, targetWallet, vaultAddr, NotePrefixCourthouseFine)
	if err != nil || !verified {
		l.releaseTxID(claimID)
		log.Printf("[COURTHOUSE] Verification failed for %s. Error: %v\n", targetWallet, err)
		http.Error(w, "Fine payment verification failed or insufficient amount", http.StatusPaymentRequired)
		return
	}

	// Career XP: Tax Auditor (Justice D2) gains XP on fine payment processing
	l.TrackCareerXP(targetWallet, "Tax Auditor", 15)

	// PILLAR 8 Task 5002 — THE COURTHOUSE RIVAL RESOLUTION, at resolution point #1 (the fine reset).
	//
	// BOTH matrices are resolved by ONE owner (`courthouseRivalAwards`) against a LOCK-HELD SNAPSHOT,
	// and applied AFTER the lock is released. This block used to range over l.leaderboard with NO lock
	// held while other goroutines wrote it — a `fatal error: concurrent map iteration and map write`,
	// which is not recoverable, so a fine payment could take the whole server down. The award itself
	// goes through TrackCareerXP, which takes the write lock on its own, so no lock is held here.
	for _, a := range l.courthouseRivalAwards(targetWallet, "Tax Auditor", CourthouseFineCareerXP) {
		l.TrackCareerXP(targetWallet, "Tax Auditor", a.Award.Bonus)
		log.Printf("[COURTHOUSE] RIVAL_BONUS (fine): +%d XP Tax Auditor for %s vs %s [%s] — %s",
			a.Award.Bonus, targetWallet, a.Wallet, strings.Join(a.Award.Layers, "+"), a.Award.Explain)
	}

	// Update Player Stats and Vault balance
	l.mutex.Lock()
	stats.WantedLevel = 0
	l.leaderboard[targetWallet] = stats

	// PILLAR 2: Industrial Loop.
	// Delegate redistribution to the specialized service to ensure Mojo and Taxes 
	// are handled atomically within the reconciliation kernel.
	l.clubService.DistributeCourthouseFineMicroToClubsLocked(l, costMicro)

	l.logAdminAuditLocked("COURTHOUSE_RESET", targetWallet, fmt.Sprintf("Paid %d.%06d $VBV fine", costMicro/1_000_000, costMicro%1_000_000))
	l.mutex.Unlock()

	// Record the consumed transaction ONLY now that the fine has been applied to
	// state. A failure between the apply and here leaves the payment replayable —
	// acceptable and preferable to recording a transaction that changed nothing.
	l.commitTxID(claimID, time.Now())

	go l.achievementService.UnlockAchievement(l, targetWallet, "REHABILITATED")

	// Update all clients with the new social standing
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":           "success",
		"message":          "Wanted level cleared. The Arena recognizes your clean slate.",
		"new_wanted_level": 0,
		// The AUTHORITY is the integer micro figure. The old `fine_paid` float was read by no client
		// (verified repo-wide: neither any *.go nor any Public/js/*.js reads it), so it is replaced
		// rather than kept beside the integer as a second, drifting statement of the same price.
		"fine_paid_micro": costMicro,
	})
}

// ── THE COURTHOUSE RIVAL RESOLUTION (§PILLAR 8 Task 5002) ───────────────────
//
// THE COURTHOUSE IS THE JUSTICE SIDE'S RESOLUTION POINT AGAINST THE CRIMINAL SYSTEM, and it
// resolves BOTH matrices at once:
//
//	CAREER matrix — the Tax Auditor's declared pairs. The Justice Commissioner is their judicial
//	                partner (+7) and the Launderer is their declared OPPOSITE (−10), so the tax arm
//	                is priced both for working with the bench and for standing against the
//	                money-laundering side of the criminal system.
//	PATH   matrix — any peer whose HELD PATH is directly opposed. justice ↔ criminal is the one
//	                DIRECT rivalry (`career_path.go`, +1000 bps) and until now it was SERVED to the
//	                client, pinned by a test, and AWARDED NOWHERE.
//
// Both resolution points (the fine reset and the legal pardon) call the SAME function, so they
// cannot drift apart, and every figure comes from `ResolveRivalXPAward` — never from subtracting
// two return values by hand. The old code read the DEFENDER's share (30% of the base) and compared
// it against the base, i.e. `4 > 15`: false for ever. Its `uint64(rivalXP - 15)` would also have
// underflowed to 18,446,744,073,709,551,605 had the guard ever passed.
//
// The pairing is computed from a LOCK-HELD SNAPSHOT and applied afterwards. The scan ranges over
// l.leaderboard, so doing it outside the lock is a `fatal error: concurrent map iteration and map
// write` — which is NOT recoverable and takes the whole process down, not just the request that
// happened to be running.

// courthouseRivalAward is one resolved pairing, ready to apply.
type courthouseRivalAward struct {
	Wallet string
	Role   string
	Award  RivalXPAward
}

// courthouseRivalAwardsLocked resolves the courthouse's rival awards for one actor.
//
// The CALLER MUST HOLD l.mutex (read OR write): this ranges the leaderboard, so it can never be
// called unlocked. Every value read is COPIED, so a caller may apply the awards after releasing the
// lock without racing a writer.
//
// A peer qualifies when EITHER matrix can price the pairing. That is what makes this a resolution
// point against the criminal system rather than only a synergy window between two justice roles.
func (l *Lobby) courthouseRivalAwardsLocked(actorWallet, actorRole string, baseXP uint64) []courthouseRivalAward {
	actorStats, ok := l.leaderboard[actorWallet]
	if !ok {
		return nil
	}
	// CareerHasRole tolerates a nil CareerXP, so this covers both the current JobRole and every
	// role the wallet has been promoted to.
	if actorStats.JobRole != actorRole && !CareerHasRole(actorStats.CareerXP, actorRole) {
		return nil
	}
	actorPath := PlayerCareerPathOfStats(&actorStats)
	actor := actorStats // a COPY — the resolver must not read live map state

	awards := make([]courthouseRivalAward, 0, 4)
	for peerWallet, peerStats := range l.leaderboard {
		if peerWallet == actorWallet || peerStats.CareerXP == nil {
			continue
		}
		peerRole := peerStats.JobRole
		// The career matrix needs a ROLE on the peer's side; the path matrix needs only the HELD
		// path, which survives an unset JobRole because the choice is stored on the career record.
		careerPair := peerRole != "" && GetRivalPairName(actorRole, peerRole) != ""
		pathDirect := PathRivalryBonusBps(actorPath, PlayerCareerPathOfStats(&peerStats)) > 0
		if !careerPair && !pathDirect {
			continue
		}
		peer := peerStats // a COPY
		award := ResolveRivalXPAward(actorRole, peerRole, baseXP, &actor, &peer)
		if award.Bonus == 0 {
			continue
		}
		awards = append(awards, courthouseRivalAward{Wallet: peerWallet, Role: peerRole, Award: award})
	}
	return awards
}

// courthouseRivalAwards is the UNLOCKED-facing form: it takes the read lock, snapshots, releases,
// and only then resolves. Used by request handlers, which hold no lock. A caller that already holds
// the lock must call courthouseRivalAwardsLocked instead (calling this while holding it would
// self-deadlock).
func (l *Lobby) courthouseRivalAwards(actorWallet, actorRole string, baseXP uint64) []courthouseRivalAward {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.courthouseRivalAwardsLocked(actorWallet, actorRole, baseXP)
}

// payCourthouseRivalAwardsLocked applies the resolved awards through the LOCK-HELD award path, so a
// caller that already holds the write lock pays without reaching for it again.
//
// It returns the total awarded, and it NAMES the matrices that paid in the log — a figure no one
// can attribute to a layer is a figure no one can audit.
func (l *Lobby) payCourthouseRivalAwardsLocked(actorWallet, actorRole, point string, baseXP uint64) uint64 {
	var total uint64
	for _, a := range l.courthouseRivalAwardsLocked(actorWallet, actorRole, baseXP) {
		l.trackCareerXPLocked(actorWallet, actorRole, a.Award.Bonus)
		total += a.Award.Bonus
		log.Printf("[COURTHOUSE] RIVAL_BONUS (%s): +%d XP %s for %s vs %s [%s] — %s",
			point, a.Award.Bonus, actorRole, actorWallet, a.Wallet,
			strings.Join(a.Award.Layers, "+"), a.Award.Explain)
	}
	return total
}

/**
 * ApplyLegalPardonLocked executes the 50% Wanted Level reduction logic.
 * PILLAR 3: Justice Layer.
 *
 * The caller holds the lobby WRITE lock (item_service.go applyItemEffect), so
 * every award here goes through the lock-held helper. Awarding XP through the
 * self-locking TrackCareerXP is what turned a shop item into a process-wide
 * freeze.
 */
func (s *CourthouseService) ApplyLegalPardonLocked(l *Lobby, judgeWallet, targetWallet string, item ShopItem) error {
	// Career XP: Lawyer-Commissioner (Underworld #5) gains XP on legal pardon execution
	l.trackCareerXPLocked(judgeWallet, "Lawyer-Commissioner", 30)

	// PILLAR 8 Task 5002 — resolution point #2 (the legal pardon): the SAME owner as point #1, so the
	// two can never drift apart. This caller HOLDS the write lock (item_service.go applyItemEffect),
	// so it pays through the lock-held path.
	l.payCourthouseRivalAwardsLocked(judgeWallet, "Tax Auditor", "pardon", CourthousePardonCareerXP)

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
