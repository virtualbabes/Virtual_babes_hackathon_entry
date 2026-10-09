// ============================================================================
// match_creator.js — Match Creation & Rule Configuration
// ----------------------------------------------------------------------------
// Configures game rules before match: Mood modifiers, Prisoner rule,
// Elemental affinities, board themes, campaign mode
// Ported from Triple Triad: campaign.js + rules-engine.js + post-game-modes.js
// ============================================================================

var API_BASE = '/api';

// Rule definitions with display info
const RULES = [
    { id: 'Mood_modifiers', name: 'Mood Modifiers', icon: '🎭', desc: 'Card mood vs board slot = ±1 power (RPS system)' },
    { id: 'Prisoner', name: 'Prisoner Rule', icon: '⛓️', desc: 'Loser loses their best card at match end' },
    { id: 'Elemental_affinity', name: 'Elemental Affinity', icon: '🔥', desc: 'Card element vs board tile = ±25 power' },
    { id: 'Power_copy', name: 'Power Copy', icon: '📋', desc: '2+ adjacent cards with same power = capture group' },
    { id: 'Power_up', name: 'Power Up', icon: '➕', desc: '2+ adjacent cards with same sum = capture group' },
    { id: 'Elemental_sync', name: 'Elemental Sync', icon: '🌪️', desc: 'Card mood vs board mood = ±50 power (existing)' }
];

// Board themes (ported from Triple Triad game-board-themes.json structure)
const BOARD_THEMES = [
    { id: 'neutral', name: 'Neutral', icon: '⬜', moods: ['Neutral','Neutral','Neutral','Neutral','Neutral','Neutral','Neutral','Neutral','Neutral'], elements: ['Neutral','Neutral','Neutral','Neutral','Neutral','Neutral','Neutral','Neutral','Neutral'] },
    { id: 'volcanic', name: 'Volcanic', icon: '🌋', moods: ['Volatile','Serene','Volatile','Grounded','Volatile','Serene','Volatile','Grounded','Volatile'], elements: ['Fire','Earth','Fire','Earth','Fire','Earth','Fire','Earth','Fire'] },
    { id: 'ocean', name: 'Ocean', icon: '🌊', moods: ['Serene','Grounded','Serene','Spirited','Serene','Grounded','Serene','Spirited','Serene'], elements: ['Water','Water','Water','Water','Water','Water','Water','Water','Water'] },
    { id: 'storm', name: 'Storm', icon: '⛈️', moods: ['Spirited','Volatile','Spirited','Grounded','Spirited','Volatile','Spirited','Grounded','Spirited'], elements: ['Air','Air','Air','Air','Air','Air','Air','Air','Air'] },
    { id: 'balanced', name: 'Balanced', icon: '☯️', moods: ['Grounded','Spirited','Serene','Volatile','Neutral','Volatile','Serene','Spirited','Grounded'], elements: ['Earth','Air','Water','Fire','Neutral','Fire','Water','Air','Earth'] }
];

// Game modes (ported from Triple Triad post-game-modes.js)
const GAME_MODES = [
    { id: 'casual', name: 'Casual', icon: '🎮', desc: 'No risk, practice mode', prisoner: false },
    { id: 'ranked', name: 'Ranked', icon: '🏆', desc: 'Competitive with ratings', prisoner: false },
    { id: 'prisoner', name: 'Prisoner Brawl', icon: '⛓️', desc: 'Loser loses a card!', prisoner: true },
    { id: 'campaign', name: 'Campaign', icon: '📖', desc: 'Story progression', prisoner: false },
    { id: 'tournament', name: 'Tournament', icon: '🏅', desc: 'Bracket competition', prisoner: false }
];

// State
let selectedRules = {};
let selectedTheme = 'neutral';
let selectedMode = 'casual';

// --- Init ---
export function initMatchCreator() {
    const el = document.getElementById('wd-create_match');
    if (!el) return;
    renderMatchCreator(el);
}

