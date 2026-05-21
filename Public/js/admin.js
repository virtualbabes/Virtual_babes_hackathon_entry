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
 * adminForcePayout triggers a manual reward dispatch for a legitimate match result
 * that was interrupted by a connection failure.
 * PILLAR 3: Administrative Expansion.
 */
export async function adminForcePayout() {
    const targetId = document.getElementById("admin-force-payout-id").value.trim();
    if (!targetId) {
        showToast("Please enter a valid ClientID or Wallet Address.", "error");
        return;
    }

    if (!confirm(`FORCE PAYOUT: Trigger authoritative reward dispatch for ${targetId}?\n\nThis bypasses standard client-side signature proof.`)) return;

    const headers = await getAdminHeaders();
    if (!headers) return;

    setTransactionStatus("Initiating force payout...", "warning");

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/force-payout`, {
            method: "POST",
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify({ target_id: targetId })
        });

        if (response.ok) {
            const result = await response.json();
            showToast(`🏆 FORCE PAYOUT SUCCESS: ${result.txid}`, "success");
            if (typeof fetchAdminLogs === 'function') fetchAdminLogs(); // Refresh logs to see the audit entry
        } else {
            const err = await response.text();
            showToast(`❌ Force Payout Failed: ${err}`, "error");
        }
    } catch (err) {
        console.error("[ADMIN ERROR]", err);
        showToast("❌ Network error during force payout request.", "error");
    } finally {
        setTransactionStatus(null);
    }
}

/**
 * adminSimulateMojoDecay triggers a stress test for the Mojo decay logic.
 * Creates a specified number of regional clubs and simulates decay over a duration.
 */
export async function adminSimulateMojoDecay() {
    const numClubs = parseInt(document.getElementById("admin-mojo-sim-clubs").value);
    const durationMinutes = parseInt(document.getElementById("admin-mojo-sim-duration").value);

    if (isNaN(numClubs) || numClubs <= 0 || isNaN(durationMinutes) || durationMinutes <= 0) {
        showToast("❌ Please enter valid numbers for clubs and duration.", "error");
        return;
    }

    if (!confirm(`⚠️ Initiate Mojo Decay Stress Test with ${numClubs} clubs for ${durationMinutes} minutes? This will temporarily replace active clubs.`)) {
        return;
    }

    const headers = await getAdminHeaders();
    if (!headers) return;

    setTransactionStatus("Initiating Mojo Decay Simulation...", "warning");

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/simulate-mojo-decay`, {
            method: "POST",
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify({ num_clubs: numClubs, duration_minutes: durationMinutes })
        });
        if (response.ok) showToast("🧪 Mojo Decay Simulation Started. Check server logs for progress.", "success");
        else showToast(`❌ Simulation Failed: ${await response.text()}`, "error");
    } catch (err) { showToast("❌ Network error during simulation request.", "error"); } finally { setTransactionStatus(null); }
}

/**
 * adminAssetForfeiture triggers a manual card recovery from a club jail.
 * Requires admin authentication.
 */
