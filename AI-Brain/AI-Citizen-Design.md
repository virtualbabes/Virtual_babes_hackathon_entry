
---

## §27 Theme-Binding: Skin → World → Outcome Coupling (Brendan — LOCKED Q31)

A player's **menu skin + all custom-assigned assets** are inferred into a single **Theme Vector** that drives the aesthetic **and the behavioral bias** of the 3D world they have built/occupied. The theme is not cosmetic-only: it *dynamically shifts the natural world's influence*, and **good themes yield good outcomes, bad themes yield bad outcomes** across many systems. Bound NFTs are locked from play unless re-bound through the entity market; modification requires the entity to be **both owner and holder**.

### 27.1 Theme Vector (inference) — full analytic range

- The Theme Vector is inferred from the player's **entire analytic profile**, not just cosmetics. Inputs (all deterministic, PILLAR 2 — no cloud):
  - **Cosmetic loadout** — applied `SkinNFT` (§23.5) + assigned `BackgroundNFT`/`BoardNFT`/`ButtonNFT`/`AppearanceNFT`/`AudioPackNFT`, each carrying a `MoodTag`.
  - **Holdings weight (entity market value — MAJOR signal):** the player's **entity market value** = `Lobby.CalculateTotalPortfolioValue(wallet)` (entity_investment_service.go:362) — total valued equity across entity shares + AMM positions + `MarketTokens` + claimed yield. This is the single strongest holdings signal: affluent market players project ORDER/MACHINE (structured affluence) and high `intensity`; depleted/isolated holders project VOID. Also folds in `PlayerStats.Inventory` (item mass) and `Salary` (career earnings). Entity market value dominates the holdings sub-score.
  - **Social status** — `Reputation`, `Mojo` (social standing for club unlocks), `SocialRank` ("Nobody"→"Icon"), `Relationships` (name→score 0-100 mean). High social capital → BENEVOLENT/ORDER lean; pariah/negative relationships → MALEVOLENT lean.
  - **Disposition** — `Nurturing` vs `Cunning` modifier pair, `WantedLevel` (risk factor). High Nurturing → NATURE/BENEVOLENT; high Cunning + WantedLevel → CHAOS/MALEVOLENT. **Entity-market conduct** also feeds disposition: a player whose held entities are `IsDividendFrozen` (PILLAR 3 justice counter-play, club_service.go) signals malpractice → MALEVOLENT lean; active yield compounding (`TotalClaimed`/`CumulativeYieldPerShare`) signals constructive ORDER.
  - **Standing** — `Achievements`, `Wins`, `JobRole`/`EmployerClubID` (org affiliation). Affiliation to a territory/region (§15/§25.8) ties the theme to that region's capital theme (§25.9).
- **Theme Vector** `T = (tone, element, intensity, entropy)`:
  - **tone** ∈ {BENEVOLENT, NEUTRAL, MALEVOLENT} — weighted vote across all input mood signals (cosmetic `MoodTag`s + social/disposition lean).
  - **element** ∈ {ORDER, CHAOS, NATURE, MACHINE, VOID} — dominant axis from holdings/social/disposition mix.
  - **intensity** = normalized composite of holdings weight + social status + asset rarity (higher ⇒ stronger world-shift).
  - **entropy** = conflict across inputs (e.g. benevolent cosmetics but malevolent social standing) ⇒ amplified *variance* rather than directional bias.
- Computed in `ThemeEngine.ComputeThemeVector(wallet)` from `PlayerStats` + active skin bindings. Recomputed on any profile/asset change (deterministic integer math).

### 27.2 World Influence (3D, §25) — natural flow fed by the full profile

- The §25 web-3D explorer reads `T` and the player's live analytics to drive the **natural world's flow and feedback**:
  - **Aesthetic / climate:** palette, lighting, fog, weather (§25.7) shift by element+tone. Benevolent/Nature → bright, fertile, calm; Malevolent/Void → dark, corrupted, hazardous. **Holdings weight** modulates world richness (affluent players see more developed/orderly regions; depleted players see sparse/chaotic ones). **Social status** modulates NPC deference/ambient civility.
  - **Spawn tables / ecology:** ambient animal + wild-bot (§25.7, §26) and NPC temperament skew by tone; **disposition** (`Nurturing`/`Cunning`) biases whether creatures are tamable vs hostile. Region ecology reflects the region capital's theme (§25.9) blended with the player's own.
  - **Feedback loop (immersion):** the world *responds* to the player's standing in real time — reputation gains brighten the world; wanted-level spikes darken it; relationship mean shifts NPC greetings. This is the "bond to the user": the 3D world is a mirror of their full analytic state, not a static skin.
  - **Region mood** = client-side derived field from `T` + region owner's theme. No server-side world simulation — presentation + spawn-weighting via `/api/regions`.

