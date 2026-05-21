import { CONFIG } from './config.js';
import { socket, myClientId } from './network.js';
import { showToast, hideAllOverlays, renderCardHTML } from './ui.js';
import { collectiveIntelligence } from '../collective-intelligence.js'; // This is a global object, not a module export
import { userAddress, walletProvider, signClient } from './wallet.js';
import { getCachedEnvoiName, getNetworkConfig, resolveEnvoiName, assetCache, resolveAssetSymbol } from './utils.js';
import { globalClubs, availableNetworks, fetchAdminLogs } from './admin.js';
import { lastLobbyPlayers } from './game.js';

const algosdk = window.algosdk;

// --- Industrial Constants ---
export const TERRITORY_MAP = [
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

export const MOOD_CLASS_MAP = {
    "Volatile": "mood-volatile", "Serene": "mood-serene",
    "Spirited": "mood-spirited", "Grounded": "mood-grounded"
};

export const MOOD_EMOJI_MAP = {
    "Volatile": "🔥", "Serene": "💧", "Spirited": "⚡", "Grounded": "⛰️"
};

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

/**
 * Populates and displays the lease board overlay using global club data.
 * PILLAR 1: Industrial Loop.
 */
export function openClubLeaseBoard() {
    const el = document.getElementById("lease-board-overlay");
    if (el) el.classList.remove("hidden");
    renderClubLeaseBoard();
}

export function renderClubLeaseBoard() {
    const container = document.getElementById("lease-grid-container");
    if (!container) return;

    container.innerHTML = `<div class="grid-span-all opacity-5 py-40 italic">Decrypting rental ledgers...</div>`;

    const state = window.GetGameState();
    const myClubId = state.employer_id;

    let leasesHTML = "";
    Object.values(globalClubs).forEach(club => {
        if (!club.leases) return;
        
        Object.values(club.leases).forEach(lease => {
            // PILLAR 3: Lease Filtering. Only display available (non-borrowed) assets.
            if (lease.borrower) return;

            const isMyClub = club.id === myClubId;
            leasesHTML += `
                <div class="lease-item glass-panel animate-slide-up ${isMyClub ? 'border-neon-cyan' : ''}">
                    <div class="item-header flex-row justify-between mb-10">
                        <b class="text-neon-purple" style="letter-spacing: 1px;">${lease.card_name.toUpperCase()}</b>
                        ${isMyClub ? '<span class="tag-cyan font-xs px-5" style="border: 1px solid; border-radius: 4px;">CLUB ASSET</span>' : ''}
                    </div>
                    <div class="font-size-0-8em opacity-6 mb-10">Lender: ${getCachedEnvoiName(lease.lender_wallet)}</div>
                    <div class="lease-stats glass-panel m-0 p-10 mb-15" style="background: rgba(0,0,0,0.2);">
                        <div class="flex-row justify-between mb-5"><span class="opacity-5">Duration:</span> <b>${lease.duration_hours}h</b></div>
                        <div class="flex-row justify-between"><span class="opacity-5">Rate:</span> <b class="text-neon-green">${lease.price.toFixed(2)} $VBV</b></div>
                    </div>
                    <button class="w-full bg-neon-cyan text-dark font-bold py-10" onclick="takeLease('${club.id}', '${lease.id}')">RENT ASSET</button>
                </div>`;
        });
    });

    if (leasesHTML === "") {
        container.innerHTML = `<div class="grid-span-all opacity-3 py-40 italic text-center">The lease board is currently vacant. No tactical assets available for rent.</div>`;
    } else {
        container.innerHTML = leasesHTML;
    }
}

/**
 * openCreateLeaseOverlay allows a player to select a card from their inventory to rent out.
 */
export function openCreateLeaseOverlay() {
    const state = window.GetGameState();
    const myClubId = state.employer_id;
    if (!myClubId) {
        showToast("❌ Access Denied: You must be a club member to list card leases.", "error");
        return;
    }

    const overlay = document.createElement("div");
    overlay.id = "create-lease-overlay";
    overlay.className = "overlay";
    
    const cards = Object.entries(state.inventory || {}).filter(([id, qty]) => qty > 0 && id.startsWith("CARD-"));

    overlay.innerHTML = `
        <div class="economy-panel consignment-panel medium animate-modal">
            <div class="market-header"><span class="market-title">LEASE ADVERTISEMENT</span></div>
            <div class="p-20">
                <p class="opacity-6 font-size-0-85em mb-20">Select an asset to list for lease within your organization. 50% revenue share applies.</p>
                <div class="flex-col gap-10 mb-20 max-h-250 overflow-y-auto">
                    ${cards.map(([id, qty]) => `
                        <div class="portfolio-item glass-panel m-0 p-10 flex-row justify-between align-center pointer" onclick="this.querySelector('input').checked = true; document.getElementById('lease-creation-fields').classList.remove('hidden')">
                            <div class="text-left"><b class="text-neon-cyan">${id.replace('CARD-', 'BABE #')}</b><br/><small class="opacity-5">Owned: ${qty}</small></div>
                            <input type="radio" name="lease-card-id" value="${id.replace('CARD-','')}">
                        </div>`).join('') || '<div class="opacity-3 italic text-center py-20">No eligible cards found.</div>'}
                </div>
                <div id="lease-creation-fields" class="hidden animate-slide-up border-top-glass pt-20">
                    <div class="flex-row gap-10 mb-15">
                        <div class="flex-1"><label class="font-size-0-7em opacity-5 block mb-5">PRICE ($VBV)</label><input type="number" id="lease-price-input" class="glass-input w-full" placeholder="100.0"></div>
                        <div class="flex-1"><label class="font-size-0-7em opacity-5 block mb-5">DURATION (HRS)</label>
                            <select id="lease-duration-input" class="glass-input w-full"><option value="24">24h</option><option value="48">48h</option></select>
                        </div>
                    </div>
                    <div class="flex-row gap-10">
                        <button class="outline w-full" onclick="document.getElementById('create-lease-overlay').remove()">ABORT</button>
                        <button class="w-full bg-neon-purple text-dark font-bold" onclick="submitCreateLease()">POST LEASE</button>
                    </div>
                </div>
            </div>
        </div>`;
    document.body.appendChild(overlay);
}

export function takeLease(clubId, leaseId) {
    if (!socket || socket.readyState !== WebSocket.OPEN) return;
    socket.send(JSON.stringify({ type: "take_lease", payload: { club_id: clubId, lease_id: leaseId } }));
    document.getElementById("lease-board-overlay")?.classList.add("hidden");
}

export function submitCreateLease() {
    const cardInput = document.querySelector('input[name="lease-card-id"]:checked');
    const priceInput = document.getElementById("lease-price-input");
    const durationInput = document.getElementById("lease-duration-input");
    const state = window.GetGameState();

    if (!cardInput || !priceInput.value) return showToast("Select a card and set a price.", "error");
    const price = parseFloat(priceInput.value);
    if (isNaN(price) || price <= 0) return showToast("Invalid price.", "error");

    socket.send(JSON.stringify({
        type: "create_lease",
        payload: {
            club_id: state.employer_id,
            card_id: parseInt(cardInput.value),
            price: price,
            duration_hours: parseInt(durationInput.value)
        }
    }));
    document.getElementById("create-lease-overlay")?.remove();
}
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

/**
 * promptBid opens an input for the user to place a higher bid on an auction item.
 * PILLAR 2: High-Finance.
 */
export function promptBid(auctionId, currentBidMicro) {
    const currentVBV = (currentBidMicro / 1000000).toFixed(1);
    const minBid = (currentBidMicro / 1000000) + 0.1;

    const overlay = document.createElement("div");
    overlay.id = "bid-prompt-overlay";
    overlay.className = "overlay";
    overlay.innerHTML = `
        <div class="economy-panel bid-panel small animate-modal">
            <div class="market-header">
                <span class="market-title">PLACE BID</span>
            </div>
            <div class="p-20 text-center">
                <p class="opacity-6 mb-10">Current High Bid: <b class="text-neon-green">${currentVBV} $VBV</b></p>
                <div class="glass-panel p-15 border-cyan mb-20">
                    <label class="font-size-0-8em text-neon-cyan font-bold block mb-5">YOUR BID ($VBV)</label>
                    <input type="number" id="bid-amount-input" class="glass-input w-full" value="${minBid.toFixed(1)}" step="0.1" min="${minBid}">
                </div>
                <div class="flex-row gap-10">
                    <button class="outline w-full" onclick="document.getElementById('bid-prompt-overlay').remove()">ABORT</button>
                    <button class="w-full bg-neon-green text-dark font-bold" onclick="submitBid('${auctionId}')">CONFIRM BID</button>
                </div>
            </div>
        </div>
    `;
    document.body.appendChild(overlay);
}

/**
 * submitBid sends the bid transaction to the backend.
 */
export async function submitBid(auctionId) {
    const amountInput = document.getElementById("bid-amount-input");
    if (!amountInput || !userAddress) return;

    const amount = parseFloat(amountInput.value);
    if (isNaN(amount) || amount <= 0) return showToast("Invalid bid amount.", "error");

    showToast("⚡ Submitting bid to the Gallery...", "info");

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/auctions/bid`, {
            method: "POST",
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                wallet: userAddress,
                auction_id: auctionId,
                amount: Math.round(amount * 1000000)
            })
        });

        if (response.ok) {
            showToast("✅ Bid accepted!", "success");
            document.getElementById("bid-prompt-overlay")?.remove();
            loadGalleryItems();
        } else {
            const err = await response.text();
            showToast(`❌ Bid Rejected: ${err}`, "error");
        }
    } catch (e) {
        showToast("Gallery connection failure.", "error");
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

// --- Market Ticker Logic ---
let tickerItems = [];
let tickerOffset = 0;
let tickerAnimId = null;
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
{}

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

/**
 * Bounty Ticker: Scrolls live rewards for hunting high-Wanted outlaws.
 * Utilizes a themed canvas (Gold/Red) positioned below the market ticker.
 */
export function updateBountyTicker(players) {
    const spacing = 80;
    let tickerContainer = document.getElementById("bounty-ticker");
    if (!tickerContainer) {
        tickerContainer = document.createElement("div");
        tickerContainer.id = "bounty-ticker";
        tickerContainer.className = "market-ticker-container";
        tickerContainer.style.top = "30px"; // Visual offset for double-ticker stack
        tickerContainer.innerHTML = `
            <div class="ticker-label" style="background: #ff4b4b; color: #fff;">WANTED:</div>
            <canvas id="bounty-ticker-canvas" style="flex: 1; height: 30px; cursor: default;"></canvas>
        `;
        const marketTicker = document.getElementById("market-ticker");
        if (marketTicker) marketTicker.after(tickerContainer);
        else document.body.prepend(tickerContainer);

        const canvas = document.getElementById("bounty-ticker-canvas");
        const resize = () => {
            const dpr = window.devicePixelRatio || 1;
            const rect = canvas.getBoundingClientRect();
            canvas.width = rect.width * dpr;
            canvas.height = 30 * dpr;
            canvas.getContext('2d').scale(dpr, dpr);
        };
        window.addEventListener('resize', resize);
        resize();
    }

    const outlaws = players.filter(p => (p.wanted_level || 0) >= 10).sort((a, b) => b.wanted_level - a.wanted_level);
    const newItems = outlaws.length === 0 ? 
        [{ symbol: "SECTOR SECURE", val: "No active bounties.", color: "#3fb950" }] :
        outlaws.map(p => ({
            symbol: getCachedEnvoiName(p.wallet),
            val: `REWARD: ${(p.wanted_level * 50).toFixed(0)} $VBV`,
            color: "#ffd700"
        }));

    const ctx = document.getElementById("bounty-ticker-canvas").getContext('2d');
    bountyItems = newItems.map(item => {
        ctx.font = "bold 12px 'Rajdhani', sans-serif";
        item.width = ctx.measureText(`${item.symbol} ${item.val}`).width + spacing;
        return item;
    });

    if (!bountyAnimId) startBountyAnimation();
}

export function startBountyAnimation() {
    const canvas = document.getElementById("bounty-ticker-canvas");
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    const animate = () => {
        const width = canvas.width / (window.devicePixelRatio || 1);
        const totalWidth = bountyItems.reduce((s, i) => s + i.width, 0);
        if (totalWidth <= 0) { bountyAnimId = requestAnimationFrame(animate); return; }
        bountyOffset = (bountyOffset + 0.6) % totalWidth;
        ctx.clearRect(0, 0, width, 30);
        ctx.textBaseline = "middle";
        let x = -bountyOffset;
        while (x < width) {
            bountyItems.forEach(item => {
                if (x + item.width > 0 && x < width) {
                    ctx.fillStyle = item.color || "#ffd700";
                    ctx.fillText(item.symbol, x, 15);
                    ctx.fillStyle = "#ffffff";
                    ctx.fillText(" " + item.val, x + ctx.measureText(item.symbol).width, 15);
                }
                x += item.width;
            });
        }
        bountyAnimId = requestAnimationFrame(animate);
    };
    bountyAnimId = requestAnimationFrame(animate);
}

{}
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

export async function openPortfolioView(initialTab = 'portfolio') {
    const el = document.getElementById("portfolio-view-overlay");
    if (el) el.classList.remove("hidden");
    await switchPortfolioTab(initialTab);
}

export async function switchPortfolioTab(tab) {
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
    } else if (tab === 'alliances') {
        const myClubID = state.employer_id;
        const widgetHtml = renderRegionalAllianceWidget(myClubID);
        container.innerHTML = widgetHtml || `<div class="opacity-3 py-40 italic text-center">No active organization or alliance detected.</div>`;
    }
}

/**
 * renderRegionalAllianceWidget generates the HTML for the alliance management block.
 * PILLAR 1: Alliance Integration.
 */
export function renderRegionalAllianceWidget(myClubID) {
    const myClub = globalClubs[myClubID];
    if (!myClub) return "";

    const isPlayerOwned = myClub.owner_wallet && myClub.owner_wallet !== "";
    const isOwner = isPlayerOwned && userAddress && myClub.owner_wallet.toLowerCase() === userAddress.toLowerCase();
    let allianceHtml = "";

    if (!isPlayerOwned) {
        // Case: System Managed Club
        allianceHtml = `
            <div class="alliance-status glass-panel border-cyan mb-10 p-15 accelerated">
                <div class="text-neon-cyan font-bold mb-5" style="letter-spacing: 1px;">🤖 SYSTEM MANAGED CLUB</div>
                <div class="text-left">
                    <b class="text-neon-purple">${myClub.name.toUpperCase()}</b><br>
                    <small class="opacity-7">Districts: ${myClub.territories ? myClub.territories.length : 0}</small><br>
                    <small class="font-xs italic opacity-5">This club is managed by Arena AI. Alliance actions are not available.</small>
                </div>
            </div>`;
    } else if (myClub.allied_club_id) {
        const ally = globalClubs[myClub.allied_club_id];
        const combinedTerritories = (myClub.territories ? myClub.territories.length : 0) + (ally && ally.territories ? ally.territories.length : 0);
        const isGovernor = combinedTerritories >= 2;
        allianceHtml = `
            <div class="alliance-status glass-panel ${isGovernor ? 'border-gold governor-highlight' : 'border-cyan'} mb-10 p-15 accelerated">
                <div class="${isGovernor ? 'text-gold' : 'text-neon-cyan'} font-bold mb-5" style="letter-spacing: 1px;">🤝 ${isGovernor ? 'REGIONAL GOVERNOR ALLIANCE' : 'ACTIVE ALLIANCE'}</div>
                <div class="flex-row justify-between align-center">
                    <div class="text-left">
                        <b class="text-neon-purple">${ally ? ally.name.toUpperCase() : 'UNKNOWN COALITION'}</b><br>
                        <small class="opacity-7">Allied Districts: ${combinedTerritories}</small>
                    </div>
                    ${isOwner ? `<button class="outline danger btn-small" onclick="window.dissolveAlliance('${myClub.id}')">DISSOLVE</button>` : ''}
                </div>
            </div>`;
    } else if (isOwner && myClub.alliance_invite_id) {
        const requester = globalClubs[myClub.alliance_invite_id];
        allianceHtml = `
            <div class="alliance-status glass-panel border-warning mb-10 p-15 accelerated">
                <div class="text-warning font-bold mb-5" style="letter-spacing: 1px;">✉️ ALLIANCE PROPOSAL</div>
                <div class="flex-row justify-between align-center">
                    <div class="text-left">
                        <b>${requester ? requester.name : 'Unknown Club'}</b><br>
                        <small>Owner: ${getCachedEnvoiName(requester ? requester.owner_wallet : '')}</small>
                    </div>
                    <div class="flex-row gap-5">
                        <button class="outline success btn-small" onclick="window.acceptAlliance('${myClub.id}')">ACCEPT</button>
                        <button class="outline btn-small" onclick="socket.send(JSON.stringify({type:'alliance_invite', payload:{my_club_id:'${myClub.id}', target_club_id:''}}))">DECLINE</button>
                    </div>
                </div>
            </div>`;
    } else {
        // PILLAR 1: Independent Status Reset.
        // This view is triggered when no alliance is active or pending.
        const isGovernor = (myClub.territories ? myClub.territories.length : 0) >= 2;
        if (isOwner) {
            const otherClubs = Object.values(globalClubs).filter(c => c.id !== myClub.id && !c.allied_club_id);
            allianceHtml = `
                <div class="alliance-status glass-panel ${isGovernor ? 'border-gold governor-highlight' : 'border-cyan'} mb-10 p-15 accelerated">
                    <div class="${isGovernor ? 'text-gold' : 'text-neon-cyan'} font-bold mb-5" style="letter-spacing: 1px;">${isGovernor ? '👑 REGIONAL GOVERNOR' : '⚔️ INDEPENDENT STATUS'}</div>
                    <div class="text-left mb-15">
                        <b>${myClub.name.toUpperCase()}</b> is currently operating without external coalitions.
                    </div>
                    <small class="section-label opacity-5">PROPOSE COALITION</small>
                    <div class="flex-col gap-5 mt-10 max-h-200 overflow-y-auto">
                        ${otherClubs.map(c => `
                            <div class="player-item p-10 m-0">
                                <div class="text-left"><b>${c.name}</b><br><small>Districts: ${c.territories ? c.territories.length : 0}</small></div>
                                <button class="outline btn-small border-cyan" onclick="window.sendAllianceInvite('${myClub.id}', '${c.id}')">INVITE</button>
                            </div>`).join('') || '<div class="opacity-3 italic text-center py-10 font-xs">No independent clubs available for alliance.</div>'}
                    </div>
                </div>`;
        } else {
            allianceHtml = `
                <div class="alliance-status glass-panel ${isGovernor ? 'border-gold governor-highlight' : 'border-cyan'} mb-10 p-15 accelerated">
                    <div class="${isGovernor ? 'text-gold' : 'text-neon-cyan'} font-bold mb-5" style="letter-spacing: 1px;">${isGovernor ? '🛡️ REGIONAL GOVERNOR' : '🛡️ INDEPENDENT STATUS'}</div>
                    <div class="text-left">
                        <b class="text-neon-purple">${myClub.name.toUpperCase()}</b><br>
                        <small class="opacity-7">Districts: ${myClub.territories ? myClub.territories.length : 0}</small><br>
                        <small class="font-xs italic opacity-5">Your organization is currently independent. Alliance coordination is restricted to the CEO.</small>
                    </div>
                </div>`;
        }
    }
    return allianceHtml;
}

export function tradeShares(entityId, action, amount) {
    if (!socket || socket.readyState !== WebSocket.OPEN) return;
    socket.send(JSON.stringify({ type: "trade_shares", payload: { entity_id: entityId, action, amount } }));
    showToast(`🛰️ Processing ${action} order for ${amount} shares...`, "info");
}
export async function openBlackMarket() {
	const state = window.GetGameState();
	const wanted = state.wanted_level || 0;
	const cunning = state.cunning || 0;

	const REQ_WANTED = 5;
	const REQ_CUNNING = 10;

	if (wanted < REQ_WANTED || cunning < REQ_CUNNING) {
		showToast(`❌ Access Denied: Underworld clearance requires Wanted Level ${REQ_WANTED}+ and Cunning ${REQ_CUNNING}+.`, "error");
		return;
	}

	const overlay = document.getElementById("black-market-overlay") || document.createElement("div");
	overlay.id = "black-market-overlay";
	overlay.className = "overlay";

	const wantedColor = wanted >= REQ_WANTED ? 'var(--neon-green)' : '#ff4b4b';
	const cunningColor = cunning >= REQ_CUNNING ? 'var(--neon-green)' : '#ff4b4b';

	overlay.innerHTML = `
		<div class="economy-panel black-market medium animate-modal">
			<div class="market-header">
				<span class="market-title">THE UNDERWORLD</span>
				<div class="access-level">RESTRICTED ACCESS</div>
			</div>
			
			<div class="market-notice mb-20">
				<div class="notice-icon">💀</div>
				<div class="notice-title">DEFAULTED COLLATERAL</div>
				<p class="notice-text">
					Underworld clearance verified. Access granted via status: 
					<span style="color: ${wantedColor}">WANTED ${wanted}/${REQ_WANTED}</span> • 
					<span style="color: ${cunningColor}">CUNNING ${cunning}/${REQ_CUNNING}</span>.
				</p>
			</div>

			<div id="black-market-grid" class="market-grid" style="max-height: 400px; overflow-y: auto;">
				<div class="opacity-5 py-40 italic text-center">Scanning datastreams for hot assets...</div>
			</div>
			
			<button class="outline mt-20 w-full" onclick="document.getElementById('black-market-overlay').remove()">CLOSE TERMINAL</button>
		</div>
	`;

	if (!document.getElementById("black-market-overlay")) document.body.appendChild(overlay);
	overlay.classList.remove("hidden");

	try {
		const response = await fetch(`${CONFIG.API_BASE}/api/black-market?wallet=${userAddress}`);
		if (!response.ok) throw new Error(await response.text());
		const items = await response.json();
		const grid = document.getElementById("black-market-grid");
		
		if (!items || items.length === 0) {
			grid.innerHTML = `<div class="opacity-3 py-40 italic text-center">No hot items currently available.</div>`;
		} else {
			const wallets = [...new Set(items.map(i => i.borrower_wallet))];
			await Promise.all(wallets.map(w => resolveEnvoiName(w)));

			grid.innerHTML = items.map(item => {
				const priceVBV = (item.repayment_amount * 0.75) / 1000000;
				const borrower = getCachedEnvoiName(item.borrower_wallet);
				return `
					<div class="player-item border-error p-15">
						<div class="text-left">
							<b class="text-neon-cyan">Collateral: ${borrower}</b>
							<div class="font-size-0-75em opacity-6">Hot Asset Bundle</div>
						</div>
						<div class="text-right">
							<b class="text-neon-green">${priceVBV.toFixed(2)} $VBV</b>
							<button class="outline btn-small border-error text-error mt-5" 
									onclick="buyBlackMarketItem('${item.id}', ${priceVBV})">BUY (RISKY)</button>
						</div>
					</div>`;
			}).join('');
		}
	} catch (err) {
		showToast(`Market Link Error: ${err.message}`, "error");
	}
}

export async function buyBlackMarketItem(loanId, price) {
	if (!userAddress) return showToast("Connect wallet first", "error");
	if (!confirm(`Are you sure you want to buy this item for ${price.toFixed(2)} $VBV? This will increase your Wanted Level.`)) return;

	try {
		const state = window.GetGameState();
		const response = await fetch(`${CONFIG.API_BASE}/api/black-market/buy`, {
			method: "POST",
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ wallet: userAddress, loan_id: loanId, network: state.network })
		});

		if (response.ok) {
			const result = await response.json();
			showToast(`🏴‍☠️ ${result.message}`, "success");
			document.getElementById("black-market-overlay")?.remove();
			if (window.syncUI) window.syncUI();
		} else {
			const err = await response.text();
			showToast(`❌ Black Market Purchase Failed: ${err}`, "error");
		}
	} catch (err) {
		showToast(`Purchase Failed: ${err.message}`, "error");
	}
}

// --- Market Ticker Logic ---
// PILLAR 4: Declarative Stability.
let tickerItems = [];
let tickerOffset = 0;
let tickerAnimId = null;
let bountyItems = [];
let bountyOffset = 0;
let bountyAnimId = null;

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

        const canvas = document.getElementById("market-ticker-canvas"); // This is a local variable, not a re-declaration
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

    const newItems = [{ symbol: "MKT TOKEN", val: "0.80 $VBV", trend: "▲", color: "#3fb950" }];

    topPerformers.forEach(p => {
        const basePrice = (p.wins * 10) + (p.reputation / 2) + 100;
        newItems.push({
            symbol: getCachedEnvoiName(p.wallet),
            badge: (p.achievements && p.achievements.length > 0) ? "🏆" : "",
            val: (basePrice + (p.id.charCodeAt(p.id.length - 1) % 5)).toFixed(2),
            trend: (p.wins > 0) ? "▲" : "─",
            color: (p.wins > 0) ? "#3fb950" : "#888",
            isNPC: (collectiveIntelligence.personalities && collectiveIntelligence.personalities[p.id] !== undefined) || p.id === "Vbabe Bot"
        });
    });

    const canvas = document.getElementById("market-ticker-canvas");
    const ctx = canvas ? canvas.getContext('2d') : null; // This is a local variable, not a re-declaration
    if (ctx) {
        tickerItems = newItems.map(item => {
            ctx.font = item.isNPC ? "italic bold 12px 'Rajdhani', sans-serif" : "bold 12px 'Rajdhani', sans-serif";
            item.width = ctx.measureText(`${item.symbol}${item.badge ? ' ' + item.badge : ''} ${item.val} ${item.trend}`).width + spacing;
            return item;
        });
    }

    if (!tickerAnimId) startTickerAnimation();
}

export function startTickerAnimation() {
    const canvas = document.getElementById("market-ticker-canvas");
    if (!canvas) return;
    const ctx = canvas.getContext('2d'); // This is a local variable, not a re-declaration

    const animate = () => {
        if (tickerItems.length === 0) { tickerAnimId = requestAnimationFrame(animate); return; }
        const width = canvas.width / (window.devicePixelRatio || 1);
        const height = 30;
        ctx.clearRect(0, 0, width, height);
        ctx.textBaseline = "middle";

        const totalContentWidth = tickerItems.reduce((sum, item) => sum + (item.width || 0), 0);
        if (totalContentWidth <= 0) { tickerAnimId = requestAnimationFrame(animate); return; }

        tickerOffset = (tickerOffset + 0.8) % totalContentWidth;

        let x = -tickerOffset;
        while (x < width) {
            tickerItems.forEach(item => {
                if (x + item.width > 0 && x < width) {
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
                x += item.width;
            });
        }
        tickerAnimId = requestAnimationFrame(animate);
    };
    tickerAnimId = requestAnimationFrame(animate);
}

/**
 * Bounty Ticker: Scrolls live rewards for hunting high-Wanted outlaws.
 */
export function updateBountyTicker(players) {
    const spacing = 80;
    let tickerContainer = document.getElementById("bounty-ticker");
    if (!tickerContainer) { // This is a local variable, not a re-declaration
        tickerContainer = document.createElement("div");
        tickerContainer.id = "bounty-ticker";
        tickerContainer.className = "market-ticker-container";
        tickerContainer.style.top = "30px";
        tickerContainer.innerHTML = `
            <div class="ticker-label" style="background: #ff4b4b; color: #fff;">WANTED:</div>
            <canvas id="bounty-ticker-canvas" style="flex: 1; height: 30px; cursor: default;"></canvas>
        `;
        const marketTicker = document.getElementById("market-ticker");
        if (marketTicker) marketTicker.after(tickerContainer);
        else document.body.prepend(tickerContainer);

        const canvas = document.getElementById("bounty-ticker-canvas");
        const resize = () => { // This is a local variable, not a re-declaration
            const dpr = window.devicePixelRatio || 1;
            const rect = canvas.getBoundingClientRect();
            canvas.width = rect.width * dpr;
            canvas.height = 30 * dpr;
            canvas.getContext('2d').scale(dpr, dpr);
        };
        window.addEventListener('resize', resize);
        resize();
    }

    const outlaws = players.filter(p => (p.wanted_level || 0) >= 10).sort((a, b) => b.wanted_level - a.wanted_level);
    const newItems = outlaws.length === 0 ? 
        [{ symbol: "SECTOR SECURE", val: "No active bounties.", color: "#3fb950" }] :
        outlaws.map(p => ({
            symbol: getCachedEnvoiName(p.wallet),
            val: `REWARD: ${(p.wanted_level * 50).toFixed(0)} $VBV`,
            color: "#ffd700"
        }));

    const ctx = document.getElementById("bounty-ticker-canvas").getContext('2d');
    bountyItems = newItems.map(item => {
        ctx.font = "bold 12px 'Rajdhani', sans-serif";
        item.width = ctx.measureText(`${item.symbol} ${item.val}`).width + spacing;
        return item;
    });

    if (!bountyAnimId) startBountyAnimation();
}

export function startBountyAnimation() {
    const canvas = document.getElementById("bounty-ticker-canvas");
    if (!canvas) return;
    const ctx = canvas.getContext('2d'); // This is a local variable, not a re-declaration
    const animate = () => {
        const width = canvas.width / (window.devicePixelRatio || 1);
        const totalWidth = bountyItems.reduce((s, i) => s + i.width, 0);
        if (totalWidth <= 0) { bountyAnimId = requestAnimationFrame(animate); return; }
        bountyOffset = (bountyOffset + 0.6) % totalWidth;
        ctx.clearRect(0, 0, width, 30);
        ctx.textBaseline = "middle";
        let x = -bountyOffset;
        while (x < width) {
            bountyItems.forEach(item => {
                if (x + item.width > 0 && x < width) {
                    ctx.fillStyle = item.color || "#ffd700";
                    ctx.fillText(item.symbol, x, 15);
                    ctx.fillStyle = "#ffffff";
                    ctx.fillText(" " + item.val, x + ctx.measureText(item.symbol).width, 15);
                }
                x += item.width;
            });
        }
        bountyAnimId = requestAnimationFrame(animate);
    };
    bountyAnimId = requestAnimationFrame(animate);
}
