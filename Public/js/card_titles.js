// ============================================================================
// card_titles.js — Card Titles & Perks System
// ----------------------------------------------------------------------------
// Ported from Triple Triad: card-titles.js + profile-titles.js
// Titles are earned via achievements/stats and equipped to cards for perks
// Connects to: Achievement Progress, Identity Editor, Deck Manager
// ============================================================================

var API_BASE = '/api';

// Title definitions (synergizes with existing achievement system)
const CARD_TITLES = [
    { id: 'first_blood', name: 'First Blood', icon: '🩸', desc: 'Win your first match', perk: '+5% power vs new players', condition: 'wins >= 1', rarity: 'common' },
    { id: 'bounty_hunter', name: 'Bounty Hunter', icon: '🎯', desc: 'Capture 5 bounties', perk: '+10% power vs Wanted', condition: 'bounties >= 5', rarity: 'rare' },
    { id: 'combo_master', name: 'Combo Master', icon: '💥', desc: 'Trigger 10 combo chains', perk: '+1 combo chain length', condition: 'combos >= 10', rarity: 'rare' },
    { id: 'elemental_sage', name: 'Elemental Sage', icon: '🔮', desc: 'Win 5 matches with elemental sync', perk: '+25 elemental bonus', condition: 'element_wins >= 5', rarity: 'epic' },
    { id: 'prisoner_taker', name: 'Prisoner Taker', icon: '⛓️', desc: 'Win 3 matches with Prisoner rule', perk: 'Prisoner rule active by default', condition: 'prisoner_wins >= 3', rarity: 'epic' },
    { id: 'mood_warden', name: 'Mood Warden', icon: '🎭', desc: 'Win 5 matches with Mood modifiers', perk: '+1 mood bonus', condition: 'mood_wins >= 5', rarity: 'rare' },
    { id: 'flawless', name: 'Flawless', icon: '💎', desc: 'Win without losing a card', perk: '+15% power', condition: 'flawless_wins >= 1', rarity: 'legendary' },
    { id: 'collector', name: 'Collector', icon: '📦', desc: 'Own 20 cards', perk: '+5% power per 10 cards', condition: 'cards_owned >= 20', rarity: 'epic' },
    { id: 'veteran', name: 'Veteran', icon: '⭐', desc: 'Win 50 matches', perk: '+10% power', condition: 'wins >= 50', rarity: 'legendary' },
    { id: 'divine', name: 'Divine', icon: '✨', desc: 'Max faith coherence', perk: '+20% faith bonus', condition: 'faith >= 100', rarity: 'legendary' }
];

// State
let equippedPlayerTitle = null;
let equippedCardTitles = {}; // cardId -> titleId
let unlockedTitles = new Set();
let playerStats = { wins: 0, bounties: 0, combos: 0, element_wins: 0, prisoner_wins: 0, mood_wins: 0, flawless_wins: 0, cards_owned: 0, faith: 0 };

// --- Init ---
export function initCardTitles() {
    const el = document.getElementById('wd-card_titles');
    if (!el) return;
    renderCardTitles(el);
}

function renderCardTitles(el) {
    el.innerHTML = `
        <div class="titles-container">
            <div class="titles-header">
                <span class="titles-icon">🏅</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Card Titles & Perks</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Earn titles by achieving milestones, equip them for perks</p>
                </div>
            </div>

            <!-- Player Title -->
            <div class="titles-section">
                <h4 style="color:#e5e7eb;">Player Title</h4>
                <div class="titles-player">
                    ${equippedPlayerTitle ? renderEquippedPlayerTitle() : '<p style="color:#90a4ae;">No title equipped</p>'}
                </div>
            </div>

            <!-- Unlocked Titles -->
            <div class="titles-section">
                <h4 style="color:#e5e7eb;">Unlocked Titles (${unlockedTitles.size}/${CARD_TITLES.length})</h4>
                <div class="titles-grid">
                    ${renderUnlockedTitles()}
                </div>
            </div>

            <!-- Locked Titles -->
            <div class="titles-section">
                <h4 style="color:#e5e7eb;">Locked Titles</h4>
                <div class="titles-grid">
                    ${renderLockedTitles()}
                </div>
            </div>

            <!-- Card Title Equipment -->
            <div class="titles-section">
                <h4 style="color:#e5e7eb;">Card Title Equipment</h4>
                <p style="color:#90a4ae;font-size:11px;">Equip titles to individual cards for bonus perks</p>
                <div class="titles-card-slots" id="titles-card-slots">
                    ${renderCardTitleSlots()}
                </div>
            </div>
        </div>
    `;
    attachTitlesListeners(el);
}

function renderEquippedPlayerTitle() {
    const title = CARD_TITLES.find(t => t.id === equippedPlayerTitle);
    if (!title) return '';
    return `
        <div class="titles-equipped rarity-${title.rarity}">
            <span class="titles-equipped-icon">${title.icon}</span>
            <div class="titles-equipped-info">
                <span class="titles-equipped-name">${title.name}</span>
                <span class="titles-equipped-perk">${title.perk}</span>
            </div>
            <button class="vbt-btn vbt-btn-secondary titles-unequip-btn">Unequip</button>
        </div>
    `;
}