### 27.2.1 Market-as-Weather (the whole entity market → climate) (Brendan)

The **entire entity market's state is rendered as the 3D world's weather** — a direct, readable manifestation of live market mechanics (§verified: entity_investment_service.go, market_service.go, backend_types.go `EntityMarketNode`, club_service.go). Not just anti-whale signals — the *full* market breathes as sky. Deterministic mapping (PILLAR 2), no cloud, client-side derived from the player's held nodes + global market nodes.

- **Reserve depth / liquidity** (`ReserveBalance` per node) → **sky clarity & pressure.** Deep reserves = clear, high-pressure, bright; thin/draining reserves = storm fronts & low visibility. The market's "air."
- **Share supply elasticity** (`TotalSharesIssued` vs `ReserveBalance`, spot price `= Reserve/(Shares×Ratio)×repMult`) → **wind / turbulence.** Tight supply + high price = gusty, charged air; loose supply = still, calm.
- **Dividend pool & yield** (`DividendPoolMicro`, `CumulativeYieldPerShare` growth) → **warmth / sunlight.** Compounding yield = warm, golden light; stagnant pool = cold, dim.
- **Dividend freeze (justice)** (`IsDividendFrozen`, PILLAR 3 counter-play) → **lightning / static storm** over that entity's region.
- **Concentration / monopoly pressure** (stake approaching the **25% per-entity cap**, `maxEntityInvestment = 25% of CalculateTotalPortfolioValue`) → **heavy fog / whale-pressure front.**
- **Whale slippage events** (quadratic penalty `1 + (units/supply)²×5`) → **pressure-drop squalls**; penalty paid = squall intensity.
- **Rumor manipulation** (`rumorMultiplier ≠ 1`, market_service.go:361) → **aurora / static distortion** in the sky.
- **Trade flow / throughput** (buy+sell volume via `STOCK_BUY`/`STOCK_SELL` audit) → **precipitation & currents** — active markets = rain/rivers of commerce; dead markets = drought / dead-calms (also triggered if `faucetBalance < payout` illiquidity guard fires).
- **Reputation coupling** (node `Reputation` lifts spot price) → **altitude of light** — reputable entities glow from above; pariah entities sit under bruised skies.

**Market Disasters (severity tier) — macro-catastrophe layer:** when market signals cross extreme thresholds, the climate escalates into **natural disasters**, scaled by severity (PILLAR 2 integer thresholds). The world *ends* where the market collapses:

