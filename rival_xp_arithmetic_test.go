//go:build !js && !wasm

package main

import (
	"reflect"
	"testing"
)

// ════════════════════════════════════════════════════════════════════════════
// THE RIVAL XP ARITHMETIC — integer-only, ONE owner, and the courthouse's two
// matrices.
//
// WHAT THESE TESTS ARE FOR. Three defects lived in this path at once:
//
//  1. Six call sites read the SECOND return value of `EvaluateCrossCareerXP`
//     (the defender's 30% share) and compared it against the BASE — `4 > 15` —
//     so five rival hooks had NEVER fired once.
//  2. `uint64(rivalXP - 15)` underflows to 18,446,744,073,709,551,605; the
//     false guard in (1) was the only thing hiding it.
//  3. The arithmetic was float on career XP, which the Architecture Ledger
//     prohibits.
//
// And a FOURTH, separate defect: the PATH matrix's direct justice↔criminal
// bonus was DECLARED, SERVED to the client and pinned by a test — and awarded
// in ZERO places, so the courthouse never acted as a rival to the criminal
// system.
// ════════════════════════════════════════════════════════════════════════════

func rivalXPActor(role, path string) PlayerStats {
	return PlayerStats{Wallet: "0xactor", JobRole: role, CareerXP: &CareerXP{Path: path}}
}

func rivalXPPeer(role, path string) PlayerStats {
	return PlayerStats{Wallet: "0xpeer", JobRole: role, CareerXP: &CareerXP{Path: path}}
}

// TestRivalXPGainPermillePricesEveryDeclaredPair pins the integer gain table,
// and pins that the float MODIFIER getter is a display wrapper over the permille
// owner rather than a second implementation.
func TestRivalXPGainPermillePricesEveryDeclaredPair(t *testing.T) {
	cases := []struct {
		pair string
		want int
	}{
		{"BountyHunter↔Kidnapper", 1150},         // −15 → the largest gain
		{"ForensicAnalyst↔Gossip", 1100},         // −10
		{"TaxAuditor↔Launderer", 1100},           // −10
		{"Warden↔HeistPlanner", 1100},            // −10
		{"SectorPeacekeeper↔Smuggler", 1100},     // −10
		{"IntelAgent↔ArcNetOperative", 1100},     // −10
		{"JusticeRecruiter↔BountyHunter", 1080},  // +8
		{"Launderer↔Fence", 1050},                // +5
		{"HeistPlanner↔Kidnapper", 1120},         // +12
		{"AOS↔SectorPeacekeeper", 1060},          // +6
		{"TaxAuditor↔JusticeCommissioner", 1070}, // +7
		{"NoSuchPair↔Nobody", 1000},              // undeclared == unchanged
	}
	for _, c := range cases {
		if got := GetRivalXPGainPermille(c.pair); got != c.want {
			t.Errorf("GetRivalXPGainPermille(%q) = %d; want %d", c.pair, got, c.want)
		}
	}

	// AN ANTAGONISTIC PAIR MUST BE WORTH MORE THAN A SYNERGISTIC ONE — that is the point of
	// declaring it antagonistic, and it is what the two identical branches of the old float code did.
	enemy := GetRivalXPGainPermille("BountyHunter↔Kidnapper")
	ally := GetRivalXPGainPermille("Launderer↔Fence")
	if enemy <= ally {
		t.Errorf("an antagonistic pair gains %d and a synergistic one %d: the enemy pair must be worth MORE", enemy, ally)
	}

	var cxp CareerXP
	for tier := 0; tier <= 6; tier++ {
		want := float64(GetRivalPairModifierPermille(tier)) / 1000.0
		if got := cxp.GetRivalPairModifier("any role", tier); got != want {
			t.Errorf("the float modifier at tier %d = %v; the permille owner says %v — the display wrapper must not be a second implementation", tier, got, want)
		}
	}
}

