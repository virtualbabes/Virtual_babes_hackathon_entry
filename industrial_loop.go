//go:build !js && !wasm

package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// ── Industrial Loop Engine ───────────────────────────────────────────────────
// Perpetual value circulation: Player Activity → Businesses → Employment →
// Purchasing → Taxes → Treasuries → Development → Events → Player Activity
//
// The Industrial Loop is sacred. Value should circulate forever.
// Money should never simply disappear.

// LoopPhase represents the current phase of the industrial loop.
type LoopPhase string

const (
	PhaseActivity    LoopPhase = "activity"
	PhaseBusiness    LoopPhase = "business"
	PhaseEmployment  LoopPhase = "employment"
	PhasePurchasing  LoopPhase = "purchasing"
	PhaseTaxation    LoopPhase = "taxation"
	PhaseTreasury    LoopPhase = "treasury"
	PhaseDevelopment LoopPhase = "development"
	PhaseEvents      LoopPhase = "events"
)

// LoopMetrics tracks the flow of value through each phase.
type LoopMetrics struct {
	Phase          LoopPhase `json:"phase"`
	TotalFlowMicro uint64    `json:"total_flow_micro"`
	ParticipantCount int     `json:"participant_count"`
	LastUpdatedAt  time.Time `json:"last_updated_at"`
}

// IndustrialLoop manages the perpetual circulation engine.
type IndustrialLoop struct {
	mu       sync.RWMutex
	metrics  map[LoopPhase]*LoopMetrics
	// Flow tracking
	ActivityFlow   uint64 // player activity generated
	BusinessFlow   uint64 // business revenue
	EmploymentFlow uint64 // salaries paid
	PurchasingFlow uint64 // purchases made
	TaxationFlow   uint64 // taxes collected
	TreasuryFlow   uint64 // treasury deposits
	DevelopmentFlow uint64 // development spending
	EventFlow      uint64 // event rewards
	// Loop health
	LoopCirculations uint64    // total complete loops
	LastLoopAt       time.Time // last complete loop
}

var industrialLoop = &IndustrialLoop{
	metrics: make(map[LoopPhase]*LoopMetrics),
}

// RecordActivity records player activity value.
func (il *IndustrialLoop) RecordActivity(wallet string, amountMicro uint64) {
	il.mu.Lock()
	defer il.mu.Unlock()
	il.ActivityFlow += amountMicro
	il.updateMetrics(PhaseActivity, amountMicro)
}

// RecordBusiness records business revenue.
func (il *IndustrialLoop) RecordBusiness(businessID string, amountMicro uint64) {
	il.mu.Lock()
	defer il.mu.Unlock()
	il.BusinessFlow += amountMicro
	il.updateMetrics(PhaseBusiness, amountMicro)
}

// RecordEmployment records salary payment.
func (il *IndustrialLoop) RecordEmployment(employer, employee string, amountMicro uint64) {
	il.mu.Lock()
	defer il.mu.Unlock()
	il.EmploymentFlow += amountMicro
	il.updateMetrics(PhaseEmployment, amountMicro)
}

// RecordPurchasing records a purchase.
func (il *IndustrialLoop) RecordPurchasing(buyer, seller string, amountMicro uint64) {
	il.mu.Lock()
	defer il.mu.Unlock()
	il.PurchasingFlow += amountMicro
	il.updateMetrics(PhasePurchasing, amountMicro)
}

// RecordTaxation records tax collection.
func (il *IndustrialLoop) RecordTaxation(region string, amountMicro uint64) {
	il.mu.Lock()
	defer il.mu.Unlock()
	il.TaxationFlow += amountMicro
	il.updateMetrics(PhaseTaxation, amountMicro)
}

// RecordTreasury records treasury deposit.
func (il *IndustrialLoop) RecordTreasury(clubID string, amountMicro uint64) {
	il.mu.Lock()
	defer il.mu.Unlock()
	il.TreasuryFlow += amountMicro
	il.updateMetrics(PhaseTreasury, amountMicro)
}

// RecordDevelopment records development spending.
func (il *IndustrialLoop) RecordDevelopment(region string, amountMicro uint64) {
	il.mu.Lock()
	defer il.mu.Unlock()
	il.DevelopmentFlow += amountMicro
	il.updateMetrics(PhaseDevelopment, amountMicro)
}

// RecordEvent records event reward.
func (il *IndustrialLoop) RecordEvent(eventID string, amountMicro uint64) {
	il.mu.Lock()
	defer il.mu.Unlock()
	il.EventFlow += amountMicro
	il.updateMetrics(PhaseEvents, amountMicro)
}

