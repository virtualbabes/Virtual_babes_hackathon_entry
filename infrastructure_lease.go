//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// ── Infrastructure Leasing System ────────────────────────────────────────────
// Developers rent mature systems. Businesses rent economic systems.
// Communities rent social systems.

type InfrastructureLease struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	SystemType    string    `json:"system_type"`
	MonthlyRate   uint64    `json:"monthly_rate"`
	LeaseStart    time.Time `json:"lease_start"`
	LeaseEnd      time.Time `json:"lease_end"`
	Active        bool      `json:"active"`
	AutoRenew     bool      `json:"auto_renew"`
}

type LeaseEngine struct {
	mu     sync.RWMutex
	leases map[string]*InfrastructureLease
}

var leaseEngine = &LeaseEngine{
	leases: make(map[string]*InfrastructureLease),
}

// ── THE TENANT WORLD LEASE — DESIGN RECORD (spawning DEFERRED) ───────────────
//
// WHAT A TENANT WORLD IS. A tenant world is a player's OWN instance of this civilization: its own
// network + asset/app ids, its own reward token (the RewardToken role `tenant`), and its own
// resource meter, leased from the house and billed per period. Nothing about that exists yet, so
// this record is a DESIGN only, and the door that would create one REFUSES with the reasons listed
// below. It exists so the shape is decided in ONE place instead of being invented later by
// whichever caller needs it first.
//
// DELIBERATELY NOT IMPLEMENTED (each one is a blocker, not an oversight):
//   * SPAWNING a world (no tenant world runtime exists; nothing can be started).
//   * An in-app WORLD SELECTOR (a player cannot choose, enter or switch a world).
//   * PER-TENANT token/network wiring (a tenant token can be REGISTERED today, but no world bills).
//   * GZIP METER ENFORCEMENT (MeterBasisBytes is a declared basis for a future meter, never read).
//   * MAINTENANCE PAY / REVOCATION (no billing run, so no suspension or eviction can be honest).
//
// THE PRICE IS NEVER THE CLIENT'S. A create request may name a system type and a duration; a rate
// supplied by a caller is IGNORED (stated in the refusal), because every price in this codebase is
// server-authoritative and served (the bonded-asset market's rule, kept here verbatim).
type TenantWorldLease struct {
	LeaseID         string    `json:"lease_id"`
	TenantWallet    string    `json:"tenant_wallet"`
	WorldID         string    `json:"world_id"`          // assigned by the spawner, never by a caller
	Network         string    `json:"network"`           // the tenant world's settlement network
	AssetID         string    `json:"asset_id"`          // the tenant's reward token
	AppID           string    `json:"app_id"`            // the tenant world's application id
	RateMicro       uint64    `json:"rate_micro"`        // SERVER-set, integer micro units
	PeriodSeconds   int64     `json:"period_seconds"`
	MeterBasisBytes uint64    `json:"meter_basis_bytes"` // declared basis for a future meter
	State           string    `json:"state"`             // proposed|active|suspended|revoked
	Spawned         bool      `json:"spawned"`           // always false: nothing spawns yet
	CreatedAt       time.Time `json:"created_at"`
}

// TenantWorldLeaseStates is the whole lifecycle vocabulary. `active` is unreachable today: a
// record cannot leave `proposed` because no spawner and no billing run exist.
var TenantWorldLeaseStates = []string{"proposed", "active", "suspended", "revoked"}

// TenantWorldLeaseBlockers names, in one place, everything that must exist before a tenant world
// lease can be created. The refusal and the served catalogue both read THIS list, so the surface
// and the door can never disagree about why leasing is unavailable.
func TenantWorldLeaseBlockers() []string {
	return []string{
		"no tenant world runtime exists: nothing can spawn, start or host a world",
		"no in-app world selector: a player cannot choose, enter or switch a world",
		"no per-tenant token/network wiring: a registered tenant token is not billed by anything",
		"no gzip meter enforcement: a declared meter basis is never read",
		"no billing run: a period cannot be charged, renewed, suspended or revoked",
	}
}

// assertTenantWorldLeaseDialect fails the BOOT if the lifecycle vocabulary drifts, so a state can
// never be added on one side (a served value) without the other (the states list) knowing it.
func assertTenantWorldLeaseDialect() {
	if len(TenantWorldLeaseStates) == 0 {
		panic("tenant world lease: the lifecycle vocabulary is empty")
	}
	seen := make(map[string]bool, len(TenantWorldLeaseStates))
	for _, state := range TenantWorldLeaseStates {
		if strings.TrimSpace(state) == "" {
			panic("tenant world lease: an empty lifecycle state would be unreachable and unservable")
		}
		if seen[state] {
			panic("tenant world lease: duplicate lifecycle state " + state)
		}
		seen[state] = true
	}
	if !seen["proposed"] {
		panic("tenant world lease: `proposed` is the only reachable state today and must be declared")
	}
	if len(TenantWorldLeaseBlockers()) == 0 {
		panic("tenant world lease: the refusal would name no reason — that is a lie in reverse")
	}
}

func init() { assertTenantWorldLeaseDialect() }

// CreateLease REFUSES, at the DOMAIN owner, while the tenant world record is design-only.
//
// The refusal lives here rather than in the HTTP handler so no caller — a handler, a bot, a future
// spawner — can create a lease by going around the door. The caller-supplied rate is deliberately
// not used for anything: a price is never the client's.
func (e *LeaseEngine) CreateLease(tenantID, systemType string, rateMicro uint64, durationDays int) (*InfrastructureLease, error) {
	return nil, fmt.Errorf("a tenant world lease cannot be created yet (%s); the request's rate (%d micro) is IGNORED because every rate is the server's",
		strings.Join(TenantWorldLeaseBlockers(), "; "), rateMicro)
}

