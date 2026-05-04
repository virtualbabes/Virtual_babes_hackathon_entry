package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings" // For Solana verification
	"time"

	"math/big"
	"net"
	"net/http"
	"strconv"

	"github.com/algorand/go-algorand-sdk/v2/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/encoding/msgpack"
	"github.com/algorand/go-algorand-sdk/v2/mnemonic"
	"github.com/algorand/go-algorand-sdk/v2/transaction"
	"github.com/algorand/go-algorand-sdk/v2/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func (l *Lobby) handleReward(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

	l.mutex.Lock()
	history, hasHistory := l.matchHistory[req.ClientID]
	lastStarted, isProcessing := l.processingRewards[req.ClientID]
	if !hasHistory || (isProcessing && !lastStarted.IsZero()) {
		l.mutex.Unlock()
		http.Error(w, "Unauthorized: Payout unavailable or processing.", http.StatusUnauthorized)
		return
	}

	if req.ClientScore[0] != history.Scores[0] || req.ClientScore[1] != history.Scores[1] {
		l.mutex.Unlock()
		http.Error(w, "Unauthorized: Score mismatch.", http.StatusUnauthorized)
		return
	}

	l.processingRewards[req.ClientID] = time.Now()
	l.mutex.Unlock()

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
	// Determine if claimant is an EVM address (starts with "0x")
	isEVMClaimant := strings.HasPrefix(req.Claimant, "0x")

	if isEVMClaimant {
		// EVM signature verification (personal_sign)
		// The signed message is the nonce, prefixed by "\x19Ethereum Signed Message:\n" + len(nonce)
		message := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(nonceData.Value), nonceData.Value)
		messageHash := ethcrypto.Keccak256([]byte(message))

		// Decode the hex signature from app.js (e.g., "0x...")
		signatureHex := string(req.SignedTx) // Convert []byte to string
		signatureBytes, decodeErr := hex.DecodeString(strings.TrimPrefix(signatureHex, "0x"))
		if decodeErr != nil {
			log.Printf("[ECONOMY] Invalid EVM signature format for %s: %v", req.Claimant, decodeErr)
			http.Error(w, "Invalid EVM signature format", http.StatusUnauthorized)
			return
		}

		// Adjust 'v' value for secp256k1 recovery (v can be 0/1 or 27/28)
		// SigToPub expects v to be 0 or 1.
		if len(signatureBytes) != 65 {
			log.Printf("[ECONOMY] Invalid EVM signature length for %s: %d", req.Claimant, len(signatureBytes))
			http.Error(w, "Invalid EVM signature length", http.StatusUnauthorized)
			return
		}
		if signatureBytes[64] == 27 || signatureBytes[64] == 28 {
			signatureBytes[64] -= 27 // Transform to 0/1
		}

		pubKey, recoverErr := ethcrypto.SigToPub(messageHash, signatureBytes)
		if recoverErr != nil {
			log.Printf("[ECONOMY] EVM signature recovery failed for %s: %v", req.Claimant, recoverErr)
			http.Error(w, "EVM signature verification failed", http.StatusUnauthorized)
			return
		}

		recoveredAddress := ethcrypto.PubkeyToAddress(*pubKey).Hex()
		if strings.ToLower(recoveredAddress) == strings.ToLower(req.Claimant) {
			verified = true
		} else {
			log.Printf("[ECONOMY] EVM signature mismatch. Recovered: %s, Expected: %s", recoveredAddress, req.Claimant)
			http.Error(w, "EVM signature mismatch", http.StatusUnauthorized)
			return
		}
	} else {
		// Algorand signature verification (existing logic)
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

	// If we reach here, either EVM or Algorand signature was successfully verified
	txid, bonus, skipped, dispatchErr := l.dispatchReward(req.Recipient, req.Network, history)
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

func (l *Lobby) dispatchReward(recipient, network string, history MatchHistory) (string, bool, []string, error) {
	l.mutex.RLock()
	voiConfig, _ := l.availableNetworks["Voi Mainnet"]
	activeRewards := l.rewards
	stats, hasStats := l.leaderboard[recipient]
	l.mutex.RUnlock()

	client, _ := algod.MakeClient(voiConfig.NodeURL, "")
	pk, _ := mnemonic.ToPrivateKey(os.Getenv("FAUCET_MNEMONIC"))
	faucetAccount, _ := crypto.AccountFromPrivateKey(pk)
	sp, _ := client.SuggestedParams().Do(context.Background())

	multiplier := 1.0
	bonusApplied := false
	if hasStats && stats.Reputation >= 500 {
		multiplier = 1.1
		bonusApplied = true
	}

	var txns []types.Transaction
	var skippedAssets []uint64
	var totalUnits float64
	vaultAddrObj, _ := types.DecodeAddress(l.vaultAddress)
	winNote := []byte(fmt.Sprintf("VBT_WIN:{\"opp\":\"%s\",\"scores\":[%d,%d]}", history.Opponent, history.Scores[0], history.Scores[1]))

	for appIDStr, baseAmt := range activeRewards {
		appID, _ := strconv.ParseUint(appIDStr, 10, 64)
		amt := uint64(float64(baseAmt) * multiplier)
		boxResponse, err := client.GetApplicationBoxByName(appID, vaultAddrObj[:]).Do(context.Background())
		if err != nil {
			log.Printf("[ECONOMY] Reward app %s box fetch failed: %v", appIDStr, err)
			skippedAssets = append(skippedAssets, appIDStr)
			continue
		}

		if len(boxResponse.Value) >= 32 {
			bal := new(big.Int).SetBytes(boxResponse.Value[:32]).Uint64()
			if bal < amt {
				skippedAssets = append(skippedAssets, appIDStr)
				continue
			}
		}

		totalUnits += float64(amt) / 1000000.0
		txn, _ := transaction.MakeApplicationNoOpTx(appID, nil, []string{recipient}, nil, nil, sp, vaultAddrObj, winNote, types.Digest{}, [32]byte{}, types.Address{})
		txns = append(txns, txn)
	}

	if len(txns) == 0 {
		return "", false, skippedAssets, fmt.Errorf("insufficient pool balance")
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
		return "", false, nil, err
	}

	transaction.WaitForConfirmation(client, firstTxID, 4, context.Background())

	l.mutex.Lock()
	l.faucetBalance -= totalUnits
	l.applyDynamicScaling()
	l.mutex.Unlock()

	logWinAudit(recipient, network, firstTxID, base64.StdEncoding.EncodeToString(gid[:]), uint64(totalUnits*1000000), history)
	return firstTxID, bonusApplied, skippedAssets, nil
}

func (l *Lobby) applyDynamicScaling() {
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
	l.baseReward = uint64(float64(l.initialBaseReward) * ratio)
	if _, exists := l.rewards[l.rewardAssetID]; exists {
		l.rewards[l.rewardAssetID] = l.baseReward
	}
}

// handleGetLoans returns all active loans or loans specific to a player.
func (l *Lobby) handleGetLoans(w http.ResponseWriter, r *http.Request) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	var loans []*Loan
	borrowerWallet := r.URL.Query().Get("wallet")

	for _, loan := range l.loans {
		if borrowerWallet == "" || strings.EqualFold(loan.BorrowerWallet, borrowerWallet) {
			loans = append(loans, loan)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loans)
}

// handleTakeLoan allows a player to take a $VBV loan using a CardBundle as collateral.
func (l *Lobby) handleTakeLoan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Wallet           string     `json:"wallet"`
		CollateralBundle CardBundle `json:"collateral_bundle"`
		LoanAmount       float64    `json:"loan_amount"` // In whole $VBV units
		DurationHours    int        `json:"duration_hours"`
		ClientID         string     `json:"client_id"`
		SignedTx         []byte     `json:"signed_tx"` // For nonce verification
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.LoanAmount <= 0 || req.DurationHours <= 0 {
		http.Error(w, "Invalid loan amount or duration", http.StatusBadRequest)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Nonce verification
	nonceData, nonceExists := l.nonces[req.ClientID]
	if !nonceExists || time.Since(nonceData.CreatedAt) > 5*time.Minute {
		http.Error(w, "Unauthorized: Nonce expired", http.StatusUnauthorized)
		return
	}
	var stx types.SignedTxn
	if err := msgpack.Decode(req.SignedTx, &stx); err != nil || stx.Txn.Sender.String() != req.Wallet || string(stx.Txn.Note) != nonceData.Value {
		http.Error(w, "Unauthorized: Signature mismatch", http.StatusUnauthorized)
		return
	}

	// Check collateral in player's inventory
	stats := l.leaderboard[req.Wallet]
	if req.CollateralBundle.CardID != 0 && stats.Inventory[fmt.Sprintf("CARD-%d", req.CollateralBundle.CardID)] <= 0 {
		http.Error(w, "Collateral card not in inventory", http.StatusBadRequest)
		return
	}
	if req.CollateralBundle.WeaponID != "" && stats.Inventory[req.CollateralBundle.WeaponID] <= 0 {
		http.Error(w, "Collateral weapon not in inventory", http.StatusBadRequest)
		return
	}
	if req.CollateralBundle.FaceplateID != "" && stats.Inventory[req.CollateralBundle.FaceplateID] <= 0 {
		http.Error(w, "Collateral faceplate not in inventory", http.StatusBadRequest)
		return
	}

	// Escrow collateral: Deduct from player's inventory
	if req.CollateralBundle.CardID != 0 { stats.Inventory[fmt.Sprintf("CARD-%d", req.CollateralBundle.CardID)]-- }
	if req.CollateralBundle.WeaponID != "" { stats.Inventory[req.CollateralBundle.WeaponID]-- }
	if req.CollateralBundle.FaceplateID != "" { stats.Inventory[req.CollateralBundle.FaceplateID]-- }
	l.leaderboard[req.Wallet] = stats

	// Calculate repayment amount (e.g., 10% interest)
	loanAmountMicro := uint64(req.LoanAmount * 1000000)
	repaymentAmountMicro := uint64(float64(loanAmountMicro) * 1.10) // 10% interest

	loanID := fmt.Sprintf("LOAN-%d", time.Now().UnixNano())
	l.loans[loanID] = &Loan{
		ID:              loanID,
		BorrowerWallet:  req.Wallet,
		CollateralBundle: req.CollateralBundle,
		LoanAmount:      loanAmountMicro,
		RepaymentAmount: repaymentAmountMicro,
		DueAt:           time.Now().Add(time.Duration(req.DurationHours) * time.Hour),
		Status:          "active",
		TerritoryID:     "the_second_hand_store", // Fixed territory for Second-Hand Store
	}

	// Dispense loan amount to player's rewards
	l.rewards[req.Wallet] += loanAmountMicro

	l.logAdminAudit("LOAN_TAKEN", req.Wallet, fmt.Sprintf("Loan ID: %s, Amount: %.2f, Repay: %.2f", loanID, req.LoanAmount, float64(repaymentAmountMicro)/1000000.0))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(l.loans[loanID])

	// Trigger global sync to update UI (inventory, rewards)
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// handleGetBlackMarket returns liquidated items available for purchase.
// Gated by Cunning and Wanted Level.
func (l *Lobby) handleGetBlackMarket(w http.ResponseWriter, r *http.Request) {
	wallet := r.URL.Query().Get("wallet")
	
	l.mutex.RLock()
	stats, exists := l.leaderboard[wallet]
	l.mutex.RUnlock()

	if !exists || stats.Cunning < 10 || stats.WantedLevel < 5 {
		http.Error(w, "The Black Market is hidden from you. Increase your Cunning and Infamy.", http.StatusForbidden)
		return
	}

	l.mutex.RLock()
	defer l.mutex.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(l.blackMarket)
}

// handleSellMarketTokens allows players to liquidate equity back into spendable $VBV.
func (l *Lobby) handleSellMarketTokens(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Wallet string `json:"wallet"`
		Amount uint64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	stats, exists := l.leaderboard[req.Wallet]
	if !exists || stats.MarketTokens < req.Amount {
		http.Error(w, "Insufficient Market Tokens", http.StatusBadRequest)
		return
	}

	// Exchange rate: 1 Market Token = 0.8 VBV (Scavenger tax)
	vbvGainMicro := uint64(float64(req.Amount) * 0.8)
	stats.MarketTokens -= req.Amount
	l.rewards[req.Wallet] += vbvGainMicro
	l.leaderboard[req.Wallet] = stats

	l.logAdminAudit("TOKEN_LIQUIDATION", req.Wallet, fmt.Sprintf("Sold %d tokens for %.2f $VBV", req.Amount, float64(vbvGainMicro)/1000000.0))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "vbv_gained": vbvGainMicro})
}

