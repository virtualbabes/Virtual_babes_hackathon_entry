// ============================================================================
// wallet_state.js — Centralized wallet state + VBV balance tracking
// ----------------------------------------------------------------------------
// Stores connected wallet address, VBV balance, and auto-adds X-Wallet-Address header.
// Provides canAfford() and deductVBV() for real economy enforcement.
// ============================================================================

(function () {
    'use strict';
    var API_BASE = '/api';
    let currentWallet = '';
    let vbvBalanceMicro = 0; // uint64 micro-units

    // Load saved wallet from storage
    try { currentWallet = localStorage.getItem('wallet_address') || ''; } catch(e) {}

    window.getWalletAddress = function() { return currentWallet; };
    window.isWalletConnected = function() { return !!currentWallet; };
    window.getVBVBalanceMicro = function() { return vbvBalanceMicro; };
    window.getVBVBalance = function() { return vbvBalanceMicro / 1000000; };

    window.setWalletAddress = function(addr) {
        currentWallet = addr ? String(addr).toLowerCase().trim() : '';
        try {
            if (currentWallet) localStorage.setItem('wallet_address', currentWallet);
            else localStorage.removeItem('wallet_address');
        } catch(e) {}
        refreshBalance();
    };

    window.setVBVBalanceMicro = function(micro) {
        vbvBalanceMicro = Math.max(0, parseInt(micro) || 0);
    };

    window.setVBVBalance = function(vbv) {
        vbvBalanceMicro = Math.max(0, Math.floor(parseFloat(vbv) * 1000000));
    };

    // REAL canAfford check
    window.canAfford = function(vbvAmount) {
        const amountMicro = Math.floor(parseFloat(vbvAmount) * 1000000);
        return vbvBalanceMicro >= amountMicro;
    };

    // REAL VBV deduction
    window.deductVBV = function(vbvAmount) {
        const amountMicro = Math.floor(parseFloat(vbvAmount) * 1000000);
        if (vbvBalanceMicro < amountMicro) return false;
        vbvBalanceMicro -= amountMicro;
        return true;
    };

    window.addVBV = function(vbvAmount) {
        const amountMicro = Math.floor(parseFloat(vbvAmount) * 1000000);
        vbvBalanceMicro += amountMicro;
    };

    // Refresh balance from server
    function refreshBalance() {
        if (!currentWallet) return;
        fetch(API_BASE + '/api/player/tokens?wallet=' + currentWallet)
            .then(r => r.ok ? r.json() : null)
            .then(d => {
                if (d && d.vbv_balance_micro) {
                    vbvBalanceMicro = parseInt(d.vbv_balance_micro) || 0;
                }
            })
            .catch(() => {});
    }

    // Centralized fetch wrapper that auto-injects wallet header
    window.api = function(url, options) {
        options = options || {};
        const headers = Object.assign({}, options.headers || {});
        if (currentWallet) {
            headers['X-Wallet-Address'] = currentWallet;
        }
        options.headers = headers;
        return fetch(url, options);
    };

    window.apiJSON = function(url, options) {
        return window.api(url, options).then(r => {
            if (!r.ok) return null;
            return r.json();
        });
    };

    // Auto-refresh balance on load
    if (currentWallet) refreshBalance();
})();
