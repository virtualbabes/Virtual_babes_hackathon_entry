// ============================================================================
// pet_breeder.js — §26.4 COMPANION KENNEL (purchase · breeding · grooming)
// ----------------------------------------------------------------------------
// The SINGLE owner of companion progression, embedded by the World Dashboard's `pets`
// tab. Companions are ACCOUNT UPGRADES, so nothing here is free: every price is served
// by GET /api/pets and computed server-side (a client never prices its own upgrade).
//
//   purchase  POST /api/pets/spawn   { name, traits }
//   breed     POST /api/pets/breed   { name, sire_id, dam_id }
//   groom     POST /api/pets/groom   { pet_id, focus }        (§26.4.2 ladder)
//
// Vehicle progression is owned by the Garage (life_assets.js); the arena is a 3D-world
// destination owned by pet_battle_arena.js. One feature, one owner.
// ============================================================================

var API_BASE = '/api';

const KENNEL_AXES = ['SPEED', 'INTELLIGENCE', 'WILLPOWER', 'STRENGTH', 'CHARISMA', 'AGILITY'];
const KENNEL_AXIS_COLOR = {
    SPEED: '#f44336', INTELLIGENCE: '#2196f3', WILLPOWER: '#9c27b0',
    STRENGTH: '#ff9800', CHARISMA: '#e91e63', AGILITY: '#4caf50'
};

let kennel = { pets: [], cal: {}, sire: null, dam: null, root: null, busy: false };

