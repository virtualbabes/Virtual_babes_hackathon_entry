// Public/js/ui.js

import { CONFIG } from './config.js';
import { myClientId, currentLatency, lastPingTime, setLastPingTime, setCurrentLatency } from './network.js';
import { myPlayerIndex, userAddress, currentOpponentId, spectatorMatchState, lastTauntPhase, lastTauntTurn, setLastTauntPhase, setLastTauntTurn, matchHistorySaved, setMatchHistorySaved, saveMatchResult, renderChatMessage, reportGloat } from '../app.js';
import { masterVolume, musicVolume, sfxVolume } from './audio.js';
import { updateAdminRewardList, fetchAdminLogs, adminLogTicker, startAdminLogPolling, stopAdminLogPolling } from './admin.js';
import { updateActiveRumors, renderRumorBoard } from './criminality.js';
import { startSeasonTimer, seasonEnd, setSeasonEnd } from './leaderboard.js';
import { getAssetSymbol } from './utils.js';

export let tooltipEl = null;
export let maintenanceTicker = null;

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