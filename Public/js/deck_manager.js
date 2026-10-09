// ============================================================================
// deck_manager.js — Deck Builder & Card Collection Manager
// ----------------------------------------------------------------------------
// Ported from Triple Triad: deck-manager.js + card-loadout.js
// Full deck construction: select 5 cards, save presets, sell cards, filter/sort
// ============================================================================

var API_BASE = '/api';

// State
let collection = [];
let currentDeck = [];
let activeDeckSlot = 0;
let sellMode = false;
let markedForSell = [];
let filterRank = 'all';
let filterMood = 'all';
let currentSort = 'rank';

// --- Init ---
export function initDeckManager() {
    const el = document.getElementById('wd-deck');
    if (!el) return;
    renderDeckManager(el);
}

function renderDeckManager(el) {
    el.innerHTML = `
        <div class="deck-manager-container">
            <div class="dm-header">
                <span class="dm-icon">🃏</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Deck Manager</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Build and manage your decks</p>
                </div>
                <div class="dm-deck-slots">
                    ${[0,1,2,3].map(i => `
                        <div class="dm-slot ${activeDeckSlot === i ? 'active' : ''}" data-slot="${i}">
                            Deck ${i+1}
                        </div>
                    `).join('')}
                </div>
            </div>

            <!-- Current Deck -->
            <div class="dm-section">
                <h4 style="color:#e5e7eb;">Current Deck (${currentDeck.length}/5)</h4>
                <div class="dm-current-deck" id="dm-current-deck">
                    ${currentDeck.length > 0 ? currentDeck.map(c => renderCardSlot(c, true)).join('') : '<p style="color:#90a4ae;">Select cards below to build your deck</p>'}
                </div>
                <div class="dm-deck-actions">
                    <button class="vbt-btn vbt-btn-primary" id="dm-save-btn">💾 Save Deck</button>
                    <button class="vbt-btn vbt-btn-secondary" id="dm-clear-btn">🗑️ Clear</button>
                </div>
            </div>

            <!-- Filters -->
            <div class="dm-section">
                <h4 style="color:#e5e7eb;">Filters</h4>
                <div class="dm-filters">
                    <select id="dm-filter-rank">
                        <option value="all">All Ranks</option>
                        <option value="Iron">Iron</option>
                        <option value="Bronze">Bronze</option>
                        <option value="Gold">Gold</option>
                        <option value="Diamond">Diamond</option>
                    </select>
                    <select id="dm-filter-mood">
                        <option value="all">All Moods</option>
                        <option value="Volatile">Volatile</option>
                        <option value="Serene">Serene</option>
                        <option value="Spirited">Spirited</option>
                        <option value="Grounded">Grounded</option>
                        <option value="Neutral">Neutral</option>
                    </select>
                    <select id="dm-sort-by">
                        <option value="rank">Sort by Rank</option>
                        <option value="power">Sort by Power</option>
                        <option value="level">Sort by Level</option>
                    </select>
                </div>
            </div>

            <!-- Collection -->
            <div class="dm-section">
                <h4 style="color:#e5e7eb;">Your Collection (${collection.length} cards)</h4>
                <div class="dm-collection" id="dm-collection">
                    ${collection.length > 0 ? getFilteredCollection().map(c => renderCardSlot(c, false)).join('') : renderSampleCollection()}
                </div>
            </div>
        </div>
    `;
    attachDeckListeners(el);
}

function renderCardSlot(card, inDeck) {
    const power = (card.power?.reduce((a,b) => a+b, 0)) || card.Power?.reduce((a,b) => a+b, 0) || 0;
    return `
        <div class="dm-card ${inDeck ? 'in-deck' : ''}" data-id="${card.id}" data-name="${card.name}" data-mood="${card.mood || card.Mood || 'Neutral'}" data-tier="${card.tier || card.Tier || 'Iron'}" data-level="${card.level || card.Level || 1}" data-power="${power}">
            <span class="dm-card-icon">${getMoodIcon(card.mood || card.Mood)}</span>
            <span class="dm-card-name">${card.name}</span>
            <span class="dm-card-tier tier-${(card.tier || card.Tier || 'Iron').toLowerCase()}">${card.tier || card.Tier || 'Iron'}</span>
            <span class="dm-card-power">${power}</span>
            ${inDeck ? '<span class="dm-card-remove">✕</span>' : '<span class="dm-card-add">+</span>'}
        </div>
    `;
}

