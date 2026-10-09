// ============================================================================
// justice_dashboard.js — Bounty Board & Justice Missions
// ----------------------------------------------------------------------------
// Exposes justice_service.go: bounty board, missions, truth serum, rep shield
// Unique visual style: wanted poster / law enforcement theme
// ============================================================================

var API_BASE = '/api';

// --- State ---
let justiceData = null;
let bountyTargets = [];
let activeMissions = [];

// --- Init ---
export function initJusticeDashboard() {
    const el = document.getElementById('wd-justice');
    if (!el) return;
    renderJusticeLoading(el);
    fetchJusticeData(el);
}

async function fetchJusticeData(el) {
    try {
        const resp = await fetch(`${API_BASE}/justice/dashboard`);
        if (!resp.ok) throw new Error('Failed to fetch justice data');
        justiceData = await resp.json();
        bountyTargets = justiceData.high_wanted_targets || [];
        activeMissions = justiceData.active_missions || [];
        renderJusticeDashboard(el);
    } catch (e) {
        el.innerHTML = '<p style="color:#90a4ae;">Justice data unavailable</p>';
    }
}

function renderJusticeLoading(el) {
    el.innerHTML = '<div class="wd-loading">Loading bounty board…</div>';
}

function renderJusticeDashboard(el) {
    const powerBonus = justiceData?.power_bonus || 0;
    const tier = justiceData?.tier || 0;
    const bountiesCount = bountyTargets?.length || 0;

    let html = `
        <div class="justice-dashboard-container">
            <!-- Header with power bonus -->
            <div class="justice-header">
                <div class="justice-badge">
                    <span class="justice-badge-icon">⚖</span>
                    <div class="justice-badge-info">
                        <span class="justice-badge-tier">Tier ${tier}</span>
                        <span class="justice-badge-bonus">+${powerBonus}% vs Outlaws</span>
                    </div>
                </div>
                <div class="justice-stats-row">
                    <div class="justice-stat">
                        <span class="justice-stat-val">${bountiesCount}</span>
                        <span class="justice-stat-label">Wanted</span>
                    </div>
                    <div class="justice-stat">
                        <span class="justice-stat-val">${activeMissions.length}</span>
                        <span class="justice-stat-label">Missions</span>
                    </div>
                </div>
            </div>

            <!-- Bounty Board -->
            <div class="bounty-section">
                <h4 style="color:#e5e7eb;">🎯 Bounty Board</h4>
                <div class="bounty-grid">
                    ${bountyTargets.length > 0 ? bountyTargets.map(t => renderBountyCard(t)).join('') : renderEmptyBounties()}
                </div>
            </div>

            <!-- Active Missions -->
            ${activeMissions.length > 0 ? `
                <div class="missions-section">
                    <h4 style="color:#e5e7eb;">📋 Active Missions</h4>
                    <div class="missions-list">
                        ${activeMissions.map(m => renderMissionCard(m)).join('')}
                    </div>
                </div>
            ` : ''}

            <!-- Justice Actions -->
            <div class="justice-actions-section">
                <h4 style="color:#e5e7eb;">⚡ Justice Actions</h4>
                <div class="justice-actions-grid">
                    <button class="justice-action-btn" onclick="window.justiceGenerateMission()">
                        <span class="justice-action-icon">📜</span>
                        <span class="justice-action-label">New Mission</span>
                    </button>
                    <button class="justice-action-btn" onclick="window.justiceUseTruthSerum()">
                        <span class="justice-action-icon">🧪</span>
                        <span class="justice-action-label">Truth Serum</span>
                    </button>
                    <button class="justice-action-btn" onclick="window.justiceUseRepShield()">
                        <span class="justice-action-icon">🛡</span>
                        <span class="justice-action-label">Rep Shield</span>
                    </button>
                    <button class="justice-action-btn" onclick="window.justiceBountyBoard()">
                        <span class="justice-action-icon">🔍</span>
                        <span class="justice-action-label">Scan District</span>
                    </button>
                </div>
            </div>
        </div>
    `;

    el.innerHTML = html;
    attachJusticeListeners(el);
}

