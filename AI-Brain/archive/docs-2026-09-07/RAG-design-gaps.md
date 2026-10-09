# RAG Design-Intent vs Code-Gap (NFT-Seduction)

> **⚠️ SUPERSEDED 2026-09-05 — read before trusting the MISSING conclusions below.**
> This scan was run 2026-08-29 against a **pre-reconciliation** tree and is **STALE**. A live-code
> re-verification (2026-09-05, post AI civilization backend complete, both targets GREEN) found the
> following rows WRONG:
> - **§23.5 asset registry is IMPLEMENTED** — `bonded_asset_registry.go` defines `BondedAsset`,
>   `ThemeBinding`, `BondedAssetRegistry` (+ `BindThemeAsset`).
> - **§27.7 AICitizen structs ARE PRESENT** — `ai_citizen_engine.go:49-61` defines `BondGraph`
>   (`WifeWallet`/`LoverWallet`/`BotChildIDs`), plus `BirthCertID`/`Certified`/`BlackMarketAdopted`;
>   `SpawnAI` sets `Certified=true` + `BirthCertID` at `:286-287`.
> - **§27.7 FaithCoherence IS WIRED** — `computeRegionFaithCoherence` reads `RitualsDone` from AI citizens
>   and `Club.Type:"Faith"` clubs, returns real value. 24 religions seeded on boot.
> - **§27.7.1 DomesticCoherence IS WIRED** — `computeRegionDomesticCoherence` reads `BondGraph` fields
>   (WifeWallet, BotChildIDs, PetIDs) from AI citizens, returns real value. AI citizens now marry, breed,
>   and adopt pets via `BehavioralTick`.
> - **§27.7.3 EntityLegitimacy IS WIRED** — `computeRegionEntityLegitimacy` reads `Certified` and
>   `BlackMarketAdopted` from AI citizens, returns real value.
> - **AI Civilization Backend IS IMPLEMENTED** — 10 dynamics wired into `BehavioralTick`: faith rituals,
>   marriage, breeding, pet adoption, justice enforcement, employment, rivalry, market activity, career
>   progression, event hosting.
> What is ACTUALLY deferred (verified): §27.6 lock *enforcement* (scaffold only, `Locked=false`).
> **Authoritative source of truth:** `AI-Brain/MASTER-PLAN.md` (reconciled 2026-09-05).

Generated: 2026-08-29 (background KEY 4 WAIT grounding task, per aggregator).
Method: every row below was verified by `search_files` (ripgrep) over the `.go` files and
cross-checked against `AI-Brain/AI-Citizen-Design.md`. **No symbol claim is invented.**
Build state at time of scan: `go build ./...` → EXIT 0 (GREEN). `theme_engine.go` is
already implemented (Act / nft-seduction-code-agent-2-2) and wired into `newLobby`.

## Scope note
`AI-Brain/AI-Citizen-Design.md` is 187 lines: it carries the **detailed §27** plus the
**§16 Q-table** where **Q1–Q26** summarise the locked §3–§26 lineage (AI citizens, item
shops, EMO scar, rival matrix, Dorks/Nuggets, §23 NFT-everything, §24 local LLM, §25 web-3D,
§25.5 dual-mode, §26 treasure/event matrix). The deep symbol table below therefore focuses on
the concrete symbols named in §27 + the four aggregator focus areas (§23.5, §27.7, §24.5/§24.6,
§25), plus a representative sampling of the §3–§26 lineage mechanics with code-presence checks.

Legend: PRESENT = symbol found in `.go` with file:line | MISSING = design-only, no code match.

---

## §27 Theme-Binding (detailed section — full symbol sweep)

