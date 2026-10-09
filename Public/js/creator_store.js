// ============================================================================
// creator_store.js — Creator Storefront (REAL API, rewired 2026-09-03)
// ----------------------------------------------------------------------------
// Backend: creator_store_service.go / server_main.go
//   GET  /api/creator/store/products  → { products: [...] }
//   POST /api/creator/store/purchase/ → { success }
//   GET  /api/creator/store/profile/  → { profile }
//   POST /api/creator/store/rate/     → { success }
// All prices are uint64 micro. No mock-data fallback.
// ============================================================================
(function() {
    'use strict';
    var API_BASE = '/api';
    let overlayEl = null;
    let productCatalogEl = null;
    let myCreationsPanelEl = null;
    let commissionSummaryEl = null;
    let statusTextEl = null;

    function init() {
        if (overlayEl) return;
        const html = `
<div id="creator-store-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel creator-panels">
        <button id="btn-close-creator" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>🎨 Creator Marketplace</h2>

        <div class="cs-tabs">
            <button class="cs-tab active" data-tab="marketplace" onclick="window.switchCSTab('marketplace')">Marketplace</button>
            <button class="cs-tab" data-tab="products" onclick="window.switchCSTab('products')">Products</button>
            <button class="cs-tab" data-tab="mycreations" onclick="window.switchCSTab('mycreations')">My Creations</button>
            <button class="cs-tab" data-tab="earnings" onclick="window.switchCSTab('earnings')">Earnings</button>
        </div>

        <div id="cs-panel-marketplace" class="cs-panel">
            <div id="creators-grid" class="creator-cards-grid"></div>
        </div>

        <div id="cs-panel-products" class="cs-panel hidden">
            <h3 id="product-title">Select a creator</h3>
            <p id="product-subtitle" style="color:#b0bec5;margin-bottom:8px;font-size:13px;"></p>
            <div id="products-grid" class="product-cards-grid"></div>
        </div>

        <div id="cs-panel-mycreations" class="cs-panel hidden">
            <h4>My Creations</h4>
            <button id="btn-list-product" class="vbt-btn vbt-btn-primary list-new-product-btn" style="margin-bottom:10px;">+ List New Product</button>
            <div id="my-products-grid" class="product-cards-grid"></div>
        </div>

        <div id="cs-panel-earnings" class="cs-panel hidden">
            <h4>Earnings Summary</h4>
            <div id="commission-summary" class="commission-summary-grid"></div>
            <h4 style="margin-top:12px;">Recent Royalties</h4>
            <ul id="royalty-history-list" class="royalty-history-list"></ul>
        </div>

        <div id="cs-panel-status" class="cs-panel hidden">
            <div id="creator-status-text" style="color:#80cbc4;font-size:13px;">No creator selected.</div>
        </div>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('creator-store-overlay');
        productCatalogEl = document.getElementById('products-grid');
        myCreationsPanelEl = document.getElementById('my-products-grid');
        commissionSummaryEl = document.getElementById('commission-summary');
        statusTextEl = document.getElementById('creator-status-text');
        document.getElementById('btn-close-creator').addEventListener('click', () => { overlayEl.style.display = 'none'; });
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }
    function fmtVBV(micro) { return (parseFloat(micro) / 1000000).toFixed(2); }

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

    // ---------------------------------------------------------------- tabs

    function switchCSTab(tab) {
        document.querySelectorAll('.cs-tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab));
        document.querySelectorAll('.cs-panel').forEach(p => p.classList.add('hidden'));
        const panel = document.getElementById('cs-panel-' + tab);
        if (panel) panel.classList.remove('hidden');
        if (tab === 'marketplace') loadMarketplace();
        if (tab === 'products') loadProducts();
        if (tab === 'mycreations') loadMyCreations();
        if (tab === 'earnings') loadEarnings();
    }

    // ---------------------------------------------------------------- load marketplace (REAL endpoint — products list)

    async function loadMarketplace() {
        try {
            const res = await api('/creator/store/products', { method: 'GET' });
            renderMarketplace(res.products || []);
        } catch (e) {
            console.warn('[CreatorStore] API unreachable:', e.message);
            renderMarketplace([]);
        }
    }

    function renderMarketplace(creators) {
        const el = document.getElementById('creators-grid');
        if (!el) return;
        if (!creators.length) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No creators yet.</p>'; return; }
        el.innerHTML = creators.map(c => `
            <div class="creator-card">
                <span class="creator-name">${esc(c.name || c.creator_wallet)}</span>
                <span class="creator-specialty">${esc(c.category || 'General')}</span>
                <span class="creator-rating">⭐ ${c.rating || '—'}</span>
                <span class="creator-sales">${c.sales || 0} sales</span>
            </div>`).join('');
    }

    // ---------------------------------------------------------------- load products (REAL endpoint)

    async function loadProducts() {
        try {
            const res = await api('/creator/store/products', { method: 'GET' });
            renderProducts(res.products || []);
        } catch (e) {
            console.warn('[CreatorStore] API unreachable:', e.message);
            renderProducts([]);
        }
    }

    function renderProducts(products) {
        const el = productCatalogEl || document.getElementById('products-grid');
        if (!el) return;
        if (!products.length) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No products found.</p>'; return; }

        let html = '';
        products.forEach(p => {
            const priceFormatted = fmtVBV(p.price_micro_vbv || 0);
            const categoryIcon = { '3d_model': '🎨', 'texture': '🖼️', 'audio': '🎵', 'dlc': '📦' }[p.category] || '📦';
            const salesCount = p.sales || 0;
            const desc = p.description || '';
            const shortDesc = desc.slice(0, 80);
            const hasMore = desc.length > 80;

            html += '<div class="product-card">' +
                '<h3>' + categoryIcon + ' ' + esc(p.name || 'Unnamed') + '</h3>' +
                '<p style="color:#90a4ae;font-size:12px;">' + esc(shortDesc) + (hasMore ? '...' : '') + '</p>' +
                '<div class="product-meta">' +
                    '<span style="color:#4dd0e1;font-size:16px;">' + priceFormatted + ' VBV</span>' +
                    '<span style="font-size:12px;color:#b0bec5;margin-left:8px;">Sales: ' + salesCount + '</span>' +
                '</div>' +
                ((p.tags && p.tags.length > 0) ? '<div class="product-tags">' + p.tags.map(function(t){return '<span class="tag">' + esc(t) + '</span>';}).join('') + '</div>' : '') +
                '<button class="vbt-btn vbt-btn-accent buy-product-btn" data-id="' + (p.id || '') + '" onclick="_buyProduct(\'' + (p.id || '') + '\', ' + (p.price_micro_vbv || 0) + ')">Buy Now</button>' +
            '</div>';
        });

        el.innerHTML = html;
    }

    // ---------------------------------------------------------------- my creations (REAL endpoint — by owner wallet)

    async function loadMyCreations() {
        const el = myCreationsPanelEl || document.getElementById('my-products-grid');
        if (!el) return;
        const wallet = getWallet();
        if (!wallet) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">Connect wallet to see creations.</p>'; return; }
        try {
            const res = await api('/creator/store/products?creator=' + encodeURIComponent(wallet), { method: 'GET' });
            renderProducts(res.products || []);
        } catch (e) {
            console.warn('[CreatorStore] API unreachable:', e.message);
            el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No creations listed.</p>';
        }
    }

    // ---------------------------------------------------------------- earnings (REAL endpoint — profile)

    async function loadEarnings() {
        const el = commissionSummaryEl || document.getElementById('commission-summary');
        if (!el) return;
        const wallet = getWallet();
        if (!wallet) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">Connect wallet to see earnings.</p>'; return; }
        try {
            const res = await api('/creator/store/profile/' + encodeURIComponent(wallet), { method: 'GET' });
            const p = res.profile || res || {};
            el.innerHTML = '<p>Total Earnings: <strong>' + fmtVBV(p.total_earnings_micro || 0) + ' VBV</strong></p>' +
                           '<p>Pending: <strong>' + fmtVBV(p.pending_micro || 0) + ' VBV</strong></p>';
        } catch (e) {
            console.warn('[CreatorStore] API unreachable:', e.message);
            el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No earnings yet.</p>';
        }
    }

    // ---------------------------------------------------------------- purchase (REAL endpoint)

    window._buyProduct = function(id, priceMicro) {
        const priceVBV = priceMicro / 1000000;
        const doBuy = async () => {
            try {
                await api('/creator/store/purchase/' + encodeURIComponent(id), {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ wallet: getWallet() }),
                });
                if (window.deductVBV) window.deductVBV(priceMicro);
                if (window.toast) window.toast('Purchased ' + id + '!', 'success');
                loadProducts(); // reload real listings
            } catch (e) {
                if (window.toast) window.toast('Purchase failed: ' + e.message, 'error');
            }
        };
        if (typeof window.txModalOpen === 'function') {
            window.txModalOpen('Buy Product', priceVBV, doBuy);
        } else {
            doBuy();
        }
    };

    window.openCreatorStore = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        switchCSTab('marketplace');
    };

    window.switchCSTab = switchCSTab;

    if (document.getElementById('creator-store-overlay')) init();

    // WS consumers (flow-doc §12.3): live-refresh creator earnings / royalty history when open.
    // Defined only here (canonical Creator Store) to avoid duplicate window.onX handlers.
    window.onCreatorRoyaltyPaid = function () {
        const el = document.getElementById('creator-store-overlay');
        if (el && el.style.display !== 'none') loadEarnings();
    };
    window.onCreatorRoyaltyReceived = function () {
        const el = document.getElementById('creator-store-overlay');
        if (el && el.style.display !== 'none') loadEarnings();
    };
})();
