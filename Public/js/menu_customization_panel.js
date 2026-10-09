// ============================================================================
// menu_customization_panel.js — Live customization overlay (ES module)
// ----------------------------------------------------------------------------
// Full UI for customizing button shapes, sizes, grid layout
// Live preview updates as settings change
// Saves to UserPreferences on confirm
// ============================================================================

// --- Default config ---
function getDefaultConfig() {
    if (window.UserPreferences) {
        return JSON.parse(JSON.stringify(window.UserPreferences.getMenuLayout()));
    }
    return {
        gridType: 'grid',
        gridX: 'center',
        gridY: 'center',
        gridW: 'auto',
        gridH: 'auto',
        gridGap: '12px',
        gridCols: 3,
        gridRows: 0,
        gridRadius: '40%',
        gridRotation: '0deg',
        gridSpin: 0,
        defaultShape: 'circle',
        defaultSize: 'md',
        buttonOverrides: {},
    };
}

// --- Sample items for preview ---
// SOURCED FROM THE ONE TAXONOMY (window.WDTaxonomy, published read-only by
// world_dashboard.js). It used to be a hard-coded list of six invented ids
// ('world','battle','justice','economy','faith','pets'): the preview showed
// categories the game does not have, and the per-button override panel keyed its
// rows to those same ids, so an override set there could NEVER match a real
// button (real ids are 'player','careers','assets',…). Falls back to a
// placeholder ONLY if the dashboard has not been composed yet.
const SAMPLE_FALLBACK = [
    { id: 'player', label: 'Player Hub', icon: '🎮', color: '#00bcd4' },
    { id: 'careers', label: 'Careers & Factions', icon: '⚔️', color: '#ff9800' },
    { id: 'assets', label: 'Assets', icon: '🎒', color: '#8bc34a' },
    { id: 'economy', label: 'Economy & Trade', icon: '💰', color: '#f57f17' },
    { id: 'faith', label: 'Faith & Church', icon: '⛪', color: '#ffc107' },
    { id: 'system', label: 'System & Ops', icon: '🛠️', color: '#607d8b' },
];

function sampleItems() {
    const tax = window.WDTaxonomy;
    if (tax && Array.isArray(tax.categories) && tax.categories.length) {
        return tax.categories.slice(0, 6).map(c => ({
            id: c.id, label: c.name, icon: c.icon, color: c.color,
        }));
    }
    return SAMPLE_FALLBACK;
}

// Layout-class predicates — read from the grid library's ONE owner so a new grid
// type is classified once, not re-guessed per consumer.
function isPositioned(gridType) {
    if (window.MenuCustomization && window.MenuCustomization.isPositionedGrid) {
        return window.MenuCustomization.isPositionedGrid(gridType);
    }
    return !['grid', 'linearH', 'linearV'].includes(gridType);
}
function isRadial(gridType) {
    if (window.MenuCustomization && window.MenuCustomization.isRadialGrid) {
        return window.MenuCustomization.isRadialGrid(gridType);
    }
    return ['circle', 'arc', 'spokes'].includes(gridType);
}

// --- State ---
let panelEl = null;
let previewEl = null;
let currentConfig = null;
let currentTab = 'shape'; // shape, grid, per-button, controller
let selectedPerButtonId = null; // For per-button customization
let currentControllerScheme = 'gamepad'; // gamepad, keyboard, tilt

