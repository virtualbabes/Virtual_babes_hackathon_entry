// ============================================================================
// seasonal_events.js — Seasonal Events & Countdown
// ----------------------------------------------------------------------------
// Displays active seasons, events, rewards, and countdown timers
// Connects to: seasonEngine, WebSocket seasonal events
// ============================================================================

var API_BASE = '/api';

// State
let seasonData = null;
let eventsData = [];
let selectedEvent = null;
let countdownInterval = null;

// --- Init ---
export function initSeasonalEvents() {
    const el = document.getElementById('wd-season');
    if (!el) return;
    renderSeasonalLoading(el);
    fetchSeasonData(el);
    startCountdownTimer(el);
}

// --- Data Fetching ---
async function fetchSeasonData(el) {
    try {
        const resp = await fetch(`${API_BASE}/season/status`);
        if (!resp.ok) throw new Error('Failed to fetch season data');
        seasonData = await resp.json();
        renderSeasonalEvents(el);
    } catch (e) {
        // Fallback: render with mock data for demo
        seasonData = getMockSeasonData();
        renderSeasonalEvents(el);
    }
}

function getMockSeasonData() {
    return {
        active: true,
        name: 'Summer Festival',
        ends_at: new Date(Date.now() + 3 * 24 * 60 * 60 * 1000 + 14 * 60 * 60 * 1000 + 22 * 60 * 1000).toISOString(),
        events: [
            { id: 'beach_bash', name: 'Beach Bash', icon: '🏖️', progress: 65, reward: 500, status: 'active' },
            { id: 'sunset_duel', name: 'Sunset Duel', icon: '🌅', progress: 30, reward: 1000, status: 'active' },
            { id: 'fireworks', name: 'Fireworks Show', icon: '🎆', progress: 0, reward: 2000, status: 'upcoming' },
        ],
        rewards: [
            { id: 'summer_card', name: 'Summer Card Pack', icon: '🎴', cost: 5000, claimed: false },
            { id: 'beach_skin', name: 'Beach Board Skin', icon: '🏝️', cost: 10000, claimed: false },
            { id: 'golden_amulet', name: 'Golden Amulet', icon: '✨', cost: 25000, claimed: true },
        ]
    };
}

// --- Render ---
function renderSeasonalLoading(el) {
    el.innerHTML = '<div class="wd-loading">Loading seasonal events…</div>';
}

function renderSeasonalEvents(el) {
    const data = seasonData || getMockSeasonData();
    const events = data.events || [];
    const rewards = data.rewards || [];

    el.innerHTML = `
        <div class="seasonal-container">
            <div class="seasonal-header">
                <span class="seasonal-icon">🎭</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Seasonal Events</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Limited-time events and rewards</p>
                </div>
                <div class="seasonal-status ${data.active ? 'active' : 'inactive'}">
                    ${data.active ? '● LIVE' : '○ OFFLINE'}
                </div>
            </div>

            <!-- Season Banner -->
            <div class="seasonal-banner">
                <div class="seasonal-banner-content">
                    <span class="seasonal-banner-icon">🌞</span>
                    <div class="seasonal-banner-text">
                        <span class="seasonal-banner-name">${data.name || 'Current Season'}</span>
                        <span class="seasonal-banner-countdown" id="seasonal-countdown">--:--:--</span>
                    </div>
                </div>
            </div>

            <!-- Active Events -->
            <div class="seasonal-section">
                <h4 style="color:#e5e7eb;">📅 Current Events</h4>
                <div class="seasonal-events-grid">
                    ${events.length > 0 ? events.map(event => `
                        <div class="seasonal-event-card ${event.status}" data-id="${event.id}">
                            <span class="seasonal-event-icon">${event.icon}</span>
                            <span class="seasonal-event-name">${event.name}</span>
                            <div class="seasonal-event-progress">
                                <div class="seasonal-event-progress-bar" style="width:${event.progress}%"></div>
                            </div>
                            <span class="seasonal-event-progress-text">${event.progress}%</span>
                            <span class="seasonal-event-reward">🎁 ${event.reward} SP</span>
                        </div>
                    `).join('') : '<p style="color:#90a4ae;">No active events</p>'}
                </div>
            </div>

            <!-- Rewards -->
            <div class="seasonal-section">
                <h4 style="color:#e5e7eb;">🏆 Rewards Available</h4>
                <div class="seasonal-rewards-grid">
                    ${rewards.length > 0 ? rewards.map(reward => `
                        <div class="seasonal-reward-card ${reward.claimed ? 'claimed' : ''}" data-id="${reward.id}">
                            <span class="seasonal-reward-icon">${reward.icon}</span>
                            <span class="seasonal-reward-name">${reward.name}</span>
                            <span class="seasonal-reward-cost">${reward.cost} SP</span>
                            ${reward.claimed ? '<span class="seasonal-reward-claimed">✓ Claimed</span>' : `<button class="vbt-btn vbt-btn-primary seasonal-claim-btn" data-id="${reward.id}" data-cost="${reward.cost}">Claim</button>`}
                        </div>
                    `).join('') : '<p style="color:#90a4ae;">No rewards available</p>'}
                </div>
            </div>

            <!-- Event History -->
            <div class="seasonal-section">
                <h4 style="color:#e5e7eb;">📜 Past Seasons</h4>
                <div class="seasonal-history">
                    <div class="seasonal-history-item">
                        <span class="seasonal-history-icon">❄️</span>
                        <span class="seasonal-history-name">Winter Wonderland</span>
                        <span class="seasonal-history-date">Jan 2026</span>
                    </div>
                    <div class="seasonal-history-item">
                        <span class="seasonal-history-icon">🌸</span>
                        <span class="seasonal-history-name">Spring Bloom</span>
                        <span class="seasonal-history-date">Mar 2026</span>
                    </div>
                    <div class="seasonal-history-item">
                        <span class="seasonal-history-icon">🎃</span>
                        <span class="seasonal-history-name">Harvest Moon</span>
                        <span class="seasonal-history-date">Oct 2025</span>
                    </div>
                </div>
            </div>
        </div>
    `;
    attachSeasonalListeners(el);
}

