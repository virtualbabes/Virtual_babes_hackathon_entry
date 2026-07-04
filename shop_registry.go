//go:build !js && !wasm

package main

import "sync"

var dlcRegistryMutex sync.RWMutex // PILLAR 4: Protects DLCRegistry for concurrent access

// DLCProduct defines a console-redeemable item and its associated economic parameters.
// PILLAR 2: Integer Supremacy.
type DLCProduct struct {
	ArenaVoucherID string `json:"arena_voucher_id"` // Unique product code from console store
	Name           string `json:"name"`             // Display name of the DLC
	Description    string `json:"description"`      // Description for console UI
	CostMicro      uint64 `json:"cost_micro"`       // Cost in micro-Arena Vouchers (1:1 with micro-VBV)
	CreatorWallet  string `json:"creator_wallet"`   // Wallet of the browser-based creator to be paid
}

// DLCRegistry maps ArenaVoucherIDs to their corresponding DLCProduct details.
var DLCRegistry map[string]DLCProduct

type ShopItem struct {
	ID                      string  `json:"id"`
	Name                    string  `json:"name"`
	Description             string  `json:"description"`                         // Item description
	Price                   float64 `json:"price"`                               // Base price in $VBV
	ClubType                string  `json:"club_type"`                           // Elemental, Tactical, Vitality, Hardware
	HeistSuccessModifier    float64 `json:"heist_success_modifier,omitempty"`    // Modifier for heist success chance (e.g., -0.10 for 10% reduction)
	MutationSuccessModifier float64 `json:"mutation_success_modifier,omitempty"` // PILLAR 6: Bonus to gene-editing stability
	MojoBonus               int     `json:"mojo_bonus,omitempty"`                // Mojo gained by the club upon turnover
	RequiredMojo            int     `json:"required_mojo,omitempty"`             // Club Mojo threshold to unlock item
	RequiredRole            string  `json:"required_role,omitempty"`             // Career role required to purchase
	IsMasterTier            bool    `json:"is_master_tier,omitempty"`            // Requires Regional Governor status (2+ districts)
}

