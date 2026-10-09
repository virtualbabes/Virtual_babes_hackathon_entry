// ============================================================================
// game_locations.js — Game Locations Hub
// ----------------------------------------------------------------------------
// Ported from Triple Triad: arcade.js + casino.js + dive-bar.js + lounge.js + underground-fight-club.js
// Each location has unique rules, themes, and stakes
// ============================================================================

var API_BASE = '/api';

// Location definitions
const LOCATIONS = [
    {
        id: 'arcade',
        name: 'Arcade',
        icon: '🕹️',
        desc: 'Casual games, win back losses',
        rules: ['Power_copy', 'Power_up'],
        theme: 'neutral',
        stakes: 'low',
        color: '#4caf50'
    },
    {
        id: 'lounge',
        name: 'Lounge',
        icon: '🛋️',
        desc: 'Premium ranked matches',
        rules: ['Power_copy', 'Power_up', 'Mood_modifiers'],
        theme: 'balanced',
        stakes: 'medium',
        color: '#9c27b0'
    },
    {
        id: 'dive_bar',
        name: 'Dive Bar',
        icon: '🍺',
        desc: 'Rough matches, no special rules',
        rules: [],
        theme: 'volcanic',
        stakes: 'medium',
        color: '#f44336'
    },
    {
        id: 'casino',
        name: 'Casino',
        icon: '🎰',
        desc: 'High-stakes wagering',
        rules: ['Power_copy', 'Power_up', 'Elemental_affinity'],
        theme: 'storm',
        stakes: 'high',
        color: '#ffc107'
    },
    {
        id: 'underground',
        name: 'Underground',
        icon: '🥊',
        desc: 'Fight club with prisoner rule',
        rules: ['Power_copy', 'Power_up', 'Prisoner'],
        theme: 'volcanic',
        stakes: 'extreme',
        color: '#ef4444'
    }
];

// State
let selectedLocation = null;
let wagerAmount = 100;

// --- Init ---
export function initGameLocations() {
    const el = document.getElementById('wd-locations');
    if (!el) return;
    renderGameLocations(el);
}

function renderGameLocations(el) {
    el.innerHTML = `
        <div class="locations-container">
            <div class="loc-header">
                <span class="loc-icon">🏟️</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Game Locations</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Choose your battleground</p>
                </div>
            </div>

            <!-- Location Grid -->
            <div class="loc-grid">
                ${LOCATIONS.map(loc => `
                    <div class="loc-card ${selectedLocation === loc.id ? 'selected' : ''}" data-id="${loc.id}" style="border-color:${loc.color}44">
                        <span class="loc-card-icon">${loc.icon}</span>
                        <span class="loc-card-name">${loc.name}</span>
                        <span class="loc-card-desc">${loc.desc}</span>
                        <span class="loc-card-stakes" style="color:${loc.color}">${loc.stakes.toUpperCase()}</span>
                    </div>
                `).join('')}
            </div>

            <!-- Selected Location Details -->
            ${selectedLocation ? renderLocationDetails() : '<p style="color:#90a4ae;text-align:center;padding:20px;">Select a location to view details</p>'}
        </div>
    `;
    attachLocationListeners(el);
}

