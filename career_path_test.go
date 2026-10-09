//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ============================================================================
// CAREER PATH · CIVIL RANK · PROMOTION & DEMOTION — the load-bearing assertions
// ============================================================================
//
// WHAT THESE TESTS GUARD. Three capabilities had no owner before their subject existed:
//
//  1. The career PATH did not exist in the server at all — `Public/js/career_tree.js` invented a
//     `faction: JUSTICE|UNDERWORLD|HYBRID` taxonomy in the CLIENT. The vocabulary, its fold and its
//     aliases are pinned here, so an invented spelling cannot reintroduce a fourth taxonomy.
//
//  2. THE TWO MATRICES MUST AGREE. The career-pair matrix (`rivalPairTable`) already brackets
//     careers as enemies and allies, and `careerPathByRole` reads the two sides OFF it. A future
//     pair that contradicts those assignments would drift the two matrices apart and make every
//     path-based bonus arbitrary, so the agreement is pinned mechanically.
//
//  3. PROMOTION AND DEMOTION had no writer: `PromotedRoles` was written by NOTHING (so
//     `CareerHasRole` was permanently false) and demotion was warning-only. The lifecycle
//     (warn -> grace -> demote, and WITHDRAW on recovery) is pinned end to end, including the
//     "nothing is taken on a refusal" and "nothing is client-declarable" properties.
// ============================================================================

const (
	cpJusticeWallet  = "0xcpt-justice"
	cpCriminalWallet = "0xcpt-criminal"
	cpNeutralWallet  = "0xcpt-neutral"
)

// careerPathTestLobby builds the minimum the career-path domain needs. The audit log is redirected
// into a temp dir so a test never writes into the repository.
func careerPathTestLobby(t *testing.T) *Lobby {
	t.Helper()
	return &Lobby{
		DataDir:        t.TempDir(),
		leaderboard:    map[string]PlayerStats{},
		clubs:          map[string]*Club{},
		playerBalances: map[string]uint64{},
		wallets:        map[string]string{},
	}
}

// careerPathSeedOwner gives a wallet a club with the requested territories, optionally with its
// REGION OPENED (the `RegionName` marker `HandleOpenRegionalManager` is the only writer of), and a
// balance. Everything a civil rank is derived from comes through here rather than being set.
func careerPathSeedOwner(l *Lobby, wallet string, territories int, regionOpened bool, balanceMicro uint64) {
	terr := make([]string, 0, territories)
	for i := 0; i < territories; i++ {
		terr = append(terr, fmt.Sprintf("%s-t%d", wallet, i))
	}
	club := &Club{
		ID:          "club-" + wallet,
		Name:        "Club " + wallet,
		OwnerWallet: wallet,
		Territories: terr,
	}
	if regionOpened {
		club.RegionName = "Governor"
	}
	l.clubs[club.ID] = club
	l.playerBalances[wallet] = balanceMicro
}

// careerPathSeedPlayer adds the leaderboard record (separate from ownership, because a wallet can
// hold a career without owning anything and vice versa).
func careerPathSeedPlayer(l *Lobby, wallet, path string, balanceMicro uint64) {
	l.leaderboard[wallet] = PlayerStats{
		Wallet: wallet,
		CareerXP: &CareerXP{
			RoleXP:            map[string]uint64{},
			Path:              path,
			AvgSustainedMicro: balanceMicro,
		},
	}
}

// ── 1. THE PATH VOCABULARY ───────────────────────────────────────────────────

func TestPathKeyFoldsEveryDeclaredSpellingOntoThreePaths(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		// The three ids, in every casing a client or a doc might use.
		{"justice", CareerPathJustice},
		{"Justice", CareerPathJustice},
		{"JUSTICE", CareerPathJustice},
		{"  justice  ", CareerPathJustice},
		{"criminal", CareerPathCriminal},
		{"CRIMINAL", CareerPathCriminal},
		{"neutral", CareerPathNeutral},
		{"Neutral", CareerPathNeutral},
		// The aliases, each declared WITH its source in career_path.go.
		{"law", CareerPathJustice},         // item families speak "law" for the justice side
		{"Underworld", CareerPathCriminal}, // career_tree.js's faction label for the criminal side
		{"UNDERWORLD", CareerPathCriminal},
		{"Hybrid", CareerPathNeutral}, // a pathway serving both sides IS the neutral position
		{"HYBRID", CareerPathNeutral},
	}
	for _, c := range cases {
		if got := PathKey(c.in); got != c.want {
			t.Errorf("PathKey(%q) = %q; want %q", c.in, got, c.want)
		}
		if !IsCareerPath(c.in) {
			t.Errorf("IsCareerPath(%q) = false; want true (it folds to %q)", c.in, c.want)
		}
	}

	// NEGATIVE CONTROLS: a spelling that is NOT one of the three must not fold onto one. Without
	// these, an aliasing bug would silently turn an unknown club type into a career path.
	for _, bad := range []string{"", "republic", "justices", "criminology", "neuter", "underworlds", "hybrids"} {
		if IsCareerPath(bad) {
			t.Errorf("IsCareerPath(%q) = true (folds to %q); want false", bad, PathKey(bad))
		}
		if PathKey(bad) != foldRole(bad) {
			t.Errorf("PathKey(%q) was aliased; an undeclared spelling must fold to itself", bad)
		}
	}
}

// ── 2. THE TWO MATRICES CANNOT DRIFT APART ───────────────────────────────────

// rivalMatrixEnemyPairs names, with its evidence, every pair the career-pair matrix treats as
// ANTAGONISTIC. Six are priced negatively by `GetRivalXPDelta`; the seventh (P2-D10) is declared
// antagonistic in the pair table's own comment while the XP switch prices it 0, so it is listed
// here explicitly rather than being classified by a delta that would mislead.
var rivalMatrixEnemyPairs = map[string]bool{
	"BountyHunter↔Kidnapper":       true, // -15
	"ForensicAnalyst↔Gossip":       true, // -10
	"TaxAuditor↔Launderer":         true, // -10
	"Warden↔HeistPlanner":          true, // -10
	"SectorPeacekeeper↔Smuggler":   true, // -10
	"IntelAgent↔ArcNetOperative":   true, // -10
	"MutationLogAuditor↔Kidnapper": true, // P2-D10 ("antagonistic — tracks their genetic tampering"), delta 0
}

