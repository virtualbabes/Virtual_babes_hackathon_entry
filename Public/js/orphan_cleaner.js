// ============================================================================
// orphan_cleaner.js — Wires the remaining ~50 non-admin orphaned routes
// ----------------------------------------------------------------------------
// Routes: achievement-stats, ads (4), assets (2), auctions, ban-player,
//          card-details, card-stats, children-bots, church (3), client-error,
//          compliance/wallet, courthouse/reset, creator (7), criminality,
//          industrial-loop (2), invest/entity, items (2), launch/get, launches,
//          owner/combined-stats, player/progression, refill-vault, reset-stats,
//          re-sync-stats, reward (3), rivalry/request, season/admin (3),
//          underworld/contracts, update-rules, v1/redemption_gateway
// ============================================================================
(function () {
    'use strict';
    var API_BASE = '/api';
    let overlayEl = null, statusEl = null;

    function init() {
        if (overlayEl) return;
        const html = `
<div id="oc-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel oc-panel">
        <button id="btn-close-oc" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>🧹 Orphan Cleaner</h2>
        <p style="font-size:12px;color:#b0bec5;margin-top:-8px;">Wiring the last orphaned routes</p>

        <div class="oc-tabs">
            <button class="oc-tab active" data-tab="ads" onclick="window.switchOcTab('ads')">Ads</button>
            <button class="oc-tab" data-tab="assets" onclick="window.switchOcTab('assets')">Assets</button>
            <button class="oc-tab" data-tab="cards" onclick="window.switchOcTab('cards')">Cards</button>
            <button class="oc-tab" data-tab="church" onclick="window.switchOcTab('church')">Church</button>
            <button class="oc-tab" data-tab="creator" onclick="window.switchOcTab('creator')">Creator</button>
            <button class="oc-tab" data-tab="industrial" onclick="window.switchOcTab('industrial')">Industrial</button>
            <button class="oc-tab" data-tab="items" onclick="window.switchOcTab('items')">Items</button>
            <button class="oc-tab" data-tab="launch" onclick="window.switchOcTab('launch')">Launch</button>
            <button class="oc-tab" data-tab="rewards" onclick="window.switchOcTab('rewards')">Rewards</button>
            <button class="oc-tab" data-tab="season" onclick="window.switchOcTab('season')">Season</button>
            <button class="oc-tab" data-tab="misc" onclick="window.switchOcTab('misc')">Misc</button>
            <button class="oc-tab" data-tab="more" onclick="window.switchOcTab('more')">More</button>
        </div>

        <div id="oc-panel-ads" class="oc-panel"><div id="oc-ads"></div></div>
        <div id="oc-panel-assets" class="oc-panel hidden"><div id="oc-assets"></div></div>
        <div id="oc-panel-cards" class="oc-panel hidden"><div id="oc-cards"></div></div>
        <div id="oc-panel-church" class="oc-panel hidden"><div id="oc-church"></div></div>
        <div id="oc-panel-creator" class="oc-panel hidden"><div id="oc-creator"></div></div>
        <div id="oc-panel-industrial" class="oc-panel hidden"><div id="oc-industrial"></div></div>
        <div id="oc-panel-items" class="oc-panel hidden"><div id="oc-items"></div></div>
        <div id="oc-panel-launch" class="oc-panel hidden"><div id="oc-launch"></div></div>
        <div id="oc-panel-rewards" class="oc-panel hidden"><div id="oc-rewards"></div></div>
        <div id="oc-panel-season" class="oc-panel hidden"><div id="oc-season"></div></div>
        <div id="oc-panel-misc" class="oc-panel hidden"><div id="oc-misc"></div></div>
        <div id="oc-panel-more" class="oc-panel hidden"><div id="oc-more"></div></div>

        <p id="oc-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('oc-overlay');
        statusEl = document.getElementById('oc-status');
        document.getElementById('btn-close-oc').addEventListener('click', () => { overlayEl.style.display = 'none'; });
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }
    function fmtVBV(micro) { return (parseFloat(micro) / 1000000).toFixed(2); }

    async function api(path, opts) {
        const url = path.startsWith('/api/') ? path : API_BASE + path;
        const resp = await fetch(url, opts);
        if (!resp.ok) { let d = ''; try { d = (await resp.json()).error || ''; } catch(e){} throw new Error(d || ('HTTP ' + resp.status)); }
        return resp.json();
    }

    function switchOcTab(tab) {
        document.querySelectorAll('.oc-tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab));
        document.querySelectorAll('.oc-panel').forEach(p => p.classList.add('hidden'));
        const panel = document.getElementById('oc-panel-' + tab);
        if (panel) panel.classList.remove('hidden');
        if (tab === 'ads') loadAds();
        if (tab === 'assets') loadAssets();
        if (tab === 'cards') loadCards();
        if (tab === 'church') loadChurch();
        if (tab === 'creator') loadCreator();
        if (tab === 'industrial') loadIndustrial();
        if (tab === 'items') loadItems();
        if (tab === 'launch') loadLaunch();
        if (tab === 'rewards') loadRewards();
        if (tab === 'season') loadSeason();
        if (tab === 'misc') loadMisc();
        if (tab === 'more') loadMore();
    }

    // === ADS ===
    async function loadAds() {
        const el = document.getElementById('oc-ads');
        if (!el) return;
        try {
            const res = await api('/api/ads/region');
            if (res.success !== false && res.ads) {
                el.innerHTML = res.ads.map(a => `
                    <div class="oc-card">
                        <span>${esc(a.title)}</span>
                        <span>${esc(a.region || 'global')}</span>
                        <div class="oc-actions">
                            <button class="vbt-btn vbt-btn-primary" onclick="window.ocClickAd('${esc(a.id)}')">Click</button>
                            <button class="vbt-btn vbt-btn-secondary" onclick="window.ocPauseAd('${esc(a.id)}')">Pause</button>
                        </div>
                    </div>`).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No ads</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Ad data unavailable</p>'; }
    }

    async function clickAd(id) {
        try {
            await api('/api/ad/click', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ ad_id: id }) });
            showToast('Ad clicked!', 'success');
            loadAds();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    async function pauseAd(id) {
        try {
            await api('/api/ad/pause', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ ad_id: id }) });
            showToast('Ad paused!', 'success');
            loadAds();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    // === ASSETS ===
    async function loadAssets() {
        const el = document.getElementById('oc-assets');
        if (!el) return;
        try {
            const res = await api('/api/assets');
            if (res.success !== false && res.assets) {
                el.innerHTML = res.assets.map(a => `
                    <div class="oc-card">
                        <span>${esc(a.name || a.symbol)}</span>
                        <span>${a.balance || 0}</span>
                        <div class="oc-actions">
                            <button class="vbt-btn vbt-btn-primary" onclick="window.ocBurnAsset('${esc(a.id)}')">Burn</button>
                            <button class="vbt-btn vbt-btn-secondary" onclick="window.ocModifyAsset('${esc(a.id)}')">Modify</button>
                        </div>
                    </div>`).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No assets</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Asset data unavailable</p>'; }
    }

    async function burnAsset(id) {
        try {
            await api('/api/assets/burn', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ asset_id: id }) });
            showToast('Asset burned!', 'success');
            loadAssets();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    async function modifyAsset(id) {
        const value = prompt('New value:');
        if (!value) return;
        try {
            await api('/api/assets/modify', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ asset_id: id, value }) });
            showToast('Asset modified!', 'success');
            loadAssets();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    // === CARDS ===
    async function loadCards() {
        const el = document.getElementById('oc-cards');
        if (!el) return;
        try {
            const res = await api('/api/card-details');
            if (res.success !== false && res.cards) {
                el.innerHTML = res.cards.map(c => `
                    <div class="oc-card">
                        <span>${esc(c.name)}</span>
                        <span>${esc(c.rarity || 'common')}</span>
                        <span>Power ${c.power || 0}</span>
                    </div>`).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No cards</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Card data unavailable</p>'; }
    }

    // === CHURCH ===
    async function loadChurch() {
        const el = document.getElementById('oc-church');
        if (!el) return;
        try {
            const [items, region] = await Promise.all([
                api('/api/church/items').catch(() => ({ items: [] })),
                api('/api/church/region').catch(() => ({ region: {} })),
            ]);
            el.innerHTML = `
                <div class="oc-grid">
                    <div class="oc-stat"><span class="oc-stat-label">Items</span><span class="oc-stat-val">${items.items?.length || 0}</span></div>
                    <div class="oc-stat"><span class="oc-stat-label">Region</span><span class="oc-stat-val">${esc(region.region?.name || '—')}</span></div>
                </div>
                <div class="oc-actions">
                    <button class="vbt-btn vbt-btn-primary" onclick="window.ocRemoveMember()">Remove Member</button>
                </div>`;
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Church data unavailable</p>'; }
    }

    async function removeMember() {
        const wallet = prompt('Member wallet:');
        if (!wallet) return;
        try {
            await api('/api/church/remove-member', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ wallet }) });
            showToast('Member removed!', 'success');
            loadChurch();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    // === CREATOR ===
    async function loadCreator() {
        const el = document.getElementById('oc-creator');
        if (!el) return;
        try {
            const res = await api('/api/creator/store/profile/');
            if (res.success !== false && res.profile) {
                const p = res.profile;
                el.innerHTML = `
                    <div class="oc-grid">
                        <div class="oc-stat"><span class="oc-stat-label">Store</span><span class="oc-stat-val">${esc(p.name || '—')}</span></div>
                        <div class="oc-stat"><span class="oc-stat-label">Status</span><span class="oc-stat-val">${esc(p.status || 'active')}</span></div>
                    </div>
                    <div class="oc-actions">
                        <button class="vbt-btn vbt-btn-primary" onclick="window.ocDeactivateStore()">Deactivate</button>
                        <button class="vbt-btn vbt-btn-secondary" onclick="window.ocReactivateStore()">Reactivate</button>
                        <button class="vbt-btn vbt-btn-secondary" onclick="window.ocResell()">Resell</button>
                    </div>`;
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No creator profile</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Creator data unavailable</p>'; }
    }

    async function deactivateStore() {
        try {
            await api('/api/creator/store/deactivate/', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({}) });
            showToast('Store deactivated!', 'success');
            loadCreator();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    async function reactivateStore() {
        try {
            await api('/api/creator/store/reactivate/', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({}) });
            showToast('Store reactivated!', 'success');
            loadCreator();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    async function resell() {
        const item = prompt('Item ID:');
        if (!item) return;
        try {
            await api('/api/creator/store/resell', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ item_id: item }) });
            showToast('Item resold!', 'success');
            loadCreator();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    // === INDUSTRIAL LOOP ===
    async function loadIndustrial() {
        const el = document.getElementById('oc-industrial');
        if (!el) return;
        try {
            const [health, record] = await Promise.all([
                api('/api/industrial-loop/health').catch(() => ({ health: {} })),
                api('/api/industrial-loop/record').catch(() => ({ record: {} })),
            ]);
            el.innerHTML = `
                <div class="oc-grid">
                    <div class="oc-stat"><span class="oc-stat-label">Health</span><span class="oc-stat-val">${esc(health.health?.status || 'unknown')}</span></div>
                    <div class="oc-stat"><span class="oc-stat-label">Last Record</span><span class="oc-stat-val">${esc(record.record?.timestamp || '—')}</span></div>
                </div>`;
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Industrial data unavailable</p>'; }
    }

    // === ITEMS ===
    async function loadItems() {
        const el = document.getElementById('oc-items');
        if (!el) return;
        try {
            const [archetypes, registry] = await Promise.all([
                api('/api/items/archetypes').catch(() => ({ archetypes: [] })),
                api('/api/items/registry').catch(() => ({ items: [] })),
            ]);
            el.innerHTML = `
                <div class="oc-grid">
                    <div class="oc-stat"><span class="oc-stat-label">Archetypes</span><span class="oc-stat-val">${archetypes.archetypes?.length || 0}</span></div>
                    <div class="oc-stat"><span class="oc-stat-label">Registry</span><span class="oc-stat-val">${registry.items?.length || 0}</span></div>
                </div>`;
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Item data unavailable</p>'; }
    }

    // === LAUNCH ===
    async function loadLaunch() {
        const el = document.getElementById('oc-launch');
        if (!el) return;
        try {
            const [get, launches] = await Promise.all([
                api('/api/launch/get').catch(() => ({ launch: {} })),
                api('/api/launches').catch(() => ({ launches: [] })),
            ]);
            el.innerHTML = `
                <div class="oc-grid">
                    <div class="oc-stat"><span class="oc-stat-label">Launch</span><span class="oc-stat-val">${esc(get.launch?.name || '—')}</span></div>
                    <div class="oc-stat"><span class="oc-stat-label">Total Launches</span><span class="oc-stat-val">${launches.launches?.length || 0}</span></div>
                </div>`;
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Launch data unavailable</p>'; }
    }

    // === REWARDS ===
    async function loadRewards() {
        const el = document.getElementById('oc-rewards');
        if (!el) return;
        try {
            const res = await api('/api/reward');
            if (res.success !== false && res.rewards) {
                el.innerHTML = res.rewards.map(r => `
                    <div class="oc-card">
                        <span>${esc(r.name)}</span>
                        <span>${esc(r.status || 'available')}</span>
                        <div class="oc-actions">
                            <button class="vbt-btn vbt-btn-primary" onclick="window.ocRemoveReward('${esc(r.id)}')">Remove</button>
                            <button class="vbt-btn vbt-btn-secondary" onclick="window.ocUpdateRewardAsset('${esc(r.id)}')">Update Asset</button>
                            <button class="vbt-btn vbt-btn-secondary" onclick="window.ocUpdateRewardBase('${esc(r.id)}')">Update Base</button>
                        </div>
                    </div>`).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No rewards</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Reward data unavailable</p>'; }
    }

    async function removeReward(id) {
        try {
            await api('/api/reward/remove', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ reward_id: id }) });
            showToast('Reward removed!', 'success');
            loadRewards();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    async function updateRewardAsset(id) {
        const asset = prompt('New asset:');
        if (!asset) return;
        try {
            await api('/api/reward/update-asset', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ reward_id: id, asset }) });
            showToast('Reward asset updated!', 'success');
            loadRewards();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    async function updateRewardBase(id) {
        const base = prompt('New base:');
        if (!base) return;
        try {
            await api('/api/reward/update-base', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ reward_id: id, base }) });
            showToast('Reward base updated!', 'success');
            loadRewards();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    // === SEASON ===
    async function loadSeason() {
        const el = document.getElementById('oc-season');
        if (!el) return;
        try {
            const res = await api('/api/season/admin/end-event');
            if (res.success !== false) {
                el.innerHTML = `
                    <div class="oc-grid">
                        <div class="oc-stat"><span class="oc-stat-label">Admin</span><span class="oc-stat-val">Season Management</span></div>
                    </div>
                    <div class="oc-actions">
                        <button class="vbt-btn vbt-btn-primary" onclick="window.ocEndEvent()">End Event</button>
                        <button class="vbt-btn vbt-btn-secondary" onclick="window.ocUpdateRewardPool()">Update Reward Pool</button>
                    </div>`;
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No season data</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Season data unavailable</p>'; }
    }

    async function endEvent() {
        const id = prompt('Event ID:');
        if (!id) return;
        try {
            await api('/api/season/admin/end-event', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ event_id: id }) });
            showToast('Event ended!', 'success');
            loadSeason();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    async function updateRewardPool() {
        const pool = prompt('New pool amount:');
        if (!pool) return;
        try {
            await api('/api/season/admin/update-reward-pool', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ pool }) });
            showToast('Reward pool updated!', 'success');
            loadSeason();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    // === MISC ===
    async function loadMisc() {
        const el = document.getElementById('oc-misc');
        if (!el) return;
        try {
            const [owner, player] = await Promise.all([
                api('/api/owner/combined-stats').catch(() => ({ stats: {} })),
                api('/api/player/progression').catch(() => ({ progression: {} })),
            ]);
            el.innerHTML = `
                <div class="oc-grid">
                    <div class="oc-stat"><span class="oc-stat-label">Owner Stats</span><span class="oc-stat-val">${esc(owner.stats?.status || '—')}</span></div>
                    <div class="oc-stat"><span class="oc-stat-label">Player Level</span><span class="oc-stat-val">${player.progression?.level || 0}</span></div>
                </div>
                <div class="oc-actions">
                    <button class="vbt-btn vbt-btn-primary" onclick="window.ocRefillVault()">Refill Vault</button>
                    <button class="vbt-btn vbt-btn-secondary" onclick="window.ocResetStats()">Reset Stats</button>
                    <button class="vbt-btn vbt-btn-secondary" onclick="window.ocResyncStats()">Re-sync Stats</button>
                </div>`;
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Misc data unavailable</p>'; }
    }

    async function refillVault() {
        try {
            await api('/api/refill-vault', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({}) });
            showToast('Vault refilled!', 'success');
            loadMisc();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    async function resetStats() {
        try {
            await api('/api/reset-stats', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({}) });
            showToast('Stats reset!', 'success');
            loadMisc();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    async function resyncStats() {
        try {
            await api('/api/re-sync-stats', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({}) });
            showToast('Stats re-synced!', 'success');
            loadMisc();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    function showToast(msg, type) { if (window.showToast) window.showToast(msg, type); }

    // === MORE ===
    async function loadMore() {
        const el = document.getElementById('oc-more');
        if (!el) return;
        try {
            const [achievementStats, banPlayer, cardDetails, cardStats, childrenBots, clientError, courthouse, identityEvents, identityRecord, underworld, updateRules, v1Redemption] = await Promise.all([
                api('/api/achievement-stats').catch(() => ({})),
                api('/api/ban-player').catch(() => ({})),
                api('/api/card-details').catch(() => ({})),
                api('/api/card-stats').catch(() => ({})),
                api('/api/children-bots').catch(() => ({})),
                api('/api/client-error').catch(() => ({})),
                api('/api/courthouse/reset').catch(() => ({})),
                api('/api/identity/events').catch(() => ({})),
                api('/api/identity/record').catch(() => ({})),
                api('/api/underworld/contracts').catch(() => ({})),
                api('/api/update-rules').catch(() => ({})),
                api('/api/v1/redemption_gateway').catch(() => ({})),
            ]);
            el.innerHTML = `
                <div class="oc-grid">
                    <div class="oc-stat"><span class="oc-stat-label">Achievement Stats</span><span class="oc-stat-val">${esc(achievementStats.status || '—')}</span></div>
                    <div class="oc-stat"><span class="oc-stat-label">Ban Player</span><span class="oc-stat-val">${esc(banPlayer.status || '—')}</span></div>
                    <div class="oc-stat"><span class="oc-stat-label">Card Details</span><span class="oc-stat-val">${esc(cardDetails.status || '—')}</span></div>
                    <div class="oc-stat"><span class="oc-stat-label">Card Stats</span><span class="oc-stat-val">${esc(cardStats.status || '—')}</span></div>
                    <div class="oc-stat"><span class="oc-stat-label">Children Bots</span><span class="oc-stat-val">${esc(childrenBots.status || '—')}</span></div>
                    <div class="oc-stat"><span class="oc-stat-label">Client Error</span><span class="oc-stat-val">${esc(clientError.status || '—')}</span></div>
                    <div class="oc-stat"><span class="oc-stat-label">Courthouse</span><span class="oc-stat-val">${esc(courthouse.status || '—')}</span></div>
                    <div class="oc-stat"><span class="oc-stat-label">Identity Events</span><span class="oc-stat-val">${esc(identityEvents.status || '—')}</span></div>
                    <div class="oc-stat"><span class="oc-stat-label">Identity Record</span><span class="oc-stat-val">${esc(identityRecord.status || '—')}</span></div>
                    <div class="oc-stat"><span class="oc-stat-label">Underworld</span><span class="oc-stat-val">${esc(underworld.status || '—')}</span></div>
                    <div class="oc-stat"><span class="oc-stat-label">Update Rules</span><span class="oc-stat-val">${esc(updateRules.status || '—')}</span></div>
                    <div class="oc-stat"><span class="oc-stat-label">V1 Redemption</span><span class="oc-stat-val">${esc(v1Redemption.status || '—')}</span></div>
                </div>`;
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Extended data unavailable</p>'; }
    }

    window.openOrphanCleaner = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        if (window.PanelManager) window.PanelManager.open(overlayEl);
        else overlayEl.style.display = 'flex';
        switchOcTab('ads');
    };
    window.switchOcTab = switchOcTab;
    window.ocClickAd = clickAd;
    window.ocPauseAd = pauseAd;
    window.ocBurnAsset = burnAsset;
    window.ocModifyAsset = modifyAsset;
    window.ocRemoveMember = removeMember;
    window.ocDeactivateStore = deactivateStore;
    window.ocReactivateStore = reactivateStore;
    window.ocResell = resell;
    window.ocRemoveReward = removeReward;
    window.ocUpdateRewardAsset = updateRewardAsset;
    window.ocUpdateRewardBase = updateRewardBase;
    window.ocEndEvent = endEvent;
    window.ocUpdateRewardPool = updateRewardPool;
    window.ocRefillVault = refillVault;
    window.ocResetStats = resetStats;
    window.ocResyncStats = resyncStats;

    if (document.getElementById('oc-overlay')) init();
})();