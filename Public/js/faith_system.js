// ============================================================================
// faith_system.js — Faith System + Religious Leader Card
// ----------------------------------------------------------------------------
// Faith = a CLUB-LIKE social structure that envelops world activity.
// Religious battles = gambit on your FAVOURITE CARD.
// Religious Leader Card = cross-mode mechanic (carries into career/tournaments).
// Rival-faith buff (+40) / same-faith weaken (-30).
// ============================================================================
(function () {
    'use strict';

    var API_BASE = '/api';

    let overlayEl = null;
    let faithsEl = null;
    let leaderCardsEl = null;
    let statusEl = null;

    function init() {
        if (overlayEl) return;
        const html = `
<div id="faith-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel faith-panel">
        <button id="btn-close-faith" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>⛪ Faith System</h2>

        <div class="faith-tabs">
            <button class="faith-tab active" data-tab="coherence" onclick="window.switchFaithTab('coherence')">Coherence</button>
            <button class="faith-tab" data-tab="leader-cards" onclick="window.switchFaithTab('leader-cards')">Leader Cards</button>
            <button class="faith-tab" data-tab="war" onclick="window.switchFaithTab('war')">Faith War</button>
            <button class="faith-tab" data-tab="market" onclick="window.switchFaithTab('market')">Market</button>
            <button class="faith-tab" data-tab="rituals" onclick="window.switchFaithTab('rituals')">Rituals</button>
            <button class="faith-tab" data-tab="members" onclick="window.switchFaithTab('members')">Members</button>
        </div>

        <div id="faith-panel-coherence" class="faith-panel">
            <h3>Regional Faith Coherence</h3>
            <label>Region <input id="faith-region" type="text" placeholder="Base" /></label>
            <button id="faith-check-btn" class="vbt-btn vbt-btn-secondary">Check Coherence</button>
            <div id="faith-coherence-result"></div>
        </div>

        <div id="faith-panel-leader-cards" class="faith-panel hidden">
            <h3>Religious Leader Cards</h3>
            <p style="font-size:11px;color:#90a4ae;">Cards that adopt a religious position. Rival-faith +40 blessing / same-faith −30 orthodoxy purge.</p>
            <div id="faith-leader-cards"></div>
        </div>

        <div id="faith-panel-war" class="faith-panel hidden">
            <h3>Faith War Gambit</h3>
            <p style="font-size:11px;color:#90a4ae;">Stake your favourite card to the faith-war pot. Win → card returns + winnings. Lose → card jailed to winning faith kitty.</p>
            <button id="faith-war-btn" class="vbt-btn vbt-btn-primary">⚔️ Enter Faith War</button>
        </div>

        <div id="faith-panel-market" class="faith-panel hidden">
            <h3>Entity Market — Religious Powers</h3>
            <p style="font-size:11px;color:#90a4ae;">Market shows religious powers but does NOT trade them. Faith is non-transferable.</p>
            <button class="vbt-btn vbt-btn-secondary" onclick="window.openEntityMarket && window.openEntityMarket()">🏪 Visit Market</button>
        </div>

        <div id="faith-panel-rituals" class="faith-panel hidden">
            <h3>Rituals</h3>
            <div id="faith-rituals"></div>
        </div>

        <div id="faith-panel-members" class="faith-panel hidden">
            <h3>Members</h3>
            <div id="faith-members"></div>
        </div>

        <p id="faith-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('faith-overlay');
        faithsEl = document.getElementById('faith-coherence-result');
        leaderCardsEl = document.getElementById('faith-leader-cards');
        statusEl = document.getElementById('faith-status');

        document.getElementById('btn-close-faith').addEventListener('click', () => { overlayEl.style.display = 'none'; });
        document.getElementById('faith-check-btn').addEventListener('click', onCheckCoherence);
        document.getElementById('faith-war-btn').addEventListener('click', onFaithWar);
    }

    function switchFaithTab(tab) {
        document.querySelectorAll('.faith-tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab));
        document.querySelectorAll('.faith-panel').forEach(p => p.classList.add('hidden'));
        const panel = document.getElementById('faith-panel-' + tab);
        if (panel) panel.classList.remove('hidden');
        if (tab === 'rituals') loadRituals();
        if (tab === 'members') loadMembers();
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>\"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }

    async function api(path, opts) {
        const url = path.startsWith('/api/') ? path : API_BASE + path;
        const resp = await fetch(url, opts);
        if (!resp.ok) { let d = ''; try { d = (await resp.json()).error || ''; } catch(e){} throw new Error(d || ('HTTP ' + resp.status)); }
        return resp.json();
    }

    async function onCheckCoherence() {
        const region = document.getElementById('faith-region')?.value.trim() || 'Base';
        try {
            const res = await api(`/api/faith/coherence?region=${encodeURIComponent(region)}`);
            if (faithsEl) {
                faithsEl.innerHTML = `<div class="faith-coherence-result">
                    <span class="faith-region-name">${esc(region)}</span>
                    <span class="faith-coherence-score">FaithCoherence: ${res.faith_coherence || 0}</span>
                    <span class="faith-score">Score: ${res.score || 0}</span>
                </div>`;
            }
        } catch (e) { setStatus('Check failed: ' + e.message); }
    }

    async function onFaithWar() {
        try {
            const res = await api('/api/faith/war-gambit', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({}) });
            if (res.success) {
                setStatus(`⚔️ Faith war entered! Staked card: ${res.staked_card_id}. ${res.note}`);
            } else {
                setStatus(`❌ ${res.error}`);
            }
        } catch (e) { setStatus('Faith war failed: ' + e.message); }
    }

    function setStatus(m) { if (statusEl) statusEl.textContent = m || ''; }

    async function loadRituals() {
        const el = document.getElementById('faith-rituals');
        if (!el) return;
        try {
            const res = await api('/api/church/rituals');
            if (res.success !== false && res.rituals) {
                el.innerHTML = res.rituals.map(r => `
                    <div class="wd-card">
                        <span>${esc(r.name)}</span>
                        <span>${esc(r.status)}</span>
                    </div>`).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No rituals available</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Rituals unavailable</p>'; }
    }

    async function loadMembers() {
        const el = document.getElementById('faith-members');
        if (!el) return;
        try {
            const res = await api('/api/church/members');
            if (res.success !== false && res.members) {
                el.innerHTML = res.members.map(m => `
                    <div class="wd-card">
                        <span>${esc(m.name || m.wallet)}</span>
                        <span>${esc(m.role || 'member')}</span>
                    </div>`).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No members</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Members unavailable</p>'; }
    }

    // The tab bar this module renders names `switchFaithTab`, and an inline handler resolves that
    // name on `window` — a function inside this IIFE is not a global, so the tabs were inert.
    window.switchFaithTab = switchFaithTab;

    window.openFaithSystem = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
    };
})();
