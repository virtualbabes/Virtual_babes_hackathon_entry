// ============================================================================
// tournament_brackets.js — Tournament Brackets & Registration
// ----------------------------------------------------------------------------
// Exposes tournament_manager.go: brackets, registration, live spectate, history
// Unique visual style: bracket tree layout with live indicators
// ============================================================================

var API_BASE = '/api';

// --- Tournament State ---
const TOURNAMENT_STATUS = {
    INACTIVE: 'inactive',
    REGISTERING: 'registering',
    ACTIVE: 'active',
    FINALIZING: 'finalizing',
};

// --- State ---
let tournamentData = null;
let bracketData = null;

// --- Init ---
export function initTournamentBrackets() {
    const el = document.getElementById('wd-tournament');
    if (!el) return;
    renderTournamentLoading(el);
    fetchTournamentData(el);
}

async function fetchTournamentData(el) {
    try {
        const resp = await fetch(`${API_BASE}/tournament/history`);
        if (!resp.ok) throw new Error('Failed to fetch tournament data');
        tournamentData = await resp.json();
        renderTournamentBrackets(el);
    } catch (e) {
        el.innerHTML = '<p style="color:#90a4ae;">Tournament data unavailable</p>';
    }
}

function renderTournamentLoading(el) {
    el.innerHTML = '<div class="wd-loading">Loading tournament brackets…</div>';
}

function renderTournamentBrackets(el) {
    const data = tournamentData || {};
    const tournaments = data.tournaments || [];
    const active = tournaments.find(t => t.status === 'active') || null;
    const upcoming = tournaments.filter(t => t.status === 'registering') || [];
    const past = tournaments.filter(t => t.status === 'completed') || [];

    let html = `
        <div class="tournament-brackets-container">
            <!-- Header -->
            <div class="tournament-header">
                <h3 style="color:#e5e7eb;margin:0;">🏆 Tournament Brackets</h3>
                <span class="tournament-live-badge ${active ? 'live' : 'offline'}">
                    ${active ? '● LIVE' : '○ OFFLINE'}
                </span>
            </div>

            <!-- Active Tournament -->
            ${active ? renderActiveTournament(active) : renderNoActiveTournament()}

            <!-- Registration Open -->
            ${upcoming.length > 0 ? renderRegistrationOpen(upcoming) : ''}

            <!-- Bracket Visualization -->
            ${active?.matches ? renderBracketTree(active.matches) : ''}

            <!-- Tournament History -->
            ${past.length > 0 ? renderTournamentHistory(past) : ''}
        </div>
    `;

    el.innerHTML = html;
    attachTournamentListeners(el);
}

function renderActiveTournament(tournament) {
    return `
        <div class="tournament-active-card">
            <div class="tournament-active-header">
                <div class="tournament-active-info">
                    <span class="tournament-name">${escapeHtml(tournament.name || 'Arena Championship')}</span>
                    <span class="tournament-round">Round ${tournament.current_round || 1}</span>
                </div>
                <div class="tournament-pot">
                    <span class="tournament-pot-label">POT</span>
                    <span class="tournament-pot-value">${formatVBV(tournament.pot_micro || 0)}</span>
                </div>
            </div>
            <div class="tournament-participants">
                <span class="tournament-participants-count">${tournament.participants || 0} participants</span>
                <span class="tournament-buy-in">Buy-in: ${formatVBV(tournament.buy_in_micro || 0)}</span>
            </div>
            <div class="tournament-actions">
                <button class="vbt-btn vbt-btn-primary" onclick="window.tournamentSpectate()">
                    👁 Spectate
                </button>
                <button class="vbt-btn vbt-btn-secondary" onclick="window.tournamentViewBracket()">
                    📊 View Bracket
                </button>
            </div>
        </div>
    `;
}

function renderNoActiveTournament() {
    return `
        <div class="tournament-inactive-card">
            <div class="tournament-inactive-icon">🏟</div>
            <h4>No Active Tournament</h4>
            <p>The arena is quiet. A new tournament will begin soon.</p>
            <div class="tournament-countdown">
                <span class="tournament-countdown-label">Next tournament in:</span>
                <span class="tournament-countdown-timer" id="tournament-countdown">--:--:--</span>
            </div>
        </div>
    `;
}

