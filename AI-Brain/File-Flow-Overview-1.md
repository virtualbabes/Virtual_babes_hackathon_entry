# File-Flow Overview â€” NFT-Seduction (AUTHORITATIVE, corrected 2026-09-09)

> **STATUS: AUTHORITATIVE CORE MAP.** Rebuilt 2026-09-09 from a **full read + audit of all 84 backend `.go` files** and the live route surface (300 `mux.HandleFunc` registrations in `server_main.go` â†’ ~298 unique `/api/*` paths). This document supersedes the archived `archive/docs-2026-09-07/File-Flow-Overview-1.md` (V2.1, previously marked SUPERSEDED and containing falsified "completed" claims â€” see Â§10).
>
> **Authority rule (Repository Truth):** The live repository overrides any prior doc. Where an old doc or Session-Handoff claimed a system was "completed" but the code shows it is partial or broken, this document states the **code truth** and flags it.

---

## 0. Executive Summary

- **Language:** Go backend (84 files, ~200 KB) + deterministic WASM game engine + ~173 JS modules + 105 SCSS partials.
- **Build:** Dual-target. Client = `GOOS=js GOARCH=wasm go build ./...` (shell default). Server = `GOOS=linux GOARCH=amd64 go build ./...` (Dockerfile `golang:1.24-alpine` â†’ `server-bin`). A bare `go build ./...` compiles **only the client** and is a FALSE-GREEN for the server. **Both targets are currently GREEN** (verified 2026-09-09).
- **Server default port:** `8088` (`server_main.go`); dev launcher (`launch_dev_server.ps1`) sets `PORT=8090`.
- **God-object:** `*Lobby` in `lobby_manager.go` (5021 lines) owns almost all mutable shared state and is the hub every service is wired into via `newLobby()` (`server.go`).
- **Economy:** `uint64` micro-units only in all ledger math (Architecture Ledger Â§Deterministic Finance). Floats prohibited in logic; permitted for UI display only.
- **Multi-chain:** **Voi is primary** (transactions). Algorand is secondary (assets/metadata only). All other chains (Ethereum/Solana/Polygon/Bitcoin/Flow/WAX) are **metadata/indexer sources only â€” for NFT import** (`server.go:456`: *"Other chains added as Metadata sources only - No transaction capability implied"*). See Â§6 for a contradiction to reconcile.

---

## 1. Build & Verification Truth

| Check | Result | Note |
|---|---|---|
| `GOOS=linux GOARCH=amd64 go build ./...` | âœ… GREEN | Authoritative server target |
| `GOOS=js GOARCH=wasm go build ./...` | âœ… GREEN | Client/WASM target |
| `npm run build` (wasm+sass+server) | âœ… exit 0 | Frontend build |
| `go vet ./...` | âœ… exit 0 (clean) | Verified 2026-09-09 (Go 1.26.4). A real bonded-asset serialization bug exists (Â§4) that `go vet` does **not** flag â€” truth is the broken round-trip, not a vet warning. |

**`go vet ./...` is CLEAN (exit 0, verified 2026-09-09 on Go 1.26.4).** The prior Session-Handoff claim of *"2 pre-existing `go vet` errors (`economy_bootstrap.go:125` copylocks + `bonded_asset_registry.go:68` unexported json tag)"* is **FALSE** â€” `go vet ./...` produces no output, and `economy_bootstrap.go:125` is an ordinary `range` loop, not a copylocks violation. (This is itself an instance of the Session-Handoff falsification called out in Â§10.)

**However, a REAL correctness bug exists that `go vet` does not catch:** `BondedAssetRegistry.assets`/`bindings` are **unexported** yet carry `json:"..."` tags (`bonded_asset_registry.go:68`). `encoding/json` silently ignores unexported fields, so `Save()` writes an effectively empty file and `Load()` can never rehydrate it â†’ **bonded assets are in-memory/ephemeral (lost on restart)**. See Â§4. This is the genuine Phase-C issue.

---

## 2. Authoritative Core Topology (84 `.go` files, grouped)

All paths relative to repo root. `package main`; server cross-compiled `!js && !wasm`, WASM engine `js && wasm`.

### 2.1 Orchestration / bootstrap
- `server_main.go` (1548) â€” `main()`, HTTP `mux`, **route registration (300 handlers)**, graceful shutdown.
- `server.go` (749) â€” `newLobby()` orchestration, `loadNetworkConfigs()` (Â§6), `getDataPath()`, network/global config.
- `lobby_manager.go` (5021) â€” `*Lobby` god-object: event loop `run()`, WS hub (`broadcast`, `sendToClientLocked`), leaderboard, clubs, global mutex, persistence orchestration.
- `main.go` (4027) â€” WASM game engine entry, `SyncMove`/`window.*` exports, avatar/deck sync.


### 2.2 Economy (Pillar 2 â€” uint64 micro ledger)
- `economy_service.go` (724), `economy_processing.go` (547), `economy_bootstrap.go` (169), `economy_persistence.go` (209), `economy_audit.go` (179), `economy_telemetry.go` (152).
- `faucet_service.go` (605), `tournament_manager.go` (956), `auction_service.go` (463), `loan_service.go` (472), `black_market_service.go` (1122), `counterfeit_service.go` (317), `redemption_gateway.go` (222), `industrial_loop.go` (257).

### 2.3 Social / career / justice / criminality
- `battle_service.go` (2320), `club_service.go` (3294), `career.go` (130), `rival_career_engine.go` (782), `rivalry_engine.go` (258), `rivalry_handlers.go` (626).
- `justice_service.go` (597), `justice_handlers.go` (425), `courthouse_service.go` (171), `underworld_contracts.go` (765), `handlers_criminality.go` (1066), `employment_service.go` (189).

