//go:build !js && !wasm

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// THE CONSOLE ENTITLEMENT PATH — the tests that make its three claims checkable.
//
//	* a receipt is never trusted because a client sent it (§4: every mismatch is refused),
//	* the path moves NO money (§6: balances and vouchers are byte-identical after a grant),
//	* it is idempotent by the ONE memo door (§3: a replay grants once and calls the platform once).
//
// The verifier table is a PACKAGE GLOBAL, so every test clears it on cleanup — otherwise one test's
// stub platform could answer another test's purchase.

const (
	ceTestWallet     = "0xcewallet000000000000000000000000000000000000000000000000000001"
	ceTestConsoleUID = "2533274899999999"
	ceTestProduct    = "DLC-ARENA-PASS"
	ceTestPurchase   = "xbox-purchase-0001"
	ceTestSecret     = "inbound-fulfilment-secret"
	ceTestVerifier   = "outbound-verifier-secret"
)

func ceClearVerifiers() {
	consoleVerifiersMu.Lock()
	defer consoleVerifiersMu.Unlock()
	consoleVerifiers = map[ConsolePlatform]ConsolePurchaseVerifier{}
}

// ceTestLobby builds the MINIMUM Lobby the entitlement path needs, plus a VERIFIED console link and a
// funded, card-holding player — so a grant that quietly touched money would be visible.
func ceTestLobby(t *testing.T, wallet, consoleUID string) *Lobby {
	t.Helper()
	l := &Lobby{
		DataDir:             t.TempDir(),
		consoleEntitlements: NewConsoleEntitlementRegistry(),
		clients:             map[string]*Client{},
		leaderboard:         map[string]PlayerStats{},
		playerBalances:      map[string]uint64{},
		linkedWallets:       map[string]WalletLinkInfo{},
		registeredTxIDs:     map[string]time.Time{},
		pendingTxIDs:        map[string]time.Time{},
	}
	l.playerBalances[wallet] = 42_000_000
	l.leaderboard[wallet] = PlayerStats{ArenaVouchers: 7_000_000, Inventory: map[string]int{"CARD-1": 3}}
	l.linkedWallets[wallet] = WalletLinkInfo{
		PrimaryAVMWallet: wallet,
		Linked:           []LinkedWallet{{Address: consoleUID, Chain: string(PlatformXbox), Verified: true}},
	}
	return l
}

// ceGoodRecord is the platform's own record for the happy path.
func ceGoodRecord() consolePurchaseRecord {
	return consolePurchaseRecord{
		Verified:        true,
		PurchaseID:      ceTestPurchase,
		Platform:        PlatformXbox,
		ConsoleUID:      ceTestConsoleUID,
		ProductCode:     ceTestProduct,
		PurchasedAtUnix: time.Now().Add(-time.Hour).Unix(),
	}
}

// ceInstallVerifier registers an HTTP verifier that answers `rec` (or `status`), counting its calls.
func ceInstallVerifier(t *testing.T, rec consolePurchaseRecord, status int) *int32 {
	t.Helper()
	if status == 0 {
		status = http.StatusOK // a stub that names no status ANSWERS 200: WriteHeader(0) panics, which would look like an unreachable platform
	}
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if r.Header.Get(consoleFulfilmentSecretHdr) != ceTestVerifier {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if status != http.StatusOK {
			w.WriteHeader(status)
			_, _ = w.Write([]byte("the platform refused"))
			return
		}
		_ = json.NewEncoder(w).Encode(rec)
	}))
	registerConsolePurchaseVerifier(PlatformXbox, newHTTPConsolePurchaseVerifier(srv.URL, ceTestVerifier, srv.Client()))
	t.Cleanup(func() {
		srv.Close()
		ceClearVerifiers()
	})
	return &calls
}

func ceRequest() consoleEntitlementRequest {
	return consoleEntitlementRequest{
		Receipt: ConsoleAssetReceipt{
			PlatformRef:    PlatformXbox,
			ConsoleUID:     ceTestConsoleUID,
			DlcProductCode: ceTestProduct,
		},
		PlatformPurchaseID: ceTestPurchase,
	}
}

