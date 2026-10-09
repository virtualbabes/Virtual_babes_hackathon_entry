// ============================================================================
// bonded_branding.js — §23.5 BONDED BRANDING STUDIO (ecosphere-wide theming)
// ----------------------------------------------------------------------------
// Bonded assets are the player's own art. This is where a player paints an asset and
// wears it on ANY entity they own: items, themes, bots, pets, vehicles, churches,
// clubs, shops, world content — anything an image can be tied to, for branding.
//
// CARDS ARE EXCLUDED (deck cards, hand cards, religious-leader cards and any other
// card type). The client does not decide that: the server refuses card kinds, refuses
// to enumerate them, and serves the blocked list in `policy.blocked_target_kinds`.
// This module RENDERS the served policy rather than re-declaring it.
//
//   GET  /api/assets?wallet=            own assets + bindings + policy (served rules)
//   GET  /api/assets/targets?wallet=    every entity this wallet may brand
//   POST /api/assets/mint?wallet=       paint a new asset  { name, asset_type, media }
//   POST /api/assets/bind?wallet=       wear it              { asset_id, target_kind, target_id }
//   POST /api/assets/unbind?wallet=     take it off         { asset_id, target_kind, target_id }
//   GET  /api/assets/branding?kind=&target_id=   read path (no wallet: branding is public)
//
// Rate-limit honesty: a refused read is retried once; if it is still refused the panel
// SAYS so rather than rendering "you own nothing" (app-entry-mandate §3).
// ============================================================================

var API_BASE = '/api';

let bbState = {
    root: null, overlay: null, policy: null, assets: [], bindings: [],
    targets: [], kindFilter: '', status: '', busy: false, _init: false,
    // §23.5.5 viewer-scoped card display: the studio also renders the viewer's OWN "card eyes"
    // setting. It is a display preference — it never brands a card.
    // §23.5.1 capacity: how many bonded assets this wallet may create (its card/deck NFT supply).
    cardView: null, cardViewPolicy: null, capacity: null,
    // §10.8 player-to-player market: the listings read, plus the served fee/price rules.
    market: null, marketError: '',
};