func TestCareerPathAgreesWithTheRivalMatrix(t *testing.T) {
	if len(rivalPairTable) == 0 {
		t.Fatal("rivalPairTable is empty; this test would pass vacuously")
	}
	seenEnemy := 0
	for _, p := range rivalPairTable {
		a, b := CareerPathOfRole(p.A), CareerPathOfRole(p.B)
		if a == "" || b == "" {
			t.Errorf("pair %s (%s vs %s): a career has NO declared path (a=%q b=%q) — every career in the "+
				"rival matrix must be placed on a side, or the pair cannot be classified", p.Name, p.A, p.B, a, b)
			continue
		}
		if rivalMatrixEnemyPairs[p.Name] {
			seenEnemy++
			if a == b {
				t.Errorf("ENEMY pair %s: both sides are on the %s path — an enemy pair must CROSS the two paths",
					p.Name, a)
			}
			continue
		}
		// Anything not declared antagonistic is an ally pair, and an ally shares a side.
		if a != b {
			t.Errorf("ALLY pair %s: %s is %s while %s is %s — an ally pair must SHARE a path",
				p.Name, p.A, a, p.B, b)
		}
	}
	if seenEnemy != len(rivalMatrixEnemyPairs) {
		t.Errorf("matched %d enemy pairs but %d are declared; the declared list and the pair table disagree",
			seenEnemy, len(rivalMatrixEnemyPairs))
	}
}

// ── 3. THE PATH MATRIX ───────────────────────────────────────────────────────

func TestCareerPathRivalryResolvesDirectInterpretedAndShared(t *testing.T) {
	cases := []struct {
		a, b    string
		want    string
		wantBps int
		why     string
	}{
		{CareerPathJustice, CareerPathCriminal, PathRivalryDirect, PathDirectRivalryBonusBps, "the one direct rivalry"},
		{CareerPathCriminal, CareerPathJustice, PathRivalryDirect, PathDirectRivalryBonusBps, "symmetric"},
		{CareerPathJustice, "law", PathRivalryShared, 0, "an alias of the same side is the same side"},
		{CareerPathCriminal, "underworld", PathRivalryShared, 0, "same"},
		{CareerPathNeutral, CareerPathNeutral, PathRivalryInterpreted, 0, "two neutrals read each other"},
		{CareerPathNeutral, CareerPathJustice, PathRivalryInterpreted, 0, "a neutral holds no direct enemy"},
		{CareerPathCriminal, CareerPathNeutral, PathRivalryInterpreted, 0, "either way round"},
		{"", CareerPathJustice, PathRivalryNone, 0, "an unset path resolves to none"},
		{"nonsense", CareerPathCriminal, PathRivalryNone, 0, "an unknown path is not a path"},
	}
	for _, c := range cases {
		if got := CareerPathRivalry(c.a, c.b); got != c.want {
			t.Errorf("CareerPathRivalry(%q,%q) = %q; want %q (%s)", c.a, c.b, got, c.want, c.why)
		}
		if got := PathRivalryBonusBps(c.a, c.b); got != c.wantBps {
			t.Errorf("PathRivalryBonusBps(%q,%q) = %d; want %d (%s)", c.a, c.b, got, c.wantBps, c.why)
		}
	}
}

func TestPathAllowsRoleGivesNeutralBreadthAndSidedRestriction(t *testing.T) {
	cases := []struct {
		path, role string
		want       bool
	}{
		{CareerPathJustice, "Bounty Hunter", true},
		{CareerPathJustice, "IntelAgent", true}, // by ROLE: the table declares "Intel-Agent"
		{CareerPathJustice, "Kidnapper", false}, // a criminal career
		{CareerPathCriminal, "Kidnapper", true},
		{CareerPathCriminal, "Bounty Hunter", false},
		{CareerPathNeutral, "Bounty Hunter", true}, // breadth: either side
		{CareerPathNeutral, "Kidnapper", true},
		{CareerPathNeutral, "NotACareer", false}, // an unknown career is never granted
		{"", "Bounty Hunter", false},             // no path, no career
		{"nonsense", "Bounty Hunter", false},
	}
	for _, c := range cases {
		if got := PathAllowsRole(c.path, c.role); got != c.want {
			t.Errorf("PathAllowsRole(%q, %q) = %v; want %v", c.path, c.role, got, c.want)
		}
	}
}

// ── 4. THE CIVIL RANK IS DERIVED, NEVER DECLARED ─────────────────────────────

func TestCivilStandingIsDerivedFromOwnedClubsOnly(t *testing.T) {
	l := careerPathTestLobby(t)

	// A wallet with nothing is a USER.
	if got := l.CivilStandingForWalletLocked("0xempty"); got.Tier != CivilTierUser || got.Shops != 0 || got.Territories != 0 {
		t.Errorf("an empty wallet = %+v; want user/0/0", got)
	}

	// A SHOP with no territory is still a user — but the shop is REPORTED, because "owns a shop and
	// nothing else" is a different state from "owns nothing" and the UI must be able to say which.
	careerPathSeedOwner(l, "0xshoponly", 0, false, 0)
	got := l.CivilStandingForWalletLocked("0xshoponly")
	if got.Tier != CivilTierUser || got.Shops != 1 || got.Territories != 0 {
		t.Errorf("a shop with no territory = %+v; want user with shops=1", got)
	}

	// One territory is a MANAGER.
	careerPathSeedOwner(l, "0xmanager", 1, false, 0)
	if got := l.CivilStandingForWalletLocked("0xmanager"); got.Tier != CivilTierManager || got.Territories != 1 {
		t.Errorf("one territory = %+v; want manager/1", got)
	}

	// Two territories WITHOUT the region opened is still a MANAGER: opening the region is a
	// separate act, and the two must not be conflated.
	careerPathSeedOwner(l, "0xtwo", 2, false, 0)
	if got := l.CivilStandingForWalletLocked("0xtwo"); got.Tier != CivilTierManager || got.RegionOpened {
		t.Errorf("two territories with no region = %+v; want manager, region NOT opened", got)
	}

	// Opening the region is a GOVERNOR.
	careerPathSeedOwner(l, "0xgov", 2, true, 0)
	if got := l.CivilStandingForWalletLocked("0xgov"); got.Tier != CivilTierGovernor || !got.RegionOpened {
		t.Errorf("region opened = %+v; want governor", got)
	}

	// AN ALLIANCE COUNTS, because the engine's own `IsClubRegionalLocked` counts an allied club's
	// territories — the rank must agree with the gate the rest of the engine already uses.
	ally := &Club{ID: "ally-club", OwnerWallet: "0xally-owner", Territories: []string{"a1", "a2"}}
	l.clubs[ally.ID] = ally
	main := &Club{ID: "main-club", OwnerWallet: "0xally-owner", Territories: []string{"m1"}, AlliedClubID: ally.ID}
	l.clubs[main.ID] = main
	if got := l.CivilStandingForWalletLocked("0xally-owner"); got.Territories != 3 {
		t.Errorf("alliance-aware territory count = %d; want 3 (1 own + 2 allied)", got.Territories)
	}
}