function kennelWallet() {
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

// Presentation-boundary micro-$VBV formatting (no ledger value is ever a float).
function kennelVBV(micro) { return ((Number(micro) || 0) / 1000000).toFixed(2); }
function kennelEsc(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}
function kennelStatSum(st) {
    st = st || {};
    return (st.speed || 0) + (st.intelligence || 0) + (st.willpower || 0) + (st.strength || 0) + (st.charisma || 0) + (st.agility || 0);
}
// §30 power overlay, mirrored from the server formula (base level + floor(statSum/scale)).
function kennelPower(p) {
    const base = Number(p.pet_level || 1);
    const sum = kennelStatSum(p.stats);
    const scale = Number(kennel.cal.power_overlay_scale) || 50;
    return Math.min(600, base + Math.floor(sum / scale));
}

// One-shot retry on HTTP 429, mirroring the Portfolio's rate-limit honesty rule: a refused
// read is retried once, and if it is still refused the caller must SAY so rather than render
// "no companions" (app-entry-mandate §3).
async function kennelApi(path, opts) {
    let resp = await fetch(API_BASE + path, opts);
    if (resp.status === 429) {
        await new Promise(function (r) { setTimeout(r, 450); });
        resp = await fetch(API_BASE + path, opts);
    }
    if (!resp.ok) {
        let d = '';
        try { const j = await resp.json(); d = j.error || j.message || ''; } catch (e) { /* non-JSON */ }
        if (resp.status === 429) throw new Error('rate-limited — could not be read; try again shortly');
        throw new Error(d || ('HTTP ' + resp.status));
    }
    return resp.json();
}

function kennelSetStatus(msg, isError) {
    const el = document.getElementById('kennel-status');
    if (!el) return;
    el.textContent = msg || '';
    el.style.color = isError ? '#ff8a80' : '#80deea';
}

export function initPetBreeder() {
    const el = document.getElementById('wd-pets');
    if (!el) return;
    kennel.root = el;
    el.innerHTML = '<div class="wd-loading">Loading the kennel…</div>';
    kennelLoad();
}

async function kennelLoad() {
    const w = kennelWallet();
    try {
        const res = await kennelApi('/pets' + (w ? '?wallet=' + encodeURIComponent(w) : ''));
        kennel.pets = Array.isArray(res.data) ? res.data : [];
        kennel.cal = {
            spawnFee: res.spawn_fee_micro,
            breedFee: res.breed_fee_micro,
            groomBase: res.groom_base_micro,
            groomMax: res.groom_max_level,
            groomGain: res.groom_stat_gain,
            maturityMs: res.maturity_ms
        };
        // A selected parent may have left the roster (sold/adopted) — drop it.
        if (kennel.sire && !kennel.pets.some(p => p.pet_id === kennel.sire)) kennel.sire = null;
        if (kennel.dam && !kennel.pets.some(p => p.pet_id === kennel.dam)) kennel.dam = null;
        kennelRender();
    } catch (e) {
        kennel.root.innerHTML = '<p style="color:#90a4ae;">Kennel data could not be read — ' + kennelEsc(e.message) + '</p>';
    }
}

// Exposed so a probe (or another panel) can tell whether the Kennel finished loading.
window.kennelReady = function () { return !!(kennel.root && kennel.root.innerHTML.indexOf('kennelBuy()') !== -1); };
function kennelRender() {
    const el = kennel.root;
    if (!el) return;
    const cal = kennel.cal;
    const pets = kennel.pets;
    const mature = pets.filter(p => p.mature).length;
    const w = kennelWallet();

    let html = `
    <div class="la-grid">
      <div class="la-card">
        <h4>🐾 Companion Kennel (§26.4)</h4>
        <p style="font-size:11px;color:#b0bec5;margin:0 0 6px;">
          Companions are <b>account upgrades</b>: purchase, then progress by <b>lineage</b> (breeding)
          and <b>investment</b> (grooming). Every fee is computed by the server and routed to the faucet sink.
        </p>
        <div class="ai-card-actions" style="gap:10px;flex-wrap:wrap;">
          <span style="font-size:11px;color:#4dd0e1;">Purchase ${cal.spawnFee != null ? kennelVBV(cal.spawnFee) + ' $VBV' : '—'}</span>
          <span style="font-size:11px;color:#4dd0e1;">Breeding ${cal.breedFee != null ? kennelVBV(cal.breedFee) + ' $VBV' : '—'}</span>
          <span style="font-size:11px;color:#4dd0e1;">Groom L1 ${cal.groomBase != null ? kennelVBV(cal.groomBase) + ' $VBV' : '—'}</span>
          <span style="font-size:11px;color:#b0bec5;">${pets.length} owned · ${mature} mature</span>
        </div>
        ${w ? '' : '<p style="font-size:11px;color:#ffb74d;margin:6px 0 0;">Connect a wallet to purchase or progress companions.</p>'}
      </div>

      <div class="la-card">
        <h4>➕ Purchase a companion</h4>
        <div class="ai-card-actions" style="flex-wrap:wrap;">
          <input id="kennel-new-name" type="text" maxlength="40" placeholder="Companion name" style="width:180px;" />
          <input id="kennel-new-traits" type="number" min="0" value="0" placeholder="Trait bits" style="width:110px;" />
          <button class="vbt-btn vbt-btn-primary" onclick="window.kennelBuy()">Buy</button>
        </div>
        <p style="font-size:11px;color:#90a4ae;margin:6px 0 0;">Trait bits are deterministic: 2 bits per stat axis (Speed, Intelligence, Willpower, Strength, Charisma, Agility) set the delivered stat floor.</p>
      </div>

      <div class="la-card">
        <h4>🧬 Breeding chamber</h4>
        <div class="ai-card-actions" style="flex-wrap:wrap;">
          <span style="font-size:11px;color:${kennel.sire ? '#80deea' : '#90a4ae'};">Sire: ${kennel.sire ? kennelEsc(kennelShortName(kennel.sire)) : 'none'}</span>
          <span style="font-size:11px;color:${kennel.dam ? '#80deea' : '#90a4ae'};">Dam: ${kennel.dam ? kennelEsc(kennelShortName(kennel.dam)) : 'none'}</span>
          <input id="kennel-offspring-name" type="text" maxlength="40" placeholder="Offspring name" style="width:170px;" />
          <button class="vbt-btn vbt-btn-primary" ${(!kennel.sire || !kennel.dam) ? 'disabled' : ''}
             onclick="window.kennelBreed()">Breed</button>
          <button class="vbt-btn vbt-btn-secondary" onclick="window.kennelClearParents()">Clear</button>
        </div>
        <p style="font-size:11px;color:#90a4ae;margin:6px 0 0;">Only mature, certified companions may breed. The offspring inherits the <b>mean</b> of both parents' stats, so grooming compounds down a bloodline.</p>
      </div>

      <div class="la-card">
        <h4>⚔️ Companion arena</h4>
        <p style="font-size:11px;color:#b0bec5;margin:0 0 6px;">Arena bouts belong to the 3D world (transport runs + gang-up events with the owner's bots). Their power already rides your deck cards in every match today.</p>
        <button class="vbt-btn vbt-btn-secondary" onclick="window.openPetBattleArena()">Open arena status</button>
      </div>
    </div>

    <h4 style="color:#e5e7eb;margin:12px 0 4px;">Your companions</h4>
    <div class="la-grid">${pets.length ? pets.map(kennelPetCard).join('') : '<p style="font-size:12px;color:#90a4ae;">No companions yet — purchase one above to start a bloodline.</p>'}</div>
    <p id="kennel-status" style="font-size:11px;color:#80deea;margin-top:8px;"></p>
    `;
    el.innerHTML = html;
}

function kennelShortName(petId) {
    const p = kennel.pets.find(x => x.pet_id === petId);
    if (!p) return petId;
    return (p.name || '') + ' (' + String(p.pet_id).slice(-6) + ')';
}
function kennelPetCard(p) {
    const st = p.stats || {};
    const sum = kennelStatSum(st);
    const power = kennelPower(p);
    const badge = p.certified ? ' ✓cert' : (p.black_market_adopted ? ' ⚠black-market' : '');
    const region = p.region ? ' @' + kennelEsc(p.region) : '';
    const groom = p.grooming || {};
    const maxLvl = Number(kennel.cal.groomMax) || 10;
    const gain = Number(kennel.cal.groomGain) || 2;
    const base = Number(kennel.cal.groomBase) || 0;
    const breedable = !!(p.mature && p.certified && !p.black_market_adopted);

    const bars = KENNEL_AXES.map(axis => {
        const val = Number(st[axis.toLowerCase()]) || 0;
        const pct = Math.max(0, Math.min(100, val));
        return `<div class="pba-stat-row" style="display:flex;align-items:center;gap:6px;margin:1px 0;">
            <span style="width:74px;font-size:10px;color:#b0bec5;">${axis.slice(0, 4)}</span>
            <div class="pba-stat-bar" style="flex:1;"><div class="pba-stat-fill" style="width:${pct}%;background:${KENNEL_AXIS_COLOR[axis]};"></div></div>
            <span style="width:26px;font-size:10px;color:#80deea;text-align:right;">${val}</span>
        </div>`;
    }).join('');

    const groomRows = KENNEL_AXES.map(axis => {
        const lvl = Number(groom[axis]) || 0;
        const maxed = lvl >= maxLvl;
        const cost = base * (lvl + 1);
        return `<div style="display:flex;align-items:center;gap:6px;margin:2px 0;">
            <span style="width:74px;font-size:10px;color:#4dd0e1;">${axis.slice(0, 4)}</span>
            <span style="width:48px;font-size:10px;color:#b0bec5;">L${lvl}/${maxLvl}</span>
            <span style="flex:1;font-size:10px;color:#90a4ae;">${maxed ? 'maxed' : kennelVBV(cost) + ' $VBV → +' + gain}</span>
            <button class="vbt-btn vbt-btn-secondary" style="font-size:10px;padding:2px 8px;" ${maxed ? 'disabled' : ''}
                onclick="window.kennelGroom('${kennelEsc(p.pet_id)}','${axis}')">Groom</button>
        </div>`;
    }).join('');

    return `<div class="la-card">
      <h4>${kennelEsc(p.name)}${badge}</h4>
      <p style="font-size:11px;color:#b0bec5;margin:0 0 4px;">
        ${String(p.pet_id).slice(0, 12)} · ${p.sire_id ? 'bred' : 'base'}${region} ·
        ${p.mature ? 'mature' : 'immature'} · level L${Number(p.pet_level) || 1}
      </p>
      <p style="font-size:11px;color:#4dd0e1;margin:0 0 6px;">PWR ${power} · ${sum} stat pts${sum ? '' : ' — groom to train'}</p>
      ${bars}
      <details style="margin-top:6px;">
        <summary style="font-size:11px;color:#80deea;cursor:pointer;">Grooming ladder (§26.4.2)</summary>
        <div style="margin-top:4px;">${groomRows}</div>
      </details>
      <div class="ai-card-actions" style="margin-top:6px;">
        ${breedable ? `
          <button class="vbt-btn vbt-btn-secondary" onclick="window.kennelSelectParent('${kennelEsc(p.pet_id)}','sire')">Set sire</button>
          <button class="vbt-btn vbt-btn-secondary" onclick="window.kennelSelectParent('${kennelEsc(p.pet_id)}','dam')">Set dam</button>` :
          '<span style="font-size:10px;color:#90a4ae;">Not eligible to breed (needs mature + certified + legitimate).</span>'}
      </div>
    </div>`;
}
async function kennelBuy() {
    const w = kennelWallet();
    if (!w) { kennelSetStatus('Connect a wallet first.', true); return; }
    const name = (document.getElementById('kennel-new-name')?.value || '').trim();
    const traits = parseInt(document.getElementById('kennel-new-traits')?.value, 10) || 0;
    if (!name) { kennelSetStatus('Name required.', true); return; }
    if (kennel.busy) return;
    kennel.busy = true;
    try {
        await kennelApi('/pets/spawn?wallet=' + encodeURIComponent(w), {
            method: 'POST', headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: name, traits: traits })
        });
        if (window.showToast) window.showToast('🐾 Companion purchased.', 'success');
        await kennelLoad();
        kennelSetStatus('Purchased ' + name + ' for ' + kennelVBV(kennel.cal.spawnFee) + ' $VBV.');
    } catch (e) {
        kennelSetStatus('Purchase failed: ' + e.message, true);
    } finally { kennel.busy = false; }
}

