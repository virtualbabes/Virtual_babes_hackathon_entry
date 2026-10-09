//go:build !js && !wasm

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrCorruptBackupFile = errors.New("bootstrap warning: state backup file corrupted or malformed")
	ErrPovertyBootstrap  = errors.New("bootstrap safety: attempt to initialize zero-value faucet from snapshot")
)

// RouterSnapshot represents the serialized state of the economic engine.
// PILLAR 2: Ledger Integrity.
type RouterSnapshot struct {
	Timestamp         int64                               `json:"timestamp"`
	GlobalFaucetValue uint64                              `json:"global_faucet_value"`
	AuditInflow       uint64                              `json:"audit_inflow,omitempty"`
	AuditAllocated    uint64                              `json:"audit_allocated,omitempty"`
	AuditSiphoned     uint64                              `json:"audit_siphoned,omitempty"`
	AuditExited       uint64                              `json:"audit_exited,omitempty"`
	AuditGhost        uint64                              `json:"audit_ghost,omitempty"`
	AuditStagnation   uint64                              `json:"audit_stagnation,omitempty"`
	AuditPlatform     uint64                              `json:"audit_platform,omitempty"`
	Clubs             map[uint64]ClubTreasuryNode         `json:"clubs"`
	MarketNodes       map[string]EntityMarketNode         `json:"market_nodes"`
	Districts         map[string]RegionalGovernanceMetric `json:"districts"`
	Identities        map[string]WalletLinkInfo           `json:"identities,omitempty"` // PILLAR 7-E: Identity Sync
}

// BootstrapEngine handles the restoration of economic state during server boot.
type BootstrapEngine struct {
	Mu         sync.Mutex
	Router     *TokenSinkRouter
	Lobby      *Lobby // PILLAR 7-E: Reference to Lobby for identity restoration
	StorageDir string
	SaveFileName string
}

func NewBootstrapEngine(router *TokenSinkRouter, lobby *Lobby, storageDir string) *BootstrapEngine {
	return &BootstrapEngine{
		Router:       router,
		Lobby:        lobby,
		StorageDir:   storageDir,
		SaveFileName: "economy_state_authoritative.json",
	}
}

