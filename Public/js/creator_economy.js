// ============================================================================
// creator_economy.js — Creator Economy System (DLC, royalties, subscriptions)
// ============================================================================
(function () {
    'use strict';
    var API_BASE = '/api';
    let overlayEl, dlcsEl, eventsEl, royaltiesEl, subsEl, statusEl;

    function init() {
        if (overlayEl) return;
        const html = `
<div id="creator-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel creator-panel">
        <button id="btn-close-cr" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>🎨 Creator Economy</h2>
        <p style="font-size:12px;color:#b0bec5;">Launch products, sell DLC, run events, earn royalties, build community.</p>
        <div class="cr-tabs">
            <button class="cr-tab active" data-tab="dlcs">DLC</button>
            <button class="cr-tab" data-tab="events">Events</button>
            <button class="cr-tab" data-tab="royalties">Royalties</button>
            <button class="cr-tab" data-tab="subs">Subscriptions</button>
        </div>
        <div id="cr-dlcs" class="cr-dlcs"></div>
        <div id="cr-events" class="cr-events" style="display:none;"></div>
        <div id="cr-royalties" class="cr-royalties" style="display:none;"></div>
        <div id="cr-subs" class="cr-subs" style="display:none;"></div>
        <p id="cr-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('creator-overlay');
        statusEl = document.getElementById('cr-status');
        document.getElementById('btn-close-cr').addEventListener('click', () => { overlayEl.style.display = 'none'; });
        document.querySelectorAll('.cr-tab').forEach(btn => {
            btn.addEventListener('click', () => {
                document.querySelectorAll('.cr-tab').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                switchCeTab(btn.dataset.tab);
            });
        });
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>\"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }
    function fmtVBV(micro) { return (parseFloat(micro) / 1000000).toFixed(2); }

    async function api(path, opts) {
        const url = path.startsWith('/api/') ? path : API_BASE + path;
        const resp = await fetch(url, opts);
        if (!resp.ok) { let d = ''; try { d = (await resp.json()).error || ''; } catch(e){} throw new Error(d || ('HTTP ' + resp.status)); }
        return resp.json();
    }

    function switchCeTab(tab) {
        document.getElementById('cr-dlcs').style.display = tab === 'dlcs' ? 'block' : 'none';
        document.getElementById('cr-events').style.display = tab === 'events' ? 'block' : 'none';
        document.getElementById('cr-royalties').style.display = tab === 'royalties' ? 'block' : 'none';
        document.getElementById('cr-subs').style.display = tab === 'subs' ? 'block' : 'none';
        if (tab === 'dlcs') loadDLCs();
        if (tab === 'events') loadEvents();
        if (tab === 'royalties') loadRoyalties();
        if (tab === 'subs') loadSubs();
    }

    async function loadDLCs() {
        try {
            const res = await api('/api/creator/dlcs');
            const el = document.getElementById('cr-dlcs');
            if (!el) return;
            if (!(res.dlcs?.length ?? 0)) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No DLC yet.</p>'; return; }
            el.innerHTML = res.dlcs.map(d => `
                <div class="cr-dlc">
                    <span class="cr-dlc-title">${esc(d.title)}</span>
                    <span class="cr-dlc-category">${esc(d.category)}</span>
                    <span class="cr-dlc-price">${fmtVBV(d.price_micro)} VBV</span>
                    <span class="cr-dlc-sales">${d.sales_count} sold</span>
                    <button class="vbt-btn vbt-btn-sm" onclick="window.purchaseDLC('${d.id}')">Buy</button>
                </div>
            `).join('');
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    async function loadEvents() {
        try {
            const res = await api('/api/creator/events');
            const el = document.getElementById('cr-events');
            if (!el) return;
            if (!(res.events?.length ?? 0)) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No events yet.</p>'; return; }
            el.innerHTML = res.events.map(e => `
                <div class="cr-event">
                    <span class="cr-event-title">${esc(e.title)}</span>
                    <span class="cr-event-date">${e.event_date}</span>
                    <span class="cr-event-attendees">${e.attendees?.length ?? 0}/${e.max_attendees}</span>
                    <button class="vbt-btn vbt-btn-sm" onclick="window.attendEvent('${e.id}')">Attend</button>
                </div>
            `).join('');
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    async function loadRoyalties() {
        try {
            const res = await api('/api/creator/royalties');
            const el = document.getElementById('cr-royalties');
            if (!el) return;
            if (!(res.royalties?.length ?? 0)) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No royalties yet.</p>'; return; }
            el.innerHTML = res.royalties.map(r => `
                <div class="cr-royalty">
                    <span class="cr-royalty-source">${esc(r.source)}</span>
                    <span class="cr-royalty-amount">${fmtVBV(r.amount)} VBV</span>
                </div>
            `).join('');
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    async function loadSubs() {
        try {
            const res = await api('/api/creator/subs');
            const el = document.getElementById('cr-subs');
            if (!el) return;
            if (!(res.subscriptions?.length ?? 0)) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No subscriptions.</p>'; return; }
            el.innerHTML = res.subscriptions.map(s => `
                <div class="cr-sub">
                    <span class="cr-sub-creator">${esc(s.creator.slice(0, 12))}…</span>
                    <span class="cr-sub-amount">${fmtVBV(s.amount_micro)} VBV/${s.frequency}</span>
                </div>
            `).join('');
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    function setStatus(m) { if (statusEl) statusEl.textContent = m || ''; }

    window.purchaseDLC = async function (dlcID) {
        try {
            await api('/api/creator/dlc/purchase', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ dlc_id: dlcID }) });
            setStatus('DLC purchased!');
        } catch (e) { setStatus('Purchase failed: ' + e.message); }
    };

    window.attendEvent = async function (eventID) {
        try {
            await api('/api/creator/event/attend', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ event_id: eventID }) });
            setStatus('Attending event!');
            loadEvents();
        } catch (e) { setStatus('Attend failed: ' + e.message); }
    };

    window.openCreatorEconomy = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        loadDLCs();
    };
})();
