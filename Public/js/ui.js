// Public/js/ui.js

import { CONFIG } from './config.js'; // Removed triggerMoodMote from here
import { triggerGlobalKidnapEffect, triggerMutationScarEffect, triggerCloakDisruptorParticles, triggerMutationSuccessParticles, triggerStaffTrainingEffect, triggerMoodMote, triggerContractCompleteEffect } from './particles.js';
import { myClientId, currentLatency, lastPingTime, setLastPingTime, setCurrentLatency } from './network.js';
import { userAddress } from './wallet.js'; // userAddress is now in wallet.js
import { activeCardId, pendingQuickCastId, myPlayerIndex, currentOpponentId, spectatorMatchState, lastTauntPhase, lastTauntTurn, setLastTauntPhase, setLastTauntTurn, matchHistorySaved, setMatchHistorySaved, saveMatchResult, renderChatMessage, reportGloat, lastLobbyPlayers } from './game.js';
import { masterVolume, musicVolume, sfxVolume, playProcedureInterruptedSFX, playLongWarningSFX, playMutationSuccessSFX, playCloakDisruptorSFX, playEcosystemAlertSFX, playMoodMoteSFX, playStaffTrainingSFX, playSabotageReparationSFX } from './audio.js';
import { updateAdminRewardList, fetchAdminLogs, adminLogTicker, startAdminLogPolling, stopAdminLogPolling, globalClubs, availableNetworks } from './admin.js';
import { updateActiveRumors, renderRumBoard, initiateBail, deployTrap, payRansom, releaseHostage, spreadRumor, currentRewardRatio } from './criminality.js';
import { seasonEnd, totalTournaments, tournamentLimit, currentTournamentPage, fetchTournamentHistory, fetchSeasonHistory } from './leaderboard.js';
import { getAssetSymbol, getCachedEnvoiName, resolveEnvoiName, assetCache, resolveAssetSymbol, shortenAddress } from './utils.js';
import { buyClubItem, submitClubFoundry, tradeShares, buyBlackMarketItem, submitConsignment, takeLease, submitDistrictTax, TERRITORY_MAP, MOOD_CLASS_MAP, MOOD_EMOJI_MAP } from './economy.js';

export let tooltipEl = document.getElementById("power-tooltip");
const cardHTMLPool = new Map();
export let maintenanceTicker = null;
export let districtScannerTimerInterval = null;
export let seasonTimerInterval = null;
let lastMaintExpiry = null;
let lastBanExpiry = null;
let lastSeasonExpiry = null;
export let banTicker = null; // PILLAR 3: Moderation HUD

export let mapZoom = 1.0;

/**
 * adjustMapZoom modifies the scale of the 3D map.
 * PILLAR 4: Immersion. Persists DOM classes by only adjusting parent transform.
 */
export function adjustMapZoom(delta) {
    mapZoom = Math.min(2, Math.max(0.5, mapZoom + delta));
    const grid = document.getElementById("map-3d-grid");
    if (grid) grid.style.transform = `rotateX(30deg) rotateY(-15deg) scale(${mapZoom})`;
}

/**
 * updateMissionHUD renders a faction-aware widget in the top bar when a mission/contract is active.
 * PILLAR 3: Criminality & Intelligence.
 */
export function updateMissionHUD(state) {
    const container = document.getElementById("mission-hud-container");
    if (!container) return;

    const activeJM = state.active_justice_mission_id || "";
    const activeUC = state.active_underworld_contract_id || "";

    if (activeJM) {
        let label = activeJM;
        // PILLAR 3: Targeted Identification for High-Tier Justice Missions.
        if (activeJM === "MISSION-023") label = "MISSION-023 [ULTIMATE SIG]";
        else if (activeJM === "MISSION-022") label = "MISSION-022 [SOVEREIGN SIG]";
        else if (activeJM === "MISSION-027") label = "MISSION-027 [BREACH SIG]";
        else if (activeJM === "MISSION-025") label = "MISSION-025 [AUDIT SIG]";
        else if (activeJM === "MISSION-012") label = "MISSION-012 [GOVERNOR SIG]";
        else if (activeJM === "MISSION-011") label = "MISSION-011 [LEADERSHIP SIG]";
        else if (activeJM === "MISSION-013") label = "MISSION-013 [KINGPIN SIG]";
        else if (activeJM === "MISSION-014") label = "MISSION-014 [AUDIT SIG]";
        else if (activeJM === "MISSION-015" || activeJM === "MISSION-016") label = `${activeJM} [HEGEMONY SIG]`;
        else if (activeJM === "MISSION-017") label = "MISSION-017 [CHAOS SIG]";
        else if (activeJM === "MISSION-018" || activeJM === "MISSION-021") label = `${activeJM} [SYNDICATE SIG]`;
        else if (activeJM === "MISSION-019") label = "MISSION-019 [LAUNDRY SIG]";
        else if (activeJM === "MISSION-020" || activeJM === "MISSION-024") label = `${activeJM} [APEX SIG]`;
        else if (activeJM === "MISSION-009") label = "MISSION-009 [SECURITY SIG]";
        else if (activeJM.startsWith("MISSION-010")) {
            const parts = activeJM.split(":");
            label = parts.length > 1 ? "MISSION-010 [ACCOMPLICE SIG]" : "MISSION-010";
        }

        container.innerHTML = `
            <div class="mission-hud pulse glass-panel" style="border-color: var(--neon-cyan); --mission-color: var(--neon-cyan);">
                <span class="text-neon-cyan font-bold font-size-0-7em letter-spacing-1">⚖️ JUSTICE MISSION:</span>
                <b class="text-white font-mono font-size-0-8em">${label}</b>
            </div>
        `;
        container.classList.remove("hidden");
    } else if (activeUC) {
        let label = activeUC;
        // PILLAR 3: Targeted Identification for High-Tier Underworld Contracts.
        if (activeUC === "CONTRACT-025") label = "CONTRACT-025 [KIDNAPPER SIG]";
        else if (activeUC === "CONTRACT-027") label = "CONTRACT-027 [SMUGGLER SIG]";
        else if (activeUC === "CONTRACT-021") label = "CONTRACT-021 [SYNDICATE SIG]";
        else if (activeUC === "CONTRACT-024") label = "CONTRACT-024 [FENCE SIG]";
        else if (activeUC === "CONTRACT-020" || activeUC === "CONTRACT-023") label = `${activeUC} [APEX SIG]`;
        else if (activeUC === "CONTRACT-019") label = "CONTRACT-019 [CHAOS SIG]";
        else if (activeUC === "CONTRACT-015" || activeUC === "CONTRACT-016") label = `${activeUC} [HEGEMONY SIG]`;
        else if (activeUC === "CONTRACT-014" || activeUC === "CONTRACT-022") label = `${activeUC} [SOVEREIGN SIG]`; // Sovereign is for heist
        else if (activeUC === "CONTRACT-013") label = "CONTRACT-013 [PREMIER SIG]";
        else if (activeUC === "CONTRACT-012") label = "CONTRACT-012 [STABILIZER SIG]";
        else if (activeUC === "CONTRACT-017") label = "CONTRACT-017 [LIBERATION SIG]";
        else if (activeUC === "CONTRACT-018") label = "CONTRACT-018 [FORTRESS SIG]";
        else if (activeUC === "CONTRACT-011") label = "CONTRACT-011 [TITAN SIG]";
        else if (activeUC === "CONTRACT-010") label = "CONTRACT-010 [GOVERNOR SIG]";

        container.innerHTML = `
            <div class="mission-hud pulse glass-panel" style="border-color: var(--warning-orange); --mission-color: var(--warning-orange);">
                <span class="text-warning font-bold font-size-0-7em letter-spacing-1">💀 UNDERWORLD CONTRACT:</span>
                <b class="text-white font-mono font-size-0-8em">${label}</b>
            </div>
        `;
        container.classList.remove("hidden");
    } else {
        container.classList.add("hidden");
        container.innerHTML = "";
    }
}

