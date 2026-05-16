//go:build js && wasm
// +build js,wasm

package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall/js"
	"time"
)

// FaceplateStats defines the RPG modifiers provided by cosmetic items.
type FaceplateStats struct {
	MojoBonus    int
	CunningBonus int
}

// FaceplateRegistry maps legacy cosmetic IDs to functional social simulation bonuses.
var FaceplateRegistry = map[string]FaceplateStats{
	"faceplate_neon_vibe":   {MojoBonus: 15, CunningBonus: 5},
	"faceplate_shadow":      {MojoBonus: 5, CunningBonus: 20},
	"faceplate_governor":    {MojoBonus: 50, CunningBonus: 10},
	"faceplate_placeholder": {MojoBonus: 0, CunningBonus: 0},
}

// Club represents a player-owned organization mirrored for client-side state.
type Club struct {
	ID              string               `json:"id"`
	Name            string               `json:"name"`
	OwnerWallet     string               `json:"owner_wallet"`
	Type            string               `json:"type"`
	Territories     []string             `json:"territories"`
	RegionName      string               `json:"region_name,omitempty"`
	Treasury        float64              `json:"treasury"`
	Mojo            int                  `json:"club_mojo"`
	LastActivity    time.Time            `json:"last_activity"`
	LastHeistAt     time.Time            `json:"last_heist_at"` // Immersion Hook
	CreatedAt       time.Time            `json:"created_at"`
}

// TournamentMatch represents a single duel within the bracket.
// This mirrors the server's TournamentMatch for client-side display.
type TournamentMatch struct {
	ID     string `json:"id"`
	P1     string `json:"p1"` // Wallet Address
	P2     string `json:"p2"` // Wallet Address
	Winner string `json:"winner,omitempty"`
	Round  int    `json:"round"`
	ReceiptTxID string `json:"receipt_txid,omitempty"` // On-chain VBT_WIN receipt ID
}

// LinkedWallet represents a non-AVM wallet linked to a primary AVM wallet.
type LinkedWallet struct {
	Address   string    `json:"address"`
	Chain     string    `json:"chain"` // e.g., "ETH", "POLY", "SOL"
	Verified  bool      `json:"verified"`
	Timestamp time.Time `json:"timestamp"` // When it was linked/verified
}

// PlaystyleTendencies tracks behavioral metrics for narrative hooks.
type PlaystyleTendencies struct {
	Aggressiveness     float64            `json:"aggressiveness"`
	RiskTolerance      float64            `json:"risk_tolerance"`
	PreferredRules     map[string]float64 `json:"preferred_rules"`
	PreferredCardMoods map[string]float64 `json:"preferred_card_moods"`
}

// TournamentState tracks the progress of an automated event on the client side.
type TournamentState struct {
	Active       bool              `json:"active"`
	Matches      []TournamentMatch `json:"matches"`
	CurrentRound int               `json:"current_round"`
	Participants []string          `json:"participants"`
	Pot          float64           `json:"pot"`
	BuyInAmount  float64           `json:"buy_in_amount"`
	IsBuyInMode  bool              `json:"is_buy_in_mode"`
	OpenTime     time.Time         `json:"open_time"`
}

// WalletLinkInfo stores the primary AVM wallet and its linked non-AVM wallets.
type WalletLinkInfo struct {
	PrimaryAVMWallet string         `json:"primary_avm_wallet"`
	Linked           []LinkedWallet `json:"linked_wallets"`
}

// -----------------------------------------------------------------------------
// 1. ASSET EMBEDDING
// -----------------------------------------------------------------------------
//
// Assets are served statically from the /Public directory by the backend.

// -----------------------------------------------------------------------------
// 2. DATA VAULT (The Unified State Machine)
// -----------------------------------------------------------------------------

type Card struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	Power     [4]int  `json:"power"` // [Top, Right, Bottom, Left]
	Owner     int     `json:"owner"` // 0 for Player 1, 1 for Player 2
	Image     string  `json:"image"`
	Tier      string  `json:"tier"`       // Iron, Bronze, Gold, Diamond
	GlowColor string  `json:"glow_color"` // Hex color for UI effects
	IsCombo   bool    `json:"is_combo"`   // True if flipped during a chain reaction
	Rarity    float64 `json:"rarity"`     // Power multiplier based on supply
	Mood      string  `json:"mood"`       // Volatile, Serene, Spirited, Grounded
	Artifact  int     `json:"artifact"`   // Flat power bonus from equipped items
	Fatigue   int     `json:"fatigue"`    // 0-100, high fatigue penalizes power
	Loyalty   int     `json:"loyalty"`    // 0-100, high loyalty grants combo bonuses
}

type ActiveBuff struct {
	Type      string `json:"type"` // "mood_resist", "stat_boost", "workout"
	Value     int    `json:"value"`
	Remaining int    `json:"matches_left"`
}

type Player struct {
	ID              string       `json:"id"`
	Wallet          string       `json:"wallet"` // The connected blockchain address
	Decks           [4][]Card    `json:"decks"`  // 4 saved deck slots
	ActiveDeck      int          `json:"active_deck"`
	Ready           bool         `json:"ready"`
	Reputation      int          `json:"reputation"`
	WantedLevel     int          `json:"wanted_level"`
	GloatMessage    string       `json:"gloat_message"`
	AvatarURL       string       `json:"avatar_url"`
	Buffs           []ActiveBuff `json:"buffs"`
	AvatarBanNotice string       `json:"avatar_ban_notice"`
	EquippedFaceplate string       `json:"equipped_faceplate"` // For UI rendering
	Mojo            int          `json:"mojo"`             // Social standing for Club unlocks
	SocialRank      string       `json:"social_rank"`      // e.g., "Nobody", "Regular", "Icon"
	JobRole         string       `json:"job_role"`         // Manager, Security, Clerk, Freelancer
	EmployerClubID  string       `json:"employer_club_id"` // The club currently paying this user
	AuctionsWon     int          `json:"auctions_won"`
	Cunning         int          `json:"cunning"`
	Nurturing       int          `json:"nurturing"`
	Achievements    []string     `json:"achievements"`
	// EmployerClubID string             `json:"employer_club_id"` // The club currently paying this user
	JailedCards      map[int]string      `json:"jailed_cards"`       // CardID -> ClubID (cards currently in jail)
	KidnappedCards   map[int]string      `json:"kidnapped_cards"`    // CardID -> VictimWallet (cards player has kidnapped)
	HeldHostageCards map[int]string      `json:"held_hostage_cards"` // CardID -> KidnapperWallet (cards player has lost to kidnapping)
	FavoriteCardID   int                 `json:"favorite_card_id"`   // Added for Collective Intelligence
	RumorCount       int                 `json:"rumor_count"`        // Number of rumors spread by this player
	Portfolio        map[string]float64  `json:"portfolio"`
	Playstyle        PlaystyleTendencies `json:"playstyle"`
}

// GetEffectiveCunning returns base cunning plus cosmetic bonuses and infamy penalties.
func (p Player) GetEffectiveCunning() int {
	eff := p.Cunning
	if p.EquippedFaceplate != "" {
		if fp, exists := FaceplateRegistry[p.EquippedFaceplate]; exists {
			eff += fp.CunningBonus
		}
	}
	penalty := p.WantedLevel / 5
	if eff < penalty { return 0 }
	return eff - penalty
}

// Engine acts as the supreme state machine for the entire App
type Engine struct {
	Network   string
	Faucet    float64
	Phase     string          // "Lobby", "Setup", "TournamentLobby", "Active", "Finished"
	Rules     map[string]bool // Holds Custom Rules (Open, Same, Plus)
	Rewards   map[uint64]float64
	Inventory []Card // Global pool of cards to pick from

	// Asset Pools for Demo/AI
	DemoPool       []string
	AIPortraitPool []string // New pool for AI character avatars
	WitchPool      []string
	LadyPool       []string

	Players             [2]Player // 2P Lobby System
	Board               [9]*Card  // 3x3 Battle Grid
	BoardMoods          [9]string // Moods assigned to specific tiles
	Multiplayer         bool      // True if playing against a human, false for AI
	LocalPlayerIndex    int       // 0 for P1, 1 for P2
	Turn                int       // 0 for Player 1, 1 for Player 2
	Scores              [2]int    // Final scores [P1, P2]
	Maintenance         bool      // True if the arena is under maintenance
	TestingMode         bool      // If true, Player 1 always wins against AI
	IsAdmin             bool      // True if the connected wallet is an administrator
	Winner              int       // -1: None, 0: P1, 1: P2, 2: Draw
	AssetBase           string    // The CDN URL for sounds/images (e.g., GitHub Pages)
	ApiBase             string    // The backend API URL for production
	AmbientAudio        js.Value  // Current background music object
	CurrentAmbientTrack string
	ShowLeaderboard     bool                      // UI Toggle for Hall of Fame
	HardMode            bool                      // If true, AI uses tactical weighted scoring
	AIScore             int                       // Tactical value of the bot's intended move
	ServerLoad          int                       // Current active matches on the server
	SpecialFanfare      string                    // Archetype for specific win/loss tracks: "Emotional", "Witch"
	TerritoryID         string                    // The location of the current match
	ActiveItemBuffs     map[string]map[string]int // PlayerID -> ItemID -> MatchesRemaining
	P1RegionalBoost     bool                      // Global +5% power for district region owners
	P2RegionalBoost     bool                      // Global +5% power for district region owners
	VaultLow            bool                      // Warning flag for low faucet balance
	DeckRating          string                    // Current player's active deck rating (e.g., [A++])
	MasterVolume        float64                   // Global master volume (0.0 - 1.0)
	MusicVolume         float64                   // Music volume (0.0 - 1.0)
	SfxVolume           float64                   // Sound effects volume (0.0 - 1.0)
	Latency             int                       // WebSocket ping in milliseconds
	NetworkHealth       string                    // "Excellent", "Good", "Poor", "Critical"
	Tournament          TournamentState           `json:"tournament"` // Current bracket info
	Clubs               map[string]*Club          `json:"clubs"`      // Global club registry
	linkedWallets       map[string]WalletLinkInfo // Key: Primary AVM Wallet Address
	mutex               sync.RWMutex              // Protects Engine state from concurrent WASM/JS events
}

// Initialize the single source of truth
var Game = Engine{
	Network:          "VOI",
	Faucet:           1000.0,
	Phase:            "Lobby",
	Rules:            map[string]bool{"Open": true, "Power_copy": false, "Power_up": false},
	Rewards:          map[uint64]float64{40227315: 5.0},
	Players:          [2]Player{{ID: "Player 1"}, {ID: "Player 2"}},
	Board:            [9]*Card{},
	BoardMoods:       [9]string{},
	LocalPlayerIndex: 0,
	Multiplayer:      false,
	Turn:             0,
	Winner:           -1,
	DemoPool: []string{ // For AI card images, aligned with Public/Assets/Images/Cards/
		"Cards/Alana.webp",
		"Cards/Bella.webp",
		"Cards/Clohey.webp",
		"Cards/Ellie.webp",
		"Cards/Fran.webp",
		"Cards/Karren.webp",
		"Cards/Kat.webp",
		"Cards/Kay.webp",
		"Cards/Lucy.webp",
		"Cards/Pip.webp",
		"Cards/Roxy.webp",
		"Cards/Sally.webp",
		"Cards/Tammara.webp",
		"Cards/Taya.webp",
		"Cards/Triz.webp",
		"Cards/Xai.webp",
	},
	AIPortraitPool: []string{ // For AI avatar images, aligned with Public/Assets/Images/portraits/
		"portraits/Boss/The_architect/The_architect.webp",
		"portraits/cute/Angelina/Angelina.webp",
		"portraits/cute/Crypto_seraph/Crypto_seraph.webp",
		"portraits/Lady/Casino_sucubus/Casino_sucubus.webp",
		"portraits/Mini-Boss/Evil_angelina/Evil_angelina.webp",
		"portraits/Witch/Evil_jackpot_Jessica/Evil_jackpot_Jessica.webp",
		"portraits/Witch/Jackpot_jessica/Jackpot_jessica.webp",
	},
	TestingMode:    false,
	HardMode:       false,
	AIScore:        0,
	ServerLoad:     0,
	SpecialFanfare: "",
	TerritoryID:    "",
	VaultLow:       false,
	DeckRating:     "[Z]",
	Latency:        0,
	MasterVolume:   0.5, // Default
	MusicVolume:    0.5, // Default
	SfxVolume:      0.5, // Default
	NetworkHealth:  "Excellent",
	ApiBase:        "", // Default to relative
	AssetBase:      "", // Default to relative, can be set via SetAssetBase
	Clubs:          make(map[string]*Club),
}

