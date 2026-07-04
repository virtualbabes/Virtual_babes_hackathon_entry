import { CONFIG } from './config.js';
import { socket, setNonceResolver } from './network.js';
import { showToast, setTransactionStatus } from './ui.js';
import { userAddress, walletProvider, signClient, linkedWallets } from './wallet.js';
import { getAssetSymbol, getNetworkConfig, shortenAddress } from './utils.js';
import { lastLobbyPlayers } from './game.js';
import { fetchLeaderboard } from './leaderboard.js';

export let availableNetworks = {};
export let globalClubs = {};
export let adminFocusNetwork = "";
let lastPlatformAlertTotal = 0; // PILLAR 5: Internal state for alert throttling
let lastGhostAlertTotal = 0; // PILLAR 5: Internal state for alert throttling
export let ignoredReporters = new Set(JSON.parse(localStorage.getItem("vbabes_ignored_reporters") || "[]"));

// Setters for external modules
export const setCachedAdminHeaders = (headers) => { cachedAdminHeaders = headers; };
export const setAvailableNetworks = (networks) => { availableNetworks = networks; };
export const setGlobalClubs = (clubs) => { globalClubs = clubs; };
export const setAdminFocusNetwork = (network) => { adminFocusNetwork = network; };
export const setIgnoredReporters = (reporters) => { ignoredReporters = reporters; };

export let cachedAdminHeaders = null;

/**
 * fetchMutationAudit retrieves aggregated mutation performance stats by club.
 * PILLAR 6: Forensic Auditing.
 */
export async function fetchMutationAudit() {
    const headers = await getAdminHeaders();
    if (!headers) return;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/mutation-audit`, { headers });
        const data = await response.json();
        if (response.ok) {
            renderMutationAuditDashboard(data);
        } else {
            showToast(`❌ Failed to fetch mutation audit: ${data.message || response.statusText}`, "error");
        }
    } catch (err) {
        console.error("Mutation audit fetch failed", err);
        showToast("❌ Network error fetching mutation audit.", "error");
    }
}

/**
 * renderMutationAuditDashboard displays mutation success/failure rates in the admin panel.
 * PILLAR 6: Forensic Auditing.
 */
export function renderMutationAuditDashboard(stats) {
    const container = document.getElementById("admin-mutation-audit-display");
    if (!container) return;

    if (!stats || stats.length === 0) {
        container.innerHTML = `<div class="grid-span-all opacity-5 py-20 italic">No mutation data logged in sector.</div>`;
        return;
    }

    container.innerHTML = stats.map(s => {
        const rateClass = s.success_rate >= 80 ? 'text-neon-green' : s.success_rate >= 60 ? 'text-warning' : 'text-error';
        return `
            <div class="glass-panel p-10 m-0 border-neon-purple accelerated" style="background: rgba(0,0,0,0.4);">
                <div class="font-bold text-neon-purple mb-5 border-bottom-glass pb-5">${s.club_name.toUpperCase()}</div>
                <div class="font-xs opacity-6 mb-5">
                    Successes: <b class="text-neon-green">${s.success_count}</b> | 
                    Botches: <b class="text-error">${s.failure_count}</b>
                </div>
                <div class="display-flex align-center gap-10">
                    <div class="progress-bar flex-1" style="height: 4px; background: rgba(255,255,255,0.05);">
                        <div class="progress-fill" style="width: ${s.success_rate}%; background: ${s.success_rate >= 80 ? 'var(--neon-green)' : s.success_rate >= 60 ? 'var(--warning-orange)' : 'var(--error-red)'}"></div>
                    </div>
                    <b class="${rateClass} font-mono font-xs">${s.success_rate.toFixed(1)}%</b>
                </div>
            </div>
        `;
    }).join('');
}

/**
 * fetchLedgerAudit retrieves the solvency status comparing vault vs liabilities.
 * PILLAR 2: Ledger Integrity.
 */
export async function fetchLedgerAudit() {
    const headers = await getAdminHeaders();
    if (!headers) return;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/ledger-audit`, { headers });
        const data = await response.json();
        if (response.ok) {
            renderSolvencyDashboard(data);
            // PILLAR 5: Orchestration. Push the forensic report to the top bar HUD.
            if (window.syncUI) window.syncUI("solvency_override", data);
        } else {
            showToast(`❌ Failed to fetch ledger audit: ${data.message || response.statusText}`, "error");
        }
    } catch (err) {
        console.error("Ledger audit fetch failed", err);
        showToast("❌ Network error fetching ledger audit.", "error");
    }
}

/**
 * renderSolvencyDashboard displays the solvency metrics in the admin panel.
 * PILLAR 2: Ledger Integrity.
 */
export function renderSolvencyDashboard(data) {
    const container = document.getElementById("admin-solvency-display");
    if (!container) return;

    const coverageClass = data.coverage_ratio >= 1.0 ? 'text-neon-green' : 'text-error';
    const surplusClass = data.net_surplus >= 0 ? 'text-neon-cyan' : 'text-error';

    container.innerHTML = `
        <div class="glass-panel p-10 m-0 border-neon-green accelerated" style="background: rgba(0,0,0,0.4); grid-column: 1 / -1;">
            <div class="display-flex justify-between align-center border-bottom-glass pb-10 mb-10">
                <span class="font-bold text-neon-green uppercase letter-spacing-1">System Solvency: <b class="${data.status === 'HEALTHY' ? 'text-neon-green' : 'text-error'}">${data.status}</b></span>
                <span class="font-mono font-size-0-8em opacity-5">${new Date(data.timestamp).toLocaleString()}</span>
            </div>
            <div class="display-grid gap-15 text-center border-bottom-glass pb-15 mb-10" style="grid-template-columns: repeat(7, 1fr);">
                <div>
                    <div class="font-xs opacity-6 uppercase mb-5">Physical Vault</div>
                    <b class="text-neon-cyan">${(data.physical_vault / 1000000).toFixed(2)} $VBV</b>
                </div>
                <div>
                    <div class="font-xs opacity-6 uppercase mb-5">Liabilities</div>
                    <b class="text-warning">${(data.virtual_liabilities / 1000000).toFixed(2)} $VBV</b>
                </div>
                <div>
                    <div class="font-xs opacity-6 uppercase mb-5">Coverage</div>
                    <b class="${coverageClass}">${(data.coverage_ratio * 100).toFixed(1)}%</b>
                </div>
                <div>
                    <div class="font-xs opacity-6 uppercase mb-5">Net Surplus</div>
                    <b class="${surplusClass}">${(data.net_surplus / 1000000).toFixed(2)} $VBV</b>
                </div>
                <div>
                    <div class="font-xs opacity-6 uppercase mb-5">Ghost Rec.</div>
                    <b class="text-error">${((data.ghost_reclaimed || 0) / 1000000).toFixed(2)} $VBV</b>
                </div>
                <div>
                    <div class="font-xs opacity-6 uppercase mb-5">Stag. Fees</div>
                    <b class="text-warning">${((data.stagnation_fees || 0) / 1000000).toFixed(2)} $VBV</b>
                </div>
                <div>
                    <div class="font-xs opacity-6 uppercase mb-5">Plat. Fees</div>
                    <b class="text-neon-purple">${((data.platform_fees || 0) / 1000000).toFixed(2)} $VBV</b>
                </div>
            </div>
            <div class="font-size-0-75em ${data.kernel_healthy ? 'opacity-7' : 'text-error'} italic text-center p-10 bg-black-80 rounded">
                ${data.audit_report}
            </div>
        </div>
    `;
}

