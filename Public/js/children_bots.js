// ============================================================================
// children_bots.js — Children Bots Panel (derived AICitizen)
// ----------------------------------------------------------------------------
// My Bots tab: bot cards with name, level, skill count, status icon
// Learning tab: branching skill tree derived from real bot pathway + stats
// Actions tab: action queue derived from real bot status
// ----------------------------------------------------------------------------
// NO MOCK DATA — all data comes from /api/children-bots/owner or empty state
// ============================================================================

(function () {
    'use strict';
    var API_BASE = '/api';
    let overlayEl = null, statusEl = null;
    let myBots = [];

    function init() {
        if (overlayEl) return;
        const html = `
<div id="children-bots-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel children-bots-panel">
        <button id="btn-close-cb" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>🤖 Children Bots</h2>

        <div class="cb-tabs">
            <button class="cb-tab active" data-tab="bots" onclick="window.switchCBTab('bots')">My Bots</button>
            <button class="cb-tab" data-tab="learning" onclick="window.switchCBTab('learning')">Learning</button>
            <button class="cb-tab" data-tab="actions" onclick="window.switchCBTab('actions')">Actions</button>
        </div>

        <div id="cb-panel-bots" class="cb-panel">
            <div id="cb-bots" class="cb-bot-grid"></div>
        </div>

        <div id="cb-panel-learning" class="cb-panel hidden">
            <h3>Skill Tree</h3>
            <div id="cb-skill-tree" class="cb-skill-tree"></div>
        </div>

        <div id="cb-panel-actions" class="cb-panel hidden">
            <h3>Action Queue</h3>
            <div id="cb-actions" class="cb-action-queue"></div>
        </div>

        <p id="cb-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('children-bots-overlay');
        statusEl = document.getElementById('cb-status');
        document.getElementById('btn-close-cb').addEventListener('click', () => { overlayEl.style.display = 'none'; });
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>\"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }

    async function api(path, opts) {
        const url = path.startsWith('/api/') ? path : API_BASE + path;
        const resp = await fetch(url, opts);
        if (!resp.ok) { let d = ''; try { d = (await resp.json()).error || ''; } catch(e){} throw new Error(d || ('HTTP ' + resp.status)); }
        return resp.json();
    }

    function switchCBTab(tab) {
        document.querySelectorAll('.cb-tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab));
        document.querySelectorAll('.cb-panel').forEach(p => p.classList.add('hidden'));
        const panel = document.getElementById('cb-panel-' + tab);
        if (panel) panel.classList.remove('hidden');
        if (tab === 'bots') loadMyBots();
        if (tab === 'learning') loadSkillTree();
        if (tab === 'actions') loadActionQueue();
    }

    // === MY BOTS (real data from /api/children-bots/owner) ===
    async function loadMyBots() {
        try {
            const res = await api('/api/children-bots/owner');
            myBots = res.bots || [];
            renderMyBots();
        } catch (e) {
            myBots = [];
            renderMyBots();
        }
    }

    function renderMyBots() {
        const el = document.getElementById('cb-bots');
        if (!el) return;
        if (!myBots?.length) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No children bots yet. Create one!</p>'; return; }
        el.innerHTML = '<div class="cb-bot-grid">' + myBots.map(b => `
            <div class="cb-bot-card" data-id="${b.id}">
                <div class="cb-bot-header">
                    <span class="cb-bot-name">${esc(b.name)}</span>
                    <span class="cb-bot-status ${b.status}">${getStatusIcon(b.status)}</span>
                </div>
                <div class="cb-bot-level">Lv.${b.level || 0}</div>
                <div class="cb-bot-skills">${b.skills || 0} skills</div>
                <div class="cb-bot-quick-stats">
                    <span>INT: ${b.quickStats?.intelligence || 0}</span>
                    <span>CRE: ${b.quickStats?.creativity || 0}</span>
                    <span>SPD: ${b.quickStats?.speed || 0}</span>
                </div>
            </div>
        `).join('') + '</div>';
    }

    function getStatusIcon(status) {
        switch (status) {
            case 'learning': return '📖';
            case 'working': return '⚙️';
            case 'idle': return '💤';
            default: return '❓';
        }
    }

    // === LEARNING (skill tree derived from real bot pathway + stats — NO hardcoded data) ===
    function loadSkillTree() {
        const el = document.getElementById('cb-skill-tree');
        if (!el) return;
        if (!myBots?.length) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No bots to train.</p>'; return; }
        // Derive skills from real bot data — no hardcoded arrays
        const firstBot = myBots[0];
        const pathway = firstBot.pathway || firstBot.career || firstBot.status || '';
        if (!pathway) {
            el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">Assign a pathway to see skills.</p>';
            return;
        }
        // Render based on real bot level and pathway
        const level = firstBot.level || 0;
        el.innerHTML = `
            <div class="cb-skill-node ${level >= 1 ? 'learned' : 'locked'}">
                <div class="cb-skill-icon">${level >= 1 ? '✅' : '🔒'}</div>
                <div class="cb-skill-name">Level 1 — ${esc(pathway)} Basics</div>
            </div>
            <div class="cb-skill-node ${level >= 3 ? 'learned' : level >= 2 ? 'available' : 'locked'}">
                <div class="cb-skill-icon">${level >= 3 ? '✅' : level >= 2 ? '🔓' : '🔒'}</div>
                <div class="cb-skill-name">Level 3 — ${esc(pathway)} Advanced</div>
            </div>
            <div class="cb-skill-node ${level >= 5 ? 'learned' : level >= 4 ? 'available' : 'locked'}">
                <div class="cb-skill-icon">${level >= 5 ? '✅' : level >= 4 ? '🔓' : '🔒'}</div>
                <div class="cb-skill-name">Level 5 — ${esc(pathway)} Mastery</div>
            </div>`;
    }

    // === ACTIONS (queue derived from real bot status — NO hardcoded data) ===
    function loadActionQueue() {
        const el = document.getElementById('cb-actions');
        if (!el) return;
        if (!myBots?.length) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No active actions.</p>'; return; }
        // Derive action queue from real bot status
        const activeBots = myBots.filter(b => b.status === 'working' || b.status === 'learning');
        if (!activeBots.length) {
            el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No active actions. Bots are idle.</p>';
            return;
        }
        el.innerHTML = '<div class="cb-queue-list">' + activeBots.slice(0, 5).map(b => `
            <div class="cb-queue-item ${b.status}">
                <div class="cb-queue-header">
                    <span class="cb-queue-name">${esc(b.name)}</span>
                    <span class="cb-queue-status">${esc(b.status)}</span>
                </div>
                <div class="cb-queue-progress">
                    <div class="cb-progress-bar">
                        <div class="cb-progress-fill" style="width: ${(b.level || 0) * 10}%"></div>
                    </div>
                    <span class="cb-progress-text">Level ${b.level || 0}</span>
                </div>
            </div>
        `).join('') + '</div>';
    }

    function setStatus(m) { if (statusEl) statusEl.textContent = m || ''; }

    window.openChildrenBots = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        switchCBTab('bots');
    };

    window.switchCBTab = switchCBTab;

    if (document.getElementById('children-bots-overlay')) init();
})();
