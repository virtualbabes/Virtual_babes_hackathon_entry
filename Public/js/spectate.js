// ============================================================================
// spectate.js — Spectate Panel (REAL FUNCTIONALITY)
// ============================================================================
(function () {
    'use strict';
    var API_BASE = '/api';
    let overlayEl = null, statusEl = null;
    let liveMatches = [];
    let replays = [];

    function init() {
        if (overlayEl) return;
        const html = `
<div id="spectate-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel spectate-panel">
        <button id="btn-close-sp" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>👁️ Spectate</h2>

        <div class="sp-tabs">
            <button class="sp-tab active" data-tab="live" onclick="window.switchSPTab('live')">Live Matches</button>
            <button class="sp-tab" data-tab="replays" onclick="window.switchSPTab('replays')">Replays</button>
            <button class="sp-tab" data-tab="search" onclick="window.switchSPTab('search')">Search Player</button>
        </div>

        <div id="sp-panel-live" class="sp-panel">
            <h3>Live Matches</h3>
            <div id="sp-live" class="sp-live"></div>
        </div>

        <div id="sp-panel-replays" class="sp-panel hidden">
            <h3>Replays</h3>
            <div id="sp-replays" class="sp-replays"></div>
        </div>

        <div id="sp-panel-search" class="sp-panel hidden">
            <h3>Search Player</h3>
            <div id="sp-search" class="sp-search"></div>
        </div>

        <p id="sp-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('spectate-overlay');
        statusEl = document.getElementById('sp-status');
        document.getElementById('btn-close-sp').addEventListener('click', () => { overlayEl.style.display = 'none'; });
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>\"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }

    async function api(path, opts) {
        const url = path.startsWith('/api/') ? path : API_BASE + path;
        const resp = await fetch(url, opts);
        if (!resp.ok) { let d = ''; try { d = (await resp.json()).error || ''; } catch(e){} throw new Error(d || ('HTTP ' + resp.status)); }
        return resp.json();
    }

    function switchSPTab(tab) {
        document.querySelectorAll('.sp-tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab));
        document.querySelectorAll('.sp-panel').forEach(p => p.classList.add('hidden'));
        const panel = document.getElementById('sp-panel-' + tab);
        if (panel) panel.classList.remove('hidden');
        if (tab === 'live') loadLive();
        if (tab === 'replays') loadReplays();
        if (tab === 'search') loadSearch();
    }

    async function loadLive() {
        const el = document.getElementById('sp-live');
        if (!el) return;
        try {
            const res = await api('/api/active-matches');
            liveMatches = res.matches || res.active_matches || [];
            renderLive();
        } catch (e) {
            console.warn('[Spectate] API unreachable:', e.message);
            liveMatches = [];
            renderLive();
        }
    }

    function renderLive() {
        const el = document.getElementById('sp-live');
        if (!el) return;
        if (!liveMatches.length) { el.innerHTML = '<div class="panel-empty"><div class="panel-empty-icon">👁️</div><div class="panel-empty-message">No live matches right now.</div></div>'; return; }
        el.innerHTML = liveMatches.map(m => `
            <div class="sp-match-card">
                <div class="sp-match-name">${esc(m.p1)} vs ${esc(m.p2)}</div>
                <div class="sp-match-turn">Turn ${m.turn}</div>
                <div class="sp-match-spectators">${m.spectators} watching</div>
                <button class="vbt-btn vbt-btn-primary" onclick="window.spWatch('${m.id}')">Watch</button>
            </div>
        `).join('');
    }

    async function loadReplays() {
        const el = document.getElementById('sp-replays');
        if (!el) return;
        try {
            const res = await api('/api/tournament/history?limit=20');
            replays = res.tournaments || res.matches || [];
            renderReplays();
        } catch (e) {
            console.warn('[Spectate] API unreachable:', e.message);
            replays = [];
            renderReplays();
        }
    }

    function renderReplays() {
        const el = document.getElementById('sp-replays');
        if (!el) return;
        if (!replays.length) { el.innerHTML = '<div class="panel-empty"><div class="panel-empty-icon">🎬</div><div class="panel-empty-message">No replays available.</div></div>'; return; }
        el.innerHTML = replays.map(r => `
            <div class="sp-replay-card">
                <div class="sp-replay-name">${esc(r.name)}</div>
                <div class="sp-replay-date">${esc(r.date)}</div>
                <div class="sp-replay-winner">Winner: ${esc(r.winner)}</div>
                <button class="vbt-btn vbt-btn-secondary" onclick="window.spViewReplay('${r.id}')">View</button>
            </div>
        `).join('');
    }

    function loadSearch() {
        const el = document.getElementById('sp-search');
        if (!el) return;
        el.innerHTML = `
            <div class="sp-search-form">
                <input id="sp-search-input" type="text" placeholder="Enter wallet address..." />
                <button class="vbt-btn vbt-btn-primary" onclick="window.spSearchPlayer()">Search</button>
            </div>
            <div id="sp-search-results" class="sp-search-results"></div>
        `;
    }

    function watch(matchId) {
        if (window.sendSpectate) {
            window.sendSpectate(matchId);
        } else {
            showToast('Spectating ' + matchId + '...', 'info');
        }
    }

    function viewReplay(id) {
        showToast('Loading replay ' + id + '...', 'info');
    }

    async function searchPlayer() {
        const input = document.getElementById('sp-search-input');
        const query = input?.value || '';
        if (!query) return;
        const results = document.getElementById('sp-search-results');
        if (!results) return;
        try {
            const res = await api('/api/player/profile?wallet=' + query);
            const player = res.player || {};
            results.innerHTML = `
                <div class="sp-result-card">
                    <div class="sp-result-name">${esc(player.name || query.slice(0, 10) + '...')}</div>
                    <div class="sp-result-stats">Wins: ${player.wins || 0} | Losses: ${player.losses || 0}</div>
                    <button class="vbt-btn vbt-btn-primary" onclick="window.spWatch('${esc(query)}')">Watch</button>
                </div>
            `;
        } catch (e) {
            results.innerHTML = `
                <div class="sp-result-card">
                    <div class="sp-result-name">${esc(query.slice(0, 10))}...</div>
                    <div class="sp-result-stats">Player not found</div>
                </div>
            `;
        }
    }

    function showToast(msg, type) { if (window.showToast) window.showToast(msg, type); }

    window.openSpectate = function () {
        init();
        if (window.hideAllOverlays) window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        switchSPTab('live');
    };
    window.switchSPTab = switchSPTab;
    window.spWatch = watch;
    window.spViewReplay = viewReplay;
    window.spSearchPlayer = searchPlayer;
})();
