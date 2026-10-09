// ============================================================================
// slide_theming.js — §10.6 THE APP'S SLIDESHOWS (boot / main menu + World Dashboard)
// ----------------------------------------------------------------------------
// Two shows, three slides each, gentle fade between them:
//   menu      — app boot / main menu background  (one image of each NPC helper)
//   dashboard — World Dashboard background       (one image of each NPC helper)
//
// EVERY slide shows the placeholder pack by default (placeholder_assets.js), and EVERY slide can be
// re-themed by the viewer with one of their OWN bonded assets:
//
//   GET  /api/assets/slide-theme          → the viewer's own slots + the pack defaults + the rule
//   POST /api/assets/slide-theme          → { slot, asset_id }   (an asset of YOURS is required)
//   POST /api/assets/slide-theme/clear    → { slot } or {} to clear every slide
//   GET  /api/assets/catalogue            → the pack + the served price
//   POST /api/assets/starter              → the free starter pack (idempotent)
//
// This module is PRESENTATION ONLY: it paints what the server says to paint and never decides who
// owns what. It writes no branding, sends nothing to the server at render time, and if a read is
// refused it keeps the last known art AND says the read failed (a refusal is never rendered as
// "no theme").
//
// The boot fallback table in placeholder_assets.js is painted BEFORE the first read settles, so the
// boot screen is never blank; when the served policy arrives it REPLACES that table (server wins).
// ============================================================================

var ST_API = '/api';
var ST_FADE_MS = 1200;      // the gentle fade between slides
var ST_INTERVAL_MS = 8000;  // how long each slide is held

const stState = {
    loaded: false, loading: false, error: '', wallet: '',
    slots: {},              // slotId -> { slot, label, show, art_uri, art_kind, default_uri, active, asset_id, asset_name }
    policy: null, capacity: null,
    // §10.6: the server's light-art progress, so the client can say whether the art it paints is a
    // derived rendition or still the shipped original.
    derivatives: null, derivRetry: false,
    shows: {},              // which -> { host, root, layers, index, timer }
    starter: null,          // result of the last starter provisioning call
    lastStarterWallet: '',
};

// --- Wallet resolution: the same authority chain the rest of the client uses -------------------
function stWallet() {
    if (typeof window.getActiveWallet === 'function') {
        const w = window.getActiveWallet();
        if (w) return w;
    }
    if (window.currentWallet) return window.currentWallet;
    try {
        if (typeof window.GetGameState === 'function') {
            const st = window.GetGameState();
            if (st && st.wallet) return st.wallet;
        }
    } catch (e) { /* engine not ready */ }
    return window.userAddress || '';
}

