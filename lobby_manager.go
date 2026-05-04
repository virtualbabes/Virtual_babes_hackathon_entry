package main

import (
	"crypto/ed25519"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings" // For Solana verification
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/gorilla/websocket"
	"github.com/mr-tron/base58" // For Solana address decoding
)

var safeAvatarPool = []string{
	"Cards/Alana.webp",
	"Cards/Bella.webp",
	"Cards/Clohey.webp",
	"Cards/Ellie.webp",
	"Cards/Fran.webp",
}

const linkedWalletsFileName = "linked_wallets.json"

func (c *Client) allowMessage() bool {
	c.msgMutex.Lock()
	defer c.msgMutex.Unlock()
	now := time.Now()
	const window = 10 * time.Second
	const limit = 30
	var active []time.Time
	for _, t := range c.messageTimestamps {
		if now.Sub(t) < window {
			active = append(active, t)
		}
	}
	if len(active) >= limit {
		return false
	}
	c.messageTimestamps = append(active, now)
	return true
}

func (c *Client) readPump() {
	defer func() {
		c.lobby.unregister <- c
		c.conn.Close()
	}()
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		if !c.allowMessage() {
			continue
		}
		var env Envelope
		if err := json.Unmarshal(message, &env); err != nil {
			continue
		}
		env.FromID = c.id
		finalMsg, _ := json.Marshal(env)
		c.lobby.broadcast <- finalMsg
	}
}

func (c *Client) writePump() {
	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}

func (l *Lobby) run() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	matchmakingTicker := time.NewTicker(5 * time.Second)
	defer matchmakingTicker.Stop()
	vaultCheckTicker := time.NewTicker(5 * time.Minute)
	defer vaultCheckTicker.Stop()
	healthTicker := time.NewTicker(10 * time.Minute)
	defer healthTicker.Stop()
	cacheSaveTicker := time.NewTicker(15 * time.Minute)
	defer cacheSaveTicker.Stop()

	go l.refreshGlobalLeaderboard()

	for {
		select {
		case <-ticker.C:
			l.cleanupNonces()
			l.processAuctions()
			l.processPlaystyleDecay()      // New: Decay playstyle tendencies
			l.processRumors()              // New: Check for expired rumors
			l.processLoans()               // New: Check for defaulted loans
			l.processMojoDecay()           // New: Penalize stagnant clubs
			l.processInsuranceRecovery()   // New: Check for expired kidnappings
			go l.observeGlobalSentiments() // Pillar 3: Aggregate meta trends
		case <-matchmakingTicker.C:
			l.processMatchmaking()
		case <-healthTicker.C:
			l.mutex.RLock()
			isOver := time.Since(l.seasonStart) > 30*24*time.Hour
			l.mutex.RUnlock()
			if isOver {
				go l.archiveSeason()
				go l.refreshRegionalRoles() // Verify ranks on rollover
			}
			go l.broadcastHealthReport()
		case <-cacheSaveTicker.C:
			go l.savePersistentCardCache()
			go l.saveRegisteredTxIDs()
			go l.saveLinkedWallets()
		case <-vaultCheckTicker.C:
			go l.checkVaultBalanceOnChain() // Monitor $VBV Reward Pool
			go l.checkNativeVaultBalanceOnChain()
		case client := <-l.register:
			l.mutex.Lock()
			l.clients[client.id] = client
			msg := l.getLobbyUpdateMsgLocked()
			l.mutex.Unlock()
			l.broadcast <- msg
		case client := <-l.unregister:
			l.handleUnregister(client)
		case message := <-l.broadcast:
			l.handleBroadcast(message)
		}
	}
}

