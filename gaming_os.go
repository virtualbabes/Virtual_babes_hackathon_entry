//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"
)

// ── Gaming OS + Compliance System ───────────────────────────────────────────
// Gaming OS = civilization-as-a-service. The infrastructure beneath games.
// Compliance = regulatory hooks, KYC/AML flags, audit trails, reporting.

// OSModule represents a Gaming OS module available for lease.
type OSModule struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Category      string    `json:"category"`
	MonthlyRate   uint64    `json:"monthly_rate"`
	Active        bool      `json:"active"`
	LeaseCount    int       `json:"lease_count"`
	CreatedAt     time.Time `json:"created_at"`
}

// OSLease represents an active OS module lease.
type OSLease struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	ModuleID    string    `json:"module_id"`
	MonthlyRate uint64    `json:"monthly_rate"`
	LeaseStart  time.Time `json:"lease_start"`
	LeaseEnd    time.Time `json:"lease_end"`
	Active      bool      `json:"active"`
	AutoRenew   bool      `json:"auto_renew"`
}

// GamingOSEngine manages OS modules and leases.
type GamingOSEngine struct {
	mu      sync.RWMutex
	modules map[string]*OSModule
	leases  map[string]*OSLease
}

var gamingOSEngine = &GamingOSEngine{
	modules: make(map[string]*OSModule),
	leases:  make(map[string]*OSLease),
}

// RegisterModule registers a new OS module.
func (gose *GamingOSEngine) RegisterModule(id, name, description, category string, monthlyRate uint64) *OSModule {
	gose.mu.Lock()
	defer gose.mu.Unlock()

	module := &OSModule{
		ID:          id,
		Name:        name,
		Description: description,
		Category:    category,
		MonthlyRate: monthlyRate,
		Active:      true,
		CreatedAt:   time.Now(),
	}
	gose.modules[id] = module
	return module
}

// GetModules returns all OS modules.
func (gose *GamingOSEngine) GetModules() []*OSModule {
	gose.mu.RLock()
	defer gose.mu.RUnlock()

	var results []*OSModule
	for _, m := range gose.modules {
		if m.Active {
			results = append(results, m)
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Category < results[j].Category
	})
	return results
}

// GetModulesByCategory returns OS modules by category.
func (gose *GamingOSEngine) GetModulesByCategory(category string) []*OSModule {
	gose.mu.RLock()
	defer gose.mu.RUnlock()

	var results []*OSModule
	for _, m := range gose.modules {
		if m.Active && m.Category == category {
			results = append(results, m)
		}
	}
	return results
}

// LeaseModule leases an OS module.
func (gose *GamingOSEngine) LeaseModule(tenantID, moduleID string, durationDays int) (*OSLease, error) {
	gose.mu.Lock()
	defer gose.mu.Unlock()

	module, ok := gose.modules[moduleID]
	if !ok || !module.Active {
		return nil, fmt.Errorf("module not found")
	}

	lease := &OSLease{
		ID:          fmt.Sprintf("os_lease_%d", time.Now().UnixNano()),
		TenantID:    tenantID,
		ModuleID:    moduleID,
		MonthlyRate: module.MonthlyRate,
		LeaseStart:  time.Now(),
		LeaseEnd:    time.Now().AddDate(0, 0, durationDays),
		Active:      true,
		AutoRenew:   true,
	}
	gose.leases[lease.ID] = lease
	module.LeaseCount++
	return lease, nil
}

// GetLeases returns all leases for a tenant.
func (gose *GamingOSEngine) GetLeases(tenantID string) []*OSLease {
	gose.mu.RLock()
	defer gose.mu.RUnlock()

	var results []*OSLease
	for _, l := range gose.leases {
		if l.TenantID == tenantID {
			results = append(results, l)
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].LeaseStart.After(results[j].LeaseStart)
	})
	return results
}

// GetSummary returns a summary of the Gaming OS.
func (gose *GamingOSEngine) GetSummary() map[string]interface{} {
	gose.mu.RLock()
	defer gose.mu.RUnlock()

	activeLeases := 0
	for _, l := range gose.leases {
		if l.Active {
			activeLeases++
		}
	}

	return map[string]interface{}{
		"total_modules":   len(gose.modules),
		"active_modules":  len(gose.modules),
		"total_leases":    len(gose.leases),
		"active_leases":   activeLeases,
	}
}

// ComplianceRecord represents a compliance audit record.
type ComplianceRecord struct {
	ID          string    `json:"id"`
	Wallet      string    `json:"wallet"`
	RecordType  string    `json:"record_type"` // kyc, aml, audit, flag
	Description string    `json:"description"`
	Severity    string    `json:"severity"` // low, medium, high, critical
	Status      string    `json:"status"` // open, resolved, escalated
	CreatedAt   time.Time `json:"created_at"`
	ResolvedAt  time.Time `json:"resolved_at"`
}

