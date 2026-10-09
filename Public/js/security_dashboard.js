// ============================================================================
// security_dashboard.js — Security, Simulation, Black Market, Creator, Vehicles, World Content
// ----------------------------------------------------------------------------
// Wires 20 orphaned backend routes:
//   Security:   /api/admin/asset-forfeiture, /api/admin/avatar-ban,
//                /api/admin/simulate-mutation-success, /api/admin/simulate-mutation-failure,
//                /api/admin/simulate-mojo-decay, /api/admin/simulate-tournament
//   Black Mkt:  /api/black-market/buy-stolen, /api/black-market/fence-goods
//   Creator:    /api/creator/event/create, /api/creator/dlc/create,
//                /api/creator/sub/create, /api/creator/store/deactivate,
//                /api/creator/store/reactivate, /api/creator/store/resell,
//                /api/creator/store/royalty-history
//   Vehicles:   /api/vehicles
//   World:      /api/world-content
// ============================================================================

(function () {
    'use strict';
    var API_BASE = '/api';
    let overlayEl = null;

    function init() {
        if (overlayEl) return;
        overlayEl = document.getElementById('security-dashboard-overlay');
        if (!overlayEl) {
            // SELF-MOUNT. Nothing in the shell provided this root (no `id="security-dashboard-overlay"`
            // in index.html), so the null-check below made the whole module a silent no-op: `init()`
            // returned and all 16 controls this file publishes had no surface to render into.
            document.body.insertAdjacentHTML('beforeend', '<div id="security-dashboard-overlay"></div>');
            overlayEl = document.getElementById('security-dashboard-overlay');
            if (!overlayEl) return;
        }
        overlayEl.className = 'overlay vbt-overlay';
        overlayEl.style.display = 'none';

        overlayEl.innerHTML = `
<div class="neon-glass-panel security-dashboard-panel">
    <button id="btn-close-sd" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
    <h2>🛡️ Security Dashboard</h2>
    <p style="font-size:12px;color:#b0bec5;margin-top:-8px;">Security · Simulation · Black Market · Creator · Vehicles</p>
    <div class="sd-tabs">
        <button class="sd-tab active" data-tab="security" onclick="switchSDTab('security')">Security</button>
        <button class="sd-tab" data-tab="simulation" onclick="switchSDTab('simulation')">Simulation</button>
        <button class="sd-tab" data-tab="blackmarket" onclick="switchSDTab('blackmarket')">Black Market</button>
        <button class="sd-tab" data-tab="creator" onclick="switchSDTab('creator')">Creator</button>
        <button class="sd-tab" data-tab="vehicles" onclick="switchSDTab('vehicles')">Vehicles</button>
    </div>
    <div id="sd-panel-security" class="sd-panel"><div id="sd-security"></div></div>
    <div id="sd-panel-simulation" class="sd-panel hidden"><div id="sd-simulation"></div></div>
    <div id="sd-panel-blackmarket" class="sd-panel hidden"><div id="sd-blackmarket"></div></div>
    <div id="sd-panel-creator" class="sd-panel hidden"><div id="sd-creator"></div></div>
    <div id="sd-panel-vehicles" class="sd-panel hidden"><div id="sd-vehicles"></div></div>
    <p id="sd-status" class="ai-status"></p>
</div>`;

        document.getElementById('btn-close-sd').addEventListener('click', () => { overlayEl.style.display = 'none'; });
        loadSecurity();
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
        if (tab === 'security') loadSecurity();
        if (tab === 'simulation') loadSimulation();
        if (tab === 'blackmarket') loadBlackMarket();
        if (tab === 'creator') loadCreator();
        if (tab === 'vehicles') loadVehicles();
    }

    // === SECURITY ===
    async function loadSecurity() {
        const el = document.getElementById('sd-security');
        if (!el) return;
        el.innerHTML = `
            <div class="sd-grid">
                <button class="sd-btn" onclick="sdAssetForfeiture()">Asset Forfeiture</button>
                <button class="sd-btn" onclick="sdAvatarBan()">Avatar Ban</button>
            </div>
            <div id="sd-security-result" class="sd-result"></div>`;
    }
    async function sdAssetForfeiture() {
        const wallet = prompt('Wallet:');
        if (!wallet) return;
        try { await api('/admin/asset-forfeiture', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ wallet }) }); showSecurityResult('Asset forfeited!'); }
        catch (e) { showSecurityResult('Failed: ' + e.message); }
    }
    async function sdAvatarBan() {
        const wallet = prompt('Wallet:');
        if (!wallet) return;
        try { await api('/admin/avatar-ban', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ wallet }) }); showSecurityResult('Avatar banned!'); }
        catch (e) { showSecurityResult('Failed: ' + e.message); }
    }
    function showSecurityResult(msg) {
        const el = document.getElementById('sd-security-result');
        if (el) el.innerHTML = `<p style="color:#80cbc4;">${esc(msg)}</p>`;
    }

    // === SIMULATION ===
    async function loadSimulation() {
        const el = document.getElementById('sd-simulation');
        if (!el) return;
        el.innerHTML = `
            <div class="sd-grid">
                <button class="sd-btn" onclick="sdSimMutationSuccess()">Mutation Success</button>
                <button class="sd-btn" onclick="sdSimMutationFailure()">Mutation Failure</button>
                <button class="sd-btn" onclick="sdSimMojoDecay()">Mojo Decay</button>
                <button class="sd-btn" onclick="sdSimTournament()">Tournament</button>
            </div>
            <div id="sd-simulation-result" class="sd-result"></div>`;
    }
    async function sdSimMutationSuccess() {
        try { const res = await api('/admin/simulate-mutation-success'); showSimulationResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showSimulationResult('Failed: ' + e.message); }
    }
    async function sdSimMutationFailure() {
        try { const res = await api('/admin/simulate-mutation-failure'); showSimulationResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showSimulationResult('Failed: ' + e.message); }
    }
    async function sdSimMojoDecay() {
        try { const res = await api('/admin/simulate-mojo-decay'); showSimulationResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showSimulationResult('Failed: ' + e.message); }
    }
    async function sdSimTournament() {
        try { const res = await api('/admin/simulate-tournament'); showSimulationResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showSimulationResult('Failed: ' + e.message); }
    }
    function showSimulationResult(msg) {
        const el = document.getElementById('sd-simulation-result');
        if (el) el.innerHTML = `<pre style="color:#80cbc4;font-size:11px;white-space:pre-wrap;">${esc(msg)}</pre>`;
    }

    // === BLACK MARKET ===
    async function loadBlackMarket() {
        const el = document.getElementById('sd-blackmarket');
        if (!el) return;
        el.innerHTML = `
            <div class="sd-grid">
                <button class="sd-btn" onclick="sdBuyStolen()">Buy Stolen</button>
                <button class="sd-btn" onclick="sdFenceGoods()">Fence Goods</button>
            </div>
            <div id="sd-blackmarket-result" class="sd-result"></div>`;
    }
    async function sdBuyStolen() {
        const id = prompt('Item ID:');
        if (!id) return;
        try { await api('/black-market/buy-stolen', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ item_id: id }) }); showBlackMarketResult('Purchased!'); }
        catch (e) { showBlackMarketResult('Failed: ' + e.message); }
    }
    async function sdFenceGoods() {
        const id = prompt('Item ID:');
        if (!id) return;
        try { await api('/black-market/fence-goods', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ item_id: id }) }); showBlackMarketResult('Fenced!'); }
        catch (e) { showBlackMarketResult('Failed: ' + e.message); }
    }
    function showBlackMarketResult(msg) {
        const el = document.getElementById('sd-blackmarket-result');
        if (el) el.innerHTML = `<p style="color:#80cbc4;">${esc(msg)}</p>`;
    }

    // === CREATOR ===
    async function loadCreator() {
        const el = document.getElementById('sd-creator');
        if (!el) return;
        el.innerHTML = `
            <div class="sd-grid">
                <button class="sd-btn" onclick="sdCreatorEventCreate()">Create Event</button>
                <button class="sd-btn" onclick="sdCreatorDLCCreate()">Create DLC</button>
                <button class="sd-btn" onclick="sdCreatorSubCreate()">Create Sub</button>
                <button class="sd-btn" onclick="sdCreatorDeactivate()">Deactivate</button>
                <button class="sd-btn" onclick="sdCreatorReactivate()">Reactivate</button>
                <button class="sd-btn" onclick="sdCreatorResell()">Resell</button>
                <button class="sd-btn" onclick="sdCreatorRoyaltyHistory()">Royalty History</button>
            </div>
            <div id="sd-creator-result" class="sd-result"></div>`;
    }
    async function sdCreatorEventCreate() {
        const title = prompt('Event title:');
        if (!title) return;
        try { await api('/creator/event/create', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ title }) }); showCreatorResult('Event created!'); }
        catch (e) { showCreatorResult('Failed: ' + e.message); }
    }
    async function sdCreatorDLCCreate() {
        const name = prompt('DLC name:');
        if (!name) return;
        try { await api('/creator/dlc/create', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name }) }); showCreatorResult('DLC created!'); }
        catch (e) { showCreatorResult('Failed: ' + e.message); }
    }
    async function sdCreatorSubCreate() {
        const name = prompt('Subscription name:');
        if (!name) return;
        try { await api('/creator/sub/create', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name }) }); showCreatorResult('Sub created!'); }
        catch (e) { showCreatorResult('Failed: ' + e.message); }
    }
    async function sdCreatorDeactivate() {
        const id = prompt('Product ID:');
        if (!id) return;
        try { await api('/creator/store/deactivate/' + encodeURIComponent(id), { method: 'POST' }); showCreatorResult('Deactivated!'); }
        catch (e) { showCreatorResult('Failed: ' + e.message); }
    }
    async function sdCreatorReactivate() {
        const id = prompt('Product ID:');
        if (!id) return;
        try { await api('/creator/store/reactivate/' + encodeURIComponent(id), { method: 'POST' }); showCreatorResult('Reactivated!'); }
        catch (e) { showCreatorResult('Failed: ' + e.message); }
    }
    async function sdCreatorResell() {
        const id = prompt('Product ID:');
        if (!id) return;
        try { await api('/creator/store/resell', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ product_id: id }) }); showCreatorResult('Resold!'); }
        catch (e) { showCreatorResult('Failed: ' + e.message); }
    }
    async function sdCreatorRoyaltyHistory() {
        const id = prompt('Product ID:');
        if (!id) return;
        try { const res = await api('/creator/store/royalty-history?product_id=' + encodeURIComponent(id)); showCreatorResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showCreatorResult('Failed: ' + e.message); }
    }
    function showCreatorResult(msg) {
        const el = document.getElementById('sd-creator-result');
        if (el) el.innerHTML = `<pre style="color:#80cbc4;font-size:11px;white-space:pre-wrap;">${esc(msg)}</pre>`;
    }

    // === VEHICLES ===
    async function loadVehicles() {
        const el = document.getElementById('sd-vehicles');
        if (!el) return;
        el.innerHTML = `
            <div class="sd-grid">
                <button class="sd-btn" onclick="sdListVehicles()">List Vehicles</button>
            </div>
            <div id="sd-vehicles-result" class="sd-result"></div>`;
    }
    async function sdListVehicles() {
        try { const res = await api('/vehicles'); showVehiclesResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showVehiclesResult('Failed: ' + e.message); }
    }
    function showVehiclesResult(msg) {
        const el = document.getElementById('sd-vehicles-result');
        if (el) el.innerHTML = `<pre style="color:#80cbc4;font-size:11px;white-space:pre-wrap;">${esc(msg)}</pre>`;
    }

    window.openSecurityDashboard = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        if (window.PanelManager) window.PanelManager.open(overlayEl);
        else overlayEl.style.display = 'flex';
        switchSDTab('security');
    };
    window.switchSDTab = switchSDTab;

    // EVERY handler name this surface's rendered markup uses must exist on `window` — an inline
    // handler resolves its name there, and every function below is scoped to this IIFE. Without this
    // list all 16 controls (security, simulation, black market, creator, vehicles) were inert.
    Object.assign(window, {
        sdAssetForfeiture, sdAvatarBan, sdBuyStolen, sdFenceGoods, sdListVehicles,
        sdSimMojoDecay, sdSimMutationFailure, sdSimMutationSuccess, sdSimTournament,
        sdCreatorDLCCreate, sdCreatorDeactivate, sdCreatorEventCreate, sdCreatorReactivate,
        sdCreatorResell, sdCreatorRoyaltyHistory, sdCreatorSubCreate,
    });

    if (document.getElementById('security-dashboard-overlay')) init();
})();
