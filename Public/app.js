// PILLAR 5: Modular Orchestration.
// app.js acts as the central hub, importing authoritative domain logic
// and exposing it to the global window scope for index.html accessibility.

import { CONFIG } from './js/config.js';
import { initWebSocket, handleServerMessage, sendPing } from './js/network.js';
import { 
    hideAllOverlays, updateDynamicArenaFloor, renderCardHTML, syncBoardParticles, 
    showToast, setTransactionStatus, openSettingsOverlay, closeSettingsOverlay, 
    showTournamentTransition, shareTournamentVictory, showQuickCastMenu, openTerritoryMapOverlay,
    updateTournamentPaginationUI, openTerritoryView, updateSpectatorHUD, adjustMapZoom, 
    updateMapStatusIndicators, handleMaintenanceUI, handleTournamentUI, renderTournamentBracket, 
    openTournamentBracket, closeTournamentBracket, switchHofTab, toggleTournamentDetails
} from './js/ui.js';
import { initWalletConnect, handleWalletAction, updateWalletUI, openPayoutSettings, savePayoutAddress, userAddress, connectWith, updatePayoutUI, closeWalletSelector, addXChainWallet, submitLinkWallet } from './js/wallet.js';
import { fetchLeaderboard, registerForTournament, fetchTournamentHistory, fetchSeasonHistory, filterSeasonHistory } from './js/leaderboard.js';
import { buildEmptyBoard, toggleMatchmakingQueue, sendChatMessage, handleChatKey, proceedToWarRoom, sendChallenge, selectCard, clickGrid, executeQuickCast, acceptChallenge, declineChallenge, triggerToggleNetwork, sendSpectate, showPowerTooltip } from './js/game.js';
import { openDeckManager, closeDeckManager, renderDeckManager, setupCropEvents, applyAvatarFilters, selectAvatar, refreshInventory, renderAvatarGrid } from './js/deck.js';
import { 
    adminRefillVault, adminAddReward, adminRemoveReward, adminAddNetwork, 
    adminBroadcast, adminUpdateRules, adminBanWallet, adminUpdatePowerScaling, 
    adminToggleMaintenance, adminToggleDevMode, adminResetStats, adminSimulateTournament, adminAssetForfeiture, adminForcePayout, adminSimulateMojoDecay, adminCyberSecurityAudit,
    onAdminNetworkSelectChange, adminSetActiveNetwork, adminSeasonRollover, adminExportAuditLog
} from './js/admin.js';
import { openShopsOverlay, buyClubItem, openClubFoundry, submitClubFoundry, openArtGalleryOverlay, openConsignmentOverlay, selectConsignmentItem, submitConsignment, promptBid, openPortfolioView, tradeShares, openBlackMarket, buyBlackMarketItem, openClubLeaseBoard, switchPortfolioTab, takeLease, openCreateLeaseOverlay, submitCreateLease } from './js/economy.js';
import { openCourthouse, submitCourthouseFine, initiateBail, openSecuritySentry, deployTrap, openBountyBoard, openRumorMill, spreadRumor, openSocialPanelOverlay, switchSocialTab, openHeistPlanningOverlay, updateHeistRiskAssessment, executeHeistStrike, openKidnapSelectionOverlay, executeKidnap, releaseHostage, payRansom, showKidnapOverlay, startRecoveryTimer, sendAllianceInvite, acceptAlliance, dissolveAlliance, openTrophyView } from './js/criminality.js';
import { masterVolume, musicVolume, sfxVolume, updateMasterVolume, updateMusicVolume, updateSfxVolume, toggleMuteMusic, initAudioContext, toggleMuteMaster, toggleMuteSfx } from './js/audio.js';
import { initParticleSystem, triggerCaptureParticles, triggerGlobalKidnapEffect } from './js/particles.js';
import { getAssetSymbol, getCachedEnvoiName, resolveEnvoiName, reportGloat, shortenAddress, resolveAssetSymbol, getNetworkConfig } from './js/utils.js';

