//go:build !js && !wasm

package main

// theme_engine.go — Phase A of the ThemeEngine (§27, LOCKED Q31).
//
// Scope (KEY 3.5 / YOLO, autonomous):
//   - Per-player ThemeVector from REAL inputs only (entity holdings + PlayerStats).
//   - OutcomeBias: deterministic integer bias in [-BIAS_MAX, +BIAS_MAX] (±15%).
//   - ComputeWorldDynamicsSignature(region): 10 LOCKED weighted signals.
//   - MarketWeather(region): deterministic entity-market state -> climate + disaster tier.
//
// DEFERRED (NOT in scope, blocked on missing registries):
//   - FaithCoherence / DomesticCoherence / EntityLegitimacy: no real data structures
//     exist in code yet (only in the §27.7 design doc) -> scored 0, NOT fabricated.
//   - Bot-children / local-model compile (§24.5/§24.6): out of scope for Phase A.
//
// NOTE: Cosmetic MoodTag IS now sourced from the §23.5 BondedAssetRegistry
// (MoodTagForWallet, non-locked bound assets only, per §27.6) — wired in slice 3.
//
// PILLAR 2 compliance: all math is integer, deterministic, no RNG.

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ── §27.4 LOCKED signature weights (single source of truth) ────────────────────
const (
	W_MARKET_VITALITY   = 45000
	W_CITIZEN_GRAVITY   = 55000
	W_THEME_COHERENCE   = 25000
	W_EVENT_DYNAMICS    = 30000
	W_PROFILE_IMPACT    = 50000
	W_FAITH_COHERENCE   = 40000
	W_DOMESTIC_COHERENCE = 35000
	W_RUMOR_COHERENCE   = 20000
	W_ECON_PERK         = 30000
	W_ENTITY_LEGITIMACY = 40000
)

// BIAS_MAX is the §27.10 locked ±15% outcome-bias cap, expressed as a basis-point
// fraction (15/100) so it can be applied to an arbitrary integer base via integer math.
const BIAS_MAX = 15

// ThemeTone enumerates the §27.1 theme-vector tone axis.
type ThemeTone int

const (
	ToneVoid       ThemeTone = 0
	ToneMalevolent ThemeTone = 1
	ToneNeutral    ThemeTone = 2
	ToneBenevolent ThemeTone = 3
	ToneOrder      ThemeTone = 4
	ToneNature     ThemeTone = 5
	ToneMachine    ThemeTone = 6
)

// ThemeElement enumerates the §27.1 theme-vector element axis.
type ThemeElement int

const (
	ElementNone  ThemeElement = 0
	ElementFire  ThemeElement = 1
	ElementWater ThemeElement = 2
	ElementEarth ThemeElement = 3
	ElementAir   ThemeElement = 4
	ElementAether ThemeElement = 5
	ElementMachine ThemeElement = 6
)

// ThemeVector is the per-player (and region-inherited) deterministic theme projection (§27.1).
// All fields are integers to satisfy PILLAR 2 determinism.
type ThemeVector struct {
	Wallet    string      `json:"wallet"`
	Tone      ThemeTone   `json:"tone"`       // 0..6
	Element   ThemeElement `json:"element"`    // 0..5
	Intensity int         `json:"intensity"`  // 0..100
	Entropy   int         `json:"entropy"`    // 0..100 — directional variance spread
	MoodTag   int         `json:"mood_tag"`   // 0 = NEUTRAL (Go default); sourced from BondedAssetRegistry.MoodTagForWallet (non-locked bound assets, §27.6)
}

// WorldDynamicsSignature is the §27.4 ten-signal weighted region vitality projection.
// Each signal is the raw (unweighted) integer contribution; Score is the weighted total.
type WorldDynamicsSignature struct {
	Region            string `json:"region"`
	MarketVitality    int    `json:"market_vitality"`
	CitizenGravity    int    `json:"citizen_gravity"`
	ThemeCoherence    int    `json:"theme_coherence"`
	EventDynamics     int    `json:"event_dynamics"`
	ProfileImpact     int    `json:"profile_impact"`
	FaithCoherence    int    `json:"faith_coherence"`    // deferred -> 0 (no real source)
	DomesticCoherence int    `json:"domestic_coherence"` // deferred -> 0 (no real source)
	RumorCoherence    int    `json:"rumor_coherence"`
	EconomicPerk      int    `json:"economic_perk"`
	EntityLegitimacy  int    `json:"entity_legitimacy"`  // deferred -> 0 (no real source)
	Score             int64  `json:"score"`
}

// WeatherState is the §27.2.1 deterministic market-weather projection.
type WeatherState struct {
	Region      string `json:"region"`
	Climate     string `json:"climate"`     // CALM | BREEZY | STORMY | BLIGHTED
	DisasterTier int   `json:"disaster_tier"` // 0 (none) .. 3 (volcanic)
}

// ThemeEngine holds the per-lobby theme state. Reuses the Lobby.mutex RWMutex
// pattern established by rivalry_engine.go — no additional mutex needed because all
// ThemeEngine reads/writes go through Lobby.mutex.
type ThemeEngine struct {
	// Phase A: stateless computation over Lobby state. Caching deferred (see note
	// on computeThemeVectorLocked) to avoid RWMutex re-entrancy hazards.
}

// InitThemeEngine wires the ThemeEngine into a lobby. Mirrors InitRivalryEngine().
func (l *Lobby) InitThemeEngine() {
	l.themeEngine = &ThemeEngine{}
}

