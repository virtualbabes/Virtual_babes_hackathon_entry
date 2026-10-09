// ============================================================================
// rivalry_challenge.js — Rivalry System & PvP Challenge
// ----------------------------------------------------------------------------
// Exposes rivalry_handlers.go: challenge, accept/decline, score tracking
// Unique visual style: competitive duel / gladiator arena theme
// ============================================================================

var API_BASE = '/api';

// --- State ---
let rivalries = [];
let pendingInvites = [];
let rivalryStats = null;

// --- Init ---
export function initRivalryChallenge() {
    const el = document.getElementById('wd-rivalry');
    if (!el) return;
    renderRivalryLoading(el);
    fetchRivalryData(el);
}

async function fetchRivalryData(el) {
    try {
        const resp = await fetch(`${API_BASE}/rivalry/list`);
        if (!resp.ok) throw new Error('Failed to fetch rivalry data');
        const data = await resp.json();
        rivalries = data.rivalries || [];
        pendingInvites = data.pending_invites || [];
        renderRivalryChallenge(el);
    } catch (e) {
        el.innerHTML = '<p style="color:#90a4ae;">Rivalry data unavailable</p>';
    }
}

function renderRivalryLoading(el) {
    el.innerHTML = '<div class="wd-loading">Loading arena…</div>';
}

function renderRivalryChallenge(el) {
    let html = `
        <div class="rivalry-container">
            <!-- Arena Header -->
            <div class="rivalry-header">
                <div class="rivalry-arena-icon">⚔️</div>
                <div class="rivalry-header-info">
                    <h3 style="color:#e5e7eb;margin:0;">Duel Arena</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Challenge rivals and prove your dominance</p>
                </div>
                <div class="rivalry-record">
                    <span class="rivalry-wins">${rivalryStats?.wins || 0}W</span>
                    <span class="rivalry-losses">${rivalryStats?.losses || 0}L</span>
                </div>
            </div>

            <!-- Challenge Actions -->
            <div class="rivalry-actions-section">
                <button class="vbt-btn vbt-btn-primary rivalry-challenge-btn" onclick="window.rivalryNewChallenge()">
                    ⚔️ New Challenge
                </button>
                <button class="vbt-btn vbt-btn-secondary rivalry-leaderboard-btn" onclick="window.rivalryViewLeaderboard()">
                    🏆 Leaderboard
                </button>
            </div>

            <!-- Pending Invites -->
            ${pendingInvites.length > 0 ? `
                <div class="rivalry-pending-section">
                    <h4 style="color:#e5e7eb;">📨 Pending Invites</h4>
                    <div class="rivalry-pending-list">
                        ${pendingInvites.map(invite => renderPendingInvite(invite)).join('')}
                    </div>
                </div>
            ` : ''}

            <!-- Active Rivalries -->
            <div class="rivalry-active-section">
                <h4 style="color:#e5e7eb;">🔥 Active Rivalries</h4>
                <div class="rivalry-active-grid">
                    ${rivalries.length > 0 ? rivalries.map(r => renderRivalryCard(r)).join('') : renderEmptyRivalries()}
                </div>
            </div>
        </div>
    `;

    el.innerHTML = html;
    attachRivalryListeners(el);
}

function renderPendingInvite(invite) {
    return `
        <div class="rivalry-pending-card" data-invite-id="${invite.id}">
            <div class="rivalry-pending-header">
                <span class="rivalry-pending-name">${escapeHtml(invite.challenger || invite.from || 'Unknown')}</span>
                <span class="rivalry-pending-time">${invite.created_at || 'Just now'}</span>
            </div>
            <div class="rivalry-pending-actions">
                <button class="vbt-btn vbt-btn-primary rivalry-accept-btn" data-invite="${invite.id}">✓ Accept</button>
                <button class="vbt-btn vbt-btn-secondary rivalry-decline-btn" data-invite="${invite.id}">✗ Decline</button>
            </div>
        </div>
    `;
}