func TestCivilStandingResolvesTheStoredWalletSpelling(t *testing.T) {
	l := careerPathTestLobby(t)
	careerPathSeedOwner(l, "0xMixedCase", 2, true, 0)
	// A caller asking in another casing must find the SAME rank, or a wallet would be told it has
	// no organisation while its own club sits in the map.
	if got := l.CivilStandingForWalletLocked("0xmixedcase"); got.Tier != CivilTierGovernor {
		t.Errorf("case-insensitive lookup = %+v; want governor", got)
	}
}

// ── 5. THE CHOICE GATE ───────────────────────────────────────────────────────

func TestCareerPathChoiceUnlocksOnlyWithAnOpenedRegionAtTwoTerritories(t *testing.T) {
	l := careerPathTestLobby(t)

	// Not enough territories: refused, and the REFUSAL NAMES the requirement.
	careerPathSeedOwner(l, cpJusticeWallet, 1, false, 0)
	careerPathSeedPlayer(l, cpJusticeWallet, "", 0)
	elig := l.CareerPathEligibilityForWalletLocked(cpJusticeWallet)
	if elig.Eligible {
		t.Fatalf("eligible with 1 territory: %+v", elig)
	}
	if !strings.Contains(elig.Reason, "2 territories") {
		t.Errorf("refusal reason %q must name the requirement", elig.Reason)
	}

	// Two territories but no region OPENED: still refused, for a DIFFERENT stated reason.
	careerPathSeedOwner(l, "0xtwo-no-region", 2, false, 0)
	careerPathSeedPlayer(l, "0xtwo-no-region", "", 0)
	elig = l.CareerPathEligibilityForWalletLocked("0xtwo-no-region")
	if elig.Eligible {
		t.Fatalf("eligible without an opened region: %+v", elig)
	}
	if !strings.Contains(elig.Reason, "region ability has not been opened") {
		t.Errorf("refusal reason %q must name the region step", elig.Reason)
	}

	// The full gate: 2 territories AND the region opened.
	careerPathSeedOwner(l, "0xready", 2, true, 0)
	careerPathSeedPlayer(l, "0xready", "", 0)
	elig = l.CareerPathEligibilityForWalletLocked("0xready")
	if !elig.Eligible {
		t.Fatalf("not eligible at 2 territories with the region opened: %+v", elig)
	}

	out, err := l.ChooseCareerPathLocked("0xready", CareerPathJustice)
	if err != nil {
		t.Fatalf("ChooseCareerPathLocked: %v", err)
	}
	if out.CurrentPath != CareerPathJustice {
		t.Errorf("chosen path = %q; want %q", out.CurrentPath, CareerPathJustice)
	}
	if got := l.leaderboard["0xready"].CareerXP.Path; got != CareerPathJustice {
		t.Errorf("stored path = %q; want %q", got, CareerPathJustice)
	}
	if l.leaderboard["0xready"].CareerXP.PathChosenAt.IsZero() {
		t.Error("PathChosenAt was not stamped")
	}

	// THE CHOICE IS MADE ONCE. A second call — even for the same path — is refused, because an
	// alignment that can be flipped on demand carries no weight in any of the three matrices.
	before := l.leaderboard["0xready"].CareerXP.Path
	if _, err := l.ChooseCareerPathLocked("0xready", CareerPathCriminal); err == nil {
		t.Error("a second path choice was ACCEPTED; the choice must be made once")
	}
	if got := l.leaderboard["0xready"].CareerXP.Path; got != before {
		t.Errorf("a refused second choice changed the path: %q -> %q", before, got)
	}
}

func TestCareerPathChoiceRefusesAnUnknownPathAndWritesNothing(t *testing.T) {
	l := careerPathTestLobby(t)
	careerPathSeedOwner(l, "0xready", 2, true, 0)
	careerPathSeedPlayer(l, "0xready", "", 0)

	for _, bad := range []string{"", "lawful", "republic", "justices"} {
		if _, err := l.ChooseCareerPathLocked("0xready", bad); err == nil {
			t.Errorf("ChooseCareerPathLocked(%q) was ACCEPTED; want a refusal", bad)
		}
		if got := l.leaderboard["0xready"].CareerXP.Path; got != "" {
			t.Fatalf("a refused choice (%q) wrote %q into the record", bad, got)
		}
	}
}

// ── 6. PROMOTION ─────────────────────────────────────────────────────────────

// careerPathMakeEligible gives a wallet everything a promotion can be gated on: a path, an opened
// region with 2 territories, the top $VBV rung and full XP. Individual tests then walk ONE gate
// back at a time, so a refusal is always attributable to exactly one missing piece.
func careerPathMakeEligible(l *Lobby, wallet, path, role string) {
	careerPathSeedOwner(l, wallet, 2, true, VBVTierBoss)
	careerPathSeedPlayer(l, wallet, path, VBVTierBoss)
	st := l.leaderboard[wallet]
	st.CareerXP.LessonLevel = 100
	st.CareerXP.RoleXP[role] = 100000
	l.leaderboard[wallet] = st
}

