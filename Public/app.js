import { collectiveIntelligence } from './collective-intelligence.js';
import { CONFIG } from './js/config.js';
import { initWebSocket, sendPing, handleServerMessage, myClientId, setMyClientId, setNonceResolver, getNonceResolver, setReconnectAttempts, getReconnectAttempts, setIdentitySyncTimeout, getIdentitySyncTimeout, setLastPingTime, getLastPingTime, setCurrentLatency, getCurrentLatency } from './js/network.js';
import { showToast, setTransactionStatus, hideAllOverlays, showMatchPreview, showTournamentTransition, showKidnapOverlay } from './js/ui.js';
import { initWalletConnect, connectWith, disconnectUserWallet, updateWalletUI, walletProvider, setWalletProvider, signClient, wcModal } from './js/wallet.js';
import { updatePayoutUI, openPayoutSettings, savePayoutAddress, payoutAddress, setPayoutAddress } from './js/wallet.js';
import { fetchLeaderboard, renderSideLeaderboard, switchHofTab, fetchTournamentHistory, filterSeasonHistory, fetchSeasonHistory, updateTournamentPaginationUI, handleTournamentUI, renderTournamentBracket, registerForTournament, openTournamentBracket, closeTournamentBracket, currentTournamentPage, tournamentLimit, totalTournaments, lastTournamentData, seasonEnd, seasonTimerInterval, startSeasonTimer } from './js/leaderboard.js';
import { updatePlayerList, handleMatchmakingUpdate, toggleMatchmakingQueue, inMatchmakingQueue, setInMatchmakingQueue, sendChallenge, acceptChallenge, declineChallenge, sendMatchSync, sendSpectate, showMatchPreview as gameShowMatchPreview, proceedToWarRoom, sendPing as gameSendPing } from './js/game.js';
import { openDeckManager, closeDeckManager, renderDeckManager, refreshInventory, renderAvatarGrid, applyAvatarFilters, selectAvatar, setupCropEvents, userNFTs, setUserNFTs, currentAvatarUrl, setCurrentAvatarUrl, cropState, setCropState, isCropInitialized, setIsCropInitialized } from './js/deck.js';
import { getAdminHeaders, adminRefillVault, updateAdminRewardList, adminAddReward, adminRemoveReward, adminAddNetwork, adminBroadcast, adminUpdateRules, adminBanWallet, adminGloatBan, adminAvatarBan, adminBanWalletFromLog, adminUpdatePowerScaling, adminToggleMaintenance, adminToggleDevMode, adminResetStats, adminSimulateTournament, startAdminLogPolling, fetchLastAdminAction, lastAdminKey, cachedAdminHeaders, setCachedAdminHeaders, availableNetworks, setAvailableNetworks, globalClubs, setGlobalClubs, adminFocusNetwork, setAdminFocusNetwork, updateAdminNetworkUI, onAdminNetworkSelectChange } from './js/admin.js';
import { openShopsOverlay, switchShopCategory, buyClubItem, openClubFoundry, submitClubFoundry, openTerritoryView, openArtGalleryOverlay, loadGalleryItems, promptBid, openConsignmentOverlay, selectConsignmentItem, submitConsignment } from './js/economy.js';
import { openCourthouse, submitCourthouseFine, openPortfolioView, switchPortfolioTab, initiateBail, openSecuritySentry, deployTrap, openBountyBoard, openBlackMarket, buyBlackMarketItem, openRumorMill, spreadRumor, openHeistPlanningOverlay, updateHeistRiskAssessment, executeHeistStrike, handleHeistResult, openKidnapSelectionOverlay, executeKidnap, payRansom, releaseHostage, showKidnapOverlay as criminalityShowKidnapOverlay, startRecoveryTimer } from './js/criminality.js';
import { setMasterVolume, setMusicVolume, setSfxVolume, toggleMuteMaster, toggleMuteMusic, toggleMuteSfx, masterVolume, musicVolume, sfxVolume } from './js/audio.js';
import { initParticleSystem, animateParticles, triggerCaptureParticles, particles, particleCanvas, particleCtx, particleAnimationId } from './js/particles.js';
import { getCachedEnvoiName, resolveAssetSymbol, getAssetSymbol, resolveEnvoiName, getNetworkConfig, assetCache, envoiCache } from './js/utils.js';

let activeCardId = null; // Tracks the card you clicked in your hand
let aiThinking = false; // To track if AI is currently performing a move
let lastBoardState = Array(9).fill(null); // Track state to detect captures
let currentChallengerId = null; // Stores the ID of the player who sent the current challenge
let currentOpponentId = null;   // The player we are currently battling
let spectatorMatchState = null; // Stores P1/P2 mapping for spectators
let isVerified = false;         // Global verification status
let lastTauntPhase = null;      // Tracks narrative state to prevent duplicate taunts
let lastTauntTurn = null;
let activeRumors = []; // Global array to hold active rumors

// --- Ticker State ---
let tickerItems = [];

let myPlayerIndex = 0;          // 0 for P1, 1 for P2
let userAddress = null;
let matchHistorySaved = false;
let lastLobbyPlayers = []; // Cache for portfolio valuation
let linkedWallets = JSON.parse(localStorage.getItem("vbabes_linked_wallets") || "[]");
let ignoredReporters = new Set(JSON.parse(localStorage.getItem("vbabes_ignored_reporters") || "[]"));

let tooltipEl = null;
// 1. Initialize Go WASM Engine
window.onload = async () => {
    const go = new Go();
    try {
        const response = await fetch("main.wasm");
        const buffer = await response.arrayBuffer();
        const obj = await WebAssembly.instantiate(buffer, go.importObject);

        // Initialize volume sliders
        document.getElementById("master-volume").value = masterVolume; // Assuming masterVolume is defined in audio.js
        document.getElementById("music-volume").value = musicVolume;   // Assuming musicVolume is defined in audio.js
        document.getElementById("sfx-volume").value = sfxVolume;       // Assuming sfxVolume is defined in audio.js
        go.run(obj.instance);
        if (window.SetApiBase) window.SetApiBase(CONFIG.API_BASE);
        if (window.SetAssetBase) {
            window.SetAssetBase(CONFIG.ASSET_URL);
            // Set specific CSS variables for background textures as CSS url() doesn't support concatenation
            const base = CONFIG.ASSET_URL;
            document.documentElement.style.setProperty('--bg-arena-floor', `url(${base}Assets/Textures/arena_floor.png)`);
            document.documentElement.style.setProperty('--bg-glass-texture', `url(${base}Assets/Textures/glass_texture.webp)`);
            // NEW: Define dynamic arena floor textures
            document.documentElement.style.setProperty('--texture-solo', `url(${base}Assets/Textures/arena_solo.webp)`);
            document.documentElement.style.setProperty('--texture-challenge', `url(${base}Assets/Textures/arena_challenge.webp)`);
            document.documentElement.style.setProperty('--texture-tournament', `url(${base}Assets/Textures/arena_tournament.webp)`);
            document.documentElement.style.setProperty('--texture-semi', `url(${base}Assets/Textures/arena_semi_final.webp)`);
            document.documentElement.style.setProperty('--texture-final', `url(${base}Assets/Textures/arena_final.webp)`);
        }
        document.getElementById("engine-status").innerHTML = "<span class='status-active'>ACTIVE</span>";
        buildEmptyBoard(); // Imported from game.js
        initWebSocket(handleServerMessage); // Pass the message handler
        initWalletConnect(); // Initialize WC alongside WS
        renderMatchHistory();
        fetchLeaderboard();
        setupCropEvents();
        initParticleSystem(); // Initialize particle system

        updatePayoutUI();
        // Check for soft-reload resume
        if (localStorage.getItem("vbabes_soft_reload") === "true") {
            const lastWallet = localStorage.getItem("vbabes_last_wallet");
            const lastProvider = localStorage.getItem("vbabes_last_provider");
            localStorage.removeItem("vbabes_soft_reload");
            if (lastWallet && lastProvider) {
                setTimeout(() => connectWith(lastProvider), 500); // Small delay for WASM stability
            }
        }
    } catch (err) {
        console.error("WASM Load Fail:", err);
        document.getElementById("engine-status").innerHTML = "<span style='color: #ff0844;'>OFFLINE</span>";
    }
    syncUI(); // Initial UI sync after WASM loads
};

// Expose HTML event handlers for inline onclicks in module mode
window.handleWalletAction = handleWalletAction;
window.connectWith = connectWith;
window.closeWalletSelector = closeWalletSelector;
window.hideAllOverlays = hideAllOverlays;
window.openPayoutSettings = openPayoutSettings;
window.savePayoutAddress = savePayoutAddress;
window.toggleMuteMusic = toggleMuteMusic;
window.toggleMatchmakingQueue = toggleMatchmakingQueue;
window.sendChatMessage = sendChatMessage;
window.registerForTournament = registerForTournament;
window.openTournamentBracket = openTournamentBracket;
window.openDeckManager = openDeckManager;
window.openShopsOverlay = openShopsOverlay;
window.openTerritoryMapOverlay = openTerritoryMapOverlay;
window.ToggleLeaderboard = window.ToggleLeaderboard || ToggleLeaderboard;
window.openSettingsOverlay = openSettingsOverlay;
window.proceedToWarRoom = proceedToWarRoom; // Imported from game.js
window.closeDeckManager = closeDeckManager; // Imported from deck.js

window.updateWalletUI = (address) => { // This function will be moved to wallet.js
    userAddress = address;
    const btn = document.getElementById("wallet-btn");
    if (address) {
        btn.innerText = address.substring(0, 8) + "...";
        btn.classList.add("outline");

        // Register wallet with the Live Lobby server
        if (socket && socket.readyState === WebSocket.OPEN) {
            socket.send(JSON.stringify({
                type: "register_wallet",
                payload: { wallet: address }
            }));
        }

        // BRIDGE LOGIC: If on Algorand, ensure the user is "birthed" into Voi
        const state = window.GetGameState(); // This will need to be imported or passed
        if (state.network === "ALGO") {
            checkVoiReadiness(address);
        }

        // If we are in the Setup phase, fetch the player's NFT collection for avatar selection
        if (state.phase === "Setup") {
            fetchUserNFTs(address);
        }
    } else {
        btn.innerText = "Connect Wallet";
        btn.classList.remove("outline");
    }
    syncUI(); // This will be a global function in app.js (defined below)
    showMainGameContainer(); // Ensure main game is visible after wallet action
}

async function checkVoiReadiness(address) {
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
                const voiConfig = availableNetworks["Voi Mainnet"];
                if (voiConfig && voiConfig.explorer_url) {
                    const explorerLink = `${voiConfig.explorer_url}/tx/${data.txid}`;
                    message += `<br><a href="${explorerLink}" target="_blank" style="color: var(--neon-green); text-decoration: underline;">View Transaction</a>`;
                } else {
                    message += `<br>TxID: ${data.txid.substring(0, 14)}...`;
                }
            }
            showToast(message, "success", 10000);
        }
        if (response.status === 503) { // Handle Service Unavailable specifically
            const errorText = await response.text();
            showToast(`⚠️ <b>ONBOARDING UNAVAILABLE:</b> ${errorText}`, "warning", 10000);
            return;
        }
    } catch (err) {
        console.error("[BRIDGE] Onboarding check failed", err);
    } finally {
        setTransactionStatus(null);
    }
}

window.adminGloatBan = async (walletToBan = null, hoursToBan = null) => { // This will be moved to admin.js
    const wallet = walletToBan || document.getElementById("admin-ban-wallet").value.trim();
    const hours = hoursToBan || parseInt(document.getElementById("admin-ban-hours").value);
    
    if (!wallet) return;
    const headers = await getAdminHeaders();
    if (!headers) return;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/gloat-ban`, {
            method: "POST",
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify({ wallet, hours: isNaN(hours) ? 48 : hours })
        });
        if (response.ok) {
            showToast(`🚫 Gloat messaging restricted for ${wallet.substring(0,6)}...`, "success");
            fetchLastAdminAction();
        } else {
            const errText = await response.text();
            showToast(`❌ Gloat ban failed: ${errText}`, "error");
        }
    } catch (err) { showToast("❌ Server connection error", "error"); }
}

window.ignoreReporter = (wallet) => { // This will be moved to admin.js
    if (!wallet) return;
    ignoredReporters.add(wallet);
    localStorage.setItem("vbabes_ignored_reporters", JSON.stringify(Array.from(ignoredReporters)));
    showToast(`🙈 Ignoring reports from ${wallet.substring(0,6)}...`, "info");
    fetchAdminLogs(); // Re-render to apply filter
}

window.fetchAdminLogs = async () => { // This will be moved to admin.js
    const headers = await getAdminHeaders();
    if (!headers) return;

    const filter = document.getElementById("admin-log-filter").value.trim();
    const startVal = document.getElementById("admin-log-start").value;
    const endVal = document.getElementById("admin-log-end").value;

    const logDisplay = document.getElementById("admin-logs-display");
    logDisplay.innerHTML = `<div class="chat-msg system">Fetching logs...</div>`;

    try {
        let url = `${CONFIG.API_BASE}/api/admin/logs?filter=${encodeURIComponent(filter)}`;
        if (startVal) {
            url += `&start_date=${encodeURIComponent(startVal + ":00Z")}`;
        }
        if (endVal) {
            url += `&end_date=${encodeURIComponent(endVal + ":00Z")}`;
        }

        const response = await fetch(url, { headers });
        if (!response.ok) throw new Error("Failed to fetch admin logs");
        const data = await response.json();

        logDisplay.innerHTML = "";
        if (data.logs && data.logs.length > 0) {
            data.logs.forEach(logJson => {
                let logEntry;
                try {
                    logEntry = JSON.parse(logJson);
                } catch (e) {
                    // Handle raw string logs (e.g., from older entries or malformed)
                    logDisplay.innerHTML += `<div class="chat-msg system">${logJson.raw || logJson}</div>`;
                    return;
                }

                // Extract reporter wallet if possible for ignoring logic
                let reporterWallet = "";
                let avatarUrl = "";
                if (logEntry.action === "REPORT_GLOAT") {
                    const rMatch = logEntry.details.match(/Reported by (.*?) for/);
                    if (rMatch) reporterWallet = rMatch[1];
                    
                    const aMatch = logEntry.details.match(/\[Avatar: (.*?)\]/);
                    if (aMatch) avatarUrl = aMatch[1];
                }

                // Local Filter for Ignored Reporters
                if (reporterWallet && ignoredReporters.has(reporterWallet)) return;

                const logDiv = document.createElement("div");
                logDiv.className = "chat-msg";
                if (logEntry.action === "REPORT_GLOAT") {
                    logDiv.classList.add("report-alert");
                }

                let logText = `[${new Date(logEntry.timestamp).toLocaleString()}] <b>${logEntry.action}</b>: ${logEntry.target} - ${logEntry.details}`;

                if (logEntry.action === "REPORT_GLOAT") {
                    const rawGloat = logEntry.details.split('for offensive gloat: "')[1]?.split('" [Avatar:')[0] || "REDACTED";
                    logText = `🚨 <b>REPORTED GLOAT</b>: ${logEntry.target} - "${rawGloat}"`;
                    
                    logDiv.innerHTML = `${logText} 
                        <div class="flex-row gap-5 mt-5 justify-end">
                            <button class="outline admin-action-btn-small text-error" onclick="adminBanWallet('${logEntry.target}', 24)">BAN PLAYER</button>
                            <button class="outline admin-action-btn-small text-error" onclick="adminGloatBan('${logEntry.target}', 48)">BAN GLOATS</button>
                            ${avatarUrl ? `<button class="outline admin-action-btn-small" style="border-color: #ffa657; color: #ffa657;" onclick="adminAvatarBan('${avatarUrl}', 720)">BAN AVATAR</button>` : ''}
                            <button class="outline admin-action-btn-small" style="border-color: #888; color: #888;" onclick="ignoreReporter('${reporterWallet}')">IGNORE REPORTER</button>
                        </div>`;
                } else if (logEntry.action.startsWith("SECURITY")) {
                    logDiv.style.backgroundColor = "rgba(255, 166, 87, 0.1)";
                    logDiv.style.border = "1px solid #ffa657";
                    logDiv.innerHTML = logText;
                } else {
                    logDiv.innerHTML = logText;
                }
                logDisplay.appendChild(logDiv);
            });
        } else {
            logDisplay.innerHTML = `<div class="chat-msg system">No logs found matching filter.</div>`;
        }
    } catch (err) {
        console.error("Failed to fetch admin logs:", err);
        logDisplay.innerHTML = `<div class="chat-msg system" style="color: #ff4b4b;">Error fetching logs.</div>`;
    }
}

window.getAdminHeaders = async () => { // This will be moved to admin.js
    if (!userAddress) { 
        showToast("Please connect your admin wallet.", "error");
        return null;
    }

    try {
        setTransactionStatus("Requesting admin nonce...", "info");
        // Request Nonce from Server via WebSocket (Anti-Replay)
        const nonce = await new Promise((resolve, reject) => {
            const timeout = setTimeout(() => reject(new Error("Nonce request timed out")), 10000);
            setNonceResolver((n) => { clearTimeout(timeout); resolve(n); });
            socket.send(JSON.stringify({ type: "nonce_request" }));
        });

        const msg = `Virtualbabes Arena Admin Auth:${nonce}`;
        const msgBytes = new TextEncoder().encode(msg);

        let signature = null;
        setTransactionStatus("Signing admin session...", "info");
        const isEVM = userAddress.startsWith("0x");

        if (isEVM) {
            if (walletProvider === 'walletconnect' && signClient) {
                const sessions = signClient.session.getAll();
                const hexMsg = "0x" + Array.from(msgBytes).map(b => b.toString(16).padStart(2, '0')).join('');
                signature = await signClient.request({
                    topic: sessions[0].topic,
                    chainId: "eip155:1",
                    request: {
                        method: "personal_sign",
                        params: [hexMsg, userAddress]
                    }
                });
            } else {
                throw new Error("EVM Admin Auth requires WalletConnect.");
            }
        } else if (walletProvider === 'nautilus') {
            const result = await window.algo.signMessage(msgBytes, userAddress);
            // Nautilus returns the raw Uint8Array signature
            signature = btoa(String.fromCharCode(...result));
        } else if (walletProvider === 'walletconnect' && signClient) {
            const sessions = signClient.session.getAll();
            const response = await signClient.request({
                topic: sessions[0].topic,
                chainId: (window.GetGameState().network === "VOI" ? CONFIG.VOI_CHAIN_ID : CONFIG.ALGO_CHAIN_ID),
                request: {
                    method: "algo_signMessage",
                    params: [userAddress, btoa(String.fromCharCode(...msgBytes))]
                }
            });
            signature = response; // WC response is usually the base64 string
        } else {
            throw new Error("Active wallet provider does not support signature authentication.");
        }

        if (!signature) throw new Error("Signature denied.");

        const headers = {
            'X-Admin-Wallet': userAddress,
            'X-Admin-Nonce': nonce,
            'X-Admin-Signature': signature
        };
        
        cachedAdminHeaders = headers; // Cache for background polling
        setTransactionStatus(null);
        return headers;
    } catch (err) {
        showToast(`❌ Admin Auth Failed: ${err.message}`, "error");
        setTransactionStatus(null);
        return null;
    }
}

// --- Admin Action Handlers ---
window.adminRefillVault = async () => { // This will be moved to admin.js
    const amount = parseFloat(document.getElementById("admin-refill-amt").value);
    if (isNaN(amount)) return;
    const headers = await getAdminHeaders();
    if (!headers) return;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/refill-vault`, {
            method: "POST",
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify({ amount })
        });
        if (response.ok) showToast("⚡ Vault balance updated successfully", "success");
        fetchLastAdminAction();
    } catch (err) { showToast("❌ Refill failed", "error"); }
}

window.updateAdminRewardList = (rewards) => { // This will be moved to admin.js
    const container = document.getElementById("admin-reward-list");
    container.innerHTML = "";
    Object.entries(rewards || {}).forEach(([id, amt]) => {
        const symbol = getAssetSymbol(id);
        const div = document.createElement("div");
        div.className = "player-item";
        div.style.padding = "5px";
        div.innerHTML = `<span style="font-size: 11px;">${symbol}: <b>${amt} units</b></span>
                         <button class="outline" style="padding: 2px 8px; font-size: 9px; border-color: #ff4b4b; color: #ff4b4b;" onclick="adminRemoveReward(${id})">DEL</button>`;
        container.appendChild(div);
    });
}

window.adminAddReward = async () => { // This will be moved to admin.js
    const assetId = parseInt(document.getElementById("admin-add-asset").value);
    const amount = parseFloat(document.getElementById("admin-add-amt").value);
    if (!assetId || isNaN(amount)) return;
    const headers = await getAdminHeaders();
    if (!headers) return;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/reward/add`, {
            method: "POST",
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify({ asset_id: assetId, amount: amount })
        });
        if (response.ok) showToast("➕ Reward added to stack", "success");
        fetchLastAdminAction();
    } catch (err) { showToast("❌ Action failed", "error"); }
}

window.adminRemoveReward = async (assetId) => { // This will be moved to admin.js
    const headers = await getAdminHeaders();
    if (!headers) return;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/reward/remove`, {
            method: "POST",
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify({ asset_id: assetId })
        });
        if (response.ok) showToast("➖ Reward removed", "info");
        fetchLastAdminAction();
    } catch (err) { showToast("❌ Update failed", "error"); }
}

window.adminAddNetwork = async () => { // This will be moved to admin.js
    const headers = await getAdminHeaders();
    if (!headers) return;

    const newNetwork = {
        network_name: document.getElementById("new-network-name").value.trim(),
        explorer_url: document.getElementById("new-explorer-url").value.trim(),
        indexer_url: document.getElementById("new-indexer-url").value.trim(),
        node_url: document.getElementById("new-node-url").value.trim(),
        faucet_url: document.getElementById("new-faucet-url").value.trim(),
        asset_id: parseInt(document.getElementById("new-asset-id").value) || 0,
        app_id: parseInt(document.getElementById("new-app-id").value) || 0,
        chain_id: document.getElementById("new-chain-id").value.trim(),
        power_divisor: parseFloat(document.getElementById("new-power-divisor").value) || 1000000,
        power_base: parseInt(document.getElementById("new-power-base").value) || 50,
    };

    // Basic validation
    if (!newNetwork.network_name || !newNetwork.node_url || !newNetwork.chain_id) {
        showToast("❌ Network Name, Node URL, and Chain ID are required.", "error");
        return;
    }

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/network/add`, {
            method: "POST",
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify(newNetwork)
        });
        if (response.ok) showToast(`➕ Network '${newNetwork.network_name}' added`, "success");
        fetchLastAdminAction();
    } catch (err) { showToast("❌ Failed to add network", "error"); }
}

window.adminBroadcast = async () => { // This will be moved to admin.js
    const text = document.getElementById("admin-msg-text").value;
    if (!text) return;
    const headers = await getAdminHeaders();
    if (!headers) return;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/system-message`, {
            method: "POST",
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify({ text })
        });
        if (response.ok) {
            showToast("📣 System message broadcasted", "success");
            fetchLastAdminAction();
            document.getElementById("admin-msg-text").value = "";
        }
    } catch (err) { showToast("❌ Broadcast failed", "error"); }
}

window.adminUpdateRules = async () => { // This will be moved to admin.js
    const req = {
        Open: document.getElementById("rule-open").checked,
        Power_copy: document.getElementById("rule-same").checked,
        Power_up: document.getElementById("rule-plus").checked,
        Elemental_sync: document.getElementById("rule-elemental").checked,
        Fallen_penalty: document.getElementById("rule-fallen").checked,
        Artifact_bonus: document.getElementById("rule-artifact").checked
    };
    const headers = await getAdminHeaders();
    if (!headers) return;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/update-rules`, {
            method: "POST",
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify(req)
        });
        if (response.ok) showToast("⚙️ Global rules synchronized", "success");
        fetchLastAdminAction();
    } catch (err) { showToast("❌ Rules update failed", "error"); }
}

window.adminBanWallet = async (walletToBan = null, hoursToBan = null) => { // This will be moved to admin.js
    const wallet = walletToBan || document.getElementById("admin-ban-wallet").value.trim();
    const hours = hoursToBan || parseInt(document.getElementById("admin-ban-hours").value);
    if (!wallet) return;
    const headers = await getAdminHeaders();
    if (!headers) return;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/ban-player`, {
            method: "POST",
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify({ wallet, hours: isNaN(hours) ? 24 : hours })
        });
        if (response.ok) {
            showToast(`🚫 Wallet ${wallet.substring(0,6)}... restricted`, "success");
            fetchLastAdminAction();
            if (!walletToBan) { // Only clear input if not called from log
                document.getElementById("admin-ban-wallet").value = "";
                document.getElementById("admin-ban-hours").value = "";
            }
            fetchLeaderboard(); // Refresh to show the lock icon
        } else {
            const errText = await response.text();
            showToast(`❌ Ban failed: ${errText}`, "error");
        }
    } catch (err) { showToast("❌ Server connection error", "error"); }
}

window.adminAvatarBan = async (url = null, hours = null) => { // This will be moved to admin.js
    const targetUrl = url || document.getElementById("admin-ban-avatar-url").value.trim();
    if (!targetUrl) return;
    const headers = await getAdminHeaders();
    if (!headers) return;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/avatar-ban`, {
            method: "POST",
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify({ url: targetUrl, hours: isNaN(hours) ? 720 : hours })
        });
        if (response.ok) {
            showToast(`🚫 Avatar asset restricted`, "success");
            fetchLastAdminAction();
            if (!url) {
                document.getElementById("admin-ban-avatar-url").value = "";
            }
        } else {
            const errText = await response.text();
            showToast(`❌ Avatar ban failed: ${errText}`, "error");
        }
    } catch (err) {
        showToast("❌ Server connection error", "error");
    }
}

window.adminBanWalletFromLog = (wallet) => { // This will be moved to admin.js
    // Default to 24 hours for a quick ban from logs
    adminBanWallet(wallet, 24);
}

window.adminUpdatePowerScaling = async () => { // This will be moved to admin.js
    const divisor = parseFloat(document.getElementById("admin-power-divisor").value);
    const base = parseInt(document.getElementById("admin-power-base").value);
    const headers = await getAdminHeaders();
    if (!headers) return;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/update-power`, {
            method: "POST",
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify({ divisor, base })
        });
        if (response.ok) showToast("⚖️ Power scaling adjusted", "success");
        fetchLastAdminAction();
    } catch (err) { showToast("❌ Power update failed", "error"); }
}