// clampInt is the integer clamp helper (mirrors rivalry_engine.go:clampInt, defined there).
// (declared in rivalry_engine.go; reused here without redefinition.)

// ComputeThemeVector derives a player's §27.1 theme projection from REAL inputs only.
//   - holdings weight: Lobby.CalculateTotalPortfolioValue(wallet) (entity_investment_service.go:362)
//   - social:    PlayerStats.Reputation / Mojo / SocialRank / Relationships
//   - disposition: PlayerStats.Nurturing / Cunning / WantedLevel
//   - standing:  PlayerStats.Achievements / Wins / JobRole / EmployerClubID
// Cosmetic MoodTag is sourced from the §23.5 BondedAssetRegistry.MoodTagForWallet
// (non-locked bound assets only, per §27.6); 0 (NEUTRAL) is the default when none bound.
// Deterministic integer math; no RNG.
func (l *Lobby) ComputeThemeVector(wallet string) ThemeVector {
	wk := normalizeWallet(wallet)

	l.mutex.RLock()
	stats, ok := l.leaderboard[wk]
	l.mutex.RUnlock()
	if !ok {
		// Unknown wallet: neutral, low-intensity default (deterministic).
		return ThemeVector{Wallet: wk, Tone: ToneNeutral, Element: ElementNone, Intensity: 0, Entropy: 0, MoodTag: 0}
	}
	return l.computeThemeVectorLocked(wk, stats)
}

// computeThemeVectorLocked is the lock-free core. Caller MUST hold l.mutex
// (read or write). Used by resolution hooks that already hold the lobby lock
// (e.g. finalizeMatchResultLocked, BreedPet) to avoid RWMutex re-entrant deadlock.
func (l *Lobby) computeThemeVectorLocked(wk string, stats PlayerStats) ThemeVector {
	// ── Holdings weight (entity market value, micro-VBV) ──
	holdings := l.CalculateTotalPortfolioValueLocked(wk) // uint64 micro-VBV (caller holds l.mutex)

	// ── Social sub-score ──
	socialRank := socialRankWeight(stats.SocialRank) // 0..30
	relSum := sumRelationships(stats.Relationships)  // 0..100 (capped)
	social := int(clampInt64(int64(stats.Reputation), 0, 100)) +
		int(clampInt64(int64(stats.Mojo), 0, 100)) +
		socialRank +
		relSum // 0..330

	// ── Disposition sub-score ──
	nurt := int(clampInt64(int64(stats.Nurturing), 0, 100))
	cun := int(clampInt64(int64(stats.Cunning), 0, 100))
	want := int(clampInt64(int64(stats.WantedLevel), 0, 100))

	// ── Standing sub-score ──
	standing := len(stats.Achievements) +
		int(clampInt64(int64(stats.Wins/10), 0, 50)) // 0..50 by wins/10
	if stats.JobRole != "" {
		standing += 10
	}
	if stats.EmployerClubID != "" {
		standing += 10
	} // 0..(len(achievements)+70)

	// ── Tone: benevolent vs malevolent axis ──
	good := nurt + social/4 + int(holdingsToScore(holdings)) // higher => benevolent/order
	bad := cun + want
	tone := ToneNeutral
	switch {
	case good >= 120 && want == 0:
		tone = ToneBenevolent
	case good >= 180:
		tone = ToneOrder
	case bad >= 120 && nurt == 0:
		tone = ToneMalevolent
	case bad >= 180:
		tone = ToneVoid
	case good > bad+40:
		tone = ToneNature
	default:
		tone = ToneNeutral
	}

	// ── Element: holdings-driven ──
	elem := ElementNone
	switch {
	case holdings >= 5_000_000_000: // >= 5,000 $VBV
		elem = ElementAether
	case holdings >= 1_000_000_000: // >= 1,000 $VBV
		elem = ElementMachine
	case holdings >= 250_000_000: // >= 250 $VBV
		elem = ElementEarth
	case holdings >= 50_000_000: // >= 50 $VBV
		elem = ElementWater
	case holdings > 0:
		elem = ElementFire
	default:
		elem = ElementNone
	}

	// ── Intensity: normalized blend of holdings + standing + social ──
	holdScore := int(holdingsToScore(holdings)) // 0..100
	intensity := (holdScore*2 + standing*1 + social/3) / 4
	intensity = int(clampInt64(int64(intensity), 0, 100))

	// ── Entropy: directional spread between good/bad forces ──
	entropy := absInt(good - bad)
	entropy = int(clampInt64(int64(entropy)/3, 0, 100))

	v := ThemeVector{
		Wallet:    wk,
		Tone:      tone,
		Element:   elem,
		Intensity: intensity,
		Entropy:   entropy,
		MoodTag:   0, // default; overridden below from §23.5 bonded assets (§27.6: locked excluded)
	}
	// ── Cosmetic MoodTag from §23.5 bonded assets (§27.6: locked assets excluded) ──
	if l.bondedAssets != nil {
		if mood, ok := l.bondedAssets.MoodTagForWallet(wk); ok {
			v.MoodTag = mood
		}
	}
	return v
}

// cacheThemeVector is intentionally omitted in Phase A: the ThemeVector is cheap and
// deterministic to recompute, and caching introduces RWMutex re-entrancy hazards at
// resolution points that already hold the lobby lock. Re-add only with a lock-free store.

