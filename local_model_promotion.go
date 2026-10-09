//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

// Â§24.5 / Â§24.6 â€” Local-Model Promotion bridge (THIN integration only).
//
// The repo UTILIZES the user's ornith compile harness (setup_custom_quant_ornith.bat +
// llama_tools/ornith-matrix.imatrix); it does NOT own or reimplement llama.cpp tooling.
// This file provides the in-game promotion path: a birth-certified bot child that reaches
// maturity may be promoted to run its own Local model. Eligibility is gated on the real
// compliance signals we already track (Â§27.7.3): Certified == true and
// BlackMarketAdopted == false, plus an owner-match check, plus the user's ornith harness
// Level gate (MinPromotionLevel, default 25) and a 50% GPU/CPU/RAM resource cap policy.
//
// The actual imatrix/server compilation is performed by the existing Windows .bat harness.
// We invoke it best-effort when VB_MODEL_COMPILE_ENABLED=true and on Windows; otherwise we
// record status=requires_manual_build with the exact command for the operator to run.
// We never fabricate a per-citizen model blob â€” the harness owns that artifact.

const MinPromotionLevel = 25 // Â§24.5/Â§24.6: ornith harness Level gate (L >= 25)

// Â§31.1 â€” Model tier / backend / port model (DESIGN LOCKED 2026-08-30).
// A tier is a SCALABLE AI POWER FIGURE. The backend mode (gpu/cpu/auto) auto-detects an
// advisory cap; we NEVER enforce a hard cap (operator may exceed). Console/Android/iPhone
// get their own deferred build; until then server-hosted.
const (
	// ORNITH_MODEL_TIER env: tiny (<=1GB, CPU/console) | small (~2-3GB) | base (6.6GB Q4, GPU).
	OrnithTierTiny  = "tiny"
	OrnithTierSmall = "small"
	OrnithTierBase  = "base"
	// ORNITH_BACKEND env: gpu | cpu | auto (auto probes nvidia-smi).
	OrnithBackendGPU  = "gpu"
	OrnithBackendCPU  = "cpu"
	OrnithBackendAuto = "auto"
	// Port model: derived free port so multiple identities don't collide on Ollama's 11434.
	OrnithBasePort    = 11434
	OrnithPortRange   = 1000
	OrnithOllamaAlias = "ornith-" // identity alias prefix when ORNITH_OLLAMA_GATEWAY=1
)

// GetOrnithTier returns the configured model tier (defaults to base).
func GetOrnithTier() string {
	t := os.Getenv("ORNITH_MODEL_TIER")
	if t == "" {
		return OrnithTierBase
	}
	return t
}

// GetOrnithBackend returns the configured backend (defaults to auto).
func GetOrnithBackend() string {
	b := os.Getenv("ORNITH_BACKEND")
	if b == "" {
		return OrnithBackendAuto
	}
	return b
}

// DerivedPort computes a free port for an identity: base + hash(identity)%range.
// Avoids collisions with Ollama's default gateway and other identities.
func DerivedPort(identity string) int {
	h := uint32(0)
	for i := 0; i < len(identity); i++ {
		h = h*31 + uint32(identity[i])
	}
	return OrnithBasePort + int(h%uint32(OrnithPortRange))
}

// HardwareScanAdvisory runs a best-effort hardware probe (VRAM via nvidia-smi on gpu/auto,
// CPU cores + RAM otherwise) and returns a RECOMMENDED max local-LLM instance count.
// ADVISORY ONLY â€” never enforced. Operator may promote beyond the suggestion.
func HardwareScanAdvisory() int {
	// Conservative default; raised if probe finds capacity. No hard cap.
	recommended := 3
	if b := GetOrnithBackend(); b == OrnithBackendCPU {
		// CPU-only: bound by cores; assume ~1 instance per 4 logical cores, floor 1.
		if n := runtime.NumCPU() / 4; n > recommended {
			recommended = n
		}
		return recommended
	}
	// gpu/auto: probe nvidia-smi for VRAM; each base instance ~7GB.
	if out, err := exec.Command("nvidia-smi", "--query-gpu=memory.total", "--format=csv,noheader,nounits").Output(); err == nil {
		var vramGB int
		fmt.Sscanf(string(out), "%d", &vramGB)
		if vramGB > 0 {
			if n := vramGB / 7; n > recommended {
				recommended = n
			}
		}
	}
	return recommended
}