window.addXChainWallet = async () => { // This will be moved to wallet.js
    if (!signClient || !wcModal || !userAddress) return;
    showToast("Opening WalletConnect for X-Chain Link...", "info");

    try {
        // Allow connection to any chain supported by the current admin config
        const chainIds = Object.values(availableNetworks).map(n => n.chain_id).filter(id => id);
        const algoChains = chainIds.filter(id => id.startsWith("algorand:"));
        const evmChains = chainIds.filter(id => id.startsWith("eip155:"));
        const solanaChains = chainIds.filter(id => id.startsWith("solana:"));
        
        const { uri, approval } = await signClient.connect({
            optionalNamespaces: {
                algorand: {
                    methods: ["algo_signTxn", "algo_signMessage"],
                    chains: algoChains.length > 0 ? algoChains : [CONFIG.VOI_CHAIN_ID, CONFIG.ALGO_CHAIN_ID],
                    events: ["accountsChanged"]
                },
                eip155: {
                    methods: ["eth_sendTransaction", "personal_sign"],
                    chains: evmChains.length > 0 ? evmChains : ["eip155:1", "eip155:137"],
                    events: ["accountsChanged", "chainChanged"]
                },
                solana: {
                    methods: ["solana_signTransaction", "solana_signMessage"],
                    chains: solanaChains.length > 0 ? solanaChains : ["solana:5eykt4UsFvXYfy2khQbSsLurFBXY"],
                    events: ["accountsChanged"]
                }
            }
        });

            if (uri) {
                wcModal.openModal({ uri });
                const session = await approval();
                // The inner try-catch was for a specific check, not the whole block.
                if (session.namespaces.eip155) { // This check was misplaced
                    // Approval logic
                }
                wcModal.closeModal(); // This should be here

            let namespace = "";
            let accountStr = "";

            if (session.namespaces.eip155) {
                namespace = "eip155";
                accountStr = session.namespaces.eip155.accounts[0];
            } else if (session.namespaces.solana) {
                namespace = "solana";
                accountStr = session.namespaces.solana.accounts[0];
            } else if (session.namespaces.algorand) {
                namespace = "algorand";
                accountStr = session.namespaces.algorand.accounts[0];
            }

            if (!accountStr) throw new Error("No account returned from session");

            const [ns, chainId, address] = accountStr.split(":");
            const newWalletInfo = { // Renamed to avoid redeclaration
                address: address,
                chain: ns === 'eip155' ? (chainId === '1' ? 'ETH' : (chainId === '137' ? 'POLY' : 'EVM')) : 
                       (ns === 'solana' ? 'SOL' : (chainId === 'mainnet-v1.0' ? 'ALGO' : 'VOI')),
                id: `Linked-${Date.now()}`
            };

            // 1. Request Nonce for Link Verification
            showToast("Verifying ownership... please sign the request.", "info");
            const nonce = await new Promise((resolve, reject) => {
                const timeout = setTimeout(() => reject(new Error("Nonce request timed out")), 10000); // Imported from network.js
                nonceResolver = (n) => { clearTimeout(timeout); resolve(n); };
                socket.send(JSON.stringify({ type: "nonce_request" }));
            });

            // 2. Sign Nonce with the newly linked wallet
            let signature = null;
            if (namespace === "eip155") {
                const hexMsg = "0x" + Array.from(new TextEncoder().encode(nonce)).map(b => b.toString(16).padStart(2, '0')).join('');
                signature = await signClient.request({
                    topic: session.topic,
                    chainId: `eip155:${chainId}`,
                    request: {
                        method: "personal_sign",
                        params: [hexMsg, address]
                    }
                });
            } else if (namespace === "solana") {
                const result = await signClient.request({
                    topic: session.topic,
                    chainId: `solana:${chainId}`,
                    request: {
                        method: "solana_signMessage",
                        params: { message: btoa(nonce), pubkey: address }
                    }
                });
                signature = result.signature; // WC Solana return pattern
            } else if (namespace === "algorand") {
                const result = await signClient.request({
                    topic: session.topic,
                    chainId: `algorand:${chainId}`,
                    request: {
                        method: "algo_signMessage",
                        params: [address, btoa(nonce)]
                    }
                });
                signature = result;
            }

            if (!signature) throw new Error("Signature denied or failed");

            // 3. Send Request to Server for persistent linking
            socket.send(JSON.stringify({
                type: "link_wallet_request",
                payload: {
                    primary_avm_wallet: userAddress,
                    linked_address: address,
                    linked_chain: newWalletInfo.chain,
                    signature: signature,
                    nonce: nonce
                }
            }));

            // Prevent duplicates
            if (!linkedWallets.find(w => w.address.toLowerCase() === newWalletInfo.address.toLowerCase())) {
                linkedWallets.push(newWalletInfo);
                localStorage.setItem("vbabes_linked_wallets", JSON.stringify(linkedWallets));
                showToast(`🔗 Requested link for ${newWalletInfo.chain} wallet...`, "info");
                refreshInventory();
            }
        }

    } catch (err) {
        console.error("X-Chain link failed", err);
    }
}

window.updateLinkedWalletsUI = () => { // This will be moved to wallet.js
    const container = document.getElementById("linked-wallets-list");
    if (!container) return;
    container.innerHTML = "";
    linkedWallets.forEach((w, idx) => {
        const div = document.createElement("div");
        div.className = "player-item";
        div.style.padding = "2px 5px";
        div.style.fontSize = "10px";
        div.innerHTML = `<span>[${w.chain}] ${w.address.substring(0,6)}...</span>
                         <button class="outline" style="padding: 0 4px; color: #ff4b4b; border-color: #ff4b4b;" onclick="removeLinkedWallet(${idx})">×</button>`;
        container.appendChild(div);
    });
}

window.removeLinkedWallet = (index) => { // This will be moved to wallet.js
    linkedWallets.splice(index, 1);
    localStorage.setItem("vbabes_linked_wallets", JSON.stringify(linkedWallets));
    refreshInventory();
}

function openPayoutSettings() {
    document.getElementById("payout-settings-overlay").classList.remove("hidden");
    document.getElementById("payout-address-input").value = payoutAddress || "";
}

window.savePayoutAddress = () => { // This will be moved to wallet.js
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

window.updatePayoutUI = () => { // This will be moved to wallet.js
    const display = document.getElementById("payout-address-display");
    if (display) {
        display.innerText = payoutAddress ? (payoutAddress.substring(0, 6) + "..." + payoutAddress.substring(54)) : "Default Wallet";
    }
}

function updateAdminNetworkUI() {
    const select = document.getElementById("admin-network-select"); // Imported from admin.js
    if (!select) return;

    const currentSelection = select.value;
    select.innerHTML = "";
    
    Object.keys(availableNetworks).forEach(name => {
        const opt = document.createElement("option");
        opt.value = name;
        opt.innerText = name;
        if (name === adminFocusNetwork && !currentSelection) opt.selected = true;
        else if (name === currentSelection) opt.selected = true;
        select.appendChild(opt);
    });

    onAdminNetworkSelectChange();
}

window.onAdminNetworkSelectChange = () => { // This will be moved to admin.js
    const name = document.getElementById("admin-network-select").value;
    const config = availableNetworks[name];
    const details = document.getElementById("admin-network-details");
    if (!details || !config) return;

    details.innerHTML = `
        <div><b>Node:</b> ${config.node_url}</div>
        <div><b>Asset ID:</b> ${config.asset_id}</div>
        <div><b>App ID:</b> ${config.app_id}</div>
        <div style="color: ${name === adminFocusNetwork ? 'var(--neon-green)' : 'inherit'}"><b>Status:</b> ${name === adminFocusNetwork ? 'ACTIVE' : 'Standby'}</div>
    `;

    if (config) {
        document.getElementById("admin-power-divisor").value = config.power_divisor || 1000000;
        document.getElementById("power-divisor-val").innerText = (config.power_divisor || 1000000).toFixed(1);
        document.getElementById("admin-power-base").value = config.power_base || 50;
        document.getElementById("power-base-val").innerText = config.power_base || 50;
    }
}

window.adminSetActiveNetwork = async () => { // This will be moved to admin.js
    const networkName = document.getElementById("admin-network-select").value;
    const headers = await getAdminHeaders();
    if (!headers) return;

    try { // Renamed endpoint for clarity
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/set-admin-focus-network`, {
            method: "POST",
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify({ network_name: networkName })
        }); // Updated toast message
        if (response.ok) showToast(`🌐 Admin focus switched to ${networkName}`, "success");
        fetchLastAdminAction();
    } catch (err) { showToast("❌ Admin focus switch failed", "error"); }
}
async function adminToggleMaintenance(active) {
    const minsInput = document.getElementById("admin-maint-mins"); // Imported from admin.js
    const minutes = parseInt(minsInput.value) || 0;
    const headers = await getAdminHeaders();
    if (!headers) return;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/maintenance-mode`, {
            method: "POST",
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify({ active, minutes })
        });
        if (response.ok) {
            showToast(`🛠️ Maintenance Mode ${active ? 'Activated' : 'Deactivated'}`, "info");
            fetchLastAdminAction();
            if (!active) minsInput.value = "";
        } else {
            const errText = await response.text();
            showToast(`❌ Action failed: ${errText}`, "error");
        }
    } catch (err) { showToast("❌ Server connection error", "error"); }
}

window.adminToggleDevMode = async () => { // This will be moved to admin.js
    const enabled = document.getElementById("dev-mode-toggle").checked;
    // Add a safety check when enabling
    if (enabled && !confirm("⚠️ DEV MODE: This will force a 100% win rate against the bot for reward testing. Enable?")) {
        document.getElementById("dev-mode-toggle").checked = false;
        return;
    }
    window.SetTestingMode(enabled);
    showToast(`🛠️ Dev Mode ${enabled ? 'Enabled' : 'Disabled'}`, enabled ? "success" : "info");
}

window.adminResetStats = async () => { // This will be moved to admin.js
    const wallet = document.getElementById("admin-ban-wallet").value.trim();
    if (!wallet) return;
    if (!confirm(`⚠️ CRITICAL: You are about to PERMANENTLY WIPE all stats for wallet: ${wallet}. This cannot be undone. Proceed?`)) return;

    const headers = await getAdminHeaders();
    if (!headers) return;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/reset-stats`, {
            method: "POST",
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify({ wallet })
        });
        if (response.ok) {
            showToast(`✨ Stats wiped for ${wallet.substring(0,6)}...`, "success");
            fetchLeaderboard();
            fetchLastAdminAction();
        } else {
            const errText = await response.text();
            showToast(`❌ Reset failed: ${errText}`, "error");
        }
    } catch (err) { showToast("❌ Server connection error", "error"); }
}

window.adminSimulateTournament = async () => { // This will be moved to admin.js
    const size = parseInt(document.getElementById("admin-sim-size").value);
    const isBuyIn = document.getElementById("admin-sim-buyin").checked;
    if (isNaN(size) || (size !== 8 && size !== 16)) {
        showToast("❌ Invalid tournament size (must be 8 or 16)", "error");
        return;
    }
    if (!confirm(`Are you sure you want to simulate a ${size}-player tournament? This will overwrite the current tournament state.`)) {
        return;
    }

    const headers = await getAdminHeaders();
    if (!headers) return;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/simulate-tournament`, {
            method: "POST",
            headers: { ...headers, 'Content-Type': 'application/json' },
            body: JSON.stringify({ size, is_buy_in: isBuyIn })
        });
        if (response.ok) showToast(`🏆 Simulating ${size}-player tournament...`, "success");
        fetchLastAdminAction();
    } catch (err) {
        showToast("❌ Simulation failed", "error");
    }
}

window.adminLogTicker = null; // Imported from admin.js
window.startAdminLogPolling = () => { // This will be moved to admin.js
    if (adminLogTicker) return;
    adminLogTicker = setInterval(fetchLastAdminAction, 15000); // Check every 15s for status bar
}

window.fetchLastAdminAction = async () => { // This will be moved to admin.js
    if (!lastAdminKey && !cachedAdminHeaders) { // Imported from admin.js
        document.getElementById("admin-last-action").innerText = "Awaiting first action..."; 
        return; 
    } 
    const headers = lastAdminKey ? { 'X-Admin-Key': lastAdminKey } : cachedAdminHeaders;
    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/admin/logs`, {
            headers: headers
        });
        if (response.ok) {
            const entry = await response.json();
            if (entry.timestamp) {
                const time = entry.timestamp.split('T')[1].substring(0, 8);
                document.getElementById("admin-last-action").innerText = 
                    `[${time}] ${entry.action}: ${entry.target}`;
            } else {
                document.getElementById("admin-last-action").innerText = entry.details || "No logs found";
            }
        }
    } catch (err) {}
}

window.disconnectUserWallet = async () => { // This will be moved to wallet.js
    console.log("[WALLET] Disconnecting...");
    try {
        if (walletProvider === 'walletconnect' && signClient) {
            const sessions = signClient.session.getAll(); // Imported from wallet.js
            if (sessions.length > 0) {
                await signClient.disconnect({
                    topic: sessions[0].topic,
                    reason: { code: 6000, message: "User disconnected" }
                });
            }
        }
        walletProvider = null;
        
        window.disconnectWallet(); // Reset Go Engine
        isVerified = false; // Global variable in app.js
        userAddress = null; // Clear user address
        updateWalletUI(null);
    } catch (err) {
        console.error("Disconnect failed", err);
    }
}
window.highlightStartButton = (isReady) => { // Imported from ui.js
    const btn = document.getElementById("start-btn");
    if (isReady) {
        btn.disabled = false;
        btn.style.boxShadow = "0 0 30px #3fb950";
        btn.innerText = "BATTLE READY - CLICK TO START!";
    } else {
        btn.disabled = true;
        btn.style.boxShadow = "none";
        btn.innerText = "Start Battle (Waiting for Ready)";
    }
};

// --- Matchmaking Logic ---

function toggleMatchmakingQueue() {
    if (!userAddress) { showToast("Connect wallet first", "error"); return; }
    const state = window.GetGameState();
    if (state.deck.length < 5) { showToast("Deck must have 5 cards", "error"); return; }

    if (!inMatchmakingQueue) { // Imported from game.js
        socket.send(JSON.stringify({
            type: "join_queue",
            payload: { 
                deck: state.deck.map(c => c.id),
                deck_rating: state.deck_rating
            }
        }));
        const btn = document.getElementById("btn-matchmaking");
        btn.disabled = true; // This will be re-enabled by handleMatchmakingUpdate
        btn.innerText = "Joining Queue...";
    } else {
        socket.send(JSON.stringify({ type: "leave_queue" }));
        const btn = document.getElementById("btn-matchmaking"); // This will be re-enabled by handleMatchmakingUpdate
        btn.disabled = true;
        btn.innerText = "Leaving Queue...";
    }
}

function handleMatchmakingUpdate(data) {
    const btn = document.getElementById("btn-matchmaking");
    const status = document.getElementById("queue-status");

    if (data.status === "queued") {
        setInMatchmakingQueue(true);
        btn.innerText = "Leave Queue";
        btn.style.background = "var(--neon-purple)";
        status.innerHTML = `<span class="status-active">SEARCHING FOR OPPONENT...</span>`;
        showToast("🛰️ Entered global matchmaking pool.", "info");
        btn.disabled = false; // Re-enable after status update
    } else if (data.status === "idle") {
        setInMatchmakingQueue(false);
        btn.innerText = "Join Matchmaking Pool";
        btn.style.background = "";
        status.innerText = "Ready for automatic pairing?";
        btn.disabled = false; // Re-enable after status update
        showToast("🛰️ Left matchmaking pool.", "info");
    } else if (data.status === "match_found") {
        setInMatchmakingQueue(false);
        btn.innerText = "Join Matchmaking Pool";
        status.innerText = "Ready for automatic pairing?";
        showToast(`⚔️ MATCH FOUND! Battle vs ${data.opponent.substring(0,8)}...`, "success");
        window.SetPhase("Active"); // Optional: logic to transition visual state
        btn.disabled = false; // Re-enable after status update
    }
}

window.sendPing = () => { // This will be moved to network.js
    if (!socket || socket.readyState !== WebSocket.OPEN) return;
    setLastPingTime(Date.now());
    socket.send(JSON.stringify({ type: "ping" }));
}

function updatePlayerList(players) {
    const list = document.getElementById("active-players");
    list.innerHTML = "";
    
    // Check if current user is banned
    const me = players.find(p => p.id === myClientId);
    handleLocalBanUI(me ? me.ban_expires : null); // Imported from ui.js
    const iAmBanned = me && me.ban_expires && new Date(me.ban_expires) > Date.now();

    players.forEach(p => {
        const li = document.createElement("li");
        li.className = "player-item";
        const isMe = p.id === myClientId;
        
        const targetBanned = p.ban_expires && new Date(p.ban_expires) > Date.now();
        const isDisabled = !isMe && (iAmBanned || targetBanned);
        const adminBadge = p.is_admin ? `<span style="color: var(--neon-cyan); font-weight: bold; font-size: 0.8em; margin-left: 5px;">[ADMIN]</span>` : '';
        const btnTitle = targetBanned ? "Player Banned" : (iAmBanned ? "You are Banned" : "Challenge");

        li.innerHTML = `<span>${p.id} ${isMe ? '(You)' : ''} ${adminBadge}</span>
                        <div style="display: flex; gap: 5px;">
                            ${!isMe ? `<button class="outline" style="padding: 5px 10px; font-size: 10px;" ${isDisabled ? 'disabled' : ''} title="${btnTitle}" onclick="sendChallenge('${p.id}')">Challenge</button>` : ''}
                            ${!isMe ? `<button class="outline" style="padding: 5px 10px; font-size: 10px; border-color: var(--neon-purple); color: var(--neon-purple);" onclick="sendSpectate('${p.id}')">Watch</button>` : ''}
                        </div>`;
        list.appendChild(li);
    });
}

window.updateMarketTicker = (players) => { // This will be moved to economy.js
    const spacing = 60;
    let tickerContainer = document.getElementById("market-ticker");
    if (!tickerContainer) {
        tickerContainer = document.createElement("div");
        tickerContainer.id = "market-ticker";
        tickerContainer.className = "market-ticker-container";
        // Performance Refactor: Use Canvas instead of DOM string manipulation
        tickerContainer.innerHTML = `
            <div class="ticker-label">LIVE MARKET:</div>
            <canvas id="market-ticker-canvas" style="flex: 1; height: 30px; cursor: default;"></canvas>
        `;
        document.body.prepend(tickerContainer);

        const canvas = document.getElementById("market-ticker-canvas");
        const resize = () => {
            const dpr = window.devicePixelRatio || 1;
            const rect = canvas.getBoundingClientRect();
            canvas.width = rect.width * dpr;
            canvas.height = 30 * dpr;
            const ctx = canvas.getContext('2d');
            ctx.scale(dpr, dpr);
        };
        window.addEventListener('resize', resize);
        resize();
    }

    // Sort by Wins -> Reputation to find "Top Performers"
    const topPerformers = [...players]
        .sort((a, b) => (b.wins - a.wins) || (b.reputation - a.reputation))
        .slice(0, 5);

    const newItems = [];
    
    // Add Global Market Token
    newItems.push({
        symbol: "MKT TOKEN",
        val: "0.80 $VBV",
        trend: "▲",
        color: "#3fb950" // Neon Green
    });

    topPerformers.forEach(p => {
        const basePrice = (p.wins * 10) + (p.reputation / 2) + 100;
        const volatility = (p.id.charCodeAt(p.id.length - 1) % 5);
        const finalPrice = basePrice + volatility;
        
        newItems.push({
            symbol: getCachedEnvoiName(p.wallet),
            badge: (p.achievements && p.achievements.length > 0) ? "🏆" : "",
            val: finalPrice.toFixed(2),
            trend: (p.wins > 0) ? "▲" : "─",
            color: (p.wins > 0) ? "#3fb950" : "#888"
        });
    });

    // PERFORMANCE OPTIMIZATION: Pre-calculate widths and measure text only when data changes
    const canvas = document.getElementById("market-ticker-canvas");
    const ctx = canvas ? canvas.getContext('2d') : null;
    if (ctx) {
        ctx.font = "bold 12px 'Rajdhani', sans-serif";
        tickerItems = newItems.map(item => {
            const str = `${item.symbol}${item.badge ? ' ' + item.badge : ''} ${item.val} ${item.trend}`;
            item.width = ctx.measureText(str).width + spacing;
            return item;
        });
    } else {
        tickerItems = newItems;
    }

    if (!tickerAnimId) {
        startTickerAnimation();
    }
}

window.startTickerAnimation = () => { // This will be moved to economy.js
    const canvas = document.getElementById("market-ticker-canvas");
    if (!canvas) return;
    const ctx = canvas.getContext('2d');

    const animate = () => {
        if (tickerItems.length === 0) {
            tickerAnimId = requestAnimationFrame(animate);
            return;
        }

        const width = canvas.width / (window.devicePixelRatio || 1);
        const height = 30;
        ctx.clearRect(0, 0, width, height);

        ctx.font = "bold 12px 'Rajdhani', sans-serif";
        ctx.textBaseline = "middle";

        // Optimized total width calculation using cached widths
        const totalContentWidth = tickerItems.reduce((sum, item) => sum + (item.width || 0), 0);
        if (totalContentWidth <= 0) {
            tickerAnimId = requestAnimationFrame(animate);
            return;
        }

        tickerOffset += 0.8; // Scrolling speed
        if (tickerOffset >= totalContentWidth) tickerOffset = 0;

        let x = -tickerOffset;
        while (x < width) {
            for (let i = 0; i < tickerItems.length; i++) {
                const item = tickerItems[i];
                const itemWidth = item.width || 100; // Fallback

                if (x + itemWidth > 0 && x < width) {
                    let curX = x;
                    ctx.fillStyle = "#00f2fe"; // Neon Cyan
                    ctx.fillText(item.symbol, curX, height / 2);
                    curX += ctx.measureText(item.symbol).width;

                    if (item.badge) {
                        ctx.fillStyle = "#ffd700"; // Gold
                        ctx.fillText(" " + item.badge, curX, height / 2);
                        curX += ctx.measureText(" " + item.badge).width;
                    }

                    ctx.fillStyle = "#ffffff";
                    ctx.fillText(" " + item.val, curX, height / 2);
                    curX += ctx.measureText(" " + item.val).width;

                    ctx.fillStyle = item.color;
                    ctx.fillText(" " + item.trend, curX, height / 2);
                }
                x += itemWidth;
            }
            if (totalContentWidth <= 0) break;
        }

        tickerAnimId = requestAnimationFrame(animate);
    };
    tickerAnimId = requestAnimationFrame(animate);
}

window.banTicker = null; // Imported from ui.js
function handleLocalBanUI(banExpires) {
    const container = document.getElementById("local-ban-cooldown");
    const fill = document.getElementById("ban-progress-fill");
    const timer = document.getElementById("ban-countdown-timer");
    
    if (banTicker) clearInterval(banTicker);

    if (!banExpires || new Date(banExpires) <= Date.now()) {
        container.classList.add("hidden");
        return;
    }

    container.classList.remove("hidden");
    const expiry = new Date(banExpires).getTime();
    const totalDuration = 24 * 60 * 60 * 1000; // 24 Hours

    const tick = () => {
        const now = Date.now();
        const remaining = expiry - now;

        if (remaining <= 0) {
            container.classList.add("hidden");
            clearInterval(banTicker);
            return;
        }

        const hours = Math.floor(remaining / (1000 * 60 * 60));
        const minutes = Math.floor((remaining % (1000 * 60 * 60)) / (1000 * 60));
        const seconds = Math.floor((remaining % (1000 * 60)) / 1000);
        timer.innerText = `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;

        const progress = ((totalDuration - remaining) / totalDuration) * 100;
        fill.style.width = `${Math.max(0, Math.min(100, progress))}%`;
    };

    tick();
    banTicker = setInterval(tick, 1000);
}

window.startSeasonTimer = () => { // This will be moved to leaderboard.js
    if (seasonTimerInterval) clearInterval(seasonTimerInterval);
    const timerEl = document.getElementById("season-timer");
    
    const update = () => {
        if (!seasonEnd) return;
        const now = new Date();
        const diff = seasonEnd - now;
        
        if (diff <= 0) {
            timerEl.innerText = "ROLLOVER IMMINENT";
            timerEl.style.color = "var(--neon-green)";
            return;
        }
        
        const days = Math.floor(diff / (1000 * 60 * 60 * 24));
        const hours = Math.floor((diff / (1000 * 60 * 60)) % 24);
        const mins = Math.floor((diff / 1000 / 60) % 60);
        
        timerEl.innerText = `${days}d ${hours}h ${mins}m`;
    };
    
    update();
    seasonTimerInterval = setInterval(update, 60000); // Check once per minute
}

