// AI Citizens frontend module (§15 v3) — neon-glass panel for spawning, adopting,
// challenging, and tracking AI citizens + region capacity. Mirrors creator_store.js conventions.
(function () {
    'use strict';

    // State
    let state = {
        citizens: [],
        freeAgents: [],
        regionCounts: {},   // region -> count (for the meter)
        currentWallet: ''
    };

    // DOM references (lazy)
    let overlayEl = null;
    let citizensGridEl = null;
    let freeAgentsGridEl = null;
    let regionMeterEl = null;
    let statusTextEl = null;
    let spawnNameEl = null;
    let spawnCareerEl = null;
    let spawnPathwayEl = null;
    let spawnRegionEl = null;

    // §15 v3 — 12 pathways (mirrors ai_citizen_engine.go consts)
    const PATHWAYS = [
        'P-Shadow', 'P-Lockdown', 'P-Ledger', 'P-Syndicate',
        'P-Tax', 'P-Peace', 'P-Intel', 'P-Justice',
        'P-AOS', 'P-Commissioner', 'P-Forensic', 'P-Boss'
    ];
    const CAREERS = [
        'Underworld', 'Lockdown', 'Ledger', 'Syndicate',
        'Tax', 'Peace', 'Intel', 'Justice',
        'AOS', 'Commissioner', 'Forensic', 'Boss'
    ];

    var API_BASE = '/api';

    function getWallet() {
        return window.currentWallet || (window.getActiveWallet && window.getActiveWallet()) || '';
    }

    function init() {
        if (state._initialized) return;
        state._initialized = true;
        buildOverlay();
        wireEvents();
    }

    function buildOverlay() {
        const html = `
<div id="ai-citizens-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel ai-citizens-panel">
        <button id="btn-close-ai-citizens" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>🤖 AI Citizens (§15)</h2>

        <div class="ai-tabs">
            <button class="ai-tab active" data-tab="spawn" onclick="window.switchAITab('spawn')">Spawn / Adopt</button>
            <button class="ai-tab" data-tab="citizens" onclick="window.switchAITab('citizens')">My Citizens</button>
            <button class="ai-tab" data-tab="agents" onclick="window.switchAITab('agents')">Free Agents</button>
            <button class="ai-tab" data-tab="stats" onclick="window.switchAITab('stats')">Stats</button>
            <button class="ai-tab" data-tab="business" onclick="window.switchAITab('business')">Business</button>
            <button class="ai-tab" data-tab="pathways" onclick="window.switchAITab('pathways')">Pathways</button>
        </div>

        <div id="ai-panel-spawn" class="ai-panel">
            <h3>Spawn / Adopt a Citizen</h3>
            <p style="color:#b0bec5;font-size:12px;">Custom asset name registers against your developer hub lease.</p>
            <label>Name (custom asset)
                <input id="ai-spawn-name" type="text" maxlength="40" placeholder="e.g. Shadow_01" />
            </label>
            <label>Career
                <select id="ai-spawn-career">
                    ${CAREERS.map(c => `<option value="${c}">${c}</option>`).join('')}
                </select>
            </label>
            <label>Pathway
                <select id="ai-spawn-pathway">
                    <option value="">(auto from career)</option>
                    ${PATHWAYS.map(p => `<option value="${p}">${p}</option>`).join('')}
                </select>
            </label>
            <label>Region (blank = Base, or number)
                <input id="ai-spawn-region" type="text" maxlength="8" placeholder="Base" />
            </label>
            <button id="ai-spawn-btn" class="vbt-btn vbt-btn-primary">Spawn Citizen</button>
        </div>

        <div id="ai-panel-citizens" class="ai-panel hidden">
            <h3>My Citizens</h3>
            <div id="ai-citizens-grid" class="ai-grid"></div>
        </div>

        <div id="ai-panel-agents" class="ai-panel hidden">
            <h3>Free Agents</h3>
            <div id="ai-free-agents-grid" class="ai-grid"></div>
        </div>

        <div id="ai-panel-stats" class="ai-panel hidden">
            <h3>AI Stats</h3>
            <div id="ai-stats" class="ai-stats-grid"></div>
        </div>
        <div id="ai-panel-business" class="ai-panel hidden">
            <h3>Business</h3>
            <button id="ai-spawn-business-btn" class="vbt-btn vbt-btn-primary">Spawn Business</button>
            <p style="font-size:12px;color:#b0bec5;">Spawn an NPC business for a citizen.</p>
        </div>
        <div id="ai-panel-pathways" class="ai-panel hidden">
            <h3>Pathways (12)</h3>
            <div class="ai-pathway-grid">${PATHWAYS.map(p => `<div class="ai-pathway-chip">${p}</div>`).join('')}</div>
        </div>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);

        overlayEl = document.getElementById('ai-citizens-overlay');
        citizensGridEl = document.getElementById('ai-citizens-grid');
        freeAgentsGridEl = document.getElementById('ai-free-agents-grid');
        regionMeterEl = document.getElementById('ai-region-meter');
        statusTextEl = document.getElementById('ai-citizens-status');
        spawnNameEl = document.getElementById('ai-spawn-name');
        spawnCareerEl = document.getElementById('ai-spawn-career');
        spawnPathwayEl = document.getElementById('ai-spawn-pathway');
        spawnRegionEl = document.getElementById('ai-spawn-region');
    }

    function wireEvents() {
        document.getElementById('btn-close-ai-citizens').addEventListener('click', () => {
            overlayEl.style.display = 'none';
        });
        document.getElementById('ai-spawn-btn').addEventListener('click', onSpawn);
    }

    function setStatus(msg) {
        if (statusTextEl) statusTextEl.textContent = msg || '';
    }

    async function api(path, opts) {
        const url = path.startsWith('/api/') ? path : API_BASE + path;
        const resp = await fetch(url, opts);
        if (!resp.ok) {
            let detail = '';
            try { const j = await resp.json(); detail = j.error || ''; } catch (e) {}
            throw new Error((detail || ('HTTP ' + resp.status)));
        }
        return resp.json();
    }

    async function onSpawn() {
        const wallet = getWallet();
        if (!wallet) { setStatus('Connect your wallet first.'); return; }
        const body = {
            name: spawnNameEl.value.trim(),
            career: spawnCareerEl.value,
            pathway: spawnPathwayEl.value,
            region: spawnRegionEl.value.trim()
        };
        try {
            setStatus('Spawning…');
            const data = await api('/ai/citizens/spawn', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(body)
            });
            setStatus('Spawned: ' + (data.data && data.data?.name) + ' (region ' + ((data.data && data.data.region) || 'Base') + ')');
            await refresh();
        } catch (err) {
            setStatus('Spawn failed: ' + err.message);
        }
    }

    async function adopt(wallet) {
        try {
            await api('/ai/citizens/adopt', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ wallet: getWallet(), citizen_wallet: wallet, tier: 1 })
            });
            setStatus('Adopted citizen.');
            await refresh();
        } catch (err) { setStatus('Adopt failed: ' + err.message); }
    }

    async function release(wallet) {
        try {
            await api('/api/ai/citizens/release', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ wallet: getWallet(), citizen_wallet: wallet })
            });
            setStatus('Released citizen (now free agent).');
            await refresh();
        } catch (err) { setStatus('Release failed: ' + err.message); }
    }

    async function challenge(wallet) {
        const bondStr = window.prompt('Challenge bond (micro-VBV):', '1000000');
        if (bondStr === null) return;
        const bond = parseInt(bondStr, 10);
        if (!bond || bond <= 0) { setStatus('Invalid bond.'); return; }
        try {
            await api('/api/ai/citizens/challenge', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ citizen_wallet: wallet, challenger_wallet: getWallet(), bond_micro: bond })
            });
            setStatus('Challenge bond staked. Settle by battling the owner.');
            await refresh();
        } catch (err) { setStatus('Challenge failed: ' + err.message); }
    }

    async function refresh() {
        try {
            const data = await api('/api/ai/citizens/list');
            state.citizens = (data.data && data.data.citizens) || [];
            state.regionCounts = (data.data && data.data.region_counts) || {};
            renderCitizens();
            renderRegionMeter();
        } catch (err) { setStatus('List failed: ' + err.message); }

        try {
            const fa = await api('/api/ai/citizens/free-agents');
            state.freeAgents = (fa.data) || [];
            renderFreeAgents();
        } catch (err) { /* non-fatal */ }
    }

    function escapeHtml(s) {
        return String(s || '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
    }

    function regionCap(region) {
        if (!region || region === 'Base' || region === '') return 1;
        const n = parseInt(region, 10);
        return isNaN(n) ? 1 : 1 + n;
    }

    function renderRegionMeter() {
        if (!regionMeterEl) return;
        const regions = Object.keys(state.regionCounts);
        if (regions.length === 0) {
            regionMeterEl.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No regions populated yet.</p>';
            return;
        }
        let html = '';
        for (const r of regions) {
            const count = state.regionCounts[r] || 0;
            const cap = regionCap(r);
            const pct = Math.min(100, Math.round((count / cap) * 100));
            html += `<div class="region-row">
                <span class="region-name">${escapeHtml(r)}</span>
                <div class="region-bar"><div class="region-fill" style="width:${pct}%;"></div></div>
                <span class="region-count">${count}/${cap}</span>
            </div>`;
        }
        regionMeterEl.innerHTML = html;
    }

    function renderCitizens() {
        if (!citizensGridEl) return;
        if (state.citizens.length === 0) {
            citizensGridEl.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No citizens yet. Spawn one above.</p>';
            return;
        }
        let html = '';
        for (const c of state.citizens) {
            const region = c.region || 'Base';
            html += `<div class="ai-citizen-card">
                <h4>${escapeHtml(c?.name)}</h4>
                <p style="font-size:12px;color:#b0bec5;">Career: ${escapeHtml(c.career)} · Pathway: ${escapeHtml(c.pathway || '-')}</p>
                <p style="font-size:12px;color:#4dd0e1;">Region: ${escapeHtml(region)} · Status: ${escapeHtml(c.status)} · Rep: ${c.reputation != null ? c.reputation : 0}</p>
                <div class="ai-card-actions">
                    <button class="vbt-btn vbt-btn-secondary" onclick="window._aiRelease('${escapeHtml(c.wallet)}')">Release</button>
                    <button class="vbt-btn vbt-btn-danger" onclick="window._aiChallenge('${escapeHtml(c.wallet)}')">Challenge</button>
                    <button class="vbt-btn vbt-btn-secondary" onclick="window._aiView3D('${escapeHtml(region)}')">🌐 View in 3D</button>
                </div>
            </div>`;
        }
        citizensGridEl.innerHTML = html;
    }

    function renderFreeAgents() {
        if (!freeAgentsGridEl) return;
        if (state.freeAgents.length === 0) {
            freeAgentsGridEl.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No free agents available.</p>';
            return;
        }
        let html = '';
        for (const c of state.freeAgents) {
            html += `<div class="ai-citizen-card">
                <h4>${escapeHtml(c?.name)}</h4>
                <p style="font-size:12px;color:#b0bec5;">Career: ${escapeHtml(c.career)} · Pathway: ${escapeHtml(c.pathway || '-')}</p>
                <p style="font-size:12px;color:#4dd0e1;">Region: ${escapeHtml(c.region || 'Base')}</p>
                <button class="vbt-btn vbt-btn-primary" onclick="window._aiAdopt('${escapeHtml(c.wallet)}')">Adopt</button>
            </div>`;
        }
        freeAgentsGridEl.innerHTML = html;
    }

    function switchAITab(tab) {
        document.querySelectorAll('.ai-tab').forEach(t => t.classList.toggle('active', t.dataset.tab === tab));
        document.querySelectorAll('.ai-panel').forEach(p => p.classList.add('hidden'));
        const panel = document.getElementById('ai-panel-' + tab);
        if (panel) panel.classList.remove('hidden');
        if (tab === 'stats') loadAIStats();
        if (tab === 'business') wireBusiness();
    }

    // ── AI Stats (REAL endpoint: /api/ai/citizens/stats) ──────────────────────
    async function loadAIStats() {
        const el = document.getElementById('ai-stats');
        if (!el) return;
        try {
            const res = await api('/ai/citizens/stats');
            const stats = res.data || {};
            el.innerHTML = `
                <div class="ai-stat-card"><label>Total Citizens</label><span>${stats.total_citizens ?? '—'}</span></div>
                <div class="ai-stat-card"><label>Free Agents</label><span>${stats.free_agents ?? '—'}</span></div>
                <div class="ai-stat-card"><label>Employed</label><span>${stats.employed ?? '—'}</span></div>
                <div class="ai-stat-card"><label>Businesses</label><span>${stats.business_count ?? '—'}</span></div>`;
        } catch (e) {
            console.warn('[AICitizens] Stats API unreachable:', e.message);
            el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">Stats unavailable.</p>';
        }
    }

    // ── Business Spawn (REAL endpoint: /api/ai/citizens/business/spawn) ────────
    function wireBusiness() {
        const btn = document.getElementById('ai-spawn-business-btn');
        if (!btn) return;
        btn.onclick = async () => {
            const wallet = getWallet();
            if (!wallet) { setStatus('Connect wallet first.'); return; }
            try {
                await api('/ai/citizens/business/spawn', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ wallet }),
                });
                setStatus('Business spawned!');
            } catch (e) {
                setStatus('Business spawn failed: ' + e.message);
            }
        };
    }

    // Public entry (wire to a nav button / HUD)
    window.openAICitizens = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        state.currentWallet = getWallet();
        refresh();
    };
    window.switchAITab = switchAITab;
    window._aiAdopt = adopt;
    window._aiRelease = release;
    window._aiChallenge = challenge;
    // 3D-world integration (§25 / §30): warp the explorer to the citizen's region capital.
    window._aiView3D = function (region) {
        if (typeof window.enter3DWorld === 'function') window.enter3DWorld();
        // give the 3D engine a tick to mount + fetch regions, then warp
        setTimeout(function () {
            if (window.World3DEngine && typeof window.World3DEngine.warpToRegion === 'function') {
                window.World3DEngine.warpToRegion(region);
            }
        }, 350);
    };
})();
