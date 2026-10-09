// ============================================================================
// game_modes.js — Casino, Dive Bar & Lounge Game Modes
// ----------------------------------------------------------------------------
// Ported from Triple Triad: casino.js + dive-bar.js + lounge.js
// Adds 3 new game modes to the Match Creator:
// - Casino: High-stakes wagering with High-Low card game
// - Dice Bar: Dice rolling mini-game with wagers
// - VIP Lounge: Premium ranked with relationship boosts
// ============================================================================

var API_BASE = '/api';

// --- Casino: High-Low Game ---
function playHighLow(wager) {
    const playerCard = Math.floor(Math.random() * 13) + 1;
    const dealerCard = Math.floor(Math.random() * 13) + 1;
    const guess = Math.random() > 0.5 ? 'higher' : 'lower';
    const playerWins = (guess === 'higher' && playerCard > dealerCard) || (guess === 'lower' && playerCard < dealerCard);
    const push = playerCard === dealerCard;
    
    let result = { wager, playerCard, dealerCard, guess, win: false, payout: 0 };
    if (push) {
        result.payout = wager; // Return wager
        result.result = 'push';
    } else if (playerWins) {
        result.win = true;
        result.payout = wager * 2;
        result.result = 'won';
    } else {
        result.payout = 0;
        result.result = 'lost';
    }
    return result;
}

// --- Dice Bar: Dice Game ---
function playDiceGame(wager) {
    const playerRoll1 = Math.floor(Math.random() * 6) + 1;
    const playerRoll2 = Math.floor(Math.random() * 6) + 1;
    const playerTotal = playerRoll1 + playerRoll2;
    
    let dealerRoll1 = Math.floor(Math.random() * 6) + 1;
    let dealerRoll2 = Math.floor(Math.random() * 6) + 1;
    let dealerTotal = dealerRoll1 + dealerRoll2;
    
    // Dealer advantage: re-roll if 4 or less
    if (dealerTotal <= 4) {
        dealerRoll1 = Math.floor(Math.random() * 6) + 1;
        dealerRoll2 = Math.floor(Math.random() * 6) + 1;
        dealerTotal = dealerRoll1 + dealerRoll2;
    }
    
    let result = { wager, playerRoll1, playerRoll2, playerTotal, dealerRoll1, dealerRoll2, dealerTotal, win: false, payout: 0 };
    if (playerTotal > dealerTotal) {
        result.win = true;
        result.payout = wager * 2;
        result.result = 'won';
    } else if (playerTotal < dealerTotal) {
        result.payout = 0;
        result.result = 'lost';
    } else {
        result.payout = wager;
        result.result = 'push';
    }
    return result;
}

// --- VIP Lounge: Relationship Drinks ---
const DRINK_OPTIONS = [
    { id: 'cheap_beer', name: 'Cheap Beer', icon: '🍺', cost: 500, boost: 5, desc: 'Small relationship boost' },
    { id: 'house_cocktail', name: 'House Special', icon: '🍸', cost: 2000, boost: 10, desc: 'Varies by mood' },
    { id: 'top_shelf', name: 'Top-Shelf Vintage', icon: '🥃', cost: 8000, boost: 20, desc: 'Significant boost + tip chance' }
];

// State
let selectedMode = 'casino';
let wagerAmount = 100;
let lastGameResult = null;
let gameHistory = [];

// --- Init ---
export function initGameModes() {
    const el = document.getElementById('wd-game_modes');
    if (!el) return;
    renderGameModes(el);
}

