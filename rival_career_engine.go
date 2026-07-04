//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"
)

// ============================================================================
// PILLAR 13: CAREER PROGRESSION SYSTEM (Section 13)
// XP/Level gating for all 20 career roles across Underworld & Justice layers.
// ============================================================================

const (
	CareerTierPeon     = 0
	CareerTierApprentice = 5
	CareerTierJourneyman = 15
	CareerTierExpert     = 30
	CareerTierMaster     = 50
	CareerTierBoss       = 75
)

// CareerXP records a player's progression through career tiers.
// Pillar 13 ($VBV-Sustained Progression): LiquiditySamples + AvgSustainedMicro gate all tier advancement.
type CareerXP struct {
	RoleXP          map[string]uint64 `json:"role_xp"`          // Role name -> XP earned
	LessonLevel     int               `json:"level"`            // Overall career level (0-100)
	PromotedRoles   []string          `json:"promoted_roles"`   // Roles this player qualifies for
	CurrentPrompts  []string          `json:"current_prompts"`  // Active promotion offers (UI)

	// Pillar 13: $VBV-sustained balance tracking
	LiquiditySamples  []uint64        `json:"-"`                // Recent player balance snapshots (micro-$VBV), last 14
	AvgSustainedMicro uint64          `json:"avg_sustained_micro"` // Computed average from samples (micro-$VBV)
	DemotionWarningAt time.Time       `json:"demotion_warning_at"` // When demotion warning was issued (0 = none)
}

// CalculateLessonLevel returns the player's career level based on total XP.
func (cxp *CareerXP) CalculateLessonLevel() int {
	totalXP := uint64(0)
	for _, xp := range cxp.RoleXP {
		totalXP += xp
	}
	// Exponential curve: each level requires 1500 more XP than previous.
	level := 0
	for totalXP >= uint64(level*level*30+level*1500) && level < 100 {
		level++
	}
	cxp.LessonLevel = level
	return level
}

// UnlockAchievementForRole grants career-related achievements.
func (cxp *CareerXP) UnlockAchievementForRole(role string) bool {
	for _, r := range cxp.PromotedRoles {
		if r == role {
			return true
		}
	}
	return false
}

// TrackCareerXP adds XP to a specific role's progression.
func (cxp *CareerXP) TrackCareerXP(role string, xp uint64) {
	if cxp == nil {
		return
	}
	if cxp.RoleXP == nil {
		cxp.RoleXP = make(map[string]uint64)
	}
	cxp.RoleXP[role] += xp
}

// CareerHasRole checks if a player has promoted to a specific role.
func CareerHasRole(cxp *CareerXP, role string) bool {
	if cxp == nil {
		return false
	}
	for _, r := range cxp.PromotedRoles {
		if r == role {
			return true
		}
	}
	return false
}

// RivalryState tracks dynamic rivalry mechanics between players.
type RivalryState struct {
	SoloHunterScore      int                `json:"solo_hunter_score"`        // Bounty hunter solo capture score
	AOSRivalryActive     bool               `json:"aos_rivalry_active"`       // Whether AOS rival is active
	AOSTeamID            string             `json:"aos_team_id"`              // Associated AOS team
	SilkRoadHoardCount   int                `json:"silk_road_hoard_count"`    // Cards hoarded by Hostage Hosts
	HoardPressure        int                `json:"hoard_pressure"`           // Ransom pressure multiplier
	InfoBrokerDeals      []InfoBrokerDeal   `json:"info_broker_deals"`        // Active info broker transactions
	ActiveRivals         []string           `json:"active_rivals,omitempty"`  // Wallet addresses of active rivals
	PendingInvitations   []PendingRivalInvite `json:"pending_invitations,omitempty"` // Pending rival invitations
	BountyLicenseActive  bool               `json:"bounty_license_active"`    // Active bounty hunter license status
	ArcNetActive         bool               `json:"arc_net_active,omitempty"`  // Arc-Net spy vision active
}

// PendingRivalInvite represents an incoming rival invitation.
type PendingRivalInvite struct {
	FromWallet  string    `json:"from_wallet"`
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Level       int       `json:"level"`
}

type InfoBrokerDeal struct {
	TargetWallet string    `json:"target_wallet"`
	SellerWallet string    `json:"seller_wallet"`
	PriceMicro   uint64    `json:"price_micro"`
	ExpiresAt    time.Time `json:"expires_at"`
	DataHash     string    `json:"data_hash"`
}

// StartRivalryEngine initializes the rivalry tracking system.
func (l *Lobby) StartRivalryEngine() {
	log.Println("[RIVALRY] Rivalry Engine initialized")
	
	// Process rivalry tick every 5 minutes for dynamic pressure updates
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			l.processRivalryTick()
		}
	}()
}

// processRivalryTick handles periodic rivalry system maintenance.
func (l *Lobby) processRivalryTick() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	now := time.Now()
	// Clean expired info broker deals
	for wallet, stats := range l.leaderboard {
		if stats.Rivalry == nil || stats.Rivalry.InfoBrokerDeals == nil {
			continue
		}
		activeDeals := make([]InfoBrokerDeal, 0)
		for _, deal := range stats.Rivalry.InfoBrokerDeals {
			if now.Before(deal.ExpiresAt) {
				activeDeals = append(activeDeals, deal)
			} else {
				// Expired: refund half the broker fee to the seller
				l.playerBalances[deal.SellerWallet] += (deal.PriceMicro * 50) / 100
				log.Printf("[RIVALRY] Info broker deal expired for %s, refunded %.2f $VBV\n",
					deal.SellerWallet, float64(deal.PriceMicro*50/100)/1000000.0)
			}
		}
		stats.Rivalry.InfoBrokerDeals = activeDeals
		l.leaderboard[wallet] = stats
	}
	
	log.Printf("[RIVALRY] Rivalry tick processed for %d players\n", len(l.leaderboard))
}

// TrackSoloCapture records a solo hunter's capture for rivalry scoring.
func (l *Lobby) TrackSoloCapture(hunterWallet, targetWallet string, wantedLevel int) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	hStats := l.leaderboard[hunterWallet]
	if hStats.Rivalry == nil {
		hStats.Rivalry = &RivalryState{}
	}
	hStats.Rivalry.SoloHunterScore += wantedLevel
	
	// Increase Silk Road hoard pressure if the target has associated hoards
	lStats := l.leaderboard[targetWallet]
	if lStats.Rivalry != nil && lStats.Rivalry.SilkRoadHoardCount > 0 {
		lStats.Rivalry.HoardPressure += 5 // +5% pressure per capture
		log.Printf("[RIVALRY] Hoard pressure increased by +5 for target %s\n", targetWallet)
		
		// Broadcast ransom pressure increase to hoarding team members
		if lStats.Rivalry.AOSTeamID != "" {
			for wallet, st := range l.leaderboard {
				if st.EmployerClubID == lStats.Rivalry.AOSTeamID {
					l.sendToClientLocked(l.getClientIDFromWalletLocked(wallet), Envelope{
						Type: "admin_notification",
						Payload: json.RawMessage(fmt.Sprintf(`{"text":"⚠️ <b>HOSTAGE PRESSURE:</b> Target captured! Ransom demand increased by +5%%."}`)),
					})
				}
			}
		}
	}
	
	l.leaderboard[hunterWallet] = hStats
	l.leaderboard[targetWallet] = lStats
	
	log.Printf("[RIVALRY] Solo hunter %s score: %d\n", hunterWallet, hStats.Rivalry.SoloHunterScore)
}

// TrackTeamCapture records an AOS team's capture for rivalry scoring.
func (l *Lobby) TrackTeamCapture(teamWallet string, targetWallet string, wantedLevel int, members int) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	teamStats := l.leaderboard[teamWallet]
	if teamStats.Rivalry == nil {
		teamStats.Rivalry = &RivalryState{}
	}
	
	// AOS capture yields more $VBV but less reputation vs solo
	rewardPerMember := uint64(wantedLevel * 30) * 1000000 // 30 $VBV per member scaling
	l.playerBalances[teamWallet] += rewardPerMember
	
	log.Printf("[RIVALRY] AOS team %s captured target, distributed %.2f $VBV total\n",
		teamWallet, float64(rewardPerMember)/1000000.0)
}

