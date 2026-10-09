// ============================================================================
// entity_shares.js — Entity Shares Market (invest in people, businesses, creators)
// ============================================================================
(function () {
    'use strict';
    var API_BASE = '/api';
    let overlayEl, tokensEl, holdingsEl, statusEl;

    function init() {
        if (overlayEl) return;
        const html = `
<div id="shares-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel shares-panel">
        <button id="btn-close-sh" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>📈 Entity Shares</h2>
        <p style="font-size:12px;color:#b0bec5;">Invest in players, businesses, creators, guilds, projects, communities.</p>
        <div class="sh-tabs">
            <button class="sh-tab active" data-tab="tokens">Tokens</button>
            <button class="sh-tab" data-tab="holdings">Holdings</button>
            <button class="sh-tab" data-tab="issue">Issue</button>
        </div>
        <div id="sh-tokens" class="sh-tokens"></div>
        <div id="sh-holdings" class="sh-holdings" style="display:none;"></div>
        <div id="sh-issue" class="sh-issue" style="display:none;">
            <input id="sh-name" placeholder="Name" />
            <input id="sh-symbol" placeholder="Symbol" />
            <input id="sh-supply" placeholder="Supply" type="number" />
            <input id="sh-price" placeholder="Price (micro)" type="number" />
            <input id="sh-dividend" placeholder="Dividend Rate" type="number" />
            <button class="vbt-btn" id="sh-issue-btn">Issue Token</button>
        </div>
        <p id="sh-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('shares-overlay');
        statusEl = document.getElementById('sh-status');
        document.getElementById('btn-close-sh').addEventListener('click', () => { overlayEl.style.display = 'none'; });
        document.querySelectorAll('.sh-tab').forEach(btn => {
            btn.addEventListener('click', () => {
                document.querySelectorAll('.sh-tab').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                switchShTab(btn.dataset.tab);
            });
        });
        document.getElementById('sh-issue-btn').addEventListener('click', issueToken);
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>\"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }
    function fmtVBV(micro) { return (parseFloat(micro) / 1000000).toFixed(2); }

    async function api(path, opts) {
        const url = path.startsWith('/api/') ? path : API_BASE + path;
        const resp = await fetch(url, opts);
        if (!resp.ok) { let d = ''; try { d = (await resp.json()).error || ''; } catch(e){} throw new Error(d || ('HTTP ' + resp.status)); }
        return resp.json();
    }

    function switchShTab(tab) {
        document.querySelectorAll('.sh-tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab)); document.getElementById('sh-tokens').style.display = tab === 'tokens' ? 'block' : 'none';
        document.getElementById('sh-holdings').style.display = tab === 'holdings' ? 'block' : 'none';
        document.getElementById('sh-issue').style.display = tab === 'issue' ? 'block' : 'none';
        if (tab === 'tokens') loadTokens();
        if (tab === 'holdings') loadHoldings();
    }

    async function loadTokens() {
        try {
            const res = await api('/api/shares/tokens');
            const el = document.getElementById('sh-tokens');
            if (!el) return;
            if (!(res.tokens?.length ?? 0)) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No tokens yet.</p>'; return; }
            el.innerHTML = res.tokens.map(t => `
                <div class="sh-token">
                    <span class="sh-symbol">${esc(t.symbol)}</span>
                    <span class="sh-name">${esc(t?.name)}</span>
                    <span class="sh-type">${esc(t.issuer_type)}</span>
                    <span class="sh-price">${fmtVBV(t.price_micro)} VBV</span>
                    <button class="vbt-btn vbt-btn-sm" onclick="window.buyShare('${t.id}')">Buy</button>
                </div>
            `).join('');
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    async function loadHoldings() {
        try {
            const res = await api('/api/shares/holdings');
            const el = document.getElementById('sh-holdings');
            if (!el) return;
            if (!(res.holdings?.length ?? 0)) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No holdings yet.</p>'; return; }
            el.innerHTML = res.holdings.map(h => `
                <div class="sh-holding">
                    <span class="sh-symbol">${esc(h.token_id.slice(0, 12))}…</span>
                    <span class="sh-qty">${h.shares}</span>
                </div>
            `).join('');
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    async function issueToken() {
        try {
            const body = {
                name: document.getElementById('sh-name')?.value,
                symbol: document.getElementById('sh-symbol')?.value,
                supply: parseInt(document.getElementById('sh-supply')?.value) || 0,
                price: parseInt(document.getElementById('sh-price')?.value) || 0,
                dividend: parseInt(document.getElementById('sh-dividend')?.value) || 0,
                issuer_type: 'player',
                issuer_id: 'self'
            };
            await api('/api/shares/issue', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
            setStatus('Token issued!');
            loadTokens();
        } catch (e) { setStatus('Issue failed: ' + e.message); }
    }

    function setStatus(m) { if (statusEl) statusEl.textContent = m || ''; }

    window.buyShare = async function (tokenID) {
        try {
            await api('/api/shares/buy', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ token_id: tokenID, amount: 1 }) });
            setStatus('Share purchased!');
            loadHoldings();
        } catch (e) { setStatus('Buy failed: ' + e.message); }
    };

    window.openEntityShares = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        loadTokens();
    };
})();
