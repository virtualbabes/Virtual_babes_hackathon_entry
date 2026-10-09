//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"
)

// ── Entity Markets (Shares + Investment) ─────────────────────────────────────
// Markets for future potential. Players invest in players, businesses,
// creators, guilds, projects, communities.

type ShareToken struct {
	ID            string `json:"id"`
	IssuerType    string `json:"issuer_type"` // player, business, creator, guild, project, community
	IssuerID      string `json:"issuer_id"`
	Name          string `json:"name"`
	Symbol        string `json:"symbol"`
	TotalSupply   uint64 `json:"total_supply"`
	PriceMicro    uint64 `json:"price_micro"`
	DividendRate  uint64 `json:"dividend_rate"` // micro per share per cycle
	IssuedAt      time.Time `json:"issued_at"`
}

type ShareHolding struct {
	Holder  string `json:"holder"`
	TokenID string `json:"token_id"`
	Shares  uint64 `json:"shares"`
}

type EntityShares struct {
	mu       sync.RWMutex
	tokens   map[string]*ShareToken
	holdings map[string][]*ShareHolding // holder -> holdings
}

var entityShares = &EntityShares{
	tokens:   make(map[string]*ShareToken),
	holdings: make(map[string][]*ShareHolding),
}

func (m *EntityShares) IssueToken(issuerType, issuerID, name, symbol string, supply, price, dividend uint64) (*ShareToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	token := &ShareToken{
		ID:           fmt.Sprintf("share_%s_%d", issuerID, len(m.tokens)),
		IssuerType:   issuerType,
		IssuerID:     issuerID,
		Name:         name,
		Symbol:       symbol,
		TotalSupply:  supply,
		PriceMicro:   price,
		DividendRate: dividend,
		IssuedAt:     time.Now(),
	}
	m.tokens[token.ID] = token
	return token, nil
}

func (m *EntityShares) BuyShares(holder, tokenID string, amount uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.tokens[tokenID]
	if !ok {
		return fmt.Errorf("token not found")
	}

	// Transfer VBV from holder to issuer
	// (simplified: deduct from faucet for demo)

	// Add holding
	holdings := m.holdings[holder]
	found := false
	for i, h := range holdings {
		if h.TokenID == tokenID {
			holdings[i].Shares += amount
			found = true
			break
		}
	}
	if !found {
		holdings = append(holdings, &ShareHolding{Holder: holder, TokenID: tokenID, Shares: amount})
	}
	m.holdings[holder] = holdings
	return nil
}

func (m *EntityShares) GetTokens() []*ShareToken {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var all []*ShareToken
	for _, t := range m.tokens {
		all = append(all, t)
	}
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].IssuedAt.After(all[j].IssuedAt)
	})
	return all
}

func (m *EntityShares) GetHoldings(holder string) []*ShareHolding {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.holdings[holder]
}

func (l *Lobby) handleEntitySharesIssue(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IssuerType string `json:"issuer_type"`
		IssuerID   string `json:"issuer_id"`
		Name       string `json:"name"`
		Symbol     string `json:"symbol"`
		Supply     uint64 `json:"supply"`
		Price      uint64 `json:"price"`
		Dividend   uint64 `json:"dividend"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	token, err := entityShares.IssueToken(req.IssuerType, req.IssuerID, req.Name, req.Symbol, req.Supply, req.Price, req.Dividend)
	if err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true, "token": token})
}

func (l *Lobby) handleEntitySharesBuy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TokenID string `json:"token_id"`
		Amount  uint64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	if err := entityShares.BuyShares(wallet, req.TokenID, req.Amount); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true})
}

func (l *Lobby) handleEntitySharesTokens(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{"success": true, "tokens": entityShares.GetTokens()})
}

func (l *Lobby) handleEntitySharesHoldings(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	writeJSON(w, map[string]interface{}{"success": true, "holdings": entityShares.GetHoldings(wallet)})
}
