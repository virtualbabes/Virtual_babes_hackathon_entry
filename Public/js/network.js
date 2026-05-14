// Public/js/network.js

import { CONFIG } from './config.js';
import { showToast, setTransactionStatus } from './ui.js';
import { updateWalletUI, disconnectUserWallet } from './wallet.js';
import { handleTournamentUI, setSeasonEnd, startSeasonTimer } from './leaderboard.js';
import { updatePlayerList } from './game.js';
import { updateMarketTicker, buyBlackMarketItem } from './economy.js';
import { handleMaintenanceUI } from './ui.js';
import { updateAdminNetworkUI, setAvailableNetworks, setGlobalClubs, setAdminFocusNetwork, fetchAdminLogs } from './admin.js';
import { updateActiveRumors, handleHeistResult, showKidnapOverlay, startRecoveryTimer } from './criminality.js';

export let socket = null;
import { setLastLobbyPlayers, setMyPlayerIndex, setCurrentOpponentId, setSpectatorMatchState, renderChatMessage, saveMatchResult, setMatchHistorySaved } from './game.js';
export let myClientId = null;
export let reconnectAttempts = 0;
export let nonceResolver = null;
export let identitySyncTimeout = null;
export let lastPingTime = null;
export let currentLatency = null;

export const setMyClientId = (id) => { myClientId = id; };
export const setNonceResolver = (resolver) => { nonceResolver = resolver; };
export const setReconnectAttempts = (attempts) => { reconnectAttempts = attempts; };
export const setIdentitySyncTimeout = (timeout) => { identitySyncTimeout = timeout; };
export const setLastPingTime = (time) => { lastPingTime = time; };
export const setCurrentLatency = (latency) => { currentLatency = latency; };

export const getNonceResolver = () => nonceResolver;
export const getReconnectAttempts = () => reconnectAttempts;
export const getIdentitySyncTimeout = () => identitySyncTimeout;
export const getLastPingTime = () => lastPingTime;
export const getCurrentLatency = () => currentLatency;

export function initWebSocket(messageHandler) {
    const protocol = window.location.protocol === "https:" ? "wss://" : "ws://";
    console.log(`[WS] Connecting to ${protocol}${CONFIG.BACKEND_URL}/ws ...`);
    socket = new WebSocket(`${protocol}${CONFIG.BACKEND_URL}/ws`);

    socket.onopen = () => {
        console.log("[WS] Connected to Live Lobby");

        // WATCHDOG: Start 5s timer for identity sync validation.
        // If identity is not received, attempt reconnection.
        if (identitySyncTimeout) clearTimeout(identitySyncTimeout);
        identitySyncTimeout = setTimeout(() => {
            if (!myClientId) {
                if (reconnectAttempts < 3) {
                    reconnectAttempts++;
                    console.warn(`[WS] Identity sync timeout reached. Attempting reconnect ${reconnectAttempts}/3...`);
                    showToast(`⚠️ Sync failed. Retrying connection (${reconnectAttempts}/3)...`, "warning", 3000);
                    socket.close(); // Force close to trigger onclose and re-init
                } else {
                    console.error("[WS] Identity sync timeout reached after multiple attempts.");
                    showToast("⚠️ <b>SYNC FAILURE:</b> Arena configuration not received after multiple attempts. Faucet payouts and tournament registrations may be unavailable. Please refresh.", "error", 0);
                }
            }
        }, 5000);
    };

    socket.onmessage = (event) => {
        const msg = JSON.parse(event.data);
        messageHandler(msg);
    };

    socket.onclose = () => {
        console.warn("[WS] Disconnected. Retrying...");
        if (identitySyncTimeout) clearTimeout(identitySyncTimeout);
        // Only attempt immediate reconnect if not due to identity sync timeout already handling it
        if (identitySyncTimeout && reconnectAttempts < 3) {
            setTimeout(() => initWebSocket(messageHandler), 3000);
        }
    };
}

export function sendPing() {
    if (!socket || socket.readyState !== WebSocket.OPEN) return;
    lastPingTime = Date.now();
    socket.send(JSON.stringify({ type: "ping" }));
}

let syncScheduled = false;
let currentSyncScope = null;

/**
 * PERFORMANCE OPTIMIZATION: Batches UI synchronization requests using requestAnimationFrame.
 * This prevents layout thrashing and redundant DOM diffing when multiple WebSocket messages 
 * arrive within the same frame (common during rapid AI moves or Combo chain reactions).
 */
