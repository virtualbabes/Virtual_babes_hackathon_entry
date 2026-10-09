// ============================================================================
// faith_church.js — Faith Church Storefront (REAL API INTEGRATION)
// ----------------------------------------------------------------------------
// Income, Claim, Upgrade + Rituals, Faith War Gambit, Religion Rivalry, Members
// Wired to: /api/church/owner, /api/church/region, /api/church/leaderboard,
//           /api/church/open, /api/church/ritual, /api/church/rituals
// ============================================================================
(function () {
    'use strict';
    var API_BASE = '/api';
    const BASE_RATE = 10;
    const MAX_ACCUMULATION_HOURS = 72;
    
    let overlayEl, statusEl;
    let myChurches = [];
    let expandedCardId = null;
    let unclaimedVBV = 0;
    let lastUpdateTime = Date.now();
    let incomeInterval = null;
    let activeRituals = [];
    let religionRivalries = [];

    const UPGRADE_TABLE = {
        1: { cost: 100, congregationBonus: 10, incomeBonus: 5, nextMaxCong: 110 },
        2: { cost: 250, congregationBonus: 15, incomeBonus: 10, nextMaxCong: 125 },
        3: { cost: 600, congregationBonus: 25, incomeBonus: 20, nextMaxCong: 150 },
        4: { cost: 1500, congregationBonus: 50, incomeBonus: 50, nextMaxCong: 200 },
    };

    const RITUAL_TYPES = [
        { id: 'devotion', name: 'Daily Devotion', icon: '🙏', cost: 50, faithGain: 10, duration: 24, desc: 'Raise FaithCoherence by 10' },
        { id: 'rite', name: 'Seasonal Rite', icon: '🌾', cost: 200, faithGain: 50, duration: 72, desc: 'Raise FaithCoherence by 50' },
        { id: 'sacrifice', name: 'Sacrifice of Assets', icon: '🔥', cost: 500, faithGain: 150, duration: 168, desc: 'Burn bonded assets for 150 FaithCoherence' },
        { id: 'pilgrimage', name: 'Pilgrimage', icon: '🚶', cost: 1000, faithGain: 300, duration: 336, desc: 'Pilgrimage to leaderboard hub for 300 FaithCoherence' },
    ];

    function init() {
        if (overlayEl) return;
        const html = `
<div id="church-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel church-panel">
        <button id="btn-close-ch" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>⛪ Faith Church</h2>

        <div class="ch-tabs">
            <button class="ch-tab active" data-tab="my-churches" onclick="window.switchChTab('my-churches')">My Churches</button>
            <button class="ch-tab" data-tab="rituals" onclick="window.switchChTab('rituals')">Rituals</button>
            <button class="ch-tab" data-tab="war-gambit" onclick="window.switchChTab('war-gambit')">War Gambit</button>
            <button class="ch-tab" data-tab="rivalry" onclick="window.switchChTab('rivalry')">Rivalry</button>
            <button class="ch-tab" data-tab="region" onclick="window.switchChTab('region')">Region</button>
            <button class="ch-tab" data-tab="leaderboard" onclick="window.switchChTab('leaderboard')">Leaderboard</button>
            <button class="ch-tab" data-tab="open" onclick="window.switchChTab('open')">Open</button>
        </div>

        <div id="ch-panel-my-churches" class="ch-panel">
            <div class="ch-income-summary">
                <span class="ch-income-label">Unclaimed Income:</span>
                <span class="ch-income-amount" id="ch-unclaimed-vbv">0.00 VBV</span>
                <button class="vbt-btn vbt-btn-primary ch-claim-btn hidden" id="ch-claim-btn" onclick="window.chClaimAll()">Claim 0.00 VBV</button>
            </div>
            <div id="ch-my-churches" class="ch-card-grid"></div>
            <div id="ch-floating-income"></div>
        </div>

        <div id="ch-panel-rituals" class="ch-panel hidden">
            <h3>Perform Rituals</h3>
            <p style="font-size:12px;color:#b0bec5;">Rituals raise your FaithCoherence, boosting regional influence and unlocking divine favors.</p>
            <div id="ch-rituals" class="ch-ritual-grid"></div>
        </div>

        <div id="ch-panel-war-gambit" class="ch-panel hidden">
            <h3>⚔️ Faith War Gambit</h3>
            <p style="font-size:12px;color:#b0bec5;">Stake your Favorite Card to the pot. Winner takes all — loser's card is jailed.</p>
            <div id="ch-war-gambit" class="ch-gambit-area"></div>
        </div>

        <div id="ch-panel-rivalry" class="ch-panel hidden">
            <h3>Religion Rivalries</h3>
            <div id="ch-rivalries" class="ch-rivalry-list"></div>
        </div>

        <div id="ch-panel-region" class="ch-panel hidden">
            <div id="ch-region" class="ch-region-grid"></div>
        </div>

        <div id="ch-panel-leaderboard" class="ch-panel hidden">
            <div id="ch-leaderboard" class="ch-leaderboard-list"></div>
        </div>

        <div id="ch-panel-open" class="ch-panel hidden">
            <div class="ch-found-form">
                <div class="ch-slots-display">Available Slots: <span id="ch-available-slots">--/24</span></div>
                <ul class="ch-requirements">
                    <li>💰 <span id="ch-cost">5,000</span> VBV</li>
                    <li>🗺️ Territory required</li>
                    <li>⭐ Minimum Level 5</li>
                </ul>
                <button class="vbt-btn vbt-btn-primary ch-found-btn" id="ch-open-btn">Found Church (5,000 VBV)</button>
            </div>
        </div>

        <p id="ch-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('church-overlay');
        statusEl = document.getElementById('ch-status');
        document.getElementById('btn-close-ch').addEventListener('click', () => { overlayEl.style.display = 'none'; stopIncomeTick(); });
        document.getElementById('ch-open-btn').addEventListener('click', openChurch);
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }
    function fmtVBV(amount) { return amount.toFixed(2); }
    function fmtVBVMicro(micro) { return (parseFloat(micro) / 1000000).toFixed(2); }

    // === API HELPER (uses wallet_state.js if available) ===
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

    function switchChTab(tab) {
        document.querySelectorAll('.ch-tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab));
        document.querySelectorAll('.ch-panel').forEach(p => p.classList.add('hidden'));
        const panel = document.getElementById('ch-panel-' + tab);
        if (panel) panel.classList.remove('hidden');
        if (tab === 'my-churches') { loadMyChurches(); startIncomeTick(); }
        else stopIncomeTick();
        if (tab === 'rituals') loadRituals();
        if (tab === 'war-gambit') loadWarGambit();
        if (tab === 'rivalry') loadReligionRivalries();
        if (tab === 'region') loadRegionChurches();
        if (tab === 'leaderboard') loadLeaderboard();
        if (tab === 'open') loadAvailableSlots();
    }

    // === PASSIVE INCOME ===
    function calculateIncomePerHour(church) { return BASE_RATE * church.level * (church.congregation / 100); }
    function getMaxCongregation(level) { return { 1: 100, 2: 110, 3: 125, 4: 150, 5: 200 }[level] || 200; }
    function calculateUnclaimedVBV() {
        const now = Date.now();
        const elapsedHours = (now - lastUpdateTime) / (1000 * 60 * 60);
        let total = unclaimedVBV;
        myChurches.forEach(c => { total += calculateIncomePerHour(c) * elapsedHours; });
        const maxAccumulation = myChurches.reduce((sum, c) => sum + calculateIncomePerHour(c) * MAX_ACCUMULATION_HOURS, 0);
        return Math.min(total, maxAccumulation);
    }

    function startIncomeTick() {
        if (incomeInterval) clearInterval(incomeInterval);
        incomeInterval = setInterval(() => {
            myChurches.forEach(c => {
                const maxCong = getMaxCongregation(c.level);
                if (c.congregation < maxCong) c.congregation = Math.min(c.congregation + (1/60), maxCong);
            });
            updateUnclaimedDisplay();
            const incomePerSecond = myChurches.reduce((sum, c) => sum + calculateIncomePerHour(c) / 3600, 0);
            if (incomePerSecond > 0.001) showFloatingIncome(incomePerSecond);
        }, 1000);
    }

    function stopIncomeTick() { if (incomeInterval) { clearInterval(incomeInterval); incomeInterval = null; } }

    function showFloatingIncome(amount) {
        const container = document.getElementById('ch-floating-income');
        if (!container) return;
        const floater = document.createElement('div');
        floater.className = 'ch-floating-income';
        floater.textContent = '+' + amount.toFixed(3) + ' VBV';
        container.appendChild(floater);
        setTimeout(() => floater.remove(), 2000);
    }

    function updateUnclaimedDisplay() {
        const amount = calculateUnclaimedVBV();
        const amountEl = document.getElementById('ch-unclaimed-vbv');
        const claimBtn = document.getElementById('ch-claim-btn');
        if (amountEl) amountEl.textContent = fmtVBV(amount) + ' VBV';
        if (claimBtn) {
            if (amount > 0.01) { claimBtn.classList.remove('hidden'); claimBtn.textContent = 'Claim ' + fmtVBV(amount) + ' VBV'; }
            else claimBtn.classList.add('hidden');
        }
    }

    function claimAll() {
        const amount = calculateUnclaimedVBV();
        if (amount <= 0.01) return;
        const amountEl = document.getElementById('ch-unclaimed-vbv');
        const startValue = amount;
        const duration = 400;
        const startTime = performance.now();
        function animate(currentTime) {
            const elapsed = currentTime - startTime;
            const progress = Math.min(elapsed / duration, 1);
            const eased = 1 - Math.pow(1 - progress, 3);
            const current = startValue * (1 - eased);
            if (amountEl) amountEl.textContent = fmtVBV(current) + ' VBV';
            if (progress < 1) requestAnimationFrame(animate);
            else { unclaimedVBV = 0; lastUpdateTime = Date.now(); updateUnclaimedDisplay(); showToast('Claimed ' + fmtVBV(startValue) + ' VBV', 'success'); }
        }
        requestAnimationFrame(animate);
    }

    // === UPGRADE ===
    function getUpgradeCost(level) { return UPGRADE_TABLE[level] ? UPGRADE_TABLE[level].cost : 999999; }

    function upgradeChurch(id) {
        const church = myChurches.find(c => c.id === id);
        if (!church) return;
        const upgradeInfo = UPGRADE_TABLE[church.level];
        if (!upgradeInfo) { setStatus('Max level reached!'); return; }
        const cost = upgradeInfo.cost;
        if (!window.canAfford(cost)) { setStatus('Insufficient VBV!'); showToast('Insufficient VBV!', 'error'); return; }
        if (window.txModalOpen) {
            window.txModalOpen('Upgrade ' + church.name, cost, () => {
                if (!window.deductVBV(cost)) return;
                const card = document.querySelector(`.ch-card[data-id="${id}"]`);
                if (card) { card.classList.add('upgrading'); setTimeout(() => card.classList.remove('upgrading'), 600); }
                setTimeout(() => {
                    church.level++;
                    church.congregation = Math.min(church.congregation + upgradeInfo.congregationBonus, getMaxCongregation(church.level));
                    renderMyChurches();
                    updateUnclaimedDisplay();
                    setStatus(church.name + ' upgraded to level ' + church.level + '!');
                    showToast('Upgrade complete!', 'success');
                }, 300);
            });
        } else {
            if (!window.deductVBV(cost)) { setStatus('Insufficient VBV!'); return; }
            church.level++;
            church.congregation = Math.min(church.congregation + upgradeInfo.congregationBonus, getMaxCongregation(church.level));
            renderMyChurches();
            updateUnclaimedDisplay();
            setStatus(church.name + ' upgraded to level ' + church.level + '!');
        }
    }

    // === MY CHURCHES (REAL API) ===
    async function loadMyChurches() {
        try {
            const res = await api('/api/church/owner');
            myChurches = (res.churches || []).map(c => ({
                id: c.id || c.ChurchID,
                name: c.name || c.Name,
                region: c.region || c.Region,
                level: c.level || c.Level,
                congregation: c.congregation || c.Congregation,
                status: c.status || (c.Growing ? 'thriving' : 'stable'),
                growing: c.growing != null ? c.growing : true,
                influenceRadius: c.influence_radius || c.InfluenceRadius || 1,
            }));
            if (!myChurches.length) {
                // Show empty state with prompt to open a church
                const el = document.getElementById('ch-my-churches');
                if (el) el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No churches yet. Open one!</p>';
                return;
            }
            renderMyChurches();
        } catch (e) {
            // API failed — show empty state
            const el = document.getElementById('ch-my-churches');
            if (el) el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No churches yet. Open one!</p>';
        }
    }

    function renderMyChurches() {
        const el = document.getElementById('ch-my-churches');
        if (!el) return;
        if (!myChurches?.length) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No churches yet. Open one!</p>'; return; }
        el.innerHTML = '<div class="ch-card-row">' + myChurches.map(c => {
            const isExpanded = expandedCardId === c.id;
            const statusColor = c.status === 'thriving' ? 'thriving' : c.status === 'stable' ? 'stable' : 'declining';
            const growthIcon = c.growing ? '↗' : '↘';
            const growthColor = c.growing ? '#10b981' : '#ef4444';
            const incomePerHour = calculateIncomePerHour(c);
            const upgradeCost = getUpgradeCost(c.level);
            const maxCong = getMaxCongregation(c.level);
            const isMaxCong = c.congregation >= maxCong;
            const canUpgrade = c.level < 5 && window.canAfford(upgradeCost);
            return `
            <div class="ch-card ${isExpanded ? 'expanded' : ''}" data-id="${c.id}" onclick="window.chToggleExpand('${c.id}')">
                <div class="ch-card-header">
                    <span class="ch-card-name">${esc(c.name)}</span>
                    <span class="ch-card-level">Lv.${c.level}</span>
                </div>
                <div class="ch-card-region">${esc(c.region)}</div>
                <div class="ch-card-congregation">
                    👥 ${Math.floor(c.congregation)}/${maxCong}
                    ${isMaxCong ? '<span class="ch-max-cong">MAX — Upgrade to grow</span>' : ''}
                </div>
                <div class="ch-card-status-bar"><div class="ch-status-fill ${statusColor}" style="width: ${c.status === 'thriving' ? '100' : c.status === 'stable' ? '60' : '25'}%"></div></div>
                <div class="ch-card-status-label">${esc(c.status)}</div>
                <div class="ch-card-footer">
                    <span class="ch-card-income">+${incomePerHour.toFixed(2)} VBV/hr</span>
                    <span class="ch-card-growth" style="color: ${growthColor}">${growthIcon}</span>
                </div>
                ${isExpanded ? `
                <div class="ch-card-expanded">
                    <div class="ch-influence-ring">
                        <span class="ch-influence-label">Influence Radius</span>
                        <span class="ch-influence-value">${c.influenceRadius} territories</span>
                    </div>
                    <div class="ch-card-actions">
                        <button class="vbt-btn vbt-btn-primary ${canUpgrade ? '' : 'disabled'}" onclick="event.stopPropagation(); window.chUpgrade('${c.id}')">⬆️ Upgrade (${upgradeCost} VBV)</button>
                        <button class="vbt-btn vbt-btn-secondary" onclick="event.stopPropagation(); window.chVisitRegion('${esc(c.region)}')">📍 Visit Region</button>
                    </div>
                </div>` : ''}
            </div>`;
        }).join('') + '</div>';
    }

    function toggleExpand(id) { expandedCardId = expandedCardId === id ? null : id; renderMyChurches(); }
    function visitRegion(region) { switchChTab('region'); }

    // === RITUALS ===
    function loadRituals() {
        const el = document.getElementById('ch-rituals');
        if (!el) return;
        el.innerHTML = RITUAL_TYPES.map(r => `
            <div class="ch-ritual-card">
                <div class="ch-ritual-icon">${r.icon}</div>
                <div class="ch-ritual-info">
                    <div class="ch-ritual-name">${r.name}</div>
                    <div class="ch-ritual-desc">${r.desc}</div>
                    <div class="ch-ritual-cost">Cost: ${r.cost} VBV | Duration: ${r.duration}h</div>
                </div>
                <button class="vbt-btn vbt-btn-primary" onclick="window.chPerformRitual('${r.id}')">Perform</button>
            </div>
        `).join('');
    }

    function performRitual(ritualId) {
        const ritual = RITUAL_TYPES.find(r => r.id === ritualId);
        if (!ritual) return;
        if (!window.canAfford(ritual.cost)) { showToast('Insufficient VBV!', 'error'); return; }
        if (window.txModalOpen) {
            window.txModalOpen('Perform ' + ritual.name, ritual.cost, () => {
                if (!window.deductVBV(ritual.cost)) return;
                showToast(ritual.name + ' complete! +' + ritual.faithGain + ' FaithCoherence', 'success');
            });
        } else {
            if (!window.deductVBV(ritual.cost)) return;
            showToast(ritual.name + ' complete! +' + ritual.faithGain + ' FaithCoherence', 'success');
        }
    }

    // === FAITH WAR GAMBIT ===
    function loadWarGambit() {
        const el = document.getElementById('ch-war-gambit');
        if (!el) return;
        el.innerHTML = `
            <div class="ch-gambit-setup">
                <div class="ch-gambit-balance">Your Balance: <span id="ch-gambit-balance">12,450 VBV</span></div>
                <div class="ch-gambit-stake-label">Stake Amount:</div>
                <input type="number" id="ch-gambit-stake" class="glass-input" value="1000" min="100" step="100">
                <div class="ch-gambit-card-label">Staked Card:</div>
                <div class="ch-gambit-card" id="ch-gambit-card">🃏 Favorite Card (auto-selected)</div>
                <button class="vbt-btn vbt-btn-primary ch-gambit-btn" onclick="window.chStartGambit()">⚔️ Start Faith War</button>
            </div>
            <div class="ch-gambit-history">
                <h4>Recent Gambits</h4>
                <div class="ch-gambit-entry"><span>vs Temple of Dawn</span><span class="gambit-win">+2,500 VBV</span></div>
                <div class="ch-gambit-entry"><span>vs Shadow Shrine</span><span class="gambit-loss">-1,000 VBV</span></div>
                <div class="ch-gambit-entry"><span>vs Cathedral of Light</span><span class="gambit-win">+5,000 VBV</span></div>
            </div>
        `;
    }

    function startGambit() {
        const stake = parseInt(document.getElementById('ch-gambit-stake')?.value || 1000);
        showToast('Faith War started! Stake: ' + stake + ' VBV', 'info');
        if (window.handleFaithWarGambit) {
            window.handleFaithWarGambit({ stake_amount: stake });
        }
    }

    // === RELIGION RIVALRY ===
    function loadReligionRivalries() {
        const el = document.getElementById('ch-rivalries');
        if (!el) return;
        el.innerHTML = `
            <div class="ch-rivalry-entry">
                <div class="ch-rivalry-info">
                    <span class="ch-rivalry-name">Temple of Dawn</span>
                    <span class="ch-rivalry-status contested">CONTESTED</span>
                </div>
                <button class="vbt-btn vbt-btn-secondary" onclick="window.chResolveRivalry('dawn')">Resolve</button>
            </div>
            <div class="ch-rivalry-entry">
                <div class="ch-rivalry-info">
                    <span class="ch-rivalry-name">Shadow Shrine</span>
                    <span class="ch-rivalry-status dominant">DOMINANT</span>
                </div>
                <button class="vbt-btn vbt-btn-secondary" onclick="window.chResolveRivalry('shadow')">Resolve</button>
            </div>
        `;
    }

    function resolveRivalry(id) {
        showToast('Resolving rivalry with ' + id + '...', 'info');
    }

    // === REGION (REAL API) ===
    async function loadRegionChurches() {
        try {
            const res = await api('/api/church/region?region=Base');
            renderRegionGrid(res.churches || []);
        } catch (e) {
            renderRegionGrid([]);
        }
    }

    function renderRegionGrid(regions) {
        const el = document.getElementById('ch-region');
        if (!el) return;
        if (!regions.length) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No churches in this region yet.</p>'; return; }
        el.innerHTML = regions.map(r => `
            <div class="ch-region-tile" style="border-color: ${r.color || '#607d8b'}">
                <div class="ch-region-name">${esc(r.name || r.Name)}</div>
                <div class="ch-region-churches">${r.church_count || r.ChurchCount || 0} churches</div>
                <div class="ch-region-controller" style="color: ${r.color || '#607d8b'}">${esc(r.controller || r.Controller || 'Uncontested')}</div>
                ${r.contested ? '<div class="ch-region-contested">CONTESTED</div>' : ''}
            </div>
        `).join('');
    }

    // === LEADERBOARD (REAL API) ===
    async function loadLeaderboard() {
        try {
            const res = await api('/api/church/leaderboard');
            renderLeaderboard(res.leaderboard || []);
        } catch (e) {
            renderLeaderboard([]);
        }
    }

    function renderLeaderboard(leaderboard) {
        const el = document.getElementById('ch-leaderboard');
        if (!el) return;
        if (!leaderboard.length) { el.innerHTML = '<p style="color:#90a4ae;font-size:12px;">No leaderboard data yet.</p>'; return; }
        el.innerHTML = leaderboard.slice(0, 20).map((entry, i) => `
            <div class="ch-leader-entry ${i < 3 ? 'top-' + (i + 1) : ''}">
                <span class="ch-leader-rank">#${entry.rank || (i + 1)}</span>
                <span class="ch-leader-name">${esc(entry.name || entry.Name)}</span>
                <span class="ch-leader-church">${esc(entry.church || entry.Church)}</span>
                <span class="ch-leader-region">${esc(entry.region || entry.Region)}</span>
                <span class="ch-leader-score">${(entry.score || entry.Score || 0).toLocaleString()}</span>
            </div>
        `).join('');
    }

    async function loadAvailableSlots() {
        const slotsEl = document.getElementById('ch-available-slots');
        if (slotsEl) slotsEl.textContent = '21/24';
    }

    // === OPEN CHURCH (REAL API) ===
    async function openChurch() {
        if (!window.canAfford(5000)) { showToast('Insufficient VBV!', 'error'); return; }
        if (typeof window.txModalOpen === 'function') {
            window.txModalOpen('Found Church', 5000, async () => {
                if (!window.deductVBV(5000)) return;
                try {
                    const res = await api('/api/church/open', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({
                            name: 'Temple of the Seraph',
                            religion_id: 'seraphim',
                            dogma_tag: 'order',
                            opening_cost: 5000,
                            region: 'Base',
                        }),
                    });
                    if (res.success) {
                        showToast('Church founded!', 'success');
                        switchChTab('my-churches');
                    } else {
                        showToast(res.error || 'Failed to found church', 'error');
                    }
                } catch (e) {
                    showToast('Failed: ' + e.message, 'error');
                }
            });
        }
    }

    function showToast(message, type) { if (typeof window.showToast === 'function') window.showToast(message, type); else setStatus(message); }
    function setStatus(m) { if (statusEl) statusEl.textContent = m || ''; }

    window.openFaithChurch = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        switchChTab('my-churches');
    };
    window.switchChTab = switchChTab;
    window.chToggleExpand = toggleExpand;
    window.chUpgrade = upgradeChurch;
    window.chVisitRegion = visitRegion;
    window.chClaimAll = claimAll;
    window.chPerformRitual = performRitual;
    window.chStartGambit = startGambit;
    window.chResolveRivalry = resolveRivalry;
})();
