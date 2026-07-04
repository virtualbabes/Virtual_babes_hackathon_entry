//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// CounterfeitService encapsulates logic for creating and detecting counterfeit currency/assets.
// PILLAR 2: Integer Supremacy — all micro-VBV values are uint64.
type CounterfeitService struct{}

// CounterfeitNote represents a forged asset entry in the active counterfeits pool.
type CounterfeitNote struct {
	ID            string    `json:"id"`
	AmountMicro   uint64    `json:"amount_micro"`
	CreatedAt     time.Time `json:"created_at"`
	CreatorWallet string    `json:"creator_wallet"`
	DetectionChance float64 `json:"detection_chance"` // 0.0 to 1.0, increased by Forensic Analyst audits
}

// ActiveCounterfeits is the in-memory pool of undetected counterfeit notes.
var ActiveCounterfeits = make(map[string]CounterfeitNote)

// HandleGenerateCounterfeit allows a Counterfeiter career to create forged assets.
func (s *CounterfeitService) HandleGenerateCounterfeit(l *Lobby, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Wallet  string `json:"wallet"`
		TxID    string `json:"txid"`
		Network string `json:"network"`
		Amount  uint64 `json:"amount"` // Amount in micro-VBV to counterfeit
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Wallet == "" || req.Amount == 0 {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	targetWallet := strings.ToLower(req.Wallet)

	l.mutex.RLock()
	stats, exists := l.leaderboard[targetWallet]
	l.mutex.RUnlock()

	if !exists {
		http.Error(w, "Player not found", http.StatusNotFound)
		return
	}

	// Career check: only Counterfeiter can generate
	if stats.Career != "Counterfeiter" && stats.CareerTier < 3 {
		http.Error(w, "Requires Counterfeiter career or Tier 3+ Underworld status", http.StatusForbidden)
		return
	}

	// Cost: generating counterfeits costs resources (materials, bribes) — 5% of counterfeit value
	generationCostMicro := uint64(float64(req.Amount) * 0.05)

	// Verify player has sufficient balance
	if stats.Credits < generationCostMicro {
		http.Error(w, "Insufficient funds for counterfeit operation", http.StatusPaymentRequired)
		return
	}

	// Check rate limiting for counterfeit attempts (prevent spam)
	l.counterfeitRateLimiterMu.RLock()
	lastGen, hasLastGen := l.counterfeitLastGen[targetWallet]
	l.counterfeitRateLimiterMu.RUnlock()

	if hasLastGen && time.Since(lastGen).Seconds() < 60 {
		http.Error(w, "Counterfeiting cooldown active (60s)", http.StatusTooManyRequests)
		return
	}

	// Generate the counterfeit note with detection chance based on counterfeiter tier
	baseDetectionChance := 0.15 // 15% base risk for Tier 1
	if stats.CareerTier >= 5 {
		baseDetectionChance = 0.08 // Tier 5+ reduces risk
	} else if stats.CareerTier >= 3 {
		baseDetectionChance = 0.12 // Tier 3-4 moderate risk
	}

	noteID := fmt.Sprintf("CN-%s-%d", targetWallet[len(targetWallet)-8:], time.Now().UnixNano())
	note := CounterfeitNote{
		ID:              noteID,
		AmountMicro:     req.Amount,
		CreatedAt:       time.Now(),
		CreatorWallet:   targetWallet,
		DetectionChance: baseDetectionChance,
	}

	l.mutex.Lock()
	l.leaderboard[targetWallet].Credits -= generationCostMicro
	ActiveCounterfeits[noteID] = note
	l.counterfeitLastGen[targetWallet] = time.Now()
	l.mutex.Unlock()

	go l.achievementService.UnlockAchievement(l, targetWallet, "FORGED_CURRENCY")

	// Career XP: Counterfeiter (Underworld #9) gains XP on successful generation
	xpGained := uint64(70) // Base XP for generation event
	if stats.CareerTier >= 5 {
		xpGained = 120 // Tier 5+ bonus XP
	}
	l.TrackCareerXP(targetWallet, "Counterfeiter", xpGained)

	log.Printf("[COUNTERFEIT] Generated note %s: %.2f micro-VBV (detection risk: %.1f%%)",
		noteID, float64(req.Amount)/1000000.0, baseDetectionChance*100)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":           "success",
		"message":          fmt.Sprintf("Counterfeit note %s generated successfully.", noteID),
		"note_id":          noteID,
		"amount_micro":     req.Amount,
		"detection_chance": baseDetectionChance,
		"cost_micro":       generationCostMicro,
		"xp_gained":        xpGained,
	})
}