// SellIntelToHighestBidder processes an info broker deal. Returns the buyer wallet who won.
func (l *Lobby) SellIntelToHighestBidder(sellerWallet string, targetWallet string, basePrice int) (string, uint64) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	sStats := l.leaderboard[sellerWallet]
	if sStats.Rivalry == nil {
		sStats.Rivalry = &RivalryState{}
	}
	
	// Simulate bids from interested players (AOS teams, rival bosses)
	bidderWallets := make([]string, 0)
	bestBid := uint64(0)
	bestBidder := ""
	
	for wallet, stats := range l.leaderboard {
		if wallet == sellerWallet || wallet == targetWallet {
			continue
		}
		
		// AOS teams pay 1.5x base price for intel
		bidMultiplier := 1.0
		if stats.Rivalry != nil && stats.Rivalry.AOSRivalryActive {
			bidMultiplier = 1.5
			stats.Rivalry.AOSTeamID = wallet // Track team affiliation
		}
		
		// Underworld bosses pay 2x base price to gain intel on rivals
		if l.playerService.GetHegemonyPath(stats.JobRole) == "UNDERWORLD" {
			bidMultiplier = 2.0
		}
		
		bid := uint64(float64(basePrice) * bidMultiplier)
		
		// Add randomness to bidding (±20%)
		bidVariation := rand.Intn(40) - 20 // -20 to +20
		bid = uint64(float64(bid) * (1.0 + float64(bidVariation)/100.0))
		
		if bid > bestBid {
			bestBid = bid
			bestBidder = wallet
		}
	}
	
	if bestBidder != "" {
		// Execute deal: buyer pays broker fee, seller gets 70%, faucet gets 30%
		l.playerBalances[bestBidder] -= bestBid
		sellerCut := (bestBid * 70) / 100
		l.playerBalances[sellerWallet] += sellerCut
		
		// Faucet takes 30% broker fee
		l.faucetBalanceMicro += (bestBid - sellerCut)
		
		// Record deal on seller's rivalry record
		sStats.Rivalry.InfoBrokerDeals = append(sStats.Rivalry.InfoBrokerDeals, InfoBrokerDeal{
			TargetWallet: targetWallet,
			SellerWallet: sellerWallet,
			PriceMicro:   bestBid,
			ExpiresAt:    time.Now().Add(30 * time.Minute), // 30 min TTL for the data
			DataHash:     fmt.Sprintf("INTEL-%d", time.Now().Unix()),
		})
		
		l.leaderboard[sellerWallet] = sStats
		
		log.Printf("[RIVALRY] Intel sold to %s for %.2f $VBV (seller got %.2f)\n",
			bestBidder, float64(bestBid)/1000000.0, float64(sellerCut)/1000000.0)
		
		return bestBidder, sellerCut
	}
	
	return "", 0
}

// ============================================================================
// PILLAR 12: JUSTICE & UNDERWORLD SHOP ITEMS (Sections 12A & 12B)
// New items for the shop registry.
// ============================================================================

// JusticeItem defines a Justice faction item.
type JusticeItem struct {
	ItemDef
	JusticeBonus map[string]int // Target stat -> bonus value
	DurationMin  int            // Effect duration in minutes (0 = permanent)
}

// UnderworldItem defines an Underworld faction item.
type UnderworldItem struct {
	ItemDef
	UnderworldBonus map[string]int // Target stat -> bonus value
	DurationMin     int            // Effect duration in minutes
	StolenTag       bool           // Whether the item carries a "stolen" tag
}

// RegisterJusticeItems adds Justice faction items to the shop.
func (l *Lobby) RegisterJusticeItems() {
	if l.shopRegistry == nil {
		l.shopRegistry = &ShopRegistry{}
	}
	
	justiceItems := map[string]interface{}{
		"truth_serum": JusticeItem{
			ItemDef: ItemDef{
				ID:         "ITEM-JUSTICE-001",
				Name:       "Truth Serum",
				Description: "Temporarily reveals all active item buffs and debuffs on an opponent's cards.",
				CostMicro:  2500 * 1000000,
				Category:   JusticeCategory,
				MaxStack:   3,
			},
			JusticeBonus: map[string]int{"reveals_opponent_buffs": 1},
			DurationMin:  5, // 5 minutes of vision
		},
		"reputation_shield": JusticeItem{
			ItemDef: ItemDef{
				ID:         "ITEM-JUSTICE-002",
				Name:       "Reputation Shield",
				Description: "Reduces Reputation penalties from failed pro-social actions by 75% for 1 hour.",
				CostMicro:  3000 * 1000000,
				Category:   JusticeCategory,
				MaxStack:   1,
			},
			JusticeBonus: map[string]int{"rep_penalty_reduction": 75},
			DurationMin:  60,
		},
		"bounty_license": JusticeItem{
			ItemDef: ItemDef{
				ID:         "ITEM-JUSTICE-003",
				Name:       "Bounty Hunter License",
				Description: "Recurring license (50 $VBV/week) to maintain Clean Hunter status and access the Justice Tier Dashboard.",
				CostMicro:  50 * 1000000,
				Category:   JusticeCategory,
				MaxStack:   1,
				Recurring:  true,
				RecurringDays: 7,
			},
			JusticeBonus: map[string]int{"access_justice_dashboard": 1},
			DurationMin:  10080, // 7 days in minutes
		},
		"arc_net_spy": JusticeItem{
			ItemDef: ItemDef{
				ID:         "ITEM-JUSTICE-004",
				Name:       "Arc-Net-Spy",
				Description: "Reveals the full inventory of a target player for 5 minutes.",
				CostMicro:  5000 * 1000000,
				Category:   JusticeCategory,
				MaxStack:   2,
			},
			JusticeBonus: map[string]int{"reveal_inventory": 1},
			DurationMin:  5,
		},
	}
	
	for name, item := range justiceItems {
		l.shopRegistry.RegisterItem(name, item)
		log.Printf("[SHOP] Registered Justice item: %s (%.2f $VBV)", name, float64(item.(JusticeItem).CostMicro)/1000000.0)
	}
	
	log.Println("[SHOP] Justice faction items registered")
}

// RegisterUnderworldItems adds Underworld faction items to the shop.
func (l *Lobby) RegisterUnderworldItems() {
	if l.shopRegistry == nil {
		l.shopRegistry = &ShopRegistry{}
	}
	
	worldItems := map[string]interface{}{
		"data_scramble": UnderworldItem{
			ItemDef: ItemDef{
				ID:         "ITEM-UNDERWORLD-001",
				Name:       "Data Scramble",
				Description: "Temporarily hides a player's entire match history from public view for 30 minutes.",
				CostMicro:  4000 * 1000000,
				Category:   UnderworldCategory,
				MaxStack:   2,
			},
			UnderworldBonus: map[string]int{"hide_match_history": 1},
			DurationMin:     30,
		},
		"signal_dampener": UnderworldItem{
			ItemDef: ItemDef{
				ID:         "ITEM-UNDERWORLD-002",
				Name:       "Signal Dampener",
				Description: "Hides criminality signatures from bounty tracking. Stacks with team members.",
				CostMicro:  800 * 1000000,
				Category:   UnderworldCategory,
				MaxStack:   5,
			},
			UnderworldBonus: map[string]int{"dampen_signal": 20}, // 20% per stack
			DurationMin:     60,
		},
		"security_override": UnderworldItem{
			ItemDef: ItemDef{
				ID:         "ITEM-UNDERWORLD-003",
				Name:       "Security Override",
				Description: "Bribes underworld security to force return of a fenced card (5,000 $VBV).",
				CostMicro:  5000 * 1000000,
				Category:   UnderworldCategory,
				MaxStack:   1,
			},
			UnderworldBonus: map[string]int{"force_card_return": 1},
			DurationMin:     0, // Instant use
		},
		"regulatory_bypass": UnderworldItem{
			ItemDef: ItemDef{
				ID:         "ITEM-UNDERWORLD-004",
				Name:       "Regulatory Bypass Permit",
				Description: "Reduces corporate tax on salary by 50% for 24 hours.",
				CostMicro:  100 * 1000000,
				Category:   UnderworldCategory,
				MaxStack:   1,
			},
			UnderworldBonus: map[string]int{"reduce_corp_tax": 50},
			DurationMin:     1440, // 24 hours
		},
	}
	
	for name, item := range worldItems {
		l.shopRegistry.RegisterItem(name, item)
		log.Printf("[SHOP] Registered Underworld item: %s (%.2f $VBV)", name, float64(item.(UnderworldItem).CostMicro)/1000000.0)
	}
	
	log.Println("[SHOP] Underworld faction items registered")
}