function renderLocationDetails() {
    const loc = LOCATIONS.find(l => l.id === selectedLocation);
    if (!loc) return '';

    return `
        <div class="loc-details" style="border-color:${loc.color}">
            <div class="loc-details-header">
                <span class="loc-details-icon">${loc.icon}</span>
                <div>
                    <h4 style="color:#e5e7eb;margin:0;">${loc.name}</h4>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">${loc.desc}</p>
                </div>
                <span class="loc-details-stakes" style="background:${loc.color}22;color:${loc.color};border:1px solid ${loc.color}">${loc.stakes.toUpperCase()}</span>
            </div>

            <!-- Rules -->
            <div class="loc-rules">
                <h5>Active Rules</h5>
                <div class="loc-rules-list">
                    ${loc.rules.length > 0 ? loc.rules.map(r => `
                        <span class="loc-rule">${getRuleIcon(r)} ${getRuleName(r)}</span>
                    `).join('') : '<span style="color:#90a4ae;">Basic rules only</span>'}
                </div>
            </div>

            <!-- Board Theme Preview -->
            <div class="loc-theme">
                <h5>Board Theme</h5>
                <div class="loc-theme-preview">
                    ${renderThemePreview(loc.theme)}
                </div>
            </div>

            <!-- Wager (for high-stakes locations) -->
            ${loc.stakes === 'high' || loc.stakes === 'extreme' ? `
                <div class="loc-wager">
                    <h5>Wager Amount</h5>
                    <div class="loc-wager-controls">
                        <button class="vbt-btn vbt-btn-secondary" id="loc-wager-down">-</button>
                        <span class="loc-wager-amount" id="loc-wager-amount">${wagerAmount} SP</span>
                        <button class="vbt-btn vbt-btn-secondary" id="loc-wager-up">+</button>
                    </div>
                </div>
            ` : ''}

            <!-- Play Button -->
            <button class="vbt-btn loc-play-btn" style="background:${loc.color};width:100%;margin-top:12px;" data-id="${loc.id}">
                ⚔️ Play at ${loc.name}
            </button>
        </div>
    `;
}

function renderThemePreview(theme) {
    const themes = {
        neutral: ['⬜','⬜','⬜','⬜','⬜','⬜','⬜','⬜','⬜'],
        volcanic: ['🌋','🔥','🌋','🔥','🌋','🔥','🌋','🔥','🌋'],
        ocean: ['🌊','🌊','🌊','🌊','🌊','🌊','🌊','🌊','🌊'],
        storm: ['⛈️','⚡','⛈️','⚡','⛈️','⚡','⛈️','⚡','⛈️'],
        balanced: ['🔥','💧','🌍','💨','⚪','🔥','💧','🌍','💨']
    };
    const icons = themes[theme] || themes.neutral;
    return `
        <div class="loc-preview-grid">
            ${icons.map((icon, i) => `
                <div class="loc-preview-slot theme-${theme}">${icon}</div>
            `).join('')}
        </div>
    `;
}

function getRuleIcon(rule) {
    const icons = { Power_copy: '📋', Power_up: '➕', Mood_modifiers: '🎭', Elemental_affinity: '🔥', Prisoner: '⛓️' };
    return icons[rule] || '📋';
}

function getRuleName(rule) {
    const names = { Power_copy: 'Power Copy', Power_up: 'Power Up', Mood_modifiers: 'Mood RPS', Elemental_affinity: 'Element', Prisoner: 'Prisoner' };
    return names[rule] || rule;
}

function attachLocationListeners(el) {
    // Location selection
    el.querySelectorAll('.loc-card').forEach(card => {
        card.addEventListener('click', () => {
            selectedLocation = card.dataset.id;
            renderGameLocations(el);
        });
    });

    // Wager controls
    const wagerDown = el.querySelector('#loc-wager-down');
    const wagerUp = el.querySelector('#loc-wager-up');
    if (wagerDown) wagerDown.addEventListener('click', () => { wagerAmount = Math.max(10, wagerAmount - 50); renderGameLocations(el); });
    if (wagerUp) wagerUp.addEventListener('click', () => { wagerAmount += 50; renderGameLocations(el); });

    // Play button
    el.querySelectorAll('.loc-play-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const locId = btn.dataset.id;
            const loc = LOCATIONS.find(l => l.id === locId);
            if (window.showToast) window.showToast(`⚔️ Entering ${loc.name}...`, 'info');
            
            // Store location config for game.js
            if (window.GameLocation) window.GameLocation = loc;
            
            // Switch to game board
            if (window.switchWDTab) window.switchWDTab('game');
        });
    });
}

// --- Globals ---
window.initGameLocations = initGameLocations;
