// ============================================================================
// faction_shop.js — Faction Quartermaster (Careers & Factions)
// ----------------------------------------------------------------------------
// Owner of two engine routes that had NO UI anywhere in the client:
//   GET  /api/faction/shop/{JUSTICE|UNDERWORLD}  → HandleGetFactionShop
//   POST /api/faction/shop/buy                   → HandleBuyFactionItem
// It is its own module and its own leaf because the shop is gated on the CALLER'S CAREER ROLE —
// HandleBuyFactionItem refuses unless the player's JobRole matches the faction. That is career
// interactivity, so it belongs to Careers & Factions (two-tier mandate §5), not to an aggregator.
// PRICES ARE THE SERVER'S: the catalogue arrives with `cost_micro` and the buy body carries only
// the faction and the item name. The client never declares a price.
// ============================================================================
(function () {
    'use strict';
    var API_BASE = '/api';
    var state = { faction: 'JUSTICE', items: [], problems: [] };

    function getWallet() {
        return window.currentWallet || (window.getActiveWallet && window.getActiveWallet()) || '';
    }

    async function api(path, opts) {
        const url = path.startsWith('/api/') ? path : API_BASE + path;
        const resp = await fetch(url, opts);
        if (!resp.ok) { let d = ''; try { d = (await resp.json()).error || ''; } catch (e) {} throw new Error(d || ('HTTP ' + resp.status)); }
        return resp.json();
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }
    function fmtVBV(micro) { return (Number(micro) / 1000000).toFixed(2) + ' $VBV'; }
    function el() { return document.getElementById('wd-faction'); }

    async function load() {
        const host = el();
        if (!host) return;
        host.innerHTML = '<div class="wd-loading">Loading the quartermaster…</div>';
        state.problems = [];
        try {
            const res = await api('/faction/shop/' + state.faction);
            // The route answers {success, faction, items[]}; a refusal carries `error`.
            if (res.success === false) throw new Error(res.error || 'refused');
            state.items = Array.isArray(res.items) ? res.items : [];
        } catch (e) {
            state.items = [];
            state.problems.push(e.message);
        }
        render();
    }

    function render() {
        const host = el();
        if (!host) return;
        const wallet = getWallet();
        const rows = state.items.length
            ? state.items.map(i => `
                <div class="faction-item" data-item="${esc(i.id)}">
                    <span class="faction-item-name">${esc(i.name)}</span>
                    <span class="faction-item-desc">${esc(i.description || '')}</span>
                    <span class="faction-item-cost">${fmtVBV(i.cost_micro || 0)}</span>
                    <span class="faction-item-stack">max ${esc(i.max_stack == null ? '—' : i.max_stack)}</span>
                    <button class="vbt-btn vbt-btn-primary faction-buy" data-item="${esc(i.id)}">Buy</button>
                </div>`).join('')
            : '<p class="faction-note">The engine reported no items for this faction.</p>';

        host.innerHTML = `
            <div class="faction-shop-container">
                <div class="faction-header">
                    <span class="faction-icon">🎖️</span>
                    <div>
                        <h3 style="color:#e5e7eb;margin:0;">Faction Quartermaster</h3>
                        <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Buying is gated on your career role — the engine refuses a faction you do not serve.</p>
                    </div>
                </div>
                <div class="faction-tabs">
                    <button class="faction-tab${state.faction === 'JUSTICE' ? ' active' : ''}" data-faction="JUSTICE">Justice</button>
                    <button class="faction-tab${state.faction === 'UNDERWORLD' ? ' active' : ''}" data-faction="UNDERWORLD">Underworld</button>
                </div>
                <div class="faction-list">${rows}</div>
                <p class="faction-note" id="faction-status">${wallet ? '' : 'Connect a wallet to buy — the engine resolves the buyer from the request.'}${state.problems.length ? ' Could not read: ' + esc(state.problems.join('; ')) : ''}</p>
            </div>
        `;

        host.querySelectorAll('.faction-tab').forEach(b => b.addEventListener('click', () => {
            state.faction = b.dataset.faction;
            load();
        }));
        host.querySelectorAll('.faction-buy').forEach(b => b.addEventListener('click', () => buy(b.dataset.item)));
    }

    async function buy(itemId) {
        const status = document.getElementById('faction-status');
        const wallet = getWallet();
        if (!wallet) { if (status) status.textContent = 'Connect a wallet first.'; return; }
        try {
            // The body carries ONLY the faction and the item name — never a price.
            const res = await api('/faction/shop/buy?wallet=' + encodeURIComponent(wallet), {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ faction: state.faction, item_name: itemId }),
            });
            if (res.success === false) throw new Error(res.error || 'refused');
            if (status) status.textContent = 'Acquired ' + itemId + '.';
        } catch (e) {
            if (status) status.textContent = 'Refused: ' + e.message;
        }
    }

    window.initFactionShop = function () { load(); };
})();
