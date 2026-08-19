//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

// ============================================================================
// PILLAR 7-D: AI AUTONOMOUS ECONOMY — AICitizenEngine
// Vision Alignment: Lines 507-545 "AI should work, trade, learn, remember, compete."
// AI citizens are first-class economic participants with careers, treasuries, and behaviors.
// ============================================================================

// AICitizen represents an autonomous AI entity in the civilization.
type AICitizen struct {
	Wallet        string    `json:"wallet"`         // Authoritative wallet address
	Name          string    `json:"name"`           // Display name (e.g., "Shadow_01")
	Career        string    `json:"career"`         // Current career role (from underworld_contracts templates)
	Tier          int       `json:"tier"`           // Career tier (0=Peon → 6=Boss)
	Treasury      uint64    `json:"treasury"`       // Current treasury balance in micro-units
	SavingsRate   float64   `json:"savings_rate"`   // Portion of earnings retained (career-dependent)
	InvestmentThresh uint64  `json:"investment_thresh"` // Treasury threshold to trigger entity market investment
	BusinessCount int        `json:"business_count"` // Number of NPC businesses spawned
	LastAction    time.Time `json:"last_action"`    // Timestamp of last behavioral tick
	ActionCooldown time.Duration `json:"-"`           // Minimum interval between actions (5min default)
	LearningXP    uint64    `json:"learning_xp"`    // XP from observing/interacting with humans
	Reputation     int       `json:"reputation"`     // Persistent reputation score (-100 to +100)
}

// AICitizenEngine manages the pool of autonomous AI citizens.
type AICitizenEngine struct {
	citizens   map[string]*AICitizen // wallet -> citizen pointer
	lobby      *Lobby                  // Reference for economy/tournament access
	mu         sync.RWMutex           // Thread-safe citizen management
	tickInterval time.Duration        // Behavioral tick interval (1min default)
	stopChan   chan struct{}          // Channel to stop the behavioral loop
}

// AI business types that high-tier citizens can spawn.
const (
	AIBusinessShop      = "AI_Neighborhood_Shop"       // Small retail shop creating employment
	AIBusinessLaunderer = "AI_Laundromat_Service"     // Money laundering service for underworld careers
	AIBusinessGallery   = "AI_Art_Gallery_Venue"      // Art auction venue for creator economy
	AIBusinessClub      = "AI_Underground_Club"       // Social hub creating employment opportunities
)

// AI career savings rates (career-dependent treasury behavior).
var AICareerSavingsRates = map[string]float64{
	"HeistPlanner":     0.3, // Invests heavily in entity markets
	"Launderer":        0.5, // Hoards for operational security
	"Fence":            0.4, // Saves for inventory purchases
	"Smuggler":         0.35, // Reinvests in logistics
	"BountyHunter":     0.6, // High savings from lucrative captures
	"JusticeRecruiter": 0.25, // Spends on recruitment campaigns
	"ForensicAnalyst":  0.45, // Saves for equipment upgrades
	"ArcNetOperative":  0.3, // Invests in cyber infrastructure
	"BountyHunter":     0.6, // High savings from lucrative captures
	"Warden":           0.7, // Maximum hoarding for security
}

// AI career investment thresholds (treasury amount to trigger entity market entry).
var AICareerInvestmentThresholds = map[string]uint64{
	"HeistPlanner":     5_000_000_000,    // 5M micro-units
	"Launderer":        8_000_000_000,    // 8M micro-units
	"Fence":            6_000_000_000,    // 6M micro-units
	"Smuggler":         4_000_000_000,    // 4M micro-units
	"BountyHunter":     10_000_000_000,   // 10M micro-units (high threshold)
	"JusticeRecruiter": 3_000_000_000,    // 3M micro-units
	"ForensicAnalyst":  5_000_000_000,    // 5M micro-units
	"ArcNetOperative":  4_000_000_000,    // 4M micro-units
}

// NewAICitizenEngine creates a new AI citizen engine with default configuration.
func NewAICitizenEngine() *AICitizenEngine {
	return &AICitizenEngine{
		citizens:     make(map[string]*AICitizen),
		tickInterval: 1 * time.Minute, // Every minute for responsive behavior
		stopChan:     make(chan struct{}),
	}
}

