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
	if ratio > 1.0 { ratio = 1.0 }
	if ratio < 0.1 { ratio = 0.1 }

	// 1. Scale the primary base reward (for internal tracking/legacy logic)
	l.baseReward = uint64(float64(l.initialBaseReward) * ratio)

	// 2. Iterate through the entire reward stack and scale based on unscaled initial values
	for assetID, initialAmt := range l.initialRewards {
		scaledAmt := uint64(float64(initialAmt) * ratio)
		l.rewards[assetID] = scaledAmt
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
	os.WriteFile("season.json", conf, 0644)
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
	client.SendRawTransaction(stxn).Do(context.Background())
	return crypto.GetTxID(txn), nil
}

func (l *Lobby) recordWinOnChain(winnerWallet string, history MatchHistory) {
	log.Printf("[ORACLE] Win Logged: %s vs %s. Payout sequence initiated.\n", winnerWallet, history.Opponent)
}

func (l *Lobby) recordDNFOnChain(wallet string) {
	l.mutex.RLock()
	voiConfig, _ := l.availableNetworks["Voi Mainnet"]
	l.mutex.RUnlock()
	client, _ := algod.MakeClient(voiConfig.NodeURL, "")
	pk, _ := mnemonic.ToPrivateKey(os.Getenv("FAUCET_MNEMONIC"))
	faucetAccount, _ := crypto.AccountFromPrivateKey(pk)
	sp, _ := client.SuggestedParams().Do(context.Background())
	senderAddr, _ := types.DecodeAddress(l.vaultAddress)
	dnfNote := []byte(fmt.Sprintf("VBT_DNF:%d", time.Now().Unix()))

	appID, _ := strconv.ParseUint(voiConfig.AppID, 10, 64)
	txn, _ := transaction.MakeApplicationNoOpTx(appID, nil, nil, nil, nil, sp, senderAddr, dnfNote, types.Digest{}, [32]byte{}, types.Address{})
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
	rep := (stats.Wins * 100) - (stats.DNFs * 50) - (stats.DisconnectStreak * 15)

	// 2. Infamy Penalty
	rep -= (stats.WantedLevel * 20)

	// 3. Achievement Weighting
	rep += (len(stats.Achievements) * 50)

	// 4. Playstyle Tendencies (Aggressiveness & Risk rewarded as "Marketable Traits")
	playstyleBonus := (stats.Playstyle.Aggressiveness * 100.0) + (stats.Playstyle.RiskTolerance * 100.0)
	rep += int(playstyleBonus)

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

	// 6. Cosmetic Prestige (Faceplates)
	// High-tier faceplates provide status boosts that manifest as Reputation.
	if stats.EquippedFaceplate != "" {
		if fp, exists := FaceplateRegistry[stats.EquippedFaceplate]; exists {
			rep += (fp.MojoBonus * 10) // 1 Mojo point from cosmetics = 10 Reputation points
		}
	}

	if rep < 0 {
		return 0
	}
	return rep
}
