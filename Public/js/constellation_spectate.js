// ============================================================================
// constellation_spectate.js — Multi-player constellation browser
// ----------------------------------------------------------------------------
// Constellation browser — each player is their own mini-constellation.
// See the world through someone else's progression.
// Mini SVG previews (node dots, no particles), click to expand to full view.
// Includes NUGGET/UNIT placeholders + integrated bounty system.
// ============================================================================

(function () {
    'use strict';
    var API_BASE = '/api';
    let overlayEl = null;
    let gridEl = null;
    let statusEl = null;
    let detailEl = null;
    let players = [];
    let selectedPlayer = null;

    // Feature node positions (normalized 0-100 for mini-constellation)
    const NODE_POSITIONS = [
        { id: 'shops', x: 12, y: 18 }, { id: 'career', x: 30, y: 10 },
        { id: 'clubs', x: 48, y: 18 }, { id: 'territory', x: 66, y: 10 },
        { id: 'governor', x: 84, y: 18 }, { id: 'rivalry', x: 10, y: 38 },
        { id: 'faith', x: 30, y: 38 }, { id: 'stats', x: 50, y: 38 },
        { id: 'regional', x: 70, y: 38 }, { id: 'tournament', x: 88, y: 38 },
        { id: 'entity', x: 12, y: 58 }, { id: 'children', x: 32, y: 58 },
        { id: 'creator', x: 52, y: 58 }, { id: 'governance', x: 72, y: 58 },
        { id: 'leasing', x: 22, y: 78 }, { id: 'identity', x: 45, y: 78 },
    ];

    const CONNECTIONS = [
        ['clubs', 'territory'], ['territory', 'governor'], ['governor', 'regional'],
        ['governor', 'governance'], ['territory', 'leasing'], ['rivalry', 'career'],
        ['entity', 'children'], ['faith', 'stats'], ['shops', 'creator'],
        ['clubs', 'rivalry'], ['faith', 'governance'], ['governor', 'tournament'],
    ];

    function init() {
        if (overlayEl) return;
        overlayEl = document.getElementById('constellation-spectate-overlay');
        if (!overlayEl) return;
        overlayEl.className = 'overlay constellation-spectate-overlay';
        overlayEl.style.display = 'none';

        overlayEl.innerHTML = `
            <div class="spectate-shell">
                <div class="bg-layer bg-stars"></div>
                <div class="bg-layer bg-nebula"></div>

                <div class="spectate-top-bar">
                    <div class="top-bar-left">
                        <h2>👁️ Constellation Browser</h2>
                        <span class="spectate-subtitle">See the world through someone else's progression</span>
                    </div>
                    <div class="top-bar-right">
                        <div class="vbx-balance-widget">
                            <span class="vbx-label">Online</span>
                            <span class="vbx-amount" id="spectate-online-count">0</span>
                        </div>
                        <div class="vbx-balance-widget nugget-placeholder">
                            <span class="vbx-label">$NUG</span>
                            <span class="vbx-amount">--</span>
                        </div>
                        <div class="vbx-balance-widget unit-placeholder">
                            <span class="vbx-label">$UNIT</span>
                            <span class="vbx-amount">--</span>
                        </div>
                    </div>
                </div>

                <div class="spectate-grid" id="spectate-grid"></div>

                <div class="spectate-bottom-controls">
                    <button class="control-btn" onclick="window.constellationSpectateRefresh()">🔄 Refresh</button>
                    <button class="control-btn" onclick="window.constellationSpectateOpenBounty()">🎯 Bounty Board</button>
                    <button class="control-btn primary" onclick="window.closeConstellationSpectate()">✕ Close</button>
                </div>

                <!-- Player detail overlay (read-only, dimmed) -->
                <div class="spectate-detail" id="spectate-detail" style="display:none;">
                    <div class="spectate-detail-inner">
                        <div class="spectate-detail-header">
                            <h3 id="detail-player-name">Player</h3>
                            <button class="vbt-btn vbt-btn-secondary" onclick="window.constellationSpectateCloseDetail()">✕</button>
                        </div>
                        <div class="spectate-detail-body">
                            <div class="spectate-detail-constellation" id="detail-constellation"></div>
                            <div class="spectate-detail-compare" id="detail-compare"></div>
                        </div>
                    </div>
                </div>
            </div>`;

        gridEl = document.getElementById('spectate-grid');
        statusEl = document.getElementById('spectate-online-count');
        detailEl = document.getElementById('spectate-detail');

        fetchSpectateData();
        loadTokenBalances();
    }

    async function fetchSpectateData() {
        try {
            const resp = await fetch(API_BASE + '/players/constellation');
            if (!resp.ok) throw new Error('Spectate endpoint unavailable');
            const data = await resp.json();
            players = data.players || [];
            if (statusEl) statusEl.textContent = players?.length ?? 0;
            renderGrid();
        } catch (e) {
            console.warn('Spectate: API unreachable, empty state:', e.message);
            players = [];
            if (statusEl) statusEl.textContent = '0';
            renderGrid();
        }
    }

    function loadTokenBalances() {
        // NUGGET/UNIT placeholders — these tokens require ASA IDs from env vars, so placeholder is intentional per Brendan directive.
        fetch(API_BASE + '/player/tokens').then(r => r.json()).then(data => {
            const nuggetEl = document.querySelector('.nugget-placeholder .vbx-amount');
            const unitEl = document.querySelector('.unit-placeholder .vbx-amount');
            if (nuggetEl && data.nugget_balance !== undefined) nuggetEl.textContent = data.nugget_balance;
            if (unitEl && data.unit_balance !== undefined) unitEl.textContent = data.unit_balance;
        }).catch(() => {
            // Placeholder remains "--" until backend ready
        });
    }

    function generateMiniConstellation(features) {
        // Generate SVG with node dots at fixed positions, colored by state
        let svg = '<svg viewBox="0 0 100 100" class="mini-constellation-svg">';
        
        // Connection lines (dim)
        CONNECTIONS.forEach(([fromId, toId]) => {
            const from = NODE_POSITIONS.find(n => n.id === fromId);
            const to = NODE_POSITIONS.find(n => n.id === toId);
            if (from && to) {
                const fromState = features[fromId] || 'dormant';
                const toState = features[toId] || 'dormant';
                const bothAlive = fromState === 'alive' && toState === 'alive';
                const opacity = bothAlive ? '0.6' : '0.08';
                const color = bothAlive ? '#ffd700' : 'rgba(0,242,254,1)';
                const sw = bothAlive ? '0.8' : '0.3';
                svg += '<line x1="' + from.x + '" y1="' + from.y + '" x2="' + to.x + '" y2="' + to.y + '" stroke="' + color + '" stroke-width="' + sw + '" opacity="' + opacity + '" />';
            }
        });

        // Node dots
        NODE_POSITIONS.forEach(node => {
            const state = features[node.id] || 'dormant';
            let color, radius;
            if (state === 'alive') { color = '#00f2fe'; radius = 2.5; }
            else if (state === 'dawning') { color = '#f59e0b'; radius = 2; }
            else { color = 'rgba(100,100,120,0.4)'; radius = 1.5; }
            svg += `<circle cx="${node.x}" cy="${node.y}" r="${radius}" fill="${color}" opacity="${state === 'alive' ? '1' : '0.5'}" />`;
        });

        svg += '</svg>';
        return svg;
    }

    function renderGrid() {
        if (!gridEl) return;
        if (!(players?.length ?? 0)) {
            gridEl.innerHTML = '<p style="color:#90a4ae;padding:40px;">No players online.</p>';
            return;
        }
        gridEl.innerHTML = players.map(p => {
            const features = p.features || {};
            const aliveCount = Object.values(features).filter(v => v === 'alive')?.length ?? 0;
            const dawningCount = Object.values(features).filter(v => v === 'dawning')?.length ?? 0;
            const dormantCount = Object.values(features).filter(v => v === 'dormant')?.length ?? 0;
            const miniSvg = generateMiniConstellation(features);
            
            return `
            <div class="spectate-card" onclick="window.constellationSpectateOpenPlayer('${p.wallet}')">
                <div class="spectate-card-header">
                    <span class="spectate-player-name">${escapeHtml(p?.name || p.wallet.slice(0, 8) + '…')}</span>
                    <span class="spectate-region">${escapeHtml(p.region || 'Base')}</span>
                </div>
                <div class="spectate-card-constellation">
                    ${miniSvg}
                </div>
                <div class="spectate-card-body">
                    <div class="spectate-node-summary">
                        <span class="spectate-node-alive">✨ ${aliveCount}</span>
                        <span class="spectate-node-dawning">🌅 ${dawningCount}</span>
                        <span class="spectate-node-dormant">🔒 ${dormantCount}</span>
                    </div>
                    <div class="spectate-tokens">
                        <span class="spectate-vbv">${((p.vbvBalance || 0) / 1000000).toFixed(2)} VBV</span>
                        <span class="spectate-nugget">${(p.nuggetBalance || 0).toLocaleString()} $NUG</span>
                        <span class="spectate-unit">${(p.unitBalance || 0).toLocaleString()} $UNIT</span>
                    </div>
                </div>
                ${p.bountyActive ? `<div class="spectate-bounty-badge">🎯 ${(p.bountyAmount / 1000000).toFixed(2)} VBV Bounty</div>` : ''}
            </div>`;
        }).join('');
    }

    function openPlayerDetail(wallet) {
        selectedPlayer = players.find(p => p.wallet === wallet);
        if (!selectedPlayer) return;

        const nameEl = document.getElementById('detail-player-name');
        const constEl = document.getElementById('detail-constellation');
        const compEl = document.getElementById('detail-compare');

        if (nameEl) nameEl.textContent = selectedPlayer?.name || selectedPlayer.wallet.slice(0, 12) + '…';
        if (constEl) constEl.innerHTML = generateMiniConstellation(selectedPlayer.features || {});

        // Compare view: highlight differences
        if (compEl) {
            const myFeatures = window.myPlayerFeatures || {};
            const theirFeatures = selectedPlayer.features || {};
            const diffs = [];
            for (const [key, theirState] of Object.entries(theirFeatures)) {
                const myState = myFeatures[key] || 'dormant';
                if (myState !== theirState) {
                    diffs.push(`You: ${myState}, They: ${theirState} (${key})`);
                }
            }
            compEl.innerHTML = diffs?.length ?? 0 ? diffs.map(d => `<div class="compare-diff">${escapeHtml(d)}</div>`).join('') : '<div class="compare-same">Progression matches yours</div>';
        }

        if (detailEl) detailEl.style.display = 'flex';
    }

    function closePlayerDetail() {
        if (detailEl) detailEl.style.display = 'none';
        selectedPlayer = null;
    }

    function escapeHtml(s) {
        return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
    }

    // === INTEGRATED BOUNTY SYSTEM ===
    // Replaces orphaned bounty_tracker.js — integrated into constellation spectate
    function openBountyBoard() {
        // Open the justice dashboard with bounty tab active
        if (typeof window.openJusticeDashboard === 'function') {
            window.openJusticeDashboard();
            // Switch to bounty tab after a brief delay
            setTimeout(() => {
                if (typeof window.switchJusticeTab === 'function') {
                    window.switchJusticeTab('bounties');
                }
            }, 100);
        } else {
            // Fallback: open criminality panel
            if (typeof window.openCriminality === 'function') window.openCriminality();
        }
    }

    window.openConstellationSpectate = function () {
        init();
        overlayEl.style.display = 'flex';
    };

    window.closeConstellationSpectate = function () {
        if (overlayEl) overlayEl.style.display = 'none';
        closePlayerDetail();
    };

    window.constellationSpectateRefresh = function () {
        fetchSpectateData();
    };

    window.constellationSpectateOpenPlayer = openPlayerDetail;
    window.constellationSpectateCloseDetail = closePlayerDetail;
    window.constellationSpectateOpenBounty = openBountyBoard;

    if (document.getElementById('constellation-spectate-overlay')) init();
})();