function bbWallet() {
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

function bbEsc(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}

// Only schemes the server accepts are renderable; anything else is stated, never loaded.
function bbSafeUri(u) {
    u = String(u || '');
    return /^(https?:\/\/|ipfs:\/\/|ar:\/\/)/i.test(u) ? u : '';
}

// One-shot retry on HTTP 429 (mirrors the Kennel / Portfolio rule).
async function bbApi(path, opts) {
    const w = bbWallet();
    const sep = path.indexOf('?') >= 0 ? '&' : '?';
    const url = API_BASE + path + (w ? sep + 'wallet=' + encodeURIComponent(w) : '');
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

function bbStatus(msg, isError) {
    bbState.status = msg || '';
    const el = document.getElementById('bb-status');
    if (el) {
        el.textContent = bbState.status;
        el.style.color = isError ? '#ff8a80' : '#80deea';
    }
}

function bbKindLabel(kind) {
    const list = (bbState.policy && bbState.policy.bindable_target_kinds) || [];
    for (const k of list) { if (k.kind === kind) return k.label; }
    return kind;
}

function bbAssetTypeLabel(id) {
    const list = (bbState.policy && bbState.policy.asset_types) || [];
    for (const t of list) { if (String(t.kind) === String(id)) return t.label; }
    return 'Type ' + id;
}

function bbMimeOptions() {
    const limits = (bbState.policy && bbState.policy.media) || {};
    const list = limits.allowed_mime_types || ['image/png'];
    return list.map(m => `<option value="${bbEsc(m)}">${bbEsc(m)}</option>`).join('');
}

// A bonded asset's media: images render inline, other media types (and non-http content
// addresses) are shown by their content address rather than through an invented gateway.
function bbMediaThumb(media) {
    if (!media || !media.uri) return '<span class="bb-nomedia">no art</span>';
    const uri = bbSafeUri(media.uri);
    if (!uri) return '<span class="bb-nomedia">unrenderable URI</span>';
    if (/^https?:\/\//i.test(uri)) {
        return `<img class="bb-thumb" src="${bbEsc(uri)}" alt="${bbEsc(media.mime_type || 'art')}" loading="lazy" />`;
    }
    return `<code class="bb-thumb-uri" title="${bbEsc(uri)}">${bbEsc(uri.slice(0, 30))}…</code>`;
}

// ── DOM ───────────────────────────────────────────────────────────────────────
function bbBuild() {
    if (bbState._init) return;
    bbState._init = true;
    const html = `
<div id="bonded-branding-overlay" class="overlay vbt-overlay" style="display:none;">
  <div class="neon-glass-panel bb-panel">
    <button id="bb-close" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
    <h2>🎨 Bonded Branding Studio <small>§23.5</small></h2>
    <p class="bb-rule" id="bb-rule"></p>
    <div class="bb-grid">
      <section class="bb-col">
        <h3>Paint a new asset</h3>
        <div class="bb-form" id="bb-mint-form"></div>
        <button class="vbt-btn bb-primary" onclick="window.bbMint()">Mint bonded asset</button>
        <h3>Your assets</h3>
        <div id="bb-assets"></div>
      </section>
      <section class="bb-col">
        <h3>Your entities</h3>
        <div class="bb-kindbar" id="bb-kindbar"></div>
        <div id="bb-targets"></div>
      </section>
    </div>
    <section class="bb-col bb-cvs">
      <h3>Card eyes <small>how YOU see other players' cards</small></h3>
      <p class="bb-rule" id="cvs-rule"></p>
      <div id="cvs-controls"></div>
      <p id="cvs-status" class="bb-hint"></p>
    </section>
    <section class="bb-col bb-slides">
      <h3>App slides <small>boot / main menu + World Dashboard</small></h3>
      <p class="bb-rule" id="bb-slide-rule"></p>
      <div id="bb-slide-slots"></div>
      <p id="bb-slide-status" class="bb-hint"></p>
      <h3>Placeholder pack <small>free starter pack + shop</small></h3>
      <p class="bb-rule" id="bb-shop-rule"></p>
      <div class="bb-target-actions">
        <button class="vbt-btn bb-primary" onclick="window.bbClaimStarter()">Claim free starter pack</button>
        <button class="vbt-btn" onclick="window.bbBuySku()">Buy frame</button>
      </div>
      <label>Frame <select id="bb-shop-sku"></select></label>
      <p id="bb-shop-status" class="bb-hint"></p>
    </section>
    <section class="bb-col bb-themeslots">
      <h3>Theme slots <small>§27.6 — wear an asset, or take it out of play</small></h3>
      <p class="bb-rule" id="bb-ts-rule"></p>
      <!-- /api/theme/bind and /api/theme/lock had NO UI anywhere: the only client reference was a
           GET health sweep inside an uncomposed panel, so no player could ever bind a bonded asset
           to a UI slot or lock one out of the theme. Both doors are placed here, beside the studio
           that owns §27 client-side branding. A LOCKED asset is excluded from the wallet's theme
           MoodTag (BondedAssetRegistry.MoodTagForWallet), which is what makes the control mean
           something rather than only setting a flag. -->
      <div class="bb-target-actions">
        <select id="bb-ts-asset"></select>
        <input id="bb-ts-slot" type="text" placeholder="slot, e.g. board_bg (no ':')" />
        <button class="vbt-btn bb-primary" onclick="window.bbThemeBind()">Bind</button>
        <button class="vbt-btn" onclick="window.bbThemeLock()">Lock</button>
        <button class="vbt-btn" onclick="window.bbThemeUnlock()">Unlock</button>
      </div>
      <div id="bb-ts-bindings"></div>
      <p id="bb-ts-status" class="bb-hint"></p>
    </section>
    <section class="bb-col bb-market">
      <h3>Market <small>sell your art to another player</small></h3>
      <p class="bb-rule" id="bb-market-rule"></p>
      <div class="bb-target-actions">
        <select id="bb-market-asset"></select>
        <input id="bb-market-price" type="text" inputmode="decimal" placeholder="price in $VBV, e.g. 250" />
        <button class="vbt-btn bb-primary" onclick="window.bbListItem()">List for sale</button>
        <button class="vbt-btn" onclick="window.bbLoadMarket(true)">Refresh</button>
      </div>
      <div id="bb-market-listings"></div>
      <p id="bb-market-status" class="bb-hint"></p>
    </section>
    <p id="bb-status" class="ai-status"></p>
  </div>
</div>`;
    document.body.insertAdjacentHTML('beforeend', html);
    bbState.overlay = document.getElementById('bonded-branding-overlay');
    document.getElementById('bb-close').addEventListener('click', function () {
        bbState.overlay.style.display = 'none';
    });
}

function bbRenderMintForm() {
    const el = document.getElementById('bb-mint-form');
    if (!el) return;
    const limits = (bbState.policy && bbState.policy.media) || {};
    const types = (bbState.policy && bbState.policy.asset_types) || [];
    el.innerHTML = `
      <label>Name <input id="bb-name" type="text" maxlength="64" placeholder="Neon Livery" /></label>
      <label>Type <select id="bb-type">${types.map(t => `<option value="${bbEsc(t.kind)}">${bbEsc(t.label)}</option>`).join('')}</select></label>
      <label>Art URI <input id="bb-uri" type="text" placeholder="ipfs://… or https://…" /></label>
      <label>MIME <select id="bb-mime">${bbMimeOptions()}</select></label>
      <label>sha256 <input id="bb-hash" type="text" maxlength="64" placeholder="64 hex chars" /></label>
      <div class="bb-row">
        <label>Mood <select id="bb-mood">
          <option value="0">Neutral</option>
          <option value="1">Benevolent</option>
          <option value="2">Malevolent</option>
        </select></label>
        <label>Royalty bps <input id="bb-royalty" type="number" min="0" max="1000" value="0" /></label>
      </div>
      <div class="bb-row">
        <label>Bytes <input id="bb-bytes" type="number" min="1" max="${bbEsc(limits.max_bytes || '')}" /></label>
        <label>Width <input id="bb-width" type="number" min="0" max="${bbEsc(limits.max_dimension || '')}" /></label>
        <label>Height <input id="bb-height" type="number" min="0" max="${bbEsc(limits.max_dimension || '')}" /></label>
      </div>`;
}

function bbRenderPolicy() {
    const ruleEl = document.getElementById('bb-rule');
    const p = bbState.policy || {};
    if (ruleEl) {
        const blocked = (p.blocked_target_kinds || []).slice(0, 6).join(', ');
        ruleEl.textContent = (p.rule || '') +
            (p.legitimacy ? '  ' + p.legitimacy : '') +
            (p.capacity_rule ? '  ' + p.capacity_rule : '') +
            (p.creation_requires ? '  ' + p.creation_requires + '.' : '') +
            (blocked ? '  Blocked kinds: ' + blocked + '…' : '');
    }
    const bar = document.getElementById('bb-kindbar');
    if (bar) {
        const kinds = p.bindable_target_kinds || [];
        bar.innerHTML = `<button class="bb-kind-btn${bbState.kindFilter === '' ? ' active' : ''}" onclick="window.bbFilterKind('')">All</button>` +
            kinds.map(k => `<button class="bb-kind-btn${bbState.kindFilter === k.kind ? ' active' : ''}" onclick="window.bbFilterKind('${bbEsc(k.kind)}')">${bbEsc(k.label)}</button>`).join('');
    }
    bbRenderMintForm();
}

function bbRenderAssets() {
    const el = document.getElementById('bb-assets');
    if (!el) return;
    const cap = bbState.capacity;
    const capLine = cap
        ? `<p class="bb-cap">Bonded assets: <strong>${bbEsc(cap.used)}/${bbEsc(cap.limit)}</strong> — capacity equals your card/deck NFT supply.${bbEsc(cap.limit === 0 ? ' You hold no card/deck NFTs yet, so you cannot create one.' : '')}</p>`
        : '';
    if (!bbState.assets.length) {
        el.innerHTML = capLine + '<p class="bb-empty">You own no bonded assets yet — paint one above.</p>';
        return;
    }
    el.innerHTML = capLine + bbState.assets.map(a => {
        const binds = bbState.bindings.filter(b => b.asset_id === a.asset_id);
        const where = binds.length
            ? binds.map(b => `${bbEsc(bbKindLabel(b.target_kind))}: ${bbEsc(b.target_id)}`).join('<br>')
            : '<em>not worn anywhere</em>';
        return `<div class="bb-asset">
            <div class="bb-asset-art">${bbMediaThumb(a.media)}</div>
            <div class="bb-asset-body">
              <strong>${bbEsc(a.name)}</strong>
              <small>${bbEsc(bbAssetTypeLabel(a.asset_type))}${a.black_market_adopted ? ' · black market' : ''}${a.certified ? ' · certified' : ''}</small>
              <small class="bb-where">${where}</small>
            </div>
          </div>`;
    }).join('');
}

function bbRenderTargets() {
    const el = document.getElementById('bb-targets');
    if (!el) return;
    const list = bbState.targets;
    if (!list.length) {
        el.innerHTML = '<p class="bb-empty">No entities of this kind are owned by the connected wallet yet.</p>';
        return;
    }
    const options = bbState.assets.length
        ? bbState.assets.map(a => `<option value="${bbEsc(a.asset_id)}">${bbEsc(a.name)}</option>`).join('')
        : '<option value="">no assets yet</option>';
    el.innerHTML = list.map(t => {
        const worn = (t.branding || []).map(b => `
            <span class="bb-worn">${bbMediaThumb(b.media)}<small>${bbEsc(b.asset_name || b.asset_id)}</small>
              <button class="bb-x" title="Remove branding"
                onclick="window.bbUnbind('${bbEsc(b.asset_id)}','${bbEsc(t.kind)}','${bbEsc(t.target_id)}')">&times;</button>
            </span>`).join('');
        return `<div class="bb-target">
            <div class="bb-target-head"><strong>${bbEsc(t.name || t.target_id)}</strong>
              <small>${bbEsc(t.kind_label || bbKindLabel(t.kind))}</small></div>
            <div class="bb-worn-row">${worn || '<span class="bb-nomedia">no art</span>'}</div>
            <div class="bb-target-actions">
              <select id="bb-sel-${bbEsc(t.target_id)}">${options}</select>
              <button class="vbt-btn" onclick="window.bbBind('${bbEsc(t.kind)}','${bbEsc(t.target_id)}')">Wear</button>
            </div>
          </div>`;
    }).join('');
}

// ── §23.5.5 Card eyes (viewer-scoped card display) ────────────────────────────
// Renders ONLY the served policy. It never sends a card, a card id or another player's wallet:
// the choice is stored against the viewer's OWN wallet (POST /api/assets/card-view) and applied
// to what THIS player sees. Cards remain excluded from bonded assets, so nothing here can brand
// a card — this is a display preference.
function cvsOptions(list, selected) {
    return (list || []).map(function (d) {
        const value = String(d.kind == null ? d : d.kind);
        const label = String(d.label == null ? value : d.label);
        return `<option value="${bbEsc(value)}"${value === String(selected == null ? '' : selected) ? ' selected' : ''}>${bbEsc(label)}</option>`;
    }).join('');
}

function cvsRender() {
    const ruleEl = document.getElementById('cvs-rule');
    const host = document.getElementById('cvs-controls');
    const statusEl = document.getElementById('cvs-status');
    if (!host) return;
    const pol = bbState.cardViewPolicy || {};
    const view = bbState.cardView || null;
    const local = (window.CardViewSkins && window.CardViewSkins.describe) ? window.CardViewSkins.describe() : null;
    if (ruleEl) {
        ruleEl.textContent = (pol.rule || 'Changes only how you see cards.') +
            (pol.cards_excluded === true ? ' Cards are excluded from bonded assets.' : '');
    }
    if (!bbWallet()) {
        host.innerHTML = '<p class="bb-empty">Connect a wallet to set how you see other players cards.</p>';
        if (statusEl) statusEl.textContent = '';
        return;
    }
    const mode = (view && view.mode) || 'engine';
    const scope = (view && view.scope) || pol.default_scope || 'foreign_cards';
    const hasAssets = bbState.assets.length > 0;
    const assetOptions = hasAssets
        ? bbState.assets.map(a => `<option value="${bbEsc(a.asset_id)}"${view && view.asset_id === a.asset_id ? ' selected' : ''}>${bbEsc(a.name)}</option>`).join('')
        : '<option value="">no bonded assets yet</option>';
    host.innerHTML = `
      <div class="bb-form">
        <div class="bb-row">
          <label>Wear it <select id="cvs-mode">${cvsOptions(pol.modes, mode)}</select></label>
          <label>Which cards <select id="cvs-scope">${cvsOptions(pol.scopes, scope)}</select></label>
        </div>
        <label>Your bonded asset <select id="cvs-asset"${hasAssets ? '' : ' disabled'}>${assetOptions}</select></label>
      </div>
      <div class="bb-target-actions">
        <button class="vbt-btn bb-primary" onclick="window.bbCvsApply()">Apply to my view</button>
        <button class="vbt-btn" onclick="window.bbCvsClear()">Turn off</button>
      </div>`;
    if (!statusEl) return;
    if (local && local.error) {
        statusEl.textContent = 'Your card display could not be read (' + local.error + ').';
    } else if (local && local.active) {
        let html = 'Active: ' + bbEsc(local.mode) + ' &middot; ' + bbEsc(local.scope) +
            (local.art_kind === 'stand-in'
                ? ' &middot; stand-in art (the asset is content-addressed)'
                : (local.art ? ' &middot; art applied' : ' &middot; no art yet'));
        if (local.art) {
            // Same-origin art renders too (the /Assets NPC pack), so the preview is not limited to
            // remote https art.
            html += ` <img class="bb-thumb" src="${bbEsc(local.art)}" alt="card art" loading="lazy" />`;
        }
        statusEl.innerHTML = html;
    } else {
        statusEl.textContent = 'Off — cards render with the engine art.';
    }
    // The creation budget is SERVED, never inferred (§23.5.1): showing it is what makes a refusal
    // understandable instead of mysterious.
    const cap = bbState.capacity;
    if (cap) {
        statusEl.innerHTML += ` <span class="bb-cap-inline">Bonded assets ${bbEsc(cap.used)}/${bbEsc(cap.limit)} — capacity equals your card/deck NFT supply.</span>`;
    }
    if (!hasAssets) {
        statusEl.innerHTML += ' A bonded asset is required for this: create one above first.';
    }
}

function cvsField(id) {
    const el = document.getElementById(id);
    return el ? String(el.value == null ? '' : el.value).trim() : '';
}

async function bbCvsApply() {
    if (!window.CardViewSkins) return;
    try {
        await window.CardViewSkins.setView({
            mode: cvsField('cvs-mode'),
            scope: cvsField('cvs-scope'),
            asset_id: cvsField('cvs-asset'),
        });
        bbStatus('Card eyes updated — this changes only what YOU see.', false);
        await bbLoad();
    } catch (e) {
        bbStatus('Card eyes refused: ' + e.message, true);
    }
}

async function bbCvsClear() {
    if (!window.CardViewSkins) return;
    try {
        await window.CardViewSkins.clear();
        bbStatus('Card eyes off — cards render with the engine art.', false);
        await bbLoad();
    } catch (e) {
        bbStatus('Could not turn card eyes off: ' + e.message, true);
    }
}

// ── Load ──────────────────────────────────────────────────────────────────────
// The two reads are independent: if one is refused, the OTHER still renders, and the refused
// one is reported as unreadable rather than presented as "nothing owned" (§3 honesty rule).
async function bbLoad() {
    const w = bbWallet();
    if (!w) {
        // No wallet: still read the PUBLIC policy so a visitor can see the rules (and that cards
        // are excluded) instead of staring at an unexplained empty studio.
        try {
            const res = await bbApi('/assets/targets');
            bbState.policy = res.policy || bbState.policy;
        } catch (e) { /* the rules simply stay unstated below */ }
        bbRenderPolicy();
        bbRenderAssets();
        bbRenderTargets();
        cvsRender();
        await bbLoadSlides();
        await bbLoadMarket();
        bbStatus('Connect a wallet to brand your entities.', false);
        const el = document.getElementById('bb-targets');
        if (el) el.innerHTML = '<p class="bb-empty">No wallet connected — connect to see the entities you may brand.</p>';
        return;
    }
    const problems = [];
    try {
        const res = await bbApi('/assets');
        bbState.assets = Array.isArray(res.assets) ? res.assets : [];
        bbState.bindings = Array.isArray(res.bindings) ? res.bindings : [];
        bbState.policy = res.policy || bbState.policy;
        // The chest serves the wallet's own creation budget (§23.5.1) - never inferred client-side.
        bbState.capacity = res.capacity || bbState.capacity;
    } catch (e) {
        problems.push('your assets could not be read (' + e.message + ')');
    }
    try {
        const path = '/assets/targets' + (bbState.kindFilter ? '?kind=' + encodeURIComponent(bbState.kindFilter) : '');
        const res = await bbApi(path);
        bbState.targets = Array.isArray(res.targets) ? res.targets : [];
        bbState.policy = res.policy || bbState.policy;
    } catch (e) {
        problems.push('your entities could not be read (' + e.message + ')');
        bbState.targets = [];
    }
    // The card-display read is INDEPENDENT: a refused read says so rather than rendering "off".
    try {
        const res = await bbApi('/assets/card-view');
        bbState.cardView = res.view || null;
        bbState.cardViewPolicy = res.policy || bbState.cardViewPolicy;
        if (window.CardViewSkins) await window.CardViewSkins.refresh({ force: true });
    } catch (e) {
        problems.push('your card display could not be read (' + e.message + ')');
    }
    bbRenderPolicy();
    bbRenderAssets();
    bbRenderTargets();
    bbRenderThemeSlots();
    cvsRender();
    await bbLoadSlides();
    await bbLoadMarket();
    if (problems.length) {
        bbStatus('Partly unreadable — ' + problems.join(', ') + '.', true);
    }
}

// ── Actions ───────────────────────────────────────────────────────────────────
function bbField(id) {
    const el = document.getElementById(id);
    return el ? String(el.value == null ? '' : el.value).trim() : '';
}

function bbReadMintForm() {
    const name = bbField('bb-name');
    const uri = bbField('bb-uri');
    const hash = bbField('bb-hash');
    const bytes = Number(bbField('bb-bytes')) || 0;
    const width = Number(bbField('bb-width')) || 0;
    const height = Number(bbField('bb-height')) || 0;
    if (!name) throw new Error('a name is required');
    if (!uri) throw new Error('the art URI is required');
    if (!/^[0-9a-f]{64}$/i.test(hash)) throw new Error('the sha256 of the art is required (64 hex characters)');
    if (bytes <= 0) throw new Error('the declared byte size is required');
    return {
        name: name,
        asset_type: Number(bbField('bb-type')) || 0,
        mood_tag: Number(bbField('bb-mood')) || 0,
        royalty_bps: Number(bbField('bb-royalty')) || 0,
        media: { uri: uri, mime_type: bbField('bb-mime'), hash_hex: hash, bytes: bytes, width: width, height: height }
    };
}

async function bbMint() {
    if (bbState.busy) return;
    bbState.busy = true;
    try {
        const body = bbReadMintForm();
        const res = await bbApi('/assets/mint', {
            method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body)
        });
        bbStatus('Minted ' + (res.asset_id || 'a bonded asset') + ' — now pick an entity below and wear it.', false);
        ['bb-name', 'bb-uri', 'bb-hash', 'bb-bytes', 'bb-width', 'bb-height'].forEach(function (id) {
            const el = document.getElementById(id);
            if (el) el.value = '';
        });
        await bbLoad();
    } catch (e) {
        bbStatus('Mint refused: ' + e.message, true);
    } finally { bbState.busy = false; }
}

async function bbBind(kind, targetId) {
    if (bbState.busy) return;
    bbState.busy = true;
    try {
        const sel = document.getElementById('bb-sel-' + targetId);
        const assetId = sel ? sel.value : '';
        if (!assetId) throw new Error('paint a bonded asset first');
        await bbApi('/assets/bind', {
            method: 'POST', headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ asset_id: assetId, target_kind: kind, target_id: targetId })
        });
        bbStatus('Wearing ' + assetId + ' on ' + kind + ' ' + targetId + '.', false);
        await bbLoad();
    } catch (e) {
        bbStatus('Binding refused: ' + e.message, true);
    } finally { bbState.busy = false; }
}

