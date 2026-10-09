// ============================================================================
// achievement_progress.js — Achievement Tracking & Badge Collection
// ----------------------------------------------------------------------------
// Exposes achievement_handlers.go: progress, auto-unlock, rarity, categories
// Unique visual style: trophy hall / medal cabinet with shine effects
// ============================================================================

var API_BASE = '/api';

// --- Rarity Tiers ---
const RARITY = {
    common: { color: '#9e9e9e', label: 'Common', shine: 'none' },
    rare: { color: '#2196f3', label: 'Rare', shine: 'blue' },
    epic: { color: '#9c27b0', label: 'Epic', shine: 'purple' },
    legendary: { color: '#ff9800', label: 'Legendary', shine: 'gold' },
    mythic: { color: '#f44336', label: 'Mythic', shine: 'red' },
};

// --- Achievement Categories ---
const CATEGORIES = [
    { id: 'battle', name: 'Battle', icon: '⚔️' },
    { id: 'career', name: 'Career', icon: '💼' },
    { id: 'social', name: 'Social', icon: '👥' },
    { id: 'economy', name: 'Economy', icon: '💰' },
    { id: 'exploration', name: 'Exploration', icon: '🗺️' },
    { id: 'special', name: 'Special', icon: '⭐' },
];

// --- State ---
let achievements = [];
let unlockedSet = new Set();
let selectedCategory = 'all';

// --- Init ---
export function initAchievementProgress() {
    const el = document.getElementById('wd-achievements');
    if (!el) return;
    renderAchievementLoading(el);
    fetchAchievements(el);
}

async function fetchAchievements(el) {
    try {
        const resp = await fetch(`${API_BASE}/achievements`);
        if (!resp.ok) throw new Error('Failed to fetch achievements');
        const data = await resp.json();
        achievements = data.definitions || [];
        unlockedSet = new Set(data.unlocked || []);
        renderAchievementProgress(el);
    } catch (e) {
        el.innerHTML = '<p style="color:#90a4ae;">Achievement data unavailable</p>';
    }
}

function renderAchievementLoading(el) {
    el.innerHTML = '<div class="wd-loading">Loading trophy hall…</div>';
}

function renderAchievementProgress(el) {
    const total = achievements.length;
    const unlocked = unlockedSet.size;
    const progress = total > 0 ? Math.round(unlocked / total * 100) : 0;

    const filtered = selectedCategory === 'all'
        ? achievements
        : achievements.filter(a => a.category === selectedCategory);

    let html = `
        <div class="achievement-container">
            <!-- Trophy Hall Header -->
            <div class="trophy-hall-header">
                <div class="trophy-hall-icon">🏆</div>
                <div class="trophy-hall-info">
                    <h3 style="color:#e5e7eb;margin:0;">Trophy Hall</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">${unlocked} of ${total} achievements unlocked</p>
                </div>
                <div class="trophy-progress-ring">
                    <svg viewBox="0 0 36 36" class="trophy-ring-svg">
                        <path class="trophy-ring-bg" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"/>
                        <path class="trophy-ring-fill" stroke-dasharray="${progress}, 100" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"/>
                    </svg>
                    <span class="trophy-ring-text">${progress}%</span>
                </div>
            </div>

            <!-- Category Filter -->
            <div class="achievement-categories">
                <button class="achievement-cat-btn ${selectedCategory === 'all' ? 'active' : ''}" data-cat="all">
                    All
                </button>
                ${CATEGORIES.map(c => `
                    <button class="achievement-cat-btn ${selectedCategory === c.id ? 'active' : ''}" data-cat="${c.id}">
                        ${c.icon} ${c.name}
                    </button>
                `).join('')}
            </div>

            <!-- Achievement Grid -->
            <div class="achievement-grid">
                ${filtered.length > 0 ? filtered.map(a => renderAchievementCard(a)).join('') : `
                    <div class="achievement-empty">
                        <span class="achievement-empty-icon">🔒</span>
                        <p>No achievements in this category</p>
                    </div>
                `}
            </div>

            <!-- Rarity Legend -->
            <div class="rarity-legend">
                <span class="rarity-legend-label">Rarity:</span>
                ${Object.entries(RARITY).map(([key, r]) => `
                    <span class="rarity-legend-item" style="--rarity-color:${r.color}">
                        <span class="rarity-dot"></span>${r.label}
                    </span>
                `).join('')}
            </div>
        </div>
    `;

    el.innerHTML = html;
    attachAchievementListeners(el);
}

function renderAchievementCard(achievement) {
    const isUnlocked = unlockedSet.has(achievement.id);
    const rarity = RARITY[achievement.rarity] || RARITY.common;
    const progress = achievement.progress || 0;
    const target = achievement.target || 1;
    const progressPct = Math.min(100, Math.round(progress / target * 100));

    return `
        <div class="achievement-card ${isUnlocked ? 'unlocked' : 'locked'} rarity-${achievement.rarity || 'common'}"
             data-id="${achievement.id}" style="--rarity-color:${rarity.color}">
            <div class="achievement-card-shine ${rarity.shine}"></div>
            <div class="achievement-card-header">
                <span class="achievement-icon">${isUnlocked ? (achievement.icon || '🏆') : '🔒'}</span>
                <span class="achievement-rarity" style="color:${rarity.color}">${rarity.label}</span>
            </div>
            <div class="achievement-card-body">
                <span class="achievement-title">${escapeHtml(achievement.title || 'Unknown')}</span>
                <span class="achievement-desc">${escapeHtml(achievement.description || '')}</span>
            </div>
            ${!isUnlocked ? `
                <div class="achievement-progress">
                    <div class="achievement-progress-bar">
                        <div class="achievement-progress-fill" style="width:${progressPct}%;background:${rarity.color}"></div>
                    </div>
                    <span class="achievement-progress-text">${progress}/${target}</span>
                </div>
            ` : `
                <div class="achievement-unlocked-badge">✓ UNLOCKED</div>
            `}
        </div>
    `;
}

function attachAchievementListeners(el) {
    // Category filter
    el.querySelectorAll('.achievement-cat-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            selectedCategory = btn.dataset.cat;
            renderAchievementProgress(el);
        });
    });

    // Click unlocked achievements
    el.querySelectorAll('.achievement-card.unlocked').forEach(card => {
        card.addEventListener('click', () => {
            const id = card.dataset.id;
            if (window.showToast) window.showToast(`🏆 Achievement: ${id}`, 'info');
        });
    });
}

// --- Global handlers ---
window.initAchievementProgress = initAchievementProgress;

// --- Utility ---
function escapeHtml(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    }[c]));
}