func (l *Lobby) handleGameProtocol(env *Envelope, rawMsg []byte) {
	switch env.Type {
	case "register_wallet":
		var data struct {
			Wallet string `json:"wallet"`
		}
		json.Unmarshal(env.Payload, &data)
		l.mutex.Lock()
		l.wallets[env.FromID] = data.Wallet
		l.ensurePlayerStatsMapsInitialized(data.Wallet) // Ensure maps are initialized

		// Trigger NPC Welcome Commentary if they have a distinct style
		go l.generateNPCCommentary(env.FromID, "LOBBY_ENTRY")

		stats := l.leaderboard[data.Wallet]
		l.mutex.Unlock()
		go l.syncStatsFromBlockchain(env.FromID, data.Wallet)
		portfolioPayload, _ := json.Marshal(stats.Portfolio)
		l.sendToClient(env.FromID, Envelope{Type: "portfolio_update", Payload: portfolioPayload})
		if l.isAdminWallet(data.Wallet) {
			if c, ok := l.clients[env.FromID]; ok {
				c.isAdmin = true
				msg := l.getLobbyUpdateMsgLocked()
				go func() { l.broadcast <- msg }()
			}
		}
	case "register_avatar":
		var data struct {
			URL, Gloat string `json:"url"`
		}
		// Also allow setting a favorite card ID here
		json.Unmarshal(env.Payload, &data)
		l.mutex.Lock()
		if c, ok := l.clients[env.FromID]; ok {
			c.avatarURL = data.URL
			c.gloat = data.Gloat
		}
		msg := l.getLobbyUpdateMsgLocked()
		l.mutex.Unlock()
		l.broadcast <- msg
	case "join_queue":
		var data struct {
			Deck           []int  `json:"deck"`
			DeckRating     string `json:"deck_rating"`
			FavoriteCardID int    `json:"favorite_card_id"` // Optional: if player explicitly set a favorite
		}
		json.Unmarshal(env.Payload, &data)
		l.mutex.Lock()
		if wallet, ok := l.wallets[env.FromID]; ok {
			l.matchmakingPool = append(l.matchmakingPool, QueueEntry{
				// ... existing code ...
				ClientID: env.FromID, Wallet: wallet, Reputation: l.leaderboard[wallet].Reputation,
				DeckRating: data.DeckRating, JoinedAt: time.Now(), // FavoriteCardID is not part of QueueEntry
			})
			l.matches[env.FromID] = &MatchState{P1ID: env.FromID, P1Deck: data.Deck}
			l.updatePlayerPlaystyleTendencies(wallet, false, [2]int{}, data.Deck, false) // Update playstyle based on deck

			go l.generateNPCCommentary(env.FromID, "MATCH_START")
		}
		l.mutex.Unlock()
		l.sendToClient(env.FromID, Envelope{Type: "matchmaking_status", Payload: json.RawMessage(`{"status":"queued"}`)})
	case "nonce_request":
		nonce := generateNonce()
		l.mutex.Lock()
		l.nonces[env.FromID] = NonceData{Value: nonce, CreatedAt: time.Now()}
		l.mutex.Unlock()
		l.sendToClient(env.FromID, Envelope{Type: "nonce_response", FromID: "SERVER", Payload: json.RawMessage(fmt.Sprintf(`{"nonce":"%s"}`, nonce))})
	case "link_wallet_request":
		var data struct {
			PrimaryAVMWallet string `json:"primary_avm_wallet"`
			LinkedAddress    string `json:"linked_address"`
			LinkedChain      string `json:"linked_chain"`
			Signature        string `json:"signature"`
			Nonce            string `json:"nonce"`
		}
		if err := json.Unmarshal(env.Payload, &data); err != nil {
			log.Printf("[LINK] Invalid link_wallet_request payload: %v\n", err)
			l.sendToClient(env.FromID, Envelope{Type: "link_wallet_response", Payload: json.RawMessage(`{"status":"error","message":"Invalid request"}`)})
			return
		}

		l.mutex.RLock()
		nonceData, exists := l.nonces[env.FromID] // Nonce is generated for the client's session
		l.mutex.RUnlock()

		if !exists || nonceData.Value != data.Nonce || time.Since(nonceData.CreatedAt) > 5*time.Minute {
			log.Printf("[LINK] Nonce verification failed for %s (linked: %s). Exists: %v, Match: %v, Expired: %v\n",
				env.FromID, data.LinkedAddress, exists, nonceData.Value == data.Nonce, time.Since(nonceData.CreatedAt) > 5*time.Minute)
			l.sendToClient(env.FromID, Envelope{Type: "link_wallet_response", Payload: json.RawMessage(fmt.Sprintf(`{"status":"error","message":"Nonce invalid or expired","address":"%s"}`, data.LinkedAddress))})
			return
		}

		var verified bool
		var verifyErr error

		switch strings.ToLower(data.LinkedChain) {
		case "eth", "poly", "evm":
			// EVM signature verification (personal_sign)
			message := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data.Nonce), data.Nonce)
			messageHash := ethcrypto.Keccak256([]byte(message))

			signatureBytes, decodeErr := hex.DecodeString(strings.TrimPrefix(data.Signature, "0x"))
			if decodeErr != nil {
				verifyErr = fmt.Errorf("invalid EVM signature format: %v", decodeErr)
				break
			}
			if len(signatureBytes) != 65 {
				verifyErr = fmt.Errorf("invalid EVM signature length: %d", len(signatureBytes))
				break
			}
			if signatureBytes[64] == 27 || signatureBytes[64] == 28 {
				signatureBytes[64] -= 27
			}

			pubKey, recoverErr := ethcrypto.SigToPub(messageHash, signatureBytes)
			if recoverErr != nil {
				verifyErr = fmt.Errorf("EVM signature recovery failed: %v", recoverErr)
				break
			}
			recoveredAddress := ethcrypto.PubkeyToAddress(*pubKey).Hex()
			if strings.ToLower(recoveredAddress) == strings.ToLower(data.LinkedAddress) {
				verified = true
			} else {
				verifyErr = fmt.Errorf("EVM signature mismatch. Recovered: %s, Expected: %s", recoveredAddress, data.LinkedAddress)
			}
		case "sol":
			// Solana signature verification (ed25519)
			// Message format: `\x19Solana Signed Message:\n` + length + message
			message := fmt.Sprintf("\x19Solana Signed Message:\n%d%s", len(data.Nonce), data.Nonce)
			messageBytes := []byte(message)

			// Decode base58 Solana address to public key bytes
			pubKeyBytes, err := base58.Decode(data.LinkedAddress)
			if err != nil {
				verifyErr = fmt.Errorf("invalid Solana address format: %v", err)
				break
			}
			if len(pubKeyBytes) != ed25519.PublicKeySize {
				verifyErr = fmt.Errorf("invalid Solana public key size: %d", len(pubKeyBytes))
				break
			}

			// Decode base64 signature
			signatureBytes, err := base64.StdEncoding.DecodeString(data.Signature)
			if err != nil {
				verifyErr = fmt.Errorf("invalid Solana signature format: %v", err)
				break
			}

			// Verify the signature
			verified = ed25519.Verify(ed25519.PublicKey(pubKeyBytes), messageBytes, signatureBytes)
			if !verified {
				verifyErr = fmt.Errorf("Solana signature verification failed")
			}
		default:
			verifyErr = fmt.Errorf("unsupported linked chain: %s", data.LinkedChain)
		}

		if !verified {
			log.Printf("[LINK] Wallet link verification failed for %s: %v\n", data.LinkedAddress, verifyErr)
			l.sendToClient(env.FromID, Envelope{Type: "link_wallet_response", Payload: json.RawMessage(fmt.Sprintf(`{"status":"error","message":"Verification failed: %s","address":"%s"}`, verifyErr.Error(), data.LinkedAddress))})
			return
		}

		l.addOrUpdateLinkedWallet(data.PrimaryAVMWallet, data.LinkedAddress, data.LinkedChain)
		l.sendToClient(env.FromID, Envelope{Type: "link_wallet_response", Payload: json.RawMessage(fmt.Sprintf(`{"status":"success","message":"Wallet linked successfully","address":"%s"}`, data.LinkedAddress))})
		log.Printf("[LINK] Successfully linked %s (%s) to primary AVM wallet %s\n", data.LinkedAddress, data.LinkedChain, data.PrimaryAVMWallet)
	case "move":
		l.mutex.RLock()
		match, ok := l.matches[env.FromID]
		l.mutex.RUnlock()
		if !ok {
			return
		}
		var move MoveData
		json.Unmarshal(env.Payload, &move)
		pIdx := 0
		if env.FromID == match.P2ID {
			pIdx = 1
		}
		l.mutex.Lock()
		if move.GridIndex >= 0 && move.GridIndex < 9 {
			match.Board[move.GridIndex] = &ServerCard{ID: move.CardID, Owner: pIdx, Power: move.Power}
			// serverCheckCaptures now returns captured cards, append them to match state
			_, flips := l.serverCheckCaptures(match, move.GridIndex, pIdx)
			match.CapturedCards = append(match.CapturedCards, flips...)
		}
		full := true
		for _, slot := range match.Board {
			if slot == nil {
				full = false
				break
			}
		}
		if full && !match.IsFinished {
			match.IsFinished = true
			l.verifyWinner(match)
		}
		l.mutex.Unlock()
	case "report_gloat":
		var data ReportGloatData
		json.Unmarshal(env.Payload, &data)
		l.mutex.RLock()
		opp, ok := l.wallets[data.OpponentClientID]
		rep, _ := l.wallets[env.FromID]
		l.mutex.RUnlock()
		if ok {
			l.logAdminAudit("REPORT_GLOAT", opp, fmt.Sprintf("Reported by %s: %s", rep, data.GloatText))
			alert, _ := json.Marshal(map[string]string{"text": fmt.Sprintf("🚨 <b>REPORT:</b> %s flagged %s", rep, opp)})
			l.broadcastToAdmins(string(alert))
		}
	case "use_item": // New, expanded item usage handler
		var data UseItemData
		if err := json.Unmarshal(env.Payload, &data); err != nil {
			log.Printf("[ITEM] Invalid use_item payload from %s: %v\n", env.FromID, err)
			return
		}

		l.mutex.Lock()
		defer l.mutex.Unlock()

		wallet, ok := l.wallets[env.FromID]
		if !ok {
			return
		}

		playerStats := l.leaderboard[wallet]
		if playerStats.Inventory == nil || playerStats.Inventory[data.ItemID] <= 0 {
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Item Use Failed: Item not found in inventory."}`)})
			return
		}

		item, itemExists := GlobalShopRegistry[data.ItemID]
		if !itemExists {
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Item Use Failed: Unknown item."}`)})
			return
		}

		// Deduct item from player's inventory
		playerStats.Inventory[data.ItemID]--
		if playerStats.Inventory[data.ItemID] == 0 {
			delete(playerStats.Inventory, data.ItemID)
		}
		l.leaderboard[wallet] = playerStats

		// Apply item effects based on ClubType or ItemID
		var notificationText string
		match, inMatch := l.matches[env.FromID]

		switch item.ClubType {
		case "Vitality": // Stamina Stim, Loyalty Pledge (affect PlayerStats or specific cards in global inventory)
			if data.TargetCardID == 0 { // Must target a specific card
				l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Item Use Failed: Vitality items require a target card."}`)})
				return
			}
			targetCard, cardExists := l.inventory[data.TargetCardID]
			if !cardExists {
				l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Item Use Failed: Target card not found."}`)})
				return
			}

			switch data.ItemID {
			case "stamina_stim":
				targetCard.Fatigue -= 20
				if targetCard.Fatigue < 0 {
					targetCard.Fatigue = 0
				}
				notificationText = fmt.Sprintf("⚡ %s's Fatigue reduced by 20!", targetCard.Name)
			case "loyalty_pledge":
				targetCard.Loyalty += 10
				if targetCard.Loyalty > 100 {
					targetCard.Loyalty = 100
				}
				notificationText = fmt.Sprintf("💖 %s's Loyalty increased by 10!", targetCard.Name)
			}
			l.updatePlayerPlaystyleTendencies(wallet, false, [2]int{}, []int{}, false)                                      // Update playstyle on item use
			l.inventory[data.TargetCardID] = targetCard                                                                     // Update global card cache
			playerStats.Playstyle.PreferredItems[data.ItemID] = playerStats.Playstyle.PreferredItems[data.ItemID]*0.9 + 1.0 // Update preferred items
			l.persistentCardCache[data.TargetCardID] = targetCard                                                           // Update persistent cache

		case "Elemental", "Tactical": // Mood Catalyst, Grounded Shield, Rule Breaker, Intel Report (affect MatchState)
			if !inMatch {
				l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Item Use Failed: This item can only be used during a match."}`)})
				return
			}
			// Delegate to battle_service for in-match effects
			playerStats.Playstyle.PreferredItems[data.ItemID] = playerStats.Playstyle.PreferredItems[data.ItemID]*0.9 + 1.0 // Update preferred items
			l.updatePlayerPlaystyleTendencies(wallet, true, [2]int{}, []int{}, false)                                       // Update playstyle on item use in match
			l.applyItemEffectToMatch(match, env.FromID, data.ItemID, data.TargetCardID, data.TargetGridIndex)
			notificationText = fmt.Sprintf("✨ %s activated!", item.Name)

		case "Hardware": // Traps: tripwire, sentry_turret, guard_dog
			if playerStats.JobRole != "Security" || playerStats.EmployerClubID == "" {
				l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Deployment Failed: Security role required."}`)})
				return
			}

			targetClub, clubExists := l.clubs[playerStats.EmployerClubID]
			if !clubExists {
				l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Deployment Failed: Club data corrupted."}`)})
				return
			}

			// Guardrail: Max 3 Active Traps per Club
			activeTraps := 0
			for key := range targetClub.ActiveBuffs {
				if strings.HasPrefix(key, "TRAP_") {
					activeTraps++
				}
			}
			if activeTraps >= 3 {
				l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Deployment Failed: Maximum defense capacity (3/3) reached."}`)})
				return
			}

			// Initialize maps if nil
			if targetClub.ActiveBuffs == nil {
				targetClub.ActiveBuffs = make(map[string]string)
			}
			if targetClub.BuffExpirations == nil {
				targetClub.BuffExpirations = make(map[string]time.Time)
			}

			// Deploy Trap with 24-hour expiration
			trapID := fmt.Sprintf("TRAP_%d", time.Now().UnixNano())
			targetClub.ActiveBuffs[trapID] = data.ItemID
			targetClub.BuffExpirations[trapID] = time.Now().Add(24 * time.Hour)
			targetClub.LastActivity = time.Now() // Mark club as active

			l.clubs[playerStats.EmployerClubID] = targetClub
			notificationText = fmt.Sprintf("🛰️ %s deployed in %s's territory!", item.Name, targetClub.Name)

		default:
			notificationText = fmt.Sprintf("❓ Used %s. Effect unknown or not yet implemented.", item.Name)
		}

		l.logAdminAudit("ITEM_USED", wallet, fmt.Sprintf("Item: %s, TargetCard: %d, TargetGrid: %d", data.ItemID, data.TargetCardID, data.TargetGridIndex))
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"%s"}`, notificationText))})

		// Trigger global sync to update UI (inventory, card stats, match state)
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	case "purchase_item":
		var data struct {
			ItemID      string `json:"item_id"`
			TerritoryID string `json:"territory_id"`
			Price       uint64 `json:"price"`
		}
		if err := json.Unmarshal(env.Payload, &data); err != nil {
			return
		}

		l.mutex.Lock()
		wallet := l.wallets[env.FromID]
		stats := l.leaderboard[wallet]

		// 1. Find the Club managing this territory
		var targetClub *Club
		for _, club := range l.clubs {
			for _, t := range club.Territories {
				if t == data.TerritoryID {
					targetClub = club
					break
				}
			}
		}

		if targetClub == nil || targetClub.Inventory[data.ItemID] <= 0 {
			l.mutex.Unlock()
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Purchase Failed: Item out of stock."}`)})
			return
		}

		// 2. Fulfillment: Deduct from Club, Grant to Player
		targetClub.Inventory[data.ItemID]--

		targetClub.LastActivity = time.Now()
		if stats.Inventory == nil {
			stats.Inventory = make(map[string]int)
		}
		stats.Inventory[data.ItemID]++
		l.leaderboard[wallet] = stats
		l.mutex.Unlock()

		// 3. Process Revenue (Existing logic)
		l.distributeShopRevenue(data.TerritoryID, data.Price, data.ItemID)

		l.logAdminAudit("ITEM_PURCHASE", wallet, fmt.Sprintf("Item: %s, Territory: %s", data.ItemID, data.TerritoryID))
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"📦 <b>PURCHASE COMPLETE:</b> %s added to inventory."}`, data.ItemID))})

		// Sync back to client to update local UI inventory
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	case "restock_inventory":
		var data struct {
			ClubID   string `json:"club_id"`
			ItemID   string `json:"item_id"`
			Quantity int    `json:"quantity"`
		}
		if err := json.Unmarshal(env.Payload, &data); err != nil {
			return
		}

		if data.Quantity <= 0 {
			return
		}

		l.mutex.Lock()
		ownerWallet, ok := l.wallets[env.FromID]
		if !ok {
			l.mutex.Unlock()
			return
		}

		club, exists := l.clubs[data.ClubID]
		if !exists {
			l.mutex.Unlock()
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Restock Failed: Club not found."}`)})
			return
		}

		// Authorization: Only the Owner (CEO) or a designated Manager can spend Treasury funds
		isOwner := strings.EqualFold(club.OwnerWallet, ownerWallet)
		isManager := club.Staff[strings.ToLower(ownerWallet)] == "Manager"
		if !isOwner && !isManager {
			l.mutex.Unlock()
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Restock Failed: Unauthorized. Manager or CEO role required."}`)})
			return
		}

		// Item Validation via Shop Registry
		item, itemExists := GlobalShopRegistry[data.ItemID]
		if !itemExists {
			l.mutex.Unlock()
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Restock Failed: Item not found in registry."}`)})
			return
		}

		// Financial Check: Spend directly from the Club Treasury
		totalCost := item.Price * float64(data.Quantity)
		if club.Treasury < totalCost {
			l.mutex.Unlock()
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"❌ Restock Failed: Insufficient Treasury funds. Need %.2f $VBV."}`, totalCost))})
			return
		}

		// Execute Transaction: Deduct from Treasury, add to Shop Stock
		club.Treasury -= totalCost
		if club.Inventory == nil {
			club.Inventory = make(map[string]int)
		}
		club.LastActivity = time.Now()
		club.Inventory[data.ItemID] += data.Quantity
		l.mutex.Unlock()

		l.logAdminAudit("CLUB_RESTOCK", ownerWallet, fmt.Sprintf("Club: %s, Item: %s, Qty: %d, Cost: %.2f", club.Name, data.ItemID, data.Quantity, totalCost))
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"📦 <b>RESTOCK COMPLETE:</b> Added %d units of %s to inventory."}`, data.Quantity, item.Name))})

		// Global sync to refresh treasury and stock levels in UI
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	case "trade_shares":
		var data struct {
			EntityID string  `json:"entity_id"`
			Action   string  `json:"action"` // "buy" or "sell"
			Amount   float64 `json:"amount"` // Number of shares
		}
		if err := json.Unmarshal(env.Payload, &data); err != nil {
			return
		}

		l.mutex.Lock()
		wallet, ok := l.wallets[env.FromID]
		if !ok {
			l.mutex.Unlock()
			return
		}

		// Pricing Logic: Map EntityID (ClientID) to their current stats
		targetWallet, targetExists := l.wallets[data.EntityID]
		if !targetExists {
			l.mutex.Unlock()
			return
		}
		targetStats := l.leaderboard[targetWallet]

		// Standard Formula: (Wins * 10) + (Reputation / 2) + 100
		basePrice := float64((targetStats.Wins * 10) + (targetStats.Reputation / 2) + 100)

		// Apply active rumor effects
		for _, rumor := range l.rumors {
			if rumor.TargetWallet == targetWallet && time.Now().Before(rumor.ExpiresAt) {
				basePrice *= rumor.Strength
			}
		}
		pricePerShare := basePrice
		totalValueMicro := uint64(data.Amount * pricePerShare * 1000000)

		stats := l.leaderboard[wallet]
		if stats.Portfolio == nil {
			stats.Portfolio = make(map[string]float64)
		}

		if data.Action == "buy" {
			if l.rewards[wallet] >= totalValueMicro {
				l.rewards[wallet] -= totalValueMicro
				stats.Portfolio[data.EntityID] += data.Amount
				l.logAdminAudit("STOCK_BUY", wallet, fmt.Sprintf("Bought %.2f shares of %s at %.2f $VBV", data.Amount, data.EntityID, pricePerShare))
			} else {
				l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Insufficient reward balance for trade."}`)})
			}
		} else if data.Action == "sell" {
			if stats.Portfolio[data.EntityID] >= data.Amount {
				stats.Portfolio[data.EntityID] -= data.Amount
				l.rewards[wallet] += totalValueMicro
				l.logAdminAudit("STOCK_SELL", wallet, fmt.Sprintf("Sold %.2f shares of %s at %.2f $VBV", data.Amount, data.EntityID, pricePerShare))
			} else {
				l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Insufficient shares in portfolio."}`)})
			}
		}

		l.leaderboard[wallet] = stats
		l.mutex.Unlock()

		// Push updated portfolio back to client
		portfolioPayload, _ := json.Marshal(stats.Portfolio)
		l.sendToClient(env.FromID, Envelope{Type: "portfolio_update", Payload: portfolioPayload})

		// Trigger global sync to update UI balances and portfolios
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	case "heist":
		var data struct {
			TargetClubID string `json:"target_club_id"`
		}
		if err := json.Unmarshal(env.Payload, &data); err != nil {
			return
		}
		l.mutex.Lock()
		defer l.mutex.Unlock()

		wallet, ok := l.wallets[env.FromID]
		if !ok {
			return
		}

		playerStats := l.leaderboard[wallet]
		targetClub, exists := l.clubs[data.TargetClubID]
		if !exists {
			return
		}

		// SUCCESS CHANCE CALCULATION: Proxy legacy real-estate security using Club Mojo
		// Base 50% chance + (Player Cunning - Territory Security) / 100
		successChance := 0.50
		securityLevel := float64(targetClub.Mojo) / 10.0

		// Employment Layer: Security staff significantly increase heist difficulty
		for _, role := range targetClub.Staff {
			if role == "Security" {
				securityLevel += 15.0 // Flat boost per security guard
			}
		}

		// NEW: Apply active trap modifiers
		for buffID, itemID := range targetClub.ActiveBuffs {
			if strings.HasPrefix(buffID, "TRAP_") { // Only consider active traps
				if item, exists := GlobalShopRegistry[itemID]; exists {
					successChance += item.HeistSuccessModifier // Modifiers are negative for traps
					log.Printf("[HEIST] Trap '%s' (%s) applied. Success chance modified by %.2f. Current: %.2f\n", item.Name, itemID, item.HeistSuccessModifier, successChance)
				}
			}
		}

		// Clamp Success Chance between 5% and 95%
		if successChance < 0.05 {
			successChance = 0.05
		}
		if successChance > 0.95 {
			successChance = 0.95
		}

		roll := rand.Float64()
		var status string
		canKidnap := false

		if roll < successChance {
			status = "success"

			// Pillar 3-A: Kidnap Gambit Eligibility
			if playerStats.Cunning >= 3 && rand.Float64() < 0.25 {
				canKidnap = true
			}

			loot := targetClub.Treasury * 0.10 // Steal 10% of Club Treasury
			if loot > 500 {
				loot = 500
			} // Cap loot for economy stability
			playerStats.Playstyle.RiskTolerance += 0.05 // Increase risk tolerance on successful heist
			playerStats.HeistAttempts++
			targetClub.Treasury -= loot
			targetClub.LastActivity = time.Now()
			playerStats.WantedLevel += 5
			playerStats.Cunning += 1

			go l.unlockAchievement(wallet, "FIRST_HEIST")
		} else {
			status = "failure"
			playerStats.WantedLevel += 15               // Higher infamy for getting caught
			playerStats.WantedLevel += 15               // Higher infamy for getting caught
			playerStats.Playstyle.RiskTolerance += 0.10 // Increase risk tolerance on failed heist
			playerStats.HeistAttempts++

			// GUARD DOG EFFECT: Check if the target club has a Bio-Guard Dog active
			hasGuardDog := false
			for _, trapItemID := range targetClub.ActiveBuffs {
				if trapItemID == "guard_dog" {
					hasGuardDog = true
					break
				}
			}

			if hasGuardDog {
				rarestCard, found := l.findRarestCardInInventory(wallet)
				if found {
					// Transfer card to Club Jail
					if targetClub.Jail == nil {
						targetClub.Jail = make(map[int]ServerCard)
					}
					targetClub.Jail[rarestCard.ID] = rarestCard

					// Remove from heister's inventory and record jailing
					delete(playerStats.Inventory, fmt.Sprintf("CARD-%d", rarestCard.ID))
					if playerStats.JailedCards == nil {
						playerStats.JailedCards = make(map[int]string)
					}
					playerStats.JailedCards[rarestCard.ID] = targetClub.ID

					log.Printf("[HEIST] Guard Dog caught %s! Rarest card %s jailed in Club %s\n", wallet, rarestCard.Name, targetClub.Name)
					l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🚨 <b>GUARD DOG BUST:</b> You were caught by a Bio-Guard Dog! Your rarest card (%s) has been jailed by %s."}`, rarestCard.Name, targetClub.Name))})
				}
			}
		}

		l.leaderboard[wallet] = playerStats
		l.logAdminAudit("HEIST_ATTEMPT", wallet, fmt.Sprintf("Target: %s, Result: %s", data.TargetClubID, status))

		response, _ := json.Marshal(map[string]interface{}{
			"status":          status,
			"wanted_level":    playerStats.WantedLevel,
			"cunning":         playerStats.Cunning,
			"playstyle":       playerStats.Playstyle,
			"heist_attempts":  playerStats.HeistAttempts,
			"kidnap_eligible": canKidnap,
			"target_club_id":  data.TargetClubID,
			"playstyle":       playerStats.Playstyle, // Include updated playstyle
		})
		l.sendToClient(env.FromID, Envelope{Type: "heist_result", Payload: response})
	case "create_club":
		var data struct {
			Name        string `json:"name"`
			Type        string `json:"type"`
			TerritoryID string `json:"territory_id"`
			TxID        string `json:"txid"`
			Network     string `json:"network"`
		}
		if err := json.Unmarshal(env.Payload, &data); err != nil {
			return
		}

		l.mutex.RLock()
		wallet, ok := l.wallets[env.FromID]
		vaultAddr := l.vaultAddress
		voiConfig := l.availableNetworks["Voi Mainnet"]
		l.mutex.RUnlock()

		if !ok {
			return
		}

		// Verify the 5000 $VBV "Fortune" Burn
		assetID := voiConfig.AssetID
		if data.Network == "ALGO" {
			assetID = l.avoiAssetID
		}

		verified, _, err := l.verifyBuyInTransaction(data.Network, data.TxID, 5000*1000000, assetID, wallet, vaultAddr)
		if err != nil || !verified {
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Club Foundry Error: Payment verification failed."}`)})
			return
		}

		l.mutex.Lock()
		clubID := fmt.Sprintf("CLUB-%d", time.Now().Unix())
		newClub := &Club{
			ID:           clubID,
			Name:         data.Name,
			OwnerWallet:  wallet,
			Type:         data.Type,
			Territories:  []string{data.TerritoryID},
			Commission:   0.05,
			Staff:        make(map[string]string),
			Members:      map[string]time.Time{strings.ToLower(wallet): time.Now()},
			CreatedAt:    time.Now(),
			Jail:         make(map[int]ServerCard), // Initialize Jail
			LastActivity: time.Now(),
		}
		newClub.Staff[strings.ToLower(wallet)] = "CEO" // Owner defaults to CEO
		l.clubs[clubID] = newClub
		l.mutex.Unlock()

		l.logAdminAudit("CLUB_CREATED", wallet, fmt.Sprintf("Name: %s, ID: %s", data.Name, clubID))
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🏛️ Club '%s' successfully founded!"}`, data.Name))})
	case "join_club":
		var data struct {
			ClubID  string `json:"club_id"`
			TxID    string `json:"txid"`
			Network string `json:"network"`
		}
		if err := json.Unmarshal(env.Payload, &data); err != nil {
			return
		}

		l.mutex.RLock()
		wallet, ok := l.wallets[env.FromID]
		vaultAddr := l.vaultAddress
		voiConfig := l.availableNetworks["Voi Mainnet"]
		l.mutex.RUnlock()

		if !ok {
			return
		}

		// Verify the 500 $VBV Join Fee (500,000,000 micro-units)
		assetID := voiConfig.AssetID
		verifyNet := "Voi"
		if data.Network == "ALGO" {
			assetID = l.avoiAssetID
			verifyNet = "Algorand"
		}

		verified, _, err := l.verifyBuyInTransaction(verifyNet, data.TxID, 500*1000000, assetID, wallet, vaultAddr)
		if err != nil || !verified {
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Club Entry Error: Payment verification failed."}`)})
			return
		}

		l.mutex.Lock()
		if club, exists := l.clubs[data.ClubID]; exists {
			if club.Members == nil {
				club.Members = make(map[string]time.Time)
			}
			club.Members[strings.ToLower(wallet)] = time.Now()
			club.Treasury += 250.0 // 50% of fee to Club Treasury
			club.LastActivity = time.Now()
			l.mutex.Unlock()
			l.logAdminAudit("CLUB_JOIN", wallet, fmt.Sprintf("Club: %s", data.ClubID))
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🤝 Welcome to %s!"}`, club.Name))})
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		} else {
			l.mutex.Unlock()
		}
	case "hire_player":
		var data struct {
			ClubID   string `json:"club_id"`
			TargetID string `json:"target_id"`
			Role     string `json:"role"` // Manager, Security, Clerk
		}
		if err := json.Unmarshal(env.Payload, &data); err != nil {
			return
		}

		l.mutex.Lock()
		defer l.mutex.Unlock()

		ownerWallet, ok := l.wallets[env.FromID]
		if !ok {
			return
		}

		club, exists := l.clubs[data.ClubID]
		if !exists || strings.ToLower(club.OwnerWallet) != strings.ToLower(ownerWallet) {
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Hiring Failed: Unauthorized or Club not found."}`)})
			return
		}

		targetWallet, targetConnected := l.wallets[data.TargetID]
		if !targetConnected {
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Hiring Failed: Player not found in lobby."}`)})
			return
		}

		// Update target player stats (Employment Record)
		stats := l.leaderboard[targetWallet]
		stats.JobRole = data.Role
		stats.EmployerClubID = data.ClubID
		l.leaderboard[targetWallet] = stats

		// Update club staffing map
		if club.Staff == nil {
			club.Staff = make(map[string]string)
		}
		club.Staff[strings.ToLower(targetWallet)] = data.Role
		l.clubs[data.ClubID] = club

		l.logAdminAudit("PLAYER_HIRED", targetWallet, fmt.Sprintf("Club: %s (%s), Role: %s", club.Name, club.ID, data.Role))

		// Notify the employee of their new position
		notification, _ := json.Marshal(map[string]string{
			"text": fmt.Sprintf("💼 <b>JOB ASSIGNMENT:</b> You are now the %s for %s!", data.Role, club.Name),
		})
		l.sendToClient(data.TargetID, Envelope{Type: "admin_notification", Payload: notification})

		// Trigger global sync to update UI roles and badges
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	case "purchase_territory":
		var data struct {
			ClubID      string `json:"club_id"`
			TerritoryID string `json:"territory_id"`
			TxID        string `json:"txid"`
			Network     string `json:"network"`
		}
		if err := json.Unmarshal(env.Payload, &data); err != nil {
			log.Printf("[TERRITORY] Invalid purchase_territory payload from %s: %v\n", env.FromID, err)
			return
		}

		l.mutex.Lock()
		defer l.mutex.Unlock()

		ownerWallet, ok := l.wallets[env.FromID]
		if !ok {
			log.Printf("[TERRITORY] Purchase failed: Sender %s not connected.\n", env.FromID)
			return
		}

		club, exists := l.clubs[data.ClubID]
		if !exists || strings.ToLower(club.OwnerWallet) != strings.ToLower(ownerWallet) {
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Territory Purchase Failed: Unauthorized or Club not found."}`)})
			log.Printf("[TERRITORY] Purchase failed for %s: Unauthorized or Club %s not found.\n", ownerWallet, data.ClubID)
			return
		}

		// Check if territory is already owned by any club
		for _, existingClub := range l.clubs {
			for _, t := range existingClub.Territories {
				if t == data.TerritoryID {
					l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Territory Purchase Failed: District already claimed."}`)})
					log.Printf("[TERRITORY] Purchase failed for %s: Territory %s already claimed.\n", ownerWallet, data.TerritoryID)
					return
				}
			}
		}

		// Verify the 2500 $VBV payment
		purchaseCost := 2500.0
		voiConfig := l.availableNetworks["Voi Mainnet"]
		assetID := voiConfig.AssetID
		verifyNet := "Voi"
		if data.Network == "ALGO" {
			assetID = l.avoiAssetID
			verifyNet = "Algorand"
		}

		verified, _, err := l.verifyBuyInTransaction(verifyNet, data.TxID, uint64(purchaseCost*1000000), assetID, ownerWallet, l.vaultAddress)
		if err != nil || !verified {
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Territory Purchase Failed: Payment verification failed."}`)})
			log.Printf("[TERRITORY] Payment verification failed for %s. Error: %v\n", ownerWallet, err)
			return
		}

		// Add territory to club
		club.Territories = append(club.Territories, data.TerritoryID)
		l.clubs[data.ClubID] = club
		club.LastActivity = time.Now()

		l.logAdminAudit("TERRITORY_PURCHASED", ownerWallet, fmt.Sprintf("Club: %s (%s), Territory: %s", club.Name, club.ID, data.TerritoryID))
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🗺️ Club '%s' has acquired %s!"}`, club.Name, data.TerritoryID))})

		// Trigger rank refresh and global sync
		go l.refreshRegionalRoles() // Check for new Governor status
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	case "kidnap_request":
		l.handleKidnapRequest(env)
	case "pay_ransom":
		l.handlePayRansom(env)
	case "spread_rumor":
		l.handleSpreadRumor(env)
	}
}

