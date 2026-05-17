//go:build !js || !wasm

package main

import (
	"crypto/ed25519"
	crand "crypto/rand"
	"math"
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

const linkedWalletsName = "linked_wallets.json"

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

	// STRESS TEST HOOK: Automatically trigger a 16-player tournament simulation on startup
	if os.Getenv("ARENA_STRESS_TEST") == "true" {
		log.Println("[STRESS TEST] ARENA_STRESS_TEST detected. Triggering 16-player tournament simulation...")
		// Simulate a 16-player buy-in tournament.
		go l.simulateTournament(16, true)
		// Give the simulation a moment to start before the main loop takes over
		time.Sleep(2 * time.Second)
	}

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
			l.processLeaseExpirations()    // New: Check for expired card leases
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
		normalizedWallet := strings.ToLower(data.Wallet)
		l.mutex.Lock()
		l.wallets[env.FromID] = normalizedWallet
		l.ensurePlayerStatsMapsInitialized(normalizedWallet)

		// Trigger NPC Welcome Commentary if they have a distinct style
		go l.generateNPCCommentary(env.FromID, "LOBBY_ENTRY")

		stats := l.leaderboard[normalizedWallet]
		portfolioPayload, _ := json.Marshal(stats.Portfolio) // Marshal portfolio while lock is held

		// Check admin status and update client while lock is held
		isAdmin := l.isAdminWallet(normalizedWallet)
		if isAdmin {
			if c, ok := l.clients[env.FromID]; ok {
				c.isAdmin = true
			}
		}
		l.mutex.Unlock() // Release lock after all state modifications
		go l.syncStatsFromBlockchain(env.FromID, normalizedWallet)
		l.sendToClient(env.FromID, Envelope{Type: "portfolio_update", Payload: portfolioPayload})
		if isAdmin {
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		}
	case "register_avatar":
		var data struct {
			URL            string `json:"url"`
			Gloat          string `json:"gloat"`
			FavoriteCardID int    `json:"favorite_card_id"`
		}
		json.Unmarshal(env.Payload, &data)

		targetURL := strings.TrimSpace(data.URL)
		l.mutex.Lock()
		wallet, ok := l.wallets[env.FromID] // Check if wallet is registered
		if !ok {
			l.mutex.Unlock()
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Avatar Registration Failed: Wallet not registered."}`)})
			return
		}

		// Enforce Avatar Ban: check against active bans
		if expiry, banned := l.bannedAvatars[targetURL]; banned && time.Now().Before(expiry) {
			l.mutex.Unlock()
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", FromID: "SERVER", Payload: json.RawMessage(`{"text":"❌ <b>AVATAR BLOCKED:</b> This image is restricted by Arena security."}`)})
			return
		}

		if stats, exists := l.leaderboard[wallet]; exists {
			stats.FavoriteCardID = data.FavoriteCardID
			l.leaderboard[wallet] = stats
		}

		if c, ok := l.clients[env.FromID]; ok {
			c.avatarURL = targetURL
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
			l.matches[env.FromID] = &MatchState{P1ID: env.FromID, P1Deck: data.Deck}                  // Initialize match state
			l.updatePlayerPlaystyleTendenciesLocked(wallet, false, [2]int{}, data.Deck, false, false) // Update playstyle based on deck

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

		// PILLAR 3: Identity Normalization.
		// Voi/AVM wallets are normalized to lowercase.
		// EVM addresses are normalized, but Solana (Base58) remains case-sensitive.
		primaryWallet := strings.ToLower(data.PrimaryAVMWallet)
		linkedAddr := data.LinkedAddress
		isSolana := strings.EqualFold(data.LinkedChain, "sol")
		if !isSolana {
			linkedAddr = strings.ToLower(data.LinkedAddress)
		}

		l.mutex.RLock()
		nonceData, exists := l.nonces[env.FromID] // Nonce is generated for the client's session
		l.mutex.RUnlock()

		if !exists || nonceData.Value != data.Nonce || time.Since(nonceData.CreatedAt) > 5*time.Minute {
			log.Printf("[LINK] Nonce verification failed for %s (linked: %s). Exists: %v, Match: %v, Expired: %v\n",
				env.FromID, linkedAddr, exists, nonceData.Value == data.Nonce, time.Since(nonceData.CreatedAt) > 5*time.Minute)
			l.sendToClient(env.FromID, Envelope{Type: "link_wallet_response", Payload: json.RawMessage(fmt.Sprintf(`{"status":"error","message":"Nonce invalid or expired","address":"%s"}`, linkedAddr))})
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
			if strings.EqualFold(recoveredAddress, linkedAddr) {
				verified = true
			} else {
				verifyErr = fmt.Errorf("EVM signature mismatch. Recovered: %s, Expected: %s", recoveredAddress, linkedAddr)
			}
		case "sol":
			// Solana signature verification (ed25519)
			// Message format: `\x19Solana Signed Message:\n` + length + message. Base58 check.
			message := fmt.Sprintf("\x19Solana Signed Message:\n%d%s", len(data.Nonce), data.Nonce)
			messageBytes := []byte(message)

			// Decode base58 Solana address to public key bytes
			pubKeyBytes, err := base58.Decode(linkedAddr)
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
			log.Printf("[LINK] Wallet link verification failed for %s: %v\n", linkedAddr, verifyErr)
			l.sendToClient(env.FromID, Envelope{Type: "link_wallet_response", Payload: json.RawMessage(fmt.Sprintf(`{"status":"error","message":"Verification failed: %s","address":"%s"}`, verifyErr.Error(), linkedAddr))})
			return
		}

		l.addOrUpdateLinkedWallet(primaryWallet, linkedAddr, data.LinkedChain)
		l.sendToClient(env.FromID, Envelope{Type: "link_wallet_response", Payload: json.RawMessage(fmt.Sprintf(`{"status":"success","message":"Wallet linked successfully","address":"%s"}`, linkedAddr))})
		log.Printf("[LINK] Successfully linked %s (%s) to primary AVM wallet %s\n", linkedAddr, data.LinkedChain, primaryWallet)
	case "move":
		l.mutex.RLock()
		match, ok := l.matches[env.FromID]
		l.mutex.RUnlock()
		if !ok {
			return
		}
		var move MoveData
		if err := json.Unmarshal(env.Payload, &move); err != nil {
			return
		}
		pIdx := 0
		if env.FromID == match.P2ID {
			pIdx = 1
		}
		l.mutex.Lock()
		if move.GridIndex >= 0 && move.GridIndex < 9 {
			// SECURE SYNC: Fetch card from server authoritative inventory to prevent power spoofing
			card, exists := l.inventory[move.CardID]
			if !exists {
				// Hardening: If card isn't in server cache, use a baseline weak card to prevent spoofing
				log.Printf("[SECURITY] Unauthorized CardID %d in move from %s. Using baseline power.\n", move.CardID, env.FromID)
				card = ServerCard{ID: move.CardID, Power: [4]int{5, 5, 5, 5}}
			}

			match.Board[move.GridIndex] = &ServerCard{
				ID: move.CardID, Owner: pIdx, Power: card.Power,
				Artifact: card.Artifact, Fatigue: card.Fatigue,
				Loyalty: card.Loyalty, Mood: card.Mood,
			}
			// serverCheckCaptures now returns captured cards, append them to match state
			_, flips := l.serverCheckCaptures(match, move.GridIndex, pIdx)
			match.CapturedCards = append(match.CapturedCards, flips...)

			// PILLAR 3: Hand Integrity.
			// Remove the card from the player's hand slice to prevent reuse and ensure score accuracy.
			hand := &match.P1Deck
			if pIdx == 1 {
				hand = &match.P2Deck
			}
			for i, id := range *hand {
				if id == move.CardID {
					*hand = append((*hand)[:i], (*hand)[i+1:]...)
					break
				}
			}
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
		opp, okOpp := l.wallets[data.OpponentClientID]
		rep, okRep := l.wallets[env.FromID] // Check if reporter's wallet is registered
		l.mutex.RUnlock()

		if !okRep {
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Report Failed: Your wallet is not registered."}`)})
			return
		}
		if !okOpp {
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Report Failed: Opponent's wallet not found."}`)})
			return
		}

		if okRep && okOpp {
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
			// Vitality items are generally non-combat or out-of-match-high-stakes
			l.updatePlayerPlaystyleTendenciesLocked(wallet, false, [2]int{}, []int{}, false, false)
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
			// In-match items reflect tactical intent: use match context for weighting
			l.updatePlayerPlaystyleTendenciesLocked(wallet, true, [2]int{}, []int{}, match.IsBountyMatch, match.TournamentMatchID != "")
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
		wallet, ok := l.wallets[env.FromID] // Check if wallet is registered
		if !ok {
			l.mutex.Unlock()
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Purchase Failed: Wallet not registered."}`)})
			return
		}
		stats, exists := l.leaderboard[wallet] // Check if player stats exist
		if !exists {
			l.mutex.Unlock()
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Purchase Failed: Player stats not found."}`)})
			return
		}

		// FINANCIAL VERIFICATION: Ensure player has enough rewards to cover the price
		if l.rewards[wallet] < data.Price {
			l.mutex.Unlock()
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Purchase Failed: Insufficient reward balance."}`)})
			return
		}

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

		// PILLAR 1: Professional Prestige & Industrial Unlocks.
		item, itemExists := GlobalShopRegistry[data.ItemID]
		if !itemExists {
			l.mutex.Unlock()
			return
		}

		// Role Check: Career role must match item requirement (e.g. Sentry Turrets require Security)
		if item.RequiredRole != "" && stats.JobRole != item.RequiredRole {
			l.mutex.Unlock()
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"❌ Purchase Failed: Career role '%s' required to access this hardware."}`, item.RequiredRole))})
			return
		}

		// Mojo Check: Club influence must meet item threshold
		if targetClub.Mojo < item.RequiredMojo {
			l.mutex.Unlock()
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"❌ Purchase Failed: Club Mojo too low (%d/%d). Increase turnover or defenses."}`, targetClub.Mojo, item.RequiredMojo))})
			return
		}

		// Regional Governance Check: Master Tier items require 2+ districts
		if item.IsMasterTier && len(targetClub.Territories) < 2 {
			l.mutex.Unlock()
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Purchase Failed: This is a Master Tier item. Requires Regional Governor status (2+ Districts)."}`)})
			return
		}

		// 2. Fulfillment: Deduct from Club, Grant to Player
		l.rewards[wallet] -= data.Price
		targetClub.Inventory[data.ItemID]--
		targetClub.LastActivity = time.Now()

		// Apply item-specific Mojo bonus to the club
		targetClub.Mojo += item.MojoBonus

		// INDUSTRIAL LOOP: Purchase proceeds return to Faucet liquidity
		l.faucetBalance += float64(data.Price) / 1000000.0

		if stats.Inventory == nil {
			stats.Inventory = make(map[string]int)
		}
		stats.Inventory[data.ItemID]++
		l.leaderboard[wallet] = stats

		// 3. Process Revenue (Internal call requires mutex)
		l.distributeShopRevenueLocked(data.TerritoryID, data.Price, data.ItemID)
		l.mutex.Unlock()

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
	// DELEGATED TO market_service.go
	case "trade_shares":
		l.handleTradeShares(env)
	// DELEGATED TO club_service.go
	case "heist":
		l.handleHeist(env)
	// DELEGATED TO club_service.go
	case "create_club":
		l.handleCreateClub(env)
	// DELEGATED TO club_service.go
	case "join_club":
		l.handleJoinClub(env)
	// DELEGATED TO employment_service.go
	case "hire_player":
		l.handleHirePlayer(env)
	// DELEGATED TO club_service.go
	case "purchase_territory":
		l.handlePurchaseTerritory(env)
	case "kidnap_request":
		l.handleKidnapRequest(env)
	case "pay_ransom":
		l.handlePayRansom(env)
	case "release_hostage":
		l.handleReleaseHostage(env)
	case "spread_rumor":
		l.handleSpreadRumor(env)
	case "create_lease":
		l.handleCreateLease(env)
	case "take_lease":
		l.handleTakeLease(env)
	case "spectate":
		l.handleSpectate(env)
	case "bail_card":
		l.handleBailCard(env)
	case "equip_cosmetic":
		var data struct {
			FaceplateID string `json:"faceplate_id"`
		}
		if err := json.Unmarshal(env.Payload, &data); err != nil {
			return
		}
		l.mutex.Lock()
		wallet, ok := l.wallets[env.FromID]
		if !ok {
			l.mutex.Unlock()
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Equip Failed: Wallet not registered."}`)})
			return
		}
		stats, exists := l.leaderboard[wallet] // Check if player stats exist
		if !exists {
			l.mutex.Unlock()
			return
		}
		// stats already declared above
		var success bool
		var notification string
		var auditAction string
		if !exists {
			l.mutex.Unlock()
			l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Equip Failed: Player stats not found."}`)})
			return
		}
		if data.FaceplateID == "" {
			stats.EquippedFaceplate = ""
			stats.Reputation = l.CalculateReputation(stats) // CalculateReputation is safe to call with lock held
			l.leaderboard[wallet] = stats
			success = true
			notification = "🎭 Cosmetic unequipped."
			auditAction = "COSMETIC_UNEQUIPPED"
		} else {
			if _, exists := FaceplateRegistry[data.FaceplateID]; !exists {
				notification = "❌ Equip Failed: Unknown cosmetic ID."
			} else if stats.Inventory == nil || stats.Inventory[data.FaceplateID] <= 0 {
				notification = "❌ Equip Failed: You do not own this cosmetic."
			} else {
				stats.EquippedFaceplate = data.FaceplateID
				stats.Reputation = l.CalculateReputation(stats) // CalculateReputation is safe to call with lock held
				l.leaderboard[wallet] = stats
				success = true
				notification = fmt.Sprintf("🎭 <b>COSMETIC EQUIPPED:</b> You are now wearing %s.", data.FaceplateID)
				auditAction = "COSMETIC_EQUIPPED"
			}
		}
		l.mutex.Unlock()
		if success {
			l.logAdminAudit(auditAction, wallet, fmt.Sprintf("ID: %s", data.FaceplateID))
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		}
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"%s"}`, notification))})
	default:
		log.Printf("[LOBBY] Unhandled message type: %s from %s\n", env.Type, env.FromID)
	}
}

// getClientIDFromWallet is a helper to find an active connection ID by wallet address.
func (l *Lobby) getClientIDFromWallet(wallet string) string {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.getClientIDFromWalletLocked(wallet)
}

// getClientIDFromWalletLocked is the internal version that assumes the mutex is already held.
func (l *Lobby) getClientIDFromWalletLocked(wallet string) string {
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

// cleanupNonces performs periodic maintenance on ephemeral server state.
// [AUDIT]: Pruning matchHistory is isolated from the 'matches' map.
// Active spectating sessions rely on 'matches', while 'matchHistory' is only used for reward verification.
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
		// Hardening: Only invalidate match if the disconnecting client is P1 or P2
		if client.id == match.P1ID || client.id == match.P2ID {
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
					oppWallet := match.P1Wallet
					if strings.EqualFold(wallet, match.P1Wallet) {
						oppWallet = match.P2Wallet
					}

					tourneyRound := 0
					if match.TournamentMatchID != "" {
						for _, tm := range l.tournament.Matches {
							if tm.ID == match.TournamentMatchID {
								tourneyRound = tm.Round
								break
							}
						}
					}
					l.incrementDNF(wallet, tourneyRound, oppWallet, match.TournamentMatchID) // Pass match context for on-chain log
				}

				// PILLAR 3: Tournament Disconnection Protocol.
				// If a player leaves during a tournament match, award the advancement to the opponent.
				if match.TournamentMatchID != "" {
					if oppWallet, ok := l.wallets[opponentID]; ok {
						log.Printf("[TOURNAMENT] Awarding win to %s due to opponent DNF.\n", oppWallet)
						l.processTournamentResult(match.TournamentMatchID, oppWallet)
					}
				}
				delete(l.matches, opponentID)
			}
			delete(l.matches, client.id)
		} else {
			// It's a spectator: just remove from spectators list
			var remaining []string
			for _, sID := range match.Spectators {
				if sID != client.id {
					remaining = append(remaining, sID)
				}
			}
			match.Spectators = remaining
			delete(l.matches, client.id)
		}
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

		// Spectator Broadcast Logic:
		// If this is a move message, also send it to everyone in match.Spectators
		if env.Type == "move" {
			if match, ok := l.matches[env.ToID]; ok {
				for _, sID := range match.Spectators {
					if sID == env.FromID {
						continue
					} // Don't echo to sender
					if s, ok := l.clients[sID]; ok {
						select {
						case s.send <- message:
						default:
						}
					}
				}
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

// handleSpectate allows a client to join an ongoing match as a viewer.
func (l *Lobby) handleSpectate(env *Envelope) {
	var data struct {
		TargetID string `json:"target_id"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Find the match associated with the target client
	match, ok := l.matches[data.TargetID]
	if !ok {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Stream Error: Match no longer active."}`)})
		return
	}

	// Prevent duplicate entries in spectators list
	alreadySpectating := false
	for _, sID := range match.Spectators {
		if sID == env.FromID {
			alreadySpectating = true
			break
		}
	}
	if !alreadySpectating {
		match.Spectators = append(match.Spectators, env.FromID)
		l.matches[env.FromID] = match // Map session to match for move routing
	}

	// Marshal entire MatchState which now includes snake_case tags for penalty snapshots
	payload, _ := json.Marshal(match)
	l.sendToClientLocked(env.FromID, Envelope{
		Type:    "match_start",
		FromID:  "SERVER",
		Payload: payload,
	})

	log.Printf("[LOBBY] Client %s is now spectating match %s vs %s\n", env.FromID, match.P1ID, match.P2ID)
}

