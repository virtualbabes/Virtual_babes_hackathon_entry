//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ============================================================================
// THE CAREER PATH · CIVIL TIER · PROMOTION & DEMOTION DOMAIN
// ============================================================================
//
// WHY THIS FILE EXISTS. Three things were asked for, and none of them had an owner:
//
//  1. A PLAYER picks a career PATH — justice, criminal or neutral — when the player OPENS A
//     REGION (which the engine gates at 2+ owned territories). Nothing in the repository held
//     a path at all: `Public/js/career_tree.js` INVENTED a `faction: JUSTICE|UNDERWORLD|HYBRID`
//     taxonomy in the client, and the server has never served a single one of those strings.
//     A client-invented taxonomy is a fabrication class this repository has already had to
//     repair twice, so the vocabulary lives HERE and is SERVED.
//
//  2. PROMOTION and DEMOTION. `CareerXP.PromotedRoles` is written by NOTHING — it is read in
//     `rival_career_engine.go` and `lobby_manager.go` and assigned nowhere — so `CareerHasRole`
//     is permanently false and `JobRole` holds only the single literal `"Freelancer"` written
//     when a salary contract defaults. There is therefore no promotion in this build, and
//     demotion is warning-only. `server.go`'s liquidity sampler does broadcast
//     `career_tier_demoted`, but under `if lwb == walletLower || cid != ""` — a condition whose
//     second clause is ALWAYS true, so the "your career is being demoted" event is sent to
//     EVERY connected client. This file is the owner of the real lifecycle.
//
//  3. A THREE-TIER CIVIL RANK — user (no shops/territories), manager (owns territories, may own
//     regions), governor (must own a region) — used as a GATE, so that a high career cannot be
//     held by a player with no organisation behind them. This is not a new idea: the pathway
//     constants already say so (`AIPathwaySyndicate` in `ai_citizen_engine.go` is commented
//     "UnderworldBoss / Judge (governor-gated)"). What was missing is ONE derivation, because
//     `isGovernor` was re-computed ad hoc in eight files with slightly different rules.
//
// EVERYTHING BELOW IS INTEGER, DETERMINISTIC AND DERIVED FROM ENGINE STATE. A client can NAME a
// path (that is the one thing it is allowed to choose); it can never name its civil tier, its
// promotion, its demotion or a price.
// ============================================================================

// ---------------------------------------------------------------------------
// 1. THE PATH VOCABULARY — three paths, one key each
// ---------------------------------------------------------------------------

const (
	CareerPathJustice  = "justice"
	CareerPathCriminal = "criminal"
	CareerPathNeutral  = "neutral"
)

// careerPathAliases folds a spelling onto one of the three paths. Both sides are outputs of
// foldRole (the same fold the career-role vocabulary uses), and every entry NAMES ITS SOURCE so
// a reader can verify the spelling really occurs rather than being invented here.
var careerPathAliases = map[string]string{
	// `Public/js/career_tree.js` declares `faction: 'UNDERWORLD'` for the Syndicate/Ledger/
	// Forensic/Boss pathways. Underworld is the criminal path; the word is the CLIENT's label
	// for it, not a fourth path.
	"underworld": CareerPathCriminal,
	// `career_tree.js` declares `faction: 'HYBRID'` for the Tax and Commissioner pathways —
	// a pathway that serves BOTH sides. The three paths are exclusive, and the position that
	// "serves both sides / owns neither" IS the neutral path (it reads the matrix rather than
	// holding a side), so HYBRID folds to neutral.
	"hybrid": CareerPathNeutral,
	// `item_shop_archetype.go` and the justice/underworld item families speak "law" for the
	// justice side; `P-Justice` below is the same side.
	"law": CareerPathJustice,
}

// PathKey returns the comparison key for a career path. Use it for EVERY path comparison instead
// of `==`, so "Justice", "JUSTICE" and "law" are one path.
func PathKey(path string) string {
	folded := foldRole(path)
	if canonical, ok := careerPathAliases[folded]; ok {
		return canonical
	}
	return folded
}

// IsCareerPath reports whether a (possibly misspelled) value names one of the three paths.
func IsCareerPath(path string) bool {
	switch PathKey(path) {
	case CareerPathJustice, CareerPathCriminal, CareerPathNeutral:
		return true
	default:
		return false
	}
}

// CareerPaths is the declared order every served list uses, so a refresh never looks like a change.
var CareerPaths = []string{CareerPathJustice, CareerPathCriminal, CareerPathNeutral}

// ---------------------------------------------------------------------------
// 2. THE CIVIL RANK VOCABULARY — user / manager / governor
// ---------------------------------------------------------------------------

const (
	CivilTierUser     = "user"     // no shops, no territories
	CivilTierManager  = "manager"  // owns territories; may also own a region
	CivilTierGovernor = "governor" // must own a REGION (the region ability is opened)
)

// CivilRanks is the declared order for serving + iteration (weakest first).
var CivilRanks = []string{CivilTierUser, CivilTierManager, CivilTierGovernor}

// civilRankIndex orders the ranks so a gate is a comparison, never a chain of ifs.
func civilRankIndex(tier string) int {
	switch PathKey(tier) {
	case CivilTierManager:
		return 1
	case CivilTierGovernor:
		return 2
	default:
		return 0
	}
}

// ---------------------------------------------------------------------------
// 3. THE PATH OF A CAREER — READ OFF THE RIVAL-PAIR MATRIX, NOT INVENTED
// ---------------------------------------------------------------------------
//
// The rival-pair matrix in `rival_career_engine.go` already brackets each career as an ENEMY or
// an ALLY of another career. That matrix IS the two sides, so the sides are read OFF it instead
// of being declared a second time:
//
//	attacker side of every ENEMY pair  → JUSTICE
//	defender side of every ENEMY pair  → CRIMINAL
//
// read from the long-standing enemy pairs (BountyHunter↔Kidnapper, ForensicAnalyst↔Gossip,
// TaxAuditor↔Launderer, Warden↔HeistPlanner, SectorPeacekeeper↔Smuggler,
// IntelAgent↔ArcNetOperative, MutationLogAuditor↔Kidnapper). The three careers that appear in NO
// enemy pair (UnderworldBoss, Judge, Saboteur) are placed from the pathway table they already sit
// in (`aiPathwayByCareer` + `itemArchetypeRegistry`), and each entry below NAMES that evidence.
//
// `TestCareerPathAgreesWithTheRivalMatrix` pins the agreement mechanically: every ENEMY pair must
// CROSS paths and every ALLY pair must SHARE a path. If a future pair contradicts these
// assignments the test fails, so the two matrices cannot silently drift apart.
var careerPathByRole = map[string]string{
	// ---- JUSTICE side (attacker of an enemy pair, or pathway P-Justice/P-Shadow/P-Lockdown) ----
	"bountyhunter":        CareerPathJustice, // enemy vs Kidnapper
	"forensicanalyst":     CareerPathJustice, // enemy vs Gossip
	"taxauditor":          CareerPathJustice, // enemy vs Launderer
	"warden":              CareerPathJustice, // enemy vs HeistPlanner
	"sectorpeacekeeper":   CareerPathJustice, // enemy vs Smuggler
	"intelagent":          CareerPathJustice, // enemy vs ArcNetOperative
	"justicerecruiter":    CareerPathJustice, // P-Shadow; ally of BountyHunter + MutationLogAuditor
	"aosleader":           CareerPathJustice, // P-AOS; ally of SectorPeacekeeper (rivalPairTable names the side "AOS Leader")
	"justicecommissioner": CareerPathJustice, // P-Commissioner; ally of TaxAuditor
	"mutationlogauditor":  CareerPathJustice, // enemy vs Kidnapper; ally of JusticeRecruiter

	// ---- CRIMINAL side (defender of an enemy pair, or pathway P-Syndicate/P-Ledger) ----
	"kidnapper":       CareerPathCriminal, // enemy vs BountyHunter + MutationLogAuditor
	"gossip":          CareerPathCriminal, // enemy vs ForensicAnalyst
	"launderer":       CareerPathCriminal, // enemy vs TaxAuditor; ally of Fence
	"heistplanner":    CareerPathCriminal, // enemy vs Warden; ally of Kidnapper
	"smuggler":        CareerPathCriminal, // enemy vs SectorPeacekeeper
	"arcnetoperative": CareerPathCriminal, // enemy vs IntelAgent
	"fence":           CareerPathCriminal, // ally of Launderer
	"underworldboss":  CareerPathCriminal, // aiPathwayByCareer → P-Syndicate (UnderworldBoss/Judge)
	"judge":           CareerPathCriminal, // aiPathwayByCareer → P-Syndicate
	"saboteur":        CareerPathCriminal, // aiPathwayByCareer → P-Intel, the ArcNetOperative pathway
}

// CareerPathOfRole returns the home path of a career role, or "" if the role is unknown to the
// matrix. Resolved BY ROLE (RoleKey), so "Int.Agent" resolves like "IntelAgent".
func CareerPathOfRole(role string) string {
	if role == "" {
		return ""
	}
	if p, ok := careerPathByRole[RoleKey(role)]; ok {
		return p
	}
	return ""
}

// PlayerCareerPathOfStats returns the path a PLAYER holds, or "" if none has been chosen.
//
// The path lives on the player's own career record (`CareerXP.Path`, written by
// ChooseCareerPathLocked and by nothing else), so this is the ONE reader the rival XP resolver and
// the served path matrix both consult — a number a player is SHOWN and a number they are PAID
// cannot come from two different sources.
//
// A player who has not chosen holds NO path, and `CareerPathRivalry` answers `none` for them: the
// direct-rival leverage is UNLOCKED by the choice (two territories + an opened region), never
// assumed on a player's behalf.
func PlayerCareerPathOfStats(stats *PlayerStats) string {
	if stats == nil || stats.CareerXP == nil {
		return ""
	}
	return PathKey(stats.CareerXP.Path)
}

// PathAllowsRole reports whether a player on `path` may PROMOTE into `role`.
//
// A sided player may hold their own side's careers only; a NEUTRAL player may hold either side's
// careers, because neutrality buys breadth — and pays for it with no direct-rival leverage
// (see PathRivalryBonusBps). A role the matrix does not know is refused, so an unknown career
// cannot be granted by a spelling nobody verified.
func PathAllowsRole(path, role string) bool {
	home := CareerPathOfRole(role)
	if home == "" {
		return false
	}
	switch PathKey(path) {
	case CareerPathNeutral:
		return true
	case CareerPathJustice, CareerPathCriminal:
		return PathKey(path) == home
	default:
		return false
	}
}