// getClientIDFromWallet is a helper to find an active connection ID by wallet address.
func (l *Lobby) getClientIDFromWallet(wallet string) string {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	for id, w := range l.wallets {
		if strings.EqualFold(w, wallet) {
			return id
		}
	}
	return ""
}

// checkRegionalStatus evaluates if a club has expanded into a Region.
func (l *Lobby) checkRegionalStatus(clubID string) bool {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	club, exists := l.clubs[clubID]
	if !exists {
		return false
	}

	// Rule: 2 or more territories = Region
	if len(club.Territories) >= 2 {
		return true
	}
	return false
}

func (l *Lobby) cleanupNonces() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	now := time.Now()
	for id, nd := range l.nonces {
		if now.Sub(nd.CreatedAt) > 5*time.Minute {
			delete(l.nonces, id)
		}
	}
	for id, history := range l.matchHistory {
		if now.Sub(history.Timestamp) > 30*time.Minute {
			delete(l.matchHistory, id)
		}
	}
	for ip, bucket := range l.httpRateLimits {
		if bucket.Tokens >= 10.0 && now.Sub(bucket.LastUpdate) > 1*time.Hour {
			delete(l.httpRateLimits, ip)
		}
	}
	for txid, ts := range l.registeredTxIDs {
		if now.Sub(ts) > 30*24*time.Hour {
			delete(l.registeredTxIDs, txid)
		}
	}
}