// --- Countdown Timer ---
function startCountdownTimer(el) {
    if (countdownInterval) clearInterval(countdownInterval);
    
    const countdownEl = document.getElementById('seasonal-countdown');
    if (!countdownEl) return;

    const endTime = seasonData?.ends_at ? new Date(seasonData.ends_at) : new Date(Date.now() + 3 * 24 * 60 * 60 * 1000);

    function updateCountdown() {
        const now = new Date();
        const diff = endTime - now;

        if (diff <= 0) {
            countdownEl.textContent = 'ENDED';
            return;
        }

        const days = Math.floor(diff / (1000 * 60 * 60 * 24));
        const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
        const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
        const seconds = Math.floor((diff % (1000 * 60)) / 1000);

        countdownEl.textContent = `${days}d ${hours}h ${minutes}m ${seconds}s`;
    }

    updateCountdown();
    countdownInterval = setInterval(updateCountdown, 1000);
}

// --- Event Listeners ---
function attachSeasonalListeners(el) {
    // Claim reward buttons
    el.querySelectorAll('.seasonal-claim-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const rewardId = btn.dataset.id;
            const cost = parseInt(btn.dataset.cost);
            claimReward(rewardId, cost);
        });
    });

    // Event cards
    el.querySelectorAll('.seasonal-event-card').forEach(card => {
        card.addEventListener('click', () => {
            const eventId = card.dataset.id;
            selectEvent(eventId);
        });
    });
}

function claimReward(rewardId, cost) {
    if (window.showToast) window.showToast(`🎁 Claimed reward for ${cost} SP!`, 'success');
    
    // Update UI
    const btn = document.querySelector(`.seasonal-claim-btn[data-id="${rewardId}"]`);
    if (btn) {
        btn.outerHTML = '<span class="seasonal-reward-claimed">✓ Claimed</span>';
    }
}

function selectEvent(eventId) {
    selectedEvent = eventId;
    document.querySelectorAll('.seasonal-event-card').forEach(c => c.classList.remove('selected'));
    const card = document.querySelector(`.seasonal-event-card[data-id="${eventId}"]`);
    if (card) card.classList.add('selected');
}

// --- WebSocket Handlers ---
export function handleSeasonalWebSocket(msg) {
    switch (msg.type) {
        case 'seasonal_event_joined':
            if (window.showToast) window.showToast(`🎯 Joined event: ${msg.payload.event_title}`, 'success');
            break;
        case 'seasonal_event_created':
            if (window.showToast) window.showToast(`🌸 New event: ${msg.payload.title}`, 'info');
            break;
        case 'seasonal_event_reward':
            if (window.showToast) window.showToast(`🎁 Reward claimed: +${msg.payload.reward_micro / 1000000} $VBV`, 'success');
            break;
        case 'seasonal_event_expired':
            if (window.showToast) window.showToast(`⏰ Event ended: ${msg.payload.title}`, 'info');
            break;
        case 'seasonal_event_activated':
            if (window.showToast) window.showToast(`⭐ Event active: ${msg.payload.title}`, 'success');
            break;
    }
}

// --- Globals ---
window.initSeasonalEvents = initSeasonalEvents;
window.handleSeasonalWebSocket = handleSeasonalWebSocket;
window.openSeasonalEvents = function () {
    if (typeof window.openWorldDashboard === 'function') window.openWorldDashboard();
    if (typeof window.switchWDTab === 'function') window.switchWDTab('season');
    if (typeof window.initSeasonalEvents === 'function') window.initSeasonalEvents();
};
