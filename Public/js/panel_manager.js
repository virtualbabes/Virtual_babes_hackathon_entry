// ============================================================================
// panel_manager.js — Centralized Panel Manager Utility
// ----------------------------------------------------------------------------
// Single source of truth for all overlay panels. Handles open/close, escape
// key, single-panel-at-a-time, and transition animations.
// ----------------------------------------------------------------------------
// RULES:
// - Only ONE panel open at a time
// - Escape key closes the active panel
// - All panels use contained-overlay pattern: position:fixed; inset:0; overflow:hidden
// - Panels animate in/out with cubic-bezier easing
// ============================================================================

(function () {
    'use strict';

    let activePanel = null;
    let activeCloseFn = null;
    let escHandler = null;
    let closeTimer = null;

    /**
     * Open a panel overlay.
     * @param {HTMLElement} el - The panel DOM element
     * @param {Function} [closeFn] - Custom close function (optional)
     */
    function openPanel(el, closeFn) {
        if (!el) return;

        // Capture any currently-open panel BEFORE closing it, so we can fully
        // neutralise it below (otherwise a stale panel can linger invisible).
        const prev = activePanel;

        // Close any currently open panel first
        closeActivePanel();

        // Cancel a pending deferred-hide (closeActivePanel schedules one on `prev`).
        // If we reopen the same element within 250ms the old timer would otherwise
        // fire later and hide the freshly-opened panel. See getActive/close path.
        if (closeTimer) {
            clearTimeout(closeTimer);
            closeTimer = null;
        }
        // Fully hide the superseded panel so it cannot linger at opacity:0 with
        // pointer-events:auto (which would silently block clicks on the new panel).
        if (prev && prev !== el) {
            prev.style.opacity = '';
            prev.style.transform = '';
            prev.style.display = 'none';
            prev.style.pointerEvents = '';
        }

        // Set as active
        activePanel = el;
        activeCloseFn = closeFn || null;

        // Show with animation
        // NOTE: hideAllOverlays() (ui.js) adds the `hidden` class to every `.overlay`
        // element. `.hidden` is `display:none !important`, which beats the inline
        // `display:flex` below — so we must strip it or the panel never appears.
        el.classList.remove('hidden');
        // Always restore interaction on the panel we are *showing*. closeActivePanel
        // sets pointer-events:none during its fade-out; if its deferred reset timer
        // was cancelled (e.g. this same element is being reopened) the flag would
        // otherwise linger and make the whole panel unclickable.
        el.style.pointerEvents = '';
        el.style.display = 'flex';
        el.style.opacity = '0';
        el.style.transform = 'scale(0.98)';
        // Force reflow
        el.offsetHeight;
        el.style.transition = 'opacity 0.25s cubic-bezier(0.4, 0, 0.2, 1), transform 0.25s cubic-bezier(0.4, 0, 0.2, 1)';
        el.style.opacity = '1';
        el.style.transform = 'scale(1)';

        // Add escape key handler
        escHandler = (e) => {
            if (e.key === 'Escape') {
                closeActivePanel();
            }
        };
        document.addEventListener('keydown', escHandler);

        // Add click-on-backdrop-to-close (but not click-on-panel)
        el.addEventListener('click', function backdropHandler(e) {
            if (e.target === el) {
                closeActivePanel();
            }
        });

        // Hide lobby clutter
        document.body.classList.add('panel-open');
    }

    /**
     * Close the currently active panel.
     */
    function closeActivePanel() {
        if (!activePanel) return;

        const el = activePanel;
        const closeFn = activeCloseFn;

        // Remove escape handler
        if (escHandler) {
            document.removeEventListener('keydown', escHandler);
            escHandler = null;
        }

        // Animate out — disable interaction during the fade so a fading panel
        // does not capture clicks meant for whatever is underneath.
        el.style.pointerEvents = 'none';
        el.style.opacity = '0';
        el.style.transform = 'scale(0.98)';

        // Track the timer so a subsequent openPanel can cancel it (prevents a
        // deferred display:none from hiding a panel that was just reopened).
        if (closeTimer) clearTimeout(closeTimer);
        closeTimer = setTimeout(() => {
            closeTimer = null;
            el.style.display = 'none';
            // Reset transform for next open
            el.style.transform = '';
            el.style.opacity = '';
            el.style.pointerEvents = '';
            // Call custom close function if provided
            if (closeFn) closeFn();
        }, 250);

        activePanel = null;
        activeCloseFn = null;

        document.body.classList.remove('panel-open');
    }

    /**
     * Check if any panel is currently open.
     */
    function isPanelOpen() {
        return activePanel !== null;
    }

    /**
     * Get the currently active panel element.
     */
    function getActivePanel() {
        return activePanel;
    }

    // ========================================================================
    // GLOBAL EXPORTS
    // ========================================================================
    window.PanelManager = {
        open: openPanel,
        close: closeActivePanel,
        isOpen: isPanelOpen,
        getActive: getActivePanel,
    };

})();
