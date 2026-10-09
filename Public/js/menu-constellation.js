// ============================================================================
// menu-constellation.js — Main screen (minimal)
// ----------------------------------------------------------------------------
// Shows only: World Dashboard + Quick Play, plus the star-pinned CATEGORY
// favorites grid. Every feature is reached through the World Dashboard
// (categories -> feature menu); no leaf feature is duplicated on the menu.
// ============================================================================

import { computePositions, getClipPath, SIZES, getShapeForSystem, isRadialGrid, isPositionedGrid } from './menu_customization.js';
import { getStarred, getMenuLayout, setMenuLayout, getDefaultShape, getDefaultSize, getGridType, getButtonOverride } from './user_preferences.js';

const TUTORIAL_VIDEOS = [
    { title: 'Endgame Blueprint', src: '/Assets/Learning-media/Endgame_Blueprint.mp4' },
    { title: 'The Pyramid of Power', src: '/Assets/Learning-media/The_Pyramid_of_Power.mp4' },
    { title: 'Virtualbabes Architecture', src: '/Assets/Learning-media/Virtualbabes_Architecture.mp4' },
];

const ARENA_FLOORS = [
    '/Assets/Textures/arena_floor.png',
    '/Assets/Textures/arena_floor_challenge.png',
    '/Assets/Textures/arena_floor_tournament.png',
    '/Assets/Textures/arena_floor_tournament_final.png',
];

let activeCategory = null;
let quickPlayActive = false;
let _lastStarredSig = null; // idempotency guard for renderStarredItems
let _spinDeg = 0;           // live wheel angle (radial layouts only)
let _spinBound = false;     // the drag / wheel handlers attach exactly once

// --- The WHEEL SPIN ---------------------------------------------------------
// Rotate the favourites ring. The ring turns; each item COUNTER-rotates so its
// label stays upright. Only the radial layouts (circle / arc / spokes) can spin —
// a layout that cannot use an angle never inherits one.
function applyMenuSpin(deg, persist) {
    const grid = document.getElementById('ch-starred-grid');
    const canSpin = isRadialGrid(getGridType());
    _spinDeg = canSpin ? Math.round(deg) : 0;
    if (grid) {
        grid.style.transform = _spinDeg ? 'rotate(' + _spinDeg + 'deg)' : '';
        grid.querySelectorAll('.ch-starred-btn').forEach(el => {
            el.style.transform = _spinDeg ? 'rotate(' + (-_spinDeg) + 'deg)' : '';
        });
        grid.style.cursor = canSpin ? 'grab' : '';
    }
    if (persist) {
        try { setMenuLayout({ gridSpin: _spinDeg }); } catch (e) {}
    }
    return _spinDeg;
}

function spinMenuWheel(deltaDeg) {
    applyMenuSpin((_spinDeg + deltaDeg) % 360, true);
}

function resetMenuSpin() { applyMenuSpin(0, true); }

// Drag-to-spin (pointer events, so mouse / pen / touch are one gesture) and
// scroll-to-spin. Attached ONCE to the grid element, which is built once.
function bindWheelSpin() {
    if (_spinBound) return;
    const grid = document.getElementById('ch-starred-grid');
    if (!grid) return;
    _spinBound = true;

    let dragging = false;
    let lastAngle = 0;
    const angleAt = (e) => {
        const r = grid.getBoundingClientRect();
        const cx = r.left + r.width / 2;
        const cy = r.top + r.height / 2;
        return Math.atan2(e.clientY - cy, e.clientX - cx) * 180 / Math.PI;
    };

    grid.addEventListener('pointerdown', (e) => {
        if (!isRadialGrid(getGridType())) return;
        dragging = true;
        lastAngle = angleAt(e);
        grid.style.cursor = 'grabbing';
        if (grid.setPointerCapture) { try { grid.setPointerCapture(e.pointerId); } catch (err) {} }
    });
    grid.addEventListener('pointermove', (e) => {
        if (!dragging) return;
        const a = angleAt(e);
        let delta = a - lastAngle;
        if (delta > 180) delta -= 360;
        if (delta < -180) delta += 360;
        lastAngle = a;
        applyMenuSpin(_spinDeg + delta, false);   // live, not persisted per frame
    });
    const endDrag = () => {
        if (!dragging) return;
        dragging = false;
        grid.style.cursor = 'grab';
        applyMenuSpin(_spinDeg, true);            // persist once, on release
    };
    grid.addEventListener('pointerup', endDrag);
    grid.addEventListener('pointercancel', endDrag);
    grid.addEventListener('pointerleave', endDrag);
    grid.addEventListener('wheel', (e) => {
        if (!isRadialGrid(getGridType())) return;
        e.preventDefault();
        spinMenuWheel(e.deltaY > 0 ? 15 : -15);
    }, { passive: false });
}

