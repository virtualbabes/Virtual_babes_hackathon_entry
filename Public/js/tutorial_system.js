// ============================================================================
// tutorial_system.js — Interactive Tutorial & Onboarding
// ----------------------------------------------------------------------------
// Ported from Triple Triad: tutorial.js (584 lines)
// Dynamic tutorial with card-specific steps, rule explanations, onboarding
// ============================================================================

var API_BASE = '/api';

// Tutorial steps
const TUTORIAL_STEPS = [
    {
        id: 'welcome',
        title: 'Welcome to the Arena',
        content: 'Welcome, challenger! This tutorial will teach you the basics of card combat.',
        visual: '🎮',
        action: 'next'
    },
    {
        id: 'hand',
        title: 'Your Hand',
        content: 'You have 5 cards. Each card has 4 power values (Top, Right, Bottom, Left). Select a card to see its stats.',
        visual: '🃏',
        action: 'select_card'
    },
    {
        id: 'placement',
        title: 'Placing Cards',
        content: 'Click a card in your hand, then click an empty tile on the 3x3 board to place it.',
        visual: '🎯',
        action: 'place_card'
    },
    {
        id: 'capture_basic',
        title: 'Basic Capture',
        content: 'When you place a card, if your power is HIGHER than an adjacent enemy card, you capture it!',
        visual: '⚔️',
        action: 'capture'
    },
    {
        id: 'elemental_sync',
        title: 'Elemental Sync',
        content: 'Card mood vs board tile mood: Same mood = +50 power, Weakness = -50 power. Match your cards to tiles!',
        visual: '🔥',
        action: 'next'
    },
    {
        id: 'mood_rps',
        title: 'Mood Modifiers (RPS)',
        content: 'Mood beats weakness: Volatile→Serene→Spirited→Grounded→Volatile. Card on matching tile = +1 power!',
        visual: '🎭',
        action: 'next'
    },
    {
        id: 'power_copy',
        title: 'Power Copy Rule',
        content: 'If 2+ adjacent cards have the SAME power as your placed card, ALL of them flip!',
        visual: '📋',
        action: 'next'
    },
    {
        id: 'power_up',
        title: 'Power Up Rule',
        content: 'If 2+ adjacent cards sum to the SAME total, ALL of them flip!',
        visual: '➕',
        action: 'next'
    },
    {
        id: 'combo',
        title: 'Combo Chains',
        content: 'Captured cards can trigger MORE captures! Chain combos for massive swings!',
        visual: '💥',
        action: 'combo'
    },
    {
        id: 'complete',
        title: 'Tutorial Complete!',
        content: 'You\'re ready! Build a deck, choose a location, and battle for glory.',
        visual: '🏆',
        action: 'finish'
    }
];

// State
let currentStep = 0;
let isActive = false;
let tutorialOverlay = null;

// --- Init ---
export function initTutorialSystem() {
    const el = document.getElementById('wd-tutorial');
    if (!el) return;
    renderTutorialSystem(el);
}

function renderTutorialSystem(el) {
    el.innerHTML = `
        <div class="tutorial-container">
            <div class="tut-header">
                <span class="tut-icon">📖</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Tutorial</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Learn the game mechanics step by step</p>
                </div>
            </div>

            <!-- Progress Bar -->
            <div class="tut-progress">
                <div class="tut-progress-bar" style="width:${((currentStep + 1) / TUTORIAL_STEPS.length) * 100}%"></div>
                <span class="tut-progress-text">Step ${currentStep + 1} of ${TUTORIAL_STEPS.length}</span>
            </div>

            <!-- Current Step -->
            <div class="tut-step-card">
                <span class="tut-step-visual">${TUTORIAL_STEPS[currentStep].visual}</span>
                <h4 class="tut-step-title">${TUTORIAL_STEPS[currentStep].title}</h4>
                <p class="tut-step-content">${TUTORIAL_STEPS[currentStep].content}</p>
            </div>

            <!-- Action Area -->
            <div class="tut-action">
                ${getActionButton(TUTORIAL_STEPS[currentStep])}
            </div>

            <!-- Navigation -->
            <div class="tut-nav">
                <button class="vbt-btn vbt-btn-secondary" id="tut-prev" ${currentStep === 0 ? 'disabled' : ''}>← Prev</button>
                <button class="vbt-btn vbt-btn-primary" id="tut-next" ${currentStep === TUTORIAL_STEPS.length - 1 ? 'disabled' : ''}>Next →</button>
            </div>

            <!-- Skip -->
            <button class="tut-skip" id="tut-skip">Skip Tutorial</button>
        </div>
    `;
    attachTutorialListeners(el);
}