var GlobalShopRegistry = map[string]ShopItem{
	// Elemental Forge
	"mood_catalyst": {
		ID: "mood_catalyst", Name: "Mood Catalyst", Price: 100, ClubType: "Elemental",
		Description: "+50 Mood Bonus (3 Matches)", MojoBonus: 5,
	},
	"grounded_shield": {
		ID: "grounded_shield", Name: "Grounded Shield", Price: 250, ClubType: "Elemental",
		Description: "Immunity to Mood Penalties (5 Matches)", MojoBonus: 12, RequiredMojo: 100,
	},
	"prism_shield": {
		ID: "prism_shield", Name: "Prism Shield", Price: 750, ClubType: "Elemental",
		Description: "Reflects Mood Penalties back to Opponent", MojoBonus: 35,
		RequiredMojo: 500, IsMasterTier: true,
	},

	// Tactical Syndicate
	"rule_breaker": {
		ID: "rule_breaker", Name: "Rule Breaker", Price: 150, ClubType: "Tactical",
		Description: "Force PLUS trigger (1 Match)", MojoBonus: 8,
	},
	"intel_report": {
		ID: "intel_report", Name: "Intel Report", Price: 300, ClubType: "Tactical",
		Description: "See Opponent Hand (3 Matches)", MojoBonus: 15, RequiredMojo: 150,
	},
	"cyber_audit": {
		ID: "cyber_audit", Name: "Cyber-Audit", Price: 400, ClubType: "Intelligence",
		Description: "Reveal target club's treasury average and crash status.", MojoBonus: 20,
		RequiredMojo: 200, RequiredRole: "Manager",
	},
	"deep_scan_decryptor": {
		ID: "deep_scan_decryptor", Name: "Deep-Scan Decryptor", Price: 250, ClubType: "Intelligence",
		Description: "Reveal target signature's inventory and active buffs for 5 minutes.", MojoBonus: 20,
		RequiredMojo: 150, RequiredRole: "Intel-Agent",
	},
	"district_scanner": {
		ID: "district_scanner", Name: "District Scanner", Price: 600, ClubType: "Intelligence",
		Description: "Reveal all active hardware traps across all territories for 10 minutes.", MojoBonus: 25,
		RequiredMojo: 300, RequiredRole: "Manager",
	},
	"cloak_disruptor": {
		ID: "cloak_disruptor", Name: "Cloak Disruptor", Price: 250, ClubType: "Intelligence",
		Description: "Temporarily reveal a target outlaw's signal on the Bounty Board for 5 minutes.", MojoBonus: 15,
		RequiredMojo: 200,
	},
	"cyber_jammer": {
		ID: "cyber_jammer", Name: "Cyber-Jammer", Price: 750, ClubType: "Intelligence",
		Description: "Prevents the 'Sabotage Warning' from being sent for a single sabotage attempt.", MojoBonus: 20,
		RequiredMojo: 250,
	},
	"ghost_protocol": {
		ID: "ghost_protocol", Name: "Ghost Protocol", Price: 1000, ClubType: "Tactical",
		Description: "Match outcome hidden from Market Ticker", MojoBonus: 50,
		RequiredMojo: 600, RequiredRole: "Security", IsMasterTier: true,
	},
	"district_stabilizer": {
		ID: "district_stabilizer", Name: "District Stabilizer", Price: 1500, ClubType: "Tactical",
		Description: "Reduces Mojo decay rate by 50% for 48 hours.", MojoBonus: 40,
		RequiredRole: "Manager", RequiredMojo: 400,
	},
	"mutation_insurance": {
		ID: "mutation_insurance", Name: "Mutation Insurance", Price: 1500, ClubType: "Tactical",
		Description: "Guarantees 100% success rate for your next gene-editing procedure.", MojoBonus: 50,
		RequiredRole: "Manager", RequiredMojo: 500,
	},

	// Vitality Lab
	"stamina_stim": {
		ID: "stamina_stim", Name: "Stamina Stim", Price: 100, ClubType: "Vitality",
		Description: "-20 Fatigue Immediately", MojoBonus: 5,
	},
	"loyalty_pledge": {
		ID: "loyalty_pledge", Name: "Loyalty Pledge", Price: 500, ClubType: "Vitality",
		Description: "+10 Loyalty Immediately", MojoBonus: 25, RequiredMojo: 200,
	},
	"hyper_stim": {
		ID: "hyper_stim", Name: "Hyper-Stim", Price: 1500, ClubType: "Vitality",
		Description: "Resets fatigue for entire current deck", MojoBonus: 75,
		RequiredMojo: 800, RequiredRole: "Manager", IsMasterTier: true,
	},
	"staff_training": {
		ID: "staff_training", Name: "Staff Training", Price: 800, ClubType: "Vitality",
		Description: "Increases card mutation success rate by +5% for 24 hours.", MojoBonus: 30,
		MutationSuccessModifier: 0.05, RequiredRole: "Manager", RequiredMojo: 300,
	},
	"forensic_audit_kit": {
		ID: "forensic_audit_kit", Name: "Forensic Audit Kit", Price: 1000, ClubType: "Intelligence",
		Description: "Reduces a target card's Artifact by 50 points. For Justice-aligned auditors.", MojoBonus: 25,
		RequiredRole: "Forensic Analyst", RequiredMojo: 300,
	},
	"audit_shield": {
		ID: "audit_shield", Name: "Audit Shield", Price: 2500, ClubType: "Tactical",
		Description: "Protects the organization from Justice flagging (e.g. MISSION-025) for 24 hours.",
		MojoBonus:   50, RequiredRole: "Manager", RequiredMojo: 500,
	},
	"legal_pardon": {
		ID: "legal_pardon", Name: "Legal Pardon", Price: 2000, ClubType: "Intelligence",
		Description: "Clears 50% of a target's Wanted Level. Role: Judge.",
		MojoBonus:   60, RequiredRole: "Judge", RequiredMojo: 500,
	},
	"market_freeze": {
		ID: "market_freeze", Name: "Market Freeze", Price: 2000, ClubType: "Intelligence",
		Description: "Disables share trading for a target with Wanted Level 20+ for 24 hours. Role: Tax Auditor.",
		MojoBonus:   50, RequiredRole: "Tax Auditor", RequiredMojo: 500,
	},
	"raid_jammer": {
		ID: "raid_jammer", Name: "Raid Jammer", Price: 500, ClubType: "Hardware",
		Description: "Reduces AOS Raid success chance against your organization by 15% for 12 hours. Role: Hostage Host.",
		MojoBonus:   20, RequiredRole: "Hostage Host", RequiredMojo: 300,
	},
	"tax_haven_license": {
		ID: "tax_haven_license", Name: "Tax Haven License", Price: 1000, ClubType: "Tactical",
		Description: "Exempts club members from the 1% Exchange Fee for 48 hours. Req: Club 5,000+ VBV.",
		MojoBonus:   40, RequiredRole: "Manager", RequiredMojo: 400,
	},
	"signal_dampener": {
		ID: "signal_dampener", Name: "Signal Dampener", Price: 300, ClubType: "Intelligence",
		Description: "Scrambles signals for all club members, hiding them from the Bounty Board for 24 hours.",
		MojoBonus:   15, RequiredRole: "Hostage Host", RequiredMojo: 250,
	},
	"regulatory_bypass_permit": {
		ID: "regulatory_bypass_permit", Name: "Regulatory Bypass Permit", Price: 1000, ClubType: "Underworld_Admin",
		Description: "Reduces Corporate Tax for a target club by 50% for 24 hours. Role: Lawyer-Commissioner.",
		MojoBonus:   50, RequiredRole: "Lawyer-Commissioner", RequiredMojo: 500,
	},
	"illicit_commission_permit": {
		ID: "illicit_commission_permit", Name: "Illicit Commission Permit", Price: 750, ClubType: "Underworld_Admin",
		Description: "Sets a target Tactical club's commission rate to 1% for 24 hours. Role: Lawyer-Commissioner.",
		MojoBonus:   40, RequiredRole: "Lawyer-Commissioner", RequiredMojo: 400,
	},

	// Hardware / Security
	"tripwire": {
		ID: "tripwire", Name: "Laser Tripwire", Price: 500, ClubType: "Hardware",
		Description: "+10% Heist Failure", HeistSuccessModifier: -0.10, MojoBonus: 20,
		RequiredRole: "Security",
	},
	"cyber_counter": {
		ID: "cyber_counter", Name: "Cyber-Counter", Price: 800, ClubType: "Hardware",
		Description: "Identifies player using Cyber-Audit against this club.", MojoBonus: 30,
		RequiredRole: "Security", RequiredMojo: 250,
	},
	"cyber_lock": {
		ID: "cyber_lock", Name: "Cyber-Lock", Price: 1000, ClubType: "Hardware",
		Description: "Prevents all Cyber-Audits against the club for 24 hours.", MojoBonus: 40,
		RequiredRole: "Security", RequiredMojo: 350,
	},
	"sentry_turret": {
		ID: "sentry_turret", Name: "Sentry Turret", Price: 1200, ClubType: "Hardware",
		Description: "+25% Heist Failure", HeistSuccessModifier: -0.25, MojoBonus: 45,
		RequiredRole: "Security", RequiredMojo: 300,
	},
	"guard_dog": {
		ID: "guard_dog", Name: "Bio-Guard Dog", Price: 2000, ClubType: "Hardware",
		Description: "Forces Jail on Failure", HeistSuccessModifier: -0.05, MojoBonus: 80,
		RequiredRole: "Security", RequiredMojo: 500, IsMasterTier: true,
	},

	// Justice Hegemony Path (Pillar 7)
	"justice_enforcer_card": {
		ID: "justice_enforcer_card", Name: "Enforcer Card (Justice)", Price: 2500, ClubType: "Justice",
		Description: "+10% power vs Outlaws. Justice faction archetype card.", MojoBonus: 60,
		RequiredRole: "Enforcer", RequiredMojo: 300,
	},
	"justice_warden_card": {
		ID: "justice_warden_card", Name: "Warden Card (Justice)", Price: 5000, ClubType: "Justice",
		Description: "+15% power vs Outlaws. Higher-tier Justice archetype card.", MojoBonus: 80,
		RequiredRole: "Warden", RequiredMojo: 600, IsMasterTier: true,
	},
	"justice_commissioner_card": {
		ID: "justice_commissioner_card", Name: "Commissioner Card (Justice)", Price: 10000, ClubType: "Justice",
		Description: "+25% power vs Outlaws. Supreme Justice archetype card.", MojoBonus: 120,
		RequiredRole: "Commissioner", RequiredMojo: 1000, IsMasterTier: true,
	},
	"truth_serum": {
		ID: "truth_serum", Name: "Truth Serum", Price: 1500, ClubType: "Justice",
		Description: "Reveals target's active buffs and debuffs for 30 seconds.", MojoBonus: 40,
		RequiredRole: "Intel-Agent", RequiredMojo: 200,
	},
	"reputation_shield": {
		ID: "reputation_shield", Name: "Reputation Shield", Price: 3000, ClubType: "Justice",
		Description: "Blocks up to 50 reputation loss for 24 hours.", MojoBonus: 50,
		RequiredRole: "Manager", RequiredMojo: 400,
	},

	// P2-D7: Intel-Agent items
	"intel_intercept_kit": {
		ID: "intel_intercept_kit", Name: "Cyber-Intercept Kit", Price: 1500, ClubType: "Justice",
		Description: "Enables cyber-intercept events for Intel-Agent role. Decrypts Arc-Net vision data.", MojoBonus: 40,
		RequiredRole: "Intel-Agent", RequiredMojo: 300,
	},
	"arc_net_vision_boost": {
		ID: "arc_net_vision_boost", Name: "Arc-Net Vision Booster", Price: 800, ClubType: "Intelligence",
		Description: "+1 decrypt depth for Intel-Agent. Extends Arc-Net spy vision duration by 50%.", MojoBonus: 35,
		RequiredRole: "Intel-Agent", RequiredMojo: 250,
	},

	// P2-D8: Justice Recruiter items
	"recruiter_outreach_terminal": {
		ID: "recruiter_outreach_terminal", Name: "Recruitment Outreach Terminal", Price: 1200, ClubType: "Justice",
		Description: "Expands Justice Recruiter recruitment range by +1 tile. Grants recruiting bonuses.", MojoBonus: 30,
		RequiredRole: "Justice Recruiter", RequiredMojo: 250,
	},
	"recruiter_power_amplifier": {
		ID: "recruiter_power_amplifier", Name: "Recruitment Power Amplifier", Price: 600, ClubType: "Justice",
		Description: "+5% recruitment power bonus for Justice Recruiter on each successful recruit.", MojoBonus: 25,
		RequiredRole: "Justice Recruiter", RequiredMojo: 200,
	},

	// P2-D9: Justice Commissioner items
	"commissioner_override_warrant": {
		ID: "commissioner_override_warrant", Name: "Override Warrant", Price: 2000, ClubType: "Justice",
		Description: "Grants Justice Commissioner regulatory override capability. Modifies Tax Auditor fiscal actions.", MojoBonus: 60,
		RequiredRole: "Justice Commissioner", RequiredMojo: 500,
	},
	"commissioner_authority_seal": {
		ID: "commissioner_authority_seal", Name: "Authority Seal", Price: 1500, ClubType: "Justice",
		Description: "+2 jurisdiction radius tiles for Justice Commissioner. Enables broader regulatory coverage.", MojoBonus: 50,
		RequiredRole: "Justice Commissioner", RequiredMojo: 400,
	},

	// P2-D10: Mutation Log Auditor items
	"mutation_audit_terminal": {
		ID: "mutation_audit_terminal", Name: "Mutation Audit Terminal", Price: 1800, ClubType: "Intelligence",
		Description: "Enables mutation log auditing. Reveals suppressed genetic data with tier-aware accuracy.", MojoBonus: 45,
		RequiredRole: "Mutation Log Auditor", RequiredMojo: 350,
	},
	"genetic_reveal_scanner": {
		ID: "genetic_reveal_scanner", Name: "Genetic Reveal Scanner", Price: 1000, ClubType: "Intelligence",
		Description: "+25% mutation data reveal percentage for Mutation Log Auditor audit events.", MojoBonus: 40,
		RequiredRole: "Mutation Log Auditor", RequiredMojo: 300,
	},

	// Underworld #9: Counterfeiter items
	"counterfeit_plate": {
		ID: "counterfeit_plate", Name: "Counterfeit Press Plate", Price: 2500, ClubType: "Underworld_Admin",
		Description: "High-quality printing plate for counterfeit currency generation. Reduces detection chance by 10%.", MojoBonus: 55,
		RequiredRole: "Counterfeiter", RequiredMojo: 400,
	},
	"ink_scrambler": {
		ID: "ink_scrambler", Name: "Ink-Scrambler Device", Price: 1500, ClubType: "Underworld_Admin",
		Description: " UV-reactive ink scrambler for counterfeit notes. Reduces Forensic Analyst detection by 15%.", MojoBonus: 40,
		RequiredRole: "Counterfeiter", RequiredMojo: 300,
	},
	"purity_booster": {
		ID: "purity_booster", Name: "Note Purity Booster", Price: 3000, ClubType: "Underworld_Admin",
		Description: "Chemical treatment for counterfeit notes. Reduces detection chance by an additional 25%.", MojoBonus: 65,
		RequiredRole: "Counterfeiter", RequiredMojo: 500, IsMasterTier: true,
	},

	// Underworld #9: Counterfeit detection items (Justice side)
	"uv_spectrometer": {
		ID: "uv_spectrometer", Name: "UV Spectrometer", Price: 1200, ClubType: "Intelligence",
		Description: "Detects UV-reactive ink anomalies in currency. +20% counterfeit detection chance.", MojoBonus: 35,
		RequiredRole: "Forensic Analyst", RequiredMojo: 250,
	},
	"authenticity_scanner": {
		ID: "authenticity_scanner", Name: "Currency Authenticity Scanner", Price: 2000, ClubType: "Intelligence",
		Description: "Full-spectrum currency scanner for Warden. +30% counterfeit detection chance.", MojoBonus: 50,
		RequiredRole: "Warden", RequiredMojo: 400,
	},
}

