import { CONFIG } from './config.js';
import { socket, myClientId } from './network.js';
import { showToast, hideAllOverlays, renderCardHTML } from './ui.js';
import { collectiveIntelligence } from '../collective-intelligence.js';
import { userAddress, walletProvider, signClient, updateWalletUI } from './wallet.js';
import { getCachedEnvoiName, getNetworkConfig, resolveEnvoiName, assetCache, resolveAssetSymbol } from './utils.js';
import { globalClubs, availableNetworks } from './admin.js';
import { lastLobbyPlayers } from './game.js';

const algosdk = window.algosdk;

// --- Item Registry (Mirrors shop_registry.go) ---
export const GlobalShopRegistry = {
    "mood_catalyst": { name: "Mood Catalyst", desc: "+50 Mood Bonus (3 Matches)", price: 100, ClubType: "Elemental" },
    "grounded_shield": { name: "Grounded Shield", desc: "Immunity to Mood Penalties (5 Matches)", price: 250, ClubType: "Elemental", requiredMojo: 100 },
    "prism_shield": { name: "Prism Shield", desc: "Reflects Mood Penalties back to Opponent", price: 750, ClubType: "Elemental", requiredMojo: 500, isMasterTier: true },
    "rule_breaker": { name: "Rule Breaker", desc: "Force PLUS trigger (1 Match)", price: 150, ClubType: "Tactical" },
    "intel_report": { name: "Intel Report", desc: "See Opponent Hand (3 Matches)", price: 300, ClubType: "Tactical", requiredMojo: 150 },
    "ghost_protocol": { name: "Ghost Protocol", desc: "Match outcome hidden from Ticker", price: 1000, ClubType: "Tactical", requiredMojo: 600, requiredRole: "Security", isMasterTier: true },
    "stamina_stim": { name: "Stamina Stim", desc: "-20 Fatigue Immediately", price: 100, ClubType: "Vitality" },
    "loyalty_pledge": { name: "Loyalty Pledge", desc: "+10 Loyalty Immediately", price: 500, ClubType: "Vitality", requiredMojo: 200 },
    "hyper_stim": { name: "Hyper-Stim", desc: "Resets fatigue for current deck", price: 1500, ClubType: "Vitality", requiredMojo: 800, requiredRole: "Manager", isMasterTier: true },
    "tripwire": { name: "Laser Tripwire", desc: "+10% Heist Failure", price: 500, ClubType: "Hardware", requiredRole: "Security" },
    "sentry_turret": { name: "Sentry Turret", desc: "+25% Heist Failure", price: 1200, ClubType: "Hardware", requiredRole: "Security", requiredMojo: 300 },
    "guard_dog": { name: "Bio-Guard Dog", desc: "Forces Jail on Failure", price: 2000, ClubType: "Hardware", requiredRole: "Security", requiredMojo: 500 }
};

// --- Economic Features ---

/**
 * Populates and displays the district shops overlay using synchronized club inventory.
 * Utilizes the high-fidelity _shops.scss styles and category filtering.
 */
export async function openShopsOverlay(initialCategory = 'Elemental') {
    const overlay = document.getElementById("shops-overlay");
    if (!overlay) return;
    overlay.classList.remove("hidden");
    switchShopCategory(initialCategory);
}

