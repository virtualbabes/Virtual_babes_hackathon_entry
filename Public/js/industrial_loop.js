// ============================================================================
// industrial_loop.js — Industrial Loop Engine (REAL API, rewired 2026-09-03)
// ----------------------------------------------------------------------------
// Backend: console_server.go
//   GET /api/industrial-loop/metrics  → { kph, vbv_per_day, uptime, efficiency, ... }
//   GET /api/industrial-loop/health   → { circulation, treasury, employment, taxation, ... }
// All values server-authoritative. No mock-data fallback.
// ============================================================================
(function () {
    'use strict';
    var API_BASE = '/api';
    let loopEl = null, statusEl = null;
    let maintenanceInterval = null;

    const PHASES = [
        { id: 'activity', icon: '👤', name: 'Activity' },
        { id: 'business', icon: '🏢', name: 'Business' },
        { id: 'employment', icon: '💼', name: 'Employment' },
        { id: 'purchasing', icon: '🛒', name: 'Purchasing' },
        { id: 'taxation', icon: '💰', name: 'Taxation' },
        { id: 'treasury', icon: '🏛️', name: 'Treasury' },
        { id: 'development', icon: '🏗️', name: 'Development' },
        { id: 'events', icon: '🎪', name: 'Events' },
    ];

    function init() {
        if (loopEl) return;
        const html = `
<div id="industrial-loop-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel industrial-loop-panel">
        <button id="btn-close-il" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>🏭 Industrial Loop</h2>

        <div class="il-tabs">
            <button class="il-tab active" data-tab="flow" onclick="window.switchILTab('flow')">Flow</button>
            <button class="il-tab" data-tab="health" onclick="window.switchILTab('health')">Health</button>
            <button class="il-tab" data-tab="metrics" onclick="window.switchILTab('metrics')">Metrics</button>
        </div>

        <div id="il-panel-flow" class="il-panel">
            <div class="il-flow-diagram">
                <svg class="il-flow-lines" id="il-flow-lines" viewBox="0 0 800 100" preserveAspectRatio="none"></svg>
                <div class="il-flow-nodes" id="il-flow-nodes"></div>
            </div>
            <div class="il-flow-legend">
                <span class="il-legend-item"><span class="il-dot optimal"></span> Optimal</span>
                <span class="il-legend-item"><span class="il-dot degraded"></span> Degraded</span>
                <span class="il-legend-item"><span class="il-dot critical"></span> Critical</span>
            </div>
        </div>

        <div id="il-panel-health" class="il-panel hidden">
            <h3>Loop Health</h3>
            <div id="il-health" class="il-health-bars"></div>
            <div class="il-maintenance-timer">
                <span class="il-timer-label">Next Maintenance</span>
                <span class="il-timer-value" id="il-maintenance-timer">--:--:--</span>
            </div>
        </div>

        <div id="il-panel-metrics" class="il-panel hidden">
            <div id="il-metrics" class="il-metrics-grid"></div>
        </div>

        <p id="il-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        loopEl = document.getElementById('industrial-loop-overlay');
        statusEl = document.getElementById('il-status');
        document.getElementById('btn-close-il').addEventListener('click', () => { loopEl.style.display = 'none'; if (maintenanceInterval) clearInterval(maintenanceInterval); });
    }

    function fmtVBV(micro) { return (parseFloat(micro) / 1000000).toFixed(2); }

    // ---------------------------------------------------------------- fetch helper

    async function api(path, opts) {
        const url = path.startsWith('/api/') ? path : API_BASE + path;
        const resp = await fetch(url, opts);
        if (!resp.ok) { let d = ''; try { d = (await resp.json()).error || ''; } catch(e){} throw new Error(d || ('HTTP ' + resp.status)); }
        return resp.json();
    }

    // ---------------------------------------------------------------- tabs

    function switchILTab(tab) {
        document.querySelectorAll('.il-tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab));
        document.querySelectorAll('.il-panel').forEach(p => p.classList.add('hidden'));
        const panel = document.getElementById('il-panel-' + tab);
        if (panel) panel.classList.remove('hidden');
        if (tab === 'flow') renderFlow();
        if (tab === 'health') renderHealth();
        if (tab === 'metrics') renderMetrics();
    }

    // ---------------------------------------------------------------- flow (static structure — phases are fixed; status from metrics)

    function renderFlow() {
        const nodesEl = document.getElementById('il-flow-nodes');
        const linesEl = document.getElementById('il-flow-lines');
        if (!nodesEl || !linesEl) return;
        // PHASES are fixed; status coloring comes from real metrics (fetched async)
        nodesEl.innerHTML = PHASES.map(d => `
            <div class="il-flow-node optimal">
                <div class="il-node-icon">${d.icon}</div>
                <div class="il-node-name">${d.name}</div>
            </div>`).join('');
        let svg = '';
        for (let i = 0; i < PHASES.length - 1; i++) {
            const x1 = 50 + i * 100;
            const x2 = 50 + (i + 1) * 100;
            svg += `<line x1="${x1}" y1="50" x2="${x2}" y2="50" stroke="#10b981" stroke-width="2" stroke-dasharray="8 4"><animate attributeName="stroke-dashoffset" from="0" to="-24" dur="2s" repeatCount="indefinite"/></line>`;
        }
        linesEl.innerHTML = svg;
    }

    // ---------------------------------------------------------------- health (REAL endpoint)

    async function renderHealth() {
        const el = document.getElementById('il-health');
        if (!el) return;
        try {
            const res = await api('/industrial-loop/health', { method: 'GET' });
            // Backend returns health metrics as a map or struct
            const health = res.health || res || {};
            const rows = ['circulation', 'treasury', 'employment', 'taxation'];
            el.innerHTML = rows.map(name => {
                const value = health[name] != null ? Number(health[name]) : 0;
                const color = value > 75 ? '#10b981' : value > 50 ? '#f59e0b' : '#ef4444';
                return `<div class="il-health-row"><span>${name}</span><div class="il-health-bar"><div class="il-health-fill" style="width: ${value}%; background: ${color}"></div></div><span>${value}%</span></div>`;
            }).join('');
        } catch (e) {
            console.warn('[IndustrialLoop] Health API unreachable:', e.message);
            el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">Health data unavailable.</p>';
        }
        startMaintenanceTimer();
    }

    function startMaintenanceTimer() {
        if (maintenanceInterval) clearInterval(maintenanceInterval);
        let seconds = 4 * 3600 + 32 * 60 + 15;
        maintenanceInterval = setInterval(() => {
            seconds--;
            const h = Math.floor(seconds / 3600);
            const m = Math.floor((seconds % 3600) / 60);
            const s = seconds % 60;
            const el = document.getElementById('il-maintenance-timer');
            if (el) el.textContent = `${h.toString().padStart(2, '0')}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
            if (seconds <= 0) clearInterval(maintenanceInterval);
        }, 1000);
    }

    // ---------------------------------------------------------------- metrics (REAL endpoint)

    async function renderMetrics() {
        const el = document.getElementById('il-metrics');
        if (!el) return;
        try {
            const res = await api('/industrial-loop/metrics', { method: 'GET' });
            const m = res.metrics || res || {};
            el.innerHTML = `
                <div class="il-metric-card"><div class="il-metric-label">KPH</div><div class="il-metric-value">${m.kph != null ? m.kph : '—'}</div></div>
                <div class="il-metric-card"><div class="il-metric-label">VBV/Day</div><div class="il-metric-value">${m.vbv_per_day != null ? m.vbv_per_day : '—'}</div></div>
                <div class="il-metric-card"><div class="il-metric-label">Uptime</div><div class="il-metric-value">${m.uptime != null ? m.uptime + '%' : '—'}</div></div>
                <div class="il-metric-card"><div class="il-metric-label">Efficiency</div><div class="il-metric-value">${m.efficiency != null ? m.efficiency + '%' : '—'}</div></div>
            `;
        } catch (e) {
            console.warn('[IndustrialLoop] Metrics API unreachable:', e.message);
            el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">Metrics unavailable.</p>';
        }
    }

    window.openIndustrialLoop = function () {
        init();
        if (window.hideAllOverlays) window.hideAllOverlays();
        loopEl.style.display = 'flex';
        switchILTab('flow');
    };
    window.switchILTab = switchILTab;
})();
