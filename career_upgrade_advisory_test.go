//go:build !js && !wasm

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ============================================================================
// THE UNLOCK ADVISORY AND THE STAFF REQUEST — "unlocked, never forced"
// ============================================================================
//
// WHAT THESE TESTS GUARD. The operator's rule has four parts and each one is a RESTRICTION on what
// the code may do, so each one is pinned as a property rather than described in a comment:
//
//  1. A career is UNLOCKED, never granted. `UnlockedCareerRolesLocked` must agree with the ONE gate
//     evaluator (so "unlocked" cannot become a second, softer gate), and deriving it must WRITE
//     NOTHING — no promotion, no notice, no record.
//  2. The LEVEL CAP is the unlock: walking ONE gate back (the level) must remove the unlock, so a
//     flag cannot masquerade as an achievement.
//  3. AN EMPLOYER MAY ASK, AND ONLY THE EMPLOYEE MAY ACT. `RequestStaffCareerUpgradeLocked` is
//     pinned to leave the employee's `PromotedRoles`, `JobRole` and `LessonLevel` UNTOUCHED on both
//     the success path and every refusal path. If this test ever fails, the request door has
//     silently become a promotion door.
//  4. NOTIFIED, NOT FORCED. The notice fires once per unlock (a second call returns nothing), and
//     it RE-ARMS when the role is promoted — so a career lost to a demotion and re-earned notifies
//     again instead of being silent forever.
// ============================================================================

const (
	cpEmployerWallet = "0xcpt-employer"
	cpStaffReady     = "0xcpt-staff-ready"   // meets a gate
	cpStaffNotReady  = "0xcpt-staff-not-yet" // does not
	cpOutsiderWallet = "0xcpt-outsider"
)

// careerUpgradeSeedStaff puts ONE wallet on an employer's club roster. It writes exactly what a hire
// writes (`Club.Staff`), so the projection is proven against the real employment record.
func careerUpgradeSeedStaff(l *Lobby, employer, staff, clubRole string) {
	club := l.clubs["club-"+employer]
	if club == nil {
		careerPathSeedOwner(l, employer, 2, true, VBVTierBoss)
		club = l.clubs["club-"+employer]
	}
	if club.Staff == nil {
		club.Staff = map[string]string{}
	}
	club.Staff[staff] = clubRole
}

// containsRole compares BY ROLE, the way every career reader in this repository does.
func containsRole(roles []string, want string) bool {
	for _, r := range roles {
		if RoleKey(r) == RoleKey(want) {
			return true
		}
	}
	return false
}

// ── 1. THE UNLOCK IS DERIVED, AND IT GRANTS NOTHING ───────────────────────────────────────────

