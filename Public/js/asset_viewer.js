// Asset Viewer (§23.5) — presentation-layer asset gallery. Every UI skin/appearance
// is an NFT (§23.5). This browses the real Public/Assets catalog with verified paths.
(function () {
    'use strict';

    // Verified catalog (paths confirmed present in Public/Assets).
    const CATALOG = [
        { cat: 'Cards', folder: 'Images/Cards', samples: ['Alana.webp', 'Bella.webp', 'Clohey.webp', 'Ellie.webp', 'Fran.webp'], note: 'Character / profile cards (NFT-backed).' },
        { cat: 'Cosmetics', folder: 'Images/Cosmetics', samples: ['faceplate_governor.webp', 'faceplate_neon_vibe.webp', 'faceplate_shadow.webp'], note: 'Faceplate / skin cosmetics (NFT).' },
        { cat: 'Effects', folder: 'Images/Effects', samples: ['Biological-corruption.webp', 'Buffer-overflow.webp', 'Core-dissolution.webp', 'Crystalline-overgrowth.webp', 'Cybernetic-fracture.webp'], note: 'Visual effect textures (NFT).' },
        { cat: 'Items', folder: 'Images/Items', samples: ['bio_guard_dog.webp', 'laser_tripwire.webp', 'sentry_turret.webp'], note: 'Item art (bonded NFT per §14).' },
        { cat: 'Textures', folder: 'Textures', samples: ['arena_floor.png', 'arena_floor_challenge.png', 'arena_floor_tournament.png', 'arena_floor_tournament_final.png', 'arena_floor_tournament_semi_final.png'], note: 'World textures (NFT).' },
        { cat: 'Audio', folder: 'Audio', samples: ['click.mp3', 'Connected.mp3', 'Challenge_accepted.mp3', 'ambient_menu_music_1.mp3'], note: 'Audio cues & ambience (NFT).' }
    ];

    let state = { _init: false };
    let overlayEl = null, gridEl = null, statusEl = null;

    function init() {
        if (state._init) return;
        state._init = true;
        const html = `
<div id="asset-viewer-overlay" class="overlay vbt-overlay" style="display:none;">
    <div class="neon-glass-panel asset-viewer-panel">
        <button id="btn-close-assets" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
        <h2>🎨 Asset Viewer (§23.5)</h2>

        <div class="av-tabs">
            <button class="av-tab active" data-tab="cards" onclick="window.switchAVTab('cards')">Cards</button>
            <button class="av-tab" data-tab="cosmetics" onclick="window.switchAVTab('cosmetics')">Cosmetics</button>
            <button class="av-tab" data-tab="effects" onclick="window.switchAVTab('effects')">Effects</button>
            <button class="av-tab" data-tab="items" onclick="window.switchAVTab('items')">Items</button>
            <button class="av-tab" data-tab="textures" onclick="window.switchAVTab('textures')">Textures</button>
            <button class="av-tab" data-tab="audio" onclick="window.switchAVTab('audio')">Audio</button>
        </div>

        <div id="asset-catalog" class="av-grid"></div>
        <p id="av-status" class="ai-status"></p>
    </div>
</div>`;
        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('asset-viewer-overlay');
        gridEl = document.getElementById('asset-catalog');
        statusEl = document.getElementById('av-status');
        document.getElementById('btn-close-assets').addEventListener('click', () => { overlayEl.style.display = 'none'; });
    }

    function esc(s) { return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }

    function render(activeCat) {
        if (!gridEl) return;
        const cats = activeCat ? CATALOG.filter(c => c.cat.toLowerCase() === activeCat) : CATALOG;
        gridEl.innerHTML = cats.map(c => {
            const thumbs = c.samples.map(s => {
                const path = 'Assets/' + c.folder + '/' + s;
                if (/\.(mp3|wav|ogg)$/i.test(s)) {
                    return `<div class="av-thumb av-audio"><span>🔊</span><small>${esc(s)}</small><audio controls src="${esc(path)}" preload="none"></audio></div>`;
                }
                return `<div class="av-thumb"><img src="${esc(path)}" alt="${esc(s)}" loading="lazy" /><small>${esc(s)}</small></div>`;
            }).join('');
            return `<section class="av-cat"><h3>${esc(c.cat)}</h3><p class="av-note">${esc(c.note)}</p><div class="av-thumbs">${thumbs}</div></section>`;
        }).join('');
        if (statusEl) statusEl.textContent = cats?.length ?? 0 + ' asset categories loaded from Public/Assets.';
    }

    function switchAVTab(tab) {
        document.querySelectorAll('.av-tab').forEach(t => t.classList.toggle('active', t.dataset.tab === tab));
        render(tab);
    }

    window.openAssetViewer = function () {
        init();
        if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
        overlayEl.style.display = 'flex';
        render('cards');
    };
    window.switchAVTab = switchAVTab;
})();
