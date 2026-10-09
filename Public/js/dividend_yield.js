// ============================================================================
// dividend_yield.js — Dividend Yield Dashboard
// ----------------------------------------------------------------------------
// Exposes entity_investment_service.go: yield, compound, portfolio
// Unique visual style: growth garden / orchard with blooming investments
// ============================================================================

var API_BASE = '/api';

// --- State ---
let portfolioData = [];
let totalYield = 0;
let autoCompound = true;

// --- Init ---
export function initDividendYield() {
    const el = document.getElementById('wd-dividends');
    if (!el) return;
    renderDividendLoading(el);
    fetchDividendData(el);
}

function activeWallet() {
    if (window.currentWallet) return window.currentWallet;
    if (typeof window.getActiveWallet === 'function') {
        const w = window.getActiveWallet();
        if (w) return w;
    }
    // Authoritative WASM engine state (main.go: GameState.Wallet, json:"wallet").
    if (typeof window.GetGameState === 'function') {
        const gs = window.GetGameState() || {};
        if (gs.wallet) return gs.wallet;
    }
    return window.userAddress || '';
}

async function fetchDividendData(el) {
    const wallet = activeWallet();
    if (!wallet) {
        el.innerHTML = '<p style="color:#90a4ae;">Connect your wallet to view your yield garden.</p>';
        return;
    }
    try {
        const resp = await fetch(`${API_BASE}/invest/portfolio?wallet=${encodeURIComponent(wallet)}`);
        if (!resp.ok) throw new Error('Failed to fetch portfolio');
        const data = await resp.json();
        // EntityPortfolio shape: { wallet_address, investments: { entity_id: record }, total_claimed_micro }
        const inv = data.investments || {};
        portfolioData = Object.keys(inv).map((id) => {
            const rec = inv[id] || {};
            const amountMicro = Number(rec.amount_micro || 0);
            return { entity_id: id, name: id, invested: amountMicro, value: amountMicro };
        });
        totalYield = data.total_claimed_micro || 0;
        renderDividendYield(el);
    } catch (e) {
        el.innerHTML = '<p style="color:#90a4ae;">Portfolio data unavailable</p>';
    }
}

function renderDividendLoading(el) {
    el.innerHTML = '<div class="wd-loading">Loading yield garden…</div>';
}

function renderDividendYield(el) {
    const totalValue = portfolioData.reduce((sum, p) => sum + (p.value || p.current_value || 0), 0);
    const totalInvested = portfolioData.reduce((sum, p) => sum + (p.invested || p.amount || 0), 0);

    let html = `
        <div class="dividend-container">
            <!-- Garden Header -->
            <div class="garden-header">
                <div class="garden-icon">🌱</div>
                <div class="garden-info">
                    <h3 style="color:#e5e7eb;margin:0;">Yield Garden</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Watch your investments bloom</p>
                </div>
                <div class="garden-total">
                    <span class="garden-total-label">Total Value</span>
                    <span class="garden-total-value">${formatVBV(totalValue)}</span>
                </div>
            </div>

            <!-- Bloom Stats -->
            <div class="bloom-stats">
                <div class="bloom-stat">
                    <span class="bloom-icon">🌻</span>
                    <span class="bloom-value">${formatVBV(totalYield)}</span>
                    <span class="bloom-label">Total Yield</span>
                </div>
                <div class="bloom-stat">
                    <span class="bloom-icon">🌳</span>
                    <span class="bloom-value">${portfolioData.length}</span>
                    <span class="bloom-label">Investments</span>
                </div>
                <div class="bloom-stat">
                    <span class="bloom-icon">📈</span>
                    <span class="bloom-value">${totalInvested > 0 ? ((totalValue / totalInvested - 1) * 100).toFixed(1) : 0}%</span>
                    <span class="bloom-label">Growth</span>
                </div>
            </div>

            <!-- Auto-Compound Toggle -->
            <div class="compound-toggle-section">
                <label class="compound-toggle">
                    <input type="checkbox" id="compound-toggle" ${autoCompound ? 'checked' : ''} />
                    <span class="compound-toggle-slider"></span>
                    <span class="compound-toggle-label">🔄 Auto-Compound</span>
                </label>
                <span class="compound-toggle-desc">Reinvest yields automatically</span>
            </div>

            <!-- Investment Trees -->
            <div class="investment-trees-section">
                <h4 style="color:#e5e7eb;">🌳 Your Trees</h4>
                <div class="investment-trees-grid">
                    ${portfolioData.length > 0 ? portfolioData.map(p => renderInvestmentTree(p)).join('') : renderEmptyGarden()}
                </div>
            </div>

            <!-- Claim All -->
            <div class="claim-all-section">
                <button class="vbt-btn vbt-btn-primary" id="claim-all-btn" ${totalYield <= 0 ? 'disabled' : ''}>
                    💰 Claim All (${formatVBV(totalYield)})
                </button>
            </div>
        </div>
    `;

    el.innerHTML = html;
    attachDividendListeners(el);
}

