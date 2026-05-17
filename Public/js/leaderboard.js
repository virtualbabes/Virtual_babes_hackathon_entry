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
                        ${window.GetTournamentArchiveBadge(t.is_verified, t.links || [], t.receipts_verified, t.payouts_hash || "")}
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
        
        // Resolve names for all unique wallets in top standings and highlights
        const wallets = new Set();
        seasons.forEach(s => {
            s.top.forEach(p => wallets.add(p.w));
            if (s.highlights) s.highlights.forEach(h => wallets.add(h.w));
        });
        await Promise.all(Array.from(wallets).map(w => resolveEnvoiName(w)));

        const highlightIcons = {
            "Tournament Champion": "🏆",
            "Master Collector": "🎨",
            "Social Titan": "🔥"
        };

        container.innerHTML = seasons.map(s => {
            const highlightsHTML = s.highlights && s.highlights.length > 0 ? `
                <div class="season-highlights mb-20">
                    <div class="highlight-label font-size-0-7em opacity-5 mb-10 letter-spacing-1">HALL OF VALOR</div>
                    <div class="flex-col gap-10">
                        ${s.highlights.map(h => `
                            <div class="highlight-row glass-panel m-0 p-10 flex-row align-center gap-15 border-gold" style="background: rgba(255, 215, 0, 0.05);">
                                <div class="highlight-icon font-size-1-5em">${highlightIcons[h.a] || '⭐'}</div>
                                <div class="text-left flex-1">
                                    <div class="highlight-title font-bold text-gold" style="font-size: 0.9em; letter-spacing: 1px;">${h.a.toUpperCase()}</div>
                                    <div class="highlight-player font-size-0-8em opacity-9">
                                        <b class="text-neon-cyan">${getCachedEnvoiName(h.w)}</b> 
                                        <span class="opacity-5" style="margin-left: 5px;">— ${h.m}</span>
                                    </div>
                                </div>
                            </div>
                        `).join('')}
                    </div>
                </div>` : '';

            return `
                <div class="season-item glass-panel">
                    <div class="flex-row justify-between align-center mb-15">
                        <b class="text-neon-purple" style="font-size: 1.1em;">SEASON ${s.season}</b>
                        <small class="opacity-5">${new Date(s.start).toLocaleDateString()} - ${new Date(s.end).toLocaleDateString()}</small>
                    </div>
                    
                    ${highlightsHTML}

                    <div class="season-winners-list">
                        <div class="highlight-label font-size-0-7em opacity-5 mb-5 letter-spacing-1">TOP STANDINGS</div>
                        ${s.top.map((p, i) => `
                            <div class="season-winner-row flex-row justify-between align-center p-5">
                                <span><span class="rank-badge mr-10" style="min-width: 25px;">#${i+1}</span> ${getCachedEnvoiName(p.w)}</span>
                                <b class="text-neon-green">${p.v} Wins</b>
                            </div>
                        `).join('')}
                    </div>
                </div>
            `;
        }).join('');
    } catch (err) {
        container.innerHTML = `<div class="text-error">Season Archive Offline.</div>`;
    }
}