// ── §1 A RECEIPT IS NEVER TRUSTED MERELY BECAUSE IT ARRIVED ─────────────────

func TestConsoleEntitlementRefusesWithNoVerifierConfigured(t *testing.T) {
	ceClearVerifiers()
	t.Cleanup(ceClearVerifiers)
	l := ceTestLobby(t, ceTestWallet, ceTestConsoleUID)

	ent, granted, err := l.GrantConsoleEntitlement(ceRequest())
	if err == nil {
		t.Fatalf("granted %v without any platform verifier: a receipt must never be trusted because it arrived", ent)
	}
	if granted || ent != nil {
		t.Errorf("granted=%v ent=%v; want a refusal that writes nothing", granted, ent)
	}
	if !strings.Contains(err.Error(), "CONSOLE_VERIFIER_URL_XBOX_SERIES_X") {
		t.Errorf("refusal %q does not NAME the environment variable it needs", err.Error())
	}
	if n := l.consoleEntitlements.Count(); n != 0 {
		t.Errorf("registry holds %d grants after a refusal; want 0", n)
	}
	if s := l.txidMemoStats(); s.Committed != 0 || s.InFlight != 0 {
		t.Errorf("memo = %+v; a refusal before verification must not leave a reservation", s)
	}

	c := consoleEntitlementContract()
	if c["available"] != false {
		t.Errorf("contract available=%v with no verifier; want false", c["available"])
	}
	blocked, _ := c["blocked_by"].([]string)
	if len(blocked) == 0 {
		t.Fatal("contract names no blocker; a refusal must always carry its reason")
	}
	if !ceAnyContains(blocked, "no purchase verifier is configured for XBOX_SERIES_X") {
		t.Errorf("blockers %v do not name the missing verifier", blocked)
	}
	if !ceAnyContains(blocked, "lease EXPIRY") {
		t.Errorf("blockers %v omit the structural lease-expiry blocker", blocked)
	}
}

// ── §2 THE GRANT ITSELF ─────────────────────────────────────────────────────

func TestConsoleEntitlementGrantsFromAVerifiedPlatformRecord(t *testing.T) {
	calls := ceInstallVerifier(t, ceGoodRecord(), http.StatusOK)
	l := ceTestLobby(t, ceTestWallet, ceTestConsoleUID)

	ent, granted, err := l.GrantConsoleEntitlement(ceRequest())
	if err != nil {
		t.Fatalf("a verified purchase was refused: %v", err)
	}
	if !granted || ent == nil {
		t.Fatalf("granted=%v ent=%v; want a new grant", granted, ent)
	}
	if atomic.LoadInt32(calls) != 1 {
		t.Errorf("platform verifier called %d times; want exactly 1", atomic.LoadInt32(calls))
	}
	if ent.Wallet != ceTestWallet {
		t.Errorf("grant lands on %q; want the VERIFIED linked wallet %q", ent.Wallet, ceTestWallet)
	}
	if ent.ProductCode != ceTestProduct || ent.Platform != PlatformXbox || ent.ConsoleUID != ceTestConsoleUID {
		t.Errorf("grant = %+v; want the platform's OWN product, platform and account", ent)
	}
	if ent.VerifiedVia == "" {
		t.Error("grant does not record WHO proved it (verified_via is empty)")
	}
	if ent.IsLease || ent.LeaseDurationBlocks != 0 {
		t.Errorf("a purchase grant reports lease fields: %+v", ent)
	}
	if n := l.consoleEntitlements.Count(); n != 1 {
		t.Errorf("registry holds %d grants; want 1", n)
	}
	if s := l.txidMemoStats(); s.Committed != 1 || s.InFlight != 0 {
		t.Errorf("memo = %+v; want exactly one committed purchase id and no reservation", s)
	}

	// The file must be readable by a FRESH registry — a grant that does not survive a restart is not a
	// grant, it is a message.
	reloaded := NewConsoleEntitlementRegistry()
	if err := reloaded.Load(l); err != nil {
		t.Fatalf("the persisted registry could not be read back: %v", err)
	}
	back := reloaded.record(ceTestPurchase)
	if back == nil || back.Wallet != ceTestWallet || back.ProductCode != ceTestProduct {
		t.Fatalf("reloaded grant = %+v; want the granted entitlement", back)
	}

	// The read path serves it, deterministically and non-null.
	grants := l.consoleEntitlements.ForWallet(strings.ToUpper(ceTestWallet))
	if len(grants) != 1 {
		t.Errorf("ForWallet with a differently-cased wallet returned %d grants; want 1", len(grants))
	}
	if empty := l.consoleEntitlements.ForWallet("0xnobody"); empty == nil || len(empty) != 0 {
		t.Errorf("ForWallet for an unknown wallet = %v; want an empty, non-nil list", empty)
	}
}

