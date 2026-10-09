//go:build !js && !wasm

package main

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/mnemonic"
	"github.com/algorand/go-algorand-sdk/v2/transaction"
	"github.com/algorand/go-algorand-sdk/v2/types"
)

// ManagedNode tracks the health and performance of a specific RPC endpoint.
type ManagedNode struct {
	URL             string
	Client          *algod.Client
	LastLatency     time.Duration
	LastBlockSeen   uint64
	IsBlacklisted   bool
	LastErrorTime   time.Time
	RateLimitCount  int           // HTTP 429 responses in current window
	RateLimitWindow time.Time     // When the rate limit counter resets
	MaxRateLimits   int           // Threshold before temporary blacklist (default: 5)
	BlockSyncLag    time.Duration // Block propagation lag threshold
	HTTPStatusHist  map[int]int   // HTTP status code frequency for diagnostics
}

// LoadBalancedLedgerClient manages a cluster of nodes to ensure high availability.
// PILLAR 4: Network Resiliency.
type LoadBalancedLedgerClient struct {
	Mu                sync.RWMutex
	Nodes             []*ManagedNode
	ProductionMode    bool          // Strict validation for production (stricter blacklisting, longer cooldowns)
	RateLimitCooldown time.Duration // Extended blacklist duration after rate limit exhaustion
	SuccessCount      int64         // Track successful RPC calls for load distribution weight
}

// NodeHealthReport provides a structured health report for all managed nodes.
type NodeHealthReport struct {
	NodeURL     string    `json:"node_url"`
	Healthy     bool      `json:"healthy"`
	LatencyMs   float64   `json:"latency_ms"`
	BlockNumber uint64    `json:"block_number"`
	Blacklisted bool      `json:"blacklisted"`
	RateLimited bool      `json:"rate_limited"`
	LastError   string    `json:"last_error,omitempty"`
	HTTPCodes   map[int]int `json:"http_codes,omitempty"`
}

// NewLoadBalancedClient initializes the cluster with a set of primary and secondary nodes.
func NewLoadBalancedClient(urls []string, token string) (*LoadBalancedLedgerClient, error) {
	lb := &LoadBalancedLedgerClient{
		Nodes:             make([]*ManagedNode, 0),
		ProductionMode:    false,
		RateLimitCooldown: 15 * time.Minute, // Production: longer cooldown to prevent cascade failures
	}

	for _, url := range urls {
		client, err := algod.MakeClient(url, token)
		if err != nil {
			continue
		}
		lb.Nodes = append(lb.Nodes, &ManagedNode{
			URL:             url,
			Client:          client,
			RateLimitWindow: time.Now(),
			MaxRateLimits:   5, // Temp blacklist after 5 rate limits in a window
			BlockSyncLag:    30 * time.Second, // Flag if block not updated in 30s
			HTTPStatusHist:  make(map[int]int),
		})
	}

	if len(lb.Nodes) == 0 {
		return nil, fmt.Errorf("node redundancy failure: no valid RPC endpoints provided")
	}

	return lb, nil
}

// SetProductionMode configures the load balancer for production-grade strictness.
func (lb *LoadBalancedLedgerClient) SetProductionMode(enabled bool) {
	lb.Mu.Lock()
	defer lb.Mu.Unlock()
	lb.ProductionMode = enabled
	if enabled {
		lb.RateLimitCooldown = 15 * time.Minute
		for _, node := range lb.Nodes {
			node.MaxRateLimits = 3 // Stricter threshold in production
			node.BlockSyncLag = 10 * time.Second // Tighter block sync tolerance
		}
	} else {
		lb.RateLimitCooldown = 5 * time.Minute
		for _, node := range lb.Nodes {
			node.MaxRateLimits = 5
			node.BlockSyncLag = 30 * time.Second
		}
	}
}

// GetHealthReport returns structured health data for all managed nodes.
func (lb *LoadBalancedLedgerClient) GetHealthReport() []NodeHealthReport {
	lb.Mu.RLock()
	defer lb.Mu.RUnlock()

	report := make([]NodeHealthReport, 0, len(lb.Nodes))
	for _, node := range lb.Nodes {
		rateLimited := node.RateLimitCount > 0 && time.Since(node.RateLimitWindow) < 2*time.Minute
		report = append(report, NodeHealthReport{
			NodeURL:     node.URL,
			Healthy:     !node.IsBlacklisted && time.Since(node.LastErrorTime) < time.Hour,
			LatencyMs:   float64(node.LastLatency.Microseconds()) / 1000.0,
			BlockNumber: node.LastBlockSeen,
			Blacklisted: node.IsBlacklisted,
			RateLimited: rateLimited,
			LastError:   formatLastError(node),
			HTTPCodes:   copyHTTPCodes(node.HTTPStatusHist),
		})
	}
	return report
}

