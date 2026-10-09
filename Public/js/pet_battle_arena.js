// ============================================================================
// pet_battle_arena.js — §30 BONDED ARENAS (3D-WORLD PLACEHOLDERS)
// ----------------------------------------------------------------------------
// Pet bouts and vehicle races/dogfights are NOT a card-game subsystem. Their home is the
// 3D world: transport runs and gang-up events run with the owner and the owner's bots.
// This module therefore owns the arena SURFACES as honest placeholders:
//
//   * it never simulates a fight (the previous revision rolled dice client-side — a
//     fabricated system that contradicted the determinism mandate and paid fake $VBV);
//   * it reports the REAL roster power the arena will use, read from the server;
//   * it reports the REAL deck-card bonus those bonded assets already lend the owner
//     today (§26.4.3 / §25.6.2), so the assets already matter in ranked play;
//   * it routes the player into the 3D world, where the arena will be built.
//
// Entry points: window.openPetBattleArena() / window.openVehicleArena()
// Embedders:    window.initPetArena() / window.initVehicleArena()  (World Dashboard tabs)
// ============================================================================

(function () {
    'use strict';

    var API_BASE = '/api';
    var overlayEl = null, bodyEl = null, titleEl = null;
    var focus = 'pets';
    var data = { pets: [], vehicles: [], deckPct: null };

    function wallet() {
        if (typeof window.getActiveWallet === 'function') {
            var w = window.getActiveWallet();
            if (w) return w;
        }
        if (window.currentWallet) return window.currentWallet;
        try {
            if (typeof window.GetGameState === 'function') {
                var st = window.GetGameState();
                if (st && st.wallet) return st.wallet;
            }
        } catch (e) { /* engine not ready */ }
        return window.userAddress || '';
    }

    function esc(s) {
        return String(s == null ? '' : s).replace(/[&<>"']/g, function (c) {
            return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c];
        });
    }

    function statSum(st) {
        st = st || {};
        return (st.speed || 0) + (st.intelligence || 0) + (st.willpower || 0) +
            (st.strength || 0) + (st.charisma || 0) + (st.agility || 0);
    }

    // §30 power overlay, mirroring the server: base level + floor(statSum / PowerOverlayScale),
    // clamped to the 600 ceiling. View-only geometry.
    function power(baseLevel, stats) {
        return Math.min(600, (Number(baseLevel) || 1) + Math.floor(statSum(stats) / 50));
    }

    function api(path) {
        return fetch(API_BASE + path).then(function (r) {
            if (!r.ok) throw new Error('HTTP ' + r.status);
            return r.json();
        });
    }

    function ensureOverlay() {
        if (overlayEl) return;
        document.body.insertAdjacentHTML('beforeend', `
<div id="bonded-arena-overlay" class="overlay vbt-overlay" style="display:none;">
  <div class="neon-glass-panel" style="max-width:860px;">
    <button id="btn-close-bonded-arena" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
    <h2 id="bonded-arena-title">⚔️ Bonded Arena (3D world)</h2>
    <div id="bonded-arena-body" class="la-grid"></div>
  </div>
</div>`);
        overlayEl = document.getElementById('bonded-arena-overlay');
        bodyEl = document.getElementById('bonded-arena-body');
        titleEl = document.getElementById('bonded-arena-title');
        document.getElementById('btn-close-bonded-arena').addEventListener('click', function () {
            overlayEl.style.display = 'none';
        });
    }
    // The shared, honest renderer for both arenas (pet focus / vehicle focus).
    function render(target) {
        target = target || bodyEl;
        if (!target) return;
        var isPet = focus === 'pets';
        var rows = (isPet ? data.pets : data.vehicles).map(function (a) {
            var isVeh = !isPet;
            var name = a.name || (isVeh ? a.vehicle_id : a.pet_id) || '—';
            var pwr = power(isVeh ? a.vehicle_level : a.pet_level, a.stats);
            var badge = a.certified ? ' ✓cert' : (a.black_market_adopted ? ' ⚠black-market' : '');
            var region = a.region ? ' @' + esc(a.region) : ' · stored';
            return `<div class="la-card">
              <h4>${esc(name)}${badge}</h4>
              <p style="font-size:11px;color:#b0bec5;margin:0;">PWR ${pwr} · ${statSum(a.stats)} stat pts${region}</p>
              <p style="font-size:11px;color:#4dd0e1;margin:2px 0 0;">${a.mature ? 'run-ready' : 'not yet run-ready'}</p>
            </div>`;
        }).join('');

        var deckLine = (data.deckPct == null)
            ? 'Deck-card bonus: reading…'
            : (data.deckPct > 0
                ? 'These bonded assets currently add <b>+' + data.deckPct + '%</b> power to every card in your deck.'
                : 'Your bonded assets add no deck-card power yet — groom a companion or fit a part to raise the household stat total.');

        target.innerHTML = `
          <div class="la-card">
            <h4>${isPet ? '🐾 Companion arena' : '🚗 Vehicle arena'} — hosted in the 3D world</h4>
            <p style="font-size:11px;color:#b0bec5;margin:0 0 6px;">
              ${isPet
                ? "Companion bouts are part of the 3D world: transport runs and gang-up events staged with the owner and the owner's bots."
                : "Vehicle races and dogfights are part of the 3D world: transport runs and gang-up events staged with the owner and the owner's bots."}
              The card game stays the card game — no dice-rolling arena is simulated here.
            </p>
            <p style="font-size:11px;color:#4dd0e1;margin:0;">${deckLine}</p>
          </div>
          <div class="la-card">
            <h4>Roster awaiting the 3D arena</h4>
            <div class="la-grid">${rows || '<p style="font-size:12px;color:#90a4ae;">Nothing registered yet.</p>'}</div>
          </div>
          <div class="ai-card-actions">
            <button class="vbt-btn vbt-btn-primary" onclick="window.enter3DWorld && window.enter3DWorld()">🌐 Enter the 3D world</button>
            <button class="vbt-btn vbt-btn-secondary" onclick="window.openLifeAssets && window.openLifeAssets('${isPet ? 'pets' : 'vehicles'}')">Manage assets</button>
          </div>
          <p style="font-size:11px;color:#90a4ae;margin-top:6px;">
            Planned for the 3D arena: transport runs, gang-up events with owner + bots, and
            region-wide spectating. Bonded stats and build levels are already live — the arena
            is the destination, not a second rule set.
          </p>`;
    }

    function load() {
        var w = wallet();
        var qs = w ? ('?wallet=' + encodeURIComponent(w)) : '';
        return Promise.all([
            api('/pets' + qs).catch(function () { return { data: [] }; }),
            api('/vehicles' + qs).catch(function () { return { data: [] }; }),
            api('/owner/combined-stats' + qs).catch(function () { return {}; })
        ]).then(function (res) {
            var pets = Array.isArray(res[0].data) ? res[0].data : [];
            var veh = Array.isArray(res[1].data) ? res[1].data : [];
            data.pets = pets.filter(function (p) { return p.certified && !p.black_market_adopted; });
            data.vehicles = veh.filter(function (v) { return v.certified && !v.black_market_adopted; });
            data.deckPct = (res[2] && res[2].bonded_deck_boost_pct != null) ? Number(res[2].bonded_deck_boost_pct) : null;
            return data;
        });
    }

    var bound = false;
    function bindOnce() {
        if (bound) return; bound = true;
        // Re-render whenever the owner buys/progresses an asset while the panel is open.
        window.addEventListener('focus', function () {
            if (overlayEl && overlayEl.style.display === 'flex') load().then(function () { render(); });
        });
    }
    function present(tab) {
        focus = (tab === 'vehicles') ? 'vehicles' : 'pets';
        ensureOverlay();
        bindOnce();
        titleEl.textContent = (focus === 'pets' ? '⚔️ Companion Arena' : '🚗 Vehicle Arena') + ' — 3D world';
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        bodyEl.innerHTML = '<div class="wd-loading">Reading your bonded roster…</div>';
        load().then(function () { render(); }).catch(function (e) {
            bodyEl.innerHTML = '<p style="color:#90a4ae;">Arena roster unavailable — ' + esc(e.message) + '</p>';
        });
    }

    // World Dashboard embed targets (materialised lazily by the dashboard).
    function initEmbedded(tab) {
        var el = document.getElementById(tab === 'vehicles' ? 'wd-vehicle_arena' : 'wd-pet_arena');
        if (!el) return false;
        focus = (tab === 'vehicles') ? 'vehicles' : 'pets';
        el.innerHTML = '<div class="wd-loading">Reading your bonded roster…</div>';
        load().then(function () { render(el); }).catch(function (e) {
            el.innerHTML = '<p style="color:#90a4ae;">Arena roster unavailable — ' + esc(e.message) + '</p>';
        });
        return true;
    }

    window.openPetBattleArena = function () { present('pets'); };
    // §25.6.1: the vehicle arena is the same 3D destination for the other asset family.
    window.openVehicleArena = function () { present('vehicles'); };
    window.initPetArena = function () { return initEmbedded('pets'); };
    window.initVehicleArena = function () { return initEmbedded('vehicles'); };

})();
