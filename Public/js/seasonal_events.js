// SeasonalEventEngine frontend module — Neon-glass seasonal events UI + WS integration
(function() {
    'use strict';

    // State
    let state = {
        activeEvents: [],
        currentWallet: '',
        selectedEventId: null,
        rewardClaimed: false,
        wsConnected: false
    };

    // DOM references (lazy-initialized)
    let overlayEl = null;
    let eventsGridEl = null;
    let participantListEl = null;
    let claimBtnEl = null;
    let statusPanelEl = null;

    // API base path
    const API_BASE = '/api';

    // Initialize module — lazy DOM setup on first open
    function init() {
        if (state._initialized) return;
        state._initialized = true;
        buildOverlay();
    }

    // Build the overlay panel HTML once
    function buildOverlay() {
        const html = `
<div id="seasonal-events-overlay" class="vbt-overlay" style="display:none;">
    <div class="neon-glass-panel seasonal-panels">
        <!-- Panel 1: Active Events Grid -->
        <section class="panel panel-events">
            <h2>🎯 Seasonal Events</h2>
            <button id="btn-close-seasonal" class="vbt-btn vbt-btn-secondary close-overlay-btn">&times;</button>
            <div id="seasonal-events-grid" class="event-cards-grid"></div>
        </section>

        <!-- Panel 2: Event Details + Participants -->
        <section class="panel panel-details">
            <h3 id="selected-event-title">Select an event</h3>
            <p id="selected-event-desc" style="color:#b0bec5;margin-bottom:8px;"></p>
            <div id="event-stats-grid" class="stats-mini-grid"></div>
        </section>

        <!-- Panel 3: Participants + Reward -->
        <section class="panel panel-rewards">
            <h4>Participants</h4>
            <ul id="participant-list" class="participant-list"></ul>
            <button id="btn-join-event" class="vbt-btn vbt-btn-primary join-event-btn" style="display:none;">Join Event</button>
            <button id="btn-claim-reward" class="vbt-btn vbt-btn-accent claim-reward-btn" disabled>Claim Reward</button>
        </section>

        <!-- Panel 4: Status -->
        <section class="panel panel-status">
            <h4>Status</h4>
            <div id="seasonal-status-text" style="color:#80cbc4;font-size:13px;">No event selected.</div>
        </section>

        <!-- Admin section (only if wallet matches admin list) -->
        <section class="panel panel-admin">
            <h4>Admin Controls</h4>
            <button id="btn-create-event" class="vbt-btn vbt-btn-secondary create-event-btn">Create Seasonal Event</button>
            <input type="number" id="admin-budget-input" placeholder="Budget (micro-VBV)" style="width:180px;padding:6px;background:#1a237e;color:#fff;border:1px solid #4dd0e1;border-radius:4px;" />
        </section>
    </div>
</div>`;

        document.body.insertAdjacentHTML('beforeend', html);
        overlayEl = document.getElementById('seasonal-events-overlay');
        eventsGridEl = document.getElementById('seasonal-events-grid');
        participantListEl = document.getElementById('participant-list');
        claimBtnEl = document.getElementById('btn-claim-reward');
        statusPanelEl = document.getElementById('seasonal-status-text');

        // Wire up close button
        overlayEl.querySelector('#btn-close-seasonal').addEventListener('click', function() {
            state.wsConnected = false;
            overlayEl.style.display = 'none';
        });

        // Join event button
        overlayEl.querySelector('.join-event-btn').addEventListener('click', onJoinEvent);

        // Claim reward button
        claimBtnEl.addEventListener('click', onClaimReward);

        // Create event (admin)
        overlayEl.querySelector('.create-event-btn').addEventListener('click', onCreateAdminEvent);
    }

    // Open the overlay
    function openSeasonalEvents() {
        init();
        if (!overlayEl) return;
        overlayEl.style.display = 'flex';
        state.wsConnected = true;
        loadActiveEvents();
    }

    // Load active events from backend
    async function loadActiveEvents() {
        try {
            const resp = await fetch(API_BASE + '/season/status');
            if (!resp.ok) throw new Error('Failed to load events: ' + resp.status);
            state.activeEvents = await resp.json();
            renderEventCards(state.activeEvents);
            updateStatus('Loaded ' + state.activeEvents.length + ' active event(s). Select one for details.');
        } catch (err) {
            console.error('[Seasonal] loadActiveEvents error:', err);
            updateStatus('Error loading events: ' + err.message);
        }
    }

    // Render event cards grid
    function renderEventCards(events) {
        if (!eventsGridEl) return;
        if (events.length === 0) {
            eventsGridEl.innerHTML = '<p style="color:#90a4ae;text-align:center;padding:2rem;">No active seasonal events. Create one or wait for auto-generation.</p>';
            return;
        }

        let html = '';
        events.forEach(function(evt) {
            const timeLeft = getTimeRemaining(evt.EndTime);
            const statusColor = evt.Status === 'active' ? '#00e676' : (evt.Status === 'announcement' ? '#ffeb3b' : '#4dd0c1');

            html += '<div class="event-card" data-event-id="' + evt.EventID + '" onclick="SeasonalEvents.selectEvent(\'' + evt.EventID + '\')">' +
                '<h5 style="color:#e0f7fa;margin-bottom:6px;">' + escapeHtml(evt.Title) + '</h5>' +
                '<span class="event-type-badge" style="background:' + statusColor + '22;color:' + statusColor + ';padding:3px 8px;border-radius:12px;font-size:11px;text-transform:uppercase;">' + evt.Type + '</span>' +
                '<p style="color:#b0bec5;margin-top:6px;font-size:13px;line-height:1.4;">' + escapeHtml(evt.Description) + '</p>' +
                '<div class="event-meta">' +
                    '<span>⏱ ' + timeLeft + '</span>' +
                    '<span>Mult: ×' + evt.Multiplier.toFixed(2) + '</span>' +
                    '<span>Budget: ' + formatMicroVBV(evt.TreasuryBudget || 0) + '</span>' +
                '</div>' +
            '</div>';
        });

        eventsGridEl.innerHTML = html;
    }

    // Select an event and show details
    function selectEvent(eventId) {
        state.selectedEventId = eventId;
        const evt = findEventById(eventId);
        if (!evt) return;

        updateStatus('Selected: ' + escapeHtml(evt.Title));

        // Show stats grid
        const statsGridEl = document.getElementById('event-stats-grid');
        if (statsGridEl) {
            statsGridEl.innerHTML = '<div class="stat-mini"><span>Phase</span><strong>' + evt.Status.toUpperCase() + '</strong></div>' +
                '<div class="stat-mini"><span>Duration</span><strong>' + Math.round(evt.EndTime.getTime()/1000 - evt.StartTime.getTime()/1000) / 3600 + 'h remaining</strong></div>';
        }

        // Load participants for this event
        loadEventParticipants(eventId);
    }

    // Load event participants (polling approach since no dedicated WS participant endpoint yet)
    async function loadEventParticipants(eventId) {
        if (!participantListEl) return;
        const evt = findEventById(eventId);
        if (!evt || !evt.ActiveParticipants) {
            updateStatus('No participants data available.');
            return;
        }

        participantListEl.innerHTML = '';
        Object.keys(evt.ActiveParticipants).forEach(function(wallet, idx) {
            const li = document.createElement('li');
            li.textContent = wallet.substring(0, 12) + '...' + wallet.substring(wallet.length - 8);
            if (wallet === state.currentWallet) {
                li.style.color = '#ffeb3b';
                li.innerHTML += ' <span style="color:#4caf50;">✓ You</span>';
            }
            participantListEl.appendChild(li);

            // Show join button only for first 10 participants (limit display)
        });

        if (Object.keys(evt.ActiveParticipants).length > 20) {
            const li = document.createElement('li');
            li.textContent = '... and ' + (Object.keys(evt.ActiveParticipants).length - 20) + ' more';
            participantListEl.appendChild(li);
        }

        // Show join button if player not yet registered
        var joined = evt.ActiveParticipants[state.currentWallet];
        document.querySelector('.join-event-btn').style.display = state.activeEvents.length > 0 && !joined ? '' : 'none';
    }

    // Join event handler
    async function onJoinEvent() {
        if (!state.selectedEventId || !state.currentWallet) return;
        try {
            const resp = await fetch(API_BASE + '/season/events/join', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({event_id: state.selectedEventId, wallet: state.currentWallet})
            });

            if (!resp.ok) throw new Error('Join failed');
            const data = await resp.json();
            updateStatus('Joined event! You will appear in the participant list shortly.');

            // Reload events to show updated participants
            loadActiveEvents();
        } catch (err) {
            console.error('[Seasonal] Join error:', err);
            updateError('Failed to join event: ' + err.message);
        }
    }

    // Claim reward handler
    async function onClaimReward() {
        if (!state.currentWallet) return;
        try {
            const resp = await fetch(API_BASE + '/season/events/reward', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({wallet: state.currentWallet})
            });

            if (!resp.ok) throw new Error('Claim failed');
            const data = await resp.json();
            updateStatus('Reward claimed! +' + formatMicroVBV(data.reward_micro || 0));
            claimBtnEl.disabled = true;
        } catch (err) {
            console.error('[Seasonal] Claim error:', err);
            updateError('Failed to claim reward: ' + err.message);
        }
    }

    // Create admin event handler
    async function onCreateAdminEvent() {
        const budgetInput = document.getElementById('admin-budget-input');
        if (!budgetInput) return;
        try {
            await fetch('/api/season/admin/create-event', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({
                    type: 'SeasonDiamondWeek',
                    title: 'Special Event - Admin Created',
                    description: 'A special seasonal event created by admin.',
                    duration_hours: 72, // 3 days default
                    multiplier: 1.5,
                    treasury_budget_micro: parseInt(budgetInput.value) || 1000000,
                    created_by: state.currentWallet || 'admin'
                })
            });

            updateStatus('Event created! Refreshing event list...');
            loadActiveEvents();
        } catch (err) {
            console.error('[Seasonal] Create error:', err);
            updateError('Failed to create event: ' + err.message);
        }
    }

    // WebSocket handler registration — called by network.js when WS events arrive
    function registerWSHandlers(networkModule) {
        if (!networkModule || !networkModule.addHandler) return;

        networkModule.addHandler('seasonal_event_joined', function(data) {
            console.log('[Seasonal] Event joined:', data);
            loadActiveEvents(); // Refresh to show updated participants
        });

        networkModule.addHandler('seasonal_event_created', function(data) {
            console.log('[Seasonal] New event created:', data.title || data.event_id);
            updateStatus('New seasonal event: ' + (data.title || data.event_id));
            loadActiveEvents(); // Refresh to show new event in grid
        });

        networkModule.addHandler('seasonal_event_reward', function(data) {
            console.log('[Seasonal] Reward received:', data.reward_micro, 'micro-VBV');
            updateStatus('Reward: +' + formatMicroVBV(data.reward_micro || 0));
            claimBtnEl.disabled = true; // Already claimed via broadcast
        });

        networkModule.addHandler('seasonal_event_pool_updated', function(data) {
            console.log('[Seasonal] Pool updated for event:', data.event_id);
            loadActiveEvents(); // Refresh to show new budget amounts
        });

        networkModule.addHandler('seasonal_event_expired', function(data) {
            console.log('[Seasonal] Event expired:', data.title || data.event_id);
            updateStatus('Event ended: ' + (data.title || data.event_id));
            loadActiveEvents(); // Refresh to remove from grid
        });

        networkModule.addHandler('seasonal_event_activated', function(data) {
            console.log('[Seasonal] Event activated:', data.title || data.event_id);
            updateStatus('Event active: ' + (data.title || data.event_id));
            loadActiveEvents(); // Refresh to show newly active event
        });

        networkModule.addHandler('seasonal_event_reward_claimed', function(data) {
            console.log('[Seasonal] Reward claimed:', data.reward_micro, 'micro-VBV');
            updateStatus('Reward: +' + formatMicroVBV(data.reward_micro || 0));
            claimBtnEl.disabled = true; // Already claimed via broadcast
        });
    }

    // Utility: find event by ID from active events list
    function findEventById(eventId) {
        return state.activeEvents.find(function(e) { return e.EventID === eventId; }) || null;
    }

    // Utility: get time remaining string
    function getTimeRemaining(endTimeStr) {
        var end = new Date(endTimeStr);
        var now = new Date();
        var diffMs = end.getTime() - now.getTime();
        if (diffMs <= 0) return 'Ended';

        var hoursLeft = Math.floor(diffMs / (1000 * 60 * 60));
        var daysLeft = Math.floor(hoursLeft / 24);
        if (daysLeft > 0) {
            var remainingHours = hoursLeft % 24;
            return daysLeft + 'd ' + remainingHours + 'h';
        }
        return hoursLeft + 'h left';
    }

    // Utility: format micro-VBV to human-readable string
    function formatMicroVBV(micro) {
        if (!micro || micro === 0) return '0 VBV';
        var vbv = (micro / 1000000).toFixed(2);
        return vbv + ' $VBV';
    }

    // Utility: escape HTML to prevent XSS in event titles/descriptions
    function escapeHtml(str) {
        if (!str) return '';
        var div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }

    // Update status text with timestamp prefix
    function updateStatus(text) {
        if (statusPanelEl) {
            statusPanelEl.textContent = '[' + new Date().toLocaleTimeString() + '] ' + text;
        } else {
            console.log('[Seasonal] Status:', text);
        }
    }

    // Update error display with red styling for errors
    function updateError(text) {
        if (statusPanelEl) {
            statusPanelEl.style.color = '#ff5252';
            statusPanelEl.textContent = '⚠ [' + new Date().toLocaleTimeString() + '] ' + text;
            setTimeout(function() {
                statusPanelEl.style.color = ''; // Reset color after 3 seconds
            }, 3000);
        } else {
            console.error('[Seasonal] Error:', text);
        }
    }

    // Set current wallet for join/claim operations (called by app.js on login)
    function setCurrentWallet(wallet) {
        state.currentWallet = wallet;
    }

    // Export public API — called from index.html or network.js
    window.SeasonalEvents = {
        open: openSeasonalEvents,
        selectEvent: selectEvent,
        registerWSHandlers: registerWSHandlers,
        setCurrentWallet: setCurrentWallet,
        getState: function() { return state; } // For debugging / inspection by other modules
    };

})();