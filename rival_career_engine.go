//go:build !js && !wasm

package main

import (
	"fmt"
	"strings"
	"time"
)

// ============================================================================
// PILLAR 13: CAREER PROGRESSION SYSTEM (Section 13)
// XP/Level gating for all 20 career roles across Underworld & Justice layers.
// ============================================================================

const (
	CareerTierPeon       = 0
	CareerTierApprentice = 5
	CareerTierJourneyman = 15
	CareerTierExpert     = 30
	CareerTierMaster     = 50
	CareerTierBoss       = 75
)

// PILLAR 13: $VBV-Sustained Tier Thresholds (micro-units)
const (
	VBVTierPeonMicro       = 0                  // Tier 0: no gate
	VBVTierApprentice      = 5_000_000_000      // 5K $VBV
	VBVTierJourneyman      = 25_000_000_000     // 25K $VBV
	VBVTierExpert          = 100_000_000_000    // 100K $VBV
	VBVTierMaster          = 500_000_000_000    // 500K $VBV
	VBVTierBoss            = 2_000_000_000_000  // 2M $VBV
	DemotionGracePeriodDays = 7
)

// ============================================================================
// THE CAREER ROLE VOCABULARY — ONE OWNER FOR ROLE NAMES
// ============================================================================
//
// WHY THIS EXISTS. Role names reach this engine as bare string literals at ~50 call sites in TWO
// competing spellings, and nothing reconciled them:
//
//	DISPLAY form (what the rival-pair table below declares):  "Forensic Analyst", "Intel-Agent",
//	          "Arc-Net Operative", "Tax Auditor", "Bounty Hunter", "Mutation Log Auditor"
//	IDENTIFIER form (what the combat hooks pass):             "ForensicAnalyst", "IntelAgent",
//	          "ArcNetOperative", "TaxAuditor", "BountyHunter", "MutationAuditor"
//
// `GetRivalPairName` compares with `==`, so a pair whose caller speaks the other spelling answers
// "" — which is not an error, not a log entry and not an XP award: the interaction is silently
// skipped, and the hook beside it looks like a balance bug rather than a string mismatch. `RoleXP`
// is keyed by whatever the WRITER passed, so XP tracked as "Int.Agent" is invisible to a reader
// asking for "IntelAgent", and the two award paths drift apart in tier as well as in name.
//
// RoleKey folds a spelling onto ONE comparison key: lower-cased, with `-`, `.`, `_` and whitespace
// treated as separators and removed, so "Arc-Net Operative"/"ArcNetOperative" and "Tax Auditor"/
// "TaxAuditor" are ONE role with no alias table at all. The names that are genuinely DIFFERENT
// WORDS are declared once in roleAliases, each with the file that proves it. Add a spelling HERE
// rather than teaching another call site how to spell a career.
// ============================================================================

// roleAliases maps a folded name onto the folded name the rest of the engine agrees on. Both sides
// are outputs of foldRole, and every entry names where its spelling comes from. A name whose fold
// is already the agreed one needs NO entry (separators and casing are handled by the fold itself).
var roleAliases = map[string]string{
	// handlers_criminality.go speaks "Int.Agent" exactly where battle_service.go speaks
	// "IntelAgent"; the code's own comments call the career "Intel-Agent". One career, one key.
	"intagent": "intelagent",
	// battle_service.go's P2-D8 ally pair passes "MutationAuditor" while the pair table declares
	// "Mutation Log Auditor" (the same career, both under the P2-D8 comment).
	"mutationauditor": "mutationlogauditor",
	// `ai_citizen_engine.go`'s aiPathwayByCareer and `item_shop_archetype.go` speak "AOS" for the
	// career `rivalPairTable` declares as "AOS Leader" (whose pair NAME is "AOS↔SectorPeacekeeper",
	// so even the pair table spells it both ways). One career, one key — the path map and the
	// promotion gate are both declared under "AOS Leader".
	"aos": "aosleader",
}