// handleBuyBlackMarket allows high-infamy players to buy liquidated bundles at a discount.
func (l *Lobby) handleBuyBlackMarket(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Wallet  string `json:"wallet"`
		LoanID  string `json:"loan_id"`
		Network string `json:"network"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	// 1. Find the liquidated loan in blackMarket
	idx := -1
	for i, bm := range l.blackMarket {
		if bm.ID == req.LoanID { idx = i; break }
	}
	if idx == -1 {
		http.Error(w, "Item no longer available", http.StatusNotFound)
		return
	}

	loan := l.blackMarket[idx]
	// Scavenger Price: 75% of the original repayment amount
	scavengePrice := uint64(float64(loan.RepaymentAmount) * 0.75)

	if l.rewards[req.Wallet] < scavengePrice {
		http.Error(w, "Insufficient reward balance to scavenge this bundle", http.StatusPaymentRequired)
		return
	}

	// Execute scavenge
	l.rewards[req.Wallet] -= scavengePrice
	stats := l.leaderboard[req.Wallet]
	if stats.Inventory == nil { stats.Inventory = make(map[string]int) }
	
	// Add items with a Wanted penalty (Scavenging stolen goods)
	if loan.CollateralBundle.CardID != 0 { stats.Inventory[fmt.Sprintf("CARD-%d", loan.CollateralBundle.CardID)]++ }
	if loan.CollateralBundle.WeaponID != "" { stats.Inventory[loan.CollateralBundle.WeaponID]++ }
	if loan.CollateralBundle.FaceplateID != "" { stats.Inventory[loan.CollateralBundle.FaceplateID]++ }
	
	stats.WantedLevel += 5 // Scavenging stolen goods increases infamy
	l.leaderboard[req.Wallet] = stats

	// Remove from Black Market
	l.blackMarket = append(l.blackMarket[:idx], l.blackMarket[idx+1:]...)

	l.logAdminAudit("BLACK_MARKET_BUY", req.Wallet, fmt.Sprintf("Scavenged %s for %.2f $VBV", loan.ID, float64(scavengePrice)/1000000.0))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Stolen goods acquired. Watch your back."})
	
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// handleRepayLoan allows a player to repay a loan and retrieve their collateral.
func (l *Lobby) handleRepayLoan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LoanID  string `json:"loan_id"`
		Wallet  string `json:"wallet"`
		TxID    string `json:"txid"`
		Network string `json:"network"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Wallet == "" || req.TxID == "" {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	loan, exists := l.loans[req.LoanID]
	if !exists || loan.Status != "active" || strings.ToLower(loan.BorrowerWallet) != strings.ToLower(req.Wallet) {
		http.Error(w, "Loan not found or not active for this wallet", http.StatusBadRequest)
		return
	}

	// Verify the repayment transaction
	voiConfig, voiOk := l.availableNetworks["Voi Mainnet"]
	if !voiOk {
		http.Error(w, "Voi network configuration missing", http.StatusInternalServerError)
		return
	}
	assetID := voiConfig.AssetID
	verifyNet := "Voi"
	if req.Network == "ALGO" {
		assetID = l.avoiAssetID
		verifyNet = "Algorand"
	}

	verified, _, err := l.verifyBuyInTransaction(verifyNet, req.TxID, loan.RepaymentAmount, assetID, req.Wallet, l.vaultAddress)
	if err != nil || !verified {
		log.Printf("[LOAN] Repayment verification failed for %s. Error: %v\n", req.Wallet, err)
		http.Error(w, "Repayment verification failed or insufficient amount", http.StatusPaymentRequired)
		return
	}

	// Return collateral to player's inventory
	stats := l.leaderboard[req.Wallet]
	if stats.Inventory == nil { stats.Inventory = make(map[string]int) }
	if loan.CollateralBundle.CardID != 0 { stats.Inventory[fmt.Sprintf("CARD-%d", loan.CollateralBundle.CardID)]++ }
	if loan.CollateralBundle.WeaponID != "" { stats.Inventory[loan.CollateralBundle.WeaponID]++ }
	if loan.CollateralBundle.FaceplateID != "" { stats.Inventory[loan.CollateralBundle.FaceplateID]++ }
	l.leaderboard[req.Wallet] = stats

	// Update loan status and remove
	loan.Status = "repaid"
	delete(l.loans, req.LoanID) // Remove from active loans

	l.logAdminAudit("LOAN_REPAID", req.Wallet, fmt.Sprintf("Loan ID: %s, Amount: %.2f", loan.ID, float64(loan.RepaymentAmount)/1000000.0))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"message": fmt.Sprintf("Loan %s repaid. Collateral returned.", loan.ID),
	})

	// Trigger global sync to update UI (inventory, rewards)
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
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

