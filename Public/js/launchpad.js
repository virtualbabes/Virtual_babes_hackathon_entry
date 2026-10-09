// ============================================================================
// launchpad.js — Launchpad System (accelerate creators, integrate launches)
// ============================================================================
(function () {
    'use strict';
    var API_BASE = '/api';
    let overlayEl, launchesEl, myLaunchesEl, statusEl;

    function init() {
        if (overlayEl) return;
        const html = `
<div id="launchpad-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel launchpad-panel">
        <button id="btn-close-lp" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>🚀 Launchpad</h2>
        <p style="font-size:12px;color:#b0bec5;">Accelerate creators. Integrate launches. Create lasting economic activity.</p>
        <div class="lp-tabs">
            <button class="lp-tab active" data-tab="active">Active</button>
            <button class="lp-tab" data-tab="my-launches">My Launches</button>
            <button class="lp-tab" data-tab="create">Create</button>
        </div>
        <div id="lp-active" class="lp-active"></div>
        <div id="lp-my-launches" class="lp-my-launches" style="display:none;"></div>
        <div id="lp-create" class="lp-create" style="display:none;">
            <input id="lp-title" placeholder="Project Title" />
            <input id="lp-description" placeholder="Description" />
            <input id="lp-category" placeholder="Category" />
            <input id="lp-goal" placeholder="Goal (micro)" type="number" />
            <input id="lp-date" placeholder="Launch Date (YYYY-MM-DD)" />
            <button class="vbt-btn" id="lp-create-btn">Create Launch</button>
        </div>
        <p id="lp-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('launchpad-overlay');
        statusEl = document.getElementById('lp-status');
        document.getElementById('btn-close-lp').addEventListener('click', () => { overlayEl.style.display = 'none'; });
        document.querySelectorAll('.lp-tab').forEach(btn => {
            btn.addEventListener('click', () => {
                document.querySelectorAll('.lp-tab').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                switchLpTab(btn.dataset.tab);
            });
        });
        document.getElementById('lp-create-btn').addEventListener('click', createLaunch);
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>\"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }
    function fmtVBV(micro) { return (parseFloat(micro) / 1000000).toFixed(2); }

    async function api(path, opts) {
        const url = path.startsWith('/api/') ? path : API_BASE + path;
        const resp = await fetch(url, opts);
        if (!resp.ok) { let d = ''; try { d = (await resp.json()).error || ''; } catch(e){} throw new Error(d || ('HTTP ' + resp.status)); }
        return resp.json();
    }

    function switchLpTab(tab) {
        document.querySelectorAll('.lp-tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab)); document.getElementById('lp-active').style.display = tab === 'active' ? 'block' : 'none';
        document.getElementById('lp-my-launches').style.display = tab === 'my-launches' ? 'block' : 'none';
        document.getElementById('lp-create').style.display = tab === 'create' ? 'block' : 'none';
        if (tab === 'active') loadActive();
        if (tab === 'my-launches') loadMyLaunches();
    }

    async function loadActive() {
        try {
            const res = await api('/api/launches?status=active');
            const el = document.getElementById('lp-active');
            if (!el) return;
            if (!(res.launches?.length ?? 0)) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No active launches.</p>'; return; }
            el.innerHTML = res.launches.map(l => `
                <div class="lp-launch">
                    <span class="lp-title">${esc(l.title)}</span>
                    <span class="lp-creator">${esc(l.creator.slice(0, 12))}…</span>
                    <span class="lp-category">${esc(l.category)}</span>
                    <span class="lp-raised">${fmtVBV(l.raised_micro)}/${fmtVBV(l.goal_micro)}</span>
                    <button class="vbt-btn vbt-btn-sm" onclick="window.backLaunch('${l.id}')">Back</button>
                </div>
            `).join('');
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    async function loadMyLaunches() {
        try {
            const res = await api('/api/launch/creator');
            const el = document.getElementById('lp-my-launches');
            if (!el) return;
            if (!(res.launches?.length ?? 0)) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No launches yet. Create one!</p>'; return; }
            el.innerHTML = res.launches.map(l => `
                <div class="lp-launch">
                    <span class="lp-title">${esc(l.title)}</span>
                    <span class="lp-status">${esc(l.status)}</span>
                    <span class="lp-raised">${fmtVBV(l.raised_micro)}/${fmtVBV(l.goal_micro)}</span>
                </div>
            `).join('');
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    async function createLaunch() {
        try {
            const body = {
                title: document.getElementById('lp-title')?.value,
                description: document.getElementById('lp-description')?.value,
                category: document.getElementById('lp-category')?.value,
                goal_micro: parseInt(document.getElementById('lp-goal')?.value) || 0,
                launch_date: document.getElementById('lp-date')?.value
            };
            await api('/api/launch/create', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
            setStatus('Launch created!');
            loadMyLaunches();
        } catch (e) { setStatus('Create failed: ' + e.message); }
    }

    function setStatus(m) { if (statusEl) statusEl.textContent = m || ''; }

    window.backLaunch = async function (projectID) {
        try {
            await api('/api/launch/back', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ project_id: projectID, amount: 1000000 }) });
            setStatus('Launch backed!');
            loadActive();
        } catch (e) { setStatus('Back failed: ' + e.message); }
    };

    window.openLaunchpad = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        loadActive();
    };
})();