// Only art a browser can actually load is painted: http(s) URLs and same-origin /Assets/ paths.
// Everything else (data:, blob:, javascript:, ipfs://, ar:// without a gateway) is refused here as
// well as server-side — we never invent a gateway.
function stSafeUri(u) {
    u = String(u || '');
    if (/^https?:\/\//i.test(u)) return u;
    if (/^\/[A-Za-z0-9._~\-/+]+$/.test(u)) return u;
    return '';
}

// One-shot retry on HTTP 429 (the studio / Kennel / Portfolio rule).
async function stApi(path, opts) {
    const w = stWallet();
    const sep = path.indexOf('?') >= 0 ? '&' : '?';
    const url = ST_API + path + (w ? sep + 'wallet=' + encodeURIComponent(w) : '');
    let resp = await fetch(url, opts);
    if (resp.status === 429) {
        await new Promise(function (r) { setTimeout(r, 450); });
        resp = await fetch(url, opts);
    }
    let data = null;
    try { data = await resp.json(); } catch (e) { data = null; }
    if (resp.status === 429) throw new Error('rate-limited — the read was refused, showing the last known slides');
    if (data && data.success === false) throw new Error(data.error || ('HTTP ' + resp.status));
    if (!resp.ok) throw new Error(data && data.error ? data.error : ('HTTP ' + resp.status));
    return data || {};
}

// The boot fallback: the pack frames this app ships with, one image of each NPC helper per show.
function stBootSlots(which) {
    const out = {};
    const conv = (which === 'dashboard') ? 'dashboard' : 'menu';
    const rows = (window.PlaceholderAssets && window.PlaceholderAssets.getSlideDefaults)
        ? window.PlaceholderAssets.getSlideDefaults(conv) : [];
    rows.forEach(function (r) {
        out[r.slot] = {
            slot: r.slot,
            label: conv + ' slide',
            show: conv,
            default_uri: stSafeUri(r.uri),
            art_uri: stSafeUri(r.uri),
            art_kind: 'default',
            active: false,
        };
    });
    return out;
}

// ── Rendering ────────────────────────────────────────────────────────────────

// stSlotIdsFor returns the slot ids of one show in declaration order, so slide 1 really is slide 1.
// The server's order wins; the boot table is the pre-read fallback.
function stSlotIdsFor(which) {
    const conv = (which === 'dashboard') ? 'dashboard' : 'menu';
    const ids = [];
    Object.keys(stState.slots).forEach(function (k) {
        const s = stState.slots[k];
        if (s && s.show === conv) ids.push(s.slot);
    });
    if (!ids.length) {
        const boot = stBootSlots(conv);
        Object.keys(boot).forEach(function (k) { ids.push(boot[k].slot); });
    }
    ids.sort();
    return ids;
}

function stArtFor(slot) {
    const s = stState.slots[slot];
    const uri = stSafeUri(s && s.art_uri ? s.art_uri : '');
    if (!uri) return '';
    // Only a PACK frame is probed: a player's own asset art is used exactly as declared.
    return stPreferLight(uri);
}

// §10.6 LIGHT ART: the server derives a small rendition of every pack frame
// (placeholder_derivatives.go). The client prefers it and probes ONCE per URI with a real <img>
// load, falling back to the shipped frame on error — so a rendition that does not exist yet can
// never paint a broken background, and the multi-megabyte original is only used until the small
// copy is known to exist.
const stLightProbe = {}; // light uri -> true | false

function stPreferLight(uri, tier) {
    if (!window.PlaceholderAssets || typeof window.PlaceholderAssets.lightPathFor !== 'function') return uri;
    const light = window.PlaceholderAssets.lightPathFor(uri, tier === 'thumb' ? 'thumb' : 'slide');
    if (!light) return uri;
    if (stLightProbe[light] === true) return light;
    if (stLightProbe[light] === false) return uri;
    const img = new Image();
    img.onload = function () {
        stLightProbe[light] = true;
        Object.keys(stState.shows).forEach(paint);
    };
    img.onerror = function () { stLightProbe[light] = false; };
    img.src = light;
    return uri; // the shipped frame is painted until the probe answers
}

function reduceMotion() {
    try {
        return !!(window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches);
    } catch (e) { return false; }
}

function setIndex(which, index) {
    const show = stState.shows[which];
    if (!show) return;
    const n = show.layers.length;
    const i = ((index % n) + n) % n;
    show.index = i;
    show.layers.forEach(function (layer, j) {
        layer.classList.toggle('is-active', j === i);
    });
    show.root.setAttribute('data-slide-index', String(i));
}

// paint writes the current art for every slide of a show. Only a background-image changes, and
// only when the value actually differs, so painting is cheap and idempotent.
function paint(which) {
    const show = stState.shows[which];
    if (!show) return 0;
    const ids = stSlotIdsFor(which);
    let painted = 0;
    for (let i = 0; i < show.layers.length; i++) {
        const layer = show.layers[i];
        const uri = ids[i] ? stArtFor(ids[i]) : '';
        if (!uri) {
            layer.style.backgroundImage = '';
            layer.setAttribute('data-slide-empty', '1');
            continue;
        }
        layer.removeAttribute('data-slide-empty');
        const css = 'url("' + uri + '")';
        if (layer.style.backgroundImage !== css) layer.style.backgroundImage = css;
        painted++;
    }
    if (show.index < 0 && painted) setIndex(which, 0);
    return painted;
}

// startCycle advances the show on a timer. The tick is skipped while the browser tab is hidden or
// the host is not in the layout, so a slideshow behind another overlay costs nothing (the same
// discipline the constellation-hub particles follow).
function startCycle(which) {
    const show = stState.shows[which];
    if (!show || show.timer || reduceMotion()) return;
    show.timer = setInterval(function () {
        if (document.hidden || show.paused || show.host.offsetParent === null) return;
        const n = Math.max(1, stSlotIdsFor(which).length);
        setIndex(which, (show.index + 1) % n);
    }, ST_INTERVAL_MS);
}

// mount attaches a slideshow to a host element. Idempotent for the same host.
function mount(which, host) {
    const conv = (which === 'dashboard') ? 'dashboard' : 'menu';
    if (!host) return null;
    const existing = stState.shows[conv];
    if (existing && existing.host === host && existing.root && host.contains(existing.root)) {
        paint(conv);
        return existing;
    }
    const root = document.createElement('div');
    root.className = 'slide-theme';
    root.setAttribute('data-slide-show', conv);
    root.setAttribute('aria-hidden', 'true');
    const layers = [];
    for (let i = 0; i < 3; i++) {
        const layer = document.createElement('div');
        layer.className = 'slide-theme-layer';
        root.appendChild(layer);
        layers.push(layer);
    }
    host.insertBefore(root, host.firstChild);
    const show = { host: host, root: root, layers: layers, index: -1, timer: null, paused: false };
    stState.shows[conv] = show;
    paint(conv);
    if (reduceMotion()) setIndex(conv, 0);
    else startCycle(conv);
    return show;
}

function unmount(which) {
    const show = stState.shows[which];
    if (!show) return;
    if (show.timer) clearInterval(show.timer);
    if (show.root && show.root.parentNode) show.root.parentNode.removeChild(show.root);
    delete stState.shows[which];
}

// pause/resume are for a caller that knows its host is about to be hidden or shown.
function pause(which) { const s = stState.shows[which]; if (s) s.paused = true; }
function resume(which) {
    const s = stState.shows[which];
    if (!s) return;
    s.paused = false;
    paint(which);
    startCycle(which);
}

// ── §10.7 UI-TREE ART (the World Dashboard's sub-UI trees) ──────────────────────────────────────
// Every World Dashboard tree (each category and each feature inside it) is assigned one placeholder
// frame of its OWN by the server (ui_tree_theming.go), with `rewards` and `achievements` pinned to
// Crypto-seraph. This renderer paints that assignment onto elements the dashboard marks with
// [data-wd-tree] — an unmarked or unassigned tree is left alone (ownership/naming is never guessed),
// and the LIGHT thumb rendition is preferred over the shipped frame.
const stUiTrees = { loaded: false, loading: false, map: {}, error: '', count: 0, unique: null, rule: '', pinned: [] };

async function loadUiTrees(force) {
    if (stUiTrees.loading) return stUiTrees;
    if (stUiTrees.loaded && !force) return stUiTrees;
    stUiTrees.loading = true;
    try {
        const res = await stApi('/assets/ui-trees', { method: 'GET' });
        const map = {};
        (res.trees || []).forEach(function (t) {
            if (t && t.tree_id) map[t.tree_id] = t;
        });
        stUiTrees.map = map;
        stUiTrees.count = Number(res.count || 0);
        stUiTrees.unique = res.unique === true;
        stUiTrees.pinned = (res.pinned_trees || []).slice();
        stUiTrees.rule = res.rule || '';
        stUiTrees.error = '';
        stUiTrees.loaded = true;
    } catch (e) {
        // A refused read keeps whatever was painted AND states the refusal — it is never rendered as
        // "this tree has no art".
        stUiTrees.error = e.message || 'the UI-tree art could not be read';
    } finally {
        stUiTrees.loading = false;
    }
    return stUiTrees;
}

// paintUiTrees applies the assignment to every marked element under `root`. Only a CSS custom
// property changes, and only when the value differs, so repeated calls are cheap.
function paintUiTrees(root) {
    const host = root && root.querySelectorAll ? root : document;
    const els = host.querySelectorAll('[data-wd-tree]');
    let painted = 0;
    const missing = [];
    for (let i = 0; i < els.length; i++) {
        const id = String(els[i].getAttribute('data-wd-tree') || '');
        const entry = stUiTrees.map[id];
        if (!entry || !entry.uri) {
            if (id && missing.indexOf(id) < 0) missing.push(id);
            continue;
        }
        const uri = stSafeUri(stPreferLight(entry.uri, 'thumb'));
        if (!uri) {
            if (missing.indexOf(id) < 0) missing.push(id);
            continue;
        }
        const css = 'url("' + uri + '")';
        if (els[i].style.getPropertyValue('--wd-tree-art') !== css) {
            els[i].style.setProperty('--wd-tree-art', css);
            els[i].setAttribute('data-wd-tree-sku', entry.sku || '');
        }
        painted++;
    }
    stUiTrees.painted = painted;
    stUiTrees.missing = missing;
    stUiTrees.marked = els.length;
    return { painted: painted, marked: els.length, missing: missing };
}

// ── Server calls ─────────────────────────────────────────────────────────────

const stListeners = [];

function stNotify() {
    stListeners.forEach(function (fn) { try { fn(describe()); } catch (e) { /* a listener must not break the slideshow */ } });
}

function onChange(fn) { if (typeof fn === 'function') stListeners.push(fn); }

// adoptSlots replaces the painted table with the served surface: the SERVER wins, so a server-side
// change to a slot, its default or its rule is never overruled by this client's boot fallback.
function adoptSlots(surface) {
    if (!surface || !Array.isArray(surface.slots)) return false;
    const next = {};
    surface.slots.forEach(function (s) {
        if (!s || !s.slot) return;
        next[s.slot] = {
            slot: s.slot,
            label: s.label || s.slot,
            show: (s.show === 'dashboard') ? 'dashboard' : 'menu',
            default_sku: s.default_sku || '',
            default_uri: stSafeUri(s.default_uri || ''),
            // §10.6: the pack ORIGINAL is named beside the (possibly lighter) art, so the studio and
            // the probes can always say which shipped frame a slide is showing.
            default_source_uri: stSafeUri(s.default_source_uri || ''),
            default_derived: !!s.default_derived,
            default_note: s.default_note || '',
            art_uri: stSafeUri(s.art_uri || ''),
            art_kind: s.art_kind || '',
            active: !!s.active,
            asset_id: s.asset_id || '',
            asset_name: s.asset_name || '',
        };
    });
    stState.slots = next;
    stState.staleSlots = (surface.stale_slots || []).slice();
    return true;
}

// refresh reads the viewer's own slide surface. A refused read KEEPS the last known art and states
// the refusal — it is never rendered as "you have no theme".
async function refresh(opts) {
    const force = !!(opts && opts.force);
    if (stState.loading) return describe();
    // A wallet that differs from the one the last read was for INVALIDATES the cache: a visitor who
    // connects mid-session must not keep seeing the anonymous defaults.
    const w = stWallet();
    if (stState.loaded && !force && w === stState.wallet) return describe();

    // Seed from the boot table so paint() always has art, even before (or without) a read.
    if (!Object.keys(stState.slots).length) {
        stState.slots = Object.assign({}, stBootSlots('menu'), stBootSlots('dashboard'));
    }
    stState.wallet = w;
    stState.loading = true;
    try {
        const res = await stApi('/assets/slide-theme', { method: 'GET' });
        adoptSlots(res.surface);
        stState.policy = res.policy || null;
        stState.capacity = res.capacity || null;
        stState.derivatives = (stState.policy && stState.policy.derivatives) || null;
        stState.error = '';
        stState.loaded = true;
        // With a wallet, take the FREE starter pack (idempotent server-side), so the frames these
        // slides show are already the player's own bonded assets and can be re-tied freely.
        if (res.wallet) ensureStarter();
        // The light renditions are derived in the BACKGROUND server-side. While any are pending, the
        // slides are showing the originals; ONE delayed re-read picks the small art up when the pass
        // finishes, so the boot screen stops fetching multi-megabyte files without polling.
        if (stState.derivatives && stState.derivatives.ready === false && !stState.derivRetry) {
            stState.derivRetry = true;
            setTimeout(function () {
                stState.derivRetry = false;
                refresh({ force: true });
            }, 20000);
        }
    } catch (e) {
        stState.error = e.message;
    } finally {
        stState.loading = false;
    }
    Object.keys(stState.shows).forEach(paint);
    stNotify();
    return describe();
}

// ensureStarter takes the FREE starter pack (once per wallet per page load). The server is
// idempotent by SKU ownership, so a repeat call is harmless; the client guard just avoids asking
// twice, and a failure clears the guard so a later attempt can retry.
async function ensureStarter() {
    const w = stWallet();
    if (!w) return null;
    if (stState.lastStarterWallet === w) return stState.starter;
    stState.lastStarterWallet = w;
    try {
        const res = await stApi('/assets/starter', { method: 'POST' });
        stState.starter = {
            granted: (res.granted || []).length,
            already: (res.already_owned || []).length,
            skipped: res.skipped || [],
            skus: res.starter_skus || [],
            note: res.note || '',
        };
    } catch (e) {
        stState.lastStarterWallet = '';
        stState.starter = { error: e.message };
    }
    stNotify();
    return stState.starter;
}

async function setSlot(slot, assetId) {
    const res = await stApi('/assets/slide-theme', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ slot: slot, asset_id: assetId || '' }),
    });
    adoptSlots(res.surface);
    stState.policy = res.policy || stState.policy;
    stState.capacity = res.capacity || stState.capacity;
    stState.error = '';
    stState.loaded = true;
    Object.keys(stState.shows).forEach(paint);
    stNotify();
    return describe();
}

