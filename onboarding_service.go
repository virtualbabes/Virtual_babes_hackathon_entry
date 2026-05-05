package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/mnemonic"
	"github.com/algorand/go-algorand-sdk/v2/transaction"
	"github.com/algorand/go-algorand-sdk/v2/types"
)

// handleVoiOnboarding provides a "Starter Pack" to Algorand users to bridge them to Voi.
// It implements a 'Processing' claim pattern to prevent concurrent Sybil/double-onboarding attacks.
func (l *Lobby) handleVoiOnboarding(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Wallet string `json:"wallet"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Wallet == "" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	targetWallet := strings.ToLower(req.Wallet)

	// 0. Safety Guard: Block onboarding if Sybil history is not yet restored.
	l.mutex.RLock()
	synced := l.SybilSyncComplete
	l.mutex.RUnlock()
	if !synced {
		http.Error(w, "Arena safety protocols are still initializing. Try again in 30 seconds.", http.StatusServiceUnavailable)
		return
	}

	// 1. Per-wallet lock: Prevents a single user from making multiple concurrent onboarding requests.
	l.mutex.Lock()
	if _, isProcessing := l.processingOnboarding[targetWallet]; isProcessing {
		l.mutex.Unlock()
		log.Printf("[BRIDGE] Onboarding already in progress for wallet: %s\n", targetWallet)
		http.Error(w, "Onboarding already in progress for this wallet", http.StatusConflict)
		return
	}
	l.processingOnboarding[targetWallet] = time.Now()
	l.mutex.Unlock()

	// Ensure the per-wallet claim is released after logic finishes (even on early exit)
	defer func() {
		l.mutex.Lock()
		delete(l.processingOnboarding, targetWallet)
		l.mutex.Unlock()
	}()

	// 2. Global semaphore: Limits concurrent onboarding dispatches to prevent vault exhaustion.
	select {
	case l.onboardingSemaphore <- struct{}{}:
		// Acquired token, proceed
	case <-time.After(10 * time.Second): // Timeout if waiting too long
		log.Printf("[BRIDGE] Onboarding dispatch timed out for wallet: %s\n", targetWallet)
		http.Error(w, "Server busy, please try again shortly.", http.StatusServiceUnavailable)
		return
	}
	defer func() {
		<-l.onboardingSemaphore // Release the token
	}()

	// NEW: Check if wallet has already been onboarded (historical check)
	l.mutex.RLock()
	alreadyOnboarded := l.onboardedWallets[targetWallet]
	l.mutex.RUnlock()
	if alreadyOnboarded {
		log.Printf("[BRIDGE] Wallet %s has already received an onboarding pack.\n", targetWallet)
		http.Error(w, "This wallet has already received an onboarding pack.", http.StatusForbidden)
		return
	}
	// 2. Sybil Protection: Check Voi side balance to see if they already have VOI
	l.mutex.RLock()
	voiConfig, ok := l.availableNetworks["Voi Mainnet"]
	l.mutex.RUnlock()
	if !ok {
		http.Error(w, "Voi Mainnet configuration not found", http.StatusInternalServerError)
		return
	}
	voiNodeURL := voiConfig.NodeURL
	client, err := algod.MakeClient(voiNodeURL, "")
	if err != nil {
		http.Error(w, "Internal Node Error", http.StatusInternalServerError)
		return
	}

	// 3. Atomic VBV balance check and decrement (under lock to prevent over-commitment)
	l.mutex.Lock()
	if l.faucetBalance < 1.0 { // Assuming 1 VBV is dispatched
		l.mutex.Unlock()
		http.Error(w, "Vault is low on VBV, please try again later.", http.StatusServiceUnavailable)
		return
	}
	l.faucetBalance -= 1.0 // Decrement for 1 VBV
	l.mutex.Unlock()       // Release lock before network I/O

	// --- Transaction Dispatch Logic ---
	// Ensure the VBV is refunded if the transaction fails
	refundVBV := true // Flag to track if VBV needs refunding
	isSkip := false   // Distinguish between skip and error
	defer func() {
		if refundVBV {
			l.mutex.Lock()
			l.faucetBalance += 1.0 // Refund VBV
			l.mutex.Unlock()
			if !isSkip {
				log.Printf("[BRIDGE] VBV refunded to vault for %s due to transaction failure.\n", targetWallet)
			}
		}
	}()

	accountInfo, err := client.AccountInformation(targetWallet).Do(context.Background())
	if err == nil && accountInfo.Amount >= 1000000 {
		isSkip = true
		w.WriteHeader(http.StatusNoContent) // User already has VOI, skip starter pack
		return
	}

	// 3. Dispatch Starter Pack (1 VOI + 1 VBV)
	faucetMnemonic := os.Getenv("FAUCET_MNEMONIC")
	pk, _ := mnemonic.ToPrivateKey(faucetMnemonic)
	faucetAccount, _ := crypto.AccountFromPrivateKey(pk)
	vaultAddr := l.vaultAddress

	sp, _ := client.SuggestedParams().Do(context.Background())
	txn1, _ := transaction.MakePaymentTxn(vaultAddr, targetWallet, 1000000, []byte("VBT_ONBOARD:GAS"), "", sp)
	rewardAsset, _ := strconv.ParseUint(voiConfig.AssetID, 10, 64)
	senderAddr, _ := types.DecodeAddress(vaultAddr)
	txn2, _ := transaction.MakeApplicationNoOpTx(rewardAsset, nil, nil, nil, nil, sp, senderAddr, []byte("VBT_ONBOARD:TOKEN"), types.Digest{}, [32]byte{}, types.Address{})

	gid, _ := crypto.ComputeGroupID([]types.Transaction{txn1, txn2})
	txn1.Group, txn2.Group = gid, gid
	_, stx1, _ := crypto.SignTransaction(faucetAccount.PrivateKey, txn1)
	_, stx2, _ := crypto.SignTransaction(faucetAccount.PrivateKey, txn2)

	txid, err := client.SendRawTransaction(append(stx1, stx2...)).Do(context.Background())
	if err != nil {
		log.Printf("[BRIDGE ERROR] Onboarding failed for %s: %v\n", targetWallet, err)
		http.Error(w, "Bridge delivery failed", http.StatusInternalServerError)
		return
	}

	// NEW: Mark wallet as onboarded after successful dispatch
	l.mutex.Lock()
	l.onboardedWallets[targetWallet] = true
	l.mutex.Unlock()

	l.logAdminAudit("BRIDGE_ONBOARD", targetWallet, "1 VOI + 1 VBV dispatched")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Voi Starter Pack sent!", "txid": txid})
	refundVBV = false // Transaction successful
}
