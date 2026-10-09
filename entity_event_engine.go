//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"
)

// §30 Pet World — Entity Event & Stat Engine.
//
// Design (locked 2026-08-30):
//   - Pets + bot-children (derived AICitizen) carry an EntityStats vector (integer 1..100, no float).
//   - Events TRAIN the entity (stat XP), REWARD for winning/trying, PUNISH for quitting.
//   - Bonuses/Degradations come from: rivalries (rival_career_engine), regional dynamics
//     (ComputeWorldDynamicsSignature), and owner relationship status (OwnerOpinion + OwnerCrossOpinion).
//   - The stats ARE the 3D-world POWER OVERLAY: ComputeEffectivePowerLevel caps the entity's
//     effective level to its stat ceiling — users are capped to the new stats in the 3D world.
//   - Tournaments (bigger rewards) for MATURE entities only; immature train but earn nothing.
//   - Hosting is gated to the host's PROFILE dynamics (tier/rep/region); a user may ENTER a bot
//     into another owner's dynamic but may NOT create events outside their own profile tier.
//   - ALL ledger/reward math is uint64 micro. No floats anywhere.

const (
	StatMax          uint64 = 100
	EventStatGain    uint64 = 2 // per win/try (uint64)
	QuitStatPenalty  uint64 = 4 // per quit (uint64 debuff)
	PowerOverlayScale uint64 = 50 // effective level = baseLevel + floor(statSum/PowerOverlayScale), capped
)

// EntityPowerOverlay is the per-region summary the 3D world renders (region capitals tint/scale
// by entity power). It is the visible form of the stat overlay across a region.
type EntityPowerOverlay struct {
	Region             string `json:"region"`
	AvgEntityPower     uint64 `json:"avg_entity_power"`     // integer avg of effective power levels
	TotalMature        int    `json:"total_mature"`         // mature entities eligible for tournaments
	TotalImmature      int    `json:"total_immature"`        // training-only
	TopDominantStat    string `json:"top_dominant_stat"`    // most common dominant stat in region
}

// EntityEvent is one pet/bot event instance.
type EntityEvent struct {
	EventID     string    `json:"event_id"`
	HostWallet  string    `json:"host_wallet"`  // gated by profile dynamics
	Region      string    `json:"region"`
	EventTier   int       `json:"event_tier"`   // must be <= host profile tier
	Kind        string    `json:"kind"`         // "TRAIN" | "TOURNAMENT" | "FUN"
	CostMicro   uint64    `json:"cost_micro"`   // host sunk cost (uint64)
	Participants []string `json:"participants"` // pet_ids + bot_wallets
	CreatedAt   time.Time `json:"created_at"`
}

// EntityEventResult records one entity's outcome in an event.
type EntityEventResult struct {
	EntityID   string `json:"entity_id"`
	IsBot      bool   `json:"is_bot"`     // true=AICitizen, false=PetNFT
	Outcome    string `json:"outcome"`    // "WIN" | "TRY" | "QUIT"
	StatDelta  int    `json:"stat_delta"` // signed integer (uint64 applied as +, - as debuff)
	RewardMicro uint64 `json:"reward_micro"` // uint64; 0 if immature/quit
}

// OwnerRelationGraph tracks owner→entity opinion + owner→owner asset-bot standing.
// Keyed by ownerWallet; CrossOpinion[otherOwner] = standing (1..100).
type OwnerRelationGraph struct {
	mu           sync.RWMutex
	Opinion      map[string]uint64            // owner -> direct opinion of their own assets (avg)
	CrossOpinion map[string]map[string]uint64 // owner -> otherOwner -> standing
}

// NewOwnerRelationGraph returns an empty graph.
func NewOwnerRelationGraph() *OwnerRelationGraph {
	return &OwnerRelationGraph{
		Opinion:      make(map[string]uint64),
		CrossOpinion: make(map[string]map[string]uint64),
	}
}

// SetOpinion records a direct owner->entity opinion (1..100, clamped).
func (g *OwnerRelationGraph) SetOpinion(owner string, v uint64) {
	if v > StatMax {
		v = StatMax
	}
	g.mu.Lock()
	g.Opinion[owner] = v
	g.mu.Unlock()
}