// --- Build Panel HTML ---
function buildPanel() {
    const shapes = window.MenuCustomization ? window.MenuCustomization.getAllShapeIds() : ['circle'];
    const sizes = window.MenuCustomization ? window.MenuCustomization.getAllSizeIds() : ['md'];
    const grids = window.MenuCustomization ? window.MenuCustomization.getAllGridIds() : ['grid'];

    let html = `
<div id="menu-customization-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel menu-customization-panel">
        <button id="btn-close-mc" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>🎨 Menu Customization</h2>
        <p style="font-size:12px;color:#b0bec5;margin-top:-8px;">Button shapes, sizes, grid layout</p>

        <div class="mc-tabs">
            <button class="mc-tab active" data-tab="shape">Shape</button>
            <button class="mc-tab" data-tab="grid">Grid</button>
            <button class="mc-tab" data-tab="per-button">Buttons</button>
            <button class="mc-tab" data-tab="controller">Controller</button>
        </div>

        <div class="mc-tab-content" data-content="shape">
            <div class="mc-section">
                <h3>Default Button Shape</h3>
                <div class="mc-shape-picker">
                    ${shapes.slice(0, 20).map(s => `
                        <button class="mc-shape-option" data-shape="${s}" title="${s}">
                            <span class="mc-shape-preview shape-${s}"></span>
                            <span class="mc-shape-name">${s}</span>
                        </button>
                    `).join('')}
                </div>
                <details class="mc-more-shapes">
                    <summary>Show all ${shapes.length} shapes...</summary>
                    <div class="mc-shape-picker">
                        ${shapes.slice(20).map(s => `
                            <button class="mc-shape-option" data-shape="${s}" title="${s}">
                                <span class="mc-shape-preview shape-${s}"></span>
                                <span class="mc-shape-name">${s}</span>
                            </button>
                        `).join('')}
                    </div>
                </details>
            </div>

            <div class="mc-section">
                <h3>Default Button Size</h3>
                <div class="mc-size-picker">
                    ${sizes.map(s => `
                        <button class="mc-size-option" data-size="${s}">
                            <span class="mc-size-preview size-${s}"></span>
                            <span class="mc-size-name">${s.toUpperCase()}</span>
                        </button>
                    `).join('')}
                </div>
            </div>
        </div>

        <div class="mc-tab-content hidden" data-content="grid">
            <div class="mc-section">
                <h3>Grid Layout</h3>
                <div class="mc-grid-picker">
                    ${grids.map(g => `
                        <button class="mc-grid-option" data-grid="${g}">
                            <span class="mc-grid-preview grid-icon-${g}"></span>
                            <span class="mc-grid-name">${g}</span>
                        </button>
                    `).join('')}
                </div>
            </div>

            <div class="mc-section">
                <h3>Grid Options</h3>
                <div class="mc-row">
                    <label>Gap</label>
                    <input type="range" id="mc-gap" min="4" max="32" value="12" step="2">
                    <span id="mc-gap-val">12px</span>
                </div>
                <div class="mc-row">
                    <label>Columns</label>
                    <input type="range" id="mc-cols" min="1" max="6" value="3" step="1">
                    <span id="mc-cols-val">3</span>
                </div>
                <div class="mc-row" id="mc-radius-row" style="display:none;">
                    <label>Radius</label>
                    <input type="range" id="mc-radius" min="20" max="60" value="40" step="5">
                    <span id="mc-radius-val">40%</span>
                </div>
                <div class="mc-row" id="mc-spin-row" style="display:none;">
                    <label>Spin</label>
                    <input type="range" id="mc-spin" min="-180" max="180" value="0" step="5">
                    <span id="mc-spin-val">0°</span>
                    <button type="button" class="vbt-btn vbt-btn-secondary mc-spin-reset" id="mc-spin-reset">Reset</button>
                </div>
                <p class="mc-hint" id="mc-spin-hint" style="display:none;">
                    Radial layouts can also be spun directly: drag the favourites wheel (or scroll over it) on the main menu.
                </p>
            </div>
        </div>

        <div class="mc-tab-content hidden" data-content="per-button">
            <div class="mc-section">
                <h3>Per-Button Overrides</h3>
                <p style="font-size:11px;color:#90a4ae;">Customize individual buttons (optional)</p>
                <div class="mc-overrides" id="mc-overrides">
                    ${sampleItems().map(item => `
                        <div class="mc-override-row" data-id="${item.id}">
                            <span class="mc-override-icon">${item.icon}</span>
                            <span class="mc-override-label">${item.label}</span>
                            <select class="mc-override-shape" data-id="${item.id}" data-prop="shape">
                                <option value="">Default</option>
                                ${shapes.slice(0, 12).map(s => `<option value="${s}">${s}</option>`).join('')}
                            </select>
                            <select class="mc-override-size" data-id="${item.id}" data-prop="size">
                                <option value="">Default</option>
                                ${sizes.map(s => `<option value="${s}">${s.toUpperCase()}</option>`).join('')}
                            </select>
                        </div>
                    `).join('')}
                </div>
            </div>
        </div>

        <div class="mc-tab-content hidden" data-content="controller">
            <div class="mc-section">
                <h3>Controller Scheme</h3>
                <p style="font-size:11px;color:#90a4ae;">Map controller buttons to menu actions</p>
                <div class="mc-controller-scheme-picker">
                    <button class="mc-scheme-btn ${currentControllerScheme === 'gamepad' ? 'active' : ''}" data-scheme="gamepad">
                        <span class="mc-scheme-icon">🎮</span>
                        <span class="mc-scheme-name">Gamepad</span>
                    </button>
                    <button class="mc-scheme-btn ${currentControllerScheme === 'keyboard' ? 'active' : ''}" data-scheme="keyboard">
                        <span class="mc-scheme-icon">⌨️</span>
                        <span class="mc-scheme-name">Keyboard</span>
                    </button>
                    <button class="mc-scheme-btn ${currentControllerScheme === 'tilt' ? 'active' : ''}" data-scheme="tilt">
                        <span class="mc-scheme-icon">📱</span>
                        <span class="mc-scheme-name">Tilt</span>
                    </button>
                </div>
                <div class="mc-controller-preview" id="mc-controller-preview">
                    ${renderControllerMappings()}
                </div>
                <button class="vbt-btn vbt-btn-secondary mc-test-controller-btn">🎮 Test Controller</button>
            </div>
        </div>

        <div class="mc-section">
            <h3>Live Preview</h3>
            <div class="mc-preview" id="mc-preview">
                <!-- Preview rendered here -->
            </div>
        </div>

        <div class="mc-actions">
            <button class="vbt-btn vbt-btn-secondary" id="mc-reset">Reset</button>
            <button class="vbt-btn vbt-btn-secondary" id="mc-cancel">Cancel</button>
            <button class="vbt-btn vbt-btn-primary" id="mc-save">Save</button>
        </div>
    </div>
</div>`;
    return html;
}