export function switchShopCategory(category) {
    const container = document.getElementById("shops-container");
    if (!container) return;

    // Update Tab State
    document.querySelectorAll('.category-tab').forEach(tab => {
        tab.classList.toggle('active', tab.dataset.category === category);
    });

    container.innerHTML = `<div class="grid-span-all opacity-5 py-40 italic">Scanning district stock for ${category} hardware...</div>`;

    const state = window.GetGameState();
    const userRole = state.job_role || "";

    const clubs = Object.values(globalClubs).filter(c => c.type === category);
    let itemsHTML = "";

    clubs.forEach(club => {
        Object.entries(club.inventory || {}).forEach(([itemId, qty]) => {
            if (qty <= 0) return;
            const meta = GlobalShopRegistry[itemId] || { name: itemId.replace(/_/g, ' '), desc: "Tactical Enhancement", price: 100 };

            // TACTICAL EVALUATION: Check if player/club meets unlock criteria
            const meetsMojo = (club.mojo || 0) >= (meta.requiredMojo || 0);
            const meetsRole = !meta.requiredRole || userRole === meta.requiredRole;
            const meetsMaster = !meta.isMasterTier || (club.territories && club.territories.length >= 2);
            const isLocked = !meetsMojo || !meetsRole || !meetsMaster;

            let reqLabels = [];
            if (meta.requiredMojo) reqLabels.push(`<span class="${meetsMojo ? 'text-neon-green' : 'text-error'}">MOJO ${meta.requiredMojo}+</span>`);
            if (meta.requiredRole) reqLabels.push(`<span class="${meetsRole ? 'text-neon-green' : 'text-error'}">${meta.requiredRole.toUpperCase()}</span>`);
            if (meta.isMasterTier) reqLabels.push(`<span class="${meetsMaster ? 'text-neon-green' : 'text-error'}">GOVERNOR</span>`);
            
            itemsHTML += `
                <div class="shop-item animate-slide-up ${meta.isMasterTier ? 'master-tier' : ''} ${isLocked ? 'locked-item' : ''}" 
                     onclick="${isLocked ? '' : `buyClubItem('${club.id}', '${itemId}', ${meta.price}, '${club.territories[0]}')`}">
                    <div class="item-image">
                        <img src="Assets/Images/portraits/placeholder.webp" alt="Hardware">
                        <div class="item-badge">${club.name}</div>
                    </div>
                    <div class="item-info">
                        <div class="item-title">${meta.name.toUpperCase()}</div>
                        <div class="item-description">${meta.desc}</div>
                        ${reqLabels.length > 0 ? `<div class="item-requirements" style="font-size: 0.7em; margin-top: 5px; font-weight: bold; letter-spacing: 1px;">${reqLabels.join(' • ')}</div>` : ''}
                        <div class="item-stats">
                            <div class="stat">
                                <div class="stat-label">STOCK</div>
                                <div class="stat-value">${qty}</div>
                            </div>
                        </div>
                    </div>
                    <div class="item-footer">
                        <div class="item-price">${meta.price}</div>
                        <button class="buy-button" ${isLocked ? 'disabled' : ''}>${isLocked ? 'LOCKED' : 'PURCHASE'}</button>
                    </div>
                </div>
            `;
        });
    });

    if (itemsHTML === "") {
        container.innerHTML = `<div class="grid-span-all opacity-3 py-40 italic">Sector is currently dry for ${category} assets.</div>`;
    } else {
        container.innerHTML = itemsHTML;
    }
}

export async function buyClubItem(clubId, itemId, price, territoryId) {
    if (!userAddress) return showToast("Connect wallet first", "error");
    
    try {
        showToast(`Purchasing ${itemId} for ${price} $VBV...`, "info");
        
        socket.send(JSON.stringify({
            type: "purchase_item",
            payload: {
                item_id: itemId,
                territory_id: territoryId,
                price: price * 1000000 // Convert to micro-units
            }
        }));

        if (itemId === "stamina_stim") {
            showToast("⚡ Fatigue reduced! Your cards are feeling refreshed.", "success");
        }

        // Overlay removal logic depends on which UI triggered it
        const territoryOverlay = document.getElementById("territory-view-overlay");
        if (territoryOverlay) territoryOverlay.remove();
    } catch (err) {
        showToast(`Purchase Failed: ${err.message}`, "error");
    }
}

/**
 * Art Gallery Interface: Consignment and Auctions.
 */
export async function openArtGalleryOverlay() {
    const overlay = document.createElement("div");
    overlay.id = "art-gallery-overlay";
    overlay.className = "overlay";
    
    overlay.innerHTML = `
        <div class="economy-panel gallery-panel large" style="max-height: 85vh; overflow-y: auto;">
            <div class="economy-header">
                <span class="economy-title">THE ART GALLERY</span>
                <div class="flex-row gap-15">
                    <button class="outline x-small" onclick="openConsignmentOverlay()">CONSIGN ITEM</button>
                    <button class="outline x-small border-error" onclick="document.getElementById('art-gallery-overlay').remove()">CLOSE</button>
                </div>
            </div>

            <div class="auction-gallery">
                <div class="gallery-header">
                    <p class="opacity-7 italic font-size-0-85em">Tactical assets and rare artifacts up for public auction. All sales support the Industrial Loop.</p>
                </div>
                
                <div id="gallery-items-container" class="gallery-grid">
                    <div class="grid-span-all opacity-5 py-40 italic">Decrypting auction datastreams...</div>
                </div>
            </div>
        </div>
    `;

    document.body.appendChild(overlay);
    loadGalleryItems();
}