func TestConsoleEntitlementIsIdempotentPerPurchase(t *testing.T) {
	calls := ceInstallVerifier(t, ceGoodRecord(), http.StatusOK)
	l := ceTestLobby(t, ceTestWallet, ceTestConsoleUID)

	first, granted, err := l.GrantConsoleEntitlement(ceRequest())
	if err != nil || !granted {
		t.Fatalf("first grant: granted=%v err=%v", granted, err)
	}
	second, granted2, err2 := l.GrantConsoleEntitlement(ceRequest())
	if err2 != nil {
		t.Fatalf("a REPLAYED fulfilment returned an error (%v): the platform retries, so a retry must be an idempotent no-op", err2)
	}
	if granted2 {
		t.Error("a replayed purchase GRANTED twice")
	}
	if second == nil || second.ReceiptID != first.ReceiptID {
		t.Errorf("replay returned %+v; want the existing grant %+v", second, first)
	}
	if n := l.consoleEntitlements.Count(); n != 1 {
		t.Errorf("registry holds %d grants after a replay; want 1", n)
	}
	if c := atomic.LoadInt32(calls); c != 1 {
		t.Errorf("platform verifier called %d times; want 1 — the replay must short-circuit before the platform call", c)
	}
	if s := l.txidMemoStats(); s.Committed != 1 {
		t.Errorf("memo = %+v; want one committed id", s)
	}
}

func ceAnyContains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if strings.Contains(h, needle) {
			return true
		}
	}
	return false
}

// ── §2 A CLAIM IS MEASURED AGAINST THE PLATFORM'S RECORD ────────────────────

