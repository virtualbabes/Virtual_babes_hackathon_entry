// ============================================================================
// underworld_contracts.js — Criminal Contract Board
// ----------------------------------------------------------------------------
// Exposes underworld_contracts.go: career-gated contracts, accept/abort, sabotage
// Unique visual style: dark underworld / criminal notice board with wanted vibe
// ============================================================================

var API_BASE = '/api';

// --- Contract Types ---
const CONTRACT_TYPES = [
    { id: 'jail_capture', name: 'Jail Capture', icon: '⛓️', difficulty: 3, reward: '500-2000', desc: 'Capture a target on club territory' },
    { id: 'sabotage_district', name: 'Sabotage District', icon: '💣', difficulty: 4, reward: '1000-5000', desc: 'Sabotage a district defenses' },
    { id: 'governor_rumor', name: 'Governor Rumor', icon: '📰', difficulty: 2, reward: '250-1000', desc: 'Spread damaging rumors' },
    { id: 'governor_heist', name: 'Governor Heist', icon: '💰', difficulty: 5, reward: '5000-25000', desc: 'Heist the governors club' },
    { id: 'kidnap', name: 'Kidnap', icon: '🎭', difficulty: 5, reward: '3000-15000', desc: 'Kidnap a high value target' },
    { id: 'hostage_liberation', name: 'Hostage Liberation', icon: '🛡️', difficulty: 3, reward: '1000-5000', desc: 'Free a hostage from governor' },
];

// --- State ---
let contracts = [];
let activeContracts = [];

// --- Init ---
export function initUnderworldContracts() {
    const el = document.getElementById('wd-contracts');
    if (!el) return;
    renderContractsLoading(el);
    fetchContractData(el);
}

async function fetchContractData(el) {
    try {
        const resp = await fetch(`${API_BASE}/contracts/list`);
        if (!resp.ok) throw new Error('Failed to fetch contracts');
        const data = await resp.json();
        contracts = data.contracts || [];
        renderUnderworldContracts(el);
    } catch (e) {
        el.innerHTML = '<p style="color:#90a4ae;">Contract data unavailable</p>';
    }
}

function renderContractsLoading(el) {
    el.innerHTML = '<div class="wd-loading">Loading contract board…</div>';
}

function renderUnderworldContracts(el) {
    let html = `
        <div class="underworld-container">
            <!-- Dark Header -->
            <div class="underworld-header">
                <div class="underworld-icon">🦹</div>
                <div class="underworld-info">
                    <h3 style="color:#e5e7eb;margin:0;">Contract Board</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Dark opportunities await. Choose wisely.</p>
                </div>
                <div class="underworld-cut">
                    <span class="cut-label">Commission</span>
                    <span class="cut-value">8%</span>
                </div>
            </div>

            <!-- Warning Banner -->
            <div class="underworld-warning">
                <span class="warning-icon">⚠️</span>
                <span class="warning-text">Contracts increase Wanted Level. Abort penalties apply.</span>
            </div>

            <!-- Active Contracts -->
            ${activeContracts.length > 0 ? `
                <div class="active-contracts-section">
                    <h4 style="color:#e5e7eb;">📋 Active Contracts</h4>
                    <div class="active-contracts-list">
                        ${activeContracts.map(c => renderActiveContract(c)).join('')}
                    </div>
                </div>
            ` : ''}

            <!-- Available Contracts -->
            <div class="available-contracts-section">
                <h4 style="color:#e5e7eb;">📜 Available Contracts</h4>
                <div class="contracts-grid">
                    ${contracts.length > 0 ? contracts.map(c => renderContractCard(c)).join('') : renderSampleContracts()}
                </div>
            </div>
        </div>
    `;

    el.innerHTML = html;
    attachContractListeners(el);
}

function renderSampleContracts() {
    return CONTRACT_TYPES.slice(0, 4).map(ct => `
        <div class="contract-card" data-type="${ct.id}">
            <div class="contract-card-header">
                <span class="contract-icon">${ct.icon}</span>
                <span class="contract-name">${ct.name}</span>
            </div>
            <div class="contract-card-body">
                <span class="contract-desc">${ct.desc}</span>
                <div class="contract-details">
                    <span class="contract-difficulty">
                        ${'★'.repeat(ct.difficulty)}${'☆'.repeat(5 - ct.difficulty)}
                    </span>
                    <span class="contract-reward">${ct.reward} $VBV</span>
                </div>
            </div>
            <button class="vbt-btn vbt-btn-primary contract-accept-btn">
                🦴 Accept
            </button>
        </div>
    `).join('');
}

function renderActiveContract(contract) {
    return `
        <div class="active-contract-card" data-id="${contract.id}">
            <div class="active-contract-header">
                <span class="active-contract-name">${escapeHtml(contract.title || contract.id)}</span>
                <span class="active-contract-status">ACTIVE</span>
            </div>
            <div class="active-contract-progress">
                <div class="progress-bar">
                    <div class="progress-fill" style="width:${contract.progress || 0}%"></div>
                </div>
                <span class="progress-text">${contract.progress || 0}%</span>
            </div>
            <div class="active-contract-actions">
                <button class="vbt-btn vbt-btn-secondary contract-abort-btn">❌ Abort</button>
            </div>
        </div>
    `;
}

function attachContractListeners(el) {
    // Accept buttons
    el.querySelectorAll('.contract-accept-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const card = btn.closest('.contract-card');
            const type = card?.dataset.type;
            if (window.showToast) window.showToast(`🦴 Contract accepted: ${type}`, 'success');
        });
    });

    // Abort buttons
    el.querySelectorAll('.contract-abort-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            if (window.showToast) window.showToast('❌ Contract aborted. Penalty applied.', 'info');
        });
    });
}

// --- Global handlers ---
window.initUnderworldContracts = initUnderworldContracts;

// --- Utility ---
function escapeHtml(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    }[c]));
}
