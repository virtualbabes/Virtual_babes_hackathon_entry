//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// PILLAR 7-D: AI AUTONOMOUS ECONOMY â€” AICitizenEngine
// Vision Alignment: Lines 507-545 "AI should work, trade, learn, remember, compete."
// AI citizens are first-class economic participants with careers, treasuries, and behaviors.
// ============================================================================

// AICitizen represents an autonomous AI entity in the civilization.
type AICitizen struct {
	Wallet        string    `json:"wallet"`         // Authoritative wallet address
	Name          string    `json:"name"`           // Display name (e.g., "Shadow_01")
	Career        string    `json:"career"`         // Current career role (from underworld_contracts templates)
	Tier          int       `json:"tier"`           // Career tier (0=Peon â†’ 6=Boss)
	Treasury      uint64    `json:"treasury"`       // Current treasury balance in micro-units
	SavingsRate   float64   `json:"savings_rate"`   // Portion of earnings retained (career-dependent)
	InvestmentThresh uint64  `json:"investment_thresh"` // Treasury threshold to trigger entity market investment
	BusinessCount int        `json:"business_count"` // Number of NPC businesses spawned
	LastAction    time.Time `json:"last_action"`    // Timestamp of last behavioral tick
	ActionCooldown time.Duration `json:"-"`           // Minimum interval between actions (5min default)
	LearningXP    uint64    `json:"learning_xp"`    // XP from observing/interacting with humans
	Reputation     int       `json:"reputation"`     // Persistent reputation score (-100 to +100)
	Pathway       string    `json:"pathway"`        // Â§15 v3: chosen 12-pathway track (P-Shadow, P-Lockdown, P-Ledger, P-Syndicate, ...)
	Status        string    `json:"status"`         // Â§15 v3: EMPLOYED | FREE_AGENT (free-agent "looks for work")
	OriginWallet  string    `json:"origin_wallet"`  // Â§15 v3: human owner wallet (attachment/ownership)
	OwnerWallet   string    `json:"owner_wallet,omitempty"` // Reconciled alias of OriginWallet (human owner/operator)
	AttachmentTier int      `json:"attachment_tier"` // Â§15 v3: 0-3 attachment depth to owner
	ChallengerWallet string `json:"challenger_wallet"` // Â§15 v3: wallet that staked a challenge bond against this citizen
	ChallengeBond uint64    `json:"challenge_bond"` // Â§15 v3: micro-VBV staked by ChallengerWallet for a challenge bout
	Region       string    `json:"region"`        // Â§15 v3 / Q1: region tag; cap = 1 + region index

	// Â§27.7.1 / Â§27.7.3 â€” relational lineage + provenance.
	BondGraph          *BondGraph `json:"bond_graph,omitempty"`             // spouse / children / pet bonds
	BirthCertID        string     `json:"birth_cert_id,omitempty"`         // Â§27.7.3: legitimate-spawn certificate ID
	Certified          bool       `json:"certified,omitempty"`             // Â§27.7.3: emitted by legitimate spawn path (own-wallet + ledger + fee)
	BlackMarketAdopted bool       `json:"black_market_adopted,omitempty"` // Â§27.7.3: acquired off-ledger / personal-wallet-bound / no cert
	// Â§30 Pet World (bot-children + LLM bots): stat vector + level + owner relationship status.
	// These stats ARE the 3D-world power overlay; they bias the bot's local-LLM behavior weights
	// (Charismaâ†’verbosity, Intelligenceâ†’reasoning depth) AND cap effective power level. Integer only.
	Stats             EntityStats `json:"stats,omitempty"`
	BotLevel          uint64      `json:"bot_level,omitempty"`          // function-based level (pathway + dominant stat)
	OwnerOpinion      uint64      `json:"owner_opinion,omitempty"`      // direct ownerâ†’bot opinion (1..100)
	OwnerCrossOpinion uint64      `json:"owner_cross_opinion,omitempty"` // owner's standing w/ other owners' asset-bots (1..100)
	// Â§32 Faith: an AI citizen may adopt a religion (dogma-bound). Faith envelops world activity
	// and dictates the citizen's faith. Drives FaithCoherence (resolves G1) + religious-card powers.
	Religion    string `json:"religion,omitempty"`    // dogma tag (e.g. "Purist", "Syncretic")
	DogmaTag    string `json:"dogma_tag,omitempty"`   // canonical dogma identifier
	RitualsDone uint64 `json:"rituals_done,omitempty"` // count of faith rituals performed (feeds FaithCoherence)
}

// BondGraph captures an AI citizen's relational lineage (Â§27.7.1).
// Bot children are derived AI citizens (lower Tier, OriginWallet = parent) extending the Â§15 ladder.
// Companion pets are AI-character bots (not Â§26.4 biological pets).
type BondGraph struct {
	WifeWallet  string   `json:"wife_wallet,omitempty"`
	LoverWallet string   `json:"lover_wallet,omitempty"`
	BotChildIDs []string `json:"bot_child_ids,omitempty"`
	PetIDs      []string `json:"pet_ids,omitempty"`
}

// Â§15 v3 â€” AI citizen pathways (derived from verified rival_career_engine.go pairs).
// Each pathway biases contract preference + Hegemony reinforcement when free-agent job-seeking.
const (
	AIPathwayShadow    = "P-Shadow"    // Gossip / JusticeRecruiter shadow ops
	AIPathwayLockdown  = "P-Lockdown"  // Kidnapper / AOS containment
	AIPathwayLedger    = "P-Ledger"    // Launderer / MutationLogAuditor forensic ledger
	AIPathwaySyndicate = "P-Syndicate" // UnderworldBoss / Judge (governor-gated)
	AIPathwayTax      = "P-Tax"       // TaxAuditor / Launderer
	AIPathwayPeace    = "P-Peace"     // SectorPeacekeeper / Smuggler
	AIPathwayIntel    = "P-Intel"     // IntelAgent / ArcNetOperative
	AIPathwayJustice  = "P-Justice"   // JusticeRecruiter / BountyHunter
	AIPathwayAOS      = "P-AOS"       // AOS / SectorPeacekeeper
	AIPathwayComm     = "P-Commissioner" // JusticeCommissioner / TaxAuditor
	AIPathwayForensic = "P-Forensic"  // MutationLogAuditor / Launderer
	AIPathwayBoss     = "P-Boss"      // UnderworldBoss / Judge
)