// MultiChainRouter routes transactions to the appropriate chain based on availability and priority.
// PILLAR-A/B: Multi-chain transaction routing (Voi + Algorand + Ethereum).
type MultiChainRouter struct {
	Mu                   sync.RWMutex
	VoiClient            *LoadBalancedLedgerClient
	AlgorandMainnet      *LoadBalancedLedgerClient
	EthereumClient       *EthereumClient // PILLAR-A: ETH trust anchor for NFT settlement
	AlgorandMainnetAsset string          // ARC-200 asset ID on Algorand Mainnet
	AlgorandAppID        uint64          // ARC-200 app ID for transfer method
	VoiAssetID           uint64          // ARC-200 asset ID on Voi
	VaultAddress         string
	VaultMnemonic        string
	AlgorandVaultAddress string
	AlgorandVaultMnemonic string
}

// NewMultiChainRouter creates a multi-chain routing layer with fallback chains.
func NewMultiChainRouter(voiClient *LoadBalancedLedgerClient, algorandMainnet *LoadBalancedLedgerClient, ethClient *EthereumClient) *MultiChainRouter {
	return &MultiChainRouter{
		VoiClient:       voiClient,
		AlgorandMainnet: algorandMainnet,
		EthereumClient:  ethClient,
	}
}

// TransferToChain routes a PAYOUT, and $VBV on Voi is the ONLY value payout (plan §1).
//
// WHY THIS REFUSES (operator adjudication 2026-09-19): the previous shape named Algorand
// as the PREFERRED rail and FELL BACK to it whenever Voi was unhealthy — two outbound
// rails, one of them a plug-in chain. Every chain other than Voi is a plug-in and every
// token other than $VBV is INWARD-ONLY, so a non-Voi outbound request is REFUSED with its
// reason instead of performed. A hint may only CONFIRM Voi; it can never redirect a
// payout. The Algorand client remains for INBOUND verification.
func (m *MultiChainRouter) TransferToChain(ctx context.Context, toWallet string, amount uint64, chainHint string) error {
	if amount == 0 {
		return nil
	}

	if chainHint != "" && chainHint != "voi" && chainHint != "VOI" {
		return fmt.Errorf("multi-chain: OUTBOUND REFUSED — rail %q is not $VBV on Voi; there is no other outbound rail (plan §1)", chainHint)
	}

	m.Mu.RLock()
	chain := m.selectChain()
	m.Mu.RUnlock()

	if chain != "voi" {
		return errors.New("multi-chain: OUTBOUND REFUSED — no healthy Voi node is available; $VBV on Voi is the only value payout and a plug-in chain cannot be used as a fallback rail (plan §1)")
	}
	return m.transferToVoi(ctx, toWallet, amount)
}

// selectChain answers with the ONE rail that may pay out: Voi.
//
// It takes NO hint by design (operator adjudication 2026-09-19, plan §1): the previous
// shape PREFERRED Algorand when hinted and FELL BACK to it when Voi was unhealthy, which
// is two outbound rails where the model allows one. The caller validates the hint and
// refuses when this answers "".
func (m *MultiChainRouter) selectChain() string {
	if m.VoiClient != nil && len(m.VoiClient.Nodes) > 0 {
		return "voi"
	}
	return ""
}