### 2.4 Entities / life / AI citizens (the "everything as NFT" domain)
- `ai_citizen_engine.go` (1568), `asset_life_engine.go` (216), `entity_event_engine.go` (829), `entity_investment_service.go` (788), `entity_market.go` (599), `entity_shares.go` (160), `entity_tournament_scheduler.go` (141).
- `item_service.go` (974), `item_shop_archetype.go` (288), `shop_registry.go` (350), `creator_economy.go` (333), `creator_store_service.go` (470).
- **`bonded_asset_registry.go` (567)** â€” Â§23.5 bonded-asset NFT registry (theming + asset assigning). **See Â§4 â€” partial + persistence-broken.**
- `pet`/`vehicle`/`world-content` NFTs are defined in `backend_types.go`, managed via `asset_life_engine.go` (`InitAssetLife`, `server.go:273`).

### 2.5 Theming / world / rivalry / faith
- `theme_engine.go` (1188) â€” Â§27 ThemeEngine: `ThemeVector`, `ComputeWorldDynamicsSignature`, `MoodTag` sourcing from bonded assets, `/api/theme/*`, `/api/rivalry/world-dynamics`.
- `seasonal_event_engine.go` (875), `faith_church.go` (480), `religion_governance.go` (458), `governance.go` (315), `launchpad.go` (273).

### 2.6 Identity / bridge / multi-chain (Â§6)
- `identity_bridge.go` (208), `persistent_identity.go` (253), `bridge_router.go` (240), `bridge_service.go` (103), `ethereum_client.go` (450), `nautilus_dex_path.go` (97), `oracle_service.go` (1800 â€” Envoi/chain oracles), `handlers_public.go` (656), `handlers_public_new.go` (654), `handlers_admin.go` (1920), `handlers_rumor.go` (236).

### 2.7 Support / infra
- `rate_limiter.go` (457), `replay_engine.go` (304), `achievement_service.go` (296) + `achievement_handlers.go` (201), `advertising.go` (359), `gaming_os.go` (402), `infrastructure_lease.go` (113), `console_server.go` (216), `local_model_promotion.go` (375), `onboarding_service.go` (471), `player_service.go` (68), `narrative_service.go` (87), `menu_state.go` (252), `stat_overlay.go` (210), `resilience_utils.go` (520), `common_types.go` (622), `common_types_wasm.go` (560), `backend_types.go` (719 â€” central type definitions).

---

## 3. Lobby Lifecycle & Service Wiring (`newLobby`, server.go:240â€“300)

`newLobby()` constructs the world in this order (verified):

1. `aiEngine = NewAICitizenEngine()` + `StartBehavioralLoop()` + `LoadCitizens()` (P7-D autonomous economy).
2. `itemRegistry = NewItemRegistry()` + `Load()` â€” **persists correctly** (contrast with Â§4).
3. `seasonEngine`, `InitRivalryEngine()` (Â§25.10), `InitAssetLife()` (Â§26.4/Â§25.6/Â§25.7), `InitThemeEngine()` (Â§27).
4. **`bondedAssets = NewBondedAssetRegistry()` + `Load()`** â€” Â§23.5 registry (Load is currently a no-op; see Â§4).
5. `localModelPromotions` (Â§24.5/Â§24.6), `entityEvents` (Â§30 Pet World).
6. Faith: seed 24 faucet-owned religions, wire `SetGlobalLobbyRef`, load churches (Â§32).

**Persistence call sites** (`economy_persistence.go` `commitState`): AI citizens, **bonded assets (197â€“201)**, local-model promotions, core faucet state. Bonded-asset `Save()` is invoked here but writes empty data (Â§4).



## 4. Bonded Asset Registry â€” Theming & Asset Assigning (`bonded_asset_registry.go`)

Â§23.5 registry of **player-created cosmetic NFTs bound to the game-hub lease**. This is the system the user flagged for "theming and asset assigning."

**Asset types (`AssetType` enum 0â€“6):** `Skin, Background, Board, Button, Appearance, Audio, Custom` â€” i.e. **UI elements**. (No `Item`/`Profile`/`Entity` type yet â€” see Â§5.)

**Core types:**
- `BondedAsset`: `AssetID` (`"BA-"+uuid`), `AssetType`, `Name` (user-customizable), `CreatorWallet`, `OwnerWallet`, `HolderWallet`, `MoodTag` (0/1/2), `RoyaltyBps` (â‰¤1000), `CreatedAt`, + Â§27.7.3 provenance (`BirthCertID`, `Certified`, `BlackMarketAdopted`).
- `ThemeBinding`: `AssetID`, `Slot`, `Locked` (Â§27.6 lock-state scaffold).
- `BondedAssetRegistry`: `mu`, `assets`, `bindings` maps.

**Theming (`MoodTagForWallet`):** aggregates the `MoodTag` of the wallet's **non-locked** bound assets into a single theme MoodTag (Â§27.1 cosmetic term: NEUTRAL/BENEVOLENT/MALEVOLENT mapped to 0/+1/âˆ’1, center-of-mass). Consumed by `theme_engine.go` to set the wallet's cosmetic theme mood. **This is the "theming" path the user referenced.**

**Asset assigning:** `BindThemeAsset(assetID, slot)` binds an asset to a slot (skin/background/board/button/â€¦); `LockThemeAsset` flips a binding to `Locked=true` (Â§27.6: removed from play unless re-bound/activated). The Â§27.8 canonical guard `guardOwnerHolder` requires `caller == OwnerWallet == HolderWallet` for modify/transfer/burn/lock.

