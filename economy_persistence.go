//go:build !js && !wasm

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// PersistenceSyncWorker handles periodic background saving of the economic state.
// PILLAR 4: Data Resilience.
type PersistenceSyncWorker struct {
	Mu           sync.Mutex
	Router       *TokenSinkRouter
	Lobby        *Lobby // PILLAR 7-E: Reference to Lobby for identity snapshots
	StorageDir   string
	SaveFileName string
	SyncInterval time.Duration
	MaxBackups   int // PILLAR 4: History Retention
}

// NewPersistenceSyncWorker initializes a new background save routine.
func NewPersistenceSyncWorker(router *TokenSinkRouter, lobby *Lobby, storageDir string, interval time.Duration) *PersistenceSyncWorker {
	return &PersistenceSyncWorker{
		Router:       router,
		Lobby:        lobby,
		StorageDir:   storageDir,
		SaveFileName: "economy_state_authoritative.json",
		SyncInterval: interval,
		MaxBackups:   5, // Maintain last 5 states
	}
}

// StartSyncDaemon boots up the continuous non-blocking backup loop.
func (psw *PersistenceSyncWorker) StartSyncDaemon(ctx context.Context) {
	ticker := time.NewTicker(psw.SyncInterval)

	go func() {
		defer ticker.Stop()
		fmt.Printf(" [Persistence Sync] Worker active. Interval: %v | Dir: %s\n", psw.SyncInterval, psw.StorageDir)
		for {
			select {
			case <-ctx.Done():
				fmt.Println(" [Persistence Sync] Stopping database sync worker...")
				return
			case <-ticker.C:
				err := psw.ExecuteSafeDiskWrite()
				if err != nil {
					fmt.Printf(" [Persistence Error] Core state serialization failed: %v\n", err)
				}
			}
		}
	}()
}

// rotateBackups shifts existing backup files to maintain a rolling history.
// PILLAR 4: Forensic Recovery.
func (psw *PersistenceSyncWorker) rotateBackups() error {
	targetPath := filepath.Join(psw.StorageDir, psw.SaveFileName)

	// PILLAR 4: Idempotency check. If no file exists, no rotation is needed.
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return nil
	}

	// 1. Remove the oldest backup if it exists
	oldestBackup := fmt.Sprintf("%s.%d", targetPath, psw.MaxBackups)
	if err := os.Remove(oldestBackup); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to prune oldest backup: %w", err)
	}

	// 2. Shift existing backups (e.g., .4 -> .5, .3 -> .4 ...)
	for i := psw.MaxBackups - 1; i >= 1; i-- {
		source := fmt.Sprintf("%s.%d", targetPath, i)
		dest := fmt.Sprintf("%s.%d", targetPath, i+1)
		if err := os.Rename(source, dest); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to shift backup index %d: %w", i, err)
		}
	}

	// 3. Move current authoritative file to .1
	backup1 := targetPath + ".1"
	return os.Rename(targetPath, backup1)
}

