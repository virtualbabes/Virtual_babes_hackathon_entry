//go:build !js && !wasm

package main

import (
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
)

// ============================================================================
// THE REWARD-TOKEN REGISTRY
//
// Before this file, a "reward token" was a bare key in two maps (initialRewards = the unscaled
// template, rewardStack = the scaled, payable set) with no record of what the asset WAS, who
// put it there, or whether it was still supposed to be paid. Three consequences were live:
//
//  1. ADMIN-CREATED TOKENS WERE UNSOURCED. `POST /api/reward/add` wrote `rewardStack[""]` for an
//     empty asset id and hard-coded the network to "VOI".
//  2. A TEMPLATE KEY COULD OUTLIVE ITS REGISTRATION. Keys restored from the economy snapshot
//     were scaled and PAID whether or not anything still registered them — so a retired token
//     kept paying forever.
//  3. THE PRIMARY TOKEN WAS RUNTIME-WRITABLE. It is not: REWARD_ASSET_ID is env-owned, so the
//     registry seeds it from the environment and refuses to let anything else own it.
//
// This file is the ONE owner of the descriptor, the reconciliation and the served view.
// ============================================================================

// Reward-token roles. PRIMARY is env-owned and may never be created from the panel; the other
// roles are admin-registered. LEGACY exists so a key recovered from an old snapshot can be
// carried deliberately instead of being silently paid or silently dropped.
const (
	RewardRolePrimary      = "primary"
	RewardRoleDistribution = "distribution"
	RewardRoleTenant       = "tenant"
	RewardRoleLegacy       = "legacy"
)

// Reward-token sources: where an entry's authority comes from.
const (
	RewardSourceEnv   = "env"
	RewardSourceAdmin = "admin"
)

