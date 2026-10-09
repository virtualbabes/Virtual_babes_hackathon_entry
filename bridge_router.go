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

// ── Multi-Chain Bridge System ────────────────────────────────────────────────
// Welcome every ecosystem. Ethereum, VOI, Polygon, Bitcoin, Solana.
// Assets retain their origin. Civilization unifies them.

type ChainType string

const (
	ChainVoi      ChainType = "voi"
	ChainAlgorand ChainType = "algorand"
	ChainEthereum ChainType = "ethereum"
	ChainPolygon  ChainType = "polygon"
	ChainBitcoin  ChainType = "bitcoin"
	ChainSolana   ChainType = "solana"
)

// ChainAsset represents an asset on its origin chain.
type ChainAsset struct {
	ID              string    `json:"id"`
	Chain           ChainType `json:"chain"`
	OriginChain     ChainType `json:"origin_chain"`
	AssetID         string    `json:"asset_id"`
	Symbol          string    `json:"symbol"`
	Name            string    `json:"name"`
	AmountMicro     uint64    `json:"amount_micro"`
	Owner           string    `json:"owner"`
	BridgedAt       time.Time `json:"bridged_at"`
	OriginTxHash    string    `json:"origin_tx_hash"`
	Status          string    `json:"status"` // pending, confirmed, failed
}

// BridgeTransaction tracks cross-chain transfers.
type BridgeTransaction struct {
	ID            string    `json:"id"`
	FromChain     ChainType `json:"from_chain"`
	ToChain       ChainType `json:"to_chain"`
	AssetID       string    `json:"asset_id"`
	AmountMicro   uint64    `json:"amount_micro"`
	Sender        string    `json:"sender"`
	Recipient     string    `json:"recipient"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	CompletedAt   time.Time `json:"completed_at"`
	FeeMicro      uint64    `json:"fee_micro"`
}

// BridgeRouter manages cross-chain assets.
type BridgeRouter struct {
	mu          sync.RWMutex
	assets      map[string]*ChainAsset
	txs         map[string]*BridgeTransaction
	chainFees   map[ChainType]uint64
}

var bridgeRouter = &BridgeRouter{
	assets:    make(map[string]*ChainAsset),
	txs:       make(map[string]*BridgeTransaction),
	chainFees: map[ChainType]uint64{
		ChainVoi:      1000,
		ChainAlgorand: 1000,
		ChainEthereum: 50000,
		ChainPolygon:  10000,
		ChainBitcoin:  100000,
		ChainSolana:   5000,
	},
}

// BridgeAsset bridges an asset from one chain to another.
func (mcr *BridgeRouter) BridgeAsset(fromChain, toChain ChainType, assetID string, amount uint64, sender, recipient string) (*BridgeTransaction, error) {
	mcr.mu.Lock()
	defer mcr.mu.Unlock()

	fee := mcr.chainFees[fromChain]
	tx := &BridgeTransaction{
		ID:          fmt.Sprintf("bridge_%d", time.Now().UnixNano()),
		FromChain:   fromChain,
		ToChain:     toChain,
		AssetID:     assetID,
		AmountMicro: amount,
		Sender:      sender,
		Recipient:   recipient,
		Status:      "pending",
		CreatedAt:   time.Now(),
		FeeMicro:    fee,
	}

	mcr.txs[tx.ID] = tx
	return tx, nil
}

// ConfirmBridge confirms a bridge transaction.
func (mcr *BridgeRouter) ConfirmBridge(txID string) error {
	mcr.mu.Lock()
	defer mcr.mu.Unlock()

	tx, ok := mcr.txs[txID]
	if !ok {
		return fmt.Errorf("transaction not found")
	}

	tx.Status = "confirmed"
	tx.CompletedAt = time.Now()

	// Create the bridged asset
	asset := &ChainAsset{
		ID:          fmt.Sprintf("bridged_%d", time.Now().UnixNano()),
		Chain:       tx.ToChain,
		OriginChain: tx.FromChain,
		AssetID:     tx.AssetID,
		AmountMicro: tx.AmountMicro,
		Owner:       tx.Recipient,
		BridgedAt:   time.Now(),
		Status:      "confirmed",
	}
	mcr.assets[asset.ID] = asset

	return nil
}

// GetAssets returns all assets for an owner.
func (mcr *BridgeRouter) GetAssets(owner string) []*ChainAsset {
	mcr.mu.RLock()
	defer mcr.mu.RUnlock()

	var results []*ChainAsset
	for _, a := range mcr.assets {
		if a.Owner == owner {
			results = append(results, a)
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].BridgedAt.After(results[j].BridgedAt)
	})
	return results
}

// GetTransactions returns all bridge transactions for a sender.
func (mcr *BridgeRouter) GetTransactions(sender string) []*BridgeTransaction {
	mcr.mu.RLock()
	defer mcr.mu.RUnlock()

	var results []*BridgeTransaction
	for _, t := range mcr.txs {
		if t.Sender == sender {
			results = append(results, t)
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})
	return results
}

// GetChainSummary returns a summary of all chains.
func (mcr *BridgeRouter) GetChainSummary() map[string]interface{} {
	mcr.mu.RLock()
	defer mcr.mu.RUnlock()

	chainCounts := make(map[string]int)
	chainValues := make(map[string]uint64)

	for _, a := range mcr.assets {
		chainCounts[string(a.Chain)]++
		chainValues[string(a.Chain)] += a.AmountMicro
	}

	return map[string]interface{}{
		"chain_counts":  chainCounts,
		"chain_values":  chainValues,
		"total_assets":  len(mcr.assets),
		"total_txs":     len(mcr.txs),
		"chain_fees":    mcr.chainFees,
	}
}

// ── HTTP Handlers ────────────────────────────────────────────────────────────

func (l *Lobby) handleBridgeAsset(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FromChain string `json:"from_chain"`
		ToChain   string `json:"to_chain"`
		AssetID   string `json:"asset_id"`
		Amount    uint64 `json:"amount"`
		Recipient string `json:"recipient"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	wallet := l.getWalletFromRequest(r)
	tx, err := bridgeRouter.BridgeAsset(ChainType(req.FromChain), ChainType(req.ToChain), req.AssetID, req.Amount, wallet, req.Recipient)
	if err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true, "tx": tx})
}

func (l *Lobby) handleBridgeConfirm(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TxID string `json:"tx_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": "invalid body"})
		return
	}
	if err := bridgeRouter.ConfirmBridge(req.TxID); err != nil {
		writeJSON(w, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"success": true})
}

func (l *Lobby) handleBridgeAssets(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	assets := bridgeRouter.GetAssets(wallet)
	writeJSON(w, map[string]interface{}{"success": true, "assets": assets})
}

func (l *Lobby) handleBridgeTxs(w http.ResponseWriter, r *http.Request) {
	wallet := l.getWalletFromRequest(r)
	txs := bridgeRouter.GetTransactions(wallet)
	writeJSON(w, map[string]interface{}{"success": true, "txs": txs})
}

func (l *Lobby) handleBridgeSummary(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{"success": true, "summary": bridgeRouter.GetChainSummary()})
}
