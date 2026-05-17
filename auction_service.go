//go:build !js || !wasm

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/encoding/msgpack"
	"github.com/algorand/go-algorand-sdk/v2/types"
)

// handleGetAuctions returns all active listings in the Art Gallery.
func (l *Lobby) handleGetAuctions(w http.ResponseWriter, r *http.Request) {
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
			a.SellerName = l.ResolveEnvoiName(a.SellerWallet)
		}
		if a.HighestBidder != "" && (a.HighestBidderName == "" || a.HighestBidderName == a.HighestBidder) {
			a.HighestBidderName = l.ResolveEnvoiName(a.HighestBidder)
		}
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

	// PILLAR 2: Financial Guardrails
	if req.StartPrice <= 0 || req.Duration <= 0 {
		http.Error(w, "Invalid starting price or duration", http.StatusBadRequest)
		return
	}

	if req.Bundle.CardID == 0 && req.Bundle.WeaponID == "" && req.Bundle.FaceplateID == "" {
		http.Error(w, "Auction bundle cannot be empty", http.StatusBadRequest)
		return
	}

	sellerName := l.ResolveEnvoiName(req.Wallet)

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

	// PILLAR 2: Inventory Integrity.
	// Ensure maps are initialized before checking item existence to prevent nil access panics.
	l.ensurePlayerStatsMapsInitialized(req.Wallet)

	stats := l.leaderboard[req.Wallet]
	if req.Bundle.CardID != 0 {
		cardKey := fmt.Sprintf("CARD-%d", req.Bundle.CardID)
		if stats.Inventory[cardKey] <= 0 {
			http.Error(w, "Card not in inventory", http.StatusBadRequest)
			return
		}
	}
	if req.Bundle.WeaponID != "" && stats.Inventory[req.Bundle.WeaponID] <= 0 {
		http.Error(w, "Weapon not in inventory", http.StatusBadRequest)
		return
	}
	if req.Bundle.FaceplateID != "" && stats.Inventory[req.Bundle.FaceplateID] <= 0 {
		http.Error(w, "Faceplate not in inventory", http.StatusBadRequest)
		return
	}

	// Escrow items
	if req.Bundle.CardID != 0 {
		stats.Inventory[fmt.Sprintf("CARD-%d", req.Bundle.CardID)]--
	}
	if req.Bundle.WeaponID != "" {
		stats.Inventory[req.Bundle.WeaponID]--
	}
	if req.Bundle.FaceplateID != "" {
		stats.Inventory[req.Bundle.FaceplateID]--
	}
	l.leaderboard[req.Wallet] = stats

	auctionID := fmt.Sprintf("AUC-%d", time.Now().UnixNano())
	l.auctions[auctionID] = &Auction{
		ID:           auctionID,
		SellerWallet: req.Wallet,
		SellerName:   sellerName,
		Bundle:       req.Bundle,
		CurrentBid:   uint64(req.StartPrice * 1000000),
		EndsAt:       time.Now().Add(time.Duration(req.Duration) * time.Hour),
		TerritoryID:  "the_archive", // Fixed territory for Art Gallery Commissions
	}

	// PILLAR 2: High-Finance Audit. Use Locked variant to prevent recursive deadlock.
	l.logAdminAuditLocked("AUCTION_CREATED", req.Wallet, fmt.Sprintf("ID: %s, Price: %.2f", auctionID, req.StartPrice))
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

	if l.playerBalances[req.Bidder] < req.Amount {
		http.Error(w, "Insufficient reward balance for bid", http.StatusBadRequest)
		return
	}

	// Store previous highest bidder and their bid for refund
	previousHighestBidder := auction.HighestBidder
	previousHighestBid := auction.CurrentBid
	bidderName := l.ResolveEnvoiName(req.Bidder)

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

	// 1. Deduct new bid from current bidder
	l.playerBalances[req.Bidder] -= req.Amount

	// 2. Refund previous highest bidder (if any)
	if previousHighestBidder != "" {
		l.playerBalances[previousHighestBidder] += previousHighestBid
		// Notify previous bidder of refund
		l.sendToClientLocked(l.getClientIDFromWalletLocked(previousHighestBidder), Envelope{
			Type:    "admin_notification",
			Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>AUCTION REFUND:</b> Your bid of %.2f $VBV for auction %s has been refunded."}`, float64(previousHighestBid)/1000000.0, req.AuctionID)),
		})
	}

	auction.CurrentBid = req.Amount
	auction.HighestBidder = req.Bidder
	auction.HighestBidderName = bidderName

	// 3. Update Faucet Balance and apply dynamic scaling (as funds move through the system)
	// This assumes bids are held by the faucet during the auction.
	// If bids are held by the auction contract, this would be different.
	// For now, we simulate the funds moving into and out of the general reward pool.
	l.faucetBalance += float64(req.Amount-previousHighestBid) / 1000000.0
	l.applyDynamicScalingLocked()

	l.logAdminAuditLocked("AUCTION_BID", req.Bidder, fmt.Sprintf("Auction: %s, Amount: %.2f", req.AuctionID, float64(req.Amount)/1000000.0))
	msg := l.getLobbyUpdateMsgLocked()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Bid successfully placed."})

	go func() { l.broadcast <- msg }()
}

