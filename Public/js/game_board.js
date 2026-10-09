// ============================================================================
// game_board.js — Game Board Battle Screen
// ----------------------------------------------------------------------------
// Enhances existing board-container with:
// - Board themes (elemental backgrounds per tile)
// - Mood/element display on each tile
// - Card flip animations on capture
// - Hand display with power values
// - Power comparison overlay
// - Turn indicator and score display
// Integrates with existing syncUI, PlaceCard, clickGrid from game.js
// ============================================================================

var API_BASE = '/api';

// State
let boardTheme = 'neutral';
let boardMoods = Array(9).fill('Neutral');
let boardElements = Array(9).fill('Neutral');
let activeRules = {};
let selectedCardId = null;
let playerHand = [];
let opponentHand = [];
let playerScore = 0;
let opponentScore = 0;
let currentTurn = 0;
let isMultiplayer = false;

// --- Init ---
export function initGameBoard() {
    const el = document.getElementById('wd-game');
    if (!el) return;
    renderGameBoard(el);
    initCardAnimations();
}

function renderGameBoard(el) {
    el.innerHTML = `
        <div class="game-board-container">
            <!-- Header -->
            <div class="gb-header">
                <div class="gb-player-info gb-opponent">
                    <span class="gb-avatar">🤖</span>
                    <span class="gb-name">Opponent</span>
                    <span class="gb-score" id="gb-opp-score">0</span>
                </div>
                <div class="gb-turn-indicator" id="gb-turn-indicator">
                    <span class="gb-turn-text">Your Turn</span>
                </div>
                <div class="gb-player-info gb-player">
                    <span class="gb-avatar">👤</span>
                    <span class="gb-name">You</span>
                    <span class="gb-score" id="gb-player-score">0</span>
                </div>
            </div>

            <!-- Opponent Hand (face down) -->
            <div class="gb-hand gb-opponent-hand" id="gb-opp-hand">
                ${renderOpponentHand()}
            </div>

            <!-- Game Board -->
            <div class="gb-board-wrapper">
                <div class="gb-board" id="gb-board">
                    ${renderBoardTiles()}
                </div>
                <!-- Power comparison overlay (shown on hover) -->
                <div class="gb-power-overlay" id="gb-power-overlay"></div>
            </div>

            <!-- Player Hand (face up) -->
            <div class="gb-hand gb-player-hand" id="gb-player-hand">
                ${renderPlayerHand()}
            </div>

            <!-- Active Effects Panel -->
            <div class="gb-effects-panel" id="gb-effects-panel">
                <h4>Active Effects</h4>
                <div class="gb-effects-list" id="gb-effects-list">
                    ${renderActiveEffects()}
                </div>
            </div>

            <!-- Action Bar -->
            <div class="gb-actions">
                <button class="vbt-btn vbt-btn-secondary" id="gb-settings-btn">⚙️ Settings</button>
                <button class="vbt-btn vbt-btn-secondary" id="gb-quickcast-btn">⚡ QuickCast</button>
                <button class="vbt-btn vbt-btn-danger" id="gb-forfeit-btn">🏳️ Forfeit</button>
            </div>
        </div>
    `;
    attachBoardListeners(el);

    // §23.5.5 viewer-scoped card display: paint the viewer's own preference over the marked card
    // surfaces. refresh() is one-shot (it fetches only once a wallet exists, and only until the
    // preference has been read). Board tiles are deliberately NOT marked: their ownership is not
    // known at this point, and this layer never guesses who a card belongs to.
    if (window.CardViewSkins) {
        window.CardViewSkins.refresh();
        window.CardViewSkins.applyTo(el);
    }
}

function renderOpponentHand() {
    // data-card-owner="foreign" is the RENDER CONTRACT read by card_view_skins.js: the viewer may
    // restyle how THEY see the opponent's cards, and only surfaces marked this way are touched.
    return [0,1,2,3,4].map(i => `
        <div class="gb-card-slot gb-opp-card" data-index="${i}" data-card-owner="foreign">
            <div class="gb-card-back">?</div>
        </div>
    `).join('');
}

function renderPlayerHand() {
    if (playerHand.length === 0) {
        return [0,1,2,3,4].map(i => `
            <div class="gb-card-slot gb-player-card empty" data-index="${i}">
                <div class="gb-card-empty">Empty ${i+1}</div>
            </div>
        `).join('');
    }
    return playerHand.map((card, i) => `
        <div class="gb-card-slot gb-player-card ${selectedCardId === card.id ? 'selected' : ''}" data-index="${i}" data-id="${card.id}" data-card-owner="self">
            <div class="gb-card-front">
                <span class="gb-card-name">${card.name}</span>
                <span class="gb-card-power">${card.Power ? card.Power.join('/') : '----'}</span>
                <span class="gb-card-mood">${getMoodIcon(card.Mood)}</span>
            </div>
        </div>
    `).join('');
}

