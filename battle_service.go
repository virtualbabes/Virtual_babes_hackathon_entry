package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"
)

// getEffectiveServerPower replicates the modifier logic for server validation
func getEffectiveServerPower(l *Lobby, match *MatchState, c *ServerCard, sideIdx int, gridIdx int) int {
	base := c.Power[sideIdx]

	// Apply Artifact/Equipment bonuses if active
	if match.Rules["Artifact_bonus"] {
		base += c.Artifact
	}

	// Wanted Level Penalty: -5 power per Wanted Level point
	playerID := match.P1ID
	if c.Owner == 1 {
		playerID = match.P2ID
	}
	wallet := l.wallets[playerID]
	stats := l.leaderboard[wallet]

	// Apply Wanted Level Penalty (Mitigated by Cunning)
	wantedPenalty := (stats.WantedLevel * 5)
	// Cunning mitigates penalty: every 1 point of Cunning reduces penalty by 2
	mitigation := stats.Cunning * 2
	if mitigation > wantedPenalty { mitigation = wantedPenalty }
	base -= (wantedPenalty - mitigation)

	// Fatigue Penalty: -1 power per point above 50
	if c.Fatigue > 50 {
		fatiguePenalty := (c.Fatigue - 50)
		// Nurturing reduces fatigue impact: 1 power back per Nurturing point
		reduction := stats.Nurturing
		if reduction > fatiguePenalty { reduction = fatiguePenalty }
		base -= (fatiguePenalty - reduction)
	}

	// Loyalty Bonus: +25 flat power for cards with max loyalty
	if c.Loyalty >= 100 {
		base += 25
	}

	if match.Rules["Elemental_sync"] {
		tileMood := match.BoardMoods[gridIdx]
		if tileMood != "" && tileMood != "Neutral" && c.Mood != "" && c.Mood != "Neutral" {
			moodWeaknesses := map[string]string{
				"Volatile": "Serene",
				"Serene":   "Spirited",
				"Spirited": "Grounded",
				"Grounded": "Volatile",
			}

			// Check for "grounded_shield" buff for the card's owner
			var hasGroundedShield bool
			if match.ActiveItemBuffs != nil && match.ActiveItemBuffs[playerID] != nil {
				if _, ok := match.ActiveItemBuffs[playerID]["grounded_shield"]; ok {
					hasGroundedShield = true
				}
			}

			if c.Mood == tileMood {
				base += 50 // Match bonus: +0.5 Tier
			} else if moodWeaknesses[c.Mood] == tileMood {
				if !hasGroundedShield { // Only apply penalty if no Grounded Shield
					base -= 50 // Weakness penalty: -0.5 Tier
				}
			}
		}
	}
	return base
}

