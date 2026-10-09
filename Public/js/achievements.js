// ============================================================================
// achievements.js — Achievement system (REAL API, rewired 2026-09-03)
// ----------------------------------------------------------------------------
// Backend: achievement_handlers.go
//   GET /api/achievements?wallet=  → { wallet, unlocked, definitions, progress, total_won }
//   GET /api/achievement-stats     → { stats: [...] }  (leaderboard)
// All values server-authoritative. No mock-data fallback.
// ============================================================================

(function () {
    'use strict';
    var API_BASE = '/api';
    let overlayEl = null;
    let achievements = [];
    let definitions = [];
    let progress = {};
    let unlockedSet = new Set();
    let activeCategory = 'all';

    function init() {
        if (overlayEl) return;
        overlayEl = document.getElementById('achievements-overlay');
        if (!overlayEl) return;
        overlayEl.className = 'overlay vbt-overlay achievements-overlay';
        overlayEl.style.display = 'none';

        overlayEl.innerHTML = `
            <div class="achievements-panel">
                <button class="achievements-close" onclick="window.closeAchievements()">✕</button>
                <h2>🏆 Achievements</h2>
                <div class="achievements-tabs" id="achievements-tabs"></div>
                <div class="achievements-grid" id="achievements-grid"></div>
            </div>`;

        loadAchievements();
    }

    function getWallet() {
        if (window.getWalletAddress) return window.getWalletAddress() || '';
        if (window.getActiveWallet) return window.getActiveWallet() || '';
        return localStorage.getItem('wallet_address') || '';
    }

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

    async function loadAchievements() {
        const wallet = getWallet();
        if (!wallet) {
            renderEmpty();
            return;
        }
        try {
            const res = await api('/achievements?wallet=' + encodeURIComponent(wallet));
            definitions = res.definitions || [];
            unlockedSet = new Set(res.unlocked || []);
            progress = res.progress || [];
            // Build achievements list from definitions + unlock state
            achievements = definitions.map(d => ({
                id: d.id,
                name: d.title,
                desc: d.description,
                category: d.category,
                rarity: d.rarity,
                unlocked: unlockedSet.has(d.id),
                claimed: unlockedSet.has(d.id),
                progress: (progress && progress[d.id]) || 0,
                icon: getRarityIcon(d.rarity),
            }));
            renderAchievements();
        } catch (e) {
            console.warn('[Achievements] API unreachable:', e.message);
            renderEmpty();
        }
    }

    function getRarityIcon(rarity) {
        return rarity === 'legendary' ? '✨' : rarity === 'rare' ? '💎' : rarity === 'uncommon' ? '🔷' : '🏅';
    }

    function renderEmpty() {
        const gridEl = document.getElementById('achievements-grid');
        const tabsEl = document.getElementById('achievements-tabs');
        if (tabsEl) tabsEl.innerHTML = '';
        if (gridEl) gridEl.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No achievements yet.</p>';
    }

    function renderAchievements() {
        const tabsEl = document.getElementById('achievements-tabs');
        const gridEl = document.getElementById('achievements-grid');
        if (!tabsEl || !gridEl) return;

        const categories = ['all', ...new Set(definitions.map(d => d.category).filter(Boolean))];
        tabsEl.innerHTML = categories.map(c =>
            `<button class="achievement-tab ${c === activeCategory ? 'active' : ''}" data-category="${c}" onclick="window.filterAchievements('${c}')">${esc(c)}</button>`
        ).join('');

        const filtered = achievements.filter(a => activeCategory === 'all' || a.category === activeCategory);
        if (!filtered.length) {
            gridEl.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No achievements yet.</p>';
            return;
        }
        gridEl.innerHTML = filtered.map(a => {
            const isUnlocked = a.unlocked || a.claimed;
            return `
            <div class="achievement-badge ${isUnlocked ? 'unlocked' : 'locked'}" data-id="${a.id}">
                <div class="achievement-icon">${a.icon || '🏅'}</div>
                <div class="achievement-name">${esc(a.name || a.title)}</div>
                <div class="achievement-desc">${esc(a.description || a.desc)}</div>
                <div class="achievement-progress">
                    <div class="achievement-progress-bar">
                        <div class="achievement-progress-fill" style="width: ${a.progress || 0}%"></div>
                    </div>
                    <span class="achievement-progress-text">${a.progress || 0}%</span>
                </div>
                ${isUnlocked && !a.claimed ? '<button class="vbt-btn vbt-btn-primary claim-btn">Claim</button>' : ''}
                ${a.claimed ? '<span class="claimed-badge">✓ Claimed</span>' : ''}
            </div>`;
        }).join('');
    }

    function filterAchievements(category) {
        activeCategory = category;
        renderAchievements();
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }

    window.openAchievements = function () {
        init();
        // Single-Navigation: opening this surface closes any other overlay. It uses
        // the INLINE display contract (`.vbt-overlay`), so no `.hidden` is applied.
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        if (overlayEl) overlayEl.style.display = 'flex';
    };
    window.closeAchievements = function () {
        if (overlayEl) overlayEl.style.display = 'none';
    };
    window.filterAchievements = filterAchievements;

    if (document.getElementById('achievements-overlay')) init();
})();
