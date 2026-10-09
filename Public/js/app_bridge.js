// app_bridge.js — Shell-level glue owned by app.js (single entry point).
//
// PURPOSE
//   All of this used to live as inline <script> blocks inside index.html. Per the
//   Single-Entry Mandate (see .clinerules/app-entry-mandate.md) index.html is a
//   PURE SHELL: it loads wasm_exec.js + third-party vendor bundles + app.js, and
//   nothing else. Everything first-party is imported by app.js.
//
// CONTRACTS PROVIDED (consumed by the WASM engine / other modules)
//   window.WasmWalletBridge      — wasm_exec wallet signature bridge
//   window.UnifiedAlertSystem    — global alert/toast surface (Roadmap 5.4)
//
// This file is imported by app.js BEFORE the WASM boot callback runs, so every
// global below is guaranteed to exist when main.wasm starts calling into JS.

// ============================================================================
// AUTHORITATIVE WALLET ACCESSOR
// ----------------------------------------------------------------------------
// `window.getActiveWallet()` is the ONE address accessor for the whole frontend.
// Before this existed, ~15 modules each read `window.currentWallet` /
// `window.userAddress` / `window.getActiveWallet` and none could rely on the
// last (it was defined nowhere), so wallet-scoped requests could silently carry
// an empty wallet.
//
// AUTHORITY ORDER (highest first):
//   1. GetGameState().wallet  — the WASM engine's own connected address, set by
//                               connectWallet(); authoritative, never guessed
//   2. window.currentWallet   — set by the wallet connector
//   3. window.userAddress     — set by the dev simulator
//   4. CONFIG.VAULT_ADDRESS   — configured vault fallback
// ============================================================================
window.getActiveWallet = function () {
    try {
        if (typeof window.GetGameState === 'function') {
            const st = window.GetGameState();
            if (st && st.wallet) return String(st.wallet);
        }
    } catch (_) { /* engine mid-boot — fall through to softer sources */ }
    return window.currentWallet || window.userAddress
        || (window.CONFIG && window.CONFIG.VAULT_ADDRESS) || '';
};

// ============================================================================
// WASM Wallet Bridge (Roadmap 5.3)
// ============================================================================
window.WasmWalletBridge = {
    // Requests a signature from an Algorand/Voi standard wallet provider
    signMarketAction: async (jsonPayloadString) => {
        try {
            const payload = JSON.parse(jsonPayloadString);

            // 1. Verify the provider is active in the user's browser context
            // We look for the standard window.voi provider (Kibisis/Lute)
            if (!window.voi || !window.voi.signBytes) {
                throw new Error("Compatible Voi Wallet Extension not detected.");
            }

            // 2. Convert the incoming JSON into a Uint8Array for signing
            const encoder = new TextEncoder();
            const dataBuffer = encoder.encode(JSON.stringify(payload));

            console.log(`[JS Bridge] Dispatching signature request for Nonce: ${payload.nonce}`);

            // 3. Request cryptographic signature via the extension
            const userAddress = payload.buyer_address;
            const signatureResult = await window.voi.signBytes(dataBuffer, userAddress);

            // Return the raw signature back to the awaiting WASM runtime context
            return {
                success: true,
                signature: btoa(String.fromCharCode(...signatureResult)), // Base64 packed
                error: ""
            };
        } catch (err) {
            console.error("[JS Bridge] Signing failed:", err.message);
            return {
                success: false,
                signature: "",
                error: err.message
            };
        }
    }
};

// ============================================================================
// Unified Alert System (Roadmap 5.4)
// ============================================================================
window.UnifiedAlertSystem = {
    showAlert: (title, message, visualType) => {
        // Check for an existing alert container to prevent layout stacking
        let alertBox = document.getElementById("game-alert-overlay");
        if (!alertBox) {
            alertBox = document.createElement("div");
            alertBox.id = "game-alert-overlay";
            document.body.appendChild(alertBox);
        }

        // Apply visual type styling (e.g., security warning or resource notice)
        const themeColor = visualType === "SECURITY" ? "#ff4d4d" : "#ffae42";

        alertBox.innerHTML = `
            <div style="position: fixed; top: 20px; right: 20px; background: #1a1a2e; color: #fff;
                        border-left: 5px solid ${themeColor}; padding: 16px; border-radius: 4px;
                        box-shadow: 0 4px 15px rgba(0,0,0,0.5); font-family: sans-serif; z-index: 9999;
                        max-width: 350px; transition: all 0.3s ease;">
                <b style="color: ${themeColor}; display: block; margin-bottom: 4px;">${title}</b>
                <span style="font-size: 13px; line-height: 1.4;">${message}</span>
            </div>
        `;

        // Automatically fade out the notification after 5 seconds
        setTimeout(() => {
            if (alertBox) alertBox.innerHTML = "";
        }, 5000);
    }
};

// ============================================================================
// Creator Storefront bootstrap (P7-C Task 7203)
// ============================================================================
// CreatorStorefront is a classic IIFE self-bound to window by
// creator_storefront.js. Initialise it once the DOM is parsed.
document.addEventListener('DOMContentLoaded', () => {
    if (window.CreatorStorefront) {
        window.CreatorStorefront.init('placeholder-wallet', null);
    }
});

// ============================================================================
// DEV-ONLY state simulation (?devsim=1 — no chain, reversible)
// ============================================================================
if (location.search.includes('devsim')) {
    const devsim = document.createElement('script');
    devsim.src = 'js/devsim.js';
    document.head.appendChild(devsim);
}

// ============================================================================
// FOUC guard — reveal the body once every dependency has settled.
// ============================================================================
window.addEventListener('load', () => {
    // Small delay to ensure all dynamic UI is rendered
    setTimeout(() => {
        document.body.classList.add('ready');
    }, 100);
});