func TestUnlockIsDerivedFromTheOneGateEvaluatorAndGrantsNothing(t *testing.T) {
	l := careerPathTestLobby(t)
	careerPathMakeEligible(l, cpJusticeWallet, CareerPathJustice, "Warden")

	unlocked := l.UnlockedCareerRolesLocked(cpJusticeWallet)
	if !containsRole(unlocked, "Warden") {
		t.Fatalf("a wallet meeting every gate does not report Warden unlocked: %v", unlocked)
	}

	// IT MUST AGREE WITH THE SERVED GATE: "unlocked" cannot be a second, softer gate that the
	// career table underneath it disagrees with.
	for _, role := range l.CareerUpgradeViewLocked(cpJusticeWallet).UnlockedRoles {
		req := l.CareerPromotionRequirementsForWalletLocked(cpJusticeWallet, role)
		if !req.Eligible {
			t.Errorf("%q is served as unlocked but its own gate says ineligible (%s)", role, req.Missing)
		}
	}

	// DERIVING IT WRITES NOTHING. Two reads, then the record must be untouched.
	before := l.leaderboard[cpJusticeWallet]
	_ = l.CareerUpgradeViewLocked(cpJusticeWallet)
	_ = l.CareerUpgradeViewLocked(cpJusticeWallet)
	after := l.leaderboard[cpJusticeWallet]
	if len(after.CareerXP.PromotedRoles) != len(before.CareerXP.PromotedRoles) {
		t.Errorf("deriving the advisory promoted somebody: %v -> %v",
			before.CareerXP.PromotedRoles, after.CareerXP.PromotedRoles)
	}
	if after.JobRole != before.JobRole {
		t.Errorf("deriving the advisory changed JobRole: %q -> %q", before.JobRole, after.JobRole)
	}
	if len(after.CareerXP.UnlockNoticesSent) != 0 {
		t.Errorf("deriving the advisory marked a notice sent: %v", after.CareerXP.UnlockNoticesSent)
	}

	// THE OFFER IS SERVED AS OPTIONAL, with the server's own basis.
	adv := l.CareerPathViewLocked(cpJusticeWallet).Upgrades
	if !adv.IsOptional {
		t.Error("the served advisory does not say the upgrade is optional")
	}
	if adv.Statement == "" || adv.UnlockBasis == "" || adv.NoticeRule == "" || adv.StaffBasis == "" {
		t.Errorf("the advisory is served without its basis: %+v", adv)
	}
	if adv.UnlockedCount != len(adv.UnlockedRoles) {
		t.Errorf("unlocked_count = %d but %d roles are listed", adv.UnlockedCount, len(adv.UnlockedRoles))
	}
}

func TestUnlockFollowsTheLevelCapAndNothingElse(t *testing.T) {
	l := careerPathTestLobby(t)
	careerPathMakeEligible(l, cpJusticeWallet, CareerPathJustice, "Warden")

	gate, declared := CareerRoleGateFor("Warden")
	if !declared {
		t.Fatal("Warden has no declared gate")
	}

	// One level BELOW the cap: not unlocked.
	st := l.leaderboard[cpJusticeWallet]
	st.CareerXP.LessonLevel = gate.MinLessonLevel - 1
	l.leaderboard[cpJusticeWallet] = st
	if got := l.UnlockedCareerRolesLocked(cpJusticeWallet); containsRole(got, "Warden") {
		t.Errorf("Warden is unlocked one level below its cap (%d); the level cap is not the unlock",
			gate.MinLessonLevel-1)
	}

	// AT the cap: unlocked.
	st = l.leaderboard[cpJusticeWallet]
	st.CareerXP.LessonLevel = gate.MinLessonLevel
	l.leaderboard[cpJusticeWallet] = st
	if got := l.UnlockedCareerRolesLocked(cpJusticeWallet); !containsRole(got, "Warden") {
		t.Errorf("Warden is not unlocked at exactly its cap (%d)", gate.MinLessonLevel)
	}
}

// ── 2. THE STAFF PROJECTION IS THE EMPLOYER'S OWN ROSTER ──────────────────────────────────────

