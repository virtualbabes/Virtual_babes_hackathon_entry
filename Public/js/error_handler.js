// ============================================================================
// error_handler.js — Graceful error handling for all async operations
// ----------------------------------------------------------------------------
// API failures: cached data, retry banners, per-panel error states
// Wallet/transaction: disconnect detection, insufficient balance, user rejection
// Battle edge cases: opponent disconnect, timeout, forfeit
// ============================================================================

(function () {
    'use strict';
    var API_BASE = '/api';
    let errorBanner = null;
    let retryCallbacks = {};

    function init() {
        // Create persistent error banner at top of page
        if (!errorBanner) {
            errorBanner = document.createElement('div');
            errorBanner.id = 'global-error-banner';
            errorBanner.className = 'error-banner hidden';
            document.body.prepend(errorBanner);
        }
    }

    // === API ERROR HANDLING ===
    function handleApiError(error, retryCallback, panelId) {
        console.warn('API Error:', error.message);
        
        // Show cached data if available
        const cached = getCachedData(panelId);
        if (cached) {
            showErrorBanner('Using cached data — connection issue', 'warning');
            return cached;
        }

        // Show error in specific panel
        if (panelId) {
            showPanelError(panelId, error.message, retryCallback);
        } else {
            showErrorBanner('Connection issue — retrying...', 'error');
        }

        // Auto-retry after 5s
        if (retryCallback) {
            const retryId = setTimeout(() => {
                retryCallback();
            }, 5000);
            retryCallbacks[panelId || 'global'] = retryId;
        }

        return null;
    }

    function showPanelError(panelId, message, retryCallback) {
        const panel = document.getElementById(panelId);
        if (!panel) return;

        panel.innerHTML = `
            <div class="error-state">
                <div class="error-icon">⚠️</div>
                <div class="error-title">Connection Issue</div>
                <div class="error-message">${escapeHtml(message)}</div>
                <button class="vbt-btn vbt-btn-primary retry-btn" onclick="window.errorRetry('${panelId}')">Retry</button>
            </div>`;

        if (retryCallback) {
            retryCallbacks[panelId] = retryCallback;
        }
    }

    function retry(panelId) {
        const callback = retryCallbacks[panelId];
        if (callback) {
            callback();
            delete retryCallbacks[panelId];
        }
    }

    function getCachedData(panelId) {
        try {
            const cached = localStorage.getItem('cache_' + panelId);
            return cached ? JSON.parse(cached) : null;
        } catch (e) { return null; }
    }

    function cacheData(panelId, data) {
        try {
            localStorage.setItem('cache_' + panelId, JSON.stringify(data));
        } catch (e) {}
    }

    // === ERROR BANNER ===
    function showErrorBanner(message, type) {
        init();
        errorBanner.textContent = message;
        errorBanner.className = 'error-banner ' + (type || 'error');
        errorBanner.classList.remove('hidden');
        
        setTimeout(() => {
            errorBanner.classList.add('hidden');
        }, 5000);
    }

    function hideErrorBanner() {
        if (errorBanner) errorBanner.classList.add('hidden');
    }

    // === WALLET/TRANSACTION ERROR HANDLING ===
    function handleWalletDisconnect() {
        showErrorBanner('Wallet disconnected', 'warning');
        
        // Revert UI to pre-connect state
        const displays = document.querySelectorAll('.wallet-address-display');
        displays.forEach(el => el.textContent = '0x----...----');
        
        // Disable interactive buttons
        const actionBtns = document.querySelectorAll('.vbt-btn-primary:not(.wallet-option-btn)');
        actionBtns.forEach(btn => btn.disabled = true);
        
        // Show wallet modal again
        if (typeof window.walletModalOpen === 'function') {
            setTimeout(() => window.walletModalOpen(), 1000);
        }
    }

    function handleTransactionError(error) {
        if (error.message && error.message.includes('User rejected')) {
            showErrorBanner('Transaction cancelled', 'neutral');
            if (typeof window.audioEngine !== 'undefined') window.audioEngine.error();
        } else if (error.message && error.message.includes('Insufficient')) {
            showErrorBanner('Insufficient VBV', 'error');
            if (typeof window.audioEngine !== 'undefined') window.audioEngine.error();
        } else {
            showErrorBanner('Transaction failed: ' + (error.message || 'Unknown error'), 'error');
            if (typeof window.audioEngine !== 'undefined') window.audioEngine.error();
        }
    }

    // === BATTLE EDGE CASES ===
    function handleOpponentDisconnect() {
        showErrorBanner('Opponent disconnected — you win by forfeit!', 'success');
        if (typeof window.audioEngine !== 'undefined') window.audioEngine.victory();
        
        // Award victory
        setTimeout(() => {
            showErrorBanner('+500 VBV (forfeit reward)', 'success');
        }, 2000);
    }

    function handleBattleTimeout() {
        showErrorBanner("Time's up — draw", 'neutral');
        if (typeof window.audioEngine !== 'undefined') window.audioEngine.error();
        
        // Small consolation
        setTimeout(() => {
            showErrorBanner('+50 VBV (consolation)', 'success');
        }, 2000);
    }

    function handlePlayerDisconnect() {
        // Show reconnection window
        showErrorBanner('Disconnected — reconnect within 10s...', 'warning');
        
        let seconds = 10;
        const interval = setInterval(() => {
            seconds--;
            if (seconds <= 0) {
                clearInterval(interval);
                showErrorBanner('Forfeit — opponent wins', 'error');
                if (typeof window.audioEngine !== 'undefined') window.audioEngine.defeat();
            }
        }, 1000);
    }

    function escapeHtml(s) {
        return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
    }

    window.errorHandler = {
        apiError: handleApiError,
        walletDisconnect: handleWalletDisconnect,
        transactionError: handleTransactionError,
        opponentDisconnect: handleOpponentDisconnect,
        battleTimeout: handleBattleTimeout,
        playerDisconnect: handlePlayerDisconnect,
        showBanner: showErrorBanner,
        hideBanner: hideErrorBanner,
        retry: retry,
        cacheData: cacheData,
        getCachedData: getCachedData,
    };

    // The error state this module renders offers a RETRY button whose handler names the GLOBAL
    // `window.errorRetry`, while the function lives here as `retry`. An inline handler resolves its
    // name on `window`, so without this line the button threw ReferenceError and a failed panel
    // could never be retried — the 5 s auto-retry was the only way back.
    window.errorRetry = retry;

    if (document.body) init();
})();
