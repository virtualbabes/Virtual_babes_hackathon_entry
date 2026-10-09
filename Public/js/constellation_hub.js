// ============================================================================
// Constellation Hub (Living Nexus) — Contained overlay centerpiece
// ----------------------------------------------------------------------------
// Star-cluster node progression map. Nodes read /api/player/progression.
// Dormant (locked, dim) → Dawning (pulsing, half-lit) → Alive (unlocked, breathing glow)
// Connection lines light up when both connected nodes are Alive.
// Clicking an Alive node opens that panel as an overlay on top of the hub.
// ============================================================================

(function () {
    'use strict';
    var API_BASE = '/api';
    let overlayEl = null;
    let canvasEl = null;
    let ctx = null;
    let particles = [];
    let particlesEnabled = true;
    let animationId = null;
    let gameData = null;
    let zoomLevel = 1.0;

    // Node definitions: position (%), icon, label, feature key, unlock gate
    const NODES = []; // Dynamically populated from starred items

    function init() {
        if (overlayEl) return;
        overlayEl = document.getElementById('constellation-overlay');
        if (!overlayEl) return;
        overlayEl.className = 'overlay constellation-overlay';
        overlayEl.style.display = 'none';

        overlayEl.innerHTML = `
            <div class="constellation-shell">
                <div class="bg-layer bg-stars"></div>
                <div class="bg-layer bg-nebula"></div>
                <canvas id="constellation-particle-canvas"></canvas>

                <div class="constellation-top-bar">
                    <div class="top-bar-left">
                        <div class="wallet-display" id="constellation-wallet">0x----...----</div>
                        <div class="theme-indicator">
                            <div class="theme-dot"></div>
                            <span id="constellation-theme-text">ORDER • AETHER • INT:72 • ENT:35</span>
                        </div>
                    </div>
                    <div class="top-bar-right">
                        <div class="vbx-balance-widget">
                            <span class="vbx-label">VBV</span>
                            <span class="vbx-amount" id="constellation-vbv">0.00</span>
                        </div>
                        <div class="theme-indicator"><span id="constellation-climate">CLIMATE: BREEZY</span></div>
                    </div>
                </div>

                <div class="constellation-container">
                    <div class="constellation-canvas" id="constellation-canvas">
                        <svg class="constellation-lines" id="constellation-lines" viewBox="0 0 100 100" preserveAspectRatio="none"></svg>
                        <div id="constellation-nodes"></div>
                    </div>
                </div>

                <div class="ch-empty hidden" id="ch-empty">
                    <div class="ch-empty-icon">🌌</div>
                    <h3>No favorites yet</h3>
                    <p>Open the World Dashboard and tap the ☆ star next to any tab to add it here.</p>
                </div>

                <div class="constellation-bottom-controls">
                    <button class="control-btn" onclick="window.constellationToggleParticles()">✨ Particles</button>
                    <button class="control-btn" onclick="window.constellationReset()">🔄 Reset</button>
                </div>

                <div class="constellation-zoom-controls">
                    <button class="zoom-btn" onclick="window.constellationZoom(0.1)">+</button>
                    <button class="zoom-btn" onclick="window.constellationZoom(-0.1)">−</button>
                    <button class="zoom-btn" onclick="window.constellationResetZoom()">⟲</button>
                </div>

                <button class="vbt-btn vbt-btn-secondary constellation-close-btn" onclick="window.closeConstellationHub()">✕ Close</button>
            </div>`;

        canvasEl = document.getElementById('constellation-particle-canvas');
        ctx = canvasEl.getContext('2d');
        resizeCanvas();
        window.addEventListener('resize', resizeCanvas);

        renderNodes();
        fetchProgression();
        startParticles();

        // Pause the particle rAF when the browser tab is backgrounded (no wasted CPU/GPU).
        if (!window.__hubVisibilityBound) {
          window.__hubVisibilityBound = true;
          document.addEventListener('visibilitychange', () => {
            if (document.hidden) {
              if (animationId) { cancelAnimationFrame(animationId); animationId = null; }
            } else if (particlesEnabled && overlayEl && overlayEl.offsetParent !== null) {
              if (!animationId) animateParticles();
            }
          });
        }

        window.constellationZoom = zoom;
        window.constellationResetZoom = resetZoom;
        window.constellationToggleParticles = toggleParticles;
        window.constellationReset = resetDemo;
        window.renderStarredItems = renderNodes;
        window.renderNodes = renderNodes;
    }

    function resizeCanvas() {
        if (!canvasEl) return;
        canvasEl.width = window.innerWidth;
        canvasEl.height = window.innerHeight;
    }

    function renderNodes() {
        const container = document.getElementById('constellation-nodes');
        const svgContainer = document.getElementById('constellation-lines');
        const emptyEl = document.getElementById('ch-empty');
        if (!container || !svgContainer) return;

        // Get starred items from UserPreferences
        let starred = [];
        if (window.UserPreferences) {
            starred = window.UserPreferences.getStarred();
        }

        // Get pathway data
        let pathwayId = 'P-Shadow'; // default
        let pathwayTier = 1;
        if (window.PathwayAvenues && gameData && gameData.pathway) {
            pathwayId = gameData.pathway;
            pathwayTier = window.PathwayAvenues.calculateTier(pathwayId, gameData.xp || 0, gameData.achievements || 0);
        }

        if (starred.length === 0) {
            container.innerHTML = '';
            svgContainer.innerHTML = '';
            if (emptyEl) emptyEl.classList.remove('hidden');
            renderPathwayAvenue(pathwayId, pathwayTier);
            return;
        }

        if (emptyEl) emptyEl.classList.add('hidden');

        // Render starred items as nodes
        const cols = Math.min(starred.length, 6);
        const rows = Math.ceil(starred.length / cols);
        
        container.innerHTML = starred.map((item, i) => {
            const col = i % cols;
            const row = Math.floor(i / cols);
            const x = 10 + (col * (80 / (cols - 1 || 1)));
            const y = 15 + (row * (70 / (rows - 1 || 1)));
            
            const icon = item.icon || '⭐';
            const label = item.label || item.wdTab;
            const color = item.color || '#ffd700';
            
            return `
                <div class="feature-node node-alive ch-starred-node" 
                     data-feature="${item.wdTab}" 
                     data-starred-idx="${i}"
                     style="left: ${x}%; top: ${y}%; --node-color: ${color};"
                     onclick="window.constellationOpenPanel('${item.wdTab}')">
                    <div class="node-core starred-core">
                        <span class="node-icon">${icon}</span>
                        <div class="animated-border"></div>
                    </div>
                    <div class="node-label">${label}</div>
                    <div class="node-tooltip">
                        <div class="tooltip-title">${label}</div>
                        <div class="tooltip-gate">★ Favorited</div>
                    </div>
                    <button class="node-unstar-btn" onclick="event.stopPropagation(); window.unstarFromConstellation('${item.wdTab}', '${item.wdSub}')">✕</button>
                </div>`;
        }).join('');

        // No connection lines for user-curated items (clean look)
        svgContainer.innerHTML = '';

        // Render pathway avenue
        renderPathwayAvenue(pathwayId, pathwayTier);
    }

    function renderPathwayAvenue(pathwayId, tier) {
        const container = document.getElementById('constellation-nodes');
        if (!container || !window.PathwayAvenues) return;

        const pathway = window.PathwayAvenues.getPathway(pathwayId);
        if (!pathway) return;

        // Add pathway avenue nodes at the bottom
        const avenueEl = document.createElement('div');
        avenueEl.id = 'pathway-avenue';
        avenueEl.className = 'pathway-avenue';
        avenueEl.innerHTML = `
            <div class="pathway-header" style="--pathway-primary: ${pathway.colors.primary}; --pathway-glow: ${pathway.colors.glow};">
                <span class="pathway-icon">${pathway.icon}</span>
                <span class="pathway-name">${pathway.name}</span>
                <span class="pathway-tier">Tier ${tier}</span>
            </div>
            <div class="pathway-tiers">
                ${pathway.tiers.map((t, i) => `
                    <div class="pathway-tier-node ${i < tier ? 'tier-unlocked' : 'tier-locked'} ${i === tier - 1 ? 'tier-current' : ''}" 
                         style="--pathway-primary: ${pathway.colors.primary}; --pathway-glow: ${pathway.colors.glow};">
                        <div class="tier-dot">${i < tier ? '✓' : (i + 1)}</div>
                        <div class="tier-name">${t.name}</div>
                    </div>
                `).join('')}
            </div>
        `;

        // Remove existing avenue if present
        const existing = document.getElementById('pathway-avenue');
        if (existing) existing.remove();

        container.appendChild(avenueEl);
    }

    async function fetchProgression() {
            try {
                const resp = await (window.api ? window.api(API_BASE + '/player/progression') : fetch(API_BASE + '/player/progression'));
                if (!resp.ok) throw new Error('Progression endpoint unavailable');
                gameData = await resp.json();
                applyLiveData();
            } catch (e) {
                console.warn('Constellation: API unreachable, empty state:', e.message);
                gameData = null;
                applyLiveData(); // renders empty states via data.state || 'dormant'
            }
        }

    function applyLiveData() {
        if (!gameData || !gameData.features) return;
        
        const states = {};
        for (const [key, data] of Object.entries(gameData.features)) {
            states[key] = data.state || 'dormant';
        }
        // NPCs always alive except crypto-seraph
        states['npc-anya'] = 'alive';
        states['npc-vbabes'] = 'alive';
        states['npc-crypto-seraph'] = 'dormant';
        
        updateNodeStates(states);

        // Update wallet display
        const walletEl = document.getElementById('constellation-wallet');
        if (walletEl && gameData.wallet) {
            const w = gameData.wallet;
            walletEl.textContent = w.slice(0, 6) + '...' + w.slice(-4);
        }

        // Update VBV balance
        const vbvEl = document.getElementById('constellation-vbv');
        if (vbvEl && gameData.vbv_balance !== undefined) {
            vbvEl.textContent = (parseFloat(gameData.vbv_balance) / 1000000).toFixed(2);
        }
    }

    function updateNodeStates(states) {
        NODES.forEach(node => {
            const state = states[node.id] || 'dormant';
            const el = document.querySelector(`[data-feature="${node.id}"]`);
            if (!el) return;
            
            el.className = `feature-node node-${state} ${node.npc ? 'node-npc-' + node.id.replace('npc-', '') : ''}`;
            
            // Update tooltip
            const tooltip = el.querySelector('.node-tooltip');
            if (tooltip) {
                const fill = tooltip.querySelector('.tooltip-progress-fill');
                const text = tooltip.querySelector('.tooltip-progress-text');
                if (fill && text) {
                    if (state === 'alive') { fill.style.width = '100%'; text.textContent = 'Unlocked'; }
                    else if (state === 'dawning') { fill.style.width = '50%'; text.textContent = 'In progress'; }
                    else { fill.style.width = '0%'; text.textContent = node.gate; }
                }
            }
        });

        // Update connection lines
        CONNECTIONS.forEach(([fromId, toId]) => {
            const line = document.querySelector(`[data-from="${fromId}"][data-to="${toId}"]`);
            if (!line) return;
            const fromAlive = states[fromId] === 'alive';
            const toAlive = states[toId] === 'alive';
            line.classList.toggle('path-active', fromAlive && toAlive);
        });
    }

    function openPanel(featureId) {
        const panelMap = {
            shops: 'openShopsOverlay',
            career: null,
            clubs: 'openClubFoundry',
            territory: 'openTerritoryMapOverlay',
            governor: null,
            rivalry: 'openRivalryViewer',
            faith: 'openFaithChurch',
            stats: 'openStatOverlay',
            regional: null,
            tournament: 'openTournamentBracket',
            entity: 'openEntityMarket',
            children: 'openChildrenBots',
            creator: 'openCreatorStore',
            // The publisher is `window.openGovernancePanel` (governance.js). This once read
            // `openGovernance`, a name that exists NOWHERE, so the guarded dispatcher below
            // silently did nothing — a dead button with no error surface.
            governance: 'openGovernancePanel',
            leasing: 'openInfrastructureLease',
            identity: 'openPersistentIdentity',
        };
        const fn = panelMap[featureId];
        if (fn && typeof window[fn] === 'function') {
            window[fn]();
        }
    }

    // === PARTICLES ===
    class Particle {
        constructor() { this.reset(); }
        reset() {
            this.x = Math.random() * canvasEl.width;
            this.y = Math.random() * canvasEl.height;
            this.size = Math.random() * 2 + 0.5;
            this.speedX = (Math.random() - 0.5) * 0.5;
            this.speedY = (Math.random() - 0.5) * 0.5;
            this.opacity = Math.random() * 0.5 + 0.1;
            this.life = Math.random() * 200 + 100;
        }
        update() {
            this.x += this.speedX;
            this.y += this.speedY;
            this.life--;
            if (this.life <= 0 || this.x < 0 || this.x > canvasEl.width || this.y < 0 || this.y > canvasEl.height) this.reset();
        }
        draw() {
            ctx.beginPath();
            ctx.arc(this.x, this.y, this.size, 0, Math.PI * 2);
            ctx.fillStyle = `rgba(0, 242, 254, ${this.opacity * (this.life / 200)})`;
            ctx.fill();
        }
    }

    function startParticles() {
        particles = [];
        // FPS cap: limit particles on low-end devices
        const isMobile = /Android|iPhone|iPad|iPod/i.test(navigator.userAgent);
        const maxParticles = isMobile ? 20 : 60;
        for (let i = 0; i < maxParticles; i++) particles.push(new Particle());
        animateParticles();
    }

    function animateParticles() {
        if (!particlesEnabled) { animationId = null; return; }
        // Pause the RAF loop entirely while the hub is hidden (e.g. another overlay is
        // open on top of it). We never burn CPU/compositing repainting an invisible canvas.
        if (!overlayEl || overlayEl.offsetParent === null) { animationId = null; return; }
        ctx.clearRect(0, 0, canvasEl.width, canvasEl.height);
        particles.forEach(p => { p.update(); p.draw(); });
        animationId = requestAnimationFrame(animateParticles);
    }

    function toggleParticles() {
        particlesEnabled = !particlesEnabled;
        if (particlesEnabled) animateParticles();
        else if (animationId) cancelAnimationFrame(animationId);
    }

    function zoom(delta) {
        zoomLevel = Math.max(0.4, Math.min(2.0, zoomLevel + delta));
        const canvas = document.getElementById('constellation-canvas');
        if (canvas) canvas.style.transform = `scale(${zoomLevel})`;
    }

    function resetZoom() {
        zoomLevel = 1.0;
        const canvas = document.getElementById('constellation-canvas');
        if (canvas) canvas.style.transform = 'scale(1)';
    }

    function simulateUnlock() {
        const dormant = NODES.filter(n => {
            const el = document.querySelector(`[data-feature="${n.id}"]`);
            return el && el.classList.contains('node-dormant');
        });
        if (dormant.length === 0) return;
        const random = dormant[Math.floor(Math.random() * dormant?.length ?? 0)];
        const el = document.querySelector(`[data-feature="${random.id}"]`);
        if (el) {
            el.className = `feature-node node-alive ${random.npc ? 'node-npc-' + random.id.replace('npc-', '') : ''}`;
            const fill = el.querySelector('.tooltip-progress-fill');
            const text = el.querySelector('.tooltip-progress-text');
            if (fill && text) { fill.style.width = '100%'; text.textContent = 'Unlocked'; }
        }
    }

    function resetDemo() {
        renderNodes();
        if (gameData) applyLiveData();
        else {
            // Empty state — no mock data
            gameData = null;
            applyLiveData();
        }
    }

    window.openConstellationHub = function () {
        init();
        renderNodes();
        overlayEl.style.display = 'flex';
        // Hide the now app.js-rendered static game screen behind the constellation hub.
        document.body.classList.add('nexus-lobby-active');
        // Resume the particle animation if it was paused while hidden.
        if (particlesEnabled && !animationId) animateParticles();
    };

    window.refreshConstellationHub = function () {
        renderNodes();
    };

    // ------------------------------------------------------------------
    // Hub close + 3D-world entry.
    // These were previously an inline <script> block at the bottom of
    // index.html (where they were unreachable — the block's opening tag had
    // been destroyed, so the code rendered as literal text). index.html is a
    // pure shell now; the hub owns its own open/close contract.
    // NOTE: the old inline copy re-declared `function openConstellationHub()`
    // and then called `window.openConstellationHub()` from inside itself,
    // which is infinite recursion. It is deliberately NOT restored.
    // ------------------------------------------------------------------
    window.closeConstellationHub = function () {
        if (overlayEl) overlayEl.style.display = 'none';
        document.body.classList.remove('nexus-lobby-active');
    };

    // Direct navigation to the §25 Web-3D world (no popup-blocker issues).
    window.open3DWorld = function () {
        window.location.href = '/world';
    };

    window.unstarFromConstellation = function (wdTab, wdSub) {
        if (window.UserPreferences) {
            window.UserPreferences.unstar(wdTab, wdSub);
            renderNodes();
        }
    };

    window.constellationOpenPanel = openPanel;

    // Auto-init on script load if overlay exists in DOM
    if (document.getElementById('constellation-overlay')) {
        init();
    }
})();
