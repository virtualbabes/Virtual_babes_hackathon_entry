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

// â”€â”€ Faith Church Storefront System â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
// Users must open a church. Storefront for faith activities.
// 24-cap. Buyouts â†’ faucet. Total loss for old owner.

// Church represents a player-owned church.
type Church struct {
	ID            string    `json:"id"`
	Owner         string    `json:"owner"`
	Name          string    `json:"name"`
	ReligionID    string    `json:"religion_id"`
	DogmaTag      string    `json:"dogma_tag"` // Purist/Syncretic/Orthodox
	FaithPower    uint64    `json:"faith_power"`
	Members       []string  `json:"members"`
	MemberCount   int       `json:"member_count"`
	OpeningCost   uint64    `json:"opening_cost"`
	RitualFees    uint64    `json:"ritual_fees"`
	TournamentPot uint64    `json:"tournament_pot"`
	ReligiousBuff uint64    `json:"religious_buff"`
	Region        string    `json:"region"`
	CreatedAt     time.Time `json:"created_at"`
	Active        bool      `json:"active"`
}

// ChurchItem represents an item sold in the church store.
type ChurchItem struct {
	ID          string    `json:"id"`
	ChurchID    string    `json:"church_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	PriceMicro  uint64    `json:"price_micro"`
	Category    string    `json:"category"` // buff, modifier, consumable
	Effect      string    `json:"effect"`
	Active      bool      `json:"active"`
}

// ChurchRitual represents a ritual performed.
type ChurchRitual struct {
	ID          string    `json:"id"`
	ChurchID    string    `json:"church_id"`
	PerformedBy string    `json:"performed_by"`
	RitualType  string    `json:"ritual_type"`
	CostMicro   uint64    `json:"cost_micro"`
	Effect      string    `json:"effect"`
	PerformedAt time.Time `json:"performed_at"`
}

// FaithChurchEngine manages churches.
type FaithChurchEngine struct {
	mu        sync.RWMutex
	churches  map[string]*Church
	items     map[string]*ChurchItem
	rituals   map[string]*ChurchRitual
}

var faithChurchEngine = &FaithChurchEngine{
	churches: make(map[string]*Church),
	items:    make(map[string]*ChurchItem),
	rituals:  make(map[string]*ChurchRitual),
}

// globalLobbyRef is set by newLobby() so faith_church.go can write to the Club registry.
var globalLobbyRef *Lobby

func SetGlobalLobbyRef(l *Lobby) { globalLobbyRef = l }
func getGlobalLobby() *Lobby     { return globalLobbyRef }

// Save persists churches, items, and rituals to disk.
func (fce *FaithChurchEngine) Save(l *Lobby) error {
	fce.mu.RLock()
	defer fce.mu.RUnlock()
	data := struct {
		Churches map[string]*Church     `json:"churches"`
		Items    map[string]*ChurchItem `json:"items"`
		Rituals  map[string]*ChurchRitual `json:"rituals"`
	}{fce.churches, fce.items, fce.rituals}
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(l.getDataPath("faith_churches.json"), bytes, 0644); err != nil {
		return err
	}
	// THE RECORD IS A TRANSPORT MIRROR of the same payload the file write produced. The faith RLock is
	// STILL HELD here (a deferred release), so the mirror is marshalled under the SAME lock that guards
	// these maps - the record can never see a live map.
	l.saveBlockchainStateSnapshotLocked(NotePrefixFaithSnapshot, data)
	return nil
}

// Load restores churches, items, and rituals from disk. Returns count of churches loaded.
func (fce *FaithChurchEngine) Load(l *Lobby) int {
	data, err := os.ReadFile(l.getDataPath("faith_churches.json"))
	if err != nil {
		return 0
	}
	var snapshot struct {
		Churches map[string]*Church     `json:"churches"`
		Items    map[string]*ChurchItem `json:"items"`
		Rituals  map[string]*ChurchRitual `json:"rituals"`
	}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return 0
	}
	fce.mu.Lock()
	fce.churches = snapshot.Churches
	fce.items = snapshot.Items
	fce.rituals = snapshot.Rituals
	fce.mu.Unlock()
	if fce.churches == nil {
		fce.churches = make(map[string]*Church)
	}
	if fce.items == nil {
		fce.items = make(map[string]*ChurchItem)
	}
	if fce.rituals == nil {
		fce.rituals = make(map[string]*ChurchRitual)
	}
	return len(fce.churches)
}

// OpenChurch opens a new church.
func (fce *FaithChurchEngine) OpenChurch(owner, name, religionID, dogmaTag string, openingCost uint64, region string) (*Church, error) {
	fce.mu.Lock()
	defer fce.mu.Unlock()

	// Check if owner already has a church
	for _, c := range fce.churches {
		if c.Owner == owner && c.Active {
			return nil, fmt.Errorf("owner already has an active church")
		}
	}

	church := &Church{
		ID:          fmt.Sprintf("church_%d", time.Now().UnixNano()),
		Owner:       owner,
		Name:        name,
		ReligionID:  religionID,
		DogmaTag:    dogmaTag,
		Members:     []string{owner},
		MemberCount: 1,
		OpeningCost: openingCost,
		Region:      region,
		CreatedAt:   time.Now(),
		Active:      true,
	}
	fce.churches[church.ID] = church
	// Â§32 Faith: mirror as a Club{Type:"Faith"} for theme engine + region views
	if l := getGlobalLobby(); l != nil {
		l.mutex.Lock()
		clubID := "faith_" + church.ID
		l.clubs[clubID] = &Club{
			ID:          clubID,
			Name:        church.Name,
			OwnerWallet: church.Owner,
			Type:        "Faith",
			DogmaTag:    church.DogmaTag,
			RegionName:  church.Region,
			CreatedAt:   church.CreatedAt,
		}
		l.mutex.Unlock()
	}
	return church, nil
}

// GetChurch returns a church by ID.
func (fce *FaithChurchEngine) GetChurch(churchID string) (*Church, bool) {
	fce.mu.RLock()
	defer fce.mu.RUnlock()
	c, ok := fce.churches[churchID]
	return c, ok
}

// GetChurchesByOwner returns all churches owned by a wallet.
func (fce *FaithChurchEngine) GetChurchesByOwner(owner string) []*Church {
	fce.mu.RLock()
	defer fce.mu.RUnlock()

	var results []*Church
	for _, c := range fce.churches {
		if c.Owner == owner && c.Active {
			results = append(results, c)
		}
	}
	return results
}

// GetChurchesByRegion returns all churches in a region.
func (fce *FaithChurchEngine) GetChurchesByRegion(region string) []*Church {
	fce.mu.RLock()
	defer fce.mu.RUnlock()

	var results []*Church
	for _, c := range fce.churches {
		if c.Region == region && c.Active {
			results = append(results, c)
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].FaithPower > results[j].FaithPower
	})
	return results
}

// AddMember adds a member to a church.
func (fce *FaithChurchEngine) AddMember(churchID, member string) error {
	fce.mu.Lock()
	defer fce.mu.Unlock()

	church, ok := fce.churches[churchID]
	if !ok {
		return fmt.Errorf("church not found")
	}

	// Check if already a member
	for _, m := range church.Members {
		if m == member {
			return fmt.Errorf("already a member")
		}
	}

	church.Members = append(church.Members, member)
	church.MemberCount = len(church.Members)
	return nil
}

// RemoveMember removes a member from a church.
func (fce *FaithChurchEngine) RemoveMember(churchID, member string) error {
	fce.mu.Lock()
	defer fce.mu.Unlock()

	church, ok := fce.churches[churchID]
	if !ok {
		return fmt.Errorf("church not found")
	}

	for i, m := range church.Members {
		if m == member {
			church.Members = append(church.Members[:i], church.Members[i+1:]...)
			church.MemberCount = len(church.Members)
			return nil
		}
	}
	return fmt.Errorf("member not found")
}

// AddItem adds an item to the church store.
func (fce *FaithChurchEngine) AddItem(churchID, name, description, category, effect string, price uint64) (*ChurchItem, error) {
	fce.mu.Lock()
	defer fce.mu.Unlock()

	item := &ChurchItem{
		ID:          fmt.Sprintf("item_%d", time.Now().UnixNano()),
		ChurchID:    churchID,
		Name:        name,
		Description: description,
		PriceMicro:  price,
		Category:    category,
		Effect:      effect,
		Active:      true,
	}
	fce.items[item.ID] = item
	return item, nil
}

// GetItems returns all items for a church.
func (fce *FaithChurchEngine) GetItems(churchID string) []*ChurchItem {
	fce.mu.RLock()
	defer fce.mu.RUnlock()

	var results []*ChurchItem
	for _, i := range fce.items {
		if i.ChurchID == churchID && i.Active {
			results = append(results, i)
		}
	}
	return results
}

// PerformRitual performs a ritual.
func (fce *FaithChurchEngine) PerformRitual(churchID, performer, ritualType string, cost uint64, effect string) (*ChurchRitual, error) {
	fce.mu.Lock()
	defer fce.mu.Unlock()

	ritual := &ChurchRitual{
		ID:          fmt.Sprintf("ritual_%d", time.Now().UnixNano()),
		ChurchID:    churchID,
		PerformedBy: performer,
		RitualType:  ritualType,
		CostMicro:   cost,
		Effect:      effect,
		PerformedAt: time.Now(),
	}
	fce.rituals[ritual.ID] = ritual

	// Increase faith power
	if church, ok := fce.churches[churchID]; ok {
		church.FaithPower += cost / 10
	}

	return ritual, nil
}

// GetRituals returns all rituals for a church.
func (fce *FaithChurchEngine) GetRituals(churchID string) []*ChurchRitual {
	fce.mu.RLock()
	defer fce.mu.RUnlock()

	var results []*ChurchRitual
	for _, r := range fce.rituals {
		if r.ChurchID == churchID {
			results = append(results, r)
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].PerformedAt.After(results[j].PerformedAt)
	})
	return results
}

// GetLeaderboard returns top churches by faith power.
func (fce *FaithChurchEngine) GetLeaderboard(n int) []*Church {
	fce.mu.RLock()
	defer fce.mu.RUnlock()

	var all []*Church
	for _, c := range fce.churches {
		if c.Active {
			all = append(all, c)
		}
	}
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].FaithPower > all[j].FaithPower
	})
	if len(all) > n {
		return all[:n]
	}
	return all
}

// â”€â”€ HTTP Handlers â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (l *Lobby) handleChurchOpen(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string `json:"name"`
		ReligionID string `json:"religion_id"`
		DogmaTag   string `json:"dogma_tag"`
		OpeningCost uint64 `json:"opening_cost"`
		Region     string `json:"region"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	church, err := faithChurchEngine.OpenChurch(wallet, req.Name, req.ReligionID, req.DogmaTag, req.OpeningCost, req.Region)
	if err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true, "church": church})
}

