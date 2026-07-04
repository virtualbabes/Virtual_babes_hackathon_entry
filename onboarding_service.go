//go:build !js && !wasm

package main

import (
	"context"
	"errors"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/v2/client/v2/common/models"
	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/mnemonic"
	"github.com/algorand/go-algorand-sdk/v2/transaction"
	"github.com/algorand/go-algorand-sdk/v2/types"
)

var (
	ErrInsufficientLiquidity = errors.New("security exception: wallet liquidity below required threshold")
)

// NewSessionWatchdog initializes the continuous auditor for active sessions.
func (l *Lobby) NewSessionWatchdog(interval time.Duration) *SessionWatchdog {
	return &SessionWatchdog{
		AuditInterval:    interval,
		ActiveMonitoring: make(map[string]time.Time),
	}
}

// StartWatchdogEngine boots up the persistent non-blocking monitoring daemon.
// PILLAR 3: Continuous Verification.
func (l *Lobby) StartWatchdogEngine(ctx context.Context) {
	if l.sessionWatchdog == nil {
		l.sessionWatchdog = l.NewSessionWatchdog(10 * time.Minute) // Audit every 10 mins
	}

	ticker := time.NewTicker(l.sessionWatchdog.AuditInterval)
	go func() {
		defer ticker.Stop()
		log.Printf(" [Watchdog] Continuous session monitoring active (Interval: %v)\n", l.sessionWatchdog.AuditInterval)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				l.AuditActivePlayerSessions(ctx)
			}
		}
	}()
}

// TrackSession registers a player inside the active watchdog pool.
func (l *Lobby) TrackSession(walletAddress string) {
	if l.sessionWatchdog == nil { return }
	l.sessionWatchdog.Mu.Lock()
	defer l.sessionWatchdog.Mu.Unlock()
	l.sessionWatchdog.ActiveMonitoring[strings.ToLower(walletAddress)] = time.Now()
}

// UntrackSession removes a player from monitoring.
func (l *Lobby) UntrackSession(walletAddress string) {
	if l.sessionWatchdog == nil { return }
	l.sessionWatchdog.Mu.Lock()
	defer l.sessionWatchdog.Mu.Unlock()
	delete(l.sessionWatchdog.ActiveMonitoring, strings.ToLower(walletAddress))
}

// AuditActivePlayerSessions scans all active players for session expiration or asset depletion.
func (l *Lobby) AuditActivePlayerSessions(ctx context.Context) {
	l.sessionWatchdog.Mu.Lock()
	players := make([]string, 0, len(l.sessionWatchdog.ActiveMonitoring))
	for wallet := range l.sessionWatchdog.ActiveMonitoring {
		players = append(players, wallet)
	}
	l.sessionWatchdog.Mu.Unlock()

	if len(players) == 0 {
		return
	}

	for _, wallet := range players {
		// 1. Session Age Check (24-hour limit)
		l.sessionWatchdog.Mu.Lock()
		joinTime, exists := l.sessionWatchdog.ActiveMonitoring[wallet]
		l.sessionWatchdog.Mu.Unlock()

		if exists && time.Since(joinTime) > 24*time.Hour {
			l.DisconnectClient(wallet, EvictionPayload{WalletAddress: wallet, ReasonCode: "SESSION_EXPIRED"})
			continue
		}

		// 2. Liquidity & Identity Integrity Check
		// PILLAR 3: Mid-Session Validation.
		go func(w string) {
			l.mutex.RLock()
			voiConfig, hasConfig := l.availableNetworks["Voi Mainnet"]
			l.mutex.RUnlock()
			
			if !hasConfig { return }

			// Dedicated evaluation context to ensure RPC timeouts don't hang the worker
			auditCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			// PILLAR 4: Multi-Node Audit Failover.
			for _, nodeURL := range voiConfig.NodeURLs {
				client, _ := algod.MakeClient(nodeURL, "")
				info, err := client.AccountInformation(w).Do(auditCtx)
				if err == nil {
					// Evict if balance falls below 0.1 VOI (minimal gas floor for match archival)
					if info.Amount < 100000 {
						log.Printf(" [Watchdog] Eviction: %s failed liquidity audit (%d micro-VOI)\n", w, info.Amount)
						l.DisconnectClient(w, EvictionPayload{WalletAddress: w, ReasonCode: "INSUFFICIENT_LIQUIDITY"})
					}
					return // Success: Audit complete for this player
				}
			}
		}(wallet)
	}
}

