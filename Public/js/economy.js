import { CONFIG } from './config.js';
import { socket, myClientId } from './network.js';
import { showToast, hideAllOverlays, renderCardHTML } from './ui.js';
import { userAddress, walletProvider, signClient } from './wallet.js';
import { getCachedEnvoiName, getNetworkConfig, resolveEnvoiName, resolveAssetSymbol, getAssetSymbol } from './utils.js';
import { globalClubs, availableNetworks } from './admin.js';
import { lastLobbyPlayers } from './game.js';
import { collectiveIntelligence } from '../collective-intelligence.js';

const algosdk = window.algosdk; // Assuming algosdk is globally available

export let tickerItems = [];
export let tickerOffset = 0;
export let tickerAnimId = null;
export let mapZoom = 1.0;

export function updateMarketTicker(players) {
    const spacing = 60;
    let tickerContainer = document.getElementById("market-ticker");
    if (!tickerContainer) {
        tickerContainer = document.createElement("div");
        tickerContainer.id = "market-ticker";
        tickerContainer.className = "market-ticker-container";
        tickerContainer.innerHTML = `
            <div class="ticker-label">LIVE MARKET:</div>
            <canvas id="market-ticker-canvas" style="flex: 1; height: 30px; cursor: default;"></canvas>
        `;
        document.body.prepend(tickerContainer);

        const canvas = document.getElementById("market-ticker-canvas");
        const resize = () => {
            const dpr = window.devicePixelRatio || 1;
            const rect = canvas.getBoundingClientRect();
            canvas.width = rect.width * dpr;
            canvas.height = 30 * dpr;
            const ctx = canvas.getContext('2d');
            ctx.scale(dpr, dpr);
        };
        window.addEventListener('resize', resize);
        resize();
    }

    const topPerformers = [...players]
        .sort((a, b) => (b.wins - a.wins) || (b.reputation - a.reputation))
        .slice(0, 5);

    const newItems = [];
    newItems.push({ symbol: "MKT TOKEN", val: "0.80 $VBV", trend: "▲", color: "#3fb950" });

    topPerformers.forEach(p => {
        const basePrice = (p.wins * 10) + (p.reputation / 2) + 100;
        const finalPrice = basePrice + (p.id.charCodeAt(p.id.length - 1) % 5);
        newItems.push({
            symbol: getCachedEnvoiName(p.wallet),
            badge: (p.achievements && p.achievements.length > 0) ? "🏆" : "",
            val: finalPrice.toFixed(2),
            trend: (p.wins > 0) ? "▲" : "─",
            color: (p.wins > 0) ? "#3fb950" : "#888",
            isNPC: collectiveIntelligence.personalities[p.id] !== undefined || p.id === "Vbabe Bot"
        });
    });

    const canvas = document.getElementById("market-ticker-canvas");
    const ctx = canvas ? canvas.getContext('2d') : null;
    if (ctx) {
        tickerItems = newItems.map(item => {
            ctx.font = item.isNPC ? "italic bold 12px 'Rajdhani', sans-serif" : "bold 12px 'Rajdhani', sans-serif";
            const str = `${item.symbol}${item.badge ? ' ' + item.badge : ''} ${item.val} ${item.trend}`;
            item.width = ctx.measureText(str).width + spacing;
            return item;
        });
    }

    if (!tickerAnimId) startTickerAnimation();
}

export function startTickerAnimation() {
    const canvas = document.getElementById("market-ticker-canvas");
    if (!canvas) return;
    const ctx = canvas.getContext('2d');

    const animate = () => {
        if (tickerItems.length === 0) { tickerAnimId = requestAnimationFrame(animate); return; }
        const width = canvas.width / (window.devicePixelRatio || 1);
        const height = 30;
        ctx.clearRect(0, 0, width, height);
        ctx.textBaseline = "middle";

        const totalContentWidth = tickerItems.reduce((sum, item) => sum + (item.width || 0), 0);
        if (totalContentWidth <= 0) { tickerAnimId = requestAnimationFrame(animate); return; }

        tickerOffset += 0.8;
        if (tickerOffset >= totalContentWidth) tickerOffset = 0;

        let x = -tickerOffset;
        while (x < width) {
            tickerItems.forEach(item => {
                const itemWidth = item.width || 100;
                if (x + itemWidth > 0 && x < width) {
                    ctx.font = item.isNPC ? "italic bold 12px 'Rajdhani', sans-serif" : "bold 12px 'Rajdhani', sans-serif";
                    ctx.fillStyle = item.isNPC ? "#9b51e0" : "#00f2fe";
                    ctx.fillText(item.symbol, x, height / 2);
                    let curX = x + ctx.measureText(item.symbol).width;
                    if (item.badge) { ctx.fillStyle = "#ffd700"; ctx.fillText(" " + item.badge, curX, height / 2); curX += ctx.measureText(" " + item.badge).width; }
                    ctx.font = "bold 12px 'Rajdhani', sans-serif";
                    ctx.fillStyle = "#ffffff";
                    ctx.fillText(" " + item.val, curX, height / 2);
                    curX += ctx.measureText(" " + item.val).width;
                    ctx.fillStyle = item.color;
                    ctx.fillText(" " + item.trend, curX, height / 2);
                }
                x += itemWidth;
            });
        }
        tickerAnimId = requestAnimationFrame(animate);
    };
    tickerAnimId = requestAnimationFrame(animate);
}

