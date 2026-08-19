// ============================================================================
// justice_dashboard.js — Justice Hegemony Dashboard (P2-A, Task 4002)
// ============================================================================
// Neon-glass bounty board view, Truth Serum overlay, shield status tracker.
// Module delegated from app.js; interacts with WS for live updates.
// ============================================================================

(function() {
    'use strict';

    // --- Constants ---
    const JUSTICE_DASHBOARD_OVERLAY = 'justice-dashboard-overlay';
    const TRUTH_SERUM_DURATION = 30; // seconds
    const REPUTATION_SHIELD_CAPACITY = 50; // default protection value

    // --- DOM References ---
    let overlayEl = null;
    let bountyListEl = null;
    let truthSerumStatusEl = null;
    let shieldStatusEl = null;

    // --- State ---
    let activeBounties = [];
    let playerJusticeStats = null;
    let truthSerumActive = false;
    let shieldActive = false;
    let shieldRemaining = 0;

    // ========================================================================
    // INITIALIZATION
    // ========================================================================

    /**
     * Initialize the Justice Dashboard overlay.
     * Creates DOM elements and binds event handlers.
     */
    function init() {
        if (overlayEl) return; // Already initialized

        // Create overlay structure
        overlayEl = document.createElement('div');
        overlayEl.id = JUSTICE_DASHBOARD_OVERLAY;
        overlayEl.className = 'justice-overlay neon-glass-panel';
        overlayEl.style.display = 'none';
        overlayEl.innerHTML = `
            <div class="justice-header">
                <h2>⚖️ Justice Hegemony Dashboard</h2>
                <button class="close-btn" onclick="window.closeJusticeDashboard()">✕</button>
            </div>
            <div class="justice-body">
                <div class="justice-stats-panel">
                    <h3>Faction Power</h3>
                    <div id="justice-power-display" class="stat-block">0</div>
                    <div id="justice-tier-display" class="stat-label">Tier 0 (Peon)</div>
                </div>
                <div class="justice-bounty-panel">
                    <h3>Active Bounties</h3>
                    <div id="bounty-list" class="bounty-scroll"></div>
                </div>
                <div class="justice-effects-panel">
                    <h3>Faction Effects</h3>
                    <div class="effect-row">
                        <span>🔬 Truth Serum:</span>
                        <span id="truth-serum-status" class="effect-indicator inactive">Inactive</span>
                    </div>
                    <div class="effect-row">
                        <span>🛡️ Reputation Shield:</span>
                        <span id="shield-status" class="effect-indicator inactive">
                            Remaining: 0/${REPUTATION_SHIELD_CAPACITY}
                        </span>
                    </div>
                </div>
            </div>
        `;

        document.body.appendChild(overlayEl);

        // Bind references
        bountyListEl = document.getElementById('bounty-list');
        truthSerumStatusEl = document.getElementById('truth-serum-status');
        shieldStatusEl = document.getElementById('shield-status');
    }

    // ========================================================================
    // OVERLAY CONTROL
    // ========================================================================

    /**
     * Open the Justice Dashboard overlay.
     */
    function openJusticeDashboard() {
        init();
        overlayEl.style.display = 'flex';
        overlayEl.style.flexDirection = 'column';
        overlayEl.style.alignItems = 'center';
        refreshDashboardData();
    }

    /**
     * Close the Justice Dashboard overlay.
     */
    function closeJusticeDashboard() {
        if (overlayEl) {
            overlayEl.style.display = 'none';
        }
    }

    // ========================================================================
    // DATA FETCHING
    // ========================================================================

    /**
     * Fetch fresh dashboard data from backend.
     * Called on open and via WebSocket refresh events.
     */
    async function refreshDashboardData() {
        // P0 FIX: Pass current player's wallet as query parameter (required by backend)
        let currentPlayerWallet = '';
        if (window.CONFIG && window.CONFIG.VAULT_ADDRESS) {
            currentPlayerWallet = window.CONFIG.VAULT_ADDRESS;
        } else if (typeof getActivePlayer === 'function') {
            const active = getActivePlayer();
            if (active && active.wallet) currentPlayerWallet = active.wallet;
        }

        let url = '/api/justice/dashboard';
        if (currentPlayerWallet !== '') {
            url += '?wallet=' + encodeURIComponent(currentPlayerWallet);
        }

        try {
            const response = await fetch(url);
            if (!response.ok) throw new Error(`HTTP ${response.status}`);

            const data = await response.json();
            playerJusticeStats = data;
            activeBounties = data.bounties || [];

            updatePowerDisplay(data.powerBonus || 0, data.tier || 0);
            renderBountyList(activeBounties);
            updateEffectStatuses(data.truthSerumActive, data.shieldRemaining);
        } catch (err) {
            console.error('[JusticeDashboard] Refresh failed:', err);
        }
    }

    // ========================================================================
    // DISPLAY UPDATES
    // ========================================================================

    /**
     * Update the faction power display.
     */
    function updatePowerDisplay(powerBonus, tier) {
        const powerEl = document.getElementById('justice-power-display');
        const tierEl = document.getElementById('justice-tier-display');
        if (powerEl) powerEl.textContent = `+${powerBonus}% Power Bonus`;
        if (tierEl) tierEl.textContent = `Tier ${tier} (${getTierLabel(tier)})`;
    }

    /**
     * Get human-readable tier label.
     */
    function getTierLabel(tier) {
        const labels = ['Peon', 'Apprentice', 'Journeyman', 'Expert', 'Master', 'Boss'];
        return labels[tier] || 'Unknown';
    }

    /**
     * Render the active bounties list.
     */
    function renderBountyList(bounties) {
        if (!bountyListEl) return;

        if (!bounties || bounties.length === 0) {
            bountyListEl.innerHTML = '<div class="empty-state">No active bounties.</div>';
            return;
        }

        bountyListEl.innerHTML = bounties.map(bounty => `
            <div class="bounty-item" data-target="${bounty.targetWallet || ''}">
                <div class="bounty-target">${truncateAddress(bounty.targetName || bounty.targetWallet)}</div>
                <div class="bounty-details">
                    <span class="wanted-level">Wanted: ${bounty.wantedLevel || 0}</span>
                    <span class="reward">${formatMicroUnits(bounty.reward || 0)} μVBV</span>
                </div>
                <button class=" bounty-action-btn" onclick="window.captureBounty('${bounty.targetWallet}')">
                    Capture
                </button>
            </div>
        `).join('');
    }

    /**
     * Update Truth Serum and Shield effect status indicators.
     */
    function updateEffectStatuses(truthSerumActive, shieldRemaining) {
        if (truthSerumStatusEl) {
            const tsa = typeof truthSerumActive === 'boolean' ? truthSerumActive : !!truthSerumActive;
            truthSerumStatusEl.className = `effect-indicator ${tsa ? 'active' : 'inactive'}`;
            truthSerumStatusEl.textContent = tsa 
                ? `${TRUTH_SERUM_DURATION}s active` 
                : 'Inactive';
        }

        if (shieldStatusEl) {
            const sr = shieldRemaining || 0;
            const isShielded = sr > 0;
            shieldStatusEl.className = `effect-indicator ${isShielded ? 'active' : 'inactive'}`;
            shieldStatusEl.textContent = isShielded
                ? `${sr}/${REPUTATION_SHIELD_CAPACITY} remaining`
                : 'Inactive';
        }
    }

    // ========================================================================
    // ACTIONS
    // ========================================================================

    /**
     * Trigger Truth Serum application on a target.
     * @param {string} targetWallet - Wallet address of the target.
     */
    async function applyTruthSerum(targetWallet) {
        if (!targetWallet) return;

        try {
            const response = await fetch('/api/justice/use-truth-serum', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ targetWallet })
            });

            if (!response.ok) throw new Error(`HTTP ${response.status}`);

            const result = await response.json();
            broadcastTruthSerumEvent(targetWallet, TRUTH_SERUM_DURATION);
        } catch (err) {
            console.error('[JusticeDashboard] Truth Serum failed:', err);
        }
    }

    /**
     * Capture a bounty target.
     * @param {string} targetWallet - Wallet address of the bounty target.
     */
    async function captureBounty(targetWallet) {
        if (!targetWallet) return;

        try {
            const response = await fetch('/api/justice/capture-bounty', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ targetWallet })
            });

            if (!response.ok) throw new Error(`HTTP ${response.status}`);

            refreshDashboardData();
        } catch (err) {
            console.error('[JusticeDashboard] Bounty capture failed:', err);
        }
    }

    // ========================================================================
    // WEBSOCKET EVENTS
    // ========================================================================

    /**
     * Handle WebSocket justice event broadcasts.
     * @param {object} event - WS message object with type and data fields.
     */
    function handleJusticeWebSocketEvent(event) {
        switch (event.type) {
            case 'justice_card_awarded':
                console.log('[JusticeDashboard] Card awarded:', event.data.card);
                refreshDashboardData();
                break;

            case 'truth_serum_applied':
                truthSerumActive = true;
                setTimeout(() => {
                    truthSerumActive = false;
                    updateEffectStatuses(false, shieldRemaining);
                }, TRUTH_SERUM_DURATION * 1000);
                break;

            case 'shield_active':
                shieldRemaining = event.data.remaining || REPUTATION_SHIELD_CAPACITY;
                updateEffectStatuses(truthSerumActive, shieldRemaining);
                break;

            case 'dashboard_refresh':
                refreshDashboardData();
                break;

            case 'bounty_updated':
                const bountyIdx = activeBounties.findIndex(b => b.targetWallet === event.data.targetWallet);
                if (bountyIdx >= 0) {
                    activeBounties[bountyIdx] = event.data;
                    renderBountyList(activeBounties);
                } else {
                    refreshDashboardData();
                }
                break;

            default:
                console.log('[JusticeDashboard] Unknown WS event:', event.type);
        }
    }

    /**
     * Broadcast a Truth Serum application toast notification.
     */
    function broadcastTruthSerumEvent(targetWallet, duration) {
        if (window.ws && window.ws.readyState === WebSocket.OPEN) {
            window.ws.send(JSON.stringify({
                type: 'truth_serum_applied',
                data: { targetWallet, duration }
            }));
        }
    }

    // ========================================================================
    // UTILITIES
    // ========================================================================

    /**
     * Truncate a wallet address for display.
     */
    function truncateAddress(addr) {
        if (!addr || addr.length < 12) return addr || 'Unknown';
        return `${addr.slice(0, 6)}...${addr.slice(-4)}`;
    }

    /**
     * Format micro-units to human-readable.
     */
    function formatMicroUnits(microUnits) {
        if (microUnits >= 1_000_000_000_000) 
            return `${(microUnits / 1_000_000_000_000).toFixed(1)}T`;
        if (microUnits >= 1_000_000_000) 
            return `${(microUnits / 1_000_000_000).toFixed(1)}B`;
        if (microUnits >= 1_000_000) 
            return `${(microUnits / 1_000_000).toFixed(1)}M`;
        return microUnits.toString();
    }

    // ========================================================================
    // WINDOW EXPORTS
    // ========================================================================

    // ========================================================================
    // NETWORK.JS CALLBACK EXPORTS (P2-A Task 4003 — WebSocket Bridge)
    // These are the dispatch targets for network.js case handlers.
    // ========================================================================

    /** @type {function} Called by network.js on justice_card_awarded */
    window.onJusticeCardAwarded = function(payload) {
        if (overlayEl && overlayEl.style.display !== 'none') refreshDashboardData();
    };

    /** @type {function} Called by network.js on truth_serum_applied */
    window.onTruthSerumApplied = function(payload) {
        truthSerumActive = true;
        setTimeout(() => {
            truthSerumActive = false;
            updateEffectStatuses(false, shieldRemaining);
        }, TRUTH_SERUM_DURATION * 1000);
        if (overlayEl && overlayEl.style.display !== 'none') refreshDashboardData();
    };

    /** @type {function} Called by network.js on shield_active */
    window.onShieldActive = function(payload) {
        shieldRemaining = payload.remaining || REPUTATION_SHIELD_CAPACITY;
        updateEffectStatuses(truthSerumActive, shieldRemaining);
        if (overlayEl && overlayEl.style.display !== 'none') refreshDashboardData();
    };

    /** @type {function} Called by network.js on dashboard_refresh */
    window.onDashboardRefresh = function() {
        if (overlayEl && overlayEl.style.display !== 'none') refreshDashboardData();
    };

    /** @type {function} Called by network.js on bounty_updated */
    window.onBountyUpdated = function(payload) {
        const idx = activeBounties.findIndex(b => b.targetWallet === payload.targetWallet);
        if (idx >= 0) {
            activeBounties[idx] = payload;
            renderBountyList(activeBounties);
        } else {
            refreshDashboardData();
        }
    };

    window.openJusticeDashboard = openJusticeDashboard;
    window.closeJusticeDashboard = closeJusticeDashboard;
    window.applyTruthSerum = applyTruthSerum;
    window.captureBounty = captureBounty;
    window.handleJusticeWebSocketEvent = handleJusticeWebSocketEvent;

})();