func (l *Lobby) serverCheckCaptures(match *MatchState, gridIndex int, pIdx int) (int, []CapturedCardInfo) {
	totalFlips := 0
	placedCard := match.Board[gridIndex]
	neighbors := []struct {
		offset           int
		placedPowerIdx   int
		neighborPowerIdx int
		boundaryCheck    func(int) bool
	}{
		{-3, 0, 2, func(idx int) bool { return idx >= 3 }},
		{+1, 1, 3, func(idx int) bool { return idx%3 != 2 }},
		{+3, 2, 0, func(idx int) bool { return idx <= 5 }},
		{-1, 3, 1, func(idx int) bool { return idx%3 != 0 }},
	}

	sameGroups := make(map[int][]int)
	plusGroups := make(map[int][]int)
	var capturedCards []CapturedCardInfo // List to store details of all cards flipped
	var comboQueue []int

	// Determine playerID for buff check (owner of the placed card)
	playerID := match.P1ID
	if pIdx == 1 {
		playerID = match.P2ID
	}

	// Check for "rule_breaker" buff (Force_Plus_Trigger)
	forcePlusTrigger := false
	if match.ActiveItemBuffs != nil && match.ActiveItemBuffs[playerID] != nil {
		if _, ok := match.ActiveItemBuffs[playerID]["rule_breaker"]; ok {
			forcePlusTrigger = true
		}
	}

	// 1. Evaluate Neighbors
	for _, n := range neighbors {
		nbIdx := gridIndex + n.offset
		if n.boundaryCheck(gridIndex) && match.Board[nbIdx] != nil {
			neighbor := match.Board[nbIdx]
			pPower := getEffectiveServerPower(l, match, placedCard, n.placedPowerIdx, gridIndex)
			nPower := getEffectiveServerPower(l, match, neighbor, n.neighborPowerIdx, nbIdx)

			// Prepare Rule Groups
			if match.Rules["Power_copy"] && pPower == nPower {
				sameGroups[pPower] = append(sameGroups[pPower], nbIdx)
			}
			// Apply "Plus" rule if active OR if "Force_Plus_Trigger" buff is active
			if match.Rules["Power_up"] || forcePlusTrigger {
				plusGroups[pPower+nPower] = append(plusGroups[pPower+nPower], nbIdx)
			}

			// Basic Capture (Direct Comparison)
			if neighbor.Owner != pIdx && pPower > nPower {
				originalOwnerWallet := l.wallets[match.P1ID] // Default to P1's wallet
				if neighbor.Owner == 1 {
					originalOwnerWallet = l.wallets[match.P2ID]
				} // If neighbor was P2's card
				neighbor.Owner = pIdx
				totalFlips++
				capturedCards = append(capturedCards, CapturedCardInfo{
					CardID:                neighbor.ID,
					OriginalOwnerWallet:   originalOwnerWallet,
					CapturingPlayerWallet: l.wallets[playerID],
				})
			}
		}
	}

	// 2. Process Rule Triggers
	for _, indices := range sameGroups {
		if len(indices) >= 2 {
			for _, idx := range indices {
				if match.Board[idx].Owner != pIdx {
					originalOwnerWallet := l.wallets[match.P1ID]
					if match.Board[idx].Owner == 1 {
						originalOwnerWallet = l.wallets[match.P2ID]
					}
					match.Board[idx].Owner = pIdx
					totalFlips++
					capturedCards = append(capturedCards, CapturedCardInfo{
						CardID:                match.Board[idx].ID,
						OriginalOwnerWallet:   originalOwnerWallet,
						CapturingPlayerWallet: l.wallets[playerID],
					})
					comboQueue = append(comboQueue, idx)
				}
			}
		}
	}
	for _, indices := range plusGroups {
		if len(indices) >= 2 {
			for _, idx := range indices {
				if match.Board[idx].Owner != pIdx {
					originalOwnerWallet := l.wallets[match.P1ID]
					if match.Board[idx].Owner == 1 {
						originalOwnerWallet = l.wallets[match.P2ID]
					}
					match.Board[idx].Owner = pIdx
					totalFlips++
					capturedCards = append(capturedCards, CapturedCardInfo{
						CardID:                match.Board[idx].ID,
						OriginalOwnerWallet:   originalOwnerWallet,
						CapturingPlayerWallet: l.wallets[playerID],
					})
					comboQueue = append(comboQueue, idx)
				}
			}
		}
	}

	// 3. Chain Reaction (Recursive Basic Captures only)
	for len(comboQueue) > 0 {
		currIdx := comboQueue[0]
		comboQueue = comboQueue[1:]
		currCard := match.Board[currIdx]
		for _, n := range neighbors {
			nbIdx := currIdx + n.offset
			if n.boundaryCheck(currIdx) && match.Board[nbIdx] != nil {
				neighbor := match.Board[nbIdx]
				cPower := getEffectiveServerPower(l, match, currCard, n.placedPowerIdx, currIdx)
				nPower := getEffectiveServerPower(l, match, neighbor, n.neighborPowerIdx, nbIdx)

				if neighbor.Owner != pIdx && cPower > nPower {
					neighbor.Owner = pIdx
					// Only add to capturedCards if it wasn't already flipped by a direct capture or rule
					// This prevents double-counting for jailing
					alreadyCaptured := false
					for _, cc := range capturedCards {
						if cc.CardID == neighbor.ID {
							alreadyCaptured = true
							break
						}
					}
					if !alreadyCaptured {
						originalOwnerWallet := l.wallets[match.P1ID]
						if neighbor.Owner == 1 {
							originalOwnerWallet = l.wallets[match.P2ID]
						}
						capturedCards = append(capturedCards, CapturedCardInfo{CardID: neighbor.ID, OriginalOwnerWallet: originalOwnerWallet, CapturingPlayerWallet: l.wallets[playerID]})
					}
					totalFlips++
					comboQueue = append(comboQueue, nbIdx)
				}
			}
		}
	}
	return totalFlips, capturedCards
}

