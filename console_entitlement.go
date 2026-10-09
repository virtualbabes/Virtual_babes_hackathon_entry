//go:build !js && !wasm

package main

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// ════════════════════════════════════════════════════════════════════════════
// THE CONSOLE ENTITLEMENT PATH — workstream C of every platform plan.
//
// `ConsoleAssetReceipt` has been DECLARED since the Phase 4 expansion and WRITTEN BY NOTHING: no handler
// and no reader existed, so the platform→server direction was a contract, not a rail. This file is the
// ONE owner of that direction:
//
//	the platform sells a DLC product
//	  -> its fulfilment call carries a ConsoleAssetReceipt plus the platform purchase id
//	  -> we ask the PLATFORM'S OWN API for its record of that purchase
//	  -> we grant the ENTITLEMENT, idempotently, keyed by the platform purchase id
//
// THREE RULES THIS FILE EXISTS TO ENFORCE:
//
//  1. A RECEIPT IS NEVER TRUSTED BECAUSE A CLIENT SENT IT. The product, the console account, the lease
//     shape and the duration all come from the PLATFORM's record and are CROSS-CHECKED against the
//     claim. A caller may name a platform purchase id and nothing else.
//  2. THE PLATFORM COLLECTED THE MONEY, SO THIS PATH MOVES NONE. No balance, no voucher, no faucet and
//     no ledger counter is touched here: the grant is OWNERSHIP, and the settlement with the platform is
//     the operator's rail (console plan §10.4 — a platform sale is an UNMATURED RECEIVABLE). A test
//     pins this so a later edit cannot quietly start minting.
//  3. IDEMPOTENT BY THE ONE MEMO DOOR. The platform purchase id is the idempotency key: claimed BEFORE
//     the (slow) platform call and committed only AFTER the grant, so a replayed fulfilment cannot grant
//     twice and a failed attempt stays retryable.
//
// BLOCKERS ARE STATED, NEVER HIDDEN. With no platform verifier configured — the default, because this
// repository holds no platform credentials — the door REFUSES with 409 and NAMES the environment
// variables it needs. That is the shape the lease catalogue already uses: a surface must never look
// like a working one.
// ════════════════════════════════════════════════════════════════════════════

const consoleEntitlementFile = "console_entitlements.json"

// The INBOUND proof: env-only, never persisted and never served (secrets are ENV-ONLY).
const (
	consoleFulfilmentSecretEnv = "CONSOLE_FULFILMENT_SECRET"
	consoleFulfilmentSecretHdr = "X-Console-Fulfilment-Secret"
)

// ConsoleEntitlement is ONE granted entitlement: what a verified platform purchase bought.
type ConsoleEntitlement struct {
	ReceiptID           string          `json:"receipt_id"`
	Platform            ConsolePlatform `json:"platform"`
	ConsoleUID          string          `json:"console_uid"`
	Wallet              string          `json:"wallet"`
	ProductCode         string          `json:"product_code"`
	IsLease             bool            `json:"is_lease"`
	LeaseDurationBlocks int64           `json:"lease_duration_blocks,omitempty"`
	PurchasedAt         time.Time       `json:"purchased_at"`
	GrantedAt           time.Time       `json:"granted_at"`
	VerifiedVia         string          `json:"verified_via"`
}

// ConsoleEntitlementRegistry holds the grants, keyed by the platform purchase id — which IS the
// idempotency key, so "granted once" is a property of the map rather than of a caller's discipline.
type ConsoleEntitlementRegistry struct {
	mu        sync.RWMutex
	byReceipt map[string]*ConsoleEntitlement
}

func NewConsoleEntitlementRegistry() *ConsoleEntitlementRegistry {
	return &ConsoleEntitlementRegistry{byReceipt: make(map[string]*ConsoleEntitlement)}
}

