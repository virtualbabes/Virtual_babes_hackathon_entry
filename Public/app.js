// PILLAR 5: Modular Orchestration.
// app.js acts as the central hub, importing authoritative domain logic
// and exposing it to the global window scope for index.html accessibility.

import { CONFIG } from './js/config.js';
import { initWebSocket, handleServerMessage, sendPing, requestMatchSync } from './js/network.js';
import { 
    hideAllOverlays, updateDynamicArenaFloor, renderCardHTML, syncBoardParticles, 
    showToast, setTransactionStatus, openSettingsOverlay, closeSettingsOverlay, 
    showTournamentTransition, shareTournamentVictory, showQuickCastMenu, openTerritoryMapOverlay,
    updateTournamentPaginationUI, openTerritoryView, updateSpectatorHUD, adjustMapZoom, 
    updateMapStatusIndicators, handleMaintenanceUI, handleTournamentUI, renderTournamentBracket, 
    openTournamentBracket, closeTournamentBracket, switchHofTab, toggleTournamentDetails, showMutationStabilityTooltip, hidePowerTooltip,
    updateAvatarIdentityStyle, updateMoodCatalystVisuals, updateStaffTrainingVisuals
} from './js/ui.js';
import { initWalletConnect, handleWalletAction, updateWalletUI, openPayoutSettings, savePayoutAddress, userAddress, connectWith, updatePayoutUI, closeWalletSelector, addXChainWallet, submitLinkWallet } from './js/wallet.js';
import { fetchLeaderboard, registerForTournament, fetchTournamentHistory, fetchSeasonHistory, filterSeasonHistory } from './js/leaderboard.js';
import { buildEmptyBoard, toggleMatchmakingQueue, sendChatMessage, handleChatKey, proceedToWarRoom, sendChallenge, selectCard, clickGrid, executeQuickCast, acceptChallenge, declineChallenge, triggerToggleNetwork, sendSpectate, showPowerTooltip, rejoinActiveMatch, setPendingQuickCastId } from './js/game.js';
import { openDeckManager, closeDeckManager, renderDeckManager, setupCropEvents, applyAvatarFilters, selectAvatar, refreshInventory, renderAvatarGrid } from './js/deck.js';
import { 
    globalClubs,
    adminRefillVault, adminAddReward, adminRemoveReward, adminAddNetwork, 
    adminBroadcast, adminUpdateRules, adminBanWallet, adminUpdatePowerScaling, adminSimulateMutationSuccess, adminSimulateMutationFailure,
    adminToggleMaintenance, adminToggleDevMode, adminResetStats, adminSimulateTournament, adminAssetForfeiture, adminForcePayout, adminSimulateMojoDecay, adminCyberSecurityAudit,
    onAdminNetworkSelectChange, adminSetActiveNetwork, adminSeasonRollover, adminExportAuditLog, adminCommissionAudit, adminTaxAudit
} from './js/admin.js'; // Note: adminDistrictTaxAudit should be added here
import { openShopsOverlay, buyClubItem, openClubFoundry, submitClubFoundry, openArtGalleryOverlay, openConsignmentOverlay, selectConsignmentItem, submitConsignment, promptBid, openPortfolioView, tradeShares, openBlackMarket, buyBlackMarketItem, openClubLeaseBoard, switchPortfolioTab, takeLease, openCreateLeaseOverlay, submitCreateLease, openMutationHistoryOverlay, openVaultInteraction, submitDistrictTax } from './js/economy.js';
import { openCourthouse, submitCourthouseFine, initiateBail, openSecuritySentry, deployTrap, openBountyBoard, openRumorMill, spreadRumor, openSocialPanelOverlay, switchSocialTab, openHeistPlanningOverlay, updateHeistRiskAssessment, executeHeistStrike, openKidnapSelectionOverlay, executeKidnap, releaseHostage, payRansom, showKidnapOverlay, startRecoveryTimer, sendAllianceInvite, acceptAlliance, dissolveAlliance, openTrophyView, reportPlayer } from './js/criminality.js';
import { 
    playProcedureInterruptedSFX, playLongWarningSFX, playCloakFailureSFX, playCloakDisruptorSFX, playMutationSoundscape, stopMutationSoundscape, playMutationSuccessSFX, playEcosystemAlertSFX
} from './js/audio.js';
import { 
    initParticleSystem, triggerCaptureParticles, triggerGlobalKidnapEffect, 
    triggerMutationScarEffect, triggerCloakFailureParticles, triggerCloakDisruptorParticles
} from './js/particles.js';
import { getAssetSymbol, getCachedEnvoiName, resolveEnvoiName, reportGloat, shortenAddress, resolveAssetSymbol, getNetworkConfig } from './js/utils.js'; // Removed playMutationSoundscape, stopMutationSoundscape

