//go:build !js && !wasm

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ════════════════════════════════════════════════════════════════════════════
// THE CAREER ROLE VOCABULARY + THE INTEL-AGENT CYBER-INTERCEPT DOOR
//
// Three defects are pinned here, all found by reading the door against the engine
// it talks to:
//
//  1. THE ROLE SPELLINGS. One career reached the engine under two names, so
//     `GetRivalPairName`'s `==` answered "" for half the pairings (a silent skip:
//     no error, no log, no XP), `RoleXP` written under one name was invisible to a
//     reader using the other, and `IsJusticeAligned` called a justice-aligned
//     TARGET an enemy because it only knew the display spelling.
//
//  2. THE SELF-DEADLOCK. `handleCyberIntercept` holds the lobby WRITE lock and
//     called the self-locking `l.isJusticeAligned` — unconditional on a
//     sync.RWMutex, and the write lock is never released, so the whole process
//     froze. It fired only when an Arc-Net Operative was on the leaderboard, which
//     is why it was never reproduced. Tests that touch this door fail by TIMEOUT.
//
//  3. THE UNREACHABLE WALLET. The door required an `X-Client-ID` header that no
//     module in Public/** sets, so it could only ever answer 401.
// ════════════════════════════════════════════════════════════════════════════

const (
	cyberCallerWallet   = "0xMixEdIntel" // mixed case: the request speaks lower case
	cyberAllyWallet     = "0xArcNetAlly"
	cyberOutsiderWallet = "0xOutsider"
	cyberCostMicro      = uint64(500 * 1000000) // the documented 500 $VBV scan fee
)

// cyberLobby is the minimum the door needs. The caller's Intel-Agent XP is stored
// under the IDENTIFIER spelling while the handler asks in the DISPLAY one, so the
// vocabulary is exercised by every test in this file rather than by a unit test
// alone. DataDir is a temp dir so the audit log never lands in the repository.
func cyberLobby(t *testing.T) *Lobby {
	t.Helper()
	return &Lobby{
		DataDir: t.TempDir(),
		leaderboard: map[string]PlayerStats{
			cyberCallerWallet: {
				Wallet:   cyberCallerWallet,
				JobRole:  "Warden", // justice-aligned, so the ally branch is reachable
				CareerXP: &CareerXP{RoleXP: map[string]uint64{"IntelAgent": 3000}},
			},
			cyberAllyWallet: {
				Wallet:   cyberAllyWallet,
				JobRole:  "Arc-Net Operative",
				CareerXP: &CareerXP{RoleXP: map[string]uint64{"Arc-Net Operative": 700}},
			},
			cyberOutsiderWallet: {
				Wallet:   cyberOutsiderWallet,
				JobRole:  "Smuggler",
				CareerXP: &CareerXP{RoleXP: map[string]uint64{"Smuggler": 100}},
			},
		},
		playerBalances: map[string]uint64{cyberCallerWallet: 2_000_000_000}, // 2,000 $VBV
		evidencePool:   &EvidencePool{ActiveRecords: map[string]*RaidEvidence{}},
		broadcast:      make(chan []byte, 8),
	}
}

// cyberCall drives the handler on its own goroutine so a lock regression fails by
// TIMEOUT instead of hanging the suite: a deadlock raises no error at all.
func cyberCall(t *testing.T, l *Lobby, wallet, body string) *httptest.ResponseRecorder {
	t.Helper()
	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		req := httptest.NewRequest(http.MethodPost, "/api/criminality/cyber-intercept", strings.NewReader(body))
		if wallet != "" {
			req.Header.Set("X-Wallet-Address", wallet)
		}
		rec := httptest.NewRecorder()
		l.handleCyberIntercept(rec, req)
		done <- rec
	}()
	select {
	case rec := <-done:
		return rec
	case <-time.After(5 * time.Second):
		t.Fatal("handleCyberIntercept did not return within 5s — it re-acquired the lobby lock " +
			"(self-deadlock), which freezes every request and every WebSocket in the process")
		return nil
	}
}

func cyberJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("the response is not JSON (%v): %s", err, rec.Body.String())
	}
	return out
}

// ── 1. THE VOCABULARY ───────────────────────────────────────────────────────

func TestRoleKeyFoldsOneCareerAcrossEverySpelling(t *testing.T) {
	same := [][2]string{
		{"Arc-Net Operative", "ArcNetOperative"},
		{"Tax Auditor", "TaxAuditor"},
		{"Bounty Hunter", "BountyHunter"},
		{"Justice Commissioner", "JusticeCommissioner"},
		{"Justice Recruiter", "JusticeRecruiter"},
		{"Sector Peacekeeper", "SectorPeacekeeper"},
		{"Heist Planner", "HeistPlanner"},
		{"Mutation Log Auditor", "MutationAuditor"}, // the declared abbreviation
		{"Intel-Agent", "IntelAgent"},
		{"IntelAgent", "Int.Agent"}, // handlers_criminality.go's spelling
		{"intel-agent", "INTEL AGENT"},
	}
	for _, p := range same {
		if a, b := RoleKey(p[0]), RoleKey(p[1]); a != b {
			t.Errorf("RoleKey(%q)=%q and RoleKey(%q)=%q — one career must fold to ONE key", p[0], a, p[1], b)
		}
	}

	// Negative control: the fold must not collapse unrelated careers.
	differ := [][2]string{
		{"Warden", "Smuggler"},
		{"Fence", "Gossip"},
		{"Judge", "Kidnapper"},
		{"Launderer", "Forensic Analyst"},
		{"Intel-Agent", "Arc-Net Operative"},
	}
	for _, p := range differ {
		if RoleKey(p[0]) == RoleKey(p[1]) {
			t.Errorf("RoleKey(%q) == RoleKey(%q) = %q — two different careers", p[0], p[1], RoleKey(p[0]))
		}
	}
}

func TestRivalPairNameResolvesAcrossSpellings(t *testing.T) {
	// Every one of these answered "" before the vocabulary: the pair table declares
	// the display spelling while the combat hooks pass the identifier spelling.
	cases := [][3]string{
		{"IntelAgent", "ArcNetOperative", "IntelAgent↔ArcNetOperative"},
		{"Int.Agent", "Arc-Net Operative", "IntelAgent↔ArcNetOperative"},
		{"ForensicAnalyst", "Gossip", "ForensicAnalyst↔Gossip"},
		{"TaxAuditor", "Launderer", "TaxAuditor↔Launderer"},
		{"BountyHunter", "Kidnapper", "BountyHunter↔Kidnapper"},
		{"JusticeRecruiter", "MutationAuditor", "JusticeRecruiter↔MutationLogAuditor"},
		{"SectorPeacekeeper", "Smuggler", "SectorPeacekeeper↔Smuggler"},
		// The display spellings still resolve, and the name does NOT depend on the argument order:
		// ONE row per unordered pair, so the reverse lookup answers the same canonical name.
		{"Forensic Analyst", "Gossip", "ForensicAnalyst↔Gossip"},
		{"Gossip", "Forensic Analyst", "ForensicAnalyst↔Gossip"},
	}
	for _, c := range cases {
		if got := GetRivalPairName(c[0], c[1]); got != c[2] {
			t.Errorf("GetRivalPairName(%q, %q) = %q; want %q", c[0], c[1], got, c[2])
		}
	}

	// Negative control: a combination with no declared pair stays unpairable, so the
	// hooks around it keep answering "not a rival" instead of paying a bonus.
	if got := GetRivalPairName("Warden", "Fence"); got != "" {
		t.Errorf("GetRivalPairName(Warden, Fence) = %q; want \"\" (no declared pair)", got)
	}
}