func (l *Lobby) verifyWinner(match *MatchState) {
	p1, p2 := 0, 0
	boardMap := make(map[int]bool)
	for _, c := range match.Board {
		if c == nil {
			continue
		}
		boardMap[c.ID] = true
		if c.Owner == 0 {
			p1++
		} else {
			p2++
		}
	}
	// Add remaining hand cards to the owner's score
	for _, id := range match.P1Deck {
		if !boardMap[id] {
			p1++
		}
	}
	for _, id := range match.P2Deck {
		if !boardMap[id] {
			p2++
		}
	}

	match.FinalScores = [2]int{p1, p2}

	// SUDDEN DEATH TRIGGER: If 5-5 Draw and rule is enabled (or it's a tournament match)
	if p1 == 5 && p2 == 5 && (match.Rules["Sudden_death"] || match.TournamentMatchID != "") {
		l.initiateSuddenDeath(match)
		return
	}

	match.IsFinished = true
	history := MatchHistory{
		Scores:           [2]int{p1, p2},
		Timestamp:        time.Now(),
		TournamentMatchID: match.TournamentMatchID,
		IsBountyMatch:    match.IsBountyMatch,
	}

	if p1 > p2 {
		history.WinnerID, history.WinnerIndex = match.P1ID, 0
		history.Opponent = l.wallets[match.P2ID]
		l.finalizeMatchResult(match.P1ID, match.P1Deck, history)

		// Achievement: Perfect Game (10-0)
		if p1 == 10 {
			l.unlockAchievement(l.wallets[match.P1ID], "PERFECT_GAME")
		}
	} else if p2 > p1 {
		history.WinnerID, history.WinnerIndex = match.P2ID, 1
		history.Opponent = l.wallets[match.P1ID]
		l.finalizeMatchResult(match.P2ID, match.P2Deck, history)
	} else { // Draw
		history.WinnerID, history.WinnerIndex = "", 2 // 2 for Draw
		history.Opponent = "DRAW"
		// No specific player wins, so no leaderboard update for a draw winner
	}

	// BOUNTY SYSTEM: Check for Hunter/Outlaw reward triggers
	if match.IsBountyMatch && history.WinnerID != "" {
		hunterID := ""
		targetWanted := 0
		if match.P1WantedLevel <= 2 && match.P2WantedLevel >= 10 {
			hunterID = match.P1ID
			targetWanted = match.P2WantedLevel
		} else if match.P2WantedLevel <= 2 && match.P1WantedLevel >= 10 {
			hunterID = match.P2ID
			targetWanted = match.P1WantedLevel
		}

		if history.WinnerID == hunterID {
			history.BountyReward = float64(targetWanted * 50)
			l.sendToClient(hunterID, Envelope{
				Type:    "admin_notification",
				Payload: json.RawMessage(fmt.Sprintf(`{"text":"🎯 <b>BOUNTY CLAIMED!</b> You've earned %.2f bonus $VBV."}`, history.BountyReward)),
			})
			l.unlockAchievement(l.wallets[hunterID], "OUTLAW_SLAYER")
		}
	}

	// Decrement and remove expired item buffs for both players
	l.processItemBuffExpiration(match)

	// PRISONER RULE: Decide which jailing logic to apply
	if match.Rules["Fallen_penalty"] && len(match.CapturedCards) > 0 {
		l.processFallenPenaltyJail(match, match.CapturedCards) // Jail all captured cards if Fallen_penalty is active
	} else if p1 != p2 { // Only apply original prisoner rule if there's a clear loser
		// Original Prisoner Rule: Jail loser's rarest card (if Fallen_penalty is not active or no cards were captured)
		if p1 > p2 { // P1 won, P2 lost, so P2 is the loser
			l.processPrisonerRule(match, l.wallets[match.P2ID], l.wallets[match.P1ID])
		} else if p2 > p1 { // P2 won, P1 lost
			l.processPrisonerRule(match, l.wallets[match.P1ID], l.wallets[match.P2ID])
		}
	}

	delete(l.matches, match.P1ID)
	delete(l.matches, match.P2ID)
}