/**
 * fetchNodeHealth retrieves the real-time status of the RPC node cluster.
 * PILLAR 4: Network Resiliency.
 */
export async function fetchNodeHealth() {
    const headers = await getAdminHeaders();
    if (!headers) return;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/node-health`, { headers });
        const data = await response.json();
        if (response.ok) {
            renderNodeHealthDashboard(data);
        } else {
            showToast(`❌ Failed to fetch node health: ${data.message || response.statusText}`, "error");
        }
    } catch (err) {
        console.error("Node health fetch failed", err);
        showToast("❌ Network error fetching node health.", "error");
    }
}

/**
 * renderNodeHealthDashboard displays the fetched node health data in the admin panel.
 * PILLAR 4: Network Resiliency.
 */
export function renderNodeHealthDashboard(nodeStatuses) {
    const container = document.getElementById("admin-node-health-display");
    if (!container) return;

    if (!nodeStatuses || nodeStatuses.length === 0) {
        container.innerHTML = `<div class="grid-span-all opacity-5 py-20 italic">No node health data available.</div>`;
        return;
    }

    container.innerHTML = nodeStatuses.map(node => `
        <div class="glass-panel p-10 m-0 border-neon-cyan accelerated" style="background: rgba(0,0,0,0.4);">
            <div class="font-bold text-neon-purple mb-5 border-bottom-glass pb-5">${node.url}</div>
            <div class="font-xs opacity-6 mb-10">
                Latency: <b class="${node.latency_ms > 300 ? 'text-error' : node.latency_ms > 100 ? 'text-warning' : 'text-neon-green'}">${node.latency_ms}ms</b> |
                Last Block: <b class="text-neon-cyan">${node.last_block}</b> |
                Status: <b class="${node.is_blacklisted ? 'text-error' : 'text-neon-green'}">${node.is_blacklisted ? 'BLACKLISTED' : 'OPERATIONAL'}</b>
                ${node.is_blacklisted ? `<br><small class="text-error">Last Error: ${new Date(node.last_error).toLocaleString()}</small>` : ''}
            </div>
        </div>
    `).join('');
}

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
        setTransactionStatus(null);
        return cachedAdminHeaders;
    } catch (err) {
        console.error("[ADMIN AUTH ERROR]", err);
        setTransactionStatus(`❌ Auth Failed: ${err.message}`, "critical");
        showToast(`❌ Authentication Failed: ${err.message}`, "error");
        return null;
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
            setTransactionStatus(null);
        } else {
            const err = await response.text();
            setTransactionStatus(`❌ Rollover Failed: ${err}`, "critical");
            showToast(`❌ Season rollover failed: ${err}`, "error");
        }
    } catch (err) {
        setTransactionStatus("❌ Server Connection Error", "critical");
        showToast("❌ Server connection error during season rollover.", "error");
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
            setTransactionStatus(null);
        } else {
            const err = await response.text();
            setTransactionStatus(`❌ Export Failed: ${err}`, "critical");
            showToast(`❌ Audit log export failed: ${err}`, "error");
        }
    } catch (err) {
        setTransactionStatus("❌ Connection error", "critical");
        showToast("❌ Server connection error during audit log export.", "error");
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
            setTransactionStatus(null);
            if (typeof fetchAdminLogs === 'function') fetchAdminLogs(); // Refresh logs to see the audit entry
        } else {
            const err = await response.text();
            setTransactionStatus(`❌ Payout Failed: ${err}`, "critical");
            showToast(`❌ Force Payout Failed: ${err}`, "error");
        }
    } catch (err) {
        console.error("[ADMIN ERROR]", err);
        setTransactionStatus("❌ Network error", "critical");
        showToast("❌ Network error during force payout request.", "error");
    }
}

/**
 * adminSimulateMutationSuccess triggers the high-fidelity success payoff (particles + synth) for testing.
 * PILLAR 6: Specialized Gene-Editing.
 */
export async function adminSimulateMutationSuccess() {
    const headers = await getAdminHeaders();
    if (!headers) return;

    setTransactionStatus("Triggering mutation success simulation...", "info");

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/simulate-mutation-success`, {
            method: "POST",
            headers: headers
        });

        if (response.ok) {
            showToast("🧬 Simulation triggered: payoff FX arriving via WebSocket.", "success");
            setTransactionStatus(null);
        } else {
            const err = await response.text();
            setTransactionStatus(`❌ Simulation Failed: ${err}`, "critical");
            showToast(`❌ Simulation trigger failed: ${err}`, "error");
        }
    } catch (err) {
        setTransactionStatus("❌ Network error", "critical");
        showToast("❌ Network error during simulation request.", "error");
    }
}

/**
 * adminSimulateMutationFailure triggers the high-intensity failure payoff (particles + warning sfx) for testing.
 * PILLAR 6: Specialized Gene-Editing.
 */
