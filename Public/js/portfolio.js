// ============================================================================
// portfolio.js — PORTFOLIO: the player's read-only ANALYTICS surface
// ----------------------------------------------------------------------------
// Renamed from player_profile.js (2026-09-14). The surface is not a "profile"
// form — it is an ANALYTICS view of every aspect of a player's standing.
//
// OWNERSHIP / NAVIGATION
//   Reached ONLY from the World Dashboard (`WD_ROUTES.portfolio -> openPortfolio`).
//   The World Dashboard is the single navigation surface (Single-Navigation
//   Mandate). This module has no other entry point.
//
// READ-ONLY MANDATE (.clinerules/app-entry-mandate.md §3)
//   This surface READS and DISPLAYS. It never transacts. No purchase,
//   activation, claim or mutation control may live here. The always-visible
//   `.pp-readonly-note` in the hub chrome states this on every tab.
//
// STRUCTURE
//   TIER 1 — analytics CATEGORIES (sidebar buttons).
//   TIER 2 — analytics SCREENS (that category's sub-tabs).
//   CONTAINMENT: fixed full-viewport shell. The PAGE never scrolls; only the
//   inner panel (.pp-panel) scrolls. `.pp-hub` stays a contained overlay that
//   ui.js hideAllOverlays() closes by class.
//
// DATA SOURCES — all READ-ONLY, all verified-registered routes:
//   GetGameState()                  authoritative WASM snapshot (no network)
//   GET /api/leaderboard            [{wins,dnfs,reputation,best_rating,wallet,…}]
//   GET /api/achievements           {unlocked[],definitions[],progress{}}
//   GET /api/card-stats?id=         {mood,loyalty,fatigue,power[4],scars,…}
//   GET /api/titles                 {titles[{id,name,icon,desc,perk,rarity}]}
//   GET /api/career/progress        career XP + tier
//   GET /api/player/progression     constellation unlock states
//   GET /api/player/tokens          {balance,nugget_balance,unit_balance}
//   GET /api/identity/profile       ReputationProfile
//   GET /api/owner/combined-stats   {combined_stats,effective_power}
//   GET /api/stat-overlay/owner     {overlays[]}
//   GET /api/ai/citizens/list       {data[]}
    //
    // ASSOCIATION & ASSET SOURCES (every one verified REGISTERED + live 200):
    //   GET /api/pets?wallet=             {data[PetNFT]}        §26.4 companions
    //   GET /api/vehicles?wallet=         {data[VehicleNFT], parts[], part_max_level,
    //                                      base_cost_micro, stat_gain, break_in_ms}
    //                                     §25.6.1 upgrade ladder (server-authored calibration)
    //   GET /api/children-bots?wallet=    {children_bots[]}     derived entities
    //   GET /api/garden?wallet=           {elements[],level,meditation_streak}
    //   GET /api/rivalry/state?wallet=    {active_rivals[],pending_invitations[]}
    //   GET /api/rivalry/factions         {factions{}}          faction standing
    //   GET /api/rivalry/list             {data[RegionRivalry]} region rivalries
    //   GET /api/clubs                    {clubs[Club]}         owner/members/alliance
    //   GET /api/assets?wallet=           {assets[BondedAsset]} §23 bonded registry
    //   GET /api/world-content            {data[WorldContentNFT]} §25.7
    //   GET /api/shares/holdings?wallet=  {holdings} | /api/shares/tokens?wallet=
    //   GET /api/invest/portfolio?wallet= EntityPortfolio (investments map)
    //   GET /api/pet-battle/list          {battles[PetBattle]}  §30 arena record
    //   GET /api/church/owner?wallet=     {churches[Church]}    §32
    //   GET /api/faith/religions          {cap,high_tier[Religion]} §32 governance
    //   GET /api/faith/coherence?region=  {faith_coherence,domestic_coherence,…}
    //   GET /api/justice/dashboard        global wanted/bounty view (wallet-less)
    //   GET /api/bounty/active            {bounties[]}
    //   GET /api/loans?wallet=            [Loan] (empty -> [] where there are none)
    //   GET /api/lease/list?wallet=       {leases[Lease]}
    //   GET /api/governance/weight?wallet={weight{…six axes…}}
    //   GET /api/items/collection?wallet= {data[]}
    //   GET /api/identity/events?wallet=  {events[]}
    //   GET /api/compliance/wallet?wallet={records[]}
    //   GET /api/player/associations?wallet=
    //       {present, alliances, active_alliance_id, wanted, wanted_level,
    //        sector_tiles, relationships, moods, buffs, active_buffs,
    //        active_item_buffs, jailed_cards, kidnapped_cards,
    //        held_hostage_cards, captured_outlaws, recovery_bounties,
    //        liquidity_samples[], avg_sustained_micro, mutation_history[],
    //        playstyle, …} — the engine associations that no other read-only
    //        route exposes. `present:false` means the engine holds no
    //        PlayerStats row yet, which is NOT the same as "no associations".
    //
    // ASSOCIATIONS MANDATE
    //   The Portfolio reports EVERY association the engine or a read-only
    //   endpoint holds for the connected wallet: cards, entities, companions,
    //   vehicles, bonded assets, clubs, alliances, rivalries, faith, custody and
    //   obligations. Nothing is inferred: where an association is not exported,
    //   the screen says so explicitly instead of stating a guess as fact.
