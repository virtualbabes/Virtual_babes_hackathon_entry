// ============================================================================
// rivalry.js — Rivalry System
// ============================================================================

var API_BASE = '/api';
let overlayEl = null, statusEl = null;
let myRivals = [];
let careerXP = {};

function init() {
    if (overlayEl) return;
    const html = `
<div id="rivalry-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel rivalry-panel">
        <button id="btn-close-riv" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>⚔️ Rivalry</h2>

        <div class="riv-tabs">
            <button class="riv-tab active" data-tab="active" onclick="window.switchRivTab('active')">Active Rivals</button>
            <button class="riv-tab" data-tab="challenge" onclick="window.switchRivTab('challenge')">Challenge</button>
            <button class="riv-tab" data-tab="career" onclick="window.switchRivTab('career')">Career XP</button>
            <button class="riv-tab" data-tab="matrix" onclick="window.switchRivTab('matrix')">Rivalry Matrix</button>
        </div>

        <div id="riv-panel-active" class="riv-panel">
            <h3>Active Rivalries</h3>
            <div id="riv-active" class="riv-active"></div>
        </div>

        <div id="riv-panel-challenge" class="riv-panel hidden">
            <h3>Challenge a Player</h3>
            <div id="riv-challenge" class="riv-challenge"></div>
        </div>

        <div id="riv-panel-career" class="riv-panel hidden">
            <h3>Career Progress</h3>
            <div id="riv-career" class="riv-career"></div>
        </div>

        <div id="riv-panel-matrix" class="riv-panel hidden">
            <h3>Rivalry Matrix</h3>
            <div id="riv-matrix" class="riv-matrix"></div>
        </div>

        <p id="riv-status" class="ai-status"></p>
    </div>
</div>`;
    document.body.insertAdjacentHTML('beforeend', html);
    overlayEl = document.getElementById('rivalry-overlay');
    statusEl = document.getElementById('riv-status');
    document.getElementById('btn-close-riv').addEventListener('click', () => { overlayEl.style.display = 'none'; });
}

function esc(s) { return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }

async function api(path, opts) {
    const url = path.startsWith('/api/') ? path : API_BASE + path;
    const resp = await fetch(url, opts);
    if (!resp.ok) { let d = ''; try { d = (await resp.json()).error || ''; } catch(e){} throw new Error(d || ('HTTP ' + resp.status)); }
    return resp.json();
}

