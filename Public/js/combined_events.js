// ============================================================================
// combined_events.js — Combined Events (all entity types)
// ----------------------------------------------------------------------------
// Events for the combination of ALL entity types:
// players + pets + AI citizens + LLM bots + children bots + vehicles
//
// EXPANDABLE: New entity types auto-integrate via server enumeration.
// WASM SPLIT: All logic runs in server; this is the UI layer only.
// VOI FAUCET: All rewards flow to l.faucetBalanceMicro via backend.
// ============================================================================
(function () {
    'use strict';

    var API_BASE = '/api';

    let overlayEl = null;
    let eventsEl = null;
    let statusEl = null;

    function init() {
        if (overlayEl) return;
        const html = `
<div id="combined-events-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel combined-events-panel">
        <button id="btn-close-ce" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>🌟 Combined Events</h2>

        <div class="ce-tabs">
            <button class="ce-tab active" data-tab="active" onclick="window.switchCETab('active')">Active Events</button>
            <button class="ce-tab" data-tab="host" onclick="window.switchCETab('host')">Host Event</button>
            <button class="ce-tab" data-tab="types" onclick="window.switchCETab('types')">Entity Types</button>
        </div>

        <div id="ce-panel-active" class="ce-panel">
            <h3>Active Combined Events</h3>
            <div id="ce-events"></div>
        </div>

        <div id="ce-panel-host" class="ce-panel hidden">
            <h3>Host Combined Event</h3>
            <label>Event Name <input id="ce-name" type="text" placeholder="Grand Tournament..." /></label>
            <label>Event Tier <input id="ce-tier" type="number" min="1" value="1" /></label>
            <label>Region <input id="ce-region" type="text" placeholder="Base" /></label>
            <label>Cost (μVBV) <input id="ce-cost" type="number" min="0" value="0" /></label>
            <button id="ce-host-btn" class="vbt-btn vbt-btn-primary">Host Event</button>
        </div>

        <div id="ce-panel-types" class="ce-panel hidden">
            <h3>Entity Types</h3>
            <div class="ce-entity-types">
                <span class="ce-type-badge">👤 Players</span>
                <span class="ce-type-badge">🐾 Pets</span>
                <span class="ce-type-badge">🤖 AI Citizens</span>
                <span class="ce-type-badge">🧠 LLM Bots</span>
                <span class="ce-type-badge">👶 Children Bots</span>
                <span class="ce-type-badge">🚗 Vehicles</span>
            </div>
        </div>

        <p id="ce-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('combined-events-overlay');
        eventsEl = document.getElementById('ce-events');
        statusEl = document.getElementById('ce-status');

        document.getElementById('btn-close-ce').addEventListener('click', () => { overlayEl.style.display = 'none'; });
        document.getElementById('ce-host-btn').addEventListener('click', onHost);
        document.getElementById('ce-submit-btn').addEventListener('click', onSubmit);
    }

    function switchCETab(tab) {
        document.querySelectorAll('.ce-tab').forEach(t => t.classList.toggle('active', t.dataset.tab === tab));
        document.querySelectorAll('.ce-panel').forEach(p => p.classList.add('hidden'));
        const panel = document.getElementById('ce-panel-' + tab);
        if (panel) panel.classList.remove('hidden');
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>\"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }
    function fmtVBV(micro) { return (parseFloat(micro) / 1000000).toFixed(4); }

    async function api(path, opts) {
        const url = path.startsWith('/api/') ? path : API_BASE + path;
        const resp = await fetch(url, opts);
        if (!resp.ok) { let d = ''; try { d = (await resp.json()).error || ''; } catch(e){} throw new Error(d || ('HTTP ' + resp.status)); }
        return resp.json();
    }

    async function onHost() {
        const name = document.getElementById('ce-name')?.value.trim();
        const tier = parseInt(document.getElementById('ce-tier')?.value, 10) || 1;
        const region = document.getElementById('ce-region')?.value.trim() || 'Base';
        const cost = parseInt(document.getElementById('ce-cost')?.value, 10) || 0;

        if (!name) { setStatus('Event name required.'); return; }
        try {
            const res = await api('/api/entity-event/host', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name, event_tier: tier, region, cost_micro: cost }),
            });
            setStatus(`✅ Event hosted! ID: ${res.event.event_id}`);
            loadEvents();
        } catch (e) { setStatus('Host failed: ' + e.message); }
    }

    async function onSubmit() {
        const eventId = document.getElementById('ce-event-id')?.value.trim();
        const entityId = document.getElementById('ce-entity-id')?.value.trim();
        const entityType = document.getElementById('ce-entity-type')?.value;
        const outcome = document.getElementById('ce-outcome')?.value;

        if (!eventId || !entityId) { setStatus('Event ID + Entity ID required.'); return; }
        try {
            const res = await api('/api/entity-event/resolve', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    event_id: eventId,
                    outcomes: [{ entity_id: entityId, is_bot: entityType !== 'player', outcome: outcome }],
                }),
            });
            setStatus(`✅ Outcome submitted! Credits: ${JSON.stringify(res.credits || {})}`);
            loadEvents();
        } catch (e) { setStatus('Submit failed: ' + e.message); }
    }

    async function loadEvents() {
        try {
            const res = await api('/api/entity-events/regions');
            renderEvents(res.regions || []);
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    function renderEvents(events) {
        if (!eventsEl) return;
        if (!(events?.length ?? 0)) {
            eventsEl.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No active events.</p>';
            return;
        }
        eventsEl.innerHTML = events.map(e => `
            <div class="ce-event">
                <span class="ce-event-name">${esc(e?.name || e.event_id)}</span>
                <span class="ce-event-tier">Tier ${e.event_tier || 1}</span>
                <span class="ce-event-region">📍 ${esc(e.region)}</span>
                <span class="ce-event-participants">${(e.participants || [])?.length ?? 0} participants</span>
            </div>
        `).join('');
    }

    function setStatus(m) { if (statusEl) statusEl.textContent = m || ''; }

    // The tab bar this module renders names `switchCETab`, and an inline handler resolves that name
    // on `window` — a function inside this IIFE is not a global, so the tabs were inert.
    window.switchCETab = switchCETab;

    window.openCombinedEvents = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        loadEvents();
    };
})();
