// World Events + Treasure Hunt frontend module (§26) — menu-mode panel for
// spawning/claiming treasure caches, creating/joining user events. Mirrors ai_citizens.js.
(function () {
    'use strict';

    var API_BASE = '/api';

    let state = {
        caches: [],
        events: [],
        regions: []
    };
    let overlayEl = null, cachesEl = null, eventsEl = null, statusEl = null;

    function getWallet() {
        return window.currentWallet || (window.getActiveWallet && window.getActiveWallet()) || '';
    }

    function init() {
        if (state._init) return;
        state._init = true;
        buildOverlay();
        wire();
    }

    function buildOverlay() {
        const html = `
<div id="world-events-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel world-events-panel">
        <button id="btn-close-world-events" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>🗺️ Treasure Hunts & Events (§26)</h2>

        <div class="we-tabs">
            <button class="we-tab active" data-tab="caches" onclick="window.switchWETab('caches')">Treasure Caches</button>
            <button class="we-tab" data-tab="events" onclick="window.switchWETab('events')">User Events</button>
            <button class="we-tab" data-tab="spawn" onclick="window.switchWETab('spawn')">Spawn Cache</button>
            <button class="we-tab" data-tab="create" onclick="window.switchWETab('create')">Create Event</button>
        </div>

        <div id="we-panel-caches" class="we-panel">
            <h3>Active Caches</h3>
            <div id="we-caches-grid" class="we-grid"></div>
        </div>

        <div id="we-panel-events" class="we-panel hidden">
            <h3>Active Events</h3>
            <div id="we-events-grid" class="we-grid"></div>
        </div>

        <div id="we-panel-spawn" class="we-panel hidden">
            <h3>Spawn Treasure Cache</h3>
            <label>Region <input id="we-cache-region" type="text" maxlength="8" placeholder="Base" /></label>
            <label>Reward (micro-VBV) <input id="we-cache-reward" type="number" min="0" value="0" /></label>
            <label>Reward kind
                <select id="we-cache-kind"><option value="VBV">VBV</option><option value="ITEM_NFT">Item NFT</option><option value="CARD_NFT">Card NFT</option><option value="SCAR_NFT">Scar NFT</option></select>
            </label>
            <label>Reward ref (asset ID) <input id="we-cache-ref" type="text" placeholder="optional" /></label>
            <label>Entry bond (micro-VBV) <input id="we-cache-bond" type="number" min="0" value="0" /></label>
            <label>Hidden (needs tunneling/fog) <input id="we-cache-hidden" type="checkbox" /></label>
            <label>TTL seconds <input id="we-cache-ttl" type="number" min="60" value="86400" /></label>
            <button id="we-cache-btn" class="vbt-btn vbt-btn-primary">Spawn Cache</button>
        </div>

        <div id="we-panel-create" class="we-panel hidden">
            <h3>Create User Event</h3>
            <label>Type
                <select id="we-ev-type">
                    <option value="TREASURE_HUNT">Treasure Hunt</option>
                    <option value="SEARCH_RESCUE">Search & Rescue</option>
                    <option value="GANG_BASHING">Gang Bashing</option>
                    <option value="WILD_BOT_HUNT">Wild Bot Hunt</option>
                    <option value="PET_BREEDING_SHOW">Pet Breeding Show</option>
                </select>
            </label>
            <label>Title (custom asset name) <input id="we-ev-title" type="text" maxlength="40" placeholder="My Event" /></label>
            <label>Region <input id="we-ev-region" type="text" maxlength="8" placeholder="Base" /></label>
            <label>Reward pool (micro-VBV) <input id="we-ev-reward" type="number" min="0" value="0" /></label>
            <label>Entry cost (micro-VBV) <input id="we-ev-entry" type="number" min="0" value="0" /></label>
            <label>Bonded NFT ID <input id="we-ev-nft" type="text" placeholder="optional" /></label>
            <label>Royalty bps (≤1000) <input id="we-ev-roy" type="number" min="0" max="1000" value="250" /></label>
            <label>TTL seconds <input id="we-ev-ttl" type="number" min="60" value="86400" /></label>
            <button id="we-ev-btn" class="vbt-btn vbt-btn-primary">Create Event</button>
        </div>

        <p id="we-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('world-events-overlay');
        cachesEl = document.getElementById('we-caches-grid');
        eventsEl = document.getElementById('we-events-grid');
        statusEl = document.getElementById('we-status');
    }

    function wire() {
        document.getElementById('btn-close-world-events').addEventListener('click', () => { overlayEl.style.display = 'none'; });
        document.getElementById('we-cache-btn').addEventListener('click', onSpawnCache);
        document.getElementById('we-ev-btn').addEventListener('click', onCreateEvent);
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

    async function onSpawnCache() {
        const wallet = getWallet();
        if (!wallet) { setStatus('Connect wallet first.'); return; }
        const body = {
            region: document.getElementById('we-cache-region')?.value.trim(),
            reward_micro: parseInt(document.getElementById('we-cache-reward')?.value, 10) || 0,
            reward_kind: document.getElementById('we-cache-kind')?.value,
            reward_ref: document.getElementById('we-cache-ref')?.value.trim(),
            bond_micro: parseInt(document.getElementById('we-cache-bond')?.value, 10) || 0,
            hidden: document.getElementById('we-cache-hidden').checked,
            ttl_seconds: parseInt(document.getElementById('we-cache-ttl')?.value, 10) || 86400
        };
        try {
            await api('/treasure/spawn', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
            setStatus('Cache spawned.');
            refresh();
        } catch (e) { setStatus('Spawn failed: ' + e.message); }
    }

    async function onCreateEvent() {
        const wallet = getWallet();
        if (!wallet) { setStatus('Connect wallet first.'); return; }
        const body = {
            type: document.getElementById('we-ev-type')?.value,
            title: document.getElementById('we-ev-title')?.value.trim(),
            description: '',
            region: document.getElementById('we-ev-region')?.value.trim(),
            reward_micro: parseInt(document.getElementById('we-ev-reward')?.value, 10) || 0,
            entry_micro: parseInt(document.getElementById('we-ev-entry')?.value, 10) || 0,
            bonded_nft: document.getElementById('we-ev-nft')?.value.trim(),
            royalty_bps: parseInt(document.getElementById('we-ev-roy')?.value, 10) || 0,
            ttl_seconds: parseInt(document.getElementById('we-ev-ttl')?.value, 10) || 86400
        };
        if (!body.title) { setStatus('Title required.'); return; }
        try {
            await api('/events/create', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
            setStatus('Event created (bonded NFT if provided).');
            refresh();
        } catch (e) { setStatus('Create failed: ' + e.message); }
    }

    async function claimCache(cacheID) {
        try {
            await api('/treasure/claim', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ cache_id: cacheID }) });
            setStatus('Cache claimed!');
            refresh();
        } catch (e) { setStatus('Claim failed: ' + e.message); }
    }

    async function enterEvent(eventID) {
        try {
            await api('/events/enter', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ event_id: eventID }) });
            setStatus('Entered event.');
        } catch (e) { setStatus('Enter failed: ' + e.message); }
    }

    async function refresh() {
        try {
            const r = await api('/regions');
            const views = (r.data) || [];
            const caches = [], events = [];
            for (const v of views) {
                (v.caches || []).forEach(c => caches.push(c));
                (v.events || []).forEach(e => events.push(e));
            }
            state.caches = caches; state.events = events;
            renderCaches(); renderEvents();
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }

    function renderCaches() {
        if (!cachesEl) return;
        if (!(state.caches?.length ?? 0)) { cachesEl.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No active caches.</p>'; return; }
        cachesEl.innerHTML = state.caches.map(c => `<div class="we-card">
            <h4>Cache ${esc(c.cache_id).slice(0, 8)}</h4>
            <p style="font-size:12px;color:#b0bec5;">Region: ${esc(c.region)} · Reward: ${c.reward_micro || c.reward_kind}</p>
            <div class="ai-card-actions">
                <button class="vbt-btn vbt-btn-primary" onclick="window._weClaim('${esc(c.cache_id)}')">Claim</button>
                ${c.region ? `<button class="vbt-btn vbt-btn-secondary" onclick="window._weView3D('${esc(c.region)}')">🌐 View in 3D</button>` : ''}
            </div>
        </div>`).join('');
    }

    function switchWETab(tab) {
        document.querySelectorAll('.we-tab').forEach(t => t.classList.toggle('active', t.dataset.tab === tab));
        document.querySelectorAll('.we-panel').forEach(p => p.classList.add('hidden'));
        const panel = document.getElementById('we-panel-' + tab);
        if (panel) panel.classList.remove('hidden');
    }

    function renderCaches() {
        const grid = document.getElementById('we-caches-grid');
        if (!grid) return;
        if (!(state.caches?.length ?? 0)) { grid.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No active caches.</p>'; return; }
        grid.innerHTML = state.caches.map(c => `<div class="we-card">
            <h4>Cache #${esc(c.cache_id)}</h4>
            <p style="font-size:12px;color:#b0bec5;">Region: ${esc(c.region)} · Reward: ${c.reward_micro || 0} μVBV</p>
            <div class="ai-card-actions">
                <button class="vbt-btn vbt-btn-primary" onclick="window._weClaim('${esc(c.cache_id)}')">Claim</button>
                ${c.region ? `<button class="vbt-btn vbt-btn-secondary" onclick="window._weView3D('${esc(c.region)}')">🌐 View in 3D</button>` : ''}
            </div>
        </div>`).join('');
    }

    function renderEvents() {
        const grid = document.getElementById('we-events-grid');
        if (!grid) return;
        if (!(state.events?.length ?? 0)) { grid.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No active events.</p>'; return; }
        grid.innerHTML = state.events.map(e => `<div class="we-card">
            <h4>${esc(e.title)}</h4>
            <p style="font-size:12px;color:#b0bec5;">${esc(e.type)} · Region: ${esc(e.region)} · Entry: ${e.entry_micro || 0}</p>
            <div class="ai-card-actions">
                <button class="vbt-btn vbt-btn-primary" onclick="window._weEnter('${esc(e.event_id)}')">Enter</button>
                ${e.region ? `<button class="vbt-btn vbt-btn-secondary" onclick="window._weView3D('${esc(e.region)}')">🌐 View in 3D</button>` : ''}
            </div>
        </div>`).join('');
    }

    window.openWorldEvents = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        refresh();
    };
    window.switchWETab = switchWETab;
    // 3D-world integration (§25): warp the explorer to the cache/event's region capital.
    window._weView3D = function (region) {
        if (typeof window.enter3DWorld === 'function') window.enter3DWorld();
        setTimeout(function () {
            if (window.World3DEngine && typeof window.World3DEngine.warpToRegion === 'function') {
                window.World3DEngine.warpToRegion(region);
            }
        }, 350);
    };
    window._weClaim = claimCache;
    window._weEnter = enterEvent;
})();