func TestRoleXPReadsTheKeyTheWriterChose(t *testing.T) {
	cxp := &CareerXP{RoleXP: map[string]uint64{"Int.Agent": 2500}}

	if got := roleXP(cxp, "IntelAgent"); got != 2500 {
		t.Errorf("roleXP(IntelAgent) = %d; want 2500 — XP written under another spelling of the same career", got)
	}
	if got := cxp.GetCareerTier("IntelAgent"); got != 1 {
		t.Errorf("GetCareerTier(IntelAgent) = %d; want 1 (2500/1500)", got)
	}
	if got := getTierFor(cxp, "IntelAgent"); got != 2 {
		t.Errorf("getTierFor(IntelAgent) = %d; want 2 (2500 >= 500)", got)
	}
	if !HasCareer(cxp, "Intel-Agent") {
		t.Error("HasCareer(Intel-Agent) = false; the writer spelled it Int.Agent")
	}

	// Two keys for one role: the HIGHEST is read, never the sum (a sum would invent
	// XP that was never awarded).
	split := &CareerXP{RoleXP: map[string]uint64{"IntelAgent": 1000, "Int.Agent": 2500}}
	if got := roleXP(split, "Intel-Agent"); got != 2500 {
		t.Errorf("roleXP with two spellings = %d; want 2500 (the highest, not 3500)", got)
	}
}

func TestIsJusticeAlignedAcceptsEverySpellingOfOneRole(t *testing.T) {
	for _, role := range []string{"IntelAgent", "Intel-Agent", "Int.Agent", "Tax Auditor", "TaxAuditor", "Warden", "Judge"} {
		if !IsJusticeAligned(role) {
			t.Errorf("IsJusticeAligned(%q) = false; it is a justice role", role)
		}
	}
	for _, role := range []string{"Smuggler", "Kidnapper", "Launderer", "Gossip", "Fence", ""} {
		if IsJusticeAligned(role) {
			t.Errorf("IsJusticeAligned(%q) = true; it is not a justice role", role)
		}
	}
}