// SetCrossOpinion records owner A's standing toward owner B's asset-bots (1..100, clamped).
func (g *OwnerRelationGraph) SetCrossOpinion(a, b string, v uint64) {
	if v > StatMax {
		v = StatMax
	}
	g.mu.Lock()
	if g.CrossOpinion[a] == nil {
		g.CrossOpinion[a] = make(map[string]uint64)
	}
	g.CrossOpinion[a][b] = v
	g.mu.Unlock()
}

// CrossStanding returns owner A's standing toward owner B (0 if unset).
func (g *OwnerRelationGraph) CrossStanding(a, b string) uint64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if m, ok := g.CrossOpinion[a]; ok {
		return m[b]
	}
	return 0
}

// EntityEventEngine owns events + the owner relation graph for a lobby.
type EntityEventEngine struct {
	mu      sync.RWMutex
	events  map[string]*EntityEvent
	graph   *OwnerRelationGraph
}

// NewEntityEventEngine returns an empty engine.
func NewEntityEventEngine() *EntityEventEngine {
	return &EntityEventEngine{
		events: make(map[string]*EntityEvent),
		graph: NewOwnerRelationGraph(),
	}
}

// HostEvent creates an event. hostTier is the host's profile tier (reputation/region/career).
// Creation is REJECTED if eventTier > hostTier (users may only host relative to their own dynamics).
// Entering a bot into ANOTHER owner's dynamic is allowed (participants list), but creating is gated.
func (e *EntityEventEngine) HostEvent(hostWallet, region string, eventTier int, kind string, costMicro uint64, hostTier int, participants []string) (*EntityEvent, error) {
	if eventTier > hostTier {
		return nil, fmt.Errorf("host profile tier %d cannot create event tier %d (gated to own dynamics)", hostTier, eventTier)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	ev := &EntityEvent{
		EventID:     fmt.Sprintf("EVT-%d", time.Now().UnixNano()),
		HostWallet:  hostWallet,
		Region:      region,
		EventTier:   eventTier,
		Kind:        kind,
		CostMicro:   costMicro,
		Participants: participants,
		CreatedAt:   time.Now(),
	}
	e.events[ev.EventID] = ev
	return ev, nil
}

// ComputeEffectivePowerLevel returns the 3D-world power-overlay level for an entity.
// baseLevel = pet_level / bot_level (or career tier for users). The overlay CAPS effective
// level to (baseLevel + floor(statSum/PowerOverlayScale)), clamped to StatMax*6 hard ceiling.
// This is pure integer math — no floats.
func ComputeEffectivePowerLevel(baseLevel uint64, stats EntityStats) uint64 {
	sum := stats.StatSum()
	bonus := sum / PowerOverlayScale // integer division
	eff := baseLevel + bonus
	ceil := StatMax * 6 // 600 hard ceiling
	if eff > ceil {
		eff = ceil
	}
	return eff
}

// ApplyEventResult trains/rewards/punishes one entity's stats + level. Returns the recorded result.
// mature=false → training only, no reward (immature pets / pre-maturity bots).
// blackMarket=true → ineligible for reward (constitutional: legitimacy = right to think).
func (e *EntityEventEngine) ApplyEventResult(entityID string, isBot bool, outcome string, stats *EntityStats, level *uint64, mature bool, blackMarket bool) EntityEventResult {
	res := EntityEventResult{EntityID: entityID, IsBot: isBot, Outcome: outcome}
	switch outcome {
	case "WIN":
		res.StatDelta = +int(EventStatGain)
		if !blackMarket && mature {
			res.RewardMicro = EventStatGain * 1000 // uint64 micro reward (scaled)
		}
	case "TRY":
		res.StatDelta = +int(EventStatGain / 2) // trying still trains (integer)
		if !blackMarket && mature {
			res.RewardMicro = (EventStatGain / 2) * 1000
		}
	case "QUIT":
		res.StatDelta = -int(QuitStatPenalty) // punish quitting
		res.RewardMicro = 0
	default:
		res.StatDelta = 0
	}
	// Apply signed delta to stats (clamped 1..StatMax) — boost/degradation overlay.
	if stats != nil {
		applyStatDelta(stats, res.StatDelta)
		// Function-based level: dominant stat raises level (integer).
		if stats.DominantStat() != "" && *level < StatMax {
			*level++ // one level per event participation (capped)
		}
	}
	return res
}

// applyStatDelta nudges every stat by delta, clamped to [1, StatMax]. Pure integer.
func applyStatDelta(s *EntityStats, delta int) {
	clampStat := func(v uint64) uint64 {
		n := int(v) + delta
		if n < 1 {
			return 1
		}
		if n > int(StatMax) {
			return StatMax
		}
		return uint64(n)
	}
	s.Speed = clampStat(s.Speed)
	s.Intelligence = clampStat(s.Intelligence)
	s.Willpower = clampStat(s.Willpower)
	s.Strength = clampStat(s.Strength)
	s.Charisma = clampStat(s.Charisma)
	s.Agility = clampStat(s.Agility)
}

// BaseUserEntityStats derives a user's base EntityStats from their PlayerStats profile.
// INTEGER ONLY — no float. This is the floor the 3D-world power overlay is applied ON TOP OF,
// so users are ALSO capped to the new stats (per locked design). Reputation/Mojo/Cunning/Nurturing/
// WantedLevel/career tier all map to the stat vector; WantedLevel degrades lawful stats.
func BaseUserEntityStats(p *PlayerStats) EntityStats {
	if p == nil {
		return EntityStats{Speed: 1, Intelligence: 1, Willpower: 1, Strength: 1, Charisma: 1, Agility: 1}
	}
	s := EntityStats{
		// Reputation (int) → Charisma (civic standing); clamp 1..StatMax
		Charisma: uint64(clampInt(p.Reputation/5+20, 1, int(StatMax))),
		// Mojo (social) → Strength of presence
		Strength: uint64(clampInt(p.Mojo/5+20, 1, int(StatMax))),
		// Cunning → Intelligence + Willpower
		Intelligence: uint64(clampInt(p.Cunning*5+20, 1, int(StatMax))),
		Willpower:    uint64(clampInt(p.Cunning*3+25, 1, int(StatMax))),
		// Nurturing → Willpower + Charisma
		Agility: uint64(clampInt(p.Nurturing*4+20, 1, int(StatMax))),
		// base speed from activity (wins proxy)
		Speed: uint64(clampInt(p.Wins/10+20, 1, int(StatMax))),
	}
	// WantedLevel degrades lawful stats (Intelligence/Charisma/Willpower) — integer subtract.
	if p.WantedLevel > 0 {
		deg := uint64(p.WantedLevel * 3)
		if s.Intelligence > deg {
			s.Intelligence -= deg
		} else {
			s.Intelligence = 1
		}
		if s.Charisma > deg {
			s.Charisma -= deg
		} else {
			s.Charisma = 1
		}
	}
	return s
}

// CombineStats adds base (user profile) + overlay (event-trained) stats, clamped to [1, StatMax].
// This is what the 3D world renders as the user's effective stat vector.
func CombineStats(base, overlay EntityStats) EntityStats {
	add := func(a, b uint64) uint64 {
		n := a + b
		if n > StatMax {
			return StatMax
		}
		return n
	}
	return EntityStats{
		Speed:       add(base.Speed, overlay.Speed),
		Intelligence: add(base.Intelligence, overlay.Intelligence),
		Willpower:   add(base.Willpower, overlay.Willpower),
		Strength:    add(base.Strength, overlay.Strength),
		Charisma:    add(base.Charisma, overlay.Charisma),
		Agility:     add(base.Agility, overlay.Agility),
	}
}
// pets + bots + vehicles enumerated; mature vs immature counted; top dominant stat computed.
// §25.6.1: deployed vehicles participate exactly as pets do — the vehicle upgrade ladder
// only means something if it shows up in the world.
func (e *EntityEventEngine) BuildRegionPowerOverlay(region string, pets []*PetNFT, bots []*AICitizen, vehicles []*VehicleNFT) *EntityPowerOverlay {
	ov := &EntityPowerOverlay{Region: region}
	var powers []uint64
	domCount := make(map[string]int)
	for _, p := range pets {
		base := p.PetLevel
		if p.Mature {
			ov.TotalMature++
		} else {
			ov.TotalImmature++
		}
		pw := ComputeEffectivePowerLevel(base, p.Stats)
		powers = append(powers, pw)
		domCount[p.Stats.DominantStat()]++
	}
	for _, v := range vehicles {
		base := v.VehicleLevel
		if base == 0 {
			base = 1
		}
		// Break-in is the vehicle's maturity analogue: broken-in → race-ready.
		if vehicleBreakInCompleteLocked(v) {
			ov.TotalMature++
		} else {
			ov.TotalImmature++
		}
		pw := ComputeEffectivePowerLevel(base, v.Stats)
		powers = append(powers, pw)
		domCount[v.Stats.DominantStat()]++
	}
	for _, b := range bots {
		base := b.BotLevel
		// bot maturity inferred: Tier>=1 && Certified → mature (bot-children mature at ornith Level>=25)
		if b.Certified && b.BotLevel > 0 {
			ov.TotalMature++
		} else {
			ov.TotalImmature++
		}
		pw := ComputeEffectivePowerLevel(base, b.Stats)
		powers = append(powers, pw)
		domCount[b.Stats.DominantStat()]++
	}
	if len(powers) > 0 {
		var total uint64
		for _, pw := range powers {
			total += pw
		}
		ov.AvgEntityPower = total / uint64(len(powers))
	}
	// top dominant stat
	best, name := 0, "None"
	for k, v := range domCount {
		if v > best {
			best, name = v, k
		}
	}
	ov.TopDominantStat = name
	return ov
}

// handleEntityRegions returns region views enriched with the §30 Pet World power overlay.
// Pure read; the 3D world (world3d.js) consumes entity_power_overlay to render entity power.
func (l *Lobby) handleEntityRegions(w http.ResponseWriter, r *http.Request) {
	if l.entityEvents == nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "entity event engine not initialized"})
		return
	}
	views := l.GetRegionViews()
	writeJSON(w, map[string]interface{}{"success": true, "regions": views})
}
// ── §31 SYNCOVERY / ORPHAN / COMBINED-EVENTS (DESIGN LOCKED 2026-08-30) ──────────
// Integer-only. All ledger math uint64 micro. No float.

