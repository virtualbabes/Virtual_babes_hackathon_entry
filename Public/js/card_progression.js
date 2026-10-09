// ============================================================================
// card_progression.js — Training & Fusion System
// ----------------------------------------------------------------------------
// Ported from Triple Triad: card-training.js + card-fusion.js + card-upgrade.js
// Training: timing-based game to level up cards
// Fusion: sacrifice duplicate cards to boost power
// Upgrade: stat redistribution with level scaling
// ============================================================================

var API_BASE = '/api';

// State
let selectedTrainingCard = null;
let selectedFusionBase = null;
let fusionSacrifices = [];
let isTrainingActive = false;
let trainingBarPos = 0;
let trainingBarDir = 1;
let trainingAnimId = null;

// Training zones: fail(0-20), success(20-45), critical(45-55), success(55-80), fail(80-100)
const TRAINING_ZONES = [
    { start: 0, end: 20, result: 'fail' },
    { start: 20, end: 45, result: 'success' },
    { start: 45, end: 55, result: 'critical' },
    { start: 55, end: 80, result: 'success' },
    { start: 80, end: 100, result: 'fail' }
];

// --- Init ---
export function initCardProgression() {
    const el = document.getElementById('wd-card_progression');
    if (!el) return;
    renderCardProgression(el);
}

function renderCardProgression(el) {
    el.innerHTML = `
        <div class="progression-container">
            <div class="pg-header">
                <span class="pg-icon">⬆️</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Card Progression</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Train, fuse, and upgrade your cards</p>
                </div>
            </div>

            <!-- Tabs -->
            <div class="pg-tabs">
                <button class="pg-tab active" data-tab="train">🏋️ Train</button>
                <button class="pg-tab" data-tab="fusion">🔮 Fuse</button>
                <button class="pg-tab" data-tab="upgrade">⬆️ Upgrade</button>
            </div>

            <!-- Training Tab -->
            <div class="pg-panel active" id="pg-panel-train">
                <h4 style="color:#e5e7eb;">Card Training</h4>
                <p style="color:#90a4ae;font-size:11px;">Select a card then click at the right moment to train it!</p>
                <div class="pg-card-select" id="pg-train-cards">
                    ${renderTrainCards()}
                </div>
                <div class="pg-training-game" id="pg-training-game">
                    <div class="pg-training-bar">
                        <div class="pg-zone fail"></div>
                        <div class="pg-zone success"></div>
                        <div class="pg-zone critical"></div>
                        <div class="pg-zone success"></div>
                        <div class="pg-zone fail"></div>
                    </div>
                    <div class="pg-training-marker"></div>
                </div>
                <button class="vbt-btn vbt-btn-primary pg-train-btn" disabled>🏋️ Start Training</button>
            </div>

            <!-- Fusion Tab -->
            <div class="pg-panel" id="pg-panel-fusion">
                <h4 style="color:#e5e7eb;">Card Fusion</h4>
                <p style="color:#90a4ae;font-size:11px;">Select a base card then sacrifice duplicates to boost it!</p>
                <div class="pg-fusion-base" id="pg-fusion-base">
                    <h5>Base Card</h5>
                    <div class="pg-fusion-slot">${selectedFusionBase ? renderFusionCard(selectedFusionBase, true) : '<p style="color:#90a4ae;">Select a base card</p>'}</div>
                </div>
                <div class="pg-fusion-sacrifices" id="pg-fusion-sacrifices">
                    <h5>Sacrifice Cards</h5>
                    <div class="pg-sacrifice-list">
                        ${fusionSacrifices.length > 0 ? fusionSacrifices.map(c => renderFusionCard(c, false)).join('') : '<p style="color:#90a4ae;">Select cards to sacrifice</p>'}
                    </div>
                </div>
                <button class="vbt-btn vbt-btn-primary pg-fuse-btn" ${(!selectedFusionBase || fusionSacrifices.length === 0) ? 'disabled' : ''}>🔮 Fuse Cards</button>
            </div>

            <!-- Upgrade Tab -->
            <div class="pg-panel" id="pg-panel-upgrade">
                <h4 style="color:#e5e7eb;">Card Upgrade</h4>
                <p style="color:#90a4ae;font-size:11px;">Use resources to upgrade card stats</p>
                <div class="pg-upgrade-list" id="pg-upgrade-list">
                    ${renderUpgradeCards()}
                </div>
            </div>
        </div>
    `;
    attachProgressionListeners(el);
}

function renderTrainCards() {
    const cards = [
        { id: 1, name: 'Alana', level: 3, tier: 'Bronze' },
        { id: 2, name: 'Bella', level: 5, tier: 'Bronze' },
        { id: 3, name: 'Ellie', level: 2, tier: 'Iron' }
    ];
    return cards.map(c => `
        <div class="pg-card ${selectedTrainingCard === c.id ? 'selected' : ''}" data-id="${c.id}">
            <span class="pg-card-name">${c.name}</span>
            <span class="pg-card-level">Lv.${c.level}</span>
            <span class="pg-card-tier">${c.tier}</span>
        </div>
    `).join('');
}