// OutcomeBias returns a deterministic integer bias in [-BIAS_MAX, +BIAS_MAX] percent
// of the base resolution value, per §27.5. Positive for BENEVOLENT/ORDER/NATURE weighted
// by social status + entity market value (holdings); negative for MALEVOLENT/VOID/CHAOS
// weighted by wanted-level + cunning. Entropy widens the effective variance bound but the
// returned delta stays within ±BIAS_MAX% of base.
func (l *Lobby) OutcomeBias(v ThemeVector, base int) int {
	if base <= 0 {
		return 0
	}
	// Direction weight from tone: benevolent/order/nature positive, malevolent/void/chaos negative.
	var dir int
	switch v.Tone {
	case ToneBenevolent, ToneOrder, ToneNature:
		dir = 1
	case ToneMalevolent, ToneVoid:
		dir = -1
	default:
		dir = 0
	}
	if dir == 0 {
		return 0
	}

	// Magnitude from holdings + social standing (read live, not from cached vector, for accuracy).
	hold := l.CalculateTotalPortfolioValue(v.Wallet)
	holdScore := int(holdingsToScore(hold)) // 0..100

	l.mutex.RLock()
	stats, ok := l.leaderboard[v.Wallet]
	l.mutex.RUnlock()
	var social, anti int
	if ok {
		social = int(clampInt64(int64(stats.Reputation), 0, 100)) +
			int(clampInt64(int64(stats.Mojo), 0, 100))
		anti = int(clampInt64(int64(stats.WantedLevel), 0, 100)) +
			int(clampInt64(int64(stats.Cunning), 0, 100))
	}

	// Positive drivers vs negative drivers (keep within 0..100 each).
	pos := clampInt(holdScore+social, 0, 200)
	neg := clampInt(anti, 0, 200)

	// Bias fraction: |dir| * (pos - neg) scaled into [0, BIAS_MAX] basis points.
	deltaFrac := (pos - neg) * BIAS_MAX / 200 // signed, range ~[-15, +15]
	if deltaFrac > BIAS_MAX {
		deltaFrac = BIAS_MAX
	}
	if deltaFrac < -BIAS_MAX {
		deltaFrac = -BIAS_MAX
	}

	// Apply entropy: entropy widens variance — scale the cap up to the full BIAS_MAX
	// only when entropy is high; otherwise tighten. (Deterministic, integer.)
	entropyScale := 50 + v.Entropy/2 // 50..100
	deltaFrac = deltaFrac * entropyScale / 100
	if deltaFrac > BIAS_MAX {
		deltaFrac = BIAS_MAX
	}
	if deltaFrac < -BIAS_MAX {
		deltaFrac = -BIAS_MAX
	}

	// base is in micro-units; result is an integer delta in the same unit.
	delta := base * deltaFrac / 100
	return delta
}