func TestCareerPromotionRefusesUntilEveryGateIsMet(t *testing.T) {
	l := careerPathTestLobby(t)
	careerPathMakeEligible(l, cpJusticeWallet, CareerPathJustice, "Warden")
	cxp := l.leaderboard[cpJusticeWallet].CareerXP

	// A. NO PATH: the first thing that must exist.
	cxp.Path = ""
	if got := l.CareerPromotionRequirementsForWalletLocked(cpJusticeWallet, "Warden"); got.Eligible || !strings.Contains(got.Missing, "career path") {
		t.Errorf("promotion with no path = %+v; want a refusal naming the path", got)
	}
	cxp.Path = CareerPathJustice

	// B. THE WRONG PATH cannot be promoted at all (path only allows a career it belongs to).
	careerPathMakeEligible(l, cpCriminalWallet, CareerPathCriminal, "Warden")
	if got := l.CareerPromotionRequirementsForWalletLocked(cpCriminalWallet, "Warden"); got.PathAllows {
		t.Errorf("a criminal-path player can promote a justice career: %+v", got)
	}

	// C. THE LEVEL CAP is the primary gate.
	cxp.LessonLevel = 9
	got := l.CareerPromotionRequirementsForWalletLocked(cpJusticeWallet, "Warden")
	if got.Eligible || !strings.Contains(got.Missing, "reach level 10") {
		t.Errorf("below the level cap = %+v; want a refusal naming level 10", got)
	}
	cxp.LessonLevel = 10

	// D. THE ROLE TIER: Warden needs tier 1, which a player with no XP already has — but a gate
	//    that required more would name it, so this asserts the CURRENT rung is satisfied.
	if got := l.CareerPromotionRequirementsForWalletLocked(cpJusticeWallet, "Warden"); got.RoleTier < got.MinRoleTier {
		t.Errorf("tier %d is below the declared minimum %d with 100000 XP", got.RoleTier, got.MinRoleTier)
	}

	// E. THE SUSTAINED BALANCE, with the EXACT shortfall reported.
	cxp.AvgSustainedMicro = 0
	got = l.CareerPromotionRequirementsForWalletLocked(cpJusticeWallet, "Warden")
	if got.Eligible {
		t.Fatalf("promoted with no sustained balance: %+v", got)
	}
	if got.ShortfallMicro != got.MinSustainedMicro {
		t.Errorf("shortfall = %d; want the full requirement %d when holding none", got.ShortfallMicro, got.MinSustainedMicro)
	}
	if !strings.Contains(got.Missing, "short") {
		t.Errorf("refusal %q must state the shortfall", got.Missing)
	}

	// F. EVERYTHING MET -> promoted, and the grant lands in the list `CareerHasRole` reads.
	cxp.AvgSustainedMicro = VBVTierBoss
	req, err := l.PromoteCareerLocked(cpJusticeWallet, "Warden")
	if err != nil {
		t.Fatalf("PromoteCareerLocked with every gate met: %v", err)
	}
	if !req.Promoted {
		t.Error("the promotion did not report Promoted")
	}
	stored := l.leaderboard[cpJusticeWallet]
	if !CareerHasRole(stored.CareerXP, "Warden") {
		t.Errorf("PromotedRoles = %v; want it to contain Warden (this list had NO writer before)", stored.CareerXP.PromotedRoles)
	}
	if stored.JobRole != "Warden" {
		t.Errorf("JobRole = %q; want Warden (the stored role must follow the promotion)", stored.JobRole)
	}
	if stored.CareerXP.LastPromotionAt.IsZero() {
		t.Error("LastPromotionAt was not stamped")
	}

	// G. IDEMPOTENT: a second call changes nothing and does not duplicate the entry.
	before := len(stored.CareerXP.PromotedRoles)
	if _, err := l.PromoteCareerLocked(cpJusticeWallet, "Warden"); err != nil {
		t.Errorf("a repeat promotion errored: %v", err)
	}
	if got := len(l.leaderboard[cpJusticeWallet].CareerXP.PromotedRoles); got != before {
		t.Errorf("PromotedRoles grew from %d to %d on a repeat call", before, got)
	}
}

func TestCareerPromotionRefusesAnUndeclaredCareer(t *testing.T) {
	l := careerPathTestLobby(t)
	careerPathMakeEligible(l, cpJusticeWallet, CareerPathJustice, "NotADeclaredCareer")

	got := l.CareerPromotionRequirementsForWalletLocked(cpJusticeWallet, "NotADeclaredCareer")
	if got.GateDeclared {
		t.Fatalf("an undeclared career reported a gate: %+v", got)
	}
	if got.Eligible {
		t.Fatal("an undeclared career was reported eligible")
	}
	if _, err := l.PromoteCareerLocked(cpJusticeWallet, "NotADeclaredCareer"); err == nil {
		t.Error("an undeclared career was PROMOTED; every gate must fail closed")
	}
	if len(l.leaderboard[cpJusticeWallet].CareerXP.PromotedRoles) != 0 {
		t.Error("a refused promotion wrote into PromotedRoles")
	}
}

// TestCareerPromotionRequiresTheCivilRankTheRoleDemands pins the operator's worker-society gate:
// a role whose authority IS a region (Judge, declared `governor`) cannot be held by a player with
// no organisation behind them — and the refusal names the ACTION that unlocks it.
func TestCareerPromotionRequiresTheCivilRankTheRoleDemands(t *testing.T) {
	l := careerPathTestLobby(t)

	// A MANAGER with every other gate satisfied.
	careerPathSeedPlayer(l, cpCriminalWallet, CareerPathCriminal, VBVTierBoss)
	st := l.leaderboard[cpCriminalWallet]
	st.CareerXP.LessonLevel = 100
	st.CareerXP.RoleXP["Judge"] = 100000
	l.leaderboard[cpCriminalWallet] = st
	careerPathSeedOwner(l, cpCriminalWallet, 1, false, VBVTierBoss) // ONE territory => manager

	if got := l.CivilStandingForWalletLocked(cpCriminalWallet); got.Tier != CivilTierManager {
		t.Fatalf("fixture is not a manager: %+v", got)
	}
	req := l.CareerPromotionRequirementsForWalletLocked(cpCriminalWallet, "Judge")
	if req.MinCivilTier != CivilTierGovernor {
		t.Fatalf("Judge's declared civil gate = %q; the fixture assumes governor", req.MinCivilTier)
	}
	if req.Eligible {
		t.Fatalf("a manager was eligible for a governor-gated career: %+v", req)
	}
	if !strings.Contains(req.Missing, CivilTierGovernor) || !strings.Contains(req.Missing, "OPEN ITS REGION") {
		t.Errorf("refusal %q must name the rank AND the action that unlocks it", req.Missing)
	}
	if _, err := l.PromoteCareerLocked(cpCriminalWallet, "Judge"); err == nil {
		t.Error("a governor-gated career was promoted from the manager rank")
	}

	// The same player, having opened the region, passes the civil gate.
	l.clubs["club-"+cpCriminalWallet].RegionName = "Governor"
	if got := l.CivilStandingForWalletLocked(cpCriminalWallet); got.Tier != CivilTierGovernor {
		t.Fatalf("opening the region did not raise the rank: %+v", got)
	}
	if _, err := l.PromoteCareerLocked(cpCriminalWallet, "Judge"); err != nil {
		t.Errorf("a governor with every other gate met was refused: %v", err)
	}
	if !CareerHasRole(l.leaderboard[cpCriminalWallet].CareerXP, "Judge") {
		t.Error("Judge was not granted after the region was opened")
	}
}