func (l *Lobby) handleChurchGet(w http.ResponseWriter, r *http.Request) {
	churchID := r.URL.Query().Get("church_id")
	church, ok := faithChurchEngine.GetChurch(churchID)
	if !ok {
		writeJSON(w, map[string]interface{}{"success": false, "error": "church not found"})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true, "church": church})
}

func (l *Lobby) handleChurchesByOwner(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	churches := faithChurchEngine.GetChurchesByOwner(wallet)
	writeJSON(w, map[string]interface{}{"success": true, "churches": churches})
}

func (l *Lobby) handleChurchesByRegion(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	churches := faithChurchEngine.GetChurchesByRegion(region)
	writeJSON(w, map[string]interface{}{"success": true, "churches": churches})
}

func (l *Lobby) handleChurchAddMember(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ChurchID string `json:"church_id"`
		Member   string `json:"member"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	if err := faithChurchEngine.AddMember(req.ChurchID, req.Member); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true})
}

func (l *Lobby) handleChurchRemoveMember(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ChurchID string `json:"church_id"`
		Member   string `json:"member"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	if err := faithChurchEngine.RemoveMember(req.ChurchID, req.Member); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true})
}

func (l *Lobby) handleChurchAddItem(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ChurchID    string `json:"church_id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Effect      string `json:"effect"`
		Price       uint64 `json:"price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	item, err := faithChurchEngine.AddItem(req.ChurchID, req.Name, req.Description, req.Category, req.Effect, req.Price)
	if err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true, "item": item})
}

func (l *Lobby) handleChurchItems(w http.ResponseWriter, r *http.Request) {
	churchID := r.URL.Query().Get("church_id")
	items := faithChurchEngine.GetItems(churchID)
	writeJSON(w, map[string]interface{}{"success": true, "items": items})
}

func (l *Lobby) handleChurchRitual(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ChurchID   string `json:"church_id"`
		RitualType string `json:"ritual_type"`
		Cost       uint64 `json:"cost"`
		Effect     string `json:"effect"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	ritual, err := faithChurchEngine.PerformRitual(req.ChurchID, wallet, req.RitualType, req.Cost, req.Effect)
	if err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true, "ritual": ritual})
}

func (l *Lobby) handleChurchRituals(w http.ResponseWriter, r *http.Request) {
	churchID := r.URL.Query().Get("church_id")
	rituals := faithChurchEngine.GetRituals(churchID)
	writeJSON(w, map[string]interface{}{"success": true, "rituals": rituals})
}

func (l *Lobby) handleChurchLeaderboard(w http.ResponseWriter, r *http.Request) {
	leaderboard := faithChurchEngine.GetLeaderboard(50)
	writeJSON(w, map[string]interface{}{"success": true, "leaderboard": leaderboard})
}
