//go:build !js && !wasm

package main

import (
	"context"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"crypto/ed25519"

	"compress/gzip"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template" // For escapeHTML
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	// For Solana verification

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/gorilla/websocket"
	"github.com/mr-tron/base58"
)

// For Solana address decoding

var safeAvatarPool = []string{
	"Cards/Alana.webp",
	"Cards/Bella.webp",
	"Cards/Clohey.webp",
	"Cards/Ellie.webp",
	"Cards/Fran.webp",
}

// NewGracePeriodMatrix initializes the connection quarantine manager.
func (l *Lobby) NewGracePeriodMatrix(grace time.Duration, evictionCallback func(string)) *GracePeriodMatrix {
	return &GracePeriodMatrix{
		ActiveSessions:  make(map[string]*PlayerSession),
		DisconnectGrace: grace,
		EvictionWorker:  evictionCallback,
	}
}

// HandleConnectionDrop intercepts unexpected WebSocket disconnect events.
// PILLAR 4: Connection Quarantine.
func (gpm *GracePeriodMatrix) HandleConnectionDrop(walletAddress string) {
	gpm.Mu.Lock()
	session, exists := gpm.ActiveSessions[walletAddress]
	if !exists || session.CurrentState != StateConnected {
		gpm.Mu.Unlock()
		return
	}

	// 1. Pivot session state to Quarantine boundaries
	session.CurrentState = StatePendingDisconnect
	log.Printf(" [Grace Matrix] Player %s lost connection. Entering 60s quarantine window.\n", walletAddress)

	// 2. Spawn an atomic, cancellable context for the countdown timer
	ctx, cancel := context.WithCancel(context.Background())
	session.CancelTimer = cancel
	gpm.Mu.Unlock()

	// 3. Launch the asynchronous countdown tracker
	go func(wAddress string, trackingCtx context.Context) {
		select {
		case <-time.After(gpm.DisconnectGrace):
			// The timer completed natively without reconnection intervention
			gpm.Mu.Lock()
			currentSession, stillExists := gpm.ActiveSessions[wAddress]
			if stillExists && currentSession.CurrentState == StatePendingDisconnect {
				delete(gpm.ActiveSessions, wAddress)
				gpm.Mu.Unlock()

				log.Printf(" [Grace Matrix] Quarantine expired for %s. Triggering forfeit eviction.\n", wAddress)
				gpm.EvictionWorker(wAddress) // Execute harsh match forfeit rule
				return
			}
			gpm.Mu.Unlock()

		case <-trackingCtx.Done():
			// The context was explicitly cancelled by a successful reconnection handshake
			log.Printf(" [Grace Matrix] Disconnect timer safely aborted for player: %s\n", wAddress)
			return
		}
	}(walletAddress, ctx)
}

// HandleReconnectionHandshake processes inbound user catch-up attempts.
func (gpm *GracePeriodMatrix) HandleReconnectionHandshake(walletAddress string) (uint64, error) {
	gpm.Mu.Lock()
	defer gpm.Mu.Unlock()

	session, exists := gpm.ActiveSessions[walletAddress]
	if !exists {
		// New login session initialization entirely
		gpm.ActiveSessions[walletAddress] = &PlayerSession{
			WalletAddress:   walletAddress,
			CurrentState:    StateConnected,
			LastActiveFrame: 0,
		}
		return 0, nil
	}

	// 🛡️ Safe Reconnection Window Intercepted:
	// Kill the background eviction countdown before it fires the forfeit rule
	if session.CancelTimer != nil {
		session.CancelTimer()
	}

	session.CurrentState = StateConnected
	log.Printf(" [Grace Matrix] Player %s reconnected within window.\n", walletAddress)

	return session.LastActiveFrame, nil
}

