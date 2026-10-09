// ============================================================================
// world_dashboard.js — World Dashboard (the SINGLE navigation surface)
// ----------------------------------------------------------------------------
// TWO-TIER NAVIGATION (binding, per operator):
//
//   TIER 1 — CATEGORIES.  The dashboard opens showing ONLY the category
//            buttons. No leaf/feature buttons are visible here.
//   TIER 2 — FEATURE MENU.  Choosing a category reveals a panel menu listing
//            that category's feature buttons. Choosing a feature then opens
//            that feature's UI (either SPA-routed out to its standalone
//            overlay, or embedded in a dashboard panel).
//
// Categories are grouped by DOMAIN, not by surface. A category absorbs every
// interaction that belongs to it — e.g. "Careers & Factions" owns the career
// tree AND the career-shaped criminality + justice functions, because those
// ARE career interactivity (Bounty Hunter, Warden, Tax Auditor, Underworld
// Boss, …). One feature, one owner, one entry point.
//
// STARBOUND QUICK ACCESS: the ☆ on a CATEGORY (never on a leaf feature) is
// what surfaces in the constellation hub / main-menu favorites. `UserPreferences`
// stores the category id as `wdTab`, so `openWorldDashboardToTab(<catId>)`
// resolves back to that category's feature menu.
// ============================================================================

