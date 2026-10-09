// ============================================================================
// character_mood.js — NPC & Citizen Mood Display
// ----------------------------------------------------------------------------
// Ported from Triple Triad: character-mood.js
// Displays mood states for NPCs and AI citizens
// Connects to: AI Citizens, Faith System, Justice Dashboard
// ============================================================================

var API_BASE = '/api';

// Mood states
const MOOD_STATES = {
    Hostile:  { color: '#ef4444', icon: '😠', min: 0,  max: 20 },
    Annoyed:  { color: '#f59e0b', icon: '😒', min: 21, max: 40 },
    Neutral:  { color: '#6b7280', icon: '😐', min: 41, max: 60 },
    Friendly: { color: '#22c55e', icon: '😊', min: 61, max: 80 },
    Loyal:    { color: '#3b82f6', icon: '😍', min: 81, max: 100 }
};

// State
let characterMoods = {};
let averageMood = 50;

// --- Init ---
export function initCharacterMood() {
    const el = document.getElementById('wd-mood');
    if (!el) return;
    renderCharacterMood(el);
}

function renderCharacterMood(el) {
    el.innerHTML = `
        <div class="mood-container">
            <div class="mood-header">
                <span class="mood-icon">🎭</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Character Mood</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">NPC and citizen emotional states</p>
                </div>
                <div class="mood-average">
                    <span class="mood-average-label">Average</span>
                    <span class="mood-average-value" style="color:${getMoodColor(averageMood)}">${averageMood}</span>
                </div>
            </div>

            <!-- Mood Overview -->
            <div class="mood-section">
                <h4 style="color:#e5e7eb;">Mood Overview</h4>
                <div class="mood-overview">
                    ${renderMoodOverview()}
                </div>
            </div>

            <!-- Character List -->
            <div class="mood-section">
                <h4 style="color:#e5e7eb;">Characters</h4>
                <div class="mood-characters">
                    ${renderCharacterList()}
                </div>
            </div>

            <!-- Mood Actions -->
            <div class="mood-section">
                <h4 style="color:#e5e7eb;">Mood Actions</h4>
                <div class="mood-actions">
                    <button class="vbt-btn vbt-btn-secondary" id="mood-boost-btn">✨ Boost Mood</button>
                    <button class="vbt-btn vbt-btn-secondary" id="mood-calm-btn">😌 Calm Down</button>
                    <button class="vbt-btn vbt-btn-secondary" id="mood-refresh-btn">🔄 Refresh</button>
                </div>
            </div>
        </div>
    `;
    attachMoodListeners(el);
}

function renderMoodOverview() {
    const counts = { Hostile: 0, Annoyed: 0, Neutral: 0, Friendly: 0, Loyal: 0 };
    Object.values(characterMoods).forEach(mood => {
        const state = getMoodState(mood.value);
        counts[state]++;
    });

    return Object.entries(counts).map(([state, count]) => {
        const info = MOOD_STATES[state];
        return `
            <div class="mood-stat" style="border-color:${info.color}">
                <span class="mood-stat-icon">${info.icon}</span>
                <span class="mood-stat-count">${count}</span>
                <span class="mood-stat-label">${state}</span>
            </div>
        `;
    }).join('');
}

function renderCharacterList() {
    const characters = Object.entries(characterMoods);
    if (characters.length === 0) {
        return '<p style="color:#90a4ae;">No character mood data available</p>';
    }
    return characters.map(([name, data]) => {
        const state = getMoodState(data.value);
        const info = MOOD_STATES[state];
        return `
            <div class="mood-character" data-name="${name}">
                <span class="mood-char-icon">${info.icon}</span>
                <span class="mood-char-name">${name}</span>
                <span class="mood-char-value" style="color:${info.color}">${data.value}</span>
                <span class="mood-char-state" style="color:${info.color}">${state}</span>
            </div>
        `;
    }).join('');
}

function getMoodState(value) {
    if (value <= 20) return 'Hostile';
    if (value <= 40) return 'Annoyed';
    if (value <= 60) return 'Neutral';
    if (value <= 80) return 'Friendly';
    return 'Loyal';
}

function getMoodColor(value) {
    return MOOD_STATES[getMoodState(value)].color;
}

function attachMoodListeners(el) {
    const boostBtn = el.querySelector('#mood-boost-btn');
    const calmBtn = el.querySelector('#mood-calm-btn');
    const refreshBtn = el.querySelector('#mood-refresh-btn');

    if (boostBtn) boostBtn.addEventListener('click', () => {
        Object.keys(characterMoods).forEach(name => {
            characterMoods[name].value = Math.min(100, characterMoods[name].value + 10);
        });
        updateAverageMood();
        renderCharacterMood(el);
        if (window.showToast) window.showToast('✨ Mood boosted!', 'success');
    });

    if (calmBtn) calmBtn.addEventListener('click', () => {
        Object.keys(characterMoods).forEach(name => {
            characterMoods[name].value = Math.max(0, characterMoods[name].value - 10);
        });
        updateAverageMood();
        renderCharacterMood(el);
        if (window.showToast) window.showToast('😌 Mood calmed', 'info');
    });

    if (refreshBtn) refreshBtn.addEventListener('click', () => {
        renderCharacterMood(el);
        if (window.showToast) window.showToast('🔄 Mood data refreshed', 'info');
    });
}

function updateAverageMood() {
    const values = Object.values(characterMoods).map(m => m.value);
    averageMood = values.length > 0 ? Math.round(values.reduce((a, b) => a + b, 0) / values.length) : 50;
}

// --- Global ---
window.initCharacterMood = initCharacterMood;