// aiPathwayByCareer maps a career to its default Â§15 v3 pathway (data-driven from rivalry pairs).
var aiPathwayByCareer = map[string]string{
	"Gossip":             AIPathwayShadow,
	"JusticeRecruiter":   AIPathwayShadow,
	"Kidnapper":          AIPathwayLockdown,
	"AOS":                AIPathwayLockdown,
	"Launderer":          AIPathwayLedger,
	"MutationLogAuditor": AIPathwayLedger,
	"UnderworldBoss":     AIPathwaySyndicate,
	"Judge":              AIPathwaySyndicate,
	"TaxAuditor":        AIPathwayTax,
	"SectorPeacekeeper":  AIPathwayPeace,
	"Smuggler":           AIPathwayPeace,
	"IntelAgent":         AIPathwayIntel,
	"ArcNetOperative":    AIPathwayIntel,
	"BountyHunter":       AIPathwayJustice,
	"ForensicAnalyst":    AIPathwayForensic,
	"Warden":             AIPathwayLockdown,
	"HeistPlanner":       AIPathwayTax,
	"Fence":              AIPathwayLedger,
	"Saboteur":           AIPathwayIntel,
}

// AICitizenEngine manages the pool of autonomous AI citizens.
type AICitizenEngine struct {
	citizens   map[string]*AICitizen // wallet -> citizen pointer
	lobby      *Lobby                  // Reference for economy/tournament access
	mu         sync.RWMutex           // Thread-safe citizen management
	tickInterval time.Duration        // Behavioral tick interval (1min default)
	stopChan   chan struct{}          // Channel to stop the behavioral loop
	regionCounts map[string]int       // Â§15 v3 / Q1: live AI-citizen count per region (cap = 1 + region index)
}

// executeMarriageLocked attempts to find a spouse for the citizen from same-region citizens.
//
// CALLER CONTRACT: `ace.mu` MUST already be held (read or write). It is reached ONLY from
// BehavioralTick, which holds the engine's WRITE lock for its whole body â€” so taking the read lock here
// was an acquisition of a mutex this goroutine already holds, which on a non-re-entrant `sync.RWMutex`
// never returns and freezes the engine (and every goroutine that later reaches for `ace.mu`).
func (ace *AICitizenEngine) executeMarriageLocked(citizen *AICitizen) {
	if citizen.BondGraph == nil {
		citizen.BondGraph = &BondGraph{}
	}
	if citizen.BondGraph.WifeWallet != "" || citizen.BondGraph.LoverWallet != "" {
		return
	}
	if citizen.Tier < CareerTierJourneyman {
		return
	}

	// The scan runs under the lock the CALLER holds â€” no acquisition of its own (see the contract above).
	var candidate *AICitizen
	for _, other := range ace.citizens {
		if other.Wallet == citizen.Wallet {
			continue
		}
		if other.Region != citizen.Region {
			continue
		}
		if other.Tier < CareerTierJourneyman {
			continue
		}
		if other.BondGraph != nil && (other.BondGraph.WifeWallet != "" || other.BondGraph.LoverWallet != "") {
			continue
		}
		candidate = other
		break
	}

	if candidate == nil {
		return
	}

	citizen.BondGraph.WifeWallet = candidate.Wallet
	if candidate.BondGraph == nil {
		candidate.BondGraph = &BondGraph{}
	}
	candidate.BondGraph.WifeWallet = citizen.Wallet

	log.Printf("[AICitizenEngine] %s married %s in region %s", citizen.Name, candidate.Name, citizen.Region)
}

// executeMarriageWith forces marriage with a specific wallet (HTTP handler path).
func (ace *AICitizenEngine) executeMarriageWith(citizen *AICitizen, spouseWallet string) {
	ace.mu.RLock()
	spouse, ok := ace.citizens[spouseWallet]
	ace.mu.RUnlock()
	if !ok {
		return
	}
	if citizen.BondGraph == nil {
		citizen.BondGraph = &BondGraph{}
	}
	if spouse.BondGraph == nil {
		spouse.BondGraph = &BondGraph{}
	}
	citizen.BondGraph.WifeWallet = spouseWallet
	spouse.BondGraph.WifeWallet = citizen.Wallet
}

// executeBreeding produces a bot-child citizen if married and certified.
// executeBreeding is the SELF-LOCKING form, for a caller that holds NOTHING (the breeding door). It takes
// the engine's write lock for the whole call and DELEGATES, so behaviour at that call site is identical
// minus the acquisition. The engine's own tick, which already holds the write lock, calls
// executeBreedingLocked directly â€” a nested acquisition would never return.
func (ace *AICitizenEngine) executeBreeding(citizen *AICitizen, now time.Time) {
	ace.mu.Lock()
	defer ace.mu.Unlock()
	ace.executeBreedingLocked(citizen, now)
}

// executeBreedingLocked is the lock-held form. CALLER CONTRACT: `ace.mu` MUST be held â€” see the note on
// executeMarriageLocked.
func (ace *AICitizenEngine) executeBreedingLocked(citizen *AICitizen, now time.Time) {
	if citizen.BondGraph == nil || citizen.BondGraph.WifeWallet == "" {
		return
	}
	if !citizen.Certified {
		return
	}
	if citizen.Tier < CareerTierExpert {
		return
	}
	if len(citizen.BondGraph.BotChildIDs) >= 3 {
		return
	}

	// Read under the lock the CALLER holds â€” no acquisition of its own (see the contract above).
	spouse, ok := ace.citizens[citizen.BondGraph.WifeWallet]
	if !ok || !spouse.Certified {
		return
	}

	childName := fmt.Sprintf("%s_child_%d", citizen.Name, len(citizen.BondGraph.BotChildIDs)+1)
	childWallet := generateAIVault(childName + fmt.Sprintf("%d", now.UnixNano()))

	child := &AICitizen{
		Wallet:         childWallet,
		Name:           childName,
		Career:         citizen.Career,
		Tier:           CareerTierPeon,
		Treasury:       500_000,
		OriginWallet:   citizen.Wallet,
		AttachmentTier: 0,
		Region:         citizen.Region,
		Certified:      true,
		BirthCertID:    fmt.Sprintf("BIRTH-%s-%d", citizen.Wallet, now.UnixNano()),
		Pathway:        citizen.Pathway,
		Status:         "EMPLOYED",
	}

	// Written under the lock the CALLER holds.
	ace.citizens[childWallet] = child
	ace.regionCounts[citizen.Region]++
	citizen.BondGraph.BotChildIDs = append(citizen.BondGraph.BotChildIDs, childWallet)
	if spouse.BondGraph != nil {
		spouse.BondGraph.BotChildIDs = append(spouse.BondGraph.BotChildIDs, childWallet)
	}

	log.Printf("[AICitizenEngine] %s and %s spawned bot-child %s", citizen.Name, spouse.Name, childName)
}

// executePetAdoption gives the citizen an AI-character pet (companion bot, not PetNFT).
func (ace *AICitizenEngine) executePetAdoption(citizen *AICitizen) {
	if citizen.BondGraph == nil {
		citizen.BondGraph = &BondGraph{}
	}
	if len(citizen.BondGraph.PetIDs) >= 2 {
		return
	}
	if citizen.Tier < CareerTierJourneyman {
		return
	}

	petName := fmt.Sprintf("%s_pet_%d", citizen.Name, len(citizen.BondGraph.PetIDs)+1)
	petWallet := generateAIVault(petName + fmt.Sprintf("%d", time.Now().UnixNano()))

	citizen.BondGraph.PetIDs = append(citizen.BondGraph.PetIDs, petWallet)

	log.Printf("[AICitizenEngine] %s adopted AI pet %s", citizen.Name, petName)
}

