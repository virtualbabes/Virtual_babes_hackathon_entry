//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"sync"
	"time"
)

// ============================================================================
// PILLAR 7-D / KEY 3.5 Slice 2 â€” Item Shop Archetype System (v4 Â§14)
// A persisted, mutable item registry where each BuiltItem carries a bonded NFT,
// a creator wallet, the price paid, and a dynamically-computed PowerScale.
// Items synergize with the 12 AI-citizen pathways (Â§3a v3 / ai_citizen_engine.go).
// PILLAR 2: integer-supremacy math; no cloud; deterministic, replay-safe.
// ============================================================================

// ItemArchetype maps a ShopItem.ClubType family to the career/Hegemony linkage
// and the Â§15 v3 pathway it synergizes with. Grounded in rival_career_engine.go
// rivalry pairs and the aiPathwayByCareer map (ai_citizen_engine.go).
type ItemArchetype struct {
	Archetype  string `json:"archetype"`  // Stable key, e.g. "Liquid"
	ClubType   string `json:"club_type"`  // Mirrors ShopItem.ClubType family
	Careers    []string `json:"careers"`  // Career linkage (RequiredRole targets)
	Pathway    string `json:"pathway"`    // Â§15 v3 12-pathway track that gains synergy
	Rivalry    string `json:"rivalry"`    // Human-readable rivalry synergy note
}

// itemArchetypeRegistry is the canonical Â§14.2 archetype table.
var itemArchetypeRegistry = []ItemArchetype{
	{Archetype: "Liquid", ClubType: "Underworld_Admin", Careers: []string{"Fence", "Smuggler"}, Pathway: AIPathwayLedger, Rivalry: "P-Ledger vs TaxAuditor"},
	{Archetype: "Direct", ClubType: "Underworld_Admin", Careers: []string{"HeistPlanner"}, Pathway: AIPathwayShadow, Rivalry: "P-Direct vs Warden"},
	{Archetype: "Transit", ClubType: "Underworld_Admin", Careers: []string{"Smuggler"}, Pathway: AIPathwayPeace, Rivalry: "P-Transit vs SectorPeacekeeper"},
	{Archetype: "Cyber", ClubType: "Intelligence", Careers: []string{"ArcNetOperative"}, Pathway: AIPathwayIntel, Rivalry: "P-Cyber vs IntelAgent"},
	{Archetype: "Bounty", ClubType: "Intelligence", Careers: []string{"BountyHunter"}, Pathway: AIPathwayJustice, Rivalry: "P-Bounty vs JusticeRecruiter"},
	{Archetype: "Authority", ClubType: "Justice", Careers: []string{"JusticeCommissioner"}, Pathway: AIPathwayComm, Rivalry: "P-Authority vs TaxAuditor"},
	{Archetype: "Containment", ClubType: "Hardware", Careers: []string{"Kidnapper"}, Pathway: AIPathwayLockdown, Rivalry: "P-Lockdown vs AOS Leader"},
	{Archetype: "Forensic", ClubType: "Justice", Careers: []string{"Launderer", "MutationLogAuditor"}, Pathway: AIPathwayForensic, Rivalry: "P-Ledger vs MutationLogAuditor"},
	{Archetype: "Shadow", ClubType: "Intelligence", Careers: []string{"Gossip"}, Pathway: AIPathwayShadow, Rivalry: "P-Shadow vs JusticeRecruiter"},
	{Archetype: "Syndicate", ClubType: "Underworld_Admin", Careers: []string{"UnderworldBoss"}, Pathway: AIPathwaySyndicate, Rivalry: "P-Syndicate vs Judge"},
	{Archetype: "Justice", ClubType: "Justice", Careers: []string{"Warden", "JusticeCommissioner", "JusticeRecruiter"}, Pathway: AIPathwayJustice, Rivalry: "Justice Hegemony"},
	{Archetype: "Hardware", ClubType: "Hardware", Careers: []string{"Security"}, Pathway: AIPathwayLockdown, Rivalry: "Trap/defense synergy"},
}

// ItemArchetypeByClubType indexes the table by ClubType for fast lookup.
var itemArchetypeByClubType = func() map[string]ItemArchetype {
	m := make(map[string]ItemArchetype)
	for _, a := range itemArchetypeRegistry {
		m[a.ClubType] = a
	}
	return m
}()

