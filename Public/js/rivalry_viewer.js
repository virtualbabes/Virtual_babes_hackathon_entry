// Rivalry Matrix Viewer (§25.10) — menu-mode panel. Pure UI over /api/rivalry/list,
// /api/rivalry/detect, /api/rivalry/resolve. Mirrors ai_citizens.js conventions.
(function () {
    'use strict';

    var API_BASE = '/api';
    let state = { rivalries: [], _init: false };
    let overlayEl = null, gridEl = null, statusEl = null;

    function init() {
        if (state._init) return;
        state._init = true;
        const html = `
<div id="rivalry-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel rivalry-panel">
        <button id="btn-close-rivalry" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>⚔️ Region / Territory Rivalries (§25.10)</h2>

        <div class="rv-tabs">
            <button class="rv-tab active" data-tab="active" onclick="window.switchRVTab('active')">Active Rivalries</button>
            <button class="rv-tab" data-tab="detect" onclick="window.switchRVTab('detect')">Detect Rivalries</button>
        </div>

        <div id="rv-panel-active" class="rv-panel">
            <h3>Active Rivalries</h3>
            <div id="rv-grid" class="rv-grid"></div>
        </div>

        <div id="rv-panel-detect" class="rv-panel hidden">
            <h3>Detect Rivalries</h3>
            <p style="color:#b0bec5;font-size:12px;">Asset-type + citizen-value keyed. Capital-vs-Capital across regions; Territory-vs-Territory inside.</p>
            <div class="rivalry-actions">
                <button id="rv-detect-btn" class="vbt-btn vbt-btn-primary">Detect Rivalries</button>
            </div>
        </div>

        <p id="rv-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('rivalry-overlay');
        gridEl = document.getElementById('rv-grid');
        statusEl = document.getElementById('rv-status');
        document.getElementById('btn-close-rivalry').addEventListener('click', () => { overlayEl.style.display = 'none'; });
        document.getElementById('rv-detect-btn').addEventListener('click', onDetect);
    }

    function switchRVTab(tab) {
        document.querySelectorAll('.rv-tab').forEach(t => t.classList.toggle('active', t.dataset.tab === tab));
        document.querySelectorAll('.rv-panel').forEach(p => p.classList.add('hidden'));
        const panel = document.getElementById('rv-panel-' + tab);
        if (panel) panel.classList.remove('hidden');
    }

    function setStatus(m) { if (statusEl) statusEl.textContent = m || ''; }

    async function api(path, opts) {
        const url = path.startsWith('/api/') ? path : API_BASE + path;
        const resp = await fetch(url, opts);
        if (!resp.ok) {
            let d = '';
            try { d = (await resp.json()).error || ''; } catch (e) {}
            throw new Error(d || ('HTTP ' + resp.status));
        }
        return resp.json();
    }

    async function load() {
        try {
            const data = await api('/rivalry/list');
            state.rivalries = Array.isArray(data) ? data : (data.data || []);
            render();
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    async function onDetect() {
        try {
            const d = await api('/rivalry/detect', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' });
            setStatus('Detected ' + (d.created || 0) + ' new rivalries.');
            load();
        } catch (e) { setStatus('Detect failed: ' + e.message); }
    }

    async function resolve(id) {
        try {
            await api('/rivalry/resolve', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ rivalry_id: id }) });
            setStatus('Resolved ' + id.slice(0, 8));
            load();
        } catch (e) { setStatus('Resolve failed: ' + e.message); }
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }

    function render() {
        if (!gridEl) return;
        if (!(state.rivalries?.length ?? 0)) { gridEl.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No rivalries yet. Hit "Detect Rivalries".</p>'; return; }
        gridEl.innerHTML = state.rivalries.map(r => {
            const resolved = !!r.resolved_at;
            const winner = r.winner ? r.winner.slice(0, 8) : '—';
            return `<div class="rv-card ${resolved ? 'rv-resolved' : ''}">
                <div class="rv-head"><span class="rv-lvl">${esc(r.level)}</span><span class="rv-class">${esc(r.asset_class)}</span></div>
                <div class="rv-sides"><span>${esc(r.side_a)}</span><span class="rv-vs">vs</span><span>${esc(r.side_b)}</span></div>
                <div class="rv-score">${r.score_a || 0} — ${r.score_b || 0}</div>
                <div class="rv-foot">Winner: ${winner}${r.declared ? ' · declared' : ''}</div>
                ${resolved ? '' : `<button class="vbt-btn vbt-btn-primary" onclick="window._rvResolve('${esc(r.rivalry_id)}')">Resolve</button>`}
            </div>`;
        }).join('');
    }

    window.openRivalryViewer = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        load();
    };
    window.switchRVTab = switchRVTab;
    window._rvResolve = resolve;

    // WS consumer (flow-doc §12.3): live-refresh the rivalry grid when its panel is open.
    window.onRivalryUpdate = function () {
        const el = document.getElementById('rivalry-overlay');
        if (el && el.style.display !== 'none') load();
    };
})();