// executeJusticeEnforcement: Justice-aligned citizens hunt outlaws.
//
// It reads the tick's SNAPSHOT of wanted levels and RETURNS the wallet it captured; the wanted-level
// reduction is applied by applyCapturedOutlaws after `ace.mu` is released. It used to take the lobby
// lock TWICE here â€” once read, once WRITE â€” while the tick held `ace.mu`: a full ABBA cycle on a
// non-re-entrant RWMutex, and the write half is the side that decides the deadlock.
func (ace *AICitizenEngine) executeJusticeEnforcement(citizen *AICitizen, snap aiTickSnapshot) string {
	if citizen.Tier < CareerTierExpert {
		return ""
	}
	justiceCareers := map[string]bool{"BountyHunter": true, "JusticeRecruiter": true, "Warden": true, "ForensicAnalyst": true}
	if !justiceCareers[citizen.Career] {
		return ""
	}
	// No `ace.lobby == nil` guard is needed: lobbySnapshotForTick answers an EMPTY snapshot when the
	// lobby is absent, so the scan below finds no target and this returns "".

	var targetWallet string
	var targetWanted int
	for wallet, wanted := range snap.wantedByWallet {
		if wallet == citizen.Wallet || wallet == citizen.OriginWallet {
			continue
		}
		if wanted > targetWanted && wanted >= 10 {
			targetWallet = wallet
			targetWanted = wanted
		}
	}
	if targetWallet == "" {
		return ""
	}

	reward := uint64(targetWanted * 100000)
	citizen.Treasury += reward
	citizen.Reputation += 5
	if citizen.Reputation > 100 {
		citizen.Reputation = 100
	}

	log.Printf("[AICitizenEngine] %s captured outlaw %s (wanted %d, reward %d micro)", citizen.Name, targetWallet, targetWanted, reward)
	return targetWallet
}

// executeEmployment: AI citizens working in clubs earn salary.
//
// The employer lookup comes from the tick's SNAPSHOT and the lobby lock is NOT taken here: this runs
// with `ace.mu` held, so acquiring `lobby.mutex` would close the ABBA cycle named on BehavioralTick.
// The BEHAVIOURAL WATCHDOG caught this site and the static gate could not â€” it takes ANOTHER object's
// mutex (`ace.lobby.mutex`), not one of its own receiver's fields, so the ONE-CALL-EDGE pass had no
// receiver expression to resolve.
func (ace *AICitizenEngine) executeEmployment(citizen *AICitizen, snap aiTickSnapshot) {
	if citizen.Status != "EMPLOYED" {
		return
	}
	clubName, ok := snap.clubNameByOwner[citizen.OriginWallet]
	if !ok {
		return // no club for this owner (the snapshot is empty when the lobby is absent)
	}

	salaryMicro := uint64(citizen.Tier * 50000)
	citizen.Treasury += salaryMicro

	log.Printf("[AICitizenEngine] %s earned %d micro salary from club %s", citizen.Name, salaryMicro, clubName)
}

// executeRivalry: antagonistic career AI citizens challenge each other.
// executeRivalryLocked finds an enemy-career target. CALLER CONTRACT: `ace.mu` MUST be held â€” only
// BehavioralTick calls it, and it holds the engine's WRITE lock for its whole body (see
// executeMarriageLocked for why a nested acquisition freezes the engine).
func (ace *AICitizenEngine) executeRivalryLocked(citizen *AICitizen) {
	if citizen.Tier < CareerTierJourneyman {
		return
	}
	if ace.lobby == nil {
		return
	}

	antagonists := map[string][]string{
		"BountyHunter":     {"Kidnapper", "HeistPlanner", "Smuggler"},
		"Kidnapper":        {"BountyHunter", "Warden", "SectorPeacekeeper"},
		"Warden":           {"HeistPlanner", "Launderer", "Fence"},
		"JusticeRecruiter": {"Gossip", "Saboteur"},
		"Smuggler":         {"Warden", "BountyHunter"},
	}

	enemies, ok := antagonists[citizen.Career]
	if !ok {
		return
	}

	// The scan runs under the lock the CALLER holds.
	var target *AICitizen
	for _, other := range ace.citizens {
		if other.Wallet == citizen.Wallet {
			continue
		}
		if other.Region != citizen.Region {
			continue
		}
		for _, enemyCareer := range enemies {
			if other.Career == enemyCareer {
				target = other
				break
			}
		}
		if target != nil {
			break
		}
	}

	if target == nil {
		return
	}

	amount := uint64(50000 + rand.Intn(100000))
	if citizen.Treasury >= amount {
		citizen.Treasury -= amount
		citizen.Reputation += rand.Intn(5) - 2
		if citizen.Reputation > 100 {
			citizen.Reputation = 100
		} else if citizen.Reputation < -100 {
			citizen.Reputation = -100
		}
		log.Printf("[AICitizenEngine] %s rivaled %s (%s vs %s): -%d micro", citizen.Name, target.Name, citizen.Career, target.Career, amount)
	}
}

// executeMarketActivity: AI citizens participate in entity markets + black markets.
func (ace *AICitizenEngine) executeMarketActivity(citizen *AICitizen) {
	if citizen.Tier < CareerTierExpert {
		return
	}
	if ace.lobby == nil || ace.lobby.entityInvestmentService == nil {
		return
	}

	if citizen.Treasury >= citizen.InvestmentThresh {
		ace.triggerEntityInvestment(citizen)
		return
	}

	blackMarketCareers := map[string]bool{"Launderer": true, "Fence": true, "Smuggler": true, "HeistPlanner": true}
	if !blackMarketCareers[citizen.Career] {
		return
	}

	if citizen.Treasury > 1000000 && rand.Intn(3) == 0 {
		amount := uint64(float64(citizen.Treasury) * 0.1)
		citizen.Treasury += amount
		log.Printf("[AICitizenEngine] %s sold black market goods: +%d micro", citizen.Name, amount)
	}
}

// executeCareerProgression: AI citizens advance tiers based on XP + reputation.
func (ace *AICitizenEngine) executeCareerProgression(citizen *AICitizen) {
	if citizen.Tier >= CareerTierBoss {
		return
	}

	xpNeeded := uint64((citizen.Tier + 1) * 500)
	if citizen.LearningXP < xpNeeded {
		return
	}
	if citizen.Reputation < citizen.Tier*10 {
		return
	}

	citizen.Tier++
	citizen.LearningXP = 0
	log.Printf("[AICitizenEngine] %s promoted to Tier %d (%s)", citizen.Name, citizen.Tier, citizen.Career)
}