function init() {
    buildMinimalMenu();
    buildQuickPlay();
    buildTutorial();
    hideLobbyClutter();
}

function hideLobbyClutter() {
    document.body.classList.add('nexus-lobby-active');
}

function buildMinimalMenu() {
    const container = document.createElement('div');
    container.id = 'constellation-lobby';
    container.className = 'constellation-lobby minimal-menu';

    const bgLayer = document.createElement('div');
    bgLayer.className = 'constellation-bg';
    bgLayer.innerHTML = `
        <div class="constellation-bg-layer bg-nebula"></div>
        <div class="constellation-bg-layer bg-stars"></div>
        <div class="constellation-bg-layer bg-particles" id="constellation-particles"></div>
    `;
    container.appendChild(bgLayer);

    // §10.6: the boot / main-menu slideshow — three living images cycling with a gentle fade, one
    // image of each NPC helper. Mounted INTO the background stack (first child) so the nebula and
    // star layers stay painted on top of it, and the menu stays legible.
    if (window.SlideTheming) {
        window.SlideTheming.mount('menu', bgLayer);
        window.SlideTheming.refresh();
        window.SlideTheming.ensureStarter();
    }

    const title = document.createElement('div');
    title.className = 'constellation-title';
    title.innerHTML = '<h1>NFT-SEDUCTION</h1><p>Choose your path</p>';
    container.appendChild(title);

    // Main menu buttons (3 buttons shown in Constellation Hub)
    const menuGrid = document.createElement('div');
    menuGrid.className = 'ch-mini-menu';

    const wdBtn = document.createElement('button');
    wdBtn.className = 'menu-btn small';
    wdBtn.style.setProperty('--btn-color', '#00bcd4');
    wdBtn.innerHTML = '<span class="menu-btn-icon">🌍</span><span class="menu-btn-label">World Dashboard</span>';
    wdBtn.addEventListener('click', () => { if (window.openWorldDashboard) window.openWorldDashboard(); });
    menuGrid.appendChild(wdBtn);

    const qpBtn = document.createElement('button');
    qpBtn.className = 'menu-btn small';
    qpBtn.style.setProperty('--btn-color', '#ff4b4b');
    qpBtn.innerHTML = '<span class="menu-btn-icon">⚡</span><span class="menu-btn-label">Quick Play</span>';
    qpBtn.addEventListener('click', () => { if (window.toggleMatchmakingQueue) window.toggleMatchmakingQueue(); });
    menuGrid.appendChild(qpBtn);

    // NOTE: no Profile button here. The Player Profile is reached FROM the World
    // Dashboard (Player Hub ▸ Player Profile) — single navigation surface, one
    // entry point per feature, no duplicate UI on the main menu.

    container.appendChild(menuGrid);

    // Starred items section (from UserPreferences)
    const starredSection = document.createElement('div');
    starredSection.className = 'ch-starred-section';
    starredSection.innerHTML = '<h3 class="section-title">★ Favorites</h3>'
        + '<div class="ch-starred-grid" id="ch-starred-grid"></div>'
        + '<p class="ch-starred-hint" id="ch-starred-hint" style="display:none;">Drag or scroll the wheel to spin it.</p>';
    container.appendChild(starredSection);

    // Tutorial card
    const tutorialCard = document.createElement('button');
    tutorialCard.className = 'constellation-card tutorial-card';
    tutorialCard.style.setProperty('--card-color', '#9c27b0');
    tutorialCard.setAttribute('aria-label', 'Tutorial');
    tutorialCard.innerHTML = `
        <span class="card-bg tutorial-bg">📚</span>
        <span class="card-effect" style="background-image: url('/Assets/Images/Effects/Core-dissolution.webp')"></span>
        <span class="card-overlay"></span>
        <span class="card-border"></span>
        <span class="card-content">
            <span class="card-icon">📚</span>
            <span class="card-label">TUTORIAL</span>
            <span class="card-name">Learn to Play</span>
        </span>
        <span class="card-glow"></span>
    `;
    tutorialCard.addEventListener('click', openTutorial);
    container.appendChild(tutorialCard);

    // Customize button (bottom-right corner)
    const customizeBtn = document.createElement('button');
    customizeBtn.className = 'customize-menu-btn';
    customizeBtn.innerHTML = '🎨<span class="customize-menu-btn-label">Customize</span>';
    customizeBtn.setAttribute('aria-label', 'Customize menu');
    customizeBtn.title = 'Customize Menu';
    customizeBtn.addEventListener('click', () => { if (window.openMenuCustomization) window.openMenuCustomization(); });
    container.appendChild(customizeBtn);

    document.body.appendChild(container);

    // Render starred items now that #ch-starred-grid is live in the DOM
    // (getElementById only resolves elements already attached to the tree).
    renderStarredItems();
}

