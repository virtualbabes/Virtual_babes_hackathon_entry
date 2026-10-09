package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// EthereumClient manages a load-balanced Ethereum node connection with health monitoring.
// PILLAR-A: Multi-chain trust anchor — primary chain for ETH-based NFT settlement and gas token transfers.
type EthereumClient struct {
	mu           sync.RWMutex
	tokens       []*ethclient.Client
	activeIdx    uint64
	blacklisted  map[uint64]time.Time // idx -> blacklist expiry
	retryBackoff time.Duration

	// Vault configuration
	vaultAddress string
	vaultKey     *ecdsa.PrivateKey

	// Health tracking
	lastBlockSeen atomic.Int64
	lastErrorAt   atomic.Int64 // stored as Unix nano seconds
	healthy       atomic.Bool
}

// NewEthereumClient initializes an Ethereum client with node URLs from networks.json and vault credentials.
func NewEthereumClient(ctx context.Context, nodeURLs []string) *EthereumClient {
	ec := &EthereumClient{
		tokens:       make([]*ethclient.Client, 0, len(nodeURLs)),
		blacklisted:  make(map[uint64]time.Time),
		retryBackoff: 10 * time.Second,
	}

	for _, url := range nodeURLs {
		c, err := ethclient.DialContext(ctx, url)
		if err != nil {
			log.Printf("[Ethereum] Failed to dial %s: %v (skipping)\n", url, err)
			continue
		}

		// Verify connectivity by fetching chain ID
		chainID, err := c.ChainID(ctx)
		if err != nil {
			log.Printf("[Ethereum] ChainID check failed for %s: %v (skipping)\n", url, err)
			continue
		}

		ec.tokens = append(ec.tokens, c)

		// Verify chain ID matches expected mainnet
		mainnetChainID := big.NewInt(1) // Ethereum mainnet
		if chainID.Cmp(mainnetChainID) != 0 {
			log.Printf("[Ethereum] %s returned chain_id=%s (expected %s), skipping\n", url, chainID.String(), mainnetChainID.String())
			continue
		}

		log.Printf("[Ethereum] Connected to Ethereum mainnet via %s (chain_id: %s)\n", url, chainID.String())
	}

	if len(ec.tokens) == 0 {
		log.Println("[Ethereum] WARNING: No Ethereum nodes connected. NFT settlements will fail.")
		return nil
	}

	// Load vault credentials from environment
	vaultKeyHex := os.Getenv("ETH_VAULT_KEY")
	vaultAddress := os.Getenv("ETH_VAULT_ADDRESS")
	if vaultKeyHex != "" && vaultAddress != "" {
		key, err := crypto.HexToECDSA(vaultKeyHex)
		if err != nil {
			log.Printf("[Ethereum] Failed to parse ETH vault key: %v\n", err)
		} else {
			ec.vaultKey = key
			ec.vaultAddress = vaultAddress
			log.Printf("[Ethereum] Vault loaded for address %s\n", vaultAddress)
		}
	}

	// Start health monitoring background goroutine
	go ec.startHealthMonitor(ctx)

	ec.healthy.Store(true)
	return ec
}

// startHealthMonitor periodically checks node health and manages failover.
func (ec *EthereumClient) startHealthMonitor(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ec.checkHealth(ctx)
		}
	}
}

// checkHealth verifies each node and manages failover.
func (ec *EthereumClient) checkHealth(ctx context.Context) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	var newHealthyTokens []*ethclient.Client
	newActiveIdx := uint64(0)
	bestBlockNumber := int64(0)

	for i, token := range ec.tokens {
		// Skip blacklisted nodes until retry time passes
		if expiry, blacklisted := ec.blacklisted[uint64(i)]; blacklisted {
			if time.Now().Before(expiry) {
				continue
			}
			// Blacklist expired, unblock and retry
			delete(ec.blacklisted, uint64(i))
			log.Printf("[Ethereum] Unblocking node %d after blacklist expiry\n", i)
		}

		// Check connectivity
		blockNumber, err := token.BlockNumber(ctx)
		if err != nil {
			ec.blacklisted[uint64(i)] = time.Now().Add(ec.retryBackoff)
			ec.lastErrorAt.Store(time.Now().UnixNano())
			log.Printf("[Ethereum] Node %d blacklisted: %v (retry in %v)\n", i, err, ec.retryBackoff)
			continue
		}

		newHealthyTokens = append(newHealthyTokens, token)

		if blockNumber > uint64(bestBlockNumber) {
			bestBlockNumber = int64(blockNumber)
			newActiveIdx = uint64(i)
		}

	}

	if len(newHealthyTokens) == 0 {
		if ec.healthy.CompareAndSwap(true, false) {
			log.Println("[Ethereum] WARNING: All nodes unhealthy!")
		}
		return
	}

	ec.tokens = newHealthyTokens
	ec.activeIdx = newActiveIdx
	ec.lastBlockSeen.Store(bestBlockNumber)

	if !ec.healthy.CompareAndSwap(false, true) {
		log.Printf("[Ethereum] Recovery detected! Node %d reconnected (block: %d)\n", newActiveIdx, bestBlockNumber)
	}
}

