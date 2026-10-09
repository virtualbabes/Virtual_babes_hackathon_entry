// ============================================================================
// placeholder_assets.js — Default placeholder images for World Dashboard tabs
// ----------------------------------------------------------------------------
// Each tab maps to an NPC-helper character. Users can override with bound NFTs.
// "none" option available to disable placeholder entirely.
// ============================================================================

(function () {
    'use strict';

    // NPC helper base paths
    const NPC = {
        anya: '/Assets/Images/NPC-helpers/Anya',
        cryptoSeraph: '/Assets/Images/NPC-helpers/Crypto-seraph',
        vbabes: '/Assets/Images/NPC-helpers/Vbabes',
    };

    // The ACTUAL frames present on disk, per character (117 PNGs total: Anya 67, Vbabes 46,
    // Crypto-seraph 4). Each entry is an inclusive [first,last] range, so the art pool is exact
    // rather than guessed — a missing frame would be a broken placeholder image.
    //
    // FILE NAMING FIX (2026-09-14): the shipped art was named "<Prefix> (N).png" — spaces and
    // parentheses, which are hostile to URLs, shells, globs and manifest tooling. Every file is
    // now "<Prefix>-NNN.png" (zero-padded to 3 digits), so a lexical sort IS a numeric sort and a
    // path needs no escaping. The RANGES below are unchanged because no frame was added or lost.
    const FRAME_SPECS = {
        anya: { dir: 'Anya', prefix: 'Anya', ranges: [[1, 1], [6, 6], [9, 73]] },
        cryptoSeraph: { dir: 'Crypto-seraph', prefix: 'Crypto-seraph', ranges: [[1, 4]] },
        vbabes: { dir: 'Vbabes', prefix: 'Vbabes', ranges: [[1, 1], [17, 61]] },
    };

    // ZERO-PADDING is part of the file name, so it lives in exactly one place.
    function pad3(n) {
        var v = Math.max(0, Math.round(Number(n) || 0));
        return (v < 10 ? '00' : v < 100 ? '0' : '') + v;
    }

    // ── SLIDESHOW SLOTS (app boot / main menu and World Dashboard background) ──
    // The server (placeholder_assets.go + slide_theming.go) is authoritative for the slot list,
    // the prices and the rule; this table is the BOOT FALLBACK the client paints before its first
    // read completes, so the boot screen is never blank. When the served policy arrives it
    // REPLACES this table (server wins), so a server-side change is never overruled by a client.
    // One image of each NPC helper per slideshow, exactly as specified.
    const SLIDES = {
        menu: {
            label: 'App boot / main menu',
            slots: ['menu_slide_1', 'menu_slide_2', 'menu_slide_3'],
            defaults: [
                { npc: 'anya', frame: 1 },
                { npc: 'cryptoSeraph', frame: 2 },
                { npc: 'vbabes', frame: 46 },
            ],
        },
        dashboard: {
            label: 'World Dashboard background',
            slots: ['dashboard_slide_1', 'dashboard_slide_2', 'dashboard_slide_3'],
            defaults: [
                { npc: 'vbabes', frame: 54 },
                { npc: 'cryptoSeraph', frame: 4 },
                { npc: 'anya', frame: 41 },
            ],
        },
    };

    // CARD-VIEW STAND-IN SUPPLY. A bonded asset's art may be CONTENT-ADDRESSED (ipfs:// / ar://),
    // which a browser cannot load without a gateway we do not invent. In that one case the client
    // paints a deterministic NPC-helper stand-in instead of nothing - and it can only reach this art
    // while wearing a REAL asset: the server refuses an asset-free theme (card_view_skins.go
    // `asset_required`), so this is not a way to theme cards for free.
    const CARD_VIEW_NPCS = ['cryptoSeraph', 'anya', 'vbabes'];

    // Deterministic FNV-1a hash (mirrors Public/js/mechanics.js). Used to PICK art without
    // Math.random, so the same slot + seed always yields the same image on every client.
    function fnv1a(str) {
        let h = 0x811c9dc5;
        const s = String(str == null ? '' : str);
        for (let i = 0; i < s.length; i++) {
            h ^= s.charCodeAt(i);
            h = (h + ((h << 1) + (h << 4) + (h << 7) + (h << 8) + (h << 24))) >>> 0;
        }
        return h >>> 0;
    }

    // frameNumbers returns every real frame number for a character (expanded from the ranges).
    function frameNumbers(npc) {
        const spec = FRAME_SPECS[npc];
        if (!spec) return [];
        const out = [];
        spec.ranges.forEach(function (r) {
            for (let n = r[0]; n <= r[1]; n++) out.push(n);
        });
        return out;
    }

    // framePath builds the URL for one frame: "<Prefix>-NNN.png" (URL-safe, no escaping needed).
    function framePath(npc, frameNumber) {
        const spec = FRAME_SPECS[npc];
        if (!spec) return null;
        return NPC[npc] + '/' + spec.prefix + '-' + pad3(frameNumber) + '.png';
    }

    // getSlideDefaults returns the boot-fallback art for one slideshow ("menu" | "dashboard"):
    // one entry per slot, in slot order. Nothing here is authoritative — see SLIDES above.
    function getSlideDefaults(which) {
        const spec = SLIDES[which];
        if (!spec) return [];
        return spec.defaults.map(function (d, i) {
            return { slot: spec.slots[i], npc: d.npc, frame: d.frame, uri: framePath(d.npc, d.frame) };
        });
    }

    // getFrames returns every art path available for a character (the full pool, in order).
    function getFrames(npc) {
        return frameNumbers(npc).map(function (n) { return framePath(npc, n); });
    }

    // getCardViewFallback picks stand-in art for ONE asset id. Seeding by the asset id (not the
    // viewer) means every viewer of that asset sees the same stand-in, and it stays stable across
    // reloads; the pick uses FNV-1a over the exact on-disk frame pool, never Math.random.
    function getCardViewFallback(seed) {
        const s = String(seed == null ? '' : seed).trim();
        if (!s) return null;
        const npc = CARD_VIEW_NPCS[fnv1a('cvs:' + s) % CARD_VIEW_NPCS.length];
        const frames = frameNumbers(npc);
        if (!frames.length) return null;
        const frame = frames[fnv1a('cvsframe:' + s) % frames.length];
        return {
            path: framePath(npc, frame),
            npc: npc,
            frame: frame,
            label: 'NPC-helper stand-in (' + npc + ' frame ' + frame + ') for a content-addressed asset',
        };
    }


    // Default placeholder per tab (which NPC character represents this aspect)
    const TAB_PLACEHOLDERS = {
        // Phase 0
        onboarding: { npc: 'anya', label: 'Anya — Tutorial Guide' },
        tutorial: { npc: 'anya', label: 'Anya — Tutorial Guide' },
        // Phase 1
        battle: { npc: 'anya', label: 'Anya — Combat Trainer' },
        rewards: { npc: 'cryptoSeraph', label: 'Crypto-Seraph — Vault Keeper' },
        faucet: { npc: 'cryptoSeraph', label: 'Crypto-Seraph — Vault Keeper' },
        // Phase 2
        markets: { npc: 'vbabes', label: 'Vbabes — Market Maker' },
        loans: { npc: 'cryptoSeraph', label: 'Crypto-Seraph — Banker' },
        investments: { npc: 'cryptoSeraph', label: 'Crypto-Seraph — Fund Manager' },
        // Phase 3
        leaderboard: { npc: 'vbabes', label: 'Vbabes — Rank Keeper' },
        identity: { npc: 'anya', label: 'Anya — Identity Clerk' },
        career: { npc: 'anya', label: 'Anya — Career Advisor' },
        // Phase 4
        clubs: { npc: 'vbabes', label: 'Vbabes — Guild Master' },
        territory: { npc: 'anya', label: 'Anya — Cartographer' },
        // Phase 5
        justice: { npc: 'anya', label: 'Anya — Justice Warden' },
        underworld: { npc: 'vbabes', label: 'Vbabes — Underworld Boss' },
        // Phase 6
        pets: { npc: 'anya', label: 'Anya — Pet Tamer' },
        vehicles: { npc: 'vbabes', label: 'Vbabes — Mechanic' },
        world: { npc: 'anya', label: 'Anya — World Explorer' },
        regions: { npc: 'anya', label: 'Anya — Regional Scout' },
        treasure: { npc: 'cryptoSeraph', label: 'Crypto-Seraph — Treasure Hunter' },
        // Phase 7
        'ai-citizens': { npc: 'anya', label: 'Anya — AI Overseer' },
        theme: { npc: 'vbabes', label: 'Vbabes — Theme Artist' },
        faith: { npc: 'cryptoSeraph', label: 'Crypto-Seraph — High Priest' },
        // Phase 8
        creator: { npc: 'vbabes', label: 'Vbabes — Creator Mentor' },
        launchpad: { npc: 'cryptoSeraph', label: 'Crypto-Seraph — Launch Director' },
        // Phase 9
        infrastructure: { npc: 'cryptoSeraph', label: 'Crypto-Seraph — SysAdmin' },
        gamingos: { npc: 'cryptoSeraph', label: 'Crypto-Seraph — OS Architect' },
        advertising: { npc: 'vbabes', label: 'Vbabes — Ad Agent' },
        governance: { npc: 'anya', label: 'Anya — Governance Clerk' },
        compliance: { npc: 'anya', label: 'Anya — Auditor' },
        'religion-gov': { npc: 'cryptoSeraph', label: 'Crypto-Seraph — Theocrat' },
        // Phase 10
        tournament: { npc: 'vbabes', label: 'Vbabes — Tournament Official' },
        spectate: { npc: 'anya', label: 'Anya — Spectator Guide' },
        // Admin
        admin: { npc: 'cryptoSeraph', label: 'Crypto-Seraph — Admin' },
    };

    // Get placeholder image path for a tab
    function getPlaceholder(tabId) {
        const config = TAB_PLACEHOLDERS[tabId];
        if (!config) return null;
        // Frame 1 is the stable "portrait" for every character (mirrors the original behaviour),
        // built by the same path builder as the deterministic pool.
        return framePath(config.npc, 1);
    }

    // getTabArt deterministically picks a frame for a World Dashboard tab instead of always using
    // the portrait, so a dashboard full of placeholders is varied but reproducible per seed.
    function getTabArt(tabId, seed) {
        const config = TAB_PLACEHOLDERS[tabId];
        if (!config) return null;
        const frames = frameNumbers(config.npc);
        if (!frames.length) return null;
        const idx = fnv1a(String(tabId) + ':' + String(seed == null ? '' : seed)) % frames.length;
        return framePath(config.npc, frames[idx]);
    }

    // Get all placeholders (for bulk operations)
    function getAllPlaceholders() {
        const result = {};
        for (const tabId of Object.keys(TAB_PLACEHOLDERS)) {
            result[tabId] = getPlaceholder(tabId);
        }
        return result;
    }

    // ── LIGHT RENDITIONS (server-derived, placeholder_derivatives.go) ──────────────────────────
    // The server derives a light rendition of every pack frame into
    //   /Assets/Generated/NPC-helpers/<Character>/<Character>-NNN-<tier>.png
    // with a deterministic integer box filter, and reports the real sha256 of what it wrote. This
    // mirrors that ONE naming rule so a client can PREFER the light art and fall back to the shipped
    // frame if the rendition is not there yet — the probe below decides, so a missing file is never
    // painted as a broken image.
    const PACK_URI_RE = /^\/Assets\/Images\/NPC-helpers\/([^/]+)\/([^/]+)-(\d{1,4})\.png$/;

    function lightPathFor(uri, tier) {
        const m = PACK_URI_RE.exec(String(uri || ''));
        if (!m) return '';
        const t = (tier === 'thumb') ? 'thumb' : 'slide';
        return '/Assets/Generated/NPC-helpers/' + m[1] + '/' + m[2] + '-' + m[3] + '-' + t + '.png';
    }

    // Get NPC character name for a tab
    function getNpcName(tabId) {
        const config = TAB_PLACEHOLDERS[tabId];
        return config ? config.npc : null;
    }

    // Get human-readable label
    function getLabel(tabId) {
        const config = TAB_PLACEHOLDERS[tabId];
        return config ? config.label : tabId;
    }

    // Expose
    window.PlaceholderAssets = {
        getPlaceholder,
        getTabArt,
        getAllPlaceholders,
        getNpcName,
        getLabel,
        // Card-view STAND-IN supply (the NPC-helper art pack) + the exact art pool.
        getCardViewFallback,
        getFrames,
        // Light-rendition path mirror (the server's ONE naming rule) — used to prefer the small art.
        lightPathFor,
        hashSlot: fnv1a,
        NPC,
        FRAME_SPECS,
        TAB_PLACEHOLDERS,
    };
})();