function renderInvestmentTree(investment) {
    const name = investment.name || investment.entity_name || 'Unknown';
    const value = investment.value || investment.current_value || 0;
    const invested = investment.invested || investment.amount || 0;
    const yieldAmount = investment.yield || investment.pending_yield || 0;
    const growth = invested > 0 ? ((value / invested - 1) * 100) : 0;
    const isBlooming = yieldAmount > 0;

    return `
        <div class="investment-tree ${isBlooming ? 'tree-blooming' : 'tree-growing'}">
            <div class="tree-canopy">
                <span class="tree-icon">${isBlooming ? '🌺' : '🌱'}</span>
                ${isBlooming ? '<span class="tree-bloom-particle particle-1">✨</span>' : ''}
                ${isBlooming ? '<span class="tree-bloom-particle particle-2">✨</span>' : ''}
            </div>
            <div class="tree-info">
                <span class="tree-name">${escapeHtml(name)}</span>
                <span class="tree-value">${formatVBV(value)}</span>
            </div>
            <div class="tree-stats">
                <span class="tree-stat tree-growth" style="color:${growth >= 0 ? '#4caf50' : '#f44336'}">
                    ${growth >= 0 ? '+' : ''}${growth.toFixed(1)}%
                </span>
                <span class="tree-stat tree-yield">+${formatVBV(yieldAmount)}</span>
            </div>
            <button class="vbt-btn vbt-btn-secondary tree-claim-btn" ${yieldAmount <= 0 ? 'disabled' : ''}>
                🌾 Harvest
            </button>
        </div>
    `;
}

function renderEmptyGarden() {
    return `
        <div class="garden-empty">
            <span class="garden-empty-icon">🌱</span>
            <p>Your garden is empty. Invest in entities to grow your yield!</p>
        </div>
    `;
}

function attachDividendListeners(el) {
    // Auto-compound toggle
    const toggle = el.querySelector('#compound-toggle');
    if (toggle) {
        toggle.addEventListener('change', () => {
            autoCompound = toggle.checked;
            if (window.showToast) window.showToast(autoCompound ? '🔄 Auto-compound enabled' : '⏸ Auto-compound disabled', 'info');
        });
    }

    // Claim all
    const claimBtn = el.querySelector('#claim-all-btn');
    if (claimBtn) {
        claimBtn.addEventListener('click', () => {
            if (window.showToast) window.showToast(`💰 Claimed ${formatVBV(totalYield)}!`, 'success');
        });
    }

    // Individual harvest
    el.querySelectorAll('.tree-claim-btn:not([disabled])').forEach(btn => {
        btn.addEventListener('click', () => {
            if (window.showToast) window.showToast('🌾 Yield harvested!', 'success');
        });
    });
}

// --- Global handlers ---
window.initDividendYield = initDividendYield;

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