async function kennelBreed() {
    const w = kennelWallet();
    if (!w) { kennelSetStatus('Connect a wallet first.', true); return; }
    if (!kennel.sire || !kennel.dam) { kennelSetStatus('Select BOTH parents.', true); return; }
    const name = (document.getElementById('kennel-offspring-name')?.value || '').trim() || ('Offspring ' + String(Date.now()).slice(-4));
    if (kennel.busy) return;
    kennel.busy = true;
    try {
        await kennelApi('/pets/breed?wallet=' + encodeURIComponent(w), {
            method: 'POST', headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: name, sire_id: kennel.sire, dam_id: kennel.dam })
        });
        kennel.sire = null; kennel.dam = null;
        if (window.showToast) window.showToast('🧬 Offspring born: ' + name, 'success');
        await kennelLoad();
        kennelSetStatus('Bred ' + name + ' for ' + kennelVBV(kennel.cal.breedFee) + ' $VBV.');
    } catch (e) {
        kennelSetStatus('Breeding failed: ' + e.message, true);
    } finally { kennel.busy = false; }
}

async function kennelGroom(petId, focus) {
    const w = kennelWallet();
    if (!w) { kennelSetStatus('Connect a wallet first.', true); return; }
    if (kennel.busy) return;
    kennel.busy = true;
    try {
        const res = await kennelApi('/pets/groom?wallet=' + encodeURIComponent(w), {
            method: 'POST', headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ pet_id: petId, focus: focus })
        });
        await kennelLoad();
        kennelSetStatus(focus + ' groomed — +' + (res.stat_gain != null ? res.stat_gain : '?') +
            ' stat for ' + kennelVBV(res.fee_micro) + ' $VBV (faucet sink).');
    } catch (e) {
        kennelSetStatus('Grooming failed: ' + e.message, true);
    } finally { kennel.busy = false; }
}

function kennelSelectParent(petId, role) {
    if (role === 'sire') kennel.sire = petId; else kennel.dam = petId;
    kennelRender();
    kennelSetStatus('Selected ' + role + ': ' + kennelShortName(petId));
}
function kennelClearParents() { kennel.sire = null; kennel.dam = null; kennelRender(); }

// --- Global handlers (the World Dashboard `pets` tab calls initPetBreeder) ---
window.initPetBreeder = initPetBreeder;
window.kennelBuy = kennelBuy;
window.kennelBreed = kennelBreed;
window.kennelGroom = kennelGroom;
window.kennelSelectParent = kennelSelectParent;
window.kennelClearParents = kennelClearParents;
window.kennelRefresh = kennelLoad;

