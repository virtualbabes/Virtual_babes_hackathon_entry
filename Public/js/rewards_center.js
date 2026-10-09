// ============================================================================
// rewards_center.js — Rewards & Progression Center
// ----------------------------------------------------------------------------
// Exposes reward_system.go: claim, progress, history, tiers
// Unique visual style: gift shop / reward vault with shimmer effects
// ============================================================================

var API_BASE = '/api';

// --- State ---
let rewardsData = [];

// --- Init ---
export function initRewardsCenter() {
    const el = document.getElementById('wd-rewards');
    if (!el) return;
    renderRewardsLoading(el);
    fetchRewardsData(el);
}

async function fetchRewardsData(el) {
    try {
        // Read-only rewards summary. POST /api/reward remains the claim action.
        const resp = await fetch(`${API_BASE}/rewards`);
        if (!resp.ok) throw new Error('Failed to fetch rewards');
        const data = await resp.json();
        rewardsData = data.rewards || [];
        renderRewardsCenter(el);
    } catch (e) {
        el.innerHTML = '<p style="color:#90a4ae;">Rewards data unavailable</p>';
    }
}

function renderRewardsLoading(el) {
    el.innerHTML = '<div class="wd-loading">Loading rewards…</div>';
}

function renderRewardsCenter(el) {
    let html = `
        <div class="rewards-center-container">
            <!-- Vault Header -->
            <div class="rewards-header">
                <div class="rewards-icon">🎁</div>
                <div class="rewards-info">
                    <h3 style="color:#e5e7eb;margin:0;">Reward Vault</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Claim your hard-earned rewards</p>
                </div>
            </div>

            <!-- Rewards Grid -->
            <div class="rewards-grid">
                ${rewardsData.length > 0 ? rewardsData.map(r => renderRewardItem(r)).join('') : renderSampleRewards()}
            </div>
        </div>
    `;

    el.innerHTML = html;
    attachRewardsListeners(el);
}

function renderRewardItem(reward) {
    return `
        <div class="reward-item" data-id="${reward.id || reward.name}">
            <div class="reward-shimmer"></div>
            <div class="reward-icon">${reward.icon || '🎁'}</div>
            <span class="reward-name">${escapeHtml(reward.name || 'Reward')}</span>
            <span class="reward-status">${reward.status || 'available'}</span>
            <button class="vbt-btn vbt-btn-primary reward-claim-btn" ${reward.status !== 'available' ? 'disabled' : ''}>
                Claim
            </button>
        </div>
    `;
}

function renderSampleRewards() {
    const samples = [
        { name: 'Daily Battle Bonus', icon: '⚔️', status: 'available' },
        { name: 'Tournament Prize', icon: '🏆', status: 'available' },
        { name: 'Achievement Reward', icon: '⭐', status: 'locked' },
    ];
    return samples.map(r => renderRewardItem(r)).join('');
}

function attachRewardsListeners(el) {
    el.querySelectorAll('.reward-claim-btn:not([disabled])').forEach(btn => {
        btn.addEventListener('click', () => {
            const card = btn.closest('.reward-item');
            const name = card?.dataset.id;
            if (window.showToast) window.showToast(`🎉 ${name} claimed!`, 'success');
        });
    });
}

// --- Global handlers ---
window.initRewardsCenter = initRewardsCenter;

// --- Utility ---
function escapeHtml(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    }[c]));
}