function createMenuButton(id, icon, label, color, onClick) {
    const btn = document.createElement('button');
    btn.className = 'menu-btn';
    btn.style.setProperty('--btn-color', color);
    btn.setAttribute('aria-label', label);
    btn.innerHTML = `
        <span class="menu-btn-icon">${icon}</span>
        <span class="menu-btn-label">${label}</span>
    `;
    btn.addEventListener('click', onClick);
    return btn;
}

function renderStarredItems() {
    const grid = document.getElementById('ch-starred-grid');
    console.log('[renderStarredItems] grid found:', !!grid);
    if (!grid) return;
    
    const starred = getStarred();
    const layout = getMenuLayout();
    const overrides = (layout && layout.buttonOverrides) || {};
    // Idempotency guard: skip no-op re-renders. renderStarredItems can be
    // invoked repeatedly (e.g. on every WS lobby_update); re-rendering the
    // favorites grid with identical data causes flicker + console spam.
    //
    // The signature covers the LAYOUT as well as the starred set. It used to cover
    // only the starred ids, so saving a customization (the panel's own Save path
    // calls renderStarredItems) hit this guard and did NOTHING until a reload —
    // a layout change looked like it had not been saved.
    const sig = starred.map(s => (s.wdTab || '') + ':' + (s.wdSub || '')).join('|')
        + '#' + [layout.gridType, layout.gridRadius, layout.gridGap, layout.gridCols,
            layout.gridRotation, layout.gridSpin, layout.defaultShape, layout.defaultSize].join(',')
        + '#' + JSON.stringify(overrides);
    if (sig === _lastStarredSig) return;
    _lastStarredSig = sig;
    console.log('[renderStarredItems] starred items:', starred.length, starred);
    
    if (starred.length === 0) {
        grid.innerHTML = '<p class="ch-no-favorites">No favorites yet. Star a CATEGORY (☆) in the World Dashboard to pin it here.</p>';
        const emptyHint = document.getElementById('ch-starred-hint');
        if (emptyHint) emptyHint.style.display = 'none';
        return;
    }

    // Spin support is bound ONCE, and the "you can spin this" hint shows only for a
    // layout that can actually spin (the wheel was reachable but nothing said so).
    bindWheelSpin();
    const spinHintEl = document.getElementById('ch-starred-hint');
    if (spinHintEl) spinHintEl.style.display = isRadialGrid(getGridType()) ? '' : 'none';

    const defaultShape = getDefaultShape();
    const defaultSize = getDefaultSize();
    const gridType = getGridType();

    const containerW = grid.parentElement?.clientWidth || 400;
    const containerH = 300;
    // Declared ONCE, at the scope both consumers need it: the per-item paint below
    // and the container's own positioning. ONE owner for the classification (the
    // grid library) — the old test named 'linear-h'/'linear-v', ids that do not
    // exist, so the linear layouts were absolutely positioned against their will.
    const isAbsolute = isPositionedGrid(gridType);

    const items = starred.map((item, i) => {
        const override = getButtonOverride(item.wdTab) || {};
        const systemShape = getShapeForSystem(item.wdTab);
        return {
            id: item.wdTab,
            shape: override.shape || systemShape,
            size: override.size || defaultSize,
            label: item.label || item.wdTab,
            icon: item.icon || '⭐',
            color: item.color || '#ffd700',
        };
    });

    let positions = [];
    try {
        positions = computePositions(gridType, items, containerW, containerH, layout);
    } catch(e) {
        console.warn('[MenuCustomization] compute failed, using grid:', e);
        positions = items.map((item, i) => ({
            x: 10 + (i % 3) * 90,
            y: 10 + Math.floor(i / 3) * 90,
            w: 72,
            h: 72,
        }));
    }

    grid.innerHTML = items.map((item, i) => {
        const pos = positions[i] || { x: 0, y: 0, w: 72, h: 72 };
        const clipPath = getClipPath(item.shape);
        const sizeConfig = SIZES[item.size] || { px: 72, font: 11, icon: 24 };

        const positionStyle = isAbsolute 
            ? `position: absolute; left: ${pos.x}px; top: ${pos.y}px; width: ${pos.w}px; height: ${pos.h}px;`
            : `width: ${pos.w}px; height: ${pos.h}px;`;

        return `
            <button class="ch-starred-btn shape-${item.shape} size-${item.size}" 
                    onclick="window.openWorldDashboardToTab && window.openWorldDashboardToTab('${item.wdTab}')" 
                    style="${positionStyle} --btn-color: ${item.color}; clip-path: ${clipPath};"
                    data-shape="${item.shape}"
                    data-size="${item.size}">
                <span class="ch-starred-icon" style="font-size: ${sizeConfig.icon}px;">${item.icon}</span>
                <span class="ch-starred-label" style="font-size: ${sizeConfig.font}px;">${item.label}</span>
            </button>
        `;
    }).join('');

    if (isAbsolute) {
        grid.style.position = 'relative';
        grid.style.height = containerH + 'px';
    } else {
        grid.style.position = '';
        grid.style.height = '';
    }

    // Re-apply the saved wheel angle. A non-radial layout clears it, so a rotation
    // can never be inherited by a layout that cannot spin.
    applyMenuSpin(parseInt(layout.gridSpin) || 0, false);

    console.log('[renderStarredItems] grid updated with', starred.length, 'items');
}

