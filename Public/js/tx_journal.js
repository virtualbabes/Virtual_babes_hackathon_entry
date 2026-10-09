// ============================================================================
// tx_journal.js — THE CLIENT'S MEMO OF ITS OWN TRANSACTIONS
// ----------------------------------------------------------------------------
// WHAT THIS IS: a memo. Not a state replica, not a cache of server state.
//
// The chain is the only authority. This journal records what THIS browser did
// and what the server said about it, so a surface can tell the difference
// between:
//
//   signed   — the client signed and broadcast a payment. NOT PROOF OF PAYMENT.
//              The chain may not have it yet, and the server has not looked.
//   verified — the server confirmed it. An OBSERVED FACT, not a local assumption.
//   failed   — the server refused it, with the reason it gave.
//
// THE RULE THAT MATTERS: a `signed` entry is rendered as PENDING and is never
// presented as settled. Nothing in this module promotes an entry to `verified`
// on its own — only a server response does. A client that decided its own
// payments had succeeded would be lying to the player whenever the chain
// disagreed.
//
// MONEY IS INTEGER. Amounts are micro-units, exactly as the ledger carries them.
// This module never divides, rounds or stores a float; formatting is the
// renderer's job.
//
// LOCAL-ONLY. The journal lives in localStorage, is never uploaded, and is
// deliberately NOT authoritative: losing it loses nothing but history.
// ============================================================================

const TXJ_KEY = 'vbt_tx_journal';
const TXJ_MAX = 200;

// Signed by the client, not yet confirmed by the server.
export const TXJ_SIGNED = 'signed';
// Confirmed by the SERVER (the only source of this state).
export const TXJ_VERIFIED = 'verified';
// Refused by the server, with its reason.
export const TXJ_FAILED = 'failed';

const txjState = { error: '', lastWriteAt: 0 };
let txjEntries = null;

function txjRead() {
    if (txjEntries) return txjEntries;
    txjEntries = [];
    try {
        const raw = window.localStorage.getItem(TXJ_KEY);
        if (raw) {
            const parsed = JSON.parse(raw);
            if (Array.isArray(parsed)) {
                txjEntries = parsed.filter((e) => e && typeof e.txid === 'string' && e.txid !== '');
            }
        }
    } catch (err) {
        // A corrupt or unavailable store must not break the app: start empty and
        // RECORD that the history was lost rather than pretending it exists.
        txjEntries = [];
        txjState.error = 'could not read the local transaction journal (' + (err && err.message ? err.message : 'unknown') + ')';
    }
    return txjEntries;
}

function txjWrite() {
    try {
        window.localStorage.setItem(TXJ_KEY, JSON.stringify(txjEntries || []));
        txjState.lastWriteAt = Date.now();
        return true;
    } catch (err) {
        txjState.error = 'could not save the local transaction journal (' + (err && err.message ? err.message : 'unknown') + ')';
        return false;
    }
}

function txjWallet() {
    try {
        if (typeof window.getActiveWallet === 'function') return window.getActiveWallet() || '';
    } catch (err) { /* fall through to the next source */ }
    return window.currentWallet || window.userAddress || '';
}

// record() appends a freshly SIGNED transaction. `amount_micro` must already be
// an integer number of micro-units.
export function record(input) {
    const entry = {
        purpose: String((input && input.purpose) || ''),
        txid: String((input && input.txid) || ''),
        amount_micro: Number.isFinite(input && input.amount_micro) ? Math.trunc(input.amount_micro) : 0,
        note: String((input && input.note) || ''),
        wallet: String((input && input.wallet) || txjWallet()),
        ts: Date.now(),
        // ALWAYS starts pending: a signature is not a settlement.
        status: TXJ_SIGNED,
        reason: '',
        settled_at: 0,
    };
    if (!entry.txid) {
        txjState.error = 'refused to journal a transaction with no txid';
        return null;
    }

    const list = txjRead();
    // Idempotent by txid: re-recording the same transaction does not duplicate it,
    // and it never overwrites what the server already said about it.
    const existing = list.findIndex((e) => e.txid === entry.txid);
    if (existing >= 0) {
        const prev = list[existing];
        list[existing] = Object.assign({}, entry, {
            ts: prev.ts || entry.ts,
            status: prev.status === TXJ_SIGNED ? entry.status : prev.status,
            reason: prev.reason || '',
            settled_at: prev.settled_at || 0,
        });
    } else {
        list.push(entry);
    }

    // Bounded: drop the OLDEST entries beyond the cap.
    if (list.length > TXJ_MAX) list.splice(0, list.length - TXJ_MAX);

    txjWrite();
    return entry;
}

// markVerified() is the ONLY path to a settled entry, and it is called with what
// the SERVER said.
export function markVerified(txid, extra) {
    return txjUpdate(txid, TXJ_VERIFIED, '', extra);
}

// markFailed() records the server's refusal and its reason.
export function markFailed(txid, reason) {
    return txjUpdate(txid, TXJ_FAILED, String(reason || ''), null);
}

