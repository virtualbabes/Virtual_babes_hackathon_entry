# §23.5 Bonded-Asset Registry — Implementation Plan (KEY 3.5, YOLO)

Status: PLAN ONLY — no code written. Author: nft-seduction-code-agent-2 (Plan/Recommend).
Trigger: SPEED-UP DIRECTIVE from aggregator (YOLO=true). Grounded in verified RAG
(RAG-design-gaps.md, SIGNED OFF) + live code grep + 3 answered calibration questions.

## Calibration answers (from aggregator — incorporated verbatim)
1. **Persistence:** atomic `bonded_assets.json`, mirroring `ItemRegistry.Save`/`Load`
   (`item_shop_archetype.go:230`/`:260` — `.tmp` swap + `os.Rename`, `IsNotExist` guard).
2. **Lifecycle hook:** rides `AuctionService.TransferBundleItems` (`auction_service.go:420`)
   as the single ledger-mutation entry point.
3. **MoodTag:** 3-value `int` enum — `0=NEUTRAL` (default), `1=BENEVOLENT`, `2=MALEVOLENT`.

## Scope (locked for this slice)
- `BondedAsset` type
- `bondedAssets map[string]*BondedAsset` on `Lobby`
- Atomic `Save`/`Load`
- Mint / Transfer / Burn lifecycle (transfer via `TransferBundleItems`)
- `MoodTag` field (3-value int enum)
- `ThemeBinding` scaffold (recorded fact ONLY — NO enforcement gate this slice)

OUT OF SCOPE (deferred, per signed-off build order): §27.6 lock *enforcement* in
`TransferBundleItems`, §27.8 `ModifyBondedAsset` owner+holder guard, §27.7.3
`BirthCertID`/`BlackMarketAdopted`. This slice unblocks them by providing the registry +
data, but does not wire the guards (those are later steps).

---

## GROUNDING (all grep-verified)
- `ItemRegistry` atomic Save/Load: `item_shop_archetype.go:230`/`:260`; `NewItemRegistry` `:82`;
  Lobby field `backend_types.go:472`; wired `server.go:252`. → mirror exactly.
- `TransferBundleItems(l, wallet, bundle, add)` at `auction_service.go:420`; assumes `l.mutex` held;
  mutates `l.leaderboard[wallet].Inventory`. Takes `CardBundle` (`common_types.go:493`:
  `CardID int`, `WeaponID string`, `FaceplateID string`). **No generic bonded-asset field yet.**
- `CreatorStoreProduct` PRESENT `creator_store_service.go:14`; `CreateProduct` `:101`;
  `ProcessSecondarySale` `:277` (royalty path). → reuse for storefront listing + §23 royalty.
- `BondedAsset` / `bondedAssets` / `MoodTag` / mint/transfer/burn: **0 matches** (confirmed
  design-only in signed-off RAG). This slice is net-new.
- `getDataPath` (`backend_types.go:24`) uses `l.DataDir` → use for the json path.
- Build gate: new file MUST open with `//go:build !js && !wasm`; verify `go build ./...`
  AND `go vet ./...` EXIT 0 (phantom-LSP rule).

---

## NEW FILE: `bonded_asset_registry.go`
Header: `//go:build !js && !wasm`
Imports: `encoding/json`, `fmt`, `log`, `os`, `sync`, `time`, `github.com/google/uuid`.

