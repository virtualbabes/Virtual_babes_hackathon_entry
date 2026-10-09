// ============================================================================
// community_dashboard.js — Church, Justice, Governance, Replay, Identity, Pet-battle
// ----------------------------------------------------------------------------
// Wires 22 orphaned backend routes:
//   Church:     /api/church/add-item, /api/church/add-member, /api/church/remove-member,
//                /api/church/items, /api/church/region, /api/church/ritual, /api/church/rituals
//   Justice:    /api/justice/award-card, /api/justice/use-rep-shield
//   Governance: /api/governance/close, /api/governance/register, /api/governance/governor
//   Replay:     /api/replay/capture, /api/replay/frames, /api/replay/player,
//                /api/replay/start, /api/replay/state, /api/replay/stop
//   Identity:   /api/identity/link, /api/identity/unlink, /api/identity/resolve,
//                /api/identity/snapshot, /api/identity/record
//   Pet-battle: /api/pet-battle/challenge, /api/pet-battle/resolve
// ============================================================================

(function () {
    'use strict';
    var API_BASE = '/api';
    let overlayEl = null;

    function init() {
        if (overlayEl) return;
        overlayEl = document.getElementById('community-dashboard-overlay');
        if (!overlayEl) {
            // SELF-MOUNT. This module owns a full-screen surface, but nothing in the shell ever
            // provided its root: no `id="community-dashboard-overlay"` exists in index.html. The
            // null-check below therefore turned the WHOLE module into a silent no-op — `init()`
            // returned, the boot-time guard never fired, `openCommunityDashboard()` dereferenced
            // null, and all 25 handler names this file publishes were reachable but had no surface
            // to render into. The root is created here, exactly as the sibling self-mounting
            // modules do, so the surface exists wherever the module is composed.
            document.body.insertAdjacentHTML('beforeend', '<div id="community-dashboard-overlay"></div>');
            overlayEl = document.getElementById('community-dashboard-overlay');
            if (!overlayEl) return;
        }
        overlayEl.className = 'overlay vbt-overlay';
        overlayEl.style.display = 'none';

        overlayEl.innerHTML = `
<div class="neon-glass-panel community-dashboard-panel">
    <button id="btn-close-cd" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
    <h2>🏛️ Community Dashboard</h2>
    <p style="font-size:12px;color:#b0bec5;margin-top:-8px;">Church · Justice · Governance · Replay · Identity · Pet-battle</p>
    <div class="cd-tabs">
        <button class="cd-tab active" data-tab="church" onclick="switchCDTab('church')">Church</button>
        <button class="cd-tab" data-tab="justice" onclick="switchCDTab('justice')">Justice</button>
        <button class="cd-tab" data-tab="governance" onclick="switchCDTab('governance')">Governance</button>
        <button class="cd-tab" data-tab="replay" onclick="switchCDTab('replay')">Replay</button>
        <button class="cd-tab" data-tab="identity" onclick="switchCDTab('identity')">Identity</button>
        <button class="cd-tab" data-tab="petbattle" onclick="switchCDTab('petbattle')">Pet-battle</button>
    </div>
    <div id="cd-panel-church" class="cd-panel"><div id="cd-church"></div></div>
    <div id="cd-panel-justice" class="cd-panel hidden"><div id="cd-justice"></div></div>
    <div id="cd-panel-governance" class="cd-panel hidden"><div id="cd-governance"></div></div>
    <div id="cd-panel-replay" class="cd-panel hidden"><div id="cd-replay"></div></div>
    <div id="cd-panel-identity" class="cd-panel hidden"><div id="cd-identity"></div></div>
    <div id="cd-panel-petbattle" class="cd-panel hidden"><div id="cd-petbattle"></div></div>
    <p id="cd-status" class="ai-status"></p>
</div>`;

        document.getElementById('btn-close-cd').addEventListener('click', () => { overlayEl.style.display = 'none'; });
        loadChurch();
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }

    async function api(path, opts) {
        const url = path.startsWith('/api/') ? path : API_BASE + path;
        const resp = await fetch(url, opts);
        if (!resp.ok) {
            let detail = '';
            try { const j = await resp.json(); detail = j.error || j.message || ''; } catch (e) {}
            throw new Error(detail || ('HTTP ' + resp.status));
        }
        return resp.json();
    }

    function switchCDTab(tab) {
        document.querySelectorAll('.cd-tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab));
        document.querySelectorAll('.cd-panel').forEach(p => p.classList.add('hidden'));
        const panel = document.getElementById('cd-panel-' + tab);
        if (panel) panel.classList.remove('hidden');
        if (tab === 'church') loadChurch();
        if (tab === 'justice') loadJustice();
        if (tab === 'governance') loadGovernance();
        if (tab === 'replay') loadReplay();
        if (tab === 'identity') loadIdentity();
        if (tab === 'petbattle') loadPetBattle();
    }

    // === CHURCH ===
    async function loadChurch() {
        const el = document.getElementById('cd-church');
        if (!el) return;
        el.innerHTML = `
            <div class="cd-grid">
                <button class="cd-btn" onclick="cdChurchAddItem()">Add Item</button>
                <button class="cd-btn" onclick="cdChurchAddMember()">Add Member</button>
                <button class="cd-btn" onclick="cdChurchRemoveMember()">Remove Member</button>
                <button class="cd-btn" onclick="cdChurchItems()">Items</button>
                <button class="cd-btn" onclick="cdChurchRegion()">Region</button>
                <button class="cd-btn" onclick="cdChurchRitual()">Ritual</button>
                <button class="cd-btn" onclick="cdChurchRituals()">Rituals</button>
            </div>
            <div id="cd-church-result" class="cd-result"></div>`;
    }
    async function cdChurchAddItem() {
        const id = prompt('Church ID:');
        const item = prompt('Item ID:');
        if (!id || !item) return;
        try { await api('/church/add-item', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ church_id: id, item_id: item }) }); showChurchResult('Item added!'); }
        catch (e) { showChurchResult('Failed: ' + e.message); }
    }
    async function cdChurchAddMember() {
        const id = prompt('Church ID:');
        const wallet = prompt('Member wallet:');
        if (!id || !wallet) return;
        try { await api('/church/add-member', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ church_id: id, wallet }) }); showChurchResult('Member added!'); }
        catch (e) { showChurchResult('Failed: ' + e.message); }
    }
    async function cdChurchRemoveMember() {
        const id = prompt('Church ID:');
        const wallet = prompt('Member wallet:');
        if (!id || !wallet) return;
        try { await api('/church/remove-member', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ church_id: id, wallet }) }); showChurchResult('Member removed!'); }
        catch (e) { showChurchResult('Failed: ' + e.message); }
    }
    async function cdChurchItems() {
        const id = prompt('Church ID:');
        if (!id) return;
        try { const res = await api('/church/items?church_id=' + encodeURIComponent(id)); showChurchResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showChurchResult('Failed: ' + e.message); }
    }
    async function cdChurchRegion() {
        const id = prompt('Church ID:');
        if (!id) return;
        try { const res = await api('/church/region?church_id=' + encodeURIComponent(id)); showChurchResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showChurchResult('Failed: ' + e.message); }
    }
    async function cdChurchRitual() {
        const id = prompt('Church ID:');
        const ritual = prompt('Ritual type:');
        if (!id || !ritual) return;
        try { await api('/church/ritual', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ church_id: id, ritual_type: ritual }) }); showChurchResult('Ritual performed!'); }
        catch (e) { showChurchResult('Failed: ' + e.message); }
    }
    async function cdChurchRituals() {
        const id = prompt('Church ID:');
        if (!id) return;
        try { const res = await api('/church/rituals?church_id=' + encodeURIComponent(id)); showChurchResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showChurchResult('Failed: ' + e.message); }
    }
    function showChurchResult(msg) {
        const el = document.getElementById('cd-church-result');
        if (el) el.innerHTML = `<pre style="color:#80cbc4;font-size:11px;white-space:pre-wrap;">${esc(msg)}</pre>`;
    }

    // === JUSTICE ===
    async function loadJustice() {
        const el = document.getElementById('cd-justice');
        if (!el) return;
        el.innerHTML = `
            <div class="cd-grid">
                <button class="cd-btn" onclick="cdJusticeAwardCard()">Award Card</button>
                <button class="cd-btn" onclick="cdJusticeUseRepShield()">Use Rep Shield</button>
            </div>
            <div id="cd-justice-result" class="cd-result"></div>`;
    }
    async function cdJusticeAwardCard() {
        const id = prompt('Player ID:');
        const card = prompt('Card type:');
        if (!id || !card) return;
        try { await api('/justice/award-card', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ player_id: id, card_type: card }) }); showJusticeResult('Card awarded!'); }
        catch (e) { showJusticeResult('Failed: ' + e.message); }
    }
    async function cdJusticeUseRepShield() {
        const id = prompt('Player ID:');
        if (!id) return;
        try { await api('/justice/use-rep-shield', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ player_id: id }) }); showJusticeResult('Rep shield used!'); }
        catch (e) { showJusticeResult('Failed: ' + e.message); }
    }
    function showJusticeResult(msg) {
        const el = document.getElementById('cd-justice-result');
        if (el) el.innerHTML = `<p style="color:#80cbc4;">${esc(msg)}</p>`;
    }

    // === GOVERNANCE ===
    async function loadGovernance() {
        const el = document.getElementById('cd-governance');
        if (!el) return;
        el.innerHTML = `
            <div class="cd-grid">
                <button class="cd-btn" onclick="cdGovernanceClose()">Close</button>
                <button class="cd-btn" onclick="cdGovernanceRegister()">Register</button>
                <button class="cd-btn" onclick="cdGovernanceGovernor()">Governor</button>
            </div>
            <div id="cd-governance-result" class="cd-result"></div>`;
    }
    async function cdGovernanceClose() {
        const id = prompt('Governance ID:');
        if (!id) return;
        try { await api('/governance/close', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ governance_id: id }) }); showGovernanceResult('Closed!'); }
        catch (e) { showGovernanceResult('Failed: ' + e.message); }
    }
    async function cdGovernanceRegister() {
        const name = prompt('Candidate name:');
        if (!name) return;
        try { await api('/governance/register', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name }) }); showGovernanceResult('Registered!'); }
        catch (e) { showGovernanceResult('Failed: ' + e.message); }
    }
    async function cdGovernanceGovernor() {
        const id = prompt('Governance ID:');
        if (!id) return;
        try { const res = await api('/governance/governor?governance_id=' + encodeURIComponent(id)); showGovernanceResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showGovernanceResult('Failed: ' + e.message); }
    }
    function showGovernanceResult(msg) {
        const el = document.getElementById('cd-governance-result');
        if (el) el.innerHTML = `<pre style="color:#80cbc4;font-size:11px;white-space:pre-wrap;">${esc(msg)}</pre>`;
    }

    // === REPLAY ===
    async function loadReplay() {
        const el = document.getElementById('cd-replay');
        if (!el) return;
        el.innerHTML = `
            <div class="cd-grid">
                <button class="cd-btn" onclick="cdReplayStart()">Start Recording</button>
                <button class="cd-btn" onclick="cdReplayStop()">Stop Recording</button>
                <button class="cd-btn" onclick="cdReplayCapture()">Capture Frame</button>
                <button class="cd-btn" onclick="cdReplayFrames()">Get Frames</button>
                <button class="cd-btn" onclick="cdReplayPlayer()">Get Player</button>
                <button class="cd-btn" onclick="cdReplayState()">Get State</button>
            </div>
            <div id="cd-replay-result" class="cd-result"></div>`;
    }
    async function cdReplayStart() {
        try { await api('/replay/start', { method: 'POST' }); showReplayResult('Recording started!'); }
        catch (e) { showReplayResult('Failed: ' + e.message); }
    }
    async function cdReplayStop() {
        try { await api('/replay/stop', { method: 'POST' }); showReplayResult('Recording stopped!'); }
        catch (e) { showReplayResult('Failed: ' + e.message); }
    }
    async function cdReplayCapture() {
        try { await api('/replay/capture', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({}) }); showReplayResult('Frame captured!'); }
        catch (e) { showReplayResult('Failed: ' + e.message); }
    }
    async function cdReplayFrames() {
        try { const res = await api('/replay/frames'); showReplayResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showReplayResult('Failed: ' + e.message); }
    }
    async function cdReplayPlayer() {
        const wallet = prompt('Wallet:');
        if (!wallet) return;
        try { const res = await api('/replay/player?wallet=' + encodeURIComponent(wallet)); showReplayResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showReplayResult('Failed: ' + e.message); }
    }
    async function cdReplayState() {
        try { const res = await api('/replay/state'); showReplayResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showReplayResult('Failed: ' + e.message); }
    }
    function showReplayResult(msg) {
        const el = document.getElementById('cd-replay-result');
        if (el) el.innerHTML = `<pre style="color:#80cbc4;font-size:11px;white-space:pre-wrap;">${esc(msg)}</pre>`;
    }

    // === IDENTITY ===
    async function loadIdentity() {
        const el = document.getElementById('cd-identity');
        if (!el) return;
        el.innerHTML = `
            <div class="cd-grid">
                <button class="cd-btn" onclick="cdIdentityLink()">Link</button>
                <button class="cd-btn" onclick="cdIdentityUnlink()">Unlink</button>
                <button class="cd-btn" onclick="cdIdentityResolve()">Resolve</button>
                <button class="cd-btn" onclick="cdIdentitySnapshot()">Snapshot</button>
                <button class="cd-btn" onclick="cdIdentityRecord()">Record</button>
            </div>
            <div id="cd-identity-result" class="cd-result"></div>`;
    }
    async function cdIdentityLink() {
        const wallet = prompt('Wallet:');
        if (!wallet) return;
        try { await api('/identity/link', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ wallet }) }); showIdentityResult('Linked!'); }
        catch (e) { showIdentityResult('Failed: ' + e.message); }
    }
    async function cdIdentityUnlink() {
        const wallet = prompt('Wallet:');
        if (!wallet) return;
        try { await api('/identity/unlink', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ wallet }) }); showIdentityResult('Unlinked!'); }
        catch (e) { showIdentityResult('Failed: ' + e.message); }
    }
    async function cdIdentityResolve() {
        const wallet = prompt('Wallet:');
        if (!wallet) return;
        try { const res = await api('/identity/resolve?wallet=' + encodeURIComponent(wallet)); showIdentityResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showIdentityResult('Failed: ' + e.message); }
    }
    async function cdIdentitySnapshot() {
        const wallet = prompt('Wallet:');
        if (!wallet) return;
        try { const res = await api('/identity/snapshot?wallet=' + encodeURIComponent(wallet)); showIdentityResult(JSON.stringify(res).slice(0, 300)); }
        catch (e) { showIdentityResult('Failed: ' + e.message); }
    }
    async function cdIdentityRecord() {
        const wallet = prompt('Wallet:');
        const data = prompt('Data (JSON):');
        if (!wallet) return;
        try { await api('/identity/record', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ wallet, data: data ? JSON.parse(data) : {} }) }); showIdentityResult('Recorded!'); }
        catch (e) { showIdentityResult('Failed: ' + e.message); }
    }
    function showIdentityResult(msg) {
        const el = document.getElementById('cd-identity-result');
        if (el) el.innerHTML = `<pre style="color:#80cbc4;font-size:11px;white-space:pre-wrap;">${esc(msg)}</pre>`;
    }

    // === PET-BATTLE ===
    async function loadPetBattle() {
        const el = document.getElementById('cd-petbattle');
        if (!el) return;
        el.innerHTML = `
            <div class="cd-grid">
                <button class="cd-btn" onclick="cdPetBattleChallenge()">Challenge</button>
                <button class="cd-btn" onclick="cdPetBattleResolve()">Resolve</button>
            </div>
            <div id="cd-petbattle-result" class="cd-result"></div>`;
    }
    async function cdPetBattleChallenge() {
        const id = prompt('Pet ID:');
        const target = prompt('Target pet ID:');
        if (!id || !target) return;
        try { await api('/pet-battle/challenge', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ pet_id: id, target_pet_id: target }) }); showPetBattleResult('Challenge sent!'); }
        catch (e) { showPetBattleResult('Failed: ' + e.message); }
    }
    async function cdPetBattleResolve() {
        const id = prompt('Battle ID:');
        if (!id) return;
        try { await api('/pet-battle/resolve', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ battle_id: id }) }); showPetBattleResult('Resolved!'); }
        catch (e) { showPetBattleResult('Failed: ' + e.message); }
    }
    function showPetBattleResult(msg) {
        const el = document.getElementById('cd-petbattle-result');
        if (el) el.innerHTML = `<p style="color:#80cbc4;">${esc(msg)}</p>`;
    }

    window.openCommunityDashboard = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        if (window.PanelManager) window.PanelManager.open(overlayEl);
        else overlayEl.style.display = 'flex';
        switchCDTab('church');
    };
    window.switchCDTab = switchCDTab;

    // EVERY handler name this surface's rendered markup uses must exist on `window` — an inline
    // handler resolves its name there, and every function below is scoped to this IIFE. Without this
    // list all 25 controls (church, justice, governance, replay, identity, pet-battle) were inert.
    Object.assign(window, {
        cdChurchAddItem, cdChurchAddMember, cdChurchItems, cdChurchRegion, cdChurchRemoveMember,
        cdChurchRitual, cdChurchRituals, cdGovernanceClose, cdGovernanceGovernor, cdGovernanceRegister,
        cdIdentityLink, cdIdentityRecord, cdIdentityResolve, cdIdentitySnapshot, cdIdentityUnlink,
        cdJusticeAwardCard, cdJusticeUseRepShield, cdPetBattleChallenge, cdPetBattleResolve,
        cdReplayCapture, cdReplayFrames, cdReplayPlayer, cdReplayStart, cdReplayState, cdReplayStop,
    });

    if (document.getElementById('community-dashboard-overlay')) init();
})();
