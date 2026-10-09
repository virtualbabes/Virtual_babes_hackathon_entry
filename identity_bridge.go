//go:build !js && !wasm

package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

// IdentityBridge owns server-authoritative cross-platform identity links.
type IdentityBridge struct{}

type identityLinkRequest struct {
	PrimaryWallet string `json:"primary_wallet"`
	Address       string `json:"address"`
	Chain         string `json:"chain"`
	Platform      string `json:"platform"`
}

type identityUnlinkRequest struct {
	PrimaryWallet string `json:"primary_wallet"`
	Address       string `json:"address"`
	Chain         string `json:"chain"`
}

func NewIdentityBridge() *IdentityBridge { return &IdentityBridge{} }

func normalizeIdentityPart(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validateIdentityInput(primary, address, chain, platform string) (string, string, string, string, error) {
	primary = normalizeIdentityPart(primary)
	address = normalizeIdentityPart(address)
	chain = normalizeIdentityPart(chain)
	platform = normalizeIdentityPart(platform)
	if primary == "" || address == "" || chain == "" || platform == "" {
		return "", "", "", "", errors.New("primary_wallet, address, chain, and platform are required")
	}
	if len(primary) > 128 || len(address) > 256 || len(chain) > 32 || len(platform) > 32 {
		return "", "", "", "", errors.New("identity field exceeds maximum length")
	}
	return primary, address, chain, platform, nil
}

func (b *IdentityBridge) HandleLink(l *Lobby, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req identityLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}
	primary, address, chain, platform, err := validateIdentityInput(req.PrimaryWallet, req.Address, req.Chain, req.Platform)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()
	for owner, info := range l.linkedWallets {
		for _, linked := range info.Linked {
			if normalizeIdentityPart(linked.Address) == address && normalizeIdentityPart(linked.Chain) == chain && owner != primary {
				http.Error(w, "identity is already linked to another primary wallet", http.StatusConflict)
				return
			}
		}
	}
	info := l.linkedWallets[primary]
	info.PrimaryAVMWallet = primary
	updated := false
	for i := range info.Linked {
		if normalizeIdentityPart(info.Linked[i].Address) == address && normalizeIdentityPart(info.Linked[i].Chain) == chain {
			info.Linked[i] = LinkedWallet{Address: address, Chain: chain, Platform: platform, Verified: true, Timestamp: time.Now().UTC()}
			updated = true
			break
		}
	}
	if !updated {
		info.Linked = append(info.Linked, LinkedWallet{Address: address, Chain: chain, Platform: platform, Verified: true, Timestamp: time.Now().UTC()})
	}
	l.linkedWallets[primary] = info
	l.saveBlockchainStateSnapshotLocked(NotePrefixLinkSnapshot, l.linkedWallets)
	writeIdentityJSON(w, http.StatusOK, info)
}

func (b *IdentityBridge) HandleUnlink(l *Lobby, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req identityUnlinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}
	primary := normalizeIdentityPart(req.PrimaryWallet)
	address := normalizeIdentityPart(req.Address)
	chain := normalizeIdentityPart(req.Chain)
	if primary == "" || address == "" || chain == "" {
		http.Error(w, "primary_wallet, address, and chain are required", http.StatusBadRequest)
		return
	}
	if len(primary) > 128 || len(address) > 256 || len(chain) > 32 {
		http.Error(w, "identity field exceeds maximum length", http.StatusBadRequest)
		return
	}
	l.mutex.Lock()
	defer l.mutex.Unlock()
	info, ok := l.linkedWallets[primary]
	if !ok {
		http.Error(w, "identity link not found", http.StatusNotFound)
		return
	}
	filtered := info.Linked[:0]
	removed := false
	for _, linked := range info.Linked {
		if normalizeIdentityPart(linked.Address) == address && normalizeIdentityPart(linked.Chain) == chain {
			removed = true
			continue
		}
		filtered = append(filtered, linked)
	}
	if !removed {
		http.Error(w, "identity link not found", http.StatusNotFound)
		return
	}
	info.Linked = filtered
	l.linkedWallets[primary] = info
	l.saveBlockchainStateSnapshotLocked(NotePrefixLinkSnapshot, l.linkedWallets)
	writeIdentityJSON(w, http.StatusOK, info)
}

func (b *IdentityBridge) HandleResolve(l *Lobby, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	address := normalizeIdentityPart(r.URL.Query().Get("address"))
	chain := normalizeIdentityPart(r.URL.Query().Get("chain"))
	if address == "" || chain == "" {
		http.Error(w, "address and chain are required", http.StatusBadRequest)
		return
	}
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	for primary, info := range l.linkedWallets {
		for _, linked := range info.Linked {
			if normalizeIdentityPart(linked.Address) == address && normalizeIdentityPart(linked.Chain) == chain {
				writeIdentityJSON(w, http.StatusOK, map[string]interface{}{"primary_wallet": primary, "linked_wallet": linked})
				return
			}
		}
	}
	http.Error(w, "identity link not found", http.StatusNotFound)
}

func (b *IdentityBridge) HandleSnapshot(l *Lobby, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	primary := normalizeIdentityPart(r.URL.Query().Get("primary_wallet"))
	if primary == "" {
		http.Error(w, "primary_wallet is required", http.StatusBadRequest)
		return
	}
	l.mutex.RLock()
	info, ok := l.linkedWallets[primary]
	l.mutex.RUnlock()
	if !ok {
		http.Error(w, "identity not found", http.StatusNotFound)
		return
	}
	writeIdentityJSON(w, http.StatusOK, info)
}

func writeIdentityJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// ResolvePrimaryWallet returns the primary AVM wallet for a given linked address and chain.
// Returns the original address if no link is found.
func (b *IdentityBridge) ResolvePrimaryWallet(l *Lobby, address, chain string) string {
	address = normalizeIdentityPart(address)
	chain = normalizeIdentityPart(chain)
	if address == "" || chain == "" {
		return address
	}
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	for primary, info := range l.linkedWallets {
		for _, linked := range info.Linked {
			if normalizeIdentityPart(linked.Address) == address && normalizeIdentityPart(linked.Chain) == chain {
				return primary
			}
		}
	}
	return address
}