// TestJusticeAlignedHasALockHeldForm pins BOTH halves of the rule: the unlocked
// face still takes the lock itself, and the locked face does not.
func TestJusticeAlignedHasALockHeldForm(t *testing.T) {
	l := cyberLobby(t)

	// The unlocked face, called with nothing held — it must take and release the lock.
	if !l.isJusticeAligned(cyberCallerWallet) {
		t.Error("isJusticeAligned(Warden caller) = false; want true")
	}
	if l.isJusticeAligned(cyberOutsiderWallet) {
		t.Error("isJusticeAligned(Smuggler) = true; want false")
	}
	// Prove the lock is free again: a plain Lock() would hang if it were not.
	locked := make(chan struct{}, 1)
	go func() { l.mutex.Lock(); l.mutex.Unlock(); locked <- struct{}{} }()
	select {
	case <-locked:
	case <-time.After(2 * time.Second):
		t.Fatal("isJusticeAligned left the lobby lock held")
	}

	// The locked face, called WITH the write lock held — the handler's case.
	done := make(chan bool, 1)
	go func() {
		l.mutex.Lock()
		defer l.mutex.Unlock()
		done <- l.isJusticeAlignedLocked(cyberCallerWallet)
	}()
	select {
	case got := <-done:
		if !got {
			t.Error("isJusticeAlignedLocked(Warden caller) = false; want true")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("isJusticeAlignedLocked blocked while the lock was held (self-deadlock)")
	}
}

func TestCareerXPGateIsIntegerAndExact(t *testing.T) {
	cases := []struct {
		sustained uint64
		permille  uint64
	}{
		{VBVTierPeonMicro, 1000},
		{VBVTierApprentice, 2000},
		{VBVTierJourneyman, 4000},
		{VBVTierExpert, 8000},
		{VBVTierMaster, 16000},
		{VBVTierBoss, 32000},
	}
	for _, c := range cases {
		cxp := &CareerXP{AvgSustainedMicro: c.sustained}
		if got := cxp.GetVBVGatingPermille(); got != c.permille {
			t.Errorf("GetVBVGatingPermille at %d micro = %d; want %d", c.sustained, got, c.permille)
		}
		// The float form is the DISPLAY wrapper: it must agree exactly.
		if got := cxp.GetVBVGatingMultiplier() * 1000; uint64(got) != c.permille {
			t.Errorf("GetVBVGatingMultiplier at %d micro = %v; the permille form says %d", c.sustained, got, c.permille)
		}
	}

	// The award the door makes is base × gate / 1000, in integers.
	base := uint64(50)
	if got := base * (&CareerXP{}).GetVBVGatingPermille() / 1000; got != 50 {
		t.Errorf("ungated award = %d; want 50", got)
	}
	if got := base * (&CareerXP{AvgSustainedMicro: VBVTierBoss}).GetVBVGatingPermille() / 1000; got != 1600 {
		t.Errorf("Boss-gated award = %d; want 1600", got)
	}
}

// ── 2. THE DOOR ─────────────────────────────────────────────────────────────

// TestCyberInterceptPaysAndDebitsExactly drives the whole door end to end: a wallet
// it can actually resolve (no X-Client-ID), the fee debited from the balance map and
// credited to the faucet, the integer XP, and a recorded fact in the pool.
func TestCyberInterceptPaysAndDebitsExactly(t *testing.T) {
	l := cyberLobby(t)

	// The request speaks LOWER CASE for a wallet that is stored mixed case.
	rec := cyberCall(t, l, strings.ToLower(cyberCallerWallet), `{}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200 (%s)", rec.Code, rec.Body.String())
	}
	body := cyberJSON(t, rec)

	if body["success"] != true {
		t.Errorf("success = %v; want true", body["success"])
	}
	if got := body["xp_awarded"]; got != float64(50) {
		t.Errorf("xp_awarded = %v; want 50 (base 50, ungated: AvgSustainedMicro is 0)", got)
	}
	if got := body["cost_micro"]; got != float64(cyberCostMicro) {
		t.Errorf("cost_micro = %v; want %d", got, cyberCostMicro)
	}
	if got := body["decrypt_bonus_permille"]; got != float64(1000) {
		t.Errorf("decrypt_bonus_permille = %v; want 1000 (the INTEGER form the XP maths uses)", got)
	}
	if got := body["wallet"]; got != cyberCallerWallet {
		t.Errorf("wallet = %v; want the STORED spelling %q", got, cyberCallerWallet)
	}

	l.mutex.RLock()
	defer l.mutex.RUnlock()

	if got := l.playerBalances[cyberCallerWallet]; got != 1_500_000_000 {
		t.Errorf("balance = %d; want 1500000000 (2,000 minus the 500 $VBV scan fee)", got)
	}
	if l.faucetBalanceMicro != cyberCostMicro {
		t.Errorf("faucet = %d; want %d — the fee must reconcile into the deterministic sink",
			l.faucetBalanceMicro, cyberCostMicro)
	}
	caller := l.leaderboard[cyberCallerWallet]
	if got := caller.CareerXP.RoleXP["Intel-Agent"]; got != 50 {
		t.Errorf("Intel-Agent XP = %d; want 50 (awarded under the canonical role)", got)
	}
	if got := caller.CareerXP.RoleXP["IntelAgent"]; got != 3000 {
		t.Errorf("the pre-existing IntelAgent XP key was disturbed: %d; want 3000", got)
	}
	if n := len(l.evidencePool.ActiveRecords); n != 1 {
		t.Errorf("evidence records = %d; want 1 (an intercept is a recorded fact)", n)
	}
}

// TestCyberInterceptGatesXPWithTheIntegerForm proves the $VBV gate reaches the award
// without a float: the same base pays 32× at the Boss tier.
func TestCyberInterceptGatesXPWithTheIntegerForm(t *testing.T) {
	l := cyberLobby(t)
	l.mutex.Lock()
	caller := l.leaderboard[cyberCallerWallet]
	caller.CareerXP.AvgSustainedMicro = VBVTierBoss
	l.leaderboard[cyberCallerWallet] = caller
	l.mutex.Unlock()

	body := cyberJSON(t, cyberCall(t, l, cyberCallerWallet, `{}`))
	if got := body["xp_awarded"]; got != float64(1600) {
		t.Errorf("xp_awarded = %v; want 1600 (50 × 32× at the Boss tier)", got)
	}
	if got := body["decrypt_bonus_permille"]; got != float64(32000) {
		t.Errorf("decrypt_bonus_permille = %v; want 32000", got)
	}
}

// TestCyberInterceptRefusalsMoveNothing covers every refusal the door has, and each
// one asserts that NOTHING changed: no balance, no faucet credit, no XP, no record.
func TestCyberInterceptRefusalsMoveNothing(t *testing.T) {
	before := func(l *Lobby) (uint64, uint64, uint64, int) {
		l.mutex.RLock()
		defer l.mutex.RUnlock()
		return l.playerBalances[cyberCallerWallet], l.faucetBalanceMicro,
			l.leaderboard[cyberCallerWallet].CareerXP.RoleXP["IntelAgent"], len(l.evidencePool.ActiveRecords)
	}

	cases := []struct {
		name   string
		wallet string
		body   string
		status int
	}{
		{"no wallet at all", "", `{}`, http.StatusUnauthorized},
		{"a wallet the engine does not know", "0xNobody", `{}`, http.StatusNotFound},
		{"a wallet with no Intel-Agent career", cyberOutsiderWallet, `{}`, http.StatusForbidden},
		{"a body that is not JSON", cyberCallerWallet, `not json`, http.StatusBadRequest},
	}
	for _, c := range cases {
		l := cyberLobby(t)
		bal, faucet, xp, records := before(l)

		rec := cyberCall(t, l, c.wallet, c.body)
		if rec.Code != c.status {
			t.Errorf("%s: status = %d; want %d (%s)", c.name, rec.Code, c.status, rec.Body.String())
		}
		if b2, f2, x2, r2 := before(l); b2 != bal || f2 != faucet || x2 != xp || r2 != records {
			t.Errorf("%s: a refused call MOVED state (balance %d→%d, faucet %d→%d, xp %d→%d, records %d→%d)",
				c.name, bal, b2, faucet, f2, xp, x2, records, r2)
		}
	}

	// Insufficient funds is its own refusal, and it names the price AND the balance.
	poor := cyberLobby(t)
	poor.playerBalances[cyberCallerWallet] = cyberCostMicro - 1
	rec := cyberCall(t, poor, cyberCallerWallet, `{}`)
	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("unfunded status = %d; want 402 (%s)", rec.Code, rec.Body.String())
	}
	body := cyberJSON(t, rec)
	if got := body["cost_micro"]; got != float64(cyberCostMicro) {
		t.Errorf("the 402 must state the price: cost_micro = %v; want %d", got, cyberCostMicro)
	}
	if got := body["balance_micro"]; got != float64(cyberCostMicro-1) {
		t.Errorf("the 402 must state the balance: balance_micro = %v; want %d", got, cyberCostMicro-1)
	}
}

// TestCyberInterceptAllySharingIsReachable is BOTH regressions at once.
//
// The ally branch was unreachable twice over: the comparison read the DEFENDER's
// award (base × 0.30) against `baseXP`, which can never hold, AND the branch called
// the self-locking `l.isJusticeAligned` while the write lock was held — firing
// whenever an Arc-Net Operative was on the leaderboard, which is exactly the lobby
// this test builds. A deadlock fails by TIMEOUT here.
func TestCyberInterceptAllySharingIsReachable(t *testing.T) {
	l := cyberLobby(t)

	rec := cyberCall(t, l, cyberCallerWallet, `{}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200 (%s)", rec.Code, rec.Body.String())
	}

	l.mutex.RLock()
	defer l.mutex.RUnlock()

	ally := l.leaderboard[cyberAllyWallet]
	// attackerXP = uint64(50 × 1.1 × 1.3) = 71 (the pair is antagonistic: delta −10
	// gives ×1.1; the tier-3 modifier gives ×1.3), so the ally's share is 71 − 50 = 21.
	if got := ally.CareerXP.RoleXP["Arc-Net Operative"]; got != 721 {
		t.Errorf("ally XP = %d; want 721 (700 + the 21-point shared visibility award)", got)
	}
	// The outsider is not an Arc-Net Operative and must not be touched.
	if got := l.leaderboard[cyberOutsiderWallet].CareerXP.RoleXP["Smuggler"]; got != 100 {
		t.Errorf("an unrelated wallet was awarded XP: %d; want 100", got)
	}
}

// TestCyberInterceptReInterceptsAnExistingEvent pins the other branch: naming an
// event already in the pool costs nothing and marks the record intercepted.
func TestCyberInterceptReInterceptsAnExistingEvent(t *testing.T) {
	l := cyberLobby(t)
	l.evidencePool.ActiveRecords["CYBER-SEED"] = &RaidEvidence{
		EvidenceID: "CYBER-SEED", SourceWallet: cyberOutsiderWallet, CrimeType: "CYBER_INTERCEPT",
	}

	rec := cyberCall(t, l, cyberCallerWallet, `{"event_id":"CYBER-SEED"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200 (%s)", rec.Code, rec.Body.String())
	}
	if got := cyberJSON(t, rec)["cost_micro"]; got != float64(0) {
		t.Errorf("cost_micro = %v; want 0 — re-intercepting a known event is free", got)
	}

	l.mutex.RLock()
	defer l.mutex.RUnlock()
	if l.playerBalances[cyberCallerWallet] != 2_000_000_000 {
		t.Error("a free re-intercept debited the balance")
	}
	if !l.evidencePool.ActiveRecords["CYBER-SEED"].Intercepted {
		t.Error("the existing record was not marked intercepted")
	}
}

// ── 3. THE PAIR TABLE'S OWN INVARIANTS ──────────────────────────────────────

// TestRivalPairTableDeclaresEachPairOnce pins the invariant that let ONE pair be both an ally and an
// enemy: the scan returns on the FIRST match and the match is order-independent, so a second row for
// the same unordered pair can never be reached and its name can contradict the first.
func TestRivalPairTableDeclaresEachPairOnce(t *testing.T) {
	seen := map[string]string{}
	for _, p := range rivalPairTable {
		a, b := RoleKey(p.A), RoleKey(p.B)
		if a > b {
			a, b = b, a
		}
		key := a + "|" + b
		if first, dup := seen[key]; dup {
			t.Errorf("the pair {%s, %s} is declared TWICE (as %q and as %q); the scan is "+
				"order-independent and returns on the first match, so %q can never be produced",
				p.A, p.B, first, p.Name, p.Name)
		}
		seen[key] = p.Name
	}
}

// TestEveryDeclaredPairHasAPrice pins the second invariant: a pair the table declares must have an XP
// delta, except the two P2-D8 / P2-D10 pairs that are declared WITHOUT one. A new pair added with no
// price fails here instead of quietly worth nothing.
func TestEveryDeclaredPairHasAPrice(t *testing.T) {
	unpriced := map[string]bool{
		"JusticeRecruiter↔MutationLogAuditor": true, // P2-D8: declared, no delta assigned
		"MutationLogAuditor↔Kidnapper":        true, // P2-D10: declared, no delta assigned
	}
	for _, p := range rivalPairTable {
		delta := GetRivalXPDelta(p.Name)
		if delta == 0 && !unpriced[p.Name] {
			t.Errorf("the declared pair %q (%s ↔ %s) has NO XP delta: GetRivalXPDelta does not know that name",
				p.Name, p.A, p.B)
		}
		if delta != 0 && unpriced[p.Name] {
			t.Errorf("the pair %q now HAS a delta (%d) — remove it from this test's declared-zero list",
				p.Name, delta)
		}
	}
}