async function bbUnbind(assetId, kind, targetId) {
    if (bbState.busy) return;
    bbState.busy = true;
    try {
        await bbApi('/assets/unbind', {
            method: 'POST', headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ asset_id: assetId, target_kind: kind, target_id: targetId })
        });
        bbStatus('Removed ' + assetId + ' from ' + kind + ' ' + targetId + '.', false);
        await bbLoad();
    } catch (e) {
        bbStatus('Unbind refused: ' + e.message, true);
    } finally { bbState.busy = false; }
}

async function bbFilterKind(kind) {
    bbState.kindFilter = kind || '';
    await bbLoad();
}

// ── Entry points ──────────────────────────────────────────────────────────────
// Reached ONLY from the World Dashboard (§5 two-tier navigation). initBondedBranding exists
// as the embed fallback the dashboard uses when the standalone opener is unavailable.
export async function openBondedBranding() {
    bbBuild();
    if (!bbState.overlay) return;
    if (bbState.panelEl && !bbState.overlay.contains(bbState.panelEl)) {
        bbState.overlay.appendChild(bbState.panelEl);
    }
    if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
    bbState.overlay.style.display = 'flex';
    bbStatus('Reading your assets and entities…', false);
    await bbLoad();
}

export async function initBondedBranding() {
    const host = document.getElementById('wd-branding');
    bbBuild();
    if (host && bbState.overlay) {
        bbState.panelEl = bbState.overlay.querySelector('.bb-panel');
        if (bbState.panelEl) host.appendChild(bbState.panelEl);
    }
    await bbLoad();
}

