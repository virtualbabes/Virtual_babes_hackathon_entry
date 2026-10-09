// ============================================================================
// controller_nav.js — Universal gamepad/keyboard/tilt navigation
// ----------------------------------------------------------------------------
// PC: DirectInput/XInput gamepads, keyboard fallback
// Console: Xbox, PlayStation, Nintendo (via browser Gamepad API)
// Android: Bluetooth gamepads, touch tilt (device orientation)
// All: D-pad/stick → grid navigation, face buttons → select/back
// ============================================================================

import { getControllerNav } from './menu_customization.js';

// ============================================================
// 1. CONTROLLER DETECTION
// ============================================================
const CONTROLLER_TYPES = {
    UNKNOWN: 'unknown',
    XBOX: 'xbox',
    PLAYSTATION: 'playstation',
    NINTENDO: 'nintendo',
    GENERIC: 'generic',
    KEYBOARD: 'keyboard',
    TOUCH_TILT: 'touch_tilt',
};

const STATE = {
    active: false,
    type: CONTROLLER_TYPES.KEYBOARD,
    gamepadIndex: -1,
    connected: false,
    lastInput: null,
    repeatDelay: 200,
    repeatRate: 80,
    lastMoveTime: 0,
    deadZone: 0.3,
    focusIndex: 0,
    items: [],
    enabled: true,
};

// Button mappings (standard Gamepad API)
const BTN = {
    A: 0, B: 1, X: 2, Y: 3,
    LB: 4, RB: 5, LT: 6, RT: 7,
    BACK: 8, START: 9,
    LS: 10, RS: 11,
    DPAD_UP: 12, DPAD_DOWN: 13, DPAD_LEFT: 14, DPAD_RIGHT: 15,
    HOME: 16,
};

// Face button labels per controller type
const BTN_LABELS = {
    [CONTROLLER_TYPES.XBOX]: { A: 'A', B: 'B', X: 'X', Y: 'Y', select: 'A', back: 'B' },
    [CONTROLLER_TYPES.PLAYSTATION]: { A: '✕', B: '○', X: '□', Y: '△', select: '✕', back: '○' },
    [CONTROLLER_TYPES.NINTENDO]: { A: 'A', B: 'B', X: 'X', Y: 'Y', select: 'A', back: 'B' },
    [CONTROLLER_TYPES.GENERIC]: { A: 'A', B: 'B', X: 'X', Y: 'Y', select: 'A', back: 'B' },
    [CONTROLLER_TYPES.KEYBOARD]: { A: 'Enter', B: 'Esc', X: 'X', Y: 'Y', select: 'Enter', back: 'Esc' },
    [CONTROLLER_TYPES.TOUCH_TILT]: { A: 'Tap', B: 'Swipe Down', X: 'X', Y: 'Y', select: 'Tap', back: 'Swipe Down' },
};

// ============================================================
// 2. DETECTION
// ============================================================
function detectControllerType(gamepad) {
    const id = (gamepad.id || '').toLowerCase();
    if (id.includes('xbox') || id.includes('xinput') || id.includes('microsoft')) {
        return CONTROLLER_TYPES.XBOX;
    }
    if (id.includes('playstation') || id.includes('ps4') || id.includes('ps5') || id.includes('dualshock') || id.includes('dualsense')) {
        return CONTROLLER_TYPES.PLAYSTATION;
    }
    if (id.includes('nintendo') || id.includes('pro controller') || id.includes('joy-con')) {
        return CONTROLLER_TYPES.NINTENDO;
    }
    return CONTROLLER_TYPES.GENERIC;
}

function getActiveGamepad() {
    const gamepads = navigator.getGamepads ? navigator.getGamepads() : [];
    for (let i = 0; i < gamepads.length; i++) {
        if (gamepads[i] && gamepads[i].connected) {
            return { gamepad: gamepads[i], index: i };
        }
    }
    return null;
}