// TestEveryDeclaredRoleTierIsReachable is the guard against an UNEARNABLE career. `getTierFor`
// returns 1..4; a gate declaring 5 (or 0) would make that career impossible to promote with no
// error anywhere. This walks EVERY declared gate and proves its exact thresholds can be met.
func TestEveryDeclaredRoleTierIsReachable(t *testing.T) {
	if len(careerRoleGates) == 0 {
		t.Fatal("careerRoleGates is empty; this test would pass vacuously")
	}
	for _, g := range careerRoleGates {
		if g.MinRoleTier < 1 || g.MinRoleTier > CareerRoleTierMax {
			t.Errorf("gate %q requires role tier %d; getTierFor only returns 1..%d, so it can never be met",
				g.Role, g.MinRoleTier, CareerRoleTierMax)
		}
		// Build a player at EXACTLY the declared thresholds and require eligibility. The XP is
		// derived from the tier the gate demands, so the assertion tracks `getTierFor`'s bands
		// rather than a number copied beside them.
		xpForTier := uint64(0)
		switch g.MinRoleTier {
		case 2:
			xpForTier = uint64(CareerTierApprentice * 100)
		case 3:
			xpForTier = uint64(CareerTierExpert * 100)
		case 4:
			xpForTier = uint64(CareerTierMaster * 100)
		}

		l := careerPathTestLobby(t)
		// The civil rank the gate demands (governor = region opened, manager = 1 territory).
		switch g.MinCivilTier {
		case CivilTierGovernor:
			careerPathSeedOwner(l, cpNeutralWallet, 2, true, g.MinSustainedMicro)
		case CivilTierManager:
			careerPathSeedOwner(l, cpNeutralWallet, 1, false, g.MinSustainedMicro)
		default:
			careerPathSeedOwner(l, cpNeutralWallet, 0, false, g.MinSustainedMicro)
		}
		// A NEUTRAL path may hold either side's careers, which keeps this test about the GATE and
		// not about the path (the path restriction has its own test).
		careerPathSeedPlayer(l, cpNeutralWallet, CareerPathNeutral, g.MinSustainedMicro)
		st := l.leaderboard[cpNeutralWallet]
		st.CareerXP.LessonLevel = g.MinLessonLevel
		if xpForTier > 0 {
			st.CareerXP.RoleXP[g.Role] = xpForTier
		}
		l.leaderboard[cpNeutralWallet] = st

		got := l.CareerPromotionRequirementsForWalletLocked(cpNeutralWallet, g.Role)
		if !got.Eligible {
			t.Errorf("gate %q is NOT satisfiable at its own declared thresholds: level %d of %d, tier %d of %d, "+
				"sustained %d of %d, civil %s of %s — missing: %s",
				g.Role, got.LessonLevel, got.MinLessonLevel, got.RoleTier, got.MinRoleTier,
				got.SustainedMicro, got.MinSustainedMicro, got.CivilTier, got.MinCivilTier, got.Missing)
		}
	}
}

// ── 7. DEMOTION — the $VBV drain that takes the career with it ───────────────

// careerPathPromotedJudge produces a wallet that HOLDS a governor-gated career, which is the
// clearest case to demote: the requirement is the top $VBV rung, so "cashing out" is unambiguous.
func careerPathPromotedJudge(t *testing.T, l *Lobby, wallet string) {
	t.Helper()
	careerPathMakeEligible(l, wallet, CareerPathCriminal, "Judge")
	if _, err := l.PromoteCareerLocked(wallet, "Judge"); err != nil {
		t.Fatalf("fixture could not promote Judge: %v", err)
	}
	if !CareerHasRole(l.leaderboard[wallet].CareerXP, "Judge") {
		t.Fatal("fixture did not land the promotion")
	}
}

func TestCareerDemotionWarnsFirstWithTheExactShortfall(t *testing.T) {
	l := careerPathTestLobby(t)
	careerPathPromotedJudge(t, l, cpCriminalWallet)

	// The balance drains (a cash-out). Nothing is taken yet: the player is WARNED.
	l.leaderboard[cpCriminalWallet].CareerXP.AvgSustainedMicro = 0
	events := l.EvaluateCareerStandingLocked(cpCriminalWallet)
	if len(events) != 1 || events[0].Kind != "warning" {
		t.Fatalf("a drained balance produced %+v; want exactly one warning", events)
	}
	ev := events[0]
	if ev.ShortfallMicro != VBVTierBoss {
		t.Errorf("shortfall = %d; want the full requirement %d when holding none", ev.ShortfallMicro, VBVTierBoss)
	}
	if ev.GraceEndsAt.IsZero() {
		t.Error("the warning must carry the deadline to earn the balance back")
	}
	if !strings.Contains(ev.Message, "demoted") {
		t.Errorf("warning %q must state the consequence", ev.Message)
	}
	cxp := l.leaderboard[cpCriminalWallet].CareerXP
	if !CareerHasRole(cxp, "Judge") {
		t.Error("the warning ALREADY removed the career; a warning must take nothing")
	}
	if cxp.DemotionWarningRole != "Judge" || cxp.DemotionWarningAt.IsZero() {
		t.Errorf("warning not recorded on the record: role=%q at=%v", cxp.DemotionWarningRole, cxp.DemotionWarningAt)
	}
	if len(cxp.Demotions) != 0 {
		t.Error("a warning appended a demotion record")
	}

	// INSIDE the grace period a second evaluation still only warns, and does not re-stamp the clock
	// (re-stamping would move the deadline every day, so the career could never be taken away).
	stamped := cxp.DemotionWarningAt
	events = l.EvaluateCareerStandingLocked(cpCriminalWallet)
	if len(events) != 1 || events[0].Kind != "warning" {
		t.Fatalf("a second evaluation produced %+v; want one warning", events)
	}
	if !l.leaderboard[cpCriminalWallet].CareerXP.DemotionWarningAt.Equal(stamped) {
		t.Error("the warning clock was RE-STAMPED inside the grace period; the deadline must be fixed")
	}
}