window.openBondedBranding = openBondedBranding;
window.initBondedBranding = initBondedBranding;
window.bbMint = bbMint;
window.bbBind = bbBind;
window.bbUnbind = bbUnbind;
window.bbFilterKind = bbFilterKind;
window.bbCvsApply = bbCvsApply;
window.bbCvsClear = bbCvsClear;

// ── §10.6 App slides + the placeholder pack ──────────────────────────────────
// The studio is the ONE theming surface, so the app's own slides are themed HERE: pick a slide, pick
// one of YOUR OWN bonded assets, apply. The pack is also purchasable here — and every price is the
// server's (never re-declared), so the button cannot mis-state what it costs.
let bbShop = null;        // the served placeholder catalogue
let bbShopPrice = 0;      // micro-$VBV per frame, as served

// bbLoadSlides reads the pack (PUBLIC — a visitor may see what a purchase would buy) and the
// viewer's slide surface, then renders the section. Both reads are fault-tolerant: a refusal is
// stated in the section, never rendered as "you have nothing".
async function bbLoadSlides() {
    try {
        const res = await bbApi('/assets/catalogue');
        bbShop = res;
        bbShopPrice = Number(res.price_micro || 0);
    } catch (e) {
        bbShop = { error: e.message };
    }
    if (window.SlideTheming && window.SlideTheming.refresh) {
        try { await window.SlideTheming.refresh(); } catch (e) { /* SlideTheming states its own error */ }
    }
    bbRenderShop();
    bbRenderSlides();
}