**HTTP surface:** `handleMintBondedAsset` (`/api/assets/mint`), `handleListBondedAssets` (`/api/assets`), `handleBurnBondedAsset` (`/api/assets/burn`), `handleModifyBondedAsset` (`/api/assets/modify`), `handleTransferBondedAsset` (`/api/assets/transfer`), `handleBindThemeAsset`/`handleLockThemeAsset`, plus `theme_engine.go` `/api/theme/bind`, `/api/theme/lock`.

### âš ï¸ PERSISTENCE BUG (real, not cosmetic)
- `BondedAssetRegistry.assets` / `.bindings` are **unexported** but carry `json:"assets"` / `json:"bindings"` tags (`bonded_asset_registry.go:68`). `go vet` does **not** flag this on Go 1.26.4, but the unexported fields are silently dropped by `encoding/json` regardless.
- `Save()` (296) marshals a `*BondedAssetRegistry` snapshot. `json.Marshal` **silently ignores unexported fields**, so the written file is effectively `{}`.
- `Load()` (333) unmarshals into `var snapshot BondedAssetRegistry`; unexported fields cannot be populated, so `if snapshot.assets != nil` is always false â†’ registry stays empty.
- **Net effect: bonded assets are in-memory/ephemeral â€” lost on every restart.** `economy_persistence.go:197` calls `Save()` each commit; failures are swallowed ("Bonded asset snapshot skipped").
- **Contrast:** `ItemRegistry.Save` (`item_shop_archetype.go:230`) serializes `[]*BuiltItem` (exported structs) â†’ persists correctly.
- **Correct fix (Phase C):** serialize a DTO with **exported** fields (mirror `ItemRegistry`), e.g. `json.Marshal(struct{ Assets map[string]*BondedAsset; Bindings map[string]*ThemeBinding })` or a slice. Do **not** blindly drop the json tag â€” fix the serialization so assets actually round-trip.

---

## 5. "Everything as NFT" â€” Current Reality (honest assessment)

The vision: UI elements, Items, Profiles, and Entities (bots/pets/llm-everything) all become NFTs. **Current state is PARTIAL and NOT unified into one registry:**

| Category | Representation | BondedAsset-linked? | Persists? |
|---|---|---|---|
| UI elements (skin/bg/board/button/audio) | `BondedAsset` (Â§23.5) | YES (native) | âŒ broken (Â§4) |
| Pets | `PetNFT` (Â§26.4) | NO â€” separate struct | YES (asset_life) |
| Vehicles | `VehicleNFT` (Â§25.6) | NO | YES |
| World content | `WorldContentNFT` (Â§25.7) | NO | YES |
| Items | `BuiltItem`/`ItemArchetype` (`item_shop_archetype.go`) | NO â€” `ItemRegistry` | âœ… YES |
| Profiles | `persistent_identity.go` | NO | YES |
| AI citizens / bots | `AICitizen` (`ai_citizen_engine.go`) | NO | YES (aiEngine) |

**Provenance is consistent** across entities via Â§27.7.3 fields `Certified` / `BlackMarketAdopted` / `BirthCertID` (applied to `PetNFT`, `AICitizen`, `BondedAsset`). Legitimacy = right to think/breed (black-market lineage is ineligible for legitimate breeding/events).

**The "everything as asset" concept already exists at the rivalry/score layer:** Â§25.10 `AssetSignature` aggregates weighted counts â€” `W_USER_WORKER, W_AI_PRIDE, W_MODEL_CITIZEN, W_VEHICLE, W_PET_BLOODLINE, W_WORLD_CONTENT, W_ITEM_ARCHETYPE, W_EVENT_TYPE` â€” into a single integer `Score`. So assets are *weighted together* for region/territory rivalry, but they are **not minted into one unified NFT/bond registry**.

**Correction of prior false claim:** Session-Handoff Â§12.1 described `bonded_asset_registry.go` as the *"NFT-as-everything record."* That is **overstated** â€” it is the **UI-element** NFT record only. The broader "everything becomes an NFT" is scaffolded across several separate structs plus the Â§25.10 weighting model, with provenance fields applied per Â§27.7.3, but **there is no single unified "mint anything as a bonded asset" path yet.** This is the genuine remaining work the directive points at.



## 6. Multi-Chain Model â€” x-chain is NFT-IMPORT ONLY (user reframing, 2026-09-09)

**Intent (confirmed by `server.go:456` + user directive):** the non-Voi chains exist so their **indexers can import NFTs / read metadata**; they are NOT transaction/settlement chains. Voi is primary (transactions). Algorand is secondary (assets/metadata only).

### 6.1 `networks.json` (8 chains)
`Voi Mainnet`, `Algorand Mainnet`, `Ethereum`, `Solana`, `Polygon`, `Bitcoin`, `Flow`, `WAX`. Each defines `explorer_url`, indexer/node URLs, `chain_id`, `power_divisor`, `power_base`.

`loadNetworkConfigs()` (`server.go:423`): Voi + Algorand get full `IndexerURLs` + `NodeURLs` (transaction-capable). The rest are added with the explicit comment **"Other chains added as Metadata sources only - No transaction capability implied"** (server.go:456).

> âš ï¸ **Concrete bug:** `networks.json` uses the **singular** keys `indexer_url`/`node_url` for the non-Voi chains, but `NetworkConfig.IndexerURLs` is **plural** (`json:"indexer_urls"`). On `json.Unmarshal` those chains' `IndexerURLs` come back **empty**, so their NFT-import indexers never load. Fix: normalize the JSON keys (or add singular-alias fields). Until then, non-Voi indexers are effectively dead even though the intent is to use them for import.

