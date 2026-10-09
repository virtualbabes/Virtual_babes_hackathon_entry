// ============================================================================
// equipment_system.js — Card Equipment & Loadout
// ----------------------------------------------------------------------------
// Ported from Triple Triad: card-loadout.js
// Weapons, borders, faceplates that modify card power
// ============================================================================

var API_BASE = '/api';

// Equipment types
const EQUIPMENT_TYPES = [
    { id: 'weapon', name: 'Weapon', icon: '⚔', desc: 'Offense boost to attacks' },
    { id: 'shield', name: 'Shield', icon: '🛡', desc: 'Defense boost when defending' },
    { id: 'charm', name: 'Charm', icon: '✨', desc: 'Special effect trigger' }
];

// Equipment catalog
const EQUIPMENT_CATALOG = [
    { id: 1, type: 'weapon', name: 'Iron Sword', offense: 5, icon: '⚔️', rarity: 'common' },
    { id: 2, type: 'weapon', name: 'Fire Blade', offense: 10, icon: '🗡️', rarity: 'rare' },
    { id: 3, type: 'shield', name: 'Wooden Shield', defense: 5, icon: '🛡️', rarity: 'common' },
    { id: 4, type: 'shield', name: 'Diamond Guard', defense: 15, icon: '💎', rarity: 'epic' },
    { id: 5, type: 'charm', name: 'Lucky Coin', special: '+5% capture chance', icon: '🪙', rarity: 'rare' },
    { id: 6, type: 'charm', name: 'Dark Amulet', special: '+10% vs Fallen', icon: '🔮', rarity: 'epic' }
];

// State
let selectedCard = null;
let equippedItems = {}; // cardId -> { weapon, shield, charm }

// --- Init ---
export function initEquipmentSystem() {
    const el = document.getElementById('wd-equipment');
    if (!el) return;
    renderEquipmentSystem(el);
}

function renderEquipmentSystem(el) {
    el.innerHTML = `
        <div class="equipment-container">
            <div class="eq-header">
                <span class="eq-icon">⚔️</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Equipment Loadout</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Equip weapons, shields, and charms to your cards</p>
                </div>
            </div>

            <!-- Card Selection -->
            <div class="eq-section">
                <h4 style="color:#e5e7eb;">Select Card</h4>
                <div class="eq-card-list" id="eq-card-list">
                    ${renderCardList()}
                </div>
            </div>

            <!-- Equipment Slots -->
            <div class="eq-section">
                <h4 style="color:#e5e7eb;">Equipment Slots</h4>
                <div class="eq-slots" id="eq-slots">
                    ${renderEquipmentSlots()}
                </div>
            </div>

            <!-- Available Equipment -->
            <div class="eq-section">
                <h4 style="color:#e5e7eb;">Available Equipment</h4>
                <div class="eq-catalog" id="eq-catalog">
                    ${renderEquipmentCatalog()}
                </div>
            </div>
        </div>
    `;
    attachEquipmentListeners(el);
}

function renderCardList() {
    const cards = [
        { id: 1, name: 'Alana', tier: 'Bronze' },
        { id: 2, name: 'Bella', tier: 'Bronze' },
        { id: 3, name: 'Ellie', tier: 'Iron' }
    ];
    return cards.map(c => `
        <div class="eq-card ${selectedCard === c.id ? 'selected' : ''}" data-id="${c.id}">
            <span class="eq-card-name">${c.name}</span>
            <span class="eq-card-tier">${c.tier}</span>
        </div>
    `).join('');
}

function renderEquipmentSlots() {
    if (!selectedCard) return '<p style="color:#90a4ae;">Select a card first</p>';
    const equipped = equippedItems[selectedCard] || {};
    return EQUIPMENT_TYPES.map(t => {
        const item = equipped[t.id];
        return `
            <div class="eq-slot" data-type="${t.id}">
                <span class="eq-slot-icon">${t.icon}</span>
                <span class="eq-slot-name">${t.name}</span>
                <span class="eq-slot-item ${item ? '' : 'empty'}">${item ? item.name : 'Empty'}</span>
                ${item ? `<span class="eq-slot-stat">${item.offense ? '+' + item.offense + ' ATK' : item.defense ? '+' + item.defense + ' DEF' : item.special}</span>` : ''}
                ${item ? '<span class="eq-slot-remove" data-type="' + t.id + '">✕</span>' : ''}
            </div>
        `;
    }).join('');
}

function renderEquipmentCatalog() {
    return EQUIPMENT_CATALOG.map(e => `
        <div class="eq-item rarity-${e.rarity}" data-id="${e.id}" data-type="${e.type}">
            <span class="eq-item-icon">${e.icon}</span>
            <span class="eq-item-name">${e.name}</span>
            <span class="eq-item-stat">${e.offense ? '+' + e.offense + ' ATK' : e.defense ? '+' + e.defense + ' DEF' : e.special}</span>
            <span class="eq-item-rarity">${e.rarity}</span>
        </div>
    `).join('');
}

function attachEquipmentListeners(el) {
    // Card selection
    el.querySelectorAll('.eq-card').forEach(card => {
        card.addEventListener('click', () => {
            selectedCard = parseInt(card.dataset.id);
            renderEquipmentSystem(el);
        });
    });

    // Equipment catalog click
    el.querySelectorAll('.eq-item').forEach(item => {
        item.addEventListener('click', () => {
            if (!selectedCard) {
                if (window.showToast) window.showToast('Select a card first', 'warning');
                return;
            }
            const itemId = parseInt(item.dataset.id);
            const itemType = item.dataset.type;
            const catalogItem = EQUIPMENT_CATALOG.find(e => e.id === itemId);
            
            if (!equippedItems[selectedCard]) equippedItems[selectedCard] = {};
            equippedItems[selectedCard][itemType] = catalogItem;
            renderEquipmentSystem(el);
            if (window.showToast) window.showToast(`Equipped ${catalogItem.name}!`, 'success');
        });
    });

    // Unequip
    el.querySelectorAll('.eq-slot-remove').forEach(btn => {
        btn.addEventListener('click', () => {
            if (!selectedCard) return;
            const type = btn.dataset.type;
            if (equippedItems[selectedCard]) {
                delete equippedItems[selectedCard][type];
                renderEquipmentSystem(el);
            }
        });
    });
}

// --- Globals ---
window.initEquipmentSystem = initEquipmentSystem;