func TestStaffProjectionIsTheEmployersOwnRoster(t *testing.T) {
	l := careerPathTestLobby(t)
	careerPathSeedOwner(l, cpEmployerWallet, 2, true, VBVTierBoss)
	careerUpgradeSeedStaff(l, cpEmployerWallet, cpStaffReady, "Manager")
	careerUpgradeSeedStaff(l, cpEmployerWallet, cpStaffNotReady, "Clerk")
	// A staff member who meets a gate...
	careerPathSeedPlayer(l, cpStaffReady, CareerPathJustice, VBVTierBoss)
	sr := l.leaderboard[cpStaffReady]
	sr.CareerXP.LessonLevel = 100
	l.leaderboard[cpStaffReady] = sr
	// ...and one who does not.
	careerPathSeedPlayer(l, cpStaffNotReady, CareerPathJustice, VBVTierBoss)

	adv := l.CareerUpgradeViewLocked(cpEmployerWallet)
	if !adv.CanRequestStaff {
		t.Error("a club owner is told they cannot request a staff upgrade")
	}
	if len(adv.Staff) != 2 {
		t.Fatalf("the projection lists %d staff; the roster has 2", len(adv.Staff))
	}
	// DETERMINISTIC ORDER: `Staff` is a map, so an unsorted list reshuffles on every refresh.
	if adv.Staff[0].Wallet >= adv.Staff[1].Wallet {
		t.Errorf("the staff list is not in a stable order: %v",
			[]string{adv.Staff[0].Wallet, adv.Staff[1].Wallet})
	}
	if adv.StaffReadyCount != 1 {
		t.Errorf("staff_ready_count = %d; exactly one staff member meets a gate", adv.StaffReadyCount)
	}
	byWallet := map[string]CareerStaffView{}
	for _, s := range adv.Staff {
		byWallet[s.Wallet] = s
	}
	if s := byWallet[cpStaffReady]; !s.UpgradeReady || s.UnlockCount == 0 {
		t.Errorf("%s meets a gate but the projection says they are not ready: %+v", cpStaffReady, s)
	}
	if s := byWallet[cpStaffNotReady]; s.UpgradeReady || s.UnlockCount != 0 {
		t.Errorf("%s does not meet a gate but the projection says they are ready: %+v", cpStaffNotReady, s)
	}
	if byWallet[cpStaffReady].ClubRole != "Manager" {
		t.Errorf("the club role is not carried through: %q", byWallet[cpStaffReady].ClubRole)
	}

	// A WALLET THAT EMPLOYS NOBODY gets an empty projection and is told so — never another
	// player's staff, and never a bare empty box with no basis.
	other := l.CareerUpgradeViewLocked(cpOutsiderWallet)
	if other.CanRequestStaff || len(other.Staff) != 0 {
		t.Errorf("a wallet with no club is served staff: %+v", other.Staff)
	}
	if other.StaffBasis == "" {
		t.Error("the staff basis is not served, so the client cannot say where staff come from")
	}
}

func TestStaffAndUnlockProjectionsAreStableAcrossReads(t *testing.T) {
	l := careerPathTestLobby(t)
	careerPathSeedOwner(l, cpEmployerWallet, 2, true, VBVTierBoss)
	careerUpgradeSeedStaff(l, cpEmployerWallet, cpStaffReady, "Manager")
	careerUpgradeSeedStaff(l, cpEmployerWallet, cpStaffNotReady, "Clerk")
	careerPathMakeEligible(l, cpEmployerWallet, CareerPathJustice, "Warden")

	first, err := json.Marshal(l.CareerUpgradeViewLocked(cpEmployerWallet))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// 25 reads: a map-iteration leak shows up as a reshuffle (the defect class the bonded-asset
	// chest and the market both had to repair).
	for i := 0; i < 25; i++ {
		again, err := json.Marshal(l.CareerUpgradeViewLocked(cpEmployerWallet))
		if err != nil {
			t.Fatalf("marshal %d: %v", i, err)
		}
		if string(again) != string(first) {
			t.Fatalf("read %d differs from the first read:\n%s\n%s", i, first, again)
		}
	}
}

// ── 3. THE EMPLOYER MAY ASK. ONLY THE EMPLOYEE MAY ACT ────────────────────────────────────────

// careerUpgradeSeedEmployerAndEligibleStaff is the fixture every request test starts from: an
// employer with one staff member who is UNLOCKED for Warden and one who is not.
func careerUpgradeSeedEmployerAndEligibleStaff(l *Lobby) {
	careerPathSeedOwner(l, cpEmployerWallet, 2, true, VBVTierBoss)
	careerUpgradeSeedStaff(l, cpEmployerWallet, cpStaffReady, "Manager")
	careerUpgradeSeedStaff(l, cpEmployerWallet, cpStaffNotReady, "Clerk")
	careerPathMakeEligible(l, cpStaffReady, CareerPathJustice, "Warden")
	careerPathSeedPlayer(l, cpStaffNotReady, CareerPathJustice, VBVTierBoss)
}