function requestBatchedSync(scope = "all") {
    // "all" scope covers any other granular scope
    if (currentSyncScope === "all") return;
    currentSyncScope = scope;

    if (syncScheduled) return;
    syncScheduled = true;

    requestAnimationFrame(() => {
        const targetScope = currentSyncScope;
        syncScheduled = false;
        currentSyncScope = null;
        if (window.syncUI) window.syncUI(targetScope);
    });
}

export function handleServerMessage(msg) {
    switch(msg.type) {
        case "pong":
            if (lastPingTime) {
                currentLatency = Date.now() - lastPingTime;
                lastPingTime = null;
                if (window.SyncLatency) window.SyncLatency(currentLatency);
                // syncUI("meta"); // This will be handled by app.js
            }
            break;
        case "identity":
            myClientId = msg.to_id;
            if (identitySyncTimeout) {
                clearTimeout(identitySyncTimeout);
                identitySyncTimeout = null;
                reconnectAttempts = 0;
            }
            if (msg.payload) {
                CONFIG.VAULT_ADDRESS = msg.payload.vault;
                CONFIG.VBV_ASSET_ID = msg.payload.vbv;
                CONFIG.AVOI_ASSET_ID = msg.payload.avoi;
                console.log("[CONFIG] Authoritative environment synced from server.");
            }
            // syncUI("all"); // This will be handled by app.js
            break;
        case "lobby_update":
            // Update the player list from the nested 'players' array and set it in game.js
            setLastLobbyPlayers(msg.payload.players);
            updatePlayerList(msg.payload.players);
            updateMarketTicker(msg.payload.players);

            // TACTICAL SYNC: If server altered our profile (Moderation), update local engine
            const me = msg.payload.players.find(p => p.id === myClientId);
            if (me) {
                if (window.SyncFullProfile) window.SyncFullProfile(me);
                if (me.avatar_url && window.SetAvatar) {
                    window.SetAvatar(me.avatar_url, me.gloat, me.avatar_notice);
                }
            }
            
            handleMaintenanceUI(msg.payload.maintenance_active, msg.payload.maintenance_time);

            if (window.SyncTournament) window.SyncTournament(msg.payload.tournament);
            handleTournamentUI(msg.payload.tournament);

            if (msg.payload.faucet_balance !== undefined) window.SyncVaultBalance(msg.payload.faucet_balance);
            if (msg.payload.rewards !== undefined) window.SyncRewards(msg.payload.rewards);
            if (window.SyncClubs) window.SyncClubs(msg.payload.clubs);
            
            if (msg.payload.available_networks) {
                setAvailableNetworks(msg.payload.available_networks);
                setGlobalClubs(msg.payload.clubs || {});
                setAdminFocusNetwork(msg.payload.admin_focus_network);
                updateAdminNetworkUI();
            }
            updateActiveRumors(msg.payload.rumors);

            if (msg.payload.season_end) {
                setSeasonEnd(new Date(msg.payload.season_end));
                document.getElementById("season-num-display").innerText = msg.payload.season_number;
                document.getElementById("season-countdown-widget").classList.remove("hidden");
                startSeasonTimer();
            } // Assuming syncUI is still in app.js
            requestBatchedSync("all");
            break;
        case "portfolio_update":
            if (window.SyncPortfolio) window.SyncPortfolio(msg.payload); // SyncPortfolio is a WASM call
            break;
        case "heist_result": // Now handled by criminality.js
            handleHeistResult(msg.payload);
            break;
        case "challenge":
            const action = msg.payload.action;
            if (action === "invite") {
                showChallengeNotification(msg.from_id);
            } else if (action === "accept") {
                // Challenger side: Receive acceptor's deck and send own deck back
                console.log("[MATCH] Challenge accepted. Syncing decks..."); 
                setCurrentOpponentId(msg.from_id);
                setMyPlayerIndex(0);
                if (window.SetLocalPlayerIndex) window.SetLocalPlayerIndex(0);
                if (window.SyncOpponentProfile) window.SyncOpponentProfile(1, msg.payload.avatar || "", msg.payload.gloat || "");
                if (window.SyncOpponentWanted) window.SyncOpponentWanted(1, msg.payload.wanted_level || 0);
                window.SyncOpponentDeck(1, msg.payload.deck);
                sendMatchSync(msg.from_id);
                window.StartMatch(true);
                if (window.triggerConnectionPulse) window.triggerConnectionPulse();
                if (window.playConnectionSFX) window.playConnectionSFX();
                requestBatchedSync("combat");
            } else if (action === "decline") {
                alert(`Challenge declined by ${msg.from_id}.`);
            } else if (action === "sync_back") {
                // Acceptor side: Receive challenger's deck and start
                setCurrentOpponentId(msg.from_id);
                setMyPlayerIndex(1);
                if (window.SetLocalPlayerIndex) window.SetLocalPlayerIndex(1);
                if (window.SyncOpponentProfile) window.SyncOpponentProfile(0, msg.payload.avatar || "", msg.payload.gloat || "");
                if (window.SyncOpponentWanted) window.SyncOpponentWanted(0, msg.payload.wanted_level || 0);
                window.SyncOpponentDeck(0, msg.payload.deck);
                window.StartMatch(true);
                if (window.triggerConnectionPulse) window.triggerConnectionPulse();
                if (window.playConnectionSFX) window.playConnectionSFX();
                requestBatchedSync("combat");
            }
            break;
        case "match_start":
            console.log("[WS] Synchronizing match state...", msg.payload); // This will be handled in app.js
            setSpectatorMatchState(msg.payload);
            showMatchPreview(msg.payload);
            break;
        case "move":
            // Performance Optimization: high-frequency move logging suppressed in production
            // console.log(`[WS] Move received from ${msg.from_id} at grid ${msg.payload.grid_index}`);
            
            if (msg.from_id !== myClientId) {
                let success = false;
                if (spectatorMatchState) { // Use window.spectatorMatchState from game.js
                    // We are a spectator: Determine player index from match state
                    const pIdx = (msg.from_id === spectatorMatchState.p1_id) ? 0 : 1;
                    success = window.SyncMove(msg.payload.grid_index, msg.payload.card_id, pIdx);
                } else {
                    // We are a player: Standard turn-based placement
                    success = window.PlaceCard(msg.payload.grid_index, msg.payload.card_id);
                }
                if (!success) console.warn("[WS] Move sync failed.");
                requestBatchedSync("combat");
            }
            break;
        case "chat":
            renderChatMessage(msg.from_id, msg.payload.text);
            if (msg.from_id === "SERVER" && msg.payload.text.includes("Match invalidated")) {
                window.ResetGame();
                requestBatchedSync("combat");
                showToast("⚠️ Match terminated: Opponent left.", "error");
            }
            break;
        case "vault_update":
            console.log("[WS] Vault balance update received:", msg.payload.balance);
            window.SyncVaultBalance(msg.payload.balance);
            break;
        case "rules_update":
            console.log("[WS] Global rules update received:", msg.payload);
            window.SyncRules(msg.payload);
            showToast("⚙️ Global Game Rules Updated by Admin", "info"); // This is a UI notification
            break;
        case "rewards_update":
            console.log("[WS] Reward stack update received:", msg.payload);
            window.SyncRewards(msg.payload);
            break;
        case "maintenance_update":
            console.log("[WS] Maintenance update received:", msg.payload);
            handleMaintenanceUI(msg.payload.active, msg.payload.timestamp);
            break;
        case "admin_notification":
            showToast(msg.payload.text, "warning", 8000);
            const adminPanel = document.getElementById("admin-control-panel");
            if (adminPanel && !adminPanel.classList.contains("hidden")) {
                fetchAdminLogs();
            }
            break;
        case "kidnap_success":
            showToast("Kidnap successful! Card held hostage.", "success", 5000); // This is a UI notification, can stay in network or move to criminality
            break;
        case "ransom_demand": // Now handled by criminality.js
            showKidnapOverlay(msg.payload);
            break;
        case "ransom_paid": // Now handled by criminality.js
            showToast("Ransom paid. Card released.", "success", 5000); 
            hideAllOverlays();
            break;
        case "insurance_recovery": // This is a UI notification, can stay in network or move to criminality
            showToast("Insurance recovery: Hostage card released.", "info", 5000);
            break;
        case "rumor_update":
            if (msg.payload && msg.payload.rumor) {
                updateActiveRumors(msg.payload.rumor);
            }
            break;
    }
}