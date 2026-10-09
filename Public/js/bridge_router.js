// ============================================================================
// bridge_router.js — Multi-Chain Bridge (ETH/MATIC/BTC/SOL)
// ============================================================================
(function () {
    'use strict';
    var API_BASE = '/api';
    let overlayEl, assetsEl, txsEl, summaryEl, statusEl;

    function init() {
        if (overlayEl) return;
        const html = `
<div id="bridge-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel bridge-panel">
        <button id="btn-close-br" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>🌉 Bridge Router</h2>
        <p style="font-size:12px;color:#b0bec5;">Welcome every ecosystem. Ethereum, VOI, Polygon, Bitcoin, Solana.</p>
        <div class="br-tabs">
            <button class="br-tab active" data-tab="assets">Assets</button>
            <button class="br-tab" data-tab="bridge">Bridge</button>
            <button class="br-tab" data-tab="summary">Summary</button>
        </div>
        <div id="br-assets" class="br-assets"></div>
        <div id="br-bridge" class="br-bridge" style="display:none;">
            <select id="br-from">
                <option value="voi">Voi</option>
                <option value="algorand">Algorand</option>
                <option value="ethereum">Ethereum</option>
                <option value="polygon">Polygon</option>
                <option value="bitcoin">Bitcoin</option>
                <option value="solana">Solana</option>
            </select>
            <span>→</span>
            <select id="br-to">
                <option value="algorand">Algorand</option>
                <option value="voi">Voi</option>
                <option value="ethereum">Ethereum</option>
                <option value="polygon">Polygon</option>
                <option value="bitcoin">Bitcoin</option>
                <option value="solana">Solana</option>
            </select>
            <input id="br-asset-id" placeholder="Asset ID" />
            <input id="br-amount" placeholder="Amount (micro)" type="number" />
            <input id="br-recipient" placeholder="Recipient" />
            <button class="vbt-btn" id="br-bridge-btn">Bridge Asset</button>
        </div>
        <div id="br-summary" class="br-summary" style="display:none;"></div>
        <p id="br-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('bridge-overlay');
        statusEl = document.getElementById('br-status');
        document.getElementById('btn-close-br').addEventListener('click', () => { overlayEl.style.display = 'none'; });
        document.querySelectorAll('.br-tab').forEach(btn => {
            btn.addEventListener('click', () => {
                document.querySelectorAll('.br-tab').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                switchBrTab(btn.dataset.tab);
            });
        });
        document.getElementById('br-bridge-btn').addEventListener('click', bridgeAsset);
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>\"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }
    function fmtVBV(micro) { return (parseFloat(micro) / 1000000).toFixed(2); }

    async function api(path, opts) {
        const url = path.startsWith('/api/') ? path : API_BASE + path;
        const resp = await fetch(url, opts);
        if (!resp.ok) { let d = ''; try { d = (await resp.json()).error || ''; } catch(e){} throw new Error(d || ('HTTP ' + resp.status)); }
        return resp.json();
    }

    function switchBrTab(tab) {
        document.querySelectorAll('.br-tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab)); document.getElementById('br-assets').style.display = tab === 'assets' ? 'block' : 'none';
        document.getElementById('br-bridge').style.display = tab === 'bridge' ? 'block' : 'none';
        document.getElementById('br-summary').style.display = tab === 'summary' ? 'block' : 'none';
        if (tab === 'assets') loadAssets();
        if (tab === 'summary') loadSummary();
    }

    async function loadAssets() {
        try {
            const res = await api('/api/bridge/assets');
            const el = document.getElementById('br-assets');
            if (!el) return;
            if (!(res.assets?.length ?? 0)) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No bridged assets.</p>'; return; }
            el.innerHTML = res.assets.map(a => `
                <div class="br-asset">
                    <span class="br-chain">${esc(a.chain)}</span>
                    <span class="br-origin">from ${esc(a.origin_chain)}</span>
                    <span class="br-amount">${fmtVBV(a.amount_micro)}</span>
                    <span class="br-status">${esc(a.status)}</span>
                </div>
            `).join('');
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    async function loadSummary() {
        try {
            const res = await api('/api/bridge/summary');
            const s = res.summary;
            const el = document.getElementById('br-summary');
            if (!el) return;
            el.innerHTML = `
                <div class="br-stat"><span>Total Assets</span><span>${s.total_assets}</span></div>
                <div class="br-stat"><span>Total Transactions</span><span>${s.total_txs}</span></div>
                <div class="br-chains">
                    ${Object.entries(s.chain_counts || {}).map(([k,v]) => `<span class="br-chain-tag">${esc(k)}: ${v}</span>`).join('')}
                </div>
            `;
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    async function bridgeAsset() {
        try {
            const body = {
                from_chain: document.getElementById('br-from')?.value,
                to_chain: document.getElementById('br-to')?.value,
                asset_id: document.getElementById('br-asset-id')?.value,
                amount: parseInt(document.getElementById('br-amount')?.value) || 0,
                recipient: document.getElementById('br-recipient')?.value
            };
            await api('/api/bridge/asset', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
            setStatus('Bridge initiated!');
            loadAssets();
        } catch (e) { setStatus('Bridge failed: ' + e.message); }
    }

    function setStatus(m) { if (statusEl) statusEl.textContent = m || ''; }

    window.openBridgeRouter = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        loadAssets();
    };
})();
