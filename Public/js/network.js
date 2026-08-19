// Public/js/network.js

import { CONFIG } from './config.js'; // Ensure CONFIG is imported
import { showToast, setTransactionStatus, handleMaintenanceUI, handleTournamentUI, startSeasonTimer } from './ui.js';
import { updateWalletUI, disconnectUserWallet, initWalletConnect } from './wallet.js';
import { setSeasonEnd } from './leaderboard.js'; // Removed handleTournamentUI and startSeasonTimer from here
import { updatePlayerList, handleMatchmakingUpdate } from './game.js';
import { updateMarketTicker, updateBountyTicker, buyBlackMarketItem } from './economy.js';
import { updateAdminNetworkUI, setAvailableNetworks, setGlobalClubs, setAdminFocusNetwork, fetchAdminLogs, globalClubs } from './admin.js';
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
    
    // PILLAR 4: Production Resilience.
    // Utilize window.location.host as an authoritative fallback to ensure the WebSocket 
    // correctly targets the production port (8088) or the Render proxy.
    const backendHost = CONFIG.BACKEND_URL || window.location.host;
    console.log(`[WS] Connecting to ${protocol}${backendHost}/ws ...`);
    socket = new WebSocket(`${protocol}${backendHost}/ws`);

    socket.onopen = () => {
        console.log("[WS] Connected to Live Lobby");

        // Notify parent HUD
        window.parent.postMessage({ spectatorStatus: "LIVE • Connected" }, "*");

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

        // Notify parent HUD
        window.parent.postMessage({ spectatorStatus: "OFFLINE • Reconnecting…" }, "*");

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

/**
 * requestMatchSync dispatches a catch-up request to the backend.
 * PILLAR 4: Replay Resilience. 
 * Invoked by the WASM engine when sequence gaps are detected.
 */
export function requestMatchSync() {
    if (!socket || socket.readyState !== WebSocket.OPEN) return;
    
    const state = window.GetGameState("meta");
    const lastSeq = state ? (state.last_sequence_id || 0) : 0;
    
    socket.send(JSON.stringify({ type: "sync_request", payload: { last_sequence_id: lastSeq } }));
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
                CONFIG.WC_PROJECT_ID = msg.payload.wc_project_id;
                console.log("[CONFIG] Authoritative environment synced from server.");
                initWalletConnect();
            }
            // syncUI("all"); // This will be handled by app.js
            break;
        case "lobby_update":
            // Update the player list from the nested 'players' array and set it in game.js
            setLastLobbyPlayers(msg.payload.players);
            updatePlayerList(msg.payload.players);
            updateMarketTicker(msg.payload.players);
            updateBountyTicker(msg.payload.players);

            const state = window.GetGameState("combat"); // Check phase with minimal overhead
            // TACTICAL SYNC: If server altered our profile (Moderation), update local engine
            const me = msg.payload.players.find(p => p.id === myClientId);
            if (me) {
                if (window.SyncFullProfile) window.SyncFullProfile(me);
                if (me.avatar_url && window.SetAvatar) {
                    window.SetAvatar(me.avatar_url, me.gloat, me.avatar_notice);
                }
            }
            
            handleMaintenanceUI(msg.payload.maintenance_active, msg.payload.maintenance_time, msg.payload.maintenance_priority);

            if (window.SyncTournament) window.SyncTournament(msg.payload.tournament);
            handleTournamentUI(msg.payload.tournament);

            if (msg.payload.faucet_balance !== undefined) window.SyncVaultBalance(msg.payload.faucet_balance);
            if (msg.payload.total_virtual_liability !== undefined && window.SyncSolvency) {
                window.SyncSolvency(msg.payload.faucet_balance_micro || 0, msg.payload.total_virtual_liability);
            }
            if (msg.payload.reward_stack !== undefined) window.SyncRewards(msg.payload.reward_stack);
            if (window.SyncClubs) window.SyncClubs(msg.payload.clubs);
            
            if (msg.payload.available_networks) {
                setAvailableNetworks(msg.payload.available_networks);
                setGlobalClubs(msg.payload.clubs || {});
                setAdminFocusNetwork(msg.payload.admin_focus_network);
                updateAdminNetworkUI();
            }
            if (msg.payload.RewardRatio !== undefined) {
                currentRewardRatio = msg.payload.RewardRatio; // PILLAR 2: Update global for scaled payouts
            }
            updateActiveRumors(msg.payload.rumors);

            if (msg.payload.season_end) {
                setSeasonEnd(new Date(msg.payload.season_end));
                document.getElementById("season-num-display").innerText = msg.payload.season_number;
                document.getElementById("season-countdown-widget").classList.remove("hidden");
                startSeasonTimer();
            } // Assuming syncUI is still in app.js
            
            // PILLAR 5: Performance Guard. If in a match, background lobby updates should stay lean.
            if (state && state.phase === "Active") {
                requestBatchedSync("meta");
            } else {
                requestBatchedSync("all");
            }
            break;
        case "matchmaking_status":
            handleMatchmakingUpdate(msg.payload);
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
                if (window.stopChallengeWaitSFX) window.stopChallengeWaitSFX();
                if (window.playChallengeAcceptedSFX) window.playChallengeAcceptedSFX();
                setCurrentOpponentId(msg.from_id);
                setMyPlayerIndex(0);
                if (window.SetLocalPlayerIndex) window.SetLocalPlayerIndex(0);
                if (window.SyncOpponentProfile) window.SyncOpponentProfile(1, msg.payload.avatar || "", msg.payload.gloat || "", msg.payload.faceplate || "");
                if (window.SyncOpponentWanted) window.SyncOpponentWanted(1, msg.payload.wanted_level || 0);
                window.SyncOpponentDeck(1, msg.payload.deck);
                if (window.SyncMatchMetadata) window.SyncMatchMetadata(msg.payload);

                // PILLAR 4: Race Condition Hardening.
                // Transition local engine to 'Active' phase BEFORE dispatching the sync response.
                if (window.StartMatch) window.StartMatch(true);
                sendMatchSync(msg.from_id);

                if (window.triggerConnectionPulse) window.triggerConnectionPulse();
                if (window.playConnectionSFX) window.playConnectionSFX();
                if (window.playBattleStartSFX) window.playBattleStartSFX();
                requestBatchedSync("combat");
            } else if (action === "decline") {
                if (window.stopChallengeWaitSFX) window.stopChallengeWaitSFX();
                if (window.playChallengeDeclinedSFX) window.playChallengeDeclinedSFX();
                showToast(`❌ Challenge declined by ${msg.from_id}.`, "error");
            } else if (action === "sync_back") {
                // Acceptor side: Receive challenger's deck and start
                if (window.playChallengeAcceptedSFX) window.playChallengeAcceptedSFX();
                setCurrentOpponentId(msg.from_id);
                setMyPlayerIndex(1);
                if (window.SetLocalPlayerIndex) window.SetLocalPlayerIndex(1);
                if (window.SyncOpponentProfile) window.SyncOpponentProfile(0, msg.payload.avatar || "", msg.payload.gloat || "", msg.payload.faceplate || "");
                if (window.SyncOpponentWanted) window.SyncOpponentWanted(0, msg.payload.wanted_level || 0);
                window.SyncOpponentDeck(0, msg.payload.deck);
                if (window.SyncMatchMetadata) window.SyncMatchMetadata(msg.payload);
                window.StartMatch(true);
                if (window.triggerConnectionPulse) window.triggerConnectionPulse();
                if (window.playConnectionSFX) window.playConnectionSFX();
                if (window.playBattleStartSFX) window.playBattleStartSFX();
                requestBatchedSync("combat");
            }
            break;
        case "match_start":
            console.log("[WS] Synchronizing match state...", msg.payload); // This will be handled in app.js
            setSpectatorMatchState(msg.payload);
            showMatchPreview(msg.payload);
            break;
        case "sudden_death_start":
            // PILLAR 3: Sudden Death Synchronization.
            // Ingest redistributed hands and authoritative metadata (Artifact scars) into WASM.
            console.log("⚔️ SUDDEN DEATH INITIALIZED");
            if (msg.payload.text) renderChatMessage("SERVER", msg.payload.text);

            if (msg.payload.card_metadata && window.SyncCardMetadata) {
                window.SyncCardMetadata(msg.payload.card_metadata);
            }

            if (window.SyncOpponentDeck) {
                window.SyncOpponentDeck(0, msg.payload.p1_deck);
                window.SyncOpponentDeck(1, msg.payload.p2_deck);
            }

            if (window.initiateSuddenDeath) window.initiateSuddenDeath();

            showToast("⚔️ <b>SUDDEN DEATH!</b> The board has cleared.", "warning", 5000);
            requestBatchedSync("all");
            break;
        case "move":
            // Performance Optimization: high-frequency move logging suppressed in production
            // console.log(`[WS] Move received from ${msg.from_id} at grid ${msg.payload.grid_index}`);
            
            if (msg.from_id !== myClientId) {
                let success = false;

                // PILLAR 4: Authoritative Frame Sync.
                // Pass the entire AuthoritativeFrame payload to SyncMove for deterministic processing.
                // msg.payload is already the AuthoritativeFrame object.
                success = window.SyncMove(msg.payload);
                if (!success) console.warn("[WS] Move sync failed.");
                // Manual requestBatchedSync removed. 
                // window.SyncMove (Go) now triggers window.syncUI("combat") immediately upon success.
            }
            break;
        case "sync_response":
            if (msg.payload.frames && window.PushReplayFrame) {
                if (msg.payload.frames.length > 0) {
                    msg.payload.frames.forEach(frame => {
                        window.PushReplayFrame(JSON.stringify(frame));
                    });
                } else {
                    // PILLAR 4: Recovery Finalization. 
                    // If no frames are needed for catch-up, explicitly release the engine state.
                    if (window.CompleteRecovery) window.CompleteRecovery();
                }
            }
            break;
        case "turn_change":
            // PILLAR 5: Reactive Atmosphere.
            // Trigger specialized audio if the move resulted in a combo chain reaction.
            if (msg.payload && msg.payload.combo) {
                if (window.playComboSFX) window.playComboSFX();
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
            handleMaintenanceUI(msg.payload.active, msg.payload.timestamp, msg.payload.priority);
            requestBatchedSync("meta"); // Force re-render and beacon snapshot
            break;
        case "tournament_update":
            // PILLAR 3: Bracket Sync. Ensure real-time tournament and OpenTime updates are processed.
            console.log("[WS] Tournament update received:", msg.payload);
            if (window.SyncTournament) window.SyncTournament(msg.payload);
            handleTournamentUI(msg.payload);
            requestBatchedSync("meta");
            break;
        case "admin_notification":
            showToast(msg.payload.text, msg.payload.priority || "info", 8000); // Use payload's priority
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
        case "achievement_unlock":
            handleAchievementUnlock(msg.payload);
            break;
        // Justice Dashboard WebSocket events
        case "justice_card_awarded":
            if (window.onJusticeCardAwarded) window.onJusticeCardAwarded(msg.payload);
            showToast(`⚖️ <b>JUSTICE CARD AWARDED:</b><br>${msg.payload.card_type || msg.payload.type} (+${msg.payload.power_bonus || 0}% bonus)`, "success", 5000);
            break;
        case "truth_serum_applied":
            if (window.onTruthSerumApplied) window.onTruthSerumApplied(msg.payload);
            showToast(`🧪 <b>TRUTH SERUM:</b><br>Target ${msg.payload.targetWallet || 'unknown'} revealed for ${msg.payload.duration || 30}s`, "info", 4000);
            break;
        case "shield_active":
            if (window.onShieldActive) window.onShieldActive(msg.payload);
            showToast(`🛡️ <b>SHIELD:</b><br>${msg.payload.remaining}/${msg.payload.capacity} remaining`, "info", 3000);
            break;
        case "dashboard_refresh":
            if (window.onDashboardRefresh) window.onDashboardRefresh();
            break;
        case "bounty_updated":
            if (window.onBountyUpdated) window.onBountyUpdated(msg.payload);
            showToast(`🎯 <b>BOUNTY UPDATED:</b><br>${msg.payload.targetWallet || 'target'} — Wanted: ${msg.payload.wantedLevel}, Reward: ${(msg.reward || 0).toLocaleString()}`, "success", 4000);
            break;
        // PILLAR 3: Underworld Contract WS events
        case "underworld_contract_assigned":
            if (window.onContractAssigned) window.onContractAssigned(msg.payload);
            showToast(`💀 <b>CONTRACT ASSIGNED:</b><br>${msg.payload.contract_title || msg.payload.id} — Reward: ${(msg.payload.reward_micro / 1000000).toFixed(2)} $VBV`, "warning", 6000);
            break;
        case "underworld_contract_completed":
            if (window.onContractCompleted) window.onContractCompleted(msg.payload);
            showToast(`✅ <b>CONTRACT COMPLETED:</b><br>${msg.payload.contract_title || msg.payload.id} — Earned ${(msg.payload.reward_micro / 1000000).toFixed(2)} $VBV + ${msg.payload.xp_awarded || 0} CareerXP`, "success", 6000);
            break;
        // Seasonal Event WS events (P7-B Task 7103)
        case "seasonal_event_joined":
            if (window.SeasonalEvents && window.SeasonalEvents.onEventJoined) {
                window.SeasonalEvents.onEventJoined(msg.payload);
            }
            showToast(`🎯 <b>EVENT JOINED:</b><br>${msg.payload.event_title || msg.payload.event_id}`, "success", 4000);
            break;
        case "seasonal_event_created":
            if (window.SeasonalEvents && window.SeasonalEvents.onEventCreated) {
                window.SeasonalEvents.onEventCreated(msg.payload);
            }
            showToast(`🌸 <b>NEW SEASONAL EVENT:</b><br>${msg.payload.title || msg.payload.event_id}`, "info", 5000);
            break;
        case "seasonal_event_reward":
            if (window.SeasonalEvents && window.SeasonalEvents.onRewardReceived) {
                window.SeasonalEvents.onRewardReceived(msg.payload);
            }
            showToast(`🎁 <b>REWARD CLAIMED:</b><br>+${(msg.payload.reward_micro / 1000000).toFixed(2)} $VBV`, "success", 4000);
            break;
        case "seasonal_event_pool_updated":
            if (window.SeasonalEvents && window.SeasonalEvents.onPoolUpdated) {
                window.SeasonalEvents.onPoolUpdated(msg.payload);
            }
            requestBatchedSync("all"); // Refresh event cards with new budget amounts
            break;
        case "seasonal_event_expired":
            if (window.SeasonalEvents && window.SeasonalEvents.onEventExpired) {
                window.SeasonalEvents.onEventExpired(msg.payload);
            }
            showToast(`⏰ <b>EVENT ENDED:</b><br>${msg.payload.title || msg.payload.event_id}`, "info", 3000);
            requestBatchedSync("all"); // Refresh to remove expired event from grid
            break;
        case "seasonal_event_activated":
            if (window.SeasonalEvents && window.SeasonalEvents.onEventActivated) {
                window.SeasonalEvents.onEventActivated(msg.payload);
            }
            showToast(`⭐ <b>EVENT ACTIVE:</b><br>${msg.payload.title || msg.payload.event_id} — Join now!`, "success", 5000);
            requestBatchedSync("all"); // Refresh to show newly active event
            break;

    }
}

/**
 * handleAchievementUnlock processes server-pushed achievement notifications.
 * Displays a styled toast and pushes to the global achievement log.
 */
let lastAchievementId = ""; // Dedup guard
function handleAchievementUnlock(payload) {
    if (!payload || !payload.achievement_id) return;
    
    // Deduplicate rapid-fire unlocks of the same achievement
    if (payload.achievement_id === lastAchievementId) return;
    lastAchievementId = payload.achievement_id;
    setTimeout(() => { lastAchievementId = ""; }, 2000);

    const toastMessage = `🏆 <b>ACHIEVEMENT UNLOCKED:</b><br>${payload.title || "Unknown Achievement"}${payload.progress_text ? "<br>" + payload.progress_text : ""}`;
    
    showToast(toastMessage, "success", 6000);
}