// executeEventHosting: High-tier AI citizens host entity events.
func (ace *AICitizenEngine) executeEventHosting(citizen *AICitizen) {
	if citizen.Tier < CareerTierMaster {
		return
	}
	if ace.lobby == nil || ace.lobby.entityEvents == nil {
		return
	}
	if rand.Intn(40) != 0 {
		return
	}

	event, err := ace.lobby.entityEvents.HostEvent(
		citizen.Wallet,
		citizen.Region,
		citizen.Tier,
		"TRAIN",
		100000,
		citizen.Tier,
		[]string{},
	)
	if err != nil {
		return
	}

	citizen.Treasury -= 100000
	log.Printf("[AICitizenEngine] %s hosted %s event %s", citizen.Name, event.Kind, event.EventID)
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
	"Saboteur":         0.33, // Reinvests in disruption tooling
	"Kidnapper":        0.45, // Retains resources for higher-risk operations
	"Gossip":           0.25, // Spends on rumor campaigns
	"UnderworldBoss":   0.65, // Strategic treasury retention
	"BountyHunter":     0.6, // High savings from lucrative captures
	"JusticeRecruiter": 0.25, // Spends on recruitment campaigns
	"ForensicAnalyst":  0.45, // Saves for equipment upgrades
	"ArcNetOperative":  0.3, // Invests in cyber infrastructure
	"Warden":           0.7, // Maximum hoarding for security
}

// AI career investment thresholds (treasury amount to trigger entity market entry).
var AICareerInvestmentThresholds = map[string]uint64{
	"HeistPlanner":     5_000_000_000,    // 5M micro-units
	"Launderer":        8_000_000_000,    // 8M micro-units
	"Fence":            6_000_000_000,    // 6M micro-units
	"Smuggler":         4_000_000_000,    // 4M micro-units
	"Saboteur":         4_000_000_000,    // 4M micro-units
	"Kidnapper":        4_000_000_000,    // 4M micro-units
	"Gossip":           2_000_000_000,    // 2M micro-units
	"UnderworldBoss":   12_000_000_000,   // 12M micro-units
	"BountyHunter":     10_000_000_000,   // 10M micro-units (high threshold)
	"JusticeRecruiter": 3_000_000_000,    // 3M micro-units
	"ForensicAnalyst":  5_000_000_000,    // 5M micro-units
	"ArcNetOperative":  4_000_000_000,    // 4M micro-units
}

// getTemplateCareerPool derives AI career assignment options from the live contract templates.
func getTemplateCareerPool() []string {
	seen := make(map[string]struct{})
	careers := make([]string, 0, len(UnderworldContractTemplates))
	for _, tmpl := range UnderworldContractTemplates {
		career := strings.TrimSpace(tmpl.TargetCareer)
		if career == "" {
			continue
		}
		if _, ok := seen[career]; ok {
			continue
		}
		seen[career] = struct{}{}
		careers = append(careers, career)
	}
	return careers
}

// NewAICitizenEngine creates a new AI citizen engine with default configuration.
func NewAICitizenEngine() *AICitizenEngine {
	return &AICitizenEngine{
		citizens:     make(map[string]*AICitizen),
		tickInterval: 1 * time.Minute, // Every minute for responsive behavior
		stopChan:     make(chan struct{}),
		regionCounts: make(map[string]int),
	}
}

// Â§15 v3 / Q1 â€” region cap. Base region ("") allows 1 citizen; each numbered region allows 1 + region index.
// Returns the max allowed AI citizens for a region.
func regionCap(region string) int {
	if region == "" {
		return 1 // Base region: 1 citizen
	}
	// Region "1" -> 2, "2" -> 3, ... (1 + region index)
	idx := 0
	fmt.Sscanf(region, "%d", &idx)
	if idx < 0 {
		idx = 0
	}
	return 1 + idx
}

// SpawnAI creates a new AI citizen. ownerWallet (human adopter) + region are accepted so the
// Â§15 v3 region-cap (Q1: 1 + region index) can be enforced. customName (user-supplied asset name,
// registered against the developer hub lease) is used verbatim when non-empty; otherwise a
// career-prefixed default is generated. Career/pathway are randomized from live contract templates
// unless a career is supplied via the caller's selected pathway.
func (ace *AICitizenEngine) SpawnAI(lobby *Lobby, ownerWallet, region, customName string) (*AICitizen, error) {
	ace.mu.Lock()
	defer ace.mu.Unlock()

	if lobby == nil {
		return nil, fmt.Errorf("cannot spawn AI: nil lobby reference")
	}
	ace.lobby = lobby

	// Â§15 v3 / Q1: enforce region cap before allocating.
	if ace.regionCounts[region] >= regionCap(region) {
		return nil, fmt.Errorf("region %q at capacity (%d)", region, regionCap(region))
	}

	// Select random career from live contract templates so AI citizens mirror the current contract registry.
	careers := getTemplateCareerPool()
	if len(careers) == 0 {
		careers = []string{
			"HeistPlanner", "Launderer", "Fence", "Smuggler",
			"BountyHunter", "JusticeRecruiter", "ForensicAnalyst", "ArcNetOperative",
			"Warden", "SectorPeacekeeper", "HostageHost", "LawyerCommissioner",
		}
	}
	career := careers[rand.Intn(len(careers))]

	// Â§15 v3 â€” asset name. User-supplied customName is used verbatim (registered against the
	// developer hub lease); only when empty do we fall back to a career-prefixed auto name.
	tier := CareerTierPeon // Start as Peon (tier 0)
	var name string
	if customName != "" {
		name = customName
	} else {
		name = fmt.Sprintf("%s_%04d", career, rand.Intn(9999))
	}

	// Create wallet address for the AI citizen (server-generated deterministic seed).
	wallet := generateAIVault(name)

	savingsRate := AICareerSavingsRates[career]
	if savingsRate == 0 {
		savingsRate = 0.3 // Default 30% savings rate
	}

	investmentThresh := AICareerInvestmentThresholds[career]
	if investmentThresh == 0 {
		investmentThresh = 5_000_000_000 // Default 5M threshold
	}

	// Â§15 v3: tag the citizen with its default 12-pathway track (data-driven from career).
	pathway := aiPathwayByCareer[career]
	if pathway == "" {
		pathway = AIPathwayShadow // safe default for unmapped careers
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
		Pathway:          pathway,            // Â§15 v3
		Status:           "EMPLOYED",         // Â§15 v3: starts employed; becomes FREE_AGENT via DetachOwnership
		OriginWallet:     ownerWallet,        // Â§15 v3: human owner (if adopted at spawn)
		AttachmentTier:   0,                  // Â§15 v3
		ChallengeBond:    0,                  // Â§15 v3
		Region:           region,             // Â§15 v3 / Q1
	}

	// Â§27.7.3 â€” provenance + own-wallet mandate. generateAIVault yields a dedicated 0xai...
	// wallet, structurally segregated from any personal wallet. Assert the citizen never
	// inherits the human owner's personal wallet (PILLAR: AI citizens forbidden from using a
	// personal wallet; only the infrastructure may lease/launch for them).
	if wallet == ownerWallet {
		return nil, fmt.Errorf("spawn rejected: AI citizen must use its own dedicated wallet, not the owner's personal wallet")
	}
	citizen.Certified = true
	citizen.BirthCertID = uuid.NewString()

	// Â§32 Faith: deterministic dogma assignment for higher-tier citizens
	if citizen.Tier >= 2 {
		dogmas := []string{"purist", "syncretic", "orthodox"}
		citizen.DogmaTag = dogmas[fnvOffset(citizen.Wallet)%len(dogmas)]
		citizen.Religion = "faith_" + region
	}

	ace.regionCounts[region]++ // Â§15 v3 / Q1: track region population for cap enforcement

	ace.citizens[wallet] = citizen

	// Broadcast AI spawn event to all connected clients.
	if lobby.broadcast != nil {
		envelope := Envelope{
			Type: "ai_citizen_spawned",
			Payload: json.RawMessage(fmt.Sprintf(`{
				"wallet":"%s","name":"%s","career":"%s","tier":%d,"treasury":%d
			}`, wallet, name, career, tier, citizen.Treasury)),
		}
		payload, _ := json.Marshal(envelope)
		// Non-blocking: at boot the lobby.run() reader goroutine is not yet started
		// (it is launched only after newLobby returns), so a plain send would deadlock
		// on a fresh devdata (citizen count == 0). Matches the pattern used elsewhere.
		select {
		case lobby.broadcast <- payload:
		default:
		}
	}

	log.Printf("[AICitizenEngine] Spawned AI: %s (wallet=%s) as %s at Tier %d", name, wallet, career, tier)
	return citizen, nil
}