// --- Global Bridge: index.html event mapping ---
window.handleWalletAction = handleWalletAction;
window.connectWith = connectWith;
window.closeWalletSelector = closeWalletSelector;
window.openPayoutSettings = openPayoutSettings;
window.reportPlayer = reportPlayer;
window.savePayoutAddress = savePayoutAddress;
window.reportGloat = reportGloat;
window.toggleMuteMusic = toggleMuteMusic;
window.toggleMuteMaster = toggleMuteMaster;
window.toggleMuteSfx = toggleMuteSfx;
window.setMasterVolume = updateMasterVolume;
window.setMusicVolume = updateMusicVolume;
window.setSfxVolume = updateSfxVolume;
window.toggleMatchmakingQueue = toggleMatchmakingQueue;
window.sendChatMessage = sendChatMessage;
window.handleChatKey = handleChatKey;
window.registerForTournament = registerForTournament;
window.filterSeasonHistory = filterSeasonHistory;
window.openTournamentBracket = openTournamentBracket;
window.closeTournamentBracket = closeTournamentBracket;
window.switchHofTab = switchHofTab;
window.fetchLeaderboard = fetchLeaderboard;
window.fetchTournamentHistory = fetchTournamentHistory;
window.fetchSeasonHistory = fetchSeasonHistory;
window.toggleTournamentDetails = toggleTournamentDetails;
window.proceedToWarRoom = proceedToWarRoom;
window.acceptChallenge = acceptChallenge;
window.declineChallenge = declineChallenge;
window.sendChallenge = sendChallenge;
window.sendSpectate = sendSpectate;
window.triggerToggleNetwork = triggerToggleNetwork;
window.openDeckManager = openDeckManager;
window.closeDeckManager = closeDeckManager;
window.setupCropEvents = setupCropEvents;
window.applyAvatarFilters = applyAvatarFilters;
window.selectAvatar = selectAvatar;
window.refreshInventory = refreshInventory;
window.openShopsOverlay = openShopsOverlay;
window.buyClubItem = buyClubItem;
window.openClubFoundry = openClubFoundry;
window.submitClubFoundry = submitClubFoundry;
window.openTerritoryMapOverlay = openTerritoryMapOverlay;
window.adjustMapZoom = adjustMapZoom;
window.openSocialPanelOverlay = openSocialPanelOverlay;
window.switchSocialTab = switchSocialTab;
window.openPortfolioView = openPortfolioView;
window.switchPortfolioTab = switchPortfolioTab;
window.tradeShares = tradeShares;
window.openBlackMarket = openBlackMarket;
window.buyBlackMarketItem = buyBlackMarketItem;
window.openArtGalleryOverlay = openArtGalleryOverlay;
window.openConsignmentOverlay = openConsignmentOverlay;
window.selectConsignmentItem = selectConsignmentItem;
window.submitConsignment = submitConsignment;
window.promptBid = promptBid;
window.submitBid = submitBid;
window.addXChainWallet = addXChainWallet;
window.submitLinkWallet = submitLinkWallet;
window.openClubLeaseBoard = openClubLeaseBoard;
window.openCreateLeaseOverlay = openCreateLeaseOverlay;
window.submitCreateLease = submitCreateLease;
window.takeLease = takeLease;
window.openCourthouse = openCourthouse;
window.submitCourthouseFine = submitCourthouseFine;
window.initiateBail = initiateBail;
window.openSecuritySentry = openSecuritySentry;
window.deployTrap = deployTrap;
window.openBountyBoard = openBountyBoard;
window.openRumorMill = openRumorMill;
window.spreadRumor = spreadRumor;
window.openHeistPlanningOverlay = openHeistPlanningOverlay;
window.updateHeistRiskAssessment = updateHeistRiskAssessment;
window.executeHeistStrike = executeHeistStrike;
window.openKidnapSelectionOverlay = openKidnapSelectionOverlay;
window.executeKidnap = executeKidnap;
window.payRansom = payRansom;
window.releaseHostage = releaseHostage;
window.sendAllianceInvite = sendAllianceInvite;
window.acceptAlliance = acceptAlliance;
window.dissolveAlliance = dissolveAlliance;
window.openTrophyView = openTrophyView;
window.triggerGlobalKidnapEffect = triggerGlobalKidnapEffect;
window.shareTournamentVictory = shareTournamentVictory;
window.adminSeasonRollover = adminSeasonRollover;
window.adminExportAuditLog = adminExportAuditLog;
window.adminSimulateTournament = adminSimulateTournament;
window.adminSimulateMojoDecay = adminSimulateMojoDecay;
window.adminCyberSecurityAudit = adminCyberSecurityAudit;
window.adminForcePayout = adminForcePayout;
window.displayCyberAuditReportInChat = displayCyberAuditReportInChat; // New: Expose for network.js to call
window.adminAssetForfeiture = adminAssetForfeiture;
window.adminToggleDevMode = adminToggleDevMode;
window.adminToggleMaintenance = adminToggleMaintenance;
window.adminUpdateRules = adminUpdateRules;
window.adminBroadcast = adminBroadcast;
window.adminAddReward = adminAddReward;
window.adminRemoveReward = adminRemoveReward;
window.adminRefillVault = adminRefillVault;
window.adminSetActiveNetwork = adminSetActiveNetwork;
window.onAdminNetworkSelectChange = onAdminNetworkSelectChange;
window.selectCard = selectCard;
window.clickGrid = clickGrid;
window.executeQuickCast = executeQuickCast;

