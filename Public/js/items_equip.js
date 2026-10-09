// ============================================================================
// items_equip.js — Item Shop & Equipment Manager
// ----------------------------------------------------------------------------
// Exposes item_shop_archetype.go: equip/unequip, build, bind NFT, categories
// Unique visual style: inventory grid / equipment slots with rarity glows
// ============================================================================

var API_BASE = '/api';

// --- Item Categories ---
const ITEM_CATEGORIES = [
    { id: 'elemental', name: 'Elemental', icon: '🔥' },
    { id: 'tactical', name: 'Tactical', icon: '⚔️' },
    { id: 'vitality', name: 'Vitality', icon: '❤️' },
    { id: 'intelligence', name: 'Intel', icon: '🧠' },
    { id: 'hardware', name: 'Hardware', icon: '🔧' },
    { id: 'justice', name: 'Justice', icon: '⚖️' },
    { id: 'underworld', name: 'Underworld', icon: '🦹' },
];

// --- Equipment Slots ---
const EQUIP_SLOTS = [
    { id: 'weapon', name: 'Weapon', icon: '⚔️' },
    { id: 'armor', name: 'Armor', icon: '🛡️' },
    { id: 'accessory', name: 'Accessory', icon: '💍' },
    { id: 'consumable', name: 'Consumable', icon: '🧪' },
];

// --- State ---
let itemCollection = [];
let equippedItems = {};
let selectedCategory = 'all';

// --- Init ---
export function initItemsEquip() {
    const el = document.getElementById('wd-items');
    if (!el) return;
    renderItemsLoading(el);
    fetchItemsData(el);
}

async function fetchItemsData(el) {
    try {
        const resp = await fetch(`${API_BASE}/items/archetypes`);
        if (!resp.ok) throw new Error('Failed to fetch items');
        const data = await resp.json();
        itemCollection = data.archetypes || data.items || [];
        renderItemsEquip(el);
    } catch (e) {
        el.innerHTML = '<p style="color:#90a4ae;">Item data unavailable</p>';
    }
}

function renderItemsLoading(el) {
    el.innerHTML = '<div class="wd-loading">Loading inventory…</div>';
}

function renderItemsEquip(el) {
    const filtered = selectedCategory === 'all'
        ? itemCollection
        : itemCollection.filter(i => i.category === selectedCategory);

    let html = `
        <div class="items-equip-container">
            <!-- Inventory Header -->
            <div class="inventory-header">
                <div class="inventory-icon">🎒</div>
                <div class="inventory-info">
                    <h3 style="color:#e5e7eb;margin:0;">Equipment Manager</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Equip items to boost your power</p>
                </div>
                <div class="inventory-count">
                    <span class="inventory-count-num">${itemCollection.length}</span>
                    <span class="inventory-count-label">Items</span>
                </div>
            </div>

            <!-- Equipment Slots. NO engine route equips an item (server_main.go registers
                 /api/items/archetypes, registry, collection, build and bind-nft — there is no
                 "equip"). These four slots are therefore NOT a state view and the equippedItems
                 map is never written by anything; they are shown as a stated placeholder rather
                 than as a capability the app does not have. -->
            <div class="equip-slots-section">
                <h4 style="color:#e5e7eb;">🎽 Equipment Slots</h4>
                <p class="forge-note">Equipping is not served yet — the engine holds no equip route, so these slots report no state.</p>
                <div class="equip-slots-grid">
                    ${EQUIP_SLOTS.map(slot => renderEquipSlot(slot)).join('')}
                </div>
            </div>

            <!-- Category Filter -->
            <div class="item-categories">
                <button class="item-cat-btn ${selectedCategory === 'all' ? 'active' : ''}" data-cat="all">All</button>
                ${ITEM_CATEGORIES.map(c => `
                    <button class="item-cat-btn ${selectedCategory === c.id ? 'active' : ''}" data-cat="${c.id}">
                        ${c.icon} ${c.name}
                    </button>
                `).join('')}
            </div>

            <!-- Item Grid -->
            <div class="item-grid-section">
                <h4 style="color:#e5e7eb;">📦 Archetype Catalogue</h4>
                <div class="item-grid">
                    ${filtered.length > 0 ? filtered.map(i => renderItemCard(i)).join('') : renderNoArchetypes()}
                </div>
            </div>

            <!-- Forge & Bind. Two engine capabilities that had NO owner anywhere in the client:
                 POST /api/items/build (BuildItem: base archetype + paid_micro, fee routed through
                 the one sink door) and POST /api/items/bind-nft (BindNFT: tie a held item to an
                 NFT). They were reachable only from an uncomposed orphan-closure panel, so in the
                 app they did not exist. Placed here because building and bonding an ITEM is this
                 leaf's domain — the same relationship the Kennel has to pets. -->
            <div class="items-forge-section">
                <h4 style="color:#e5e7eb;">🔨 Forge &amp; Bond</h4>
                <div class="forge-row">
                    <input id="forge-base-id" placeholder="Base item id (e.g. ITEM-…)" />
                    <input id="forge-paid" placeholder="Paid (micro-VBV)" type="number" />
                    <input id="forge-bond" placeholder="Bonded NFT id (optional)" />
                    <button class="vbt-btn vbt-btn-primary" id="forge-build-btn">Build Item</button>
                </div>
                <div class="forge-row">
                    <input id="forge-item-id" placeholder="Built item id" />
                    <input id="forge-nft-id" placeholder="Bonded NFT id" />
                    <button class="vbt-btn vbt-btn-secondary" id="forge-bind-btn">Bind NFT</button>
                </div>
                <p class="forge-note" id="forge-status"></p>
            </div>
        </div>
    `;

    el.innerHTML = html;
    attachItemsListeners(el);
}

