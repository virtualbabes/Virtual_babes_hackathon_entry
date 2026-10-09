// Life Assets panel (§26.4 Pets / §25.6 Vehicles / §25.7 World Content) — menu-mode.
// Mirrors ai_citizens.js conventions. Consumes /api/pets, /api/vehicles, /api/world-content.
(function () {
    'use strict';

    var API_BASE = '/api';
    let state = { pets: [], vehicles: [], world: [], parts: [], spawnFees: [], partMaxLevel: 10, baseCostMicro: 0, _init: false };
    let overlayEl = null, petsEl = null, vehEl = null, worldEl = null, statusEl = null;

    function getWallet() {
        return window.currentWallet || (window.getActiveWallet && window.getActiveWallet()) || '';
    }

    // Every write route here resolves the wallet from the request (header or query), so the
    // panel MUST send one — without it the server can only answer "owner required".
    function walletQS() {
        const w = getWallet();
        return w ? ('?wallet=' + encodeURIComponent(w)) : '';
    }

    // Presentation-boundary micro-$VBV formatting (no ledger value is ever a float).
    function fmtMicro(m) {
        return ((Number(m) || 0) / 1000000).toFixed(2);
    }

    function init() {
        if (state._init) return;
        state._init = true;
        const html = `
<div id="life-assets-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel life-assets-panel">
        <button id="btn-close-life" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>🐾 Life Assets — Pets · Vehicles · World (§26.4/§25.6/§25.7)</h2>

        <div class="la-tabs">
            <button class="la-tab active" data-tab="pets" onclick="window.switchLATab('pets')">Pets</button>
            <button class="la-tab" data-tab="vehicles" onclick="window.switchLATab('vehicles')">Vehicles</button>
            <button class="la-tab" data-tab="world" onclick="window.switchLATab('world')">World Content</button>
            <button class="la-tab" data-tab="events" onclick="window.switchLATab('events')">Entity Events</button>
        </div>

        <div id="la-panel-pets" class="la-panel">
            <h3>Companions (§26.4) — Kennel-owned</h3>
            <p style="font-size:12px;color:#b0bec5;">Companion purchase, breeding and grooming are owned by the
            <b>Companion Kennel</b> (World Dashboard ▸ Assets ▸ Companion Kennel). This panel no longer
            duplicates those controls.</p>
            <button class="vbt-btn vbt-btn-primary" onclick="window.openWorldDashboardToTab && window.openWorldDashboardToTab('pets')">Open the Kennel</button>
            <div id="la-pets-grid" class="la-grid"></div>
        </div>

        <div id="la-panel-vehicles" class="la-panel hidden">
            <h3>Garage (§25.6)</h3>
            <label>Name <input id="la-veh-name" type="text" maxlength="40" placeholder="Ride name" /></label>
            <label>Class <select id="la-veh-kind"></select></label>
            <p id="la-veh-classes" style="font-size:11px;color:#90a4ae;">Loading the class table…</p>
            <button id="la-veh-spawn" class="vbt-btn vbt-btn-primary">Buy Vehicle</button>
            <button class="vbt-btn vbt-btn-secondary" onclick="window.openVehicleArena && window.openVehicleArena()">⚔️ Vehicle arena (3D world)</button>
            <p style="font-size:11px;color:#b0bec5;margin-top:8px;">🔧 <b>Upgrade ladder (§25.6.1)</b> — the counterpart to companion grooming. Companions progress by lineage;
            vehicles progress by <b>parts + investment</b>. Each part raises one stat axis; every fee is routed to the faucet sink.
            The part table and every price below are served by the server, so this panel never guesses the calibration.</p>
            <div id="la-veh-grid" class="la-grid"></div>
        </div>

        <div id="la-panel-world" class="la-panel hidden">
            <h3>World Content (§25.7)</h3>
            <label>Kind <select id="la-wc-kind"><option value="NPC">NPC</option><option value="ANIMAL">Animal</option><option value="SCENERY">Scenery</option><option value="WEATHER">Weather</option></select></label>
            <button id="la-wc-create" class="vbt-btn vbt-btn-primary">Author Content</button>
            <label>Deploy ID <input id="la-wc-id" type="text" placeholder="WC-..." /></label>
            <label>Region <input id="la-wc-region" type="text" placeholder="Base" /></label>
            <button id="la-wc-deploy" class="vbt-btn vbt-btn-secondary">Deploy</button>
            <div id="la-wc-grid" class="la-grid"></div>
        </div>

        <div id="la-panel-events" class="la-panel hidden">
            <h3>Entity Events (§30)</h3>
            <p style="font-size:11px;color:#b0bec5;">Submit event outcomes for your pets/bots. WIN/TRY trains stats + earns $VBV. QUIT punishes.</p>
            <label>Event ID <input id="la-event-id" type="text" placeholder="EVT-..." /></label>
            <div id="la-event-outcomes"></div>
            <button id="la-event-add" class="vbt-btn vbt-btn-secondary">+ Add Outcome</button>
            <button id="la-event-resolve" class="vbt-btn vbt-btn-primary">Resolve Event</button>
            <div id="la-event-result" class="la-result"></div>
        </div>

        <p id="la-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('life-assets-overlay');
        petsEl = document.getElementById('la-pets-grid');
        vehEl = document.getElementById('la-veh-grid');
        worldEl = document.getElementById('la-wc-grid');
        statusEl = document.getElementById('la-status');
        document.getElementById('btn-close-life').addEventListener('click', () => { overlayEl.style.display = 'none'; });
        // NOTE: no pet-purchase listener here — the Kennel owns companion acquisition/progression.
        document.getElementById('la-veh-spawn').addEventListener('click', onSpawnVeh);
        document.getElementById('la-veh-spawn').addEventListener('click', onSpawnVeh);
        document.getElementById('la-wc-create').addEventListener('click', onCreateWC);
        document.getElementById('la-wc-deploy').addEventListener('click', onDeployWC);
        document.getElementById('la-event-add').addEventListener('click', onAddOutcome);
        document.getElementById('la-event-resolve').addEventListener('click', onResolveEvent);
    }

    function switchLATab(tab) {
        document.querySelectorAll('.la-tab').forEach(t => t.classList.toggle('active', t.dataset.tab === tab));
        document.querySelectorAll('.la-panel').forEach(p => p.classList.add('hidden'));
        const panel = document.getElementById('la-panel-' + tab);
        if (panel) panel.classList.remove('hidden');
    }

    function setStatus(m) { if (statusEl) statusEl.textContent = m || ''; }
    function esc(s) { return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }

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

    async function onSpawnVeh() {
        const w = getWallet(); if (!w) { setStatus('Connect wallet.'); return; }
        // No min_level is sent: the class owns the Level gate AND the price (§25.6.1).
        const body = { name: document.getElementById('la-veh-name')?.value.trim(), kind: document.getElementById('la-veh-kind')?.value };
        if (!body?.name) { setStatus('Name required.'); return; }
        const cls = (state.spawnFees || []).find(c => c.kind === body.kind);
        try {
            await api('/vehicles/spawn' + walletQS(), { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
            setStatus('Vehicle purchased for ' + fmtMicro(cls ? cls.fee_micro : 0) + ' $VBV — certified, break-in started.');
            refresh();
        } catch (e) { setStatus('Purchase failed: ' + e.message); }
    }
    async function onCreateWC() {
        const w = getWallet(); if (!w) { setStatus('Connect wallet.'); return; }
        const body = { kind: document.getElementById('la-wc-kind')?.value };
        try { await api('/world-content/create' + walletQS(), { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }); setStatus('Content authored.'); refresh(); }
        catch (e) { setStatus('Create failed: ' + e.message); }
    }
    async function onDeployWC() {
        const w = getWallet(); if (!w) { setStatus('Connect wallet.'); return; }
        const body = { content_id: document.getElementById('la-wc-id')?.value.trim(), region: document.getElementById('la-wc-region')?.value.trim() };
        if (!body.content_id || !body.region) { setStatus('ID + region required.'); return; }
        try { await api('/world-content/deploy' + walletQS(), { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }); setStatus('Deployed to ' + body.region); refresh(); }
        catch (e) { setStatus('Deploy failed: ' + e.message); }
    }

    // ── §25.6.1 Vehicle upgrade ladder ───────────────────────────────────────
    async function onUpgradePart(vehicleId, part) {
        const w = getWallet(); if (!w) { setStatus('Connect wallet.'); return; }
        try {
            const res = await api('/vehicles/upgrade' + walletQS(), {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ vehicle_id: vehicleId, part: part }),
            });
            const gain = (res && res.stat_gain != null) ? res.stat_gain : '?';
            const fee = (res && res.fee_micro != null) ? fmtMicro(res.fee_micro) : '?';
            setStatus(part + ' fitted — +' + gain + ' stat, ' + fee + ' $VBV routed to the faucet sink.');
            refresh();
        } catch (e) { setStatus('Upgrade failed: ' + e.message); }
    }

    async function onDeployVehicle(vehicleId) {
        const w = getWallet(); if (!w) { setStatus('Connect wallet.'); return; }
        const input = document.querySelector('.la-veh-deploy-region[data-veh="' + vehicleId + '"]');
        const region = input ? String(input.value || '').trim() : '';
        if (!region) { setStatus('Region required.'); return; }
        try {
            await api('/vehicles/deploy' + walletQS(), {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ vehicle_id: vehicleId, region: region }),
            });
            setStatus('Vehicle deployed to ' + region + ' — it now feeds the region power overlay.');
            refresh();
        } catch (e) { setStatus('Deploy failed: ' + e.message); }
    }

    async function refresh() {
        const w = walletQS();
        // Paths are relative to API_BASE (which already ends in /api). The previous revision
        // passed '/api/pets' into a helper that prepends '/api', so every read 404'd and the
        // whole panel reported "Load failed" — the Garage could never populate.
        const results = await Promise.all([
            api('/pets' + w).catch(e => ({ __err: 'companions: ' + e.message })),
            api('/vehicles' + w).catch(e => ({ __err: 'vehicles: ' + e.message })),
            api('/world-content' + w).catch(e => ({ __err: 'world content: ' + e.message })),
        ]);
        const [p, v, c] = results;
        const refused = results.filter(r => r && r.__err).map(r => r.__err);
        state.pets = (p && p.data) || [];
        state.vehicles = (v && v.data) || [];
        state.world = (c && c.data) || [];
        // §25.6.1: the part table is server-owned — the client never re-declares it.
        state.parts = (v && v.parts && v.parts.length) ? v.parts : state.parts;
        state.partMaxLevel = (v && v.part_max_level != null) ? Number(v.part_max_level) : state.partMaxLevel;
        state.baseCostMicro = (v && v.base_cost_micro != null) ? Number(v.base_cost_micro) : state.baseCostMicro;
        // §25.6.1 class table: the server decides each class's price + Level gate.
        state.spawnFees = (v && Array.isArray(v.spawn_fees)) ? v.spawn_fees : state.spawnFees;
        renderVehClasses();
        renderPets(); renderVeh(); renderWorld();
        // A refused read is reported as refused — never rendered as "you own nothing".
        if (refused.length) setStatus('Some reads were refused — ' + refused.join('; '));
    }

    // §25.6.1: the class dropdown and its prices are SERVER-served, never hard-coded here.
    function renderVehClasses() {
        const sel = document.getElementById('la-veh-kind');
        const note = document.getElementById('la-veh-classes');
        const fees = state.spawnFees || [];
        if (sel && fees.length) {
            const current = sel.value;
            sel.innerHTML = fees.map(c => `<option value="${esc(c.kind)}">${esc(c.kind)}</option>`).join('');
            if (current && fees.some(c => c.kind === current)) sel.value = current;
        }
        if (note) {
            note.textContent = fees.length
                ? fees.map(c => `${c.kind}: ${fmtMicro(c.fee_micro)} $VBV · min level ${c.min_level}`).join('  |  ')
                : 'Class table not reported by the server.';
        }
    }

    function renderPets() {
        if (!(state.pets?.length ?? 0)) { petsEl.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No pets.</p>'; return; }
        petsEl.innerHTML = state.pets.map(p => {
            const st = p.stats || {};
            const sum = (st.speed||0)+(st.intelligence||0)+(st.willpower||0)+(st.strength||0)+(st.charisma||0)+(st.agility||0);
            const lvl = p.pet_level || Math.min(600, 1 + Math.floor(sum / 50)); // §30 power-overlay level (clamped 600)
            const badge = p.certified ? ' ✓cert' : (p.black_market_adopted ? ' ⚠black-market' : '');
            const region = p.region || '';
            return `<div class="la-card">
                <h4>${esc(p?.name)}${badge}</h4>
                <p style="font-size:11px;color:#b0bec5;">${esc((p.pet_id||'').slice(0,10))}${p.sire_id ? ' · bred' : ' · base'}</p>
                <p style="font-size:11px;color:#4dd0e1;">PWR ${lvl}${p.mature ? ' · mature' : ' · immature'}${region ? ' · @' + esc(region) : ''}</p>
                <div class="ai-card-actions">
                    ${region ? `<button class="vbt-btn vbt-btn-secondary" onclick="window._laView3D('${esc(region)}')">🌐 View in 3D</button>` : ''}
                </div>
            </div>`;
        }).join('');
    }
    function renderVeh() {
        if (!(state.vehicles?.length ?? 0)) { vehEl.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No vehicles.</p>'; return; }
        const parts = state.parts || [];
        const maxLevel = Number(state.partMaxLevel) || 10;
        const baseCost = Number(state.baseCostMicro) || 0;
        vehEl.innerHTML = state.vehicles.map(v => {
            const st = v.stats || {};
            const sum = (st.speed||0)+(st.intelligence||0)+(st.willpower||0)+(st.strength||0)+(st.charisma||0)+(st.agility||0);
            const ups = v.upgrades || {};
            const sumParts = Object.keys(ups).reduce((a, k) => a + (Number(ups[k]) || 0), 0);
            const build = v.vehicle_level || (1 + sumParts);
            const power = Math.min(600, build + Math.floor(sum / 50)); // §30 power overlay (clamped 600)
            const badge = v.certified ? ' ✓cert' : (v.black_market_adopted ? ' ⚠black-market' : '');
            const region = v.region || '';
            const rows = parts.map(p => {
                const lvl = Number(ups[p.part]) || 0;
                const cost = baseCost * (lvl + 1);
                const maxed = lvl >= maxLevel;
                return `<div class="la-outcome-row" style="display:flex;gap:6px;align-items:center;margin:3px 0;">
                    <span style="width:92px;font-size:11px;color:#4dd0e1;">${esc(p.part)}</span>
                    <span style="width:56px;font-size:11px;color:#b0bec5;">L${lvl}/${maxLevel}</span>
                    <span style="width:74px;font-size:11px;color:#b0bec5;">${esc(p.axis)}</span>
                    <span style="flex:1;font-size:11px;color:#90a4ae;">${maxed ? 'max' : fmtMicro(cost) + ' $VBV'}</span>
                    <button class="vbt-btn vbt-btn-secondary"${maxed ? ' disabled' : ''}
                        onclick="window.vehUpgradePart('${esc(v.vehicle_id)}','${esc(p.part)}')">Fit</button>
                </div>`;
            }).join('');
            return `<div class="la-card">
                <h4>${esc(v?.name)}${badge}</h4>
                <p style="font-size:11px;color:#b0bec5;">${esc(v.kind)} · build L${build}${region ? ' · @' + esc(region) : ' · stored'}${v.mature ? ' · ready' : ' · breaking in'}</p>
                <p style="font-size:11px;color:#4dd0e1;">PWR ${power} · ${sum} stat pts · ${sumParts}/${parts.length * maxLevel} parts</p>
                <div style="margin-top:4px;">${rows || '<p style="font-size:11px;color:#90a4ae;">No part table reported by the server.</p>'}</div>
                <div class="ai-card-actions">
                    <input class="la-veh-deploy-region" data-veh="${esc(v.vehicle_id)}" type="text" placeholder="Region" style="width:110px;" />
                    <button class="vbt-btn vbt-btn-secondary" onclick="window.vehDeploy('${esc(v.vehicle_id)}')">Deploy</button>
                    ${region ? `<button class="vbt-btn vbt-btn-secondary" onclick="window._laView3D('${esc(region)}')">🌐 View in 3D</button>` : ''}
                </div>
            </div>`;
        }).join('');
    }
    function renderWorld() {
        if (!(state.world?.length ?? 0)) { worldEl.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No world content.</p>'; return; }
        worldEl.innerHTML = state.world.map(c => {
            const region = c.region || '';
            return `<div class="la-card">
                <h4>${esc(c.kind)}</h4>
                <p style="font-size:11px;color:#b0bec5;">${esc((c.content_id||'').slice(0,10))}${c.deployed ? ' · @' + esc(region) : ' · stored'}</p>
                <div class="ai-card-actions">
                    ${region ? `<button class="vbt-btn vbt-btn-secondary" onclick="window._laView3D('${esc(region)}')">🌐 View in 3D</button>` : ''}
                </div>
            </div>`;
        }).join('');
    }

    // ── §30 Entity Event Payout Loop ────────────────────────────────────────
    let eventOutcomes = [];

    function renderEventOutcomes() {
        const container = document.getElementById('la-event-outcomes');
        if (!container) return;
        container.innerHTML = eventOutcomes.map((o, i) => `
            <div class="la-outcome-row" style="display:flex;gap:6px;margin:4px 0;align-items:center;">
                <select class="la-eo-isbot" data-i="${i}" style="width:80px;">
                    <option value="false" ${!o.isBot ? 'selected' : ''}>Pet</option>
                    <option value="true" ${o.isBot ? 'selected' : ''}>Bot</option>
                </select>
                <input class="la-eo-id" data-i="${i}" type="text" placeholder="Entity ID" value="${esc(o.entityId)}" style="flex:1;" />
                <select class="la-eo-outcome" data-i="${i}" style="width:90px;">
                    <option value="WIN" ${o.outcome === 'WIN' ? 'selected' : ''}>WIN</option>
                    <option value="TRY" ${o.outcome === 'TRY' ? 'selected' : ''}>TRY</option>
                    <option value="QUIT" ${o.outcome === 'QUIT' ? 'selected' : ''}>QUIT</option>
                </select>
                <button class="vbt-btn vbt-btn-secondary la-eo-remove" data-i="${i}">✕</button>
            </div>
        `).join('');
        // bind events
        container.querySelectorAll('.la-eo-isbot').forEach(el => el.addEventListener('change', e => { eventOutcomes[+e.target.dataset.i].isBot = e.target.value === 'true'; }));
        container.querySelectorAll('.la-eo-id').forEach(el => el.addEventListener('input', e => { eventOutcomes[+e.target.dataset.i].entityId = e.target.value.trim(); }));
        container.querySelectorAll('.la-eo-outcome').forEach(el => el.addEventListener('change', e => { eventOutcomes[+e.target.dataset.i].outcome = e.target.value; }));
        container.querySelectorAll('.la-eo-remove').forEach(el => el.addEventListener('click', e => { eventOutcomes.splice(+e.target.dataset.i, 1); renderEventOutcomes(); }));
    }

    function onAddOutcome() { eventOutcomes.push({ entityId: '', isBot: false, outcome: 'WIN' }); renderEventOutcomes(); }

    async function onResolveEvent() {
        const eventId = document.getElementById('la-event-id')?.value.trim();
        if (!eventId) { setStatus('Event ID required.'); return; }
        if (!(eventOutcomes?.length ?? 0)) { setStatus('Add at least one outcome.'); return; }
        const outcomes = eventOutcomes.map(o => ({ entity_id: o.entityId, is_bot: o.isBot, outcome: o.outcome }));
        try {
            const res = await api('/entity-event/resolve', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ event_id: eventId, outcomes }) });
            const resultEl = document.getElementById('la-event-result');
            if (resultEl && res.credits) {
                const credits = Object.entries(res.credits).map(([w, m]) => `${w.slice(0, 8)}…: +${(m / 1000000).toFixed(4)} $VBV`).join('<br>');
                resultEl.innerHTML = `<p style="color:#4dd0e1;font-size:12px;">✅ Resolved!<br>${credits || 'No credits (immature/quit)'}</p>`;
            }
            setStatus('Event resolved.');
            eventOutcomes = [];
            renderEventOutcomes();
        } catch (e) { setStatus('Resolve failed: ' + e.message); }
    }

    window.openLifeAssets = function (tab) {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        if (tab) switchLATab(tab);
        refresh();
    };
    window.switchLATab = switchLATab;
    // §25.6.1 vehicle ladder bindings (used by the Garage card markup above).
    window.vehUpgradePart = onUpgradePart;
    window.vehDeploy = onDeployVehicle;
    // 3D-world integration (§25 / §30): warp the explorer to the asset's region capital.
    window._laView3D = function (region) {
        if (typeof window.enter3DWorld === 'function') window.enter3DWorld();
        setTimeout(function () {
            if (window.World3DEngine && typeof window.World3DEngine.warpToRegion === 'function') {
                window.World3DEngine.warpToRegion(region);
            }
        }, 350);
    };
})();