func TestStaffUpgradeRequestAsksAndNeverPromotes(t *testing.T) {
	l := careerPathTestLobby(t)
	careerUpgradeSeedEmployerAndEligibleStaff(l)

	before := l.leaderboard[cpStaffReady]
	notice, err := l.RequestStaffCareerUpgradeLocked(cpEmployerWallet, cpStaffReady, "Warden")
	if err != nil {
		t.Fatalf("the employer could not request an unlocked upgrade: %v", err)
	}
	if notice.ClubID == "" || notice.ClubName == "" || RoleKey(notice.Role) != RoleKey("Warden") {
		t.Errorf("the notice does not name the club and role: %+v", notice)
	}
	if notice.Message == "" {
		t.Error("the request carries no message for the staff member")
	}

	// IT ASKS AND NOTHING ELSE. This is the assertion that must never weaken: if a request can
	// promote, then employment has become a way to force a career on somebody.
	after := l.leaderboard[cpStaffReady]
	if len(after.CareerXP.PromotedRoles) != 0 {
		t.Errorf("the request PROMOTED the staff member: %v", after.CareerXP.PromotedRoles)
	}
	if after.JobRole != before.JobRole {
		t.Errorf("the request changed the staff member's JobRole: %q -> %q", before.JobRole, after.JobRole)
	}
	if after.CareerXP.LessonLevel != before.CareerXP.LessonLevel {
		t.Errorf("the request changed the staff member's level: %d -> %d",
			before.CareerXP.LessonLevel, after.CareerXP.LessonLevel)
	}

	// The request is recorded on the EMPLOYER's club, keyed by the folded wallet and role.
	club := l.clubs["club-"+cpEmployerWallet]
	if len(club.StaffUpgradeRequests) != 1 {
		t.Errorf("the request was recorded %d times; want exactly 1", len(club.StaffUpgradeRequests))
	}
	if _, ok := club.StaffUpgradeRequests[staffUpgradeRequestKey(cpStaffReady, "Warden")]; !ok {
		t.Errorf("the request is not keyed by the folded wallet+role: %v", club.StaffUpgradeRequests)
	}

	// THE EMPLOYEE SEES IT, and nobody else does. The employment record is written the way a HIRE
	// writes it (`EmployerClubID`), and the backstop is proven separately underneath.
	st := l.leaderboard[cpStaffReady]
	st.EmployerClubID = club.ID
	l.leaderboard[cpStaffReady] = st
	emp := l.CareerUpgradeViewLocked(cpStaffReady)
	if len(emp.RequestsToMe) != 1 {
		t.Fatalf("the staff member is served %d requests; want 1", len(emp.RequestsToMe))
	}
	if RoleKey(emp.RequestsToMe[0].EmployerWallet) != RoleKey(cpEmployerWallet) {
		t.Errorf("the request does not name the employer: %+v", emp.RequestsToMe[0])
	}
	if other := l.CareerUpgradeViewLocked(cpOutsiderWallet); len(other.RequestsToMe) != 0 {
		t.Errorf("a request addressed to one wallet was served to another: %+v", other.RequestsToMe)
	}

	// THE BACKSTOP. A roster row can outlive a cleared employment field, and a request that exists
	// but never surfaces is a notification that SILENTLY FAILED — so the read falls back to the
	// roster instead of dropping it.
	st = l.leaderboard[cpStaffReady]
	st.EmployerClubID = ""
	l.leaderboard[cpStaffReady] = st
	if emp = l.CareerUpgradeViewLocked(cpStaffReady); len(emp.RequestsToMe) != 1 {
		t.Errorf("a request vanished when the employment field was emptied: %+v", emp.RequestsToMe)
	}

	// THE EMPLOYER'S OWN VIEW shows the request so the panel does not offer it twice.
	adv := l.CareerUpgradeViewLocked(cpEmployerWallet)
	for _, s := range adv.Staff {
		if RoleKey(s.Wallet) == RoleKey(cpStaffReady) && RoleKey(s.RequestedRole) != RoleKey("Warden") {
			t.Errorf("the employer's projection does not show the request: %+v", s)
		}
	}

	// ONE REQUEST PER UNLOCK: a repeat is REPORTED, not re-sent and not duplicated. A nudge that can
	// be repeated on demand is not a notification.
	repeat, err := l.RequestStaffCareerUpgradeLocked(cpEmployerWallet, cpStaffReady, "Warden")
	if err != nil {
		t.Fatalf("a repeat request errored instead of being reported: %v", err)
	}
	if repeat.RequestedAt != notice.RequestedAt {
		t.Errorf("the repeat re-stamped the request (%v -> %v); the first request must stand",
			notice.RequestedAt, repeat.RequestedAt)
	}
	if len(l.clubs["club-"+cpEmployerWallet].StaffUpgradeRequests) != 1 {
		t.Error("the repeat recorded a second request")
	}
}