// record returns a COPY of the grant for a purchase id, or nil.
func (r *ConsoleEntitlementRegistry) record(receiptID string) *ConsoleEntitlement {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.byReceipt[strings.TrimSpace(receiptID)]
	if !ok || e == nil {
		return nil
	}
	cp := *e
	return &cp
}

// ForWallet returns a wallet's grants in a DETERMINISTIC order (receipt id ascending), because a served
// list that reshuffles between reads looks like a change to the player.
func (r *ConsoleEntitlementRegistry) ForWallet(wallet string) []*ConsoleEntitlement {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*ConsoleEntitlement, 0, 4)
	for _, e := range r.byReceipt {
		if e != nil && strings.EqualFold(e.Wallet, wallet) {
			cp := *e
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ReceiptID < out[j].ReceiptID })
	return out
}

// Count reports how many grants are held.
func (r *ConsoleEntitlementRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byReceipt)
}

// Snapshot returns private COPIES, so a caller can marshal them without holding the registry lock.
func (r *ConsoleEntitlementRegistry) Snapshot() []*ConsoleEntitlement {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*ConsoleEntitlement, 0, len(r.byReceipt))
	for _, e := range r.byReceipt {
		if e == nil {
			continue
		}
		cp := *e
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ReceiptID < out[j].ReceiptID })
	return out
}

// grant inserts a grant. It refuses a grant whose purchase id is already held, so this registry cannot
// be talked into a second grant for one purchase even if a caller's memo discipline were wrong.
func (r *ConsoleEntitlementRegistry) grant(e *ConsoleEntitlement) (*ConsoleEntitlement, bool, error) {
	if e == nil || strings.TrimSpace(e.ReceiptID) == "" {
		return nil, false, fmt.Errorf("an entitlement needs the platform purchase id it was granted for")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.byReceipt[e.ReceiptID]; ok && existing != nil {
		cp := *existing
		return &cp, false, nil
	}
	stored := *e
	r.byReceipt[stored.ReceiptID] = &stored
	cp := stored
	return &cp, true, nil
}

// Save writes the registry to its file and dispatches the SAME payload as a transport mirror (the
// snapshot above is private COPIES, so the record never sees a live map). Its family is declared in
// record_families.go, which is what makes the mirror legal — the writer refuses an undeclared prefix.
func (r *ConsoleEntitlementRegistry) Save(l *Lobby) error {
	if l == nil {
		return fmt.Errorf("lobby not initialized")
	}
	snapshot := r.Snapshot()
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal console entitlements: %w", err)
	}
	if err := os.WriteFile(l.getDataPath(consoleEntitlementFile), data, 0644); err != nil {
		return err
	}
	l.saveBlockchainStateSnapshotLocked(NotePrefixConsoleEntitlementSnapshot, snapshot)
	return nil
}