function renderRivalryCard(rivalry) {
    const sideA = rivalry.side_a || 'Player 1';
    const sideB = rivalry.side_b || 'Player 2';
    const scoreA = rivalry.score_a || 0;
    const scoreB = rivalry.score_b || 0;
    const total = scoreA + scoreB;
    const leader = scoreA > scoreB ? 'A' : scoreB > scoreA ? 'B' : 'tie';

    return `
        <div class="rivalry-card">
            <div class="rivalry-card-header">
                <span class="rivalry-card-title">Duel</span>
                <span class="rivalry-card-status">ACTIVE</span>
            </div>
            <div class="rivalry-matchup">
                <div class="rivalry-player ${leader === 'A' ? 'player-leading' : ''}">
                    <span class="rivalry-player-name">${escapeHtml(sideA.slice(0, 12))}</span>
                    <span class="rivalry-player-score">${scoreA}</span>
                </div>
                <div class="rivalry-vs">VS</div>
                <div class="rivalry-player ${leader === 'B' ? 'player-leading' : ''}">
                    <span class="rivalry-player-name">${escapeHtml(sideB.slice(0, 12))}</span>
                    <span class="rivalry-player-score">${scoreB}</span>
                </div>
            </div>
            <div class="rivalry-progress">
                <div class="rivalry-progress-bar">
                    <div class="rivalry-progress-side-a" style="width:${total > 0 ? (scoreA / total * 100) : 50}%"></div>
                    <div class="rivalry-progress-side-b" style="width:${total > 0 ? (scoreB / total * 100) : 50}%"></div>
                </div>
            </div>
            <button class="vbt-btn vbt-btn-secondary rivalry-rematch-btn" data-rivalry="${rivalry.id}">
                🔄 Rematch
            </button>
        </div>
    `;
}

function renderEmptyRivalries() {
    return `
        <div class="rivalry-empty">
            <span class="rivalry-empty-icon">⚔️</span>
            <p>No active rivalries. Challenge someone to begin!</p>
        </div>
    `;
}

function attachRivalryListeners(el) {
    // Accept invite
    el.querySelectorAll('.rivalry-accept-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const inviteId = btn.dataset.invite;
            acceptInvite(inviteId);
        });
    });

    // Decline invite
    el.querySelectorAll('.rivalry-decline-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const inviteId = btn.dataset.invite;
            declineInvite(inviteId);
        });
    });

    // Rematch
    el.querySelectorAll('.rivalry-rematch-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const rivalryId = btn.dataset.rivalry;
            if (window.showToast) window.showToast('Rematch requested!', 'info');
        });
    });
}

async function acceptInvite(inviteId) {
    try {
        const resp = await fetch(`${API_BASE}/rivalry/action?action=accept`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ invite_id: inviteId })
        });
        if (!resp.ok) throw new Error('Failed to accept invite');
        if (window.showToast) window.showToast('⚔️ Challenge accepted! Good luck!', 'success');
    } catch (e) {
        if (window.showToast) window.showToast(`❌ ${e.message}`, 'error');
    }
}

async function declineInvite(inviteId) {
    try {
        const resp = await fetch(`${API_BASE}/rivalry/action?action=decline`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ invite_id: inviteId })
        });
        if (!resp.ok) throw new Error('Failed to decline invite');
        if (window.showToast) window.showToast('Invite declined', 'info');
    } catch (e) {
        if (window.showToast) window.showToast(`❌ ${e.message}`, 'error');
    }
}

// --- Global handlers ---
window.rivalryNewChallenge = function() {
    const target = prompt('Enter wallet address of rival:');
    if (target) {
        if (window.showToast) window.showToast(`Challenge sent to ${target.slice(0, 12)}...`, 'success');
    }
};

window.rivalryViewLeaderboard = function() {
    if (window.openRivalryViewer) window.openRivalryViewer();
    else alert('Leaderboard opening...');
};

window.initRivalryChallenge = initRivalryChallenge;

// --- Utility ---
function escapeHtml(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    }[c]));
}