// initiateSuddenDeath shuffles and redistributes remaining hand cards for a high-stakes tie-breaker.
func (l *Lobby) initiateSuddenDeath(match *MatchState) {
	var p1NewDeck []int
	var p2NewDeck []int

	boardMap := make(map[int]bool)
	for _, c := range match.Board {
		if c == nil {
			continue
		}
		boardMap[c.ID] = true
		if c.Owner == 0 {
			p1NewDeck = append(p1NewDeck, c.ID)
		} else {
			p2NewDeck = append(p2NewDeck, c.ID)
		}
	}

	for _, id := range match.P1Deck {
		if !boardMap[id] {
			p1NewDeck = append(p1NewDeck, id)
		}
	}
	for _, id := range match.P2Deck {
		if !boardMap[id] {
			p2NewDeck = append(p2NewDeck, id)
		}
	}

	rand.Shuffle(len(p1NewDeck), func(i, j int) { p1NewDeck[i], p1NewDeck[j] = p1NewDeck[j], p1NewDeck[i] })
	rand.Shuffle(len(p2NewDeck), func(i, j int) { p2NewDeck[i], p2NewDeck[j] = p2NewDeck[j], p2NewDeck[i] })

	match.Board = [9]*ServerCard{}
	match.P1Deck = p1NewDeck
	match.P2Deck = p2NewDeck
	match.FinalScores = [2]int{0, 0}

	payload, _ := json.Marshal(map[string]interface{}{
		"text":    "⚔️ <b>SUDDEN DEATH!</b> The board has cleared. Decks have been redistributed based on card ownership.",
		"p1_deck": p1NewDeck,
		"p2_deck": p2NewDeck,
	})

	l.sendToClient(match.P1ID, Envelope{Type: "sudden_death_start", FromID: "SERVER", Payload: payload})
	l.sendToClient(match.P2ID, Envelope{Type: "sudden_death_start", FromID: "SERVER", Payload: payload})
	log.Printf("[BATTLE] Sudden Death tie-breaker initiated for match %s vs %s\n", match.P1ID, match.P2ID)
}

func (l *Lobby) finalizeMatchResult(winnerID string, deck []int, history MatchHistory) {
	l.matchHistory[winnerID] = history
	if wallet, ok := l.wallets[winnerID]; ok {
		l.updateLeaderboard(wallet, history.TournamentMatchID != "", history.Scores, deck, history.IsBountyMatch) // Pass match context
		go func() {
			rating := l.calculateDeckRating(deck)
			l.mutex.Lock() // Lock for leaderboard modification
			s := l.leaderboard[wallet]

			// Achievement: Arena Legend (100 Wins)
			if s.Wins == 100 {
				l.unlockAchievement(wallet, "ARENA_LEGEND")
			}

			if l.isBetterRating(rating, s.BestRating) {
				s.BestRating = rating // Update best rating
				l.leaderboard[wallet] = s
			}
			l.mutex.Unlock()
		}()
		if history.TournamentMatchID != "" {
			l.processTournamentResult(history.TournamentMatchID, wallet)
		}
	}
}

func (l *Lobby) calculateDeckRating(cardIDs []int) string {
	if len(cardIDs) == 0 {
		return "[Z]"
	}
	maxBin := -1
	for _, id := range cardIDs {
		l.mutex.RLock()
		card := l.inventory[id]
		l.mutex.RUnlock()
		highest := 0
		for _, p := range card.Power {
			if p > highest {
				highest = p
			}
		}
		bin := (highest - 1) / 100
		if bin > maxBin {
			maxBin = bin
		}
	}
	if maxBin == -1 {
		return "[Z]"
	}
	alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	base := string(alphabet[25-maxBin])
	plus := ""
	for _, id := range cardIDs {
		l.mutex.RLock()
		card := l.inventory[id]
		l.mutex.RUnlock()
		highest := 0
		for _, p := range card.Power {
			if p > highest {
				highest = p
			}
		}
		if (highest-1)/100 == maxBin {
			plus += "+"
		}
	}
	return fmt.Sprintf("[%s%s]", base, plus)
}