function renderControllerMappings() {
    const mappings = {
        gamepad: [
            { button: 'A', action: 'Select' },
            { button: 'B', action: 'Back' },
            { button: 'X', action: 'Quick Menu' },
            { button: 'Y', action: 'Favorites' },
            { button: 'LB', action: 'Prev Tab' },
            { button: 'RB', action: 'Next Tab' },
        ],
        keyboard: [
            { button: 'Enter', action: 'Select' },
            { button: 'Esc', action: 'Back' },
            { button: 'Space', action: 'Quick Menu' },
            { button: 'F', action: 'Favorites' },
            { button: '←', action: 'Prev Tab' },
            { button: '→', action: 'Next Tab' },
        ],
        tilt: [
            { button: 'Tilt Left', action: 'Prev Tab' },
            { button: 'Tilt Right', action: 'Next Tab' },
            { button: 'Shake', action: 'Quick Menu' },
            { button: 'Flip', action: 'Favorites' },
        ],
    };

    const scheme = mappings[currentControllerScheme] || mappings.gamepad;
    return `
        <div class="mc-mapping-list">
            ${scheme.map(m => `
                <div class="mc-mapping-row">
                    <span class="mc-mapping-button">${m.button}</span>
                    <span class="mc-mapping-arrow">→</span>
                    <span class="mc-mapping-action">${m.action}</span>
                </div>
            `).join('')}
        </div>
    `;
}