| Specified (§ref) | Symbol | Status | Location / Notes |
|---|---|---|---|
| §27.1 | `ComputeThemeVector(wallet)` | PRESENT | `theme_engine.go:128` |
| §27.1 | `ThemeVector{Tone,Element,Intensity,Entropy}` | PRESENT | `theme_engine.go:72` |
| §27.1 | Cosmetic `MoodTag` input (§23.5 registry) | MISSING | only deferred stub `MoodTag int = 0` at `theme_engine.go:78`; no `MoodTag` source in code |
| §27.1 | `CalculateTotalPortfolioValue(wallet)` (holdings term) | PRESENT | `entity_investment_service.go:362` |
| §27.2 / §27.2.1 | `MarketWeather` / `WeatherState` | PRESENT | `theme_engine.go:535` / `theme_engine.go:99` |
| §27.2.1 | `rumorMultiplier` (market distortion) | PRESENT | `market_service.go:361` |
| §27.3 | Citizen-Gravity term in signature | PRESENT | `theme_engine.go:325-327` (reads `AICitizen.Reputation` × `(AttachmentTier+1)`) |
| §27.3 | `GetAllCitizens()` (gravity source) | PRESENT | `ai_citizen_engine.go:300` |
| §27.4 | `ComputeWorldDynamicsSignature(region)` | PRESENT | `theme_engine.go:301` |
| §27.4 | `WorldDynamicsSignature` (10 fields) | PRESENT | `theme_engine.go:83` |
| §27.4 | 10 `W_*` weights (locked) | PRESENT | `theme_engine.go` const block (W_MARKET_VITALITY=45000 … W_ENTITY_LEGITIMACY=40000) |
| §27.4 | `DetectRivalries` collision reuse | PRESENT | `rivalry_engine.go:119` (base engine) |
| §27.5 | `OutcomeBias(v, base)` | PRESENT | `theme_engine.go:236` (BIAS_MAX=15%) |
| §27.5 | Battle payout hook | PRESENT | `battle_service.go:1604` (comment hook; resolves via `l.OutcomeBias` on demand) |
| §27.5 | Treasure claim hook | PRESENT | `seasonal_event_engine.go:145` (EnterUserEvent precompute) |
| §27.5 | Breeding hook | PRESENT | `asset_life_engine.go:61` (BreedPet comment hook) |
| §27.5 | Rivalry resolution hook | PRESENT | `rivalry_engine.go:203` (`ResolveRivalry` calls `ComputeWorldDynamicsSignature`) |
| §27.6 | `ThemeBinding{Wallet,AssetID,Slot,Locked}` | MISSING | no `type ThemeBinding`, `BindThemeAsset`, or `isThemeLocked` anywhere in `.go` |
| §27.6 | Lock enforcement on `TransferBundleItems`/`ProcessSecondarySale` | MISSING | swap-only re-bind contract not implemented (only scoring + hooks exist) |
| §27.7 | `FaithCoherence` (real data structure) | **PRESENT (2026-09-05)** | `theme_engine.go:90` — `computeRegionFaithCoherence` reads `RitualsDone` from AI citizens and `Club.Type:"Faith"` clubs |
| §27.7.1 | `DomesticCoherence` (real data structure) | **PRESENT (2026-09-05)** | `theme_engine.go:91` — `computeRegionDomesticCoherence` reads `BondGraph` (WifeWallet/BotChildIDs/PetIDs) |
| §27.7.1 | `AICitizen.BondGraph` / `WifeWallet` / `LoverWallet` / `BotChildID` / `AICharPetID` | PRESENT | `ai_citizen_engine.go:49-61` — `BondGraph` struct with `WifeWallet`/`LoverWallet`/`BotChildIDs`/`PetIDs` |
| §27.7.2 | `RumorCoherence` / `EconomicPerk` | PARTIAL | `rumorMultiplier` PRESENT (`market_service.go:361`); `TaxHavenExpiresAt` PRESENT (`common_types.go:56`); but signature terms not yet wired into `ComputeWorldDynamicsSignature` |
| §27.7.3 | `EntityLegitimacy` (real data structure) | **PRESENT (2026-09-05)** | `theme_engine.go:94` — `computeRegionEntityLegitimacy` reads `Certified`/`BlackMarketAdopted` |
| §27.7.3 | `BirthCertID` / `BlackMarketAdopted` / `Certified` | PRESENT | `ai_citizen_engine.go:50-52` |
| §27.7.3 | own-wallet check at spawn + every transfer | PARTIAL | `SpawnAI` exists (`ai_citizen_engine.go:185`) but no explicit own-wallet mandate assertion; no transfer-ledger check found |
| §27.8 | `ModifyBondedAsset(wallet, assetID)` owner+holder guard | MISSING | no such function; `BondedAsset` type itself absent (see §23.5) |
| §27 routes | `GET /api/theme/vector` | PRESENT | `server.go:791` → `handleThemeVector` (`theme_engine.go:667`) |
| §27 routes | `GET /api/market/weather` | PRESENT | `server.go:792` → `handleMarketWeather` (`theme_engine.go:686`) |
| §27 routes | `GET /api/rivalry/world-dynamics` | PRESENT | `server.go:793` → `handleWorldDynamics` (`theme_engine.go:702`) |
| §27 wiring | `InitThemeEngine()` | PRESENT | `theme_engine.go:114` + call `server.go:269` (after `InitRivalryEngine`) |

---

## FOCUS AREA 1 — §23.5 Asset Registry (BondedAsset / SkinNFT / MoodTag / ThemeBinding)