// -----------------------------------------------------------------------------
// 3. FAUCET & NETWORK (The Ecosystem)
// -----------------------------------------------------------------------------

func connectWallet(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{"status": "error", "message": "No address provided"}
	}
	address := args[0].String()
	
	Game.mutex.Lock()
	Game.Players[0].Wallet = address
	Game.Players[0].ID = address[:6] + "..." + address[len(address)-4:]
	// Transition to Setup Phase for Avatar selection
	Game.Phase = "Setup"
	Game.mutex.Unlock()

	// UpdateAmbientMusic and PlaySound use their own internal logic or are safe
	// but we should ideally ensure Game state is consistent during calls.

	fmt.Printf("[ENGINE] Wallet %s Connected to: %s\n", address, Game.Network)
	PlaySound("click.mp3") // click.mp3 is lowercase in DIR.md
	UpdateAmbientMusic()
	return map[string]interface{}{"status": "success", "address": address, "network": Game.Network}
}

func disconnectWallet(this js.Value, args []js.Value) interface{} {
	Game.mutex.Lock()
	Game.Players[0].Wallet = ""
	Game.Players[0].ID = "Player 1"
	Game.IsAdmin = false
	Game.Players[0].Ready = false
	Game.mutex.Unlock()

	fmt.Println("[ENGINE] Wallet Disconnected.")
	PlaySound("click.mp3")
	UpdateAmbientMusic()
	return true
}

func toggleNetwork(this js.Value, args []js.Value) interface{} {
	Game.mutex.Lock()
	if Game.Network == "VOI" {
		Game.Network = "ALGO"
	} else {
		Game.Network = "VOI"
	}
	network := Game.Network
	Game.mutex.Unlock()
	fmt.Printf("[ENGINE] Network Switched to: %s\n", network)
	PlaySound("click.mp3")
	return network
}

// SetAvatar sets the player's profile image and transitions the game to Lobby phase
func SetAvatar(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	// The payload now includes an optional FavoriteCardID
	url := args[0].String()
	gloat := ""
	if len(args) > 1 {
		gloat = args[1].String()
	}
	notice := ""
	if len(args) > 2 {
		notice = args[2].String()
	}

	favoriteCardID := 0
	if len(args) > 3 {
		favoriteCardID = args[3].Int()
	}

	Game.mutex.Lock()
	Game.Players[0].AvatarURL = url
	Game.Players[0].GloatMessage = gloat
	Game.Players[0].AvatarBanNotice = notice
	Game.Players[0].FavoriteCardID = favoriteCardID
	Game.Phase = "Lobby"
	Game.mutex.Unlock()
	fmt.Println("[ENGINE] Avatar Set. Transitioning to LOBBY.")
	PlaySound("Gear_up_shot.mp3")
	return true
}

func SendReward(this js.Value, args []js.Value) interface{} {
	recipientAddr := Game.Players[0].Wallet
	if recipientAddr == "" {
		fmt.Println("[FAUCET ERROR] No wallet connected. Payout aborted.")
		return Game.Faucet
	}

	// Decrement locally for immediate UI feedback.
	Game.mutex.Lock()
	for _, amt := range Game.Rewards {
		Game.Faucet -= amt
	}
	Game.mutex.Unlock()

	// Payout is now handled by the secure backend to prevent mnemonic exposure.
	// We use a goroutine to trigger a JS fetch call so we don't block the WASM thread.
	go func() {
		fmt.Printf("[ENGINE] Requesting Payout for %s via Backend...\n", recipientAddr)

		clientID := js.Global().Get("myClientId").String()

		payload, _ := json.Marshal(map[string]interface{}{
			"recipient":    recipientAddr,
			"network":      Game.Network,
			"client_id":    clientID,
			"client_score": Game.Scores,
		})

		// Hand off to JavaScript to manage the UI feedback and Transaction lifecycle
		js.Global().Call("processRewardPayout", string(payload))
	}()

	return Game.Faucet
}

// SyncTournament synchronizes the local engine with the server's bracket state
func SyncTournament(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	// Convert JS object to JSON string for easy unmarshaling
	jsonStr := js.Global().Get("JSON").Call("stringify", args[0]).String()

	var ts TournamentState
	if err := json.Unmarshal([]byte(jsonStr), &ts); err != nil {
		fmt.Printf("[ENGINE ERROR] Tournament Sync failed: %v\n", err)
		return false
	}

	Game.mutex.Lock()
	Game.Tournament = ts
	Game.mutex.Unlock()
	fmt.Println("[ENGINE] Tournament State Synchronized.")
	return true
}

// -----------------------------------------------------------------------------
// 4. LOBBY & DECK LOGIC (The Preparation)
// -----------------------------------------------------------------------------

func ToggleRule(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	rule := args[0].String()

	Game.mutex.Lock()
	Game.Rules[rule] = !Game.Rules[rule]
	Game.mutex.Unlock()
	fmt.Printf("[LOBBY] Rule '%s' set to: %v\n", rule, Game.Rules[rule])
	PlaySound("click.mp3")
	return Game.Rules[rule]
}

// PlaySelectSound triggers the card selection audio feedback
func PlaySelectSound(this js.Value, args []js.Value) interface{} {
	PlaySound("Select-place-card.mp3")
	return nil
}

// SelectDeck changes the active deck slot for a player after checking reputation
func SelectDeck(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	slot := args[0].Int()
	if slot < 0 || slot > 3 {
		return false
	}

	// Reputation Thresholds
	thresholds := [4]int{0, 250, 600, 1000}
	if Game.Players[0].Reputation < thresholds[slot] {
		fmt.Printf("[ENGINE] Deck slot %d locked. Need %d Reputation.\n", slot+1, thresholds[slot])
		return false
	}

	Game.mutex.Lock()
	Game.Players[0].ActiveDeck = slot
	Game.mutex.Unlock()
	fmt.Printf("[ENGINE] Deck slot %d selected.\n", slot+1)
	PlaySound("Gear_up_shot.mp3")
	return true
}

// RemoveFromDeck clears a specific card from the current active deck
func RemoveFromDeck(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	cardIdx := args[0].Int()
	p := &Game.Players[0]
	Game.mutex.Lock()
	if cardIdx >= 0 && cardIdx < len(p.Decks[p.ActiveDeck]) {
		p.Decks[p.ActiveDeck] = append(p.Decks[p.ActiveDeck][:cardIdx], p.Decks[p.ActiveDeck][cardIdx+1:]...)
		Game.mutex.Unlock()
		PlaySound("click.mp3")
		return true
	}
	return false
}

func AddToDeck(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	cardID := args[0].Int()
	p := &Game.Players[0]

	Game.mutex.Lock()
	defer Game.mutex.Unlock()
	activeSlot := p.ActiveDeck

	// Guardrail: Max 5 cards per deck
	if len(p.Decks[activeSlot]) >= 5 {
		return false
	}

	// Prevent Duplicates: Check if the card is already in the player's deck
	for _, dc := range p.Decks[activeSlot] {
		if dc.ID == cardID {
			return false
		}
	}

	if c, found := findCard(cardID); found {
		c.Owner = 0
		p.Decks[activeSlot] = append(p.Decks[activeSlot], c)
		PlaySound("click.mp3")
		UpdateAmbientMusic()
		return true
	}
	return false
}

func SetPlayerReady(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	pIndex := args[0].Int()

	// HOTSEAT SIMULATOR: Auto-generate a deck for Player 2 if they are empty
	Game.mutex.Lock()
	if !Game.Multiplayer && pIndex == 1 && len(Game.Players[1].Decks[0]) == 0 {
		fmt.Println("[ENGINE] Generating Demo Deck for CPU...")
		Game.Players[1].ID = "Vbabe Bot"

		portraitPath := Game.AIPortraitPool[rand.Intn(len(Game.AIPortraitPool))]
		Game.Players[1].AvatarURL = Game.resolvePath("Images", portraitPath)

		// Map portrait folder to SpecialFanfare archetype for consistent character feedback
		if strings.Contains(portraitPath, "portraits/Witch/") {
			Game.SpecialFanfare = "Witch"
		} else if strings.Contains(portraitPath, "portraits/Boss/") || strings.Contains(portraitPath, "portraits/Mini-Boss/") {
			Game.SpecialFanfare = "Boss"
		} else if strings.Contains(portraitPath, "portraits/Lady/") {
			Game.SpecialFanfare = "Lady"
		} else {
			Game.SpecialFanfare = "cute"
		}

		Game.Players[1].ActiveDeck = 0
		for i := 0; i < 5; i++ {
			img := Game.DemoPool[i%len(Game.DemoPool)]
			simCard := GenerateCard(1000+i, fmt.Sprintf("Demo Babe %d", i+1), 60.0)
			simCard.Owner = 1
			simCard.Image = img
			Game.Players[1].Decks[0] = append(Game.Players[1].Decks[0], simCard)
		}
	}

	p := &Game.Players[pIndex]
	if len(p.Decks[p.ActiveDeck]) == 5 {
		Game.Players[pIndex].Ready = true
		fmt.Printf("[LOBBY] %s is READY.\n", Game.Players[pIndex].ID)
		PlaySound("click.mp3")
	}
	Game.mutex.Unlock()

	// Trigger UI Start Button if both are ready
	updateStartButton()
	return true
}

// SyncClubs ingests the global club registry from the server to support UI map immersion.
func SyncClubs(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	jsonStr := js.Global().Get("JSON").Call("stringify", args[0]).String()

	var clubs map[string]*Club
	if err := json.Unmarshal([]byte(jsonStr), &clubs); err != nil {
		return false
	}

	Game.mutex.Lock()
	Game.Clubs = clubs
	Game.mutex.Unlock()
	return true
}

// SyncPlayerStats updates reputation and other metrics for players in the lobby
func SyncPlayerStats(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return false
	}
	pIdx := args[0].Int()
	rep := args[1].Int()

	Game.mutex.Lock()
	if pIdx >= 0 && pIdx < 2 {
		Game.Players[pIdx].Reputation = rep
	}
	Game.mutex.Unlock()
	return true
}

// SyncFullProfile ingests a complete player profile from the server to ensure high-fidelity UI rendering.
func SyncFullProfile(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 { return false }
	data := args[0]

	Game.mutex.Lock()
	defer Game.mutex.Unlock()

	p := &Game.Players[0]
	p.Reputation = data.Get("reputation").Int()
	p.Mojo = data.Get("mojo").Int()
	p.SocialRank = data.Get("social_rank").String()
	p.JobRole = data.Get("job_role").String()
	p.EmployerClubID = data.Get("employer_id").String()
	p.WantedLevel = data.Get("wanted_level").Int()
	p.AuctionsWon = data.Get("auctions_won").Int()
	p.Cunning = data.Get("cunning").Int()
	p.Nurturing = data.Get("nurturing").Int()
	p.RumorCount = data.Get("rumor_count").Int()

	p.EquippedFaceplate = data.Get("equipped_faceplate").String()
	p.FavoriteCardID = data.Get("favorite_card_id").Int()
	// Sync Jailed Cards map
	p.JailedCards = make(map[int]string)
	jsJailed := data.Get("jailed_cards")
	if jsJailed.Type() == js.TypeObject {
		keys := js.Global().Get("Object").Call("keys", jsJailed)
		for i := 0; i < keys.Length(); i++ {
			k := keys.Index(i).String()
			id, _ := strconv.Atoi(k)
			p.JailedCards[id] = jsJailed.Get(k).String()
		}
	}

	// Sync Achievements slice
	p.Achievements = []string{}
	jsAch := data.Get("achievements")
	if jsAch.Type() == js.TypeObject && jsAch.Get("length").Truthy() {
		for i := 0; i < jsAch.Length(); i++ {
			p.Achievements = append(p.Achievements, jsAch.Index(i).String())
		}
	}

	p.KidnappedCards = make(map[int]string) // Reset ephemeral criminal tracking for sync
	p.HeldHostageCards = make(map[int]string)

	fmt.Printf("[ENGINE] Profile Synergized: %d Achievements, %d REP, Faceplate: %s, Fav Card: %d\n", len(p.Achievements), p.Reputation, p.EquippedFaceplate, p.FavoriteCardID)
	return true
}