func (l *Lobby) isBetterRating(newR, oldR string) bool {
	if oldR == "" || oldR == "[Z]" {
		return true
	}
	parse := func(r string) (rune, int) {
		if len(r) < 3 {
			return 'Z', 0
		}
		return rune(r[1]), strings.Count(r, "+")
	}
	nL, nP := parse(newR)
	oL, oP := parse(oldR)
	alpha := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	nI, oI := strings.IndexRune(alpha, nL), strings.IndexRune(alpha, oL)
	if nI < oI {
		return true
	}
	if nI > oI {
		return false
	}
	return nP >= oP
}

func (l *Lobby) updateLeaderboard(wallet string, isTournamentWin bool, scores [2]int, deck []int, isBountyWin bool) {
	stats := l.leaderboard[wallet]
	stats.Wins++
	stats.DisconnectStreak = 0
	l.updatePlayerPlaystyleTendencies(wallet, true, scores, deck, isBountyWin) // Update playstyle on win
	stats.Reputation = l.CalculateReputation(stats)                            // Ensure reputation is updated
	l.leaderboard[wallet] = stats
}

func (l *Lobby) incrementDNF(wallet string) {
	stats := l.leaderboard[wallet]
	stats.DNFs++
	stats.DisconnectStreak++
	if stats.DisconnectStreak > 3 {
		stats.BanExpires = time.Now().Add(24 * time.Hour)
	}
	l.updatePlayerPlaystyleTendencies(wallet, false, [2]int{}, []int{}, false) // Update playstyle on DNF (no match context)
	stats.Reputation = l.CalculateReputation(stats)                            // Update the map with modified stats
	l.leaderboard[wallet] = stats
	go l.recordDNFOnChain(wallet)
}