window.maintenanceTicker = null; // Imported from ui.js
function handleMaintenanceUI(active, targetTimestamp) {
    const bar = document.getElementById("maintenance-bar");
    const timerDisplay = document.getElementById("maintenance-timer");

    if (maintenanceTicker) clearInterval(maintenanceTicker);

    // Notify the Go WASM Engine to enforce maintenance guards
    if (window.SetMaintenanceState) window.SetMaintenanceState(active);

    if (!active) {
        bar.classList.add("hidden");
        return;
    }

    bar.classList.remove("hidden");
    const targetTime = new Date(targetTimestamp).getTime();

    const tick = () => {
        const now = Date.now();
        const diff = targetTime - now;

        if (diff <= 0) {
            timerDisplay.innerText = "STARTING NOW";
            return;
        }

        const mins = Math.floor(diff / 60000);
        const secs = Math.floor((diff % 60000) / 1000);
        timerDisplay.innerText = `${String(mins).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
    };

    tick();
    maintenanceTicker = setInterval(tick, 1000);
}

window.sendChatMessage = () => { // This will be moved to game.js
    const input = document.getElementById("chat-input");
    const text = input.value.trim();
    if (!text || !socket) return;

    const envelope = {
        type: "chat",
        payload: { text: text }
    };
    socket.send(JSON.stringify(envelope));
    input.value = "";
}

window.switchHofTab = (tab) => { // This will be moved to leaderboard.js
    document.getElementById("hof-rankings-view").classList.add("hidden");
    document.getElementById("hof-history-view").classList.add("hidden");
    document.getElementById("hof-seasons-view").classList.add("hidden");
    document.getElementById("tab-rankings").classList.remove("active");
    document.getElementById("tab-history").classList.remove("active");
    document.getElementById("tab-seasons").classList.remove("active");

    if (tab === 'rankings') {
        document.getElementById("hof-rankings-view").classList.remove("hidden");
        document.getElementById("tab-rankings").classList.add("active");
        fetchLeaderboard();
    } else if (tab === 'history') {
        document.getElementById("hof-history-view").classList.remove("hidden");
        document.getElementById("tab-history").classList.add("active");
        fetchTournamentHistory(1);
    } else if (tab === 'seasons') {
        // Clear previous filter when opening the tab
        const filterInput = document.getElementById("season-filter-input");
        if (filterInput) filterInput.value = "";
        document.getElementById("hof-seasons-view").classList.remove("hidden");
        document.getElementById("tab-seasons").classList.add("active");
        fetchSeasonHistory();
    }
}

window.toggleTournamentDetails = (id) => { // This will be moved to leaderboard.js
    const details = document.getElementById(`details-${id}`);
    if (!details) return;
    details.classList.toggle("hidden");
}

async function fetchTournamentHistory(page = 1, deepVerify = false) {
    const prevBtn = document.getElementById("prev-tournament-btn");
    const nextBtn = document.getElementById("next-tournament-btn");
    if (prevBtn) prevBtn.disabled = true;
    if (nextBtn) nextBtn.disabled = true;

    const oldPage = currentTournamentPage;
    currentTournamentPage = page;
    const list = document.getElementById("tournament-history-list");
    list.innerHTML = `<div class="chat-msg system">${deepVerify ? 'Executing Deep Reconstruction...' : 'Decrypting archives...'}</div>`;
    
    try {
        const url = `${CONFIG.API_BASE}/api/tournament/history?page=${page}&limit=${tournamentLimit}${deepVerify ? '&deep_verify=true' : ''}`;
        const response = await fetch(url);

        // Handle Indexer 404/General Failure (Bad Gateway from server)
        if (response.status === 502) {
            list.innerHTML = '<div class="chat-msg system" style="color: var(--warning-orange);">⚠️ The Indexer is temporarily unreachable. Historical tournament data cannot be retrieved at this time.</div>';
            if (document.getElementById("tournament-pagination")) document.getElementById("tournament-pagination").classList.add("hidden");
            return;
        }

        if (!response.ok) throw new Error("Server error");
        
        const result = await response.json();
        const history = result.history || [];
        totalTournaments = result.total || 0;
        
        list.innerHTML = "";
        if (history.length === 0) {
            list.innerHTML = '<div class="chat-msg system">No recorded events found in the database.</div>';
            document.getElementById("tournament-pagination").classList.add("hidden");
            return;
        }

        document.getElementById("tournament-pagination").classList.remove("hidden");
        updateTournamentPaginationUI();

        // Batch resolve Envoi names for all participants in the history page
        const participants = new Set();
        history.forEach(t => {
            participants.add(t.winner);
            t.matches.forEach(m => {
                if (m.p1) participants.add(m.p1);
                if (m.p2) participants.add(m.p2);
            });
        });
        await Promise.all(Array.from(participants).filter(p => p && p !== "TBD").map(p => resolveEnvoiName(p)));

        // API is already sorted newest first
        history.forEach(t => {
            const div = document.createElement("div");
            div.className = "glass-panel";
            div.style.margin = "0";
            div.style.borderColor = "var(--neon-purple)";
            
            const date = new Date(t.timestamp).toLocaleDateString();
            const time = new Date(t.timestamp).toLocaleTimeString();
            
            const verifyBtn = !t.is_verified ? `
                <div style="margin-top: 10px;">
                    <button class="outline" style="font-size: 10px; padding: 4px 12px; border-color: #ffa657; color: #ffa657;" 
                            onclick="event.stopPropagation(); fetchTournamentHistory(${currentTournamentPage}, true);">
                        🔍 DEEP VERIFY DATA
                    </button>
                </div>
            ` : '';

            div.innerHTML = `
                <div style="cursor: pointer;" onclick="toggleTournamentDetails('${t.id}')">
                    <div style="display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid var(--glass-border); padding-bottom: 10px; margin-bottom: 15px;">
                    <div style="display: flex; align-items: center;">
                        <span style="font-weight: bold; color: var(--neon-purple); letter-spacing: 1px;">${t.id}</span>
                        ${window.GetTournamentArchiveBadge ? window.GetTournamentArchiveBadge(t.is_verified, t.links) : ''}
                    </div>
                    <span style="color: var(--neon-cyan); font-weight: bold;">POT: ${t.pot.toFixed(1)} $VBV</span>
                </div>
                <div style="display: flex; align-items: center; gap: 20px; justify-content: center; margin: 20px 0;">
                     <div style="text-align: center;">
                        <div style="font-size: 0.8em; opacity: 0.6; margin-bottom: 5px;">TOURNAMENT CHAMPION</div>
                        <div style="font-size: 1.5em; font-weight: bold; color: var(--neon-green); text-shadow: 0 0 10px var(--neon-green);">${getCachedEnvoiName(t.winner)}</div>
                        ${verifyBtn}
                     </div>
                </div>
                <div style="display: flex; justify-content: space-between; font-size: 0.8em; opacity: 0.5;">
                    <span>Matches: ${t.matches.length}</span>
                    <span>Archived: ${date} ${time}</span>
                </div>
                </div>
				<div id="details-${t.id}" class="hidden" style="margin-top: 20px; padding-top: 20px; border-top: 1px dashed var(--glass-border); display: flex; gap: 30px; overflow-x: auto; padding-bottom: 15px; -webkit-overflow-scrolling: touch; scrollbar-width: thin;">
                    ${generateBracketHTML(t.matches, -1)}
                </div>
            `;
            list.appendChild(div);
        });
    } catch (err) {
        currentTournamentPage = oldPage;
        updateTournamentPaginationUI();
        list.innerHTML = '<div class="chat-msg system" style="color: #ff4b4b;">Database Error: Could not retrieve archives.</div>';
    }
}

window.filterSeasonHistory = () => { // This will be moved to leaderboard.js
    const input = document.getElementById("season-filter-input");
    const val = input.value.trim();
    fetchSeasonHistory(val ? parseInt(val) : null);
}

async function fetchSeasonHistory(seasonNum = null) {
    const list = document.getElementById("season-history-list");
    list.innerHTML = `<div class="chat-msg system">Consulting the Oracle for past epochs...</div>`;
    
    try {
        const url = seasonNum ? `${CONFIG.API_BASE}/api/season/history?season=${seasonNum}` : `${CONFIG.API_BASE}/api/season/history`;
        const response = await fetch(url);
        
        // Handle Indexer 404 (Bad Gateway from server) when reward asset is not found
        if (response.status === 502) {
            list.innerHTML = '<div class="chat-msg system" style="color: var(--warning-orange);">⚠️ The Indexer is currently unable to locate the reward asset history. It may be initializing or the asset ID is invalid.</div>';
            return;
        }

        if (!response.ok) throw new Error("Server error");
        
        const history = await response.json();
        list.innerHTML = "";

        if (history.length === 0) {
            const msg = seasonNum ? `No records found for Season ${seasonNum}.` : "The history books are currently empty. Check back after the next rollover!";
            list.innerHTML = `<div class="chat-msg system">${msg}</div>`;
            return;
        }

        for (const s of history) {
            const div = document.createElement("div");
            div.className = "glass-panel";
            div.style.margin = "0";
            div.style.borderColor = "var(--neon-cyan)";
            
            const startDate = new Date(s.start).toLocaleDateString();
            const endDate = new Date(s.end).toLocaleDateString();

            let winnersHTML = "";
            for (let i = 0; i < s.top.length; i++) {
                const entry = s.top[i];
                const name = await resolveEnvoiName(entry.w);
                winnersHTML += `
                    <div class="leaderboard-row season-winner-row">
                        <span class="rank-badge" style="color: ${i === 0 ? 'var(--neon-green)' : 'inherit'}">#${i+1}</span>
                        <span class="player-name">${name}</span>
                        <span class="player-stats"><b>${entry.v}</b> Wins | <small>${entry.r}</small></span>
                    </div>
                `;
            }

            div.innerHTML = `
                <div class="season-card-header">
                    <span style="font-weight: bold; color: var(--neon-cyan); letter-spacing: 2px;">SEASON ${s.season}</span>
                    <span style="font-size: 0.8em; opacity: 0.6;">${startDate} — ${endDate}</span>
                </div>
                <div class="season-performers-label">TOP PERFORMERS</div>
                <div class="season-winners-list">
                    ${winnersHTML}
                </div>
            `;
            list.appendChild(div);
        }
    } catch (err) {
        console.error("[SEASON HISTORY] Fetch failed:", err);
        list.innerHTML = '<div class="chat-msg system" style="color: #ff4b4b;">Indexer connection failed. Archives are temporarily unavailable.</div>';
    }
}

window.handleChatKey = (e) => { // This will be moved to game.js
    if (e.key === 'Enter') sendChatMessage();
}

function renderChatMessage(sender, text) {
    const display = document.getElementById("chat-display");
    const msgDiv = document.createElement("div");
    msgDiv.className = "chat-msg";
    
    if (sender === "SERVER") msgDiv.classList.add("system");
    
    msgDiv.innerHTML = `<b>${sender}:</b> ${text}`;
    display.appendChild(msgDiv);
    
    // Auto-scroll to bottom
    display.scrollTop = display.scrollHeight;
}

window.saveMatchResult = async (state) => { // This will be moved to game.js
    const history = JSON.parse(localStorage.getItem("vbabes_history") || "[]");
    const opponent = currentOpponentId || (state.multiplayer ? "Unknown Human" : "Vbabe Bot");
    
    const newEntry = {
        winner: state.winner,
        scores: state.scores,
        opponent: opponent,
        timestamp: new Date().toLocaleString()
    };

    history.unshift(newEntry);
    if (history.length > 10) history.pop(); // Keep last 10 matches
    localStorage.setItem("vbabes_history", JSON.stringify(history));
    await renderMatchHistory();
}

window.renderMatchHistory = async () => { // This will be moved to game.js
    const history = JSON.parse(localStorage.getItem("vbabes_history") || "[]");
    const display = document.getElementById("history-display");
    if (!display || history.length === 0) return;
    
    // Batch resolve names for wallets in local history
    const wallets = history.map(e => e.opponent).filter(o => o && o.length > 50);
    await Promise.all(wallets.map(w => resolveEnvoiName(w)));
    
    display.innerHTML = "";
    history.forEach(entry => {
        const div = document.createElement("div");
        div.className = "chat-msg";
        const colors = ["var(--neon-green)", "#ff4b4b", "var(--neon-cyan)"]; // Win, Loss, Draw
        const labels = ["WIN", "LOSS", "DRAW"];
        const color = colors[entry.winner] || "white";
        const label = labels[entry.winner] || "END";

        const opponentDisplay = getCachedEnvoiName(entry.opponent);

        div.innerHTML = `<span style="color: ${color}; font-weight: bold;">${label}</span> vs ${opponentDisplay} <br/> 
                         <small style="opacity: 0.7;">${entry.scores[0]}-${entry.scores[1]} | ${entry.timestamp}</small>`;
        display.appendChild(div);
    });
}

window.showToast = (message, type = 'info', duration = 5000) => { // This will be moved to ui.js
    const container = document.getElementById("toast-container");
    const toast = document.createElement("div");
    toast.className = `toast ${type}`;
    toast.innerHTML = message;
    container.appendChild(toast);

    if (duration > 0) {
        setTimeout(() => {
            toast.style.opacity = '0';
            toast.style.transform = 'translateX(100%)';
            toast.style.transition = '0.5s';
            setTimeout(() => toast.remove(), 500);
        }, duration);
    }
}

// Bridge for Go to trigger Payout UI flow
window.processRewardPayout = async (payloadStr) => { // Imported from wallet.js
    const payload = JSON.parse(payloadStr);
    showToast("🛰️ Requesting secure nonce from server...", "info");

    try {
        // 1. Request Nonce from Server via WebSocket (Anti-Replay)
        const nonce = await new Promise((resolve, reject) => {
            const timeout = setTimeout(() => reject(new Error("Nonce request timed out")), 10000);
            setNonceResolver((n) => { clearTimeout(timeout); resolve(n); });
            socket.send(JSON.stringify({ type: "nonce_request" }));
            setTransactionStatus("Requesting secure nonce...", "info");
        });

        showToast("🔐 Please sign the verification request in your wallet...", "info");

        // 2. Construct 0-amount "Reverse Sign" Transaction
        const tx = {
            from: userAddress,
            to: userAddress,
            amount: 0,
            note: new TextEncoder().encode(nonce),
            type: 'pay'
        };

        setTransactionStatus("Signing verification request...", "info");
        // 3. Request Signature via active provider
        let signedTx = null;
        
        // The payload identifies the winner, but the faucet pays the specified recipient.
        payload.claimant = payload.recipient; // The authenticated playing address
        payload.recipient = payoutAddress || payload.recipient; // The Voi payout target

        const isEVM = payload.claimant.startsWith("0x");

        if (isEVM) {
            if (walletProvider === 'walletconnect' && signClient) {
                const sessions = signClient.session.getAll();
                if (sessions.length === 0) throw new Error("No active WalletConnect session found for EVM");
                
                // Encode nonce as hex for EVM personal_sign
                const msgBuffer = new TextEncoder().encode(nonce);
                const hexMsg = "0x" + Array.from(msgBuffer).map(b => b.toString(16).padStart(2, '0')).join('');
                
                const signature = await signClient.request({
                    topic: sessions[0].topic,
                    chainId: "eip155:1", // Default to mainnet for verification
                    request: {
                        method: "personal_sign",
                        params: [hexMsg, payload.claimant]
                    }
                });
                if (!signature) throw new Error("Signature denied.");
                signedTx = new TextEncoder().encode(signature);
            } else {
                throw new Error("EVM verification requires WalletConnect.");
            }
        } else if (walletProvider === 'nautilus') {
            const result = await window.algo.signTxn([
                { txn: algosdk.encodeObj(tx), signers: [payload.claimant] }
            ]);
            signedTx = result[0];
        } else if (walletProvider === 'kibisis') {
            const txnB64 = btoa(String.fromCharCode(...algosdk.encodeObj(tx)));
            const result = await window.kibisis.signTxns([{ txn: txnB64 }]);
            // Kibisis returns base64 strings
            signedTx = new Uint8Array(atob(result[0]).split("").map(c => c.charCodeAt(0)));
        } else if (walletProvider === 'walletconnect' && signClient) {
            const sessions = signClient.session.getAll();
            const txnB64 = btoa(String.fromCharCode(...algosdk.encodeObj(tx)));
            const response = await signClient.request({
                topic: sessions[0].topic,
                chainId: (window.GetGameState().network === "VOI" ? CONFIG.VOI_CHAIN_ID : CONFIG.ALGO_CHAIN_ID),
                request: {
                    method: "algo_signTxn",
                    params: [[{ txn: txnB64, signers: [payload.claimant] }]]
                }
            });
            signedTx = new Uint8Array(atob(response[0]).split("").map(c => c.charCodeAt(0)));
        }

        if (!signedTx) throw new Error("Signature cancelled or provider error");

        payload.signed_tx = signedTx;
        setTransactionStatus("Submitting proof to Switchboard...", "info");
        showToast("🛰️ Submitting proof to Switchboard...", "info");

        const response = await fetch(`${CONFIG.API_BASE}/api/reward`, {
            method: "POST",
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        
        if (!response.ok) {
            const errorMsg = await response.text();
            showToast(`❌ Payout Error: ${errorMsg}`, "error");
            setTransactionStatus("Payout Failed!", "error");
            return;
        }

        const data = await response.json();
        
        let successMsg = `✅ Reward Sent! TxID: ${data.txid.substring(0, 14)}...`;
        if (data.bonus_applied) {
            successMsg += `<br><span style="color: var(--neon-cyan); font-weight: bold; font-size: 0.85em; text-shadow: 0 0 8px var(--neon-cyan);">[ REPUTATION BONUS ACTIVE ⚡ ]</span>`;
        }
        
        // Handle skipped assets UI
        if (data.skipped_assets && data.skipped_assets.length > 0) {
            const symbols = [];
            for (const id of data.skipped_assets) {
                // Use resolveAssetSymbol to populate cache then pull symbol
                const sym = await resolveAssetSymbol(id);
                symbols.push(sym);
            }
            successMsg += `<br><small style="color: #ffa657; font-size: 0.8em; font-style: italic;">⚠️ Some rewards skipped (Low Pool): ${symbols.join(", ")}</small>`;
        }

        showToast(successMsg, "success", 8000);
        setTransactionStatus("Reward Sent! Confirming...", "success");
        updateWalletUI(userAddress);
        syncUI(); // Trigger UI sync to show updated balance
    } catch (err) {
        showToast("⚠️ Payout Failed: " + err.message, "error");
        setTransactionStatus("Payout Failed!", "error");
    } finally {
        setTimeout(() => setTransactionStatus(null), 3000);
    }
};

window.showChallengeNotification = (challengerId) => { // This will be moved to game.js
    currentChallengerId = challengerId;
    const challengeOverlay = document.getElementById("challenge-overlay");
    const challengeText = document.getElementById("challenge-text");

    challengeText.innerText = `${challengerId}`;
    challengeOverlay.classList.remove("hidden");
    // Optionally play a sound or vibrate
}

window.acceptChallenge = () => { // This will be moved to game.js
    if (!socket || !currentChallengerId) return;
    const state = window.GetGameState();
    const envelope = {
        type: "challenge",
        to_id: currentChallengerId,
            from_id: myClientId, // Ensure from_id is set for server
        payload: { 
            action: "accept",
            to_id: currentChallengerId,
            deck: state.deck.map(c => c.id),
            avatar: state.p1_avatar,
            gloat: state.p1_gloat,
            rules: state.rules
        }
    };

    socket.send(JSON.stringify(envelope));
    document.getElementById("challenge-overlay").classList.add("hidden");
    currentChallengerId = null;
}

window.sendMatchSync = (targetId) => { // This will be moved to game.js
    const state = window.GetGameState();
    const envelope = {
        type: "challenge",
        to_id: targetId,
        from_id: myClientId, // Ensure from_id is set for server
        payload: { 
            action: "sync_back", 
            deck: state.deck.map(c => c.id),
            avatar: state.p1_avatar,
            gloat: state.p1_gloat
        }
    };
    socket.send(JSON.stringify(envelope));
}

window.reportGloat = (opponentClientId, gloatText) => { // This will be moved to game.js
    if (!socket || socket.readyState !== WebSocket.OPEN) {
        showToast("Cannot report: Not connected to server.", "error");
        return;
    }
    if (!confirm("Are you sure you want to report this gloat message as offensive?")) {
        return;
    }

    const envelope = {
        type: "report_gloat",
        payload: {
            opponent_client_id: opponentClientId,
            gloat_text: gloatText
        }
    };
    socket.send(JSON.stringify(envelope));
    showToast("Gloat message reported. Thank you for helping keep the arena clean!", "success");
}

window.declineChallenge = () => { // This will be moved to game.js
    if (!socket || !currentChallengerId) return;

    const envelope = {
        type: "challenge",
        from_id: myClientId, // Ensure from_id is set for server
        to_id: currentChallengerId,
        payload: { action: "decline" }
    };

    socket.send(JSON.stringify(envelope));
    document.getElementById("challenge-overlay").classList.add("hidden");
    currentChallengerId = null;
}

window.sendSpectate = (targetId) => { // This will be moved to game.js
    if (!socket || socket.readyState !== WebSocket.OPEN) return;

    const envelope = {
        type: "spectate",
        from_id: myClientId, // Ensure from_id is set for server
        payload: { target_id: targetId }
    };
    spectatorMatchState = null; // Reset for new spectate session

    socket.send(JSON.stringify(envelope));
    showToast(`👁️ Requesting access to stream...`, "info");
}

window.showMatchPreview = (data) => { // This will be moved to ui.js
    document.getElementById("preview-p1-id").innerText = data.p1_id;
    document.getElementById("preview-p1-rating").innerText = data.p1_rating || "[Z]";
    document.getElementById("preview-p2-id").innerText = data.p2_id;
    document.getElementById("preview-p2-rating").innerText = data.p2_rating || "[Z]";
    
    document.getElementById("match-preview-overlay").classList.remove("hidden");
}

window.proceedToWarRoom = () => { // This will be moved to game.js
    if (!spectatorMatchState) return;
    
    document.getElementById("match-preview-overlay").classList.add("hidden");
    window.ResetGame();
    window.SetBoardState(spectatorMatchState);
    window.ForceActive();
    syncUI("all");
}

window.sendChallenge = (targetId) => { // This will be moved to game.js
    if (!socket || socket.readyState !== WebSocket.OPEN) return;

    const state = window.GetGameState();
    const envelope = {
        type: "challenge",
        from_id: myClientId, // Ensure from_id is set for server
        to_id: targetId,
        payload: { 
            action: "invite",
            avatar: state.p1_avatar || "",
            gloat: state.p1_gloat || "",
            deck: state.deck.map(c => c.id)
        }
    };

    socket.send(JSON.stringify(envelope));
    alert(`Challenge sent to ${targetId}`);
}

window.triggerToggleNetwork = () => { // This will be moved to game.js
    window.toggleNetwork();
    syncUI();
}

window.selectCard = (id) => { // This will be moved to game.js
    activeCardId = id; // Global variable in app.js
    if (window.PlaySelectSound) window.PlaySelectSound();
    syncUI("inventory"); // Re-render to show the selected card glowing
}

function clickGrid(index) {
    const state = window.GetGameState();
    
    // Multiplayer Guard: Only allow move if it's actually our turn
    if (state.phase === "Active" && state.turn !== state.local_player_index) {
        console.warn("It is not your turn!"); // Imported from game.js
        return;
    }

    if (activeCardId === null) {
        return;
    }
    
    const selectedCardId = activeCardId;

    // Execute locally
    const success = window.PlaceCard(index, activeCardId); // This will be moved to game.js
    if (success) {
        // If in multiplayer, broadcast the move to the opponent
        if (state.phase === "Active" && currentOpponentId) {
            // Find card power for server verification
            const card = state.deck.find(c => c.id === selectedCardId);
            const envelope = {
                type: "move",
                to_id: currentOpponentId, // This should be the opponent's client ID
                payload: {
                    grid_index: index,
                    card_id: selectedCardId,
                    power: card ? card.power : [0,0,0,0]
                }
            };
            socket.send(JSON.stringify(envelope));
        }
        activeCardId = null; 
        syncUI("combat");
    }
} 

// --- Deck Manager Logic --- // These will be moved to deck.js
window.openDeckManager = () => {
    document.getElementById("deck-manager-overlay").classList.remove("hidden");
    renderDeckManager();
}

window.closeDeckManager = () => {
    document.getElementById("deck-manager-overlay").classList.add("hidden"); // Imported from deck.js
    
    // TACTICAL SYNC: Report the highest possible deck rating to the Hall of Fame
    const state = window.GetGameState();
    const rating = calculateDeckRating(state.deck);
    if (socket && socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify({
            type: "update_rating",
            payload: { best_rating: rating }
        }));
    }
    syncUI("all");
}

window.renderDeckManager = () => { // This will be moved to deck.js
    const state = window.GetGameState();
    const invGrid = document.getElementById("inventory-grid");
    const deckZone = document.getElementById("deck-drop-zone");
    const selector = document.getElementById("deck-selector-bar");
    const deckRatingEl = document.getElementById("deck-stat-summary"); // Target summary area
    const atkEl = document.getElementById("total-atk");
    const defEl = document.getElementById("total-def");

    invGrid.innerHTML = "";
    deckZone.innerHTML = "";
    selector.innerHTML = "";
    const isMobile = window.innerWidth <= 768;

    let totalAtk = 0;
    let totalDef = 0;

    // 1. Render Inventory
    state.inventory.forEach(card => {
        const cardEl = document.createElement("div");
        const isSelected = activeCardId === card.id;
        cardEl.className = `card-mini ${isSelected ? 'selected-item' : ''}`;
        cardEl.draggable = true;
        cardEl.innerHTML = renderCardHTML(card);
        cardEl.ondragstart = (e) => e.dataTransfer.setData("cardID", card.id);
        
        // Mobile Fallback: Tap to select
        cardEl.onclick = () => {
            activeCardId = card.id;
            renderDeckManager();
            if (window.PlaySelectSound) window.PlaySelectSound();
        };

        invGrid.appendChild(cardEl);
    });

    // 2. Render Active Deck
    state.deck.forEach((card, idx) => {
        const cardEl = document.createElement("div");
        cardEl.className = "card-mini";
        cardEl.style.width = "100%";
        cardEl.style.height = "60px";
        cardEl.innerHTML = `<span style="font-size: 10px;">${card.name}</span><button onclick="window.RemoveFromDeck(${idx}); renderDeckManager();" style="float: right; padding: 2px 5px; font-size: 9px;">X</button>`;
        
        // Calculate Stats: Attack (Top + Right), Defense (Bottom + Left)
        totalAtk += (card.power[0] + card.power[1]);
        totalDef += (card.power[2] + card.power[3]);
        
        deckZone.appendChild(cardEl);
    });

    // Mobile Fallback: Tap zone to place primed card
    deckZone.onclick = () => {
        if (activeCardId !== null) {
            window.AddToDeck(activeCardId);
            activeCardId = null;
            renderDeckManager();
        }
    };

    atkEl.innerText = totalAtk;
    defEl.innerText = totalDef;

    // 3. Render Deck Selectors (Unlocks)
    const thresholds = [0, 250, 600, 1000];
    for(let i=0; i<4; i++) {
        const btn = document.createElement("button");
        const isLocked = state.reputation < thresholds[i];
        btn.className = `deck-slot-btn ${i === state.active_deck ? 'active' : ''} ${isLocked ? 'locked' : ''}`;
        btn.innerText = isLocked ? `🔒 ${thresholds[i]} REP` : `Deck ${i+1}`;
        btn.onclick = () => { if(!isLocked) { window.SelectDeck(i); renderDeckManager(); } };
        selector.appendChild(btn);
    }
}

// Initialize Drag & Drop
const dropZone = document.getElementById("deck-drop-zone"); // Imported from deck.js
dropZone.ondragover = (e) => { e.preventDefault(); dropZone.classList.add("drag-over"); };
dropZone.ondragleave = () => dropZone.classList.remove("drag-over");
dropZone.ondrop = (e) => {
    e.preventDefault();
    dropZone.classList.remove("drag-over");
    const id = parseInt(e.dataTransfer.getData("cardID"));
    window.AddToDeck(id);
    renderDeckManager();
};

// 4. THE RENDER LOOP (The Camera fetching Go State)
window.syncUI = async (scope = "all") => { // This will be the main sync function in app.js
    if (!window.GetGameState) return; // Ensure Go function exists
    // Partial state sync: only update sections present in the current state snapshot
    const state = window.GetGameState(scope);
    
    // --- Update Dynamic Environment ---
    if (state.phase !== undefined || state.multiplayer !== undefined || state.tournament !== undefined) {
        updateDynamicArenaFloor(state);
    }

	// DYNAMIC MOOD SHIFT: Adjust global accents based on game state
	if (state.phase === "Active") {
		const isCriminal = (state.p1_wanted_level || 0) > 10;
		document.documentElement.style.setProperty('--neon-accent', isCriminal ? '#ff0844' : '#00f2fe');
		document.body.classList.toggle('criminal-activity', isCriminal);
	} else {
		document.documentElement.style.setProperty('--neon-accent', '#00f2fe');
	}

    // Update Deck Rating in UI
    if (state.deck_rating !== undefined) { // Imported from ui.js
        document.getElementById("deck-rating-display").innerText = state.deck_rating;
    }

    // Update Mojo Display
    const mojoEl = document.getElementById("mojo-display"); // Assuming this element exists in index.html
    if (mojoEl && state.mojo !== undefined) mojoEl.innerHTML = `MOJO: ${state.mojo || 0} [${state.social_rank || 'Nobody'}] <span style="font-size: 0.7em; opacity: 0.7; margin-left: 10px;">RUMORS: ${state.rumor_count || 0}</span>`; // Imported from ui.js

    // 0. Resolve missing reward symbols concurrently to prevent UI flickering
    if (state.rewards) { // This will be moved to economy.js or utils.js
        const rewardIds = Object.keys(state.rewards || {});
        const missingSymbols = rewardIds.filter(id => !assetCache[id]);
        if (missingSymbols.length > 0) {
            await Promise.all(missingSymbols.map(id => resolveAssetSymbol(id)));
        }
    }
    
    // --- Update Dashboard ---
    // Overlay Management
    if (state.phase !== undefined || state.show_leaderboard !== undefined) { // Imported from ui.js
        hideAllOverlays();
        const mainContainer = document.getElementById("main-game-container");
        mainContainer.classList.add('hidden'); // Hide main game by default

        if (state.show_leaderboard) {
            document.getElementById("leaderboard-overlay").classList.remove("hidden");
        } else if (state.phase === "TournamentLobby") {
            document.getElementById("tournament-overlay").classList.remove("hidden");
            // Populate bracket visualization if data exists
            if (state.tournament) {
                await renderTournamentBracket(state.tournament);
            }
        } else if (state.phase === "Setup" && userAddress) {
            document.getElementById("setup-overlay").classList.remove("hidden");
        } else if (!userAddress) {
            // If no wallet connected, show wallet selector
            document.getElementById("wallet-selector-overlay").classList.remove("hidden");
            renderRumorBoard(); // Ensure rumor board is rendered even if no wallet is connected
        } else {
            // Default to showing main game container if no specific overlay is needed
            mainContainer.classList.remove('hidden');
        }
    }

    // --- Narrative Intelligence Hook & AI Indicator ---
    if (state.phase === "Active" && !state.multiplayer) { // Imported from game.js
        // 1. Show thinking indicator on AI turn
        if (state.turn === 1) {
            document.getElementById("ai-thinking-indicator").classList.remove("hidden");
        } else {
            document.getElementById("ai-thinking-indicator").classList.add("hidden");
        }

        // 2. Trigger taunt on phase entry or turn change
        if (state.phase !== lastTauntPhase || state.turn !== lastTauntTurn) {
            if (state.playstyle) {
                const npcName = state.p2_id || "Bot";
                const taunt = collectiveIntelligence.generatePlaystyleTaunt(npcName, state.playstyle);
                if (taunt) renderChatMessage("SYSTEM", taunt);
            }
            lastTauntPhase = state.phase;
            lastTauntTurn = state.turn;
        }
    } else {
        document.getElementById("ai-thinking-indicator")?.classList.add("hidden");
        lastTauntPhase = state.phase;
        lastTauntTurn = null;
    }

    // --- Winner Overlay: Character-Aware Feedback ---
    if (state.phase === "Finished" && state.winner !== undefined) { // Imported from ui.js
        const overlay = document.getElementById("winner-overlay");
        const winText = document.getElementById("winner-text");
        const scoreText = document.getElementById("score-text");

        if (overlay) overlay.classList.remove("hidden");

        if (winText && scoreText) {
            let title = "MATCH OVER";
            let gloat = "";
            
            const localPIdx = state.local_player_index !== undefined ? state.local_player_index : myPlayerIndex;
            const isWinner = state.winner === localPIdx;
            const isDraw = state.winner === 2;

            if (isDraw) {
                title = "DRAW";
                winText.style.color = "var(--neon-cyan)";
                gloat = "Perfect balance. Neither side could find the opening.";
            } else if (isWinner) {
                title = "VICTORY";
                winText.style.color = "var(--neon-green)";
                const winnerGloat = (localPIdx === 0) ? state.p1_gloat : state.p2_gloat;
                const defaultGloat = state.multiplayer ? "Victory achieved in combat." : "The Arena recognizes your dominance.";
                gloat = (state.multiplayer && winnerGloat) ? winnerGloat : defaultGloat;
            } else {
                title = "DEFEAT";
                winText.style.color = "#ff4b4b";
                const opponentGloat = (localPIdx === 0) ? state.p2_gloat : state.p1_gloat;

                if (state.multiplayer) {
                    const rawGloat = opponentGloat || "Your opponent has prevailed.";
                    gloat = rawGloat + `<span class="report-gloat-icon" onclick="reportGloat('${currentOpponentId}', '${rawGloat.replace(/'/g, "\\'")}')" title="Report offensive gloat"> 🚨</span>`;
                } else {
                    // Archetype gloats based on SpecialFanfare assigned in main.go
                    switch (state.special_fanfare) {
                        case "Witch": gloat = "A charming attempt, but your luck has run out! Hexed!"; break;
                        case "Boss": gloat = "Calculated. Efficient. You were never a variable in my success."; break;
                        case "Lady": gloat = "Don't look so sad, darling. You simply weren't a match for me."; break;
                        case "cute": gloat = "Tee-hee! I won! You're still my favorite person to play with though!"; break;
                        default: gloat = "The Vbabe Bot has outplayed you this time.";
                    }
                }
            }

            winText.innerText = title;
            scoreText.innerHTML = `${state.scores[0]} - ${state.scores[1]}<br/><span style="font-size: 0.5em; opacity: 0.8; letter-spacing: 2px; display: block; margin-top: 15px; color: #fff; font-family: 'Rajdhani', sans-serif; text-transform: uppercase;">"${gloat}"</span>`;
        }
    }

    // Challenge Overlay (if active)
    if (currentChallengerId) { // Imported from game.js
        document.getElementById("challenge-overlay").classList.remove("hidden");
    }

    if (state.faucet !== undefined) {
        const faucetEl = document.getElementById("faucet-display"); // Imported from economy.js
        const faucetValue = state.faucet.toFixed(2); // Imported from economy.js
        const currentHTML = faucetEl.innerHTML;
        let newHTML = "";
        
        if (state.faucet < 50) {
            newHTML = `${faucetValue} $VBV <span style="font-size: 0.7em; margin-left: 5px;">[ VAULT LOW ]</span>`;
            faucetEl.classList.add("faucet-depleted");
        } else {
            newHTML = faucetValue + " $VBV";
            faucetEl.classList.remove("faucet-depleted");
        }
        
        if (currentHTML !== newHTML) faucetEl.innerHTML = newHTML;
    }

    // --- Update Rewards Dashboard (Economy Scope) ---
    if (state.rewards !== undefined) { // Imported from economy.js
        const rewardsDashboard = document.getElementById("rewards-dashboard");
        if (rewardsDashboard) {
            let totalValue = 0;
            let rewardItems = [];
            Object.entries(state.rewards || {}).forEach(([id, amt]) => {
                totalValue += amt;
                const symbol = getAssetSymbol(id);
                rewardItems.push(`<span style="color: var(--neon-green)">${amt.toFixed(1)}</span> <small>${symbol}</small>`);
            });
            const playerRewards = state.rewards[CONFIG.VBV_ASSET_ID] || 0;
            const rumorCost = 500;
            const myJailedCards = state.jailed_cards || {};
            const myKidnappedCards = state.kidnapped_cards || {};
            const myHeldHostageCards = state.held_hostage_cards || {};
            const wantedVal = state.wanted_level || 0;
            const cunningVal = state.cunning || 0;
            const nurturingVal = state.nurturing || 0; // Get nurturing value
            const jobRole = state.job_role || "";
            const outlawsInLobby = lastLobbyPlayers.filter(p => (p.wanted_level || 0) >= 10);

            const courthouseBtn = wantedVal > 0 ? ` <button class="outline" style="padding: 2px 8px; font-size: 10px; margin-left: 10px; border-color: #ff4b4b; color: #ff4b4b;" onclick="openCourthouse()">⚖️ COURTHOUSE (${wantedVal})</button>` : '';
            const blackMarketBtn = (wantedVal >= 5 && cunningVal >= 10) ? ` <button class="outline" style="padding: 2px 8px; font-size: 10px; margin-left: 10px; border-color: #ff4b4b; color: #ff4b4b;" onclick="openBlackMarket()">🏴‍☠️ BLACK MARKET</button>` : '';
            const rumorMillBtn = (playerRewards >= rumorCost) ? ` <button class="outline" style="padding: 2px 8px; font-size: 10px; margin-left: 10px; border-color: var(--neon-green); color: var(--neon-green);" onclick="openRumorMill()">📢 RUMOR MILL</button>` : '';
            const securityBtn = (jobRole === "Security") ? ` <button class="outline" style="padding: 2px 8px; font-size: 10px; margin-left: 10px; border-color: var(--neon-cyan); color: var(--neon-cyan);" onclick="openSecuritySentry()">🛡️ SECURITY SENTRY</button>` : '';
            const bountyBoardBtn = (outlawsInLobby.length > 0 || wantedVal <= 2) ? ` <button class="outline" style="padding: 2px 8px; font-size: 10px; margin-left: 10px; border-color: #ffd700; color: #ffd700;" onclick="openBountyBoard()">🎯 BOUNTY BOARD (${outlawsInLobby.length})</button>` : '';
            const leaseBoardBtn = ` <button class="outline" style="padding: 2px 8px; font-size: 10px; margin-left: 10px; border-color: var(--neon-purple); color: var(--neon-purple);" onclick="openClubLeaseBoard()">📜 LEASE BOARD</button>`;
			const socialHubBtn = ` <button class="outline" style="padding: 2px 8px; font-size: 10px; margin-left: 10px; border-color: var(--neon-blue); color: var(--neon-blue);" onclick="openSocialPanelOverlay()">👥 SOCIAL HUB</button>`;
			const galleryBtn = ` <button class="outline" style="padding: 2px 8px; font-size: 10px; margin-left: 10px; border-color: var(--neon-cyan); color: var(--neon-cyan);" onclick="openArtGalleryOverlay()">🎨 ART GALLERY</button>`;

            const newHTML = `Win Total: <b style="color: var(--neon-green); text-shadow: 0 0 10px var(--neon-green);">${totalValue.toFixed(1)}</b> | ` + rewardItems.join(" + ") +
                ` <span style="margin-left: 10px; color: var(--neon-cyan); font-weight: bold;">CUNNING: ${cunningVal}</span>` + // Display Cunning
                ` <span style="margin-left: 10px; color: var(--neon-purple); font-weight: bold;">NURTURING: ${nurturingVal}</span>` + // Display Nurturing
				socialHubBtn + galleryBtn +
				` <button class="outline" style="padding: 2px 8px; font-size: 10px; margin-left: 10px; border-color: #ff4b4b; color: #ff4b4b;" onclick="openHeistPlanningOverlay()">🔪 HEIST TERMINAL</button>` +
                ` <button class="outline" style="padding: 2px 8px; font-size: 10px; margin-left: 10px; border-color: var(--neon-purple); color: var(--neon-purple);" onclick="openPortfolioView()">VIEW PORTFOLIO</button>` + 
                courthouseBtn + blackMarketBtn + rumorMillBtn + securityBtn + bountyBoardBtn + leaseBoardBtn + 
                (Object.keys(myJailedCards).length > 0 ? ` <button class="outline" style="padding: 2px 8px; font-size: 10px; margin-left: 10px; border-color: #ff4b4b; color: #ff4b4b;" onclick="openPortfolioView('jailed')">⛓️ JAILED CARDS (${Object.keys(myJailedCards).length})</button>` : '') + 
                (Object.keys(myKidnappedCards).length > 0 ? ` <button class="outline" style="padding: 2px 8px; font-size: 10px; margin-left: 10px; border-color: #ff4b4b; color: #ff4b4b;" onclick="openPortfolioView('kidnapped')">😈 KIDNAPPED (${Object.keys(myKidnappedCards).length})</button>` : '') + 
                (Object.keys(myHeldHostageCards).length > 0 ? ` <button class="outline" style="padding: 2px 8px; font-size: 10px; margin-left: 10px; border-color: #ffd700; color: #ffd700;" onclick="openPortfolioView('hostage')">🛑 HOSTAGE (${Object.keys(myHeldHostageCards).length})</button>` : '');
            
            if (rewardsDashboard.innerHTML !== newHTML) {
                rewardsDashboard.innerHTML = newHTML;
            }
        }
    }

    // --- Update Latency ---
    const latencyEl = document.getElementById("latency-display"); // Imported from network.js or ui.js
    if (state.latency > 0) {
        latencyEl.innerText = `${state.latency} ms (${state.network_health})`;
        // Use health levels for UI color
        const colors = {"Excellent": "var(--neon-green)", "Good": "#ffd700", "Poor": "#ffa657", "Critical": "#ff4b4b"};
        latencyEl.style.color = colors[state.network_health] || "white";
    } else {
        latencyEl.innerText = "-- ms";
    }
    renderRumorBoard(); // Ensure rumor board is updated

    updateAdminRewardList(state.rewards); // Imported from admin.js

    document.getElementById("network-display").innerText = state.network;

    // --- Update Avatars from WASM URLs ---
    // Update music toggle button icon
    const musicToggleBtn = document.getElementById("music-toggle-btn"); // Imported from audio.js
    if (musicToggleBtn) {
        musicToggleBtn.innerText = state.musicVolume === 0 ? "🔇" : "🎵";
        musicToggleBtn.title = state.musicVolume === 0 ? "Unmute Music" : "Mute Music";
    }

    document.getElementById("p1-avatar").style.backgroundImage = `url('${state.p1_avatar}')`;
    document.getElementById("p2-avatar").style.backgroundImage = `url('${state.p2_avatar}')`;

    // --- Update Avatar Ban Notice ---
    const noticeEl = document.getElementById("avatar-notice-banner"); // Imported from ui.js
    if (state.p1_avatar_notice) {
        if (noticeEl) {
            noticeEl.classList.remove("hidden");
            noticeEl.innerText = state.p1_avatar_notice;
        }
    } else if (noticeEl) {
        noticeEl.classList.add("hidden");
    }

    // --- Admin Panel Visibility ---
    const adminPanel = document.getElementById("admin-control-panel"); // Imported from admin.js
    if (state.is_admin) {
        adminPanel.classList.remove("hidden");
        
        // Sync checkbox states from engine
        if (state.rules) {
            document.getElementById("rule-open").checked = state.rules.Open;
            document.getElementById("rule-same").checked = state.rules.Power_copy;
            document.getElementById("rule-plus").checked = state.rules.Power_up;
            document.getElementById("rule-elemental").checked = state.rules.Elemental_sync;
            document.getElementById("rule-fallen").checked = state.rules.Fallen_penalty;
            document.getElementById("rule-artifact").checked = state.rules.Artifact_bonus;
        }

        // Update Power Scaling Sliders
        if (state.power_divisor) {
            document.getElementById("admin-power-divisor").value = state.power_divisor;
            document.getElementById("admin-power-base").value = state.power_base;
        }

        document.getElementById("dev-mode-toggle").checked = state.testing_mode;
        if (!adminLogTicker) startAdminLogPolling();
    } else {
        if (adminLogTicker) clearInterval(adminLogTicker); // Stop polling if admin panel is closed
        adminPanel.classList.add("hidden");
    }

    // --- Logic for Saving History (Moved to after overlay logic) ---
    if (state.phase === "Active") { matchHistorySaved = false; } // Global variable in app.js
    else if (state.phase === "Finished" && !matchHistorySaved) { await saveMatchResult(state); matchHistorySaved = true; } // Global variable in app.js

    // --- Update Turn Display ---
    let turnDisplay = "Lobby"; // Imported from ui.js
    if (state.phase === "Active") turnDisplay = state.turn === 0 ? "Your Turn" : "Bot Thinking...";
    if (state.phase === "Finished") turnDisplay = "Match Over";
    document.getElementById("turn-display").innerText = turnDisplay;

    // --- Render 3x3 Board ---
    if (scope === "all" || scope === "combat") {
        const boardContainer = document.getElementById("board-container");
        if (boardContainer) {
            // Initialize grid slots if they don't exist (first render)
            if (boardContainer.children.length === 0) {
                for (let i = 0; i < 9; i++) {
                    const slot = document.createElement("div");
                    slot.className = "grid-slot";
                    slot.onclick = () => clickGrid(i);
                    boardContainer.appendChild(slot);
                }
            }

            state.board.forEach((card, index) => {
                const slot = boardContainer.children[index];
                const prevCard = lastBoardState[index];
                const tileMood = state.board_moods ? state.board_moods[index] : "Neutral";

                // Update Mood Visuals for the slot
                const moodClass = `mood-${tileMood.toLowerCase()}`;
                // Remove old mood classes, add new one if not neutral
                // Resetting className to "grid-slot" first ensures only current mood is applied
                if (!slot.classList.contains("grid-slot")) { // Avoid unnecessary re-assignment
                    slot.className = "grid-slot";
                }
                if (tileMood !== "Neutral" && !slot.classList.contains(moodClass)) {
                    slot.classList.add(moodClass);
                } else if (tileMood === "Neutral" && slot.classList.contains(moodClass)) {
                    slot.classList.remove(moodClass);
                }

                // Handle card presence and changes
                if (card) {
                    let cardDiv = slot.querySelector(".playing-card");
                    const isCaptured = card && prevCard && card.owner !== prevCard.owner;

                    if (!cardDiv) {
                        // Card added to an empty slot
                        cardDiv = document.createElement("div");
                        cardDiv.className = "playing-card";
                        slot.appendChild(cardDiv);
                    }

                    // Apply flip animation if captured
                    if (isCaptured) {
                        cardDiv.classList.add("flip-capture");
                        // Remove after animation to allow re-triggering
                        cardDiv.addEventListener('animationend', () => {
                            cardDiv.classList.remove("flip-capture");
                        }, { once: true });
                    }

                    // Update card content and styling
                    const newCardHTML = renderCardHTML(card);
                    if (cardDiv.innerHTML !== newCardHTML) { // Only update if content changed
                        cardDiv.innerHTML = newCardHTML;
                    }
                    const newBorderColor = card.owner === 0 ? "var(--neon-cyan)" : "#ff4b4b";
                    if (cardDiv.style.borderColor !== newBorderColor) { // Only update if color changed
                        cardDiv.style.borderColor = newBorderColor;
                    }

                    // Tooltip Interaction (re-attach if cardDiv was new or replaced)
                    cardDiv.onmouseenter = (e) => {
                        if (tooltipEl && tooltipEl.style.opacity === "1") return;
                        showPowerTooltip(e, card, index, state);
                    };
                    cardDiv.onmousemove = (e) => movePowerTooltip(e);
                    cardDiv.onmouseleave = (e) => {
                        if (e.relatedTarget === tooltipEl) return;
                        hidePowerTooltip();
                    };

                } else {
                    // Slot is empty
                    const cardDiv = slot.querySelector(".playing-card");
                    if (cardDiv) {
                        slot.removeChild(cardDiv);
                    }
                }
            });
        }
    }

    // Update local state tracking for next sync
    lastBoardState = JSON.parse(JSON.stringify(state.board));

    // --- Render Player Hand ---
    const handContainer = document.getElementById("hand-container");
    handContainer.innerHTML = "";
    const placedIds = state.board.filter(c => c !== null).map(c => c.id);
    state.deck.forEach(card => {
        if (!placedIds.includes(card.id)) {
            const cardDiv = document.createElement("div");
            const isSelected = activeCardId === card.id ? 'selected-card' : '';
            cardDiv.className = `playing-card hand-card ${isSelected}`;
            cardDiv.onclick = () => selectCard(card.id);
            cardDiv.innerHTML = renderCardHTML(card);
            handContainer.appendChild(cardDiv);
        }
    });
}