function txjUpdate(txid, status, reason, extra) {
    const list = txjRead();
    const idx = list.findIndex((e) => e.txid === txid);

    if (idx < 0) {
        // The server reported something this browser has no record of (another
        // device, or the journal was cleared). Record WHAT THE SERVER SAID rather
        // than inventing the missing detail.
        const stamped = {
            purpose: (extra && extra.purpose) || '',
            txid: String(txid),
            amount_micro: Number.isFinite(extra && extra.amount_micro) ? Math.trunc(extra.amount_micro) : 0,
            note: '',
            wallet: txjWallet(),
            ts: Date.now(),
            status: status,
            reason: reason,
            settled_at: Date.now(),
        };
        list.push(stamped);
        if (list.length > TXJ_MAX) list.splice(0, list.length - TXJ_MAX);
        txjWrite();
        return stamped;
    }

    list[idx] = Object.assign({}, list[idx], { status: status, reason: reason, settled_at: Date.now() });
    txjWrite();
    return list[idx];
}

// ---------------------------------------------------------------------------
// Queries — what a surface reads
// ---------------------------------------------------------------------------

export function all() { return txjRead().slice(); }
export function find(txid) { return txjRead().find((e) => e.txid === txid) || null; }

// pending() — everything the server has NOT confirmed. A `signed` entry is an
// INTENTION, not a payment, and a surface must label it as pending.
export function pending() {
    return txjRead().filter((e) => e.status === TXJ_SIGNED);
}

export function pendingFor(purpose) {
    return txjRead().filter((e) => e.status === TXJ_SIGNED && e.purpose === purpose);
}

// pendingSummary() — integer micro-unit total plus the count, so a caller can
// judge whether a delta is material without any float arithmetic.
export function pendingSummary() {
    const items = pending();
    let total = 0;
    for (const e of items) total += (Number.isFinite(e.amount_micro) ? e.amount_micro : 0);
    return { count: items.length, total_micro: total, items: items };
}

export function clear() {
    txjEntries = [];
    try {
        window.localStorage.removeItem(TXJ_KEY);
    } catch (err) {
        txjState.error = 'could not clear the local transaction journal (' + (err && err.message ? err.message : 'unknown') + ')';
    }
}

// ---------------------------------------------------------------------------
// Rendering — a pending entry is ALWAYS shown as pending
// ---------------------------------------------------------------------------

function txjEscape(s) {
    return String(s === undefined || s === null ? '' : s)
        .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;').replace(/'/g, '&#39;');
}

// Integer-only micro formatting: whole + 6-digit remainder, no float maths.
export function formatMicro(micro) {
    const n = Number.isFinite(micro) ? Math.trunc(micro) : 0;
    const neg = n < 0;
    const abs = Math.abs(n);
    const whole = Math.floor(abs / 1000000);
    const frac = String(abs % 1000000).padStart(6, '0').replace(/0+$/, '');
    return (neg ? '-' : '') + whole + (frac ? '.' + frac : '');
}

// renderPendingHTML() returns markup for the unconfirmed deltas, or '' when there
// are none. Every row is labelled with its true status — a signed row says
// PENDING and never "paid".
export function renderPendingHTML() {
    const summary = pendingSummary();
    if (summary.count === 0) return '';

    const rows = summary.items.map((e) => {
        const label = e.purpose || 'transaction';
        const shortTx = e.txid.length > 14 ? e.txid.slice(0, 8) + '…' + e.txid.slice(-5) : e.txid;
        return '<div class="txj-row" data-txj-status="' + txjEscape(e.status) + '">'
            + '<span class="txj-purpose">' + txjEscape(label) + '</span>'
            + '<span class="txj-amount">' + txjEscape(formatMicro(e.amount_micro)) + '</span>'
            + '<span class="txj-status txj-pending">PENDING</span>'
            + '<span class="txj-txid" title="' + txjEscape(e.txid) + '">' + txjEscape(shortTx) + '</span>'
            + '</div>';
    }).join('');

    return '<div class="txj-pending-block" data-txj-pending="' + summary.count + '">'
        + '<div class="txj-head">Awaiting confirmation '
        + '<span class="txj-count">' + summary.count + '</span> &middot; '
        + txjEscape(formatMicro(summary.total_micro)) + ' $VBV</div>'
        + rows
        + '<div class="txj-note">Signed locally; the chain and the server have not confirmed these yet.</div>'
        + '</div>';
}

export const TxJournal = {
    record,
    markVerified,
    markFailed,
    all,
    find,
    pending,
    pendingFor,
    pendingSummary,
    renderPendingHTML,
    formatMicro,
    clear,
    state: txjState,
    KEY: TXJ_KEY,
    MAX: TXJ_MAX,
    SIGNED: TXJ_SIGNED,
    VERIFIED: TXJ_VERIFIED,
    FAILED: TXJ_FAILED,
};

window.VbtTxJournal = TxJournal;

export default TxJournal;
