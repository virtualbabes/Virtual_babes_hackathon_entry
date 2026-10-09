// ============================================================================
// stat_overlay.js — Stat Overlay System (power buff across ALL entities)
// ----------------------------------------------------------------------------
// The stat overlay intercepts/overlays a power buff across all entities.
// Related to 3D world functionality and entity power combinations.
// Every real-world entity has them as base stats.
//
// EXPANDABLE: New entity types auto-integrate via server enumeration.
// WASM SPLIT: All logic runs in server; this is the UI layer only.
// VOI FAUCET: All rewards flow to l.faucetBalanceMicro via backend.
// ============================================================================
(function () {
    'use strict';

    var API_BASE = '/api';

    let overlayEl = null;
    let leaderboardEl = null;
    let statusEl = null;

    function init() {
        if (overlayEl) return;
        const html = `
<div id="stat-overlay-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel stat-overlay-panel">
        <button id="btn-close-so" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>⚡ Stat Overlay System</h2>
        <p style="font-size:12px;color:#b0bec5;">Power buff across ALL entities. Related to 3D world functionality.</p>

        <div class="so-tabs">
            <button class="so-tab active" data-tab="leaderboard">Leaderboard</button>
            <button class="so-tab" data-tab="my-entities">My Entities</button>
            <button class="so-tab" data-tab="region">Region</button>
        </div>

        <div id="so-leaderboard" class="so-leaderboard"></div>
        <div id="so-entities" class="so-entities" style="display:none;"></div>
        <div id="so-region" class="so-region" style="display:none;"></div>

        <p id="so-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('stat-overlay-overlay');
        leaderboardEl = document.getElementById('so-leaderboard');
        statusEl = document.getElementById('so-status');

        document.getElementById('btn-close-so').addEventListener('click', () => { overlayEl.style.display = 'none'; });

        document.querySelectorAll('.so-tab').forEach(btn => {
            btn.addEventListener('click', () => {
                document.querySelectorAll('.so-tab').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                switchSoTab(btn.dataset.tab);
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

    function switchSoTab(tab) {
        document.querySelectorAll('.so-tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab)); document.getElementById('so-leaderboard').style.display = tab === 'leaderboard' ? 'block' : 'none';
        document.getElementById('so-entities').style.display = tab === 'my-entities' ? 'block' : 'none';
        document.getElementById('so-region').style.display = tab === 'region' ? 'block' : 'none';
        if (tab === 'leaderboard') loadLeaderboard();
        if (tab === 'my-entities') loadMyEntities();
        if (tab === 'region') loadRegion();
    }

    async function loadLeaderboard() {
        try {
            const res = await api('/api/stat-overlay/leaderboard');
            renderLeaderboard(res.leaderboard || []);
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    async function loadMyEntities() {
        try {
            const res = await api('/api/stat-overlay/owner');
            renderEntities(res.overlays || []);
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    async function loadRegion() {
        try {
            const res = await api('/api/stat-overlay/region?region=Base');
            renderEntities(res.overlays || []);
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    function renderLeaderboard(overlays) {
        const el = document.getElementById('so-leaderboard');
        if (!el) return;
        if (!(overlays?.length ?? 0)) {
            el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No entities yet.</p>';
            return;
        }
        el.innerHTML = overlays.map((o, i) => `
            <div class="so-entity">
                <span class="so-rank">#${i + 1}</span>
                <span class="so-name">${esc(o.entity_id.slice(0, 12))}…</span>
                <span class="so-type">${esc(o.entity_type)}</span>
                <span class="so-power">⚡ ${o.effective_power}</span>
                <span class="so-dominant">${esc(o.dominant_stat)}</span>
                <span class="so-buff">${o.power_buff >= 0 ? '+' : ''}${o.power_buff}</span>
            </div>
        `).join('');
    }

    function renderEntities(overlays) {
        const el = document.getElementById('so-entities') || document.getElementById('so-region');
        if (!el) return;
        if (!(overlays?.length ?? 0)) {
            el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No entities found.</p>';
            return;
        }
        el.innerHTML = overlays.map(o => `
            <div class="so-entity">
                <span class="so-name">${esc(o.entity_id.slice(0, 12))}…</span>
                <span class="so-type">${esc(o.entity_type)}</span>
                <span class="so-power">⚡ ${o.effective_power}</span>
                <span class="so-stats">
                    <span>SPD:${o.stats.speed||0}</span>
                    <span>INT:${o.stats.intelligence||0}</span>
                    <span>WIL:${o.stats.willpower||0}</span>
                    <span>STR:${o.stats.strength||0}</span>
                    <span>CHA:${o.stats.charisma||0}</span>
                    <span>AGI:${o.stats.agility||0}</span>
                </span>
                <span class="so-earn">${o.can_earn_vbv ? '💰' : ''} ${o.can_earn_stats ? '📊' : ''}</span>
            </div>
        `).join('');
    }

    function setStatus(m) { if (statusEl) statusEl.textContent = m || ''; }

    window.openStatOverlay = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        loadLeaderboard();
    };
})();