// HandleDetectCounterfeit allows Justice careers to detect and seize counterfeit notes.
func (s *CounterfeitService) HandleDetectCounterfeit(l *Lobby, w http.ResponseWriter, r *http.Request) {
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

	if !exists {
		http.Error(w, "Player not found", http.StatusNotFound)
		return
	}

	// Career check: only Justice careers (Warden D3, Forensic Analyst D5, Sector Peacekeeper D6) can detect
	validDetectingCareers := map[string]bool{
		"Warden": true, "Forensic Analyst": true, "Sector Peacekeeper": true,
	}
	if !validDetectingCareers[stats.Career] {
		http.Error(w, "Requires Justice career (Warden, Forensic Analyst, or Sector Peacekeeper)", http.StatusForbidden)
		return
	}

	assetID := voiConfig.AssetID
	if assetID == "" {
		assetID = voiConfig.AppID
	}
	verifyNet := "Voi"
	if req.Network == "ALGO" {
		assetID = avoiAssetID
		verifyNet = "Algorand"
	}

	// Verify detection transaction cost
	detectionCostMicro := uint64(float64(stats.Credits) * 0.01) // 1% of credits as detection cost
	if detectionCostMicro < 100 {
		detectionCostMicro = 100 // Minimum detection cost
	}

	verified, _, err := l.verifyBuyInTransaction(verifyNet, req.TxID, detectionCostMicro, assetID, targetWallet, vaultAddr, "COUNTERFEIT_DETECT:")
	if err != nil || !verified {
		log.Printf("[COUNTERFEIT] Detection verification failed for %s. Error: %v\n", targetWallet, err)
		http.Error(w, "Detection payment verification failed", http.StatusPaymentRequired)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Find counterfeit notes associated with this player's network/clubs
	detectedNotes := make([]string, 0)
	detectedAmount := uint64(0)

	for noteID, note := range ActiveCounterfeits {
		// Detection chance modified by Forensic Analyst tier and sector coverage
		noteDetectionChance := note.DetectionChance

		// Forensic Analyst bonus detection (D5 career scaling)
		if stats.Career == "Forensic Analyst" {
			noteDetectionChance *= 0.5 // Reduces detection chance by half (makes it easier to find)
		}

		// Sector Peacekeeper coverage bonus (D6)
		if stats.Career == "Sector Peacekeeper" && stats.SectorTiles > 0 {
			noteDetectionChance *= 0.7 // Additional 30% detection improvement
		}

		if rand.Float64() < noteDetectionChance {
			detectedNotes = append(detectedNotes, noteID)
			detectedAmount += note.AmountMicro

			// Remove from active pool and transfer detected amount to vault
			delete(ActiveCounterfeits, noteID)
			l.leaderboard[note.CreatorWallet].Credits -= note.AmountMicro

			log.Printf("[COUNTERFEIT] DETECTED: Note %s from %s seized (%.2f micro-VBV)",
				noteID, note.CreatorWallet, float64(note.AmountMicro)/1000000.0)
		}
	}

	if len(detectedNotes) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":       "success",
			"message":      "No counterfeit notes detected in target sector.",
			"detection_id": fmt.Sprintf("CD-%s-%d", targetWallet[len(targetWallet)-8:], time.Now().UnixNano()),
		})
		return
	}

	// Career XP: Warden (D3) gains on detection + seizure
	lxps := map[string]uint64{
		"Warden":               25, // Detection XP
		"Forensic Analyst":     60, // Advanced forensic analysis XP
		"Sector Peacekeeper":   20, // Sector sweep XP
	}
	for career, xp := range lxps {
		if stats.Career == career {
			l.TrackCareerXP(targetWallet, career, xp)
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":            "detected",
		"message":           fmt.Sprintf("Seized %d counterfeit note(s) totaling %.2f micro-VBV.", len(detectedNotes), float64(detectedAmount)/1000000.0),
		"detected_notes":    detectedNotes,
		"amount_seized_micro": detectedAmount,
		"detection_id":      fmt.Sprintf("CD-%s-%d", targetWallet[len(targetWallet)-8:], time.Now().UnixNano()),
	})
}

// SeizeCounterfeitNoteLocked forcefully removes a counterfeit note and penalizes the creator.
func (s *CounterfeitService) SeizeCounterfeitNoteLocked(l *Lobby, noteID, seizingAgent string) {
	note, exists := ActiveCounterfeits[noteID]
	if !exists {
		return
	}

	delete(ActiveCounterfeits, noteID)

	// Apply reputation penalty to creator
	creatorStats, exists := l.leaderboard[note.CreatorWallet]
	if exists {
		creatorStats.Reputation -= 50 // Significant reputation hit
		creatorStats.WantedLevel += 10 // Increase wanted level
		l.leaderboard[note.CreatorWallet] = creatorStats

		log.Printf("[COUNTERFEIT] NOTE SEIZED: %s from %s (Reputation -50, Wanted +10)",
			noteID, note.CreatorWallet)
	}

	// Award XP to seizing agent for justice career
	l.TrackCareerXP(seizingAgent, "Warden", 40) // Seizure bonus XP
}

// CleanupExpiredCounterfeits removes counterfeit notes older than 24 hours (they degrade).
func (s *CounterfeitService) CleanupExpiredCounterfeits(l *Lobby) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	now := time.Now()
	expired := make([]string, 0)

	for noteID, note := range ActiveCounterfeits {
		if now.Sub(note.CreatedAt).Hours() > 24 {
			expired = append(expired, noteID)
			delete(ActiveCounterfeits, noteID)

			log.Printf("[COUNTERFEIT] EXPIRED: Note %s degraded naturally.", noteID)
		}
	}

	if len(expired) > 0 {
		log.Printf("[COUNTERFEIT] Cleaned up %d expired notes.", len(expired))
	}
}