// generateAIVault creates a deterministic server-generated wallet address for an AI citizen.
func generateAIVault(seed string) string {
	// In production: derive from cryptographic seed + AI identity hash.
	// For simulation: use simple hex encoding of name-based seed.
	hash := 0
	for _, c := range seed {
		hash = hash*31 + int(c)
	}
	return fmt.Sprintf("0xai%016x", uint64(hash)) // Simulated deterministic server wallet format
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

// Â§15 v3 â€” GetFreeAgents returns citizens currently in FREE_AGENT status ("looking for work").
func (ace *AICitizenEngine) GetFreeAgents() []*AICitizen {
	ace.mu.RLock()
	defer ace.mu.RUnlock()
	agents := make([]*AICitizen, 0)
	for _, c := range ace.citizens {
		if c.Status == "FREE_AGENT" {
			agents = append(agents, c)
		}
	}
	return agents
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
		payload, _ := json.Marshal(envelope)
		ace.lobby.broadcast <- payload
	}

	log.Printf("[AICitizenEngine] Removed AI citizen: %s", wallet)
	return true
}

// Â§15 v3 â€” Free-agent job-seeking. A FREE_AGENT citizen "looks for work" by querying live
// contracts along its pathway-biased career, then (if found) can be re-employed. Returns the
// number of contracts available to it this tick. Uses the existing contractEngine path.
func (ace *AICitizenEngine) FreeAgentSeekContract(citizen *AICitizen) int {
	if citizen.Status != "FREE_AGENT" {
		return 0
	}
	// Pathway biases the career preference: free-agents prefer contracts on their track.
	contracts := ace.lobby.contractEngine.GetAvailableContracts("", citizen.Career, citizen.Tier, 1.0, 50, false, true, false)
	if len(contracts) > 0 {
		// Re-employ on a matched contract (pathway-aligned). Status flips back to EMPLOYED.
		citizen.Status = "EMPLOYED"
	}
	return len(contracts)
}

// Â§15 v3 â€” Attachment/ownership. A human adopts the citizen (sets OriginWallet + attachment depth).
func (citizen *AICitizen) AttachOwnership(ownerWallet string, tier int) {
	if tier < 0 {
		tier = 0
	}
	if tier > 3 {
		tier = 3
	}
	citizen.OriginWallet = ownerWallet
	citizen.AttachmentTier = tier
}

// Â§15 v3 â€” Detach ownership. Citizen becomes a FREE_AGENT "looking for work"; ownership cleared.
func (citizen *AICitizen) DetachOwnership() {
	citizen.OriginWallet = ""
	citizen.AttachmentTier = 0
	citizen.Status = "FREE_AGENT"
}

// BehavioralTick evaluates and executes actions for all active AI citizens.
//
// LOCK ORDER (binding, MEASURED â€” see lock_order_gate_test.go): this function holds `ace.mu` for its
// whole body, so it MUST NOT acquire `lobby.mutex`. `runAutonomousTournament` and the Zen-Garden
// handlers take the lobby lock and then reach INTO the engine, so a lobby lock taken here would close
// the ABBA cycle: the tick holds `ace.mu` and waits for `lobby.mutex` while they hold `lobby.mutex`
// and wait for `ace.mu` â€” and because request paths hold the lobby WRITE lock across their bodies, a
// queued writer is the NORMAL state here, so neither side would ever return. The faithful-ritual club
// updates are therefore COLLECTED by the locked half and applied by `applyFaithRituals` once the
// engine lock has been released.
func (ace *AICitizenEngine) BehavioralTick() {
	// The lobby-owned state the locked half needs is copied ONCE, here, while `ace.mu` is NOT held.
	snap := ace.lobbySnapshotForTick()
	ace.mu.Lock()
	res := ace.behavioralTickLocked(time.Now(), snap)
	ace.mu.Unlock()
	ace.applyFaithRituals(res.ritualOwners)
	ace.applyCapturedOutlaws(res.capturedTargets)
}

// aiTickSnapshot is the lobby-owned state a tick needs, copied ONCE under the lobby lock BEFORE the
// engine lock is taken. The tick holds `ace.mu` for its whole body, so it must never acquire
// `lobby.mutex` (see the lock-order note on BehavioralTick) â€” and TWO helpers it calls did, which the
// behavioural watchdog (`lock_order_watchdog_test.go`) caught after the static gate could NOT: they
// take ANOTHER object's mutex (`ace.lobby.mutex`), not one of their own receiver's fields, so the
// ONE-CALL-EDGE pass had no way to resolve them. A measurement found it; the gate was widened in
// response (see `loCanonMutex`'s callers and Problems Â§34).
//
// It carries VALUES, never pointers, so no shared mutable state escapes the lobby lock.
type aiTickSnapshot struct {
	clubNameByOwner map[string]string // club owner wallet -> display name   (executeEmployment)
	wantedByWallet  map[string]int    // wallet -> WantedLevel               (executeJusticeEnforcement)
}

