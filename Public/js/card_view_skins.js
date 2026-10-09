// ============================================================================
// card_view_skins.js — §23.5.5 VIEWER-SCOPED CARD DISPLAY
// ----------------------------------------------------------------------------
// "Player 1 can change how they see player 2's cards."
//
// This module is PRESENTATION ONLY. It paints the viewer's own art over the card
// surfaces the viewer is looking at (the opponent's hand, the quick-play opponent
// art) and nothing else:
//   * it never writes game state, never contacts the match, and never tells the
//     server anything at render time;
//   * it never asks the server to brand a card — the only server call is the
//     viewer's OWN display preference (GET/POST /api/assets/card-view), which is
//     keyed by the viewer's wallet and names only the viewer's own asset;
//   * the engine's card VALUES, names and powers are untouched: only a background
//     image and a marker class are added, so a skinned card remains the same card.
//
// RENDER CONTRACT: a card surface marks WHO it belongs to with
//     data-card-owner="self" | "foreign" | "<wallet>"
// and this module skins it only when the served scope permits:
//     foreign_cards (default) -> only "foreign" surfaces
//     own_cards               -> only "self" surfaces
//     all_cards               -> both
// A surface with NO marker is never skinned (we do not guess ownership).
//
// THE THEME IS ALWAYS A BONDED ASSET (§23.5.1). The server serves `asset_required: true` and
// refuses any request that does not name one of the viewer's OWN assets, so there is no
// asset-free mode and no way to paint a card with art the viewer does not own.
// ============================================================================

const CVS_API = '/api';
const CVS_ATTR = 'data-card-owner';

const cvsState = {
    loaded: false, loading: false, active: false, mode: 'engine',
    scope: 'foreign_cards', asset_id: '', asset_name: '', media: null,
    capacity: null, policy: null, error: '', wallet: '',
};

let cvsObserver = null;
let cvsRafPending = false;
let cvsListeners = [];

