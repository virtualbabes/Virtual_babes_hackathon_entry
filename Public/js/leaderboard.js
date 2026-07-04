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
let cachedSeasons = []; // Cache for filtering
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

        leaderboardList.innerHTML = players.map((p, i) => {
            const isVault = p.wallet?.toLowerCase() === CONFIG.VAULT_ADDRESS?.toLowerCase();
            const rowClass = isVault ? 'vault-highlight' : (p.id === myClientId ? 'me' : '');
            
            // PILLAR 1: House Highlighting.
            const houseTag = isVault ? `<span class="tag-gold font-xs px-5 py-2 mr-5" style="border: 1px solid; border-radius: 4px; vertical-align: middle; background: rgba(212, 175, 55, 0.1);">HOUSE</span>` : '';
            const vaultStyle = isVault ? `style="border: 1px solid #ffd700; box-shadow: 0 0 15px rgba(212, 175, 55, 0.4); background: linear-gradient(90deg, rgba(212, 175, 55, 0.15), transparent);"` : '';

            // PILLAR 1: Donation Tier Badges.
            const donated = p.total_donated || 0;
            const donatedVBV = donated / 1000000;
            let donationTag = "";
            if (donatedVBV >= 500) {
                let color = "#cd7f32"; // Bronze
                let label = "BENEFACTOR";
                if (donatedVBV >= 5000) { color = "#b9f2ff"; label = "PATRON"; }
                else if (donatedVBV >= 2500) { color = "#ffd700"; label = "GUARDIAN"; }
                else if (donatedVBV >= 1000) { color = "#c0c0c0"; label = "SUPPORTER"; }
                
                donationTag = `<span class="font-xs px-5 py-2 mr-5" style="border: 1px solid ${color}; color: ${color}; border-radius: 4px; vertical-align: middle; background: ${color}1A;" title="Total Donated: ${donatedVBV.toFixed(0)} VBV">${label}</span>`;
            }

            // PILLAR 1: Hardened Status Badge.
            const reparations = p.reparations_received_count || 0;
            const hardenedTag = reparations >= 5 ? `<span class="font-xs px-5 py-2 mr-5" style="border: 1px solid var(--neon-cyan); color: var(--neon-cyan); border-radius: 4px; vertical-align: middle; background: rgba(0, 242, 254, 0.1);" title="Hardened: Secured ${reparations} reparations this session.">🛡️ HARDENED</span>` : '';

            return `
                <div class="leaderboard-row ${rowClass} accelerated" ${vaultStyle}>
                    <span class="rank-badge">#${i + 1}</span>
                    <span class="player-name">${houseTag}${donationTag}${hardenedTag}${getCachedEnvoiName(p.wallet)}</span>
                    <span class="player-stats">${p.wins}W | ${p.reputation} REP</span>
                </div>
            `;
        }).join('');
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
            // PILLAR 4: Historical Immersion.
            // Batch resolve Envoi names for all winners and participants in the history
            // to ensure high-fidelity handles are displayed in the archived brackets.
            const wallets = new Set();
            data.history.forEach(t => {
                if (t.winner) wallets.add(t.winner);
                if (t.matches) {
                    t.matches.forEach(m => {
                        if (m.p1) wallets.add(m.p1);
                        if (m.p2) wallets.add(m.p2);
                    });
                }
            });
            await Promise.all(Array.from(wallets).filter(w => w && w.length > 50).map(w => resolveEnvoiName(w)));

            container.innerHTML = data.history.map(t => `
                <div class="tournament-item glass-panel">
                    <div class="flex-row justify-between align-center mb-10">
                        <b class="text-neon-cyan">TOURNAMENT #${t.id.substring(0, 8)}</b>
                        <small class="opacity-5">${new Date(t.timestamp).toLocaleDateString()}</small>
                        ${window.GetTournamentArchiveBadge(t.is_verified, t.links || [], t.receipts_verified, t.payouts_hash || "")}
                    </div>
                    <div class="flex-row justify-between">
                        <span>Winner: <b class="text-gold">${getCachedEnvoiName(t.winner)}</b></span>
                        <span>Pot: <b class="text-neon-green">${(t.pot_micro / 1000000).toFixed(2)} $VBV</b></span>
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
        
        cachedSeasons = seasons; // Cache the fetched data
        // Resolve names for all unique wallets in top standings and highlights
        const wallets = new Set();
        seasons.forEach(s => {
            s.top.forEach(p => wallets.add(p.w));
            if (s.highlights) s.highlights.forEach(h => wallets.add(h.w));
        });
        await Promise.all(Array.from(wallets).map(w => resolveEnvoiName(w)));

        renderSeasonHistory(seasons);
    } catch (err) {
        container.innerHTML = `<div class="text-error">Season Archive Offline.</div>`;
    }
}

/**
 * Renders the season history from a given array of seasons.
 * This function is now used by both fetchSeasonHistory and filterSeasonHistory.
 */
function renderSeasonHistory(seasonsToRender) {
    const container = document.getElementById("season-history-list");
    if (!container) return;

    if (!seasonsToRender || seasonsToRender.length === 0) {
        container.innerHTML = `<div class="opacity-3 py-40 italic">No season data found.</div>`;
        return;
    }

    const highlightIcons = {
        "Tournament Champion": "🏆",
        "Master Collector": "🎨",
        "Social Titan": "🔥"
    };

    container.innerHTML = seasonsToRender.map(s => {
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
}

export function filterSeasonHistory() {
    const filterInput = document.getElementById("season-filter-input");
    if (!filterInput) return;

    const filterValue = parseInt(filterInput.value);

    let seasonsToRender = cachedSeasons;
    if (!isNaN(filterValue) && filterValue > 0) {
        seasonsToRender = cachedSeasons.filter(s => s.season === filterValue);
    }

    renderSeasonHistory(seasonsToRender);
}