// ---------------------------------------------------------------------------
// 4. THE PATH MATRIX — direct / interpreted / shared / none
// ---------------------------------------------------------------------------
//
// "direct rivalry is justice/criminal; a neutral user will interpret rivals from other neutral
// users and the rival matrix dynamic behind other driving aspects."
//
// That is a four-valued relation and it is RESOLVED HERE, because the client must never decide
// it: a DIRECT pair is an XP-leverage pair (the engine already prices enemy pairs with a positive
// `GetRivalXPDelta` bonus), an INTERPRETED pair is a player READING the matrix (no direct
// leverage), a SHARED path is an ally relation, and an unset path resolves to NONE.
const (
	PathRivalryDirect      = "direct"      // justice vs criminal — holds a side, earns the rival delta
	PathRivalryInterpreted = "interpreted" // a neutral reading another neutral, or reading a sided pair
	PathRivalryShared      = "shared"      // same side — an ally relation, not a rivalry
	PathRivalryNone        = "none"        // no path is held on at least one side
)

// CareerPathRivalry resolves the relation between two players' paths.
func CareerPathRivalry(a, b string) string {
	ka, kb := PathKey(a), PathKey(b)
	if !IsCareerPath(ka) || !IsCareerPath(kb) {
		return PathRivalryNone
	}
	switch {
	case ka == kb:
		// Two neutrals DO read a rivalry off each other (the matrix behind other driving
		// aspects); two sided players on the same side are allies.
		if ka == CareerPathNeutral {
			return PathRivalryInterpreted
		}
		return PathRivalryShared
	case ka == CareerPathNeutral || kb == CareerPathNeutral:
		// A neutral holds no side, so it never has a DIRECT enemy; it interprets the pairing.
		return PathRivalryInterpreted
	default:
		// The one direct rivalry in the design.
		return PathRivalryDirect
	}
}

// PathDirectRivalryBonusBps is the integer bonus (basis points) a DIRECT path relation adds.
//
// Neutrality buys BREADTH — a neutral may promote into either side's careers (PathAllowsRole) —
// and this constant is what it costs: a neutral holds no direct enemy, so it never earns this
// bonus. Without this the neutral path would be strictly better than the other two.
const PathDirectRivalryBonusBps = 1000 // +10% XP when acting against the opposed path

// PathRivalryBonusBps returns the bonus in basis points for a path relation.
func PathRivalryBonusBps(a, b string) int {
	if CareerPathRivalry(a, b) == PathRivalryDirect {
		return PathDirectRivalryBonusBps
	}
	return 0
}

// RivalryMatrixStage names the THREE matrices a player is rivalled through, so the UI can show
// "multiple rival matrices across the entire user experience" without guessing which layer a
// number came from.
const (
	MatrixStagePath   = "path"   // justice ↔ criminal (this file)
	MatrixStageCareer = "career" // role ↔ role (rivalPairTable, rival_career_engine.go)
	MatrixStageRegion = "region" // territory/region signatures (rivalry_engine.go + theme_engine.go)
)

// RivalryMatrixStages is the declared order of the matrices.
var RivalryMatrixStages = []string{MatrixStagePath, MatrixStageCareer, MatrixStageRegion}

// ---------------------------------------------------------------------------
// 5. THE CIVIL RANK — DERIVED FROM ENGINE STATE, NEVER FROM A REQUEST
// ---------------------------------------------------------------------------

// CivilStanding is what the engine can SEE about a wallet's organisation, and the rank derived
// from it. Every field is a count or a boolean read from `l.clubs`; nothing here is cached and
// nothing is client-supplied, so a rank cannot be claimed.
type CivilStanding struct {
	Tier         string `json:"tier"`          // user | manager | governor
	Shops        int    `json:"shops"`         // clubs owned by the wallet
	Territories  int    `json:"territories"`   // territories held across those clubs (alliance-aware)
	Region       string `json:"region"`        // the region this wallet governs, "" if none
	RegionOpened bool   `json:"region_opened"` // the club's region ability has been opened
}

// CivilStandingForWalletLocked derives a wallet's civil rank. The CALLER HOLDS the lobby lock.
//
// The rule is the engine's own, reused rather than re-invented:
//   - governor — owns a club whose REGION is opened (`RegionName != ""`). `HandleOpenRegionalManager`
//     is the only writer of that field and it refuses below 2 territories, so "must own a region"
//     is enforced at the door that opens one.
//   - manager  — holds at least one territory (counted alliance-aware, the way
//     `IsClubRegionalLocked` already counts, so an alliance counts the way the engine says).
//   - user     — neither. A wallet may still own a SHOP and be a user: that is reported, because
//     "you own a shop but no territory" is a different state from "you own nothing" and the UI
//     must be able to say which one it is.
func (l *Lobby) CivilStandingForWalletLocked(wallet string) CivilStanding {
	standing := CivilStanding{Tier: CivilTierUser}
	if l == nil || wallet == "" {
		return standing
	}
	for _, club := range l.clubs {
		if club == nil || !strings.EqualFold(club.OwnerWallet, wallet) {
			continue
		}
		standing.Shops++
		territories := len(club.Territories)
		if club.AlliedClubID != "" {
			if allied, ok := l.clubs[club.AlliedClubID]; ok && allied != nil {
				territories += len(allied.Territories)
			}
		}
		if territories > standing.Territories {
			standing.Territories = territories
		}
		if club.RegionName != "" {
			standing.RegionOpened = true
			standing.Region = club.RegionName
		}
	}
	switch {
	case standing.RegionOpened:
		standing.Tier = CivilTierGovernor
	case standing.Territories >= 1:
		standing.Tier = CivilTierManager
	default:
		standing.Tier = CivilTierUser
	}
	return standing
}

// CivilStandingForWallet is the unlocked form, for a caller that holds no lock.
func (l *Lobby) CivilStandingForWallet(wallet string) CivilStanding {
	if l == nil {
		return CivilStanding{Tier: CivilTierUser}
	}
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.CivilStandingForWalletLocked(wallet)
}

// ---------------------------------------------------------------------------
// 6. THE PATH CHOICE — the ONE thing a client may name
// ---------------------------------------------------------------------------

// CareerPathEligibility states whether a wallet may CHOOSE a path, and if not, exactly what is
// missing. A refusal that names its reason is the difference between a locked door and a broken
// one, so this struct is SERVED rather than summarised.
type CareerPathEligibility struct {
	Eligible      bool   `json:"eligible"`
	Reason        string `json:"reason"`
	Territories   int    `json:"territories"`
	RequiredHolds int    `json:"required_territories"`
	RegionOpened  bool   `json:"region_opened"`
	CurrentPath   string `json:"current_path"`
}

// CareerPathTerritoryRequirement is the gate the operator specified: the choice unlocks when the
// player OPENS A REGION, which the engine already refuses below two territories.
const CareerPathTerritoryRequirement = 2

// CareerPathEligibilityForWalletLocked answers the gate. The CALLER HOLDS the lobby lock.
func (l *Lobby) CareerPathEligibilityForWalletLocked(wallet string) CareerPathEligibility {
	standing := l.CivilStandingForWalletLocked(wallet)
	out := CareerPathEligibility{
		Territories:   standing.Territories,
		RequiredHolds: CareerPathTerritoryRequirement,
		RegionOpened:  standing.RegionOpened,
	}
	if stats, ok := l.leaderboard[wallet]; ok && stats.CareerXP != nil {
		out.CurrentPath = PathKey(stats.CareerXP.Path)
	}
	switch {
	case out.CurrentPath != "" && IsCareerPath(out.CurrentPath):
		out.Eligible = false
		out.Reason = "a path is already held; the choice is made once"
	case standing.RegionOpened && standing.Territories >= CareerPathTerritoryRequirement:
		out.Eligible = true
		out.Reason = "region opened with the required territories"
	case standing.Territories < CareerPathTerritoryRequirement:
		out.Reason = fmt.Sprintf("own at least %d territories to open a region (holding %d)",
			CareerPathTerritoryRequirement, standing.Territories)
	default:
		out.Reason = "the region ability has not been opened yet"
	}
	return out
}

// ChooseCareerPathLocked records a wallet's path. The CALLER HOLDS the lobby WRITE lock.
//
// The gate is re-checked HERE, at the domain owner, so no handler, bot or future spawner can set
// a path by going round the door. A second choice is REFUSED rather than overwritten: an alignment
// that can be flipped on demand carries no weight in any of the three matrices.
func (l *Lobby) ChooseCareerPathLocked(wallet, path string) (CareerPathEligibility, error) {
	elig := l.CareerPathEligibilityForWalletLocked(wallet)
	if !IsCareerPath(path) {
		return elig, fmt.Errorf("%q is not a career path (want one of %s)", path, strings.Join(CareerPaths, ", "))
	}
	if !elig.Eligible {
		if elig.CurrentPath != "" {
			return elig, fmt.Errorf("a path is already held (%s); the choice is made once", elig.CurrentPath)
		}
		return elig, fmt.Errorf("a career path cannot be chosen yet: %s", elig.Reason)
	}
	stats, ok := l.leaderboard[wallet]
	if !ok {
		return elig, fmt.Errorf("no engine record for this wallet")
	}
	if stats.CareerXP == nil {
		stats.CareerXP = &CareerXP{RoleXP: make(map[string]uint64)}
	}
	stats.CareerXP.Path = PathKey(path)
	stats.CareerXP.PathChosenAt = time.Now()
	l.leaderboard[wallet] = stats
	elig.CurrentPath = stats.CareerXP.Path
	elig.Eligible = false
	elig.Reason = "a path is already held; the choice is made once"
	l.logAdminAuditLocked("CAREER_PATH_CHOSEN", wallet, fmt.Sprintf("Path: %s (region opened: %t, territories: %d)",
		stats.CareerXP.Path, elig.RegionOpened, elig.Territories))
	return elig, nil
}

// ---------------------------------------------------------------------------
// 7. PROMOTION — the level-cap unlock, plus the civil-rank gate
// ---------------------------------------------------------------------------
//
// "the promotion of careers is in accordance to level cap unlocks … these three roles are career
// gates to force a worker society … as high level careers will need active user management".
//
// A role is promoted when the player has crossed the role's declared LEVEL CAP, holds the role's
// own XP tier, SUSTAINS the $VBV that tier requires, holds the path the role belongs to, AND holds
// the civil rank the role demands. The civil rank is what makes the worker society necessary: a
// role whose authority IS a region (Judge, Underworld Boss, Justice Commissioner) requires a
// governor, a role that directs a shop's operations requires a manager, and a role that only works
// requires a user.
//
// The $VBV requirement is the SAME ladder `GetVBVGatingPermille`/`CheckCareerTierGate` already
// price (VBVTierApprentice 5K → VBVTierBoss 2M) — restated per role FROM those constants so there
// is one ladder, not two.
type CareerRoleGate struct {
	Role              string `json:"role"`
	HomePath          string `json:"home_path"` // justice | criminal
	MinLessonLevel    int    `json:"min_lesson_level"`
	MinRoleTier       int    `json:"min_role_tier"`
	MinSustainedMicro uint64 `json:"min_sustained_micro"`
	MinCivilTier      string `json:"min_civil_tier"`
	Evidence          string `json:"evidence"`
}