// Expose to global scope so World Dashboard can refresh hub after starring
window.renderStarredItems = renderStarredItems;
// The wheel's programmatic API — one entry point, so the customization panel, the
// controller and any future surface spin the SAME wheel instead of re-implementing
// a transform.
window.spinMenuWheel = spinMenuWheel;
window.resetMenuSpin = resetMenuSpin;
window.applyMenuSpin = applyMenuSpin;

function buildQuickPlay() {
    const qp = document.createElement('div');
    qp.id = 'quick-play-screen';
    qp.className = 'quick-play-screen hidden';
    qp.innerHTML = `
        <div class="qp-arena-floor" id="qp-arena-floor"></div>
        <div class="qp-content">
            <div class="qp-player-card" id="qp-player-card">
                <div class="qp-card-label">YOU</div>
                <div class="qp-card-art" id="qp-player-art" data-card-owner="self"></div>
            </div>
            <div class="qp-center">
                <div class="qp-radar" id="qp-radar">
                    <div class="qp-radar-sweep"></div>
                    <div class="qp-radar-ring"></div>
                    <div class="qp-radar-ring ring-2"></div>
                </div>
                <div class="qp-status" id="qp-status">Searching for opponent...</div>
                <div class="qp-eta" id="qp-eta">ETA ~15s</div>
            </div>
            <div class="qp-opponent-card" id="qp-opponent-card">
                <div class="qp-card-label">OPPONENT</div>
                <div class="qp-card-art qp-card-back" id="qp-opponent-art" data-card-owner="foreign">?</div>
            </div>
        </div>
        <button class="qp-cancel-btn" id="qp-cancel">CANCEL</button>
        <button class="qp-accept-btn hidden" id="qp-accept">ACCEPT</button>
    `;
    document.body.appendChild(qp);

    // §23.5.5: the quick-play card faces are marked self/foreign, so the viewer's own card-display
    // preference paints here too (the opponent search is where "how I see their card" first shows).
    if (window.CardViewSkins) {
        window.CardViewSkins.refresh();
        window.CardViewSkins.applyTo(qp);
    }

    document.getElementById('qp-cancel').addEventListener('click', closeQuickPlay);
    document.getElementById('qp-accept').addEventListener('click', acceptMatch);
}