// SyncPortfolio updates the local player's stock holdings
func SyncPortfolio(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	jsMap := args[0]

	Game.mutex.Lock()
	defer Game.mutex.Unlock()

	Game.Players[0].Portfolio = make(map[string]float64)
	keys := js.Global().Get("Object").Call("keys", jsMap)
	for i := 0; i < keys.Length(); i++ {
		k := keys.Index(i).String()
		Game.Players[0].Portfolio[k] = jsMap.Get(k).Float()
	}

	return true
}

// SyncPlaystyle updates the local player's behavioral analysis from the server
func SyncPlaystyle(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	jsonStr := js.Global().Get("JSON").Call("stringify", args[0]).String()

	var ps PlaystyleTendencies
	if err := json.Unmarshal([]byte(jsonStr), &ps); err != nil {
		return false
	}

	Game.mutex.Lock()
	Game.Players[0].Playstyle = ps
	Game.mutex.Unlock()
	return true
}

// SyncMove allows spectators or remote syncs to place a card for a specific player
func SyncMove(this js.Value, args []js.Value) interface{} {
	if len(args) < 3 {
		return false
	}
	gridIndex := args[0].Int()
	cardID := args[1].Int()
	pIdx := args[2].Int() // 0 for P1, 1 for P2

	// Reset combo flags for visual feedback
	for _, bc := range Game.Board {
		if bc != nil {
			bc.IsCombo = false
		}
	}

	c, found := findCard(cardID)
	if !found {
		// Fallback for missing metadata: provide a baseline card so the UI doesn't crash
		c = Card{ID: cardID, Name: "Syncing...", Power: [4]int{5, 5, 5, 5}, Owner: pIdx}
	}

	Game.mutex.Lock()
	c.Owner = pIdx
	tier, color, _ := calculateTier(Game.Players[pIdx].Reputation)
	c.Tier = tier
	c.GlowColor = color

	heapCard := new(Card)
	*heapCard = c
	Game.Board[gridIndex] = heapCard

	checkCaptures(heapCard, gridIndex)
	Game.Turn = 1 - pIdx // Keep local turn state in sync
	Game.mutex.Unlock()
	return true
}

// SetPhase allows the UI to manually transition the engine's state machine
func SetPhase(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	Game.mutex.Lock()
	Game.Phase = args[0].String()
	Game.mutex.Unlock()
	fmt.Printf("[ENGINE] Phase transitioned to: %s\n", Game.Phase)
	UpdateAmbientMusic()
	return true
}

// SyncServerLoad updates the current count of active matches from the lobby
func SyncServerLoad(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	Game.mutex.Lock()
	Game.ServerLoad = args[0].Int()
	Game.mutex.Unlock()
	return true
}

// SyncLatency updates the engine's network performance state from the JS WebSocket
func SyncLatency(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	ms := args[0].Int()
	Game.mutex.Lock()
	Game.Latency = ms

	if ms < 100 {
		Game.NetworkHealth = "Excellent"
	} else if ms < 300 {
		Game.NetworkHealth = "Good"
	} else if ms < 500 {
		Game.NetworkHealth = "Poor"
	} else {
		Game.NetworkHealth = "Critical"
	}
	Game.mutex.Unlock()
	return true
}

// startLatencyMonitor runs in a background goroutine to periodically trigger pings via JS
func startLatencyMonitor() {
	go func() {
		for {
			// Ping the server every 15 seconds to monitor connection health
			time.Sleep(15 * time.Second)
			js.Global().Call("sendPing")
		}
	}()
}

// AutoBuildDeck picks the 5 strongest cards from inventory, prioritizing the highest possible
// letter grade and maximizing the '+' count (number of cards in that tier).
func AutoBuildDeck(this js.Value, args []js.Value) interface{} {
	if len(Game.Inventory) < 5 {
		fmt.Println("[ENGINE] Not enough cards to auto-build.")
		return false
	}

	Game.mutex.RLock()
	// 1. Create a copy of inventory to sort
	tempInv := make([]Card, len(Game.Inventory))
	copy(tempInv, Game.Inventory)
	Game.mutex.RUnlock()

	// 2. Sort by Tactical Tiering
	// Primary: Highest Letter Grade (Bin) - e.g., A > B
	// Secondary: Power Sum * Scarcity (Battle Score)
	sort.Slice(tempInv, func(i, j int) bool {
		getMaxBin := func(card Card) int {
			maxP := 1
			for _, p := range card.Power {
				if p > maxP {
					maxP = p
				}
			}
			return (maxP - 1) / 100
		}

		binI := getMaxBin(tempInv[i])
		binJ := getMaxBin(tempInv[j])

		if binI != binJ {
			return binI > binJ
		}

		scoreI := float64(tempInv[i].Power[0]+tempInv[i].Power[1]+tempInv[i].Power[2]+tempInv[i].Power[3]) * tempInv[i].Rarity
		scoreJ := float64(tempInv[j].Power[0]+tempInv[j].Power[1]+tempInv[j].Power[2]+tempInv[j].Power[3]) * tempInv[j].Rarity
		return scoreI > scoreJ
	})

	// 3. Populate active deck
	Game.mutex.Lock()
	p := &Game.Players[0]
	p.Decks[p.ActiveDeck] = []Card{}
	for i := 0; i < 5; i++ {
		c := tempInv[i]
		c.Owner = 0
		p.Decks[p.ActiveDeck] = append(p.Decks[p.ActiveDeck], c)
	}
	Game.mutex.Unlock()

	fmt.Printf("[ENGINE] Auto-Built Deck %d (Max Tier Strategy).\n", p.ActiveDeck+1)
	PlaySound("Gear_up_shot.mp3")
	return true
}

// SetTestingMode toggles the 100% win rate against AI for rapid development
func SetTestingMode(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	Game.mutex.Lock()
	Game.TestingMode = args[0].Bool()
	Game.mutex.Unlock()
	fmt.Printf("[ENGINE] Testing Mode set to: %v\n", Game.TestingMode)
	return true
}

// SetHardMode toggles the tactical weighted scoring for the AI bot
func SetHardMode(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	Game.mutex.Lock()
	Game.HardMode = args[0].Bool()
	Game.mutex.Unlock()
	fmt.Printf("[ENGINE] Hard Mode AI: %v\n", Game.HardMode)
	return true
}

// updateStartButton re-evaluates if the "Start Battle" button should be enabled
func updateStartButton() {
	Game.mutex.RLock()
	defer Game.mutex.RUnlock()
	canStart := (Game.Players[0].Ready && Game.Players[1].Ready) && (!Game.Maintenance || Game.IsAdmin)
	js.Global().Call("highlightStartButton", canStart)
}

// SetAdminState allows manual override of admin status (e.g., from server or testing)
func SetAdminState(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	Game.mutex.Lock()
	Game.IsAdmin = args[0].Bool()
	Game.mutex.Unlock()
	fmt.Printf("[ENGINE] Admin State manually set to: %v\n", Game.IsAdmin)
	updateStartButton()
	return true
}

// SetMaintenanceState informs the engine about the arena's maintenance status
func SetMaintenanceState(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	Game.mutex.Lock()
	Game.Maintenance = args[0].Bool()
	Game.mutex.Unlock()
	fmt.Printf("[ENGINE] Maintenance Mode set to: %v\n", Game.Maintenance)
	updateStartButton()
	return true
}

// SyncOpponentProfile updates the metadata for an opponent during a multiplayer match
func SyncOpponentProfile(this js.Value, args []js.Value) interface{} {
	if len(args) < 3 {
		return false
	}
	pIdx := args[0].Int()
	avatar := args[1].String()
	gloat := args[2].String()

	Game.mutex.Lock()
	if pIdx >= 0 && pIdx < 2 {
		Game.Players[pIdx].AvatarURL = avatar
		Game.Players[pIdx].GloatMessage = gloat
	}
	Game.mutex.Unlock()
	return true
}

// SyncOpponentDeck populates a player's deck from a list of IDs (used for Multiplayer Handshakes)
func SyncOpponentDeck(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return false
	}
	pIndex := args[0].Int()
	idsVal := args[1]

	if idsVal.Type() != js.TypeObject {
		return false
	}

	// Clear existing deck for the specified player
	Game.mutex.Lock()
	Game.Players[pIndex].Decks[0] = []Card{}

	for i := 0; i < idsVal.Length(); i++ {
		id := idsVal.Index(i).Int()
		if c, found := findCard(id); found {
			c.Owner = pIndex
			Game.Players[pIndex].Decks[0] = append(Game.Players[pIndex].Decks[0], c)
		}
	}

	Game.Players[pIndex].Ready = true
	Game.mutex.Unlock()
	return true
}

// ForceActive allows spectators to bypass lobby requirements and enter combat mode
func ForceActive(this js.Value, args []js.Value) interface{} {
	Game.mutex.Lock()
	Game.Phase = "Active"
	Game.Board = [9]*Card{} // Clear local board for fresh sync
	Game.mutex.Unlock()
	fmt.Println("[ENGINE] Phase forced to ACTIVE for Spectating.")
	return true
}

// SyncVaultBalance updates the local faucet state from a server broadcast
func SyncVaultBalance(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	newBalance := args[0].Float()
	Game.mutex.Lock()
	Game.Faucet = newBalance
	Game.VaultLow = newBalance < 1000.0
	Game.mutex.Unlock()
	fmt.Printf("[ENGINE] Vault Balance Synced: %.2f $VBV\n", Game.Faucet)
	return true
}

// TriggerManualSync allows the UI to force a server-side re-sync of the player's on-chain stats.
func TriggerManualSync(this js.Value, args []js.Value) interface{} {
	wallet := Game.Players[0].Wallet
	if wallet == "" {
		fmt.Println("[ENGINE ERROR] Cannot trigger sync: No wallet connected.")
		return false
	}

	go func() {
		fmt.Printf("[ENGINE] Requesting manual stats re-sync for %s...\n", wallet)

		payload, _ := json.Marshal(map[string]string{"wallet": wallet})
		window := js.Global()

		// Construct fetch options
		options := window.Get("Object").New()
		options.Set("method", "POST")
		headers := window.Get("Object").New()
		headers.Set("Content-Type", "application/json")
		options.Set("headers", headers)
		options.Set("body", string(payload))

		// Execute fetch to the backend endpoint
		promise := window.Call("fetch", Game.ApiBase+"/api/re-sync-stats", options)

		success := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			fmt.Println("[ENGINE] Manual sync initiated successfully.")
			return nil
		})

		failure := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			fmt.Printf("[ENGINE ERROR] Manual sync request failed: %v\n", args[0])
			return nil
		})

		promise.Call("then", success).Call("catch", failure)
	}()

	return true
}

// SyncRewards updates the multi-reward registry from server
func SyncRewards(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	jsMap := args[0]
	Game.mutex.Lock()
	// Refactor: Atomic Map Update (Clearing and repopulating avoids panic)
	Game.Rewards = make(map[uint64]float64)
	keys := js.Global().Get("Object").Call("keys", jsMap)
	for i := 0; i < keys.Length(); i++ {
		k := keys.Index(i).String()
		id, _ := strconv.ParseUint(k, 10, 64)
		Game.Rewards[id] = jsMap.Get(k).Float() / 1000000.0 // Convert micro to base
	}
	Game.mutex.Unlock()

	fmt.Printf("[ENGINE] Multi-Rewards Synced: %d active assets\n", len(Game.Rewards))
	return true
}