function renderBoardTiles() {
    const theme = getThemeClasses(boardTheme);
    return Array(9).fill(0).map((_, i) => {
        const mood = boardMoods[i] || 'Neutral';
        const element = boardElements[i] || 'Neutral';
        return `
            <div class="gb-tile ${theme.tile} ${getMoodClass(mood)} ${getElementClass(element)}" data-index="${i}" data-mood="${mood}" data-element="${element}">
                <div class="gb-tile-inner">
                    <span class="gb-tile-element">${getElementIcon(element)}</span>
                    <span class="gb-tile-mood">${mood.slice(0,3)}</span>
                    <div class="gb-tile-card" id="gb-tile-card-${i}"></div>
                </div>
            </div>
        `;
    }).join('');
}

function getThemeClasses(theme) {
    const themes = {
        neutral: { tile: 'theme-neutral' },
        volcanic: { tile: 'theme-volcanic' },
        ocean: { tile: 'theme-ocean' },
        storm: { tile: 'theme-storm' },
        balanced: { tile: 'theme-balanced' }
    };
    return themes[theme] || themes.neutral;
}

function getMoodClass(mood) {
    const classes = { Volatile: 'mood-volatile', Serene: 'mood-serene', Spirited: 'mood-spirited', Grounded: 'mood-grounded', Neutral: '' };
    return classes[mood] || '';
}

function getElementClass(element) {
    const classes = { Fire: 'elem-fire', Water: 'elem-water', Earth: 'elem-earth', Air: 'elem-air', Neutral: '' };
    return classes[element] || '';
}

function getMoodIcon(mood) {
    const icons = { Volatile: '🔥', Serene: '💧', Spirited: '⚡', Grounded: '🌿', Neutral: '⚪' };
    return icons[mood] || '⚪';
}

function getElementIcon(element) {
    const icons = { Fire: '🔥', Water: '💧', Earth: '🌍', Air: '💨', Neutral: '⚪' };
    return icons[element] || '⚪';
}

function renderActiveEffects() {
    const effects = [];
    if (activeRules.Mood_modifiers) effects.push({ name: 'Mood RPS', icon: '🎭', color: '#9c27b0' });
    if (activeRules.Elemental_affinity) effects.push({ name: 'Element', icon: '🔥', color: '#f44336' });
    if (activeRules.Prisoner) effects.push({ name: 'Prisoner', icon: '⛓️', color: '#ffc107' });
    if (activeRules.Power_copy) effects.push({ name: 'Copy', icon: '📋', color: '#4caf50' });
    if (activeRules.Power_up) effects.push({ name: 'Plus', icon: '➕', color: '#3b82f6' });
    
    return effects.length > 0 ? effects.map(e => `
        <span class="gb-effect" style="border-color:${e.color};color:${e.color}">
            ${e.icon} ${e.name}
        </span>
    `).join('') : '<span style="color:#90a4ae;">No active effects</span>';
}

function attachBoardListeners(el) {
    // Player hand selection
    el.querySelectorAll('.gb-player-card:not(.empty)').forEach(card => {
        card.addEventListener('click', () => {
            selectedCardId = parseInt(card.dataset.id);
            renderGameBoard(el);
        });
    });

    // Tile clicks
    el.querySelectorAll('.gb-tile').forEach(tile => {
        tile.addEventListener('click', () => {
            const index = parseInt(tile.dataset.index);
            if (selectedCardId !== null) {
                // Place card
                if (window.PlaceCard) {
                    window.PlaceCard(index, selectedCardId);
                    selectedCardId = null;
                }
            }
        });
    });

    // Action buttons
    const forfeitBtn = el.querySelector('#gb-forfeit-btn');
    if (forfeitBtn) forfeitBtn.addEventListener('click', () => {
        if (window.showToast) window.showToast('🏳️ Match forfeited', 'warning');
    });

    const quickcastBtn = el.querySelector('#gb-quickcast-btn');
    if (quickcastBtn) quickcastBtn.addEventListener('click', () => {
        if (window.showQuickCastMenu) window.showQuickCastMenu();
    });
}

// --- Global ---
window.initGameBoard = initGameBoard;
