// Public/js/ui.js

import { CONFIG } from './config.js';
import { myClientId, currentLatency, lastPingTime, setLastPingTime, setCurrentLatency } from './network.js';
import { userAddress } from './wallet.js'; // userAddress is now in wallet.js
import { myPlayerIndex, currentOpponentId, spectatorMatchState, lastTauntPhase, lastTauntTurn, setLastTauntPhase, setLastTauntTurn, matchHistorySaved, setMatchHistorySaved, saveMatchResult, renderChatMessage, reportGloat, lastLobbyPlayers } from './game.js';
import { masterVolume, musicVolume, sfxVolume } from './audio.js';
import { updateAdminRewardList, fetchAdminLogs, adminLogTicker, startAdminLogPolling, stopAdminLogPolling } from './admin.js';
import { updateActiveRumors, renderRumorBoard } from './criminality.js';
import { seasonEnd, totalTournaments, tournamentLimit, currentTournamentPage, fetchTournamentHistory, fetchSeasonHistory } from './leaderboard.js';
import { getAssetSymbol, getCachedEnvoiName, resolveEnvoiName } from './utils.js';

export let tooltipEl = document.getElementById("power-tooltip");
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
        const colorMap = {
            'error': '#ff4b4b',
            'critical': '#ff4b4b', // Critical messages will use the error red color
            'success': 'var(--neon-green)',
            'info': 'var(--neon-cyan)',
            'warning': '#ffd700' // Warning messages will use a gold/yellow color
        };
        statusEl.innerHTML = `<span style="color: ${colorMap[type] || 'white'};">${message}</span>`;
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
            if (Math.random() > 0.85) {
                if (window.triggerMoodMote) window.triggerMoodMote(idx, mood);
                if (window.playMoodMoteSFX) window.playMoodMoteSFX(mood);
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
            const p1Short = getCachedEnvoiName(m.p1);
            const p2Short = getCachedEnvoiName(m.p2);
            let p1Class = "", p2Class = "";
            if (m.winner) {
                if (m.winner === m.p1) { p1Class = "winner"; p2Class = "loser"; }
                else if (m.winner === m.p2) { p2Class = "winner"; p1Class = "loser"; }
            }
            html += `
                <div class="bracket-match ${isCurrentRound && !m.winner ? 'active' : ''}">
                    <div class="bracket-player ${p1Class}">${p1Short}</div>
                    <div class="vs-label">VS</div>
                    <div class="bracket-player ${p2Class}">${p2Short}</div>
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

export let seasonTimerInterval = null;
export function startSeasonTimer() {
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

export function switchHofTab(tab) {
    const views = ["hof-rankings-view", "hof-history-view", "hof-seasons-view"];
    views.forEach(v => document.getElementById(v).classList.add("hidden"));
    document.querySelectorAll(".hof-tab").forEach(t => t.classList.remove("active"));
    const target = document.getElementById(`hof-${tab}-view`);
    if (target) target.classList.remove("hidden");
    const activeTab = Array.from(document.querySelectorAll(".hof-tab")).find(t => t.onclick.toString().includes(tab));
    if (activeTab) activeTab.classList.add("active");

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