const (
	// OrphanGraceDays — an owner silent > this long → entity becomes Orphaned.
	OrphanGraceDays = 90
	// AlimonyCommissionBps — adoptive parent pays this fraction (basis points) of a
	// micro-grant to the ORIGINAL owner when adopting an orphaned entity.
	AlimonyCommissionBps = 500 // 5%
	// SynergyWeightFloor — opinion below this contributes nothing to combined stats.
	SynergyWeightFloor uint64 = 10
	// InverseGapBonus — complementary (non-dominant) stats get a small synergy bonus.
	InverseGapBonus uint64 = 3
)

// CombineOwnerStats merges a user's profile-derived base stats with the pathway-weighted,
// opinion-weighted contribution of their pets + bot-children. Synergy dynamic maps to PATHWAY:
// same-pathway entities amplify; complementary dominant stats grant an inverse-gap bonus.
// Returns the household's effective EntityStats (clamped 1..StatMax). Integer math only.
func CombineOwnerStats(owner *PlayerStats, pets []*PetNFT, bots []*AICitizen) EntityStats {
	base := BaseUserEntityStats(owner)
	combine := func(acc, s EntityStats, opinion uint64) EntityStats {
		// SynergyWeight = opinion (1..100). Below floor → no contribution.
		weight := opinion
		if weight < SynergyWeightFloor {
			weight = 0
		}
		add := func(a, b uint64) uint64 {
			n := a + (b*weight)/100 // opinion-weighted contribution
			if n > StatMax {
				return StatMax
			}
			return n
		}
		return EntityStats{
			Speed:       add(acc.Speed, s.Speed),
			Intelligence: add(acc.Intelligence, s.Intelligence),
			Willpower:   add(acc.Willpower, s.Willpower),
			Strength:    add(acc.Strength, s.Strength),
			Charisma:    add(acc.Charisma, s.Charisma),
			Agility:     add(acc.Agility, s.Agility),
		}
	}
	acc := base
	for _, p := range pets {
		// complementary dominant stat → inverse-gap bonus
		if p.Stats.DominantStat() != base.DominantStat() {
			acc = acc.plus(InverseGapBonus)
		}
		acc = combine(acc, p.Stats, p.OwnerOpinion)
	}
	for _, b := range bots {
		if b.Stats.DominantStat() != base.DominantStat() {
			acc = acc.plus(InverseGapBonus)
		}
		acc = combine(acc, b.Stats, b.OwnerOpinion)
	}
	return acc
}