function bbRenderShop() {
    const sel = document.getElementById('bb-shop-sku');
    const ruleEl = document.getElementById('bb-shop-rule');
    const statusEl = document.getElementById('bb-shop-status');
    if (!sel) return;
    const c = bbShop || {};
    if (ruleEl) ruleEl.textContent = c.rule || 'The pack is free to use for theming; the frames themselves are purchasable bonded assets.';
    if (c.error) {
        sel.innerHTML = '<option value="">pack unreadable</option>';
        if (statusEl) statusEl.textContent = 'The pack could not be read (' + c.error + ').';
        return;
    }
    const list = (c.skus || []).filter(function (s) { return s.purchasable; });
    sel.innerHTML = list.length
        ? list.map(function (s) {
            return `<option value="${bbEsc(s.sku)}">${bbEsc(s.name)} — ${bbEsc((Number(s.price_micro || 0) / 1000000).toFixed(2))} VBV${s.starter ? ' (free in the starter pack)' : ''}</option>`;
        }).join('')
        : '<option value="">no purchasable frames</option>';
    if (statusEl) {
        const unavailable = (c.skus || []).filter(function (s) { return !s.purchasable; });
        statusEl.textContent = (c.count || 0) + ' frame(s) shipped, ' + (c.purchasable || 0) +
            ' purchasable at ' + (bbShopPrice / 1000000).toFixed(2) + ' VBV each' +
            (unavailable.length ? ' (' + unavailable.length + ' frame(s) cannot be a bonded asset — see the reason in the pack list)' : '') + '.';
    }
}