export function adjustMapZoom(delta) {
    mapZoom = Math.min(2, Math.max(0.5, mapZoom + delta));
    const grid = document.getElementById("map-3d-grid");
    if (grid) grid.style.transform = `rotateX(30deg) rotateY(-15deg) scale(${mapZoom})`;
}

export function openTerritoryMapOverlay() {
    const grid = document.getElementById("map-3d-grid");
    if (!grid) return;
    mapZoom = 1.0;
    grid.style.transform = `rotateX(30deg) rotateY(-15deg) scale(${mapZoom})`;
    grid.innerHTML = "";
    const territoryMap = [
        { id: "the_lab", name: "The Lab", icon: "🧪" }, { id: "north_district", name: "North Gate", icon: "⛩️" },
        { id: "the_archive", name: "The Archive", icon: "📜" }, { id: "west_port", name: "West Port", icon: "⚓" },
        { id: "arena_center", name: "Arena Center", icon: "⚔️" }, { id: "east_gate", name: "East Gate", icon: "🏯" },
        { id: "south_slums", name: "The Slums", icon: "🏚️" }, { id: "casino", name: "The Casino", icon: "🎰" },
        { id: "data_haven", name: "Data Haven", icon: "💾" }
    ];

    territoryMap.forEach(t => {
        const club = Object.values(globalClubs).find(c => c.territories && c.territories.includes(t.id));
        const isOwned = !!club;
        const isUnderAttack = isOwned && club.last_heist_at && (Date.now() - new Date(club.last_heist_at).getTime()) < 300000;
        const tile = document.createElement("div");
        tile.className = `map-tile-3d accelerated ${isOwned ? (club.region_name ? 'governor-controlled' : 'controlled') : 'neutral'}`;
        tile.onclick = () => { hideAllOverlays(); openTerritoryView(t.id); };
        tile.innerHTML = `
            <div class="tile-label">
                <div class="tile-icon">${t.icon}</div>
                <div class="tile-name">${t.name.toUpperCase()}</div>
                <div class="tile-owner">${isOwned ? club.name : 'NEUTRAL ZONE'}</div>
            </div>
            <div class="tile-status ${isUnderAttack ? 'under-attack' : (isOwned && club.region_name ? 'developing' : '')}"></div>
        `;
        grid.appendChild(tile);
    });
    document.getElementById("territory-map-overlay").classList.remove("hidden");
}

export function openClubFoundry() {
    const claimed = new Set();
    Object.values(globalClubs).forEach(c => c.territories?.forEach(t => claimed.add(t)));
    const territoryCatalog = [
        { id: "the_lab", name: "The Lab" }, { id: "north_district", name: "North Gate" },
        { id: "the_archive", name: "The Archive" }, { id: "west_port", name: "West Port" },
        { id: "arena_center", name: "Arena Center" }, { id: "east_gate", name: "East Gate" },
        { id: "south_slums", name: "The Slums" }, { id: "casino", name: "The Casino" },
        { id: "data_haven", name: "Data Haven" }
    ];
    const available = territoryCatalog.filter(t => !claimed.has(t.id));

    const overlay = document.createElement("div");
    overlay.id = "club-foundry-overlay";
    overlay.className = "overlay";
    overlay.innerHTML = `
        <div class="glass-panel" style="width: 450px; text-align: center;">
            <h2 style="color: var(--neon-purple);">CLUB FOUNDRY</h2>
            <div class="flex-col gap-10 mt-20">
                <input type="text" id="foundry-club-name" class="glass-input w-full" placeholder="Enter Club Name" maxlength="20">
                <select id="foundry-shop-type" class="glass-input w-full"><option value="Elemental">Elemental Forge</option><option value="Tactical">Tactical Syndicate</option><option value="Vitality">Vitality Lab</option></select>
                <select id="foundry-territory" class="glass-input w-full" ${available.length === 0 ? 'disabled' : ''}>
                    ${available.length > 0 ? available.map(t => `<option value="${t.id}">${t.name}</option>`).join('') : '<option value="">NO DISTRICTS AVAILABLE</option>'}
                </select>
            </div>
            <div class="mt-20 flex-row justify-center gap-15">
                <button class="outline" onclick="document.getElementById('club-foundry-overlay').remove()">CANCEL</button>
                <button id="foundry-submit-btn" onclick="submitClubFoundry()">FOUND CLUB (5,000 $VBV)</button>
            </div>
        </div>
    `;
    document.body.appendChild(overlay);
}