// --- Global Bridge: index.html event mapping ---
window.handleWalletAction = handleWalletAction;
window.connectWith = connectWith;
window.closeWalletSelector = closeWalletSelector;
window.openPayoutSettings = openPayoutSettings;
window.reportPlayer = reportPlayer;
window.savePayoutAddress = savePayoutAddress;
window.reportGloat = reportGloat;
window.requestMatchSync = requestMatchSync;
window.playEcosystemAlertSFX = playEcosystemAlertSFX;
window.toggleMuteMusic = toggleMuteMusic;
window.toggleMuteMaster = toggleMuteMaster;
window.toggleMuteSfx = toggleMuteSfx;
window.setMasterVolume = updateMasterVolume;
window.playMutationSuccessSFX = playMutationSuccessSFX;
window.playCloakDisruptorSFX = playCloakDisruptorSFX;
window.playProcedureInterruptedSFX = playProcedureInterruptedSFX;
window.playCloakFailureSFX = playCloakFailureSFX;
window.playLongWarningSFX = playLongWarningSFX;
window.setMusicVolume = updateMusicVolume;
window.setSfxVolume = updateSfxVolume;
window.toggleMatchmakingQueue = toggleMatchmakingQueue;
window.sendChatMessage = sendChatMessage;
window.playMutationSoundscape = playMutationSoundscape; // Expose new audio function
window.stopMutationSoundscape = stopMutationSoundscape; // Expose new audio function
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
window.openVaultInteraction = openVaultInteraction;
window.tradeShares = tradeShares;
window.openMutationHistoryOverlay = openMutationHistoryOverlay;
window.submitDistrictTax = submitDistrictTax;
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
window.releaseHostage = releaseHostage; // Existing
window.sendAllianceInvite = sendAllianceInvite;
window.acceptAlliance = acceptAlliance;
window.dissolveAlliance = dissolveAlliance;
window.openTrophyView = openTrophyView;
window.triggerGlobalKidnapEffect = triggerGlobalKidnapEffect;
window.triggerMutationScarEffect = triggerMutationScarEffect;
window.shareTournamentVictory = shareTournamentVictory;
window.adminSeasonRollover = adminSeasonRollover;
window.initiateRegionalSabotage = initiateRegionalSabotage; // New: Regional Warfare
window.adminExportAuditLog = adminExportAuditLog;
window.adminSimulateMutationSuccess = adminSimulateMutationSuccess;
window.adminSimulateMutationFailure = adminSimulateMutationFailure;
window.adminSimulateTournament = adminSimulateTournament;
window.adminCommissionAudit = adminCommissionAudit;
window.adminTaxAudit = adminTaxAudit;
window.adminDistrictTaxAudit = adminDistrictTaxAudit;
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
window.showMutationStabilityTooltip = showMutationStabilityTooltip;
window.hidePowerTooltip = hidePowerTooltip;
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
                // PILLAR 4: Sequence Restoration.
                // Restore the Replay Engine sequence count to enable seamless catch-up.
                if (beacon.profile.last_sequence_id && window.SetLastSequenceID) {
                    window.SetLastSequenceID(beacon.profile.last_sequence_id);
                }

                // PILLAR 4: Active State Restoration.
                // Re-prime the engine with the phase and board state from the beacon.
                if (beacon.profile.phase && window.SetPhase) window.SetPhase(beacon.profile.phase);
                if (beacon.profile.board && window.SetBoardState) window.SetBoardState(beacon.profile);

                // PILLAR 2: Integer Supremacy. Ensure vault_balance is passed as float for WASM.
                // The WASM engine will convert it to its internal float representation.
                // The server's authoritative micro-unit balance is used for calculations.
                if (window.SyncVaultBalance) {
                    // beacon.vault_balance is already float from app.js's syncUI
                    window.SyncVaultBalance(beacon.vault_balance);
                }
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

        // PILLAR 4: Warm-Boot Restoration.
        // If the beacon restored an 'Active' state, trigger the catch-up protocol.
        const postSyncState = window.GetGameState("combat");
        if (postSyncState && postSyncState.phase === "Active") {
            rejoinActiveMatch();
        }
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

let dashboardCache = { stateKey: "", lastBalance: -1, lastGhostActive: null };

/**
 * updateMojoDecayStatus displays the current Mojo decay rate if a stabilizer is active.
 * PILLAR 1: Infrastructure Prestige.
 */