// slideAssetOptions lists the wallet's OWN assets for one slide. The server refuses anything else,
// so the select never offers art the player does not own.
function slideAssetOptions(slot) {
    if (!bbState.assets.length) return '<option value="">no bonded assets yet</option>';
    return bbState.assets.map(function (a) {
        const sel = (slot && slot.asset_id === a.asset_id) ? ' selected' : '';
        return `<option value="${bbEsc(a.asset_id)}"${sel}>${bbEsc(a.name)}${a.source ? ' (' + bbEsc(a.source) + ')' : ''}</option>`;
    }).join('');
}

function bbRenderSlides() {
    const host = document.getElementById('bb-slide-slots');
    const ruleEl = document.getElementById('bb-slide-rule');
    const statusEl = document.getElementById('bb-slide-status');
    if (!host) return;
    const st = (window.SlideTheming && window.SlideTheming.describe) ? window.SlideTheming.describe() : null;
    const pol = (st && st.policy) || {};
    if (ruleEl) ruleEl.textContent = pol.rule ||
        'Each slide can wear one of your own bonded assets; with none it shows the pack default.';
    if (!bbWallet()) {
        host.innerHTML = '<p class="bb-empty">Connect a wallet to re-theme the screens you are looking at.</p>';
        if (statusEl) statusEl.textContent = '';
        return;
    }
    const slots = (window.SlideTheming && window.SlideTheming.slotViews) ? window.SlideTheming.slotViews() : [];
    if (!slots.length) {
        host.innerHTML = '<p class="bb-empty">The slide list could not be read.</p>';
        return;
    }
    host.innerHTML = slots.map(function (s) {
        const art = s.art_uri
            ? `<img class="bb-thumb" src="${bbEsc(s.art_uri)}" alt="slide art" loading="lazy" />`
            : '<span class="bb-nomedia">no art</span>';
        return `<div class="bb-target">
          <div class="bb-target-head"><strong>${bbEsc(s.label || s.slot)}</strong>
            <small class="bb-where">${bbEsc(s.show)} &middot; ${bbEsc(s.active ? 'your asset' : 'pack default')}</small></div>
          <div class="bb-worn-row">${art}</div>
          <div class="bb-target-actions">
            <select id="bb-sl-${bbEsc(s.slot)}">${slideAssetOptions(s)}</select>
            <button class="vbt-btn bb-primary" onclick="window.bbSlideApply('${bbEsc(s.slot)}')">Apply</button>
            <button class="vbt-btn" onclick="window.bbSlideClear('${bbEsc(s.slot)}')">Off</button>
          </div></div>`;
    }).join('');
    if (!statusEl) return;
    if (st && st.error) {
        // A refused read is REPORTED: it is never rendered as "no theme".
        statusEl.textContent = 'Your slide theming could not be read (' + st.error + ').';
    } else {
        statusEl.textContent = (st ? st.themed : 0) + ' of ' + (st ? st.slots : 0) +
            ' slide(s) wear your own art; the rest show the pack default.' +
            (st && st.stale && st.stale.length ? ' Reverted: ' + st.stale.join('; ') + '.' : '');
    }
}

// ── Slide + shop actions ─────────────────────────────────────────────────────
// §10.8 MARKET (player-to-player). Every rule is the SERVER's: the fee, the price range and the
// "a purchase is an acquisition" exemption from the creation cap are rendered from the served
// payload, so this client never re-declares a price or a policy. A typed price is converted to
// integer micro-$VBV WITHOUT float arithmetic (the ledger is integers), and a refused action is
// REPORTED, never silently dropped.

// bbParseVbvMicro turns a typed "$VBV" amount into micro units, exactly: the string is parsed as
// digits plus at most six decimal places, so "12.5" is 12_500_000 micro with no float anywhere.
// Returns null when the text is not money.
function bbParseVbvMicro(text) {
    const m = /^(\d{1,15})(?:\.(\d{1,6}))?$/.exec(String(text == null ? '' : text).trim());
    if (!m) return null;
    const whole = Number(m[1]);
    const frac = m[2] ? Number((m[2] + '000000').slice(0, 6)) : 0;
    return whole * 1000000 + frac;
}

function bbMarketAssetOptions() {
    const list = (bbState.assets || []).filter(function (a) { return a && a.asset_id; });
    if (!list.length) return '<option value="">no bonded assets yet</option>';
    return list.map(function (a) {
        return `<option value="${bbEsc(a.asset_id)}">${bbEsc(a.name || a.asset_id)}${a.sku ? ' (pack ' + bbEsc(a.sku) + ')' : ''}</option>`;
    }).join('');
}

function bbMarketRow(row) {
    const mine = !!row.mine;
    const status = String(row.status || '');
    const art = row.media && row.media.uri
        ? `<img class="bb-thumb" src="${bbEsc(bbSafeUri(row.media.uri))}" alt="listing art" loading="lazy" />`
        : '<span class="bb-nomedia">no art</span>';
    const price = row.price_micro != null ? (Number(row.price_micro) / 1000000).toFixed(2) + ' VBV' : '';
    const net = row.net_micro != null ? (Number(row.net_micro) / 1000000).toFixed(2) + ' VBV' : '';
    let actions = '';
    if (status === 'active' && !row.stale_reason) {
        actions = mine
            ? `<button class="vbt-btn" onclick="window.bbCancelListing('${bbEsc(row.listing_id)}')">Cancel</button>`
            : `<button class="vbt-btn bb-primary" onclick="window.bbBuyListing('${bbEsc(row.listing_id)}')">Buy</button>`;
    }
    const why = row.stale_reason
        ? `<small class="bb-where">cannot be bought: ${bbEsc(row.stale_reason)}</small>`
        : (row.note ? `<small class="bb-where">${bbEsc(row.note)}</small>` : '');
    return `<div class="bb-target">
      <div class="bb-target-head"><strong>${bbEsc(row.asset_name || row.asset_id)}</strong>
        <small class="bb-where">${bbEsc(status)}${row.seller ? ' · seller ' + bbEsc(String(row.seller).slice(0, 10)) + '…' : ''}${mine ? ' · yours' : ''}</small></div>
      <div class="bb-worn-row">${art}</div>
      <div class="bb-target-actions">
        <span class="bb-hint">price ${bbEsc(price)}${net ? ' · seller nets ' + bbEsc(net) : ''}</span>
        ${actions}
      </div>
      ${why}
    </div>`;
}

