/**
 * ADMIN CONSOLE — the ONE reachable, signature-gated administrative surface.
 *
 * WHY THIS MODULE EXISTS. The Admin Suite markup has lived in index.html with no opener at all:
 * `network.js` only refreshed its logs IF it was already visible, and the four "dashboard" modules
 * that offered their own admin tabs (admin_dashboard, operations_dashboard, final_dashboard,
 * admin_panel) were never mounted or called by anything. So the entire administrative surface was
 * unreachable, and four competing copies of it existed. This module is the single owner of
 * REACHING it.
 *
 * AUTHORITY. Access is a WALLET SIGNATURE, never a password: admin.js builds
 * X-Admin-Wallet / X-Admin-Nonce / X-Admin-Signature from the connected wallet, and the server
 * accepts only wallets listed in the ADMIN_WALLETS environment variable. This console therefore
 * cannot grant access — it only REPORTS what the server answers, and says so when refused.
 *
 * CONTAINED CONTRACT. The console reveals the existing `#admin-control-panel` overlay root (class
 * contract: `.overlay` is closed by the `.hidden` class, which is how hideAllOverlays() closes it).
 * It deliberately does NOT wrap that root in another overlay: nesting an overlay root inside
 * another keeps the inner one invisible, because `visibility` inherits.
 */

import { getAdminHeaders } from './admin.js';
import { bindAdminSuiteControls } from './admin_suite.js';

const PANEL_ID = 'admin-control-panel';
const STATUS_ID = 'admin-console-status';
const AUTHORITY_ID = 'admin-console-authority';
const REWARD_LIST_ID = 'admin-reward-list';

function panel() {
    return document.getElementById(PANEL_ID);
}

/**
 * setStatus states the CURRENT access state in the panel chrome. A refused console must never look
 * like an empty console: "not an admin wallet" and "an admin session with nothing to show" are
 * different facts.
 */
function setStatus(kind, text) {
    const root = panel();
    if (!root) return;
    const main = root.querySelector('.admin-panel-main') || root;
    let status = document.getElementById(STATUS_ID);
    if (!status) {
        status = document.createElement('div');
        status.id = STATUS_ID;
        status.className = 'font-sm p-10 rounded-md mb-10';
        status.style.textAlign = 'left';
        main.insertBefore(status, main.firstChild);
    }
    const palette = {
        probing: ['rgba(120,120,120,0.18)', 'rgba(255,255,255,0.75)'],
        denied: ['rgba(248,81,73,0.18)', '#f85149'],
        ready: ['rgba(63,185,80,0.15)', '#3fb950'],
    };
    const [bg, fg] = palette[kind] || palette.probing;
    status.style.background = bg;
    status.style.color = fg;
    status.textContent = text;
}

/**
 * fetchAuthorised calls an admin endpoint with the signature headers admin.js builds.
 * Returns { ok, status, data, text } — a refused call is DATA, not an exception, so the caller can
 * state the refusal instead of rendering an empty panel.
 */
/**
 * fetchAuthorised calls an admin endpoint with the signature headers admin.js builds.
 * Returns { ok, status, data, text } — a refused call is DATA, not an exception, so the caller can
 * STATE the refusal instead of rendering an empty panel.
 */
async function fetchAuthorised(path, options = {}) {
    const headers = await getAdminHeaders();
    if (!headers) {
        return { ok: false, status: 0, text: 'a wallet must be connected and able to sign the admin nonce' };
    }
    try {
        const response = await fetch(`${CONFIG.API_BASE}${path}`, {
            ...options,
            headers: { ...headers, 'Content-Type': 'application/json', ...(options.headers || {}) },
        });
        const text = await response.text();
        let data = null;
        try { data = JSON.parse(text); } catch (_) { /* a non-JSON body is reported as text */ }
        return { ok: response.ok, status: response.status, data, text };
    } catch (err) {
        return { ok: false, status: 0, text: err.message };
    }
}

/**
 * renderAuthority renders the READ-ONLY authority view served by GET /api/admin/networks: which
 * environment values are authoritative, which networks hold secret material (PRESENCE only), and
 * which documented env vars are read by no code. It never invents a value.
 */