// ApplyJusticeItemEffect applies the effect of a Justice faction item to a player's match state.
func (l *Lobby) ApplyJusticeItemEffect(playerWallet string, itemName string, match *MatchState) bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	stats := l.leaderboard[playerWallet]
	if stats.JobRole == "" || l.playerService.GetHegemonyPath(stats.JobRole) != "JUSTICE" {
		return false
	}
	
	// Apply item-specific buffs to the match state
	switch itemName {
	case "truth_serum":
		// Reveal opponent's buffs in UI (set a flag on match)
		match.ActiveItemBuffs[playerStatsToID(stats)]["truth_serum_active"] = true
		log.Printf("[ITEM] Truth Serum activated for %s\n", playerWallet)
		
	case "reputation_shield":
		// Apply rep shield buff to match state
		if match.ActiveItemBuffs == nil {
			match.ActiveItemBuffs = make(map[string]map[string]bool)
		}
		pID := match.P1Wallet
		if playerWallet == match.P2Wallet {
			pID = match.P2Wallet
		}
		match.ActiveItemBuffs[pID]["rep_shield"] = true
		
	case "bounty_license":
		// Grant dashboard access (set flag in stats, not match)
		stats.BountyLicenseActive = true
		l.leaderboard[playerWallet] = stats
		log.Printf("[ITEM] Bounty Hunter License activated for %s\n", playerWallet)
		
	case "arc_net_spy":
		// Reveal target inventory (handled via separate broadcast, not match)
		stats.ArcNetActive = true
		l.leaderboard[playerWallet] = stats
		
	default:
		return false
	}
	
	return true
}

// ApplyUnderworldItemEffect applies the effect of an Underworld faction item to a player's state.
func (l *Lobby) ApplyUnderworldItemEffect(playerWallet string, itemName string, match *MatchState) bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	stats := l.leaderboard[playerWallet]
	if stats.JobRole == "" || l.playerService.GetHegemonyPath(stats.JobRole) != "UNDERWORLD" {
		return false
	}
	
	switch itemName {
	case "data_scramble":
		// Hide match history
		stats.History = []MatchHistory{} // Clear visible history
		if match.ActiveItemBuffs == nil {
			match.ActiveItemBuffs = make(map[string]map[string]bool)
		}
		pID := match.P1Wallet
		if playerWallet == match.P2Wallet {
			pID = match.P2Wallet
		}
		match.ActiveItemBuffs[pID]["data_scrambled"] = true
		log.Printf("[ITEM] Data Scramble activated for %s\n", playerWallet)
		
	case "signal_dampener":
		// Apply signal dampening (stacking) - update rival state
		if stats.Rivalry == nil {
			stats.Rivalry = &RivalryState{}
		}
		stats.Rivalry.HoardPressure += 20 // Base dampening
		l.leaderboard[playerWallet] = stats
		
	case "security_override":
		// Force card return from black market (handled by redemption_gateway.go)
		log.Printf("[ITEM] Security Override activated for %s\n", playerWallet)
		
	case "regulatory_bypass":
		// Set buff expiration on employer club
		if stats.EmployerClubID != "" {
			if club, exists := l.clubs[stats.EmployerClubID]; exists {
				if club.BuffExpirations == nil {
					club.BuffExpirations = make(map[string]time.Time)
				}
				club.BuffExpirations["REGULATORY_BYPASS"] = time.Now().Add(24 * time.Hour)
				l.clubs[stats.EmployerClubID] = club
			}
		}
		
	default:
		return false
	}
	
	return true
}

// ============================================================================
// PILLAR 5: CAREER PROGRESSION XP SYSTEM (Section 13)
// XP accumulation and promotion for all career roles.
// ============================================================================

// TrackCareerXP records XP toward a player's career role progression.
func (l *Lobby) TrackCareerXP(playerWallet string, roleName string, amount uint64) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	stats := l.leaderboard[playerWallet]
	if stats.CareerXP == nil {
		stats.CareerXP = &CareerXP{
			RoleXP:      make(map[string]uint64),
			PromotedRoles: []string{},
		}
	}
	
	stats.CareerXP.RoleXP[roleName] += amount
	
	// Check for promotion eligibility (now includes $VBV gate)
	l.checkCareerPromotion(playerWallet, roleName)
	
	l.leaderboard[playerWallet] = stats
}

// UpdateLiquiditySample records a balance snapshot for $VBV-sustained career gating.
// Per Pillar 13: samples player.VBVBalance * 1_000_000 every active play session,
// keeping last 14 samples (rolling window). AvgSustainedMicro = sum(samples) / len(samples).
func (l *Lobby) UpdateLiquiditySample(playerWallet string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	
	stats := l.leaderboard[playerWallet]
	if stats.CareerXP == nil {
		stats.CareerXP = &CareerXP{
			RoleXP:      make(map[string]uint64),
			PromotedRoles: []string{},
		}
	}
	
	// Record micro-conversion of current balance
	microBalance := uint64(float64(stats.VBVBalance) * 1_000_000)
	stats.CareerXP.LiquiditySamples = append(stats.CareerXP.LiquiditySamples, microBalance)
	
	// Keep last 14 samples (rolling window)
	if len(stats.CareerXP.LiquiditySamples) > 14 {
		stats.CareerXP.LiquiditySamples = stats.CareerXP.LiquiditySamples[len(stats.CareerXP.LiquiditySamples)-14:]
	}
	
	// Compute running average
	stats.CareerXP.computeAvgSustained()
	
	// Check for demotion warning: if avg drops below threshold, issue grace period
	l.checkDemotionWarning(playerWallet, stats)
	
	l.leaderboard[playerWallet] = stats
}

// computeAvgSustained recalculates the average from samples in-place.
func (cxp *CareerXP) computeAvgSustained() {
	if cxp == nil || len(cxp.LiquiditySamples) == 0 {
		cxp.AvgSustainedMicro = 0
		return
	}
	var sum uint64
	for _, s := range cxp.LiquiditySamples {
		sum += s
	}
	cxp.AvgSustainedMicro = sum / uint64(len(cxp.LiquiditySamples))
}

// GetAvgSustainedVBV returns the average sustained $VBV balance (converted from micro).
func (cxp *CareerXP) GetAvgSustainedVBV() float64 {
	if cxp == nil {
		return 0
	}
	return float64(cxp.AvgSustainedMicro) / 1_000_000.0
}

// checkDemotionWarning issues a demotion grace period if avg $VBV drops below tier threshold.
func (l *Lobby) checkDemotionWarning(wallet string, stats PlayerStats) {
	if stats.CareerXP == nil {
		return
	}
	
	// Check each promoted role's $VBV threshold against current average
	for _, role := range stats.CareerXP.PromotedRoles {
		thresholdMicro := getRoleThresholdMicro(role)
		if thresholdMicro == 0 {
			continue // Peon tier, no gate
		}
		
		if stats.CareerXP.AvgSustainedMicro < thresholdMicro {
			// Below threshold — check if grace period has expired
			graceExpires := stats.CareerXP.DemotionWarningAt.Add(7 * 24 * time.Hour)
			if stats.CareerXP.DemotionWarningAt.IsZero() {
				// Issue first warning
				now := time.Now()
				stats.CareerXP.DemotionWarningAt = now
				log.Printf("[CAREER] Demotion warning issued for %s: avg $VBV %.0f below threshold for role %s (%.0f)",
					wallet, stats.CareerXP.GetAvgSustainedVBV(), role, float64(thresholdMicro)/1_000_000.0)
			} else if now := time.Now(); now.After(graceExpires) {
				// Grace period expired — demote
				cxp := stats.CareerXP
				for i, r := range cxp.PromotedRoles {
					if r == role {
						cxp.PromotedRoles = append(cxp.PromotedRoles[:i], cxp.PromotedRoles[i+1:]...)
						log.Printf("[CAREER] Demotion: %s demoted from %s (avg $VBV too low)", wallet, role)
						break
					}
				}
				stats.CareerXP.DemotionWarningAt = time.Time{} // Reset grace period
			}
		} else {
			// Back above threshold — clear warning if it was active
			if !stats.CareerXP.DemotionWarningAt.IsZero() {
				stats.CareerXP.DemotionWarningAt = time.Time{}
				log.Printf("[CAREER] Demotion warning cleared for %s (avg $VBV recovered)", wallet)
			}
		}
	}
	
	l.leaderboard[wallet] = stats
}