func (l *Lobby) handleUnregister(client *Client) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if match, ok := l.matches[client.id]; ok {
		opponentID := match.P1ID
		if client.id == match.P1ID {
			opponentID = match.P2ID
		}
		if opponentID != "" {
			if opponent, exists := l.clients[opponentID]; exists {
				notification, _ := json.Marshal(Envelope{
					Type: "chat", FromID: "SERVER",
					Payload: json.RawMessage(`{"text":"Match invalidated: Opponent disconnected."}`),
				})
				select {
				case opponent.send <- notification:
				default:
				}
			}
			if wallet, ok := l.wallets[client.id]; ok {
				l.incrementDNF(wallet)
			}
			delete(l.matches, opponentID)
		}
		delete(l.matches, client.id)
	}
	delete(l.wallets, client.id)
	if _, ok := l.clients[client.id]; ok {
		delete(l.clients, client.id)
		close(client.send)
	}
	msg := l.getLobbyUpdateMsgLocked()
	go func() { l.broadcast <- msg }()
}

func (l *Lobby) handleBroadcast(message []byte) {
	var env Envelope
	if err := json.Unmarshal(message, &env); err != nil {
		return
	}
	l.handleGameProtocol(&env, message) // Process logic before routing

	l.mutex.RLock()
	defer l.mutex.RUnlock()
	if env.ToID != "" && env.ToID != "ALL" {
		if target, ok := l.clients[env.ToID]; ok {
			select {
			case target.send <- message:
			default:
			}
		}
	} else {
		for _, client := range l.clients {
			select {
			case client.send <- message:
			default:
			}
		}
	}
}