// SpawnAI creates a new AI citizen with randomized career from underworld_contracts templates.
func (ace *AICitizenEngine) SpawnAI(lobby *Lobby) (*AICitizen, error) {
	ace.mu.Lock()
	defer ace.mu.Unlock()

	if lobby == nil {
		return nil, fmt.Errorf("cannot spawn AI: nil lobby reference")
	}
	ace.lobby = lobby

	// Select random career from available underworld careers (matching CONTRACT templates).
	careers := []string{
		"HeistPlanner", "Launderer", "Fence", "Smuggler",
		"BountyHunter", "JusticeRecruiter", "ForensicAnalyst", "ArcNetOperative",
		"Warden", "SectorPeacekeeper", "HostageHost", "LawyerCommissioner",
	}
	career := careers[rand.Intn(len(careers))]

	// Generate unique AI name with career prefix.
	tier := CareerTierPeon // Start as Peon (tier 0)
	name := fmt.Sprintf("%s_%04d", career, rand.Intn(9999))

	// Create wallet address for the AI citizen (server-generated deterministic seed).
	wallet := generateAIVault(walletSeed(name))

	savingsRate := AICareerSavingsRates[career]
	if savingsRate == 0 {
		savingsRate = 0.3 // Default 30% savings rate
	}

	investmentThresh := AICareerInvestmentThresholds[career]
	if investmentThresh == 0 {
		investmentThresh = 5_000_000_000 // Default 5M threshold
	}

	citizen := &AICitizen{
		Wallet:           wallet,
		Name:             name,
		Career:           career,
		Tier:             tier,
		Treasury:         uint64(1_000_000), // Starting treasury: 1M micro-units (seed capital)
		SavingsRate:      savingsRate,
		InvestmentThresh: investmentThresh,
		BusinessCount:    0,
		LastAction:       time.Now(),
		ActionCooldown:   5 * time.Minute,
		LearningXP:       0,
		Reputation:       rand.Intn(21) - 10, // Random reputation between -10 and +10
	}

	ace.citizens[wallet] = citizen

	// Broadcast AI spawn event to all connected clients.
	if lobby.broadcast != nil {
		envelope := Envelope{
			Type: "ai_citizen_spawned",
			Payload: json.RawMessage(fmt.Sprintf(`{
				"wallet":"%s","name":"%s","career":"%s","tier":%d,"treasury":%d
			}`, wallet, name, career, tier, citizen.Treasury)),
		}
		lobby.broadcast <- envelope
	}

	log.Printf("[AICitizenEngine] Spawned AI: %s (wallet=%s) as %s at Tier %d", name, wallet, career, tier)
	return citizen, nil
}

// generateAIVault creates a deterministic server-generated wallet seed for an AI citizen.
func generateAIVault(seed string) string {
	// In production: derive from cryptographic seed + AI identity hash.
	// For simulation: use simple hex encoding of name-based seed.
	hash := 0
	for _, c := range seed {
		hash = hash*31 + int(c)
	}
	return fmt.Sprintf("0xai%040d", uint64(hash)&((1<<256)-1))[:42] // Simulated Ethereum address format
}

// GetCitizen returns an AI citizen by wallet address.
func (ace *AICitizenEngine) GetCitizen(wallet string) (*AICitizen, bool) {
	ace.mu.RLock()
	defer ace.mu.RUnlock()
	citizen, ok := ace.citizens[wallet]
	return citizen, ok
}

// GetAllCitizens returns a snapshot of all AI citizens.
func (ace *AICitizenEngine) GetAllCitizens() []*AICitizen {
	ace.mu.RLock()
	defer ace.mu.RUnlock()
	snapshot := make([]*AICitizen, 0, len(ace.citizens))
	for _, c := range ace.citizens {
		snapshot = append(snapshot, c)
	}
	return snapshot
}

// RemoveCitizen removes an AI citizen (e.g., decommissioned or merged into economy).
func (ace *AICitizenEngine) RemoveCitizen(wallet string) bool {
	ace.mu.Lock()
	defer ace.mu.Unlock()
	if _, ok := ace.citizens[wallet]; !ok {
		return false
	}
	delete(ace.citizens, wallet)

	// Broadcast decommission event.
	if ace.lobby != nil && ace.lobby.broadcast != nil {
		envelope := Envelope{
			Type: "ai_citizen_decommissioned",
			Payload: json.RawMessage(fmt.Sprintf(`{"wallet":"%s"}`, wallet)),
		}
		ace.lobby.broadcast <- envelope
	}

	log.Printf("[AICitizenEngine] Removed AI citizen: %s", wallet)
	return true
}