function updateMojoDecayStatus(state) {
    const container = getEl("mojo-decay-status-hud");
    if (!container) return;

    const isStabilizerActive = state.is_mojo_stabilizer_active;
    const decayRate = state.mojo_decay_rate;

    if (!isStabilizerActive || decayRate === 0) {
        container.classList.add("hidden");
        return;
    }

    container.innerHTML = `
        <div class="glass-panel p-5-10 border-neon-cyan flex-row align-center gap-5 accelerated" 
             style="background: rgba(0, 242, 254, 0.1); height: 32px;"
             title="MOJO DECAY MITIGATION ACTIVE">
            <span class="text-neon-cyan font-bold font-size-0-7em letter-spacing-1">📉 DECAY:</span>
            <b class="text-white font-mono font-size-0-8em">${(decayRate * 100).toFixed(1)}%</b>
        </div>`;
    container.classList.remove("hidden");
}

/**
 * updateSolvencyDashboard renders a reactive indicator of the Arena's coverage ratio.
 * PILLAR 2: Ledger Integrity. Supports high-fidelity administrative overrides.
 */
function updateSolvencyDashboard(state, adminData = null) {
    const container = getEl("solvency-hud-container");
    if (!container) return;

    // PILLAR 5: Data Authority. Prioritize backend admin audit data over local WASM sync.
    const physical = adminData ? adminData.physical_vault : (state.faucet_micro || 0);
    const virtual = adminData ? adminData.virtual_liabilities : (state.total_virtual_liability || 0);
    const healthy = adminData ? adminData.kernel_healthy : true;
    const report = adminData ? adminData.audit_report : "Real-time ledger synchronization active.";

    // PILLAR 2: Coverage Calculation.
    // Default to 100% if no liabilities exist (perfectly solvent).
    const ratio = virtual > 0 ? (physical / virtual) : 1.0;
    const percent = (ratio * 100).toFixed(1);
    
    // PILLAR 2: Solvency Tiers.
    let status = ratio >= 1.0 ? "HEALTHY" : "CRITICAL";
    if (!healthy) status = "DEGRADED"; // Structural drift detected by kernel

    let color = ratio >= 1.0 ? "var(--neon-green)" : "#ff4b4b";
    if (!healthy) color = "var(--warning-orange)"; // Integrity warning color (Drift)

    container.innerHTML = `
        <div class="glass-panel p-5-10 flex-row align-center gap-5 accelerated" 
             style="background: rgba(63, 185, 80, 0.05); height: 32px; border-color: ${color}; cursor: ${adminData ? 'help' : 'default'};"
             title="${report}">
            <span style="color: ${color}; font-weight: bold; font-size: 0.7em; letter-spacing: 1px;">⚖️ COV:</span>
            <b class="text-white font-mono font-size-0-8em">${percent}%</b>
            <span class="font-xs opacity-5 ml-5" style="color: ${color}; font-weight: bold;">${status}</span>
        </div>`;
    container.classList.remove("hidden");
}

/**
 * updateVolumeSlidersUI synchronizes the range inputs with persisted values.
 * PILLAR 4: Persistence Hardening.
 */
function updateVolumeSlidersUI() {
    const mv = getEl("master-volume");
    const mu = getEl("music-volume");
    const sf = getEl("sfx-volume");
    
    // Values are retrieved from localStorage via audio.js logic
    if (mv) mv.value = localStorage.getItem('masterVolume') || 0.5;
    if (mu) mu.value = localStorage.getItem('musicVolume') || 0.5;
    if (sf) sf.value = localStorage.getItem('sfxVolume') || 0.5;
}

/**
 * displayCyberAuditReportInChat processes and displays a Cyber-Audit report in the chat.
 * This function is called by network.js when a Cyber-Audit admin_notification is received.
 * PILLAR 3: Intelligence Display.
 */
export function displayCyberAuditReportInChat(reportText) {
    // Directly render the detailed report in the chat using the game.js helper
    import('./js/game.js').then(m => m.renderChatMessage("SYSTEM", reportText));
}
window.displayCyberAuditReportInChat = displayCyberAuditReportInChat;

/**
 * triggerFoundryFusion - Special FX for Club actions.
 */
window.triggerFoundryFusion = (type) => {
    import('./js/particles.js').then(m => m.triggerFoundryFusion(type));
};

/**
 * updateDistrictStabilizerHUD renders a status widget for the active mojo stabilizer.
 * PILLAR 1: Infrastructure Prestige.
 */
