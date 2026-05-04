package main

type ShopItem struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"` // Item description
	Price       float64 `json:"price"`       // Base price in $VBV
	ClubType    string  `json:"club_type"`   // Elemental, Tactical, Vitality, Hardware
	HeistSuccessModifier float64 `json:"heist_success_modifier,omitempty"` // Modifier for heist success chance (e.g., -0.10 for 10% reduction)
}

var GlobalShopRegistry = map[string]ShopItem{
	// Elemental Forge
	"mood_catalyst": {
		ID: "mood_catalyst", Name: "Mood Catalyst", Price: 100, ClubType: "Elemental",
		Description: "+50 Mood Bonus (3 Matches)", HeistSuccessModifier: 0,
	},
	"grounded_shield": {
		ID: "grounded_shield", Name: "Grounded Shield", Price: 250, ClubType: "Elemental",
		Description: "Immunity to Mood Penalties (5 Matches)", HeistSuccessModifier: 0,
	},

	// Tactical Syndicate
	"rule_breaker": {
		ID: "rule_breaker", Name: "Rule Breaker", Price: 150, ClubType: "Tactical",
		Description: "Force PLUS trigger (1 Match)", HeistSuccessModifier: 0,
	},
	"intel_report": {
		ID: "intel_report", Name: "Intel Report", Price: 300, ClubType: "Tactical",
		Description: "See Opponent Hand (3 Matches)", HeistSuccessModifier: 0,
	},

	// Vitality Lab
	"stamina_stim": {
		ID: "stamina_stim", Name: "Stamina Stim", Price: 100, ClubType: "Vitality",
		Description: "-20 Fatigue Immediately",
	},
	"loyalty_pledge": {
		ID: "loyalty_pledge", Name: "Loyalty Pledge", Price: 500, ClubType: "Vitality",
		Description: "+10 Loyalty Immediately", HeistSuccessModifier: 0,
	},

	// Hardware / Security
	"tripwire": {
		ID: "tripwire", Name: "Laser Tripwire", Price: 500, ClubType: "Hardware",
		Description: "+10% Heist Failure", HeistSuccessModifier: -0.10,
	},
	"sentry_turret": {
		ID: "sentry_turret", Name: "Sentry Turret", Price: 1200, ClubType: "Hardware",
		Description: "+25% Heist Failure", HeistSuccessModifier: -0.25,
	},
	"guard_dog": {
		ID: "guard_dog", Name: "Bio-Guard Dog", Price: 2000, ClubType: "Hardware",
		Description: "Forces Jail on Failure", HeistSuccessModifier: -0.05, // Small direct success reduction, main effect is post-failure
	},
}

	// Hardware / Security