// CompleteLoop records a complete circulation.
func (il *IndustrialLoop) CompleteLoop() {
	il.mu.Lock()
	defer il.mu.Unlock()
	il.LoopCirculations++
	il.LastLoopAt = time.Now()
}

// GetMetrics returns all loop metrics.
func (il *IndustrialLoop) GetMetrics() map[LoopPhase]*LoopMetrics {
	il.mu.RLock()
	defer il.mu.RUnlock()
	return il.metrics
}

// GetFlowSummary returns a summary of all flows.
func (il *IndustrialLoop) GetFlowSummary() map[string]uint64 {
	il.mu.RLock()
	defer il.mu.RUnlock()
	return map[string]uint64{
		"activity":    il.ActivityFlow,
		"business":    il.BusinessFlow,
		"employment":  il.EmploymentFlow,
		"purchasing":  il.PurchasingFlow,
		"taxation":    il.TaxationFlow,
		"treasury":    il.TreasuryFlow,
		"development": il.DevelopmentFlow,
		"events":      il.EventFlow,
	}
}

// GetLoopHealth returns the health of the industrial loop.
func (il *IndustrialLoop) GetLoopHealth() map[string]interface{} {
	il.mu.RLock()
	defer il.mu.RUnlock()
	
	// Calculate circulation efficiency
	totalFlow := il.ActivityFlow + il.BusinessFlow + il.EmploymentFlow + 
		il.PurchasingFlow + il.TaxationFlow + il.TreasuryFlow + 
		il.DevelopmentFlow + il.EventFlow
	
	// Loop efficiency = how much value is circulating vs sitting idle
	var efficiency float64
	if totalFlow > 0 {
		efficiency = float64(il.LoopCirculations) / float64(totalFlow) * 100
	}
	
	return map[string]interface{}{
		"total_flow":         totalFlow,
		"loop_circulations":  il.LoopCirculations,
		"efficiency":         efficiency,
		"last_loop_at":       il.LastLoopAt,
		"status":             il.getHealthStatus(efficiency),
	}
}

func (il *IndustrialLoop) getHealthStatus(efficiency float64) string {
	if efficiency > 80 {
		return "THRIVING"
	} else if efficiency > 50 {
		return "HEALTHY"
	} else if efficiency > 20 {
		return "STAGNANT"
	}
		return "DECLINING"
}

func (il *IndustrialLoop) updateMetrics(phase LoopPhase, amount uint64) {
	m, ok := il.metrics[phase]
	if !ok {
		m = &LoopMetrics{Phase: phase}
		il.metrics[phase] = m
	}
	m.TotalFlowMicro += amount
	m.ParticipantCount++
	m.LastUpdatedAt = time.Now()
}

// ── HTTP Handlers ────────────────────────────────────────────────────────────

func (l *Lobby) handleIndustrialLoopMetrics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"success": true,
		"metrics": industrialLoop.GetMetrics(),
		"flows":   industrialLoop.GetFlowSummary(),
		"health":  industrialLoop.GetLoopHealth(),
	})
}

func (l *Lobby) handleIndustrialLoopHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"success": true,
		"health":  industrialLoop.GetLoopHealth(),
	})
}

func (l *Lobby) handleIndustrialLoopRecord(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Phase  string `json:"phase"`
		Amount uint64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	
	switch LoopPhase(req.Phase) {
	case PhaseActivity:
		wallet := l.getWalletFromRequest(r)
		industrialLoop.RecordActivity(wallet, req.Amount)
	case PhaseBusiness:
		industrialLoop.RecordBusiness("manual", req.Amount)
	case PhaseEmployment:
		industrialLoop.RecordEmployment("manual", "manual", req.Amount)
	case PhasePurchasing:
		industrialLoop.RecordPurchasing("manual", "manual", req.Amount)
	case PhaseTaxation:
		industrialLoop.RecordTaxation("Base", req.Amount)
	case PhaseTreasury:
		industrialLoop.RecordTreasury("manual", req.Amount)
	case PhaseDevelopment:
		industrialLoop.RecordDevelopment("Base", req.Amount)
	case PhaseEvents:
		industrialLoop.RecordEvent("manual", req.Amount)
	default:
		writeJSON(w, map[string]interface{}{"success": false, "error": "unknown phase"})
		return
	}
	
	writeJSON(w, map[string]interface{}{"success": true})
}