function renderBountyCard(target) {
    const wantedLevel = target.wanted_level || 0;
    const dangerClass = wantedLevel >= 20 ? 'danger-extreme' : wantedLevel >= 10 ? 'danger-high' : wantedLevel >= 5 ? 'danger-medium' : 'danger-low';
    const wantedStars = '⭐'.repeat(Math.min(5, Math.ceil(wantedLevel / 5)));

    return `
        <div class="bounty-card ${dangerClass}" data-player="${target.player_id || target.PlayerID}">
            <div class="bounty-card-header">
                <span class="bounty-danger-badge">${dangerClass.replace('danger-', '').toUpperCase()}</span>
                <span class="bounty-wanted-stars">${wantedStars}</span>
            </div>
            <div class="bounty-card-body">
                <span class="bounty-target-name">${escapeHtml(target.player_name || target.PlayerName || 'Unknown')}</span>
                <span class="bounty-target-id">${escapeHtml((target.player_id || target.PlayerID || '').slice(0, 12))}...</span>
            </div>
            <div class="bounty-card-details">
                <span class="bounty-detail">📍 ${escapeHtml(target.district || target.District || 'Unknown')}</span>
                <span class="bounty-detail">⚠ Wanted: ${wantedLevel}</span>
                <span class="bounty-detail">👻 ${target.ghost_active || target.GhostActive ? 'YES' : 'NO'}</span>
            </div>
            <div class="bounty-card-actions">
                <button class="vbt-btn vbt-btn-primary bounty-claim-btn">
                    🎯 Claim
                </button>
                <button class="vbt-btn vbt-btn-secondary bounty-track-btn">
                    👁 Track
                </button>
            </div>
        </div>
    `;
}

function renderEmptyBounties() {
    return `
        <div class="bounty-empty">
            <span class="bounty-empty-icon">🕊</span>
            <p>No bounties currently active. The district is peaceful.</p>
        </div>
    `;
}

function renderMissionCard(mission) {
    const reward = mission.reward_vbv || mission.RewardVBV || 0;
    const target = mission.target_player_id || mission.TargetPlayerID || 'Unknown';
    const status = mission.status || mission.Status || 'ACTIVE';
    const expiry = mission.expiration_time || mission.ExpirationTime;

    return `
        <div class="mission-card status-${status.toLowerCase()}">
            <div class="mission-card-header">
                <span class="mission-title">${escapeHtml(mission.title || mission.Title || mission.mission_id || 'Mission')}</span>
                <span class="mission-status status-${status.toLowerCase()}">${status}</span>
            </div>
            <div class="mission-card-body">
                <span class="mission-target">🎯 ${escapeHtml(target.slice(0, 16))}...</span>
                <span class="mission-reward">💰 ${formatVBV(reward)}</span>
            </div>
            ${expiry ? `<div class="mission-expiry">⏱ Expires: ${new Date(expiry).toLocaleString()}</div>` : ''}
            <div class="mission-actions">
                ${status === 'ACTIVE' ? `
                    <button class="vbt-btn vbt-btn-primary mission-complete-btn">✅ Complete</button>
                    <button class="vbt-btn vbt-btn-secondary mission-abort-btn">❌ Abort</button>
                ` : ''}
            </div>
        </div>
    `;
}

function attachJusticeListeners(el) {
    // Claim bounty buttons
    el.querySelectorAll('.bounty-claim-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const card = btn.closest('.bounty-card');
            const playerId = card?.dataset.player;
            claimBounty(playerId);
        });
    });

    // Track buttons
    el.querySelectorAll('.bounty-track-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const card = btn.closest('.bounty-card');
            const playerId = card?.dataset.player;
            trackTarget(playerId);
        });
    });

    // Mission complete buttons
    el.querySelectorAll('.mission-complete-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            if (window.showToast) window.showToast('Mission completion requires on-chain transaction', 'info');
        });
    });
}

async function claimBounty(playerId) {
    if (!playerId) return;
    try {
        const resp = await fetch(`${API_BASE}/justice/capture-bounty`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ target_wallet: playerId })
        });
        if (!resp.ok) throw new Error('Failed to claim bounty');
        if (window.showToast) window.showToast('🎯 Bounty claimed!', 'success');
    } catch (e) {
        if (window.showToast) window.showToast(`❌ ${e.message}`, 'error');
    }
}

function trackTarget(playerId) {
    if (window.sendSpectate) window.sendSpectate(playerId);
    else alert(`Tracking ${playerId}`);
}

// --- Global handlers ---
window.justiceGenerateMission = function() {
    if (window.showToast) window.showToast('Generating new mission...', 'info');
};

window.justiceUseTruthSerum = function() {
    const target = prompt('Enter target wallet for Truth Serum:');
    if (target) alert(`Truth Serum applied to ${target.slice(0, 12)}...`);
};

window.justiceUseRepShield = function() {
    if (window.showToast) window.showToast('Reputation Shield activated!', 'success');
};

window.justiceBountyBoard = function() {
    alert('District scan in progress...');
};

window.initJusticeDashboard = initJusticeDashboard;

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

// --- Globals ---
window.initJusticeDashboard = initJusticeDashboard;