func TestConsoleEntitlementRefusesATamperedReceipt(t *testing.T) {
	leaseClaim := ceRequest()
	leaseClaim.Receipt.IsLeaseAction = true
	leaseClaim.Receipt.LeaseDuration = 9_999

	leaseRecord := ceGoodRecord()
	leaseRecord.IsLease = true
	leaseRecord.LeaseDurationBlocks = 500

	undatedLease := ceGoodRecord()
	undatedLease.IsLease = true

	otherProduct := ceGoodRecord()
	otherProduct.ProductCode = "DLC-SOMETHING-ELSE"

	otherAccount := ceGoodRecord()
	otherAccount.ConsoleUID = "0000000000000000"

	otherPlatform := ceGoodRecord()
	otherPlatform.Platform = PlatformPlayStation

	otherPurchase := ceGoodRecord()
	otherPurchase.PurchaseID = "xbox-purchase-9999"

	unverified := ceGoodRecord()
	unverified.Verified = false

	cases := []struct {
		name     string
		record   consolePurchaseRecord
		status   int
		req      consoleEntitlementRequest
		lobbyUID string
		want     string
	}{
		{name: "the platform sold a different product", record: otherProduct, want: "not the claimed"},
		{name: "the record is for another console account", record: otherAccount, want: "console account"},
		{name: "the record is for another platform", record: otherPlatform, want: "PLAYSTATION_5, not XBOX_SERIES_X"},
		{name: "the record is for another purchase", record: otherPurchase, want: "purchase"},
		{name: "the platform did not verify it", record: unverified, want: "does not mark this purchase as verified"},
		{name: "the claim calls a purchase a lease", record: ceGoodRecord(), req: leaseClaim, want: "but the claim is a lease"},
		{name: "a lease with no duration", record: undatedLease, req: leaseClaim, want: "no duration"},
		{name: "the platform answered 500", record: ceGoodRecord(), status: http.StatusInternalServerError, want: "HTTP 500"},
		{name: "the platform answered 404", record: ceGoodRecord(), status: http.StatusNotFound, want: "HTTP 404"},
		{name: "the console account has no VERIFIED link", record: ceGoodRecord(), lobbyUID: "0xsomeoneelse", want: "not linked to a VERIFIED wallet"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ceClearVerifiers()
			t.Cleanup(ceClearVerifiers)
			uid := tc.lobbyUID
			if uid == "" {
				uid = ceTestConsoleUID
			}
			l := ceTestLobby(t, ceTestWallet, uid)
			ceInstallVerifier(t, tc.record, tc.status)

			req := tc.req
			if req.PlatformPurchaseID == "" {
				req = ceRequest()
			}
			ent, granted, err := l.GrantConsoleEntitlement(req)
			if err == nil {
				t.Fatalf("the claim was GRANTED with mismatch %q: %+v", tc.name, ent)
			}
			if granted || ent != nil {
				t.Errorf("granted=%v ent=%v; want a refusal that writes nothing", granted, ent)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("refusal %q does not name the mismatch (%q)", err.Error(), tc.want)
			}
			if n := l.consoleEntitlements.Count(); n != 0 {
				t.Errorf("registry holds %d grants after a refusal; want 0", n)
			}
			// A REFUSED attempt must free its reservation, or one bad answer would block the retry for
			// as long as the process lives.
			if l.isTxIDConsumed(ceTestPurchase) {
				t.Error("the purchase id is still consumed after a refusal: a genuine retry would be blocked for ever")
			}
		})
	}
}

func TestConsoleEntitlementRefusesAGarbledPlatformAnswer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("this is not a purchase record"))
	}))
	t.Cleanup(func() {
		srv.Close()
		ceClearVerifiers()
	})
	ceClearVerifiers()
	registerConsolePurchaseVerifier(PlatformXbox, newHTTPConsolePurchaseVerifier(srv.URL, ceTestVerifier, srv.Client()))

	l := ceTestLobby(t, ceTestWallet, ceTestConsoleUID)
	if _, granted, err := l.GrantConsoleEntitlement(ceRequest()); err == nil || granted {
		t.Fatalf("a payload that is not a purchase record was accepted: granted=%v err=%v", granted, err)
	}
	if n := l.consoleEntitlements.Count(); n != 0 {
		t.Errorf("registry holds %d grants after a garbled answer; want 0", n)
	}
}

// ── §4 THE VERIFIERS COME FROM THE ENVIRONMENT, AND THE CONTRACT SAYS SO ────