func (l *Lobby) getLobbyUpdateMsgLocked() []byte {
	type playerInfo struct {
		ID               string         `json:"id"`
		IsAdmin          bool           `json:"is_admin"`
		AvatarURL        string         `json:"avatar_url"`
		Gloat            string         `json:"gloat"`
		AvatarNotice     string         `json:"avatar_notice"`
		BanExpires       time.Time      `json:"ban_expires"`
		HasMardonBadge   bool           `json:"has_mardon_badge"`
		Wins             int            `json:"wins"`
		Reputation       int            `json:"reputation"`
		JailedCards      map[int]string `json:"jailed_cards"`       // Added for UI display
		SocialRank       string         `json:"social_rank"`        // Added for UI display
		KidnappedCards   map[int]string `json:"kidnapped_cards"`    // Added for UI display
		HeldHostageCards map[int]string `json:"held_hostage_cards"` // Added for UI display
		Achievements     []string       `json:"achievements"`       // Added for UI display
		// JobRole and EmployerID are already present in the playerInfo struct
		RumorCount int    `json:"rumor_count"` // Added for UI display
		JobRole    string `json:"job_role"`    // Manager, Security, Clerk
		EmployerID string `json:"employer_id"`
	}
	var players []playerInfo
	for _, client := range l.clients {
		hasMardon := false
		var banExpires time.Time
		wins, reputation := 0, 0
		var jailedCards map[int]string
		// kidnappedCards and heldHostageCards not currently used; TODO: implement criminal tracking in lobby display
		var socialRank string
		var achievements []string
		var jobRole string
		var employerID string
		if wallet, ok := l.wallets[client.id]; ok {
			if stats, exists := l.leaderboard[wallet]; exists {
				banExpires = stats.BanExpires
				wins = stats.Wins
				reputation = stats.Reputation
				jailedCards = stats.JailedCards
				socialRank = stats.SocialRank
				achievements = stats.Achievements
				jobRole = stats.JobRole
				employerID = stats.EmployerClubID
				achievements = stats.Achievements
				if stats.Wins >= 50 && stats.DisconnectStreak == 0 {
					hasMardon = true
				}
			}
		}
		players = append(players, playerInfo{
			ID: client.id, IsAdmin: client.isAdmin, AvatarURL: client.avatarURL,
			Gloat: client.gloat, AvatarNotice: client.avatarBanNotice,
			BanExpires: banExpires, HasMardonBadge: hasMardon, Wins: wins, Reputation: reputation,
			JailedCards: jailedCards, SocialRank: socialRank, Achievements: achievements,
			JobRole: jobRole, EmployerID: employerID,
		})
	}

	update := struct {
		Players           []playerInfo             `json:"players"`
		MaintenanceActive bool                     `json:"maintenance_active"`
		FaucetBalance     float64                  `json:"faucet_balance"`
		Clubs             map[string]*Club         `json:"clubs"`
		Rewards           map[string]uint64        `json:"rewards"`
		ActiveMatchCount  int                      `json:"active_match_count"`
		Tournament        TournamentState          `json:"tournament"`
		AvailableNetworks map[string]NetworkConfig `json:"available_networks"`
		Rumors            map[string]*Rumor        `json:"rumors"` // Added for UI display
		AdminFocusNetwork string                   `json:"admin_focus_network"`
	}{
		Players: players, MaintenanceActive: l.maintenanceMode,
		Clubs:   l.clubs,
		Rewards: l.rewards, FaucetBalance: l.faucetBalance,
		ActiveMatchCount: len(l.matches) / 2, Tournament: l.tournament,
		AvailableNetworks: l.availableNetworks, AdminFocusNetwork: l.adminFocusNetwork,
		Rumors: l.rumors,
	}

	payload, _ := json.Marshal(update)
	env := Envelope{Type: "lobby_update", FromID: "SERVER", Payload: payload}
	msg, _ := json.Marshal(env)
	return msg
}