// getRoleThresholdMicro returns the $VBV threshold in micro-units for a role tier.
func getRoleThresholdMicro(role string) uint64 {
	switch {
	case role == "Peon" || role == "Freelancer":
		return 0 // No gate for base tier
	default:
		// Use standard thresholds from Pillar 13 table
		// Apprentice ≥ 5K, Journeyman ≥ 25K, Expert ≥ 100K, Master ≥ 500K, Boss ≥ 2M
		return uint64(5_000 * 1_000_000) // Default to Apprentice minimum
	}
}

// GetVbvafter sample check returns the $VBV gate status for career advancement.
// Returns (isGated, reason) — isGated=true means advancement is blocked with explanation.
func GetVbvgate(cxp *CareerXP) (bool, string) {
	if cxp == nil || len(cxp.LiquiditySamples) < 7 {
		return true, "requires 7+ liquidity samples for $VBV gate"
	}
	if cxp.AvgSustainedMicro < uint64(5_000*1_000_000) {
		return true, fmt.Sprintf("avg sustained %.2f VBV below minimum 5,000 VBV", cxp.GetAvgSustainedVBV())
	}
	return false, "" // Passes gate
}

// checkCareerPromotion evaluates if a player qualifies for role promotion.
// Pillar 13: Career tier advancement REQUIRES both XP threshold AND $VBV-sustained gate.
func (l *Lobby) checkCareerPromotion(wallet string, role string) {
	stats := l.leaderboard[wallet]
	cxp := stats.CareerXP
	
	// Calculate overall level
	level := cxp.CalculateLessonLevel()
	
	// Role-specific XP thresholds for promotion
	typeRoleThresholds := map[string]int{
		// Underworld roles
		"Gossip":              10,
		"Fence":               25,
		"Kidnapper":           40,
		"Hostage Host":       60,
		"Lawyer-Commissioner": 55,
		"Underworld Boss":    CareerTierBoss,
		"Arc-Net Operative":  35,
		"Smuggler":           20,
		"Heist Planner":      45,
		"Launderer":          50,
		// Justice roles
		"Intel-Agent":        15,
		"Bounty Hunter":      20,
		"AOS Leader":         30,
		"Justice Recruiter":  25,
		"Justice Commissioner": 40,
		"Mutation Log Auditor": 35,
		"Judge":              CareerTierBoss,
		"Warden":             55,
		"Forensic Analyst":   35,
		"Tax Auditor":        45,
		"Sector Peacekeeper": 30,
	}
	
	threshold, exists := typeRoleThresholds[role]
	if !exists {
		return
	}
	
	// Check if XP meets threshold for this role
	xpForRole := cxp.RoleXP[role]
	if level >= threshold && xpForRole >= uint64(threshold*100) {
		// Pillar 13: $VBV-sustained gate — check liquidity samples first
		gated, reason := GetVbvgate(cxp)
		if gated {
			log.Printf("[CAREER] %s blocked from %s promotion: %s (samples=%d)", wallet, role, reason, len(cxp.LiquiditySamples))
			return // Blocked — not yet promoted
		}
		
		// Check $VBV gate threshold for this specific tier
		tierVBV := getTierThresholdVBV(threshold)
		if tierVBV > 0 && stats.CareerXP.AvgSustainedMicro < uint64(tierVBV*1_000_000) {
			log.Printf("[CAREER] %s blocked from %s: avg %.2f VBV below %.0f VBV tier threshold", 
				wallet, role, stats.CareerXP.GetAvgSustainedVBV(), float64(tierVBV))
			return // Blocked — $VBV not sustained long enough
		}
		
		// Promote to this role (both XP and $VBV gates passed)
		promoted := false
		for _, r := range cxp.PromotedRoles {
			if r == role {
				promoted = true
				break
			}
		}
		
		if !promoted {
			cxp.PromotedRoles = append(cxp.PromotedRoles, role)
			
			// Set the actual job role if not already employed in a different one
			if stats.JobRole == "Freelancer" || stats.JobRole == "" {
				stats.JobRole = role
				l.logAdminAuditLocked("CAREER_PROMOTED", wallet, fmt.Sprintf("Promoted to %s (Level: %d, Avg $VBV: %.0f)", role, level, cxp.GetAvgSustainedVBV()))
				
				notification := fmt.Sprintf(`{"text":"🎖️ <b>PROMOTED:</b> You've been promoted to %s! Career Level: %d | Avg Sustained: $%.0f VBV"}`, escapeHTML(role), level, cxp.GetAvgSustainedVBV())
				l.sendToClientLocked(l.getClientIDFromWalletLocked(wallet), Envelope{
					Type: "admin_notification",
					Payload: json.RawMessage(notification),
				})
				
				log.Printf("[CAREER] %s promoted to %s (Level: %d, Avg $VBV: %.0f)\n", wallet, role, level, cxp.GetAvgSustainedVBV())
			}
		}
	}
}

// getTierThresholdVBV returns the $VBV threshold for a career tier level.
func getTierThresholdVBV(tier int) float64 {
	switch {
	case tier <= CareerTierPeon:
		return 0 // No gate
	case tier <= CareerTierApprentice:
		return 5_000 // Apprentice ≥ 5K $VBV for 7 days
	case tier <= CareerTierJourneyman:
		return 25_000 // Journeyman ≥ 25K $VBV for 10 days
	case tier <= CareerTierExpert:
		return 100_000 // Expert ≥ 100K $VBV for 12 days
	case tier <= CareerTierMaster:
		return 500_000 // Master ≥ 500K $VBV for 14 days
	default:
		return 2_000_000 // Boss ≥ 2M $VBV for 14 days
	}
}

// GetCareerProgress returns a player's career XP progress summary.
// Pillar 13: Includes $VBV-sustained metrics for career advancement gating.
func (l *Lobby) GetCareerProgress(wallet string) map[string]interface{} {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	
	stats := l.leaderboard[wallet]
	if stats.CareerXP == nil {
		return map[string]interface{}{
			"level":            0,
			"total_xp":         uint64(0),
			"promoted_roles":   []string{},
			"available_roles":  []string{},
			"avg_sustained_vbv": 0,
			"liquidity_samples": 0,
			"vbb_gate_status": map[string]interface{}{
				"gated":    true,
				"reason":   "no liquidity history",
				"required": 7,
			},
		}
	}
	
	level := stats.CareerXP.CalculateLessonLevel()
	totalXP := uint64(0)
	for _, xp := range stats.CareerXP.RoleXP {
		totalXP += xp
	}
	
	// $VBV gate status for client UI
	gated, reason := GetVbvgate(stats.CareerXP)
	vbvStatus := map[string]interface{}{
		"gated": gated,
		"reason": reason,
		"required_samples": 7,
	}
	
	return map[string]interface{}{
		"level":               level,
		"total_xp":            totalXP,
		"role_xp":             stats.CareerXP.RoleXP,
		"promoted_roles":      stats.CareerXP.PromotedRoles,
		"current_role":        stats.JobRole,
		"avg_sustained_vbv":   stats.CareerXP.GetAvgSustainedVBV(),
		"avg_sustained_micro": stats.CareerXP.AvgSustainedMicro,
		"liquidity_samples":   len(stats.CareerXP.LiquiditySamples),
		"demotion_warning_at": stats.CareerXP.DemotionWarningAt.Format(time.RFC3339),
		"vbb_gate_status":     vbvStatus,
	}
}

