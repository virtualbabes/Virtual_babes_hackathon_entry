//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"
)

// ── Entity Market System ────────────────────────────────────────────────────
// Public listings for pets, children bots, and vehicles.
// These are entity assets — distinct from AI citizens, LLM characters, and rogue bots.
// Pets and children bots compete in events for stats (not economy), but as entity
// assets they can be listed, traded, and rewarded with $VBV.

// EntityListingType distinguishes what's being listed.
type EntityListingType string

const (
	ListingTypePet         EntityListingType = "pet"
	ListingTypeChildBot    EntityListingType = "child_bot"
	ListingTypeVehicle     EntityListingType = "vehicle"
)

// EntityListing is a public market listing for an entity asset.
// Pets and children bots earn stats through events; $VBV rewards flow to owners.
type EntityListing struct {
	ListingID    string            `json:"listing_id"`
	Type         EntityListingType `json:"type"`
	EntityID     string            `json:"entity_id"`
	Owner        string            `json:"owner"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	PriceMicro   uint64            `json:"price_micro"`   // 0 = not for sale (showcase only)
	Currency     string            `json:"currency"`      // "VBV" default
	Stats        EntityStats       `json:"stats"`         // for pets/bots
	Level        uint64            `json:"level"`         // pet/bot level
	Certified    bool              `json:"certified"`     // provenance
	Mature       bool              `json:"mature"`        // eligible for tournaments
	Region       string            `json:"region"`        // deployed region
	Kind         string            `json:"kind"`          // vehicle kind
	MinLevel     uint64            `json:"min_level"`     // vehicle min level
	Bloodline    string            `json:"bloodline"`     // pet lineage
	Rarity       string            `json:"rarity"`        // common/rare/epic/legendary
	PowerScore   uint64            `json:"power_score"`   // computed effective power
	CreatedAt    time.Time         `json:"created_at"`
	ExpiresAt    time.Time         `json:"expires_at"`    // 0 = never
}

// EntityMarket manages public listings + trade history.
type EntityMarket struct {
	mu        sync.RWMutex
	listings  map[string]*EntityListing // listing_id -> listing
	tradeLog  []TradeRecord
}

// TradeRecord logs a completed trade.
type TradeRecord struct {
	ListingID    string    `json:"listing_id"`
	Seller       string    `json:"seller"`
	Buyer        string    `json:"buyer"`
	PriceMicro   uint64    `json:"price_micro"`
	EntityType   string    `json:"entity_type"`
	EntityID     string    `json:"entity_id"`
	TradedAt     time.Time `json:"traded_at"`
}

// RivalryGroup tracks rival factions (vehicle builders vs breeders).
type RivalryGroup struct {
	mu       sync.RWMutex
	Groups   map[string]*Faction // faction_id -> faction
}

type Faction struct {
	FactionID   string            `json:"faction_id"`
	Name        string            `json:"name"`
	Kind        string            `json:"kind"` // "vehicle_builders" | "breeders"
	Members     map[string]bool   `json:"members"` // wallet -> true
	Reputation  uint64            `json:"reputation"`
	PowerScore  uint64            `json:"power_score"`
	Region      string            `json:"region"`
	CreatedAt   time.Time         `json:"created_at"`
}

var (
	entityMarket  = &EntityMarket{listings: make(map[string]*EntityListing)}
	rivalryGroups = &RivalryGroup{Groups: make(map[string]*Faction)}
)

// ── Pet Battle Arena ─────────────────────────────────────────────────────────
// Pets battle in events for stats. Stat overlay rewards apply to 3D world dynamics.
// Rivalry engine stacks: vehicle builders vs breeders.

// PetBattle is one battle instance between two pets.
type PetBattle struct {
	BattleID    string    `json:"battle_id"`
	Challenger  string    `json:"challenger"`   // pet_id
	Defender    string    `json:"defender"`     // pet_id
	ChallengerOwner string `json:"challenger_owner"`
	DefenderOwner   string `json:"defender_owner"`
	Region      string    `json:"region"`
	Status      string    `json:"status"`       // "pending" | "active" | "resolved"
	Winner      string    `json:"winner"`       // pet_id or "draw"
	ChallengerStats EntityStats `json:"challenger_stats"`
	DefenderStats   EntityStats `json:"defender_stats"`
	StatDelta   int       `json:"stat_delta"`   // signed delta applied to winner
	RewardMicro uint64    `json:"reward_micro"` // $VBV reward to winner's owner
	CreatedAt   time.Time `json:"created_at"`
	ResolvedAt  time.Time `json:"resolved_at"`
}

// PetBattleArena manages active battles.
type PetBattleArena struct {
	mu       sync.RWMutex
	battles  map[string]*PetBattle
}

var petBattleArena = &PetBattleArena{battles: make(map[string]*PetBattle)}

// ── Children Bots ────────────────────────────────────────────────────────────
// Distinct from AI citizens — children bots are derived AICitizen records
// with OriginWallet = parent, reduced Tier. They cannot compete like AI citizens
// or rogue bots, but they CAN battle in pet events and be listed on the market.

// IsChildBot reports whether a citizen is a child bot (derived, not rogue).
func (c *AICitizen) IsChildBot() bool {
	return c.OriginWallet != "" && c.OriginWallet != c.Wallet
}

// ── Entity Market Functions ──────────────────────────────────────────────────

// CreateListing creates a public listing for an entity asset.
func (em *EntityMarket) CreateListing(l *Lobby, owner string, entityID string, listingType EntityListingType, priceMicro uint64, description string, durationHours int) (*EntityListing, error) {
	em.mu.Lock()
	defer em.mu.Unlock()

	// Verify ownership
	switch listingType {
	case ListingTypePet:
		if pet, ok := l.pets[entityID]; ok {
			if pet.Owner != owner {
				return nil, fmt.Errorf("not owner of pet %s", entityID)
			}
		} else {
			return nil, fmt.Errorf("pet %s not found", entityID)
		}
	case ListingTypeChildBot:
		if c, ok := l.aiEngine.GetCitizen(entityID); ok {
			if !c.IsChildBot() {
				return nil, fmt.Errorf("citizen %s is not a child bot", entityID)
			}
			if c.OriginWallet != owner && c.OwnerWallet != owner {
				return nil, fmt.Errorf("not owner of child bot %s", entityID)
			}
		} else {
			return nil, fmt.Errorf("child bot %s not found", entityID)
		}
	case ListingTypeVehicle:
		if veh, ok := l.vehicles[entityID]; ok {
			if veh.Owner != owner {
				return nil, fmt.Errorf("not owner of vehicle %s", entityID)
			}
		} else {
			return nil, fmt.Errorf("vehicle %s not found", entityID)
		}
	}

	listing := &EntityListing{
		ListingID:  fmt.Sprintf("LST-%d", time.Now().UnixNano()),
		Type:       listingType,
		EntityID:   entityID,
		Owner:      owner,
		PriceMicro: priceMicro,
		Currency:   "VBV",
		CreatedAt:  time.Now(),
	}

	if durationHours > 0 {
		listing.ExpiresAt = time.Now().Add(time.Duration(durationHours) * time.Hour)
	}

	// Populate entity-specific fields
	switch listingType {
	case ListingTypePet:
		if pet, ok := l.pets[entityID]; ok {
			listing.Name = pet.Name
			listing.Stats = pet.Stats
			listing.Level = pet.PetLevel
			listing.Certified = pet.Certified
			listing.Mature = pet.Mature
			listing.Region = pet.Region
			listing.Bloodline = fmt.Sprintf("%s x %s", pet.SireID, pet.DamID)
			listing.Rarity = computeRarity(pet.Traits)
			listing.PowerScore = ComputeEffectivePowerLevel(pet.PetLevel, pet.Stats)
		}
	case ListingTypeChildBot:
		if c, ok := l.aiEngine.GetCitizen(entityID); ok {
			listing.Name = c.Name
			listing.Stats = c.Stats
			listing.Level = uint64(c.BotLevel)
			listing.Certified = c.Certified
			listing.Mature = c.Certified && c.BotLevel > 0
			listing.Region = c.Region
			listing.Rarity = computeBotRarity(c.Tier)
			listing.PowerScore = ComputeEffectivePowerLevel(uint64(c.BotLevel), c.Stats)
		}
	case ListingTypeVehicle:
		if veh, ok := l.vehicles[entityID]; ok {
			level := veh.VehicleLevel
			if level == 0 {
				level = 1
			}
			listing.Name = veh.Name
			listing.Kind = veh.Kind
			listing.MinLevel = veh.MinLevel
			// §25.6.1: the listing reports the vehicle's real BUILD, not its spawn gate.
			// Previously Level was MinLevel and Stats were never set, so PowerScore stayed 0
			// and a vehicle could not be compared with a pet or bot at all.
			listing.Stats = veh.Stats
			listing.Level = level
			// Broken-in is the vehicle's 'mature' analogue (race/tournament eligibility).
			listing.Mature = vehicleBreakInCompleteLocked(veh)
			listing.Certified = veh.Certified && !veh.BlackMarketAdopted
			listing.Region = veh.Region
			listing.Bloodline = veh.Kind + " build"
			listing.Rarity = vehicleRarity(veh.Kind)
			listing.PowerScore = ComputeEffectivePowerLevel(level, veh.Stats)
		}
	}

	em.listings[listing.ListingID] = listing
	return listing, nil
}

// ListListings returns all active listings, filtered by type.
func (em *EntityMarket) ListListings(listingType EntityListingType, region string) []*EntityListing {
	em.mu.RLock()
	defer em.mu.RUnlock()

	now := time.Now()
	var results []*EntityListing
	for _, l := range em.listings {
		if listingType != "" && l.Type != listingType {
			continue
		}
		if region != "" && l.Region != region {
			continue
		}
		if !l.ExpiresAt.IsZero() && l.ExpiresAt.Before(now) {
			continue
		}
		results = append(results, l)
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})
	return results
}

// PurchaseListing completes a trade.
func (em *EntityMarket) PurchaseListing(l *Lobby, listingID, buyer string) (*TradeRecord, error) {
	em.mu.Lock()
	defer em.mu.Unlock()

	listing, ok := em.listings[listingID]
	if !ok {
		return nil, fmt.Errorf("listing not found")
	}
	if listing.Owner == buyer {
		return nil, fmt.Errorf("cannot buy own listing")
	}
	if listing.PriceMicro == 0 {
		return nil, fmt.Errorf("listing is showcase-only (not for sale)")
	}
	if !listing.ExpiresAt.IsZero() && listing.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("listing expired")
	}

	// Buyer pays
	if l.playerBalances[buyer] < listing.PriceMicro {
		return nil, fmt.Errorf("insufficient balance")
	}
	l.playerBalances[buyer] -= listing.PriceMicro
	l.playerBalances[listing.Owner] += listing.PriceMicro

	// Transfer entity ownership
	switch listing.Type {
	case ListingTypePet:
		if pet, ok := l.pets[listing.EntityID]; ok {
			pet.Owner = buyer
		}
	case ListingTypeChildBot:
		if c, ok := l.aiEngine.GetCitizen(listing.EntityID); ok {
			c.OwnerWallet = buyer
		}
	case ListingTypeVehicle:
		if veh, ok := l.vehicles[listing.EntityID]; ok {
			veh.Owner = buyer
		}
	}

	record := TradeRecord{
		ListingID:  listing.ListingID,
		Seller:     listing.Owner,
		Buyer:      buyer,
		PriceMicro: listing.PriceMicro,
		EntityType: string(listing.Type),
		EntityID:   listing.EntityID,
		TradedAt:   time.Now(),
	}
	em.tradeLog = append(em.tradeLog, record)
	delete(em.listings, listingID)
	return &record, nil
}

// ── Rivalry Groups ───────────────────────────────────────────────────────────

// JoinFaction adds a wallet to a rivalry group.
func (rg *RivalryGroup) JoinFaction(factionID, wallet string) error {
	rg.mu.Lock()
	defer rg.mu.Unlock()

	faction, ok := rg.Groups[factionID]
	if !ok {
		return fmt.Errorf("faction not found")
	}
	faction.Members[wallet] = true
	return nil
}

// CreateFaction creates a new rivalry group.
func (rg *RivalryGroup) CreateFaction(factionID, name, kind, region string) *Faction {
	rg.mu.Lock()
	defer rg.mu.Unlock()

	faction := &Faction{
		FactionID: factionID,
		Name:      name,
		Kind:      kind,
		Members:   make(map[string]bool),
		Region:    region,
		CreatedAt: time.Now(),
	}
	rg.Groups[factionID] = faction
	return faction
}

// ── Pet Battle Arena ─────────────────────────────────────────────────────────

// ChallengePet creates a pet battle challenge.
func (ba *PetBattleArena) ChallengePet(l *Lobby, challengerID, defenderID, region string) (*PetBattle, error) {
	ba.mu.Lock()
	defer ba.mu.Unlock()

	challenger, ok := l.pets[challengerID]
	if !ok {
		return nil, fmt.Errorf("challenger pet not found")
	}
	defender, ok := l.pets[defenderID]
	if !ok {
		return nil, fmt.Errorf("defender pet not found")
	}
	if !challenger.Mature || !defender.Mature {
		return nil, fmt.Errorf("both pets must be mature")
	}

	battle := &PetBattle{
		BattleID:        fmt.Sprintf("BAT-%d", time.Now().UnixNano()),
		Challenger:      challengerID,
		Defender:        defenderID,
		ChallengerOwner: challenger.Owner,
		DefenderOwner:   defender.Owner,
		Region:          region,
		Status:          "active",
		ChallengerStats: challenger.Stats,
		DefenderStats:   defender.Stats,
		CreatedAt:       time.Now(),
	}
	ba.battles[battle.BattleID] = battle
	return battle, nil
}

// ResolveBattle determines winner based on power score, applies stat delta + reward.
func (ba *PetBattleArena) ResolveBattle(l *Lobby, battleID string) (*PetBattle, error) {
	ba.mu.Lock()
	defer ba.mu.Unlock()

	battle, ok := ba.battles[battleID]
	if !ok {
		return nil, fmt.Errorf("battle not found")
	}
	if battle.Status == "resolved" {
		return nil, fmt.Errorf("battle already resolved")
	}

	challengerPower := ComputeEffectivePowerLevel(l.pets[battle.Challenger].PetLevel, battle.ChallengerStats)
	defenderPower := ComputeEffectivePowerLevel(l.pets[battle.Defender].PetLevel, battle.DefenderStats)

	// Apply rivalry bonus: vehicle builders vs breeders
	rivalryBonus := computeRivalryBonus(l, battle)

	if challengerPower+uint64(rivalryBonus) > defenderPower {
		battle.Winner = battle.Challenger
		battle.StatDelta = 2
		battle.RewardMicro = 500000 // 0.5 $VBV to winner's owner
	} else if defenderPower > challengerPower+uint64(rivalryBonus) {
		battle.Winner = battle.Defender
		battle.StatDelta = 2
		battle.RewardMicro = 500000
	} else {
		battle.Winner = "draw"
		battle.StatDelta = 1
	}

	// Apply stat delta to winner
	if battle.Winner == battle.Challenger {
		applyStatDelta(&l.pets[battle.Challenger].Stats, battle.StatDelta)
		l.playerBalances[battle.ChallengerOwner] += battle.RewardMicro
	} else if battle.Winner == battle.Defender {
		applyStatDelta(&l.pets[battle.Defender].Stats, battle.StatDelta)
		l.playerBalances[battle.DefenderOwner] += battle.RewardMicro
	}

	battle.Status = "resolved"
	battle.ResolvedAt = time.Now()
	return battle, nil
}

// computeRivalryBonus applies vehicle-builder vs breeder rivalry.
func computeRivalryBonus(l *Lobby, battle *PetBattle) int {
	bonus := 0
	// Check if owners are in rival factions
	rivalryGroups.mu.RLock()
	for _, faction := range rivalryGroups.Groups {
		challengerInFaction := faction.Members[battle.ChallengerOwner]
		defenderInFaction := faction.Members[battle.DefenderOwner]
		if challengerInFaction && !defenderInFaction {
			bonus += 5 // rivalry advantage
		} else if defenderInFaction && !challengerInFaction {
			bonus -= 5
		}
	}
	rivalryGroups.mu.RUnlock()
	return bonus
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func computeRarity(traits uint64) string {
	if traits > 80 {
		return "legendary"
	} else if traits > 60 {
		return "epic"
	} else if traits > 40 {
		return "rare"
	}
	return "common"
}

func computeBotRarity(tier int) string {
	if tier >= 5 {
		return "legendary"
	} else if tier >= 4 {
		return "epic"
	} else if tier >= 2 {
		return "rare"
	}
	return "common"
}

// ── HTTP Handlers ────────────────────────────────────────────────────────────

func (l *Lobby) handleEntityMarketList(w http.ResponseWriter, r *http.Request) {
	listingType := EntityListingType(r.URL.Query().Get("type"))
	region := r.URL.Query().Get("region")
	listings := entityMarket.ListListings(listingType, region)
	writeJSON(w, map[string]interface{}{"success": true, "listings": listings})
}

func (l *Lobby) handleEntityMarketCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EntityID      string `json:"entity_id"`
		Type          string `json:"type"`
		PriceMicro    uint64 `json:"price_micro"`
		Description   string `json:"description"`
		DurationHours int    `json:"duration_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	listing, err := entityMarket.CreateListing(l, wallet, req.EntityID, EntityListingType(req.Type), req.PriceMicro, req.Description, req.DurationHours)
	if err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true, "listing": listing})
}