// ComputeWorldDynamicsSignature builds the §27.4 ten-signal weighted region vitality
// projection from REAL sources only. Signals without a real data source yet score 0
// (FaithCoherence, DomesticCoherence, EntityLegitimacy) and are NOT fabricated.
func (l *Lobby) ComputeWorldDynamicsSignature(region string) WorldDynamicsSignature {
	sig := WorldDynamicsSignature{Region: region}

	// 1. MarketVitality — Σ reserve balances across entity market nodes (tokenSinkRouter.MarketNodes).
	if l.tokenSinkRouter != nil {
		// SNAPSHOT UNDER THE MAP'S OWNER LOCK, RELEASE, THEN TAKE EACH NODE'S LOCK. This HTTP door holds
		// NO lobby lock, and the lobby lock would not cover the map anyway: `Lobby.marketNodes` and
		// `TokenSinkRouter.MarketNodes` are THE SAME map, whose ONE owner mutex is the router's. The
		// range here used to run with NO lock at all, against every insert on the trade path — a
		// `fatal error: concurrent map iteration and map write`, which Go does not let anything recover
		// from. Taking the node locks while still holding `tsr.Mu` would create `tsr.Mu -> node.Mu`, the
		// REVERSE of the order the investment doors use when they call RouteCriminalTax holding a node,
		// so the two are taken SEQUENTIALLY by construction (`marketNodeRefs` releases before it returns).
		for _, ref := range l.tokenSinkRouter.marketNodeRefs() {
			node := ref.Node
			node.Mu.RLock()
			// Dividend pool + reserve both signal market vitality (§27.2.1).
			sig.MarketVitality += int(node.ReserveBalance / 1_000_000) // $VBV units
			sig.MarketVitality += int(node.DividendPoolMicro / 1_000_000)
			node.Mu.RUnlock()
		}
		sig.MarketVitality = int(clampInt64(int64(sig.MarketVitality), 0, 1_000_000))
	}

	// 2. CitizenGravity — §27.3: Σ over AICitizen in region of Reputation×(AttachmentTier+1).
	if l.aiEngine != nil {
		for _, c := range l.aiEngine.GetAllCitizens() {
			if c.Region != region {
				continue
			}
			sig.CitizenGravity += c.Reputation * (c.AttachmentTier + 1)
		}
		sig.CitizenGravity = int(clampInt64(int64(sig.CitizenGravity), 0, 1_000_000))
	}

	// 3. ThemeCoherence — §27.1: spread of per-player theme vectors in the region.
	// Computed from leaderboard players whose employer club is in this region (best available
	// real proxy for "players resident to a region"). Coherence = inverse of tone variance.
	sig.ThemeCoherence = l.computeRegionThemeCoherence(region)

	// 4. EventDynamics — §26: active caches + active user events in the region.
	if l.seasonEngine != nil {
		l.seasonEngine.Mu.RLock()
		for _, c := range l.seasonEngine.Caches {
			if c.Region == region && !c.Claimed {
				sig.EventDynamics += 10
			}
		}
		for _, e := range l.seasonEngine.UserEvents {
			if e.Region == region && e.Status != "RESOLUTION" {
				sig.EventDynamics += 15
			}
		}
		l.seasonEngine.Mu.RUnlock()
		sig.EventDynamics = int(clampInt64(int64(sig.EventDynamics), 0, 100_000))
	}

	// 5. ProfileImpact — §27.3: aggregate player Reputation + Wins + Achievements in region.
	sig.ProfileImpact = l.computeRegionProfileImpact(region)

	// 6. FaithCoherence — §27.7 / §32: now COMPUTABLE. Each region's AI citizens + clubs carry
	// a religion/dogma; rituals raise coherence. Envelops world activity → dictates the region's
	// faith. Integer math only (no float on the ledger-facing signal).
	sig.FaithCoherence = l.computeRegionFaithCoherence(region)

	// 7. DomesticCoherence — §27.7.1: NOW WIRED (structs present at ai_citizen_engine.go:49-63).
	// Σ_citizens(hasSpouse×8 + hasLover×4 + children×6 + aipets×3) × weight, clamped to 0..1_000_000.
	// (childrenEventsHeld not yet tracked on BondGraph → omitted; added when event matrix logs it.)
	sig.DomesticCoherence = l.computeRegionDomesticCoherence(region)

	// 8. RumorCoherence — §27.7.2: net social rumour sentiment per region.
	// Σ (positive − slander) × Strength for rumors targeting wallets in this region.
	sig.RumorCoherence = l.computeRegionRumorCoherence(region)

	// 9. EconomicPerk — §27.7.2: tax-haven activity for clubs in this region.
	sig.EconomicPerk = l.computeRegionEconomicPerk(region)

	// 10. EntityLegitimacy — §27.7.3: NOW WIRED (structs present at ai_citizen_engine.go:50-52).
	// Σ_citizens(Certified×10 − BlackMarketAdopted×12), clamped to 0..1_000_000.
	sig.EntityLegitimacy = l.computeRegionEntityLegitimacy(region)

	// Weighted total (PILLAR 2 integer math).
	var total int64
	total += int64(sig.MarketVitality) * int64(W_MARKET_VITALITY)
	total += int64(sig.CitizenGravity) * int64(W_CITIZEN_GRAVITY)
	total += int64(sig.ThemeCoherence) * int64(W_THEME_COHERENCE)
	total += int64(sig.EventDynamics) * int64(W_EVENT_DYNAMICS)
	total += int64(sig.ProfileImpact) * int64(W_PROFILE_IMPACT)
	total += int64(sig.FaithCoherence) * int64(W_FAITH_COHERENCE)
	total += int64(sig.DomesticCoherence) * int64(W_DOMESTIC_COHERENCE)
	total += int64(sig.RumorCoherence) * int64(W_RUMOR_COHERENCE)
	total += int64(sig.EconomicPerk) * int64(W_ECON_PERK)
	total += int64(sig.EntityLegitimacy) * int64(W_ENTITY_LEGITIMACY)
	sig.Score = total
	return sig
}

// computeRegionDomesticCoherence — §27.7.1 (NOW WIRED).
// Σ_citizens(hasSpouse×8 + hasLover×4 + children×6 + aipets×3), clamped 0..1_000_000.
// Reads BondGraph (ai_citizen_engine.go:58) — present on AICitizen. Integer math only.
func (l *Lobby) computeRegionDomesticCoherence(region string) int {
	if l.aiEngine == nil {
		return 0
	}
	var sum int64
	for _, c := range l.aiEngine.GetAllCitizens() {
		if c.Region != region {
			continue
		}
		var contrib int64
		if c.BondGraph != nil {
			if c.BondGraph.WifeWallet != "" {
				contrib += 8
			}
			if c.BondGraph.LoverWallet != "" {
				contrib += 4
			}
			contrib += int64(len(c.BondGraph.BotChildIDs)) * 6
			contrib += int64(len(c.BondGraph.PetIDs)) * 3
		}
		sum += contrib
	}
	return int(clampInt64(sum, 0, 1_000_000))
}

// computeRegionEntityLegitimacy — §27.7.3 (NOW WIRED).
// Σ_citizens(Certified×10 − BlackMarketAdopted×12), clamped 0..1_000_000.
// Reads Certified/BirthCertID/BlackMarketAdopted (ai_citizen_engine.go:50-52). Integer math only.
func (l *Lobby) computeRegionEntityLegitimacy(region string) int {
	if l.aiEngine == nil {
		return 0
	}
	var sum int64
	for _, c := range l.aiEngine.GetAllCitizens() {
		if c.Region != region {
			continue
		}
		if c.Certified {
			sum += 10
		}
		if c.BlackMarketAdopted {
			sum -= 12
		}
	}
	return int(clampInt64(sum, 0, 1_000_000))
}

// computeRegionFaithCoherence — §27.7 / §32 (NOW WIRED; resolves G1 FaithCoherence=0 gap).
// Faith acts like a club that envelops world activity. Each region's AI citizens + faith-clubs
// contribute: Σ_citizens(rituals_done×10 + (faithful? +5 : 0)) + Σ_faithclubs(rituals×8).
// Clamped 0..1_000_000. Integer math only (no float on the ledger-facing signal).
func (l *Lobby) ComputeRegionFaithCoherence(region string) int {
	return l.computeRegionFaithCoherence(region)
}