// GetRumorFeeDiscount returns the fee discount multiplier for a Gossip career role.
// Tier 3+ = 20% discount (0.80), lower tiers = no discount (1.0).
func (cxp *CareerXP) GetRumorFeeDiscount() float64 {
	if cxp == nil {
		return 1.0
	}
	xp := cxp.RoleXP["Gossip"]
	if xp >= uint64(CareerTierJourneyman*100) { // Tier 3 threshold: 15 * 100 = 1500 XP
		return 0.80
	}
	return 1.0
}

// GetRumorDiscountActive returns whether a Gossip has the discount buff active.
func (cxp *CareerXP) GetRumorDiscountActive() bool {
	return cxp.GetRumorFeeDiscount() < 1.0
}

// GetGossipRoleTier returns the career tier level for the Gossip role specifically.
// Returns CareerTierPeon (0), Apprentice (5), Journeyman (15), Expert (30), etc.
func (cxp *CareerXP) GetGossipRoleTier() int {
	if cxp == nil {
		return CareerTierPeon
	}
	xp := cxp.RoleXP["Gossip"]
	switch {
	case xp >= uint64(CareerTierBoss*100):
		return CareerTierBoss
	case xp >= uint64(CareerTierMaster*100):
		return CareerTierMaster
	case xp >= uint64(CareerTierExpert*100):
		return CareerTierExpert
	case xp >= uint64(CareerTierJourneyman*100):
		return CareerTierJourneyman
	case xp >= uint64(CareerTierApprentice*100):
		return CareerTierApprentice
	default:
		return CareerTierPeon
	}
}

// GetFenceFeeDiscount returns the fee discount multiplier for a Fence career role.
// Returns 0.50 (50% discount) when Fence XP >= Tier3 threshold, else 1.0 (full fee).
func (cxp *CareerXP) GetFenceFeeDiscount() float64 {
	if cxp == nil {
		return 1.0
	}
	xp := cxp.RoleXP["Fence"]
	// Tier 3 threshold: CareerTierJourneyman (15) * 100 = 1500 XP
	if xp >= uint64(CareerTierJourneyman*100) {
		return 0.50
	}
	return 1.0
}

// GetFenceDiscountActive returns whether a Fence has the discount buff active.
func (cxp *CareerXP) GetFenceDiscountActive() bool {
	return cxp.GetFenceFeeDiscount() < 1.0
}

// ============================================================================
// UNDERWORLD CAREER MECHANIC HOOKS (Tasks 4201-3A through 4201-10B)
// ============================================================================

// GetKidnapSuccessMultiplier returns the kidnap success rate multiplier for Kidnapper role.
// Base rate 50% × multiplier. Tier 1-2: 1.0x, Tier 3+: 1.5x (75%), Tier 4+: 2.0x (100%).
func (cxp *CareerXP) GetKidnapSuccessMultiplier() float64 {
	if cxp == nil {
		return 1.0
	}
	xp := cxp.RoleXP["Kidnapper"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4 threshold: 30 * 100 = 3000 XP
		return 2.0 // 100% base success
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3 threshold: 15 * 100 = 1500 XP
		return 1.5 // 75% base success
	default:
		return 1.0 // 50% base success
	}
}

// GetSignalDampenerStacking returns the number of signal dampeners a Hostage Host can maintain.
// Each tier adds +2 stack capacity. Tier 1-2: 2 stacks, Tier 3+: 4 stacks, Tier 4+: 6 stacks.
func (cxp *CareerXP) GetSignalDampenerStacking() int {
	if cxp == nil {
		return 2
	}
	xp := cxp.RoleXP["Hostage Host"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 6
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 4
	default:
		return 2
	}
}

// GetCorporateTaxDiscount returns the corporate tax reduction for Lawyer-Commissioner.
// Base rate is defined in economy config. Tier 1-2: 0% discount, Tier 3+: 25% off, Tier 4+: 50% off.
func (cxp *CareerXP) GetCorporateTaxDiscount() float64 {
	if cxp == nil {
		return 0.0
	}
	xp := cxp.RoleXP["Lawyer-Commissioner"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 0.50 // 50% tax reduction
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 0.25 // 25% tax reduction
	default:
		return 0.0
	}
}

// GetIntelBidModifier returns the intel auction bid modifier for Underworld Boss.
// Tier 1-2: 1.0x (no modifier), Tier 3+: +10% max bid, Tier 4+: +25% max bid.
func (cxp *CareerXP) GetIntelBidModifier() float64 {
	if cxp == nil {
		return 1.0
	}
	xp := cxp.RoleXP["Underworld Boss"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 1.25
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 1.10
	default:
		return 1.0
	}
}

// GetSpyVisionDuration returns the Arc-Net operative spy vision duration in minutes.
// Base is 60s. Tier 1-2: ×1, Tier 3+: ×2, Tier 4+: ×3.
func (cxp *CareerXP) GetSpyVisionDuration() float64 {
	if cxp == nil {
		return 1.0
	}
	xp := cxp.RoleXP["Arc-Net Operative"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 3.0
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 2.0
	default:
		return 1.0
	}
}

// GetTransitTaxExemption returns the fraction of transit taxes exempt for Smuggler role.
// Tier 1-2: 0% exempt, Tier 3+: 50% exempt, Tier 4+: 100% exempt.
func (cxp *CareerXP) GetTransitTaxExemption() float64 {
	if cxp == nil {
		return 0.0
	}
	xp := cxp.RoleXP["Smuggler"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 1.0
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 0.50
	default:
		return 0.0
	}
}

// GetHeistBonusMultiplier returns the heist profit bonus multiplier for Heist Planner.
// Tier 1-2: ×1.0 (no bonus), Tier 3+: ×1.25, Tier 4+: ×1.50.
func (cxp *CareerXP) GetHeistBonusMultiplier() float64 {
	if cxp == nil {
		return 1.0
	}
	xp := cxp.RoleXP["Heist Planner"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 1.50
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 1.25
	default:
		return 1.0
	}
}

// GetHeistXPModifier returns the XP multiplier for Heist Planner when participating in heists.
// Tier 1-2: ×1.0, Tier 3+: ×1.5, Tier 4+: ×2.0.
func (cxp *CareerXP) GetHeistXPModifier() float64 {
	if cxp == nil {
		return 1.0
	}
	xp := cxp.RoleXP["Heist Planner"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 2.0
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 1.5
	default:
		return 1.0
	}
}

// GetCleanMoneyMultiplier returns the laundered money multiplier for Launderer role.
// Tier 1-2: ×0.30 (30% clean), Tier 3+: ×0.50, Tier 4+: ×0.75.
func (cxp *CareerXP) GetCleanMoneyMultiplier() float64 {
	if cxp == nil {
		return 0.30
	}
	xp := cxp.RoleXP["Launderer"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 0.75
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 0.50
	default:
		return 0.30
	}
}

// ============================================================================
// JUSTICE CAREER MECHANIC HOOKS (P2-D1 through P2-D6)
// ============================================================================

// GetTaxPenaltyRate returns the tax penalty multiplier for Tax Auditor role.
// Tier 1-2: ×1.0 (no penalty), Tier 3+: ×1.5, Tier 4+: ×2.0 on evaded amounts.
func (cxp *CareerXP) GetTaxPenaltyRate() float64 {
	if cxp == nil {
		return 1.0
	}
	xp := cxp.RoleXP["Tax Auditor"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 2.0
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 1.5
	default:
		return 1.0
	}
}

// GetMonitoringBonus returns the monitoring radius bonus for Warden role.
// Each tier level adds +1 to the base monitoring radius of 3.
func (cxp *CareerXP) GetMonitoringBonus() int {
	if cxp == nil {
		return 3 // Base radius
	}
	xp := cxp.RoleXP["Warden"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 6
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 5
	case xp >= uint64(CareerTierApprentice*100): // Tier 2: 5 * 100 = 500 XP
		return 4
	default:
		return 3
	}
}

// GetRaidSuccessModifier returns the AOS raid success chance modifier.
// Base raid success is 40%. Tier 1-2: ×1.0, Tier 3+: ×1.25, Tier 4+: ×1.50.
func (cxp *CareerXP) GetRaidSuccessModifier() float64 {
	if cxp == nil {
		return 1.0
	}
	xp := cxp.RoleXP["AOS"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 1.50
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 1.25
	default:
		return 1.0
	}
}

