// ============================================================================
// governance.js — Governance System (REAL API WIRING)
// ============================================================================
(function () {
    'use strict';
    var API_BASE = '/api';
    let overlayEl, statusEl;
    let electionTimerInterval = null;

    function init() {
        if (overlayEl) return;
        const html = `
<div id="governance-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel governance-panel">
        <button id="btn-close-gv" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>⚖️ Governance</h2>

        <div class="gv-tabs">
            <button class="gv-tab active" data-tab="weight" onclick="window.switchGvTab('weight')">My Weight</button>
            <button class="gv-tab" data-tab="election" onclick="window.switchGvTab('election')">Election</button>
            <button class="gv-tab" data-tab="missions" onclick="window.switchGvTab('missions')">Justice Missions</button>
            <button class="gv-tab" data-tab="leaderboard" onclick="window.switchGvTab('leaderboard')">Leaderboard</button>
        </div>

        <div id="gv-panel-weight" class="gv-panel">
            <div id="gv-weight" class="gv-weight"></div>
        </div>

        <div id="gv-panel-election" class="gv-panel hidden">
            <div id="gv-election" class="gv-election"></div>
        </div>

        <div id="gv-panel-missions" class="gv-panel hidden">
            <h3>Justice Missions</h3>
            <div id="gv-missions" class="gv-missions"></div>
        </div>

        <div id="gv-panel-leaderboard" class="gv-panel hidden">
            <div id="gv-leaderboard" class="gv-leaderboard"></div>
        </div>

        <p id="gv-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('governance-overlay');
        statusEl = document.getElementById('gv-status');
        document.getElementById('btn-close-gv').addEventListener('click', () => { overlayEl.style.display = 'none'; if (electionTimerInterval) clearInterval(electionTimerInterval); });
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }
    function fmtVBV(micro) { return (parseFloat(micro) / 1000000).toFixed(2); }

    async function api(path, opts) {
        const url = path.startsWith('/api/') ? path : API_BASE + path;
        let resp;
        if (typeof window.api === 'function') {
            resp = await window.api(url, opts);
        } else {
            resp = await fetch(url, opts);
        }
        if (!resp.ok) { let d = ''; try { d = (await resp.json()).error || ''; } catch(e){} throw new Error(d || ('HTTP ' + resp.status)); }
        return resp.json();
    }

    function switchGvTab(tab) {
        document.querySelectorAll('.gv-tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab));
        document.querySelectorAll('.gv-panel').forEach(p => p.classList.add('hidden'));
        const panel = document.getElementById('gv-panel-' + tab);
        if (panel) panel.classList.remove('hidden');
        if (tab === 'weight') loadWeight();
        if (tab === 'election') loadElection();
        if (tab === 'missions') loadMissions();
        if (tab === 'leaderboard') loadLeaderboard();
    }

    async function loadWeight() {
        try {
            const res = await api('/api/governance/weight');
            const w = res.weight || {};
            const el = document.getElementById('gv-weight');
            if (!el) return;
            el.innerHTML = `
                <div class="gv-total">${w.total_score || 0}</div>
                <div class="gv-weight-row"><span>Trust</span><span>${w.trust_score || 0}</span></div>
                <div class="gv-weight-row"><span>Reputation</span><span>${w.reputation_score || 0}</span></div>
                <div class="gv-weight-row"><span>Economic</span><span>${w.economic_score || 0}</span></div>
                <div class="gv-weight-row"><span>Competitive</span><span>${w.competitive_score || 0}</span></div>
                <div class="gv-weight-row"><span>Community</span><span>${w.community_score || 0}</span></div>
                <div class="gv-weight-row"><span>Creative</span><span>${w.creative_score || 0}</span></div>
            `;
        } catch (e) {
            const el = document.getElementById('gv-weight');
            if (el) el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No governance data available</p>';
        }
    }

    async function loadElection() {
        const el = document.getElementById('gv-election');
        if (!el) return;
        try {
            const res = await api('/api/governance/election');
            if (res.success && res.election) {
                const election = res.election;
                el.innerHTML = `
                    <div class="gv-election-status">
                        <div class="gv-election-timer" id="gv-election-timer">Active Election</div>
                        <div class="gv-election-label">${esc(election.title || 'Regional Election')}</div>
                    </div>
                    <div class="gv-candidates">
                        ${(election.candidates || []).map(c => `
                            <div class="gv-candidate">
                                <div class="gv-candidate-name">${esc(c.name)}</div>
                                <div class="gv-candidate-score">${c.votes || 0} votes</div>
                                <button class="vbt-btn vbt-btn-primary gv-vote-btn" onclick="window.gvVote('${esc(c.id)}')">Vote</button>
                            </div>
                        `).join('')}
                    </div>
                `;
            } else {
                el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No active election</p>';
            }
        } catch (e) {
            el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No active election</p>';
        }
    }

    async function vote(candidate) {
        try {
            const res = await api('/api/governance/vote', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ candidate }),
            });
            if (res.success) {
                showToast('Vote cast!', 'success');
                loadElection();
            } else {
                showToast(res.error || 'Vote failed', 'error');
            }
        } catch (e) {
            showToast('Vote failed: ' + e.message, 'error');
        }
    }

    async function loadMissions() {
        const el = document.getElementById('gv-missions');
        if (!el) return;
        try {
            const res = await api('/api/justice/missions');
            if (res.success && res.missions && res.missions.length > 0) {
                el.innerHTML = res.missions.map(m => `
                    <div class="gv-mission">
                        <div class="gv-mission-info">
                            <div class="gv-mission-name">${esc(m.name)}</div>
                            <div class="gv-mission-desc">${esc(m.description)}</div>
                            <div class="gv-mission-reward">Reward: ${m.reward_micro ? fmtVBV(m.reward_micro) : 0} VBV</div>
                        </div>
                        <button class="vbt-btn vbt-btn-primary" onclick="window.gvAcceptMission('${esc(m.id)}')">Accept</button>
                    </div>
                `).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No missions available</p>';
            }
        } catch (e) {
            el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No missions available</p>';
        }
    }

    async function acceptMission(id) {
        try {
            const res = await api('/api/justice/missions/' + id + '/accept', { method: 'POST' });
            if (res.success) {
                showToast('Mission accepted!', 'success');
                loadMissions();
            } else {
                showToast(res.error || 'Failed', 'error');
            }
        } catch (e) {
            showToast('Failed: ' + e.message, 'error');
        }
    }

    async function loadLeaderboard() {
        const el = document.getElementById('gv-leaderboard');
        if (!el) return;
        try {
            const res = await api('/api/governance/leaderboard');
            if (res.success && res.leaderboard && res.leaderboard.length > 0) {
                el.innerHTML = res.leaderboard.map((entry, i) => `
                    <div class="gv-leader-entry ${i < 3 ? 'top-' + (i + 1) : ''}">
                        <span class="gv-leader-rank">#${i + 1}</span>
                        <span class="gv-leader-name">${esc(entry.name)}</span>
                        <span class="gv-leader-score">${entry.score.toLocaleString()}</span>
                    </div>
                `).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No leaderboard data</p>';
            }
        } catch (e) {
            el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No leaderboard data</p>';
        }
    }

    function showToast(msg, type) { if (window.showToast) window.showToast(msg, type); }

    window.openGovernancePanel = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        switchGvTab('weight');
    };
    window.switchGvTab = switchGvTab;
    window.gvVote = vote;
    window.gvAcceptMission = acceptMission;
})();
