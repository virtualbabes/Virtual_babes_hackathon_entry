// ============================================================================
// amm_chart.js — AMM Bonding Curve Visualization for Markets Tab
// ----------------------------------------------------------------------------
// Exposes entity_market.go: AMM curve, buy/sell, price history, slippage
// Unique visual style: trading terminal / financial chart theme
// ============================================================================

var API_BASE = '/api';

// --- State ---
let marketData = null;
let selectedEntity = null;
let priceHistory = [];

// --- Init ---
export function initAmmChart() {
    const el = document.getElementById('wd-markets');
    if (!el) return;
    renderMarketLoading(el);
    fetchMarketData(el);
}

async function fetchMarketData(el) {
    try {
        const [listResp, auctionsResp, bmResp] = await Promise.all([
            fetch(`${API_BASE}/entity/market/list`).catch(() => ({ json: () => ({ listings: [] }) })),
            fetch(`${API_BASE}/auctions`).catch(() => ({ json: () => ({ auctions: [] }) })),
            fetch(`${API_BASE}/black-market/buy`).catch(() => ({ json: () => ({ items: [] }) })),
        ]);
        marketData = {
            listings: (await listResp.json()).listings || [],
            auctions: (await auctionsResp.json()).auctions || [],
            blackMarket: (await bmResp.json()).items || [],
        };
        renderAmmChart(el);
    } catch (e) {
        el.innerHTML = '<p style="color:#90a4ae;">Market data unavailable</p>';
    }
}

function renderMarketLoading(el) {
    el.innerHTML = '<div class="wd-loading">Loading market data…</div>';
}

function renderAmmChart(el) {
    const listings = marketData?.listings || [];
    const auctions = marketData?.auctions || [];
    const bm = marketData?.blackMarket || [];

    let html = `
        <div class="amm-chart-container">
            <!-- Header -->
            <div class="amm-header">
                <h3 style="color:#e5e7eb;margin:0;">📈 Entity Markets</h3>
                <span class="amm-market-status market-open">● OPEN</span>
            </div>

            <!-- Market Tabs -->
            <div class="amm-market-tabs">
                <button class="amm-market-tab active" data-tab="entity">Entity Shares</button>
                <button class="amm-market-tab" data-tab="auctions">Auctions</button>
                <button class="amm-market-tab" data-tab="blackmarket">Black Market</button>
            </div>

            <!-- Entity Market Panel -->
            <div class="amm-panel" data-panel="entity">
                <!-- AMM Curve Visualization -->
                <div class="amm-curve-section">
                    <h4 style="color:#e5e7eb;">📊 Bonding Curve</h4>
                    <div class="amm-curve-container">
                        <div class="amm-curve-canvas" id="amm-curve-canvas">
                            <svg class="amm-curve-svg" viewBox="0 0 300 150" preserveAspectRatio="none">
                                <!-- Grid lines -->
                                <line x1="0" y1="30" x2="300" y2="30" stroke="rgba(255,255,255,0.05)" stroke-width="1"/>
                                <line x1="0" y1="60" x2="300" y2="60" stroke="rgba(255,255,255,0.05)" stroke-width="1"/>
                                <line x1="0" y1="90" x2="300" y2="90" stroke="rgba(255,255,255,0.05)" stroke-width="1"/>
                                <line x1="0" y1="120" x2="300" y2="120" stroke="rgba(255,255,255,0.05)" stroke-width="1"/>
                                <!-- AMM Curve (quadratic) -->
                                <path d="M 0 150 Q 150 100 300 0" fill="none" stroke="#00bcd4" stroke-width="2"/>
                                <!-- Current price marker -->
                                <circle cx="150" cy="75" r="4" fill="#ff9800"/>
                                <text x="160" y="78" fill="#ff9800" font-size="8">Current</text>
                            </svg>
                        </div>
                        <div class="amm-curve-labels">
                            <span class="amm-curve-label">Price ↑</span>
                            <span class="amm-curve-label">Quantity →</span>
                        </div>
                    </div>
                </div>

                <!-- Entity Listings -->
                <div class="amm-listings-section">
                    <h4 style="color:#e5e7eb;">🏷️ Active Listings</h4>
                    <div class="amm-listings-grid">
                        ${listings.length > 0 ? listings.slice(0, 6).map(l => renderListingCard(l)).join('') : renderEmptyListings()}
                    </div>
                </div>

                <!-- Trade Panel -->
                <div class="amm-trade-section">
                    <h4 style="color:#e5e7eb;">💱 Trade</h4>
                    <div class="amm-trade-panel">
                        <div class="amm-trade-row">
                            <label>Entity</label>
                            <select id="amm-trade-entity">
                                ${listings.map(l => `<option value="${l.listing_id || l.id}">${l.name || 'Unknown'}</option>`).join('')}
                            </select>
                        </div>
                        <div class="amm-trade-row">
                            <label>Amount (shares)</label>
                            <input type="number" id="amm-trade-amount" min="1" value="1" />
                        </div>
                        <div class="amm-trade-row">
                            <label>Slippage Tolerance</label>
                            <select id="amm-trade-slippage">
                                <option value="0.5">0.5%</option>
                                <option value="1">1%</option>
                                <option value="2">2%</option>
                                <option value="5">5%</option>
                            </select>
                        </div>
                        <div class="amm-trade-summary">
                            <div class="amm-trade-item">
                                <span>Price per share:</span>
                                <span id="amm-trade-price">--</span>
                            </div>
                            <div class="amm-trade-item">
                                <span>Total cost:</span>
                                <span id="amm-trade-total">--</span>
                            </div>
                            <div class="amm-trade-item">
                                <span>Est. slippage:</span>
                                <span id="amm-trade-slippage-est">--</span>
                            </div>
                        </div>
                        <div class="amm-trade-actions">
                            <button class="vbt-btn vbt-btn-primary" id="amm-buy-btn">📈 Buy</button>
                            <button class="vbt-btn vbt-btn-secondary" id="amm-sell-btn">📉 Sell</button>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Auctions Panel -->
            <div class="amm-panel hidden" data-panel="auctions">
                <div class="amm-auctions-grid">
                    ${auctions.length > 0 ? auctions.slice(0, 4).map(a => renderAuctionCard(a)).join('') : renderEmptyAuctions()}
                </div>
            </div>

            <!-- Black Market Panel -->
            <div class="amm-panel hidden" data-panel="blackmarket">
                <div class="amm-bm-grid">
                    ${bm.length > 0 ? bm.slice(0, 4).map(i => renderBmCard(i)).join('') : renderEmptyBm()}
                </div>
            </div>
        </div>
    `;

    el.innerHTML = html;
    attachAmmListeners(el);
}