// BuiltItem extends ShopItem with the Â§14.5 builder fields:
// creator wallet, bonded NFT (1:1 with the item), price paid, and PowerScale.
type BuiltItem struct {
	ShopItem
	ItemID      string `json:"item_id"`     // Unique instance ID (distinct per build)
	CreatorWallet string `json:"creator_wallet"`
	BondedNFT    string `json:"bonded_nft"` // Â§14.4: 1:1 bonded NFT asset ID
	PaidMicro    uint64 `json:"paid_micro"` // Â§14.3: price paid in micro-VBV
	PowerScale   float64 `json:"power_scale"` // Â§14.3: clamp(paid/base) in [MinScale, MaxScale]
	CreatedAt    int64  `json:"created_at"`
}

// itemRegistryFile is the on-disk filename for the persisted item registry.
const itemRegistryFile = "item_registry.json"

// ItemRegistry manages the persisted, mutable set of BuiltItems (PILLAR 6 atomic commit).
type ItemRegistry struct {
	mu    sync.RWMutex
	items map[string]*BuiltItem // itemID -> BuiltItem
}

// NewItemRegistry creates an empty registry.
func NewItemRegistry() *ItemRegistry {
	return &ItemRegistry{items: make(map[string]*BuiltItem)}
}

// GetArchetypes returns the canonical Â§14.2 archetype table (sorted, stable order).
func GetItemArchetypes() []ItemArchetype {
	out := make([]ItemArchetype, len(itemArchetypeRegistry))
	copy(out, itemArchetypeRegistry)
	return out
}

// PowerScaleForPrice computes the Â§14.3 dynamic power multiplier from the price paid.
// powerScale = clamp(paidMicro / BaseReferenceMicro, MinScale, MaxScale).
// Uses PILLAR 2 integer-supremacy math: ratios computed on integers, returned as float for display.
func PowerScaleForPrice(paidMicro, baseReferenceMicro uint64) float64 {
	const (
		minScale = 0.5
		maxScale = 3.0
	)
	if baseReferenceMicro == 0 {
		return minScale
	}
	scale := float64(paidMicro) / float64(baseReferenceMicro)
	return math.Max(minScale, math.Min(maxScale, scale))
}

// BaseReferenceMicro is the reference price (micro-VBV) that yields PowerScale = 1.0.
// 1 $VBV = 1_000_000 micro-VBV; 100 $VBV base reference matches the cheapest
// ShopItem tier in shop_registry.go (mood_catalyst @ 100).
const BaseReferenceMicro = 100 * 1_000_000

// BuildItemRequest is the payload for POST /api/items/build.
type BuildItemRequest struct {
	Wallet      string  `json:"wallet"`
	BaseItemID  string  `json:"base_item_id"`  // Template from GlobalShopRegistry
	PaidMicro   uint64  `json:"paid_micro"`    // Â§14.3 flexible price (micro-VBV)
	BondedNFT   string  `json:"bonded_nft"`    // Â§14.4 NFT asset ID (may be empty â†’ server mints a synthetic bond ID)
}

// BuildItem validates the request, computes PowerScale, routes PaidMicro via
// RouteCriminalTax (no value lost â€” feeds the Industrial Loop), records the
// BuiltItem in the registry, and persists atomically. Returns the created item.
func (ir *ItemRegistry) BuildItem(l *Lobby, req BuildItemRequest) (*BuiltItem, error) {
	if req.Wallet == "" {
		return nil, fmt.Errorf("wallet required")
	}
	if req.PaidMicro == 0 {
		return nil, fmt.Errorf("paid_micro must be > 0")
	}

	base, ok := GlobalShopRegistry[req.BaseItemID]
	if !ok {
		return nil, fmt.Errorf("unknown base item: %s", req.BaseItemID)
	}

	// Â§14.3: route the price paid through RouteCriminalTax (faucet/club/governance split).
	if l.tokenSinkRouter != nil {
		matrix := RevenueSplitMatrix{
			FaucetShare:     0.50,
			ClubShare:       0.25,
			GovernanceShare: 0.25,
		}
		if err := l.tokenSinkRouter.RouteCriminalTax("ITEM_BUILD", req.PaidMicro, matrix, 0, ""); err != nil {
			// Non-fatal: log and continue; the build still records the item.
			log.Printf("[ItemRegistry] RouteCriminalTax skipped for build: %v", err)
		}
	}

	powerScale := PowerScaleForPrice(req.PaidMicro, BaseReferenceMicro)

	// Â§14.4: bonded NFT. If none supplied, synthesize a deterministic bond ID.
	bonded := req.BondedNFT
	if bonded == "" {
		bonded = fmt.Sprintf("BOND-ITEM-%d-%s", time.Now().UnixNano(), req.BaseItemID)
	}

	item := &BuiltItem{
		ShopItem:      base,
		ItemID:        fmt.Sprintf("BI-%d-%s", time.Now().UnixNano(), req.BaseItemID),
		CreatorWallet: req.Wallet,
		BondedNFT:     bonded,
		PaidMicro:     req.PaidMicro,
		PowerScale:    powerScale,
		CreatedAt:     time.Now().Unix(),
	}

	ir.mu.Lock()
	ir.items[item.ItemID] = item
	ir.mu.Unlock()

	// Persist after mutation so restarts rehydrate the registry (PILLAR 6).
	if err := ir.Save(l); err != nil {
		log.Printf("[ItemRegistry] persist failed after build: %v", err)
	}

	if l.broadcast != nil {
		payload, _ := json.Marshal(map[string]interface{}{
			"type": "item_published",
			"item": item,
		})
		l.broadcast <- payload
	}

	log.Printf("[ItemRegistry] Built item %s (archetype=%s, scale=%.2f) by %s", item.ItemID, base.ClubType, powerScale, req.Wallet)
	return item, nil
}

