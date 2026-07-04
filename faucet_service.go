//go:build !js && !wasm

package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
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

// TransferTokens executes a single on-chain transfer from the vault to a governor.
// This satisfies the ExternalLedgerClient interface for the PayoutScheduler.
// PILLAR-B: Multi-chain routing via LoadBalancedLedgerClient cluster.
func (l *Lobby) TransferTokens(ctx context.Context, toWallet string, amount uint64) error {
	if amount == 0 {
		return nil
	}

	l.mutex.RLock()
	voiConfig, ok := l.availableNetworks["Voi Mainnet"]
	vaultAddr := l.vaultAddress
	hasRouter := l.multiChainRouter != nil
	l.mutex.RUnlock()

	if !ok || vaultAddr == "" {
		return errors.New("ledger error: voi network configuration missing")
	}

	// PILLAR 2: Single-Transaction Safety Cap.
	// Prevent anomalous outflows from draining the vault in a single event.
	if amount > MaxGovPayoutMicro {
		return fmt.Errorf("ledger error: amount %d exceeds governor payout cap", amount)
	}

	// 1. Gas Floor Audit (Voi native balance check)
	client, _ := algod.MakeClient(voiConfig.NodeURLs[0], "")
	accInfo, err := client.AccountInformation(vaultAddr).Do(ctx)
	if err != nil || accInfo.Amount < 1000000 {
		return errors.New("ledger error: vault native balance below gas floor")
	}

	// 2. Multi-chain dispatch via router (PILLAR-B)
	if hasRouter {
		err = l.multiChainRouter.TransferToChain(ctx, toWallet, amount, "voi")
		if err != nil {
			return fmt.Errorf("multi-chain dispatch failed: %w", err)
		}

		// 3. Economic state reconciliation (PILLAR 2: Industrial Loop)
		l.mutex.Lock()
		if l.faucetBalanceMicro >= amount {
			l.faucetBalanceMicro -= amount
		} else {
			l.faucetBalanceMicro = 0
		}
		l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0
		l.applyDynamicScalingLocked()

		if l.tokenSinkRouter != nil && l.tokenSinkRouter.Audit != nil {
			l.tokenSinkRouter.Audit.LogPhysicalOutflow(amount)
		}
		l.mutex.Unlock()

		return nil
	}

	// 3. Fallback: Direct Voi ARC-200 transfer (legacy path)
	mnemonicRaw := os.Getenv("FAUCET_MNEMONIC")
	if mnemonicRaw == "" {
		return errors.New("ledger error: faucet mnemonic missing from environment")
	}

	pk, err := mnemonic.ToPrivateKey(mnemonicRaw)
	if err != nil {
		return errors.New("ledger error: malformed mnemonic")
	}
	faucetAccount, _ := crypto.AccountFromPrivateKey(pk)

	sp, err := client.SuggestedParams().Do(ctx)
	if err != nil {
		return fmt.Errorf("ledger error: failed to fetch suggested params: %w", err)
	}

	// 4. Construct ARC-200 Transfer
	// method: transfer(address,uint256) -> selector: 0x2b426dec
	recipientAddr, _ := types.DecodeAddress(toWallet)
	methodSelector := []byte{0x2b, 0x42, 0x6d, 0xec}
	amountBytes := make([]byte, 32)
	new(big.Int).SetUint64(amount).FillBytes(amountBytes)

	appArgs := [][]byte{
		methodSelector,
		recipientAddr[:],
		amountBytes,
	}

	appID, _ := strconv.ParseUint(voiConfig.AssetID, 10, 64)
	senderAddr, _ := types.DecodeAddress(vaultAddr)
	note := []byte(fmt.Sprintf("VBT_GOV_DIVIDEND:{\"amount\":%d}", amount))

	txn, err := transaction.MakeApplicationNoOpTx(appID, appArgs, nil, nil, nil, sp, senderAddr, note, types.Digest{}, [32]byte{}, types.Address{})
	if err != nil {
		return fmt.Errorf("ledger error: transaction construction failed: %w", err)
	}

	// 5. Dispatch and Wait (fallback path)
	txid, stxn, err := crypto.SignTransaction(faucetAccount.PrivateKey, txn)
	if err != nil {
		return fmt.Errorf("ledger error: signing failed: %w", err)
	}

	if _, err := client.SendRawTransaction(stxn).Do(ctx); err != nil {
		return fmt.Errorf("ledger error: dispatch failed: %w", err)
	}

	_, err = transaction.WaitForConfirmation(client, txid, 4, ctx)
	if err == nil {
		// PILLAR 2: Industrial Loop (Physical Outflow).
		// Decrement the authoritative integer reservoir after successful confirmation.
		l.mutex.Lock()
		if l.faucetBalanceMicro >= amount {
			l.faucetBalanceMicro -= amount
		} else {
			l.faucetBalanceMicro = 0
		}
		l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0
		l.applyDynamicScalingLocked()

		if l.tokenSinkRouter != nil && l.tokenSinkRouter.Audit != nil {
			l.tokenSinkRouter.Audit.LogPhysicalOutflow(amount)
		}
		l.mutex.Unlock()
	}
	return err
}

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

	// PILLAR 5: Defensive Map Handling.
	if l.processingRewards == nil {
		l.processingRewards = make(map[string]time.Time)
	}

	// PILLAR 2: Secure Payout Tracking.
	// Use the authoritative wallet (Claimant) as the key to prevent multi-session claim exploits.
	lastStarted, isProcessing := l.processingRewards[req.Claimant]

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

	l.processingRewards[req.Claimant] = time.Now() // Mark as processing
	l.mutex.Unlock()

	// Ensure processing status is cleared after function execution
	defer func() {
		l.mutex.Lock()
		delete(l.processingRewards, req.Claimant)
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

	l.mutex.Lock()
	// PILLAR 4: Historical Immersion.
	// Update both winner and loser history records with the on-chain Receipt ID.
	// This ensures the verification checkmark appears in the UI for both parties.
	claimantLower := strings.ToLower(req.Claimant)
	if stats, exists := l.leaderboard[claimantLower]; exists {
		for i := range stats.History {
			if stats.History[i].Timestamp.Equal(history.Timestamp) {
				stats.History[i].ReceiptTxID = txid
				l.leaderboard[claimantLower] = stats
				break
			}
		}
	}
	if history.Opponent != "" && history.Opponent != "DRAW" {
		loserLower := strings.ToLower(history.Opponent)
		if lStats, exists := l.leaderboard[loserLower]; exists {
			for i := range lStats.History {
				if lStats.History[i].Timestamp.Equal(history.Timestamp) {
					lStats.History[i].ReceiptTxID = txid
					l.leaderboard[loserLower] = lStats
					break
				}
			}
		}
	}
	delete(l.matchHistory, req.ClientID)
	updateMsg := l.getLobbyUpdateMsgLocked()
	l.mutex.Unlock()
	l.broadcast <- updateMsg

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

	optedIn, _, err := l.oracleService.CheckAssetOptIn(l, "VOI", recipient, assetID)
	if err != nil {
		return fmt.Errorf("failed to verify payout recipient opt-in: %w", err)
	}
	if !optedIn {
		return fmt.Errorf("payout recipient is not opted in to VBV on Voi")
	}
	return nil
}