func (l *Lobby) computeRegionFaithCoherence(region string) int {
	var sum int64
	if l.aiEngine != nil {
		for _, c := range l.aiEngine.GetAllCitizens() {
			if c.Region != region {
				continue
			}
			if c.DogmaTag != "" {
				sum += int64(c.RitualsDone) * 10
				sum += 5 // faithful presence
			}
		}
	}
	l.mutex.RLock()
	for _, club := range l.clubs {
		if club.RegionName != region {
			continue
		}
		if club.Type == "Faith" && club.DogmaTag != "" {
			// faith clubs carry a RitualsDone ledger (reuse MutationSuccesses as ritual counter
			// proxy until a dedicated field is added; integer, no float).
			sum += int64(club.RitualsDone) * 8
		}
	}
	l.mutex.RUnlock()
	return int(clampInt64(sum, 0, 1_000_000))
}

// computeRegionThemeCoherence returns inverse-variance coherence of player theme tones
// for players resident to the region (employer club in region). 0..100.
func (l *Lobby) computeRegionThemeCoherence(region string) int {
	regionClubs := l.clubsInRegion(region)
	if len(regionClubs) == 0 {
		return 0
	}
	// SNAPSHOT the resident wallets under the read lock, then derive each vector OUTSIDE it.
	// ComputeThemeVector takes the lobby lock ITSELF, so calling it here was a RECURSIVE READ LOCK:
	// a sync.RWMutex blocks a second RLock the moment a writer queues, and this goroutine never
	// releases the first one — the process freezes (measured by selflock_gate_test.go, which reports
	// this shape as a violation). The snapshot also keeps the locked window to a map walk.
	var residents []string
	l.mutex.RLock()
	for _, st := range l.leaderboard {
		if st.EmployerClubID == "" {
			continue
		}
		if _, inRegion := regionClubs[st.EmployerClubID]; !inRegion {
			continue
		}
		residents = append(residents, st.Wallet)
	}
	l.mutex.RUnlock()

	var tones []int
	for _, w := range residents {
		v := l.ComputeThemeVector(w)
		tones = append(tones, int(v.Tone))
	}
	if len(tones) == 0 {
		return 0
	}
	// Variance of tone values; coherence = 100 - normalized spread.
	mean := 0
	for _, t := range tones {
		mean += t
	}
	mean /= len(tones)
	var varSum int
	for _, t := range tones {
		d := t - mean
		varSum += d * d
	}
	variance := varSum / len(tones) // 0..~9 for tone range 0..6
	coh := 100 - variance*10
	if coh < 0 {
		coh = 0
	}
	if coh > 100 {
		coh = 100
	}
	return coh
}

// computeRegionProfileImpact aggregates Reputation + Wins/10 + Achievements for players
// resident to the region (employer club in region). Bounded 0..100_000.
func (l *Lobby) computeRegionProfileImpact(region string) int {
	regionClubs := l.clubsInRegion(region)
	if len(regionClubs) == 0 {
		return 0
	}
	var sum int
	l.mutex.RLock()
	for _, st := range l.leaderboard {
		if st.EmployerClubID == "" {
			continue
		}
		if _, inRegion := regionClubs[st.EmployerClubID]; !inRegion {
			continue
		}
		sum += int(clampInt64(int64(st.Reputation), 0, 1000))
		sum += int(clampInt64(int64(st.Wins/10), 0, 500))
		sum += len(st.Achievements) * 5
	}
	l.mutex.RUnlock()
	return int(clampInt64(int64(sum), 0, 100_000))
}

// computeRegionRumorCoherence computes net rumour sentiment per region. Rumors target a
// wallet; a wallet is "in region" if its employer club is in the region. Positive rumors
// add Strength×10, negative subtract. Bounded [-100_000, 100_000].
func (l *Lobby) computeRegionRumorCoherence(region string) int {
	regionClubs := l.clubsInRegion(region)
	if len(regionClubs) == 0 {
		return 0
	}
	inRegion := make(map[string]bool)
	l.mutex.RLock()
	for wk, st := range l.leaderboard {
		if st.EmployerClubID != "" {
			if _, ok := regionClubs[st.EmployerClubID]; ok {
				inRegion[wk] = true
			}
		}
	}
	var net int
	for _, rumor := range l.rumors {
		if !inRegion[rumor.TargetWallet] {
			continue
		}
		strength := int(clampInt64(int64(rumor.Strength*10), 0, 100))
		if rumor.Type == "positive" {
			net += strength
		} else if rumor.Type == "negative" {
			net -= strength
		}
	}
	l.mutex.RUnlock()
	return int(clampInt64(int64(net), -100_000, 100_000))
}

// computeRegionEconomicPerk sums tax-haven activity for clubs in the region
// (TokenSinkRouter.IsTaxHavenActive + Club.TaxHavenExpiresAt). One point per active haven.
// Bounded 0..10_000.
func (l *Lobby) computeRegionEconomicPerk(region string) int {
	regionClubs := l.clubsInRegion(region)
	if len(regionClubs) == 0 {
		return 0
	}
	var perks int
	l.mutex.RLock()
	for id := range regionClubs {
		club, ok := l.clubs[id]
		if !ok {
			continue
		}
		if l.tokenSinkRouter != nil && l.tokenSinkRouter.IsTaxHavenActive(club) {
			perks += 1
		}
	}
	l.mutex.RUnlock()
	return int(clampInt64(int64(perks), 0, 10_000))
}