function renderRegistrationOpen(tournaments) {
    return `
        <div class="tournament-registration-section">
            <h4 style="color:#e5e7eb;">📝 Registration Open</h4>
            <div class="tournament-registration-grid">
                ${tournaments.map(t => `
                    <div class="tournament-registration-card" data-tournament-id="${t.id || t.tournament_id}">
                        <div class="tournament-registration-header">
                            <span class="tournament-registration-name">${escapeHtml(t.name || 'Tournament')}</span>
                            <span class="tournament-registration-status status-open">OPEN</span>
                        </div>
                        <div class="tournament-registration-details">
                            <span class="tournament-detail">💰 Buy-in: ${formatVBV(t.buy_in_micro || 0)}</span>
                            <span class="tournament-detail">👥 ${(t.participants || 0)} / ${t.max_participants || 16}</span>
                            <span class="tournament-detail">🏆 Pot: ${formatVBV(t.pot_micro || 0)}</span>
                        </div>
                        <button class="vbt-btn vbt-btn-primary tournament-register-btn">
                            Register Now
                        </button>
                    </div>
                `).join('')}
            </div>
        </div>
    `;
}

function renderBracketTree(matches) {
    if (!matches || matches.length === 0) return '';

    // Group matches by round
    const rounds = {};
    matches.forEach(m => {
        const round = m.round || 1;
        if (!rounds[round]) rounds[round] = [];
        rounds[round].push(m);
    });

    const roundNames = ['Round of 16', 'Quarter-Finals', 'Semi-Finals', 'Finals'];

    return `
        <div class="tournament-bracket-section">
            <h4 style="color:#e5e7eb;">📊 Bracket</h4>
            <div class="tournament-bracket-scroll">
                <div class="tournament-bracket">
                    ${Object.entries(rounds).sort(([a], [b]) => a - b).map(([round, roundMatches]) => `
                        <div class="tournament-round">
                            <div class="tournament-round-header">
                                ${roundNames[round - 1] || `Round ${round}`}
                            </div>
                            <div class="tournament-round-matches">
                                ${roundMatches.map(m => `
                                    <div class="tournament-match ${m.winner ? 'match-complete' : 'match-pending'}">
                                        <div class="tournament-match-player ${m.winner === m.p1 ? 'player-winner' : ''} ${!m.p1 ? 'player-empty' : ''}">
                                            <span class="player-name">${m.p1 ? escapeHtml(m.p1.slice(0, 8) + '...') : 'TBD'}</span>
                                            <span class="player-score">${m.score_p1 !== undefined ? m.score_p1 : '-'}</span>
                                        </div>
                                        <div class="tournament-match-divider"></div>
                                        <div class="tournament-match-player ${m.winner === m.p2 ? 'player-winner' : ''} ${!m.p2 ? 'player-empty' : ''}">
                                            <span class="player-name">${m.p2 ? escapeHtml(m.p2.slice(0, 8) + '...') : 'TBD'}</span>
                                            <span class="player-score">${m.score_p2 !== undefined ? m.score_p2 : '-'}</span>
                                        </div>
                                    </div>
                                `).join('')}
                            </div>
                        </div>
                    `).join('')}
                </div>
            </div>
        </div>
    `;
}

function renderTournamentHistory(tournaments) {
    return `
        <div class="tournament-history-section">
            <h4 style="color:#e5e7eb;">📜 Recent Tournaments</h4>
            <div class="tournament-history-list">
                ${tournaments.slice(0, 5).map(t => `
                    <div class="tournament-history-item">
                        <div class="tournament-history-info">
                            <span class="tournament-history-name">${escapeHtml(t.name || 'Tournament')}</span>
                            <span class="tournament-history-winner">🏆 ${t.winner ? escapeHtml(t.winner.slice(0, 12) + '...') : 'Unknown'}</span>
                        </div>
                        <span class="tournament-history-pot">${formatVBV(t.pot_micro || 0)}</span>
                    </div>
                `).join('')}
            </div>
        </div>
    `;
}

function attachTournamentListeners(el) {
    // Register buttons
    el.querySelectorAll('.tournament-register-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const card = btn.closest('.tournament-registration-card');
            const tournamentId = card?.dataset.tournamentId;
            registerForTournament(tournamentId);
        });
    });
}