(function () {
    'use strict';
    var API_BASE = '/api';
    let overlayEl = null, statusEl = null;

    // --- Category Definitions (TIER 1) -------------------------------------
    // Each category owns a DOMAIN and lists its FEATURES (Tier 2). Routing is NOT
    // duplicated here — WD_ROUTES below stays the single owner of routing, and a
    // feature renders a ↗ badge when it has a standalone overlay to open.
    // `tabs` is derived after the literal so existing consumers keep working.
    const WD_CATEGORIES = [
        {
            id: 'player', icon: '🎮', name: 'Player Hub', color: '#00bcd4',
            features: [
                { tab: 'portfolio', label: 'Portfolio (Analytics)' },
                { tab: 'identity', label: 'Identity Editor' },
                { tab: 'stats', label: 'Player Stats' },
                { tab: 'achievements', label: 'Achievements' },
                { tab: 'deck', label: 'Deck Manager' },
                { tab: 'card_titles', label: 'Card Titles' },
                { tab: 'card_progression', label: 'Card Progression' },
                { tab: 'mood', label: 'Character Mood' },
                { tab: 'wagers', label: 'Post-Match Wagers' },
                // §27 THEME ENGINE. `theme_dashboard.js` was composed and `openThemeDashboard` was
                // called from NOWHERE, so the only surface that shows the player's computed theme
                // vector — the value `theme_engine.js` now feeds into `--theme-accent*` — had no way
                // in. It lives here because the vector IS the player's own visual identity, which is
                // this category's domain.
                { tab: 'theme', label: 'Theme Engine — element · mood · projections' },
            ],
        },
        {
            // Careers absorb EVERY career-shaped interaction: the career tree, the
            // criminality functions (courthouse + bounty board) and the justice
            // functions (hegemony, truth serum, shield) — those are the career
            // pathways' active surfaces, not separate domains.
            id: 'careers', icon: '⚔️', name: 'Careers & Factions', color: '#ff9800',
            features: [
                { tab: 'career', label: 'Career Pathways' },
                { tab: 'criminality', label: 'Criminality — Courthouse, Bounties & Intel' },
                { tab: 'justice', label: 'Justice Hegemony' },
                { tab: 'contracts', label: 'Underworld Contracts' },
                { tab: 'counterfeit', label: 'Counterfeit Scanner' },
                // The faction shops (/api/faction/shop/{JUSTICE|UNDERWORLD} + /buy) had NO UI
                // anywhere: buying is gated on the caller's career ROLE, so it is career
                // interactivity and belongs in this category.
                { tab: 'faction', label: 'Faction Quartermaster — Justice & Underworld shops' },
            ],
        },
        {
            id: 'assets', icon: '🎒', name: 'Assets', color: '#8bc34a',
            features: [
                { tab: 'items', label: 'Inventory & Equipment' },
                { tab: 'equipment', label: 'Equipment System' },
                { tab: 'pets', label: 'Companion Kennel — Breed & Groom' },
                { tab: 'vehicles', label: 'Garage — Build & Upgrade' },
                // AI Citizens + Children Bots are OWNED ENTITIES THE PLAYER DEVELOPS, which is
                // this category's domain (the Kennel breeds/grooms, the Garage builds/upgrades).
                // They previously had NO home here at all — `openAICitizens` was called from
                // NOWHERE and Children Bots were reachable only via the constellation hub — so
                // two fully built surfaces were invisible to the single navigation surface.
                { tab: 'citizens', label: 'AI Citizens — Spawn & Develop' },
                { tab: 'children', label: 'Children Bots — Learning Pathways' },
                { tab: 'branding', label: 'Bonded Branding Studio' },
                { tab: 'pet_arena', label: 'Companion Arena (3D world)' },
                { tab: 'vehicle_arena', label: 'Vehicle Arena (3D world)' },
                { tab: 'clubs', label: 'Club Foundry' },
                { tab: 'orphan', label: 'Orphan Cleaner' },
            ],
        },
        {
            id: 'economy', icon: '💰', name: 'Economy & Trade', color: '#ffd700',
            features: [
                { tab: 'markets', label: 'Markets & AMM' },
                { tab: 'dividends', label: 'Dividend Yield' },
                { tab: 'loans', label: 'Loan Terms' },
                { tab: 'blackmarket', label: 'Black Market' },
                // The multi-chain Bridge Router was composed and `openBridgeRouter` was called from
                // NOWHERE. Moving the player's own assets between ecosystems is an economic act, so
                // Economy & Trade is its domain. This also gives /api/bridge/onboard — which had no
                // owner anywhere in the client — a reachable surface.
                { tab: 'bridge', label: 'Bridge Router — move assets between ecosystems' },
                { tab: 'ads', label: 'Advertising' },
            ],
        },
        {
            id: 'governance', icon: '🏛️', name: 'Governance', color: '#9c27b0',
            features: [
                { tab: 'governance', label: 'Governance Chambers' },
                { tab: 'governor', label: 'Governor Office' },
                { tab: 'territory', label: 'Territory Map' },
                { tab: 'regions', label: 'Regions' },
                { tab: 'compliance', label: 'Compliance' },
                { tab: 'leaderboard', label: 'Regional Leaderboard' },
            ],
        },
        {
            id: 'play', icon: '🃏', name: 'Play & Board', color: '#e91e63',
            features: [
                { tab: 'game', label: 'Game Board' },
                { tab: 'create_match', label: 'Create Match' },
                { tab: 'match', label: 'Match Arena' },
                { tab: 'multiplayer', label: 'Multiplayer' },
                { tab: 'game_modes', label: 'Game Modes' },
                { tab: 'campaign', label: 'Campaign' },
                { tab: 'tutorial', label: 'Tutorial' },
                { tab: 'locations', label: 'Locations' },
                { tab: 'npc_taunts', label: 'NPC Taunts' },
                { tab: 'tea_house', label: 'Tea House' },
                { tab: 'zen_garden', label: 'Zen Garden' },
            ],
        },
        {
            id: 'world', icon: '🌍', name: 'World & Events', color: '#03a9f4',
            features: [
                { tab: 'season', label: 'Season' },
                { tab: 'events', label: 'World Events' },
                { tab: 'tournament', label: 'Tournament Brackets' },
                { tab: 'replay', label: 'Replay Viewer' },
                { tab: 'treasure', label: 'Treasure Map' },
                { tab: 'worldcontent', label: 'World Content' },
                { tab: 'industrial', label: 'Industrial Loop' },
            ],
        },
        {
            id: 'faith', icon: '⛪', name: 'Faith & Church', color: '#ffc107',
            features: [
                { tab: 'faith', label: 'Faith System' },
                // Religion Governance (the 24-cap faucet-owned religion chain + buyouts) was composed
                // and `openReligionGovernance` was called from NOWHERE, so the surface that owns six
                // /api/faith/religion/* routes had no way in.
                { tab: 'religion', label: 'Religion Governance — chain, buyouts, rivals' },
                { tab: 'church', label: 'Church Storefront' },
            ],
        },
        {
            id: 'competition', icon: '🏆', name: 'Competition', color: '#ff5722',
            features: [
                { tab: 'rivalry', label: 'Rivalry Matrix' },
                { tab: 'rewards', label: 'Rewards Center' },
            ],
        },
        {
            id: 'creator', icon: '🎨', name: 'Creator Economy', color: '#673ab7',
            features: [
                { tab: 'creator', label: 'Creator Store' },
                { tab: 'launches', label: 'Launchpad' },
            ],
        },
        {
            id: 'system', icon: '🛠️', name: 'System & Ops', color: '#607d8b',
            features: [
                { tab: 'infrastructure', label: 'Infrastructure' },
                { tab: 'gamingos', label: 'Gaming OS' },
                { tab: 'devhub', label: 'Dev/Game Hub (framework catalogue)' },
                { tab: 'localmodel', label: 'Local Model' },
                { tab: 'maintenance', label: 'Maintenance Mode' },
                { tab: 'systemmsg', label: 'System Messages' },
                { tab: 'report', label: 'Report History' },
                { tab: 'admin', label: 'Admin Console (signature-gated)' },
                // FIVE OPERATIONS CONSOLES. Each was COMPOSED by app.js, published every one of its
                // handler names, and rendered NOTHING: the root container it looked up had no markup
                // in the shell, so `init()` returned on the null-check and the boot-time guard never
                // fired. They are operator surfaces that span domains (which is why they are
                // consoles, not leaves), so System & Ops — the operations domain that already owns
                // Infrastructure, Gaming OS and the Admin Console — is where they belong.
                { tab: 'community', label: 'Community Console — Church · Justice · Governance · Identity' },
                { tab: 'exconsole', label: 'Extended Console — Assets · Orphans · OS · Launches · Rewards' },
                { tab: 'secconsole', label: 'Security Console — Security · Simulation · Black Market' },
                { tab: 'sysconsole', label: 'System Console — Achievements · Rivalry · Season · Sync' },
                { tab: 'utconsole', label: 'Utilities Console — Ads · Bridge · Cards · Industrial · Stats' },
            ],
        },
    ];

    // Derived: `tabs` (back-compat for anything iterating cat.tabs) + O(1) lookups.
    WD_CATEGORIES.forEach(function (c) { c.tabs = c.features.map(function (f) { return f.tab; }); });

    function categoryById(id) {
        for (const c of WD_CATEGORIES) { if (c.id === id) return c; }
        return null;
    }
    function categoryForTab(tab) {
        for (const c of WD_CATEGORIES) { if (c.tabs.indexOf(tab) >= 0) return c; }
        return null;
    }

    let activeCategory = null;

    // --- SPA Routing ---------------------------------------------------------
    // Tabs that have a DEDICATED, full-screen standalone feature UI. Clicking one
    // closes the World Dashboard and opens that feature's real overlay (single-page
    // app navigation) instead of embedding the feature as a sub-panel inside the
    // dashboard. Tabs absent from this map have no separate location and remain
    // embedded in the dashboard (their only UI). Every entry is typeof-guarded at
    // click time, so a missing/renamed opener safely falls back to embedding.
    // Tab ids renamed by an architectural change. A legacy quick-access star
    // stored under an old id is re-pointed here so it never dead-ends.
    const RENAMED_TABS = { profile: 'portfolio' };

    const WD_ROUTES = {
        // PORTFOLIO is a DATA-ONLY analytics view and is reached FROM here — the World
        // Dashboard is the single navigation surface (Single-Navigation Mandate).
        portfolio: 'openPortfolio',
        // §27 theme visualizer: composed by app.js, opened by NOTHING. Its Theme Vector tab is the
        // only surface that shows the computed vector the palette layer now consumes.
        theme: 'openThemeDashboard',
        identity: 'openPersistentIdentity',
        career: 'openCareers',
        achievements: 'openAchievements',
        stats: 'openStatOverlay',
        leaderboard: 'openLeaderboardRegion',
        rivalry: 'openRivalryViewer',
        match: 'openMatchArena',
        creator: 'openCreatorStore',
        events: 'openWorldEvents',
        faith: 'openFaithSystem',
        religion: 'openReligionGovernance',
        church: 'openFaithChurch',
        bridge: 'openBridgeRouter',
        governance: 'openGovernancePanel',
        orphan: 'openOrphanCleaner',
        // The admin surface is ONE reachable panel, opened by ITS owner (admin_console.js). The four
        // competing "admin dashboard" modules (admin_dashboard, operations_dashboard, final_dashboard,
        // admin_panel) were never mounted, never called, and offered password-based access; they are
        // gone. Access is a wallet signature against ADMIN_WALLETS, so this route only REACHES the
        // panel — it cannot grant anything.
        admin: 'openAdminConsole',
        // NOTE: `pets` deliberately has NO route. Companion progression (purchase, breeding,
        // grooming) is the Kennel, which EMBEDS here via initPetBreeder — it used to route out
        // to the (fabricated) battle arena, which hid the only progression UI behind an arena.
        vehicles: 'openLifeAssets',
        // §23.5 Bonded Branding Studio: the ecosphere-wide theming surface. Cards are
        // excluded by the server (it serves the blocked kind list), never by this client.
        branding: 'openBondedBranding',
        // AI Citizens and Children Bots own their overlays (each module builds and mounts its own
        // root, so there is nothing to embed), therefore they ROUTE OUT. `citizens` had no route
        // and no category — `window.openAICitizens` was published but never called.
        citizens: 'openAICitizens',
        children: 'openChildrenBots',
        // Both arenas are 3D-world destinations: they own their own placeholder surfaces.
        pet_arena: 'openPetBattleArena',
        vehicle_arena: 'openVehicleArena',
        contracts: 'openUnderworld',
        // NOTE: `counterfeit` deliberately has NO route here. It previously duplicated
        // `contracts` (both opened `openUnderworld` — the SAME UI reachable twice).
        // It now EMBEDS its own unique Counterfeit Scanner panel (initCounterfeitScanner).
        criminality: 'openCriminality',
        industrial: 'openIndustrialLoop',
        gamingos: 'openGamingOS',
        // The Dev/Game Hub was composed by app.js but opened by NOTHING: its
        // `window.openDevGameHub` had no caller anywhere. It is now a real leaf under
        // System & Ops, so the surface is reachable through the ONE navigation surface
        // (and it states why nothing in its catalogue can be bought).
        devhub: 'openDevGameHub',
        launches: 'openLaunchpad',
        // The five operations consoles. Each owns a full-screen overlay and is opened by ITS owner
        // module, so they route out rather than embed. Without these entries the WD fell through to
        // `initXxx()` — names those modules never publish — and the consoles stayed unreachable even
        // after their missing root was self-mounted.
        community: 'openCommunityDashboard',
        exconsole: 'openExtendedDashboard',
        secconsole: 'openSecurityDashboard',
        sysconsole: 'openSystemDashboard',
        utconsole: 'openUtilitiesDashboard',
        ads: 'openAdvertising'
    };

    const TAB_LABELS = {
        portfolio: 'Portfolio',
        theme: 'Theme',
        identity: 'Identity', career: 'Career', achievements: 'Achievements', stats: 'Stats',
        items: 'Items', pets: 'Pets', vehicles: 'Vehicles', clubs: 'Clubs',
        branding: 'Branding',
        citizens: 'AI Citizens', children: 'Children Bots',
        pet_arena: 'Pet Arena', vehicle_arena: 'Vehicle Arena',
        markets: 'Markets', dividends: 'Dividends', loans: 'Loans', blackmarket: 'Black Mkt',
        justice: 'Justice', counterfeit: 'Counterfeit', contracts: 'Contracts',
        criminality: 'Criminality',
        faction: 'Faction Shop',
        admin: 'Admin Console',
        community: 'Community Console',
        exconsole: 'Extended Console',
        secconsole: 'Security Console',
        sysconsole: 'System Console',
        utconsole: 'Utilities Console',
        governance: 'Governance', governor: 'Governor', territory: 'Territory', regions: 'Regions',
        compliance: 'Compliance', leaderboard: 'Leaderbd',
        season: 'Season', events: 'Events', tournament: 'Tournament', replay: 'Replay',
        treasure: 'Treasure', worldcontent: 'World', industrial: 'Industrial',
        maintenance: 'Maint', systemmsg: 'SysMsg',
        faith: 'Faith', church: 'Church',
        religion: 'Religion', bridge: 'Bridge',
        orphan: 'Orphans',
        creator: 'Creator', launches: 'Launches', ads: 'Ads',
        gamingos: 'Gaming OS', localmodel: 'Local LLM',
        devhub: 'Dev/Game Hub',
        rivalry: 'Rivalry', match: 'Match', rewards: 'Rewards',
        report: 'Report',
        create_match: 'Create', deck: 'Deck', campaign: 'Campaign',
        equipment: 'Equip', card_progression: 'Progress', game: 'Game',
        locations: 'Locations', tutorial: 'Tutorial', card_titles: 'Titles', wagers: 'Wagers', mood: 'Mood', tea_house: 'Tea', zen_garden: 'Zen', npc_taunts: 'NPCs', game_modes: 'Modes', multiplayer: 'PvP'
    };

    function tabLabel(tab) { return TAB_LABELS[tab] || tab.charAt(0).toUpperCase() + tab.slice(1); }

    // TIER 1 — the ONLY thing rendered when the dashboard opens: category buttons.
    // The ☆ lives HERE (never on a leaf feature): a starred CATEGORY is what shows
    // up in the constellation hub / main-menu favorites.
    function buildCategoryGrid() {
        return WD_CATEGORIES.map(function (cat) {
            // data-wd-tree marks the element so §10.7 UI-tree art (one unique placeholder frame per
            // tree, served by ui_tree_theming.go) can paint it; the renderer never guesses an id.
            return `<button class="wd-cat-btn" data-cat="${cat.id}" data-wd-tree="${cat.id}" style="--cat-color:${cat.color}" onclick="window.openWDCategory('${cat.id}')">` +
                `<span class="wd-cat-btn-icon">${cat.icon}</span>` +
                `<span class="wd-cat-btn-name">${esc(cat.name)}</span>` +
                `<span class="wd-cat-btn-count" title="${cat.features.length} features">${cat.features.length}</span>` +
                `<span class="wd-cat-star" data-star-cat="${cat.id}" title="Star this category for quick access">☆</span>` +
                `</button>`;
        }).join('');
    }

    // TIER 2 — the feature menu for ONE category. Choosing a feature opens that
    // feature's UI: ↗ = it opens its own standalone screen (SPA route-out),
    // ▸ = it opens inside this dashboard panel.
    function buildFeatureMenu(cat) {
        return cat.features.map(function (f) {
            const routed = !!WD_ROUTES[f.tab];
            const treeAttr = f.tab === 'rewards' || f.tab === 'achievements'
                ? `data-wd-tree="${f.tab}" data-wd-tree-pinned="Crypto-seraph"` // §10.7: the rule's two pins
                : `data-wd-tree="${f.tab}"`;
            return `<button class="wd-tab wd-feature-btn" data-tab="${f.tab}" data-cat="${cat.id}" ${treeAttr} onclick="window.switchWDTab('${f.tab}')">` +
                `<span class="wd-feature-label">${esc(f.label)}</span>` +
                `<span class="wd-feature-badge" title="${routed ? 'Opens its own screen' : 'Opens in this panel'}">${routed ? '↗' : '▸'}</span>` +
                `</button>`;
        }).join('');
    }

    // §10.7 UI-tree art: ask the server for the tree→frame assignment once, then paint every marked
    // tree. The read is lazy (first paint) and a refused read is left to the renderer, which keeps
    // whatever it painted and states the refusal.
    function paintTreeArt() {
        const st = window.SlideTheming;
        if (!st || typeof st.paintUiTrees !== 'function') return;
        if (!st.uiTrees || !st.uiTrees.loaded) {
            st.loadUiTrees().then(function () { st.paintUiTrees(overlayEl); });
            return;
        }
        st.paintUiTrees(overlayEl);
    }

    // Show TIER 2 for one category and hide TIER 1 (breadcrumb back to categories).
    function openWDCategory(catId) {
        if (!overlayEl) init();
        const cat = categoryById(catId);
        if (!cat) return;
        activeCategory = catId;

        const grid = document.getElementById('wd-cats');
        const menu = document.getElementById('wd-feature-menu');
        const list = document.getElementById('wd-feature-list');
        const title = document.getElementById('wd-feature-title');
        if (title) title.innerHTML = `<span class="wd-feature-title-icon">${cat.icon}</span> ${esc(cat.name)}`;
        if (list) list.innerHTML = buildFeatureMenu(cat);
        if (grid) grid.classList.add('hidden');
        if (menu) menu.classList.remove('hidden');

        // §10.7: the feature buttons were just (re)built, so paint their tree art now.
        paintTreeArt();

        // Nothing is selected yet — hide every panel until a feature is chosen.
        document.querySelectorAll('.wd-panel').forEach(function (p) { p.classList.add('hidden'); });
        document.querySelectorAll('.wd-cat-btn').forEach(function (b) {
            b.classList.toggle('active', b.dataset.cat === catId);
        });
    }

    // Back to TIER 1 (the category grid).
    function wdBackToCategories() {
        activeCategory = null;
        const grid = document.getElementById('wd-cats');
        const menu = document.getElementById('wd-feature-menu');
        if (menu) menu.classList.add('hidden');
        if (grid) grid.classList.remove('hidden');
        document.querySelectorAll('.wd-panel').forEach(function (p) { p.classList.add('hidden'); });
        document.querySelectorAll('.wd-cat-btn').forEach(function (b) { b.classList.remove('active'); });
    }

    function init() {
        if (overlayEl) return;
        const html = `
<div id="world-dashboard-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="wd-bg-slides" id="wd-bg-slides"></div>
    <div class="neon-glass-panel world-dashboard-panel">
        <button id="btn-close-wd" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>🌍 World Dashboard</h2>
        <p class="wd-subtitle">Choose a category to open its features</p>

        <!-- TIER 1 — categories only. A starred category (☆) is what appears in
             the constellation hub / main-menu favorites for quick access. -->
        <div class="wd-cats" id="wd-cats">
            ${buildCategoryGrid()}
        </div>

        <!-- TIER 2 — the feature menu for the chosen category. -->
        <div class="wd-feature-menu hidden" id="wd-feature-menu">
            <div class="wd-feature-head">
                <button class="wd-back-btn" onclick="window.wdBackToCategories()">‹ Categories</button>
                <span class="wd-feature-title" id="wd-feature-title"></span>
            </div>
            <div class="wd-feature-list" id="wd-feature-list"></div>
        </div>

        <div id="wd-panel-career" class="wd-panel"><div id="wd-career"></div></div>
        <div id="wd-panel-orphan" class="wd-panel hidden"><div id="wd-orphan"></div></div>
        <div id="wd-panel-replay" class="wd-panel hidden"><div id="wd-replay"></div></div>
        <div id="wd-panel-season" class="wd-panel hidden"><div id="wd-season"></div></div>
        <div id="wd-panel-stats" class="wd-panel hidden"><div id="wd-stats"></div></div>
        <div id="wd-panel-markets" class="wd-panel hidden"><div id="wd-markets"></div></div>
        <div id="wd-panel-items" class="wd-panel hidden"><div id="wd-items"></div></div>
        <div id="wd-panel-identity" class="wd-panel hidden"><div id="wd-identity"></div></div>
        <div id="wd-panel-loans" class="wd-panel hidden"><div id="wd-loans"></div></div>
        <div id="wd-panel-contracts" class="wd-panel hidden"><div id="wd-contracts"></div></div>
        <div id="wd-panel-counterfeit" class="wd-panel hidden"><div id="wd-counterfeit"></div></div>
        <div id="wd-panel-blackmarket" class="wd-panel hidden"><div id="wd-blackmarket"></div></div>
        <div id="wd-panel-clubs" class="wd-panel hidden"><div id="wd-clubs"></div></div>
        <div id="wd-panel-territory" class="wd-panel hidden"><div id="wd-territory"></div></div>
        <div id="wd-panel-governor" class="wd-panel hidden"><div id="wd-governor"></div></div>
        <div id="wd-panel-infrastructure" class="wd-panel hidden"><div id="wd-infrastructure"></div></div>
        <div id="wd-panel-gamingos" class="wd-panel hidden"><div id="wd-gamingos"></div></div>
        <div id="wd-panel-creator" class="wd-panel hidden"><div id="wd-creator"></div></div>
        <div id="wd-panel-launches" class="wd-panel hidden"><div id="wd-launches"></div></div>
        <div id="wd-panel-localmodel" class="wd-panel hidden"><div id="wd-localmodel"></div></div>
        <div id="wd-panel-tournament" class="wd-panel hidden"><div id="wd-tournament"></div></div>
        <div id="wd-panel-faith" class="wd-panel hidden"><div id="wd-faith"></div></div>
        <div id="wd-panel-governance" class="wd-panel hidden"><div id="wd-governance"></div></div>
        <div id="wd-panel-justice" class="wd-panel hidden"><div id="wd-justice"></div></div>
        <div id="wd-panel-rivalry" class="wd-panel hidden"><div id="wd-rivalry"></div></div>
        <div id="wd-panel-dividends" class="wd-panel hidden"><div id="wd-dividends"></div></div>
        <div id="wd-panel-events" class="wd-panel hidden"><div id="wd-events"></div></div>
        <div id="wd-panel-church" class="wd-panel hidden"><div id="wd-church"></div></div>
        <div id="wd-panel-industrial" class="wd-panel hidden"><div id="wd-industrial"></div></div>
        <div id="wd-panel-vehicles" class="wd-panel hidden"><div id="wd-vehicles"></div></div>
        <div id="wd-panel-pets" class="wd-panel hidden"><div id="wd-pets"></div></div>
        <div id="wd-panel-regions" class="wd-panel hidden"><div id="wd-regions"></div></div>
        <div id="wd-panel-treasure" class="wd-panel hidden"><div id="wd-treasure"></div></div>
        <div id="wd-panel-rewards" class="wd-panel hidden"><div id="wd-rewards"></div></div>
        <div id="wd-panel-compliance" class="wd-panel hidden"><div id="wd-compliance"></div></div>
        <div id="wd-panel-leaderboard" class="wd-panel hidden"><div id="wd-leaderboard"></div></div>
        <div id="wd-panel-match" class="wd-panel hidden"><div id="wd-match"></div></div>
        <div id="wd-panel-ads" class="wd-panel hidden"><div id="wd-ads"></div></div>
        <div id="wd-panel-maintenance" class="wd-panel hidden"><div id="wd-maintenance"></div></div>
        <div id="wd-panel-systemmsg" class="wd-panel hidden"><div id="wd-systemmsg"></div></div>
        <div id="wd-panel-report" class="wd-panel hidden"><div id="wd-report"></div></div>
        <div id="wd-panel-worldcontent" class="wd-panel hidden"><div id="wd-worldcontent"></div></div>
        <div id="wd-panel-achievements" class="wd-panel hidden"><div id="wd-achievements"></div></div>

        <p id="wd-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('world-dashboard-overlay');
        statusEl = document.getElementById('wd-status');
        document.getElementById('btn-close-wd').addEventListener('click', () => { overlayEl.style.display = 'none'; });

        // §10.6: the dashboard's own slideshow backdrop — three living images cycling with a gentle
        // fade, one image of each NPC helper, each re-themeable by the viewer with a bonded asset.
        if (window.SlideTheming) {
            window.SlideTheming.mount('dashboard', document.getElementById('wd-bg-slides'));
            window.SlideTheming.refresh();
            window.SlideTheming.ensureStarter();
        }
        // Promote any legacy LEAF stars to their owning CATEGORY (quick access now
        // means categories, not the buttons inside them), then paint star state.
        migrateLegacyStars();
        bindStarClicks();
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }
    function fmtVBV(micro) { return (parseFloat(micro) / 1000000).toFixed(2); }

    async function api(path, opts) {
        const url = path.startsWith('/api/') ? path : API_BASE + path;
        const resp = await fetch(url, opts);
        if (!resp.ok) { let d = ''; try { d = (await resp.json()).error || ''; } catch(e){} throw new Error(d || ('HTTP ' + resp.status)); }
        return resp.json();
    }

    function switchWDTab(tab) {
        // Merge 'gov' into 'governance'
        if (tab === 'gov') tab = 'governance';
        if (!overlayEl) init();

        // Callers may address a CATEGORY rather than a feature — the starred
        // quick-access buttons do exactly that via `openWorldDashboardToTab`.
        if (tab.indexOf('cat:') === 0) tab = tab.slice(4);
        if (categoryById(tab)) { openWDCategory(tab); return; }
        // SPA navigation: if this location has a dedicated full-screen feature UI,
        // open it directly and close the World Dashboard (do NOT embed it as a
        // sub-panel inside the dashboard). This makes the dashboard behave as a
        // launcher/hub rather than a container for every feature.
        const opener = WD_ROUTES[tab];
        if (opener && typeof window[opener] === 'function') {
            if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
            if (overlayEl) overlayEl.style.display = 'none';
            window[opener]();
            if (overlayEl) overlayEl.style.display = 'none'; // safety: keep dashboard closed
            // Reveal the feature overlay if the opener displayed it but left the legacy
            // `.hidden` class. Global `.hidden { display:none !important }` would otherwise
            // keep it invisible. Only touches overlays the opener explicitly showed.
            try {
                document.querySelectorAll('.overlay, .vbt-overlay').forEach(function (el) {
                    if (el.style.display && el.style.display !== 'none' && el.classList.contains('hidden')) el.classList.remove('hidden');
                });
            } catch (_) {}
            return;
        }
        document.querySelectorAll('.wd-tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab));

        // Keep the Tier-2 menu coherent: if this feature belongs to a category that
        // is not currently open, open it first, so the user always sees which
        // category they are inside and can switch features without backing out.
        const owner = categoryForTab(tab);
        if (owner && activeCategory !== owner.id) openWDCategory(owner.id);

        // Lazily materialise the panel. THIS is the fix for the features that used to
        // dead-end: their `initXxx()` module exists and renders into `#wd-<tab>`, but
        // no container was ever created, so the module early-returned and the user got
        // a "Not Wired Yet" placeholder for a feature that was in fact fully built.
        const panel = ensureWDPanel(tab);
        if (!panel) {
            // Only reachable if the dashboard chrome itself is missing — surface a
            // graceful notice instead of hiding every panel and blanking the dashboard.
            const notice = ensureWDPlaceholder(tab);
            document.querySelectorAll('.wd-panel').forEach(p => p.classList.add('hidden'));
            if (notice) notice.classList.remove('hidden');
            return;
        }
        document.querySelectorAll('.wd-panel').forEach(p => p.classList.add('hidden'));
        panel.classList.remove('hidden');
        if (tab === 'career') {
            if (window.initCareerTree) window.initCareerTree();
            else loadCareer();
        }
        if (tab === 'orphan') {
            if (window.initOrphans) window.initOrphans();
            else loadOrphan();
        }
        if (tab === 'replay') {
            if (window.initReplayViewer) window.initReplayViewer();
            else loadReplay();
        }
        if (tab === 'season') {
            if (window.initSeasonCountdown) window.initSeasonCountdown();
            if (window.initSeasonalEvents) window.initSeasonalEvents();
            else loadSeason();
        }
        if (tab === 'stats') {
            if (window.initStatsOverlay) window.initStatsOverlay();
            else loadStats();
        }
        if (tab === 'markets') {
            if (window.initAmmChart) window.initAmmChart();
            else loadMarkets();
        }
        if (tab === 'items') {
            if (window.initItemsEquip) window.initItemsEquip();
            else loadItems();
        }
        if (tab === 'identity') {
            if (window.initIdentityEditor) window.initIdentityEditor();
            else loadIdentity();
        }
        if (tab === 'loans') {
            if (window.initLoanTerms) window.initLoanTerms();
            else loadLoans();
        }
        if (tab === 'contracts') {
            if (window.initUnderworldContracts) window.initUnderworldContracts();
            else loadContracts();
        }
        if (tab === 'create_match') {
            if (window.initMatchCreator) window.initMatchCreator();
        }
        if (tab === 'counterfeit') {
            if (window.initCounterfeitScanner) window.initCounterfeitScanner();
            else loadCounterfeit();
        }
        if (tab === 'faction') {
            if (window.initFactionShop) window.initFactionShop();
        }
        if (tab === 'blackmarket') {
            if (window.initBlackMarket) window.initBlackMarket();
            else loadBlackMarket();
        }
        if (tab === 'clubs') {
            if (window.initClubFoundry) window.initClubFoundry();
            else loadClubs();
        }
        if (tab === 'territory') {
            if (window.initTerritoryMap) window.initTerritoryMap();
            else loadTerritory();
        }
        if (tab === 'governor') {
            if (window.initGovernor) window.initGovernor();
            else loadGovernor();
        }
        if (tab === 'infrastructure') {
            if (window.initInfrastructure) window.initInfrastructure();
            else loadInfrastructure();
        }
        if (tab === 'gamingos') {
            if (window.initGamingOS) window.initGamingOS();
            else loadGamingOS();
        }
        if (tab === 'creator') {
            if (window.initCreatorStudio) window.initCreatorStudio();
            else loadCreator();
        }
        if (tab === 'launches') {
            if (window.initLaunchpad) window.initLaunchpad();
            else loadLaunches();
        }
        if (tab === 'localmodel') {
            if (window.initLocalModel) window.initLocalModel();
            else loadLocalModel();
        }
        if (tab === 'tournament') {
            if (window.initTournamentBrackets) window.initTournamentBrackets();
            else loadTournament();
        }
        if (tab === 'faith') loadFaith();
        if (tab === 'governance') {
            if (window.initGovernanceChambers) window.initGovernanceChambers();
            else loadGovernance();
        }
        if (tab === 'justice') {
            if (window.initJusticeDashboard) window.initJusticeDashboard();
            else loadJustice();
        }
        if (tab === 'rivalry') {
            if (window.initRivalryChallenge) window.initRivalryChallenge();
            else loadRivalry();
        }
        if (tab === 'dividends') {
            if (window.initDividendYield) window.initDividendYield();
            else loadDividends();
        }
        if (tab === 'events') {
            if (window.initEventsArena) window.initEventsArena();
            else loadEvents();
        }
        if (tab === 'create_match') {
            if (window.initMatchCreator) window.initMatchCreator();
        }
        if (tab === 'deck') {
            if (window.initDeckManager) window.initDeckManager();
        }
        if (tab === 'campaign') {
            if (window.initCampaignMode) window.initCampaignMode();
        }
        if (tab === 'equipment') {
            if (window.initEquipmentSystem) window.initEquipmentSystem();
        }
        if (tab === 'card_progression') {
            if (window.initCardProgression) window.initCardProgression();
        }
        if (tab === 'locations') {
            if (window.initGameLocations) window.initGameLocations();
        }
        if (tab === 'tutorial') {
            if (window.initTutorialSystem) window.initTutorialSystem();
        }
        if (tab === 'game') {
            if (window.initGameBoard) window.initGameBoard();
        }
        if (tab === 'multiplayer') {
            if (window.initGameMultiplayer) window.initGameMultiplayer();
        }
        if (tab === 'card_titles') {
            if (window.initCardTitles) window.initCardTitles();
        }
        if (tab === 'wagers') {
            if (window.initPostMatchWagers) window.initPostMatchWagers();
        }
        if (tab === 'mood') {
            if (window.initCharacterMood) window.initCharacterMood();
        }
        if (tab === 'tea_house') {
            if (window.initTeaHouse) window.initTeaHouse();
        }
        if (tab === 'zen_garden') {
            if (window.initZenGarden) window.initZenGarden();
        }
        if (tab === 'npc_taunts') {
            if (window.initNpcTaunts) window.initNpcTaunts();
        }
        if (tab === 'game_modes') {
            if (window.initGameModes) window.initGameModes();
        }
        if (tab === 'church') {
            if (window.initChurchStorefront) window.initChurchStorefront();
            else loadChurch();
        }
        if (tab === 'industrial') {
            if (window.initIndustrialFlow) window.initIndustrialFlow();
            else loadIndustrial();
        }
        if (tab === 'vehicles') {
            if (window.initVehicles) window.initVehicles();
            else loadVehicles();
        }
        if (tab === 'pets') {
            if (window.initPetBreeder) window.initPetBreeder();
            else loadPets();
        }
        if (tab === 'pet_arena') {
            if (window.initPetArena) window.initPetArena();
        }
        if (tab === 'vehicle_arena') {
            if (window.initVehicleArena) window.initVehicleArena();
        }
        if (tab === 'regions') {
            if (window.initRegions) window.initRegions();
            else loadRegions();
        }
        if (tab === 'treasure') {
            if (window.initTreasureMap) window.initTreasureMap();
            else loadTreasure();
        }
        if (tab === 'rewards') {
            if (window.initRewardsCenter) window.initRewardsCenter();
            else loadRewards();
        }
        if (tab === 'compliance') {
            if (window.initCompliance) window.initCompliance();
            else loadCompliance();
        }
        if (tab === 'leaderboard') {
            if (window.initLeaderboard) window.initLeaderboard();
            else loadLeaderboard();
        }
        if (tab === 'match') {
            if (window.initMatchArena) window.initMatchArena();
            else loadMatch();
        }
        if (tab === 'ads') {
            if (window.initAds) window.initAds();
            else loadAds();
        }
        if (tab === 'maintenance') {
            if (window.initMaintenance) window.initMaintenance();
            else loadMaintenance();
        }
        if (tab === 'systemmsg') {
            if (window.initSystemMsg) window.initSystemMsg();
            else loadSystemMsg();
        }
        if (tab === 'report') {
            if (window.initReportHistory) window.initReportHistory();
            else loadReport();
        }
        if (tab === 'worldcontent') {
            if (window.initWorldContent) window.initWorldContent();
            else loadWorldContent();
        }
        if (tab === 'achievements') {
            if (window.initAchievementProgress) window.initAchievementProgress();
            else loadAchievementsList();
        }
        bindStarClicks();
    }

    // Materialise the `wd-panel-<tab>` / `#wd-<tab>` container for a feature on
    // demand. The original panels are still declared statically; this covers the
    // features that never had one, so their owning `initXxx()` module — which looks
    // up `#wd-<tab>` and early-returns when it is absent — can finally render.
    function ensureWDPanel(tab) {
        let panel = document.getElementById('wd-panel-' + tab);
        if (panel) return panel;
        const root = overlayEl ? overlayEl.querySelector('.world-dashboard-panel')
                               : document.querySelector('.world-dashboard-panel');
        if (!root) return null;
        panel = document.createElement('div');
        panel.id = 'wd-panel-' + tab;
        panel.className = 'wd-panel hidden';
        const inner = document.createElement('div');
        inner.id = 'wd-' + tab;
        panel.appendChild(inner);
        root.appendChild(panel);
        return panel;
    }

    function ensureWDPlaceholder(tab) {
        let p = document.getElementById('wd-panel-__missing');
        if (!p) {
            const root = overlayEl ? overlayEl.querySelector('.world-dashboard-panel') : document.querySelector('.world-dashboard-panel');
            if (!root) return null;
            p = document.createElement('div');
            p.id = 'wd-panel-__missing';
            p.className = 'wd-panel';
            root.appendChild(p);
        }
        p.innerHTML = '<div class="wd-placeholder">' +
            '<h3 style="color:#e5e7eb;margin:0 0 8px;">🚧 ' + esc(tabLabel(tab)) + ' — Not Wired Yet</h3>' +
            '<p style="color:#90a4ae;font-size:13px;margin:0;">This section is defined in the dashboard navigation but its panel/loader is not implemented in this build.</p>' +
            '</div>';
        return p;
    }

    // --- Quick access (STARS) -----------------------------------------------
    // Only CATEGORIES are starrable. A star stores the category id as `wdTab`, so
    // `openWorldDashboardToTab(<catId>)` — which is exactly what the favorites grid
    // calls — resolves back to that category's feature menu.
    function bindStarClicks() {
        document.querySelectorAll('.wd-cat-star').forEach(star => {
            star.addEventListener('click', (e) => {
                e.stopPropagation(); // must not ALSO open the category
                toggleStar(star.dataset.starCat);
            });
        });
        updateStarDisplay();
    }

    function toggleStar(catId) {
        if (!window.UserPreferences) return;
        const cat = categoryById(catId);
        if (!cat) return;
        const label = cat.name;
        const wasStarred = window.UserPreferences.isStarred(cat.id, 'main');
        window.UserPreferences.toggleStar(cat.id, 'main', cat.icon + ' ' + cat.name, cat.icon, cat.color);
        updateStarDisplay();
        if (window.renderStarredItems) window.renderStarredItems();
        if (wasStarred) showToast(`☆ Unstarred "${label}"`, 'info');
        else showToast(`★ Starred "${label}"! View in the Constellation Hub`, 'success');
    }

    function updateStarDisplay() {
        if (!window.UserPreferences) return;
        document.querySelectorAll('.wd-cat-star').forEach(star => {
            const catId = star.dataset.starCat;
            const isStarred = window.UserPreferences.isStarred(catId, 'main');
            star.textContent = isStarred ? '★' : '☆';
            star.classList.toggle('starred', isStarred);
        });
    }

    // Promotion of legacy stars: older builds starred individual FEATURES. Quick
    // access now means CATEGORIES, so each legacy leaf star is replaced by the
    // category that owns it (never left duplicated at both levels).
    function migrateLegacyStars() {
        if (!window.UserPreferences || !window.UserPreferences.getStarred) return;
        let starred = [];
        try { starred = window.UserPreferences.getStarred() || []; } catch (_) { return; }
        for (const item of starred.slice()) {
            const id = item && item.wdTab;
            if (!id || categoryById(id)) continue;   // already a category star
            const cat = categoryForTab(RENAMED_TABS[id] || id);
            if (!cat) continue;                      // unknown/retired feature — leave as-is
            window.UserPreferences.toggleStar(id, item.wdSub || 'main'); // drop the leaf star
            if (!window.UserPreferences.isStarred(cat.id, 'main')) {
                window.UserPreferences.toggleStar(cat.id, 'main', cat.icon + ' ' + cat.name, cat.icon, cat.color);
            }
        }
    }

    // === CAREER ===
    async function loadCareer() {
        const el = document.getElementById('wd-career');
        if (!el) return;
        try {
            const res = await api('/api/career/progress');
            if (res.success !== false) {
                el.innerHTML = `
                    <div class="wd-grid">
                        <div class="wd-stat"><span class="wd-stat-label">Current Path</span><span class="wd-stat-val">${esc(res.current_path || '—')}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Level</span><span class="wd-stat-val">${res.level || 0}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">XP</span><span class="wd-stat-val">${res.xp || 0}</span></div>
                    </div>
                    ${res.bonuses ? `<div class="wd-sub">Bonuses: ${Object.entries(res.bonuses).map(([k,v]) => `${k}: +${v}`).join(', ')}</div>` : ''}`;
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No career data</p>';
            }
        } catch (e) {
            el.innerHTML = '<p style="color:#90a4ae;">Career data unavailable</p>';
        }
    }

    // === ORPHANS ===
    async function loadOrphan() {
        const el = document.getElementById('wd-orphan');
        if (!el) return;
        try {
            const res = await api('/api/orphan/status');
            if (res.success !== false) {
                el.innerHTML = `
                    <div class="wd-grid">
                        <div class="wd-stat"><span class="wd-stat-label">Orphaned</span><span class="wd-stat-val">${res.orphaned ? 'Yes' : 'No'}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Grace Days Left</span><span class="wd-stat-val">${res.grace_days_left || 0}</span></div>
                    </div>
                    <div class="wd-actions">
                        <button class="vbt-btn vbt-btn-primary" onclick="window.wdAdoptOrphan()">Adopt</button>
                        <button class="vbt-btn vbt-btn-secondary" onclick="window.wdReclaimOrphan()">Reclaim</button>
                    </div>`;
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No orphan data</p>';
            }
        } catch (e) {
            el.innerHTML = '<p style="color:#90a4ae;">Orphan data unavailable</p>';
        }
    }

    async function adoptOrphan() {
        const entityId = prompt('Entity ID to adopt:');
        if (!entityId) return;
        try {
            await api('/api/orphan/adopt', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ entity_id: entityId }) });
            showToast('Adoption initiated!', 'success');
            loadOrphan();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    async function reclaimOrphan() {
        const entityId = prompt('Entity ID to reclaim:');
        if (!entityId) return;
        try {
            await api('/api/orphan/reclaim', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ entity_id: entityId }) });
            showToast('Reclaim initiated!', 'success');
            loadOrphan();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    // === REPLAY ===
    async function loadReplay() {
        const el = document.getElementById('wd-replay');
        if (!el) return;
        try {
            const res = await api('/api/replay/latest');
            if (res.success !== false && res.frames) {
                el.innerHTML = `
                    <div class="wd-grid">
                        <div class="wd-stat"><span class="wd-stat-label">Frames</span><span class="wd-stat-val">${res.frames.length}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Match ID</span><span class="wd-stat-val">${esc(res.match_id || '—')}</span></div>
                    </div>`;
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No replays available</p>';
            }
        } catch (e) {
            el.innerHTML = '<p style="color:#90a4ae;">Replay data unavailable</p>';
        }
    }

    // === SEASON ===
    async function loadSeason() {
        const el = document.getElementById('wd-season');
        if (!el) return;
        try {
            const res = await api('/api/season/status');
            if (res.success !== false) {
                el.innerHTML = `
                    <div class="wd-grid">
                        <div class="wd-stat"><span class="wd-stat-label">Season</span><span class="wd-stat-val">${res.season_number || 1}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Status</span><span class="wd-stat-val">${esc(res.status || 'Active')}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Ends</span><span class="wd-stat-val">${esc(res.ends_at || '—')}</span></div>
                    </div>`;
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No season data</p>';
            }
        } catch (e) {
            el.innerHTML = '<p style="color:#90a4ae;">Season data unavailable</p>';
        }
    }

    // === STATS ===
    async function loadStats() {
        const el = document.getElementById('wd-stats');
        if (!el) return;
        try {
            const res = await api('/api/stat-overlay');
            if (res.success !== false) {
                el.innerHTML = `
                    <div class="wd-grid">
                        <div class="wd-stat"><span class="wd-stat-label">Power Level</span><span class="wd-stat-val">${res.power_level || 0}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Effective Level</span><span class="wd-stat-val">${res.effective_level || 0}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Stat Sum</span><span class="wd-stat-val">${res.stat_sum || 0}</span></div>
                    </div>`;
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No stat data</p>';
            }
        } catch (e) {
            el.innerHTML = '<p style="color:#90a4ae;">Stat data unavailable</p>';
        }
    }

    // === MARKETS ===
    async function loadMarkets() {
        const el = document.getElementById('wd-markets');
        if (!el) return;
        el.innerHTML = '<div class="wd-loading">Loading markets…</div>';
        try {
            const [list, auctions, bm] = await Promise.all([
                api('/api/entity/market/list').catch(() => ({ listings: [] })),
                api('/api/auctions').catch(() => ({ auctions: [] })),
                api('/api/black-market/buy').catch(() => ({ items: [] })),
            ]);
            let html = '';
            if (list.listings && list.listings.length) {
                html += '<h4>Entity Market</h4>' + list.listings.slice(0, 5).map(l => `
                    <div class="wd-card">
                        <span>${esc(l.name)}</span>
                        <span>${fmtVBV(l.price_micro)} VBV</span>
                        <button class="vbt-btn vbt-btn-primary" onclick="window.wdBuyMarket('${esc(l.listing_id)}')">Buy</button>
                    </div>`).join('');
            }
            if (auctions.auctions && auctions.auctions.length) {
                html += '<h4>Auctions</h4>' + auctions.auctions.slice(0, 5).map(a => `
                    <div class="wd-card">
                        <span>${esc(a.bundle?.card_id || 'Bundle')}</span>
                        <span>Bid: ${fmtVBV(a.current_bid)} VBV</span>
                    </div>`).join('');
            }
            if (bm.items && bm.items.length) {
                html += '<h4>Black Market</h4>' + bm.items.slice(0, 5).map(i => `
                    <div class="wd-card">
                        <span>${esc(i.name)}</span>
                        <span>${fmtVBV(i.price_micro)} VBV</span>
                    </div>`).join('');
            }
            el.innerHTML = html || '<p style="color:#90a4ae;">No market data</p>';
        } catch (e) {
            el.innerHTML = '<p style="color:#90a4ae;">Market data unavailable</p>';
        }
    }

    async function buyMarket(listingId) {
        try {
            await api('/api/entity/market/purchase', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ listing_id: listingId }) });
            showToast('Purchase initiated!', 'success');
            loadMarkets();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    // === ITEMS ===
    async function loadItems() {
        const el = document.getElementById('wd-items');
        if (!el) return;
        try {
            const res = await api('/api/items/archetypes');
            if (res.success !== false && res.archetypes) {
                el.innerHTML = `<div class="wd-grid">${res.archetypes.map(a => `
                    <div class="wd-stat"><span class="wd-stat-label">${esc(a.name)}</span><span class="wd-stat-val">${esc(a.category)}</span></div>
                `).join('')}</div>`;
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No items</p>';
            }
        } catch (e) {
            el.innerHTML = '<p style="color:#90a4ae;">Item data unavailable</p>';
        }
    }

    // === IDENTITY ===
    async function loadIdentity() {
        const el = document.getElementById('wd-identity');
        if (!el) return;
        try {
            const res = await api('/api/identity/profile');
            if (res.success !== false) {
                el.innerHTML = `
                    <div class="wd-grid">
                        <div class="wd-stat"><span class="wd-stat-label">Handle</span><span class="wd-stat-val">${esc(res.handle || '—')}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Reputation</span><span class="wd-stat-val">${res.reputation || 0}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Social Rank</span><span class="wd-stat-val">${esc(res.social_rank || '—')}</span></div>
                    </div>`;
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No identity data</p>';
            }
        } catch (e) {
            el.innerHTML = '<p style="color:#90a4ae;">Identity data unavailable</p>';
        }
    }

    // === LOANS ===
    async function loadLoans() {
        const el = document.getElementById('wd-loans');
        if (!el) return;
        try {
            const res = await api('/api/loans');
            if (res.loans && res.loans.length) {
                el.innerHTML = res.loans.map(l => `
                    <div class="wd-card">
                        <span>${esc(l.purpose || 'Loan')}</span>
                        <span>${fmtVBV(l.amount_micro)} VBV</span>
                        <span>Rate: ${l.interest_rate || 0}%</span>
                        ${l.borrower ? `<button class="vbt-btn vbt-btn-primary" onclick="window.wdRepayLoan('${esc(l.id)}')">Repay</button>` : `<button class="vbt-btn vbt-btn-primary" onclick="window.wdTakeLoan('${esc(l.id)}')">Take</button>`}
                    </div>`).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No active loans</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Loan data unavailable</p>'; }
    }

    async function takeLoan() {
        const amount = prompt('Loan amount (micro-VBV):');
        if (!amount) return;
        try {
            await api('/api/loans/take', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ amount: parseInt(amount) }) });
            showToast('Loan taken!', 'success');
            loadLoans();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    async function repayLoan(id) {
        try {
            await api('/api/loans/repay', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ loan_id: id }) });
            showToast('Loan repaid!', 'success');
            loadLoans();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    // === CONTRACTS ===
    async function loadContracts() {
        const el = document.getElementById('wd-contracts');
        if (!el) return;
        // Contract eligibility is evaluated per wallet, so the wallet must be sent
        // or the server can only answer "empty".
        const cw = (typeof window.getActiveWallet === 'function')
            ? (window.getActiveWallet() || '') : (window.currentWallet || '');
        if (!cw) {
            el.innerHTML = '<p style="color:#90a4ae;">Connect a wallet to list the contracts it qualifies for.</p>';
            return;
        }
        try {
            const res = await api('/api/contracts/list?wallet=' + encodeURIComponent(cw));
            if (res.contracts && res.contracts.length) {
                el.innerHTML = res.contracts.map(c => `
                    <div class="wd-card">
                        <span>${esc(c.title || c.id)}</span>
                        <span>${fmtVBV(c.reward_micro)} VBV</span>
                        <button class="vbt-btn vbt-btn-primary" onclick="window.wdAcceptContract('${esc(c.id)}')">Accept</button>
                    </div>`).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No contracts available</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Contract data unavailable</p>'; }
    }

    async function acceptContract(id) {
        try {
            await api('/api/contracts/assign', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ contract_id: id }) });
            showToast('Contract accepted!', 'success');
            loadContracts();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    // === COUNTERFEIT ===
    async function loadCounterfeit() {
        const el = document.getElementById('wd-counterfeit');
        if (!el) return;
        try {
            const res = await api('/api/counterfeit/detect');
            if (res.success !== false) {
                el.innerHTML = `<div class="wd-grid">
                    <div class="wd-stat"><span class="wd-stat-label">Detected</span><span class="wd-stat-val">${res.detected || 0}</span></div>
                    <div class="wd-stat"><span class="wd-stat-label">Last Scan</span><span class="wd-stat-val">${esc(res.last_scan || '—')}</span></div>
                </div>
                <div class="wd-actions">
                    <button class="vbt-btn vbt-btn-primary" onclick="window.wdGenerateCounterfeit()">Generate Counterfeit</button>
                </div>`;
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No counterfeit data</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Counterfeit data unavailable</p>'; }
    }

    async function generateCounterfeit() {
        const amount = prompt('Counterfeit amount (micro-VBV):');
        if (!amount) return;
        try {
            await api('/api/counterfeit/generate', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ amount: parseInt(amount) }) });
            showToast('Counterfeit generated!', 'success');
            loadCounterfeit();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    // === BLACK MARKET ===
    async function loadBlackMarket() {
        const el = document.getElementById('wd-blackmarket');
        if (!el) return;
        try {
            const res = await api('/api/black-market/sell-tokens');
            if (res.success !== false && res.items) {
                el.innerHTML = res.items.map(i => `
                    <div class="wd-card">
                        <span>${esc(i.name)}</span>
                        <span>${fmtVBV(i.price_micro)} VBV</span>
                        <button class="vbt-btn vbt-btn-primary" onclick="window.buyBlackMarketItem('${esc(i.id)}', ${Number(i.price_micro || 0) / 1000000})">Buy</button>
                    </div>`).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No black market data</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Black market data unavailable</p>'; }
    }

    // === CLUBS ===
    async function loadClubs() {
        const el = document.getElementById('wd-clubs');
        if (!el) return;
        el.innerHTML = '<div class="wd-loading">Loading clubs…</div>';
        try {
            const res = await api('/api/clubs');
            if (res.success !== false && res.clubs && res.clubs.length) {
                el.innerHTML = `<div class="wd-grid">${res.clubs.map(c => `
                    <div class="wd-card">
                        <span class="wd-card-name">${esc(c.name || c.id)}</span>
                        <span>Type: ${esc(c.type || '—')}</span>
                        <span>Members: ${(c.member_count ?? 0)}</span>
                        <span>Treasury: ${fmtVBV(c.treasury)} VBV</span>
                        <button class="vbt-btn vbt-btn-primary" onclick="window.wdOpenClub('${esc(c.id)}')">View</button>
                    </div>`).join('')}</div>`;
            } else {
                el.innerHTML = `
                    <div class="wd-empty">
                        <p>No clubs yet.</p>
                        <button class="vbt-btn vbt-btn-primary" onclick="if(window.initClubFoundry) window.initClubFoundry()">🏛️ Found a Club</button>
                    </div>`;
            }
        } catch (e) {
            el.innerHTML = `
                <div class="wd-empty">
                    <p>Club data unavailable.</p>
                    <button class="vbt-btn vbt-btn-primary" onclick="if(window.initClubFoundry) window.initClubFoundry()">🏛️ Found a Club</button>
                </div>`;
        }
    }

    // === TERRITORY ===
    async function loadTerritory() {
        const el = document.getElementById('wd-territory');
        if (!el) return;
        el.innerHTML = '<div class="wd-loading">Loading territory…</div>';
        try {
            const res = await api('/api/regions');
            if (res.success !== false && res.regions && res.regions.length) {
                el.innerHTML = `<div class="wd-grid">${res.regions.map(r => `
                    <div class="wd-card">
                        <span class="wd-card-name">${esc(r.name || r.id)}</span>
                        <span>Districts: ${(r.districts?.length ?? 0)}</span>
                        <span>Controlled: ${esc(r.controlled_by || 'Neutral')}</span>
                    </div>`).join('')}</div>`;
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No territory data</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Territory data unavailable</p>'; }
    }

    // === GOVERNOR ===
    async function loadGovernor() {
        const el = document.getElementById('wd-governor');
        if (!el) return;
        el.innerHTML = '<div class="wd-loading">Loading governor…</div>';
        try {
            const res = await api('/api/governance/weight');
            if (res.success !== false) {
                el.innerHTML = `
                    <div class="wd-grid">
                        <div class="wd-stat"><span class="wd-stat-label">Voting Power</span><span class="wd-stat-val">${res.voting_power || 0}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Delegated</span><span class="wd-stat-val">${res.delegated || 'None'}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Proposals</span><span class="wd-stat-val">${res.proposal_count || 0}</span></div>
                    </div>`;
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No governor data</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Governor data unavailable</p>'; }
    }

    window.wdOpenClub = function (clubId) {
        if (window.openClubFoundry) {
            window.openClubFoundry(clubId);
        } else {
            showToast('Club foundry not available', 'error');
        }
    };
    // The infrastructure panel reads the SERVED catalogue shape: `systems[]` with `type`, `name`,
    // `description` and `monthly_rate` (integer micro units). It used to read
    // `res.leases[].price_micro`, a shape this route has never served, so the panel could only ever
    // answer "No leases available". Leasing a TENANT WORLD is design-only today and the route states
    // why, so the panel states that too instead of offering a control that can only refuse.
    async function loadInfrastructure() {
        const el = document.getElementById('wd-infrastructure');
        if (!el) return;
        try {
            const res = await api('/api/lease/available');
            const systems = Array.isArray(res.systems) ? res.systems : [];
            const blocked = Array.isArray(res.creation_blocked_by) ? res.creation_blocked_by : [];
            const notice = res.creation_available === false
                ? `<p style="color:#ffb74d;">Leasing is not available yet. Blocked by: ${blocked.map(b => esc(b)).join('; ')}</p>`
                : '';
            if (!systems.length) {
                el.innerHTML = notice + '<p style="color:#90a4ae;">The lease catalogue is empty.</p>';
                return;
            }
            el.innerHTML = notice + systems.map(s => `
                <div class="wd-card">
                    <span>${esc(s.name)}</span>
                    <span>${fmtVBV(s.monthly_rate)} VBV/period</span>
                    <small>${esc(s.description)}</small>
                </div>`).join('');
        } catch (e) {
            el.innerHTML = `<p style="color:#90a4ae;">Infrastructure data unavailable: ${esc(e.message)}</p>`;
        }
    }

    // === GAMING OS ===
    async function loadGamingOS() {
        const el = document.getElementById('wd-gamingos');
        if (!el) return;
        try {
            const res = await api('/api/os/modules');
            if (res.modules && res.modules.length) {
                el.innerHTML = res.modules.map(m => `
                    <div class="wd-card">
                        <span>${esc(m.name)}</span>
                        <span>${esc(m.version || 'v1.0')}</span>
                        <span>${m.active ? '✅ Active' : 'Inactive'}</span>
                    </div>`).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No modules loaded</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Gaming OS data unavailable</p>'; }
    }

    // === CREATOR ===
    async function loadCreator() {
        const el = document.getElementById('wd-creator');
        if (!el) return;
        try {
            const res = await api('/api/creator/store/products');
            if (res.products && res.products.length) {
                el.innerHTML = res.products.map(p => `
                    <div class="wd-card">
                        <span>${esc(p.name)}</span>
                        <span>${fmtVBV(p.price_micro)} VBV</span>
                    </div>`).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No creator products</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Creator data unavailable</p>'; }
    }

    // === LAUNCHES ===
    async function loadLaunches() {
        const el = document.getElementById('wd-launches');
        if (!el) return;
        try {
            const res = await api('/api/launches');
            if (res.launches && res.launches.length) {
                el.innerHTML = res.launches.map(l => `
                    <div class="wd-card">
                        <span>${esc(l.name)}</span>
                        <span>${esc(l.status)}</span>
                    </div>`).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No launches</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Launch data unavailable</p>'; }
    }

    // === LOCAL LLM ===
    async function loadLocalModel() {
        const el = document.getElementById('wd-localmodel');
        if (!el) return;
        try {
            const res = await api('/api/local-model/status');
            if (res.success !== false) {
                el.innerHTML = `<div class="wd-grid">
                    <div class="wd-stat"><span class="wd-stat-label">Tier</span><span class="wd-stat-val">${esc(res.tier || '—')}</span></div>
                    <div class="wd-stat"><span class="wd-stat-label">Instances</span><span class="wd-stat-val">${res.instances || 0}</span></div>
                    <div class="wd-stat"><span class="wd-stat-label">Backend</span><span class="wd-stat-val">${esc(res.backend || '—')}</span></div>
                </div>`;
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No local model data</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Local model data unavailable</p>'; }
    }

    async function promoteLocalModel() {
        const wallet = prompt('Wallet to promote:');
        if (!wallet) return;
        try {
            await api('/api/local-model/promote', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ wallet }) });
            showToast('Promotion initiated!', 'success');
            loadLocalModel();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    // === TOURNAMENT ===
    async function loadTournament() {
        const el = document.getElementById('wd-tournament');
        if (!el) return;
        try {
            const res = await api('/api/tournament/history');
            if (res.tournaments && res.tournaments.length) {
                el.innerHTML = res.tournaments.map(t => `
                    <div class="wd-card">
                        <span>${esc(t.name)}</span>
                        <span>Pot: ${fmtVBV(t.pot_micro)} VBV</span>
                        <span>${esc(t.winner || '—')}</span>
                    </div>`).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No tournaments</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Tournament data unavailable</p>'; }
    }

    async function registerTournament() {
        const id = prompt('Tournament ID to register:');
        if (!id) return;
        try {
            await api('/api/tournament/register', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ tournament_id: id }) });
            showToast('Registered!', 'success');
            loadTournament();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    // === ACHIEVEMENTS ===
    // Accepts the achievement id from the rendered button, and falls back to a prompt when it is
    // called with none — the markup has always passed an id, which the prompt-only version ignored.
    async function unlockAchievement(achievementId) {
        const id = achievementId || prompt('Achievement ID to unlock:');
        if (!id) return;
        try {
            await api('/api/achievement/unlock', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ achievement_id: id }) });
            showToast('Achievement unlocked!', 'success');
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    // === ACHIEVEMENTS LIST ===
    async function loadAchievementsList() {
        const el = document.getElementById('wd-achievements');
        if (!el) return;
        const wallet = (window.getWalletAddress && window.getWalletAddress()) || '';
        if (!wallet) { el.innerHTML = '<p style="color:#90a4ae;">Connect wallet to view achievements.</p>'; return; }
        try {
            const res = await api('/api/achievements?wallet=' + encodeURIComponent(wallet));
            const defs = res.definitions || [];
            const unlocked = new Set(res.unlocked || []);
            if (!defs.length) { el.innerHTML = '<p style="color:#90a4ae;">No achievements defined.</p>'; return; }
            el.innerHTML = defs.map(d => `
                <div class="wd-card">
                    <span>${esc(d.title)}</span>
                    <span>${esc(d.rarity || '')}</span>
                    ${unlocked.has(d.id) ? '<span>✓ Unlocked</span>' : `<button class="vbt-btn vbt-btn-primary" onclick="window.wdUnlockAchievement('${esc(d.id)}')">Unlock</button>`}
                </div>`).join('');
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Achievement data unavailable</p>'; }
    }

    // === FAITH ===
    async function loadFaith() {
        const el = document.getElementById('wd-faith');
        if (!el) return;
        try {
            const res = await api('/api/faith/coherence');
            if (res.success !== false) {
                el.innerHTML = `
                    <div class="wd-grid">
                        <div class="wd-stat"><span class="wd-stat-label">Faith Coherence</span><span class="wd-stat-val">${res.faith_coherence || 0}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Score</span><span class="wd-stat-val">${res.score || 0}</span></div>
                    </div>
                    <div class="wd-actions">
                        <button class="pp-action-btn" onclick="window.openFaithChurch && window.openFaithChurch()">⛪ Open Faith Church</button>
                    </div>`;
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No faith data</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Faith data unavailable</p>'; }
    }

    // === GOVERNANCE (merged with gov) ===
    async function loadGovernance() {
        const el = document.getElementById('wd-governance');
        if (!el) return;
        try {
            const res = await api('/api/governance/weight');
            if (res.success !== false && res.weight) {
                const w = res.weight;
                el.innerHTML = `
                    <div class="wd-grid">
                        <div class="wd-stat"><span class="wd-stat-label">Total Score</span><span class="wd-stat-val">${w.total_score || 0}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Trust</span><span class="wd-stat-val">${w.trust_score || 0}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Reputation</span><span class="wd-stat-val">${w.reputation_score || 0}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Economic</span><span class="wd-stat-val">${w.economic_score || 0}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Competitive</span><span class="wd-stat-val">${w.competitive_score || 0}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Community</span><span class="wd-stat-val">${w.community_score || 0}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Creative</span><span class="wd-stat-val">${w.creative_score || 0}</span></div>
                    </div>
                    <div class="wd-actions">
                        <button class="pp-action-btn" onclick="window.openGovernancePanel && window.openGovernancePanel()">⚖️ Open Governance</button>
                    </div>`;
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No governance data</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Governance data unavailable</p>'; }
    }

    // === JUSTICE ===
    async function loadJustice() {
        const el = document.getElementById('wd-justice');
        if (!el) return;
        try {
            const res = await api('/api/justice/dashboard');
            if (res.success !== false) {
                el.innerHTML = `
                    <div class="wd-grid">
                        <div class="wd-stat"><span class="wd-stat-label">Power Bonus</span><span class="wd-stat-val">${res.powerBonus || 0}%</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Tier</span><span class="wd-stat-val">${res.tier || 0}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Bounties</span><span class="wd-stat-val">${res.bounties?.length || 0}</span></div>
                    </div>`;
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No justice data</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Justice data unavailable</p>'; }
    }

    // === RIVALRY ===
    async function loadRivalry() {
        const el = document.getElementById('wd-rivalry');
        if (!el) return;
        try {
            const res = await api('/api/rivalry/list');
            if (res.success !== false && res.rivalries) {
                el.innerHTML = res.rivalries.slice(0, 5).map(r => `
                    <div class="wd-card">
                        <span>${esc(r.side_a)} vs ${esc(r.side_b)}</span>
                        <span>${r.score_a} - ${r.score_b}</span>
                    </div>`).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No rivalries</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Rivalry data unavailable</p>'; }
    }

    // === DIVIDENDS ===
    async function loadDividends() {
        const el = document.getElementById('wd-dividends');
        if (!el) return;
        try {
            // Read-only status; the POST /api/claim/dividends endpoint owns the claim.
            const res = await api('/api/dividends');
            if (res.success !== false) {
                el.innerHTML = `
                    <div class="wd-grid">
                        <div class="wd-stat"><span class="wd-stat-label">Available</span><span class="wd-stat-val">${fmtVBV(res.available_micro || 0)} VBV</span></div>
                    </div>
                    <div class="wd-actions">
                        <button class="vbt-btn vbt-btn-primary" onclick="window.wdClaimDividends()">Claim Dividends</button>
                    </div>`;
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No dividends</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Dividend data unavailable</p>'; }
    }

    async function claimDividends() {
        try {
            await api('/api/claim/dividends', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({}) });
            showToast('Dividends claimed!', 'success');
            loadDividends();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    // === EVENTS ===
    async function loadEvents() {
        const el = document.getElementById('wd-events');
        if (!el) return;
        try {
            const res = await api('/api/season/events');
            if (res.success !== false && res.events) {
                el.innerHTML = res.events.slice(0, 5).map(ev => `
                    <div class="wd-card">
                        <span>${esc(ev.title || ev.name)}</span>
                        <span>${esc(ev.status)}</span>
                        <button class="vbt-btn vbt-btn-primary" onclick="window.wdJoinEvent('${esc(ev.id)}')">Join</button>
                    </div>`).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No events</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Event data unavailable</p>'; }
    }

    async function joinEvent(id) {
        try {
            await api('/api/season/events/join', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ event_id: id }) });
            showToast('Joined event!', 'success');
            loadEvents();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    // === CHURCH ===
    async function loadChurch() {
        const el = document.getElementById('wd-church');
        if (!el) return;
        try {
            const res = await api('/api/church/owner');
            if (res.success !== false) {
                el.innerHTML = `
                    <div class="wd-grid">
                        <div class="wd-stat"><span class="wd-stat-label">Churches</span><span class="wd-stat-val">${res.churches?.length || 0}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Total Income</span><span class="wd-stat-val">${fmtVBV(res.total_income_micro || 0)} VBV</span></div>
                    </div>
                    <div class="wd-actions">
                        <button class="pp-action-btn" onclick="window.openFaithChurch && window.openFaithChurch()">⛪ Open Faith Church</button>
                    </div>`;
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No church data</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Church data unavailable</p>'; }
    }

    // === INDUSTRIAL LOOP ===
    async function loadIndustrial() {
        const el = document.getElementById('wd-industrial');
        if (!el) return;
        try {
            const res = await api('/api/industrial-loop/metrics');
            if (res.success !== false) {
                el.innerHTML = `
                    <div class="wd-grid">
                        <div class="wd-stat"><span class="wd-stat-label">Businesses</span><span class="wd-stat-val">${res.business_count || 0}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Employees</span><span class="wd-stat-val">${res.employee_count || 0}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Treasury</span><span class="wd-stat-val">${fmtVBV(res.treasury_micro || 0)} VBV</span></div>
                    </div>`;
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No industrial data</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Industrial data unavailable</p>'; }
    }

    // === VEHICLES ===
    async function loadVehicles() {
        const el = document.getElementById('wd-vehicles');
        if (!el) return;
        try {
            const res = await api('/api/vehicles');
            if (res.success !== false && res.vehicles) {
                el.innerHTML = res.vehicles.slice(0, 5).map(v => `
                    <div class="wd-card">
                        <span>${esc(v.name || v.model)}</span>
                        <span>${esc(v.status || 'idle')}</span>
                    </div>`).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No vehicles</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Vehicle data unavailable</p>'; }
    }

    // === PETS ===
    async function loadPets() {
        const el = document.getElementById('wd-pets');
        if (!el) return;
        try {
            const res = await api('/api/pets');
            if (res.success !== false && res.pets) {
                el.innerHTML = res.pets.slice(0, 5).map(p => `
                    <div class="wd-card">
                        <span>${esc(p.name)}</span>
                        <span>${esc(p.element || 'unknown')}</span>
                        <span>Lv ${p.level || 1}</span>
                    </div>`).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No pets</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Pet data unavailable</p>'; }
    }

    // === REGIONS ===
    async function loadRegions() {
        const el = document.getElementById('wd-regions');
        if (!el) return;
        try {
            const res = await api('/api/regions');
            if (res.success !== false && res.regions) {
                el.innerHTML = res.regions.slice(0, 5).map(r => `
                    <div class="wd-card">
                        <span>${esc(r.name)}</span>
                        <span>${esc(r.status || 'neutral')}</span>
                    </div>`).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No regions</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Region data unavailable</p>'; }
    }

    // === TREASURE ===
    async function loadTreasure() {
        const el = document.getElementById('wd-treasure');
        if (!el) return;
        try {
            const res = await api('/api/treasure/spawn');
            if (res.success !== false) {
                el.innerHTML = `
                    <div class="wd-grid">
                        <div class="wd-stat"><span class="wd-stat-label">Treasures</span><span class="wd-stat-val">${res.treasures?.length || 0}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Claimed</span><span class="wd-stat-val">${res.claimed || 0}</span></div>
                    </div>
                    <div class="wd-actions">
                        <button class="vbt-btn vbt-btn-primary" onclick="window.wdClaimTreasure()">Claim Treasure</button>
                    </div>`;
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No treasures</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Treasure data unavailable</p>'; }
    }

    async function claimTreasure() {
        try {
            await api('/api/treasure/claim', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({}) });
            showToast('Treasure claimed!', 'success');
            loadTreasure();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    // === REWARDS ===
    async function loadRewards() {
        const el = document.getElementById('wd-rewards');
        if (!el) return;
        try {
            // Read-only summary; the POST /api/reward endpoint owns the actual claim.
            const res = await api('/api/rewards');
            if (res.success !== false && res.rewards) {
                el.innerHTML = res.rewards.slice(0, 5).map(r => `
                    <div class="wd-card">
                        <span>${esc(r.name)}</span>
                        <span>${esc(r.status || 'available')}</span>
                    </div>`).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No rewards</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Reward data unavailable</p>'; }
    }

    // === COMPLIANCE ===
    async function loadCompliance() {
        const el = document.getElementById('wd-compliance');
        if (!el) return;
        try {
            const res = await api('/api/compliance/records');
            if (res.success !== false && res.records) {
                el.innerHTML = res.records.slice(0, 5).map(r => `
                    <div class="wd-card">
                        <span>${esc(r.type)}</span>
                        <span>${esc(r.status)}</span>
                    </div>`).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No compliance records</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Compliance data unavailable</p>'; }
    }

    // === LEADERBOARD ===
    async function loadLeaderboard() {
        const el = document.getElementById('wd-leaderboard');
        if (!el) return;
        try {
            const res = await api('/api/leaderboard');
            if (res.success !== false && res.leaderboard) {
                el.innerHTML = res.leaderboard.slice(0, 10).map((p, i) => `
                    <div class="wd-card">
                        <span>#${i + 1}</span>
                        <span>${esc(p.name || p.wallet)}</span>
                        <span>${p.score || 0}</span>
                    </div>`).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No leaderboard data</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Leaderboard unavailable</p>'; }
    }

    // === MATCH ===
    async function loadMatch() {
        const el = document.getElementById('wd-match');
        if (!el) return;
        try {
            const res = await api('/api/match/wager');
            if (res.success !== false) {
                el.innerHTML = `
                    <div class="wd-grid">
                        <div class="wd-stat"><span class="wd-stat-label">Active Matches</span><span class="wd-stat-val">${res.active_matches || 0}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Wager Min</span><span class="wd-stat-val">${fmtVBV(res.wager_min_micro || 0)} VBV</span></div>
                    </div>`;
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No match data</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Match data unavailable</p>'; }
    }

    // === ADS ===
    async function loadAds() {
        const el = document.getElementById('wd-ads');
        if (!el) return;
        try {
            const res = await api('/api/ads/advertiser');
            if (res.success !== false && res.ads) {
                el.innerHTML = res.ads.slice(0, 5).map(a => `
                    <div class="wd-card">
                        <span>${esc(a.title)}</span>
                        <span>${esc(a.status)}</span>
                    </div>`).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No ads</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Ad data unavailable</p>'; }
    }

    // === MAINTENANCE ===
    async function loadMaintenance() {
        const el = document.getElementById('wd-maintenance');
        if (!el) return;
        try {
            const res = await api('/api/maintenance-mode');
            if (res.success !== false) {
                el.innerHTML = `
                    <div class="wd-grid">
                        <div class="wd-stat"><span class="wd-stat-label">Status</span><span class="wd-stat-val">${res.enabled ? 'Active' : 'Inactive'}</span></div>
                        <div class="wd-stat"><span class="wd-stat-label">Message</span><span class="wd-stat-val">${esc(res.message || '—')}</span></div>
                    </div>`;
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No maintenance data</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">Maintenance data unavailable</p>'; }
    }

    // === SYSTEM MSG ===
    async function loadSystemMsg() {
        const el = document.getElementById('wd-systemmsg');
        if (!el) return;
        try {
            const res = await api('/api/system-message');
            if (res.success !== false && res.messages) {
                el.innerHTML = res.messages.slice(0, 5).map(m => `
                    <div class="wd-card">
                        <span>${esc(m.text || m.message)}</span>
                        <span>${esc(m.timestamp || '')}</span>
                    </div>`).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No system messages</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">System messages unavailable</p>'; }
    }

    // === REPORT ===
    async function loadReport() {
        const el = document.getElementById('wd-report');
        if (!el) return;
        el.innerHTML = `
            <div class="wd-grid">
                <div class="wd-stat"><span class="wd-stat-label">Report Player</span><span class="wd-stat-val">File a report</span></div>
            </div>
            <div class="wd-actions">
                <input id="wd-report-input" type="text" placeholder="Player wallet..." style="padding:8px;background:rgba(255,255,255,0.05);border:1px solid rgba(255,255,255,0.1);color:#fff;border-radius:4px;" />
                <button class="vbt-btn vbt-btn-primary" onclick="window.wdSubmitReport()">Submit Report</button>
            </div>`;
    }

    async function submitReport() {
        const input = document.getElementById('wd-report-input');
        const wallet = input?.value.trim();
        if (!wallet) return;
        try {
            await api('/api/report-player', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ wallet }) });
            showToast('Report submitted!', 'success');
            loadReport();
        } catch (e) { showToast('Failed: ' + e.message, 'error'); }
    }

    // === WORLD CONTENT ===
    async function loadWorldContent() {
        const el = document.getElementById('wd-worldcontent');
        if (!el) return;
        try {
            const res = await api('/api/world-content');
            if (res.success !== false && res.content) {
                el.innerHTML = res.content.slice(0, 5).map(c => `
                    <div class="wd-card">
                        <span>${esc(c.name || c.title)}</span>
                        <span>${esc(c.status || 'active')}</span>
                    </div>`).join('');
            } else {
                el.innerHTML = '<p style="color:#90a4ae;">No world content</p>';
            }
        } catch (e) { el.innerHTML = '<p style="color:#90a4ae;">World content unavailable</p>'; }
    }

    function setStatus(m) { if (statusEl) statusEl.textContent = m || ''; }
    function showToast(msg, type) { if (window.showToast) window.showToast(msg, type); }

    // Show the dashboard chrome. Deliberately does NOT auto-route to any feature
    // overlay — the dashboard must land on its own Tier-1 category grid.
    function showWD() {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        if (window.PanelManager) window.PanelManager.open(overlayEl);
        else overlayEl.style.display = 'flex';
        // §10.6: wake the dashboard slideshow now that it is in the layout (its cycle skips every
        // tick while the overlay is hidden, so it costs nothing behind another panel).
        if (window.SlideTheming) window.SlideTheming.resume('dashboard');
        // §10.7: give every marked UI tree its own placeholder frame.
        paintTreeArt();
    }

    // Open the World Dashboard at TIER 1 (the category grid) — the single entry
    // point to every feature in the game.
    window.openWorldDashboard = function () {
        showWD();
        wdBackToCategories();
    };

    // Open at TIER 2 for one category. This is what a STARRED CATEGORY resolves to.
    window.openWorldDashboardToCategory = function (catId) {
        const id = String(catId || '').replace(/^cat:/, '');
        showWD();
        if (categoryById(id)) openWDCategory(id);
        else wdBackToCategories();
    };

    // The single tab dispatcher. Kept as the one entry point because the favorites
    // grid, other modules and deep-links all call it: a CATEGORY id opens that
    // category's feature menu, a FEATURE id opens the feature itself.
    window.openWorldDashboardToTab = function (tab) {
        showWD();
        switchWDTab(tab);
    };

    window.openWDCategory = openWDCategory;
    window.wdBackToCategories = wdBackToCategories;
    window.switchWDTab = switchWDTab;

    // -------------------------------------------------------------------------
    // THE TAXONOMY, PUBLISHED READ-ONLY — the ONE owner of the category list
    // -------------------------------------------------------------------------
    // Any surface that needs to know which categories/features exist reads THIS.
    // It exists because competing lists had accumulated: `constellation_config.js`
    // (a third list of 12 ids, retired 2026-09-15) and `menu-dock.js` (8 hard-coded
    // ids, retired the same day), while the customization panel shipped its own
    // sample list whose ids ('world','battle','justice',…) matched no real button —
    // so a per-button override set there could never apply. A FROZEN projection
    // keeps the World Dashboard the single owner without letting a consumer mutate
    // the dashboard's own table.
    const WD_TAXONOMY = Object.freeze({
        categories: Object.freeze(WD_CATEGORIES.map(function (c) {
            return Object.freeze({
                id: c.id, icon: c.icon, name: c.name, color: c.color,
                features: Object.freeze(c.features.map(function (f) {
                    return Object.freeze({
                        tab: f.tab,
                        label: f.label,
                        routes_out: !!WD_ROUTES[f.tab],
                        opener: WD_ROUTES[f.tab] || null,
                    });
                })),
            });
        })),
        categoryById: function (id) {
            const key = String(id == null ? '' : id).replace(/^cat:/, '');
            for (const c of WD_TAXONOMY.categories) { if (c.id === key) return c; }
            return null;
        },
        categoryForTab: function (tab) {
            for (const c of WD_TAXONOMY.categories) {
                if (c.features.some(function (f) { return f.tab === tab; })) return c;
            }
            return null;
        },
        featureTabs: function () {
            const out = [];
            WD_TAXONOMY.categories.forEach(function (c) {
                c.features.forEach(function (f) { out.push(f.tab); });
            });
            return out;
        },
    });
    window.WDTaxonomy = WD_TAXONOMY;

    window.wdAdoptOrphan = adoptOrphan;
    window.wdReclaimOrphan = reclaimOrphan;
    window.wdBuyMarket = buyMarket;
    window.wdTakeLoan = takeLoan;
    window.wdRepayLoan = repayLoan;
    window.wdClaimDividends = claimDividends;
    window.wdJoinEvent = joinEvent;
    window.wdClaimTreasure = claimTreasure;
    window.wdSubmitReport = submitReport;
    // Three handler names this module's OWN rendered markup uses (the contract list's Accept button,
    // the counterfeit panel's Generate button and the achievements list's Unlock button). Each
    // function already existed here; an inline handler resolves its name on `window`, so all three
    // buttons were inert until they were published from this list.
    window.wdAcceptContract = acceptContract;
    window.wdGenerateCounterfeit = generateCounterfeit;
    window.wdUnlockAchievement = unlockAchievement;

    if (document.getElementById('world-dashboard-overlay')) init();
})();