// clubsInRegion returns the set of club IDs whose RegionName matches the given region.
func (l *Lobby) clubsInRegion(region string) map[string]bool {
	out := make(map[string]bool)
	if region == "" {
		return out
	}
	l.mutex.RLock()
	for id, club := range l.clubs {
		if club.RegionName == region {
			out[id] = true
		}
	}
	l.mutex.RUnlock()
	return out
}

// MarketWeather derives a deterministic §27.2.1 climate + disaster tier from the region's
// entity-market state. Uses the WorldDynamicsSignature.MarketVitality + CitizenGravity as
// the real inputs (no RNG).
func (l *Lobby) MarketWeather(region string) WeatherState {
	sig := l.ComputeWorldDynamicsSignature(region)
	ws := WeatherState{Region: region}

	// Climate from market vitality (proxy for economic weather).
	switch {
	case sig.MarketVitality >= 5000:
		ws.Climate = "CALM"
	case sig.MarketVitality >= 1000:
		ws.Climate = "BREEZY"
	case sig.MarketVitality >= 100:
		ws.Climate = "STORMY"
	default:
		ws.Climate = "BLIGHTED"
	}

	// Disaster tier from citizen gravity deficit vs market vitality.
	// High vitality + high gravity => stable (tier 0). Low vitality => higher tier.
	if sig.MarketVitality >= 5000 && sig.CitizenGravity >= 5000 {
		ws.DisasterTier = 0
	} else if sig.MarketVitality >= 1000 {
		ws.DisasterTier = 1
	} else if sig.MarketVitality >= 100 {
		ws.DisasterTier = 2
	} else {
		ws.DisasterTier = 3
	}
	return ws
}

// ── small deterministic helpers (integer, PILLAR 2) ─────────────────────────────

// normalizeWallet lower-cases a wallet for stable map keys (mirrors rivalry_engine usage).
func normalizeWallet(w string) string {
	out := []rune(w)
	for i, c := range out {
		if c >= 'A' && c <= 'Z' {
			out[i] = c + ('a' - 'A')
		}
	}
	return string(out)
}

// holdingsToScore maps micro-VBV holdings to a 0..100 score on a log-ish scale.
func holdingsToScore(holdings uint64) int {
	// Thresholds (micro-VBV): 0, 10M(10$VBV), 50M, 250M, 1B, 5B.
	switch {
	case holdings >= 5_000_000_000:
		return 100
	case holdings >= 1_000_000_000:
		return 80
	case holdings >= 250_000_000:
		return 60
	case holdings >= 50_000_000:
		return 40
	case holdings >= 10_000_000:
		return 20
	case holdings > 0:
		return 5
	default:
		return 0
	}
}

// socialRankWeight maps SocialRank string to a 0..30 weight.
func socialRankWeight(rank string) int {
	switch rank {
	case "Icon":
		return 30
	case "Regular":
		return 15
	case "Nobody":
		return 0
	default:
		return 0
	}
}

// sumRelationships sums relationship scores (0..100), capped at 100.
func sumRelationships(rels map[string]int) int {
	var s int
	for _, v := range rels {
		s += v
	}
	if s > 100 {
		s = 100
	}
	if s < 0 {
		s = 0
	}
	return s
}

// clampInt64 clamps an int64 to [lo, hi].
func clampInt64(v, lo, hi int64) int64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// absInt returns the absolute value of an int.
func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// getWalletFromRequest extracts the wallet address from request headers or query params.
// Mirrors RateLimiterService.getWalletFromRequest semantics (X-Wallet-Address header,
// fallback to ?wallet=). Defined on Lobby so handler code does not depend on the
// (currently missing) package-level extractWalletFromRequest helper.
func (l *Lobby) getWalletFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	if w := r.Header.Get("X-Wallet-Address"); w != "" {
		return strings.ToLower(strings.TrimSpace(w))
	}
	if w := r.URL.Query().Get("wallet"); w != "" {
		return strings.ToLower(strings.TrimSpace(w))
	}
	return ""
}

// ── HTTP handlers (mirror the rate-limited JSON pattern in server.go) ───────────

// handleThemeVector returns the computed ThemeVector for a wallet.
func (l *Lobby) handleThemeVector(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	v := l.ComputeThemeVector(wallet)
	writeJSON(w, map[string]interface{}{
		"success":      true,
		"wallet":       v.Wallet,
		"tone":         int(v.Tone),
		"element":      int(v.Element),
		"intensity":    v.Intensity,
		"entropy":      v.Entropy,
		"mood_tag":     v.MoodTag,
	})
}

// handleMarketWeather returns the §27.2.1 weather projection for a region.
func (l *Lobby) handleMarketWeather(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	if region == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "region required"})
		return
	}
	ws := l.MarketWeather(region)
	writeJSON(w, map[string]interface{}{
		"success":        true,
		"region":         ws.Region,
		"climate":        ws.Climate,
		"disaster_tier":  ws.DisasterTier,
	})
}

// handleWorldDynamics returns the §27.4 ten-signal signature for a region.
func (l *Lobby) handleWorldDynamics(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	if region == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "region required"})
		return
	}
	sig := l.ComputeWorldDynamicsSignature(region)
	writeJSON(w, map[string]interface{}{
		"success":             true,
		"region":              sig.Region,
		"market_vitality":     sig.MarketVitality,
		"citizen_gravity":     sig.CitizenGravity,
		"theme_coherence":     sig.ThemeCoherence,
		"event_dynamics":      sig.EventDynamics,
		"profile_impact":      sig.ProfileImpact,
		"faith_coherence":     sig.FaithCoherence,
		"domestic_coherence":  sig.DomesticCoherence,
		"rumor_coherence":     sig.RumorCoherence,
		"economic_perk":       sig.EconomicPerk,
		"entity_legitimacy":   sig.EntityLegitimacy,
		"score":               sig.Score,
	})
}