func TestStaffUpgradeRequestRefusesSevenWaysAndMovesNothing(t *testing.T) {
	l := careerPathTestLobby(t)
	careerUpgradeSeedEmployerAndEligibleStaff(l)
	// A club owned by SOMEBODY ELSE, with its own staff: it must not be reachable by this employer.
	careerPathSeedOwner(l, cpOutsiderWallet, 2, true, VBVTierBoss)
	careerUpgradeSeedStaff(l, cpOutsiderWallet, "0xcpt-rival-staff", "Clerk")
	careerPathMakeEligible(l, "0xcpt-rival-staff", CareerPathJustice, "Warden")

	cases := []struct {
		name   string
		owner  string
		staff  string
		role   string
		reason string
	}{
		{"no staff wallet named", cpEmployerWallet, "", "Warden", "needs the staff wallet"},
		{"the employer is the target", cpEmployerWallet, cpEmployerWallet, "Warden", "take the upgrade yourself"},
		{"the role has no declared gate", cpEmployerWallet, cpStaffReady, "Ghost", "no promotion gate is declared"},
		{"the caller owns no club", "0xcpt-employer-with-nothing", cpStaffReady, "Warden", "own no club"},
		{"the target is not your staff", cpEmployerWallet, "0xcpt-not-my-staff", "Warden", "not on the staff"},
		{"the caller names ANOTHER owner's staff", cpEmployerWallet, "0xcpt-rival-staff", "Warden", "not on the staff"},
		{"the level cap has not opened it", cpEmployerWallet, cpStaffNotReady, "Warden", "not unlocked"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			notice, err := l.RequestStaffCareerUpgradeLocked(c.owner, c.staff, c.role)
			if err == nil {
				t.Fatalf("the request was allowed — an employer could ask for something the gates have not opened: %+v", notice)
			}
			if !strings.Contains(err.Error(), c.reason) {
				t.Errorf("the refusal does not state the reason %q: %v", c.reason, err)
			}
			// NOTHING MOVED: no request recorded anywhere, and no career touched.
			for id, club := range l.clubs {
				if len(club.StaffUpgradeRequests) != 0 {
					t.Errorf("a refused request was recorded on club %s: %v", id, club.StaffUpgradeRequests)
				}
			}
			if st, ok := l.leaderboard[c.staff]; ok && st.CareerXP != nil && len(st.CareerXP.PromotedRoles) != 0 {
				t.Errorf("a refused request promoted %s: %v", c.staff, st.CareerXP.PromotedRoles)
			}
		})
	}

	// THE UNLOCK IS THE EMPLOYEE'S, NOT THE EMPLOYER'S: the same request that is refused for a locked
	// employee succeeds once THEIR gate is met — so the door reads the employee's record, not a flag
	// the employer controls.
	l.leaderboard[cpStaffNotReady].CareerXP.LessonLevel = 100
	l.leaderboard[cpStaffNotReady].CareerXP.RoleXP["Warden"] = 100000
	notice, err := l.RequestStaffCareerUpgradeLocked(cpEmployerWallet, cpStaffNotReady, "Warden")
	if err != nil {
		t.Fatalf("an employer could not ask an unlocked employee: %v", err)
	}
	if RoleKey(notice.StaffWallet) != RoleKey(cpStaffNotReady) {
		t.Errorf("the notice names the wrong wallet: %+v", notice)
	}

	// THE "ALREADY HELD" REFUSAL, proven on its own so a fixture cannot mask it.
	if _, err := l.PromoteCareerLocked(cpStaffNotReady, "Warden"); err != nil {
		t.Fatalf("fixture could not promote the staff member: %v", err)
	}
	if _, err := l.RequestStaffCareerUpgradeLocked(cpEmployerWallet, cpStaffNotReady, "Warden"); err == nil ||
		!strings.Contains(err.Error(), "already holds") {
		t.Errorf("requesting a role the staff member already holds was not refused: %v", err)
	}
}