function renderSampleCollection() {
    const samples = [
        { id: 1, name: 'Alana', power: [3,5,4,2], Tier: 'Bronze', Mood: 'Serene', Level: 1 },
        { id: 2, name: 'Bella', power: [4,4,4,4], Tier: 'Bronze', Mood: 'Spirited', Level: 1 },
        { id: 3, name: 'Ellie', power: [5,3,3,5], Tier: 'Iron', Mood: 'Volatile', Level: 1 },
        { id: 4, name: 'Karren', power: [2,6,5,3], Tier: 'Iron', Mood: 'Grounded', Level: 1 },
        { id: 5, name: 'Lucy', power: [4,4,5,4], Tier: 'Bronze', Mood: 'Neutral', Level: 1 },
    ];
    return samples.map(c => renderCardSlot(c, false)).join('');
}

function getMoodIcon(mood) {
    const icons = { Volatile: '🔥', Serene: '💧', Spirited: '⚡', Grounded: '🌿', Neutral: '⚪' };
    return icons[mood] || '⚪';
}

function getFilteredCollection() {
    let filtered = [...collection];
    if (filterRank !== 'all') filtered = filtered.filter(c => (c.tier || c.Tier) === filterRank);
    if (filterMood !== 'all') filtered = filtered.filter(c => (c.mood || c.Mood) === filterMood);
    if (currentSort === 'power') filtered.sort((a, b) => (b.power?.[0] || 0) - (a.power?.[0] || 0));
    if (currentSort === 'level') filtered.sort((a, b) => (b.level || b.Level || 1) - (a.level || a.Level || 1));
    return filtered;
}

function attachDeckListeners(el) {
    // Deck slot selection
    el.querySelectorAll('.dm-slot').forEach(slot => {
        slot.addEventListener('click', () => {
            activeDeckSlot = parseInt(slot.dataset.slot);
            renderDeckManager(el);
        });
    });

    // Add card to deck
    el.querySelectorAll('.dm-card:not(.in-deck) .dm-card-add').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const cardEl = btn.closest('.dm-card');
            if (currentDeck.length >= 5) {
                if (window.showToast) window.showToast('Deck is full (5 cards max)', 'warning');
                return;
            }
            const card = {
                id: parseInt(cardEl.dataset.id),
                name: cardEl.dataset.name,
                mood: cardEl.dataset.mood,
                tier: cardEl.dataset.tier,
                level: parseInt(cardEl.dataset.level)
            };
            currentDeck.push(card);
            renderDeckManager(el);
        });
    });

    // Remove card from deck
    el.querySelectorAll('.dm-card.in-deck .dm-card-remove').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const cardEl = btn.closest('.dm-card');
            const cardId = parseInt(cardEl.dataset.id);
            currentDeck = currentDeck.filter(c => c.id !== cardId);
            renderDeckManager(el);
        });
    });

    // Filters
    const rankSelect = el.querySelector('#dm-filter-rank');
    const moodSelect = el.querySelector('#dm-filter-mood');
    const sortSelect = el.querySelector('#dm-sort-by');
    if (rankSelect) rankSelect.addEventListener('change', () => { filterRank = rankSelect.value; renderDeckManager(el); });
    if (moodSelect) moodSelect.addEventListener('change', () => { filterMood = moodSelect.value; renderDeckManager(el); });
    if (sortSelect) sortSelect.addEventListener('change', () => { currentSort = sortSelect.value; renderDeckManager(el); });

    // Save & Clear
    const saveBtn = el.querySelector('#dm-save-btn');
    const clearBtn = el.querySelector('#dm-clear-btn');
    if (saveBtn) saveBtn.addEventListener('click', () => {
        if (window.showToast) window.showToast(`💾 Deck ${activeDeckSlot + 1} saved!`, 'success');
    });
    if (clearBtn) clearBtn.addEventListener('click', () => {
        currentDeck = [];
        renderDeckManager(el);
    });
}

// --- Globals ---
window.initDeckManager = initDeckManager;