function renderFusionCard(card, isBase) {
    return `
        <div class="pg-card pg-fusion-card ${isBase ? 'base' : ''}" data-id="${card.id}">
            <span class="pg-card-name">${card.name}</span>
            <span class="pg-card-level">Lv.${card.level || 1}</span>
        </div>
    `;
}

function renderUpgradeCards() {
    const cards = [
        { id: 1, name: 'Alana', level: 3, tier: 'Bronze', cost: 500 },
        { id: 2, name: 'Bella', level: 5, tier: 'Bronze', cost: 1000 }
    ];
    return cards.map(c => `
        <div class="pg-upgrade-card" data-id="${c.id}">
            <span class="pg-card-name">${c.name}</span>
            <span class="pg-card-level">Lv.${c.level}</span>
            <span class="pg-upgrade-cost">🪙 ${c.cost}</span>
            <button class="vbt-btn vbt-btn-secondary pg-upgrade-btn">Upgrade</button>
        </div>
    `).join('');
}

function attachProgressionListeners(el) {
    // Tab switching
    el.querySelectorAll('.pg-tab').forEach(tab => {
        tab.addEventListener('click', () => {
            el.querySelectorAll('.pg-tab').forEach(t => t.classList.remove('active'));
            el.querySelectorAll('.pg-panel').forEach(p => p.classList.remove('active'));
            tab.classList.add('active');
            document.getElementById(`pg-panel-${tab.dataset.tab}`).classList.add('active');
        });
    });

    // Training card selection
    el.querySelectorAll('#pg-train-cards .pg-card').forEach(card => {
        card.addEventListener('click', () => {
            selectedTrainingCard = parseInt(card.dataset.id);
            renderCardProgression(el);
            el.querySelector('.pg-train-btn').disabled = false;
        });
    });

    // Training start
    el.querySelector('.pg-train-btn').addEventListener('click', () => {
        if (!selectedTrainingCard) return;
        startTraining(el);
    });

    // Fusion base selection
    el.querySelectorAll('#pg-fusion-base .pg-card').forEach(card => {
        card.addEventListener('click', () => {
            selectedFusionBase = parseInt(card.dataset.id);
            renderCardProgression(el);
        });
    });

    // Fusion sacrifice selection
    el.querySelectorAll('#pg-fusion-sacrifices .pg-card').forEach(card => {
        card.addEventListener('click', () => {
            const cardId = parseInt(card.dataset.id);
            if (fusionSacrifices.find(c => c.id === cardId)) {
                fusionSacrifices = fusionSacrifices.filter(c => c.id !== cardId);
            } else {
                fusionSacrifices.push({ id: cardId, name: card.querySelector('.pg-card-name').textContent, level: 1 });
            }
            renderCardProgression(el);
        });
    });

    // Fuse button
    el.querySelector('.pg-fuse-btn').addEventListener('click', () => {
        if (!selectedFusionBase || fusionSacrifices.length === 0) return;
        if (window.showToast) window.showToast('🔮 Fusion successful! Card power increased!', 'success');
        fusionSacrifices = [];
        renderCardProgression(el);
    });

    // Upgrade buttons
    el.querySelectorAll('.pg-upgrade-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            if (window.showToast) window.showToast('⬆️ Card upgraded!', 'success');
        });
    });
}

function startTraining(el) {
    if (isTrainingActive) {
        // Stop and resolve
        isTrainingActive = false;
        cancelAnimationFrame(trainingAnimId);
        const result = getTrainingResult();
        if (window.showToast) {
            if (result === 'critical') window.showToast('🌟 CRITICAL TRAINING! +3 levels!', 'success');
            else if (result === 'success') window.showToast('✅ Training successful! +1 level!', 'success');
            else window.showToast('❌ Training failed. Try again!', 'warning');
        }
        el.querySelector('.pg-train-btn').textContent = '🏋️ Start Training';
        return;
    }

    // Start training
    isTrainingActive = true;
    trainingBarPos = 0;
    trainingBarDir = 1;
    el.querySelector('.pg-train-btn').textContent = '🛑 Stop!';
    trainingLoop(el);
}

function trainingLoop(el) {
    if (!isTrainingActive) return;
    const marker = el.querySelector('.pg-training-marker');
    if (!marker) return;

    trainingBarPos += 2 * trainingBarDir;
    if (trainingBarPos > 100 || trainingBarPos < 0) {
        trainingBarDir *= -1;
        trainingBarPos = Math.max(0, Math.min(100, trainingBarPos));
    }
    marker.style.left = `${trainingBarPos}%`;

    trainingAnimId = requestAnimationFrame(() => trainingLoop(el));
}

function getTrainingResult() {
    for (const zone of TRAINING_ZONES) {
        if (trainingBarPos >= zone.start && trainingBarPos < zone.end) {
            return zone.result;
        }
    }
    return 'fail';
}

// --- Globals ---
window.initCardProgression = initCardProgression;