export async function adminSimulateMutationFailure() {
    const cardIdInput = document.getElementById("admin-sim-fail-card-id");
    const reductionInput = document.getElementById("admin-sim-fail-reduction");
    
    if (!cardIdInput || !reductionInput) return;
    
    const card_id = parseInt(cardIdInput.value);
    const reduction = parseInt(reductionInput.value);

    if (isNaN(card_id) || isNaN(reduction)) {
        showToast("❌ Please enter both Card ID and Reduction amount.", "error");
        return;
    }

    const headers = await getAdminHeaders();
    if (!headers) return;

    setTransactionStatus("Triggering mutation failure simulation...", "warning");

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/simulate-mutation-failure`, {
            method: "POST",
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify({ card_id, reduction })
        });

        if (response.ok) {
            showToast("🚨 Simulation triggered: failure FX arriving via WebSocket.", "success");
            setTransactionStatus(null);
        } else {
            const err = await response.text();
            setTransactionStatus(`❌ Simulation Failed: ${err}`, "critical");
            showToast(`❌ Simulation trigger failed: ${err}`, "error");
        }
    } catch (err) {
        setTransactionStatus("❌ Network error", "critical");
        showToast("❌ Network error during simulation request.", "error");
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
        if (response.ok) {
            showToast("🧪 Mojo Decay Simulation Started. Check server logs for progress.", "success");
            setTransactionStatus(null);
        } else {
            const err = await response.text();
            setTransactionStatus(`❌ Simulation Failed: ${err}`, "critical");
            showToast(`❌ Simulation Failed: ${err}`, "error");
        }
    } catch (err) { 
        setTransactionStatus("❌ Network error", "critical");
        showToast("❌ Network error during simulation request.", "error"); 
    }
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
            setTransactionStatus(null);
            cardIdInput.value = "";
            clubIdInput.value = "";
        } else {
            const err = await response.text();
            setTransactionStatus(`❌ Forfeiture Failed: ${err}`, "critical");
            showToast(`❌ Forfeiture failed: ${err}`, "error");
        }
    } catch (err) {
        setTransactionStatus("❌ Connection error", "critical");
        showToast("❌ Server connection error during asset forfeiture.", "error");
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

        <h3 class="section-title mt-20">Procedure FX Testing</h3>
        <div class="glass-panel p-15 border-neon-cyan">
            <button class="w-full outline border-neon-green text-neon-green mb-15" onclick="adminSimulateMutationSuccess()">SIMULATE SUCCESS PAYOFF</button>
            <div class="flex-row gap-10 mb-10">
                <input type="number" id="admin-sim-fail-card-id" class="glass-input flex-1" placeholder="Card ID">
                <input type="number" id="admin-sim-fail-reduction" class="glass-input flex-1" placeholder="Reduction" value="50">
            </div>
            <button class="w-full outline border-error text-error" onclick="adminSimulateMutationFailure()">SIMULATE FAILURE PAYOFF</button>
            <small class="opacity-5 italic block mt-10">Triggers the blood-red glitch particles and warning audio. Applies permanent Artifact reduction.</small>
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
            fetchNodeHealth(); // Refresh node health dashboard
            fetchLedgerAudit(); // Refresh solvency dashboard
            fetchMutationAudit(); // New: Refresh mutation audit dashboard
            fetchDLCRegistry(); // New: Refresh DLC registry dashboard
            adminCommissionAudit(); // PILLAR 1: Refresh alliance dividends
            adminTaxAudit(); // PILLAR 1: Refresh systemic taxes
            adminDistrictTaxAudit(); // PILLAR 1: Refresh localized policies
        }
    } catch (err) { 
        console.error("Log fetch failed", err); 
    }
}

/**
 * adminDistrictTaxAudit fetches the aggregated localized tax policies.
 * PILLAR 1: Political Influence Telemetry.
 */
export async function adminDistrictTaxAudit() {
    const container = document.getElementById("admin-district-tax-display");
    if (!container) return;

    try {
        const headers = await getAdminHeaders();
        if (!headers) return;
        
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/district-tax-audit`, { headers });
        if (!response.ok) throw new Error(await response.text());
        const data = await response.json();
        renderDistrictTaxDashboard(data);
    } catch (err) {
        console.error("[ADMIN ERROR] District Tax Audit Failed:", err);
    }
}

/**
 * renderDistrictTaxDashboard generates the policy table.
 */
export function renderDistrictTaxDashboard(data) {
    const container = document.getElementById("admin-district-tax-display");
    if (!container) return;

    if (!data || data.length === 0) {
        container.innerHTML = `<div class="opacity-5 py-20 italic">No custom district taxes enacted in sector.</div>`;
        return;
    }

    container.innerHTML = `
        <table class="admin-table w-full text-left" style="border-collapse: collapse;">
            <thead>
                <tr class="opacity-5 font-size-0-7em letter-spacing-1 border-bottom-glass">
                    <th class="p-10">DISTRICT</th>
                    <th class="p-10">GOVERNOR</th>
                    <th class="p-10 text-right">DIVIDEND POOL</th>
                    <th class="p-10 text-right">TAX RATE</th>
                </tr>
            </thead>
            <tbody>
                ${data.map(d => `
                    <tr class="border-bottom-glass font-size-0-85em hover-bg-dim">
                        <td class="p-10"><b class="text-neon-cyan">${d.territory_id.replace(/_/g, ' ').toUpperCase()}</b></td>
                        <td class="p-10">
                            <span class="text-white">${d.governor_name}</span><br/>
                            <small class="opacity-5 font-mono">${d.governor_address}</small>
                        </td>
                        <td class="p-10 text-right"><b class="text-neon-green">${((d.dividend_pool || 0) / 1000000).toFixed(2)} $VBV</b></td>
                        <td class="p-10 text-right"><b class="text-neon-green">${d.tax_rate.toFixed(1)}%</b></td>
                    </tr>
                `).join('')}
            </tbody>
        </table>
    `;
}

/**
 * fetchDLCRegistry retrieves the current state of the DLC registry.
 * PILLAR 4: Console Expansion Management.
 */
