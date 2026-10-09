//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
)

// ── Religion Governance System ──────────────────────────────────────────────
// Faucet-owned chain store. 24-cap server-wide. All buyouts → faucet.
// Users buy governorship of one of 24 religions. Once full, only buyouts work.
// Buyout = total loss for old owner. No commission, no safety net.
// Converted users can ONLY buy out another religion to return.
// On buyout, user chooses: restore old religion OR create new one.

const (
	MaxReligionsPerServer = 24
	DefaultReligionPrice  = 5000000 // 5 $VBV in micro
)

// Religion represents a governorship-slot religion.
type Religion struct {
	ReligionID   string            `json:"religion_id"`
	Name         string            `json:"name"`
	Dogma        string            `json:"dogma"`        // "purist" | "syncretic" | "orthodox" | custom
	Governor     string            `json:"governor"`     // wallet of current owner (empty = faucet-owned)
	PricePaid    uint64            `json:"price_paid"`   // last price paid
	Members      map[string]bool   `json:"members"`      // wallet -> true (includes governor)
	Rivalries    map[string]bool   `json:"rivalries"`    // religion_id -> true
	FaithPower   uint64            `json:"faith_power"`  // computed from members + rituals
	Coherence    uint64            `json:"coherence"`    // regional FaithCoherence contribution
	RitualCount  uint64            `json:"ritual_count"` // total rituals performed
	Region       string            `json:"region"`
	CreatedAt    time.Time         `json:"created_at"`
	LastBuyoutAt time.Time         `json:"last_buyout_at"`
}

// ReligionGovernance manages all religions server-wide.
type ReligionGovernance struct {
	mu        sync.RWMutex
	Religions map[string]*Religion
}

var religionGov = &ReligionGovernance{
	Religions: make(map[string]*Religion),
}

// InitializeFaithReligions creates the first 24 faucet-owned religions.
func (rg *ReligionGovernance) InitializeFaithReligions(l *Lobby) {
	rg.mu.Lock()
	defer rg.mu.Unlock()

	if len(rg.Religions) > 0 {
		return // already initialized
	}

	defaultNames := []string{
		"Church of the Neon Seraph", "Order of the Iron Will", "Crimson Covenant",
		"Silent Congregation", "Radiant Path", "Shadow Ascendancy",
		"Eternal Flame", "Storm Callers", "Crystal Sanctum",
		"Void Walkers", "Solar Accord", "Moonlit Veil",
		"Thunder Reign", "Frost Covenant", "Ember Communion",
		"Tide Whisperers", "Stone Sentinels", "Wind Dancers",
		"Night Vigil", "Dawn Breakers", "Ash Collective",
		"Star Forgers", "Dust Makers", "Light Bearers",
	}

	defaultDogmas := []string{
		"purist", "syncretic", "orthodox", "purist", "syncretic", "orthodox",
		"purist", "syncretic", "orthodox", "purist", "syncretic", "orthodox",
		"purist", "syncretic", "orthodox", "purist", "syncretic", "orthodox",
		"purist", "syncretic", "orthodox", "purist", "syncretic", "orthodox",
	}

	for i := 0; i < MaxReligionsPerServer && i < len(defaultNames); i++ {
		rel := &Religion{
			ReligionID: fmt.Sprintf("REL-%03d", i+1),
			Name:       defaultNames[i],
			Dogma:      defaultDogmas[i],
			Governor:   "", // faucet-owned initially
			PricePaid:  DefaultReligionPrice,
			Members:    make(map[string]bool),
			Rivalries:  make(map[string]bool),
			Region:     "Base",
			CreatedAt:  time.Now(),
		}
		rg.Religions[rel.ReligionID] = rel
	}
}

// BuyGovernorship lets a user buy an unowned (faucet) religion.
func (rg *ReligionGovernance) BuyGovernorship(l *Lobby, religionID, userWallet string) (*Religion, error) {
	rg.mu.Lock()
	defer rg.mu.Unlock()

	rel, ok := rg.Religions[religionID]
	if !ok {
		return nil, fmt.Errorf("religion not found")
	}
	if rel.Governor != "" {
		return nil, fmt.Errorf("religion already owned — must buy out current governor")
	}

	price := rel.PricePaid
	if l.playerBalances[userWallet] < price {
		return nil, fmt.Errorf("insufficient balance: need %d micro", price)
	}

	l.playerBalances[userWallet] -= price
	l.faucetBalanceMicro += price

	rel.Governor = userWallet
	rel.PricePaid = price
	rel.Members[userWallet] = true
	rel.LastBuyoutAt = time.Now()

	return rel, nil
}

