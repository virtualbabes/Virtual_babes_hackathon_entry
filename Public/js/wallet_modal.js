// ============================================================================
// wallet_modal.js — Wallet Connection Front Door
// ----------------------------------------------------------------------------
// Glass modal with wallet options, loading states, success/error feedback.
// Post-connect: address display, dropdown with copy/disconnect.
// ============================================================================

(function () {
    'use strict';
    var API_BASE = '/api';
    let modalEl = null;
    let onConnectedCallback = null;

    function init() {
        if (modalEl) return;
        modalEl = document.getElementById('wallet-modal-overlay');
        if (!modalEl) return;
        modalEl.className = 'overlay wallet-modal-overlay';
        modalEl.style.display = 'none';

        modalEl.innerHTML = `
            <div class="wallet-modal">
                <button class="wallet-modal-close" onclick="window.walletModalClose()">✕</button>
                
                <div class="wallet-modal-header">
                    <h2>Connect Wallet</h2>
                    <p>Connect to begin your journey</p>
                </div>

                <div class="wallet-modal-options">
                    <button class="wallet-option-btn" data-wallet="metamask" onclick="window.walletConnect('metamask')">
                        <span class="wallet-icon">🦊</span>
                        <span class="wallet-name">MetaMask</span>
                        <span class="wallet-connect-text">Connect</span>
                    </button>
                    <button class="wallet-option-btn" data-wallet="walletconnect" onclick="window.walletConnect('walletconnect')">
                        <span class="wallet-icon">🔗</span>
                        <span class="wallet-name">WalletConnect</span>
                        <span class="wallet-connect-text">Connect</span>
                    </button>
                    <button class="wallet-option-btn" data-wallet="coinbase" onclick="window.walletConnect('coinbase')">
                        <span class="wallet-icon">🔵</span>
                        <span class="wallet-name">Coinbase Wallet</span>
                        <span class="wallet-connect-text">Connect</span>
                    </button>
                </div>

                <div class="wallet-modal-loading" style="display:none;">
                    <div class="wallet-spinner"></div>
                    <span>Awaiting connection...</span>
                </div>

                <div class="wallet-modal-success" style="display:none;">
                    <div class="wallet-checkmark">✓</div>
                    <span>Connected!</span>
                </div>

                <div class="wallet-modal-error" style="display:none;">
                    <span class="wallet-error-text">Connection failed. Try again.</span>
                </div>
            </div>`;
    }

    function open(callback) {
        init();
        onConnectedCallback = callback;
        modalEl.style.display = 'flex';
        resetState();
    }

    function close() {
        if (modalEl) modalEl.style.display = 'none';
    }

    function resetState() {
        if (!modalEl) return;
        modalEl.querySelector('.wallet-modal-options').style.display = 'flex';
        modalEl.querySelector('.wallet-modal-loading').style.display = 'none';
        modalEl.querySelector('.wallet-modal-success').style.display = 'none';
        modalEl.querySelector('.wallet-modal-error').style.display = 'none';
    }

    function connect(walletType) {
        if (!modalEl) return;
        
        // Show loading
        modalEl.querySelector('.wallet-modal-options').style.display = 'none';
        modalEl.querySelector('.wallet-modal-loading').style.display = 'flex';

        // Simulate connection (replace with actual wallet connection)
        setTimeout(() => {
            const success = Math.random() > 0.2; // 80% success rate for demo
            
            if (success) {
                modalEl.querySelector('.wallet-modal-loading').style.display = 'none';
                modalEl.querySelector('.wallet-modal-success').style.display = 'flex';
                
                // Store wallet
                const mockAddress = '0x' + Math.random().toString(16).slice(2, 10) + '...' + Math.random().toString(16).slice(2, 6);
                if (window.setWalletAddress) window.setWalletAddress(mockAddress);
                
                // Close after delay
                setTimeout(() => {
                    close();
                    if (onConnectedCallback) onConnectedCallback(mockAddress);
                    updateWalletDisplay(mockAddress);
                }, 1000);
            } else {
                modalEl.querySelector('.wallet-modal-loading').style.display = 'none';
                modalEl.querySelector('.wallet-modal-error').style.display = 'flex';
                
                setTimeout(() => {
                    resetState();
                }, 2000);
            }
        }, 1500);
    }

    function updateWalletDisplay(address) {
        // Update any wallet displays in the app
        const displays = document.querySelectorAll('.wallet-address-display');
        displays.forEach(el => el.textContent = address);
        
        // Update VBV balance (mock)
        const balances = document.querySelectorAll('.vbx-amount');
        balances.forEach(el => {
            if (el.id === 'constellation-vbv') {
                el.textContent = (Math.random() * 10000).toFixed(2);
            }
        });
    }

    function disconnect() {
        if (window.setWalletAddress) window.setWalletAddress('');
        const displays = document.querySelectorAll('.wallet-address-display');
        displays.forEach(el => el.textContent = '0x----...----');
    }

    function isConnected() {
        return !!(window.getWalletAddress && window.getWalletAddress());
    }

    function getAddress() {
        return (window.getWalletAddress && window.getWalletAddress()) || '';
    }

    window.walletModalOpen = open;
    window.walletModalClose = close;
    window.walletConnect = connect;
    window.walletDisconnect = disconnect;
    window.walletIsConnected = isConnected;
    window.walletGetAddress = getAddress;

    if (document.getElementById('wallet-modal-overlay')) init();
})();