func TestConsoleEntitlementVerifiersComeFromTheEnvironment(t *testing.T) {
	ceClearVerifiers()
	t.Cleanup(ceClearVerifiers)

	t.Setenv("CONSOLE_VERIFIER_URL_XBOX_SERIES_X", "")
	t.Setenv("CONSOLE_VERIFIER_SECRET_XBOX_SERIES_X", "")
	if n := syncConsoleVerifiersFromEnv(); n != 0 {
		t.Errorf("registered %d verifiers with nothing in the environment; want 0", n)
	}
	if _, ok := consoleVerifierFor(PlatformXbox); ok {
		t.Error("a verifier exists with no environment configuration")
	}

	// A URL without its secret is NOT half-configured: a verifier that cannot authenticate is not a
	// verifier, so it must not be registered at all.
	t.Setenv("CONSOLE_VERIFIER_URL_XBOX_SERIES_X", "https://example.invalid/verify")
	if n := syncConsoleVerifiersFromEnv(); n != 0 {
		t.Errorf("registered %d verifiers from a URL with no secret; want 0", n)
	}

	t.Setenv("CONSOLE_VERIFIER_SECRET_XBOX_SERIES_X", ceTestVerifier)
	if n := syncConsoleVerifiersFromEnv(); n != 1 {
		t.Fatalf("registered %d verifiers from a complete environment; want 1", n)
	}
	if _, ok := consoleVerifierFor(PlatformXbox); !ok {
		t.Error("the Xbox verifier was not registered from the environment")
	}
	if _, ok := consoleVerifierFor(PlatformPlayStation); ok {
		t.Error("a PlayStation verifier exists although only Xbox was configured")
	}
}

func TestConsoleEntitlementContractNamesItsBlockers(t *testing.T) {
	ceClearVerifiers()
	t.Cleanup(ceClearVerifiers)
	ceInstallVerifier(t, ceGoodRecord(), http.StatusOK)
	t.Setenv(consoleFulfilmentSecretEnv, ceTestSecret)

	c := consoleEntitlementContract()
	if c["available"] != true {
		t.Errorf("contract available=%v with a secret and a verifier; want true", c["available"])
	}
	if c["verifiers_configured"] != 1 {
		t.Errorf("verifiers_configured=%v; want 1", c["verifiers_configured"])
	}
	platforms, ok := c["platforms"].([]map[string]interface{})
	if !ok || len(platforms) != 3 {
		t.Fatalf("platforms=%v; want all three platforms listed", c["platforms"])
	}
	found := false
	for _, p := range platforms {
		if p["platform"] == string(PlatformXbox) {
			found = true
			if p["verifier_configured"] != true {
				t.Errorf("Xbox verifier_configured=%v; want true", p["verifier_configured"])
			}
			if p["verifier_url_env"] != "CONSOLE_VERIFIER_URL_XBOX_SERIES_X" {
				t.Errorf("Xbox verifier_url_env=%v; want the environment variable it needs", p["verifier_url_env"])
			}
		}
	}
	if !found {
		t.Error("the Xbox platform is missing from the served contract")
	}

	blocked, _ := c["blocked_by"].([]string)
	if !ceAnyContains(blocked, "lease EXPIRY") || !ceAnyContains(blocked, "credits NOBODY") {
		t.Errorf("the STRUCTURAL blockers are missing: %v", blocked)
	}
	for _, key := range []string{"verification_rule", "money_rule", "idempotency_rule", "lease_rule", "authoritative_surface"} {
		s, _ := c[key].(string)
		if strings.TrimSpace(s) == "" {
			t.Errorf("contract key %q is empty; a served rule must say what it does", key)
		}
	}
	if rule, _ := c["money_rule"].(string); !strings.Contains(rule, "NO balance") {
		t.Errorf("money_rule %q does not state that this path moves no money", rule)
	}
}

// ── §3 THE PATH MOVES NO MONEY ──────────────────────────────────────────────