// BuyoutReligion lets a user buy out a governor's religion.
// ALL members auto-convert. Old governor gets NOTHING. Price → faucet.
func (rg *ReligionGovernance) BuyoutReligion(l *Lobby, religionID, buyerWallet string, newName string, restoreOld bool) (*Religion, error) {
	rg.mu.Lock()
	defer rg.mu.Unlock()

	rel, ok := rg.Religions[religionID]
	if !ok {
		return nil, fmt.Errorf("religion not found")
	}
	if rel.Governor == "" {
		return nil, fmt.Errorf("religion is faucet-owned — use BuyGovernorship")
	}
	if rel.Governor == buyerWallet {
		return nil, fmt.Errorf("already your religion")
	}

	oldGovernor := rel.Governor
	buyoutPrice := rel.PricePaid + (rel.PricePaid / 10) // 10% above last price

	if l.playerBalances[buyerWallet] < buyoutPrice {
		return nil, fmt.Errorf("insufficient balance: need %d micro", buyoutPrice)
	}

	l.playerBalances[buyerWallet] -= buyoutPrice
	l.faucetBalanceMicro += buyoutPrice // ALL buyouts → faucet

	// Auto-convert ALL members (including old governor) to new owner
	rel.Governor = buyerWallet
	rel.PricePaid = buyoutPrice
	rel.LastBuyoutAt = time.Now()

	// Rename if requested
	if newName != "" && !restoreOld {
		rel.Name = newName
	}

	// Restore old rivalries if requested
	if restoreOld {
		// Old rivalries are preserved (they belong to the religion, not the governor)
	}

	// Old governor is now a regular member (converted)
	rel.Members[oldGovernor] = true
	rel.Members[buyerWallet] = true

	// Update faction power
	rel.FaithPower = rg.computeReligionPowerLocked(rel)

	return rel, nil
}

// GetReligionsByPower returns religions sorted by power (high-tier first).
func (rg *ReligionGovernance) GetReligionsByPower() []*Religion {
	rg.mu.RLock()
	defer rg.mu.RUnlock()

	var rels []*Religion
	for _, r := range rg.Religions {
		rels = append(rels, r)
	}
	sort.SliceStable(rels, func(i, j int) bool {
		return rels[i].FaithPower > rels[j].FaithPower
	})
	return rels
}

// GetHighTierRivals returns top 12 religions by power.
func (rg *ReligionGovernance) GetHighTierRivals() []*Religion {
	rels := rg.GetReligionsByPower()
	if len(rels) > 12 {
		return rels[:12]
	}
	return rels
}

// IsConvertedUser reports whether a user has been converted (lost their religion).
func (rg *ReligionGovernance) IsConvertedUser(wallet string) bool {
	rg.mu.RLock()
	defer rg.mu.RUnlock()

	for _, rel := range rg.Religions {
		if rel.Governor == wallet {
			return false // user is still a governor
		}
	}
	// Check if user is a member but not governor
	for _, rel := range rg.Religions {
		if rel.Members[wallet] {
			return true // converted member
		}
	}
	return false
}

// AddRitual adds a ritual to a religion, raising coherence.
func (rg *ReligionGovernance) AddRitual(l *Lobby, religionID string, feeMicro uint64) error {
	rg.mu.Lock()
	defer rg.mu.Unlock()

	rel, ok := rg.Religions[religionID]
	if !ok {
		return fmt.Errorf("religion not found")
	}

	rel.RitualCount++
	rel.Coherence += 10 // per ritual
	rel.FaithPower = rg.computeReligionPowerLocked(rel)

	return nil
}

// AddMember adds a member to a religion.
func (rg *ReligionGovernance) AddMember(religionID, wallet string) error {
	rg.mu.Lock()
	defer rg.mu.Unlock()

	rel, ok := rg.Religions[religionID]
	if !ok {
		return fmt.Errorf("religion not found")
	}
	rel.Members[wallet] = true
	rel.FaithPower = rg.computeReligionPowerLocked(rel)
	return nil
}

// AddRivalry adds a rivalry between two religions.
func (rg *ReligionGovernance) AddRivalry(religionID, rivalID string) error {
	rg.mu.Lock()
	defer rg.mu.Unlock()

	rel, ok := rg.Religions[religionID]
	if !ok {
		return fmt.Errorf("religion not found")
	}
	rel.Rivalries[rivalID] = true
	return nil
}

// computeReligionPowerLocked computes faith power from members + rituals.
func (rg *ReligionGovernance) computeReligionPowerLocked(rel *Religion) uint64 {
	return uint64(len(rel.Members))*100 + rel.RitualCount*10 + rel.Coherence
}