func (l *Lobby) handleEntityMarketPurchase(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ListingID string `json:"listing_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	record, err := entityMarket.PurchaseListing(l, req.ListingID, wallet)
	if err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true, "trade": record})
}

func (l *Lobby) handlePetBattleList(w http.ResponseWriter, r *http.Request) {
	petBattleArena.mu.RLock()
	defer petBattleArena.mu.RUnlock()
	battles := make([]*PetBattle, 0, len(petBattleArena.battles))
	for _, b := range petBattleArena.battles {
		battles = append(battles, b)
	}
	writeJSON(w, map[string]interface{}{"success": true, "battles": battles})
}

func (l *Lobby) handlePetBattleChallenge(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ChallengerID string `json:"challenger_id"`
		DefenderID   string `json:"defender_id"`
		Region       string `json:"region"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	region := req.Region
	if region == "" {
		region = "Base"
	}
	battle, err := petBattleArena.ChallengePet(l, req.ChallengerID, req.DefenderID, region)
	if err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true, "battle": battle})
}

func (l *Lobby) handlePetBattleResolve(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BattleID string `json:"battle_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	battle, err := petBattleArena.ResolveBattle(l, req.BattleID)
	if err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true, "battle": battle})
}

func (l *Lobby) handleRivalryFactions(w http.ResponseWriter, r *http.Request) {
	rivalryGroups.mu.RLock()
	defer rivalryGroups.mu.RUnlock()
	writeJSON(w, map[string]interface{}{"success": true, "factions": rivalryGroups.Groups})
}

func (l *Lobby) handleRivalryJoin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FactionID string `json:"faction_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if err := rivalryGroups.JoinFaction(req.FactionID, wallet); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true})
}

func (l *Lobby) handleChildrenBots(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	var bots []*AICitizen
	if l.aiEngine != nil {
		for _, c := range l.aiEngine.GetAllCitizens() {
			if c.IsChildBot() && (c.OwnerWallet == wallet || c.OriginWallet == wallet) {
				bots = append(bots, c)
			}
		}
	}
	writeJSON(w, map[string]interface{}{"success": true, "children_bots": bots})
}

func init() {
	// Seed rivalry factions
	rivalryGroups.CreateFaction("vehicle_builders", "Vehicle Builders Guild", "vehicle_builders", "Base")
	rivalryGroups.CreateFaction("breeders", "Master Breeders Alliance", "breeders", "Base")
}
