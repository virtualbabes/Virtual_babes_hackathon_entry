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

// ── Creator Economy System ───────────────────────────────────────────────────
// Creators become first-class citizens. Launching products, selling DLC,
// leasing infrastructure, running businesses, hiring employees, receiving royalties,
// creating events, building communities.

// DLCPack represents a downloadable content pack.
type DLCPack struct {
	ID            string    `json:"id"`
	Creator       string    `json:"creator"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	PriceMicro    uint64    `json:"price_micro"`
	Category      string    `json:"category"`
	SalesCount    uint64    `json:"sales_count"`
	RevenueTotal  uint64    `json:"revenue_total"`
	CreatedAt     time.Time `json:"created_at"`
	Active        bool      `json:"active"`
}

// CreatorRoyalty tracks royalty payments.
type CreatorRoyalty struct {
	Creator       string    `json:"creator"`
	Amount        uint64    `json:"amount"`
	Source        string    `json:"source"`
	PaidAt        time.Time `json:"paid_at"`
}

// CreatorEvent represents a creator-hosted event.
type CreatorEvent struct {
	ID            string    `json:"id"`
	Creator       string    `json:"creator"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	EventDate     time.Time `json:"event_date"`
	PriceMicro    uint64    `json:"price_micro"`
	Attendees     []string  `json:"attendees"`
	MaxAttendees  int       `json:"max_attendees"`
	CreatedAt     time.Time `json:"created_at"`
}

// Subscription represents a recurring payment.
type Subscription struct {
	ID            string    `json:"id"`
	Subscriber    string    `json:"subscriber"`
	Creator       string    `json:"creator"`
	AmountMicro   uint64    `json:"amount_micro"`
	Frequency     string    `json:"frequency"` // monthly, weekly
	NextPayment   time.Time `json:"next_payment"`
	Active        bool      `json:"active"`
}

// CreatorEconomy manages creator systems.
type CreatorEconomy struct {
	mu            sync.RWMutex
	dlcs          map[string]*DLCPack
	royalties     map[string][]*CreatorRoyalty
	events        map[string]*CreatorEvent
	subscriptions map[string]*Subscription
}

var creatorEconomy = &CreatorEconomy{
	dlcs:          make(map[string]*DLCPack),
	royalties:     make(map[string][]*CreatorRoyalty),
	events:        make(map[string]*CreatorEvent),
	subscriptions: make(map[string]*Subscription),
}

// CreateDLC creates a new DLC pack.
func (ce *CreatorEconomy) CreateDLC(creator, title, description, category string, price uint64) *DLCPack {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	dlc := &DLCPack{
		ID:          fmt.Sprintf("dlc_%d", time.Now().UnixNano()),
		Creator:     creator,
		Title:       title,
		Description: description,
		PriceMicro:  price,
		Category:    category,
		CreatedAt:   time.Now(),
		Active:      true,
	}
	ce.dlcs[dlc.ID] = dlc
	return dlc
}

// PurchaseDLC purchases a DLC pack.
func (ce *CreatorEconomy) PurchaseDLC(dlcID, buyer string) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	dlc, ok := ce.dlcs[dlcID]
	if !ok || !dlc.Active {
		return fmt.Errorf("DLC not found")
	}

	dlc.SalesCount++
	dlc.RevenueTotal += dlc.PriceMicro

	// Add royalty
	ce.royalties[dlc.Creator] = append(ce.royalties[dlc.Creator], &CreatorRoyalty{
		Creator: dlc.Creator,
		Amount:  dlc.PriceMicro * 90 / 100, // 90% to creator
		Source:  fmt.Sprintf("dlc:%s", dlcID),
		PaidAt:  time.Now(),
	})

	return nil
}

// GetDLCs returns all DLCs.
func (ce *CreatorEconomy) GetDLCs() []*DLCPack {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	var results []*DLCPack
	for _, d := range ce.dlcs {
		if d.Active {
			results = append(results, d)
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].SalesCount > results[j].SalesCount
	})
	return results
}

// GetRoyalties returns royalties for a creator.
func (ce *CreatorEconomy) GetRoyalties(creator string) []*CreatorRoyalty {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	return ce.royalties[creator]
}