// Load rehydrates the registry from disk.
func (r *ConsoleEntitlementRegistry) Load(l *Lobby) error {
	if l == nil {
		return fmt.Errorf("lobby not initialized")
	}
	data, err := os.ReadFile(l.getDataPath(consoleEntitlementFile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var grants []*ConsoleEntitlement
	if err := json.Unmarshal(data, &grants); err != nil {
		return fmt.Errorf("failed to unmarshal console entitlements: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byReceipt = make(map[string]*ConsoleEntitlement, len(grants))
	for _, e := range grants {
		if e == nil || strings.TrimSpace(e.ReceiptID) == "" {
			continue
		}
		r.byReceipt[e.ReceiptID] = e
	}
	return nil
}

// ── THE PLATFORM'S PROOF ────────────────────────────────────────────────────

// consolePurchaseRecord is the PLATFORM'S OWN record of a purchase. It is the only source of the
// product, the console account, the lease shape and the duration: the receipt a caller sends is a
// CLAIM, and this record is what the claim is measured against.
type consolePurchaseRecord struct {
	Verified            bool            `json:"verified"`
	PurchaseID          string          `json:"purchase_id"`
	Platform            ConsolePlatform `json:"platform"`
	ConsoleUID          string          `json:"console_uid"`
	ProductCode         string          `json:"product_code"`
	PurchasedAtUnix     int64           `json:"purchased_at_unix"`
	IsLease             bool            `json:"is_lease"`
	LeaseDurationBlocks int64           `json:"lease_duration_blocks"`
}

// ConsolePurchaseVerifier asks a platform for its record of one purchase. `name` is recorded on the
// grant (`verified_via`), so an entitlement always says WHO proved it.
type ConsolePurchaseVerifier func(platform ConsolePlatform, purchaseID string, receipt ConsoleAssetReceipt) (*consolePurchaseRecord, string, error)

var (
	consoleVerifiersMu sync.RWMutex
	consoleVerifiers   = map[ConsolePlatform]ConsolePurchaseVerifier{}
)

// registerConsolePurchaseVerifier installs the verifier for a platform. The DEFAULT table is EMPTY:
// this repository holds no platform credentials, and that is exactly why the door REFUSES rather than
// trusting a receipt a caller sent.
func registerConsolePurchaseVerifier(platform ConsolePlatform, fn ConsolePurchaseVerifier) {
	if fn == nil {
		return
	}
	consoleVerifiersMu.Lock()
	defer consoleVerifiersMu.Unlock()
	consoleVerifiers[platform] = fn
}

func consoleVerifierFor(platform ConsolePlatform) (ConsolePurchaseVerifier, bool) {
	consoleVerifiersMu.RLock()
	defer consoleVerifiersMu.RUnlock()
	fn, ok := consoleVerifiers[platform]
	return fn, ok
}

// consoleVerifierEnvNames names the environment variables a platform's verifier needs. The names are
// SERVED (in the contract), so an operator is never left guessing what to set.
func consoleVerifierEnvNames(platform ConsolePlatform) (urlVar, secretVar string) {
	suffix := string(platform)
	return "CONSOLE_VERIFIER_URL_" + suffix, "CONSOLE_VERIFIER_SECRET_" + suffix
}

// consoleVerifierNames lists every platform this path knows, in a stable order.
func consoleVerifierNames() []ConsolePlatform {
	out := []ConsolePlatform{PlatformXbox, PlatformPlayStation, PlatformNintendo}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// newHTTPConsolePurchaseVerifier builds the ONE HTTP verifier shape: POST the purchase id (with the
// claim, so the platform can answer a mismatch) to the platform's verification endpoint with its secret
// in a header, then decode the platform's record. A non-200 answer is an ERROR, never a soft pass.
func newHTTPConsolePurchaseVerifier(url, secret string, client *http.Client) ConsolePurchaseVerifier {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	endpoint := strings.TrimSpace(url)
	key := strings.TrimSpace(secret)
	return func(platform ConsolePlatform, purchaseID string, receipt ConsoleAssetReceipt) (*consolePurchaseRecord, string, error) {
		body, err := json.Marshal(map[string]string{
			"purchase_id":      purchaseID,
			"platform":         string(platform),
			"console_uid":      receipt.ConsoleUID,
			"dlc_product_code": receipt.DlcProductCode,
		})
		if err != nil {
			return nil, "", fmt.Errorf("could not encode the verification request: %w", err)
		}
		req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, "", fmt.Errorf("could not build the verification request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if key != "" {
			req.Header.Set(consoleFulfilmentSecretHdr, key)
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, "", fmt.Errorf("the platform verifier could not be reached: %w", err)
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if resp.StatusCode != http.StatusOK {
			return nil, "", fmt.Errorf("the platform verifier answered HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
		}
		var rec consolePurchaseRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			return nil, "", fmt.Errorf("the platform verifier answered a payload that is not a purchase record: %w", err)
		}
		return &rec, "platform_api:" + string(platform), nil
	}
}

// syncConsoleVerifiersFromEnv registers an HTTP verifier for every platform whose URL AND secret are
// present in the environment, returning how many it registered so boot can STATE the number rather than
// leaving the operator to infer it.
func syncConsoleVerifiersFromEnv() int {
	n := 0
	for _, p := range consoleVerifierNames() {
		urlVar, secretVar := consoleVerifierEnvNames(p)
		url := strings.TrimSpace(os.Getenv(urlVar))
		secret := strings.TrimSpace(os.Getenv(secretVar))
		if url == "" || secret == "" {
			continue
		}
		registerConsolePurchaseVerifier(p, newHTTPConsolePurchaseVerifier(url, secret, nil))
		n++
	}
	return n
}

// validateConsolePurchaseRecord is the ONE cross-check between the platform's record and the claim.
func validateConsolePurchaseRecord(rec *consolePurchaseRecord, receipt ConsoleAssetReceipt, purchaseID string) error {
	if rec == nil {
		return fmt.Errorf("the platform returned no purchase record")
	}
	if !rec.Verified {
		return fmt.Errorf("the platform's own record does not mark this purchase as verified")
	}
	if strings.TrimSpace(purchaseID) == "" {
		return fmt.Errorf("a platform purchase id is required")
	}
	if !strings.EqualFold(strings.TrimSpace(rec.PurchaseID), strings.TrimSpace(purchaseID)) {
		return fmt.Errorf("the platform's record is for purchase %q, not %q", rec.PurchaseID, purchaseID)
	}
	if rec.Platform != receipt.PlatformRef {
		return fmt.Errorf("the platform's record is for %s, not %s", rec.Platform, receipt.PlatformRef)
	}
	if !strings.EqualFold(strings.TrimSpace(rec.ConsoleUID), strings.TrimSpace(receipt.ConsoleUID)) {
		return fmt.Errorf("the platform's record is for console account %q, not %q", rec.ConsoleUID, receipt.ConsoleUID)
	}
	if !strings.EqualFold(strings.TrimSpace(rec.ProductCode), strings.TrimSpace(receipt.DlcProductCode)) {
		return fmt.Errorf("the platform's record covers product %q, not the claimed %q", rec.ProductCode, receipt.DlcProductCode)
	}
	if rec.IsLease != receipt.IsLeaseAction {
		return fmt.Errorf("the platform's record is %s, but the claim is %s", leaseWords(rec.IsLease), leaseWords(receipt.IsLeaseAction))
	}
	if rec.IsLease && rec.LeaseDurationBlocks <= 0 {
		return fmt.Errorf("the platform's record is a lease with no duration, so the lease term cannot be stated")
	}
	return nil
}

func leaseWords(isLease bool) string {
	if isLease {
		return "a lease"
	}
	return "a purchase"
}

// ── THE LINKED WALLET (one resolver for BOTH console rails) ─────────────────

// consoleLinkedWalletLocked resolves the AVM wallet a console account is VERIFIED to own. It is the ONE
// owner of that rule: the voucher redemption gateway used to open-code this loop, so the two console
// rails could have drifted about what "linked" means. Caller MUST hold l.mutex.
func (l *Lobby) consoleLinkedWalletLocked(consoleUID, platform string) string {
	uid := strings.TrimSpace(consoleUID)
	chain := strings.TrimSpace(platform)
	if uid == "" || chain == "" {
		return ""
	}
	keys := make([]string, 0, len(l.linkedWallets))
	for wallet := range l.linkedWallets {
		keys = append(keys, wallet)
	}
	// DETERMINISTIC: with two matching links, map order would resolve differently per read.
	sort.Strings(keys)
	for _, wallet := range keys {
		for _, linked := range l.linkedWallets[wallet].Linked {
			if strings.EqualFold(linked.Address, uid) && strings.EqualFold(linked.Chain, chain) && linked.Verified {
				return wallet
			}
		}
	}
	return ""
}

// consoleLinkedWallet is the self-locking form, for a caller that holds nothing.
func (l *Lobby) consoleLinkedWallet(consoleUID, platform string) string {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.consoleLinkedWalletLocked(consoleUID, platform)
}

// ── THE SERVED CONTRACT ─────────────────────────────────────────────────────

// consoleEntitlementContract is the SERVED shape: what this path can do RIGHT NOW, what it is blocked
// on, and the rules it obeys. It is served even when the door refuses, so a refusal always carries its
// reason rather than a bare status code.
func consoleEntitlementContract() map[string]interface{} {
	secretSet := strings.TrimSpace(os.Getenv(consoleFulfilmentSecretEnv)) != ""
	platforms := make([]map[string]interface{}, 0, 3)
	blocked := make([]string, 0, 3)
	configured := 0
	for _, p := range consoleVerifierNames() {
		urlVar, secretVar := consoleVerifierEnvNames(p)
		_, ok := consoleVerifierFor(p)
		if ok {
			configured++
		} else {
			blocked = append(blocked, fmt.Sprintf("no purchase verifier is configured for %s: set %s and %s to the platform's own purchase-verification endpoint", p, urlVar, secretVar))
		}
		platforms = append(platforms, map[string]interface{}{
			"platform":            string(p),
			"verifier_configured": ok,
			"verifier_url_env":    urlVar,
			"verifier_secret_env": secretVar,
		})
	}
	if !secretSet {
		blocked = append(blocked, fmt.Sprintf("no inbound fulfilment secret is set: %s is required, because the platform's fulfilment call is the only caller this door accepts", consoleFulfilmentSecretEnv))
	}
	// STRUCTURAL blockers: true whatever the configuration, so they cannot be read as misconfiguration.
	blocked = append(blocked,
		"lease EXPIRY is recorded from the platform's own duration but ENFORCED nowhere: no block clock is wired, so a lapsed lease is not withdrawn",
		"the creator payout for a platform sale is the operator's settlement rail: this path grants the entitlement and credits NOBODY, because the platform has not settled yet",
	)
	return map[string]interface{}{
		"available":                    secretSet && configured > 0,
		"fulfilment_secret_env":        consoleFulfilmentSecretEnv,
		"fulfilment_secret_configured": secretSet,
		"verifiers_configured":         configured,
		"platforms":                    platforms,
		"blocked_by":                   blocked,
		"verification_rule":            "a receipt is proved by the PLATFORM's own record: the product, the console account, the lease shape and the duration are cross-checked against it, so a caller may name a platform purchase id and nothing else",
		"money_rule":                   "the platform collected the money, so this path moves NO balance, voucher, faucet credit or ledger counter: a platform sale is an unmatured receivable until the platform settles (console plan §10.4)",
		"idempotency_rule":             "the platform purchase id is the idempotency key, claimed through the ONE memo door before the platform call and committed only after the grant, so a replayed fulfilment grants once and a failed attempt stays retryable",
		"lease_rule":                   "the lease duration comes from the platform's record; the duration in the claim is IGNORED",
		"authoritative_surface":        "the grant must happen on the server that owns the record rail: a console-local grant is a virtual-mirror record until the bridge carries it, because the console build holds 0 chain rails by ruling",
	}
}

// ── THE DOOR ────────────────────────────────────────────────────────────────

// consoleEntitlementRequest is the inbound fulfilment payload. It carries the platform's receipt, the
// purchase id, and NOTHING ELSE: no price, no wallet, no product name, no duration — the decoder's
// DisallowUnknownFields makes that structural rather than a convention.
type consoleEntitlementRequest struct {
	Receipt            ConsoleAssetReceipt `json:"receipt"`
	PlatformPurchaseID string              `json:"platform_purchase_id"`
}

// GrantConsoleEntitlement is the ONE writer for a platform purchase. It holds NO lobby lock across the
// platform call (txid_memo.go's contract), grants exactly once per purchase id, and moves no money.
func (l *Lobby) GrantConsoleEntitlement(req consoleEntitlementRequest) (*ConsoleEntitlement, bool, error) {
	if l == nil {
		return nil, false, fmt.Errorf("lobby not initialized")
	}
	if l.consoleEntitlements == nil {
		return nil, false, fmt.Errorf("the console entitlement registry is not constructed on this server")
	}
	receipt := req.Receipt
	purchaseID := strings.TrimSpace(req.PlatformPurchaseID)
	if strings.TrimSpace(string(receipt.PlatformRef)) == "" {
		return nil, false, fmt.Errorf("a console receipt must name the platform it came from")
	}
	if strings.TrimSpace(receipt.ConsoleUID) == "" {
		return nil, false, fmt.Errorf("a console receipt must name the console account that bought it")
	}
	if strings.TrimSpace(receipt.DlcProductCode) == "" {
		return nil, false, fmt.Errorf("a console receipt must name the DLC product that was bought")
	}
	if purchaseID == "" {
		return nil, false, fmt.Errorf("a platform purchase id is required: it is both the idempotency key and the only proof of WHICH purchase this is")
	}

	verifier, ok := consoleVerifierFor(receipt.PlatformRef)
	if !ok {
		urlVar, secretVar := consoleVerifierEnvNames(receipt.PlatformRef)
		return nil, false, fmt.Errorf("no purchase verifier is configured for %s, so this receipt cannot be proved: set %s and %s", receipt.PlatformRef, urlVar, secretVar)
	}

	// A purchase already granted is answered as an IDEMPOTENT no-op, not an error: the platform may
	// retry a fulfilment it could not confirm, and a retry must never grant twice.
	if existing := l.consoleEntitlements.record(purchaseID); existing != nil {
		return existing, false, nil
	}

	claimed, reason := l.claimTxID(purchaseID)
	if claimed == "" {
		if reason == TxIDAlreadyUtilized {
			if existing := l.consoleEntitlements.record(purchaseID); existing != nil {
				return existing, false, nil
			}
		}
		return nil, false, fmt.Errorf("%s", reason)
	}

	// THE SLOW CALL, with NO lobby lock held.
	rec, verifierName, err := verifier(receipt.PlatformRef, purchaseID, receipt)
	if err != nil {
		l.releaseTxID(purchaseID) // nothing was written, so a genuine retry stays possible
		return nil, false, err
	}
	if err := validateConsolePurchaseRecord(rec, receipt, purchaseID); err != nil {
		l.releaseTxID(purchaseID)
		return nil, false, err
	}

	wallet := l.consoleLinkedWallet(rec.ConsoleUID, string(rec.Platform))
	if wallet == "" {
		l.releaseTxID(purchaseID)
		return nil, false, fmt.Errorf("the console account %s is not linked to a VERIFIED wallet on %s, so there is no account to grant this entitlement to", rec.ConsoleUID, rec.Platform)
	}

	purchasedAt := time.Now().UTC()
	if rec.PurchasedAtUnix > 0 {
		purchasedAt = time.Unix(rec.PurchasedAtUnix, 0).UTC()
	}
	granted, isNew, err := l.consoleEntitlements.grant(&ConsoleEntitlement{
		ReceiptID:           purchaseID,
		Platform:            rec.Platform,
		ConsoleUID:          rec.ConsoleUID,
		Wallet:              wallet,
		ProductCode:         rec.ProductCode,
		IsLease:             rec.IsLease,
		LeaseDurationBlocks: rec.LeaseDurationBlocks,
		PurchasedAt:         purchasedAt,
		GrantedAt:           time.Now().UTC(),
		VerifiedVia:         verifierName,
	})
	if err != nil {
		l.releaseTxID(purchaseID)
		return nil, false, err
	}
	if isNew {
		l.commitTxID(purchaseID, purchasedAt) // committed only AFTER the grant
		if err := l.consoleEntitlements.Save(l); err != nil {
			log.Printf("[CONSOLE_ENTITLEMENT] granted %s to %s but persisting the registry failed: %v", purchaseID, wallet, err)
		}
		l.logAdminAudit("CONSOLE_ENTITLEMENT_GRANTED", wallet, fmt.Sprintf("%s %s on %s (%s), proved by %s", leaseWords(granted.IsLease), granted.ProductCode, granted.Platform, purchaseID, granted.VerifiedVia))
		if l.broadcast != nil { // a NIL channel would block this goroutine for ever
			go func() { l.broadcast <- l.getLobbyUpdateMsg() }()
		}
	}
	return granted, isNew, nil
}

// ── THE HTTP BOUNDARY ───────────────────────────────────────────────────────

// handleConsoleEntitlementGrant is the PLATFORM-FACING fulfilment door.
func (l *Lobby) handleConsoleEntitlementGrant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONStatus(w, http.StatusMethodNotAllowed, map[string]interface{}{"success": false, "error": "POST required"})
		return
	}
	if !consoleFulfilmentAuthorized(r) {
		writeJSONStatus(w, http.StatusUnauthorized, map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("unauthorized: send %s in the %s header. The secret is env-owned and never served, and an unset secret authorizes NOBODY", consoleFulfilmentSecretEnv, consoleFulfilmentSecretHdr),
		})
		return
	}
	var req consoleEntitlementRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "invalid console entitlement body: a receipt and a platform purchase id are accepted, and nothing else: " + err.Error(),
		})
		return
	}
	ent, granted, err := l.GrantConsoleEntitlement(req)
	if err != nil {
		writeJSONStatus(w, http.StatusConflict, map[string]interface{}{
			"success":  false,
			"granted":  false,
			"error":    err.Error(),
			"contract": consoleEntitlementContract(),
		})
		return
	}
	writeJSONStatus(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"granted":     granted,
		"idempotent":  !granted,
		"entitlement": ent,
	})
}