// foldRole lower-cases a role name and removes its separators. Internal to this file: the
// comparison key is an implementation detail of the fold, not a second spelling to pass around.
func foldRole(role string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(role)) {
		switch r {
		case '-', '.', '_', ' ', '\t', '\n', '\r':
			// separators carry no meaning between words of a role name
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// RoleKey returns the comparison key for a career role name. Two spellings of one career share a
// RoleKey; two different careers never do. Use it for EVERY role comparison instead of `==`.
func RoleKey(role string) string {
	folded := foldRole(role)
	if canonical, ok := roleAliases[folded]; ok {
		return canonical
	}
	return folded
}

// roleXP returns the XP this player has for a role, resolving the RoleXP key by ROLE rather than by
// exact spelling, so XP tracked as "Int.Agent" is read by a caller asking for "IntelAgent".
//
// When several keys name the same role (the drift this vocabulary exists to end) the HIGHEST is
// returned — never a sum, because summing would invent XP that was never awarded.
func roleXP(cxp *CareerXP, role string) uint64 {
	if cxp == nil || len(cxp.RoleXP) == 0 {
		return 0
	}
	want := RoleKey(role)
	var best uint64
	for rName, xp := range cxp.RoleXP {
		if RoleKey(rName) == want && xp > best {
			best = xp
		}
	}
	return best
}

// CareerTier captures per-role tier progression (reconciled field used by rumor/loyalty logic).
type CareerTier struct {
	Role              string `json:"role,omitempty"`
	LessonsCompleted  int    `json:"lessons_completed,omitempty"`
	StandingTier      int    `json:"standing_tier,omitempty"`
}

// CareerXP records a player's progression through career tiers.
// Pillar 13 ($VBV-Sustained Progression): LiquiditySamples + AvgSustainedMicro gate all tier advancement.
type CareerXP struct {
	RoleXP          map[string]uint64 `json:"role_xp"`          // Role name -> XP earned
	LessonLevel     int               `json:"level"`            // Overall career level (0-100)
	PromotedRoles   []string          `json:"promoted_roles"`   // Roles this player qualifies for
	CurrentPrompts  []string          `json:"current_prompts"`  // Active promotion offers (UI)
	RoleName        string            `json:"role_name,omitempty"` // Active/primary role label (reconciled)
	Tiers           []CareerTier      `json:"tiers,omitempty"`   // Per-role tier progression (LessonsCompleted/StandingTier)

	// Pillar 13: $VBV-sustained balance tracking
	LiquiditySamples  []uint64  `json:"-"`                // Recent player balance snapshots (micro-$VBV), last 14
	AvgSustainedMicro uint64    `json:"avg_sustained_micro"` // Computed average from samples (micro-$VBV)
	DemotionWarningAt time.Time `json:"demotion_warning_at"` // When demotion warning was issued (0 = none)

	// THE CAREER PATH + LIFECYCLE. `career_path.go` owns the RULES for these; the fields live here
	// because this struct is the record that persists them. `PromotedRoles` above is the GRANT — a
	// demotion removes an entry from it — and it is written only by `Lobby.PromoteCareerLocked`.
	Path                string           `json:"path,omitempty"`                   // justice | criminal | neutral, chosen ONCE
	PathChosenAt        time.Time        `json:"path_chosen_at,omitempty"`         // when the path was chosen
	DemotionWarningRole string           `json:"demotion_warning_role,omitempty"`  // the role the outstanding warning names
	LastPromotionAt     time.Time        `json:"last_promotion_at,omitempty"`      // when the last promotion landed
	Demotions           []CareerDemotion `json:"demotions,omitempty"`              // auditable demotion history

	// THE UNLOCK NOTICE. The operator's rule is that a career is UNLOCKED and never forced, so an
	// unlock is a NOTIFICATION and nothing more. This records the roles the player has ALREADY been
	// told about — one entry per role — so the notice fires once per unlock instead of on every
	// sample. The entry is CLEARED when the role is promoted (`PromoteCareerLocked`), so a career
	// lost to a demotion and unlocked again genuinely re-notifies. `career_path.go` owns the rule.
	UnlockNoticesSent map[string]time.Time `json:"unlock_notices_sent,omitempty"`
}

// GetCareerTier returns the player's career tier for a given role (reconciled helper).
func (cxp *CareerXP) GetCareerTier(role string) int {
	if cxp == nil {
		return 0
	}
	// roleXP resolves the RoleXP key BY ROLE, so XP tracked under another spelling of the same
	// career is not invisible to a reader asking in this one (see the role vocabulary above).
	xp := roleXP(cxp, role)
	if xp == 0 {
		return 0
	}
	return int(xp / 1500)
}

// GetRumorFeeDiscount returns the rumor-spreading fee discount from career standing (reconciled).
func (cxp *CareerXP) GetRumorFeeDiscount() float64 {
	if cxp == nil {
		return 0.0
	}
	return float64(cxp.LessonLevel) * 0.01
}

// GetJusticeMissionFeeDiscount returns the justice-mission fee discount from career standing (reconciled).
func (cxp *CareerXP) GetJusticeMissionFeeDiscount() float64 {
	if cxp == nil {
		return 0.0
	}
	return float64(cxp.LessonLevel) * 0.01
}

// CalculateLessonLevel returns the player's career level based on total XP.
func (cxp *CareerXP) CalculateLessonLevel() int {
	totalXP := uint64(0)
	for _, xp := range cxp.RoleXP {
		totalXP += xp
	}
	// Exponential curve: each level requires 1500 more XP than previous.
	level := 0
	for totalXP >= uint64(level*level*30+level*1500) && level < 100 {
		level++
	}
	cxp.LessonLevel = level
	return level
}

// UnlockAchievementForRole grants career-related achievements.
func (cxp *CareerXP) UnlockAchievementForRole(role string) bool {
	for _, r := range cxp.PromotedRoles {
		if r == role {
			return true
		}
	}
	return false
}

// TrackCareerXP adds XP to a specific role's progression.
func (cxp *CareerXP) TrackCareerXP(role string, xp uint64) {
	if cxp == nil {
		return
	}
	if cxp.RoleXP == nil {
		cxp.RoleXP = make(map[string]uint64)
	}
	cxp.RoleXP[role] += xp
}

// CareerHasRole checks if a player has promoted to a specific role.
func CareerHasRole(cxp *CareerXP, role string) bool {
	if cxp == nil {
		return false
	}
	for _, r := range cxp.PromotedRoles {
		if r == role {
			return true
		}
	}
	return false
}

// CheckCareerTierGate validates if player's sustained $VBV balance meets tier requirement.
// PILLAR 13: Returns (tierGatePass, currentTier, requiredMicro, isDemotionWarning).
func (cxp *CareerXP) CheckCareerTierGate(requiredTier int) (bool, int, uint64, bool) {
	if cxp == nil || len(cxp.LiquiditySamples) == 0 {
		return requiredTier == 0, 0, 0, false
	}

	// Compute average sustained balance
	sum := uint64(0)
	for _, sample := range cxp.LiquiditySamples {
		sum += sample
	}
	avg := sum / uint64(len(cxp.LiquiditySamples))
	cxp.AvgSustainedMicro = avg

	// Determine current tier from average balance
	currentTier := 0
	switch {
	case avg >= VBVTierBoss:
		currentTier = 5
	case avg >= VBVTierMaster:
		currentTier = 4
	case avg >= VBVTierExpert:
		currentTier = 3
	case avg >= VBVTierJourneyman:
		currentTier = 2
	case avg >= VBVTierApprentice:
		currentTier = 1
	default:
		currentTier = 0
	}

	// Determine required micro value for requested tier
	requiredMicro := uint64(0)
	switch requiredTier {
	case 5:
		requiredMicro = VBVTierBoss
	case 4:
		requiredMicro = VBVTierMaster
	case 3:
		requiredMicro = VBVTierExpert
	case 2:
		requiredMicro = VBVTierJourneyman
	case 1:
		requiredMicro = VBVTierApprentice
	default:
		requiredMicro = 0
	}

	// The demotion warning is an AGE signal: a warning was issued and its grace period has run out.
	// `career_path.go` withdraws the warning the moment the balance recovers, so this is true only
	// while a shortfall is genuinely outstanding.
	isDemotionWarning := !cxp.DemotionWarningAt.IsZero() &&
		time.Since(cxp.DemotionWarningAt) >= time.Duration(DemotionGracePeriodDays)*24*time.Hour

	// THE GATE IS THE BALANCE ALONE. This used to read `avg >= requiredMicro && !isDemotionWarning`,
	// which failed a player FOREVER once a warning aged past the grace period — even after they had
	// restored the balance — because the stale timestamp never cleared. A recovered player could not
	// pass their own gate. The warning is reported separately now, and `EvaluateCareerStandingLocked`
	// is what acts on it.
	gatePass := avg >= requiredMicro
	return gatePass, currentTier, requiredMicro, isDemotionWarning
}

// RivalryState tracks dynamic rivalry mechanics between players.
type RivalryState struct {
	SoloHunterScore      int                `json:"solo_hunter_score"`        // Bounty hunter solo capture score
	AOSRivalryActive     bool               `json:"aos_rivalry_active"`       // Whether AOS rival is active
	AOSTeamID            string             `json:"aos_team_id"`              // Associated AOS team
	SilkRoadHoardCount   int                `json:"silk_road_hoard_count"`    // Cards hoarded by Hostage Hosts
	HoardPressure        int                `json:"hoard_pressure"`           // Ransom pressure multiplier
	InfoBrokerDeals      []InfoBrokerDeal   `json:"info_broker_deals"`        // Active info broker transactions
	ActiveRivals         []string           `json:"active_rivals,omitempty"`  // Wallet addresses of active rivals
	PendingInvitations   []PendingRivalInvite `json:"pending_invitations,omitempty"` // Pending rival invitations
	BountyLicenseActive  bool               `json:"bounty_license_active"`    // Active bounty hunter license status
	ArcNetActive         bool               `json:"arc_net_active,omitempty"` // Arc-Net spy vision active
}

// ============================================================================
// ADDITIONAL STRUCTS (Rivalry & Info Broker)
// ============================================================================

// InfoBrokerDeal represents an active info trade between two players.
type InfoBrokerDeal struct {
	ID          string    `json:"id"`
	SellerWallet string    `json:"seller_wallet"`
	BuyerWallet string    `json:"buyer_wallet"`
	InfoType    string    `json:"info_type"`    // e.g., "player_location", "card_pool"
	PriceVBV    uint64    `json:"price_vbv"`
	ExpiresAt   time.Time `json:"expires_at"`
	Completed   bool      `json:"completed"`
}

// PendingRivalInvite represents a pending rival request between players.
type PendingRivalInvite struct {
	ID               string    `json:"id"`
	InitiatorWallet  string    `json:"initiator_wallet"`
	TargetWallet     string    `json:"target_wallet"`
	InitiatorCareer  string    `json:"initiator_career"`
	TargetCareer     string    `json:"target_career"`
	CreatedAt        time.Time `json:"created_at"`
	ExpiresAt        time.Time `json:"expires_at"`
	// Reconciled alias fields (mechanical reconciliation)
	FromWallet string    `json:"from_wallet,omitempty"` // legacy alias for InitiatorWallet
	ToWallet   string    `json:"to_wallet,omitempty"`   // legacy alias for TargetWallet
	Level      int       `json:"level,omitempty"`        // career tier at invite time
	Timestamp  time.Time `json:"timestamp,omitempty"`   // legacy alias for CreatedAt
}

// ============================================================================
// THE RIVAL XP ARITHMETIC — INTEGER ONLY, ONE OWNER
// ============================================================================
//
// THE OWNER OF EVERY RIVAL AWARD, and of the modifier the tier ladder applies. Three defects lived
// in this path at once and hid each other:
//
//  1. THE RETURN VALUES WERE READ BY POSITION. `EvaluateCrossCareerXP` returns
//     `(attackerXP, defenderXP, pairName, isRival)`. SIX call sites read the SECOND value into a
//     local named `rivalXP` and compared it against the BASE (`rivalXP > 15`), but the second value
//     is the DEFENDER's monitoring share — `base × 0.30` — so the comparison was `4 > 15`: false,
//     for ever. FIVE of those six hooks had never fired once. The sixth fired and paid 30% of the
//     base where the ATTACKER's earned award was meant.
//  2. THE BONUS SUBTRACTION UNDERFLOWED. `uint64(rivalXP - 15)` with `rivalXP == 4` is not −11: it
//     is 18,446,744,073,709,551,605. The false guard in (1) was the only thing standing between
//     that expression and the ledger.
//  3. IT WAS FLOAT. `uint64(float64(baseXP) * scaling * modifier)` and `uint64(float64(base) * 0.30)`
//     are floats on career XP, which the Architecture Ledger prohibits outright.
//
// The fix is STRUCTURAL, not a corrected comparison: the result is a struct, so a wrong number
// cannot be selected; `Bonus` is computed HERE (never `a - b` at a call site) and is guarded; and
// every factor is an integer in parts-per-thousand.
const (
	// rivalXPPermille is 1.0 as an integer (parts-per-thousand).
	rivalXPPermille = 1000
	// rivalTierBonusPermille is the gain per career tier: 10% per tier, the same ladder the float
	// form applied as `0.1 * tier`.
	rivalTierBonusPermille = 100
	// RivalDefenderSharePermille is what the DEFENDER earns for being monitored/suppressed: 30% of
	// the base interaction. It was `uint64(float64(base) * 0.30)`.
	RivalDefenderSharePermille = 300
)

// GetRivalXPGainPermille returns a declared pair's gain in parts-per-thousand.
//
// `GetRivalXPDelta` is SIGNED — negative for an antagonistic pair, positive for a synergistic one —
// and BOTH branches of the float version added the magnitude (`1.0 + -delta/100` and
// `1.0 + delta/100` are the same number). So the gain is `1000 + |delta| * 10`, which also means an
// antagonistic pair is worth MORE than a synergistic one — the point of declaring it antagonistic.
func GetRivalXPGainPermille(pairName string) int {
	d := GetRivalXPDelta(pairName)
	if d < 0 {
		d = -d
	}
	return rivalXPPermille + d*10
}

// GetRivalPairModifierPermille returns the tier-scaled rival modifier in parts-per-thousand.
//
// This is the OWNER of the modifier; `GetRivalPairModifier` below is a display wrapper over it,
// exactly as `GetVBVGatingMultiplier` wraps `GetVBVGatingPermille`. The tier ladder is exact
// integers (10% per tier), so the integer form is not an approximation of the float one — it is the
// same number before the division.
func GetRivalPairModifierPermille(tier int) int {
	if tier < 0 {
		tier = 0
	}
	return rivalXPPermille + tier*rivalTierBonusPermille
}

// GetRivalPairModifier returns the rival XP modifier as a multiplier.
// DISPLAY FORM of the modifier — the arithmetic lives in GetRivalPairModifierPermille.
// `rivalCareer` is accepted for call-site compatibility: the modifier is per TIER, not per role
// (the float version read it and never used it either).
func (cxp *CareerXP) GetRivalPairModifier(rivalCareer string, tier int) float64 {
	return float64(GetRivalPairModifierPermille(tier)) / 1000.0
}

// ============================================================================
// CAREER XP & TIER GATE HELPERS
// ============================================================================

// GetRivalXPDelta returns the XP modifier delta for a rival pair.
// Negative = antagonistic (attacker gains bonus), Positive = synergistic (attacker gains less).
func GetRivalXPDelta(pairName string) int {
	// Enemy pairs (antagonistic - attacker gains bonus)
	switch pairName {
	case "BountyHunter↔Kidnapper", "BountyHunter_Kidnapper":
		return -15
	case "ForensicAnalyst↔Gossip", "ForensicAnalyst_Gossip":
		return -10
	case "TaxAuditor↔Launderer", "TaxAuditor_Launderer":
		return -10
	case "Warden↔HeistPlanner", "Warden_HeistPlanner":
		return -10
	case "SectorPeacekeeper↔Smuggler", "SectorPeacekeeper_Smuggler":
		return -10
	case "IntelAgent↔ArcNetOperative", "IntelAgent_ArcNetOperative":
		return -10
	// Synergistic pairs (existing)
	case "JusticeRecruiter↔BountyHunter", "JusticeRecruiter_BountyHunter":
		return +8
	case "Launderer↔Fence", "Launderer_Fence":
		return +5
	case "HeistPlanner↔Kidnapper", "HeistPlanner_Kidnapper":
		return +12
	case "AOS↔SectorPeacekeeper", "AOS_SectorPeacekeeper":
		return +6
	case "TaxAuditor↔JusticeCommissioner", "TaxAuditor_JusticeCommissioner":
		return +7
	case "Gossip↔ForensicAnalyst", "Gossip_ForensicAnalyst":
		return +5
	default:
		return 0
	}
}

// rivalPairTable is THE declaration of which careers interact, and what that interaction is called.
//
// It is package-level (not a local literal) so its invariants can be PINNED BY A TEST:
//   - ONE row per UNORDERED pair. The scan below returns on the FIRST match and the match is
//     order-independent, so a second row naming the same pair could never be reached. Two such rows
//     used to sit in here, each naming the pair differently — which is how one pair came to be
//     declared BOTH antagonistic (−10) and synergistic (+5) in GetRivalXPDelta while only one of
//     those names was ever produced.
//   - every Name is one GetRivalXPDelta prices (or is a declared zero, see the table there), so a
//     declared pair is never silently worth nothing.
var rivalPairTable = []struct{ A, B, Name string }{
	{"Bounty Hunter", "Kidnapper", "BountyHunter↔Kidnapper"},
	{"Forensic Analyst", "Gossip", "ForensicAnalyst↔Gossip"},
	{"Tax Auditor", "Launderer", "TaxAuditor↔Launderer"},
	{"Warden", "Heist Planner", "Warden↔HeistPlanner"},
	{"Sector Peacekeeper", "Smuggler", "SectorPeacekeeper↔Smuggler"},
	{"Intel-Agent", "Arc-Net Operative", "IntelAgent↔ArcNetOperative"},
	{"Justice Recruiter", "Bounty Hunter", "JusticeRecruiter↔BountyHunter"},
	{"Launderer", "Fence", "Launderer↔Fence"},
	{"Heist Planner", "Kidnapper", "HeistPlanner↔Kidnapper"},
	{"AOS Leader", "Sector Peacekeeper", "AOS↔SectorPeacekeeper"},
	{"Tax Auditor", "Justice Commissioner", "TaxAuditor↔JusticeCommissioner"},
	// {Gossip, Forensic Analyst} stood here a SECOND time, named "Gossip↔ForensicAnalyst". The
	// pair is declared above as "ForensicAnalyst↔Gossip" and the scan is order-independent, so
	// this row could never be reached — and that unreachable name is the one GetRivalXPDelta
	// prices SYNERGISTICALLY (+5), while the reachable one is antagonistic (−10). One unordered
	// pair is one interaction, and the reachable declaration is the one that holds. Recorded in
	// Problems.md §24.
	// The three rows that stood here were EXACT repeats of the three above them; the loop
	// returns on first match, so the copies could never fire. One owner per pair.
	// P2-D8: Justice Recruiter ↔ Mutation Log Auditor (ally pair)
	{"Justice Recruiter", "Mutation Log Auditor", "JusticeRecruiter↔MutationLogAuditor"},
	// P2-D9 repeated {Tax Auditor, Justice Commissioner} under the name
	// "JusticeCommissioner↔TaxAuditor". The row above already declares that pair, as
	// "TaxAuditor↔JusticeCommissioner" — the name GetRivalXPDelta prices at +7 — so this row was
	// unreachable and its name was unknown to the XP table. One row per pair.
	// P2-D10: Mutation Log Auditor ↔ Kidnapper (antagonistic - tracks their genetic tampering)
	{"Mutation Log Auditor", "Kidnapper", "MutationLogAuditor↔Kidnapper"},
}

// GetRivalPairName returns the canonical rival pair name for a given attacker-defender combination.
func GetRivalPairName(attackerCareer, defenderCareer string) string {
	for _, p := range rivalPairTable {
		// Compared BY ROLE, not by spelling: a caller passing "IntelAgent" must find the pair the
		// table declares as "Intel-Agent" (see the role vocabulary at the head of this file). The
		// `==` this replaces answered "" for every pair whose caller spoke the other spelling — a
		// silent skip with no error, no log and no XP.
		if (RoleKey(attackerCareer) == RoleKey(p.A) && RoleKey(defenderCareer) == RoleKey(p.B)) ||
			(RoleKey(attackerCareer) == RoleKey(p.B) && RoleKey(defenderCareer) == RoleKey(p.A)) {
			return p.Name
		}
	}
	return ""
}

// RivalXPAward is the COMPLETE result of resolving one cross-career interaction across BOTH
// matrices. Call sites must not do arithmetic on these fields: `Bonus` is already the difference
// and it is guarded, so it can never wrap to 18 quintillion the way `uint64(rivalXP - base)` did.
type RivalXPAward struct {
	Pair         string   `json:"pair"`          // the declared ROLE pair, or "" if none
	IsRival      bool     `json:"is_rival"`      // a career-matrix pair is declared
	IsDirect     bool     `json:"is_direct"`     // the two held PATHS are opposed (path matrix)
	BaseXP       uint64   `json:"base_xp"`       // the interaction's base award
	AttackerXP   uint64   `json:"attacker_xp"`   // what the ATTACKER earns (the first return value)
	DefenderXP   uint64   `json:"defender_xp"`   // the defender's monitoring share (the second)
	Bonus        uint64   `json:"bonus"`         // AttackerXP − BaseXP, guarded; NEVER underflows
	GainPermille int      `json:"gain_permille"` // the composed integer gain (1000 = unchanged)
	PathRelation string   `json:"path_relation"` // direct | interpreted | shared | none
	PathBonusBps int      `json:"path_bonus_bps"`
	Layers       []string `json:"layers"` // which matrices paid: career, path
	Explain      string   `json:"explain"`
}

// Awarded reports whether a matrix priced this interaction. Call sites use this (or `Bonus > 0`)
// instead of comparing a return value against the base.
func (a RivalXPAward) Awarded() bool { return a.IsRival || a.IsDirect }

// ResolveRivalXPAward resolves one cross-career interaction across the TWO player-facing matrices,
// in integers:
//
//	CAREER matrix — `rivalPairTable` → `GetRivalXPDelta`, scaled by the attacker's tier in that role.
//	PATH   matrix — `career_path.go` → `PathRivalryBonusBps`, where justice ↔ criminal is DIRECT.
//
// It is the ONE place an award is composed, so `TrackRivalInteraction` and
// `EvaluateCrossCareerXP` cannot drift apart and no call site can pick the wrong number.
//
// The TIER modifier applies ONLY when a role pair is declared — otherwise the tier ladder alone
// would pay every interaction with a stranger. A NEUTRAL path never earns the direct bonus
// (`CareerPathRivalry` answers `interpreted` for it), which is precisely what neutrality costs.
func ResolveRivalXPAward(
	attackerCareer, defenderCareer string,
	baseXP uint64,
	myStats *PlayerStats,
	targetStats *PlayerStats,
) RivalXPAward {
	award := RivalXPAward{BaseXP: baseXP, GainPermille: rivalXPPermille, PathRelation: PathRivalryNone}
	if baseXP == 0 || attackerCareer == "" || defenderCareer == "" || myStats == nil || targetStats == nil {
		award.Explain = "no context: a base, both roles and both players are required"
		return award
	}

	award.Pair = GetRivalPairName(attackerCareer, defenderCareer)
	award.IsRival = award.Pair != ""
	if award.IsRival {
		award.GainPermille = award.GainPermille * GetRivalXPGainPermille(award.Pair) / rivalXPPermille
		award.GainPermille = award.GainPermille *
			GetRivalPairModifierPermille(getTierFor(myStats.CareerXP, attackerCareer)) / rivalXPPermille
		award.Layers = append(award.Layers, MatrixStageCareer)
	}

	// The PATH matrix is read from the HELD path (`CareerXP.Path`, written by the one-time choice) —
	// the same field the served path matrix is computed from, so the number a player is SHOWN and
	// the number they are PAID come from one resolver.
	attackerPath, defenderPath := PlayerCareerPathOfStats(myStats), PlayerCareerPathOfStats(targetStats)
	award.PathRelation = CareerPathRivalry(attackerPath, defenderPath)
	award.PathBonusBps = PathRivalryBonusBps(attackerPath, defenderPath)
	award.IsDirect = award.PathBonusBps > 0
	if award.IsDirect {
		award.GainPermille = award.GainPermille * (10_000 + award.PathBonusBps) / 10_000
		award.Layers = append(award.Layers, MatrixStagePath)
	}

	award.AttackerXP = baseXP * uint64(award.GainPermille) / rivalXPPermille
	// The defender's monitoring share is 30% of the BASE; it never counted the attacker's gain.
	award.DefenderXP = baseXP * RivalDefenderSharePermille / rivalXPPermille
	// The guard the old call sites were missing. `AttackerXP` can never be below `BaseXP` (the gain
	// is >= 1000), but the subtraction is written so that even a future gain below 1000 cannot wrap.
	if award.AttackerXP > baseXP {
		award.Bonus = award.AttackerXP - baseXP
	}

	switch {
	case award.IsRival && award.IsDirect:
		award.Explain = fmt.Sprintf("career pair %s and the %s path, gain %d permille",
			award.Pair, award.PathRelation, award.GainPermille)
	case award.IsRival:
		award.Explain = fmt.Sprintf("career pair %s only (paths are %s), gain %d permille",
			award.Pair, award.PathRelation, award.GainPermille)
	case award.IsDirect:
		award.Explain = fmt.Sprintf("the %s path only (no declared role pair), gain %d permille",
			award.PathRelation, award.GainPermille)
	default:
		award.Explain = fmt.Sprintf("no matrix priced this interaction (paths are %s) — the base stands",
			award.PathRelation)
	}
	return award
}

// TrackRivalInteraction evaluates a cross-career interaction between attacker and defender,
// returns (xpAwarded, rivalPairName, isRivalPair).
// P2-A: Core wiring function — called at criminality handlers.
//
// It delegates to `ResolveRivalXPAward` so there is ONE composition, and it reports the AWARDED
// (attacker) figure — never the defender's share.
func TrackRivalInteraction(
	attackerCareer, defenderCareer string,
	baseXP uint64,
	myStats *PlayerStats,
	targetStats *PlayerStats,
) (uint64, string, bool) {
	award := ResolveRivalXPAward(attackerCareer, defenderCareer, baseXP, myStats, targetStats)
	if !award.Awarded() {
		return baseXP, "", false
	}
	return award.AttackerXP, award.Pair, award.IsRival
}

// getTierFor returns the career tier for a role from the player's XP map.
func getTierFor(cxp *CareerXP, role string) int {
	if cxp == nil || cxp.RoleXP == nil {
		return 1
	}
	xp := roleXP(cxp, role)
	switch {
	case xp >= uint64(CareerTierMaster*100): // Tier 4: 50 * 100 = 5000 XP
		return 4
	case xp >= uint64(CareerTierExpert*100): // Tier 3: 30 * 100 = 3000 XP
		return 3
	case xp >= uint64(CareerTierApprentice*100): // Tier 2: 5 * 100 = 500 XP
		return 2
	default:
		return 1
	}
}

// EvaluateCrossCareerXP applies rival modifiers for both attacker and defender.
// Returns (attackerXP, defenderXP, pairName, isRivalPair).
// P2-A: Called at interaction resolution to distribute XP fairly between both parties.
//
// THE TWO RETURN VALUES ARE NOT INTERCHANGEABLE, and six call sites treated them as though they
// were: the FIRST is what the attacker EARNS, the SECOND is the defender's 30% monitoring share.
// Use `ResolveRivalXPAward` (or the `Bonus` it computes) rather than subtracting from either — a
// bare `uint64(rivalXP - base)` underflows to 18,446,744,073,709,551,605 when the figure read is
// the second one.
func EvaluateCrossCareerXP(
	attackerCareer, defenderCareer string,
	baseAttackerXP uint64,
	myStats *PlayerStats,
	targetStats *PlayerStats,
) (uint64, uint64, string, bool) {
	award := ResolveRivalXPAward(attackerCareer, defenderCareer, baseAttackerXP, myStats, targetStats)
	if !award.Awarded() {
		return baseAttackerXP, 0, "", false
	}
	return award.AttackerXP, award.DefenderXP, award.Pair, award.IsRival
}

// ============================================================================
// HELPER: Map wallet to match player ID for buff application.
// ============================================================================

func playerStatsToID(stats PlayerStats) string {
	// This is a placeholder - in production this would resolve via the lobby's wallet-to-ID map
	return stats.ID
}

// computeScaledXPPermille scales raw career XP by the loyalty and fame bonuses, in
// PARTS-PER-THOUSAND.
//
// INTEGER-ONLY. The float form was `uint64(float64(baseXP) * scaler)` with `scaler` built from two
// float bonuses, which the Architecture Ledger prohibits for career XP. Loyalty is social influence
// (rumor discounts, trust); fame is public visibility / recognition (bounty multipliers, reputation
// buffs). Both compose here as integers, so the result is exact and identical on every platform.
//
// PRECISION: composing two permille factors truncates at each step, so a result can differ from the
// old float form by at most 1 XP. That is the intended trade — deterministic and float-free,
// rather than approximated in binary.
func (cxp *CareerXP) computeScaledXPPermille(baseXP uint64, loyaltyPermille, famePermille int) uint64 {
	if cxp == nil || baseXP == 0 {
		return baseXP
	}
	xp := baseXP
	if loyaltyPermille > 0 {
		xp = xp * uint64(rivalXPPermille+loyaltyPermille) / rivalXPPermille
	}
	if famePermille > 0 {
		xp = xp * uint64(rivalXPPermille+famePermille) / rivalXPPermille
	}
	return xp
}

// ComputeXPWithBonusesPermille applies loyalty and fame bonuses to a base award, in integers.
//
// It is the PUBLIC entry point for callers that derive their own bonuses from a career record — the
// rumor and fence handlers read `LessonsCompleted` / `LessonLevel` rather than the RoleXP totals
// `ComputeScaledXP` uses. Taking PERMILLE rather than a float is what keeps those two paths off the
// float arithmetic the Architecture Ledger prohibits.
func (cxp *CareerXP) ComputeXPWithBonusesPermille(baseXP uint64, loyaltyPermille, famePermille int) uint64 {
	return cxp.computeScaledXPPermille(baseXP, loyaltyPermille, famePermille)
}

// ComputeScaledXP is the public unified XP scaling helper for Task 4301 (PILLAR 13 career wiring).
// It computes loyalty and fame bonuses internally from CareerXP state, then applies them to baseXP.
// Usage: scaledXP := stats.CareerXP.ComputeScaledXP(baseXP, role)
func (cxp *CareerXP) ComputeScaledXP(baseXP uint64, role string) uint64 {
	if cxp == nil || baseXP == 0 {
		return baseXP
	}

	var loyaltyPermille, famePermille int

	// Only compute bonuses if the player has active career XP for this role.
	hasCareer := false
	for rName := range cxp.RoleXP {
		if strings.EqualFold(rName, role) || CareerHasRole(cxp, rName) && HasCareer(cxp, role) {
			hasCareer = true
			break
		}
	}

	// Also check if the player has any career XP at all (indicates active progression).
	if !hasCareer {
		for _, xpVal := range cxp.RoleXP {
			if xpVal > 0 {
				hasCareer = true
				break
			}
		}
	}

	if !hasCareer {
		return baseXP // No career match — return raw XP
	}

	// Loyalty: +1 permille per total RoleXP point, capped at +500 permille (+50%).
	// The old expression was `math.Min(0.50, float64(totalRoleXP)/1000.0)`: a FRACTION of the base,
	// `total/1000`, which is exactly `total` parts-per-thousand — so the integer form is not an
	// approximation of the float one, it is the same number before the division.
	var totalRoleXP uint64
	for _, xpVal := range cxp.RoleXP {
		totalRoleXP += xpVal
	}
	switch {
	case totalRoleXP > 500:
		loyaltyPermille = 500
	case totalRoleXP > 0:
		loyaltyPermille = int(totalRoleXP)
	}

	// Fame: +100 permille per career tier (10% per tier), capped at +600 permille (+60%).
	// The old expression was `math.Min(0.60, float64(tierLevel)*0.10)`.
	var maxRoleXP uint64
	for _, xpVal := range cxp.RoleXP {
		if xpVal > maxRoleXP {
			maxRoleXP = xpVal
		}
	}
	if maxRoleXP >= 500 { // Apprentice tier threshold (5 * 100)
		famePermille = getTierFor(cxp, role) * rivalTierBonusPermille
		if famePermille > 600 {
			famePermille = 600
		}
	}

	return cxp.computeScaledXPPermille(baseXP, loyaltyPermille, famePermille)
}

// HasCareer checks if a player has any career XP for the given role name (case-insensitive).
func HasCareer(cxp *CareerXP, role string) bool {
	if cxp == nil || cxp.RoleXP == nil {
		return false
	}
	// Compared BY ROLE (RoleKey): "Int.Agent" and "IntelAgent" name one career, and a caller asking
	// about it must not be told the player has no such career because the WRITER spelled it
	// differently (see the role vocabulary at the head of this file).
	want := RoleKey(role)
	for rName := range cxp.RoleXP {
		if RoleKey(rName) == want {
			return true
		}
	}
	return CareerHasRole(cxp, role)
}

// ============================================================================
// PILLAR 8: FENCE CAREER METHODS
// Fence fee discount and tier gating for black market operations.
// ============================================================================

// GetFenceFeeDiscount returns the fee discount multiplier for a Fence career role.
// Tier scaling (based on $VBV-sustained thresholds):
//   - Tier < 3 (below Journeyman): returns 1.0 (no discount)
//   - Tier >= 3 (Journeyman+):     returns 0.50 (50% fee reduction per PILLAR 8 spec)
func (cxp *CareerXP) GetFenceFeeDiscount() float64 {
	if cxp == nil || cxp.RoleXP == nil {
		return 1.0 // Default: no discount if no career data
	}

	fenceXP := cxp.RoleXP["Fence"]
	_ = fenceXP
	tier := getTierFor(cxp, "Fence")

	switch {
	case tier >= 3: // Journeyman+ (25K $VBV sustained) — full PILLAR 8 discount
		return 0.50
	case tier >= 2: // Apprentice+ (5K $VBV sustained) — partial discount
		return 0.75
	default: // Peon (< 5K $VBV) — no career discount
		return 1.0
	}
}

// GetFenceTier returns the integer tier level for the Fence career role.
// Tier mapping (consistent with CareerXP progression):
//   - Tier 1 (Peon):        < 5K $VBV sustained
//   - Tier 2 (Apprentice):  >= 5K $VBV sustained
//   - Tier 3 (Journeyman):  >= 25K $VBV sustained
//   - Tier 4 (Expert):      >= 100K $VBV sustained
//   - Tier 5 (Master):      >= 500K $VBV sustained
//   - Tier 6 (Boss):        >= 2M $VBV sustained
func (cxp *CareerXP) GetFenceTier() int {
	if cxp == nil || cxp.RoleXP == nil {
		return 1 // Default: Peon tier if no career data
	}

	tier := getTierFor(cxp, "Fence")
	if tier < 1 {
		return 1
	}
	return tier
}

// ============================================================================
// PILLAR 13: $VBV-SUSTAINED GATE MULTIPLIER (Section 15)
// Returns the XP multiplier based on sustained liquidity tier.
// Multiplier tiers: Peon(0)=×1, Apprentice=×2, Journeyman=×4, Expert=×8, Master=×16, Boss=×32
// ============================================================================

// GetVBVGatingPermille returns the $VBV-sustained XP gate in PARTS-PER-THOUSAND (1000 = ungated).
//
// This is the OWNER of the gate: the float getter below is a display wrapper over it, because the
// Architecture Ledger prohibits a float anywhere in the career-XP path ("Floating point prohibited
// for: ... Career / combat XP") while permitting one for presentation. The gate's values are exact
// integers (1×/2×/4×/8×/16×/32×), so the integer form is not an approximation of the float one —
// it is the same number before the division.
func (cxp *CareerXP) GetVBVGatingPermille() uint64 {
	if cxp == nil || cxp.AvgSustainedMicro < VBVTierApprentice {
		return 1000 // Tier 0: no gate — baseline multiplier
	}

	switch {
	case cxp.AvgSustainedMicro >= VBVTierBoss:
		return 32_000
	case cxp.AvgSustainedMicro >= VBVTierMaster:
		return 16_000
	case cxp.AvgSustainedMicro >= VBVTierExpert:
		return 8_000
	case cxp.AvgSustainedMicro >= VBVTierJourneyman:
		return 4_000
	default: // Apprentice tier (5K–25K $VBV sustained)
		return 2_000
	}
}

// GetVBVGatingMultiplier returns the $VBV-gated XP multiplier based on AvgSustainedMicro.
// DISPLAY FORM of the gate — the arithmetic lives in GetVBVGatingPermille.
func (cxp *CareerXP) GetVBVGatingMultiplier() float64 {
	return float64(cxp.GetVBVGatingPermille()) / 1000.0
}

// ============================================================================
// PILLAR 13: TAX AUDITOR CAREER — GetAuditPrecisionBonus
// Returns the audit accuracy multiplier for TaxAuditor career tier.
// Tier 1 (Junior): ×1.0 baseline precision
// Tier 2 (Auditor): ×1.15 +15% detection rate on hidden treasury operations
// Tier 3 (Senior Auditor): ×1.35 flags cross-club fund routing
// Tier 4 (Chief Auditor): ×1.60 uncovers offshore shell entities
// Tier 5 (Commissioner): ×2.00 full civilization-wide audit capability
// ============================================================================

// GetAuditPrecisionBonus returns the audit accuracy multiplier for TaxAuditor career tier.
func (cxp *CareerXP) GetAuditPrecisionBonus() float64 {
	if cxp == nil || cxp.RoleXP == nil {
		return 1.0 // no bonus if invalid input or no role data
	}

	hasTaxAuditor := false
	for rName := range cxp.RoleXP {
		if strings.EqualFold(rName, "TaxAuditor") {
			hasTaxAuditor = true
			break
		}
	}
	if !hasTaxAuditor && !CareerHasRole(cxp, "TaxAuditor") {
		return 1.0 // no bonus for non-TaxAuditor players
	}

	tier := getTierFor(cxp, "TaxAuditor")
	switch {
	case tier >= 5: // Commissioner+
		return 2.00
	case tier >= 4: // Chief Auditor+
		return 1.60
	case tier >= 3: // Senior Auditor+
		return 1.35
	case tier >= 2: // Auditor+
		return 1.15
	default: // Junior (Tier 1) — baseline precision
		return 1.0
	}
}

// IsJusticeAligned returns true if the given career choice string corresponds to a Justice-aligned career path.
func IsJusticeAligned(careerChoice string) bool {
	// Compared BY ROLE (RoleKey), not by spelling. Callers pass the identifier form ("IntelAgent",
	// "TaxAuditor") and the display form ("Intel-Agent", "Tax Auditor") for the same careers, and
	// the old `switch` on exact literals answered false for the other spelling — which made a
	// justice-aligned TARGET look like an enemy to the hooks that branch on it (the enemy branch
	// then paid an intercept bonus for acting against justice's own).
	//
	// "judge" is included because the lock-held lobby form this replaces declared it (a Judge is
	// a justice role, and battle_service.go gates Judge bonuses on that same JobRole).
	switch RoleKey(careerChoice) {
	case "bountyhunter", "intelagent", "aosleader", "justicerecruiter",
		"warden", "mutationlogauditor", "taxauditor", "sectorpeacekeeper",
		"justicecommissioner", "judge":
		return true
	default:
		return false
	}
}

// GetDecryptBonus returns the Arc-Net vision decryption multiplier for Intel-Agent at the given tier.
// Tier≥3 → 2.0× decrypted visibility; Tier≥5 → 3.0× (full sector sweep).
func GetDecryptBonus(tier int) float64 {
	switch {
	case tier >= 5: // Commissioner+ — full sector sweep
		return 3.0
	case tier >= 3: // Expert+ — double visibility
		return 2.0
	default: // Apprentice or below — baseline reveal
		return 1.0
	}
}

// GetEvidenceAccuracyBonus returns the evidence accuracy multiplier for Forensic Analyst at the given tier.
// Tier≥3 → 2.0× effectiveness (double-clean vs Gossip records).
func GetEvidenceAccuracyBonus(tier int) float64 {
	switch {
	case tier >= 3: // Expert+ — double-clean
		return 2.0
	default: // Apprentice or below — baseline evidence capture
		return 1.0
	}
}

// GetRecruitmentBonus returns the starting power multiplier for new justice-aligned players recruited by Justice Recruiter.
func GetRecruitmentBonus() float64 {
	return 1.05 // +5% starting power bonus per recruit
}

// ============================================================================
// PILLAR 13: HEIST PLANNER CAREER — GetPlanningBuff (Task 4201-9B)
// Returns the team heist success rate multiplier for Heist Planner at given tier.
// Tier≥1 → +5% per tier level (minimum ×1.0, scales linearly up to Boss tier).
// ============================================================================

// GetPlanningBuff returns the team planning buff multiplier for Heist Planner career tier.
// Returns base 1.0 if player is not a Heist Planner or has no CareerXP data.
func (cxp *CareerXP) GetPlanningBuff() float64 {
	if cxp == nil || cxp.RoleXP == nil {
		return 1.0 // No buff for non-HeistPlanner players
	}

	hasHP := false
	for rName := range cxp.RoleXP {
		if strings.EqualFold(rName, "Heist Planner") {
			hasHP = true
			break
		}
	}
	if !hasHP && !CareerHasRole(cxp, "Heist Planner") {
		return 1.0 // No buff for non-HeistPlanner players
	}

	tier := getTierFor(cxp, "Heist Planner")
	// +5% per tier level: Tier 1 = ×1.05, Tier 2 = ×1.10, ..., Boss (tier≥7) = ×1.35 cap
	buff := 1.0 + float64(tier)*0.05
	if buff > 1.35 {
		buff = 1.35 // Cap at Boss tier equivalent (+35% max team success rate)
	}
	return buff
}

// GetHeistDividendRate returns the dividend multiplier for Heist Planner based on career tier.
// Returns base 0.02 (2%) per member in planner's org, scaled by tier level.
func (cxp *CareerXP) GetHeistDividendRate() float64 {
	if cxp == nil || cxp.RoleXP == nil {
		return 0.0 // No dividend for non-Planner players
	}

	hasHP := false
	for rName := range cxp.RoleXP {
		if strings.EqualFold(rName, "Heist Planner") {
			hasHP = true
			break
		}
	}
	if !hasHP && !CareerHasRole(cxp, "Heist Planner") {
		return 0.0 // No dividend for non-Planner players
	}

	tier := getTierFor(cxp, "Heist Planner")
	switch {
	case tier >= 5: // Boss+ — maximum dividend share
		return 0.08
	case tier >= 4: // Master+ 
		return 0.06
	case tier >= 3: // Expert+
		return 0.05
	default: // Apprentice and below
		return 0.02 + float64(tier)*0.01 // Tier 1 = ×1%, Tier 2 = ×2%
	}
}

// HasHeistPlannerRole checks if the player has promoted to Heist Planner career role.
func (cxp *CareerXP) HasHeistPlannerRole() bool {
	if cxp == nil || cxp.RoleXP == nil {
		return false
	}
	for rName := range cxp.RoleXP {
		if strings.EqualFold(rName, "Heist Planner") {
			return true
		}
	}
	return CareerHasRole(cxp, "Heist Planner")
}

func (c *CareerXP) GetTransitTaxExemption() float64 {
	if !HasCareer(c, "Smuggler") {
		return 0.0
	}
	tier := c.GetCareerTier("Smuggler")
	switch {
	case tier >= 3:
		return 1.0 // 100% exemption
	case tier >= 2:
		return 0.50 // 50% exemption
	case tier >= 1:
		return 0.25 // 25% exemption
	default:
		return 0.0
	}
}

func (c *CareerXP) GetWardenDetentionBonus() float64 {
	if !HasCareer(c, "Warden") {
		return 1.0
	}
	tier := c.GetCareerTier("Warden")
	return 1.0 + float64(tier)*0.10 // 10% bonus per tier
}

func (c *CareerXP) GetEvidenceAccuracyBonus() float64 {
	if !HasCareer(c, "ForensicAnalyst") {
		return 1.0
	}
	tier := c.GetCareerTier("ForensicAnalyst")
	return 1.0 + float64(tier)*0.15 // 15% bonus per tier
}

