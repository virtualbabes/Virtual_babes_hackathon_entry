import { CONFIG } from './config.js';
import { socket, setNonceResolver } from './network.js';
import { showToast, setTransactionStatus } from './ui.js';
import { userAddress, walletProvider, signClient, linkedWallets } from './wallet.js';
import { getAssetSymbol, getNetworkConfig } from './utils.js';
import { fetchLeaderboard } from './leaderboard.js';

export let availableNetworks = {};
export let globalClubs = {};
export let adminFocusNetwork = "";
export let ignoredReporters = new Set(JSON.parse(localStorage.getItem("vbabes_ignored_reporters") || "[]"));

// Setters for external modules
export const setCachedAdminHeaders = (headers) => { cachedAdminHeaders = headers; };
export const setAvailableNetworks = (networks) => { availableNetworks = networks; };
export const setGlobalClubs = (clubs) => { globalClubs = clubs; };
export const setAdminFocusNetwork = (network) => { adminFocusNetwork = network; };
export const setIgnoredReporters = (reporters) => { ignoredReporters = reporters; };

let cachedAdminHeaders = null;

/**
 * getAdminHeaders constructs the authentication headers required for administrative APIs.
 * PILLAR 5: Admin Security. Strictly enforces WalletConnect for administrative signatures.
 */
export async function getAdminHeaders() {
    if (!userAddress) {
        showToast("❌ Admin access requires a connected wallet.", "error");
        return null;
    }

    if (walletProvider !== 'walletconnect') {
        showToast("🚨 <b>SECURITY POLICY:</b> Administrative actions are restricted to WalletConnect sessions only.", "critical", 10000);
        return null;
    }

    if (cachedAdminHeaders && cachedAdminHeaders['X-Admin-Wallet'] === userAddress) {
        return cachedAdminHeaders;
    }

    try {
        setTransactionStatus("Requesting administrative nonce...", "info");
        
        const nonce = await new Promise((resolve, reject) => {
            setNonceResolver(resolve);
            socket.send(JSON.stringify({ type: "nonce_request" }));
            setTimeout(() => reject(new Error("Nonce request timed out")), 10000);
        });

        setTransactionStatus("Signing administrative proof...", "info");

        const sessions = signClient.session.getAll();
        if (!sessions || sessions.length === 0) throw new Error("Active session not found.");
        const topic = sessions[0].topic;
        let signature = "";
        const msg = `Virtualbabes Arena Admin Auth:${nonce}`;

        if (userAddress.startsWith("0x")) {
            signature = await signClient.request({
                topic,
                chainId: CONFIG.ETH_CHAIN_ID || "eip155:1",
                request: { method: "personal_sign", params: [msg, userAddress] }
            });
        } else {
            const response = await signClient.request({
                topic,
                chainId: CONFIG.VOI_CHAIN_ID,
                request: { method: "algo_signMessage", params: { address: userAddress, message: msg } }
            });
            signature = response.signature;
        }

        cachedAdminHeaders = { "X-Admin-Wallet": userAddress, "X-Admin-Nonce": nonce, "X-Admin-Signature": signature };
        return cachedAdminHeaders;
    } catch (err) {
        console.error("[ADMIN AUTH ERROR]", err);
        showToast(`❌ Authentication Failed: ${err.message}`, "error");
        return null;
    } finally {
        setTransactionStatus(null);
    }
}

// New admin functions for Season Rollover and Audit Export

/**
 * adminSeasonRollover triggers a manual season archival and leaderboard reset.
 * Requires admin authentication.
 */
