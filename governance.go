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

// ── Governance System ────────────────────────────────────────────────────────
// Authority through contribution. Leadership combines trust, reputation,
// economic contribution, competitive success, community support, creative output.

// GovernanceWeight represents the weighted factors for leadership.
type GovernanceWeight struct {
	TrustScore        uint64 `json:"trust_score"`
	ReputationScore   uint64 `json:"reputation_score"`
	EconomicScore     uint64 `json:"economic_score"`
	CompetitiveScore  uint64 `json:"competitive_score"`
	CommunityScore    uint64 `json:"community_score"`
	CreativeScore     uint64 `json:"creative_score"`
	TotalScore        uint64 `json:"total_score"`
}

// RegionalGovernor represents a regional governor.
type RegionalGovernor struct {
	Wallet         string    `json:"wallet"`
	Region         string    `json:"region"`
	ElectedAt      time.Time `json:"elected_at"`
	TermExpiresAt  time.Time `json:"term_expires_at"`
	Weight         GovernanceWeight `json:"weight"`
	VotesFor       uint64    `json:"votes_for"`
	VotesAgainst   uint64    `json:"votes_against"`
	Active         bool      `json:"active"`
}

// GovernanceElection tracks elections.
type GovernanceElection struct {
	ID            string    `json:"id"`
	Region        string    `json:"region"`
	Candidates    []string  `json:"candidates"`
	Votes         map[string]uint64 `json:"votes"`
	Winner        string    `json:"winner"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	ClosedAt      time.Time `json:"closed_at"`
}

// GovernanceEngine manages governance.
type GovernanceEngine struct {
	mu         sync.RWMutex
	governors  map[string]*RegionalGovernor
	elections  map[string]*GovernanceElection
}

var governanceEngine = &GovernanceEngine{
	governors: make(map[string]*RegionalGovernor),
	elections: make(map[string]*GovernanceElection),
}

// ComputeWeight computes the governance weight for a wallet.
func (ge *GovernanceEngine) ComputeWeight(wallet string) GovernanceWeight {
	ge.mu.RLock()
	defer ge.mu.RUnlock()

	var reputationScore uint64
	if profile, ok := persistentIdentity.GetProfile(wallet); ok {
		reputationScore = uint64(profile.TotalReputation)
	}

	// Use reputation-based proxy for economic/competitive/community/creative
	economicScore := reputationScore / 4
	competitiveScore := reputationScore / 4
	communityScore := reputationScore / 8
	creativeScore := reputationScore / 8
	trustScore := reputationScore / 2

	total := trustScore + reputationScore + economicScore + competitiveScore + communityScore + creativeScore

	return GovernanceWeight{
		TrustScore:       trustScore,
		ReputationScore:  reputationScore,
		EconomicScore:    economicScore,
		CompetitiveScore: competitiveScore,
		CommunityScore:   communityScore,
		CreativeScore:    creativeScore,
		TotalScore:       total,
	}
}

// RegisterCandidate registers a candidate for regional election.
func (ge *GovernanceEngine) RegisterCandidate(region, wallet string) error {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	election, ok := ge.elections[region]
	if !ok {
		election = &GovernanceElection{
			ID:         fmt.Sprintf("election_%s_%d", region, time.Now().Unix()),
			Region:     region,
			Candidates: []string{},
			Votes:      make(map[string]uint64),
			Status:     "open",
			CreatedAt:  time.Now(),
		}
		ge.elections[region] = election
	}

	if election.Status != "open" {
		return fmt.Errorf("election not open")
	}

	election.Candidates = append(election.Candidates, wallet)
	return nil
}

// Vote casts a weighted vote.
func (ge *GovernanceEngine) Vote(region, voter, candidate string, weight uint64) error {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	election, ok := ge.elections[region]
	if !ok || election.Status != "open" {
		return fmt.Errorf("election not open")
	}

	election.Votes[candidate] += weight
	return nil
}

// CloseElection closes the election and selects a winner.
func (ge *GovernanceEngine) CloseElection(region string) (*RegionalGovernor, error) {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	election, ok := ge.elections[region]
	if !ok {
		return nil, fmt.Errorf("election not found")
	}

	var winner string
	var maxVotes uint64
	for candidate, votes := range election.Votes {
		if votes > maxVotes {
			maxVotes = votes
			winner = candidate
		}
	}

	election.Winner = winner
	election.Status = "closed"
	election.ClosedAt = time.Now()

	governor := &RegionalGovernor{
		Wallet:        winner,
		Region:        region,
		ElectedAt:     time.Now(),
		TermExpiresAt: time.Now().AddDate(0, 3, 0),
		Weight:        ge.computeWeightUnlocked(winner),
		VotesFor:      maxVotes,
		VotesAgainst:  0,
		Active:        true,
	}
	ge.governors[region] = governor

	return governor, nil
}

// GetGovernor returns the governor for a region.
func (ge *GovernanceEngine) GetGovernor(region string) (*RegionalGovernor, bool) {
	ge.mu.RLock()
	defer ge.mu.RUnlock()
	g, ok := ge.governors[region]
	return g, ok
}

// GetElection returns the election for a region.
func (ge *GovernanceEngine) GetElection(region string) (*GovernanceElection, bool) {
	ge.mu.RLock()
	defer ge.mu.RUnlock()
	e, ok := ge.elections[region]
	return e, ok
}

// GetLeaderboard returns top candidates by weight.
func (ge *GovernanceEngine) GetLeaderboard(n int) []*RegionalGovernor {
	ge.mu.RLock()
	defer ge.mu.RUnlock()

	var all []*RegionalGovernor
	for _, g := range ge.governors {
		if g.Active {
			all = append(all, g)
		}
	}
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].Weight.TotalScore > all[j].Weight.TotalScore
	})
	if len(all) > n {
		return all[:n]
	}
	return all
}

func (ge *GovernanceEngine) computeWeightUnlocked(wallet string) GovernanceWeight {
	var reputationScore uint64
	if profile, ok := persistentIdentity.GetProfile(wallet); ok {
		reputationScore = uint64(profile.TotalReputation)
	}

	economicScore := reputationScore / 4
	competitiveScore := reputationScore / 4
	communityScore := reputationScore / 8
	creativeScore := reputationScore / 8
	trustScore := reputationScore / 2

	total := trustScore + reputationScore + economicScore + competitiveScore + communityScore + creativeScore

	return GovernanceWeight{
		TrustScore:       trustScore,
		ReputationScore:  reputationScore,
		EconomicScore:    economicScore,
		CompetitiveScore: competitiveScore,
		CommunityScore:   communityScore,
		CreativeScore:    creativeScore,
		TotalScore:       total,
	}
}

// ── HTTP Handlers ────────────────────────────────────────────────────────────

func (l *Lobby) handleGovernanceWeight(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	weight := governanceEngine.ComputeWeight(wallet)
	writeJSON(w, map[string]interface{}{"success": true, "weight": weight})
}

func (l *Lobby) handleGovernanceRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Region string `json:"region"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if err := governanceEngine.RegisterCandidate(req.Region, wallet); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true})
}

func (l *Lobby) handleGovernanceVote(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Region     string `json:"region"`
		Candidate  string `json:"candidate"`
		Weight     uint64 `json:"weight"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if err := governanceEngine.Vote(req.Region, wallet, req.Candidate, req.Weight); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true})
}

func (l *Lobby) handleGovernanceClose(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Region string `json:"region"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	governor, err := governanceEngine.CloseElection(req.Region)
	if err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true, "governor": governor})
}

func (l *Lobby) handleGovernanceGovernor(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	governor, ok := governanceEngine.GetGovernor(region)
	if !ok {
		writeJSON(w, map[string]interface{}{"success": false, "error": "no governor"})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true, "governor": governor})
}

func (l *Lobby) handleGovernanceElection(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	election, ok := governanceEngine.GetElection(region)
	if !ok {
		writeJSON(w, map[string]interface{}{"success": false, "error": "no election"})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true, "election": election})
}

func (l *Lobby) handleGovernanceLeaderboard(w http.ResponseWriter, r *http.Request) {
	leaderboard := governanceEngine.GetLeaderboard(50)
	writeJSON(w, map[string]interface{}{"success": true, "leaderboard": leaderboard})
}