// handleAuthoritativeForfeit executes the match cleanup for a player who failed to reconnect.
func (l *Lobby) handleAuthoritativeForfeit(wallet string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Find the ClientID associated with this wallet
	clientID := ""
	for id, w := range l.wallets {
		if strings.EqualFold(w, wallet) {
			clientID = id
			break
		}
	}

	if clientID == "" {
		return
	}

	// Execute Match Forfeit Logic
	if match, ok := l.matches[clientID]; ok {
		isP1 := clientID == match.P1ID
		isP2 := clientID == match.P2ID

		if isP1 || isP2 {
			// PILLAR 4: Combatant Eviction.
			opponentID := match.P2ID
			if isP2 {
				opponentID = match.P1ID
			}
			if opponentID != "" {
				// Award advancement/win to the opponent
				if match.TournamentMatchID != "" {
					if oppWallet, ok := l.wallets[opponentID]; ok {
						l.tournamentService.ProcessTournamentResult(l, match.TournamentMatchID, oppWallet)
					}
				}
				delete(l.matches, opponentID)
			}
			delete(l.matchHandshakers, match.P1ID) // PILLAR 4: Handshaker Pruning.
		} else {
			// PILLAR 4: Spectator Eviction. 
			// Timeout reached without reconnection; remove from stream list.
			var remaining []string
			for _, sID := range match.Spectators {
				if sID != clientID { remaining = append(remaining, sID) }
			}
			match.Spectators = remaining
		}

		delete(l.matches, clientID)

		// Penalize the leaver
		l.incrementDNF(wallet, 0, "", match.TournamentMatchID)
	}

	delete(l.wallets, clientID)
	l.UntrackSession(wallet)

	// Sync UI to remove the dead session from lobby lists
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// NewSyncHandshaker initializes the multi-frame state verification engine.
func (l *Lobby) NewSyncHandshaker() *SyncHandshaker {
	return &SyncHandshaker{
		HistoricalFrames: make(map[uint64]FrameDelta),
	}
}

// ReconcileFrame verifies client speculative frames against authoritative records.
// PILLAR 4: Frame sequence matching.
func (sh *SyncHandshaker) ReconcileFrame(clientFrame FrameDelta) (bool, error) {
	sh.Mu.Lock()
	defer sh.Mu.Unlock()

	// PILLAR 4: Duplicate Sequence Guard.
	// Gracefully ignore frames that have already been processed to support rapid reconnection retries.
	if clientFrame.SequenceID <= sh.CurrentSequence {
		return true, nil
	}

	// Verify continuous, chronological frame increments
	if clientFrame.SequenceID != sh.CurrentSequence+1 {
		return false, fmt.Errorf("sequence gap: expected %d, got %d", sh.CurrentSequence+1, clientFrame.SequenceID)
	}

	// Commit frame to recovery history log
	sh.CurrentSequence = clientFrame.SequenceID
	sh.HistoricalFrames[sh.CurrentSequence] = clientFrame
	sh.LastVerifiedHash = clientFrame.StateHash

	return true, nil
}

// CatchUpPlayer generates a structural playback history buffer to clear network drift.
func (sh *SyncHandshaker) CatchUpPlayer(fromSequence uint64) []FrameDelta {
	sh.Mu.RLock()
	defer sh.Mu.RUnlock()

	var recoveryBuffer []FrameDelta
	for i := fromSequence + 1; i <= sh.CurrentSequence; i++ {
		if frame, exists := sh.HistoricalFrames[i]; exists {
			recoveryBuffer = append(recoveryBuffer, frame)
		}
	}
	return recoveryBuffer
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

	// PILLAR 7-D: AI Autonomous Economy — Behavioral loop ticker (~60s per citizen cycle)
	citizenBehaviorTicker := time.NewTicker(30 * time.Second)
	defer citizenBehaviorTicker.Stop()

	matchmakingTicker := time.NewTicker(5 * time.Second)
	defer matchmakingTicker.Stop()
	vaultCheckTicker := time.NewTicker(5 * time.Minute)
	defer vaultCheckTicker.Stop()
	healthTicker := time.NewTicker(10 * time.Minute)
	defer healthTicker.Stop()
	cacheSaveTicker := time.NewTicker(15 * time.Minute)
	defer cacheSaveTicker.Stop()

	go l.oracleService.RefreshGlobalLeaderboard(l)

	for {
		select {
		case <-ticker.C:
			l.cleanupNonces()
			l.auctionService.ProcessAuctions(l)
			l.processPlaystyleDecay()      // New: Decay playstyle tendencies
			l.processRumors()              // New: Check for expired rumors
			l.loanService.ProcessLoans(l)  // New: Check for defaulted loans
			l.processMojoDecay()           // New: Penalize stagnant clubs
			l.processInsuranceRecovery()   // New: Check for expired kidnappings
			l.clubService.ProcessLeaseExpirations(l)    // New: Check for expired card leases
			l.processAllianceExpirations() // PILLAR 1: Clear stale proposals
			l.broadcastBountyBoard()       // PILLAR 3: Criminality Intel
			l.processTreasuryAnalytics()   // PILLAR 2: Economic Intel
			go l.observeGlobalSentiments() // Pillar 3: Aggregate meta trends
		case <-matchmakingTicker.C:
			l.processMatchmaking()
		case <-healthTicker.C:
			l.mutex.RLock()
			isOver := time.Since(l.seasonStart) > 30*24*time.Hour
			l.mutex.RUnlock()
			if isOver {
				go func() {
					l.archiveSeason()
					l.refreshRegionalRoles() // Verify ranks on rollover
				}()
			}
			go l.broadcastHealthReport()
		case <-cacheSaveTicker.C:
			l.CheckCorporateBailouts() // PILLAR 1: Macro-economic organizational check
			go l.oracleService.SavePersistentCardCache(l)
			go l.saveRegisteredTxIDs()
			go l.saveLinkedWallets()
			go l.oracleService.SaveOnboardedWallets(l)
			go l.saveLeaderboard()  // PILLAR 3: Persistent Behavioral Analysis
			go l.saveEconomyState() // PILLAR 2: Persistent Virtual Ledger
		case <-citizenBehaviorTicker.C:
			if l.aiEngine != nil {
				l.aiEngine.UpdateBehavioralLoop(l)
			}

		case <-vaultCheckTicker.C:
			go l.oracleService.CheckVaultBalanceOnChain(l) // Monitor $VBV Reward Pool
			go l.oracleService.CheckNativeVaultBalanceOnChain(l)
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

// loadLeaderboard loads player statistics and playstyle tendencies from a file.
// PILLAR 6: Blockchain Persistence. Reconstructs leaderboard state from blockchain snapshots.
func (l *Lobby) loadLeaderboard() {
	l.mutex.Lock() // Acquire lock for initialization and modification
	defer l.mutex.Unlock()

	l.leaderboard = make(map[string]PlayerStats) // Initialize to empty map

	voiConfig, ok := l.availableNetworks["Voi Mainnet"]
	vaultAddr := l.vaultAddress

	if !ok || vaultAddr == "" {
		log.Println("[CACHE] Voi Mainnet config or Vault address missing. Cannot reconstruct leaderboard state from blockchain.")
		return
	}

	log.Println("[CACHE] Reconstructing leaderboard state from blockchain snapshots...")

	resp, err := l.indexerRequest(voiConfig, fmt.Sprintf("/arc200/transfers?contractId=%s&from=%s&to=%s&note_prefix=%s&limit=100",
		voiConfig.AssetID, vaultAddr, vaultAddr, "VBT_STATE_SNAPSHOT:"))

	if err != nil {
		log.Printf("[CACHE ERROR] Failed to query indexer for leaderboard snapshots: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[CACHE ERROR] Indexer returned non-200 status for leaderboard snapshots: %d %s\n", resp.StatusCode, resp.Status)
		return
	}

	var res struct {
		Transfers []struct {
			TransactionID string `json:"transactionId"`
			Metadata      string `json:"metadata"`
			Timestamp     int64  `json:"timestamp"`
		} `json:"transfers"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		log.Printf("[CACHE ERROR] Failed to decode indexer response for leaderboard snapshots: %v\n", err)
		return
	}

	var latestSnapshotData []byte
	var latestTimestamp int64 = 0

	for _, tx := range res.Transfers {
		if strings.HasPrefix(tx.Metadata, "VBT_STATE_SNAPSHOT:") {
			if tx.Timestamp > latestTimestamp {
				compressedB64 := strings.TrimPrefix(tx.Metadata, "VBT_STATE_SNAPSHOT:")
				decodedBytes, err := base64.StdEncoding.DecodeString(compressedB64)
				if err != nil {
					log.Printf("[CACHE ERROR] Failed to base64 decode leaderboard snapshot from TxID %s: %v\n", tx.TransactionID, err)
					continue
				}

				gzr, err := gzip.NewReader(bytes.NewReader(decodedBytes))
				if err != nil {
					log.Printf("[CACHE ERROR] Failed to create gzip reader for leaderboard snapshot from TxID %s: %v\n", tx.TransactionID, err)
					continue
				}
				defer gzr.Close()

				var decompressedData bytes.Buffer
				if _, err := decompressedData.ReadFrom(gzr); err != nil {
					log.Printf("[CACHE ERROR] Failed to decompress leaderboard snapshot from TxID %s: %v\n", tx.TransactionID, err)
					continue
				}

				latestSnapshotData = decompressedData.Bytes()
				latestTimestamp = tx.Timestamp
			}
		}
	}

	if latestSnapshotData != nil {
		if err := json.Unmarshal(latestSnapshotData, &l.leaderboard); err != nil {
			log.Printf("[CACHE ERROR] Failed to unmarshal latest leaderboard snapshot: %v\n", err)
		} else {
			// PILLAR 3: Identity Persistence.
			// Ensure all loaded player records have initialized maps for new features like AuditedClubs.
			for wallet := range l.leaderboard {
				l.ensurePlayerStatsMapsInitialized(wallet)
				// PILLAR 3: Identity Persistence.
				// Ensure the wallet field is populated for all loaded PlayerStats
				// to enable Governor and Owner-based reputation multipliers.
				s := l.leaderboard[wallet]
				s.Wallet = wallet
				l.leaderboard[wallet] = s
			}
			log.Printf("[CACHE] Reconstructed leaderboard state with %d player records from latest blockchain snapshot (Timestamp: %s).\n", len(l.leaderboard), time.Unix(latestTimestamp, 0).Format(time.RFC3339))
		}
	} else {
		// PILLAR 3: Identity Persistence.
		// Ensure the leaderboard is initialized even if no snapshots exist.
		if l.leaderboard == nil {
			l.leaderboard = make(map[string]PlayerStats)
		}
		log.Println("[CACHE] No VBT_STATE_SNAPSHOT found on-chain. Initializing empty leaderboard state.")
	}
}

// saveLeaderboard saves player statistics and playstyle tendencies to a file.
// PILLAR 6: Blockchain Persistence. This now sends a compressed JSON snapshot as a VBT_STATE_SNAPSHOT blockchain note.
func (l *Lobby) saveLeaderboard() {
	l.mutex.RLock()

	// PILLAR 4: Mutex Contention Minimization.
	// Extract a point-in-time snapshot of the leaderboard state while holding RLock.
	// This allows the expensive JSON marshalling to happen outside the lock.
	leaderboardSnapshot := make(map[string]PlayerStats, len(l.leaderboard))

	for wallet, stats := range l.leaderboard {
		// Create a deep copy of the PlayerStats struct to prevent concurrent access panics
		cloned := stats

		// 1. Deep Copy Nested Maps
		if stats.Inventory != nil {
			cloned.Inventory = make(map[string]int, len(stats.Inventory))
			for k, v := range stats.Inventory { cloned.Inventory[k] = v }
		}
		if stats.Relationships != nil {
			cloned.Relationships = make(map[string]int, len(stats.Relationships))
			for k, v := range stats.Relationships { cloned.Relationships[k] = v }
		}
		if stats.Portfolio != nil {
			cloned.Portfolio = make(map[string]uint64, len(stats.Portfolio)) // PILLAR 2: Integer Supremacy
			for k, v := range stats.Portfolio { cloned.Portfolio[k] = v }
		}
		if stats.JailedCards != nil {
			cloned.JailedCards = make(map[int]string, len(stats.JailedCards))
			for k, v := range stats.JailedCards { cloned.JailedCards[k] = v }
		}
		if stats.KidnappedCards != nil {
			cloned.KidnappedCards = make(map[int]string, len(stats.KidnappedCards))
			for k, v := range stats.KidnappedCards { cloned.KidnappedCards[k] = v }
		}
		if stats.HeldHostageCards != nil {
			cloned.HeldHostageCards = make(map[int]string, len(stats.HeldHostageCards))
			for k, v := range stats.HeldHostageCards { cloned.HeldHostageCards[k] = v }
		}
		if stats.AuditedClubs != nil {
			cloned.AuditedClubs = make(map[string]bool, len(stats.AuditedClubs))
			for k, v := range stats.AuditedClubs { cloned.AuditedClubs[k] = v }
		}
		if stats.CapturedOutlaws != nil {
			cloned.CapturedOutlaws = make(map[string]bool, len(stats.CapturedOutlaws))
			for k, v := range stats.CapturedOutlaws { cloned.CapturedOutlaws[k] = v }
		}
		if stats.PreferredRules != nil {
			cloned.PreferredRules = make(map[string]int, len(stats.PreferredRules))
			for k, v := range stats.PreferredRules { cloned.PreferredRules[k] = v }
		}
		if stats.Moods != nil {
			cloned.Moods = make(map[string]int, len(stats.Moods))
			for k, v := range stats.Moods { cloned.Moods[k] = v }
		}
		if stats.ActiveItemBuffs != nil {
			cloned.ActiveItemBuffs = make(map[string]int, len(stats.ActiveItemBuffs))
			for k, v := range stats.ActiveItemBuffs { cloned.ActiveItemBuffs[k] = v }
		}

		// 2. Playstyle Tendencies Maps
		if stats.Playstyle.PreferredRules != nil {
			cloned.Playstyle.PreferredRules = make(map[string]float64, len(stats.Playstyle.PreferredRules))
			for k, v := range stats.Playstyle.PreferredRules { cloned.Playstyle.PreferredRules[k] = v }
		}
		if stats.Playstyle.PreferredCardMoods != nil {
			cloned.Playstyle.PreferredCardMoods = make(map[string]float64, len(stats.Playstyle.PreferredCardMoods))
			for k, v := range stats.Playstyle.PreferredCardMoods { cloned.Playstyle.PreferredCardMoods[k] = v }
		}
		if stats.Playstyle.PreferredItems != nil {
			cloned.Playstyle.PreferredItems = make(map[string]float64, len(stats.Playstyle.PreferredItems))
			for k, v := range stats.Playstyle.PreferredItems { cloned.Playstyle.PreferredItems[k] = v }
		}

		// 3. Deep Copy Slices
		if stats.Achievements != nil {
			cloned.Achievements = make([]string, len(stats.Achievements))
			copy(cloned.Achievements, stats.Achievements)
		}
		if stats.History != nil {
			cloned.History = make([]MatchHistory, len(stats.History))
			copy(cloned.History, stats.History)
		}
		if stats.MutationHistory != nil {
			cloned.MutationHistory = make([]MutationEvent, len(stats.MutationHistory))
			copy(cloned.MutationHistory, stats.MutationHistory)
		}
		if stats.CapturedOutlaws != nil {
			cloned.CapturedOutlaws = make(map[string]bool, len(stats.CapturedOutlaws))
			for k, v := range stats.CapturedOutlaws { cloned.CapturedOutlaws[k] = v }
		}
		if stats.ActiveItemBuffs != nil {
			cloned.ActiveItemBuffs = make(map[string]int, len(stats.ActiveItemBuffs))
			for k, v := range stats.ActiveItemBuffs { cloned.ActiveItemBuffs[k] = v }
		}
		if stats.AuditedClubs != nil {
			cloned.AuditedClubs = make(map[string]bool, len(stats.AuditedClubs))
			for k, v := range stats.AuditedClubs { cloned.AuditedClubs[k] = v }
		}

		leaderboardSnapshot[wallet] = cloned
	}

	l.mutex.RUnlock()

	// Marshalling Phase (Lock-Free)
	data, err := json.Marshal(leaderboardSnapshot)
	if err != nil {
		log.Printf("[CACHE ERROR] Failed to marshal leaderboard for blockchain snapshot: %v\n", err)
		return
	}
	l.dispatchBlockchainSnapshot("VBT_STATE_SNAPSHOT:", data)
}

// loadEconomyState reconstructs virtual balances (salaries/heists) and active kidnappings from the latest VBT_ECONOMY_SNAPSHOT blockchain note.
// PILLAR 6: Blockchain Persistence. Reconstructs economy state from blockchain snapshots.
func (l *Lobby) loadEconomyState() bool {
	// PILLAR 6: Blockchain Persistence.
	l.mutex.Lock() // Acquire lock for initialization and modification
	defer l.mutex.Unlock()

	l.playerBalances = make(map[string]uint64)
	l.activeKidnappings = make(map[int]KidnapState)

	voiConfig, ok := l.availableNetworks["Voi Mainnet"]
	vaultAddr := l.vaultAddress

	if !ok || vaultAddr == "" {
		log.Println("[CACHE] Voi Mainnet config or Vault address missing. Cannot reconstruct economy state from blockchain.")
		return false
	}

	log.Println("[CACHE] Reconstructing economy state from blockchain snapshots...")

	resp, err := l.indexerRequest(voiConfig, fmt.Sprintf("/arc200/transfers?contractId=%s&from=%s&to=%s&note_prefix=%s&limit=100",
		voiConfig.AssetID, vaultAddr, vaultAddr, "VBT_ECONOMY_SNAPSHOT:"))

	if err != nil {
		log.Printf("[CACHE ERROR] Failed to query indexer for economy snapshots: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[CACHE ERROR] Indexer returned non-200 status for economy snapshots: %d %s\n", resp.StatusCode, resp.Status)
		return false
	}

	var res struct {
		Transfers []struct {
			TransactionID string `json:"transactionId"`
			Metadata      string `json:"metadata"`
			Timestamp     int64  `json:"timestamp"`
		} `json:"transfers"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		log.Printf("[CACHE ERROR] Failed to decode indexer response for economy snapshots: %v\n", err)
		return false
	}

	var latestSnapshotData []byte
	var latestTimestamp int64 = 0

	for _, tx := range res.Transfers {
		if strings.HasPrefix(tx.Metadata, "VBT_ECONOMY_SNAPSHOT:") {
			if tx.Timestamp > latestTimestamp {
				compressedB64 := strings.TrimPrefix(tx.Metadata, "VBT_ECONOMY_SNAPSHOT:")
				decodedBytes, err := base64.StdEncoding.DecodeString(compressedB64)
				if err != nil {
					log.Printf("[CACHE ERROR] Failed to base64 decode economy snapshot from TxID %s: %v\n", tx.TransactionID, err)
					continue
				}

				gzr, err := gzip.NewReader(bytes.NewReader(decodedBytes))
				if err != nil {
					log.Printf("[CACHE ERROR] Failed to create gzip reader for economy snapshot from TxID %s: %v\n", tx.TransactionID, err)
					continue
				}
				defer gzr.Close()

				var decompressedData bytes.Buffer
				if _, err := decompressedData.ReadFrom(gzr); err != nil {
					log.Printf("[CACHE ERROR] Failed to decompress economy snapshot from TxID %s: %v\n", tx.TransactionID, err)
					continue
				}

				latestSnapshotData = decompressedData.Bytes()
				latestTimestamp = tx.Timestamp
			}
		}
	}

	if latestSnapshotData != nil {
		var state struct {
			Balances         map[string]uint64           `json:"balances"`
			Kidnappings      map[int]KidnapState         `json:"active_kidnappings"`
			MatchHistory     map[string]MatchHistory     `json:"match_history"`
			Tournament       TournamentState             `json:"tournament"`
			SeasonNum        int                         `json:"season_num"`
			MarketNodes      map[string]EntityMarketNode `json:"market_nodes"`
			SeasonStart      time.Time                   `json:"season_start"`
			Rewards          map[string]uint64           `json:"initial_rewards"`
			OnboardedWallets map[string]bool             `json:"onboarded_wallets"`
			PaidParticipants []string                    `json:"paid_participants"`
			AuditInflow      uint64                      `json:"audit_inflow"`    // PILLAR 2: Reconstruction Parity
			AuditAllocated   uint64                      `json:"audit_allocated"`
			AuditSiphoned    uint64                      `json:"audit_siphoned"`
			AuditExited       uint64                      `json:"audit_exited"`
		}
		if err := json.Unmarshal(latestSnapshotData, &state); err != nil {
			log.Printf("[CACHE ERROR] Failed to unmarshal latest economy snapshot: %v\n", err)
		} else {
			l.playerBalances = state.Balances
			l.activeKidnappings = state.Kidnappings
			l.tournament = state.Tournament
			l.seasonNumber = state.SeasonNum

			// PILLAR 2: Authoritative Reconciliation.
			// Restore absolute ledger counters to prevent Structural Drift after blockchain recovery.
			if l.tokenSinkRouter != nil && l.tokenSinkRouter.Audit != nil {
				atomic.StoreUint64(&l.tokenSinkRouter.Audit.TotalSystemInputVetted, state.AuditInflow)
				atomic.StoreUint64(&l.tokenSinkRouter.Audit.TotalSystemAllocated, state.AuditAllocated)
				atomic.StoreUint64(&l.tokenSinkRouter.Audit.TotalSystemSiphoned, state.AuditSiphoned)
				atomic.StoreUint64(&l.tokenSinkRouter.Audit.TotalRewardsExited, state.AuditExited)
			}

			// PILLAR 6: Bootstrap Optimization.
			// Restore cached onboarding and registration lists to minimize indexer traffic.
			if state.OnboardedWallets != nil {
				l.onboardedWallets = state.OnboardedWallets
				l.SybilSyncComplete = true
			}
			if state.PaidParticipants != nil {
				l.paidParticipants = state.PaidParticipants
			}

			// PILLAR 4: AMM Reconstruction.
			// Re-link Market Nodes to both Lobby and Router, initializing fresh mutexes.
			l.marketNodes = make(map[string]*EntityMarketNode)
			for id, node := range state.MarketNodes {
				captured := node
				captured.Mu = sync.RWMutex{}
				l.marketNodes[id] = &captured
			}
			if l.tokenSinkRouter != nil { l.tokenSinkRouter.MarketNodes = l.marketNodes }

			l.seasonStart = state.SeasonStart
			l.initialRewards = state.Rewards
			if state.MatchHistory != nil {
				// PILLAR 4: Historical Immersion.
				// Ensure matchHistory is initialized if it was nil in the snapshot.
				// This prevents nil map panics when processing new match results.
				l.matchHistory = make(map[string]MatchHistory)
				l.matchHistory = state.MatchHistory
			}
			log.Printf("[CACHE] Reconstructed economy state: %d active balances, %d hostage records, %d pending rewards (Snapshot TS: %s).\n",
				len(l.playerBalances), len(l.activeKidnappings), len(l.matchHistory), time.Unix(latestTimestamp, 0).Format(time.RFC3339))
			return true
		}
	} else {
		if l.playerBalances == nil {
			l.playerBalances = make(map[string]uint64)
		}
		log.Println("[CACHE] No VBT_ECONOMY_SNAPSHOT found on-chain. Initializing empty economy state.")
	}
	return false
}

// saveEconomyState sends a compressed JSON snapshot of virtual balances and active kidnappings.
func (l *Lobby) saveEconomyState() {
	l.mutex.RLock()

	// PILLAR 4: Mutex Contention Minimization.
	// Extract a point-in-time snapshot of the economic state while holding RLock.
	// This allows the expensive JSON marshalling to happen outside the lock.

	// 1. Deep Copy Maps
	balancesSnapshot := make(map[string]uint64, len(l.playerBalances))
	for k, v := range l.playerBalances {
		balancesSnapshot[k] = v
	}

	kidnappingsSnapshot := make(map[int]KidnapState, len(l.activeKidnappings))
	for k, v := range l.activeKidnappings {
		kidnappingsSnapshot[k] = v
	}

	matchHistorySnapshot := make(map[string]MatchHistory, len(l.matchHistory))
	for k, v := range l.matchHistory {
		matchHistorySnapshot[k] = v
	}

	rewardsSnapshot := make(map[string]uint64, len(l.initialRewards))
	for k, v := range l.initialRewards {
		rewardsSnapshot[k] = v
	}

	marketNodesSnapshot := make(map[string]EntityMarketNode)
	for id, node := range l.marketNodes {
		if node != nil { marketNodesSnapshot[id] = *node }
	}

	// PILLAR 2: Authoritative Counter Persistence.
	var auditInflow, auditAllocated, auditSiphoned, auditExited uint64
	if l.tokenSinkRouter != nil && l.tokenSinkRouter.Audit != nil {
		auditInflow = atomic.LoadUint64(&l.tokenSinkRouter.Audit.TotalSystemInputVetted)
		auditAllocated = atomic.LoadUint64(&l.tokenSinkRouter.Audit.TotalSystemAllocated)
		auditSiphoned = atomic.LoadUint64(&l.tokenSinkRouter.Audit.TotalSystemSiphoned)
		auditExited = atomic.LoadUint64(&l.tokenSinkRouter.Audit.TotalRewardsExited)
	}

	// 2. Capture Structs and Basic Types
	// TournamentState contains slices which require deep copying to prevent races
	tournament := l.tournament
	tournament.Participants = make([]string, len(l.tournament.Participants))
	copy(tournament.Participants, l.tournament.Participants)
	tournament.Matches = make([]TournamentMatch, len(l.tournament.Matches))
	copy(tournament.Matches, l.tournament.Matches)

	// PILLAR 6: Historical State Capture.
	// Snapshot onboarding and participant lists to ensure warm-start performance.
	onboardedSnapshot := make(map[string]bool, len(l.onboardedWallets))
	for k, v := range l.onboardedWallets {
		onboardedSnapshot[k] = v
	}
	participantsSnapshot := make([]string, len(l.paidParticipants))
	copy(participantsSnapshot, l.paidParticipants)

	seasonNum := l.seasonNumber
	seasonStart := l.seasonStart

	l.mutex.RUnlock()

	// 3. Marshalling Phase (Lock-Free)
	state := struct {
		Balances         map[string]uint64           `json:"balances"`
		Kidnappings      map[int]KidnapState         `json:"active_kidnappings"`
		MatchHistory     map[string]MatchHistory     `json:"match_history"`
		Tournament       TournamentState             `json:"tournament"`
		SeasonNum        int                         `json:"season_num"`
		MarketNodes      map[string]EntityMarketNode `json:"market_nodes"`
		SeasonStart      time.Time                   `json:"season_start"`
		Rewards          map[string]uint64           `json:"initial_rewards"`
		OnboardedWallets map[string]bool             `json:"onboarded_wallets"`
		PaidParticipants []string                    `json:"paid_participants"`
		AuditInflow      uint64                      `json:"audit_inflow"`
		AuditAllocated   uint64                      `json:"audit_allocated"`
		AuditSiphoned    uint64                      `json:"audit_siphoned"`
		AuditExited      uint64                      `json:"audit_exited"`
	}{
		Balances:     balancesSnapshot,
		Kidnappings:  kidnappingsSnapshot,
		MatchHistory: matchHistorySnapshot,
		Tournament:   tournament,
		MarketNodes:  marketNodesSnapshot,
		SeasonNum:    seasonNum,
		SeasonStart:  seasonStart,
		Rewards:      rewardsSnapshot,
		OnboardedWallets: onboardedSnapshot,
		PaidParticipants: participantsSnapshot,
		AuditInflow:      auditInflow,
		AuditAllocated:   auditAllocated,
		AuditSiphoned:    auditSiphoned,
		AuditExited:      auditExited,
	}

	data, err := json.Marshal(state)
	if err != nil {
		log.Printf("[CACHE ERROR] Failed to marshal economy snapshot: %v\n", err)
		return
	}

	l.dispatchBlockchainSnapshot("VBT_ECONOMY_SNAPSHOT:", data)
}

// dispatchBlockchainSnapshot handles the authoritative compression and on-chain dispatch of serialized state.
// PILLAR 6: Blockchain Persistence. Performs slow GZIP operations outside the global mutex.
func (l *Lobby) dispatchBlockchainSnapshot(prefix string, data []byte) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write(data); err != nil {
		log.Printf("[CACHE ERROR] Failed to compress state for %s: %v\n", prefix, err)
		return
	}
	gz.Close()

	compressedB64 := base64.StdEncoding.EncodeToString(b.Bytes())
	note := prefix + compressedB64

	// Send the compressed snapshot as an on-chain note in a goroutine
	go func(p, n string) {
		txid, err := l.sendNoteTx(n)
		if err != nil {
			log.Printf("[CACHE ERROR] Failed to send %s snapshot note: %v\n", p, err)
			return
		}
		log.Printf("[CACHE] %s snapshot archived on-chain (TxID: %s).\n", p, txid)
	}(prefix, note)
}

// saveBlockchainStateSnapshotLocked handles the compression and on-chain dispatch of a state map.
func (l *Lobby) saveBlockchainStateSnapshotLocked(prefix string, state interface{}) {
	data, err := json.Marshal(state)
	if err != nil {
		log.Printf("[CACHE ERROR] Failed to marshal state for %s: %v\n", prefix, err)
		return
	}
	l.dispatchBlockchainSnapshot(prefix, data)
}


// loadBlockchainStateSnapshotLocked is a helper to reconstruct state from blockchain snapshots.
func (l *Lobby) loadBlockchainStateSnapshotLocked(prefix string, target interface{}) bool {
	voiConfig, ok := l.availableNetworks["Voi Mainnet"]
	vaultAddr := l.vaultAddress

	if !ok || vaultAddr == "" {
		log.Printf("[CACHE] Voi Mainnet config or Vault address missing. Cannot reconstruct state for %s.\n", prefix)
		return false
	}

	resp, err := l.indexerRequest(voiConfig, fmt.Sprintf("/arc200/transfers?contractId=%s&from=%s&to=%s&note_prefix=%s&limit=100",
		voiConfig.AssetID, vaultAddr, vaultAddr, prefix))

	if err != nil {
		log.Printf("[CACHE ERROR] Failed to query indexer for %s snapshots: %v\n", prefix, err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	var res struct {
		Transfers []struct {
			TransactionID string `json:"transactionId"`
			Metadata      string `json:"metadata"`
			Timestamp     int64  `json:"timestamp"`
		} `json:"transfers"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return false
	}

	var latestSnapshotData []byte
	var latestTimestamp int64 = 0

	for _, tx := range res.Transfers {
		if strings.HasPrefix(tx.Metadata, prefix) && tx.Timestamp > latestTimestamp {
			compressedB64 := strings.TrimPrefix(tx.Metadata, prefix)
			decodedBytes, err := base64.StdEncoding.DecodeString(compressedB64)
			if err != nil {
				continue
			}

			gzr, err := gzip.NewReader(bytes.NewReader(decodedBytes))
			if err != nil {
				continue
			}

			var decompressedData bytes.Buffer
			if _, err := decompressedData.ReadFrom(gzr); err == nil {
				latestSnapshotData = decompressedData.Bytes()
				latestTimestamp = tx.Timestamp
			}
			gzr.Close()
		}
	}

	if latestSnapshotData != nil {
		if err := json.Unmarshal(latestSnapshotData, target); err != nil {
			log.Printf("[CACHE ERROR] Failed to unmarshal latest %s snapshot: %v\n", prefix, err)
			return false
		}
		log.Printf("[CACHE] Reconstructed %s state from latest blockchain snapshot (Timestamp: %s).\n", prefix, time.Unix(latestTimestamp, 0).Format(time.RFC3339))
		return true
	}

	log.Printf("[CACHE] No %s snapshot found on-chain. Initializing empty state.\n", prefix)
	return false
}

func (l *Lobby) handleGameProtocol(env *Envelope, _ []byte) {
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
		go l.narrativeService.GenerateNPCCommentary(l, env.FromID, "LOBBY_ENTRY")

		stats := l.leaderboard[normalizedWallet]
		portfolioPayload, _ := json.Marshal(stats.Portfolio) // Marshal portfolio while lock is held

		// Check admin status and update client while lock is held
		isAdmin := l.isAdminWallet(normalizedWallet)
		if isAdmin {
			if c, ok := l.clients[env.FromID]; ok {
				c.isAdmin = true
			}
		}

		// PILLAR 4: Reconnection Handshake.
		if l.gracePeriodMatrix != nil {
			_, _ = l.gracePeriodMatrix.HandleReconnectionHandshake(normalizedWallet)
		}

		// PILLAR 3: Continuous Verification.
		// Start tracking the session for the watchdog auditor.
		l.TrackSession(normalizedWallet)
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

		// PILLAR 5: Identity Hardening.
		l.ensurePlayerStatsMapsInitialized(wallet)

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
			l.matches[env.FromID] = &MatchState{
				P1ID:        env.FromID,
				P1Deck:      data.Deck,
				StartTime:   time.Now(),
				MatchRating: data.DeckRating,
			} // Initialize match state
			l.updatePlayerPlaystyleTendenciesLocked(wallet, false, [2]int{}, data.Deck, false, false) // Update playstyle based on deck

			go l.narrativeService.GenerateNPCCommentary(l, env.FromID, "MATCH_START")
		}
		l.mutex.Unlock()
		l.sendToClient(env.FromID, Envelope{Type: "matchmaking_status", Payload: json.RawMessage(`{"status":"queued"}`)})
	case "nonce_request":
		nonce := generateNonce()
		l.mutex.Lock()
		l.nonces[env.FromID] = NonceData{Value: nonce, CreatedAt: time.Now()}
		l.mutex.Unlock()
		l.sendToClient(env.FromID, Envelope{Type: "nonce_response", FromID: "SERVER", Payload: json.RawMessage(fmt.Sprintf(`{"nonce":"%s"}`, nonce))})
	case "sync_request":
		var data struct { LastSeq uint64 `json:"last_sequence_id"` }
		json.Unmarshal(env.Payload, &data)
		
		l.mutex.Lock()
		match, ok := l.matches[env.FromID]
		if ok {
			// PILLAR 4: Replay Resilience.
			// Ensure the handshaker is resolved or initialized under the master lock
			// to prevent map access panics and guarantee a response.
			sh, exists := l.matchHandshakers[match.P1ID]
			if !exists {
				sh = l.NewSyncHandshaker()
				l.matchHandshakers[match.P1ID] = sh
			}
			
			delta := sh.CatchUpPlayer(data.LastSeq)
			var catchup []json.RawMessage
			for _, f := range delta { catchup = append(catchup, f.MoveIntent) }
			
			resp, _ := json.Marshal(map[string]interface{}{"frames": catchup})
			l.sendToClientLocked(env.FromID, Envelope{Type: "sync_response", FromID: "SERVER", Payload: resp})
		}
		l.mutex.Unlock()
	case "redemption_gateway":
		var req RedemptionRequest
		if err := json.Unmarshal(env.Payload, &req); err != nil {
			return
		}
		l.mutex.Lock()
		err := l.executeRedemptionLocked(req)
		if err != nil {
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"❌ Redemption Failed: %s"}`, err.Error()))})
		} else {
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"✅ <b>REDEMPTION SUCCESS:</b> Your console DLC has been unlocked."}`)})
		}
		l.mutex.Unlock()
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
			// EVM signature verification (personal_sign).
			// The message signed by the wallet is expected to be the raw nonce string,
			// which the wallet then prefixes with "\x19Ethereum Signed Message:\n<length>".
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

		// PILLAR 2: Phase 4 Conversion Funnel.
		// Trigger the migration of non-crypto Arena Vouchers accumulated on console hardware.
		l.onboardingService.HandleVoucherConversion(l, primaryWallet)

		l.sendToClient(env.FromID, Envelope{Type: "link_wallet_response", Payload: json.RawMessage(fmt.Sprintf(`{"status":"success","message":"Wallet linked successfully","address":"%s"}`, linkedAddr))})
		log.Printf("[LINK] Successfully linked %s (%s) to primary AVM wallet %s\n", linkedAddr, data.LinkedChain, primaryWallet)
	case "move":
		l.mutex.RLock()
		match, ok := l.matches[env.FromID]
		l.mutex.RUnlock()
		if !ok {
			return
		}
		pIdx := 0
		if env.FromID == match.P2ID {
			pIdx = 1
		}

		// SECURITY AUDIT: Verify that the sender is actually a player in this match.
		// Spectators should never be able to inject moves or trigger AI delays for participants.
		if env.FromID != match.P1ID && env.FromID != match.P2ID {
			log.Printf("[SECURITY] Unauthorized move attempt from spectator: %s (Player Index: %d)\n", env.FromID, pIdx)
			return
		}

		var move MoveData
		if err := json.Unmarshal(env.Payload, &move); err != nil {
			return
		}
		l.mutex.Lock()
		if move.GridIndex >= 0 && move.GridIndex < 9 {
			// SECURE SYNC: Fetch card from server authoritative inventory to prevent power spoofing
			card, exists := l.inventory[move.CardID]

			// PILLAR 5: Hardware Inversion.
			// Map active club hardware traps to the card instance on placement.
			if match.TerritoryID != "" {
				owningClub := l.getClubByTerritoryID(match.TerritoryID)
				if owningClub != nil && owningClub.ActiveBuffs != nil {
					for buffID := range owningClub.ActiveBuffs {
						if strings.HasPrefix(buffID, "TRAP_") {
							if expiry, ok := owningClub.BuffExpirations[buffID]; ok && time.Now().Before(expiry) {
								card.EquippedItems = append(card.EquippedItems, buffID)
							}
						}
					}
				}
			}
			if !exists {
				// Hardening: If card isn't in server cache, use a baseline weak card to prevent spoofing
				card = ServerCard{ID: move.CardID, Power: [4]int{5, 5, 5, 5}}
				// PILLAR 3: Diagnostic Read. Use the field to satisfy static analysis.
				log.Printf("[SECURITY] Unauthorized CardID %d in move from %s. Using baseline power.\n", card.ID, env.FromID)
			}

			// Authoritative Assignment: Assign a copy of the server card to the board and set ownership.
			// This ensures 'card.ID' and other fields are correctly registered for win verification.
			card.Owner = pIdx
			match.Board[move.GridIndex] = &card

			// serverCheckCaptures now returns captured cards and the deterministic state hash.
			// PILLAR 4: Deterministic Sync.
			_, flips, stateHash := l.serverCheckCaptures(match, move.GridIndex, pIdx)
			match.CapturedCards = append(match.CapturedCards, flips...)

			// PILLAR 5: Reactive Atmosphere.
			// If the move resulted in a rule-based trigger (Same/Plus) or a combo chain,
			// send a specific "turn_change" notification to trigger high-intensity UI feedback.
			hasCombo := false
			for _, f := range flips {
				if f.CaptureType != "BASIC" {
					hasCombo = true
					break
				}
			}
			if hasCombo {
				// PILLAR 5: Authoritative Feedback.
				// Snapshot IDs and delay slightly to ensure ordering after the main 'move' broadcast.
				p1, p2 := match.P1ID, match.P2ID
				specs := make([]string, len(match.Spectators))
				copy(specs, match.Spectators)
				go func() {
					time.Sleep(150 * time.Millisecond)
					l.sendToClient(p1, Envelope{Type: "turn_change", FromID: "SERVER", Payload: json.RawMessage(`{"combo":true}`)})
					l.sendToClient(p2, Envelope{Type: "turn_change", FromID: "SERVER", Payload: json.RawMessage(`{"combo":true}`)})
					for _, sID := range specs {
						l.sendToClient(sID, Envelope{Type: "turn_change", FromID: "SERVER", Payload: json.RawMessage(`{"combo":true}`)})
					}
				}()
			}

			wallet := l.wallets[env.FromID]

			// PILLAR 4: Sequence Hardening.
			// Generate FrameDelta and increment SequenceID atomically with the state transition.
			// This ensures that desync recovery catch-up always sees a continuous chronological chain.
			// PILLAR 4: Sequence Reset.
			// Explicitly initialize or reset the handshaker for the new match.
			// This ensures that SequenceID starts at 0 for every fresh engagement,
			// preventing catch-up loops from previous sessions.
			sh, exists := l.matchHandshakers[match.P1ID] // Use P1ID as the canonical match ID for handshaker
			if !exists || sh.CurrentSequence == 0 { // Re-initialize if not exists or if it's a new match (sequence 0)
				// If it's a new match, ensure the handshaker is clean
				delete(l.matchHandshakers, match.P1ID) // Prune old handshaker if it exists
				sh = l.NewSyncHandshaker()
				l.matchHandshakers[match.P1ID] = sh
			}

			sh.Mu.Lock()
			sh.CurrentSequence++

			// PILLAR 4: Session Parity (Multi-Participant Sync).
			// Synchronize the 'LastActiveFrame' for all participants and spectators
			// to ensure they receive a continuous catch-up stream upon reconnection.
			if l.gracePeriodMatrix != nil {
				l.gracePeriodMatrix.Mu.Lock()
				
				// 1. Update Sender
				if sess, ok := l.gracePeriodMatrix.ActiveSessions[wallet]; ok {
					sess.LastActiveFrame = sh.CurrentSequence
				}

				// 2. Update Opponent
				oppID := match.P1ID
				if env.FromID == match.P1ID { oppID = match.P2ID }
				if oppW, ok := l.wallets[oppID]; ok {
					if sess, ok := l.gracePeriodMatrix.ActiveSessions[strings.ToLower(oppW)]; ok {
						sess.LastActiveFrame = sh.CurrentSequence
					}
				}

				// 3. Update Spectators
				for _, sID := range match.Spectators {
					if sW, ok := l.wallets[sID]; ok {
						if sess, ok := l.gracePeriodMatrix.ActiveSessions[strings.ToLower(sW)]; ok {
							sess.LastActiveFrame = sh.CurrentSequence
						}
					}
				}
				l.gracePeriodMatrix.Mu.Unlock()
			}

			// PILLAR 4: Authoritative Snapshot.
			// Move PlayerIndex assignment up to ensure it's captured in the marshaled broadcast.
			move.PlayerIndex = pIdx

			// Construct the AuthoritativeFrame for broadcast and replay storage.
			authoritativeFrame := AuthoritativeFrame{
				SequenceID: sh.CurrentSequence,
				MoveIntent: move, // Use the MoveData struct directly
				StateHash:  stateHash,
			}
			// Marshal the entire AuthoritativeFrame to be sent as env.Payload
			env.Payload, _ = json.Marshal(authoritativeFrame)

			// Store the same authoritative frame in HistoricalFrames for recovery catch-up.
			sh.HistoricalFrames[sh.CurrentSequence] = FrameDelta{
				SequenceID: sh.CurrentSequence,
				MoveIntent: env.Payload,
				StateHash:  stateHash,
			}
			sh.LastVerifiedHash = stateHash
			sh.Mu.Unlock()

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

		// PILLAR 5: Identity Hardening.
		l.ensurePlayerStatsMapsInitialized(wallet)

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

		// PILLAR 5: Separation of Concerns.
		// Delegate item effect application to the specialized item_service.go.
		// This centralizes item logic and prevents duplication.
		notificationText, err := l.applyItemEffect(env, data, wallet, &playerStats, item)
		if err != nil {
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"❌ Item Use Failed: %s"}`, err.Error()))})
			return
		}
		// Update playerStats in leaderboard after applyItemEffect might have modified it
		l.leaderboard[wallet] = playerStats

		l.logAdminAudit("ITEM_USED", wallet, fmt.Sprintf("Item: %s, TargetCard: %d, TargetGrid: %d", data.ItemID, data.TargetCardID, data.TargetGridIndex))
		l.sendToClient(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"%s"}`, notificationText))})

		// Trigger global sync to update UI (inventory, card stats, match state)
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	case "purchase_item":
		l.clubService.HandlePurchaseItem(l, env)
	case "restock_inventory":
		l.clubService.HandleRestockInventory(l, env)
	case "alliance_invite":
		l.clubService.HandleAllianceInvite(l, env)
	case "alliance_accept":
		l.clubService.HandleAllianceAccept(l, env)
	case "alliance_dissolve":
		l.clubService.HandleAllianceDissolve(l, env)
	// DELEGATED TO market_service.go
	case "trade_shares":
		l.handleTradeShares(env)
	// DELEGATED TO club_service.go
	case "heist":
		l.clubService.HandleHeist(l, env)
	case "sabotage":
		l.clubService.HandleSabotage(l, env)
	// DELEGATED TO club_service.go
	case "create_club":
		l.clubService.HandleCreateClub(l, env)
	// DELEGATED TO club_service.go
	case "join_club":
		l.clubService.HandleJoinClub(l, env)
	// DELEGATED TO employment_service.go
	case "hire_player":
		l.handleHirePlayer(env)
	// DELEGATED TO club_service.go
	case "purchase_territory":
		l.clubService.HandlePurchaseTerritory(l, env)
	case "kidnap_request":
		l.handleKidnapRequest(env)
	case "vector_realignment": // Pillar 6: Mutation Foundry
		l.clubService.HandleMutationVectorRealignment(l, env)
	case "mood_recalibration": // Pillar 6: Mutation Foundry
		l.clubService.HandleMutationMoodRecalibration(l, env)
	case "loyalty_synthesis": // Pillar 6: Mutation Foundry
		l.clubService.HandleMutationLoyaltySynthesis(l, env)
	case "sell_to_black_market": // PILLAR 3: Silkroad Expansion
		l.blackMarketService.HandleSellToBlackMarket(l, env)
	case "abort_underworld_contract": // PILLAR 3: Underworld Contracts
		l.blackMarketService.HandleAbortUnderworldContract(l, env)
	case "launder_capital": // PILLAR 3: Career Path Actions
		l.HandleLaunderCapital(env)
	case "purify_card": // PILLAR 7: Underworld Recovery
		l.clubService.HandlePurifyCard(l, env)
	case "justice_flag_player": // PILLAR 3: Justice Terminal
		l.HandleJusticeFlagPlayer(env)
	case "abort_justice_mission": // PILLAR 3: Justice Layer
		l.HandleAbortJusticeMission(env)
	case "accept_justice_mission": // PILLAR 3: Justice Layer
		l.HandleAcceptJusticeMission(env)
	case "accept_underworld_contract": // PILLAR 3: Underworld Contracts
		l.blackMarketService.HandleAcceptUnderworldContract(l, env)
	case "aos_raid": // PILLAR 3: Justice Layer
		l.HandleAOSRaid(env)
	case "purchase_raid_insurance": // PILLAR 1 & 3
		l.HandlePurchaseRaidInsurance(env)
	case "purchase_bounty_license": // PILLAR 3: Justice Layer
		l.HandlePurchaseBountyLicense(env)
	case "purchase_bounty_bond": // PILLAR 1: Justice Layer
		l.HandlePurchaseBountyHunterBond(env)
	case "refund_bounty_bond": // PILLAR 1: Justice Layer
		l.HandleRefundBountyHunterBond(env)
	case "claim_dividends": // PILLAR 1: Organizational Yield
		l.HandleClaimDividends(env)
	case "freeze_dividends": // PILLAR 3: Justice Path
		l.HandleJusticeFreezeDividends(env)
	case "harvest_all_dividends": // PILLAR 1: Organizational Yield
		l.HandleHarvestAllDividends(env)
	case "regional_sabotage": // New: Regional Warfare Protocol
		l.clubService.HandleRegionalSabotage(l, env)
	case "refresh_identity": // PILLAR 3: Identity Management
		l.onboardingService.HandleIdentityRefresh(l, env)
	case "initiate_recovery": // PILLAR 7: Underworld Recovery
		l.HandleInitiateRecovery(env)
	case "list_recovery_bounty": // PILLAR 7: Recovery Bounties
		l.blackMarketService.HandleListRecoveryBounty(l, env)
	case "pay_ransom":
		l.handlePayRansom(env)
	case "release_hostage":
		l.handleReleaseHostage(env)
	case "spread_rumor":
		l.handleSpreadRumor(env)
	case "create_lease":
		l.clubService.HandleCreateLease(l, env)
	case "take_lease":
		l.clubService.HandleTakeLease(l, env)
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
		stats, exists := l.leaderboard[wallet]
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
			stats.Reputation = l.CalculateReputation(stats)
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
				stats.Reputation = l.CalculateReputation(stats)
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

// HandleInitiateRecovery sets the state for a 3-win asset retrieval challenge.
func (l *Lobby) HandleInitiateRecovery(env *Envelope) {
	var data struct {
		CardID int `json:"card_id"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return
	}
	l.mutex.Lock()
	defer l.mutex.Unlock()
	wallet, ok := l.wallets[env.FromID]
	if !ok { return }
	l.ensurePlayerStatsMapsInitialized(wallet)
	stats := l.leaderboard[wallet]

	if stats.RecoveryChallengeCardID != 0 {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"⚠️ <b>ACTIVE CHALLENGE:</b> Complete your current retrieval first."}`)})
		return
	}

	// PILLAR 7: Challenge Escrow.
	// If the player holds the 'Fallen' version of the asset, remove it from inventory.
	// It will be restored upon 3 successful wins.
	cardKey := fmt.Sprintf("CARD-%d", data.CardID)
	if qty, has := stats.Inventory[cardKey]; has && qty > 0 {
		stats.Inventory[cardKey]--
		if stats.Inventory[cardKey] <= 0 {
			delete(stats.Inventory, cardKey)
		}
		l.logAdminAuditLocked("RECOVERY_ESCROW", wallet, fmt.Sprintf("Card: %d", data.CardID))
	}

	stats.RecoveryChallengeCardID = data.CardID
	stats.RecoveryChallengeWins = 0
	l.leaderboard[wallet] = stats

	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"🏴‍☠️ <b>SILKROAD:</b> 3-win challenge initiated for BABE #%d. Maintain the streak to liberate the asset."}`, data.CardID))})
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// HandleAOSRaid is declared in handlers_criminality.go to avoid duplicate method definitions.

