// ============================================================================
// utilities_dashboard.js — Ads, Bridge, Cards, Rewards, Industrial, Stat Overlay
// ----------------------------------------------------------------------------
// Wires 15 orphaned backend routes:
//   Ads:        /api/ads, /api/ad/activate, /api/ad/click, /api/ad/impression,
//                /api/ad/pause, /api/ad/stats
//   Bridge:     /api/bridge/confirm, /api/bridge/txs
//   Cards:      /api/card-stats
//   Rewards:    /api/reward
//   Industrial: /api/industrial-loop/record
//   Stat:       /api/stat-overlay/owner, /api/stat-overlay/region
//   Client:     /api/client-error
// ============================================================================

(function () {
    'use strict';
    var API_BASE = '/api';
    let overlayEl = null;

    function init() {
        if (overlayEl) return;
        overlayEl = document.getElementById('utilities-dashboard-overlay');
        if (!overlayEl) {
            // SELF-MOUNT. Nothing in the shell provided this root (no `id="utilities-dashboard-overlay"`
            // in index.html), so the null-check below made the whole module a silent no-op: `init()`
            // returned and all 14 controls this file publishes had no surface to render into.
            document.body.insertAdjacentHTML('beforeend', '<div id="utilities-dashboard-overlay"></div>');
            overlayEl = document.getElementById('utilities-dashboard-overlay');
            if (!overlayEl) return;
        }
        overlayEl.className = 'overlay vbt-overlay';
        overlayEl.style.display = 'none';

        overlayEl.innerHTML = `
<div class="neon-glass-panel utilities-dashboard-panel">
    <button id="btn-close-ud" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
    <h2>🔧 Utilities Dashboard</h2>
    <p style="font-size:12px;color:#b0bec5;margin-top:-8px;">Ads · Bridge · Cards · Rewards · Industrial · Stats</p>
    <div class="ud-tabs">
        <button class="ud-tab active" data-tab="ads" onclick="switchUDTab('ads')">Ads</button>
        <button class="ud-tab" data-tab="bridge" onclick="switchUDTab('bridge')">Bridge</button>
        <button class="ud-tab" data-tab="rewards" onclick="switchUDTab('rewards')">Rewards</button>
        <button class="ud-tab" data-tab="industrial" onclick="switchUDTab('industrial')">Industrial</button>
        <button class="ud-tab" data-tab="stats" onclick="switchUDTab('stats')">Stat Overlay</button>
    </div>
    <div id="ud-panel-ads" class="ud-panel"><div id="ud-ads"></div></div>
    <div id="ud-panel-bridge" class="ud-panel hidden"><div id="ud-bridge"></div></div>
    <div id="ud-panel-rewards" class="ud-panel hidden"><div id="ud-rewards"></div></div>
    <div id="ud-panel-industrial" class="ud-panel hidden"><div id="ud-industrial"></div></div>
    <div id="ud-panel-stats" class="ud-panel hidden"><div id="ud-stats"></div></div>
    <p id="ud-status" class="ai-status"></p>
</div>`;

        document.getElementById('btn-close-ud').addEventListener('click', () => { overlayEl.style.display = 'none'; });
        loadAds();
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

    function switchUDTab(tab) {
        document.querySelectorAll('.ud-tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab));
        document.querySelectorAll('.ud-panel').forEach(p => p.classList.add('hidden'));
        const panel = document.getElementById('ud-panel-' + tab);
        if (panel) panel.classList.remove('hidden');
        if (tab === 'ads') loadAds();
        if (tab === 'bridge') loadBridge();
        if (tab === 'rewards') loadRewards();
        if (tab === 'industrial') loadIndustrial();
        if (tab === 'stats') loadStats();
    }

    // === ADS ===
    async function loadAds() {
        const el = document.getElementById('ud-ads');
        if (!el) return;
        el.innerHTML = `
            <div class="ud-grid">
                <button class="ud-btn" onclick="udAdActivate()">Activate</button>
                <button class="ud-btn" onclick="udAdPause()">Pause</button>
                <button class="ud-btn" onclick="udAdClick()">Click</button>
                <button class="ud-btn" onclick="udAdImpression()">Impression</button>
                <button class="ud-btn" onclick="udAdStats()">Stats</button>
                <button class="ud-btn" onclick="udListAds()">List Ads</button>
            </div>
            <div id="ud-ads-result" class="ud-result"></div>`;
    }
    async function udAdActivate() {
        const id = prompt('Ad ID:');
        if (!id) return;
        try { await api('/ad/activate', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ ad_id: id }) }); showAdsResult('Ad activated!'); }
        catch (e) { showAdsResult('Failed: ' + e.message); }
    }
    async function udAdPause() {
        const id = prompt('Ad ID:');
        if (!id) return;
        try { await api('/ad/pause', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ ad_id: id }) }); showAdsResult('Ad paused!'); }
        catch (e) { showAdsResult('Failed: ' + e.message); }
    }
    async function udAdClick() {
        const id = prompt('Ad ID:');
        if (!id) return;
        try { await api('/ad/click', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ ad_id: id }) }); showAdsResult('Click recorded!'); }
        catch (e) { showAdsResult('Failed: ' + e.message); }
    }
    async function udAdImpression() {
        const id = prompt('Ad ID:');
        if (!id) return;
        try { await api('/ad/impression', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ ad_id: id }) }); showAdsResult('Impression recorded!'); }
        catch (e) { showAdsResult('Failed: ' + e.message); }
    }
    async function udAdStats() {
        const id = prompt('Ad ID:');
        if (!id) return;
        try { const res = await api('/ad/stats?ad_id=' + encodeURIComponent(id)); showAdsResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showAdsResult('Failed: ' + e.message); }
    }
    async function udListAds() {
        try { const res = await api('/ads'); showAdsResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showAdsResult('Failed: ' + e.message); }
    }
    function showAdsResult(msg) {
        const el = document.getElementById('ud-ads-result');
        if (el) el.innerHTML = `<pre style="color:#80cbc4;font-size:11px;white-space:pre-wrap;">${esc(msg)}</pre>`;
    }

    // === BRIDGE ===
    async function loadBridge() {
        const el = document.getElementById('ud-bridge');
        if (!el) return;
        el.innerHTML = `
            <div class="ud-grid">
                <button class="ud-btn" onclick="udBridgeConfirm()">Confirm</button>
                <button class="ud-btn" onclick="udBridgeTxs()">Transactions</button>
            </div>
            <div id="ud-bridge-result" class="ud-result"></div>`;
    }
    async function udBridgeConfirm() {
        const txid = prompt('Transaction ID:');
        if (!txid) return;
        try { await api('/bridge/confirm', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ txid }) }); showBridgeResult('Confirmed!'); }
        catch (e) { showBridgeResult('Failed: ' + e.message); }
    }
    async function udBridgeTxs() {
        try { const res = await api('/bridge/txs'); showBridgeResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showBridgeResult('Failed: ' + e.message); }
    }
    function showBridgeResult(msg) {
        const el = document.getElementById('ud-bridge-result');
        if (el) el.innerHTML = `<pre style="color:#80cbc4;font-size:11px;white-space:pre-wrap;">${esc(msg)}</pre>`;
    }

    // === REWARDS ===
    async function loadRewards() {
        const el = document.getElementById('ud-rewards');
        if (!el) return;
        el.innerHTML = `
            <div class="ud-grid">
                <button class="ud-btn" onclick="udGetReward()">Get Reward</button>
                <button class="ud-btn" onclick="udSendClientError()">Send Client Error</button>
            </div>
            <div id="ud-rewards-result" class="ud-result"></div>`;
    }
    async function udGetReward() {
        const id = prompt('Reward ID:');
        if (!id) return;
        try { const res = await api('/reward?id=' + encodeURIComponent(id)); showRewardsResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showRewardsResult('Failed: ' + e.message); }
    }
    async function udSendClientError() {
        const msg = prompt('Error message:');
        if (!msg) return;
        try { await api('/client-error', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ message: msg }) }); showRewardsResult('Error sent!'); }
        catch (e) { showRewardsResult('Failed: ' + e.message); }
    }
    function showRewardsResult(msg) {
        const el = document.getElementById('ud-rewards-result');
        if (el) el.innerHTML = `<pre style="color:#80cbc4;font-size:11px;white-space:pre-wrap;">${esc(msg)}</pre>`;
    }

    // === INDUSTRIAL LOOP ===
    async function loadIndustrial() {
        const el = document.getElementById('ud-industrial');
        if (!el) return;
        el.innerHTML = `
            <div class="ud-grid">
                <button class="ud-btn" onclick="udIndustrialRecord()">Record</button>
            </div>
            <div id="ud-industrial-result" class="ud-result"></div>`;
    }
    async function udIndustrialRecord() {
        const data = prompt('Record data (JSON):');
        if (!data) return;
        try { await api('/industrial-loop/record', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: data }); showIndustrialResult('Recorded!'); }
        catch (e) { showIndustrialResult('Failed: ' + e.message); }
    }
    function showIndustrialResult(msg) {
        const el = document.getElementById('ud-industrial-result');
        if (el) el.innerHTML = `<pre style="color:#80cbc4;font-size:11px;white-space:pre-wrap;">${esc(msg)}</pre>`;
    }

    // === STAT OVERLAY ===
    async function loadStats() {
        const el = document.getElementById('ud-stats');
        if (!el) return;
        el.innerHTML = `
            <div class="ud-grid">
                <button class="ud-btn" onclick="udStatOwner()">Owner</button>
                <button class="ud-btn" onclick="udStatRegion()">Region</button>
                <button class="ud-btn" onclick="udCardStats()">Card Stats</button>
            </div>
            <div id="ud-stats-result" class="ud-result"></div>`;
    }
    async function udStatOwner() {
        const wallet = prompt('Wallet:');
        if (!wallet) return;
        try { const res = await api('/stat-overlay/owner?wallet=' + encodeURIComponent(wallet)); showStatsResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showStatsResult('Failed: ' + e.message); }
    }
    async function udStatRegion() {
        const region = prompt('Region:');
        if (!region) return;
        try { const res = await api('/stat-overlay/region?region=' + encodeURIComponent(region)); showStatsResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showStatsResult('Failed: ' + e.message); }
    }
    async function udCardStats() {
        const id = prompt('Card ID:');
        if (!id) return;
        try { const res = await api('/card-stats?id=' + encodeURIComponent(id)); showStatsResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showStatsResult('Failed: ' + e.message); }
    }
    function showStatsResult(msg) {
        const el = document.getElementById('ud-stats-result');
        if (el) el.innerHTML = `<pre style="color:#80cbc4;font-size:11px;white-space:pre-wrap;">${esc(msg)}</pre>`;
    }

    window.openUtilitiesDashboard = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        if (window.PanelManager) window.PanelManager.open(overlayEl);
        else overlayEl.style.display = 'flex';
        switchUDTab('ads');
    };
    window.switchUDTab = switchUDTab;

    // EVERY handler name this surface's rendered markup uses must exist on `window` — an inline
    // handler resolves its name there, and every function below is scoped to this IIFE. Without this
    // list all 14 controls (ads, bridge, cards, rewards, industrial, stat overlay) were inert.
    Object.assign(window, {
        udAdActivate, udAdClick, udAdImpression, udAdPause, udAdStats, udListAds,
        udBridgeConfirm, udBridgeTxs, udCardStats, udGetReward, udIndustrialRecord,
        udSendClientError, udStatOwner, udStatRegion,
    });

    if (document.getElementById('utilities-dashboard-overlay')) init();
})();