function bbRenderMarket() {
    const host = document.getElementById('bb-market-listings');
    const ruleEl = document.getElementById('bb-market-rule');
    const statusEl = document.getElementById('bb-market-status');
    const assetSel = document.getElementById('bb-market-asset');
    if (!host) return;
    const view = bbState.market || {};
    if (assetSel) assetSel.innerHTML = bbMarketAssetOptions();
    if (ruleEl) {
        const fee = view.fee_bps != null ? view.fee_bps : '?';
        ruleEl.textContent = 'Sell any bonded asset you own and hold; only you may cancel the offer. ' +
            'The house takes ' + fee + ' bps of the price and the rest goes to the seller. ' +
            (view.acquisition_rule || '');
    }
    if (bbState.marketError) {
        // A refused read is STATED: never rendered as an empty market.
        host.innerHTML = '<p class="bb-empty">The market could not be read (' + bbEsc(bbState.marketError) + ').</p>';
    } else {
        const rows = (view.listings || []);
        host.innerHTML = rows.length
            ? rows.map(bbMarketRow).join('')
            : '<p class="bb-empty">Nothing is for sale yet. List one of your own assets above.</p>';
    }
    if (!statusEl) return;
    const parts = [];
    if (view.active_count != null) parts.push(view.active_count + ' active');
    if (view.sold_count) parts.push(view.sold_count + ' sold');
    if (view.mine_count) parts.push(view.mine_count + ' yours');
    if (view.stale_count) parts.push(view.stale_count + ' stale (' + (view.stale || []).join('; ') + ')');
    if (!bbWallet()) parts.push('connect a wallet to sell');
    statusEl.textContent = parts.join(' · ');
}

async function bbLoadMarket(force) {
    if (bbState.market && !force) { bbRenderMarket(); return bbState.market; }
    try {
        const res = await bbApi('/assets/market');
        bbState.market = res;
        bbState.marketError = '';
    } catch (e) {
        bbState.marketError = e.message;
    }
    bbRenderMarket();
    return bbState.market;
}

async function bbListItem() {
    const sel = document.getElementById('bb-market-asset');
    const priceEl = document.getElementById('bb-market-price');
    const assetId = sel ? String(sel.value || '') : '';
    const micro = bbParseVbvMicro(priceEl ? priceEl.value : '');
    if (!assetId) { bbStatus('Pick one of your own bonded assets to sell.', true); return; }
    if (micro === null) { bbStatus('Enter a price as a number of $VBV (up to six decimal places).', true); return; }
    try {
        const res = await bbApi('/assets/market/list', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            // ONLY asset_id + price_micro: the server refuses any other field, so this client cannot
            // name a seller, a status or a fee.
            body: JSON.stringify({ asset_id: assetId, price_micro: micro }),
        });
        bbStatus('Listed ' + assetId + ' for ' + (res.listing ? (Number(res.listing.price_micro) / 1000000).toFixed(2) : '?') + ' VBV.', false);
    } catch (e) {
        bbStatus('Listing refused: ' + e.message, true);
    }
    await bbLoadMarket(true);
}

async function bbCancelListing(listingId) {
    try {
        await bbApi('/assets/market/cancel', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ listing_id: listingId }),
        });
        bbStatus('Listing cancelled.', false);
    } catch (e) {
        bbStatus('Cancel refused: ' + e.message, true);
    }
    await bbLoadMarket(true);
}

async function bbBuyListing(listingId) {
    try {
        const res = await bbApi('/assets/market/buy', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            // ONLY the listing id: the price is the listing's, so it can never be named here.
            body: JSON.stringify({ listing_id: listingId }),
        });
        bbStatus('Bought ' + (res.asset && res.asset.name ? res.asset.name : listingId) + ' — it is in your chest now.', false);
        // The chest changed, so re-read it: the studio is one surface and must not show stale art.
        try {
            const chest = await bbApi('/assets');
            bbState.assets = Array.isArray(chest.assets) ? chest.assets : bbState.assets;
            bbState.bindings = Array.isArray(chest.bindings) ? chest.bindings : bbState.bindings;
            bbState.capacity = chest.capacity || bbState.capacity;
            bbRenderAssets();
        } catch (e) { /* the purchase succeeded; the chest re-read is reported by the studio status */ }
    } catch (e) {
        bbStatus('Purchase refused: ' + e.message, true);
    }
    await bbLoadMarket(true);
}

async function bbSlideApply(slot) {
    const sel = document.getElementById('bb-sl-' + slot);
    const assetId = sel ? String(sel.value || '') : '';
    if (!assetId) {
        bbStatus('Pick one of your own bonded assets first — the free starter pack already granted you the frames these slides show.', true);
        return;
    }
    try {
        await window.SlideTheming.setSlot(slot, assetId);
        bbStatus('Slide ' + slot + ' now wears ' + assetId + '.', false);
    } catch (e) {
        bbStatus('Slide theming refused: ' + e.message, true);
    }
    bbRenderSlides();
}

async function bbSlideClear(slot) {
    try {
        await window.SlideTheming.clearSlot(slot);
        bbStatus('Slide ' + slot + ' is back to the pack default.', false);
    } catch (e) {
        bbStatus('Could not clear that slide: ' + e.message, true);
    }
    bbRenderSlides();
}