// processAuctions handles auction expiration and settlement.
func (l *Lobby) processAuctions() {
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
				l.transferBundleItems(auction.HighestBidder, auction.Bundle, true)

				// 2. Calculate commission (10%) and net payout to seller
				// PILLAR 1: Precision Rounding for the Industrial Loop.
				commissionMicro := (auction.CurrentBid*10 + 50) / 100 // Round to nearest micro-unit
				netPayoutToSellerMicro := auction.CurrentBid - commissionMicro

				// PILLAR 3: Financial Proof.
				// Record auction settlement on-chain for the audit trail.
				settleDetails := map[string]interface{}{
					"id":      id,
					"winner":  auction.HighestBidder,
					"seller":  auction.SellerWallet,
					"card_id": auction.Bundle.CardID,
					"amount":  float64(auction.CurrentBid) / 1000000.0,
					"ts":      now.Unix(),
				}

				// 3. Pay seller
				l.playerBalances[auction.SellerWallet] += netPayoutToSellerMicro

				// 4. Distribute commission
				artGalleryClub := l.getClubByTerritoryID(auction.TerritoryID) // "the_art_gallery"
				if artGalleryClub != nil {
					artGalleryClub.Treasury += float64(commissionMicro) / 1000000.0
					artGalleryClub.LastActivity = now
					l.logAdminAuditLocked("AUCTION_COMMISSION_TO_CLUB", artGalleryClub.ID, fmt.Sprintf("Auction: %s, Commission: %.2f", id, float64(commissionMicro)/1000000.0))
				} else {
					// If no club owns the Art Gallery, commission goes to the Faucet
					l.faucetBalance += float64(commissionMicro) / 1000000.0
					l.logAdminAuditLocked("AUCTION_COMMISSION_TO_FAUCET", "GLOBAL", fmt.Sprintf("Auction: %s, Commission: %.2f", id, float64(commissionMicro)/1000000.0))
				}

				// 5. Update Faucet balance and dynamic scaling (as funds move through the system)
				// The bid amount was already deducted from the bidder's rewards when placed.
				// The net payout to seller and commission distribution effectively re-routes these funds.
				// No direct change to l.faucetBalance for the full bid, only for the commission if it goes there.
				l.applyDynamicScalingLocked() // Re-evaluate scaling due to potential faucet change

				// 6. Track Achievement: ART_COLLECTOR (3 Wins)
				winnerStats := l.leaderboard[auction.HighestBidder]
				winnerStats.AuctionsWon++
				l.leaderboard[auction.HighestBidder] = winnerStats
				if winnerStats.AuctionsWon >= 3 {
					l.unlockAchievementLocked(auction.HighestBidder, "ART_COLLECTOR")
				}

				l.logAdminAuditLocked("AUCTION_SETTLED", auction.HighestBidder, fmt.Sprintf("Auction: %s, Winner: %s, Seller: %s, Amount: %.2f (Net: %.2f, Commission: %.2f)",
					id, auction.HighestBidder, auction.SellerWallet, float64(auction.CurrentBid)/1000000.0, float64(netPayoutToSellerMicro)/1000000.0, float64(commissionMicro)/1000000.0))

				// Record on-chain settlement for immutable verification
				go func(sd interface{}) {
					jsonPayload, _ := json.Marshal(sd)
					l.sendNoteTx(fmt.Sprintf("VBT_AUCTION_SETTLE:%s", string(jsonPayload)))
				}(settleDetails)

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
				l.transferBundleItems(auction.SellerWallet, auction.Bundle, true)
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
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	}
}

// transferBundleItems handles adding or removing items from a player's inventory.
// If add is true, items are added. If add is false, items are removed.
// It assumes the lobby mutex is held.
func (l *Lobby) transferBundleItems(wallet string, bundle CardBundle, add bool) {
	l.ensurePlayerStatsMapsInitialized(wallet)
	stats := l.leaderboard[wallet]

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
