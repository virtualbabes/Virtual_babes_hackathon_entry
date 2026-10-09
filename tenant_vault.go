//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// THE TENANT VAULT REGISTRY (Stage A build 1, plan 4).
//
// A tenant leases the architecture and seeds their own records from a VAULT-ONLY address, which they
// give to their OWN bot. The app holds NO tenant key, ever - so this registry stores an ADDRESS and
// nothing that could sign with it.
//
// VAULT-ONLY IS ENFORCED HERE, not asked for in a document: an address the engine already knows as a
// PLAYER cannot be a tenant vault. A vault that also plays is a player wallet wearing a label, and the
// record rail asserts sender == vault on every read.
//
// The signature PROOF is not wired yet (Stage A build 2), so a registration is stored as `proposed`
// and Proven stays false. Nothing is charged and nothing is seeded here.
const tenantVaultFile = "tenant_vaults.json"

const tenantVaultStateProposed = "proposed"

// TenantVault is ONE registered tenant vault.
type TenantVault struct {
	Vault        string    `json:"vault"`
	Tenant       string    `json:"tenant"`
	Label        string    `json:"label,omitempty"`
	Network      string    `json:"network"`
	State        string    `json:"state"`
	Proven       bool      `json:"proven"`
	RegisteredAt time.Time `json:"registered_at"`
}

// TenantVaultRegistry holds the registered vaults.
type TenantVaultRegistry struct {
	mu     sync.RWMutex
	vaults map[string]*TenantVault
}

func NewTenantVaultRegistry() *TenantVaultRegistry {
	return &TenantVaultRegistry{vaults: make(map[string]*TenantVault)}
}

// Register performs the VAULT-ONLY checks and records the vault in state `proposed`.
func (r *TenantVaultRegistry) Register(l *Lobby, tenant, vault, label string) (*TenantVault, error) {
	if l == nil {
		return nil, fmt.Errorf("lobby not initialized")
	}
	tenant = strings.TrimSpace(tenant)
	vault = strings.TrimSpace(vault)
	if tenant == "" {
		return nil, fmt.Errorf("a tenant vault needs the tenant wallet that owns the lease")
	}
	if vault == "" {
		return nil, fmt.Errorf("a tenant vault needs an address")
	}
	if strings.EqualFold(tenant, vault) {
		return nil, fmt.Errorf("the vault must be a DIFFERENT wallet from the tenant: the app can never sign for it, so the same wallet proves nothing")
	}
	if isRegisteredPlayerWallet(l, vault) {
		return nil, fmt.Errorf("the vault %s is a PLAYER wallet on this server, so it is not vault-only: register an address that never plays", vault)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for existing := range r.vaults {
		if strings.EqualFold(existing, vault) {
			return nil, fmt.Errorf("the vault %s is already registered", vault)
		}
	}
	tv := &TenantVault{Vault: vault, Tenant: tenant, Label: label, Network: "Voi", State: tenantVaultStateProposed, RegisteredAt: time.Now().UTC()}
	r.vaults[vault] = tv
	cp := *tv
	return &cp, nil
}

// isRegisteredPlayerWallet answers the VAULT-ONLY question CASE-INSENSITIVELY against BOTH player
// registries: a wallet that holds a balance or a leaderboard row is a player, whatever its spelling.
func isRegisteredPlayerWallet(l *Lobby, wallet string) bool {
	// THE LOBBY LOCK, because these are LOBBY-OWNED maps: reading them unlocked is the map race the
	// unfenced-map gate exists to catch - and it caught THIS function on its first run.
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	for key := range l.playerBalances {
		if strings.EqualFold(key, wallet) {
			return true
		}
	}
	for key := range l.leaderboard {
		if strings.EqualFold(key, wallet) {
			return true
		}
	}
	return false
}

// Snapshot returns private COPIES, so a caller can marshal them without holding the registry lock.
func (r *TenantVaultRegistry) Snapshot() []*TenantVault {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*TenantVault, 0, len(r.vaults))
	for _, v := range r.vaults {
		cp := *v
		out = append(out, &cp)
	}
	return out
}

// Save writes the registry to its file. THE CHAIN RECORD IS NOT DISPATCHED YET, deliberately: its
// family and its note prefix land WITH it in the SAME change (A6), because
// dispatchBlockchainSnapshot rightly REFUSES a prefix that is not a declared family - a mirror added
// here first would be refused, not written, and would look like a broken writer.
func (r *TenantVaultRegistry) Save(l *Lobby) error {
	if l == nil {
		return fmt.Errorf("lobby not initialized")
	}
	snapshot := r.Snapshot()
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tenant vaults: %w", err)
	}
	if err := os.WriteFile(l.getDataPath(tenantVaultFile), data, 0644); err != nil {
		return err
	}
	// THE CHAIN RECORD IS A TRANSPORT MIRROR of the same payload the file write produces: the snapshot
	// above is private COPIES, so the record never sees a live map. Its family is DECLARED (A6), which
	// is what makes the write LEGAL - the writer refuses an undeclared prefix by design.
	l.saveBlockchainStateSnapshotLocked(NotePrefixTenantVaultSnapshot, snapshot)
	return nil
}

// Load rehydrates the registry from disk.
func (r *TenantVaultRegistry) Load(l *Lobby) error {
	if l == nil {
		return fmt.Errorf("lobby not initialized")
	}
	data, err := os.ReadFile(l.getDataPath(tenantVaultFile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var vaults []*TenantVault
	if err := json.Unmarshal(data, &vaults); err != nil {
		return fmt.Errorf("failed to unmarshal tenant vaults: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.vaults = make(map[string]*TenantVault, len(vaults))
	for _, v := range vaults {
		if v != nil && v.Vault != "" {
			r.vaults[v.Vault] = v
		}
	}
	return nil
}