| Specified (§ref) | Symbol | Status | Location / Notes |
|---|---|---|---|
| §23.5 | `BondedAsset` (creator/owner/holder ledger) | MISSING | 0 matches in `.go` |
| §23.5 | `SkinNFT` | MISSING | 0 matches (only referenced in design docs) |
| §23.5 | `BackgroundNFT` / `BoardNFT` / `ButtonNFT` / `AppearanceNFT` / `AudioPackNFT` | MISSING | 0 matches |
| §23.5 | `MoodTag` (author-declared at mint) | MISSING | no mint-time registry; only deferred `MoodTag int` in `theme_engine.go:78` |
| §23.5 | `CreatorStoreProduct` (existing storefront) | PRESENT | `creator_store_service.go:14` (CategoryAsset/Cosmetic exist, `:73`/`:76`) |
| §23.5 | `ProcessSecondarySale` (royalty path) | PRESENT | `creator_store_service.go:277` |
| §23.5 | `AuctionService.TransferBundleItems` | PRESENT | `auction_service.go:420` |

> CONCLUSION: §23.5 registry is **design-only**. This is the upstream blocker for §27.1
> cosmetic MoodTag input and §27.6 lock enforcement (both currently stub/no-op).

---

## FOCUS AREA 2 — §27.7 Faith / Domestic / Legitimacy DATA STRUCTURES

> **SUPERSEDED 2026-09-05:** All three signature terms are now WIRED and reading real data.
> - `FaithCoherence` — `computeRegionFaithCoherence` reads `RitualsDone` from AI citizens + Faith clubs
> - `DomesticCoherence` — `computeRegionDomesticCoherence` reads `BondGraph` fields
> - `EntityLegitimacy` — `computeRegionEntityLegitimacy` reads `Certified`/`BlackMarketAdopted`

| Specified (§ref) | Symbol | Status | Location / Notes |
|---|---|---|---|
| §27.7 | `FaithCoherence` | **WIRED (2026-09-05)** | `theme_engine.go` `computeRegionFaithCoherence` — reads `RitualsDone` from AI citizens and `Club.Type:"Faith"` clubs |
| §27.7.1 | `DomesticCoherence` | **WIRED (2026-09-05)** | `theme_engine.go` `computeRegionDomesticCoherence` — reads `BondGraph` (WifeWallet/BotChildIDs/PetIDs) |
| §27.7.1 | `BondGraph` struct | PRESENT | `ai_citizen_engine.go:58` — `WifeWallet`/`LoverWallet`/`BotChildIDs`/`PetIDs` |
| §27.7.3 | `EntityLegitimacy` | **WIRED (2026-09-05)** | `theme_engine.go` `computeRegionEntityLegitimacy` — reads `Certified`/`BlackMarketAdopted` |
| §27.7.3 | `BirthCertID` / `Certified` / `BlackMarketAdopted` | PRESENT | `ai_citizen_engine.go:50-52` |
| §27.7.3 | AI citizen marriage/breeding/pets | **IMPLEMENTED (2026-09-05)** | `ai_citizen_engine.go` — `executeMarriage`/`executeBreeding`/`executePetAdoption` in `BehavioralTick` |
| §27.7.3 | Faith system seeding | **IMPLEMENTED (2026-09-05)** | `server.go` `newLobby()` calls `InitializeFaithReligions(l)` — 24 religions seeded |
| §27.7.3 | Church storefront | **IMPLEMENTED (2026-09-05)** | `faith_church.go` — `OpenChurch`/`AddMember`/`AddItem`/`PerformRitual`/`Save`/`Load` |

---

## FOCUS AREA 3 — AI Civilization Backend (NEW — IMPLEMENTED 2026-09-05)

| Dynamic | Method | Status | Location |
|---|---|---|---|
| Faith Rituals | `executeFaithRituals` | IMPLEMENTED | `ai_citizen_engine.go` |
| Marriage | `executeMarriage` | IMPLEMENTED | `ai_citizen_engine.go` |
| Breeding | `executeBreeding` | IMPLEMENTED | `ai_citizen_engine.go` |
| Pet Adoption | `executePetAdoption` | IMPLEMENTED | `ai_citizen_engine.go` |
| Justice Enforcement | `executeJusticeEnforcement` | IMPLEMENTED | `ai_citizen_engine.go` |
| Employment | `executeEmployment` | IMPLEMENTED | `ai_citizen_engine.go` |
| Rivalry | `executeRivalry` | IMPLEMENTED | `ai_citizen_engine.go` |
| Market Activity | `executeMarketActivity` | IMPLEMENTED | `ai_citizen_engine.go` |
| Career Progression | `executeCareerProgression` | IMPLEMENTED | `ai_citizen_engine.go` |
| Event Hosting | `executeEventHosting` | IMPLEMENTED | `ai_citizen_engine.go` |
| All wired into BehavioralTick | — | IMPLEMENTED | `ai_citizen_engine.go` `BehavioralTick` |
| HTTP Routes | `/api/ai/citizens/*` | IMPLEMENTED | `server_main.go` |
| Persistence | `SaveCitizens`/`LoadCitizens` | IMPLEMENTED | `ai_citizen_engine.go` gzip-backed |
| Dev Seed | 2 citizens on boot | IMPLEMENTED | `server.go` `newLobby()` |