/**
 * updateAvatarIdentityStyle applies a "local-player" border to the correct avatar frame.
 * PILLAR 4: Session Identity.
 */
export function updateAvatarIdentityStyle(state) {
    const p1Frame = document.getElementById("p1-avatar");
    const p2Frame = document.getElementById("p2-avatar");
    if (!p1Frame || !p2Frame) return;

    // Reset styles
    p1Frame.classList.remove("local-player-frame");
    p2Frame.classList.remove("local-player-frame");
    // Clear all faceplate classes
    p1Frame.classList.remove("faceplate-neon_vibe", "faceplate-shadow", "faceplate-governor");
    p2Frame.classList.remove("faceplate-neon_vibe", "faceplate-shadow", "faceplate-governor");

    // Apply high-visibility border to the local player's frame
    const localIdx = state.local_player_index || 0;
    const target = localIdx === 0 ? p1Frame : p2Frame;
    target.classList.add("local-player-frame");

    // PILLAR 4: Multi-Player Cosmetic Identity.
    // Apply faceplate styling to both frames if provided in the combat snapshot.
    if (state.p1_faceplate) p1Frame.classList.add(`faceplate-${state.p1_faceplate}`);
    if (state.p2_faceplate) p2Frame.classList.add(`faceplate-${state.p2_faceplate}`);

    // Update image sources for the lobby/combat frames
    const p1Img = document.getElementById("p1-avatar-img");
    const p2Img = document.getElementById("p2-avatar-img");
    if (p1Img && state.p1_avatar) p1Img.src = state.p1_avatar;
    if (p2Img && state.p2_avatar) p2Img.src = state.p2_avatar;

    // PILLAR 4: Multi-Slot Card Identity.
    // Apply identity markers to grid slots so that cards on the board reflect local ownership.
    // This ensures that during capture events, the visual theme of the card slot reactively
    // shifts to reflect whether the local player or the opponent now controls the position.
    const slots = document.querySelectorAll(".grid-slot");
    if (state.board && slots.length > 0) {
        state.board.forEach((card, i) => {
            const slot = slots[i];
            if (!slot) return;
            slot.classList.remove("local-owner-slot");
            if (card && card.owner === localIdx) {
                slot.classList.add("local-owner-slot");
            }
        });
    }
}

/**
 * updateStaffTrainingVisuals applies a pulsing cyan glow to the local player's avatar.
 * PILLAR 6: Specialized Gene-Editing Feedback.
 */
