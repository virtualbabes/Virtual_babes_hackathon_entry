// Public/js/creator_storefront.js

/**
 * CreatorStorefront Module
 * PILLAR 7-C: Frontend UI for Creator Economy
 */
const CreatorStorefront = (function() {
    let wsConnection = null;
    let currentWallet = null;

    // DOM Elements
    let container = null;
    let grid = null;
    let detailView = null;

    function init(wallet, ws) {
        currentWallet = wallet;
        wsConnection = ws;
        
        // Find or create container
        container = document.getElementById('creator-store-container');
        if (!container) {
            console.error("[CreatorStore] Container #creator-store-container not found in index.html");
            return;
        }

        grid = document.getElementById('creator-store-grid');
        detailView = document.getElementById('creator-store-detail');

        // Bind UI events
        const closeBtn = document.getElementById('creator-store-close');
        if (closeBtn) {
            closeBtn.addEventListener('click', closeStore);
        }

        const backBtn = document.getElementById('creator-detail-back');
        if (backBtn) {
            backBtn.addEventListener('click', showGrid);
        }
    }

    function openStore() {
        if (container) {
            container.style.display = 'flex';
            fetchProducts();
        }
    }

    function closeStore() {
        if (container) {
            container.style.display = 'none';
        }
    }

    function showGrid() {
        if (grid && detailView) {
            grid.style.display = 'grid';
            detailView.style.display = 'none';
        }
    }

    async function fetchProducts() {
        try {
            const response = await fetch('/api/creator/store/products');
            if (!response.ok) throw new Error('Network response was not ok');
            const products = await response.json();
            renderGrid(products);
        } catch (error) {
            console.error("[CreatorStore] Failed to fetch products:", error);
            if (grid) grid.innerHTML = '<div class="error-msg">Failed to load marketplace.</div>';
        }
    }

    function renderGrid(products) {
        if (!grid) return;
        
        if (!products || products.length === 0) {
            grid.innerHTML = '<div class="empty-msg">No products available.</div>';
            return;
        }

        grid.innerHTML = ''; // Clear
        
        products.forEach(p => {
            const card = document.createElement('div');
            card.className = 'creator-product-card glass-panel';
            card.innerHTML = `
                <div class="product-type">${escapeHTML(p.product_type)}</div>
                <h3 class="product-title">${escapeHTML(p.metadata?.name || 'Unknown')}</h3>
                <div class="product-creator">By: <span class="creator-address" data-wallet="${p.creator_wallet}">${formatWallet(p.creator_wallet)}</span></div>
                <div class="product-price">${p.price_micro} VBV</div>
                <div class="product-stats">★ ${p.rating.toFixed(1)} (${p.total_ratings})</div>
                <button class="action-btn view-btn" data-id="${p.id}">View Details</button>
            `;
            
            card.querySelector('.view-btn').addEventListener('click', () => fetchProductDetails(p.id));
            card.querySelector('.creator-address').addEventListener('click', (e) => {
                e.stopPropagation();
                fetchCreatorProfile(p.creator_wallet);
            });

            grid.appendChild(card);
        });
    }

    async function fetchProductDetails(id) {
        try {
            const response = await fetch(`/api/creator/store/product?id=${id}`);
            if (!response.ok) throw new Error('Failed to fetch details');
            const p = await response.json();
            renderDetailView(p);
        } catch (error) {
            console.error("[CreatorStore] Failed detail fetch:", error);
        }
    }

    function renderDetailView(p) {
        if (!grid || !detailView) return;
        
        grid.style.display = 'none';
        detailView.style.display = 'block';

        const content = detailView.querySelector('.detail-content');
        if (!content) return;

        content.innerHTML = `
            <h2>${escapeHTML(p.metadata?.name || 'Product Details')}</h2>
            <div class="detail-creator" data-wallet="${p.creator_wallet}">Creator: ${formatWallet(p.creator_wallet)}</div>
            <div class="detail-desc">${escapeHTML(p.metadata.description || 'No description provided.')}</div>
            <div class="detail-meta">
                <span>Price: ${p.price_micro} VBV</span>
                <span>Type: ${escapeHTML(p.product_type)}</span>
                <span>Rating: ★ ${p.rating.toFixed(1)}</span>
            </div>
            <div class="detail-actions">
                <button id="btn-buy-product" class="action-btn buy-btn" data-id="${p.id}">Purchase</button>
                <button id="btn-rate-product" class="action-btn rate-btn" data-id="${p.id}">Rate</button>
            </div>
        `;

        document.getElementById('btn-buy-product').addEventListener('click', () => buyProduct(p.id));
        document.getElementById('btn-rate-product').addEventListener('click', () => rateProduct(p.id));
    }

    async function fetchCreatorProfile(wallet) {
         try {
            const response = await fetch(`/api/creator/store/creator?wallet=${wallet}`);
            if (!response.ok) throw new Error('Failed to fetch profile');
            const profile = await response.json();
            // In a full implementation, this would render a dedicated profile view.
            // For now, we alert the core stats.
            alert(`Creator: ${profile.creator_name || formatWallet(profile.wallet)}\nTotal Sales: ${profile.total_sales}\nAvg Rating: ★ ${profile.average_rating.toFixed(1)}`);
        } catch (error) {
            console.error("[CreatorStore] Failed profile fetch:", error);
        }
    }

    async function buyProduct(id) {
        if (!currentWallet) {
            alert("Wallet not connected.");
            return;
        }
        
        try {
            const response = await fetch('/api/creator/store/purchase/', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Wallet-Address': currentWallet
                },
                body: JSON.stringify({ product_id: id })
            });
            
            if (!response.ok) {
                const text = await response.text();
                throw new Error(text);
            }
            
            alert("Purchase successful!");
            fetchProducts(); // Refresh
            showGrid();
        } catch (error) {
            alert(`Purchase failed: ${error.message}`);
        }
    }

    async function rateProduct(id) {
        if (!currentWallet) {
            alert("Wallet not connected.");
            return;
        }

        const score = prompt("Enter rating (1-5):", "5");
        const s = parseInt(score, 10);
        if (isNaN(s) || s < 1 || s > 5) {
            alert("Invalid rating.");
            return;
        }

        try {
            const response = await fetch('/api/creator/store/rate', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Wallet-Address': currentWallet
                },
                body: JSON.stringify({ product_id: id, score: s })
            });
            
            if (!response.ok) {
                const text = await response.text();
                throw new Error(text);
            }
            
            alert("Rating submitted!");
            fetchProductDetails(id); // Refresh details
        } catch (error) {
            alert(`Rating failed: ${error.message}`);
        }
    }

    // Utilities
    function escapeHTML(str) {
        if (!str) return '';
        return str.replace(/[&<>'"]/g, 
            tag => ({
                '&': '&amp;',
                '<': '&lt;',
                '>': '&gt;',
                "'": '&#39;',
                '"': '&quot;'
            }[tag] || tag)
        );
    }

    function formatWallet(wallet) {
            if (!wallet || (wallet?.length ?? 0) < 8) return wallet;
            return `${wallet.substring(0, 4)}...${wallet.substring((wallet?.length ?? 0) - 4)}`;
        }

    return {
        init,
        openStore,
        closeStore
    };
})();

// Export for global access if using ES modules, otherwise binds to window
if (typeof window !== 'undefined') {
    window.CreatorStorefront = CreatorStorefront;
}
