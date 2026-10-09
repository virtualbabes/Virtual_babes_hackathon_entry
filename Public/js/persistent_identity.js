// ============================================================================
// persistent_identity.js — Persistent Identity System
// ----------------------------------------------------------------------------
// History creates reputation. Reputation creates opportunity.
// Nothing important should disappear.
// ============================================================================
(function () {
    'use strict';
    var API_BASE = '/api';
    let overlayEl, profileEl, eventsEl, leaderboardEl, statusEl;

    function init() {
        if (overlayEl) return;
        const html = `
<div id="identity-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel identity-panel">
        <button id="btn-close-id" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>🏛️ Persistent Identity</h2>
        <p style="font-size:12px;color:#b0bec5;">History → Reputation → Opportunity. Nothing important disappears.</p>
        <div class="id-tabs">
            <button class="id-tab active" data-tab="profile">Profile</button>
            <button class="id-tab" data-tab="events">History</button>
            <button class="id-tab" data-tab="leaderboard">Leaderboard</button>
        </div>
        <div id="id-profile" class="id-profile"></div>
        <div id="id-events" class="id-events" style="display:none;"></div>
        <div id="id-leaderboard" class="id-leaderboard" style="display:none;"></div>
        <p id="id-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('identity-overlay');
        statusEl = document.getElementById('id-status');
        document.getElementById('btn-close-id').addEventListener('click', () => { overlayEl.style.display = 'none'; });
        document.querySelectorAll('.id-tab').forEach(btn => {
            btn.addEventListener('click', () => {
                document.querySelectorAll('.id-tab').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                switchIdTab(btn.dataset.tab);
            });
        });
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>\"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }

    async function api(path, opts) {
        const url = path.startsWith('/api/') ? path : API_BASE + path;
        const resp = await fetch(url, opts);
        if (!resp.ok) { let d = ''; try { d = (await resp.json()).error || ''; } catch(e){} throw new Error(d || ('HTTP ' + resp.status)); }
        return resp.json();
    }

    function switchIdTab(tab) {
        document.querySelectorAll('.id-tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab)); document.getElementById('id-profile').style.display = tab === 'profile' ? 'block' : 'none';
        document.getElementById('id-events').style.display = tab === 'events' ? 'block' : 'none';
        document.getElementById('id-leaderboard').style.display = tab === 'leaderboard' ? 'block' : 'none';
        if (tab === 'profile') loadProfile();
        if (tab === 'events') loadEvents();
        if (tab === 'leaderboard') loadLeaderboard();
    }

    async function loadProfile() {
        try {
            const wallet = getWallet();
            if (!wallet) { setStatus('Connect a wallet to view your identity profile.'); return; }
            const res = await api('/identity/profile?wallet=' + encodeURIComponent(wallet));
            const p = res.profile;
            const el = document.getElementById('id-profile');
            if (!el) return;
            el.innerHTML = `
                <div class="id-tier ${p.tier.toLowerCase()}">${esc(p.tier)}</div>
                <div class="id-stat"><span>Total Reputation</span><span>${p.total_reputation}</span></div>
                <div class="id-stat"><span>Identity Score</span><span>${p.identity_score}</span></div>
                <div class="id-stat"><span>Best Streak</span><span>${p.best_streak}</span></div>
                <div class="id-stat"><span>Redemptions</span><span>${p.redemptions}</span></div>
                <div class="id-categories">
                    ${Object.entries(p.category_reps || {}).map(([k,v]) => `<span class="id-cat">${esc(k)}: ${v}</span>`).join('')}
                </div>
            `;
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    async function loadEvents() {
        try {
            const wallet = getWallet();
            if (!wallet) { setStatus('Connect a wallet to view your history.'); return; }
            const res = await api('/identity/events?wallet=' + encodeURIComponent(wallet));
            const el = document.getElementById('id-events');
            if (!el) return;
            if (!(res.events?.length ?? 0)) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No history yet.</p>'; return; }
            el.innerHTML = res.events.map(e => `
                <div class="id-event ${esc(e.type)}">
                    <span class="id-event-type">${esc(e.type)}</span>
                    <span class="id-event-title">${esc(e.title)}</span>
                    <span class="id-event-impact">${e.impact >= 0 ? '+' : ''}${e.impact}</span>
                    <span class="id-event-date">${new Date(e.timestamp).toLocaleDateString()}</span>
                </div>
            `).join('');
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    async function loadLeaderboard() {
        try {
            const res = await api('/identity/leaderboard');
            const el = document.getElementById('id-leaderboard');
            if (!el) return;
            if (!res.leaderboard || !(res.leaderboard?.length ?? 0)) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No players yet.</p>'; return; }
            el.innerHTML = res.leaderboard.map((p, i) => `
                <div class="id-leader">
                    <span class="id-rank">#${i + 1}</span>
                    <span class="id-name">${esc(p.wallet.slice(0, 12))}…</span>
                    <span class="id-tier ${p.tier.toLowerCase()}">${esc(p.tier)}</span>
                    <span class="id-rep">${p.total_reputation}</span>
                </div>
            `).join('');
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    function setStatus(m) { if (statusEl) statusEl.textContent = m || ''; }

    function getWallet() {
        try {
            const gs = (typeof window.GetGameState === 'function') ? window.GetGameState() : null;
            return (gs && typeof gs.wallet === 'string' && gs.wallet) ? gs.wallet : '';
        } catch (e) { return ''; }
    }

    window.openPersistentIdentity = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        loadProfile();
    };
})();
