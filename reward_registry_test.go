//go:build !js && !wasm

package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// ════════════════════════════════════════════════════════════════════════════
// SLICE 2 — THE REWARD-TOKEN REGISTRY (reward_registry.go)
//
// The registry is the authority for WHAT MAY BE PAID. These tests pin the three
// defects it was built to remove: an unsourced admin token, a template key that
// outlived its registration, and a runtime-writable primary token.
// ════════════════════════════════════════════════════════════════════════════

// rewardLobby carries enough payout state for the REAL scaling pass to run.
func rewardLobby() *Lobby {
	return &Lobby{
		vaultAddress:      "VAULT-ADDR-TEST",
		rewardAssetID:     "40227315",
		initialBaseReward: 5_000_000,
		baseReward:        5_000_000,
		maxFaucetCapacity: 10_000,
		faucetBalance:     10_000, // a full pool, so the scaling ratio saturates near 1.0
		playerBalances:    map[string]uint64{},
		leaderboard:       map[string]PlayerStats{},
		clubs:             map[string]*Club{},
		matchHistory:      map[string]MatchHistory{},
		rewardStack:       map[string]uint64{},
		initialRewards:    map[string]uint64{},
	}
}

func TestRewardRegistrySeedsTheEnvPrimaryEntry(t *testing.T) {
	l := rewardLobby()

	if pruned := l.reconcileRewardRegistryLocked(); len(pruned) != 0 {
		t.Fatalf("a fresh lobby pruned %v; want nothing", pruned)
	}

	entry, seeded := l.rewardTokens["40227315"]
	if !seeded {
		t.Fatal("the env-owned primary token was not seeded")
	}
	if entry.Role != RewardRolePrimary || entry.Source != RewardSourceEnv || !entry.Enabled {
		t.Errorf("primary entry = %+v; want role=primary, source=env, enabled", entry)
	}
	if l.initialRewards["40227315"] != 5_000_000 {
		t.Errorf("the template must carry the env base reward, got %d", l.initialRewards["40227315"])
	}

	l.applyDynamicScalingLocked()
	views := l.rewardTokenViewsLocked()
	if len(views) != 1 {
		t.Fatalf("served views = %d; want 1 (the registry is the single source)", len(views))
	}
	if views[0].AssetID != "40227315" || !views[0].Payable || views[0].ScaledMicro == 0 {
		t.Errorf("served view = %+v; want a payable primary token with a live scaled amount", views[0])
	}
	if views[0].Role != RewardRolePrimary || views[0].Source != RewardSourceEnv {
		t.Errorf("the served role/source must state where authority comes from: %+v", views[0])
	}
}

func TestRewardRegistryPrunesTemplateKeysWithNoEnabledEntry(t *testing.T) {
	l := rewardLobby()
	l.rewardTokens = map[string]RewardToken{
		"111": {AssetID: "111", Role: RewardRoleDistribution, Source: RewardSourceAdmin, Enabled: true},
		"222": {AssetID: "222", Role: RewardRoleTenant, Source: RewardSourceAdmin, Enabled: false},
	}
	// A RESTORED snapshot can hold all four shapes: a live token, an unregistered token, a
	// disabled token, and the empty key the pre-registry handler wrote.
	l.initialRewards = map[string]uint64{
		"111": 1_000_000, "222": 2_000_000, "999": 3_000_000, "": 4_000_000, "40227315": 9_000_000,
	}

	pruned := l.reconcileRewardRegistryLocked()
	if len(pruned) != 3 {
		t.Fatalf("pruned = %v; want the disabled, the unregistered and the empty key", pruned)
	}
	for _, id := range []string{"222", "999", "(empty asset id)"} {
		if !containsString(pruned, id) {
			t.Errorf("pruned = %v; want it to name %q", pruned, id)
		}
	}

	if _, stale := l.initialRewards["999"]; stale {
		t.Error("an unregistered token must not stay in the template")
	}
	if _, off := l.initialRewards["222"]; off {
		t.Error("a disabled token must not stay in the template")
	}
	if _, empty := l.initialRewards[""]; empty {
		t.Error("the empty asset id must be pruned")
	}
	if l.initialRewards["111"] != 1_000_000 {
		t.Error("a registered, enabled token must survive untouched")
	}
	// The env primary's amount comes from the ENVIRONMENT, never from the restored snapshot.
	if l.initialRewards["40227315"] != l.initialBaseReward {
		t.Errorf("primary amount = %d; want the env base reward %d", l.initialRewards["40227315"], l.initialBaseReward)
	}

	// And a pruned token is not merely absent from a list — it is not PAYABLE.
	l.applyDynamicScalingLocked()
	if _, paying := l.rewardStack["999"]; paying {
		t.Error("a pruned token must not be payable")
	}
	if _, paying := l.rewardStack["222"]; paying {
		t.Error("a disabled token must not be payable")
	}
	if l.rewardStack["111"] == 0 {
		t.Error("the surviving token must still scale to a payable amount")
	}
}