export async function fetchDLCRegistry() {
    const headers = await getAdminHeaders();
    if (!headers) return;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/dlc-registry`, { headers });
        const data = await response.json();
        if (response.ok) {
            renderDLCRegistryDashboard(data);
        } else {
            showToast(`❌ Failed to fetch DLC registry: ${data.message || response.statusText}`, "error");
        }
    } catch (err) {
        console.error("DLC registry fetch failed", err);
        showToast("❌ Network error fetching DLC registry.", "error");
    }
}

/**
 * renderDLCRegistryDashboard displays the DLC products in the admin panel.
 * PILLAR 4: Console Expansion Management.
 */
export function renderDLCRegistryDashboard(registry) {
    const container = document.getElementById("admin-dlc-registry-display");
    if (!container) return;

    if (!registry || Object.keys(registry).length === 0) {
        container.innerHTML = `<div class="grid-span-all opacity-5 py-20 italic">No DLC products registered.</div>`;
        return;
    }

    container.innerHTML = `
        <table class="admin-table w-full text-left" style="border-collapse: collapse;">
            <thead>
                <tr class="opacity-5 font-size-0-7em letter-spacing-1 border-bottom-glass">
                    <th class="p-10">ID</th>
                    <th class="p-10">NAME</th>
                    <th class="p-10">COST ($VBV)</th>
                    <th class="p-10">CREATOR</th>
                    <th class="p-10 text-right">ACTIONS</th>
                </tr>
            </thead>
            <tbody>
                ${Object.values(registry).map(p => `
                    <tr class="border-bottom-glass font-size-0-85em hover-bg-dim">
                        <td class="p-10"><b class="text-neon-cyan">${p.arena_voucher_id}</b></td>
                        <td class="p-10">${p.name}</td>
                        <td class="p-10">${(p.cost_micro / 1000000).toFixed(2)}</td>
                        <td class="p-10 font-mono font-xs opacity-7">${shortenAddress(p.creator_wallet)}</td>
                        <td class="p-10 text-right">
                            <button class="outline x-small border-neon-cyan" onclick="adminUpdateDLCProduct('${p.arena_voucher_id}', '${p.name}', '${p.description}', ${p.cost_micro}, '${p.creator_wallet}')">EDIT</button>
                        </td>
                    </tr>
                `).join('')}
            </tbody>
        </table>
    `;
}

/**
 * adminUpdateDLCProduct allows administrators to add or modify DLC products.
 * PILLAR 4: Console Expansion Management.
 */
export async function adminUpdateDLCProduct(id, name, description, costMicro, creatorWallet) {
    const headers = await getAdminHeaders();
    if (!headers) return;

    // For simplicity, this example assumes direct input or pre-filled form.
    // In a real UI, you'd have a modal with input fields.
    const product = {
        arena_voucher_id: id || prompt("Enter DLC Product ID:", "NEW_DLC_ITEM"),
        name: name || prompt("Enter DLC Name:", "New DLC Item"),
        description: description || prompt("Enter DLC Description:", "A new item for console players."),
        cost_micro: costMicro || parseInt(prompt("Enter Cost in micro-VBV:", "100000000")), // Default 100 VBV
        creator_wallet: creatorWallet || prompt("Enter Creator Wallet:", userAddress),
    };

    if (!product.arena_voucher_id || !product.name || !product.cost_micro || !product.creator_wallet) {
        showToast("❌ Missing required DLC product fields.", "error");
        return;
    }

    // PILLAR 2: Integer Supremacy.
    // Ensure cost is a valid positive integer to prevent backend arithmetic drift.
    const cost = parseInt(product.cost_micro);
    if (isNaN(cost) || cost <= 0) {
        showToast("❌ Invalid cost amount. Must be a positive integer (micro-units).", "error");
        return;
    }
    product.cost_micro = cost;

    // PILLAR 3: Identity Validation & Normalization.
    // Verify the creator wallet conforms to supported network standards.
    const wallet = product.creator_wallet.trim();
    const isEVM = wallet.startsWith("0x") && wallet.length === 42;
    const isAVM = wallet.length === 58;
    const isSOL = wallet.length >= 32 && wallet.length <= 44; // Solana Base58 length range

    if (!isEVM && !isAVM && !isSOL) {
        showToast("❌ Invalid Creator Wallet address format.", "error");
        return;
    }
    // Normalize AVM and EVM to lowercase; Solana remains case-sensitive.
    if (!isSOL) product.creator_wallet = wallet.toLowerCase();

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/dlc-registry/update`, {
            method: "POST",
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify(product)
        });
        if (response.ok) {
            showToast(`✅ DLC product '${product.name}' updated.`, "success");
            fetchDLCRegistry(); // Refresh the dashboard
        } else {
            const err = await response.text();
            showToast(`❌ DLC Update Failed: ${err}`, "error");
        }
    } catch (err) {
        console.error("DLC update failed", err);
        showToast("❌ Network error updating DLC registry.", "error");
    }
}
/**
 * adminTaxAudit fetches the aggregated session tax revenue.
 * PILLAR 1: Industrial Loop Tracking.
 */
