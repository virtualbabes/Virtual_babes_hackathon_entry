// ============================================================================
// advertising.js — Advertising System (ads, sponsored content)
// ============================================================================
(function () {
    'use strict';
    var API_BASE = '/api';
    let overlayEl, adsEl, myAdsEl, regionAdsEl, statusEl;

    function init() {
        if (overlayEl) return;
        const html = `
<div id="ad-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel ad-panel">
        <button id="btn-close-ad" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>📺 Advertising</h2>
        <p style="font-size:12px;color:#b0bec5;">Discovery, not interruption. Sponsored content, infrastructure modules, educational content.</p>
        <div class="ad-tabs">
            <button class="ad-tab active" data-tab="active">Active Ads</button>
            <button class="ad-tab" data-tab="my-ads">My Ads</button>
            <button class="ad-tab" data-tab="region">Region</button>
            <button class="ad-tab" data-tab="create">Create</button>
        </div>
        <div id="ad-active" class="ad-active"></div>
        <div id="ad-my-ads" class="ad-my-ads" style="display:none;"></div>
        <div id="ad-region" class="ad-region" style="display:none;"></div>
        <div id="ad-create" class="ad-create" style="display:none;">
            <input id="ad-title" placeholder="Ad Title" />
            <input id="ad-description" placeholder="Description" />
            <select id="ad-type">
                <option value="sponsored">Sponsored</option>
                <option value="infrastructure">Infrastructure</option>
                <option value="educational">Educational</option>
                <option value="community">Community</option>
                <option value="creator">Creator</option>
            </select>
            <input id="ad-budget" placeholder="Budget (micro)" type="number" />
            <input id="ad-cpm" placeholder="CPM (micro)" type="number" />
            <input id="ad-regions" placeholder="Target Regions (comma-separated)" />
            <input id="ad-expires" placeholder="Expires (YYYY-MM-DD)" />
            <button class="vbt-btn" id="ad-create-btn">Create Ad</button>
        </div>
        <p id="ad-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('ad-overlay');
        statusEl = document.getElementById('ad-status');
        document.getElementById('btn-close-ad').addEventListener('click', () => { overlayEl.style.display = 'none'; });
        document.querySelectorAll('.ad-tab').forEach(btn => {
            btn.addEventListener('click', () => {
                document.querySelectorAll('.ad-tab').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                switchAdTab(btn.dataset.tab);
            });
        });
        document.getElementById('ad-create-btn').addEventListener('click', createAd);
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>\"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }
    function fmtVBV(micro) { return (parseFloat(micro) / 1000000).toFixed(2); }

    async function api(path, opts) {
        const url = path.startsWith('/api/') ? path : API_BASE + path;
        const resp = await fetch(url, opts);
        if (!resp.ok) { let d = ''; try { d = (await resp.json()).error || ''; } catch(e){} throw new Error(d || ('HTTP ' + resp.status)); }
        return resp.json();
    }

    function switchAdTab(tab) {
        document.querySelectorAll('.ad-tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab)); document.getElementById('ad-active').style.display = tab === 'active' ? 'block' : 'none';
        document.getElementById('ad-my-ads').style.display = tab === 'my-ads' ? 'block' : 'none';
        document.getElementById('ad-region').style.display = tab === 'region' ? 'block' : 'none';
        document.getElementById('ad-create').style.display = tab === 'create' ? 'block' : 'none';
        if (tab === 'active') loadActive();
        if (tab === 'my-ads') loadMyAds();
        if (tab === 'region') loadRegionAds();
    }

    async function loadActive() {
        try {
            const res = await api('/api/ads?status=active');
            const el = document.getElementById('ad-active');
            if (!el) return;
            if (!(res.ads?.length ?? 0)) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No active ads.</p>'; return; }
            el.innerHTML = res.ads.map(a => `
                <div class="ad-ad">
                    <span class="ad-title">${esc(a.title)}</span>
                    <span class="ad-type">${esc(a.ad_type)}</span>
                    <span class="ad-advertiser">${esc(a.advertiser.slice(0, 12))}…</span>
                    <span class="ad-impressions">${a.impressions} impressions</span>
                </div>
            `).join('');
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    async function loadMyAds() {
        try {
            const res = await api('/api/ads/advertiser');
            const el = document.getElementById('ad-my-ads');
            if (!el) return;
            if (!(res.ads?.length ?? 0)) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No ads yet. Create one!</p>'; return; }
            el.innerHTML = res.ads.map(a => `
                <div class="ad-ad">
                    <span class="ad-title">${esc(a.title)}</span>
                    <span class="ad-status">${esc(a.status)}</span>
                    <span class="ad-spent">${fmtVBV(a.spent_micro)}/${fmtVBV(a.budget_micro)}</span>
                </div>
            `).join('');
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    async function loadRegionAds() {
        try {
            const res = await api('/api/ads/region?region=Base');
            const el = document.getElementById('ad-region');
            if (!el) return;
            if (!(res.ads?.length ?? 0)) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No ads in this region.</p>'; return; }
            el.innerHTML = res.ads.map(a => `
                <div class="ad-ad">
                    <span class="ad-title">${esc(a.title)}</span>
                    <span class="ad-type">${esc(a.ad_type)}</span>
                    <span class="ad-advertiser">${esc(a.advertiser.slice(0, 12))}…</span>
                </div>
            `).join('');
        } catch (e) { setStatus('Load failed: ' + e.message); }
    }

    async function createAd() {
        try {
            const body = {
                title: document.getElementById('ad-title')?.value,
                description: document.getElementById('ad-description')?.value,
                ad_type: document.getElementById('ad-type')?.value,
                budget_micro: parseInt(document.getElementById('ad-budget')?.value) || 0,
                cpm_micro: parseInt(document.getElementById('ad-cpm')?.value) || 0,
                target_regions: document.getElementById('ad-regions')?.value.split(',').map(s => s.trim()).filter(s => s),
                expires_at: document.getElementById('ad-expires')?.value
            };
            await api('/api/ad/create', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
            setStatus('Ad created!');
            loadMyAds();
        } catch (e) { setStatus('Create failed: ' + e.message); }
    }

    function setStatus(m) { if (statusEl) statusEl.textContent = m || ''; }

    window.openAdvertising = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        loadActive();
    };
})();
