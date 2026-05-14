import { CONFIG } from './config.js';
import { socket, myClientId } from './network.js';
import { showToast, showTournamentTransition, tooltipEl } from './ui.js';
import { userAddress, walletProvider, signClient } from './wallet.js';
import { getCachedEnvoiName, resolveEnvoiName, getNetworkConfig } from './utils.js';

const algosdk = window.algosdk; // Assuming algosdk is globally available
export let totalTournaments = 0;
export let lastTournamentData = null;
export let seasonEnd = null;
export let currentTournamentPage = 1;
export const tournamentLimit = 5;

// --- Setters for external modules ---
export const setSeasonEnd = (date) => { seasonEnd = date; };

// --- Leaderboard Functions ---
export async function fetchLeaderboard() {
    const leaderboardList = document.getElementById("leaderboard-list");
    if (!leaderboardList) return;
    leaderboardList.innerHTML = `<div class="chat-msg system">Fetching top players...</div>`;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/leaderboard`);
        const players = await response.json();
        
        if (players.length === 0) {
            leaderboardList.innerHTML = `<div class="chat-msg system">Arena is fresh. No legends yet.</div>`;
            return;
        }

        // Resolve names for top 10
        await Promise.all(players.slice(0, 10).map(p => resolveEnvoiName(p.wallet)));

        leaderboardList.innerHTML = players.map((p, i) => `
            <div class="leaderboard-row ${p.id === myClientId ? 'me' : ''}">
                <span class="rank-badge">#${i + 1}</span>
                <span class="player-name">${getCachedEnvoiName(p.wallet)}</span>
                <span class="player-stats">${p.wins}W | ${p.reputation} REP</span>
            </div>
        `).join('');
    } catch (err) {
        leaderboardList.innerHTML = `<div class="chat-msg system error">Leaderboard uplink offline.</div>`;
    }
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
    seasonTimerInterval = setInterval(update, 60000); // Check once per minute
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

export async function fetchTournamentHistory(page = 1) {
    currentTournamentPage = page;
    const container = document.getElementById("tournament-history-list");
    if (!container) return;
    container.innerHTML = `<div class="opacity-5 py-20">Fetching archives...</div>`;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/tournament/history?page=${page}&limit=${tournamentLimit}`);
        const data = await response.json();
        
        totalTournaments = data.total;
        
        if (data.history.length === 0) {
            container.innerHTML = `<div class="opacity-3 py-40 italic">No tournament data found in this sector.</div>`;
        } else {
            container.innerHTML = data.history.map(t => `
                <div class="tournament-item glass-panel">
                    <div class="flex-row justify-between align-center mb-10">
                        <b class="text-neon-cyan">TOURNAMENT #${t.id.substring(0, 8)}</b>
                        <small class="opacity-5">${new Date(t.timestamp).toLocaleDateString()}</small>
                    </div>
                    <div class="flex-row justify-between">
                        <span>Winner: <b class="text-gold">${getCachedEnvoiName(t.winner)}</b></span>
                        <span>Pot: <b class="text-neon-green">${t.pot} $VBV</b></span>
                    </div>
                    <button class="outline x-small mt-10" onclick="toggleTournamentDetails('${t.id}')">VIEW BRACKET</button>
                    <div id="details-${t.id}" class="hidden mt-10 pt-10 border-top-glass">
                        ${window.generateBracketHTML(t.matches)}
                    </div>
                </div>
            `).join('');
        }
        window.updateTournamentPaginationUI();
    } catch (err) {
        container.innerHTML = `<div class="text-error">Archive Uplink Failed.</div>`;
    }
}

export async function fetchSeasonHistory() {
    const container = document.getElementById("season-history-list");
    if (!container) return;
    container.innerHTML = `<div class="opacity-5 py-20">Accessing seasonal records...</div>`;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/season/history`);
        const seasons = await response.json();
        
        container.innerHTML = seasons.map(s => `
            <div class="season-item glass-panel">
                <div class="flex-row justify-between align-center mb-10">
                    <b class="text-neon-purple">SEASON ${s.season}</b>
                    <small class="opacity-5">${new Date(s.start).toLocaleDateString()} - ${new Date(s.end).toLocaleDateString()}</small>
                </div>
                <div class="season-winners-list">
                    ${s.top.map((p, i) => `
                        <div class="season-winner-row flex-row justify-between">
                            <span>#${i+1} ${getCachedEnvoiName(p.w)}</span>
                            <b class="text-neon-green">${p.v} Wins</b>
                        </div>
                    `).join('')}
                </div>
            </div>
        `).join('');
    } catch (err) {
        container.innerHTML = `<div class="text-error">Season Archive Offline.</div>`;
    }
}

export function toggleTournamentDetails(id) {
    const details = document.getElementById(`details-${id}`);
    if (!details) return;
    details.classList.toggle("hidden");
}

export function openTournamentBracket() {
    window.SetPhase("TournamentLobby");
    window.syncUI();
}

export function closeTournamentBracket() {
    window.SetPhase("Lobby");
    window.syncUI();
}

window.generateBracketHTML = (matches, activeRound = -1) => {
    if (!matches || matches.length === 0) return `<div class="opacity-5 italic text-center p-10">Generating bracket...</div>`;

    const rounds = {};
    matches.forEach(m => {
        if (!rounds[m.round]) rounds[m.round] = [];
        rounds[m.round].push(m);
    });

    return Object.keys(rounds).sort((a, b) => a - b).map(r => `
        <div class="bracket-round">
            <div class="bracket-round-title">ROUND ${r}</div>
            ${rounds[r].map(m => {
                const p1Short = getCachedEnvoiName(m.p1);
                const p2Short = getCachedEnvoiName(m.p2);
                const isWinnerP1 = m.winner === m.p1;
                const isWinnerP2 = m.winner === m.p2;
                
                return `
                    <div class="bracket-match ${(activeRound == r && !m.winner) ? 'active' : ''}">
                        <div class="bracket-player ${isWinnerP1 ? 'winner' : m.winner ? 'loser' : ''}">${p1Short}</div>
                        <div class="vs-label">VS</div>
                        <div class="bracket-player ${isWinnerP2 ? 'winner' : m.winner ? 'loser' : ''}">${p2Short}</div>
                    </div>
                `;
            }).join('')}
        </div>
    `).join('');
};

window.updateTournamentPaginationUI = () => {
    const prevBtn = document.getElementById("prev-tournament-btn");
    const nextBtn = document.getElementById("next-tournament-btn");
    const info = document.getElementById("tournament-page-info");
    if (!prevBtn || !nextBtn || !info) return;

    const totalPages = Math.ceil(totalTournaments / tournamentLimit);
    info.innerText = `Page ${currentTournamentPage} of ${totalPages || 1}`;
    prevBtn.disabled = currentTournamentPage <= 1;
    nextBtn.disabled = currentTournamentPage >= totalPages || totalPages === 0;
};

window.switchHofTab = switchHofTab;
window.toggleTournamentDetails = toggleTournamentDetails;
window.fetchTournamentHistory = fetchTournamentHistory;
