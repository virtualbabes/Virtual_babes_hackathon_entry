//go:build !js && !wasm

package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// ════════════════════════════════════════════════════════════════════════════
// SLICE 4 — POWER SCALING IS PERSISTED, VALIDATED AND REPORTED HONESTLY
//
// PowerDivisor and PowerBase are READ at runtime (club_service.go verification,
// oracle_service.go artifact power) and are stored in the registry map, which is
// loaded from networks.json at BOOT. The handler used to mutate the map only —
// so a change took effect and then vanished on the next restart — and it answered
// `{"status":"success"}` even when the focused network did not exist.
// ════════════════════════════════════════════════════════════════════════════

// powerScalingLobby is a lobby whose registry lives in a TEMPORARY working directory.
//
// WHY THE CHDIR IS NOT OPTIONAL. handleUpdatePowerScaling persists through
// saveNetworkConfigs(), which writes the RELATIVE path networks.json — the repository's
// GIT-TRACKED registry, whose mtime and content are part of the working tree. Running the
// test in a temp directory means a test can never rewrite (or reorder) the real registry.
func powerScalingLobby(t *testing.T) *Lobby {
	t.Helper()
	t.Chdir(t.TempDir())

	l := &Lobby{
		DataDir:       t.TempDir(),
		nonces:        map[string]NonceData{},
		broadcast:     make(chan []byte, 8),
		rewardTokens:  map[string]RewardToken{},
		rewardStack:   map[string]uint64{},
		playerBalances: map[string]uint64{},
		availableNetworks: map[string]NetworkConfig{
			"Voi Mainnet": {
				NetworkName: "Voi Mainnet",
				IndexerURLs: []string{"https://mainnet-idx.voi.nodely.dev"},
				NodeURLs:    []string{"https://mainnet-api.voi.nodely.dev"},
				ChainID:     "algorand:r20fSQI8gWe_kFZziNonSPCXLwcQmH_n",
				AlgodToken:  testAlgodSecret,
				// The shipped values, so an unpersisted change is detectable as a REVERSION.
				PowerDivisor: 1000000,
				PowerBase:    50,
			},
		},
	}
	l.adminFocusNetwork = "Voi Mainnet"
	l.saveNetworkConfigs() // seed the registry file this test then verifies
	return l
}

