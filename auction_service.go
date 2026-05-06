package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
	l.rewards[req.Bidder] -= req.Amount

	// 2. Refund previous highest bidder (if any)
	if previousHighestBidder != "" {
		l.rewards[previousHighestBidder] += previousHighestBid
		// Notify previous bidder of refund
		l.sendToClient(l.getClientIDFromWallet(previousHighestBidder), Envelope{
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
	l.applyDynamicScaling()

	l.logAdminAudit("AUCTION_BID", req.Bidder, fmt.Sprintf("Auction: %s, Amount: %.2f, Previous Bidder: %s, Refunded: %.2f", req.AuctionID, float64(req.Amount)/1000000.0, previousHighestBidder, float64(previousHighestBid)/1000000.0))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Bid successfully placed."})
}