// handlePlayerProgression returns the full unlock/progression snapshot for a wallet.
// Aggregates ThemeVector, WorldDynamics, club/territory ownership, pets, vehicles,
// church ownership, and creator registration into a single constellation-state response.
func (l *Lobby) handlePlayerProgression(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}

	// ── Read lobby state under RLock ──
	l.mutex.RLock()
	stats, hasStats := l.leaderboard[wallet]
	vbvBalance := l.playerBalances[wallet]

	// Find player's club + territory count
	var playerClub *Club
	ownedTerritories := 0
	for _, club := range l.clubs {
		if strings.EqualFold(club.OwnerWallet, wallet) {
			playerClub = club
			ownedTerritories = len(club.Territories)
			break
		}
	}

	// Governor status (IsClubRegionalLocked reads l.clubs — lock must be held)
	isGovernor := false
	if playerClub != nil && l.clubService != nil {
		isGovernor = l.clubService.IsClubRegionalLocked(l, playerClub)
	}

	// Career tier
	careerTier := 0
	if hasStats && stats.CareerXP != nil && stats.JobRole != "" {
		careerTier = stats.CareerXP.GetCareerTier(stats.JobRole)
	}

	// Region from club or default
	regionName := "Governor"
	if playerClub != nil && playerClub.RegionName != "" {
		regionName = playerClub.RegionName
	}
	l.mutex.RUnlock()

	// ── Theme vector (stateless computation) ──
	tv := l.ComputeThemeVector(wallet)

	// ── Region weather ──
	ws := l.MarketWeather(regionName)

	// ── Pets + vehicles (each acquires RLock internally) ──
	pets := l.GetPets(wallet)
	vehicles := l.GetVehicles(wallet)
	hasPetsOrVehicles := len(pets) > 0 || len(vehicles) > 0

	// ── Church ownership (faithChurchEngine has its own mutex) ──
	churches := faithChurchEngine.GetChurchesByOwner(wallet)
	hasChurch := len(churches) > 0

	// ── Creator registration (creatorStore has its own mutex) ──
	isCreator := false
	if l.creatorStore != nil {
		_, err := l.creatorStore.GetCreatorProfile(wallet)
		isCreator = err == nil
	}

	// ── Compute feature states ──
	features := map[string]interface{}{}

	// Shops: always unlocked
	features["shops"] = map[string]interface{}{
		"state":     "alive",
		"progress":  1.0,
		"gate_text": "Unlocked",
	}

	// Career: always unlocked (XP-gated tiers)
	features["career"] = map[string]interface{}{
		"state":     "alive",
		"progress":  1.0,
		"gate_text": fmt.Sprintf("Tier %d — Level up through VBV", careerTier),
	}

	// Clubs: 5000 VBV to found
	if playerClub != nil {
		features["clubs"] = map[string]interface{}{
			"state":     "alive",
			"progress":  1.0,
			"gate_text": fmt.Sprintf("Founded: %s", playerClub.Name),
		}
	} else {
		progress := float64(vbvBalance) / float64(5000*1000000)
		if progress > 1.0 {
			progress = 1.0
		}
		state := "dormant"
		if progress >= 0.5 {
			state = "dawning"
		}
		features["clubs"] = map[string]interface{}{
			"state":     state,
			"progress":  progress,
			"gate_text": "5000 VBV to found",
		}
	}

	// Territory: need club first, then 2500 VBV per territory
	if playerClub != nil {
		territoryProgress := float64(ownedTerritories) / 5.0
		if territoryProgress > 1.0 {
			territoryProgress = 1.0
		}
		territoryState := "dormant"
		if ownedTerritories > 0 {
			territoryState = "alive"
		} else if vbvBalance >= 2500*1000000 {
			territoryState = "dawning"
		}
		features["territory"] = map[string]interface{}{
			"state":     territoryState,
			"progress":  territoryProgress,
			"gate_text": fmt.Sprintf("%d territories owned", ownedTerritories),
		}
	} else {
		features["territory"] = map[string]interface{}{
			"state":     "dormant",
			"progress":  0.0,
			"gate_text": "Found a club first",
		}
	}

	// Governor: 2+ territories
	if isGovernor {
		features["governor"] = map[string]interface{}{
			"state":     "alive",
			"progress":  1.0,
			"gate_text": "Governor status active",
		}
	} else {
		govProgress := float64(ownedTerritories) / 2.0
		if govProgress > 1.0 {
			govProgress = 1.0
		}
		govState := "dormant"
		if govProgress >= 0.5 {
			govState = "dawning"
		}
		features["governor"] = map[string]interface{}{
			"state":     govState,
			"progress":  govProgress,
			"gate_text": "Own 2 territories",
		}
	}

	// Regional Manager: same gate as governor
	if isGovernor {
		features["regional"] = map[string]interface{}{
			"state":     "alive",
			"progress":  1.0,
			"gate_text": "Regional Manager unlocked",
		}
	} else {
		regProgress := float64(ownedTerritories) / 2.0
		if regProgress > 1.0 {
			regProgress = 1.0
		}
		regState := "dormant"
		if regProgress >= 0.5 {
			regState = "dawning"
		}
		features["regional"] = map[string]interface{}{
			"state":     regState,
			"progress":  regProgress,
			"gate_text": "Become Governor first",
		}
	}

	// Rivalry: own a pet or vehicle
	if hasPetsOrVehicles {
		features["rivalry"] = map[string]interface{}{
			"state":     "alive",
			"progress":  1.0,
			"gate_text": fmt.Sprintf("%d pets, %d vehicles", len(pets), len(vehicles)),
		}
	} else {
		features["rivalry"] = map[string]interface{}{
			"state":     "dormant",
			"progress":  0.0,
			"gate_text": "Own a pet or vehicle",
		}
	}

	// Faith: own a church
	if hasChurch {
		features["faith"] = map[string]interface{}{
			"state":     "alive",
			"progress":  1.0,
			"gate_text": fmt.Sprintf("%d churches", len(churches)),
		}
	} else {
		features["faith"] = map[string]interface{}{
			"state":     "dormant",
			"progress":  0.0,
			"gate_text": "Open a church",
		}
	}

	// Stats overlay: power system intercept (career-tier driven)
	statsProgress := float64(careerTier) / 10.0
	if statsProgress > 1.0 {
		statsProgress = 1.0
	}
	statsState := "dormant"
	if hasStats && stats.CareerXP != nil {
		statsState = "alive"
		statsProgress = 1.0
	} else if statsProgress >= 0.5 {
		statsState = "dawning"
	}
	features["stats"] = map[string]interface{}{
		"state":     statsState,
		"progress":  statsProgress,
		"gate_text": "Power system intercept",
	}

	// Tournament: own a deck + entry fee (gate on VBV balance for entry)
	tournamentProgress := float64(vbvBalance) / float64(1000*1000000)
	if tournamentProgress > 1.0 {
		tournamentProgress = 1.0
	}
	tournamentState := "dormant"
	if tournamentProgress >= 0.5 {
		tournamentState = "dawning"
	}
	features["tournament"] = map[string]interface{}{
		"state":     tournamentState,
		"progress":  tournamentProgress,
		"gate_text": "Own a deck + entry fee",
	}

	// Entity Market: own a listable entity
	if hasPetsOrVehicles {
		features["entity"] = map[string]interface{}{
			"state":     "alive",
			"progress":  1.0,
			"gate_text": fmt.Sprintf("%d entities", len(pets)+len(vehicles)),
		}
	} else {
		features["entity"] = map[string]interface{}{
			"state":     "dormant",
			"progress":  0.0,
			"gate_text": "Own a listable entity",
		}
	}

	// Children Bots: born from entities
	if hasPetsOrVehicles {
		features["children"] = map[string]interface{}{
			"state":     "alive",
			"progress":  1.0,
			"gate_text": "Children bots available",
		}
	} else {
		features["children"] = map[string]interface{}{
			"state":     "dormant",
			"progress":  0.0,
			"gate_text": "Own a pet or vehicle",
		}
	}

	// Creator Store: free registration
	if isCreator {
		features["creator"] = map[string]interface{}{
			"state":     "alive",
			"progress":  1.0,
			"gate_text": "Creator registered",
		}
	} else {
		features["creator"] = map[string]interface{}{
			"state":     "dawning",
			"progress":  0.5,
			"gate_text": "Register as creator (free)",
		}
	}

	// Governance: governor-only
	if isGovernor {
		features["governance"] = map[string]interface{}{
			"state":     "alive",
			"progress":  1.0,
			"gate_text": "Governance unlocked",
		}
	} else {
		govProgress := float64(ownedTerritories) / 2.0
		if govProgress > 1.0 {
			govProgress = 1.0
		}
		govState := "dormant"
		if govProgress >= 0.5 {
			govState = "dawning"
		}
		features["governance"] = map[string]interface{}{
			"state":     govState,
			"progress":  govProgress,
			"gate_text": "Governor status required",
		}
	}

	// Infrastructure Leasing: territory ownership
	if ownedTerritories > 0 {
		features["leasing"] = map[string]interface{}{
			"state":     "alive",
			"progress":  1.0,
			"gate_text": fmt.Sprintf("Lease in %d territories", ownedTerritories),
		}
	} else {
		features["leasing"] = map[string]interface{}{
			"state":     "dormant",
			"progress":  0.0,
			"gate_text": "Own a territory",
		}
	}

	// Identity: always unlocked
	features["identity"] = map[string]interface{}{
		"state":     "alive",
		"progress":  1.0,
		"gate_text": "Always available",
	}

	// ── Constellation paths ──
	constellationPaths := [][]string{
		{"clubs", "territory", "governor", "regional"},
		{"clubs", "territory", "governor", "governance"},
		{"territory", "leasing"},
		{"rivalry", "career"},
		{"entity", "children"},
		{"faith", "stats"},
		{"shops", "creator"},
	}

	writeJSON(w, map[string]interface{}{
		"success": true,
		"wallet":  wallet,
		"theme": map[string]interface{}{
			"tone":      int(tv.Tone),
			"element":   int(tv.Element),
			"intensity": tv.Intensity,
			"entropy":   tv.Entropy,
			"mood_tag":  tv.MoodTag,
		},
		"region": map[string]interface{}{
			"name":          ws.Region,
			"climate":       ws.Climate,
			"disaster_tier": ws.DisasterTier,
		},
		"features":            features,
		"constellation_paths": constellationPaths,
	})
}

// ensure time import is used (RegionView/weather rely on determinism; keep explicit).
var _ = time.Now
