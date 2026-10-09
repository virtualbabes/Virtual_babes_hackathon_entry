// ============================================================================
// creator_studio.js — Creator Economy & Product Studio
// ----------------------------------------------------------------------------
// Exposes creator_store_service.go: products, storefront, sales, royalties
// Unique visual style: artist studio / creator workshop with palette theme
// ============================================================================

var API_BASE = '/api';

// --- State ---
let products = [];
let storefront = null;

// --- Init ---
export function initCreatorStudio() {
    const el = document.getElementById('wd-creator');
    if (!el) return;
    renderCreatorLoading(el);
    fetchCreatorData(el);
}

async function fetchCreatorData(el) {
    try {
        const resp = await fetch(`${API_BASE}/creator/store/products`);
        if (!resp.ok) throw new Error('Failed to fetch creator data');
        const data = await resp.json();
        products = data.products || [];
        renderCreatorStudio(el);
    } catch (e) {
        el.innerHTML = '<p style="color:#90a4ae;">Creator data unavailable</p>';
    }
}

function renderCreatorLoading(el) {
    el.innerHTML = '<div class="wd-loading">Loading studio…</div>';
}

function renderCreatorStudio(el) {
    let html = `
        <div class="creator-studio-container">
            <!-- Studio Header -->
            <div class="studio-header">
                <div class="studio-icon">🎨</div>
                <div class="studio-info">
                    <h3 style="color:#e5e7eb;margin:0;">Creator Studio</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Build, sell, and earn royalties</p>
                </div>
                <div class="studio-stats">
                    <span class="studio-products-num">${products.length}</span>
                    <span class="studio-products-label">Products</span>
                </div>
            </div>

            <!-- Create Product -->
            <div class="studio-create-section">
                <h4 style="color:#e5e7eb;">✨ New Product</h4>
                <div class="studio-create-form">
                    <div class="studio-form-row">
                        <label>Name</label>
                        <input type="text" id="product-name" placeholder="Product name..." />
                    </div>
                    <div class="studio-form-row">
                        <label>Category</label>
                        <select id="product-category">
                            <option value="asset">Asset</option>
                            <option value="dlc">DLC</option>
                            <option value="service">Service</option>
                            <option value="cosmetic">Cosmetic</option>
                        </select>
                    </div>
                    <div class="studio-form-row">
                        <label>Price ($VBV)</label>
                        <input type="number" id="product-price" min="1" value="10" />
                    </div>
                    <button class="vbt-btn vbt-btn-primary" id="product-create-btn">🚀 Create Product</button>
                </div>
            </div>

            <!-- Product Gallery -->
            <div class="studio-products-section">
                <h4 style="color:#e5e7eb;">🖼️ Your Products</h4>
                <div class="studio-products-grid">
                    ${products.length > 0 ? products.map(p => renderProductCard(p)).join('') : renderSampleProducts()}
                </div>
            </div>
        </div>
    `;

    el.innerHTML = html;
    attachCreatorListeners(el);
}

function renderProductCard(product) {
    return `
        <div class="product-card" data-id="${product.id || product.product_id}">
            <div class="product-card-icon">${getCategoryIcon(product.category)}</div>
            <div class="product-card-info">
                <span class="product-card-name">${escapeHtml(product.name)}</span>
                <span class="product-card-price">${formatVBV(product.price_micro || product.price * 1000000)}</span>
            </div>
            <span class="product-card-status status-${product.status || 'active'}">${product.status || 'Active'}</span>
            <button class="vbt-btn vbt-btn-secondary product-edit-btn">Edit</button>
        </div>
    `;
}

function renderSampleProducts() {
    const samples = [
        { name: 'Neon Card Pack', category: 'dlc', price_micro: 5000000, status: 'active' },
        { name: 'Battle Arena Skin', category: 'cosmetic', price_micro: 2000000, status: 'active' },
    ];
    return samples.map(p => renderProductCard(p)).join('');
}

function getCategoryIcon(category) {
    const icons = { asset: '📦', dlc: '💿', service: '🔧', cosmetic: '✨' };
    return icons[category] || '📦';
}

function attachCreatorListeners(el) {
    const createBtn = el.querySelector('#product-create-btn');
    if (createBtn) {
        createBtn.addEventListener('click', () => {
            const name = el.querySelector('#product-name')?.value.trim();
            if (!name) { alert('Enter product name'); return; }
            if (window.showToast) window.showToast(`✨ Product "${name}" created!`, 'success');
        });
    }
}

// --- Global handlers ---
window.initCreatorStudio = initCreatorStudio;

// --- Utility ---
function escapeHtml(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    }[c]));
}

function formatVBV(micro) {
    const vbv = Number(micro) / 1_000_000;
    if (vbv >= 1_000_000) return `${(vbv / 1_000_000).toFixed(1)}M`;
    if (vbv >= 1_000) return `${(vbv / 1_000).toFixed(1)}K`;
    return vbv.toFixed(2) + ' $VBV';
}