// transferToAlgorandMainnet REFUSES — the outbound rail is CLOSED.
//
// OPERATOR ADJUDICATION (2026-09-19, plan §1, binding): "$VBV on Voi is the ONLY value
// payout; there is no other outbound rail." This function used to BUILD AND BROADCAST an
// Algorand payout — a plug-in chain paying out — and it was REACHABLE through
// TransferToChain → selectChain, which both PREFERRED Algorand when hinted and FELL BACK
// to it when Voi was unhealthy.
//
// It is kept rather than deleted so a future caller receives a STATED REFUSAL instead of
// a compile error that invites someone to re-implement the rail. The Algorand CLIENT stays
// configured for INBOUND verification only: money ARRIVING on Algorand is allowed and
// expected (plug-in chains are inbound-only), and native VOI may leave the vault ONLY as
// the onboarding gas stipend (onboarding_service.go, note VBT_ONBOARD:GAS).
func (m *MultiChainRouter) transferToAlgorandMainnet(ctx context.Context, toWallet string, amount uint64) error {
	return errors.New("multi-chain: OUTBOUND REFUSED — Algorand is a plug-in chain and INBOUND-ONLY; $VBV on Voi is the only value payout and there is no other outbound rail (plan §1). Native VOI may leave the vault only as the onboarding gas stipend")
}
// transferToVoi executes a transfer on Voi Mainnet through the load balancer.
func (m *MultiChainRouter) transferToVoi(ctx context.Context, toWallet string, amount uint64) error {
	if m.VoiClient == nil {
		return errors.New("multi-chain: Voi client not configured")
	}

	voiMnemonic := os.Getenv("FAUCET_MNEMONIC")
	if voiMnemonic == "" {
		return errors.New("multi-chain: FAUCET_MNEMONIC not configured")
	}

	pk, err := mnemonic.ToPrivateKey(voiMnemonic)
	if err != nil {
		return fmt.Errorf("multi-chain: invalid Voi mnemonic: %w", err)
	}
	vaultAccount, _ := crypto.AccountFromPrivateKey(pk)

	bestNode := m.VoiClient.GetBestNode(ctx)
	if bestNode == nil {
		return errors.New("multi-chain: no healthy Voi node available")
	}

	sp, err := bestNode.Client.SuggestedParams().Do(ctx)
	if err != nil {
		return fmt.Errorf("multi-chain: Voi params fetch failed: %w", err)
	}

	recipientAddr, err := types.DecodeAddress(toWallet)
	if err != nil {
		return fmt.Errorf("multi-chain: invalid recipient address: %w", err)
	}

	methodSelector := []byte{0x2b, 0x42, 0x6d, 0xec}
	amountBytes := make([]byte, 32)
	new(big.Int).SetUint64(amount).FillBytes(amountBytes)

	appArgs := [][]byte{
		methodSelector,
		recipientAddr[:],
		amountBytes,
	}

	txn, err := transaction.MakeApplicationNoOpTx(m.VoiAssetID, appArgs, nil, nil, nil, sp, vaultAccount.Address, []byte(NotePrefixVoiDividend), types.Digest{}, [32]byte{}, types.Address{})
	if err != nil {
		return fmt.Errorf("multi-chain: Voi txn construction failed: %w", err)
	}

	txid, stxn, err := crypto.SignTransaction(vaultAccount.PrivateKey, txn)
	if err != nil {
		return fmt.Errorf("multi-chain: Voi signing failed: %w", err)
	}

	if _, err := bestNode.Client.SendRawTransaction(stxn).Do(ctx); err != nil {
		return fmt.Errorf("multi-chain: Voi dispatch failed: %w", err)
	}

	_, err = transaction.WaitForConfirmation(bestNode.Client, txid, 4, ctx)
	if err != nil {
		return fmt.Errorf("multi-chain: Voi confirmation failed: %w", err)
	}

	fmt.Printf("[MultiChain] Transferred %d micro-tokens to %s via Voi Mainnet\n", amount, toWallet)
	return nil
}

// ReportHTTPStatus logs an HTTP status code from a node response for diagnostic tracking.
func (lb *LoadBalancedLedgerClient) ReportHTTPStatus(nodeURL string, statusCode int) {
	lb.Mu.Lock()
	defer lb.Mu.Unlock()

	for _, node := range lb.Nodes {
		if node.URL == nodeURL {
			node.HTTPStatusHist[statusCode]++
			if statusCode >= 500 {
				// Server errors (5xx) immediately blacklist with longer cooldown
				now := time.Now()
				node.IsBlacklisted = true
				node.LastErrorTime = now
				if lb.ProductionMode {
					lb.RateLimitCooldown = 15 * time.Minute
				}
			} else if statusCode == 429 {
				now := time.Now()
				if now.Sub(node.RateLimitWindow) > 2*time.Minute {
					node.RateLimitCount = 0
					node.RateLimitWindow = now
				}
				node.RateLimitCount++
				if node.RateLimitCount >= node.MaxRateLimits {
					node.IsBlacklisted = true
					node.LastErrorTime = now
				}
			}
			break
		}
	}
}

// RecordSuccess increments the success counter for load distribution weight.
func (lb *LoadBalancedLedgerClient) RecordSuccess() {
	lb.Mu.Lock()
	defer lb.Mu.Unlock()
	lb.SuccessCount++
}