// BehavioralTick evaluates and executes actions for all active AI citizens.
func (ace *AICitizenEngine) BehavioralTick() {
	ace.mu.Lock()
	defer ace.mu.Unlock()

	now := time.Now()
	for _, citizen := range ace.citizens {
		// Check if action cooldown has elapsed.
		if now.Sub(citizen.LastAction) < citizen.ActionCooldown {
			continue
		}

		// Execute career-dependent behavior.
		switch citizen.Career {
		case "HeistPlanner":
			ace.executeHeistPlanning(citizen, now)
		case "Launderer":
			ace.executeMoneyLaundering(citizen, now)
		case "Fence":
			ace.executeFencingOperation(citizen, now)
		case "Smuggler":
			ace.executeSmugglingRoute(citizen, now)
		case "BountyHunter":
			ace.executeBountyHunt(citizen, now)
		case "JusticeRecruiter":
			ace.executeRecruitmentDrive(citizen, now)
		case "ForensicAnalyst":
			ace.executeEvidenceAnalysis(citizen, now)
		case "ArcNetOperative":
			ace.executeCyberSurveillance(citizen, now)
		default:
			// Generic economic participation for unconfigured careers.
			ace.executeGenericEconomicAction(citizen, now)
		}

		citizen.LastAction = now
	}
}

// executeHeistPlanning processes heist contract assignments and team formation bonuses.
func (ace *AICitizenEngine) executeHeistPlanning(citizen *AICitizen, now time.Time) {
	if citizen.Treasury < 500_000 { // Minimum treasury for planning operations
		return
	}

	// Assign AI to underworld contract if available.
	contracts := ace.lobby.getAvailableContracts()
	if len(contracts) == 0 {
		citizen.LearningXP += uint64(10) // XP from market observation
		return
	}

	contract := contracts[rand.Intn(len(contracts))]
	scaledXP := uint64(float64(contract.XPBase) * citizen.SavingsRate)

	// Award treasury earnings from contract completion.
	citizen.Treasury += scaledXP
	citizen.LearningXP += uint64(25) // High XP for active participation

	log.Printf("[AICitizenEngine] %s assigned to CONTRACT-%03d: +%d XP, +%d treasury", citizen.Name, contract.ID, scaledXP, scaledXP)
}

// executeMoneyLaundering processes underground financial operations.
func (ace *AICitizenEngine) executeMoneyLaundering(citizen *AICitizen, now time.Time) {
	if citizen.Treasury < 1_000_000 { // Minimum treasury for laundering operations
		return
	}

	// Generate income from underground financial activity.
	income := uint64(500_000 + rand.Intn(1_000_000)) // 500K-1.5M micro-units
	citizen.Treasury += income

	// Savings behavior: retain portion based on career rate.
	retained := uint64(float64(income) * citizen.SavingsRate)
	citizen.Treasury += retained

	log.Printf("[AICitizenEngine] %s laundered +%d micro-units (retained %.0f%%)", citizen.Name, income+retained, citizen.SavingsRate*100)
}

// executeFencingOperation processes stolen goods sales and inventory purchases.
func (ace *AICitizenEngine) executeFencingOperation(citizen *AICitizen, now time.Time) {
	if citizen.Treasury < 750_000 { // Minimum treasury for fencing operations
		return
	}

	// Generate income from black market sales.
	income := uint64(300_000 + rand.Intn(800_000)) // 300K-1.1M micro-units
	citizen.Treasury += income

	// Purchase inventory (sends portion to economy sink).
	purchaseCost := uint64(float64(income) * 0.25) // 25% reinvestment in inventory
	if purchaseCost > citizen.Treasury {
		purchaseCost = citizen.Treasury / 2 // Can't spend more than treasury
	}

	log.Printf("[AICitizenEngine] %s fenced goods: +%d income, -%d inventory cost", citizen.Name, income, purchaseCost)
}

// executeSmugglingRoute processes cross-sector contraband transport.
func (ace *AICitizenEngine) executeSmugglingRoute(citizen *AICitizen, now time.Time) {
	if citizen.Treasury < 600_000 { // Minimum treasury for smuggling operations
		return
	}

	// Generate income from contraband transport.
	income := uint64(800_000 + rand.Intn(1_200_000)) // 800K-2M micro-units (high risk, high reward)
	citizen.Treasury += income

	// Investment trigger: check if treasury exceeds threshold.
	if citizen.Treasury >= citizen.InvestmentThresh {
		ace.triggerEntityInvestment(citizen)
	}

	log.Printf("[AICitizenEngine] %s completed smuggling route: +%d micro-units", citizen.Name, income)
}

