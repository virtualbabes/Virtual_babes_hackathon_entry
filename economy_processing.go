//go:build !js && !wasm

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidSplitProportions = errors.New("routing matrix validation error: sum of allocations must equal 100%")
	ErrZeroLiquidityInput      = errors.New("routing bypass: execution payload contains zero token velocity")
)

// RevenueSplitMatrix defines the percentage distribution for tax routing.
type RevenueSplitMatrix struct {
	FaucetShare     float64 // Percentage (0.0 to 1.0) directed to system pool
	ClubShare       float64 // Percentage directed to local club treasury
	GovernanceShare float64 // Percentage directed to spatial district leader
	EntityDividend  float64 // PILLAR 7-A: Percentage directed to entity dividend pools (Task 7002)
	CreatorRoyalty  float64 // PILLAR 7-C/Task 7202: Percentage directed to creator royalty pool on secondary sales
}

// EntityRevenueRouter routes a portion of trade revenue to entity dividend pools.
// Task 7002: Wire AMM revenue → economy_processing.go splits → investor dividends.
func (tsr *TokenSinkRouter) RouteEntityDividend(entityID string, amountMicro uint64, source string) {
	if tsr == nil || amountMicro == 0 || entityID == "" {
		return
	}

	tsr.Mu.Lock()
	defer tsr.Mu.Unlock()

	tsr.routeEntityDividendInternal(entityID, amountMicro, source)
}

// routeEntityDividendInternal is the internal unlocked variant.
// Caller MUST hold tsr.Mu if applicable to avoid deadlock.
func (tsr *TokenSinkRouter) routeEntityDividendInternal(entityID string, amountMicro uint64, source string) {
	node, exists := tsr.MarketNodes[entityID]
	if !exists || node == nil {
		return // Entity not in market — route to faucet instead
	}

	node.DividendPoolMicro += amountMicro

	// Audit trail for entity dividend routing
	if tsr.Audit != nil {
		_ = tsr.Audit.InterceptAndAudit("ENTITY_DIVIDEND_ROUTED", amountMicro, 0, 0, 0, 0)
	}

	log.Printf("[DIVIDEND_ROUTE] %.2f $VBV from %s routed to entity %s dividend pool", 
		float64(amountMicro)/1000000.0, source, entityID)
}

// ExternalLedgerClient defines the interface for executing on-chain transfers.
type ExternalLedgerClient interface {
	TransferTokens(ctx context.Context, toWallet string, amount uint64) error
}

// PayoutScheduler automates the distribution of regional taxes.
// PILLAR 2: Industrial Loop.
type PayoutScheduler struct {
	Mu             sync.Mutex
	Router         *TokenSinkRouter
	Ledger         ExternalLedgerClient
	PayoutInterval time.Duration
}

// NewPayoutScheduler initializes a new governor dividend distributor.
func NewPayoutScheduler(router *TokenSinkRouter, ledger ExternalLedgerClient, interval time.Duration) *PayoutScheduler {
	return &PayoutScheduler{
		Router:         router,
		Ledger:         ledger,
		PayoutInterval: interval,
	}
}

// StartPayoutEngine initializes a non-blocking background daemon ticker loop.
func (ps *PayoutScheduler) StartPayoutEngine(ctx context.Context) {
	ticker := time.NewTicker(ps.PayoutInterval)

	go func() {
		defer ticker.Stop()
		fmt.Printf(" [Governor Scheduler] Payout engine active. Interval: %v\n", ps.PayoutInterval)
		for {
			select {
			case <-ctx.Done():
				fmt.Println(" [Governor Scheduler] Shutting down payout engine daemon...")
				return
			case <-ticker.C:
				ps.ProcessAllRegionalDividends(ctx)
			}
		}
	}()
}