func TestConsoleEntitlementMovesNoMoney(t *testing.T) {
	ceInstallVerifier(t, ceGoodRecord(), http.StatusOK)
	l := ceTestLobby(t, ceTestWallet, ceTestConsoleUID)

	balanceBefore := l.playerBalances[ceTestWallet]
	statsBefore := l.leaderboard[ceTestWallet]
	vouchersBefore := statsBefore.ArenaVouchers
	cardBefore := statsBefore.Inventory["CARD-1"]
	invLenBefore := len(statsBefore.Inventory)

	ent, granted, err := l.GrantConsoleEntitlement(ceRequest())
	if err != nil || !granted {
		t.Fatalf("grant: granted=%v err=%v", granted, err)
	}
	if ent == nil {
		t.Fatal("no entitlement returned")
	}

	if l.playerBalances[ceTestWallet] != balanceBefore {
		t.Errorf("balance changed on a platform sale (%d -> %d): the PLATFORM collected the money", balanceBefore, l.playerBalances[ceTestWallet])
	}
	statsAfter := l.leaderboard[ceTestWallet]
	if statsAfter.ArenaVouchers != vouchersBefore {
		t.Errorf("ArenaVouchers changed (%d -> %d); this path must not touch them", vouchersBefore, statsAfter.ArenaVouchers)
	}
	if statsAfter.Inventory["CARD-1"] != cardBefore || len(statsAfter.Inventory) != invLenBefore {
		t.Errorf("inventory changed (%v -> %v); a grant must not move stock", statsBefore.Inventory, statsAfter.Inventory)
	}
	// The only state a grant may add is the grant itself, plus its one memo entry.
	if n := l.consoleEntitlements.Count(); n != 1 {
		t.Errorf("registry holds %d grants; want 1", n)
	}
	if n := len(l.registeredTxIDs); n != 1 {
		t.Errorf("memo holds %d committed ids; want 1", n)
	}
}

// ── §5 THE HTTP BOUNDARY ────────────────────────────────────────────────────