// CareerRoleTierMax is the HIGHEST value `getTierFor` can return, and therefore the highest rung a
// gate may require.
//
// MEASURED: `getTierFor` (rival_career_engine.go) bands XP into 1..4 — default 1, ≥500 XP 2,
// ≥3000 3, ≥5000 4 — using the `CareerTier*` constants ×100. NOTE the engine ALSO has
// `CareerXP.GetCareerTier`, which is a DIFFERENT scale (`xp/1500`, unbounded), and
// `CheckCareerTierGate`, which is a third ($VBV, 0..5). Only the 1..4 band is the career-role tier
// this table gates on, and a gate written against one of the other two scales would be silently
// unsatisfiable — an unearnable career with no error anywhere. `assertCareerPathDialect` therefore
// refuses a declared tier above this constant, and
// `TestEveryDeclaredRoleTierIsReachable` proves each one can actually be reached.
const CareerRoleTierMax = 4

// careerRoleGates is THE promotion table. A role ABSENT from this table may be worked but never
// PROMOTED, so an undeclared career fails closed instead of silently unlocking.
//
// The civil-tier column is where the operator's rule lands, and the code already said so before
// this table existed: `AIPathwaySyndicate` (`ai_citizen_engine.go`) is commented
// "UnderworldBoss / Judge (governor-gated)".
var careerRoleGates = []CareerRoleGate{
	// ---- JUSTICE SIDE ----
	{Role: "Warden", HomePath: CareerPathJustice, MinLessonLevel: 10, MinRoleTier: 1, MinSustainedMicro: VBVTierApprentice,
		MinCivilTier: CivilTierUser, Evidence: "detention-line work; no organisation required"},
	{Role: "Forensic Analyst", HomePath: CareerPathJustice, MinLessonLevel: 15, MinRoleTier: 2, MinSustainedMicro: VBVTierJourneyman,
		MinCivilTier: CivilTierUser, Evidence: "P-Forensic analysis (GetEvidenceAccuracyBonus)"},
	{Role: "Bounty Hunter", HomePath: CareerPathJustice, MinLessonLevel: 20, MinRoleTier: 2, MinSustainedMicro: VBVTierJourneyman,
		MinCivilTier: CivilTierUser, Evidence: "a licenced lone trade (BountyLicenseActive)"},
	{Role: "Sector Peacekeeper", HomePath: CareerPathJustice, MinLessonLevel: 25, MinRoleTier: 3, MinSustainedMicro: VBVTierExpert,
		MinCivilTier: CivilTierManager, Evidence: "patrols a SECTOR — commands a territory"},
	{Role: "Justice Recruiter", HomePath: CareerPathJustice, MinLessonLevel: 30, MinRoleTier: 3, MinSustainedMicro: VBVTierExpert,
		MinCivilTier: CivilTierManager, Evidence: "recruits players (GetRecruitmentBonus); needs an org to place them in"},
	{Role: "Intel-Agent", HomePath: CareerPathJustice, MinLessonLevel: 30, MinRoleTier: 3, MinSustainedMicro: VBVTierExpert,
		MinCivilTier: CivilTierManager, Evidence: "runs a network; the cyber-intercept door bills the caller"},
	{Role: "AOS Leader", HomePath: CareerPathJustice, MinLessonLevel: 40, MinRoleTier: 4, MinSustainedMicro: VBVTierMaster,
		MinCivilTier: CivilTierManager, Evidence: "leads advanced operations (P-AOS)"},
	{Role: "Tax Auditor", HomePath: CareerPathJustice, MinLessonLevel: 45, MinRoleTier: 4, MinSustainedMicro: VBVTierMaster,
		MinCivilTier: CivilTierManager, Evidence: "audits other players' organisations — a manager's jurisdiction"},
	{Role: "Mutation Log Auditor", HomePath: CareerPathJustice, MinLessonLevel: 45, MinRoleTier: 4, MinSustainedMicro: VBVTierMaster,
		MinCivilTier: CivilTierManager, Evidence: "audits entities across a region"},
	{Role: "Justice Commissioner", HomePath: CareerPathJustice, MinLessonLevel: 60, MinRoleTier: CareerRoleTierMax, MinSustainedMicro: VBVTierBoss,
		MinCivilTier: CivilTierGovernor, Evidence: "commissioner authority IS a region (P-Commissioner)"},

	// ---- CRIMINAL SIDE ----
	{Role: "Gossip", HomePath: CareerPathCriminal, MinLessonLevel: 10, MinRoleTier: 1, MinSustainedMicro: VBVTierApprentice,
		MinCivilTier: CivilTierUser, Evidence: "sells rumours (GetRumorFeeDiscount) — a lone trade"},
	{Role: "Fence", HomePath: CareerPathCriminal, MinLessonLevel: 15, MinRoleTier: 2, MinSustainedMicro: VBVTierJourneyman,
		MinCivilTier: CivilTierUser, Evidence: "fence fee discount (GetFenceFeeDiscount)"},
	{Role: "Launderer", HomePath: CareerPathCriminal, MinLessonLevel: 20, MinRoleTier: 2, MinSustainedMicro: VBVTierJourneyman,
		MinCivilTier: CivilTierUser, Evidence: "laundering terminal; small-scale by licence"},
	{Role: "Smuggler", HomePath: CareerPathCriminal, MinLessonLevel: 25, MinRoleTier: 3, MinSustainedMicro: VBVTierExpert,
		MinCivilTier: CivilTierManager, Evidence: "transit exemption (GetTransitTaxExemption) needs a routed territory"},
	{Role: "Kidnapper", HomePath: CareerPathCriminal, MinLessonLevel: 30, MinRoleTier: 3, MinSustainedMicro: VBVTierExpert,
		MinCivilTier: CivilTierManager, Evidence: "holds assets; the ransom pressure system needs a base"},
	{Role: "Arc-Net Operative", HomePath: CareerPathCriminal, MinLessonLevel: 30, MinRoleTier: 3, MinSustainedMicro: VBVTierExpert,
		MinCivilTier: CivilTierManager, Evidence: "cyber-intercept (ArcNetActive) — operates from an organisation"},
	{Role: "Saboteur", HomePath: CareerPathCriminal, MinLessonLevel: 35, MinRoleTier: 3, MinSustainedMicro: VBVTierExpert,
		MinCivilTier: CivilTierManager, Evidence: "cloak disruption / sabotage missions (P-Intel)"},
	{Role: "Heist Planner", HomePath: CareerPathCriminal, MinLessonLevel: 45, MinRoleTier: 4, MinSustainedMicro: VBVTierMaster,
		MinCivilTier: CivilTierManager, Evidence: "team heist buff + dividend (GetHeistDividendRate) — commands a crew"},
	{Role: "Judge", HomePath: CareerPathCriminal, MinLessonLevel: 60, MinRoleTier: CareerRoleTierMax, MinSustainedMicro: VBVTierBoss,
		MinCivilTier: CivilTierGovernor, Evidence: "P-Syndicate; aiPathwayByCareer marks it governor-gated"},
	{Role: "Underworld Boss", HomePath: CareerPathCriminal, MinLessonLevel: 60, MinRoleTier: CareerRoleTierMax, MinSustainedMicro: VBVTierBoss,
		MinCivilTier: CivilTierGovernor, Evidence: "P-Syndicate; CONTRACT-029..033 target this role"},
}

// careerRoleGateByKey indexes the table by RoleKey for O(1) lookup, so a caller spelling a role
// differently cannot miss its own gate.
var careerRoleGateByKey = func() map[string]CareerRoleGate {
	m := make(map[string]CareerRoleGate, len(careerRoleGates))
	for _, g := range careerRoleGates {
		m[RoleKey(g.Role)] = g
	}
	return m
}()

// CareerRoleGateFor returns the declared promotion gate for a role (BY ROLE, not spelling).
func CareerRoleGateFor(role string) (CareerRoleGate, bool) {
	g, ok := careerRoleGateByKey[RoleKey(role)]
	return g, ok
}

// CareerPromotionRequirements is what a wallet still lacks for ONE role. It is SERVED, so a locked
// promotion always states the missing piece instead of rendering as "unavailable" — and the
// shortfall is an exact integer, not an adjective.
type CareerPromotionRequirements struct {
	Role              string `json:"role"`
	HomePath          string `json:"home_path"`
	GateDeclared      bool   `json:"gate_declared"`
	Promoted          bool   `json:"promoted"`
	Eligible          bool   `json:"eligible"`
	Missing           string `json:"missing"`
	LessonLevel       int    `json:"lesson_level"`
	MinLessonLevel    int    `json:"min_lesson_level"`
	RoleTier          int    `json:"role_tier"`
	MinRoleTier       int    `json:"min_role_tier"`
	SustainedMicro    uint64 `json:"sustained_micro"`
	MinSustainedMicro uint64 `json:"min_sustained_micro"`
	ShortfallMicro    uint64 `json:"shortfall_micro"`
	CivilTier         string `json:"civil_tier"`
	MinCivilTier      string `json:"min_civil_tier"`
	Path              string `json:"path"`
	PathAllows        bool   `json:"path_allows"`
}

// CareerPromotionRequirementsForWalletLocked evaluates ONE role's gate for a wallet. The CALLER
// HOLDS the lobby lock. It NEVER mutates, so it is safe to call from a read path.
func (l *Lobby) CareerPromotionRequirementsForWalletLocked(wallet, role string) CareerPromotionRequirements {
	standing := l.CivilStandingForWalletLocked(wallet)
	out := CareerPromotionRequirements{
		Role:         role,
		HomePath:     CareerPathOfRole(role),
		CivilTier:    standing.Tier,
		MinCivilTier: CivilTierUser,
	}
	gate, declared := CareerRoleGateFor(role)
	out.GateDeclared = declared
	if declared {
		out.Role = gate.Role
		out.HomePath = gate.HomePath
		out.MinLessonLevel = gate.MinLessonLevel
		out.MinRoleTier = gate.MinRoleTier
		out.MinSustainedMicro = gate.MinSustainedMicro
		out.MinCivilTier = gate.MinCivilTier
	}
	stats, ok := l.leaderboard[wallet]
	if !ok || stats.CareerXP == nil {
		out.Missing = "no engine career record for this wallet"
		return out
	}
	cxp := stats.CareerXP
	out.Path = PathKey(cxp.Path)
	out.PathAllows = PathAllowsRole(cxp.Path, role)
	out.LessonLevel = cxp.LessonLevel
	out.RoleTier = getTierFor(cxp, role)
	out.SustainedMicro = cxp.AvgSustainedMicro
	out.Promoted = CareerHasRole(cxp, role)
	if !declared {
		out.Missing = "no promotion gate is declared for this career, so it cannot be promoted (it can still be worked)"
		return out
	}
	switch {
	case out.Promoted:
		out.Missing = ""
	case out.Path == "" || !IsCareerPath(out.Path):
		out.Missing = "choose a career path first (open a region while holding " +
			fmt.Sprint(CareerPathTerritoryRequirement) + " territories)"
	case !out.PathAllows:
		out.Missing = fmt.Sprintf("this career belongs to the %s path and your path is %s", out.HomePath, out.Path)
	case out.LessonLevel < out.MinLessonLevel:
		out.Missing = fmt.Sprintf("reach level %d (currently %d)", out.MinLessonLevel, out.LessonLevel)
	case out.RoleTier < out.MinRoleTier:
		out.Missing = fmt.Sprintf("reach tier %d in this career (currently %d)", out.MinRoleTier, out.RoleTier)
	case out.SustainedMicro < out.MinSustainedMicro:
		out.ShortfallMicro = out.MinSustainedMicro - out.SustainedMicro
		out.Missing = fmt.Sprintf("sustain %s $VBV (currently %s — %s short)",
			formatMicroShort(out.MinSustainedMicro), formatMicroShort(out.SustainedMicro), formatMicroShort(out.ShortfallMicro))
	case civilRankIndex(standing.Tier) < civilRankIndex(out.MinCivilTier):
		out.Missing = fmt.Sprintf("hold the %s rank (currently %s): %s", out.MinCivilTier, standing.Tier,
			civilTierRequirementHint(out.MinCivilTier))
	default:
		out.Eligible = true
	}
	return out
}

