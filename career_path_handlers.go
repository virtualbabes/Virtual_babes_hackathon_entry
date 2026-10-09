//go:build !js && !wasm

package main

import (
	"encoding/json"
	"net/http"
)

// ============================================================================
// CAREER PATH · CIVIL RANK · PROMOTION — HTTP BOUNDARY
// ============================================================================
//
// Three doors, and every one of them resolves the caller the way every other door in this
// repository does (`getWalletFromRequest`: X-Wallet-Address header, then ?wallet=), keeps
// `X-Client-ID` only as a SECONDARY source, and resolves the STORED key case-insensitively
// (`leaderboardKeyLocked`) before it reads or writes anything.
//
// THE CLIENT MAY NAME EXACTLY TWO THINGS HERE: the path it chooses and the role it promotes into.
// Both bodies decode with `DisallowUnknownFields()`, so a payload carrying `civil_tier`,
// `promoted_roles`, `sustained_micro` or a price cannot even be parsed. Everything else — the
// civil rank, the eligibility, whether a promotion is earned, whether a career is demoted — is
// DERIVED from engine state by `career_path.go`, which is the domain owner.
// ============================================================================

// resolveCareerCaller resolves the calling wallet for these doors. Returns "" when unresolvable.
// It is a helper rather than four copies of the same five lines.
func (l *Lobby) resolveCareerCaller(r *http.Request) string {
	walletAddr := l.getWalletFromRequest(r)
	if walletAddr == "" {
		if clientID := r.Header.Get("X-Client-ID"); clientID != "" {
			walletAddr, _ = l.wallets[clientID]
		}
	}
	return walletAddr
}

// handleCareerPath serves GET /api/career/path — the whole career-path system as the SERVER
// understands it: the three paths and how they rival each other, the civil ladder, the caller's
// own eligibility, every declared career with the exact missing requirement, the promoted roles,
// the demotion history, any outstanding warning, and the three matrices with their owners.
//
// It is READ-ONLY and takes the read lock: no call in this path mutates anything.
func (l *Lobby) handleCareerPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONStatus(w, http.StatusMethodNotAllowed, map[string]interface{}{"success": false, "error": "GET required"})
		return
	}
	walletAddr := l.resolveCareerCaller(r)
	if walletAddr == "" {
		writeJSONStatus(w, http.StatusUnauthorized, map[string]interface{}{
			"success": false,
			"error":   "wallet required: send X-Wallet-Address or ?wallet=",
		})
		return
	}

	l.mutex.RLock()
	defer l.mutex.RUnlock()

	// Resolve the stored spelling so a mixed-case caller reads its OWN record rather than an empty
	// one (the rule every wallet-keyed read in this repository follows).
	walletAddr = l.leaderboardKeyLocked(walletAddr)
	view := l.CareerPathViewLocked(walletAddr)
	writeJSONStatus(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    view,
	})
}

// handleCareerPathChoose serves POST /api/career/path/choose — the ONE choice a player makes.
//
// The gate is enforced at the DOMAIN owner (`ChooseCareerPathLocked`), not here, so no handler,
// bot or future spawner can set a path round the door; this function only translates the outcome
// into a status code. A refusal names what is missing rather than answering "not allowed".
func (l *Lobby) handleCareerPathChoose(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONStatus(w, http.StatusMethodNotAllowed, map[string]interface{}{"success": false, "error": "POST required"})
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "invalid body: only `path` is accepted (the civil rank and the eligibility are derived by the server)",
		})
		return
	}

	walletAddr := l.resolveCareerCaller(r)
	if walletAddr == "" {
		writeJSONStatus(w, http.StatusUnauthorized, map[string]interface{}{
			"success": false,
			"error":   "wallet required: send X-Wallet-Address or ?wallet=",
		})
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	walletAddr = l.leaderboardKeyLocked(walletAddr)
	elig, err := l.ChooseCareerPathLocked(walletAddr, req.Path)
	if err != nil {
		status := http.StatusForbidden
		if !IsCareerPath(req.Path) {
			status = http.StatusBadRequest
		}
		writeJSONStatus(w, status, map[string]interface{}{
			"success":     false,
			"error":       err.Error(),
			"eligibility": elig,
		})
		return
	}
	writeJSONStatus(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"path":        elig.CurrentPath,
		"eligibility": elig,
	})
}