func TestCareerDemotionTakesTheCareerWhenTheGracePeriodExpires(t *testing.T) {
	l := careerPathTestLobby(t)
	careerPathPromotedJudge(t, l, cpCriminalWallet)

	// Drained, warned, and the grace period has run out.
	cxp := l.leaderboard[cpCriminalWallet].CareerXP
	cxp.AvgSustainedMicro = 0
	cxp.DemotionWarningAt = time.Now().Add(-(careerDemotionGrace() + time.Minute))
	cxp.DemotionWarningRole = "Judge"

	events := l.EvaluateCareerStandingLocked(cpCriminalWallet)
	if len(events) != 1 || events[0].Kind != "demoted" {
		t.Fatalf("an expired grace period produced %+v; want exactly one demotion", events)
	}
	after := l.leaderboard[cpCriminalWallet]
	if CareerHasRole(after.CareerXP, "Judge") {
		t.Error("the career survived the demotion")
	}
	if after.JobRole == "Judge" {
		t.Errorf("JobRole still names the demoted career: %q", after.JobRole)
	}
	if len(after.CareerXP.Demotions) != 1 {
		t.Fatalf("demotion records = %d; want 1", len(after.CareerXP.Demotions))
	}
	rec := after.CareerXP.Demotions[0]
	if rec.Role != "Judge" || rec.RequiredMicro != VBVTierBoss || rec.AvgMicro != 0 || rec.ShortfallMicro != VBVTierBoss {
		t.Errorf("the demotion record must carry the EXACT arithmetic; got %+v", rec)
	}
	if rec.Reason != CareerDemotionReasonDrain {
		t.Errorf("reason = %q; want %q", rec.Reason, CareerDemotionReasonDrain)
	}
	if rec.DemotedAt.IsZero() {
		t.Error("DemotedAt was not stamped")
	}
	if !after.CareerXP.DemotionWarningAt.IsZero() || after.CareerXP.DemotionWarningRole != "" {
		t.Error("the warning was not cleared by the demotion it produced")
	}
}

func TestCareerDemotionWarningIsWithdrawnWhenTheBalanceRecovers(t *testing.T) {
	l := careerPathTestLobby(t)
	careerPathPromotedJudge(t, l, cpCriminalWallet)

	// Warned...
	l.leaderboard[cpCriminalWallet].CareerXP.AvgSustainedMicro = 0
	if ev := l.EvaluateCareerStandingLocked(cpCriminalWallet); len(ev) != 1 || ev[0].Kind != "warning" {
		t.Fatalf("setup warning failed: %+v", ev)
	}
	// ...then the player EARNS IT BACK. The warning must be WITHDRAWN rather than left to expire,
	// because leaving it would demote a FUNDED player the next time the clock was checked.
	l.leaderboard[cpCriminalWallet].CareerXP.AvgSustainedMicro = VBVTierBoss
	events := l.EvaluateCareerStandingLocked(cpCriminalWallet)
	if len(events) != 1 || events[0].Kind != "cleared" {
		t.Fatalf("a recovered balance produced %+v; want exactly one cleared event", events)
	}
	cxp := l.leaderboard[cpCriminalWallet].CareerXP
	if !cxp.DemotionWarningAt.IsZero() || cxp.DemotionWarningRole != "" {
		t.Error("the warning survived a recovery")
	}
	if !CareerHasRole(cxp, "Judge") {
		t.Error("a recovered player lost the career")
	}
	// And a further evaluation produces NOTHING (the state has settled).
	if ev := l.EvaluateCareerStandingLocked(cpCriminalWallet); len(ev) != 0 {
		t.Errorf("a settled state produced %+v; want nothing", ev)
	}
}

// TestCareerTierGateIsNotPermanentlyFailedByAStaleWarning pins the fix in `CheckCareerTierGate`.
// The gate used to read `avg >= requiredMicro && !isDemotionWarning`, so a player who fell below,
// was warned, and then RESTORED the balance was failed by their own gate forever — the stale
// timestamp never cleared and no code path withdrew it. A recovered player must pass.
func TestCareerTierGateIsNotPermanentlyFailedByAStaleWarning(t *testing.T) {
	cxp := &CareerXP{
		AvgSustainedMicro: VBVTierBoss,
		LiquiditySamples:  []uint64{VBVTierBoss, VBVTierBoss},
		// A warning issued well past the grace period, i.e. exactly the state that used to
		// permanently fail the gate.
		DemotionWarningAt: time.Now().Add(-(careerDemotionGrace() + time.Hour)),
	}
	gatePass, currentTier, requiredMicro, isDemotionWarning := cxp.CheckCareerTierGate(5)
	if !gatePass {
		t.Errorf("a fully-funded player FAILED their own gate (tier %d, required %d, sustained %d, warning=%v)",
			currentTier, requiredMicro, cxp.AvgSustainedMicro, isDemotionWarning)
	}
	if !isDemotionWarning {
		t.Error("the age of the warning must still be reported, even though it no longer blocks the gate")
	}
}

// TestCareerDemotionNeverJudgesAnUndeclaredCareer pins the fail-closed rule: a role with no
// declared gate has no requirement to fall below, so it is never silently taken away.
func TestCareerDemotionNeverJudgesAnUndeclaredCareer(t *testing.T) {
	l := careerPathTestLobby(t)
	careerPathSeedPlayer(l, cpJusticeWallet, CareerPathJustice, 0)
	st := l.leaderboard[cpJusticeWallet]
	st.CareerXP.PromotedRoles = []string{"SomeLegacyRole"}
	st.CareerXP.DemotionWarningAt = time.Now().Add(-(careerDemotionGrace() + time.Hour))
	l.leaderboard[cpJusticeWallet] = st

	if ev := l.EvaluateCareerStandingLocked(cpJusticeWallet); len(ev) != 0 {
		t.Errorf("an undeclared career produced %+v; want nothing (no gate, nothing to fall below)", ev)
	}
	if !CareerHasRole(l.leaderboard[cpJusticeWallet].CareerXP, "SomeLegacyRole") {
		t.Error("an undeclared career was demoted")
	}
}