export function updateStaffTrainingVisuals(state) {
    const localIdx = state.local_player_index || 0;
    const avatarFrame = document.getElementById(`p${localIdx + 1}-avatar`);
    if (!avatarFrame) return;

    // Buff state is organization-scoped. Check if the player belongs to a lab.
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
            .local-player-frame {
                border: 2px solid var(--neon-cyan) !important;
                box-shadow: 0 0 10px var(--neon-cyan);
            }
        `;
        document.head.appendChild(style);
    }
}

/**
 * updateMoodCatalystVisuals applies an elemental tint to the local player's avatar frame.
 * PILLAR 6: Specialized Gene-Editing Feedback.
 */
export function updateMoodCatalystVisuals(state) {
    const localIdx = state.local_player_index || 0;
    const avatarFrame = document.getElementById(`p${localIdx + 1}-avatar`);
    if (!avatarFrame) return;

    // Reset mood-specific properties (Identity border is handled by updateAvatarIdentityStyle)
    avatarFrame.style.boxShadow = "";
    if (!avatarFrame.classList.contains("local-player-frame")) {
        avatarFrame.style.borderColor = "";
    }

    // PILLAR 6: Mood Catalyst Feedback.
    const isCatalystActive = state.profile_buffs?.["mood_catalyst"] > 0;
    if (!isCatalystActive || !state.favorite_card_id) return;

    // Profile data returned by GetGameState is already scoped to local_player_index
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

// --- Transaction Feedback (Toast) ---
export function showToast(message, type = 'info', duration = 5000) {
    const container = document.getElementById("toast-container");
    const toast = document.createElement("div");
    toast.className = `toast ${type}`;
    toast.innerHTML = message;
    container.appendChild(toast);

    // PILLAR 1, 5 & 6: Immersive Feedback.
    // Trigger global particle effects for critical gameplay and economic events.
    if (type === "critical" || message.includes("KIDNAP GAMBIT") || message.includes("HOSTAGE SECURED") || message.includes("MUTATION FAILURE") || message.includes("TERRITORY INVASION") || message.includes("CLOAK DISRUPTED") || message.includes("MUTATION SUCCESS") || message.includes("ECOSYSTEM GUARDIAN") || message.includes("STAFF TRAINING ACTIVE") || message.includes("REPARATION RECEIVED") || message.includes("CONTRACT COMPLETED")) {
        if (message.includes("MUTATION FAILURE")) {
            triggerMutationScarEffect();
            playProcedureInterruptedSFX();
        } else if (message.includes("TERRITORY INVASION")) {
            playLongWarningSFX();
        } else if (message.includes("MUTATION SUCCESS")) {
            triggerMutationSuccessParticles();
            playMutationSuccessSFX();
        } else if (message.includes("CLOAK DISRUPTED")) {
            triggerCloakDisruptorParticles();
            playCloakDisruptorSFX();
        } else if (message.includes("ECOSYSTEM GUARDIAN")) {
            if (window.triggerEcosystemAlertVisuals) window.triggerEcosystemAlertVisuals();
            playEcosystemAlertSFX();
        } else if (message.includes("STAFF TRAINING ACTIVE")) {
            triggerStaffTrainingEffect();
            playStaffTrainingSFX();
        } else if (message.includes("REPARATION RECEIVED")) {
            playSabotageReparationSFX();
        } else if (message.includes("CONTRACT COMPLETED")) {
            // PILLAR 3: Underworld mission completion signature.
            toast.classList.add("contract-completed-flourish");
            triggerContractCompleteEffect();
        } else {
            triggerGlobalKidnapEffect();
        }

        if (type === "critical") duration = 8000; // Longer duration for critical alerts
    }

    if (duration > 0) {
        setTimeout(() => {
            toast.style.opacity = '0';
            toast.style.transform = 'translateX(100%)';
            toast.style.transition = '0.5s';
            setTimeout(() => toast.remove(), 500);
        }, duration);
    }
}

export function openTerritoryMapOverlay() {
    const grid = document.getElementById("map-3d-grid");
    if (!grid) return;
    mapZoom = 1.0;
    grid.style.transform = `rotateX(30deg) rotateY(-15deg) scale(${mapZoom})`;
    grid.innerHTML = "";

    // PILLAR 3: Intelligence Integration.
    const state = window.GetGameState();

    // PILLAR 4: Replay Resilience.
    // Suppress tactical intel during catch-up to prevent information leaks or UI ghosting.
    const isSynchronized = !state.replay_state || state.replay_state === "SYNCHRONIZED";
    const scannerActive = isSynchronized && state.district_scanner_expires_at && new Date(state.district_scanner_expires_at) > Date.now();

    // Start/stop timer based on state
    startDistrictScannerTimer(state.district_scanner_expires_at); // Start/stop timer based on state

    TERRITORY_MAP.forEach(t => {
        const club = Object.values(globalClubs).find(c => c.territories && c.territories.includes(t.id));
        const isOwned = !!club;
        const isGovernor = isOwned && club.region_name;

        // Detect hardware traps if intelligence window is open
        const hasTraps = isOwned && scannerActive && Object.keys(club.active_buffs || {}).some(k => k.startsWith("TRAP_"));

        // Detect regional disruption (Cyber-Lock)
        const disruptionKey = "DISRUPTION_" + t.id;
        const isDisrupted = isOwned && club.buff_expirations && club.buff_expirations[disruptionKey] && new Date(club.buff_expirations[disruptionKey]) > Date.now();

        let isUnderAttack = false;
        if (isOwned && club.last_heist_at) {
            isUnderAttack = (Date.now() - new Date(club.last_heist_at).getTime()) < 300000;
        }
        const tile = document.createElement("div");
        tile.className = `map-tile-3d accelerated ${isGovernor ? 'governor-controlled' : isOwned ? 'controlled' : 'neutral'}`;
        tile.onclick = () => { hideAllOverlays(); openTerritoryView(t.id); };
        tile.innerHTML = `
            <div class="tile-label">
                <div class="tile-icon">${t.icon}</div>
                <div class="tile-name">${t.name.toUpperCase()}</div>
                <div class="tile-owner">${isOwned ? club.name : 'NEUTRAL ZONE'}</div>
                ${isOwned ? `<div class="tile-stats"><span class="stat population">${Object.keys(club.staff || {}).length}</span><span class="stat resources">${club.treasury.toFixed(0)}</span></div>` : ''}
            </div>
            <div class="tile-status ${isUnderAttack ? 'under-attack' : (isDisrupted ? 'cyber-locked' : (hasTraps ? 'trap-detected' : (isGovernor ? 'developing' : '')))}"></div>`;
        grid.appendChild(tile);
    });
    document.getElementById("territory-map-overlay").classList.remove("hidden");
}

/**
 * updateMapStatusIndicators - Reactively updates the world map tile indicators.
 * Called by syncUI during server broadcasts to reflect real-time intelligence (Traps/Attacks).
 */
export function updateMapStatusIndicators() {
    const overlay = document.getElementById("territory-map-overlay");
    if (!overlay || overlay.classList.contains("hidden")) return;

    const state = window.GetGameState();

    // PILLAR 4: Replay Resilience.
    // Suppress tactical intel during catch-up to prevent information leaks or UI ghosting.
    const isSynchronized = !state.replay_state || state.replay_state === "SYNCHRONIZED";
    const scannerActive = isSynchronized && state.district_scanner_expires_at && new Date(state.district_scanner_expires_at) > Date.now();

    startDistrictScannerTimer(state.district_scanner_expires_at); // Ensure timer is managed
    const grid = document.getElementById("map-3d-grid");
    if (!grid) return;

    const tiles = grid.querySelectorAll(".map-tile-3d");
    TERRITORY_MAP.forEach((t, idx) => {
        const tile = tiles[idx];
        if (!tile) return;

        const club = Object.values(globalClubs).find(c => c.territories && c.territories.includes(t.id));
        const isOwned = !!club;
        const isGovernor = isOwned && club.region_name;
        const hasTraps = isOwned && scannerActive && Object.keys(club.active_buffs || {}).some(k => k.startsWith("TRAP_"));
        
        // Detect regional disruption (Cyber-Lock)
        const disruptionKey = "DISRUPTION_" + t.id;
        const isDisrupted = isOwned && club.buff_expirations && club.buff_expirations[disruptionKey] && new Date(club.buff_expirations[disruptionKey]) > Date.now();

        let isUnderAttack = false;
        if (isOwned && club.last_heist_at) {
            isUnderAttack = (Date.now() - new Date(club.last_heist_at).getTime()) < 300000;
        }

        // PILLAR 4: Reactive Immersion. Update tile class to reflect rank shifts (e.g. Governor upgrades).
        tile.className = `map-tile-3d accelerated ${isGovernor ? 'governor-controlled' : isOwned ? 'controlled' : 'neutral'}`;

        // Update owner text and status indicators
        const ownerEl = tile.querySelector(".tile-owner");
        if (ownerEl) ownerEl.innerText = isOwned ? club.name : 'NEUTRAL ZONE';

        const statusEl = tile.querySelector(".tile-status");
        if (statusEl) {
            statusEl.className = `tile-status ${isUnderAttack ? 'under-attack' : (isDisrupted ? 'cyber-locked' : (hasTraps ? 'trap-detected' : (isGovernor ? 'developing' : '')))}`;
        }
    });
}

/**
 * startDistrictScannerTimer manages the countdown display for the District Scanner.
 * PILLAR 3: Intelligence Integration.
 */
let activeScannerExpiry = null;

export function startDistrictScannerTimer(expiresAt) {
    const timerWidget = document.getElementById("district-scanner-timer-widget");
    const countdownEl = document.getElementById("district-scanner-countdown");
    if (!timerWidget || !countdownEl) return;

    // PILLAR 5: Efficiency Guard.
    if (activeScannerExpiry === expiresAt && districtScannerTimerInterval) return;
    activeScannerExpiry = expiresAt;

    if (districtScannerTimerInterval) {
        clearInterval(districtScannerTimerInterval);
        districtScannerTimerInterval = null;
    }

    const expiryTime = (expiresAt && !isNaN(new Date(expiresAt).getTime())) ? new Date(expiresAt).getTime() : 0;

    // Guard: If date is invalid or already in the past, ensure UI is hidden and abort
    if (!expiryTime || expiryTime <= Date.now()) {
        if (timerWidget) timerWidget.classList.add("hidden");
        activeScannerExpiry = null;
        return;
    }

    const updateCountdown = () => {
        const now = Date.now();
        const diff = expiryTime - now;

        if (diff <= 0) {
            timerWidget.classList.add("hidden");
            countdownEl.innerText = "00:00";
            clearInterval(districtScannerTimerInterval);
            activeScannerExpiry = null;
            return;
        }

        const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
        const seconds = Math.floor((diff % (1000 * 60)) / 1000);
        countdownEl.innerText = `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
        timerWidget.classList.remove("hidden");
    };
    updateCountdown(); // Initial call
    districtScannerTimerInterval = setInterval(updateCountdown, 1000);
}

// PILLAR 4: Browser Graceful Resumption.
// Handle tab suspension/thawing by refreshing the countdown immediately when the tab becomes visible.
document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'visible' && activeScannerExpiry) {
        // Re-invoking handles immediate UI refresh and interval restart with fresh Date.now()
        startDistrictScannerTimer(activeScannerExpiry);
    }
});

