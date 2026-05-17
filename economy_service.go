//go:build !js || !wasm

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/mnemonic"
	"github.com/algorand/go-algorand-sdk/v2/transaction"
	"github.com/algorand/go-algorand-sdk/v2/types"
)

func (l *Lobby) applyDynamicScaling() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.applyDynamicScalingLocked()
}

// applyDynamicScalingLocked contains the core logic for reward scaling and assumes the mutex is held.
func (l *Lobby) applyDynamicScalingLocked() {
	// Scaling is based on how full the faucet is relative to its target maximum
	if l.maxFaucetCapacity <= 0 {
		return
	}
	ratio := l.faucetBalance / l.maxFaucetCapacity
	if ratio > 1.0 {
		ratio = 1.0
	}
	if ratio < 0.1 {
		ratio = 0.1
	}

	// 1. Scale the primary base reward (for internal tracking/legacy logic)
	l.baseReward = uint64(float64(l.initialBaseReward) * ratio)

	// 2. Iterate through the entire reward stack and scale based on unscaled initial values
	for assetID, initialAmt := range l.initialRewards {
		scaledAmt := uint64(float64(initialAmt) * ratio)
		l.rewardStack[assetID] = scaledAmt
	}

	log.Printf("[ECONOMY] Dynamic Scaling Applied (Ratio: %.2f). Faucet Capacity: %.2f units.\n", ratio, l.faucetBalance)
}

// saveSeasonMetadataLocked persists the current season state and reward configuration to disk.
// This function assumes the Lobby mutex is already held.
func (l *Lobby) saveSeasonMetadataLocked() {
	data := map[string]interface{}{
		"start":           l.seasonStart,
		"num":             l.seasonNumber,
		"initial_rewards": l.initialRewards,
	}
	conf, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("[ECONOMY ERROR] Failed to marshal season metadata: %v\n", err)
		return
	}
	os.WriteFile(l.getDataPath("season.json"), conf, 0644)
}

func (l *Lobby) sendNoteTx(note string) (string, error) {
	l.mutex.RLock()
	voiConfig, _ := l.availableNetworks["Voi Mainnet"]
	l.mutex.RUnlock()
	client, _ := algod.MakeClient(voiConfig.NodeURL, "")
	pk, _ := mnemonic.ToPrivateKey(os.Getenv("FAUCET_MNEMONIC"))
	faucetAccount, _ := crypto.AccountFromPrivateKey(pk)
	sp, _ := client.SuggestedParams().Do(context.Background())
	senderAddr, _ := types.DecodeAddress(l.vaultAddress)

	appID, _ := strconv.ParseUint(voiConfig.AppID, 10, 64)
	txn, _ := transaction.MakeApplicationNoOpTx(appID, nil, nil, nil, nil, sp, senderAddr, []byte(note), types.Digest{}, [32]byte{}, types.Address{})
	_, stxn, _ := crypto.SignTransaction(faucetAccount.PrivateKey, txn)
	txid, err := client.SendRawTransaction(stxn).Do(context.Background())
	if err != nil {
		log.Printf("[ECONOMY ERROR] sendNoteTx failed: %v\n", err)
		return "", err
	}
	return txid, nil
}

func (l *Lobby) recordWinOnChain(winnerWallet string, history MatchHistory) {
	log.Printf("[ORACLE] Win Logged: %s vs %s. Payout sequence initiated.\n", winnerWallet, history.Opponent)
}

// recordDNFOnChain persists a match disconnection to the blockchain.
// Metadata includes the leaver, the opponent, and the tournament context for reconstruction.
func (l *Lobby) recordDNFOnChain(wallet, opponent, tid string) {
	if wallet == "" {
		return
	}

	l.mutex.RLock()
	voiConfig, _ := l.availableNetworks["Voi Mainnet"]
	l.mutex.RUnlock()
	client, _ := algod.MakeClient(voiConfig.NodeURL, "")
	pk, _ := mnemonic.ToPrivateKey(os.Getenv("FAUCET_MNEMONIC"))
	faucetAccount, _ := crypto.AccountFromPrivateKey(pk)
	sp, _ := client.SuggestedParams().Do(context.Background())
	senderAddr, _ := types.DecodeAddress(l.vaultAddress)

	meta := map[string]string{"leaver": wallet, "opp": opponent, "tid": tid}
	jsonData, _ := json.Marshal(meta)
	dnfNote := []byte(fmt.Sprintf("VBT_DNF:%s", string(jsonData)))

	appID, _ := strconv.ParseUint(voiConfig.AppID, 10, 64)
	// PILLAR 4: Historical Persistence. Send NoOp to vault with leaver context.
	txn, _ := transaction.MakeApplicationNoOpTx(appID, nil, []string{wallet}, nil, nil, sp, senderAddr, dnfNote, types.Digest{}, [32]byte{}, types.Address{})
	_, stxn, _ := crypto.SignTransaction(faucetAccount.PrivateKey, txn)
	client.SendRawTransaction(stxn).Do(context.Background())
}

