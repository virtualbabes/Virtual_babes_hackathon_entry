//go:build !js && !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// RedemptionRequest defines the payload for a console voucher redemption.
type RedemptionRequest struct {
	ConsoleUID     string `json:"console_uid"`
	ArenaVoucherID string `json:"arena_voucher_id"`
	Platform       string `json:"platform"`
	TxID           string `json:"txid"`
}

// handleRedemptionGateway processes console voucher redemptions.
// Residing in the root directory allows it to function as a method of Lobby.
func (l *Lobby) handleRedemptionGateway(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RedemptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	log.Printf("[REDEMPTION_GATEWAY] Received redemption request from ConsoleUID: %s, Voucher: %s, Platform: %s\n",
		req.ConsoleUID, req.ArenaVoucherID, req.Platform)

	l.mutex.Lock()
	defer l.mutex.Unlock()

	err := l.executeRedemptionLocked(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Redemption processed."})
}

// executeRedemptionLocked encapsulates the core settlement sequence.
// PILLAR 2: Industrial Seal. Supports both HTTP and WebSocket entry points.
func (l *Lobby) executeRedemptionLocked(req RedemptionRequest) error {
	// 1. Find Primary AVM Wallet associated with the ConsoleUID
	var primaryAVMWallet string
	for avmWallet, linkInfo := range l.linkedWallets {
		for _, linked := range linkInfo.Linked {
			// PILLAR 3: Switchboard Security. 
			// Only process redemptions for console accounts that have been cryptographically verified.
			if strings.EqualFold(linked.Address, req.ConsoleUID) && strings.EqualFold(linked.Chain, req.Platform) && linked.Verified {
				primaryAVMWallet = avmWallet
				break
			}
		}
		if primaryAVMWallet != "" { break }
	}

	if primaryAVMWallet == "" {
		return fmt.Errorf("console account not linked to an AVM wallet")
	}

	playerStats, exists := l.leaderboard[primaryAVMWallet]
	if !exists {
		return fmt.Errorf("player profile not found")
	}

	// 2. Retrieve DLC Product Details
	dlcRegistryMutex.RLock()
	product, productExists := DLCRegistry[req.ArenaVoucherID]
	dlcRegistryMutex.RUnlock()

	if !productExists {
		return fmt.Errorf("invalid product identifier")
	}

	dlcValueMicro := product.CostMicro
	creatorWallet := product.CreatorWallet

	// PILLAR 2: Stock Enforcement.
	// If the creator is initialized, verify they have stock available in their inventory.
	// Ghost creators skip this check to allow redemptions to trigger 100% tax recycling.
	creatorStats, creatorExists := l.leaderboard[strings.ToLower(creatorWallet)]
	if creatorExists && (creatorStats.Inventory == nil || creatorStats.Inventory[req.ArenaVoucherID] <= 0) {
		return fmt.Errorf("DLC product out of stock in creator inventory")
	}

	// 3. Entitlement Validation
	if playerStats.ArenaVouchers < dlcValueMicro {
		return fmt.Errorf("insufficient Arena Vouchers")
	}

	// 4. Verify Product Entitlement via Manufacturer API
	// PILLAR 3: Switchboard Security.
	switch strings.ToUpper(req.Platform) {
	case "XBOX":
		log.Printf("[REDEMPTION_GATEWAY] Verifying entitlement for %s on Xbox Live API (Simulated Success).\n", req.ConsoleUID)
	case "PLAYSTATION":
		log.Printf("[REDEMPTION_GATEWAY] Verifying entitlement for %s on PlayStation Network API (Simulated Success).\n", req.ConsoleUID)
	case "NINTENDO":
		log.Printf("[REDEMPTION_GATEWAY] Verifying entitlement for %s on Nintendo eShop API (Simulated Success).\n", req.ConsoleUID)
	default:
		log.Printf("[REDEMPTION_GATEWAY] Standard platform verification for %s (Placeholder Success).\n", req.ConsoleUID)
	}

	// 6. Creator Payout & Ghost/Stagnation Tax Evaluation
	// PILLAR 2: Industrial Seal.
	isSelfRedemption := strings.EqualFold(primaryAVMWallet, creatorWallet)

	if creatorWallet != "" && l.nautilusDEXPathService != nil {
		// Evaluate Initialization & Stagnation Status.
		siphonPercent := uint64(10)
		taxAuditAction := ""

		// PILLAR 2: Platform Surcharge.
		// Self-redemptions incur a 10% platform fee to fund the Global Faucet independently of the base siphon.
		if isSelfRedemption {
			siphonPercent += 10
			taxAuditAction = "PLATFORM_SURCHARGE"
		}

		// PILLAR 4: Console Hub Exemption.
		// Console-native creators (identified by UIDs < 32 chars) are exempt from Ghost/Stagnation taxes.
		isConsoleNative := len(creatorWallet) < 32

		if !creatorExists && !isConsoleNative {
			// PILLAR 2: Ghost Tax. Wallets that never joined the game are taxed 100%.
			siphonPercent = 100
			taxAuditAction = "GHOST_TAX_ENFORCED"

			// PILLAR 5: Reactive Atmosphere. 
			// Broadcast 50% reclaim alert to chat to celebrate ecosystem recycling.
			reclaimAmtMicro := (dlcValueMicro * 50 / 100)
			chatMsg := fmt.Sprintf("👻 <b>GHOST RECLAIM:</b> %.2f $VBV (50%%) from uninitialized creator recycled to Global Faucet.", float64(reclaimAmtMicro)/1000000.0)
			payload, _ := json.Marshal(map[string]string{"text": chatMsg})
			go func() { l.broadcast <- jsonListEnvelope("chat", payload) }()
		} else if !isConsoleNative && !creatorStats.LastActivity.IsZero() && time.Since(creatorStats.LastActivity) > 30*24*time.Hour {
			// PILLAR 2: Inactive Stewardship Fee (Additive).
			siphonPercent += 15 // Add 15% Stewardship Fee to existing base/surcharge
			if taxAuditAction == "" {
				taxAuditAction = "STAGNATION_TAX"
			} else {
				taxAuditAction += "_STAGNANT"
			}
		}

		taxAmountMicro := (dlcValueMicro * siphonPercent / 100)
		amountForCreatorSwap := dlcValueMicro - taxAmountMicro

		// PILLAR 5: Reactive Atmosphere.
		// Broadcast alert if stagnation fees exceed the significance threshold (500 $VBV).
		if (strings.Contains(taxAuditAction, "STAGNATION") || strings.Contains(taxAuditAction, "STAGNANT")) && taxAmountMicro > 500*1000000 {
			chatMsg := fmt.Sprintf("⚠️ <b>STAGNATION FEE:</b> %.2f $VBV captured from inactive creator and recycled to Global Faucet.", float64(taxAmountMicro)/1000000.0)
			payload, _ := json.Marshal(map[string]string{"text": chatMsg})
			go func() { l.broadcast <- jsonListEnvelope("chat", payload) }()
		}

		// PILLAR 2: Deterministic Settlement via Token-Sink Router.
		// Execute routing with the specific enforcement context to enable granular forensic reconciliation.
		routerContext := "CONSOLE_DLC"
		if taxAuditAction != "" {
			routerContext = taxAuditAction
		}

		if l.tokenSinkRouter != nil && dlcValueMicro > 0 {
			matrix := RevenueSplitMatrix{FaucetShare: 1.0, ClubShare: 0.0, GovernanceShare: 0.0}
			_ = l.tokenSinkRouter.RouteCriminalTax(routerContext, dlcValueMicro, matrix, 0, "")
		}

		// Only attempt market buy if the creator is entitled to a non-zero payout.
		if amountForCreatorSwap > 0 {
			vbvPayoutMicro, _, swapErr := l.nautilusDEXPathService.SimulateVoiToVbvSwap(amountForCreatorSwap)
			if swapErr == nil {
				_ = l.nautilusDEXPathService.ExecuteMarketBuyLocked(l, creatorWallet, vbvPayoutMicro)
				log.Printf("[REDEMPTION] Creator %s paid %d micro-VBV (Siphon: %d%%)\n", creatorWallet, vbvPayoutMicro, siphonPercent)
			}
		}

		if taxAuditAction != "" {
			// PILLAR 2: Integer Supremacy. Track reclaimed revenue for administrative auditing.
			if strings.HasPrefix(taxAuditAction, "GHOST") {
				l.GhostTaxTotal += taxAmountMicro
			} else if strings.HasPrefix(taxAuditAction, "PLATFORM") {
				l.PlatformTaxTotal += taxAmountMicro
			} else if strings.Contains(taxAuditAction, "STAGNATION") || strings.Contains(taxAuditAction, "STAGNANT") {
				l.StagnationTaxTotal += taxAmountMicro
			}
			l.logAdminAuditLocked(taxAuditAction, creatorWallet, fmt.Sprintf("Applied %d%% tax to DLC redemption for %s", siphonPercent, req.ArenaVoucherID))
		}
	}

	// 7. Stock Settlement
	if creatorExists {
		creatorStats.Inventory[req.ArenaVoucherID]--
		l.leaderboard[strings.ToLower(creatorWallet)] = creatorStats
	}

	// Decrement ArenaVouchers
	playerStats.ArenaVouchers -= dlcValueMicro
	l.leaderboard[primaryAVMWallet] = playerStats

	l.applyDynamicScalingLocked()
	msg := l.getLobbyUpdateMsgLocked()
	go func() { l.broadcast <- msg }()

	auditAction := "DLC_REDEMPTION"
	if isSelfRedemption {
		auditAction = "DLC_SELF_REDEMPTION"
	}
	l.logAdminAuditLocked(auditAction, primaryAVMWallet, fmt.Sprintf("Redeemed Voucher: %s, Value: %d micro-VBV", req.ArenaVoucherID, dlcValueMicro))
	return nil
}