export function openTerritoryView(territoryId) {
    const state = window.GetGameState();
    const club = Object.values(globalClubs).find(c => c.territories && c.territories.includes(territoryId));
    const overlay = document.createElement("div");
    overlay.id = "territory-view-overlay";
    overlay.className = "overlay";
    let body = `<p style="opacity: 0.7;">This territory is currently unclaimed. Found a Club to take control!</p>`;
    
    // PILLAR 1: Political Influence.
    // Regional Governors can adjust tax policy for their districts.
    let taxUI = "";
    if (club) {
        const userRole = state.job_role || "";
        const isGovernor = (club.territories?.length || 0) + (club.allied_club_id ? (globalClubs[club.allied_club_id]?.territories?.length || 0) : 0) >= 2;
        
        let itemsHTML = "";
        Object.entries(club.inventory || {}).forEach(([itemId, qty]) => {
            if (qty <= 0) return;
            const meta = GlobalShopRegistry[itemId];
            if (!meta) return;

            const meetsMojo = (club.mojo || 0) >= (meta.requiredMojo || 0);
            const meetsRole = !meta.requiredRole || userRole === meta.requiredRole;
            const meetsMaster = !meta.isMasterTier || isGovernor;
            const isLocked = !meetsMojo || !meetsRole || !meetsMaster;

            let reqLabels = [];
            if (meta.requiredMojo) reqLabels.push(`<span class="${meetsMojo ? 'text-neon-green' : 'text-error'}">MOJO ${meta.requiredMojo}+</span>`);
            if (meta.requiredRole) reqLabels.push(`<span class="${meetsRole ? 'text-neon-green' : 'text-error'}">${meta.requiredRole.toUpperCase()}</span>`);
            if (meta.isMasterTier) reqLabels.push(`<span class="${meetsMaster ? 'text-neon-green' : 'text-error'}">GOVERNOR</span>`);

            itemsHTML += `
                <div class="shop-item-row glass-panel p-10 m-0 flex-row justify-between align-center ${meta.isMasterTier ? 'master-tier' : ''} ${isLocked ? 'opacity-5 locked-item' : ''}">
                    <div class="text-left">
                        <b class="${meta.isMasterTier ? 'text-gold' : 'text-white'}">${meta.name.toUpperCase()}</b>
                        ${reqLabels.length > 0 ? `<div class="font-size-0-7em font-bold mt-2" style="letter-spacing: 1px;">${reqLabels.join(' • ')}</div>` : ''}
                        <div class="font-size-0-75em opacity-6">${meta.desc}</div>
                    </div>
                    <div class="text-right">
                        <button class="outline btn-small ${isLocked ? 'border-grey text-grey' : (meta.isMasterTier ? 'border-gold text-gold' : 'border-neon-cyan text-neon-cyan')}" 
                                ${isLocked ? 'disabled' : ''}
                                onclick="buyClubItem('${club.id}', '${itemId}', ${meta.price}, '${territoryId}')">
                            ${isLocked ? 'LOCKED' : `${meta.price} $VBV`}
                        </button>
                    </div>
                </div>`;
        });
        body = `<div class="flex-col gap-10">${itemsHTML || '<div class="opacity-3 py-20 italic">District shop inventory is depleted.</div>'}</div>`;

        const isOwnerOfDistrict = userAddress && club.owner_wallet && club.owner_wallet.toLowerCase() === userAddress.toLowerCase();
        const combinedCount = (club.territories?.length || 0) + (club.allied_club_id ? (globalClubs[club.allied_club_id]?.territories?.length || 0) : 0);
        const isGovernor = isOwnerOfDistrict && combinedCount >= 2;

        const targetOwnerStats = lastLobbyPlayers.find(p => p.wallet?.toLowerCase() === club.owner_wallet?.toLowerCase());
        const isHardened = targetOwnerStats && (targetOwnerStats.reparations_received_count || 0) >= 5;

        if (isGovernor) {
            const hardenedHTML = isHardened ? `<div class="badge-hardened text-gold font-bold font-size-0-7em mb-10 pulse" style="border: 1px solid gold; padding: 2px 8px; border-radius: 4px; display: inline-block;">🛡️ HARDENED SECURITY ACTIVE</div>` : '';
            taxUI = `
                <div class="glass-panel p-15 m-0 border-gold mt-15" style="background: rgba(212, 175, 55, 0.1);">
                    ${hardenedHTML}
                    <div class="text-gold font-bold font-size-0-8em mb-10 letter-spacing-1">🏛️ DISTRICT TAX POLICY</div>
                    <div class="flex-row align-center gap-10 mb-10">
                        <input type="number" id="district-tax-input" class="glass-input flex-1" placeholder="Rate (0-20)" min="0" max="20" step="0.5">
                        <span class="font-bold opacity-7">%</span>
                    </div>
                    <button class="w-full bg-gold text-dark font-bold font-size-0-8em" onclick="submitDistrictTax('${territoryId}')">ENACT POLICY</button>
                    <div class="font-size-0-6em opacity-5 mt-5 italic">Changes incur a 1% Governor Surcharge on Treasury.</div>
                </div>`;
        }
    }
    overlay.innerHTML = `<div class="glass-panel medium" style="text-align: center;"><h2>TERRITORY: ${territoryId.replace('_',' ').toUpperCase()}</h2>${body}${taxUI}
        <div class="mt-20"><button class="outline" onclick="document.getElementById('territory-view-overlay').remove()">CLOSE</button>${!club ? `<button onclick="document.getElementById('territory-view-overlay').remove(); openClubFoundry()">FOUND CLUB</button>` : ''}</div></div>`;
    document.body.appendChild(overlay);
}

// Global function to manage transaction status display
export function setTransactionStatus(message, type = 'info') {
    const statusEl = document.getElementById("transaction-status");
    if (!statusEl) return;

    // Reset visibility and priority classes
    statusEl.classList.remove("priority-critical", "priority-warning");

    if (message) {
        statusEl.classList.remove("hidden");
        const isCritical = type === 'critical';
        if (isCritical) statusEl.classList.add("priority-critical");
        if (type === 'warning') statusEl.classList.add("priority-warning");

        const colorMap = {
            'error': 'var(--error-red, #ff4b4b)',
            'critical': 'var(--error-red, #ff4b4b)',
            'success': 'var(--neon-green)',
            'info': 'var(--neon-cyan)',
            'warning': '#ffd700'
        };

        // PILLAR 4: High-Visibility Hardening.
        // Apply bold weights and uppercase styling for critical admin-level alerts.
        const fontWeight = isCritical ? 'bold' : 'normal';
        const textTransform = isCritical ? 'uppercase' : 'none';
        const letterSpacing = isCritical ? '1px' : 'normal';

        statusEl.innerHTML = `<span style="color: ${colorMap[type] || 'white'}; font-weight: ${fontWeight}; text-transform: ${textTransform}; letter-spacing: ${letterSpacing};">${message}</span>`;
    } else {
        statusEl.classList.add("hidden");
        statusEl.innerHTML = "";
    }
}

export function hideAllOverlays() {
    document.querySelectorAll('.overlay').forEach(el => el.classList.add('hidden'));
}

// Function to show the main game container and hide other overlays
export function showMainGameContainer() {
    document.getElementById("main-game-container").classList.remove("hidden");
}

export function highlightStartButton(isReady) {
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
}