// SyncRules updates the internal rule set from a server broadcast (Admin control)
func SyncRules(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	jsRules := args[0]
	if jsRules.Type() != js.TypeObject {
		return false
	}

	Game.mutex.Lock()
	Game.Rules["Open"] = jsRules.Get("Open").Bool()
	Game.Rules["Power_copy"] = jsRules.Get("Power_copy").Bool()
	Game.Rules["Power_up"] = jsRules.Get("Power_up").Bool()
	Game.Rules["Elemental_sync"] = jsRules.Get("Elemental_sync").Bool()
	Game.Rules["Fallen_penalty"] = jsRules.Get("Fallen_penalty").Bool()
	Game.Rules["Artifact_bonus"] = jsRules.Get("Artifact_bonus").Bool()
	Game.mutex.Unlock()

	fmt.Printf("[ENGINE] Rules Synchronized: %v\n", Game.Rules)
	return true
}

// SetBoardState bulk-loads the 3x3 grid and rules (used for Spectator Sync)
func SetBoardState(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	data := args[0]

	Game.mutex.Lock()
	defer Game.mutex.Unlock()
	jsBoard := data.Get("board")
	if jsBoard.Type() == js.TypeObject {
		for i := 0; i < 9; i++ {
			val := jsBoard.Index(i)
			if val.IsNull() || val.IsUndefined() {
				Game.Board[i] = nil
				continue
			}
			id := val.Get("id").Int()
			owner := val.Get("owner").Int()
			if c, found := findCard(id); found {
				c.Owner = owner
				// Create a new pointer instance for the board
				heapCard := new(Card)
				*heapCard = c
				Game.Board[i] = heapCard
			}
		}
	}

	// 3. Sync Board Moods (Authoritative Environmental Hazards)
	jsMoods := data.Get("board_moods")
	if jsMoods.Type() == js.TypeObject {
		for i := 0; i < 9; i++ {
			mood := jsMoods.Index(i)
			if mood.Type() == js.TypeString {
				Game.BoardMoods[i] = mood.String()
			} else {
				Game.BoardMoods[i] = "Neutral"
			}
		}
	}

	// 4. Sync Ruleset (Deterministic Alignment)
	jsRules := data.Get("rules")
	if jsRules.Type() == js.TypeObject {
		ruleKeys := []string{"Open", "Power_copy", "Power_up", "Elemental_sync", "Fallen_penalty", "Artifact_bonus"}
		for _, k := range ruleKeys {
			if r := jsRules.Get(k); !r.IsUndefined() {
				Game.Rules[k] = r.Bool()
			}
		}
	}

	// 5. Sync Tactical & Penalty Metadata for Spectator Accuracy
	if tid := data.Get("territory_id"); !tid.IsUndefined() {
		Game.TerritoryID = tid.String()
	}

	// 5.1 Sync Active Item Buffs
	jsActiveItemBuffs := data.Get("active_item_buffs")
	if jsActiveItemBuffs.Type() == js.TypeObject {
		Game.ActiveItemBuffs = make(map[string]map[string]int)
		js.Global().Get("Object").Call("keys", jsActiveItemBuffs).Call("forEach", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			playerID := args[0].String()
			Game.ActiveItemBuffs[playerID] = make(map[string]int)
			js.Global().Get("Object").Call("keys", jsActiveItemBuffs.Get(playerID)).Call("forEach", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				itemID := args[0].String()
				Game.ActiveItemBuffs[playerID][itemID] = jsActiveItemBuffs.Get(playerID).Get(itemID).Int()
				return nil
			}))
			return nil
		}))
	}
	if a1 := data.Get("p1_avatar"); !a1.IsUndefined() { Game.Players[0].AvatarURL = a1.String() }
	if g1 := data.Get("p1_gloat"); !g1.IsUndefined() { Game.Players[0].GloatMessage = g1.String() }
	if a2 := data.Get("p2_avatar"); !a2.IsUndefined() { Game.Players[1].AvatarURL = a2.String() }
	if g2 := data.Get("p2_gloat"); !g2.IsUndefined() { Game.Players[1].GloatMessage = g2.String() }

	if w1 := data.Get("p1_wanted_level"); !w1.IsUndefined() { Game.Players[0].WantedLevel = w1.Int() }
	if c1 := data.Get("p1_cunning"); !c1.IsUndefined() { Game.Players[0].Cunning = c1.Int() }
	if n1 := data.Get("p1_nurturing"); !n1.IsUndefined() { Game.Players[0].Nurturing = n1.Int() }
	
	if w2 := data.Get("p2_wanted_level"); !w2.IsUndefined() { Game.Players[1].WantedLevel = w2.Int() }
	if c2 := data.Get("p2_cunning"); !c2.IsUndefined() { Game.Players[1].Cunning = c2.Int() }
	if n2 := data.Get("p2_nurturing"); !n2.IsUndefined() { Game.Players[1].Nurturing = n2.Int() }

	jsScores := data.Get("scores")
	if jsScores.Type() == js.TypeObject && jsScores.Length() >= 2 {
		Game.Scores[0] = jsScores.Index(0).Int()
		Game.Scores[1] = jsScores.Index(1).Int()
	}

	// 6. Recalculate Turn based on board occupancy (Deterministic Inference)
	placedCount := 0
	for _, c := range Game.Board {
		if c != nil {
			placedCount++
		}
	}
	Game.Turn = placedCount % 2

	fmt.Printf("[ENGINE] Spectator state synchronized. Phase: %s, Territory: %s, Turn: %d\n", Game.Phase, Game.TerritoryID, Game.Turn)
	return true
}

// SyncMatchMetadata ingests tactical match parameters broadcasted during matchmaking.
func SyncMatchMetadata(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	data := args[0]

	Game.mutex.Lock()
	defer Game.mutex.Unlock()

	if t := data.Get("territory"); !t.IsUndefined() {
		Game.TerritoryID = t.String()
	}
	if b1 := data.Get("p1_boost"); !b1.IsUndefined() {
		Game.P1RegionalBoost = b1.Bool()
	}
	if b2 := data.Get("p2_boost"); !b2.IsUndefined() {
		Game.P2RegionalBoost = b2.Bool()
	}

	jsMoods := data.Get("moods")
	if jsMoods.Type() == js.TypeObject {
		for i := 0; i < 9; i++ {
			m := jsMoods.Index(i)
			if m.Type() == js.TypeString {
				Game.BoardMoods[i] = m.String()
			} else {
				Game.BoardMoods[i] = "Neutral"
			}
		}
	}

	fmt.Printf("[ENGINE] Match Metadata Synced. Territory: %s, P1Boost: %v, P2Boost: %v\n",
		Game.TerritoryID, Game.P1RegionalBoost, Game.P2RegionalBoost)
	return true
}

// findCard is a private helper to retrieve a card from the global inventory by ID
func findCard(id int) (Card, bool) {
	Game.mutex.RLock()
	defer Game.mutex.RUnlock()
	for _, c := range Game.Inventory {
		if c.ID == id {
			return c, true
		}
	}
	return Card{}, false
}

func ResetGame(this js.Value, args []js.Value) interface{} {
	Game.mutex.Lock()
	defer Game.mutex.Unlock()
	Game.Phase = "Lobby"
	Game.Board = [9]*Card{}
	Game.Multiplayer = false
	Game.Turn = 0
	Game.Winner = -1
	Game.Scores = [2]int{0, 0}
	Game.LocalPlayerIndex = 0
	Game.SpecialFanfare = ""

	// Clear player readiness and decks
	for i := range Game.Players {
		Game.Players[i].Ready = false
		for d := 0; d < 4; d++ {
			Game.Players[i].Decks[d] = []Card{}
		}
	}

	fmt.Println("[ENGINE] Game Reset to Lobby. State Cleared.")
	PlaySound("click.mp3")
	return true
}

// -----------------------------------------------------------------------------
// 5. THE WAR ROOM (The Combat Grid)
// -----------------------------------------------------------------------------

func StartMatch(this js.Value, args []js.Value) interface{} {
	if Game.Maintenance && !Game.IsAdmin {
		fmt.Println("[ENGINE ERROR] Cannot start match: Maintenance in progress.")
		return false
	}

	if !Game.Players[0].Ready || !Game.Players[1].Ready {
		return false
	}

	// Optional: Check if StartMatch(true) was passed for Multiplayer
	Game.Multiplayer = false
	if len(args) > 0 && args[0].Type() == js.TypeBoolean {
		Game.Multiplayer = args[0].Bool()
	}

	if Game.Multiplayer {
		Game.Players[1].ID = "Opponent"
	}

	Game.mutex.Lock()
	defer Game.mutex.Unlock()
	Game.Phase = "Active"
	Game.Turn = 0           // Player 1 starts
	Game.Board = [9]*Card{} // Clear board

	// Randomize Board Moods if Elemental Rule is active
	moodTypes := []string{"Volatile", "Serene", "Spirited", "Grounded", "Neutral"}
	for i := 0; i < 9; i++ {
		if Game.Rules["Elemental_sync"] && rand.Intn(10) > 6 {
			Game.BoardMoods[i] = moodTypes[rand.Intn(4)]
		} else {
			Game.BoardMoods[i] = "Neutral"
		}
	}

	Game.Winner = -1
	Game.Scores = [2]int{0, 0}

	UpdateAmbientMusic()
	fmt.Println("=================================")
	fmt.Printf(" BATTLE START! Rules: %v\n", Game.Rules)
	fmt.Println("=================================")
	PlaySound("flip.mp3")
	return true
}

func PlaceCard(this js.Value, args []js.Value) interface{} {
	if Game.Phase != "Active" || len(args) < 2 {
		return false
	}

	gridIndex := args[0].Int()
	cardID := args[1].Int()

	// Reset combo flags for all cards on board at the start of a move
	Game.mutex.Lock()
	for _, boardCard := range Game.Board {
		if boardCard != nil {
			boardCard.IsCombo = false
		}
	}
	Game.mutex.Unlock()
	
	// Reset AI score when the player takes their turn
	Game.AIScore = 0

	// Guardrails: Check if grid is valid and empty
	if gridIndex < 0 || gridIndex > 8 || Game.Board[gridIndex] != nil {
		fmt.Println("[BATTLE ERROR] Invalid or occupied slot.")
		return false
	}

	pIndex := Game.Turn

	p := &Game.Players[pIndex]
	for i, c := range p.Decks[p.ActiveDeck] {
		Game.mutex.Lock()
		defer Game.mutex.Unlock()
		if c.ID == cardID {
			Game.Board[gridIndex] = &p.Decks[p.ActiveDeck][i]

			// Apply Visual Tier Effects based on the player's current reputation
			tier, color, _ := calculateTier(Game.Players[pIndex].Reputation)
			Game.Board[gridIndex].Tier = tier
			Game.Board[gridIndex].GlowColor = color

			fmt.Printf("[BATTLE] %s placed %s at Grid %d\n", Game.Players[pIndex].ID, c.Name, gridIndex)

			checkCaptures(&p.Decks[p.ActiveDeck][i], gridIndex)
			PlaySound("Select-place-card.mp3")

			// Switch Turn
			if Game.Turn == 0 {
				Game.Turn = 1

				// TACTICAL REFACTOR: Implement 'Lag Guard' for AI triggers
				if !Game.Multiplayer {
					health := Game.NetworkHealth
					go func() {
						// If network is critical, wait for socket to settle
						if health == "Critical" {
							// Use state variables carefully across the boundary
							fmt.Printf("[LAG GUARD] Latency at %dms. Delaying AI response for stability...\n", Game.Latency)
							time.Sleep(2 * time.Second)
						}
						PerformAIMove()
					}()
				}
			} else {
				Game.Turn = 0
			}

			checkWinCondition()
			UpdateAmbientMusic()
			return true
		}
	}
	return false
}