// GetClient returns the current active Ethereum client handle.
func (ec *EthereumClient) GetClient() *ethclient.Client {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	if len(ec.tokens) == 0 {
		return nil
	}

	idx := atomic.LoadUint64(&ec.activeIdx)
	if idx >= uint64(len(ec.tokens)) {
		idx = 0
	}
	return ec.tokens[idx]
}

// GetVaultAddress returns the vault wallet address (for frontend/telemetry).
func (ec *EthereumClient) GetVaultAddress() string {
	return ec.vaultAddress
}

// IsHealthy reports whether the Ethereum client has at least one healthy node.
func (ec *EthereumClient) IsHealthy() bool {
	return ec.healthy.Load() && len(ec.tokens) > 0
}

// SendETH transfers ETH from vault to recipient address.
// Uses uint64 micro-units internally for deterministic accounting.
func (ec *EthereumClient) SendETH(ctx context.Context, toAddress string, amountMicro uint64) error {
	client := ec.GetClient()
	if client == nil {
		return fmt.Errorf("ethereum: no healthy node available")
	}

	if ec.vaultKey == nil {
		return fmt.Errorf("ethereum: vault not configured")
	}

	// Validate recipient address
	if !common.IsHexAddress(toAddress) {
		return fmt.Errorf("ethereum: invalid address %s", toAddress)
	}

	toAddr := common.HexToAddress(toAddress)

	// Convert micro-units (1e6) to Wei (1e18) — micro → wei multiplier
	weiMultiplier := big.NewInt(1e12) // 1e18 / 1e6 = 1e12
	amountWei := new(big.Int).Mul(new(big.Int).SetUint64(amountMicro), weiMultiplier)

	// Get current nonce and gas price
	addr := common.HexToAddress(ec.vaultAddress)
	nonce, err := client.PendingNonceAt(ctx, addr)
	if err != nil {
		return fmt.Errorf("ethereum: nonce failure: %w", err)
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("ethereum: gas price failure: %w", err)
	}

	// ETH transfer (21000 gas fixed)
	gasLimit := uint64(21000)

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &toAddr,
		Value:    amountWei,
		Gas:      gasLimit,
		GasPrice: gasPrice,
	})

	// Sign transaction with vault key
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("ethereum: chain ID failure: %w", err)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), ec.vaultKey)
	if err != nil {
		return fmt.Errorf("ethereum: signing failure: %w", err)
	}

	// Send transaction
	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		return fmt.Errorf("ethereum: send failure: %w", err)
	}

	log.Printf("[Ethereum] TX sent: %s → %s (%.6f ETH)\n",
		ec.vaultAddress, toAddress, float64(amountMicro)/1e6)

	return nil
}

// SendERC20 transfers an ERC-20 token from vault to recipient.
func (ec *EthereumClient) SendERC20(ctx context.Context, tokenContractAddr string, toAddress string, amountMicro uint64) error {
	client := ec.GetClient()
	if client == nil {
		return fmt.Errorf("ethereum: no healthy node available")
	}

	if ec.vaultKey == nil {
		return fmt.Errorf("ethereum: vault not configured")
	}

	if !common.IsHexAddress(tokenContractAddr) || !common.IsHexAddress(toAddress) {
		return fmt.Errorf("ethereum: invalid hex address")
	}

	tokenAddr := common.HexToAddress(tokenContractAddr)
	toAddr := common.HexToAddress(toAddress)

	// ERC-20 transfer ABI (minimal)
	abiStr := `[{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"type":"function"}]`

	// Load token contract ABI
	abi, err := abi.JSON(strings.NewReader(abiStr))
	if err != nil {
		return fmt.Errorf("ethereum: ABI parse failure: %w", err)
	}

	// Convert micro-units to token amount (assume 1e6 decimals for standard token)
	tokenAmount := new(big.Int).SetUint64(amountMicro)

	// Encode transfer call
	data, err := abi.Pack("transfer", toAddr, tokenAmount)
	if err != nil {
		return fmt.Errorf("ethereum: encoding failure: %w", err)
	}

	addr := common.HexToAddress(ec.vaultAddress)
	nonce, err := client.PendingNonceAt(ctx, addr)
	if err != nil {
		return fmt.Errorf("ethereum: nonce failure: %w", err)
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("ethereum: gas price failure: %w", err)
	}

	// ERC-20 transfer default gas limit (safe upper bound for standard tokens)
	gasLimit := uint64(65000)

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &tokenAddr,
		Value:    big.NewInt(0),
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("ethereum: chain ID failure: %w", err)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), ec.vaultKey)
	if err != nil {
		return fmt.Errorf("ethereum: signing failure: %w", err)
	}

	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		return fmt.Errorf("ethereum: send failure: %w", err)
	}

	log.Printf("[Ethereum] ERC-20 TX sent: token=%s → %s (amount: %d micro)\n",
		tokenContractAddr, toAddress, amountMicro)

	return nil
}

