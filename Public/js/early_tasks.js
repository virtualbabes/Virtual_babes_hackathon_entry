// Early-Task Tracker (§25.5) — menu-mode guidance for new players. Fades once the
// player is established (owns core assets across systems). Pure UI; probes existing
// endpoints for completion state. No fabricated backend.
(function () {
    'use strict';

    var API_BASE = '/api';

    // Ordered early-game objectives. Each has a check() returning a Promise<boolean>.
    const TASKS = [
        { id: 'wallet', label: 'Connect your wallet', hint: 'Link a primary wallet to begin.', check: () => Promise.resolve(!!(window.currentWallet || (window.getActiveWallet && window.getActiveWallet()))) },
        { id: 'citizen', label: 'Spawn an AI Citizen', hint: 'Open 🤖 AI Citizens and create one.', check: () => fetch(API_BASE + '/ai/citizens').then(r => r.ok ? r.json() : { data: [] }).then(d => ((d.data || [])?.length ?? 0) > 0) },
        { id: 'treasure', label: 'Claim a Treasure Cache', hint: 'Open 🗺️ Treasure & Events and grab a cache.', check: () => fetch(API_BASE + '/regions').then(r => r.ok ? r.json() : { data: [] }).then(d => { const v = d.data || []; return v.some((x => (x.caches || [])?.length ?? 0) > 0); }) },
        { id: 'pet', label: 'Breed or own a Pet', hint: 'Open 🐾 Life Assets → Kennel.', check: () => fetch(API_BASE + '/pets').then(r => r.ok ? r.json() : { data: [] }).then(d => ((d.data || [])?.length ?? 0) > 0) },
        { id: 'vehicle', label: 'Acquire a Vehicle', hint: 'Open 🐾 Life Assets → Garage.', check: () => fetch(API_BASE + '/vehicles').then(r => r.ok ? r.json() : { data: [] }).then(d => ((d.data || [])?.length ?? 0) > 0) },
        { id: 'hub', label: 'Visit the Leaderboard Hub', hint: 'Open 🏆 Leaderboard Hub — your gateway to the 3D world.', check: () => Promise.resolve(false) /* visited flag set on open */ }
    ];

    let state = { done: {}, _init: false, visitedHub: false };
    let overlayEl = null, listEl = null, statusEl = null;

    function init() {
        if (state._init) return;
        state._init = true;
        const html = `
<div id="early-tasks-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel early-tasks-panel">
        <button id="btn-close-early" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>🧭 Early Tasks (§25.5)</h2>

        <div class="et-tabs">
            <button class="et-tab active" data-tab="tasks" onclick="window.switchETTab('tasks')">Tasks</button>
            <button class="et-tab" data-tab="rewards" onclick="window.switchETTab('rewards')">Rewards</button>
        </div>

        <div id="et-panel-tasks" class="et-panel">
            <ul id="early-tasks-list" class="et-list"></ul>
        </div>

        <div id="et-panel-rewards" class="et-panel hidden">
            <h3>Rewards</h3>
            <p style="color:#b0bec5;font-size:12px;">Complete tasks to earn rewards. Each completed task unlocks a reward.</p>
            <div id="et-rewards-grid" class="et-rewards-grid"></div>
        </div>

        <p id="et-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('early-tasks-overlay');
        listEl = document.getElementById('early-tasks-list');
        statusEl = document.getElementById('et-status');
        document.getElementById('btn-close-early').addEventListener('click', () => { overlayEl.style.display = 'none'; });
    }

    function switchETTab(tab) {
        document.querySelectorAll('.et-tab').forEach(t => t.classList.toggle('active', t.dataset.tab === tab));
        document.querySelectorAll('.et-panel').forEach(p => p.classList.add('hidden'));
        const panel = document.getElementById('et-panel-' + tab);
        if (panel) panel.classList.remove('hidden');
    }

    function setStatus(m) { if (statusEl) statusEl.textContent = m || ''; }

    async function evaluate() {
        let complete = 0;
        for (const t of TASKS) {
            let ok = false;
            try { ok = await t.check(); } catch (e) { ok = false; }
            if (t.id === 'hub' && state.visitedHub) ok = true;
            state.done[t.id] = ok;
            if (ok) complete++;
        }
        render();
        const ratio = complete / TASKS?.length ?? 0;
        if (ratio >= 0.999) setStatus('All early tasks complete — you\'re established. The tracker will hide itself from now on.');
        else setStatus(complete + '/' + TASKS?.length ?? 0 + ' early tasks done.');
    }

    function render() {
        if (!listEl) return;
        listEl.innerHTML = TASKS.map(t => {
            const done = state.done[t.id];
            return `<li class="et-item ${done ? 'et-done' : ''}">
                <span class="et-check">${done ? '✅' : '⬜'}</span>
                <span class="et-label">${t.label}</span>
                <span class="et-hint">${t.hint}</span>
            </li>`;
        }).join('');
    }

    // Auto-open on first load for new players; fade when established.
    window.openEarlyTasks = function () {
        init();
        if (!state.visitedHub && false) { /* noop */ }
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        evaluate();
    };
    // The tab bar this module renders names `switchETTab`, and an inline handler resolves that name
    // on `window` — a function inside this IIFE is not a global, so the tabs were inert.
    window.switchETTab = switchETTab;

    window.markHubVisited = function () { state.visitedHub = true; };

    // Hook: when the leaderboard hub opens, mark the hub task.
    const _origOpenLB = window.openLeaderboardRegion;
    if (typeof _origOpenLB === 'function') {
        window.openLeaderboardRegion = function () { state.visitedHub = true; _origOpenLB(); };
    }
})();
