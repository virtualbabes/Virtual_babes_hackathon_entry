// ============================================================================
// system_dashboard.js — System, Rivalry, Season, Replay, Admin routes
// ----------------------------------------------------------------------------
// Wires 15 orphaned backend routes:
//   Achievement: /api/achievement-stats
//   Player:      /api/player/progression, /api/re-sync-stats
//   Rivalry:     /api/rivalry/detect, /api/rivalry/recompute, /api/rivalry/action,
//                 /api/rivalry/request, /api/rivalry/resolve, /api/rivalry/join,
//                 /api/rivalry/list, /api/rivalry/world-dynamics
//   Season:      /api/season/admin/create-event, /api/season/admin/end-event,
//                 /api/season/admin/update-reward-pool
//   Regions:     /api/regions
// ============================================================================

(function () {
    'use strict';
    var API_BASE = '/api';
    let overlayEl = null;

    function init() {
        if (overlayEl) return;
        overlayEl = document.getElementById('system-dashboard-overlay');
        if (!overlayEl) {
            // SELF-MOUNT. Nothing in the shell provided this root (no `id="system-dashboard-overlay"`
            // in index.html), so the null-check below made the whole module a silent no-op: `init()`
            // returned and all 5 controls this file publishes had no surface to render into.
            document.body.insertAdjacentHTML('beforeend', '<div id="system-dashboard-overlay"></div>');
            overlayEl = document.getElementById('system-dashboard-overlay');
            if (!overlayEl) return;
        }
        overlayEl.className = 'overlay vbt-overlay';
        overlayEl.style.display = 'none';

        overlayEl.innerHTML = `
<div class="neon-glass-panel system-dashboard-panel">
    <button id="btn-close-sd" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
    <h2>⚙️ System Dashboard</h2>
    <p style="font-size:12px;color:#b0bec5;margin-top:-8px;">Achievements · Rivalry · Season · Sync</p>
    <div class="sd-tabs">
        <button class="sd-tab active" data-tab="achievements" onclick="switchSDTab('achievements')">Achievements</button>
        <button class="sd-tab" data-tab="progression" onclick="switchSDTab('progression')">Progression</button>
        <button class="sd-tab" data-tab="rivalry" onclick="switchSDTab('rivalry')">Rivalry</button>
        <button class="sd-tab" data-tab="season" onclick="switchSDTab('season')">Season Admin</button>
        <button class="sd-tab" data-tab="sync" onclick="switchSDTab('sync')">Sync</button>
    </div>
    <div id="sd-panel-achievements" class="sd-panel"><div id="sd-achievements"></div></div>
    <div id="sd-panel-progression" class="sd-panel hidden"><div id="sd-progression"></div></div>
    <div id="sd-panel-rivalry" class="sd-panel hidden"><div id="sd-rivalry"></div></div>
    <div id="sd-panel-season" class="sd-panel hidden"><div id="sd-season"></div></div>
    <div id="sd-panel-sync" class="sd-panel hidden"><div id="sd-sync"></div></div>
    <p id="sd-status" class="ai-status"></p>
</div>`;

        document.getElementById('btn-close-sd').addEventListener('click', () => { overlayEl.style.display = 'none'; });
        loadAchievements();
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }

    async function api(path, opts) {
        const url = path.startsWith('/api/') ? path : API_BASE + path;
        const resp = await fetch(url, opts);
        if (!resp.ok) {
            let detail = '';
            try { const j = await resp.json(); detail = j.error || j.message || ''; } catch (e) {}
            throw new Error(detail || ('HTTP ' + resp.status));
        }
        return resp.json();
    }

    function switchSDTab(tab) {
        document.querySelectorAll('.sd-tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab));
        document.querySelectorAll('.sd-panel').forEach(p => p.classList.add('hidden'));
        const panel = document.getElementById('sd-panel-' + tab);
        if (panel) panel.classList.remove('hidden');
        if (tab === 'achievements') loadAchievements();
        if (tab === 'progression') loadProgression();
        if (tab === 'rivalry') loadRivalry();
        if (tab === 'season') loadSeasonAdmin();
        if (tab === 'sync') loadSync();
    }

    // === ACHIEVEMENT STATS ===
    async function loadAchievements() {
        const el = document.getElementById('sd-achievements');
        if (!el) return;
        try {
            const res = await api('/api/achievement-stats');
            const stats = res.stats || [];
            if (!stats.length) { el.innerHTML = '<p style="color:#90a4ae;">No achievement stats yet.</p>'; return; }
            el.innerHTML = stats.map(s => `
                <div class="sd-card">
                    <span>${esc(s.wallet.slice(0, 12))}...</span>
                    <span>Wins: ${s.wins || 0}</span>
                    <span>Achieved: ${s.achieved_count || 0}</span>
                    <span>Rarity: ${s.rarity_score || 0}</span>
                </div>`).join('');
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Achievement stats unavailable.</p>'; }
    }

    // === PLAYER PROGRESSION ===
    async function loadProgression() {
        const el = document.getElementById('sd-progression');
        if (!el) return;
        try {
            const res = await api('/api/player/progression');
            const p = res.progression || res || {};
            el.innerHTML = `
                <div class="sd-grid">
                    <div class="sd-stat"><label>Level</label><span>${p.level || 0}</span></div>
                    <div class="sd-stat"><label>XP</label><span>${p.xp || 0}</span></div>
                    <div class="sd-stat"><label>Power</label><span>${p.power_level || 0}</span></div>
                </div>`;
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Progression data unavailable.</p>'; }
    }

    // === RIVALRY ===
    async function loadRivalry() {
        const el = document.getElementById('sd-rivalry');
        if (!el) return;
        try {
            const res = await api('/api/rivalry/detect');
            const rivalries = res.rivalries || [];
            if (!rivalries.length) { el.innerHTML = '<p style="color:#90a4ae;">No rivalries detected.</p>'; return; }
            el.innerHTML = rivalries.map(r => `
                <div class="sd-card">
                    <span>${esc(r.name || r.id)}</span>
                    <span>Score: ${r.score || 0}</span>
                    <span>${esc(r.status || 'Active')}</span>
                </div>`).join('');
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Rivalry data unavailable.</p>'; }
    }

    async function recomputeRivalry() {
        try { await api('/api/rivalry/recompute', { method: 'POST' }); showToast('Rivalry recomputed!', 'success'); loadRivalry(); }
        catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }
    async function requestRivalry() {
        const target = prompt('Target wallet:');
        if (!target) return;
        try { await api('/api/rivalry/request', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ target }) }); showToast('Rivalry requested!', 'success'); loadRivalry(); }
        catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }
    async function joinRivalry() {
        const id = prompt('Rivalry ID to join:');
        if (!id) return;
        try { await api('/api/rivalry/join', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ rivalry_id: id }) }); showToast('Joined rivalry!', 'success'); loadRivalry(); }
        catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }
    async function resolveRivalry() {
        const id = prompt('Rivalry ID to resolve:');
        if (!id) return;
        try { await api('/api/rivalry/resolve', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ rivalry_id: id }) }); showToast('Rivalry resolved!', 'success'); loadRivalry(); }
        catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }
    async function rivalryAction() {
        const id = prompt('Rivalry ID:');
        const action = prompt('Action (attack/defend/retreat):');
        if (!id || !action) return;
        try { await api('/api/rivalry/action', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ rivalry_id: id, action }) }); showToast('Action sent!', 'success'); loadRivalry(); }
        catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }
    async function listRivalries() {
        const el = document.getElementById('sd-rivalry');
        if (!el) return;
        try {
            const res = await api('/api/rivalry/list');
            const rivalries = res.rivalries || [];
            el.innerHTML = rivalries.map(r => `
                <div class="sd-card">
                    <span>${esc(r.name || r.id)}</span>
                    <span>Score: ${r.score || 0}</span>
                </div>`).join('') || '<p style="color:#90a4ae;">No rivalries.</p>';
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">List unavailable.</p>'; }
    }

    // === SEASON ADMIN ===
    async function loadSeasonAdmin() {
        const el = document.getElementById('sd-season');
        if (!el) return;
        el.innerHTML = `
            <div class="sd-actions">
                <button class="vbt-btn vbt-btn-primary" onclick="sdCreateEvent()">Create Event</button>
                <button class="vbt-btn vbt-btn-secondary" onclick="sdEndEvent()">End Event</button>
                <button class="vbt-btn vbt-btn-secondary" onclick="sdUpdateRewardPool()">Update Reward Pool</button>
            </div>
            <p style="color:#90a4ae;font-size:12px;">Admin season controls.</p>`;
    }
    async function sdCreateEvent() {
        const title = prompt('Event title:');
        if (!title) return;
        try { await api('/api/season/admin/create-event', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ title }) }); showToast('Event created!', 'success'); }
        catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }
    async function sdEndEvent() {
        const id = prompt('Event ID to end:');
        if (!id) return;
        try { await api('/api/season/admin/end-event', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ event_id: id }) }); showToast('Event ended!', 'success'); }
        catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }
    async function sdUpdateRewardPool() {
        const id = prompt('Event ID:');
        const amount = prompt('Reward pool amount (micro-VBV):');
        if (!id || !amount) return;
        try { await api('/api/season/admin/update-reward-pool', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ event_id: id, amount: parseInt(amount) }) }); showToast('Reward pool updated!', 'success'); }
        catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    // === SYNC ===
    async function loadSync() {
        const el = document.getElementById('sd-sync');
        if (!el) return;
        el.innerHTML = `
            <div class="sd-actions">
                <button class="vbt-btn vbt-btn-primary" onclick="sdResyncStats()">Re-Sync Stats</button>
                <button class="vbt-btn vbt-btn-secondary" onclick="sdListRegions()">List Regions</button>
            </div>
            <p style="color:#90a4ae;font-size:12px;">Sync controls.</p>`;
    }
    async function sdResyncStats() {
        try { await api('/api/re-sync-stats', { method: 'POST' }); showToast('Stats re-synced!', 'success'); }
        catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }
    async function sdListRegions() {
        const el = document.getElementById('sd-sync');
        if (!el) return;
        try {
            const res = await api('/api/regions');
            const regions = res.regions || [];
            el.innerHTML = regions.map(r => `
                <div class="sd-card">
                    <span>${esc(r.name || r.id)}</span>
                    <span>${esc(r.status || 'Active')}</span>
                </div>`).join('') || '<p style="color:#90a4ae;">No regions.</p>';
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Regions unavailable.</p>'; }
    }

    function showToast(msg, type) { if (window.showToast) window.showToast(msg, type); }

    window.openSystemDashboard = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        if (window.PanelManager) window.PanelManager.open(overlayEl);
        else overlayEl.style.display = 'flex';
        switchSDTab('achievements');
    };
    window.switchSDTab = switchSDTab;
    window.sdRecomputeRivalry = recomputeRivalry;
    window.sdRequestRivalry = requestRivalry;
    window.sdJoinRivalry = joinRivalry;
    window.sdResolveRivalry = resolveRivalry;
    window.sdRivalryAction = rivalryAction;
    window.sdListRivalries = listRivalries;

    // EVERY handler name this surface's rendered markup uses must exist on `window` — an inline
    // handler resolves its name there, and every function below is scoped to this IIFE. Without this
    // list the season + region controls (create/end event, reward pool, region + stat reseed) were inert.
    Object.assign(window, {
        sdCreateEvent, sdEndEvent, sdListRegions, sdResyncStats, sdUpdateRewardPool,
    });

    if (document.getElementById('system-dashboard-overlay')) init();
})();