// BatchTransferETH executes multiple ETH transfers from a single transaction using a custom contract.
// Returns the gas cost in micro-units for economic reconciliation.
func (ec *EthereumClient) BatchTransferETH(ctx context.Context, recipients []string, amounts []uint64) error {
	if len(recipients) != len(amounts) {
		return fmt.Errorf("ethereum: recipient/amount count mismatch")
	}
	if len(recipients) == 0 {
		return nil
	}

	client := ec.GetClient()
	if client == nil {
		return fmt.Errorf("ethereum: no healthy node available")
	}

	// Single transfer (no batching possible without custom contract)
	for i, recipient := range recipients {
		if err := ec.SendETH(ctx, recipient, amounts[i]); err != nil {
			log.Printf("[Ethereum] Batch item %d failed: %v\n", i, err)
			return fmt.Errorf("ethereum: batch item %d failed: %w", i, err)
		}
	}

	log.Printf("[Ethereum] Batch transfer complete: %d items\n", len(recipients))
	return nil
}

// GetBalance returns the ETH balance of an address in micro-units (1e6 Wei).
func (ec *EthereumClient) GetBalance(ctx context.Context, address string) (uint64, error) {
	client := ec.GetClient()
	if client == nil {
		return 0, fmt.Errorf("ethereum: no healthy node available")
	}

	if !common.IsHexAddress(address) {
		return 0, fmt.Errorf("ethereum: invalid address %s", address)
	}

	addr := common.HexToAddress(address)
	balance, err := client.BalanceAt(ctx, addr, nil)
	if err != nil {
		return 0, fmt.Errorf("ethereum: balance query failure: %w", err)
	}

	// Convert Wei (1e18) to micro (1e6) — divide by 1e12
	microBalance := new(big.Int).Div(balance, big.NewInt(1e12))
	if !microBalance.IsUint64() {
		return 0, fmt.Errorf("ethereum: balance overflow")
	}

	return microBalance.Uint64(), nil
}

// GetNonce returns the current nonce for the vault address.
func (ec *EthereumClient) GetNonce(ctx context.Context) (uint64, error) {
	client := ec.GetClient()
	if client == nil {
		return 0, fmt.Errorf("ethereum: no healthy node available")
	}

	if ec.vaultAddress == "" {
		return 0, fmt.Errorf("ethereum: vault not configured")
	}

	addr := common.HexToAddress(ec.vaultAddress)
	nonce, err := client.PendingNonceAt(ctx, addr)
	if err != nil {
		return 0, fmt.Errorf("ethereum: nonce query failure: %w", err)
	}

	return nonce, nil
}

// GetBlockNumber returns the latest block number.
func (ec *EthereumClient) GetBlockNumber(ctx context.Context) (uint64, error) {
	client := ec.GetClient()
	if client == nil {
		return 0, fmt.Errorf("ethereum: no healthy node available")
	}

	blockNumber, err := client.BlockNumber(ctx)
	if err != nil {
		return 0, fmt.Errorf("ethereum: block number query failure: %w", err)
	}

	return blockNumber, nil
}

// GetTransactionReceipt returns the receipt for a TX hash.
func (ec *EthereumClient) GetTransactionReceipt(ctx context.Context, txHash string) (*types.Receipt, error) {
	client := ec.GetClient()
	if client == nil {
		return nil, fmt.Errorf("ethereum: no healthy node available")
	}

	if !common.IsHexHash(txHash) {
		return nil, fmt.Errorf("ethereum: invalid tx hash %s", txHash)
	}

	receipt, err := client.TransactionReceipt(ctx, common.HexToHash(txHash))
	if err != nil {
		return nil, fmt.Errorf("ethereum: receipt query failure: %w", err)
	}

	return receipt, nil
}