function renderListingCard(listing) {
    return `
        <div class="amm-listing-card" data-id="${listing.listing_id || listing.id}">
            <div class="amm-listing-header">
                <span class="amm-listing-name">${escapeHtml(listing.name || 'Unknown')}</span>
                <span class="amm-listing-price">${formatVBV(listing.price_micro || 0)}</span>
            </div>
            <div class="amm-listing-details">
                <span class="amm-listing-detail">📦 Supply: ${listing.supply || '∞'}</span>
                <span class="amm-listing-detail">📊 Reserve: ${formatVBV(listing.reserve_micro || 0)}</span>
            </div>
            <button class="vbt-btn vbt-btn-primary amm-listing-buy">Buy</button>
        </div>
    `;
}

function renderEmptyListings() {
    return '<div class="amm-empty"><span class="amm-empty-icon">📭</span><p>No active listings</p></div>';
}

function renderAuctionCard(auction) {
    return `
        <div class="amm-auction-card">
            <div class="amm-auction-header">
                <span class="amm-auction-name">${escapeHtml(auction.bundle?.card_id || 'Bundle')}</span>
                <span class="amm-auction-bid">${formatVBV(auction.current_bid || 0)}</span>
            </div>
            <div class="amm-auction-time">⏱ ${auction.ends_at ? new Date(auction.ends_at).toLocaleString() : 'No expiry'}</div>
            <button class="vbt-btn vbt-btn-primary amm-auction-bid">Place Bid</button>
        </div>
    `;
}

function renderEmptyAuctions() {
    return '<div class="amm-empty"><span class="amm-empty-icon">🔨</span><p>No active auctions</p></div>';
}

function renderBmCard(item) {
    return `
        <div class="amm-bm-card">
            <div class="amm-bm-header">
                <span class="amm-bm-name">${escapeHtml(item.name || 'Unknown')}</span>
                <span class="amm-bm-price">${formatVBV(item.price_micro || 0)}</span>
            </div>
            <div class="amm-bm-badge">🔒 BLACK MARKET</div>
            <button class="vbt-btn vbt-btn-secondary amm-bm-buy">Purchase</button>
        </div>
    `;
}

function renderEmptyBm() {
    return '<div class="amm-empty"><span class="amm-empty-icon">🔒</span><p>Black market access requires Wanted Level</p></div>';
}

function attachAmmListeners(el) {
    // Market tab switching
    el.querySelectorAll('.amm-market-tab').forEach(tab => {
        tab.addEventListener('click', () => {
            el.querySelectorAll('.amm-market-tab').forEach(t => t.classList.remove('active'));
            tab.classList.add('active');
            const tabName = tab.dataset.tab;
            el.querySelectorAll('.amm-panel').forEach(p => {
                p.classList.toggle('hidden', p.dataset.panel !== tabName);
            });
        });
    });

    // Buy button
    const buyBtn = el.querySelector('#amm-buy-btn');
    if (buyBtn) {
        buyBtn.addEventListener('click', () => {
            const entityId = el.querySelector('#amm-trade-entity')?.value;
            const amount = el.querySelector('#amm-trade-amount')?.value;
            executeTrade('buy', entityId, amount);
        });
    }

    // Sell button
    const sellBtn = el.querySelector('#amm-sell-btn');
    if (sellBtn) {
        sellBtn.addEventListener('click', () => {
            const entityId = el.querySelector('#amm-trade-entity')?.value;
            const amount = el.querySelector('#amm-trade-amount')?.value;
            executeTrade('sell', entityId, amount);
        });
    }

    // Listing buy buttons
    el.querySelectorAll('.amm-listing-buy').forEach(btn => {
        btn.addEventListener('click', () => {
            const card = btn.closest('.amm-listing-card');
            const id = card?.dataset.id;
            if (window.showToast) window.showToast(`Purchasing listing ${id}...`, 'info');
        });
    });
}

async function executeTrade(side, entityId, amount) {
    if (!entityId || !amount) {
        alert('Please select an entity and amount');
        return;
    }
    try {
        const resp = await fetch(`${API_BASE}/entity/market/purchase`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ listing_id: entityId, amount: parseInt(amount) })
        });
        if (!resp.ok) throw new Error('Trade failed');
        if (window.showToast) window.showToast(`✅ ${side === 'buy' ? 'Purchase' : 'Sale'} executed!`, 'success');
    } catch (e) {
        if (window.showToast) window.showToast(`❌ ${e.message}`, 'error');
    }
}

// --- Global handlers ---
window.initAmmChart = initAmmChart;

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