// processPrisonerRule checks if a card should be jailed based on match outcome and territory.
// This version jails the LOSER'S RAREST CARD.
func (l *Lobby) processPrisonerRule(match *MatchState, loserWallet, winnerWallet string) {
	// Rule applies if:
	// 1. Match has a defined territory.
	// 2. A Club owns this territory.
	// 3. The winner is associated with the owning Club.
	// 4. The loser is NOT associated with the owning Club.
	if match.TerritoryID == "" {
		return
	}

	owningClub := l.getClubByTerritoryID(match.TerritoryID)
	if owningClub == nil {
		return
	}

	// Check if winner is associated with the owning club
	winnerIsOwner := strings.EqualFold(owningClub.OwnerWallet, winnerWallet)
	winnerIsMember := false
	if _, ok := owningClub.Members[strings.ToLower(winnerWallet)]; ok {
		winnerIsMember = true
	}

	if !winnerIsOwner && !winnerIsMember {
		return // Winner is not associated with the territory's club
	}

	// Check if loser is NOT associated with the owning club
	loserIsOwner := strings.EqualFold(owningClub.OwnerWallet, loserWallet)
	loserIsMember := false
	if _, ok := owningClub.Members[strings.ToLower(loserWallet)]; ok {
		loserIsMember = true
	}

	if loserIsOwner || loserIsMember {
		return // Loser is associated with the territory's club, no jailing
	}

	// Conditions met for Prisoner Rule
	l.mutex.Lock()
	defer l.mutex.Unlock()

	rarestCard, found := l.findRarestCardInInventory(loserWallet)
	if !found {
		log.Printf("[PRISONER_RULE] No cards found in %s's inventory to jail.\n", loserWallet)
		return
	}

	// Transfer card to Club Jail
	if owningClub.Jail == nil {
		owningClub.Jail = make(map[int]ServerCard)
	}
	owningClub.Jail[rarestCard.ID] = rarestCard

	// Remove from loser's inventory
	loserStats := l.leaderboard[loserWallet]
	delete(loserStats.Inventory, fmt.Sprintf("CARD-%d", rarestCard.ID))
	if loserStats.JailedCards == nil {
		loserStats.JailedCards = make(map[int]string)
	}
	loserStats.JailedCards[rarestCard.ID] = owningClub.ID
	l.leaderboard[loserWallet] = loserStats

	log.Printf("[PRISONER_RULE] %s's rarest card (%s) jailed by Club %s in territory %s.\n", loserWallet, rarestCard.Name, owningClub.Name, match.TerritoryID)
	l.sendToClient(l.getClientIDFromWallet(loserWallet), Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🚨 <b>PRISONER RULE:</b> Your rarest card (%s) has been jailed by Club %s!"}`, escapeHTML(rarestCard.Name), escapeHTML(owningClub.Name)))})
	l.sendToClient(l.getClientIDFromWallet(winnerWallet), Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"⛓️ <b>PRISONER RULE:</b> You jailed %s's rarest card (%s)!"}`, escapeHTML(loserWallet), escapeHTML(rarestCard.Name)))})
}

// processFallenPenaltyJail implements the jailing logic when the Fallen_penalty rule is active.
// It jails ALL captured cards, not just the loser's rarest.
func (l *Lobby) processFallenPenaltyJail(match *MatchState, capturedCards []CapturedCardInfo) {
	if match.TerritoryID == "" || len(capturedCards) == 0 {
		return
	}

	owningClub := l.getClubByTerritoryID(match.TerritoryID)
	if owningClub == nil {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	for _, captured := range capturedCards {
		// Ensure the capturing player is associated with the owning club
		capturingPlayerIsOwner := strings.EqualFold(owningClub.OwnerWallet, captured.CapturingPlayerWallet)
		capturingPlayerIsMember := false
		if _, ok := owningClub.Members[strings.ToLower(captured.CapturingPlayerWallet)]; ok {
			capturingPlayerIsMember = true
		}

		if !capturingPlayerIsOwner && !capturingPlayerIsMember {
			continue // Capturing player is not associated with the territory's club, no jailing
		}

		// Ensure the original owner is NOT associated with the owning club
		originalOwnerIsOwner := strings.EqualFold(owningClub.OwnerWallet, captured.OriginalOwnerWallet)
		originalOwnerIsMember := false
		if _, ok := owningClub.Members[strings.ToLower(captured.OriginalOwnerWallet)]; ok {
			originalOwnerIsMember = true
		}

		if originalOwnerIsOwner || originalOwnerIsMember {
			continue // Original owner is associated with the territory's club, no jailing
		}

		// Conditions met for Fallen Penalty Jailing
		card, cardExists := l.inventory[captured.CardID]
		if !cardExists {
			log.Printf("[FALLEN_PENALTY_JAIL] Captured card %d not found in global inventory.\n", captured.CardID)
			continue
		}

		// Transfer card to Club Jail
		if owningClub.Jail == nil {
			owningClub.Jail = make(map[int]ServerCard)
		}
		owningClub.Jail[card.ID] = card

		// Remove from original owner's inventory
		originalOwnerStats := l.leaderboard[captured.OriginalOwnerWallet]
		delete(originalOwnerStats.Inventory, fmt.Sprintf("CARD-%d", card.ID))
		if originalOwnerStats.JailedCards == nil {
			originalOwnerStats.JailedCards = make(map[int]string)
		}
		originalOwnerStats.JailedCards[card.ID] = owningClub.ID
		l.leaderboard[captured.OriginalOwnerWallet] = originalOwnerStats

		log.Printf("[FALLEN_PENALTY_JAIL] %s's card (%s) jailed by Club %s in territory %s due to Fallen Penalty.\n", captured.OriginalOwnerWallet, card.Name, owningClub.Name, match.TerritoryID)
		l.sendToClient(l.getClientIDFromWallet(captured.OriginalOwnerWallet), Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🚨 <b>FALLEN PENALTY:</b> Your card '%s' has been jailed by Club %s!"}`, escapeHTML(card.Name), escapeHTML(owningClub.Name)))})
		l.sendToClient(l.getClientIDFromWallet(captured.CapturingPlayerWallet), Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"⛓️ <b>FALLEN PENALTY:</b> You jailed '%s's card (%s)!"}`, escapeHTML(captured.OriginalOwnerWallet), escapeHTML(card.Name)))})
	}
}

// applyItemEffectToMatch applies the in-match effects of an item to the MatchState or a specific card.
// This function is called by lobby_manager.go's use_item handler.
func (l *Lobby) applyItemEffectToMatch(match *MatchState, playerID string, itemID string, targetCardID int, targetGridIndex int) {
	item, itemExists := GlobalShopRegistry[itemID]
	if !itemExists {
		log.Printf("[BATTLE] Attempted to apply unknown item effect: %s\n", itemID)
		return
	}

	// Determine player index (0 for P1, 1 for P2)
	pIdx := 0
	if playerID == match.P2ID {
		pIdx = 1
	}

	// Initialize ActiveItemBuffs map if nil
	if match.ActiveItemBuffs == nil {
		match.ActiveItemBuffs = make(map[string]map[string]int)
	}
	if match.ActiveItemBuffs[playerID] == nil {
		match.ActiveItemBuffs[playerID] = make(map[string]int)
	}

	switch item.ClubType {
	case "Elemental":
		switch itemID {
		case "mood_catalyst": // +50 Mood Bonus (3 Matches)
			if targetGridIndex >= 0 && targetGridIndex < 9 && match.Board[targetGridIndex] != nil {
				card := match.Board[targetGridIndex]
				if card.Owner == pIdx {
					card.Artifact += 50                         // Apply as artifact bonus for simplicity in power calculation
					match.ActiveItemBuffs[playerID][itemID] = 3 // Track for 3 matches
					log.Printf("[BATTLE] Player %s used Mood Catalyst on card %d at grid %d. Artifact +50.\n", playerID, card.ID, targetGridIndex)
				}
			}
		case "grounded_shield": // Immunity to Mood Penalties (5 Matches)
			// This would typically be a rule override. For now, we track it.
			match.ActiveItemBuffs[playerID][itemID] = 5 // Track for 5 matches
			log.Printf("[BATTLE] Player %s activated Grounded Shield. Immunity for 5 matches.\n", playerID)
		}

	case "Tactical":
		switch itemID {
		case "rule_breaker": // Force PLUS trigger (1 Match)
			// This is a temporary rule. Add it to match.Rules and track its duration.
			match.Rules["Force_Plus_Trigger"] = true
			match.ActiveItemBuffs[playerID][itemID] = 1 // Track for 1 match
			log.Printf("[BATTLE] Player %s activated Rule Breaker. Force Plus Trigger for 1 match.\n", playerID)
		case "intel_report": // See Opponent Hand (3 Matches)
			// This would require a flag in MatchState and UI logic to reveal opponent's hand.
			// For now, we just track it.
			match.ActiveItemBuffs[playerID][itemID] = 3 // Track for 3 matches
			log.Printf("[BATTLE] Player %s activated Intel Report. Opponent hand visible for 3 matches.\n", playerID)
		}

	case "Vitality":
		// Vitality items (Stamina Stim, Loyalty Pledge) are handled directly in lobby_manager.go
		// as they modify persistent card stats, not just in-match state.
		log.Printf("[BATTLE] Vitality item %s used by %s. Persistent effect handled by lobby_manager.\n", itemID, playerID)

	default:
		log.Printf("[BATTLE] Unknown item type or effect for item %s used by %s.\n", itemID, playerID)
	}
}

// processItemBuffExpiration decrements the duration of active item buffs and removes expired ones.
func (l *Lobby) processItemBuffExpiration(match *MatchState) {
	if match.ActiveItemBuffs == nil {
		return
	}

	playersToProcess := []string{match.P1ID, match.P2ID}

	for _, playerID := range playersToProcess {
		if playerBuffs, ok := match.ActiveItemBuffs[playerID]; ok {
			newPlayerBuffs := make(map[string]int)
			for itemID, matchesRemaining := range playerBuffs {
				matchesRemaining-- // Decrement for the just-completed match
				if matchesRemaining > 0 {
					newPlayerBuffs[itemID] = matchesRemaining
				} else {
					log.Printf("[BATTLE] Item buff %s for player %s expired.\n", itemID, playerID)
					// Remove any temporary rules applied by this item
					switch itemID {
					case "rule_breaker":
						delete(match.Rules, "Force_Plus_Trigger")
					case "intel_report":
						// Logic to hide opponent's hand if it was revealed
					}
				}
			}
			if len(newPlayerBuffs) > 0 {
				match.ActiveItemBuffs[playerID] = newPlayerBuffs
			} else {
				delete(match.ActiveItemBuffs, playerID) // Remove player entry if no buffs remain
			}
		}
	}
}