export async function adminSeasonRollover() {
    if (!confirm("⚠️ CRITICAL: Are you sure you want to manually trigger a Season Rollover? This will archive current standings and reset the leaderboard.")) {
        return;
    }

    const headers = await getAdminHeaders();
    if (!headers) return;

    setTransactionStatus("Initiating season rollover...", "warning");

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/season-rollover`, {
            method: "POST",
            headers: {
                ...headers,
                'Content-Type': 'application/json'
            }
        });

        if (response.ok) {
            showToast("✅ Season rollover initiated. Check logs for archival status.", "success", 10000);
            // Optionally trigger a full lobby update to reflect new season number, etc.
            // window.syncUI("all"); // Assuming syncUI is globally available
        } else {
            const err = await response.text();
            showToast(`❌ Season rollover failed: ${err}`, "error");
        }
    } catch (err) {
        showToast("❌ Server connection error during season rollover.", "error");
    } finally {
        setTransactionStatus(null);
    }
}

/**
 * adminExportAuditLog triggers a download of the admin_audit.log as a CSV file.
 * Requires admin authentication.
 */
export async function adminExportAuditLog() {
    const headers = await getAdminHeaders();
    if (!headers) return;

    setTransactionStatus("Exporting audit logs...", "info");

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/export-logs`, {
            method: "GET",
            headers: headers
        });

        if (response.ok) {
            const blob = await response.blob();
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.style.display = 'none';
            a.href = url;
            a.download = 'admin_audit_export.csv';
            document.body.appendChild(a);
            a.click();
            window.URL.revokeObjectURL(url);
            a.remove();
            showToast("✅ Audit logs exported successfully.", "success");
        } else {
            const err = await response.text();
            showToast(`❌ Audit log export failed: ${err}`, "error");
        }
    } catch (err) {
        showToast("❌ Server connection error during audit log export.", "error");
    } finally {
        setTransactionStatus(null);
    }
}

/**
 * Renders the automation and reporting buttons in the Admin Panel UI.
 * Assumes a div with id="admin-automation-section" exists in the Admin Panel HTML.
 */
export function renderAdminAutomationButtons() {
    const container = document.getElementById("admin-automation-section");
    if (!container) {
        console.warn("Admin automation section (id='admin-automation-section') not found in DOM. Cannot render buttons.");
        return;
    }

    // Clear existing content to prevent duplicates on re-render
    container.innerHTML = `
        <h3 class="section-title">Automation & Reporting</h3>
        <div class="section-actions">
            <button class="danger" onclick="adminSeasonRollover()">SEASON ROLLOVER</button>
            <button class="outline" onclick="adminExportAuditLog()">EXPORT AUDIT LOGS</button>
        </div>
    `;
}

export function ignoreReporter(wallet) { // Exported for use in app.js
    if (!wallet) return;
    ignoredReporters.add(wallet);
    localStorage.setItem("vbabes_ignored_reporters", JSON.stringify(Array.from(ignoredReporters)));
    fetchAdminLogs(); // Re-render to apply filter
}

export async function fetchAdminLogs() { // Exported for use in app.js
    const headers = await getAdminHeaders();
    if (!headers) return;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/logs?filter=${document.getElementById("admin-log-filter")?.value || ""}`, {
            headers: headers
        });
        const data = await response.json();
        if (data.status === "success") {
            renderAdminLogs(data.logs);
        }
    } catch (err) { 
        console.error("Log fetch failed", err); 
    }
}

export async function adminRefillVault() { // Exported for use in app.js
    try {
        const amount = parseFloat(document.getElementById("admin-refill-amt").value);
        if (isNaN(amount)) return;
        const headers = await getAdminHeaders();
        // Implementation logic...
    } catch (err) { 
        showToast("❌ Refill failed", "error"); 
    }
}

export function updateAdminRewardList(rewards) { // Exported for use in app.js
    const container = document.getElementById("admin-reward-list");
    container.innerHTML = "";
    Object.entries(rewards || {}).forEach(([id, amt]) => {
    });
}

export async function adminAddReward() { // Exported for use in app.js
    const assetId = parseInt(document.getElementById("admin-add-asset").value);
    const amount = parseFloat(document.getElementById("admin-add-amt").value);
    if (!assetId || isNaN(amount)) return;
    } catch (err) { showToast("❌ Action failed", "error"); }
}

export async function adminRemoveReward(assetId) { // Exported for use in app.js
    const headers = await getAdminHeaders();
    if (!headers) return;

    } catch (err) { showToast("❌ Update failed", "error"); }
}

export async function adminAddNetwork() { // Exported for use in app.js
    const headers = await getAdminHeaders();
    if (!headers) return;

    } catch (err) { showToast("❌ Failed to add network", "error"); }
}

export async function adminBroadcast() { // Exported for use in app.js
    const text = document.getElementById("admin-msg-text").value;
    const priority = document.getElementById("admin-msg-priority")?.value || "info"; 
    if (!text) return;

    const headers = await getAdminHeaders();
    if (!headers) return;

    setTransactionStatus("Broadcasting system message...", "info");

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/system-message`, {
            method: "POST",
            headers: {
                ...headers,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ text, priority })
        });

        if (response.ok) {
            showToast("📢 Message broadcasted successfully.", "success");
            document.getElementById("admin-msg-text").value = "";
        } else {
            const err = await response.text();
            showToast(`❌ Broadcast failed: ${err}`, "error");
        }
    } catch (err) {
        showToast("❌ Broadcast failed", "error");
    } finally {
        setTransactionStatus(null);
    }
}