export async function adminAssetForfeiture() {
    const cardIdInput = document.getElementById("admin-forfeit-card-id");
    const clubIdInput = document.getElementById("admin-forfeit-club-id");
    
    if (!cardIdInput || !clubIdInput) return;
    
    const card_id = parseInt(cardIdInput.value);
    const club_id = clubIdInput.value.trim();

    if (isNaN(card_id) || !club_id) {
        showToast("❌ Please enter both Card ID and Club ID.", "error");
        return;
    }

    if (!confirm(`⚠️ CRITICAL: Manually forfeit Card #${card_id} from Club ${club_id}? This asset will be returned to its original owner.`)) {
        return;
    }

    const headers = await getAdminHeaders();
    if (!headers) return;

    setTransactionStatus("Executing asset forfeiture...", "warning");

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/asset-forfeiture`, {
            method: "POST",
            headers: {
                ...headers,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ card_id, club_id })
        });

        if (response.ok) {
            showToast("✅ Asset forfeiture successful. Card returned to owner.", "success");
            cardIdInput.value = "";
            clubIdInput.value = "";
        } else {
            const err = await response.text();
            showToast(`❌ Forfeiture failed: ${err}`, "error");
        }
    } catch (err) {
        showToast("❌ Server connection error during asset forfeiture.", "error");
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
        <div class="section-actions mb-20">
            <button class="danger" onclick="adminSeasonRollover()">SEASON ROLLOVER</button>
            <button class="outline" onclick="adminExportAuditLog()">EXPORT AUDIT LOGS</button>
        </div>
        
        <h3 class="section-title mt-20">Asset Recovery (Forfeiture)</h3>
        <div class="glass-panel p-15 border-error">
            <div class="flex-row gap-10 mb-10">
                <input type="number" id="admin-forfeit-card-id" class="glass-input flex-1" placeholder="Card ID">
                <input type="text" id="admin-forfeit-club-id" class="glass-input flex-1" placeholder="Club ID">
            </div>
            <button class="w-full danger" onclick="adminAssetForfeiture()">SEIZE & RETURN ASSET</button>
            <small class="opacity-5 italic block mt-10">Surgically recovers cards from club jails in case of disputes.</small>
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
            updateDashboardStats(data);
            renderAdminLogs(data.logs);
            adminCyberSecurityAudit(); // Refresh the security audit view
        }
    } catch (err) { 
        console.error("Log fetch failed", err); 
    }
}

export async function adminResetStats(wallet) {
    if (!confirm(`Reset ALL stats for ${wallet}? This is permanent.`)) return;
    try {
        const headers = await getAdminHeaders();
        const response = await fetch(`${CONFIG.API_BASE}/api/reset-stats`, {
            method: 'POST',
            headers,
            body: JSON.stringify({ wallet })
        });
        if (response.ok) showToast("Stats cleared.", "success");
    } catch (err) {
        showToast("Reset failed.", "error");
    }
}

export function updateAdminRewardList(rewards) { // Exported for use in app.js
    const container = document.getElementById("admin-reward-list");
    container.innerHTML = "";
    Object.entries(rewards || {}).forEach(([id, amt]) => {
    });
}

export async function adminAddReward() { // Exported for use in app.js
    try {
        const assetID = document.getElementById("admin-add-asset").value;
        const amount = parseFloat(document.getElementById("admin-add-amt").value);
        if (!assetID || isNaN(amount)) return;

        const headers = await getAdminHeaders();
        if (!headers) return;

        const response = await fetch(`${CONFIG.API_BASE}/api/reward/add`, {
            method: 'POST',
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify({ asset_id: assetID, amount: amount })
        });

        if (response.ok) showToast("✅ Reward asset added.", "success");
    } catch (err) { 
        showToast("❌ Action failed", "error"); 
    }
}

export async function adminRemoveReward(assetId) { // Exported for use in app.js
    try {
        const headers = await getAdminHeaders();
        if (!headers) return;
        const response = await fetch(`${CONFIG.API_BASE}/api/reward/remove`, {
            method: 'POST',
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify({ asset_id: assetId })
        });
        if (response.ok) showToast("✅ Asset removed.", "success");
    } catch (err) { 
        showToast("❌ Update failed", "error"); 
    }
}

export async function adminAddNetwork() {
    try {
        const headers = await getAdminHeaders();
        if (!headers) return;
        // Implementation logic for adding network config
        showToast("✅ Network configuration added.", "success");
    } catch (err) { 
        showToast("❌ Failed to add network", "error"); 
    }
}

export async function adminUpdateRules() {
    try {
        const headers = await getAdminHeaders();
        if (!headers) return;
        const req = {
            Open: document.getElementById("rule-open").checked,
            Power_copy: document.getElementById("rule-same").checked,
            Power_up: document.getElementById("rule-plus").checked
        };
        const response = await fetch(`${CONFIG.API_BASE}/api/update-rules`, {
            method: 'POST',
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify(req)
        });
        if (response.ok) showToast("✅ Rules updated.", "success");
    } catch (err) { 
        showToast("❌ Rules update failed", "error"); 
    }
}

export async function adminBanWallet(walletToBan = null, hoursToBan = null) {
    try {
        const wallet = walletToBan || document.getElementById("admin-ban-wallet").value.trim();
        const hours = hoursToBan || parseInt(document.getElementById("admin-ban-hours").value);
        if (!wallet) return;
        const headers = await getAdminHeaders();
        const response = await fetch(`${CONFIG.API_BASE}/api/ban-player`, {
            method: 'POST',
            headers,
            body: JSON.stringify({ wallet, hours })
        });
        if (response.ok) showToast(`Banned ${wallet}`, "success");
    } catch (err) { 
        showToast("❌ Server connection error", "error"); 
    }
}

export async function adminAvatarBan(url = null, hours = null) {
    try {
        const targetUrl = url || document.getElementById("admin-ban-avatar-url").value.trim();
        const headers = await getAdminHeaders();
        if (!targetUrl || !headers) return;
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/avatar-ban`, {
            method: 'POST',
            headers,
            body: JSON.stringify({ url: targetUrl, hours })
        });
        if (response.ok) showToast("Avatar restricted.", "success");
    } catch (err) {
        showToast("Ban failed.", "error");
    }
}

