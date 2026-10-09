//go:build !js && !wasm

package main

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrAuditDiscrepancyDetected = errors.New("critical financial ledger failure: transactional input does not match cumulative target allocations")
)

// AuditLogEntry records a single reconciliation event.
type AuditLogEntry struct {
	Timestamp       int64  `json:"timestamp"`
	Context         string `json:"context"` // PILLAR 2: Forensic Visibility
	PayloadAmount   uint64 `json:"payload_amount"`
	FaucetRouted    uint64 `json:"faucet_routed"`
	ClubRouted      uint64 `json:"club_routed"`
	GovRouted       uint64 `json:"gov_routed"`
	SiphonedRouted  uint64 `json:"siphoned_routed"` // PILLAR 2: Infrastructure Visibility
	DiscrepancySkew int64  `json:"discrepancy_skew"`
}

// TokenSinkAuditReporter implements real-time financial reconciliation.
// PILLAR 2: Ledger Integrity.
type TokenSinkAuditReporter struct {
	Mu                     sync.RWMutex
	TotalSystemInputVetted uint64 // Atomic tracker for absolute inflow
	TotalSystemAllocated   uint64 // Cumulative liabilities assigned (f+c+g)
	TotalSystemSiphoned    uint64 // Cumulative infrastructure siphons
	TotalRewardsExited     uint64 // Cumulative physical dispatches
	TotalGhostReclaimed    uint64 // PILLAR 2: Console specific reconciliation
	TotalStagnationFees    uint64 // PILLAR 2: Activity enforcement reconciliation
	TotalPlatformFees      uint64 // PILLAR 2: Self-redemption surcharges
	AuditTrail             []AuditLogEntry
	MaxLogRetention        int
	Telemetry              *TelemetryLogger // PILLAR 4: Continuous Tracking
}

// NewTokenSinkAuditReporter initializes a new reconciliation engine.
func NewTokenSinkAuditReporter(retentionLimit int) *TokenSinkAuditReporter {
	return &TokenSinkAuditReporter{
		MaxLogRetention: retentionLimit,
		AuditTrail:      make([]AuditLogEntry, 0),
	}
}

// InterceptAndAudit verifies an active RouteCriminalTax execution payload.
// PILLAR 2: Invariant Validation.
func (tsa *TokenSinkAuditReporter) InterceptAndAudit(context string, payload uint64, faucet uint64, club uint64, gov uint64, siphoned uint64) error {
	// 1. Calculate sum of all distribution targets (Liabilities + Siphon)
	allocatedSum := faucet + club + gov + siphoned

	// 2. Compute skew discrepancy variance
	// Should resolve to 0 due to the Micro-Unit Remainder Rule in RouteCriminalTax
	skew := int64(payload) - int64(allocatedSum)

	var platDelta, ghostDelta, stagDelta uint64

	// PILLAR 2: Forensic Enforcement Reconciliation.
	// Ensure forensic counters match the enforcement logic defined in redemption_gateway.go.
	// These sub-counters track the total value recovered through specific tax actions.
	if context == "GHOST_TAX_ENFORCED" {
		ghostDelta = payload
		atomic.AddUint64(&tsa.TotalGhostReclaimed, payload)
	} else if context == "STAGNATION_TAX" {
		// Reconcile against the 25% total penalty (10% base + 15% stagnation fee).
		stagDelta = (payload * 25) / 100
		atomic.AddUint64(&tsa.TotalStagnationFees, stagDelta)
	} else if context == "PLATFORM_SURCHARGE" {
		platDelta = (payload * 20) / 100
		atomic.AddUint64(&tsa.TotalPlatformFees, platDelta)
	} else if context == "PLATFORM_SURCHARGE_STAGNANT" {
		// PILLAR 2: Hybrid Surcharge. (20% Platform + 15% Stagnant = 35% Total)
		platDelta = (payload * 20) / 100
		stagDelta = (payload * 15) / 100
		atomic.AddUint64(&tsa.TotalPlatformFees, platDelta)
		atomic.AddUint64(&tsa.TotalStagnationFees, stagDelta)
	}

	// 3. Increment internal atomic audit tally counters
	atomic.AddUint64(&tsa.TotalSystemInputVetted, payload)
	atomic.AddUint64(&tsa.TotalSystemAllocated, faucet+club+gov)
	atomic.AddUint64(&tsa.TotalSystemSiphoned, siphoned)

	// PILLAR 4: Telemetry Integration.
	// Forward real-time transactional metrics to the Prometheus exporter.
	if tsa.Telemetry != nil {
		tsa.Telemetry.UpdateRealTimeAccounting(payload, allocatedSum, skew)
		tsa.Telemetry.UpdateEnforcementAccounting(platDelta, ghostDelta, stagDelta)

		// PILLAR 1: Regional Tax Telemetry.
		// If a governor share was allocated, update the district-specific gauge using the context label.
		if gov > 0 {
			tsa.Telemetry.UpdateGovernorTax(context, gov)
		}
	}

	tsa.Mu.Lock()
	defer tsa.Mu.Unlock()

	// 4. Log the ledger footprint entry
	entry := AuditLogEntry{
		Timestamp:       time.Now().Unix(),
		Context:         context,
		PayloadAmount:   payload,
		FaucetRouted:    faucet,
		ClubRouted:      club,
		GovRouted:       gov,
		SiphonedRouted:  siphoned,
		DiscrepancySkew: skew,
	}

	// Maintain memory bound limits to optimize RAM heap footprints
	if len(tsa.AuditTrail) >= tsa.MaxLogRetention {
		tsa.AuditTrail = tsa.AuditTrail[1:] // Pop oldest entry
	}
	tsa.AuditTrail = append(tsa.AuditTrail, entry)

	// 5. Invariant Assertion Guardrail
	if skew != 0 {
		fmt.Printf(" [CRITICAL AUDIT ALERT] Precision leakage detected! Variance: %d micro-tokens.\n", skew)
		return ErrAuditDiscrepancyDetected
	}

	return nil
}