// --- Rumor System UI ---
window.rumorTimers = {}; // Imported from criminality.js

function updateActiveRumors(rumorsData) {
    // Clear existing timers
    for (const id in rumorTimers) {
        clearInterval(rumorTimers[id]);
    }
    rumorTimers = {};

    activeRumors = [];
    if (rumorsData) {
        // If rumorsData is an object (from lobby_update), convert to array
        if (typeof rumorsData === 'object' && !Array.isArray(rumorsData)) {
            for (const id in rumorsData) {
                activeRumors.push(rumorsData[id]);
            }
        } else if (Array.isArray(rumorsData)) { // If it's already an array
            activeRumors = rumorsData;
        } else if (rumorsData.id) { // If it's a single rumor object (from rumor_update)
            activeRumors.push(rumorsData);
        }
    }

    // Filter out expired rumors immediately
    activeRumors = activeRumors.filter(r => new Date(r.ExpiresAt) > Date.now());

    renderRumorBoard();
}

async function renderRumorBoard() {
    let rumorBoard = document.getElementById("rumor-board");
    if (!rumorBoard) {
        rumorBoard = document.createElement("div");
        rumorBoard.id = "rumor-board";
        rumorBoard.className = "rumor-board-container";
        // Find a good place to insert it, e.g., before the chat container
        const chatContainer = document.getElementById("chat-container");
        if (chatContainer) {
            chatContainer.parentNode.insertBefore(rumorBoard, chatContainer);
        } else {
            document.querySelector('.column.right').prepend(rumorBoard); // Fallback
        }
    }

    rumorBoard.innerHTML = "";
    if (activeRumors.length === 0) {
        rumorBoard.classList.add("hidden");
        return;
    }
    rumorBoard.classList.remove("hidden");

    rumorBoard.innerHTML += `<div class="rumor-board-title">ACTIVE RUMORS</div>`;

    for (const rumor of activeRumors) {
        const targetName = await getCachedEnvoiName(rumor.TargetWallet);
        const rumorEl = document.createElement("div");
        rumorEl.className = `rumor-item rumor-${rumor.Type}`;
        rumorEl.innerHTML = `
            <span class="rumor-text">📣 ${rumor.Type.toUpperCase()}: ${targetName}</span>
            <span class="rumor-timer" id="rumor-timer-${rumor.ID}"></span>
        `;
        rumorBoard.appendChild(rumorEl);

        // Start countdown for this rumor
        startRumorCountdown(rumor.ID, rumor.ExpiresAt);
    }
}