async function clearSlot(slot) {
    const res = await stApi('/assets/slide-theme/clear', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ slot: slot || '' }),
    });
    adoptSlots(res.surface);
    stState.error = '';
    stState.loaded = true;
    Object.keys(stState.shows).forEach(paint);
    stNotify();
    return describe();
}

function clearAll() { return clearSlot(''); }

// slotViews is the studio's read: every slot with its art, in declaration order.
function slotViews() {
    return Object.keys(stState.slots).sort().map(function (k) { return stState.slots[k]; });
}

// ── Introspection (the probes and the studio render from this; nothing here is invented) ────────
function describe() {
    const shows = {};
    Object.keys(stState.shows).forEach(function (k) {
        const s = stState.shows[k];
        shows[k] = {
            index: s.index,
            layers: s.layers.length,
            painted: s.layers.filter(function (l) { return !!l.style.backgroundImage; }).length,
            active: s.layers.filter(function (l) { return l.classList.contains('is-active'); }).length,
            visible: s.host.offsetParent !== null,
        };
    });
    const keys = Object.keys(stState.slots);
    return {
        loaded: stState.loaded,
        error: stState.error,
        wallet: stState.wallet,
        slots: keys.length,
        themed: keys.filter(function (k) { return stState.slots[k] && stState.slots[k].active; }).length,
        stale: (stState.staleSlots || []).slice(),
        shows: shows,
        starter: stState.starter,
        capacity: stState.capacity,
        policy: stState.policy,
        derivatives: stState.derivatives,
        // §10.6: which painted slides are the LIGHT rendition rather than the shipped frame.
        derivedSlides: keys.filter(function (k) { return stState.slots[k] && stState.slots[k].default_derived; }).length,
        uiTrees: {
            loaded: stUiTrees.loaded,
            error: stUiTrees.error,
            count: stUiTrees.count,
            unique: stUiTrees.unique,
            pinned: stUiTrees.pinned.slice(),
            marked: stUiTrees.marked || 0,
            painted: stUiTrees.painted || 0,
            missing: (stUiTrees.missing || []).slice(),
        },
    };
}

// Convenience globals — the main menu and the World Dashboard each mount their own show.
window.mountBootSlides = function (host) { return mount('menu', host); };
window.mountDashboardSlides = function (host) { return mount('dashboard', host); };
window.SlideTheming = {
    refresh,
    ensureStarter,
    setSlot,
    clearSlot,
    clearAll,
    describe,
    onChange,
    mount,
    unmount,
    pause,
    resume,
    paint,
    slotViews,
    // §10.7 UI-tree art: the dashboard marks its trees and asks this to paint them.
    loadUiTrees,
    paintUiTrees,
    uiTrees: stUiTrees,
    // §10.6 light-art preference (a probe decides; the shipped frame is the fallback).
    preferLight: stPreferLight,
    lightProbe: stLightProbe,
    state: stState,
    BOOT_SLOTS: stBootSlots,
};

export { refresh, ensureStarter, setSlot, clearSlot, clearAll, describe, mount, unmount, pause, resume, onChange, slotViews, loadUiTrees, paintUiTrees };



