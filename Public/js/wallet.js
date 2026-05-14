// Public/js/wallet.js

import { CONFIG } from './config.js';
import { showToast, setTransactionStatus, hideAllOverlays, showMainGameContainer } from './ui.js';
import { getNetworkConfig } from './utils.js';
import { socket, setNonceResolver } from './network.js';
import { fetchUserNFTs } from './deck.js';

export let userAddress = null;
export let isVerified = false;
export let linkedWallets = JSON.parse(localStorage.getItem("vbabes_linked_wallets") || "[]");
export let payoutAddress = localStorage.getItem("vbabes_payout_address") || null;

export let walletProvider = null;      // Current active provider (nautilus, kibisis, etc.)
export let signClient = null; // WalletConnect State
export let wcModal = null;    // WalletConnect Modal State

export const setWalletProvider = (provider) => { walletProvider = provider; };
export const setPayoutAddress = (address) => { payoutAddress = address; };
export const setUserAddress = (address) => { userAddress = address; };
export const setIsVerified = (verified) => { isVerified = verified; };
export const setLinkedWallets = (wallets) => { linkedWallets = wallets; };

// --- WalletConnect Initialization ---
export async function initWalletConnect() {
    const projectId = (CONFIG.WC_PROJECT_ID || "").toString().trim();
    if (!projectId || projectId.toLowerCase().includes('your_walletconnect_project_id')) {
        console.warn("[WC] WalletConnect Project ID not configured.");
        showToast("WalletConnect is not configured. Set walletconnect-project-id in index.html.", "warning");
        return;
    }

    try {
        // The UMD build of sign-client exports globally as SignClient
        const SignClient = window.SignClient;
        if (!SignClient) return;

        const WalletConnectModal = window.WalletConnectModal;
        if (WalletConnectModal) {
            wcModal = new WalletConnectModal.WalletConnectModal({
                projectId: CONFIG.WC_PROJECT_ID,
                chains: [CONFIG.VOI_CHAIN_ID, CONFIG.ALGO_CHAIN_ID]
            });
        }

        signClient = await SignClient.init({
            projectId: CONFIG.WC_PROJECT_ID,
            metadata: {
                name: "Virtualbabes Arena",
                description: "The premier NFT Seduction battleground on Voi.",
                url: window.location.origin,
                icons: [(CONFIG.IS_LOCAL ? window.location.origin : "") + CONFIG.ASSET_URL + "Assets/logo.png"],
            },
        });

        // Handle session events
        signClient.on("session_event", ({ event }) => { console.log("[WC] Event:", event); });
        signClient.on("session_update", ({ topic, params }) => { console.log("[WC] Session Updated:", params); });
        signClient.on("session_delete", () => { 
            console.log("[WC] Session Deleted");
            disconnectUserWallet();
        });

        // Restore existing session
        const sessions = signClient.session.getAll();
        if (sessions.length > 0) {
            const session = sessions[0];
            const account = session.namespaces.algorand.accounts[0];
            const addr = account.split(":")[2];
            walletProvider = 'walletconnect';
            console.log("[WC] Session Restored:", addr);
            updateWalletUI(addr);
        }

        console.log("[WC] Initialization Complete.");
    } catch (err) {
        console.error("[WC] Initialization Failed:", err);
        showToast("WalletConnect failed to initialize.", "error");
    }
}