function renderUnlockedTitles() {
    const unlocked = CARD_TITLES.filter(t => unlockedTitles.has(t.id));
    if (unlocked.length === 0) return '<p style="color:#90a4ae;">No titles unlocked yet. Start playing!</p>';
    return unlocked.map(t => `
        <div class="titles-card rarity-${t.rarity} ${equippedPlayerTitle === t.id ? 'equipped' : ''}" data-id="${t.id}">
            <span class="titles-card-icon">${t.icon}</span>
            <span class="titles-card-name">${t.name}</span>
            <span class="titles-card-desc">${t.desc}</span>
            <span class="titles-card-perk">${t.perk}</span>
            <button class="vbt-btn vbt-btn-primary titles-equip-btn" data-id="${t.id}">${equippedPlayerTitle === t.id ? 'Equipped' : 'Equip'}</button>
        </div>
    `).join('');
}

function renderLockedTitles() {
    const locked = CARD_TITLES.filter(t => !unlockedTitles.has(t.id));
    return locked.map(t => `
        <div class="titles-card locked" data-id="${t.id}">
            <span class="titles-card-icon">🔒</span>
            <span class="titles-card-name">${t.name}</span>
            <span class="titles-card-desc">${t.desc}</span>
            <span class="titles-card-perk">${t.perk}</span>
            <span class="titles-card-progress">${getProgressText(t)}</span>
        </div>
    `).join('');
}

function getProgressText(title) {
    const cond = title.condition;
    let current = 0, target = 1;
    if (cond.startsWith('wins >=')) { current = playerStats.wins; target = parseInt(cond.split('>=')[1]); }
    else if (cond.startsWith('bounties >=')) { current = playerStats.bounties; target = parseInt(cond.split('>=')[1]); }
    else if (cond.startsWith('combos >=')) { current = playerStats.combos; target = parseInt(cond.split('>=')[1]); }
    else if (cond.startsWith('element_wins >=')) { current = playerStats.element_wins; target = parseInt(cond.split('>=')[1]); }
    else if (cond.startsWith('prisoner_wins >=')) { current = playerStats.prisoner_wins; target = parseInt(cond.split('>=')[1]); }
    else if (cond.startsWith('mood_wins >=')) { current = playerStats.mood_wins; target = parseInt(cond.split('>=')[1]); }
    else if (cond.startsWith('flawless_wins >=')) { current = playerStats.flawless_wins; target = parseInt(cond.split('>=')[1]); }
    else if (cond.startsWith('cards_owned >=')) { current = playerStats.cards_owned; target = parseInt(cond.split('>=')[1]); }
    else if (cond.startsWith('faith >=')) { current = playerStats.faith; target = parseInt(cond.split('>=')[1]); }
    return `${Math.min(current, target)}/${target}`;
}

function renderCardTitleSlots() {
    const cards = [
        { id: 1, name: 'Alana', tier: 'Bronze' },
        { id: 2, name: 'Bella', tier: 'Bronze' },
        { id: 3, name: 'Ellie', tier: 'Iron' }
    ];
    return cards.map(c => {
        const titleId = equippedCardTitles[c.id];
        const title = titleId ? CARD_TITLES.find(t => t.id === titleId) : null;
        return `
            <div class="titles-slot">
                <span class="titles-slot-card">${c.name}</span>
                <span class="titles-slot-title ${title ? '' : 'empty'}">${title ? `${title.icon} ${title.name}` : 'No title'}</span>
                ${title ? `<span class="titles-slot-perk">${title.perk}</span>` : ''}
                <button class="vbt-btn vbt-btn-secondary titles-slot-btn" data-card="${c.id}">${title ? 'Change' : 'Equip'}</button>
            </div>
        `;
    }).join('');
}

function attachTitlesListeners(el) {
    // Unequip player title
    el.querySelectorAll('.titles-unequip-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            equippedPlayerTitle = null;
            renderCardTitles(el);
            if (window.showToast) window.showToast('Title unequipped', 'info');
        });
    });

    // Equip player title
    el.querySelectorAll('.titles-equip-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            equippedPlayerTitle = btn.dataset.id;
            renderCardTitles(el);
            const title = CARD_TITLES.find(t => t.id === equippedPlayerTitle);
            if (window.showToast && title) window.showToast(`Equipped: ${title.name}!`, 'success');
        });
    });

    // Card title slots
    el.querySelectorAll('.titles-slot-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const cardId = parseInt(btn.dataset.card);
            // Cycle through unlocked titles
            const unlocked = CARD_TITLES.filter(t => unlockedTitles.has(t.id));
            if (unlocked.length === 0) {
                if (window.showToast) window.showToast('No titles unlocked', 'warning');
                return;
            }
            const currentIdx = unlocked.findIndex(t => t.id === equippedCardTitles[cardId]);
            const nextIdx = (currentIdx + 1) % (unlocked.length + 1); // +1 for "no title"
            equippedCardTitles[cardId] = nextIdx === unlocked.length ? null : unlocked[nextIdx].id;
            renderCardTitles(el);
        });
    });
}

// --- Simulate unlocking titles (for demo) ---
export function checkTitleUnlocks(stats) {
    playerStats = { ...playerStats, ...stats };
    let newUnlock = null;
    for (const title of CARD_TITLES) {
        if (!unlockedTitles.has(title.id) && checkCondition(title.condition, playerStats)) {
            unlockedTitles.add(title.id);
            newUnlock = title;
        }
    }
    return newUnlock;
}

function checkCondition(cond, stats) {
    const [key, op, val] = cond.split(/([><=]+)/);
    const statVal = stats[key.trim()] || 0;
    const target = parseInt(val.trim());
    if (op === '>=') return statVal >= target;
    if (op === '>') return statVal > target;
    if (op === '<=') return statVal <= target;
    if (op === '<') return statVal < target;
    if (op === '===') return statVal === target;
    return false;
}

// --- Globals ---
window.initCardTitles = initCardTitles;
window.checkTitleUnlocks = checkTitleUnlocks;
