
## 6. THE CROSS-PLATFORM COMBAT LOOP
To maintain economic parity when console players engage in cross-platform matches, the server executes a **Symmetrical Liability Shift**.

### 6.1 Victory Against Crypto-Active Players
1.  **Browser Player (Loser):** Their wager or DNF penalty is processed as a standard $VBV virtual liability deduction and routed to the **Global Faucet**.
2.  **Console Player (Winner):** The server increments their **Arena Vouchers** balance 1:1.
3.  **Industrial Result:** The physical $VBV remains in the Arena Vault, while the console player receives non-crypto spending power for DLC and specialized hardware.

### 6.2 The "Funnel" Redemption
*   Console players who wish to "Exit" their credits to the Voi Network must connect a wallet via the **Browser/WASM Gateway**.
*   The server verifies the **Console UID** linkage.
*   "Arena Vouchers" are converted 1:1 into **Claimable $VBV**, allowing the player to perform an on-chain "Dispatch" from the browser.

### 6.3 Compliance Invariant
*   At no point does the console hardware interact with the private key or the signature group. 
*   The console only sees an increment in its local "Arena Vouchers" display.
# 🎮 PLATFORM EXPANSION SPECIFICATION: CONSOLE HUB

### 📑 DOCUMENT STATUS: INACTIVE SPECIFICATION (NON-CRYPTO COMPLIANT)
This document stipulates the technical facts and architectural requirements for the future Console Deployment Phase. 
**Strict Compliance Note:** To satisfy platform policies, the console runtime is **strictly NON-crypto active**.
**Priority Note:** These features are strictly unimplemented for the current build.

---

## 1. ARCHITECTURAL BOUNDARY: ISOLATION PROTOCOL
To prevent state drifting and maintain the security of the browser-based WASM engine, the following isolation rules are enforced:

*   **Gateway Access**: Consoles must interface exclusively via the `api/v1/` REST/WebSocket array.
*   **Zero Blockchain Interaction**: The console client is prohibited from performing on-chain signing, gas estimation, or wallet connections. 
*   **Voucher Representation**: Consoles use "Arena Vouchers" (uint64) as a 1:1 local representation of potential $VBV value, isolated from the blockchain.
*   **Bridge Restriction**: Consoles are strictly prohibited from interacting with `bridge_service.go` or any `syscall/js` logic, which are reserved for DOM-based browser interactions.
*   **Authoritative Parity**: Both Browser and Console clients utilize the same Go core logic for economic routing (`economy_processing.go`) and combat validation (`battle_service.go`).

---

## 2. THE ENTITLEMENT PIPELINE (IAP-TO-VIRTUAL)
Console revenue is processed via platform-native payment gateways and translated into Arena Vouchers by the Authoritative Core.

### 2.1 Platform Entitlement Ingestion
1.  Players purchase "Voucher Packs" or "Season Passes" using the console store (Fiat/Platform Funds).
2.  **Authoritative Validation**: The console client sends an `api/v1/entitlement_receipt` to the Go Backend.
3.  The backend verifies the receipt with the manufacturer's API (e.g., Xbox Live or PSN).

## 3. DATA CONTRACTS: SYMMETRICAL INTERFACE
Any future console implementation must adhere to these Go schemas defined in `common_types.go`:

```go
type ConsolePlatform string
const (
	PlatformXbox       ConsolePlatform = "XBOX_SERIES_X"
	PlatformPlayStation ConsolePlatform = "PLAYSTATION_5"
	PlatformNintendo    ConsolePlatform = "NINTENDO_SWITCH_NEXT"
)

type ConsoleAssetReceipt struct {
	PlatformRef    ConsolePlatform `json:"platform_ref"`
	ConsoleUID     string          `json:"console_user_uid"`
	DlcProductCode string          `json:"dlc_product_code"`
	IsLeaseAction  bool            `json:"is_lease_action"`
	LeaseDuration  int64           `json:"lease_duration_blocks"`
}
```

---

## 4. OPERATIONAL DIRECTIVES
*   **Lease Timelocks**: Authoritative blockchain tickers managed by `loan_service.go` will broadcast invalidation packets to the Console API upon lease expiration, locking local DLC profiles automatically.
*   **State Reconstruction**: Console users' Arena Vouchers (Salaries/Heists) are stored in the same blockchain-native snapshots used by browser users, ensuring a single, unified economy.

## 5. POLICY COMPLIANCE FACT SHEET
*   **No Private Keys**: Console clients never handle seed phrases or mnemonics.
*   **No Transaction Signing**: All "payouts" on console remain as Virtual Credits until the player logs into the Browser/WASM client to execute an on-chain "Dispatch."
*   **Representation**: $VBV on console is displayed as "Arena Credits (VBV Equivalent)" to ensure platform guidelines regarding speculative asset terminology are respected.