### Types
```go
const (
    MoodTagNeutral     = 0
    MoodTagBenevolent  = 1
    MoodTagMalevolent  = 2
    bondedAssetsFile   = "bonded_assets.json"
)

// BondedAsset is the §23.5 single-source record for every user-facing NFT
// (skins/backgrounds/buttons/appearances/audio + any custom asset).
type BondedAsset struct {
    AssetID      string    `json:"asset_id"`       // UUID, mint-time
    AssetType    string    `json:"asset_type"`     // "SKIN"|"BACKGROUND"|"BOARD"|"BUTTON"|"APPEARANCE"|"AUDIO"|"CUSTOM"
    Name         string    `json:"name"`           // user-supplied display name (§15.3 hub-lease)
    CreatorWallet string   `json:"creator_wallet"` // mint author (royalty recipient)
    OwnerWallet  string    `json:"owner_wallet"`   // current owner-of-record
    HolderWallet string    `json:"holder_wallet"`  // current equipping/holding wallet (session bind)
    MoodTag      int       `json:"mood_tag"`       // 0/1/2 (answered Q3)
    RoyaltyBps   uint64    `json:"royalty_bps"`    // ≤1000 (§16 Q22)
    CreatedAt    time.Time `json:"created_at"`
    // §27.7.3 provenance (scaffold fields, populated later):
    BirthCertID  string    `json:"birth_cert_id,omitempty"`
    Certified    bool      `json:"certified,omitempty"`
    BlackMarketAdopted bool `json:"black_market_adopted,omitempty"`
}

// ThemeBinding §27.6 scaffold — recorded fact only, NO enforcement this slice.
type ThemeBinding struct {
    Wallet  string `json:"wallet"`
    AssetID string `json:"asset_id"`
    Slot    string `json:"slot"`   // "SKIN"|"BACKGROUND"|...
    Locked  bool   `json:"locked"`
}

type BondedAssetRegistry struct {
    Mu      sync.RWMutex
    assets  map[string]*BondedAsset   // assetID -> asset
    bindings map[string]*ThemeBinding // assetID -> binding
}
```

### Constructors / lifecycle
- `NewBondedAssetRegistry() *BondedAssetRegistry` — init both maps.
- `MintBondedAsset(l *Lobby, creator, assetType, name string, moodTag int, royaltyBps uint64) (*BondedAsset, error)`
  - guard: creator != "" && name != "" (§15.3 naming rule); royaltyBps ≤ 1000.
  - `assetID = "BA-" + uuid.New().String()`; CreatorWallet=OwnerWallet=HolderWallet=creator;
    MoodTag clamped to {0,1,2}; RoyaltyBps stored.
  - under `reg.Mu.Lock()`: insert into `reg.assets`; under `l.mutex.Lock()`: no inventory change at
    mint (asset is a registry record, not yet in a bundle). Persist via `reg.Save(l)`.
  - also calls `l.creatorStore.CreateProduct(assetID, creator, name, assetType, price, ...)` so the
    asset is listable + royalty-tracked on the existing §23 storefront (reuse, not reinvent).
- `TransferBundleItems` EXTENSION (additive, `auction_service.go:420`):
  - add field `BondedAssetIDs []string` to `CardBundle` (`common_types.go:493`).
  - inside the existing function (mutex already held by caller): after the card/weapon/faceplate
    block, iterate `bundle.BondedAssetIDs`:
    - `add==true`: for each id, `reg.assets[id].HolderWallet = wallet` (transfer of holding/equip).
    - `add==false`: `reg.assets[id].HolderWallet = ""` (un-equip / consign-out).
  - **No Locked check here** (§27.6 enforcement deferred). This slice only records holder movement.
  - Caller-audit (Pitfall rule): `CardBundle` is a struct literal everywhere — adding an optional
    slice field breaks NO existing caller. Grep confirms no positional construction. Safe.
- `BurnBondedAsset(l *Lobby, wallet, assetID string) error`
  - require `OwnerWallet==wallet && HolderWallet==wallet` (burn needs both, mirrors §27.8 intent
    but is a *delete*, not a modify-guard — in scope as the burn primitive).
  - under `reg.Mu.Lock()`: delete `reg.assets[assetID]`; `reg.Save(l)`.
  - §27.7 ritual-burn path will call this + route offering via `RouteCriminalTax` later.

### Atomic persistence (mirror ItemRegistry exactly)
- `func (reg *BondedAssetRegistry) Save(l *Lobby) error`
  - `reg.Mu.RLock()`; snapshot slice; `RUnlock()`.
  - `json.MarshalIndent` → `l.getDataPath(bondedAssetsFile)` via `.tmp` + `os.Rename`
    (identical to `item_shop_archetype.go:242-254`).