// --- Bootstrapping Lifecycle ---
window.onload = async () => {
    console.log("[ARENA] Initiating Neural Uplink...");
    
    const go = new Go();
    try {
        const result = await WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject);
        go.run(result.instance);
        console.log("[ARENA] WASM Engine ACTIVE.");

        // PILLAR 6: Client Beacon Recovery (Warm Start).
        // Immediately prime the engine with the last known "Push" state from the server.
        const cachedBeacon = localStorage.getItem("vbabes_state_beacon");
        if (cachedBeacon) {
            try {
                const beacon = JSON.parse(cachedBeacon);
                if (window.SyncFullProfile) window.SyncFullProfile(beacon.profile);
                if (window.SyncVaultBalance) window.SyncVaultBalance(beacon.vault_balance);
                if (window.SyncRewards) window.SyncRewards(beacon.rewards);
                if (window.SyncClubs) window.SyncClubs(beacon.clubs);
            } catch (e) { console.warn("[BOOT] Beacon corrupt or unavailable."); }
        }

        // 1. Initial configuration sync
        if (window.SetApiBase) window.SetApiBase(CONFIG.API_BASE);
        if (window.SetAssetBase) window.SetAssetBase(CONFIG.ASSET_URL);

        // 2. Establish Network Switchboard
        initWebSocket(handleServerMessage);

        // 3. Initialize Visuals
        initParticleSystem();
        buildEmptyBoard();

        // 4. Initial Heartbeat
        setInterval(sendPing, 30000);
        
        window.syncUI();
    } catch (err) {
        console.error("[BOOT ERROR] Engine initialization failed:", err);
        showToast("❌ Critical Error: Neural Uplink Failed. Please refresh.", "error", 0);
    }
};

// --- UI Performance Layer ---
const UI_CACHE = new Map();
const getEl = (id) => {
    if (!UI_CACHE.has(id)) UI_CACHE.set(id, document.getElementById(id));
    return UI_CACHE.get(id);
};

let dashboardCache = { stateKey: "", lastBalance: -1 };

/**
 * triggerFoundryFusion - Special FX for Club actions.
 */
window.triggerFoundryFusion = (type) => {
    import('./js/particles.js').then(m => m.triggerFoundryFusion(type));
};

/**
 * window.syncUI - The heart of the client orchestrator.
 * Reads authoritative state from the WASM engine and updates the DOM.
 */