async function registerForTournament(tournamentId) {
    if (!tournamentId) {
        alert('Please select a tournament');
        return;
    }
    try {
        const resp = await fetch(`${API_BASE}/tournament/register`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ tournament_id: tournamentId })
        });
        if (!resp.ok) throw new Error('Registration failed');
        if (window.showToast) window.showToast('✅ Registered for tournament!', 'success');
    } catch (e) {
        if (window.showToast) window.showToast(`❌ ${e.message}`, 'error');
    }
}

// --- Global handlers ---
window.tournamentSpectate = function() {
    if (window.sendSpectate) window.sendSpectate('tournament');
    else alert('Spectate mode loading...');
};

window.tournamentViewBracket = function() {
    const modal = document.createElement('div');
    modal.className = 'tournament-bracket-modal';
    modal.innerHTML = renderBracketModal();
    document.body.appendChild(modal);
    attachBracketModalListeners(modal);
};

function renderBracketModal() {
    const data = tournamentData || {};
    const tournaments = data.tournaments || [];
    const active = tournaments.find(t => t.status === 'active') || null;
    const matches = active?.matches || [];

    // Group matches by round
    const rounds = {};
    matches.forEach(m => {
        const round = m.round || 1;
        if (!rounds[round]) rounds[round] = [];
        rounds[round].push(m);
    });

    const roundNames = ['Round of 16', 'Quarter-Finals', 'Semi-Finals', 'Finals'];

    return `
        <div class="bracket-modal-overlay">
            <div class="bracket-modal">
                <div class="bracket-modal-header">
                    <h3>🏆 ${escapeHtml(active?.name || 'Arena Championship')} — Full Bracket</h3>
                    <button class="bracket-modal-close" onclick="this.closest('.tournament-bracket-modal').remove()">✕</button>
                </div>
                <div class="bracket-modal-body">
                    <div class="bracket-scroll">
                        <div class="bracket-tree">
                            ${Object.entries(rounds).sort(([a], [b]) => a - b).map(([round, roundMatches]) => `
                                <div class="bracket-round">
                                    <div class="bracket-round-header">${roundNames[round - 1] || `Round ${round}`}</div>
                                    <div class="bracket-round-matches">
                                        ${roundMatches.map(m => `
                                            <div class="bracket-match ${m.winner ? 'match-complete' : 'match-pending'}" data-match-id="${m.id || ''}">
                                                <div class="bracket-match-player ${m.winner === m.p1 ? 'player-winner' : ''} ${!m.p1 ? 'player-empty' : ''}">
                                                    <span class="player-name">${m.p1 ? escapeHtml(m.p1.slice(0, 8) + '...') : 'TBD'}</span>
                                                    <span class="player-score">${m.score_p1 !== undefined ? m.score_p1 : '-'}</span>
                                                </div>
                                                <div class="bracket-match-divider"></div>
                                                <div class="bracket-match-player ${m.winner === m.p2 ? 'player-winner' : ''} ${!m.p2 ? 'player-empty' : ''}">
                                                    <span class="player-name">${m.p2 ? escapeHtml(m.p2.slice(0, 8) + '...') : 'TBD'}</span>
                                                    <span class="player-score">${m.score_p2 !== undefined ? m.score_p2 : '-'}</span>
                                                </div>
                                            </div>
                                        `).join('')}
                                    </div>
                                </div>
                            `).join('')}
                        </div>
                    </div>
                    <div class="bracket-details" id="bracket-details">
                        <p style="color:#90a4ae;">Click a match to view details</p>
                    </div>
                </div>
            </div>
        </div>
    `;
}

function attachBracketModalListeners(modal) {
    // Match click handlers
    modal.querySelectorAll('.bracket-match').forEach(match => {
        match.addEventListener('click', () => {
            const matchId = match.dataset.matchId;
            showMatchDetails(matchId);
        });
    });
}

function showMatchDetails(matchId) {
    const details = document.getElementById('bracket-details');
    if (!details) return;
    details.innerHTML = `<p style="color:#90a4ae;">Match ${matchId} details loading...</p>`;
}

window.initTournamentBrackets = initTournamentBrackets;

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
