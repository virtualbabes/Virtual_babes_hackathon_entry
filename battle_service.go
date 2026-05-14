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
	// Determinism Sync: Artifact bonuses are added unconditionally to match main.go logic
	base := c.Power[sideIdx] + c.Artifact

	// Resolve Player Identity & Snapshotted Stats
	pID := match.P1ID
	wanted := match.P1WantedLevel
	cunning := match.P1Cunning
	nurturing := match.P1Nurturing

	if c.Owner == 1 {
		pID = match.P2ID
		wanted = match.P2WantedLevel
		cunning = match.P2Cunning
		nurturing = match.P2Nurturing
	}

	// Dynamic Buff Check: Re-apply global match-wide power boosts.
	if match.ActiveItemBuffs != nil && match.ActiveItemBuffs[pID] != nil {
		if _, ok := match.ActiveItemBuffs[pID]["mood_catalyst"]; ok {
			base += 50
		}
	}

	// Apply Wanted Level Penalty (Mitigated by Cunning)
	wantedPenalty := (wanted * 5)
	// Cunning mitigates penalty: every 1 point of Cunning reduces penalty by 2
	// Hardening: Use snapshotted values from the match state to ensure consistency.
	// These values are captured in lobby_manager.go during initiatePairedMatch.
	mitigation := cunning * 2
	if mitigation > wantedPenalty { mitigation = wantedPenalty }
	base -= (wantedPenalty - mitigation)

	// Fatigue Penalty: -1 power per point above 50
	if c.Fatigue > 50 {
		fatiguePenalty := (c.Fatigue - 50)
		// Nurturing reduces fatigue impact: 1 power back per Nurturing point
		reduction := nurturing
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

			// Check for "grounded_shield" buff (Immunity to Mood Penalties)
			var hasGroundedShield bool
			if match.ActiveItemBuffs != nil && match.ActiveItemBuffs[pID] != nil {
				if _, ok := match.ActiveItemBuffs[pID]["grounded_shield"]; ok {
					hasGroundedShield = true
				}
			}

			if c.Mood == tileMood {
				base += 50 // Match bonus: +0.5 Tier
			} else if moodWeaknesses[c.Mood] == tileMood { // Check for weakness
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
	// Hardening: Resolve wallets from match snapshot to handle mid-turn disconnects.
	capturingPlayerWallet := match.P1Wallet
	pID := match.P1ID
	if pIdx == 1 {
		capturingPlayerWallet = match.P2Wallet
		pID = match.P2ID
	}

	// Check for "rule_breaker" buff (Force_Plus_Trigger)
	forcePlusTrigger := false
	if match.ActiveItemBuffs != nil && match.ActiveItemBuffs[pID] != nil {
		if _, ok := match.ActiveItemBuffs[pID]["rule_breaker"]; ok {
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
				originalOwnerWallet := match.P1Wallet // Default to P1's wallet
				if neighbor.Owner == 1 {
					originalOwnerWallet = match.P2Wallet
				} // If neighbor was P2's card

				// Fallen Penalty Rule: Captured cards lose 20 Artifact power (Deterministic Sync with WASM)
				if match.Rules["Fallen_penalty"] {
					neighbor.Artifact -= 20
				}

				neighbor.Owner = pIdx
				totalFlips++
				capturedCards = append(capturedCards, CapturedCardInfo{
					CardID:                neighbor.ID,
					OriginalOwnerWallet:   originalOwnerWallet,
					CapturingPlayerWallet: capturingPlayerWallet,
					CaptureType:           "BASIC",
					GridIndex:             nbIdx,
					Round:                 match.Round,
				})
			}
		}
	}

	// 2. Process Rule Triggers
	for _, indices := range sameGroups {
		if len(indices) >= 2 {
			for _, idx := range indices {
				if match.Board[idx].Owner != pIdx {
					originalOwnerWallet := match.P1Wallet
					if match.Board[idx].Owner == 1 {
						originalOwnerWallet = match.P2Wallet
					}

					// Fallen Penalty Rule: Captured cards lose 20 Artifact power
					if match.Rules["Fallen_penalty"] {
						match.Board[idx].Artifact -= 20
					}

					match.Board[idx].Owner = pIdx
					totalFlips++
					capturedCards = append(capturedCards, CapturedCardInfo{
						CardID:                match.Board[idx].ID,
						OriginalOwnerWallet:   originalOwnerWallet,
						CapturingPlayerWallet: capturingPlayerWallet,
						CaptureType:           "SAME",
						GridIndex:             idx,
						Round:                 match.Round,
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
					originalOwnerWallet := match.P1Wallet
					if match.Board[idx].Owner == 1 {
						originalOwnerWallet = match.P2Wallet
					}

					// Fallen Penalty Rule: Captured cards lose 20 Artifact power
					if match.Rules["Fallen_penalty"] {
						match.Board[idx].Artifact -= 20
					}

					match.Board[idx].Owner = pIdx
					totalFlips++
					capturedCards = append(capturedCards, CapturedCardInfo{
						CardID:                match.Board[idx].ID,
						OriginalOwnerWallet:   originalOwnerWallet,
						CapturingPlayerWallet: capturingPlayerWallet,
						CaptureType:           "POWER_UP",
						GridIndex:             idx,
						Round:                 match.Round,
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

					// Fallen Penalty Rule: Captured cards lose 20 Artifact power
					if match.Rules["Fallen_penalty"] {
						neighbor.Artifact -= 20
					}

					oldOwner := neighbor.Owner
					neighbor.Owner = pIdx
					// Only add to capturedCards if it wasn't already flipped by a direct capture or rule
					// This prevents double-counting for jailing
					alreadyCaptured := false
					for _, cc := range capturedCards {
						if cc.GridIndex == nbIdx {
							alreadyCaptured = true
							break
						}
					}
					if !alreadyCaptured {
						originalOwnerWallet := match.P1Wallet
						if oldOwner == 1 {
							originalOwnerWallet = match.P2Wallet
						}
						capturedCards = append(capturedCards, CapturedCardInfo{
							CardID:                neighbor.ID,
							OriginalOwnerWallet:   originalOwnerWallet,
							CapturingPlayerWallet: capturingPlayerWallet,
							CaptureType:           "COMBO",
							GridIndex:             nbIdx,
							Round:                 match.Round,
						})
					}
					totalFlips++
					comboQueue = append(comboQueue, nbIdx)
				}
			}
		}
	}
	return totalFlips, capturedCards
}

// verifyWinner determines the match outcome and initiates reward/jailing logic.
// Note: This function assumes the Lobby mutex is already held by the caller.
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
		// PILLAR 3: Prevent Capture Amnesty. Jail cards flipped before the tie-breaker.
		if match.Rules["Fallen_penalty"] && len(match.CapturedCards) > 0 {
			l.processFallenPenaltyJailLocked(match, match.CapturedCards)
			match.CapturedCards = []CapturedCardInfo{} // Clear queue to prevent double-jailing
		}

		l.initiateSuddenDeath(match)
		return
	}

	match.IsFinished = true
	history := MatchHistory{
		Scores:           [2]int{p1, p2},
		Timestamp:        time.Now(),
		TournamentMatchID: match.TournamentMatchID,
		IsBountyMatch:    match.IsBountyMatch,
		ActiveItemBuffs:  match.ActiveItemBuffs, // Snapshot item effects into history for archival
		CapturedCards:    match.CapturedCards,   // Preserve detailed capture log for audit
	}

	if p1 > p2 {
		history.WinnerID, history.WinnerIndex = match.P1ID, 0
		history.Opponent = match.P2Wallet
		l.finalizeMatchResultLocked(match.P1ID, match.P1Deck, history)

		// Achievement: Perfect Game (10-0)
		if p1 == 10 {
			l.unlockAchievementLocked(match.P1Wallet, "PERFECT_GAME")
		}
	} else if p2 > p1 {
		history.WinnerID, history.WinnerIndex = match.P2ID, 1
		history.Opponent = match.P1Wallet
		l.finalizeMatchResultLocked(match.P2ID, match.P2Deck, history)
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
			hunterWallet := match.P1Wallet
			if hunterID == match.P2ID { hunterWallet = match.P2Wallet }

			history.BountyReward = float64(targetWanted * 50)
			l.sendToClientLocked(hunterID, Envelope{
				Type:    "admin_notification",
				Payload: json.RawMessage(fmt.Sprintf(`{"text":"🎯 <b>BOUNTY CLAIMED!</b> You've earned %.2f bonus $VBV."}`, history.BountyReward)),
			})
			l.unlockAchievementLocked(hunterWallet, "OUTLAW_SLAYER")
		}
	}

	// Decrement and remove expired item buffs for both players
	l.processItemBuffExpiration(match)

	// PRISONER RULE: Decide which jailing logic to apply
	if match.Rules["Fallen_penalty"] && len(match.CapturedCards) > 0 {
		l.processFallenPenaltyJailLocked(match, match.CapturedCards) // Jail all captured cards if Fallen_penalty is active
	} else if p1 != p2 { // Only apply original prisoner rule if there's a clear loser
		// Original Prisoner Rule: Jail loser's rarest card (if Fallen_penalty is not active or no cards were captured)
		if p1 > p2 { // P1 won, P2 lost, so P2 is the loser
			l.processPrisonerRuleLocked(match, match.P2Wallet, match.P1Wallet)
		} else if p2 > p1 { // P2 won, P1 lost
			l.processPrisonerRuleLocked(match, match.P1Wallet, match.P2Wallet)
		}
	}

	delete(l.matches, match.P1ID)
	delete(l.matches, match.P2ID)
}

// initiateSuddenDeath shuffles and redistributes remaining hand cards for a high-stakes tie-breaker.
func (l *Lobby) initiateSuddenDeath(match *MatchState) {
	var p1NewDeck []int
	var p2NewDeck []int
	match.Round++ // Increment round for capture instance isolation

	// Reconstruct hands based on current board ownership to handle tie-breakers.
	// Hardening: We must handle duplicate card IDs correctly to prevent unplayed cards from being lost.
	// Frequency maps of starting decks allow us to identify which specific instances remain in hands.
	p1Starting := make(map[int]int)
	for _, id := range match.P1Deck {
		p1Starting[id]++
	}
	p2Starting := make(map[int]int)
	for _, id := range match.P2Deck {
		p2Starting[id]++
	}

	for _, c := range match.Board {
		if c == nil {
			continue
		}

		// Redistribute: Cards on the board move to the CURRENT owner's deck.
		if c.Owner == 0 {
			p1NewDeck = append(p1NewDeck, c.ID)
		} else {
			p2NewDeck = append(p2NewDeck, c.ID)
		}

		// Decrement from starting pools to track used instances.
		if p1Starting[c.ID] > 0 {
			p1Starting[c.ID]--
		} else if p2Starting[c.ID] > 0 {
			p2Starting[c.ID]--
		}
	}

	// Any instances remaining in starting pools were never played; they return to the original owners.
	for id, count := range p1Starting {
		for i := 0; i < count; i++ {
			p1NewDeck = append(p1NewDeck, id)
		}
	}
	for id, count := range p2Starting {
		for i := 0; i < count; i++ {
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
		"active_item_buffs": match.ActiveItemBuffs, // Sync active buffs to client UI
		"rules":             match.Rules,             // Sync authoritative rules
	})

	l.sendToClientLocked(match.P1ID, Envelope{Type: "sudden_death_start", FromID: "SERVER", Payload: payload})
	l.sendToClientLocked(match.P2ID, Envelope{Type: "sudden_death_start", FromID: "SERVER", Payload: payload})
	log.Printf("[BATTLE] Sudden Death tie-breaker initiated for match %s vs %s\n", match.P1ID, match.P2ID)
}

func (l *Lobby) finalizeMatchResultLocked(winnerID string, deck []int, history MatchHistory) {
	l.matchHistory[winnerID] = history
	if wallet, ok := l.wallets[winnerID]; ok {
		l.updateLeaderboard(wallet, history.TournamentMatchID != "", history.Scores, deck, history.IsBountyMatch) // Pass match context
		go func() {
			rating := l.calculateDeckRating(deck)
			l.mutex.Lock() // Lock for leaderboard modification
			s := l.leaderboard[wallet]

			// Achievement: Arena Legend (100 Wins)
			if s.Wins == 100 {
				l.unlockAchievementLocked(wallet, "ARENA_LEGEND")
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
	l.updatePlayerPlaystyleTendenciesLocked(wallet, true, scores, deck, isBountyWin, isTournamentWin) // Pass tournament context

	// REFRESH: Fetch updated Playstyle from the map to prevent clobbering behavioral data.
	stats.Playstyle = l.leaderboard[wallet].Playstyle
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
	l.updatePlayerPlaystyleTendenciesLocked(wallet, false, [2]int{}, []int{}, false) // Update playstyle on DNF

	// REFRESH: Sync local stats with the newly calculated playstyle before calculating Standing.
	stats.Playstyle = l.leaderboard[wallet].Playstyle
	stats.Reputation = l.CalculateReputation(stats)                            // Update the map with modified stats
	l.leaderboard[wallet] = stats
	go l.recordDNFOnChain(wallet)
}

// processPrisonerRule checks if a card should be jailed based on match outcome and territory.
// This version jails the LOSER'S RAREST CARD.
func (l *Lobby) processPrisonerRuleLocked(match *MatchState, loserWallet, winnerWallet string) {
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

	// Hardening: Ensure the loser is a valid player with a persistent record (not AI or tournament BYE)
	if loserWallet == "" || strings.EqualFold(loserWallet, "BYE") {
		return
	}

	// Conditions met for Prisoner Rule
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

	owningClub.LastActivity = time.Now() // Industrial Loop: Turf defense refreshes club activity
	// Remove from loser's inventory
	loserStats := l.leaderboard[loserWallet]
	delete(loserStats.Inventory, fmt.Sprintf("CARD-%d", rarestCard.ID))
	if loserStats.JailedCards == nil {
		loserStats.JailedCards = make(map[int]string)
	}
	loserStats.JailedCards[rarestCard.ID] = owningClub.ID
	l.leaderboard[loserWallet] = loserStats

	log.Printf("[PRISONER_RULE] %s's rarest card (%s) jailed by Club %s in territory %s.\n", loserWallet, rarestCard.Name, owningClub.Name, match.TerritoryID)
	l.sendToClientLocked(l.getClientIDFromWalletLocked(loserWallet), Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🚨 <b>PRISONER RULE:</b> Your rarest card (%s) has been jailed by Club %s!"}`, escapeHTML(rarestCard.Name), escapeHTML(owningClub.Name)))})
	l.sendToClientLocked(l.getClientIDFromWalletLocked(winnerWallet), Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"⛓️ <b>PRISONER RULE:</b> You jailed %s's rarest card (%s)!"}`, escapeHTML(loserWallet), escapeHTML(rarestCard.Name)))})
}

// processFallenPenaltyJail implements the jailing logic when the Fallen_penalty rule is active.
// It jails ALL captured cards, not just the loser's rarest.
func (l *Lobby) processFallenPenaltyJailLocked(match *MatchState, capturedCards []CapturedCardInfo) {
	if match.TerritoryID == "" || len(capturedCards) == 0 {
		return
	}

	owningClub := l.getClubByTerritoryID(match.TerritoryID)
	if owningClub == nil {
		return
	}

	// High-Fidelity Jailing: Use Round and GridIndex to ensure each capture event is handled,
	// preventing collisions during Sudden Death or when players use duplicate card archetypes.
	jailedThisMatch := make(map[string]bool) 

	for _, captured := range capturedCards {
		jailKey := fmt.Sprintf("%d-%d", captured.Round, captured.GridIndex)
		if jailedThisMatch[jailKey] {
			continue
		}

		// Hardening: Ensure the original owner is a valid player with a leaderboard record.
		// Tournament BYE or AI opponents should never have their "cards" jailed.
		if captured.OriginalOwnerWallet == "" || strings.EqualFold(captured.OriginalOwnerWallet, "BYE") {
			continue
		}

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
		cardKey := fmt.Sprintf("CARD-%d", captured.CardID)
		originalOwnerStats, exists := l.leaderboard[captured.OriginalOwnerWallet]
		if !exists {
			continue
		}

		// CRITICAL AUDIT FIX: Verify the original owner actually possesses the card in their persistent inventory.
		// This prevents attempting to jail "board-only" captures or causing negative inventory counts.
		count, hasCard := originalOwnerStats.Inventory[cardKey]
		if !hasCard || count <= 0 {
			log.Printf("[FALLEN_PENALTY_JAIL] Card %d not found in %s's persistent collection (Capture: %s). Skipping.\n", 
				captured.CardID, captured.OriginalOwnerWallet, captured.CaptureType)
			continue
		}

		// Use the card instance from the board, which contains the Artifact reductions applied during the match.
		// This ensures the "Fallen_penalty" power loss is captured for persistence.
		cardPtr := match.Board[captured.GridIndex]
		if cardPtr == nil || cardPtr.ID != captured.CardID {
			continue
		}
		card := *cardPtr

		// INDUSTRIAL LOOP: Persist the "Battle Scars" (Artifact reduction) to the global cache.
		// This makes the power loss permanent for this card archetype in the Arena ecosystem.
		l.inventory[card.ID] = card
		l.persistentCardCache[card.ID] = card

		// Transfer card to Club Jail
		if owningClub.Jail == nil {
			owningClub.Jail = make(map[int]ServerCard)
		}
		owningClub.Jail[card.ID] = card
		owningClub.LastActivity = time.Now() // Defensive success prevents Mojo decay

		// Remove from original owner's inventory (Decrementing instead of absolute deletion)
		originalOwnerStats.Inventory[cardKey]--
		if originalOwnerStats.Inventory[cardKey] <= 0 {
			delete(originalOwnerStats.Inventory, cardKey)
		}

		if originalOwnerStats.JailedCards == nil {
			originalOwnerStats.JailedCards = make(map[int]string)
		}
		originalOwnerStats.JailedCards[card.ID] = owningClub.ID
		l.leaderboard[captured.OriginalOwnerWallet] = originalOwnerStats
		jailedThisMatch[jailKey] = true
		match.Board[captured.GridIndex] = nil // Seized cards leave the arena immediately

		log.Printf("[FALLEN_PENALTY_JAIL] %s's card (%s) jailed by Club %s via %s capture in %s.\n", 
			captured.OriginalOwnerWallet, card.Name, owningClub.Name, captured.CaptureType, match.TerritoryID)

		// Use CaptureType in client notifications for high-fidelity tactical feedback
		l.sendToClientLocked(l.getClientIDFromWalletLocked(captured.OriginalOwnerWallet), Envelope{
			Type:    "admin_notification",
			Payload: json.RawMessage(fmt.Sprintf(`{"text":"🚨 <b>FALLEN PENALTY:</b> Your card '%s' was seized via %s and jailed by Club %s!"}`, 
				escapeHTML(card.Name), captured.CaptureType, escapeHTML(owningClub.Name))),
		})
		l.sendToClientLocked(l.getClientIDFromWalletLocked(captured.CapturingPlayerWallet), Envelope{
			Type:    "admin_notification",
			Payload: json.RawMessage(fmt.Sprintf(`{"text":"⛓️ <b>FALLEN PENALTY:</b> You jailed '%s's card (%s) via %s capture!"}`, 
				escapeHTML(captured.OriginalOwnerWallet), escapeHTML(card.Name), captured.CaptureType)),
		})
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
			match.ActiveItemBuffs[playerID][itemID] = 3 // Track for 3 matches
			log.Printf("[BATTLE] Player %s activated Mood Catalyst. +50 Power for 3 matches.\n", playerID)
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