// ExecuteSafeDiskWrite snapshots the memory matrix and commits it to disk atomically.
// PILLAR 2: Industrial Seal. Uses .tmp swap pattern to prevent corruption.
func (psw *PersistenceSyncWorker) ExecuteSafeDiskWrite() error {
	psw.Mu.Lock()
	defer psw.Mu.Unlock()

	if psw.Router == nil {
		return fmt.Errorf("router not initialized")
	}

	// Ensure target directory exists for Render volume mounting
	if err := os.MkdirAll(psw.StorageDir, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// 1. Snapshot Extraction: Extract data under a Read-Lock to minimize game-loop latency.
	psw.Router.Mu.RLock()

	snapshot := RouterSnapshot{
		Timestamp:         time.Now().Unix(),
		GlobalFaucetValue: 0,
		Clubs:             make(map[uint64]ClubTreasuryNode),
		MarketNodes:       make(map[string]EntityMarketNode),
		Districts:         make(map[string]RegionalGovernanceMetric),
		Identities:        make(map[string]WalletLinkInfo),
	}

	if psw.Router.GlobalFaucetPool != nil {
		snapshot.GlobalFaucetValue = *psw.Router.GlobalFaucetPool
	}

	// NOTE: the lobby-owned identity snapshot is taken AFTER the router lock is released (below).
	// Taking `lobby.mutex` inside the router window is the REVERSE of `Lobby.mutex -> tsr.Mu`, which
	// every fee-routing path takes, and the two together are an ABBA cycle (measured in
	// lock_order_gate_test.go). `snapshot.Identities` is a local copy, so nothing is re-read.

	// PILLAR 2: Authoritative Counter Persistence.
	// Capture absolute audit totals to ensure 'Structural Drift' and 'Solvency'
	// checks in economy_audit.go remain valid across restarts.
	if psw.Router.Audit != nil {
		snapshot.AuditInflow = atomic.LoadUint64(&psw.Router.Audit.TotalSystemInputVetted)
		snapshot.AuditAllocated = atomic.LoadUint64(&psw.Router.Audit.TotalSystemAllocated)
		snapshot.AuditSiphoned = atomic.LoadUint64(&psw.Router.Audit.TotalSystemSiphoned)
		snapshot.AuditExited = atomic.LoadUint64(&psw.Router.Audit.TotalRewardsExited)
		snapshot.AuditGhost = atomic.LoadUint64(&psw.Router.Audit.TotalGhostReclaimed)
		snapshot.AuditStagnation = atomic.LoadUint64(&psw.Router.Audit.TotalStagnationFees)
		snapshot.AuditPlatform = atomic.LoadUint64(&psw.Router.Audit.TotalPlatformFees)
	}

	for id, node := range psw.Router.ActiveClubs {
		if node != nil {
			snapshot.Clubs[id] = *node
		}
	}
	for id, node := range psw.Router.MarketNodes {
		if node != nil {
			snapshot.MarketNodes[id] = *node
		}
	}
	for code, metric := range psw.Router.RegionalDistricts {
		if metric != nil {
			snapshot.Districts[code] = *metric
		}
	}

	psw.Router.Mu.RUnlock()

	// Identity extraction happens OUTSIDE the router window (see the note above): it is the only
	// place this worker would otherwise take the lobby lock while holding the router lock.
	if psw.Lobby != nil {
		psw.Lobby.mutex.RLock()
		for k, v := range psw.Lobby.linkedWallets {
			snapshot.Identities[k] = v
		}
		psw.Lobby.mutex.RUnlock()
	}

	// 2. Deterministic Serialization
	dataBytes, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("snapshot marshalling failed: %w", err)
	}

	// 3. Atomic Commit Protocol: Write to .tmp then Rename (Atomic on POSIX).
	targetFilePath := filepath.Join(psw.StorageDir, psw.SaveFileName)
	tempFilePath := targetFilePath + ".tmp"

	if err := os.WriteFile(tempFilePath, dataBytes, 0644); err != nil {
		return fmt.Errorf("failed to write temporary state file: %w", err)
	}

	// 4. Implement Rotator Strategy before finalizing the new save.
	// This shifts current to .1, .1 to .2, etc.
	if err := psw.rotateBackups(); err != nil {
		// PILLAR 4: Graceful Degradation. Log rotation failure but allow the main save to proceed.
		fmt.Printf(" [Persistence Warning] Backup rotation bypassed (check directory permissions): %v\n", err)
	}

	if err := os.Rename(tempFilePath, targetFilePath); err != nil {
		_ = os.Remove(tempFilePath) // Cleanup orphan data trace if rename fails
		return fmt.Errorf("failed to finalize state commit: %w", err)
	}

	fmt.Printf(" [Persistence Sync] Authoritative snapshot committed. Faucet: %d VBV.\n", snapshot.GlobalFaucetValue)

	// P7-D "Remember": persist the autonomous AI civilization alongside core state.
	if psw.Lobby != nil && psw.Lobby.aiEngine != nil {
		if err := psw.Lobby.aiEngine.SaveCitizens(); err != nil {
			fmt.Printf(" [Persistence Warning] AI citizen snapshot skipped: %v\n", err)
		}
	}
	// §23.5: persist bonded assets alongside the AI citizen civilization.
	if psw.Lobby != nil && psw.Lobby.bondedAssets != nil {
		if err := psw.Lobby.bondedAssets.Save(psw.Lobby); err != nil {
			fmt.Printf(" [Persistence Warning] Bonded asset snapshot skipped: %v\n", err)
		}
	}
	// §24.5/§24.6: persist local-model promotion bridge state.
	if psw.Lobby != nil && psw.Lobby.localModelPromotions != nil {
		if err := psw.Lobby.localModelPromotions.Save(psw.Lobby); err != nil {
			fmt.Printf(" [Persistence Warning] Local-model promotion snapshot skipped: %v\n", err)
		}
	}
	return nil
}
