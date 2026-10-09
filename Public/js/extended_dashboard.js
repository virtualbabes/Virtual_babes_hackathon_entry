// ============================================================================
// extended_dashboard.js — Extended routes: Assets, Orphans, OS, Launches, Rewards, Invest
// ----------------------------------------------------------------------------
// Wires 20 orphaned backend routes:
//   Assets:     /api/assets, /api/assets/burn, /api/assets/mint, /api/assets/modify, /api/assets/transfer
//   Orphans:    /api/orphan/adopt, /api/orphan/reclaim, /api/orphan/status
//   OS:         /api/os/module/register, /api/os/summary
//   Launches:   /api/launch/activate, /api/launch/get, /api/launch/integrate
//   Rewards:    /api/reward/add, /api/reward/remove, /api/reward/update-asset, /api/reward/update-base
//   Invest:     /api/invest/dividends/history, /api/invest/entity, /api/invest/portfolio
//   Items:      /api/items/registry
// ============================================================================

(function () {
    'use strict';
    var API_BASE = '/api';
    let overlayEl = null;

    function init() {
        if (overlayEl) return;
        overlayEl = document.getElementById('extended-dashboard-overlay');
        if (!overlayEl) {
            // SELF-MOUNT. Nothing in the shell provided this root (no `id="extended-dashboard-overlay"`
            // in index.html), so the null-check below made the whole module a silent no-op: `init()`
            // returned and all 20 controls this file publishes had no surface to render into.
            document.body.insertAdjacentHTML('beforeend', '<div id="extended-dashboard-overlay"></div>');
            overlayEl = document.getElementById('extended-dashboard-overlay');
            if (!overlayEl) return;
        }
        overlayEl.className = 'overlay vbt-overlay';
        overlayEl.style.display = 'none';

        overlayEl.innerHTML = `
<div class="neon-glass-panel extended-dashboard-panel">
    <button id="btn-close-ed" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
    <h2>📦 Extended Dashboard</h2>
    <p style="font-size:12px;color:#b0bec5;margin-top:-8px;">Assets · Orphans · OS · Launches · Rewards · Invest</p>
    <div class="ed-tabs">
        <button class="ed-tab active" data-tab="assets" onclick="switchEDTab('assets')">Assets</button>
        <button class="ed-tab" data-tab="orphans" onclick="switchEDTab('orphans')">Orphans</button>
        <button class="ed-tab" data-tab="os" onclick="switchEDTab('os')">OS</button>
        <button class="ed-tab" data-tab="launches" onclick="switchEDTab('launches')">Launches</button>
        <button class="ed-tab" data-tab="rewards" onclick="switchEDTab('rewards')">Rewards</button>
        <button class="ed-tab" data-tab="invest" onclick="switchEDTab('invest')">Invest</button>
    </div>
    <div id="ed-panel-assets" class="ed-panel"><div id="ed-assets"></div></div>
    <div id="ed-panel-orphans" class="ed-panel hidden"><div id="ed-orphans"></div></div>
    <div id="ed-panel-os" class="ed-panel hidden"><div id="ed-os"></div></div>
    <div id="ed-panel-launches" class="ed-panel hidden"><div id="ed-launches"></div></div>
    <div id="ed-panel-rewards" class="ed-panel hidden"><div id="ed-rewards"></div></div>
    <div id="ed-panel-invest" class="ed-panel hidden"><div id="ed-invest"></div></div>
    <p id="ed-status" class="ai-status"></p>
</div>`;

        document.getElementById('btn-close-ed').addEventListener('click', () => { overlayEl.style.display = 'none'; });
        loadAssets();
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

    function switchEDTab(tab) {
        document.querySelectorAll('.ed-tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab));
        document.querySelectorAll('.ed-panel').forEach(p => p.classList.add('hidden'));
        const panel = document.getElementById('ed-panel-' + tab);
        if (panel) panel.classList.remove('hidden');
        if (tab === 'assets') loadAssets();
        if (tab === 'orphans') loadOrphans();
        if (tab === 'os') loadOS();
        if (tab === 'launches') loadLaunches();
        if (tab === 'rewards') loadRewards();
        if (tab === 'invest') loadInvest();
    }

    // === ASSETS ===
    async function loadAssets() {
        const el = document.getElementById('ed-assets');
        if (!el) return;
        el.innerHTML = `
            <div class="ed-grid">
                <button class="ed-btn" onclick="edAssetMint()">Mint</button>
                <button class="ed-btn" onclick="edAssetBurn()">Burn</button>
                <button class="ed-btn" onclick="edAssetTransfer()">Transfer</button>
                <button class="ed-btn" onclick="edAssetModify()">Modify</button>
                <button class="ed-btn" onclick="edAssetList()">List</button>
            </div>
            <div id="ed-assets-result" class="ed-result"></div>`;
    }
    async function edAssetMint() {
        const name = prompt('Asset name:');
        if (!name) return;
        try { await api('/assets/mint', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name }) }); showAssetsResult('Asset minted!'); }
        catch (e) { showAssetsResult('Failed: ' + e.message); }
    }
    async function edAssetBurn() {
        const id = prompt('Asset ID:');
        if (!id) return;
        try { await api('/assets/burn', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ asset_id: id }) }); showAssetsResult('Asset burned!'); }
        catch (e) { showAssetsResult('Failed: ' + e.message); }
    }
    async function edAssetTransfer() {
        const id = prompt('Asset ID:');
        const to = prompt('To wallet:');
        if (!id || !to) return;
        try { await api('/assets/transfer', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ asset_id: id, to }) }); showAssetsResult('Asset transferred!'); }
        catch (e) { showAssetsResult('Failed: ' + e.message); }
    }
    async function edAssetModify() {
        const id = prompt('Asset ID:');
        const data = prompt('Data (JSON):');
        if (!id) return;
        try { await api('/assets/modify', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ asset_id: id, data: data ? JSON.parse(data) : {} }) }); showAssetsResult('Asset modified!'); }
        catch (e) { showAssetsResult('Failed: ' + e.message); }
    }
    async function edAssetList() {
        try { const res = await api('/assets'); showAssetsResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showAssetsResult('Failed: ' + e.message); }
    }
    function showAssetsResult(msg) {
        const el = document.getElementById('ed-assets-result');
        if (el) el.innerHTML = `<pre style="color:#80cbc4;font-size:11px;white-space:pre-wrap;">${esc(msg)}</pre>`;
    }

    // === ORPHANS ===
    async function loadOrphans() {
        const el = document.getElementById('ed-orphans');
        if (!el) return;
        el.innerHTML = `
            <div class="ed-grid">
                <button class="ed-btn" onclick="edOrphanAdopt()">Adopt</button>
                <button class="ed-btn" onclick="edOrphanReclaim()">Reclaim</button>
                <button class="ed-btn" onclick="edOrphanStatus()">Status</button>
            </div>
            <div id="ed-orphans-result" class="ed-result"></div>`;
    }
    async function edOrphanAdopt() {
        const id = prompt('Entity ID:');
        if (!id) return;
        try { await api('/orphan/adopt', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ entity_id: id }) }); showOrphansResult('Adopted!'); }
        catch (e) { showOrphansResult('Failed: ' + e.message); }
    }
    async function edOrphanReclaim() {
        const id = prompt('Entity ID:');
        if (!id) return;
        try { await api('/orphan/reclaim', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ entity_id: id }) }); showOrphansResult('Reclaimed!'); }
        catch (e) { showOrphansResult('Failed: ' + e.message); }
    }
    async function edOrphanStatus() {
        const id = prompt('Entity ID:');
        if (!id) return;
        try { const res = await api('/orphan/status?entity_id=' + encodeURIComponent(id)); showOrphansResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showOrphansResult('Failed: ' + e.message); }
    }
    function showOrphansResult(msg) {
        const el = document.getElementById('ed-orphans-result');
        if (el) el.innerHTML = `<pre style="color:#80cbc4;font-size:11px;white-space:pre-wrap;">${esc(msg)}</pre>`;
    }

    // === OS ===
    async function loadOS() {
        const el = document.getElementById('ed-os');
        if (!el) return;
        el.innerHTML = `
            <div class="ed-grid">
                <button class="ed-btn" onclick="edOSModuleRegister()">Module Register</button>
                <button class="ed-btn" onclick="edOSSummary()">Summary</button>
            </div>
            <div id="ed-os-result" class="ed-result"></div>`;
    }
    async function edOSModuleRegister() {
        const name = prompt('Module name:');
        if (!name) return;
        try { await api('/os/module/register', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name }) }); showOSResult('Module registered!'); }
        catch (e) { showOSResult('Failed: ' + e.message); }
    }
    async function edOSSummary() {
        try { const res = await api('/os/summary'); showOSResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showOSResult('Failed: ' + e.message); }
    }
    function showOSResult(msg) {
        const el = document.getElementById('ed-os-result');
        if (el) el.innerHTML = `<pre style="color:#80cbc4;font-size:11px;white-space:pre-wrap;">${esc(msg)}</pre>`;
    }

    // === LAUNCHES ===
    async function loadLaunches() {
        const el = document.getElementById('ed-launches');
        if (!el) return;
        el.innerHTML = `
            <div class="ed-grid">
                <button class="ed-btn" onclick="edLaunchActivate()">Activate</button>
                <button class="ed-btn" onclick="edLaunchGet()">Get</button>
                <button class="ed-btn" onclick="edLaunchIntegrate()">Integrate</button>
            </div>
            <div id="ed-launches-result" class="ed-result"></div>`;
    }
    async function edLaunchActivate() {
        const id = prompt('Launch ID:');
        if (!id) return;
        try { await api('/launch/activate', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ launch_id: id }) }); showLaunchesResult('Activated!'); }
        catch (e) { showLaunchesResult('Failed: ' + e.message); }
    }
    async function edLaunchGet() {
        const id = prompt('Launch ID:');
        if (!id) return;
        try { const res = await api('/launch/get?launch_id=' + encodeURIComponent(id)); showLaunchesResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showLaunchesResult('Failed: ' + e.message); }
    }
    async function edLaunchIntegrate() {
        const id = prompt('Launch ID:');
        if (!id) return;
        try { await api('/launch/integrate', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ launch_id: id }) }); showLaunchesResult('Integrated!'); }
        catch (e) { showLaunchesResult('Failed: ' + e.message); }
    }
    function showLaunchesResult(msg) {
        const el = document.getElementById('ed-launches-result');
        if (el) el.innerHTML = `<pre style="color:#80cbc4;font-size:11px;white-space:pre-wrap;">${esc(msg)}</pre>`;
    }

    // === REWARDS ===
    async function loadRewards() {
        const el = document.getElementById('ed-rewards');
        if (!el) return;
        el.innerHTML = `
            <div class="ed-grid">
                <button class="ed-btn" onclick="edRewardAdd()">Add</button>
                <button class="ed-btn" onclick="edRewardRemove()">Remove</button>
                <button class="ed-btn" onclick="edRewardUpdateAsset()">Update Asset</button>
                <button class="ed-btn" onclick="edRewardUpdateBase()">Update Base</button>
            </div>
            <div id="ed-rewards-result" class="ed-result"></div>`;
    }
    async function edRewardAdd() {
        const name = prompt('Reward name:');
        if (!name) return;
        try { await api('/reward/add', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name }) }); showRewardsResult('Reward added!'); }
        catch (e) { showRewardsResult('Failed: ' + e.message); }
    }
    async function edRewardRemove() {
        const id = prompt('Reward ID:');
        if (!id) return;
        try { await api('/reward/remove', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ reward_id: id }) }); showRewardsResult('Reward removed!'); }
        catch (e) { showRewardsResult('Failed: ' + e.message); }
    }
    async function edRewardUpdateAsset() {
        const id = prompt('Reward ID:');
        const asset = prompt('Asset ID:');
        if (!id || !asset) return;
        try { await api('/reward/update-asset', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ reward_id: id, asset_id: asset }) }); showRewardsResult('Asset updated!'); }
        catch (e) { showRewardsResult('Failed: ' + e.message); }
    }
    async function edRewardUpdateBase() {
        const id = prompt('Reward ID:');
        const base = prompt('Base amount:');
        if (!id || !base) return;
        try { await api('/reward/update-base', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ reward_id: id, base_amount: parseInt(base) }) }); showRewardsResult('Base updated!'); }
        catch (e) { showRewardsResult('Failed: ' + e.message); }
    }
    function showRewardsResult(msg) {
        const el = document.getElementById('ed-rewards-result');
        if (el) el.innerHTML = `<pre style="color:#80cbc4;font-size:11px;white-space:pre-wrap;">${esc(msg)}</pre>`;
    }

    // === INVEST ===
    async function loadInvest() {
        const el = document.getElementById('ed-invest');
        if (!el) return;
        el.innerHTML = `
            <div class="ed-grid">
                <button class="ed-btn" onclick="edInvestEntity()">Entity</button>
                <button class="ed-btn" onclick="edInvestPortfolio()">Portfolio</button>
                <button class="ed-btn" onclick="edInvestDividendsHistory()">Dividends History</button>
            </div>
            <div id="ed-invest-result" class="ed-result"></div>`;
    }
    async function edInvestEntity() {
        const id = prompt('Entity ID:');
        if (!id) return;
        try { const res = await api('/invest/entity?entity_id=' + encodeURIComponent(id)); showInvestResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showInvestResult('Failed: ' + e.message); }
    }
    async function edInvestPortfolio() {
        try { const res = await api('/invest/portfolio'); showInvestResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showInvestResult('Failed: ' + e.message); }
    }
    async function edInvestDividendsHistory() {
        try { const res = await api('/invest/dividends/history'); showInvestResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showInvestResult('Failed: ' + e.message); }
    }
    function showInvestResult(msg) {
        const el = document.getElementById('ed-invest-result');
        if (el) el.innerHTML = `<pre style="color:#80cbc4;font-size:11px;white-space:pre-wrap;">${esc(msg)}</pre>`;
    }

    window.openExtendedDashboard = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        if (window.PanelManager) window.PanelManager.open(overlayEl);
        else overlayEl.style.display = 'flex';
        switchEDTab('assets');
    };
    window.switchEDTab = switchEDTab;

    // EVERY handler name this surface's rendered markup uses must exist on `window` — an inline
    // handler resolves its name there, and every function below is scoped to this IIFE. Without this
    // list all 20 controls (assets, orphans, OS modules, launches, rewards, invest) were inert.
    Object.assign(window, {
        edAssetBurn, edAssetList, edAssetMint, edAssetModify, edAssetTransfer,
        edInvestDividendsHistory, edInvestEntity, edInvestPortfolio,
        edLaunchActivate, edLaunchGet, edLaunchIntegrate, edOSModuleRegister, edOSSummary,
        edOrphanAdopt, edOrphanReclaim, edOrphanStatus,
        edRewardAdd, edRewardRemove, edRewardUpdateAsset, edRewardUpdateBase,
    });

    if (document.getElementById('extended-dashboard-overlay')) init();
})();