// ── 4. NOTIFIED, NOT FORCED ───────────────────────────────────────────────────────────────────

func TestUnlockNoticeFiresOnceAndReArmsAfterAPromotion(t *testing.T) {
	l := careerPathTestLobby(t)
	careerPathMakeEligible(l, cpJusticeWallet, CareerPathJustice, "Warden")

	first := l.CareerUnlockNoticesLocked(cpJusticeWallet)
	if len(first) != 1 {
		t.Fatalf("an unlocked career produced %d notices; want 1", len(first))
	}
	if RoleKey(first[0].Role) != RoleKey("Warden") || first[0].Wallet != cpJusticeWallet {
		t.Errorf("the notice names the wrong role/wallet: %+v", first[0])
	}
	if first[0].Message == "" || first[0].UnlockBasis == "" {
		t.Errorf("the notice states no basis: %+v", first[0])
	}
	// It must say the upgrade is an OFFER: the notice is the whole of "notified, not forced".
	if !strings.Contains(strings.ToLower(first[0].Message), "yours to take") {
		t.Errorf("the notice does not say the upgrade is the player's to take: %q", first[0].Message)
	}

	// ONCE. A second call must not re-notify: a notice that repeats on every sample is a nag.
	if again := l.CareerUnlockNoticesLocked(cpJusticeWallet); len(again) != 0 {
		t.Errorf("the notice fired twice: %+v", again)
	}

	// A DEMOTION CYCLE RE-ARMS IT. Promoting the role clears the record, so a career lost and
	// re-earned notifies again instead of being silent forever.
	if _, err := l.PromoteCareerLocked(cpJusticeWallet, "Warden"); err != nil {
		t.Fatalf("promote: %v", err)
	}
	if got := l.CareerUnlockNoticesLocked(cpJusticeWallet); len(got) != 0 {
		t.Errorf("a HELD career was announced as unlocked: %+v", got)
	}
	// The demotion path removes the role from PromotedRoles — that IS the demotion.
	l.leaderboard[cpJusticeWallet].CareerXP.PromotedRoles = nil
	reArmed := l.CareerUnlockNoticesLocked(cpJusticeWallet)
	if len(reArmed) != 1 || RoleKey(reArmed[0].Role) != RoleKey("Warden") {
		t.Errorf("the notice did not re-arm after the career was lost and unlocked again: %+v", reArmed)
	}

	// A wallet with no engine record is never notified about anything.
	if got := l.CareerUnlockNoticesLocked(cpOutsiderWallet); len(got) != 0 {
		t.Errorf("a wallet with no record was notified: %+v", got)
	}
}

// ── 5. THE HTTP BOUNDARY ──────────────────────────────────────────────────────────────────────

