// ============================================================================
// religion_governance.js — Religion Governance Storefront
// ----------------------------------------------------------------------------
// Faucet-owned chain store. 24-cap server-wide. All buyouts → faucet.
// Users buy governorship. Once full, only buyouts work.
// Buyout = total loss. Converted users can ONLY buy out another to return.
// On buyout, user chooses: restore old OR create new.
// ============================================================================
(function () {
    'use strict';

    var API_BASE = '/api';
    const MICRO = 1000000;

    let overlayEl = null;
    let religionsEl = null;
    let highTierEl = null;
    let statusEl = null;
    let buyoutModalEl = null;

    let currentReligions = [];
    let selectedReligion = null;
    // null = the /api/faith/high-tier read was refused (a state the UI states, rather than
    // rendering as "no rivals"). [] = the engine reported none.
    let highTierRivals = null;

    function init() {
        if (overlayEl) return;
        const html = `
<div id="religion-gov-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel religion-gov-panel">
        <button id="btn-close-rg" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>⛪ Religion Governance</h2>
        <p style="font-size:12px;color:#b0bec5;">Faucet-owned chain store. 24-cap. All buyouts → faucet. Total loss for old owner.</p>

        <div class="rg-stats">
            <span class="rg-stat">📦 Owned: <b id="rg-owned">0</b> / 24</span>
            <!-- The 12 here was HARD-CODED. The value is now the engine's own count, read from
                 /api/faith/high-tier (ReligionGovernance.GetHighTierRivals → the top 12 religions
                 BY POWER). It reads "—" until the server answers, because a number the client
                 invented is not a fact. -->
            <span class="rg-stat">⚡ High-Tier Rivals: <b id="rg-high-tier">—</b></span>
            <span id="rg-converted-badge" class="rg-converted-badge" style="display:none;">⚠️ CONVERTED</span>
        </div>

        <div class="rg-tabs">
            <button class="rg-tab active" data-tab="all">All Religions</button>
            <button class="rg-tab" data-tab="high-tier">High-Tier Rivals</button>
            <button class="rg-tab" data-tab="mine">My Religion</button>
        </div>

        <div id="rg-religions" class="rg-religions"></div>

        <div id="rg-buyout-modal" class="rg-buyout-modal" style="display:none;">
            <h3>Buyout Religion</h3>
            <p style="font-size:11px;color:#90a4ae;">All members auto-convert. Old governor gets NOTHING.</p>
            <label>New Name (leave blank to keep old)</label>
            <input id="rg-buyout-name" type="text" placeholder="Optional new name" />
            <label><input type="checkbox" id="rg-buyout-restore" /> Restore old rivalries</label>
            <button id="rg-buyout-confirm" class="vbt-btn vbt-btn-primary">Confirm Buyout</button>
            <button id="rg-buyout-cancel" class="vbt-btn vbt-btn-secondary">Cancel</button>
        </div>

        <p id="rg-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('religion-gov-overlay');
        religionsEl = document.getElementById('rg-religions');
        statusEl = document.getElementById('rg-status');

        document.getElementById('btn-close-rg').addEventListener('click', () => { overlayEl.style.display = 'none'; });

        document.querySelectorAll('.rg-tab').forEach(btn => {
            btn.addEventListener('click', () => {
                
                
                renderTab(btn.dataset.tab);
            });
        });

        document.getElementById('rg-buyout-confirm').addEventListener('click', onBuyoutConfirm);
        document.getElementById('rg-buyout-cancel').addEventListener('click', () => {
            document.getElementById('rg-buyout-modal').style.display = 'none';
        });
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>\"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }
    function fmtVBV(micro) { return (parseFloat(micro) / MICRO).toFixed(4); }

    async function api(path, opts) {
        const url = path.startsWith('/api/') ? path : API_BASE + path;
        const resp = await fetch(url, opts);
        if (!resp.ok) { let d = ''; try { d = (await resp.json()).error || ''; } catch(e){} throw new Error(d || ('HTTP ' + resp.status)); }
        return resp.json();
    }

    async function load() {
        try {
            const res = await api('/api/faith/religions');
            currentReligions = res.religions || [];
            document.getElementById('rg-owned').textContent = res.owned || 0;
            renderTab('all');
            checkConverted();
        } catch (e) { setStatus('Load failed: ' + e.message); }
        // /api/faith/high-tier had NO owner in the client: the "High-Tier Rivals" tab showed
        // `currentReligions.slice(0, 12)` — the first twelve in whatever order the list arrived —
        // and the header printed a hard-coded 12. The engine's own top-by-power set is read here.
        await loadHighTier();
    }

    async function loadHighTier() {
        const badge = document.getElementById('rg-high-tier');
        try {
            const res = await api('/api/faith/high-tier');
            highTierRivals = Array.isArray(res.high_tier) ? res.high_tier : [];
            if (badge) badge.textContent = String(highTierRivals.length);
        } catch (e) {
            highTierRivals = null;
            if (badge) badge.textContent = '—';
            setStatus('High-tier rivals could not be read: ' + e.message);
        }
    }

    async function checkConverted() {
        try {
            const res = await api('/api/faith/converted');
            const badge = document.getElementById('rg-converted-badge');
            if (badge) badge.style.display = res.converted ? 'inline' : 'none';
        } catch(e) {}
    }

    function switchRgTab(tab) {
        if (!religionsEl) return;
        let rels = currentReligions;
        if (tab === 'high-tier') {
            // The engine's top-by-power set, not "the first twelve of whatever we happen to hold".
            // `null` means the read was refused, which is STATED rather than shown as an empty list.
            if (highTierRivals === null) {
                religionsEl.innerHTML = '<p style="color:#90a4ae;font-size:12px;">High-tier rivals could not be read from the engine.</p>';
                return;
            }
            renderReligions(highTierRivals);
            return;
        } else if (tab === 'mine') {
            // filter to user's religion
            rels = rels.filter(r => r.governor === window.userAddress || r.members[window.userAddress]);
        }
        renderReligions(rels);
    }

    function renderReligions(rels) {
        if (!religionsEl) return;
        if (!(rels?.length ?? 0)) {
            religionsEl.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No religions found.</p>';
            return;
        }
        religionsEl.innerHTML = rels.map(r => {
            const isOwned = r.governor === window.userAddress;
            const isMember = r.members && r.members[window.userAddress];
            const isFaucet = r.governor === '';
            const dogmaClass = r.dogma || 'custom';
            return `<div class="rg-religion ${isOwned ? 'rg-owned' : ''} ${isFaucet ? 'rg-faucet' : ''}">
                <div class="rg-rel-head">
                    <span class="rg-rel-name">${esc(r?.name)}</span>
                    <span class="rg-rel-dogma dogma-${dogmaClass}">${esc(r.dogma)}</span>
                </div>
                <div class="rg-rel-meta">
                    <span>📍 ${esc(r.region)}</span>
                    <span>👥 ${Object.keys(r.members || {})?.length ?? 0} members</span>
                    <span>🕯️ ${r.ritual_count || 0} rituals</span>
                    <span>⚡ ${r.faith_power || 0} power</span>
                </div>
                <div class="rg-rel-foot">
                    <span class="rg-rel-price">${fmtVBV(r.price_paid || 0)} $VBV</span>
                    <span class="rg-rel-governor">${isFaucet ? '🏛️ FAUCET' : (isOwned ? '👑 YOU' : '👤 ' + esc((r.governor || '').slice(0, 8)))}</span>
                    ${isFaucet ? `<button class="vbt-btn vbt-btn-primary rg-buy" data-id="${r.religion_id}">Buy</button>` : ''}
                    ${!isFaucet && !isOwned ? `<button class="vbt-btn vbt-btn-secondary rg-buyout-btn" data-id="${r.religion_id}">Buyout</button>` : ''}
                    ${isMember && !isOwned ? `<button class="vbt-btn vbt-btn-secondary rg-ritual" data-id="${r.religion_id}">🕯️ Ritual</button>` : ''}
                    ${!isMember && !isFaucet ? `<button class="vbt-btn vbt-btn-secondary rg-join" data-id="${r.religion_id}">Join</button>` : ''}
                </div>
                ${r.rivalries && (Object.keys(r.rivalries)?.length ?? 0) > 0 ? `
                    <div class="rg-rivalries">
                        <span>⚔️ Rivals:</span>
                        ${Object.keys(r.rivalries).map(rid => `<span class="rg-rival-tag">${esc(rid)}</span>`).join('')}
                    </div>
                ` : ''}
                ${isOwned ? `
                    <div class="rg-rivalry-add">
                        <input id="rg-rival-${r.religion_id}" type="text" placeholder="Add rival religion ID" />
                        <button class="vbt-btn vbt-btn-secondary rg-add-rival" data-id="${r.religion_id}">+ Rival</button>
                    </div>
                ` : ''}
            </div>`;
        }).join('');

        // bind events
        religionsEl.querySelectorAll('.rg-buy').forEach(btn => btn.addEventListener('click', () => buyGovernorship(btn.dataset.id)));
        religionsEl.querySelectorAll('.rg-buyout-btn').forEach(btn => btn.addEventListener('click', () => showBuyoutModal(btn.dataset.id)));
        religionsEl.querySelectorAll('.rg-ritual').forEach(btn => btn.addEventListener('click', () => performRitual(btn.dataset.id)));
        religionsEl.querySelectorAll('.rg-join').forEach(btn => btn.addEventListener('click', () => joinReligion(btn.dataset.id)));
        religionsEl.querySelectorAll('.rg-add-rival').forEach(btn => btn.addEventListener('click', () => addRivalry(btn.dataset.id)));
    }

    async function buyGovernorship(religionId) {
        try {
            const res = await api('/api/faith/religion/buy', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ religion_id: religionId }),
            });
            setStatus(`✅ You are now governor of ${res.religion?.name}!`);
            load();
        } catch (e) { setStatus('Buy failed: ' + e.message); }
    }

    function showBuyoutModal(religionId) {
        selectedReligion = currentReligions.find(r => r.religion_id === religionId);
        if (!selectedReligion) return;
        document.getElementById('rg-buyout-name').value = '';
        document.getElementById('rg-buyout-restore').checked = false;
        document.getElementById('rg-buyout-modal').style.display = 'block';
    }

    async function onBuyoutConfirm() {
        if (!selectedReligion) return;
        const newName = document.getElementById('rg-buyout-name')?.value.trim();
        const restoreOld = document.getElementById('rg-buyout-restore').checked;

        try {
            const res = await api('/api/faith/religion/buyout', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    religion_id: selectedReligion.religion_id,
                    new_name: newName,
                    restore_old: restoreOld,
                }),
            });
            setStatus(`✅ Buyout complete! ${res.religion?.name} is yours. Old governor gets NOTHING.`);
            document.getElementById('rg-buyout-modal').style.display = 'none';
            selectedReligion = null;
            load();
        } catch (e) { setStatus('Buyout failed: ' + e.message); }
    }

    async function performRitual(religionId) {
        try {
            await api('/api/faith/religion/ritual', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ religion_id: religionId }),
            });
            setStatus('🕯️ Ritual performed! FaithCoherence raised.');
            load();
        } catch (e) { setStatus('Ritual failed: ' + e.message); }
    }

    async function joinReligion(religionId) {
        try {
            await api('/api/faith/religion/join', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ religion_id: religionId }),
            });
            setStatus('✅ Joined religion!');
            load();
        } catch (e) { setStatus('Join failed: ' + e.message); }
    }

    async function addRivalry(religionId) {
        const input = document.getElementById(`rg-rival-${religionId}`);
        const rivalId = input ? input.value.trim() : '';
        if (!rivalId) { setStatus('Enter rival religion ID.'); return; }
        try {
            await api('/api/faith/religion/rivalry', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ religion_id: religionId, rival_id: rivalId }),
            });
            setStatus(`⚔️ Rivalry added: ${religionId} vs ${rivalId}`);
            load();
        } catch (e) { setStatus('Rivalry failed: ' + e.message); }
    }

    function setStatus(m) { if (statusEl) statusEl.textContent = m || ''; }

    window.openReligionGovernance = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        load();
    };
})();