function buildTutorial() {
    const tut = document.createElement('div');
    tut.id = 'tutorial-screen';
    tut.className = 'tutorial-screen hidden';
    tut.innerHTML = `
        <div class="tutorial-bg"></div>
        <div class="tutorial-content">
            <div class="tutorial-header">
                <h2>📚 Tutorial</h2>
                <button class="tutorial-close" id="tutorial-close">✕</button>
            </div>
            <div class="tutorial-video-container">
                <video id="tutorial-video" controls></video>
            </div>
            <div class="tutorial-list" id="tutorial-list"></div>
        </div>
    `;
    document.body.appendChild(tut);

    document.getElementById('tutorial-close').addEventListener('click', closeTutorial);

    const list = document.getElementById('tutorial-list');
    TUTORIAL_VIDEOS.forEach((vid) => {
        const item = document.createElement('button');
        item.className = 'tutorial-item';
        item.textContent = vid.title;
        item.addEventListener('click', () => playVideo(vid.src));
        list.appendChild(item);
    });
}

function openQuickPlay() {
    const state = window.GetGameState ? window.GetGameState("all") : null;
    if (state && (state.deck?.length ?? 0) < 5) {
        showToast("Deck must have 5 cards! Open Deck Manager to build your deck.", "error");
        setTimeout(() => { if (window.openDeckManager) window.openDeckManager(); }, 1500);
        return;
    }
    quickPlayActive = true;
    const qp = document.getElementById('quick-play-screen');
    qp.classList.remove('hidden');

    const floor = ARENA_FLOORS[Math.floor(Math.random() * ARENA_FLOORS.length)];
    document.getElementById('qp-arena-floor').style.backgroundImage = `url('${floor}')`;

    const playerArt = '/Assets/Images/Cards/Roxy.webp';
    document.getElementById('qp-player-art').style.backgroundImage = `url('${playerArt}')`;

    document.getElementById('qp-opponent-art').className = 'qp-card-art qp-card-back';
    document.getElementById('qp-opponent-art').textContent = '?';
    document.getElementById('qp-status').textContent = 'Searching for opponent...';
    document.getElementById('qp-eta').textContent = 'ETA ~15s';
    document.getElementById('qp-accept').classList.add('hidden');
    document.getElementById('qp-cancel').classList.remove('hidden');

    setTimeout(() => {
        if (!quickPlayActive) return;
        const oppArt = '/Assets/Images/Cards/Kat.webp';
        document.getElementById('qp-opponent-art').className = 'qp-card-art';
        document.getElementById('qp-opponent-art').style.backgroundImage = `url('${oppArt}')`;
        document.getElementById('qp-opponent-art').textContent = '';
        document.getElementById('qp-status').textContent = 'Match found!';
        document.getElementById('qp-eta').textContent = 'Kat';
        document.getElementById('qp-accept').classList.remove('hidden');
        document.getElementById('qp-cancel').classList.add('hidden');
    }, 3000);
}

function closeQuickPlay() {
    quickPlayActive = false;
    document.getElementById('quick-play-screen').classList.add('hidden');
}

function acceptMatch() {
    closeQuickPlay();
    if (window.toggleMatchmakingQueue) {
        window.toggleMatchmakingQueue();
    } else if (window.StartMatch) {
        window.StartMatch(true);
    } else {
        showToast("Match accepted! Loading battle...", "success");
    }
}

function openTutorial() {
    const tut = document.getElementById('tutorial-screen');
    tut.classList.remove('hidden');
    if (TUTORIAL_VIDEOS.length > 0) playVideo(TUTORIAL_VIDEOS[0].src);
}

function closeTutorial() {
    const tut = document.getElementById('tutorial-screen');
    tut.classList.add('hidden');
    const video = document.getElementById('tutorial-video');
    video.pause();
    video.src = '';
}

function playVideo(src) {
    const video = document.getElementById('tutorial-video');
    video.src = src;
    video.play();
}

function showToast(msg, type) {
    if (window.showToast) window.showToast(msg || 'info');
    else console.log('[Menu]', msg);
}

// ============================================================
// INITIALIZATION
// ============================================================
window.addEventListener('DOMContentLoaded', init);

export {
    init,
    renderStarredItems,
    spinMenuWheel,
    resetMenuSpin,
    applyMenuSpin,
    openQuickPlay,
    closeQuickPlay,
    acceptMatch,
    openTutorial,
    closeTutorial,
    playVideo,
};