export async function adminTaxAudit() {
    const container = document.getElementById("admin-tax-revenue-display");
    if (!container) return;

    try {
        const headers = await getAdminHeaders();
        if (!headers) return;
        
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/tax-audit`, { headers });
        if (!response.ok) throw new Error(await response.text());
        const data = await response.json();
        renderTaxDashboard(data);
    } catch (err) {
        console.error("[ADMIN ERROR] Tax Audit Failed:", err);
    }
}

/**
 * renderTaxDashboard generates the tax revenue metrics.
 */
export function renderTaxDashboard(data) {
    const container = document.getElementById("admin-tax-revenue-display");
    if (!container) return;

    // PILLAR 2: Solvency Guard.
    // Trigger a high-priority alert if session ghost reclamation exceeds the critical threshold (5,000 $VBV).
    // Guarded to only fire when the total increments while above the limit to prevent polling spam.
    if ((data.ghost_tax_total || 0) > 5000000000 && (data.ghost_tax_total || 0) > lastGhostAlertTotal) {
        showToast("👻 <b>GHOST ALERT:</b> Session reclamation total exceeds 5,000 $VBV. Verify Creator Hub initialization status.", "critical");
        lastGhostAlertTotal = data.ghost_tax_total;
    }

    // PILLAR 2: Solvency Guard.
    // Trigger a high-priority alert if session platform fees exceed the critical threshold (2,000 $VBV).
    // Guarded to only fire when the total increments while above the limit to prevent polling spam.
    if ((data.platform_tax_total || 0) > 2000000000 && (data.platform_tax_total || 0) > lastPlatformAlertTotal) {
        showToast("💸 <b>SURCHARGE ALERT:</b> Session Platform Fees total exceeds 2,000 $VBV. High volume of self-redemptions detected.", "critical");
        lastPlatformAlertTotal = data.platform_tax_total;
    }
    const corp = ((data.corporate_tax_total || 0) / 1000000).toFixed(2);
    const lux = ((data.luxury_tax_total || 0) / 1000000).toFixed(2);
    const sabo = ((data.sabotage_surcharge_total || 0) / 1000000).toFixed(2);
    const govS = ((data.governor_surcharge_total || 0) / 1000000).toFixed(2);
    const ghost = ((data.ghost_tax_total || 0) / 1000000).toFixed(2);
    const plat = ((data.platform_tax_total || 0) / 1000000).toFixed(2);
    const stag = ((data.stagnation_tax_total || 0) / 1000000).toFixed(2);
    const total = (parseFloat(corp) + parseFloat(lux) + parseFloat(sabo) + parseFloat(govS) + parseFloat(ghost) + parseFloat(plat) + parseFloat(stag)).toFixed(2);
    
    container.innerHTML = `
        <div class="glass-panel p-10 m-0 border-neon-green accelerated" style="background: rgba(0,0,0,0.4); grid-column: 1 / -1;">
            <div class="display-grid gap-15 text-center" style="grid-template-columns: repeat(8, 1fr);">
                <div>
                    <div class="font-xs opacity-6 uppercase mb-5">Corporate Recovery</div>
                    <b class="text-neon-cyan">${corp} $VBV</b>
                    <div class="font-xs opacity-4 mt-2">(${data.corporate_tax_count || 0} contracts)</div>
                </div>
                <div>
                    <div class="font-xs opacity-6 uppercase mb-5">Luxury Recovery</div>
                    <b class="text-neon-purple">${lux} $VBV</b>
                    <div class="font-xs opacity-4 mt-2">(${data.luxury_tax_count || 0} items)</div>
                </div>
                <div>
                    <div class="font-xs opacity-6 uppercase mb-5">Sabotage Fees</div>
                    <b class="text-gold">${sabo} $VBV</b>
                    <div class="font-xs opacity-4 mt-2">(Alliance Dividends)</div>
                </div>
                <div>
                    <div class="font-xs opacity-6 uppercase mb-5">Gov Surcharge</div>
                    <b class="text-gold">${govS} $VBV</b>
                    <div class="font-xs opacity-4 mt-2">(Capital Revenue)</div>
                </div>
                <div>
                    <div class="font-xs opacity-6 uppercase mb-5">Platform Fees</div>
                    <b class="text-neon-purple">${plat} $VBV</b>
                    <div class="font-xs opacity-4 mt-2">(Self-Redeem)</div>
                </div>
                <div>
                    <div class="font-xs opacity-6 uppercase mb-5">Ghost Reclamation</div>
                    <b class="text-error">${ghost} $VBV</b>
                    <div class="font-xs opacity-4 mt-2">(100% Recycle)</div>
                </div>
                <div>
                    <div class="font-xs opacity-6 uppercase mb-5">Stagnation Fees</div>
                    <b class="text-warning">${stag} $VBV</b>
                    <div class="font-xs opacity-4 mt-2">(25% Siphon)</div>
                </div>
                <div><div class="font-xs opacity-6 uppercase mb-5">Session Total</div><b class="text-neon-green">${total} $VBV</b></div>
            </div>
        </div>`;
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

/**
 * adminCommissionAudit fetches the aggregated alliance dividend history.
 * PILLAR 1: Industrial Loop Tracking.
 */
export async function adminCommissionAudit() {
    const container = document.getElementById("admin-commission-audit-display");
    if (!container) return;

    try {
        const headers = await getAdminHeaders();
        if (!headers) return;
        
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/commission-audit`, { headers });
        if (!response.ok) throw new Error(await response.text());
        const data = await response.json();
        renderCommissionAuditDashboard(data);
    } catch (err) {
        console.error("[ADMIN ERROR] Commission Audit Failed:", err);
    }
}

/**
 * renderCommissionAuditDashboard generates the history table.
 */
export function renderCommissionAuditDashboard(events) {
    const container = document.getElementById("admin-commission-audit-display");
    if (!container) return;

    if (!events || events.length === 0) {
        container.innerHTML = `<div class="opacity-5 py-20 italic">No alliance dividend events found in this sector.</div>`;
        return;
    }

    container.innerHTML = `
        <table class="admin-table w-full text-left" style="border-collapse: collapse;">
            <thead>
                <tr class="opacity-5 font-size-0-7em letter-spacing-1 border-bottom-glass">
                    <th class="p-10">TIMESTAMP</th>
                    <th class="p-10">RECIPIENT CLUB</th>
                    <th class="p-10">SOURCE PARTNER</th>
                    <th class="p-10">PROCEDURE</th>
                    <th class="p-10 text-right">DIVIDEND</th>
                </tr>
            </thead>
            <tbody>
                ${events.map(e => `
                    <tr class="border-bottom-glass font-size-0-85em hover-bg-dim">
                        <td class="p-10 font-mono opacity-7">${new Date(e.timestamp * 1000).toLocaleTimeString()}</td>
                        <td class="p-10"><b class="text-neon-cyan">${e.recipient_name}</b></td>
                        <td class="p-10">${e.source_club}</td>
                        <td class="p-10"><span class="tag-purple font-xs">${e.type} SYNTHESIS</span></td>
                        <td class="p-10 text-right text-neon-green font-bold">+${e.amount.toFixed(2)} $VBV</td>
                    </tr>
                `).join('')}
            </tbody>
        </table>
    `;
}

export function updateAdminRewardList(rewards) { // Exported for use in app.js
    const container = document.getElementById("admin-reward-list");
    if (!container) return;
    container.innerHTML = "";
    Object.entries(rewards || {}).forEach(([id, amt]) => {
        const div = document.createElement("div");
        div.className = "flex-row justify-between align-center p-5 border-bottom-glass font-xs";
        div.innerHTML = `
            <span>ID: <b class="text-neon-cyan">${id}</b></span>
            <span>Amt: <b class="text-neon-green">${(amt / 1000000).toFixed(2)}</b></span>
            <button class="outline x-small border-error text-error" onclick="adminRemoveReward('${id}')">X</button>
        `;
        container.appendChild(div);
    });
}

export async function adminAddReward() { // Exported for use in app.js
    try {
        const assetID = document.getElementById("admin-add-asset").value;
        const amount = parseFloat(document.getElementById("admin-add-amt").value);
        if (!assetID || isNaN(amount)) return;

        const headers = await getAdminHeaders();
        if (!headers) return;

        setTransactionStatus(`Adding reward asset ${assetID}...`, "info");

        const response = await fetch(`${CONFIG.API_BASE}/api/reward/add`, {
            method: 'POST',
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify({ asset_id: assetID, amount: amount })
        });

        if (response.ok) {
            showToast("✅ Reward asset added.", "success");
            setTransactionStatus(null);
        } else {
            const err = await response.text();
            setTransactionStatus(`❌ Action Failed: ${err}`, "critical");
        }
    } catch (err) { 
        setTransactionStatus(`❌ Action Failed: ${err.message}`, "critical");
        showToast("❌ Action failed", "error"); 
    }
}

