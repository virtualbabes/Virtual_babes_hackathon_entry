//go:build !js && !wasm

package main

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
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
func NewLoadBalancedClient(urls []string) (*LoadBalancedLedgerClient, error) {
	lb := &LoadBalancedLedgerClient{
		Nodes:             make([]*ManagedNode, 0),
		ProductionMode:    false,
		RateLimitCooldown: 15 * time.Minute, // Production: longer cooldown to prevent cascade failures
	}

	for _, url := range urls {
		client, err := algod.MakeClient(url, "")
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
			Healthy:     !node.IsBlacklisted && time.Since(node.LastErrorTime).Hour() < 1,
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
	AlgorandMainnet      *algod.Client
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
func NewMultiChainRouter(voiClient *LoadBalancedLedgerClient, algorandMainnet *algod.Client, ethClient *EthereumClient) *MultiChainRouter {
	return &MultiChainRouter{
		VoiClient:       voiClient,
		AlgorandMainnet: algorandMainnet,
		EthereumClient:  ethClient,
	}
}

// TransferToChain routes a token transfer to the appropriate chain.
// PILLAR-B: Chain selection priority is Algorand Mainnet > Voi Mainnet.
func (m *MultiChainRouter) TransferToChain(ctx context.Context, toWallet string, amount uint64, chainHint string) error {
	if amount == 0 {
		return nil
	}

	m.Mu.RLock()
	chain := m.selectChain(chainHint)
	m.Mu.RUnlock()

	switch chain {
	case "algorand":
		return m.transferToAlgorandMainnet(ctx, toWallet, amount)
	case "voi":
		fallthrough
	default:
		return m.transferToVoi(ctx, toWallet, amount)
	}
}

// selectChain determines the target chain based on hint and availability.
func (m *MultiChainRouter) selectChain(hint string) string {
	switch hint {
	case "algorand", "ALGO":
		if m.AlgorandMainnet != nil {
			return "algorand"
		}
		fallthrough
	default:
		if m.VoiClient != nil && len(m.VoiClient.Nodes) > 0 {
			return "voi"
		}
		if m.AlgorandMainnet != nil {
			return "algorand"
		}
		return ""
	}
}

// transferToAlgorandMainnet executes a transfer on Algorand Mainnet.
func (m *MultiChainRouter) transferToAlgorandMainnet(ctx context.Context, toWallet string, amount uint64) error {
	if m.AlgorandMainnet == nil {
		return errors.New("multi-chain: Algorand Mainnet client not configured")
	}

	algoMnemonic := os.Getenv("ALGORAND_MAINNET_VAULT_MNEMONIC")
	if algoMnemonic == "" {
		return errors.New("multi-chain: ALGORAND_MAINNET_VAULT_MNEMONIC not configured")
	}

	pk, err := mnemonic.ToPrivateKey(algoMnemonic)
	if err != nil {
		return fmt.Errorf("multi-chain: invalid Algorand Mainnet mnemonic: %w", err)
	}
	vaultAccount, _ := crypto.AccountFromPrivateKey(pk)

	sp, err := m.AlgorandMainnet.SuggestedParams().Do(ctx)
	if err != nil {
		return fmt.Errorf("multi-chain: failed to fetch Algorand Mainnet params: %w", err)
	}

	recipientAddr, err := types.DecodeAddress(toWallet)
	if err != nil {
		return fmt.Errorf("multi-chain: invalid recipient address: %w", err)
	}

	// ARC-200 Application Call for transfer
	methodSelector := []byte{0x2b, 0x42, 0x6d, 0xec}
	amountBytes := make([]byte, 32)
	new(big.Int).SetUint64(amount).FillBytes(amountBytes)

	appArgs := [][]byte{
		methodSelector,
		recipientAddr[:],
		amountBytes,
	}

	txn, err := transaction.MakeApplicationNoOpTx(m.AlgorandAppID, appArgs, nil, nil, nil, sp, vaultAccount.Address, []byte("VBET_ALGO_DIV"), types.Digest{}, [32]byte{}, types.Address{})
	if err != nil {
		return fmt.Errorf("multi-chain: Algorand txn construction failed: %w", err)
	}

	txid, stxn, err := crypto.SignTransaction(vaultAccount.PrivateKey, txn)
	if err != nil {
		return fmt.Errorf("multi-chain: Algorand signing failed: %w", err)
	}

	if _, err := m.AlgorandMainnet.SendRawTransaction(stxn).Do(ctx); err != nil {
		return fmt.Errorf("multi-chain: Algorand dispatch failed: %w", err)
	}

	_, err = transaction.WaitForConfirmation(m.AlgorandMainnet, txid, 4, ctx)
	if err != nil {
		return fmt.Errorf("multi-chain: Algorand confirmation failed: %w", err)
	}

	fmt.Printf("[MultiChain] Transferred %d micro-tokens to %s via Algorand Mainnet\n", amount, toWallet)
	return nil
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

	txn, err := transaction.MakeApplicationNoOpTx(m.VoiAssetID, appArgs, nil, nil, nil, sp, vaultAccount.Address, []byte("VBET_VOI_DIV"), types.Digest{}, [32]byte{}, types.Address{})
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
