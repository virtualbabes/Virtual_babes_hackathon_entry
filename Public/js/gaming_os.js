// ============================================================================
// gaming_os.js — Gaming OS + Compliance System
// ----------------------------------------------------------------------------
// Gaming OS = civilization-as-a-service. The infrastructure beneath games.
// Compliance = regulatory hooks, KYC/AML flags, audit trails, reporting.
// ============================================================================
(function () {
    'use strict';
    var API_BASE = '/api';
    let overlayEl, modulesEl, leasesEl, complianceEl, statusEl;

    function init() {
        if (overlayEl) return;
        const html = `
<div id="gaming-os-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel gaming-os-panel">
        <button id="btn-close-go" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>🖥️ Gaming OS + Compliance</h2>
        <p style="font-size:12px;color:#b0bec5;">Civilization-as-a-service. Lease OS modules. Compliance records, KYC/AML, audit trails.</p>
        <div class="go-tabs">
            <button class="go-tab active" data-tab="modules">OS Modules</button>
            <button class="go-tab" data-tab="leases">My Leases</button>
            <button class="go-tab" data-tab="compliance">Compliance</button>
        </div>
        <div id="go-modules" class="go-modules"></div>
        <div id="go-leases" class="go-leases" style="display:none;"></div>
        <div id="go-compliance" class="go-compliance" style="display:none;"></div>
        <p id="go-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('gaming-os-overlay');
        statusEl = document.getElementById('go-status');
        document.getElementById('btn-close-go').addEventListener('click', () => { overlayEl.style.display = 'none'; });
        document.querySelectorAll('.go-tab').forEach(btn => {
            btn.addEventListener('click', () => {
                document.querySelectorAll('.go-tab').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                switchGoTab(btn.dataset.tab);
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

    function switchGoTab(tab) {
        document.querySelectorAll('.go-tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab)); document.getElementById('go-modules').style.display = tab === 'modules' ? 'block' : 'none';
        document.getElementById('go-leases').style.display = tab === 'leases' ? 'block' : 'none';
        document.getElementById('go-compliance').style.display = tab === 'compliance' ? 'block' : 'none';
        if (tab === 'modules') loadModules();
        if (tab === 'leases') loadLeases();
        if (tab === 'compliance') loadCompliance();
    }

    async function loadModules() {
        try {
            const res = await api('/api/os/modules');
            const el = document.getElementById('go-modules');
            if (!el) return;
            if (!(res.modules?.length ?? 0)) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No OS modules yet.</p>'; return; }
            el.innerHTML = res.modules.map(m => `
                <div class="go-module">
                    <span class="go-name">${esc(m?.name)}</span>
                    <span class="go-category">${esc(m.category)}</span>
                    <span class="go-rate">${fmtVBV(m.monthly_rate)} VBV/mo</span>
                    <span class="go-leases">${m.lease_count} leases</span>
                    <button class="vbt-btn vbt-btn-sm" onclick="window.leaseModule('${m.id}')">Lease</button>
                </div>
            `).join('');
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    async function loadLeases() {
        try {
            const res = await api('/api/os/leases');
            const el = document.getElementById('go-leases');
            if (!el) return;
            if (!(res.leases?.length ?? 0)) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No active leases.</p>'; return; }
            el.innerHTML = res.leases.map(l => `
                <div class="go-lease">
                    <span class="go-module-id">${esc(l.module_id)}</span>
                    <span class="go-rate">${fmtVBV(l.monthly_rate)} VBV/mo</span>
                    <span class="go-status ${l.active ? 'active' : 'expired'}">${l.active ? 'ACTIVE' : 'EXPIRED'}</span>
                </div>
            `).join('');
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    async function loadCompliance() {
        try {
            const res = await api('/api/compliance/records');
            const el = document.getElementById('go-compliance');
            if (!el) return;
            if (!(res.records?.length ?? 0)) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No compliance records.</p>'; return; }
            el.innerHTML = res.records.map(r => `
                <div class="go-record ${esc(r.severity)}">
                    <span class="go-record-type">${esc(r.record_type)}</span>
                    <span class="go-record-desc">${esc(r.description)}</span>
                    <span class="go-record-status">${esc(r.status)}</span>
                    <span class="go-record-severity">${esc(r.severity)}</span>
                </div>
            `).join('');
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    function setStatus(m) { if (statusEl) statusEl.textContent = m || ''; }

    window.leaseModule = async function (moduleID) {
        try {
            await api('/api/os/lease', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ module_id: moduleID, duration_days: 30 }) });
            setStatus('Module leased!');
            loadLeases();
        } catch (e) { setStatus('Lease failed: ' + e.message); }
    };

    window.openGamingOS = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        loadModules();
    };
})();