// HandlePurchaseBountyLicense processes a $VBV payment for a law-enforcement license.
func (l *Lobby) HandlePurchaseBountyLicense(env *Envelope) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	wallet, ok := l.wallets[env.FromID]
	if !ok { return }
	stats := l.leaderboard[wallet]

	const licenseCost = 50 * 1000000
	if l.playerBalances[wallet] < licenseCost {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Access Denied: Insufficient rewards for license (50 $VBV required)."}`)})
		return
	}

	l.playerBalances[wallet] -= licenseCost
	l.faucetBalanceMicro += licenseCost
	l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0

	// Set or extend license for 7 days
	baseTime := time.Now()
	if stats.BountyHunterLicenseExpiresAt.After(baseTime) {
		baseTime = stats.BountyHunterLicenseExpiresAt
	}
	stats.BountyHunterLicenseExpiresAt = baseTime.Add(168 * time.Hour)
	l.leaderboard[wallet] = stats

	l.logAdminAuditLocked("LICENSE_PURCHASED", wallet, "Bounty Hunter License (7 Days)")
	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"⚖️ <b>LICENSE ACTIVE:</b> Status maintained for 7 days. Enforcer Dashboard updated."}`)})
	
	l.applyDynamicScalingLocked()
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
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

	// PILLAR 4: Handshaker Memory Hardening.
	// Aggressively prune handshakers for IDs that are no longer associated with active matches.
	for id := range l.matchHandshakers {
		if _, ok := l.matches[id]; !ok {
			delete(l.matchHandshakers, id)
		}
	}
}

func (l *Lobby) handleUnregister(client *Client) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	wallet, ok := l.wallets[client.id]
	if ok {
		// PILLAR 4: Connection Quarantine Hardening.
		// Only combatants (P1/P2) qualify for the 60s recovery window.
		match, inMatch := l.matches[client.id]
		isCombatant := inMatch && (client.id == match.P1ID || client.id == match.P2ID)

		if isCombatant && l.gracePeriodMatrix != nil {
			// Check if the match is actually ongoing before triggering quarantine
			if !match.IsFinished {
				l.mutex.Unlock()
				l.gracePeriodMatrix.HandleConnectionDrop(wallet)
				l.mutex.Lock()

				// Remove only the connection object; wallet/match state persists in matrix
				delete(l.clients, client.id)
				return
			}
		}

		// PILLAR 4: Memory Management. 
		// Prune idlers from the grace matrix to prevent map leaks.
		l.UntrackSession(wallet)
		if l.gracePeriodMatrix != nil {
			l.gracePeriodMatrix.Mu.Lock()
			delete(l.gracePeriodMatrix.ActiveSessions, wallet)
			l.gracePeriodMatrix.Mu.Unlock()
		}
	}

	if match, ok := l.matches[client.id]; ok {
		// Immediate cleanup for P1/P2 ONLY if grace matrix is NOT active
		if (client.id == match.P1ID || client.id == match.P2ID) && l.gracePeriodMatrix == nil {
			opponentID := match.P1ID
			if client.id == match.P1ID { opponentID = match.P2ID }
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
					l.incrementDNF(wallet, tourneyRound, oppWallet, match.TournamentMatchID)
				}

				// award advancement if in tournament
				if match.TournamentMatchID != "" {
					if oppWallet, ok := l.wallets[opponentID]; ok {
						log.Printf("[TOURNAMENT] Awarding win to %s due to opponent DNF.\n", oppWallet)
						l.tournamentService.ProcessTournamentResult(l, match.TournamentMatchID, oppWallet)
					}
				}
				delete(l.matches, opponentID)
			}
			delete(l.matches, client.id)
		} else {
			// Spectator removal
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

// incrementDNF penalizes a leaver and records the event on-chain for reconstruction.
func (l *Lobby) incrementDNF(wallet string, round int, opponent string, tid string) {
	l.ensurePlayerStatsMapsInitialized(wallet)
	stats := l.leaderboard[wallet]
	stats.DNFs++
	stats.DisconnectStreak++

	// Infamy Scaling: DNFs increase Wanted Level by 2 (scaled by tournament round)
	infamyGain := 2
	if round > 0 {
		infamyGain = round * 2
	}
	stats.WantedLevel += infamyGain

	// Recalculate reputation to reflect the social penalty of abandoning a match
	stats.Reputation = l.CalculateReputation(stats)

	// PILLAR 7: Underworld Streak Reset (DNF Case).
	// Abandoning a match terminates any active recovery challenge.
	stats.RecoveryChallengeWins = 0

	l.leaderboard[wallet] = stats

	log.Printf("[BATTLE] Player %s penalized for DNF (Round: %d, Wanted: +%d). Standing: %d\n", wallet, round, infamyGain, stats.Reputation)

	// PILLAR 4: Historical Persistence.
	go l.recordDNFOnChain(wallet, opponent, tid)
}

func (l *Lobby) handleBroadcast(message []byte) {
	var env Envelope
	if err := json.Unmarshal(message, &env); err != nil {
		return
	}
	l.handleGameProtocol(&env, message) // Process logic before routing

	// PILLAR 4: Frame Integrity.
	// Re-marshal the envelope to include any injected metadata (like SequenceID) before transmission.
	message, _ = json.Marshal(env)

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

	// PILLAR 4: Spectator Portal Hardening.
	// If the client is already watching a match, remove them from that match's
	// spectator list before transitioning to the new one.
	if existing, ok := l.matches[env.FromID]; ok {
		if env.FromID != existing.P1ID && env.FromID != existing.P2ID {
			var remaining []string
			for _, sID := range existing.Spectators {
				if sID != env.FromID {
					remaining = append(remaining, sID)
				}
			}
			existing.Spectators = remaining
		}
	}

	// SECURITY: Prevent active players from abandoning their match to spectate.
	// This ensures that participants in tourney matches cannot trigger a DNF by switching to a stream.
	if existing, ok := l.matches[env.FromID]; ok {
		if env.FromID == existing.P1ID || env.FromID == existing.P2ID {
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Error: Cannot spectate while your own match is active."}`)})
			return
		}
	}

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

	// PILLAR 4: Replay Resilience. 
	// Proactively ensure a handshaker exists for the match to prevent 
	// sync_request failures if the viewer joins before the first move.
	if _, exists := l.matchHandshakers[match.P1ID]; !exists {
		l.matchHandshakers[match.P1ID] = l.NewSyncHandshaker()
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

// countUniqueMatchesLocked counts actual active duels, excluding spectator sessions.
// Assumptions: The lobby mutex is held by the caller.
func (l *Lobby) countUniqueMatchesLocked() int {
	seen := make(map[*MatchState]bool)
	count := 0
	for _, m := range l.matches {
		// PILLAR 4: Load Parity. Exclude queue-only (unpaired) matches and spectators.
		if !seen[m] && !m.IsFinished && m.P2ID != "" {
			seen[m] = true
			count++
		}
	}
	return count
}

// LobbyPlayerInfo represents a snapshot of a player's status for the lobby broadcast.
// PILLAR 4: Performance Hardening. Moved to package scope to resolve local unusedwrite diagnostics.
type LobbyPlayerInfo struct {
	ID                       string              `json:"id"`
	Wallet                   string              `json:"wallet"` // PILLAR 3: Identity resolution
	IsAdmin                  bool                `json:"is_admin"`
	AvatarURL                string              `json:"avatar_url"`
	Gloat                    string              `json:"gloat"`
	AvatarNotice             string              `json:"avatar_notice"`
	BanExpires               time.Time           `json:"ban_expires"`
	HasMardonBadge           bool                `json:"has_mardon_badge"`
	Wins                     int                 `json:"wins"`
	Reputation               int                 `json:"reputation"`
	BestRating               string              `json:"best_rating"`
	AuctionsWon              int                 `json:"auctions_won"`
	Salary                   uint64              `json:"salary"`
	MarketTokens             uint64              `json:"market_tokens"`
	VirtualBalance           uint64              `json:"virtual_balance"`
	WantedLevel              int                 `json:"wanted_level"`
	Cunning                  int                 `json:"cunning"`
	Mojo                     int                 `json:"mojo"`
	Nurturing                int                 `json:"nurturing"`
	Inventory                map[string]int      `json:"inventory"`
	JailedCards              map[int]string      `json:"jailed_cards"`
	SocialRank               string              `json:"social_rank"`
	EquippedFaceplate        string              `json:"equipped_faceplate"`
	KidnappedCards           map[int]string      `json:"kidnapped_cards"`
	MatchHistory             []MatchHistory      `json:"match_history"`
	HeldHostageCards         map[int]string      `json:"held_hostage_cards"`
	HeistAlarmsJammerCount   int                 `json:"heist_alarms_jammer_count"`
	Achievements             []string            `json:"achievements"`
	CapturedOutlaws          map[string]bool     `json:"captured_outlaws"` // PILLAR 3: Unique capture progress
	Playstyle                PlaystyleTendencies `json:"playstyle"`
	GhostProtocolExpiresAt   time.Time           `json:"ghost_protocol_expires_at"`
	DistrictScannerExpiresAt time.Time           `json:"district_scanner_expires_at"`
	DisruptorCooldownAt      time.Time           `json:"disruptor_cooldown_at"` // PILLAR 3: Tactical HUD sync
	RumorCount               int                 `json:"rumor_count"`
	JobRole                  string              `json:"job_role"`
	EmployerID               string              `json:"employer_id"`
	LastSeenDistrict         string              `json:"last_seen_district"` // PILLAR 3: Tactical Tracking
	TotalDonated             uint64              `json:"total_donated"`      // PILLAR 1: Philanthropy
	IsMojoStabilizerActive   bool                `json:"is_mojo_stabilizer_active"` // PILLAR 1: UI Sync
	MojoDecayRate            float64             `json:"mojo_decay_rate"`           // PILLAR 1: UI Sync
}

func (l *Lobby) isClubRegionalLocked(club *Club) bool {
	if l == nil || club == nil || l.clubService == nil {
		return false
	}
	return l.clubService.IsClubRegionalLocked(l, club)
}

func (l *Lobby) unlockAchievementLocked(wallet, achievementID string) {
	if l == nil || l.achievementService == nil {
		return
	}
	targetWallet := strings.ToLower(wallet)
	l.achievementService.UnlockAchievementLocked(l, targetWallet, achievementID)
}

func (l *Lobby) processTournamentResult(tournamentMatchID string, wallet string) {
	if l == nil || l.tournamentService == nil {
		return
	}
	l.tournamentService.ProcessTournamentResult(l, tournamentMatchID, wallet)
}

func (l *Lobby) getLobbyUpdateMsgLocked() []byte {
	var players []LobbyPlayerInfo
	for _, client := range l.clients {
		hasMardon := false
		var banExpires time.Time
		var salary, marketTokens uint64
		wins, reputation, wanted, cunning, nurturing, mojo, auctionsWon, vBal, arenaVouchers := 0, 0, 0, 0, 0, 0, 0, uint64(0), uint64(0)
		var totalDonated uint64
		var inventory map[string]int
		var jailedCards, kidnappedCards, heldHostageCards map[int]string
		var matches []MatchHistory
		var districtScannerExpiresAt time.Time
		var disruptorCooldownAt time.Time
		var lastSeenDistrict string
		var bestRating string
		var heistAlarmsJammerCount int
		var equippedFaceplate string
		var socialRank string
		var ghostProtocolExpiresAt time.Time
		var capturedOutlaws map[string]bool
		var achievements []string
		var jobRole string
		var employerID string
		var rumorCount int
		var playstyle PlaystyleTendencies
		var walletAddr string
		if wallet, ok := l.wallets[client.id]; ok {
			if stats, exists := l.leaderboard[wallet]; exists {
				banExpires = stats.BanExpires
				wins = stats.Wins
				reputation = stats.Reputation
				bestRating = stats.BestRating
				vBal = l.playerBalances[wallet]
				salary = stats.Salary
				marketTokens = stats.MarketTokens
				arenaVouchers = stats.ArenaVouchers
				auctionsWon = stats.AuctionsWon
				// UI Sync: Use Effective Mojo (including faceplate) for Career Path display
				mojo = l.playerService.GetEffectiveMojo(stats)
				wanted = stats.WantedLevel
				// Alignment: Broadcast the Effective Cunning (including faceplate/penalty)
				// to ensure the UI heist heuristic matches the server calculation.
				cunning = l.playerService.GetEffectiveCunning(stats)
				inventory = stats.Inventory
				nurturing = stats.Nurturing
				jailedCards = stats.JailedCards
				kidnappedCards = stats.KidnappedCards
				heldHostageCards = stats.HeldHostageCards
				playstyle = stats.Playstyle
				equippedFaceplate = stats.EquippedFaceplate
				socialRank = stats.SocialRank
				jobRole = stats.JobRole
				districtScannerExpiresAt = stats.DistrictScannerExpiresAt
				disruptorCooldownAt = stats.DisruptorCooldownAt
				heistAlarmsJammerCount = stats.HeistAlarmsJammerCount
				ghostProtocolExpiresAt = stats.GhostProtocolExpiresAt
				employerID = stats.EmployerClubID
				totalDonated = stats.TotalDonated
				capturedOutlaws = stats.CapturedOutlaws
				// PILLAR 1: Mojo Decay Status Sync.
				isMojoStabilizerActive, mojoDecayRate := l.calculateMojoDecayRateLocked(stats.EmployerClubID)
				achievements = stats.Achievements
				// PILLAR 4: Historical Immersion. Send the last 5 matches for display.
				walletAddr = wallet
				lastSeenDistrict = l.lastSeenDistricts[wallet]
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
		players = append(players, LobbyPlayerInfo{
			ID: client.id, Wallet: walletAddr, IsAdmin: client.isAdmin, AvatarURL: client.avatarURL,
			Gloat: client.gloat, AvatarNotice: client.avatarBanNotice,
			BanExpires: banExpires, HasMardonBadge: hasMardon, Wins: wins, Reputation: reputation,
			BestRating: bestRating,
			AuctionsWon: auctionsWon, VirtualBalance: vBal, Salary: salary, MarketTokens: marketTokens,
			WantedLevel: wanted, Cunning: cunning, Nurturing: nurturing, Mojo: mojo,
			MatchHistory:     matches,
			Inventory:        inventory,
			JailedCards:      jailedCards,
			KidnappedCards:   kidnappedCards,
			HeldHostageCards: heldHostageCards,
			SocialRank:       socialRank, EquippedFaceplate: equippedFaceplate,
			Achievements: achievements, RumorCount: rumorCount,
			CapturedOutlaws: capturedOutlaws,
			GhostProtocolExpiresAt:   ghostProtocolExpiresAt,
			DistrictScannerExpiresAt: districtScannerExpiresAt,
			DisruptorCooldownAt:      disruptorCooldownAt,
			HeistAlarmsJammerCount:   heistAlarmsJammerCount,
			Playstyle:                playstyle,
			Faction:                  l.playerService.GetHegemonyPath(jobRole),
			JobRole:                  jobRole, EmployerID: employerID,
			LastSeenDistrict: lastSeenDistrict,
			TotalDonated:     totalDonated,
			IsMojoStabilizerActive:   isMojoStabilizerActive,
			MojoDecayRate:            mojoDecayRate,
			ArenaVouchers:            arenaVouchers,
		})
	}

	// PILLAR 2: Ledger Integrity (Verification Plan Step 1).
	// Calculate Total Virtual Liability for real-time solvency monitoring.
	var totalLiabilities uint64
	for _, bal := range l.playerBalances {
		totalLiabilities += bal
	}

	// PILLAR 2: Account for non-crypto Arena Vouchers in total systemic liability.
	for _, stats := range l.leaderboard {
		totalLiabilities += stats.ArenaVouchers
		for _, bounty := range stats.RecoveryBounties {
			totalLiabilities += bounty
		}
		totalLiabilities += stats.BountyHunterBondMicro
	}
	totalLiabilities += l.pendingTournamentPayoutsMicro // PILLAR 2: Integer Supremacy

	// PILLAR 2: Authoritative Accounting.
	// Include Club Treasuries and Regional Dividends in the systemic liability total.
	// This ensures the solvency dashboard accurately reflects all virtual commitments.
	if l.tokenSinkRouter != nil {
		l.tokenSinkRouter.Mu.RLock()
		for _, clubNode := range l.tokenSinkRouter.ActiveClubs {
			if clubNode != nil {
				totalLiabilities += clubNode.TreasuryBalance
			}
		}
		for _, districtNode := range l.tokenSinkRouter.RegionalDistricts {
			if districtNode != nil {
				totalLiabilities += districtNode.DistrictDividendPool
			}
		}
		l.tokenSinkRouter.Mu.RUnlock()
	}

	// PILLAR 4: UI Stability. Sort players by ID to prevent random shuffling in the lobby list.
	sort.Slice(players, func(i, j int) bool {
		return players[i].ID < players[j].ID
	})

	update := struct {
		Players               []LobbyPlayerInfo        `json:"players"`
		MaintenanceActive     bool                     `json:"maintenance_active"`
		MaintenanceTime       time.Time                `json:"maintenance_time"`
		MaintenancePriority   string                   `json:"maintenance_priority"`
		FaucetBalance         float64                  `json:"faucet_balance"`
		FaucetBalanceMicro    uint64                   `json:"faucet_balance_micro"`
		TotalVirtualLiability uint64                   `json:"total_virtual_liability"`
		Clubs                 map[string]*Club         `json:"clubs"`
		RewardStack           map[string]uint64        `json:"reward_stack"`
		ActiveMatchCount      int                      `json:"active_match_count"` // Fixed: Uses unique match counting
		Tournament            TournamentState          `json:"tournament"`
		AvailableNetworks     map[string]NetworkConfig `json:"available_networks"`
		Rumors                map[string]*Rumor        `json:"rumors"` // Added for UI display
		AdminFocusNetwork     string                   `json:"admin_focus_network"`
		BannedAvatars         map[string]time.Time     `json:"banned_avatars"`
		BlackMarket           []Loan                   `json:"black_market"` // Added for real-time economy feel
	}{
		Players: players, MaintenanceActive: l.maintenanceMode,
		MaintenanceTime: l.maintenanceTime,
		MaintenancePriority: l.maintenancePriority,
		Clubs:           l.clubs,
		FaucetBalanceMicro:    l.faucetBalanceMicro,
		RewardStack:     l.rewardStack, FaucetBalance: l.faucetBalance,
		TotalVirtualLiability: totalLiabilities,
		ActiveMatchCount:      l.countUniqueMatchesLocked(), Tournament: l.tournament,
		AvailableNetworks: l.availableNetworks, AdminFocusNetwork: l.adminFocusNetwork,
		Rumors:        l.rumors,
		BannedAvatars: l.bannedAvatars,
		BlackMarket:   l.blackMarket,
		RewardRatio:   l.RewardRatio,
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
		// PILLAR 3: Authoritative Pointer Access.
		for i := range l.tournament.Matches {
			match := &l.tournament.Matches[i]
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
					if l.initiatePairedMatch(l.matchmakingPool[idx1].ClientID, l.matchmakingPool[idx2].ClientID, match.ID) {
						// Link to bracket for automatic result reporting
						if mState, ok := l.matches[l.matchmakingPool[idx1].ClientID]; ok {
							mState.TournamentID = l.tournament.ID
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
		p1Stats := l.leaderboard[p1.Wallet]
		p1Wanted := p1Stats.WantedLevel
		p1GhostActive := time.Now().Before(p1Stats.GhostProtocolExpiresAt)

		for j := i + 1; j < len(l.matchmakingPool); j++ {
			if matchedIndices[j] {
				continue
			}
			if tourneyLocked[strings.ToLower(l.matchmakingPool[j].Wallet)] {
				continue
			}

			p2 := l.matchmakingPool[j]
			p2Stats := l.leaderboard[p2.Wallet]
			p2Wanted := p2Stats.WantedLevel
			p2GhostActive := time.Now().Before(p2Stats.GhostProtocolExpiresAt)

			isBounty := false
			// Hunter (Wanted <= 2) vs Outlaw (Wanted >= 10)
			if (p1Wanted <= 2 && p2Wanted >= 10 && !p2GhostActive) || (p2Wanted <= 2 && p1Wanted >= 10 && !p1GhostActive) {
				isBounty = true
			}

			if isBounty {
				// Looser constraints for bounty matches: Reputation diff up to 400
				repDiff := p1.Reputation - p2.Reputation
				if repDiff < 0 {
					repDiff = -repDiff
				}

				if repDiff <= 400 {
					if l.initiatePairedMatch(p1.ClientID, p2.ClientID, "") {
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
				if l.initiatePairedMatch(p1.ClientID, p2.ClientID, "") {
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

func (l *Lobby) initiatePairedMatch(id1, id2 string, tournamentMatchID string) bool {
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

	// PILLAR 1: Regional Transit Tax (Section 11)
	// Deduct 1 $VBV if entering match in non-allied territory.
	const transitTaxMicro = 1 * 1000000
	owningClub := l.getClubByTerritoryID(territoryID)
	
	for _, player := range []struct{ w string; stats *PlayerStats }{ {p1Wallet, &p1Stats}, {p2Wallet, &p2Stats} } {
		if owningClub != nil && !l.clubService.IsPlayerAffiliatedWithClubLocked(l, player.w, owningClub) {
			// PILLAR 3: Smuggler Exemption. Professional transporters ignore regional fees.
			if player.stats.JobRole != "Smuggler" && l.playerBalances[player.w] >= transitTaxMicro {
				l.playerBalances[player.w] -= transitTaxMicro
				
				// Routing: 50% Club Treasury / 50% Governor. If unclaimed, 100% Faucet.
				matrix := RevenueSplitMatrix{FaucetShare: 0.0, ClubShare: 0.5, GovernanceShare: 0.5}
				clubID, _ := strconv.ParseUint(strings.TrimPrefix(owningClub.ID, "CLUB-"), 10, 64)

				_ = l.tokenSinkRouter.RouteCriminalTax("TRANSIT_TAX", transitTaxMicro, matrix, clubID, territoryID)
				l.logAdminAuditLocked("TRANSIT_TAX_COLLECTED", player.w, fmt.Sprintf("District: %s", territoryID))
			}
		}
	}

	l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0
	l.applyDynamicScalingLocked()

	// PILLAR 3: Intelligence Tracking. Update positions for both combatants.
	l.lastSeenDistricts[p1Wallet] = territoryID
	l.lastSeenDistricts[p2Wallet] = territoryID

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
	p1Boost, p2Boost, p1Coalition, p2Coalition := false, false, false, false
	owningClub := l.getClubByTerritoryID(territoryID)

	// PILLAR 1: Conflict Enforcement.
	// Check if the current district is under a Regional Sabotage blackout.
	// If disrupted, all organizational and coalition defensive boosts are disabled.
	disrupted := false
	if owningClub != nil && owningClub.BuffExpirations != nil {
		if expiry, exists := owningClub.BuffExpirations["DISRUPTION_"+territoryID]; exists {
			if time.Now().Before(expiry) {
				disrupted = true
				log.Printf("[WARFARE] Blackout enforced for match in %s. Organizational boosters OFFLINE.\n", territoryID)
			} else {
				// Lazy pruning of expired disruption tags
				delete(owningClub.BuffExpirations, "DISRUPTION_"+territoryID)
			}
		}
	}

	if !disrupted && owningClub != nil && l.clubService.IsClubRegionalLocked(l, owningClub) {
		p1Boost = l.clubService.IsPlayerAffiliatedWithClubLocked(l, p1Wallet, owningClub)
		p2Boost = l.clubService.IsPlayerAffiliatedWithClubLocked(l, p2Wallet, owningClub)
	}

	// PILLAR 1: Coalition Defense Calculation.
	// Members of an allied club receive a +10% boost when defending a partner's territory.
	if !disrupted && owningClub != nil && owningClub.AlliedClubID != "" {
		if allied := l.clubs[owningClub.AlliedClubID]; allied != nil {
			// Check P1
			lowerP1 := strings.ToLower(p1Wallet)
			if strings.EqualFold(allied.OwnerWallet, p1Wallet) {
				p1Coalition = true
			} else if _, ok := allied.Members[lowerP1]; ok {
				p1Coalition = true
			} else if _, ok := allied.Staff[lowerP1]; ok {
				// PILLAR 1: Staff recognition. Ensure specialists receive coalition boosts.
				p1Coalition = true
			}

			// Check P2
			lowerP2 := strings.ToLower(p2Wallet)
			if strings.EqualFold(allied.OwnerWallet, p2Wallet) {
				p2Coalition = true
			} else if _, ok := allied.Members[lowerP2]; ok {
				p2Coalition = true
			} else if _, ok := allied.Staff[lowerP2]; ok {
				// PILLAR 1: Staff recognition.
				p2Coalition = true
			}
		}
	}

	match := &MatchState{
		P1ID: id1, P2ID: id2, P1Deck: m1.P1Deck, P2Deck: m2.P1Deck,
		P1Wallet:         p1Wallet,
		StartTime:        m1.StartTime,
		MatchRating:      m1.MatchRating, // PILLAR 4: Tier Snapshotting.
		P2Wallet:         p2Wallet,
		TournamentMatchID: tournamentMatchID,
		Rules:            matchRules,
		BoardMoods:       boardMoods,
		P1WantedLevel:    p1Stats.WantedLevel,
		P2WantedLevel:    p2Stats.WantedLevel,
		P1Cunning:        p1Stats.GetEffectiveCunning(),
		P1Nurturing:      p1Stats.Nurturing,
		P2Cunning:        l.playerService.GetEffectiveCunning(p2Stats),
		P2Nurturing:      p2Stats.Nurturing,
		Round:            1, // Standard match initialization
		TerritoryID:      territoryID,
		P1RegionalBoost:  p1Boost,
		P2RegionalBoost:  p2Boost,
		P1CoalitionBoost: p1Coalition,
		P2CoalitionBoost: p2Coalition,
		ActiveItemBuffs:  make(map[string]map[string]int),
		Multiplayer:      true, // paired matches are authoritative multiplayer
	}

	// Snapshot persistent buffs into the match state
	if len(p1Stats.ActiveItemBuffs) > 0 {
		match.ActiveItemBuffs[id1] = make(map[string]int)
		for k, v := range p1Stats.ActiveItemBuffs {
			match.ActiveItemBuffs[id1][k] = v
		}
	}
	if len(p2Stats.ActiveItemBuffs) > 0 {
		match.ActiveItemBuffs[id2] = make(map[string]int)
		for k, v := range p2Stats.ActiveItemBuffs {
			match.ActiveItemBuffs[id2][k] = v
		}
	}

	if c1, ok := l.clients[id1]; ok {
		match.P1Avatar, match.P1Gloat = c1.avatarURL, c1.gloat
	}
	if c2, ok := l.clients[id2]; ok {
		match.P2Avatar, match.P2Gloat = c2.avatarURL, c2.gloat
	}

	l.matches[id1], l.matches[id2] = match, match

	// PILLAR 4: Sequence Reset.
	// Explicitly initialize or reset the handshaker for the new match.
	// This ensures that SequenceID starts at 0 for every fresh engagement,
	// preventing catch-up loops from previous sessions.
	l.matchHandshakers[id1] = l.NewSyncHandshaker()

	p1Sync, _ := json.Marshal(Envelope{
		Type: "challenge", FromID: id2, ToID: id1,
		Payload: json.RawMessage(fmt.Sprintf(`{"action":"accept","deck":%v,"wanted_level":%d,"territory":"%s","match_id":"%s","p1_regional_boost":%v,"p2_regional_boost":%v,"p1_coalition_boost":%v,"p2_coalition_boost":%v,"moods":%v}`, jsonList(match.P2Deck), match.P2WantedLevel, match.TerritoryID, match.TournamentMatchID, match.P1RegionalBoost, match.P2RegionalBoost, match.P1CoalitionBoost, match.P2CoalitionBoost, jsonListString(match.BoardMoods[:]))),
	})
	p2Sync, _ := json.Marshal(Envelope{
		Type: "challenge", FromID: id1, ToID: id2,
		Payload: json.RawMessage(fmt.Sprintf(`{"action":"sync_back","deck":%v,"wanted_level":%d,"territory":"%s","match_id":"%s","p1_regional_boost":%v,"p2_regional_boost":%v,"p1_coalition_boost":%v,"p2_coalition_boost":%v,"moods":%v}`, jsonList(match.P1Deck), match.P1WantedLevel, match.TerritoryID, match.TournamentMatchID, match.P1RegionalBoost, match.P2RegionalBoost, match.P1CoalitionBoost, match.P2CoalitionBoost, jsonListString(match.BoardMoods[:]))),
	})

	if c1, ok := l.clients[id1]; ok {
		c1.send <- p1Sync
	}
	if c2, ok := l.clients[id2]; ok {
		c2.send <- p2Sync
	}

	l.sendToClient(id1, Envelope{Type: "matchmaking_status", Payload: json.RawMessage(fmt.Sprintf(`{"status":"match_found","opponent":"%s"}`, p2Wallet))})
	l.sendToClient(id2, Envelope{Type: "matchmaking_status", Payload: json.RawMessage(fmt.Sprintf(`{"status":"match_found","opponent":"%s"}`, p1Wallet))})

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
// PILLAR 6: Blockchain Persistence. Reconstructs state from on-chain snapshots.
func (l *Lobby) loadRegisteredTxIDs() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.registeredTxIDs = make(map[string]time.Time)
	l.loadBlockchainStateSnapshotLocked("VBT_REG_TX_SNAPSHOT:", &l.registeredTxIDs)
}

// saveRegisteredTxIDs saves tournament registration transaction IDs to a file.
// PILLAR 6: Blockchain Persistence. Migrated from local disk to on-chain notes.
func (l *Lobby) saveRegisteredTxIDs() {
	l.mutex.RLock()
	state := l.registeredTxIDs
	l.mutex.RUnlock()
	l.saveBlockchainStateSnapshotLocked("VBT_REG_TX_SNAPSHOT:", state)
}

// loadLinkedWallets loads linked wallet information from a file.
// PILLAR 6: Blockchain Persistence.
func (l *Lobby) loadLinkedWallets() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.linkedWallets = make(map[string]WalletLinkInfo)
	l.loadBlockchainStateSnapshotLocked("VBT_LINK_SNAPSHOT:", &l.linkedWallets)
}

// saveLinkedWallets saves linked wallet information to a file.
// PILLAR 6: Blockchain Persistence. Migrated from local disk to on-chain notes.
func (l *Lobby) saveLinkedWallets() {
	l.mutex.RLock()
	state := l.linkedWallets
	l.mutex.RUnlock()
	l.saveBlockchainStateSnapshotLocked("VBT_LINK_SNAPSHOT:", state)
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

	// PILLAR 6: Blockchain Persistence.
	// Snapshot the updated state immediately while holding the mutex to prevent deadlocks.
	l.saveBlockchainStateSnapshotLocked("VBT_LINK_SNAPSHOT:", l.linkedWallets)
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
	if stats.Playstyle.PreferredItems == nil {
		stats.Playstyle.PreferredItems = make(map[string]float64)
	}
	if stats.ActiveItemBuffs == nil {
		stats.ActiveItemBuffs = make(map[string]int)
	}
	if stats.AuditedClubs == nil {
		stats.AuditedClubs = make(map[string]bool)
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

	// PILLAR 3: Item-Driven Aggressiveness.
	// Certain items reflect offensive intent. We calculate a "Tactical Intent Boost"
	// based on the usage frequency of aggressive hardware and stims.
	itemAggBoost := 0.0
	aggItems := []string{"mood_catalyst", "rule_breaker", "stamina_stim", "sentry_turret"}
	for _, itemID := range aggItems {
		if score, ok := stats.Playstyle.PreferredItems[itemID]; ok {
			itemAggBoost += score * 0.05 // Incremental boost based on usage weight
		}
	}
	if itemAggBoost > 0.2 {
		itemAggBoost = 0.2
	} // Cap the intent boost
	matchAgg += itemAggBoost
	if matchAgg > 1.0 {
		matchAgg = 1.0
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
	anyExpired := false
	for id, rumor := range l.rumors {
		if now.After(rumor.ExpiresAt) {
			log.Printf("[RUMOR] Rumor %s about %s expired.\n", id, rumor.TargetWallet)

			// PILLAR 3: Rumor Decay & Social Standing Sync.
			// 1. Decrement RumorCount for the spreader to reflect temporary influence loss.
			spreaderWallet := strings.ToLower(rumor.SpreaderWallet)
			if s, exists := l.leaderboard[spreaderWallet]; exists {
				if s.RumorCount > 0 {
					s.RumorCount--
				}
				s.Reputation = l.CalculateReputation(s)
				l.leaderboard[spreaderWallet] = s
			}

			// 2. Refresh target Standing to trigger valuation updates in the market.
			targetWallet := strings.ToLower(rumor.TargetWallet)
			if t, exists := l.leaderboard[targetWallet]; exists {
				t.Reputation = l.CalculateReputation(t)
				l.leaderboard[targetWallet] = t
			}

			delete(l.rumors, id)
			anyExpired = true
		}
	}

	if anyExpired {
		msg := l.getLobbyUpdateMsgLocked()
		go func() { l.broadcast <- msg }()
	}
}

// processTreasuryAnalytics monitors club wealth and broadcasts alerts on significant drops.
func (l *Lobby) processTreasuryAnalytics() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Weekly Alpha for Exponential Moving Average (1 / minutes in a week)
	const alpha = 1.0 / 10080.0

	for id, club := range l.clubs {
		avg, exists := l.treasuryAverages[id]
		if !exists {
			l.treasuryAverages[id] = club.Treasury
			continue
		}

		// PILLAR 3: Intelligence Alert.
		// Trigger if current treasury < 10% of weekly average.
		// ignore tiny/new clubs with less than 10 VBV average to prevent spam.
		if avg > 10.0 {
			if club.Treasury < (avg * 0.10) {
				if !l.treasuryCrashed[id] {
					l.treasuryCrashed[id] = true
					msg := fmt.Sprintf("📉 <b>DISTRICT ALERT:</b> %s treasury has plummeted to critical levels! (Current: %.2f, Weekly Avg: %.2f)",
						escapeHTML(club.Name), club.Treasury, avg)
					l.logAdminAuditLocked("TREASURY_CRASH", club.ID, fmt.Sprintf("Treasury: %.2f, Avg: %.2f", club.Treasury, avg))
					payload, _ := json.Marshal(map[string]string{"text": msg})
					l.broadcast <- jsonListEnvelope("chat", payload)
				}
			} else {
				// PILLAR 1: Organizational Achievement.
				l.checkTreasuryRecoveryAchievementLocked(id, club.Treasury, avg)
			}
		}

		// Update the EMA to incorporate the current minute's value.
		// This ensures the "Weekly Average" evolves naturally without storing history slices.
		newAvg := (avg * (1 - alpha)) + (club.Treasury * alpha)
		l.treasuryAverages[id] = newAvg
	}
}

// broadcastBountyBoard identifies top wanted connected players and alerts the lobby.
func (l *Lobby) broadcastBountyBoard() {
	l.mutex.Lock()
	type bounty struct {
		wallet   string
		wanted   int
		district string
		isGhost  bool
	}
	var pool []bounty

	// 1. Identify connected wallets
	connected := make(map[string]bool)
	for _, w := range l.wallets {
		connected[strings.ToLower(w)] = true
	}

	// PILLAR 3: State Bloat Prevention.
	// Lazy prune the intelligence map to remove entries for players who are no longer connected.
	for w := range l.lastSeenDistricts {
		if !connected[w] {
			delete(l.lastSeenDistricts, w)
		}
	}

	// 2. Filter for high-wanted connected players
	for w, stats := range l.leaderboard {
		// PILLAR 3: Signal Scrambling. Check for Signal Dampener in organization.
		isDamped := false
		if stats.EmployerClubID != "" {
			if c, ok := l.clubs[stats.EmployerClubID]; ok {
				if exp, act := c.BuffExpirations["SIGNAL_DAMPENER"]; act && time.Now().Before(exp) {
					isDamped = true
				}
			}
		}

		// PILLAR 3: Intelligence Counter. Revealing a cloaked signal via Disruptor.
		isGhost := (time.Now().Before(stats.GhostProtocolExpiresAt) || isDamped) && time.Now().After(stats.CloakDisruptedUntil)
		if connected[strings.ToLower(w)] && stats.WantedLevel >= 10 {
			district := l.lastSeenDistricts[w]
			if district == "" {
				district = "Unknown Sector"
			}
			pool = append(pool, bounty{wallet: w, wanted: stats.WantedLevel, district: district, isGhost: isGhost})
		}
	}
	l.mutex.Unlock() // Release before name resolution to prevent deadlock

	if len(pool) == 0 {
		return
	}

	// 3. Sort by Wanted Level descending
	sort.Slice(pool, func(i, j int) bool { return pool[i].wanted > pool[j].wanted })

	// 4. Aggregate Top 3
	limit := 3
	if len(pool) < limit {
		limit = len(pool)
	}

	var reports []string
	for i := 0; i < limit; i++ {
		target := pool[i]
		name := l.ResolveEnvoiName(target.wallet)
		district := target.district

		// PILLAR 3: Signal Scrambling.
		// Mask identity and sector for players with active Ghost Protocol.
		if target.isGhost {
			name = "Ghost"
			district = "Undetectable Sector"
		}
		reports = append(reports, fmt.Sprintf("⚠️ <b>%s</b> (Wanted: %d) last seen in <i>%s</i>", template.HTMLEscapeString(name), target.wanted, district))
	}

	msg := fmt.Sprintf("📡 <b>BOUNTY BOARD UPLINK:</b><br/>%s", strings.Join(reports, "<br/>"))
	payload, _ := json.Marshal(map[string]string{"text": msg})
	l.broadcast <- jsonListEnvelope("chat", payload)
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
			// PILLAR 2: Unified Accounting.
			l.clubService.DistributeTournamentKickbackLocked(l, mockWallet, uint64(50*1000000), time.Now())
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

	potMicro := uint64(500 * 1000000)
	if isBuyIn {
		potMicro += l.tournamentPotBonusMicro
		l.tournamentPotBonusMicro = 0
	}

	l.tournament = TournamentState{
		Active:       true,
		ID:           fmt.Sprintf("SIM-T-%d", time.Now().Unix()),
		Participants: participants,
		Matches:      matches,
		CurrentRound: 1, // PILLAR 2: Integer Supremacy
		PotMicro:     potMicro,
		BuyInMicro:   uint64(50 * 1000000),
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

		for i := range currentRoundMatches {
			m := &currentRoundMatches[i]
			winner := m.P1 // Simulated outcome
			if rand.Intn(2) == 1 {
				winner = m.P2
			}
			l.tournamentService.ProcessTournamentResult(l, m.ID, winner)
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
	decayedClubIDs := make(map[string]bool)

	for _, club := range l.clubs {
		if club.Mojo <= 0 {
			continue
		}

		if now.Sub(club.LastActivity) > stagnationThreshold {
			isMojoStabilizerActive, decayRate := l.calculateMojoDecayRateLocked(club.ID)
			decayAmount := int(float64(club.Mojo)*decayRate + 0.5)
			if decayAmount < minDecay {
				decayAmount = minDecay
			}

			club.Mojo -= decayAmount
			if club.Mojo < 0 {
				club.Mojo = 0
			}
			decayOccurred = true
			decayedClubIDs[club.ID] = true
			log.Printf("[INDUSTRIAL] Club %s suffered Mojo decay (isRegional: %v). New Mojo: %d\n", club.Name, l.isClubRegionalLocked(club), club.Mojo)

			// Reset clock to 'now' so decay is periodic (e.g., every 48h) rather than continuous
			club.LastActivity = now
		}
	}

	// PILLAR 1: Performance-Optimized Reputation Ripple.
	// If any clubs decayed, perform a single pass over the leaderboard to update
	// employee standings, preventing O(N^2) complexity.
	if len(decayedClubIDs) > 0 {
		for wallet, stats := range l.leaderboard {
			if decayedClubIDs[stats.EmployerClubID] {
				stats.Reputation = l.CalculateReputation(stats)
				l.leaderboard[wallet] = stats
			}
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
	// PILLAR 3: Local Persistent Archive.
	// Save a high-fidelity JSON snapshot to the DATA_DIR before resetting state.
	archivePath := l.getDataPath(fmt.Sprintf("season_%d_archive.json", l.seasonNumber))
	if err := os.WriteFile(archivePath, jsonData, 0644); err != nil {
		log.Printf("[SEASON ERROR] Failed to save local persistent archive: %v\n", err)
	}

	// PILLAR 6: Forensic Continuity.
	// Commit a final blockchain-native snapshot of the economy for the
	// concluding season before resetting the temporal counters.
	l.saveSeasonMetadataLocked()

	// PILLAR 1: Stagnation Reset.
	// Ensure all organizations start the new season with a fresh activity clock.
	// This prevents "Instant Decay" if a rollover occurs late in a stagnation window.
	for _, club := range l.clubs {
		club.LastActivity = time.Now()
	}

	// Reset Cycle
	l.seasonNumber++
	l.seasonStart = time.Now()
	l.leaderboard = make(map[string]PlayerStats) // Clear HoF

	// Persist new config while preserving initialRewards
	l.saveSeasonMetadataLocked()
	l.mutex.Unlock()

	// PILLAR 3: Persistence Sync.
	// Update the on-disk leaderboard.json to reflect the reset state immediately.
	l.saveLeaderboard()

	// PILLAR 3: Audit trail completion.
	l.logAdminAudit("SEASON_ROLLOVER_COMPLETE", "GLOBAL", fmt.Sprintf("Season %d reset and committed to disk.", l.seasonNumber-1))

	// PILLAR 4: Deadlock Prevention.
	// Dispatch the on-chain transaction AFTER releasing the state lock to avoid
	// recursive lock contention within sendNoteTx.
	note := fmt.Sprintf("VBT_SEASON_ARCHIVE:%s", string(jsonData))
	l.sendNoteTx(note) // Record on chain

	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// ProcessDailyAdministrativeMaintenanceFee deducts a fee from all organizational treasuries daily.
// PILLAR 2: Administrative Maintenance Pool (Task 2412).
func (l *Lobby) ProcessDailyAdministrativeMaintenanceFee() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	const maintenanceFeeMicro = 10 * 1000000 // 10 $VBV
	anyFeeCollected := false

	if l.tokenSinkRouter != nil {
		for _, club := range l.clubs {
			numericID, _ := strconv.ParseUint(strings.TrimPrefix(club.ID, "CLUB-"), 10, 64)
			l.tokenSinkRouter.Mu.Lock()
			if node, ok := l.tokenSinkRouter.ActiveClubs[numericID]; ok {
				if node.TreasuryBalance >= maintenanceFeeMicro {
					node.TreasuryBalance -= maintenanceFeeMicro
					club.TreasuryMicro = node.TreasuryBalance
					club.Treasury = club.TreasuryMicro

					// Route to Faucet via Router (100% Faucet share)
					matrix := RevenueSplitMatrix{FaucetShare: 1.0, ClubShare: 0.0, GovernanceShare: 0.0}
					_ = l.tokenSinkRouter.RouteCriminalTax("ADMIN_MAINTENANCE", maintenanceFeeMicro, matrix, 0, "")
					l.logAdminAuditLocked("ADMIN_MAINTENANCE_FEE", club.ID, "Deducted 10 $VBV daily maintenance fee")
					anyFeeCollected = true
				} else {
					l.logAdminAuditLocked("ADMIN_MAINTENANCE_SKIPPED", club.ID, "Treasury insufficient for daily maintenance fee")
				}
			}
			l.tokenSinkRouter.Mu.Unlock()
		}
		if anyFeeCollected {
			l.faucetBalance = float64(l.faucetBalanceMicro) / 1000000.0
			l.applyDynamicScalingLocked()
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		}
	}
}

// ensurePlayerStatsMapsInitialized ensures that all map fields in PlayerStats are initialized.
func (l *Lobby) ensurePlayerStatsMapsInitialized(wallet string) {
	stats := l.leaderboard[wallet]

	// PILLAR 3: Identity Persistence.
	// Populate the Wallet field to enable Governor and Owner-based reputation multipliers.
	stats.Wallet = wallet

	if stats.Inventory == nil {
		stats.Inventory = make(map[string]int)
	}
	if stats.Relationships == nil {
		stats.Relationships = make(map[string]int)
	}
	if stats.Portfolio == nil {
		stats.Portfolio = make(map[string]uint64) // PILLAR 2: Integer Supremacy
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
	if stats.AuditedClubs == nil {
		stats.AuditedClubs = make(map[string]bool)
	}
	if stats.CapturedOutlaws == nil {
		stats.CapturedOutlaws = make(map[string]bool)
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
	if stats.HeistAlarmsJammerCount == 0 {
		stats.HeistAlarmsJammerCount = 0
	}
	// RumorCount is an int, no map initialization needed
	l.leaderboard[wallet] = stats
}

// refreshRegionalRoles updates governor status for clubs based on territory control.
func (l *Lobby) refreshRegionalRoles() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	changedClubIDs := make(map[string]bool)

	for _, club := range l.clubs {
		isRegional := l.isClubRegionalLocked(club)
		wasRegional := club.RegionName != ""

		if isRegional {
			// Set a default region name for governors, or keep existing if set
			if club.RegionName == "" {
				club.RegionName = "Governor"
				changedClubIDs[club.ID] = true
			}

			// PILLAR 1: Achievement Integration.
			// Grant the GOVERNOR trophy to the club owner for expanded regional influence.
			// Note: unlockAchievementLocked handles reputation recalculation and seeds
			// the record if this is running immediately after a season reset.
			l.achievementService.UnlockAchievementLocked(l, strings.ToLower(club.OwnerWallet), "GOVERNOR")
		} else {
			if wasRegional {
				// Remove governor status if they no longer control 2+ territories
				club.RegionName = ""
				changedClubIDs[club.ID] = true
			}
		}
	}

	// PILLAR 1: Reputation Ripple.
	// If a club's regional status changed, update the owner's reputation to reflect
	// the loss or gain of the administrative influence multiplier.
	if len(changedClubIDs) > 0 {
		for wallet, stats := range l.leaderboard {
			// Owners receive a +10% multiplier bonus while their club is regional.
			if changedClubIDs[stats.EmployerClubID] && strings.EqualFold(l.clubs[stats.EmployerClubID].OwnerWallet, wallet) {
				stats.Reputation = l.CalculateReputation(stats)
				l.leaderboard[wallet] = stats
			}
		}
	}
}

// simulateMojoDecayStressTest creates a cluster of regional clubs and simulates Mojo decay over time.
// This is intended for performance and logic validation of the processMojoDecay function.
func (l *Lobby) simulateMojoDecayStressTest(numClubs int, durationMinutes int) {
	l.mutex.Lock()
	log.Printf("[SIMULATION] Starting Mojo Decay Stress Test: %d clubs for %d minutes...\n", numClubs, durationMinutes)

	// Clear existing clubs to isolate the test environment
	originalClubs := l.clubs
	l.clubs = make(map[string]*Club)

	// 1. Generate test clubs
	testClubIDs := make([]string, numClubs)
	for i := 0; i < numClubs; i++ {
		clubID := fmt.Sprintf("SIM_CLUB_%d_%d", i, time.Now().UnixNano()%10000)
		testClubIDs[i] = clubID

		// Ensure clubs are regional (2+ territories)
		territories := []string{fmt.Sprintf("TERR_%d_A", i), fmt.Sprintf("TERR_%d_B", i)}

		// Vary member counts (0 to 50)
		numMembers := rand.Intn(51)
		members := make(map[string]time.Time)
		for m := 0; m < numMembers; m++ {
			members[fmt.Sprintf("SIM_MEMBER_%d_%d", i, m)] = time.Now().Add(-time.Duration(rand.Intn(24)) * time.Hour)
		}

		// Random initial Mojo (0 to 1000)
		initialMojo := rand.Intn(1001)

		// Randomly activate MOJO_STABILIZER
		buffExpirations := make(map[string]time.Time)
		activeBuffs := make(map[string]string)
		if rand.Float32() < 0.3 { // 30% chance to have stabilizer
			activeBuffs["MOJO_STABILIZER"] = "district_stabilizer"
			buffExpirations["MOJO_STABILIZER"] = time.Now().Add(time.Duration(rand.Intn(48)+1) * time.Hour) // 1-48 hours
		}

		l.clubs[clubID] = &Club{
			ID:              clubID,
			Name:            fmt.Sprintf("Simulated Club %d", i),
			OwnerWallet:     fmt.Sprintf("SIM_OWNER_%d", i),
			Territories:     territories,
			RegionName:      "Governor", // Mark as regional
			Treasury:        float64(rand.Intn(10000)),
			Mojo:            initialMojo,
			Members:         members,
			ActiveBuffs:     activeBuffs,
			BuffExpirations: buffExpirations,
			LastActivity:    time.Now().Add(-time.Duration(rand.Intn(72)) * time.Hour), // Random last activity
			CreatedAt:       time.Now(),
		}
	}
	l.mutex.Unlock()

	// 2. Simulate decay over durationMinutes
	for minute := 0; minute < durationMinutes; minute++ {
		l.mutex.Lock()
		log.Printf("[SIMULATION] Mojo Decay Test - Minute %d. Processing decay...\n", minute+1)
		l.processMojoDecayLocked() // FIXED: Use locked variant to avoid deadlock

		// Log Mojo for a few sample clubs
		for i := 0; i < 3 && i < numClubs; i++ {
			club := l.clubs[testClubIDs[i]]
			log.Printf("[SIMULATION] Club %s (Mojo: %d, Stabilizer: %v) after %d min.\n", club.Name, club.Mojo, club.ActiveBuffs["MOJO_STABILIZER"] != "", minute+1)
		}
		l.mutex.Unlock()
		time.Sleep(1 * time.Second) // Simulate real-time minute processing
	}

	log.Printf("[SIMULATION] Mojo Decay Stress Test Complete. Cleaning up simulated clubs.\n")
	l.mutex.Lock()
	l.clubs = originalClubs // Restore original clubs
	l.mutex.Unlock()
}

// processAllianceExpirations checks for expired alliance invitations and clears them.
// PILLAR 1: Alliance Integration.
func (l *Lobby) processAllianceExpirations() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	now := time.Now()
	anyExpired := false

	for _, club := range l.clubs {
		if club.AllianceInviteID != "" && !club.AllianceInviteExpiresAt.IsZero() && now.After(club.AllianceInviteExpiresAt) {
			log.Printf("[SOCIAL] Alliance proposal from %s to %s has expired.\n", club.AllianceInviteID, club.Name)

			// Log the expiration for administrative tracking
			l.logAdminAuditLocked("ALLIANCE_EXPIRED", club.ID, fmt.Sprintf("Proposal from %s expired.", club.AllianceInviteID))

			club.AllianceInviteID = ""
			club.AllianceInviteExpiresAt = time.Time{}
			anyExpired = true
		}
	}

	if anyExpired {
		go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
	}
}

// calculateMojoDecayRateLocked determines the current Mojo decay rate for a club.
// This function assumes the Lobby mutex is held by the caller.
// PILLAR 1: Infrastructure Prestige.
func (l *Lobby) calculateMojoDecayRateLocked(clubID string) (bool, float64) {
	club, exists := l.clubs[clubID]
	if !exists || club.Mojo <= 0 {
		return false, 0.0
	}

	now := time.Now()
	stagnationThreshold := 48 * time.Hour

	// If not stagnant, no decay is currently applied.
	if now.Sub(club.LastActivity) <= stagnationThreshold {
		return false, 0.0
	}

	// 1. Check for active MOJO_STABILIZER buff.
	isMojoStabilizerActive := false
	if expiry, exists := club.BuffExpirations["MOJO_STABILIZER"]; exists {
		if now.Before(expiry) {
			isMojoStabilizerActive = true
		} else {
			// Buff expired, remove it
			delete(club.ActiveBuffs, "MOJO_STABILIZER")
			delete(club.BuffExpirations, "MOJO_STABILIZER")
			log.Printf("[INDUSTRIAL] MOJO_STABILIZER expired for club %s\n", club.Name)
		}
	}

	// 2. Determine base decay rate.
	isRegion := l.isClubRegionalLocked(club)
	decayRate := 0.02

	if isRegion {
		// Established regions suffer 2.5x higher decay to prevent sector stagnation.
		decayRate = 0.05
	}

	// Inactive Member Scaling.
	decayRate += float64(len(club.Members)) * 0.002

	// 3. Apply District Stabilizer Effect.
	if isMojoStabilizerActive {
		decayRate *= 0.50
	}

	return isMojoStabilizerActive, decayRate
}

// PILLAR 3: Justice Layer.
func (l *Lobby) HandleGetJusticeMissions(w http.ResponseWriter, r *http.Request) {
	// Pro-social objectives: Bounty hunting, Bail assistance, and Regulatory audits.
	missions := []JusticeMission{
		{
			ID:          "MISSION-001",
			Title:       "Apprehend High-Infamy Signatures",
			Description: "Capture a player with Wanted Level 15+ in combat. Uphold the Arena's security protocols.",
			RewardMicro: 1200 * 1000000, // 1,200 VBV
			Target:      "wanted_15",
			Type:        "Bounty",
		},
		{
			ID:          "MISSION-002",
			Title:       "Facilitate Legal Rehabilitation",
			Description: "Process a Bail payment for any card currently held in organization jails. Restore liquidity to the sector.",
			RewardMicro: 500 * 1000000, // 500 VBV
			Target:      "jail_bail",
			Type:        "Bail",
		},
		{
			ID:          "MISSION-003",
			Title:       "Conduct Regulatory Audit",
			Description: "Perform a successful Cyber-Audit on any club controlling the Elemental Forge district. Monitor organizational health.",
			RewardMicro: 800 * 1000000, // 800 VBV
			Target:      "elemental_forge",
			Type:        "Audit",
		},
		{
			ID:          "MISSION-004",
			Title:       "Regional Peacekeeping",
			Description: "Apprehend a signature associated with a Regional Governor (Clubs owning 2+ districts). Enforce sector stability.",
			RewardMicro: 1000 * 1000000, // 1,000 VBV
			Target:      "governor_member",
			Type:        "Bounty",
		},
		{
			ID:          "MISSION-005",
			Title:       "Forensic Audit: Grade F",
			Description: "Conduct a forensic audit that reduces a target card's grade to 'F' (Artifact <= -100).",
			RewardMicro: 2000 * 1000000, // 2,000 VBV
			Target:      "card_grade_f",
			Type:        "Audit",
		},
		{
			ID:          "MISSION-006",
			Title:       "Amplify Justice Reputation",
			Description: "Successfully spread a Positive Rumor about a Justice-aligned player. Reinforce sector trust.",
			RewardMicro: 1500 * 1000000, // 1,500 VBV
			Target:      "justice_player_rumor",
			Type:        "Rumor",
		},
		{
			ID:          "MISSION-007",
			Title:       "Targeted Forensic Seizure",
			Description: "Conduct a forensic audit that reduces a card owned by a high-infamy outlaw (Wanted 15+) to 'F' grade (Artifact <= -100).",
			RewardMicro: 2500 * 1000000, // 2,500 VBV
			Target:      "outlaw_card_grade_f",
			Type:        "Audit",
		},
		{
			ID:          "MISSION-008",
			Title:       "Governor's Asset Devaluation",
			Description: "Conduct a forensic audit that reduces a Regional Governor's favorite card to 'F' grade (Artifact <= -100).",
			RewardMicro: 3000 * 1000000, // 3,000 VBV
			Target:      "governor_favorite_f",
			Type:        "Audit",
		},
		{
			ID:          "MISSION-009",
			Title:       "Security Corruption Exposure",
			Description: "Flag an outlaw (Wanted 15+) currently employed as 'Security' by a Regional Governor. Expose organizational corruption.",
			RewardMicro: 4000 * 1000000, // 4,000 VBV
			Target:      "governor_security_outlaw",
			Type:        "Enforcement",
		},
		{
			ID:          "MISSION-010",
			Title:       "Alliance Shadow Network",
			Description: "Flag two different outlaws (Wanted 10+) belonging to the same Alliance. Expose coordinated criminal syndicates.",
			RewardMicro: 5000 * 1000000, // 5,000 VBV
			Target:      "alliance_coordinated_outlaws",
			Type:        "Enforcement",
		},
		{
			ID:          "MISSION-011",
			Title:       "Cripple Criminal Leadership",
			Description: "Flag an outlaw (Wanted 20+) who is currently the Owner of a club. Strike at the head of the serpent.",
			RewardMicro: 6000 * 1000000, // 6,000 VBV
			Target:      "outlaw_club_owner",
			Type:        "Enforcement",
		},
		{
			ID:          "MISSION-012",
			Title:       "Regional Corruption Audit",
			Description: "Flag a Regional Governor (Owner of 2+ districts) with a Wanted Level of 10+. Enforce transparency at the highest level.",
			RewardMicro: 10000 * 1000000, // 10,000 VBV
			Target:      "governor_wanted_10",
			Type:        "Enforcement",
		},
		{
			ID:          "MISSION-013",
			Title:       "Hostage Kingpin Takedown",
			Description: "Flag an outlaw (Wanted 25+) who has kidnapped at least 2 cards from different owners. Dismantle criminal abduction rings.",
			RewardMicro: 7500 * 1000000, // 7,500 VBV
			Target:      "hostage_kingpin",
			Type:        "Enforcement",
		},
		{
			ID:          "MISSION-014",
			Title:       "High-Stakes Governor Audit",
			Description: "Flag a Regional Governor (Wanted 15+) currently holding 2+ hostages. Dismantle institutional kidnapping rings.",
			RewardMicro: 15000 * 1000000, // 15,000 VBV
			Target:      "governor_wanted_15_hostage_2",
			Type:        "Enforcement",
		},
		{
			ID:          "MISSION-015",
			Title:       "Hegemony Counter-Strike",
			Description: "Flag an outlaw (Wanted 30+) whose infamy indicates involvement in an Arena Center Fortress Breach. Protect the core.",
			RewardMicro: 20000 * 1000000, // 20,000 VBV
			Target:      "fortress_breach_perp",
			Type:        "Enforcement",
		},
		{
			ID:          "MISSION-016",
			Title:       "Arena Center Hegemony Audit",
			Description: "Flag an outlaw (Wanted 35+) who is currently the Owner of the organization controlling the 'Arena Center'. Decapitate the sector's central authority.",
			RewardMicro: 25000 * 1000000, // 25,000 VBV
			Target:      "arena_center_owner_wanted_35",
			Type:        "Enforcement",
		},
		{
			ID:          "MISSION-017",
			Title:       "Chaos Containment Protocol",
			Description: "Flag an outlaw (Wanted 40+) who has executed a 'CONTRACT-019' event (The Invisible Hand of Chaos). Restore order.",
			RewardMicro: 30000 * 1000000, // 30,000 VBV
			Target:      "chaos_perp_wanted_40",
			Type:        "Enforcement",
		},
		{
			ID:          "MISSION-018",
			Title:       "Criminal Employment Audit",
			Description: "Flag an outlaw (Wanted 20+) who is currently employing a 'Criminal' role player. Dismantle the illicit workforce.",
			RewardMicro: 10000 * 1000000, // 10,000 VBV
			Target:      "criminal_employer_wanted_20",
			Type:        "Enforcement",
		},
		{
			ID:          "MISSION-019",
			Title:       "Financial Laundering Audit",
			Description: "Flag an outlaw (Wanted 25+) who is currently employing a 'Launderer' role player. Sever the criminal financial link.",
			RewardMicro: 15000 * 1000000, // 15,000 VBV
			Target:      "launderer_employer_wanted_25",
			Type:        "Enforcement",
		},
		{
			ID:          "MISSION-020",
			Title:       "Apex Predator Takedown",
			Description: "Flag an outlaw (Wanted 50+) who has successfully executed both 'CONTRACT-018' and 'CONTRACT-019'. Neutralize the ultimate threat.",
			RewardMicro: 40000 * 1000000, // 40,000 VBV
			Target:      "apex_predator_wanted_50",
			Type:        "Enforcement",
		},
		{
			ID:          "MISSION-021",
			Title:       "Hostage Syndicate Takedown",
			Description: "Flag an outlaw (Wanted 60+) who is currently holding 3+ hostages simultaneously. Terminate the syndicate.",
			RewardMicro: 50000 * 1000000, // 50,000 VBV
			Target:      "hostage_syndicate_wanted_60",
			Type:        "Enforcement",
		},
		{
			ID:          "MISSION-022",
			Title:       "Sovereign Hostage Recovery",
			Description: "Flag an outlaw (Wanted 70+) who has successfully executed 'CONTRACT-022'. Bring the ultimate syndicate to justice.",
			RewardMicro: 35000 * 1000000, // 35,000 VBV
			Target:      "sovereign_perp_wanted_70",
			Type:        "Enforcement",
		},
		{
			ID:          "MISSION-023",
			Title:       "Ultimate Syndicate Dissolution",
			Description: "Flag an outlaw (Wanted 80+) who has successfully executed both 'CONTRACT-021' and 'CONTRACT-022'. Bring an end to the peak syndicate dominance.",
			RewardMicro: 40000 * 1000000, // 40,000 VBV
			Target:      "syndicate_boss_wanted_80",
			Type:        "Enforcement",
		},
		{
			ID:          "MISSION-024",
			Title:       "Apex Scourge Neutralization",
			Description: "Flag an outlaw (Wanted 110+) who has successfully executed 'CONTRACT-023'. End the ultimate criminal threat.",
			RewardMicro: 50000 * 1000000, // 50,000 VBV
			Target:      "apex_scourge_wanted_110",
			Type:        "Enforcement",
		},
		{
			ID:          "MISSION-025",
			Title:       "Tax Haven Exposure",
			Description: "Flag a Club Owner whose Treasury exceeds 10,000 $VBV. Requires the 'Tax Auditor' career role.",
			RewardMicro: 25000 * 1000000, // 25,000 VBV
			Target:      "tax_haven_audit",
			Type:        "Audit",
		},
		{
			ID:          "MISSION-026",
			Title:       "Abduction Staff Audit",
			Description: "Flag an outlaw (Wanted 15+) who employs at least one 'Kidnapper' in their organization. Requires the 'Tax Auditor' career role.",
			RewardMicro: 12000 * 1000000, // 12,000 VBV
			Target:      "kidnapper_staff_audit",
			Type:        "Audit",
		},
		{
			ID:          "MISSION-027",
			Title:       "Arena Center Breach Audit",
			Description: "Flag an outlaw (Wanted 40+) who has successfully executed a heist in the 'Arena Center' district. Requires the 'Intel-Agent' career role.",
			RewardMicro: 15000 * 1000000, // 15,000 VBV
			Target:      "arena_center_heist_audit",
			Type:        "Enforcement",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(missions)
}

// HandleAcceptJusticeMission processes a player's request to begin a law-enforcement objective.
// PILLAR 3: Justice Layer.
// $VBV-GATE + CareerXP Discount: Mission deployment fees scale with Justice career tier.
func (l *Lobby) HandleAcceptJusticeMission(env *Envelope) {
	var data struct {
		MissionID string `json:"mission_id"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		log.Printf("[JUSTICE] Invalid accept_justice_mission payload from %s: %v\n", env.FromID, err)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	wallet, ok := l.wallets[env.FromID]
	if !ok {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Access Denied: Identification failed."}`)})
		return
	}

	l.ensurePlayerStatsMapsInitialized(wallet)
	stats := l.leaderboard[wallet]

	// PILLAR 3: CareerXP Discount — Justice deployment fee scales by career tier.
	// Base cost: 1,000 $VBV. Tiers: Bronze(0%), Silver(-15%), Gold(-30%), Boss/Platinum(-45%).
	const standardJusticeMissionFee = 1000 * 1000000
	justiceMissionFee := standardJusticeMissionFee

	if stats.JobRole == "Intel-Agent" || stats.JobRole == "Bounty Hunter" ||
		stats.JobRole == "AOS Leader" || stats.JobRole == "Justice Recruiter" ||
		stats.JobRole == "Warden" || stats.JobRole == "Forensic Analyst" ||
		stats.JobRole == "Tax Auditor" || stats.JobRole == "Judge" ||
		stats.JobRole == "Justice Commissioner" || stats.JobRole == "Sector Peacekeeper" {

		if stats.CareerXP != nil && stats.CareerXP.Level > 0 {
			discount := stats.CareerXP.GetJusticeMissionFeeDiscount()
			justiceMissionFee = int(float64(standardJusticeMissionFee) * discount)

			// PILLAR 13: Add visible buff tag when discount active
			if stats.ActiveBuffs == nil {
				stats.ActiveBuffs = make(map[string]string)
			}
			if discount < 1.0 {
				stats.ActiveBuffs["Justice_Discount"] = "active"
				log.Printf("[JUSTICE] CareerXP discount applied: %s (Tier %d, Rate: %.0f%%)\n",
					stats.JobRole, stats.CareerXP.Level, discount*100)
			} else {
				delete(stats.ActiveBuffs, "Justice_Discount")
			}
		}
	}

	if l.playerBalances[wallet] < justiceMissionFee {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"❌ Mission Failed: Insufficient $VBV. Deployment fee: %.0f $VBV (CareerXP discount applied)."}`, float64(justiceMissionFee)/1000000.0))})
		return
	}

	// Charge deployment fee
	l.playerBalances[wallet] -= justiceMissionFee

	// PILLAR 3: Career Role Gating (must come AFTER fee check so discount applies)
	if (data.MissionID == "MISSION-025" || data.MissionID == "MISSION-026") && stats.JobRole != "Tax Auditor" {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Access Denied: This mission requires the 'Tax Auditor' career path."}`)})
		return
	}
	if data.MissionID == "MISSION-027" && stats.JobRole != "Intel-Agent" {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Access Denied: This mission requires the 'Intel-Agent' career path."}`)})
		return
	}

	// PILLAR 3: Bounty Hunter License Gating (Section 11).
	isJustice := l.playerService.GetHegemonyPath(stats.JobRole) == "JUSTICE"
	if isJustice && (stats.BountyHunterLicenseExpiresAt.IsZero() || time.Now().After(stats.BountyHunterLicenseExpiresAt)) {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Access Denied: Enforcement License expired. Renew at the Social Hub."}`)})
		return
	}

	// PILLAR 1 & 3: Justice Bond Gating.
	// Missions MISSION-020 through MISSION-027 represent high-tier objectives requiring capital commitment.
	isHighTier := false
	if strings.HasPrefix(data.MissionID, "MISSION-") {
		midStr := strings.TrimPrefix(data.MissionID, "MISSION-")
		// Support multi-stage mission ID parsing (e.g., MISSION-010:target)
		midStr = strings.Split(midStr, ":")[0]
		mid, _ := strconv.Atoi(midStr)
		if mid >= 20 && mid <= 27 { isHighTier = true }
	}
	if isHighTier && stats.BountyHunterBondMicro == 0 {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Access Denied: High-tier Justice missions require a security bond deposit."}`)})
		return
	}

	// 1. Mission Capacity Check
	if stats.ActiveJusticeMissionID != "" {
		l.sendToClientLocked(env.FromID, Envelope{
			Type:    "admin_notification",
			Payload: json.RawMessage(`{"text":"⚖️ <b>MISSION ACTIVE:</b> You are already assigned to a pro-social operation. Complete or abort it first."}`),
		})
		return
	}

	// 2. Mission Validation
	valid := false
	switch data.MissionID {
	case "MISSION-001", "MISSION-002", "MISSION-003", "MISSION-004", "MISSION-005", "MISSION-006", "MISSION-007", "MISSION-008", "MISSION-009", "MISSION-010", "MISSION-011", "MISSION-012", "MISSION-013", "MISSION-014", "MISSION-015", "MISSION-016", "MISSION-017", "MISSION-018", "MISSION-019", "MISSION-020", "MISSION-021", "MISSION-022", "MISSION-023", "MISSION-024", "MISSION-025", "MISSION-026", "MISSION-027":
		valid = true
	}

	if !valid {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Error: Specified mission dossier is restricted or missing."}`)})
		return
	}

	// 3. Mission Assignment
	stats.ActiveJusticeMissionID = data.MissionID
	l.leaderboard[wallet] = stats
	l.logAdminAuditLocked("JUSTICE_MISSION_ACCEPTED", wallet, fmt.Sprintf("ID: %s", data.MissionID))

	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"⚖️ <b>MISSION ACCEPTED:</b> Dossier downloaded. Enforce the Arena's protocols."}`)})
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}

// HandleJusticeFlagPlayer processes a request from a law-enforcement player to flag an outlaw.
// PILLAR 3: Criminality & Intelligence.
func (l *Lobby) HandleJusticeFlagPlayer(env *Envelope) {
	var data struct {
		TargetWallet string `json:"target_wallet"`
	}
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		log.Printf("[JUSTICE] Invalid justice_flag_player payload from %s: %v\n", env.FromID, err)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	senderWallet, ok := l.wallets[env.FromID]
	if !ok {
		return
	}

	l.ensurePlayerStatsMapsInitialized(senderWallet)
	stats := l.leaderboard[senderWallet]

	// 1. Authorization Check: Only Justice-aligned players can use the terminal.
	if l.playerService.GetHegemonyPath(stats.JobRole) != "JUSTICE" {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Access Denied: Justice Terminal restricted to law enforcement personnel."}`)})
		return
	}

	targetWallet := strings.ToLower(data.TargetWallet)
	targetStats, exists := l.leaderboard[targetWallet]

	// 2. Target Validation: High infamy check (Wanted Level 10+).
	if !exists || targetStats.WantedLevel < 10 {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"⚠️ <b>INVALID SIGNATURE:</b> Target does not meet the infamy threshold for priority flagging."}`)})
		return
	}

	// 3. Flagging & Audit
	l.logAdminAuditLocked("JUSTICE_FLAG", senderWallet, fmt.Sprintf("Flagged: %s (Wanted: %d)", targetWallet, targetStats.WantedLevel))

	// PILLAR 3: Justice Mission Completion (MISSION-009)
	// Security Corruption Exposure: Flag an outlaw (Wanted 15+) employed as 'Security' by a Regional Governor.
	if stats.ActiveJusticeMissionID == "MISSION-009" && targetStats.WantedLevel >= 15 && targetStats.JobRole == "Security" && targetStats.EmployerClubID != "" {
		club, exists := l.clubs[targetStats.EmployerClubID]
		if exists && l.clubService.IsClubRegionalLocked(l, club) {
			const rewardMicro = 4000 * 1000000
			l.playerBalances[senderWallet] += rewardMicro
			stats.ActiveJusticeMissionID = ""
			l.leaderboard[senderWallet] = stats
			l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", senderWallet, "ID: MISSION-009, Payout: 4000.00")
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> Security corruption exposed. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
			l.applyDynamicScalingLocked()
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		}
	}

	// PILLAR 3: Justice Mission Completion (MISSION-010)
	// Alliance Shadow Network: Flag two different outlaws (Wanted 10+) in the same Alliance.
	if strings.HasPrefix(stats.ActiveJusticeMissionID, "MISSION-010") && targetStats.WantedLevel >= 10 {
		if club, exists := l.clubs[targetStats.EmployerClubID]; exists && club.AlliedClubID != "" {
			// Determine unique Alliance ID (sorted pair of club IDs)
			cid1, cid2 := club.ID, club.AlliedClubID
			if cid1 > cid2 {
				cid1, cid2 = cid2, cid1
			}
			allianceUID := cid1 + ":" + cid2

			if stats.ActiveJusticeMissionID == "MISSION-010" {
				// First target flagged
				stats.ActiveJusticeMissionID = fmt.Sprintf("MISSION-010:%s:%s", targetWallet, allianceUID)
				l.leaderboard[senderWallet] = stats
				l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"⚖️ <b>MISSION UPDATE:</b> First target flagged. Locate another outlaw in alliance with %s."}`, escapeHTML(club.Name)))})
			} else {
				// Second target check
				parts := strings.Split(stats.ActiveJusticeMissionID, ":")
				if len(parts) == 3 && parts[0] == "MISSION-010" {
					prevWallet := parts[1]
					prevAlliance := parts[2]

					if !strings.EqualFold(targetWallet, prevWallet) && allianceUID == prevAlliance {
						const rewardMicro = 5000 * 1000000
						l.playerBalances[senderWallet] += rewardMicro
						stats.ActiveJusticeMissionID = ""
						l.leaderboard[senderWallet] = stats
						l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", senderWallet, "ID: MISSION-010, Payout: 5000.00")
						l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> Alliance shadow network exposed. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
						l.applyDynamicScalingLocked()
						go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
					} else if strings.EqualFold(targetWallet, prevWallet) {
						l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"⚠️ Already flagged this signature. Locate their accomplice."}`)})
					}
				}
			}
		}
	}

	// PILLAR 3: Justice Mission Completion (MISSION-011)
	// Cripple Criminal Leadership: Flag an outlaw (Wanted 20+) who is a Club Owner.
	if stats.ActiveJusticeMissionID == "MISSION-011" && targetStats.WantedLevel >= 20 {
		isOwner := false
		for _, club := range l.clubs {
			if strings.EqualFold(club.OwnerWallet, targetWallet) {
				isOwner = true
				break
			}
		}
		if isOwner {
			const rewardMicro = 6000 * 1000000
			l.playerBalances[senderWallet] += rewardMicro
			stats.ActiveJusticeMissionID = ""
			l.leaderboard[senderWallet] = stats
			l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", senderWallet, "ID: MISSION-011, Payout: 6000.00")
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> Criminal leadership exposed. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
			l.applyDynamicScalingLocked()
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		}
	}

	// PILLAR 3: Justice Mission Completion (MISSION-012)
	// Regional Corruption Audit: Flag a Regional Governor with Wanted Level 10+.
	if stats.ActiveJusticeMissionID == "MISSION-012" && targetStats.WantedLevel >= 10 {
		isGov := false
		if club, exists := l.clubs[targetStats.EmployerClubID]; exists && strings.EqualFold(club.OwnerWallet, targetWallet) {
			if l.clubService.IsClubRegionalLocked(l, club) {
				isGov = true
			}
		}
		if isGov {
			const rewardMicro = 10000 * 1000000
			l.playerBalances[senderWallet] += rewardMicro
			stats.ActiveJusticeMissionID = ""
			l.leaderboard[senderWallet] = stats
			l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", senderWallet, "ID: MISSION-012, Payout: 10000.00")
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> Regional corruption flagged. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
			l.applyDynamicScalingLocked()
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		}
	}

	// PILLAR 3: Justice Mission Completion (MISSION-013)
	// Hostage Kingpin Takedown: Flag an outlaw (Wanted 25+) with 2+ kidnapped cards from different owners.
	if stats.ActiveJusticeMissionID == "MISSION-013" && targetStats.WantedLevel >= 25 {
		if len(targetStats.KidnappedCards) >= 2 {
			uniqueVictims := make(map[string]bool)
			for _, victim := range targetStats.KidnappedCards {
				uniqueVictims[strings.ToLower(victim)] = true
			}
			if len(uniqueVictims) >= 2 {
				const rewardMicro = 7500 * 1000000
				l.playerBalances[senderWallet] += rewardMicro
				stats.ActiveJusticeMissionID = ""
				l.leaderboard[senderWallet] = stats
				l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", senderWallet, "ID: MISSION-013, Payout: 7500.00")
				l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> Hostage ring dismantled. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
				l.applyDynamicScalingLocked()
				go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
			}
		}
	}

	// PILLAR 3: Justice Mission Completion (MISSION-014)
	// High-Stakes Governor Audit: Flag a Regional Governor (Wanted 15+) holding 2+ hostages.
	if stats.ActiveJusticeMissionID == "MISSION-014" && targetStats.WantedLevel >= 15 {
		isGov := false
		if club, exists := l.clubs[targetStats.EmployerClubID]; exists && strings.EqualFold(club.OwnerWallet, targetWallet) {
			if l.clubService.IsClubRegionalLocked(l, club) {
				isGov = true
			}
		}
		if isGov && len(targetStats.KidnappedCards) >= 2 {
			const rewardMicro = 15000 * 1000000
			l.playerBalances[senderWallet] += rewardMicro
			stats.ActiveJusticeMissionID = ""
			l.leaderboard[senderWallet] = stats
			l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", senderWallet, "ID: MISSION-014, Payout: 15000.00")
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> Governor's hostage ring exposed. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
			l.applyDynamicScalingLocked()
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		}
	}

	// PILLAR 3: Justice Mission Completion (MISSION-015)
	// Hegemony Counter-Strike: Flag an outlaw (Wanted 30+) involved in extreme breaches.
	if stats.ActiveJusticeMissionID == "MISSION-015" && targetStats.WantedLevel >= 30 {
		executedContract018 := false
		for _, h := range targetStats.History {
			if h.UnderworldContractID == "CONTRACT-018" && h.IsUnderworldContractSuccess {
				executedContract018 = true
				break
			}
		}
		if executedContract018 {
			const rewardMicro = 20000 * 1000000
			l.playerBalances[senderWallet] += rewardMicro
			stats.ActiveJusticeMissionID = ""
			l.leaderboard[senderWallet] = stats
			l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", senderWallet, "ID: MISSION-015, Payout: 20000.00")
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> High-tier threat neutralized. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
			l.applyDynamicScalingLocked()
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		}
	}

	// PILLAR 3: Justice Mission Completion (MISSION-016)
	// Arena Center Hegemony Audit: Flag an outlaw (Wanted 35+) who owns the Arena Center.
	if stats.ActiveJusticeMissionID == "MISSION-016" && targetStats.WantedLevel >= 35 {
		var arenaCenterOwner string
		for _, club := range l.clubs {
			for _, t := range club.Territories {
				if t == "arena_center" {
					arenaCenterOwner = club.OwnerWallet
					break
				}
			}
			if arenaCenterOwner != "" {
				break
			}
		}

		if arenaCenterOwner != "" && strings.EqualFold(targetWallet, arenaCenterOwner) {
			const rewardMicro = 25000 * 1000000
			l.playerBalances[senderWallet] += rewardMicro
			stats.ActiveJusticeMissionID = ""
			l.leaderboard[senderWallet] = stats
			l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", senderWallet, "ID: MISSION-016, Payout: 25000.00")
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> Sector hegemony challenged. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
			l.applyDynamicScalingLocked()
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		}
	}

	// PILLAR 3: Justice Mission Completion (MISSION-017)
	// Chaos Containment Protocol: Flag an outlaw (Wanted 40+) involved in chaos-tier heists.
	if stats.ActiveJusticeMissionID == "MISSION-017" && targetStats.WantedLevel >= 40 {
		executedContract019 := false
		for _, h := range targetStats.History {
			if h.UnderworldContractID == "CONTRACT-019" && h.IsUnderworldContractSuccess {
				executedContract019 = true
				break
			}
		}
		if executedContract019 {
			const rewardMicro = 30000 * 1000000
			l.playerBalances[senderWallet] += rewardMicro
			stats.ActiveJusticeMissionID = ""
			l.leaderboard[senderWallet] = stats
			l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", senderWallet, "ID: MISSION-017, Payout: 30000.00")
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> Chaos contained. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
			l.applyDynamicScalingLocked()
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		}
	}

	// PILLAR 3: Justice Mission Completion (MISSION-018)
	// Criminal Employment Audit: Flag an outlaw (Wanted 20+) employing a 'Criminal'.
	if stats.ActiveJusticeMissionID == "MISSION-018" && targetStats.WantedLevel >= 20 {
		isCriminalEmployer := false
		for _, club := range l.clubs {
			// Check if target is the owner
			if strings.EqualFold(club.OwnerWallet, targetWallet) {
				// Check for 'Criminal' role in staff
				for _, role := range club.Staff {
					if strings.EqualFold(role, "Criminal") {
						isCriminalEmployer = true
						break
					}
				}
			}
			if isCriminalEmployer { break }
		}

		if isCriminalEmployer {
			const rewardMicro = 10000 * 1000000
			l.playerBalances[senderWallet] += rewardMicro
			stats.ActiveJusticeMissionID = ""
			l.leaderboard[senderWallet] = stats
			l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", senderWallet, "ID: MISSION-018, Payout: 10000.00")
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> Illicit workforce exposed. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
			l.applyDynamicScalingLocked()
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		}
	}

	// PILLAR 3: Justice Mission Completion (MISSION-019)
	// Financial Laundering Audit: Flag an outlaw (Wanted 25+) employing a 'Launderer'.
	if stats.ActiveJusticeMissionID == "MISSION-019" && targetStats.WantedLevel >= 25 {
		isLaundererEmployer := false
		for _, club := range l.clubs {
			// Check if target is the owner
			if strings.EqualFold(club.OwnerWallet, targetWallet) {
				// Check for 'Launderer' role in staff
				for _, role := range club.Staff {
					if strings.EqualFold(role, "Launderer") {
						isLaundererEmployer = true
						break
					}
				}
			}
			if isLaundererEmployer { break }
		}

		if isLaundererEmployer {
			const rewardMicro = 15000 * 1000000
			l.playerBalances[senderWallet] += rewardMicro
			stats.ActiveJusticeMissionID = ""
			l.leaderboard[senderWallet] = stats
			l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", senderWallet, "ID: MISSION-019, Payout: 15000.00")
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> Financial laundering link severed. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
			l.applyDynamicScalingLocked()
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		}
	}

	// PILLAR 3: Justice Mission Completion (MISSION-020)
	// Apex Predator Takedown: Flag an outlaw (Wanted 50+) who executed CONTRACT-018 and CONTRACT-019.
	if stats.ActiveJusticeMissionID == "MISSION-020" && targetStats.WantedLevel >= 50 {
		executedContract018 := false
		executedContract019 := false

		for _, h := range targetStats.History {
			if h.UnderworldContractID == "CONTRACT-018" && h.IsUnderworldContractSuccess {
				executedContract018 = true
			}
			if h.UnderworldContractID == "CONTRACT-019" && h.IsUnderworldContractSuccess {
				executedContract019 = true
			}
			if executedContract018 && executedContract019 { break }
		}

		if executedContract018 && executedContract019 {
			const rewardMicro = 40000 * 1000000
			l.playerBalances[senderWallet] += rewardMicro
			stats.ActiveJusticeMissionID = ""
			l.leaderboard[senderWallet] = stats
			l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", senderWallet, "ID: MISSION-020, Payout: 40000.00")
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> Apex predator neutralized. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
			l.applyDynamicScalingLocked()
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		}
	}

	// PILLAR 3: Justice Mission Completion (MISSION-021)
	// Hostage Syndicate Takedown: Flag an outlaw (Wanted 60+) holding 3+ hostages.
	if stats.ActiveJusticeMissionID == "MISSION-021" && targetStats.WantedLevel >= 60 {
		if len(targetStats.KidnappedCards) >= 3 {
			const rewardMicro = 50000 * 1000000
			l.playerBalances[senderWallet] += rewardMicro
			stats.ActiveJusticeMissionID = ""
			l.leaderboard[senderWallet] = stats
			l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", senderWallet, "ID: MISSION-021, Payout: 50000.00")
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> Syndicate dismantled. Order restored. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
			l.applyDynamicScalingLocked()
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		}
	}

	// PILLAR 3: Justice Mission Completion (MISSION-022)
	// Sovereign Hostage Recovery: Flag an outlaw (Wanted 70+) who executed CONTRACT-022.
	if stats.ActiveJusticeMissionID == "MISSION-022" && targetStats.WantedLevel >= 70 {
		executedContract022 := false
		for _, h := range targetStats.History {
			if h.UnderworldContractID == "CONTRACT-022" && h.IsUnderworldContractSuccess {
				executedContract022 = true
				break
			}
		}

		if executedContract022 {
			const rewardMicro = 60000 * 1000000
			l.playerBalances[senderWallet] += rewardMicro
			stats.ActiveJusticeMissionID = ""
			l.leaderboard[senderWallet] = stats
			l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", senderWallet, "ID: MISSION-022, Payout: 60000.00")
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> Sovereign threat neutralized. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
			l.applyDynamicScalingLocked()
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		}
	}

	// PILLAR 3: Justice Mission Completion (MISSION-023)
	// Ultimate Syndicate Dissolution: Flag an outlaw (Wanted 80+) who executed 021 AND 022.
	if stats.ActiveJusticeMissionID == "MISSION-023" && targetStats.WantedLevel >= 80 {
		executed021 := false
		executed022 := false
		for _, h := range targetStats.History {
			if h.UnderworldContractID == "CONTRACT-021" && h.IsUnderworldContractSuccess {
				executed021 = true
			}
			if h.UnderworldContractID == "CONTRACT-022" && h.IsUnderworldContractSuccess {
				executed022 = true
			}
			if executed021 && executed022 {
				break
			}
		}

		if executed021 && executed022 {
			const rewardMicro = 40000 * 1000000
			l.playerBalances[senderWallet] += rewardMicro
			stats.ActiveJusticeMissionID = ""
			l.leaderboard[senderWallet] = stats
			l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", senderWallet, "ID: MISSION-023, Payout: 75000.00")
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> The peak syndicate has been dissolved. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
			l.applyDynamicScalingLocked()
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		}
	}

	// PILLAR 3: Justice Mission Completion (MISSION-024)
	// Apex Scourge Neutralization: Flag an outlaw (Wanted 110+) who executed CONTRACT-023.
	if stats.ActiveJusticeMissionID == "MISSION-024" && targetStats.WantedLevel >= 110 {
		executedContract023 := false
		for _, h := range targetStats.History {
			if h.UnderworldContractID == "CONTRACT-023" && h.IsUnderworldContractSuccess {
				executedContract023 = true
				break
			}
		}

		if executedContract023 {
			const rewardMicro = 50000 * 1000000
			l.playerBalances[senderWallet] += rewardMicro
			stats.ActiveJusticeMissionID = ""
			l.leaderboard[senderWallet] = stats
			l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", senderWallet, "ID: MISSION-024, Payout: 100000.00")
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> Apex Scourge neutralized. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
			l.applyDynamicScalingLocked()
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		}
	}

	// PILLAR 3: Justice Mission Completion (MISSION-025)
	// Tax Haven Exposure: Flag an owner with > 10,000 VBV treasury.
	if stats.ActiveJusticeMissionID == "MISSION-025" && stats.JobRole == "Tax Auditor" {
		isTargetOwner := false
		var targetClub *Club
		for _, club := range l.clubs {
			if strings.EqualFold(club.OwnerWallet, targetWallet) {
				isTargetOwner = true
				targetClub = club
				break
			}
		}

		if isTargetOwner && targetClub.Treasury >= 10000.0 {
			// PILLAR 1: Tactical Defense Check.
			// Verify if the target organization has an active Audit Shield before allowing exposure.
			if exp, active := targetClub.BuffExpirations["AUDIT_SHIELD"]; active && time.Now().Before(exp) {
				l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ <b>TERMINAL ERROR:</b> Target organization is protected by an active Audit Shield."}`)})
				return
			}

			const rewardMicro = 25000 * 1000000
			l.playerBalances[senderWallet] += rewardMicro
			stats.ActiveJusticeMissionID = ""
			l.leaderboard[senderWallet] = stats
			l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", senderWallet, "ID: MISSION-025, Payout: 25000.00")
			l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> Tax haven exposed. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
			l.applyDynamicScalingLocked()
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		}
	}

	// PILLAR 3: Justice Mission Completion (MISSION-026)
	// Abduction Staff Audit: Flag an outlaw (Wanted 15+) employing a 'Kidnapper'.
	if stats.ActiveJusticeMissionID == "MISSION-026" && stats.JobRole == "Tax Auditor" {
		if targetStats.WantedLevel >= 15 {
			isTargetEmployer := false
			for _, club := range l.clubs {
				if strings.EqualFold(club.OwnerWallet, targetWallet) {
					for _, role := range club.Staff {
						if strings.EqualFold(role, "Kidnapper") {
							isTargetEmployer = true
							break
						}
					}
				}
				if isTargetEmployer {
					break
				}
			}

			if isTargetEmployer {
				const rewardMicro = 12000 * 1000000
				l.playerBalances[senderWallet] += rewardMicro
				stats.ActiveJusticeMissionID = ""
				l.leaderboard[senderWallet] = stats
				l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", senderWallet, "ID: MISSION-026, Payout: 12000.00")
				l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> Abduction staff exposed. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
				l.applyDynamicScalingLocked()
				go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
			}
		}
	}

	// PILLAR 3: Justice Mission Completion (MISSION-027)
	// Arena Center Breach Audit: Flag Wanted 40+ who heisted Arena Center.
	if stats.ActiveJusticeMissionID == "MISSION-027" && stats.JobRole == "Intel-Agent" {
		if targetStats.WantedLevel >= 40 {
			heistedArenaCenter := false
			for _, h := range targetStats.History {
				// CONTRACT-015 and CONTRACT-023 are Arena Center heists.
				if (h.UnderworldContractID == "CONTRACT-015" || h.UnderworldContractID == "CONTRACT-023") && h.IsUnderworldContractSuccess {
					heistedArenaCenter = true
					break
				}
			}

			if heistedArenaCenter {
				const rewardMicro = 15000 * 1000000
				l.playerBalances[senderWallet] += rewardMicro
				stats.ActiveJusticeMissionID = ""
				l.leaderboard[senderWallet] = stats
				l.logAdminAuditLocked("JUSTICE_MISSION_COMPLETED", senderWallet, "ID: MISSION-027, Payout: 15000.00")
				l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>MISSION COMPLETED:</b> District breach confirmed. Payout: %.2f $VBV."}`, float64(rewardMicro)/1000000.0))})
				l.applyDynamicScalingLocked()
				go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
			}
		}
	}

	adminMsg := fmt.Sprintf("⚖️ <b>JUSTICE TERMINAL:</b> %s has flagged high-infamy outlaw %s for priority review.", escapeHTML(l.oracleService.ResolveEnvoiName(l, senderWallet)), escapeHTML(l.oracleService.ResolveEnvoiName(l, targetWallet)))
	l.broadcastToAdmins(adminMsg)

	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"⚖️ <b>FLAGGED:</b> Outlaw signature submitted for priority administrative review."}`)})
}

// HandleAbortJusticeMission terminates the active law-enforcement objective.
// PILLAR 3: Justice Layer.
func (l *Lobby) HandleAbortJusticeMission(env *Envelope) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	wallet, ok := l.wallets[env.FromID]
	if !ok {
		return
	}

	l.ensurePlayerStatsMapsInitialized(wallet)
	stats := l.leaderboard[wallet]

	if stats.ActiveJusticeMissionID == "" {
		l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"❌ Error: No active mission to abort."}`)})
		return
	}

	missionID := stats.ActiveJusticeMissionID
	stats.ActiveJusticeMissionID = ""

	// Aborting reflects administrative withdrawal.
	stats.Reputation -= 25
	if stats.Reputation < 0 { stats.Reputation = 0 }

	l.leaderboard[wallet] = stats
	l.logAdminAuditLocked("JUSTICE_MISSION_ABORTED", wallet, fmt.Sprintf("ID: %s", missionID))
	l.sendToClientLocked(env.FromID, Envelope{Type: "admin_notification", Payload: json.RawMessage(`{"text":"⚖️ <b>MISSION ABORTED:</b> Dossier returned to archives. Reputation penalized."}`)})
	go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
}