// OnboardingService encapsulates logic for new player registration and Sybil protection.
// PILLAR 5: Stateless Service Design.
type OnboardingService struct{}

// HandleVoiOnboarding provides a "Starter Pack" to Algorand users to bridge them to Voi.
// It implements a 'Processing' claim pattern to prevent concurrent Sybil/double-onboarding attacks.
func (s *OnboardingService) HandleVoiOnboarding(l *Lobby, w http.ResponseWriter, r *http.Request) {
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
	recipientAddr, _ := types.DecodeAddress(targetWallet)

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

	// PILLAR 5: Defensive Map Handling.
	if l.processingOnboarding == nil {
		l.processingOnboarding = make(map[string]time.Time)
	}

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
	if len(voiConfig.NodeURLs) == 0 {
		http.Error(w, "No nodes configured", http.StatusInternalServerError)
		return
	}

	// 3. PILLAR 2: Integer Supremacy.
	// Execute atomic micro-unit check and decrement to prevent over-commitment.
	const packAmtMicro = 1000000 // 1 VBV
	l.mutex.Lock()
	if l.faucetBalanceMicro < packAmtMicro {
		l.mutex.Unlock()
		http.Error(w, "Vault is low on VBV, please try again later.", http.StatusServiceUnavailable)
		return
	}
	l.faucetBalanceMicro -= packAmtMicro
	l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0
	l.applyDynamicScalingLocked() // Recalculate reward stack based on new liquidity
	l.mutex.Unlock()              // Release lock before network I/O

	// --- Transaction Dispatch Logic ---
	// Ensure the VBV is refunded if the transaction fails
	refundVBV := true // Flag to track if VBV needs refunding
	isSkip := false   // Distinguish between skip and error
	defer func() {
		if refundVBV {
			l.mutex.Lock()
			l.faucetBalanceMicro += packAmtMicro
			l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0
			l.applyDynamicScalingLocked()
			l.mutex.Unlock()
			if !isSkip {
				log.Printf("[BRIDGE] VBV refunded to vault for %s due to transaction failure.\n", targetWallet)
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), indexerTimeout)
	defer cancel()

	// PILLAR 4: Multi-Node Onboarding Failover.
	var accountInfo *models.Account
	for _, nodeURL := range voiConfig.NodeURLs {
		c, _ := algod.MakeClient(nodeURL, "")
		info, err := c.AccountInformation(targetWallet).Do(ctx)
		if err == nil {
			accountInfo = &info
			break
		}
	}

	if accountInfo != nil && accountInfo.Amount >= 1000000 {
		isSkip = true
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// 3. Dispatch Starter Pack (1 VOI + 1 VBV)
	faucetMnemonic := os.Getenv("FAUCET_MNEMONIC")
	if faucetMnemonic == "" {
		log.Println("[BRIDGE CRITICAL] FAUCET_MNEMONIC environment variable is NOT SET.")
		http.Error(w, "server configuration error: faucet mnemonic missing", http.StatusInternalServerError)
		return
	}

	pk, err := mnemonic.ToPrivateKey(faucetMnemonic)
	if err != nil {
		log.Printf("[BRIDGE CRITICAL] Failed to convert FAUCET_MNEMONIC to private key: %v", err)
		http.Error(w, "faucet configuration error: invalid mnemonic", http.StatusInternalServerError)
		return
	}
	faucetAccount, err := crypto.AccountFromPrivateKey(pk)
	if err != nil {
		log.Printf("[BRIDGE CRITICAL] Failed to create account from private key: %v", err)
		http.Error(w, "internal cryptographic error", http.StatusInternalServerError)
		return
	}
	vaultAddr := l.vaultAddress

	// ARC-200 Protocol: transfer(address,uint256)
	// Selector: 0x2b426dec
	methodSelector := []byte{0x2b, 0x42, 0x6d, 0xec}
	amountMicro := big.NewInt(1000000)
	amountBytes := make([]byte, 32)
	amountMicro.FillBytes(amountBytes)

	appArgs := [][]byte{
		methodSelector,
		recipientAddr[:],
		amountBytes,
	}

	// PILLAR 4: Resilient Transaction Dispatch
	var txid string
	var lastErr error
	for _, nodeURL := range voiConfig.NodeURLs {
		client, _ := algod.MakeClient(nodeURL, "")
		sp, err := client.SuggestedParams().Do(context.Background())
		if err != nil {
			lastErr = err
			continue
		}

		txn1, _ := transaction.MakePaymentTxn(vaultAddr, targetWallet, 1000000, []byte("VBT_ONBOARD:GAS"), "", sp)
		rewardAsset, _ := strconv.ParseUint(voiConfig.AssetID, 10, 64)
		senderAddr, _ := types.DecodeAddress(vaultAddr)

		txn2, _ := transaction.MakeApplicationNoOpTx(rewardAsset, appArgs, nil, nil, nil, sp, senderAddr, []byte("VBT_ONBOARD:TOKEN"), types.Digest{}, [32]byte{}, types.Address{})

		gid, _ := crypto.ComputeGroupID([]types.Transaction{txn1, txn2})
		txn1.Group, txn2.Group = gid, gid
		_, stx1, _ := crypto.SignTransaction(faucetAccount.PrivateKey, txn1)
		_, stx2, _ := crypto.SignTransaction(faucetAccount.PrivateKey, txn2)

		res, err := client.SendRawTransaction(append(stx1, stx2...)).Do(ctx)
		if err == nil {
			txid = res
			break
		}
		lastErr = err
	}

	if txid == "" {
		log.Printf("[BRIDGE ERROR] Onboarding failed across all nodes: %v\n", lastErr)
		http.Error(w, "Bridge delivery failed", http.StatusInternalServerError)
		return
	}

	// NEW: Mark wallet as onboarded after successful dispatch
	l.mutex.Lock()
	l.onboardedWallets[targetWallet] = true

	// PILLAR 2: Industrial Loop (Physical Outflow).
	// Synchronize the Starter Pack exit with the audit kernel to ensure ledger circularity.
	if l.tokenSinkRouter != nil && l.tokenSinkRouter.Audit != nil {
		l.tokenSinkRouter.Audit.LogPhysicalOutflow(packAmtMicro)
	}

	l.mutex.Unlock()

	l.logAdminAudit("BRIDGE_ONBOARD", targetWallet, "1 VOI + 1 VBV dispatched")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Voi Starter Pack sent!", "txid": txid})
	refundVBV = false // Transaction successful, no refund needed
}

// HandleVoucherConversion migrates non-crypto Arena Vouchers to the virtual liability ledger.
// This is triggered when a console-originated player first links their wallet in the browser.
// PILLAR 2: Phase 4 Conversion Funnel.
func (s *OnboardingService) HandleVoucherConversion(l *Lobby, wallet string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	target := strings.ToLower(wallet)
	
	// PILLAR 2: Phase 4 Conversion Funnel.
	// We must harvest vouchers from all linked console accounts associated with this wallet.
	// Console players accumulate rewards on their hardware-bound profiles (ConsoleUID)
	// which are stored as temporary entries in the leaderboard before linking.
	
	totalHarvested := uint64(0)
	
	if linkInfo, ok := l.linkedWallets[target]; ok {
		for _, linked := range linkInfo.Linked {
			// PILLAR 3: Switchboard Security. Only harvest from cryptographically verified links.
			if !linked.Verified {
				continue
			}
			
			sourceID := strings.ToLower(linked.Address)
			if consoleStats, exists := l.leaderboard[sourceID]; exists && consoleStats.ArenaVouchers > 0 {
				v := consoleStats.ArenaVouchers
				totalHarvested += v
				
				// PILLAR 2: Integer Supremacy. Clear the source balance to prevent double-claiming.
				consoleStats.ArenaVouchers = 0
				l.leaderboard[sourceID] = consoleStats
				
				l.logAdminAuditLocked("VOUCHER_HARVEST", target, fmt.Sprintf("Harvested %d vouchers from console UID: %s", v, sourceID))
			}
		}
	}

	// Also check if the primary wallet itself has any accumulated vouchers
	if stats, exists := l.leaderboard[target]; exists && stats.ArenaVouchers > 0 {
		totalHarvested += stats.ArenaVouchers
		stats.ArenaVouchers = 0
		l.leaderboard[target] = stats
	}

	if totalHarvested == 0 {
		return
	}

	// Transfer the balance 1:1 to the virtual liability ledger (PlayerBalances).
	l.playerBalances[target] += totalHarvested

	// PILLAR 4: Activity Tracking.
	// Update LastActivity to prevent immediate Stagnation Tax triggers for newly engaged players.
	if stats, exists := l.leaderboard[target]; exists {
		stats.LastActivity = time.Now()
		l.leaderboard[target] = stats
	}

	// PILLAR 2: Solvency Check.
	l.applyDynamicScalingLocked()

	l.logAdminAuditLocked("VOUCHER_CONVERSION", target, fmt.Sprintf("Finalized conversion of %d vouchers to virtual VBV.", totalHarvested))

	// Notify the player
	clientID := l.getClientIDFromWalletLocked(target)
	if clientID != "" {
		msg := fmt.Sprintf(`{"text":"🎉 <b>VOUCHERS CONVERTED:</b> %d Arena Vouchers have been added to your reward balance!"}`, totalHarvested)
		l.sendToClientLocked(clientID, Envelope{Type: "admin_notification", Payload: json.RawMessage(msg)})
	}

	msg := l.getLobbyUpdateMsgLocked()
	go func() { l.broadcast <- msg }()
}

/**
 * HandleIdentityRefresh processes a 100 $VBV fee to update player metadata.
 * PILLAR 2: Economic Sink (Section 11).
 */
func (s *OnboardingService) HandleIdentityRefresh(l *Lobby, env *Envelope) {
	var data struct {
		NewHandle string `json:"handle"`
		NewBio    string `json:"bio"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil { return }

	l.mutex.Lock()
	defer l.mutex.Unlock()
	wallet, ok := l.wallets[env.FromID]
	if !ok { return }

	const refreshFeeMicro = 100 * 1000000
	if l.playerBalances[wallet] < refreshFeeMicro {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Insufficient rewards for identity refresh (100 $VBV required)."}`)})
		return
	}

	// Execute Sink: 90% Faucet / 10% Club
	l.playerBalances[wallet] -= refreshFeeMicro
	matrix := RevenueSplitMatrix{FaucetShare: 0.90, ClubShare: 0.10, GovernanceShare: 0.0}
	
	stats := l.leaderboard[wallet]
	clubID, _ := strconv.ParseUint(strings.TrimPrefix(stats.EmployerClubID, "CLUB-"), 10, 64)
	_ = l.tokenSinkRouter.RouteCriminalTax("IDENTITY_REFRESH", refreshFeeMicro, matrix, clubID, "")

	l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0
	l.applyDynamicScalingLocked()

	l.logAdminAuditLocked("IDENTITY_REFRESH", wallet, fmt.Sprintf("Handle: %s", data.NewHandle))
	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"🆔 <b>IDENTITY UPDATED:</b> Your signature has been refreshed across the sector."}`)})
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}