// plus returns the stats with each component raised by delta (clamped to StatMax).
func (s EntityStats) plus(delta uint64) EntityStats {
	add := func(v uint64) uint64 {
		n := v + delta
		if n > StatMax {
			return StatMax
		}
		return n
	}
	return EntityStats{
		Speed:       add(s.Speed),
		Intelligence: add(s.Intelligence),
		Willpower:   add(s.Willpower),
		Strength:    add(s.Strength),
		Charisma:    add(s.Charisma),
		Agility:     add(s.Agility),
	}
}

// IsOrphaned reports whether an owner has been inactive beyond OrphanGraceDays.
// lastActive = owner's last confirmed action; now = current time. Pure logic (no I/O).
func IsOrphaned(lastActive time.Time, now time.Time) bool {
	return now.Sub(lastActive) > (time.Duration(OrphanGraceDays) * 24 * time.Hour)
}

// AdoptEntityResult records an adoption of an orphaned entity (pet or bot).
type AdoptEntityResult struct {
	EntityID         string `json:"entity_id"`
	OriginalOwner    string `json:"original_owner"`
	AdoptiveParent   string `json:"adoptive_parent"`
	AlimonyMicro     uint64 `json:"alimony_micro"`     // paid to original owner
	GrantMicro       uint64 `json:"grant_micro"`       // micro-grant to adoptive parent (from faucet)
	AdoptiveTreasury uint64 `json:"adoptive_treasury"` // remaining after alimony (if paid from treasury)
}