- **Ocean tides / flood isolation** — triggered by **reserve collapse** (`ReserveBalance` of a held node falling below a hard floor, e.g. <10% of its peak) or **mass sell-off** (`STOCK_SELL` throughput spiking while `faucetBalance` tightens). Rising tides flood low regions; isolated territories (§25.8) become cut off until reserves recover. Severity = depth of reserve drawdown.
- **Wildfires** — triggered by **yield death-spiral**: `DividendPoolMicro` stagnant + `CumulativeYieldPerShare` flat while `TotalSharesIssued` keeps climbing (issuance without backing) → tinder-dry conditions → wildfire sweeps the region. Also fed by **rumor-manipulation heat** (`rumorMultiplier` distortion) acting as arson risk.
- **Volcanic eruptions** — triggered by **monopoly pressure / over-concentration**: a single wallet or entity crossing toward the **25% cap** (or a node's `ReserveBalance` super-concentrated in one holder) builds "magma pressure"; at threshold the entity's region erupts, scorching the map. The ultimate anti-whale spectacle — the market's pressure literally vents as lava.
- **Droughts / dead-calms** — triggered by **illiquidity guard firing** (`faucetBalance < payout` reject, market_service.go:477) or sustained zero trade flow → rivers of commerce dry up; regions wither.
- **Lightning / static storms** — `IsDividendFrozen` justice action (PILLAR 3) over a region → sanctioned skies.
- **Aurora / reality distortion** — active `rumorMultiplier ≠ 1` manipulation → sky tears with false-color aurora (market perception divorced from fundamentals).

Disaster severity is **deterministic** from the same `EntityMarketNode` fields (no RNG, PILLAR 2). Regions can be **rendered uninhabitable / isolated** until the underlying market heals (reserves refill, concentration eases, freeze lifts) — making the player's economic conduct have *physical-world consequence*. This is the peak of the "bond to the user": their market behavior is the weather, the tides, the fire, the volcano.

Weather = **client-side synthesis** of all held + global `EntityMarketNode` state; doubles as an **at-a-glance market-health HUD** — players *feel* the market's integrity and vitality as atmosphere. A region's base weather preset (§25.7 buyable/deployable) is **modulated/overridden by this market-health weather layer**, so the economy literally weathers the world. No server-side simulation; pure presentation + signal read via `/api/regions` (nodes serialized) or a new `/api/market/weather` projection.

### 27.3 AI-Citizen Population Gravity (Brendan) — citizen weight themes the social world + attracts users

The aggregate **weight of a user's owned AI citizens** (§3/§15) themes the **social and populational natural world** of the region they occupy, and acts as a **gravity well that attracts more users** to that region. AI citizens are not just economic actors — they are the *population* that makes a region feel alive and worth visiting.

- **Citizen Weight** `W_citizen(owner) = Σ_over_owned [ (1 + Tier) × (1 + AttachmentTier/3) × (1 + Reputation/100) × (1 + BusinessCount×0.1) ]`, read from `AICitizen` fields (`Tier` 0–6, `AttachmentTier` 0–3, `Reputation` −100..+100, `BusinessCount`, `OriginWallet` = owner), per region (citizens carry a `Region` tag, cap = `1 + regionIndex` per §15).
- **Social world theming:** high `W_citizen` in a region → the 3D social layer renders **denser, more civil, more prosperous** populations — more NPCs milling, friendlier ambient interaction, brighter civic lighting, populated plazas. Low `W_citizen` → sparse, withdrawn, dull social spaces. This is the "living city" effect driven by who has invested citizens there.
- **User attraction (gravity):** regions with high aggregate `W_citizen` (especially high-tier, high-attachment, high-reputation citizens) project an **attraction field** — the leaderboard hub (§25.9) and region directory surface them as "thriving" destinations, and the menu's region-warp (§25.5) nudges new players toward high-gravity regions. More users → more commerce → higher entity-market value in that region (closing the loop with §27.1/§27.2.1).
- **Owner's own theme tie-in:** a user's personal `Theme Vector` (§27.1) is *boosted* by their `W_citizen` — owning a strong citizen population reinforces BENEVOLENT/ORDER tone (established, civic) and raises `intensity`. Their citizens become an extension of their world-bond.
- **Deterministic (PILLAR 2):** no RNG; computed from `aiEngine.GetAllCitizens()` filtered by `OriginWallet` + `Region` (client or server-side aggregation; serialized into `/api/regions` view). Cross-links §15 (AI citizens), §25.8/§25.9 (region/leaderboard gravity), §25.10 (citizen-value ladder — user-owned citizens are the top rung).


### 27.4 World-Dynamics Rivalry Matrix (Brendan) — rivaling worlds on vitality, not just assets

The world-dynamics signals from §27.1–§27.2.1 compose into a **rivalry matrix** that scores each region's *living-world vitality* and pits regions/worlds against each other — extending §25.10 (which rivals on asset-type + citizen-value) with a **dynamic, time-varying** dimension. Worlds compete on health, not just holdings.

- **World-Dynamics Signature** `WD(region)` augments the §25.10 `AssetSignature` with vitality terms:
  - `MarketVitality` — derived from §27.2.1: deep reserves + compounding yield + healthy trade flow = high; freezes / illiquidity / reserve collapse = low (the inverse of disaster severity). Weighted `W_MARKET_VITALITY`.
  - `CitizenGravity` — the §27.4 `W_citizen` aggregate for the region. Weighted `W_CITIZEN_GRAVITY`.
  - `ThemeCoherence` — low `entropy` in §27.1 (stable, non-conflicting theme) = high coherence; high entropy (chaotic identity) = low. Weighted `W_THEME_COHERENCE`.
  - `EventDynamics` — the **world-events layer** (§26): regions with active, well-attended events (Treasure Hunts, Search-Rescues, Gang-Bashings, Wild Bot Hunts, Pet Breeding Shows) score high *vitality*; regions with stale/empty event calendars score low. Event *type* modulates tone — Search-Rescue/Pet-Show = BENEVOLENT lean; Gang-Bashing/Wild-Bot-Hunt = CHAOS lean (still vitality-positive if populated). Weighted `W_EVENT_DYNAMICS`.
  - `ProfileImpact` — the **net profile effect** of a region's dynamics on its participants: positive flows (reputation gains, mojo growth, salary/employment via §15 careers, healed relationships) raise vitality; negative flows (wanted-level spikes, bans, relationship decay, reputation loss) lower it. Derived from `PlayerStats` deltas (Reputation, Mojo, SocialRank, WantedLevel, Relationships) aggregated over the region's population. Weighted `W_PROFILE_IMPACT`.
- **Positive / Negative effect flow (the feedback that bites):** the matrix's verdict is not cosmetic — it pushes **concrete profile effects** back onto players:
  - **Positive (high-vitality region wins):** residents gain a `VITALITY_BONUS` — reputation +mojo accrual, lowered effective wanted-level decay, relationship repair, and an `OutcomeBias` (§27.4) nudge toward good outcomes. The winning region's weather/gravity renders dominant (vitality crown, §27.3 prize).
  - **Negative (low-vitality / losing region):** residents accrue `VITALITY_PENALTY` — reputation stagnation, mojo drain, slower salary accrual, relationship decay, and an `OutcomeBias` nudge toward bad outcomes. Disaster weather (§27.2.1) compounds the penalty (tides/fires/volcanoes already degrade the space).
  - Effects are **deterministic and bounded** (PILLAR 2 integer math, capped per cycle) — no snowballing beyond `BIAS_MAX`. Applied at the same resolution hooks as §27.4 (combat, treasure, breeding, rivalry) so world-standing and personal profile move together.
  - `OutcomeBias` (§27.3) folded in as a vitality modifier (good-outcome regions trend upward).
- **New weights (added to §25.10.1 calibration table):** `W_MARKET_VITALITY = 45000`, `W_CITIZEN_GRAVITY = 55000`, `W_THEME_COHERENCE = 25000`, `W_EVENT_DYNAMICS = 30000`, `W_PROFILE_IMPACT = 50000`, `W_FAITH_COHERENCE = 40000`, `W_DOMESTIC_COHERENCE = 35000`, `W_RUMOR_COHERENCE = 20000`, `W_ECON_PERK = 30000`, `W_ENTITY_LEGITIMACY = 40000` (integer, PILLAR 2). Composed with existing `W_*` asset weights so the total region score = asset score + world-dynamics score.
- **Rivalry detection:** reuses §25.10 `DetectRivalries` collision logic (`RIVAL_COLLISION_THRESHOLD = 40000`) but on the *combined* signature — two regions with overlapping vitality profiles auto-duel. Cross-region = Capital-vs-Capital (§25.9); inside = Territory-vs-Territory.
- **Prize (4-part, §25.10.1):** winner region gets the showcase buff + resident cache + governor rep + worker bonus — **plus** a **vitality crown**: the winning region's weather/gravity renders *dominant* across the leaderboard hub (§25.9) for the rivalry cycle, attracting even more users (closing the §27.4 gravity loop).
- **Implementation:** `ThemeEngine.ComputeWorldDynamicsSignature(region)` (server-side, reads `tokenSinkRouter.MarketNodes`, `aiEngine.GetAllCitizens()`, `leaderboard` theme state) feeds `rivalryEngine`; serialized into `/api/regions` + a new `/api/rivalry/world-dynamics` projection. Deterministic (PILLAR 2) — no RNG.
- **Cross-links:** §25.10 (base rivalry + weights), §27.1 (theme vector/entropy), §27.2.1 (market-weather → vitality), §27.4 (citizen gravity), §25.9 (capital hub dominance).

### 27.5 Outcome Bias (the core mechanic) — full-profile weighted

- `ThemeEngine.OutcomeBias(T, profile) → int` in `[-BIAS_MAX, +BIAS_MAX]` (proposed ±15%). Positive for BENEVOLENT/ORDER/NATURE weighted by **social status + entity market value (holdings)**; negative for MALEVOLENT/VOID/CHAOS weighted by **wanted-level + cunning**. `entropy` converts directional bias into bounded **variance**. `ThemeEngine` reads `Lobby.CalculateTotalPortfolioValue(wallet)` for the holdings term.
- Applied as a multiplier/modifier hook at resolution points (good things → good outcomes, bad → bad):
  - **Combat / battles** (battle_service.go settlement) — theme bonus/penalty to payout or win-weight.
  - **Treasure caches** (§26.1) — reward magnitude + discovery chance shift.
  - **Breeding** (§26.4.1) — offspring trait quality skews good/bad.
  - **Rivalry** (§25.10) — signature score nudged by capital theme.
  - **Events** (§26.2) — participant outcome skew.
- **Mechanic:** *good things equal good outcomes, bad things equal bad outcomes.* A player who builds a benevolent, ordered world is rewarded by the systems; a malevolent, void world is punished. This is the immersive "bond to the user" — their aesthetic choices are consequential, not decorative.

### 27.6 NFT Lock Rule (theme-bound assets removed from play)
- When an asset (`SkinNFT` / `BackgroundNFT` / etc.) is assigned to a **theme binding** (i.e. it is actively contributing to a player's `Theme Vector`), it is **locked out of normal play/use** — it cannot be equipped elsewhere, traded freely, or consumed — **unless** it is **re-bound through the entity market** (§3/§14/§21 transfer paths: `auction_service.TransferBundleItems`, `CreatorStore.ProcessSecondarySale`).
- **Re-activation via trade-off:** an NFT locked in a theme can be **swapped out** only by trading it for *another* NFT through the entity market — you cannot simply unbind and reuse; you must exchange. The incoming NFT takes the slot; the outgoing one leaves the theme and re-enters tradeable state. This enforces *commitment*: your world's character is a standing economic position, not a free toggle.
- Implementation: `ThemeBinding{ Wallet, AssetID, Slot, Locked bool }` on `Lobby`; `Locked=true` blocks `TransferBundleItems`/equip except via the market swap path which sets `Locked=false` on the outgoing asset atomically.

### 27.7 Religions · Rituals · Godly Acts (Brendan) — the spiritual world-dynamics layer

Players and AI citizens may form **religions**, perform **rituals**, and commit **godly acts** — a spiritual/awe dimension of the world that themes regions and feeds the rivalry matrix + outcome bias (§27.3/§27.4) alongside market, citizen, event, and profile signals. Deterministic (PILLAR 2), no cloud — faith here is *emergent social consensus*, not a real deity.

- **Religion** — a player-founded belief group tied to a region/territory (§25.8), carrying a `DogmaTag` (e.g. ORDER/BENEVOLENT, VOID/MALEVOLENT, NATURE, MACHINE) that contributes to the region's `Theme Vector` tone (§27.1) like a cosmic-scale mood. Membership = wallet list; adherence measured by ritual participation + godly-act count.
- **Rituals** — scheduled or triggered ceremonies (daily devotion, seasonal rite, sacrifice-of-assets, pilgrimage to the leaderboard hub §25.9). Each completed ritual accrues **FaithCoherence** for the religion and a small `VITALITY_BONUS` (§27.3) to participants. Rituals consume/burn bonded assets (§23) as offerings — tying the spiritual layer to the entity market (offerings can route via `RouteCriminalTax` to a religion treasury, PILLAR 2).
- **Godly Acts** — unambiguous good deeds: rescue (§26 Search-Rescue), charity (yield donation), protecting low-tier citizens, healing relationships. Each godly act raises the actor's `Reputation` (§27.3 `ProfileImpact`) AND the religion's/region's **Awe** score. Malevolent mirror = "blasphemy" acts (targeted griefing, desecration) that lower Awe and raise `WantedLevel`.
- **Faith Signals in the World (§27.2/§27.2.1):** high regional Awe → **cathedral-like geometry, golden ambient light, calm weather, pilgrimage trails**; schism/low Awe → **bleak, blighted** regions. The market-weather layer (§27.2.1) is *modulated* by Awe (reverent regions weather milder disasters).
- **Rivalry matrix term:** `FaithCoherence` added to the §27.3 World-Dynamics Signature — weight `W_FAITH_COHERENCE = 40000`. Religions rival each other (heresy wars) and regions rival on Awe, exactly like the asset/citizen/vitality duels (§25.10 collision logic). A religion that wins the cycle grants its adherents a region-wide `VITALITY_BONUS` (§27.3 positive flow).
- **Deterministic scoring:** `FaithCoherence(region) = Σ_members (rituals_done×10 + godly_acts×5 − blasphemy×8)`, bounded; `Awe(region) = clamp(FaithCoherence / population, 0, 1000)`. No RNG. Serialized into `/api/regions` + `/api/rivalry/world-dynamics`.
- **Cross-links:** §27.1 (tone via DogmaTag), §27.2/§27.2.1 (world rendering + weather modulation), §27.3 (signature term + bonus/penalty flow), §25.8 (territory religion), §25.9 (pilgrimage hub), §26 (Search-Rescue = godly act), §23 (offerings burn bonded assets), §25.10 (rivalry collision).

### 27.7.1 Domestic & Bot-Family Bonds (Brendan) — wives, lovers, bot children, AI "pets", children's events

The **domestic/social-bond layer** of AI characters (and players): wives, lovers, bot children, AI-character "pets" (distinct from §26.4 PetNFT creatures), and **children's events**. This is the *intimate* social fabric of a region — it themes the world's warmth/civility and feeds the §27.4 rivalry matrix as a vitality signal, parallel to citizen gravity (§27.3) and faith (§27.7).

- **Bond graph:** each AI citizen (§3/§15, `AICitizen`) carries a `BondGraph` — spouse (`WifeWallet`/`LoverWallet`), children (`[]BotChildID`), and "pets" (`[]AICharPetID`). Bot children are derived AI citizens (lower `Tier`, `OriginWallet` = parent) — a lineage that extends the §15 citizen ladder. AI-character pets are companion bots (not §26.4 biological pets) bound to a citizen.
- **Children's events:** family-scale events — births/naming ceremonies, coming-of-age rites, play-dates, schooling — surface in the §26 event matrix as a `CHILDREN_EVENT` type, authored by the family/region (via shops §14/§21 or admin §26.2). These raise regional **warmth** and `Relationship` scores among participants.
- **World theming (§27.2):** regions rich in stable families + bot-children render **warmer, more civil, more "home-like"** — populated hearths, playgrounds, calmer NPC temperament. Fractured/loveless regions render cold and tense. The domestic layer modulates the same social-space rendering as citizen gravity, but at the *household* scale.
- **Rivalry matrix term:** `DomesticCoherence` added to the §27.4 World-Dynamics Signature — weight `W_DOMESTIC_COHERENCE = 35000`. Regions rival on *social health* (family stability, child welfare, bond-count) exactly like the asset/citizen/faith/vitality duels (§25.10 collision logic). A region that "wins" on domestic coherence projects an attraction field (§27.3 gravity) drawing families + new users.
- **Deterministic scoring:** `DomesticCoherence(region) = Σ_citizens (hasSpouse×8 + hasLover×4 + children×6 + aipets×3 + childrenEventsHeld×5)`, bounded; no RNG. Serialized into `/api/regions` + `/api/rivalry/world-dynamics`. Cross-links §15 (AI citizens/lineage), §26 (event matrix, `CHILDREN_EVENT`), §27.3 (gravity loop), §25.10 (rivalry).

### 27.7.2 Relationship Rumours & Economic Perks (Brendan)

The **social-rumour** and **economic-perk** layers of a region — both feed world-dynamics vitality and the §27.4 rivalry matrix.

- **Relationship rumours:** the existing `l.rumors` engine (market_service.go `rumorMultiplier`) is extended to a *social* rumour layer — players/citizens spread rumours about relationships, reputations, and holdings. A region with **high trust-rumour density** (verified, positive relationship gossip) raises `Relationship` means + `SocialRank`; a region riddled with **slander/defamation rumours** lowers them and raises `WantedLevel` risk. Rumours are deterministic (PILLAR 2): each carries a `Strength` + `ExpiresAt`; the social layer aggregates net sentiment per region.
- **Economic perks:** regional **economic perks** — tax havens (club_service.go `TaxHavenExpiresAt`), dividend yields (§27.2.1 `DividendPoolMicro`), fee rebates, governance grants (§25.10 `GOV_REP` prizes) — make a region *materially* more attractive. Perks = a `PerkScore` signal: regions that consistently pay yields + grant havens project prosperity (bright markets, busy plazas) and pull migration.
- **World theming (§27.2) + rivalry term:** rumour-sentiment + perk-score modulate the same social-space + market-weather rendering as §27.7.1; added to the §27.4 signature as `SocialRumorCoherence` (weight `W_RUMOR_COHERENCE = 20000`) and `EconomicPerkScore` (weight `W_ECON_PERK = 30000`). Regions rival on *social trust + economic generosity* via §25.10 collision logic; winning regions project a **trust halo** (attraction field, §27.3 gravity) drawing users seeking safe, prosperous homes.
- **Deterministic:** rumour net = `Σ (positive − slander) × Strength`, perk-score = normalized yield+haven+grant sum; bounded; no RNG. Serialized into `/api/regions` + `/api/rivalry/world-dynamics`. Cross-links market_service.go (`rumors`/`rumorMultiplier`), club_service.go (`TaxHavenExpiresAt`), §27.2.1 (yield), §25.10 (GOV_REP), §27.3 (gravity).

### 27.7.3 Entity Provenance — Birth-Certificates vs Black-Market Adoptions (Brendan)

The **legitimacy layer** for AI entities (§3/§15 `AICitizen`, including bot children from §27.7.1): every entity is either **birth-certified** (spawned through the compliant flow) or **black-market adopted** (acquired off-ledger / non-compliant). This is a hard compliance signal tied to the standing mandate: *an AI citizen must hold its OWN wallet and is forbidden from using a personal wallet; the check must be enforced.*

- **Birth-Certificate (`Certified`):** emitted by the legitimate spawn path (server.go `/spawn`, §15 `SpawnCitizen`). Proof that the entity was allocated its **own** wallet (`EntityWallet ≠ creator personal wallet`), registered on the single-source transfer ledger (§23.2), and paid the spawn fee through the faucet/treasury — NOT a personal wallet. Carries `BirthCertID`, `EntityID`, `EntityWallet`, `CreatorWallet` (who paid), `Tier`, `IssuedAt`. A bot child (§27.7.1) inherits certification from its parent's birth-certificate chain.
- **Black-Market Adoption (`BlackMarketAdopted`):** an entity acquired through any non-compliant path — (a) bound to a **personal wallet** (violates the mandate → flagged by the `own-wallet` check), (b) transferred **off-ledger** bypassing the §23.2 single-source royalty/transfer ledger, or (c) minted with no `BirthCertID`. Such entities are tagged `BlackMarketAdopted = true` and are **ineligible** for economic perks (§27.7.2), governance roles (§25.10 `GOV_REP`), and legitimate breeding (§26.4.1).
- **World theming (§27.2):** certified-heavy regions render **legitimate, institutionally bright** (civic halls, clean markets, perk kiosks). Black-market-heavy regions render **shady/underground** (back-alley renders, dim lighting, "off-ledger" tint) and elevate regional `WantedLevel` risk — a visible consequence of illegitimate adoption.
- **Rivalry matrix term:** `EntityLegitimacy` added to the §27.4 World-Dynamics Signature — weight `W_ENTITY_LEGITIMACY = 40000`. Regions rival on *legitimacy* (certified population vs black-market penetration) via §25.10 collision logic; certified regions project a **trust halo** (§27.3 gravity) and unlock perk eligibility, black-market regions accrue a `VITALITY_PENALTY` (§27.5 negative flow) + WantedLevel exposure.
- **Deterministic:** `EntityLegitimacy(region) = Σ_entities (Certified×10 − BlackMarketAdopted×12)`, bounded; the `own-wallet` enforcement check runs at spawn/transfer. No RNG. Serialized into `/api/regions` + `/api/rivalry/world-dynamics`. Cross-links §15 (spawn/own-wallet mandate), §23.2 (single-source transfer), §27.7.1 (bot-child lineage), §27.7.2 (perk eligibility), §27.8 (owner/holder modification), PlayerStats (`WantedLevel`).
- **Bot-Child → Local Model Promotion (Brendan):** a **birth-certified** bot child (§27.7.1 lineage) that reaches maturity can be promoted to run its own **Local model** — a per-user compiled LLM per §24.5/§24.6 (`setup_custom_quant_ornith.bat`, gated at `Level L ≥ 25`, capped at 50% GPU/CPU/RAM, output to the owner's Ollama model dir). This is the *legitimate* path to giving a bot child its own cognition/voice. **Black-market-adopted children are INELIGIBLE** — without a `BirthCertID` they cannot enter the compile pipeline, so only clean lineage gets a brain. This directly ties provenance (§27.7.3) to the local-LLM feature: **legitimacy = the right to think.**

### 27.8 Modification Rule (owner AND holder)
- Any modification of a **sold/assigned** asset (rename, re-skin, re-tag mood, re-bind) is permitted **only by the entity that is BOTH the owner AND the current holder**.
  - Owner = `BondedAsset.CreatorWallet` (or current owner-of-record from transfer ledger).
  - Holder = the wallet currently holding/equipping the asset (session bind).
  - If owner ≠ holder (asset is lent/consigned/in-market), **neither** may modify it unilaterally — only a joint action (both sign) mutates it. This prevents a seller from altering an asset mid-trade and prevents a holder from editing something they don't own.
- Enforcement point: `ModifyBondedAsset(wallet, assetID)` checks `owner==wallet && holder==wallet`; otherwise returns `ERR_NEEDS_OWNER_AND_HOLDER`.

### 27.9 Cross-links
- §23.5 (assets-as-NFT, skin/background/button classes), §25 (3D world, weather, spawn), §25.7 (living world), §25.9 (capital/leaderboard theme), §26 (treasure/events/breeding), §25.10 (rivalry), §3/§14 (entity market, bonded transfer), §23.2 (BondedAsset royalty/transfer single-source).
- **No cloud:** theme inference + outcome bias are integer/deterministic; the 3D client renders the bias locally. Constraint #7/#9 preserved.

### 27.10 Locked calibration (LOCKED Q31)

All five open questions resolved **as proposed** on lock:

1. **`BIAS_MAX = 15%`** — outcome-bias magnitude capped at ±15% of the base resolution value (range considered 5–25%; 15% chosen as the immersive-but-fair midpoint).
2. **`MoodTag` source = author-declared at mint** — each asset carries an explicit `MoodTag` (BENEVOLENT / NEUTRAL / MALEVOLENT) authored by the creator at mint time and stored on the §23.5 asset registry; the inference engine reads it (does not infer it).
3. **Theme scope = per-player + region inherits governor's** — a player's own world is themed by their own vector; a region's capital inherits the **governor's** theme per §25.9; non-capital territories inherit the region capital's theme.
4. **Bot children = derived `AICitizen` records** — bot children (§27.7.1) are full `AICitizen` records with `OriginWallet = parent` and reduced `Tier`; no separate `BotChild` type.
5. **Entity provenance = own-wallet check at spawn AND every transfer** — the §15 own-wallet mandate (`EntityWallet ≠ personal wallet`) is enforced at `/spawn` and at every §23.2 ledger transfer; black-market adoption (off-ledger / personal-wallet-bound / no `BirthCertID`) is flagged and ineligible for perks, governance, breeding, and local-model promotion.

- **Locked weights (§27.4 signature):** `W_MARKET_VITALITY=45000`, `W_CITIZEN_GRAVITY=55000`, `W_THEME_COHERENCE=25000`, `W_EVENT_DYNAMICS=30000`, `W_PROFILE_IMPACT=50000`, `W_FAITH_COHERENCE=40000`, `W_DOMESTIC_COHERENCE=35000`, `W_RUMOR_COHERENCE=20000`, `W_ECON_PERK=30000`, `W_ENTITY_LEGITIMACY=40000`.
- **Status: LOCKED** — ready for KEY 3.5 implementation (ThemeEngine).

*LOCKED Q31. Implementation authorized under KEY 3.5 / YOLO. Grounded in: §23.5 asset classes, §25/§25.7/§25.9 world systems, §26/§26.4.1/§25.10 resolution points, auction_service.go:420 + CreatorStore.ProcessSecondarySale transfer paths, PlayerStats.Inventory, §15 own-wallet mandate, §24.5/§24.6 local-model compile. Ready for ThemeEngine build.*

---

## §27.11 AI Civilization Backend (NEW — IMPLEMENTED 2026-09-05)

The full AI civilization backend is implemented in `ai_citizen_engine.go` and wired into `server_main.go`. This realizes the §27.7 faith and domestic systems by giving AI citizens autonomous behavioral drives that populate the data structures those systems read.

### BehavioralTick Extensions

All new behaviors are probabilistic (matching the existing `executeHeistPlanning` pattern) and gated by `Tier`:

| Dynamic | Method | Tier Gate | Probability | Effect |
|---|---|---|---|---|
| Faith Rituals | `executeFaithRituals` | ≥3 | Every tick | `RitualsDone++`, FaithCoherence+ |
| Marriage | `executeMarriage` | ≥Journeyman | 10% | `WifeWallet` set, DomesticCoherence+ |
| Breeding | `executeBreedAI` | ≥Expert, married, certified | 5% | `BotChild` spawned, DomesticCoherence+ |
| Pet Adoption | `executePetAdoption` | ≥Journeyman | 8% | `PetIDs+`, DomesticCoherence+ |
| Justice Enforcement | `executeJusticeEnforcement` | ≥Expert | 7% | Captures outlaws, treasury+, reputation+ |
| Employment | `executeEmployment` | EMPLOYED | Every tick | Salary from owned club |
| Rivalry | `executeRivalry` | ≥Journeyman | 4% | Reputation swing, treasury drain |
| Market Activity | `executeMarketActivity` | ≥Expert | 10% | Entity investment or black market |
| Career Progression | `executeCareerProgression` | Any | 3% | Tier++ (XP + reputation gated) |
| Event Hosting | `executeEventHosting` | ≥Master | 2.5% | Hosts entity events |

### HTTP Routes

| Method | Route | Description |
|---|---|---|
| POST | `/api/ai/citizens/spawn` | Spawn a new AI citizen |
| POST | `/api/ai/citizens/marry` | Marry two AI citizens |
| POST | `/api/ai/citizens/breed` | Breed a bot-child |
| POST | `/api/ai/citizens/adopt-pet` | Adopt a companion pet |
| POST | `/api/ai/citizens/progress` | Trigger career progression |
| GET | `/api/ai/citizens/list` | List all citizens (includes BondGraph) |
| GET | `/api/ai/citizens/stats` | Aggregate citizen stats |
| GET | `/api/ai/citizens/free-agents` | Free agents (unattached) |

### Data Flow

```
BehavioralTick (per citizen, per tick)
  → executeFaithRituals → RitualsDone++
  → executeMarriage → BondGraph.WifeWallet
  → executeBreeding → AICitizen (new), BondGraph.BotChildIDs
  → executePetAdoption → BondGraph.PetIDs
  → executeJusticeEnforcement → Treasury++, reputation++, WantedLevel--
  → executeEmployment → Treasury += salary
  → executeRivalry → Treasury--, reputation ±
  → executeMarketActivity → Entity investment OR black market
  → executeCareerProgression → Tier++ (XP + reputation gated)
  → executeEventHosting → EntityEvent created

Theme Engine (per region)
  → computeRegionFaithCoherence → Σ(rituals_done×10)
  → computeRegionDomesticCoherence → Σ(hasSpouse×8 + children×6 + pets×3)
  → computeRegionEntityLegitimacy → Σ(Certified×10 − BlackMarket×12)
  → WorldDynamicsSignature → feeds rivalry matrix + outcome bias
```

### Persistence

- `SaveCitizens` / `LoadCitizens` — gzip-backed, 15-minute snapshots
- `Save` / `Load` for religions and churches
- Citizens rehydrated on boot via `newLobby()` in `server.go`

---

## §16 Calibration & Question-Log

Canonical question/calibration ledger. Q1–Q30 summarize the locked design lineage (§3–§26 + §25.10); **Q31 is the Theme-Binding lock** added on 2026-08-29.