function updateDistrictStabilizerHUD(state) {
    const container = getEl("district-stabilizer-hud");
    if (!container) return;

    const myClub = globalClubs[state.employer_id];
    const expiry = myClub?.buff_expirations?.["MOJO_STABILIZER"];
    const isStabilizerActive = expiry && new Date(expiry) > Date.now();

    if (!isStabilizerActive) {
        container.classList.add("hidden");
        return;
    }

    const remaining = Math.max(0, new Date(expiry) - Date.now());
    const totalHours = Math.floor(remaining / 3600000);
    const mins = Math.ceil((remaining % 3600000) / 60000);

    container.innerHTML = `
        <div class="glass-panel p-5-10 border-neon-cyan flex-row align-center gap-5 accelerated" 
             style="background: rgba(0, 242, 254, 0.1); height: 32px;"
             title="MOJO STABILIZER FIELD ACTIVE">
            <span class="text-neon-cyan font-bold font-size-0-7em letter-spacing-1">📡 STABILIZER:</span>
            <b class="text-white font-mono font-size-0-8em">${totalHours}h ${mins}m</b>
        </div>`;
    container.classList.remove("hidden");
}

/**
 * updateSabotageHUD renders a persistent countdown for owners during blackouts.
 * PILLAR 1: Regional Warfare Intelligence.
 */
function updateSabotageHUD(state) {
    const container = getEl("sabotage-hud-container");
    if (!container) return;

    const myClub = globalClubs[state.employer_id];
    const isOwner = myClub && myClub.owner_wallet && userAddress && myClub.owner_wallet.toLowerCase() === userAddress.toLowerCase();

    if (!isOwner || !myClub.buff_expirations) {
        container.classList.add("hidden");
        return;
    }

    const disruptions = Object.entries(myClub.buff_expirations)
        .filter(([key, expiry]) => key.startsWith("DISRUPTION_") && new Date(expiry) > Date.now());

    if (disruptions.length === 0) {
        container.classList.add("hidden");
        return;
    }

    // Display only the most urgent disruption (shortest time remaining)
    const mostUrgent = disruptions.sort((a, b) => new Date(a[1]) - new Date(b[1]))[0];
    const remaining = Math.max(0, new Date(mostUrgent[1]) - Date.now());
    const mins = Math.ceil(remaining / 60000);

    container.innerHTML = `
        <div class="glass-panel p-5-10 border-error animate-pulse flex-row align-center gap-5" style="background: rgba(255, 0, 255, 0.15); height: 32px;">
            <span class="text-error font-bold font-size-0-7em letter-spacing-1">📡 BLACKOUT:</span>
            <b class="text-warning font-mono font-size-0-8em">${mins}m</b>
        </div>`;
    container.classList.remove("hidden");
}

/**
 * updateMutationStabilityHUD renders the real-time success chance for lab personnel.
 * PILLAR 6: Specialized Gene-Editing.
 */
function updateMutationStabilityHUD(state) {
    const container = getEl("mutation-stability-hud");
    if (!container) return;

    const myClub = globalClubs[state.employer_id];
    if (!myClub || (myClub.type !== "Vitality" && myClub.type !== "Elemental")) {
        container.classList.add("hidden");
        return;
    }

    const mojo = myClub.club_mojo || 0;
    const staffCount = Object.keys(myClub.staff || {}).length;
    const hasInsurance = state.has_mutation_insurance;
    const isGovernor = (myClub.territories?.length || 0) + (myClub.allied_club_id ? (globalClubs[myClub.allied_club_id]?.territories?.length || 0) : 0) >= 2;
    const isSabotaged = myClub.buff_expirations?.["SABOTAGE"] && new Date(myClub.buff_expirations["SABOTAGE"]) > Date.now();
    const isTrainingActive = myClub.buff_expirations?.["STAFF_TRAINING"] && new Date(myClub.buff_expirations["STAFF_TRAINING"]) > Date.now();

    let chance = 0.70;
    if (hasInsurance) {
        chance = 1.0;
    } else {
        let mojoBonus = mojo / 5000.0;
        if (mojoBonus > 0.20) mojoBonus = 0.20;
        chance += mojoBonus;
        let staffBonus = staffCount * 0.02;
        if (staffBonus > 0.10) staffBonus = 0.10;
        chance += staffBonus;
        if (isSabotaged) chance -= 0.15;
        if (isGovernor) chance += 0.05;
        if (isTrainingActive) chance += 0.05;
        if (chance > 0.98) chance = 0.98;
        if (chance < 0.50) chance = 0.50;
    }

    const percent = Math.floor(chance * 100);
    const statusClass = percent >= 90 ? 'text-neon-green' : percent >= 70 ? 'text-neon-cyan' : 'text-warning';

    container.innerHTML = `
        <div class="glass-panel p-5-10 border-neon-purple flex-row align-center gap-5 accelerated" 
             style="background: rgba(180, 0, 255, 0.1); height: 32px; cursor: help;"
             onmouseenter="window.showMutationStabilityTooltip(event, ${mojo}, ${staffCount}, ${hasInsurance}, ${isSabotaged}, ${isGovernor}, ${isTrainingActive})"
             onmouseleave="window.hidePowerTooltip()">
            <span class="text-neon-purple font-bold font-size-0-7em letter-spacing-1">🧬 STABILITY:</span>
            <b class="${statusClass} font-mono font-size-0-8em">${percent}%</b>
        </div>`;
    container.classList.remove("hidden");
}