func TestPowerScalingIsAppliedPersistedAndReported(t *testing.T) {
	l := powerScalingLobby(t)
	headers := adminTestAuth(t, l)

	rec, req := adminRequest(http.MethodPost, "/api/admin/update-power", headers, `{"divisor":2000000,"base":75}`)
	l.handleUpdatePowerScaling(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("authorised update = %d; want 200 (%s)", rec.Code, rec.Body.String())
	}

	// 1. The response reports what was APPLIED, never a bare "success".
	var payload struct {
		Status          string  `json:"status"`
		Network         string  `json:"network"`
		PowerDivisor    float64 `json:"power_divisor"`
		PowerBase       int     `json:"power_base"`
		RegistryFile    string  `json:"registry_file"`
		Persisted       bool    `json:"persisted"`
		RestartRequired bool    `json:"restart_required"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode the response: %v", err)
	}
	if payload.Network != "Voi Mainnet" || payload.PowerDivisor != 2000000 || payload.PowerBase != 75 {
		t.Errorf("the response does not report the applied values: %+v", payload)
	}
	if !payload.Persisted || payload.RegistryFile != "networks.json" {
		t.Errorf("the response must state that the change is persisted to the registry file: %+v", payload)
	}
	if payload.RestartRequired {
		t.Error("power scaling is read live from the map, so no restart is required — reporting one would be untrue")
	}

	// 2. The RUNNING registry carries the new values (the club/oracle readers consult this map).
	if got := l.availableNetworks["Voi Mainnet"]; got.PowerDivisor != 2000000 || got.PowerBase != 75 {
		t.Errorf("running registry = %v/%v; want 2000000/75", got.PowerDivisor, got.PowerBase)
	}

	// 3. THE FILE carries them: a FRESH lobby loading the same directory sees them. This is the
	//    restart proof — before the fix the reload answered the shipped 1000000/50.
	reloaded := &Lobby{availableNetworks: map[string]NetworkConfig{}}
	reloaded.loadNetworkConfigs()
	got := reloaded.availableNetworks["Voi Mainnet"]
	if got.PowerDivisor != 2000000 || got.PowerBase != 75 {
		t.Errorf("after a restart the scaling reads %v/%v; want 2000000/75 (the change was not persisted)",
			got.PowerDivisor, got.PowerBase)
	}

	// 4. The write is still REDACTED: the scaling persists, the env-threaded token does not.
	raw, err := os.ReadFile("networks.json")
	if err != nil {
		t.Fatalf("read the registry file: %v", err)
	}
	if !strings.Contains(string(raw), "2000000") {
		t.Error("the persisted registry does not carry the new power divisor")
	}
	if strings.Contains(string(raw), testAlgodSecret) {
		t.Error("the power-scaling write persisted a SECRET value")
	}

	// 5. The change is ANNOUNCED, so connected clients stop showing the previous scaling.
	select {
	case msg := <-l.broadcast:
		if !strings.Contains(string(msg), "lobby_update") {
			t.Errorf("the broadcast is not a lobby_update: %s", string(msg))
		}
		if !strings.Contains(string(msg), "2000000") {
			t.Error("the broadcast does not carry the new scaling")
		}
	case <-time.After(2 * time.Second):
		t.Error("the change was never broadcast; clients would keep the previous scaling")
	}
}

// TestPowerScalingRefusesWithoutMovingAnything pins the value refusals. Each must answer with a
// status AND a reason, and leave both the running registry and the registry file untouched.
func TestPowerScalingRefusesWithoutMovingAnything(t *testing.T) {
	l := powerScalingLobby(t)
	headers := adminTestAuth(t, l)

	before, err := os.ReadFile("networks.json")
	if err != nil {
		t.Fatalf("read the seeded registry: %v", err)
	}

	cases := []struct {
		why  string
		body string
		want int
	}{
		{"a zero divisor divides by zero in every power derivation", `{"divisor":0,"base":75}`, http.StatusBadRequest},
		{"a negative divisor is meaningless", `{"divisor":-1000,"base":75}`, http.StatusBadRequest},
		{"a negative base would produce negative card power", `{"divisor":1000000,"base":-1}`, http.StatusBadRequest},
		{"a body that is not a number is not a scaling change", `{"divisor":"lots","base":75}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.why, func(t *testing.T) {
			rec, req := adminRequest(http.MethodPost, "/api/admin/update-power", headers, tc.body)
			l.handleUpdatePowerScaling(rec, req)
			if rec.Code != tc.want {
				t.Errorf("status = %d; want %d (%s)", rec.Code, tc.want, rec.Body.String())
			}
			if strings.TrimSpace(rec.Body.String()) == "" {
				t.Error("a refusal must state a reason")
			}
			if got := l.availableNetworks["Voi Mainnet"]; got.PowerDivisor != 1000000 || got.PowerBase != 50 {
				t.Errorf("a refused update moved the registry to %v/%v; want 1000000/50", got.PowerDivisor, got.PowerBase)
			}
		})
	}

	after, err := os.ReadFile("networks.json")
	if err != nil {
		t.Fatalf("re-read the registry: %v", err)
	}
	if string(before) != string(after) {
		t.Error("a refused update rewrote the registry file")
	}
}

// TestPowerScalingRefusesAnUnknownNetworkInsteadOfLying pins the false success the old handler
// served: it wrote only `if` the focused network existed, but always answered
// `{"status":"success"}` — reporting a change it had not made.
func TestPowerScalingRefusesAnUnknownNetworkInsteadOfLying(t *testing.T) {
	l := powerScalingLobby(t)
	l.adminFocusNetwork = "Nowhere Mainnet"
	headers := adminTestAuth(t, l)

	before, err := os.ReadFile("networks.json")
	if err != nil {
		t.Fatalf("read the seeded registry: %v", err)
	}

	rec, req := adminRequest(http.MethodPost, "/api/admin/update-power", headers, `{"divisor":2000000,"base":75}`)
	l.handleUpdatePowerScaling(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d; want 409 — a false success is worse than a refusal (%s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Nowhere Mainnet") {
		t.Errorf("the refusal must name the network it could not find: %s", rec.Body.String())
	}
	if got := l.availableNetworks["Voi Mainnet"]; got.PowerDivisor != 1000000 || got.PowerBase != 50 {
		t.Error("a refusal for one network changed ANOTHER network's scaling")
	}

	after, err := os.ReadFile("networks.json")
	if err != nil {
		t.Fatalf("re-read the registry: %v", err)
	}
	if string(before) != string(after) {
		t.Error("a refused update rewrote the registry file")
	}
}

// TestPowerScalingRefusesAnonymousCallers keeps the door on the same signature gate as every other
// admin mutation.
func TestPowerScalingRefusesAnonymousCallers(t *testing.T) {
	l := powerScalingLobby(t)

	rec, req := adminRequest(http.MethodPost, "/api/admin/update-power", nil, `{"divisor":2000000,"base":75}`)
	l.handleUpdatePowerScaling(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous update = %d; want 401", rec.Code)
	}
	if got := l.availableNetworks["Voi Mainnet"]; got.PowerDivisor != 1000000 || got.PowerBase != 50 {
		t.Error("an unauthenticated caller moved the registry")
	}
}