// GetTraceDepthMultiplier returns the forensic trace depth multiplier.
// Base depth is 1 hop. Each tier adds +1 to maximum trace depth.
func (cxp *CareerXP) GetTraceDepthMultiplier() int {
	if cxp == nil {
		return 1
	}
	xp := cxp.RoleXP["Forensic Analyst"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 4
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 3
	case xp >= uint64(CareerTierApprentice*100): // Tier 2: 5 * 100 = 500 XP
		return 2
	default:
		return 1
	}
}

// GetPeacekeeperCombatBonus returns the combat power bonus for Sector Peacekeeper.
// Tier 1-2: ×1.0, Tier 3+: +15% combat power, Tier 4+: +30% combat power.
func (cxp *CareerXP) GetPeacekeeperCombatBonus() float64 {
	if cxp == nil {
		return 1.0
	}
	xp := cxp.RoleXP["Sector Peacekeeper"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 1.30
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 1.15
	default:
		return 1.0
	}
}

// GetPeacekeeperPatrolRadius returns the patrol radius bonus for Sector Peacekeeper.
// Base radius is 2 tiles. Each tier adds +1 tile to patrol range.
func (cxp *CareerXP) GetPeacekeeperPatrolRadius() int {
	if cxp == nil {
		return 2 // Base patrol radius
	}
	xp := cxp.RoleXP["Sector Peacekeeper"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 5
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 4
	case xp >= uint64(CareerTierApprentice*100): // Tier 2: 5 * 100 = 500 XP
		return 3
	default:
		return 2
	}
}

// ============================================================================
// JUSTICE CAREER MECHANIC HOOKS — P2-D7 through P2-D10 (Phase 2 completion)
// ============================================================================

// GetIntelDecryptDepth returns the cyber-intercept decrypt depth for Intel-Agent.
// Base depth is 1 hop. Each tier adds +1 to maximum intercept range.
// Tier 1-2: 1 hop, Tier 3+: 2 hops, Tier 4+: 3 hops.
func (cxp *CareerXP) GetIntelDecryptDepth() int {
	if cxp == nil {
		return 1 // Base depth
	}
	xp := cxp.RoleXP["Intel-Agent"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 3
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 2
	case xp >= uint64(CareerTierApprentice*100): // Tier 2: 5 * 100 = 500 XP
		return 2
	default:
		return 1
	}
}

// GetIntelDecryptPercentage returns the Arc-Net vision decryption percentage for Intel-Agent.
// When targeting an Arc-Net Operative, reveals their hidden data with this accuracy.
// Tier 1-2: 50%, Tier 3+: 75%, Tier 4+: 100%.
func (cxp *CareerXP) GetIntelDecryptPercentage() float64 {
	if cxp == nil {
		return 0.50
	}
	xp := cxp.RoleXP["Intel-Agent"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 1.0
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 0.75
	case xp >= uint64(CareerTierApprentice*100): // Tier 2: 5 * 100 = 500 XP
		return 0.60
	default:
		return 0.50
	}
}

// GetIntelXPBase returns the base XP awarded for Intel-Agent cyber-intercept events.
// Tier 1-2: +40 XP, Tier 3+: +70 XP, Tier 4+: +100 XP.
func (cxp *CareerXP) GetIntelXPBase() int {
	if cxp == nil {
		return 40
	}
	xp := cxp.RoleXP["Intel-Agent"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 100
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 70
	default:
		return 40
	}
}

// GetRecruiterPowerBonus returns the recruitment power bonus for Justice Recruiter.
// When recruiting a new justice-aligned player, grants this multiplicative power buff.
// Tier 1-2: ×1.0 (no bonus), Tier 3+: +5%, Tier 4+: +10%.
func (cxp *CareerXP) GetRecruiterPowerBonus() float64 {
	if cxp == nil {
		return 1.0
	}
	xp := cxp.RoleXP["Justice Recruiter"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 1.10
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 1.05
	default:
		return 1.0
	}
}

// GetRecruiterXPBase returns the base XP awarded for Justice Recruiter recruitment events.
// Tier 1-2: +30 XP, Tier 3+: +55 XP, Tier 4+: +80 XP.
func (cxp *CareerXP) GetRecruiterXPBase() int {
	if cxp == nil {
		return 30
	}
	xp := cxp.RoleXP["Justice Recruiter"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 80
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 55
	default:
		return 30
	}
}

// GetRecruiterRecruitRange returns the maximum distance (in tiles) for recruitment outreach.
// Base range is 3 tiles. Each tier adds +2 tiles to range.
func (cxp *CareerXP) GetRecruiterRecruitRange() int {
	if cxp == nil {
		return 3 // Base recruitment range
	}
	xp := cxp.RoleXP["Justice Recruiter"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 7
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 5
	case xp >= uint64(CareerTierApprentice*100): // Tier 2: 5 * 100 = 500 XP
		return 4
	default:
		return 3
	}
}

// GetCommissionerOverrideRate returns the regulatory override rate for Justice Commissioner.
// Determines how much a Commissioner can modify Tax Auditor fiscal actions.
// Tier 1-2: ×1.0 (no override), Tier 3+: ×1.25, Tier 4+: ×1.50.
func (cxp *CareerXP) GetCommissionerOverrideRate() float64 {
	if cxp == nil {
		return 1.0
	}
	xp := cxp.RoleXP["Justice Commissioner"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 1.50
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 1.25
	default:
		return 1.0
	}
}

// GetCommissionerXPBase returns the base XP awarded for Justice Commissioner override events.
// Tier 1-2: +50 XP, Tier 3+: +80 XP, Tier 4+: +120 XP.
func (cxp *CareerXP) GetCommissionerXPBase() int {
	if cxp == nil {
		return 50
	}
	xp := cxp.RoleXP["Justice Commissioner"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 120
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 80
	default:
		return 50
	}
}

// GetCommissionerAuthorityRadius returns the jurisdiction radius in tiles for Justice Commissioner.
// Base radius is 2 tiles. Each tier adds +2 tiles to authority area.
func (cxp *CareerXP) GetCommissionerAuthorityRadius() int {
	if cxp == nil {
		return 2 // Base authority radius
	}
	xp := cxp.RoleXP["Justice Commissioner"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 6
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 4
	case xp >= uint64(CareerTierApprentice*100): // Tier 2: 5 * 100 = 500 XP
		return 3
	default:
		return 2
	}
}

// GetMutationAuditReveal returns the hidden data reveal percentage for Mutation Log Auditor.
// When auditing a mutation event, reveals this percentage of suppressed genetic data.
// Tier 1-2: 40%, Tier 3+: 65%, Tier 4+: 100%.
func (cxp *CareerXP) GetMutationAuditReveal() float64 {
	if cxp == nil {
		return 0.40
	}
	xp := cxp.RoleXP["Mutation Log Auditor"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 1.0
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 0.65
	case xp >= uint64(CareerTierApprentice*100): // Tier 2: 5 * 100 = 500 XP
		return 0.50
	default:
		return 0.40
	}
}

// GetMutationAuditXPBase returns the base XP awarded for Mutation Log Auditor audit events.
// Tier 1-2: +35 XP, Tier 3+: +60 XP, Tier 4+: +90 XP.
func (cxp *CareerXP) GetMutationAuditXPBase() int {
	if cxp == nil {
		return 35
	}
	xp := cxp.RoleXP["Mutation Log Auditor"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 90
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 60
	default:
		return 35
	}
}

// GetMutationAuditScope returns the mutation log search scope (number of past events reviewed).
// Base scope is 5 past events. Each tier adds +5 to scope.
func (cxp *CareerXP) GetMutationAuditScope() int {
	if cxp == nil {
		return 5 // Base audit scope
	}
	xp := cxp.RoleXP["Mutation Log Auditor"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 15
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 10
	case xp >= uint64(CareerTierApprentice*100): // Tier 2: 5 * 100 = 500 XP
		return 8
	default:
		return 5
	}
}

// ============================================================================
// JUSTICE CAREER MECHANIC HOOKS — MISSING FUNCTIONS (Phase 2 addendum)
// ============================================================================