export async function loadGalleryItems() {
    const container = document.getElementById("gallery-items-container");
    if (!container) return;

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/auctions`);
        const auctions = await response.json();

        if (!auctions || auctions.length === 0) {
            container.innerHTML = `<div style="grid-column: 1/-1;" class="opacity-5 py-40 italic text-center">The gallery floor is currently vacant. Check back during peak match hours.</div>`;
            return;
        }

        container.innerHTML = auctions.map(a => {
            const timeRemaining = Math.max(0, new Date(a.ends_at) - new Date());
            const hours = Math.floor(timeRemaining / 3600000);
            const mins = Math.floor((timeRemaining % 3600000) / 60000);
            
            return `
                <div class="gallery-grid__item-bundle animate-slide-up">
                    <div class="item-image">
                        <img src="Assets/Images/portraits/placeholder.webp" alt="Exhibit">
                    </div>
                    <div class="item-info text-left">
                        <div class="item-title font-bold text-neon-cyan">${a.bundle.weapon_id ? a.bundle.weapon_id.replace(/_/g, ' ') : 'Tactical Artifact'}</div>
                        <div class="item-description font-size-0-8em opacity-6">Seller: ${a.seller_name}</div>
                    </div>
                    <div class="auction-info mt-10">
                        <div class="current-bid">
                            <span class="bid-label">HIGHEST BID</span>
                            <span class="bid-amount text-neon-green">${(a.current_bid / 1000000).toFixed(1)} $VBV</span>
                        </div>
                        <div class="time-remaining">
                            <span class="time-label">REMAINING</span>
                            <span class="time-value">${hours}h ${mins}m</span>
                        </div>
                    </div>
                    <button class="outline mt-15 w-full border-cyan text-neon-cyan" onclick="promptBid('${a.id}', ${a.current_bid})">PLACE BID</button>
                </div>`;
        }).join('');
    } catch (err) {
        container.innerHTML = `<div style="grid-column: 1/-1;" class="text-error py-40 text-center">Gallery Indexer Unreachable.</div>`;
    }
}

export function openConsignmentOverlay() {
    const state = window.GetGameState();
    const overlay = document.createElement("div");
    overlay.id = "consignment-overlay";
    overlay.className = "overlay";

    const listableItems = Object.entries(state.inventory || {}).filter(([id, qty]) => qty > 0 && id.startsWith("CARD-"));

    overlay.innerHTML = `
        <div class="economy-panel consignment-panel medium">
            <div class="market-header">
                <span class="market-title text-neon-purple">ASSET CONSIGNMENT</span>
                <div class="access-level">GALLERY PROTOCOL: CARDS ONLY</div>
            </div>

            <div class="p-20">
                <p class="opacity-6 font-size-0-85em mb-20">Select an asset from your collection to list on the public auction floor. 10% commission applies on successful settlement.</p>
                
                <div class="flex-col gap-10 mb-20" style="max-height: 300px; overflow-y: auto;">
                    ${listableItems.length === 0 ? '<div class="opacity-3 italic py-20 text-center">No listable tactical assets detected.</div>' : 
                        listableItems.map(([id, qty]) => `
                            <div class="portfolio-item glass-panel m-0 p-10 flex-row justify-between align-center pointer" onclick="selectConsignmentItem('${id}')">
                                <div class="flex-row align-center gap-10">
                                    <div class="item-icon font-size-1-2em">📦</div>
                                    <div class="text-left">
                                        <div id="item-name-${id}" class="font-bold text-neon-cyan">${id.replace(/_/g, ' ').toUpperCase()}</div>
                                        <div class="font-size-0-75em opacity-5">Available: ${qty}</div>
                                    </div>
                                </div>
                                <input type="radio" name="consignment-target" value="${id}">
                            </div>
                        `).join('')}
                </div>

                <div id="consignment-pricing" class="hidden animate-slide-up">
                    <div class="glass-panel p-15 border-cyan">
                        <label class="font-size-0-8em text-neon-cyan font-bold block mb-5">STARTING BID ($VBV)</label>
                        <input type="number" id="consignment-bid-input" class="glass-input w-full mb-10" placeholder="e.g. 500.00" step="0.1">
                        <small class="opacity-5 italic">Note: Auctions run for 24 hours from timestamp of listing.</small>
                    </div>
                    
                    <div class="flex-row gap-15 mt-20">
                        <button class="outline w-full" onclick="document.getElementById('consignment-overlay').remove()">ABORT</button>
                        <button class="w-full bg-neon-purple text-dark font-bold" onclick="submitConsignment()">LIST ASSET</button>
                    </div>
                </div>
            </div>
        </div>
    `;

    document.body.appendChild(overlay);
}

export function selectConsignmentItem(id) {
    const radio = document.querySelector(`input[value="${id}"]`);
    if (radio) radio.checked = true;
    document.getElementById("consignment-pricing")?.classList.remove("hidden");
}

export async function submitConsignment() {
    const selectedInput = document.querySelector('input[name="consignment-target"]:checked');
    const bidInput = document.getElementById("consignment-bid-input");
    
    if (!selectedInput || !bidInput || !bidInput.value) {
        showToast("Please select an item and enter a starting bid.", "error");
        return;
    }

    const itemId = selectedInput.value;
    const bidBase = parseFloat(bidInput.value);
    if (isNaN(bidBase) || bidBase <= 0) {
        showToast("Invalid starting bid.", "error");
        return;
    }

    showToast("⚡ Authorizing consignment protocol...", "info");
    
    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/auctions/create`, {
            method: "POST",
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                wallet: userAddress,
                item_id: itemId,
                starting_bid: Math.round(bidBase * 1000000), 
                territory_id: "the_art_gallery"
            })
        });

        if (response.ok) {
            showToast(`✅ Asset listed! ${itemId.replace(/_/g, ' ')} is now on the auction floor.`, "success");
            document.getElementById("consignment-overlay")?.remove();
            loadGalleryItems(); 
        } else {
            const err = await response.text();
            showToast(`❌ Listing Failed: ${err}`, "error");
        }
    } catch (e) {
        showToast("Gallery connection failure.", "error");
    }
}


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
            isNPC: (collectiveIntelligence.personalities && collectiveIntelligence.personalities[p.id] !== undefined) || p.id === "Vbabe Bot"
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
        let txid = "SIM_TX_" + Date.now(); 
        socket.send(JSON.stringify({ type: "create_club", payload: { name, type, territory_id: territory, txid, network: state.network } }));
        document.getElementById("club-foundry-overlay").remove();
        if (window.triggerFoundryFusion) window.triggerFoundryFusion(type);
    } catch (err) { showToast(`Founding Failed: ${err.message}`, "error"); }
}