/**
 * updateBountyWarningHUD displays a high-priority alarm for outlaws with active bounties.
 * PILLAR 3: Criminality & Intelligence.
 */
function updateBountyWarningHUD(state) {
    const container = getEl("active-bounty-warning");
    if (!container) return;

    const wanted = state.wanted_level || 0;
    const isGhostActive = state.ghost_protocol_expires_at && new Date(state.ghost_protocol_expires_at) > Date.now();

    // PILLAR 2: Cloak Failure Trigger.
    // Detect the transition from Active to Expired while under high infamy.
    if (dashboardCache.lastGhostActive === true && !isGhostActive && wanted > 10) {
        if (window.triggerCloakFailureParticles) window.triggerCloakFailureParticles();
        if (window.playCloakFailureSFX) window.playCloakFailureSFX();
        showToast("⚠️ <b>CLOAK FAILURE:</b> Your signal is now visible on the Bounty Board!", "critical");
    }
    
    dashboardCache.lastGhostActive = isGhostActive;

    // Trigger alarm if Wanted Level is 10 or higher and signals are not scrambled.
    if (wanted > 10 && !isGhostActive) {
        container.innerHTML = `
            <div class="glass-panel p-5-10 border-error animate-pulse flex-row align-center gap-5 accelerated" 
                 style="background: rgba(255, 75, 75, 0.2); height: 32px; box-shadow: 0 0 10px rgba(255, 75, 75, 0.3);">
                <span class="text-error font-bold font-size-0-7em letter-spacing-1">🎯 BOUNTY ACTIVE:</span>
                <b class="text-white font-mono font-size-0-8em">${wanted * 50} $VBV</b>
            </div>`;
        container.classList.remove("hidden");
    } else {
        container.classList.add("hidden");
    }
}

/**
 * updateBountyHunterHUD renders tactical tracking intel for clean players.
 * PILLAR 3: Criminality & Intelligence.
 */
function updateBountyHunterHUD(state) {
    const container = getEl("bounty-hunter-hud");
    if (!container) return;

    const myWanted = state.wanted_level || 0;
    // Bounty Hunters must maintain a clean record (Wanted Level <= 2).
    if (myWanted > 2) {
        container.classList.add("hidden");
        return;
    }

    // Find high-priority targets (Wanted >= 10) who aren't under Ghost Protocol.
    const outlaws = lastLobbyPlayers.filter(p => {
        const isGhost = p.ghost_protocol_expires_at && new Date(p.ghost_protocol_expires_at) > Date.now();
        return (p.wanted_level || 0) >= 10 && !isGhost && p.id !== myClientId;
    });

    if (outlaws.length > 0) {
        const target = outlaws.sort((a, b) => b.wanted_level - a.wanted_level)[0];
        const name = getCachedEnvoiName(target.wallet);
        const district = (target.last_seen_district || "Sector Unknown").replace(/_/g, ' ').toUpperCase();

        container.innerHTML = `
            <div class="glass-panel p-5-10 border-neon-cyan flex-row align-center gap-5 accelerated" 
                 style="background: rgba(0, 242, 254, 0.1); height: 32px; cursor: pointer; border-style: dashed;"
                 title="CLICK TO ENGAGE OUTLAW"
                 onclick="window.sendChallenge('${target.id}')">
                <span class="text-neon-cyan font-bold font-size-0-7em letter-spacing-1">📡 TRACKING:</span>
                <b class="text-white font-mono font-size-0-8em">${name}</b>
                <span class="text-neon-purple font-mono font-size-0-7em ml-5">[${district}]</span>
            </div>`;
        container.classList.remove("hidden");
    } else {
        container.classList.add("hidden");
    }
}

/**
 * updateStaffTrainingVisuals applies a pulsing cyan glow to the avatar if the buff is active.
 * PILLAR 6: Specialized Gene-Editing Feedback.
 */
