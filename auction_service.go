//go:build !js && !wasm

package main

import (
	"context"
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
	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/encoding/msgpack"
	"github.com/algorand/go-algorand-sdk/v2/mnemonic"
	"github.com/algorand/go-algorand-sdk/v2/transaction"
	"github.com/algorand/go-algorand-sdk/v2/types"
)

// AuctionService encapsulates logic for high-fidelity asset sales and Art Gallery operations.
// PILLAR 5: Stateless Service Design.
type AuctionService struct{}

// HandleGetAuctions returns all active listings in the Art Gallery.
func (s *AuctionService) HandleGetAuctions(l *Lobby, w http.ResponseWriter, r *http.Request) {
	l.mutex.RLock()
	var list []*Auction
	for _, a := range l.auctions {
		list = append(list, a)
	}
	l.mutex.RUnlock()

	// Lazy Resolution: We resolve names outside the global state lock to prevent
	// display latency. Since we're iterating pointers, updating 'a' populates the
	// master record for all future requests.
	for _, a := range list {
		if a.SellerName == "" {
			a.SellerName = l.oracleService.ResolveEnvoiName(l, a.SellerWallet)
		}
		if a.HighestBidder != "" && (a.HighestBidderName == "" || a.HighestBidderName == a.HighestBidder) {
			a.HighestBidderName = l.oracleService.ResolveEnvoiName(l, a.HighestBidder)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// HandleCreateAuction allows a player to list a bundle for $VBV.
func (s *AuctionService) HandleCreateAuction(l *Lobby, w http.ResponseWriter, r *http.Request) {
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

	// PILLAR 2: Financial Guardrails
	if req.StartPrice <= 0 || req.Duration <= 0 {
		http.Error(w, "Invalid starting price or duration", http.StatusBadRequest)
		return
	}

	if req.Bundle.CardID == 0 && req.Bundle.WeaponID == "" && req.Bundle.FaceplateID == "" {
		http.Error(w, "Auction bundle cannot be empty", http.StatusBadRequest)
		return
	}

	// PILLAR 3: Identity Normalization.
	targetWallet := strings.ToLower(req.Wallet)
	sellerName := l.oracleService.ResolveEnvoiName(l, targetWallet)

	l.mutex.RLock()
	nonceData, nonceExists := l.nonces[req.ClientID]
	l.mutex.RUnlock()
	if !nonceExists || time.Since(nonceData.CreatedAt) > 5*time.Minute {
		http.Error(w, "Unauthorized: Nonce expired", http.StatusUnauthorized)
		return
	}

	var stx types.SignedTxn
	if err := msgpack.Decode(req.SignedTx, &stx); err != nil || !strings.EqualFold(stx.Txn.Sender.String(), targetWallet) || string(stx.Txn.Note) != nonceData.Value {
		http.Error(w, "Unauthorized: Signature mismatch", http.StatusUnauthorized)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	// PILLAR 2: Inventory Integrity. Ensure stats are ready for verification.
	stats := l.leaderboard[targetWallet]
	if req.Bundle.CardID != 0 {
		cardKey := fmt.Sprintf("CARD-%d", req.Bundle.CardID)
		if stats.Inventory == nil || stats.Inventory[cardKey] <= 0 {
			http.Error(w, "Card not in inventory", http.StatusBadRequest)
			return
		}
	}

	// PILLAR 5: Modular Orchestration.
	// Utilize authoritative helper for escrow removal to ensure consistent key pruning.
	s.TransferBundleItems(l, targetWallet, req.Bundle, false)

	auctionID := fmt.Sprintf("AUC-%d", time.Now().UnixNano())
	startPriceMicro := uint64(req.StartPrice*1000000 + 0.5)
	l.auctions[auctionID] = &Auction{
		ID:           auctionID,
		SellerWallet: targetWallet,
		SellerName:   sellerName,
		Bundle:       req.Bundle,
		CurrentBid:   startPriceMicro,
		EndsAt:       time.Now().Add(time.Duration(req.Duration) * time.Hour),
		TerritoryID:  "the_archive", // Fixed territory for Art Gallery Commissions
	}

	// PILLAR 2: High-Finance Audit. Use Locked variant to prevent recursive deadlock.
	l.logAdminAuditLocked("AUCTION_CREATED", targetWallet, fmt.Sprintf("ID: %s, Price: %.2f", auctionID, float64(startPriceMicro)/1000000.0))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(l.auctions[auctionID])
}

// HandlePlaceBid processes inbound currency for competitive item acquisition.
func (s *AuctionService) HandlePlaceBid(l *Lobby, w http.ResponseWriter, r *http.Request) {
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

	// PILLAR 5: Performance & Safety.
	// Resolve names before acquiring the global lock to prevent I/O blocking and deadlocks.
	targetWallet := strings.ToLower(req.Bidder)
	bidderName := l.oracleService.ResolveEnvoiName(l, targetWallet)

	l.mutex.Lock()
	defer l.mutex.Unlock()

	auction, exists := l.auctions[req.AuctionID]
	if !exists || time.Now().After(auction.EndsAt) {
		http.Error(w, "Auction expired or not found", http.StatusNotFound)
		return
	}

	// PILLAR 2: Minimum Bid Increment.
	// Enforce a 1% minimum increment (with 0.01 VBV floor) to ensure meaningful auction velocity.
	minIncrementMicro := (auction.CurrentBid + 99) / 100
	if minIncrementMicro < 10000 { minIncrementMicro = 10000 } // 0.01 VBV floor

	if req.Amount < auction.CurrentBid + minIncrementMicro {
		http.Error(w, fmt.Sprintf("Bid increment too low. Minimum bid: %.2f $VBV", float64(auction.CurrentBid + minIncrementMicro)/1000000.0), http.StatusBadRequest)
		return
	}

	// PILLAR 2: Non-Custodial Bidding Verification.
	// Check if the bidder has approved the vault to pull the VBV on-chain.
	// This allows for high-value bidding without depositing rewards first.
	approvedAmt, _ := l.oracleService.CheckAssetApproval(l, "VOI", targetWallet, l.vaultAddress, l.rewardAssetID)

	hasVirtual := l.playerBalances[targetWallet] >= req.Amount
	hasApproved := approvedAmt >= req.Amount

	if !hasVirtual && !hasApproved {
		http.Error(w, "Insufficient funds: Either internal rewards or on-chain approval required.", http.StatusPaymentRequired)
		return
	}

	// Store previous highest bidder and their bid for refund
	previousHighestBidder := auction.HighestBidder
	previousHighestBid := auction.CurrentBid
	wasApproved := auction.HighestBidIsApproved

	nonceData, ok := l.nonces[req.ClientID]
	if !ok || time.Since(nonceData.CreatedAt) > 5*time.Minute {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var stx types.SignedTxn
	if err := msgpack.Decode(req.SignedTx, &stx); err != nil || !strings.EqualFold(stx.Txn.Sender.String(), targetWallet) || string(stx.Txn.Note) != nonceData.Value {
		http.Error(w, "Bid authentication failed", http.StatusUnauthorized)
		return
	}

	// 1. Payout Handling.
	// If the user has sufficient internal rewards, we escrow them immediately to ensure 
	// the bid is "locked." If they are using an approval, we skip the immediate 
	// deduction and verify/pull at settlement.
	if hasVirtual {
		l.playerBalances[targetWallet] -= req.Amount
		l.logAdminAuditLocked("AUCTION_BID_INTERNAL", targetWallet, fmt.Sprintf("Virtual deduction: %.2f", float64(req.Amount)/1000000.0))
		auction.HighestBidIsApproved = false
	} else {
		l.logAdminAuditLocked("AUCTION_BID_APPROVED", targetWallet, fmt.Sprintf("Non-custodial approval verified: %.2f", float64(approvedAmt)/1000000.0))
		auction.HighestBidIsApproved = true
	}

	// 2. Refund previous highest bidder (if any)
	if previousHighestBidder != "" {
		// PILLAR 2: "Free Refund" Guard. Only refund if the previous bid was virtual.
		if !wasApproved {
			l.playerBalances[previousHighestBidder] += previousHighestBid
			// Notify previous bidder of refund
			l.sendToClientLocked(l.getClientIDFromWalletLocked(previousHighestBidder), Envelope{
				Type:    "admin_notification",
				Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>AUCTION REFUND:</b> Your bid of %.2f $VBV for auction %s has been refunded."}`, float64(previousHighestBid)/1000000.0, req.AuctionID)),
			})
		}
	}

	auction.CurrentBid = req.Amount
	auction.HighestBidder = targetWallet
	auction.HighestBidderName = bidderName

	// PILLAR 2: Ledger Integrity.
	// Physical vault balance (faucetBalance) remains unchanged as funds move between
	// virtual liability categories. Scaling is recalculated based on the new liability sum.
	l.applyDynamicScalingLocked()

	l.logAdminAuditLocked("AUCTION_BID", targetWallet, fmt.Sprintf("Auction: %s, Amount: %.2f", req.AuctionID, float64(req.Amount)/1000000.0))
	msg := l.getLobbyUpdateMsgLocked()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Bid successfully placed."})

	go func() { l.broadcast <- msg }()
}

// ProcessAuctions handles auction expiration and settlement.
func (s *AuctionService) ProcessAuctions(l *Lobby) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	now := time.Now()
	anyProcessed := false

	for id, auction := range l.auctions {
		if now.After(auction.EndsAt) {
			// Auction expired
			if auction.HighestBidder != "" {
				// Settle auction: transfer item to winner, pay seller
				// 1. Transfer items to winner
				s.TransferBundleItems(l, auction.HighestBidder, auction.Bundle, true)

				// 2. Calculate commission (10%) and net payout to seller
				// PILLAR 1: Precision Rounding for the Industrial Loop.
				commissionMicro := (auction.CurrentBid*10 + 50) / 100 // Round to nearest micro-unit
				netPayoutToSellerMicro := auction.CurrentBid - commissionMicro
				amountBase := float64(auction.CurrentBid) / 1000000.0

				// PILLAR 3: Financial Proof.
				// Record auction settlement on-chain for the audit trail, capturing the full bundle.
				settleDetails := map[string]interface{}{
					"id":      id,
					"winner":  auction.HighestBidder,
					"seller":  auction.SellerWallet,
					"bundle":  auction.Bundle,
					"amount":  amountBase,
					"sector":  auction.TerritoryID, // PILLAR 3: Localized Economic Auditing.
					"ts":      now.Unix(),
				}

				// 3. Pay seller
				if auction.HighestBidIsApproved {
					// PILLAR 2: Non-Custodial Settlement.
					// Winner used an on-chain approval. Pull the funds now.
					if err := s.pullApprovedTokens(l, auction.HighestBidder, auction.CurrentBid); err != nil {
						log.Printf("[AUCTION ERROR] Critical settlement failure: Failed to pull tokens from %s: %v\n", auction.HighestBidder, err)
						// We proceed to pay the seller virtual rewards, maintaining the "Token-Sink" promise.
					}
				}
				l.playerBalances[auction.SellerWallet] += netPayoutToSellerMicro

				// 4. Distribute commission
				artGalleryClub := l.getClubByTerritoryID(auction.TerritoryID) // "the_archive"
				if artGalleryClub != nil {
					totalCommissionBase := float64(commissionMicro) / 1000000.0

					// Club.TreasuryMicro is the authoritative uint64 vault field.
					artGalleryClub.TreasuryMicro += commissionMicro

					artGalleryClub.LastActivity = now
					l.logAdminAuditLocked("AUCTION_COMMISSION_TO_CLUB", artGalleryClub.ID, fmt.Sprintf("Auction: %s, Commission: %.2f", id, totalCommissionBase))
				}

				// PILLAR 2: Dynamic Scaling Refresh.
				// Re-calculate the reward ratio to reflect the shift from unreserved
				// liquidity into seller liabilities and club reserves.
				l.applyDynamicScalingLocked()

				// 6. Track Achievement: ART_COLLECTOR (3 Wins)
				winnerStats := l.leaderboard[auction.HighestBidder]
				winnerStats.AuctionsWon++
				l.leaderboard[auction.HighestBidder] = winnerStats
				if winnerStats.AuctionsWon >= 3 {
					l.achievementService.UnlockAchievementLocked(l, auction.HighestBidder, "ART_COLLECTOR")
				}

				l.logAdminAuditLocked("AUCTION_SETTLED", auction.HighestBidder, fmt.Sprintf("Auction: %s, Winner: %s, Seller: %s, Amount: %.2f (Net: %.2f, Commission: %.2f)",
					id, auction.HighestBidder, auction.SellerWallet, amountBase, float64(netPayoutToSellerMicro)/1000000.0, float64(commissionMicro)/1000000.0))

				// PILLAR 3: Financial Proof. Record high-value auction settlement on-chain.
				if amountBase >= 100.0 {
					// Record on-chain settlement for immutable verification
					go func(sd interface{}) {
						jsonPayload, _ := json.Marshal(sd)
						l.sendNoteTx(fmt.Sprintf("VBT_AUCTION_SETTLE:%s", string(jsonPayload)))
					}(settleDetails)
				}

				// Notify winner and seller
				l.sendToClientLocked(l.getClientIDFromWalletLocked(auction.HighestBidder), Envelope{
					Type:    "admin_notification",
					Payload: json.RawMessage(fmt.Sprintf(`{"text":"🎉 <b>AUCTION WON:</b> You won auction %s for %.2f $VBV! Items added to inventory."}`, id, float64(auction.CurrentBid)/1000000.0)),
				})
				l.sendToClientLocked(l.getClientIDFromWalletLocked(auction.SellerWallet), Envelope{
					Type:    "admin_notification",
					Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>AUCTION SOLD:</b> Your auction %s sold for %.2f $VBV (Net: %.2f after commission)."}`, id, float64(auction.CurrentBid)/1000000.0, float64(netPayoutToSellerMicro)/1000000.0)),
				})
			} else {
				// No bids, return item to seller
				s.TransferBundleItems(l, auction.SellerWallet, auction.Bundle, true)
				l.logAdminAuditLocked("AUCTION_EXPIRED", auction.SellerWallet, fmt.Sprintf("Auction: %s, No bids, items returned.", id))
				l.sendToClientLocked(l.getClientIDFromWalletLocked(auction.SellerWallet), Envelope{
					Type:    "admin_notification",
					Payload: json.RawMessage(fmt.Sprintf(`{"text":"📦 <b>AUCTION EXPIRED:</b> Your auction %s received no bids. Items returned to inventory."}`, id)),
				})
			}
			delete(l.auctions, id)
			anyProcessed = true
		}
	}

	if anyProcessed {
		// PILLAR 5: Snapshot Pattern. Prevent deadlocks by capturing msg before goroutine.
		msg := l.getLobbyUpdateMsgLocked()
		go func() { l.broadcast <- msg }()
	}

}

// pullApprovedTokens executes an on-chain transferFrom to move tokens from the winner to the vault.
// PILLAR 2: High-Finance.
func (s *AuctionService) pullApprovedTokens(l *Lobby, from string, amount uint64) error {
	l.mutex.RLock()
	voiConfig, _ := l.availableNetworks["Voi Mainnet"]
	vaultAddr := l.vaultAddress
	rewardAsset := l.rewardAssetID
	l.mutex.RUnlock()

	if len(voiConfig.NodeURLs) == 0 {
		return fmt.Errorf("no nodes available")
	}

	client, _ := algod.MakeClient(voiConfig.NodeURLs[0], "")
	pk, _ := mnemonic.ToPrivateKey(os.Getenv("FAUCET_MNEMONIC"))
	faucetAccount, _ := crypto.AccountFromPrivateKey(pk)
	sp, _ := client.SuggestedParams().Do(context.Background())

	// ARC-200 transferFrom(address from, address to, uint256 value) -> selector: 0x23b872dd
	methodSelector := []byte{0x23, 0xb8, 0x72, 0xdd}
	fromAddr, _ := types.DecodeAddress(from)
	toAddr, _ := types.DecodeAddress(vaultAddr)
	amountBytes := make([]byte, 32)
	new(big.Int).SetUint64(amount).FillBytes(amountBytes)

	appArgs := [][]byte{
		methodSelector,
		fromAddr[:],
		toAddr[:],
		amountBytes,
	}

	appID, _ := strconv.ParseUint(rewardAsset, 10, 64)
	senderAddr, _ := types.DecodeAddress(vaultAddr)
	note := []byte(fmt.Sprintf("VBT_AUCTION_PULL:{\"from\":\"%s\",\"amt\":%d}", from, amount))

	txn, _ := transaction.MakeApplicationNoOpTx(appID, appArgs, nil, nil, nil, sp, senderAddr, note, types.Digest{}, [32]byte{}, types.Address{})
	txid, stxn, err := crypto.SignTransaction(faucetAccount.PrivateKey, txn)
	if err != nil { return err }
	
	if _, err := client.SendRawTransaction(stxn).Do(context.Background()); err != nil {
		return err
	}
	log.Printf("[AUCTION] Pulled %d micro-tokens from %s. TxID: %s\n", amount, from, txid)

	l.mutex.Lock()
	l.faucetBalanceMicro += amount
	l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0

	// PILLAR 2: Real-time Reconciliation.
	// Capture the non-custodial auction settlement in the authoritative audit trail.
	if l.tokenSinkRouter != nil && l.tokenSinkRouter.Audit != nil {
		// PILLAR 2: Structural Audit. Report zero siphon for standard on-chain pulls.
		_ = l.tokenSinkRouter.Audit.InterceptAndAudit("AUCTION_PULL", amount, amount, 0, 0, 0)
	}

	l.mutex.Unlock()

	return nil
}

// transferBundleItems handles adding or removing items from a player's inventory.
// If add is true, items are added. If add is false, items are removed.
// It assumes the lobby mutex is held.
func (s *AuctionService) TransferBundleItems(l *Lobby, wallet string, bundle CardBundle, add bool) {
	l.ensurePlayerStatsMapsInitialized(wallet)
	stats := l.leaderboard[wallet]

	// PILLAR 5: Defensive Map Handling.
	// Initialize Inventory if nil to prevent panics during deductions, 
	// ensuring parity with the global initialization helper.
	if stats.Inventory == nil {
		stats.Inventory = make(map[string]int)
	}

	if bundle.CardID != 0 {
		cardKey := fmt.Sprintf("CARD-%d", bundle.CardID)
		if add {
			stats.Inventory[cardKey]++
		} else {
			stats.Inventory[cardKey]--
			if stats.Inventory[cardKey] <= 0 {
				delete(stats.Inventory, cardKey)
			}
		}
	}
	if bundle.WeaponID != "" {
		if add {
			stats.Inventory[bundle.WeaponID]++
		} else {
			stats.Inventory[bundle.WeaponID]--
			if stats.Inventory[bundle.WeaponID] <= 0 {
				delete(stats.Inventory, bundle.WeaponID)
			}
		}
	}
	if bundle.FaceplateID != "" {
		if add {
			stats.Inventory[bundle.FaceplateID]++
		} else {
			stats.Inventory[bundle.FaceplateID]--
			if stats.Inventory[bundle.FaceplateID] <= 0 {
				delete(stats.Inventory, bundle.FaceplateID)
			}
		}
	}
	l.leaderboard[wallet] = stats
}