export function openShopsOverlay(category = 'Elemental') {
    const el = document.getElementById("shops-overlay");
    if (el) el.classList.remove("hidden");
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
    const el = document.getElementById("portfolio-view-overlay");
    if (el) el.classList.remove("hidden");
    switchPortfolioTab(initialTab);
}

export function switchPortfolioTab(tab) {
    const container = document.getElementById("portfolio-content-area");
    if (!container) return;

    document.querySelectorAll('.portfolio-tab').forEach(t => t.classList.toggle('active', t.dataset.tab === tab));

    const state = window.GetGameState();
    if (tab === 'portfolio') {
        const entries = Object.entries(state.portfolio || {});
        if (entries.length === 0) {
            container.innerHTML = `<div class="opacity-3 py-40 italic">No entity holdings detected.</div>`;
            return;
        }
        await Promise.all(entries.map(([w]) => resolveEnvoiName(w)));
        container.innerHTML = `
            <div class="portfolio-grid" style="display: grid; grid-template-columns: 1fr; gap: 10px;">
                ${entries.map(([id, amt]) => {
                    const p = lastLobbyPlayers.find(pl => pl.wallet?.toLowerCase() === id.toLowerCase());
                    const price = p ? ((p.wins * 10) + (p.reputation / 2) + 100) : 100;
                    return `
                        <div class="portfolio-item glass-panel m-0 flex-row justify-between align-center p-15">
                            <div class="text-left">
                                <b class="text-neon-cyan">${getCachedEnvoiName(id)}</b>
                                <div class="font-size-0-75em opacity-5">${amt.toFixed(2)} SHARES</div>
                            </div>
                            <div class="text-right">
                                <div class="text-neon-green font-bold">${(amt * price).toFixed(1)} $VBV</div>
                                <button class="outline x-small border-error mt-5" onclick="tradeShares('${id}', 'sell', ${amt})">LIQUIDATE</button>
                            </div>
                        </div>`;
                }).join('')}
            </div>`;
    } else if (tab === 'jailed') {
        const jailed = state.jailed_cards || {};
        const entries = Object.entries(jailed);
        container.innerHTML = entries.length ? entries.map(([cardId, clubId]) => `
            <div class="player-item border-error p-15">
                <div class="text-left">
                    <b class="text-error">CARD #${cardId}</b>
                    <div class="font-size-0-75em opacity-6">Held by: ${globalClubs[clubId]?.name || 'Unknown Entity'}</div>
                </div>
                <button class="outline btn-small border-neon-green text-neon-green" onclick="window.initiateBail(${cardId}, '${clubId}')">PAY BAIL (200 $VBV)</button>
            </div>`).join('') : `<div class="opacity-3 py-40 italic">No cards in sector custody.</div>`;
    } else if (tab === 'kidnapped') {
        const kidnapped = state.kidnapped_cards || {};
        const entries = Object.entries(kidnapped);
        if (entries.length > 0) await Promise.all(entries.map(([_, w]) => resolveEnvoiName(w)));
        container.innerHTML = entries.length ? entries.map(([cardId, victimWallet]) => `
            <div class="player-item border-warning p-15" style="border-color: #ffa500;">
                <div class="text-left">
                    <b style="color: #ffa500;">CARD #${cardId}</b>
                    <div class="font-size-0-75em opacity-6">Victim: ${getCachedEnvoiName(victimWallet)}</div>
                </div>
                <button class="outline btn-small border-gold text-gold" onclick="window.releaseHostage(${cardId})">RELEASE</button>
            </div>`).join('') : `<div class="opacity-3 py-40 italic">No hostages in your custody.</div>`;
    } else if (tab === 'hostage') {
        const heldHostage = state.held_hostage_cards || {};
        const entries = Object.entries(heldHostage);
        if (entries.length > 0) await Promise.all(entries.map(([_, w]) => resolveEnvoiName(w)));
        container.innerHTML = entries.length ? entries.map(([cardId, perpWallet]) => `
            <div class="player-item border-gold p-15">
                <div class="text-left">
                    <b class="text-gold">CARD #${cardId}</b>
                    <div class="font-size-0-75em opacity-6">Kidnapper: ${getCachedEnvoiName(perpWallet)}</div>
                </div>
                <button class="outline btn-small border-error text-error" onclick="window.payRansom(${cardId}, '${perpWallet}')">PAY RANSOM</button>
            </div>`).join('') + `<div class="mt-10 p-10 border-top-glass opacity-5 italic font-size-0-75em text-center">Insurance recovery active: 48h cycle.</div>` 
            : `<div class="opacity-3 py-40 italic">No assets currently held for ransom.</div>`;
    }
}