func (e *LeaseEngine) GetLeases(tenantID string) []*InfrastructureLease {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var results = make([]*InfrastructureLease, 0)
	for _, l := range e.leases {
		if l.TenantID == tenantID {
			results = append(results, l)
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].LeaseStart.After(results[j].LeaseStart)
	})
	return results
}

func (e *LeaseEngine) GetAvailableSystems() []map[string]interface{} {
	return []map[string]interface{}{
		{"type": "wallet", "name": "Wallet System", "description": "Self-custody wallet integration", "monthly_rate": 1000000},
		{"type": "auth", "name": "Auth System", "description": "Identity verification + login", "monthly_rate": 500000},
		{"type": "economy", "name": "Economy Engine", "description": "Full economic simulation", "monthly_rate": 2000000},
		{"type": "ai", "name": "AI Citizens", "description": "AI citizen spawning + management", "monthly_rate": 1500000},
		{"type": "social", "name": "Social Systems", "description": "Clubs, factions, alliances", "monthly_rate": 800000},
		{"type": "tournament", "name": "Tournament Manager", "description": "Bracket + reward system", "monthly_rate": 1200000},
		{"type": "leaderboard", "name": "Leaderboards", "description": "Global + regional rankings", "monthly_rate": 600000},
		{"type": "crosschain", "name": "Cross-Chain Bridge", "description": "Multi-chain asset routing", "monthly_rate": 2500000},
		{"type": "creator", "name": "Creator Store", "description": "Product listing + sales", "monthly_rate": 900000},
		{"type": "analytics", "name": "Analytics", "description": "Player behavior + metrics", "monthly_rate": 700000},
	}
}

// leaseSystemIsCatalogued reports whether a system type may be leased at all. The catalogue is the
// owner of that vocabulary, so a request cannot invent a system to lease.
func leaseSystemIsCatalogued(systemType string) bool {
	for _, system := range leaseEngine.GetAvailableSystems() {
		if kind, ok := system["type"].(string); ok && strings.EqualFold(kind, systemType) {
			return true
		}
	}
	return false
}

// handleLeaseCreate REFUSES with a stated reason. It validates the request first, so a malformed
// request is distinguished from the structural refusal, and it never reads a rate from the body: a
// client-declared price is REPORTED as ignored, never honoured.
func (l *Lobby) handleLeaseCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SystemType  string `json:"system_type"`
		MonthlyRate uint64 `json:"monthly_rate"` // legacy clients send this; it is IGNORED
		Duration    int    `json:"duration_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body: expected JSON naming a system_type and a duration_days"})
		return
	}
	wallet := l.getWalletFromRequest(r)

	switch {
	case strings.TrimSpace(req.SystemType) == "":
		writeJSON(w, map[string]interface{}{"success": false, "error": "a lease must name the system type it is for"})
		return
	case req.Duration <= 0:
		writeJSON(w, map[string]interface{}{"success": false, "error": "a lease must name a positive duration in days"})
		return
	case !leaseSystemIsCatalogued(req.SystemType):
		writeJSON(w, map[string]interface{}{"success": false, "error": fmt.Sprintf("%q is not a catalogued system: read the served catalogue before requesting a lease", req.SystemType)})
		return
	case strings.TrimSpace(wallet) == "":
		writeJSON(w, map[string]interface{}{"success": false, "error": "a lease needs a tenant wallet"})
		return
	}

	// The DOMAIN owner refuses; this handler only reports it. HTTP 409: the request is well formed
	// and the world it would lease does not exist.
	_, err := leaseEngine.CreateLease(wallet, req.SystemType, req.MonthlyRate, req.Duration)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusConflict)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":             false,
		"error":               err.Error(),
		"spawned":             false,
		"state":               TenantWorldLeaseStates[0], // `proposed` — the only reachable state today
		"creation_available":  false,
		"creation_blocked_by": TenantWorldLeaseBlockers(),
		"client_rate_ignored": req.MonthlyRate > 0,
		"price_rule":          "the rate is the SERVER's: a create request may name only a system type and a duration",
	})
}

// handleLeaseList serves the tenant's leases. It is empty today by construction — creation refuses —
// and it STATES that, so an empty list is never read as "you have nothing leased".
func (l *Lobby) handleLeaseList(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	leases := leaseEngine.GetLeases(wallet)
	writeJSON(w, map[string]interface{}{
		"success":             true,
		"leases":              leases,
		"count":               len(leases),
		"creation_available":  false,
		"creation_blocked_by": TenantWorldLeaseBlockers(),
	})
}

// handleLeaseAvailable serves the CATALOGUE — the only part of this domain that exists — plus the
// stated reason a lease cannot be created.
//
// THE SHAPE IS THE CATALOGUE'S, and the clients read THAT: `systems[]` with `type` / `name` /
// `description` / `monthly_rate` (integer micro units). The World Dashboard's infrastructure panel
// read `res.leases[].price_micro` — a shape this route has never served — so it could only ever
// answer "No leases available". `monthly_rate` is the ONE name for a system's rate.
func (l *Lobby) handleLeaseAvailable(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"success":             true,
		"systems":             leaseEngine.GetAvailableSystems(),
		"creation_available":  false,
		"creation_blocked_by": TenantWorldLeaseBlockers(),
		"states":              TenantWorldLeaseStates,
		"price_rule":          "the rate is the SERVER's: a create request may name only a system type and a duration",
		"money_rule":          "rates are integer micro units of $VBV and are formatted for display only",
	})
}
