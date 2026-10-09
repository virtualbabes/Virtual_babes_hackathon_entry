// ============================================================================
// post_match_wagers.js — Post-Match Wager Resolution
// ----------------------------------------------------------------------------
// Ported from Triple Triad: post-game-wagers.js
// Handles wager payouts, bounty claims, bounty survival
// Connects to: Economy Service, Black Market, Match Creator
// ============================================================================

var API_BASE = '/api';

// State
let activeWager = 0;
let activeBounty = null;
let wagerHistory = [];

// --- Init ---
export function initPostMatchWagers() {
    const el = document.getElementById('wd-wagers');
    if (!el) return;
    renderPostMatchWagers(el);
}

function renderPostMatchWagers(el) {
    el.innerHTML = `
        <div class="wagers-container">
            <div class="wagers-header">
                <span class="wagers-icon">💰</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Wagers & Bounties</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Track wagers, claim bounties, survive attacks</p>
                </div>
            </div>

            <!-- Active Wager -->
            <div class="wagers-section">
                <h4 style="color:#e5e7eb;">Active Wager</h4>
                <div class="wagers-active">
                    ${activeWager > 0 ? `
                        <span class="wagers-amount">${activeWager} SP</span>
                        <span class="wagers-status pending">Pending</span>
                    ` : '<p style="color:#90a4ae;">No active wager</p>'}
                </div>
            </div>

            <!-- Active Bounty -->
            <div class="wagers-section">
                <h4 style="color:#e5e7eb;">Active Bounty</h4>
                <div class="wagers-bounty">
                    ${activeBounty ? `
                        <span class="wagers-bounty-target">${activeBounty.target}</span>
                        <span class="wagers-bounty-amount">${activeBounty.amount} SP</span>
                        <span class="wagers-bounty-status ${activeBounty.status}">${activeBounty.status}</span>
                    ` : '<p style="color:#90a4ae;">No active bounty</p>'}
                </div>
            </div>

            <!-- Wager History -->
            <div class="wagers-section">
                <h4 style="color:#e5e7eb;">Wager History</h4>
                <div class="wagers-history">
                    ${wagerHistory.length > 0 ? wagerHistory.map(w => `
                        <div class="wagers-history-item ${w.result}">
                            <span class="wagers-history-amount">${w.amount} SP</span>
                            <span class="wagers-history-result">${w.result}</span>
                            <span class="wagers-history-date">${w.date}</span>
                        </div>
                    `).join('') : '<p style="color:#90a4ae;">No wager history</p>'}
                </div>
            </div>

            <!-- Quick Wager -->
            <div class="wagers-section">
                <h4 style="color:#e5e7eb;">Quick Wager</h4>
                <div class="wagers-quick">
                    <button class="vbt-btn vbt-btn-secondary" data-amount="100">100 SP</button>
                    <button class="vbt-btn vbt-btn-secondary" data-amount="500">500 SP</button>
                    <button class="vbt-btn vbt-btn-secondary" data-amount="1000">1000 SP</button>
                    <button class="vbt-btn vbt-btn-secondary" data-amount="5000">5000 SP</button>
                </div>
            </div>
        </div>
    `;
    attachWagerListeners(el);
}

function attachWagerListeners(el) {
    el.querySelectorAll('.wagers-quick button').forEach(btn => {
        btn.addEventListener('click', () => {
            const amount = parseInt(btn.dataset.amount);
            activeWager = amount;
            renderPostMatchWagers(el);
            if (window.showToast) window.showToast(`Wager set: ${amount} SP`, 'info');
        });
    });
}

// --- Wager Resolution (called from match end) ---
export function resolveWager(wagerData, winner) {
    const result = {
        amount: wagerData.wager || 0,
        result: 'pending',
        date: new Date().toLocaleDateString()
    };

    if (!wagerData || wagerData.wager <= 0) {
        return null;
    }

    if (winner === 'player') {
        result.result = 'won';
        result.payout = wagerData.wager * 2;
    } else if (winner === 'opponent') {
        result.result = 'lost';
        result.payout = 0;
    } else {
        result.result = 'draw';
        result.payout = wagerData.wager;
    }

    wagerHistory.unshift(result);
    if (wagerHistory.length > 10) wagerHistory.pop();

    activeWager = 0;
    return result;
}

// --- Bounty Resolution ---
export function resolveBounty(matchData, winner) {
    if (winner !== 'player') return null;

    // Bounty survival
    if (matchData.activePlayerBounty && matchData.activePlayerBounty.placedBy === matchData.opponentName) {
        activeBounty = null;
        return { type: 'survived', message: `Bounty from ${matchData.opponentName} survived!` };
    }

    // Bounty claim
    if (activeBounty && activeBounty.target === matchData.opponentName) {
        const reward = activeBounty.amount * 2;
        activeBounty = null;
        return { type: 'claimed', amount: reward, message: `Bounty claimed! +${reward} SP` };
    }

    return null;
}

// --- Globals ---
window.initPostMatchWagers = initPostMatchWagers;
window.resolveWager = resolveWager;
window.resolveBounty = resolveBounty;