func TestCareerStaffUpgradeRequestHTTPBoundary(t *testing.T) {
	l := careerPathTestLobby(t)
	careerUpgradeSeedEmployerAndEligibleStaff(l)

	// ANONYMOUS: refused BY THE DOOR, before any lookup.
	anon := httptest.NewRequest(http.MethodPost, "/api/career/staff/request",
		strings.NewReader(`{"staff_wallet":"0x1","role":"Warden"}`))
	rec := httptest.NewRecorder()
	l.handleCareerStaffUpgradeRequest(rec, anon)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("an anonymous request got %d; want 401", rec.Code)
	}

	// THE VERB.
	get := httptest.NewRequest(http.MethodGet, "/api/career/staff/request?wallet="+cpEmployerWallet, nil)
	rec = httptest.NewRecorder()
	l.handleCareerStaffUpgradeRequest(rec, get)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET got %d; want 405", rec.Code)
	}

	// THE CLIENT OWNS NOTHING HERE: a body that tries to hand itself a role grant, an unlock or a
	// civil rank cannot even be PARSED (DisallowUnknownFields), so it fails before any lookup.
	for _, body := range []string{
		`{"staff_wallet":"0x1","role":"Warden","promoted_roles":["Judge"]}`,
		`{"staff_wallet":"0x1","role":"Warden","unlocked":true}`,
		`{"staff_wallet":"0x1","role":"Warden","civil_tier":"governor"}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/career/staff/request?wallet="+cpEmployerWallet,
			strings.NewReader(body))
		rec = httptest.NewRecorder()
		l.handleCareerStaffUpgradeRequest(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("a client-owned field was accepted (%d): %s", rec.Code, body)
		}
	}

	// MISSING FIELDS.
	req := httptest.NewRequest(http.MethodPost, "/api/career/staff/request?wallet="+cpEmployerWallet,
		strings.NewReader(`{"role":"Warden"}`))
	rec = httptest.NewRecorder()
	l.handleCareerStaffUpgradeRequest(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("a body with no staff wallet got %d; want 400", rec.Code)
	}

	// A CALLER WHO EMPLOYS NOBODY: refused with the reason, and NOTHING is written.
	req = httptest.NewRequest(http.MethodPost, "/api/career/staff/request?wallet="+cpOutsiderWallet,
		strings.NewReader(`{"staff_wallet":"`+cpStaffReady+`","role":"Warden"}`))
	rec = httptest.NewRecorder()
	l.handleCareerStaffUpgradeRequest(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("a wallet with no club got %d; want 403", rec.Code)
	}
	var refusal struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &refusal); err != nil {
		t.Fatalf("the refusal is not JSON: %v", err)
	}
	if !strings.Contains(refusal.Error, "own no club") {
		t.Errorf("the refusal does not name the reason: %q", refusal.Error)
	}
	if len(l.clubs["club-"+cpEmployerWallet].StaffUpgradeRequests) != 0 {
		t.Error("a REFUSED request was recorded")
	}

	// THE ALLOWED PATH, with the employee's career proven untouched on the way out.
	req = httptest.NewRequest(http.MethodPost, "/api/career/staff/request?wallet="+cpEmployerWallet,
		strings.NewReader(`{"staff_wallet":"`+cpStaffReady+`","role":"Warden"}`))
	rec = httptest.NewRecorder()
	l.handleCareerStaffUpgradeRequest(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("the employer's request got %d: %s", rec.Code, rec.Body.String())
	}
	var allowed struct {
		Success  bool               `json:"success"`
		Upgrades *CareerUpgradeView `json:"upgrades"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &allowed); err != nil {
		t.Fatalf("the response is not JSON: %v", err)
	}
	if !allowed.Success || allowed.Upgrades == nil {
		t.Errorf("the response does not carry the advisory it was asked with: %s", rec.Body.String())
	}
	if len(l.leaderboard[cpStaffReady].CareerXP.PromotedRoles) != 0 {
		t.Errorf("the HTTP door PROMOTED the staff member: %v",
			l.leaderboard[cpStaffReady].CareerXP.PromotedRoles)
	}
}