// ProcessAllRegionalDividends safely isolates and clears regional balances.
// PILLAR 2: Ledger Integrity.
func (ps *PayoutScheduler) ProcessAllRegionalDividends(ctx context.Context) {
	if ps.Router == nil || ps.Ledger == nil {
		return
	}

	// 1. Lock the global router state to safely snapshot and isolate the balances
	ps.Router.Mu.Lock()
	anyChange := false

	// Create a localized clone of the payout targets to minimize our lock footprint
	payoutTargets := make(map[string]uint64)
	for _, metric := range ps.Router.RegionalDistricts {
		if metric != nil && metric.DistrictDividendPool > 0 {
			if metric.GovernorAddress != "" {
				payoutTargets[metric.GovernorAddress] += metric.DistrictDividendPool
				anyChange = true
				// Reset the state pool inside the router IMMEDIATELY to prevent double-spending
				metric.DistrictDividendPool = 0
			} else {
				// PILLAR 2: Industrial Seal (Remainder Recovery).
				// If a governor address is missing (dissolved), return the pool to the Global Faucet.
				if ps.Router.GlobalFaucetPool != nil {
					*ps.Router.GlobalFaucetPool += metric.DistrictDividendPool
					anyChange = true
					fmt.Printf(" [Governor Scheduler] Recovered %d micro-tokens from dissolved district to Faucet.\n", metric.DistrictDividendPool)
					metric.DistrictDividendPool = 0
				}
			}
		}
	}

	ps.Router.Mu.Unlock()

	if len(payoutTargets) == 0 {
		return
	}

	// 2. Process transactions using a dedicated context timeout
	for wallet, amount := range payoutTargets {
		txCtx, txCancel := context.WithTimeout(ctx, 30*time.Second)
		err := ps.Ledger.TransferTokens(txCtx, wallet, amount)
		txCancel()

		if err != nil {
			// PILLAR 2: Audit Trail. Log failed payout attempts.
			fmt.Printf(" [Payout Error] Failed to distribute %d micro-tokens to Governor %s: %v\n", amount, wallet, err)

			fmt.Printf(" [Payout Error] Failed to distribute %d micro-tokens to Governor %s: %v\n", amount, wallet, err)

			// 🛡️ Fail-Safe Rollback Circuit: return funds to the pool on failure
			ps.Router.Mu.Lock()
			for _, metric := range ps.Router.RegionalDistricts {
				if metric != nil && metric.GovernorAddress == wallet {
					metric.DistrictDividendPool += amount
					break
				}
			}
			ps.Router.Mu.Unlock()
			continue
		}
		fmt.Printf(" [Payout Success] Disbursed %d micro-tokens dividend to active Governor: %s\n", amount, wallet)

		// PILLAR 2: Audit Trail. Log successful payouts.
		fmt.Printf(" [Payout Success] Disbursed %d micro-tokens dividend to active Governor: %s\n", amount, wallet)
		anyChange = true
	}

	if anyChange {
		// PILLAR 2: UI Parity Sync.
		// Trigger a global update to ensure the physical balance and
		// virtual organizational treasuries are synchronized in the UI.
		if lobby, ok := ps.Ledger.(*Lobby); ok {
			lobby.mutex.Lock()

			// Synchronize the Lobby's float representation of the vault balance
			// with the authoritative micro-unit integer reservoir.
			lobby.faucetBalance = float64(lobby.faucetBalanceMicro) / 1000000.0

			// Synchronize organizational treasuries with the router state.
			// This ensures the float 'Treasury' field reflects both accumulated revenue
			// and any remaining district-level dividends.
			for _, club := range lobby.clubs {
				numericID, _ := strconv.ParseUint(strings.TrimPrefix(club.ID, "CLUB-"), 10, 64)

				var totalMicro uint64
				ps.Router.Mu.RLock()
				if node, exists := ps.Router.ActiveClubs[numericID]; exists {
					totalMicro += node.TreasuryBalance
				}
				// Factor in remaining dividends from all territories managed by this organization
				for _, tID := range club.Territories {
					if metric, exists := ps.Router.RegionalDistricts[tID]; exists {
						totalMicro += metric.DistrictDividendPool
					}
				}
				ps.Router.Mu.RUnlock()

				// PILLAR 2: Integer Supremacy Sync.
				club.TreasuryMicro = totalMicro
				club.Treasury = float64(totalMicro) / 1000000.0
			}

			msg := lobby.getLobbyUpdateMsgLocked()
			lobby.mutex.Unlock()
			lobby.broadcast <- msg
		}
		// PILLAR 2: Administrative Maintenance Fee. Deduct daily fee from all clubs.
		if lobby, ok := ps.Ledger.(*Lobby); ok {
			lobby.ProcessDailyAdministrativeMaintenanceFee()
		}
	}
}

