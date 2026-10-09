// ============================================================================
// dev_game_hub.js — Dev/Game Hub framework CATALOGUE (§25.5 / developer game-hub)
// ----------------------------------------------------------------------------
// The dev-game-hub catalogues the app's base framework functions so a dev-user can
// see what the engine composes. It is a READ-ONLY CATALOGUE — there is no purchase
// door.
//
// MEASURED 2026-09-15 (the fix, not an assumption): this file used to
// `setStatus('✅ Purchased …')`, add the power and mark the row Owned — while
// charging nothing, sending NOTHING to the server and losing all of it the moment
// the panel closed. Evidence: the file contains no `fetch`, no `/api/` call and no
// persistence of any kind, and NO Go file references a single catalogue id
// (`mech_region_vitality`, `econ_shop_token`, `match_autonomous`, …) — the whole
// inventory existed only in this client. A shop that reports a purchase that did
// not happen is worse than one that says it is closed, so the panel now STATES why
// nothing here can be bought.
//
// The `priceMicro` values are uint64 micro and express the CONFIG-side intent for a
// future server-owned catalogue. They are NOT chargeable: a price becomes a price
// only when a server door charges it through the one sink and records ownership
// (Industrial Loop). Until then no client may present one as payable.
// ============================================================================
(function () {
    'use strict';

    var API_BASE = '/api';
    const MICRO = 1000000;

    // --- Catalog of framework functions (the "shop inventory") ---
    // Each entry: { id, name, desc, category, priceMicro, power }
    const CATALOG = [
        // Composable Mechanics (§25.5)
        { id: 'mech_region_vitality', name: 'Region Vitality', desc: 'More AI citizens + caches => richer region. Stackable, repeatable.', category: 'mechanics', priceMicro: 5000000, power: 10 },
        { id: 'mech_club_mojo', name: 'Club Mojo Field', desc: 'Mojo boosts regional coherence. Stackable, repeatable per tick.', category: 'mechanics', priceMicro: 3000000, power: 8 },
        { id: 'mech_rivalry_aura', name: 'Rivalry Aura', desc: 'Contested regions emit a rivalry field. Once-per-resolve.', category: 'mechanics', priceMicro: 8000000, power: 15 },
        { id: 'mech_theme_gravity', name: 'Theme Gravity', desc: 'AI-citizen population gravity pulls players (§27.3).', category: 'mechanics', priceMicro: 6000000, power: 12 },
        { id: 'mech_career_ripple', name: 'Career XP Ripple', desc: 'Career progress emits local vitality. Stackable.', category: 'mechanics', priceMicro: 4000000, power: 9 },

        // Economy & Ledger (§1)
        { id: 'econ_shop_token', name: 'Token Resolution', desc: 'Category to token preset resolver. Locks shops to their token.', category: 'economy', priceMicro: 2000000, power: 5 },
        { id: 'econ_vault_donation', name: 'Vault Donation', desc: 'Players donate $VBV to House vault. Credits faucet.', category: 'economy', priceMicro: 1000000, power: 3 },
        { id: 'econ_district_tax', name: 'District Tax', desc: 'Governor sets localized sales tax (0-20%).', category: 'economy', priceMicro: 7000000, power: 13 },

        // Clubs / Territories / Regions (§2)
        { id: 'club_create', name: 'Club Foundry', desc: 'Found a club (1 territory, 5,000 VBV).', category: 'clubs', priceMicro: 5000000, power: 10 },
        { id: 'club_purchase_territory', name: 'Territory Purchase', desc: 'Acquire unclaimed district (2,500 VBV on-chain).', category: 'clubs', priceMicro: 2500000, power: 7 },
        { id: 'club_regional_manager', name: 'Regional Manager', desc: 'Player-initiated governor ability (2+ territories).', category: 'clubs', priceMicro: 8000000, power: 16 },

        // AI Citizens (§3 / §15)
        { id: 'citizen_spawn', name: 'Spawn AI Citizen', desc: 'Dedicated 0xai wallet (never personal). Guard enforced.', category: 'citizens', priceMicro: 3000000, power: 8 },
        { id: 'citizen_invest', name: 'Entity Investment', desc: 'Citizen invests from OWN wallet.', category: 'citizens', priceMicro: 4000000, power: 9 },
        { id: 'citizen_business', name: 'Spawn Business', desc: 'Citizen spawns NPC business (Journeyman+).', category: 'citizens', priceMicro: 6000000, power: 12 },

        // Matchmaking / Combat (§4)
        { id: 'match_queue', name: 'Matchmaking Queue', desc: 'Join/leave matchmaking pool.', category: 'combat', priceMicro: 2000000, power: 5 },
        { id: 'match_challenge', name: 'PvP Challenge', desc: 'Challenge another player to battle.', category: 'combat', priceMicro: 3000000, power: 7 },
        { id: 'match_autonomous', name: 'Autonomous Tournament', desc: '15m ticker auto-hosts tournaments.', category: 'combat', priceMicro: 10000000, power: 20 },

        // Justice / Underworld (§5)
        { id: 'justice_bounty', name: 'Bounty Board', desc: 'Real-time tracking of high-Wanted players.', category: 'justice', priceMicro: 5000000, power: 11 },
        { id: 'justice_truth_serum', name: 'Truth Serum', desc: 'Reveals opponent card buff/debuff state.', category: 'justice', priceMicro: 2500000, power: 6 },
        { id: 'underworld_contract', name: 'Underworld Contracts', desc: 'Kidnap, heist, sabotage, launder.', category: 'underworld', priceMicro: 7000000, power: 14 },
        { id: 'underworld_kidnap', name: 'Kidnap Gambit', desc: 'Multi-slot victim registry, ransom economy.', category: 'underworld', priceMicro: 8000000, power: 16 },
        { id: 'underworld_cyber', name: 'Cyber Ops', desc: 'Cyber-Audit / Cyber-Lock / Cyber-Counter.', category: 'underworld', priceMicro: 6000000, power: 13 },

        // Markets / Shops (§7)
        { id: 'market_global_shop', name: 'Global Shop Registry', desc: '6 shop categories (Elemental/Tactical/Vitality/Hardware/Nugget/Unit).', category: 'markets', priceMicro: 4000000, power: 9 },
        { id: 'market_creator', name: 'Creator Store', desc: 'Create, list, buy, rate, review products.', category: 'markets', priceMicro: 5000000, power: 11 },
        { id: 'market_black', name: 'Black Market', desc: 'Fence stolen cards, kidnapped assets.', category: 'markets', priceMicro: 7000000, power: 14 },

        // World / Events (§8)
        { id: 'world_treasure', name: 'Treasure Caches', desc: 'Spawn and claim treasure caches.', category: 'world', priceMicro: 3000000, power: 7 },
        { id: 'world_seasonal', name: 'Seasonal Engine', desc: 'Seasonal events with rewards.', category: 'world', priceMicro: 6000000, power: 12 },
        { id: 'world_3d', name: '3D World Explorer', desc: 'Three.js explorer + spectator cycle.', category: 'world', priceMicro: 12000000, power: 25 },

        // Pet World / Entities (§30/§31)
        { id: 'pet_stats', name: 'Entity Stats', desc: '6 uint64 stats on PetNFT + AICitizen.', category: 'pets', priceMicro: 5000000, power: 11 },
        { id: 'pet_power_overlay', name: 'Power Overlay', desc: 'Caps effective level to stat ceiling (600).', category: 'pets', priceMicro: 8000000, power: 17 },
        { id: 'pet_breed', name: 'Pet Breeding', desc: 'Breed pets with cert-gating + trait skew.', category: 'pets', priceMicro: 4000000, power: 9 },
        { id: 'pet_events', name: 'Entity Events', desc: 'Train/reward/punish events + payout loop.', category: 'pets', priceMicro: 9000000, power: 19 },

        // Rivalry / Synergy / Faith (§9)
        { id: 'rivalry_matrix', name: 'Rivalry Matrix', desc: '12-pair rivalry system with XP modifiers.', category: 'rivalry', priceMicro: 10000000, power: 22 },
        { id: 'synergy_combined', name: 'Synergy Combined', desc: 'Household combined events (owner + pets + bots).', category: 'rivalry', priceMicro: 7000000, power: 14 },
        { id: 'faith_coherence', name: 'Faith Coherence', desc: 'Region rituals raise FaithCoherence.', category: 'faith', priceMicro: 6000000, power: 13 },
        { id: 'faith_war_gambit', name: 'Faith War Gambit', desc: 'Religious card battles with pot stakes.', category: 'faith', priceMicro: 8000000, power: 17 },

        // Identity / Admin (§11)
        { id: 'identity_bridge', name: 'Identity Bridge', desc: 'Cross-platform identity with linked wallets.', category: 'identity', priceMicro: 3000000, power: 7 },
        // NOTE: the Admin Suite is deliberately NOT purchasable. It is a signature-gated operator
        // surface (World Dashboard ▸ System & Ops ▸ Admin Console), not an in-world feature: an
        // admin surface that can be bought is not an authority. Its only gate is ADMIN_WALLETS.
        { id: 'local_model', name: 'Local Model Promotion', desc: 'Per-user ornith model (§24.5/§24.6).', category: 'identity', priceMicro: 5000000, power: 11 },
    ];

    // --- State ---
    // No `purchased` / `totalPower` arrays: with no purchase door there is nothing to
    // track, and a local "receipt" is precisely the lie this panel used to tell.
    let state = {
        _init: false,
        category: 'all',
    };

    let overlayEl = null;
    let catalogEl = null;
    let statusEl = null;
    let powerEl = null;

    function init() {
        if (state._init) return;
        state._init = true;
        const html = `
<div id="dev-game-hub-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel dev-game-hub-panel">
        <button id="btn-close-dgh" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>🛠️ Dev/Game Hub — Function Shop</h2>
        <p style="color:#b0bec5;font-size:12px;">The app's base framework functions, <b>catalogue only</b>.
        <b style="color:#ffb74d;">Nothing here is purchasable yet</b> — no server purchase door exists, so no price
        is charged and no ownership is recorded.</p>
        <div class="dgh-stats">
            <span class="dgh-stat">⚡ Total Power: <b id="dgh-total-power">0</b></span>
            <span class="dgh-stat">📦 Owned: <b id="dgh-owned">0</b> / ${CATALOG?.length ?? 0}</span>
        </div>
        <div class="dgh-categories">
            <button class="vbt-btn vbt-btn-secondary dgh-cat active" data-cat="all">All</button>
            <button class="vbt-btn vbt-btn-secondary dgh-cat" data-cat="mechanics">Mechanics</button>
            <button class="vbt-btn vbt-btn-secondary dgh-cat" data-cat="economy">Economy</button>
            <button class="vbt-btn vbt-btn-secondary dgh-cat" data-cat="clubs">Clubs</button>
            <button class="vbt-btn vbt-btn-secondary dgh-cat" data-cat="citizens">Citizens</button>
            <button class="vbt-btn vbt-btn-secondary dgh-cat" data-cat="combat">Combat</button>
            <button class="vbt-btn vbt-btn-secondary dgh-cat" data-cat="justice">Justice</button>
            <button class="vbt-btn vbt-btn-secondary dgh-cat" data-cat="underworld">Underworld</button>
            <button class="vbt-btn vbt-btn-secondary dgh-cat" data-cat="markets">Markets</button>
            <button class="vbt-btn vbt-btn-secondary dgh-cat" data-cat="world">World</button>
            <button class="vbt-btn vbt-btn-secondary dgh-cat" data-cat="pets">Pets</button>
            <button class="vbt-btn vbt-btn-secondary dgh-cat" data-cat="rivalry">Rivalry</button>
            <button class="vbt-btn vbt-btn-secondary dgh-cat" data-cat="faith">Faith</button>
            <button class="vbt-btn vbt-btn-secondary dgh-cat" data-cat="identity">Identity</button>
            <button class="vbt-btn vbt-btn-secondary dgh-cat" data-cat="admin">Admin</button>
        </div>
        <div id="dgh-catalog" class="dgh-catalog"></div>
        <p id="dgh-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('dev-game-hub-overlay');
        catalogEl = document.getElementById('dgh-catalog');
        statusEl = document.getElementById('dgh-status');
        powerEl = document.getElementById('dgh-total-power');
        document.getElementById('btn-close-dgh').addEventListener('click', () => { overlayEl.style.display = 'none'; });
        document.querySelectorAll('.dgh-cat').forEach(btn => {
            btn.addEventListener('click', () => {
                document.querySelectorAll('.dgh-cat').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                state.category = btn.dataset.cat;
                render();
            });
        });
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>\"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }

    function fmtVBV(micro) { return (parseFloat(micro) / MICRO).toFixed(4); }

    function render() {
        if (!catalogEl) return;
        const items = state.category === 'all' ? CATALOG : CATALOG.filter(i => i.category === state.category);
        catalogEl.innerHTML = items.map(i => {
            return `<div class="dgh-item">
                <div class="dgh-item-head">
                    <span class="dgh-item-name">${esc(i?.name)}</span>
                    <span class="dgh-item-power">⚡ ${i.power}</span>
                </div>
                <div class="dgh-item-desc">${esc(i.desc)}</div>
                <div class="dgh-item-foot">
                    <span class="dgh-item-price">${fmtVBV(i.priceMicro)} $VBV <span style="color:#90a4ae;">· not chargeable yet</span></span>
                    <button class="vbt-btn vbt-btn-secondary dgh-buy" data-id="${i.id}">Why not?</button>
                </div>
            </div>`;
        }).join('');
        catalogEl.querySelectorAll('.dgh-buy').forEach(btn => {
            btn.addEventListener('click', () => explainNotPurchasable(btn.dataset.id));
        });
    }

    // A framework function has NO purchase door, so this explains rather than sells.
    // It replaces a stub that reported success: `state.purchased.push(id)`,
    // `state.totalPower += item.power`, a row re-rendered as "✅ OWNED" and
    // `setStatus('✅ Purchased …')` — with no request, no charge and nothing
    // recorded anywhere. Nothing here mutates state, because no purchase occurred.
    function explainNotPurchasable(id) {
        const item = CATALOG.find(i => i.id === id);
        if (!item) return;
        setStatus('⏳ ' + (item.name || id) + ' cannot be purchased yet: the catalogue is a client-side ' +
            'inventory and no server door exists to charge its price or record ownership. ' +
            'A framework function becomes purchasable when a server route owns it.');
        if (window.showToast) window.showToast('Not purchasable yet — see the reason in the panel', 'info');
    }

    // The catalogue can hand out NOTHING today, so "owned" is structurally 0 — not a
    // number that merely happens to be 0 until the next refresh.
    function updateStats() {
        if (powerEl) powerEl.textContent = '0';
        const ownedEl = document.getElementById('dgh-owned');
        if (ownedEl) ownedEl.textContent = '0';
    }

    function setStatus(m) { if (statusEl) statusEl.textContent = m || ''; }

    window.openDevGameHub = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        render();
        updateStats();
    };

    // Expose catalog for other modules
    window.__devGameHubCatalog = CATALOG;
})();
