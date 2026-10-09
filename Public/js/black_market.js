// ============================================================================
// black_market.js — Criminal Trading & Fence
// ----------------------------------------------------------------------------
// Exposes black_market_service.go: buy/sell stolen, fence goods, Wanted gate
// Unique visual style: neon-noir criminal bazaar with glitch effects
// ============================================================================

var API_BASE = '/api';

// --- State ---
let marketItems = [];
let fenceGoods = [];
let wantedLevel = 0;

// --- Init ---
export function initBlackMarket() {
    const el = document.getElementById('wd-blackmarket');
    if (!el) return;
    renderBmLoading(el);
    fetchBmData(el);
}

async function fetchBmData(el) {
    try {
        const [bmResp, fenceResp] = await Promise.all([
            fetch(`${API_BASE}/black-market/buy`).catch(() => ({ json: () => ({ items: [] }) })),
            fetch(`${API_BASE}/black-market/fence-goods`).catch(() => ({ json: () => ({ items: [] }) })),
        ]);
        marketItems = (await bmResp.json()).items || [];
        fenceGoods = (await fenceResp.json()).items || [];
        renderBlackMarket(el);
    } catch (e) {
        el.innerHTML = '<p style="color:#90a4ae;">Black market data unavailable</p>';
    }
}

function renderBmLoading(el) {
    el.innerHTML = '<div class="wd-loading">Entering bazaar…</div>';
}

function renderBlackMarket(el) {
    let html = `
        <div class="blackmarket-container">
            <!-- Neon Header -->
            <div class="bm-header">
                <div class="bm-neon-sign">
                    <span class="bm-neon-text">BLACK MARKET</span>
                    <span class="bm-neon-flicker"></span>
                </div>
                <div class="bm-wanted">
                    <span class="bm-wanted-label">WANTED LEVEL</span>
                    <span class="bm-wanted-value" style="color:${wantedLevel >= 10 ? '#f44336' : wantedLevel >= 5 ? '#ff9800' : '#4caf50'}">${wantedLevel}</span>
                </div>
            </div>

            <!-- Gate Warning -->
            ${wantedLevel < 3 ? `
                <div class="bm-gate-warning">
                    <span class="bm-gate-icon">🔒</span>
                    <span class="bm-gate-text">Higher Wanted Level unlocks more goods</span>
                </div>
            ` : ''}

            <!-- Market Tabs -->
            <div class="bm-tabs">
                <button class="bm-tab active" data-tab="buy">🛒 Buy</button>
                <button class="bm-tab" data-tab="sell">💰 Sell</button>
                <button class="bm-tab" data-tab="fence">🏚️ Fence</button>
            </div>

            <!-- Buy Panel -->
            <div class="bm-panel" data-panel="buy">
                <div class="bm-goods-grid">
                    ${marketItems.length > 0 ? marketItems.map(i => renderBmItem(i)).join('') : renderSampleBmGoods()}
                </div>
            </div>

            <!-- Sell Panel -->
            <div class="bm-panel hidden" data-panel="sell">
                <div class="bm-sell-section">
                    <p style="color:#90a4ae;font-size:11px;">Sell your illicit goods for $VBV</p>
                    <div class="bm-goods-grid">
                        ${fenceGoods.map(i => renderBmItem(i, 'sell')).join('') || '<p style="color:#607d8b;">No goods to sell</p>'}
                    </div>
                </div>
            </div>

            <!-- Fence Panel -->
            <div class="bm-panel hidden" data-panel="fence">
                <div class="bm-fence-section">
                    <p style="color:#90a4ae;font-size:11px;">Fence stolen goods (5% commission)</p>
                    <div class="bm-goods-grid">
                        ${fenceGoods.map(i => renderBmItem(i, 'fence')).join('') || '<p style="color:#607d8b;">No goods to fence</p>'}
                    </div>
                </div>
            </div>
        </div>
    `;

    el.innerHTML = html;
    attachBmListeners(el);
}

function renderSampleBmGoods() {
    const samples = [
        { name: 'Stolen Card #4521', price: 1500, rarity: 'rare' },
        { name: 'Sabotage Plans', price: 3000, rarity: 'epic' },
        { name: 'Ghost Protocol', price: 5000, rarity: 'legendary' },
    ];
    return samples.map(i => renderBmItem(i)).join('');
}

function renderBmItem(item, action = 'buy') {
    const rarityColors = { common: '#9e9e9e', rare: '#2196f3', epic: '#9c27b0', legendary: '#ff9800' };
    const color = rarityColors[item.rarity] || '#9e9e9e';

    return `
        <div class="bm-item-card" data-item="${item.id || item.name}" style="--item-color:${color}">
            <div class="bm-item-shine"></div>
            <div class="bm-item-header">
                <span class="bm-item-name">${escapeHtml(item.name)}</span>
                <span class="bm-item-rarity" style="color:${color}">${(item.rarity || 'common').toUpperCase()}</span>
            </div>
            <div class="bm-item-price">${formatVBV((item.price_micro || item.price * 1000000))} $VBV</div>
            <button class="vbt-btn vbt-btn-primary bm-action-btn" data-action="${action}">
                ${action === 'buy' ? '🛒 Buy' : action === 'sell' ? '💰 Sell' : '🏚️ Fence'}
            </button>
        </div>
    `;
}

function attachBmListeners(el) {
    // Tab switching
    el.querySelectorAll('.bm-tab').forEach(tab => {
        tab.addEventListener('click', () => {
            el.querySelectorAll('.bm-tab').forEach(t => t.classList.remove('active'));
            tab.classList.add('active');
            el.querySelectorAll('.bm-panel').forEach(p => p.classList.add('hidden'));
            const panel = el.querySelector(`[data-panel="${tab.dataset.tab}"]`);
            if (panel) panel.classList.remove('hidden');
        });
    });

    // Action buttons
    el.querySelectorAll('.bm-action-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const card = btn.closest('.bm-item-card');
            const itemName = card?.dataset.item;
            const action = btn.dataset.action;
            if (window.showToast) window.showToast(`${action === 'buy' ? '🛒' : action === 'sell' ? '💰' : '🏚️'} ${action}: ${itemName}`, 'info');
        });
    });
}

// --- Global handlers ---
window.initBlackMarket = initBlackMarket;

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
    return vbv.toFixed(2);
}