// BootstrapAuthoritativeState reads the persistent file log to hydrate live engine memory trees.
// PILLAR 4: Authoritative state reconstruction.
func (be *BootstrapEngine) BootstrapAuthoritativeState() (bool, error) {
	if be.Router == nil {
		return false, errors.New("bootstrap failed: router not provided")
	}

	targetFilePath := filepath.Join(be.StorageDir, be.SaveFileName)

	// 1. Safe Fallback Matrix: If the file does not exist, initialize standard clean default containers
	if _, err := os.Stat(targetFilePath); os.IsNotExist(err) {
		fmt.Println(" [Bootstrap System] Authoritative save state file not found. Initializing empty economic trees.")
		be.loadDefaultStructures()
		return false, nil
	}

	// 2. Read raw byte stream from disk container
	dataBytes, err := os.ReadFile(targetFilePath)
	if err != nil {
		be.loadDefaultStructures()
		return false, fmt.Errorf("bootstrap system failure: unable to read state file path: %w", err)
	}

	// 3. Unmarshal JSON structural schema payload into an isolated container node
	var snapshot RouterSnapshot
	if err := json.Unmarshal(dataBytes, &snapshot); err != nil {
		be.loadDefaultStructures()
		return false, ErrCorruptBackupFile
	}

	// 3.1 Safety Audit: Ensure we aren't bootstrapping a 'black hole' economy.
	//
	// This runs BEFORE the lock because it reads NOTHING shared: `snapshot` is this function's own value,
	// so holding the exclusive router lock to read it was unnecessary work — and an early RELEASE inside
	// the lock window is the one shape the aliased-map gate's MUST-held analysis cannot see (a released
	// branch joins to "not definitely held" after the `if`). The check itself is unchanged.
	if snapshot.GlobalFaucetValue == 0 {
		return false, ErrPovertyBootstrap
	}

	// 4. Secure the runtime pointers via write-lock before updating the engine state matrix
	// NOTE: this window is released EXPLICITLY (before the identity hydration at the end) rather than
	// deferred, because that hydration must NOT run inside it: taking `lobby.mutex` while holding the
	// router lock is the reverse of `Lobby.mutex -> tsr.Mu`, and the pair is an ABBA cycle (measured in
	// lock_order_gate_test.go).
	be.Router.Mu.Lock()

	// Update the dynamic system global rewards pool reference allocation
	if be.Router.GlobalFaucetPool != nil {
		*be.Router.GlobalFaucetPool = snapshot.GlobalFaucetValue
	}

	// PILLAR 2: Authoritative Reconciliation.
	// Restore the absolute ledger counters to ensure the health report
	// accounts for historical vouchers and exits since the start of the season.
	if be.Router.Audit != nil {
		atomic.StoreUint64(&be.Router.Audit.TotalSystemInputVetted, snapshot.AuditInflow)
		atomic.StoreUint64(&be.Router.Audit.TotalSystemAllocated, snapshot.AuditAllocated)
		atomic.StoreUint64(&be.Router.Audit.TotalSystemSiphoned, snapshot.AuditSiphoned)
		atomic.StoreUint64(&be.Router.Audit.TotalRewardsExited, snapshot.AuditExited)
		atomic.StoreUint64(&be.Router.Audit.TotalGhostReclaimed, snapshot.AuditGhost)
		atomic.StoreUint64(&be.Router.Audit.TotalStagnationFees, snapshot.AuditStagnation)
		atomic.StoreUint64(&be.Router.Audit.TotalPlatformFees, snapshot.AuditPlatform)
	}

	// Allocate and populate the thread-safe active club memory maps
	be.Router.ActiveClubs = make(map[uint64]*ClubTreasuryNode)
	for id, node := range snapshot.Clubs {
		// Allocate a local variable copy to ensure pointers hold unique memory boundaries
		capturedNode := node
		be.Router.ActiveClubs[id] = &capturedNode
	}

	// PILLAR 2: AMM State Hydration.
	// Allocate and populate the thread-safe AMM Market Nodes
	be.Router.MarketNodes = make(map[string]*EntityMarketNode)
	for entityID, node := range snapshot.MarketNodes {
		capturedNode := node
		capturedNode.Mu = sync.RWMutex{} // Initialize fresh mutex for runtime
		be.Router.MarketNodes[entityID] = &capturedNode
	}

	// Allocate and populate the thread-safe regional district tracking map nodes
	be.Router.RegionalDistricts = make(map[string]*RegionalGovernanceMetric)
	for code, metric := range snapshot.Districts {
		capturedMetric := metric
		be.Router.RegionalDistricts[code] = &capturedMetric
	}

	// Router window CLOSED before the lobby lock is taken (see the note at the lock site).
	be.Router.Mu.Unlock()

	if be.Lobby != nil && snapshot.Identities != nil {
		be.Lobby.mutex.Lock()
		if be.Lobby.linkedWallets == nil {
			be.Lobby.linkedWallets = make(map[string]WalletLinkInfo)
		}
		for k, v := range snapshot.Identities {
			be.Lobby.linkedWallets[k] = v
		}
		be.Lobby.mutex.Unlock()
	}

	fmt.Printf(" [Bootstrap Success] System state successfully restored from disk frame. Timestamp: %s | Faucet: %d VBV.\n",
		time.Unix(snapshot.Timestamp, 0).Format(time.RFC3339), snapshot.GlobalFaucetValue)

	return true, nil
}

// loadDefaultStructures ensures memory pointers stay initialized even if backup records are missing.
func (be *BootstrapEngine) loadDefaultStructures() {
	be.Router.Mu.Lock()
	defer be.Router.Mu.Unlock()

	if be.Router.ActiveClubs == nil {
		be.Router.ActiveClubs = make(map[uint64]*ClubTreasuryNode)
	}
	if be.Router.MarketNodes == nil {
		be.Router.MarketNodes = make(map[string]*EntityMarketNode)
	}
	if be.Router.RegionalDistricts == nil {
		be.Router.RegionalDistricts = make(map[string]*RegionalGovernanceMetric)
	}
}