// TestResolveRivalXPAwardIsExactIntegerArithmetic pins the composition and the
// exact numbers, so a later "small tidy-up" cannot reintroduce a float.
func TestResolveRivalXPAwardIsExactIntegerArithmetic(t *testing.T) {
	actor := rivalXPActor("Tax Auditor", "")
	peer := rivalXPPeer("JusticeCommissioner", "")

	a := ResolveRivalXPAward("Tax Auditor", "JusticeCommissioner", CourthouseFineCareerXP, &actor, &peer)

	// career pair +7 → 1070 permille; tier 1 → 1100 permille; composed → 1070 × 1100 / 1000 = 1177.
	if a.GainPermille != 1177 {
		t.Errorf("composed gain = %d permille; want 1177 (1070 from the pair × 1100 from tier 1)", a.GainPermille)
	}
	// 15 × 1177 / 1000 = 17. The float form computed 15 × 1.07 × 1.1 = 17.655 → 17, so the integer
	// form is not a change of economics — it is the same number without the binary approximation.
	if a.AttackerXP != 17 {
		t.Errorf("attacker XP = %d; want 17 (15 × 1177 / 1000)", a.AttackerXP)
	}
	if a.DefenderXP != 4 {
		t.Errorf("defender share = %d; want 4 (15 × 300 / 1000 — it used to be `float64(base) * 0.30`)", a.DefenderXP)
	}
	if a.Bonus != 2 {
		t.Errorf("bonus = %d; want 2 (17 − 15)", a.Bonus)
	}
	if a.Bonus != a.AttackerXP-a.BaseXP {
		t.Errorf("bonus %d != attacker %d − base %d", a.Bonus, a.AttackerXP, a.BaseXP)
	}
	if !a.IsRival || a.IsDirect || len(a.Layers) != 1 || a.Layers[0] != MatrixStageCareer {
		t.Errorf("the pair is a CAREER-matrix pairing only; got rival=%v direct=%v layers=%v", a.IsRival, a.IsDirect, a.Layers)
	}
	if a.Explain == "" {
		t.Error("an award must state why it was priced — Explain is empty")
	}
}

// TestTheCourthousePairingActuallyPays IS THE REGRESSION for the dead arithmetic.
//
// The old code read the second return value (`defenderXP` = base × 0.30 = 4 and 9) into a variable
// named `rivalXP` and compared it against the base (15 and 30). `4 > 15` is false, so the
// TaxAuditor↔JusticeCommissioner hook had never been awarded anywhere — at either resolution point.
func TestTheCourthousePairingActuallyPays(t *testing.T) {
	actor := rivalXPActor("Tax Auditor", "")
	peer := rivalXPPeer("JusticeCommissioner", "")

	for _, base := range []uint64{CourthouseFineCareerXP, CourthousePardonCareerXP} {
		a := ResolveRivalXPAward("Tax Auditor", "JusticeCommissioner", base, &actor, &peer)
		if a.Bonus == 0 {
			t.Errorf("base %d: the TaxAuditor↔JusticeCommissioner hook paid NOTHING — this is the dead arithmetic returning", base)
		}
		// The guard can only be written safely because the defender share stays BELOW the base: if a
		// future change made the share exceed the base, `uint64(share - base)` would wrap instead of
		// going negative — which is exactly how this defect hid the second one.
		if a.DefenderXP >= base {
			t.Errorf("base %d: the defender share is %d, which is not below the base — the old comparison would read as true", base, a.DefenderXP)
		}
		if a.Bonus != a.AttackerXP-base {
			t.Errorf("base %d: bonus %d != attacker %d − base %d", base, a.Bonus, a.AttackerXP, base)
		}
	}
}