function getActionButton(step) {
    switch (step.action) {
        case 'select_card': return '<p style="color:#90a4ae;">Select a card in your hand below</p>';
        case 'place_card': return '<p style="color:#90a4ae;">Click an empty tile to place your card</p>';
        case 'capture': return '<p style="color:#90a4e;">Capture an enemy card!</p>';
        case 'combo': return '<p style="color:#90a4ae;">Trigger a combo chain!</p>';
        case 'finish': return '<button class="vbt-btn vbt-btn-primary" id="tut-finish">🎮 Start Playing!</button>';
        default: return '<p style="color:#90a4ae;">Click Next to continue</p>';
    }
}

function attachTutorialListeners(el) {
    const prevBtn = el.querySelector('#tut-prev');
    const nextBtn = el.querySelector('#tut-next');
    const skipBtn = el.querySelector('#tut-skip');
    const finishBtn = el.querySelector('#tut-finish');

    if (prevBtn) prevBtn.addEventListener('click', () => {
        if (currentStep > 0) {
            currentStep--;
            renderTutorialSystem(el);
        }
    });

    if (nextBtn) nextBtn.addEventListener('click', () => {
        if (currentStep < TUTORIAL_STEPS.length - 1) {
            currentStep++;
            renderTutorialSystem(el);
        }
    });

    if (skipBtn) skipBtn.addEventListener('click', () => {
        currentStep = 0;
        if (window.showToast) window.showToast('Tutorial skipped', 'info');
        renderTutorialSystem(el);
    });

    if (finishBtn) finishBtn.addEventListener('click', () => {
        currentStep = 0;
        if (window.showToast) window.showToast('🎮 Welcome to the Arena!', 'success');
        if (window.switchWDTab) window.switchWDTab('create_match');
    });
}

// --- In-Game Tutorial Overlay ---
export function showTutorialOverlay(title, content) {
    if (tutorialOverlay) tutorialOverlay.remove();
    
    tutorialOverlay = document.createElement('div');
    tutorialOverlay.className = 'tutorial-overlay';
    tutorialOverlay.innerHTML = `
        <div class="tutorial-bubble">
            <h4>${title}</h4>
            <p>${content}</p>
            <button class="vbt-btn vbt-btn-primary tutorial-got-it">Got it!</button>
        </div>
    `;
    
    document.body.appendChild(tutorialOverlay);
    
    const gotIt = tutorialOverlay.querySelector('.tutorial-got-it');
    if (gotIt) gotIt.addEventListener('click', () => {
        tutorialOverlay.remove();
        tutorialOverlay = null;
    });
}

// --- Inject Tutorial CSS ---
export function injectTutorialCSS() {
    if (document.getElementById('tutorial-styles')) return;
    
    const style = document.createElement('style');
    style.id = 'tutorial-styles';
    style.textContent = TUTORIAL_CSS;
    document.head.appendChild(style);
}

const TUTORIAL_CSS = `
.tutorial-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 10000;
    animation: fadeIn 0.3s ease-out;
}

.tutorial-bubble {
    background: linear-gradient(135deg, #1a1a2e, #16213e);
    border: 2px solid #9c27b0;
    border-radius: 12px;
    padding: 24px;
    max-width: 400px;
    text-align: center;
    box-shadow: 0 0 30px rgba(156, 39, 176, 0.3);
}

.tutorial-bubble h4 {
    color: #e5e7eb;
    margin: 0 0 12px;
    font-size: 16px;
}

.tutorial-bubble p {
    color: #90a4ae;
    font-size: 12px;
    margin: 0 0 16px;
    line-height: 1.5;
}

@keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
}
`;

// --- Globals ---
window.initTutorialSystem = initTutorialSystem;
window.showTutorialOverlay = showTutorialOverlay;