// LocalModelPromotion records one bot-child â†’ local-model promotion request.
type LocalModelPromotion struct {
	CitizenWallet string    `json:"citizen_wallet"`
	OwnerWallet   string    `json:"owner_wallet"`
	AttestedLevel int       `json:"attested_level"` // operator-attested ornith harness Level (>= MinPromotionLevel)
	Status        string    `json:"status"`         // pending | building | ready | requires_manual_build | rejected
	RunnerPath    string    `json:"runner_path,omitempty"`
	RequestedAt   time.Time `json:"requested_at"`
	CompletedAt   time.Time `json:"completed_at,omitempty"`
	RejectReason  string    `json:"reject_reason,omitempty"`
}

// LocalModelPromotionRegistry tracks promotion requests for a lobby.
type LocalModelPromotionRegistry struct {
	mu          sync.RWMutex
	Promotions  map[string]*LocalModelPromotion `json:"promotions"` // key: citizenWallet
}

const localModelPromotionFile = "local_model_promotions.json"

// NewLocalModelPromotionRegistry returns an empty registry.
func NewLocalModelPromotionRegistry() *LocalModelPromotionRegistry {
	return &LocalModelPromotionRegistry{
		Promotions: make(map[string]*LocalModelPromotion),
	}
}

// RequestLocalModelPromotion validates eligibility and opens a promotion request.
// Eligibility (Â§27.7.3 + Â§24.5/Â§24.6):
//   - citizen must exist and be Certified (legitimate spawn, own-wallet)
//   - citizen must NOT be BlackMarketAdopted
//   - requesterWallet must be the citizen's OwnerWallet (only the in-game owner/holder)
//   - attestedLevel must be >= MinPromotionLevel
func (r *LocalModelPromotionRegistry) RequestLocalModelPromotion(ace *AICitizenEngine, citizenWallet, requesterWallet string, attestedLevel int) (*LocalModelPromotion, error) {
	if ace == nil {
		return nil, fmt.Errorf("citizen engine not initialized")
	}
	citizen, ok := ace.GetCitizen(citizenWallet)
	if !ok {
		return nil, fmt.Errorf("citizen %s not found", citizenWallet)
	}
	if !citizen.Certified {
		return nil, fmt.Errorf("citizen %s is not birth-certified (ineligible for local-model promotion)", citizenWallet)
	}
	if citizen.BlackMarketAdopted {
		return nil, fmt.Errorf("citizen %s is black-market adopted (ineligible for local-model promotion)", citizenWallet)
	}
	if citizen.OwnerWallet != requesterWallet {
		return nil, fmt.Errorf("requester %s is not the in-game owner of citizen %s", requesterWallet, citizenWallet)
	}
	if attestedLevel < MinPromotionLevel {
		return nil, fmt.Errorf("ornith harness Level %d below required %d", attestedLevel, MinPromotionLevel)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.Promotions[citizenWallet]; ok && existing.Status != "rejected" {
		return existing, nil // idempotent â€” return the in-flight request
	}
	// Â§31.1: NO enforced cap on local-LLM citizen/pet/children count. The harness hardware scan
	// RECOMMENDS a max (HardwareScanAdvisory) but the gate never blocks on it â€” operator may exceed.
	p := &LocalModelPromotion{
		CitizenWallet: citizenWallet,
		OwnerWallet:   requesterWallet,
		AttestedLevel: attestedLevel,
		Status:        "pending",
		RequestedAt:   time.Now(),
	}
	r.Promotions[citizenWallet] = p
	return p, nil
}

// TriggerBuild best-effort invokes the existing ornith compile harness for the citizen.
// On Windows + VB_MODEL_COMPILE_ENABLED, it shells setup_custom_quant_ornith.bat. The
// harness owns the artifact; we only record the emitted runner path on success. On any
// other platform or when disabled, status flips to requires_manual_build with the command.
func (r *LocalModelPromotionRegistry) TriggerBuild(citizenWallet string) (*LocalModelPromotion, error) {
	r.mu.Lock()
	p, ok := r.Promotions[citizenWallet]
	if !ok {
		r.mu.Unlock()
		return nil, fmt.Errorf("no promotion request for citizen %s", citizenWallet)
	}
	if p.Status == "building" || p.Status == "ready" {
		r.mu.Unlock()
		return p, nil
	}
	p.Status = "building"
	r.mu.Unlock()

	// Â§24.5/Â§24.6: invoke the pathway wrapper (feeds curated behavioral corpus into the owner's
	// ornith harness) instead of the raw harness. The wrapper leaves setup_custom_quant_ornith.bat
	// untouched (owner-owned boundary) and selects the citizen's pathway corpus.
	batPath := "setup_bot_pathway.bat"
	if runtime.GOOS == "windows" && os.Getenv("VB_MODEL_COMPILE_ENABLED") == "true" {
		// Â§31.1: pass identity + tier + backend to the wrapper so the harness bakes a
		// namespaced runner (start_ornith_<identity>.bat) on a derived free port / Ollama alias.
		// Optional Ollama-gateway mode (ORNITH_OLLAMA_GATEWAY=1) skips the standalone server.
		cmd := exec.Command("cmd", "/c", batPath)
		cmd.Env = append(os.Environ(),
			"ORNITH_IDENTITY="+citizenWallet,
			"ORNITH_PATHWAY="+os.Getenv("ORNITH_PATHWAY"),
			"ORNITH_MODEL_TIER="+GetOrnithTier(),
			"ORNITH_BACKEND="+GetOrnithBackend(),
			fmt.Sprintf("ORNITH_DERIVED_PORT=%d", DerivedPort(citizenWallet)),
			"ORNITH_OLLAMA_GATEWAY="+os.Getenv("ORNITH_OLLAMA_GATEWAY"),
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			r.mu.Lock()
			p.Status = "requires_manual_build"
			p.RejectReason = fmt.Sprintf("harness invocation failed: %v", err)
			r.mu.Unlock()
			return p, err
		}
		// Harness emits a namespaced runner. Record identity-namespaced RunnerPath
		// (or the Ollama alias when gateway mode is on). ADVISORY ONLY â€” no cap enforced.
		r.mu.Lock()
		if os.Getenv("ORNITH_OLLAMA_GATEWAY") == "1" {
			p.RunnerPath = OrnithOllamaAlias + citizenWallet
		} else {
			p.RunnerPath = fmt.Sprintf("start_ornith_%s.bat", citizenWallet)
		}
		p.Status = "ready"
		p.CompletedAt = time.Now()
		r.mu.Unlock()
		return p, nil
	}

	// Non-Windows or disabled: operator must run the harness manually.
	r.mu.Lock()
	p.Status = "requires_manual_build"
	p.RejectReason = fmt.Sprintf("run '%s' on Windows with VB_MODEL_COMPILE_ENABLED=true (50%% GPU/CPU/RAM cap honored by operator)", batPath)
	r.mu.Unlock()
	return p, nil
}

// Get returns a promotion by citizen wallet.
func (r *LocalModelPromotionRegistry) Get(citizenWallet string) (*LocalModelPromotion, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.Promotions[citizenWallet]
	return p, ok
}

// Save atomically persists the registry (mirrors BondedAssetRegistry.Save).
func (r *LocalModelPromotionRegistry) Save(l *Lobby) error {
	if l == nil {
		return fmt.Errorf("lobby not initialized")
	}
	r.mu.RLock()
	snapshot := &LocalModelPromotionRegistry{
		Promotions: make(map[string]*LocalModelPromotion, len(r.Promotions)),
	}
	for k, v := range r.Promotions {
		cp := *v
		snapshot.Promotions[k] = &cp
	}
	r.mu.RUnlock()

	// EXPORTED FIELD, and the RECORD IS A TRANSPORT MIRROR of the same payload the file write
	// produces: this snapshot was taken under the REGISTRY OWN lock and released above, so the
	// record never sees a live map. The field had to be EXPORTED first - an unexported field
	// silently vanishes from json, which is why the local-model file was persisted EMPTY.
	l.saveBlockchainStateSnapshotLocked(NotePrefixLocalModelSnapshot, snapshot)

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal local-model promotions: %w", err)
	}
	targetPath := l.getDataPath(localModelPromotionFile)
	tempPath := targetPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write local-model promotions temp: %w", err)
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to commit local-model promotions: %w", err)
	}
	log.Printf("[LocalModel] Persisted %d promotion requests to %s", len(snapshot.Promotions), targetPath)
	return nil
}

