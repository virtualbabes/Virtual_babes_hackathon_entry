/**
 * ADMIN SUITE CONTROLS — the ONE place the administrative controls are wired.
 *
 * WHY THIS MODULE EXISTS. Every control in the Admin Suite carried an inline `onclick="adminX()"`
 * in index.html. An inline handler resolves `adminX` on `window`, and EIGHT of these functions
 * (adminUpdatePowerScaling, adminBanWallet, adminAvatarBan, adminResetStats, adminUpdateDLCProduct,
 * fetchDLCRegistry, fetchAdminLogs, renderShopTokenPresetEditor) were published by nothing — so
 * those controls threw `ReferenceError` and did absolutely nothing. `adminResetStats()` was
 * additionally called with NO argument while the markup supplied an input for one, and its sibling
 * `saveShopTokenPresetsFromEditor` (which admin.js renders into the preset modal) was unbound too.
 *
 * Bound here, each handler calls the IMPORTED function directly, so a control works whether or not
 * a global exists; the suite's own public functions are published from ONE list so a rendered row
 * that names a global can still resolve it.
 *
 * ONE ATOMIC CHANGE. The inline attributes were deleted in the same change that added these
 * bindings: with both in place every control would DOUBLE-FIRE.
 *
 * CONTRACT. The markup declares WHAT a control does (`data-admin-action="<name>"`); this module
 * declares HOW. A control whose action is not declared here is reported in the console rather than
 * silently inert — an unwired control must never look like a working one.
 */

import {
    adminAddNetwork,
    adminAddReward,
    adminAvatarBan,
    adminBanWallet,
    adminBanWalletFromLog,
    adminBroadcast,
    adminForcePayout,
    adminRemoveReward,
    adminResetStats,
    adminSetActiveNetwork,
    adminSimulateMojoDecay,
    adminSimulateTournament,
    adminToggleDevMode,
    adminToggleMaintenance,
    adminUpdateDLCProduct,
    adminUpdatePowerScaling,
    adminUpdateRules,
    fetchAdminLogs,
    fetchDLCRegistry,
    onAdminNetworkSelectChange,
    renderShopTokenPresetEditor,
    saveShopTokenPresetsFromEditor,
} from './admin.js';

const PANEL_ID = 'admin-control-panel';

/** Reads a text/number input's trimmed value, or '' when the control is absent. */
function inputValue(id) {
    const el = document.getElementById(id);
    return el && typeof el.value === 'string' ? el.value.trim() : '';
}

/** Writes a read-out span. DISPLAY formatting only — never ledger arithmetic. */
function setText(id, value) {
    const el = document.getElementById(id);
    if (el) el.innerText = value;
}

/** refuse STATES a refusal. A refused action must never look like a control that did nothing. */
function refuse(message) {
    if (typeof window.setTransactionStatus === 'function') window.setTransactionStatus(`❌ ${message}`, 'critical');
    if (typeof window.showToast === 'function') window.showToast(`❌ ${message}`, 'error');
}

/**
 * Every Admin Suite control, keyed by the `data-admin-action` value in index.html. Each entry names
 * the DOM event it answers and the action it runs. Nothing here reads a global.
 */
