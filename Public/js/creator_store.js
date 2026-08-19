// CreatorStore frontend module — Neon-glass creator storefront UI + WS integration
(function() {
    'use strict';

    // State
    let state = {
        creators: [],             // List of all visible creators
        selectedCreatorId: null,  // Currently viewed creator wallet
        products: [],             // Products from selected creator
        myProducts: [],           // User's own listed products
        reviews: [],              // Reviews for selected creator's products
        commissionTotalMicroVBV: 0,
        royaltyHistory: [],       // Secondary-sale royalties earned
        currentWallet: '',
        wsConnected: false
    };

    // DOM references (lazy-initialized)
    let overlayEl = null;
    let creatorsGridEl = null;
    let productCatalogEl = null;
    let myCreationsPanelEl = null;
    let reviewsPanelEl = null;
    let commissionSummaryEl = null;
    let statusTextEl = null;

    // API base path
    const API_BASE = '/api';

    // Initialize module — lazy DOM setup on first open
    function init() {
        if (state._initialized) return;
        state._initialized = true;
        buildOverlay();
        wireEvents();
    }

    // Build the overlay panel HTML once
    function buildOverlay() {
        const html = `
<div id="creator-store-overlay" class="vbt-overlay" style="display:none;">
    <div class="neon-glass-panel creator-panels">
        <!-- Panel 1: Creator Marketplace -->
        <section class="panel panel-marketplace">
            <h2>🎨 Creator Marketplace</h2>
            <button id="btn-close-creator" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
            <div id="creators-grid" class="creator-cards-grid"></div>
        </section>

        <!-- Panel 2: Product Catalog -->
        <section class="panel panel-products">
            <h3 id="product-title">Select a creator</h3>
            <p id="product-subtitle" style="color:#b0bec5;margin-bottom:8px;font-size:13px;"></p>
            <div id="products-grid" class="product-cards-grid"></div>
        </section>

        <!-- Panel 3: My Creations + Commission Tracker -->
        <section class="panel panel-mycreations">
            <h4>My Creations</h4>
            <button id="btn-list-product" class="vbt-btn vbt-btn-primary list-new-product-btn" style="margin-bottom:10px;">+ List New Product</button>
            <div id="my-products-grid" class="product-cards-grid"></div>
        </section>

        <!-- Panel 4: Reviews & Commissions -->
        <section class="panel panel-reviews">
            <h4>Earnings Summary</h4>
            <div id="commission-summary" class="commission-summary-grid"></div>
            <h4 style="margin-top:12px;">Recent Royalties</h4>
            <ul id="royalty-history-list" class="royalty-history-list"></ul>
        </section>

        <!-- Status -->
        <section class="panel panel-status">
            <div id="creator-status-text" style="color:#80cbc4;font-size:13px;">No creator selected.</div>
        </section>

        <!-- Create Product Modal (hidden by default) -->
        <div id="create-product-modal" class="vbt-overlay-inner" style="display:none;position:absolute;top:50%;left:50%;transform:translate(-50%,-50%);background:#1a237e;border:1px solid #4dd0e1;border-radius:8px;padding:20px;width:400px;z-index:10;">
            <h3 style="color:#fff;margin-bottom:12px;">List New Product</h3>
            <input id="cp-name" placeholder="Product name (max 128 chars)" style="width:95%;padding:6px;margin-bottom:6px;background:#0d1b4a;color:#fff;border:1px solid #4dd0e1;border-radius:4px;" />
            <textarea id="cp-desc" placeholder="Description (max 2048 chars)" style="width:95%;padding:6px;margin-bottom:6px;background:#0d1b4a;color:#fff;border:1px solid #4dd0e1;border-radius:4px;height:60px;"></textarea>
            <select id="cp-category" style="width:95%;padding:6px;margin-bottom:6px;background:#0d1b4a;color:#fff;border:1px solid #4dd0e1;border-radius:4px;">
                <option value="asset">Asset</option>
                <option value="dlc">DLC</option>
                <option value="service">Service</option>
                <option value="cosmetic">Cosmetic</option>
            </select>
            <input id="cp-price" placeholder="Price (micro-VBV)" type="number" style="width:95%;padding:6px;margin-bottom:6px;background:#0d1b4a;color:#fff;border:1px solid #4dd0e1;border-radius:4px;" />
            <input id="cp-tags" placeholder="Tags (comma-separated, max 8)" style="width:95%;padding:6px;margin-bottom:6px;background:#0d1b4a;color:#fff;border:1px solid #4dd0e1;border-radius:4px;" />
            <div style="display:flex;gap:8px;">
                <button id="btn-confirm-list" class="vbt-btn vbt-btn-accent">Confirm Listing</button>
                <button id="btn-cancel-list" class="vbt-btn vbt-btn-secondary">Cancel</button>
            </div>
        </div>
    </div>
</div>`;

        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('creator-store-overlay');
        creatorsGridEl = document.getElementById('creators-grid');
        productCatalogEl = document.getElementById('products-grid');
        myCreationsPanelEl = document.getElementById('my-products-grid');
        commissionSummaryEl = document.getElementById('commission-summary');
        statusTextEl = document.getElementById('creator-status-text');

        // Wire up close button
        overlayEl.addEventListener('click', function(e) {
            if (e.target === overlayEl || e.target.id === 'btn-close-creator') {
                state._visible = false;
                overlayEl.style.display = 'none';
            }
        });
    }

    // Wire event listeners for interactive elements
    function wireEvents() {
        const btnListProduct = document.getElementById('btn-list-product');
        if (btnListProduct) {
            btnListProduct.addEventListener('click', showCreateProductModal);
        }

        const btnConfirm = document.getElementById('btn-confirm-list');
        if (btnConfirm) {
            btnConfirm.addEventListener('click', handleCreateProduct);
        }

        const btnCancel = document.getElementById('btn-cancel-list');
        if (cancelBtnEl = document.getElementById('btn-cancel-list')) {
            btnCancel.addEventListener('click', hideCreateProductModal);
        }
    }

    // Show the create product modal overlay
    function showCreateProductModal() {
        const modal = document.getElementById('create-product-modal');
        if (modal) modal.style.display = 'block';
    }

    function hideCreateProductModal() {
        const modal = document.getElementById('create-product-modal');
        if (modal) modal.style.display = 'none';
    }

    // Handle product listing creation
    async function handleCreateProduct() {
        const nameEl = document.getElementById('cp-name');
        const descEl = document.getElementById('cp-desc');
        const catEl = document.getElementById('cp-category');
        const priceEl = document.getElementById('cp-price');
        const tagsEl = document.getElementById('cp-tags');

        if (!nameEl || !priceEl) return;

        const name = (nameEl.value || '').trim();
        const description = (descEl ? descEl.value : '').trim();
        const category = catEl ? catEl.value : 'asset';
        const priceStr = priceEl.value.trim();
        const tagsRaw = tagsEl ? (tagsEl.value || '') : '';

        if (!name) { setStatus('Product name is required.'); return; }
        if (!priceStr || parseInt(priceStr, 10) <= 0) { setStatus('Price must be greater than zero.'); return; }

        const price = parseInt(priceStr, 10);
        const tags = tagsRaw ? tagsRaw.split(',').map(function(t){return t.trim();}).filter(Boolean).slice(0,8) : [];

        // Generate product ID from wallet + timestamp
        const productId = 'prod_' + state.currentWallet.slice(-6) + '_' + Date.now();

        setStatus('Listing product...');

        try {
            const resp = await fetch(API_BASE + '/creator/list-product', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({
                    creator_wallet: state.currentWallet,
                    product_id: productId,
                    name: name,
                    description: description,
                    category: category,
                    price_micro_vbv: price,
                    tags: tags,
                    dlc_links: []
                })
            });

            if (!resp.ok) {
                const err = await resp.text();
                throw new Error(err || 'Failed to list product');
            }

            setStatus('Product listed successfully! Refreshing catalog...');
            hideCreateProductModal();
            // Clear inputs
            if (nameEl) nameEl.value = '';
            if (descEl) descEl.value = '';
            if (priceEl) priceEl.value = '';
            if (tagsEl) tagsEl.value = '';

            await fetchAllCreatorData();
        } catch(err) {
            setStatus('Error: ' + err.message);
        }
    }

    // Set status text in the bottom panel
    function setStatus(text) {
        if (statusTextEl) {
            statusTextEl.textContent = text;
            statusTextEl.style.color = text.indexOf('Error') === 0 ? '#ef5350' : '#80cbc4';
        }
    }

    // Export: toggle overlay visibility
    window.openCreatorStore = function() {
        if (!state._initialized) init();

        state._visible = !state._visible;
        if (overlayEl) {
            overlayEl.style.display = state._visible ? 'block' : 'none';
        }
        if (state._visible) {
            fetchAllCreatorData();
        }
    };

    // Fetch all creator store data on first open
    async function fetchAllCreatorData() {
        setStatus('Loading marketplace...');

        try {
            await Promise.all([
                fetchCreators(),
                fetchMyProducts(),
                fetchCommissionSummary()
            ]);
            setStatus('Marketplace loaded.');
        } catch(err) {
            setStatus('Error loading marketplace: ' + err.message);
        }
    }

    // Fetch list of all creators (top 20 by revenue)
    async function fetchCreators() {
        try {
            const resp = await fetch(API_BASE + '/creator/list-creators?limit=20');
            if (!resp.ok) throw new Error('Failed to fetch creators: ' + resp.status);

            // Handle both array response and object wrapper
            let data;
            const ct = resp.headers.get('content-type') || '';
            if (ct.includes('application/json')) {
                data = await resp.json();
            } else {
                throw new Error('Unexpected content type from /creator/list-creators');
            }

            // Normalize: backend may return array directly or wrapped in a field
            const creatorsList = Array.isArray(data) ? data : (data.creators || []);
            state.creators = creatorsList;
            renderCreatorsGrid(creatorsList);
        } catch(err) {
            console.error('CreatorStore: fetchCreators error:', err);
            // Non-fatal — grid stays empty, status shows "no data"
        }
    }

    // Render the creator cards grid (Panel 1)
    function renderCreatorsGrid(creators) {
        if (!creatorsGridEl) return;

        if (!creators || creators.length === 0) {
            creatorsGridEl.innerHTML = '<p style="color:#90a4ae;padding:20px;text-align:center;">No active creators found. Be the first to join!</p>';
            return;
        }

        let html = '';
        for (let i = 0; i < creators.length; i++) {
            const c = creators[i];
            const displayName = c.display_name || ('Creator_' + (c.wallet_address ? c.wallet_address.slice(0,8) : 'unknown'));
            const avgRating = c.avg_rating != null ? c.avg_rating.toFixed(2) : '0.00';
            const ratingCount = c.rating_count || 0;

            html += '<div class="creator-card" data-wallet="' + (c.wallet_address || '') + '" onclick="window._selectCreator(\'' + (c.wallet_address || '') + '\')">' +
                '<h3>' + escapeHtml(displayName) + '</h3>' +
                '<p style="color:#90a4ae;font-size:12px;">⭐ ' + avgRating + ' (' + ratingCount + ' ratings)</p>' +
                '<p style="font-size:12px;color:#b0bec5;">Products: ' + (c.product_count || 0) + '</p>' +
                '<p style="font-size:12px;color:#4dd0e1;">Revenue: ' + formatMicroVBV(c.total_revenue_micro_vbv || 0) + ' VBV</p>' +
            '</div>';
        }

        creatorsGridEl.innerHTML = html;
    }

    // Select a creator to view their products (Panel 2)
    window._selectCreator = async function(walletAddress) {
        state.selectedCreatorId = walletAddress;
        setStatus('Loading ' + walletAddress.slice(0,8) + '\'s catalog...');

        try {
            const [productsResp] = await Promise.all([
                fetch(API_BASE + '/creator/products?wallet=' + encodeURIComponent(walletAddress))
            ]);

            let productsData;
            if (productsResp.headers.get('content-type') && productsResp.headers.get('content-type').includes('application/json')) {
                productsData = await productsResp.json();
            } else {
                throw new Error('Unexpected response from /creator/products');
            }

            const productList = Array.isArray(productsData) ? productsData : (productsData.products || []);
            state.products = productList;
            renderProductCatalog(productList, walletAddress);

            // Fetch reviews for this creator's products
            try {
                const revResp = await fetch(API_BASE + '/creator/reviews?wallet=' + encodeURIComponent(walletAddress) + '&limit=10');
                if (revResp.ok) {
                    let revData = await revResp.json();
                    state.reviews = Array.isArray(revData) ? revData : (revData.reviews || []);
                }
            } catch(e) {/* reviews non-fatal */}

        } catch(err) {
            setStatus('Error loading catalog: ' + err.message);
        }
    };

    // Render product cards grid (Panel 2)
    function renderProductCatalog(products, creatorWallet) {
        if (!productCatalogEl) return;

        const displayName = state.creators.find(function(c){return c.wallet_address === creatorWallet;})?.display_name || ('Creator_' + (creatorWallet ? creatorWallet.slice(0,8) : 'unknown'));

        productCatalogEl.previousElementSibling && (productCatalogEl.previousElementSibling.textContent = displayName + '\'s Products');
        document.getElementById('product-subtitle') && (document.getElementById('product-subtitle').textContent = creatorWallet);

        if (!products || products.length === 0) {
            productCatalogEl.innerHTML = '<p style="color:#90a4ae;padding:20px;text-align:center;">No active products listed.</p>';
            return;
        }

        let html = '';
        for (let i = 0; i < products.length; i++) {
            const p = products[i];
            const priceFormatted = formatMicroVBV(p.price_micro_vbv || 0);
            const salesCount = p.sales_count || 0;
            const categoryIcon = getCategoryIcon(p.category);

            html += '<div class="product-card">' +
                '<h3>' + categoryIcon + ' ' + escapeHtml(p.name) + '</h3>' +
                '<p style="color:#90a4ae;font-size:12px;">' + escapeHtml(p.description ? p.description.slice(0, 80) : '') + (p.description && p.description.length > 80 ? '...' : '') + '</p>' +
                '<div class="product-meta">' +
                    '<span style="color:#4dd0e1;font-size:16px;">' + priceFormatted + ' VBV</span>' +
                    '<span style="font-size:12px;color:#b0bec5;margin-left:8px;">Sales: ' + salesCount + '</span>' +
                '</div>' +
                (p.tags && p.tags.length > 0 ? '<div class="product-tags">' + p.tags.map(function(t){return '<span class="tag">' + escapeHtml(t) + '</span>';}).join('') : '') +
                '<button class="vbt-btn vbt-btn-accent buy-product-btn" data-id="' + (p.id || '') + '" onclick="window._buyProduct(\'' + (p.id || '') + '\', ' + (p.price_micro_vbv || 0) + ')">Buy Now</button>' +
            '</div>';
        }

        productCatalogEl.innerHTML = html;
    }

    // Buy a product from the catalog
    window._buyProduct = async function(productId, priceMicroVBV) {
        if (!productId) return;
        setStatus('Purchasing ' + productId.slice(0,12) + '...');

        try {
            const resp = await fetch(API_BASE + '/creator/buy', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({
                    product_id: productId,
                    buyer_wallet: state.currentWallet || getActiveWallet()
                })
            });

            if (!resp.ok) {
                const err = await resp.text();
                throw new Error(err || 'Purchase failed');
            }

            setStatus('Product purchased successfully! Refreshing...');
            // Re-fetch products to update sales count
            if (state.selectedCreatorId) {
                window._selectCreator(state.selectedCreatorId);
            }
        } catch(err) {
            setStatus('Error: ' + err.message);
        }
    };

    // Fetch user's own listed products (Panel 3)
    async function fetchMyProducts() {
        if (!state.currentWallet && !getActiveWallet()) return;

        const wallet = state.currentWallet || getActiveWallet();
        try {
            const resp = await fetch(API_BASE + '/creator/my-products?wallet=' + encodeURIComponent(wallet));
            if (!resp.ok) throw new Error('Failed to fetch my products: ' + resp.status);

            let data;
            const ct = resp.headers.get('content-type') || '';
            if (ct.includes('application/json')) {
                data = await resp.json();
            } else {
                state.myProducts = [];
                renderMyCreations([]);
                return;
            }

            // Normalize: may be array or wrapped in object
            const myProds = Array.isArray(data) ? data : (data.products || []);
            state.myProducts = myProds;
            renderMyCreations(myProds);
        } catch(err) {
            console.error('CreatorStore: fetchMyProducts error:', err);
            state.myProducts = [];
            if (myCreationsPanelEl) myCreationsPanelEl.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No products listed yet. Click "+ List New Product" to get started.</p>';
        }
    }

    // Render "My Creations" grid (Panel 3)
    function renderMyCreations(products) {
        if (!myCreationsPanelEl) return;

        if (!products || products.length === 0) {
            myCreationsPanelEl.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No products listed yet. Click "+ List New Product" to get started.</p>';
            return;
        }

        let html = '';
        for (let i = 0; i < products.length; i++) {
            const p = products[i];
            const priceFormatted = formatMicroVBV(p.price_micro_vbv || 0);

            html += '<div class="product-card my-product">' +
                '<h3>' + escapeHtml(p.name) + '</h3>' +
                '<p style="color:#90a4ae;font-size:12px;">' + (p.category ? getCategoryIcon(p.category) : '') + ' ' + p.id + '</p>' +
                '<div class="product-meta">' +
                    '<span style="color:#4dd0e1;">' + priceFormatted + ' VBV</span>' +
                    '<span style="font-size:12px;color:#b0bec5;margin-left:8px;">Sales: ' + (p.sales_count || 0) + '</span>' +
                '</div>' +
            '</div>';
        }

        myCreationsPanelEl.innerHTML = html;
    }

    // Fetch commission/royalty summary for the active wallet
    async function fetchCommissionSummary() {
        if (!state.currentWallet && !getActiveWallet()) return;

        const wallet = state.currentWallet || getActiveWallet();
        try {
            const resp = await fetch(API_BASE + '/creator/commission-summary?wallet=' + encodeURIComponent(wallet));
            if (!resp.ok) throw new Error('Failed to fetch commission summary: ' + resp.status);

            let data;
            const ct = resp.headers.get('content-type') || '';
            if (ct.includes('application/json')) {
                data = await resp.json();
            } else {
                renderCommissionSummary({totalRevenueMicroVBV: 0, totalRoyaltiesEarnedMicroVBV: 0});
                return;
            }

            // Normalize response — backend may wrap in object or return flat fields
            const summary = data.summary || data || {};
            state.commissionTotalMicroVBV = summary.total_revenue_micro_vbv || summary.totalRevenueMicroVBV || 0;
            renderCommissionSummary(summary);

        } catch(err) {
            console.error('CreatorStore: fetchCommissionSummary error:', err);
            // Non-fatal — show zeroed summary
            state.commissionTotalMicroVBV = 0;
            if (commissionSummaryEl) commissionSummaryEl.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No earnings yet.</p>';
        }
    }

    // Render the commission/earnings summary grid (Panel 4 top)
    function renderCommissionSummary(summary) {
        if (!commissionSummaryEl) return;

        const totalRev = formatMicroVBV(summary.total_revenue_micro_vbv || summary.totalRevenueMicroVBV || state.commissionTotalMicroVBV);
        const royaltiesEarned = formatMicroVBV(summary.total_royalties_earned_micro_vbv || 0);
        const productCount = summary.product_count || (state.myProducts ? state.myProducts.length : 0);

        commissionSummaryEl.innerHTML = '<div class="commission-grid">' +
            '<div class="comm-item"><span style="color:#90a4ae;font-size:12px;">Total Revenue</span><br/><strong style="color:#ffd54f;font-size:18px;">' + totalRev + ' VBV</strong></div>' +
            '<div class="comm-item"><span style="color:#90a4ae;font-size:12px;">Royalties Earned</span><br/><strong style="color:#ce93d8;font-size:16px;">' + royaltiesEarned + ' VBV</strong></div>' +
            '<div class="comm-item"><span style="color:#90a4ae;font-size:12px;">Products Listed</span><br/><strong style="color:#4dd0e1;font-size:16px;">' + productCount + '</strong></div>' +
        '</div>';
    }

    // Helper: format micro-VBV to human-readable VBV (divide by 1,000,000)
    function formatMicroVBV(microAmount) {
        if (!microAmount && microAmount !== 0) return '0';
        const v = Number(microAmount);
        if (v >= 1e9) return (v / 1e6).toFixed(2) + 'M';
        if (v >= 1e6) return (v / 1e6).toFixed(2);
        return (v / 1e6).toFixed(4);
    }

    // Helper: get category icon mapping
    function getCategoryIcon(category) {
        switch(category) {
            case 'asset': return '\uD83D\uDCE5';   // 📥
            case 'dlc': return '\uD83C\uDFAE';     // 🎮
            case 'service': return '\u2699\uFE0F';  // ⚙️
            case 'cosmetic': return '\uD83D\uDCA5'; // 💥 (fallback)
            default: return '\uD83D\uDCE6';          // 📦
        }
    }

    // Helper: escape HTML to prevent XSS in product names/descriptions
    function escapeHtml(str) {
        if (!str && str !== 0) return '';
        const div = document.createElement('div');
        div.appendChild(document.createTextNode(String(str)));
        return div.innerHTML;
    }

    // Helper: get active wallet from global state (if available via network.js or localStorage)
    function getActiveWallet() {
        if (typeof window !== 'undefined' && window._activeWallet) return window._activeWallet;
        try {
            const saved = localStorage.getItem('active_wallet');
            if (saved) return saved;
        } catch(e) {/* ignore */}
        return '';
    }

    // Export public API for network.js WS integration and button trigger
    window.CreatorStoreModule = {
        init: init,
        open: function() { window.openCreatorStore(); },
        getState: function() { return state; },
        setState: function(s) { Object.assign(state, s); }
    };

})();