### 6.2 Bridge (cross-chain asset records)
- `bridge_router.go` â€” `BridgeRouter`, `ChainAsset`, `BridgeTransaction`; `BridgeAsset(from,to,â€¦)`, `ConfirmBridge`. In-memory only (no persistence of bridged assets).
- `bridge_service.go` â€” mostly **frontend registration stubs** (`connectWallet`, `disconnectWallet`, `SendReward`, etc. are `registration-only`, `implementedIn:null`); real logic is in `bridge_router.go`.
- Routes: `/api/bridge/assets`, `/api/bridge/summary`, `/api/bridge/asset`, `/api/bridge/txs`, `/api/bridge/confirm`, `/api/bridge/onboard`.

### 6.3 Ethereum client â€” âš ï¸ contradiction to reconcile
`ethereum_client.go:23` declares Ethereum the *"primary chain for ETH-based NFT settlement and gas token transfers"* and implements `SendETH` + a vault (`ETH_VAULT_KEY` / `ETH_VAULT_ADDRESS`). **This implies real ETH settlement/gas spends**, which conflicts with the "x-chain is NFT-import only" intent. Either (a) the settlement path is dead/incomplete scaffolding, or (b) the intent statement needs updating. **Flagged for reconciliation** â€” do not assume the ETH vault is live without verifying call sites.

### 6.4 Other chain services
- `nautilus_dex_path.go` â€” *"simulates interaction with a DEX to acquire $VBV from a system reserve (funded by console revenue) and distribute to browser-based creators."* Consoleâ†’browser creator payout, not NFT import.
- `oracle_service.go` (1800) â€” Envoi/chain oracles. `ResolveEnvoiName` now calls the **live canonical Envoi API** (`api.envoi.sh`); `/api/envoi-name` resolves Voi addressâ†’`*.voi` name. This is the real Voi naming path (not the legacy indexer `*.voi` NFT scan workaround).

**Conclusion:** Infrastructure mostly matches the NFT-import-only intent (indexer URLs for metadata). Two real issues: (1) `networks.json` singular/plural key mismatch nullifies non-Voi indexers; (2) `ethereum_client.go` settlement/vault capability contradicts the intent and must be reconciled.

---

## 7. Core Economic Sequences (Pillar 2 â€” uint64 micro ledger)

- **Deterministic finance:** all balances, transfers, taxes, rewards, debts, interest, fees, royalties, career/combat XP are `uint64` micro-units. **No float in logic** (Architecture Ledger Â§Deterministic Finance). Floats allowed for UI display only.
- **Industrial Loop:** every transaction reconciles; no silent mint/burn; remainders route to a deterministic sink (Faucet / Treasury / approved sink).
- **Flows:** Faucet (`faucet_service.go`) = global reservoir â†’ rewards/wages â†’ sinks (taxes, siphon, burns) â†’ back to faucet/treasury.
- **Authority:** Go server is authoritative; WASM mirrors gameplay. No simulation drift across server/WASM/browser.
- **Recovery:** authoritative state reconstructible from blockchain; `DATA_DIR` (env) holds performance/recovery snapshots via `getDataPath()` (`server.go:21`).



## 8. Route Surface (authoritative â€” 300 `mux.HandleFunc`, ~298 unique `/api/*`)

Enumerated directly from `server_main.go` (2026-09-09). Grouped by domain:

- **Core economy:** `/api/reward`, `/api/leaderboard`, `/api/card-stats`, `/api/card-details`, `/api/auctions` (GET/POST), `/api/loans/{take,repay}`, `/api/black-market/{buy,buy-stolen,fence-goods,sell-tokens}`, `/api/tournament/{register,history}`, `/api/industrial-loop/{metrics,health,record}`, `/api/match/wager`, `/api/wagers`, `/api/wagers/resolve`.
- **Achievements:** `/api/achievements`, `/api/achievement-stats`, `/api/achievement/unlock`.
- **Faction shop:** `/api/faction/shop/` (GET list / POST buy).
- **Underworld / criminality:** `/api/underworld/{contracts,heists,kidnaps}`, `/api/bounty/active`, `/api/criminality/cyber-intercept`, `/api/contracts/{list,assign}`, `/api/courthouse/reset`.
- **Clubs / territory:** `/api/clubs`.
- **Justice:** `/api/justice/{award-card,bounty-board,capture-bounty,dashboard,missions,missions/accept,use-rep-shield,use-truth-serum}`.
- **AI citizens:** `/api/ai/citizens/{adopt,adopt-pet,breed,business/spawn,challenge,free-agents,list,marry,progress,release,spawn,stats}`.
- **Entities / pets / vehicles / world-content:** `/api/pets`, `/api/pets/{breed,spawn}`, `/api/pet-battle/{challenge,list,resolve}`, `/api/entity/market/{create,list,purchase}`, `/api/entity-event/{host,resolve}`, `/api/entity-events/regions`, `/api/children-bots`, `/api/orphan/{adopt,reclaim,status}`, `/api/vehicles`, `/api/vehicles/spawn`, `/api/world-content`, `/api/world-content/{create,deploy}`.
- **Items:** `/api/items/{archetypes,bind-nft,build,collection,registry}`, `/api/shop/purchase`.
- **Creator store:** `/api/creator/{dlc/create,dlc/purchase,dlcs,event/attend,event/create,events,royalties,store/*,sub/create,subs}` (+ `/api/creator/store/{purchase,products,profile,rate,resell,royalty-history,â€¦}`).
- **Bonded assets / theming:** `/api/assets`, `/api/assets/{mint,burn,modify,transfer}`, `/api/theme/{bind,lock,vector}`, `/api/market/weather`.
- **Rivalry / world-dynamics:** `/api/rivalry/{action,detect,factions,join,list,recompute,request,resolve,state,world-dynamics}`.
- **Faith / church:** `/api/faith/{coherence,high-tier,religion/*,religions,war-gambit}`, `/api/church/{add-item,add-member,get,items,leaderboard,members,open,owner,region,remove-member,ritual,rituals}`.
- **Governance:** `/api/governance/{close,election,governor,leaderboard,register,vote,weight}`.
- **Identity:** `/api/identity/{events,leaderboard,link,profile,record,resolve,snapshot,unlink}`.
- **Bridge:** `/api/bridge/{asset,assets,confirm,onboard,summary,txs}`.
- **Oracles / naming:** `/api/envoi-name`, `/api/rumors`.
- **Investment / shares:** `/api/invest/{entity,portfolio,dividends/history}`, `/api/shares/{buy,holdings,issue,tokens}`, `/api/claim/dividends`.
- **Launchpad:** `/api/launch/{activate,back,create,creator,get,integrate}`, `/api/launches`.
- **Ads / gaming-OS / social:** `/api/ads`, `/api/ad/{activate,click,create,impression,pause,stats}`, `/api/os/*`, `/api/garden*`, `/api/tea*`, `/api/moods*`, `/api/titles*`.
- **Player / replay / regions:** `/api/player/*`, `/api/players/constellation`, `/api/owner/combined-stats`, `/api/stat-overlay*`, `/api/replay/*`, `/api/regions`.
- **Compliance:** `/api/compliance/{escalate,record,records,resolve,summary,wallet}`.
- **Faucet:** `/api/faucet/{status,claim,vault-balance}`.
- **Events / treasure / seasons:** `/api/events/{create,enter}`, `/api/treasure/{claim,spawn}`, `/api/season/*`.
- **Counterfeit:** `/api/counterfeit/{detect,generate}`.
- **Redemption:** `/api/v1/redemption_gateway`.
- **Admin (wallet-default rate limit):** `/api/admin/*` (asset-forfeiture, avatar-ban, commission-audit, district-tax-audit, dlc-registry, emergency-shutdown, export-logs, force-payout, ledger-audit,logs, mutation-audit, network/add, open-registration, sanity-check, season-rollover, set-admin-focus-network, simulate-*, start-tournament, tax-audit, update-power), `/api/refill-vault`, `/api/update-rules`, `/api/maintenance-mode`, `/api/system-message`, `/api/ban-player`, `/api/reset-stats`, `/api/re-sync-stats`, `/api/report-player`.
- **Static / HTML / WS:** `/` , `/dashboard`, `/diagnostics`, `/manuals/`, `/spectate`, `/split`, `/tutorials/`, `/watch`, `/world`, `/ws`.