const ADMIN_SUITE_ACTIONS = {
    // Power-scaling read-outs. Both are DISPLAY formatting of a slider's own value.
    'power-divisor': { event: 'input', run: (el) => setText('power-divisor-val', parseFloat(el.value).toFixed(1)) },
    'power-base': { event: 'input', run: (el) => setText('power-base-val', el.value) },
    'update-power-scaling': { event: 'click', run: () => adminUpdatePowerScaling() },
    'sync-rules': { event: 'click', run: () => adminUpdateRules() },
    'toggle-dev-mode': { event: 'change', run: () => adminToggleDevMode() },
    'network-select': { event: 'change', run: () => onAdminNetworkSelectChange() },
    'set-active-network': { event: 'click', run: () => adminSetActiveNetwork() },
    'add-network': { event: 'click', run: () => adminAddNetwork() },
    'ban-wallet': { event: 'click', run: () => adminBanWallet() },
    'ban-avatar': { event: 'click', run: () => adminAvatarBan(inputValue('admin-ban-avatar-url')) },
    'reset-stats': {
        event: 'click',
        run: () => {
            // The markup has always carried `#admin-reset-wallet`; the inline handler never read it,
            // so the request named no wallet at all. Read it, and refuse an empty one outright.
            const wallet = inputValue('admin-reset-wallet');
            if (!wallet) { refuse('Enter the wallet whose history should be reset.'); return; }
            return adminResetStats(wallet);
        },
    },
    'broadcast': { event: 'click', run: () => adminBroadcast() },
    'maintenance-start': { event: 'click', run: () => adminToggleMaintenance(true) },
    'maintenance-stop': { event: 'click', run: () => adminToggleMaintenance(false) },
    'add-reward': { event: 'click', run: () => adminAddReward() },
    'simulate-tournament': { event: 'click', run: () => adminSimulateTournament() },
    'force-payout': { event: 'click', run: () => adminForcePayout() },
    'simulate-mojo-decay': { event: 'click', run: () => adminSimulateMojoDecay() },
    'edit-presets': { event: 'click', run: () => renderShopTokenPresetEditor() },
    'refresh-dlc': { event: 'click', run: () => fetchDLCRegistry() },
    'add-dlc-product': { event: 'click', run: () => adminUpdateDLCProduct() },
    'admin-logs-filter': { event: 'click', run: () => fetchAdminLogs() },
    'admin-logs-refresh': { event: 'click', run: () => fetchAdminLogs() },
    'close-panel': {
        event: 'click',
        run: () => { if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays(); }
    },
};

/**
 * The suite's PUBLIC FUNCTIONS. admin.js renders some of its own rows with inline handlers, so any
 * function a rendered row names must also exist as a global. Publishing them from ONE list keeps
 * that surface declared here instead of spread across the composition root, and covers the seven
 * that had no publisher at all. Re-asserting a reference app.js already published is a no-op.
 */
const ADMIN_SUITE_GLOBALS = {
    adminUpdatePowerScaling,
    adminBanWallet,
    adminBanWalletFromLog,
    adminAvatarBan,
    adminResetStats,
    adminUpdateDLCProduct,
    fetchDLCRegistry,
    fetchAdminLogs,
    renderShopTokenPresetEditor,
    saveShopTokenPresetsFromEditor,
    adminAddReward,
    adminRemoveReward,
    adminSimulateTournament,
    adminForcePayout,
    adminSimulateMojoDecay,
    adminUpdateRules,
    adminBroadcast,
    adminToggleMaintenance,
    adminToggleDevMode,
    adminSetActiveNetwork,
    adminAddNetwork,
    onAdminNetworkSelectChange,
};

let suiteControlsBound = false;

/**
 * bindAdminSuiteControls attaches every declared control ONCE and publishes the suite's globals.
 * Idempotent by design: a second call returns immediately, because binding twice would double-fire.
 * Returns true when the panel is present (whether or not THIS call did the binding).
 */
export function bindAdminSuiteControls() {
    const root = document.getElementById(PANEL_ID);
    if (!root) return false;
    if (suiteControlsBound) return true;

    let bound = 0;
    let undeclared = 0;
    root.querySelectorAll('[data-admin-action]').forEach((control) => {
        const action = control.dataset.adminAction;
        const spec = ADMIN_SUITE_ACTIONS[action];
        if (!spec) {
            undeclared++;
            console.warn(`[ADMIN SUITE] no handler is declared for the control "${action}"`);
            return;
        }
        control.addEventListener(spec.event, () => {
            Promise.resolve()
                .then(() => spec.run(control))
                .catch((err) => {
                    const detail = `${action} failed: ${(err && err.message) || err}`;
                    console.error('[ADMIN SUITE]', detail);
                    refuse(detail);
                });
        });
        bound++;
    });

    Object.entries(ADMIN_SUITE_GLOBALS).forEach(([name, fn]) => {
        if (typeof fn === 'function') window[name] = fn;
    });

    suiteControlsBound = true;
    console.log(`[ADMIN SUITE] ${bound} control(s) wired` + (undeclared ? `, ${undeclared} undeclared` : ''));
    return true;
}

/** Reports whether the suite's controls have been wired. Used by the entry probe. */
export function adminSuiteWired() {
    return suiteControlsBound;
}

window.bindAdminSuiteControls = bindAdminSuiteControls;


