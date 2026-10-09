//go:build !js && !wasm

package main

import (
	"sync"
	"sync/atomic"
	"testing"
)

// MockEntityMarketNode initializes a test-specific state node with predefined ratios.
func MockEntityMarketNode() *EntityMarketNode {
	return &EntityMarketNode{
		EntityID:          "888",
		TotalSharesIssued: 10000,           // Active baseline supply in units
		ReserveBalance:    50000 * 1000000, // Backed by a 50,000 VBV liquidity reserve (micro-units)
		ReserveRatio:      0.33,            // Standard quadratic fractional reserve coefficient
	}
}

// TestCalculateBuyCost_WhaleSlippagePenalty asserts that large individual blocks
// trigger a significantly higher per-share price compared to smaller retail orders.
func TestCalculateBuyCost_WhaleSlippagePenalty(t *testing.T) {
	node := MockEntityMarketNode()

	// 1. Evaluate a small retail transaction (10 units)
	retailUnits := uint64(10)
	retailCost, retailSlippage := node.CalculateBuyCost(retailUnits)

	// 2. Evaluate a massive whale transaction (2,000 units at once - 20% of supply)
	whaleUnits := uint64(2000)
	whaleCost, whaleSlippage := node.CalculateBuyCost(whaleUnits)

	// 3. Compute effective per-unit price benchmarks
	effectiveRetailPricePerUnit := float64(retailCost) / float64(retailUnits)
	effectiveWhalePricePerUnit := float64(whaleCost) / float64(whaleUnits)

	t.Logf("[Test Log] Retail Cost: %d micro-VBV (Slippage: %.2f%%) | Per Unit: %.2f", retailCost, retailSlippage, effectiveRetailPricePerUnit)
	t.Logf("[Test Log] Whale Cost: %d micro-VBV (Slippage: %.2f%%) | Per Unit: %.2f", whaleCost, whaleSlippage, effectiveWhalePricePerUnit)

	// Assertions: The whale's per-unit price must be heavily penalized by the quadratic multiplier
	if effectiveWhalePricePerUnit <= effectiveRetailPricePerUnit {
		t.Errorf("Security invariant failure: Whale price per unit (%.2f) must be higher than retail price (%.2f) due to anti-whale slippage factors.", effectiveWhalePricePerUnit, effectiveRetailPricePerUnit)
	}

	if whaleSlippage < 50.0 {
		t.Errorf("Slippage guardrail anomaly: Expected high-percentage block buy to trigger massive slippage, got %.2f%%", whaleSlippage)
	}
}

// TestCalculateDynamicRumorFee_ScalesWithMarketCap verifies that market manipulation
// becomes increasingly expensive as an asset collects capital density.
func TestCalculateDynamicRumorFee_ScalesWithMarketCap(t *testing.T) {
	node := MockEntityMarketNode()

	// Capture baseline rumor fee floor at current 10k supply
	baselineFee := node.CalculateDynamicRumorFee()

	// Manually swell market metrics to simulate a heavily capitalized asset
	node.Mu.Lock()
	node.TotalSharesIssued = 500000
	node.ReserveBalance = 2500000 * 1000000
	node.Mu.Unlock()

	highCapFee := node.CalculateDynamicRumorFee()

	t.Logf("[Test Log] Baseline Rumor Fee: %d micro-VBV | High-Cap Rumor Fee: %d micro-VBV", baselineFee, highCapFee)

	if highCapFee <= baselineFee {
		t.Errorf("Economic flaw: Rumor fee must scale up alongside asset capitalization to prevent whale manipulation dominance.")
	}

	// Ensure the built-in soft cap safety guard holds true (Max 50,000 VBV limit)
	maxFeeMicro := uint64(50000 * 1000000)
	if highCapFee > maxFeeMicro {
		t.Errorf("Guardrail failure: Rumor pricing engine exceeded the system-defined ceiling. Got: %d", highCapFee)
	}
}

// TestHighVolumeConcurrentTrading Stress tests the system's coarse-grained RWMutex locking boundaries.
func TestHighVolumeConcurrentTrading(t *testing.T) {
	node := MockEntityMarketNode()
	var wg sync.WaitGroup

	concurrentWorkers := 200
	var successfulCalculations uint64 = 0

	for i := 0; i < concurrentWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Alternate between calculating cost and reading dynamic rumor fees
			if workerID%2 == 0 {
				_, slippage := node.CalculateBuyCost(15)
				if slippage >= 0 {
					atomic.AddUint64(&successfulCalculations, 1)
				}
			} else {
				fee := node.CalculateDynamicRumorFee()
				if fee >= (500 * 1000000) {
					atomic.AddUint64(&successfulCalculations, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	if successfulCalculations != uint64(concurrentWorkers) {
		t.Errorf("Concurrency failure: Only %d out of %d concurrent requests resolved safely.", successfulCalculations, concurrentWorkers)
	} else {
		t.Logf("[Test Log] Concurrency Stress Passed: Safely processed %d parallel calculations with zero lock conflicts.", successfulCalculations)
	}
}
