//go:build !js && !wasm

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// InitRivalryEngine initializes the region/territory rivalry engine for a lobby.
func (l *Lobby) InitRivalryEngine() {
	l.rivalryEngine = &RivalryEngine{
		Signatures: make(map[string]*AssetSignature),
		Rivalries:  make(map[string]*RegionRivalry),
	}
}

// ComputeAssetSignature (§25.10.1) — deterministic integer-weighted signature for a region.
// Reads deployed assets + citizen-value tiers. Scope: deployed-in-capital/territory locations (§25.8).
func (l *Lobby) ComputeAssetSignature(region, territory string, isCapital bool) *AssetSignature {
	sig := &AssetSignature{Region: region, Territory: territory, IsCapital: isCapital}

	// User workers (career.go): JobRole != "" && EmployerClubID == territory && Salary > 0.
	// Weighted by SalaryTier × JobRoleTier via W_USER_WORKER base.
	if territory != "" {
		l.mutex.RLock()
		for w, st := range l.leaderboard {
			if st.JobRole != "" && st.EmployerClubID == territory && st.Salary > 0 {
				_ = w
				jobTier := 1 // career tier proxy; workers already weighted highest (W_USER_WORKER)
				salaryTier := 1 + int(st.Salary/(100*1000000))
				sig.UserWorkers += (RIVAL_W_WORKER_BASE * jobTier * salaryTier) / 1000
			}
		}
		l.mutex.RUnlock()
	}

	// AI-citizen pride (§3): Σ Reputation × (AttachmentTier+1), scoped to region.
	if l.aiEngine != nil {
		for _, c := range l.aiEngine.GetAllCitizens() {
			if c.Region != region {
				continue
			}
			sig.AICitizenPride += c.Reputation * (c.AttachmentTier + 1)
			// Model citizens: leveled avatars (use Tier as proxy for Level L, §24.5).
			modelL := clampInt(c.Tier, 0, 75)
			sig.ModelCitizens += (W_MODEL_CITIZEN * modelL) / 75
			// Vehicles / pets / world content deployed by this citizen's owner are captured
			// at the territory level via club inventory below; here we count AI-citizen presence.
		}
	}

	// Deployed world content / items / events scoped to territory/region (§25.7 / §14 / §26).
	// These are sourced from the seasonal engine caches/events (§26) + club treasury holdings.
	if l.seasonEngine != nil {
		for _, c := range l.seasonEngine.Caches {
			if c.Region == region && !c.Claimed {
				sig.WorldContent += W_WORLD_CONTENT
			}
		}
		for _, e := range l.seasonEngine.UserEvents {
			if e.Region == region && e.Status != "RESOLUTION" {
				sig.Events += W_EVENT_TYPE
			}
		}
	}
	if club, ok := l.clubs[territory]; ok {
		// Club treasury scale as a proxy for deployed item/vehicle/pet stock (§14/§25.6/§26.4).
		tier := clampInt(int(club.TreasuryMicro/(1000*1000000)), 0, 100)
		sig.Items += (W_ITEM_ARCHETYPE * tier) / 100
		sig.Vehicles += (W_VEHICLE * tier) / 100
		sig.Pets += (W_PET_BLOODLINE * tier) / 100
	}

	// Final weighted score (PILLAR 2 integer math).
	var total uint64
	total += uint64(sig.UserWorkers)              // already weighted
	total += uint64(sig.AICitizenPride) * uint64(W_AI_PRIDE) / 1000
	total += uint64(sig.ModelCitizens)
	total += uint64(sig.Vehicles)
	total += uint64(sig.Pets)
	total += uint64(sig.WorldContent)
	total += uint64(sig.Items)
	total += uint64(sig.Events)
	sig.Score = total
	return sig
}

// RecomputeAllSignatures refreshes every known region/territory signature.
func (l *Lobby) RecomputeAllSignatures() {
	l.rivalryEngine.Mu.Lock()
	defer l.rivalryEngine.Mu.Unlock()

	regions := map[string]bool{}
	if l.aiEngine != nil {
		for _, c := range l.aiEngine.GetAllCitizens() {
			regions[c.Region] = true
		}
	}
	for r := range regions {
		key := r + "|"
		l.rivalryEngine.Signatures[key] = l.ComputeAssetSignature(r, "", false)
	}
	// Territory-level (capital) signatures from clubs. SNAPSHOT the club ids under the read lock and
	// derive each signature OUTSIDE it: ComputeAssetSignature takes the lobby read lock ITSELF, so
	// calling it here was a RECURSIVE READ LOCK (a queued writer blocks the second RLock for ever,
	// and this goroutine never releases the first one — the whole process freezes). The signatures
	// map is KEYED, so the write order of a map range cannot change the result.
	type clubRef struct{ id, region string }
	var clubRefs []clubRef
	l.mutex.RLock()
	for id, club := range l.clubs {
		clubRefs = append(clubRefs, clubRef{id: id, region: club.RegionName})
	}
	l.mutex.RUnlock()

	for _, c := range clubRefs {
		key := c.region + "|" + c.id
		l.rivalryEngine.Signatures[key] = l.ComputeAssetSignature(c.region, c.id, true)
	}
}