// Load rehydrates the registry from disk (mirrors BondedAssetRegistry.Load).
func (r *LocalModelPromotionRegistry) Load(l *Lobby) error {
	if l == nil {
		return fmt.Errorf("lobby not initialized")
	}
	targetPath := l.getDataPath(localModelPromotionFile)
	data, err := os.ReadFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("[LocalModel] No existing promotions at %s (fresh start)", targetPath)
			return nil
		}
		return fmt.Errorf("failed to read local-model promotions: %w", err)
	}
	var snapshot LocalModelPromotionRegistry
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return fmt.Errorf("failed to unmarshal local-model promotions: %w", err)
	}
	r.mu.Lock()
	if snapshot.Promotions != nil {
		r.Promotions = snapshot.Promotions
	}
	r.mu.Unlock()
	log.Printf("[LocalModel] Loaded %d promotion requests from %s", len(r.Promotions), targetPath)
	return nil
}

// --- HTTP handlers (mirror theme_engine.go / bonded_asset_registry.go idiom) ---

func (l *Lobby) handlePromoteLocalModel(w http.ResponseWriter, r *http.Request) {
	if l.localModelPromotions == nil || l.aiEngine == nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "promotion service not initialized"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if wallet == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "wallet required"})
		return
	}
	var req struct {
		CitizenWallet string `json:"citizen_wallet"`
		AttestedLevel int    `json:"attested_level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid request body"})
		return
	}
	p, err := l.localModelPromotions.RequestLocalModelPromotion(l.aiEngine, req.CitizenWallet, wallet, req.AttestedLevel)
	if err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	// Auto-trigger the build (best-effort; flips to requires_manual_build if not Windows/enabled).
	if _, err := l.localModelPromotions.TriggerBuild(req.CitizenWallet); err != nil {
		// Non-fatal: status recorded as requires_manual_build.
	}
	p, _ = l.localModelPromotions.Get(req.CitizenWallet)
	writeJSON(w, map[string]interface{}{
		"success":       true,
		"status":        p.Status,
		"runner":        p.RunnerPath,
		"note":          p.RejectReason,
		"tier":          GetOrnithTier(),
		"backend":       GetOrnithBackend(),
		"derived_port":  DerivedPort(req.CitizenWallet),
		"max_suggested": HardwareScanAdvisory(), // Â§31.1: advisory only, never enforced
	})
}

func (l *Lobby) handleLocalModelStatus(w http.ResponseWriter, r *http.Request) {
	if l.localModelPromotions == nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "promotion service not initialized"})
		return
	}
	citizen := r.URL.Query().Get("citizen")
	if citizen == "" {
		writeJSON(w, map[string]interface{}{"success": false, "error": "citizen query param required"})
		return
	}
	p, ok := l.localModelPromotions.Get(citizen)
	if !ok {
		writeJSON(w, map[string]interface{}{"success": false, "error": "no promotion request for citizen"})
		return
	}
	writeJSON(w, map[string]interface{}{
		"success": true,
		"status":  p.Status,
		"runner":  p.RunnerPath,
		"note":    p.RejectReason,
	})
}