function startRumorCountdown(rumorID, expiresAt) {
    const timerEl = document.getElementById(`rumor-timer-${rumorID}`);
    if (!timerEl) return;

    rumorTimers[rumorID] = setInterval(() => {
        const remaining = new Date(expiresAt).getTime() - Date.now();
        if (remaining <= 0) {
            clearInterval(rumorTimers[rumorID]);
            delete rumorTimers[rumorID];
            updateActiveRumors(); // Re-render to remove expired rumor
            return;
        }
        const minutes = Math.floor((remaining % (1000 * 60 * 60)) / (1000 * 60));
        const seconds = Math.floor((remaining % (1000 * 60)) / 1000);
        timerEl.textContent = `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    }, 1000);
}

// --- Initial UI State --- // Imported from ui.js
if (!userAddress) document.getElementById("wallet-selector-overlay").classList.remove("hidden"); // Imported from wallet.js

function buildEmptyBoard() {
    const boardContainer = document.getElementById("board-container");
    boardContainer.innerHTML = "";
    for(let i=0; i<9; i++) {
        boardContainer.innerHTML += `<div class="grid-slot" onclick="clickGrid(${i})">Slot ${i}</div>`;
    }
}

window.renderCardHTML = (card) => { // This will be moved to ui.js
    const rarityBadge = (card.rarity && card.rarity > 1.0) ? `<div class="rarity-badge">${card.rarity.toFixed(1)}x</div>` : '';
    
    // Mood Icon Mapping
    let moodHTML = '';
    if (card.mood && card.mood !== "Neutral") {
        const moodClassMap = { "Volatile": "fire", "Serene": "water", "Spirited": "lightning", "Grounded": "earth" };
        const moodEmojiMap = { "Volatile": "🔥", "Serene": "💧", "Spirited": "⚡", "Grounded": "🌿" };
        const mClass = moodClassMap[card.mood] || "";
        const mEmoji = moodEmojiMap[card.mood] || "✨";
        if (mClass) moodHTML = `<div class="card-type-icon ${mClass}" title="Mood: ${card.mood}">${mEmoji}</div>`;
    }

    // Artifact / Bonus Display
    let artifactHTML = '';
    if (card.artifact > 0) {
        artifactHTML = `<div class="artifact-badge" style="position: absolute; bottom: 30px; right: 5px; color: var(--neon-cyan); font-size: 9px; font-weight: bold; text-shadow: 0 0 5px var(--neon-cyan);">+${card.artifact}</div>`;
    } else if (card.artifact < 0) {
        artifactHTML = `<div class="debuff-badge">PRISONER ${card.artifact}</div>`;
    }

    // Fatigue & Loyalty Indicators
    const fatigue = card.fatigue || 0;
    const loyalty = card.loyalty || 0;
    const statsHTML = `
        <div class="card-mini-stats" style="position: absolute; bottom: 23px; left: 5px; right: 5px; display: flex; justify-content: space-between; font-size: 7px; font-family: 'Rajdhani', sans-serif; letter-spacing: 0.5px; pointer-events: none;">
            <span style="color: ${fatigue > 50 ? '#ff4b4b' : '#8b949e'}">F:${fatigue}</span>
            <span style="color: ${loyalty >= 100 ? 'var(--neon-green)' : '#8b949e'}">L:${loyalty}</span>
        </div>
    `;

    return `
        ${rarityBadge}
        ${artifactHTML}
        ${moodHTML}
        <div class="power-grid">
            <div style="grid-area: top">${window.GetLevelLabelForDisplay(card.power[0])}</div>
            <div style="grid-area: left">${window.GetLevelLabelForDisplay(card.power[3])}</div>
            <div style="grid-area: right">${window.GetLevelLabelForDisplay(card.power[1])}</div>
            <div style="grid-area: bottom">${window.GetLevelLabelForDisplay(card.power[2])}</div>
        </div>
        ${statsHTML}
        <div class="card-name">${card.name}</div>
    `;
}

window.showPowerTooltip = (e, card, index, state) => { // This will be moved to ui.js
    if (!tooltipEl) {
        tooltipEl = document.createElement("div");
        tooltipEl.className = "power-tooltip";
        document.body.appendChild(tooltipEl);
    }

    const tileMood = state.board_moods ? state.board_moods[index] : "Neutral";
    const moodWeaknesses = { "Volatile": "Serene", "Serene": "Spirited", "Spirited": "Grounded", "Grounded": "Volatile" };
    
    let html = `<div style="color: var(--neon-cyan); font-weight: bold; margin-bottom: 8px; border-bottom: 1px solid var(--neon-cyan); padding-bottom: 5px;">${card.name.toUpperCase()} DATA</div>`;
    
    const sides = ["TOP", "RIGHT", "BOTTOM", "LEFT"];
    
    // Get player stats for the card owner to calculate player-level modifiers
    const ownerPlayerIndex = card.owner;
    const ownerWantedLevel = (ownerPlayerIndex === 0 ? state.p1_wanted_level : state.p2_wanted_level) || 0;
    const ownerCunning = (ownerPlayerIndex === 0 ? state.p1_cunning : state.p2_cunning) || 0;
    const ownerNurturing = (ownerPlayerIndex === 0 ? state.p1_nurturing : state.p2_nurturing) || 0;

    // Calculate player-level modifiers once
    let netWantedPenalty = 0;
    if (ownerWantedLevel > 0) {
        const baseWantedPenalty = ownerWantedLevel * 5;
        const mitigation = ownerCunning * 2;
        netWantedPenalty = -(baseWantedPenalty - Math.min(mitigation, baseWantedPenalty));
    }

    sides.forEach((side, sideIndex) => {
        const base = card.power[i];
        const artifactBonus = card.artifact || 0;
        
        let moodModifier = 0;
        if (state.rules?.Elemental_sync && tileMood !== "Neutral" && card.mood && card.mood !== "Neutral") {
            if (card.mood === tileMood) {
                moodModifier = 50; // Match bonus
            } else if (moodWeaknesses[card.mood] === tileMood) {
                moodModifier = -50; // Weakness penalty
            }
        }

        let netFatiguePenalty = 0;
        if (card.fatigue > 50) {
            const baseFatiguePenalty = (card.fatigue - 50);
            const reduction = ownerNurturing;
            netFatiguePenalty = -(baseFatiguePenalty - Math.min(reduction, baseFatiguePenalty));
        }

        const loyaltyBonus = card.loyalty >= 100 ? 25 : 0;

        const totalEffectivePower = base + artifactBonus + moodModifier + netFatiguePenalty + loyaltyBonus + netWantedPenalty;
        const grade = window.GetLevelLabelForDisplay(totalEffectivePower);
        
        // Build the HTML for modifiers
        let modifiersHtml = '';
        if (artifactBonus !== 0) {
            modifiersHtml += `<span style="color: ${artifactBonus > 0 ? 'var(--neon-cyan)' : '#ff4b4b'}">${artifactBonus > 0 ? '+' : ''}${artifactBonus}A</span> `;
        }
        if (moodModifier !== 0) {
            modifiersHtml += `<span style="color: ${moodModifier > 0 ? 'var(--neon-green)' : '#ff4b4b'}">${moodModifier > 0 ? '+' : ''}${moodModifier}M</span> `;
        }
        if (netFatiguePenalty !== 0) {
            modifiersHtml += `<span style="color: #ff4b4b">${netFatiguePenalty}F</span> `;
        }
        if (loyaltyBonus !== 0) {
            modifiersHtml += `<span style="color: var(--neon-cyan)">+${loyaltyBonus}L</span> `;
        }
        if (netWantedPenalty !== 0) {
            modifiersHtml += `<span style="color: #ff4b4b">${netWantedPenalty}W</span> `;
        }

        html += `
            <div class="tooltip-row">
                <span style="opacity: 0.7;">${side}:</span>
                <span style="display: flex; align-items: center; gap: 5px;">
                    <span>${base}</span>
                    ${modifiersHtml ? `<span style="font-size: 0.8em; opacity: 0.8;">(${modifiersHtml.trim()})</span>` : ''}
                    <span>=</span>
                    <b style="color: var(--neon-cyan)">${totalEffectivePower} (${grade})</b>
                </span>
            </div>
        `;
    });

    if (state.rules?.Artifact_bonus && card.owner === myPlayerIndex) {
        html += `
            <div class="tooltip-quickcast">
                <button onclick="event.stopPropagation(); showQuickCastMenu(${index})">
                    ⚡ QUICK-CAST ITEM
                </button>
            </div>
        `;
    }

    if (card.mood && card.mood !== "Neutral") {
        html += `<div style="margin-top: 8px; font-size: 10px; opacity: 0.6; text-align: center;">MOOD: ${card.mood.toUpperCase()} vs TILE: ${tileMood.toUpperCase()}</div>`;
    }

    tooltipEl.innerHTML = html;
    tooltipEl.style.opacity = "1";
    tooltipEl.style.pointerEvents = (state.rules?.Artifact_bonus && card.owner === myPlayerIndex) ? "auto" : "none";
    tooltipEl.onmouseleave = () => hidePowerTooltip();
    movePowerTooltip(e);
}

window.movePowerTooltip = (e) => { // This will be moved to ui.js
    if (!tooltipEl) return;
    const padding = 15;
    let x = e.clientX + padding;
    let y = e.clientY + padding;

    // Boundary check to keep tooltip on screen
    if (x + 220 > window.innerWidth) x = e.clientX - 230;
    if (y + 180 > window.innerHeight) y = e.clientY - 190;

    tooltipEl.style.left = x + "px";
    tooltipEl.style.top = y + "px";
}

window.hidePowerTooltip = () => { // This will be moved to ui.js
    if (tooltipEl) tooltipEl.style.opacity = "0";
}

window.showQuickCastMenu = (gridIndex) => { // This will be moved to game.js
    const container = document.querySelector(".tooltip-quickcast"); // Imported from ui.js
    if (!container) return;

    const state = window.GetGameState();
    // Filter inventory for items that aren't currently in the active deck
    const deckIds = state.deck.map(c => c.id);
    const artifacts = state.inventory.filter(c => !deckIds.includes(c.id) && c.artifact > 0);
    
    if (artifacts.length === 0) {
        container.innerHTML = `<span style="color: #ff4b4b; font-size: 11px; font-weight: bold;">NO ITEMS AVAILABLE</span>`;
        return;
    }

    let html = `<div class="quickcast-item-list">`;
    artifacts.forEach(item => {
        html += `
            <button class="quickcast-item-btn" onclick="event.stopPropagation(); executeQuickCast(${item.id}, ${gridIndex})">
                <span>${item.name}</span>
                <b style="color: inherit;">+${item.artifact}</b>
            </button>
        `;
    });
    html += `</div>`;
    container.innerHTML = html;
}

window.executeQuickCast = async (itemId, gridIndex) => {
    const state = window.GetGameState();
    const item = state.inventory.find(c => c.id === itemId);
    if (!item) return;

    const success = window.ApplyArtifactToBoard(gridIndex, item.artifact);

    if (success) {
        showToast(`⚡ Used ${item.name} on ${state.board[gridIndex].name}!`, "success");
        if (state.multiplayer && currentOpponentId) {
            socket.send(JSON.stringify({
                type: "use_item",
                to_id: currentOpponentId,
                payload: { grid_index: gridIndex, bonus: item.artifact }
            }));
        }
        hidePowerTooltip();
        syncUI();
    }
}
// Imported from economy.js
window.openClubFoundry = () => {
    const overlay = document.createElement("div");
    overlay.id = "club-foundry-overlay";
    overlay.className = "overlay";
    overlay.innerHTML = `
        <div class="glass-panel" style="width: 450px; text-align: center;">
            <h2 style="color: var(--neon-purple);">CLUB FOUNDRY</h2>
            <p style="font-size: 0.9em; opacity: 0.8;">Founding a club costs a fortune (5,000 $VBV).<br>Owners earn commissions from relative buffs sold in their territory.</p>
            
            <div class="flex-col gap-10 mt-20">
                <input type="text" id="foundry-club-name" class="glass-input w-full" placeholder="Enter Club Name (max 20 chars)" maxlength="20">
                
                <select id="foundry-shop-type" class="glass-input w-full">
                    <option value="Elemental">Elemental Forge (Mood Buffs)</option>
                    <option value="Tactical">Tactical Syndicate (Rule Mastery)</option>
                    <option value="Vitality">Vitality Lab (Health/Loyalty)</option>
                </select>
                
                <select id="foundry-territory" class="glass-input w-full">
                    <option value="the_lab">The Lab</option>
                    <option value="casino">The Casino</option>
                    <option value="arena_center">The Central Arena</option>
                </select>
            </div>

            <div class="mt-20 flex-row justify-center gap-15">
                <button class="outline" onclick="document.getElementById('club-foundry-overlay').remove()">CANCEL</button>
                <button id="foundry-submit-btn" onclick="submitClubFoundry()">FOUND CLUB (5,000 $VBV)</button>
            </div>
        </div>
    `;
    document.body.appendChild(overlay);
}

/**
 * Populates and displays the district shops overlay using synchronized club inventory.
 * Now utilizes the high-fidelity _shops.scss styles and category filtering.
 */
window.openShopsOverlay = async (initialCategory = 'Elemental') => { // This will be moved to economy.js
	document.getElementById("shops-overlay").classList.remove("hidden");
	switchShopCategory(initialCategory);
}

function switchShopCategory(category) {
	const container = document.getElementById("shops-container");
	if (!container) return;

	// Update Tab State
	document.querySelectorAll('.category-tab').forEach(tab => {
		tab.classList.toggle('active', tab.dataset.category === category);
	});

	container.innerHTML = `<div class="grid-span-all opacity-5 py-40 italic">Scanning district stock for ${category} hardware...</div>`;

	// Client-side item metadata registry (mirrors shop_registry.go)
	const itemRegistry = {
		"mood_catalyst": { name: "Mood Catalyst", desc: "+50 Mood Bonus (3 Matches)", price: 100 },
		"grounded_shield": { name: "Grounded Shield", desc: "Immunity to Mood Penalties (5 Matches)", price: 250 },
		"rule_breaker": { name: "Rule Breaker", desc: "Force PLUS trigger (1 Match)", price: 150 },
		"intel_report": { name: "Intel Report", desc: "See Opponent Hand (3 Matches)", price: 300 },
		"stamina_stim": { name: "Stamina Stim", desc: "-20 Fatigue Immediately", price: 100 },
		"loyalty_pledge": { name: "Loyalty Pledge", desc: "+10 Loyalty Immediately", price: 500 },
		"tripwire": { name: "Laser Tripwire", desc: "+10% Heist Failure", price: 500 },
		"sentry_turret": { name: "Sentry Turret", desc: "+25% Heist Failure", price: 1200 },
		"guard_dog": { name: "Bio-Guard Dog", desc: "Forces Jail on Failure", price: 2000 }
	};

	const clubs = Object.values(globalClubs).filter(c => c.type === category);
	let itemsHTML = "";

	clubs.forEach(club => {
		Object.entries(club.inventory || {}).forEach(([itemId, qty]) => {
			if (qty <= 0) return;
			const meta = itemRegistry[itemId] || { name: itemId.replace(/_/g, ' '), desc: "Tactical Enhancement", price: 100 };
			
			itemsHTML += `
				<div class="shop-item animate-slide-up" onclick="buyClubItem('${club.id}', '${itemId}', ${meta.price}, '${club.territories[0]}')">
					<div class="item-image">
						<img src="Assets/Images/portraits/placeholder.webp" alt="Hardware">
						<div class="item-badge">${club.name}</div>
					</div>
					<div class="item-info">
						<div class="item-title">${meta.name.toUpperCase()}</div>
						<div class="item-description">${meta.desc}</div>
						<div class="item-stats">
							<div class="stat">
								<div class="stat-label">STOCK</div>
								<div class="stat-value">${qty}</div>
							</div>
						</div>
					</div>
					<div class="item-footer">
						<div class="item-price">${meta.price}</div>
						<button class="buy-button">PURCHASE</button>
					</div>
				</div>
			`;
		});
	});

	if (itemsHTML === "") {
		container.innerHTML = `<div class="grid-span-all opacity-3 py-40 italic">Sector is currently dry for ${category} assets.</div>`;
	} else {
		container.innerHTML = itemsHTML;
	}
}

window.mapZoom = 1.0; // Imported from economy.js
/**
 * Adjusts the zoom level of the 3D map grid.
 */
function adjustMapZoom(delta) {
    mapZoom += delta;
    if (mapZoom < 0.5) mapZoom = 0.5;
    if (mapZoom > 2.0) mapZoom = 2.0;
    const grid = document.getElementById("map-3d-grid");
    if (grid) {
        grid.style.transform = `rotateX(30deg) rotateY(-15deg) scale(${mapZoom})`;
    }
}

window.openTerritoryMapOverlay = () => { // This will be moved to economy.js
    const grid = document.getElementById("map-3d-grid");
    if (!grid) return;
    
    // Reset Zoom
    mapZoom = 1.0;
    grid.style.transform = `rotateX(30deg) rotateY(-15deg) scale(${mapZoom})`;
    
    grid.innerHTML = "";
    
    const territoryMap = [
        { id: "the_lab", name: "The Lab", icon: "🧪" },
        { id: "north_district", name: "North Gate", icon: "⛩️" },
        { id: "the_archive", name: "The Archive", icon: "📜" },
        { id: "west_port", name: "West Port", icon: "⚓" },
        { id: "arena_center", name: "Arena Center", icon: "⚔️" },
        { id: "east_gate", name: "East Gate", icon: "🏯" },
        { id: "south_slums", name: "The Slums", icon: "🏚️" },
        { id: "casino", name: "The Casino", icon: "🎰" },
        { id: "data_haven", name: "Data Haven", icon: "💾" }
    ];

    territoryMap.forEach(t => {
        const club = Object.values(globalClubs).find(c => c.territories && c.territories.includes(t.id));
        const isOwned = !!club;
        const isGovernor = isOwned && club.region_name;
        
        let tileClasses = `map-tile-3d accelerated`;
        if (isGovernor) tileClasses += " governor-controlled";
        else if (isOwned) tileClasses += " controlled";
        else tileClasses += " neutral";

        const tile = document.createElement("div");
        tile.className = tileClasses;
        tile.onclick = () => {
            hideAllOverlays();
            openTerritoryView(t.id);
        };
        
        tile.innerHTML = `
            <div class="tile-label">
                <div style="font-size: 24px; margin-bottom: 5px;">${t.icon}</div>
                <div class="tile-name">${t.name.toUpperCase()}</div>
                <div class="tile-owner">${isOwned ? club.name : 'NEUTRAL ZONE'}</div>
                ${isOwned ? `
                <div class="tile-stats">
                    <span class="stat population" title="Staff Count">${Object.keys(club.staff || {}).length}</span>
                    <span class="stat resources" title="Treasury">${club.treasury.toFixed(0)}</span>
                    <span class="stat defense" title="Club Mojo">${club.club_mojo}</span>
                </div>` : ''}
            </div>
            ${isGovernor ? '<div class="tile-status developing"></div>' : ''}
        `;
        grid.appendChild(tile);
    });

    document.getElementById("territory-map-overlay").classList.remove("hidden");
}

window.submitClubFoundry = async () => {
    const name = document.getElementById("foundry-club-name").value.trim();
    const type = document.getElementById("foundry-shop-type").value;
    const territory = document.getElementById("foundry-territory").value;
    const btn = document.getElementById("foundry-submit-btn");

    if (!name) return showToast("Club name required", "error");
    if (!userAddress) return showToast("Connect wallet first", "error");

    btn.disabled = true;
    btn.innerText = "Processing...";

    try {
        const state = window.GetGameState();
        const network = state.network;
        const assetId = network === "VOI" ? CONFIG.VBV_ASSET_ID : CONFIG.AVOI_ASSET_ID;
        const amountMicro = 5000 * 1000000;

        showToast("Signing 5,000 $VBV Fortune Burn...", "info");

        let txid = "";
        // Reusing construction logic from registerForTournament
        if (network === "VOI") {
            const methodSelector = new Uint8Array([0x2b, 0x42, 0x6d, 0xec]); // transfer(address,uint256)
            const recipientAddr = algosdk.decodeAddress(CONFIG.VAULT_ADDRESS).publicKey;
            const amountArg = new Uint8Array(32);
            const amountBI = BigInt(amountMicro);
            for (let i = 0; i < 8; i++) amountArg[31 - i] = Number((amountBI >> BigInt(i * 8)) & 0xffn);

            const txObj = {
                from: userAddress, type: 'appl', appIndex: parseInt(assetId),
                appArgs: [methodSelector, recipientAddr, amountArg],
                note: new TextEncoder().encode(`FOUND_CLUB:${name}`)
            };
            
            if (walletProvider === 'nautilus') {
                const signed = await window.algo.signTxn([{ txn: algosdk.encodeObj(txObj), signers: [userAddress] }]);
                const { txId } = await window.algo.sendRawTransaction(signed[0]);
                txid = txId;
            }
            // Additional providers would be handled here as in registerForTournament
        }

        if (!txid) throw new Error("Transaction cancelled or provider not supported.");

        socket.send(JSON.stringify({
            type: "create_club",
            payload: {
                name: name,
                type: type,
                territory_id: territory,
                txid: txid,
                network: network
            }
        }));

        document.getElementById("club-foundry-overlay").remove();
    } catch (err) {
        showToast(`Founding Failed: ${err.message}`, "error");
        btn.disabled = false;
        btn.innerText = "FOUND CLUB (5,000 $VBV)";
    }
}

window.openTerritoryView = (territoryId) => { // This will be moved to economy.js
    const club = Object.values(globalClubs).find(c => c.territory === territoryId);
    const overlay = document.createElement("div");
    overlay.id = "territory-view-overlay";
    overlay.className = "overlay";
    
    let header = `<h2>TERRITORY: ${territoryId.replace('_', ' ').toUpperCase()}</h2>`;
    let body = `<p style="opacity: 0.7;">This territory is currently unclaimed. Found a Club to take control!</p>`;

    if (club) {
        header = `
            <h2 style="color: var(--neon-cyan); margin-bottom: 5px;">${club.name}</h2>
            <div style="font-size: 0.8em; opacity: 0.6; margin-bottom: 15px;">Controlled by: ${club.owner_wallet.substring(0,8)}...</div>
            <div class="flex-row justify-center gap-15 mb-20">
                <div class="glass-panel p-10 m-0" style="min-width: 120px;">
                    <div style="font-size: 0.7em; opacity: 0.5;">TREASURY</div>
                    <b style="color: var(--neon-green);">${club.treasury.toFixed(2)} $VBV</b>
                </div>
                <div class="glass-panel p-10 m-0" style="min-width: 120px;">
                    <div style="font-size: 0.7em; opacity: 0.5;">MOJO</div>
                    <b style="color: var(--neon-purple);">${club.club_mojo}</b>
                </div>
            </div>
        `;

        const shopItems = {
            "Elemental": [
                { id: "mood_catalyst", name: "Mood Catalyst", price: 100, desc: "+50 Mood Bonus (3 Matches)" },
                { id: "grounded_shield", name: "Grounded Shield", price: 250, desc: "Immunity to Mood Penalties (5 Matches)" }
            ],
            "Tactical": [
                { id: "rule_breaker", name: "Rule Breaker", price: 150, desc: "Force PLUS trigger (1 Match)" },
                { id: "intel_report", name: "Intel Report", price: 300, desc: "See Opponent Hand (3 Matches)" }
            ],
            "Vitality": [
                { id: "stamina_stim", name: "Stamina Stim", price: 100, desc: "-20 Fatigue Immediately" },
                { id: "loyalty_pledge", name: "Loyalty Pledge", price: 500, desc: "+10 Loyalty Immediately" }
            ]
        };

        const items = shopItems[club.type] || [];
        body = `
            <div class="flex-col gap-10">
                ${items.map(item => `
					<div class="shop-item-row glass-panel p-15 m-0 flex-row justify-between align-center animate-shimmer">
                        <div style="text-align: left;">
							<b class="text-neon-cyan">${item.name}</b>
							<div class="item-description" style="font-size: 0.8em; opacity: 0.6;">${item.desc}</div>
                        </div>
						<button class="buy-btn outline" style="min-width: 100px; padding: 8px;" onclick="buyClubItem('${club.id}', '${item.id}', ${item.price}, '${territoryId}')">
                            ${item.price} $VBV
                        </button>
                    </div>
                `).join('')}
            </div>
        `;
    }

    overlay.innerHTML = `
        <div class="glass-panel" style="width: 500px; text-align: center;">
            ${header}
            ${body}
            <div class="mt-20">
                <button class="outline" onclick="document.getElementById('territory-view-overlay').remove()">CLOSE MAP</button>
                ${!club ? `<button onclick="document.getElementById('territory-view-overlay').remove(); openClubFoundry()">FOUND CLUB</button>` : ''}
            </div>
        </div>
    `;
    document.body.appendChild(overlay);
}

window.buyClubItem = async (clubId, itemId, price, territoryId) => { // This will be moved to economy.js
    if (!userAddress) return showToast("Connect wallet first", "error");
    
    try {
        const state = window.GetGameState();
        showToast(`Purchasing ${itemId} for ${price} $VBV...`, "info");
        
        // In a full implementation, this would involve an ARC-200/ASA transfer to the Club Owner's address
        // For now, we simulate the economic signal to the server
        socket.send(JSON.stringify({
            type: "purchase_item",
            payload: {
                item_id: itemId,
                territory_id: territoryId,
                price: price * 1000000 // Convert to micro-units
            }
        }));

        // If it's a Vitality item, apply it immediately to the local engine
        if (itemId === "stamina_stim") {
            showToast("⚡ Fatigue reduced! Your cards are feeling refreshed.", "success");
        }

        document.getElementById("territory-view-overlay")?.remove();
    } catch (err) {
        showToast(`Purchase Failed: ${err.message}`, "error");
    }
}


window.openCourthouse = () => { // This will be moved to criminality.js
    const state = window.GetGameState();
    const wanted = state.wanted_level || 0;
    if (wanted <= 0) return;

    const fine = wanted * 100;
    const overlay = document.createElement("div");
    overlay.id = "courthouse-overlay";
    overlay.className = "overlay";
    overlay.innerHTML = `
        <div class="glass-panel courthouse-panel">
            <h2 class="text-error" style="letter-spacing: 3px;">ARENA COURTHOUSE</h2>
            <p style="font-size: 0.9em; opacity: 0.8;">The High Council has flagged you for criminal activities.<br>Infamy Status: <b>LEVEL ${wanted}</b></p>
            
            <div class="glass-panel fine-display-box">
                <div style="font-size: 0.75em; opacity: 0.7;">REHABILITATION FINE</div>
                <b class="fine-amount">${fine} $VBV</b>
                <div style="font-size: 0.7em; opacity: 0.5; margin-top: 5px;">(100 $VBV per Wanted point)</div>
            </div>

            <p style="font-size: 0.8em; opacity: 0.6; padding: 0 20px;">Settling your debt to society will clear your Wanted Level and restore your cards to peak combat performance.</p>

            <div class="mt-20 flex-row justify-center gap-15">
                <button class="outline" onclick="document.getElementById('courthouse-overlay').remove()">LURK IN SHADOWS</button>
                <button id="courthouse-pay-btn" style="background: linear-gradient(45deg, #ff4b4b, #ff0844);" onclick="submitCourthouseFine()">PAY FINE & CLEAR NAME</button>
            </div>
        </div>
    `;
    document.body.appendChild(overlay);
}

window.submitCourthouseFine = async () => { // This will be moved to criminality.js
    const state = window.GetGameState();
    const wanted = state.wanted_level || 0;
    if (wanted <= 0) return;

    const btn = document.getElementById("courthouse-pay-btn");
    const amountMicro = wanted * 100 * 1000000;
    const network = state.network;
    const assetId = network === "VOI" ? CONFIG.VBV_ASSET_ID : CONFIG.AVOI_ASSET_ID;

    btn.disabled = true;
    btn.innerText = "PROCESSING...";

    try {
        showToast(`⚖️ Signing ${wanted * 100} $VBV Fine...`, "info");
        let txid = "";
        let txObj = null;

        if (network === "VOI") {
            const methodSelector = new Uint8Array([0x2b, 0x42, 0x6d, 0xec]); // transfer(address,uint256)
            const recipientAddr = algosdk.decodeAddress(CONFIG.VAULT_ADDRESS).publicKey;
            const amountArg = new Uint8Array(32);
            const amountBI = BigInt(amountMicro);
            for (let i = 0; i < 8; i++) amountArg[31 - i] = Number((amountBI >> BigInt(i * 8)) & 0xffn);

            txObj = {
                from: userAddress, type: 'appl', appIndex: parseInt(assetId),
                appArgs: [methodSelector, recipientAddr, amountArg],
                note: new TextEncoder().encode(`ARENA_COURTHOUSE_FINE:${wanted}`)
            };
        } else {
            txObj = {
                from: userAddress, to: CONFIG.VAULT_ADDRESS, type: 'axfer',
                assetIndex: parseInt(assetId), amount: amountMicro,
                note: new TextEncoder().encode(`ARENA_COURTHOUSE_FINE:${wanted}`)
            };
        }

        if (walletProvider === 'nautilus') {
            const signed = await window.algo.signTxn([{ txn: algosdk.encodeObj(txObj), signers: [userAddress] }]);
            const { txId } = await window.algo.sendRawTransaction(signed[0]);
            txid = txId;
        } else if (walletProvider === 'kibisis') {
            const txnB64 = btoa(String.fromCharCode(...algosdk.encodeObj(txObj)));
            const signed = await window.kibisis.signTxns([{ txn: txnB64 }]);
            const { txId } = await window.kibisis.pushTxns(signed);
            txid = txId;
        } else if (walletProvider === 'walletconnect' && signClient) {
            const sessions = signClient.session.getAll();
            const chainId = network === "VOI" ? CONFIG.VOI_CHAIN_ID : CONFIG.ALGO_CHAIN_ID;
            const txnB64 = btoa(String.fromCharCode(...algosdk.encodeObj(txObj)));
            const response = await signClient.request({
                topic: sessions[0].topic, chainId: chainId,
                request: { method: "algo_signTxn", params: [[{ txn: txnB64, signers: [userAddress] }]] }
            });
            const signedTxnBytes = new Uint8Array(atob(response[0]).split("").map(c => c.charCodeAt(0)));
            const netCfg = getNetworkConfig(network);
            const client = new algosdk.Algodv2("", netCfg.node_url, "");
            const { txId } = await client.sendRawTransaction(signedTxnBytes).do();
            txid = txId;
        }

        if (!txid) throw new Error("Transaction cancelled or failed.");

        const response = await fetch(`${CONFIG.API_BASE}/api/courthouse/reset`, {
            method: "POST",
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ wallet: userAddress, txid: txid, network: network })
        });

        if (response.ok) {
            const result = await response.json();
            showToast(`⚖️ ${result.message}`, "success");
            document.getElementById("courthouse-overlay")?.remove();
            if (window.SyncOpponentWanted) window.SyncOpponentWanted(myPlayerIndex, 0);
            syncUI();
        } else {
            const err = await response.text();
            showToast(`❌ Courthouse Error: ${err}`, "error");
        }
    } catch (err) {
        showToast(`Fine Payment Failed: ${err.message}`, "error");
    } finally {
        btn.disabled = false;
        btn.innerText = "PAY FINE & CLEAR NAME";
    }
}

window.openPortfolioView = async (initialTab = 'portfolio') => { // This will be moved to economy.js
    const state = window.GetGameState();
    const overlay = document.createElement("div");
    const myJailedCards = state.jailed_cards || {};
    const myKidnappedCards = state.kidnapped_cards || {};
    const myHeldHostageCards = state.held_hostage_cards || {};
    overlay.id = "portfolio-view-overlay";
    overlay.className = "overlay";
    
    overlay.innerHTML = `
        <div class="glass-panel" style="width: 500px; text-align: center;">
            <h2 style="color: var(--neon-cyan);">ENTITY PORTFOLIO</h2>
            <div class="flex-row justify-center gap-10 mt-10 mb-20">
                <button id="tab-holdings" class="tab-btn ${initialTab === 'portfolio' ? 'active' : ''}" onclick="switchPortfolioTab('portfolio')">📈 HOLDINGS</button>
                <button id="tab-jailed" class="tab-btn ${initialTab === 'jailed' ? 'active' : ''}" onclick="switchPortfolioTab('jailed')">⛓️ JAILED (${Object.keys(myJailedCards).length})</button>
                <button id="tab-kidnapped" class="tab-btn ${initialTab === 'kidnapped' ? 'active' : ''}" onclick="switchPortfolioTab('kidnapped')">😈 KIDNAPPED (${Object.keys(myKidnappedCards).length})</button>
                <button id="tab-hostage" class="tab-btn ${initialTab === 'hostage' ? 'active' : ''}" onclick="switchPortfolioTab('hostage')">🛑 HOSTAGE (${Object.keys(myHeldHostageCards).length})</button>
            </div>
            
            <div id="portfolio-content-area" class="flex-col gap-10" style="max-height: 400px; overflow-y: auto; padding-right: 5px;">
                <!-- Content injected by switchPortfolioTab -->
            </div>

            <button class="outline mt-20 w-full" onclick="document.getElementById('portfolio-view-overlay').remove()">CLOSE</button>
        </div>
    `;

    document.body.appendChild(overlay);
    switchPortfolioTab(initialTab);
}

window.switchPortfolioTab = async (tab) => { // This will be moved to economy.js
    const container = document.getElementById("portfolio-content-area");
    const holdingsBtn = document.getElementById("tab-holdings");
    const jailedBtn = document.getElementById("tab-jailed");
    const kidnappedBtn = document.getElementById("tab-kidnapped");
    const hostageBtn = document.getElementById("tab-hostage");
    const state = window.GetGameState();

    holdingsBtn.classList.toggle("active", tab === 'portfolio');
    jailedBtn.classList.toggle("active", tab === 'jailed');
    if (kidnappedBtn) kidnappedBtn.classList.toggle("active", tab === 'kidnapped');
    if (hostageBtn) hostageBtn.classList.toggle("active", tab === 'hostage');
    container.innerHTML = `<div style="padding: 20px; opacity: 0.5;">Loading details...</div>`;

    if (tab === 'portfolio') {
        const portfolio = state.portfolio || {};
        const entries = Object.entries(portfolio);
        let html = "";
        let totalMarketValue = 0;

        if (entries.length === 0) {
            html = `<div style="padding: 40px; opacity: 0.5;">No active investments found.</div>`;
        } else {
            const walletsToResolve = entries.map(([w]) => w);
            await Promise.all(walletsToResolve.map(w => resolveEnvoiName(w)));

			const itemsHtml = entries.map(([id, amount]) => {
                if (amount <= 0) return;
                const p = lastLobbyPlayers.find(pl => pl.wallet && pl.wallet.toLowerCase() === id.toLowerCase());
                const price = p ? ((p.wins * 10) + (p.reputation / 2) + 100) : 100;
                const marketValue = amount * price;
                totalMarketValue += marketValue;
                const displayName = getCachedEnvoiName(id);
				
				return `
					<div class="portfolio-item glass-panel m-0 mb-10">
						<div class="item-info">
							<div class="item-icon">👤</div>
							<div class="item-details text-left">
								<div class="item-name font-bold text-neon-cyan">${displayName}</div>
								<div class="item-type font-size-0-75em opacity-5">Entity Shares</div>
							</div>
						</div>
						<div class="item-stats">
							<div class="stat">
								<div class="stat-label">SHARES</div>
								<div class="stat-value text-neon-green">${amount.toFixed(2)}</div>
							</div>
						</div>
						<div class="item-value">
							<div class="font-bold text-neon-green">${marketValue.toFixed(1)} $VBV</div>
							<button class="outline x-small border-error mt-5" onclick="tradeShares('${id}', 'sell', ${amount})">SELL ALL</button>
						</div>
					</div>`;
			}).join('');

			html = `
				<div class="portfolio-system">
					<div class="portfolio-header mb-15">
						<span class="portfolio-title text-neon-purple font-bold">MARKET HOLDINGS</span>
						<div class="portfolio-value font-size-1-2em text-neon-green">VALUATION: ${totalMarketValue.toFixed(1)} $VBV</div>
					</div>
					<div class="portfolio-view">
						${itemsHtml}
					</div>
				</div>`;
        }
        container.innerHTML = html;
    } else if (tab === 'jailed') {
        const jailed = state.jailed_cards || {};
        const cardIds = Object.keys(jailed);
        if (cardIds.length === 0) {
            container.innerHTML = `<div style="padding: 40px; opacity: 0.5;">No cards currently in custody.</div>`;
            return;
        }

        let html = "";
        for (const cardId of cardIds) {
            const clubId = jailed[cardId];
            const club = globalClubs[clubId] || { name: "Underworld Entity" };
            html += `
                <div class="player-item" style="padding: 15px; border-color: #ff4b4b;">
                    <div style="text-align: left;">
                        <b style="color: #ff4b4b;">ID: #${cardId}</b>
                        <div style="font-size: 0.75em; opacity: 0.6;">Held by: ${club.name}</div>
                    </div>
                    <div style="text-align: right;">
                        <button class="outline" style="font-size: 10px; padding: 6px 12px; border-color: var(--neon-green); color: var(--neon-green);" 
                                onclick="initiateBail(${cardId}, '${clubId}')">PAY BAIL (200 $VBV)</button>
                    </div>
                </div>`;
        }
        container.innerHTML = html;
    } else if (tab === 'kidnapped') {
        const kidnapped = state.kidnapped_cards || {};
        const cardIds = Object.keys(kidnapped);
        if (cardIds.length === 0) {
            container.innerHTML = `<div style="padding: 40px; opacity: 0.5;">No kidnapped cards at the moment.</div>`;
            return;
        }

        let html = "";
        for (const cardId of cardIds) {
            const victimWallet = kidnapped[cardId] || "Unknown";
            html += `
                <div class="player-item" style="padding: 15px; border-color: #ffa500;">
                    <div style="text-align: left;">
                        <b style="color: #ffa500;">ID: #${cardId}</b>
                        <div style="font-size: 0.75em; opacity: 0.6;">Victim Wallet: ${victimWallet}</div>
                    </div>
                    <div style="text-align: right;">
                        <button class="outline" style="font-size: 9px; padding: 4px 8px; border-color: #ffd700; color: #ffd700;" onclick="releaseHostage(${cardId})">RELEASE HOSTAGE</button>
                    </div>
                </div>`;
        }
        container.innerHTML = html;
    } else if (tab === 'hostage') {
        const heldHostage = state.held_hostage_cards || {};
        const cardIds = Object.keys(heldHostage);
        if (cardIds.length === 0) {
            container.innerHTML = `<div style="padding: 40px; opacity: 0.5;">No cards currently held hostage.</div>`;
            return;
        }

        let html = "";
        for (const cardId of cardIds) {
            const perpWallet = heldHostage[cardId] || "Unknown";
            html += `
                <div class="player-item" style="padding: 15px; border-color: #ffd700;">
                    <div style="text-align: left;">
                        <b style="color: #ffd700;">ID: #${cardId}</b>
                        <div style="font-size: 0.75em; opacity: 0.6;">Kidnapper: ${perpWallet}</div>
                    </div>
                    <div style="text-align: right;">
                        <button class="outline" style="font-size: 9px; padding: 4px 8px; border-color: #ff4b4b; color: #ff4b4b;" onclick="payRansom(${cardId}, '${perpWallet}')">PAY RANSOM</button>
                    </div>
                </div>`;
        }
        html += `<div style="margin-top: 10px; padding: 12px; border: 1px dashed var(--glass-border); color: #ffd700; font-size: 0.85em;">Ransom amount will be requested after you initiate payment.</div>`;
        container.innerHTML = html;
    } else {
        container.innerHTML = `<div style="padding: 40px; opacity: 0.5;">No details available for this tab.</div>`;
    }
}

window.initiateBail = async (cardId, clubId) => { // This will be moved to criminality.js
    if (!userAddress) return;
    if (!confirm(`Are you sure you want to pay 200 $VBV to release Card #${cardId}?`)) return;

    try {
        const state = window.GetGameState();
        const network = state.network;
        const bailAmountMicro = 200 * 1000000;
        const assetId = network === "VOI" ? CONFIG.VBV_ASSET_ID : CONFIG.AVOI_ASSET_ID;

        showToast(`⚖️ Signing Bail Payment for Card #${cardId}...`, "info");
        
        let txid = "";
        // Construction logic mirroring courthouse fine
        if (network === "VOI") {
            const methodSelector = new Uint8Array([0x2b, 0x42, 0x6d, 0xec]); 
            const recipientAddr = algosdk.decodeAddress(CONFIG.VAULT_ADDRESS).publicKey;
            const amountArg = new Uint8Array(32);
            const amountBI = BigInt(bailAmountMicro);
            for (let i = 0; i < 8; i++) amountArg[31 - i] = Number((amountBI >> BigInt(i * 8)) & 0xffn);

            const txObj = {
                from: userAddress, type: 'appl', appIndex: parseInt(assetId),
                appArgs: [methodSelector, recipientAddr, amountArg],
                note: new TextEncoder().encode(`BAIL_PAYMENT:${cardId}`)
            };
            
            if (walletProvider === 'nautilus') {
                const signed = await window.algo.signTxn([{ txn: algosdk.encodeObj(txObj), signers: [userAddress] }]);
                const { txId } = await window.algo.sendRawTransaction(signed[0]);
                txid = txId;
            } else if (walletProvider === 'walletconnect') {
                const sessions = signClient.session.getAll();
                const response = await signClient.request({
                    topic: sessions[0].topic, chainId: CONFIG.VOI_CHAIN_ID,
                    request: { method: "algo_signTxn", params: [[{ txn: btoa(String.fromCharCode(...algosdk.encodeObj(txObj))), signers: [userAddress] }]] }
                });
                const signedTxnBytes = new Uint8Array(atob(response[0]).split("").map(c => c.charCodeAt(0)));
                const netCfg = availableNetworks["Voi Mainnet"];
                const client = new algosdk.Algodv2("", netCfg.node_url, "");
                const { txId: broadcastId } = await client.sendRawTransaction(signedTxnBytes).do();
                txid = broadcastId;
            }
        }

        if (!txid) throw new Error("Transaction verification failed.");

        socket.send(JSON.stringify({
            type: "bail_card",
            payload: {
                card_id: parseInt(cardId),
                club_id: clubId,
                txid: txid,
                network: network
            }
        }));

        document.getElementById("portfolio-view-overlay")?.remove();
    } catch (err) {
        showToast(`Bail Request Failed: ${err.message}`, "error");
    }
    
    document.body.appendChild(overlay);
}
// Imported from criminality.js
window.openSecuritySentry = () => {
    const state = window.GetGameState();
    const club = globalClubs[state.employer_id];
    if (!club) return;

    const overlay = document.createElement("div");
    overlay.id = "security-sentry-overlay";
    overlay.className = "overlay";

    // Heuristic: Traps are items with "tripwire", "sentry", or "dog" in ID
    const availableTraps = [
        { id: "tripwire", name: "Laser Tripwire", desc: "+10% Heist Failure" },
        { id: "sentry_turret", name: "Sentry Turret", desc: "+25% Heist Failure" },
        { id: "guard_dog", name: "Bio-Guard Dog", desc: "Forces Jail on Failure" }
    ];

    const activeTraps = Object.entries(club.active_buffs || {})
        .filter(([key]) => key.startsWith("TRAP_"));

    overlay.innerHTML = `
        <div class="glass-panel" style="width: 550px; text-align: center; border-color: var(--neon-cyan);">
            <h2 style="color: var(--neon-cyan); letter-spacing: 2px;">🛡️ SECURITY SENTRY: ${club.name.toUpperCase()}</h2>
            <p style="font-size: 0.8em; opacity: 0.7; margin-bottom: 20px;">Deploy tactical hardware to protect the Treasury from heisters.</p>
            
            <div style="text-align: left; margin-bottom: 20px;">
                <small style="color: var(--neon-cyan); font-weight: bold; opacity: 0.5;">ACTIVE DEFENSES (${activeTraps.length}/3)</small>
                <div class="flex-col gap-5 mt-5">
                    ${activeTraps.length === 0 ? '<div style="opacity: 0.3; font-style: italic;">No active traps detected.</div>' : 
                      activeTraps.map(([id, type]) => `
                        <div class="player-item" style="padding: 8px 12px; border-color: var(--neon-green);">
                            <span>🛰️ ${type.toUpperCase()}</span>
                            <span style="color: var(--neon-green); font-size: 10px;">ONLINE</span>
                        </div>
                      `).join('')}
                </div>
            </div>

            <div style="text-align: left;">
                <small style="color: var(--neon-cyan); font-weight: bold; opacity: 0.5;">AVAILABLE HARDWARE</small>
                <div class="flex-col gap-10 mt-5">
                    ${availableTraps.map(trap => {
                        const count = state.inventory[trap.id] || 0;
                        return `
                            <div class="glass-panel p-10 m-0 flex-row justify-between align-center">
                                <div>
                                    <b>${trap.name}</b>
                                    <div style="font-size: 0.75em; opacity: 0.6;">${trap.desc}</div>
                                </div>
                                <div class="flex-row align-center gap-10">
                                    <span style="font-size: 11px; opacity: 0.8;">Owned: ${count}</span>
                                    <button class="outline" style="font-size: 10px; padding: 5px 15px;" 
                                            ${count === 0 || activeTraps.length >= 3 ? 'disabled' : ''} 
                                            onclick="deployTrap('${trap.id}')">DEPLOY</button>
                                </div>
                            </div>
                        `;
                    }).join('')}
                </div>
            </div>

            <button class="outline mt-20 w-full" onclick="document.getElementById('security-sentry-overlay').remove()">CLOSE TERMINAL</button>
        </div>
    `;
    document.body.appendChild(overlay);
}

window.deployTrap = (trapId) => { // This will be moved to criminality.js
    if (!socket || socket.readyState !== WebSocket.OPEN) return;
    
    showToast(`🛰️ Deploying ${trapId.replace('_', ' ')}...`, "info");
    socket.send(JSON.stringify({
        type: "use_item",
        payload: {
            item_id: trapId
        }
    }));
    document.getElementById("security-sentry-overlay")?.remove();
}

window.openBountyBoard = async () => { // This will be moved to criminality.js
    const state = window.GetGameState();
    const myWanted = state.wanted_level || 0;
    const isHunter = myWanted <= 2;
    
    const overlay = document.createElement("div");
    overlay.id = "bounty-board-overlay";
    overlay.className = "overlay";
    
    const outlaws = lastLobbyPlayers.filter(p => (p.wanted_level || 0) >= 10);
    
    let targetsHtml = "";
    if (outlaws.length === 0) {
        targetsHtml = `<div style="padding: 40px; opacity: 0.5;">No active bounties in this sector.</div>`;
    } else {
        // Pre-resolve envoi names
        const wallets = outlaws.map(p => p.wallet);
        await Promise.all(wallets.map(w => resolveEnvoiName(w)));

        outlaws.forEach(p => {
            const name = getCachedEnvoiName(p.wallet);
            const bounty = p.wanted_level * 50;
            const isMe = p.id === myClientId;
            
            targetsHtml += `
                <div class="player-item" style="padding: 15px; border-color: #ffd700;">
                    <div style="text-align: left;">
                        <b style="color: #ffd700;">${name}</b>
                        <div style="font-size: 0.75em; opacity: 0.6;">WANTED LEVEL: ${p.wanted_level}</div>
                    </div>
                    <div style="text-align: right;">
                        <div style="color: var(--neon-green); font-weight: bold;">${bounty} $VBV</div>
                        ${isHunter && !isMe ? `<button class="outline" style="font-size: 10px; padding: 6px 12px; border-color: #ffd700; color: #ffd700;" onclick="document.getElementById('bounty-board-overlay').remove(); sendChallenge('${p.id}')">HUNT TARGET</button>` : ''}
                        ${isMe ? `<span style="font-size: 10px; color: #ff4b4b;">YOU ARE THE TARGET</span>` : ''}
                    </div>
                </div>`;
        });
    }

    overlay.innerHTML = `
        <div class="glass-panel" style="width: 500px; text-align: center; border-color: #ffd700;">
            <h2 style="color: #ffd700; letter-spacing: 3px;">🎯 BOUNTY BOARD</h2>
            <p style="font-size: 0.8em; opacity: 0.7; margin-bottom: 20px;">High-infamy outlaws currently in the lobby. Hunters (Wanted ≤ 2) earn 50 $VBV per Wanted point on victory.</p>
            <div class="flex-col gap-10" style="max-height: 400px; overflow-y: auto; padding-right: 5px;">${targetsHtml}</div>
            <button class="outline mt-20 w-full" onclick="document.getElementById('bounty-board-overlay').remove()">CLOSE BOARD</button>
        </div>`;
    document.body.appendChild(overlay);
}

window.openBlackMarket = async () => { // This will be moved to economy.js
    const state = window.GetGameState();
    const wanted = state.wanted_level || 0;
    const cunning = state.cunning || 0;

    if (wanted < 5 || cunning < 10) {
        showToast("❌ Access Denied: Black Market requires Wanted Level 5+ and Cunning 10+.", "error");
        return;
    }

    const overlay = document.createElement("div");
    overlay.id = "black-market-overlay";
    overlay.className = "overlay";

    let html = `
		<div class="economy-panel black-market" style="width: 650px;">
			<div class="market-header">
				<span class="market-title">THE UNDERWORLD</span>
				<div class="access-level">RESTRICTED ACCESS</div>
			</div>
			
			<div class="market-notice mb-20">
				<div class="notice-icon">💀</div>
				<div class="notice-title">DEFAULTED COLLATERAL</div>
				<p class="notice-text">Assets listed here were seized from failed loans. Purchasing them triggers an infamy penalty but offers extreme tactical discounts.</p>
			</div>

			<div id="black-market-grid" class="market-grid" style="max-height: 400px; overflow-y: auto;">
    `;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/black-market?wallet=${userAddress}`);
        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(errorText);
        }
        const blackMarketItems = await response.json();

        if (blackMarketItems.length === 0) {
            html += `<div style="padding: 40px; opacity: 0.5;">No hot items currently available. Check back later.</div>`;
        } else {
            // Pre-resolve envoi names for all borrowers
            const borrowerWallets = new Set(blackMarketItems.map(item => item.borrower_wallet));
            await Promise.all(Array.from(borrowerWallets).map(w => resolveEnvoiName(w)));

            for (const item of blackMarketItems) {
                const cardName = item.collateral_bundle.card_id ? `CARD-${item.collateral_bundle.card_id}` : 'N/A';
                const weaponName = item.collateral_bundle.weapon_id || 'N/A';
                const faceplateName = item.collateral_bundle.faceplate_id || 'N/A';
                const borrowerName = getCachedEnvoiName(item.borrower_wallet);

                // Scavenger price is 75% of the original repayment amount
                const scavengePrice = (item.repayment_amount * 0.75) / 1000000; // Convert micro-units to VBV

                html += `
                    <div class="player-item" style="padding: 15px; border-color: #ff4b4b;">
                        <div style="text-align: left;">
                            <b style="color: var(--neon-cyan);">Collateral from ${borrowerName}</b>
                            <div style="font-size: 0.75em; opacity: 0.6;">
                                Card: ${cardName} <br>
                                Weapon: ${weaponName} <br>
                                Faceplate: ${faceplateName}
                            </div>
                        </div>
                        <div style="text-align: right;">
                            <div style="color: var(--neon-green); font-weight: bold;">${scavengePrice.toFixed(2)} $VBV</div>
                            <button class="outline" style="font-size: 9px; padding: 4px 8px; border-color: #ff4b4b; color: #ff4b4b;" 
                                    onclick="buyBlackMarketItem('${item.id}', ${scavengePrice})">BUY (RISKY)</button>
                        </div>
                    </div>
                `;
            }
        }
    } catch (err) {
        showToast(`❌ Black Market Access Failed: ${err.message}`, "error");
        html += `<div style="padding: 40px; opacity: 0.5; color: #ff4b4b;">Error loading Black Market: ${err.message}</div>`;
    }

    html += `
            </div>
            <button class="outline mt-20" onclick="document.getElementById('black-market-overlay').remove()">CLOSE</button>
        </div>
    `;

    overlay.innerHTML = html;
    document.body.appendChild(overlay);
}

/** // This will be moved to economy.js
 * Art Gallery Interface: Consignment and Auctions. // Imported from economy.js
 */
async function openArtGalleryOverlay() {
    const overlay = document.createElement("div");
    overlay.id = "art-gallery-overlay";
    overlay.className = "overlay";
    
    overlay.innerHTML = `
        <div class="economy-panel gallery-panel" style="width: 900px; max-height: 85vh; overflow-y: auto;">
            <div class="economy-header">
                <span class="economy-title">THE ART GALLERY</span>
                <div class="flex-row gap-15">
                    <button class="outline x-small" onclick="openConsignmentOverlay()">CONSIGN ITEM</button>
                    <button class="outline x-small border-error" onclick="document.getElementById('art-gallery-overlay').remove()">CLOSE</button>
                </div>
            </div>

            <div class="auction-gallery">
                <div class="gallery-header">
                    <p class="opacity-7 italic font-size-0-85em">Tactical assets and rare artifacts up for public auction. All sales support the Industrial Loop.</p>
                </div>
                
                <div id="gallery-items-container" class="gallery-grid">
                    <div class="grid-span-all opacity-5 py-40 italic">Decrypting auction datastreams...</div>
                </div>
            </div>
        </div>
    `;

    document.body.appendChild(overlay);
    loadGalleryItems();
}

window.loadGalleryItems = async () => { // This will be moved to economy.js
    const container = document.getElementById("gallery-items-container");
    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/auctions`);
        const auctions = await response.json();

        if (!auctions || auctions.length === 0) {
            container.innerHTML = `<div style="grid-column: 1/-1;" class="opacity-5 py-40 italic">The gallery floor is currently vacant. Check back during peak match hours.</div>`;
            return;
        }

        container.innerHTML = auctions.map(a => {
            const timeRemaining = Math.max(0, new Date(a.ends_at) - new Date());
            const hours = Math.floor(timeRemaining / 3600000);
            const mins = Math.floor((timeRemaining % 3600000) / 60000);
            
            return `
                <div class="gallery-grid__item-bundle animate-slide-up">
                    <div class="item-image">
                        <img src="Assets/Images/portraits/placeholder.webp" alt="Exhibit">
                    </div>
                    <div class="item-info text-left">
                        <div class="item-title font-bold text-neon-cyan">${a.bundle.weapon_id ? a.bundle.weapon_id.replace(/_/g, ' ') : 'Tactical Artifact'}</div>
                        <div class="item-description font-size-0-8em opacity-6">Seller: ${a.seller_name}</div>
                    </div>
                    <div class="auction-info mt-10">
                        <div class="current-bid">
                            <span class="bid-label">HIGHEST BID</span>
                            <span class="bid-amount text-neon-green">${(a.current_bid / 1000000).toFixed(1)} $VBV</span>
                        </div>
                        <div class="time-remaining">
                            <span class="time-label">REMAINING</span>
                            <span class="time-value">${hours}h ${mins}m</span>
                        </div>
                    </div>
                    <button class="outline mt-15 w-full border-cyan text-neon-cyan" onclick="promptBid('${a.id}', ${a.current_bid})">PLACE BID</button>
                </div>`;
        }).join('');
    } catch (err) {
        container.innerHTML = `<div style="grid-column: 1/-1;" class="text-error py-40">Gallery Indexer Unreachable.</div>`;
    }
}