func (l *Lobby) getLobbyUpdateMsgLocked() []byte {
	type playerInfo struct {
		ID                string         `json:"id"`
		IsAdmin           bool           `json:"is_admin"`
		AvatarURL         string         `json:"avatar_url"`
		Gloat             string         `json:"gloat"`
		AvatarNotice      string         `json:"avatar_notice"`
		BanExpires        time.Time      `json:"ban_expires"`
		HasMardonBadge    bool           `json:"has_mardon_badge"`
		Wins              int            `json:"wins"`
		Reputation        int            `json:"reputation"`
		AuctionsWon       int            `json:"auctions_won"`
		VirtualBalance    uint64         `json:"virtual_balance"`
		WantedLevel       int            `json:"wanted_level"`
		Cunning           int            `json:"cunning"`
		Mojo              int            `json:"mojo"`
		Nurturing         int            `json:"nurturing"`
		JailedCards       map[int]string `json:"jailed_cards"`       // Added for UI display
		SocialRank        string         `json:"social_rank"`        // Added for UI display
		EquippedFaceplate string         `json:"equipped_faceplate"` // For UI rendering
		KidnappedCards    map[int]string `json:"kidnapped_cards"`    // Added for UI display
		MatchHistory      []MatchHistory `json:"match_history"`      // Last 5 matches for immersion
		HeldHostageCards  map[int]string `json:"held_hostage_cards"`
		Achievements      []string       `json:"achievements"` // Added for UI display
		Playstyle         PlaystyleTendencies `json:"playstyle"`
		// JobRole and EmployerID are already present in the playerInfo struct
		RumorCount int    `json:"rumor_count"` // Added for UI display
		JobRole    string `json:"job_role"`    // Manager, Security, Clerk
		EmployerID string `json:"employer_id"`
	}
	var players []playerInfo
	for _, client := range l.clients {
		hasMardon := false
		var banExpires time.Time 
		wins, reputation, wanted, cunning, nurturing, mojo, auctionsWon, vBal := 0, 0, 0, 0, 0, 0, 0, uint64(0)
		var jailedCards, kidnappedCards, heldHostageCards map[int]string
		var matches []MatchHistory
		var equippedFaceplate string
		var socialRank string
		var achievements []string
		var jobRole string
		var employerID string
		var rumorCount int
		var playstyle PlaystyleTendencies
		if wallet, ok := l.wallets[client.id]; ok {
			if stats, exists := l.leaderboard[wallet]; exists {
				banExpires = stats.BanExpires
				wins = stats.Wins
				reputation = stats.Reputation
				vBal = l.playerBalances[wallet]
				auctionsWon = stats.AuctionsWon
				// UI Sync: Use Effective Mojo (including faceplate) for Career Path display
				mojo = stats.GetEffectiveMojo()
				wanted = stats.WantedLevel
				// Alignment: Broadcast the Effective Cunning (including faceplate/penalty)
				// to ensure the UI heist heuristic matches the server calculation.
				cunning = stats.GetEffectiveCunning()
				nurturing = stats.Nurturing
				jailedCards = stats.JailedCards
				kidnappedCards = stats.KidnappedCards
				heldHostageCards = stats.HeldHostageCards
				playstyle = stats.Playstyle
				equippedFaceplate = stats.EquippedFaceplate
				socialRank = stats.SocialRank
				jobRole = stats.JobRole
				employerID = stats.EmployerClubID
				achievements = stats.Achievements
				// PILLAR 4: Historical Immersion. Send the last 5 matches for display.
				matches = stats.History
				if len(matches) > 5 {
					matches = matches[:5]
				}
				if stats.Wins >= 50 && stats.DisconnectStreak == 0 {
					hasMardon = true
				}
				rumorCount = stats.RumorCount
			}
		}
		players = append(players, playerInfo{
			ID: client.id, IsAdmin: client.isAdmin, AvatarURL: client.avatarURL,
			Gloat: client.gloat, AvatarNotice: client.avatarBanNotice,
			BanExpires: banExpires, HasMardonBadge: hasMardon, Wins: wins, Reputation: reputation,
			AuctionsWon: auctionsWon, VirtualBalance: vBal,
			WantedLevel: wanted, Cunning: cunning, Nurturing: nurturing, Mojo: mojo,
			MatchHistory: matches,
			JailedCards:      jailedCards,
			KidnappedCards:   kidnappedCards,
			HeldHostageCards: heldHostageCards,
			SocialRank:       socialRank, EquippedFaceplate: equippedFaceplate,
			Achievements:     achievements, RumorCount: rumorCount,
			Playstyle:        playstyle,
			JobRole: jobRole, EmployerID: employerID,
		})
	}

	update := struct {
		Players           []playerInfo             `json:"players"`
		MaintenanceActive bool                     `json:"maintenance_active"`
		MaintenanceTime   time.Time                `json:"maintenance_time"`
		FaucetBalance     float64                  `json:"faucet_balance"`
		Clubs             map[string]*Club         `json:"clubs"`
		RewardStack       map[string]uint64        `json:"reward_stack"`
		ActiveMatchCount  int                      `json:"active_match_count"`
		Tournament        TournamentState          `json:"tournament"`
		AvailableNetworks map[string]NetworkConfig `json:"available_networks"`
		Rumors            map[string]*Rumor        `json:"rumors"` // Added for UI display
		AdminFocusNetwork string                   `json:"admin_focus_network"`
		BannedAvatars     map[string]time.Time     `json:"banned_avatars"`
		BlackMarket       []Loan                   `json:"black_market"` // Added for real-time economy feel
	}{
		Players: players, MaintenanceActive: l.maintenanceMode,
		MaintenanceTime: l.maintenanceTime,
		Clubs:           l.clubs,
		RewardStack:     l.rewardStack, FaucetBalance: l.faucetBalance,
		ActiveMatchCount: len(l.matches) / 2, Tournament: l.tournament,
		AvailableNetworks: l.availableNetworks, AdminFocusNetwork: l.adminFocusNetwork,
		Rumors:        l.rumors,
		BannedAvatars: l.bannedAvatars,
		BlackMarket:   l.blackMarket,
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

	// 0. TOURNAMENT LOCK ANALYSIS: Identify players who MUST play their tournament match.
	// This prevents bracket participants from being "stolen" by Standard or Bounty matchmaking
	// if their assigned opponent hasn't joined the pool yet.
	tourneyLocked := make(map[string]bool)
	if l.tournament.Active && l.tournament.CurrentRound > 0 {
		for _, match := range l.tournament.Matches {
			if match.Round == l.tournament.CurrentRound && match.Winner == "" && match.P2 != "BYE" {
				tourneyLocked[strings.ToLower(match.P1)] = true
				tourneyLocked[strings.ToLower(match.P2)] = true
			}
		}
	}

	// 1. TOURNAMENT PRIORITY PASS: Pair players scheduled in the current bracket round
	if l.tournament.Active && l.tournament.CurrentRound > 0 {
		for _, match := range l.tournament.Matches {
			if match.Round == l.tournament.CurrentRound && match.Winner == "" && match.P2 != "BYE" {
				idx1, idx2 := -1, -1
				for k, entry := range l.matchmakingPool {
					if matchedIndices[k] {
						continue
					}
					if strings.EqualFold(entry.Wallet, match.P1) {
						idx1 = k
					}
					if strings.EqualFold(entry.Wallet, match.P2) {
						idx2 = k
					}
				}
				if idx1 != -1 && idx2 != -1 {
					if l.initiatePairedMatch(l.matchmakingPool[idx1].ClientID, l.matchmakingPool[idx2].ClientID) {
						// Link to bracket for automatic result reporting
						if mState, ok := l.matches[l.matchmakingPool[idx1].ClientID]; ok {
							mState.TournamentID = l.tournament.ID
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

		// Skip if player belongs to an active tournament match but opponent isn't here yet.
		if tourneyLocked[strings.ToLower(l.matchmakingPool[i].Wallet)] {
			continue
		}

		p1 := l.matchmakingPool[i]
		p1Wanted := l.leaderboard[p1.Wallet].WantedLevel

		for j := i + 1; j < len(l.matchmakingPool); j++ {
			if matchedIndices[j] {
				continue
			}
			if tourneyLocked[strings.ToLower(l.matchmakingPool[j].Wallet)] {
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
		if tourneyLocked[strings.ToLower(l.matchmakingPool[i].Wallet)] {
			continue
		}

		p1 := l.matchmakingPool[i]
		for j := i + 1; j < len(l.matchmakingPool); j++ {
			if matchedIndices[j] {
				continue
			}
			if tourneyLocked[strings.ToLower(l.matchmakingPool[j].Wallet)] {
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

	p1Wallet := l.wallets[id1]
	p2Wallet := l.wallets[id2]
	p1Stats := l.leaderboard[p1Wallet]
	p2Stats := l.leaderboard[p2Wallet]

	// PILLAR 3: Environment Authorization.
	// Determine territory and authoritative moods before match initialization.
	territoryID := l.assignMatchTerritoryLocked()
	matchRules := map[string]bool{
		"Open": true, "Power_copy": false, "Power_up": false,
		"Elemental_sync": true, "Fallen_penalty": true, "Sudden_death": true,
	}

	var boardMoods [9]string
	moodTypes := []string{"Volatile", "Serene", "Spirited", "Grounded"}
	for i := 0; i < 9; i++ {
		if rand.Intn(10) > 7 { // 20% chance for a tile mood
			boardMoods[i] = moodTypes[rand.Intn(len(moodTypes))]
		} else {
			boardMoods[i] = "Neutral"
		}
	}

	// PILLAR 1: Regional Power Boost Calculation.
	// Determine if the territory belongs to a Region (2+ districts)
	// and if players are affiliated with the owning club.
	p1Boost, p2Boost := false, false
	if owningClub := l.getClubByTerritoryID(territoryID); owningClub != nil && len(owningClub.Territories) >= 2 {
		// Check P1
		if strings.EqualFold(owningClub.OwnerWallet, p1Wallet) {
			p1Boost = true
		} else if _, ok := owningClub.Members[strings.ToLower(p1Wallet)]; ok {
			p1Boost = true
		}
		// Check P2
		if strings.EqualFold(owningClub.OwnerWallet, p2Wallet) {
			p2Boost = true
		} else if _, ok := owningClub.Members[strings.ToLower(p2Wallet)]; ok {
			p2Boost = true
		}
	}

	match := &MatchState{
		P1ID: id1, P2ID: id2, P1Deck: m1.P1Deck, P2Deck: m2.P1Deck,
		P1Wallet:        p1Wallet,
		P2Wallet:        p2Wallet,
		Rules:           matchRules,
		BoardMoods:      boardMoods,
		P1WantedLevel:   p1Stats.WantedLevel,
		P2WantedLevel:   p2Stats.WantedLevel,
		P1Cunning:       p1Stats.GetEffectiveCunning(),
		P1Nurturing:     p1Stats.Nurturing,
		P2Cunning:       p2Stats.GetEffectiveCunning(),
		P2Nurturing:     p2Stats.Nurturing,
		Round:           1, // Standard match initialization
		TerritoryID:     territoryID,
		P1RegionalBoost: p1Boost,
		P2RegionalBoost: p2Boost,
		ActiveItemBuffs: make(map[string]map[string]int),
	}

	if c1, ok := l.clients[id1]; ok {
		match.P1Avatar, match.P1Gloat = c1.avatarURL, c1.gloat
	}
	if c2, ok := l.clients[id2]; ok {
		match.P2Avatar, match.P2Gloat = c2.avatarURL, c2.gloat
	}

	l.matches[id1], l.matches[id2] = match, match

	p1Sync, _ := json.Marshal(Envelope{
		Type: "challenge", FromID: id2, ToID: id1,
		Payload: json.RawMessage(fmt.Sprintf(`{"action":"accept","deck":%v,"wanted_level":%d,"territory":"%s","p1_boost":%v,"p2_boost":%v,"moods":%v}`, jsonList(match.P2Deck), match.P2WantedLevel, match.TerritoryID, match.P1RegionalBoost, match.P2RegionalBoost, jsonListString(match.BoardMoods[:]))),
	})
	p2Sync, _ := json.Marshal(Envelope{
		Type: "challenge", FromID: id1, ToID: id2,
		Payload: json.RawMessage(fmt.Sprintf(`{"action":"sync_back","deck":%v,"wanted_level":%d,"territory":"%s","p1_boost":%v,"p2_boost":%v,"moods":%v}`, jsonList(match.P1Deck), match.P1WantedLevel, match.TerritoryID, match.P1RegionalBoost, match.P2RegionalBoost, jsonListString(match.BoardMoods[:]))),
	})

	if c1, ok := l.clients[id1]; ok {
		c1.send <- p1Sync
	}
	if c2, ok := l.clients[id2]; ok {
		c2.send <- p2Sync
	}

	l.sendToClient(id1, Envelope{Type: "matchmaking_status", Payload: json.RawMessage(`{"status":"match_found"}`)})
	l.sendToClient(id2, Envelope{Type: "matchmaking_status", Payload: json.RawMessage(`{"status":"match_found"}`)})

	log.Printf("[MATCHMAKING] Duel started on %s between %s and %s. Boosts - P1: %v, P2: %v\n", 
		territoryID, p1Wallet, p2Wallet, p1Boost, p2Boost)
	return true
}
// This prioritizes owned territories to trigger Industrial Loop mechanics (jailing/revenue).
func (l *Lobby) assignMatchTerritoryLocked() string {
	// 1. Compile pool of territories currently claimed by Clubs
	var ownedTerritories []string
	for _, club := range l.clubs {
		ownedTerritories = append(ownedTerritories, club.Territories...)
	}

	// 2. INDUSTRIAL LOOP Priority: High chance to fight on Club Turf
	if len(ownedTerritories) > 0 && rand.Float64() < 0.70 {
		// Pick a random territory that a club has actually invested in
		return ownedTerritories[rand.Intn(len(ownedTerritories))]
	}

	// 3. Fallback: Standard neutral grounds if no clubs exist or for variety
	neutralPool := []string{"the_lab", "casino", "arena_center", "north_district", "south_slums", "east_gate", "west_port", "the_archive", "data_haven"}
	return neutralPool[rand.Intn(len(neutralPool))]
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

// jsonListString is a helper to marshal a slice of strings to a JSON string.
func jsonListString(strs []string) string {
	b, _ := json.Marshal(strs)
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
	_, ok := l.clients[clientID]
	l.mutex.RUnlock()
	if !ok {
		return
	}
	l.sendToClientLocked(clientID, env)
}

// sendToClientLocked sends an Envelope message to a specific client, assuming the lock is held.
func (l *Lobby) sendToClientLocked(clientID string, env Envelope) {
	client, ok := l.clients[clientID]
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
	data, err := os.ReadFile(l.getDataPath(regCacheName))
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
	data, err := os.ReadFile(l.getDataPath(linkedWalletsName))
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
	if err := os.WriteFile(l.getDataPath(linkedWalletsName), data, 0644); err != nil {
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
func (l *Lobby) updatePlayerPlaystyleTendencies(wallet string, inMatchContext bool, scores [2]int, deck []int, isBountyMatch bool, isTournament bool) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.updatePlayerPlaystyleTendenciesLocked(wallet, inMatchContext, scores, deck, isBountyMatch, isTournament)
}

// updatePlayerPlaystyleTendenciesLocked is the internal version that assumes the mutex is already held.
func (l *Lobby) updatePlayerPlaystyleTendenciesLocked(wallet string, inMatchContext bool, scores [2]int, deck []int, isBountyMatch bool, isTournament bool) {

	stats, exists := l.leaderboard[wallet]
	if !exists {
		return
	}

	// PILLAR 3: Dynamic Intensity Weighting.
	// Tournament matches reveal core "clutch" behaviors and carry double weight.
	alpha := 0.2
	if isTournament {
		alpha = 0.4
	} else if isBountyMatch {
		alpha = 0.3
	}

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
		match, matchExists := l.matches[l.getClientIDFromWalletLocked(wallet)] // FIXED: Deadlock risk
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
		// PILLAR 3: Behavioral Relevance logic.
		// Adjusted Decay (0.99 per minute) ensures "Trends" persist over several hours (~50% half-life).
		const decayFactor = 0.99
		const cleanupThreshold = 0.05 // Deletes entries below this to prevent state bloat

		// 1. Preference Decay with State Pruning
		// Refactored into a reusable helper to handle rules, moods, and items.
		decayAndPrune := func(m map[string]float64) {
			if m == nil {
				return
			}
			for k, v := range m {
				newVal := v * decayFactor
				if newVal < cleanupThreshold {
					delete(m, k)
				} else {
					m[k] = newVal
				}
			}
		}

		decayAndPrune(stats.Playstyle.PreferredRules)
		decayAndPrune(stats.Playstyle.PreferredCardMoods)
		decayAndPrune(stats.Playstyle.PreferredItems)

		// 2. Trait Normalization (Aggressiveness & Risk Tolerance)
		// Instead of decaying to zero, we decay towards a neutral 0.5 baseline.
		// This ensures inactive players' commentary eventually reverts to "Standard" behavior
		// rather than retaining "Extreme" taunts from days-old matches.
		const epsilon = 0.001
		normalize := func(val float64) float64 {
			if math.Abs(val-0.5) < epsilon {
				return 0.5
			}
			return 0.5 + (val-0.5)*decayFactor
		}

		stats.Playstyle.Aggressiveness = normalize(stats.Playstyle.Aggressiveness)
		stats.Playstyle.RiskTolerance = normalize(stats.Playstyle.RiskTolerance)

		// PILLAR 3: Identity Sync. Ensure root trait fields match behavioral analysis.
		stats.Aggressiveness = stats.Playstyle.Aggressiveness
		stats.RiskTolerance = stats.Playstyle.RiskTolerance

		l.leaderboard[wallet] = stats
	}
}

// simulateTournament orchestrates a full tournament simulation, including bracket generation and match results.
func (l *Lobby) simulateTournament(size int, isBuyIn bool) {
	l.mutex.Lock()

	log.Printf("[SIMULATION] Starting %d-player tournament simulation (Buy-in: %v)...\n", size, isBuyIn)

	// Identify clubs for member assignment testing
	var clubIDs []string
	for id := range l.clubs {
		clubIDs = append(clubIDs, id)
	}

	// 1. Generate mock participants
	participants := make([]string, size)
	for i := 0; i < size; i++ {
		mockWallet := fmt.Sprintf("SIM_PLAYER_%d_%d", i, time.Now().UnixNano()%10000)
		participants[i] = mockWallet
		stats := PlayerStats{
			Reputation: rand.Intn(1000),
			Wins:       rand.Intn(50),
		}

		// PILLAR 1: Kickback Verification Setup.
		// Assign 30% of participants to existing clubs to ensure kickbacks are distributed during the simulation.
		if len(clubIDs) > 0 && rand.Float64() < 0.30 {
			cid := clubIDs[rand.Intn(len(clubIDs))]
			stats.EmployerClubID = cid
			if l.clubs[cid].Members == nil {
				l.clubs[cid].Members = make(map[string]time.Time)
			}
			l.clubs[cid].Members[strings.ToLower(mockWallet)] = time.Now().Add(-2 * time.Hour)
		}

		l.leaderboard[mockWallet] = stats
		l.ensurePlayerStatsMapsInitialized(mockWallet)
		// Simulate registration to trigger kickback logic (if isBuyIn)
		if isBuyIn {
			l.paidParticipants = append(l.paidParticipants, mockWallet)
			l.faucetBalance += (50.0 / 2.0) // Simulate half buy-in to pot
			l.tournamentPotBonus += (50.0 / 2.0)
			// PILLAR 3: Deadlock Fix. Use variant that assumes lock is held.
			l.distributeTournamentKickbackLocked(mockWallet, uint64(50*1000000), time.Now(), "Voi")
		}
	}

	// 2. Initialize tournament state (similar to handleStartTournament)
	matches := []TournamentMatch{}
	seedMap := map[int][]int{8: {0, 7, 3, 4, 1, 6, 2, 5}, 16: {0, 15, 7, 8, 4, 11, 3, 12, 1, 14, 6, 9, 5, 10, 2, 13}}[size]

	if seedMap == nil {
		log.Printf("[SIMULATION ERROR] Invalid tournament size: %d\n", size)
		l.mutex.Unlock()
		return
	}

	for i := 0; i < len(seedMap); i += 2 {
		matches = append(matches, TournamentMatch{
			ID: fmt.Sprintf("R1-M%d", (i/2)+1), P1: participants[seedMap[i]], P2: participants[seedMap[i+1]], Round: 1,
		})
	}

	pot := 500.0
	if isBuyIn {
		pot += l.tournamentPotBonus
		l.tournamentPotBonus = 0
	}

	l.tournament = TournamentState{
		Active:       true,
		ID:           fmt.Sprintf("SIM-T-%d", time.Now().Unix()),
		Participants: participants,
		Matches:      matches,
		CurrentRound: 1,
		Pot:          pot,
		BuyInAmount:  50.0,
		IsBuyInMode:  isBuyIn,
		OpenTime:     time.Now().Add(-1 * time.Hour), // Set in the past for registration
	}
	l.mutex.Unlock()

	// 3. Simulate rounds until a winner is determined
	for {
		l.mutex.Lock()
		if !l.tournament.Active || len(l.tournament.Matches) == 0 {
			l.mutex.Unlock()
			break
		}

		currentRoundMatches := []TournamentMatch{}
		for _, m := range l.tournament.Matches {
			if m.Round == l.tournament.CurrentRound && m.Winner == "" {
				currentRoundMatches = append(currentRoundMatches, m)
			}
		}

		if len(currentRoundMatches) == 0 {
			l.mutex.Unlock()
			break
		} // No more matches in this round

		for _, m := range currentRoundMatches {
			winner := m.P1 // Simulated outcome
			if rand.Intn(2) == 1 { winner = m.P2 }
			l.processTournamentResult(m.ID, winner) // This will advance rounds and finalize
		}
		l.mutex.Unlock()
		// PILLAR 3: Performance Hardening. Pulse the lock to allow standard lobby traffic.
		time.Sleep(100 * time.Millisecond)
	}
	log.Printf("[SIMULATION] Tournament simulation complete. Final Winner ID recorded in archival summary.\n")
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
	decayOccurred := false

	for _, club := range l.clubs {
		if club.Mojo <= 0 {
			continue
		}

		if now.Sub(club.LastActivity) > stagnationThreshold {
			// PILLAR 1: Dynamic Decay Scaling.
			// Larger clubs lose more Mojo to maintain competitive churn.

			isRegion := len(club.Territories) >= 2
			decayRate := 0.02
			minDecay := 5

			if isRegion {
				// PILLAR 1: Regional Governor Accountability.
				// Established regions suffer 2.5x higher decay to prevent sector stagnation.
				decayRate = 0.05
				minDecay = 15
			}

			// PILLAR 1: Inactive Member Scaling.
			// Larger clubs lose Mojo faster when stagnant to reflect organizational overhead.
			// Add 0.2% to the decay rate for every member (e.g. 50 members = +10% rate).
			decayRate += float64(len(club.Members)) * 0.002

			decayAmount := int(float64(club.Mojo)*decayRate + 0.5)
			if decayAmount < minDecay {
				decayAmount = minDecay
			}

			club.Mojo -= decayAmount
			if club.Mojo < 0 {
				club.Mojo = 0
			}
			decayOccurred = true
			log.Printf("[INDUSTRIAL] Club %s suffered Mojo decay (isRegion: %v). New Mojo: %d\n", club.Name, isRegion, club.Mojo)

			// PILLAR 1: Rippled Standing Decay.
			// Recalculate reputation for all employees whose standing relies on this club's Mojo.
			for wallet, stats := range l.leaderboard {
				if stats.EmployerClubID == club.ID {
					stats.Reputation = l.CalculateReputation(stats)
					l.leaderboard[wallet] = stats
				}
			}
			// Reset clock to 'now' so decay is periodic (e.g., every 48h) rather than continuous
			club.LastActivity = now
		}
	}

	if decayOccurred {
		// Trigger global sync so UI reflects the Mojo loss
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
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
	type highlight struct {
		W string `json:"w"` // Wallet
		A string `json:"a"` // Award
		M string `json:"m"` // Meta/Detail
	}

	var standings []hofEntry
	var highlights []highlight

	var topMojo int = -1
	var mojoKing string

	for w, s := range l.leaderboard {
		if s.Wins > 0 {
			standings = append(standings, hofEntry{W: w, V: s.Wins, R: s.BestRating})
		}

		// PILLAR 4: Prestige Highlights.
		// 1. Identify Tournament Champions using achievement triggers
		for _, ach := range s.Achievements {
			if ach == "TOURNAMENT_CHAMPION" {
				// Scan history for the most recent Tournament ID
				eventID := "Elite Event"
				for _, h := range s.History {
					if h.TournamentID != "" && h.WinnerIndex == 0 {
						eventID = h.TournamentID
						break
					}
				}
				highlights = append(highlights, highlight{W: w, A: "Tournament Champion", M: eventID})
				break
			}
		}

		// 2. Identify High-Finance Leaders (Art Collectors)
		if s.AuctionsWon >= 3 {
			highlights = append(highlights, highlight{W: w, A: "Master Collector", M: fmt.Sprintf("%d Gallery Victories", s.AuctionsWon)})
		}

		// 3. Track social peak for Mojo highlight
		if s.Mojo > topMojo {
			topMojo = s.Mojo
			mojoKing = w
		}
	}
	sort.Slice(standings, func(i, j int) bool { return standings[i].V > standings[j].V })

	if mojoKing != "" && topMojo > 0 {
		highlights = append(highlights, highlight{W: mojoKing, A: "Social Titan", M: fmt.Sprintf("%d Mojo", topMojo)})
	}

	// Take Top 10 for the archive note
	limit := 10
	if len(standings) < limit {
		limit = len(standings)
	}

	summary := struct {
		Season     int         `json:"season"`
		Start      time.Time   `json:"start"`
		End        time.Time   `json:"end"`
		Highlights []highlight `json:"highlights,omitempty"`
		Top        []hofEntry  `json:"top"`
	}{
		Season:     l.seasonNumber,
		Start:      l.seasonStart,
		End:        time.Now(),
		Highlights: highlights,
		Top:        standings[:limit],
	}

	jsonData, _ := json.Marshal(summary)
	note := fmt.Sprintf("VBT_SEASON_ARCHIVE:%s", string(jsonData))
	l.sendNoteTx(note) // Record on chain

	// Reset Cycle
	l.seasonNumber++
	l.seasonStart = time.Now()
	l.leaderboard = make(map[string]PlayerStats) // Clear HoF

	// Persist new config while preserving initialRewards
	l.saveSeasonMetadataLocked()
}

// ensurePlayerStatsMapsInitialized ensures that all map fields in PlayerStats are initialized.
func (l *Lobby) ensurePlayerStatsMapsInitialized(wallet string) {
	stats := l.leaderboard[wallet]
	if stats.Inventory == nil {
		stats.Inventory = make(map[string]int)
	}
	if stats.Relationships == nil {
		stats.Relationships = make(map[string]int)
	}
	if stats.Portfolio == nil {
		stats.Portfolio = make(map[string]float64)
	}
	if stats.JailedCards == nil {
		stats.JailedCards = make(map[int]string)
	}
	if stats.KidnappedCards == nil {
		stats.KidnappedCards = make(map[int]string)
	}
	if stats.HeldHostageCards == nil {
		stats.HeldHostageCards = make(map[int]string)
	}
	if stats.PreferredRules == nil {
		stats.PreferredRules = make(map[string]int)
	}
	if stats.Moods == nil {
		stats.Moods = make(map[string]int)
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

// refreshRegionalRoles updates governor status for clubs based on territory control.
func (l *Lobby) refreshRegionalRoles() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	for _, club := range l.clubs {
		if len(club.Territories) >= 2 {
			// Set a default region name for governors, or keep existing if set
			if club.RegionName == "" {
				club.RegionName = "Governor"
			}

			// PILLAR 1: Achievement Integration.
			// Grant the GOVERNOR trophy to the club owner for expanded regional influence.
			l.unlockAchievementLocked(strings.ToLower(club.OwnerWallet), "GOVERNOR")
		} else {
			// Remove governor status if they no longer control 2+ territories
			club.RegionName = ""
		}
	}
}