// consoleFulfilmentAuthorized compares the inbound secret in CONSTANT TIME and fails CLOSED: with no
// secret configured, nobody is authorized — there is deliberately no "unset means open" path.
func consoleFulfilmentAuthorized(r *http.Request) bool {
	secret := strings.TrimSpace(os.Getenv(consoleFulfilmentSecretEnv))
	if secret == "" {
		return false
	}
	got := strings.TrimSpace(r.Header.Get(consoleFulfilmentSecretHdr))
	if got == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(secret)) == 1
}

// handleConsoleEntitlements is the READ-ONLY view of a wallet's grants.
func (l *Lobby) handleConsoleEntitlements(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONStatus(w, http.StatusMethodNotAllowed, map[string]interface{}{"success": false, "error": "GET required"})
		return
	}
	if l.consoleEntitlements == nil {
		writeJSONStatus(w, http.StatusServiceUnavailable, map[string]interface{}{
			"success": false,
			"error":   "the console entitlement registry is not constructed on this server",
		})
		return
	}
	wallet := strings.TrimSpace(r.URL.Query().Get("wallet"))
	if wallet == "" {
		wallet = l.getWalletFromRequest(r)
	}
	if wallet == "" {
		writeJSONStatus(w, http.StatusBadRequest, map[string]interface{}{
			"success":  false,
			"error":    "wallet required: send ?wallet= or the wallet header",
			"contract": consoleEntitlementContract(),
		})
		return
	}
	grants := l.consoleEntitlements.ForWallet(wallet)
	if grants == nil {
		grants = []*ConsoleEntitlement{} // never null: an absent LIST must not read as an absent feature
	}
	writeJSONStatus(w, http.StatusOK, map[string]interface{}{
		"success":      true,
		"wallet":       wallet,
		"entitlements": grants,
		"count":        len(grants),
		"contract":     consoleEntitlementContract(),
	})
}