func (l *Lobby) dispatchReward(recipient, claimant, network string, history MatchHistory) (string, bool, []string, error) {
	// PILLAR 2: Ledger Integrity.
	// This function dispatches rewards from the 'playerBalances' virtual liability ledger.
	// 'ArenaVouchers' are non-crypto and must be converted to 'playerBalances' via
	// HandleVoucherConversion (onboarding_service.go) before they can be dispatched on-chain.
	// Therefore, this function correctly does NOT directly access 'PlayerStats.ArenaVouchers'.
	l.mutex.RLock()
	voiConfig, _ := l.availableNetworks["Voi Mainnet"]
	activeRewards := l.rewardStack
	stats, hasStats := l.leaderboard[claimant] // Reputation bonus applies to the player (claimant)
	vaultAddr := l.vaultAddress
	l.mutex.RUnlock()

	if len(voiConfig.NodeURLs) == 0 {
		return "", false, nil, fmt.Errorf("no Voi nodes configured")
	}

	client, _ := algod.MakeClient(voiConfig.NodeURLs[0], "")

	// PILLAR 2: Gas Floor Protection.
	// Verify the vault has sufficient native VOI to cover transaction fees.
	// We enforce a 1.0 VOI minimum "gas floor" to ensure network operationality.
	if accInfo, err := client.AccountInformation(vaultAddr).Do(context.Background()); err != nil || accInfo.Amount < 1000000 {
		log.Printf("[FAUCET ERROR] Reward blocked: Vault native balance below gas floor (1.0 VOI).\n")
		return "", false, nil, fmt.Errorf("arena is temporarily low on gas. Please notify an administrator")
	}

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
	var totalMicro uint64
	vaultAddrObj, _ := types.DecodeAddress(vaultAddr)
	// PILLAR 4: Match History Continuity. Include tournament ID and scores for reconstruction.
	// Refactored: mid = Match ID, tid = Tournament Instance ID
	winNote := []byte(fmt.Sprintf("VBT_WIN:{\"opp\":\"%s\",\"scores\":[%d,%d],\"tid\":\"%s\",\"mid\":\"%s\"}", history.Opponent, history.Scores[0], history.Scores[1], history.TournamentID, history.TournamentMatchID))

	l.mutex.Lock()
	virtualBalance := l.playerBalances[claimant]
	l.playerBalances[claimant] = 0 // Reset virtual balance as it's being committed to this payout
	l.mutex.Unlock()

	// PILLAR 2: Exit Siphon (Section 11).
	// A 2% fee is extracted when converting virtual $VBV into on-chain tokens.
	// This amount returns to the Faucet pool to maintain mathematical circularity.
	exitSiphonMicro := (virtualBalance * 2) / 100
	actualPayoutMicro := virtualBalance - exitSiphonMicro

	l.logAdminAudit("EXIT_SIPHON", claimant, fmt.Sprintf("Extracted %d micro-VBV from bridge conversion.", exitSiphonMicro))

	virtualBalanceIncluded := false

	for appIDStr, baseAmt := range activeRewards {
		appID, err := strconv.ParseUint(appIDStr, 10, 64)
		if err != nil {
			continue
		}
		// PILLAR 3: Integer Supremacy. Avoid floating point for balance derivation.
		amt := (baseAmt * uint64(multiplier*100)) / 100

		// PILLAR 2: Virtual Balance Integration.
		// Add accumulated balances from non-match activities (Salaries, Heists, Loans) to the primary reward.
		if appIDStr == l.rewardAssetID {
			amt += actualPayoutMicro
		}

		// PILLAR 4: Bounty System Integration.
		// Add the calculated bounty from MatchHistory if this is the primary Arena asset.
		if appIDStr == l.rewardAssetID && history.BountyRewardMicro > 0 {
			// Apply Mojo tier multiplier to the bounty portion
			finalBounty := float64(history.BountyRewardMicro) / 1000000.0 * mojoMultiplier
			amt += uint64(finalBounty * 1000000)
			log.Printf("[FAUCET] Bounty payout of %.2f (scaled by %.2fx Mojo) included for %s", finalBounty, mojoMultiplier, recipient)
		}

		// NEW: Granular Opt-in Verification
		// Check if the recipient has a balance box/opt-in for this specific asset in the stack.
		optedIn, _, err := l.oracleService.CheckAssetOptIn(l, "VOI", recipient, appIDStr)
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
		totalMicro += amt
		txn, _ := transaction.MakeApplicationNoOpTx(appID, nil, []string{recipient}, nil, nil, sp, vaultAddrObj, winNote, types.Digest{}, [32]byte{}, types.Address{})
		txns = append(txns, txn)

		// PILLAR 2: Virtual Balance Tracking.
		// Mark that the virtual liability has been successfully included in the outgoing group.
		if appIDStr == l.rewardAssetID {
			virtualBalanceIncluded = true
		}
	}

	// PILLAR 2: Single-Transaction Safety Cap.
	// Enforce the 'Circuit Breaker' (1,000 VBV) before signing or dispatching.
	// If the group exceeds the systemic limit, rollback the virtual balance.
	if totalMicro > MaxSinglePayoutMicro {
		l.mutex.Lock()
		l.playerBalances[claimant] += virtualBalance // ROLLBACK
		l.mutex.Unlock()
		log.Printf("[SECURITY ALERT] Payout for %s BLOCKED: Total %d exceeds safety cap.\n", recipient, totalMicro)
		return "", false, skippedAssets, fmt.Errorf("payout exceeds single-transaction safety limit (%d VBV)", MaxSinglePayoutMicro/1000000)
	}

	if len(txns) == 0 {
		// PILLAR 2: Rollback Circuit.
		// If no transactions were built, restore the player's virtual balance to the ledger.
		l.mutex.Lock()
		l.playerBalances[claimant] += virtualBalance
		l.mutex.Unlock()
		return "", false, skippedAssets, fmt.Errorf("no rewards dispatched due to insufficient pool balance or configuration issues")
	}

	// PILLAR 2: Partial Rollback.
	// If the group is valid but the primary asset was skipped, restore the virtual balance.
	if !virtualBalanceIncluded && virtualBalance > 0 {
		l.mutex.Lock()
		l.playerBalances[claimant] += virtualBalance
		l.mutex.Unlock()
		log.Printf("[FAUCET] Virtual balance of %d returned to %s: Primary asset was skipped.", virtualBalance, claimant)
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
		// PILLAR 2: Rollback Circuit.
		// If the broadcast fails, restore the virtual balance to the ledger to prevent fund loss.
		l.mutex.Lock()
		l.playerBalances[claimant] += virtualBalance
		l.mutex.Unlock()
		return "", false, skippedAssets, fmt.Errorf("failed to send reward transaction: %v", err)
	}

	// Wait for confirmation to ensure the transaction is processed before updating internal state
	transaction.WaitForConfirmation(client, firstTxID, 4, context.Background())

	l.mutex.Lock()                // Lock to update faucet balance and re-evaluate dynamic scaling
	// PILLAR 2: Integer Supremacy.
	// Decrement the authoritative integer reservoir before deriving the display float.
	if l.faucetBalanceMicro >= totalMicro {
		l.faucetBalanceMicro -= totalMicro
	} else {
		l.faucetBalanceMicro = 0
	}
	l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0
	l.applyDynamicScalingLocked() // Re-evaluate dynamic scaling after payout

	if l.tokenSinkRouter != nil && l.tokenSinkRouter.Audit != nil {
		l.tokenSinkRouter.Audit.LogPhysicalOutflow(totalMicro)
	}
	l.mutex.Unlock()

	logWinAudit(recipient, network, firstTxID, base64.StdEncoding.EncodeToString(gid[:]), totalMicro, history)
	return firstTxID, bonusApplied, skippedAssets, nil
}
