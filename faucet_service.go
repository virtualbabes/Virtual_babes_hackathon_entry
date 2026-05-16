package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/encoding/msgpack"
	"github.com/algorand/go-algorand-sdk/v2/mnemonic"
	"github.com/algorand/go-algorand-sdk/v2/transaction"
	"github.com/algorand/go-algorand-sdk/v2/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// handleReward processes a request for a reward payout, verifying the client's intent
// via a reverse-signed nonce and then dispatching the reward on-chain.
func (l *Lobby) handleReward(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Rate limiting based on IP address
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	l.mutex.Lock()
	if lastReq, ok := l.rateLimits[ip]; ok && time.Since(lastReq) < 30*time.Second {
		l.mutex.Unlock()
		http.Error(w, "Rate limit exceeded. Please wait 30 seconds.", http.StatusTooManyRequests)
		return
	}
	l.rateLimits[ip] = time.Now()
	l.mutex.Unlock()

	var req struct {
		Recipient   string `json:"recipient"`
		Claimant    string `json:"claimant"`
		Network     string `json:"network"`
		ClientID    string `json:"client_id"`
		SignedTx    []byte `json:"signed_tx"`
		ClientScore [2]int `json:"client_score"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Claimant == "" {
		req.Claimant = req.Recipient
	}

	if strings.EqualFold(req.Network, "VOI") {
		if _, err := types.DecodeAddress(req.Recipient); err != nil {
			http.Error(w, "Invalid Voi payout recipient", http.StatusBadRequest)
			return
		}
		if err := l.verifyVoiPayoutOptIn(req.Recipient); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	l.mutex.Lock()
	history, hasHistory := l.matchHistory[req.ClientID]
	lastStarted, isProcessing := l.processingRewards[req.ClientID]
	if !hasHistory || (isProcessing && !lastStarted.IsZero()) {
		l.mutex.Unlock()
		http.Error(w, "Unauthorized: Payout unavailable or processing.", http.StatusUnauthorized)
		return
	}

	// SECURITY AUDIT: Verify that the claimant wallet matches the wallet registered to this session ID.
	actualWinnerWallet, walletOk := l.wallets[req.ClientID]
	if !walletOk || !strings.EqualFold(actualWinnerWallet, req.Claimant) {
		l.mutex.Unlock()
		http.Error(w, "Unauthorized: Identity mismatch.", http.StatusUnauthorized)
		return
	}

	// Score mismatch check to prevent tampering
	if req.ClientScore[0] != history.Scores[0] || req.ClientScore[1] != history.Scores[1] {
		l.mutex.Unlock()
		http.Error(w, "Unauthorized: Score mismatch.", http.StatusUnauthorized)
		return
	}

	l.processingRewards[req.ClientID] = time.Now() // Mark as processing
	l.mutex.Unlock()

	// Ensure processing status is cleared after function execution
	defer func() {
		l.mutex.Lock()
		delete(l.processingRewards, req.ClientID)
		l.mutex.Unlock()
	}()

	l.mutex.RLock()
	nonceData, exists := l.nonces[req.ClientID]
	l.mutex.RUnlock()

	if !exists || time.Since(nonceData.CreatedAt) > 5*time.Minute {
		http.Error(w, "Unauthorized: Session nonce expired.", http.StatusUnauthorized)
		return
	}

	var verified bool
	isEVMClaimant := strings.HasPrefix(req.Claimant, "0x")

	if isEVMClaimant {
		// EVM signature verification (personal_sign)
		message := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(nonceData.Value), nonceData.Value)
		messageHash := ethcrypto.Keccak256([]byte(message))

		signatureHex := string(req.SignedTx)
		signatureBytes, decodeErr := hex.DecodeString(strings.TrimPrefix(signatureHex, "0x"))
		if decodeErr != nil {
			log.Printf("[FAUCET] Invalid EVM signature format for %s: %v", req.Claimant, decodeErr)
			http.Error(w, "Invalid EVM signature format", http.StatusUnauthorized)
			return
		}

		if len(signatureBytes) != 65 {
			log.Printf("[FAUCET] Invalid EVM signature length for %s: %d", req.Claimant, len(signatureBytes))
			http.Error(w, "Invalid EVM signature length", http.StatusUnauthorized)
			return
		}
		if signatureBytes[64] == 27 || signatureBytes[64] == 28 {
			signatureBytes[64] -= 27
		}

		pubKey, recoverErr := ethcrypto.SigToPub(messageHash, signatureBytes)
		if recoverErr != nil {
			log.Printf("[FAUCET] EVM signature recovery failed for %s: %v", req.Claimant, recoverErr)
			http.Error(w, "EVM signature verification failed", http.StatusUnauthorized)
			return
		}

		recoveredAddress := ethcrypto.PubkeyToAddress(*pubKey).Hex()
		if strings.EqualFold(recoveredAddress, req.Claimant) {
			verified = true
		} else {
			log.Printf("[FAUCET] EVM signature mismatch. Recovered: %s, Expected: %s", recoveredAddress, req.Claimant)
			http.Error(w, "EVM signature mismatch", http.StatusUnauthorized)
			return
		}
	} else {
		// Algorand signature verification
		var stx types.SignedTxn
		if err := msgpack.Decode(req.SignedTx, &stx); err != nil {
			if err = json.Unmarshal(req.SignedTx, &stx); err != nil { // Fallback for JSON-encoded signed txn
				http.Error(w, "Invalid Algorand transaction proof format", http.StatusUnauthorized)
				return
			}
		}
		if stx.Txn.Sender.String() != req.Claimant || string(stx.Txn.Note) != nonceData.Value {
			http.Error(w, "Invalid Algorand Reverse Sign: Security mismatch", http.StatusUnauthorized)
			return
		}
		verified = true
	}

	if !verified {
		http.Error(w, "Signature verification failed.", http.StatusUnauthorized)
		return
	}

	// Dispatch the reward on-chain
	txid, bonus, skipped, dispatchErr := l.dispatchReward(req.Recipient, req.Claimant, req.Network, history)
	if dispatchErr != nil {
		http.Error(w, dispatchErr.Error(), http.StatusInternalServerError)
		return
	}

	l.mutex.Lock() // Lock to delete match history after successful dispatch
	delete(l.matchHistory, req.ClientID)
	l.mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success", "txid": txid, "bonus_applied": bonus, "skipped_assets": skipped,
	})
}

// dispatchReward constructs and sends the reward transaction(s) on-chain.
func (l *Lobby) verifyVoiPayoutOptIn(recipient string) error {
	l.mutex.RLock()
	assetID := l.availableNetworks["Voi Mainnet"].AssetID
	l.mutex.RUnlock()
	if assetID == "" || assetID == "0" {
		return nil
	}

	optedIn, _, err := l.checkAssetOptIn("VOI", recipient, assetID)
	if err != nil {
		return fmt.Errorf("failed to verify payout recipient opt-in: %w", err)
	}
	if !optedIn {
		return fmt.Errorf("payout recipient is not opted in to VBV on Voi")
	}
	return nil
}

func (l *Lobby) dispatchReward(recipient, claimant, network string, history MatchHistory) (string, bool, []string, error) {
	l.mutex.RLock()
	voiConfig, _ := l.availableNetworks["Voi Mainnet"]
	activeRewards := l.rewardStack
	stats, hasStats := l.leaderboard[claimant] // Reputation bonus applies to the player (claimant)
	vaultAddr := l.vaultAddress
	l.mutex.RUnlock()

	client, _ := algod.MakeClient(voiConfig.NodeURL, "")
	var skippedAssets []string
	mnemonicRaw := os.Getenv("FAUCET_MNEMONIC")
	if mnemonicRaw == "" {
		log.Println("[FAUCET CRITICAL] FAUCET_MNEMONIC environment variable is NOT SET.")
		return "", false, skippedAssets, fmt.Errorf("server configuration error: faucet mnemonic missing")
	}

	pk, err := mnemonic.ToPrivateKey(mnemonicRaw)
	if err != nil {
		log.Printf("[FAUCET CRITICAL] Failed to convert FAUCET_MNEMONIC to private key: %v", err)
		return "", false, skippedAssets, fmt.Errorf("faucet configuration error: invalid mnemonic")
	}
	faucetAccount, _ := crypto.AccountFromPrivateKey(pk)
	sp, _ := client.SuggestedParams().Do(context.Background())

	multiplier := 1.0
	bonusApplied := false
	if hasStats && stats.Reputation >= 500 { // Reputation bonus for rewards
		multiplier = 1.1
		bonusApplied = true
	}

	// PILLAR 1: Professional Hunter Bonus.
	// Hunters gain a multiplier on Bounty payouts based on their Mojo Tier.
	mojoMultiplier := 1.0
	if hasStats {
		if stats.Mojo >= 1000 {
			mojoMultiplier = 1.25 // Diamond
		} else if stats.Mojo >= 600 {
			mojoMultiplier = 1.15 // Gold
		} else if stats.Mojo >= 300 {
			mojoMultiplier = 1.10 // Silver
		} else if stats.Mojo >= 100 {
			mojoMultiplier = 1.05 // Bronze
		}
	}

	var txns []types.Transaction
	var totalUnits float64
	vaultAddrObj, _ := types.DecodeAddress(vaultAddr)
	// PILLAR 4: Match History Continuity. Include tournament ID and scores for reconstruction.
	// Refactored: mid = Match ID, tid = Tournament Instance ID
	winNote := []byte(fmt.Sprintf("VBT_WIN:{\"opp\":\"%s\",\"scores\":[%d,%d],\"tid\":\"%s\",\"mid\":\"%s\"}", history.Opponent, history.Scores[0], history.Scores[1], history.TournamentID, history.TournamentMatchID))

	l.mutex.Lock()
	virtualBalance := l.playerBalances[claimant]
	l.playerBalances[claimant] = 0 // Reset virtual balance as it's being committed to this payout
	l.mutex.Unlock()

	for appIDStr, baseAmt := range activeRewards {
		appID, err := strconv.ParseUint(appIDStr, 10, 64)
		if err != nil {
			continue
		}
		amt := uint64(float64(baseAmt) * multiplier)

		// PILLAR 2: Virtual Balance Integration.
		// Add accumulated balances from non-match activities (Salaries, Heists, Loans) to the primary reward.
		if appIDStr == l.rewardAssetID {
			amt += virtualBalance
		}

		// PILLAR 4: Bounty System Integration.
		// Add the calculated bounty from MatchHistory if this is the primary Arena asset.
		if appIDStr == l.rewardAssetID && history.BountyReward > 0 {
			// Apply Mojo tier multiplier to the bounty portion
			finalBounty := history.BountyReward * mojoMultiplier
			amt += uint64(finalBounty * 1000000)
			log.Printf("[FAUCET] Bounty payout of %.2f (scaled by %.2fx Mojo) included for %s", finalBounty, mojoMultiplier, recipient)
		}

		// NEW: Granular Opt-in Verification
		// Check if the recipient has a balance box/opt-in for this specific asset in the stack.
		optedIn, _, err := l.checkAssetOptIn("VOI", recipient, appIDStr)
		if err != nil {
			log.Printf("[FAUCET] Opt-in check failed for %s on asset %s: %v", recipient, appIDStr, err)
			skippedAssets = append(skippedAssets, appIDStr)
			continue
		}
		if !optedIn {
			log.Printf("[FAUCET] Recipient %s not opted-in to asset %s. Skipping to prevent group failure.", recipient, appIDStr)
			skippedAssets = append(skippedAssets, appIDStr)
			continue
		}

		// Check vault's balance for this specific asset
		boxResponse, err := client.GetApplicationBoxByName(appID, vaultAddrObj[:]).Do(context.Background())
		if err != nil {
			log.Printf("[FAUCET] Reward app %s box fetch failed: %v", appIDStr, err)
			skippedAssets = append(skippedAssets, appIDStr)
			continue
		}

		if len(boxResponse.Value) >= 32 {
			bal := new(big.Int).SetBytes(boxResponse.Value[:32]).Uint64()
			if bal < amt {
				log.Printf("[FAUCET] Insufficient balance in vault for asset %s. Needed: %d, Available: %d", appIDStr, amt, bal)
				skippedAssets = append(skippedAssets, appIDStr)
				continue
			}
		}

		totalUnits += float64(amt) / 1000000.0
		txn, _ := transaction.MakeApplicationNoOpTx(appID, nil, []string{recipient}, nil, nil, sp, vaultAddrObj, winNote, types.Digest{}, [32]byte{}, types.Address{})
		txns = append(txns, txn)
	}

	if len(txns) == 0 {
		return "", false, skippedAssets, fmt.Errorf("no rewards dispatched due to insufficient pool balance or configuration issues")
	}

	gid, _ := crypto.ComputeGroupID(txns)
	var signedGroup []byte
	var firstTxID string
	for i := range txns {
		txns[i].Group = gid
		txid, stxn, _ := crypto.SignTransaction(faucetAccount.PrivateKey, txns[i])
		signedGroup = append(signedGroup, stxn...)
		if i == 0 {
			firstTxID = txid
		}
	}

	if _, err := client.SendRawTransaction(signedGroup).Do(context.Background()); err != nil {
		return "", false, skippedAssets, fmt.Errorf("failed to send reward transaction: %v", err)
	}

	// Wait for confirmation to ensure the transaction is processed before updating internal state
	transaction.WaitForConfirmation(client, firstTxID, 4, context.Background())

	l.mutex.Lock()                // Lock to update faucet balance and re-evaluate dynamic scaling
	l.faucetBalance -= totalUnits // Deduct from the overall faucet balance
	l.applyDynamicScalingLocked() // Re-evaluate dynamic scaling after payout
	l.mutex.Unlock()

	logWinAudit(recipient, network, firstTxID, base64.StdEncoding.EncodeToString(gid[:]), uint64(totalUnits*1000000), history)
	return firstTxID, bonusApplied, skippedAssets, nil
}
