//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"
)

// BlackMarketService encapsulates logic for the underworld economy.
// PILLAR 5: Stateless Service Design.
type BlackMarketService struct{}

// HandleSellToBlackMarket processes a player's request to sell a card to the Black Market.
// This provides a new avenue for criminals to liquidate kidnapped assets or other illicit goods.
// PILLAR 3: Silkroad Expansion.
func (bs *BlackMarketService) HandleSellToBlackMarket(l *Lobby, env *Envelope) {
	var data struct {
		CardID int `json:"card_id"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		log.Printf("[BLACK_MARKET] Invalid sell_to_black_market payload from %s: %v\n", env.FromID, err)
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Invalid sell request."}`)})
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	wallet, ok := l.wallets[env.FromID]
	if !ok {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Wallet not registered."}`)})
		return
	}

	stats, exists := l.leaderboard[wallet]
	if !exists {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Player stats not found."}`)})
		return
	}

	// 1. Verify card ownership (either in inventory or held hostage)
	cardKey := fmt.Sprintf("CARD-%d", data.CardID)
	qty, hasCard := stats.Inventory[cardKey]
	if !hasCard || qty <= 0 {
		// Check if it's a held hostage card
		_, isHostage := stats.KidnappedCards[data.CardID]
		if !isHostage {
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Card not found in your inventory or held hostages."}`)})
			return
		}
	}

	card, cardExists := l.inventory[data.CardID]
	if !cardExists {
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Card metadata not found."}`)})
		return
	}

	// 2. Calculate Black Market Price (e.g., 50% of estimated market value)
	// For simplicity, let's use a fixed percentage of its base power value for now.
	// A more complex system would involve market demand, rarity, etc.
	estimatedValueMicro := uint64(card.Power[0]+card.Power[1]+card.Power[2]+card.Power[3]) * 100000 // Example: 100 micro-VBV per power point
	blackMarketPriceMicro := estimatedValueMicro / 2                                                // 50% of estimated value

	// 3. Apply Fence Fee (Standard: 10%).
	// PILLAR 3: Career Path influence. 'Fences' incur 50% lower protocol taxes.
	// Career XP Trigger (#2A): +30 XP per fence sale toward Fence career progression.
	fenceFeePercent := 10
	var fenceDiscount float64 = 1.0

	if stats.JobRole == "Fence" {
		fenceFeePercent = 5 // Base discount for having the role
		// PILLAR 8: Fence Fee Discount — checks CareerXP.RoleXP["Fence"] >= Tier3 → returns 0.50
		if stats.CareerXP != nil {
			fenceDiscount = stats.CareerXP.GetFenceFeeDiscount()
		}
	}

	fenceFeeMicro := (blackMarketPriceMicro * uint64(fenceFeePercent)) / 100

	// Apply discount multiplier if Fence tier >= 3
	if fenceDiscount < 1.0 {
		fenceFeeMicro = uint64(float64(fenceFeeMicro) * fenceDiscount)
		// Add "Fenced Rate Active" buff tag
		if stats.Buffs == nil {
			stats.Buffs = make(map[string]bool)
		}
		stats.Buffs["fenced_rate_active"] = true
	}
	netPayoutMicro := blackMarketPriceMicro - fenceFeeMicro

	// 4. Transfer funds and update inventory
	if hasCard {
		stats.Inventory[cardKey]--
		if stats.Inventory[cardKey] <= 0 {
			delete(stats.Inventory, cardKey)
		}
	} else { // It was a kidnapped card
		// PILLAR 3: Underworld Liquidation.
		// Clear the hostage records from the system and the victim's profile.
		if kidnap, active := l.activeKidnappings[data.CardID]; active {
			if victimStats, vExists := l.leaderboard[kidnap.VictimWallet]; vExists {
				delete(victimStats.HeldHostageCards, data.CardID)
				l.leaderboard[kidnap.VictimWallet] = victimStats
			}
			delete(l.activeKidnappings, data.CardID)

			// PILLAR 3: Multi-Slot Registry Cleanup.
			l.victimRegistry.Mu.Lock()
			if attackers, exists := l.victimRegistry.ActiveKidnaps[kidnap.VictimWallet]; exists {
				delete(attackers, wallet)
				if len(attackers) == 0 {
					delete(l.victimRegistry.ActiveKidnaps, kidnap.VictimWallet)
				}
			}
			l.victimRegistry.Mu.Unlock()
		}
		delete(stats.KidnappedCards, data.CardID)
	}

	// PILLAR 7: Asset Devaluation.
	// Cards sold to the black market are marked as 'Fallen' immediately.
	card.Fallen = true
	l.inventory[data.CardID] = card

	l.playerBalances[wallet] += netPayoutMicro
	stats.WantedLevel += 1 // Selling illicit goods increases Wanted Level
	stats.Reputation = l.CalculateReputation(stats)

	// PILLAR 3: Underworld Contract Completion (CONTRACT-024)
	// Black Market Monopoly: Sell an Epic or Legendary card.
	if stats.ActiveUnderworldContractID == "CONTRACT-024" && stats.JobRole == "Fence" {
		// PILLAR 3: Rarity Alignment. Epic >= 1.5, Legendary >= 2.0.
		if card.Rarity >= 1.5 {
			const rewardMicro = 30000 * 1000000
			l.playerBalances[wallet] += rewardMicro
			stats.ActiveUnderworldContractID = ""
			l.logAdminAuditLocked("CONTRACT_COMPLETED", wallet, "ID: CONTRACT-024, Payout: 30000.00")
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> Elite asset fenced. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
			l.applyDynamicScalingLocked()
		}
	}

	l.leaderboard[wallet] = stats

	// PILLAR 13: Career Progression — Fence XP trigger (#2A).
	// +30 XP toward Fence career progression per asset fenced.
	// Phase 4: Scale XP by loyalty/fame multipliers from career state (Task 4301).
	baseFenceXP := uint64(30)
	if stats.CareerXP != nil {
		lessonsCompleted := int32(0)
		for _, tier := range stats.CareerXP.Tiers {
			lessonsCompleted += tier.LessonsCompleted
		}
		loyaltyBonus := 0.0
		if lessonsCompleted > 0 {
			loyaltyBonus = math.Min(0.50, float64(lessonsCompleted)/200.0)
		}
		fameBonus := 0.0
		for _, tier := range stats.CareerXP.Tiers {
			if tier.LessonsCompleted > 0 && tier.StandingTier > 0 {
				tierFame := float64(tier.StandingTier) * 0.10
				if tierFame > fameBonus {
					fameBonus = tierFame
				}
			}
		}
		fameBonus = math.Min(0.60, fameBonus)
		scaledXP := stats.CareerXP.computeScaledXP(baseFenceXP, loyaltyBonus, fameBonus)
		l.TrackCareerXP(wallet, "Fence", scaledXP)
	} else {
		l.TrackCareerXP(wallet, "Fence", baseFenceXP)
	}

	// 5. Route Fence Fee to Faucet via Token-Sink Router
	if l.tokenSinkRouter != nil {
		matrix := RevenueSplitMatrix{FaucetShare: 1.0, ClubShare: 0.0, GovernanceShare: 0.0}
		_ = l.tokenSinkRouter.RouteCriminalTax("BLACK_MARKET_FENCE_FEE", fenceFeeMicro, matrix, 0, "")
	}

	// 6. Add card to Black Market's pool of liquidated assets (for future purchase by others)
	// For now, we just "burn" it into the black market, but a future feature could make it purchasable.
	log.Printf("[BLACK_MARKET] Card %d sold by %s for %d micro-VBV (Fence Fee: %d micro-VBV).\n", data.CardID, wallet, netPayoutMicro, fenceFeeMicro)
	l.logAdminAuditLocked("BLACK_MARKET_SELL", wallet, fmt.Sprintf("Sold Card %d for %d micro-VBV (Fence Fee: %d)", data.CardID, netPayoutMicro, fenceFeeMicro))

	l.applyDynamicScalingLocked()
	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🎉 <b>ASSET FENCED:</b> Card #%d sold for %.2f $VBV."}`, data.CardID, float64(netPayoutMicro)/1000000.0))})

	// Trigger Global Sync to update balances and inventory
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// UnderworldContract represents a task available for criminal players.
type UnderworldContract struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	RewardMicro uint64 `json:"reward_micro"`
	Target      string `json:"target"` // e.g., ClubID, WalletAddress, CardID
	Type        string `json:"type"`   // e.g., "Sabotage", "Kidnap", "Espionage"
}