// --- Wallet resolution: the same authority chain the rest of the client uses ----
function cvsWallet() {
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

// Only a source the browser can load directly is painted:
//   * http(s) — remote art the server validated (https|http|ipfs|ar are the accepted schemes; a
//     content address is NOT loadable without a gateway, and we do not invent one);
//   * a same-origin /path — the local NPC-helper placeholder pack lives at /Assets/Images/….
// Everything else (data:, javascript:, blob:, ipfs://, ar://) is refused here as well as server-side.
function cvsRenderableUri(u) {
    u = String(u || '');
    if (/^https?:\/\//i.test(u)) return u;
    if (/^\/[A-Za-z0-9._~\-/%() ]+$/.test(u)) return u;
    return '';
}

// One-shot retry on HTTP 429 (the studio / Kennel / Portfolio rule).
async function cvsApi(path, opts) {
    const w = cvsWallet();
    const sep = path.indexOf('?') >= 0 ? '&' : '?';
    const url = CVS_API + path + (w ? sep + 'wallet=' + encodeURIComponent(w) : '');
    let resp = await fetch(url, opts);
    if (resp.status === 429) {
        await new Promise(function (r) { setTimeout(r, 450); });
        resp = await fetch(url, opts);
    }
    let data = null;
    try { data = await resp.json(); } catch (e) { data = null; }
    if (resp.status === 429) throw new Error('rate-limited — could not be read; try again shortly');
    if (!resp.ok) throw new Error((data && (data.error || data.message)) || ('HTTP ' + resp.status));
    if (data && data.success === false) throw new Error(data.error || 'request refused');
    return data || {};
}

function cvsNotify() {
    cvsListeners.forEach(function (fn) {
        try { fn(cvsState); } catch (e) { /* a listener must not break rendering */ }
    });
}

function cvsApplyPayload(res) {
    const view = (res && res.view) || {};
    cvsState.policy = (res && res.policy) || cvsState.policy;
    cvsState.active = view.active === true;
    cvsState.mode = view.mode || 'engine';
    cvsState.scope = view.scope || 'foreign_cards';
    cvsState.asset_id = view.asset_id || '';
    cvsState.asset_name = view.asset_name || '';
    cvsState.media = view.media || null;
    // The server serves the wallet's own creation budget with every view (§23.5.1).
    cvsState.capacity = (res && res.capacity) || cvsState.capacity;
    cvsState.wallet = (res && res.wallet) || cvsWallet();
}

// artUri resolves the art this viewer should paint, or '' when nothing should be painted.
//   * `asset` mode is the ONLY theming mode: a bonded asset the viewer owns is required, and the
//     server refuses an asset-free request, so this art can never be reached without one.
//   * an https:// URI, or a same-origin /path (the NPC pack), paints directly;
//   * a CONTENT-ADDRESSED URI (ipfs:// / ar://) cannot be loaded in a browser without a gateway we
//     do not invent, so the deterministic NPC-helper STAND-IN for THAT ASSET is painted instead -
//     still tied to a real owned asset, and reported as a stand-in by describe().
function artUri() {
    if (!cvsState.active || cvsState.mode !== 'asset') return '';
    const direct = cvsRenderableUri(cvsState.media && cvsState.media.uri);
    if (direct) return direct;
    const pa = window.PlaceholderAssets;
    if (!pa || typeof pa.getCardViewFallback !== 'function') return '';
    const art = pa.getCardViewFallback(cvsState.asset_id);
    return art ? cvsRenderableUri(art.path) : '';
}

// artKind names where the painted art came from, for the studio's status line.
function artKind() {
    if (!cvsState.active || cvsState.mode !== 'asset') return '';
    if (cvsRenderableUri(cvsState.media && cvsState.media.uri)) return 'asset';
    return artUri() ? 'stand-in' : '';
}

// ownerKind classifies a marker value. An unknown wallet is 'foreign' unless it is ours.
function ownerKind(marker) {
    const v = String(marker || '').trim().toLowerCase();
    if (v === 'self') return 'self';
    if (v === 'foreign') return 'foreign';
    if (!v) return '';
    return (v === String(cvsWallet() || '').toLowerCase()) ? 'self' : 'foreign';
}

function scopeAllows(kind) {
    if (!kind) return false;
    if (cvsState.scope === 'all_cards') return true;
    if (cvsState.scope === 'own_cards') return kind === 'self';
    return kind === 'foreign';
}

// ── Painting ──────────────────────────────────────────────────────────────────
// applyTo paints (or clears) every marked card surface under `root`. It only ever sets a
// background image plus a marker class — never text, never data, never engine state.
function applyTo(root) {
    const host = root || document;
    if (!host || typeof host.querySelectorAll !== 'function') return 0;
    const art = artUri();
    const nodes = host.querySelectorAll('[' + CVS_ATTR + ']');
    let painted = 0;
    for (let i = 0; i < nodes.length; i++) {
        const el = nodes[i];
        const kind = ownerKind(el.getAttribute(CVS_ATTR));
        if (cvsState.active && art && scopeAllows(kind)) {
            if (el.getAttribute('data-cvs') !== 'on') {
                el.setAttribute('data-cvs', 'on');
                el.classList.add('cvs-art');
            }
            el.style.backgroundImage = 'url("' + art + '")';
            painted++;
        } else if (el.getAttribute('data-cvs') === 'on') {
            // Switching mode/scope must RESTORE the original look, not leave a stale skin.
            el.removeAttribute('data-cvs');
            el.classList.remove('cvs-art');
            el.style.backgroundImage = '';
        }
    }
    return painted;
}

function cvsSchedule() {
    if (cvsRafPending) return;
    cvsRafPending = true;
    requestAnimationFrame(function () {
        cvsRafPending = false;
        applyTo(document);
    });
}

// An observer is attached ONLY while a skin is configured, so an unconfigured viewer pays
// nothing (the lobby-lag lesson: no always-on render work). Attribute mutations are not
// observed, so the painting itself can never re-trigger the observer.
function ensureObserver() {
    const wanted = cvsState.active && !!artUri();
    if (wanted && !cvsObserver && typeof MutationObserver === 'function') {
        cvsObserver = new MutationObserver(cvsSchedule);
        cvsObserver.observe(document.body, { childList: true, subtree: true });
    } else if (!wanted && cvsObserver) {
        cvsObserver.disconnect();
        cvsObserver = null;
    }
    return !!cvsObserver;
}

// ── Server calls (the viewer's OWN preference only) ───────────────────────────
async function refresh(opts) {
    const force = !!(opts && opts.force);
    if (cvsState.loading) return cvsState;
    if (cvsState.loaded && !force) return cvsState;
    if (!cvsWallet()) {
        // No wallet: nothing to read, and the layer stays off rather than guessing. `loaded` stays
        // false so a later call (after the wallet connects) does the one real read.
        ensureObserver();
        applyTo(document);
        return cvsState;
    }
    cvsState.loading = true;
    try {
        const res = await cvsApi('/assets/card-view', { method: 'GET' });
        cvsApplyPayload(res);
        cvsState.error = '';
        cvsState.loaded = true;
    } catch (e) {
        // A refused read is REPORTED, never presented as "no preference".
        cvsState.error = e.message;
    } finally {
        cvsState.loading = false;
    }
    ensureObserver();
    applyTo(document);
    cvsNotify();
    return cvsState;
}

async function setView(opts) {
    const o = opts || {};
    const body = {
        mode: o.mode || 'engine',
        scope: o.scope || cvsState.scope || 'foreign_cards',
        asset_id: o.asset_id || '',
    };
    const res = await cvsApi('/assets/card-view', {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body),
    });
    cvsApplyPayload(res);
    cvsState.error = '';
    cvsState.loaded = true;
    ensureObserver();
    applyTo(document);
    cvsNotify();
    return cvsState;
}

async function clear() {
    const res = await cvsApi('/assets/card-view/clear', { method: 'POST' });
    cvsApplyPayload(res);
    cvsState.error = '';
    cvsState.loaded = true;
    ensureObserver();
    applyTo(document);
    cvsNotify();
    return cvsState;
}

// ── Introspection (the studio renders from this; nothing here is invented) ─────
function describe() {
    return {
        active: cvsState.active,
        mode: cvsState.mode,
        scope: cvsState.scope,
        asset_id: cvsState.asset_id,
        asset_name: cvsState.asset_name,
        media_uri: (cvsState.media && cvsState.media.uri) || '',
        art_kind: artKind(),
        capacity: cvsState.capacity,
        art: artUri(),
        error: cvsState.error,
        policy: cvsState.policy,
        marked: document.querySelectorAll('[' + CVS_ATTR + ']').length,
        painted: document.querySelectorAll('[data-cvs="on"]').length,
    };
}

function onChange(fn) {
    if (typeof fn === 'function') cvsListeners.push(fn);
}

window.CardViewSkins = {
    refresh,
    setView,
    clear,
    describe,
    applyTo,
    onChange,
    artUri,
    artKind,
    scopeAllows,
    ensureObserver,
    state: cvsState,
    ATTR: CVS_ATTR,
};
// Convenience globals for inline handlers / classic callers.
window.cvsApply = function () { return applyTo(document); };

export { refresh, setView, clear, describe, applyTo, onChange, ensureObserver, artUri, artKind };

