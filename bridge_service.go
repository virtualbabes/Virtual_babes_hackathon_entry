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

	// 1. Per-wallet lock: Prevents a single user from making multiple concurrent onboarding requests.
	l.mutex.Lock()
	if _, isProcessing := l.processingOnboarding[req.Wallet]; isProcessing {
		l.mutex.Unlock()
		log.Printf("[BRIDGE] Onboarding already in progress for wallet: %s\n", req.Wallet)
		http.Error(w, "Onboarding already in progress for this wallet", http.StatusConflict)
		return
	}
	l.processingOnboarding[req.Wallet] = time.Now()
	l.mutex.Unlock()

	// Ensure the per-wallet claim is released after logic finishes (even on early exit)
	defer func() {
		l.mutex.Lock()
		delete(l.processingOnboarding, req.Wallet)
		l.mutex.Unlock()
	}()

	// 2. Global semaphore: Limits concurrent onboarding dispatches to prevent vault exhaustion.
	select {
	case l.onboardingSemaphore <- struct{}{}:
		// Acquired token, proceed
	case <-time.After(10 * time.Second): // Timeout if waiting too long
		log.Printf("[BRIDGE] Onboarding dispatch timed out for wallet: %s\n", req.Wallet)
		http.Error(w, "Server busy, please try again shortly.", http.StatusServiceUnavailable)
		return
	}
	defer func() {
		<-l.onboardingSemaphore // Release the token
	}()

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
	defer func() {
		if refundVBV {
			l.mutex.Lock()
			l.faucetBalance += 1.0 // Refund VBV
			l.mutex.Unlock()
			log.Printf("[BRIDGE] VBV refunded to vault for %s due to transaction failure.\n", req.Wallet)
		}
	}()

	accountInfo, err := client.AccountInformation(req.Wallet).Do(context.Background())
	if err == nil && accountInfo.Amount >= 1000000 {
		w.WriteHeader(http.StatusNoContent) // User already has VOI, skip starter pack
		return
	}

	// 3. Dispatch Starter Pack (1 VOI + 1 VBV)
	faucetMnemonic := os.Getenv("FAUCET_MNEMONIC")
	pk, err := mnemonic.ToPrivateKey(faucetMnemonic)
	if err != nil {
		http.Error(w, "Vault keys unavailable", http.StatusInternalServerError)
		return
	}
	faucetAccount, _ := crypto.AccountFromPrivateKey(pk)
	vaultAddr := l.vaultAddress

	sp, err := client.SuggestedParams().Do(context.Background())
	if err != nil {
		http.Error(w, "Failed to get suggested params", http.StatusInternalServerError)
		return
	}

	// A. Send 1 VOI (Native Gas)
	txn1, _ := transaction.MakePaymentTxn(vaultAddr, req.Wallet, 1000000, []byte("VBT_ONBOARD:GAS"), "", sp)

	// B. Send 1 $VBV (ARC-200 Reward Token)
	rewardAsset, _ := strconv.ParseUint(voiConfig.AssetID, 10, 64)
	senderAddr, _ := types.DecodeAddress(vaultAddr)
	txn2, _ := transaction.MakeApplicationNoOpTx(rewardAsset, nil, nil, nil, nil, sp, senderAddr, []byte("VBT_ONBOARD:TOKEN"), types.Digest{}, [32]byte{}, types.Address{})

	// Combine into Atomic Group
	gid, _ := crypto.ComputeGroupID([]types.Transaction{txn1, txn2})
	txn1.Group, txn2.Group = gid, gid
	_, stx1, _ := crypto.SignTransaction(faucetAccount.PrivateKey, txn1)
	_, stx2, _ := crypto.SignTransaction(faucetAccount.PrivateKey, txn2)

	txid, err := client.SendRawTransaction(append(stx1, stx2...)).Do(context.Background())
	if err != nil {
		log.Printf("[BRIDGE ERROR] Onboarding failed for %s: %v\n", req.Wallet, err)
		http.Error(w, "Bridge delivery failed", http.StatusInternalServerError)
		return
	}

	l.logAdminAudit("BRIDGE_ONBOARD", req.Wallet, "1 VOI + 1 VBV dispatched")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Voi Starter Pack sent!", "txid": txid})
	refundVBV = false // Transaction successful
}