function switchRivTab(tab) {
    document.querySelectorAll('.riv-tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab));
    document.querySelectorAll('.riv-panel').forEach(p => p.classList.add('hidden'));
    const panel = document.getElementById('riv-panel-' + tab);
    if (panel) panel.classList.remove('hidden');
    if (tab === 'active') loadActiveRivals();
    if (tab === 'challenge') loadChallenge();
    if (tab === 'career') loadCareerXP();
    if (tab === 'matrix') loadMatrix();
}

async function loadActiveRivals() {
    const el = document.getElementById('riv-active');
    if (!el) return;
    try {
        const res = await api('/api/rivalry/state');
        myRivals = res.rivals || res.active_rivals || [];
        renderActiveRivals();
    } catch (e) {
        console.warn('[Rivalry] API unreachable:', e.message);
        myRivals = [];
        renderActiveRivals();
    }
}

function renderActiveRivals() {
    const el = document.getElementById('riv-active');
    if (!el) return;
    if (!myRivals.length) { el.innerHTML = '<div class="panel-empty"><div class="panel-empty-icon">⚔️</div><div class="panel-empty-message">No active rivals. Challenge someone!</div></div>'; return; }
    el.innerHTML = myRivals.map(r => `
        <div class="riv-rival-card">
            <div class="riv-rival-name">${esc(r.name)}</div>
            <div class="riv-rival-record">${r.wins}W / ${r.losses}L</div>
            <div class="riv-rival-status ${r.status}">${esc(r.status)}</div>
            <button class="vbt-btn vbt-btn-primary" onclick="window.rivChallenge('${r.id}')">Challenge</button>
        </div>
    `).join('');
}

function loadChallenge() {
    const el = document.getElementById('riv-challenge');
    if (!el) return;
    el.innerHTML = `
        <div class="riv-challenge-form">
            <label>Opponent Wallet <input id="riv-opponent" type="text" placeholder="0x..." /></label>
            <label>Wager (VBV) <input id="riv-wager" type="number" value="100" /></label>
            <button class="vbt-btn vbt-btn-primary" onclick="window.rivSendChallenge()">Send Challenge</button>
        </div>
    `;
}

async function loadCareerXP() {
    const el = document.getElementById('riv-career');
    if (!el) return;
    try {
        const wallet = (window.getWalletAddress && window.getWalletAddress()) || '';
        const path = wallet ? `/api/career/progress?wallet=${encodeURIComponent(wallet)}` : '/api/career/progress';
        const res = await api(path);
        careerXP = res.careers || {};
        renderCareerXP();
    } catch (e) {
        console.warn('[Rivalry] API unreachable:', e.message);
        careerXP = {};
        renderCareerXP();
    }
}

function renderCareerXP() {
    const el = document.getElementById('riv-career');
    if (!el) return;
    const careers = Object.entries(careerXP);
    if (!careers.length) { el.innerHTML = '<div class="panel-empty"><div class="panel-empty-icon">📈</div><div class="panel-empty-message">No career progress yet.</div></div>'; return; }
    el.innerHTML = careers.map(([name, data]) => {
        const pct = Math.min(100, Math.floor((data.xp / data.nextLevel) * 100));
        return `
        <div class="riv-career-card">
            <div class="riv-career-name">${esc(name)}</div>
            <div class="riv-career-tier">Tier ${data.tier}</div>
            <div class="riv-career-xp">${data.xp} / ${data.nextLevel} XP</div>
            <div class="riv-career-bar"><div class="riv-career-fill" style="width: ${pct}%"></div>
        </div>`;
    }).join('');
}

async function loadMatrix() {
    const el = document.getElementById('riv-matrix');
    if (!el) return;
    try {
        const res = await api('/api/rivalry/detect');
        const rivalries = res.rivalries || [];
        renderMatrix(rivalries);
    } catch (e) {
        console.warn('[Rivalry] API unreachable:', e.message);
        renderMatrix([]);
    }
}

function renderMatrix(rivalries) {
    const el = document.getElementById('riv-matrix');
    if (!el) return;
    if (!rivalries.length) { el.innerHTML = '<div class="panel-empty"><div class="panel-empty-icon">📊</div><div class="panel-empty-message">No rivalries detected.</div></div>'; return; }
    el.innerHTML = rivalries.map(r => `
        <div class="riv-matrix-card">
            <div class="riv-matrix-name">${esc(r.name)}</div>
            <div class="riv-matrix-score">${r.score.toLocaleString()}</div>
            <div class="riv-matrix-status">${esc(r.status)}</div>
        </div>
    `).join('');
}

function challenge(id) {
    if (!window.canAfford(100)) { showToast('Need 100 VBV to challenge', 'error'); return; }
    if (window.txModalOpen) {
        window.txModalOpen('Challenge Rival', 100, () => {
            if (!window.deductVBV(100)) return;
            showToast('Challenge sent!', 'success');
        });
    }
}

function sendChallenge() {
    const opponent = document.getElementById('riv-opponent')?.value;
    const wager = parseInt(document.getElementById('riv-wager')?.value || 100);
    if (!opponent) { showToast('Enter opponent wallet', 'error'); return; }
    if (!window.canAfford(wager)) { showToast('Insufficient VBV', 'error'); return; }
    if (window.txModalOpen) {
        window.txModalOpen('Challenge ' + opponent.slice(0, 8), wager, () => {
            if (!window.deductVBV(wager)) return;
            showToast('Challenge sent!', 'success');
        });
    }
}

function showToast(msg, type) { if (window.showToast) window.showToast(msg, type); }

function openRivalryViewer() {
    init();
    if (window.hideAllOverlays) window.hideAllOverlays();
    overlayEl.style.display = 'flex';
    switchRivTab('active');
}

// ============================================================================
// RivalryEngine — exported API for app.js import
// ============================================================================
export const RivalryEngine = {
    init,
    openRivalryViewer,
    switchRivTab,
    challenge,
    sendChallenge,
    loadActiveRivals,
    loadCareerXP,
    loadMatrix,
};

// ============================================================================
// Window bindings (backward compatibility for inline HTML)
// ============================================================================
window.openRivalryViewer = openRivalryViewer;
window.switchRivTab = switchRivTab;
window.rivChallenge = challenge;
window.rivSendChallenge = sendChallenge;
