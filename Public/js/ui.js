// Public/js/ui.js

import { CONFIG } from './config.js';
import { myClientId, currentLatency, lastPingTime, setLastPingTime, setCurrentLatency } from './network.js';
import { userAddress } from './wallet.js'; // userAddress is now in wallet.js
import { myPlayerIndex, currentOpponentId, spectatorMatchState, lastTauntPhase, lastTauntTurn, setLastTauntPhase, setLastTauntTurn, matchHistorySaved, setMatchHistorySaved, saveMatchResult, renderChatMessage, reportGloat, lastLobbyPlayers } from './game.js';
import { masterVolume, musicVolume, sfxVolume } from './audio.js';
import { updateAdminRewardList, fetchAdminLogs, adminLogTicker, startAdminLogPolling, stopAdminLogPolling } from './admin.js';
import { updateActiveRumors, renderRumorBoard } from './criminality.js';
import { seasonEnd, totalTournaments, tournamentLimit, currentTournamentPage, fetchTournamentHistory, fetchSeasonHistory } from './leaderboard.js';
import { getAssetSymbol, getCachedEnvoiName, resolveEnvoiName, assetCache, resolveAssetSymbol } from './utils.js';
import { globalClubs, availableNetworks } from './admin.js';
import { buyClubItem, submitClubFoundry, tradeShares, buyBlackMarketItem, submitConsignment, takeLease } from './economy.js';
import { initiateBail, deployTrap, payRansom, releaseHostage, spreadRumor } from './criminality.js';

export let tooltipEl = document.getElementById("power-tooltip");
export let maintenanceTicker = null;

// PERFORMANCE OPTIMIZATION: Move static maps outside the render loop to prevent re-allocation
const MOOD_CLASS_MAP = { "Volatile": "fire", "Serene": "water", "Spirited": "lightning", "Grounded": "earth" };
const MOOD_EMOJI_MAP = { "Volatile": "🔥", "Serene": "💧", "Spirited": "⚡", "Grounded": "🌿" };

const TERRITORY_MAP = [
    { id: "the_lab", name: "The Lab", icon: "🧪" },
    { id: "north_district", name: "North Gate", icon: "⛩️" },
    { id: "the_archive", name: "The Archive", icon: "📜" },
    { id: "west_port", name: "West Port", icon: "⚓" },
    { id: "arena_center", name: "Arena Center", icon: "⚔️" },
    { id: "east_gate", name: "East Gate", icon: "🏯" },
    { id: "south_slums", name: "The Slums", icon: "🏚️" },
    { id: "casino", name: "The Casino", icon: "🎰" },
    { id: "data_haven", name: "Data Haven", icon: "💾" }
];

export let mapZoom = 1.0;

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

export function openClubFoundry() {
    const available = TERRITORY_MAP.filter(t => !Object.values(globalClubs).find(c => c.territories && c.territories.includes(t.id)));
    const overlay = document.createElement("div");
    overlay.id = "club-foundry-overlay";
    overlay.className = "overlay";
    overlay.innerHTML = `
        <div class="glass-panel medium" style="text-align: center;">
            <h2 class="text-neon-purple">CLUB FOUNDRY</h2>
            <p style="font-size: 0.9em; opacity: 0.8;">Founding a club costs a fortune (5,000 $VBV).<br>Owners earn commissions from relative buffs sold in their territory.</p>
            <div class="flex-col gap-10 mt-20">
                <input type="text" id="foundry-club-name" class="glass-input w-full" placeholder="Enter Club Name (max 20 chars)" maxlength="20">
                <select id="foundry-shop-type" class="glass-input w-full" aria-label="Select Shop Specialization" title="Shop Type">
                    <option value="Elemental">Elemental Forge (Mood Buffs)</option>
                    <option value="Tactical">Tactical Syndicate (Rule Mastery)</option>
                    <option value="Vitality">Vitality Lab (Health/Loyalty)</option>
                </select>
                <select id="foundry-territory" class="glass-input w-full" ${available.length === 0 ? 'disabled' : ''} aria-label="Select territory to claim" title="District Selection">
                    ${available.length > 0 ? available.map(t => `<option value="${t.id}">${t.name}</option>`).join('') : '<option value="">NO DISTRICTS AVAILABLE</option>'}
                </select>
            </div>
            <div class="mt-20 flex-row justify-center gap-15">
                <button class="outline" onclick="document.getElementById('club-foundry-overlay').remove()">CANCEL</button>
                <button id="foundry-submit-btn" onclick="submitClubFoundry()">FOUND CLUB (5,000 $VBV)</button>
            </div>
        </div>`;
    document.body.appendChild(overlay);
}

