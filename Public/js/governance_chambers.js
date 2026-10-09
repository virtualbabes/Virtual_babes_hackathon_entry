// ============================================================================
// governance_chambers.js — 6-Dimension Governance & Election Arena
// ----------------------------------------------------------------------------
// Exposes governance.go: elections, 6-dim weight, proposals, regional control
// Unique visual style: senate chamber / political arena with radar charts
// ============================================================================

var API_BASE = '/api';

// --- 6 Governance Dimensions ---
const GOV_DIMENSIONS = [
    { id: 'trust', name: 'Trust', icon: '🤝', color: '#4caf50', desc: 'Community trust score' },
    { id: 'reputation', name: 'Reputation', icon: '⭐', color: '#2196f3', desc: 'Overall reputation' },
    { id: 'economic', name: 'Economic', icon: '💰', color: '#ff9800', desc: 'Economic contribution' },
    { id: 'competitive', name: 'Competitive', icon: '⚔️', color: '#f44336', desc: 'Tournament success' },
    { id: 'community', name: 'Community', icon: '👥', color: '#9c27b0', desc: 'Social engagement' },
    { id: 'creative', name: 'Creative', icon: '🎨', color: '#e91e63', desc: 'Creator output' },
];

// --- State ---
let governanceData = null;
let electionData = null;

// --- Init ---
export function initGovernanceChambers() {
    const el = document.getElementById('wd-governance');
    if (!el) return;
    renderGovernanceLoading(el);
    fetchGovernanceData(el);
}

async function fetchGovernanceData(el) {
    try {
        const resp = await fetch(`${API_BASE}/governance/weight`);
        if (!resp.ok) throw new Error('Failed to fetch governance data');
        governanceData = await resp.json();
        renderGovernanceChambers(el);
    } catch (e) {
        el.innerHTML = '<p style="color:#90a4ae;">Governance data unavailable</p>';
    }
}

function renderGovernanceLoading(el) {
    el.innerHTML = '<div class="wd-loading">Loading chambers…</div>';
}

function renderGovernanceChambers(el) {
    const weight = governanceData?.weight || {};
    const totalScore = weight.total_score || 0;

    let html = `
        <div class="governance-chambers-container">
            <!-- Senate Header -->
            <div class="senate-header">
                <div class="senate-icon">🏛️</div>
                <div class="senate-info">
                    <h3 style="color:#e5e7eb;margin:0;">Governance Chambers</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Shape the civilization's future</p>
                </div>
                <div class="senate-power">
                    <span class="senate-power-label">Total Power</span>
                    <span class="senate-power-value">${totalScore}</span>
                </div>
            </div>

            <!-- 6-Dimension Radar -->
            <div class="gov-radar-section">
                <h4 style="color:#e5e7eb;">📊 Influence Radar</h4>
                <div class="gov-radar-container">
                    <svg class="gov-radar-svg" viewBox="0 0 200 200">
                        <!-- Radar background -->
                        ${[100, 75, 50, 25].map(r => `
                            <polygon points="${generateRadarPoints(6, r)}" fill="none" stroke="rgba(255,255,255,0.05)" stroke-width="1"/>
                        `).join('')}
                        <!-- Axis lines -->
                        ${GOV_DIMENSIONS.map((d, i) => {
                            const angle = (Math.PI * 2 * i) / 6 - Math.PI / 2;
                            const x = 100 + Math.cos(angle) * 100;
                            const y = 100 + Math.sin(angle) * 100;
                            return `<line x1="100" y1="100" x2="${x}" y2="${y}" stroke="rgba(255,255,255,0.1)" stroke-width="1"/>`;
                        }).join('')}
                        <!-- Player polygon -->
                        <polygon points="${generateRadarPlayerPoints(weight)}" fill="rgba(0,188,212,0.2)" stroke="#00bcd4" stroke-width="2"/>
                        <!-- Dimension dots & labels -->
                        ${GOV_DIMENSIONS.map((d, i) => {
                            const angle = (Math.PI * 2 * i) / 6 - Math.PI / 2;
                            const value = weight[d.id + '_score'] || 0;
                            const dist = Math.min(100, value / 100);
                            const x = 100 + Math.cos(angle) * dist;
                            const y = 100 + Math.sin(angle) * dist;
                            const lx = 100 + Math.cos(angle) * 115;
                            const ly = 100 + Math.sin(angle) * 115;
                            return `
                                <circle cx="${x}" cy="${y}" r="3" fill="${d.color}"/>
                                <text x="${lx}" y="${ly}" fill="${d.color}" font-size="8" text-anchor="middle" dominant-baseline="middle">${d.icon}</text>
                            `;
                        }).join('')}
                    </svg>
                </div>
            </div>

            <!-- Dimension Breakdown -->
            <div class="gov-dimensions-section">
                <h4 style="color:#e5e7eb;">📋 Dimension Breakdown</h4>
                <div class="gov-dimensions-list">
                    ${GOv_DIMENSIONS.map(d => renderDimensionCard(d, weight[`${d.id}_score`] || 0)).join('')}
                </div>
            </div>

            <!-- Election Status -->
            <div class="gov-election-section">
                <h4 style="color:#e5e7eb;">🗳️ Elections</h4>
                <div class="gov-election-actions">
                    <button class="vbt-btn vbt-btn-primary" onclick="window.govRunCandidate()">
                        🎯 Run for Office
                    </button>
                    <button class="vbt-btn vbt-btn-secondary" onclick="window.govViewElection()">
                        📊 View Election
                    </button>
                </div>
            </div>

            <!-- Proposals -->
            <div class="gov-proposals-section">
                <h4 style="color:#e5e7eb;">📜 Proposals</h4>
                <div class="gov-proposals-list">
                    <div class="gov-proposal-card status-active">
                        <div class="gov-proposal-header">
                            <span class="gov-proposal-title">Increase district tax to 3%</span>
                            <span class="gov-proposal-status">ACTIVE</span>
                        </div>
                        <div class="gov-proposal-votes">
                            <span class="gov-vote-yes">👍 45</span>
                            <span class="gov-vote-no">👎 12</span>
                        </div>
                        <div class="gov-proposal-actions">
                            <button class="vbt-btn vbt-btn-primary gov-vote-btn">👍 Vote Yes</button>
                            <button class="vbt-btn vbt-btn-secondary gov-vote-btn">👎 Vote No</button>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    `;

    el.innerHTML = html;
}