// copyHTTPCodes returns a deep copy of the HTTP status histogram.
func copyHTTPCodes(src map[int]int) map[int]int {
	if src == nil {
		return nil
	}
	dst := make(map[int]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// formatLastError returns a sanitized error string or empty if none.
func formatLastError(node *ManagedNode) string {
	if node.LastErrorTime.IsZero() || time.Since(node.LastErrorTime) > time.Hour {
		return ""
	}
	return "last_failure_" + node.LastErrorTime.Format(time.RFC3339)
}

// GetBestClient selects the optimal node based on latency and health status.
func (lb *LoadBalancedLedgerClient) GetBestClient() (*algod.Client, string, error) {
	lb.Mu.RLock()
	defer lb.Mu.RUnlock()

	var bestNode *ManagedNode
	now := time.Now()

	for _, node := range lb.Nodes {
		// Circuit Breaker: Skip blacklisted nodes for the configured cooldown period
		if node.IsBlacklisted && now.Sub(node.LastErrorTime) < lb.RateLimitCooldown {
			continue
		}
		// Also check 5-minute minimum regardless of mode
		if node.IsBlacklisted && now.Sub(node.LastErrorTime) < 5*time.Minute {
			continue
		}

		// Least-Latency selection strategy with anti-sticky bias
		if bestNode == nil || node.LastLatency < bestNode.LastLatency {
			bestNode = node
		}
	}

	if bestNode == nil {
		// Degraded mode: return the first available node (all may be blacklisted but cooldown expired)
		return lb.Nodes[0].Client, lb.Nodes[0].URL, nil
	}

	return bestNode.Client, bestNode.URL, nil
}

// GetBestNode returns the healthiest available node (reconciled alias for indexer/path selection).
func (lb *LoadBalancedLedgerClient) GetBestNode(ctx context.Context) *ManagedNode {
	lb.Mu.RLock()
	defer lb.Mu.RUnlock()
	now := time.Now()
	best := lb.Nodes[0]
	for i := range lb.Nodes {
		n := lb.Nodes[i]
		// Circuit Breaker: skip blacklisted nodes only within the 5-minute cooldown (mirrors GetBestClient)
		if n.IsBlacklisted && now.Sub(n.LastErrorTime) < 5*time.Minute {
			continue
		}
		if best == nil || n.LastLatency < best.LastLatency {
			best = n
		}
	}
	return best
}

// UnmarkNode attempts to unblacklist a node after timeout.
func (lb *LoadBalancedLedgerClient) UnmarkNode(url string) {
	lb.Mu.Lock()
	defer lb.Mu.Unlock()
	for _, node := range lb.Nodes {
		if node.URL == url && node.IsBlacklisted {
			now := time.Now()
			if now.Sub(node.LastErrorTime) >= 5*time.Minute {
				node.IsBlacklisted = false
			}
			break
		}
	}
}

// MarkNodeFailure blacklists a node after an RPC error (e.g., 429 or 5xx).
func (lb *LoadBalancedLedgerClient) MarkNodeFailure(url string) {
	lb.Mu.Lock()
	defer lb.Mu.Unlock()
	for _, node := range lb.Nodes {
		if node.URL == url {
			node.IsBlacklisted = true
			node.LastErrorTime = time.Now()
			break
		}
	}
}

// IsSemantic404Error classifies an RPC/indexer error as an authoritative HTTP 404
// "not found" answer (e.g. the ARC-200 vault box does not exist on-chain yet, or an
// account has no on-chain state). The NODE responded correctly — the requested datum
// simply does not exist, which is a valid answer (zero balance), NOT a node fault.
// Callers must treat it as a healthy-node result and must NOT blacklist the node.
func IsSemantic404Error(err error) bool {
	msg := fmt.Sprintf("%v", err)
	return strings.HasPrefix(msg, "HTTP 404")
}

// RunHealthMonitor starts a background daemon to refresh node metrics.
func (lb *LoadBalancedLedgerClient) RunHealthMonitor(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				lb.performSyncCheck()
			}
		}
	}()
}

func (lb *LoadBalancedLedgerClient) performSyncCheck() {
	// PILLAR 4: High Availability.
	// Acquire an RLock to safely copy the node slice, allowing network I/O 
	// to proceed without holding the global write lock.
	lb.Mu.RLock()
	nodes := lb.Nodes
	lb.Mu.RUnlock()

	for _, node := range nodes {
		start := time.Now()
		// PILLAR 4: Resilience Hardening. Implement 5s timeout for health pings.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		status, err := node.Client.Status().Do(ctx)
		cancel()

		lb.Mu.Lock()
		if err != nil {
			node.LastLatency = time.Since(start) // Record failed attempt duration
			if !node.IsBlacklisted {
				node.IsBlacklisted = true
				node.LastErrorTime = time.Now()
			}
			lb.Mu.Unlock()
			continue
		}
		// Update metrics: Healthy node detected
		node.LastLatency = time.Since(start)
		node.LastBlockSeen = status.LastRound
		node.IsBlacklisted = false
		lb.Mu.Unlock()
	}
}
