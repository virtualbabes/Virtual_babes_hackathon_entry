// Bounty Tracker Overlay — Active Bounty Display with PILLAR 13 Buff Bonuses
// Displays nearby bounties with enhanced tracking from Bounty Hunter career tier.

(function() {
    'use strict';

    // ============================================================================
    // BOUNTY_TRACKER_MODULE — Namespace for bounty tracker overlay system
    // ============================================================================
    
    const BountyTracker = {};
    
    // Internal state
    let activeBounties = [];        // Array of bounty objects with target info, value, timer
    let hunterTier = 0;             // Current Bounty Hunter career tier (0-6)
    let trackingBonusMultiplier = 1.0; // Computed from tier: ×1 at Peon, up to ×2.5+ at Boss
    let wsConnection = null;        // WebSocket connection for live updates
    
    // ============================================================================
    // DOM REFERENCES — Lazy initialization of overlay elements
    // ============================================================================
    
    function getOverlayContainer() {
        let container = document.getElementById('bounty-tracker-overlay');
        if (!container) {
            container = createBountyTrackerDOM();
        }
        return container;
    }
    
    function createBountyTrackerDOM() {
        // Create the overlay panel (neon-glass style matching game aesthetic)
        const overlay = document.createElement('div');
        overlay.id = 'bounty-tracker-overlay';
        overlay.className = 'bounty-tracker-panel hidden';
        
        overlay.innerHTML = `
            <div class="bounty-header">
                <h3>⚡ Active Bounties</h3>
                <button class="bounty-close-btn" title="Close tracker">&times;</button>
            </div>
            
            <!-- Hunter Tier Display -->
            <div class="hunter-tier-display">
                <span class="tier-label">Bounty Hunter Tier:</span>
                <span id="bounty-hunter-tier-value" class="tier-value tier-${getTierName(hunterTier)}">${getTierLabel(hunterTier)}</span>
                ${hunterTier > 0 ? `<span class="tracking-bonus active">×${(2 * hunterTier).toFixed(1)} Tracking Bonus</span>` : '<span class="tracking-bonus inactive">Upgrade to unlock tracking bonus</span>'}
            </div>
            
            <!-- Bounty List -->
            <div id="bounty-list" class="bounty-list-container">
                ${activeBounties.length === 0 ? 
                    '<div class="empty-state bounty-empty"><p>No active bounties. Purchase a Bounty License to begin tracking.</p></div>' : 
                    renderActiveBounties()
                }
            </div>
            
            <!-- Quick Actions -->
            <div class="bounty-actions">
                <button id="btn-refresh-bounties" class="bounty-action-btn refresh-btn">🔄 Refresh</button>
                <button id="btn-buy-license" class="bounty-action-btn license-btn">📋 Buy License (50K VBV)</button>
            </div>
        `;
        
        // Attach event listeners
        const closeBtn = overlay.querySelector('.bounty-close-btn');
        if (closeBtn) {
            closeBtn.addEventListener('click', () => toggleOverlay(false));
        }
        
        const refreshBtn = overlay.getElementById('btn-refresh-bounties');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => BountyTracker.refreshBounties());
        }
        
        const licenseBtn = overlay.getElementById('btn-buy-license');
        if (licenseBtn) {
            licenseBtn.addEventListener('click', () => BountyTracker.purchaseLicense());
        }
        
        // Append to game container
        const gameContainer = document.querySelector('.game-container, #game-app, body');
        if (gameContainer) {
            gameContainer.appendChild(overlay);
        }
        
        return overlay;
    }
    
    function getTierName(tier) {
        const tiers = ['Peon', 'Apprentice', 'Journeyman', 'Expert', 'Master', 'Boss'];
        return tier < 0 ? 'Unranked' : (tier >= tiers.length ? 'Boss' : tiers[tier]);
    }
    
    function getTierLabel(tier) {
        const labels = ['Peon (×1.0)', 'Apprentice (×2.0)', 'Journeyman (×4.0)', 
                        'Expert (×8.0)', 'Master (×16.0)', 'Boss (×32.0)'];
        return tier < 0 ? labels[0] : (tier >= labels.length ? labels[label.length - 1] : labels[tier]);
    }
    
    function renderActiveBounties() {
        if (!activeBounties || activeBounties.length === 0) {
            return '<div class="empty-state bounty-empty"><p>No active bounties. Purchase a Bounty License to begin tracking.</p></div>';
        }
        
        let html = '';
        for (let i = 0; i < activeBounties.length; i++) {
            const bounty = activeBounties[i];
            const enhancedValue = Math.round(bounty.value * trackingBonusMultiplier);
            
            html += `
                <div class="bounty-card" data-bounty-id="${bounty.id}">
                    <div class="bounty-target-info">
                        <span class="target-name">${escapeHtml(bounty.targetName || bounty.walletPrefix)}</span>
                        <span class="wallet-prefix">[${bounty.walletPrefix || 'N/A'}]</span>
                    </div>
                    <div class="bounty-stats">
                        <span class="bounty-value ${enhancedValue > bounty.value ? 'enhanced' : ''}">
                            💰 ${formatNumber(enhancedValue)} VBV 
                            ${enhancedValue > bounty.value ? '<span class="bonus-indicator">+${formatNumber(enhancedValue - bounty.value)}</span>' : ''}
                        </span>
                    </div>
                    <div class="bounty-timer">
                        ⏱️ <span id="bounty-timer-${bounty.id}">${formatTimeRemaining(bounty.remainingMs || 0)}</span>
                    </div>
                    ${bounty.trackerDroneActive ? '<span class="drone-badge active-drone">🛰️ Drone Active</span>' : ''}
                    <button class="capture-btn" data-bounty-id="${bounty.id}" title="Capture bounty reward">⚔️ Capture</button>
                </div>
            `;
        }
        
        return html;
    }
    
    // ============================================================================
    // PUBLIC API — BountyTracker module interface
    // ============================================================================
    
    /**
     * Initialize the bounty tracker overlay.
     * Called when player has Bounty Hunter career role or purchases a license.
     */
    BountyTracker.init = function(playerStats) {
        if (!playerStats || !playerStats.CareerXP) return;
        
        const careerXP = playerStats.CareerXP;
        hunterTier = getBountyHunterTier(careerXP);
        trackingBonusMultiplier = 1 + (hunterTier * 0.5); // ×1 at Peon, ×2.5+ scaling
        
        updateTierDisplay();
        
        // Fetch active bounties from server if player has role
        if (playerStats.JobRole === 'Bounty Hunter' || hunterTier > 0) {
            BountyTracker.refreshBounties();
            
            // Connect to WebSocket for live bounty updates
            connectBountyWebSocket();
        }
    };
    
    /**
     * Refresh active bounties from server.
     */
    BountyTracker.refreshBounties = function() {
        fetch('/api/bounty/active', {
            method: 'GET',
            headers: { 'Content-Type': 'application/json' },
            credentials: 'include'
        })
        .then(resp => resp.json())
        .then(data => {
            if (data.bounties) {
                activeBounties = data.bounties;
                updateBountyList();
            }
        })
        .catch(err => console.error('BountyTracker: refresh failed:', err));
    };
    
    /**
     * Purchase a bounty license for tracking.
     */
    BountyTracker.purchaseLicense = function(targetWallet) {
        if (!targetWallet) {
            targetWallet = prompt('Enter target wallet address to track:');
            if (!targetWallet) return;
        }
        
        fetch('/api/shop/purchase', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            credentials: 'include',
            body: JSON.stringify({ itemId: 'bounty_license', targetWallet: targetWallet })
        })
        .then(resp => resp.json())
        .then(data => {
            if (data.success) {
                showNotification('Bounty License purchased! Tracking active.', 'success');
                BountyTracker.refreshBounties();
            } else {
                showNotification(data.error || 'Failed to purchase license', 'error');
            }
        })
        .catch(err => console.error('BountyTracker: purchase failed:', err));
    };
    
    /**
     * Capture bounty reward from tracked target.
     */
    BountyTracker.captureBounty = function(bountyId) {
        fetch('/api/justice/capture-bounty', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            credentials: 'include',
            body: JSON.stringify({ bountyId: bountyId })
        })
        .then(resp => resp.json())
        .then(data => {
            if (data.success) {
                showNotification(`Bounty captured! +${formatNumber(data.reward || 0)} VBV`, 'success');
                // Remove from active list and refresh display
                activeBounties = activeBounties.filter(b => b.id !== bountyId);
                updateBountyList();
            } else {
                showNotification(data.error || 'Capture failed', 'error');
            }
        })
        .catch(err => console.error('BountyTracker: capture failed:', err));
    };
    
    /**
     * Toggle overlay visibility.
     */
    BountyTracker.toggle = function() {
        const container = getOverlayContainer();
        if (container) {
            container.classList.toggle('hidden');
        }
    };
    
    // ============================================================================
    // INTERNAL HELPERS — Private functions
    // ============================================================================
    
    function updateTierDisplay() {
        const tierValueEl = document.getElementById('bounty-hunter-tier-value');
        if (tierValueEl) {
            tierValueEl.textContent = getTierLabel(hunterTier);
            tierValueEl.className = `tier-value tier-${getTierName(hunterTier).toLowerCase()}`;
        }
        
        const bonusEl = document.querySelector('.tracking-bonus');
        if (bonusEl) {
            if (hunterTier > 0) {
                bonusEl.textContent = `×${(2 * hunterTier).toFixed(1)} Tracking Bonus`;
                bonusEl.className = 'tracking-bonus active';
            } else {
                bonusEl.textContent = 'Upgrade to unlock tracking bonus';
                bonusEl.className = 'tracking-bonus inactive';
            }
        }
    }
    
    function updateBountyList() {
        const listEl = document.getElementById('bounty-list');
        if (listEl) {
            listEl.innerHTML = renderActiveBounties();
            
            // Attach capture button listeners
            const captureBtns = listEl.querySelectorAll('.capture-btn');
            for (let btn of captureBtns) {
                btn.addEventListener('click', function() {
                    const bountyId = this.getAttribute('data-bounty-id');
                    BountyTracker.captureBounty(bountyId);
                });
            }
        }
    }
    
    function connectBountyWebSocket() {
        if (wsConnection && wsConnection.readyState === WebSocket.OPEN) return; // Already connected
        
        const protocol = window.location.protocol === 'https:' ? 'wss://' : 'ws://';
        const host = window.location.host || 'localhost:8080';
        const wsUrl = `${protocol}${host}/ws/bounty`;
        
        try {
            wsConnection = new WebSocket(wsUrl);
            
            wsConnection.onmessage = function(event) {
                try {
                    const msg = JSON.parse(event.data);
                    handleBountyWebSocketMessage(msg);
                } catch (e) {
                    console.error('BountyTracker: WS message parse error:', e);
                }
            };
            
            wsConnection.onclose = function() {
                // Reconnect after 5 seconds if not manually closed
                setTimeout(() => connectBountyWebSocket(), 5000);
            };
        } catch (e) {
            console.error('BountyTracker: WS connection failed:', e);
        }
    }
    
    function handleBountyWebSocketMessage(msg) {
        switch (msg.type) {
            case 'bounty_updated':
                if (msg.bounty) {
                    // Update or add bounty to active list
                    const idx = activeBounties.findIndex(b => b.id === msg.bounty.id);
                    if (idx >= 0) {
                        activeBounties[idx] = msg.bounty;
                    } else {
                        activeBounties.push(msg.bounty);
                    }
                    updateBountyList();
                }
                break;
                
            case 'bounty_captured':
                // Remove captured bounty and show notification
                activeBounties = activeBounties.filter(b => b.id !== msg.bountyId);
                updateBountyList();
                showNotification(`Target ${msg.targetWalletPrefix} bounty captured!`, 'info');
                break;
                
            case 'bounty_expired':
                // Remove expired bounty
                activeBounties = activeBounties.filter(b => b.id !== msg.bountyId);
                updateBountyList();
                showNotification('Tracking drone has expired.', 'warning');
                break;
        }
    }
    
    function getBountyHunterTier(careerXP) {
        if (!careerXP || !careerXP.RoleXP) return 0;
        const xp = careerXP.RoleXP['Bounty Hunter'] || 0;
        
        // Match tier thresholds from rival_career_engine.go (CareerTier constants)
        if (xp >= 75) return 6;   // Boss
        if (xp >= 50) return 5;   // Master
        if (xp >= 30) return 4;   // Expert
        if (xp >= 15) return 3;   // Journeyman
        if (xp >= 5) return 2;    // Apprentice
        if (xp > 0) return 1;     // Peon
        return 0;                  // Unranked
    }
    
    function formatNumber(n) {
        if (!n && n !== 0) return '0';
        return Number(n).toLocaleString('en-US');
    }
    
    function formatTimeRemaining(ms) {
        if (!ms || ms <= 0) return 'Expired';
        const seconds = Math.floor(ms / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);
        
        if (hours > 0) {
            return `${hours}h ${minutes % 60}m`;
        }
        return `${minutes}m ${seconds % 60}s`;
    }
    
    function escapeHtml(str) {
        if (!str) return '';
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }
    
    // Global notification helper (falls back to console if game UI not available)
    function showNotification(message, type) {
        type = type || 'info';
        
        // Try using existing toast/notification system first
        if (typeof window.showToast === 'function') {
            window.showToast(message, type);
            return;
        }
        
        // Fallback: create temporary notification element
        const notif = document.createElement('div');
        notif.className = `bounty-notification bounty-notif-${type}`;
        notif.textContent = message;
        notif.style.cssText = `
            position: fixed; top: 20px; right: 20px; z-index: 10000;
            padding: 12px 24px; border-radius: 8px; color: #fff; font-size: 14px;
            background: ${type === 'success' ? '#2ecc71' : type === 'error' ? '#e74c3c' : type === 'warning' ? '#f39c12' : '#3498db'};
            box-shadow: 0 4px 16px rgba(0,0,0,0.3); opacity: 0; transition: opacity 0.3s ease;
        `;
        
        document.body.appendChild(notif);
        
        // Fade in/out animation
        requestAnimationFrame(() => { notif.style.opacity = '1'; });
        setTimeout(() => { 
            notif.style.opacity = '0'; 
            setTimeout(() => notif.remove(), 300); 
        }, 4000);
    }
    
    // ============================================================================
    // INITIALIZATION — Auto-init when DOM is ready
    // ============================================================================
    
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', () => {
            console.log('[BountyTracker] Module loaded. Awaiting player stats for initialization.');
        });
    } else {
        document.addEventListener('DOMContentLoaded', () => {
            console.log('[BountyTracker] Module ready (no DOM wait needed).');
        });
    }
    
    // Expose module globally
    window.BountyTracker = BountyTracker;
    
})();