window.promptBid = async (auctionId, currentBidMicro) => { // This will be moved to economy.js
    const minBid = (currentBidMicro + (currentBidMicro * 0.05)) / 1000000;
    const bidAmount = prompt(`Enter your bid in $VBV (Minimum: ${minBid.toFixed(2)}):`, minBid.toFixed(2));
    
    if (bidAmount && !isNaN(bidAmount)) {
        const bidMicro = Math.round(parseFloat(bidAmount) * 1000000);
        if (bidMicro <= currentBidMicro) {
            showToast("Bid must be at least 5% higher than current.", "error");
            return;
        }
        
        showToast("⚡ Authorizing gallery bid...", "info");
        const state = window.GetGameState();
        
        try {
            const response = await fetch(`${CONFIG.API_BASE}/api/auctions/bid`, {
                method: "POST",
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    auction_id: auctionId,
                    wallet: userAddress,
                    amount: bidMicro,
                    network: state.network
                })
            });
            
            if (response.ok) {
                showToast("✅ Bid accepted! You are now the highest bidder.", "success");
                loadGalleryItems();
            } else {
                const err = await response.text();
                showToast(`❌ Bid Rejected: ${err}`, "error");
            }
        } catch (e) {
            showToast("Gallery connection failed.", "error");
        }
    }
}

/**
 * Consignment Flow: Interface for listing items in the Art Gallery. // Imported from economy.js
 */
function openConsignmentOverlay() {
    const state = window.GetGameState();
    const overlay = document.createElement("div");
    overlay.id = "consignment-overlay";
    overlay.className = "overlay";

    // Filter inventory for listable assets (Cards and known items)
    const listableItems = Object.entries(state.inventory || {}).filter(([id, qty]) => qty > 0);

    overlay.innerHTML = `
        <div class="economy-panel consignment-panel" style="width: 550px;">
            <div class="market-header">
                <span class="market-title text-neon-purple">ASSET CONSIGNMENT</span>
                <div class="access-level">GALLERY PROTOCOL</div>
            </div>

            <div class="p-20">
                <p class="opacity-6 font-size-0-85em mb-20">Select an asset from your collection to list on the public auction floor. 10% commission applies on successful settlement.</p>
                
                <div class="flex-col gap-10 mb-20" style="max-height: 300px; overflow-y: auto;">
                    ${listableItems.length === 0 ? '<div class="opacity-3 italic py-20">No listable tactical assets detected.</div>' : 
                        listableItems.map(([id, qty]) => `
                            <div class="portfolio-item glass-panel m-0 p-10 flex-row justify-between align-center pointer" onclick="selectConsignmentItem('${id}')">
                                <div class="flex-row align-center gap-10">
                                    <div class="item-icon font-size-1-2em">📦</div>
                                    <div class="text-left">
                                        <div id="item-name-${id}" class="font-bold text-neon-cyan">${id.replace(/_/g, ' ').toUpperCase()}</div>
                                        <div class="font-size-0-75em opacity-5">Available: ${qty}</div>
                                    </div>
                                </div>
                                <input type="radio" name="consignment-target" value="${id}">
                            </div>
                        `).join('')}
                </div>

                <div id="consignment-pricing" class="hidden animate-slide-up">
                    <div class="glass-panel p-15 border-cyan">
                        <label class="font-size-0-8em text-neon-cyan font-bold block mb-5">STARTING BID ($VBV)</label>
                        <input type="number" id="consignment-bid-input" class="glass-input w-full mb-10" placeholder="e.g. 500.00" step="0.1">
                        <small class="opacity-5 italic">Note: Auctions run for 24 hours from timestamp of listing.</small>
                    </div>
                    
                    <div class="flex-row gap-15 mt-20">
                        <button class="outline w-full" onclick="document.getElementById('consignment-overlay').remove()">ABORT</button>
                        <button class="w-full bg-neon-purple text-dark font-bold" onclick="submitConsignment()">LIST ASSET</button>
                    </div>
                </div>
            </div>
        </div>
    `;

    document.body.appendChild(overlay);
}

window.selectConsignmentItem = (id) => { // This will be moved to economy.js
    const radio = document.querySelector(`input[value="${id}"]`);
    if (radio) radio.checked = true;
    document.getElementById("consignment-pricing").classList.remove("hidden");
}