- `func (reg *BondedAssetRegistry) Load(l *Lobby) error`
  - `os.ReadFile`; `IsNotExist` → fresh start (return nil); `Unmarshal`; under `reg.Mu.Lock()`
    rebuild map skipping nil/empty AssetID (identical to `:260-287`).

### ThemeBinding scaffold methods (no enforcement)
- `func (reg *BondedAssetRegistry) BindThemeAsset(wallet, assetID, slot string)`
  - `reg.bindings[assetID] = &ThemeBinding{Wallet:wallet, AssetID:assetID, Slot:slot, Locked:true}`.
  - NOTE: this records the lock; the actual block in `TransferBundleItems`/equip is DEFERRED to
    the §27.6 step. Documented inline so Act cannot mistake it for enforcement.
- `func (reg *BondedAssetRegistry) isThemeLocked(assetID string) bool` — getter (used by §27.6 later).

### Lobby field + wiring
- `backend_types.go` Lobby struct: add `bondedAssets *BondedAssetRegistry` (next to
  `itemRegistry` field `:472` and `themeEngine` field).
- `server.go` `newLobby()`: after `l.itemRegistry = NewItemRegistry()` (`:252`) add
  `l.bondedAssets = NewBondedAssetRegistry()`; after `l.itemRegistry.Load(l)` add
  `l.bondedAssets.Load(l)` (rehydrate on boot).

### Routes (server.go, minimal, build-green)
- `POST /api/assets/mint` (rate-limit `economy-tight`): body `{asset_type,name,mood_tag,royalty_bps}`;
  wallet via `extractWalletFromRequest(r)`; → `MintBondedAsset`.
- `GET  /api/assets` (rate-limit `standard`): `?wallet=` → collection (registry.GetCollection).
- `POST /api/assets/burn` (rate-limit `economy-tight`): body `{asset_id}`; wallet check; → `BurnBondedAsset`.
- Transfer reuses the EXISTING bundle-transfer flow (auction/secondary-sale) extended above — no
  new transfer route needed this slice.

### Concurrency (mirror ItemRegistry + rivalry pattern)
- `BondedAssetRegistry.Mu` (RWMutex) guards `assets`/`bindings`.
- Every `l.leaderboard`/`l.playerBalances`/Lobby-field read under `l.mutex.RLock()`/`RUnlock()`
  (or the already-held caller lock in `TransferBundleItems`).
- `Save` snapshots under `RLock`; `Load` rebuilds under `Lock`. No RLock-held-during-write hazard.

### Build / Verify (mandatory gate)
- `go build ./...` → EXIT 0
- `go vet ./...`   → EXIT 0   (vet catches data races build won't)
- Trust `go build`, not the LSP panel (phantom-LSP rule for this windows repo).

---

## Integration notes (follow-up, NOT this slice's code)
- **§27.1 unblock:** once `BondedAsset.MoodTag` exists, `ComputeThemeVector` (`theme_engine.go:126`)
  can read a wallet's held `BondedAssetIDs` and fold MoodTag into the tone vote (replacing the
  forced `MoodTag: 0` at `:222`). That edit is the natural next step but is OUT of this slice's
  scope per the directive; flag it as the immediate follow-up after sign-off.
- **§27.6:** wire `isThemeLocked(assetID)` into `TransferBundleItems` (block equip/transfer unless
  market-swap path) — deferred.
- **§27.7.3:** populate `BirthCertID`/`Certified` at `SpawnAI` + assert own-wallet mandate — deferred.

## Recommendation to aggregator / Brendan
- Approve this slice as-is (registry + lifecycle + MoodTag + ThemeBinding scaffold). It is the
  upstream unlock for §27.1/§27.6/§27.8 and is build-green by construction (mirrors the proven
  ItemRegistry atomic pattern). Keep §27.6 enforcement and §27.7.3 provenance as the NEXT slices.