export async function submitClubFoundry() {
    const name = document.getElementById("foundry-club-name").value.trim();
    const type = document.getElementById("foundry-shop-type").value;
    const territory = document.getElementById("foundry-territory").value;
    if (!name || !userAddress) return;

    try {
        const state = window.GetGameState();
        const amountMicro = 5000 * 1000000;
        let txid = "SIM_TX_" + Date.now(); // Replace with actual signing logic
        socket.send(JSON.stringify({ type: "create_club", payload: { name, type, territory_id: territory, txid, network: state.network } }));
        document.getElementById("club-foundry-overlay").remove();
    } catch (err) { showToast(`Founding Failed: ${err.message}`, "error"); }
}

export function openShopsOverlay(category = 'Elemental') {
    document.getElementById("shops-overlay").classList.remove("hidden");
    switchShopCategory(category);
}

export function switchShopCategory(category) {
    const container = document.getElementById("shops-container");
    if (!container) return;
    document.querySelectorAll('.category-tab').forEach(tab => tab.classList.toggle('active', tab.dataset.category === category));
    const clubs = Object.values(globalClubs).filter(c => c.type === category);
    container.innerHTML = clubs.length ? clubs.map(club => Object.entries(club.inventory || {}).map(([itemId, qty]) => `
        <div class="shop-item animate-slide-up" onclick="buyClubItem('${club.id}', '${itemId}', 100, '${club.territories[0]}')">
            <div class="item-info"><div class="item-title">${itemId.toUpperCase()}</div><div class="item-value">STOCK: ${qty}</div></div>
            <button class="buy-button">PURCHASE</button>
        </div>`).join('')).join('') : '<div class="opacity-5 py-40 italic">Sector dry for assets.</div>';
}

export async function openPortfolioView(initialTab = 'portfolio') {
    document.getElementById("portfolio-view-overlay")?.classList.remove("hidden");
    switchPortfolioTab(initialTab);
}

export function switchPortfolioTab(tab) {
    const container = document.getElementById("portfolio-content-area");
    if (!container) return;
    const state = window.GetGameState();
    if (tab === 'portfolio') {
        const entries = Object.entries(state.portfolio || {});
        container.innerHTML = entries.length ? entries.map(([id, amt]) => `<div class="portfolio-item glass-panel"><b>${id.substring(0,8)}</b>: ${amt.toFixed(2)} Shares</div>`).join('') : "No holdings.";
    } else {
        container.innerHTML = "Accessing encrypted records...";
    }
}

export function tradeShares(entityId, action, amount) {
    if (!socket || socket.readyState !== WebSocket.OPEN) return;
    socket.send(JSON.stringify({ type: "trade_shares", payload: { entity_id: entityId, action, amount } }));
}

export function openClubLeaseBoard() {
    document.getElementById("lease-board-overlay")?.classList.remove("hidden");
}

export function takeLease(clubId, leaseId) {
    socket.send(JSON.stringify({ type: "take_lease", payload: { club_id: clubId, lease_id: leaseId } }));
    document.getElementById("lease-board-overlay")?.remove();
}
    const spacing = 60;
    let tickerContainer = document.getElementById("market-ticker");
    if (!tickerContainer) {
        tickerContainer = document.createElement("div");
        tickerContainer.id = "market-ticker";
        tickerContainer.className = "market-ticker-container";
        tickerContainer.innerHTML = `<div class="ticker-label">LIVE MARKET:</div><canvas id="market-ticker-canvas" style="flex: 1; height: 30px;"></canvas>`;
        document.body.prepend(tickerContainer);
    }
    // Implementation uses canvas drawing for performance (hoisted from app.js)
    // ...
}

export function openTerritoryMapOverlay() {
    document.getElementById("territory-map-overlay").classList.remove("hidden");
}

export function adjustMapZoom(delta) {
    const grid = document.getElementById("map-3d-grid");
    if (grid) {
        let zoom = parseFloat(grid.dataset.zoom || 1.0) + delta;
        zoom = Math.min(2, Math.max(0.5, zoom));
        grid.dataset.zoom = zoom;
        grid.style.transform = `rotateX(30deg) rotateY(-15deg) scale(${zoom})`;
    }
}

export function openClubLeaseBoard() {
    document.getElementById("lease-board-overlay")?.classList.remove("hidden");
}

export function takeLease(clubId, leaseId) {
    socket.send(JSON.stringify({ type: "take_lease", payload: { club_id: clubId, lease_id: leaseId } }));
}

export function openShopsOverlay(category = 'Elemental') {
    document.getElementById("shops-overlay").classList.remove("hidden");
    switchShopCategory(category);
}

export async function openPortfolioView(initialTab = 'portfolio') {
    document.getElementById("portfolio-view-overlay")?.classList.remove("hidden");
    switchPortfolioTab(initialTab);
}

export function switchPortfolioTab(tab) {
    const container = document.getElementById("portfolio-content-area");
    if (!container) return;
    container.innerHTML = "Refreshing data...";
    // implementation details migrated from app.js
}
    }
}

export function tradeShares(entityId, action, amount) { // Exported for use in app.js
    if (!socket || socket.readyState !== WebSocket.OPEN) return;

    socket.send(JSON.stringify({
    }));

    document.getElementById("portfolio-view-overlay")?.remove();
}

export async function openBlackMarket() {
    const state = window.GetGameState();