// ComplianceEngine manages compliance.
type ComplianceEngine struct {
	mu      sync.RWMutex
	records map[string]*ComplianceRecord
}

var complianceEngine = &ComplianceEngine{
	records: make(map[string]*ComplianceRecord),
}

// CreateRecord creates a compliance record.
func (ce *ComplianceEngine) CreateRecord(wallet, recordType, description, severity string) *ComplianceRecord {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	record := &ComplianceRecord{
		ID:          fmt.Sprintf("compliance_%d", time.Now().UnixNano()),
		Wallet:      wallet,
		RecordType:  recordType,
		Description: description,
		Severity:    severity,
		Status:      "open",
		CreatedAt:   time.Now(),
	}
	ce.records[record.ID] = record
	return record
}

// ResolveRecord resolves a compliance record.
func (ce *ComplianceEngine) ResolveRecord(recordID string) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	record, ok := ce.records[recordID]
	if !ok {
		return fmt.Errorf("record not found")
	}
	record.Status = "resolved"
	record.ResolvedAt = time.Now()
	return nil
}

// EscalateRecord escalates a compliance record.
func (ce *ComplianceEngine) EscalateRecord(recordID string) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	record, ok := ce.records[recordID]
	if !ok {
		return fmt.Errorf("record not found")
	}
	record.Status = "escalated"
	return nil
}

// GetRecords returns all compliance records.
func (ce *ComplianceEngine) GetRecords(status string) []*ComplianceRecord {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	var results []*ComplianceRecord
	for _, r := range ce.records {
		if status == "" || r.Status == status {
			results = append(results, r)
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})
	return results
}

// GetRecordsByWallet returns compliance records for a wallet.
func (ce *ComplianceEngine) GetRecordsByWallet(wallet string) []*ComplianceRecord {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	var results []*ComplianceRecord
	for _, r := range ce.records {
		if r.Wallet == wallet {
			results = append(results, r)
		}
	}
	return results
}

// GetSummary returns a summary of compliance.
func (ce *ComplianceEngine) GetSummary() map[string]interface{} {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	open := 0
	resolved := 0
	escalated := 0
	for _, r := range ce.records {
		switch r.Status {
		case "open":
			open++
		case "resolved":
			resolved++
		case "escalated":
			escalated++
		}
	}

	return map[string]interface{}{
		"total_records":    len(ce.records),
		"open":             open,
		"resolved":         resolved,
		"escalated":        escalated,
	}
}

// ── HTTP Handlers ────────────────────────────────────────────────────────────

// Gaming OS handlers
func (l *Lobby) handleOSModules(w http.ResponseWriter, r *http.Request) {
	modules := gamingOSEngine.GetModules()
	writeJSON(w, map[string]interface{}{"success": true, "modules": modules})
}

func (l *Lobby) handleOSModuleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Category    string `json:"category"`
		MonthlyRate uint64 `json:"monthly_rate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	module := gamingOSEngine.RegisterModule(req.ID, req.Name, req.Description, req.Category, req.MonthlyRate)
	writeJSON(w, map[string]interface{}{"success": true, "module": module})
}

func (l *Lobby) handleOSLease(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ModuleID    string `json:"module_id"`
		Duration    int    `json:"duration_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	lease, err := gamingOSEngine.LeaseModule(wallet, req.ModuleID, req.Duration)
	if err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true, "lease": lease})
}

func (l *Lobby) handleOSLeases(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	leases := gamingOSEngine.GetLeases(wallet)
	writeJSON(w, map[string]interface{}{"success": true, "leases": leases})
}

func (l *Lobby) handleOSSummary(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{"success": true, "summary": gamingOSEngine.GetSummary()})
}

// Compliance handlers
func (l *Lobby) handleComplianceRecord(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Wallet      string `json:"wallet"`
		RecordType  string `json:"record_type"`
		Description string `json:"description"`
		Severity    string `json:"severity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	record := complianceEngine.CreateRecord(req.Wallet, req.RecordType, req.Description, req.Severity)
	writeJSON(w, map[string]interface{}{"success": true, "record": record})
}

func (l *Lobby) handleComplianceResolve(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RecordID string `json:"record_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	if err := complianceEngine.ResolveRecord(req.RecordID); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true})
}

func (l *Lobby) handleComplianceEscalate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RecordID string `json:"record_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	if err := complianceEngine.EscalateRecord(req.RecordID); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true})
}

func (l *Lobby) handleComplianceRecords(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	records := complianceEngine.GetRecords(status)
	writeJSON(w, map[string]interface{}{"success": true, "records": records})
}

func (l *Lobby) handleComplianceByWallet(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	records := complianceEngine.GetRecordsByWallet(wallet)
	writeJSON(w, map[string]interface{}{"success": true, "records": records})
}

func (l *Lobby) handleComplianceSummary(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{"success": true, "summary": complianceEngine.GetSummary()})
}
