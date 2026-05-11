// Public/js/ui.js

import { CONFIG } from './config.js';
import { myClientId, currentLatency, lastPingTime, setLastPingTime, setCurrentLatency } from './network.js';
import { userAddress } from './wallet.js'; // userAddress is now in wallet.js
import { myPlayerIndex, currentOpponentId, spectatorMatchState, lastTauntPhase, lastTauntTurn, setLastTauntPhase, setLastTauntTurn, matchHistorySaved, setMatchHistorySaved, saveMatchResult, renderChatMessage, reportGloat, lastLobbyPlayers } from './game.js';
import { masterVolume, musicVolume, sfxVolume } from './audio.js';
import { updateAdminRewardList, fetchAdminLogs, adminLogTicker, startAdminLogPolling, stopAdminLogPolling } from './admin.js';
import { updateActiveRumors, renderRumorBoard } from './criminality.js';
import { startSeasonTimer, seasonEnd, setSeasonEnd } from './leaderboard.js';
import { getAssetSymbol } from './utils.js';

export let tooltipEl = null;
export let maintenanceTicker = null;

// PERFORMANCE OPTIMIZATION: Move static maps outside the render loop to prevent re-allocation
const MOOD_CLASS_MAP = { "Volatile": "fire", "Serene": "water", "Spirited": "lightning", "Grounded": "earth" };
const MOOD_EMOJI_MAP = { "Volatile": "🔥", "Serene": "💧", "Spirited": "⚡", "Grounded": "🌿" };

// --- Transaction Feedback (Toast) ---
export function showToast(message, type = 'info', duration = 5000) {
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
        }, 500); // Allow transition to complete before removing
    }
}

// Global function to manage transaction status display
export function setTransactionStatus(message, type = 'info') {
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

export function handleMaintenanceUI(active, targetTimestamp) {
    const bar = document.getElementById("maintenance-bar");
    const timerDisplay = document.getElementById("maintenance-timer");

    if (maintenanceTicker) clearInterval(maintenanceTicker);

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
    const rarityBadge = (card.rarity && card.rarity > 1.0) ? `<div class="rarity-badge">${card.rarity.toFixed(1)}x</div>` : '';
    
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

    // Cache global lookups for the power grid
    const getLabel = window.GetLevelLabelForDisplay || ((v) => "Z");

    return `
        ${rarityBadge}
        ${artifactHTML}
        ${moodHTML}
        <div class="power-grid">
            <div style="grid-area: top">${getLabel(card.power[0])}</div>
            <div style="grid-area: left">${getLabel(card.power[3])}</div>
            <div style="grid-area: right">${getLabel(card.power[1])}</div>
            <div style="grid-area: bottom">${getLabel(card.power[2])}</div>
        </div>
        ${statsHTML}
        <div class="card-name">${card.name}</div>
    `;
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

export function showQuickCastMenu(gridIndex) {
    const container = document.querySelector(".tooltip-quickcast");
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

export function handleLocalBanUI(banExpires) {
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

export function showMatchPreview(data) {
    document.getElementById("preview-p1-id").innerText = data.p1_id;
    document.getElementById("preview-p1-rating").innerText = data.p1_rating || "[Z]";
    document.getElementById("preview-p2-id").innerText = data.p2_id;
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