// NewTokenSinkRouter initializes a new router for economic flow control.
func NewTokenSinkRouter(globalFaucet *uint64, adminPool *uint64) *TokenSinkRouter {
	return &TokenSinkRouter{
		GlobalFaucetPool:  globalFaucet,
		AdminMaintenancePool: adminPool,
		ActiveClubs:       make(map[uint64]*ClubTreasuryNode),
		MarketNodes:       make(map[string]*EntityMarketNode), // PILLAR 2: AMM Persistence
		RegionalDistricts: make(map[string]*RegionalGovernanceMetric),
		Audit:             NewTokenSinkAuditReporter(1000), // Monitor last 1000 transactions
	}
}

// RouteCriminalTax executes atomic redistribution of system fees using remainder rerouting.
// PILLAR 2: Ledger Integrity.
func (tsr *TokenSinkRouter) RouteCriminalTax(context string, totalTaxPayload uint64, matrix RevenueSplitMatrix, targetClubID uint64, targetDistrict string) error {
	if totalTaxPayload == 0 {
		return ErrZeroLiquidityInput
	}

	// 1. Structural Validation: Sum of allocation nodes must resolve to exactly 1.0 (100%)
	totalAllocation := matrix.FaucetShare + matrix.ClubShare + matrix.GovernanceShare
	if math.Abs(totalAllocation-1.0) > 1e-9 {
		return ErrInvalidSplitProportions
	}

	tsr.Mu.Lock()
	defer tsr.Mu.Unlock()

	// PILLAR 2: Administrative USDC Siphon (Section 11)
	// If Faucet coverage > 150%, extract 10% for Admin Maintenance.
	inflow := atomic.LoadUint64(&tsr.Audit.TotalSystemInputVetted)
	allocated := atomic.LoadUint64(&tsr.Audit.TotalSystemAllocated)
	if allocated > 0 && (float64(inflow)/float64(allocated)) > 1.5 {
		siphonAmt := totalTaxPayload / 10
		totalTaxPayload -= siphonAmt
		
		if tsr.AdminMaintenancePool != nil {
			*tsr.AdminMaintenancePool += siphonAmt
		}
		// PILLAR 2: Track siphoned amount for InterceptAndAudit
		siphonAmount = siphonAmt 
	}

	// PILLAR 2: Phase 4 Console Synergy (Inactive Hook).
	// Evaluate the 'Excessive Accumulation Siphon' for console transactions.
	// If faucet reserves are optimal (>8,000 VBV), a 10% cut is redirected to infrastructure 
	// maintenance before the industrial split occurs.
	// Supports all console-related enforcement contexts for unified auditing.
	actualTaxPayload := totalTaxPayload
	var siphonAmount uint64 = 0
	if (context == "CONSOLE_DLC" || context == "GHOST_TAX_ENFORCED" || context == "STAGNATION_TAX") && tsr.GlobalFaucetPool != nil {
		// Optimal Reserve Threshold: 80% of 10,000 VBV capacity.
		// PILLAR 2: Thread-safe reserve check.
		const optimalThreshold = 8000 * 1000000 
		if *tsr.GlobalFaucetPool > optimalThreshold {
			siphonAmount = totalTaxPayload / 10 // 10% Maintenance Siphon
			if siphonAmount > 0 {
				actualTaxPayload -= siphonAmount

				// PILLAR 2: Notification Cap.
				// Ensure the reported siphon amount does not exceed the admin notification cap.
				displaySiphonAmount := siphonAmount
				if displaySiphonAmount > MaxAdminNotificationAmountMicro {
					displaySiphonAmount = MaxAdminNotificationAmountMicro
				}
				// PILLAR 5: Reactive Atmosphere.
				// Trigger the siphon notification hook to alert administrators of incoming infrastructure funding.
				if tsr.SiphonNotifier != nil {
					tsr.SiphonNotifier(fmt.Sprintf("🛡️ <b>INFRASTRUCTURE SIPHON:</b> %.2f $VBV redirected from Console DLC to maintenance ledger.", float64(displaySiphonAmount)/1000000.0))
				}
				fmt.Printf(" [ECONOMY] CONSOLE_DLC Siphon active: Redirected %d micro-tokens to infrastructure.\n", siphonAmount)
			}
		}
	}

	var fRouted, cRouted, gRouted uint64

	// 2. Calculate & Route Global Faucet Allocation
	if matrix.FaucetShare > 0 {
		fRouted = uint64(math.Floor(float64(actualTaxPayload) * matrix.FaucetShare))
		if tsr.GlobalFaucetPool != nil {
			*tsr.GlobalFaucetPool += fRouted
		}
	}

	// 3. Calculate & Route Club Treasury Allocation.
	// PILLAR 2: Supports both targeted and global redistribution.
	if matrix.ClubShare > 0 {
		cTotal := uint64(math.Floor(float64(actualTaxPayload) * matrix.ClubShare))
		if targetClubID != 0 {
			if clubNode, exists := tsr.ActiveClubs[targetClubID]; exists {
				clubNode.TreasuryBalance += cTotal
				cRouted = cTotal
			}
		} else if len(tsr.ActiveClubs) > 0 {
			// PILLAR 2: Global Club Distribution (Courthouse penalty redistribution).
			share := cTotal / uint64(len(tsr.ActiveClubs))
			for _, node := range tsr.ActiveClubs {
				node.TreasuryBalance += share
			}
			cRouted = share * uint64(len(tsr.ActiveClubs))
		}
	}

	// 4. Calculate & Route Regional Governor Metrics Dividend.
	// ============================================================================
	// PILLAR 7-A/Task 7002: Entity Dividend Seeding from ALL Economic Activity (repeated for each call site).
	// This block is intentionally duplicated at every economic event handler to ensure 
	// entity dividend pools receive yield from contracts, fines, trades, and auctions.

	// PILLAR 1: Governor Incentives. Supports targeted district or sector-wide payout.
	if matrix.GovernanceShare > 0 {
		gTotal := uint64(math.Floor(float64(actualTaxPayload) * matrix.GovernanceShare))
		if targetDistrict != "" && targetDistrict != "GLOBAL" {
			if govNode, exists := tsr.RegionalDistricts[targetDistrict]; exists && govNode.GovernorAddress != "" {
				govNode.DistrictDividendPool += gTotal
				gRouted = gTotal
			}
		} else {
			// PILLAR 2: Sector-Wide Governor Distribution.
			var activeGovs []*RegionalGovernanceMetric
			for _, m := range tsr.RegionalDistricts {
				if m != nil && m.GovernorAddress != "" {
					activeGovs = append(activeGovs, m)
				}
			}
			if len(activeGovs) > 0 {
				share := gTotal / uint64(len(activeGovs))
				for _, m := range activeGovs {
					m.DistrictDividendPool += share
				}
				gRouted = share * uint64(len(activeGovs))
			}
		}
	}

	// 🛡️ Micro-Unit Precision Remainder Accounting:
	// Reroute rounding fractions straight into the Faucet Pool to lock down the industrial seal.
	distributedTotal := fRouted + cRouted + gRouted
	if distributedTotal < actualTaxPayload {
		remainder := actualTaxPayload - distributedTotal
		fRouted += remainder // Track remainder as routed to faucet for the audit call
		if tsr.GlobalFaucetPool != nil {
			*tsr.GlobalFaucetPool += remainder
		}
	}

	// PILLAR 2: Real-time Reconciliation.
	// Connect the audit reporter to monitor balance drift for this routing event.
	if tsr.Audit != nil {
		// PILLAR 2: Structural Audit. Pass siphoned tokens separately to enable granular health reporting.
		_ = tsr.Audit.InterceptAndAudit(context, totalTaxPayload, fRouted, cRouted, gRouted, siphonAmount)
	}

	// ============================================================================
	// PILLAR 7-A/Task 7002: Entity Dividend Seeding from ALL Economic Activity.
	// Route a portion of every economic event (contracts, fines, trades, auctions) 
	// into entity dividend pools so investors receive yield from real activity.
	if matrix.EntityDividend > 0 && totalTaxPayload > 0 {
		dividendAmount := uint64(float64(totalTaxPayload) * matrix.EntityDividend)
		
		// Determine target entity wallet for the revenue source context
		var targetEntityWallet string
		switch context {
		case "CONTRACT_FEE", "UNDERWORLD_CONTRACT":
			targetEntityWallet = targetDistrict // district maps to entity ID
		default:
			// For other contexts, use club owner or governor as fallback entity
			if tsr.RegionalDistricts != nil && targetDistrict != "" {
				if metric := tsr.RegionalDistricts[targetDistrict]; metric != nil && metric.GovernorAddress != "" {
					targetEntityWallet = metric.GovernorAddress
				}
			}
		}

		if dividendAmount > 0 && targetEntityWallet != "" && tsr.MarketNodes != nil {
			tsr.routeEntityDividendInternal(targetEntityWallet, dividendAmount, context)
		} else if dividendAmount > 0 {
			log.Printf("[DIVIDEND_ROUTE] WARNING: Cannot route %.2f $VBV entity dividend — missing target entity or market node", float64(dividendAmount)/1000000.0)
		}
	}

	// ============================================================================
	// PILLAR 7-C/Task 7202: Creator Royalty Routing on Secondary Sales.
	// Routes creator royalty percentage from secondary-sale events to the CreatorStore pool.
	// Only applies when context indicates a marketplace/product sale event.
	// The actual per-creator wallet routing is handled by ProcessSecondarySale in creator_store_service.go.
	// This block tracks aggregate economy-wide royalty volume for audit reporting.
	// ============================================================================
	if matrix.CreatorRoyalty > 0 && totalTaxPayload > 0 {
		royaltyAmount := uint64(float64(totalTaxPayload) * matrix.CreatorRoyalty)

		if tsr.Audit != nil {
			_ = tsr.Audit.InterceptAndAudit("CREATOR_ROYALTY_ROUTED", royaltyAmount, 0, 0, 0, 0)
		}

		log.Printf("[ROYALTY_ROUTE] %.2f $VBV creator royalty routed from %s event", float64(royaltyAmount)/1000000.0, context)
	}

	// ============================================================================
	// UNDERWORLD CAREER XP TRIGGERS — TaxAuditor (P2-A tax collection audit)
	// PILLAR 13: Career progression tied to economic enforcement events.
	// +30 XP per audited tax transaction; bonus +50 for flagged violations (siphons).
	// ============================================================================
	if tsr.Ledger != nil {
		if lobby, ok := tsr.Ledger.(*Lobby); ok && lobby != nil {
			lobby.mutex.RLock()

			// XP trigger: TaxAuditor gains +30 per tax transaction audited
			for wallet := range lobby.wallets {
				stats := lobby.leaderboard[wallet]
				if stats.CareerXP != nil && (stats.JobRole == "TaxAuditor" || CareerHasRole(stats.CareerXP, "TaxAuditor")) {
					lobby.mutex.RUnlock()

					// PILLAR 2: Integer Supremacy — deterministic XP award.
					stats.ensureCareerXPInitialized()
					baseXPAudited := uint64(30) // Base +30 XP per audited tax transaction

					if stats.CareerXP != nil {
						// $VBV-gate multiplier: scale XP by player's sustained liquidity tier (PILLAR 13).
						vbvMultiplier := stats.CareerXP.GetVBVGatingMultiplier()
						scaledTaxXP := uint64(float64(baseXPAudited) * vbvMultiplier)

						stats.CareerXP.TrackCareerXP("TaxAuditor", scaledTaxXP)
						lobby.logAdminAuditLocked("CAREER_TAX_AUDITOR_XP", wallet, fmt.Sprintf("+%d XP (base: %d, $VBV-gate: ×%.0f)", scaledTaxXP, baseXPAudited, vbvMultiplier))

						if vbvMultiplier > 1.0 {
							lobby.logAdminAuditLocked("CAREER_TAX_AUDITOR_VBV_GATE", wallet, fmt.Sprintf("$VBV-gate active on audit XP: ×%.0f (AvgSustainedMicro: %d μVBV)", vbvMultiplier, stats.CareerXP.AvgSustainedMicro))
						}

						// Bonus: +50 for flagged violations (siphon events detected in routing) — also $VBV-gated.
						if siphonAmount > 0 {
							bonusXP := uint64(50) // Flagged violation bonus per transaction
							scaledBonusXP := uint64(float64(bonusXP) * vbvMultiplier)

							stats.CareerXP.TrackCareerXP("TaxAuditor", scaledBonusXP)
							lobby.logAdminAuditLocked("CAREER_TAX_AUDITOR_BONUS_XP", wallet, fmt.Sprintf("+%d XP bonus (flagged violation, base: %d, $VBV-gate: ×%.0f)", scaledBonusXP, bonusXP, vbvMultiplier))
						}
					}

					lobby.mutex.RLock()
				}
			}

			lobby.mutex.RUnlock()
		}
	}

	return nil
}

