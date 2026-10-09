//go:build !js && !wasm

package main

import (
	"strings"
	"testing"
)

// TestTheTenantVaultContractRefusesUntilTheRegistryExists pins the refuse-first shape: every blocker
// is NAMED, custody is stated as NO OPTION, the toll is $VBV on Voi and NOT charged, and the wizard
// is SEVEN steps. A contract that claimed availability before the registry exists would be a false
// statement about state - the class this session has been closing all along.
func TestTheTenantVaultContractRefusesUntilTheRegistryExists(t *testing.T) {
	c := tenantVaultSetupContract()
	if c["available"] != false {
		t.Error("the contract must REFUSE while the registry does not exist")
	}
	blockers, ok := c["blocked_by"].([]string)
	if !ok || len(blockers) == 0 {
		t.Fatal("a refusal must NAME every blocker, or it is not a refusal")
	}
	custody, _ := c["custody"].(string)
	if !strings.Contains(custody, "never holds a tenant key") {
		t.Error("the custody rule is NO OPTION and must state it")
	}
	toll, ok := c["toll"].(map[string]any)
	if !ok {
		t.Fatal("the toll must be declared")
	}
	if toll["asset"] != "VBV" || toll["chain"] != "Voi" {
		t.Error("the toll is $VBV on Voi only: there is no other outbound rail")
	}
	if toll["charged"] != false {
		t.Error("nothing is charged while the contract refuses")
	}
	steps, _ := c["steps"].([]string)
	if len(steps) != 7 {
		t.Errorf("the wizard is SEVEN steps; got %d", len(steps))
	}
}
