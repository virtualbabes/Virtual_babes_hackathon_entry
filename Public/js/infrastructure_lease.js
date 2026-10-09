// ============================================================================
// infrastructure_lease.js — Infrastructure Leasing System
// ----------------------------------------------------------------------------
// Developers rent mature systems. Businesses rent economic systems.
// Communities rent social systems.
// ============================================================================
(function () {
    'use strict';
    var API_BASE = '/api';
    let overlayEl, leasesEl, availableEl, statusEl;

    function init() {
        if (overlayEl) return;
        const html = `
<div id="lease-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel lease-panel">
        <button id="btn-close-ls" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>🏗️ Infrastructure Leasing</h2>
        <p style="font-size:12px;color:#b0bec5;">Rent systems, don't rebuild. Wallet, AI, economy, tournaments.</p>
        <div class="ls-tabs">
            <button class="ls-tab active" data-tab="available">Available</button>
            <button class="ls-tab" data-tab="leases">My Leases</button>
        </div>
        <div id="ls-available" class="ls-available"></div>
        <div id="ls-leases" class="ls-leases" style="display:none;"></div>
        <p id="ls-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('lease-overlay');
        statusEl = document.getElementById('ls-status');
        document.getElementById('btn-close-ls').addEventListener('click', () => { overlayEl.style.display = 'none'; });
        document.querySelectorAll('.ls-tab').forEach(btn => {
            btn.addEventListener('click', () => {
                document.querySelectorAll('.ls-tab').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                switchLsTab(btn.dataset.tab);
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

    function switchLsTab(tab) {
        document.querySelectorAll('.ls-tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab)); document.getElementById('ls-available').style.display = tab === 'available' ? 'block' : 'none';
        document.getElementById('ls-leases').style.display = tab === 'leases' ? 'block' : 'none';
        if (tab === 'available') loadAvailable();
        if (tab === 'leases') loadLeases();
    }

    async function loadAvailable() {
        try {
            const res = await api('/api/lease/available');
            const el = document.getElementById('ls-available');
            if (!el) return;
            const systems = Array.isArray(res.systems) ? res.systems : [];
            const blocked = Array.isArray(res.creation_blocked_by) ? res.creation_blocked_by : [];
            // Leasing a TENANT WORLD is design-only: the route states why, so the panel states it
            // rather than offering a Lease control that can only ever refuse. No surface may look
            // like a working one.
            const notice = res.creation_available === false
                ? '<p style="color:#ffb74d;font-size:12px;">A lease cannot be created yet. Blocked by: ' +
                  blocked.map(b => esc(b)).join('; ') + '</p>'
                : '';
            el.innerHTML = notice + systems.map(s => `
                <div class="ls-system">
                    <span class="ls-icon">${esc(s?.name.split(' ')[0])}</span>
                    <span class="ls-name">${esc(s?.name)}</span>
                    <span class="ls-desc">${esc(s.description)}</span>
                    <span class="ls-rate">${fmtVBV(s.monthly_rate)} VBV/mo</span>
                </div>
            `).join('');
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    async function loadLeases() {
        try {
            const res = await api('/api/lease/list');
            const el = document.getElementById('ls-leases');
            if (!el) return;
            if (!(res.leases?.length ?? 0)) {
                el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No active leases: a tenant world lease cannot be created yet, so none can exist.</p>';
                return;
            }
            el.innerHTML = res.leases.map(l => `
                <div class="ls-lease">
                    <span class="ls-name">${esc(l.system_type)}</span>
                    <span class="ls-rate">${fmtVBV(l.monthly_rate)} VBV/mo</span>
                    <span class="ls-status ${l.active ? 'active' : 'expired'}">${l.active ? 'ACTIVE' : 'EXPIRED'}</span>
                </div>
            `).join('');
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    function setStatus(m) { if (statusEl) statusEl.textContent = m || ''; }

    // The body names only WHAT would be leased and for how long. A RATE is never the client's — the
    // server refuses (no tenant world runtime, no billing run) and names every blocker, and this
    // reports the refusal verbatim instead of claiming a lease was created.
    window.createLease = async function (systemType) {
        try {
            const res = await api('/api/lease/create', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ system_type: systemType, duration_days: 30 })
            });
            setStatus(res && res.success ? 'Lease created.' : 'Lease refused: ' + ((res && res.error) || 'no reason was served'));
            loadLeases();
        } catch (e) { setStatus('Lease refused: ' + e.message); }
    };

    window.openInfrastructureLease = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        loadAvailable();
    };
})();