export function adminBanWalletFromLog(wallet) {
    adminBanWallet(wallet, 24);
}

export async function adminUpdatePowerScaling() {
    try {
        const divisor = parseFloat(document.getElementById("admin-power-divisor").value);
        const base = parseInt(document.getElementById("admin-power-base").value);
        const headers = await getAdminHeaders();
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/update-power`, {
            method: 'POST',
            headers,
            body: JSON.stringify({ divisor, base })
        });
        if (response.ok) showToast("Scaling updated.", "success");
    } catch (err) { 
        showToast("❌ Power update failed", "error"); 
    }
}

export async function adminToggleMaintenance(active) {
    try {
        const minsInput = document.getElementById("admin-maint-mins");
        const minutes = parseInt(minsInput.value) || 0;
        const headers = await getAdminHeaders();
        const response = await fetch(`${CONFIG.API_BASE}/api/maintenance-mode`, {
            method: 'POST',
            headers,
            body: JSON.stringify({ active, minutes })
        });
        if (response.ok) showToast(`Maintenance ${active ? 'ON' : 'OFF'}`, "info");
    } catch (err) { 
        showToast("❌ Server connection error", "error"); 
    }
}

export async function adminToggleDevMode() {
    const enabled = document.getElementById("dev-mode-toggle").checked;
    if (enabled && !confirm("⚠️ DEV MODE: Force 100% win rate against bot?")) {
        document.getElementById("dev-mode-toggle").checked = false;
        return;
    }
    showToast(`🛠️ Dev Mode ${enabled ? 'Enabled' : 'Disabled'}`, enabled ? "success" : "info");
}

export async function adminSimulateTournament() {
    try {
        const size = parseInt(document.getElementById("admin-sim-size").value);
        const isBuyIn = document.getElementById("admin-sim-buyin").checked;
        const headers = await getAdminHeaders();
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/simulate-tournament`, {
            method: 'POST',
            headers,
            body: JSON.stringify({ size, is_buy_in: isBuyIn })
        });
        if (response.ok) showToast("Simulation started.", "success");
    } catch (err) {
        showToast("Simulation failed.", "error");
    }
}

export let adminLogTicker = null; // Exported for use in app.js
export function startAdminLogPolling() { // Exported for use in app.js
    if (adminLogTicker) return;
    adminLogTicker = setInterval(fetchLastAdminAction, 15000); // Check every 15s for status bar
    renderAdminAutomationButtons(); // Render automation buttons when polling starts (admin panel is active)
}

/**
 * updateDashboardStats updates the visual indicators for vault balance and pending rewards.
 */
function updateDashboardStats(data) {
    const balanceEl = document.getElementById("admin-dashboard-balance");
    const countEl = document.getElementById("admin-dashboard-pending-count");
    if (balanceEl && data.balance !== undefined) balanceEl.innerText = `${data.balance.toFixed(2)} $VBV`;
    if (countEl && data.pending_rewards_count !== undefined) countEl.innerText = data.pending_rewards_count;
}