function renderAuthority(payload) {
    const root = panel();
    if (!root) return;
    const main = root.querySelector('.admin-panel-main') || root;

    let box = document.getElementById(AUTHORITY_ID);
    if (!box) {
        box = document.createElement('div');
        box.id = AUTHORITY_ID;
        box.className = 'glass-panel admin-section w-full mt-20';
        box.style.textAlign = 'left';
        main.appendChild(box);
    }
    box.innerHTML = '';

    const heading = document.createElement('h2');
    heading.textContent = 'Authority (read-only)';
    box.appendChild(heading);

    const row = (label, value) => {
        const div = document.createElement('div');
        div.className = 'font-xs line-height-1-2';
        div.textContent = `${label}: ${value}`;
        box.appendChild(div);
    };

    const env = payload.env_authority || {};
    const units = (micro) => (Number(micro || 0) / 1000000); // display only
    row('Vault (VAULT_ADDRESS)', env.vault_address || '(unset)');
    row('Primary reward asset (REWARD_ASSET_ID)', env.reward_asset_id || '(unset)');
    row('Base reward', `${units(env.base_reward_micro).toFixed(2)} $VBV (${env.base_reward_micro || 0} micro-VBV)`);
    row('Faucet mnemonic', env.faucet_mnemonic_configured
        ? `configured (${env.faucet_mnemonic_word_count} words — the value is never served)`
        : 'NOT configured (payouts cannot be signed)');
    row('Administrators (ADMIN_WALLETS)', `${env.admin_wallets_configured || 0} wallet(s) configured`);
    row('Node credentials', `VOI token ${env.algod_token_voi_configured ? 'set' : 'unset'} · ALGO token ${env.algod_token_algo_configured ? 'set' : 'unset'} · IPFS key ${env.ipfs_api_key_configured ? 'set' : 'unset'}`);
    row('Registry file', `${payload.registry_file || '(unknown)'}${payload.restart_required ? ' — a change needs a server RESTART' : ''}`);

    const secrets = payload.secrets_configured || {};
    const holders = secrets.networks_holding_secrets || [];
    row('Secrets in the running registry', `${secrets.algod_tokens_configured || 0} token(s), ${secrets.ipfs_api_keys_configured || 0} IPFS key(s)` +
        (holders.length ? ` — held by ${holders.join(', ')}` : '') + ' (never served, never persisted)');

    const unread = payload.declared_but_unread || [];
    const notice = document.createElement('div');
    notice.className = 'font-xs p-10 rounded-md mt-10';
    notice.style.background = unread.length ? 'rgba(210,153,34,0.15)' : 'rgba(63,185,80,0.12)';
    notice.style.color = unread.length ? '#d29922' : '#3fb950';
    notice.textContent = unread.length
        ? `Documented but read by NO code (changing these changes nothing): ${unread.join(', ')}`
        : 'Every documented environment variable has a reader.';
    box.appendChild(notice);
}

/**
 * refreshAdminConsole reads the two read-only surfaces the console owns: the registry/authority
 * view (admin-gated) and the reward-token registry (served publicly by the faucet status, which is
 * the same registry the game pays from). A refused read is STATED, never rendered as "no data".
 */
export async function refreshAdminConsole() {
    const root = panel();
    if (!root) return;

    const authority = await fetchAuthorised('/api/admin/networks', { method: 'GET' });
    if (authority.status === 401 || authority.status === 403) {
        setStatus('denied', `Access refused (HTTP ${authority.status}): this wallet is not an authorised administrator. ` +
            'Administrators are listed in the ADMIN_WALLETS environment variable, which this console cannot change.');
        return;
    }
    if (!authority.ok) {
        setStatus('denied', `The authority view could not be read (HTTP ${authority.status || 'n/a'}): ${authority.text || 'no detail served'}`);
        return;
    }

    const payload = authority.data || {};
    const env = payload.env_authority || {};
    setStatus('ready', `Signed in as an authorised administrator · ${env.admin_wallets_configured || 0} administrator wallet(s) · ` +
        `registry ${payload.registry_file || '(unknown)'}${payload.restart_required ? ' (a change needs a RESTART)' : ''}`);
    renderAuthority(payload);

    try {
        const response = await fetch(`${CONFIG.API_BASE}/api/faucet/status`);
        const status = await response.json();
        if (typeof window.updateAdminRewardRegistry === 'function') {
            window.updateAdminRewardRegistry(status.reward_tokens, REWARD_LIST_ID);
        }
    } catch (err) {
        const list = document.getElementById(REWARD_LIST_ID);
        if (list) list.textContent = `The reward-token registry could not be read: ${err.message}`;
    }
}

/**
 * openAdminConsole is the ONE way in. It closes every other overlay first, wires the suite's
 * controls (idempotent — app.js wires them at boot; this is the safety net for an early open),
 * reveals the panel using the legacy `.overlay` contract (remove `.hidden`), and states the
 * access result.
 */
export function openAdminConsole() {
    if (typeof window.hideAllOverlays === 'function') window.hideAllOverlays();
    // The console must be usable even if it is opened before app.js reached its boot binding.
    bindAdminSuiteControls();

    const root = panel();
    if (!root) {
        if (typeof window.showToast === 'function') {
            window.showToast('❌ The admin console markup is missing from the application shell.', 'error');
        }
        return;
    }
    root.classList.remove('hidden');
    // Legacy `.overlay` roots are CLOSED by hideAllOverlays() adding `.hidden`, so `.hidden` is the
    // authority for this panel. The inline display is also set because every other opener in this
    // app does, and the visibility audit detects a reveal through inline display — one contract for
    // all openers, and a panel that cannot be left "revealed but invisible".
    root.style.display = 'flex';
    setStatus('probing', 'Checking administrator authority…');
    refreshAdminConsole();
}

export function closeAdminConsole() {
    const root = panel();
    if (root) {
        root.style.display = '';
        root.classList.add('hidden');
    }
}

window.openAdminConsole = openAdminConsole;
window.closeAdminConsole = closeAdminConsole;
window.refreshAdminConsole = refreshAdminConsole;