async function submitConsignment() {
    const selectedInput = document.querySelector('input[name="consignment-target"]:checked');
    const bidInput = document.getElementById("consignment-bid-input");
    
    if (!selectedInput || !bidInput.value) {
        showToast("Please select an item and enter a starting bid.", "error");
        return;
    }

    const itemId = selectedInput.value;
    const bidBase = parseFloat(bidInput.value);
    if (isNaN(bidBase) || bidBase <= 0) {
        showToast("Invalid starting bid.", "error");
        return;
    }

    showToast("⚡ Authorizing consignment protocol...", "info");
    
    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/auctions/create`, {
            method: "POST",
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                wallet: userAddress,
                item_id: itemId,
                starting_bid: Math.round(bidBase * 1000000), // Convert to micro-units
                territory_id: "the_art_gallery"
            })
        });

        if (response.ok) {
            showToast(`✅ Asset listed! ${itemId.replace(/_/g, ' ')} is now on the auction floor.`, "success");
            document.getElementById("consignment-overlay").remove();
            loadGalleryItems(); // Refresh the gallery list
        } else {
            const err = await response.text();
            showToast(`❌ Listing Failed: ${err}`, "error");
        }
    } catch (e) {
        showToast("Gallery connection failure.", "error");
    }
}

window.openConsignmentOverlay = openConsignmentOverlay; // This will be moved to economy.js
window.openArtGalleryOverlay = openArtGalleryOverlay; // This will be moved to economy.js
window.promptBid = promptBid; // This will be moved to economy.js

async function buyBlackMarketItem(loanId, price) { // This will be moved to economy.js
    if (!userAddress) return showToast("Connect wallet first", "error");
    if (!confirm(`Are you sure you want to buy this item for ${price.toFixed(2)} $VBV? This will increase your Wanted Level.`)) return;

    try {
        const state = window.GetGameState();
        const network = state.network;

        const response = await fetch(`${CONFIG.API_BASE}/api/black-market/buy`, {
            method: "POST",
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ wallet: userAddress, loan_id: loanId, network: network })
        });

        if (response.ok) {
            const result = await response.json();
            showToast(`🏴‍☠️ ${result.message}`, "success");
            document.getElementById("black-market-overlay")?.remove();
            syncUI();
        } else {
            const err = await response.text();
            showToast(`❌ Black Market Purchase Failed: ${err}`, "error");
        }
    } catch (err) {
        showToast(`Purchase Failed: ${err.message}`, "error");
    }
}

window.openRumorMill = async () => { // This will be moved to criminality.js
    const state = window.GetGameState();
    const playerRewards = state.rewards[CONFIG.VBV_ASSET_ID] || 0;
    const rumorCost = 500; // Matches server-side cost

    if (playerRewards < rumorCost) {
        showToast(`❌ Insufficient $VBV. Spreading a rumor costs ${rumorCost} $VBV.`, "error");
        return;
    }

    const overlay = document.createElement("div");
    overlay.id = "rumor-mill-overlay";
    overlay.className = "overlay";

    let targetsHtml = '';
    if (lastLobbyPlayers.length === 0) {
        targetsHtml = `<div style="padding: 20px; opacity: 0.5;">No other players in the lobby to spread rumors about.</div>`;
    } else {
        // Filter out self
        const otherPlayers = lastLobbyPlayers.filter(p => p.id !== myClientId);
        if (otherPlayers.length === 0) {
            targetsHtml = `<div style="padding: 20px; opacity: 0.5;">No other players in the lobby to spread rumors about.</div>`;
        } else {
            // Pre-resolve envoi names for all targets
            const targetWallets = new Set(otherPlayers.map(p => p.wallet));
            await Promise.all(Array.from(targetWallets).map(w => resolveEnvoiName(w)));

            targetsHtml = otherPlayers.map(p => {
                const targetName = getCachedEnvoiName(p.wallet);
                return `
                    <div class="player-item" style="padding: 10px; border-color: var(--glass-border);">
                        <div style="text-align: left;">
                            <b style="color: var(--neon-cyan);">${targetName}</b>
                            <div style="font-size: 0.75em; opacity: 0.6;">${p.reputation} REP | ${p.wins} WINS</div>
                        </div>
                        <div class="flex-row gap-5">
                            <button class="outline" style="font-size: 9px; padding: 4px 8px; border-color: var(--neon-green); color: var(--neon-green);" 
                                    onclick="spreadRumor('${p.wallet}', 'positive', 1.1, 60)">+ POSITIVE</button>
                            <button class="outline" style="font-size: 9px; padding: 4px 8px; border-color: #ff4b4b; color: #ff4b4b;" 
                                    onclick="spreadRumor('${p.wallet}', 'negative', 0.9, 60)">- NEGATIVE</button>
                        </div>
                    </div>
                `;
            }).join('');
        }
    }

    overlay.innerHTML = `
        <div class="glass-panel" style="width: 600px; text-align: center;">
            <h2 style="color: var(--neon-green); letter-spacing: 3px;">RUMOR MILL</h2>
            <p style="font-size: 0.8em; opacity: 0.7; margin-bottom: 20px;">Influence market sentiment. Cost: <b style="color: var(--neon-green);">${rumorCost} $VBV</b></p>
            <div class="flex-col gap-10" style="max-height: 400px; overflow-y: auto; padding-right: 5px;">
                ${targetsHtml}
            </div>
            <button class="outline mt-20" onclick="document.getElementById('rumor-mill-overlay').remove()">CLOSE</button>
        </div>
    `;

    document.body.appendChild(overlay);
}

window.spreadRumor = async (targetWallet, type, strength, durationMinutes) => { // This will be moved to criminality.js
    if (!socket || socket.readyState !== WebSocket.OPEN) return showToast("❌ Not connected to server.", "error");
    if (!userAddress) return showToast("❌ Connect wallet first.", "error");

    const rumorCost = 500; // Matches server-side cost
    const state = window.GetGameState();
    const playerRewards = state.rewards[CONFIG.VBV_ASSET_ID] || 0;

    if (playerRewards < rumorCost) {
        showToast(`❌ Insufficient $VBV. Spreading a rumor costs ${rumorCost} $VBV.`, "error");
        return;
    }

    if (!confirm(`Are you sure you want to spread a ${type} rumor about ${getCachedEnvoiName(targetWallet)} for ${rumorCost} $VBV?`)) return;

    try {
        showToast(`📢 Spreading ${type} rumor about ${getCachedEnvoiName(targetWallet)}...`, "info");
        
        socket.send(JSON.stringify({
            type: "spread_rumor",
            payload: {
                target_wallet: targetWallet,
                type: type,
                strength: strength,
                duration_minutes: durationMinutes
            }
        }));
        
        document.getElementById("rumor-mill-overlay")?.remove();
    } catch (err) {
        showToast(`❌ Failed to spread rumor: ${err.message}`, "error");
    }
}

window.openTrophyView = () => { // This will be moved to criminality.js
    openSocialPanelOverlay('achievements');
}

/**
 * Opens the integrated Social Hub featuring Alliances, Career paths, and Achievements. // Imported from criminality.js
 * Utilizes orphaned _social.scss styles for immersive hierarchy.
 */
async function openSocialPanelOverlay(initialTab = 'alliances') {
    const state = window.GetGameState();
    const overlay = document.createElement("div");
    overlay.id = "social-hub-overlay";
    overlay.className = "overlay";

    overlay.innerHTML = `
        <div class="social-panel glass-panel" style="width: 750px;">
            <div class="social-header">
                <span class="social-title">NEON SOCIAL HUB</span>
                <div class="social-stats">
                    <div class="stat-item">
                        <div class="stat-label">MOJO</div>
                        <div class="stat-value">${state.mojo || 0}</div>
                    </div>
                    <div class="stat-item">
                        <div class="stat-label">REP</div>
                        <div class="stat-value">${state.reputation || 0}</div>
                    </div>
                </div>
            </div>

            <div class="flex-row justify-center gap-10 mb-20">
                <button id="social-tab-alliances" class="tab-btn ${initialTab === 'alliances' ? 'active' : ''}" onclick="switchSocialTab('alliances')">🤝 ALLIANCES</button>
                <button id="social-tab-career" class="tab-btn ${initialTab === 'career' ? 'active' : ''}" onclick="switchSocialTab('career')">💼 CAREER</button>
                <button id="social-tab-achievements" class="tab-btn ${initialTab === 'achievements' ? 'active' : ''}" onclick="switchSocialTab('achievements')">🏆 VALOR</button>
            </div>

            <div id="social-content-hub" class="flex-col gap-15" style="max-height: 500px; overflow-y: auto; padding-right: 5px;">
                <!-- Content injected by switchSocialTab -->
            </div>

            <button class="outline mt-20 w-full" onclick="document.getElementById('social-hub-overlay').remove()">DISCONNECT HUB</button>
        </div>
    `;

    document.body.appendChild(overlay);
    switchSocialTab(initialTab);
}

window.switchSocialTab = async (tab) => { // This will be moved to criminality.js
    const container = document.getElementById("social-content-hub");
    if (!container) return;

    // Update Tab Styles
    document.querySelectorAll('#social-hub-overlay .tab-btn').forEach(b => b.classList.remove('active'));
    const tabBtn = document.getElementById(`social-tab-${tab}`);
    if (tabBtn) tabBtn.classList.add('active');

    const state = window.GetGameState();
    container.innerHTML = `<div class="opacity-5 py-40 italic">Decrypting social datastreams...</div>`;

    if (tab === 'alliances') {
        const otherPlayers = lastLobbyPlayers.filter(p => p.id !== myClientId);
        if (otherPlayers.length > 0) {
            await Promise.all(otherPlayers.map(p => resolveEnvoiName(p.wallet)));
        }

        // Filter for existing alliances (simulated from portfolio/employment state)
        const myClub = Object.values(globalClubs).find(c => c.id === state.employer_id);
        const allianceWallets = myClub ? Object.keys(myClub.members || {}) : [];

        const renderConnection = (p, isAlly) => `
            <div class="connection-item glass-panel m-0 ${isAlly ? 'border-cyan' : ''}">
                <div class="connection-avatar">
                    <img src="${p.avatar_url || 'Assets/Images/portraits/placeholder.webp'}" alt="Entity">
                </div>
                <div class="connection-info text-left">
                    <div class="connection-name font-bold ${isAlly ? 'text-neon-cyan' : ''}">${getCachedEnvoiName(p.wallet)}</div>
                    <div class="connection-role font-size-0-75em opacity-6">${p.social_rank} | ${p.job_role || 'Freelancer'}</div>
                    <div class="connection-status online font-size-0-7em">ACTIVE LINK</div>
                </div>
                <div class="connection-actions">
                    <button class="action-btn message" onclick="document.getElementById('social-hub-overlay').remove(); sendChallenge('${p.id}')" title="Challenge duel"></button>
                    ${!isAlly ? `<button class="action-btn invite" onclick="proposeAlliance('${p.id}')" title="Propose Alliance"></button>` : ''}
                    <button class="action-btn block" onclick="showToast('Entity communication restricted.', 'info')" title="Block stream"></button>
                </div>
            </div>`;

        const allies = otherPlayers.filter(p => allianceWallets.includes(p.wallet?.toLowerCase()));
        const others = otherPlayers.filter(p => !allianceWallets.includes(p.wallet?.toLowerCase()));

        container.innerHTML = `
            <div class="social-network flex-col gap-20">
                <div class="alliance-management">
                    <div class="network-header mb-10">
                        <span class="network-title text-neon-cyan font-bold letter-spacing-1">CONFIRMED ALLIANCES</span>
                        <div class="network-stats opacity-5 font-size-0-8em">STRENGTH: ${allies.length}</div>
                    </div>
                    <div class="connections-list">
                        ${allies.length === 0 ? '<div class="opacity-3 p-20 italic font-size-0-9em border-glass">No active alliance contracts found.</div>' : 
                            allies.map(p => renderConnection(p, true)).join('')}
                    </div>
                </div>

                <div class="sector-discovery">
                    <div class="network-header mb-10">
                        <span class="network-title text-neon-purple font-bold letter-spacing-1">SECTOR ENTITIES</span>
                        <div class="network-stats opacity-5 font-size-0-8em">DETECTED: ${others.length}</div>
                    </div>
                    <div class="connections-list">
                        ${others.length === 0 ? '<div class="opacity-3 p-20 italic font-size-0-9em">Scanning... no other entities in proximity.</div>' : 
                            others.map(p => renderConnection(p, false)).join('')}
                    </div>
                </div>
            </div>`;
    } else if (tab === 'career') {
        const tiers = [
            { name: "Iron", mojo: 0, desc: "A nobody in the neon gutter.", icon: "🌑" },
            { name: "Bronze", mojo: 100, desc: "A regular face at the local shops.", icon: "🥉" },
            { name: "Silver", mojo: 300, desc: "Gaining recognition in the sector.", icon: "🥈" },
            { name: "Gold", mojo: 600, desc: "An icon of the regional circuit.", icon: "🥇" },
            { name: "Diamond", mojo: 1000, desc: "Arena legend. The elite respect you.", icon: "💎" }
        ];

        container.innerHTML = `
            <div class="career-system">
                <div class="career-header">
                    <span class="career-title">PATH: <span class="job-role job-role--${(state.job_role || 'Freelancer').toLowerCase()}">${state.job_role || 'Freelancer'}</span></span>
                    <div class="career-level">MOJO ${state.mojo || 0}</div>
                </div>
                <div class="career-path">
                    ${tiers.map(t => {
                        const isCurrent = (state.mojo || 0) >= t.mojo && (state.mojo || 0) < (tiers[tiers.indexOf(t)+1]?.mojo || 9999);
                        const isCompleted = (state.mojo || 0) >= (tiers[tiers.indexOf(t)+1]?.mojo || 9999);
                        const isLocked = (state.mojo || 0) < t.mojo;
                        return `
                            <div class="career-tier ${isCurrent ? 'current' : ''} ${isCompleted ? 'completed' : ''} ${isLocked ? 'locked' : ''}">
                                <div class="tier-content">
                                    <div class="tier-icon">${t.icon}</div>
                                    <div class="tier-info">
                                        <div class="tier-name">${t.name}</div>
                                        <div class="tier-description">${t.desc}</div>
                                        <div class="tier-requirements">
                                            <span class="requirement ${!isLocked ? 'completed' : ''}">REQ: ${t.mojo} MOJO</span>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        `;
                    }).join('')}
                </div>
            </div>
        `;
    } else if (tab === 'achievements') {
        const unlocked = new Set(state.achievements || []);
        const trophyCatalog = [
            { id: "FIRST_VICTORY", name: "First Victory", description: "Win your first match.", tier: 1 },
            { id: "TOURNAMENT_CHAMPION", name: "Tournament Champion", description: "Win a tournament.", tier: 2 },
            { id: "FIRST_HEIST", name: "First Heist", description: "Complete a successful Club heist.", tier: 1 },
            { id: "OUTLAW_SLAYER", name: "Outlaw Slayer", description: "Defeat a high-infamy opponent.", tier: 2 },
            { id: "ARENA_LEGEND", name: "Arena Legend", description: "Achieve legendary status in the arena.", tier: 3 },
            { id: "REHABILITATED", name: "Rehabilitated", description: "Pay off your courthouse fine and reset wanted level.", tier: 2 },
            { id: "GOVERNOR", name: "Governor", description: "Control 2+ territories as a club leader.", tier: 3 }
        ];

        container.innerHTML = `
            <div class="achievement-system">
                <div class="achievements-header">
                    <span class="achievements-title">HALL OF VALOR</span>
                    <div class="achievements-progress">UNLOCKED: <span class="progress-text">${unlocked.size}/${trophyCatalog.length}</span></div>
                </div>
                <div class="achievements-grid">
                    ${trophyCatalog.map(trophy => {
                        const hasUnlocked = unlocked.has(trophy.id);
                        return `
                            <div class="trophy-badge tier-${trophy.tier} ${hasUnlocked ? 'unlocked' : 'locked'}">
                                <div class="badge-icon ${hasUnlocked ? 'unlocked' : ''}">${hasUnlocked ? '🏆' : ''}</div>
                                <div class="badge-name">${trophy.name}</div>
                                <div class="badge-description">${trophy.description}</div>
                            </div>
                        `;
                    }).join('')}
                </div>
            </div>
        `;
    }
}

window.openSocialPanelOverlay = openSocialPanelOverlay; // Imported from criminality.js
window.switchSocialTab = switchSocialTab; // Imported from criminality.js

function tradeShares(entityId, action, amount) {
    if (!socket || socket.readyState !== WebSocket.OPEN) return;

    socket.send(JSON.stringify({
        type: "trade_shares",
        payload: {
            entity_id: entityId,
            action: action,
            amount: amount
        }
    }));

    document.getElementById("portfolio-view-overlay")?.remove();
}

// Function to show the main game container and hide other overlays // Imported from ui.js
window.showMainGameContainer = () => {
    document.getElementById("main-game-container").classList.remove("hidden");
}

// Placeholder for setupCropEvents - assuming it's defined elsewhere or will be moved here // Imported from deck.js
// --- Avatar Setup & Cropping Logic ---

async function refreshInventory() {
    if (!userAddress) return;
    
    const grid = document.getElementById("avatar-grid");
    const loader = document.getElementById("setup-loader");
    if (loader) loader.classList.remove("hidden");

    userNFTs = []; // Clear for aggregate fetch
    const state = window.GetGameState();
    
    // 1. Compile list of wallets to scan.
    // The primary userAddress is assumed to be on the currently selected game network.
    const primaryNetworkShortName = state.network; // e.g., "VOI" or "ALGO"
    const sources = [{ address: userAddress, chain: primaryNetworkShortName }];
    linkedWallets.forEach(w => sources.push(w));

    // 2. Fetch from all sources in parallel
    await Promise.all(sources.map(async (src) => {
        try {
            // Use Indexer URL from admin availableNetworks if available, otherwise fallback
            const networkConfig = getNetworkConfig(src.chain); // Use helper for consistency
            const baseUrl = networkConfig ? networkConfig.indexer_url : "";

            if (!baseUrl) {
                console.warn(`[FETCH] No indexer URL found for network ${src.chain}. Skipping NFT fetch for ${src.address}.`);
                return; // Skip if no base URL
            }

            if (src.chain === "SOL") {
                // Solana DAS API specific fetch via POST to NodeURL
                const solRes = await fetch(baseUrl, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ jsonrpc: "2.0", id: 1, method: "getAssetsByOwner", params: { ownerAddress: src.address, page: 1, limit: 50 }})
                });
                const solData = await solRes.json();
                if (solData.result && solData.result.items) userNFTs = [...userNFTs, ...solData.result.items];
                return;
            }

            const response = await fetch(`${baseUrl}/tokens?owner=${src.address}`);
            if (!response.ok) {
                console.warn(`[FETCH] Indexer returned error for ${src.address}: ${response.status}`);
                return;
            }

            const data = await response.json();
            if (data.tokens) userNFTs = [...userNFTs, ...data.tokens];
        } catch (err) { console.warn(`[FETCH] Source ${src.address} failed:`, err); }
    }));

    renderAvatarGrid(userNFTs);
    updateLinkedWalletsUI();
    if (loader) loader.classList.add("hidden");
}

window.renderAvatarGrid = (nfts) => { // This will be moved to deck.js
    const grid = document.getElementById("avatar-grid");
    if (!grid) return;
    grid.innerHTML = "";
    
    // Filter out banned avatars
    nfts.forEach(nft => {
        let meta = {};
        try { meta = JSON.parse(nft.metadata || "{}"); } catch(e) {}
        const url = meta.image || "";
        if (!url) return;
        
        // Check if this URL is banned
        const state = window.GetGameState();
        const isBanned = state.banned_avatars && state.banned_avatars[url];
        const item = document.createElement("div"); // Create the element regardless
        item.className = "avatar-item";
        item.style.backgroundImage = `url(${url})`;
        item.onclick = () => selectAvatar(url);
        grid.appendChild(item);
    });
}

window.applyAvatarFilters = () => { // This will be moved to deck.js
    const search = document.getElementById("avatar-search").value.toLowerCase();
    const sort = document.getElementById("avatar-sort").value;
    
    let filtered = userNFTs.filter(nft => {
        let meta = {};
        try { meta = JSON.parse(nft.metadata || "{}"); } catch(e) {}
        return (meta.name || "").toLowerCase().includes(search);
    });
    
    if (sort === "oldest") {
        filtered.sort((a, b) => a.mintRound - b.mintRound);
    } else if (sort === "newest") {
        filtered.sort((a, b) => b.mintRound - a.mintRound);
    }
    
    renderAvatarGrid(filtered);
}

window.updateDynamicArenaFloor = (state) => { // This will be moved to ui.js
function updateDynamicArenaFloor(state) { 
    let texture = "var(--texture-solo)"; // Default AI/Solo

    if (state.phase === "TournamentLobby") {
        // Always show a tournament background in the tournament lobby
        texture = "var(--texture-tournament)";
    } else if (state.phase === "Active") {
        if (state.multiplayer) {
            if (state.tournament && state.tournament.active) {
                const currentRound = state.tournament.current_round;
                const participants = state.tournament.participants ? state.tournament.participants.length : 8;
                const maxRounds = Math.log2(participants); // 8 = 3 rounds, 16 = 4 rounds

                if (currentRound === maxRounds) {
                    texture = "var(--texture-final)";
                } else if (currentRound === maxRounds - 1) {
                    texture = "var(--texture-semi)";
                } else {
                    texture = "var(--texture-tournament)";
                }
            } else {
                // Standard 2 Player Match (Challenge)
                texture = "var(--texture-challenge)";
            }
        }
    }

    // Apply to body background
    document.body.style.backgroundImage = `${texture}, radial-gradient(circle at top center, #1a0b2e, var(--bg-dark), #000000)`;
}

window.selectAvatar = (url) => { // This will be moved to deck.js
    const preview = document.getElementById("avatar-preview-section");
    const img = document.getElementById("crop-image");
    if (!preview || !img) return;
    
    currentAvatarUrl = url;
    img.src = url;
    
    // Pre-populate gloat from cache
    const cachedGloat = localStorage.getItem("vbabes_gloat_msg") || "";
    document.getElementById("gloat-message-input").value = cachedGloat;

    preview.classList.remove("hidden");
    // Calibration is handled by the img.onload listener in setupCropEvents
}

window.setupCropEvents = () => { // This will be moved to deck.js
    const frame = document.getElementById("crop-frame");
    const img = document.getElementById("crop-image");
    const slider = document.getElementById("zoom-slider");
    const zoomVal = document.getElementById("zoom-val");
    const confirmBtn = document.getElementById("confirm-avatar-btn");
    
    if (!frame || !img || !slider || !confirmBtn) return;
    if (isCropInitialized) return; // Prevent duplicate global listeners
    isCropInitialized = true;

    let isDragging = false;
    let startX, startY;

    const updateTransform = () => {
        img.style.transform = `translate(${cropState.x}px, ${cropState.y}px) scale(${cropState.zoom})`;
    };

    // ASPECT RATIO & INITIAL CALIBRATION: Ensure image covers the 220px circle frame
    img.onload = () => {
        const frameSize = 220; // Diameter of the circle
        const w = img.naturalWidth;
        const h = img.naturalHeight;

        // Calculate minimal scale to completely fill the frame (CSS 'cover' behavior)
        const scaleW = frameSize / w;
        const scaleH = frameSize / h;
        const baseScale = Math.max(scaleW, scaleH);

        // Initialize state variables for pan/zoom logic
        cropState.zoom = baseScale;
        cropState.x = (frameSize - (w * baseScale)) / 2;
        cropState.y = (frameSize - (h * baseScale)) / 2;

        // Sync UI Sliders
        slider.min = baseScale.toFixed(2);
        slider.max = (baseScale * 4).toFixed(2);
        slider.value = baseScale;
        if (zoomVal) zoomVal.innerText = "1.0x";
        
        updateTransform();
    };

    slider.oninput = () => {
        cropState.zoom = parseFloat(slider.value);
        const relZoom = cropState.zoom / parseFloat(slider.min);
        if (zoomVal) zoomVal.innerText = relZoom.toFixed(1) + "x";
        updateTransform();
    };

    frame.onmousedown = (e) => {
        if (e.button !== 0) return; // Only primary mouse button
        isDragging = true;
        startX = e.clientX - cropState.x;
        startY = e.clientY - cropState.y;
        frame.style.cursor = "grabbing";
    };

    window.addEventListener('mousemove', (e) => {
        if (!isDragging) return;
        cropState.x = e.clientX - startX;
        cropState.y = e.clientY - startY;
        updateTransform();
    });

    window.addEventListener('mouseup', () => {
        isDragging = false;
        if (frame) frame.style.cursor = "grab";
    });

    // Mobile Touch Support
    frame.ontouchstart = (e) => {
        isDragging = true;
        startX = e.touches[0].clientX - cropState.x;
        startY = e.touches[0].clientY - cropState.y;
    };
    frame.ontouchmove = (e) => {
        if (!isDragging) return;
        e.preventDefault();
        cropState.x = e.touches[0].clientX - startX;
        cropState.y = e.touches[0].clientY - startY;
        updateTransform();
    };
    frame.ontouchend = () => isDragging = false;

    confirmBtn.onclick = () => {
        if (window.SetAvatar && currentAvatarUrl) {
            const gloat = document.getElementById("gloat-message-input").value.trim();
            localStorage.setItem("vbabes_gloat_msg", gloat);

            // Pass the favorite card ID to the server
            const state = window.GetGameState();
            window.SetAvatar(currentAvatarUrl, gloat, "", state.favorite_card_id || 0);

            // Synchronize profile metadata with the server for lobby visibility and moderation
            if (socket && socket.readyState === WebSocket.OPEN) {
                socket.send(JSON.stringify({
                    type: "register_avatar",
                    payload: { 
                        url: currentAvatarUrl,
                        gloat: gloat
                    }
                }));
            }

            showToast("Avatar verified. Entering Arena.", "success");
        }
    };
    
    // Export to window for access from index.html attributes
    window.applyAvatarFilters = applyAvatarFilters;
}

window.generateBracketHTML = (matches, activeRound = -1) => { // This will be moved to leaderboard.js
    if (!matches || matches.length === 0) {
        const msg = activeRound === -1 ? "Match data pending blockchain verification or unavailable." : "Matches will be generated once tournament starts...";
        return `<div style="color: #888; font-style: italic; padding: 10px; text-align: center; width: 100%;">${msg}</div>`;
    }

    // Group matches by round
    const rounds = {};
    matches.forEach(m => {
        if (!rounds[m.round]) rounds[m.round] = [];
        rounds[m.round].push(m);
    });

    const sortedRounds = Object.keys(rounds).sort((a, b) => a - b);
    
    let html = "";
    sortedRounds.forEach(r => {
        const isCurrentRound = (activeRound == r);

        html += `<div class="bracket-round">`;
        html += `<div class="bracket-round-title">ROUND ${r}</div>`;
        rounds[r].forEach(m => {
            const p1Short = getCachedEnvoiName(m.p1);
            const p2Short = getCachedEnvoiName(m.p2);
            
            let p1Class = "";
            let p2Class = "";
            if (m.winner) {
                if (m.winner === m.p1) {
                    p1Class = "winner"; p2Class = "loser";
                } else if (m.winner === m.p2) {
                    p2Class = "winner"; p1Class = "loser";
                }
            }
            
            html += `
                <div class="bracket-match ${isCurrentRound && !m.winner ? 'active' : ''}">
                    <div class="bracket-player ${p1Class}">${p1Short}</div>
                    <div class="vs-label">VS</div>
                    <div class="bracket-player ${p2Class}">${p2Short}</div>
                </div>
            `;
        });
        html += `</div>`;
    });
    return html;
}

window.updateTournamentPaginationUI = () => { // This will be moved to leaderboard.js
    const prevBtn = document.getElementById("prev-tournament-btn");
    const nextBtn = document.getElementById("next-tournament-btn");
    const info = document.getElementById("tournament-page-info");
    
    if (!prevBtn || !nextBtn || !info) return;

    const totalPages = Math.ceil(totalTournaments / tournamentLimit);
    info.innerText = `Page ${currentTournamentPage} of ${totalPages || 1}`;

    prevBtn.disabled = (currentTournamentPage <= 1);
    nextBtn.disabled = (currentTournamentPage >= totalPages || totalPages === 0);

    const prevIdx = currentTournamentPage - 1;
    const nextIdx = currentTournamentPage + 1;

    prevBtn.onclick = () => {
        fetchTournamentHistory(prevIdx);
        document.getElementById("hof-history-view").scrollTop = 0;
    };
    nextBtn.onclick = () => {
        fetchTournamentHistory(nextIdx);
        document.getElementById("hof-history-view").scrollTop = 0;
    };
}

function handleTournamentUI(tournamentState) { // This will be moved to leaderboard.js
    const banner = document.getElementById("tournament-banner");
    const statusText = document.getElementById("tournament-status-text");
    const regBtn = document.getElementById("tournament-reg-btn");

    if (!tournamentState || !tournamentState.active) {
        if (banner) banner.classList.add("hidden");
        return;
    }

    if (banner) banner.classList.remove("hidden");
    if (statusText) {
        const network = window.GetGameState()?.network || "VOI";
        const currency = network === "VOI" ? "$VBV" : "$AVoi";

        if (tournamentState.current_round === 0) {
            statusText.innerText = `Registration Open! Buy-in: ${tournamentState.buy_in_amount} ${currency}`;
            
            // PROACTIVE CHECK: Only show the Join button if critical network config has arrived
            const assetId = network === "VOI" ? CONFIG.VBV_ASSET_ID : CONFIG.AVOI_ASSET_ID;
            if (CONFIG.VAULT_ADDRESS && assetId) {
                if (regBtn) regBtn.classList.remove("hidden");
            } else {
                // If config is missing, inform the user why the button isn't visible yet
                statusText.innerText += " (Establishing Secure Sync...)";
                if (regBtn) regBtn.classList.add("hidden");
            }
        } else {
            statusText.innerText = `Tournament Active - Round ${tournamentState.current_round}`;
            if (regBtn) regBtn.classList.add("hidden");
        }
    }
}

window.renderTournamentBracket = async (state) => { // This will be moved to leaderboard.js
    // Prime Envoi names for all bracket participants
    const participants = new Set();
    state.matches.forEach(m => {
        if (m.p1) participants.add(m.p1);
        if (m.p2) participants.add(m.p2);
        if (m.winner) participants.add(m.winner);
    });
    await Promise.all(Array.from(participants).filter(p => p && p !== "TBD").map(p => resolveEnvoiName(p)));

    const potEl = document.getElementById("tournament-pot-display");
    if (potEl) potEl.innerText = `POT: ${state.pot.toFixed(1)} $VBV`;
    
    const visualization = document.getElementById("bracket-visualization");
    if (visualization) visualization.innerHTML = generateBracketHTML(state.matches, state.current_round);
}

window.registerForTournament = async () => { // This will be moved to leaderboard.js
    const regBtn = document.getElementById("tournament-reg-btn");
    if (!userAddress) { showToast("Connect wallet first", "error"); return; }
    const state = window.GetGameState();
    if (!state.tournament) return;

    try {
        const buyInBase = state.tournament.buy_in_amount;
        const buyInMicro = Math.floor(buyInBase * 1000000);
        const network = state.network;
        const currency = network === "VOI" ? "$VBV" : "$AVoi";
        const assetId = network === "VOI" ? CONFIG.VBV_ASSET_ID : CONFIG.AVOI_ASSET_ID;

        // HARD GUARD: Block registration if configuration hasn't been synced from the server identity message yet
        if (!assetId || !CONFIG.VAULT_ADDRESS) {
            showToast("⚠️ <b>CRITICAL SYNC ERROR:</b> Arena configuration is missing. Registration is impossible at this time. Please try refreshing.", "error", 10000);
            regBtn.disabled = false;
            regBtn.innerText = "JOIN EVENT";
            return;
        }

        const originalBtnText = regBtn.innerText;
        regBtn.disabled = true;
        regBtn.innerText = "Processing...";

        showToast(`✍️ Signing ${buyInBase} ${currency} Buy-in...`, "info");

        let txid = "";
        let txObj = null;

        // 1. CONSTRUCT TRANSACTION BASED ON NETWORK
        if (network === "VOI") {
            // ARC-200 Transfer (Application Call)
            // Selector for transfer(address,uint256): 0x2b426dec
            const methodSelector = new Uint8Array([0x2b, 0x42, 0x6d, 0xec]);
            const recipientAddr = algosdk.decodeAddress(CONFIG.VAULT_ADDRESS).publicKey;
            
            // Encode amount as 32-byte uint256 for ARC-200
            const amountArg = new Uint8Array(32);
            const amountBI = BigInt(buyInMicro);
            for (let i = 0; i < 8; i++) {
                amountArg[31 - i] = Number((amountBI >> BigInt(i * 8)) & 0xffn);
            }

            txObj = {
                from: userAddress,
                type: 'appl',
                appIndex: assetId,
                appArgs: [methodSelector, recipientAddr, amountArg],
                note: new TextEncoder().encode(`ARENA_TOURN_BUYIN:${Date.now()}`)
            };
        } else if (network === "ALGO") {
            // Standard ASA Transfer
            txObj = {
                from: userAddress,
                to: CONFIG.VAULT_ADDRESS,
                type: 'axfer',
                assetIndex: assetId,
                amount: buyInMicro,
                note: new TextEncoder().encode(`ARENA_TOURN_BUYIN:${Date.now()}`)
            };
        }

        if (!txObj) throw new Error(`Unsupported network configuration: ${network}`);

        // 2. SIGN AND BROADCAST BASED ON PROVIDER
        if (walletProvider === 'nautilus') {
            const signed = await window.algo.signTxn([{ txn: algosdk.encodeObj(txObj), signers: [userAddress] }]);
            const { txId: broadcastId } = await window.algo.sendRawTransaction(signed[0]);
            txid = broadcastId;
        } else if (walletProvider === 'kibisis') {
            const txnB64 = btoa(String.fromCharCode(...algosdk.encodeObj(txObj)));
            const signed = await window.kibisis.signTxns([{ txn: txnB64 }]);
            const { txId: broadcastId } = await window.kibisis.pushTxns(signed);
            txid = broadcastId;
        } else if (walletProvider === 'walletconnect' && signClient) {
            const sessions = signClient.session.getAll();
            if (sessions.length === 0) throw new Error("WalletConnect session not found.");
            
            const chainId = network === "VOI" ? CONFIG.VOI_CHAIN_ID : CONFIG.ALGO_CHAIN_ID;
            const txnB64 = btoa(String.fromCharCode(...algosdk.encodeObj(txObj)));
            
            const response = await signClient.request({
                topic: sessions[0].topic,
                chainId: chainId,
                request: {
                    method: "algo_signTxn",
                    params: [[{ txn: txnB64, signers: [userAddress] }]]
                }
            });

            if (!response || !response[0]) throw new Error("WalletConnect signing failed or was cancelled.");
            
            const signedTxnBytes = new Uint8Array(atob(response[0]).split("").map(c => c.charCodeAt(0)));
            const netCfg = getNetworkConfig(network);
            if (!netCfg || !netCfg.node_url) throw new Error(`Node configuration for ${network} not found. Syncing...`);
            
            const client = new algosdk.Algodv2("", netCfg.node_url, "");
            const { txId: broadcastId } = await client.sendRawTransaction(signedTxnBytes).do();
            txid = broadcastId;
        } else {
            throw new Error("Active wallet provider is not supported for tournament buy-ins.");
        }

        if (!txid) throw new Error("Transaction failed or was cancelled.");

        showToast(`🛰️ Payout Confirmed: ${txid.substring(0,8)}... Registering with server.`, "info");

        // 2. SUBMIT REGISTRATION TO BACKEND
        const response = await fetch(`${CONFIG.API_BASE}/api/tournament/register`, {
            method: "POST",
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                wallet: userAddress,
                txid: txid,
                network: network
            })
        });

        if (response.ok) {
            const result = await response.json();
            showToast(`🏆 Registration Finalized! ${result.message}`, "success", 8000);
            document.getElementById("tournament-reg-btn")?.classList.add("hidden");
        } else {
            const err = await response.text();
            if (response.status === 403) {
                if (err.includes("Opt-in Required")) {
                    showToast(`🚫 <b>PROTOCOL BLOCKED</b><br>${err}`, "error", 20000);
                } else if (err.includes("Wallet already registered")) {
                    showToast(`🚫 <b>ALREADY REGISTERED:</b> ${err}`, "warning", 10000);
                    document.getElementById("tournament-reg-btn")?.classList.add("hidden"); // Hide button if already registered
                } else {
                    showToast(`❌ Server Sync Failed (403): ${err}. Please contact support with TxID: ${txid}`, "error", 15000);
                }
                return;
            } else if (response.status === 409) { // Handle Conflict specifically
                showToast(`⚠️ <b>REGISTRATION CONFLICT:</b> ${err}`, "warning", 10000);
                return;
            }
            showToast(`❌ Server Sync Failed: ${err}. Please contact support with TxID: ${txid}`, "error", 15000);

        }
    } catch (err) {
        console.error("[TOURNAMENT ERROR]", err);
        showToast(`⚠️ Payment aborted: ${err.message}`, "error");
    } finally {
        regBtn.disabled = false;
        regBtn.innerText = originalBtnText;
    }
}