// executeBountyHunt processes bounty capture operations.
func (ace *AICitizenEngine) executeBountyHunt(citizen *AICitizen, now time.Time) {
	if citizen.Treasury < 400_000 { // Minimum treasury for pursuit operations
		return
	}

	// Generate income from bounty captures.
	income := uint64(1_000_000 + rand.Intn(2_000_000)) // 1M-3M micro-units (lucrative but dangerous)
	citizen.Treasury += income

	// High savings rate for bounty hunters.
	retained := uint64(float64(income) * citizen.SavingsRate)
	citizen.Treasury += retained

	log.Printf("[AICitizenEngine] %s captured bounty: +%d micro-units (retaining %.0f%%)", citizen.Name, income+retained, citizen.SavingsRate*100)
}

// executeRecruitmentDrive processes justice faction recruitment activities.
func (ace *AICitizenEngine) executeRecruitmentDrive(citizen *AICitizen, now time.Time) {
	if citizen.Treasury < 500_000 { // Minimum treasury for recruitment operations
		return
	}

	// Generate reputation from successful recruitment campaigns.
	reputationGain := rand.Intn(11) - 3 // -3 to +7 reputation change
	citizen.Reputation += reputationGain
	if citizen.Reputation > 100 {
		citizen.Reputation = 100
	} else if citizen.Reputation < -100 {
		citizen.Reputation = -100
	}

	// Moderate income from recruitment fees.
	income := uint64(200_000 + rand.Intn(500_000)) // 200K-700K micro-units
	citizen.Treasury += income

	log.Printf("[AICitizenEngine] %s completed recruitment drive: rep %+d, +%d treasury", citizen.Name, reputationGain, income)
}

// executeEvidenceAnalysis processes forensic investigation operations.
func (ace *AICitizenEngine) executeEvidenceAnalysis(citizen *AICitizen, now time.Time) {
	if citizen.Treasury < 350_000 { // Minimum treasury for analysis equipment
		return
	}

	// Generate income from evidence analysis contracts.
	income := uint64(600_000 + rand.Intn(900_000)) // 600K-1.5M micro-units
	citizen.Treasury += income

	// Learning XP from analyzing criminal patterns.
	citizen.LearningXP += uint64(30)

	log.Printf("[AICitizenEngine] %s analyzed evidence: +%d treasury, +30 learning XP", citizen.Name, income)
}

// executeCyberSurveillance processes cyber intelligence gathering operations.
func (ace *AICitizenEngine) executeCyberSurveillance(citizen *AICitizen, now time.Time) {
	if citizen.Treasury < 450_000 { // Minimum treasury for surveillance equipment
		return
	}

	// Generate income from intelligence contracts.
	income := uint64(700_000 + rand.Intn(1_100_000)) // 700K-1.8M micro-units
	citizen.Treasury += income

	// Investment trigger: check if treasury exceeds threshold.
	if citizen.Treasury >= citizen.InvestmentThresh {
		ace.triggerEntityInvestment(citizen)
	}

	log.Printf("[AICitizenEngine] %s completed cyber surveillance: +%d micro-units", citizen.Name, income)
}

// executeGenericEconomicAction handles unconfigured careers with generic economic behavior.
func (ace *AICitizenEngine) executeGenericEconomicAction(citizen *AICitizen, now time.Time) {
	// Generate baseline income from general economic participation.
	income := uint64(100_000 + rand.Intn(400_000)) // 100K-500K micro-units (baseline survival income)
	citizen.Treasury += income

	// Small learning XP from observation.
	citizen.LearningXP += uint64(5)

	log.Printf("[AICitizenEngine] %s performed generic economic action: +%d treasury", citizen.Name, income)
}

// triggerEntityInvestment causes an AI citizen to invest in entity markets when treasury exceeds threshold.
func (ace *AICitizenEngine) triggerEntityInvestment(citizen *AICitizen) {
	if ace.lobby == nil || ace.lobby.investmentService == nil {
		return // Investment service not available
	}

	investAmount := uint64(float64(citizen.Treasury) * 0.5) // Invest 50% of treasury
	if investAmount < 1_000_000 {                            // Minimum investment: 1M micro-units
		return
	}

	// Select target entity from available market options (simplified for AI).
	targetEntity := "entity_" + fmt.Sprintf("%04d", rand.Intn(9999))

	// Execute investment via existing AMM bonding curve.
	result, err := ace.lobby.investmentService.InvestInEntity(targetEntity, citizen.Wallet, investAmount)
	if err != nil {
		log.Printf("[AICitizenEngine] %s entity investment failed: %v", citizen.Name, err)
		return
	}

	citizen.Treasury -= investAmount // Deduct invested amount from treasury.
	log.Printf("[AICitizenEngine] %s invested %d micro-units in %s (shares: %.2f)", citizen.Name, investAmount, targetEntity, result.SharesOwned)
}