// RewardToken is the registry DESCRIPTOR of one payable reward token: what the template entry
// IS, beside the amount the scaling pass computes for it.
type RewardToken struct {
	AssetID  string `json:"asset_id"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
	Network  string `json:"network"`
	Role     string `json:"role"`
	Source   string `json:"source"`
	Enabled  bool   `json:"enabled"`
	// When the vault's opt-in to this asset was last VERIFIED (unix seconds, 0 = never verified
	// by this build). It is a recorded fact, never a live re-check on every read: an opt-out that
	// happens later does not silently rewrite history.
	OptInVerifiedAtUnix int64 `json:"vault_opt_in_verified_at_unix"`
}

// RewardTokenView is the SERVED projection: the descriptor plus the amounts and whether the
// token is payable right now.
type RewardTokenView struct {
	AssetID      string `json:"asset_id"`
	Symbol       string `json:"symbol"`
	Decimals     int    `json:"decimals"`
	Network      string `json:"network"`
	Role         string `json:"role"`
	Source       string `json:"source"`
	Enabled      bool   `json:"enabled"`
	InitialMicro uint64 `json:"initial_micro"`
	ScaledMicro  uint64 `json:"scaled_micro"`
	Payable      bool   `json:"payable"`
}

// rewardRoleRank orders the served list: the env-owned primary first, then the admin roles in
// declaration order, so a refresh never looks like a change.
func rewardRoleRank(role string) int {
	switch role {
	case RewardRolePrimary:
		return 0
	case RewardRoleDistribution:
		return 1
	case RewardRoleTenant:
		return 2
	case RewardRoleLegacy:
		return 3
	default:
		return 4
	}
}

// adminCreatableRewardRole reports whether the panel may create an entry with this role. PRIMARY
// is refused (env-owned) and an unknown role is refused (fail closed): a role that does not exist
// cannot be invented by a request body.
func adminCreatableRewardRole(role string) bool {
	return role == RewardRoleDistribution || role == RewardRoleTenant
}

// seedPrimaryRewardTokenLocked guarantees the env-owned primary entry exists, is ENABLED and is
// sourced from the environment. It is the ONLY place the primary token is created, and it never
// lets a restored value overwrite the environment's authority.
func (l *Lobby) seedPrimaryRewardTokenLocked() {
	if l.rewardTokens == nil {
		l.rewardTokens = make(map[string]RewardToken, 4)
	}
	if l.rewardAssetID == "" {
		// Nothing to seed. The boot log states the misconfiguration rather than inventing an id.
		return
	}

	entry, known := l.rewardTokens[l.rewardAssetID]
	if !known {
		// ARC-200 on Voi reports in micro units throughout this codebase (PowerDivisor 1e6), so
		// six decimals is the honest default rather than a guess about a specific asset.
		entry = RewardToken{Decimals: 6}
	}
	// The SYMBOL is only claimed when the environment itself says the primary asset IS the game's
	// $VBV (VBV_ASSET_ID); otherwise an operator-registered asset would be mislabelled.
	if strings.TrimSpace(entry.Symbol) == "" && l.rewardAssetID == strings.TrimSpace(os.Getenv("VBV_ASSET_ID")) {
		entry.Symbol = "VBV"
	}
	entry.AssetID = l.rewardAssetID
	entry.Role = RewardRolePrimary
	entry.Source = RewardSourceEnv
	entry.Enabled = true
	l.rewardTokens[l.rewardAssetID] = entry
}

// reconcileRewardRegistryLocked makes the TEMPLATE agree with the REGISTRY:
//
//   - the env primary entry is present and carries the environment's base reward;
//   - a template key with no enabled registry entry is DROPPED (this is what stops a retired or
//     stale token from being paid forever);
//   - the pruned ids are returned so the caller can REPORT them.
//
// The caller must hold l.mutex.
func (l *Lobby) reconcileRewardRegistryLocked() []string {
	if l.initialRewards == nil {
		l.initialRewards = make(map[string]uint64)
	}
	l.seedPrimaryRewardTokenLocked()

	if l.rewardAssetID != "" {
		l.initialRewards[l.rewardAssetID] = l.initialBaseReward
	}

	var pruned []string
	for assetID := range l.initialRewards {
		if assetID == "" {
			delete(l.initialRewards, assetID)
			pruned = append(pruned, "(empty asset id)")
			continue
		}
		entry, registered := l.rewardTokens[assetID]
		if !registered || !entry.Enabled {
			delete(l.initialRewards, assetID)
			pruned = append(pruned, assetID)
		}
	}
	sort.Strings(pruned)
	return pruned
}

// rewardTokenViewsLocked serves the registry with the LIVE amounts beside the template ones.
// The caller must hold l.mutex (a read lock is sufficient).
func (l *Lobby) rewardTokenViewsLocked() []RewardTokenView {
	views := make([]RewardTokenView, 0, len(l.rewardTokens))
	for assetID, entry := range l.rewardTokens {
		initial, inTemplate := l.initialRewards[assetID]
		views = append(views, RewardTokenView{
			AssetID:      assetID,
			Symbol:       entry.Symbol,
			Decimals:     entry.Decimals,
			Network:      entry.Network,
			Role:         entry.Role,
			Source:       entry.Source,
			Enabled:      entry.Enabled,
			InitialMicro: initial,
			ScaledMicro:  l.rewardStack[assetID],
			Payable:      entry.Enabled && inTemplate,
		})
	}
	sort.Slice(views, func(i, j int) bool {
		ri, rj := rewardRoleRank(views[i].Role), rewardRoleRank(views[j].Role)
		if ri != rj {
			return ri < rj
		}
		return views[i].AssetID < views[j].AssetID
	})
	return views
}

// rewardTokenViews serves the registry to an HTTP caller (takes the read lock).
func (l *Lobby) rewardTokenViews() []RewardTokenView {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.rewardTokenViewsLocked()
}

// currentRewardAssetID reads the env-owned primary asset id under the read lock.
func (l *Lobby) currentRewardAssetID() string {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.rewardAssetID
}

// isRewardAssetIDShape reports whether a registry key can name an asset at all: a non-empty
// DECIMAL id. "0" and "" are refused because the pre-registry handler happily wrote
// `rewardStack[""]`, a key nothing can ever pay and nothing can ever match.
func isRewardAssetIDShape(assetID string) bool {
	id := strings.TrimSpace(assetID)
	if id == "" || id == "0" {
		return false
	}
	for i := 0; i < len(id); i++ {
		if id[i] < '0' || id[i] > '9' {
			return false
		}
	}
	return true
}

// parseRewardEnvMicro reads an environment amount as EXACT integer micro units.
//
// Why this exists: BASE_REWARD was parsed with strconv.ParseUint, so the documented spelling
// "5.0" (see .env.example) FAILED, the error was DISCARDED, and the base reward silently became
// 0 — a dead payout path that reads like a funding problem. This parser is integer-only (the
// ledger rule): whole units plus an optional fraction of up to six digits, converted by digit
// arithmetic, never by float multiplication. It answers ok=false WITH A NOTE rather than
// inventing a value.
func parseRewardEnvMicro(raw string) (uint64, bool, string) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, false, "not set"
	}
	value = strings.TrimPrefix(value, "+")

	whole, fraction := value, ""
	if dot := strings.IndexByte(value, '.'); dot >= 0 {
		whole, fraction = value[:dot], value[dot+1:]
		if len(fraction) > 6 {
			return 0, false, "more than six decimal places"
		}
	}
	if whole == "" && fraction == "" {
		return 0, false, "no digits"
	}
	for _, part := range []string{whole, fraction} {
		for i := 0; i < len(part); i++ {
			if part[i] < '0' || part[i] > '9' {
				return 0, false, "not a decimal number"
			}
		}
	}
	if whole == "" {
		whole = "0"
	}

	units, err := strconv.ParseUint(whole, 10, 64)
	if err != nil {
		return 0, false, "whole units do not fit in uint64"
	}
	frac, err := strconv.ParseUint((fraction+"000000")[:6], 10, 64)
	if err != nil {
		return 0, false, "fraction is not numeric"
	}
	if units > (^uint64(0)-frac)/1000000 {
		return 0, false, "value overflows micro units"
	}
	return units*1000000 + frac, true, ""
}

// resolveEnvBaseRewardMicro reads BASE_REWARD_MICRO (exact micro units) or BASE_REWARD (whole
// units, optional fraction), STATES the outcome at boot, and returns 0 only with a stated reason.
// A misconfigured payout must be readable in the log, not inferred from silence.
func resolveEnvBaseRewardMicro() uint64 {
	if rawMicro := strings.TrimSpace(os.Getenv("BASE_REWARD_MICRO")); rawMicro != "" {
		micro, err := strconv.ParseUint(rawMicro, 10, 64)
		if err != nil {
			log.Printf("[REWARD CONFIG ERROR] BASE_REWARD_MICRO=%q is not an integer; IGNORED, falling back to BASE_REWARD.\n", rawMicro)
		} else {
			return micro
		}
	}

	raw := strings.TrimSpace(os.Getenv("BASE_REWARD"))
	micro, ok, note := parseRewardEnvMicro(raw)
	if !ok {
		log.Printf("[REWARD CONFIG WARNING] the base reward could not be read (%s), so it is 0 micro-VBV: every un-boosted payout is 0. "+
			"Set BASE_REWARD in WHOLE units (e.g. BASE_REWARD=5) or BASE_REWARD_MICRO=5000000.\n", note)
		return 0
	}
	if strings.Contains(raw, ".") {
		log.Printf("[REWARD CONFIG] BASE_REWARD=%q read as %d micro-VBV (a fraction is honoured exactly; the documented spelling "+
			"used to parse as 0). Prefer BASE_REWARD=%d or BASE_REWARD_MICRO=%d.\n", raw, micro, micro/1000000, micro)
	}
	return micro
}
