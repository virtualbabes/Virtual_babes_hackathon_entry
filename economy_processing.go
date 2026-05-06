package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// processLoans checks for defaulted loans and handles collateral liquidation.
func (l *Lobby) processLoans() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	now := time.Now()

	for id, loan := range l.loans {
		if loan.Status == "active" && now.After(loan.DueAt) {
			loan.Status = "defaulted"

			// Residual Value: 15% of the loan amount is returned as Market Tokens
			tokenReward := uint64(float64(loan.LoanAmount) * 0.15)

			borrowerWallet := loan.BorrowerWallet
			borrowerStats, exists := l.leaderboard[borrowerWallet]
			if exists {
				borrowerStats.MarketTokens += tokenReward
				borrowerStats.Reputation -= 50
				if borrowerStats.Reputation < 0 {
					borrowerStats.Reputation = 0
				}
				// RECONCILE: Ensure calculated stats are in sync
				borrowerStats.Reputation = l.CalculateReputation(borrowerStats)
				l.leaderboard[borrowerWallet] = borrowerStats
				l.leaderboard[borrowerWallet] = borrowerStats

				l.sendToClient(l.getClientIDFromWallet(borrowerWallet), Envelope{
					Type:    "admin_notification",
					Payload: json.RawMessage(fmt.Sprintf(`{"text":"🚨 <b>LOAN DEFAULTED:</b> Collateral moved to Black Market. You received %.2f Market Tokens as equity."}`, float64(tokenReward)/1000000.0)),
				})
			}

			// Update playstyle on loan default (Internal call to avoid deadlock)
			l.updatePlayerPlaystyleTendenciesLocked(borrowerWallet, false, [2]int{}, []int{}, false)
			l.logAdminAudit("LOAN_LIQUIDATED", borrowerWallet, fmt.Sprintf("ID: %s, Tokens: %d", loan.ID, tokenReward))

			// Add the defaulted loan to the black market
			l.blackMarket = append(l.blackMarket, *loan)

			delete(l.loans, id)
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		}
	}
}

// processAuctions checks for expired auctions and settles them.
func (l *Lobby) processAuctions() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	now := time.Now()
	for id, a := range l.auctions {
		if now.After(a.EndsAt) {
			if a.HighestBidder != "" {
				commissionMicro := uint64(float64(a.CurrentBid) * 0.10)
				payoutMicro := a.CurrentBid - commissionMicro

				// 2. Settle $VBV rewards (deduct bid from highest bidder, pay seller)
				if l.rewards[a.HighestBidder] >= a.CurrentBid { // Ensure bidder still has funds
					// Distribute commission: 10% to the club owning the Art Gallery territory, else to faucet
					artGalleryClub := l.getClubByTerritoryID(a.TerritoryID) // a.TerritoryID is "the_art_gallery"
					if artGalleryClub != nil {
						artGalleryClub.Treasury += float64(commissionMicro) / 1000000.0
						artGalleryClub.LastActivity = time.Now() // Update club activity
						log.Printf("[AUCTION] Club %s (%s) earned %.2f $VBV commission from auction %s.\n", artGalleryClub.Name, artGalleryClub.ID, float64(commissionMicro)/1000000.0, a.ID)
					} else {
						// Fallback: If no club owns the Art Gallery, the commission goes to the faucet
						l.faucetBalance += float64(commissionMicro) / 1000000.0
						log.Printf("[AUCTION] No club owns 'the_art_gallery'. Commission from auction %s added to faucet.\n", a.ID)
					}

					l.rewards[a.HighestBidder] -= a.CurrentBid
					l.rewards[a.SellerWallet] += payoutMicro

					// 3. Apply dynamic scaling due to faucet balance change
					l.applyDynamicScalingLocked() // Call the locked version

					// Deliver items
					stats := l.leaderboard[a.HighestBidder]
					if stats.Inventory == nil {
						stats.Inventory = make(map[string]int)
					}
					if a.Bundle.CardID != 0 {
						stats.Inventory[fmt.Sprintf("CARD-%d", a.Bundle.CardID)]++
					}
					if a.Bundle.WeaponID != "" {
						stats.Inventory[a.Bundle.WeaponID]++
					}
					if a.Bundle.FaceplateID != "" {
						stats.Inventory[a.Bundle.FaceplateID]++
					}
					l.leaderboard[a.HighestBidder] = stats

					l.logAdminAudit("AUCTION_FINALIZED", a.SellerWallet, fmt.Sprintf("Sold to %s for %.2f", a.HighestBidder, float64(a.CurrentBid)/1000000.0))
					l.sendToClient(l.getClientIDFromWallet(a.HighestBidder), Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🎉 <b>AUCTION WON:</b> You won the auction for %s!"}`, a.Bundle.WeaponID))})
					l.sendToClient(l.getClientIDFromWallet(a.SellerWallet), Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>AUCTION SETTLED:</b> Your item was sold for %.2f $VBV!"}`, float64(payoutMicro)/1000000.0))})
				} else {
					// Bidder no longer has funds, return item to seller
					log.Printf("[AUCTION] Bidder %s for auction %s has insufficient funds. Returning item to seller %s.\n", a.HighestBidder, a.ID, a.SellerWallet)
					stats := l.leaderboard[a.SellerWallet]
					if stats.Inventory == nil {
						stats.Inventory = make(map[string]int)
					}
					if a.Bundle.CardID != 0 {
						stats.Inventory[fmt.Sprintf("CARD-%d", a.Bundle.CardID)]++
					}
					if a.Bundle.WeaponID != "" {
						stats.Inventory[a.Bundle.WeaponID]++
					}
					if a.Bundle.FaceplateID != "" {
						stats.Inventory[a.Bundle.FaceplateID]++
					}
					l.leaderboard[a.SellerWallet] = stats
					l.logAdminAudit("AUCTION_FAILED_BIDDER_FUNDS", a.SellerWallet, fmt.Sprintf("Auction: %s, Bidder: %s", a.ID, a.HighestBidder))
					l.sendToClient(l.getClientIDFromWallet(a.SellerWallet), Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"⚠️ <b>AUCTION FAILED:</b> Bidder had insufficient funds. Item returned."}`))})
				}
			} else {
				// No bids: return items to seller
				stats := l.leaderboard[a.SellerWallet]
				if stats.Inventory == nil {
					stats.Inventory = make(map[string]int)
				}
				if a.Bundle.CardID != 0 {
					stats.Inventory[fmt.Sprintf("CARD-%d", a.Bundle.CardID)]++
				}
				if a.Bundle.WeaponID != "" {
					stats.Inventory[a.Bundle.WeaponID]++
				}
				if a.Bundle.FaceplateID != "" {
					stats.Inventory[a.Bundle.FaceplateID]++
				}
				l.leaderboard[a.SellerWallet] = stats
				l.logAdminAudit("AUCTION_EXPIRED", a.SellerWallet, "No bidders found.")
				l.sendToClient(l.getClientIDFromWallet(a.SellerWallet), Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"😔 <b>AUCTION EXPIRED:</b> No bids received. Item returned."}`)})
			}
			delete(l.auctions, id)
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		}
	}
}