// handleCareerPromote serves POST /api/career/promote — the door `PromotedRoles` never had.
//
// The body may carry ONLY `role`. Every gate (the path the career belongs to, the level cap, the
// role tier, the sustained $VBV and the civil rank) is evaluated by `PromoteCareerLocked`, and a
// refusal returns that struct so the UI can state the ONE missing piece with its exact shortfall.
func (l *Lobby) handleCareerPromote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONStatus(w, http.StatusMethodNotAllowed, map[string]interface{}{"success": false, "error": "POST required"})
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "invalid body: only `role` is accepted (every gate is derived by the server)",
		})
		return
	}
	if req.Role == "" {
		writeJSONStatus(w, http.StatusBadRequest, map[string]interface{}{
			"success": false, "error": "role is required",
		})
		return
	}

	walletAddr := l.resolveCareerCaller(r)
	if walletAddr == "" {
		writeJSONStatus(w, http.StatusUnauthorized, map[string]interface{}{
			"success": false,
			"error":   "wallet required: send X-Wallet-Address or ?wallet=",
		})
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	walletAddr = l.leaderboardKeyLocked(walletAddr)
	req2, err := l.PromoteCareerLocked(walletAddr, req.Role)
	if err != nil {
		writeJSONStatus(w, http.StatusForbidden, map[string]interface{}{
			"success":      false,
			"error":        err.Error(),
			"requirements": req2,
		})
		return
	}
	writeJSONStatus(w, http.StatusOK, map[string]interface{}{
		"success":      true,
		"role":         req2.Role,
		"requirements": req2,
	})
}
// handleCareerStaffUpgradeRequest serves POST /api/career/staff/request — the EMPLOYER's half of the
// operator's rule. The body may carry ONLY `staff_wallet` and `role`, exactly as the other two doors
// may name only a path or a role; the unlock is DERIVED from the employee's own engine record.
//
// IT IS NOT A PROMOTION AND HAS NO ABILITY TO GRANT A ROLE. `RequestStaffCareerUpgradeLocked`
// refuses anything that is not an upgrade the EMPLOYEE is already UNLOCKED for, stores the request
// on the EMPLOYER's own club record, and notifies the employee. The employee's `PromotedRoles` and
// `JobRole` are never written here — that is "not forced" made mechanical, and a refusal quotes the
// employee's OWN missing gate so the employer can see the level cap has not opened it yet.
func (l *Lobby) handleCareerStaffUpgradeRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONStatus(w, http.StatusMethodNotAllowed, map[string]interface{}{"success": false, "error": "POST required"})
		return
	}

	var req struct {
		StaffWallet string `json:"staff_wallet"`
		Role        string `json:"role"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "invalid body: only `staff_wallet` and `role` are accepted (the unlock is derived by the server, and the upgrade itself belongs to the staff member)",
		})
		return
	}
	if req.StaffWallet == "" || req.Role == "" {
		writeJSONStatus(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   "staff_wallet and role are both required",
		})
		return
	}

	walletAddr := l.resolveCareerCaller(r)
	if walletAddr == "" {
		writeJSONStatus(w, http.StatusUnauthorized, map[string]interface{}{
			"success": false,
			"error":   "wallet required: send X-Wallet-Address or ?wallet=",
		})
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	walletAddr = l.leaderboardKeyLocked(walletAddr)
	notice, err := l.RequestStaffCareerUpgradeLocked(walletAddr, req.StaffWallet, req.Role)
	if err != nil {
		writeJSONStatus(w, http.StatusForbidden, map[string]interface{}{
			"success":  false,
			"error":    err.Error(),
			"upgrades": l.CareerUpgradeViewLocked(walletAddr),
		})
		return
	}
	writeJSONStatus(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"request":  notice,
		"upgrades": l.CareerUpgradeViewLocked(walletAddr),
	})
}