// TestTheDirectPathRivalryIsAwarded pins that the PATH matrix is no longer a
// declaration nobody pays: `PathDirectRivalryBonusBps` was computed, served to
// the client and asserted by a test — and awarded in zero places.
func TestTheDirectPathRivalryIsAwarded(t *testing.T) {
	// The fixture MUST use a role with no declared career pair, or the career matrix would pay and
	// the path matrix could still be unwired without this test noticing.
	if pair := GetRivalPairName("Tax Auditor", "Judge"); pair != "" {
		t.Fatalf("the fixture needs a peer role with no declared pair; Tax Auditor↔Judge resolved to %q", pair)
	}

	justice := rivalXPActor("Tax Auditor", CareerPathJustice)
	criminal := rivalXPPeer("Judge", CareerPathCriminal)

	a := ResolveRivalXPAward("Tax Auditor", "Judge", CourthousePardonCareerXP, &justice, &criminal)
	if a.IsRival {
		t.Error("no career pair is declared for this fixture, so IsRival must be false")
	}
	if !a.IsDirect || a.PathRelation != PathRivalryDirect {
		t.Errorf("justice vs criminal must resolve DIRECT; got %q (IsDirect=%v)", a.PathRelation, a.IsDirect)
	}
	if a.PathBonusBps != PathDirectRivalryBonusBps {
		t.Errorf("path bonus = %d bps; want the declared %d", a.PathBonusBps, PathDirectRivalryBonusBps)
	}
	if len(a.Layers) != 1 || a.Layers[0] != MatrixStagePath {
		t.Errorf("layers = %v; want exactly the path matrix", a.Layers)
	}
	// 30 × (1000 × 11000 / 10000) / 1000 = 30 × 1100 / 1000 = 33.
	if a.AttackerXP != 33 || a.Bonus != 3 {
		t.Errorf("attacker XP = %d (want 33) and bonus = %d (want 3) — the DIRECT path bonus must be PAID, not merely declared", a.AttackerXP, a.Bonus)
	}

	// A NEUTRAL player INTERPRETS the matrix and holds no direct enemy. This is what neutrality
	// costs — without it, holding a side would be strictly worse than staying neutral.
	neutral := rivalXPActor("Tax Auditor", CareerPathNeutral)
	if b := ResolveRivalXPAward("Tax Auditor", "Judge", CourthousePardonCareerXP, &neutral, &criminal); b.IsDirect || b.Bonus != 0 {
		t.Errorf("a neutral actor earned the direct bonus (IsDirect=%v bonus=%d); the neutral path reads the matrix and never leverages it", b.IsDirect, b.Bonus)
	}

	// A player who has NOT chosen holds NO path. The leverage is UNLOCKED by the choice (two
	// territories and an opened region), never assumed on a player's behalf.
	unchosen := rivalXPActor("Tax Auditor", "")
	if c := ResolveRivalXPAward("Tax Auditor", "Judge", CourthousePardonCareerXP, &unchosen, &criminal); c.IsDirect || c.PathRelation != PathRivalryNone {
		t.Errorf("a player who has chosen no path resolved as %q — it must resolve none", c.PathRelation)
	}
}

// TestRivalXPAwardIsDeterministic pins that identical inputs give identical
// outputs, 25 times over: the Ledger's determinism rule applies to XP.
func TestRivalXPAwardIsDeterministic(t *testing.T) {
	actor := rivalXPActor("Tax Auditor", CareerPathJustice)
	peer := rivalXPPeer("Launderer", CareerPathCriminal)

	first := ResolveRivalXPAward("Tax Auditor", "Launderer", CourthouseFineCareerXP, &actor, &peer)
	for i := 0; i < 25; i++ {
		got := ResolveRivalXPAward("Tax Auditor", "Launderer", CourthouseFineCareerXP, &actor, &peer)
		if !reflect.DeepEqual(got, first) {
			t.Fatalf("resolution %d differs from the first:\n first=%+v\n got  =%+v", i, first, got)
		}
	}
	if !first.IsRival || !first.IsDirect {
		t.Fatalf("the fixture must fire BOTH matrices; got rival=%v direct=%v (%s)", first.IsRival, first.IsDirect, first.Explain)
	}
	if len(first.Layers) != 2 {
		t.Errorf("layers = %v; want both the career and the path matrix to be named", first.Layers)
	}
}