export function tradeShares(entityId, action, amount) {
    if (!socket || socket.readyState !== WebSocket.OPEN) return;
    socket.send(JSON.stringify({ type: "trade_shares", payload: { entity_id: entityId, action, amount } }));
    showToast(`🛰️ Processing ${action} order for ${amount} shares...`, "info");
}

export function openClubLeaseBoard() {
    const el = document.getElementById("lease-board-overlay");
    if (el) el.classList.remove("hidden");
}

export function takeLease(clubId, leaseId) {
    if (!socket || socket.readyState !== WebSocket.OPEN) return;
    socket.send(JSON.stringify({ type: "take_lease", payload: { club_id: clubId, lease_id: leaseId } }));
    document.getElementById("lease-board-overlay")?.classList.add("hidden");
}

export async function openBlackMarket() {
    const state = window.GetGameState();
    const el = document.getElementById("black-market-overlay");
    if (el) el.classList.remove("hidden");
}

export function buyBlackMarketItem(itemId, price) {
    if (!socket || socket.readyState !== WebSocket.OPEN) return;
    socket.send(JSON.stringify({ type: "buy_black_market", payload: { item_id: itemId, price: price } }));
}

window.updateMarketTicker = updateMarketTicker;
window.adjustMapZoom = adjustMapZoom;
window.openTerritoryMapOverlay = openTerritoryMapOverlay;
window.openClubFoundry = openClubFoundry;
window.submitClubFoundry = submitClubFoundry;
window.openShopsOverlay = openShopsOverlay;
window.switchShopCategory = switchShopCategory;
window.openPortfolioView = openPortfolioView;
window.switchPortfolioTab = switchPortfolioTab;
window.tradeShares = tradeShares;
window.openClubLeaseBoard = openClubLeaseBoard;
window.takeLease = takeLease;
window.openBlackMarket = openBlackMarket;
window.buyBlackMarketItem = buyBlackMarketItem;