// ── 8. THE SERVED PROJECTION ─────────────────────────────────────────────────

func TestCareerPathViewServesTheWholeTaxonomy(t *testing.T) {
	l := careerPathTestLobby(t)
	careerPathMakeEligible(l, cpJusticeWallet, CareerPathJustice, "Warden")
	// The wallet must actually HOLD a career before a warning about one can exist.
	if _, err := l.PromoteCareerLocked(cpJusticeWallet, "Warden"); err != nil {
		t.Fatalf("fixture could not promote Warden: %v", err)
	}

	view := l.CareerPathViewLocked(cpJusticeWallet)

	if len(view.Paths) != 3 {
		t.Fatalf("served %d paths; want 3", len(view.Paths))
	}
	for _, p := range view.Paths {
		if len(p.Rivals) != 3 {
			t.Errorf("path %s served %d matrix cells; want 3", p.ID, len(p.Rivals))
		}
		if p.Summary == "" {
			t.Errorf("path %s has no summary, so the choice cannot be informed", p.ID)
		}
		for _, r := range p.Rivals {
			if r.Explain == "" {
				t.Errorf("path %s vs %s has no explanation", p.ID, r.Against)
			}
			want := CareerPathRivalry(p.ID, r.Against)
			if r.Relation != want {
				t.Errorf("served relation %s vs %s = %q; the resolver says %q", p.ID, r.Against, r.Relation, want)
			}
			if r.BonusBps != PathRivalryBonusBps(p.ID, r.Against) {
				t.Errorf("served bonus %s vs %s = %d; the resolver says %d", p.ID, r.Against, r.BonusBps, PathRivalryBonusBps(p.ID, r.Against))
			}
		}
	}

	if len(view.CivilRanks) != 3 {
		t.Errorf("served %d civil ranks; want 3", len(view.CivilRanks))
	}
	for _, r := range view.CivilRanks {
		if r.Requires == "" {
			t.Errorf("civil rank %s does not state what unlocks it", r.Tier)
		}
	}

	if len(view.Matrices) != len(RivalryMatrixStages) {
		t.Errorf("served %d matrices; want %d (the client shows which layer a number came from)",
			len(view.Matrices), len(RivalryMatrixStages))
	}
	for _, m := range view.Matrices {
		if m.Owner == "" || m.Describes == "" {
			t.Errorf("matrix %s is served without its owner/description: %+v", m.Stage, m)
		}
	}

	// Every declared pair is served, with the delta the engine actually prices it at.
	if len(view.CareerPairs) != len(rivalPairTable) {
		t.Errorf("served %d career pairs; the table declares %d", len(view.CareerPairs), len(rivalPairTable))
	}
	for _, p := range view.CareerPairs {
		if p.Delta != GetRivalXPDelta(p.Pair) {
			t.Errorf("served delta for %s = %d; GetRivalXPDelta says %d", p.Pair, p.Delta, GetRivalXPDelta(p.Pair))
		}
		if p.A == "" || p.B == "" {
			t.Errorf("pair %s is served without both sides' paths (%q / %q)", p.Pair, p.A, p.B)
		}
	}

	// The career table is complete and DETERMINISTICALLY ordered, so a refresh never looks like a
	// change (the defect class the bonded-asset chest and the market both had to repair).
	if len(view.Careers) != len(careerRoleGates) {
		t.Errorf("served %d careers; %d are declared", len(view.Careers), len(careerRoleGates))
	}
	for i := 1; i < len(view.Careers); i++ {
		prev, cur := view.Careers[i-1], view.Careers[i]
		if prev.HomePath > cur.HomePath ||
			(prev.HomePath == cur.HomePath && prev.MinLessonLevel > cur.MinLessonLevel) ||
			(prev.HomePath == cur.HomePath && prev.MinLessonLevel == cur.MinLessonLevel && prev.Role > cur.Role) {
			t.Errorf("the career table is not in its declared order at %d: %+v then %+v", i, prev, cur)
		}
	}

	// The rules travel WITH the data, so the client quotes the server rather than restating it.
	for _, key := range []string{"choose_once", "choice_gate", "neutral_breadth", "direct_bonus_bps",
		"promotion_basis", "demotion_basis", "demotion_grace_days", "civil_rank_is_engine_state"} {
		if _, ok := view.Rules[key]; !ok {
			t.Errorf("the served rules are missing %q", key)
		}
	}
	if got, ok := view.Rules["direct_bonus_bps"].(int); !ok || got != PathDirectRivalryBonusBps {
		t.Errorf("served direct_bonus_bps = %v; want %d", view.Rules["direct_bonus_bps"], PathDirectRivalryBonusBps)
	}

	// A warned player's warning is served, so the panel can state it after a reload — the reason a
	// missed WebSocket delivery loses nothing.
	l.leaderboard[cpJusticeWallet].CareerXP.AvgSustainedMicro = 0
	l.EvaluateCareerStandingLocked(cpJusticeWallet)
	view = l.CareerPathViewLocked(cpJusticeWallet)
	if view.Warning == nil || view.Warning.Role != "Warden" {
		t.Errorf("the served view lost the outstanding warning: %+v", view.Warning)
	}
	if view.Warning.ShortfallMicro == 0 {
		t.Error("the served warning carries no shortfall")
	}
}

// ── 9. THE HTTP BOUNDARY — the client may name the path and the role, nothing else ──