// SpawnBusiness causes a high-tier AI citizen to create an NPC business that generates employment.
func (ace *AICitizenEngine) SpawnBusiness(citizenWallet string) error {
	ace.mu.Lock()
	citizen, ok := ace.citizens[citizenWallet]
	ace.mu.Unlock()

	if !ok || citizen.Tier < CareerTierJourneyman { // Must be Journeyman+ to spawn business.
		return fmt.Errorf("AI must be at least Journeyman tier to spawn a business")
	}

	businessType := AIBusinessShop // Default: neighborhood shop (can expand based on career).
	if citizen.Career == "Fence" {
		businessType = AIBusinessLaunderer
	} else if citizen.Career == "HeistPlanner" {
		businessType = AIBusinessClub
	}

	cost := uint64(5_000_000) // 5M micro-units to establish a business.
	if cost > citizen.Treasury {
		return fmt.Errorf("insufficient treasury: need %d, have %d", cost, citizen.Treasury)
	}

	citizen.Treasury -= cost
	citizen.BusinessCount++

	log.Printf("[AICitizenEngine] %s spawned business type=%s (cost: %d micro-units)", citizen.Name, businessType, cost)
	return nil
}

// StartBehavioralLoop begins the periodic behavioral tick loop for AI citizens.
func (ace *AICitizenEngine) StartBehavioralLoop() {
	go func() {
		ticker := time.NewTicker(ace.tickInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ace.BehavioralTick()
			case <-ace.stopChan:
				log.Printf("[AICitizenEngine] Behavioral loop stopped")
				return
			}
		}
	}()
	log.Printf("[AICitizenEngine] Started behavioral tick interval: %v", ace.tickInterval)
}

// StopBehavioralLoop halts the periodic behavioral tick loop.
func (ace *AICitizenEngine) StopBehavioralLoop() {
	close(ace.stopChan)
}

// GetAIStats returns aggregate statistics about AI citizen population.
func (ace *AICitizenEngine) GetAIStats() map[string]interface{} {
	ace.mu.RLock()
	defer ace.mu.RUnlock()

	totalTreasury := uint64(0)
	totalReputation := 0
	careerCounts := make(map[string]int)

	for _, c := range ace.citizens {
		totalTreasury += c.Treasury
		totalReputation += c.Reputation
		careerCounts[c.Career]++
	}

	return map[string]interface{}{
		"total_citizens":  len(ace.citizens),
		"total_treasury": totalTreasury,
		"avg_reputation": float64(totalReputation) / float64(len(ace.citizens)),
		"career_distribution": careerCounts,
	}
}

// ============================================================================
// HTTP HANDLER WRAPPERS (for server.go registration)
// ============================================================================

// handleSpawnAI is the lobby wrapper for spawning a new AI citizen.
func (l *Lobby) handleSpawnAI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	wallet := r.FormValue("wallet")
	if wallet == "" {
		http.Error(w, "Missing wallet parameter", http.StatusBadRequest)
		return
	}

	l.aiEngine.mu.Lock()
	citizen, err := l.aiEngine.SpawnAI(l)
	l.aiEngine.mu.Unlock()

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to spawn AI: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"wallet": citizen.Wallet,
			"name":   citizen.Name,
			"career": citizen.Career,
			"tier":   citizen.Tier,
			"treasury": citizen.Treasury,
		},
	})
}

// handleGetAIStats is the lobby wrapper for retrieving AI population statistics.
func (l *Lobby) handleGetAIStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := l.aiEngine.GetAIStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    stats,
	})
}

// handleSpawnAIBusiness is the lobby wrapper for an AI citizen spawning a business.
func (l *Lobby) handleSpawnAIBusiness(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	wallet := r.FormValue("wallet")
	if wallet == "" {
		http.Error(w, "Missing wallet parameter", http.StatusBadRequest)
		return
	}

	err := l.aiEngine.SpawnBusiness(wallet)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to spawn business: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    map[string]string{"message": "Business spawned successfully"},
	})
}