// AdoptEntity implements the orphan re-imbursement flow (§31, integer-only):
//   - the adoptive parent pays AlimonyCommission (uint64 micro) to the INACTIVE original owner
//     (drawn from the adoptive parent's treasury if available, else a faucet micro-grant).
//   - the adoptive parent receives a small faucet micro-grant to bootstrap the new bond.
// The original owner may later ReclaimEntity (§31) and be reimbursed by the adoptive parent.
// treasuryPtr: if non-nil, alimony is drawn from *treasuryPtr (and remaining written back).
// faucetPtr:   if treasury insufficient, the shortfall is covered by *faucetPtr (system reservoir).
func AdoptEntity(entityID, originalOwner, adoptiveParent string, treasuryPtr, faucetPtr *uint64) AdoptEntityResult {
	res := AdoptEntityResult{
		EntityID:       entityID,
		OriginalOwner:  originalOwner,
		AdoptiveParent: adoptiveParent,
	}
	// Base adoption micro-grant from faucet.
	const adoptionGrant uint64 = 1000 // micro-VBV bootstrap
	if faucetPtr != nil && *faucetPtr >= adoptionGrant {
		*faucetPtr -= adoptionGrant
		res.GrantMicro = adoptionGrant
	}
	// Alimony commission to the original (inactive) owner: drawn from adoptive treasury if present.
	alimony := adoptionGrant * AlimonyCommissionBps / 10000
	if treasuryPtr != nil && *treasuryPtr >= alimony {
		*treasuryPtr -= alimony
		res.AdoptiveTreasury = *treasuryPtr
		res.AlimonyMicro = alimony
	} else if faucetPtr != nil && *faucetPtr >= alimony {
		// treasury empty → faucet covers alimony (still paid to original owner)
		*faucetPtr -= alimony
		res.AlimonyMicro = alimony
	}
	// NOTE: actual wallet transfer to original owner is performed by the caller (lobby/economy
	// layer) using res.AlimonyMicro; this engine computes the constitutional integer amounts.
	return res
}

// ReclaimEntityResult records an inactive owner reclaiming their orphaned entity.
type ReclaimEntityResult struct {
	EntityID         string `json:"entity_id"`
	OriginalOwner    string `json:"original_owner"`
	AdoptiveParent   string `json:"adoptive_parent"`
	ReimburseMicro   uint64 `json:"reimburse_micro"` // adoptive parent is reimbursed
	OwnerOpinion     uint64 `json:"owner_opinion"`   // restored to full (100)
}

