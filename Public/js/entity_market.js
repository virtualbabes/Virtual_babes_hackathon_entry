// ============================================================================
// entity_market.js — Public Entity Market (REAL API, rewired 2026-09-03)
// ----------------------------------------------------------------------------
// Backend: entity_market.go
//   GET  /api/entity/market/list?type=&region=  → { success, listings: [...] }
//   POST /api/entity/market/create              → { success, listing }
//   POST /api/entity/market/purchase            → { success }
// All prices are uint64 micro. No mock-data fallback.
// ============================================================================
(function () {
    'use strict';
    var API_BASE = '/api';
    const MICRO = 1_000_000;
    let overlayEl = null, catalogEl = null, statusEl = null;
    let currentType = 'pet';
    let currentFilters = { element: 'all', rarity: 'all', minPrice: 0, maxPrice: 10000 };

    function init() {
        if (overlayEl) return;
        const html = `
<div id="entity-market-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel entity-market-panel">
        <button id="btn-close-em" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>🏪 Entity Market</h2>

        <div class="em-tabs">
            <button class="em-tab active" data-type="pet" onclick="window.switchEMTab('pet')">🐾 Pets</button>
            <button class="em-tab" data-type="child_bot" onclick="window.switchEMTab('child_bot')">🤖 Children Bots</button>
            <button class="em-tab" data-type="vehicle" onclick="window.switchEMTab('vehicle')">🚗 Vehicles</button>
        </div>

        <div class="em-filters">
            <select id="em-filter-element" onchange="window.emApplyFilters()">
                <option value="all">All Elements</option>
                <option value="fire">🔥 Fire</option>
                <option value="water">💧 Water</option>
                <option value="earth">🌍 Earth</option>
                <option value="air">💨 Air</option>
                <option value="aether">✨ Aether</option>
                <option value="machine">⚙️ Machine</option>
            </select>
            <select id="em-filter-rarity" onchange="window.emApplyFilters()">
                <option value="all">All Rarities</option>
                <option value="common">Common</option>
                <option value="rare">Rare</option>
                <option value="epic">Epic</option>
                <option value="legendary">Legendary</option>
            </select>
            <input type="number" id="em-filter-min-price" placeholder="Min VBV" value="0" onchange="window.emApplyFilters()" />
            <input type="number" id="em-filter-max-price" placeholder="Max VBV" value="10000" onchange="window.emApplyFilters()" />
        </div>

        <div id="em-catalog" class="em-catalog"></div>

        <div class="em-create">
            <h3>+ Create Listing</h3>
            <div class="em-create-form">
                <label>Entity ID <input id="em-entity-id" type="text" placeholder="PET-..., AI-..., VEH-..." /></label>
                <label>Price (VBV) <input id="em-price" type="number" min="0" value="0" step="0.01" /></label>
                <label>Description <input id="em-desc" type="text" maxlength="120" placeholder="Bloodline, stats, etc." /></label>
                <label>Duration (hours) <input id="em-duration" type="number" min="0" value="24" /></label>
                <button id="em-create-btn" class="vbt-btn vbt-btn-primary">Post Listing</button>
            </div>
            <div id="em-my-listings" class="em-my-listings"></div>
        </div>

        <p id="em-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('entity-market-overlay');
        catalogEl = document.getElementById('em-catalog');
        statusEl = document.getElementById('em-status');
        document.getElementById('btn-close-em').addEventListener('click', () => { overlayEl.style.display = 'none'; });
        document.getElementById('em-create-btn').addEventListener('click', onCreate);
        load();
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }
    function fmtVBV(micro) { return (parseFloat(micro) / MICRO).toFixed(4); }

    // ---------------------------------------------------------------- fetch helper

    async function api(path, opts) {
        const url = path.startsWith('/api/') ? path : API_BASE + path;
        const resp = await fetch(url, opts);
        if (!resp.ok) { let d = ''; try { d = (await resp.json()).error || ''; } catch(e){} throw new Error(d || ('HTTP ' + resp.status)); }
        return resp.json();
    }

    function getWallet() {
        if (window.getWalletAddress) return window.getWalletAddress() || '';
        if (window.getActiveWallet) return window.getActiveWallet() || '';
        return localStorage.getItem('wallet_address') || '';
    }

    // ---------------------------------------------------------------- tabs/filters

    function switchEMTab(type) {
        currentType = type;
        document.querySelectorAll('.em-tab').forEach(b => b.classList.toggle('active', b.dataset.type === type));
        load();
    }

    function applyFilters() {
        currentFilters.element = document.getElementById('em-filter-element')?.value || 'all';
        currentFilters.rarity = document.getElementById('em-filter-rarity')?.value || 'all';
        currentFilters.minPrice = parseFloat(document.getElementById('em-filter-min-price')?.value) || 0;
        currentFilters.maxPrice = parseFloat(document.getElementById('em-filter-max-price')?.value) || 10000;
        load();
    }

    // ---------------------------------------------------------------- load listings (REAL endpoint)

    async function load() {
        if (!catalogEl) return;
        try {
            // Backend route is /api/entity-market/list (hyphen, not /api/entity/market/list)
            const res = await api('/entity-market/list?type=' + encodeURIComponent(currentType));
            renderCatalog(res.listings || []);
        } catch (e) {
            // EMPTY STATE on failure — never mock data
            console.warn('[EntityMarket] API unreachable:', e.message);
            renderCatalog([]);
        }
    }

    // ---------------------------------------------------------------- render

    function renderCatalog(listings) {
        if (!catalogEl) return;
        if (!listings.length) { catalogEl.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No listings found.</p>'; return; }
        catalogEl.innerHTML = '<div class="em-grid">' + listings.map(l => {
            const rarityColor = l.rarity === 'legendary' ? '#ffd700' : l.rarity === 'epic' ? '#a855f7' : l.rarity === 'rare' ? '#3b82f6' : '#9ca3af';
            // Backend returns price_micro (uint64). Fall back to price for compatibility.
            const priceVBV = l.price_micro !== undefined ? fmtVBV(l.price_micro) : (l.price || 0);
            return `
            <div class="em-card" style="border-color: ${rarityColor}">
                <div class="em-card-header">
                    <span class="em-card-name">${esc(l.name)}</span>
                    <span class="em-card-rarity" style="color: ${rarityColor}">${esc(l.rarity)}</span>
                </div>
                <div class="em-card-level">Lv.${l.level}</div>
                <div class="em-card-price">${priceVBV} VBV</div>
                <div class="em-card-seller">${esc(l.seller)}</div>
                <button class="vbt-btn vbt-btn-primary em-buy-btn" onclick="window.emBuy('${l.id}', ${l.price_micro || (l.price * MICRO)}, '${esc(l.name)}')">Buy</button>
            </div>`;
        }).join('') + '</div>';
    }

    // ---------------------------------------------------------------- create listing (REAL endpoint)

    async function onCreate() {
        const entityId = document.getElementById('em-entity-id')?.value.trim();
        const priceVBV = parseFloat(document.getElementById('em-price')?.value) || 0;
        const desc = document.getElementById('em-desc')?.value.trim();
        const duration = parseInt(document.getElementById('em-duration')?.value) || 24;
        const wallet = getWallet();

        if (!entityId || !priceVBV) { setStatus('Fill all fields!'); return; }
        if (!wallet) { setStatus('Connect wallet first!'); return; }

        // uint64 strictness: convert VBV → micro-units (no float math on ledger values)
        const priceMicro = Math.round(priceVBV * MICRO);

        try {
            await api('/entity/market/create', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    entity_id: entityId,
                    type: currentType,
                    price_micro: priceMicro,
                    description: desc,
                    duration_hours: duration,
                }),
            });
            setStatus('Listing posted!');
            if (window.toast) window.toast('Listing posted!', 'success');
            load(); // reload real listings
        } catch (e) {
            setStatus('Error: ' + e.message);
            if (window.toast) window.toast('Create failed: ' + e.message, 'error');
        }
    }

    // ---------------------------------------------------------------- purchase (REAL endpoint)

    async function emBuy(id, priceMicro, name) {
        const wallet = getWallet();
        if (!wallet) { if (window.toast) window.toast('Connect wallet first', 'warn'); return; }

        // priceMicro is uint64. txModalOpen expects VBV for display.
        const priceVBV = priceMicro / MICRO;

        const doPurchase = async () => {
            try {
                await api('/entity/market/purchase', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ entity_id: id, type: currentType }),
                });
                if (window.deductVBV) window.deductVBV(priceMicro); // deduct in micro (uint64 safe)
                if (window.toast) window.toast('Purchased ' + (name || id) + '!', 'success');
                load(); // reload real listings
            } catch (e) {
                if (window.toast) window.toast('Purchase failed: ' + e.message, 'error');
            }
        };

        if (window.txModalOpen) {
            window.txModalOpen('Buy ' + (name || id), priceVBV, doPurchase);
        } else {
            doPurchase();
        }
    }

    function setStatus(m) { if (statusEl) statusEl.textContent = m || ''; }

    window.openEntityMarket = function () {
        init();
        if (window.hideAllOverlays) window.hideAllOverlays();
        overlayEl.style.display = 'flex';
    };
    window.switchEMTab = switchEMTab;
    window.emApplyFilters = applyFilters;
    window.emBuy = emBuy;
})();