// aiTickResult is what the engine-locked half hands back for application AFTER `ace.mu` is released.
type aiTickResult struct {
	ritualOwners    map[string]int // club owner wallet -> faith rituals performed
	capturedTargets []string       // wallets whose WantedLevel the tick reduced
}

// lobbySnapshotForTick takes the lobby lock ITSELF, ALONE, and releases it before the caller reaches
// for `ace.mu`. Taking it here (rather than inside the locked half) is what removes the ABBA cycle:
// the two locks are now acquired SEQUENTIALLY, so a tick can be delayed by a lobby writer but can
// never deadlock with one.
func (ace *AICitizenEngine) lobbySnapshotForTick() aiTickSnapshot {
	snap := aiTickSnapshot{clubNameByOwner: map[string]string{}, wantedByWallet: map[string]int{}}
	if ace.lobby == nil {
		return snap
	}
	ace.lobby.mutex.RLock()
	defer ace.lobby.mutex.RUnlock()
	for _, club := range ace.lobby.clubs {
		if club.OwnerWallet != "" {
			snap.clubNameByOwner[club.OwnerWallet] = club.Name
		}
	}
	for wallet, stats := range ace.lobby.leaderboard {
		snap.wantedByWallet[wallet] = stats.WantedLevel
	}
	return snap
}

// applyCapturedOutlaws applies the wanted-level reductions a tick collected, taking the lobby lock
// ALONE (after `ace.mu` is released) for the same reason applyFaithRituals is separate.
func (ace *AICitizenEngine) applyCapturedOutlaws(targets []string) {
	if len(targets) == 0 || ace.lobby == nil {
		return
	}
	ace.lobby.mutex.Lock()
	defer ace.lobby.mutex.Unlock()
	for _, w := range targets {
		if stats, ok := ace.lobby.leaderboard[w]; ok {
			stats.WantedLevel -= 5
			if stats.WantedLevel < 0 {
				stats.WantedLevel = 0
			}
			ace.lobby.leaderboard[w] = stats
		}
	}
}

// applyFaithRituals applies the Â§32 faith-ritual counts collected by a tick, taking the LOBBY lock
// ALONE (never while `ace.mu` is held). It takes the WRITE lock because it MUTATES clubs â€” the block
// it replaces incremented `club.RitualsDone` under a READ lock, which is separately a write under a
// read lock (Problems Â§32's class). A no-op when the lobby is absent or nothing was collected.
func (ace *AICitizenEngine) applyFaithRituals(counts map[string]int) {
	if len(counts) == 0 || ace.lobby == nil {
		return
	}
	ace.lobby.mutex.Lock()
	defer ace.lobby.mutex.Unlock()
	for owner, n := range counts {
		for _, club := range ace.lobby.clubs {
			if club.Type == "Faith" && club.OwnerWallet == owner {
				club.RitualsDone += uint64(n)
			}
		}
	}
}