export function handleMaintenanceUI(active, targetTimestamp, priority = "info") {
    const bar = document.getElementById("maintenance-bar");
    const timerDisplay = document.getElementById("maintenance-timer");

    // PILLAR 5: Efficiency Guard. Prevent restarting the interval if the target hasn't changed.
    if (active && lastMaintExpiry === targetTimestamp && maintenanceTicker) return;
    lastMaintExpiry = targetTimestamp;

    if (maintenanceTicker) clearInterval(maintenanceTicker);

    if (window.SetMaintenanceState) window.SetMaintenanceState(active, priority);

    if (!active) {
        bar.classList.add("hidden");
        return;
    }

    bar.classList.remove("hidden");
    // PILLAR 4: Critical Alerts. Apply high-visibility styling based on priority.
    bar.classList.remove("priority-critical", "priority-warning");
    if (priority === "critical") bar.classList.add("priority-critical");
    if (priority === "warning") bar.classList.add("priority-warning");

    const targetTime = new Date(targetTimestamp).getTime();

    const tick = () => {
        const now = Date.now();
        const diff = targetTime - now;

        if (diff <= 0) {
            timerDisplay.innerText = "STARTING NOW";
            clearInterval(maintenanceTicker);
            maintenanceTicker = null;
            return;
        }

        const mins = Math.floor(diff / 60000);
        const secs = Math.floor((diff % 60000) / 1000); // FIXED: minutes was undefined
        timerDisplay.innerText = `${String(mins).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
    };

    tick();
    maintenanceTicker = setInterval(tick, 1000);
}

/**
 * Orchestrates ambient board effects (Mood Motes) based on tile state.
 * Throttled to ensure subtlety and prevent performance degradation.
 */
export function syncBoardParticles(state) {
    if (state.phase !== "Active" || !state.board_moods) return;

    state.board_moods.forEach((mood, idx) => {
        if (mood && mood !== "Neutral") {
            // Ambient Trigger: Only spawn on ~15% of sync cycles to keep the effect sparse
            if (Math.random() > 0.85) { // PILLAR 5: Throttled for performance
                triggerMoodMote(idx, mood);
                playMoodMoteSFX(mood);
            }
        }
    });
}

export function showTournamentTransition(roundNumber) {
    const overlay = document.getElementById("tournament-transition-overlay");
    if (!overlay) return;
    
    overlay.querySelector(".round-number-display").innerText = `ROUND ${roundNumber}`;
    overlay.classList.remove("hidden");

    if (window.PlaySound) {
        window.PlaySound('Pay_out-in.mp3');
    }

    setTimeout(() => overlay.classList.add("hidden"), 3000);
}

export function updateDynamicArenaFloor(state) { 
    let texture = "var(--texture-solo)"; // Default AI/Solo

    // PILLAR 3: Underworld Atmosphere cleanup.
    // Ensure the criminal-underworld class is only applied during active combat.
    const shouldShowUnderworld = state.phase === "Active" && (state.wanted_level || 0) >= 10;
    if (document.body.classList.contains("criminal-underworld") && !shouldShowUnderworld) {
        document.body.classList.remove("criminal-underworld");
    } else if (shouldShowUnderworld) {
        document.body.classList.add("criminal-underworld");
    }

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

export function renderCardHTML(card) {
    // PILLAR 5: String Pooling (Memoization).
    // PILLAR 4: Selection Feedback.
    // Selection is only relevant for the local player's cards (Inventory/Hand).
    const isSelected = activeCardId === card.id && (card.owner === -1 || card.owner === myPlayerIndex);

    // Generate a deterministic state key including selection status to prevent cache-ghosting.
    const scarsKey = (card.scars || []).join(',');
    
    // PILLAR 3: Factional Sync. Include local faction in state key.
    const combatState = window.GetGameState("combat");
    const myFaction = combatState?.faction || "NEUTRAL";
    const stateKey = `${card.id}-${card.owner}-${card.power.join('')}-${card.artifact}-${card.fatigue}-${card.loyalty}-${card.mood}-${card.image}-${isSelected}-${scarsKey}-${card.fallen}-${myFaction}`;
    if (cardHTMLPool.has(stateKey)) return cardHTMLPool.get(stateKey);

    const rarityBadge = (card.rarity && card.rarity > 1.0) ? `<div class="rarity-badge">${card.rarity.toFixed(1)}x</div>` : '';
    
    // PILLAR 4: Aspect-Ratio Compliance.
    // We use a dedicated layer for the artwork with 'cover' sizing to ensure external 
    // Solana/IPFS images fill the frame without distorting the neon-glass container. 
    // The background-image remains inline as it's dynamic, but other styles are now in _cards.scss.
    const artworkHTML = card.image ? `
        <div class="card-artwork-layer" style="background-image: url('${card.image}');"></div>
        <div class="card-glass-tint"></div>` : '';

    // PILLAR 6: Mutation Scar Overlay Loop.
    // Dynamically inject translucent overlays matching the card's procedure failure history.
    let scarsHTML = '';
    if (card.scars && Array.isArray(card.scars)) {
        card.scars.forEach(scar => {
            scarsHTML += `<div class="card-scar-overlay" style="background-image: url('./Assets/Images/Effects/${scar}.webp');"></div>`;
        });
    }

    // PILLAR 5: Hardware Item Overlay Layer.
    // Render trap/protection indicators for equipped items on cards.
    // NOTE: Backend must expose card.equipped_items array from club active buffs mapping TRAP_* IDs to card IDs.
    // Currently deferred pending Card struct enhancement in WASM GetGameState.
    let itemsHTML = '';
    if (card.equipped_items && Array.isArray(card.equipped_items)) {
        card.equipped_items.forEach(itemId => {
            // Map item ID to asset filename
            const itemMap = {
                'TRAP_BIO_GUARD_DOG': 'bio_guard_dog',
                'TRAP_LASER_TRIPWIRE': 'laser_tripwire',
                'TRAP_SENTRY_TURRET': 'sentry_turret'
            };
            const itemName = itemMap[itemId] || itemId.toLowerCase();
            const isActiveTrap = card.active_items && card.active_items.includes(itemId);
            itemsHTML += `<div class="item-overlay item-${itemName}${isActiveTrap ? ' item-pulse' : ''}" style="background-image: url('./Assets/Images/Items/${itemName}.webp');" title="Protected by ${itemName}"></div>`;
        });
    }

    // PILLAR 3: Factional Alignment Highlighting
    let factionBoostHTML = "";
    if (myFaction === "JUSTICE" && card.fallen) {
        factionBoostHTML = `<div class="faction-bonus-pulse animate-pulse" style="position:absolute; top:0; left:0; width:100%; height:100%; box-shadow: inset 0 0 20px var(--neon-cyan); pointer-events:none; border-radius: inherit;"></div>`;
    } else if (myFaction === "UNDERWORLD" && !card.fallen) {
        factionBoostHTML = `<div class="faction-bonus-pulse animate-pulse" style="position:absolute; top:0; left:0; width:100%; height:100%; box-shadow: inset 0 0 20px var(--neon-green); pointer-events:none; border-radius: inherit;"></div>`;
    }

    // PILLAR 7: Underworld Fallen Status.
    let fallenHTML = '';
    if (card.fallen) {
        fallenHTML = `<div class="fallen-badge" title="Fallen Asset: -50 Power Penalty">☣️</div>`;
    }

    // Mood Icon Mapping
    let moodHTML = '';
    if (card.mood && card.mood !== "Neutral" && MOOD_CLASS_MAP[card.mood]) {
        moodHTML = `<div class="card-type-icon ${MOOD_CLASS_MAP[card.mood]}" title="Mood: ${card.mood}">${MOOD_EMOJI_MAP[card.mood]}</div>`;
    } else if (card.mood && card.mood !== "Neutral") {
        moodHTML = `<div class="card-type-icon" title="Mood: ${card.mood}">✨</div>`;
    }

    // Artifact / Bonus Display
    let artifactHTML = '';
    if (card.artifact > 0) {
        artifactHTML = `<div class="artifact-badge" style="position: absolute; bottom: 30px; right: 5px; color: var(--neon-cyan); font-size: 9px; font-weight: bold; text-shadow: 0 0 5px var(--neon-cyan);">+${card.artifact}</div>`;
    } else if (card.artifact < 0) {
        artifactHTML = `<div class="debuff-badge" title="Battle Scar / Prisoner Penalty">PRISONER ${card.artifact}</div>`;
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

    // Cache global lookups for the power grid
    const getLabel = window.GetLevelLabelForDisplay || ((v) => "Z");

    const html = `
        ${artworkHTML} 
        ${scarsHTML}
        ${itemsHTML}
        <div class="card-content-wrapper ${isSelected ? 'selected-item' : ''} ${card.fallen ? 'fallen-asset' : ''}">
            ${factionBoostHTML}
            ${rarityBadge}
            ${fallenHTML}
            ${artifactHTML}
            ${moodHTML}
            <div class="power-grid" style="pointer-events: auto;">
                <div style="grid-area: top">${getLabel(card.power[0])}</div>
                <div style="grid-area: left">${getLabel(card.power[3])}</div>
                <div style="grid-area: right">${getLabel(card.power[1])}</div>
                <div style="grid-area: bottom">${getLabel(card.power[2])}</div>
            </div>
            ${statsHTML}
            <div class="card-name" style="pointer-events: none;">${card.name}</div>
        </div>
    `;

    // PILLAR 5: Cache management. Prune pool if session exceeds reasonable density bounds.
    if (cardHTMLPool.size > 250) cardHTMLPool.clear();
    cardHTMLPool.set(stateKey, html);
    return html;
}

export function movePowerTooltip(e) {
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

export function hidePowerTooltip() {
    if (tooltipEl) tooltipEl.style.opacity = "0";
}

/**
 * showMutationStabilityTooltip displays the breakdown of success chance modifiers.
 * PILLAR 6: Specialized Gene-Editing.
 */
export function showMutationStabilityTooltip(e, mojo, staffCount, hasInsurance, isSabotaged, isGovernor, isTrainingActive) {
    // Ensure tooltip container exists
    if (!tooltipEl) {
        tooltipEl = document.createElement("div");
        tooltipEl.id = "power-tooltip";
        tooltipEl.className = "power-tooltip";
        document.body.appendChild(tooltipEl);
    }

    const mojoBonus = Math.floor(Math.min(0.20, mojo / 5000) * 100);
    const staffBonus = Math.floor(Math.min(0.10, staffCount * 0.02) * 100);
    const base = 70;
    
    let html = `
        <div style="color: var(--neon-cyan); font-weight: bold; margin-bottom: 8px; border-bottom: 1px solid var(--neon-cyan); padding-bottom: 5px;">STABILITY ANALYSIS</div>
        <div class="tooltip-row" style="display: flex; justify-content: space-between; font-size: 0.85em; margin-bottom: 4px;">
            <span style="opacity: 0.7;">Base Rate:</span>
            <b>${base}%</b>
        </div>
        <div class="tooltip-row" style="display: flex; justify-content: space-between; font-size: 0.85em; margin-bottom: 4px;">
            <span style="opacity: 0.7;">Mojo Bonus:</span>
            <b class="text-neon-purple">+${mojoBonus}%</b>
        </div>
        <div class="tooltip-row" style="display: flex; justify-content: space-between; font-size: 0.85em; margin-bottom: 4px;">
            <span style="opacity: 0.7;">Staff Bonus:</span>
            <b class="text-neon-blue">+${staffBonus}%</b>
        </div>
        ${isGovernor ? `
        <div class="tooltip-row" style="display: flex; justify-content: space-between; font-size: 0.85em; margin-bottom: 4px;">
            <span style="opacity: 0.7;">Governor Bonus:</span>
            <b class="text-gold">+5%</b>
        </div>` : ''}
        ${isSabotaged ? `
        <div class="tooltip-row" style="display: flex; justify-content: space-between; font-size: 0.85em; margin-bottom: 4px; color: #ff4b4b;">
            <span>SABOTAGE PENALTY:</span>
            <b>-15%</b>
        </div>` : ''}
        ${isTrainingActive ? `
        <div class="tooltip-row" style="display: flex; justify-content: space-between; font-size: 0.85em; margin-bottom: 4px;">
            <span style="opacity: 0.7;">Staff Training:</span>
            <b class="text-neon-cyan">+5%</b>
        </div>` : ''}
    `;

    if (hasInsurance) {
        html += `
            <div style="margin-top: 10px; padding-top: 10px; border-top: 1px solid rgba(255,255,255,0.1); text-align: center; color: var(--neon-cyan); font-weight: bold; font-size: 0.8em;">
                🛡️ INSURANCE ACTIVE:<br>Result Guaranteed
            </div>`;
    } else {
        // PILLAR 6: Accurate Probability Reconstruction.
        // Aggregate all strategic modifiers to ensure UI parity with club_service.go
        let total = base + mojoBonus + staffBonus;
        if (isGovernor) total += 5;
        if (isTrainingActive) total += 5;
        if (isSabotaged) total -= 15;

        total = Math.min(98, Math.max(50, total));
        html += `
            <div style="margin-top: 10px; padding-top: 10px; border-top: 1px solid rgba(255,255,255,0.1); text-align: center; font-size: 0.9em;">
                ESTIMATED CHANCE: <b class="${total >= 90 ? 'text-neon-green' : 'text-warning'}">${total}%</b>
            </div>`;
    }

    tooltipEl.innerHTML = html;
    tooltipEl.style.opacity = "1";
    tooltipEl.style.pointerEvents = "none";
    movePowerTooltip(e);
}

export function showQuickCastMenu(gridIndex) {
    const container = document.querySelector(".tooltip-quickcast");
    if (!container) return;

    const state = window.GetGameState();
    // PILLAR 3: Redundancy Filtering. 
    // Filter inventory for items that aren't in the hand (deck), 
    // aren't already on the board, and aren't currently being cast.
    const deckIds = state.deck.map(c => c.id);
    const boardIds = state.board ? state.board.filter(c => c !== null).map(c => c.id) : [];
    const artifacts = state.inventory.filter(c => 
        !deckIds.includes(c.id) && 
        !boardIds.includes(c.id) && 
        c.id !== pendingQuickCastId && 
        c.artifact > 0
    );
    
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

export function handleLocalBanUI(banExpires) {
    const container = document.getElementById("local-ban-cooldown");
    const fill = document.getElementById("ban-progress-fill");
    const timer = document.getElementById("ban-countdown-timer");
    
    // PILLAR 5: Efficiency Guard.
    if (lastBanExpiry === banExpires && banTicker) return;
    lastBanExpiry = banExpires;

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

export function showMatchPreview(data) {
    // PILLAR 3: Factional Icon Identification.
    const p1 = lastLobbyPlayers.find(p => p.id === data.p1_id);
    const p2 = lastLobbyPlayers.find(p => p.id === data.p2_id);
    const p1Icon = p1?.faction === "JUSTICE" ? "⚖️ " : (p1?.faction === "UNDERWORLD" ? "💀 " : "");
    const p2Icon = p2?.faction === "JUSTICE" ? "⚖️ " : (p2?.faction === "UNDERWORLD" ? "💀 " : "");

    document.getElementById("preview-p1-id").innerText = p1Icon + data.p1_id;
    document.getElementById("preview-p1-rating").innerText = data.p1_rating || "[Z]";
    document.getElementById("preview-p2-id").innerText = p2Icon + data.p2_id;
    document.getElementById("preview-p2-rating").innerText = data.p2_rating || "[Z]";
    
    document.getElementById("match-preview-overlay").classList.remove("hidden");
}

// This function was previously in app.js, but is now moved to ui.js as it's purely UI-related.
export function shareTournamentVictory() {
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

export function openSettingsOverlay() {
    document.getElementById("settings-overlay").classList.remove("hidden");
}

export function closeSettingsOverlay() {
    document.getElementById("settings-overlay").classList.add("hidden");
}

/**
 * Generates the HTML structure for the tournament bracket.
 * PILLAR 4: Modular UI. Moved from legacy app.js to enforce UI authority.
 */
export function generateBracketHTML(matches, activeRound = -1) {
    if (!matches || matches.length === 0) {
        const msg = activeRound === -1 ? "Match data pending blockchain verification or unavailable." : "Matches will be generated once tournament starts...";
        return `<div style="color: #888; font-style: italic; padding: 10px; text-align: center; width: 100%;">${msg}</div>`;
    }

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
            const p1Obj = lastLobbyPlayers.find(p => p.wallet?.toLowerCase() === m.p1?.toLowerCase());
            const p2Obj = lastLobbyPlayers.find(p => p.wallet?.toLowerCase() === m.p2?.toLowerCase());
            const p1FactionIcon = p1Obj?.faction === "JUSTICE" ? "⚖️ " : (p1Obj?.faction === "UNDERWORLD" ? "💀 " : "");
            const p2FactionIcon = p2Obj?.faction === "JUSTICE" ? "⚖️ " : (p2Obj?.faction === "UNDERWORLD" ? "💀 " : "");

            const p1Short = getCachedEnvoiName(m.p1);
            const p2Short = getCachedEnvoiName(m.p2);
            let p1Class = "", p2Class = "";
            if (m.winner) {
                if (m.winner === m.p1) { p1Class = "winner"; p2Class = "loser"; }
                else if (m.winner === m.p2) { p2Class = "winner"; p1Class = "loser"; }
            }
            html += `
                <div id="match-${m.match_id}" class="bracket-match ${isCurrentRound && !m.winner ? 'active' : ''}">
                    <div class="bracket-player ${p1Class}">${p1FactionIcon}${p1Short}</div>
                    <div class="vs-label">VS</div>
                    <div class="bracket-player ${p2Class}">${p2FactionIcon}${p2Short}</div>
                </div>`;
        });
        html += `</div>`;
    });
    return html;
}

/**
 * Updates the tournament history pagination controls.
 */
export function updateTournamentPaginationUI() {
    const prevBtn = document.getElementById("prev-tournament-btn");
    const nextBtn = document.getElementById("next-tournament-btn");
    const info = document.getElementById("tournament-page-info");
    
    if (!prevBtn || !nextBtn || !info) return;

    const totalPages = Math.ceil(totalTournaments / tournamentLimit);
    info.innerText = `Page ${currentTournamentPage} of ${totalPages || 1}`;

    prevBtn.disabled = (currentTournamentPage <= 1);
    nextBtn.disabled = (currentTournamentPage >= totalPages || totalPages === 0);

    prevBtn.onclick = () => {
        fetchTournamentHistory(currentTournamentPage - 1);
        document.getElementById("hof-history-view").scrollTop = 0;
    };
    nextBtn.onclick = () => {
        fetchTournamentHistory(currentTournamentPage + 1);
        document.getElementById("hof-history-view").scrollTop = 0;
    };
}

export function startSeasonTimer() {
    // PILLAR 5: Efficiency Guard.
    const expiryTime = seasonEnd ? seasonEnd.getTime() : 0;
    if (lastSeasonExpiry === expiryTime && seasonTimerInterval) return;
    lastSeasonExpiry = expiryTime;

    if (seasonTimerInterval) clearInterval(seasonTimerInterval);
    const timerEl = document.getElementById("season-timer");
    if (!timerEl) return;

    const update = () => {
        if (!seasonEnd) return;
        const now = new Date();
        const diff = seasonEnd - now;
        if (diff <= 0) {
            timerEl.innerText = "SEASON ENDED - ROLLOVER IN PROGRESS";
            clearInterval(seasonTimerInterval);
            return;
        }
        const days = Math.floor(diff / (1000 * 60 * 60 * 24));
        const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
        const mins = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
        timerEl.innerText = `${days}d ${hours}h ${mins}m REMAINING`;
    };
    update();
    seasonTimerInterval = setInterval(update, 60000);
}

/**
 * renderHofEcosystemGuardianBanner highlights the vault milestone in the Hall of Fame.
 * PILLAR 1: Industrial Loop.
 */
export function renderHofEcosystemGuardianBanner() {
    const container = document.getElementById("hof-rankings-view");
    if (!container) return;

    const existing = document.getElementById("hof-ecosystem-guardian-banner");
    if (existing) existing.remove();

    const vaultStats = lastLobbyPlayers.find(p => p.wallet?.toLowerCase() === CONFIG.VAULT_ADDRESS?.toLowerCase());
    if (!vaultStats || !vaultStats.achievements || !vaultStats.achievements.includes("ECOSYSTEM_GUARDIAN")) return;

    const banner = document.createElement("div");
    banner.id = "hof-ecosystem-guardian-banner";
    banner.className = "glass-panel border-gold mb-20 p-20 animate-scale-in accelerated pulse";
    banner.style.cssText = "background: linear-gradient(135deg, rgba(212, 175, 55, 0.1), rgba(0, 0, 0, 0.8)); box-shadow: 0 0 30px rgba(212, 175, 55, 0.15); border-width: 2px; display: flex; flex-direction: row; align-items: center; gap: 20px; text-align: left;";
    
    banner.innerHTML = `
        <div style="font-size: 3.5em; filter: drop-shadow(0 0 15px gold);">🛡️</div>
        <div style="flex: 1;">
            <div class="text-gold font-bold font-size-1-1em letter-spacing-2 mb-5 uppercase" style="text-shadow: 0 0 10px rgba(212, 175, 55, 0.5);">House Milestone: Ecosystem Guardian</div>
            <p class="font-size-0-8em opacity-9 m-0 line-height-1-5">
                The House Vault has officially recovered <b class="text-gold">10,000 $VBV</b> via systemic protocol taxes. 
                This achievement marks a peak level of organizational solvency for the current sector.
            </p>
        </div>`;
    container.prepend(banner);
}

/**
 * renderDonationLeaderboard displays the top 5 contributors to the faucet.
 * PILLAR 1: Industrial Loop.
 */
export function renderDonationLeaderboard() {
    const container = document.getElementById("hof-rankings-view");
    if (!container) return;

    const existing = document.getElementById("donation-leaderboard-widget");
    if (existing) existing.remove();

    const topDonors = [...lastLobbyPlayers]
        .filter(p => (p.total_donated || 0) > 0)
        .sort((a, b) => b.total_donated - a.total_donated)
        .slice(0, 5);

    if (topDonors.length === 0) return;

    const widget = document.createElement("div");
    widget.id = "donation-leaderboard-widget";
    widget.className = "glass-panel border-gold mb-20 p-15 animate-scale-in accelerated";
    widget.style.cssText = "background: linear-gradient(135deg, rgba(212, 175, 55, 0.05), rgba(0, 0, 0, 0.6)); border-width: 1px; display: flex; flex-direction: column; gap: 10px;";

    widget.innerHTML = `
        <div class="flex-row align-center gap-10 mb-5">
            <span style="font-size: 1.5em;">🏛️</span>
            <div class="text-gold font-bold font-size-0-8em letter-spacing-2 uppercase" style="text-shadow: 0 0 10px rgba(212, 175, 55, 0.3);">Top Benevolent Contributors</div>
        </div>
        <div class="flex-col gap-8">
            ${topDonors.map((p, i) => `
                <div class="flex-row justify-between align-center font-size-0-9em">
                    <span class="flex-row align-center">
                        <span class="text-gold font-bold mr-10" style="min-width: 20px;">#${i+1}</span>
                        <span class="text-white">${p.faction === 'JUSTICE' ? '⚖️ ' : (p.faction === 'UNDERWORLD' ? '💀 ' : '')}${getCachedEnvoiName(p.wallet)}</span>
                    </span>
                    <b class="text-neon-green font-mono">${(p.total_donated / 1000000).toFixed(2)} $VBV</b>
                </div>
            `).join('')}
        </div>
        <div class="mt-5 pt-10 border-top-glass opacity-5 italic font-size-0-7em text-center">
            Generosity sustains the Global Faucet and builds personal Standing.
        </div>`;
    
    const guardianBanner = document.getElementById("hof-ecosystem-guardian-banner");
    if (guardianBanner) {
        guardianBanner.after(widget);
    } else {
        container.prepend(widget);
    }
}

export function switchHofTab(tab) {
    const views = ["hof-rankings-view", "hof-history-view", "hof-seasons-view"];
    views.forEach(v => document.getElementById(v).classList.add("hidden"));
    document.querySelectorAll(".hof-tab").forEach(t => t.classList.remove("active"));
    const target = document.getElementById(`hof-${tab}-view`);
    if (target) target.classList.remove("hidden");
    const activeTab = Array.from(document.querySelectorAll(".hof-tab")).find(t => t.onclick.toString().includes(tab));
    if (activeTab) activeTab.classList.add("active");

    if (tab === 'rankings') {
        renderHofEcosystemGuardianBanner();
        renderDonationLeaderboard();
    }
    if (tab === 'history') fetchTournamentHistory(1);
    if (tab === 'seasons') fetchSeasonHistory();
}

export function toggleTournamentDetails(id) {
    const details = document.getElementById(`details-${id}`);
    if (details) details.classList.toggle("hidden");
}

export function handleTournamentUI(tournamentState) {
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
            const assetId = network === "VOI" ? CONFIG.VBV_ASSET_ID : CONFIG.AVOI_ASSET_ID;
            if (CONFIG.VAULT_ADDRESS && assetId) {
                if (regBtn) regBtn.classList.remove("hidden");
            } else {
                statusText.innerText += " (Establishing Secure Sync...)";
                if (regBtn) regBtn.classList.add("hidden");
            }
        } else {
            statusText.innerText = `Tournament Active - Round ${tournamentState.current_round}`;
            if (regBtn) regBtn.classList.add("hidden");
        }
    }
}

export async function renderTournamentBracket(state) {
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

export function openTournamentBracket() {
    if (window.SetPhase) {
        window.SetPhase("TournamentLobby");
        window.syncUI();
    }
}

export function closeTournamentBracket() {
    if (window.SetPhase) {
        window.SetPhase("Lobby");
        window.syncUI();
    }
}

/**
 * Updates the spectator-specific HUD overlay.
 * Displays VBT Synergy (Arena Resonance) and Match metadata for immersive viewing.
 */
export function updateSpectatorHUD(state) {
    let hud = document.getElementById("spectator-hud");
    
    // Only show HUD if we are in an active match and spectating
    const isSpectating = spectatorMatchState !== null;
    
    if (!isSpectating || state.phase !== "Active") {
        if (hud) {
            hud.classList.add("hidden");
            hud.remove(); // PILLAR 5: DOM Integrity. Destroy the HUD when the spectating context is inactive.
        }
        return;
    }

    if (!hud) {
        hud = document.createElement("div");
        hud.id = "spectator-hud";
        hud.className = "spectator-hud glass-panel animate-fade-in";
        hud.style.cssText = "position: fixed; top: 100px; right: 20px; z-index: 100; pointer-events: none; padding: 15px; border-color: rgba(0, 242, 254, 0.4); min-width: 250px;";
        document.body.appendChild(hud);
    }
    hud.classList.remove("hidden");

    // Calculate VBT Synergy (Arena Resonance)
    // Logic: Base (100) + Buffs (15/ea) + Mood Alignments (25/ea)
    let synergy = 100;
    // PILLAR 5: Calculation Hardening.
    // Ensure that even if a player's buff map is null, the HUD remains stable.
    if (state.active_item_buffs) {
        Object.values(state.active_item_buffs).forEach(pb => {
            if (pb) synergy += Object.keys(pb).length * 15;
        });
    }
    if (state.board && state.board_moods) {
        state.board.forEach((c, i) => {
            if (c && c.mood === state.board_moods[i] && c.mood !== "Neutral") synergy += 25;
        });
    }

    // PILLAR 3: Standardized Identification.
    // Prioritize the match_id from the live spectator state.
    const matchID = (spectatorMatchState ? spectatorMatchState.match_id : state.match_id) || "ARENA-STND";
    const territory = (state.territory_id || "Arena Center").replace(/_/g, ' ').toUpperCase();
    const rulesCount = Object.values(state.rules || {}).filter(v => v).length;

    // PILLAR 5: Reactive Atmosphere.
    // Trigger a high-intensity 'LIVE' pulse when VBT Synergy reaches PEAK levels (>150).
    const isPeak = synergy > 150;

    hud.innerHTML = `
        <div style="border-bottom: 1px solid rgba(0, 242, 254, 0.3); padding-bottom: 8px; margin-bottom: 10px; display: flex; justify-content: space-between; align-items: center;">
            <span style="font-size: 0.6em; color: ${isPeak ? 'var(--error-red)' : 'var(--neon-cyan)'}; letter-spacing: 2px; font-weight: bold;" class="${isPeak ? 'animate-pulse' : ''}">
                ${isPeak ? '🔴 ' : ''}LIVE BROADCAST</span>
            <span style="font-size: 0.7em; opacity: 0.8; font-family: monospace;">#${matchID.substring(0, 10)}</span>
        </div>
        <div class="flex-row gap-20" style="justify-content: space-between;">
            <div style="text-align: center;"><small style="display: block; font-size: 0.6em; opacity: 0.5;">VBT SYNERGY</small><b class="text-neon-green" style="font-size: 1.3em;">${synergy}</b></div>
            <div style="text-align: center;"><small style="display: block; font-size: 0.6em; opacity: 0.5;">LOCATION</small><b class="text-neon-cyan" style="font-size: 0.9em; letter-spacing: 1px;">${territory}</b></div>
            <div style="text-align: center;"><small style="display: block; font-size: 0.6em; opacity: 0.5;">RULES</small><b class="text-neon-purple" style="font-size: 0.9em;">${rulesCount} ACTIVE</b></div>
        </div>
        ${(userAddress && spectatorMatchState.p1_wallet?.toLowerCase() !== userAddress.toLowerCase() && spectatorMatchState.p2_wallet?.toLowerCase() !== userAddress.toLowerCase()) ? `
            <div style="margin-top: 12px; pointer-events: auto;">
                <button class="w-full bg-neon-green text-dark font-bold btn-small" 
                        style="box-shadow: 0 0 10px var(--neon-green); border-radius: 4px;"
                        onclick="window.openSpectatorWagerOverlay('${matchID}', '${spectatorMatchState.p1_wallet}', '${spectatorMatchState.p2_wallet}')">
                    PLACE WAGER
                </button>
            </div>
        ` : ''}
        <div style="margin-top: 12px; font-size: 0.7em; text-align: center; color: #888; font-style: italic; border-top: 1px solid rgba(255,255,255,0.05); pt-5">
            RESONANCE: ${synergy > 150 ? 'PEAK' : synergy > 120 ? 'STABLE' : 'SYNCING...'}
        </div>`;
}

/**
 * updateEnforcementHUD renders status indicators for License, Insurance, and Bonds.
 * PILLAR 1 & 3: Justice vs Underworld Hegemony.
 */
function updateEnforcementHUD(state) {
    const container = getEl("enforcement-hud-container");
    if (!container) return;

    const hasLicense = state.bounty_hunter_license_expires_at && new Date(state.bounty_hunter_license_expires_at) > Date.now();
    const hasInsurance = (state.raid_insurance_claims_remaining || 0) > 0 && new Date(state.raid_insurance_expires_at) > Date.now();
    const hasBond = (state.bounty_hunter_bond_micro || 0) > 0;

    // PILLAR 2: Siphon Alerts.
    if (state.last_siphon_amt > 0) {
        showToast(`💸 <b>PROTOCOL SIPHON:</b> ${(state.last_siphon_amt / 1000000).toFixed(2)} $VBV diverted to Faucet.`, "info");
    }

    if (!hasLicense && !hasInsurance && !hasBond) {
        container.classList.add("hidden");
        return;
    }

    let html = `<div class="flex-row gap-10">`;
    if (hasLicense) {
        html += `<div class="glass-panel p-5-10 border-neon-cyan flex-row align-center gap-5" style="height: 32px;" title="ENFORCEMENT LICENSE ACTIVE">
                    <span class="text-neon-cyan font-bold font-size-0-7em">⚖️ LICENSED</span>
                 </div>`;
    }
    if (hasBond) {
        html += `<div class="glass-panel p-5-10 border-gold flex-row align-center gap-5" style="height: 32px;" title="SECURITY BOND DEPOSITED">
                    <span class="text-gold font-bold font-size-0-7em">🛡️ BONDED</span>
                 </div>`;
    }
    if (hasInsurance) {
        html += `<div class="glass-panel p-5-10 border-neon-green flex-row align-center gap-5" style="height: 32px;" title="RAID INSURANCE ACTIVE">
                    <span class="text-neon-green font-bold font-size-0-7em">🛡️ INSURED</span>
                 </div>`;
    }
    html += `</div>`;

    container.innerHTML = html;
    container.classList.remove("hidden");

    // PILLAR 3: Social Hub Sync.
    const banner = document.getElementById("license-status-banner");
    if (banner) {
        const expiry = state.bounty_hunter_license_expires_at ? new Date(state.bounty_hunter_license_expires_at) : null;
        const valid = expiry && expiry > Date.now();
        banner.className = `glass-panel p-10 m-0 mb-15 flex-row justify-between align-center ${valid ? 'border-neon-cyan' : 'expired'}`;
        const statusEl = banner.querySelector("b");
        if (statusEl) {
            statusEl.className = valid ? 'text-neon-green' : 'text-error';
            statusEl.innerText = valid ? 'STATUS: VALID' : 'STATUS: EXPIRED / NONE';
        }
    }
}