// --- Render Preview ---
export function renderPreview() {
    const preview = document.getElementById('mc-preview');
    if (!preview || !window.MenuCustomization) return;

    const config = currentConfig;
    const items = sampleItems().map(item => {
        const override = config.buttonOverrides?.[item.id] || {};
        const systemShape = window.MenuCustomization.getShapeForSystem(item.id);
        return {
            id: item.id,
            shape: override.shape || config.defaultShape || 'circle',
            size: override.size || config.defaultSize || 'md',
            label: item.label,
            icon: item.icon,
            color: item.color,
        };
    });

    const containerW = preview.clientWidth || 350;
    const containerH = 200;

    let positions = [];
    try {
        positions = window.MenuCustomization.computePositions(config.gridType, items, containerW, containerH, config);
    } catch(e) {
        positions = items.map((item, i) => ({
            x: 10 + (i % 3) * 80,
            y: 10 + Math.floor(i / 3) * 80,
            w: 60,
            h: 60,
        }));
    }

    const isAbsolute = isPositioned(config.gridType);
    preview.style.position = isAbsolute ? 'relative' : '';
    preview.style.height = isAbsolute ? containerH + 'px' : '';

    preview.innerHTML = items.map((item, i) => {
        const pos = positions[i] || { x: 0, y: 0, w: 60, h: 60 };
        const clipPath = window.MenuCustomization.getClipPath(item.shape);
        const sizeConfig = window.MenuCustomization.getSize(item.size);
        const positionStyle = isAbsolute
            ? `position:absolute;left:${pos.x}px;top:${pos.y}px;width:${pos.w}px;height:${pos.h}px;`
            : `width:${pos.w}px;height:${pos.h}px;`;

        return `
            <button class="mc-preview-btn shape-${item.shape}" 
                    style="${positionStyle}clip-path:${clipPath};background:${item.color}22;border-color:${item.color};">
                <span class="mc-preview-icon" style="font-size:${sizeConfig.icon * 0.7}px;">${item.icon}</span>
                <span class="mc-preview-label" style="font-size:${sizeConfig.font * 0.7}px;">${item.label}</span>
            </button>
        `;
    }).join('');

    // The spin is a CSS rotation of the RING with each item counter-rotated so labels
    // stay upright — exactly the transform the live menu applies, so the preview is
    // not a different visual rule from the thing it previews.
    const spinDeg = (isAbsolute && isRadial(config.gridType)) ? (parseInt(config.gridSpin) || 0) : 0;
    preview.style.transform = spinDeg ? `rotate(${spinDeg}deg)` : '';
    preview.querySelectorAll('.mc-preview-btn').forEach(el => {
        el.style.transform = spinDeg ? `rotate(${-spinDeg}deg)` : '';
    });
}

// --- Update UI from config ---
function updateUIFromConfig() {
    // Shape selection
    document.querySelectorAll('.mc-shape-option').forEach(btn => {
        btn.classList.toggle('selected', btn.dataset.shape === currentConfig.defaultShape);
    });
    // Size selection
    document.querySelectorAll('.mc-size-option').forEach(btn => {
        btn.classList.toggle('selected', btn.dataset.size === currentConfig.defaultSize);
    });
    // Grid selection
    document.querySelectorAll('.mc-grid-option').forEach(btn => {
        btn.classList.toggle('selected', btn.dataset.grid === currentConfig.gridType);
    });
    // Sliders
    const gapEl = document.getElementById('mc-gap');
    const colsEl = document.getElementById('mc-cols');
    const radiusEl = document.getElementById('mc-radius');
    if (gapEl) { gapEl.value = parseInt(currentConfig.gridGap) || 12; document.getElementById('mc-gap-val').textContent = gapEl.value + 'px'; }
    if (colsEl) { colsEl.value = currentConfig.gridCols || 3; document.getElementById('mc-cols-val').textContent = colsEl.value; }
    if (radiusEl) { radiusEl.value = parseInt(currentConfig.gridRadius) || 40; document.getElementById('mc-radius-val').textContent = radiusEl.value + '%'; }
    // Show/hide radius + spin for the RADIAL layouts (one owner: the grid library's
    // RADIAL_GRIDS via isRadial). A non-radial layout keeps the controls hidden so a
    // spin can never be set on a layout that cannot use it.
    const radialNow = isRadial(currentConfig.gridType);
    const radiusRow = document.getElementById('mc-radius-row');
    if (radiusRow) radiusRow.style.display = radialNow ? '' : 'none';
    const spinRow = document.getElementById('mc-spin-row');
    const spinHint = document.getElementById('mc-spin-hint');
    if (spinRow) spinRow.style.display = radialNow ? '' : 'none';
    if (spinHint) spinHint.style.display = radialNow ? '' : 'none';
    const spinEl = document.getElementById('mc-spin');
    const spinVal = document.getElementById('mc-spin-val');
    if (spinEl) {
        spinEl.value = parseInt(currentConfig.gridSpin) || 0;
        if (spinVal) spinVal.textContent = spinEl.value + '°';
    }
    // Overrides
    document.querySelectorAll('.mc-override-shape').forEach(sel => {
        const id = sel.dataset.id;
        sel.value = currentConfig.buttonOverrides?.[id]?.shape || '';
    });
    document.querySelectorAll('.mc-override-size').forEach(sel => {
        const id = sel.dataset.id;
        sel.value = currentConfig.buttonOverrides?.[id]?.size || '';
    });
}