// bbClaimStarter takes the FREE starter pack. The server is idempotent, so pressing it twice is
// harmless; the status line reports exactly what happened (including anything it had to skip).
async function bbClaimStarter() {
    if (!window.SlideTheming) return;
    try {
        const res = await window.SlideTheming.ensureStarter();
        if (!res) { bbStatus('Connect a wallet first.', true); return; }
        if (res.error) { bbStatus('Starter pack refused: ' + res.error, true); return; }
        bbStatus('Starter pack: ' + res.granted + ' frame(s) granted, ' + res.already + ' already yours' +
            (res.skipped && res.skipped.length ? '; skipped ' + res.skipped.join('; ') : '') + '.', false);
        await bbLoad();
    } catch (e) {
        bbStatus('Starter pack failed: ' + e.message, true);
    }
}

// bbBuySku buys the selected frame at the SERVED price (the body carries only the sku, so a client
// cannot declare a price).
async function bbBuySku() {
    const sel = document.getElementById('bb-shop-sku');
    const sku = sel ? String(sel.value || '') : '';
    if (!sku) { bbStatus('Pick a frame from the pack first.', true); return; }
    if (bbState.busy) return;
    bbState.busy = true;
    try {
        const res = await bbApi('/assets/purchase', {
            method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ sku: sku }),
        });
        bbStatus('Bought ' + sku + ' for ' + (Number(res.price_micro || 0) / 1000000).toFixed(2) + ' VBV.', false);
        await bbLoad();
    } catch (e) {
        bbStatus('Purchase refused: ' + e.message, true);
    } finally { bbState.busy = false; }
}

window.bbSlideApply = bbSlideApply;
window.bbSlideClear = bbSlideClear;
window.bbClaimStarter = bbClaimStarter;
window.bbBuySku = bbBuySku;
window.bbRenderSlides = bbRenderSlides;
// ── §27.6 theme slots ────────────────────────────────────────────────────────
// /api/theme/bind and /api/theme/lock are the two doors that had no UI in the entire client. They
// are the §27.6 "removed from play" control: a bound asset contributes its cosmetic MoodTag to the
// wallet's theme, and a LOCKED one is excluded (BondedAssetRegistry.MoodTagForWallet), so the
// control changes a real, served outcome rather than only setting a flag.
function bbRenderThemeSlots() {
    const rule = document.getElementById('bb-ts-rule');
    const sel = document.getElementById('bb-ts-asset');
    const host = document.getElementById('bb-ts-bindings');
    if (rule) {
        rule.textContent = 'A theme slot binding is keyed <asset>:<slot>. Locking a binding takes the ' +
            'asset out of play for the theme (its MoodTag stops counting). The slot label may not ' +
            'contain ":" — that key shape belongs to an entity target binding.';
    }
    if (sel) sel.innerHTML = bbMarketAssetOptions();
    if (!host) return;
    // Only slot bindings (no target) belong to this section; entity bindings are shown above.
    const slots = bbState.bindings.filter(b => b.slot && !b.target_kind);
    host.innerHTML = slots.length
        ? slots.map(b => `<div class="bb-target-row"><span class="bb-target-name">${bbEsc(b.asset_id)} → ${bbEsc(b.slot)}</span><span class="bb-target-state">${b.locked ? 'LOCKED (out of play)' : 'in play'}</span></div>`).join('')
        : '<p class="bb-empty">No theme slots bound yet.</p>';
}

function bbThemeSel() {
    const sel = document.getElementById('bb-ts-asset');
    return sel ? String(sel.value || '') : '';
}

function bbThemeSlot() {
    const el = document.getElementById('bb-ts-slot');
    return el ? String(el.value || '').trim() : '';
}

async function bbThemeBind() {
    const assetId = bbThemeSel();
    const slot = bbThemeSlot();
    if (!assetId) { bbThemeStatus('Pick one of your own bonded assets.', true); return; }
    if (!slot) { bbThemeStatus('Enter a slot label.', true); return; }
    if (slot.indexOf(':') >= 0) { bbThemeStatus('A slot label may not contain ":".', true); return; }
    try {
        await bbApi('/theme/bind', {
            method: 'POST', headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ asset_id: assetId, slot: slot }),
        });
        bbThemeStatus('Bound ' + assetId + ' → ' + slot + '.');
        await bbLoad();
        bbRenderThemeSlots();
    } catch (e) { bbThemeStatus('Bind refused: ' + e.message, true); }
}

async function bbThemeLockToggle(locked) {
    const assetId = bbThemeSel();
    const slot = bbThemeSlot();
    if (!assetId || !slot) { bbThemeStatus('Pick an asset and the slot to ' + (locked ? 'lock' : 'unlock') + '.', true); return; }
    try {
        // `locked` is the ONLY thing this route sets — there is no "unlock" route, so an unlock is
        // stated as unavailable rather than faked by a client-side flag.
        if (!locked) { bbThemeStatus('The engine exposes lock only; a locked binding is not unlocked by any served route.', true); return; }
        await bbApi('/theme/lock', {
            method: 'POST', headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ asset_id: assetId, slot: slot }),
        });
        bbThemeStatus('Locked ' + assetId + ':' + slot + ' — out of play for your theme.');
        await bbLoad();
        bbRenderThemeSlots();
    } catch (e) { bbThemeStatus('Lock refused: ' + e.message, true); }
}

function bbThemeStatus(msg, bad) {
    const el = document.getElementById('bb-ts-status');
    if (el) { el.textContent = msg || ''; el.classList.toggle('bad', !!bad); }
}

window.bbThemeBind = bbThemeBind;
window.bbThemeLock = function () { return bbThemeLockToggle(true); };
window.bbThemeUnlock = function () { return bbThemeLockToggle(false); };
window.bbRenderThemeSlots = bbRenderThemeSlots;

// §10.8 market entry points (the studio's buttons call these).
window.bbLoadMarket = bbLoadMarket;
window.bbListItem = bbListItem;
window.bbCancelListing = bbCancelListing;
window.bbBuyListing = bbBuyListing;