> Rate limiting: every handler is wrapped via `lobby.rateLimiter.WithRateLimit(handler, bucket)` with buckets `economy-tight`, `core-economy`, `standard`, `achievement`, `wallet-default`, `default` (see `rate_limiter.go`).

---

## 9. WebSocket Protocol (brief)

- Entry: `/ws` â†’ `serveWs(lobby, â€¦)`. `CheckOrigin` handles WS CORS; handshakes are not rate-limited.
- Messages are `Envelope{Type, Payload}` JSON. Serverâ†’client broadcasts run through `lobby.broadcast` channel; targeted sends via `sendToClientLocked` / `sendToClient`.
- Authoritative frames (`AuthoritativeFrame`: Move + SequenceID + `BoardStateHash`) are committed to `sh.HistoricalFrames` and broadcast; the WASM client (`main.go`, `window.SyncMove`) verifies sequence continuity + hash parity, rolls back on mismatch (replay recovery).
- Lobby state push: `getLobbyUpdateMsg()` (high fan-in: 73) drives the client HUD/leaderboard.



## 10. Corrected Falsifications (prior docs vs code truth)

| Prior claim (doc/handoff) | Reality (verified 2026-09-09) |
|---|---|
| V2.1 Â§12.4: *"server target does NOT compile (14 errors)"* | **FALSE now** â€” both `GOOS=linux GOARCH=amd64` and `GOOS=js GOARCH=wasm` `go build ./...` are **GREEN**. The 14-error state was historical and since fixed. |
| Session-Handoff Â§12.1: *"Bonded Asset Registry (NFT-as-everything record) â€” Completed"* | **MISLEADING** â€” it is the **UI-element** NFT record only (Â§4), and its **persistence is broken** (Â§4 bug). Not "everything," not fully working. |
| Prior audit note: *"/api/envoi-name is dead code"* | **FALSE** â€” it is the planned **and now-built** Voi naming endpoint, wired to the live Envoi API (`oracle_service.go` + `handlers_public.go` + `server_main.go:325`). |
| Earlier Phase-A plan: *"clean stray ETH/Flow/Polygon/Solana/WAX `indexer_url` garbage in networks.json"* | **WRONG under current framing** â€” those indexers are legitimate **NFT-import/metadata** sources (Â§6). Do **not** delete them. The real bug is the singular/plural key mismatch (Â§6.1), not their existence. |
| Implicit: *"App is Algorand-primary"* | **FALSE** â€” app is **Voi-primary** by default (`server.go` `DEFAULT_NETWORK` â†’ Voi); Algorand is secondary/metadata-only. |
| *"All systems shipped / Phase 7 complete"* narrative | True for gameplay pillars; **NOT** true for asset/NFT unification + bonded-asset persistence. |

---

## 11. Audit Findings & Recommended Next Steps (honest, not stubbed)

