//go:build !js && !wasm

package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"time"
)

// §30 Pet-World AUTONOMOUS TOURNAMENT SCHEDULER (closes G4 — deferred design item).
//
// The payout loop (ProcessEntityEvent / ResolveEntityEventPayout) is client-triggered via
// /api/entity-event/resolve. This scheduler makes it AUTONOMOUS: on a fixed interval it
// gathers the lobby's MATURE entities (pets + bot-children), hosts a deterministic tournament
// bracket, resolves outcomes through the existing payout loop, and credits owner wallets
// (uint64 micro-VBV). No external trigger required.
//
// Determinism (PILLAR 2): outcomes are derived from a SHA-256 hash of (eventID + entityID + index),
// NEVER math/rand. The sim is bit-reproducible for the same entity set + event ID.
// ALL ledger math is uint64 micro. No floats anywhere (constitutional no-float rule).

const (
	// AutonomousTournamentInterval — how often the scheduler runs a bracket.
	AutonomousTournamentInterval = 15 * time.Minute
	// AutonomousTournamentMinBracket — need at least this many mature entities to run.
	AutonomousTournamentMinBracket = 2
)

// RunAutonomousTournamentScheduler is the long-lived goroutine. Launched once from Lobby.run()
// (mirrors oracleService.RefreshGlobalLeaderboard). It self-schedules on a ticker and is alive
// for the server's lifetime.
func (e *EntityEventEngine) RunAutonomousTournamentScheduler(l *Lobby) {
	ticker := time.NewTicker(AutonomousTournamentInterval)
	defer ticker.Stop()
	log.Printf("[PET-WORLD] Autonomous tournament scheduler active (interval %s)", AutonomousTournamentInterval)
	for range ticker.C {
		e.runAutonomousTournament(l)
	}
}

// runAutonomousTournament executes ONE autonomous bracket.
func (e *EntityEventEngine) runAutonomousTournament(l *Lobby) {
	if e == nil || l == nil {
		return
	}
	// Snapshot mature entities under read lock.
	l.mutex.RLock()
	type ent struct {
		id    string
		isBot bool
		owner string
	}
	var bracket []ent
	for _, p := range l.pets {
		if p.Mature && !p.BlackMarketAdopted {
			bracket = append(bracket, ent{id: p.PetID, isBot: false, owner: p.Owner})
		}
	}
	l.mutex.RUnlock()

	// The AI-engine read happens OUTSIDE the lobby lock: `GetAllCitizens` takes `ace.mu`, and the
	// behavioural tick holds `ace.mu` while it needs the lobby lock, so nesting the two here closes
	// an ABBA cycle (measured in lock_order_gate_test.go).
	if l.aiEngine != nil {
		for _, c := range l.aiEngine.GetAllCitizens() {
			if c.Certified && c.BotLevel > 0 && !c.BlackMarketAdopted {
				owner := c.OwnerWallet
				if owner == "" {
					owner = c.OriginWallet
				}
				bracket = append(bracket, ent{id: c.Wallet, isBot: true, owner: owner})
			}
		}
	}

	if len(bracket) < AutonomousTournamentMinBracket {
		// Not enough mature entities yet — skip this cycle (no error, just no bracket).
		return
	}

	// Host = owner of the highest-power mature entity. Host tier gates event creation.
	hostWallet := bracket[0].owner
	hostTier := 0
	l.mutex.RLock()
	if st, ok := l.leaderboard[hostWallet]; ok {
		hostTier = st.CareerTier
	}
	l.mutex.RUnlock()
	if hostWallet == "" {
		return
	}

	// Create the tournament (gated to host's own tier — always passes since eventTier==hostTier).
	ev, err := e.HostEvent(hostWallet, "Base", hostTier, "TOURNAMENT", 0, hostTier, nil)
	if err != nil {
		log.Printf("[PET-WORLD] autonomous HostEvent rejected: %v", err)
		return
	}

	// Build deterministic outcomes for the bracket. Index-based seeding keeps it reproducible.
	outcomes := make([]EntityOutcome, 0, len(bracket))
	for i, b := range bracket {
		outcomes = append(outcomes, EntityOutcome{
			EntityID: b.id,
			IsBot:    b.isBot,
			Outcome:  determineOutcome(ev.EventID, b.id, i),
		})
	}

	// Resolve through the existing payout loop (mutates stats/levels + credits wallets).
	l.mutex.Lock()
	results := e.ProcessEntityEvent(l, ev.EventID, outcomes)
	credits := e.ResolveEntityEventPayout(l, results)
	l.mutex.Unlock()

	log.Printf("[PET-WORLD] autonomous tournament %s: %d entities, %d wallets credited (%d micro-VBV total)",
		ev.EventID, len(bracket), len(credits), sumCredits(credits))
}

// determineOutcome returns a deterministic WIN/TRY/QUIT for an entity in a bracket.
// Pure function of (eventID, entityID, index) — no RNG, no clock, no float.
func determineOutcome(eventID, entityID string, index int) string {
	seed := fmt.Sprintf("%s|%s|%d", eventID, entityID, index)
	h := sha256.Sum256([]byte(seed))
	// Take the first byte mod 3: 0=WIN, 1=TRY, 2=QUIT.
	switch h[0] % 3 {
	case 0:
		return "WIN"
	case 1:
		return "TRY"
	default:
		return "QUIT"
	}
}

// sumCredits totals a per-wallet credit map (uint64, no overflow risk at micro scale).
func sumCredits(m map[string]uint64) uint64 {
	var total uint64
	for _, v := range m {
		total += v
	}
	return total
}