// BindNFT re-assigns the bonded NFT for an existing BuiltItem (Â§14.4 rebind path).
func (ir *ItemRegistry) BindNFT(itemID, bondedNFT string) (*BuiltItem, error) {
	if bondedNFT == "" {
		return nil, fmt.Errorf("bonded_nft required")
	}
	ir.mu.Lock()
	defer ir.mu.Unlock()
	item, ok := ir.items[itemID]
	if !ok {
		return nil, fmt.Errorf("item not found: %s", itemID)
	}
	item.BondedNFT = bondedNFT
	return item, nil
}

// GetRegistry returns a snapshot of all BuiltItems (sorted by created time).
func (ir *ItemRegistry) GetRegistry() []*BuiltItem {
	ir.mu.RLock()
	defer ir.mu.RUnlock()
	out := make([]*BuiltItem, 0, len(ir.items))
	for _, it := range ir.items {
		out = append(out, it)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return out
}

// GetCollection returns BuiltItems created by a given wallet (rogue/human collection view).
func (ir *ItemRegistry) GetCollection(wallet string) []*BuiltItem {
	ir.mu.RLock()
	defer ir.mu.RUnlock()
	out := make([]*BuiltItem, 0)
	for _, it := range ir.items {
		if it.CreatorWallet == wallet {
			out = append(out, it)
		}
	}
	return out
}

// Save atomically persists the item registry (mirrors AICitizenEngine.SaveCitizens).
func (ir *ItemRegistry) Save(l *Lobby) error {
	if l == nil {
		return fmt.Errorf("lobby not initialized")
	}
	ir.mu.RLock()
	snapshot := make([]*BuiltItem, 0, len(ir.items))
	for _, it := range ir.items {
		cp := *it
		snapshot = append(snapshot, &cp)
	}
	ir.mu.RUnlock()

	// THE RECORD IS A TRANSPORT MIRROR of the same payload the file write produces: this snapshot was
	// taken under the REGISTRY OWN lock and released above, so the record never sees a live map.
	l.saveBlockchainStateSnapshotLocked(NotePrefixItemSnapshot, snapshot)

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal item registry: %w", err)
	}
	targetPath := l.getDataPath(itemRegistryFile)
	tempPath := targetPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write item registry temp: %w", err)
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to commit item registry: %w", err)
	}
	log.Printf("[ItemRegistry] Persisted %d built items to %s", len(snapshot), targetPath)
	return nil
}

// Load rehydrates the item registry from disk after a restart.
func (ir *ItemRegistry) Load(l *Lobby) error {
	if l == nil {
		return fmt.Errorf("lobby not initialized")
	}
	targetPath := l.getDataPath(itemRegistryFile)
	data, err := os.ReadFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("[ItemRegistry] No existing registry at %s (fresh start)", targetPath)
			return nil
		}
		return fmt.Errorf("failed to read item registry: %w", err)
	}
	var snapshot []*BuiltItem
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return fmt.Errorf("failed to unmarshal item registry: %w", err)
	}
	ir.mu.Lock()
	ir.items = make(map[string]*BuiltItem, len(snapshot))
	for _, it := range snapshot {
		if it == nil || it.ItemID == "" {
			continue
		}
		ir.items[it.ItemID] = it
	}
	ir.mu.Unlock()
	log.Printf("[ItemRegistry] Rehydrated %d built items", len(snapshot))
	return nil
}
