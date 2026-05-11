import { CONFIG } from './config.js';
import { socket } from './network.js';
import { showToast, showTournamentTransition, tooltipEl } from './ui.js';
import { userAddress, walletProvider, signClient } from './wallet.js';
import { getCachedEnvoiName, resolveEnvoiName, getNetworkConfig } from './utils.js';

const algosdk = window.algosdk; // Assuming algosdk is globally available
export let totalTournaments = 0;
export let lastTournamentData = null;
export let seasonEnd = null;

// --- Setters for external modules ---
export const setSeasonEnd = (date) => { seasonEnd = date; };
// --- Leaderboard Functions ---

export async function fetchLeaderboard() {
    const leaderboardList = document.getElementById("leaderboard-list"); // Get the leaderboard list element
    if (!leaderboardList) return;
    leaderboardList.innerHTML = `<div class="chat-msg system">Fetching top players...</div>`;

    }
}

export let seasonTimerInterval = null; // Moved here from app.js
export function startSeasonTimer() { // Exported for use in network.js
    if (seasonTimerInterval) clearInterval(seasonTimerInterval);
    const timerEl = document.getElementById("season-timer");
    
    
    update();
    seasonTimerInterval = setInterval(update, 60000); // Check once per minute
}

export function switchHofTab(tab) {
    document.getElementById("hof-rankings-view").classList.add("hidden");
    }
}

export function toggleTournamentDetails(id) { // Exported for use in app.js
    const details = document.getElementById(`details-${id}`);
    if (!details) return;
    details.classList.toggle("hidden");
    }
}

export async function registerForTournament() { // Exported for use in app.js
    const regBtn = document.getElementById("tournament-reg-btn");
    if (!userAddress) { showToast("Connect wallet first", "error"); return; }
    const state = window.GetGameState();
    }
}

export function openTournamentBracket() { // Exported for use in app.js
    window.SetPhase("TournamentLobby");
    window.syncUI();
}

export function closeTournamentBracket() { // Exported for use in app.js
    window.SetPhase("Lobby");
    window.syncUI();
}