// GetBountyTrackingBonus returns the bounty tracking speed bonus for Bounty Hunter.
// Tier 1-2: ×1.0 (no bonus), Tier 3+: +15% tracking speed, Tier 4+: +25% tracking speed.
func (cxp *CareerXP) GetBountyTrackingBonus() float64 {
	if cxp == nil {
		return 1.0
	}
	xp := cxp.RoleXP["Bounty Hunter"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 1.25
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 1.15
	default:
		return 1.0
	}
}

// GetAuditPrecisionBonus returns the audit reveal percentage for Tax Auditor.
// Tier 1-2: ×1.0 (no bonus), Tier 3+: reveals +50% of hidden revenue, Tier 4+: reveals +75%.
func (cxp *CareerXP) GetAuditPrecisionBonus() float64 {
	if cxp == nil {
		return 1.0
	}
	xp := cxp.RoleXP["Tax Auditor"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 1.75
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 1.50
	default:
		return 1.0
	}
}

// GetWardenDetentionBonus returns the detention duration multiplier for Warden.
// Tier 1-2: ×1.0 (no bonus), Tier 3+: ×1.5 duration, Tier 4+: ×2.0 duration.
func (cxp *CareerXP) GetWardenDetentionBonus() float64 {
	if cxp == nil {
		return 1.0
	}
	xp := cxp.RoleXP["Warden"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 2.0
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 1.5
	default:
		return 1.0
	}
}

// GetEvidenceAccuracyBonus returns the evidence accuracy multiplier for Forensic Analyst.
// Tier 1-2: ×1.0 (standard), Tier 3+: ×2.0 effectiveness (double-clean), Tier 4+: ×3.0.
func (cxp *CareerXP) GetEvidenceAccuracyBonus() float64 {
	if cxp == nil {
		return 1.0
	}
	xp := cxp.RoleXP["Forensic Analyst"]
	switch {
	case xp >= uint64(CareerTierExpert*100): // Tier 4: 30 * 100 = 3000 XP
		return 3.0
	case xp >= uint64(CareerTierJourneyman*100): // Tier 3: 15 * 100 = 1500 XP
		return 2.0
	default:
		return 1.0
	}
}

// ============================================================================
// RIVAL PAIR MECHANICS (P2-E: Cross-career rival interactions)
// Each function checks if target has a rival career and returns bonus/penalty.
// ============================================================================

// GetRivalPairModifier returns the XP/rival modifier for a specific attacker↔defender career pair.
// It implements the 6 defined rival pairs with tier-aware bonuses.
func (cxp *CareerXP) GetRivalPairModifier(targetCareer string, myTier int) float64 {
	if cxp == nil || targetCareer == "" || myTier <= 0 {
		return 1.0
	}

	switch {
	// ---- ENEMY RIVAL PAIRS (negative modifier = bonus for attacker) ----

	// Bounty Hunter ↔ Kidnapper: On capture, enhanced tracking if target is Kidnapper
	case cxp.HasCareer("Bounty Hunter") && targetCareer == "Kidnapper":
		return getBonusForTier(myTier, 1.15, 1.25) // Tier3/4 tracking bonus

	// Forensic Analyst ↔ Gossip: On raid, double-clean if target is Gossip
	case cxp.HasCareer("Forensic Analyst") && targetCareer == "Gossip":
		return getBonusForTier(myTier, 2.0, 3.0) // Tier3/4 evidence multiplier

	// Tax Auditor ↔ Launderer: On audit, reveals +50%/+75% if target is Launderer
	case cxp.HasCareer("Tax Auditor") && targetCareer == "Launderer":
		return getBonusForTier(myTier, 1.50, 1.75) // Tier3/4 reveal bonus

	// Warden ↔ Heist Planner: On mission complete, double detention if target is Planner
	case cxp.HasCareer("Warden") && targetCareer == "Heist Planner":
		return getBonusForTier(myTier, 1.5, 2.0) // Tier3/4 detention multiplier

	// Sector Peacekeeper ↔ Smuggler: On patrol, blocks routes
	case cxp.HasCareer("Sector Peacekeeper") && targetCareer == "Smuggler":
		return getBonusForTier(myTier, 1.15, 1.30) // Tier3/4 combat bonus for block

	// Intel-Agent ↔ Arc-Net Operative: On cyber-intercept, decrypts vision
	case cxp.HasCareer("Intel-Agent") && targetCareer == "Arc-Net Operative":
		return getBonusForTier(myTier, 1.5, 2.0) // Tier3/4 decryption bonus

	// ---- ALLY RIVAL PAIRS (positive modifier = synergy bonus) ----

	// Justice Recruiter ↔ Bounty Hunter: On recruit, +5% power for new justice player
	case cxp.HasCareer("Justice Recruiter") && targetCareer == "Bounty Hunter":
		return getBonusForTier(myTier, 1.05, 1.10) // Tier3/4 recruitment bonus

	// Launderer ↔ Fence: On fence transaction, synergistic money flow
	case cxp.HasCareer("Launderer") && targetCareer == "Fence":
		return getBonusForTier(myTier, 1.20, 1.25) // Tier3/4 fee discount synergy

	// Heist Planner ↔ Kidnapper: On heist with hostage, team bonus
	case cxp.HasCareer("Heist Planner") && targetCareer == "Kidnapper":
		return getBonusForTier(myTier, 1.05, 1.10) // Tier3/4 team synergy

	// AOS ↔ Sector Peacekeeper: On org patrol, coordination bonus
	case cxp.HasCareer("AOS") && targetCareer == "Sector Peacekeeper":
		return getBonusForTier(myTier, 1.08, 1.12) // Tier3/4 coordination bonus

	// Tax Auditor ↔ Justice Commissioner: On commissioner override, fiscal alliance
	case cxp.HasCareer("Tax Auditor") && targetCareer == "Justice Commissioner":
		return getBonusForTier(myTier, 1.20, 1.25) // Tier3/4 revenue share bonus

	// Gossip ↔ Forensic Analyst (Ally variant): Org synergy
	case cxp.HasCareer("Gossip") && targetCareer == "Forensic Analyst":
		return getBonusForTier(myTier, 1.10, 1.20)

	// P2-D7: Intel-Agent ↔ Arc-Net Operative (ally variant for same-org intel sharing)
	case cxp.HasCareer("Intel-Agent") && targetCareer == "Arc-Net Operative":
		return getBonusForTier(myTier, 1.15, 1.20)

	// P2-D8: Justice Recruiter ↔ Mutation Log Auditor (ally pair - shared mutation data)
	case cxp.HasCareer("Justice Recruiter") && targetCareer == "Mutation Log Auditor":
		return getBonusForTier(myTier, 1.10, 1.15)

	// P2-D9: Justice Commissioner ↔ Tax Auditor (synergy via override alliance)
	case cxp.HasCareer("Justice Commissioner") && targetCareer == "Tax Auditor":
		return getBonusForTier(myTier, 1.20, 1.30)

	// P2-D10: Mutation Log Auditor ↔ Kidnapper (antagonistic - tracks their genetic tampering)
	case cxp.HasCareer("Mutation Log Auditor") && targetCareer == "Kidnapper":
		return getBonusForTier(myTier, 1.40, 1.50)

	default:
		return 1.0
	}
}

// HasCareer checks if the player has promoted to a specific career role.
func (cxp *CareerXP) HasCareer(role string) bool {
	if cxp == nil {
		return false
	}
	for _, r := range cxp.PromotedRoles {
		if r == role {
			return true
		}
	}
	return cxp.RoleXP != nil && cxp.RoleXP[role] > 0
}

// getBonusForTier returns the tier-aware bonus: Tier1-2 → noBonus, Tier3 → tier3Bonus, Tier4+ → tier4Bonus.
func getBonusForTier(tier int, tier3Bonus float64, tier4Bonus float64) float64 {
	switch {
	case tier >= 4:
		return tier4Bonus
	case tier >= 3:
		return tier3Bonus
	default:
		return 1.0
	}
}