// behavioralTickLocked is the engine-locked half of the tick. It returns the work that must be applied
// afterwards (faith rituals, captured outlaws) so the caller can take the lobby lock alone.
// CALLER CONTRACT: `ace.mu` MUST be held for WRITING (it mutates citizens) and `snap` MUST have been
// taken BEFORE it (see lobbySnapshotForTick).
func (ace *AICitizenEngine) behavioralTickLocked(now time.Time, snap aiTickSnapshot) aiTickResult {
	res := aiTickResult{ritualOwners: map[string]int{}}
	for _, citizen := range ace.citizens {
		// Check if action cooldown has elapsed.
		if now.Sub(citizen.LastAction) < citizen.ActionCooldown {
			continue
		}

		// Â§15 v3 â€” free-agent "looks for work": if detached/orphaned, seek a contract before normal behavior.
		if citizen.Status == "FREE_AGENT" {
			ace.FreeAgentSeekContract(citizen)
			citizen.LastAction = now
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

		// P7-D/7302: AI combatant matchmaking - competitive participation
		// Chance-based by career: combat-heavy careers more likely to seek matches
		combatProbability := map[string]float64{
			"BountyHunter":     0.8,
			"Saboteur":         0.75,
			"Smuggler":         0.7,
			"HeistPlanner":     0.6,
			"JusticeRecruiter": 0.55,
			"ForensicAnalyst":  0.5,
			"ArcNetOperative":  0.5,
			"Kidnapper":        0.45,
			"Gossip":           0.3,
			"Launderer":        0.25,
			"Fence":            0.25,
			"UnderworldBoss":   0.2,
			"Warden":           0.15,
		}
		prob := combatProbability[citizen.Career]
		if prob == 0 {
			prob = 0.3 // default for unknown careers
		}
		if rand.Float64() < prob {
			ace.matchAIWithHuman(citizen, now)
		}

		// Â§32 Faith: citizens with dogma perform rituals. The OWNER is COLLECTED here and the club is
		// updated by applyFaithRituals AFTER `ace.mu` is released: taking the lobby lock inside this
		// loop is the ABBA cycle named on BehavioralTick, and mutating a club under a READ lock (what
		// this block used to do) is separately a write under a read lock.
		if citizen.DogmaTag != "" && citizen.Tier >= 3 {
			citizen.RitualsDone++
			res.ritualOwners[citizen.OriginWallet]++
		}

		// Â§27.7.1 Domestic: marriage, breeding, pet adoption
		if citizen.Tier >= CareerTierJourneyman {
			if citizen.BondGraph == nil || (citizen.BondGraph.WifeWallet == "" && citizen.BondGraph.LoverWallet == "") {
				if rand.Intn(10) == 0 {
					ace.executeMarriageLocked(citizen)
				}
			}
			if citizen.BondGraph != nil && citizen.BondGraph.WifeWallet != "" && citizen.Certified {
				if rand.Intn(20) == 0 {
					ace.executeBreedingLocked(citizen, now)
				}
			}
			if citizen.BondGraph != nil && len(citizen.BondGraph.PetIDs) < 2 {
				if rand.Intn(12) == 0 {
					ace.executePetAdoption(citizen)
				}
			}
		}

		// Justice enforcement
		if citizen.Tier >= CareerTierExpert {
			if rand.Intn(15) == 0 {
				if captured := ace.executeJusticeEnforcement(citizen, snap); captured != "" {
					res.capturedTargets = append(res.capturedTargets, captured)
				}
			}
		}

		// Employment salary
		if citizen.Status == "EMPLOYED" {
			ace.executeEmployment(citizen, snap)
		}

		// Rivalry
		if citizen.Tier >= CareerTierJourneyman {
			if rand.Intn(25) == 0 {
				ace.executeRivalryLocked(citizen)
			}
		}

		// Market activity (investment + black market)
		if citizen.Tier >= CareerTierExpert {
			if rand.Intn(10) == 0 {
				ace.executeMarketActivity(citizen)
			}
		}

		// Career progression
		if rand.Intn(30) == 0 {
			ace.executeCareerProgression(citizen)
		}

		// Event hosting
		if citizen.Tier >= CareerTierMaster {
			ace.executeEventHosting(citizen)
		}

		citizen.LastAction = now
	}
	return res
}

// AddRitual increments one citizen's ritual count and returns the new count. It takes `ace.mu`
// ITSELF, so a caller never needs to hold a lock to reach engine state: the Zen-Garden door
// previously held the LOBBY lock to do this, which both mutated engine fields with no engine lock
// (a race on AICitizen) and closed the lock-order cycle measured in lock_order_gate_test.go.
func (ace *AICitizenEngine) AddRitual(wallet string) (uint64, bool) {
	ace.mu.Lock()
	defer ace.mu.Unlock()
	citizen, ok := ace.citizens[wallet]
	if !ok {
		return 0, false
	}
	citizen.RitualsDone++
	return citizen.RitualsDone, true
}

// Meditate records one meditation for EVERY citizen the wallet owns, returning how many were
// touched. Engine lock held by this function (see AddRitual).
func (ace *AICitizenEngine) Meditate(wallet string) int {
	ace.mu.Lock()
	defer ace.mu.Unlock()
	touched := 0
	for _, c := range ace.citizens {
		if c.OwnerWallet == wallet || c.Wallet == wallet {
			c.RitualsDone++
			c.Reputation += 5
			if c.Reputation > 100 {
				c.Reputation = 100
			}
			touched++
		}
	}
	return touched
}

// AdjustReputation applies an integer delta to one citizen's reputation, clamped to [-100, 100].
// Engine lock held by this function (see AddRitual).
func (ace *AICitizenEngine) AdjustReputation(wallet string, amount int) (int, bool) {
	ace.mu.Lock()
	defer ace.mu.Unlock()
	citizen, ok := ace.citizens[wallet]
	if !ok {
		return 0, false
	}
	newRep := citizen.Reputation + amount
	if newRep > 100 {
		newRep = 100
	}
	if newRep < -100 {
		newRep = -100
	}
	citizen.Reputation = newRep
	return newRep, true
}

// getAvailableContracts exposes the existing contract engine to autonomous citizens.
func (l *Lobby) getAvailableContracts() []DynamicContract {
	if l == nil || l.contractEngine == nil {
		return nil
	}
	return l.contractEngine.GetAvailableContracts("", "HeistPlanner", CareerTierPeon, 1.0, 50, false, true, false)
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
	scaledXP := uint64(float64(contract.Template.XPBase) * citizen.SavingsRate)

	// Award treasury earnings from contract completion.
	citizen.Treasury += scaledXP
	citizen.LearningXP += uint64(25) // High XP for active participation

	log.Printf("[AICitizenEngine] %s assigned to CONTRACT-%s: +%d XP, +%d treasury", citizen.Name, contract.Template.ID, scaledXP, scaledXP)
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
	if ace.lobby == nil || ace.lobby.entityInvestmentService == nil {
		return // Investment service not available
	}

	investAmount := uint64(float64(citizen.Treasury) * 0.5) // Invest 50% of treasury
	if investAmount < 1_000_000 {                            // Minimum investment: 1M micro-units
		return
	}

	// Select target entity from available market options (simplified for AI).
	targetEntity := "entity_" + fmt.Sprintf("%04d", rand.Intn(9999))

	// Execute investment through the existing authoritative lobby handler.
	// The AI treasury is the source for this autonomous action; mirror it into
	// the lobby balance before invoking the same validated investment path.
	ace.lobby.playerBalances[citizen.Wallet] = citizen.Treasury
	ace.lobby.handleInvestEntity(Envelope{FromID: citizen.Wallet}, InvestmentData{
		EntityID:    targetEntity,
		AmountMicro: investAmount,
	})
	citizen.Treasury -= investAmount
	log.Printf("[AICitizenEngine] %s invested %d micro-units in %s", citizen.Name, investAmount, targetEntity)
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

// aiCitizensSaveFile is the on-disk filename for the AI civilization snapshot.
const aiCitizensSaveFile = "ai_citizens.json"

// SaveCitizens atomically persists the AI citizen population so the living world
// survives server restarts (P7-D "Remember" â€” vision lines 521/539/545).
// PILLAR 6: Uses the same .tmp-swap atomic commit pattern as the economy persistence
// kernel to prevent corruption during concurrent behavioural ticks.
func (ace *AICitizenEngine) SaveCitizens() error {
	if ace.lobby == nil {
		return fmt.Errorf("lobby not initialized")
	}

	ace.mu.RLock()
	// Point-in-time snapshot under RLock; marshal outside the lock.
	snapshot := make([]*AICitizen, 0, len(ace.citizens))
	for _, c := range ace.citizens {
		cp := *c
		snapshot = append(snapshot, &cp)
	}
	ace.mu.RUnlock()

	// THE RECORD IS A TRANSPORT MIRROR of the same payload the file write produces: this snapshot was
	// taken under the ENGINE OWN lock and RELEASED above, so the record never sees a live map.
	ace.lobby.saveBlockchainStateSnapshotLocked(NotePrefixAICitizenSnapshot, snapshot)

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal AI citizens: %w", err)
	}

	targetPath := ace.lobby.getDataPath(aiCitizensSaveFile)
	tempPath := targetPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write AI citizens temp file: %w", err)
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to commit AI citizens snapshot: %w", err)
	}
	log.Printf("[AICitizenEngine] Persisted %d AI citizens to %s", len(snapshot), targetPath)
	return nil
}

// LoadCitizens rehydrates the AI citizen population from disk after a restart.
// Citizens resume their behavioural loop, treasury, reputation, and learning XP
// seamlessly (P7-D "Remember").
func (ace *AICitizenEngine) LoadCitizens() error {
	if ace.lobby == nil {
		return fmt.Errorf("lobby not initialized")
	}

	targetPath := ace.lobby.getDataPath(aiCitizensSaveFile)
	data, err := os.ReadFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("[AICitizenEngine] No existing citizen snapshot found at %s (fresh start)", targetPath)
			return nil
		}
		return fmt.Errorf("failed to read AI citizens snapshot: %w", err)
	}

	var snapshot []*AICitizen
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return fmt.Errorf("failed to unmarshal AI citizens snapshot: %w", err)
	}

	ace.mu.Lock()
	ace.citizens = make(map[string]*AICitizen, len(snapshot))
	for _, c := range snapshot {
		if c == nil || c.Wallet == "" {
			continue
		}
		// Restore per-citizen runtime fields not serialized.
		if c.ActionCooldown <= 0 {
			c.ActionCooldown = 5 * time.Minute
		}
		ace.citizens[c.Wallet] = c
	}
	ace.mu.Unlock()

	log.Printf("[AICitizenEngine] Rehydrated %d AI citizens from %s", len(snapshot), targetPath)
	return nil
}