// LogInitialReserves vets the starting liquidity at server boot to ensure balanced reporting.
func (tsa *TokenSinkAuditReporter) LogInitialReserves(amount uint64) {
	atomic.AddUint64(&tsa.TotalSystemInputVetted, amount)
	atomic.AddUint64(&tsa.TotalSystemAllocated, amount)
}

// LogPhysicalOutflow records a confirmed on-chain reward dispatch.
func (tsa *TokenSinkAuditReporter) LogPhysicalOutflow(amount uint64) {
	atomic.AddUint64(&tsa.TotalRewardsExited, amount)
}

// GenerateFinancialHealthReport returns a snapshot evaluation of the engine balances.
func (tsa *TokenSinkAuditReporter) GenerateFinancialHealthReport() (string, bool) {
	tsa.Mu.RLock()
	defer tsa.Mu.RUnlock()

	inflow := atomic.LoadUint64(&tsa.TotalSystemInputVetted)
	allocated := atomic.LoadUint64(&tsa.TotalSystemAllocated)
	siphoned := atomic.LoadUint64(&tsa.TotalSystemSiphoned)
	exited := atomic.LoadUint64(&tsa.TotalRewardsExited)
	ghost := atomic.LoadUint64(&tsa.TotalGhostReclaimed)
	stag := atomic.LoadUint64(&tsa.TotalStagnationFees)
	platform := atomic.LoadUint64(&tsa.TotalPlatformFees)

	// PILLAR 2: Structural Integrity Check.
	// Transactional Drift: verify that all vetted inflow is accounted for as either liability or infra-siphon.
	drift := int64(inflow) - (int64(allocated) + int64(siphoned))
	isSystemHealthy := drift == 0
	status := "✓ OPERATIONAL"
	if !isSystemHealthy {
		status = "🚨 COMPROMISED - PRECISION LOSS"
	}

	lastCtx := "NONE"
	if len(tsa.AuditTrail) > 0 {
		lastCtx = tsa.AuditTrail[len(tsa.AuditTrail)-1].Context
	}

	// Solvency Check: verify that physical exits do not exceed created liabilities.
	isSolvent := exited <= allocated

	return fmt.Sprintf(
		" [ECONOMY AUDIT] Status: %s | Last: %s | In: %d | Allocated: %d | Siphoned: %d | Exits: %d | Drift: %d | Solvency: (Ghost: %d, Plat: %d, Stag: %d) | Solvent: %v",
		status, lastCtx, inflow, allocated, siphoned, exited, drift, ghost, platform, stag, isSolvent,
	), isSystemHealthy && isSolvent
}