1. **Phase C â€” Bonded-asset persistence (highest priority correctness bug).** Root cause verified: `bonded_asset_registry.go:68` unexported `assets`/`bindings` fields with `json` tags â†’ `Save()` writes `{}`, `Load()` never repopulates â†’ assets lost on restart (`economy_persistence.go:197` calls `Save()` each commit; failures swallowed). **Fix:** serialize an exported DTO (mirror `ItemRegistry.Save`, `item_shop_archetype.go:230`). This also clears the `go vet` error.
2. **`networks.json` key normalization.** Non-Voi chains use singular `indexer_url`/`node_url`; struct expects plural `indexer_urls`. Normalize so NFT-import indexers actually load (Â§6.1).
3. **Ethereum client reconciliation.** `ethereum_client.go:23` + `SendETH` + `ETH_VAULT_*` imply real ETH settlement, contradicting the "x-chain is NFT-import only" intent. Verify whether this is dead scaffolding or live; reconcile code + docs (Â§6.3).
4. **"Everything as NFT" unification (the directive's core ask).** Decide: extend `BondedAsset.AssetType` to cover Item/Profile/Entity and link `PetNFT`/`VehicleNFT`/`WorldContentNFT`/`AICitizen` via `AssetID`, **or** keep separate registries (current state) and rely on Â§25.10 `AssetSignature` weighting for unified asset power. Provenance fields (`Certified`/`BlackMarketAdopted`/`BirthCertID`, Â§27.7.3) are already consistent across entity types.
5. **Session-Handoff falsification.** Prior Session-Handoff overstated "completed" claims (e.g., bonded asset). Treat **this document + live code** as the corrected baseline; reconcile Session-Handoff before trusting it.
6. **Frontend/backend 404 gaps.** ~26 frontend calls still hit missing/renamed backend routes (2026-09-04 gap analysis). Pending route registration or button removal â€” separate from this doc's scope but tracked.
7. **Build hygiene.** Keep both `go build` targets + `npm run build` green. `go vet ./...` is clean (exit 0). The bonded-asset serialization bug (Â§4) is a real runtime defect that vet does not surface â€” fix it via the exported-DTO serialization described in Â§4.

---

---
## 12. Frontend Flow (read from source, 2026-09-09)

> All claims below trace to files actually read/grepped: `Public/index.html`, `Public/app.js`, `Public/js/network.js`, `Public/js/player_profile.js`, `Public/js/ui.js`, `Public/src/scss/components/_overlays.scss`, plus a full `/api` + WS surface extraction across all 141 JS modules.

### 12.1 Inventory & bootstrap
- **141 JS modules** (`Public/js/*.js`), **106 SCSS partials** (`Public/src/scss/**`), built `Public/styles.css` (~484 KB, single minified line), `Public/index.html` (~1063 lines).
- **Load order (`index.html`):** `wasm_exec.js` â†’ vendor SDKs (`algosdk`, `walletconnect-sign-client`, `walletconnect-modal`) â†’ `buffer` polyfill â†’ inline `WasmWalletBridge` â†’ **~140 classic `<script src="js/X.js">` panel modules** (each self-binds a `window.openX` global) â†’ **`app.js` (`type="module"`)** the ESM orchestration hub â†’ `world3d.js` (`type="module"`) â†’ `panel_manager.js`.
- **Two hubs:** `app.js` imports domain logic + `network.js` and exposes it to `window`. `network.js` owns the WebSocket (`initWebSocket` â†’ `ws://<host>/ws`, `network.js:34-42`). `player_profile.js` is the **Player Profile Hub** (12 categories, each `renderX` opener). `ui.js` holds the action bar + `hideAllOverlays()`.
- **Three overlay classes, all contained:** legacy `.overlay` (economy/criminality/game), the Profile Hub `.pp-hub`, and the 13 design-system `.vbt-overlay` panels.

### 12.2 API call surface (the request flow contract)
- Extracted **284 unique `/api/*` paths** called from JS. Cross-referenced against **303 backend-registered routes** â†’ **0 path mismatches**. The 2026-09-04 "26 frontend 404s" are **RESOLVED** (fixes: `economy.js` auction `POST /api/auctions`; `creator_storefront.js` `POST /api/creator/store/purchase/`; 4 new backend routes `/api/faucet/vault-balance`, `/api/faucet/claim`, `/api/underworld/heists`, `/api/underworld/kidnaps`).
- **HTTP-method matrix (CLOSED 2026-09-09):** exhaustively diffed **304 unique FE `(method,path)` calls** (extracted from all `Public/js/*.js`) against **222 backend routes** mapped to their handler + `MethodPost`/`MethodGet` guard. Result: **7 FE-GET / BE-POST candidates**, all verified real (the per-module `api(path,opts)` helper passes `opts` through, so no-method calls default to GET; the target handlers enforce POST):
  - **6 of 7** are inside `misc_panel.js` `loadExtended()` â€” a **diagnostic smoke-test** that fires ~150 `api('/api/...')` calls in a `Promise.all`, swallows every error (`.catch(()=>({}))`), and only counts "N endpoints available." Low impact (non-functional prefetch; panel data comes from dedicated loaders).
  - **1 of 7** is `match_arena.js:24` â€” `fetch('/api/match/wager')` with no method (GET) on the POST-only `handleSpectatorWager` (`handlers_public.go:21`). **`match_arena.js` is an ORPHAN** (never loaded in `index.html`), so this is latent dead-code, not a live break. No GET match-listing endpoint exists (the only `active_matches` source is a GET *status* handler, `handlers_public.go:267`, whose shape omits `wager_min_micro`). â†’ Recommend: point the arena at the status endpoint or add a GET match-list endpoint (out of scope this pass).
  - **No FE-POST / BE-GET mismatches found.** Caveat resolved â€” method axis fully verified.

### 12.3 WebSocket event surface (the live-update contract) â€” **contains the headline flow error**
- `network.js:68` is the **sole** `socket.onmessage` (no second listener anywhere â€” grep confirms). It is a single `switch` with **no `default` branch** and ~45 `case` labels (identity, lobby_update, mp_*, portfolio_update, challenge, match_start, sync_response, turn_change, chat, vault_update, rules_update, rewards_update, maintenance_update, tournament_update, admin_notification, kidnap_*, ransom_*, rumor_update, achievement_unlock, justice_*, underworld_contract_*, seasonal_event_*).
- Backend emits **21 `Envelope{Type:...}` types**. Of these, **9 have NO frontend `case`** and are **silently dropped** (switch falls through to `}` at `network.js:475`):

  | Backend-emitted event | Frontend consumer? | Impact |
  |---|---|---|
   | `rivalry_update` | [CONSUMED] `rivalry_viewer.js` `window.onRivalryUpdate` -> live grid refresh when panel open |
   | `investment_confirmed` | [CONSUMED] `investment_dashboard.js` `window.onInvestmentConfirmed` -> portfolio refresh |
   | `investment_update` | [CONSUMED] `window.onInvestmentUpdate` -> portfolio + marketplace refresh |
   | `dividend_claimed` | [CONSUMED] `window.onDividendClaimed` -> dividend tracker refresh |
   | `creator_royalty_paid` | [CONSUMED] `creator_store.js` `window.onCreatorRoyaltyPaid` -> earnings/royalty history refresh |
   | `creator_royalty_received` | [CONSUMED] `window.onCreatorRoyaltyReceived` -> earnings/royalty history refresh |
   | `career_tier_demoted` | [CONSUMED] `player_profile.js` `window.onCareerTierDemoted` -> career-demotion toast |
   | `link_wallet_response` | [CONSUMED] `wallet.js` `window.onLinkWalletResponse` -> toast + close link overlay on success |
   | `nonce_response` | [CONSUMED] `network.js` `window.onNonceResponse` resolves the pending wallet/admin nonce promise (REQUIRED; identity case does NOT resolve nonce) |

   **Severity: MEDIUM — dispatch path FIXED 2026-09-09; module consumers BUILT 2026-09-09 (see table above).** `network.js` now has explicit `case` handlers for all 9 events (lines 478–494) plus a `default:` branch (496–499) that forwards any unhandled event to `window.__vbtWsDispatch(msg)` (preventing silent drops). The 9 cases route to the conventional `window.on<Event>(payload)` handler **if the module defines one** — but as of this audit **all 9 consumers now defined in their domain modules**, so the events reach a dispatch point but are not yet consumed for live UI refresh. Module consumers built 2026-09-09 (`window.onRivalryUpdate`, `window.onInvestmentUpdate`, `window.onDividendClaimed`, `window.onCreatorRoyaltyPaid/Received`, `window.onCareerTierDemoted`, `window.onLinkWalletResponse`, `window.onNonceResponse`) — the dispatch plumbing is already in place. Affected UIs still refresh via the periodic `requestBatchedSync("all")` polling, so data is eventually correct; real-time push now works (visibility-guarded refresh)




### 12.4 CSS / containment â€” **VERIFIED HEALTHY**
- `main.scss` imports `layouts/app-shell` (single contained shell). `_overlays.scss:9` `.vbt-overlay` and `:44` `.overlay` are both `position: fixed; top/left:0; width/height:100%`, internally scrollable (`overflow-y:auto`, children `max-height:92vh`), hidden via inline `display:none`. Constitution's *contained-overlay mandate* (`position:fixed; inset:0; overflow:hidden`) is satisfied. No page scroll.

### 12.5 Dead code / orphan modules
- **`bounty_tracker.js` (425 lines) â€” ORPHAN.** Not present in `index.html` script list; superseded by `criminality.js` `openBountyBoard()`. RESOLVED 2026-09-09: DELETED (was never in `index.html`; superseded by `criminality.js` `openBountyBoard`). Bounty system single-source.
- **`openCriminality` â€” UNDEFINED.** Only referenced guarded in `constellation_spectate.js:254` (`typeof window.openCriminality === 'function'`); the Underworld hub panel (`player_profile.js:740` `renderUnderworld`) actually calls `window.openBountyBoard` (works). No crash, but the `openCriminality` hook is dead.

### 12.6 Flow-error summary (the deliverable)
| # | Flow error | Evidence (read) | Severity |
|---|---|---|---|
| 1 | Backend **double-registers** `/api/entity-market/*` (hyphen) **and** `/api/entity/market/*` (slash); frontend uses the slash form â†’ hyphen form redundant (different server) | `server_main.go` `HandleFunc` lines show both forms | Low -> RESOLVED 2026-09-09 (console_server.go aligned to slash) |
| 2 | **9 backend WS pushes silently dropped** â†’ **FIXED 2026-09-09** (`network.js` switch now has the 9 `case`s + `default`; ends `:500`); module consumers BUILT same day (see table above) (Â§12.3) | `network.js:68` switch (ends `:500`); backend `Envelope{Type}` grep | Mediumâ†’Low (resolved) |
| 3 | `openCriminality` undefined (guarded-only dead hook) | `constellation_spectate.js:254`; no def in any JS | Low |
| 4 | `bounty_tracker.js` orphan â†’ **RESOLVED 2026-09-09 (deleted)**; `criminality.js` `openBountyBoard()` is the sole bounty system (NOT dropped, NOT duplicated) | deleted file; `criminality.js:318` | resolved |
| â€” | (RESOLVED) 26 frontend `/api` 404s from 2026-09-04 | path diff = 0 across 284 FE / 303 BE paths | resolved |

### 12.7 Recommended fixes â€” status (2026-09-09)
- [DONE] **#1 WS wiring:** `network.js` 9 `case` handlers + `default` dispatcher (lines 478â€“499). module-side window.on<Event> consumers BUILT same day (see table above) (see Â§12.3).
- [DONE] **#2 `entity-market` route consolidation:** `console_server.go` hyphen routes aligned to slash; both servers use `/api/entity/market/*`.
- [DONE] **#3 `bounty_tracker.js`:** deleted (orphan duplicate of `criminality.js` `openBountyBoard`). Bounty system intact, single source.
- [DONE] **#4 HTTP-method matrix:** exhaustive diff run; 7 candidates documented (Â§12.2), all low-impact (diagnostic smoke-test + orphan file).
- [DONE] **#5 Module-side WS consumers** (`window.on<Event>`) for the 9 events so the wired dispatch actually refreshes Rivalry/Investment/Creator-royalty/Career/Wallet-link UIs.
- [DONE] **#6 `match_arena.js` orphan RESOLVED + REBUILT:** deleted orphan replaced by rebuilt `match_arena.js` — GET `/api/match/active` route registered (handler was defined-but-unrouted), UI lists active matches + places wagers via POST `/api/match/wager`; wired into index.html action bar (Match Arena button).
- [NEW] **#7 `openCriminality` undefined** (`constellation_spectate.js:254`): still a guarded dead hook â€” either define `window.openCriminality` (â†’ `criminality.js`) or remove the call site.

### 12.8 Comparison with prior reference docs (2026-09-07) â€” drift check
Compared Â§12 against `archive/docs-2026-09-07/File-Flow-Overview-1.md` (OLD logical flow doc), `App-Aspect-Index.md` (39-section consolidated ref), `Background-vs-UI-Categorization.md` (Background/UI/Hybrid classification). All three are dated **2026-09-07**, ~2 days before this 2026-09-09 code audit.

**Alignment (docs state intent; Â§12 confirms/refines the code):**
- `Background-vs-UI-Categorization.md:169` asserts the 8 hybrid systems "surface back to the player through **real-time UI updates**" (WebSocket). Â§12.3 shows this holds for ~12 event types but is **FALSE for 9** (`rivalry_update`, `investment_confirmed`, `investment_update`, `dividend_claimed`, `creator_royalty_paid/received`, `career_tier_demoted`, `link_wallet_response`, `nonce_response`) â€” those hybrid domains emit pushes the frontend silently ignores. **Code-drift: the WS bridge the doc describes is only partially wired.**
- `App-Aspect-Index.md:723` "WebSocket notification to creator" + Â§421/Â§491 real-time WS for Auctions/Mechanics corroborates that Creator/Investment/Entity-Market domains are *meant* to push; Â§12.3 shows `creator_royalty_*`, `investment_*`, `dividend_claimed` are dropped on the frontend. Docs describe the intent; code is missing the consumer.
- OLD flow doc (Â§0â€“Â§5) describes `app.js` â†’ `network.js` â†’ WebSocket (lines 66/81/165/571). Â§12.1 confirms this topology. No conflict â€” OLD doc is high-level narrative; Â§12 is event-precise.

**New findings NOT present in any reference doc:**
- `entity-market` hyphen/slash duplicate route registration (Â§12.6 #1).
- `openCriminality` undefined (Â§12.6 #3).
- `bounty_tracker.js` orphan (Â§12.6 #4) â€” nuance: archive orphan docs declare a philosophy that "orphaned files may be strategic hooks for future expansion" (`orphan_fix_list.md:7`); however `bounty_tracker.js` is **not** in the OLD doc's explicit "Protected Placeholders" list (OLD Â§285 lists only `bridge_service.go`), so it reads as a genuine dead supersession of `criminality.js` `openBountyBoard`, not an intentional future hook. **â†’ RESOLVED 2026-09-09: file deleted; `criminality.js` `openBountyBoard()` is the sole bounty system.**
- **0 API path mismatches** (prior 26 404s resolved, Â§12.2) â€” the prior gap analyses flagged 404s; Â§12 shows they are now fixed.

**Conclusion:** The three reference docs are accurate at the *intent/architecture* level but **overstate the completeness of the hybrid-system WebSocket bridge**. Â§12.3 is the concrete, code-verified gap. No factual contradiction â€” the docs are ~2 days stale vs the 2026-09-09 audit and predate the 2026-09-04 404 fixes.


---

*End of authoritative flow doc (Â§0â€“Â§12). Supersedes `archive/docs-2026-09-07/File-Flow-Overview-1.md`. Repository Truth prevails over any prior narrative.*


### 12.9 2026-09-09 (c) Resolution addendum (supersedes stale §12.3/§12.6/§12.7 status)
- WS consumers BUILT: all 9 backend pushes now have window.on<Event> handlers in their domain modules (rivalry_viewer.js, investment_dashboard.js, creator_store.js, player_profile.js, wallet.js, network.js). Visibility-guarded refresh — panels re-render only when open.
- nonce_response CORRECTION: previously labelled redundant/dead — WRONG. Grep proves 
onceResolver is never invoked anywhere (the identity case does NOT resolve it). window.onNonceResponse (network.js) now resolves the pending wallet/admin nonce promise, actually FIXING the link/sign flow that would otherwise time out.
- Backend integrity: entity-market hyphen routes in console_server.go aligned to slash (both servers use /api/entity/market/*); match_arena.js orphan DELETED; bonded-asset ssets/indings exported (Phase C persistence fix, both go build targets GREEN); 
etworks.json singular indexer_url -> plural indexer_urls (loader expects plural); ETH SendETH/BatchTransferETH confirmed dormant (no caller) — optional/secondary, core $VBV settlement unaffected.


### 12.10 2026-09-09 (d) Remaining open items resolved
- openCriminality dead hook RESOLVED: window.openCriminality = openCourthouse defined in criminality.js (opens the courthouse/criminality panel; non-recursive with openBountyBoard). constellation_spectate.js fallback now functional.
- Match Arena feature REBUILT: handleActiveMatches (GET) was defined-but-unrouted — registered as /api/match/active in server_main.go + console_server.go. match_arena.js recreated (lists active matches, wager form -> POST /api/match/wager), wired into index.html action bar. Both go build targets GREEN; node --check clean.