// ============================================================
// 3. EVENT LISTENERS
// ============================================================
function init() {
    window.addEventListener('gamepadconnected', onGamepadConnected);
    window.addEventListener('gamepaddisconnected', onGamepadDisconnected);
    window.addEventListener('keydown', onKeyDown);

    if (window.DeviceOrientationEvent) {
        window.addEventListener('deviceorientation', onDeviceOrientation);
    }

    startPolling();

    const existing = getActiveGamepad();
    if (existing) {
        STATE.gamepadIndex = existing.index;
        STATE.connected = true;
        STATE.type = detectControllerType(existing.gamepad);
        STATE.active = true;
        console.log('[Controller] Already connected:', STATE.type, existing.gamepad.id);
    }
}

function onGamepadConnected(e) {
    STATE.connected = true;
    STATE.gamepadIndex = e.gamepad.index;
    STATE.type = detectControllerType(e.gamepad);
    STATE.active = true;
    console.log('[Controller] Connected:', STATE.type, e.gamepad.id);
    if (window.showToast) window.showToast(`${STATE.type.toUpperCase()} controller connected!`, 'success');
    showFocusRing();
    // Sync to WASM
    if (window.SetControllerType) window.SetControllerType(STATE.type);
}

function onGamepadDisconnected(e) {
    STATE.connected = false;
    STATE.gamepadIndex = -1;
    STATE.active = false;
    STATE.type = CONTROLLER_TYPES.KEYBOARD;
    console.log('[Controller] Disconnected');
    hideFocusRing();
    if (window.SetControllerType) window.SetControllerType(CONTROLLER_TYPES.KEYBOARD);
}

function onKeyDown(e) {
    if (!STATE.enabled) return;
    if (STATE.connected) return;

    let action = null;
    switch (e.key) {
        case 'ArrowUp':    action = 'up'; break;
        case 'ArrowDown':  action = 'down'; break;
        case 'ArrowLeft':  action = 'left'; break;
        case 'ArrowRight': action = 'right'; break;
        case 'Enter':      action = 'select'; break;
        case 'Escape':     action = 'back'; break;
        case 'w': case 'W': action = 'up'; break;
        case 's': case 'S': action = 'down'; break;
        case 'a': case 'A': action = 'left'; break;
        case 'd': case 'D': action = 'right'; break;
    }

    if (action) {
        e.preventDefault();
        handleAction(action);
    }
}

function onDeviceOrientation(e) {
    if (!STATE.enabled) return;
    if (STATE.connected) return;
    if (!e.beta && !e.gamma) return;

    const gamma = e.gamma || 0;
    const beta = e.beta || 0;
    const threshold = 15;

    if (Math.abs(gamma) > threshold || Math.abs(beta) > threshold) {
        STATE.type = CONTROLLER_TYPES.TOUCH_TILT;
        STATE.active = true;
        showFocusRing();

        if (Date.now() - STATE.lastMoveTime < STATE.repeatDelay) return;

        if (gamma > threshold) handleAction('right');
        else if (gamma < -threshold) handleAction('left');
        else if (beta > threshold) handleAction('down');
        else if (beta < -threshold) handleAction('up');
    }
}

// ============================================================
// 4. POLLING LOOP (for gamepad analog + dpad)
// ============================================================
function startPolling() {
    function poll() {
        if (STATE.connected && STATE.enabled) {
            pollGamepad();
        }
        requestAnimationFrame(poll);
    }
    requestAnimationFrame(poll);
}

function pollGamepad() {
    const gp = getActiveGamepad();
    if (!gp) return;

    const now = Date.now();
    const pad = gp.gamepad;

    let action = null;
    if (pad.buttons[BTN.DPAD_UP]?.pressed) action = 'up';
    else if (pad.buttons[BTN.DPAD_DOWN]?.pressed) action = 'down';
    else if (pad.buttons[BTN.DPAD_LEFT]?.pressed) action = 'left';
    else if (pad.buttons[BTN.DPAD_RIGHT]?.pressed) action = 'right';

    if (!action && pad.axes.length >= 2) {
        const lx = pad.axes[0];
        const ly = pad.axes[1];

        if (Math.abs(lx) > STATE.deadZone || Math.abs(ly) > STATE.deadZone) {
            if (Math.abs(lx) > Math.abs(ly)) {
                action = lx > 0 ? 'right' : 'left';
            } else {
                action = ly > 0 ? 'down' : 'up';
            }
        }
    }

    if (pad.buttons[BTN.A]?.pressed) {
        handleAction('select');
        return;
    }
    if (pad.buttons[BTN.B]?.pressed) {
        handleAction('back');
        return;
    }

    if (action) {
        if (now - STATE.lastMoveTime < STATE.repeatDelay) return;
        STATE.lastMoveTime = now;
        handleAction(action);
    } else {
        STATE.lastMoveTime = 0;
    }
}

