// ============================================================================
// underworld.js — Underworld Panel (REAL FUNCTIONALITY)
// ============================================================================
(function () {
    'use strict';
    var API_BASE = '/api';
    let overlayEl = null, statusEl = null;
    let heists = [];
    let bounties = [];
    let rumors = [];
    let contracts = [];

    function init() {
        if (overlayEl) return;
        const html = `
<div id="underworld-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel underworld-panel">
        <button id="btn-close-uw" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>💀 Underworld</h2>

        <div class="uw-tabs">
            <button class="uw-tab active" data-tab="heist" onclick="window.switchUWTab('heist')">Heist</button>
            <button class="uw-tab" data-tab="kidnap" onclick="window.switchUWTab('kidnap')">Kidnap</button>
            <button class="uw-tab" data-tab="bounty" onclick="window.switchUWTab('bounty')">Bounty Board</button>
            <button class="uw-tab" data-tab="rumors" onclick="window.switchUWTab('rumors')">Rumor Mill</button>
            <button class="uw-tab" data-tab="contracts" onclick="window.switchUWTab('contracts')">Contracts</button>
        </div>

        <div id="uw-panel-heist" class="uw-panel">
            <h3>Heist Planning</h3>
            <p style="font-size:12px;color:#b0bec5;">Plan heists against targets. Higher difficulty = higher reward but greater risk.</p>
            <div id="uw-heist" class="uw-heist"></div>
        </div>

        <div id="uw-panel-kidnap" class="uw-panel hidden">
            <h3>Kidnap Operations</h3>
            <p style="font-size:12px;color:#b0bec5;">Kidnap players to hold for ransom. Ransom paid to release.</p>
            <div id="uw-kidnap" class="uw-kidnap"></div>
        </div>

        <div id="uw-panel-bounty" class="uw-panel hidden">
            <h3>Bounty Board</h3>
            <div id="uw-bounty" class="uw-bounty"></div>
        </div>

        <div id="uw-panel-rumors" class="uw-panel hidden">
            <h3>Rumor Mill</h3>
            <p style="font-size:12px;color:#b0bec5;">Spread or trade rumors to manipulate market perception.</p>
            <div id="uw-rumors" class="uw-rumors"></div>
        </div>

        <div id="uw-panel-contracts" class="uw-panel hidden">
            <h3>Underworld Contracts</h3>
            <div id="uw-contracts" class="uw-contracts"></div>
        </div>

        <p id="uw-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('underworld-overlay');
        statusEl = document.getElementById('uw-status');
        document.getElementById('btn-close-uw').addEventListener('click', () => { overlayEl.style.display = 'none'; });
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>\"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }

    // Active wallet for wallet-scoped reads. Contract eligibility is evaluated per
    // wallet, so this panel must send one or the server can only answer "empty".
    // Layered so a mid-boot engine cannot break the panel.
    function activeWallet() {
        if (typeof window.getActiveWallet === 'function') {
            try { const w = window.getActiveWallet(); if (w) return String(w); } catch (_) { /* mid-boot */ }
        }
        return window.currentWallet || window.userAddress || '';
    }

    function walletQS() {
        const w = activeWallet();
        return w ? ('?wallet=' + encodeURIComponent(w)) : '';
    }

    async function api(path, opts) {
        const url = path.startsWith('/api/') ? path : API_BASE + path;
        const resp = await fetch(url, opts);
        if (!resp.ok) { let d = ''; try { d = (await resp.json()).error || ''; } catch(e){} throw new Error(d || ('HTTP ' + resp.status)); }
        return resp.json();
    }

    function switchUWTab(tab) {
        document.querySelectorAll('.uw-tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab));
        document.querySelectorAll('.uw-panel').forEach(p => p.classList.add('hidden'));
        const panel = document.getElementById('uw-panel-' + tab);
        if (panel) panel.classList.remove('hidden');
        if (tab === 'heist') loadHeist();
        if (tab === 'kidnap') loadKidnap();
        if (tab === 'bounty') loadBounty();
        if (tab === 'rumors') loadRumors();
        if (tab === 'contracts') loadContracts();
    }

    async function loadHeist() {
        const el = document.getElementById('uw-heist');
        if (!el) return;
        try {
            const res = await api('/api/underworld/heists');
            heists = res.heists || [];
            renderHeist();
        } catch (e) {
            console.warn('[Underworld] API unreachable:', e.message);
            heists = [];
            renderHeist();
        }
    }

    function renderHeist() {
        const el = document.getElementById('uw-heist');
        if (!el) return;
        if (!heists.length) { el.innerHTML = '<div class="panel-empty"><div class="panel-empty-icon">💀</div><div class="panel-empty-message">No heists available.</div></div>'; return; }
        el.innerHTML = heists.map(h => `
            <div class="uw-heist-card">
                <div class="uw-heist-name">${esc(h.target)}</div>
                <div class="uw-heist-difficulty">Difficulty: ${esc(h.difficulty)}</div>
                <div class="uw-heist-reward">💰 ${h.reward} VBV</div>
                <div class="uw-heist-risk">Risk: ${h.risk}%</div>
                <button class="vbt-btn vbt-btn-primary" onclick="window.uwStartHeist('${h.id}')">Plan Heist</button>
            </div>
        `).join('');
    }

    async function loadKidnap() {
        const el = document.getElementById('uw-kidnap');
        if (!el) return;
        try {
            const res = await api('/api/underworld/kidnaps');
            const kidnaps = res.kidnaps || [];
            renderKidnap(kidnaps);
        } catch (e) {
            console.warn('[Underworld] API unreachable:', e.message);
            renderKidnap([]);
        }
    }

    function renderKidnap(kidnaps) {
        const el = document.getElementById('uw-kidnap');
        if (!el) return;
        if (!kidnaps.length) { el.innerHTML = '<div class="panel-empty"><div class="panel-empty-icon">🔗</div><div class="panel-empty-message">No kidnap targets.</div></div>'; return; }
        el.innerHTML = kidnaps.map(k => `
            <div class="uw-kidnap-card">
                <div class="uw-kidnap-name">${esc(k.target)}</div>
                <div class="uw-kidnap-ransom">💰 ${k.ransom} VBV ransom</div>
                <div class="uw-kidnap-status">${esc(k.status)}</div>
                <button class="vbt-btn vbt-btn-secondary" onclick="window.uwKidnap('${k.id}')">${k.status === 'held' ? 'Held' : 'Kidnap'}</button>
            </div>
        `).join('');
    }

    async function loadBounty() {
        const el = document.getElementById('uw-bounty');
        if (!el) return;
        try {
            const res = await api('/api/justice/bounty-board');
            bounties = res.bounties || res.bounty_board || [];
            renderBounty();
        } catch (e) {
            console.warn('[Underworld] API unreachable:', e.message);
            bounties = [];
            renderBounty();
        }
    }

    function renderBounty() {
        const el = document.getElementById('uw-bounty');
        if (!el) return;
        if (!bounties.length) { el.innerHTML = '<div class="panel-empty"><div class="panel-empty-icon">🎯</div><div class="panel-empty-message">No active bounties.</div></div>'; return; }
        el.innerHTML = bounties.map(b => `
            <div class="uw-bounty-card">
                <div class="uw-bounty-name">${esc(b.target)}</div>
                <div class="uw-bounty-amount">💰 ${b.amount} VBV</div>
                <div class="uw-bounty-level">Wanted Level: ${b.wanted_level || '—'}</div>
                <button class="vbt-btn vbt-btn-primary" onclick="window.uwCapture('${b.id}', ${b.amount})">Capture</button>
            </div>
        `).join('');
    }

    async function loadRumors() {
        const el = document.getElementById('uw-rumors');
        if (!el) return;
        try {
            const res = await api('/api/rumors');
            rumors = res.rumors || [];
            renderRumors();
        } catch (e) {
            console.warn('[Underworld] API unreachable:', e.message);
            rumors = [];
            renderRumors();
        }
    }

    function renderRumors() {
        const el = document.getElementById('uw-rumors');
        if (!el) return;
        if (!rumors.length) { el.innerHTML = '<div class="panel-empty"><div class="panel-empty-icon">📰</div><div class="panel-empty-message">No rumors circulating.</div></div>'; return; }
        el.innerHTML = rumors.map(r => `
            <div class="uw-rumor-card ${r.type}">
                <div class="uw-rumor-text">${esc(r.text)}</div>
                <div class="uw-rumor-strength">Strength: ${r.strength}x | Multiplier: ${r.multiplier}</div>
            </div>
        `).join('');
    }

    async function loadContracts() {
        const el = document.getElementById('uw-contracts');
        if (!el) return;
        if (!activeWallet()) {
            contracts = [];
            el.innerHTML = '<p style="color:#90a4ae;">Connect a wallet to list the contracts it qualifies for.</p>';
            return;
        }
        try {
            const res = await api('/api/contracts/list' + walletQS());
            contracts = res.contracts || [];
            renderContracts();
        } catch (e) {
            console.warn('[Underworld] API unreachable:', e.message);
            contracts = [];
            renderContracts();
        }
    }

    function renderContracts() {
        const el = document.getElementById('uw-contracts');
        if (!el) return;
        if (!contracts.length) { el.innerHTML = '<div class="panel-empty"><div class="panel-empty-icon">📜</div><div class="panel-empty-message">No contracts available.</div></div>'; return; }
        el.innerHTML = contracts.map(c => `
            <div class="uw-contract-card">
                <div class="uw-contract-id">${esc(c.id)}</div>
                <div class="uw-contract-type">Type: ${esc(c.type)}</div>
                <div class="uw-contract-target">Target: ${esc(c.target)}</div>
                <div class="uw-contract-reward">💰 ${c.reward} VBV</div>
                <div class="uw-contract-difficulty">Difficulty: ${esc(c.difficulty)}</div>
                <button class="vbt-btn vbt-btn-primary" onclick="window.uwAcceptContract('${c.id}', ${c.reward})">Accept</button>
            </div>
        `).join('');
    }

    function startHeist(id) {
        if (!window.canAfford(100)) { showToast('Need 100 VBV to plan heist', 'error'); return; }
        if (window.txModalOpen) {
            window.txModalOpen('Plan Heist', 100, () => {
                if (!window.deductVBV(100)) return;
                showToast('Heist planned! Awaiting execution.', 'success');
            });
        }
    }

    function kidnap(id) {
        if (!window.canAfford(200)) { showToast('Need 200 VBV for kidnap', 'error'); return; }
        if (window.txModalOpen) {
            window.txModalOpen('Kidnap Operation', 200, () => {
                if (!window.deductVBV(200)) return;
                showToast('Kidnap initiated!', 'success');
            });
        }
    }

    function capture(id, amount) {
        if (window.txModalOpen) {
            window.txModalOpen('Capture Bounty', 50, () => {
                if (!window.deductVBV(50)) return;
                window.addVBV(amount);
                showToast(`Captured! +${amount} VBV`, 'success');
            });
        }
    }

    function acceptContract(id, reward) {
        if (window.txModalOpen) {
            window.txModalOpen('Accept Contract', 0, () => {
                showToast(`Contract ${id} accepted!`, 'success');
            });
        }
    }

    function showToast(msg, type) { if (window.showToast) window.showToast(msg, type); }

    window.openUnderworld = function () {
        init();
        if (window.hideAllOverlays) window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        switchUWTab('heist');
    };
    window.switchUWTab = switchUWTab;
    window.uwStartHeist = startHeist;
    window.uwKidnap = kidnap;
    window.uwCapture = capture;
    window.uwAcceptContract = acceptContract;
})();