func TestConsoleEntitlementHTTPBoundary(t *testing.T) {
	ceClearVerifiers()
	t.Cleanup(ceClearVerifiers)
	l := ceTestLobby(t, ceTestWallet, ceTestConsoleUID)

	goodBody := `{"receipt":{"platform_ref":"XBOX_SERIES_X","console_user_uid":"2533274899999999","dlc_product_code":"DLC-ARENA-PASS"},"platform_purchase_id":"` + ceTestPurchase + `"}`
	unknownFieldBody := `{"receipt":{"platform_ref":"XBOX_SERIES_X","console_user_uid":"2533274899999999","dlc_product_code":"DLC-ARENA-PASS"},"platform_purchase_id":"` + ceTestPurchase + `","price_micro":1}`

	// 1. GET on the writer.
	rr := httptest.NewRecorder()
	l.handleConsoleEntitlementGrant(rr, httptest.NewRequest(http.MethodGet, "/api/console/entitlement", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET on the writer = %d; want 405", rr.Code)
	}

	// 2. NO SECRET CONFIGURED AT ALL: fail closed. There is deliberately no "unset means open" path.
	t.Setenv(consoleFulfilmentSecretEnv, "")
	rr = httptest.NewRecorder()
	l.handleConsoleEntitlementGrant(rr, httptest.NewRequest(http.MethodPost, "/api/console/entitlement", strings.NewReader(goodBody)))
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("POST with no configured secret = %d; want 401 (fail closed)", rr.Code)
	}

	// 3. The wrong secret.
	t.Setenv(consoleFulfilmentSecretEnv, ceTestSecret)
	req := httptest.NewRequest(http.MethodPost, "/api/console/entitlement", strings.NewReader(goodBody))
	req.Header.Set(consoleFulfilmentSecretHdr, "not-the-secret")
	rr = httptest.NewRecorder()
	l.handleConsoleEntitlementGrant(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("POST with a wrong secret = %d; want 401", rr.Code)
	}

	// 4. A body that names a price: the decoder must refuse it BEFORE any lookup, so a caller can never
	//    name a price, a wallet or a duration.
	req = httptest.NewRequest(http.MethodPost, "/api/console/entitlement", strings.NewReader(unknownFieldBody))
	req.Header.Set(consoleFulfilmentSecretHdr, ceTestSecret)
	rr = httptest.NewRecorder()
	l.handleConsoleEntitlementGrant(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("POST naming price_micro = %d; want 400 (a caller may name a purchase id and nothing else)", rr.Code)
	}

	// 5. Correct secret, correct body, NO verifier: 409 carrying its blockers.
	req = httptest.NewRequest(http.MethodPost, "/api/console/entitlement", strings.NewReader(goodBody))
	req.Header.Set(consoleFulfilmentSecretHdr, ceTestSecret)
	rr = httptest.NewRecorder()
	l.handleConsoleEntitlementGrant(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("POST with no verifier = %d; want 409 with its reason", rr.Code)
	}
	var refusal struct {
		Success  bool                   `json:"success"`
		Error    string                 `json:"error"`
		Contract map[string]interface{} `json:"contract"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &refusal); err != nil {
		t.Fatalf("the 409 body is not the served refusal shape: %v", err)
	}
	if refusal.Success || refusal.Error == "" {
		t.Errorf("409 body = %+v; want success=false and a named reason", refusal)
	}
	if refusal.Contract == nil || refusal.Contract["blocked_by"] == nil {
		t.Errorf("409 body carries no contract; a refusal must always say why: %s", rr.Body.String())
	}

	// 6. With a verifier: the grant, then the idempotent replay.
	ceInstallVerifier(t, ceGoodRecord(), http.StatusOK)
	req = httptest.NewRequest(http.MethodPost, "/api/console/entitlement", strings.NewReader(goodBody))
	req.Header.Set(consoleFulfilmentSecretHdr, ceTestSecret)
	rr = httptest.NewRecorder()
	l.handleConsoleEntitlementGrant(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("a verified purchase = %d (%s); want 200", rr.Code, rr.Body.String())
	}
	var ok struct {
		Granted    bool `json:"granted"`
		Idempotent bool `json:"idempotent"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &ok); err != nil || !ok.Granted || ok.Idempotent {
		t.Fatalf("grant body = %s (err=%v); want granted=true", rr.Body.String(), err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/console/entitlement", strings.NewReader(goodBody))
	req.Header.Set(consoleFulfilmentSecretHdr, ceTestSecret)
	rr = httptest.NewRecorder()
	l.handleConsoleEntitlementGrant(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("the REPLAY = %d; want 200 as an idempotent no-op", rr.Code)
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &ok); err != nil || ok.Granted || !ok.Idempotent {
		t.Fatalf("replay body = %s; want granted=false idempotent=true", rr.Body.String())
	}

	// 7. The READ path.
	rr = httptest.NewRecorder()
	l.handleConsoleEntitlements(rr, httptest.NewRequest(http.MethodGet, "/api/console/entitlements", nil))
	if rr.Code != http.StatusBadRequest {
		t.Errorf("read with no wallet = %d; want 400", rr.Code)
	}

	rr = httptest.NewRecorder()
	l.handleConsoleEntitlements(rr, httptest.NewRequest(http.MethodGet, "/api/console/entitlements?wallet="+ceTestWallet, nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("read = %d; want 200", rr.Code)
	}
	var read struct {
		Count        int                      `json:"count"`
		Entitlements []map[string]interface{} `json:"entitlements"`
		Contract     map[string]interface{}   `json:"contract"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &read); err != nil {
		t.Fatalf("read body is not JSON: %v", err)
	}
	if read.Count != 1 || len(read.Entitlements) != 1 {
		t.Errorf("read count=%d entitlements=%d; want 1 and 1", read.Count, len(read.Entitlements))
	}
	if read.Contract == nil {
		t.Error("the read does not serve the contract, so a viewer cannot see what is blocked")
	}

	// An unknown wallet must answer an EMPTY LIST, never null: an absent list must not read as an
	// absent feature.
	rr = httptest.NewRecorder()
	l.handleConsoleEntitlements(rr, httptest.NewRequest(http.MethodGet, "/api/console/entitlements?wallet=0xnobody", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("read for an unknown wallet = %d; want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"entitlements":[]`) {
		t.Errorf("read for an unknown wallet = %s; want an empty (non-null) list", rr.Body.String())
	}

	// POST on the read is refused.
	rr = httptest.NewRecorder()
	l.handleConsoleEntitlements(rr, httptest.NewRequest(http.MethodPost, "/api/console/entitlements", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST on the read = %d; want 405", rr.Code)
	}
}