// Hardware / Security

func init() {
	DLCRegistry = make(map[string]DLCProduct) // Initialize the map
	// Initialize the DLCRegistry with example products.
	// In a production environment, this would likely be loaded from a persistent store
	// or dynamically updated by browser-based creators.
	// PILLAR 2: Integer Supremacy for costs.
	dlcRegistryMutex.Lock()
	DLCRegistry["DLC_SKIN_CYBER_NEON"] = DLCProduct{ // PILLAR 4: Initial population
		ArenaVoucherID: "DLC_SKIN_CYBER_NEON",
		Name:           "Cyber Neon Skin Pack",
		Description:    "Unlock exclusive neon-themed card skins.",
		CostMicro:      500 * 1000000, // 500 Arena Vouchers
		CreatorWallet:  "browser_creator_wallet_001",
	}
	DLCRegistry["DLC_ARENA_GLITCH"] = DLCProduct{ // PILLAR 4: Initial population
		ArenaVoucherID: "DLC_ARENA_GLITCH",
		Name:           "Glitch Arena Map",
		Description:    "New combat environment with dynamic glitch effects.",
		CostMicro:      1000 * 1000000, // 1000 Arena Vouchers
		CreatorWallet:  "browser_creator_wallet_002",
	}
	dlcRegistryMutex.Unlock()
}
