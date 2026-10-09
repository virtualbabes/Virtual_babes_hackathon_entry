// ============================================================================
// tea_house.js — Tea House Club & Divination
// ----------------------------------------------------------------------------
// Ported from Triple Triad: tea-house.js
// Gourmet Chef: recipes that provide buffs
// Tea Leaf Divination: random hints about NPCs and events
// Connects to: Club Foundry, Character Mood, Social Hub
// ============================================================================

var API_BASE = '/api';

// Gourmet recipes (provide buffs)
const RECIPES = [
    { id: 'green_tea', name: 'Green Tea', icon: '🍵', desc: 'Calming blend', buff: '+5% focus', cost: 100, rarity: 'common' },
    { id: 'chai', name: 'Spiced Chai', icon: '☕', desc: 'Energizing spices', buff: '+10% speed', cost: 250, rarity: 'rare' },
    { id: 'matcha', name: 'Matcha Latte', icon: '🥛', desc: 'Pure energy', buff: '+15% power', cost: 500, rarity: 'epic' },
    { id: 'herbal', name: 'Herbal Infusion', icon: '🌿', desc: 'Healing herbs', buff: '+20% defense', cost: 750, rarity: 'epic' },
    { id: 'golden', name: 'Golden Oolong', icon: '✨', desc: 'Legendary blend', buff: '+25% all stats', cost: 1500, rarity: 'legendary' }
];

// Divination hints
const DIVINATION_HINTS = [
    { text: 'A challenger approaches from the east...', weight: 10, type: 'warning' },
    { text: 'Fortune favors your next match.', weight: 8, type: 'positive' },
    { text: 'Beware the one with many bounties.', weight: 6, type: 'danger' },
    { text: 'A new title awaits the worthy.', weight: 5, type: 'achievement' },
    { text: 'The cards align in your favor today.', weight: 7, type: 'positive' },
    { text: 'A rival plots in the shadows.', weight: 4, type: 'danger' },
    { text: 'Your faith shall be rewarded.', weight: 6, type: 'faith' },
    { text: 'The underground stirs with activity.', weight: 5, type: 'underworld' }
];

// State
let selectedRecipe = null;
let lastDivination = null;
let playerRecipes = [];

// --- Init ---
export function initTeaHouse() {
    const el = document.getElementById('wd-tea_house');
    if (!el) return;
    renderTeaHouse(el);
}

function renderTeaHouse(el) {
    el.innerHTML = `
        <div class="tea-house-container">
            <div class="tea-header">
                <span class="tea-icon">🍵</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Tea House</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">A tranquil oasis for mind and spirit</p>
                </div>
            </div>

            <!-- Tabs -->
            <div class="tea-tabs">
                <button class="tea-tab active" data-tab="chef">🍳 Gourmet Chef</button>
                <button class="tea-tab" data-tab="divination">🔮 Tea Leaf Divination</button>
            </div>

            <!-- Gourmet Chef Tab -->
            <div class="tea-panel active" id="tea-panel-chef">
                <h4 style="color:#e5e7eb;">Gourmet Chef</h4>
                <p style="color:#90a4ae;font-size:11px;">Brew recipes to gain temporary buffs</p>
                <div class="tea-recipes">
                    ${renderRecipes()}
                </div>
            </div>

            <!-- Divination Tab -->
            <div class="tea-panel" id="tea-panel-divination">
                <h4 style="color:#e5e7eb;">Tea Leaf Divination</h4>
                <p style="color:#90a4ae;font-size:11px;">Read the leaves for hints about your future</p>
                <div class="tea-divination">
                    <div class="tea-divination-display" id="tea-divination-display">
                        ${lastDivination ? `
                            <p style="color:#e5e7eb;font-size:12px;font-style:italic;">"${lastDivination.text}"</p>
                            <span class="tea-hint-type ${lastDivination.type}">${lastDivination.type}</span>
                        ` : '<p style="color:#90a4ae;">Click below to read the leaves...</p>'}
                    </div>
                    <button class="vbt-btn vbt-btn-primary tea-divinate-btn">🔮 Read the Leaves (500 SP)</button>
                </div>
            </div>
        </div>
    `;
    attachTeaListeners(el);
}

function renderRecipes() {
    return RECIPES.map(r => `
        <div class="tea-recipe rarity-${r.rarity}" data-id="${r.id}">
            <span class="tea-recipe-icon">${r.icon}</span>
            <span class="tea-recipe-name">${r.name}</span>
            <span class="tea-recipe-desc">${r.desc}</span>
            <span class="tea-recipe-buff">${r.buff}</span>
            <span class="tea-recipe-cost">${r.cost} SP</span>
            <button class="vbt-btn vbt-btn-secondary tea-brew-btn" data-id="${r.id}">Brew</button>
        </div>
    `).join('');
}

function attachTeaListeners(el) {
    // Tab switching
    el.querySelectorAll('.tea-tab').forEach(tab => {
        tab.addEventListener('click', () => {
            el.querySelectorAll('.tea-tab').forEach(t => t.classList.remove('active'));
            el.querySelectorAll('.tea-panel').forEach(p => p.classList.remove('active'));
            tab.classList.add('active');
            document.getElementById(`tea-panel-${tab.dataset.tab}`).classList.add('active');
        });
    });

    // Brew buttons
    el.querySelectorAll('.tea-brew-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const recipeId = btn.dataset.id;
            const recipe = RECIPES.find(r => r.id === recipeId);
            if (window.showToast) window.showToast(`🍵 Brewed ${recipe.name}! ${recipe.buff}`, 'success');
        });
    });

    // Divination
    const divBtn = el.querySelector('.tea-divinate-btn');
    if (divBtn) {
        divBtn.addEventListener('click', () => {
            const hint = DIVINATION_HINTS[Math.floor(Math.random() * DIVINATION_HINTS.length)];
            lastDivination = hint;
            renderTeaHouse(el);
            if (window.showToast) window.showToast('🔮 The leaves have spoken...', 'info');
        });
    }
}

// --- Globals ---
window.initTeaHouse = initTeaHouse;