// getEffectivePower applies Mood and Artifact modifiers to a card's side
func getEffectivePower(c *Card, sideIdx int, gridIdx int, pIdx int) int {
	base := c.Power[sideIdx] + c.Artifact

	// PILLAR 1: Regional Power Boost.
	// Global +5% power for district region owners.
	hasBoost := false
	if pIdx == 0 {
		hasBoost = Game.P1RegionalBoost
	} else {
		hasBoost = Game.P2RegionalBoost
	}
	if hasBoost {
		base += (base * 5) / 100
	}

	player := &Game.Players[pIdx]

	// Apply Wanted Level Penalty (Mitigated by Cunning)
	wantedPenalty := (player.WantedLevel * 5)
	// Cunning mitigates penalty: every 1 point of Cunning reduces penalty by 2
	mitigation := player.GetEffectiveCunning() * 2
	if mitigation > wantedPenalty { mitigation = wantedPenalty }
	base -= (wantedPenalty - mitigation)

	// Fatigue Penalty: -1 power per point above 50
	if c.Fatigue > 50 {
		fatiguePenalty := (c.Fatigue - 50)
		// Nurturing reduces fatigue impact: 1 power back per Nurturing point
		reduction := player.Nurturing
		if reduction > fatiguePenalty { reduction = fatiguePenalty }
		base -= (fatiguePenalty - reduction)
	}

	// Loyalty Bonus: +25 power for soul-bonded cards
	if c.Loyalty >= 100 {
		base += 25
	}

	if Game.Rules["Elemental_sync"] {
		tileMood := Game.BoardMoods[gridIdx]
		if tileMood != "Neutral" && c.Mood != "" && c.Mood != "Neutral" {
			moodWeaknesses := map[string]string{
				"Volatile": "Serene",
				"Serene":   "Spirited",
				"Spirited": "Grounded",
				"Grounded": "Volatile",
			}

			if c.Mood == tileMood {
				base += 50 // Match bonus: +0.5 Tier
			} else if moodWeaknesses[c.Mood] == tileMood {
				base -= 50 // Weakness penalty: -0.5 Tier
			}
		}
	}

	return base
}

// triggerCaptureParticlesInternal is the Go-internal call to JS
func triggerCaptureParticlesInternal(gridIndex int, owner int) {
	js.Global().Call("triggerCaptureParticles", gridIndex, owner)
}

// PlayCaptureEffect is the bridge function for js.FuncOf
func PlayCaptureEffect(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 { return nil }
	triggerCaptureParticlesInternal(args[0].Int(), args[1].Int())
	return nil
}

// checkCaptures applies the combat logic for a newly placed card
func checkCaptures(placedCard *Card, gridIndex int) int {
	totalFlips := 0
	// Define relative indices for neighbors and corresponding power indices
	// Power: [Top, Right, Bottom, Left]
	// {offset_from_current_index, placed_card_power_index, neighbor_card_power_index, boundary_check_function}
	neighbors := []struct {
		offset           int
		placedPowerIdx   int
		neighborPowerIdx int
		boundaryCheck    func(int) bool // Function to check if neighbor is within bounds
	}{
		{-3, 0, 2, func(idx int) bool { return idx >= 3 }},   // Top: placed.Top vs neighbor.Bottom
		{+1, 1, 3, func(idx int) bool { return idx%3 != 2 }}, // Right: placed.Right vs neighbor.Left
		{+3, 2, 0, func(idx int) bool { return idx <= 5 }},   // Bottom: placed.Bottom vs neighbor.Top
		{-1, 3, 1, func(idx int) bool { return idx%3 != 0 }}, // Left: placed.Left vs neighbor.Right
	}

	// Groups to track rule matches (Value/Sum -> list of neighbor indices)
	sameGroups := make(map[int][]int)
	plusGroups := make(map[int][]int)

	var comboQueue []int // Indices of cards flipped by Same/Plus to start combos

	for _, n := range neighbors {
		neighborIndex := gridIndex + n.offset

		// Check if the neighbor index is within board bounds and the slot is occupied
		if n.boundaryCheck(gridIndex) && Game.Board[neighborIndex] != nil {
			neighborCard := Game.Board[neighborIndex]
			placedPower := getEffectivePower(placedCard, n.placedPowerIdx, gridIndex, placedCard.Owner)
			neighborPower := getEffectivePower(neighborCard, n.neighborPowerIdx, neighborIndex, neighborCard.Owner)

			// 1. Prepare Power_copy Rule Data (Equality check)
			if Game.Rules["Power_copy"] && placedPower == neighborPower {
				sameGroups[placedPower] = append(sameGroups[placedPower], neighborIndex)
			}

			// 2. Prepare Power_up Rule Data (Sum check)
			if Game.Rules["Power_up"] {
				sum := placedPower + neighborPower
				plusGroups[sum] = append(plusGroups[sum], neighborIndex)
			}

			// 3. Basic Capture (Direct Power Comparison)
			if neighborCard.Owner != placedCard.Owner && placedPower > neighborPower {
				if flipCard(neighborIndex, placedCard.Owner, "BASIC") {
					totalFlips++
				}
			}
		}
	}

	// 4. Process Power_copy Rule triggers (Requires at least 2 matching sides)
	for _, indices := range sameGroups {
		if len(indices) >= 2 {
			for _, idx := range indices {
				if flipCard(idx, placedCard.Owner, "SAME") {
					totalFlips++
					comboQueue = append(comboQueue, idx)
				}
			}
		}
	}

	// 5. Process Power_up Rule triggers (Requires at least 2 matching sums)
	for _, indices := range plusGroups {
		if len(indices) >= 2 {
			for _, idx := range indices {
				if flipCard(idx, placedCard.Owner, "POWER_UP") {
					totalFlips++
					comboQueue = append(comboQueue, idx)
				}
			}
		}
	}

	// 6. Process Combo Chain (Recursive Basic Captures)
	for len(comboQueue) > 0 {
		currentIndex := comboQueue[0]
		comboQueue = comboQueue[1:]
		currentCard := Game.Board[currentIndex]

		// Define neighbors for the combo card
		comboNeighbors := []struct {
			offset           int
			placedPowerIdx   int
			neighborPowerIdx int
			boundaryCheck    func(int) bool
		}{
			{-3, 0, 2, func(idx int) bool { return idx >= 3 }},
			{+1, 1, 3, func(idx int) bool { return idx%3 != 2 }},
			{+3, 2, 0, func(idx int) bool { return idx <= 5 }},
			{-1, 3, 1, func(idx int) bool { return idx%3 != 0 }},
		}

		for _, cn := range comboNeighbors {
			nbIdx := currentIndex + cn.offset
			if cn.boundaryCheck(currentIndex) && Game.Board[nbIdx] != nil {
				neighbor := Game.Board[nbIdx]
				// Combo only triggers Basic Capture logic (Power Comparison)

				cPower := getEffectivePower(currentCard, cn.placedPowerIdx, currentIndex, currentCard.Owner)
				nPower := getEffectivePower(neighbor, cn.neighborPowerIdx, nbIdx, neighbor.Owner)

				if neighbor.Owner != currentCard.Owner && cPower > nPower {
					if flipCard(nbIdx, currentCard.Owner, "COMBO") {
						totalFlips++
						comboQueue = append(comboQueue, nbIdx)
					}
				}
			}
		}
	}
	return totalFlips
}

// flipCard handles owner change and rule-specific debuffs
func flipCard(idx int, newOwner int, reason string) bool {
	c := Game.Board[idx]
	if c == nil || c.Owner == newOwner {
		return false
	}

	c.Owner = newOwner
	c.IsCombo = (reason == "COMBO" || reason == "SAME" || reason == "POWER_UP")

	// Visual Feedback: Trigger capture sparks for all capture events
	triggerCaptureParticlesInternal(idx, newOwner)

	if Game.Rules["Fallen_penalty"] {
		c.Artifact -= 20 // Permanent debuff for being captured
	}
	return true
}

// simulateCapturesOnBoard is a helper to calculate score for a move on a given board.
// It's a slightly modified version of the main simulateCaptures logic,
// adapted to work on a passed board and player index.
func simulateCapturesOnBoard(board [9]*Card, placedCard *Card, gridIndex int, playerIndex int, rules map[string]bool) int {
	totalScore := 0
	flipped := make(map[int]bool) // Tracks cards that would be flipped in this simulation

	neighbors := []struct {
		offset           int
		placedPowerIdx   int
		neighborPowerIdx int
		boundaryCheck    func(int) bool
	}{
		{-3, 0, 2, func(idx int) bool { return idx >= 3 }},
		{+1, 1, 3, func(idx int) bool { return idx%3 != 2 }},
		{+3, 2, 0, func(idx int) bool { return idx <= 5 }},
		{-1, 3, 1, func(idx int) bool { return idx%3 != 0 }},
	}

	sameGroups := make(map[int][]int)
	plusGroups := make(map[int][]int)
	var comboQueue []int

	// These weights are for evaluating the *player's* potential, so they should reflect
	// how much the AI *doesn't* want the player to achieve these.
	// Using the HardMode weights for the player's potential makes sense.
	playerRuleTriggerWeight := 250
	playerRuleFlipWeight := 100
	playerComboFlipWeight := 60

	// 1. Initial Scan
	for _, n := range neighbors {
		neighborIndex := gridIndex + n.offset
		if n.boundaryCheck(gridIndex) && board[neighborIndex] != nil {
			neighborCard := board[neighborIndex]
			pPower := getEffectivePower(placedCard, n.placedPowerIdx, gridIndex, playerIndex)
			nPower := getEffectivePower(neighborCard, n.neighborPowerIdx, neighborIndex, neighborCard.Owner)

			if rules["Power_copy"] && pPower == nPower {
				sameGroups[pPower] = append(sameGroups[pPower], neighborIndex)
			}
			if rules["Power_up"] {
				sum := pPower + nPower
				plusGroups[sum] = append(plusGroups[sum], neighborIndex)
			}
			// Basic Capture (10 points per flip)
			// Check if it flips an *opponent's* card (from the perspective of playerIndex)
			if neighborCard.Owner != playerIndex && pPower > nPower {
				neighborCard.Owner = playerIndex // Simulation: update owner to allow combo pathing
				flipped[neighborIndex] = true
				totalScore += 10
			}
		}
	}

	// 2. Rules check
	for _, indices := range sameGroups {
		if len(indices) >= 2 {
			totalScore += playerRuleTriggerWeight
			for _, idx := range indices {
				// Check if it flips an *opponent's* card (from the perspective of playerIndex)
				if board[idx].Owner != playerIndex {
					board[idx].Owner = playerIndex
					flipped[idx] = true
					totalScore += playerRuleFlipWeight
					comboQueue = append(comboQueue, idx)
				}
			}
		}
	}
	for _, indices := range plusGroups {
		if len(indices) >= 2 {
			totalScore += playerRuleTriggerWeight
			for _, idx := range indices {
				// Check if it flips an *opponent's* card (from the perspective of playerIndex)
				if board[idx].Owner != playerIndex {
					board[idx].Owner = playerIndex
					flipped[idx] = true
					totalScore += playerRuleFlipWeight
					comboQueue = append(comboQueue, idx)
				}
			}
		}
	}

	// 3. Combo Chain Simulation
	for len(comboQueue) > 0 {
		currIdx := comboQueue[0]
		comboQueue = comboQueue[1:]
		currentCard := board[currIdx]

		for _, n := range neighbors {
			nbIdx := currIdx + n.offset
			if n.boundaryCheck(currIdx) && board[nbIdx] != nil {
				neighbor := board[nbIdx]
				// Combo only triggers Basic Capture logic (Power Comparison)
				// Check if it flips an *opponent's* card (from the perspective of playerIndex)
				if neighbor.Owner != playerIndex && !flipped[nbIdx] {
					if getEffectivePower(currentCard, n.placedPowerIdx, currIdx, playerIndex) > getEffectivePower(neighbor, n.neighborPowerIdx, nbIdx, neighbor.Owner) {
						neighbor.Owner = playerIndex
						flipped[nbIdx] = true
						totalScore += playerComboFlipWeight
						comboQueue = append(comboQueue, nbIdx)
					}
				}
			}
		}
	}
	return totalScore
}

// calculateMaxPlayerPotential simulates the best possible move for the player
// on their next turn given a board state.
func calculateMaxPlayerPotential(board [9]*Card, playerHand []Card, playerIndex int, rules map[string]bool) int {
	maxPotentialScore := 0
	emptySlots := []int{}
	for i, c := range board {
		if c == nil {
			emptySlots = append(emptySlots, i)
		}
	}

	if len(emptySlots) == 0 || len(playerHand) == 0 {
		return 0
	}

	for _, playerCard := range playerHand {
		for _, slot := range emptySlots {
			// Create a deep copy of the playerCard to avoid modifying the original in hand
			playerCardCopy := playerCard

			// Simulate the player placing their card on a temporary board
			simulatedBoard := [9]*Card{}
			for i, c := range board {
				if c != nil {
					// Deep copy the card on the board to avoid modifying the original Game.Board cards
					tempCard := *c
					simulatedBoard[i] = &tempCard
				}
			}
			simulatedBoard[slot] = &playerCardCopy // Place the player's card on the simulated board

			// Calculate the score the player would get from this move
			score := simulateCapturesOnBoard(simulatedBoard, &playerCardCopy, slot, playerIndex, rules)
			if score > maxPotentialScore {
				maxPotentialScore = score
			}
		}
	}
	return maxPotentialScore
}

