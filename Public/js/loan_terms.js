// ============================================================================
// loan_terms.js — Loan Management for World Dashboard
// ----------------------------------------------------------------------------
// Exposes loan_service.go: take loan, repay, collateral view, interest calc
// Unique visual style: financial/banking theme with risk indicators
// ============================================================================

var API_BASE = '/api';

// --- Loan Tiers ---
const LOAN_TIERS = [
    { id: 'micro', name: 'Micro Loan', max: 10_000_000_000, rate: 5, collateral: 150, term: 7, color: '#4caf50' },
    { id: 'standard', name: 'Standard Loan', max: 100_000_000_000, rate: 8, collateral: 200, term: 14, color: '#ff9800' },
    { id: 'premium', name: 'Premium Loan', max: 500_000_000_000, rate: 12, collateral: 300, term: 30, color: '#f44336' },
];

// --- State ---
let loanData = null;
let activeLoans = [];

// --- Init ---
export function initLoanTerms() {
    const el = document.getElementById('wd-loans');
    if (!el) return;
    renderLoanLoading(el);
    fetchLoanData(el);
}

async function fetchLoanData(el) {
    try {
        const resp = await fetch(`${API_BASE}/loans`);
        if (!resp.ok) throw new Error('Failed to fetch loan data');
        loanData = await resp.json();
        activeLoans = loanData.loans || [];
        renderLoanTerms(el);
    } catch (e) {
        el.innerHTML = '<p style="color:#90a4ae;">Loan data unavailable</p>';
    }
}

function renderLoanLoading(el) {
    el.innerHTML = '<div class="wd-loading">Loading loan terms…</div>';
}

function renderLoanTerms(el) {
    let html = `
        <div class="loan-terms-container">
            <!-- Header -->
            <div class="loan-header">
                <div class="loan-header-info">
                    <h3 style="color:#e5e7eb;margin:0;">💰 Loan Center</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:4px 0 0;">
                        Borrow against your market tokens. Default triggers liquidation.
                    </p>
                </div>
                <div class="loan-health-indicator">
                    <span class="loan-health-label">Credit Score</span>
                    <span class="loan-health-value">${calculateCreditScore()}</span>
                </div>
            </div>

            <!-- Loan Tiers -->
            <div class="loan-tiers-section">
                <h4 style="color:#e5e7eb;">📊 Available Loan Tiers</h4>
                <div class="loan-tiers-grid">
                    ${LOAN_TIERS.map(tier => `
                        <div class="loan-tier-card" data-tier="${tier.id}" style="--tier-color:${tier.color}">
                            <div class="loan-tier-header">
                                <span class="loan-tier-name">${tier.name}</span>
                                <span class="loan-tier-risk ${tier.id}">${tier.id.toUpperCase()}</span>
                            </div>
                            <div class="loan-tier-details">
                                <div class="loan-tier-detail">
                                    <span class="loan-detail-label">Max Borrow</span>
                                    <span class="loan-detail-value">${formatVBV(tier.max)}</span>
                                </div>
                                <div class="loan-tier-detail">
                                    <span class="loan-detail-label">Interest</span>
                                    <span class="loan-detail-value" style="color:${tier.color}">${tier.rate}%</span>
                                </div>
                                <div class="loan-tier-detail">
                                    <span class="loan-detail-label">Collateral</span>
                                    <span class="loan-detail-value">${tier.collateral}%</span>
                                </div>
                                <div class="loan-tier-detail">
                                    <span class="loan-detail-label">Term</span>
                                    <span class="loan-detail-value">${tier.term} days</span>
                                </div>
                            </div>
                            <button class="vbt-btn vbt-btn-primary loan-apply-btn" data-tier="${tier.id}">
                                Apply
                            </button>
                        </div>
                    `).join('')}
                </div>
            </div>

            <!-- Active Loans -->
            ${activeLoans.length > 0 ? `
                <div class="loan-active-section">
                    <h4 style="color:#e5e7eb;">📋 Active Loans</h4>
                    <div class="loan-active-list">
                        ${activeLoans.map(l => renderActiveLoan(l)).join('')}
                    </div>
                </div>
            ` : renderNoActiveLoans()}

            <!-- Loan Calculator -->
            <div class="loan-calculator-section">
                <h4 style="color:#e5e7eb;">🧮 Loan Calculator</h4>
                <div class="loan-calculator">
                    <div class="loan-calc-row">
                        <label>Borrow Amount ($VBV)</label>
                        <input type="number" id="loan-calc-amount" min="1" value="100" />
                    </div>
                    <div class="loan-calc-row">
                        <label>Loan Tier</label>
                        <select id="loan-calc-tier">
                            ${LOAN_TIERS.map(t => `<option value="${t.id}">${t.name} (${t.rate}%)</option>`).join('')}
                        </select>
                    </div>
                    <div class="loan-calc-result">
                        <div class="loan-calc-item">
                            <span>Required Collateral:</span>
                            <span id="loan-calc-collateral">--</span>
                        </div>
                        <div class="loan-calc-item">
                            <span>Total Repayment:</span>
                            <span id="loan-calc-repayment">--</span>
                        </div>
                        <div class="loan-calc-item">
                            <span>Daily Interest:</span>
                            <span id="loan-calc-daily">--</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    `;

    el.innerHTML = html;
    attachLoanListeners(el);
    calculateLoan();
}

