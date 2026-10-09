// Leaderboard Region / Mode-Switch Hub (§25.9) — menu-mode panel.
// The leaderboard region is the neutral cross-mode lobby: entering it presents the
// switch between the 3D explorer and the menu world. This panel is the menu-side hub.
(function () {
    'use strict';

    var API_BASE = '/api';
    let state = { standings: [], _init: false };
    let overlayEl = null, standingsEl = null, statusEl = null;

    function init() {
        if (state._init) return;
        state._init = true;
        const html = `
<div id="leaderboard-region-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel leaderboard-region-panel">
        <button id="btn-close-lb-region" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>🏆 Leaderboard Region — The Gathering Hub (§25.9)</h2>

        <div class="lb-tabs">
            <button class="lb-tab active" data-tab="standings" onclick="window.switchLBTab('standings')">Top Standings</button>
            <button class="lb-tab" data-tab="switch" onclick="window.switchLBTab('switch')">World Switch</button>
        </div>

        <div id="lb-panel-standings" class="lb-panel">
            <h3>Top Standings</h3>
            <div id="lb-region-standings" class="lb-grid"></div>
        </div>

        <div id="lb-panel-switch" class="lb-panel hidden">
            <h3>World Switch</h3>
            <button id="lb-enter-3d" class="vbt-btn vbt-btn-primary">🌐 Enter 3D World</button>
            <button id="lb-enter-menu" class="vbt-btn vbt-btn-secondary">🗂️ Menu World</button>
            <p style="color:#90a4ae;font-size:11px;margin-top:8px;">(3D client hooks: window.enter3DWorld / window.enterMenuWorld)</p>
        </div>

        <p id="lb-region-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('leaderboard-region-overlay');
        standingsEl = document.getElementById('lb-region-standings');
        statusEl = document.getElementById('lb-region-status');
        document.getElementById('btn-close-lb-region').addEventListener('click', () => { overlayEl.style.display = 'none'; });
        document.getElementById('lb-enter-3d').addEventListener('click', () => {
            if (window.enter3DWorld) window.enter3DWorld();
            else setStatus('3D client not mounted yet — hub ready.');
        });
        document.getElementById('lb-enter-menu').addEventListener('click', () => {
            if (window.enterMenuWorld) window.enterMenuWorld();
            else setStatus('Already in menu world.');
        });
    }

    function switchLBTab(tab) {
        document.querySelectorAll('.lb-tab').forEach(t => t.classList.toggle('active', t.dataset.tab === tab));
        document.querySelectorAll('.lb-panel').forEach(p => p.classList.add('hidden'));
        const panel = document.getElementById('lb-panel-' + tab);
        if (panel) panel.classList.remove('hidden');
    }

    function setStatus(m) { if (statusEl) statusEl.textContent = m || ''; }

    async function load() {
        try {
            const resp = await fetch(API_BASE + '/leaderboard');
            const players = await resp.json();
            state.standings = Array.isArray(players) ? players : [];
            render();
        } catch (e) {
            // Fallback: reuse existing leaderboard.js fetcher if present.
            setStatus('Standings unavailable.');
        }
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }

    function render() {
        if (!standingsEl) return;
        if (!(state.standings?.length ?? 0)) { standingsEl.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No standings yet.</p>'; return; }
        standingsEl.innerHTML = state.standings.slice(0, 20).map((p, i) => {
            const name = p.wallet ? p.wallet.slice(0, 10) : ('#' + i);
            const rep = p.reputation != null ? p.reputation : (p.Reputation != null ? p.Reputation : 0);
            return `<div class="lb-row"><span class="lb-rank">${i + 1}</span><span class="lb-name">${esc(name)}</span><span class="lb-rep">${rep}</span></div>`;
        }).join('');
    }

    window.openLeaderboardRegion = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        load();
    };
    window.switchLBTab = switchLBTab;
})();
