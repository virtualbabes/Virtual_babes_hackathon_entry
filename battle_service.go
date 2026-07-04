//go:build !js && !wasm

package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"
)

// getEffectiveServerPower replicates the modifier logic for server validation
func getEffectiveServerPower(_ *Lobby, match *MatchState, c *ServerCard, sideIdx int, gridIdx int) int {
	// Determinism Sync: Artifact bonuses are added unconditionally to match main.go logic
	base := c.Power[sideIdx] + c.Artifact

	// Resolve Player Identity & Snapshotted Stats
	pID := match.P1ID
	wanted := match.P1WantedLevel
	cunning := match.P1Cunning
	nurturing := match.P1Nurturing
	regBoost := match.P1RegionalBoost
	coalitionBoost := match.P1CoalitionBoost

	if c.Owner == 1 {
		pID = match.P2ID
		wanted = match.P2WantedLevel
		cunning = match.P2Cunning
		nurturing = match.P2Nurturing
		regBoost = match.P2RegionalBoost
		coalitionBoost = match.P2CoalitionBoost
	}

	// PILLAR 7: Fallen Status Penalty.
	if c.Fallen {
		base -= 50
	}

	// Dynamic Buff Check: Re-apply global match-wide power boosts.
	if match.ActiveItemBuffs != nil && match.ActiveItemBuffs[pID] != nil {
		if _, ok := match.ActiveItemBuffs[pID]["mood_catalyst"]; ok {
			base += 50
		}
	}

	// PILLAR 1: Coalition & Regional Power Boosts.
	// We prioritize the +10% Coalition boost for allied members defending a partner's turf.
	// This encourages inter-organizational protection. If the player is not an ally 
	// but belongs to the owning regional club, they receive the standard +5% boost.
	if coalitionBoost {
		base += (base * 10) / 100
	} else if regBoost {
		base += (base * 5) / 100
	}

	// Apply Wanted Level Penalty (Mitigated by Cunning)
	wantedPenalty := (wanted * 5)
	// Cunning mitigates penalty: every 1 point of Cunning reduces penalty by 2
	// Hardening: Use snapshotted values from the match state to ensure consistency. (This is already using match.P1Cunning, not PlayerStats.GetEffectiveCunning)
	// These values are captured in lobby_manager.go during initiatePairedMatch.
	mitigation := cunning * 2
	if mitigation > wantedPenalty {
		mitigation = wantedPenalty
	}
	base -= (wantedPenalty - mitigation)

	// Fatigue Penalty: -1 power per point above 50
	if c.Fatigue > 50 {
		fatiguePenalty := (c.Fatigue - 50)
		// Nurturing reduces fatigue impact: 1 power back per Nurturing point
		reduction := nurturing
		if reduction > fatiguePenalty {
			reduction = fatiguePenalty
		}
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

// ComputeBoardHash creates a deterministic checksum of the board for client verification.
// PILLAR 4: Bit-Perfect Parity.
func (l *Lobby) ComputeBoardHash(board [9]*ServerCard, playerIndex int) BoardStateHash {
	hasher := sha256.New()
	for _, c := range board {
		var id uint32 = 0
		var owner int32 = -1
		var artifact int32 = 0

		if c != nil {
			id = uint32(c.ID)
			owner = int32(c.Owner)
			artifact = int32(c.Artifact)
		}

		// Write ID, Owner, and Artifact as fixed-width binary to ensure 
		// cross-platform parity between 64-bit Server and 32-bit WASM.
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, id)
		hasher.Write(buf)
		binary.BigEndian.PutUint32(buf, uint32(owner))
		hasher.Write(buf)
		binary.BigEndian.PutUint32(buf, uint32(artifact))
		hasher.Write(buf)
	}

	// Include the PlayerIndex (turn actor) in the hash to bound the state transition
	turnBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(turnBuf, uint32(playerIndex))
	hasher.Write(turnBuf)

	var hash BoardStateHash
	copy(hash[:], hasher.Sum(nil))
	return hash
}

func (l *Lobby) serverCheckCaptures(match *MatchState, gridIndex int, pIdx int) (int, []CapturedCardInfo, BoardStateHash) {
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

	// PILLAR 3: Factional Alignment Analysis
	attackerStats := l.leaderboard[capturingPlayerWallet]
	attackerFaction := l.playerService.GetHegemonyPath(attackerStats.JobRole)

	oppID := match.P2ID
	oppWallet := match.P2Wallet
	oppWanted := match.P2WantedLevel
	if pIdx == 1 {
		oppID = match.P1ID
		oppWallet = match.P1Wallet
		oppWanted = match.P1WantedLevel
	}
	oppStats := l.leaderboard[oppWallet]
	oppFaction := l.playerService.GetHegemonyPath(oppStats.JobRole)

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

			// PILLAR 3: Factional Power Scaling (Section 12.A & 12.B)
			if attackerFaction == "JUSTICE" {
				// Justice Boost: vs Fallen cards or Outlaws (Wanted >= 15)
				if neighbor.Fallen || oppWanted >= 15 {
					pPower = (pPower * 110) / 100
				}
			} else if attackerFaction == "UNDERWORLD" {
				// Underworld Boost: vs Justice aligned players or Clean signatures (Wanted <= 2)
				if oppFaction == "JUSTICE" || oppWanted <= 2 {
					pPower = (pPower * 110) / 100
				}
			}

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
					l.applyCapturePenalty(match, neighbor)
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
						l.applyCapturePenalty(match, match.Board[idx])
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
						l.applyCapturePenalty(match, match.Board[idx])
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
				cPower := getEffectiveServerPower(l, match, currCard, 0, currIdx)
				nPower := getEffectiveServerPower(l, match, neighbor, 0, nbIdx)

				// PILLAR 3: Factional Power Scaling in Combo
				if attackerFaction == "JUSTICE" {
					if neighbor.Fallen || oppWanted >= 15 {
						cPower = (cPower * 110) / 100
					}
				} else if attackerFaction == "UNDERWORLD" {
					if oppFaction == "JUSTICE" || oppWanted <= 2 {
						cPower = (cPower * 110) / 100
					}
				}

				if neighbor.Owner != pIdx && cPower > nPower {

					// Fallen Penalty Rule: Captured cards lose 20 Artifact power
					if match.Rules["Fallen_penalty"] {
						l.applyCapturePenalty(match, neighbor)
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

	// PILLAR 4: Deterministic Sync. 
	// Return the board hash after all captures and combos have settled.
	return totalFlips, capturedCards, l.ComputeBoardHash(match.Board, pIdx)
}

// applyCapturePenalty handles Artifact reduction and global persistence during capture events.
// PILLAR 4: Deterministic Sync.
func (l *Lobby) applyCapturePenalty(match *MatchState, card *ServerCard) {
	if match.Rules["Fallen_penalty"] {
		card.Artifact -= 20

		// PILLAR 6: Selective Forensic Persistence.
		// We must only persist Artifact changes (scars). Transient match properties 
		// like territory-based 'EquippedItems' (traps) must not leak into the 
		// global inventory or persistent cache.
		if persistentCard, exists := l.inventory[card.ID]; exists {
			persistentCard.Artifact = card.Artifact
			l.inventory[card.ID] = persistentCard
			l.persistentCardCache[card.ID] = persistentCard
		} else {
			// Fallback: If the card isn't in inventory (rare), 
			// create an entry and ensure transient items are stripped.
			cleanCard := *card
			cleanCard.EquippedItems = nil
			l.inventory[card.ID] = cleanCard
			l.persistentCardCache[card.ID] = cleanCard
		}
	}
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
		// PILLAR 3: Economic Consequence.
		// Increment fatigue for all cards on the board before they are jailed or redistributed.
		l.processMatchFatigueLocked(match)

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
		Scores:            [2]int{p1, p2},
		Timestamp:         time.Now(),
		TournamentID:      match.TournamentID,
		TournamentMatchID: match.TournamentMatchID,
		IsBountyMatch:     match.IsBountyMatch,
		P1WantedLevel:     match.P1WantedLevel,
		P2WantedLevel:     match.P2WantedLevel,
		P1Cunning:         match.P1Cunning,
		P2Cunning:         match.P2Cunning,
		P1Nurturing:       match.P1Nurturing,
		P2Nurturing:       match.P2Nurturing,
		ActiveItemBuffs:   match.ActiveItemBuffs,
		CapturedCards:     match.CapturedCards,
	}

	if p1 > p2 {
		history.WinnerID, history.WinnerIndex = match.P1ID, 0
		history.Opponent = match.P2Wallet
		l.finalizeMatchResultLocked(match, history)

		// Achievement: Perfect Game (10-0)
		if p1 == 10 {
			l.achievementService.UnlockAchievementLocked(l, match.P1Wallet, "PERFECT_GAME")
		}
	} else if p2 > p1 {
		history.WinnerID, history.WinnerIndex = match.P2ID, 1
		history.Opponent = match.P1Wallet
		l.finalizeMatchResultLocked(match, history)
	} else { // Draw
		history.WinnerID, history.WinnerIndex = "", 2 // 2 for Draw
		history.Opponent = "DRAW"

		// PILLAR 1: Bracket Integrity.
		// If Sudden Death was bypassed or disabled in a tournament, resolve via Reputation to prevent stall.
		if match.TournamentMatchID != "" {
			p1Stats := l.leaderboard[match.P1Wallet]
			p2Stats := l.leaderboard[match.P2Wallet]

			log.Printf("[BATTLE] Tournament Draw safety trigger. Resolving via Reputation for %s vs %s\n", match.P1Wallet, match.P2Wallet)

			if p1Stats.Reputation >= p2Stats.Reputation {
				history.WinnerID, history.WinnerIndex = match.P1ID, 0
				history.Opponent = match.P2Wallet
				l.finalizeMatchResultLocked(match, history)
			} else {
				history.WinnerID, history.WinnerIndex = match.P2ID, 1
				history.Opponent = match.P1Wallet
				l.finalizeMatchResultLocked(match, history)
			}
		}

		// PILLAR 4: Immersion. Update ephemeral history for standard draws.
		// This ensures non-tournament ties appear in the player's recent history list.
		if match.TournamentMatchID == "" {
			for _, wallet := range []string{match.P1Wallet, match.P2Wallet} {
				if wallet == "" || strings.EqualFold(wallet, "BYE") {
					continue
				}
				l.ensurePlayerStatsMapsInitialized(wallet)
				st := l.leaderboard[wallet]
				drawRecord := history
				drawRecord.Opponent = match.P2Wallet
				if strings.EqualFold(wallet, match.P2Wallet) {
					drawRecord.Opponent = match.P1Wallet
				}
				drawRecord.WinnerIndex = 2 // 2=Draw
				st.History = append([]MatchHistory{drawRecord}, st.History...)
				if len(st.History) > 15 {
					st.History = st.History[:15]
				}
				l.leaderboard[wallet] = st
			}
		}

		// PILLAR 7: Underworld Streak Reset (Draw Case).
		// A draw breaks a win streak. Both players' progress is reset.
		for _, w := range []string{match.P1Wallet, match.P2Wallet} {
			if w == "" || strings.EqualFold(w, "BYE") {
				continue
			}
			l.ensurePlayerStatsMapsInitialized(w)
			st := l.leaderboard[w]
			if st.RecoveryChallengeCardID != 0 {
				st.RecoveryChallengeWins = 0
				l.leaderboard[w] = st
				if cid := l.getClientIDFromWalletLocked(w); cid != "" {
					l.sendToClientLocked(cid, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"💀 <b>CHALLENGE FAILED:</b> Match draw. Retrieval streak reset to 0."}`)})
				}
			}
		}
	}

	// PILLAR 3: Justice Mission Completion (MISSION-001)
	// Apprehend High-Infamy Signatures: Capture a player with Wanted Level 15+ in combat.
	if history.WinnerID != "" {
		winnerWallet := l.wallets[history.WinnerID]
		winnerStats := l.leaderboard[winnerWallet]

		if winnerStats.ActiveJusticeMissionID == "MISSION-001" {
			oppWanted := match.P2WantedLevel
			if history.WinnerIndex == 1 {
				oppWanted = match.P1WantedLevel
			}

			if oppWanted >= 15 {
				const rewardMicro = 1200 * 1000000
				l.playerBalances[winnerWallet] += rewardMicro
				winnerStats.ActiveJusticeMissionID = ""
				l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", winnerWallet, "ID: MISSION-001, Payout: 1200.00")
				l.sendToClientLocked(history.WinnerID, Envelope{
					Type:    "admin_notification",
					Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> High-infamy signature apprehended. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0)),
				})
				l.leaderboard[winnerWallet] = winnerStats
			}
		}

		// PILLAR 3: Justice Mission Completion (MISSION-004)
		// Regional Peacekeeping: Apprehend a signature associated with a Regional Governor.
		if winnerStats.ActiveJusticeMissionID == "MISSION-004" {
			loserWallet := history.Opponent
			if loserWallet != "" && loserWallet != "DRAW" && loserWallet != "BYE" {
				l.ensurePlayerStatsMapsInitialized(loserWallet)
				loserStats := l.leaderboard[loserWallet]

				// Check if loser is associated with a Regional Governor's organization
				if loserStats.EmployerClubID != "" {
					loserClub, clubExists := l.clubs[loserStats.EmployerClubID]
					if clubExists && l.clubService.IsClubRegionalLocked(l, loserClub) {
						// Ensure the loser is actually a member/staff of that club, not just a freelancer.
						// This prevents flagging a random player who once worked for a governor.
						if l.clubService.IsPlayerAffiliatedWithClubLocked(l, loserWallet, loserClub) {
							const rewardMicro = 1000 * 1000000
							l.playerBalances[winnerWallet] += rewardMicro
							winnerStats.ActiveJusticeMissionID = ""
							l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", winnerWallet, "ID: MISSION-004, Payout: 1000.00")
							l.sendToClientLocked(history.WinnerID, Envelope{
								Type:    "admin_notification",
								Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> Regional Governor operative apprehended. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0)),
							})
							l.leaderboard[winnerWallet] = winnerStats
							l.applyDynamicScalingLocked() // Synchronize scaling with new virtual liability
						}
					}
				}
			}
		}
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

		hunterWallet := ""
		isSolo := false
		if history.WinnerID == hunterID {
			hunterWallet = match.P1Wallet
			if hunterID == match.P2ID {
				hunterWallet = match.P2Wallet
			}

			// PILLAR 3: Solo Hunter vs AOS Rivalry (Section 14).
			// 2x Reputation bonus for Bounty Hunters operating solo (no employer_id).
			history.BountyRewardMicro = uint64(targetWanted) * 50 * 1000000
			// PILLAR 1: Bounty Hunter Tax (5%) to fund the Justice Faction recruitment pool.
			history.BountyRewardMicro = l.ApplyBountyHunterTaxLocked(history.BountyRewardMicro)

			hStats := l.leaderboard[hunterWallet]
			isSolo = hStats.EmployerClubID == ""

			if isSolo {
				// Apply double reputation weighting for this victory
				hStats.Reputation += l.CalculateReputation(hStats)
				l.leaderboard[hunterWallet] = hStats
				l.sendToClientLocked(hunterID, Envelope{
					Type:    "admin_notification",
					Payload: json.RawMessage(`{"text":"👑 <b>SOLO CAPTURE:</b> Reputation gain doubled for solo enforcement."}`),
				})
			}

			l.sendToClientLocked(hunterID, Envelope{
				Type:    "admin_notification",
				Payload: json.RawMessage(fmt.Sprintf(`{"text":"🎯 <b>BOUNTY CLAIMED!</b> You've earned %.2f bonus $VBV."}`, float64(history.BountyRewardMicro)/1000000.0)),
			})
			l.unlockAchievementLocked(hunterWallet, "OUTLAW_SLAYER")
		}

		if hunterWallet != "" {
			l.achievementService.CheckBountyHunterAchievementLocked(l, hunterWallet, history.Opponent, targetWanted)

			// CAREER D1: Bounty Hunter — Track XP per bounty capture
			l.ensurePlayerStatsMapsInitialized(hunterWallet)
			stats := l.leaderboard[hunterWallet]
			baseXP := 60 + (targetWanted / 5)
			stats.CareerXP["Bounty Hunter"] += baseXP

			// Career progression milestones (D1: 4 tiers)
			level := stats.CareerLevel["Bounty Hunter"] + 1
			const xpPerLevel = 300
			for stats.CareerXP["Bounty Hunter"] >= level*xpPerLevel && level <= 4 {
				stats.CareerLevel["Bounty Hunter"] = level
				if cid := l.getClientIDFromWalletLocked(hunterWallet); cid != "" {
					l.sendToClientLocked(cid, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"⭐ <b>BOUNTY HUNTER PROMOTION:</b> Reached level %d!"}`, level))})
				}
				level++
			}

			l.leaderboard[hunterWallet] = stats
			l.logAdminAuditLocked("CAREER_BOUNTY_HUNTER_XP", hunterWallet, fmt.Sprintf("+%d XP (target wanted: %d)", baseXP, targetWanted))

			// P2-D1 Mechanic Hook: Bounty Hunter tracking speed bonus applied to capture range
			if stats.CareerXP != nil {
				trackingBonus := stats.CareerXP.GetBountyTrackingBonus()
				if trackingBonus > 1.0 {
					l.logAdminAuditLocked("CAREER_BOunTY_HUNTER_TRACKING_BONUS", hunterWallet, fmt.Sprintf("Tracking speed multiplier: %.2f (tier bonus)", trackingBonus))
					if cid := l.getClientIDFromLocked(hunterWallet); cid != "" {
						l.sendToClientLocked(cid, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🎯 <b>BOUNTY HUNTER TRACKING BONUS:</b> %dx tracking speed!"}`, trackingBonus))})
					}
				}
			}

			// P2-A: Rival Pair hook — Bounty Hunter vs Criminal defenders
			if rivalXP, pair, isRival := EvaluateCrossCareerXP("Bounty Hunter", stats.JobRole, baseXP, &stats, l.leaderboard[match.P2Wallet]); isRival {
				stats.CareerXP["Bounty Hunter"] += rivalXP - baseXP
				l.logAdminAuditLocked("RIVAL_BOunTY_HUNTER_XP", hunterWallet, fmt.Sprintf("+%d bonus XP (rival pair: %s)", pair))
			}

			// P2-E1: Bounty Hunter ↔ Kidnapper (enemy pair) — TrackRivalInteraction caller
			// When bounty hunter captures a card belonging to a player with Kidnapper career, grant rival XP
			if match.P2Wallet != "" && l.leaderboard[match.P2Wallet] != nil {
				p2Stats := l.leaderboard[match.P2Wallet]
				if p2Stats.CareerXP != nil && (p2Stats.JobRole == "Kidnapper" || CareerHasRole(p2Stats.CareerXP, "Kidnapper")) {
					rivalXPDelta, rivalName, _ := TrackRivalInteraction("BountyHunter", "Kidnapper", baseXP, &stats, p2Stats)
					if rivalXPDelta > 0 {
						stats.CareerXP["Bounty Hunter"] += rivalXPDelta
						l.logAdminAuditLocked("RIVAL_BOUNTY_HUNTER_KIDNAPPER", hunterWallet, fmt.Sprintf("Tracking bonus +%d XP (rival: %s)", rivalXPDelta, rivalName))
					}
				}
			}
		}
	}


	// ============================================================================
	// UNDERWORLD CAREER XP TRIGGERS — Heist Planner
	// ============================================================================
	if history.WinnerID != "" {
		winnerWallet := history.Opponent
		if history.WinnerIndex == 0 {
			winnerWallet = match.P1Wallet
		} else if history.WinnerIndex == 1 {
			winnerWallet = match.P2Wallet
		}

		l.ensurePlayerStatsMapsInitialized(winnerWallet)
		wStats := l.leaderboard[winnerWallet]

		if wStats.CareerXP != nil {
			// Heist Planner: XP per battle win (tactical leadership in combat operations)
			if wStats.JobRole == "Heist Planner" || CareerHasRole(wStats.CareerXP, "Heist Planner") {
				baseHPXP := uint64(40)
				cardBonusXP := uint64(len(match.CapturedCards)) * 12
				hpXP := baseHPXP + cardBonusXP
				wStats.CareerXP.TrackCareerXP("Heist Planner", hpXP)

				// Evaluate cross-career rival pair: Heist Planner ↔ Warden (hostile)
				if rivalXP, _, isRival := EvaluateCrossCareerXP("Heist Planner", wStats.JobRole, hpXP, &wStats, nil); isRival {
					rivalTarget := "Warden"
					if wStats.JobRole == "Warden" {
						rivalTarget = wStats.JobRole
					}
					wStats.CareerXP.TrackCareerXP("Heist Planner", rivalXP-hpXP)
					l.logAdminAuditLocked("RIVAL_HEIST_PLANNER_XP", winnerWallet, fmt.Sprintf("+%d bonus XP (rival pair: %s↔%s)", rivalXP-hpXP, "Heist Planner", rivalTarget))
				}

				// Heist Planner ↔ Kidnapper team synergy (if both in same organization)
				if wStats.EmployerClubID != "" {
					for _, otherWallet := range l.getAllTeamMemberWalletsLocked(wStats.EmployerClubID) {
						if otherWallet == winnerWallet {
							continue
						}
						l.ensurePlayerStatsMapsInitialized(otherWallet)
						otherStats := l.leaderboard[otherWallet]
						if otherStats.JobRole == "Kidnapper" && (wStats.JobRole == "Heist Planner" || CareerHasRole(wStats.CareerXP, "Heist Planner")) {
							synergyXP := uint64(15)
							wStats.CareerXP.TrackCareerXP("Heist Planner", synergyXP)
							otherStats.CareerXP.TrackCareerXP("Kidnapper", synergyXP)
							l.logAdminAuditLocked("SYNERGY_HEIST_KIDNAP", winnerWallet, fmt.Sprintf("+%d XP team bonus (Kidnapper: %s)", synergyXP, otherWallet))
							l.leaderboard[otherWallet] = otherStats
						}
					}
				}

				// Check Heist Planner promotion milestone
				level := wStats.CareerLevel["Heist Planner"] + 1
				const hpXPPerLevel = 350
				for wStats.CareerXP.RoleXP["Heist Planner"] >= level*hpXPPerLevel && level <= CareerTierBoss {
					wStats.CareerLevel["Heist Planner"] = level
					if cid := l.getClientIDFromWalletLocked(winnerWallet); cid != "" {
						l.sendToClientLocked(cid, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"⭐ <b>HEIST PLANNER PROMOTION:</b> Reached level %d!"}`, level))})
					}
					level++
				}

				l.logAdminAuditLocked("CAREER_HEIST_PLANNER_XP", winnerWallet, fmt.Sprintf("+%d XP (captures: %d)", hpXP, len(match.CapturedCards)))
			}
		}
	}

	// ============================================================================
	// UNDERWORLD CAREER XP TRIGGERS — #3-10 (Smuggler, Hostage Host, Lawyer-Commissioner, Arc-Net Operative, Launderer)
	// NOTE: Kidnapper XP triggered in handlers_criminality.go; Fence XP in black_market_service.go
	// ============================================================================
	if history.WinnerID != "" {
		winnerWallet := history.Opponent
		if history.WinnerIndex == 0 {
			winnerWallet = match.P1Wallet
		} else if history.WinnerIndex == 1 {
			winnerWallet = match.P2Wallet
		}

		l.ensurePlayerStatsMapsInitialized(winnerWallet)
		wStats := l.leaderboard[winnerWallet]

		if wStats.CareerXP != nil {
			// Underworld Career #4: Hostage Host — XP per capture held (hoarding efficiency)
			if wStats.JobRole == "Hostage Host" || CareerHasRole(wStats.CareerXP, "Hostage Host") {
				hostageXP := uint64(len(match.CapturedCards)) * 40
				if hostageXP > 0 {
					wStats.CareerXP.TrackCareerXP("Hostage Host", hostageXP)
					l.logAdminAuditLocked("CAREER_HOSTAGE_HOST_XP", winnerWallet, fmt.Sprintf("+%d XP (captured cards held: %d)", hostageXP, len(match.CapturedCards)))

					// Check for Hostage Host promotion milestone (4 tiers: Hoarder → Collector → Curator → Archivist)
					level := wStats.CareerLevel["Hostage Host"] + 1
					const hhXPPerLevel = 350
					for wStats.CareerXP.RoleXP["Hostage Host"] >= level*hhXPPerLevel && level <= CareerTierBoss {
						wStats.CareerLevel["Hostage Host"] = level
						if cid := l.getClientIDFromWalletLocked(winnerWallet); cid != "" {
							l.sendToClientLocked(cid, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"⭐ <b>HOSTAGE HOST PROMOTION:</b> Reached level %d!"}`, level))})
						}
						level++
					}
				}
			}

			// Underworld Career #5: Lawyer-Commissioner — XP per match (legal oversight bonus)
			if wStats.JobRole == "Lawyer-Commissioner" || CareerHasRole(wStats.CareerXP, "Lawyer-Commissioner") {
				baseLC := uint64(25)
				wStats.CareerXP.TrackCareerXP("Lawyer-Commissioner", baseLC)

				// Check for Lawyer-Commissioner promotion milestone (4 tiers: Associate → Partner → Senior → Commissioner)
				level := wStats.CareerLevel["Lawyer-Commissioner"] + 1
				const lcXPPerLevel = 300
				for wStats.CareerXP.RoleXP["Lawyer-Commissioner"] >= level*lcXPPerLevel && level <= CareerTierBoss {
					wStats.CareerLevel["Lawyer-Commissioner"] = level
					if cid := l.getClientIDFromWalletLocked(winnerWallet); cid != "" {
						l.sendToClientLocked(cid, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"⭐ <b>LAWYER-COMMISSIONER PROMOTION:</b> Reached level %d!"}`, level))})
					}
					level++
				}

				l.logAdminAuditLocked("CAREER_LAWYER_COMMISSIONER_XP", winnerWallet, fmt.Sprintf("+%d XP (legal oversight)", baseLC))
			}

			// Underworld Career #7: Arc-Net Operative — XP per cyber-deploy in battle (signal ops expertise)
			if wStats.JobRole == "Arc-Net Operative" || CareerHasRole(wStats.CareerXP, "Arc-Net Operative") {
				cyberDeployBonus := uint64(len(match.CapturedCards)) * 20
				baseANXP := uint64(20) + cyberDeployBonus
				wStats.CareerXP.TrackCareerXP("Arc-Net Operative", baseANXP)

				// Check for Arc-Net Operative promotion milestone (4 tiers: Novice → Analyst → Operative → Director)
				level := wStats.CareerLevel["Arc-Net Operative"] + 1
				const anXPPerLevel = 320
				for wStats.CareerXP.RoleXP["Arc-Net Operative"] >= level*anXPPerLevel && level <= CareerTierBoss {
					wStats.CareerLevel["Arc-Net Operative"] = level
					if cid := l.getClientIDFromWalletLocked(winnerWallet); cid != "" {
						l.sendToClientLocked(cid, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"⭐ <b>ARC-NET OPERATIVE PROMOTION:</b> Reached level %d!"}`, level))})
					}
					level++
				}

				l.logAdminAuditLocked("CAREER_ARC_NET_OPERATIVE_XP", winnerWallet, fmt.Sprintf("+%d XP (cyber-deploys: %d)", baseANXP, len(match.CapturedCards)))
			}

			// Underworld Career #8: Smuggler — XP per match won while on duty (cross-sector transit bonus)
			if wStats.JobRole == "Smuggler" || CareerHasRole(wStats.CareerXP, "Smuggler") {
				baseSMXP := uint64(35)

				// P2-Underworld Mechanic Hook: Smuggler transit tax exemption reduces economy transfer tax
				taxExemption := wStats.CareerXP.GetTransitTaxExemption()
				if taxExemption > 0.0 {
					l.logAdminAuditLocked("CAREER_SMUGGLER_TRANSIT_TAX", winnerWallet, fmt.Sprintf("+%d XP, transit tax exemption: %.0f%% (tier bonus)", baseSMXP, taxExemption*100))
				}

				wStats.CareerXP.TrackCareerXP("Smuggler", baseSMXP)

				// Check for Smuggler promotion milestone (4 tiers: Runner → Transporter → Captain → Consortium)
				level := wStats.CareerLevel["Smuggler"] + 1
				const smXPPerLevel = 300
				for wStats.CareerXP.RoleXP["Smuggler"] >= level*smXPPerLevel && level <= CareerTierBoss {
					wStats.CareerLevel["Smuggler"] = level
					if cid := l.getClientIDFromWalletLocked(winnerWallet); cid != "" {
						l.sendToClientLocked(cid, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"⭐ <b>SMUGGLER PROMOTION:</b> Reached level %d!"}`, level))})
					}
					level++
				}

				l.logAdminAuditLocked("CAREER_SMUGGLER_XP", winnerWallet, fmt.Sprintf("+%d XP (transit bonus)", baseSMXP))
			}

			// Underworld Career #10: Launderer — XP per match won (financial cleaning via battle winnings)
			if wStats.JobRole == "Launderer" || CareerHasRole(wStats.CareerXP, "Launderer") {
				baseLA := uint64(45)
				// Additional XP proportional to battle wagers processed through cleaning
				wagerCleanXP := uint64(float64(match.WagersMicro) * 0.001 / 1000000.0)
				if wagerCleanXP < 5 {
					wagerCleanXP = 5
				}
				totalLA := baseLA + wagerCleanXP
				wStats.CareerXP.TrackCareerXP("Launderer", totalLA)

				// Check for Launderer promotion milestone (4 tiers: Cleaner → Processor → Director → Shadow Bank)
				level := wStats.CareerLevel["Launderer"] + 1
				const laXPPerLevel = 340
				for wStats.CareerXP.RoleXP["Launderer"] >= level*laXPPerLevel && level <= CareerTierBoss {
					wStats.CareerLevel["Launderer"] = level
					if cid := l.getClientIDFromWalletLocked(winnerWallet); cid != "" {
						l.sendToClientLocked(cid, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"⭐ <b>LAUNDERER PROMOTION:</b> Reached level %d!"}`, level))})
					}
					level++
				}

				l.logAdminAuditLocked("CAREER_LAUNDERER_XP", winnerWallet, fmt.Sprintf("+%d XP (wagers cleaned: %d)", totalLA, wagerCleanXP))
			}
		}
	}

			// ============================================================================
			// UNDERWORLD CAREER XP TRIGGERS — Heist Planner
			// ============================================================================

			// P2-Underworld Mechanic Hook: Heist Planner team planning buff applied to capture bonus
			if wStats.JobRole == "Heist Planner" || CareerHasRole(wStats.CareerXP, "Heist Planner") {
				heistPlanningBuff := wStats.CareerXP.GetPlanningBuff()
				if heistPlanningBuff > 1.0 {
					l.logAdminAuditLocked("CAREER_HEIST_PLANNER_BUFF", winnerWallet, fmt.Sprintf("Team planning buff: %.2fx capture bonus (tier bonus)", heistPlanningBuff))
				}
			}

			// P2-Underworld Mechanic Hook: Kidnapper success multiplier applied to capture outcome
	if history.WinnerID != "" {
		winnerWallet := history.Opponent
		if history.WinnerIndex == 0 {
			winnerWallet = match.P1Wallet
		} else if history.WinnerIndex == 1 {
			winnerWallet = match.P2Wallet
		}

		l.ensurePlayerStatsMapsInitialized(winnerWallet)
		wStats := l.leaderboard[winnerWallet]

		if wStats.CareerXP != nil {
			// Warden: XP per capture (monitoring efficiency)
			if wStats.JobRole == "Warden" || CareerHasRole(wStats.CareerXP, "Warden") {
				captureXP := uint64(len(match.CapturedCards)) * 25
				if captureXP > 0 {
					wStats.CareerXP.TrackCareerXP("Warden", captureXP)

					// P2-D3 Mechanic Hook: Warden detention duration multiplier applied to captures
					detentionBonus := wStats.CareerXP.GetWardenDetentionBonus()
					if detentionBonus > 1.0 {
						l.logAdminAuditLocked("CAREER_WARDEN_DETENTION_BONUS", winnerWallet, fmt.Sprintf("+%d XP, detention duration multiplier: %.2fx (tier bonus)", captureXP, detentionBonus))
						if cid := l.getClientIDFromWalletLocked(winnerWallet); cid != "" {
							l.sendToClientLocked(cid, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"⚔️ <b>WARDEN DETENTION BONUS:</b> %dx detention duration on captured cards!"}`, int(detentionBonus*100)/100))})
						}
					}

					l.logAdminAuditLocked("CAREER_WARDEN_XP", winnerWallet, fmt.Sprintf("+%d XP (captures: %d)", captureXP, len(match.CapturedCards)))
				}
			}

			// AOS Leader: XP per successful raid (combat participation + capture efficiency)
			if wStats.JobRole == "AOS Leader" || CareerHasRole(wStats.CareerXP, "AOS Leader") {
				baseAOSXP := uint64(80)
				ratioBonus := uint64(float64(len(match.CapturedCards)) * 20.0)
				aosXP := baseAOSXP + ratioBonus
				wStats.CareerXP.TrackCareerXP("AOS Leader", aosXP)

				// Check for AOS promotion milestone
				level := wStats.CareerLevel["AOS Leader"] + 1
				const xpPerLevel = 400
				for wStats.CareerXP.RoleXP["AOS Leader"] >= level*xpPerLevel && level <= CareerTierBoss {
					wStats.CareerLevel["AOS Leader"] = level
					if cid := l.getClientIDFromWalletLocked(winnerWallet); cid != "" {
						l.sendToClientLocked(cid, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"⭐ <b>AOS PROMOTION:</b> Reached level %d!"}`, level))})
					}
					level++
				}

				l.logAdminAuditLocked("CAREER_AOS_XP", winnerWallet, fmt.Sprintf("+%d XP (captures: %d)", aosXP, len(match.CapturedCards)))
			}

			// ForensicAnalyst: XP per capture analysis (evidence gathering efficiency)
			if wStats.JobRole == "ForensicAnalyst" || CareerHasRole(wStats.CareerXP, "ForensicAnalyst") {
				baseFA := uint64(35)
				evidenceBonus := uint64(len(match.CapturedCards)) * 18
				faXP := baseFA + evidenceBonus

				// P2-D5 Mechanic Hook: Forensic Analyst evidence accuracy multiplier
				evidenceAccuracy := wStats.CareerXP.GetEvidenceAccuracyBonus()
				if evidenceAccuracy > 1.0 {
					faXP = uint64(float64(faXP) * evidenceAccuracy)
					l.logAdminAuditLocked("CAREER_FORENSIC_ANALYST_EVIDENCE_BONUS", winnerWallet, fmt.Sprintf("+%d XP (evidence accuracy: %.2fx multiplier)", faXP, evidenceAccuracy))
				}
				level := wStats.CareerLevel["ForensicAnalyst"] + 1
				const faXPPerLevel = 320
				for wStats.CareerXP.RoleXP["ForensicAnalyst"] >= level*faXPPerLevel && level <= CareerTierBoss {
					wStats.CareerLevel["ForensicAnalyst"] = level
					if cid := l.getClientIDFromWalletLocked(winnerWallet); cid != "" {
						l.sendToClientLocked(cid, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"⭐ <b>FORENSIC ANALYST PROMOTION:</b> Reached level %d!"}`, level))})
					}
					level++
				}

				l.logAdminAuditLocked("CAREER_FORENSIC_ANALYST_XP", winnerWallet, fmt.Sprintf("+%d XP (captures: %d)", faXP, len(match.CapturedCards)))
			}

			// Sector Peacekeeper: XP per match + bonus for each opponent card captured
			if wStats.JobRole == "Sector Peacekeeper" || CareerHasRole(wStats.CareerXP, "Sector Peacekeeper") {
				basePKXP := uint64(50)
				pkBonusXP := uint64(len(match.CapturedCards)) * 15
				pkXP := basePKXP + pkBonusXP
				wStats.CareerXP.TrackCareerXP("Sector Peacekeeper", pkXP)

				// Check for Sector Peacekeeper promotion milestone
				level := wStats.CareerLevel["Sector Peacekeeper"] + 1
				const pkXPPerLevel = 350
				for wStats.CareerXP.RoleXP["Sector Peacekeeper"] >= level*pkXPPerLevel && level <= CareerTierBoss {
					wStats.CareerLevel["Sector Peacekeeper"] = level
					if cid := l.getClientIDFromWalletLocked(winnerWallet); cid != "" {
						l.sendToClientLocked(cid, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"⭐ <b>SECTOR PEACEKEEPER PROMOTION:</b> Reached level %d!"}`, level))})
					}
					level++
				}

				l.logAdminAuditLocked("CAREER_SECTOR_PEACEKEEPER_XP", winnerWallet, fmt.Sprintf("+%d XP (captures: %d)", pkXP, len(match.CapturedCards)))
			}

			// ============================================================================
			// JUSTICE CAREERS D7-D10 — Placeholder Hooks
			// These are stub implementations ready for expansion.
			// ============================================================================

			// D7: Circuit Judge — XP per match resolved (judicial oversight)
			if wStats.JobRole == "Circuit Judge" || CareerHasRole(wStats.CareerXP, "Circuit Judge") {
				baseJXP := uint64(45)
				wStats.CareerXP.TrackCareerXP("Circuit Judge", baseJXP)

				// Check for Circuit Judge promotion milestone
				level := wStats.CareerLevel["Circuit Judge"] + 1
				const jXPPerLevel = 380
				for wStats.CareerXP.RoleXP["Circuit Judge"] >= level*jXPPerLevel && level <= CareerTierBoss {
					wStats.CareerLevel["Circuit Judge"] = level
					if cid := l.getClientIDFromWalletLocked(winnerWallet); cid != "" {
						l.sendToClientLocked(cid, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"⭐ <b>CIRCUIT JUDGE PROMOTION:</b> Reached level %d!"}`, level))})
					}
					level++
				}

				l.logAdminAuditLocked("CAREER_CIRCUIT_JUDGE_XP", winnerWallet, fmt.Sprintf("+%d XP (match resolved)", baseJXP))
			}

			// D8: Magistrate — XP per territory control (administrative jurisdiction)
			if wStats.JobRole == "Magistrate" || CareerHasRole(wStats.CareerXP, "Magistrate") {
				baseMXP := uint64(50)
				wStats.CareerXP.TrackCareerXP("Magistrate", baseMXP)

				// Check for Magistrate promotion milestone
				level := wStats.CareerLevel["Magistrate"] + 1
				const mXPPerLevel = 350
				for wStats.CareerXP.RoleXP["Magistrate"] >= level*mXPPerLevel && level <= CareerTierBoss {
					wStats.CareerLevel["Magistrate"] = level
					if cid := l.getClientIDFromWalletLocked(winnerWallet); cid != "" {
						l.sendToClientLocked(cid, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"⭐ <b>MAGISTRATE PROMOTION:</b> Reached level %d!"}`, level))})
					}
					level++
				}

				l.logAdminAuditLocked("CAREER_MAGISTRATE_XP", winnerWallet, fmt.Sprintf("+%d XP (territory administered)", baseMXP))
			}

			// D9: Compliance Auditor — XP per regulatory audit completion
			if wStats.JobRole == "ComplianceAuditor" || CareerHasRole(wStats.CareerXP, "ComplianceAuditor") {
				baseCA := uint64(40)
				wStats.CareerXP.TrackCareerXP("ComplianceAuditor", baseCA)

				// Check for Compliance Auditor promotion milestone
				level := wStats.CareerLevel["ComplianceAuditor"] + 1
				const caXPPerLevel = 360
				for wStats.CareerXP.RoleXP["ComplianceAuditor"] >= level*caXPPerLevel && level <= CareerTierBoss {
					wStats.CareerLevel["ComplianceAuditor"] = level
					if cid := l.getClientIDFromWalletLocked(winnerWallet); cid != "" {
						l.sendToClientLocked(cid, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"⭐ <b>COMPLIANCE AUDITOR PROMOTION:</b> Reached level %d!"}`, level))})
					}
					level++
				}

				l.logAdminAuditLocked("CAREER_COMPLIANCE_AUDITOR_XP", winnerWallet, fmt.Sprintf("+%d XP (audit completed)", baseCA))
			}

			// D10: Ethics Overseer — XP per compliance verification
			if wStats.JobRole == "EthicsOverseer" || CareerHasRole(wStats.CareerXP, "EthicsOverseer") {
				baseEO := uint64(35)
				wStats.CareerXP.TrackCareerXP("EthicsOverseer", baseEO)

				// Check for Ethics Overseer promotion milestone
				level := wStats.CareerLevel["EthicsOverseer"] + 1
				const eoXPPerLevel = 340
				for wStats.CareerXP.RoleXP["EthicsOverseer"] >= level*eoXPPerLevel && level <= CareerTierBoss {
					wStats.CareerLevel["EthicsOverseer"] = level
					if cid := l.getClientIDFromWalletLocked(winnerWallet); cid != "" {
						l.sendToClientLocked(cid, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"⭐ <b>ETHICS OVERSEER PROMOTION:</b> Reached level %d!"}`, level))})
					}
					level++
				}

				l.logAdminAuditLocked("CAREER_ETHICS_OVERSEER_XP", winnerWallet, fmt.Sprintf("+%d XP (ethics verified)", baseEO))
			}
		}
	}

	// ============================================================================
	// UNDERWORLD CAREER XP TRIGGERS — TaxAuditor (P2-A tax collection)
	// Note: TaxAuditor XP is triggered in handlers_criminality.go via tax events.
	// This battle_section provides a baseline "patrol XP" for matches won while on duty.
	// ============================================================================
	if history.WinnerID != "" {
		taxAuditorWallet := history.Opponent
		if history.WinnerIndex == 0 {
			taxAuditorWallet = match.P1Wallet
		} else if history.WinnerIndex == 1 {
			taxAuditorWallet = match.P2Wallet
		}

		l.ensurePlayerStatsMapsInitialized(taxAuditorWallet)
		taxStats := l.leaderboard[taxAuditorWallet]

		if taxStats.CareerXP != nil {
			// TaxAuditor: XP per patrol match (administrative enforcement presence)
			if taxStats.JobRole == "TaxAuditor" || CareerHasRole(taxStats.CareerXP, "TaxAuditor") {
				baseTA := uint64(30) // Base patrol XP
				taxStats.CareerXP.TrackCareerXP("TaxAuditor", baseTA)

				// Check for TaxAuditor promotion milestone (5 tiers: Junior → Auditor → Senior → Chief → Commissioner)
				level := taxStats.CareerLevel["TaxAuditor"] + 1
				const taXPPerLevel = 300
				for taxStats.CareerXP.RoleXP["TaxAuditor"] >= level*taXPPerLevel && level <= CareerTierBoss {
					taxStats.CareerLevel["TaxAuditor"] = level
					if cid := l.getClientIDFromWalletLocked(taxAuditorWallet); cid != "" {
						l.sendToClientLocked(cid, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"⭐ <b>TAX AUDITOR PROMOTION:</b> Reached level %d!"}`, level))})
					}
					level++
				}

				l.logAdminAuditLocked("CAREER_TAX_AUDITOR_XP", taxAuditorWallet, fmt.Sprintf("+%d XP (patrol match won)", baseTA))
			}
		}
	}

	// PILLAR 1: Industrial Loop (Winning Side Distribution).
	// Deduct 2% "House Cut" and distribute the remaining pool to the winning player.
	if match.WagersMicro > 0 && l.tokenSinkRouter != nil {
		siphonAmtMicro := (match.WagersMicro * 2) / 100
		matrix := RevenueSplitMatrix{FaucetShare: 1.0, ClubShare: 0.0, GovernanceShare: 0.0}
		
		_ = l.tokenSinkRouter.RouteCriminalTax("SPECTATOR_SIPHON", siphonAmtMicro, matrix, 0, "")

		// Distribute net pool to winner if not a draw
		if p1 != p2 {
			winnerWallet := match.P1Wallet
			if p2 > p1 {
				winnerWallet = match.P2Wallet
			}
			
			if winnerWallet != "" {
				netPayoutMicro := match.WagersMicro - siphonAmtMicro
				l.playerBalances[winnerWallet] += netPayoutMicro
				l.logAdminAuditLocked("SPECTATOR_WAGERS_SETTLED", winnerWallet, fmt.Sprintf("Winner: %s, Payout: %d, Siphon: %d", winnerWallet, netPayoutMicro, siphonAmtMicro))
				l.applyDynamicScalingLocked()
			}
		} else {
			l.logAdminAuditLocked("SPECTATOR_SIPHON_COLLECTED", "faucet", fmt.Sprintf("Amt: %d from pool %d (Draw)", siphonAmtMicro, match.WagersMicro))
		}

		l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0
	}

	// PILLAR 3: Economic Consequence.
	// Increment fatigue for all cards played to the board during this match.
	l.processMatchFatigueLocked(match)

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

// processMatchFatigueLocked increases the wear-and-tear of cards used in combat.
func (l *Lobby) processMatchFatigueLocked(match *MatchState) {
	for _, card := range match.Board {
		if card == nil {
			continue
		}

		card.Fatigue += 5 // Fixed fatigue cost per match deployment
		if card.Fatigue > 100 {
			card.Fatigue = 100
		}

		// INDUSTRIAL LOOP: Persist fatigue to global and persistent cache.
		// This state is archived on-chain via periodic VBT_CARD_CACHE_SNAPSHOT notes.
		l.inventory[card.ID] = *card
		l.persistentCardCache[card.ID] = *card
	}
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

	// PILLAR 3: Metadata Sync for Battle Scars.
	// Re-fetch updated metadata for all cards in the new decks to ensure
	// clients receive the persisted Fallen_penalty Artifact reductions.
	// redistributed hands and authoritative metadata (Artifact scars) into WASM.
	handMetadata := make(map[int]ServerCard)
	for _, id := range append(p1NewDeck, p2NewDeck...) {
		// PILLAR 4: Sudden Death Integrity.
		// Reset Fatigue for redistributed hands so the tie-breaker starts at peak power.
		// This must be persisted to l.inventory to ensure match archival is accurate.
		if c, exists := l.inventory[id]; exists {
			c.Fatigue = 0
			l.inventory[id] = c
			l.persistentCardCache[id] = c
			handMetadata[id] = c
		}
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"text":              "⚔️ <b>SUDDEN DEATH!</b> The board has cleared. Decks have been redistributed based on card ownership.",
		"p1_deck":           p1NewDeck,
		"p2_deck":           p2NewDeck,
		"card_metadata":     handMetadata,          // Sync persisted scars to client WASM/UI
		"active_item_buffs": match.ActiveItemBuffs, // Sync active buffs to client UI
		"rules":             match.Rules,           // Sync authoritative rules
	})

	l.sendToClientLocked(match.P1ID, Envelope{Type: "sudden_death_start", FromID: "SERVER", Payload: payload})
	l.sendToClientLocked(match.P2ID, Envelope{Type: "sudden_death_start", FromID: "SERVER", Payload: payload})
	log.Printf("[BATTLE] Sudden Death tie-breaker initiated for match %s vs %s\n", match.P1ID, match.P2ID)
}

func (l *Lobby) finalizeMatchResultLocked(match *MatchState, history MatchHistory) {
	winnerID := history.WinnerID
	l.matchHistory[winnerID] = history

	// Determine participants
	p1Wallet, p1Exists := l.wallets[match.P1ID]
	p2Wallet, p2Exists := l.wallets[match.P2ID]

	// PILLAR 4: Deterministic Rating Update.
	// Ensure both players have their BestRating updated regardless of the match outcome.
	if p1Exists {
		l.updatePlayerRatingLocked(p1Wallet, match.P1Deck)
	}
	if p2Exists {
		l.updatePlayerRatingLocked(p2Wallet, match.P2Deck)
	}

	if wallet, ok := l.wallets[winnerID]; ok {
		// PILLAR 4: Historical Immersion. Update winner's ephemeral history for immediate UI refresh.
		// This ensures standard and bracket victories appear instantly in the "Recent Victories" panel.
		l.ensurePlayerStatsMapsInitialized(wallet)
		stats := l.leaderboard[wallet]
		winnerRecord := history
		winnerRecord.WinnerIndex = 0 // Relative win for this record
		stats.History = append([]MatchHistory{winnerRecord}, stats.History...)
		if len(stats.History) > 15 {
			stats.History = stats.History[:15]
		}
		l.leaderboard[wallet] = stats

		// Determine winner's deck for tendencies update
		winnerDeck := match.P1Deck
		if winnerID == match.P2ID { winnerDeck = match.P2Deck }

		l.updateLeaderboard(wallet, history.TournamentMatchID != "", history.Scores, winnerDeck, history.IsBountyMatch)
		s := l.leaderboard[wallet]

		// Achievement: Arena Legend (100 Wins)
		if s.Wins >= 100 {
			l.achievementService.UnlockAchievementLocked(l, wallet, "ARENA_LEGEND")
			s = l.leaderboard[wallet] // Refresh after achievement reputation bump
		}

		if history.TournamentMatchID != "" {
			l.processTournamentResult(history.TournamentMatchID, wallet)
		}

		// PILLAR 7: 3-Win Challenge Tracking.
		// If the winner is currently in a recovery challenge for a specific card, increment wins.
		if stats.RecoveryChallengeCardID != 0 {
			stats.RecoveryChallengeWins++
			if stats.RecoveryChallengeWins >= 3 {
				// Success: Restore the card to inventory.
				cardKey := fmt.Sprintf("CARD-%d", stats.RecoveryChallengeCardID)
				if card, exists := l.inventory[stats.RecoveryChallengeCardID]; exists {
					// PILLAR 7: Asset Liberation. Purify the archetype.
					card.Fallen = false
					l.inventory[stats.RecoveryChallengeCardID] = card
					l.persistentCardCache[stats.RecoveryChallengeCardID] = card
				}

				stats.Inventory[cardKey]++
				
				l.sendToClientLocked(winnerID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"✨ <b>ASSET LIBERATED:</b> 3-win streak complete. Card restored to inventory."}`)})
				
				// PILLAR 3: Underworld Contract Completion (CONTRACT-028)
				if stats.ActiveUnderworldContractID == "CONTRACT-028" && stats.JobRole == "Launderer" {
					const rewardMicro = 15000 * 1000000
					l.playerBalances[wallet] += rewardMicro
					stats.ActiveUnderworldContractID = ""
					l.logAdminAuditLocked("CONTRACT_COMPLETED", wallet, "ID: CONTRACT-028, Payout: 15000.00")
					l.sendToClientLocked(winnerID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"💰 <b>CONTRACT COMPLETED:</b> Asset liberated. Payout: 15,000.00 $VBV."}`)})
				}

				// PILLAR 7: Recovery Bounty Return.
				if stats.RecoveryBounties != nil {
					if bounty, exists := stats.RecoveryBounties[stats.RecoveryChallengeCardID]; exists {
						l.playerBalances[wallet] += bounty
						delete(stats.RecoveryBounties, stats.RecoveryChallengeCardID)
						l.sendToClientLocked(winnerID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🏴‍☠️ <b>BOUNTY RECLAIMED:</b> %.2f $VBV recovery reward released."}`, float64(bounty)/1000000.0))})
					}
				}

				stats.RecoveryChallengeCardID = 0
				stats.RecoveryChallengeWins = 0
			}
			l.leaderboard[wallet] = stats
		}
	}

	// PILLAR 4: Historical Immersion. Update loser's history for real-time feedback.
	// The loser wallet is derived from the 'Opponent' field of the winner's record.
	loserWallet := history.Opponent
	if loserWallet != "" && loserWallet != "DRAW" && loserWallet != "BYE" {
		l.ensurePlayerStatsMapsInitialized(loserWallet)
		lStats := l.leaderboard[loserWallet]

		// Mirrored entry for loser: Winner becomes the opponent, and record is a Loss.
		loserRecord := history
		if wWallet, ok := l.wallets[winnerID]; ok {
			loserRecord.Opponent = wWallet
		}

		// PILLAR 4: Historical Immersion.
		// Mirrored logic: The loser's record always reflects a relative Loss (1) in their personal history.
		loserRecord.WinnerIndex = 1

		lStats.History = append([]MatchHistory{loserRecord}, lStats.History...)
		if len(lStats.History) > 15 {
			lStats.History = lStats.History[:15]
		}

		// PILLAR 7: Challenge Streak Reset.
		// Losing a match during a recovery challenge wipes all progress.
		if lStats.RecoveryChallengeCardID != 0 {
			lStats.RecoveryChallengeWins = 0
			l.sendToClientLocked(l.getClientIDFromWalletLocked(loserWallet), Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"💀 <b>CHALLENGE FAILED:</b> Match lost. Retrieval streak reset to 0."}`)})
		}

		l.leaderboard[loserWallet] = lStats
	}
}

// updatePlayerRatingLocked calculates the deck rating and updates BestRating if improved.
func (l *Lobby) updatePlayerRatingLocked(wallet string, deck []int) {
	if len(deck) == 0 { return }
	rating := l.calculateDeckRatingLocked(deck)
	s := l.leaderboard[wallet]
	if l.isBetterRating(rating, s.BestRating) {
		s.BestRating = rating
		l.leaderboard[wallet] = s
	}
}

func (l *Lobby) calculateDeckRating(cardIDs []int) string {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.calculateDeckRatingLocked(cardIDs)
}

// calculateDeckRatingLocked is the internal implementation that assumes the lock is held.
func (l *Lobby) calculateDeckRatingLocked(cardIDs []int) string {
	if len(cardIDs) == 0 {
		return "[Z]"
	}
	maxBin := -1
	for _, id := range cardIDs {
		card := l.inventory[id]
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
		card := l.inventory[id]
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
	stats.Reputation = l.CalculateReputation(stats) // Ensure reputation is updated
	l.leaderboard[wallet] = stats
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

	// PILLAR 3: Underworld Contract Completion.
	// Check if the winner has an active contract to jail an asset.
	winnerStats := l.leaderboard[winnerWallet]
	if winnerStats.ActiveUnderworldContractID == "CONTRACT-003" {
		const rewardMicro = 1000 * 1000000
		l.playerBalances[winnerWallet] += rewardMicro
		winnerStats.ActiveUnderworldContractID = ""
		l.logAdminAuditLocked("CONTRACT_COMPLETED", winnerWallet, "ID: CONTRACT-003, Payout: 1000.00")
		if cid := l.getClientIDFromWalletLocked(winnerWallet); cid != "" {
			l.sendToClientLocked(cid, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> Asset incarcerated. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
		}
		l.leaderboard[winnerWallet] = winnerStats
	}

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
		card.Fallen = true // PILLAR 7: Seized assets enter the 'Fallen' state.

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

		// PILLAR 1: Mojo Gain for Capturing Player's Club
		mojoGain := l.clubService.CalculateMojoGain(l, owningClub, "JAIL_CAPTURE", 0)
		owningClub.Mojo += mojoGain
		l.achievementService.CheckMojoSurgeAchievementLocked(l, owningClub.ID)

		// PILLAR 3: Underworld Contract Completion.
		capturerWallet := strings.ToLower(captured.CapturingPlayerWallet)
		capturerStats := l.leaderboard[capturerWallet]
		if capturerStats.ActiveUnderworldContractID == "CONTRACT-003" {
			const rewardMicro = 1000 * 1000000
			l.playerBalances[capturerWallet] += rewardMicro
			capturerStats.ActiveUnderworldContractID = ""
			l.logAdminAuditLocked("CONTRACT_COMPLETED", capturerWallet, "ID: CONTRACT-003, Payout: 1000.00")
			if cid := l.getClientIDFromWalletLocked(capturerWallet); cid != "" {
				l.sendToClientLocked(cid, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CONTRACT COMPLETED:</b> Asset incarcerated. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
			}
			l.leaderboard[capturerWallet] = capturerStats
		}

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

		log.Printf("[FALLEN_PENALTY_JAIL] %s's card (%s) jailed by Club %s via %s capture in %s. Club gained %d Mojo.\n",
			captured.OriginalOwnerWallet, card.Name, owningClub.Name, captured.CaptureType, match.TerritoryID, mojoGain)

		// Use CaptureType in client notifications for high-fidelity tactical feedback
		l.sendToClientLocked(l.getClientIDFromWalletLocked(captured.OriginalOwnerWallet), Envelope{
			Type: "admin_notification",
			Payload: json.RawMessage(fmt.Sprintf(`{"text":"🚨 <b>FALLEN PENALTY:</b> Your card '%s' was seized via %s and jailed by Club %s!"}`,
				escapeHTML(card.Name), captured.CaptureType, escapeHTML(owningClub.Name))),
		})
		l.sendToClientLocked(l.getClientIDFromWalletLocked(captured.CapturingPlayerWallet), Envelope{
			Type: "admin_notification",
			Payload: json.RawMessage(fmt.Sprintf(`{"text":"⛓️ <b>FALLEN PENALTY:</b> You jailed '%s's card (%s) via %s capture!"}`,
				escapeHTML(captured.OriginalOwnerWallet), escapeHTML(card.Name), captured.CaptureType)),
		})
	}
}

// applyItemEffectToMatch applies the in-match effects of an item to the MatchState or a specific card.
// This function is called by lobby_manager.go's use_item handler.
func (l *Lobby) applyItemEffectToMatch(match *MatchState, playerID string, itemID string, _ int, _ int) {
	item, itemExists := GlobalShopRegistry[itemID]
	if !itemExists {
		log.Printf("[BATTLE] Attempted to apply unknown item effect: %s\n", itemID)
		return
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
			// PILLAR 4: Persistence Sync.
			if wallet, ok := l.wallets[playerID]; ok {
				l.ensurePlayerStatsMapsInitialized(wallet)
				s := l.leaderboard[wallet]
				s.ActiveItemBuffs[itemID] = 3
				l.leaderboard[wallet] = s
			}
		case "grounded_shield": // Immunity to Mood Penalties (5 Matches)
			// This would typically be a rule override. For now, we track it.
			match.ActiveItemBuffs[playerID][itemID] = 5 // Track for 5 matches
			log.Printf("[BATTLE] Player %s activated Grounded Shield. Immunity for 5 matches.\n", playerID)
			// PILLAR 4: Persistence Sync.
			if wallet, ok := l.wallets[playerID]; ok {
				l.ensurePlayerStatsMapsInitialized(wallet)
				s := l.leaderboard[wallet]
				s.ActiveItemBuffs[itemID] = 5
				l.leaderboard[wallet] = s
			}
		}

	case "Tactical":
		switch itemID {
		case "rule_breaker": // Force PLUS trigger (1 Match)
			// This is a temporary rule. Add it to match.Rules and track its duration.
			match.Rules["Force_Plus_Trigger"] = true
			match.ActiveItemBuffs[playerID][itemID] = 1 // Track for 1 match
			log.Printf("[BATTLE] Player %s activated Rule Breaker. Force Plus Trigger for 1 match.\n", playerID)
			// PILLAR 4: Persistence Sync.
			if wallet, ok := l.wallets[playerID]; ok {
				l.ensurePlayerStatsMapsInitialized(wallet)
				s := l.leaderboard[wallet]
				s.ActiveItemBuffs[itemID] = 1
				l.leaderboard[wallet] = s
			}
		case "intel_report": // See Opponent Hand (3 Matches)
			// This would require a flag in MatchState and UI logic to reveal opponent's hand.
			// For now, we just track it.
			match.ActiveItemBuffs[playerID][itemID] = 3 // Track for 3 matches
			log.Printf("[BATTLE] Player %s activated Intel Report. Opponent hand visible for 3 matches.\n", playerID)
			// PILLAR 4: Persistence Sync.
			if wallet, ok := l.wallets[playerID]; ok {
				l.ensurePlayerStatsMapsInitialized(wallet)
				s := l.leaderboard[wallet]
				s.ActiveItemBuffs[itemID] = 3
				l.leaderboard[wallet] = s
			}
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
			wallet, okWallet := l.wallets[playerID]
			var s PlayerStats

			if okWallet {
				l.ensurePlayerStatsMapsInitialized(wallet) // PILLAR 3: Persistence Hardening
				s = l.leaderboard[wallet]
			}

			for itemID, matchesRemaining := range playerBuffs {
				matchesRemaining-- // Decrement for the just-completed match

				// PILLAR 4: Persistence Sync.
				if okWallet {
					if matchesRemaining <= 0 {
						delete(s.ActiveItemBuffs, itemID)
					} else {
						s.ActiveItemBuffs[itemID] = matchesRemaining
					}
				}

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

			if okWallet {
				l.leaderboard[wallet] = s
			}

			if len(newPlayerBuffs) > 0 {
				match.ActiveItemBuffs[playerID] = newPlayerBuffs
			} else {
				delete(match.ActiveItemBuffs, playerID) // Remove player entry if no buffs remain
			}
		}
	}
}

// evaluateRivalThresholds checks if any career XP changes crossed rivalry thresholds and applies bonuses.
func (l *Lobby) evaluateRivalThresholds(wallet, career string, newXP uint64) {
	l.ensurePlayerStatsMapsInitialized(wallet)
	stats := l.leaderboard[wallet]

	// RIVAL DECAY: -1 level every 24 hours since last interaction
	now := time.Now().Unix()
	if stats.LastRivalCheck > 0 {
		hoursSinceLastCheck := float64(now - stats.LastRivalCheck)
		decays := int(hoursSinceLastCheck / 86400)
		if decays > 0 && stats.Rivalries != nil {
			for careerName, level := range stats.Rivalries {
				newLevel := level - decays
				if newLevel <= 0 {
					delete(stats.Rivalries, careerName)
				} else {
					stats.Rivalries[careerName] = newLevel
				}
			}
			stats.LastRivalCheck = now - (int64(decays) * 86400)
		}
	}

	// RIVAL THRESHOLD: +5 XP bonus if rivalry level reaches 3+ with enemy career
	if stats.Rivalries != nil {
		if rivalLevel, exists := stats.Rivalries[career]; exists && rivalLevel >= 3 {
			const thresholdBonus = 5
			stats.CareerXP.TrackCareerXP(career, uint64(thresholdBonus))
			l.sendToClientLocked(l.getClientIDFromWalletLocked(wallet), Envelope{
				Type: "admin_notification",
				Payload: json.RawMessage(fmt.Sprintf(`{"text":"⚔️ <b>RIVALRY MASTER:</b> Reached level %d vs %s! +%d XP bonus!"}`, rivalLevel, career, thresholdBonus)),
			})
		}
	}

	stats.LastRivalCheck = now
	l.leaderboard[wallet] = stats
}

// broadcastRivalryEvent sends a WebSocket event to both players about the rivalry interaction.
func (l *Lobby) broadcastRivalryEvent(player1, player2, career1, career2 string, rivalryLevel int, eventType string) {
	cid1 := l.getClientIDFromWalletLocked(player1)
	cid2 := l.getClientIDFromWalletLocked(player2)

	eventText := ""
	switch eventType {
	case "ENEMY_DETECTED":
		eventText = fmt.Sprintf(`{"text":"⚔️ <b>RIVALRY DETECTED:</b> A %s (rival level %d) has encountered a %s!"}`, career1, rivalryLevel, career2)
	case "ALLY_BONDING":
		eventText = fmt.Sprintf(`{"text":"🤝 <b>ALLIANCE STRENGTHENED:</b> Your %s team bonded with a %s (rivalry level %d)!"}`, career1, career2, rivalryLevel)
	case "XP_GAINED":
		eventText = fmt.Sprintf(`{"text":"+ %s ↔ %s: +%d XP synergy interaction (rivalry level %d)"}`, career1, career2, rivalryLevel, rivalryLevel)
	default:
		eventText = fmt.Sprintf(`{"text":"⚔️ <b>RIVALRY EVENT:</b> Level %d - %s vs %s"}`, rivalryLevel, career1, career2)
	}

	payload := json.RawMessage(eventText)
	if cid1 != "" {
		l.sendToClientLocked(cid1, Envelope{Type: "rivalry_update", Payload: payload})
	}
	if cid2 != "" {
		l.sendToClientLocked(cid2, Envelope{Type: "rivalry_update", Payload: payload})
	}
}

// applyMutationScars permanently reduces a card's Artifact value due to a mutation failure.
// PILLAR 6: Specialized Gene-Editing.
func (l *Lobby) applyMutationScars(cardID int, reduction int) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.applyMutationScarsLocked(cardID, reduction)
}

// applyMutationScarsLocked handles Artifact reduction without acquiring the lock.
// PILLAR 6: Specialized Gene-Editing.
func (l *Lobby) applyMutationScarsLocked(cardID int, reduction int) error {
	card, exists := l.inventory[cardID]
	if !exists {
		return fmt.Errorf("card %d not found in inventory", cardID)
	}

	card.Artifact -= reduction
	if card.Artifact < 0 {
		card.Artifact = 0 // Artifact cannot go below zero
	}
	l.inventory[cardID] = card
	l.persistentCardCache[cardID] = card // Persist the scar to the blockchain snapshot

	// PILLAR 6: Forensic Audit. Identify owner to record the scar in their personal history.
	var ownerWallet string
	for wallet, s := range l.leaderboard {
		if _, has := s.Inventory[fmt.Sprintf("CARD-%d", cardID)]; has {
			ownerWallet = wallet
			break
		}
	}
	if ownerWallet != "" {
		s := l.leaderboard[ownerWallet]
		s.MutationHistory = append(s.MutationHistory, MutationEvent{
			Timestamp: time.Now().Unix(),
			Type:      "SCAR",
			CardID:    cardID,
			Details:   fmt.Sprintf("Mutation Scar: -%d Artifact points", reduction),
		})
		l.leaderboard[ownerWallet] = s
	}

	l.logAdminAuditLocked("MUTATION_SCAR", fmt.Sprintf("CARD-%d", cardID), fmt.Sprintf("Artifact reduced by %d. New Artifact: %d", reduction, card.Artifact))
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }() // Trigger UI update
	return nil
}