func logWinAudit(recipient, network, txid, groupID string, amount uint64, history MatchHistory) {
	entry := struct {
		Timestamp string       `json:"timestamp"`
		Recipient string       `json:"recipient"`
		Network   string       `json:"network"`
		TxID      string       `json:"txid"`
		GroupID   string       `json:"group_id"`
		Amount    string       `json:"amount"`
		History   MatchHistory `json:"history"`
	}{
		Timestamp: time.Now().Format(time.RFC3339), Recipient: recipient, Network: network,
		TxID: txid, GroupID: groupID, Amount: fmt.Sprintf("%.1f $VBV", float64(amount)/1000000.0), History: history,
	}
	b, _ := json.Marshal(entry)
	f, _ := os.OpenFile("win_audit.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	f.Write(append(b, '\n'))
}

// CalculateReputation computes a player's social standing based on their performance and infamy.
func (l *Lobby) CalculateReputation(stats PlayerStats) int {
	// 1. Base Performance Score (Wins vs DNFs/Streak)
	// PILLAR 4: Competitive Balance.
	// Implemented diminishing returns for raw wins to prevent "grinding" from overwhelming
	// social markers (Achievements/Mojo) during long seasons or extreme simulations.
	winRep := 0
	if stats.Wins <= 100 {
		winRep = stats.Wins * 100
	} else if stats.Wins <= 500 {
		winRep = 10000 + (stats.Wins-100)*25
	} else {
		winRep = 20000 + (stats.Wins-500)*5
	}

	rep := winRep - (stats.DNFs * 50) - (stats.DisconnectStreak * 15)

	// 2. Infamy Penalty
	rep -= (stats.WantedLevel * 20)

	// 2.1 Asset Impoundment Penalty: Cards in sector custody reduce social reach
	rep -= (len(stats.JailedCards) * 25)

	// 3. Achievement Weighting
	for _, id := range stats.Achievements {
		bonus := 50 // Standard achievement
		switch id {
		case "GOVERNOR":
			bonus = 250 // Regional influence milestone
		case "ARENA_LEGEND":
			bonus = 150 // Career veteran milestone
		case "REHABILITATED", "OUTLAW_SLAYER":
			bonus = 75 // Specialized mid-tier achievements
		}
		rep += bonus
	}

	// 4. Marketability Multiplier (Aggressiveness & Risk rewarded as "Marketable Traits")
	// Instead of a flat bonus, playstyle now acts as a multiplier to scale with player performance.
	// Aggressiveness: Max +15%, Risk Tolerance: Max +10% (Total potential: 1.25x)
	marketabilityMult := 1.0 + (stats.Playstyle.Aggressiveness * 0.15) + (stats.Playstyle.RiskTolerance * 0.10)
	rep = int(float64(rep) * marketabilityMult)

	// 5. Employment Multiplier (Social Trust from high-Mojo Clubs)
	if stats.EmployerClubID != "" {
		// Note: Mutex expected to be held by caller (e.g. updateLeaderboard)
		club, exists := l.clubs[stats.EmployerClubID]
		if exists { // Check if the club actually exists
			// Multiplier scales with Club Mojo: 1.0 to 1.5 (at 1000 Mojo)
			multiplier := 1.0 + (float64(club.Mojo) / 2000.0)
			if multiplier > 1.5 {
				multiplier = 1.5
			}
			rep = int(float64(rep) * multiplier)
		}
	}

	// 6. Cosmetic Prestige Multiplier (Faceplates)
	// For Standard players, faceplates provide a flat Reputation boost to aid their climb.
	// For Diamond Tier (Rep >= 500) players, cosmetics provide a "Prestige Multiplier".
	if stats.EquippedFaceplate != "" {
		if fp, exists := FaceplateRegistry[stats.EquippedFaceplate]; exists {
			if rep >= 500 {
				// Diamond Tier: 1 Mojo point = 0.5% prestige multiplier (Max +25% for Governor)
				prestigeMult := 1.0 + (float64(fp.MojoBonus) * 0.005)
				rep = int(float64(rep) * prestigeMult)
			} else {
				// Standard: Additive bonus (1 Mojo = 10 Reputation points)
				rep += (fp.MojoBonus * 10)
			}
		}
	}

	// 7. Spreader Multiplier (Market Manipulation Reward)
	// Active participants in the Rumor Mill gain Standing for their social influence.
	// Hardening: Changed from a multiplier to an additive bonus to ensure players with
	// zero wins still gain visible "Standing" from rumor activity.
	if stats.RumorCount > 0 {
		spreaderBonus := int(float64(stats.RumorCount) * 10) // 10 reputation points per rumor spread
		if spreaderBonus > 100 {                             // Cap the bonus to prevent excessive reputation from rumors alone
			spreaderBonus = 100
		}
		rep += spreaderBonus
	}

	if rep < 0 {
		return 0
	}
	return rep
}