function renderActiveLoan(loan) {
    const amount = loan.amount_micro || 0;
    const rate = loan.interest_rate || 0;
    const due = loan.due_date || loan.expires_at;
    const collateral = loan.collateral_micro || 0;
    const progress = loan.repayment_progress || 0;

    return `
        <div class="loan-active-card">
            <div class="loan-active-header">
                <span class="loan-active-amount">${formatVBV(amount)}</span>
                <span class="loan-active-rate">${rate}% APR</span>
            </div>
            <div class="loan-active-progress">
                <div class="loan-progress-bar">
                    <div class="loan-progress-fill" style="width:${progress}%;background:${progress > 80 ? '#f44336' : progress > 50 ? '#ff9800' : '#4caf50'}"></div>
                </div>
                <span class="loan-progress-label">${progress}% repaid</span>
            </div>
            <div class="loan-active-details">
                <span class="loan-detail">🏦 Collateral: ${formatVBV(collateral)}</span>
                <span class="loan-detail">📅 Due: ${due ? new Date(due).toLocaleDateString() : 'N/A'}</span>
            </div>
            <div class="loan-active-actions">
                <button class="vbt-btn vbt-btn-primary loan-repay-btn">💳 Repay</button>
                <button class="vbt-btn vbt-btn-secondary loan-refinance-btn">🔄 Refinance</button>
            </div>
        </div>
    `;
}

function renderNoActiveLoans() {
    return `
        <div class="loan-empty">
            <span class="loan-empty-icon">🏦</span>
            <p>No active loans. Your credit line is clear.</p>
        </div>
    `;
}

function calculateCreditScore() {
    if (!activeLoans || activeLoans.length === 0) return 'A+';
    const totalDebt = activeLoans.reduce((sum, l) => sum + (l.amount_micro || 0), 0);
    if (totalDebt > 500_000_000_000) return 'C';
    if (totalDebt > 100_000_000_000) return 'B';
    return 'A';
}

function calculateLoan() {
    const amountEl = document.getElementById('loan-calc-amount');
    const tierEl = document.getElementById('loan-calc-tier');
    if (!amountEl || !tierEl) return;

    const amount = parseFloat(amountEl.value) * 1_000_000 || 0;
    const tier = LOAN_TIERS.find(t => t.id === tierEl.value) || LOAN_TIERS[0];

    const collateral = Math.ceil(amount * tier.collateral / 100);
    const interest = Math.ceil(amount * tier.rate / 100);
    const total = amount + interest;
    const daily = Math.ceil(interest / tier.term);

    document.getElementById('loan-calc-collateral').textContent = formatVBV(collateral);
    document.getElementById('loan-calc-repayment').textContent = formatVBV(total);
    document.getElementById('loan-calc-daily').textContent = formatVBV(daily);
}

function attachLoanListeners(el) {
    // Apply buttons
    el.querySelectorAll('.loan-apply-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const tierId = btn.dataset.tier;
            applyForLoan(tierId);
        });
    });

    // Calculator inputs
    const amountEl = el.querySelector('#loan-calc-amount');
    const tierEl = el.querySelector('#loan-calc-tier');
    if (amountEl) amountEl.addEventListener('input', calculateLoan);
    if (tierEl) tierEl.addEventListener('change', calculateLoan);

    // Repay buttons
    el.querySelectorAll('.loan-repay-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            if (window.showToast) window.showToast('Repayment requires on-chain transaction', 'info');
        });
    });
}

async function applyForLoan(tierId) {
    const tier = LOAN_TIERS.find(t => t.id === tierId);
    if (!tier) return;

    try {
        const resp = await fetch(`${API_BASE}/loans/take`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ amount: tier.max, tier: tierId })
        });
        if (!resp.ok) throw new Error('Failed to apply for loan');
        if (window.showToast) window.showToast(`✅ ${tier.name} application submitted!`, 'success');
    } catch (e) {
        if (window.showToast) window.showToast(`❌ ${e.message}`, 'error');
    }
}

// --- Global handlers ---
window.initLoanTerms = initLoanTerms;

// --- Utility ---
function formatVBV(micro) {
    const vbv = Number(micro) / 1_000_000;
    if (vbv >= 1_000_000) return `${(vbv / 1_000_000).toFixed(1)}M`;
    if (vbv >= 1_000) return `${(vbv / 1_000).toFixed(1)}K`;
    return vbv.toFixed(2) + ' $VBV';
}