function updateStaffTrainingVisuals(state) {
    const avatarFrame = getEl("p1-avatar");
    if (!avatarFrame) return;

    const myClub = globalClubs[state.employer_id];
    const isTrainingActive = myClub?.buff_expirations?.["STAFF_TRAINING"] && new Date(myClub.buff_expirations["STAFF_TRAINING"]) > Date.now();

    avatarFrame.classList.toggle("buff-training-active", isTrainingActive);
    
    // Ensure the animation style is injected into the document head
    if (!document.getElementById("staff-training-glow-style")) {
        const style = document.createElement("style");
        style.id = "staff-training-glow-style";
        style.innerHTML = `
            @keyframes pulse-cyan-glow {
                0% { box-shadow: 0 0 5px var(--neon-cyan); }
                50% { box-shadow: 0 0 15px var(--neon-cyan), 0 0 25px var(--neon-cyan); }
                100% { box-shadow: 0 0 5px var(--neon-cyan); }
            }
            .buff-training-active {
                animation: pulse-cyan-glow 2s infinite !important;
                border-color: var(--neon-cyan) !important;
            }
        `;
        document.head.appendChild(style);
    }
}

/**
 * updateDistrictStabilizerVisuals triggers the shimmering grid effect if the buff is active.
 * PILLAR 1: Infrastructure Prestige.
 */
function updateDistrictStabilizerVisuals(state) {
    const myClub = globalClubs[state.employer_id];
    // Check for MOJO_STABILIZER buff expiration
    const isStabilizerActive = myClub?.buff_expirations?.["MOJO_STABILIZER"] && new Date(myClub.buff_expirations["MOJO_STABILIZER"]) > Date.now();
    
    // Trigger or remove the grid effect
    triggerDistrictStabilizerEffect(isStabilizerActive);

    // PILLAR 1: Infrastructure Audio.
    if (isStabilizerActive) { if (window.playDistrictStabilizerThrum) window.playDistrictStabilizerThrum(); }
    else { if (window.stopDistrictStabilizerThrum) window.stopDistrictStabilizerThrum(); }
}

/**
 * updateMojoDecayStatus displays the current Mojo decay rate if a stabilizer is active.
 * PILLAR 1: Infrastructure Prestige.
 */
function updateMojoDecayStatus(state) {
    const container = getEl("mojo-decay-status-hud");
    if (!container) return;

    const isStabilizerActive = state.is_mojo_stabilizer_active;
    const decayRate = state.mojo_decay_rate;

    if (!isStabilizerActive || decayRate === 0) {
        container.classList.add("hidden");
        return;
    }

    container.innerHTML = `
        <div class="glass-panel p-5-10 border-neon-cyan flex-row align-center gap-5 accelerated" 
             style="background: rgba(0, 242, 254, 0.1); height: 32px;"
             title="MOJO DECAY MITIGATION ACTIVE">
            <span class="text-neon-cyan font-bold font-size-0-7em letter-spacing-1">📉 DECAY:</span>
            <b class="text-white font-mono font-size-0-8em">${(decayRate * 100).toFixed(1)}%</b>
        </div>`;
    container.classList.remove("hidden");
}

/**
 * updateMoodCatalystVisuals applies an elemental tint to the avatar frame.
 * PILLAR 6: Specialized Gene-Editing Feedback.
 */
function updateMoodCatalystVisuals(state) {
    const avatarFrame = getEl("p1-avatar");
    if (!avatarFrame) return;

    // Reset styles to default glass state
    avatarFrame.style.boxShadow = "";
    avatarFrame.style.borderColor = "";

    // PILLAR 6: Mood Catalyst Feedback.
    const isCatalystActive = state.profile_buffs?.["mood_catalyst"] > 0;
    if (!isCatalystActive || !state.favorite_card_id) return;

    const favCard = (state.inventory || []).find(c => c.id === state.favorite_card_id);
    if (!favCard || !favCard.mood || favCard.mood === "Neutral") return;

    const moodColors = {
        "Volatile": "var(--error-red)",
        "Serene": "var(--neon-blue)",
        "Spirited": "var(--warning-orange)",
        "Grounded": "var(--neon-green)"
    };

    const color = moodColors[favCard.mood];
    if (color) {
        avatarFrame.style.boxShadow = `0 0 15px ${color}`;
        avatarFrame.style.borderColor = color;
    }
}

/**
 * updateCommissionSummaryHUD renders the total alliance dividends for Regional Governors.
 * PILLAR 1: Industrial Loop.
 */
