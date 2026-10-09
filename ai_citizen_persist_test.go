//go:build !js && !wasm

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestAICitizenPersistenceRoundTrip verifies P7-D "Remember": citizens persisted
// to disk reload with treasury/reputation/learning intact and resume behavior.
func TestAICitizenPersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	l := &Lobby{DataDir: dir}
	ace := NewAICitizenEngine()
	ace.lobby = l

	// Manually register two citizens (bypassing SpawnAI's wallet generation).
	ace.mu.Lock()
	ace.citizens["AI_TEST_1"] = &AICitizen{
		Wallet:           "AI_TEST_1",
		Name:             "Shadow_01",
		Career:           "HeistPlanner",
		Tier:             2,
		Treasury:         7_500_000_000,
		SavingsRate:      0.3,
		InvestmentThresh: 5_000_000_000,
		BusinessCount:    1,
		LastAction:       time.Now().Add(-10 * time.Minute),
		ActionCooldown:   5 * time.Minute,
		LearningXP:       420,
		Reputation:       35,
	}
	ace.citizens["AI_TEST_2"] = &AICitizen{
		Wallet:      "AI_TEST_2",
		Name:        "Ghost_02",
		Career:      "Fence",
		Tier:        1,
		Treasury:    2_000_000_000,
		SavingsRate: 0.4,
		Reputation:  -10,
		LearningXP:  12,
	}
	ace.mu.Unlock()

	// Save
	if err := ace.SaveCitizens(); err != nil {
		t.Fatalf("SaveCitizens failed: %v", err)
	}
	savedPath := filepath.Join(dir, aiCitizensSaveFile)
	if _, err := os.Stat(savedPath); err != nil {
		t.Fatalf("save file not written: %v", err)
	}

	// New engine + load
	ace2 := NewAICitizenEngine()
	ace2.lobby = l
	if err := ace2.LoadCitizens(); err != nil {
		t.Fatalf("LoadCitizens failed: %v", err)
	}

	ace2.mu.RLock()
	defer ace2.mu.RUnlock()
	if len(ace2.citizens) != 2 {
		t.Fatalf("expected 2 citizens after reload, got %d", len(ace2.citizens))
	}
	c1, ok := ace2.citizens["AI_TEST_1"]
	if !ok {
		t.Fatal("AI_TEST_1 missing after reload")
	}
	if c1.Treasury != 7_500_000_000 {
		t.Errorf("treasury not preserved: got %d", c1.Treasury)
	}
	if c1.Reputation != 35 || c1.LearningXP != 420 {
		t.Errorf("reputation/learning not preserved: rep=%d xp=%d", c1.Reputation, c1.LearningXP)
	}
	if c1.ActionCooldown <= 0 {
		t.Error("ActionCooldown should be restored for runtime behavior")
	}
	if c1.LastAction.IsZero() {
		t.Error("LastAction should be preserved")
	}
	c2 := ace2.citizens["AI_TEST_2"]
	if c2.Reputation != -10 {
		t.Errorf("negative reputation not preserved: got %d", c2.Reputation)
	}
}