// CountOwned returns the number of user-owned (non-faucet) religions.
func (rg *ReligionGovernance) CountOwned() int {
	rg.mu.RLock()
	defer rg.mu.RUnlock()
	count := 0
	for _, r := range rg.Religions {
		if r.Governor != "" {
			count++
		}
	}
	return count
}

// CanCreateNewReligion reports whether a new religion can be created.
func (rg *ReligionGovernance) CanCreateNewReligion() bool {
	return len(rg.Religions) < MaxReligionsPerServer
}

// ── HTTP Handlers ────────────────────────────────────────────────────────────

func (l *Lobby) handleFaithReligions(w http.ResponseWriter, r *http.Request) {
	religions := religionGov.GetReligionsByPower()
	highTier := religionGov.GetHighTierRivals()
	writeJSON(w, map[string]interface{}{
		"success":    true,
		"religions":  religions,
		"high_tier":  highTier,
		"total":      len(religions),
		"cap":        MaxReligionsPerServer,
		"owned":      religionGov.CountOwned(),
	})
}

func (l *Lobby) handleFaithReligionBuy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ReligionID string `json:"religion_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	rel, err := religionGov.BuyGovernorship(l, req.ReligionID, wallet)
	if err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true, "religion": rel})
}

func (l *Lobby) handleFaithReligionBuyout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ReligionID string `json:"religion_id"`
		NewName    string `json:"new_name"`
		RestoreOld bool   `json:"restore_old"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	rel, err := religionGov.BuyoutReligion(l, req.ReligionID, wallet, req.NewName, req.RestoreOld)
	if err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true, "religion": rel})
}

func (l *Lobby) handleFaithReligionJoin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ReligionID string `json:"religion_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if err := religionGov.AddMember(req.ReligionID, wallet); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true})
}

func (l *Lobby) handleFaithReligionRitual(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ReligionID string `json:"religion_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	fee := uint64(100000) // 0.1 $VBV per ritual
	if l.playerBalances[wallet] < fee {
		writeJSON(w, map[string]interface{}{"success": false, "error": "insufficient balance"})
		return
	}
	l.playerBalances[wallet] -= fee
	l.faucetBalanceMicro += fee
	if err := religionGov.AddRitual(l, req.ReligionID, fee); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true})
}

func (l *Lobby) handleFaithReligionRivalry(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ReligionID string `json:"religion_id"`
		RivalID    string `json:"rival_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	if err := religionGov.AddRivalry(req.ReligionID, req.RivalID); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true})
}

func (l *Lobby) handleFaithHighTier(w http.ResponseWriter, r *http.Request) {
	highTier := religionGov.GetHighTierRivals()
	writeJSON(w, map[string]interface{}{"success": true, "high_tier": highTier})
}

// Save persists religions to disk via lobby's data path.
func (rg *ReligionGovernance) Save(l *Lobby) error {
	rg.mu.RLock()
	defer rg.mu.RUnlock()
	data, err := json.MarshalIndent(rg.Religions, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.getDataPath("religions.json"), data, 0644)
}

// Load restores religions from disk. Returns count loaded.
func (rg *ReligionGovernance) Load(l *Lobby) int {
	data, err := os.ReadFile(l.getDataPath("religions.json"))
	if err != nil {
		return 0
	}
	var religions map[string]*Religion
	if err := json.Unmarshal(data, &religions); err != nil {
		return 0
	}
	rg.mu.Lock()
	rg.Religions = religions
	rg.mu.Unlock()
	return len(religions)
}

// ResolveFaithWar resolves a war between two religions.
// Winner = higher faith_power. Loser's members auto-convert to winner.
func (rg *ReligionGovernance) ResolveFaithWar(l *Lobby, religionID, challengerID string) (*Religion, *Religion, error) {
	rg.mu.Lock()
	defer rg.mu.Unlock()

	rel, ok := rg.Religions[religionID]
	if !ok {
		return nil, nil, fmt.Errorf("religion not found")
	}
	challenger, ok := rg.Religions[challengerID]
	if !ok {
		return nil, nil, fmt.Errorf("challenger religion not found")
	}

	winner, loser := rel, challenger
	if challenger.FaithPower > rel.FaithPower {
		winner, loser = challenger, rel
	}

	for member := range loser.Members {
		winner.Members[member] = true
	}
	loser.Members = make(map[string]bool)

	winner.RitualCount += loser.RitualCount
	loser.RitualCount = 0

	winner.FaithPower = rg.computeReligionPowerLocked(winner)
	loser.FaithPower = rg.computeReligionPowerLocked(loser)

	return winner, loser, nil
}
