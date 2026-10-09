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

// ── Launchpad System ─────────────────────────────────────────────────────────
// Accelerate creators. Projects should not merely advertise — they should
// integrate. Successful launches immediately become part of the ecosystem.
// The launchpad creates lasting economic activity rather than temporary hype.

// LaunchStatus represents the status of a launch.
type LaunchStatus string

const (
	LaunchDraft     LaunchStatus = "draft"
	LaunchPending   LaunchStatus = "pending"
	LaunchActive    LaunchStatus = "active"
	LaunchFunded    LaunchStatus = "funded"
	LaunchFailed    LaunchStatus = "failed"
	LaunchIntegrated LaunchStatus = "integrated"
)

// LaunchProject represents a creator launch project.
type LaunchProject struct {
	ID              string       `json:"id"`
	Creator         string       `json:"creator"`
	Title           string       `json:"title"`
	Description     string       `json:"description"`
	Category        string       `json:"category"`
	GoalMicro       uint64       `json:"goal_micro"`
	RaisedMicro     uint64       `json:"raised_micro"`
	Status          LaunchStatus `json:"status"`
	Backers         []string     `json:"backers"`
	BackerCounts    map[string]uint64 `json:"backer_counts"`
	CreatedAt       time.Time    `json:"created_at"`
	LaunchDate      time.Time    `json:"launch_date"`
	IntegratedAt    time.Time    `json:"integrated_at"`
	RewardTiers     []RewardTier `json:"reward_tiers"`
}

// RewardTier represents a backer reward tier.
type RewardTier struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	MinMicro    uint64 `json:"min_micro"`
	MaxBackers  int    `json:"max_backers"`
	CurrentBackers int `json:"current_backers"`
}

// LaunchEngine manages launchpad projects.
type LaunchEngine struct {
	mu       sync.RWMutex
	projects map[string]*LaunchProject
}

var launchEngine = &LaunchEngine{
	projects: make(map[string]*LaunchProject),
}

// CreateLaunch creates a new launch project.
func (le *LaunchEngine) CreateLaunch(creator, title, description, category string, goalMicro uint64, launchDate time.Time, tiers []RewardTier) *LaunchProject {
	le.mu.Lock()
	defer le.mu.Unlock()

	project := &LaunchProject{
		ID:           fmt.Sprintf("launch_%d", time.Now().UnixNano()),
		Creator:      creator,
		Title:        title,
		Description:  description,
		Category:     category,
		GoalMicro:    goalMicro,
		RaisedMicro:  0,
		Status:       LaunchDraft,
		Backers:      []string{},
		BackerCounts: make(map[string]uint64),
		CreatedAt:    time.Now(),
		LaunchDate:   launchDate,
		RewardTiers:  tiers,
	}
	le.projects[project.ID] = project
	return project
}

// BackLaunch backs a launch project.
func (le *LaunchEngine) BackLaunch(projectID, backer string, amount uint64) error {
	le.mu.Lock()
	defer le.mu.Unlock()

	project, ok := le.projects[projectID]
	if !ok {
		return fmt.Errorf("project not found")
	}
	if project.Status != LaunchActive {
		return fmt.Errorf("project not active")
	}

	project.RaisedMicro += amount
	project.Backers = append(project.Backers, backer)
	project.BackerCounts[backer] += amount

	if project.RaisedMicro >= project.GoalMicro {
		project.Status = LaunchFunded
	}

	return nil
}

// ActivateLaunch activates a launch.
func (le *LaunchEngine) ActivateLaunch(projectID string) error {
	le.mu.Lock()
	defer le.mu.Unlock()

	project, ok := le.projects[projectID]
	if !ok {
		return fmt.Errorf("project not found")
	}
	project.Status = LaunchActive
	project.LaunchDate = time.Now()
	return nil
}

// IntegrateLaunch integrates a successful launch into the ecosystem.
func (le *LaunchEngine) IntegrateLaunch(projectID string) error {
	le.mu.Lock()
	defer le.mu.Unlock()

	project, ok := le.projects[projectID]
	if !ok {
		return fmt.Errorf("project not found")
	}
	if project.Status != LaunchFunded {
		return fmt.Errorf("project not funded")
	}
	project.Status = LaunchIntegrated
	project.IntegratedAt = time.Now()
	return nil
}

// GetLaunches returns all launches.
func (le *LaunchEngine) GetLaunches(status LaunchStatus) []*LaunchProject {
	le.mu.RLock()
	defer le.mu.RUnlock()

	var results []*LaunchProject
	for _, p := range le.projects {
		if status == "" || p.Status == status {
			results = append(results, p)
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})
	return results
}

// GetLaunch returns a launch by ID.
func (le *LaunchEngine) GetLaunch(projectID string) (*LaunchProject, bool) {
	le.mu.RLock()
	defer le.mu.RUnlock()
	p, ok := le.projects[projectID]
	return p, ok
}

// GetLaunchesByCreator returns launches by a creator.
func (le *LaunchEngine) GetLaunchesByCreator(creator string) []*LaunchProject {
	le.mu.RLock()
	defer le.mu.RUnlock()

	var results []*LaunchProject
	for _, p := range le.projects {
		if p.Creator == creator {
			results = append(results, p)
		}
	}
	return results
}

// ── HTTP Handlers ────────────────────────────────────────────────────────────

func (l *Lobby) handleLaunchCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Category    string `json:"category"`
		GoalMicro   uint64 `json:"goal_micro"`
		LaunchDate  string `json:"launch_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	launchDate, _ := time.Parse("2006-01-02", req.LaunchDate)
	project := launchEngine.CreateLaunch(wallet, req.Title, req.Description, req.Category, req.GoalMicro, launchDate, nil)
	writeJSON(w, map[string]interface{}{"success": true, "project": project})
}

func (l *Lobby) handleLaunchBack(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID string `json:"project_id"`
		Amount    uint64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if err := launchEngine.BackLaunch(req.ProjectID, wallet, req.Amount); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true})
}

func (l *Lobby) handleLaunchActivate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	if err := launchEngine.ActivateLaunch(req.ProjectID); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true})
}

func (l *Lobby) handleLaunchIntegrate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	if err := launchEngine.IntegrateLaunch(req.ProjectID); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true})
}

func (l *Lobby) handleLaunches(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	launches := launchEngine.GetLaunches(LaunchStatus(status))
	writeJSON(w, map[string]interface{}{"success": true, "launches": launches})
}

func (l *Lobby) handleLaunchGet(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	project, ok := launchEngine.GetLaunch(projectID)
	if !ok {
		writeJSON(w, map[string]interface{}{"success": false, "error": "not found"})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true, "project": project})
}

func (l *Lobby) handleLaunchesByCreator(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	launches := launchEngine.GetLaunchesByCreator(wallet)
	writeJSON(w, map[string]interface{}{"success": true, "launches": launches})
}