window.openTournamentBracket = () => { // This will be moved to leaderboard.js
    window.SetPhase("TournamentLobby");
    syncUI();
}

window.closeTournamentBracket = () => { // This will be moved to leaderboard.js
    window.SetPhase("Lobby");
    syncUI();
}

window.openSettingsOverlay = () => { // This will be moved to ui.js
    document.getElementById("settings-overlay").classList.remove("hidden");
}

window.closeSettingsOverlay = () => { // This will be moved to ui.js
    document.getElementById("settings-overlay").classList.add("hidden");
}

window.setMasterVolume = (value) => { // This will be moved to audio.js
    masterVolume = parseFloat(value); // Imported from audio.js
    window.SetMasterVolume(masterVolume); // This is a WASM call
    syncUI();
}

window.setMusicVolume = (value) => {
    musicVolume = parseFloat(value);
    window.SetMusicVolume(musicVolume);
    syncUI();
}

window.setSfxVolume = (value) => { // This will be moved to audio.js
    sfxVolume = parseFloat(value);
    window.SetSfxVolume(sfxVolume);
    syncUI();
}

function toggleMuteMaster() {
    masterVolume = masterVolume === 0 ? 0.5 : 0;
    document.getElementById("master-volume").value = masterVolume;
    setMasterVolume(masterVolume);
}

window.toggleMuteMusic = () => { // This will be moved to audio.js
    const state = window.GetGameState();
    let newMusicVolume = state.musicVolume === 0 ? 0.5 : 0; // Toggle between 0 and 0.5
    window.SetMusicVolume(newMusicVolume); // Update WASM engine
    document.getElementById("music-volume").value = newMusicVolume; // Update settings slider
    syncUI(); // Re-sync UI to reflect changes, including the new button
}

window.toggleMuteSfx = () => { // This will be moved to audio.js
    sfxVolume = sfxVolume === 0 ? 0.5 : 0;
    document.getElementById("sfx-volume").value = sfxVolume;
    setSfxVolume(sfxVolume);
}

// Global function to manage transaction status display // Imported from ui.js
window.setTransactionStatus = (message, type = 'info') => {
    const statusEl = document.getElementById("transaction-status");
    if (!statusEl) return;
    
    if (message) {
        statusEl.classList.remove("hidden");
        statusEl.innerHTML = `<span style="color: ${type === 'error' ? '#ff4b4b' : type === 'success' ? 'var(--neon-green)' : 'var(--neon-cyan)'};">${message}</span>`;
    } else {
        statusEl.classList.add("hidden");
        statusEl.innerHTML = "";
    }
}

window.shareTournamentVictory = () => { // This will be moved to ui.js
    const state = window.GetGameState();
    const rating = state.deck_rating || "[Z]";
    const score = `${state.scores[0]}-${state.scores[1]}`;
    const arenaUrl = window.location.origin;

    // Construct the text for the tweet
    const tweetText = `🏆 Just dominated the Virtualbabes Arena!\n\n` +
                      `⚔️ Victory: ${score}\n` +
                      `🎴 Deck Rating: ${rating}\n\n` +
                      `Come challenge me on @Voi_Network! 🚀\n\n` +
                      `#Virtualbabes #Voi #NFTGaming #Web3`;

    const twitterUrl = `<https://x.com/intent/tweet?text=${encodeURIComponent(tweetText)}&url=${encodeURIComponent(arenaUrl)}>`;
    
    // Open in a new tab
    window.open(twitterUrl, '_blank');
    
    showToast("Opening X Social Share...", "info");
}

window.showTournamentTransition = (roundNumber) => { // This will be moved to ui.js
    const overlay = document.getElementById("tournament-transition-overlay"); // Assume an overlay exists
    if (!overlay) return;
    
    overlay.querySelector(".round-number-display").innerText = `ROUND ${roundNumber}`;
    overlay.classList.remove("hidden");

    // Trigger fanfare sound effect for high-intensity round advancement
    if (window.PlaySound) {
        window.PlaySound('Pay_out-in.mp3');
    }

    setTimeout(() => overlay.classList.add("hidden"), 3000); // Hide after 3 seconds
}

// Show kidnap overlay with ransom demand
window.showKidnapOverlay = (payload) => { // This will be moved to criminality.js
    const overlay = document.getElementById("kidnap-overlay");
    const content = document.getElementById("kidnap-content");
    if (!overlay || !content) return;

    const ransomValue = payload.ransom || payload.ransom_amount || 0;
    const perpWallet = payload.perp_wallet || "Unknown";

    content.innerHTML = `
        <p>Your card <strong>${payload.card_name}</strong> has been kidnapped!</p>
        <p>Ransom: <span class="ransom-amount">${(ransomValue / 1000000).toFixed(2)} $VBV</span></p>
        <p style="opacity:0.7; font-size:0.9em;">Kidnapper: ${perpWallet}</p>
        <button class="pay-ransom-btn" onclick="payRansom(${payload.card_id}, '${perpWallet}', ${ransomValue})">Pay Ransom</button>
        <p class="insurance-timer">Insurance recovery in: <span id="recovery-timer">48:00:00</span></p>
    `;
    overlay.classList.remove("hidden");

    // Start countdown timer
    startRecoveryTimer(payload.expires_at);
}

function payRansom(cardId, perpWallet, ransomAmount) { // This will be moved to criminality.js
    if (!socket || socket.readyState !== WebSocket.OPEN) return;
    if (!perpWallet) {
        showToast("Unable to pay ransom: missing kidnapper wallet.", "error");
        return;
    }

    if (!ransomAmount || ransomAmount <= 0) {
        const amountInput = prompt("Enter the ransom amount in VBV to pay for this hostage card:", "0");
        if (!amountInput) return;
        const amountNumber = Number(amountInput);
        if (isNaN(amountNumber) || amountNumber <= 0) {
            showToast("Invalid ransom amount entered.", "error");
            return;
        }
        ransomAmount = Math.round(amountNumber * 1000000);
    }

    socket.send(JSON.stringify({
        type: "pay_ransom",
        payload: { card_id: cardId, perp_wallet: perpWallet, ransom_amount: ransomAmount }
    }));
}

window.releaseHostage = (cardId) => { // This will be moved to criminality.js
    if (!socket || socket.readyState !== WebSocket.OPEN) return;
    if (!confirm(`Release Card #${cardId} back to its victim?`)) return;

    socket.send(JSON.stringify({
        type: "release_hostage",
        payload: { card_id: cardId }
    }));
}

function startRecoveryTimer(expiresAt) { // This will be moved to criminality.js
    const timerEl = document.getElementById("recovery-timer");
    if (!timerEl) return;

    const interval = setInterval(() => {
        const now = Date.now();
        const remaining = expiresAt - now;
        if (remaining <= 0) {
            clearInterval(interval);
            timerEl.textContent = "00:00:00";
            return;
        }
        const hours = Math.floor(remaining / (1000 * 60 * 60));
        const minutes = Math.floor((remaining % (1000 * 60 * 60)) / (1000 * 60));
        const seconds = Math.floor((remaining % (1000 * 60)) / 1000);
        timerEl.textContent = `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    }, 1000);
}

async function openClubLeaseBoard() { // This will be moved to economy.js
    const state = window.GetGameState();
    const overlay = document.createElement("div");
    overlay.id = "lease-board-overlay";
    overlay.className = "overlay";

    // Detect priority region from employment
    const myClub = globalClubs[state.employer_id];
    const myRegion = myClub ? myClub.region_name : null;

    let html = `
        <div class="glass-panel" style="width: 700px; text-align: center; border-color: var(--neon-purple);">
            <h2 style="color: var(--neon-purple); letter-spacing: 2px;">INDUSTRIAL LEASE BOARD</h2>
            <p style="font-size: 0.8em; opacity: 0.7; margin-bottom: 20px;">
                Secure high-value tactical assets through the Club rental network.
                ${myRegion ? `<br><span style="color: var(--neon-cyan);">Priority Access: <b>${myRegion}</b></span>` : ''}
            </p>
            <div id="lease-list-container" class="flex-col gap-10" style="max-height: 450px; overflow-y: auto; padding-right: 10px;">
    `;

    const clubs = Object.values(globalClubs);
    // Sort: Priority Region first, then Mojo
    clubs.sort((a, b) => {
        if (a.region_name === myRegion && b.region_name !== myRegion) return -1;
        if (b.region_name === myRegion && a.region_name !== myRegion) return 1;
        return b.club_mojo - a.club_mojo;
    });

    let found = 0;
    for (const club of clubs) {
        if (!club.leases) continue;
        const available = Object.values(club.leases).filter(l => !l.borrower_wallet);
        if (available.length === 0) continue;

        html += `
            <div style="text-align: left; margin-bottom: 5px; margin-top: 15px; border-bottom: 1px solid rgba(155, 81, 224, 0.4);">
                <small style="color: var(--neon-purple); font-weight: bold; letter-spacing: 1px;">${club.name.toUpperCase()} / ${club.region_name || 'District Sector'}</small>
            </div>
        `;

        for (const lease of available) {
            found++;
            const lender = getCachedEnvoiName(lease.lender_wallet);
            html += `
                <div class="player-item" style="padding: 12px; border-color: var(--glass-border); background: rgba(0,0,0,0.25);">
                    <div style="text-align: left; flex: 1;">
                        <b style="color: var(--neon-cyan); font-size: 1.1em;">${lease.card_name}</b>
                        <div style="font-size: 0.7em; opacity: 0.6;">Lender: ${lender} | Term: ${lease.duration_hours}h</div>
                    </div>
                    <div style="text-align: right; display: flex; align-items: center; gap: 15px;">
                        <div style="color: var(--neon-green); font-weight: bold; font-family: 'Rajdhani', sans-serif;">${lease.price.toFixed(1)} $VBV</div>
                        <button class="outline" style="min-width: 100px; padding: 8px; border-color: var(--neon-purple); color: var(--neon-purple);" 
                                onclick="takeLease('${club.id}', '${lease.id}', ${lease.price})">RENT</button>
                    </div>
                </div>
            `;
        }
    }

    if (found === 0) {
        html += `<div style="padding: 60px; opacity: 0.4; font-style: italic;">No tactical assets are currently listed for lease.</div>`;
    }

    html += `
            </div>
            <button class="outline mt-20 w-full" onclick="document.getElementById('lease-board-overlay').remove()">DISCONNECT BOARD</button>
        </div>
    `;

    overlay.innerHTML = html;
    document.body.appendChild(overlay);
}

window.takeLease = async (clubId, leaseId, price) => { // This will be moved to economy.js
    if (!userAddress) return showToast("Connect wallet first", "error"); // Imported from ui.js
    if (!confirm(`Rent this card for ${price} $VBV?\n\nProceeding will commit funds from your victory balance.`)) return; // Imported from ui.js
    socket.send(JSON.stringify({ type: "take_lease", payload: { club_id: clubId, lease_id: leaseId } })); // Imported from network.js
    document.getElementById("lease-board-overlay")?.remove();
}

// --- Particle System --- // Imported from particles.js
window.initParticleSystem = () => {
    particleCanvas = document.getElementById("particle-canvas");
    if (!particleCanvas) return;

    particleCtx = particleCanvas.getContext("2d");
    
    // Resize canvas to match its parent (battle-board)
    const battleBoard = document.getElementById("board-container");
    if (battleBoard) {
        const rect = battleBoard.getBoundingClientRect();
        particleCanvas.width = rect.width;
        particleCanvas.height = rect.height;
        particleCanvas.style.left = battleBoard.offsetLeft + "px";
        particleCanvas.style.top = battleBoard.offsetTop + "px";
    }

    // Start animation loop
    if (!particleAnimationId) {
        particleAnimationId = requestAnimationFrame(animateParticles);
    }
}

window.animateParticles = () => { // Imported from particles.js
    if (!particleCtx) return;

    particleCtx.clearRect(0, 0, particleCanvas.width, particleCanvas.height);

    for (let i = particles.length - 1; i >= 0; i--) {
        const p = particles[i];

        // Update position
        p.x += p.vx;
        p.y += p.vy;
        p.vy += 0.1; // Gravity
        p.life--;

        // Fade out
        p.alpha = p.life / p.initialLife;

        // Draw particle
        particleCtx.fillStyle = `rgba(${p.color.r}, ${p.color.g}, ${p.color.b}, ${p.alpha})`;
        particleCtx.beginPath();
        particleCtx.arc(p.x, p.y, p.size * p.alpha, 0, Math.PI * 2);
        particleCtx.fill();

        if (p.life <= 0) {
            particles.splice(i, 1);
        }
    }

    if (particles.length > 0) {
        particleAnimationId = requestAnimationFrame(animateParticles);
    } else {
        particleAnimationId = null; // Stop animation if no particles
    }
}

window.openHeistPlanningOverlay = () => { // This will be moved to criminality.js

function openHeistPlanningOverlay() {
	const state = window.GetGameState();
	const overlay = document.createElement("div");
	overlay.id = "heist-overlay";
	overlay.className = "overlay";

	// Filter for external clubs only
	const clubs = Object.values(globalClubs).filter(c => c.id !== state.employer_id);
	
	overlay.innerHTML = `
		<div class="criminality-panel glass-panel animate-modal" style="width: 700px; max-height: 90vh;">
			<div class="criminality-header">
				<span class="criminality-title" style="text-shadow: 0 0 15px rgba(255, 75, 75, 0.5);">HEIST PLANNING TERMINAL</span>
				<div class="criminality-stats">
					<div class="stat-item">
						<div class="stat-label" style="color: var(--color-error-red); opacity: 0.7;">WANTED</div>
						<div class="stat-value" style="color: var(--color-error-red);">${state.wanted_level || 0}</div>
					</div>
					<div class="stat-item">
						<div class="stat-label" style="color: var(--color-neon-cyan); opacity: 0.7;">CUNNING</div>
						<div class="stat-value" style="color: var(--color-neon-cyan);">${state.cunning || 0}</div>
					</div>
				</div>
			</div>

			<div class="p-20">
				<div class="criminality-targets mb-20">
					<div class="targets-header">
						<div class="targets-title" style="font-size: 0.85em; opacity: 0.6; letter-spacing: 2px;">DETECTED SECTOR ENTITIES</div>
					</div>
					<div class="targets-list" style="grid-template-columns: repeat(2, 1fr); gap: 12px; max-height: 300px;">
						${clubs.length === 0 ? '<div class="grid-span-all opacity-3 italic py-40">No external club treasuries detected in local range.</div>' : 
							clubs.map(club => `
								<div class="target-item glass-panel m-0 p-15 hover-lift" onclick="updateHeistRiskAssessment('${club.id}')">
									<div class="target-info">
										<div class="target-name font-bold text-neon-purple" style="font-size: 1.1em;">${club.name.toUpperCase()}</div>
										<div class="target-details mt-5">
											<span class="detail-item wealth" style="color: var(--color-neon-green); font-weight: bold;">${club.treasury.toFixed(2)} $VBV</span>
											<span class="detail-item level" style="opacity: 0.6; font-size: 0.9em;">MOJO: ${club.club_mojo}</span>
										</div>
									</div>
									<div class="target-select-btn mt-10">ANALYZE DEFENSES</div>
								</div>
							`).join('')}
					</div>
				</div>

				<div id="heist-risk-section" class="criminality-risk invisible mt-10 p-20 glass-panel animate-shimmer" style="background: rgba(0,0,0,0.4); border-color: rgba(255, 166, 87, 0.3);">
					<div class="risk-header mb-15">
						<span class="risk-icon">📡</span>
						<span class="risk-title">TACTICAL PROBABILITY ANALYSIS</span>
					</div>
					
					<div class="risk-meter">
						<div class="risk-labels">
							<span class="risk-low" style="color: var(--color-neon-green);">SURGICAL</span>
							<span class="risk-high" style="color: var(--color-error-red);">CRITICAL RISK</span>
						</div>
						<div class="risk-bar" style="height: 14px; background: rgba(0,0,0,0.6); border: 1px solid rgba(255,255,255,0.1);">
							<div id="heist-risk-fill" class="risk-fill" style="width: 0%;"></div>
						</div>
					</div>
					
					<div class="flex-row justify-between align-center mt-15">
						<div id="heist-chance-text" class="progress-status" style="text-align: left; font-size: 1em;"></div>
						<div id="heist-security-details" class="font-mono" style="font-size: 0.75em; opacity: 0.5;"></div>
					</div>
					
					<div class="flex-row justify-center gap-15 mt-25">
						<button class="outline w-full secondary" onclick="document.getElementById('heist-overlay').remove()">ABORT OPS</button>
						<button id="heist-execute-btn" class="w-full danger" style="letter-spacing: 2px;">EXECUTE STRIKE</button>
					</div>
				</div>
			</div>
			
			<div class="text-center pb-20 opacity-4 font-size-0-7em letter-spacing-1">
				SECURITY ENFORCED BY THE INDUSTRIAL LOOP PROTOCOL
			</div>
		</div>
	`;

	document.body.appendChild(overlay);
}
window.updateHeistRiskAssessment = (clubId) => { // This will be moved to criminality.js
	const state = window.GetGameState(); // This will be moved to criminality.js
	const club = globalClubs[clubId];
	const section = document.getElementById("heist-risk-section");
	const fill = document.getElementById("heist-risk-fill");
	const text = document.getElementById("heist-chance-text");
	const secText = document.getElementById("heist-security-details");
	const btn = document.getElementById("heist-execute-btn");

	if (!club || !section) return;

	// Visual activation
	section.classList.remove("invisible");
	document.querySelectorAll('.target-item').forEach(item => item.classList.remove('selected'));
	event.currentTarget.classList.add('selected');
	
	// Tactical Math: Base 50% + (Effective Cunning - Security Level)
	let securityStaff = 0;
	if (club.staff) Object.values(club.staff).forEach(role => { if(role === "Security") securityStaff++; });
	
	const securityLevel = (club.club_mojo / 10) + (securityStaff * 15);

	// Registry-aligned Trap Modifiers
	const trapModifiers = {
		"tripwire": -0.10,
		"sentry_turret": -0.25,
		"guard_dog": -0.05
	};
	let trapPenalty = 0;
	if (club.active_buffs) {
		Object.values(club.active_buffs).forEach(itemId => {
			if (trapModifiers[itemId]) trapPenalty += trapModifiers[itemId];
		});
	}

	const successChance = Math.min(0.95, Math.max(0.05, 0.50 + (state.cunning - securityLevel) / 100 + trapPenalty));
	const riskPercent = (1 - successChance) * 100;

	// UI Feedback
	fill.style.width = `${riskPercent}%`;
	text.innerHTML = `ESTIMATED SUCCESS: <b style="color: var(--color-neon-green); font-size: 1.2em;">${(successChance * 100).toFixed(0)}%</b>`;
	secText.innerHTML = `TARGET SEC_LEVEL: ${securityLevel.toFixed(1)} [STAFF: ${securityStaff}]`;
	
	btn.disabled = false;
	btn.onclick = () => executeHeistStrike(clubId);
}
/**
 * Dispatches the heist request to the server.
 */
window.executeHeistStrike = (clubId) => { // This will be moved to criminality.js
	if (!socket || socket.readyState !== WebSocket.OPEN) return;
	
	showToast("🔪 Deploying field operatives...", "warning");
	socket.send(JSON.stringify({
		type: "heist",
		payload: { target_club_id: clubId }
	}));
	document.getElementById("heist-overlay")?.remove();
}

window.handleHeistResult = (payload) => { // This will be moved to criminality.js
	const title = payload.status === "success" ? "HEIST SUCCESS" : "HEIST FAILED";
	const type = payload.status === "success" ? "success" : "error";
	const msg = payload.status === "success" ? `Successfully looted the treasury! Infamy increased.` : `The alarm was triggered! You barely escaped.`;
	showToast(`<b>${title}</b><br>${msg}`, type, 8000);

	// Trigger Kidnap Gambit if eligible
	if (payload.status === "success" && payload.kidnap_eligible) {
		setTimeout(() => openKidnapSelectionOverlay(payload.target_club_id), 1500);
	}
}

/**
 * Opens the Kidnap Selection interface following a successful heist.
 * Utilizes hostage-card and criminality styles for a high-stakes feel.
 */ // Imported from criminality.js
function openKidnapSelectionOverlay(targetClubId) {
	const club = globalClubs[targetClubId];
	if (!club) return;

	const overlay = document.createElement("div");
	overlay.id = "kidnap-selection-overlay";
	overlay.className = "overlay";
	
	overlay.innerHTML = `
		<div class="criminality-panel glass-panel animate-slide-up" style="width: 450px; border-color: #ffa657;">
			<div class="criminality-header" style="border-bottom-color: rgba(255, 166, 87, 0.3);">
				<span class="criminality-title" style="color: #ffa657; text-shadow: 0 0 10px rgba(255, 166, 87, 0.5);">KIDNAP GAMBIT</span>
			</div>
			
			<div class="p-20 text-center">
				<p style="font-size: 0.9em; opacity: 0.8;">The heist was so clean you've cornered a high-value asset of <b class="text-neon-purple">${club.name}</b>.</p>
				
				<div class="hostage-card p-15 mt-20 mb-20 glass-panel">
					<div style="font-size: 0.75em; opacity: 0.5; letter-spacing: 1px;">TARGET IDENTIFIED</div>
					<b style="font-size: 1.2em; color: var(--color-error-red);">CLUB OWNER: ${club.owner_wallet.substring(0,12)}...</b>
					<div class="mt-10 italic" style="font-size: 0.8em; opacity: 0.6;">"A hostage ensures they won't retaliate... or provides a secondary payday."</div>
				</div>

				<div class="flex-col gap-10">
					<button class="outline w-full" onclick="document.getElementById('kidnap-selection-overlay').remove()">RELEASE & VANISH</button>
					<button class="w-full" style="background: var(--color-error-red); color: white;" onclick="executeKidnap('${targetClubId}')">EXECUTE KIDNAPPING</button>
				</div>
			</div>
		</div>
	`;
	document.body.appendChild(overlay);
}

window.executeKidnap = (targetClubId) => { // This will be moved to criminality.js
	if (!socket || socket.readyState !== WebSocket.OPEN) return;
	
	showToast("💀 Seizing the hostage...", "warning");
	socket.send(JSON.stringify({
		type: "kidnap_request",
		payload: { target_club_id: targetClubId }
	}));
	document.getElementById("kidnap-selection-overlay")?.remove();
}

window.triggerCaptureParticles = (gridIndex, owner) => { // Imported from particles.js
    if (!particleCtx) return;

    const boardContainer = document.getElementById("board-container");
    const slotSize = boardContainer.offsetWidth / 3; // Assuming 3x3 grid
    const col = gridIndex % 3;
    const row = Math.floor(gridIndex / 3);

    const centerX = col * slotSize + slotSize / 2;
    const centerY = row * slotSize + slotSize / 2;

    let color = { r: 0, g: 242, b: 254 }; // Neon Cyan for P1
    if (owner === 1) {
        color = { r: 255, g: 75, b: 75 }; // Error Red for P2
    }

    for (let i = 0; i < 30; i++) { // 30 particles per capture
        const angle = Math.random() * Math.PI * 2;
        const speed = Math.random() * 3 + 1;
        particles.push({
            x: centerX,
            y: centerY,
            vx: Math.cos(angle) * speed,
            vy: Math.sin(angle) * speed,
            size: Math.random() * 3 + 1,
            color: color,
            life: Math.random() * 60 + 30, // 30-90 frames life
            initialLife: Math.random() * 60 + 30,
            alpha: 1
        });
    }

    if (!particleAnimationId) {
        particleAnimationId = requestAnimationFrame(animateParticles);
    }
};