//
// ECONOMIC INTEGRITY (architecture-ledger.md)
//   Money is carried as uint64 micro-units and converted to a display string
//   ONLY at the presentation boundary (fmtVBV). No float is produced for, or
//   stored on, a ledger value. Bar widths are view-only geometry derived from
//   integer counts.
// ============================================================================
(function () {
    'use strict';

    const HUB_ID = 'portfolio-hub';
    const MICRO = 1000000;
    const DEFAULT_TTL = 15000;   // ms — mirrors the wallet-default rate-limit window

    // --- DOM refs (lazy: the hub is built on first open) ---
    let hubEl = null, sidebarEl = null, subtabsEl = null, panelEl = null, titleEl = null;
    let renderToken = 0;

    const state = {
        activeTab: 'overview',
        activeSub: 'summary',
        cache: new Map(),        // path -> { at, data }
        regions: [],
        regionsLoaded: false,
        activeRegion: null,
        rateLimited: false,      // a read was refused by the rate limiter
    };

    // ========================================================================
    // WALLET — delegated to the ONE authoritative accessor defined in
    // app_bridge.js (backed by GetGameState().wallet). Falls back defensively
    // so the Portfolio still renders if the engine has not booted yet.
    // ========================================================================
    function activeWallet() {
        if (typeof window.getActiveWallet === 'function') {
            try {
                const w = window.getActiveWallet();
                if (w) return String(w);
            } catch (_) { /* engine mid-boot — fall through */ }
        }
        return window.currentWallet || window.userAddress || '';
    }

    // ========================================================================
    // DATA LAYER — read-only JSON with a short TTL cache so switching tabs
    // does not re-hammer the rate limiter.
    // ========================================================================
    async function getJSON(path, ttl) {
        const life = (ttl == null) ? DEFAULT_TTL : ttl;
        const now = Date.now();
        const hit = state.cache.get(path);
        if (hit && ((now - hit.at) < life)) return hit.data;
        try {
            let res = await fetch(path);
            // The server rate-limits wallet-scoped reads (the `wallet-default`
            // bucket refills one token per second). A multi-screen sweep can trip
            // it, and reporting "no data" when the truth is "not read yet" would
            // be a false statement — so retry exactly once, then flag the panel.
            if (res && res.status === 429) {
                state.rateLimited = true;
                await new Promise(function (r) { setTimeout(r, 450); });
                res = await fetch(path);
            }
            const data = await res.json();
            state.cache.set(path, { at: now, data: data });
            return data;
        } catch (err) {
            return null;
        }
    }

    // Wallet-scoped endpoint. Returns null (not a broken request) when no wallet
    // is connected, so renderers can show a connect prompt instead of an error.
    async function getWalletJSON(path, ttl) {
        const w = activeWallet();
        if (!w) return null;
        const sep = (path.indexOf('?') === -1) ? '?' : '&';
        return getJSON(path + sep + 'wallet=' + encodeURIComponent(w), ttl);
    }

    function gameState() {
        if (typeof window.GetGameState !== 'function') return {};
        try { return window.GetGameState() || {}; } catch (err) { return {}; }
    }

    // ========================================================================
    // ASSOCIATION HELPERS — read the wallet's associations defensively
    // ------------------------------------------------------------------------
    // Every helper resolves to "nothing" rather than inventing a fact. The
    // endpoints this surface reads are owned by the server: where a payload is
    // absent, null, or shaped differently than expected, the screen must SAY SO
    // instead of presenting a guess as analytics.
    // ========================================================================
    function lcWallet(v) { return String(v == null ? '' : v).trim().toLowerCase(); }

    function sameWallet(a, b) {
        if (!a || !b) return false;
        return lcWallet(a) === lcWallet(b);
    }

    // Tolerant array extraction: an endpoint may return a bare array (e.g.
    // /api/loans), a named envelope key (e.g. {data:[…]}), or a map keyed by id
    // (e.g. {factions:{…}}). All three resolve to a list.
    function pickArray(json, key) {
        if (!json) return [];
        if (Array.isArray(json)) return json;
        const v = key ? json[key] : null;
        if (Array.isArray(v)) return v;
        if (v && typeof v === 'object') {
            return Object.keys(v).map(function (k) { return v[k]; });
        }
        return [];
    }

    // Membership test for the engine's map-of-wallets shapes (Club.Members,
    // Religion.Members, PlayerStats.Relationships, …).
    function hasMember(map, wallet) {
        if (!map || !wallet || typeof map !== 'object') return false;
        const keys = Object.keys(map);
        for (let i = 0; i < keys.length; i++) {
            if (sameWallet(keys[i], wallet)) return true;
        }
        return false;
    }

    function countWhere(arr, fn) {
        let n = 0;
        (arr || []).forEach(function (x) { if (fn(x)) n++; });
        return n;
    }

    function sumMicro(arr, fn) {
        return (arr || []).reduce(function (a, x) { return a + (Number(fn(x)) || 0); }, 0);
    }

    function maxOf(arr, fn) {
        return (arr || []).reduce(function (m, x) { return Math.max(m, Number(fn(x)) || 0); }, 0);
    }

    // EntityStats (§30) is a six-axis uint64 vector, each axis 1..100.
    const ENTITY_STAT_KEYS = ['speed', 'intelligence', 'willpower', 'strength', 'charisma', 'agility'];

    // Integer-only aggregate of an entity stat vector (mirrors the engine's
    // EntityStats.StatSum — no float is produced for a game value).
    function entityStatSum(s) {
        if (!s || typeof s !== 'object') return 0;
        return ENTITY_STAT_KEYS.reduce(function (a, k) { return a + (Number(s[k]) || 0); }, 0);
    }

    function dominantEntityStat(s) {
        if (!s || typeof s !== 'object') return '—';
        let best = 0, name = '—';
        ENTITY_STAT_KEYS.forEach(function (k) {
            const v = Number(s[k]) || 0;
            if (v > best) { best = v; name = humanKey(k); }
        });
        return name;
    }

    // View-only bar geometry from a 0..100 integer axis.
    function statBars(s) {
        if (!s || typeof s !== 'object') return emptyNote('No stat vector reported.');
        return ENTITY_STAT_KEYS.map(function (k) {
            const v = Number(s[k]) || 0;
            return barRow(humanKey(k), v, 100, String(v));
        }).join('');
    }

    // Maturity (§26.4 pets / §25.6 vehicles): the engine-reported flag first,
    // otherwise derived from the engine-reported birth time + maturity window
    // (integer arithmetic only).
    function petMature(p) {
        if (!p) return false;
        if (p.mature === true || p.Mature === true) return true;
        return maturityRemainingMs(p) <= 0;
    }

    function maturityRemainingMs(p) {
        const born = p ? (p.birth_at || p.BirthAt) : null;
        const win = p ? (p.maturity_ms == null ? p.MaturityMs : p.maturity_ms) : null;
        if (!born || !win) return 0;
        const t = new Date(born).getTime();
        if (isNaN(t)) return 0;
        return (t + (Number(win) || 0)) - Date.now();
    }

    function fmtDuration(ms) {
        const n = Number(ms) || 0;
        if (n <= 0) return 'mature';
        const d = Math.floor(n / 86400000);
        const h = Math.floor((n % 86400000) / 3600000);
        if (d > 0) return d + 'd ' + h + 'h';
        const m = Math.floor((n % 3600000) / 60000);
        return h + 'h ' + m + 'm';
    }

    // ========================================================================
    // ENGINE ASSOCIATION RECORD — /api/player/associations
    // ------------------------------------------------------------------------
    // The engine's own PlayerStats associations (alliance graph, warrant
    // network, buff flags, sector tiles, character relationships, sustained
    // liquidity, custody maps). Returns null when no wallet is connected, and
    // an object with present:false when the engine holds no record yet — the
    // renderers state that plainly instead of implying the wallet has none.
    // ========================================================================
    async function engineAssoc(ttl) {
        const json = await getWalletJSON('/api/player/associations', ttl || 30000);
        if (!json || typeof json !== 'object') return null;
        return json;
    }

    // Explicit "the engine holds no record yet" note, as opposed to "none".
    function noEngineRecord() {
        return '<p class="pp-hint">The engine holds no PlayerStats record for this wallet yet, so its associations are not reported. This is not the same as having none — associate the wallet in-game and reopen this screen.</p>';
    }

    function invalidate() { state.cache.clear(); }

    // ========================================================================
    // FORMAT PRIMITIVES
    // ========================================================================
    function esc(s) {
        return String(s == null ? '' : s).replace(/[&<>"]/g, function (c) {
            return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c];
        });
    }

    // Presentation-boundary conversion of uint64 micro-units -> display string.
    function fmtVBV(micro) {
        const n = Number(micro) || 0;
        return (n / MICRO).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 4 });
    }

    // Presentation-only ratio (integer counts in, display string out).
    function pct(part, total) {
        const p = Number(part) || 0;
        const t = Number(total) || 0;
        if (t <= 0) return '0.0%';
        return ((p / t) * 100).toFixed(1) + '%';
    }

    // View-only bar geometry derived from integer counts.
    function barWidth(value, max) {
        const v = Number(value) || 0;
        const m = Number(max) || 0;
        if (m <= 0) return '0%';
        let w = (v / m) * 100;
        if (w < 0) w = 0;
        if (w > 100) w = 100;
        return w.toFixed(1) + '%';
    }

    function countOf(x) {
        if (Array.isArray(x)) return x.length;
        if (x && typeof x === 'object') return Object.keys(x).length;
        return 0;
    }

    function humanKey(k) {
        return String(k == null ? '' : k)
            .replace(/[_-]+/g, ' ')
            .replace(/\b\w/g, function (c) { return c.toUpperCase(); });
    }

    function fmtAny(v) {
        if (v == null) return '—';
        if (typeof v === 'boolean') return v ? 'Yes' : 'No';
        if (Array.isArray(v)) return v.length ? (v.length + ' recorded') : '—';
        if (typeof v === 'object') return countOf(v) + ' field(s)';
        return String(v);
    }

    function fmtWhen(ts) {
        if (!ts) return '—';
        const d = new Date(ts);
        if (isNaN(d.getTime())) return '—';
        return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    }

    function shortWallet(w) {
        if (!w) return '—';
        const s = String(w);
        return s.length > 12 ? (s.slice(0, 6) + '…' + s.slice(-4)) : s;
    }

    // ========================================================================
    // RENDER PRIMITIVES — analytical building blocks
    // ========================================================================
    function kpi(label, value, sub) {
        return '<div class="pp-stat"><span class="pp-stat-label">' + esc(label) + '</span>'
            + '<span class="pp-stat-val">' + esc(value) + '</span>'
            + (sub ? '<span class="pf-kpi-sub">' + esc(sub) + '</span>' : '')
            + '</div>';
    }

    function kpiGrid(items) {
        return '<div class="pp-grid">' + items.join('') + '</div>';
    }

    function panelHead(title, hint) {
        return '<div class="pf-head"><span class="pf-head-title">' + esc(title) + '</span>'
            + (hint ? '<span class="pf-head-hint">' + esc(hint) + '</span>' : '')
            + '</div>';
    }

    function emptyNote(msg) {
        return '<div class="pp-empty">' + esc(msg) + '</div>';
    }

    function loadingNote(msg) {
        return '<div class="pp-loading">' + esc(msg || 'Loading…') + '</div>';
    }

    function connectNote() {
        return emptyNote('Connect a wallet to view your analytics.');
    }

    // Generic, DEFENSIVE key/value table. Used for endpoint payloads whose full
    // shape is owned by the backend: we render exactly the fields the server
    // reports and never invent a key that was not returned.
    function rowsTable(obj) {
        if (!obj || typeof obj !== 'object') return emptyNote('No data reported.');
        const keys = Object.keys(obj);
        if (!keys.length) return emptyNote('No data reported.');
        return '<div class="pf-rows">' + keys.map(function (k) {
            return '<div class="pf-row"><span class="pf-row-k">' + esc(humanKey(k)) + '</span>'
                + '<span class="pf-row-v">' + esc(fmtAny(obj[k])) + '</span></div>';
        }).join('') + '</div>';
    }

    function barRow(label, value, max, display) {
        return '<div class="pf-bar-row">'
            + '<span class="pf-bar-label">' + esc(label) + '</span>'
            + '<span class="pf-bar-track"><span class="pf-bar-fill" style="width:' + barWidth(value, max) + '"></span></span>'
            + '<span class="pf-bar-val">' + esc(display == null ? value : display) + '</span>'
            + '</div>';
    }

    // Analytical table: header row + body rows.
    function table(headers, rows) {
        if (!rows || !rows.length) return emptyNote('Nothing recorded yet.');
        return '<div class="pf-table"><div class="pf-tr pf-th">'
            + headers.map(function (h) { return '<span class="pf-td">' + esc(h) + '</span>'; }).join('')
            + '</div>'
            + rows.map(function (r) {
                return '<div class="pf-tr">'
                    + r.map(function (c, i) {
                        return '<span class="pf-td' + (i === 0 ? ' pf-td-strong' : '') + '">' + esc(c) + '</span>';
                    }).join('')
                    + '</div>';
            }).join('')
            + '</div>';
    }

    function chipRow(chips) {
        if (!chips || !chips.length) return emptyNote('Nothing recorded yet.');
        return '<div class="pf-chips">' + chips.map(function (c) {
            return '<span class="pf-chip">' + esc(c) + '</span>';
        }).join('') + '</div>';
    }

    // ---- card analytics helpers -------------------------------------------
    // A card's PROGRESSION score is an integer combination of the three
    // progression axes the engine tracks per card: level, loyalty, artifact
    // power. Integer-only (no float) so ordering is deterministic.
    function cardProgressionScore(card) {
        if (!card) return 0;
        const lvl = Number(card.level) || 0;
        const loy = Number(card.loyalty) || 0;
        const art = Number(card.artifact) || 0;
        return (lvl * 100) + loy + art;
    }

    function cardPowerSum(card) {
        if (!card || !Array.isArray(card.power)) return 0;
        return card.power.reduce(function (a, b) { return a + (Number(b) || 0); }, 0);
    }

    function cardsFromState(gs) {
        if (gs && Array.isArray(gs.deck)) return gs.deck;
        return [];
    }

    function sortedByProgression(cards) {
        return cards.slice().sort(function (a, b) {
            return cardProgressionScore(b) - cardProgressionScore(a);
        });
    }

    function tally(cards, keyFn) {
        const out = new Map();
        cards.forEach(function (c) {
            const k = keyFn(c);
            if (k == null || k === '') return;
            out.set(k, (out.get(k) || 0) + 1);
        });
        return Array.from(out.entries()).sort(function (a, b) { return b[1] - a[1]; });
    }

    // ========================================================================
    // ANALYTICS TAXONOMY
    // ------------------------------------------------------------------------
    // TIER 1 categories. Each owns ONE analytical aspect of the player's
    // standing and exposes its analytics as TIER 2 screens. Categories group by
    // ASPECT (record / attributes / holdings / lineage), never by click-target.
    // ========================================================================
    const TABS = [
        {
            id: 'overview', label: 'Overview', icon: '📊',
            blurb: 'Headline standing across every aspect',
            subs: [
                { id: 'summary', label: 'Summary' },
                { id: 'standing', label: 'Standing' },
                { id: 'notables', label: 'Notables' },
            ], render: renderOverview,
        },
        {
            id: 'record', label: 'Record', icon: '🏆',
            blurb: 'Win / loss analytics and match history',
            subs: [
                { id: 'wl', label: 'Win / Loss' },
                { id: 'history', label: 'Match History' },
                { id: 'bounties', label: 'Bounty Record' },
            ], render: renderRecord,
        },
        {
            id: 'stats', label: 'Stats', icon: '📈',
            blurb: 'Attributes, social standing and risk profile',
            subs: [
                { id: 'core', label: 'Core' },
                { id: 'attributes', label: 'Attributes' },
                { id: 'risk', label: 'Risk Profile' },
            ], render: renderStats,
        },
        {
            id: 'identity', label: 'Identity', icon: '🪪',
            blurb: 'Faceplate, reputation trail and playstyle',
            subs: [
                { id: 'profile', label: 'Profile' },
                { id: 'playstyle', label: 'Playstyle' },
                { id: 'trail', label: 'Reputation Trail' },
                { id: 'links', label: 'Linked Identity' },
            ], render: renderIdentity,
        },
        {
            id: 'deck', label: 'Deck', icon: '🃏',
            blurb: 'Current deck composition and slots',
            subs: [
                { id: 'current', label: 'Current Deck' },
                { id: 'composition', label: 'Composition' },
                { id: 'slots', label: 'Deck Slots' },
            ], render: renderDeck,
        },
        {
            id: 'progression', label: 'Progression', icon: '🔺',
            blurb: 'Card progression — highest five and full board',
            subs: [
                { id: 'top5', label: 'Highest 5' },
                { id: 'all', label: 'All Cards' },
                { id: 'power', label: 'Power Breakdown' },
            ], render: renderProgression,
        },
        {
            id: 'titles', label: 'Titles', icon: '🏅',
            blurb: 'Titles held and the full catalogue',
            subs: [
                { id: 'current', label: 'Current Titles' },
                { id: 'catalogue', label: 'Catalogue' },
            ], render: renderTitles,
        },
        {
            id: 'moods', label: 'Moods', icon: '🎭',
            blurb: 'Relative character and board moods',
            subs: [
                { id: 'characters', label: 'Character Moods' },
                { id: 'board', label: 'Board Moods' },
                { id: 'spread', label: 'Mood Spread' },
            ], render: renderMoods,
        },
        {
            id: 'economy', label: 'Economy', icon: '💰',
            blurb: 'Balances, holdings and philanthropy',
            subs: [
                { id: 'balances', label: 'Balances' },
                { id: 'holdings', label: 'Holdings' },
                { id: 'philanthropy', label: 'Philanthropy' },
                { id: 'markets', label: 'Market & Weather' },
            ], render: renderEconomy,
        },
        {
            id: 'careers', label: 'Careers', icon: '⚔️',
            blurb: 'Career tier, XP and constellation unlocks',
            subs: [
                { id: 'tier', label: 'Career Tier' },
                { id: 'xp', label: 'Career XP' },
                { id: 'unlocks', label: 'Unlocks' },
            ], render: renderCareers,
        },
        {
            id: 'entities', label: 'Entities', icon: '🤖',
            blurb: 'AI citizens, life assets and combined power',
            subs: [
                { id: 'citizens', label: 'AI Citizens' },
                { id: 'assets', label: 'Life Assets' },
                { id: 'power', label: 'Combined Power' },
                { id: 'children', label: 'Children Bots' },
                { id: 'garden', label: 'Zen Garden' },
            ], render: renderEntities,
        },
        {
            id: 'achievements', label: 'Achievements', icon: '🎖️',
            blurb: 'Unlocked achievements and completion analytics',
            subs: [
                { id: 'unlocked', label: 'Unlocked' },
                { id: 'progress', label: 'Completion' },
            ], render: renderAchievements,
        },
        {
            id: 'companions', label: 'Companions', icon: '🐾',
            blurb: 'Pets, vehicles and their lineage',
            subs: [
                { id: 'pets', label: 'Pets' },
                { id: 'vehicles', label: 'Vehicles' },
                { id: 'lineage', label: 'Lineage & Maturity' },
            ], render: renderCompanions,
        },
        {
            id: 'rivalry', label: 'Rivalry', icon: '⚔️',
            blurb: 'Rivals, invitations and faction standing',
            subs: [
                { id: 'rivals', label: 'Active Rivals' },
                { id: 'invitations', label: 'Invitations' },
                { id: 'factions', label: 'Factions' },
            ], render: renderRivalry,
        },
        {
            id: 'clubs', label: 'Clubs & Alliances', icon: '🏛️',
            blurb: 'Organisations, memberships, alliances and treasuries',
            subs: [
                { id: 'owned', label: 'Clubs' },
                { id: 'alliance', label: 'Alliance Links' },
                { id: 'territory', label: 'Territory Tiles' },
                { id: 'treasury', label: 'Treasury & Commission' },
            ], render: renderClubsAlliances,
        },
        {
            id: 'bonded', label: 'Bonded Assets', icon: '📦',
            blurb: 'Bonded registry, world content and share holdings',
            subs: [
                { id: 'registry', label: 'Bonded Registry' },
                { id: 'branding', label: 'Branding Coverage' },
                { id: 'market', label: 'Asset Market' },
                { id: 'world', label: 'World Content' },
                { id: 'holdings', label: 'Share Holdings' },
            ], render: renderBondedAssets,
        },
        {
            id: 'faith', label: 'Faith', icon: '⛪',
            blurb: 'Churches owned, religion standing and region coherence',
            subs: [
                { id: 'churches', label: 'Churches' },
                { id: 'religion', label: 'Religion' },
                { id: 'coherence', label: 'Region Coherence' },
            ], render: renderFaith,
        },
        {
            id: 'custody', label: 'Legal & Custody', icon: '⚖️',
            blurb: 'Warrants, detained cards and open bounties',
            subs: [
                { id: 'warrants', label: 'Warrants' },
                { id: 'custody', label: 'Custody' },
                { id: 'bounties', label: 'Bounties' },
            ], render: renderCustody,
        },
        {
            id: 'obligations', label: 'Obligations', icon: '🧾',
            blurb: 'Loans, leases and governance weight',
            subs: [
                { id: 'loans', label: 'Loans' },
                { id: 'leases', label: 'Leases' },
                { id: 'governance', label: 'Governance' },
            ], render: renderObligations,
        },
    ];

    function tabById(id) {
        for (let i = 0; i < TABS.length; i++) {
            if (TABS[i].id === id) return TABS[i];
        }
        return TABS[0];
    }

    function currentSubId() {
        const active = subtabsEl ? subtabsEl.querySelector('.pp-subtab.active') : null;
        return active ? active.dataset.sub : tabById(state.activeTab).subs[0].id;
    }

    // ========================================================================
    // HUB CHROME — contained shell (built lazily on first open)
    // ========================================================================
    function ensureHub() {
        if (hubEl) return;
        hubEl = document.createElement('div');
        hubEl.id = HUB_ID;
        // `.pp-hub` is the containment contract (ui.js hideAllOverlays closes it
        // by class); `.overlay` is the shared base class.
        hubEl.className = 'pp-hub overlay';
        hubEl.style.display = 'none';
        hubEl.innerHTML = ''
            + '<div class="pp-shell">'
            + '  <aside class="pp-sidebar">'
            + '    <div class="pp-sidebar-head">'
            + '      <div class="pp-avatar" id="pp-avatar">📊</div>'
            + '      <div class="pp-id">'
            + '        <div class="pp-id-name" id="pp-id-name">Portfolio</div>'
            + '        <div class="pp-id-wallet" id="pp-id-wallet">—</div>'
            + '      </div>'
            + '      <button class="pp-close" onclick="window.closePortfolio()" aria-label="Close portfolio">✕</button>'
            + '    </div>'
            + '    <nav class="pp-cats" id="pp-cats"></nav>'
            + '  </aside>'
            + '  <main class="pp-content">'
            + '    <header class="pp-topbar">'
            + '      <div class="pf-brand"><span class="pf-brand-title">PORTFOLIO</span>'
            + '        <span class="pf-brand-sub">analytics</span></div>'
            + '      <div class="pp-region-switch">'
            + '        <span class="pp-region-label">REGION:</span>'
            + '        <select id="pp-region-select" class="pp-region-select" onchange="window.ppSwitchRegion(this.value)">'
            + '          <option value="">— all regions —</option>'
            + '        </select>'
            + '      </div>'
            + '      <div class="pp-cat-title" id="pp-cat-title"></div>'
            + '    </header>'
            // READ-ONLY MANDATE: this notice lives in the HUB CHROME (not inside a
            // panel) so it is visible on EVERY analytics screen, even empty ones.
            + '    <div class="pp-readonly-note pp-rm-hint">🔒 Read-only analytics — all actions live in the World Dashboard.</div>'
            + '    <div class="pp-subtabs" id="pp-subtabs"></div>'
            + '    <section class="pp-panel" id="pp-panel"></section>'
            + '  </main>'
            + '</div>';
        document.body.appendChild(hubEl);

        sidebarEl = hubEl.querySelector('#pp-cats');
        subtabsEl = hubEl.querySelector('#pp-subtabs');
        panelEl = hubEl.querySelector('#pp-panel');
        titleEl = hubEl.querySelector('#pp-cat-title');

        renderSidebar();
        loadRegions();
    }

    function renderSidebar() {
        if (!sidebarEl) return;
        sidebarEl.innerHTML = TABS.map(function (t) {
            const active = (t.id === state.activeTab) ? ' active' : '';
            return '<button class="pp-cat' + active + '" data-cat="' + t.id + '"'
                + ' title="' + esc(t.blurb) + '"'
                + ' onclick="window.ppOpenCategory(\'' + t.id + '\')">'
                + '<span class="pp-cat-icon">' + t.icon + '</span>'
                + '<span class="pp-cat-label">' + esc(t.label) + '</span>'
                + '</button>';
        }).join('');
    }

    function renderSubtabs(tab) {
        if (!subtabsEl) return;
        subtabsEl.innerHTML = tab.subs.map(function (sp) {
            return '<button class="pp-subtab" data-sub="' + sp.id + '"'
                + ' onclick="window.ppOpenSub(\'' + tab.id + '\',\'' + sp.id + '\')">'
                + esc(sp.label) + '</button>';
        }).join('');
    }

    function refreshIdentityChrome() {
        const nameEl = hubEl.querySelector('#pp-id-name');
        const walEl = hubEl.querySelector('#pp-id-wallet');
        const gs = gameState();
        const w = activeWallet();
        if (nameEl) nameEl.textContent = gs.job_role || 'Unassigned';
        if (walEl) walEl.textContent = w ? shortWallet(w) : 'wallet not connected';
    }

    function openPortfolio(tabId, subId) {
        ensureHub();
        if (tabId) state.activeTab = tabId;
        refineActiveTab();
        invalidate();                                  // always show fresh analytics on open
        refreshIdentityChrome();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        // hideAllOverlays() adds `.hidden` to EVERY `.overlay`, and `.hidden` carries
        // `display:none !important` (plus visibility:hidden / opacity:0). Because this
        // hub is `class="pp-hub overlay"`, that class would beat the inline display
        // below and the Portfolio would never actually become visible — so clear it.
        hubEl.classList.remove('hidden');
        hubEl.style.display = 'flex';
        renderSidebar();
        openCategory(state.activeTab, subId);
    }

    // A requested tab may no longer exist (retired id / stale star). Fall back to
    // the Overview so the Portfolio can never open on a dead screen.
    function refineActiveTab() {
        const known = TABS.some(function (t) { return t.id === state.activeTab; });
        if (!known) state.activeTab = TABS[0].id;
    }

    function closePortfolio() {
        if (hubEl) hubEl.style.display = 'none';
    }

    function openCategory(tabId, subId) {
        const tab = tabById(tabId);
        state.activeTab = tab.id;
        if (sidebarEl) {
            const btns = sidebarEl.querySelectorAll('.pp-cat');
            for (let i = 0; i < btns.length; i++) {
                btns[i].classList.toggle('active', btns[i].dataset.cat === tab.id);
            }
        }
        if (titleEl) titleEl.textContent = tab.label + ' — ' + tab.blurb;
        renderSubtabs(tab);
        openSub(tab.id, subId || tab.subs[0].id);
    }

    function openSub(tabId, subId) {
        const tab = tabById(tabId);
        state.activeTab = tab.id;
        state.activeSub = subId || tab.subs[0].id;
        if (subtabsEl) {
            const btns = subtabsEl.querySelectorAll('.pp-subtab');
            for (let i = 0; i < btns.length; i++) {
                btns[i].classList.toggle('active', btns[i].dataset.sub === state.activeSub);
            }
        }
        if (!panelEl) return;
        const token = ++renderToken;
        panelEl.innerHTML = loadingNote('Reading analytics…');
        try {
            const out = tab.render(panelEl, state.activeSub, token);
            // Renderers are async (they read live endpoints). Surface a rejected
            // promise as a panel error instead of an unhandled rejection.
            if (out && typeof out.catch === 'function') {
                out.catch(function (err) {
                    console.error('[Portfolio] render error:', err);
                    setPanel(token, emptyNote('Failed to render ' + tab.label + '.'));
                });
            }
        } catch (err) {
            console.error('[Portfolio] render error:', err);
            panelEl.innerHTML = emptyNote('Failed to render ' + tab.label + '.');
        }
    }

    // Guards against a slow fetch for screen A overwriting screen B, which the
    // user has since opened.
    function setPanel(token, html) {
        if (token !== renderToken) return;
        let out = html;
        if (state.rateLimited) {
            // One or more reads were refused even after a retry. Say so, rather
            // than letting a screen imply the wallet has no associated data.
            out += '<p class="pp-hint">Some figures could not be read — the server rate-limited this view. Reopen the screen in a moment to refresh.</p>';
            state.rateLimited = false;
        }
        if (panelEl) panelEl.innerHTML = out;
    }

    function switchRegion(regionId) {
        state.activeRegion = regionId || null;
        openSub(state.activeTab, state.activeSub);
    }

    async function loadRegions() {
        if (state.regionsLoaded) return;
        const json = await getJSON('/api/regions', 60000);
        const list = (json && json.data) ? json.data : (Array.isArray(json) ? json : []);
        state.regions = list;
        state.regionsLoaded = true;
        const sel = hubEl ? hubEl.querySelector('#pp-region-select') : null;
        if (sel && list.length) {
            sel.innerHTML = '<option value="">— all regions —</option>' + list.map(function (r) {
                const val = r.id || r.region_id || r.name || '';
                return '<option value="' + esc(val) + '">' + esc(r.name || r.id || r.region_id || val) + '</option>';
            }).join('');
        }
    }

    // ========================================================================
    // SHARED ANALYTICS
    // ========================================================================
    // Outcome of one match-history record from the LOCAL player's perspective.
    // Derived from `scores` when present (authoritative [p1,p2]) and falling back
    // to `winner_index`. Integer comparisons only.
    function outcomeOf(m, myIdx) {
        const sc = Array.isArray(m.scores) ? m.scores : null;
        if (sc && sc.length === 2) {
            const a = Number(sc[0]) || 0;
            const b = Number(sc[1]) || 0;
            if (a === b) return 'draw';
            const w = (a > b) ? 0 : 1;
            return (w === myIdx) ? 'win' : 'loss';
        }
        const wi = Number(m.winner_index);
        if (wi === 0 || wi === 1) return (wi === myIdx) ? 'win' : 'loss';
        return 'draw';
    }

    function computeRecord(gs) {
        const history = Array.isArray(gs.match_history) ? gs.match_history : [];
        const myIdx = (typeof gs.local_player_index === 'number') ? gs.local_player_index : 0;
        const rec = { total: history.length, wins: 0, losses: 0, draws: 0, bountyWins: 0, bountyMicro: 0 };
        history.forEach(function (m) {
            const o = outcomeOf(m, myIdx);
            if (o === 'win') rec.wins++;
            else if (o === 'loss') rec.losses++;
            else rec.draws++;
            if (m.is_bounty_match && o === 'win') {
                rec.bountyWins++;
                rec.bountyMicro += (Number(m.bounty_reward_micro) || 0);
            }
        });
        return rec;
    }

    async function leaderboardStanding() {
        const w = activeWallet();
        const list = await getJSON('/api/leaderboard');
        const arr = Array.isArray(list) ? list : [];
        if (!w) return { list: arr, entry: null, rank: 0, size: arr.length };
        const lower = String(w).toLowerCase();
        for (let i = 0; i < arr.length; i++) {
            if (String(arr[i].wallet || '').toLowerCase() === lower) {
                return { list: arr, entry: arr[i], rank: i + 1, size: arr.length };
            }
        }
        return { list: arr, entry: null, rank: 0, size: arr.length };
    }

    async function achievementsPayload() {
        const w = activeWallet();
        const path = '/api/achievements' + (w ? ('?wallet=' + encodeURIComponent(w)) : '');
        return await getJSON(path);
    }

    // ========================================================================
    // OVERVIEW — headline standing across every aspect
    // ========================================================================
    async function renderOverview(panel, sub, token) {
        const gs = gameState();
        const cards = cardsFromState(gs);
        const rec = computeRecord(gs);
        const ach = await achievementsPayload();
        const unlockedCount = countOf(ach && ach.unlocked);
        const defsCount = countOf(ach && ach.definitions);
        const decided = rec.wins + rec.losses;

        if (sub === 'standing') {
            const lb = await leaderboardStanding();
            const e = lb.entry;
            let html = panelHead('Standing', 'leaderboard position is ranked server-side by wins');
            html += kpiGrid([
                kpi('Rank', lb.rank ? ('#' + lb.rank) : '—', lb.size ? ('of ' + lb.size + ' ranked') : 'no ranked players'),
                kpi('Recorded Wins', e ? e.wins : '—', 'server leaderboard'),
                kpi('DNFs', e ? e.dnfs : '—', 'did-not-finish'),
                kpi('Best Rating', e ? (e.best_rating || '—') : '—', 'career peak'),
                kpi('Reputation', e ? e.reputation : '—', 'server-side'),
                kpi('Disconnect Streak', e ? e.disconnect_streak : '—', 'reliability'),
            ]);
            if (!e) html += emptyNote('This wallet has no server leaderboard record yet.');
            if (lb.size) {
                html += panelHead('Top of the ladder', 'first five ranked players');
                html += table(['Wallet', 'Wins', 'Rating', 'Reputation'], lb.list.slice(0, 5).map(function (r) {
                    return [shortWallet(r.wallet), String(r.wins), r.best_rating || '—', String(r.reputation)];
                }));
            }
            return setPanel(token, html);
        }

        if (sub === 'notables') {
            const mutations = countOf(gs.mutation_history);
            const scars = cards.reduce(function (a, c) { return a + countOf(c.scars); }, 0);
            const fallen = cards.filter(function (c) { return c.fallen === true; }).length;
            return setPanel(token, panelHead('Notables', 'countable state that defines this portfolio')
                + kpiGrid([
                    kpi('Achievements', unlockedCount + '/' + defsCount, 'unlocked of catalogue'),
                    kpi('Mutations', mutations, 'procedure history'),
                    kpi('Card Scars', scars, 'recorded on owned cards'),
                    kpi('Fallen Cards', fallen, 'underworld status'),
                    kpi('Active Buffs', countOf(gs.profile_buffs), 'profile buffs'),
                    kpi('Rumours', (gs.rumor_count == null ? '—' : gs.rumor_count), 'spread by this player'),
                    kpi('Auctions Won', (gs.auctions_won == null ? '—' : gs.auctions_won), 'lifetime'),
                    kpi('Hostage Cards', countOf(gs.held_hostage_cards), 'lost to kidnapping'),
                ]));
        }

        // ---- summary (default) ----
        let html = panelHead('Portfolio summary', 'every headline metric, read-only');
        html += kpiGrid([
            kpi('Record', rec.wins + 'W / ' + rec.losses + 'L' + (rec.draws ? (' / ' + rec.draws + 'D') : ''), rec.total + ' matches on record'),
            kpi('Win Rate', pct(rec.wins, decided), decided ? (decided + ' decided') : 'no decided matches'),
            kpi('Reputation', (gs.reputation == null ? '—' : gs.reputation), 'profile'),
            kpi('Mojo', (gs.mojo == null ? '—' : gs.mojo), gs.social_rank || 'social standing'),
            kpi('$VBV', fmtVBV(gs.virtual_balance == null ? 0 : (Number(gs.virtual_balance) * MICRO)), 'in-game balance'),
            kpi('Deck Rating', (gs.deck_rating == null ? '—' : gs.deck_rating), 'active deck'),
            kpi('Cards in Deck', String(cards.length), 'active deck size'),
            kpi('Achievements', unlockedCount + '/' + defsCount, 'unlocked'),
        ]);
        html += panelHead('Aspect coverage', 'every analytical aspect exposed by this portfolio');
        html += chipRow(['Record', 'Stats', 'Identity', 'Deck', 'Progression', 'Titles',
            'Moods', 'Economy', 'Careers', 'Entities', 'Achievements']);
        html += '<p class="pp-hint">Use the categories on the left; each opens its analytical screens.</p>';
        return setPanel(token, html);
    }

    // ========================================================================
    // RECORD — total win / loss and match history
    // ========================================================================
    async function renderRecord(panel, sub, token) {
        const gs = gameState();
        const rec = computeRecord(gs);
        const history = Array.isArray(gs.match_history) ? gs.match_history : [];
        const myIdx = (typeof gs.local_player_index === 'number') ? gs.local_player_index : 0;
        const decided = rec.wins + rec.losses;

        if (sub === 'history') {
            const rows = history.slice(-15).reverse().map(function (m) {
                const o = outcomeOf(m, myIdx);
                const sc = Array.isArray(m.scores) ? m.scores : [];
                return [
                    o.toUpperCase(),
                    shortWallet(m.opponent_wallet || '—'),
                    (sc.length === 2 ? (sc[0] + '–' + sc[1]) : '—'),
                    (m.is_bounty_match ? 'bounty' : '—'),
                    fmtWhen(m.timestamp),
                ];
            });
            return setPanel(token, panelHead('Match history', 'most recent 15 recorded matches')
                + (history.length ? table(['Outcome', 'Opponent', 'Score', 'Type', 'When'], rows)
                    : emptyNote('No matches recorded for this player yet.')));
        }

        if (sub === 'bounties') {
            const bountyRows = history.filter(function (m) { return m.is_bounty_match; }).map(function (m) {
                const o = outcomeOf(m, myIdx);
                return [o.toUpperCase(), shortWallet(m.opponent_wallet || '—'),
                    fmtVBV(m.bounty_reward_micro || 0), fmtWhen(m.timestamp)];
            });
            return setPanel(token, panelHead('Bounty record', 'bounty matches and rewards collected')
                + kpiGrid([
                    kpi('Bounty Matches', bountyRows.length, 'of ' + rec.total + ' total'),
                    kpi('Bounty Wins', rec.bountyWins, 'captures'),
                    kpi('Bounty $VBV', fmtVBV(rec.bountyMicro), 'rewarded micro-units'),
                    kpi('Capture Rate', pct(rec.bountyWins, bountyRows.length), 'bounty matches won'),
                ])
                + (bountyRows.length ? table(['Outcome', 'Opponent', '$VBV', 'When'], bountyRows)
                    : emptyNote('No bounty matches on record.')));
        }

        // ---- win / loss (default) ----
        const lb = await leaderboardStanding();
        const e = lb.entry;
        const maxSide = Math.max(rec.wins, rec.losses, 1);
        let html = panelHead('Win / Loss', 'derived from this player\u2019s own match history');
        html += kpiGrid([
            kpi('Wins', String(rec.wins), 'of ' + rec.total + ' recorded'),
            kpi('Losses', String(rec.losses), 'of ' + rec.total + ' recorded'),
            kpi('Draws', String(rec.draws), 'undecided'),
            kpi('Win Rate', pct(rec.wins, decided), decided ? (decided + ' decided') : 'no decided matches'),
            kpi('Server Wins', e ? e.wins : '—', 'leaderboard cross-check'),
            kpi('DNFs', e ? e.dnfs : '—', 'incomplete matches'),
        ]);
        html += panelHead('Result split', 'relative volume');
        html += barRow('Wins', rec.wins, maxSide, String(rec.wins));
        html += barRow('Losses', rec.losses, maxSide, String(rec.losses));
        html += barRow('Draws', rec.draws, maxSide, String(rec.draws));
        if (!e) html += emptyNote('No server-side leaderboard record for this wallet yet.');
        return setPanel(token, html);
    }

    // ========================================================================
    // STATS — attributes, social standing and risk profile
    // ========================================================================
    async function renderStats(panel, sub, token) {
        const gs = gameState();
        const cards = cardsFromState(gs);

        if (sub === 'attributes') {
            const owned = await getWalletJSON('/api/owner/combined-stats');
            const overlays = await getWalletJSON('/api/stat-overlay/owner');
            let html = panelHead('Attributes', 'entity statistics combined across all owned entities');
            html += kpiGrid([
                kpi('Effective Power', (owned && owned.effective_power != null) ? owned.effective_power : '—', 'capped power level'),
                kpi('Overlays', (overlays && Array.isArray(overlays.overlays)) ? overlays.overlays.length : '—', 'power overlays owned'),
                kpi('Cards', String(cards.length), 'active deck'),
            ]);
            html += panelHead('Combined entity stats', 'server-authored entity statistics');
            html += (owned && owned.combined_stats)
                ? rowsTable(owned.combined_stats)
                : (activeWallet() ? emptyNote('No combined entity statistics reported yet.') : connectNote());
            if (overlays && Array.isArray(overlays.overlays) && overlays.overlays.length) {
                html += panelHead('Power overlays', 'per-entity effective power');
                html += table(['Entity', 'Region', 'Effective', 'Dominant'], overlays.overlays.map(function (o) {
                    const eff = (o.effective_power == null) ? o.EffectivePower : o.effective_power;
                    return [o.entity_id || o.EntityID || '—',
                        o.region || o.Region || '—',
                        (eff == null ? '—' : String(eff)),
                        o.dominant_stat || o.DominantStat || '—'];
                }));
            }

            // Engine playstyle + buff state (PlayerStats). Ratios arrive as integer
            // parts-per-million so the payload carries no float; scale to a 0-100
            // display axis at the presentation boundary only.
            const assoc = await engineAssoc();
            html += panelHead('Engine attributes', 'playstyle and buff state the engine holds on PlayerStats');
            if (!activeWallet()) return setPanel(token, html + connectNote());
            if (!assoc) return setPanel(token, html + emptyNote('The engine association record could not be read.'));
            if (assoc.present === false) return setPanel(token, html + noEngineRecord());
            const ps = assoc.playstyle || {};
            const ppmPct = function (v) {
                if (v == null) return null;
                return Math.round((Number(v) / 10000) * 100) / 100;   // view geometry only
            };
            html += kpiGrid([
                kpi('Aggressiveness', ppmPct(assoc.aggressiveness_ppm) == null ? '—' : ppmPct(assoc.aggressiveness_ppm) + '%', 'engine ratio'),
                kpi('Risk Tolerance', ppmPct(assoc.risk_tolerance_ppm) == null ? '—' : ppmPct(assoc.risk_tolerance_ppm) + '%', 'engine ratio'),
                kpi('Mojo Decay', ppmPct(assoc.mojo_decay_rate_ppm) == null ? '—' : ppmPct(assoc.mojo_decay_rate_ppm) + '%', 'per period'),
                kpi('Buffs Active', String(countOf(assoc.buffs)), 'flag map'),
                kpi('Timed Buffs', String(countOf(assoc.active_buffs)), 'string-valued map'),
                kpi('Item Buffs', String(countOf(assoc.active_item_buffs)), 'item → matches remaining'),
            ]);
            html += panelHead('Active buffs', 'buff → state');
            html += (countOf(assoc.buffs) ? rowsTable(assoc.buffs) : emptyNote('The engine reports no active buff for this wallet.'));
            html += panelHead('Active buff detail', 'buff → value');
            html += (countOf(assoc.active_buffs) ? rowsTable(assoc.active_buffs) : emptyNote('The engine reports no timed buff for this wallet.'));
            html += panelHead('Item buffs', 'item → matches remaining');
            html += (countOf(assoc.active_item_buffs) ? rowsTable(assoc.active_item_buffs) : emptyNote('The engine reports no item buff for this wallet.'));
            html += panelHead('Preferred rules (parts-per-million)', 'rule → preference weight');
            html += (countOf(ps.preferred_rules_ppm) ? rowsTable(ps.preferred_rules_ppm) : emptyNote('The engine reports no rule preference for this wallet.'));
            html += panelHead('Preferred card moods (parts-per-million)', 'mood → preference weight');
            html += (countOf(ps.preferred_card_moods_ppm) ? rowsTable(ps.preferred_card_moods_ppm) : emptyNote('The engine reports no card-mood preference for this wallet.'));
            html += panelHead('Preferred items (parts-per-million)', 'item → preference weight');
            html += (countOf(ps.preferred_items_ppm) ? rowsTable(ps.preferred_items_ppm) : emptyNote('The engine reports no item preference for this wallet.'));
            return setPanel(token, html);
        }

        if (sub === 'risk') {
            return setPanel(token, panelHead('Risk profile', 'exposure, cooldowns and protective state')
                + kpiGrid([
                    kpi('Wanted Level', (gs.wanted_level == null ? '—' : gs.wanted_level), 'heat'),
                    kpi('Jailed Cards', countOf(gs.jailed_cards), 'detained'),
                    kpi('Kidnapped Cards', countOf(gs.kidnapped_cards), 'held by this player'),
                    kpi('Hostage Cards', countOf(gs.held_hostage_cards), 'lost by this player'),
                    kpi('Rumours', (gs.rumor_count == null ? '—' : gs.rumor_count), 'spread'),
                    kpi('Reparations', (gs.reparations_received_count == null ? '—' : gs.reparations_received_count), 'received'),
                ])
                + panelHead('Protective state', 'active safeguards')
                + kpiGrid([
                    kpi('Ghost Protocol', gs.ghost_protocol_expires_at ? fmtWhen(gs.ghost_protocol_expires_at) : 'inactive', 'signal scrambling'),
                    kpi('Mojo Stabilizer', gs.is_mojo_stabilizer_active ? 'Active' : 'Inactive', 'decay protection'),
                    kpi('Mutation Insurance', gs.has_mutation_insurance ? 'Held' : 'Not held', 'procedure cover'),
                    kpi('Mojo Decay Rate', (gs.mojo_decay_rate == null ? '—' : gs.mojo_decay_rate), 'per period'),
                ]));
        }

        // ---- core (default) ----
        return setPanel(token, panelHead('Core stats', 'identity-bearing attributes')
            + kpiGrid([
                kpi('Reputation', (gs.reputation == null ? '—' : gs.reputation), 'standing'),
                kpi('Mojo', (gs.mojo == null ? '—' : gs.mojo), 'social currency'),
                kpi('Social Rank', gs.social_rank || '—', 'club tier'),
                kpi('Job Role', gs.job_role || '—', gs.employer_id ? ('@ ' + shortWallet(gs.employer_id)) : 'unassigned'),
                kpi('Cunning', (gs.cunning == null ? '—' : gs.cunning), 'effective'),
                kpi('Nurturing', (gs.nurturing == null ? '—' : gs.nurturing), 'effective'),
                kpi('Deck Rating', (gs.deck_rating == null ? '—' : gs.deck_rating), 'active deck'),
                kpi('Favourite Card', (gs.favorite_card_id == null ? '—' : ('#' + gs.favorite_card_id)), 'most used'),
            ]));
    }

    // ========================================================================
    // IDENTITY — faceplate, reputation trail and playstyle
    // ========================================================================
    async function renderIdentity(panel, sub, token) {
        const gs = gameState();
        const w = activeWallet();

        if (sub === 'playstyle') {
            return setPanel(token, panelHead('Playstyle', 'tendencies the engine records against this player')
                + (gs.playstyle ? rowsTable(gs.playstyle) : emptyNote('No playstyle tendencies recorded yet.')));
        }

        // ---- linked identity (every associated wallet) ----
        if (sub === 'links') {
            const snap = w ? await getJSON('/api/identity/snapshot?primary_wallet=' + encodeURIComponent(w), 30000) : null;
            const linked = pickArray(snap, 'linked_wallets');
            const events = pickArray(await getWalletJSON('/api/identity/events', 30000), 'events');
            const compliance = pickArray(await getWalletJSON('/api/compliance/wallet', 30000), 'records');
            const envoi = w ? await getJSON('/api/envoi-name?address=' + encodeURIComponent(w), 60000) : null;
            const name = envoi && (envoi.name || envoi.Name);
            let html = panelHead('Linked identity', 'associated wallets, naming and identity records');
            html += kpiGrid([
                kpi('Wallet Name', name ? name : (w ? 'unresolved' : '—'), 'Envoi naming service'),
                kpi('Linked Wallets', w ? String(linked.length) : '—', 'secondary associations'),
                kpi('Identity Events', String(events.length), 'recorded'),
                kpi('Compliance Records', String(compliance.length), 'server-held'),
            ]);
            if (!w) return setPanel(token, html + connectNote());
            html += panelHead('Associated wallets', 'every wallet linked to this primary address');
            html += (linked.length
                ? table(['Address', 'Chain', 'Platform', 'Verified', 'Linked At'], linked.map(function (l) {
                    return [shortWallet(l.address), l.chain || '—', l.platform || '—',
                        l.verified ? 'Yes' : 'No', fmtWhen(l.timestamp)];
                }))
                : emptyNote('No secondary wallets are linked to this address.'));
            html += panelHead('Identity event trail', 'events recorded against this wallet');
            html += (events.length ? objectArrayTable(events, 6, 20) : emptyNote('No identity events are recorded.'));
            html += panelHead('Compliance records', 'enforcement records held by the server');
            html += (compliance.length ? objectArrayTable(compliance, 6, 20) : emptyNote('No compliance records are held for this wallet.'));
            return setPanel(token, html);
        }

        if (sub === 'trail') {
            const ident = await getWalletJSON('/api/identity/profile');
            const html = panelHead('Reputation trail', 'persistent identity records');
            if (!w) return setPanel(token, html + connectNote());
            if (!ident || ident.success === false || !ident.profile) {
                return setPanel(token, html + emptyNote('No persistent identity profile recorded yet.'));
            }
            const p = ident.profile;
            let body = kpiGrid([
                kpi('Identity Score', (p.identity_score == null ? '—' : p.identity_score), 'server-authored'),
                kpi('Tier', p.tier || '—', 'identity tier'),
                kpi('Total Reputation', (p.total_reputation == null ? '—' : p.total_reputation), 'aggregate'),
                kpi('Current Streak', (p.current_streak == null ? '—' : p.current_streak), 'active'),
                kpi('Best Streak', (p.best_streak == null ? '—' : p.best_streak), 'personal best'),
                kpi('Redemptions', (p.redemptions == null ? '—' : p.redemptions), 'lifetime'),
            ]);
            body += panelHead('Category reputation', 'reputation per category');
            body += rowsTable(p.category_reps);
            const achRows = Array.isArray(p.achievements) ? p.achievements.map(function (a) {
                return [a.title || a.category || '—', a.detail || '—', String(a.impact == null ? '—' : a.impact), fmtWhen(a.timestamp)];
            }) : [];
            const failRows = Array.isArray(p.failures) ? p.failures.map(function (a) {
                return [a.title || a.category || '—', a.detail || '—', String(a.impact == null ? '—' : a.impact), fmtWhen(a.timestamp)];
            }) : [];
            body += panelHead('Identity achievements', 'recorded positive events');
            body += (achRows.length ? table(['Title', 'Detail', 'Impact', 'When'], achRows) : emptyNote('None recorded.'));
            body += panelHead('Identity failures', 'recorded negative events');
            body += (failRows.length ? table(['Title', 'Detail', 'Impact', 'When'], failRows) : emptyNote('None recorded.'));
            return setPanel(token, html + body);
        }

        // ---- profile (default) ----
        const envoi = w ? await getJSON('/api/envoi-name?address=' + encodeURIComponent(w), 60000) : null;
        return setPanel(token, panelHead('Identity', 'read-only identity face')
            + kpiGrid([
                kpi('Wallet', w ? shortWallet(w) : '—', 'connected address'),
                kpi('Envoi Name', (envoi && envoi.name) ? envoi.name : '—', 'Voi naming service'),
                kpi('Faceplate', gs.equipped_faceplate || '—', 'equipped'),
                kpi('Avatar', gs.avatar_url ? 'Set' : '—', 'profile image'),
                kpi('Faction', gs.faction || '—', 'hegemony alignment'),
                kpi('Employer', gs.employer_id ? shortWallet(gs.employer_id) : '—', 'current club'),
            ])
            + panelHead('Gloat message', 'shown to opponents')
            + '<div class="pf-quote">' + esc(gs.gloat_message || 'None set.') + '</div>');
    }

    // ========================================================================
    // DECK — current deck composition and slots
    // ========================================================================
    async function renderDeck(panel, sub, token) {
        const gs = gameState();
        const cards = cardsFromState(gs);

        if (!cards.length) {
            return setPanel(token, panelHead('Deck', 'active deck analytics')
                + emptyNote('No active deck reported. The engine exposes the deck once a match deck is loaded.'));
        }

        if (sub === 'composition') {
            const byElement = tally(cards, function (c) { return c.element || 'Neutral'; });
            const byTier = tally(cards, function (c) { return c.tier || 'Unranked'; });
            const byMood = tally(cards, function (c) { return c.mood || 'Unknown'; });
            const maxE = byElement.length ? byElement[0][1] : 1;
            let html = panelHead('Composition', 'how this deck is built');
            html += panelHead('By element', 'count per element');
            html += byElement.map(function (e) { return barRow(e[0], e[1], maxE, String(e[1])); }).join('');
            html += panelHead('By tier', 'count per tier');
            html += byTier.map(function (e) { return barRow(e[0], e[1], maxE, String(e[1])); }).join('');
            html += panelHead('By mood', 'count per mood');
            html += byMood.map(function (e) { return barRow(e[0], e[1], maxE, String(e[1])); }).join('');
            return setPanel(token, html);
        }

        if (sub === 'slots') {
            const fat = cards.reduce(function (a, c) { return a + (Number(c.fatigue) || 0); }, 0);
            const loy = cards.reduce(function (a, c) { return a + (Number(c.loyalty) || 0); }, 0);
            const pw = cards.reduce(function (a, c) { return a + cardPowerSum(c); }, 0);
            return setPanel(token, panelHead('Deck slots', 'active slot analytics')
                + kpiGrid([
                    kpi('Active Slot', (gs.active_deck == null ? '—' : ('#' + gs.active_deck)), 'engine-declared'),
                    kpi('Deck Rating', (gs.deck_rating == null ? '—' : gs.deck_rating), 'computed'),
                    kpi('Cards', String(cards.length), 'in the active deck'),
                    kpi('Total Power', String(pw), 'sum of all sides'),
                    kpi('Total Loyalty', String(loy), 'accumulated'),
                    kpi('Total Fatigue', String(fat), 'accumulated penalty'),
                ])
                + '<p class="pp-hint">Deck editing is owned by the World Dashboard (Player Hub ▸ Deck Manager).</p>');
        }

        // ---- current (default) ----
        const rows = cards.map(function (c) {
            return [c.name || ('#' + (c.id == null ? '?' : c.id)),
                c.element || '—', c.mood || '—', c.tier || '—',
                (c.level == null ? '—' : String(c.level)),
                (c.loyalty == null ? '—' : String(c.loyalty)),
                String(cardPowerSum(c))];
        });
        const total = cards.reduce(function (a, c) { return a + cardPowerSum(c); }, 0);
        return setPanel(token, panelHead('Current deck', 'every card in the active deck')
            + kpiGrid([
                kpi('Cards', String(cards.length), 'deck size'),
                kpi('Total Power', String(total), 'sum of power sides'),
                kpi('Deck Rating', (gs.deck_rating == null ? '—' : gs.deck_rating), 'engine-computed'),
            ])
            + table(['Card', 'Element', 'Mood', 'Tier', 'Lvl', 'Loyalty', 'Power'], rows));
    }

    // ========================================================================
    // PROGRESSION — card progression analytics
    // ========================================================================
    async function renderProgression(panel, sub, token) {
        const gs = gameState();
        const cards = cardsFromState(gs);
        if (!cards.length) {
            return setPanel(token, panelHead('Progression', 'card progression analytics')
                + emptyNote('No active deck reported, so no card progression can be measured.'));
        }
        const ranked = sortedByProgression(cards);

        if (sub === 'power') {
            const sides = ['Top', 'Right', 'Bottom', 'Left'];
            const totals = [0, 0, 0, 0];
            cards.forEach(function (c) {
                if (!Array.isArray(c.power)) return;
                for (let i = 0; i < 4; i++) totals[i] += (Number(c.power[i]) || 0);
            });
            const maxSide = Math.max(totals[0], totals[1], totals[2], totals[3], 1);
            let html = panelHead('Power breakdown', 'power summed across every card, per facing');
            html += kpiGrid([
                kpi('Total Power', String(totals.reduce(function (a, b) { return a + b; }, 0)), 'all facings'),
                kpi('Strongest Facing', sides[totals.indexOf(maxSide)], String(maxSide)),
                kpi('Cards', String(cards.length), 'measured'),
            ]);
            html += sides.map(function (s, i) { return barRow(s, totals[i], maxSide, String(totals[i])); }).join('');
            html += '<p class="pp-hint">Facings are the card\u2019s four power sides: top, right, bottom, left.</p>';
            return setPanel(token, html);
        }

        if (sub === 'all') {
            const rows = ranked.map(function (c, i) {
                return ['#' + (i + 1) + ' ' + (c.name || ('#' + c.id)),
                    String(cardProgressionScore(c)),
                    (c.level == null ? '—' : String(c.level)),
                    (c.loyalty == null ? '—' : String(c.loyalty)),
                    (c.artifact == null ? '—' : String(c.artifact)),
                    String(cardPowerSum(c))];
            });
            return setPanel(token, panelHead('All cards', 'every card ranked by progression score')
                + table(['Card', 'Progression', 'Lvl', 'Loyalty', 'Artifact', 'Power'], rows));
        }

        // ---- highest 5 (default) ----
        const top = ranked.slice(0, 5);
        const best = top.length ? cardProgressionScore(top[0]) : 1;
        const mean = cards.length
            ? Math.round(cards.reduce(function (a, c) { return a + cardProgressionScore(c); }, 0) / cards.length)
            : 0;
        let html = panelHead('Highest 5', 'the five most-progressed cards in the active deck');
        html += kpiGrid([
            kpi('Peak Progression', String(best), 'top card score'),
            kpi('Tracked Cards', String(cards.length), 'in active deck'),
            kpi('Mean Progression', String(mean), 'across the deck'),
        ]);
        html += top.map(function (c, i) {
            return barRow('#' + (i + 1) + ' ' + (c.name || ('#' + c.id)),
                cardProgressionScore(c), best, String(cardProgressionScore(c)));
        }).join('');
        html += '<p class="pp-hint">Progression score = (level × 100) + loyalty + artifact power. Integer-only.</p>';
        return setPanel(token, html);
    }

    // ========================================================================
    // TITLES — titles held and the full catalogue
    // ========================================================================
    // Only titles whose unlock condition is DERIVABLE from the portfolio's own
    // state are asserted. Every other title is reported as "not derivable here"
    // rather than guessed: the Portfolio never claims a title the engine has
    // not confirmed.
    function titleHeld(id, ctx) {
        if (id === 'first_blood') return ctx.wins >= 1;
        if (id === 'veteran') return ctx.wins >= 50;
        if (id === 'collector') return ctx.cards >= 20;
        return null;   // engine-held; not exported to this view
    }

    async function renderTitles(panel, sub, token) {
        const gs = gameState();
        const cards = cardsFromState(gs);
        const payload = await getJSON('/api/titles', 60000);
        const titles = (payload && Array.isArray(payload.titles)) ? payload.titles : [];
        if (!titles.length) {
            return setPanel(token, panelHead('Titles', 'title catalogue')
                + emptyNote('The title catalogue could not be read.'));
        }

        if (sub === 'catalogue') {
            const rows = titles.map(function (t) {
                return [t.name || t.id, t.rarity || '—', t.desc || '—', t.perk || '—'];
            });
            return setPanel(token, panelHead('Title catalogue', 'every title, its condition and its perk')
                + table(['Title', 'Rarity', 'Condition', 'Perk'], rows));
        }

        // ---- current (default) ----
        const rec = computeRecord(gs);
        const ctx = { wins: rec.wins, cards: cards.length };
        const held = [];
        const unknown = [];
        titles.forEach(function (t) {
            const st = titleHeld(t.id, ctx);
            if (st === true) held.push(t);
            else if (st === null) unknown.push(t);
        });
        let html = panelHead('Current titles', 'titles this portfolio can confirm are earned');
        html += kpiGrid([
            kpi('Confirmed', String(held.length), 'derivable from portfolio state'),
            kpi('Not Derivable', String(unknown.length), 'engine-held, not exported'),
            kpi('Catalogue', String(titles.length), 'total titles'),
            kpi('Wins', String(rec.wins), 'used by win-based titles'),
        ]);
        html += panelHead('Confirmed titles', 'earned on measurable conditions');
        html += (held.length
            ? chipRow(held.map(function (t) { return (t.icon ? t.icon + ' ' : '') + (t.name || t.id); }))
            : emptyNote('No titles are derivable from the current portfolio state.'));
        if (unknown.length) {
            html += panelHead('Not derivable here', 'the engine owns these; the Portfolio will not guess');
            html += chipRow(unknown.map(function (t) { return t.name || t.id; }));
            html += '<p class="pp-hint">Conditions tracked by the engine but not exported to this view '
                + 'include combo chains, flawless wins, mood wins, bounty captures and faith coherence. '
                + 'See the catalogue screen for each title\u2019s exact condition.</p>';
        }
        return setPanel(token, html);
    }

    // ========================================================================
    // MOODS — relative character and board moods
    // ========================================================================
    async function renderMoods(panel, sub, token) {
        const gs = gameState();
        const cards = cardsFromState(gs);

        // The engine keeps its OWN character-relationship scores and mood counters on
        // PlayerStats; the deck only reports live card moods. Both are shown.
        async function engineSocial(html) {
            const a = await engineAssoc();
            html += panelHead('Engine social record', 'relationships and mood counts held on PlayerStats');
            if (!activeWallet()) return html + connectNote();
            if (!a) return html + emptyNote('The engine association record could not be read.');
            if (a.present === false) return html + noEngineRecord();
            const rel = a.relationships || {};
            const moods = a.moods || {};
            html += kpiGrid([
                kpi('Relationships', String(countOf(rel)), 'characters with a score'),
                kpi('Mood Records', String(countOf(moods)), 'mood → times observed'),
            ]);
            html += panelHead('Relationship scores', 'character → score (0-100)');
            html += (countOf(rel) ? rowsTable(rel) : emptyNote('The engine records no character relationship for this wallet.'));
            html += panelHead('Engine mood counts', 'mood → times observed');
            html += (countOf(moods) ? rowsTable(moods) : emptyNote('The engine records no mood count for this wallet.'));
            return html;
        }

        if (sub === 'board') {
            const bm = gs.board_moods;
            let html = panelHead('Board moods', 'mood modifiers applied to the live board');
            if (bm == null) {
                return setPanel(token, html + emptyNote('No board moods reported (no active board).'));
            }
            if (Array.isArray(bm)) {
                const rows = bm.map(function (m, i) {
                    if (m && typeof m === 'object') return ['Slot ' + i, m.mood || m.name || '—', m.effect || '—'];
                    return ['Slot ' + i, String(m), '—'];
                });
                html += table(['Slot', 'Mood', 'Effect'], rows);
            } else {
                html += rowsTable(bm);
            }
            return setPanel(token, html);
        }

        if (sub === 'spread') {
            const dist = tally(cards, function (c) { return c.mood || 'Unknown'; });
            if (!dist.length) {
                return setPanel(token, panelHead('Mood spread', 'distribution of character moods')
                    + emptyNote('No cards available to measure mood spread.'));
            }
            const maxM = dist[0][1];
            const total = cards.length;
            let html = panelHead('Mood spread', 'relative weight of each mood in the active deck');
            html += dist.map(function (d) {
                return barRow(d[0], d[1], maxM, d[1] + '  (' + pct(d[1], total) + ')');
            }).join('');
            return setPanel(token, html);
        }

        // ---- character moods (default) ----
        if (!cards.length) {
            let html = panelHead('Character moods', 'mood state per character card')
                + emptyNote('No active deck reported, so no live card moods can be read.');
            html = await engineSocial(html);
            return setPanel(token, html);
        }
        const rows = cards.map(function (c) {
            return [c.name || ('#' + c.id),
                c.mood || '—',
                (c.loyalty == null ? '—' : String(c.loyalty)),
                (c.fatigue == null ? '—' : String(c.fatigue)),
                (c.fallen ? 'Fallen' : 'Active')];
        });
        const dist = tally(cards, function (c) { return c.mood || 'Unknown'; });
        let html = panelHead('Character moods', 'mood, loyalty and fatigue per character card')
            + kpiGrid([
                kpi('Characters', String(cards.length), 'measured'),
                kpi('Distinct Moods', String(dist.length), 'in deck'),
                kpi('Dominant Mood', dist.length ? dist[0][0] : '—', dist.length ? (dist[0][1] + ' cards') : '—'),
                kpi('Fallen', String(cards.filter(function (c) { return c.fallen === true; }).length), 'underworld status'),
            ])
            + table(['Character', 'Mood', 'Loyalty', 'Fatigue', 'State'], rows);
        html = await engineSocial(html);
        return setPanel(token, html);
    }

    // ========================================================================
    // ECONOMY — balances, holdings and philanthropy
    // ========================================================================
    async function renderEconomy(panel, sub, token) {
        const gs = gameState();
        const w = activeWallet();

        if (sub === 'holdings') {
            const portfolio = gs.portfolio;
            const html = panelHead('Holdings', 'entity shares carried by this player');
            if (!portfolio || !countOf(portfolio)) {
                return setPanel(token, html + emptyNote('No entity holdings recorded for this player.'));
            }
            const rows = Object.keys(portfolio).map(function (k) {
                return [k, String(portfolio[k])];
            });
            return setPanel(token, html + table(['Entity', 'Shares'], rows));
        }

        // ---- market & weather ----
        if (sub === 'markets') {
            const listings = pickArray(await getJSON('/api/entity/market/list', 30000), 'listings');
            const weather = await getJSON('/api/market/weather', 45000);
            const items = pickArray(await getWalletJSON('/api/items/collection', 30000), 'data');
            const mine = listings.filter(function (l) { return sameWallet(l.seller_wallet || l.owner_wallet, w); });
            let html = panelHead('Market & weather', 'entity market, market weather and your item collection');
            html += kpiGrid([
                kpi('Active Listings', String(listings.length), 'civilization marketplace'),
                kpi('Your Listings', w ? String(mine.length) : '—', 'offered by this wallet'),
                kpi('Collection Items', w ? String(items.length) : '—', 'built / bound items'),
                kpi('Market Weather', (weather && (weather.weather || weather.condition)) ? (weather.weather || weather.condition) : '—', 'engine-reported'),
            ]);
            if (weather) {
                html += panelHead('Weather payload', 'as the market engine reports it');
                html += rowsTable(weather);
            }
            html += panelHead('Your listings', 'what this wallet is offering');
            html += (mine.length ? objectArrayTable(mine, 9, 20) : emptyNote('This wallet has no active market listings.'));
            if (w) {
                html += panelHead('Item collection', 'items built or bound to this wallet');
                html += (items.length ? objectArrayTable(items, 8, 20) : emptyNote('No items are registered to this wallet.'));
            }
            return setPanel(token, html);
        }

        if (sub === 'philanthropy') {
            return setPanel(token, panelHead('Philanthropy', 'value given away and received')
                + kpiGrid([
                    kpi('Total Donated', fmtVBV(gs.total_donated || 0) + ' $VBV', 'lifetime'),
                    kpi('Reparations', (gs.reparations_received_count == null ? '—' : gs.reparations_received_count), 'payments received'),
                    kpi('Auctions Won', (gs.auctions_won == null ? '—' : gs.auctions_won), 'lifetime'),
                    kpi('Rumours Spread', (gs.rumor_count == null ? '—' : gs.rumor_count), 'influence spent'),
                ]));
        }

        // ---- balances (default) ----
        const tokens = await getWalletJSON('/api/player/tokens');
        const walletMicro = (gs.virtual_balance == null ? 0 : (Number(gs.virtual_balance) * MICRO));
        let html = panelHead('Balances', 'authoritative wallet and token state');
        html += kpiGrid([
            kpi('$VBV', fmtVBV(walletMicro), 'in-game balance'),
            kpi('Server Balance', (tokens && tokens.balance != null) ? fmtVBV(tokens.balance) : '—', 'engine-side ledger'),
            kpi('Salary', fmtVBV(gs.salary || 0) + ' $VBV', 'employment income'),
            kpi('Market Tokens', (gs.market_tokens == null ? '—' : String(gs.market_tokens)), 'market scrip'),
            kpi('Arena Vouchers', (gs.arena_vouchers == null ? '—' : String(gs.arena_vouchers)), 'arena entry'),
            kpi('Holdings', String(countOf(gs.portfolio)), 'entity positions'),
        ]);
        html += panelHead('Settlement tokens', 'native tokens scheduled for later settlement');
        html += kpiGrid([
            kpi('$NUGGET', (tokens && tokens.nugget_balance != null) ? String(tokens.nugget_balance) : 'roadmap', 'Algorand / Voi settlement'),
            kpi('$UNIT', (tokens && tokens.unit_balance != null) ? String(tokens.unit_balance) : 'roadmap', 'Algorand / Voi settlement'),
        ]);

        // Engine ledger mirrors + sustained-liquidity record (PlayerStats). The
        // liquidity window is what makes a career tier ELIGIBLE, so it is an
        // economic analytic in its own right.
        const assoc = await engineAssoc();
        html += panelHead('Engine ledger record', 'balances and liquidity the engine holds on PlayerStats');
        if (!w) return setPanel(token, html + connectNote());
        if (!assoc) return setPanel(token, html + emptyNote('The engine association record could not be read.'));
        if (assoc.present === false) return setPanel(token, html + noEngineRecord());
        const samples = Array.isArray(assoc.liquidity_samples) ? assoc.liquidity_samples : [];
        html += kpiGrid([
            kpi('Engine $VBV', fmtVBV(assoc.vbv_balance_micro || 0) + ' $VBV', 'playerBalances (authoritative)'),
            kpi('Credits', (assoc.credits == null ? '—' : fmtVBV(assoc.credits)), 'legacy micro-credit balance'),
            kpi('Market Tokens', (assoc.market_tokens_micro == null ? '—' : String(assoc.market_tokens_micro)), 'equity from liquidated loans'),
            kpi('Avg Sustained', fmtVBV(assoc.avg_sustained_micro || 0) + ' $VBV', 'career-tier eligibility'),
            kpi('Samples Kept', String(assoc.liquidity_sample_count == null ? '—' : assoc.liquidity_sample_count), 'window of ' + (assoc.liquidity_window_min == null ? '—' : assoc.liquidity_window_min) + ' min'),
            kpi('Samples Shown', String(samples.length), 'most recent, capped at 64'),
        ]);
        if (samples.length) {
            html += panelHead('Sustained liquidity samples', 'balance snapshot → micro-$VBV');
            html += table(['#', 'Balance'], samples.map(function (v, i) {
                return [String(i + 1), fmtVBV(v) + ' $VBV'];
            }));
        } else {
            html += emptyNote('The engine has recorded no liquidity sample for this wallet yet.');
        }
        html += panelHead('Ledger detail', 'donations, dividends, bonds and vouchers');
        html += kpiGrid([
            kpi('Total Donated', fmtVBV(assoc.total_donated_micro || 0) + ' $VBV', 'lifetime philanthropy'),
            kpi('Dividends Claimed', fmtVBV(assoc.total_dividend_claimed_micro || 0) + ' $VBV', 'lifetime'),
            kpi('Hunter Bond', fmtVBV(assoc.bounty_hunter_bond_micro || 0) + ' $VBV', 'locked deposit'),
            kpi('Arena Vouchers', (assoc.arena_vouchers == null ? '—' : String(assoc.arena_vouchers)), 'console reward units'),
            kpi('Yield Claims', String(countOf(assoc.last_claimed_yield)), 'entity → last claim'),
            kpi('Inventory Lines', String(countOf(assoc.inventory)), 'item → quantity'),
        ]);
        html += panelHead('Last claimed yield', 'entity → last claimed micro-$VBV');
        html += (countOf(assoc.last_claimed_yield) ? rowsTable(assoc.last_claimed_yield) : emptyNote('No dividend claim is recorded for this wallet.'));
        html += panelHead('Inventory', 'item → quantity');
        html += (countOf(assoc.inventory) ? rowsTable(assoc.inventory) : emptyNote('The engine records no inventory line for this wallet.'));
        html += '<p class="pp-hint">All figures are read as uint64 micro-units and formatted for display only.</p>';
        return setPanel(token, html);
    }

    // Builds a table from an array of objects using the UNION of their keys.
    // Used for endpoint payloads whose element shape is owned by the backend.
    function objectArrayTable(arr, maxKeys, maxRows) {
        const list = Array.isArray(arr) ? arr.slice(0, maxRows || 25) : [];
        if (!list.length) return emptyNote('Nothing recorded yet.');
        if (typeof list[0] !== 'object' || list[0] === null) {
            return table(['Value'], list.map(function (v) { return [String(v)]; }));
        }
        const keys = [];
        list.forEach(function (o) {
            Object.keys(o || {}).forEach(function (k) {
                if (keys.indexOf(k) === -1 && keys.length < (maxKeys || 7)) keys.push(k);
            });
        });
        const rows = list.map(function (o) {
            return keys.map(function (k) { return fmtAny(o[k]); });
        });
        return table(keys.map(humanKey), rows);
    }

    // ========================================================================
    // CAREERS — career tier, XP and constellation unlocks
    // ========================================================================
    async function renderCareers(panel, sub, token) {
        const gs = gameState();
        const w = activeWallet();
        if (!w) {
            return setPanel(token, panelHead('Careers', 'career progression analytics') + connectNote());
        }

        if (sub === 'unlocks') {
            const prog = await getWalletJSON('/api/player/progression');
            let html = panelHead('Constellation unlocks', 'unlock states across the constellation');
            if (!prog || prog.success === false) {
                return setPanel(token, html + emptyNote('No constellation progression reported for this wallet.'));
            }
            const features = prog.features || prog.Features;
            if (features && typeof features === 'object') {
                html += objectArrayTable(Object.keys(features).map(function (k) {
                    return { feature: k, state: features[k] };
                }), 4, 60);
            } else {
                html += rowsTable(prog);
            }
            return setPanel(token, html);
        }

        const cp = await getWalletJSON('/api/career/progress');
        if (!cp || cp.success === false) {
            return setPanel(token, panelHead('Careers', 'career progression analytics')
                + emptyNote('No career progression reported for this wallet yet.'));
        }

        if (sub === 'xp') {
            const xpObj = cp.role_xp || cp.RoleXP || cp.careers || cp.progress || cp.xp;
            return setPanel(token, panelHead('Career XP', 'experience accumulated per career')
                + (xpObj
                    ? (Array.isArray(xpObj) ? objectArrayTable(xpObj, 7, 40) : rowsTable(xpObj))
                    : emptyNote('No career XP reported yet.')));
        }

        // ---- tier (default) ----
        const tier = (cp.tier == null) ? (cp.Tier == null ? '—' : cp.Tier) : cp.tier;
        return setPanel(token, panelHead('Career tier', 'current career standing from the engine')
            + kpiGrid([
                kpi('Job Role', gs.job_role || cp.role || cp.Role || '—', 'active career'),
                kpi('Career Tier', String(tier), 'engine-declared'),
                kpi('Employer', gs.employer_id ? shortWallet(gs.employer_id) : '—', 'current club'),
                kpi('Social Rank', gs.social_rank || '—', 'club tier'),
            ])
            + panelHead('Career payload', 'everything the engine reports for this career')
            + rowsTable(cp));
    }

    // ========================================================================
    // ENTITIES — AI citizens, life assets and combined power
    // ========================================================================
    async function renderEntities(panel, sub, token) {
        const w = activeWallet();

        if (sub === 'citizens') {
            const list = await getJSON('/api/ai/citizens/list', 30000);
            const all = (list && Array.isArray(list.data)) ? list.data : [];
            const mine = w ? all.filter(function (c) {
                const ow = String(c.OwnerWallet || c.owner_wallet || '').toLowerCase();
                const or = String(c.OriginWallet || c.origin_wallet || '').toLowerCase();
                const me = w.toLowerCase();
                return ow === me || or === me;
            }) : [];
            let html = panelHead('AI Citizens', 'citizens this wallet owns or originated');
            html += kpiGrid([
                kpi('Owned / Originated', w ? String(mine.length) : '—', 'this wallet'),
                kpi('Total Citizens', String(all.length), 'in the civilization'),
                kpi('Share', w ? pct(mine.length, all.length) : '—', 'of all citizens'),
            ]);
            html += (w
                ? (mine.length ? objectArrayTable(mine, 6, 30) : emptyNote('No AI citizens owned by this wallet.'))
                : connectNote());
            return setPanel(token, html);
        }

        // ---- children bots (derived citizens) ----
        if (sub === 'children') {
            const bots = pickArray(await getWalletJSON('/api/children-bots', 30000), 'children_bots');
            let html = panelHead('Children bots', 'derived entities raised by this wallet');
            html += kpiGrid([
                kpi('Children Bots', String(bots.length), 'owned or originated'),
                kpi('Certified', String(countWhere(bots, function (b) { return !!b.certified; })), 'legitimate lineage (§27.7.3)'),
                kpi('Combined Level', String(sumMicro(bots, function (b) { return b.bot_level; })), 'summed bot levels'),
                kpi('Faith-Bound', String(countWhere(bots, function (b) { return !!b.religion; })), 'adopted a religion'),
            ]);
            if (!w) return setPanel(token, html + connectNote());
            html += panelHead('Roster', 'career, tier and dominant axis per bot');
            html += (bots.length
                ? table(['Bot', 'Career', 'Tier', 'Bot Level', 'Dominant', 'Region'], bots.map(function (b) {
                    return [b.name || shortWallet(b.wallet), b.career || '—',
                        String(b.tier == null ? '—' : b.tier),
                        String(b.bot_level == null ? '—' : b.bot_level),
                        dominantEntityStat(b.stats), b.region || '—'];
                }))
                : emptyNote('No children bots are associated with this wallet.'));
            html += panelHead('Stat vectors', 'per-axis power for every child bot');
            bots.forEach(function (b) {
                html += '<div class="pf-quote">' + esc(b.name || shortWallet(b.wallet)) + '</div>' + statBars(b.stats);
            });
            html += panelHead('Full payload', 'every field the citizen engine reports');
            html += (bots.length ? objectArrayTable(bots, 14, 20) : emptyNote('Nothing further is reported.'));
            return setPanel(token, html);
        }

        // ---- zen garden ----
        if (sub === 'garden') {
            const garden = await getWalletJSON('/api/garden', 30000);
            const elements = pickArray(garden, 'elements');
            let html = panelHead('Zen garden', 'nurturing analytics derived from your entities');
            html += kpiGrid([
                kpi('Garden Level', (garden && garden.level != null) ? String(garden.level) : '—', 'engine-reported'),
                kpi('Meditation Streak', (garden && garden.meditation_streak != null) ? String(garden.meditation_streak) : '—', 'rituals completed'),
                kpi('Tended Entities', String(elements.length), 'contributing citizens'),
                kpi('Nurturing', (gameState().nurturing == null ? '—' : gameState().nurturing), 'effective modifier'),
            ]);
            if (!w) return setPanel(token, html + connectNote());
            html += panelHead('Tended entities', 'citizens feeding this garden');
            html += (elements.length ? objectArrayTable(elements, 5, 20) : emptyNote('No entities are tending this garden.'));
            return setPanel(token, html);
        }

        if (sub === 'assets') {
            const orphan = await getWalletJSON('/api/orphan/status');
            const gs = gameState();
            let html = panelHead('Life assets', 'companions and orphaned entities');
            html += kpiGrid([
                kpi('Pets', String(countOf(gs.pets)), 'companions'),
                kpi('Children Bots', String(countOf(gs.children_bots)), 'derived entities'),
                kpi('Orphan View', (orphan ? 'reported' : 'unavailable'), 'server view'),
            ]);
            if (!w) return setPanel(token, html + connectNote());
            html += panelHead('Orphan payload', 'as reported by the engine');
            html += (orphan ? rowsTable(orphan) : emptyNote('No orphan status reported.'));
            html += '<p class="pp-hint">Life-asset management is owned by the World Dashboard (Life &amp; Assets).</p>';
            return setPanel(token, html);
        }

        // ---- combined power (default) ----
        const owned = await getWalletJSON('/api/owner/combined-stats');
        const overlays = await getWalletJSON('/api/stat-overlay/owner');
        let html = panelHead('Combined power', 'aggregate power across every owned entity');
        html += kpiGrid([
            kpi('Effective Power', (owned && owned.effective_power != null) ? owned.effective_power : '—', 'capped level'),
            kpi('Overlays', (overlays && Array.isArray(overlays.overlays)) ? overlays.overlays.length : '—', 'power overlays'),
        ]);
        if (!w) return setPanel(token, html + connectNote());
        html += panelHead('Combined stats', 'server-authored');
        html += (owned && owned.combined_stats) ? rowsTable(owned.combined_stats) : emptyNote('No combined stats reported.');
        if (overlays && Array.isArray(overlays.overlays) && overlays.overlays.length) {
            html += panelHead('Overlays', 'per-entity effective power');
            html += objectArrayTable(overlays.overlays, 5, 30);
        }
        return setPanel(token, html);
    }

    // ========================================================================
    // ACHIEVEMENTS — unlocked achievements and completion analytics
    // ========================================================================
    async function renderAchievements(panel, sub, token) {
        const payload = await achievementsPayload();
        if (!payload) {
            return setPanel(token, panelHead('Achievements', 'achievement analytics')
                + emptyNote('Achievement data could not be read.'));
        }
        const unlocked = Array.isArray(payload.unlocked) ? payload.unlocked : [];
        const defs = Array.isArray(payload.definitions) ? payload.definitions : [];
        const progress = (payload.progress && typeof payload.progress === 'object') ? payload.progress : {};

        if (sub === 'progress') {
            const rows = defs.map(function (d) {
                const id = d.id || d.ID || '';
                const isOn = unlocked.indexOf(id) !== -1;
                const pv = (progress[id] == null) ? 0 : progress[id];
                return [d.name || d.title || id, d.rarity || '—', isOn ? 'Unlocked' : 'Locked', String(pv)];
            });
            return setPanel(token, panelHead('Completion', 'catalogue-wide completion state')
                + kpiGrid([
                    kpi('Completion', pct(unlocked.length, defs.length), unlocked.length + ' of ' + defs.length),
                    kpi('Unlocked', String(unlocked.length), 'earned'),
                    kpi('Remaining', String(Math.max(defs.length - unlocked.length, 0)), 'to earn'),
                    kpi('Tracked Counters', String(countOf(progress)), 'incremental progress'),
                ])
                + (defs.length ? table(['Achievement', 'Rarity', 'State', 'Progress'], rows)
                    : emptyNote('No achievement definitions reported.')));
        }

        // ---- unlocked (default) ----
        const rows = defs.filter(function (d) {
            return unlocked.indexOf(d.id || d.ID || '') !== -1;
        }).map(function (d) {
            return [d.name || d.title || d.id, d.rarity || '—', d.description || d.desc || '—'];
        });
        return setPanel(token, panelHead('Unlocked achievements', 'earned by this player')
            + kpiGrid([
                kpi('Unlocked', String(unlocked.length), 'of ' + defs.length + ' in catalogue'),
                kpi('Completion', pct(unlocked.length, defs.length), 'catalogue-wide'),
            ])
            + (rows.length ? table(['Achievement', 'Rarity', 'Detail'], rows) : emptyNote('No achievements unlocked yet.')));
    }

    // ========================================================================
    // ========================================================================
    // OBLIGATIONS — loans, leases and governance weight
    // ========================================================================
    async function renderObligations(panel, sub, token) {
        const w = activeWallet();

        // ---- leases ----
        if (sub === 'leases') {
            const leases = pickArray(await getWalletJSON('/api/lease/list', 30000), 'leases');
            const lent = countWhere(leases, function (l) { return sameWallet(l.lender_wallet, w); });
            const borrowed = countWhere(leases, function (l) { return sameWallet(l.borrower_wallet, w); });
            const expired = countWhere(leases, function (l) {
                return !!l.expires_at && new Date(l.expires_at).getTime() < Date.now();
            });
            let html = panelHead('Leases', 'card leases lent or borrowed by this wallet');
            html += kpiGrid([
                kpi('Leases', String(leases.length), 'total'),
                kpi('Lent Out', String(lent), 'as lender'),
                kpi('Borrowed', String(borrowed), 'as borrower'),
                kpi('Expired', String(expired), 'past expiry'),
            ]);
            if (!w) return setPanel(token, html + connectNote());
            if (!leases.length) return setPanel(token, html + emptyNote('No leases involve this wallet.'));
            html += panelHead('Lease book', 'card, counterparty, price and expiry');
            html += table(['Card', 'Club', 'Price', 'Hours', 'Expires', 'Borrower'], leases.map(function (l) {
                return [l.card_name || (l.card_id == null ? '—' : ('#' + l.card_id)), l.club_id || '—',
                    fmtVBV(l.price || 0) + ' $VBV',
                    String(l.duration_hours == null ? '—' : l.duration_hours),
                    l.expires_at ? fmtWhen(l.expires_at) : '—',
                    l.borrower_wallet ? shortWallet(l.borrower_wallet) : '—'];
            }));
            html += panelHead('Full lease payload', 'every field the lease engine reports');
            html += objectArrayTable(leases, 9, 20);
            return setPanel(token, html);
        }

        // ---- governance weight ----
        if (sub === 'governance') {
            const payload = await getWalletJSON('/api/governance/weight', 30000);
            const weight = (payload && payload.weight) ? payload.weight : null;
            let html = panelHead('Governance weight', 'the axes that determine voting power');
            html += kpiGrid([
                kpi('Total Score', weight && weight.total_score != null ? String(weight.total_score) : '—', 'aggregate voting weight'),
                kpi('Trust', weight && weight.trust_score != null ? String(weight.trust_score) : '—', 'axis'),
                kpi('Reputation', weight && weight.reputation_score != null ? String(weight.reputation_score) : '—', 'axis'),
                kpi('Economic', weight && weight.economic_score != null ? String(weight.economic_score) : '—', 'axis'),
                kpi('Competitive', weight && weight.competitive_score != null ? String(weight.competitive_score) : '—', 'axis'),
                kpi('Community', weight && weight.community_score != null ? String(weight.community_score) : '—', 'axis'),
                kpi('Creative', weight && weight.creative_score != null ? String(weight.creative_score) : '—', 'axis'),
            ]);
            if (!w) return setPanel(token, html + connectNote());
            if (!weight) return setPanel(token, html + emptyNote('No governance weight is reported for this wallet.'));
            html += panelHead('Axis breakdown', 'score per contribution axis');
            html += rowsTable(weight);
            const maxAxis = Math.max(1,
                Number(weight.trust_score) || 0, Number(weight.reputation_score) || 0,
                Number(weight.economic_score) || 0, Number(weight.competitive_score) || 0,
                Number(weight.community_score) || 0, Number(weight.creative_score) || 0);
            html += panelHead('Relative contribution', 'each axis against the strongest axis');
            html += barRow('Trust', weight.trust_score, maxAxis, String(weight.trust_score));
            html += barRow('Reputation', weight.reputation_score, maxAxis, String(weight.reputation_score));
            html += barRow('Economic', weight.economic_score, maxAxis, String(weight.economic_score));
            html += barRow('Competitive', weight.competitive_score, maxAxis, String(weight.competitive_score));
            html += barRow('Community', weight.community_score, maxAxis, String(weight.community_score));
            html += barRow('Creative', weight.creative_score, maxAxis, String(weight.creative_score));
            return setPanel(token, html);
        }
        // ---- loans (default) ----
        const loans = pickArray(await getWalletJSON('/api/loans', 30000), null);
        const activeLoans = countWhere(loans, function (l) { return (l.status || '') === 'active'; });
        const owedMicro = sumMicro(loans, function (l) {
            return ((l.status || '') === 'active') ? (Number(l.repayment_amount) || 0) : 0;
        });
        let html = panelHead('Loans', 'borrowed against card bundles, with interest');
        html += kpiGrid([
            kpi('Loans', String(loans.length), 'recorded'),
            kpi('Active', String(activeLoans), 'outstanding'),
            kpi('Repayment Due', fmtVBV(owedMicro) + ' $VBV', 'principal + interest'),
            kpi('Market Tokens', (gameState().market_tokens == null ? '—' : String(gameState().market_tokens)), 'equity from liquidations'),
        ]);
        if (!w) return setPanel(token, html + connectNote());
        if (!loans.length) return setPanel(token, html + emptyNote('This wallet carries no loans.'));
        html += panelHead('Loan book', 'amounts, due dates and status');
        html += table(['Loan', 'Principal', 'Repayment', 'Due', 'Status'], loans.map(function (l) {
            return [shortWallet(l.id),
                fmtVBV(l.loan_amount || 0) + ' $VBV',
                fmtVBV(l.repayment_amount || 0) + ' $VBV',
                l.due_at ? fmtWhen(l.due_at) : '—',
                l.status || '—'];
        }));
        html += panelHead('Full loan payload', 'every field the loan service reports');
        html += objectArrayTable(loans, 9, 20);
        return setPanel(token, html);
    }



    // ========================================================================
    // LEGAL & CUSTODY — warrants, detained cards and open bounties
    // ------------------------------------------------------------------------
    // Detained-card state is exported by the engine on `GetGameState()`
    // (jailed_cards / kidnapped_cards / held_hostage_cards). `PlayerStats.Wanted`
    // (the per-agency warrant map) is NOT exported by any endpoint — that is
    // stated rather than shown as an empty list.
    // ========================================================================
    async function renderCustody(panel, sub, token) {
        const gs = gameState();
        const w = activeWallet();
        const jailed = gs.jailed_cards || {};
        const kidnapped = gs.kidnapped_cards || {};
        const hostages = gs.held_hostage_cards || {};

        // ---- custody ----
        if (sub === 'custody') {
            let html = panelHead('Custody', 'cards detained by, or taken from, this wallet');
            html += kpiGrid([
                kpi('Cards In Jail', String(countOf(jailed)), 'held by organisations'),
                kpi('Cards Kidnapped', String(countOf(kidnapped)), 'taken from this wallet'),
                kpi('Hostages Held', String(countOf(hostages)), 'taken by this wallet'),
                kpi('Active Hostages', (gs.active_hostage_count == null ? '—' : gs.active_hostage_count), 'engine-reported'),
            ]);
            html += panelHead('Jailed cards', 'card → holding club');
            html += (countOf(jailed) ? rowsTable(jailed) : emptyNote('No cards of this wallet are in jail.'));
            html += panelHead('Kidnapped cards', 'card → wallet holding it');
            html += (countOf(kidnapped) ? rowsTable(kidnapped) : emptyNote('No cards have been kidnapped from this wallet.'));
            html += panelHead('Hostage cards', 'card → wallet it was taken from');
            html += (countOf(hostages) ? rowsTable(hostages) : emptyNote('This wallet holds no hostage cards.'));

            // Engine-owned custody maps. These live on PlayerStats, not on the WASM
            // snapshot, so they are read from the read-only association endpoint.
            const a = await engineAssoc();
            html += panelHead('Engine custody record', 'the maps the engine holds on PlayerStats');
            if (!w) return setPanel(token, html + connectNote());
            if (!a) return setPanel(token, html + emptyNote('The engine association record could not be read.'));
            if (a.present === false) return setPanel(token, html + noEngineRecord());
            const jC = a.jailed_cards || {};
            const kC = a.kidnapped_cards || {};
            const hC = a.held_hostage_cards || {};
            html += kpiGrid([
                kpi('In Jail', String(countOf(jC)), 'engine-owned map'),
                kpi('Kidnapped', String(countOf(kC)), 'engine-owned map'),
                kpi('Hostages Held', String(countOf(hC)), 'engine-owned map'),
                kpi('Captured Outlaws', String(countOf(a.captured_outlaws)), 'bounty record'),
                kpi('Recovery Bounties', String(countOf(a.recovery_bounties)), 'cards being reclaimed'),
                kpi('Active Hostages', (a.active_hostage_count == null ? '—' : String(a.active_hostage_count)), 'engine-reported'),
            ]);
            html += panelHead('Engine jailed cards', 'card → holding club');
            html += (countOf(jC) ? rowsTable(jC) : emptyNote('The engine records no jailed cards for this wallet.'));
            html += panelHead('Engine kidnapped cards', 'card → wallet holding it');
            html += (countOf(kC) ? rowsTable(kC) : emptyNote('The engine records no kidnapped cards for this wallet.'));
            html += panelHead('Engine hostage cards', 'card → wallet it was taken from');
            html += (countOf(hC) ? rowsTable(hC) : emptyNote('The engine records no hostage cards for this wallet.'));
            html += panelHead('Captured outlaws', 'unique bounty captures on record');
            html += (countOf(a.captured_outlaws) ? rowsTable(a.captured_outlaws) : emptyNote('No outlaws are recorded as captured by this wallet.'));
            html += panelHead('Recovery bounties', 'card → reward offered for its return');
            html += (countOf(a.recovery_bounties) ? rowsTable(a.recovery_bounties) : emptyNote('No recovery bounties are posted by this wallet.'));
            return setPanel(token, html);
        }

        // ---- bounties ----
        if (sub === 'bounties') {
            const active = pickArray(await getJSON('/api/bounty/active', 45000), 'bounties');
            const dash = await getJSON('/api/justice/dashboard', 45000);
            const wanted = pickArray(dash, 'bounties');
            const mine = active.filter(function (b) {
                return sameWallet(b.target_wallet || b.target, w) || sameWallet(b.hunter_wallet, w);
            });
            let html = panelHead('Bounties', 'open bounties and the civilization wanted board');
            html += kpiGrid([
                kpi('Open Bounties', String(active.length), 'civilization-wide'),
                kpi('Involving You', String(mine.length), 'as target or hunter'),
                kpi('Wanted Targets', String(wanted.length), 'above the justice threshold'),
                kpi('Bounty Licence', gs.bounty_license_active ? 'Active' : 'Inactive', 'hunter authority'),
            ]);
            html += panelHead('Bounties involving this wallet', 'posted or pursued');
            html += (mine.length ? objectArrayTable(mine, 8, 20) : emptyNote('No bounties involve this wallet.'));
            html += panelHead('Wanted board', 'outlaws the justice layer has flagged');
            html += (wanted.length
                ? table(['Target', 'Name', 'Wanted Level', 'Reward'], wanted.map(function (b) {
                    return [shortWallet(b.target_wallet), b.target_name || '—',
                        String(b.wanted_level == null ? '—' : b.wanted_level),
                        fmtVBV(b.reward || 0) + ' $VBV'];
                }))
                : emptyNote('No targets currently meet the justice threshold.'));
            return setPanel(token, html);
        }

        // ---- warrants (default) ----
        const dash = await getJSON('/api/justice/dashboard', 45000);
        const wanted = pickArray(dash, 'bounties');
        let html = panelHead('Warrants', 'your heat and the civilization warrant board');
        html += kpiGrid([
            kpi('Wanted Level', (gs.wanted_level == null ? '—' : gs.wanted_level), 'your heat'),
            kpi('Wanted Targets', String(wanted.length), 'flagged by the justice layer'),
            kpi('Reparations Paid To You', (gs.reparations_received_count == null ? '—' : gs.reparations_received_count), 'settlements received'),
            kpi('Market Freeze', gs.market_frozen_until ? fmtWhen(gs.market_frozen_until) : 'not frozen', 'trade restriction'),
        ]);
        html += panelHead('Civilization warrant board', 'targets, heat and posted reward');
        html += (wanted.length
            ? table(['Target', 'Name', 'Wanted Level', 'Reward'], wanted.map(function (b) {
                return [shortWallet(b.target_wallet), b.target_name || '—',
                    String(b.wanted_level == null ? '—' : b.wanted_level),
                    fmtVBV(b.reward || 0) + ' $VBV'];
            }))
            : emptyNote('No warrants are posted against any player.'));

        // The engine's own warrant network + heat state (PlayerStats.Wanted and the
        // risk counters that determine heist eligibility).
        const a = await engineAssoc();
        html += panelHead('Your warrant network', 'per-agency warrants held against this wallet by the engine');
        if (!w) return setPanel(token, html + connectNote());
        if (!a) return setPanel(token, html + emptyNote('The engine association record could not be read.'));
        if (a.present === false) return setPanel(token, html + noEngineRecord());
        html += kpiGrid([
            kpi('Agencies With Warrants', String(countOf(a.wanted)), 'engine-owned map'),
            kpi('Engine Wanted Level', (a.wanted_level == null ? '—' : String(a.wanted_level)), 'risk factor for heists'),
            kpi('Heist Attempts', (a.heist_attempts == null ? '—' : String(a.heist_attempts)), 'lifetime'),
            kpi('Audited Clubs', String(countOf(a.audited_clubs)), 'cyber-audit targets'),
            kpi('Rumours Spread', (a.rumor_count == null ? '—' : String(a.rumor_count)), 'influence spent'),
            kpi('Disconnect Streak', (a.disconnect_streak == null ? '—' : String(a.disconnect_streak)), 'reliability'),
        ]);
        html += panelHead('Warrants by agency', 'agency → warrant held');
        html += (countOf(a.wanted) ? rowsTable(a.wanted) : emptyNote('The engine holds no warrant against this wallet.'));
        html += panelHead('Engine heat & cooldowns', 'timers the engine reports for this wallet');
        html += kpiGrid([
            kpi('Market Freeze', a.market_frozen_until ? fmtWhen(a.market_frozen_until) : 'not frozen', 'trade restriction'),
            kpi('Ghost Protocol', a.ghost_protocol_expires_at ? fmtWhen(a.ghost_protocol_expires_at) : 'inactive', 'signal scrambling'),
            kpi('Deep Scan', a.last_deep_scan_at ? fmtWhen(a.last_deep_scan_at) : 'never', 'intel-agent cooldown'),
            kpi('District Scanner', a.district_scanner_expires_at ? fmtWhen(a.district_scanner_expires_at) : 'inactive', 'active scanning'),
            kpi('Disruptor', a.disruptor_cooldown_at ? fmtWhen(a.disruptor_cooldown_at) : 'ready', 'hunter item cooldown'),
            kpi('Cloak Disrupted', a.cloak_disrupted_until ? fmtWhen(a.cloak_disrupted_until) : 'no', 'revealed by a hunter'),
            kpi('Ban Expiry', a.ban_expires ? fmtWhen(a.ban_expires) : 'not banned', 'moderation'),
            kpi('Gloat Ban', a.gloat_banned_until ? fmtWhen(a.gloat_banned_until) : 'not banned', 'moderation'),
        ]);
        return setPanel(token, html);
    }

    // ========================================================================
    // FAITH — churches owned (§32), religion governance and region coherence
    // ========================================================================
    async function renderFaith(panel, sub, token) {
        const w = activeWallet();

        // ---- religion governance ----
        if (sub === 'religion') {
            const payload = await getJSON('/api/faith/religions', 45000);
            let religions = pickArray(payload, 'high_tier');
            if (!religions.length) religions = pickArray(payload, 'religions');
            const governing = religions.filter(function (r) { return sameWallet(r.governor, w); });
            const memberOf = religions.filter(function (r) { return hasMember(r.members, w); });
            let html = panelHead('Religion', 'religion-governance standing (§32)');
            html += kpiGrid([
                kpi('Religion Cap', (payload && payload.cap != null) ? String(payload.cap) : '—', 'faucet-owned chain store'),
                kpi('Listed', String(religions.length), 'religions reported'),
                kpi('Governed', String(governing.length), 'you hold the governorship'),
                kpi('Memberships', String(memberOf.length), 'religions you belong to'),
            ]);
            if (!w) return setPanel(token, html + connectNote());
            if (!religions.length) return setPanel(token, html + emptyNote('No religions are reported by the governance engine.'));
            html += panelHead('Religions you govern', 'governorship, power and coherence');
            html += (governing.length
                ? table(['Religion', 'Dogma', 'Region', 'Faith Power', 'Coherence'], governing.map(function (r) {
                    return [r.name || r.religion_id || '—', r.dogma || '—', r.region || '—',
                        String(r.faith_power == null ? '—' : r.faith_power),
                        String(r.coherence == null ? '—' : r.coherence)];
                }))
                : emptyNote('This wallet governs no religion.'));
            html += panelHead('Memberships', 'religions you belong to without governing');
            html += (memberOf.length
                ? table(['Religion', 'Dogma', 'Region', 'Governor'], memberOf.map(function (r) {
                    return [r.name || r.religion_id || '—', r.dogma || '—', r.region || '—',
                        r.governor ? shortWallet(r.governor) : 'faucet-owned'];
                }))
                : emptyNote('This wallet belongs to no religion.'));
            html += panelHead('Full religion payload', 'every field the governance engine reports');
            html += objectArrayTable(religions, 12, 24);
            return setPanel(token, html);
        }

        // ---- region coherence ----
        if (sub === 'coherence') {
            const region = state.activeRegion || '';
            const coh = await getJSON('/api/faith/coherence' + (region ? ('?region=' + encodeURIComponent(region)) : ''), 45000);
            let html = panelHead('Region coherence', 'faith, domestic and legitimacy scores (§27.7)');
            html += kpiGrid([
                kpi('Region', (coh && coh.region) ? coh.region : (region || 'Base'), state.activeRegion ? 'selected region' : 'engine default'),
                kpi('Faith Coherence', (coh && coh.faith_coherence != null) ? String(coh.faith_coherence) : '—', 'ritual / dogma alignment'),
                kpi('Domestic Coherence', (coh && coh.domestic_coherence != null) ? String(coh.domestic_coherence) : '—', 'bond-graph density'),
                kpi('Entity Legitimacy', (coh && coh.entity_legitimacy != null) ? String(coh.entity_legitimacy) : '—', 'certified vs off-ledger'),
                kpi('World Dynamics Score', (coh && coh.score != null) ? String(coh.score) : '—', 'weighted region score'),
            ]);
            if (!coh) return setPanel(token, html + emptyNote('Region coherence could not be read.'));
            html += panelHead('Weightings', 'the ten-signal world-dynamics signature (§27.4)');
            html += rowsTable(coh);
            html += '<p class="pp-hint">Coherence is reported PER REGION, not per wallet — a wallet’s standing is its membership in the churches and religions above.</p>';
            return setPanel(token, html);
        }

        // ---- churches (default) ----
        const churches = pickArray(await getWalletJSON('/api/church/owner', 30000), 'churches');
        const mine = churches.filter(function (c) { return sameWallet(c.owner, w) || sameWallet(c.Owner, w); });
        let html = panelHead('Churches', 'faith organisations owned by this wallet (§32)');
        html += kpiGrid([
            kpi('Churches', String(mine.length), 'owned'),
            kpi('Faith Power', String(sumMicro(mine, function (c) { return c.faith_power; })), 'combined'),
            kpi('Members', String(sumMicro(mine, function (c) { return c.member_count; })), 'congregations'),
            kpi('Rituals', String(sumMicro(mine, function (c) { return c.rituals_done; })), 'performed'),
        ]);
        if (!w) return setPanel(token, html + connectNote());
        if (!mine.length) return setPanel(token, html + emptyNote('This wallet owns no churches.'));
        html += panelHead('Congregations', 'power, doctrine and funds per church');
        html += table(['Church', 'Dogma', 'Region', 'Faith Power', 'Members', 'Ritual Fees'], mine.map(function (c) {
            return [c.name || c.id || '—', c.dogma_tag || '—', c.region || '—',
                String(c.faith_power == null ? '—' : c.faith_power),
                String(c.member_count == null ? '—' : c.member_count),
                fmtVBV(c.ritual_fees || 0) + ' $VBV'];
        }));
        html += panelHead('Full church payload', 'every field the faith engine reports');
        html += objectArrayTable(mine, 14, 20);
        return setPanel(token, html);
    }

    // ========================================================================
    // BONDED ASSETS — §23 bonded registry, §25.7 world content, share holdings
    // ========================================================================
    async function renderBondedAssets(panel, sub, token) {
        const w = activeWallet();
        const gs = gameState();

        // ---- asset market (§10.8) ----
        // A READ-ONLY analytics screen: what this wallet is offering, what it has sold, and what is
        // for sale across the civilization. It states the SERVER's fee and acquisition rules verbatim
        // rather than inventing a policy, and it carries NO purchase control (§3: the Portfolio never
        // transacts — the studio and the World Dashboard own the actions).
        if (sub === 'market') {
            const market = await getWalletJSON('/api/assets/market', 30000);
            const rows = pickArray(market, 'listings');
            if (!market) {
                return setPanel(token, panelHead('Asset market', 'player-to-player bonded-asset sales (§10.8)') +
                    emptyNote('The market could not be read (rate-limited or unavailable).'));
            }
            const active = rows.filter(function (r) { return r.status === 'active'; });
            const mine = rows.filter(function (r) { return r.mine; });
            const sold = rows.filter(function (r) { return r.status === 'sold'; });
            const stale = rows.filter(function (r) { return !!r.stale_reason; });
            let html = panelHead('Asset market', 'player-to-player bonded-asset sales — read-only analytics (§10.8)');
            html += kpiGrid([
                kpi('For Sale', String(active.length), 'active listings in the world'),
                kpi('Yours Listed', String(mine.filter(function (r) { return r.status === 'active'; }).length), 'assets you are offering'),
                kpi('Your Sales', String(mine.filter(function (r) { return r.status === 'sold'; }).length), 'listings of yours that sold'),
                kpi('Unbuyable (stale)', String(stale.length), 'listings whose seller no longer owns the art'),
            ]);
            if (!rows.length) {
                html += emptyNote('No listings exist yet — nobody has offered a bonded asset for sale.');
                return setPanel(token, html);
            }
            html += objectArrayTable('Every listing', rows, [
                'status', 'asset_name', 'asset_id', 'seller', 'price_micro', 'fee_micro', 'net_micro',
                'buyer', 'mine', 'stale_reason', 'note',
            ]);
            html += quoteNote('The engine\'s rule', String(market.rule || '') + ' ' + String(market.acquisition_rule || ''));
            html += quoteNote('Served fee', String(market.fee_bps) + ' bps of the price, routed to the faucet sink.');
            return setPanel(token, html);
        }

        // ---- branding coverage (§23.5) ----
        // Bonded assets brand every entity a wallet owns — EXCEPT cards, which the engine refuses
        // outright (both bind and enumeration). This screen reports BOTH facts: where each asset is
        // worn, and which entities are still unpainted. The rule text is the SERVER's, never this
        // screen's: an analytics surface states what the engine reports, nothing more.
        if (sub === 'branding') {
            const assetsRes = await getWalletJSON('/api/assets', 30000);
            const assets = pickArray(assetsRes, 'assets');
            const bindings = pickArray(assetsRes, 'bindings');
            const policy = (assetsRes && assetsRes.policy) || {};
            const targets = pickArray(await getWalletJSON('/api/assets/targets', 30000), 'targets');
            const branded = targets.filter(function (t) { return (t.branding || []).length > 0; });
            const byAsset = {};
            bindings.forEach(function (b) {
                if (!byAsset[b.asset_id]) byAsset[b.asset_id] = [];
                byAsset[b.asset_id].push(b);
            });
            let html = panelHead('Branding coverage', 'where this wallet\'s bonded art is worn across the ecosphere (§23.5)');
            html += kpiGrid([
                kpi('Bonded Assets', w ? String(assets.length) : '—', 'art this wallet has painted'),
                kpi('Worn', String(bindings.length), 'asset → entity bindings'),
                kpi('Entities Branded', String(branded.length), 'entities wearing your art'),
                kpi('Unpainted', String(Math.max(0, targets.length - branded.length)), 'your entities with no art'),
            ]);
            if (!w) return setPanel(token, html + connectNote());
            html += panelHead('Your bonded assets', 'each asset and every entity wearing it');
            html += (assets.length
                ? table(['Asset', 'Type', 'Art', 'Worn on'], assets.map(function (a) {
                    const worn = (byAsset[a.asset_id] || []).map(function (b) {
                        return (b.target_kind || b.slot || 'slot') + ' ' + shortWallet(b.target_id);
                    });
                    return [a.name || shortWallet(a.asset_id),
                        String(a.asset_type),
                        (a.media && a.media.mime_type) ? a.media.mime_type : 'no art',
                        worn.length ? worn.join(', ') : 'not worn'];
                }))
                : emptyNote('This wallet has painted no bonded assets yet.'));
            html += panelHead('Branded entities', 'the entities whose art is your bonded asset');
            html += (branded.length
                ? table(['Entity', 'Category', 'Wearing'], branded.map(function (t) {
                    return [t.name || shortWallet(t.target_id),
                        t.kind_label || t.kind,
                        (t.branding || []).map(function (b) { return b.asset_name || shortWallet(b.asset_id); }).join(', ')];
                }))
                : emptyNote('No entity owned by this wallet wears a bonded asset.'));
            if (policy.rule) {
                html += panelHead('The rule', 'served by the engine, not asserted by this screen');
                html += '<p class="pf-quote">' + esc(policy.rule) +
                    ((policy.blocked_target_kinds || []).length
                        ? ' Blocked categories: ' + esc(policy.blocked_target_kinds.join(', ')) + '.'
                        : '') + '</p>';
            }
            return setPanel(token, html);
        }

        // ---- world content ----
        if (sub === 'world') {
            const content = pickArray(await getJSON('/api/world-content', 45000), 'data');
            const mine = content.filter(function (c) { return sameWallet(c.creator, w); });
            const bots = pickArray(await getWalletJSON('/api/children-bots', 30000), 'children_bots');
            let html = panelHead('World content', 'authored world entities and derived bots (§25.7)');
            html += kpiGrid([
                kpi('Authored', w ? String(mine.length) : '—', 'your NPC / animal / scenery / weather'),
                kpi('Deployed', String(countWhere(mine, function (c) { return !!c.deployed; })), 'placed in a region'),
                kpi('Content In World', String(content.length), 'civilization-wide'),
                kpi('Children Bots', w ? String(bots.length) : '—', 'derived entities (§26)'),
            ]);
            if (!w) return setPanel(token, html + connectNote());
            html += panelHead('Your world content', 'each authored entity and its deployment');
            html += (mine.length
                ? table(['Content', 'Kind', 'Region', 'Deployed'], mine.map(function (c) {
                    return [shortWallet(c.content_id), c.kind || '—', c.region || '—', c.deployed ? 'Yes' : 'No'];
                }))
                : emptyNote('This wallet has authored no world content.'));
            html += panelHead('Children bots', 'derived citizens owned or originated by this wallet');
            html += (bots.length ? objectArrayTable(bots, 8, 20) : emptyNote('No children bots are associated with this wallet.'));
            return setPanel(token, html);
        }

        // ---- share holdings ----
        if (sub === 'holdings') {
            const shares = pickArray(await getWalletJSON('/api/shares/holdings', 30000), 'holdings');
            const tokens = pickArray(await getWalletJSON('/api/shares/tokens', 30000), 'tokens');
            const invested = await getWalletJSON('/api/invest/portfolio', 30000);
            const positions = pickArray(invested, 'investments');
            const engineMap = gs.portfolio;
            let html = panelHead('Share holdings', 'entity shares, issued tokens and dividend positions');
            html += kpiGrid([
                kpi('Holding Lines', String(shares.length), 'share positions'),
                kpi('Issued Tokens', String(tokens.length), 'tokens issued by this wallet'),
                kpi('Investments', String(positions.length), 'invested entities'),
                kpi('Engine Positions', String(countOf(engineMap)), 'engine-reported share map'),
            ]);
            if (!w) return setPanel(token, html + connectNote());
            html += panelHead('Share positions', 'shares held per token');
            html += (shares.length ? objectArrayTable(shares, 6, 25) : emptyNote('No share holdings are reported.'));
            html += panelHead('Tokens issued', 'share tokens issued by this wallet');
            html += (tokens.length ? objectArrayTable(tokens, 8, 20) : emptyNote('This wallet has issued no share tokens.'));
            html += panelHead('Entity investment portfolio', 'positions and their claimable yield');
            html += (positions.length ? objectArrayTable(positions, 8, 20) : emptyNote('No entity investment positions are reported.'));
            if (countOf(engineMap)) {
                html += panelHead('Engine share map', 'entity shares carried by the running engine');
                html += rowsTable(engineMap);
            }
            return setPanel(token, html);
        }

        // ---- bonded registry (default) ----
        const assets = pickArray(await getWalletJSON('/api/assets', 30000), 'assets');
        const byType = {};
        assets.forEach(function (a) {
            const t = a.asset_type || a.AssetType || 'UNSPECIFIED';
            byType[t] = (byType[t] || 0) + 1;
        });
        const typeKeys = Object.keys(byType);
        let html = panelHead('Bonded registry', 'every §23 bonded asset owned or held by this wallet');
        html += kpiGrid([
            kpi('Bonded Assets', String(assets.length), 'owned or held'),
            kpi('Asset Types', String(typeKeys.length), 'classes in play'),
            kpi('Certified', String(countWhere(assets, function (a) { return !!(a.certified || a.Certified); })), 'legitimate provenance'),
            kpi('Black Market', String(countWhere(assets, function (a) { return !!(a.black_market_adopted || a.BlackMarketAdopted); })), 'off-ledger provenance'),
        ]);
        if (typeKeys.length) {
            html += panelHead('Holdings by class', 'bonded assets per asset type');
            html += chipRow(typeKeys.map(function (k) { return k + ' ×' + byType[k]; }));
        }
        if (!w) return setPanel(token, html + connectNote());
        if (!assets.length) return setPanel(token, html + emptyNote('No bonded assets are held by this wallet.'));
        html += panelHead('Registry detail', 'every field the bonded registry reports');
        html += objectArrayTable(assets, 12, 30);
        return setPanel(token, html);
    }

    // ========================================================================
    // CLUBS & ALLIANCES — organisations, memberships, alliance links, treasuries
    // ------------------------------------------------------------------------
    // `Club` is the engine's organisation record (common_types.go): owner, type,
    // territories, treasury (uint64 micro), members, leases, mojo, and the
    // alliance link + invitation state. Every figure is read as reported.
    // ========================================================================
    async function renderClubsAlliances(panel, sub, token) {
        const w = activeWallet();
        const clubs = pickArray(await getJSON('/api/clubs', 30000), 'clubs');
        const owned = clubs.filter(function (c) { return sameWallet(c.owner_wallet, w); });
        const memberOf = clubs.filter(function (c) { return !sameWallet(c.owner_wallet, w) && hasMember(c.members, w); });
        const territoryCount = owned.reduce(function (a, c) {
            return a + (Array.isArray(c.territories) ? c.territories.length : 0);
        }, 0);
        const treasuryMicro = sumMicro(owned, function (c) { return c.treasury_micro; });

        // ---- alliance links ----
        if (sub === 'alliance') {
            const coalition = [];
            owned.forEach(function (c) {
                if (!c.allied_club_id) return;
                const ally = clubs.filter(function (x) { return sameWallet(x.id, c.allied_club_id); })[0];
                coalition.push([c.name || c.id || '—',
                    ally ? (ally.name || ally.id) : shortWallet(c.allied_club_id),
                    String(ally && Array.isArray(ally.territories) ? ally.territories.length : 0),
                    String(ally && ally.mojo != null ? ally.mojo : '—')]);
            });
            const invites = owned.filter(function (c) { return !!c.alliance_invite_id; });
            let html = panelHead('Alliance links', 'club-to-club alliances this wallet is party to');
            html += kpiGrid([
                kpi('Alliances', String(coalition.length), 'active club links'),
                kpi('Coalition Territories', String(sumMicro(coalition, function (r) { return r[2]; })), 'held by allies'),
                kpi('Open Invitations', String(invites.length), 'received'),
            ]);
            if (!w) return setPanel(token, html + connectNote());
            if (!owned.length) return setPanel(token, html + emptyNote('This wallet owns no club, so it holds no alliance links.'));
            html += panelHead('Allied organisations', 'your club, its ally and the ally’s holdings');
            html += (coalition.length
                ? table(['Your Club', 'Ally', 'Ally Territories', 'Ally Mojo'], coalition)
                : emptyNote('None of your clubs has an active alliance.'));
            html += panelHead('Alliance invitations', 'incoming invitations and their expiry');
            html += (invites.length
                ? table(['Club', 'Invited By', 'Expires'], invites.map(function (c) {
                    return [c.name || c.id || '—', shortWallet(c.alliance_invite_id),
                        c.alliance_invite_expires_at ? fmtWhen(c.alliance_invite_expires_at) : '—'];
                }))
                : emptyNote('No alliance invitations are pending.'));
            html += '<p class="pp-hint">Alliance terms, boosts and limits are owned by the club engine; only reported state is shown here.</p>';

            // Engine-owned alliance graph (PlayerStats.Alliances / ActiveAllianceID).
            const a = await engineAssoc();
            html += panelHead('Engine alliance graph', 'allies the engine records against this wallet');
            if (!w) return setPanel(token, html + connectNote());
            if (!a) return setPanel(token, html + emptyNote('The engine association record could not be read.'));
            if (a.present === false) return setPanel(token, html + noEngineRecord());
            const allies = a.alliances || {};
            html += kpiGrid([
                kpi('Engine Alliances', String(countOf(allies)), 'allyClubID → relation'),
                kpi('Active Alliance', a.active_alliance_id ? shortWallet(a.active_alliance_id) : 'none set', 'currently in force'),
                kpi('Club Alliances', String(coalition.length), 'reported by the club engine'),
            ]);
            html += panelHead('Allies by relation', 'ally club → relation the engine records');
            html += (countOf(allies) ? rowsTable(allies) : emptyNote('The engine records no alliance entry for this wallet.'));
            return setPanel(token, html);
        }

        // ---- territory tiles ----
        if (sub === 'territory') {
            const a = await engineAssoc();
            let html = panelHead('Territory tiles', 'sector tiles the engine records for this wallet');
            html += kpiGrid([
                kpi('Sector Tiles', String(countOf(a && a.sector_tiles)), 'engine-owned map'),
                kpi('Club Territories', String(territoryCount), 'held by your clubs'),
                kpi('Clubs', String(owned.length), 'organisations you own'),
            ]);
            if (!w) return setPanel(token, html + connectNote());
            if (!a) return setPanel(token, html + emptyNote('The engine association record could not be read.'));
            if (a.present === false) return setPanel(token, html + noEngineRecord());
            const tiles = a.sector_tiles || {};
            html += panelHead('Sector tiles', 'tile → held');
            html += (countOf(tiles) ? rowsTable(tiles) : emptyNote('The engine records no sector tile for this wallet.'));
            const rollUp = owned.reduce(function (acc, c) {
                if (!Array.isArray(c.territories)) return acc;
                c.territories.forEach(function (t) { acc.push(c.name || c.id || '—'); });
                return acc;
            }, []);
            html += panelHead('Territories held by your clubs', 'each entry is one territory across your organisations');
            html += (rollUp.length ? chipRow(rollUp.map(function (n) { return String(n); })) : emptyNote('None of your clubs holds a territory.'));
            html += '<p class="pp-hint">Territory acquisition and management are actions owned by the World Dashboard (Governance ▸ Territory).</p>';
            return setPanel(token, html);
        }


        // ---- treasury & commission ----
        if (sub === 'treasury') {
            let html = panelHead('Treasury & commission', 'organisation funds held by this wallet’s clubs');
            html += kpiGrid([
                kpi('Combined Treasury', fmtVBV(treasuryMicro) + ' $VBV', 'uint64 micro-units'),
                kpi('Clubs', String(owned.length), 'under your ownership'),
                kpi('Territories', String(territoryCount), 'held by your clubs'),
                kpi('Members', String(sumMicro(owned, function (c) { return countOf(c.members); })), 'across your clubs'),
            ]);
            if (!w) return setPanel(token, html + connectNote());
            if (!owned.length) return setPanel(token, html + emptyNote('This wallet owns no clubs.'));
            html += panelHead('Per-club treasury', 'funds, rate, standing and obligations');
            html += table(['Club', 'Treasury', 'Commission', 'Mojo', 'Leases', 'Commission Events'], owned.map(function (c) {
                return [c.name || c.id || '—',
                    fmtVBV(c.treasury_micro) + ' $VBV',
                    (c.commission_rate == null ? '—' : c.commission_rate),
                    String(c.mojo == null ? '—' : c.mojo),
                    String(countOf(c.leases)),
                    String(countOf(c.commission_history))];
            }));
            html += panelHead('Full club payload', 'every field the club engine reports');
            html += objectArrayTable(owned, 14, 20);
            return setPanel(token, html);
        }

        // ---- clubs (default) ----
        let html = panelHead('Clubs', 'organisations owned by, or joined by, this wallet');
        html += kpiGrid([
            kpi('Clubs Owned', String(owned.length), 'ownership'),
            kpi('Memberships', String(memberOf.length), 'clubs you belong to'),
            kpi('Territories', String(territoryCount), 'held by your clubs'),
            kpi('Treasury', fmtVBV(treasuryMicro) + ' $VBV', 'combined funds'),
        ]);
        if (!w) return setPanel(token, html + connectNote());
        html += panelHead('Organisations you own', 'ownership, holdings and standing');
        html += (owned.length
            ? table(['Club', 'Type', 'Region', 'Territories', 'Members', 'Mojo'], owned.map(function (c) {
                return [c.name || c.id || '—', c.type || '—', c.region_name || '—',
                    String(Array.isArray(c.territories) ? c.territories.length : 0),
                    String(countOf(c.members)),
                    String(c.mojo == null ? '—' : c.mojo)];
            }))
            : emptyNote('This wallet owns no clubs.'));
        html += panelHead('Memberships', 'clubs you belong to without owning');
        html += (memberOf.length
            ? table(['Club', 'Type', 'Region', 'Owner'], memberOf.map(function (c) {
                return [c.name || c.id || '—', c.type || '—', c.region_name || '—', shortWallet(c.owner_wallet)];
            }))
            : emptyNote('This wallet is not a member of another club.'));
        const heldTerritories = owned.reduce(function (acc, c) {
            return acc.concat(Array.isArray(c.territories) ? c.territories : []);
        }, []);
        if (heldTerritories.length) {
            html += panelHead('Territory roll-up', 'territories held across your organisations');
            html += chipRow(heldTerritories.map(function (tid) { return String(tid); }));
        }
        return setPanel(token, html);
    }

    // ========================================================================
    // RIVALRY — active rivals, invitations and faction standing (§25.10 / P12)
    // ========================================================================
    async function renderRivalry(panel, sub, token) {
        const w = activeWallet();

        // ---- factions ----
        if (sub === 'factions') {
            const factions = pickArray(await getJSON('/api/rivalry/factions', 45000), 'factions');
            const mine = factions.filter(function (f) { return hasMember(f.members, w); });
            let html = panelHead('Factions', 'civilization factions and this wallet’s standing');
            html += kpiGrid([
                kpi('Factions', String(factions.length), 'civilization-wide'),
                kpi('Joined', String(mine.length), 'memberships held'),
                kpi('Total Power', String(sumMicro(factions, function (f) { return f.power_score; })), 'summed faction power'),
            ]);
            if (!w) return setPanel(token, html + connectNote());
            if (!factions.length) return setPanel(token, html + emptyNote('No factions are reported by the engine.'));
            html += panelHead('Membership', 'factions this wallet has joined');
            html += (mine.length
                ? chipRow(mine.map(function (f) { return (f.name || f.faction_id) + ' · ' + (f.region || '—'); }))
                : emptyNote('This wallet has not joined a faction.'));
            html += panelHead('All factions', 'members, reputation and power as the engine reports them');
            html += table(['Faction', 'Kind', 'Region', 'Members', 'Reputation', 'Power'], factions.map(function (f) {
                return [f.name || f.faction_id || '—', f.kind || '—', f.region || '—',
                    String(countOf(f.members)),
                    String(f.reputation == null ? '—' : f.reputation),
                    String(f.power_score == null ? '—' : f.power_score)];
            }));
            return setPanel(token, html);
        }

        const st = await getWalletJSON('/api/rivalry/state', 20000);
        const rivals = pickArray(st, 'active_rivals');
        const invites = pickArray(st, 'pending_invitations');
        // The engine only creates a record for a wallet that has rivalry state.
        // `success:false` is the endpoint's explicit "no record" answer and is
        // reported as such rather than rendered as "zero rivals".
        const noRecord = !!(st && st.success === false);

        // ---- invitations ----
        if (sub === 'invitations') {
            let html = panelHead('Rivalry invitations', 'standing challenges issued TO this wallet');
            html += kpiGrid([kpi('Pending', String(invites.length), 'awaiting a response')]);
            if (!w) return setPanel(token, html + connectNote());
            if (noRecord) return setPanel(token, html + emptyNote('No rivalry record exists for this wallet yet.'));
            html += (invites.length
                ? table(['From', 'Level', 'Invitation'], invites.map(function (i) {
                    return [shortWallet(i.from_wallet), String(i.level == null ? '—' : i.level), shortWallet(i.id)];
                }))
                : emptyNote('No rivalry invitations are pending.'));
            return setPanel(token, html);
        }

        // ---- active rivals (default) ----
        const regionRivalries = pickArray(await getJSON('/api/rivalry/list', 45000), 'data');
        let html = panelHead('Active rivals', 'head-to-head relationships tracked by the rivalry engine');
        html += kpiGrid([
            kpi('Active Rivals', String(rivals.length), 'declared opponents'),
            kpi('Pending Invitations', String(invites.length), 'awaiting a response'),
            kpi('Region Rivalries', String(regionRivalries.length), 'territory / region level'),
        ]);
        if (!w) return setPanel(token, html + connectNote());
        if (noRecord) {
            html += emptyNote('No rivalry record exists for this wallet yet.');
        } else {
            html += panelHead('Opponents', 'wallets flagged as active rivals');
            html += (rivals.length
                ? chipRow(rivals.map(function (r) { return shortWallet(r && (r.wallet || r.Wallet) ? (r.wallet || r.Wallet) : r); }))
                : emptyNote('This wallet has no active rivals.'));
        }
        html += panelHead('Region & territory rivalries', 'civilization-level rivalries (§25.10)');
        html += (regionRivalries.length
            ? table(['Level', 'Side A', 'Side B', 'Contested Class', 'A', 'B', 'Winner'], regionRivalries.map(function (r) {
                return [r.level || '—', r.side_a || '—', r.side_b || '—', r.asset_class || '—',
                    String(r.score_a == null ? '—' : r.score_a),
                    String(r.score_b == null ? '—' : r.score_b),
                    r.winner || 'open'];
            }))
            : emptyNote('No region rivalries are declared.'));
        return setPanel(token, html);
    }

    // ========================================================================
    // COMPANIONS — pets (§26.4), vehicles (§25.6) and their lineage
    // ------------------------------------------------------------------------
    // Every figure comes from the pet/vehicle registry or the §30 stat vector.
    // Nothing is derived that the engine does not report.
    // ========================================================================
    async function renderCompanions(panel, sub, token) {
        const w = activeWallet();

        // ---- vehicles ----
        if (sub === 'vehicles') {
            const vehPayload = await getWalletJSON('/api/vehicles', 30000);
            const vehicles = pickArray(vehPayload, 'data');
            // §25.6.1: the part table is server-authored — read it, never re-declare it.
            const vpartsRaw = vehPayload ? vehPayload.parts : null;
            const partMaxRaw = vehPayload ? vehPayload.part_max_level : null;
            const baseCostRaw = vehPayload ? vehPayload.base_cost_micro : null;
            const kinds = {};
            vehicles.forEach(function (v) {
                const k = v.kind || v.Kind || 'UNSPECIFIED';
                kinds[k] = (kinds[k] || 0) + 1;
            });
            const kindKeys = Object.keys(kinds);
            let html = panelHead('Vehicles', 'level-gated mounts bonded to this wallet (§25.6)');
            html += kpiGrid([
                kpi('Vehicles', w ? String(vehicles.length) : '—', 'owned'),
                kpi('Types', String(kindKeys.length), 'GROUND / FLYER / DIGGER'),
                kpi('Highest Level Gate', String(maxOf(vehicles, function (v) { return v.min_level; })), 'required level (§24.5)'),
            ]);
            // §26.4.3 / §25.6.2: the REAL percentage these bonded assets add to every deck card
            // in a match (the server snapshots it into MatchState; this is the same number).
            const bondedV = await getWalletJSON('/api/owner/combined-stats', 20000);
            const deckPctV = (bondedV && bondedV.bonded_deck_boost_pct != null) ? Number(bondedV.bonded_deck_boost_pct) : null;
            html += kpiGrid([
                kpi('Deck Bonus', deckPctV == null ? '—' : ('+' + deckPctV + '%'), 'added to every deck card (§26.4.3)'),
                kpi('Purchase Classes', (vehPayload && Array.isArray(vehPayload.spawn_fees)) ? String(vehPayload.spawn_fees.length) : '—', 'server-priced classes (§25.6.1)'),
            ]);
            if (kindKeys.length) {
                html += panelHead('Fleet by type', 'count per vehicle class');
                html += chipRow(kindKeys.map(function (k) { return k + ' ×' + kinds[k]; }));
            }
            if (!w) return setPanel(token, html + connectNote());
            if (!vehicles.length) return setPanel(token, html + emptyNote('No vehicles are bonded to this wallet.'));

            // ── §25.6.1 upgrade ladder (read-only analytics) ──────────────────
            // Vehicles progress by PARTS + INVESTMENT — the deliberate counterpart to
            // §26.4 pet breeding. Analytics only: the *actions* live in the World
            // Dashboard (Assets ▸ Life Assets), never here (app-entry-mandate §3).
            const vparts = Array.isArray(vpartsRaw) ? vpartsRaw : [];
            const partMax = Number(partMaxRaw) || 0;
            const baseCost = Number(baseCostRaw) || 0;
            const buildOf = function (v) {
                if (v.vehicle_level != null) return Number(v.vehicle_level) || 1;
                const ups = v.upgrades || {};
                return 1 + Object.keys(ups).reduce(function (a, k) { return a + (Number(ups[k]) || 0); }, 0);
            };
            const partOf = function (v, part) {
                const ups = v.upgrades || {};
                return Number(ups[part]) || 0;
            };
            const statSumOf = function (v) {
                return entityStatSum(v.stats || v.Stats);
            };
            const totalPartLevels = vehicles.reduce(function (a, v) {
                const ups = v.upgrades || {};
                return a + Object.keys(ups).reduce(function (b, k) { return b + (Number(ups[k]) || 0); }, 0);
            }, 0);
            const certified = countWhere(vehicles, function (v) { return !!(v.certified || v.Certified); });
            const blackMarket = countWhere(vehicles, function (v) { return !!(v.black_market_adopted || v.BlackMarketAdopted); });
            const deployed = countWhere(vehicles, function (v) { return !!(v.region || v.Region); });
            const ready = countWhere(vehicles, function (v) { return v.mature === true || v.Mature === true; });
            const investedMicro = vehicles.reduce(function (a, v) {
                const ups = v.upgrades || {};
                return a + Object.keys(ups).reduce(function (b, k) {
                    // Deterministic replay of the server's linear cost curve:
                    // Σ (base × (level+1)) for level = 0..n-1.
                    const n = Number(ups[k]) || 0;
                    return b + (baseCost * ((n * (n + 1)) / 2));
                }, 0);
            }, 0);
            const maxBuild = vehicles.reduce(function (a, v) { return Math.max(a, buildOf(v)); }, 0);
            const bestPower = vehicles.reduce(function (a, v) {
                return Math.max(a, Math.min(600, buildOf(v) + Math.floor(statSumOf(v) / 50)));
            }, 0);

            html += panelHead('Upgrade ladder', 'parts fitted — the vehicle counterpart to pet breeding');
            html += kpiGrid([
                kpi('Parts Fitted', String(totalPartLevels), 'levels across ' + (vparts.length || 0) + ' parts'),
                kpi('Highest Build', String(maxBuild), 'build level (1 + Σ parts)'),
                kpi('Best Power', String(bestPower), '§30 overlay level, capped 600'),
                kpi('Invested', fmtVBV(investedMicro) + ' $VBV', 'replayed from the cost curve'),
                kpi('Certified', String(certified), 'legitimate builds'),
                kpi('Black Market', String(blackMarket), 'ineligible for progress (§27.7.3)'),
                kpi('Broken In', String(ready), 'past the break-in window'),
                kpi('Deployed', String(deployed), 'parked in a region'),
            ]);
            if (vparts.length) {
                html += panelHead('Part table', 'server-authored calibration (the client never re-declares it)');
                html += table(['Part', 'Stat Axis', 'Max Level', 'Cost (level 1)'], vparts.map(function (p) {
                    return [p.part || '—', p.axis || '—', String(partMax || '—'),
                        fmtVBV(baseCost) + ' $VBV'];
                }));
                html += panelHead('Per-vehicle build', 'each vehicle against every part');
                html += table(['Vehicle', 'Build', 'PWR', 'Stat Pts'].concat(vparts.map(function (p) { return p.part; })), vehicles.map(function (v) {
                    return [v.name || shortWallet(v.vehicle_id), String(buildOf(v)),
                        String(Math.min(600, buildOf(v) + Math.floor(statSumOf(v) / 50))),
                        String(statSumOf(v))].concat(vparts.map(function (p) {
                            return partOf(v, p.part) + '/' + (partMax || '—');
                        }));
                }));
                html += panelHead('Parts fitted per vehicle', 'part → level (read-only)');
                vehicles.forEach(function (v) {
                    const ups = v.upgrades || {};
                    const keys = Object.keys(ups);
                    if (!keys.length) return;
                    html += kpiGrid(keys.map(function (k) {
                        return kpi(k, String(ups[k]), v.name || shortWallet(v.vehicle_id));
                    }));
                });
                html += '<p class="pp-hint">Fitting parts is an action — it lives in the World Dashboard ▸ Assets ▸ Life Assets (Garage).</p>';
            } else {
                html += emptyNote('The server reports no vehicle part table yet, so no upgrade ladder can be shown.');
            }

            html += panelHead('Owned vehicles', 'every field the registry reports');
            html += objectArrayTable(vehicles, 6, 30);
            return setPanel(token, html);
        }

        const pets = pickArray(await getWalletJSON('/api/pets', 30000), 'data');
        const matured = countWhere(pets, petMature);
        const certified = countWhere(pets, function (p) { return !!(p.certified || p.Certified); });
        const blackMarket = countWhere(pets, function (p) { return !!(p.black_market_adopted || p.BlackMarketAdopted); });
        const deployed = countWhere(pets, function (p) { return !!(p.region || p.Region); });
        const bredIn = countWhere(pets, function (p) { return !!(p.sire_id || p.SireID || p.dam_id || p.DamID); });

        // ---- lineage & maturity ----
        if (sub === 'lineage') {
            const pending = pets.map(maturityRemainingMs).filter(function (n) { return n > 0; });
            let html = panelHead('Lineage & maturity', 'breeding stock and maturity windows (§26.4.1)');
            html += kpiGrid([
                kpi('Founded Bloodlines', String(pets.length - bredIn), 'no recorded parents'),
                kpi('Bred In', String(bredIn), 'offspring of recorded parents'),
                kpi('Breeding Eligible', String(countWhere(pets, function (p) {
                    return petMature(p) && !!(p.certified || p.Certified) && !(p.black_market_adopted || p.BlackMarketAdopted);
                })), 'mature + certified'),
                kpi('Next Maturity', pending.length ? fmtDuration(Math.min.apply(null, pending)) : 'none pending', 'soonest companion'),
            ]);
            if (!w) return setPanel(token, html + connectNote());
            if (!pets.length) return setPanel(token, html + emptyNote('No pets are bonded to this wallet.'));
            html += panelHead('Recorded lineage', 'parents held against each owned pet');
            html += table(['Pet', 'Sire', 'Dam', 'Maturity'], pets.map(function (p) {
                const sire = p.sire_id || p.SireID;
                const dam = p.dam_id || p.DamID;
                return [p.name || p.Name || shortWallet(p.pet_id || p.PetID),
                    sire ? shortWallet(sire) : '—',
                    dam ? shortWallet(dam) : '—',
                    fmtDuration(maturityRemainingMs(p))];
            }));
            const battles = pickArray(await getJSON('/api/pet-battle/list', 30000), 'battles').filter(function (b) {
                return sameWallet(b.challenger_owner, w) || sameWallet(b.defender_owner, w);
            });
            html += panelHead('Arena record', 'pet battles involving this wallet (§30)');
            html += (battles.length
                ? table(['Battle', 'Challenger', 'Defender', 'Status', 'Winner', 'Reward'], battles.map(function (b) {
                    return [shortWallet(b.battle_id), shortWallet(b.challenger), shortWallet(b.defender),
                        b.status || '—', b.winner || '—', fmtVBV(b.reward_micro || 0) + ' $VBV'];
                }))
                : emptyNote('No arena battles recorded for these companions.'));
            return setPanel(token, html);
        }

        // ---- pets (default) ----
        // §26.4.3 / §25.6.2: the bonded assets' real contribution to the owner's deck cards.
        const bondedP = await getWalletJSON('/api/owner/combined-stats', 20000);
        const deckPctP = (bondedP && bondedP.bonded_deck_boost_pct != null) ? Number(bondedP.bonded_deck_boost_pct) : null;
        const groomLevels = pets.reduce(function (a, p) {
            const g = p.grooming || p.Grooming || {};
            return a + Object.keys(g).reduce(function (x, k) { return x + (Number(g[k]) || 0); }, 0);
        }, 0);
        let html = panelHead('Pets', 'breedable companions bonded to this wallet (§26.4)');
        html += kpiGrid([
            kpi('Pets', String(pets.length), 'owned'),
            kpi('Mature', String(matured), 'breeding / competing age'),
            kpi('Certified', String(certified), 'legitimate lineage (§27.7.3)'),
            kpi('Black Market', String(blackMarket), 'breeding-ineligible (§26.4.1)'),
            kpi('Deployed', String(deployed), 'assigned to a region'),
            kpi('Bred In', String(bredIn), 'recorded parents'),
            kpi('Groomed Levels', String(groomLevels), 'purchased progression steps (§26.4.2)'),
            kpi('Deck Bonus', deckPctP == null ? '—' : ('+' + deckPctP + '%'), 'added to every deck card (§26.4.3)'),
        ]);
        if (!w) return setPanel(token, html + connectNote());
        if (!pets.length) return setPanel(token, html + emptyNote('No pets are bonded to this wallet.'));
        html += panelHead('Companion power', 'level, stat vector and dominant axis per pet');
        html += table(['Pet', 'Level', 'Stat Sum', 'Dominant', 'Certified', 'Region'], pets.map(function (p) {
            return [p.name || p.Name || shortWallet(p.pet_id || p.PetID),
                String(p.pet_level == null ? '—' : p.pet_level),
                String(entityStatSum(p.stats)),
                dominantEntityStat(p.stats),
                ((p.certified || p.Certified) ? 'Yes' : 'No'),
                p.region || p.Region || '—'];
        }));
        html += panelHead('Grooming ladder', 'purchased levels per axis (§26.4.2 — the mirror of §25.6.1 vehicle parts)');
        html += table(['Companion'].concat(ENTITY_STAT_KEYS.map(function (k) {
            return k.charAt(0).toUpperCase() + k.slice(1);
        })), pets.map(function (p) {
            const g = p.grooming || p.Grooming || {};
            return [p.name || p.Name || shortWallet(p.pet_id || p.PetID)].concat(
                ENTITY_STAT_KEYS.map(function (k) { return String(Number(g[k.toUpperCase()]) || 0); }));
        }));
        html += panelHead('Owner relationship', 'opinion axes the event engine maintains (§30)');
        html += table(['Pet', 'Owner Opinion', 'Cross Opinion'], pets.map(function (p) {
            return [p.name || p.Name || '—',
                String(p.owner_opinion == null ? '—' : p.owner_opinion),
                String(p.owner_cross_opinion == null ? '—' : p.owner_cross_opinion)];
        }));
        html += panelHead('Stat vectors', 'per-axis power for every companion');
        pets.forEach(function (p) {
            html += '<div class="pf-quote">' + esc(p.name || p.Name || 'companion') + '</div>' + statBars(p.stats);
        });
        html += panelHead('Full registry payload', 'every field reported for each pet');
        html += objectArrayTable(pets, 12, 30);
        return setPanel(token, html);
    }


    // PUBLIC API
    // ------------------------------------------------------------------------
    // One feature, one entry point: the World Dashboard owns navigation to this
    // surface via WD_ROUTES.portfolio. `openPlayerProfile` is retained ONLY as a
    // deprecated alias so already-cached callers cannot break; it is not a
    // second UI and creates no duplicate navigation path.
    // ========================================================================
    window.openPortfolio = openPortfolio;
    window.closePortfolio = closePortfolio;
    window.ppOpenCategory = openCategory;
    window.ppOpenSub = openSub;
    window.ppSwitchRegion = switchRegion;

    // Deprecated aliases (pre-rename callers).
    window.openPlayerProfile = openPortfolio;
    window.closePlayerProfile = closePortfolio;

    // Close on Escape while the Portfolio is open.
    document.addEventListener('keydown', function (e) {
        if (e.key === 'Escape' && hubEl && hubEl.style.display !== 'none') closePortfolio();
    });

    // WS consumer (flow-doc §12.3): surface a career-tier demotion as a toast, and
    // drop cached analytics so the next screen read reflects the new state.
    window.onCareerTierDemoted = function (payload) {
        const role = (payload && payload.role) ? payload.role : '';
        invalidate();
        if (window.showToast) {
            window.showToast('⚠️ Career tier demoted' + (role ? ' (' + role + ')' : ''), 'warning');
        }
    };
})();