function generateRadarPoints(sides, radius) {
    const points = [];
    for (let i = 0; i < sides; i++) {
        const angle = (Math.PI * 2 * i) / sides - Math.PI / 2;
        const x = 100 + Math.cos(angle) * radius;
        const y = 100 + Math.sin(angle) * radius;
        points.push(`${x},${y}`);
    }
    return points.join(' ');
}

function generateRadarPlayerPoints(weight) {
    const points = [];
    for (let i = 0; i < GOv_DIMENSIONS.length; i++) {
        const dim = GOv_DIMENSIONS[i];
        const value = weight[dim.id + '_score'] || 0;
        const dist = Math.min(100, value / 100);
        const angle = (Math.PI * 2 * i) / GOv_DIMENSIONS.length - Math.PI / 2;
        const x = 100 + Math.cos(angle) * dist;
        const y = 100 + Math.sin(angle) * dist;
        points.push(`${x},${y}`);
    }
    return points.join(' ');
}

function renderDimensionCard(dim, score) {
    const maxScore = 1000;
    const pct = Math.min(100, Math.round(score / maxScore * 100));

    return `
        <div class="gov-dimension-card" style="--dim-color:${dim.color}">
            <div class="gov-dimension-icon">${dim.icon}</div>
            <div class="gov-dimension-info">
                <span class="gov-dimension-name">${dim.name}</span>
                <span class="gov-dimension-desc">${dim.desc}</span>
            </div>
            <div class="gov-dimension-bar">
                <div class="gov-dimension-fill" style="width:${pct}%;background:${dim.color}"></div>
            </div>
            <span class="gov-dimension-score">${score}</span>
        </div>
    `;
}

// --- Global handlers ---
window.govRunCandidate = function() {
    const region = prompt('Enter region name to run for:');
    if (region) {
        if (window.showToast) window.showToast(`🎯 Candidacy filed for ${region}!`, 'success');
    }
};

window.govViewElection = function() {
    if (window.showToast) window.showToast('Opening election details...', 'info');
};

window.initGovernanceChambers = initGovernanceChambers;