export function adjustMapZoom(delta) {
    mapZoom += delta;
    if (mapZoom < 0.5) mapZoom = 0.5;
    if (mapZoom > 2.0) mapZoom = 2.0;
    const grid = document.getElementById("map-3d-grid");
    if (grid) grid.style.transform = `rotateX(30deg) rotateY(-15deg) scale(${mapZoom})`;
}

export function openTerritoryMapOverlay() {
    const grid = document.getElementById("map-3d-grid");
    if (!grid) return;
    mapZoom = 1.0;
    grid.style.transform = `rotateX(30deg) rotateY(-15deg) scale(${mapZoom})`;
    grid.innerHTML = "";
    TERRITORY_MAP.forEach(t => {
        const club = Object.values(globalClubs).find(c => c.territories && c.territories.includes(t.id));
        const isOwned = !!club;
        const isGovernor = isOwned && club.region_name;
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
            <div class="tile-status ${isUnderAttack ? 'under-attack' : isGovernor ? 'developing' : ''}"></div>`;
        grid.appendChild(tile);
    });
    document.getElementById("territory-map-overlay").classList.remove("hidden");
}

export function openTerritoryView(territoryId) {
    const club = Object.values(globalClubs).find(c => c.territory === territoryId);
    const overlay = document.createElement("div");
    overlay.id = "territory-view-overlay";
    overlay.className = "overlay";
    let body = `<p style="opacity: 0.7;">This territory is currently unclaimed. Found a Club to take control!</p>`;
    if (club) {
        const items = { "Elemental": [{ id: "mood_catalyst", name: "Mood Catalyst", price: 100, desc: "+50 Mood Bonus" }], "Tactical": [{ id: "rule_breaker", name: "Rule Breaker", price: 150, desc: "Force PLUS trigger" }], "Vitality": [{ id: "stamina_stim", name: "Stamina Stim", price: 100, desc: "-20 Fatigue" }] }[club.type] || [];
        body = `<div class="flex-col gap-10">${items.map(i => `
            <div class="shop-item-row glass-panel p-15 m-0 flex-row justify-between align-center animate-shimmer">
                <div class="text-left"><b>${i.name}</b><div class="font-size-0-8em opacity-6">${i.desc}</div></div>
                <button class="outline" onclick="buyClubItem('${club.id}', '${i.id}', ${i.price}, '${territoryId}')">${i.price} $VBV</button>
            </div>`).join('')}</div>`;
    }
    overlay.innerHTML = `<div class="glass-panel medium" style="text-align: center;"><h2>TERRITORY: ${territoryId.replace('_',' ').toUpperCase()}</h2>${body}
        <div class="mt-20"><button class="outline" onclick="document.getElementById('territory-view-overlay').remove()">CLOSE</button>${!club ? `<button onclick="document.getElementById('territory-view-overlay').remove(); openClubFoundry()">FOUND CLUB</button>` : ''}</div></div>`;
    document.body.appendChild(overlay);
}

export async function openPortfolioView(initialTab = 'portfolio') {
    const state = window.GetGameState();
    const overlay = document.createElement("div");
    overlay.id = "portfolio-view-overlay";
    overlay.className = "overlay";
    overlay.innerHTML = `
        <div class="glass-panel medium" style="text-align: center;">
            <h2 class="text-neon-cyan">ENTITY PORTFOLIO</h2>
            <div class="flex-row justify-center gap-10 mt-10 mb-20">
                <button id="tab-holdings" class="tab-btn" onclick="switchPortfolioTab('portfolio')">📈 HOLDINGS</button>
                <button id="tab-jailed" class="tab-btn" onclick="switchPortfolioTab('jailed')">⛓️ JAILED</button>
                <button id="tab-kidnapped" class="tab-btn" onclick="switchPortfolioTab('kidnapped')">😈 KIDNAPPED</button>
                <button id="tab-hostage" class="tab-btn" onclick="switchPortfolioTab('hostage')">🛑 HOSTAGE</button>
            </div>
            <div id="portfolio-content-area" class="flex-col gap-10" style="max-height: 400px; overflow-y: auto;"></div>
            <button class="outline mt-20 w-full" onclick="document.getElementById('portfolio-view-overlay').remove()">CLOSE</button>
        </div>`;
    document.body.appendChild(overlay);
    switchPortfolioTab(initialTab);
}

export async function switchPortfolioTab(tab) {
    const container = document.getElementById("portfolio-content-area");
    const state = window.GetGameState();
    document.querySelectorAll('#portfolio-view-overlay .tab-btn').forEach(b => b.classList.toggle('active', b.id.includes(tab)));
    container.innerHTML = `<div class="opacity-5 py-20">Decrypting assets...</div>`;

    if (tab === 'portfolio') {
        const entries = Object.entries(state.portfolio || {});
        if (entries.length === 0) { container.innerHTML = `<div class="py-40 opacity-5">No active investments found.</div>`; return; }
        await Promise.all(entries.map(([w]) => resolveEnvoiName(w)));
        container.innerHTML = `<div class="portfolio-view">${entries.map(([id, amt]) => `
            <div class="portfolio-item glass-panel m-0 mb-10">
                <div class="text-left"><div class="item-name text-neon-cyan">${getCachedEnvoiName(id)}</div><div class="opacity-5 font-size-0-75em">Shares: ${amt.toFixed(2)}</div></div>
                <button class="outline x-small border-error" onclick="tradeShares('${id}', 'sell', ${amt})">SELL ALL</button>
            </div>`).join('')}</div>`;
    } else if (tab === 'jailed') {
        const jailed = Object.keys(state.jailed_cards || {});
        if (jailed.length === 0) { container.innerHTML = `<div class="py-40 opacity-5">No cards in custody.</div>`; return; }
        container.innerHTML = jailed.map(id => `
            <div class="player-item" style="border-color: #ff4b4b;">
                <div class="text-left"><b class="text-error">ID: #${id}</b></div>
                <button class="outline x-small border-green" onclick="initiateBail(${id}, '${state.jailed_cards[id]}')">PAY BAIL</button>
            </div>`).join('');
    } else if (tab === 'hostage') {
        const hostage = Object.keys(state.held_hostage_cards || {});
        if (hostage.length === 0) { container.innerHTML = `<div class="py-40 opacity-5">No cards held hostage.</div>`; return; }
        container.innerHTML = hostage.map(id => `
            <div class="player-item" style="border-color: #ffd700;">
                <div class="text-left"><b class="text-gold">ID: #${id}</b></div>
                <button class="outline x-small border-error" onclick="payRansom(${id}, '${state.held_hostage_cards[id]}')">PAY RANSOM</button>
            </div>`).join('');
    }
}

export function openSecuritySentry() {
    const state = window.GetGameState();
    const club = globalClubs[state.employer_id];
    if (!club) return;
    const overlay = document.createElement("div");
    overlay.id = "security-sentry-overlay";
    overlay.className = "overlay";
    overlay.innerHTML = `
        <div class="glass-panel" style="width: 550px; text-align: center;">
            <h2 class="text-neon-cyan">🛡️ SECURITY SENTRY: ${club.name.toUpperCase()}</h2>
            <div class="flex-col gap-10 mt-20">
                ${[{id:"tripwire", name:"Tripwire"}, {id:"sentry_turret", name:"Turret"}, {id:"guard_dog", name:"Guard Dog"}].map(t => `
                    <div class="glass-panel p-10 m-0 flex-row justify-between align-center">
                        <b>${t.name}</b>
                        <button class="outline" ${state.inventory[t.id] > 0 ? '' : 'disabled'} onclick="deployTrap('${t.id}')">DEPLOY</button>
                    </div>`).join('')}
            </div>
            <button class="outline mt-20 w-full" onclick="document.getElementById('security-sentry-overlay').remove()">CLOSE</button>
        </div>`;
    document.body.appendChild(overlay);
}

export async function openBountyBoard() {
    const outlaws = lastLobbyPlayers.filter(p => (p.wanted_level || 0) >= 10);
    const overlay = document.createElement("div");
    overlay.id = "bounty-board-overlay";
    overlay.className = "overlay";
    if (outlaws.length > 0) await Promise.all(outlaws.map(p => resolveEnvoiName(p.wallet)));
    overlay.innerHTML = `
        <div class="glass-panel" style="width: 500px; text-align: center; border-color: #ffd700;">
            <h2 style="color: #ffd700;">🎯 BOUNTY BOARD</h2>
            <div class="flex-col gap-10 mt-20">${outlaws.length === 0 ? '<div class="opacity-5 py-20">No active bounties.</div>' : outlaws.map(p => `
                <div class="player-item" style="border-color: #ffd700;">
                    <div class="text-left"><b>${getCachedEnvoiName(p.wallet)}</b><br><small>Wanted: ${p.wanted_level}</small></div>
                    <button class="outline x-small" onclick="document.getElementById('bounty-board-overlay').remove(); window.sendChallenge('${p.id}')">HUNT</button>
                </div>`).join('')}</div>
            <button class="outline mt-20 w-full" onclick="document.getElementById('bounty-board-overlay').remove()">CLOSE</button>
        </div>`;
    document.body.appendChild(overlay);
}

export async function openBlackMarket() {
    const state = window.GetGameState();
    const overlay = document.createElement("div");
    overlay.id = "black-market-overlay";
    overlay.className = "overlay";
    overlay.innerHTML = `
        <div class="economy-panel black-market" style="width: 650px;">
            <div class="market-header"><span class="market-title">THE UNDERWORLD</span></div>
            <div id="black-market-grid" class="market-grid p-20">Loading hot items...</div>
            <button class="outline mt-20" onclick="document.getElementById('black-market-overlay').remove()">CLOSE</button>
        </div>`;
    document.body.appendChild(overlay);
    try {
        const res = await fetch(`${CONFIG.API_BASE}/api/black-market?wallet=${userAddress}`);
        const items = await res.json();
        const grid = document.getElementById("black-market-grid");
        if (items.length === 0) { grid.innerHTML = `<div class="opacity-5">No hot items currently available.</div>`; return; }
        grid.innerHTML = items.map(i => `
            <div class="player-item" style="border-color: #ff4b4b;">
                <div class="text-left"><b>Hot Bundle: CARD-#${i.collateral_bundle.card_id}</b></div>
                <button class="outline x-small border-error" onclick="buyBlackMarketItem('${i.id}', 100)">BUY (RISKY)</button>
            </div>`).join('');
    } catch (e) { document.getElementById("black-market-grid").innerHTML = `<div class="text-error">Uplink Failed.</div>`; }
}

export function openArtGalleryOverlay() {
    const overlay = document.createElement("div");
    overlay.id = "art-gallery-overlay";
    overlay.className = "overlay";
    overlay.innerHTML = `
        <div class="economy-panel gallery-panel" style="width: 900px;">
            <div class="economy-header"><span class="economy-title">THE ART GALLERY</span>
                <button class="outline x-small border-error" onclick="document.getElementById('art-gallery-overlay').remove()">CLOSE</button>
            </div>
            <div id="gallery-items-container" class="gallery-grid p-20"></div>
        </div>`;
    document.body.appendChild(overlay);
    loadGalleryItems();
}

export async function loadGalleryItems() {
    const container = document.getElementById("gallery-items-container");
    try {
        const res = await fetch(`${CONFIG.API_BASE}/api/auctions`);
        const auctions = await res.json();
        if (!auctions || auctions.length === 0) { container.innerHTML = `<div class="opacity-5 py-40">Gallery floor is vacant.</div>`; return; }
        container.innerHTML = auctions.map(a => `
            <div class="gallery-grid__item-bundle glass-panel">
                <div class="item-title font-bold text-neon-cyan">${a.bundle.weapon_id || 'Tactical Artifact'}</div>
                <button class="outline mt-15 w-full border-cyan" onclick="promptBid('${a.id}', ${a.current_bid})">PLACE BID</button>
            </div>`).join('');
    } catch (e) { container.innerHTML = `<div class="text-error">Gallery Indexer Offline.</div>`; }
}

export function openConsignmentOverlay() {
    const state = window.GetGameState();
    const overlay = document.createElement("div");
    overlay.id = "consignment-overlay";
    overlay.className = "overlay";
    overlay.innerHTML = `
        <div class="economy-panel consignment-panel" style="width: 550px;">
            <div class="market-header"><span class="market-title">ASSET CONSIGNMENT</span></div>
            <div class="p-20"><p class="opacity-6">Select an asset from your collection to list.</p>
                <button class="outline mt-20 w-full" onclick="document.getElementById('consignment-overlay').remove()">ABORT</button>
            </div>
        </div>`;
    document.body.appendChild(overlay);
}

export function openRumorMill() {
    const overlay = document.createElement("div");
    overlay.id = "rumor-mill-overlay";
    overlay.className = "overlay";
    overlay.innerHTML = `
        <div class="glass-panel" style="width: 600px; text-align: center;">
            <h2 class="text-neon-green">RUMOR MILL</h2>
            <div id="rumor-targets" class="flex-col gap-10 mt-20"></div>
            <button class="outline mt-20 w-full" onclick="document.getElementById('rumor-mill-overlay').remove()">CLOSE</button>
        </div>`;
    document.body.appendChild(overlay);
    const otherPlayers = lastLobbyPlayers.filter(p => p.id !== myClientId);
    document.getElementById("rumor-targets").innerHTML = otherPlayers.map(p => `
        <div class="player-item">
            <div class="text-left"><b>${p.id}</b></div>
            <button class="outline x-small" onclick="spreadRumor('${p.wallet}', 'positive', 1.1, 60)">SPREAD</button>
        </div>`).join('');
}

export function openSocialPanelOverlay(initialTab = 'alliances') {
    const state = window.GetGameState();
    const overlay = document.createElement("div");
    overlay.id = "social-hub-overlay";
    overlay.className = "overlay";
    overlay.innerHTML = `
        <div class="social-panel glass-panel" style="width: 750px;">
            <div class="social-header"><span class="social-title">NEON SOCIAL HUB</span></div>
            <div class="flex-row justify-center gap-10 mt-10 mb-20">
                <button id="social-tab-alliances" class="tab-btn" onclick="switchSocialTab('alliances')">🤝 ALLIANCES</button>
                <button id="social-tab-career" class="tab-btn" onclick="switchSocialTab('career')">💼 CAREER</button>
            </div>
            <div id="social-content-hub" class="flex-col gap-15"></div>
            <button class="outline mt-20 w-full" onclick="document.getElementById('social-hub-overlay').remove()">CLOSE</button>
        </div>`;
    document.body.appendChild(overlay);
    switchSocialTab(initialTab);
}

export function switchSocialTab(tab) {
    const container = document.getElementById("social-content-hub");
    document.querySelectorAll('#social-hub-overlay .tab-btn').forEach(b => b.classList.toggle('active', b.id.includes(tab)));
    container.innerHTML = `<div class="opacity-5 py-40">Loading social hub...</div>`;
    if (tab === 'career') {
        container.innerHTML = `<div class="career-system"><div class="career-title">PATH: FREELANCER</div></div>`;
    }
}

export function openClubLeaseBoard() {
    const overlay = document.createElement("div");
    overlay.id = "lease-board-overlay";
    overlay.className = "overlay";
    overlay.innerHTML = `
        <div class="glass-panel" style="width: 700px; text-align: center;">
            <h2 class="text-neon-purple">INDUSTRIAL LEASE BOARD</h2>
            <div id="lease-list-container" class="flex-col gap-10 mt-20"></div>
            <button class="outline mt-20 w-full" onclick="document.getElementById('lease-board-overlay').remove()">CLOSE</button>
        </div>`;
    document.body.appendChild(overlay);
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

/**
 * Updates the spectator-specific HUD overlay.
 * Displays VBT Synergy (Arena Resonance) and Match metadata for immersive viewing.
 */
export function updateSpectatorHUD(state) {
    let hud = document.getElementById("spectator-hud");
    
    // Only show HUD if we are in an active match and spectating
    const isSpectating = spectatorMatchState !== null;
    
    if (!isSpectating || state.phase !== "Active") {
        if (hud) hud.classList.add("hidden");
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
    if (state.active_item_buffs) {
        Object.values(state.active_item_buffs).forEach(pb => synergy += Object.keys(pb).length * 15);
    }
    if (state.board && state.board_moods) {
        state.board.forEach((c, i) => {
            if (c && c.mood === state.board_moods[i] && c.mood !== "Neutral") synergy += 25;
        });
    }

    const matchID = state.tournament_match_id || "ARENA-STND";
    const territory = (state.territory_id || "Arena Center").replace(/_/g, ' ').toUpperCase();
    const rulesCount = Object.values(state.rules || {}).filter(v => v).length;

    hud.innerHTML = `
        <div style="border-bottom: 1px solid rgba(0, 242, 254, 0.3); padding-bottom: 8px; margin-bottom: 10px; display: flex; justify-content: space-between; align-items: center;">
            <span style="font-size: 0.6em; color: var(--neon-cyan); letter-spacing: 2px; font-weight: bold;">LIVE BROADCAST</span>
            <span style="font-size: 0.7em; opacity: 0.8; font-family: monospace;">#${matchID.substring(0, 10)}</span>
        </div>
        <div class="flex-row gap-20" style="justify-content: space-between;">
            <div style="text-align: center;"><small style="display: block; font-size: 0.6em; opacity: 0.5;">VBT SYNERGY</small><b class="text-neon-green" style="font-size: 1.3em;">${synergy}</b></div>
            <div style="text-align: center;"><small style="display: block; font-size: 0.6em; opacity: 0.5;">LOCATION</small><b class="text-neon-cyan" style="font-size: 0.9em; letter-spacing: 1px;">${territory}</b></div>
            <div style="text-align: center;"><small style="display: block; font-size: 0.6em; opacity: 0.5;">RULES</small><b class="text-neon-purple" style="font-size: 0.9em;">${rulesCount} ACTIVE</b></div>
        </div>
        <div style="margin-top: 12px; font-size: 0.7em; text-align: center; color: #888; font-style: italic; border-top: 1px solid rgba(255,255,255,0.05); pt-5">
            RESONANCE: ${synergy > 150 ? 'PEAK' : synergy > 120 ? 'STABLE' : 'SYNCING...'}
        </div>`;
}