func TestRewardRegistryViewsAreRoleOrderedAndStable(t *testing.T) {
	l := rewardLobby()
	l.rewardTokens = map[string]RewardToken{
		"555": {AssetID: "555", Role: RewardRoleLegacy, Source: RewardSourceAdmin, Enabled: true},
		"111": {AssetID: "111", Role: RewardRoleTenant, Source: RewardSourceAdmin, Enabled: true},
		"222": {AssetID: "222", Role: RewardRoleDistribution, Source: RewardSourceAdmin, Enabled: true},
	}
	l.initialRewards = map[string]uint64{"111": 1, "222": 2, "555": 3, "40227315": 5_000_000}
	l.seedPrimaryRewardTokenLocked()

	first := l.rewardTokenViewsLocked()
	if len(first) != 4 {
		t.Fatalf("served views = %d; want 4", len(first))
	}
	got := []string{first[0].AssetID, first[1].AssetID, first[2].AssetID, first[3].AssetID}
	want := []string{"40227315", "222", "111", "555"} // primary, distribution, tenant, legacy
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("served order = %v; want %v", got, want)
		}
	}

	// Map iteration order must not leak into the wire: 25 reads are byte-identical.
	blob, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal the views: %v", err)
	}
	for i := 0; i < 25; i++ {
		next, err := json.Marshal(l.rewardTokenViewsLocked())
		if err != nil {
			t.Fatalf("marshal read %d: %v", i, err)
		}
		if string(next) != string(blob) {
			t.Fatalf("read %d differed from the first: %s vs %s", i, next, blob)
		}
	}
}

// TestRewardRegistryAddRefusesBadBodiesBeforeTouchingTheVault pins the integer-micro boundary and
// the empty-asset-id hole. Every case below is refused BEFORE the indexer opt-in lookup, which is
// why this test needs no network: the refusal order is part of the contract.
func TestRewardRegistryAddRefusesBadBodiesBeforeTouchingTheVault(t *testing.T) {
	l := rewardLobby()
	l.reconcileRewardRegistryLocked()
	beforeTemplate := len(l.initialRewards)
	headers := adminTestAuth(t, l)

	cases := []struct {
		name     string
		body     string
		wantCode int
		wantText string
	}{
		{"a float `amount` is not a field", `{"asset_id":"999","amount":5.0}`, http.StatusBadRequest, "amount_micro"},
		{"empty asset id", `{"asset_id":"","amount_micro":1}`, http.StatusBadRequest, "non-empty decimal"},
		{"zero asset id", `{"asset_id":"0","amount_micro":1}`, http.StatusBadRequest, "non-empty decimal"},
		{"non-numeric asset id", `{"asset_id":"abc","amount_micro":1}`, http.StatusBadRequest, "non-empty decimal"},
		{"the env primary token", `{"asset_id":"40227315","amount_micro":1}`, http.StatusConflict, "ENV-owned primary"},
		{"a role the panel may not create", `{"asset_id":"999","amount_micro":1,"role":"primary"}`, http.StatusBadRequest, "cannot be created here"},
		{"over the admin cap", `{"asset_id":"999","amount_micro":5000000001}`, http.StatusBadRequest, "exceeds the maximum"},
		{"decimals out of range", `{"asset_id":"999","amount_micro":1,"decimals":19}`, http.StatusBadRequest, "decimals"},
	}

	for _, tc := range cases {
		rec, req := adminRequest(http.MethodPost, "/api/reward/add", headers, tc.body)
		l.handleAdminAddReward(rec, req)
		if rec.Code != tc.wantCode {
			t.Errorf("%s: code = %d; want %d (%s)", tc.name, rec.Code, tc.wantCode, strings.TrimSpace(rec.Body.String()))
			continue
		}
		if !strings.Contains(rec.Body.String(), tc.wantText) {
			t.Errorf("%s: body %q must state %q", tc.name, strings.TrimSpace(rec.Body.String()), tc.wantText)
		}
	}

	if _, registered := l.rewardTokens["999"]; registered {
		t.Error("a refused registration must not enter the registry")
	}
	if len(l.initialRewards) != beforeTemplate {
		t.Errorf("a refused registration moved the template: %d keys, want %d", len(l.initialRewards), beforeTemplate)
	}
}

// TestRewardRegistryEnvAmountParsingIsExactIntegers pins the parser that replaced the discarding
// strconv.ParseUint: the DOCUMENTED spelling "5.0" used to read as 0 (a dead base reward), and a
// fraction must convert exactly without any float multiplication.
func TestRewardRegistryEnvAmountParsingIsExactIntegers(t *testing.T) {
	ok := []struct {
		raw  string
		want uint64
	}{
		{"5", 5_000_000},
		{"5.0", 5_000_000}, // the documented spelling that used to parse as 0
		{"5.5", 5_500_000},
		{"0.000001", 1},
		{"0", 0},
		{".5", 500_000},
		{"5.", 5_000_000},
		{"+7", 7_000_000},
		{"  12.25  ", 12_250_000},
		{"1234567.123456", 1_234_567_123_456},
	}
	for _, tc := range ok {
		micro, parsed, note := parseRewardEnvMicro(tc.raw)
		if !parsed {
			t.Errorf("parseRewardEnvMicro(%q) refused (%s); want %d micro", tc.raw, note, tc.want)
			continue
		}
		if micro != tc.want {
			t.Errorf("parseRewardEnvMicro(%q) = %d; want %d", tc.raw, micro, tc.want)
		}
	}

	refused := []string{"", "   ", "abc", "5.1234567", "-5", "5,0", "0x10", "1e6", "..", "5..0"}
	for _, raw := range refused {
		if micro, parsed, note := parseRewardEnvMicro(raw); parsed {
			t.Errorf("parseRewardEnvMicro(%q) = %d, accepted; want a refusal with a reason (note was %q)", raw, micro, note)
		} else if note == "" {
			t.Errorf("parseRewardEnvMicro(%q) refused without stating why", raw)
		}
	}
}
