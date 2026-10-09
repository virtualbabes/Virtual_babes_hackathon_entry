//go:build !js && !wasm

package main

import (
	"fmt"
	"log"
	"strings"
	"encoding/json"
)

// NautilusDEXPathService handles server-side market-buys of $VBV for creator payouts.
// This service simulates the interaction with a DEX to acquire $VBV from a system reserve
// (funded by console revenue) and distribute it to browser-based creators.
// PILLAR 2: Non-Custodial Economic Settlement.
type NautilusDEXPathService struct{}

// ExecuteMarketBuy simulates a DEX market buy of $VBV to pay a browser-based creator.
// This function assumes the $VOI equivalent of the console revenue (after siphon) is available
// in the system's faucet/reserve to perform the buy.
func (s *NautilusDEXPathService) ExecuteMarketBuy(l *Lobby, creatorWallet string, vbvAmountMicro uint64) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return s.ExecuteMarketBuyLocked(l, creatorWallet, vbvAmountMicro)
}

// ExecuteMarketBuyLocked is the internal implementation that assumes the Lobby mutex is held.
// PILLAR 5: Deadlock Prevention.
func (s *NautilusDEXPathService) ExecuteMarketBuyLocked(l *Lobby, creatorWallet string, vbvAmountMicro uint64) error {
	if vbvAmountMicro == 0 {
		return fmt.Errorf("cannot execute market buy for zero amount")
	}
	if creatorWallet == "" {
		return fmt.Errorf("creator wallet cannot be empty for payout")
	}

	// PILLAR 3: Identity Normalization.
	targetWallet := strings.ToLower(creatorWallet)

	// PILLAR 2: Integer Supremacy Safety Cap (Industrial Guardrail).
	if vbvAmountMicro > MaxSinglePayoutMicro {
		return fmt.Errorf("security exception: market buy amount %d exceeds single payout cap", vbvAmountMicro)
	}

	// PILLAR 2: Ledger Integrity & Double-Accounting Fix.
	// We do NOT deduct from faucetBalanceMicro here. The faucetBalanceMicro represents 
	// physical tokens in the vault; the deduction occurs only when tokens leave the 
	// system on-chain (faucet_service.go). 
	// We simply shift the liability from the unreserved faucet pool into the 
	// creator's virtual balance. This shift was previously funded by the 
	// console revenue inflow recorded in redemption_gateway.go.

	// 1. Credit Creator Wallet with $VBV (Virtual Liability Creation)
	l.ensurePlayerStatsMapsInitialized(targetWallet) // Ensure creator's playerStats are initialized
	l.playerBalances[targetWallet] += vbvAmountMicro

	// 2. Trigger Dynamic Scaling
	l.applyDynamicScalingLocked()

	// 3. Forensic Tracking: Log the liability shift to the authoritative audit trail.
	l.logAdminAuditLocked("NAUTILUS_DEX_PAYOUT", targetWallet, fmt.Sprintf("Creator payout: %d micro-VBV via simulated market buy. Context: CONSOLE_REDEEM", vbvAmountMicro))

	// 4. Notify Creator (if connected)
	creatorClientID := l.getClientIDFromWalletLocked(targetWallet)
	if creatorClientID != "" {
		l.sendToClientLocked(creatorClientID, Envelope{
			Type:    "admin_notification",
			Payload: json.RawMessage(fmt.Sprintf(`{"text":"💰 <b>CREATOR PAYOUT:</b> You received %.2f $VBV from console DLC sales!"}`, float64(vbvAmountMicro)/1000000.0)),
		})
	}

	log.Printf("[NAUTILUS_DEX] Market buy executed: %d micro-VBV paid to creator %s\n", vbvAmountMicro, targetWallet)
	return nil
}

// SimulateVoiToVbvSwap simulates a DEX market swap from native currency to $VBV.
// PILLAR 2: Integer Supremacy. Uses uint64 basis point math for slippage penalties.
func (s *NautilusDEXPathService) SimulateVoiToVbvSwap(voiAmountMicro uint64) (uint64, uint64, error) {
	if voiAmountMicro == 0 {
		return 0, 0, fmt.Errorf("cannot simulate swap for zero amount")
	}

	// PILLAR 2: Slippage Model. 
	// Penalty: 1 Basis Point (0.01%) for every 1000 VBV equivalent (1,000,000,000 micro).
	// Calculation: (amount / 1e9) = BPS. 
	penaltyBps := voiAmountMicro / 1000000000
	
	// Safety Clamping: Max 50% (5000 BPS) slippage to prevent arithmetic overflow/underflow.
	if penaltyBps > 5000 {
		penaltyBps = 5000
	}

	penaltyMicro := (voiAmountMicro * penaltyBps) / 10000
	vbvAmountMicro := voiAmountMicro - penaltyMicro

	return vbvAmountMicro, penaltyBps, nil
}