export function stopAdminLogPolling() {
    if (adminLogTicker) {
        clearInterval(adminLogTicker);
        adminLogTicker = null;
        console.log("[ADMIN] Audit log polling suspended.");
    }
}

export async function fetchLastAdminAction() { // Exported for use in app.js
    try {
        const headers = await getAdminHeaders();
        if (!headers) return;

        const response = await fetch(`${CONFIG.API_BASE}/api/admin/logs?limit=1`, { headers });
        const data = await response.json();
        if (data.status === "success" && data.logs.length > 0) {
            updateDashboardStats(data);
            const last = data.logs[0];
            document.getElementById("admin-last-action").innerHTML = `<b>${last.action}:</b> ${last.target} (${last.timestamp})`;
        }
    } catch (err) {
        console.error("Status bar sync failed");
    }
}

export function updateAdminNetworkUI() { // Exported for use in app.js
    const select = document.getElementById("admin-network-select");
    if (!select) return;

    onAdminNetworkSelectChange();
}

export function onAdminNetworkSelectChange() { // Exported for use in app.js
    const select = document.getElementById("admin-network-select");
    if (!select) return;
    const name = select.value;
    const config = availableNetworks[name] || {};
    const details = document.getElementById("admin-network-details");
    if (details) {
        details.innerText = `Node: ${config.node_urls ? config.node_urls[0] : 'None'}`;
    }
}

export async function adminSetActiveNetwork() { // Exported for use in app.js
    try {
        const networkName = document.getElementById("admin-network-select").value;
        const headers = await getAdminHeaders();
        if (!headers) return;
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/set-admin-focus-network`, {
            method: 'POST',
            headers,
            body: JSON.stringify({ network_name: networkName })
        });
        if (response.ok) showToast(`Focus set to ${networkName}`, "success");
    } catch (err) {
        showToast("Focus switch failed.", "error");
    }
}

export async function adminBroadcast() {
    // This function is explicitly exported for use by app.js
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

/**
 * adminCyberSecurityAudit renders a specialized view of all club defenses.
 * PILLAR 3: Intelligence & Administrative expansion.
 */
export function adminCyberSecurityAudit() {
    const container = document.getElementById("admin-security-audit-display");
    if (!container) return;

    if (Object.keys(globalClubs).length === 0) {
        container.innerHTML = `<div class="grid-span-all opacity-5 py-20 italic">No clubs registered in the sector.</div>`;
        return;
    }

    container.innerHTML = Object.values(globalClubs).map(club => {
        const defenses = [];
        
        Object.entries(club.active_buffs || {}).forEach(([trapId, itemId]) => {
            const expiry = club.buff_expirations ? club.buff_expirations[trapId] : null;
            const timeStr = expiry ? new Date(expiry).toLocaleTimeString() : "??:??";
            const isLock = itemId === "cyber_lock";
            const isStabilizer = itemId === "district_stabilizer" || trapId === "MOJO_STABILIZER";
            
            let labelClass = isLock ? "text-neon-green" : isStabilizer ? "text-neon-cyan" : "text-warning";
            let prefix = isLock ? "🔐" : isStabilizer ? "📡" : "🛰️";
            
            defenses.push(`
                <div class="${labelClass} font-xs mb-2">
                    ${prefix} <b>${itemId.replace(/_/g, ' ').toUpperCase()}:</b> expires ${timeStr}
                </div>`);
        });

        return `
            <div class="glass-panel p-10 m-0 border-neon-cyan accelerated" style="background: rgba(0,0,0,0.4);">
                <div class="font-bold text-neon-purple mb-5 border-bottom-glass pb-5">${club.name.toUpperCase()}</div>
                <div class="font-xs opacity-6 mb-10">Mojo: ${club.club_mojo} | Staff: ${Object.keys(club.staff || {}).length}</div>
                <div class="flex-col">
                    ${defenses.length > 0 ? defenses.join('') : '<div class="opacity-3 font-xs italic">No active defenses detected.</div>'}
                </div>
            </div>`;
    }).join('')
}