func (l *Lobby) processMatchmaking() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if len(l.matchmakingPool) < 2 {
		return
	}

	var matchedIndices = make(map[int]bool)

	// 1. TOURNAMENT PRIORITY PASS: Pair players scheduled in the current bracket round
	if l.tournament.Active && l.tournament.CurrentRound > 0 {
		for _, match := range l.tournament.Matches {
			if match.Round == l.tournament.CurrentRound && match.Winner == "" && match.P2 != "BYE" {
				idx1, idx2 := -1, -1
				for k, entry := range l.matchmakingPool {
					if matchedIndices[k] {
						continue
					}
					if entry.Wallet == match.P1 {
						idx1 = k
					}
					if entry.Wallet == match.P2 {
						idx2 = k
					}
				}
				if idx1 != -1 && idx2 != -1 {
					if l.initiatePairedMatch(l.matchmakingPool[idx1].ClientID, l.matchmakingPool[idx2].ClientID) {
						// Link to bracket for automatic result reporting
						if mState, ok := l.matches[l.matchmakingPool[idx1].ClientID]; ok {
							mState.TournamentMatchID = match.ID
						}
						matchedIndices[idx1], matchedIndices[idx2] = true, true
						log.Printf("[MATCHMAKING] Tournament Pairing: %s vs %s\n", match.P1, match.P2)
					}
				}
			}
		}
	}

	// 2. BOUNTY HUNTER PASS: Pair low-Wanted players (Hunters) against high-Wanted players (Outlaws)
	for i := 0; i < len(l.matchmakingPool); i++ {
		if matchedIndices[i] {
			continue
		}
		p1 := l.matchmakingPool[i]
		p1Wanted := l.leaderboard[p1.Wallet].WantedLevel

		for j := i + 1; j < len(l.matchmakingPool); j++ {
			if matchedIndices[j] {
				continue
			}
			p2 := l.matchmakingPool[j]
			p2Wanted := l.leaderboard[p2.Wallet].WantedLevel

			isBounty := false
			// Hunter (Wanted <= 2) vs Outlaw (Wanted >= 10)
			if (p1Wanted <= 2 && p2Wanted >= 10) || (p2Wanted <= 2 && p1Wanted >= 10) {
				isBounty = true
			}

			if isBounty {
				// Looser constraints for bounty matches: Reputation diff up to 400
				repDiff := p1.Reputation - p2.Reputation
				if repDiff < 0 {
					repDiff = -repDiff
				}

				if repDiff <= 400 {
					if l.initiatePairedMatch(p1.ClientID, p2.ClientID) {
						matchedIndices[i], matchedIndices[j] = true, true
						// Flag the match as a bounty duel
						if mState, ok := l.matches[p1.ClientID]; ok {
							mState.IsBountyMatch = true
						}
						log.Printf("[MATCHMAKING] Bounty Match Initiated: Hunter/Outlaw pair %s vs %s\n", p1.Wallet, p2.Wallet)
						break
					}
				}
			}
		}
	}

	getGradeIdx := func(rating string) int {
		if len(rating) < 3 {
			return 25
		}
		return int(rating[1] - 'A')
	}

	// 3. STANDARD POOL: Match by Reputation and Deck Tier
	for i := 0; i < len(l.matchmakingPool); i++ {
		if matchedIndices[i] {
			continue
		}
		p1 := l.matchmakingPool[i]
		for j := i + 1; j < len(l.matchmakingPool); j++ {
			if matchedIndices[j] {
				continue
			}
			p2 := l.matchmakingPool[j]

			repDiff := p1.Reputation - p2.Reputation
			if repDiff < 0 {
				repDiff = -repDiff
			}
			gradeDiff := getGradeIdx(p1.DeckRating) - getGradeIdx(p2.DeckRating)
			if gradeDiff < 0 {
				gradeDiff = -gradeDiff
			}

			if repDiff <= 200 && gradeDiff <= 2 {
				if l.initiatePairedMatch(p1.ClientID, p2.ClientID) {
					matchedIndices[i], matchedIndices[j] = true, true
					break
				}
			}
		}
	}
	var remaining []QueueEntry
	for i, entry := range l.matchmakingPool {
		if !matchedIndices[i] {
			remaining = append(remaining, entry)
		}
	}
	l.matchmakingPool = remaining
}