// HandleGetUnderworldContracts returns a list of available contracts for criminal players.
// PILLAR 3: Underworld Contracts.
func (bs *BlackMarketService) HandleGetUnderworldContracts(l *Lobby, w http.ResponseWriter, r *http.Request) {
	// In a real scenario, this would dynamically generate contracts based on game state,
	// player's Wanted Level, available targets, etc. For now, we provide static examples.
	contracts := []UnderworldContract{
		{
			ID:          "CONTRACT-001",
			Title:       "Sabotage East Gate Foundry",
			Description: "Disrupt the Elemental Forge in East Gate to destabilize their production. Requires a Sabotage item.",
			RewardMicro: 1500 * 1000000, // 1,500 VBV
			Target:      "east_gate",
			Type:        "Sabotage",
		},
		{
			ID:          "CONTRACT-002",
			Title:       "Acquire Data Haven Intel",
			Description: "Perform a Cyber-Audit on the Data Haven club. Report their treasury status.",
			RewardMicro: 750 * 1000000, // 750 VBV
			Target:      "data_haven",
			Type:        "Espionage",
		},
		{
			ID:          "CONTRACT-003",
			Title:       "Incarcerate Assets",
			Description: "Successfully jail a rival's card during combat on club territory. High infamy, high reward.",
			RewardMicro: 1000 * 1000000, // 1,000 VBV
			Target:      "jail_capture",
			Type:        "Combat",
		},
		{
			ID:          "CONTRACT-004",
			Title:       "Disrupt Regional Governor",
			Description: "Successfully sabotage a district controlled by a Regional Governor. High risk, high reward.",
			RewardMicro: 2000 * 1000000, // 2,000 VBV
			Target:      "regional_governor_district",
			Type:        "Sabotage",
		},
		{
			ID:          "CONTRACT-005",
			Title:       "High-Value Ransom",
			Description: "Successfully execute a Kidnap Gambit against a rival club owner. Extract VBV through hostage leverage.",
			RewardMicro: 3000 * 1000000, // 3,000 VBV
			Target:      "kidnap_event",
			Type:        "Kidnap",
		},
		{
			ID:          "CONTRACT-006",
			Title:       "Defame Regional Leadership",
			Description: "Successfully spread a Negative Rumor about a Regional Governor. Influence the markets to destabilize their power.",
			RewardMicro: 1500 * 1000000, // 1,500 VBV
			Target:      "governor_rumor",
			Type:        "Rumor",
		},
		{
			ID:          "CONTRACT-007",
			Title:       "Neutralize Arena Center",
			Description: "Successfully sabotage the organization controlling the Arena Center. Destabilize the sector's administrative core.",
			RewardMicro: 2500 * 1000000, // 2,500 VBV
			Target:      "arena_center",
			Type:        "Sabotage",
		},
		{
			ID:          "CONTRACT-008",
			Title:       "Neutralize Advanced Defenses",
			Description: "Successfully sabotage a club that has active 'Cyber-Counter' or 'Cyber-Lock' defenses. Cripple their security grid.",
			RewardMicro: 3500 * 1000000, // 3,500 VBV
			Target:      "advanced_defenses",
			Type:        "Sabotage",
		},
		{
			ID:          "CONTRACT-009",
			Title:       "Infiltrate Regional Leadership",
			Description: "Execute a successful Heist against a club controlled by a Regional Governor (owning 2+ districts). Infiltrate the elite.",
			RewardMicro: 4000 * 1000000, // 4,000 VBV
			Target:      "regional_governor_heist",
			Type:        "Heist",
		},
		{
			ID:          "CONTRACT-010",
			Title:       "Governor's Favorite Hostage",
			Description: "Successfully execute a Kidnap Gambit against a Regional Governor's favorite card. Demand the ultimate ransom.",
			RewardMicro: 5000 * 1000000, // 5,000 VBV
			Target:      "governor_favorite_kidnap",
			Type:        "Kidnap",
		},
		{
			ID:          "CONTRACT-011",
			Title:       "Cripple Megalopolis Infrastructure",
			Description: "Successfully sabotage a club that controls 3 or more territories. Break the spine of a local titan.",
			RewardMicro: 6000 * 1000000, // 6,000 VBV
			Target:      "titan_club_sabotage",
			Type:        "Sabotage",
		},
		{
			ID:          "CONTRACT-012",
			Title:       "Neutralize Mojo Field",
			Description: "Successfully execute a Heist against a club with an active 'MOJO_STABILIZER'. Disrupt their social preservation field.",
			RewardMicro: 7500 * 1000000, // 7,500 VBV
			Target:      "mojo_stabilizer_heist",
			Type:        "Heist",
		},
		{
			ID:          "CONTRACT-013",
			Title:       "Arena Center Alliance Breach",
			Description: "Execute a successful Heist against the organization controlling the 'Arena Center' while they are in an Alliance. Strike at the heart of coordinated power.",
			RewardMicro: 10000 * 1000000, // 10,000 VBV
			Target:      "arena_center_alliance_heist",
			Type:        "Heist",
		},
		{
			ID:          "CONTRACT-014",
			Title:       "Cripple Sovereign Infrastructure",
			Description: "Successfully sabotage a Regional Governor's district while holding a Wanted Level of 20+. The ultimate defiance.",
			RewardMicro: 8000 * 1000000, // 8,000 VBV
			Target:      "sovereign_sabotage",
			Type:        "Sabotage",
		},
		{
			ID:          "CONTRACT-015",
			Title:       "Hegemony Breach",
			Description: "Execute a successful Heist against the organization controlling the 'Arena Center' while holding a Wanted Level of 25+. Challenge the core of the Arena.",
			RewardMicro: 12000 * 1000000, // 12,000 VBV
			Target:      "arena_center_heist_wanted",
			Type:        "Heist",
		},
		{
			ID:          "CONTRACT-016",
			Title:       "Arena Center Hegemony Breach",
			Description: "Successfully sabotage the organization controlling the 'Arena Center' while holding a Wanted Level of 30+. Challenge the core of the Arena.",
			RewardMicro: 15000 * 1000000, // 15,000 VBV
			Target:      "arena_center_sabotage_wanted",
			Type:        "Sabotage",
		},
		{
			ID:          "CONTRACT-017",
			Title:       "Hostage Liberation Strike",
			Description: "Successfully execute a Heist against a Regional Governor who is currently holding 3 or more hostages. Liberate the sector's assets.",
			RewardMicro: 20000 * 1000000, // 20,000 VBV
			Target:      "governor_hostage_liberation",
			Type:        "Heist",
		},
		{
			ID:          "CONTRACT-018",
			Title:       "Arena Center Fortress Breach",
			Description: "Successfully sabotage the organization controlling the 'Arena Center' while they are bolstered by 3+ active allied hardware traps. Crush their ultimate fortification.",
			RewardMicro: 25000 * 1000000, // 25,000 VBV
			Target:      "arena_center_fortress_sabotage",
			Type:        "Sabotage",
		},
		{
			ID:          "CONTRACT-019",
			Title:       "The Invisible Hand of Chaos",
			Description: "Successfully execute a Heist against the organization controlling the 'Arena Center' while under 'Ghost Protocol' and they have 3+ active allied hardware traps.",
			RewardMicro: 30000 * 1000000, // 30,000 VBV
			Target:      "arena_center_ghost_fortress_heist",
			Type:        "Heist",
		},
		{
			ID:          "CONTRACT-020",
			Title:       "The Apex Syndicate Fleecing",
			Description: "Execute a successful Heist against an organization holding 5+ hostages simultaneously while maintaining a Wanted Level of 40+. Topple the giants.",
			RewardMicro: 20000 * 1000000, // 20,000 VBV (Adjusted for Faucet Resilience)
			Target:      "apex_hostage_org_heist",
			Type:        "Heist",
		},
		{
			ID:          "CONTRACT-021",
			Title:       "The Hostage Hegemony Collapse",
			Description: "Successfully sabotage an organization holding 7+ hostages simultaneously while maintaining a Wanted Level of 50+. Topple the peak syndicates.",
			RewardMicro: 25000 * 1000000, // 25,000 VBV (Adjusted for Faucet Resilience)
			Target:      "apex_hostage_syndicate_sabotage",
			Type:        "Sabotage",
		},
		{
			ID:          "CONTRACT-022",
			Title:       "The Sovereign Hostage Heist",
			Description: "Execute a successful Heist against an organization holding 10+ hostages simultaneously while maintaining a Wanted Level of 60+. Take down the ultimate syndicate.",
			RewardMicro: 35000 * 1000000, // 35,000 VBV (Adjusted for Faucet Resilience)
			Target:      "sovereign_hostage_org_heist",
			Type:        "Heist",
		},
		{
			ID:          "CONTRACT-023",
			Title:       "The Arena Center Apex Heist",
			Description: "Execute a successful Heist against the 'Arena Center' organization while maintaining a Wanted Level of 100+. Claim the ultimate prize.",
			RewardMicro: 50000 * 1000000, // 50,000 VBV (New Absolute Peak for resilience)
			Target:      "arena_center_apex_heist",
			Type:        "Heist",
		},
		{
			ID:          "CONTRACT-024",
			Title:       "Black Market Monopoly",
			Description: "Sell an 'Epic' or 'Legendary' card to the Black Market. Requires the 'Fence' career role.",
			RewardMicro: 30000 * 1000000, // 30,000 VBV
			Target:      "high_tier_fence",
			Type:        "Fencing",
		},
		{
			ID:          "CONTRACT-025",
			Title:       "The Abductor's Disruptive Strike",
			Description: "Successfully sabotage an organization while holding the 'Kidnapper' career role. Disrupt their sector influence.",
			RewardMicro: 15000 * 1000000, // 15,000 VBV
			Target:      "any_org_sabotage",
			Type:        "Sabotage",
		},
		{
			ID:          "CONTRACT-026",
			Title:       "The Reparation Sabotage",
			Description: "Successfully sabotage an organization whose owner has received 5+ reparations. Requires the 'Saboteur' career role.",
			RewardMicro: 18000 * 1000000, // 18,000 VBV
			Target:      "high_reparation_org",
			Type:        "Sabotage",
		},
		{
			ID:          "CONTRACT-027",
			Title:       "The Smuggler's Turf Breach",
			Description: "Successfully heist an organization while the 'Smuggler' career role is active. Requires Wanted Level 30+. Extract capital under heat.",
			RewardMicro: 20000 * 1000000, // 20,000 VBV
			Target:      "high_infamy_smuggle_heist",
			Type:        "Heist",
		},
		{
			ID:          "CONTRACT-028",
			Title:       "Elite Liberation",
			Description: "Successfully liberate a 'Fallen' card through the 3-win challenge. Requires the 'Launderer' career role.",
			RewardMicro: 15000 * 1000000, // 15,000 VBV
			Target:      "liberation_success",
			Type:        "Recovery",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contracts)
}

/**
 * HandleGetBlackMarket returns liquidated assets with dynamic Dutch Auction pricing.
 * PILLAR 7: Underworld Recovery.
 */
func (bs *BlackMarketService) HandleGetBlackMarket(l *Lobby, w http.ResponseWriter, r *http.Request) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	wallet := strings.ToLower(r.URL.Query().Get("wallet"))
	stats, exists := l.leaderboard[wallet]
	if !exists {
		http.Error(w, "Identification failed", http.StatusUnauthorized)
		return
	}

	// Access Control: Wanted Level 5+ or Cunning 10+
	if stats.WantedLevel < 5 && l.playerService.GetEffectiveCunning(stats) < 10 {
		http.Error(w, "Access Denied: High-infamy profile required", http.StatusForbidden)
		return
	}

	type blackMarketListing struct {
		Loan
		CurrentPriceMicro uint64 `json:"current_price_micro"`
	}

	var listings []blackMarketListing
	now := time.Now()

	for _, item := range l.blackMarket {
		// PILLAR 7: Dutch Auction Pricing.
		// Price starts at 75% of original repayment and decays 5% every hour.
		hoursInMarket := now.Sub(item.DueAt).Hours()
		if hoursInMarket < 0 {
			hoursInMarket = 0
		}

		basePriceMicro := (item.RepaymentAmount * 75) / 100
		decayPercent := uint64(hoursInMarket) * 5
		if decayPercent > 70 {
			decayPercent = 70
		} // Cap at 70% decay (approx 5% floor)

		currentPriceMicro := basePriceMicro - ((basePriceMicro * decayPercent) / 100)

		listings = append(listings, blackMarketListing{
			Loan:              item,
			CurrentPriceMicro: currentPriceMicro,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(listings)
}

/**
 * HandleBuyBlackMarket processes the acquisition of hot assets.
 * PILLAR 7: Underworld Recovery.
 */
func (bs *BlackMarketService) HandleBuyBlackMarket(l *Lobby, w http.ResponseWriter, r *http.Request) {
	var req struct {
		Wallet string `json:"wallet"`
		LoanID string `json:"loan_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	wallet := strings.ToLower(req.Wallet)
	foundIdx := -1
	for i, item := range l.blackMarket {
		if item.ID == req.LoanID {
			foundIdx = i
			break
		}
	}

	if foundIdx == -1 {
		http.Error(w, "Asset seized by another scavenger", http.StatusNotFound)
		return
	}

	item := l.blackMarket[foundIdx]
	hoursInMarket := time.Since(item.DueAt).Hours()
	if hoursInMarket < 0 {
		hoursInMarket = 0
	}
	basePriceMicro := (item.RepaymentAmount * 75) / 100
	decayPercent := uint64(hoursInMarket) * 5
	if decayPercent > 70 {
		decayPercent = 70
	}
	currentPriceMicro := basePriceMicro - ((basePriceMicro * decayPercent) / 100)

	if l.playerBalances[wallet] < currentPriceMicro {
		http.Error(w, "Insufficient rewards for seizure", http.StatusPaymentRequired)
		return
	}

	l.playerBalances[wallet] -= currentPriceMicro
	l.faucetBalanceMicro += currentPriceMicro
	l.blackMarket = append(l.blackMarket[:foundIdx], l.blackMarket[foundIdx+1:]...)

	stats := l.leaderboard[wallet]
	stats.Inventory[fmt.Sprintf("CARD-%d", item.CollateralBundle.CardID)]++
	stats.WantedLevel += 5 // Buying hot goods increases infamy
	stats.Reputation = l.CalculateReputation(stats)
	l.leaderboard[wallet] = stats

	l.logAdminAuditLocked("BLACK_MARKET_PURCHASE", wallet, fmt.Sprintf("Card: %d, Price: %d", item.CollateralBundle.CardID, currentPriceMicro))

	// PILLAR 13: Career Progression — Black Market Dealer XP trigger (#8A).
	// +50 XP toward Black Market Dealer career progression per successful acquisition.
	l.TrackCareerXP(wallet, "Black Market Dealer", 50)

	l.applyDynamicScalingLocked()
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// HandleAcceptUnderworldContract processes a player's request to begin an illicit mission.
// PILLAR 3: Criminality & Intelligence.
func (bs *BlackMarketService) HandleAcceptUnderworldContract(l *Lobby, env *Envelope) {
	var data struct {
		ContractID string `json:"contract_id"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		log.Printf("[UNDERWORLD] Invalid accept_underworld_contract payload from %s: %v\n", env.FromID, err)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	wallet, ok := l.wallets[env.FromID]
	if !ok {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Access Denied: Identification failed."}`)})
		return
	}

	l.ensurePlayerStatsMapsInitialized(wallet)
	stats := l.leaderboard[wallet]

	// PILLAR 3: Career Role Gating
	if data.ContractID == "CONTRACT-024" && stats.JobRole != "Fence" {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Access Denied: This contract requires the 'Fence' career path."}`)})
		return
	}
	if data.ContractID == "CONTRACT-025" && stats.JobRole != "Kidnapper" {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Access Denied: This contract requires the 'Kidnapper' career path."}`)})
		return
	}
	if data.ContractID == "CONTRACT-026" && stats.JobRole != "Saboteur" {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Access Denied: This contract requires the 'Saboteur' career path."}`)})
		return
	}
	if data.ContractID == "CONTRACT-027" && stats.JobRole != "Smuggler" {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Access Denied: This contract requires the 'Smuggler' career path."}`)})
		return
	}
	if data.ContractID == "CONTRACT-028" && stats.JobRole != "Launderer" {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Access Denied: This contract requires the 'Launderer' career path."}`)})
		return
	}

	// 1. Mission Capacity Check
	if stats.ActiveUnderworldContractID != "" {
		l.sendToClientLocked(env.FromID, Envelope{
			Type:    "admin_notification",
			Payload: json.RawMessage(`{"text":"⚠️ <b>CONTRACT ACTIVE:</b> You are already assigned to a high-infamy operation. Complete or abort it first."}`),
		})
		return
	}

	// 2. Contract Validation (Sync with HandleGetUnderworldContracts)
	// In production, this would query a dynamic registry or database.
	valid := false
	var title string
	switch data.ContractID {
	case "CONTRACT-001":
		title = "Sabotage East Gate Foundry"
		valid = true
	case "CONTRACT-002":
		title = "Acquire Data Haven Intel"
		valid = true
	case "CONTRACT-003":
		title = "Incarcerate Assets"
		valid = true
	case "CONTRACT-004":
		title = "Disrupt Regional Governor"
		valid = true
	case "CONTRACT-005":
		title = "High-Value Ransom"
		valid = true
	case "CONTRACT-006":
		title = "Defame Regional Leadership"
		valid = true
	case "CONTRACT-007":
		title = "Neutralize Arena Center"
		valid = true
	case "CONTRACT-008":
		title = "Neutralize Advanced Defenses"
		valid = true
	case "CONTRACT-009":
		title = "Infiltrate Regional Leadership"
		valid = true
	case "CONTRACT-010":
		title = "Governor's Favorite Hostage"
		valid = true
	case "CONTRACT-011":
		title = "Cripple Megalopolis Infrastructure"
		valid = true
	case "CONTRACT-012":
		title = "Neutralize Mojo Field"
		valid = true
	case "CONTRACT-013":
		title = "Arena Center Alliance Breach"
		valid = true
	case "CONTRACT-016":
		title = "Arena Center Hegemony Breach"
		valid = true
	case "CONTRACT-014":
		title = "Cripple Sovereign Infrastructure"
		valid = true
	case "CONTRACT-015":
		title = "Hegemony Breach"
		valid = true
	case "CONTRACT-017":
		title = "Hostage Liberation Strike"
		valid = true
	case "CONTRACT-018":
		title = "Arena Center Fortress Breach"
		valid = true
	case "CONTRACT-019":
		title = "The Invisible Hand of Chaos"
		valid = true
	case "CONTRACT-020":
		title = "The Apex Syndicate Fleecing"
		valid = true
	case "CONTRACT-021":
		title = "The Hostage Hegemony Collapse"
		valid = true
	case "CONTRACT-022":
		title = "The Sovereign Hostage Heist"
		valid = true
	case "CONTRACT-023":
		title = "The Arena Center Apex Heist"
		valid = true
	case "CONTRACT-024":
		title = "Black Market Monopoly"
		valid = true
	case "CONTRACT-025":
		title = "The Abductor's Disruptive Strike"
		valid = true
	case "CONTRACT-026":
		title = "The Reparation Sabotage"
		valid = true
	case "CONTRACT-027":
		title = "The Smuggler's Turf Breach"
		valid = true
	case "CONTRACT-028":
		title = "Elite Liberation"
		valid = true
	}

	if !valid {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Error: Specified contract has been terminated or moved."}`)})
		return
	}

	// 3. Mission Assignment
	stats.ActiveUnderworldContractID = data.ContractID

	// Underworld mechanics: Accepting a contract immediately increases visibility
	stats.WantedLevel += 1
	stats.Reputation = l.CalculateReputation(stats)
	l.leaderboard[wallet] = stats

	l.logAdminAuditLocked("CONTRACT_ACCEPTED", wallet, fmt.Sprintf("ID: %s, Title: %s", data.ContractID, title))

	// 4. Immersive Feedback
	msg := fmt.Sprintf(`{"text":"💀 <b>CONTRACT SIGNED:</b> %s is now active. Expect high-infamy consequences."}`, escapeHTML(title))
	l.sendToClientLocked(env.FromID, Envelope{
		Type:    "admin_notification",
		Payload: json.RawMessage(msg),
	})

	// Trigger Global Sync to update profile icons and status
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// HandleAbortUnderworldContract terminates the active criminal mission with a penalty.
// PILLAR 3: Underworld Contracts.
func (bs *BlackMarketService) HandleAbortUnderworldContract(l *Lobby, env *Envelope) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	wallet, ok := l.wallets[env.FromID]
	if !ok {
		return
	}

	l.ensurePlayerStatsMapsInitialized(wallet)
	stats := l.leaderboard[wallet]

	if stats.ActiveUnderworldContractID == "" {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Error: No active contract to abort."}`)})
		return
	}

	contractID := stats.ActiveUnderworldContractID
	stats.ActiveUnderworldContractID = ""

	// PILLAR 3: Social Consequences.
	stats.Reputation -= 50
	if stats.Reputation < 0 {
		stats.Reputation = 0
	}

	l.leaderboard[wallet] = stats
	l.logAdminAuditLocked("CONTRACT_ABORTED", wallet, fmt.Sprintf("ID: %s", contractID))
	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"💀 <b>CONTRACT TERMINATED:</b> Operational data purged. Reputation penalized."}`)})
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// ============================================================================
// P2-B3: FENCED GOODS MARKETPLACE — Types & Handlers
// ============================================================================

// FenceListing represents a player-posted item on the fenced goods marketplace.
type FenceListing struct {
	ID            string    `json:"id"`
	SellerWallet  string    `json:"seller_wallet"`
	CardID        int       `json:"card_id"`
	CardName      string    `json:"card_name"`
	PriceMicro    uint64    `json:"price_micro"`
	ListedAt      time.Time `json:"listed_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	CommissionPct int       `json:"commission_pct"` // 8% standard, 5% at Fence Tier 3+
}

// HandleListFenceGoods allows a player to list items for sale on the fenced marketplace.
// Task 4104-3A: POST /api/black-market/fence-goods endpoint
func (bs *BlackMarketService) HandleListFenceGoods(l *Lobby, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Wallet       string `json:"wallet"`
		CardID       int    `json:"card_id"`
		PriceMicro   uint64 `json:"price_micro"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	wallet := strings.ToLower(strings.TrimSpace(req.Wallet))
	if wallet == "" {
		http.Error(w, "wallet is required", http.StatusUnauthorized)
		return
	}
	if req.CardID <= 0 {
		http.Error(w, "card_id is required", http.StatusBadRequest)
		return
	}
	if req.PriceMicro < 1000000 { // Minimum 1 VBV
		http.Error(w, "price_micro must be at least 1 VBV", http.StatusBadRequest)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	stats, exists := l.leaderboard[wallet]
	if !exists {
		http.Error(w, "player not found", http.StatusNotFound)
		return
	}

	cardKey := fmt.Sprintf("CARD-%d", req.CardID)
	qty, hasCard := stats.Inventory[cardKey]
	if !hasCard || qty <= 0 {
		http.Error(w, "card not in inventory", http.StatusBadRequest)
		return
	}

	// Determine commission: 8% standard, 5% at Fence Tier 3+
	commissionPct := 8
	if stats.JobRole == "Fence" && stats.CareerXP != nil {
		tier := stats.CareerXP.GetFenceTier()
		if tier >= 3 {
			commissionPct = 5
		}
	}

	now := time.Now()
	listingID := fmt.Sprintf("FENCE-%s-%d", wallet, now.UnixNano())

	listing := FenceListing{
		ID:            listingID,
		SellerWallet:  wallet,
		CardID:        req.CardID,
		CardName:      cardKey,
		PriceMicro:    req.PriceMicro,
		ListedAt:      now,
		ExpiresAt:     now.Add(24 * time.Hour), // Task 4104-3D: 24h expiry
		CommissionPct: commissionPct,
	}

	if l.fencedListings == nil {
		l.fencedListings = make(map[string]FenceListing)
	}
	l.fencedListings[listingID] = listing

	stats.Inventory[cardKey]--
	if stats.Inventory[cardKey] <= 0 {
		delete(stats.Inventory, cardKey)
	}
	l.leaderboard[wallet] = stats

	log.Printf("[FENCED_MARKET] Listing created: %s (%s) for %.2f $VBV (commission: %d%%)", listingID, cardKey, float64(req.PriceMicro)/1000000.0, commissionPct)
	l.logAdminAuditLocked("FENCE_LISTING_CREATED", wallet, fmt.Sprintf("Card: %d, Price: %d micro-VBV, Commission: %d%%", req.CardID, req.PriceMicro, commissionPct))

	l.applyDynamicScalingLocked()
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"text":       fmt.Sprintf("Listed %s for %.2f $VBV (8%% commission)", cardKey, float64(req.PriceMicro)/1000000.0),
		"listing_id": listingID,
	})
}

// HandleGetFencedGoods returns all active listings on the fenced marketplace.
func (bs *BlackMarketService) HandleGetFencedGoods(l *Lobby, w http.ResponseWriter, r *http.Request) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	if l.fencedListings == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]FenceListing{})
		return
	}

	var active []FenceListing
	now := time.Now()
	for _, listing := range l.fencedListings {
		if listing.ExpiresAt.After(now) && listing.CardID > 0 {
			active = append(active, listing)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(active)
}

// HandleBuyFencedGood allows a player to purchase a fenced good with escrow.
// Task 4104-3B: POST /api/black-market/buy-stolen endpoint
func (bs *BlackMarketService) HandleBuyFencedGood(l *Lobby, w http.ResponseWriter, r *http.Request) {
	var req struct {
		Wallet       string `json:"wallet"`
		ListingID    string `json:"listing_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	wallet := strings.ToLower(strings.TrimSpace(req.Wallet))
	listingID := strings.TrimSpace(req.ListingID)

	if wallet == "" {
		http.Error(w, "wallet is required", http.StatusUnauthorized)
		return
	}
	if listingID == "" {
		http.Error(w, "listing_id is required", http.StatusBadRequest)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	stats, exists := l.leaderboard[wallet]
	if !exists {
		http.Error(w, "buyer not found", http.StatusNotFound)
		return
	}

	listing, hasListing := l.fencedListings[listingID]
	if !hasListing {
		http.Error(w, "listing not found", http.StatusNotFound)
		return
	}

	// Check expiry: Task 4104-3D
	if listing.ExpiresAt.Before(time.Now()) {
		delete(l.fencedListings, listingID)
		http.Error(w, "listing has expired", http.StatusGone)
		return
	}

	// Verify card still in seller's inventory (anti-race-condition)
	sellerStats, sellerExists := l.leaderboard[listing.SellerWallet]
	if !sellerExists {
		delete(l.fencedListings, listingID)
		http.Error(w, "seller no longer exists", http.StatusNotFound)
		return
	}

	cardKey := fmt.Sprintf("CARD-%d", listing.CardID)
	sellerQty, hasCard := sellerStats.Inventory[cardKey]
	if !hasCard || hasCard == 0 {
		delete(l.fencedListings, listingID)
		http.Error(w, "item no longer listed", http.StatusGone)
		return
	}

	// Escrow: buyer pays + commission
	commissionPct := listing.CommissionPct
	if commissionPct == 0 {
		commissionPct = 8 // default fallback
	}

	totalCost := listing.PriceMicro + (listing.PriceMicro * uint64(commissionPct) / 100)

	if l.playerBalances[wallet] < totalCost {
		http.Error(w, fmt.Sprintf("insufficient funds: need %.2f $VBV", float64(totalCost)/1000000.0), http.StatusPaymentRequired)
		return
	}

	// Process payment
	l.playerBalances[wallet] -= totalCost

	// Seller receives the sale price (commission is kept by system/faucet)
	sellerPayout := listing.PriceMicro
	if sellerStats.CareerXP != nil {
		// Fence sellers get 5% commission rebate at Tier 3+
		if stats.JobRole == "Fence" {
			tier := stats.CareerXP.GetFenceTier()
			if tier >= 3 {
				_ = tier // already counted in listing.CommissionPct
			}
		}
	}

	l.playerBalances[listing.SellerWallet] += sellerPayout

	// Transfer card to buyer
	sellerStats.Inventory[cardKey]--
	if sellerStats.Inventory[cardKey] <= 0 {
		delete(sellerStats.Inventory, cardKey)
	}
	l.leaderboard[listing.SellerWallet] = sellerStats

	stats.Inventory[cardKey]++
	l.leaderboard[wallet] = stats

	// Remove listing
	delete(l.fencedListings, listingID)

	log.Printf("[FENCED_MARKET] Sale completed: %s from %s to %s for %.2f $VBV (commission: %d%%)", listing.CardName, listing.SellerWallet[:8]+"...", wallet[:8]+"...", float64(sellerPayout)/1000000.0, commissionPct)
	l.logAdminAuditLocked("FENCE_SALE_COMPLETED", wallet, fmt.Sprintf("Card: %s, Seller: %s, Price: %d micro-VBV, Commission: %d%%", cardKey, listing.SellerWallet[:8]+"...", sellerPayout, commissionPct))

	// Route commission to faucet
	if l.tokenSinkRouter != nil {
		commissionAmount := totalCost - sellerPayout
		if commissionAmount > 0 {
			matrix := RevenueSplitMatrix{FaucetShare: 1.0, ClubShare: 0.0, GovernanceShare: 0.0}
			_ = l.tokenSinkRouter.RouteCriminalTax("FENCED_GOODS_COMMISSION", commissionAmount, matrix, 0, "")
		}
	}

	l.applyDynamicScalingLocked()
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"text":      fmt.Sprintf("Acquired %s for %.2f $VBV", cardKey, float64(totalCost)/1000000.0),
		"card_id":   listing.CardID,
		"seller":    listing.SellerWallet,
	})
}

// cleanupExpiredFencedListings removes listings older than 24 hours.
// Task 4104-3D: 24h listing expiry with cleanup
func (bs *BlackMarketService) cleanupExpiredFencedListings(l *Lobby) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.fencedListings == nil {
		return
	}

	now := time.Now()
	expired := 0
	for id, listing := range l.fencedListings {
		if listing.ExpiresAt.Before(now) && listing.CardID > 0 {
			// Return card to seller
			sellerStats, ok := l.leaderboard[listing.SellerWallet]
			if ok {
				cardKey := fmt.Sprintf("CARD-%d", listing.CardID)
				sellerStats.Inventory[cardKey]++
				l.leaderboard[listing.SellerWallet] = sellerStats
			}
			delete(l.fencedListings, id)
			expired++
		}
	}

	if expired > 0 {
		log.Printf("[FENCED_MARKET] Cleaned up %d expired listings", expired)
	}
}

// HandleGetSellerListings returns all active listings by a specific seller.
func (bs *BlackMarketService) HandleGetSellerListings(l *Lobby, w http.ResponseWriter, r *http.Request) {
	wallet := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("wallet")))
	if wallet == "" {
		http.Error(w, "wallet query parameter required", http.StatusBadRequest)
		return
	}

	l.mutex.RLock()
	defer l.mutex.RUnlock()

	if l.fencedListings == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]FenceListing{})
		return
	}

	var listings []FenceListing
	now := time.Now()
	for _, listing := range l.fencedListings {
		if listing.SellerWallet == wallet && listing.ExpiresAt.After(now) && listing.CardID > 0 {
			listings = append(listings, listing)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(listings)
}