// ReclaimEntity reimburses the adoptive parent and restores ownership + OwnerOpinion.
// reimbursePtr (adoptive parent treasury) is credited with ReimburseMicro by the caller; this
// engine computes the integer amount (mirror of the alimony paid at adoption).
func ReclaimEntity(entityID, originalOwner, adoptiveParent string, alimonyPaid uint64, reimbursePtr *uint64) ReclaimEntityResult {
	res := ReclaimEntityResult{
		EntityID:       entityID,
		OriginalOwner:  originalOwner,
		AdoptiveParent: adoptiveParent,
		ReimburseMicro: alimonyPaid, // full reimbursement of the alimony commission
		OwnerOpinion:   100,
	}
	if reimbursePtr != nil {
		*reimbursePtr += res.ReimburseMicro
	}
	return res
}

// CombinedEvent is a training/battle unit of [owner + pets + bots] acting as ONE entity.
// Each member contributes a UNIQUE dominant-stat specialty; the combined result feeds the
// §30 power overlay. Outcomes train/reward/punish every member (per ApplyEventResult).
type CombinedEvent struct {
	EventID     string    `json:"event_id"`
	HostWallet  string    `json:"host_wallet"`
	Region      string    `json:"region"`
	OwnerWallet string    `json:"owner_wallet"`
	PetIDs      []string  `json:"pet_ids"`
	BotWallets  []string  `json:"bot_wallets"`
	CreatedAt   time.Time `json:"created_at"`
}

// CombineOwnerStatsForEvent returns the household stats used as the combined unit's vector.
func (l *Lobby) BuildCombinedHousehold(owner *PlayerStats, pets []*PetNFT, bots []*AICitizen) EntityStats {
	return CombineOwnerStats(owner, pets, bots)
}

// ── §30 Pet-World PAYOUT LOOP (DESIGN LOCKED 2026-08-30) ───────────────────────
// The §30 ApplyEventResult computes EntityEventResult.RewardMicro (uint64) but never credits a
// wallet. This loop closes that gap: it resolves an event's outcomes into owner-wallet micro-VBV
// credits (deterministic, integer-only — constitutional no-float rule). Immature + black-market
// entities earn 0 (enforced inside ApplyEventResult).

// EntityOutcome is one entity's submitted outcome for an event (from the host/client).
type EntityOutcome struct {
	EntityID string `json:"entity_id"`
	IsBot    bool   `json:"is_bot"`
	Outcome  string `json:"outcome"` // WIN | TRY | QUIT
}

// ownerWalletOf resolves the human owner wallet for a pet or bot entity (uint64 credit target).
func (e *EntityEventEngine) ownerWalletOf(l *Lobby, id string, isBot bool) string {
	if l == nil {
		return ""
	}
	if isBot {
		if c, ok := l.aiEngine.GetCitizen(id); ok {
			if c.OwnerWallet != "" {
				return c.OwnerWallet
			}
			return c.OriginWallet
		}
		return ""
	}
	if p, ok := l.pets[id]; ok {
		return p.Owner
	}
	return ""
}

// ProcessEntityEvent applies each outcome (train/reward/punish) to the entity's stats/level and
// returns the recorded results. It mutates the live pet/bot stat vectors (under lobby mutex).
func (e *EntityEventEngine) ProcessEntityEvent(l *Lobby, eventID string, outcomes []EntityOutcome) []EntityEventResult {
	results := make([]EntityEventResult, 0, len(outcomes))
	for _, o := range outcomes {
		if c, ok := l.aiEngine.GetCitizen(o.EntityID); ok {
			stats := c.Stats
			level := c.BotLevel
			res := e.ApplyEventResult(o.EntityID, true, o.Outcome, &stats, &level, c.Certified && c.BotLevel > 0, c.BlackMarketAdopted)
			c.Stats = stats
			c.BotLevel = level
			results = append(results, res)
			continue
		}
		if pet, ok := l.pets[o.EntityID]; ok {
			stats := pet.Stats
			level := pet.PetLevel
			res := e.ApplyEventResult(o.EntityID, false, o.Outcome, &stats, &level, pet.Mature, pet.BlackMarketAdopted)
			pet.Stats = stats
			pet.PetLevel = level
			results = append(results, res)
			continue
		}
	}
	return results
}