// ============================================================
// 5. ACTION HANDLER
// ============================================================
function handleAction(action) {
    STATE.lastInput = action;

    // Use WASM bridge for navigation if available
    if (window.HandleControllerInput) {
        const newFocus = window.HandleControllerInput(action);
        if (newFocus !== undefined && newFocus !== null) {
            STATE.focusIndex = newFocus;
        }
    }

    const grid = document.getElementById('ch-starred-grid');
    if (!grid) return;
    const items = grid.querySelectorAll('.ch-starred-btn, .preview-btn');
    if (items.length === 0) return;

    const layout = window.UserPreferences ? window.UserPreferences.getMenuLayout() : { gridType: 'grid' };
    const gridType = layout.gridType || 'grid';

    switch (action) {
        case 'up':
            moveFocus(gridType, items, 'up');
            break;
        case 'down':
            moveFocus(gridType, items, 'down');
            break;
        case 'left':
            moveFocus(gridType, items, 'left');
            break;
        case 'right':
            moveFocus(gridType, items, 'right');
            break;
        case 'select':
            activateItem(items[STATE.focusIndex]);
            break;
        case 'back':
            if (window.closeMenuCustomization) window.closeMenuCustomization();
            break;
    }
}

function moveFocus(gridType, items, direction) {
    const count = items.length;
    if (count === 0) return;

    let newIndex = STATE.focusIndex;

    switch (gridType) {
        case 'grid': {
            const cols = parseInt(window.UserPreferences?.getMenuLayout().gridCols) || Math.ceil(Math.sqrt(count));
            const row = Math.floor(STATE.focusIndex / cols);
            const col = STATE.focusIndex % cols;

            switch (direction) {
                case 'up':    newIndex = (row - 1) * cols + col; break;
                case 'down':  newIndex = (row + 1) * cols + col; break;
                case 'left':  newIndex = row * cols + (col - 1); break;
                case 'right': newIndex = row * cols + (col + 1); break;
            }
            break;
        }
        case 'circle': {
            switch (direction) {
                case 'right': newIndex = (STATE.focusIndex + 1) % count; break;
                case 'left':  newIndex = (STATE.focusIndex - 1 + count) % count; break;
                case 'down':  newIndex = Math.min(count - 1, STATE.focusIndex + 1); break;
                case 'up':    newIndex = Math.max(0, STATE.focusIndex - 1); break;
            }
            break;
        }
        case 'linear-h':
        case 'arc': {
            switch (direction) {
                case 'right': newIndex = Math.min(count - 1, STATE.focusIndex + 1); break;
                case 'left':  newIndex = Math.max(0, STATE.focusIndex - 1); break;
                default: break;
            }
            break;
        }
        case 'linear-v':
        case 'triangle': {
            switch (direction) {
                case 'down':  newIndex = Math.min(count - 1, STATE.focusIndex + 1); break;
                case 'up':    newIndex = Math.max(0, STATE.focusIndex - 1); break;
                default: break;
            }
            break;
        }
        case 'cross': {
            const crossMap = { up: 1, right: 2, down: 3, left: 4 };
            newIndex = crossMap[direction] ?? 0;
            break;
        }
        case 'diamond': {
            const diamondMap = { up: 0, right: 1, down: 2, left: 3 };
            newIndex = diamondMap[direction] ?? 0;
            break;
        }
        default: {
            switch (direction) {
                case 'right': case 'down': newIndex = Math.min(count - 1, STATE.focusIndex + 1); break;
                case 'left':  case 'up':    newIndex = Math.max(0, STATE.focusIndex - 1); break;
            }
        }
    }

    newIndex = Math.max(0, Math.min(count - 1, newIndex));
    setFocus(newIndex, items);
}