func (l *Lobby) initiatePairedMatch(id1, id2 string) bool {
	m1, ok1 := l.matches[id1]
	m2, ok2 := l.matches[id2]
	if !ok1 || !ok2 {
		return false
	}

	match := &MatchState{
		P1ID: id1, P2ID: id2, P1Deck: m1.P1Deck, P2Deck: m2.P1Deck,
		Rules:         map[string]bool{"Open": true},             // Default rules
		P1WantedLevel: l.leaderboard[l.wallets[id1]].WantedLevel, // Wanted levels from leaderboard
		P2WantedLevel: l.leaderboard[l.wallets[id2]].WantedLevel,
		TerritoryID:   l.assignMatchTerritory(), // Assign a territory to the match
	}
	l.matches[id1], l.matches[id2] = match, match

	p1Sync, _ := json.Marshal(Envelope{
		Type: "challenge", FromID: id2, ToID: id1,
		Payload: json.RawMessage(fmt.Sprintf(`{"action":"accept","deck":%v,"wanted_level":%d}`, jsonList(match.P2Deck), match.P2WantedLevel)),
	})
	p2Sync, _ := json.Marshal(Envelope{
		Type: "challenge", FromID: id1, ToID: id2,
		Payload: json.RawMessage(fmt.Sprintf(`{"action":"sync_back","deck":%v,"wanted_level":%d}`, jsonList(match.P1Deck), match.P1WantedLevel)),
	})

	if c1, ok := l.clients[id1]; ok {
		c1.send <- p1Sync
	}
	if c2, ok := l.clients[id2]; ok {
		c2.send <- p2Sync
	}

	l.sendToClient(id1, Envelope{Type: "matchmaking_status", Payload: json.RawMessage(`{"status":"match_found"}`)})
	l.sendToClient(id2, Envelope{Type: "matchmaking_status", Payload: json.RawMessage(`{"status":"match_found"}`)})
	return true
}

// assignMatchTerritory assigns a territory ID to a new match.
// For now, it picks a random territory. In the future, this could be based on
// player location, club challenges, or tournament settings.
func (l *Lobby) assignMatchTerritory() string {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	territories := []string{"the_lab", "casino", "arena_center", "north_district", "south_slums", "east_gate", "west_port", "the_archive", "data_haven"}
	if len(territories) == 0 {
		return ""
	}
	return territories[rand.Intn(len(territories))]
}

// isWalletConnected checks if the given wallet address is currently associated with an active connection.
func (l *Lobby) isWalletConnected(wallet string) bool {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	for _, w := range l.wallets {
		if w == wallet {
			return true
		}
	}
	return false
}

// jsonList is a helper to marshal a slice of ints to a JSON string.
func jsonList(ints []int) string {
	b, _ := json.Marshal(ints)
	return string(b)
}

// jsonListEnvelope creates a JSON-encoded Envelope for broadcasting.
func jsonListEnvelope(envType string, payload []byte) []byte {
	msg, _ := json.Marshal(Envelope{Type: envType, FromID: "SERVER", Payload: payload})
	return msg
}

// sendToClient sends an Envelope message to a specific client.
func (l *Lobby) sendToClient(clientID string, env Envelope) {
	l.mutex.RLock()
	client, ok := l.clients[clientID]
	l.mutex.RUnlock()
	if !ok {
		return
	}

	msg, err := json.Marshal(env)
	if err != nil {
		return
	}

	select {
	case client.send <- msg:
	default: // Drop if full
	}
}

// generateNonce creates a cryptographically secure random string.
func generateNonce() string {
	b := make([]byte, 16)
	crand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

// getLobbyUpdateMsg is a thread-safe wrapper for generating a lobby state snapshot.
func (l *Lobby) getLobbyUpdateMsg() []byte {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.getLobbyUpdateMsgLocked()
}

// loadRegisteredTxIDs loads tournament registration transaction IDs from a file.
func (l *Lobby) loadRegisteredTxIDs() {
	data, err := os.ReadFile(regCacheFileName)
	if err != nil {
		return
	}
	l.mutex.Lock()
	json.Unmarshal(data, &l.registeredTxIDs)
	l.mutex.Unlock()
	log.Printf("[CACHE] Loaded %d tournament registration records.\n", len(l.registeredTxIDs))
}

// saveRegisteredTxIDs saves tournament registration transaction IDs to a file.
func (l *Lobby) saveRegisteredTxIDs() {
	l.mutex.RLock()
	data, err := json.Marshal(l.registeredTxIDs)
	l.mutex.RUnlock()
	if err != nil {
		return
	}
	if err := os.WriteFile(regCacheFileName, data, 0644); err != nil {
		log.Printf("[CACHE ERROR] Failed to save registrations: %v\n", err)
	}
}

// loadLinkedWallets loads linked wallet information from a file.
func (l *Lobby) loadLinkedWallets() {
	data, err := os.ReadFile(linkedWalletsFileName)
	if err != nil {
		log.Printf("[CACHE] No linked_wallets.json found, starting fresh: %v\n", err)
		return
	}
	l.mutex.Lock()
	json.Unmarshal(data, &l.linkedWallets)
	l.mutex.Unlock()
	log.Printf("[CACHE] Loaded %d linked wallet records.\n", len(l.linkedWallets))
}

// saveLinkedWallets saves linked wallet information to a file.
func (l *Lobby) saveLinkedWallets() {
	l.mutex.RLock()
	data, err := json.MarshalIndent(l.linkedWallets, "", "  ")
	l.mutex.RUnlock()
	if err != nil {
		log.Printf("[CACHE ERROR] Failed to marshal linked wallets: %v\n", err)
		return
	}
	if err := os.WriteFile(linkedWalletsFileName, data, 0644); err != nil {
		log.Printf("[CACHE ERROR] Failed to write linked wallets file: %v\n", err)
	}
}

func (l *Lobby) addOrUpdateLinkedWallet(primaryAVM, linkedAddr, linkedChain string) {
	l.mutex.Lock()
	defer l.mutex.Unlock() // Ensure mutex is unlocked

	linkInfo, ok := l.linkedWallets[primaryAVM]
	if !ok {
		linkInfo = WalletLinkInfo{PrimaryAVMWallet: primaryAVM}
	}

	// Check if already linked, update if so
	found := false
	for i, lw := range linkInfo.Linked {
		if strings.EqualFold(lw.Address, linkedAddr) && strings.EqualFold(lw.Chain, linkedChain) {
			linkInfo.Linked[i].Verified = true
			linkInfo.Linked[i].Timestamp = time.Now()
			found = true
			break
		}
	}

	if !found {
		linkInfo.Linked = append(linkInfo.Linked, LinkedWallet{Address: linkedAddr, Chain: linkedChain, Verified: true, Timestamp: time.Now()})
	}
	l.linkedWallets[primaryAVM] = linkInfo
	l.saveLinkedWallets() // Save immediately on change
}

// updatePlayerPlaystyleTendencies calculates and updates a player's observed playstyle, including rule and card preferences.
func (l *Lobby) updatePlayerPlaystyleTendencies(wallet string, inMatchContext bool, scores [2]int, deck []int, isBountyMatch bool) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	stats, exists := l.leaderboard[wallet]
	if !exists {
		return
	}

	// Pillar 3: EMA (Exponential Moving Average) Logic
	// Alpha = 0.2 (Recent matches represent 20% of the tendency)
	const alpha = 0.2

	// Initialize and Decay
	if stats.Playstyle.PreferredRules == nil {
		stats.Playstyle.PreferredRules = make(map[string]float64)
	}
	if stats.Playstyle.PreferredCardMoods == nil {
		stats.Playstyle.PreferredCardMoods = make(map[string]float64)
	}

	// 1. Aggressiveness (Direct captures vs Rule-based)
	// For now, we use a scoring heuristic based on victory margins
	matchAgg := 0.5
	if scores[0] > scores[1] {
		margin := scores[0] - scores[1]
		if margin >= 4 {
			matchAgg = 0.9
		} // Crushing victory
	}
	stats.Playstyle.Aggressiveness = (matchAgg * alpha) + (stats.Playstyle.Aggressiveness * (1 - alpha))

	// 2. Risk Tolerance (Based on Wanted Level and Heist success)
	matchRisk := float64(stats.WantedLevel) / 20.0
	if matchRisk > 1 {
		matchRisk = 1
	}
	stats.Playstyle.RiskTolerance = (matchRisk * alpha) + (stats.Playstyle.RiskTolerance * (1 - alpha))

	// Preferred Rules (if in match context)
	if inMatchContext {
		// Get the match state for the player
		match, matchExists := l.matches[l.getClientIDFromWallet(wallet)]
		if matchExists {
			for ruleName, isActive := range match.Rules {
				if isActive {
					// Increment score for active rules, decay others
					stats.Playstyle.PreferredRules[ruleName] = stats.Playstyle.PreferredRules[ruleName]*0.9 + 1.0
				} else {
					stats.Playstyle.PreferredRules[ruleName] *= 0.9 // Decay inactive rules
				}
			}
		}
	}

	// Preferred Card Moods (if deck context is provided)
	if len(deck) > 0 {
		for _, cardID := range deck {
			if card, exists := l.inventory[cardID]; exists {
				if card.Mood != "" && card.Mood != "Neutral" {
					stats.Playstyle.PreferredCardMoods[card.Mood] = stats.Playstyle.PreferredCardMoods[card.Mood]*0.9 + 1.0
				}
			}
		}
		// Decay moods not in the current deck
		for mood := range stats.Playstyle.PreferredCardMoods {
			found := false
			for _, cardID := range deck {
				if card, exists := l.inventory[cardID]; exists && card.Mood == mood {
					found = true
					break
				}
			}
			if !found {
				stats.Playstyle.PreferredCardMoods[mood] *= 0.9
			}
		}
	}

	l.leaderboard[wallet] = stats
}

