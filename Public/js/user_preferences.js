// ============================================================================
// user_preferences.js — User-curated constellation preferences
// ----------------------------------------------------------------------------
// Manages:
//   - Starred/favorited World Dashboard items (synced via WASM bridge)
//   - Constellation Hub layout (node positions, order)
//   - Bound asset theming (NFT -> button assignments)
//   - Per-user customization that survives sessions
// ============================================================================

const STORAGE_KEY = 'vbabes_user_prefs_v1';

const DEFAULTS = {
    starred: [],           // [{ wdTab, wdSub, label, icon, color, assetId }]
    layout: { zoom: 1.0, particles: true },
    boundAssets: {},       // { buttonId: assetId }
    mainMenu: {            // main screen customization
        showTutorial: true,
        showQuickPlay: true,
    },
    menuLayout: {          // menu customization
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
        gridSpin: 0,          // degrees the favourites WHEEL has been spun (radial grids only)
        defaultShape: 'circle',
        defaultSize: 'md',
        buttonOverrides: {},
    },
};

let _state = null;
let _listeners = [];

// --- Persistence ---
function load() {
    if (_state) return _state;
    try {
        const raw = localStorage.getItem(STORAGE_KEY);
        if (raw) {
            _state = Object.assign({}, DEFAULTS, JSON.parse(raw));
            // Ensure arrays exist (migration safety)
            if (!Array.isArray(_state.starred)) _state.starred = [];
            if (typeof _state.boundAssets !== 'object') _state.boundAssets = {};
            if (typeof _state.layout !== 'object') _state.layout = DEFAULTS.layout;
            if (typeof _state.mainMenu !== 'object') _state.mainMenu = DEFAULTS.mainMenu;
            if (typeof _state.menuLayout !== 'object') _state.menuLayout = DEFAULTS.menuLayout;
        } else {
            _state = JSON.parse(JSON.stringify(DEFAULTS));
        }
    } catch (e) {
        console.warn('[UserPrefs] load failed, using defaults:', e);
        _state = JSON.parse(JSON.stringify(DEFAULTS));
    }
    return _state;
}

function save() {
    if (!_state) return;
    try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(_state));
    } catch (e) {
        console.warn('[UserPrefs] save failed:', e);
    }
}

function notify() {
    _listeners.forEach(fn => {
        try { fn(_state); } catch (e) {}
    });
}

// --- WASM Bridge Sync ---
function syncToWasm() {
    if (!window.SyncMenuState) return;
    try {
        const wasmState = {
            grid_type: _state.menuLayout.gridType,
            default_shape: _state.menuLayout.defaultShape,
            default_size: _state.menuLayout.defaultSize,
            grid_cols: _state.menuLayout.gridCols,
            grid_gap: _state.menuLayout.gridGap,
            grid_radius: _state.menuLayout.gridRadius,
            grid_rotation: _state.menuLayout.gridRotation,
            starred_items: _state.starred.map(s => ({
                wd_tab: s.wdTab,
                wd_sub: s.wdSub,
                label: s.label,
                icon: s.icon,
                color: s.color,
                asset_id: s.assetId || '',
            })),
            button_overrides: Object.fromEntries(
                Object.entries(_state.menuLayout.buttonOverrides).map(([k, v]) => [k, { shape: v.shape, size: v.size }])
            ),
            quick_play_visible: _state.mainMenu.showQuickPlay,
            tutorial_visible: _state.mainMenu.showTutorial,
        };
        window.SyncMenuState(JSON.stringify(wasmState));
    } catch (e) {
        console.warn('[UserPrefs] WASM sync failed:', e);
    }
}

function syncFromWasm() {
    if (!window.GetMenuState) return;
    try {
        const raw = window.GetMenuState();
        if (!raw || raw === '{}') return;
        const ms = JSON.parse(raw);
        if (ms.grid_type) _state.menuLayout.gridType = ms.grid_type;
        if (ms.default_shape) _state.menuLayout.defaultShape = ms.default_shape;
        if (ms.default_size) _state.menuLayout.defaultSize = ms.default_size;
        if (ms.grid_cols) _state.menuLayout.gridCols = ms.grid_cols;
        if (ms.grid_gap) _state.menuLayout.gridGap = ms.grid_gap;
        if (ms.grid_radius) _state.menuLayout.gridRadius = ms.grid_radius;
        if (ms.grid_rotation) _state.menuLayout.gridRotation = ms.grid_rotation;
        if (ms.starred_items && ms.starred_items.length > 0) {
            // Only adopt WASM favorites when WASM actually has curated items.
            // An empty WASM menu-state must never wipe the user's localStorage
            // favorites (this caused the Favorites grid to flicker 1 -> 0 -> 1).
            _state.starred = ms.starred_items.map(s => ({
                wdTab: s.wd_tab,
                wdSub: s.wd_sub,
                label: s.label,
                icon: s.icon,
                color: s.color,
                assetId: s.asset_id,
            }));
        }
        if (ms.button_overrides) {
            _state.menuLayout.buttonOverrides = {};
            for (const [k, v] of Object.entries(ms.button_overrides)) {
                _state.menuLayout.buttonOverrides[k] = { shape: v.shape, size: v.size };
            }
        }
        if (ms.quick_play_visible !== undefined) _state.mainMenu.showQuickPlay = ms.quick_play_visible;
        if (ms.tutorial_visible !== undefined) _state.mainMenu.showTutorial = ms.tutorial_visible;
        save();
    } catch (e) {
        console.warn('[UserPrefs] WASM load failed:', e);
    }
}

// --- Public API ---
function get() { return load(); }

function onChange(fn) {
    _listeners.push(fn);
    return () => { _listeners = _listeners.filter(l => l !== fn); };
}