// ResolveEntityEventPayout credits each outcome's RewardMicro to the owning wallet's in-game
// balance (l.playerBalances, uint64 micro). Returns the per-wallet credit map. Integer-only.
func (e *EntityEventEngine) ResolveEntityEventPayout(l *Lobby, results []EntityEventResult) map[string]uint64 {
	credits := make(map[string]uint64)
	for _, res := range results {
		if res.RewardMicro == 0 {
			continue
		}
		owner := e.ownerWalletOf(l, res.EntityID, res.IsBot)
		if owner == "" {
			continue
		}
		l.playerBalances[owner] += res.RewardMicro
		credits[owner] += res.RewardMicro
	}
	return credits
}

func (l *Lobby) handleEntityEventHost(w http.ResponseWriter, r *http.Request) {
	if l.entityEvents == nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "entity event engine not initialized"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	var req struct {
		Name       string `json:"name"`
		EventTier  int    `json:"event_tier"`
		Region     string `json:"region"`
		CostMicro  uint64 `json:"cost_micro"`
		Kind       string `json:"kind"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	if req.Region == "" { req.Region = "Base" }
	if req.Kind == "" { req.Kind = "COMBINED" }

	l.mutex.Lock()
	hostTier := 0
	if st, ok := l.leaderboard[wallet]; ok {
		hostTier = st.CareerTier
	}
	ev, err := l.entityEvents.HostEvent(wallet, req.Region, req.EventTier, req.Kind, req.CostMicro, hostTier, nil)
	l.mutex.Unlock()

	if err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true, "event": ev})
}

// handleEntityEventResolve applies submitted entity outcomes (train/reward/punish) and runs the
// §30 payout loop, crediting owner wallets (uint64 micro). Host gated to profile tier by HostEvent.
func (l *Lobby) handleEntityEventResolve(w http.ResponseWriter, r *http.Request) {
	if l.entityEvents == nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "entity event engine not initialized"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	var req struct {
		EventID  string          `json:"event_id"`
		Outcomes []EntityOutcome `json:"outcomes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}
	l.mutex.Lock()
	results := l.entityEvents.ProcessEntityEvent(l, req.EventID, req.Outcomes)
	credits := l.entityEvents.ResolveEntityEventPayout(l, results)
	l.mutex.Unlock()
	writeJSON(w, map[string]interface{}{
		"success": true,
		"results": results,
		"credits": credits,
	})
}

// ── §31 HTTP handlers: synergy / orphan / combined-events ──────────────────────

// handleCombineOwnerStats returns the household's combined EntityStats for the requesting owner
// (pathway-weighted + opinion-weighted; §31). Consumed by the 3D world + combined-event UI.
func (l *Lobby) handleCombineOwnerStats(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	owner := l.leaderboard[wallet]
	pets := l.GetPets(wallet)
	var bots []*AICitizen
	if l.aiEngine != nil {
		for _, c := range l.aiEngine.GetAllCitizens() {
			if c.OwnerWallet == wallet || c.OriginWallet == wallet {
				bots = append(bots, c)
			}
		}
	}
	combined := CombineOwnerStats(&owner, pets, bots)
	writeJSON(w, map[string]interface{}{
		"success":         true,
		"combined_stats":  combined,
		"effective_power": ComputeEffectivePowerLevel(0, combined),
		// §26.4.3/§25.6.2: the real percentage this household's bonded assets (certified,
		// non-black-market pets + vehicles) lend to the owner's deck cards in a match.
		"bonded_deck_boost_pct": l.BondedDeckBoostPct(wallet),
	})
}

// handleAdoptEntity records an adoption of an orphaned entity, computing the constitutional
// integer alimony (§31). The actual wallet transfer is performed by the economy layer using
// the returned AlimonyMicro. Treasury/faucet pointers are best-effort (nil-safe).
func (l *Lobby) handleAdoptEntity(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EntityID      string `json:"entity_id"`
		OriginalOwner string `json:"original_owner"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}
	adopter := l.getWalletFromRequest(r)
	if adopter == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	var treasuryPtr, faucetPtr *uint64
	if _, ok := l.leaderboard[adopter]; ok {
		tb := l.playerBalances[adopter]
		treasuryPtr = &tb
	}
	fb := l.faucetBalanceMicro
	faucetPtr = &fb
	res := AdoptEntity(req.EntityID, req.OriginalOwner, adopter, treasuryPtr, faucetPtr)
	if faucetPtr != nil {
		l.faucetBalanceMicro = *faucetPtr
	}
	if treasuryPtr != nil {
		l.playerBalances[adopter] = *treasuryPtr
	}
	writeJSON(w, map[string]interface{}{"success": true, "adoption": res})
}

// handleReclaimEntity reimburses the adoptive parent and restores ownership + OwnerOpinion (§31).
func (l *Lobby) handleReclaimEntity(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EntityID       string `json:"entity_id"`
		OriginalOwner  string `json:"original_owner"`
		AdoptiveParent string `json:"adoptive_parent"`
		AlimonyPaid    uint64 `json:"alimony_paid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}
	var reimbursePtr *uint64
	if req.AdoptiveParent != "" {
		rb := l.playerBalances[req.AdoptiveParent]
		reimbursePtr = &rb
	}
	res := ReclaimEntity(req.EntityID, req.OriginalOwner, req.AdoptiveParent, req.AlimonyPaid, reimbursePtr)
	if reimbursePtr != nil {
		l.playerBalances[req.AdoptiveParent] = *reimbursePtr
	}
	writeJSON(w, map[string]interface{}{"success": true, "reclaim": res})
}

// handleOrphanStatus reports whether the requesting owner is currently orphaned (§31: 90d grace).
func (l *Lobby) handleOrphanStatus(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	last, ok := l.lastActive[wallet] // last confirmed activity (set on paired match)
	orphaned := false
	if ok {
		orphaned = IsOrphaned(last, time.Now())
	}
	writeJSON(w, map[string]interface{}{
		"success":    true,
		"orphaned":   orphaned,
		"grace_days": OrphanGraceDays,
	})
}

// ── §32 HTTP handlers: faith + religious card-battle gambit ─────────────────────

// handleFaithCoherence returns the region's live FaithCoherence (§27.7/§32, resolves G1).
func (l *Lobby) handleFaithCoherence(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	if region == "" {
		region = "Base"
	}
	writeJSON(w, map[string]interface{}{
		"success":            true,
		"region":             region,
		"faith_coherence":    l.ComputeRegionFaithCoherence(region),
		"domestic_coherence": l.computeRegionDomesticCoherence(region),
		"entity_legitimacy":  l.computeRegionEntityLegitimacy(region),
		"score":              l.ComputeWorldDynamicsSignature(region).Score,
	})
}

// handleFaithWarGambit is the religious card-battle entry: the requester stakes their FAVOURITE
// CARD (PlayerStats.FavoriteCardID) into the faith-war POT. On loss the card is jailed to the
// winning faith's kitty; on win it returns + winnings. Hosted by an entity (user/bot). A large
// tournament aggregates these into a dynamic religious buff.
func (l *Lobby) handleFaithWarGambit(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	st, ok := l.leaderboard[wallet]
	if !ok || st.FavoriteCardID == 0 {
		writeJSON(w, map[string]interface{}{"success": false, "error": "no favourite card designated"})
		return
	}
	fav := st.FavoriteCardID
	card, ok := l.persistentCardCache[fav]
	if !ok {
		writeJSON(w, map[string]interface{}{"success": false, "error": "favourite card not found"})
		return
	}
	// Stake: jail the favourite card to the faith-war pot (Pillar 3 capture semantics).
	st.JailedCards[fav] = "FAITH_WAR_POT"
	writeJSON(w, map[string]interface{}{
		"success":         true,
		"staked_card_id":  fav,
		"card_religion":   card.Religion,
		"note":            "favourite card staked to faith-war pot; loss -> jailed to winning faith kitty, win -> returned + winnings",
	})
}

// tournament seeding and 3D world layering.
func SortEntitiesByPower(pets []*PetNFT, bots []*AICitizen) []string {
	type pe struct {
		id    string
		power uint64
	}
	var list []pe
	for _, p := range pets {
		list = append(list, pe{p.PetID, ComputeEffectivePowerLevel(p.PetLevel, p.Stats)})
	}
	for _, b := range bots {
		list = append(list, pe{b.Wallet, ComputeEffectivePowerLevel(b.BotLevel, b.Stats)})
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].power > list[j].power })
	out := make([]string, len(list))
	for i, x := range list {
		out[i] = x.id
	}
	return out
}