// formatMicroShort renders micro-$VBV as whole units for a refusal message. Integer-only: the
// VALUE never touches a float, and neither does its display.
func formatMicroShort(micro uint64) string {
	return fmt.Sprintf("%d", micro/1_000_000)
}

// civilTierRequirementHint states what a rank requires, so a locked promotion tells the player the
// ACTION that unlocks it rather than naming a rank and leaving them to guess.
func civilTierRequirementHint(tier string) string {
	switch PathKey(tier) {
	case CivilTierGovernor:
		return "own a club and OPEN ITS REGION (which needs at least 2 territories)"
	case CivilTierManager:
		return "hold at least one territory"
	default:
		return "no organisation is required"
	}
}

// PromoteCareerLocked grants a role when every gate passes. The CALLER HOLDS the lobby WRITE lock.
//
// This is the WRITER `PromotedRoles` never had. It is deliberately narrow: it refuses an
// undeclared career, a path the role does not belong to, an unmet level cap, an unearned tier, an
// unsustainable balance and an insufficient civil rank — each naming the ONE that failed — and it
// is IDEMPOTENT, so a second call cannot duplicate the entry or re-time the promotion.
func (l *Lobby) PromoteCareerLocked(wallet, role string) (CareerPromotionRequirements, error) {
	req := l.CareerPromotionRequirementsForWalletLocked(wallet, role)
	if req.Promoted {
		return req, nil // idempotent: already promoted, nothing changes
	}
	if !req.Eligible {
		return req, fmt.Errorf("%s cannot be promoted: %s", req.Role, req.Missing)
	}
	stats, ok := l.leaderboard[wallet]
	if !ok || stats.CareerXP == nil {
		return req, fmt.Errorf("no engine career record for this wallet")
	}
	stats.CareerXP.PromotedRoles = append(stats.CareerXP.PromotedRoles, req.Role)
	stats.CareerXP.LastPromotionAt = time.Now()
	// THE UNLOCK NOTICE IS CLEARED for the role now held, so the notice re-arms: a career lost to a
	// demotion and unlocked again genuinely re-notifies instead of being suppressed forever by a
	// stale "already told" entry.
	delete(stats.CareerXP.UnlockNoticesSent, RoleKey(req.Role))
	stats.JobRole = req.Role
	l.leaderboard[wallet] = stats
	req.Promoted = true
	req.Eligible = false
	req.Missing = ""
	l.logAdminAuditLocked("CAREER_PROMOTED", wallet, fmt.Sprintf("Role: %s (path: %s, level: %d, sustained: %d micro)",
		req.Role, req.Path, req.LessonLevel, req.SustainedMicro))
	return req, nil
}

// ---------------------------------------------------------------------------
// 7.5 THE UNLOCK ADVISORY AND THE STAFF REQUEST — "unlocked, never forced"
// ---------------------------------------------------------------------------
//
// THE OPERATOR'S RULE, verbatim: "a user does not have to upgrade career it is only unlocked to
// upgrade if level cap allows it, a user may request staff users to upgrade when they notice there
// staff can upgrade and it is upto the user to upgrade them seflves, yes they may be notified not
// forced."
//
// Four binding consequences, and each one constrains what this code is ALLOWED to do:
//
//  1. NOT MANDATORY. `PromoteCareerLocked` is the only writer of a career and only the player calls
//     it. Nothing in this repository promotes automatically, and nothing may be added that does: an
//     unlock is an OFFER. `IsOptional` is SERVED so the client cannot render it as an obligation.
//  2. UNLOCKED BY THE LEVEL CAP. `Eligible` already means "every gate passes" — the level cap among
//     them. This section gives that state the operator's own word, UNLOCKED, and serves the list of
//     roles in it, so "what may I upgrade?" is answerable without the client re-deriving a gate.
//  3. STAFF MAY BE ASKED. An employer may SEE which of their staff can upgrade (the staff
//     projection below) and may REQUEST that they do. The request lives on the EMPLOYER's own club
//     record and never writes the staff member's career — so an employer can ask, and only the
//     employee can act.
//  4. NOTIFIED, NOT FORCED. A newly unlocked career produces ONE notice, to the ONE wallet it
//     concerns, once (see `CareerUnlockNoticesLocked`) — never a broadcast, never a repeated nag.
//
// WHERE THE UNLOCK IS ENFORCED. `RequestStaffCareerUpgradeLocked` refuses a request for an upgrade
// the level cap has NOT opened, quoting the EMPLOYEE's own missing gate. That is what makes the cap
// real in both directions: it gates the employee's own promotion AND the employer's ability to ask.
// ---------------------------------------------------------------------------

// CareerUpgradeStatement is the operator's rule, SERVED rather than written into the client so the
// two can never disagree about whether an upgrade is mandatory.
const CareerUpgradeStatement = "A career is UNLOCKED to upgrade and never granted: the level cap opens it, the upgrade is yours to take, and nothing here is forced."

// CareerUnlockBasisStatement names what opens an unlock — the SAME gate the career table states.
const CareerUnlockBasisStatement = "the level cap, the role tier, the sustained $VBV, the civil rank and the path — the same gate the career table states, read from ONE evaluator"

// CareerStaffBasisStatement names where "staff" comes from, so the projection is never a guess.
const CareerStaffBasisStatement = "staff = the wallets on the Staff roster of a club YOU own (Club.OwnerWallet -> Club.Staff; the same relationship PlayerStats.EmployerClubID records)"

// CareerNoticeRuleStatement states the notification rule, so the client does not invent one.
const CareerNoticeRuleStatement = "an unlock is notified once, to the wallet it concerns, and never broadcast; a promotion that is not unlocked is refused with the one missing gate, never queued and never applied"

// CareerStaffUpgradeRequest is an EMPLOYER's request that a staff member take an upgrade they are
// ALREADY UNLOCKED for. It is stored on the employer's club (`Club.StaffUpgradeRequests`) and it is
// a REQUEST: the career it names is untouched until the staff member promotes themselves.
type CareerStaffUpgradeRequest struct {
	StaffWallet string    `json:"staff_wallet"`
	Role        string    `json:"role"`
	RequestedAt time.Time `json:"requested_at"`
}

// CareerStaffView is ONE employee of a club the caller owns, and what they may upgrade.
type CareerStaffView struct {
	Wallet        string    `json:"wallet"`
	ClubID        string    `json:"club_id"`
	ClubRole      string    `json:"club_role"`
	JobRole       string    `json:"job_role"`
	LessonLevel   int       `json:"lesson_level"`
	PromotedRoles []string  `json:"promoted_roles"`
	UnlockedRoles []string  `json:"unlocked_roles"`
	UnlockCount   int       `json:"unlock_count"`
	UpgradeReady  bool      `json:"upgrade_ready"`
	RequestedRole string    `json:"requested_role,omitempty"`
	RequestedAt   time.Time `json:"requested_at,omitempty"`
}

// CareerStaffRequestNotice is a request as the STAFF MEMBER sees it. A request always names the one
// wallet it concerns, so an employee sees their own employer's request and never another player's.
type CareerStaffRequestNotice struct {
	EmployerWallet string    `json:"employer_wallet"`
	ClubID         string    `json:"club_id"`
	ClubName       string    `json:"club_name"`
	StaffWallet    string    `json:"staff_wallet"`
	Role           string    `json:"role"`
	Message        string    `json:"message"`
	RequestedAt    time.Time `json:"requested_at"`
}

// CareerUpgradeView is the ADVISORY half of the career system: what is unlocked to upgrade right
// now, and who the caller employs. Nothing in it is a write, and nothing in it is a decision the
// player has already been made to take.
type CareerUpgradeView struct {
	Statement       string                     `json:"statement"`
	IsOptional      bool                       `json:"is_optional"`
	UnlockBasis     string                     `json:"unlock_basis"`
	NoticeRule      string                     `json:"notice_rule"`
	UnlockedRoles   []string                   `json:"unlocked_roles"`
	UnlockedCount   int                        `json:"unlocked_count"`
	StaffBasis      string                     `json:"staff_basis"`
	Staff           []CareerStaffView          `json:"staff"`
	StaffReadyCount int                        `json:"staff_ready_count"`
	CanRequestStaff bool                       `json:"can_request_staff"`
	RequestsToMe    []CareerStaffRequestNotice `json:"requests_to_me"`
}

// CareerUnlockNotice is the ONE notice a newly unlocked career produces. It is RETURNED rather than
// sent so the rule is testable without a WebSocket, and the caller owns delivery (the same shape
// `CareerStandingEvent` already uses).
type CareerUnlockNotice struct {
	Kind        string    `json:"kind"`
	Wallet      string    `json:"wallet"`
	Role        string    `json:"role"`
	HomePath    string    `json:"home_path"`
	UnlockBasis string    `json:"unlock_basis"`
	Message     string    `json:"message"`
	At          time.Time `json:"at"`
}

// orderedCareerGates returns the declared promotion table in ONE deterministic order (path, then
// level cap, then name), so the served career table and the unlocked list cannot disagree about
// which career comes first. The served table and the advisory both read this, not two sorts.
func orderedCareerGates() []CareerRoleGate {
	ordered := make([]CareerRoleGate, len(careerRoleGates))
	copy(ordered, careerRoleGates)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].HomePath != ordered[j].HomePath {
			return ordered[i].HomePath < ordered[j].HomePath
		}
		if ordered[i].MinLessonLevel != ordered[j].MinLessonLevel {
			return ordered[i].MinLessonLevel < ordered[j].MinLessonLevel
		}
		return ordered[i].Role < ordered[j].Role
	})
	return ordered
}

// UnlockedCareerRolesLocked returns the declared careers a wallet is UNLOCKED to upgrade into right
// now — the operator's "only unlocked to upgrade if the level cap allows it", made mechanical. It
// asks the ONE gate evaluator, so the unlock and the served requirement cannot drift apart.
//
// IT GRANTS NOTHING. There is no queue, no deadline and no automatic action anywhere: the player
// takes the upgrade through `PromoteCareerLocked`, or does not.
func (l *Lobby) UnlockedCareerRolesLocked(wallet string) []string {
	out := []string{}
	for _, g := range orderedCareerGates() {
		req := l.CareerPromotionRequirementsForWalletLocked(wallet, g.Role)
		if req.Eligible && !req.Promoted {
			out = append(out, req.Role)
		}
	}
	return out
}