// --- Starred items ---
function isStarred(wdTab, wdSub) {
    const s = load();
    return s.starred.some(item => item.wdTab === wdTab && item.wdSub === wdSub);
}

function toggleStar(wdTab, wdSub, label, icon, color) {
    const s = load();
    const idx = s.starred.findIndex(item => item.wdTab === wdTab && item.wdSub === wdSub);
    if (idx >= 0) {
        s.starred.splice(idx, 1);
    } else {
        s.starred.push({
            wdTab, wdSub, label, icon, color,
            assetId: s.boundAssets[wdTab + ':' + wdSub] || null,
            addedAt: Date.now()
        });
    }
    save();
    notify();
    syncToWasm();
    return idx < 0; // true = now starred
}

function getStarred() {
    return load().starred;
}

function getStarredByTab(wdTab) {
    return load().starred.filter(item => item.wdTab === wdTab);
}

function reorderStarred(fromIndex, toIndex) {
    const s = load();
    if (fromIndex < 0 || fromIndex >= s.starred.length) return;
    if (toIndex < 0 || toIndex >= s.starred.length) return;
    const [item] = s.starred.splice(fromIndex, 1);
    s.starred.splice(toIndex, 0, item);
    save();
    notify();
    syncToWasm();
}

function unstar(wdTab, wdSub) {
    const s = load();
    s.starred = s.starred.filter(item => !(item.wdTab === wdTab && item.wdSub === wdSub));
    save();
    notify();
    syncToWasm();
}

// --- Placeholder ---
function setPlaceholder(tabId, url, label) {
    const s = load();
    if (!s.placeholders) s.placeholders = {};
    if (url) {
        s.placeholders[tabId] = { url, label };
    } else {
        delete s.placeholders[tabId];
    }
    save();
    notify();
}

function getPlaceholder(tabId) {
    return load().placeholders?.[tabId] || null;
}

function getAllPlaceholders() {
    return load().placeholders || {};
}

function bindAsset(wdTab, wdSub, assetId) {
    const s = load();
    const key = wdTab + ':' + wdSub;
    if (assetId) {
        s.boundAssets[key] = assetId;
    } else {
        delete s.boundAssets[key];
    }
    const starred = s.starred.find(i => i.wdTab === wdTab && i.wdSub === wdSub);
    if (starred) starred.assetId = assetId || null;
    save();
    notify();
}

function getBoundAsset(wdTab, wdSub) {
    return load().boundAssets[wdTab + ':' + wdSub] || null;
}

function getAllBoundAssets() {
    return load().boundAssets;
}

// --- Layout ---
function getLayout() {
    return load().layout;
}

function setLayout(partial) {
    const s = load();
    Object.assign(s.layout, partial);
    save();
    notify();
}

// --- Menu Layout ---
function getMenuLayout() {
    return load().menuLayout;
}

function setMenuLayout(partial) {
    const s = load();
    Object.assign(s.menuLayout, partial);
    save();
    notify();
    syncToWasm();
}

function getMainMenu() {
    return load().mainMenu;
}

function setMainMenu(partial) {
    const s = load();
    Object.assign(s.mainMenu, partial);
    save();
    notify();
    syncToWasm();
}

function getButtonOverride(buttonId) {
    return load().menuLayout.buttonOverrides?.[buttonId] || null;
}

function setButtonOverride(buttonId, override) {
    const s = load();
    if (!s.menuLayout.buttonOverrides) s.menuLayout.buttonOverrides = {};
    s.menuLayout.buttonOverrides[buttonId] = override;
    save();
    notify();
    syncToWasm();
}

function removeButtonOverride(buttonId) {
    const s = load();
    if (s.menuLayout.buttonOverrides) {
        delete s.menuLayout.buttonOverrides[buttonId];
        save();
        notify();
        syncToWasm();
    }
}

function getDefaultShape() {
    return load().menuLayout.defaultShape || 'circle';
}

function getDefaultSize() {
    return load().menuLayout.defaultSize || 'md';
}

function setDefaultShape(shape) {
    const s = load();
    s.menuLayout.defaultShape = shape;
    save();
    notify();
    syncToWasm();
}

function setDefaultSize(size) {
    const s = load();
    s.menuLayout.defaultSize = size;
    save();
    notify();
    syncToWasm();
}

function getGridType() {
    return load().menuLayout.gridType || 'grid';
}

function setGridType(type) {
    const s = load();
    s.menuLayout.gridType = type;
    save();
    notify();
    syncToWasm();
}

// --- Import / Export ---
function exportPrefs() {
    return JSON.stringify(load(), null, 2);
}

function importPrefs(json) {
    try {
        const data = typeof json === 'string' ? JSON.parse(json) : json;
        _state = Object.assign({}, DEFAULTS, data);
        save();
        notify();
        syncToWasm();
        return true;
    } catch (e) {
        console.error('[UserPrefs] import failed:', e);
        return false;
    }
}

function resetAll() {
    _state = JSON.parse(JSON.stringify(DEFAULTS));
    save();
    notify();
    syncToWasm();
}

// --- Expose ---
export {
    get, onChange,
    isStarred, toggleStar, getStarred, getStarredByTab, reorderStarred, unstar,
    bindAsset, getBoundAsset, getAllBoundAssets,
    getLayout, setLayout,
    getMenuLayout, setMenuLayout,
    getMainMenu, setMainMenu,
    getButtonOverride, setButtonOverride, removeButtonOverride,
    getDefaultShape, getDefaultSize, setDefaultShape, setDefaultSize,
    getGridType, setGridType,
    exportPrefs, importPrefs, resetAll,
    syncFromWasm, syncToWasm,
};