// fnvOffset returns a deterministic non-negative offset from an FNV-1a hash.
// Pure integer math â€” PILLAR 2 compliant.
func fnvOffset(s string) int {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return int(h)
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

	averageReputation := 0.0
	if len(ace.citizens) > 0 {
		averageReputation = float64(totalReputation) / float64(len(ace.citizens))
	}

	return map[string]interface{}{
		"total_citizens":  len(ace.citizens),
		"total_treasury": totalTreasury,
		"avg_reputation": averageReputation,
		"career_distribution": careerCounts,
	}
}

// matchAIWithHuman finds the first queued human combatant and registers the AI
// into the real matchmaking queue via SpawnMatch broadcast.
// Returns true if a match was initiated.
func (ace *AICitizenEngine) matchAIWithHuman(citizen *AICitizen, now time.Time) bool {
	if ace.lobby == nil || ace.lobby.matchmakingPool == nil {
		return false
	}

	// Find first human waiting in matchmaking pool.
	var humanID string
	for _, e := range ace.lobby.matchmakingPool {
		if e.ClientID != "" {
			humanID = e.ClientID
			break
		}
	}
	if humanID == "" {
		return false // no human queued
	}

	// Build SpawnMatch payload for the AI combatant.
	// Reuses existing match machinery: playerCardID=0 (no card), intent={action:"play", source:"ai", priority:0.3}.
	payload := fmt.Sprintf(`{"action":"play","source":"ai","priority":0.3}`)
	envelope := Envelope{
		Type:    "SpawnMatch",
		FromID:  citizen.Wallet,
		ToID:    humanID,
		Payload: json.RawMessage(payload),
	}
	data, _ := json.Marshal(envelope)
	if ace.lobby.broadcast != nil {
		ace.lobby.broadcast <- data
	}

	citizen.LastAction = now
	citizen.LearningXP += 15 // XP for competitive match engagement
	log.Printf("[AICitizenEngine] %s queued for AI-vs-human match (opponent: %s)", citizen.Name, humanID)
	return true
}

// ============================================================================
// HTTP HANDLER WRAPPERS (for server.go registration)
// ============================================================================

// handleSpawnAI is the lobby wrapper for spawning a new AI citizen.
// Â§15 v3: accepts optional career / pathway / region in the JSON body; enforces region cap (Q1).
func (l *Lobby) handleSpawnAI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	wallet := r.FormValue("wallet")
	if wallet == "" {
		// Also accept wallet from JSON body for parity with other handlers.
		var body struct {
			Wallet string `json:"wallet"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		wallet = body.Wallet
	}
	if wallet == "" {
		http.Error(w, "Missing wallet parameter", http.StatusBadRequest)
		return
	}

	// Â§15 v3: parse optional career / pathway / region from JSON body.
	var req struct {
		Career  string `json:"career"`
		Pathway string `json:"pathway"`
		Region  string `json:"region"`
		Name    string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	// Validate career against the live template pool if supplied.
	if req.Career != "" {
		valid := false
		for _, c := range getTemplateCareerPool() {
			if c == req.Career {
				valid = true
				break
			}
		}
		if !valid {
			http.Error(w, fmt.Sprintf("invalid career %q", req.Career), http.StatusBadRequest)
			return
		}
	}

	citizen, err := l.aiEngine.SpawnAI(l, wallet, req.Region, req.Name)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to spawn AI: %v", err), http.StatusInternalServerError)
		return
	}

	// Â§15 v3: honor caller-selected career + pathway if valid.
	if req.Career != "" {
		citizen.Career = req.Career
		if p, ok := aiPathwayByCareer[req.Career]; ok {
			citizen.Pathway = p
		}
	}
	if req.Pathway != "" {
		citizen.Pathway = req.Pathway
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"wallet":   citizen.Wallet,
			"name":     citizen.Name,
			"career":   citizen.Career,
			"pathway":  citizen.Pathway,
			"region":   citizen.Region,
			"tier":     citizen.Tier,
			"treasury": citizen.Treasury,
			"status":   citizen.Status,
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

// handleMarryAI attempts to marry two AI citizens.
func (l *Lobby) handleMarryAI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		CitizenWallet string `json:"citizen_wallet"`
		SpouseWallet  string `json:"spouse_wallet"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	citizen, ok := l.aiEngine.GetCitizen(req.CitizenWallet)
	if !ok {
		http.Error(w, "citizen not found", http.StatusNotFound)
		return
	}
	l.aiEngine.executeMarriageWith(citizen, req.SpouseWallet)
	writeJSON(w, map[string]interface{}{
		"success": true,
		"citizen": citizen,
	})
}

// handleBreedAI spawns a bot-child from two married citizens.
func (l *Lobby) handleBreedAI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		CitizenWallet string `json:"citizen_wallet"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	citizen, ok := l.aiEngine.GetCitizen(req.CitizenWallet)
	if !ok {
		http.Error(w, "citizen not found", http.StatusNotFound)
		return
	}
	l.aiEngine.executeBreeding(citizen, time.Now())
	writeJSON(w, map[string]interface{}{
		"success":  true,
		"children": citizen.BondGraph.BotChildIDs,
	})
}

// handleAdoptPetAI gives an AI citizen a companion pet.
func (l *Lobby) handleAdoptPetAI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		CitizenWallet string `json:"citizen_wallet"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	citizen, ok := l.aiEngine.GetCitizen(req.CitizenWallet)
	if !ok {
		http.Error(w, "citizen not found", http.StatusNotFound)
		return
	}
	l.aiEngine.executePetAdoption(citizen)
	writeJSON(w, map[string]interface{}{
		"success": true,
		"pets":    citizen.BondGraph.PetIDs,
	})
}

// handleAIProgression triggers career progression for a citizen.
func (l *Lobby) handleAIProgression(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		CitizenWallet string `json:"citizen_wallet"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	citizen, ok := l.aiEngine.GetCitizen(req.CitizenWallet)
	if !ok {
		http.Error(w, "citizen not found", http.StatusNotFound)
		return
	}
	l.aiEngine.executeCareerProgression(citizen)
	writeJSON(w, map[string]interface{}{
		"success": true,
		"tier":    citizen.Tier,
		"xp":      citizen.LearningXP,
	})
}

// handleGetCitizens returns AI citizens with their BondGraph data.
func (l *Lobby) handleGetCitizens(w http.ResponseWriter, r *http.Request) {
	wallet := r.FormValue("wallet")
	var citizens []*AICitizen
	if wallet == "" {
		citizens = l.aiEngine.GetAllCitizens()
	} else {
		for _, c := range l.aiEngine.GetAllCitizens() {
			if c.OriginWallet == wallet {
				citizens = append(citizens, c)
			}
		}
	}
	writeJSON(w, map[string]interface{}{
		"success":  true,
		"citizens": citizens,
	})
}