// --- Bind Events ---
function bindEvents() {
    // Tab switching
    document.querySelectorAll('.mc-tab').forEach(tab => {
        tab.addEventListener('click', () => {
            document.querySelectorAll('.mc-tab').forEach(t => t.classList.remove('active'));
            tab.classList.add('active');
            const tabName = tab.dataset.tab;
            document.querySelectorAll('.mc-tab-content').forEach(content => {
                content.classList.toggle('hidden', content.dataset.content !== tabName);
            });
        });
    });
    // Shape picker
    document.querySelectorAll('.mc-shape-option').forEach(btn => {
        btn.addEventListener('click', () => {
            currentConfig.defaultShape = btn.dataset.shape;
            updateUIFromConfig();
            renderPreview();
        });
    });
    // Size picker
    document.querySelectorAll('.mc-size-option').forEach(btn => {
        btn.addEventListener('click', () => {
            currentConfig.defaultSize = btn.dataset.size;
            updateUIFromConfig();
            renderPreview();
        });
    });
    // Grid picker
    document.querySelectorAll('.mc-grid-option').forEach(btn => {
        btn.addEventListener('click', () => {
            currentConfig.gridType = btn.dataset.grid;
            updateUIFromConfig();
            renderPreview();
        });
    });
    // Sliders
    const gapEl = document.getElementById('mc-gap');
    const colsEl = document.getElementById('mc-cols');
    const radiusEl = document.getElementById('mc-radius');
    if (gapEl) gapEl.addEventListener('input', () => {
        currentConfig.gridGap = gapEl.value + 'px';
        document.getElementById('mc-gap-val').textContent = gapEl.value + 'px';
        renderPreview();
    });
    if (colsEl) colsEl.addEventListener('input', () => {
        currentConfig.gridCols = parseInt(colsEl.value);
        document.getElementById('mc-cols-val').textContent = colsEl.value;
        renderPreview();
    });
    if (radiusEl) radiusEl.addEventListener('input', () => {
        currentConfig.gridRadius = radiusEl.value + '%';
        document.getElementById('mc-radius-val').textContent = radiusEl.value + '%';
        renderPreview();
    });
    // Spin — the wheel's persisted angle. Only radial layouts can spin, so the row
    // (and the hint) are shown together with Radius.
    const spinEl = document.getElementById('mc-spin');
    const spinReset = document.getElementById('mc-spin-reset');
    if (spinEl) spinEl.addEventListener('input', () => {
        currentConfig.gridSpin = parseInt(spinEl.value) || 0;
        document.getElementById('mc-spin-val').textContent = currentConfig.gridSpin + '°';
        renderPreview();
    });
    if (spinReset) spinReset.addEventListener('click', () => {
        currentConfig.gridSpin = 0;
        if (spinEl) spinEl.value = 0;
        const out = document.getElementById('mc-spin-val');
        if (out) out.textContent = '0°';
        renderPreview();
    });
    // Override selects
    document.querySelectorAll('.mc-override-shape, .mc-override-size').forEach(sel => {
        sel.addEventListener('change', () => {
            const id = sel.dataset.id;
            const prop = sel.dataset.prop;
            if (!currentConfig.buttonOverrides) currentConfig.buttonOverrides = {};
            if (!currentConfig.buttonOverrides[id]) currentConfig.buttonOverrides[id] = {};
            if (sel.value) {
                currentConfig.buttonOverrides[id][prop] = sel.value;
            } else {
                delete currentConfig.buttonOverrides[id][prop];
            }
            renderPreview();
        });
    });
    // Action buttons
    document.getElementById('mc-reset').addEventListener('click', () => {
        currentConfig = getDefaultConfig();
        updateUIFromConfig();
        renderPreview();
    });
    document.getElementById('mc-cancel').addEventListener('click', () => {
        close();
    });
    document.getElementById('mc-save').addEventListener('click', () => {
        saveConfig();
        close();
        if (typeof window.renderStarredItems === 'function') window.renderStarredItems();
        if (window.showToast) window.showToast('Menu customization saved!', 'success');
    });
    document.getElementById('btn-close-mc').addEventListener('click', close);

    // Controller scheme picker
    document.querySelectorAll('.mc-scheme-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            currentControllerScheme = btn.dataset.scheme;
            document.querySelectorAll('.mc-scheme-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            renderControllerTab();
        });
    });

    // Test controller button
    const testBtn = document.querySelector('.mc-test-controller-btn');
    if (testBtn) testBtn.addEventListener('click', () => {
        if (window.showToast) window.showToast('🎮 Controller test: rumble activated!', 'info');
    });
}

