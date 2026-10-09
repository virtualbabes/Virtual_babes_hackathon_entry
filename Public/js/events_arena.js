// ============================================================================
// events_arena.js — Event Arena & Competition Hub
// ----------------------------------------------------------------------------
// Exposes season_engine.go: events, brackets, rewards, participation
// Unique visual style: festival/arena with ticket stubs and winner podiums
// ============================================================================

var API_BASE = '/api';

// --- Event Types ---
const EVENT_TYPES = [
    { id: 'treasure_hunt', name: 'Treasure Hunt', icon: '🗺️', color: '#ffc107', desc: 'Find hidden treasures' },
    { id: 'search_rescue', name: 'Search & Rescue', icon: '🔍', color: '#4caf50', desc: 'Godly acts for faith' },
    { id: 'gang_bash', name: 'Gang Bash', icon: '⚔️', color: '#f44336', desc: 'Combat event' },
    { id: 'wild_bot_hunt', name: 'Wild Bot Hunt', icon: '🤖', color: '#9c27b0', desc: 'Hunt rogue AI' },
    { id: 'pet_breeding_show', name: 'Pet Breeding Show', icon: '🐾', color: '#2196f3', desc: 'Compete for stat bonuses' },
];

// --- State ---
let events = [];
let activeEvent = null;

// --- Init ---
export function initEventsArena() {
    const el = document.getElementById('wd-events');
    if (!el) return;
    renderEventsLoading(el);
    fetchEventsData(el);
}

async function fetchEventsData(el) {
    try {
        const resp = await fetch(`${API_BASE}/season/events`);
        if (!resp.ok) throw new Error('Failed to fetch events');
        const data = await resp.json();
        events = data.events || [];
        activeEvent = events.find(e => e.status === 'active') || null;
        renderEventsArena(el);
    } catch (e) {
        el.innerHTML = '<p style="color:#90a4ae;">Event data unavailable</p>';
    }
}

function renderEventsLoading(el) {
    el.innerHTML = '<div class="wd-loading">Loading events…</div>';
}

function renderEventsArena(el) {
    let html = `
        <div class="events-arena-container">
            <!-- Festival Header -->
            <div class="events-header">
                <div class="events-fireworks">🎆</div>
                <div class="events-info">
                    <h3 style="color:#e5e7eb;margin:0;">Event Arena</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Compete for glory and rewards</p>
                </div>
                <div class="events-count">
                    <span class="events-count-num">${events.length}</span>
                    <span class="events-count-label">Active</span>
                </div>
            </div>

            <!-- Active Event Spotlight -->
            ${activeEvent ? renderEventSpotlight(activeEvent) : ''}

            <!-- Event Types -->
            <div class="event-types-section">
                <h4 style="color:#e5e7eb;">🎪 Event Types</h4>
                <div class="event-types-grid">
                    ${EVENT_TYPES.map(t => `
                        <div class="event-type-card" data-type="${t.id}" style="--type-color:${t.color}">
                            <span class="event-type-icon">${t.icon}</span>
                            <span class="event-type-name">${t.name}</span>
                            <span class="event-type-desc">${t.desc}</span>
                        </div>
                    `).join('')}
                </div>
            </div>

            <!-- Event List -->
            <div class="event-list-section">
                <h4 style="color:#e5e7eb;">📋 Active Events</h4>
                <div class="event-list">
                    ${events.length > 0 ? events.map(e => renderEventCard(e)).join('') : renderEmptyEvents()}
                </div>
            </div>

            <!-- Create Event -->
            <div class="create-event-section">
                <button class="vbt-btn vbt-btn-primary" onclick="window.eventsCreate()">
                    ✨ Create Event
                </button>
            </div>
        </div>
    `;

    el.innerHTML = html;
    attachEventsListeners(el);
}

function renderEventSpotlight(event) {
    return `
        <div class="event-spotlight">
            <div class="event-spotlight-header">
                <span class="event-spotlight-badge">⭐ FEATURED</span>
            </div>
            <div class="event-spotlight-content">
                <span class="event-spotlight-icon">${event.icon || '🎉'}</span>
                <div class="event-spotlight-info">
                    <span class="event-spotlight-name">${escapeHtml(event.title || event.name || 'Event')}</span>
                    <span class="event-spotlight-participants">👥 ${(event.participants || 0)} participants</span>
                </div>
                <div class="event-spotlight-pot">🏆 ${formatVBV(event.reward_micro || event.pot_micro || 0)}</div>
            </div>
            <button class="vbt-btn vbt-btn-primary event-join-btn">Join Event</button>
        </div>
    `;
}

function renderEventCard(event) {
    const statusColors = { active: '#4caf50', upcoming: '#2196f3', ended: '#9e9e9e' };
    const statusColor = statusColors[event.status] || '#9e9e9e';

    return `
        <div class="event-card status-${event.status}">
            <div class="event-card-header">
                <span class="event-card-icon">${event.icon || '🎉'}</span>
                <span class="event-card-name">${escapeHtml(event.title || event.name || 'Event')}</span>
                <span class="event-card-status" style="background:${statusColor}22;color:${statusColor}">${(event.status || 'unknown').toUpperCase()}</span>
            </div>
            <div class="event-card-details">
                <span class="event-detail">👥 ${event.participants || 0} participants</span>
                <span class="event-detail">🏆 ${formatVBV(event.reward_micro || event.pot_micro || 0)}</span>
            </div>
            <button class="vbt-btn vbt-btn-secondary event-view-btn">View</button>
        </div>
    `;
}

function renderEmptyEvents() {
    return `
        <div class="events-empty">
            <span class="events-empty-icon">🎪</span>
            <p>No active events. Create one to get the party started!</p>
        </div>
    `;
}

function attachEventsListeners(el) {
    // Event type cards
    el.querySelectorAll('.event-type-card').forEach(card => {
        card.addEventListener('click', () => {
            const type = card.dataset.type;
            if (window.showToast) window.showToast(`Creating ${type} event...`, 'info');
        });
    });

    // Join button
    const joinBtn = el.querySelector('.event-join-btn');
    if (joinBtn) {
        joinBtn.addEventListener('click', () => {
            if (window.showToast) window.showToast('🎉 Joined event! Good luck!', 'success');
        });
    }
}

// --- Global handlers ---
window.eventsCreate = function() {
    const name = prompt('Event name:');
    if (name) {
        if (window.showToast) window.showToast(`✨ Event "${name}" created!`, 'success');
    }
};

window.initEventsArena = initEventsArena;

// --- Utility ---
function escapeHtml(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    }[c]));
}

function formatVBV(micro) {
    const vbv = Number(micro) / 1_000_000;
    if (vbv >= 1_000_000) return `${(vbv / 1_000_000).toFixed(1)}M`;
    if (vbv >= 1_000) return `${(vbv / 1_000).toFixed(1)}K`;
    return vbv.toFixed(2) + ' $VBV';
}