function renderMatchCreator(el) {
    el.innerHTML = `
        <div class="match-creator-container">
            <div class="mc-header">
                <span class="mc-icon">⚔️</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Create Match</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Configure rules, theme, and mode</p>
                </div>
            </div>

            <!-- Game Mode -->
            <div class="mc-section">
                <h4 style="color:#e5e7eb;">🎯 Game Mode</h4>
                <div class="mc-modes">
                    ${GAME_MODES.map(m => `
                        <div class="mc-mode ${selectedMode === m.id ? 'active' : ''}" data-mode="${m.id}">
                            <span class="mc-mode-icon">${m.icon}</span>
                            <span class="mc-mode-name">${m.name}</span>
                            <span class="mc-mode-desc">${m.desc}</span>
                        </div>
                    `).join('')}
                </div>
            </div>

            <!-- Rules -->
            <div class="mc-section">
                <h4 style="color:#e5e7eb;">📋 Rules</h4>
                <div class="mc-rules">
                    ${RULES.map(r => `
                        <div class="mc-rule ${selectedRules[r.id] ? 'active' : ''}" data-rule="${r.id}">
                            <span class="mc-rule-icon">${r.icon}</span>
                            <div class="mc-rule-info">
                                <span class="mc-rule-name">${r.name}</span>
                                <span class="mc-rule-desc">${r.desc}</span>
                            </div>
                            <div class="mc-rule-toggle ${selectedRules[r.id] ? 'on' : ''}"></div>
                        </div>
                    `).join('')}
                </div>
            </div>

            <!-- Board Theme -->
            <div class="mc-section">
                <h4 style="color:#e5e7eb;">🎨 Board Theme</h4>
                <div class="mc-themes">
                    ${BOARD_THEMES.map(t => `
                        <div class="mc-theme ${selectedTheme === t.id ? 'active' : ''}" data-theme="${t.id}">
                            <span class="mc-theme-icon">${t.icon}</span>
                            <span class="mc-theme-name">${t.name}</span>
                        </div>
                    `).join('')}
                </div>
            </div>

            <!-- Selected Theme Preview -->
            <div class="mc-section">
                <h4 style="color:#e5e7eb;">👁️ Board Preview</h4>
                <div class="mc-board-preview" id="mc-board-preview">
                    ${renderBoardPreview()}
                </div>
            </div>

            <!-- Queue Button -->
            <div class="mc-section">
                <button class="vbt-btn vbt-btn-primary mc-queue-btn">⚔️ Queue for Match</button>
            </div>
        </div>
    `;
    attachCreatorListeners(el);
}

function renderBoardPreview() {
    const theme = BOARD_THEMES.find(t => t.id === selectedTheme) || BOARD_THEMES[0];
    let html = '<div class="mc-preview-grid">';
    for (let i = 0; i < 9; i++) {
        const mood = theme.moods[i];
        const element = theme.elements[i];
        const moodColor = getMoodColor(mood);
        const elementIcon = getElementIcon(element);
        html += `
            <div class="mc-preview-slot" style="background:${moodColor}22;border-color:${moodColor};">
                <span class="mc-preview-element">${elementIcon}</span>
                <span class="mc-preview-mood" style="color:${moodColor};font-size:8px;">${mood}</span>
            </div>
        `;
    }
    html += '</div>';
    return html;
}

function getMoodColor(mood) {
    const colors = { Volatile: '#ef4444', Serene: '#3b82f6', Spirited: '#f59e0b', Grounded: '#22c55e', Neutral: '#6b7280' };
    return colors[mood] || '#6b7280';
}

function getElementIcon(element) {
    const icons = { Fire: '🔥', Water: '💧', Earth: '🌍', Air: '💨', Neutral: '⚪' };
    return icons[element] || '⚪';
}

function attachCreatorListeners(el) {
    // Mode selection
    el.querySelectorAll('.mc-mode').forEach(mode => {
        mode.addEventListener('click', () => {
            selectedMode = mode.dataset.mode;
            const m = GAME_MODES.find(g => g.id === selectedMode);
            if (m && m.prisoner) selectedRules['Prisoner'] = true;
            renderMatchCreator(el);
        });
    });

    // Rule toggles
    el.querySelectorAll('.mc-rule').forEach(rule => {
        rule.addEventListener('click', () => {
            const ruleId = rule.dataset.rule;
            selectedRules[ruleId] = !selectedRules[ruleId];
            renderMatchCreator(el);
        });
    });

    // Theme selection
    el.querySelectorAll('.mc-theme').forEach(theme => {
        theme.addEventListener('click', () => {
            selectedTheme = theme.dataset.theme;
            renderMatchCreator(el);
        });
    });

    // Queue button
    const queueBtn = el.querySelector('.mc-queue-btn');
    if (queueBtn) {
        queueBtn.addEventListener('click', () => {
            queueMatch();
        });
    }
}

function queueMatch() {
    const config = {
        rules: selectedRules,
        theme: selectedTheme,
        mode: selectedMode,
        boardMoods: BOARD_THEMES.find(t => t.id === selectedTheme)?.moods || [],
        boardElements: BOARD_THEMES.find(t => t.id === selectedTheme)?.elements || []
    };

    // Store config for game.js to read
    if (window.GameConfig) window.GameConfig = config;

    if (window.showToast) window.showToast(`⚔️ Queuing for ${selectedMode} match...`, 'info');

    // Call existing matchmaking
    if (window.toggleMatchmakingQueue) window.toggleMatchmakingQueue();
}

// --- Global ---
window.initMatchCreator = initMatchCreator;