function setFocus(index, items) {
    if (index < 0 || index >= items.length) return;

    items.forEach(item => item.classList.remove('controller-focus'));

    STATE.focusIndex = index;
    const item = items[index];
    item.classList.add('controller-focus');
    item.focus({ preventScroll: true });

    // Sync to WASM
    if (window.SetControllerFocus) window.SetControllerFocus(index);
}

function activateItem(item) {
    if (item && item.click) {
        item.click();
    }
}

function showFocusRing() {
    STATE.active = true;
    const grid = document.getElementById('ch-starred-grid');
    if (grid) {
        const items = grid.querySelectorAll('.ch-starred-btn');
        if (items.length > 0) {
            setFocus(STATE.focusIndex, items);
        }
    }
}

function hideFocusRing() {
    STATE.active = false;
    document.querySelectorAll('.controller-focus').forEach(el => {
        el.classList.remove('controller-focus');
    });
}

// ============================================================
// 6. HAPTIC FEEDBACK (where supported)
// ============================================================
function hapticFeedback(type) {
    if (!STATE.connected) return;
    const gp = getActiveGamepad();
    if (!gp || !gp.gamepad.vibrationActuator) return;

    try {
        const actuator = gp.gamepad.vibrationActuator;
        switch (type) {
            case 'move':
                actuator.playEffect('dual-rumble', { duration: 50, strongMagnitude: 0.2, weakMagnitude: 0.2 });
                break;
            case 'select':
                actuator.playEffect('dual-rumble', { duration: 150, strongMagnitude: 0.8, weakMagnitude: 0.8 });
                break;
            case 'back':
                actuator.playEffect('dual-rumble', { duration: 100, strongMagnitude: 0.4, weakMagnitude: 0.4 });
                break;
        }
    } catch (e) {
        // Vibration not supported
    }
}

// ============================================================
// 7. CONSOLE-SPECIFIC ADAPTATIONS
// ============================================================
function getConsoleHint() {
    const labels = BTN_LABELS[STATE.type] || BTN_LABELS[CONTROLLER_TYPES.GENERIC];
    return {
        select: `${labels.select} = Select`,
        back: `${labels.back} = Back`,
        navigate: 'D-Pad / Stick = Move',
    };
}

function injectConsoleHints() {
    if (STATE.type === CONTROLLER_TYPES.KEYBOARD) return;

    const grid = document.getElementById('ch-starred-grid');
    if (!grid) return;

    let hintEl = document.getElementById('controller-hints');
    if (!hintEl) {
        hintEl = document.createElement('div');
        hintEl.id = 'controller-hints';
        hintEl.className = 'controller-hints';
        grid.parentElement.appendChild(hintEl);
    }

    const hints = getConsoleHint();
    hintEl.innerHTML = `
        <span class="hint-item">${hints.select}</span>
        <span class="hint-item">${hints.back}</span>
        <span class="hint-item">${hints.navigate}</span>
    `;
    hintEl.style.display = 'flex';
}

// ============================================================
// 8. PUBLIC API
// ============================================================

export {
    init,
    STATE,
    CONTROLLER_TYPES,
    BTN_LABELS,
    handleAction,
    setFocus,
    showFocusRing,
    hideFocusRing,
    hapticFeedback,
    injectConsoleHints,
    getConsoleHint,
    getActiveGamepad,
    detectControllerType,
};

export function enable() { STATE.enabled = true; showFocusRing(); }
export function disable() { STATE.enabled = false; hideFocusRing(); }
export function isActive() { return STATE.active; }
export function getType() { return STATE.type; }
export function getFocusIndex() { return STATE.focusIndex; }

// Auto-init
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
} else {
    init();
}