// DetectRivalries (§25.10) — auto-form rivalries where two sides share a dominant asset class
// (signature overlap ≥ RIVAL_COLLISION_THRESHOLD). Cross-region = Capital-vs-Capital.
func (l *Lobby) DetectRivalries() []*RegionRivalry {
	l.rivalryEngine.Mu.Lock()
	defer l.rivalryEngine.Mu.Unlock()

	var created []*RegionRivalry
	sigs := make([]*AssetSignature, 0, len(l.rivalryEngine.Signatures))
	for _, s := range l.rivalryEngine.Signatures {
		sigs = append(sigs, s)
	}
	for i := 0; i < len(sigs); i++ {
		for j := i + 1; j < len(sigs); j++ {
			a, b := sigs[i], sigs[j]
			// Only compare same level (region-vs-region or territory-vs-territory).
			if (a.Territory == "") != (b.Territory == "") {
				continue
			}
			if a.Score == 0 || b.Score == 0 {
				continue
			}
			// Collision: dominant class overlap ≥ threshold (simplified: min/max ratio).
			ratio := uint64(0)
			if a.Score > b.Score {
				ratio = (b.Score * 100000) / a.Score
			} else {
				ratio = (a.Score * 100000) / b.Score
			}
			if ratio < uint64(RIVAL_COLLISION_THRESHOLD) {
				continue // too disparate in strength → not a contested rivalry
			}
			// Determine contested asset class (highest non-zero component).
			assetClass := dominantClass(a)
			level := "REGION"
			sideA, sideB := a.Region, b.Region
			if a.Territory != "" {
				level = "TERRITORY"
				sideA, sideB = a.Territory, b.Territory
			}
			rv := &RegionRivalry{
				RivalryID:  uuid.New().String(),
				Level:      level,
				SideA:      sideA,
				SideB:      sideB,
				AssetClass: assetClass,
				ScoreA:     a.Score,
				ScoreB:     b.Score,
				CreatedAt:  time.Now(),
			}
			l.rivalryEngine.Rivalries[rv.RivalryID] = rv
			created = append(created, rv)
		}
	}
	return created
}

// ResolveRivalry (§25.10.1) — settle: higher score wins; prize = 4-part (showcase buff, resident cache, gov rep, worker bonus).
func (l *Lobby) ResolveRivalry(rivalryID string) (*RegionRivalry, error) {
	l.rivalryEngine.Mu.Lock()
	rv, ok := l.rivalryEngine.Rivalries[rivalryID]
	if !ok || rv.Winner != "" {
		l.rivalryEngine.Mu.Unlock()
		return nil, fmt.Errorf("rivalry unavailable")
	}
	winner := rv.SideA
	winnerScore := rv.ScoreA
	if rv.ScoreB > rv.ScoreA {
		winner = rv.SideB
		winnerScore = rv.ScoreB
	}
	rv.Winner = winner
	rv.ResolvedAt = time.Now()

	// §27.5 — ThemeEngine OutcomeBias hook (Phase A). The winning side's region world-dynamics
	// signature is precomputed here (l.mutex is free; rivalryEngine.Mu is held, which is
	// independent). Non-structural: resolution uses l.OutcomeBias on demand.
	winnerRegion := ""
	if l.aiEngine != nil {
		for _, c := range l.aiEngine.GetAllCitizens() {
			if c.Wallet == strings.ToLower(winner) {
				winnerRegion = c.Region
				break
			}
		}
	}
	if winnerRegion != "" {
		_ = l.ComputeWorldDynamicsSignature(winnerRegion)
	}

	l.rivalryEngine.Mu.Unlock()

	// 4-part prize (PILLAR 2 — RouteCriminalTax for the resident cache).
	cacheMicro := uint64(RIVAL_CACHE_BASE_MICRO) * (1 + winnerScore/1000000)
	if l.tokenSinkRouter != nil {
		_ = l.tokenSinkRouter.RouteCriminalTax("RIVALRY_CACHE", cacheMicro, RevenueSplitMatrix{FaucetShare: 1.0, ClubShare: 0.0, GovernanceShare: 0.0}, 0, "")
	}
	l.mutex.Lock()
	// Governor rep (§25.9).
	rep := l.leaderboard[strings.ToLower(winner)]
	rep.Reputation += RIVAL_GOV_REP
	l.leaderboard[strings.ToLower(winner)] = rep
	l.mutex.Unlock()
	l.logAdminAuditLocked("RIVALRY_RESOLVED", winner, fmt.Sprintf("Level: %s, Class: %s, Cache: %d", rv.Level, rv.AssetClass, cacheMicro))
	return rv, nil
}

// dominantClass returns the highest-weighted component label of a signature.
func dominantClass(s *AssetSignature) string {
	best, name := uint64(s.UserWorkers), "USER_WORKER"
	if v := uint64(s.AICitizenPride) * uint64(W_AI_PRIDE) / 1000; v > best {
		best, name = v, "AI_PRIDE"
	}
	if v := uint64(s.ModelCitizens); v > best {
		best, name = v, "MODEL_CITIZEN"
	}
	if v := uint64(s.Vehicles); v > best {
		best, name = v, "VEHICLE"
	}
	if v := uint64(s.Pets); v > best {
		best, name = v, "PET_BLOODLINE"
	}
	if v := uint64(s.WorldContent); v > best {
		best, name = v, "WORLD_CONTENT"
	}
	if v := uint64(s.Items); v > best {
		best, name = v, "ITEM_ARCHETYPE"
	}
	if v := uint64(s.Events); v > best {
		best, name = v, "EVENT_TYPE"
	}
	return name
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