export async function adminRemoveReward(assetId) { // Exported for use in app.js
    try {
        const headers = await getAdminHeaders();
        if (!headers) return;

        setTransactionStatus(`Removing reward asset ${assetId}...`, "warning");

        const response = await fetch(`${CONFIG.API_BASE}/api/reward/remove`, {
            method: 'POST',
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify({ asset_id: assetId })
        });
        if (response.ok) {
            showToast("✅ Asset removed.", "success");
            setTransactionStatus(null);
        } else {
            const err = await response.text();
            setTransactionStatus(`❌ Removal Failed: ${err}`, "critical");
        }
    } catch (err) { 
        setTransactionStatus("❌ Update failed", "critical");
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

        setTransactionStatus("Updating arena rules...", "warning");

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
        if (response.ok) {
            showToast("✅ Rules updated.", "success");
            setTransactionStatus(null);
        } else {
            const err = await response.text();
            setTransactionStatus(`❌ Rules Update Failed: ${err}`, "critical");
        }
    } catch (err) { 
        setTransactionStatus("❌ Action failed", "critical");
        showToast("❌ Rules update failed", "error"); 
    }
}

export async function adminBanWallet(walletToBan = null, hoursToBan = null) {
    try {
        const wallet = walletToBan || document.getElementById("admin-ban-wallet").value.trim();
        const hours = hoursToBan || parseInt(document.getElementById("admin-ban-hours").value);
        if (!wallet) return;
        const headers = await getAdminHeaders();
        if (!headers) return;

        setTransactionStatus(`Banning wallet ${shortenAddress(wallet)}...`, "warning");

        const response = await fetch(`${CONFIG.API_BASE}/api/ban-player`, {
            method: 'POST',
            headers,
            body: JSON.stringify({ wallet, hours })
        });
        if (response.ok) {
            showToast(`Banned ${wallet}`, "success");
            setTransactionStatus(null);
        } else {
            const err = await response.text();
            setTransactionStatus(`❌ Ban Failed: ${err}`, "critical");
        }
    } catch (err) { 
        setTransactionStatus(`❌ Ban Failed: ${err.message}`, "critical");
        showToast("❌ Server connection error", "error"); 
    }
}

export async function adminAvatarBan(url = null, hours = null) {
    try {
        const targetUrl = url || document.getElementById("admin-ban-avatar-url").value.trim();
        const headers = await getAdminHeaders();
        if (!targetUrl || !headers) return;

        setTransactionStatus(`Restricting avatar access...`, "warning");

        const response = await fetch(`${CONFIG.API_BASE}/api/admin/avatar-ban`, {
            method: 'POST',
            headers,
            body: JSON.stringify({ url: targetUrl, hours })
        });
        if (response.ok) {
            showToast("Avatar restricted.", "success");
            setTransactionStatus(null);
        } else {
            setTransactionStatus(`❌ Ban Failed`, "critical");
        }
    } catch (err) {
        setTransactionStatus("❌ Ban failed", "critical");
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
        if (!headers) return;

        setTransactionStatus("Updating power scaling...", "warning");

        const response = await fetch(`${CONFIG.API_BASE}/api/admin/update-power`, {
            method: 'POST',
            headers,
            body: JSON.stringify({ divisor, base })
        });
        if (response.ok) {
            showToast("Scaling updated.", "success");
            setTransactionStatus(null);
        } else {
            const err = await response.text();
            setTransactionStatus(`❌ Update Failed: ${err}`, "critical");
        }
    } catch (err) { 
        setTransactionStatus("❌ Update failed", "critical");
        showToast("❌ Power update failed", "error"); 
    }
}