// GetRivalXPDelta returns the rivalry XP delta for a given rival pair type.
// Negative = antagonistic (rival), Positive = synergistic (ally).
func GetRivalXPDelta(pairType string) int {
	switch pairType {
	// Antagonistic pairs (existing)
	case "BountyHunter↔Kidnapper", "BountyHunter_Kidnapper":
		return -10
	case "ForensicAnalyst↔Gossip", "ForensicAnalyst_Gossip":
		return -10
	case "TaxAuditor↔Launderer", "TaxAuditor_Launderer":
		return -10
	case "Warden↔HeistPlanner", "Warden_HeistPlanner":
		return -10
	case "SectorPeacekeeper↔Smuggler", "SectorPeacekeeper_Smuggler":
		return -10
	case "IntelAgent↔ArcNetOperative", "IntelAgent_ArcNetOperative":
		return -10
	// Synergistic pairs (existing)
	case "JusticeRecruiter↔BountyHunter", "JusticeRecruiter_BountyHunter":
		return +8
	case "Launderer↔Fence", "Launderer_Fence":
		return +5
	case "HeistPlanner↔Kidnapper", "HeistPlanner_Kidnapper":
		return +12
	case "AOS↔SectorPeacekeeper", "AOS_SectorPeacekeeper":
		return +6
	case "TaxAuditor↔JusticeCommissioner", "TaxAuditor_JusticeCommissioner":
		return +7
	case "Gossip↔ForensicAnalyst", "Gossip_ForensicAnalyst":
		return +5
	default:
		return 0
	}
}

// GetRivalPairName returns the canonical rival pair name for a given attacker-defender combination.
func GetRivalPairName(attackerCareer, defenderCareer string) string {
	// Check all pairs including P2-D7 through P2-D10
	pairs := []struct{ A, B, Name string }{
		{"Bounty Hunter", "Kidnapper", "BountyHunter↔Kidnapper"},
		{"Forensic Analyst", "Gossip", "ForensicAnalyst↔Gossip"},
		{"Tax Auditor", "Launderer", "TaxAuditor↔Launderer"},
		{"Warden", "Heist Planner", "Warden↔HeistPlanner"},
		{"Sector Peacekeeper", "Smuggler", "SectorPeacekeeper↔Smuggler"},
		{"Intel-Agent", "Arc-Net Operative", "IntelAgent↔ArcNetOperative"},
		{"Justice Recruiter", "Bounty Hunter", "JusticeRecruiter↔BountyHunter"},
		{"Launderer", "Fence", "Launderer↔Fence"},
		{"Heist Planner", "Kidnapper", "HeistPlanner↔Kidnapper"},
		{"AOS", "Sector Peacekeeper", "AOS↔SectorPeacekeeper"},
		{"Tax Auditor", "Justice Commissioner", "TaxAuditor↔JusticeCommissioner"},
		{"Gossip", "Forensic Analyst", "Gossip↔ForensicAnalyst"},
		// P2-D8: Justice Recruiter ↔ Mutation Log Auditor (ally pair)
		{"Justice Recruiter", "Mutation Log Auditor", "JusticeRecruiter↔MutationLogAuditor"},
		// P2-D9: Justice Commissioner ↔ Tax Auditor (allied override)
		{"Justice Commissioner", "Tax Auditor", "JusticeCommissioner↔TaxAuditor"},
		// P2-D10: Mutation Log Auditor ↔ Kidnapper (antagonistic - tracks their genetic tampering)
		{"Mutation Log Auditor", "Kidnapper", "MutationLogAuditor↔Kidnapper"},
	}
	for _, p := range pairs {
		if (attackerCareer == p.A && defenderCareer == p.B) ||
			(attackerCareer == p.B && defenderCareer == p.A) {
			return p.Name
		}
	}
	return ""
}

// TrackRivalInteraction evaluates a cross-career interaction between attacker and defender,
// returns (xpAwarded, rivalPairName, isRivalPair).
// P2-A: Core wiring function — called at criminality handlers.
func TrackRivalInteraction(
	attackerCareer, defenderCareer string,
	baseXP uint64,
	myStats *PlayerStats,
	targetStats *PlayerStats,
) (uint64, string, bool) {
	if baseXP == 0 || attackerCareer == "" || defenderCareer == "" || myStats == nil || targetStats == nil {
		return baseXP, "", false
	}

	pairName := GetRivalPairName(attackerCareer, defenderCareer)
	if pairName == "" {
		return baseXP, "", false
	}

	delta := GetRivalXPDelta(pairName)
	modifier := myStats.CareerXP.GetRivalPairModifier(defenderCareer, getTierFor(myStats.CareerXP, attackerCareer))

	var xpAwarded uint64
	if delta < 0 {
		// Antagonistic: attacker bonus scaled by absolute delta + modifier
		scaling := 1.0 + float64(-delta)/100.0
		xpAwarded = uint64(float64(baseXP) * scaling * modifier)
	} else {
		// Synergistic: attacker bonus scaled by delta + modifier
		scaling := 1.0 + float64(delta)/100.0
		xpAwarded = uint64(float64(baseXP) * scaling * modifier)
	}

	return xpAwarded, pairName, true
}

// getTierFor returns the career tier for a role from the player's XP map.
func getTierFor(cxp *CareerXP, role string) int {
	if cxp == nil || cxp.RoleXP == nil {
		return 1
	}
	xp := cxp.RoleXP[role]
	switch {
	case xp >= uint64(CareerTierMaster*100): // Tier 4: 50 * 100 = 5000 XP
		return 4
	case xp >= uint64(CareerTierExpert*100): // Tier 3: 30 * 100 = 3000 XP
		return 3
	case xp >= uint64(CareerTierApprentice*100): // Tier 2: 5 * 100 = 500 XP
		return 2
	default:
		return 1
	}
}

// EvaluateCrossCareerXP applies rival modifiers for both attacker and defender.
// Returns (attackerXP, defenderXP, pairName, isRivalPair).
// P2-A: Called at interaction resolution to distribute XP fairly between both parties.
func EvaluateCrossCareerXP(
	attackerCareer, defenderCareer string,
	baseAttackerXP uint64,
	myStats *PlayerStats,
	targetStats *PlayerStats,
) (uint64, uint64, string, bool) {
	if baseAttackerXP == 0 || attackerCareer == "" || defenderCareer == "" || myStats == nil || targetStats == nil {
		return baseAttackerXP, 0, "", false
	}

	pairName := GetRivalPairName(attackerCareer, defenderCareer)
	if pairName == "" {
		return baseAttackerXP, 0, "", false
	}

	delta := GetRivalXPDelta(pairName)
	modifier := myStats.CareerXP.GetRivalPairModifier(defenderCareer, getTierFor(myStats.CareerXP, attackerCareer))

	var attackerXP uint64
	if delta < 0 {
		scaling := 1.0 + float64(-delta)/100.0
		attackerXP = uint64(float64(baseAttackerXP) * scaling * modifier)
	} else {
		scaling := 1.0 + float64(delta)/100.0
		attackerXP = uint64(float64(baseAttackerXP) * scaling * modifier)
	}

	// Defender gets a reduced reward (30% of base, representing monitoring/suppression activity)
	defenderXP := uint64(float64(baseAttackerXP) * 0.30)

	return attackerXP, defenderXP, pairName, true
}

// ============================================================================
// HELPER: Map wallet to match player ID for buff application.
// ============================================================================

func playerStatsToID(stats PlayerStats) string {
	// This is a placeholder - in production this would resolve via the lobby's wallet-to-ID map
	return stats.ID
}

// computeScaledXP scales raw career XP by loyalty/fame bonuses from rival pairings and active stats.
// Loyalty bonus represents social influence (rumor discounts, trust).
// Fame bonus represents public visibility/recognition (bounty multipliers, reputation buffs).
func (cxp *CareerXP) computeScaledXP(baseXP uint64, loyaltyBonus, fameBonus float64) uint64 {
	if cxp == nil || baseXP == 0 {
		return baseXP
	}

	scaler := 1.0

	// Apply loyalty bonus (social influence)
	if loyaltyBonus > 0.0 {
		scaler *= 1.0 + loyaltyBonus
	}

	// Apply fame bonus (public visibility / recognition)
	if fameBonus > 0.0 {
		scaler *= 1.0 + fameBonus
	}

	xp := uint64(float64(baseXP) * scaler)

	return xp
}
