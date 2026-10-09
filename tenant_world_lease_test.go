//go:build !js && !wasm

package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// ════════════════════════════════════════════════════════════════════════════
// SLICE 5 — THE TENANT WORLD LEASE (DESIGN ONLY: create REFUSES with reasons)
//
// Before this pass the create door accepted a caller-supplied `monthly_rate`, wrote
// it into a process-global map that is never persisted and never billed, and
// answered success. So a client could declare its own price for something that did
// not exist, and no surface ever read it.
//
// The refusal now lives at the DOMAIN owner (LeaseEngine.CreateLease), so no caller
// can create a lease by going around the door.
// ════════════════════════════════════════════════════════════════════════════

func leasePost(t *testing.T, body string) (int, map[string]interface{}) {
	t.Helper()
	rec, req := adminRequest(http.MethodPost, "/api/lease/create?wallet=0xTENANT", nil, body)
	(&Lobby{}).handleLeaseCreate(rec, req)

	var payload map[string]interface{}
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode the lease response: %v (%s)", err, rec.Body.String())
		}
	}
	return rec.Code, payload
}

func stringList(t *testing.T, raw interface{}) []string {
	t.Helper()
	items, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// TestTenantWorldLeaseCreateRefusesAtTheDomainOwner pins that the refusal is not a handler-level
// courtesy: the engine itself cannot mint a lease while nothing can host one.
func TestTenantWorldLeaseCreateRefusesAtTheDomainOwner(t *testing.T) {
	const tenant = "0xDOMAIN-TENANT"
	lease, err := leaseEngine.CreateLease(tenant, "economy", 999_999_999, 30)
	if err == nil || lease != nil {
		t.Fatalf("the domain owner created a lease (%+v) while no world can host it", lease)
	}
	for _, blocker := range TenantWorldLeaseBlockers() {
		if !strings.Contains(err.Error(), blocker) {
			t.Errorf("the refusal does not name the blocker %q: %v", blocker, err)
		}
	}
	if !strings.Contains(err.Error(), "IGNORED") {
		t.Errorf("the refusal must state that the caller's rate is ignored: %v", err)
	}
	if leases := leaseEngine.GetLeases(tenant); len(leases) != 0 {
		t.Errorf("a refused creation stored %d lease(s)", len(leases))
	}
}

// TestLeaseCreateRefusesTheHttpDoorWithAStatedReason pins the served refusal: 409, every blocker,
// and no client-declared price honoured.
func TestLeaseCreateRefusesTheHttpDoorWithAStatedReason(t *testing.T) {
	code, payload := leasePost(t, `{"system_type":"economy","monthly_rate":1,"duration_days":30}`)
	if code != http.StatusConflict {
		t.Fatalf("status = %d; want 409 (a well-formed request for a world that does not exist)", code)
	}
	if payload["success"] != false {
		t.Errorf("success = %v; want false", payload["success"])
	}
	if payload["spawned"] != false {
		t.Error("the refusal must state that nothing was spawned")
	}
	if payload["creation_available"] != false {
		t.Error("the refusal must state that creation is unavailable")
	}
	if payload["state"] != TenantWorldLeaseStates[0] {
		t.Errorf("state = %v; want %q (the only reachable state)", payload["state"], TenantWorldLeaseStates[0])
	}
	if blocked := stringList(t, payload["creation_blocked_by"]); len(blocked) != len(TenantWorldLeaseBlockers()) {
		t.Errorf("creation_blocked_by = %v; want the full blocker list", blocked)
	}
	if payload["client_rate_ignored"] != true {
		t.Error("a request carrying a rate must be told the rate is ignored")
	}
	if rule, _ := payload["price_rule"].(string); rule == "" {
		t.Error("the refusal must state the price rule")
	}
	if errText, _ := payload["error"].(string); !strings.Contains(errText, "cannot be created yet") {
		t.Errorf("the refusal must say what it is refusing: %q", errText)
	}
}

// TestLeaseCreateRefusesMalformedRequestsBeforeTheStructuralRefusal keeps the two kinds of refusal
// distinct: a bad request is not the same fact as a missing world.
func TestLeaseCreateRefusesMalformedRequestsBeforeTheStructuralRefusal(t *testing.T) {
	cases := []struct {
		why   string
		body  string
		match string
	}{
		{"no system type", `{"duration_days":30}`, "must name the system type"},
		{"a system that is not catalogued", `{"system_type":"time-machine","duration_days":30}`, "not a catalogued system"},
		{"no duration", `{"system_type":"economy"}`, "positive duration"},
		{"a negative duration", `{"system_type":"economy","duration_days":-5}`, "positive duration"},
		{"a body that is not JSON", `not json`, "invalid body"},
	}
	for _, tc := range cases {
		t.Run(tc.why, func(t *testing.T) {
			code, payload := leasePost(t, tc.body)
			if code == http.StatusConflict {
				t.Error("a malformed request answered 409 (the world's fault) instead of refusing the request")
			}
			if payload["success"] != false {
				t.Errorf("success = %v; want false", payload["success"])
			}
			if errText, _ := payload["error"].(string); !strings.Contains(errText, tc.match) {
				t.Errorf("error = %q; want it to contain %q", errText, tc.match)
			}
		})
	}
}

// TestLeaseAvailableServesTheShapeTheClientsRead pins the contract mismatch this pass fixed: the
// route serves `systems[]`, while the World Dashboard read `leases[].price_micro` — a shape this
// route has never produced, so its infrastructure panel could only ever answer "No leases
// available".
func TestLeaseAvailableServesTheShapeTheClientsRead(t *testing.T) {
	rec, req := adminRequest(http.MethodGet, "/api/lease/available", nil, "")
	(&Lobby{}).handleLeaseAvailable(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200", rec.Code)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode the catalogue: %v", err)
	}

	systems, ok := payload["systems"].([]interface{})
	if !ok || len(systems) == 0 {
		t.Fatalf("systems = %v; want the served catalogue", payload["systems"])
	}
	for _, raw := range systems {
		system, ok := raw.(map[string]interface{})
		if !ok {
			t.Fatalf("a catalogue entry is not an object: %v", raw)
		}
		for _, field := range []string{"type", "name", "description", "monthly_rate"} {
			if _, present := system[field]; !present {
				t.Errorf("a catalogue entry is missing %q: %v", field, system)
			}
		}
		if rate, ok := system["monthly_rate"].(float64); !ok || rate <= 0 {
			t.Errorf("monthly_rate must be a positive integer micro amount: %v", system["monthly_rate"])
		}
	}

	// The route must NOT serve a `leases` key: serving one would let a client keep reading a shape
	// the catalogue does not have — which is exactly how the panel stayed empty.
	if _, present := payload["leases"]; present {
		t.Error("the catalogue route must not serve `leases`; the catalogue is `systems[]`")
	}
	if payload["creation_available"] != false {
		t.Error("the catalogue must state that creation is unavailable")
	}
	if len(stringList(t, payload["creation_blocked_by"])) == 0 {
		t.Error("the catalogue must state WHY creation is unavailable")
	}
	if states := stringList(t, payload["states"]); len(states) != len(TenantWorldLeaseStates) {
		t.Errorf("states = %v; want the declared lifecycle vocabulary", states)
	}
	if rule, _ := payload["price_rule"].(string); rule == "" {
		t.Error("the catalogue must state the price rule")
	}
}

// TestLeaseListIsEmptyNonNullAndSaysWhy pins the list contract: [] (never null), zero count, and the
// reason — an empty list must never be read as "you have nothing leased".
func TestLeaseListIsEmptyNonNullAndSaysWhy(t *testing.T) {
	rec, req := adminRequest(http.MethodGet, "/api/lease/list?wallet=0xTENANT", nil, "")
	(&Lobby{}).handleLeaseList(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, `"leases":null`) {
		t.Errorf("an absent lease list must serialise as [] and never null: %s", body)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode the lease list: %v", err)
	}
	leases, ok := payload["leases"].([]interface{})
	if !ok {
		t.Fatalf("leases = %v; want an array", payload["leases"])
	}
	if len(leases) != 0 {
		t.Errorf("leases = %v; want none (creation refuses)", leases)
	}
	if payload["creation_available"] != false || len(stringList(t, payload["creation_blocked_by"])) == 0 {
		t.Error("the list must state that creation is unavailable and why")
	}
}