// simulateCaptures calculates how many cards would be flipped without modifying the board
func simulateCaptures(placedCard *Card, gridIndex int) int {
	totalScore := 0
	flipped := make(map[int]bool)

	// Helper for checking neighbors (identical to checkCaptures logic)
	neighbors := []struct {
		offset           int
		placedPowerIdx   int
		neighborPowerIdx int
		boundaryCheck    func(int) bool
	}{
		{-3, 0, 2, func(idx int) bool { return idx >= 3 }},
		{+1, 1, 3, func(idx int) bool { return idx%3 != 2 }},
		{+3, 2, 0, func(idx int) bool { return idx <= 5 }},
		{-1, 3, 1, func(idx int) bool { return idx%3 != 0 }},
	}

	sameGroups := make(map[int][]int)
	plusGroups := make(map[int][]int)
	var comboQueue []int

	// Tactical Weights for Hard Mode
	ruleTriggerWeight := 50
	ruleFlipWeight := 20
	comboFlipWeight := 15

	if Game.HardMode {
		ruleTriggerWeight = 250 // Heavily prioritize setting up Same/Plus
		ruleFlipWeight = 100
		comboFlipWeight = 60
	}

	// 1. Initial Scan
	for _, n := range neighbors {
		neighborIndex := gridIndex + n.offset
		if n.boundaryCheck(gridIndex) && Game.Board[neighborIndex] != nil {
			neighborCard := Game.Board[neighborIndex]
			pPower := getEffectivePower(placedCard, n.placedPowerIdx, gridIndex, placedCard.Owner)
			nPower := getEffectivePower(neighborCard, n.neighborPowerIdx, neighborIndex, neighborCard.Owner)

			if Game.Rules["Power_copy"] && pPower == nPower {
				sameGroups[pPower] = append(sameGroups[pPower], neighborIndex)
			}
			if Game.Rules["Power_up"] {
				sum := pPower + nPower
				plusGroups[sum] = append(plusGroups[sum], neighborIndex)
			}
			// Basic Capture (10 points per flip)
			if neighborCard.Owner != placedCard.Owner && pPower > nPower {
				flipped[neighborIndex] = true
				totalScore += 10
			}
		}
	}

	// 2. Rules check
	for _, indices := range sameGroups {
		if len(indices) >= 2 {
			totalScore += ruleTriggerWeight // Rule trigger bonus (Tactical Priority)
			for _, idx := range indices {
				if Game.Board[idx].Owner != placedCard.Owner {
					flipped[idx] = true
					totalScore += ruleFlipWeight // Rule flip weight
					comboQueue = append(comboQueue, idx)
				}
			}
		}
	}
	for _, indices := range plusGroups {
		if len(indices) >= 2 {
			totalScore += ruleTriggerWeight // Rule trigger bonus
			for _, idx := range indices {
				if Game.Board[idx].Owner != placedCard.Owner {
					flipped[idx] = true
					totalScore += ruleFlipWeight // Rule flip weight
					comboQueue = append(comboQueue, idx)
				}
			}
		}
	}

	// 3. Combo Chain Simulation
	for len(comboQueue) > 0 {
		currIdx := comboQueue[0]
		comboQueue = comboQueue[1:]
		currCard := Game.Board[currIdx]

		for _, n := range neighbors {
			nbIdx := currIdx + n.offset
			if n.boundaryCheck(currIdx) && Game.Board[nbIdx] != nil {
				neighbor := Game.Board[nbIdx]
				
				// Simulation logic: treat 'flipped' cards as belonging to the attacker for the power check
				nOwner := neighbor.Owner
				if flipped[nbIdx] { nOwner = placedCard.Owner }

				if nOwner != placedCard.Owner && !flipped[nbIdx] {
					if getEffectivePower(currentCard, n.placedPowerIdx, currIdx, placedCard.Owner) > getEffectivePower(neighbor, n.neighborPowerIdx, nbIdx, nOwner) {
						flipped[nbIdx] = true
						totalScore += comboFlipWeight // Combo flip weight
						comboQueue = append(comboQueue, nbIdx)
					}
				}
			}
		}
	}

	if Game.HardMode {
		// Defensive penalty: avoid placing weak sides against open board slots
		for _, n := range neighbors {
			if n.boundaryCheck(gridIndex) && Game.Board[gridIndex+n.offset] == nil {
				power := getEffectivePower(placedCard, n.placedPowerIdx, gridIndex, placedCard.Owner)
				if power < 1500 { // Significant penalty for exposing sides lower than Level P (1500)
					totalScore -= (1500 - power) / 10
				}
			}
		}
		return totalScore
	}

	return len(flipped)
}

// SyncOpponentWanted updates the wanted level for a specific player (P1 or P2)
func SyncOpponentWanted(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return false
	}
	pIndex := args[0].Int()
	wantedLevel := args[1].Int()

	Game.mutex.Lock()
	if pIndex >= 0 && pIndex < 2 {
		Game.Players[pIndex].WantedLevel = wantedLevel
	}
	Game.mutex.Unlock()
	return true
}

// PerformAIMove implements the AI's decision-making logic.
func PerformAIMove() {
	Game.mutex.Lock()
	defer Game.mutex.Unlock()

	if Game.Phase != "Active" || Game.Turn != 1 {
		return
	}

	// AI Thinking Delay
	time.Sleep(time.Duration(rand.Intn(1000)+500) * time.Millisecond) // 0.5 to 1.5 seconds

	aiPlayer := &Game.Players[1]
	aiHand := aiPlayer.Decks[aiPlayer.ActiveDeck]

	bestScore := -1
	bestCardIdx := -1
	bestGridIdx := -1

	emptySlots := []int{}
	for i, c := range Game.Board {
		if c == nil {
			emptySlots = append(emptySlots, i)
		}
	}

	if len(emptySlots) == 0 || len(aiHand) == 0 {
		fmt.Println("[AI] No valid moves for AI.")
		return
	}

	// AI Strategy: Iterate through all possible moves and pick the one that maximizes flips
	for cardHandIdx, card := range aiHand {
		for _, gridIdx := range emptySlots {
			// Simulate the move without actually modifying the game board
			score := simulateCaptures(&card, gridIdx)

			// If HardMode is active, also consider the opponent's potential next move
			if Game.HardMode {
				// Create a temporary board state for opponent's turn simulation
				tempBoard := [9]*Card{}
				for i, c := range Game.Board {
					if c != nil {
						tempCard := *c
						tempBoard[i] = &tempCard
					}
				}
				tempCard := card // Copy the AI's card for the temporary board
				tempBoard[gridIdx] = &tempCard

				// Calculate opponent's hand (cards not yet on board)
				opponentHand := []Card{}
				placedCardIDs := make(map[int]bool)
				for _, bc := range tempBoard {
					if bc != nil {
						placedCardIDs[bc.ID] = true
					}
				}
				for _, c := range Game.Players[0].Decks[Game.Players[0].ActiveDeck] {
					if !placedCardIDs[c.ID] {
						opponentHand = append(opponentHand, c)
					}
				}

				// Calculate the maximum score the opponent could achieve after this AI move
				opponentPotential := calculateMaxPlayerPotential(tempBoard, opponentHand, 0, Game.Rules)

				// Adjust AI's score: maximize own score, minimize opponent's potential
				score = score - opponentPotential
			}

			if score > bestScore {
				bestScore = score
				bestCardIdx = cardHandIdx
				bestGridIdx = gridIdx
			}
		}
	}

	if bestCardIdx != -1 && bestGridIdx != -1 {
		chosenCard := aiHand[bestCardIdx]
		Game.Board[bestGridIdx] = &chosenCard
		chosenCard.Owner = 1

		// Apply Visual Tier Effects based on the player's current reputation
		tier, color, _ := calculateTier(Game.Players[1].Reputation)
		Game.Board[bestGridIdx].Tier = tier
		Game.Board[bestGridIdx].GlowColor = color

		fmt.Printf("[AI] %s placed %s at Grid %d (Score: %d)\n", aiPlayer.ID, chosenCard.Name, bestGridIdx, bestScore)
		Game.AIScore = bestScore // Store AI's chosen score for UI feedback

		checkCaptures(&chosenCard, bestGridIdx)
		PlaySound("Select-place-card.mp3")

		// Remove card from AI's hand
		aiPlayer.Decks[aiPlayer.ActiveDeck] = append(aiHand[:bestCardIdx], aiHand[bestCardIdx+1:]...)
		Game.Turn = 0 // Switch turn back to Player 1
		checkWinCondition()
	}
}

// -----------------------------------------------------------------------------
// 6. THE STATE EXPORTER (The Camera)
// -----------------------------------------------------------------------------

// GetGameState sends a secure snapshot of the vault to the JavaScript UI
func GetGameState(this js.Value, args []js.Value) interface{} {
	Game.mutex.RLock()
	defer Game.mutex.RUnlock()

	filter := "all"
	if len(args) > 0 && args[0].Type() == js.TypeString {
		filter = args[0].String()
	}

	state := make(map[string]interface{})

	// Profile & Stats
	if filter == "all" || filter == "profile" {
		state["reputation"] = Game.Players[0].Reputation
		state["mojo"] = Game.Players[0].Mojo
		state["social_rank"] = Game.Players[0].SocialRank
		state["job_role"] = Game.Players[0].JobRole
		state["employer_id"] = Game.Players[0].EmployerClubID
		state["wanted_level"] = Game.Players[0].WantedLevel
		state["auctions_won"] = Game.Players[0].AuctionsWon
		state["jailed_cards"] = Game.Players[0].JailedCards
		state["kidnapped_cards"] = Game.Players[0].KidnappedCards
		state["held_hostage_cards"] = Game.Players[0].HeldHostageCards
		state["rumor_count"] = Game.Players[0].RumorCount
		state["playstyle"] = Game.Players[0].Playstyle
		state["favorite_card_id"] = Game.Players[0].FavoriteCardID
		state["cunning"] = Game.Players[0].GetEffectiveCunning()
		state["nurturing"] = Game.Players[0].Nurturing
		state["achievements"] = Game.Players[0].Achievements
	}

	// Combat State
	if filter == "all" || filter == "combat" {
		state["phase"] = Game.Phase
		state["turn"] = Game.Turn
		state["board"] = Game.Board
		state["board_moods"] = Game.BoardMoods
		state["scores"] = Game.Scores
		state["winner"] = Game.Winner
		state["ai_score"] = Game.AIScore
		state["p1_avatar"] = Game.Players[0].AvatarURL
		state["p2_avatar"] = Game.Players[1].AvatarURL
		state["p1_gloat"] = Game.Players[0].GloatMessage
		state["p2_gloat"] = Game.Players[1].GloatMessage
		state["p1_avatar_notice"] = Game.Players[0].AvatarBanNotice
		state["p2_id"] = Game.Players[1].ID
		state["multiplayer"] = Game.Multiplayer
		state["special_fanfare"] = Game.SpecialFanfare
		state["territory_id"] = Game.TerritoryID
		state["active_item_buffs"] = Game.ActiveItemBuffs
		// Expose player-specific stats for accurate client-side power calculations in tooltips
		state["p1_wanted_level"] = Game.Players[0].WantedLevel
		state["p1_cunning"] = Game.Players[0].Cunning
		state["p1_nurturing"] = Game.Players[0].Nurturing
		state["p2_wanted_level"] = Game.Players[1].WantedLevel
		state["p2_cunning"] = Game.Players[1].Cunning
		state["p2_nurturing"] = Game.Players[1].Nurturing
		state["p1_regional_boost"] = Game.P1RegionalBoost
		state["p2_regional_boost"] = Game.P2RegionalBoost
		state["rules"] = Game.Rules
		state["local_player_index"] = Game.LocalPlayerIndex
	}

	// Economy
	if filter == "all" || filter == "economy" {
		state["rewards"] = Game.Rewards
		state["faucet"] = Game.Faucet
		state["vault_low"] = Game.VaultLow
		state["portfolio"] = Game.Players[0].Portfolio
	}

	// Player Assets
	if filter == "all" || filter == "inventory" {
		state["deck"] = Game.Players[Game.Turn].Decks[Game.Players[Game.Turn].ActiveDeck]
		state["inventory"] = Game.Inventory
		state["active_deck"] = Game.Players[0].ActiveDeck
		state["deck_rating"] = calculateDeckRating(Game.Players[0].Decks[Game.Players[0].ActiveDeck])
	}

	// System / Meta
	if filter == "all" || filter == "meta" {
		state["maintenance"] = Game.Maintenance
		state["testing_mode"] = Game.TestingMode
		state["is_admin"] = Game.IsAdmin
		state["show_leaderboard"] = Game.ShowLeaderboard
		state["tournament"] = Game.Tournament
		state["server_load"] = Game.ServerLoad
		state["latency"] = Game.Latency
		state["network_health"] = Game.NetworkHealth
		state["api_base"] = Game.ApiBase
		state["network"] = Game.Network
		state["clubs"] = Game.Clubs
		state["server_load_color"] = calculateLoadColor(Game.ServerLoad)
		state["master_volume"] = Game.MasterVolume
		state["music_volume"] = Game.MusicVolume
		state["sfx_volume"] = Game.SfxVolume
	}

	stateJSON, err := json.Marshal(state)
	if err != nil {
		fmt.Printf("[ENGINE ERROR] State serialization failed: %v\n", err)
		return nil
	}

	// Use browser's native JSON.parse to efficiently materialize the JS object
	return js.Global().Get("JSON").Call("parse", string(stateJSON))
}

