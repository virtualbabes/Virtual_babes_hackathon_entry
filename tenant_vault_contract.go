//go:build !js && !wasm

package main

// THE TENANT VAULT SETUP CONTRACT (Stage A, plan 4) - what the WIZARD reads.
//
// It is served BEFORE the registry exists, in the same form the lease catalogue uses: a contract
// that REFUSES with a STATED reason is honest, while a wizard that silently cannot complete is not.
//
// CUSTODY IS NOT A FEATURE FLAG. The app holds NO tenant key, ever: the tenant hands the vault
// address to their OWN bot, so there is nothing here to configure, revoke or recover.
func tenantVaultSetupContract() map[string]any {
	return map[string]any{
		"available": false,
		"blocked_by": []string{
			"the tenant vault registry does not exist yet (Stage A build 1)",
			"the vault-only rule is not enforced yet: a registered player wallet must be REFUSED",
			"the signature proof is not wired to ONE extracted verifier yet",
		},
		"custody": "NO OPTION: the app never holds a tenant key; the tenant gives the vault address to its OWN bot",
		"toll": map[string]any{
			"asset":   "VBV",
			"chain":   "Voi",
			"rule":    "a tenant pays the toll in $VBV on Voi only; there is no other outbound rail",
			"charged": false,
		},
		"steps": []string{
			"state what is being set up",
			"connect a VAULT-ONLY wallet",
			"prove it by signature",
			"fund it",
			"install the signer bot",
			"confirm the lease and pay the toll in $VBV",
			"seed",
		},
		"seed_rule":        "the seed IS the record/save file and must restore the whole server state; it is planted only after the wizard is complete",
		"records_dispatch": recordsDispatchEnabled(),
	}
}
