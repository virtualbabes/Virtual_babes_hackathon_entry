// ============================================================================
// season_countdown.js — Season Status & Countdown
// ----------------------------------------------------------------------------
// Exposes season_engine.go: season status, countdown, rewards, history
// Unique visual style: celestial/calendar with orbiting elements
// ============================================================================

var API_BASE = '/api';

// --- State ---
// `history` is `{seasons: <number>}` — /api/season/history returns the CURRENT season NUMBER
// (lobby_manager.go handleSeasonHistory → l.seasonNumber), not a list. `events` is the ARRAY that
// /api/season/status actually returns: [{event, reward_pool}]. The panel used to read
// `season_number` / `status` / `ends_at` / `rewards` off that response, none of which it has ever
// contained, so it showed a hard-coded "Season 1 · Active" with a timer that never started.
let seasonData = null;
let seasonHistory = null;
let seasonEvents = [];
let seasonProblems = [];

// --- Init ---
export function initSeasonCountdown() {
    const el = document.getElementById('wd-season');
    if (!el) return;
    renderSeasonLoading(el);
    fetchSeasonData(el);
}

async function fetchSeasonData(el) {
    seasonProblems = [];
    try {
        seasonHistory = await api('/season/history');
    } catch (e) { seasonHistory = null; seasonProblems.push('history: ' + e.message); }
    try {
        const res = await api('/season/status');
        // The route answers a bare ARRAY. Keep the legacy `{rewards}` tolerance so a future
        // envelope cannot silently blank the panel.
        seasonEvents = Array.isArray(res) ? res : (Array.isArray(res.events) ? res.events : []);
    } catch (e) { seasonEvents = []; seasonProblems.push('status: ' + e.message); }
    seasonData = seasonEvents.length ? seasonEvents[0].event : null;
    renderSeasonCountdown(el);
}

// fetch uses API_BASE = '/api'; `api()` prefixes only when the path is not already absolute.
async function api(path, opts) {
    const url = path.startsWith('/api/') ? path : API_BASE + path;
    const resp = await fetch(url, opts);
    if (!resp.ok) { let d = ''; try { d = (await resp.json()).error || ''; } catch (e) {} throw new Error(d || ('HTTP ' + resp.status)); }
    return resp.json();
}

function renderSeasonLoading(el) {
    el.innerHTML = '<div class="wd-loading">Loading season…</div>';
}

function renderSeasonCountdown(el) {
    // The engine's own season number, or a stated absence. Never a defaulted "1".
    const season = (seasonHistory && Number.isFinite(Number(seasonHistory.seasons))) ? Number(seasonHistory.seasons) : null;

    const eventRows = seasonEvents.length
        ? seasonEvents.map(row => {
            const ev = row.event || {};
            const pool = row.reward_pool || {};
            return `
                <div class="season-reward-card">
                    <span class="season-reward-rank">${escapeHtml(ev.name || ev.event_id || 'Event')}</span>
                    <span class="season-reward-name">${escapeHtml(ev.status || '')}</span>
                    <span class="season-reward-amount">${pool.total_allocated ? formatVBV(pool.total_allocated) : 'no pool reported'}</span>
                </div>`;
        }).join('')
        : '<p class="season-note">The engine reports no active seasonal events.</p>';

    el.innerHTML = `
        <div class="season-countdown-container">
            <div class="season-header">
                <div class="season-celestial">
                    <div class="season-orbit">
                        <div class="season-planet"></div>
                        <div class="season-moon"></div>
                    </div>
                </div>
                <div class="season-info">
                    <h3 style="color:#e5e7eb;margin:0;">${season === null ? 'Season — not reported' : 'Season ' + season}</h3>
                    <span class="season-status active">${seasonEvents.length} active event${seasonEvents.length === 1 ? '' : 's'}</span>
                </div>
            </div>

            <div class="season-rewards-section">
                <h4 style="color:#e5e7eb;">🎯 Active Seasonal Events</h4>
                <div class="season-rewards-grid">${eventRows}</div>
            </div>

            <div class="season-progress-section">
                <h4 style="color:#e5e7eb;">🏆 Claim Season Reward</h4>
                <!-- /api/season/events/reward had NO owner anywhere in the client: it is the only way
                     to claim a seasonal payout (HandleClaimSeasonReward). Placed here — claiming a
                     season reward IS this leaf's domain. The body carries only the wallet. -->
                <div class="season-claim-row">
                    <button class="vbt-btn vbt-btn-primary" id="season-claim-btn">Claim</button>
                    <span class="season-note" id="season-claim-status"></span>
                </div>
                <p class="season-note">${seasonProblems.length ? 'Could not read — ' + escapeHtml(seasonProblems.join('; ')) : ''}</p>
            </div>
        </div>
    `;

    const claimBtn = document.getElementById('season-claim-btn');
    if (claimBtn) claimBtn.addEventListener('click', async () => {
        const status = document.getElementById('season-claim-status');
        const wallet = window.currentWallet || (window.getActiveWallet && window.getActiveWallet()) || '';
        if (!wallet) { if (status) status.textContent = 'Connect a wallet first.'; return; }
        try {
            const res = await api('/season/events/reward', {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ wallet }),
            });
            // The payout amount comes back from the engine; it is never computed here.
            if (status) status.textContent = res.reward_micro
                ? 'Claimed ' + formatVBV(res.reward_micro) + ' on ' + (res.event_id || 'the season')
                : 'The engine granted no reward for this wallet.';
            fetchSeasonData(el);
        } catch (e) { if (status) status.textContent = 'Claim refused: ' + e.message; }
    });
}

// --- Global handlers ---
window.initSeasonCountdown = initSeasonCountdown;

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