function updateCommissionSummaryHUD(state) {
    const container = getEl("commission-summary-hud");
    if (!container) return;

    const myClubID = state.employer_id;
    const myClub = globalClubs[myClubID];
    
    // Verification: Must be the owner to see the organization's war chest summary.
    const isOwner = myClub && myClub.owner_wallet && userAddress && myClub.owner_wallet.toLowerCase() === userAddress.toLowerCase();
    
    if (!isOwner || !myClub.commission_history || myClub.commission_history.length === 0) {
        container.classList.add("hidden");
        return;
    }

    const totalEarned = myClub.commission_history.reduce((sum, event) => sum + (event.amount || 0), 0);

    if (totalEarned <= 0) {
        container.classList.add("hidden");
        return;
    }

    container.innerHTML = `
        <div class="glass-panel p-5-10 border-neon-green flex-row align-center gap-5 accelerated" 
             style="background: rgba(63, 185, 80, 0.1); height: 32px; cursor: help;"
             title="Rolling Dividend Total (Alliance Procedures)">
            <span class="text-neon-green font-bold font-size-0-7em letter-spacing-1">💰 DIVIDENDS:</span>
            <b class="text-white font-mono font-size-0-8em">${totalEarned.toFixed(2)} $VBV</b>
        </div>`;
    container.classList.remove("hidden");
}

/**
 * updateBountyTallyHUD calculates and displays the total $VBV value of all active bounties.
 * PILLAR 3: Criminality & Intelligence.
 */
function updateBountyTallyHUD() {
    const container = getEl("bounty-tally-hud");
    if (!container) return;

    let totalBounty = 0;
    lastLobbyPlayers.forEach(p => {
        const isGhost = p.ghost_protocol_expires_at && new Date(p.ghost_protocol_expires_at) > Date.now();
        if ((p.wanted_level || 0) >= 10 && !isGhost) {
            totalBounty += (p.wanted_level * 50);
        }
    });

    if (totalBounty > 0) {
        container.innerHTML = `
            <div class="glass-panel p-5-10 border-gold flex-row align-center gap-5 accelerated" 
                 style="background: rgba(255, 215, 0, 0.1); height: 32px;"
                 title="TOTAL ACTIVE BOUNTIES IN SECTOR">
                <span class="text-gold font-bold font-size-0-7em letter-spacing-1">💰 SECTOR BOUNTY:</span>
                <b class="text-white font-mono font-size-0-8em">${totalBounty} $VBV</b>
            </div>`;
        container.classList.remove("hidden");
    } else {
        container.classList.add("hidden");
    }
}

/**
 * window.syncUI - The heart of the client orchestrator.
 * Reads authoritative state from the WASM engine and updates the DOM.
 */