export async function checkVoiReadiness(address) {
    console.log("[BRIDGE] Checking Voi readiness for Algorand wallet...");
    setTransactionStatus("Checking Voi onboarding status...", "info");
    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/bridge/onboard`, {
            method: "POST",
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ wallet: address })
        });
        if (response.ok && response.status !== 204) {
            const data = await response.json();
            let message = `🌉 ${data.message}`;
            if (data.txid) {
                const netCfg = getNetworkConfig("Voi Mainnet");
                if (netCfg && netCfg.explorer_url) {
                    message += `<br><a href="${netCfg.explorer_url}/tx/${data.txid}" target="_blank" style="color: var(--neon-green); text-decoration: underline;">View Transaction</a>`;
                }
            }
            showToast(message, "success", 10000);
        }
    } catch (err) { console.warn("[BRIDGE] Onboarding check failed", err); }
    finally { setTransactionStatus(null); }
}

export function openPayoutSettings() {
    document.getElementById("payout-settings-overlay").classList.remove("hidden");
    document.getElementById("payout-address-input").value = payoutAddress || "";
}

export function savePayoutAddress() {
    const addr = document.getElementById("payout-address-input").value.trim();
    if (addr && addr.length === 58) {
        payoutAddress = addr;
        localStorage.setItem("vbabes_payout_address", addr);
        showToast("✅ Voi payout address updated", "success");
        updatePayoutUI();
        hideAllOverlays();
    } else {
        showToast("❌ Invalid Voi Address", "error");
    }
}

export function updatePayoutUI() {
    const display = document.getElementById("payout-address-display");
    if (display) {
        display.innerText = payoutAddress ? (payoutAddress.substring(0, 6) + "..." + payoutAddress.substring(54)) : "Default Wallet";
    }
}

export async function processRewardPayout(payloadStr) {
    const payload = JSON.parse(payloadStr);
    showToast("🛰️ Requesting secure nonce from server...", "info");
    try {
        const nonce = await new Promise((resolve, reject) => {
            const timeout = setTimeout(() => reject(new Error("Nonce timed out")), 10000);
            setNonceResolver((n) => { clearTimeout(timeout); resolve(n); });
            socket.send(JSON.stringify({ type: "nonce_request" }));
        });

        const tx = { from: userAddress, to: userAddress, amount: 0, note: new TextEncoder().encode(nonce), type: 'pay' };
        let signedTx = null;
        payload.claimant = payload.recipient; 
        payload.recipient = payoutAddress || payload.recipient; 

        if (walletProvider === 'nautilus') {
            const result = await window.algo.signTxn([{ txn: algosdk.encodeObj(tx), signers: [payload.claimant] }]);
            signedTx = result[0];
        } else if (walletProvider === 'walletconnect' && signClient) {
            const response = await signClient.request({
                topic: signClient.session.getAll()[0].topic,
                chainId: CONFIG.VOI_CHAIN_ID,
                request: { method: "algo_signTxn", params: [[{ txn: btoa(String.fromCharCode(...algosdk.encodeObj(tx))), signers: [payload.claimant] }]] }
            });
            signedTx = new Uint8Array(atob(response[0]).split("").map(c => c.charCodeAt(0)));
        }

        if (!signedTx) throw new Error("Signature failed.");
        payload.signed_tx = Array.from(signedTx);

        const response = await fetch(`${CONFIG.API_BASE}/api/reward`, {
            method: "POST",
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        if (!response.ok) throw new Error(await response.text());
        showToast("✅ Reward Sent!", "success");
    } catch (err) { showToast("⚠️ Payout Failed: " + err.message, "error"); }
}

export async function handleWalletAction() {
    if (window.userAddress) {
        await disconnectUserWallet();
    } else {
        await connectUserWallet();
    }
}

export function closeWalletSelector() {
    document.getElementById("wallet-selector-overlay").classList.add("hidden");
    showMainGameContainer(); // Show main game if no other overlay is active
}

export async function connectUserWallet() {
    document.getElementById("wallet-selector-overlay").classList.remove("hidden");
}

export async function connectWith(provider) {
    if (CONFIG.VAULT_ADDRESS === null) {
        showToast("⚠️ Arena configuration not yet synced. Please wait a moment.", "warning");
        return;
    }

    closeWalletSelector();
    showToast(`Connecting to ${provider}...`, "info");
    
    try {
        let address = null;
        if (provider === 'nautilus') {
            if (!window.algo) throw new Error("Nautilus not installed");
            const accounts = await window.algo.enable();
            address = accounts[0];
            walletProvider = 'nautilus';
        } else if (provider === 'kibisis') {
            if (!window.kibisis) throw new Error("Kibisis not installed");
            const accounts = await window.kibisis.enable();
            address = accounts[0];
            walletProvider = 'kibisis';
        } else if (provider === 'walletconnect') {
            if (!signClient || !wcModal) throw new Error("WalletConnect not initialized");

            const { uri, approval } = await signClient.connect({
                optionalNamespaces: {
                    algorand: {
                        methods: ["algo_signTxn", "algo_signMessage"],
                        chains: [CONFIG.VOI_CHAIN_ID, CONFIG.ALGO_CHAIN_ID],
                        events: ["chainChanged", "accountsChanged"],
                    },
                    eip155: {
                        methods: ["eth_signTransaction", "eth_sendTransaction", "personal_sign"],
                        chains: ["eip155:1", "eip155:137"], // ETH & Polygon
                        events: ["chainChanged", "accountsChanged"],
                    }
                },
            });
            if (uri) {
                wcModal.openModal({ uri });
                const session = await approval();
                wcModal.closeModal();
                
                const account = session.namespaces.algorand.accounts[0];
                address = account.split(":")[2];
                walletProvider = 'walletconnect';
            }
        }

        if (address) {
            const result = window.connectWallet(address);
            if (result.status === "success") {
                updateWalletUI(result.address);
                showToast("Wallet Connected!", "success");
                closeWalletSelector();
            }
        }
    } catch (err) {
        console.error("Connection failed", err);
        showToast(err.message, "error");
    }
}

export async function disconnectUserWallet() {
    console.log("[WALLET] Disconnecting...");
    try {
        if (walletProvider === 'walletconnect' && signClient) {
            const sessions = signClient.session.getAll();
            if (sessions.length > 0) {
                await signClient.disconnect({
                    topic: sessions[0].topic,
                    reason: { code: 6000, message: "User disconnected" }
                });
            }
        }
        walletProvider = null;
        
        window.disconnectWallet();
        setIsVerified(false);
        setUserAddress(null);
        updateWalletUI(null);
    } catch (err) {
        console.error("Disconnect failed", err);
    }