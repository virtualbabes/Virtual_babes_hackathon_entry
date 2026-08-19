// ============================================================================
// investment_dashboard.js — Entity Investment Dashboard (P7-A Task 7003)
// ============================================================================
// Neon-glass entity marketplace, portfolio holdings tracker, dividend claimer.
// Module follows established patterns from justice_dashboard.js + rivalry.js.
// Interacts with backend: /api/invest/entity, /api/claim/dividends, 
//   /api/invest/portfolio, /api/invest/dividends/history
// ============================================================================

const InvestmentDashboard = (() => {
    'use strict';

    // --- Constants ---
    const INVESTMENT_OVERLAY_ID = 'investment-dashboard-overlay';
    const MICRO_UNITS_PER_VBV = 1000000;
    const YIELD_DISPLAY_DECIMALS = 4;

    // --- DOM References (lazy initialized) ---
    let overlayEl = null;
    let entityMarketplaceEl = null;
    let portfolioTableEl = null;
    let dividendTrackerEl = null;
    let yieldSummaryEl = null;

    // --- State ---
    let state = {
        portfolio: null,           // Current holdings map[entity_id] → investment data
        totalValueMicro: 0,         // Total portfolio value in micro-units  
        entityValuations: [],       // Investable entities with AMM valuations
        dividendHistory: [],        // Historical claims per entity
        currentPlayerWallet: '',    // Active player wallet address
        isRefreshing: false,        // Loading guard during API calls
    };

    // ========================================================================
    // INITIALIZATION
    // ========================================================================

    /**
     * Initialize the Investment Dashboard overlay.
     * Creates DOM elements and binds event handlers.
     */
    function init() {
        if (overlayEl) return; // Already initialized

        // Get current player wallet from CONFIG or active player state
        let currentPlayerWallet = '';
        if (window.CONFIG && window.CONFIG.VAULT_ADDRESS) {
            currentPlayerWallet = window.CONFIG.VAULT_ADDRESS;
        } else if (typeof getActivePlayer === 'function') {
            const active = getActivePlayer();
            if (active && active.wallet) currentPlayerWallet = active.wallet;
        }

        // Create overlay structure with 4-panel layout
        overlayEl = document.createElement('div');
        overlayEl.id = INVESTMENT_OVERLAY_ID;
        overlayEl.className = 'investment-overlay neon-glass-panel';
        overlayEl.style.display = 'none';
        overlayEl.innerHTML = `
            <div class="investment-header">
                <h2>💼 Entity Investment Dashboard</h2>
                <button class="close-btn" onclick="window.closeInvestmentDashboard()">✕</button>
            </div>
            <div class="investment-body">
                <!-- Panel 1: Yield Summary -->
                <div class="invest-panel invest-panel-summary">
                    <h3>📊 Portfolio Summary</h3>
                    <div id="yield-summary" class="summary-grid">
                        <div class="summary-item">
                            <span class="summary-label">Total Invested</span>
                            <span id="total-invested" class="summary-value">0.00 $VBV</span>
                        </div>
                        <div class="summary-item">
                            <span class="summary-label">Current Value</span>
                            <span id="current-value" class="summary-value">0.00 $VBV</span>
                        </div>
                        <div class="summary-item">
                            <span class="summary-label">Unrealized P&L</span>
                            <span id="unrealized-pnl" class="summary-value">+0.00%</span>
                        </div>
                        <div class="summary-item">
                            <span class="summary-label">Active Entities</span>
                            <span id="active-entities" class="summary-value">0</span>
                        </div>
                    </div>
                </div>

                <!-- Panel 2: Entity Marketplace -->
                <div class="invest-panel invest-panel-marketplace">
                    <h3>🏪 Entity Marketplace</h3>
                    <div id="entity-marketplace" class="marketplace-grid">
                        <div class="empty-state">Loading entities...</div>
                    </div>
                </div>

                <!-- Panel 3: Portfolio Holdings -->
                <div class="invest-panel invest-panel-portfolio">
                    <h3>📁 My Portfolio</h3>
                    <div id="portfolio-table" class="portfolio-scroll">
                        <div class="empty-state">No investments yet. Invest in the marketplace above.</div>
                    </div>
                </div>

                <!-- Panel 4: Dividend Tracker -->
                <div class="invest-panel invest-panel-dividends">
                    <h3>💰 Dividend Tracker</h3>
                    <div id="dividend-tracker" class="dividend-list">
                        <div class="empty-state">No dividends to claim.</div>
                    </div>
                </div>
            </div>
        `;

        document.body.appendChild(overlayEl);

        // Bind DOM references
        entityMarketplaceEl = document.getElementById('entity-marketplace');
        portfolioTableEl = document.getElementById('portfolio-table');
        dividendTrackerEl = document.getElementById('dividend-tracker');
        yieldSummaryEl = document.getElementById('yield-summary');

        state.currentPlayerWallet = currentPlayerWallet;
    }

    // ========================================================================
    // OVERLAY CONTROL
    // ========================================================================

    /**
     * Open the Investment Dashboard overlay.
     */
    function openInvestmentDashboard() {
        init();
        overlayEl.style.display = 'flex';
        overlayEl.style.flexDirection = 'column';
        overlayEl.style.alignItems = 'center';
        
        refreshPortfolioData();
        renderEntityMarketplace();
    }

    /**
     * Close the Investment Dashboard overlay.
     */
    function closeInvestmentDashboard() {
        if (overlayEl) {
            overlayEl.style.display = 'none';
        }
    }

    // ========================================================================
    // DATA FETCHING
    // ========================================================================

    /**
     * Fetch fresh portfolio data from backend.
     */
    async function refreshPortfolioData() {
        if (state.isRefreshing) return;
        
        const wallet = state.currentPlayerWallet || getActiveWallet();
        if (!wallet) {
            console.warn('[InvestmentDashboard] No active player wallet found.');
            return;
        }

        try {
            state.isRefreshing = true;
            
            // Fetch portfolio data (includes entity valuations + holdings)
            const response = await fetch(`/api/invest/portfolio?wallet=${encodeURIComponent(wallet)}`);
            if (!response.ok) throw new Error(`HTTP ${response.status} fetching portfolio`);

            const data = await response.json();
            state.portfolio = data.portfolio || {};
            state.totalValueMicro = data.totalValueMicro || 0;
            state.entityValuations = data.entityValuations || [];

            updateYieldSummary(data);
            renderPortfolioHoldings(state.portfolio);
            
        } catch (err) {
            console.error('[InvestmentDashboard] Portfolio refresh failed:', err);
            showNotification('Failed to load portfolio data.', 'error');
        } finally {
            state.isRefreshing = false;
        }
    }

    /**
     * Fetch entity marketplace for investment.
     */
    async function renderEntityMarketplace() {
        const wallet = state.currentPlayerWallet || getActiveWallet();
        
        try {
            // Use portfolio endpoint which includes entityValuations array
            if (state.entityValuations.length === 0) {
                await refreshPortfolioData();
            }

            if (!entityMarketplaceEl) return;

            const entities = state.entityValuations || [];
            
            if (entities.length === 0) {
                entityMarketplaceEl.innerHTML = '<div class="empty-state">No investable entities available.</div>';
                return;
            }

            entityMarketplaceEl.innerHTML = entities.map(entity => `
                <div class="entity-card" data-entity-id="${entity.entity_id}">
                    <div class="entity-name">${escapeHtml(entity.name || entity.entity_id.slice(0, 8) + '...')}</div>
                    <div class="entity-meta">
                        <span class="valuation-badge">Valuation: ${formatVBV(entity.valuationMicro)}</span>
                        <span class="yield-badge">Yield: ${(entity.yieldPerShare * 100).toFixed(2)}%</span>
                    </div>
                    <div class="invest-controls">
                        <input type="number" 
                               id="invest-amount-${entity.entity_id}" 
                               class="invest-input" 
                               placeholder="$VBV amount" 
                               min="0.01" 
                               step="0.01"/>
                        <button class="invest-btn neon-button-cyan" 
                                onclick="window.investInEntity('${entity.entity_id}')">
                            Invest
                        </button>
                    </div>
                </div>
            `).join('');

        } catch (err) {
            console.error('[InvestmentDashboard] Marketplace render failed:', err);
            if (entityMarketplaceEl) {
                entityMarketplaceEl.innerHTML = '<div class="empty-state error">Failed to load marketplace.</div>';
            }
        }
    }

    // ========================================================================
    // DISPLAY UPDATES
    // ========================================================================

    /**
     * Update the portfolio yield summary panel.
     */
    function updateYieldSummary(data) {
        const totalInvested = data.totalInvestedMicro || 0;
        const currentValue = state.totalValueMicro || 0;
        
        // Calculate unrealized P&L percentage
        let pnlPercent = 0;
        if (totalInvested > 0) {
            pnlPercent = ((currentValue - totalInvested) / totalInvested) * 100;
        }

        const activeCount = Object.keys(state.portfolio || {}).length;

        // Update DOM elements with P&L color coding
        updateSummaryField('total-invested', `${formatVBV(totalInvested)} $VBV`);
        updateSummaryField('current-value', `${formatVBV(currentValue)} $VBV`);
        
        const pnlEl = document.getElementById('unrealized-pnl');
        if (pnlEl) {
            const sign = pnlPercent >= 0 ? '+' : '';
            pnlEl.textContent = `${sign}${pnlPercent.toFixed(2)}%`;
            pnlEl.className = `summary-value ${pnlPercent > 0 ? 'profit' : pnlPercent < 0 ? 'loss' : ''}`;
        }

        updateSummaryField('active-entities', activeCount.toString());
    }

    /**
     * Update a single summary field element.
     */
    function updateSummaryField(elementId, value) {
        const el = document.getElementById(elementId);
        if (el) el.textContent = value;
    }

    /**
     * Render portfolio holdings table.
     */
    function renderPortfolioHoldings(portfolio) {
        if (!portfolioTableEl) return;

        const entries = Object.entries(portfolio || {});
        
        if (entries.length === 0) {
            portfolioTableEl.innerHTML = '<div class="empty-state">No investments yet. Invest in the marketplace above.</div>';
            return;
        }

        let totalInvestedMicro = 0;

        const rows = entries.map(([entityId, investment]) => {
            const investedVBV = parseFloat(investment.amount) || 0;
            const sharesOwned = parseFloat(investment.shares) || 0;
            const currentValue = parseFloat(investment.currentValue) || 0;
            
            totalInvestedMicro += (investment.microAmount || 0);

            return `
                <div class="portfolio-row">
                    <div class="entity-name">${escapeHtml(entityId.slice(0, 12))}...</div>
                    <div class="invested-amount">${formatVBV(investedVBV * MICRO_UNITS_PER_VBV)}</div>
                    <div class="shares-owned">${sharesOwned.toFixed(4)} shares</div>
                    <div class="current-value">${formatVBV(currentValue * MICRO_UNITS_PER_VBV)}</div>
                </div>
            `;
        }).join('');

        portfolioTableEl.innerHTML = rows;
    }

    /**
     * Render dividend tracker with claim buttons.
     */
    function renderDividendTracker() {
        if (!dividendTrackerEl) return;

        // Compute accrued dividends from current holdings + time delta
        const entries = Object.entries(state.portfolio || {});
        
        if (entries.length === 0) {
            dividendTrackerEl.innerHTML = '<div class="empty-state">No investments to earn dividends.</div>';
            return;
        }

        // Fetch latest history from backend for accurate accrual data
        const wallet = state.currentPlayerWallet || getActiveWallet();
        
        fetch(`/api/invest/dividends/history?wallet=${encodeURIComponent(wallet)}`)
            .then(r => r.json())
            .then(data => {
                const history = data.history || [];
                
                if (history.length === 0) {
                    dividendTrackerEl.innerHTML = '<div class="empty-state">No dividends available to claim.</div>';
                    return;
                }

                // Group by entity and compute total accrued
                const entityAccruals = {};
                history.forEach(record => {
                    if (!entityAccruals[record.entity_id]) {
                        entityAccruals[record.entity_id] = { 
                            daysAccrued: 0, 
                            investedAmount: record.amount_micro || 0 
                        };
                    }
                    entityAccruals[record.entity_id].daysAccrued += record.days_accrued;
                });

                dividendTrackerEl.innerHTML = Object.entries(entityAccruals).map(([entityId, accrual]) => {
                    // Daily yield rate: 0.5% (from backend economy_processing.go)
                    const dailyYieldRate = 0.005;
                    const accruedDividendsMicro = Math.floor(accrual.investedAmount * dailyYieldRate * accrual.daysAccrued);
                    
                    return `
                        <div class="dividend-row" data-entity="${entityId}">
                            <span class="entity-name">${escapeHtml(entityId.slice(0, 12))}...</span>
                            <span class="accrual-amount">${formatVBV(accruedDividendsMicro)} $VBV</span>
                            <span class="days-counting">${accrual.daysAccrued.toFixed(1)} days accrued</span>
                            <button class="claim-btn neon-button-green" 
                                    onclick="window.claimDividends('${entityId}')">
                                Claim Dividend
                            </button>
                        </div>
                    `;
                }).join('');

            })
            .catch(err => {
                console.error('[InvestmentDashboard] Dividend history fetch failed:', err);
                dividendTrackerEl.innerHTML = '<div class="empty-state error">Failed to load dividends.</div>';
            });
    }

    // ========================================================================
    // ACTIONS (Public API)
    // ========================================================================

    /**
     * Invest in an entity. Called from marketplace button onclick.
     * @param {string} entityId - Entity wallet address or ID.
     */
    async function investInEntity(entityId) {
        const amountInput = document.getElementById(`invest-amount-${entityId}`);
        
        if (!amountInput || !amountInput.value) {
            showNotification('Please enter an investment amount.', 'warning');
            return;
        }

        const amountVBV = parseFloat(amountInput.value);
        if (isNaN(amountVBV) || amountVBV <= 0) {
            showNotification('Invalid investment amount.', 'error');
            return;
        }

        // Check minimum investment threshold based on player tier
        let minInvestmentMicro = 10 * MICRO_UNITS_PER_VBV; // Default: 10 VBV
        
        try {
            const response = await fetch(`/api/invest/entity?wallet=${encodeURIComponent(state.currentPlayerWallet)}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    entity_id: entityId,
                    amount_micro: Math.floor(amountVBV * MICRO_UNITS_PER_VBV)
                })
            });

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.error || `HTTP ${response.status}`);
            }

            const result = await response.json();
            
            showNotification(`✅ Invested ${formatVBV(result.invested_micro)} $VBV in entity!`, 'success');
            
            // Clear input field
            amountInput.value = '';
            
            // Refresh portfolio data to reflect new investment
            refreshPortfolioData();

        } catch (err) {
            console.error('[InvestmentDashboard] Investment failed:', err);
            showNotification(err.message || 'Investment failed.', 'error');
        }
    }

    /**
     * Claim dividends from an entity. Called from dividend tracker button onclick.
     * @param {string} entityId - Optional: specific entity to claim from (empty = all entities).
     */
    async function claimDividends(entityId) {
        const wallet = state.currentPlayerWallet || getActiveWallet();
        
        try {
            // POST /api/claim/dividends with optional entity_id filter
            let url = `/api/invest/dividends/history?wallet=${encodeURIComponent(wallet)}&action=claim`;
            
            if (entityId) {
                url += `&entity_id=${encodeURIComponent(entityId)}`;
            }

            const response = await fetch(url, { method: 'POST' });

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.error || `HTTP ${response.status}`);
            }

            const result = await response.json();
            
            showNotification(`💰 Claimed ${formatVBV(result.amount_micro)} $VBV in dividends!`, 'success');
            
            // Refresh portfolio and dividend data
            refreshPortfolioData();
            renderDividendTracker();

        } catch (err) {
            console.error('[InvestmentDashboard] Dividend claim failed:', err);
            showNotification(err.message || 'Failed to claim dividends.', 'error');
        }
    }

    // ========================================================================
    // UTILITY FUNCTIONS
    // ========================================================================

    /**
     * Format micro-units to human-readable $VBV string.
     */
    function formatVBV(microAmount) {
        if (!microAmount && microAmount !== 0) return '0';
        
        const vbv = parseFloat(microAmount) / MICRO_UNITS_PER_VBV;
        return vbv.toFixed(YIELD_DISPLAY_DECIMALS);
    }

    /**
     * Escape HTML to prevent XSS in entity names.
     */
    function escapeHtml(text) {
        if (!text || typeof text !== 'string') return '';
        
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    /**
     * Show notification to user (follows existing UI.showNotification pattern).
     */
    function showNotification(message, type) {
        if (typeof UI !== 'undefined' && typeof UI.showNotification === 'function') {
            UI.showNotification(message, type);
        } else {
            // Fallback: console log with emoji prefix
            const prefixes = { success: '✅', error: '❌', warning: '⚠️', info: 'ℹ️' };
            const prefix = prefixes[type] || '';
            console.log(`[InvestmentDashboard] ${prefix} ${message}`);
        }
    }

    /**
     * Get active wallet from CONFIG or getActivePlayer helper.
     */
    function getActiveWallet() {
        if (window.CONFIG && window.CONFIG.VAULT_ADDRESS) return window.CONFIG.VAULT_ADDRESS;
        
        if (typeof getActivePlayer === 'function') {
            const active = getActivePlayer();
            if (active && active.wallet) return active.wallet;
        }
        
        return '';
    }

    // ========================================================================
    // PUBLIC API EXPORTS
    // ========================================================================

    return {
        openInvestmentDashboard,
        closeInvestmentDashboard,
        refreshPortfolioData,
        investInEntity,
        claimDividends,
        renderDividendTracker,
        
        // State access for external modules (WS event handlers)
        getState: () => state,
    };

})();

// ============================================================================
// GLOBAL EXPORTS — Match existing pattern from justice_dashboard.js + rivalry.js
// ============================================================================

window.openInvestmentDashboard = InvestmentDashboard.openInvestmentDashboard;
window.closeInvestmentDashboard = InvestmentDashboard.closeInvestmentDashboard;
window.investInEntity = InvestmentDashboard.investInEntity;
window.claimDividends = InvestmentDashboard.claimDividends;