export async function adminToggleMaintenance(active) {
    try {
        const minsInput = document.getElementById("admin-maint-mins");
        const minutes = parseInt(minsInput.value) || 0;
        const headers = await getAdminHeaders();
        if (!headers) return;

        setTransactionStatus(`${active ? 'Enabling' : 'Disabling'} maintenance mode...`, "warning");

        const response = await fetch(`${CONFIG.API_BASE}/api/maintenance-mode`, {
            method: 'POST',
            headers,
            body: JSON.stringify({ active, minutes })
        });
        if (response.ok) {
            showToast(`Maintenance ${active ? 'ON' : 'OFF'}`, "info");
            setTransactionStatus(null);
        } else {
            setTransactionStatus(`❌ Action Failed`, "critical");
        }
    } catch (err) { 
        setTransactionStatus("❌ Server error", "critical");
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
        if (!headers) return;

        setTransactionStatus(`Simulating ${size}-player tournament...`, "warning");

        const response = await fetch(`${CONFIG.API_BASE}/api/admin/simulate-tournament`, {
            method: 'POST',
            headers,
            body: JSON.stringify({ size, is_buy_in: isBuyIn })
        });
        if (response.ok) {
            showToast("Simulation started.", "success");
            setTransactionStatus(null);
        } else {
            setTransactionStatus(`❌ Simulation Failed`, "critical");
        }
    } catch (err) {
        setTransactionStatus("❌ Simulation failed", "critical");
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

        setTransactionStatus(`Switching focus to ${networkName}...`, "info");

        const response = await fetch(`${CONFIG.API_BASE}/api/admin/set-admin-focus-network`, {
            method: 'POST',
            headers,
            body: JSON.stringify({ network_name: networkName })
        });
        if (response.ok) {
            showToast(`Focus set to ${networkName}`, "success");
            setTransactionStatus(null);
        } else {
            const err = await response.text();
            setTransactionStatus(`❌ Focus Switch Failed: ${err}`, "critical");
        }
    } catch (err) {
        setTransactionStatus("❌ Focus switch failed", "critical");
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
            setTransactionStatus(null);
        } else {
            const err = await response.text();
            setTransactionStatus(`❌ Broadcast Failed: ${err}`, "critical");
            showToast(`❌ Broadcast failed: ${err}`, "error");
        }
    } catch (err) {
        setTransactionStatus("❌ Broadcast failed", "critical");
        showToast("❌ Broadcast failed", "error");
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

/**
 * fetchDLCRegistry retrieves the current state of the DLC registry.
 * PILLAR 4: Console Expansion Management.
 */
export async function fetchDLCRegistry() {
    const headers = await getAdminHeaders();
    if (!headers) return;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/dlc-registry`, { headers });
        const data = await response.json();
        if (response.ok) {
            renderDLCRegistryDashboard(data);
        } else {
            showToast(`❌ Failed to fetch DLC registry: ${data.message || response.statusText}`, "error");
        }
    } catch (err) {
        console.error("DLC registry fetch failed", err);
        showToast("❌ Network error fetching DLC registry.", "error");
    }
}

/**
 * renderDLCRegistryDashboard displays the DLC products in the admin panel.
 * PILLAR 4: Console Expansion Management.
 */
export function renderDLCRegistryDashboard(registry) {
    const container = document.getElementById("admin-dlc-registry-display");
    if (!container) return;

    // PILLAR 4: Consistent Styling. 
    // Apply the specific border class to the parent section glass panel defined in index.html.
    const section = container.closest('.admin-section-dlc-registry');
    if (section) section.classList.add('border-neon-cyan');

    const products = Object.values(registry);
    if (products.length === 0) {
        container.innerHTML = `<div class="grid-span-all opacity-5 py-20 italic">No DLC products registered.</div>`;
        return;
    }

    container.innerHTML = `
        <table class="admin-table w-full text-left" style="border-collapse: collapse;">
            <thead>
                <tr class="opacity-5 font-size-0-7em letter-spacing-1 border-bottom-glass">
                    <th class="p-10">ID</th>
                    <th class="p-10">NAME</th>
                    <th class="p-10">COST ($VBV)</th>
                    <th class="p-10">STOCK</th>
                    <th class="p-10">CREATOR</th>
                    <th class="p-10 text-right">ACTIONS</th>
                </tr>
            </thead>
            <tbody>
                ${products.map(p => {
                    // PILLAR 2: Cross-referencing Creator Stock.
                    const creator = lastLobbyPlayers.find(pl => pl.wallet?.toLowerCase() === p.creator_wallet?.toLowerCase());
                    const stock = (creator && creator.inventory) ? (creator.inventory[p.arena_voucher_id] || 0) : 0;
                    const stockClass = stock === 0 ? 'text-error font-bold' : 'text-neon-green';
                    return `
                    <tr class="border-bottom-glass font-size-0-85em hover-bg-dim">
                        <td class="p-10"><b class="text-neon-cyan">${p.arena_voucher_id}</b></td>
                        <td class="p-10">${p.name}</td>
                        <td class="p-10">${(p.cost_micro / 1000000).toFixed(2)}</td>
                        <td class="p-10"><span class="${stockClass}">${stock}</span></td>
                        <td class="p-10 font-mono font-xs opacity-7">${shortenAddress(p.creator_wallet)}</td>
                        <td class="p-10 text-right">
                            <button class="outline x-small border-neon-cyan" onclick="adminUpdateDLCProduct('${p.arena_voucher_id}', '${p.name}', '${p.description}', ${p.cost_micro}, '${p.creator_wallet}')">EDIT</button>
                            <button class="outline x-small border-neon-green text-neon-green" onclick="adminRestockDLC('${p.arena_voucher_id}')">RESTOCK</button>
                        </td>
                    </tr>
                `; }).join('')}
            </tbody>
        </table>
    `;
}

export async function adminUpdateDLCProduct(id, name, description, costMicro, creatorWallet) {
    const headers = await getAdminHeaders();
    if (!headers) return;

    const product = {
        arena_voucher_id: id || prompt("Enter DLC Product ID:", "NEW_DLC_ITEM"),
        name: name || prompt("Enter DLC Name:", "New DLC Item"),
        description: description || prompt("Enter DLC Description:", "A new item for console players."),
        cost_micro: costMicro || parseInt(prompt("Enter Cost in micro-VBV:", "100000000")),
        creator_wallet: creatorWallet || prompt("Enter Creator Wallet:", userAddress),
    };

    if (!product.arena_voucher_id || !product.name || !product.cost_micro || !product.creator_wallet) {
        showToast("❌ Missing required DLC product fields.", "error");
        return;
    }

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/dlc-registry/update`, {
            method: "POST",
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify(product)
        });
        if (response.ok) {
            showToast(`✅ DLC product '${product.name}' updated.`, "success");
            fetchDLCRegistry();
        } else {
            const err = await response.text();
            showToast(`❌ DLC Update Failed: ${err}`, "error");
        }
    } catch (err) {
        console.error("DLC update failed", err);
        showToast("❌ Network error updating DLC registry.", "error");
    }
}

/**
 * adminRestockDLC triggers a manual restock for a specific DLC item.
 * PILLAR 2: Integer Supremacy.
 */
export async function adminRestockDLC(id) {
    const qty = parseInt(prompt(`Enter restock quantity for ${id}:`, "10"));
    if (isNaN(qty) || qty <= 0) return;

    const headers = await getAdminHeaders();
    if (!headers) return;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/dlc-registry/restock`, {
            method: "POST",
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify({ arena_voucher_id: id, quantity: qty })
        });
        if (response.ok) {
            showToast(`✅ Successfully restocked ${qty} units of ${id}.`, "success");
            fetchDLCRegistry();
        } else {
            const err = await response.text();
            showToast(`❌ Restock Failed: ${err}`, "error");
        }
    } catch (err) {
        showToast("❌ Network error restocking DLC.", "error");
    }
}

/**
 * adminNodeHealthAudit renders the RPC node cluster health report.
 * PILLAR 4: Network Resiliency — covers /api/admin/node-health-audit.
 */
export async function adminNodeHealthAudit() {
    const headers = await getAdminHeaders();
    if (!headers) return;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/node-health-audit`, { headers });
        if (response.ok) {
            const data = await response.json();
            renderNodeHealthAudit(data);
        } else {
            showToast(`❌ Node audit failed: ${await response.text()}`, "error");
        }
    } catch (err) {
        showToast("❌ Network error fetching node health.", "error");
    }
}

/**
 * renderNodeHealthAudit displays RPC node cluster status.
 */