export async function adminUpdateRules() { // Exported for use in app.js
    const req = {
        Open: document.getElementById("rule-open").checked,
        Power_copy: document.getElementById("rule-same").checked,
    } catch (err) { showToast("❌ Rules update failed", "error"); }
}

export async function adminBanWallet(walletToBan = null, hoursToBan = null) { // Exported for use in app.js
    const wallet = walletToBan || document.getElementById("admin-ban-wallet").value.trim();
    const hours = hoursToBan || parseInt(document.getElementById("admin-ban-hours").value);
    if (!wallet) return;
    } catch (err) { showToast("❌ Server connection error", "error"); }
}

export async function adminAvatarBan(url = null, hours = null) { // Exported for use in app.js
    const targetUrl = url || document.getElementById("admin-ban-avatar-url").value.trim();
    if (!targetUrl) return;
    const headers = await getAdminHeaders();
    }
}

export function adminBanWalletFromLog(wallet) { // Exported for use in app.js
    // Default to 24 hours for a quick ban from logs
    adminBanWallet(wallet, 24);
}

export async function adminUpdatePowerScaling() { // Exported for use in app.js
    const divisor = parseFloat(document.getElementById("admin-power-divisor").value);
    const base = parseInt(document.getElementById("admin-power-base").value);
    const headers = await getAdminHeaders();
    } catch (err) { showToast("❌ Power update failed", "error"); }
}

export async function adminToggleMaintenance(active) { // Exported for use in app.js
    const minsInput = document.getElementById("admin-maint-mins");
    const minutes = parseInt(minsInput.value) || 0;
    const headers = await getAdminHeaders();
    } catch (err) { showToast("❌ Server connection error", "error"); }
}

export async function adminToggleDevMode() { // Exported for use in app.js
    const enabled = document.getElementById("dev-mode-toggle").checked;
    // Add a safety check when enabling
    if (enabled && !confirm("⚠️ DEV MODE: This will force a 100% win rate against the bot for reward testing. Enable?")) {
    showToast(`🛠️ Dev Mode ${enabled ? 'Enabled' : 'Disabled'}`, enabled ? "success" : "info");
}

export async function adminResetStats() { // Exported for use in app.js
    const wallet = document.getElementById("admin-ban-wallet").value.trim();
    if (!wallet) return;
    if (!confirm(`⚠️ CRITICAL: You are about to PERMANENTLY WIPE all stats for wallet: ${wallet}. This cannot be undone. Proceed?`)) return;
    } catch (err) { showToast("❌ Server connection error", "error"); }
}

export async function adminSimulateTournament() { // Exported for use in app.js
    const size = parseInt(document.getElementById("admin-sim-size").value);
    const isBuyIn = document.getElementById("admin-sim-buyin").checked;
    if (isNaN(size) || (size !== 8 && size !== 16)) {
    }
}

export let adminLogTicker = null; // Exported for use in app.js
export function startAdminLogPolling() { // Exported for use in app.js
    if (adminLogTicker) return;
    adminLogTicker = setInterval(fetchLastAdminAction, 15000); // Check every 15s for status bar
    renderAdminAutomationButtons(); // Render automation buttons when polling starts (admin panel is active)
}

export async function fetchLastAdminAction() { // Exported for use in app.js
    if (!lastAdminKey && !cachedAdminHeaders) {
        document.getElementById("admin-last-action").innerText = "Awaiting first action..."; 
        return; 
    } catch (err) {}
}

export function updateAdminNetworkUI() { // Exported for use in app.js
    const select = document.getElementById("admin-network-select");
    if (!select) return;

    onAdminNetworkSelectChange();
}

export function onAdminNetworkSelectChange() { // Exported for use in app.js
    const name = document.getElementById("admin-network-select").value;
    const config = availableNetworks[name];
    const details = document.getElementById("admin-network-details");
    }
}

export async function adminSetActiveNetwork() { // Exported for use in app.js
    const networkName = document.getElementById("admin-network-select").value;
    const headers = await getAdminHeaders();
    if (!headers) return;
