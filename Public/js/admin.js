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

export async function getAdminHeaders() {
    if (!userAddress) { 
    }
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

    }
}

export async function adminRefillVault() { // Exported for use in app.js
    const amount = parseFloat(document.getElementById("admin-refill-amt").value);
    if (isNaN(amount)) return;
    const headers = await getAdminHeaders();
    } catch (err) { showToast("❌ Refill failed", "error"); }
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
    } catch (err) { showToast("❌ Broadcast failed", "error"); }
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