// processRumors checks for expired rumors and removes them.
func (l *Lobby) processRumors() {
	l.mutex.Lock()

	defer l.mutex.Unlock()
	now := time.Now()
	for id, rumor := range l.rumors {
		if now.After(rumor.ExpiresAt) {
			log.Printf("[RUMOR] Rumor %s about %s expired.\n", id, rumor.TargetWallet)
			delete(l.rumors, id)
		}
	}
}

// processPlaystyleDecay periodically decays playstyle tendencies.
func (l *Lobby) processPlaystyleDecay() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	for wallet, stats := range l.leaderboard {
		// Decay rates
		const decayFactor = 0.95 // 5% decay per tick (e.g., per minute)

		for ruleName := range stats.Playstyle.PreferredRules {
			stats.Playstyle.PreferredRules[ruleName] *= decayFactor
		}
		for mood := range stats.Playstyle.PreferredCardMoods {
			stats.Playstyle.PreferredCardMoods[mood] *= decayFactor
		}
		for itemID := range stats.Playstyle.PreferredItems {
			stats.Playstyle.PreferredItems[itemID] *= decayFactor
		}
		l.leaderboard[wallet] = stats
	}
}

// getClubByTerritoryID returns the club that owns the given territory, or nil if none.
func (l *Lobby) getClubByTerritoryID(territoryID string) *Club {
	for _, club := range l.clubs {
		for _, t := range club.Territories {
			if t == territoryID {
				return club
			}
		}
	}
	return nil
}

// findRarestCardInInventory finds the card with the highest rarity in a player's inventory.
func (l *Lobby) findRarestCardInInventory(wallet string) (ServerCard, bool) {
	stats, exists := l.leaderboard[wallet]
	if !exists || stats.Inventory == nil || len(stats.Inventory) == 0 {
		return ServerCard{}, false
	}

	var rarestCard ServerCard
	maxRarity := -1.0
	found := false

	for itemID, quantity := range stats.Inventory {
		if quantity <= 0 {
			continue
		}
		// Assuming card IDs are prefixed with "CARD-"
		if strings.HasPrefix(itemID, "CARD-") {
			cardIDStr := strings.TrimPrefix(itemID, "CARD-")
			cardID, err := strconv.Atoi(cardIDStr)
			if err != nil {
				continue
			}
			if card, cardExists := l.inventory[cardID]; cardExists {
				if card.Rarity > maxRarity {
					maxRarity = card.Rarity
					rarestCard = card
					found = true
				}
			}
		}
	}
	return rarestCard, found
}

// processMojoDecay reduces a Club's Mojo if it has been stagnant for too long.
func (l *Lobby) processMojoDecay() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	now := time.Now()
	stagnationThreshold := 48 * time.Hour

	for _, club := range l.clubs {
		if club.Mojo <= 0 {
			continue
		}

		if now.Sub(club.LastActivity) > stagnationThreshold {
			decayAmount := 5 // Flat decay
			club.Mojo -= decayAmount
			if club.Mojo < 0 {
				club.Mojo = 0
			}

			log.Printf("[INDUSTRIAL] Club %s suffered Mojo decay due to stagnation. New Mojo: %d\n", club.Name, club.Mojo)
			// We don't update LastActivity here, so it continues to decay every tick until activity occurs.
		}
	}
}

// archiveSeason persists current HoF standings to the blockchain and resets the clock.
func (l *Lobby) archiveSeason() {
	l.mutex.Lock()

	defer l.mutex.Unlock()

	log.Printf("[SEASON] Archiving Season %d Standings...\n", l.seasonNumber)

	type hofEntry struct {
		W string `json:"w"` // Wallet
		V int    `json:"v"` // Victories
		R string `json:"r"` // Rating
	}
	var standings []hofEntry
	for w, s := range l.leaderboard {
		if s.Wins > 0 {
			standings = append(standings, hofEntry{W: w, V: s.Wins, R: s.BestRating})
		}
	}
	sort.Slice(standings, func(i, j int) bool { return standings[i].V > standings[j].V })

	// Take Top 10 for the archive note
	limit := 10
	if len(standings) < limit {
		limit = len(standings)
	}

	summary := struct {
		Season int        `json:"season"`
		Start  time.Time  `json:"start"`
		End    time.Time  `json:"end"`
		Top    []hofEntry `json:"top"`
	}{
		Season: l.seasonNumber,
		Start:  l.seasonStart,
		End:    time.Now(),
		Top:    standings[:limit],
	}

	jsonData, _ := json.Marshal(summary)
	note := fmt.Sprintf("VBT_SEASON_ARCHIVE:%s", string(jsonData))
	l.sendNoteTx(note) // Record on chain

	// Reset Cycle
	l.seasonNumber++
	l.seasonStart = time.Now()
	l.leaderboard = make(map[string]PlayerStats) // Clear HoF

	// Persist new config
	conf, _ := json.Marshal(map[string]interface{}{"start": l.seasonStart, "num": l.seasonNumber})
	os.WriteFile("season.json", conf, 0644)
}

// ensurePlayerStatsMapsInitialized ensures that all map fields in PlayerStats are initialized.
func (l *Lobby) ensurePlayerStatsMapsInitialized(wallet string) {
	stats := l.leaderboard[wallet]
	if stats.Inventory == nil {
		stats.Inventory = make(map[string]int)
	}
	if stats.JailedCards == nil {
		stats.JailedCards = make(map[int]string)
	}
	if stats.Playstyle.PreferredRules == nil {
		stats.Playstyle.PreferredRules = make(map[string]float64)
	}
	if stats.Playstyle.PreferredCardMoods == nil {
		stats.Playstyle.PreferredCardMoods = make(map[string]float64)
	}
	if stats.Playstyle.PreferredItems == nil {
		stats.Playstyle.PreferredItems = make(map[string]float64)
	}
	// RumorCount is an int, no map initialization needed
	l.leaderboard[wallet] = stats
}