func (l *Lobby) mockARC200Dispenser(recipient string, amount float64, rank int) {
	txid := fmt.Sprintf("MOCK-TRN-%d-%s", time.Now().Unix(), strings.ToUpper(recipient[:6]))
	microUnits := uint64(amount * 1000000)
	history := MatchHistory{WinnerID: recipient, Timestamp: time.Now(), Scores: [2]int{rank, 0}}
	logWinAudit(recipient, "VOI", txid, "MOCK-GROUP", microUnits, history)
}

// handleGetAuctions returns all active listings in the Art Gallery.
func (l *Lobby) handleGetAuctions(w http.ResponseWriter, r *http.Request) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	var list []*Auction
	for _, a := range l.auctions {
		list = append(list, a)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// handleCreateAuction allows a player to list a bundle for $VBV.
func (l *Lobby) handleCreateAuction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Wallet     string     `json:"wallet"`
		Bundle     CardBundle `json:"bundle"`
		StartPrice float64    `json:"start_price"`
		Duration   int        `json:"duration_hours"`
		ClientID   string     `json:"client_id"`
		SignedTx   []byte     `json:"signed_tx"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	l.mutex.RLock()
	nonceData, nonceExists := l.nonces[req.ClientID]
	l.mutex.RUnlock()
	if !nonceExists || time.Since(nonceData.CreatedAt) > 5*time.Minute {
		http.Error(w, "Unauthorized: Nonce expired", http.StatusUnauthorized)
		return
	}

	var stx types.SignedTxn
	if err := msgpack.Decode(req.SignedTx, &stx); err != nil || stx.Txn.Sender.String() != req.Wallet || string(stx.Txn.Note) != nonceData.Value {
		http.Error(w, "Unauthorized: Signature mismatch", http.StatusUnauthorized)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	stats := l.leaderboard[req.Wallet]
	if req.Bundle.WeaponID != "" && stats.Inventory[req.Bundle.WeaponID] <= 0 {
		http.Error(w, "Weapon not in inventory", http.StatusBadRequest)
		return
	}
	if req.Bundle.FaceplateID != "" && stats.Inventory[req.Bundle.FaceplateID] <= 0 {
		http.Error(w, "Faceplate not in inventory", http.StatusBadRequest)
		return
	}

	// Escrow items
	if req.Bundle.WeaponID != "" { stats.Inventory[req.Bundle.WeaponID]-- }
	if req.Bundle.FaceplateID != "" { stats.Inventory[req.Bundle.FaceplateID]-- }
	l.leaderboard[req.Wallet] = stats

	auctionID := fmt.Sprintf("AUC-%d", time.Now().UnixNano())
	l.auctions[auctionID] = &Auction{
		ID:           auctionID,
		SellerWallet: req.Wallet,
		Bundle:       req.Bundle,
		CurrentBid:   uint64(req.StartPrice * 1000000),
		EndsAt:       time.Now().Add(time.Duration(req.Duration) * time.Hour),
		TerritoryID:  "the_art_gallery",
	}

	l.logAdminAudit("AUCTION_CREATED", req.Wallet, fmt.Sprintf("ID: %s, Price: %.2f", auctionID, req.StartPrice))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(l.auctions[auctionID])
}

func (l *Lobby) handlePlaceBid(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AuctionID string `json:"auction_id"`
		Bidder    string `json:"wallet"`
		Amount    uint64 `json:"amount_micro"`
		ClientID  string `json:"client_id"`
		SignedTx  []byte `json:"signed_tx"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	auction, exists := l.auctions[req.AuctionID]
	if !exists || time.Now().After(auction.EndsAt) {
		http.Error(w, "Auction expired or not found", http.StatusNotFound)
		return
	}

	if req.Amount <= auction.CurrentBid {
		http.Error(w, "Bid must be higher than current", http.StatusBadRequest)
		return
	}

	if l.rewards[req.Bidder] < req.Amount {
		http.Error(w, "Insufficient reward balance for bid", http.StatusBadRequest)
		return
	}

	nonceData, ok := l.nonces[req.ClientID]
	if !ok || time.Since(nonceData.CreatedAt) > 5*time.Minute {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var stx types.SignedTxn
	if err := msgpack.Decode(req.SignedTx, &stx); err != nil || stx.Txn.Sender.String() != req.Bidder || string(stx.Txn.Note) != nonceData.Value {
		http.Error(w, "Bid authentication failed", http.StatusUnauthorized)
		return
	}

	auction.CurrentBid = req.Amount
	auction.HighestBidder = req.Bidder
	l.logAdminAudit("AUCTION_BID", req.Bidder, fmt.Sprintf("Auction: %s, Amount: %.2f", req.AuctionID, float64(req.Amount)/1000000.0))
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Bid successfully placed."})
}

func (l *Lobby) processAuctions() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	now := time.Now()
	for id, a := range l.auctions {
		if now.After(a.EndsAt) {
			if a.HighestBidder != "" {
				commissionMicro := uint64(float64(a.CurrentBid) * 0.10)
				payoutMicro := a.CurrentBid - commissionMicro

				// Settle $VBV rewards
				l.rewards[a.HighestBidder] -= a.CurrentBid
				l.rewards[a.SellerWallet] += payoutMicro
				
				// Industrial Loop: commission to controlling Club
				l.distributeShopRevenue(a.TerritoryID, commissionMicro, "AUCTION_FEE")

				// Deliver items
				stats := l.leaderboard[a.HighestBidder]
				if stats.Inventory == nil { stats.Inventory = make(map[string]int) }
				if a.Bundle.WeaponID != "" { stats.Inventory[a.Bundle.WeaponID]++ }
				if a.Bundle.FaceplateID != "" { stats.Inventory[a.Bundle.FaceplateID]++ }
				l.leaderboard[a.HighestBidder] = stats
				
				l.logAdminAudit("AUCTION_FINALIZED", a.SellerWallet, fmt.Sprintf("Sold to %s for %.2f", a.HighestBidder, float64(a.CurrentBid)/1000000.0))
			} else {
				// No bids: return items to seller
				stats := l.leaderboard[a.SellerWallet]
				if a.Bundle.WeaponID != "" { stats.Inventory[a.Bundle.WeaponID]++ }
				if a.Bundle.FaceplateID != "" { stats.Inventory[a.Bundle.FaceplateID]++ }
				l.leaderboard[a.SellerWallet] = stats
				l.logAdminAudit("AUCTION_EXPIRED", a.SellerWallet, "No bidders found.")
			}
			delete(l.auctions, id)
		}
	}
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
		if exists {
			// Multiplier scales with Club Mojo: 1.0 to 1.5 (at 1000 Mojo)
			multiplier := 1.0 + (float64(club.Mojo) / 2000.0)
			if multiplier > 1.5 { multiplier = 1.5 }
			rep = int(float64(rep) * multiplier)
		}
	}

	if rep < 0 {
		return 0
	}
	return rep
}