window.syncUI = (scope = "all") => {
    const state = window.GetGameState();
    if (!state) return;

    // PERFORMANCE GUARD: Detect state changes to prevent redundant re-renders
    // PILLAR 5: Precise Synchronization. Include tournament and player count in state key.
    const scannerActive = state.district_scanner_expires_at && (new Date(state.district_scanner_expires_at) > Date.now());
    const cyberJammerActive = state.has_cyber_jammer;
    const currentStateKey = `${state.phase}-${state.turn}-${state.wanted_level}-${state.reputation}-${state.tournament?.matches?.length || 0}-${state.tournament?.participants?.length || 0}-${scannerActive}-${cyberJammerActive}`;
    if (scope !== "combat" && currentStateKey === dashboardCache.stateKey && state.vault_balance === dashboardCache.lastBalance) {
        // Only proceed if scope is combat or state has evolved
        if (scope !== "all") return;
    }
    dashboardCache.stateKey = currentStateKey;
    dashboardCache.lastBalance = state.vault_balance;

    // PILLAR 2: Usable Total.
    // Combine physical blockchain balance ($VBV) with virtual salary/heist rewards.
    const physicalVBV = state.vault_balance || 0;
    const virtualVBV = state.virtual_balance || 0; // virtual_balance is already in base units from WASM
    const totalLiquid = physicalVBV + virtualVBV;

    const liquidEl = getEl("total-liquid-balance");
    if (liquidEl) liquidEl.innerText = totalLiquid.toFixed(2);

    updateVolumeSlidersUI(); // Ensure volume sliders reflect persisted values

    // Atmospheric Underworld Shift: Trigger red tint on high-infamy
    document.body.classList.toggle("criminal-underworld", state.wanted_level >= 10);

    // --- Domain Orchestration ---
    if (scope === "combat" || scope === "all") {
        updateDynamicArenaFloor(state);
        syncBoardParticles(state);
        updateSpectatorHUD(state);
        
        // Dynamic taunts from NPCs based on observed playstyle
        if (state.multiplayer === false && state.phase === "Active") {
            // Heuristic to ensure taunts don't spam every frame
            const turnKey = `${state.turn}`;
            if (window.lastTauntTurn !== turnKey) {
                window.lastTauntTurn = turnKey;
                import('./collective-intelligence.js').then(m => {
                    const taunt = m.collectiveIntelligence.generatePlaystyleTaunt(state.p2_name, state.playstyle);
                    if (taunt) renderChatMessage("SYSTEM", taunt);
                });
            }
        }
    }

    if (scope === "meta" || scope === "all") {
        handleTournamentUI(state.tournament);
        if (state.tournament && state.tournament.active) {
            renderTournamentBracket(state.tournament);
        }
        updateMapStatusIndicators();
    }

    // --- Phase Transitions ---
    const lobby = getEl("lobby-container");
    const combat = getEl("combat-container");
    const tourney = getEl("tournament-lobby-container");

    if (lobby && combat && tourney) {
        lobby.classList.add("hidden");
        combat.classList.add("hidden");
        tourney.classList.add("hidden");

        switch(state.phase) {
            case "Active":
                combat.classList.remove("hidden");
                break;
            case "TournamentLobby":
                tourney.classList.remove("hidden");
                break;
            default:
                lobby.classList.remove("hidden");
        }
    }

    // Initial wallet detection
    if (!userAddress) {
        const overlay = getEl("wallet-selector-overlay");
        if (overlay) overlay.classList.remove("hidden");
    }

    // PILLAR 3: Cyber-Jammer UI State.
    const cyberJammerEl = getEl("cyber-jammer-status");
    if (cyberJammerEl) {
        if (state.has_cyber_jammer) {
            cyberJammerEl.classList.remove("hidden");
        } else {
            cyberJammerEl.classList.add("hidden");
        }
    }

    // PILLAR 3: Heist Saboteur Progress.
    const jammerCount = state.heist_alarms_jammer_count || 0;
    const heistProgressEl = getEl("heist-saboteur-progress");
    const isHeistSaboteur = state.achievements && state.achievements.includes("HEIST_SABOTEUR");

    if (heistProgressEl) {
        if (jammerCount > 0 && !isHeistSaboteur) {
            heistProgressEl.classList.remove("hidden");
            const percent = Math.min(100, (jammerCount / 3) * 100);
            heistProgressEl.innerHTML = `
                <div class="font-size-0-7em text-warning mb-2 uppercase letter-spacing-1">SABOTEUR: ${jammerCount}/3</div>
                <div class="progress-bar" style="width: 80px; height: 3px;">
                    <div class="progress-fill" style="width: ${percent}%"></div>
                </div>`;
        } else {
            heistProgressEl.classList.add("hidden");
        }
    }

    // PILLAR 6: Push-Authority Beacon.
    // If the UI is in a stable lobby state, cache the authoritative push to localStorage.
    if (scope === "all" && state.phase === "Lobby" && userAddress) {
        const beaconData = {
            profile: state,
            vault_balance: state.vault_balance,
            rewards: state.rewards,
            clubs: state.clubs,
            ts: Date.now()
        };
        localStorage.setItem("vbabes_state_beacon", JSON.stringify(beaconData));
    }
};

/**
 * displayCyberAuditReportInChat processes and displays a Cyber-Audit report in the chat.
 * This function is intended to be called by network.js when a Cyber-Audit admin_notification is received.
 */
export function displayCyberAuditReportInChat(reportText) {
    // PILLAR 3: Intelligence Display. Directly render the detailed report in the chat.
    renderChatMessage("SYSTEM", reportText);
}

/**
 * reportPlayer triggers the automated reporting protocol for a specific target.
 * PILLAR 3: Criminality & Intelligence.
 */
export async function reportPlayer(targetWallet) {
    const reason = prompt("Enter reason for report (e.g. Harassment, Exploiting, Cheating):");
    if (!reason) return;
    const details = prompt("Enter additional details (Match ID, specific behavior):") || "";

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/report-player`, {
            method: "POST",
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                reporter_wallet: userAddress,
                target_wallet: targetWallet,
                reason: reason,
                details: details
            })
        });

        if (response.ok) {
            showToast("🚩 Report submitted to Arena Security.", "success");
        } else {
            const err = await response.text();
            showToast(`❌ Failed to submit report: ${err}`, "error");
        }
    } catch (err) {
        console.error("[MODERATION ERROR]", err);
        showToast("❌ Connection error during report submission.", "error");
    }
}