// --- Save Config ---
function saveConfig() {
    if (!window.UserPreferences) return;
    window.UserPreferences.setMenuLayout(currentConfig);
}

// --- Open Panel ---
export function open(onConfirm) {
    if (!panelEl) {
        document.body.insertAdjacentHTML('beforeend', buildPanel());
        panelEl = document.getElementById('menu-customization-overlay');
        bindEvents();
    }
    currentConfig = getDefaultConfig();
    updateUIFromConfig();
    renderPreview();
    panelEl.style.display = 'flex';
}

// --- Close Panel ---
export function close() {
    if (panelEl) panelEl.style.display = 'none';
    if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
}

// --- Public API for app.js ---
export function switchTab(tab) { currentTab = tab; }
export function selectShape(shape) { if (currentConfig) { currentConfig.defaultShape = shape; renderPreview(); } }
export function selectSize(size) { if (currentConfig) { currentConfig.defaultSize = size; renderPreview(); } }
export function onGridChange(type) { if (currentConfig) { currentConfig.gridType = type; renderPreview(); } }
export function setControllerScheme(scheme) { currentControllerScheme = scheme; renderControllerTab(); }
export function testController() { if (window.showToast) window.showToast('🎮 Controller test: rumble activated!', 'info'); }
export function selectPerButton(id) { selectedPerButtonId = id; renderPerButtonList(); renderPreview(); }
export function onPerButtonShapeChange(id, shape) { if (currentConfig) { if (!currentConfig.buttonOverrides) currentConfig.buttonOverrides = {}; if (!currentConfig.buttonOverrides[id]) currentConfig.buttonOverrides[id] = {}; currentConfig.buttonOverrides[id].shape = shape; renderPreview(); } }
export function onPerButtonSizeChange(id, size) { if (currentConfig) { if (!currentConfig.buttonOverrides) currentConfig.buttonOverrides = {}; if (!currentConfig.buttonOverrides[id]) currentConfig.buttonOverrides[id] = {}; currentConfig.buttonOverrides[id].size = size; renderPreview(); } }
export function removePerButtonOverride(id) { if (currentConfig && currentConfig.buttonOverrides) { delete currentConfig.buttonOverrides[id]; renderPreview(); } }
export function saveMenuCustomization() { saveConfig(); if (typeof window.renderStarredItems === 'function') window.renderStarredItems(); if (window.showToast) window.showToast('Menu customization saved!', 'success'); }
export function resetMenuCustomization() { currentConfig = getDefaultConfig(); updateUIFromConfig(); renderPreview(); }
export function exportMenuCustomization() { return JSON.stringify(currentConfig, null, 2); }
export function importMenuCustomization(json) { try { currentConfig = JSON.parse(json); updateUIFromConfig(); renderPreview(); } catch(e) { console.warn('[MenuCustomization] import failed:', e); } }
export function renderPerButtonList() { /* Rendered by buildPanel */ }
export function renderControllerTab() {
    const preview = document.getElementById('mc-controller-preview');
    if (preview) preview.innerHTML = renderControllerMappings();
}
export function renderGridDiagram() { renderPreview(); }

// --- Expose to global for index.html ---
window.openMenuCustomization = open;
window.closeMenuCustomization = close;

// Auto-init
if (document.getElementById('menu-customization-overlay')) {
    open();
}