func TestCareerPathHTTPBoundaryRefusesWhatItDoesNotOwn(t *testing.T) {
	l := careerPathTestLobby(t)
	careerPathSeedOwner(l, "0xready", 2, true, 0)
	careerPathSeedPlayer(l, "0xready", "", 0)

	// GET only.
	rec := httptest.NewRecorder()
	l.handleCareerPath(rec, httptest.NewRequest(http.MethodPost, "/api/career/path", strings.NewReader("{}")))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /api/career/path = %d; want 405", rec.Code)
	}

	// A wallet is required, and the refusal says how to supply one.
	rec = httptest.NewRecorder()
	l.handleCareerPath(rec, httptest.NewRequest(http.MethodGet, "/api/career/path", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("GET with no wallet = %d; want 401", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "X-Wallet-Address") {
		t.Errorf("the 401 must name how to supply a wallet: %s", rec.Body.String())
	}

	// The happy read.
	rec = httptest.NewRecorder()
	l.handleCareerPath(rec, httptest.NewRequest(http.MethodGet, "/api/career/path?wallet=0xready", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET with a wallet = %d; body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("the served payload is not JSON: %v", err)
	}
	data, ok := payload["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("the payload has no data object: %v", payload)
	}
	if paths, ok := data["paths"].([]interface{}); !ok || len(paths) != 3 {
		t.Errorf("served paths = %v; want 3", data["paths"])
	}

	// POST only on the choose door.
	rec = httptest.NewRecorder()
	l.handleCareerPathChoose(rec, httptest.NewRequest(http.MethodGet, "/api/career/path/choose", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/career/path/choose = %d; want 405", rec.Code)
	}

	// A body naming anything the SERVER owns cannot even be parsed.
	for _, body := range []string{
		`{"path":"justice","civil_tier":"governor"}`,
		`{"path":"justice","promoted_roles":["Judge"]}`,
		`{"path":"justice","sustained_micro":999999999999}`,
	} {
		rec = httptest.NewRecorder()
		l.handleCareerPathChoose(rec, httptest.NewRequest(http.MethodPost, "/api/career/path/choose?wallet=0xready", strings.NewReader(body)))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("body %s = %d; want 400 (only `path` is accepted)", body, rec.Code)
		}
		if got := l.leaderboard["0xready"].CareerXP.Path; got != "" {
			t.Fatalf("a refused body (%s) chose the path %q", body, got)
		}
	}

	// An unknown path is a 400 (a malformed request), not a 403 (a refusal of a real one).
	rec = httptest.NewRecorder()
	l.handleCareerPathChoose(rec, httptest.NewRequest(http.MethodPost, "/api/career/path/choose?wallet=0xready", strings.NewReader(`{"path":"republic"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("an unknown path = %d; want 400", rec.Code)
	}

	// A wallet that is not ELIGIBLE gets 403 with the eligibility object, so the UI can state why.
	careerPathSeedPlayer(l, "0xpoor", "", 0)
	rec = httptest.NewRecorder()
	l.handleCareerPathChoose(rec, httptest.NewRequest(http.MethodPost, "/api/career/path/choose?wallet=0xpoor", strings.NewReader(`{"path":"justice"}`)))
	if rec.Code != http.StatusForbidden {
		t.Errorf("an ineligible wallet = %d; want 403", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "eligibility") {
		t.Errorf("the 403 must carry the eligibility: %s", rec.Body.String())
	}

	// The real choice succeeds and is persisted.
	rec = httptest.NewRecorder()
	l.handleCareerPathChoose(rec, httptest.NewRequest(http.MethodPost, "/api/career/path/choose?wallet=0xready", strings.NewReader(`{"path":"justice"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("an eligible wallet was refused: %d %s", rec.Code, rec.Body.String())
	}
	if got := l.leaderboard["0xready"].CareerXP.Path; got != CareerPathJustice {
		t.Errorf("stored path = %q; want justice", got)
	}

	// And the choice is made once, over HTTP too.
	rec = httptest.NewRecorder()
	l.handleCareerPathChoose(rec, httptest.NewRequest(http.MethodPost, "/api/career/path/choose?wallet=0xready", strings.NewReader(`{"path":"criminal"}`)))
	if rec.Code != http.StatusForbidden {
		t.Errorf("a second choice = %d; want 403", rec.Code)
	}
}

func TestCareerPromoteHTTPBoundaryNamesTheMissingGate(t *testing.T) {
	l := careerPathTestLobby(t)
	careerPathMakeEligible(l, "0xready", CareerPathJustice, "Warden")

	// POST only.
	rec := httptest.NewRecorder()
	l.handleCareerPromote(rec, httptest.NewRequest(http.MethodGet, "/api/career/promote", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/career/promote = %d; want 405", rec.Code)
	}

	// An empty role is a 400.
	rec = httptest.NewRecorder()
	l.handleCareerPromote(rec, httptest.NewRequest(http.MethodPost, "/api/career/promote?wallet=0xready", strings.NewReader(`{}`)))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("an empty role = %d; want 400", rec.Code)
	}

	// A body naming a gate cannot be parsed, so a client cannot grant itself one.
	rec = httptest.NewRecorder()
	l.handleCareerPromote(rec, httptest.NewRequest(http.MethodPost, "/api/career/promote?wallet=0xready",
		strings.NewReader(`{"role":"Warden","min_lesson_level":0}`)))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("a body naming a gate = %d; want 400", rec.Code)
	}

	// A refused promotion carries the requirements object, so the UI states the missing piece.
	l.leaderboard["0xready"].CareerXP.AvgSustainedMicro = 0
	rec = httptest.NewRecorder()
	l.handleCareerPromote(rec, httptest.NewRequest(http.MethodPost, "/api/career/promote?wallet=0xready", strings.NewReader(`{"role":"Warden"}`)))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("an unfunded promotion = %d; want 403 (body %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "requirements") {
		t.Errorf("the 403 must carry the requirements: %s", rec.Body.String())
	}
	if len(l.leaderboard["0xready"].CareerXP.PromotedRoles) != 0 {
		t.Error("a refused HTTP promotion wrote into PromotedRoles")
	}

	// The funded promotion succeeds.
	l.leaderboard["0xready"].CareerXP.AvgSustainedMicro = VBVTierBoss
	rec = httptest.NewRecorder()
	l.handleCareerPromote(rec, httptest.NewRequest(http.MethodPost, "/api/career/promote?wallet=0xready", strings.NewReader(`{"role":"Warden"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("a funded promotion was refused: %d %s", rec.Code, rec.Body.String())
	}
	if !CareerHasRole(l.leaderboard["0xready"].CareerXP, "Warden") {
		t.Error("the HTTP promotion did not land in PromotedRoles")
	}
}