// checkWinCondition checks if the board is full and determines the winner.
func checkWinCondition() {
	Game.mutex.Lock()
	defer Game.mutex.Unlock()

	full := true
	for _, slot := range Game.Board {
		if slot == nil {
			full = false
			break
		}
	}

	if full {
		p1Score, p2Score := 0, 0
		for _, card := range Game.Board {
			if card != nil {
				if card.Owner == 0 {
					p1Score++
				} else {
					p2Score++
				}
			}
		}
		Game.Scores = [2]int{p1Score, p2Score}
		Game.Phase = "Finished"
		if p1Score > p2Score {
			Game.Winner = 0
		} else if p2Score > p1Score {
			Game.Winner = 1
		} else {
			Game.Winner = 2
		}
	}
}

// -----------------------------------------------------------------------------
// 7. BROWSER BRIDGES & AUDIO
// -----------------------------------------------------------------------------

// calculateTier is an internal helper to determine ranking metadata
func calculateTier(rep int) (string, string, bool) {
	tier := "Iron"
	color := "#a19d94" // Iron Grey

	if rep >= 500 {
		tier = "Diamond"
		color = "#b9f2ff" // Diamond Blue
	} else if rep >= 300 {
		tier = "Gold"
		color = "#ffd700" // Classic Gold
	} else if rep >= 150 {
		tier = "Bronze"
		color = "#cd7f32" // Bronze Orange
	}

	return tier, color, rep >= 500
}

// calculateLoadColor determines the hex color based on the match count
func calculateLoadColor(load int) string {
	if load >= 25 {
		return "#ff0000" // Red (Heavy)
	} else if load >= 10 {
		return "#ffff00" // Yellow (Moderate)
	}
	return "#00ff00" // Green (Optimal)
}

// GetServerLoadColor returns the current load color to the UI
func GetServerLoadColor(this js.Value, args []js.Value) interface{} {
	color := calculateLoadColor(Game.ServerLoad)
	status := "Optimal"
	if Game.ServerLoad >= 25 {
		status = "Heavy"
	} else if Game.ServerLoad >= 10 {
		status = "Moderate"
	}
	return map[string]interface{}{"color": color, "status": status}
}

// GetTierInfo returns the tier name and thematic color for a given reputation score.
func GetTierInfo(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return nil
	}
	rep := args[0].Int()
	tier, color, bonus := calculateTier(rep)

	return map[string]interface{}{
		"tier":  tier,
		"color": color,
		"bonus": bonus,
	}
}

func ToggleLeaderboard(this js.Value, args []js.Value) interface{} {
	Game.ShowLeaderboard = !Game.ShowLeaderboard
	Game.mutex.Lock()
	fmt.Printf("[ENGINE] Leaderboard Visible: %v\n", Game.ShowLeaderboard)
	Game.mutex.Unlock()
	PlaySound("click.mp3")
	return Game.ShowLeaderboard
}

func UpdateAmbientMusic() {
	var track string
	var category string

	// 1. Determine Category based on Game State
	if Game.Players[0].Wallet == "" {
		category = "not_connected"
		track = "Not_connected_ambient" // Correct TitleCase
	} else if Game.Phase == "TournamentLobby" {
		category = "tournament_menu"
		// Use one of the high-intensity tournament tracks for the bracket view
		track = "Tournament_game_ambient" // Correct TitleCase
	} else if len(Game.Players[0].Decks[Game.Players[0].ActiveDeck]) < 5 {
		category = "unbuilt"
		track = "Unbuilt_deck_ambient" // Correct TitleCase
	} else if Game.Phase == "Active" {
		category = "match"
		matchPool := []string{
			"2_player_ambient_1", "2_player_ambient_2", "2_player_ambient_3",
			"quick_play_ambient_1", "quick_play_ambient_2", "quick_play_ambient_3",
			"Tournament_game_ambient", "Tournament_game_ambient_2", "Tournament_game_ambient3", "Tournament_game_ambient4", "Tournament_game_ambient5", // Correct TitleCase
		}
		track = matchPool[rand.Intn(len(matchPool))]
	} else {
		category = "menu"
		menuPool := []string{
			"ambient_menu_music_1", "ambient_menu_music_2", "ambient_menu_music_3", "ambient_menu_music_4",
		}
		track = menuPool[rand.Intn(len(menuPool))]
	}

	// 2. Only switch if the category or track has changed to prevent resetting audio on every UI click
	if Game.CurrentAmbientTrack == category && (category == "not_connected" || category == "unbuilt") {
		return
	}

	StopAmbient()
	Game.CurrentAmbientTrack = category
	PlayAmbient(track)
}

// SetMasterVolume updates the global master volume.
func SetMasterVolume(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		Game.MasterVolume = args[0].Float()
		UpdateAmbientMusic() // Re-apply volume to current track
	}
	return nil
}

// SetMusicVolume updates the music volume.
func SetMusicVolume(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		Game.MusicVolume = args[0].Float()
		UpdateAmbientMusic() // Re-apply volume to current track
	}
	return nil
}

// SetSfxVolume updates the sound effects volume.
func SetSfxVolume(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		Game.SfxVolume = args[0].Float()
	}
	return nil
}

func StopAmbient() {
	if Game.AmbientAudio.Type() == js.TypeObject {
		Game.AmbientAudio.Call("pause")
		Game.AmbientAudio.Set("currentTime", 0)
	}
}

// Global variable to store the current ambient audio element for volume control
var currentAmbientAudio js.Value

func PlayAmbient(path string) {
	fullPath := Game.resolvePath("Audio", path)
	audio := js.Global().Get("Audio").New(fullPath)
	if audio.Type() == js.TypeObject {
		audio.Set("loop", true)
		audio.Set("volume", 0.5)                                // Lower volume for background music
		audio.Set("volume", Game.MusicVolume*Game.MasterVolume) // Apply current volume settings
		currentAmbientAudio = audio                             // Store for later volume adjustments
		Game.AmbientAudio = audio                               // Keep for the old reference

		// Play requires a promise handle in modern browsers
		promise := audio.Call("play")
		if promise.Type() == js.TypeObject {
			promise.Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				fmt.Printf("[AUDIO] Ambient blocked by browser: %v\n", args[0])
				return nil
			}))
		}
		fmt.Printf("[AUDIO] Playing Ambient: %s\n", path)
	}
}

func SetAssetBase(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		Game.mutex.Lock()
		Game.AssetBase = args[0].String()
		Game.mutex.Unlock()
		fmt.Printf("[ENGINE] Asset Base URL set to: %s\n", Game.AssetBase)
	}
	return nil
}

func SetApiBase(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		Game.mutex.Lock()
		Game.ApiBase = args[0].String()
		Game.mutex.Unlock()
		fmt.Printf("[ENGINE] API Base URL set to: %s\n", Game.ApiBase)
	}
	return nil
}

// resolvePath unifies asset pathing, handling AssetBase and Category folders.
func (e *Engine) resolvePath(category string, subPath string) string {
	e.mutex.RLock()
	base := e.AssetBase
	e.mutex.RUnlock()

	// category: "Audio", "Images"
	cleanSub := strings.TrimPrefix(subPath, "Public/Assets/")
	cleanSub = strings.TrimPrefix(cleanSub, "Assets/")
	cleanSub = strings.TrimPrefix(cleanSub, category+"/")
	cleanSub = strings.TrimLeft(cleanSub, "/")

	// Handle Audio extension defaulting to .mp3 if no extension is present
	if category == "Audio" && !strings.Contains(cleanSub, ".") {
		cleanSub += ".mp3"
	}

	// Restored case sensitivity: Path must match DIR.md exactly
	resolved := fmt.Sprintf("%sAssets/%s/%s", base, category, cleanSub)
	// fmt.Printf("[ENGINE] Path Resolved: %s -> %s\n", subPath, resolved)
	return resolved
}

func PlaySound(name string) {
	audio := js.Global().Get("Audio").New(Game.resolvePath("Audio", name))
	if audio.Type() == js.TypeObject {
		audio.Set("volume", Game.SfxVolume*Game.MasterVolume) // Apply SFX volume
		audio.Call("play")
	}
}

func SetLocalPlayerIndex(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	idx := args[0].Int()
	if idx < 0 || idx > 1 {
		return false
	}
	Game.LocalPlayerIndex = idx
	fmt.Printf("[ENGINE] Local Player Index set to: %d\n", idx)
	return true
}

// ApplyArtifactToBoard adds a power bonus to a card already placed on the board
func ApplyArtifactToBoard(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return false
	}
	gridIdx := args[0].Int()
	bonus := args[1].Int()

	Game.mutex.Lock()
	defer Game.mutex.Unlock()

	if gridIdx < 0 || gridIdx >= 9 || Game.Board[gridIdx] == nil {
		return false
	}

	Game.Board[gridIdx].Artifact += bonus
	return true
}

