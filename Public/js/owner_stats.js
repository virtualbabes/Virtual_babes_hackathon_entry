// ============================================================================
// owner_stats.js — Household Combined Stats (REAL API, 2026-09-03)
// ----------------------------------------------------------------------------
// Backend: entity_event_engine.go handleCombineOwnerStats
//   GET /api/owner/combined-stats  → { success, combined_stats: EntityStats, effective_power }
// Wires the orphaned /api/owner/combined-stats endpoint.
// EntityStats: speed, intelligence, willpower, strength, charisma, agility (all uint64).
// ============================================================================

const OwnerStats = (() => {
    let container = null;
    let statsData = null;

    function getWallet() {
        if (window.getWalletAddress) return window.getWalletAddress() || '';
        if (window.getActiveWallet) return window.getActiveWallet() || '';
        return localStorage.getItem('wallet_address') || '';
    }

    async function api(path, opts) {
        const resp = await fetch('/api' + path, opts);
        if (!resp.ok) {
            let detail = '';
            try { const j = await resp.json(); detail = j.error || j.message || ''; } catch (e) {}
            throw new Error(detail || ('HTTP ' + resp.status));
        }
        return resp.json();
    }

    async function loadOwnerStats() {
        // EMPTY STATE default — never mock data.
        statsData = {
            combined_stats: { speed: 0, intelligence: 0, willpower: 0, strength: 0, charisma: 0, agility: 0 },
            effective_power: 0,
        };
        try {
            const res = await api('/owner/combined-stats', { method: 'GET' });
            if (res && res.success) {
                statsData = {
                    combined_stats: res.combined_stats || statsData.combined_stats,
                    effective_power: res.effective_power || 0,
                };
            }
        } catch (e) {
            console.warn('[OwnerStats] API unreachable, showing empty state:', e.message);
        }
    }

    function renderStats() {
        if (!container) return;
        const s = statsData.combined_stats || {};
        const map = {
            'os-speed': s.speed || 0,
            'os-intelligence': s.intelligence || 0,
            'os-willpower': s.willpower || 0,
            'os-strength': s.strength || 0,
            'os-charisma': s.charisma || 0,
            'os-agility': s.agility || 0,
            'os-power': statsData.effective_power || 0,
        };
        for (const [id, val] of Object.entries(map)) {
            const el = document.getElementById(id);
            if (el) el.textContent = val;
        }
    }

    function ownerStatsHtml() {
        return `
      <div class="vbt-overlay owner-stats-dashboard" id="owner-stats-dashboard">
        <div class="overlay-header">
          <h2>👤 Household Stats</h2>
          <button class="close-btn" onclick="OwnerStats.close()">✕</button>
        </div>
        <div class="overlay-body">
          <p class="hint">Combined stats from your household (you + pets + bots). Server-authoritative.</p>
          <div class="stat-grid">
            <div class="stat-card"><label>Speed</label><span id="os-speed">—</span></div>
            <div class="stat-card"><label>Intelligence</label><span id="os-intelligence">—</span></div>
            <div class="stat-card"><label>Willpower</label><span id="os-willpower">—</span></div>
            <div class="stat-card"><label>Strength</label><span id="os-strength">—</span></div>
            <div class="stat-card"><label>Charisma</label><span id="os-charisma">—</span></div>
            <div class="stat-card"><label>Agility</label><span id="os-agility">—</span></div>
          </div>
          <div class="power-card"><label>Effective Power</label><span id="os-power">—</span></div>
        </div>
      </div>`;
    }

    async function refresh() {
        await loadOwnerStats();
        renderStats();
    }

    function open() {
        const existing = document.getElementById('owner-stats-dashboard');
        if (existing) existing.remove();

        const div = document.createElement('div');
        div.innerHTML = ownerStatsHtml();
        document.body.appendChild(div.firstElementChild);
        container = document.getElementById('owner-stats-dashboard');
        if (container) container.classList.add('vbt-overlay');

        refresh();
        return container;
    }

    function close() {
        const el = document.getElementById('owner-stats-dashboard');
        if (el) el.remove();
        container = null;
    }

    return { open, close, refresh };
})();

window.OwnerStats = OwnerStats;
window.openOwnerStats = OwnerStats.open;