function renderGameModes(el) {
    el.innerHTML = `
        <div class="game-modes-container">
            <div class="gm-header">
                <span class="gm-icon">🎰</span>
                <div>
                    <h3 style="color:#e5e7eb;margin:0;">Game Modes</h3>
                    <p style="color:#90a4ae;font-size:11px;margin:2px 0 0;">Casino, Dice Bar, and VIP Lounge</p>
                </div>
            </div>

            <!-- Mode Selection -->
            <div class="gm-modes">
                <div class="gm-mode ${selectedMode === 'casino' ? 'active' : ''}" data-mode="casino">
                    <span class="gm-mode-icon">🎰</span>
                    <span class="gm-mode-name">Casino</span>
                    <span class="gm-mode-desc">High-Low card game</span>
                </div>
                <div class="gm-mode ${selectedMode === 'dice' ? 'active' : ''}" data-mode="dice">
                    <span class="gm-mode-icon">🎲</span>
                    <span class="gm-mode-name">Dice Bar</span>
                    <span class="gm-mode-desc">Roll against the house</span>
                </div>
                <div class="gm-mode ${selectedMode === 'lounge' ? 'active' : ''}" data-mode="lounge">
                    <span class="gm-mode-icon">🛋️</span>
                    <span class="gm-mode-name">VIP Lounge</span>
                    <span class="gm-mode-desc">Premium ranked + drinks</span>
                </div>
            </div>

            <!-- Casino Panel -->
            ${selectedMode === 'casino' ? renderCasinoPanel() : ''}

            <!-- Dice Bar Panel -->
            ${selectedMode === 'dice' ? renderDicePanel() : ''}

            <!-- VIP Lounge Panel -->
            ${selectedMode === 'lounge' ? renderLoungePanel() : ''}

            <!-- Result Display -->
            ${lastGameResult ? `
                <div class="gm-section">
                    <h4 style="color:#e5e7eb;">Last Result</h4>
                    <div class="gm-result ${lastGameResult.result}">
                        <span class="gm-result-text">${getResultText()}</span>
                        <span class="gm-result-payout">${lastGameResult.payout > 0 ? '+' + lastGameResult.payout + ' SP' : '-' + lastGameResult.wager + ' SP'}</span>
                    </div>
                </div>
            ` : ''}

            <!-- Game History -->
            <div class="gm-section">
                <h4 style="color:#e5e7eb;">History</h4>
                <div class="gm-history">
                    ${gameHistory.length > 0 ? gameHistory.slice(0, 5).map(g => `
                        <div class="gm-history-item ${g.result}">
                            <span class="gm-history-mode">${g.mode}</span>
                            <span class="gm-history-result">${g.result}</span>
                            <span class="gm-history-payout">${g.payout > 0 ? '+' + g.payout : '-' + g.wager}</span>
                        </div>
                    `).join('') : '<p style="color:#90a4ae;">No games played yet</p>'}
                </div>
            </div>
        </div>
    `;
    attachGameModesListeners(el);
}

function renderCasinoPanel() {
    return `
        <div class="gm-section">
            <h4 style="color:#e5e7eb;">Casino: High-Low</h4>
            <p style="color:#90a4ae;font-size:11px;">Guess if your card is higher or lower than the dealer's. Win 2x your wager!</p>
            <div class="gm-wager">
                <label>Wager:</label>
                <input type="number" id="gm-wager-input" value="${wagerAmount}" min="10" max="10000" />
            </div>
            <div class="gm-actions">
                <button class="vbt-btn vbt-btn-primary gm-play-btn" data-game="high">⬆️ Higher</button>
                <button class="vbt-btn vbt-btn-primary gm-play-btn" data-game="low">⬇️ Lower</button>
            </div>
        </div>
    `;
}

function renderDicePanel() {
    return `
        <div class="gm-section">
            <h4 style="color:#e5e7eb;">Dice Bar</h4>
            <p style="color:#90a4ae;font-size:11px;">Roll two dice against the house. Higher total wins 2x your wager!</p>
            <div class="gm-wager">
                <label>Wager:</label>
                <input type="number" id="gm-wager-input" value="${wagerAmount}" min="10" max="5000" />
            </div>
            <div class="gm-actions">
                <button class="vbt-btn vbt-btn-primary gm-play-btn" data-game="dice">🎲 Roll Dice</button>
            </div>
        </div>
    `;
}

