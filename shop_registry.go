package main

type ShopItem struct {
	ID                   string  `json:"id"`
	Name                 string  `json:"name"`
	Description          string  `json:"description"`                      // Item description
	Price                float64 `json:"price"`                            // Base price in $VBV
	ClubType             string  `json:"club_type"`                        // Elemental, Tactical, Vitality, Hardware
	HeistSuccessModifier float64 `json:"heist_success_modifier,omitempty"` // Modifier for heist success chance (e.g., -0.10 for 10% reduction)
	MojoBonus            int     `json:"mojo_bonus,omitempty"`             // Mojo gained by the club upon turnover
	RequiredMojo         int     `json:"required_mojo,omitempty"`          // Club Mojo threshold to unlock item
	RequiredRole         string  `json:"required_role,omitempty"`          // Career role required to purchase
	IsMasterTier         bool    `json:"is_master_tier,omitempty"`         // Requires Regional Governor status (2+ districts)
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
	"ghost_protocol": {
		ID: "ghost_protocol", Name: "Ghost Protocol", Price: 1000, ClubType: "Tactical",
		Description: "Match outcome hidden from Market Ticker", MojoBonus: 50,
		RequiredMojo: 600, RequiredRole: "Security", IsMasterTier: true,
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

	// Hardware / Security
	"tripwire": {
		ID: "tripwire", Name: "Laser Tripwire", Price: 500, ClubType: "Hardware",
		Description: "+10% Heist Failure", HeistSuccessModifier: -0.10, MojoBonus: 20,
		RequiredRole: "Security",
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
}

// Hardware / Security