function renderEquipSlot(slot) {
    const equipped = equippedItems[slot.id];
    return `
        <div class="equip-slot ${equipped ? 'slot-filled' : 'slot-empty'}" data-slot="${slot.id}">
            ${equipped ? `
                <div class="equip-slot-item">
                    <span class="equip-slot-icon">${equipped.icon || '📦'}</span>
                    <span class="equip-slot-name">${escapeHtml(equipped.name)}</span>
                </div>
            ` : `
                <div class="equip-slot-placeholder">
                    <span class="equip-slot-icon">${slot.icon}</span>
                    <span class="equip-slot-label">${slot.name}</span>
                </div>
            `}
        </div>
    `;
}

function renderItemCard(item) {
    const rarityColors = { common: '#9e9e9e', rare: '#2196f3', epic: '#9c27b0', legendary: '#ff9800' };
    const color = rarityColors[item.rarity] || '#9e9e9e';

    return `
        <div class="item-card" data-item="${item.id || item.name}" style="--item-color:${color}">
            <div class="item-card-header">
                <span class="item-icon">${item.icon || '📦'}</span>
                <span class="item-rarity" style="color:${color}">${(item.rarity || 'common').toUpperCase()}</span>
            </div>
            <div class="item-card-body">
                <span class="item-name">${escapeHtml(item.name)}</span>
                <span class="item-effect">${escapeHtml(item.effect || item.description || '')}</span>
            </div>
            <button class="vbt-btn vbt-btn-primary item-detail-btn">Details</button>
        </div>
    `;
}

function renderNoArchetypes() {
    // An empty catalogue is STATED. The previous `renderSampleItems()` invented three items
    // ("Mood Catalyst", "Ghost Protocol", "Mutation Insurance") with effects and rarities that no
    // endpoint reports, so a 404 and a genuinely empty shop looked identical — and a player was
    // shown a shop that does not exist. Never fill a failed read with invented data.
    return '<p class="forge-note">No item archetypes were reported by the engine.</p>';
}

function attachItemsListeners(el) {
    // Category filter
    el.querySelectorAll('.item-cat-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            selectedCategory = btn.dataset.cat;
            renderItemsEquip(el);
        });
    });

    // Item detail. There is NO /api/items/equip route (server_main.go registers /api/items/
    // archetypes, registry, collection, build and bind-nft only), so the old button toasted
    // "Equipped: X" while sending NOTHING — a receipt for a request that was never made, and the
    // `equippedItems` map it implied was client-only state with no server behind it. The control
    // now reports what the SERVER said about the archetype.
    el.querySelectorAll('.item-detail-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const card = btn.closest('.item-card');
            const id = card?.dataset.item;
            const item = itemCollection.find(i => (i.id || i.name) === id);
            if (!item) return;
            if (window.showToast) window.showToast('Server record: ' + JSON.stringify(item).slice(0, 240), 'success');
        });
    });

    const forgeStatus = (msg) => { const p = document.getElementById('forge-status'); if (p) p.textContent = msg || ''; };

    const buildBtn = document.getElementById('forge-build-btn');
    if (buildBtn) buildBtn.addEventListener('click', async () => {
        const wallet = window.currentWallet || (window.getActiveWallet && window.getActiveWallet()) || '';
        const baseId = document.getElementById('forge-base-id')?.value.trim();
        if (!wallet) { forgeStatus('Connect a wallet first — the fee is charged to it.'); return; }
        if (!baseId) { forgeStatus('A base item id is required.'); return; }
        const paid = Number(document.getElementById('forge-paid')?.value || 0);
        const bonded = document.getElementById('forge-bond')?.value.trim() || '';
        try {
            // paid_micro is an integer micro-VBV amount (Architecture Ledger: no float money).
            const res = await api('/items/build', {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ wallet, base_item_id: baseId, paid_micro: Math.max(0, Math.trunc(paid)), bonded_nft: bonded }),
            });
            forgeStatus('Built: ' + (res.data?.ItemID || res.data?.item_id || 'ok'));
            fetchItemsData(el);
        } catch (e) { forgeStatus('Build refused: ' + e.message); }
    });

    const bindBtn = document.getElementById('forge-bind-btn');
    if (bindBtn) bindBtn.addEventListener('click', async () => {
        const itemId = document.getElementById('forge-item-id')?.value.trim();
        const nftId = document.getElementById('forge-nft-id')?.value.trim();
        if (!itemId || !nftId) { forgeStatus('Both the item id and the bonded NFT id are required.'); return; }
        try {
            await api('/items/bind-nft', {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ item_id: itemId, bonded_nft: nftId }),
            });
            forgeStatus('Bound ' + itemId + ' → ' + nftId);
        } catch (e) { forgeStatus('Bind refused: ' + e.message); }
    });
}

// fetch uses API_BASE = '/api'; `api()` below prefixes only when the path is not already absolute.
async function api(path, opts) {
    const url = path.startsWith('/api/') ? path : API_BASE + path;
    const resp = await fetch(url, opts);
    if (!resp.ok) { let d = ''; try { d = (await resp.json()).error || ''; } catch (e) {} throw new Error(d || ('HTTP ' + resp.status)); }
    return resp.json();
}

// --- Global handlers ---
window.initItemsEquip = initItemsEquip;

// --- Utility ---
function escapeHtml(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    }[c]));
}