function renderLoungePanel() {
    return `
        <div class="gm-section">
            <h4 style="color:#e5e7eb;">VIP Lounge</h4>
            <p style="color:#90a4ae;font-size:11px;">Buy drinks to boost relationships with NPCs</p>
            <div class="gm-drinks">
                ${DRINK_OPTIONS.map(d => `
                    <div class="gm-drink" data-id="${d.id}" data-cost="${d.cost}">
                        <span class="gm-drink-icon">${d.icon}</span>
                        <span class="gm-drink-name">${d.name}</span>
                        <span class="gm-drink-desc">${d.desc}</span>
                        <span class="gm-drink-cost">${d.cost} SP</span>
                        <span class="gm-drink-boost">+${d.boost} relationship</span>
                        <button class="vbt-btn vbt-btn-secondary gm-drink-btn" data-id="${d.id}" data-cost="${d.cost}">Buy</button>
                    </div>
                `).join('')}
            </div>
        </div>
        <div class="gm-section">
            <h4 style="color:#e5e7eb;">Premium Ranked</h4>
            <p style="color:#90a4ae;font-size:11px;">Queue for high-stakes ranked matches with enhanced rewards</p>
            <button class="vbt-btn vbt-btn-primary gm-queue-btn">🏆 Queue for Ranked (1000 SP)</button>
        </div>
    `;
}

function getResultText() {
    if (!lastGameResult) return '';
    if (lastGameResult.mode === 'casino') {
        return `Card: ${lastGameResult.playerCard} vs Dealer: ${lastGameResult.dealerCard} (${lastGameResult.guess})`;
    } else if (lastGameResult.mode === 'dice') {
        return `Roll: ${lastGameResult.playerTotal} (${lastGameResult.playerRoll1}+${lastGameResult.playerRoll2}) vs Dealer: ${lastGameResult.dealerTotal}`;
    }
    return '';
}

function attachGameModesListeners(el) {
    // Mode selection
    el.querySelectorAll('.gm-mode').forEach(mode => {
        mode.addEventListener('click', () => {
            selectedMode = mode.dataset.mode;
            lastGameResult = null;
            renderGameModes(el);
        });
    });

    // Wager input
    const wagerInput = el.querySelector('#gm-wager-input');
    if (wagerInput) {
        wagerInput.addEventListener('change', () => {
            wagerAmount = parseInt(wagerInput.value) || 100;
        });
    }

    // Play buttons
    el.querySelectorAll('.gm-play-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const game = btn.dataset.game;
            let result;
            if (game === 'high') {
                result = playHighLow(wagerAmount);
                result.guess = 'higher';
            } else if (game === 'low') {
                result = playHighLow(wagerAmount);
                result.guess = 'lower';
            } else if (game === 'dice') {
                result = playDiceGame(wagerAmount);
            }
            result.mode = selectedMode;
            lastGameResult = result;
            gameHistory.unshift(result);
            if (gameHistory.length > 20) gameHistory.pop();
            renderGameModes(el);
            
            if (result.win) {
                if (window.showToast) window.showToast(`🎉 Won ${result.payout} SP!`, 'success');
            } else if (result.result === 'push') {
                if (window.showToast) window.showToast(`Push! Wager returned.`, 'info');
            } else {
                if (window.showToast) window.showToast(`Lost ${result.wager} SP.`, 'error');
            }
        });
    });

    // Drink buttons
    el.querySelectorAll('.gm-drink-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const cost = parseInt(btn.dataset.cost);
            if (window.showToast) window.showToast(`Bought drink for ${cost} SP! Relationship boosted.`, 'success');
        });
    });

    // Queue button
    const queueBtn = el.querySelector('.gm-queue-btn');
    if (queueBtn) {
        queueBtn.addEventListener('click', () => {
            if (window.showToast) window.showToast('🏆 Queuing for premium ranked match...', 'info');
        });
    }
}

// --- Globals ---
window.initGameModes = initGameModes;
