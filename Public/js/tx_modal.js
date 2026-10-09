// ============================================================================
// tx_modal.js — Reusable Transaction Confirmation
// ============================================================================

(function () {
    'use strict';
    let modalEl = null;
    let currentAction = null;

    function init() {
        if (modalEl) return;
        modalEl = document.getElementById('tx-modal-overlay');
        if (!modalEl) return;
        modalEl.className = 'overlay tx-modal-overlay';
        modalEl.style.display = 'none';

        modalEl.innerHTML = `
            <div class="tx-modal">
                <button class="tx-modal-close" onclick="window.txModalCancel()">✕</button>
                
                <div class="tx-modal-confirm">
                    <div class="tx-modal-header">
                        <h3>Confirm Transaction</h3>
                    </div>
                    <div class="tx-modal-body">
                        <div class="tx-modal-action" id="tx-action-name">Action</div>
                        <div class="tx-modal-cost">
                            <span class="tx-cost-label">Cost:</span>
                            <span class="tx-cost-amount" id="tx-cost-amount">0 VBV</span>
                        </div>
                    </div>
                    <div class="tx-modal-actions">
                        <button class="vbt-btn vbt-btn-secondary" onclick="window.txModalCancel()">Cancel</button>
                        <button class="vbt-btn vbt-btn-primary" id="tx-confirm-btn" onclick="window.txModalConfirm()">Confirm</button>
                    </div>
                </div>

                <div class="tx-modal-loading" style="display:none;">
                    <div class="tx-spinner"></div>
                    <span>Awaiting confirmation...</span>
                </div>

                <div class="tx-modal-success" style="display:none;">
                    <div class="tx-checkmark">✓</div>
                    <span>Transaction confirmed</span>
                </div>

                <div class="tx-modal-error" style="display:none;">
                    <div class="tx-error-icon">✕</div>
                    <span id="tx-error-text">Transaction failed</span>
                </div>
            </div>`;
    }

    function open(actionName, costVBV, onConfirm) {
        init();
        currentAction = { actionName, costVBV, onConfirm };
        
        modalEl.style.display = 'flex';
        resetState();

        const actionEl = document.getElementById('tx-action-name');
        const costEl = document.getElementById('tx-cost-amount');
        if (actionEl) actionEl.textContent = actionName;
        if (costEl) costEl.textContent = costVBV.toFixed(2) + ' VBV';
    }

    function cancel() {
        if (modalEl) modalEl.style.display = 'none';
        currentAction = null;
    }

    function resetState() {
        if (!modalEl) return;
        modalEl.querySelector('.tx-modal-confirm').style.display = 'block';
        modalEl.querySelector('.tx-modal-loading').style.display = 'none';
        modalEl.querySelector('.tx-modal-success').style.display = 'none';
        modalEl.querySelector('.tx-modal-error').style.display = 'none';
    }

    function confirm() {
        if (!modalEl || !currentAction) return;

        modalEl.querySelector('.tx-modal-confirm').style.display = 'none';
        modalEl.querySelector('.tx-modal-loading').style.display = 'flex';

        setTimeout(() => {
            const success = Math.random() > 0.15;
            
            if (success) {
                modalEl.querySelector('.tx-modal-loading').style.display = 'none';
                modalEl.querySelector('.tx-modal-success').style.display = 'flex';
                
                setTimeout(() => {
                    cancel();
                    if (currentAction && currentAction.onConfirm) {
                        currentAction.onConfirm();
                    }
                    showToast('Transaction confirmed', 'success');
                }, 1200);
            } else {
                modalEl.querySelector('.tx-modal-loading').style.display = 'none';
                modalEl.querySelector('.tx-modal-error').style.display = 'flex';
                const errorEl = document.getElementById('tx-error-text');
                if (errorEl) errorEl.textContent = 'Transaction failed. Insufficient balance or network error.';
                
                setTimeout(() => {
                    cancel();
                    showToast('Transaction failed', 'error');
                }, 2000);
            }
        }, 2000);
    }

    function showToast(message, type) {
        if (typeof window.showToast === 'function') {
            window.showToast(message, type);
        }
    }

    window.txModalOpen = open;
    window.txModalCancel = cancel;
    window.txModalConfirm = confirm;

    if (document.getElementById('tx-modal-overlay')) init();
})();
