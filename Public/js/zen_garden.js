// ============================================================================
// zen_garden.js — Zen Garden & Sanctuary
// ----------------------------------------------------------------------------
// Ported from Triple Triad: zen-garden.js + sanctuary.js
// Visual meditation space, achievement visualization, faith enhancement
// Connects to: Dividend Yield, Faith System, Achievement Progress
// ============================================================================

var API_BASE = '/api';

// Garden elements
const GARDEN_ELEMENTS = [
    { id: 'stone', name: 'Stone', icon: '🪨', desc: 'Stability +5%', cost: 100, effect: 'stability' },
    { id: 'water', name: 'Water Feature', icon: '💧', desc: 'Flow +10%', cost: 250, effect: 'flow' },
    { id: 'tree', name: 'Bonsai Tree', icon: '🌳', desc: 'Growth +15%', cost: 500, effect: 'growth' },
    { id: 'lantern', name: 'Lantern', icon: '🏮', desc: 'Light +20%', cost: 750, effect: 'light' },
    { id: 'bridge', name: 'Bridge', icon: '🌉', desc: 'Connection +25%', cost: 1000, effect: 'connection' },
    { id: 'koi', name: 'Koi Pond', icon: '🐟', desc: 'Prosperity +30%', cost: 1500, effect: 'prosperity' }
];

// State
let gardenElements = [];
let gardenLevel = 1;
let meditationStreak = 0;
let lastMeditation = null;

// --- Init ---
export function initZenGarden() {
    const el = document.getElementById('wd-zen_garden');
    if (!el) return;
    renderZenGarden(el);
}

function renderZenGarden(el) {
    el.innerHTML = `
        <div class="zen-garden-container">
            <div class="zen-header">
                <span class="zen-icon">🧘</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Zen Garden</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">A sanctuary for meditation and reflection</p>
                </div>
                <div class="zen-level">
                    <span class="zen-level-label">Level</span>
                    <span class="zen-level-value">${gardenLevel}</span>
                </div>
            </div>

            <!-- Garden Display -->
            <div class="zen-section">
                <h4 style="color:#e5e7eb;">Your Garden</h4>
                <div class="zen-garden-display" id="zen-garden-display">
                    ${renderGardenDisplay()}
                </div>
            </div>

            <!-- Garden Elements -->
            <div class="zen-section">
                <h4 style="color:#e5e7eb;">Add Elements</h4>
                <div class="zen-elements">
                    ${renderElementCatalog()}
                </div>
            </div>

            <!-- Meditation -->
            <div class="zen-section">
                <h4 style="color:#e5e7eb;">Meditation</h4>
                <div class="zen-meditation">
                    <div class="zen-meditation-stats">
                        <span class="zen-streak">🔥 Streak: ${meditationStreak} days</span>
                        <span class="zen-last">${lastMeditation ? `Last: ${lastMeditation}` : 'Not yet meditated'}</span>
                    </div>
                    <button class="vbt-btn vbt-btn-primary zen-meditate-btn">🧘 Meditate Now</button>
                </div>
            </div>

            <!-- Garden Benefits -->
            <div class="zen-section">
                <h4 style="color:#e5e7eb;">Garden Benefits</h4>
                <div class="zen-benefits">
                    ${renderGardenBenefits()}
                </div>
            </div>
        </div>
    `;
    attachZenListeners(el);
}

function renderGardenDisplay() {
    if (gardenElements.length === 0) {
        return '<p style="color:#90a4ae;text-align:center;padding:20px;">Your garden is empty. Add elements below to begin.</p>';
    }
    return `
        <div class="zen-garden-grid">
            ${gardenElements.map(e => `
                <div class="zen-element" data-id="${e.id}">
                    <span class="zen-element-icon">${e.icon}</span>
                    <span class="zen-element-name">${e.name}</span>
                </div>
            `).join('')}
        </div>
    `;
}

function renderElementCatalog() {
    return GARDEN_ELEMENTS.map(e => {
        const owned = gardenElements.find(ge => ge.id === e.id);
        return `
            <div class="zen-element-card ${owned ? 'owned' : ''}" data-id="${e.id}">
                <span class="zen-element-icon">${e.icon}</span>
                <span class="zen-element-name">${e.name}</span>
                <span class="zen-element-desc">${e.desc}</span>
                <span class="zen-element-cost">${e.cost} SP</span>
                <button class="vbt-btn vbt-btn-secondary zen-add-btn" data-id="${e.id}" ${owned ? 'disabled' : ''}>${owned ? 'Owned' : 'Add'}</button>
            </div>
        `;
    }).join('');
}

function renderGardenBenefits() {
    const benefits = [];
    if (gardenElements.length >= 1) benefits.push({ icon: '✨', text: '+5% focus in matches' });
    if (gardenElements.length >= 3) benefits.push({ icon: '🛡', text: '+10% defense' });
    if (gardenElements.length >= 5) benefits.push({ icon: '⚡', text: '+15% power' });
    if (meditationStreak >= 7) benefits.push({ icon: '🔥', text: 'Meditation master: +20% all stats' });
    
    return benefits.length > 0 ? benefits.map(b => `
        <div class="zen-benefit">
            <span class="zen-benefit-icon">${b.icon}</span>
            <span class="zen-benefit-text">${b.text}</span>
        </div>
    `).join('') : '<p style="color:#90a4ae;">Add elements and meditate to unlock benefits</p>';
}

function attachZenListeners(el) {
    // Add element buttons
    el.querySelectorAll('.zen-add-btn:not([disabled])').forEach(btn => {
        btn.addEventListener('click', () => {
            const elementId = btn.dataset.id;
            const element = GARDEN_ELEMENTS.find(e => e.id === elementId);
            if (element && !gardenElements.find(e => e.id === elementId)) {
                gardenElements.push(element);
                gardenLevel = Math.floor(gardenElements.length / 2) + 1;
                renderZenGarden(el);
                if (window.showToast) window.showToast(`Added ${element.name} to garden!`, 'success');
            }
        });
    });

    // Meditate button
    const meditateBtn = el.querySelector('.zen-meditate-btn');
    if (meditateBtn) {
        meditateBtn.addEventListener('click', () => {
            meditationStreak++;
            lastMeditation = new Date().toLocaleDateString();
            renderZenGarden(el);
            if (window.showToast) window.showToast('🧘 Meditation complete. Peace flows through you.', 'success');
        });
    }
}

// --- Globals ---
window.initZenGarden = initZenGarden;