// OwnedClubsForWalletLocked returns the clubs a wallet OWNS, in deterministic ID order. "Staff" is
// defined from these and from nothing else: an employer is a club owner, which is the relationship
// the engine already records (`Club.OwnerWallet` → `Club.Staff`, and `PlayerStats.EmployerClubID`
// on the other side of it).
func (l *Lobby) OwnedClubsForWalletLocked(wallet string) []*Club {
	out := []*Club{}
	if l == nil || wallet == "" {
		return out
	}
	ids := make([]string, 0, len(l.clubs))
	for id, club := range l.clubs {
		if club == nil || !strings.EqualFold(club.OwnerWallet, wallet) {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		out = append(out, l.clubs[id])
	}
	return out
}

// staffUpgradeRequestKey keys a request by the FOLDED wallet and the FOLDED role, so the same
// request cannot be recorded twice under two spellings — the same rule `RoleKey` already applies to
// the career vocabulary.
func staffUpgradeRequestKey(staffWallet, role string) string {
	return RoleKey(staffWallet) + ":" + RoleKey(role)
}

// staffRequestClubsForWalletLocked returns the clubs that may hold an upgrade request ADDRESSED TO
// this wallet: the club its own employment record names, plus any club whose request records name
// this wallet. The CALLER HOLDS the lobby lock.
//
// WHY THE SECOND SOURCE. A hire writes BOTH `PlayerStats.EmployerClubID` and `Club.Staff`, but a
// roster row can outlive a cleared field, and a request that exists but never surfaces is a
// notification that silently failed — this repository's recurring defect class. So the employment
// record is tried first and a roster scan backs it up. The scan is bounded by the clubs that
// actually HOLD a request (a map-length check per club, which is a comparison, not a walk), so the
// common case costs one length check per club and never touches a roster.
func (l *Lobby) staffRequestClubsForWalletLocked(wallet string) []*Club {
	seen := map[string]bool{}
	out := []*Club{}
	add := func(club *Club) {
		if club == nil || seen[club.ID] {
			return
		}
		seen[club.ID] = true
		out = append(out, club)
	}
	// 1. The authoritative record: the club the wallet's own employment row names.
	if stats, ok := l.leaderboard[l.leaderboardKeyLocked(wallet)]; ok && stats.EmployerClubID != "" {
		if club, ok := l.clubs[stats.EmployerClubID]; ok {
			add(club)
		}
	}
	// 2. The backstop: any club holding a request for this wallet, in deterministic club order.
	ids := make([]string, 0, len(l.clubs))
	for id, club := range l.clubs {
		if club == nil || len(club.StaffUpgradeRequests) == 0 {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		club := l.clubs[id]
		for _, rec := range club.StaffUpgradeRequests {
			if RoleKey(rec.StaffWallet) == RoleKey(wallet) {
				add(club)
				break
			}
		}
	}
	return out
}

// careerStaffLatestRequestLocked returns the MOST RECENT request recorded for one staff wallet, or
// nil. The CALLER HOLDS the lobby lock. A staff member may hold requests for more than one role, so
// which one is shown must be DECIDED rather than left to map order: latest wins, ties broken by the
// role name, so a refresh never looks like a change.
func careerStaffLatestRequestLocked(club *Club, staffWallet string) *CareerStaffUpgradeRequest {
	if club == nil || len(club.StaffUpgradeRequests) == 0 {
		return nil
	}
	var best *CareerStaffUpgradeRequest
	for _, rec := range club.StaffUpgradeRequests {
		if RoleKey(rec.StaffWallet) != RoleKey(staffWallet) {
			continue
		}
		r := rec
		if best == nil || r.RequestedAt.After(best.RequestedAt) ||
			(r.RequestedAt.Equal(best.RequestedAt) && r.Role < best.Role) {
			best = &r
		}
	}
	return best
}

// careerStaffViewForLocked describes ONE employee of a club the caller owns. The CALLER HOLDS the
// lobby lock, and NOTHING here mutates: the staff member's own record is READ, never written.
func (l *Lobby) careerStaffViewForLocked(club *Club, staffWallet string) CareerStaffView {
	view := CareerStaffView{
		Wallet:        staffWallet,
		ClubID:        club.ID,
		ClubRole:      club.Staff[staffWallet],
		PromotedRoles: []string{},
		UnlockedRoles: []string{},
	}
	// The roster spelling and the stored spelling can differ (a hire lower-cases the wallet), so the
	// record is resolved case-insensitively like every other wallet-keyed read here.
	key := l.leaderboardKeyLocked(staffWallet)
	if stats, ok := l.leaderboard[key]; ok {
		view.JobRole = stats.JobRole
		if stats.CareerXP != nil {
			view.LessonLevel = stats.CareerXP.LessonLevel
			if stats.CareerXP.PromotedRoles != nil {
				view.PromotedRoles = append(view.PromotedRoles, stats.CareerXP.PromotedRoles...)
			}
		}
	}
	view.UnlockedRoles = l.UnlockedCareerRolesLocked(key)
	view.UnlockCount = len(view.UnlockedRoles)
	// "When they notice their staff can upgrade" — this flag is that notice, derived, not declared.
	view.UpgradeReady = view.UnlockCount > 0
	if req := careerStaffLatestRequestLocked(club, staffWallet); req != nil {
		view.RequestedRole = req.Role
		view.RequestedAt = req.RequestedAt
	}
	return view
}

// CareerUpgradeViewLocked assembles the advisory. The CALLER HOLDS the lobby lock.
//
// IT NEVER MUTATES — no request is recorded, no notice is marked sent and no career is touched.
// That is what makes it safe on the read path, and what makes "not forced" checkable: two reads in
// a row return the same thing whether or not the player acted.
func (l *Lobby) CareerUpgradeViewLocked(wallet string) CareerUpgradeView {
	view := CareerUpgradeView{
		Statement:    CareerUpgradeStatement,
		IsOptional:   true,
		UnlockBasis:  CareerUnlockBasisStatement,
		NoticeRule:   CareerNoticeRuleStatement,
		StaffBasis:   CareerStaffBasisStatement,
		Staff:        []CareerStaffView{},
		RequestsToMe: []CareerStaffRequestNotice{},
	}
	view.UnlockedRoles = l.UnlockedCareerRolesLocked(wallet)
	view.UnlockedCount = len(view.UnlockedRoles)

	for _, club := range l.OwnedClubsForWalletLocked(wallet) {
		view.CanRequestStaff = true
		// Deterministic roster order: `Staff` is a map, and a list that reshuffles on every refresh
		// reads as a change.
		roster := make([]string, 0, len(club.Staff))
		for w := range club.Staff {
			roster = append(roster, w)
		}
		sort.Strings(roster)
		for _, w := range roster {
			sv := l.careerStaffViewForLocked(club, w)
			if sv.UpgradeReady {
				view.StaffReadyCount++
			}
			view.Staff = append(view.Staff, sv)
		}
	}

	// REQUESTS ADDRESSED TO THE CALLER. A request names exactly one wallet, so an employee sees
	// their own employer's request and never another player's — there is no query parameter and no
	// key a caller could point at somebody else.
	for _, club := range l.staffRequestClubsForWalletLocked(wallet) {
		keys := make([]string, 0, len(club.StaffUpgradeRequests))
		for k := range club.StaffUpgradeRequests {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			rec := club.StaffUpgradeRequests[k]
			if RoleKey(rec.StaffWallet) != RoleKey(wallet) {
				continue
			}
			view.RequestsToMe = append(view.RequestsToMe, CareerStaffRequestNotice{
				EmployerWallet: club.OwnerWallet,
				ClubID:         club.ID,
				ClubName:       club.Name,
				StaffWallet:    wallet,
				Role:           rec.Role,
				RequestedAt:    rec.RequestedAt,
				Message: fmt.Sprintf("%s has asked you to take the %s upgrade — you are unlocked for it, and it is yours to take or leave.",
					club.Name, rec.Role),
			})
		}
	}
	return view
}

// RequestStaffCareerUpgradeLocked records an EMPLOYER's request that one of their staff take an
// upgrade they are ALREADY UNLOCKED for, and notifies that staff member. The CALLER HOLDS the lobby
// WRITE lock.
//
// IT IS NOT A PROMOTION AND IT CANNOT BECOME ONE. No outcome of this function writes the staff
// member's `PromotedRoles`, `JobRole`, `LessonLevel` or path: the request is stored on the
// EMPLOYER's own club record and delivered as a notification. That is the whole of "not forced",
// mechanically — an employer can ask, and only the employee can act.
//
// SEVEN REFUSALS, each naming what is wrong, and every one of them takes no action:
//  1. the request names no staff wallet
//  2. the caller IS the target        -> take it yourself; a self-request is an action-shaped no-op
//  3. the role has no declared gate   -> it can be worked but never promoted into
//  4. the caller owns no club         -> they have no staff to ask
//  5. the target is not on a roster   -> not your staff
//  6. the role is already held        -> nothing is left to upgrade
//  7. THE UNLOCK IS NOT MET           -> the EMPLOYEE's own missing gate, quoted exactly
//
// (7) is what makes the level cap REAL in the second direction: an employer cannot ask for an
// upgrade the level cap has not opened, so a request can never become a way round the gate.
func (l *Lobby) RequestStaffCareerUpgradeLocked(ownerWallet, staffWallet, role string) (CareerStaffRequestNotice, error) {
	notice := CareerStaffRequestNotice{Role: role, StaffWallet: staffWallet}
	if staffWallet == "" {
		return notice, fmt.Errorf("a request needs the staff wallet it is addressed to")
	}
	if RoleKey(ownerWallet) == RoleKey(staffWallet) {
		return notice, fmt.Errorf("that wallet is you: take the upgrade yourself instead of requesting it from yourself")
	}
	if _, declared := CareerRoleGateFor(role); !declared {
		return notice, fmt.Errorf("no promotion gate is declared for %q, so it cannot be upgraded into", role)
	}
	owned := l.OwnedClubsForWalletLocked(ownerWallet)
	if len(owned) == 0 {
		return notice, fmt.Errorf("you own no club, so there is no staff for you to request an upgrade from")
	}
	var club *Club
	var rosterWallet string
	for _, c := range owned {
		for w := range c.Staff {
			if RoleKey(w) == RoleKey(staffWallet) {
				club = c
				rosterWallet = w
				break
			}
		}
		if club != nil {
			break
		}
	}
	if club == nil {
		return notice, fmt.Errorf("%s is not on the staff of any club you own", staffWallet)
	}

	// THE UNLOCK, EVALUATED ON THE EMPLOYEE'S OWN RECORD. This is the gate that cannot be bypassed
	// by asking: the level cap opens the upgrade for them, and only then may anyone suggest it.
	empKey := l.leaderboardKeyLocked(rosterWallet)
	req := l.CareerPromotionRequirementsForWalletLocked(empKey, role)
	if req.Promoted {
		return notice, fmt.Errorf("%s already holds %s — there is nothing to upgrade", staffWallet, req.Role)
	}
	if !req.Eligible {
		return notice, fmt.Errorf("%s is not unlocked for %s yet: %s", staffWallet, req.Role, req.Missing)
	}

	if club.StaffUpgradeRequests == nil {
		club.StaffUpgradeRequests = make(map[string]CareerStaffUpgradeRequest)
	}
	key := staffUpgradeRequestKey(rosterWallet, req.Role)
	if existing, already := club.StaffUpgradeRequests[key]; already {
		// ONE request per unlock. A repeat is REPORTED rather than re-sent: a nudge that can be
		// re-sent on demand is not a notification, it is pressure.
		return CareerStaffRequestNotice{
			EmployerWallet: club.OwnerWallet, ClubID: club.ID, ClubName: club.Name,
			StaffWallet: existing.StaffWallet, Role: existing.Role, RequestedAt: existing.RequestedAt,
			Message: fmt.Sprintf("%s was already asked to take the %s upgrade; the request stands and nothing is repeated.",
				existing.StaffWallet, existing.Role),
		}, nil
	}

	at := time.Now()
	club.StaffUpgradeRequests[key] = CareerStaffUpgradeRequest{
		StaffWallet: rosterWallet, Role: req.Role, RequestedAt: at,
	}
	l.clubs[club.ID] = club
	l.logAdminAuditLocked("CAREER_STAFF_UPGRADE_REQUESTED", rosterWallet, fmt.Sprintf(
		"Employer: %s, Club: %s (%s), Role: %s (unlocked, not taken)", ownerWallet, club.Name, club.ID, req.Role))

	notice = CareerStaffRequestNotice{
		EmployerWallet: club.OwnerWallet, ClubID: club.ID, ClubName: club.Name,
		StaffWallet: rosterWallet, Role: req.Role, RequestedAt: at,
		Message: fmt.Sprintf("%s has asked you to take the %s upgrade — you are unlocked for it, and it is yours to take or leave.",
			club.Name, req.Role),
	}
	l.NotifyCareerStaffRequestLocked(rosterWallet, notice)
	return notice, nil
}

// NotifyCareerStaffRequestLocked delivers a staff-upgrade request to the ONE wallet it concerns.
// The CALLER HOLDS the lobby lock.
//
// An offline staff member loses nothing: the request is STORED on the club and the panel renders it
// from `requests_to_me` on the next read. Nothing is queued, retried or escalated — a request that
// nobody acts on simply stays a request.
func (l *Lobby) NotifyCareerStaffRequestLocked(staffWallet string, notice CareerStaffRequestNotice) {
	if l == nil {
		return
	}
	cid := l.getClientIDFromWalletLocked(staffWallet)
	if cid == "" {
		return
	}
	payload, err := json.Marshal(notice)
	if err != nil {
		return
	}
	l.sendToClientLocked(cid, Envelope{Type: "career_upgrade_requested", Payload: json.RawMessage(payload)})
}

// CareerUnlockNoticesLocked returns ONE notice for every career the wallet has just become
// UNLOCKED for that it has not already been told about, and RECORDS that they were told. The CALLER
// HOLDS the lobby WRITE lock.
//
// IT NOTIFIES AND NOTHING ELSE. There is no promotion here, no queue, no deadline and no
// escalation: the operator asked for a notification, not an obligation. Two properties make that
// real rather than stated:
//
//   - the notice fires ONCE per unlock (the record of it is on the career record itself), so it
//     cannot become a nag;
//   - the record is CLEARED when the role is promoted (`PromoteCareerLocked`), so a career lost to
//     a demotion and unlocked again genuinely re-notifies instead of being silent forever.
//
// CADENCE. This runs where the balance is sampled (the 24h liquidity daemon, beside the standing
// events). One of the gates IS the sustained balance, so a notice derived from the sample is at
// most one sampling window late — that is the honest bound, and it is stated rather than implied.
func (l *Lobby) CareerUnlockNoticesLocked(wallet string) []CareerUnlockNotice {
	out := []CareerUnlockNotice{}
	stats, ok := l.leaderboard[wallet]
	if !ok || stats.CareerXP == nil {
		return out
	}
	unlocked := l.UnlockedCareerRolesLocked(wallet)
	if len(unlocked) == 0 {
		return out
	}
	cxp := stats.CareerXP
	if cxp.UnlockNoticesSent == nil {
		cxp.UnlockNoticesSent = make(map[string]time.Time)
	}
	now := time.Now()
	for _, role := range unlocked {
		key := RoleKey(role)
		if _, told := cxp.UnlockNoticesSent[key]; told {
			continue
		}
		cxp.UnlockNoticesSent[key] = now
		gate, _ := CareerRoleGateFor(role)
		out = append(out, CareerUnlockNotice{
			Kind:        "unlocked",
			Wallet:      wallet,
			Role:        role,
			HomePath:    gate.HomePath,
			UnlockBasis: CareerUnlockBasisStatement,
			At:          now,
			Message: fmt.Sprintf("UNLOCKED: %s (level %d, tier %d, %s sustained, %s rank). The upgrade is yours to take — nothing is applied automatically.",
				role, gate.MinLessonLevel, gate.MinRoleTier, formatMicroShort(gate.MinSustainedMicro), gate.MinCivilTier),
		})
	}
	if len(out) > 0 {
		l.leaderboard[wallet] = stats
	}
	return out
}

// NotifyCareerUnlockLocked delivers an unlock notice to the ONE wallet it concerns. The CALLER HOLDS
// the lobby lock.
//
// The guard is the ADDRESS, not a broadcast: this is the same defect the standing events had (their
// `lwb == walletLower || cid != ""` was always true, so every player's demotion went to every
// client). An offline player misses nothing — the unlock is DERIVED on every read, so the panel
// shows it whether or not a notice ever landed.
func (l *Lobby) NotifyCareerUnlockLocked(ev CareerUnlockNotice) {
	if l == nil {
		return
	}
	cid := l.getClientIDFromWalletLocked(ev.Wallet)
	if cid == "" {
		return
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		return
	}
	l.sendToClientLocked(cid, Envelope{Type: "career_unlocked", Payload: json.RawMessage(payload)})
}

// ---------------------------------------------------------------------------
// 8. DEMOTION — the $VBV drain that takes the career with it
// ---------------------------------------------------------------------------
//
// "when a user drains their level from cashing out it drains their career opportunities and will
// warn them to make the $VBV back to remain employable in the career that they have promoted to,
// or they will be demoted accordingly."
//
// The engine already SAMPLES the balance (`StartLiquiditySamplingDaemon` → `LiquiditySamples` →
// `AvgSustainedMicro`) and already knows the ladder a career requires. What it did NOT do is act:
// demotion was warning-only, and the warning was broadcast to every connected client because of an
// always-true condition in `server.go`. This is the owner of the real lifecycle, and the three
// outcomes are named:
//
//	warning  — the sustained $VBV fell below what the promoted career requires. Nothing is taken
//	           yet; the player is told the EXACT shortfall and the deadline to earn it back.
//	cleared  — the balance recovered inside the grace period, so employability is restored and the
//	           warning is withdrawn. (Without this a recovered player stayed permanently failed.)
//	demoted  — the grace period expired with the shortfall still open, so the career is removed.
//	           `PromotedRoles` IS the grant, so removing it IS the demotion, and the reason is kept.
const CareerDemotionReasonDrain = "sustained $VBV fell below the career's requirement and the grace period expired"

// CareerDemotion is an auditable record of a career being taken away, kept with the exact integer
// arithmetic that caused it so the decision can be re-checked later rather than trusted.
type CareerDemotion struct {
	Role           string    `json:"role"`
	RequiredMicro  uint64    `json:"required_micro"`
	AvgMicro       uint64    `json:"avg_micro"`
	ShortfallMicro uint64    `json:"shortfall_micro"`
	Reason         string    `json:"reason"`
	DemotedAt      time.Time `json:"demoted_at"`
}

// CareerStandingEvent is one outcome of the standing evaluation. It is RETURNED rather than sent,
// so the lifecycle is testable without a WebSocket and the caller owns delivery.
type CareerStandingEvent struct {
	Kind           string    `json:"kind"` // warning | cleared | demoted
	Wallet         string    `json:"wallet"`
	Role           string    `json:"role"`
	RequiredMicro  uint64    `json:"required_micro"`
	AvgMicro       uint64    `json:"avg_micro"`
	ShortfallMicro uint64    `json:"shortfall_micro"`
	GraceEndsAt    time.Time `json:"grace_ends_at,omitempty"`
	Message        string    `json:"message"`
	At             time.Time `json:"at"`
}

// careerDemotionGrace is the window a player has to restore the balance. It reuses the declared
// `DemotionGracePeriodDays` so there is ONE grace period in the engine, not two.
func careerDemotionGrace() time.Duration {
	return time.Duration(DemotionGracePeriodDays) * 24 * time.Hour
}

// careerDemotionWarningLocked reconstructs the OUTSTANDING warning from the record, for the served
// view. The CALLER HOLDS the lobby lock.
//
// It computes the shortfall DIRECTLY against the recorded role's gate rather than reading it from
// `CareerPromotionRequirementsForWalletLocked`: that function's answer short-circuits on "already
// promoted" (the role IS held, which is why a warning about it exists), so it reports no shortfall
// at all — and a warning that states a balance problem without the amount is the "figure could not
// be read" defect this repository has already had to repair once.
func careerDemotionWarningLocked(wallet string, cxp *CareerXP) *CareerStandingEvent {
	if cxp == nil || cxp.DemotionWarningAt.IsZero() || cxp.DemotionWarningRole == "" {
		return nil
	}
	gate, declared := CareerRoleGateFor(cxp.DemotionWarningRole)
	required := uint64(0)
	if declared {
		required = gate.MinSustainedMicro
	}
	avg := cxp.AvgSustainedMicro
	shortfall := uint64(0)
	if avg < required {
		shortfall = required - avg
	}
	return &CareerStandingEvent{
		Kind:           "warning",
		Wallet:         wallet,
		Role:           cxp.DemotionWarningRole,
		RequiredMicro:  required,
		AvgMicro:       avg,
		ShortfallMicro: shortfall,
		GraceEndsAt:    cxp.DemotionWarningAt.Add(careerDemotionGrace()),
		Message: fmt.Sprintf("%s needs %s $VBV sustained (you hold %s — %s short). Restore it within %d days or the career is demoted.",
			cxp.DemotionWarningRole, formatMicroShort(required), formatMicroShort(avg),
			formatMicroShort(shortfall), DemotionGracePeriodDays),
		At: cxp.DemotionWarningAt,
	}
}

// EvaluateCareerStandingLocked audits every promoted role against the sustained balance and applies
// the warning → grace → demotion lifecycle. The CALLER HOLDS the lobby WRITE lock.
//
// It RETURNS the events it produced (possibly none) so the caller owns delivery; it never sends.
// Only the role with the HIGHEST unmet requirement is warned at a time, because `CareerXP.
// DemotionWarningAt` is ONE clock per player and warning about several careers off one clock would
// misreport each of their deadlines.
func (l *Lobby) EvaluateCareerStandingLocked(wallet string) []CareerStandingEvent {
	stats, ok := l.leaderboard[wallet]
	if !ok || stats.CareerXP == nil {
		return nil
	}
	cxp := stats.CareerXP
	if len(cxp.PromotedRoles) == 0 {
		return nil
	}
	now := time.Now()
	avg := cxp.AvgSustainedMicro

	// Find the worst shortfall among the promoted roles that HAVE a declared gate.
	var worst *CareerRoleGate
	var worstShortfall uint64
	judged := 0
	for _, role := range cxp.PromotedRoles {
		gate, declared := CareerRoleGateFor(role)
		if !declared {
			continue // an undeclared career cannot be judged, so it is never silently demoted
		}
		judged++
		if avg >= gate.MinSustainedMicro {
			continue
		}
		shortfall := gate.MinSustainedMicro - avg
		if worst == nil || shortfall > worstShortfall {
			g := gate
			worst = &g
			worstShortfall = shortfall
		}
	}

	// NOTHING JUDGEABLE. A record whose promoted roles all lack a declared gate has no requirement
	// to fall below, so the lifecycle must do NOTHING — not even withdraw a warning. Emitting
	// "sustained $VBV restored" here would be a FALSE STATEMENT about a balance this function has
	// no basis to judge.
	if judged == 0 {
		return nil
	}

	if worst == nil {
		// Every promoted role is funded. If a warning was outstanding, the player earned it back
		// and the warning is WITHDRAWN — employability restored.
		if !cxp.DemotionWarningAt.IsZero() {
			role := cxp.DemotionWarningRole
			cxp.DemotionWarningAt = time.Time{}
			cxp.DemotionWarningRole = ""
			l.leaderboard[wallet] = stats
			return []CareerStandingEvent{{
				Kind:     "cleared",
				Wallet:   wallet,
				Role:     role,
				AvgMicro: avg,
				Message:  "sustained $VBV restored — the career remains employable",
				At:       now,
			}}
		}
		return nil
	}

	// A role is unfunded. Has the grace period already run out?
	graceExpired := !cxp.DemotionWarningAt.IsZero() &&
		cxp.DemotionWarningRole != "" &&
		RoleKey(cxp.DemotionWarningRole) == RoleKey(worst.Role) &&
		now.Sub(cxp.DemotionWarningAt) >= careerDemotionGrace()

	if !graceExpired {
		// First time below (or a still-running warning for this same role): WARN, do not act.
		firstWarning := cxp.DemotionWarningAt.IsZero() ||
			RoleKey(cxp.DemotionWarningRole) != RoleKey(worst.Role)
		if firstWarning {
			cxp.DemotionWarningAt = now
			cxp.DemotionWarningRole = worst.Role
			l.leaderboard[wallet] = stats
			l.logAdminAuditLocked("CAREER_DEMOTION_WARNING", wallet, fmt.Sprintf(
				"Role: %s, required: %d micro, sustained: %d micro, shortfall: %d micro",
				worst.Role, worst.MinSustainedMicro, avg, worstShortfall))
		}
		return []CareerStandingEvent{{
			Kind:           "warning",
			Wallet:         wallet,
			Role:           worst.Role,
			RequiredMicro:  worst.MinSustainedMicro,
			AvgMicro:       avg,
			ShortfallMicro: worstShortfall,
			GraceEndsAt:    cxp.DemotionWarningAt.Add(careerDemotionGrace()),
			Message: fmt.Sprintf("%s needs %s $VBV sustained (you hold %s — %s short). Restore it within %d days or the career is demoted.",
				worst.Role, formatMicroShort(worst.MinSustainedMicro), formatMicroShort(avg),
				formatMicroShort(worstShortfall), DemotionGracePeriodDays),
			At: now,
		}}
	}
	return l.applyCareerDemotionLocked(wallet, stats, worst, avg, worstShortfall, now)
}

// applyCareerDemotionLocked removes a career and records why. The CALLER HOLDS the lobby WRITE lock.
// It is separated from the evaluation so the decision (warn vs demote) and the consequence (remove
// the grant) are two readable steps rather than one long branch.
func (l *Lobby) applyCareerDemotionLocked(wallet string, stats PlayerStats, gate *CareerRoleGate, avg, shortfall uint64, now time.Time) []CareerStandingEvent {
	cxp := stats.CareerXP
	demoted := gate.Role
	remaining := make([]string, 0, len(cxp.PromotedRoles))
	for _, role := range cxp.PromotedRoles {
		if RoleKey(role) != RoleKey(demoted) {
			remaining = append(remaining, role)
		}
	}
	cxp.PromotedRoles = remaining
	cxp.Demotions = append(cxp.Demotions, CareerDemotion{
		Role:           demoted,
		RequiredMicro:  gate.MinSustainedMicro,
		AvgMicro:       avg,
		ShortfallMicro: shortfall,
		Reason:         CareerDemotionReasonDrain,
		DemotedAt:      now,
	})
	cxp.DemotionWarningAt = time.Time{}
	cxp.DemotionWarningRole = ""
	// The stored role must stop naming a career the player no longer holds.
	if RoleKey(stats.JobRole) == RoleKey(demoted) {
		if len(remaining) > 0 {
			stats.JobRole = remaining[0]
		} else {
			stats.JobRole = "Freelancer"
		}
	}
	l.leaderboard[wallet] = stats
	l.logAdminAuditLocked("CAREER_DEMOTED", wallet, fmt.Sprintf(
		"Role: %s (required: %d micro, sustained: %d micro, shortfall: %d micro) — %s",
		demoted, gate.MinSustainedMicro, avg, shortfall, CareerDemotionReasonDrain))
	return []CareerStandingEvent{{
		Kind:           "demoted",
		Wallet:         wallet,
		Role:           demoted,
		RequiredMicro:  gate.MinSustainedMicro,
		AvgMicro:       avg,
		ShortfallMicro: shortfall,
		Message: fmt.Sprintf("%s has been demoted: %s $VBV sustained is required (you held %s — %s short).",
			demoted, formatMicroShort(gate.MinSustainedMicro), formatMicroShort(avg), formatMicroShort(shortfall)),
		At: now,
	}}
}

// NotifyCareerStandingLocked delivers a standing event to the ONE wallet it concerns. The CALLER
// HOLDS the lobby lock.
//
// This replaces the broadcast in `server.go`, whose guard was `lwb == walletLower || cid != ""` —
// always true, so every connected client received every other player's demotion (a wallet, a role
// and a balance shortfall are not public facts). Delivery is now addressed by
// `getClientIDFromWalletLocked`, and an offline player simply misses nothing: the WARNING is stored
// on the record and the panel shows it on the next read.
func (l *Lobby) NotifyCareerStandingLocked(ev CareerStandingEvent) {
	if l == nil {
		return
	}
	cid := l.getClientIDFromWalletLocked(ev.Wallet)
	if cid == "" {
		return
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		return
	}
	eventType := "career_tier_demoted"
	if ev.Kind == "warning" {
		eventType = "career_demotion_warning"
	} else if ev.Kind == "cleared" {
		eventType = "career_demotion_cleared"
	}
	l.sendToClientLocked(cid, Envelope{Type: eventType, Payload: json.RawMessage(payload)})
}

// ---------------------------------------------------------------------------
// 9. THE SERVED PROJECTION — the ONE place the client learns this taxonomy
// ---------------------------------------------------------------------------

// CareerPathView is the served shape of the whole career-path system. The client RENDERS this and
// re-declares none of it (the rule the bonded-branding and slide-theming surfaces already follow),
// which is what removes the invented `faction: JUSTICE|UNDERWORLD|HYBRID` taxonomy.
type CareerPathView struct {
	Wallet        string                        `json:"wallet"`
	Path          string                        `json:"path"`
	PathChosenAt  time.Time                     `json:"path_chosen_at,omitempty"`
	Paths         []CareerPathOption            `json:"paths"`
	Civil         CivilStanding                 `json:"civil"`
	CivilRanks    []CivilRankOption             `json:"civil_ranks"`
	Eligibility   CareerPathEligibility         `json:"eligibility"`
	PromotedRoles []string                      `json:"promoted_roles"`
	Demotions     []CareerDemotion              `json:"demotions"`
	Warning       *CareerStandingEvent          `json:"warning,omitempty"`
	Careers       []CareerPromotionRequirements `json:"careers"`
	Upgrades      CareerUpgradeView             `json:"upgrades"`
	Matrices      []RivalryMatrixRef            `json:"matrices"`
	CareerPairs   []CareerPairRef               `json:"career_pairs"`
	Rules         map[string]interface{}        `json:"rules"`
}

// CareerPathOption describes one selectable path and how it relates to the other two. The relation
// is SERVED per pair so the client never implements the matrix.
type CareerPathOption struct {
	ID      string            `json:"id"`
	Label   string            `json:"label"`
	Summary string            `json:"summary"`
	Rivals  []CareerPathRival `json:"rivals"`
}

// CareerPathRival is one cell of the 3x3 path matrix.
type CareerPathRival struct {
	Against  string `json:"against"`
	Relation string `json:"relation"`
	BonusBps int    `json:"bonus_bps"`
	Explain  string `json:"explain"`
}

// CivilRankOption is one step of the civil ladder and what unlocks it.
type CivilRankOption struct {
	Tier     string `json:"tier"`
	Requires string `json:"requires"`
}

// RivalryMatrixRef names one of the three matrices, its owner and what it scores.
type RivalryMatrixRef struct {
	Stage     string `json:"stage"`
	Owner     string `json:"owner"`
	Describes string `json:"describes"`
}

// CareerPairRef is one row of the career-pair matrix, with the delta the engine actually prices it
// at (read from `GetRivalXPDelta`, never restated).
type CareerPairRef struct {
	Pair  string `json:"pair"`
	A     string `json:"a"`
	B     string `json:"b"`
	Delta int    `json:"delta"`
}

// careerPathLabel is the display name of a path (the vocabulary is the ids; this is presentation).
func careerPathLabel(path string) string {
	switch PathKey(path) {
	case CareerPathJustice:
		return "Justice"
	case CareerPathCriminal:
		return "Criminal"
	case CareerPathNeutral:
		return "Neutral"
	default:
		return path
	}
}

// careerPathSummary explains a path in one line, so a choice is informed rather than a coin toss.
func careerPathSummary(path string) string {
	switch PathKey(path) {
	case CareerPathJustice:
		return "Holds the justice side: direct rivalry against the criminal path and its careers only."
	case CareerPathCriminal:
		return "Holds the criminal side: direct rivalry against the justice path and its careers only."
	case CareerPathNeutral:
		return "Holds no side: may promote into EITHER side's careers, and interprets rivalries from other neutrals and the matrices behind the other driving aspects. Pays for that breadth with no direct-rival leverage."
	default:
		return ""
	}
}

// pathRelationExplain states WHY a relation resolved the way it did, so a "shared"/"none" cell is
// explained instead of looking broken.
func pathRelationExplain(relation string) string {
	switch relation {
	case PathRivalryDirect:
		return "direct rivalry — opposed paths, earns the rival delta"
	case PathRivalryInterpreted:
		return "interpreted — no direct opponent; the player reads the matrices instead"
	case PathRivalryShared:
		return "shared path — an ally relation, not a rivalry"
	default:
		return "no path held"
	}
}

// CareerPathViewLocked builds the served projection. The CALLER HOLDS the lobby lock.
func (l *Lobby) CareerPathViewLocked(wallet string) CareerPathView {
	view := CareerPathView{
		Wallet:        wallet,
		Paths:         make([]CareerPathOption, 0, len(CareerPaths)),
		CivilRanks:    make([]CivilRankOption, 0, len(CivilRanks)),
		PromotedRoles: []string{},
		Demotions:     []CareerDemotion{},
		Careers:       make([]CareerPromotionRequirements, 0, len(careerRoleGates)),
		Matrices:      make([]RivalryMatrixRef, 0, len(RivalryMatrixStages)),
		CareerPairs:   make([]CareerPairRef, 0, len(rivalPairTable)),
	}
	view.Civil = l.CivilStandingForWalletLocked(wallet)
	view.Eligibility = l.CareerPathEligibilityForWalletLocked(wallet)

	var cxp *CareerXP
	if stats, ok := l.leaderboard[wallet]; ok {
		cxp = stats.CareerXP
	}
	if cxp != nil {
		view.Path = PathKey(cxp.Path)
		view.PathChosenAt = cxp.PathChosenAt
		if cxp.PromotedRoles != nil {
			view.PromotedRoles = append(view.PromotedRoles, cxp.PromotedRoles...)
		}
		if cxp.Demotions != nil {
			view.Demotions = append(view.Demotions, cxp.Demotions...)
		}
		if !cxp.DemotionWarningAt.IsZero() && cxp.DemotionWarningRole != "" {
			view.Warning = careerDemotionWarningLocked(wallet, cxp)
		}
	}

	// The 3x3 path matrix, computed from the resolver (never transcribed by hand).
	for _, p := range CareerPaths {
		opt := CareerPathOption{
			ID:      p,
			Label:   careerPathLabel(p),
			Summary: careerPathSummary(p),
			Rivals:  make([]CareerPathRival, 0, len(CareerPaths)),
		}
		for _, q := range CareerPaths {
			rel := CareerPathRivalry(p, q)
			opt.Rivals = append(opt.Rivals, CareerPathRival{
				Against:  q,
				Relation: rel,
				BonusBps: PathRivalryBonusBps(p, q),
				Explain:  pathRelationExplain(rel),
			})
		}
		view.Paths = append(view.Paths, opt)
	}

	for _, rank := range CivilRanks {
		view.CivilRanks = append(view.CivilRanks, CivilRankOption{Tier: rank, Requires: civilTierRequirementHint(rank)})
	}
	return l.careerPathViewTailLocked(wallet, view)
}

// careerPathViewTailLocked fills the career table, the three matrix references and the rules. Split
// from the head of the builder purely so each half stays readable. The CALLER HOLDS the lobby lock.
func (l *Lobby) careerPathViewTailLocked(wallet string, view CareerPathView) CareerPathView {
	// Every declared career with the wallet's own progress against it. The ORDER comes from
	// `orderedCareerGates` — the SAME ordering the unlock advisory reads — so the served table and
	// the unlocked list cannot disagree about which career comes first.
	for _, g := range orderedCareerGates() {
		view.Careers = append(view.Careers, l.CareerPromotionRequirementsForWalletLocked(wallet, g.Role))
	}

	// THE ADVISORY (§7.5). "Unlocked, never forced": what the caller may upgrade right now, and who
	// they may ask. Derived on every read; nothing in it is a write.
	view.Upgrades = l.CareerUpgradeViewLocked(wallet)

	view.Matrices = []RivalryMatrixRef{
		{Stage: MatrixStagePath, Owner: "career_path.go",
			Describes: "justice ↔ criminal is the one DIRECT rivalry; every neutral pairing is INTERPRETED (no direct opponent)."},
		{Stage: MatrixStageCareer, Owner: "rival_career_engine.go (rivalPairTable)",
			Describes: "role ↔ role XP deltas; every enemy pair crosses the two paths and every ally pair shares one."},
		{Stage: MatrixStageRegion, Owner: "rivalry_engine.go + theme_engine.go",
			Describes: "territory/region asset signatures and the 10 world-dynamics weights (RIVAL_COLLISION_THRESHOLD 40000)."},
	}

	// The career-pair matrix, with the delta the engine actually prices each pair at.
	for _, p := range rivalPairTable {
		view.CareerPairs = append(view.CareerPairs, CareerPairRef{
			Pair:  p.Name,
			A:     CareerPathOfRole(p.A),
			B:     CareerPathOfRole(p.B),
			Delta: GetRivalXPDelta(p.Name),
		})
	}

	view.Rules = map[string]interface{}{
		"choose_once":                "A career path is chosen ONCE and cannot be flipped on demand.",
		"choice_gate":                fmt.Sprintf("Opening a region requires %d territories; the choice unlocks with the region.", CareerPathTerritoryRequirement),
		"neutral_breadth":            "A neutral player may promote into either side's careers and earns no direct-rival bonus.",
		"direct_bonus_bps":           PathDirectRivalryBonusBps,
		"promotion_basis":            "level cap + role tier + sustained $VBV + civil rank + path",
		"demotion_basis":             "sustained $VBV below the career's requirement, warned first, demoted when the grace period expires",
		"demotion_grace_days":        DemotionGracePeriodDays,
		"civil_rank_is_engine_state": "a civil rank is derived from owned clubs/territories/region and can never be declared by a client",
		"upgrade_is_opt_in":          CareerUpgradeStatement,
		"unlock_basis":               CareerUnlockBasisStatement,
		"notice_only":                CareerNoticeRuleStatement,
		"staff_upgrades":             CareerStaffBasisStatement + " — an employer may SEE and REQUEST; the upgrade itself belongs to the staff member and only they can take it",
	}

	return view
}

// ---------------------------------------------------------------------------
// 10. THE DIALECT GUARD — fail at BOOT, not silently in play
// ---------------------------------------------------------------------------
//
// The same discipline the bundled-asset card rule and the note vocabulary already use: if a future
// change adds a path, a rank or a career that disagrees with the others, the process must not run
// quietly with the rule broken. An unknown path folds to itself, `IsCareerPath` returns false, and
// the whole feature would then answer "not eligible" for every player with NO error anywhere — the
// silent-skip class this repository has had to repair repeatedly. So it PANICS at init instead.
func assertCareerPathDialect() {
	// 1. The declared lists must be exactly the constants, and each must fold to itself (a constant
	//    that folded to another would make two paths the same path).
	if len(CareerPaths) != 3 {
		panic("career_path: CareerPaths must declare exactly three paths")
	}
	for _, p := range CareerPaths {
		if PathKey(p) != p {
			panic(fmt.Sprintf("career_path: path %q does not fold to itself — it would collide with another path", p))
		}
	}
	if len(CivilRanks) != 3 {
		panic("career_path: CivilRanks must declare exactly three ranks")
	}
	for _, r := range CivilRanks {
		if PathKey(r) != r {
			panic(fmt.Sprintf("career_path: civil rank %q does not fold to itself", r))
		}
	}

	// 2. Every alias must land on a real path (an alias to a typo is a dead end).
	for spelling, target := range careerPathAliases {
		if !IsCareerPath(target) {
			panic(fmt.Sprintf("career_path: alias %q targets %q, which is not a declared path", spelling, target))
		}
	}

	// 3. Every career must name a real path AND have a promotion gate that AGREES with it. A career
	//    in the path map with no gate at all is almost always an omission, so it is named at boot.
	for key, home := range careerPathByRole {
		if !IsCareerPath(home) {
			panic(fmt.Sprintf("career_path: career %q declares path %q, which is not a declared path", key, home))
		}
		gate, ok := careerRoleGateByKey[key]
		if !ok {
			panic(fmt.Sprintf("career_path: career %q has a declared path but no promotion gate — add it to careerRoleGates", key))
		}
		if gate.HomePath != home {
			panic(fmt.Sprintf("career_path: career %q is %s in careerPathByRole but %s in careerRoleGates",
				key, home, gate.HomePath))
		}
	}

	// 4. The gate table must be one row per role, with a real path, a real rank and a real ladder rung.
	seen := make(map[string]bool, len(careerRoleGates))
	for _, g := range careerRoleGates {
		key := RoleKey(g.Role)
		if seen[key] {
			panic(fmt.Sprintf("career_path: role %q is declared twice in careerRoleGates — one owner per gate", g.Role))
		}
		seen[key] = true
		if !IsCareerPath(g.HomePath) {
			panic(fmt.Sprintf("career_path: gate %q declares path %q, which is not a declared path", g.Role, g.HomePath))
		}
		switch PathKey(g.MinCivilTier) {
		case CivilTierUser, CivilTierManager, CivilTierGovernor:
		default:
			panic(fmt.Sprintf("career_path: gate %q declares civil rank %q, which is not a declared rank", g.Role, g.MinCivilTier))
		}
		switch g.MinSustainedMicro {
		case VBVTierPeonMicro, VBVTierApprentice, VBVTierJourneyman, VBVTierExpert, VBVTierMaster, VBVTierBoss:
		default:
			panic(fmt.Sprintf("career_path: gate %q declares %d micro, which is not a rung of the $VBV tier ladder",
				g.Role, g.MinSustainedMicro))
		}
		if g.MinLessonLevel <= 0 || g.MinLessonLevel > 100 {
			panic(fmt.Sprintf("career_path: gate %q declares level cap %d, outside the 1..100 career level", g.Role, g.MinLessonLevel))
		}
		// A tier above what `getTierFor` can RETURN would make the career permanently unearnable,
		// with no error anywhere — the silent-skip class. Fail the boot instead.
		if g.MinRoleTier < 1 || g.MinRoleTier > CareerRoleTierMax {
			panic(fmt.Sprintf("career_path: gate %q requires role tier %d, but getTierFor only returns 1..%d — "+
				"this career could never be promoted", g.Role, g.MinRoleTier, CareerRoleTierMax))
		}
	}

	// 5. The path matrix must be internally consistent: the two sided paths are the ONE direct
	//    rivalry, and everything else is not a direct rivalry.
	if CareerPathRivalry(CareerPathJustice, CareerPathCriminal) != PathRivalryDirect {
		panic("career_path: justice ↔ criminal must be the direct rivalry")
	}
	if CareerPathRivalry(CareerPathNeutral, CareerPathNeutral) != PathRivalryInterpreted {
		panic("career_path: two neutral players must INTERPRET a rivalry, never hold a direct one")
	}
	if CareerPathRivalry(CareerPathJustice, CareerPathJustice) == PathRivalryDirect {
		panic("career_path: two players on the same side cannot be direct rivals")
	}
}

func init() { assertCareerPathDialect() }