window.syncUI = (scope = "all", overrideData = null) => {
    const state = window.GetGameState(scope);
    if (!state) return;

    // PERFORMANCE GUARD: Detect state changes to prevent redundant re-renders
    // PILLAR 5: Precise Synchronization. Include tournament and player count in state key.
    const scannerActive = state.district_scanner_expires_at && (new Date(state.district_scanner_expires_at) > Date.now());
    const cyberJammerActive = state.has_cyber_jammer;
    const isGhostActive = state.ghost_protocol_expires_at && new Date(state.ghost_protocol_expires_at) > Date.now();
    const isQueued = state.in_matchmaking_queue;
    const isMaint = state.maintenance ? "ON" : "OFF";
    const maintPrio = state.maintenance_priority || "info";
    const currentStateKey = `${state.phase}-${state.turn}-${state.wanted_level}-${state.reputation}-${state.tournament?.matches?.length || 0}-${state.tournament?.participants?.length || 0}-${scannerActive}-${cyberJammerActive}-${isGhostActive}-${isQueued}-${isMaint}-${maintPrio}`;

    if (scope !== "combat" && scope !== "solvency_override" && currentStateKey === dashboardCache.stateKey && state.faucet === dashboardCache.lastBalance) {
        // Only proceed if scope is combat or state has evolved
        if (scope !== "all") return;
    }
    dashboardCache.stateKey = currentStateKey;
    dashboardCache.lastBalance = state.faucet;

    // PILLAR 4: Critical Alerts.
    // The native VOI 'gas_warning' toast is handled by the 'admin_notification'
    // system (network.js -> ui.js:showToast), not directly by syncUI.
    // syncUI primarily updates persistent UI elements based on GetGameState.
    // PILLAR 2: Usable Total.
    // Combine physical blockchain balance ($VBV) with virtual salary/heist rewards.
    // PILLAR 5: Delta-Safe Sync. Only update liquidity if values are present in scope.
    // PILLAR 2: Integer Supremacy alignment. state.faucet is the Arena's physical balance.
    const physicalVBV = state.faucet;
    const virtualVBV = state.virtual_balance;
    const totalLiquid = (physicalVBV !== undefined && virtualVBV !== undefined) ? (physicalVBV + virtualVBV) : null;

    const liquidEl = getEl("total-liquid-balance");
    if (liquidEl && totalLiquid !== null) liquidEl.innerText = totalLiquid.toFixed(2);

    updateVolumeSlidersUI(); // Ensure volume sliders reflect persisted values

    // --- Domain Orchestration ---
    if (scope === "combat" || scope === "all") {
        updateDynamicArenaFloor(state);
        syncBoardParticles(state);
        updateSpectatorHUD(state);

        // PILLAR 4: Session Identity styling.
        updateAvatarIdentityStyle(state);
        updateStaffTrainingVisuals(state);
        updateMoodCatalystVisuals(state);

        // PILLAR 3: Interaction Lock Reset.
        // authoritative update received; clear the quick-cast lock.
        setPendingQuickCastId(null);

        // PILLAR 5: Granular Node-Diffing (Combat Grid).
        // Update individual grid slots to preserve the .onclick listeners 
        // established in game.js:buildEmptyBoard.
        const slots = document.querySelectorAll(".grid-slot");
        if (state.board && slots.length === 9) {
            state.board.forEach((card, i) => {
                const slot = slots[i];
                if (!card) {
                    if (slot.innerHTML !== "") slot.innerHTML = "";
                    slot.classList.remove("owner-0", "owner-1", "common", "rare", "epic", "legendary");
                    return;
                }
                
                const cardHTML = renderCardHTML(card);
                if (slot.innerHTML !== cardHTML) {
                    slot.innerHTML = cardHTML;

                    // PILLAR 5: Visual Authority. 
                    // Apply owner and rarity classes to the slot for CSS targeting.
                    slot.classList.remove("owner-0", "owner-1", "common", "rare", "epic", "legendary");
                    slot.classList.add(`owner-${card.owner}`);
                    
                    const r = card.rarity || 1.0;
                    if (r >= 2.0) slot.classList.add("legendary");
                    else if (r >= 1.5) slot.classList.add("epic");
                    else if (r > 1.0) slot.classList.add("rare");
                    else slot.classList.add("common");
                    
                    // PILLAR 4: Reactive Captured Animation.
                    if (card.is_combo) {
                        slot.classList.remove("flip-capture");
                        void slot.offsetWidth; // Force Reflow to re-trigger animation
                        slot.classList.add("flip-capture");
                    } else {
                        slot.classList.remove("flip-capture");
                    }
                }
            });
        }
        
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
        updateSabotageHUD(state);
        updateMutationStabilityHUD(state);
        updateDistrictStabilizerHUD(state);
    }

    // PILLAR 5: Matchmaking Interface Sync.
    // Ensure the button state and status text reflect the authoritative WASM engine state.
    const matchmakingBtn = getEl("btn-matchmaking");
    const queueStatus = getEl("queue-status");
    if (matchmakingBtn && queueStatus && state.in_matchmaking_queue !== undefined) {
        matchmakingBtn.disabled = false; // PILLAR 5: Re-enable once authoritative state is synced
        if (state.in_matchmaking_queue) {
            matchmakingBtn.innerText = "Leave Queue";
            matchmakingBtn.style.background = "var(--neon-purple)";
            queueStatus.innerHTML = `<span class="status-active">SEARCHING FOR OPPONENT...</span>`;
        } else {
            matchmakingBtn.innerText = "Join Matchmaking Pool";
            matchmakingBtn.style.background = "";
            queueStatus.innerText = "Ready for automatic pairing?";
        }
    }

    if (scope === "meta" || scope === "all") {
        handleTournamentUI(state.tournament);
        if (state.tournament && state.tournament.active) {
            renderTournamentBracket(state.tournament);
        }
        updateMapStatusIndicators();
        updateSolvencyDashboard(state, overrideData);
    }

    if (scope === "solvency_override" && overrideData) {
        updateSolvencyDashboard(state, overrideData);
    }

    if (scope === "inventory" || scope === "all") {
        renderDeckManager(state);
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
    // Cache the authoritative push to localStorage to support warm-boot restoration.
    if (scope === "all" && (state.phase === "Lobby" || state.phase === "Active") && userAddress) {
        const beaconData = {
            profile: state,
            vault_balance: state.faucet, // PILLAR 2: Synchronize with WASM export key
            maintenance_priority: state.maintenance_priority, // PILLAR 4: Critical Alert state preservation
            match_id: state.match_id, // PILLAR 3: Standardized identification persistence
            rewards: state.rewards,
            clubs: state.clubs,
            ts: Date.now()
        };
        localStorage.setItem("vbabes_state_beacon", JSON.stringify(beaconData));
    }
};