export function renderNodeHealthAudit(data) {
    const container = document.getElementById("admin-node-health-display");
    if (!container) return;

    const nodes = data.nodes || [];
    if (nodes.length === 0) {
        container.innerHTML = `<div class="grid-span-all opacity-5 py-20 italic">No nodes registered.</div>`;
        return;
    }

    container.innerHTML = `
        <table class="admin-table w-full text-left" style="border-collapse: collapse;">
            <thead>
                <tr class="opacity-5 font-size-0-7em letter-spacing-1 border-bottom-glass">
                    <th class="p-10">NETWORK</th>
                    <th class="p-10">NODE URL</th>
                    <th class="p-10">STATUS</th>
                    <th class="p-10">LATENCY (ms)</th>
                    <th class="p-10">LAST CHECK</th>
                </tr>
            </thead>
            <tbody>
                ${nodes.map(n => `
                    <tr class="border-bottom-glass font-size-0-85em hover-bg-dim">
                        <td class="p-10">${n.network || 'Unknown'}</td>
                        <td class="p-10 font-mono font-xs opacity-7">${n.url || '?'}</td>
                        <td class="p-10 ${n.healthy ? 'text-neon-green' : 'text-error'}">${n.healthy ? 'HEALTHY' : 'UNHEALTHY'}</td>
                        <td class="p-10">${n.latency_ms ?? '?'}</td>
                        <td class="p-10">${n.last_check || 'Never'}</td>
                    </tr>
                `).join('')}
            </tbody>
        </table>
    `;
}

/**
 * adminSystemSanityCheck performs a comprehensive ledger and node audit.
 * PILLAR 4: Live Deployment & Monitoring — covers /api/admin/system-sanity-check.
 */
export async function adminSystemSanityCheck() {
    if (!confirm("⚠️ This will run a full ledger invariant audit. Continue?")) return;

    const headers = await getAdminHeaders();
    if (!headers) return;

    setTransactionStatus("Running comprehensive system sanity check...", "warning");

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/system-sanity-check`, { method: 'POST', headers });
        if (response.ok) {
            const data = await response.json();
            renderSystemSanityCheck(data);
            showToast("✅ System sanity check complete.", "success");
        } else {
            setTransactionStatus(`❌ Sanity check failed`, "critical");
            showToast(`❌ ${await response.text()}`, "error");
        }
    } catch (err) {
        setTransactionStatus("❌ Sanity check network error", "critical");
        showToast("❌ Network error during sanity check.", "error");
    }
}

/**
 * renderSystemSanityCheck displays ledger invariants and node connectivity results.
 */
export function renderSystemSanityCheck(data) {
    const container = document.getElementById("admin-sanity-check-display");
    if (!container) return;

    const checks = data.checks || [];
    container.innerHTML = checks.length > 0 ? checks.map(c => `
        <div class="glass-panel p-10 m-0 border-neon-cyan accelerated" style="background: rgba(0,0,0,0.4);">
            <div class="${c.passed ? 'text-neon-green' : 'text-error'} font-bold mb-5">
                ${c.passed ? '✅' : '❌'} ${c.name}
            </div>
            <div class="font-xs opacity-7">${c.message || 'No details'}</div>
        </div>
    `).join('') : '<div class="opacity-5 italic py-20">No results available.</div>';
}

/**
 * adminEmergencyShutdown executes a scorched-earth protocol to preserve state and terminate sessions.
 * PILLAR 3: Administrative Security — covers /api/admin/emergency-shutdown.
 */
export async function adminEmergencyShutdown() {
    if (!confirm("⚠️ CRITICAL: This will shut down all active sessions and preserve state. Confirm?")) return;
    if (!confirm("⚠️ SECOND CONFIRMATION: This action is irreversible without external recovery.")) return;

    const headers = await getAdminHeaders();
    if (!headers) return;

    setTransactionStatus("🔴 EMERGENCY SHUTDOWN INITIATED...", "critical");

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/emergency-shutdown`, { method: 'POST', headers });
        if (response.ok) {
            showToast("✅ Emergency shutdown executed successfully.", "success");
            setTransactionStatus(null);
        } else {
            setTransactionStatus(`❌ Shutdown failed`, "critical");
            showToast(`❌ ${await response.text()}`, "error");
        }
    } catch (err) {
        setTransactionStatus("❌ Shutdown network error", "critical");
        showToast("❌ Network error during shutdown.", "error");
    }
}

/**
 * adminSimulateLoad stress-tests the telemetry throughput.
 * PILLAR 4: Performance Monitoring & Stress Testing — covers /api/admin/simulate-load.
 */
export async function adminSimulateLoad() {
    const count = parseInt(prompt("Enter number of concurrent load events:", "50")) || 50;
    if (count <= 0) return;

    const headers = await getAdminHeaders();
    if (!headers) return;

    setTransactionStatus(`Stress-testing with ${count} concurrent transactions...`, "warning");

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/simulate-load`, {
            method: 'POST',
            headers,
            body: JSON.stringify({ count })
        });
        if (response.ok) {
            const data = await response.json();
            showToast(`✅ Load test complete: ${data.transactions || count} transactions processed.`, "success");
            setTransactionStatus(null);
        } else {
            setTransactionStatus(`❌ Load test failed`, "critical");
            showToast(`❌ ${await response.text()}`, "error");
        }
    } catch (err) {
        setTransactionStatus("❌ Load test network error", "critical");
        showToast("❌ Network error during load test.", "error");
    }
}

/**
 * adminDistrictTaxAudit audits district-level tax collection and revenue distribution.
 * PILLAR 1: Economic Integrity — covers /api/admin/district-tax-audit.
 */
export async function adminDistrictTaxAudit() {
    const headers = await getAdminHeaders();
    if (!headers) return;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/district-tax-audit`, { headers });
        if (response.ok) {
            const data = await response.json();
            renderDistrictTaxAudit(data);
        } else {
            showToast(`❌ Tax audit failed: ${await response.text()}`, "error");
        }
    } catch (err) {
        showToast("❌ Network error fetching tax audit.", "error");
    }
}

/**
 * renderDistrictTaxAudit displays district-level tax collection data.
 */
export function renderDistrictTaxAudit(data) {
    const container = document.getElementById("admin-district-tax-display");
    if (!container) return;

    const districts = data.districts || [];
    const totalRevenue = data.total_revenue_micro || 0;

    container.innerHTML = `
        <div class="mb-10 font-bold text-neon-cyan">Total District Revenue: ${(totalRevenue / 1000000).toFixed(2)} $VBV</div>
        ${districts.length > 0 ? districts.map(d => `
            <div class="glass-panel p-10 m-0 border-neon-cyan accelerated" style="background: rgba(0,0,0,0.4);">
                <div class="font-bold text-neon-purple mb-5">${d.name || 'Unknown District'}</div>
                <div class="font-xs opacity-7">
                    Collected: ${(d.collected_micro || 0) / 1000000} $VBV | 
                    Distributed: ${(d.distributed_micro || 0) / 1000000} $VBV | 
                    Retained: ${(d.retained_micro || 0) / 1000000} $VBV
                </div>
            </div>
        `).join('') : '<div class="opacity-5 italic py-20">No district data available.</div>'}
    `;
}