func registerFunctions() {
	js.Global().Set("connectWallet", js.FuncOf(connectWallet))
	js.Global().Set("disconnectWallet", js.FuncOf(disconnectWallet))
	js.Global().Set("toggleNetwork", js.FuncOf(toggleNetwork))
	js.Global().Set("SetAvatar", js.FuncOf(SetAvatar))
	js.Global().Set("SendReward", js.FuncOf(SendReward))

	js.Global().Set("ToggleRule", js.FuncOf(ToggleRule))
	js.Global().Set("AddToDeck", js.FuncOf(AddToDeck))
	js.Global().Set("AutoBuildDeck", js.FuncOf(AutoBuildDeck))
	js.Global().Set("PlaySelectSound", js.FuncOf(PlaySelectSound))
	js.Global().Set("SelectDeck", js.FuncOf(SelectDeck))
	js.Global().Set("RemoveFromDeck", js.FuncOf(RemoveFromDeck))
	js.Global().Set("SyncOpponentDeck", js.FuncOf(SyncOpponentDeck))
	js.Global().Set("SyncOpponentProfile", js.FuncOf(SyncOpponentProfile))
	js.Global().Set("SetPlayerReady", js.FuncOf(SetPlayerReady))
	js.Global().Set("SyncFullProfile", js.FuncOf(SyncFullProfile))
	js.Global().Set("SyncPlaystyle", js.FuncOf(SyncPlaystyle))
	js.Global().Set("SyncOpponentWanted", js.FuncOf(SyncOpponentWanted))
	js.Global().Set("SyncPortfolio", js.FuncOf(SyncPortfolio))

	js.Global().Set("StartMatch", js.FuncOf(StartMatch))
	js.Global().Set("PlaceCard", js.FuncOf(PlaceCard))
	js.Global().Set("GetGameState", js.FuncOf(GetGameState)) // Expose the Camera
	js.Global().Set("SetAdminState", js.FuncOf(SetAdminState))
	js.Global().Set("SyncPlayerStats", js.FuncOf(SyncPlayerStats))
	js.Global().Set("SyncServerLoad", js.FuncOf(SyncServerLoad))
	js.Global().Set("SyncLatency", js.FuncOf(SyncLatency))
	js.Global().Set("GetLevelLabelForDisplay", js.FuncOf(GetLevelLabelForDisplay))
	js.Global().Set("TriggerManualSync", js.FuncOf(TriggerManualSync))
	js.Global().Set("SyncMatchMetadata", js.FuncOf(SyncMatchMetadata))
	js.Global().Set("SyncTournament", js.FuncOf(SyncTournament))
	js.Global().Set("GetTournamentArchiveBadge", js.FuncOf(GetTournamentArchiveBadge))
	js.Global().Set("SyncMove", js.FuncOf(SyncMove))
	js.Global().Set("SetPhase", js.FuncOf(SetPhase))
	js.Global().Set("GetServerLoadColor", js.FuncOf(GetServerLoadColor))
	js.Global().Set("SetTestingMode", js.FuncOf(SetTestingMode))
	js.Global().Set("SetHardMode", js.FuncOf(SetHardMode))
	js.Global().Set("GetTierInfo", js.FuncOf(GetTierInfo))
	js.Global().Set("SyncRules", js.FuncOf(SyncRules))
	js.Global().Set("SyncRewards", js.FuncOf(SyncRewards))
	js.Global().Set("SyncVaultBalance", js.FuncOf(SyncVaultBalance))
	js.Global().Set("SyncClubs", js.FuncOf(SyncClubs))
	js.Global().Set("SetMaintenanceState", js.FuncOf(SetMaintenanceState))
	js.Global().Set("ForceActive", js.FuncOf(ForceActive))
	js.Global().Set("SetBoardState", js.FuncOf(SetBoardState))
	js.Global().Set("ResetGame", js.FuncOf(ResetGame))
	js.Global().Set("SetMasterVolume", js.FuncOf(SetMasterVolume))
	js.Global().Set("SetMusicVolume", js.FuncOf(SetMusicVolume))
	js.Global().Set("SetSfxVolume", js.FuncOf(SetSfxVolume))
	js.Global().Set("SetAssetBase", js.FuncOf(SetAssetBase))
	js.Global().Set("SetApiBase", js.FuncOf(SetApiBase))
	js.Global().Set("SetLocalPlayerIndex", js.FuncOf(SetLocalPlayerIndex))
	js.Global().Set("ImportARC72Card", js.FuncOf(ImportARC72Card))
	js.Global().Set("ApplyArtifactToBoard", js.FuncOf(ApplyArtifactToBoard))
}

// GetTournamentArchiveBadge returns a stylized HTML badge based on verification status
func GetTournamentArchiveBadge(this js.Value, args []js.Value) interface{} {
	if len(args) < 4 { // Now expects 4 arguments: isVerified, links, receiptsVerified, payoutsHash
		return ""
	}
	isVerified := args[0].Bool()
	jsLinks := args[1] // This is a JS array
	receiptsVerified := args[2].Bool()
	payoutsHash := args[3].String()

	var links []string
	if jsLinks.Type() == js.TypeObject && jsLinks.Get("length").Truthy() {
		for i := 0; i < jsLinks.Length(); i++ {
			links = append(links, jsLinks.Index(i).String())
		}
	}

	tooltipText := ""
	if len(links) > 0 {
		tooltipText = "Blockchain Links:\\n" + strings.Join(links, "\\n")
	} else if isVerified {
		tooltipText = "Archive verified on-chain."
		if receiptsVerified {
			tooltipText = "Deep Archive: Checksum and Payout Receipts verified on-chain."
		}
	} else {
		tooltipText = "Data could not be fully verified or reconstructed."
	}

	if isVerified && receiptsVerified {
		return fmt.Sprintf(`<span class="verified-badge" title="%s" style="font-size: 0.7em; padding: 2px 6px; border: 1px solid var(--neon-cyan); color: var(--neon-cyan); border-radius: 4px; margin-left: 10px; background: rgba(0, 242, 254, 0.1); box-shadow: 0 0 10px rgba(0, 242, 254, 0.2); vertical-align: middle;">✓ RECEIPT VERIFIED</span>`, tooltipText)
	} else if isVerified {
		return fmt.Sprintf(`<span class="verified-badge" title="%s" style="font-size: 0.7em; padding: 2px 6px; border: 1px solid var(--neon-green); color: var(--neon-green); border-radius: 4px; margin-left: 10px; background: rgba(63, 185, 80, 0.1); box-shadow: 0 0 10px rgba(63, 185, 80, 0.2); vertical-align: middle;">✓ VERIFIED ARCHIVE</span>`, tooltipText)
	}
	return fmt.Sprintf(`<span style="font-size: 0.7em; padding: 2px 6px; border: 1px solid #ffa657; color: #ffa657; border-radius: 4px; margin-left: 10px; opacity: 0.8; background: rgba(255, 166, 87, 0.1); vertical-align: middle;" title="%s">⚠ PARTIAL DATA</span>`, tooltipText)
}

// ImportARC72Card validates raw JSON metadata and converts it into a playable Card
// It now fetches card details from the backend's centralized cache.
func ImportARC72Card(this js.Value, args []js.Value) interface{} { // Renamed to accept network
	if len(args) < 2 { // Now expects 2 arguments: tokenID and networkName
		fmt.Println("[ENGINE ERROR] ImportARC72Card: Token ID or network name not provided.")
		return false
	}
	tokenID := args[0].Int()
	networkName := args[1].String() // New argument: network name for the card

	Game.mutex.RLock()
	apiBase := Game.ApiBase
	Game.mutex.RUnlock()

	// Make a fetch call to the backend's /api/card-details endpoint
	go func() {
		url := fmt.Sprintf("%s/api/card-details?ids=%d&network=%s", apiBase, tokenID, networkName)
		window := js.Global()

		promise := window.Call("fetch", url)

		// Handle success
		success := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resp := args[0]
			if !resp.Get("ok").Bool() {
				fmt.Printf("[ENGINE ERROR] Failed to fetch card %d from backend: %s\n", tokenID, resp.Get("statusText").String())
				return nil
			}

			jsonPromise := resp.Call("json")
			jsonPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				cardsJS := args[0] // This will be a JS array of card objects
				if cardsJS.Length() == 0 {
					fmt.Printf("[ENGINE ERROR] Backend returned no data for card %d\n", tokenID)
					return nil
				}

				// Convert JS object to Go struct
				cardJSON := cardsJS.Index(0)
				var newCard Card
				newCard.ID = cardJSON.Get("id").Int()
				newCard.Name = cardJSON.Get("name").String()
				newCard.Image = cardJSON.Get("image").String()
				newCard.Rarity = cardJSON.Get("rarity").Float()

				// Extract power values with safety checks
				jsPower := cardJSON.Get("power")
				if jsPower.Type() == js.TypeObject && jsPower.Get("length").Int() >= 4 {
					for i := 0; i < 4; i++ {
						newCard.Power[i] = jsPower.Index(i).Int()
					}
				}

				// Set game-state specific defaults
				newCard.Owner = -1 // Not owned yet
				newCard.Tier = "Iron"
				newCard.GlowColor = "#a19d94"
				newCard.IsCombo = false
				newCard.Image = Game.resolvePath("Images", newCard.Image)

				Game.mutex.Lock()
				Game.Inventory = append(Game.Inventory, newCard)

				// CORRECTION MECHANISM: Scan board for "Syncing..." dummies and update them
				for _, boardCard := range Game.Board {
					if boardCard != nil && boardCard.ID == newCard.ID && boardCard.Name == "Syncing..." {
						boardCard.Name = newCard.Name
						boardCard.Power = newCard.Power
						boardCard.Image = newCard.Image
						boardCard.Rarity = newCard.Rarity
						fmt.Printf("[ENGINE] Corrected dummy card on board: %s\n", boardCard.Name)
					}
				}
				Game.mutex.Unlock()

				fmt.Printf("[ENGINE] Imported %s | ID: %d | Power: %v | Rarity: %.2f\n", newCard.Name, newCard.ID, newCard.Power, newCard.Rarity)
				js.Global().Call("syncUI") // Trigger UI update
				return nil
			}))
			return nil
		})

		// Handle error
		failure := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			err := args[0]
			fmt.Printf("[ENGINE ERROR] Fetching card %d from backend failed: %v\n", tokenID, err.String())
			return nil
		})

		promise.Call("then", success).Call("catch", failure)
	}()

	return true // Return immediately, actual import happens asynchronously
}

// Helper to generate test inventory
func GenerateCard(id int, name string, price float64) Card {
	base := int(price / 10)
	if base < 2 {
		base = 2
	}
	if base > 9 {
		base = 9
	}
	return Card{ID: id, Name: name, Power: [4]int{base, base - 1, base + 1, base}, Image: fmt.Sprintf("Cards/%d.webp", id), Rarity: 1.0}
}

// getLevelFromValue maps a power value (1-2600) to an A-Z letter grade.
func getLevelFromValue(val int) string {
	alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	// Corrected Mapping: 1-100 = Z, 101-200 = Y, ..., 2501-2600 = A
	// Bin 0: 1-100 (Z) -> index 25
	// Bin 1: 101-200 (Y) -> index 24
	// Bin 25: 2501-2600 (A) -> index 0
	bin := (val - 1) / 100
	if bin < 0 {
		bin = 0
	} // Handle 0 or negative values as lowest tier
	if bin > 25 {
		bin = 25
	} // Handle values > 2600 as highest tier
	return string(alphabet[25-bin])
}

// GetLevelLabelForDisplay is a bridge function to expose getLevelFromValue to JS.
func GetLevelLabelForDisplay(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return "Z" // Default for invalid input
	}
	val := args[0].Int()
	return getLevelFromValue(val)
}

// calculateDeckRating computes the [Letter++] rating for a given deck.
func calculateDeckRating(deck []Card) string {
	if len(deck) == 0 {
		return "[Z]"
	}

	maxBin := -1
	// 1. Find the highest card tier (bin) in the deck
	for _, card := range deck {
		highestPower := 0
		for _, p := range card.Power {
			if p > highestPower {
				highestPower = p
			}
		}
		bin := (highestPower - 1) / 100
		if bin < 0 {
			bin = 0
		}
		if bin > maxBin {
			maxBin = bin
		}
	}

	if maxBin == -1 {
		return "[Z]"
	} // Should not happen with non-empty deck

	// 2. Map maxBin to Letter
	baseLetter := getLevelFromValue((maxBin * 100) + 1) // Get the letter for the start of the bin

	// 3. Count how many cards share this highest tier
	plusCount := 0
	for _, card := range deck {
		highestPower := 0
		for _, p := range card.Power {
			if p > highestPower {
				highestPower = p
			}
		}
		bin := (highestPower - 1) / 100
		if bin == maxBin {
			plusCount++
		}
	}

	// 4. Construct Suffix
	suffix := ""
	for i := 0; i < plusCount; i++ {
		suffix += "+"
	}

	return fmt.Sprintf("[%s%s]", baseLetter, suffix)
}

func main() {
	wait := make(chan struct{})

	fmt.Println("-------------------------------------------------")
	fmt.Println(" NFT Seduction WASM Engine: SYNC ONLINE          ")
	fmt.Println(" Camera Exporter & AI Simulation Active          ")
	fmt.Println("-------------------------------------------------")

	// Seed Global Inventory with Demo Assets
	for i, path := range Game.DemoPool {
		if i >= 5 {
			break
		} // Seed first 5 as inventory
		c := GenerateCard(100+i, fmt.Sprintf("Babe %d", i+1), 50.0+float64(i*10))
		c.Image = path
		Game.Inventory = append(Game.Inventory, c)
	}

	registerFunctions()
	<-wait
}