/**
 * IsTaxHavenActive verifies if a club's management has secured a localized tax exemption.
 * PILLAR 1: Political Influence.
 */
func (tsr *TokenSinkRouter) IsTaxHavenActive(club *Club) bool {
	if club == nil { return false }
	return !club.TaxHavenExpiresAt.IsZero() && time.Now().Before(club.TaxHavenExpiresAt)
}

/**
 * CheckCorporateBailouts evaluates organizational health and provides Faucet-backed 
 * stimulus to struggling clubs.
 * PILLAR 1: Industrial Loop (Safety Net).
 */
func (l *Lobby) CheckCorporateBailouts() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	const bailoutAmtMicro = 5000 * 1000000
	const totalSectors = 9 // authoritative territory count from TERRITORY_MAP

	// Faucet liquidity check: ensure the House can afford the stimulus
	if l.faucetBalanceMicro < bailoutAmtMicro {
		return
	}

	anyBailed := false
	for _, club := range l.clubs {
		// Coverage calculation based on total Arena sectors (9)
		coverage := float64(len(club.Territories)) / float64(totalSectors)
		
		// PILLAR 1: Bailout trigger threshold (< 20% coverage)
		// Only applies to established clubs with at least one territory.
		if coverage < 0.20 && len(club.Territories) > 0 {
			numericID, _ := strconv.ParseUint(strings.TrimPrefix(club.ID, "CLUB-"), 10, 64)
			
			if l.tokenSinkRouter != nil {
				l.tokenSinkRouter.Mu.Lock()
				if node, ok := l.tokenSinkRouter.ActiveClubs[numericID]; ok {
					// Liability Shift: Increase organization debt (treasury) from unreserved faucet pool.
					node.TreasuryBalance += bailoutAmtMicro
					club.TreasuryMicro = node.TreasuryBalance
					club.Treasury = node.TreasuryBalance // Syncing alias for UI
					club.LastBailoutAt = time.Now()
					
					// PILLAR 2: Integer Supremacy & Ledger Integrity.
					// Physically decrement Faucet reserves to fund the organizational bailout.
					l.faucetBalanceMicro -= bailoutAmtMicro
					l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0

					// PILLAR 2: Audit Trail Integration.
					if l.tokenSinkRouter != nil && l.tokenSinkRouter.Audit != nil {
						_ = l.tokenSinkRouter.Audit.InterceptAndAudit("CORPORATE_BAILOUT", bailoutAmtMicro, 0, bailoutAmtMicro, 0, 0)
					}

					l.logAdminAuditLocked("CORPORATE_BAILOUT", club.ID, "Injected 5,000 $VBV from Faucet pool")
					
					if cid := l.getClientIDFromWalletLocked(club.OwnerWallet); cid != "" {
						l.sendToClientLocked(cid, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"🏛️ <b>CORPORATE BAILOUT:</b> Your organization received 5,000 $VBV stimulus due to critical sector coverage."}`)})
					}
					anyBailed = true
				}
				l.tokenSinkRouter.Mu.Unlock()
			}
		}
	}

	if anyBailed {
		l.applyDynamicScalingLocked()
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	}
}