// CreateEvent creates a creator event.
func (ce *CreatorEconomy) CreateEvent(creator, title, description string, eventDate time.Time, price uint64, maxAttendees int) *CreatorEvent {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	event := &CreatorEvent{
		ID:           fmt.Sprintf("event_%d", time.Now().UnixNano()),
		Creator:      creator,
		Title:        title,
		Description:  description,
		EventDate:    eventDate,
		PriceMicro:   price,
		MaxAttendees: maxAttendees,
		CreatedAt:    time.Now(),
	}
	ce.events[event.ID] = event
	return event
}

// AttendEvent attends a creator event.
func (ce *CreatorEconomy) AttendEvent(eventID, attendee string) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	event, ok := ce.events[eventID]
	if !ok {
		return fmt.Errorf("event not found")
	}

	if len(event.Attendees) >= event.MaxAttendees {
		return fmt.Errorf("event full")
	}

	event.Attendees = append(event.Attendees, attendee)
	return nil
}

// GetEvents returns all creator events.
func (ce *CreatorEconomy) GetEvents() []*CreatorEvent {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	var results []*CreatorEvent
	for _, e := range ce.events {
		results = append(results, e)
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].EventDate.Before(results[j].EventDate)
	})
	return results
}

// CreateSubscription creates a subscription.
func (ce *CreatorEconomy) CreateSubscription(subscriber, creator string, amount uint64, frequency string) *Subscription {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	sub := &Subscription{
		ID:          fmt.Sprintf("sub_%d", time.Now().UnixNano()),
		Subscriber:  subscriber,
		Creator:     creator,
		AmountMicro: amount,
		Frequency:   frequency,
		NextPayment: time.Now().AddDate(0, 1, 0),
		Active:      true,
	}
	ce.subscriptions[sub.ID] = sub
	return sub
}

// GetSubscriptions returns subscriptions for a subscriber.
func (ce *CreatorEconomy) GetSubscriptions(subscriber string) []*Subscription {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	var results []*Subscription
	for _, s := range ce.subscriptions {
		if s.Subscriber == subscriber && s.Active {
			results = append(results, s)
		}
	}
	return results
}

// ── HTTP Handlers ────────────────────────────────────────────────────────────

func (l *Lobby) handleCreatorDLCCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Price       uint64 `json:"price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	dlc := creatorEconomy.CreateDLC(wallet, req.Title, req.Description, req.Category, req.Price)
	writeJSON(w, map[string]interface{}{"success": true, "dlc": dlc})
}

func (l *Lobby) handleCreatorDLCPurchase(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DLCID string `json:"dlc_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if err := creatorEconomy.PurchaseDLC(req.DLCID, wallet); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true})
}

func (l *Lobby) handleCreatorDLCs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{"success": true, "dlcs": creatorEconomy.GetDLCs()})
}

func (l *Lobby) handleCreatorRoyalties(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	royalties := creatorEconomy.GetRoyalties(wallet)
	writeJSON(w, map[string]interface{}{"success": true, "royalties": royalties})
}

func (l *Lobby) handleCreatorEventCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title        string `json:"title"`
		Description  string `json:"description"`
		EventDate    string `json:"event_date"`
		Price        uint64 `json:"price"`
		MaxAttendees int    `json:"max_attendees"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	eventDate, _ := time.Parse("2006-01-02", req.EventDate)
	event := creatorEconomy.CreateEvent(wallet, req.Title, req.Description, eventDate, req.Price, req.MaxAttendees)
	writeJSON(w, map[string]interface{}{"success": true, "event": event})
}

func (l *Lobby) handleCreatorEvents(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{"success": true, "events": creatorEconomy.GetEvents()})
}

func (l *Lobby) handleCreatorEventAttend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EventID string `json:"event_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if err := creatorEconomy.AttendEvent(req.EventID, wallet); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true})
}

func (l *Lobby) handleCreatorSubCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Creator   string `json:"creator"`
		Amount    uint64 `json:"amount"`
		Frequency string `json:"frequency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	sub := creatorEconomy.CreateSubscription(wallet, req.Creator, req.Amount, req.Frequency)
	writeJSON(w, map[string]interface{}{"success": true, "subscription": sub})
}

func (l *Lobby) handleCreatorSubs(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	subs := creatorEconomy.GetSubscriptions(wallet)
	writeJSON(w, map[string]interface{}{"success": true, "subscriptions": subs})
}
