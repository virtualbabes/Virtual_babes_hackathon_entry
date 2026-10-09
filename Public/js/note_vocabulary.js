// ============================================================================
// note_vocabulary.js — the CLIENT'S READ-ONLY VIEW OF THE NOTE VOCABULARY
// ----------------------------------------------------------------------------
// The server owns every on-chain note prefix (Go: note_vocabulary.go). This
// module is the client's ONLY way to learn one:
//
//     GET /api/notes/vocabulary  ->  { purposes:[{key,prefix,scope,...}], ... }
//
// WHY IT IS NOT A LOCAL TABLE: a note prefix is the ONE thing a client and a
// server must agree on EXACTLY. If they disagree, nothing throws — the server
// simply never matches the payment and every attempt fails verification, which
// looks like "payments are broken" rather than "the client sent the wrong
// prefix". So the prefix is fetched, never re-typed. A copy of the vocabulary
// living in this file would be exactly the drift the server's single-owner rule
// exists to prevent.
//
// FAIL CLOSED. If the vocabulary cannot be read, prefix()/buildNote() return ''
// and the refusal is stated. They never fall back to a remembered prefix and
// never invent one. A caller that gets '' must refuse to sign.
//
// SCOPE. A client may name MONEY-DOOR purposes only (client -> vault). The
// server states that rule in `rules.client_may_name_money_door_only`; this
// module exposes it so a surface renders the rule instead of assuming it.
// ============================================================================

const NV_API = '/api';
const NV_ENDPOINT = NV_API + '/notes/vocabulary';

const nvState = {
    loaded: false,
    loading: false,
    error: '',
    fetchedAt: 0,
    purposes: [],
    byKey: Object.create(null),
    byPrefix: Object.create(null),
    moneyDoors: [],
    rules: null,
};

let nvLoadPromise = null;

// One-shot 429 retry (the repo-wide convention: a rate-limited read is retried
// once and then REPORTED — never silently rendered as "no data").
async function nvFetchJSON(url) {
    for (let attempt = 0; attempt < 2; attempt++) {
        const res = await fetch(url, { headers: { 'Accept': 'application/json' } });
        if (res.status === 429 && attempt === 0) {
            await new Promise((r) => setTimeout(r, 450));
            continue;
        }
        if (!res.ok) throw new Error('HTTP ' + res.status);
        return await res.json();
    }
    throw new Error('HTTP 429');
}

// load() reads the served vocabulary once. `force` re-reads.
export async function load(force) {
    if (nvState.loaded && !force) return nvState;
    if (nvLoadPromise && !force) return nvLoadPromise;

    nvState.loading = true;
    nvLoadPromise = (async () => {
        try {
            const data = await nvFetchJSON(NV_ENDPOINT);
            const purposes = Array.isArray(data && data.purposes) ? data.purposes : [];

            const byKey = Object.create(null);
            const byPrefix = Object.create(null);
            const moneyDoors = [];
            for (const p of purposes) {
                if (!p || !p.key || !p.prefix) continue;
                byKey[p.key] = p;
                byPrefix[p.prefix] = p;
                if (p.scope === 'money_door') moneyDoors.push(p.prefix);
            }

            // A served vocabulary with NO money doors means the server and client
            // disagree about the contract: refuse rather than half-believe it.
            if (moneyDoors.length === 0) {
                throw new Error('served vocabulary declares no money-door purposes');
            }

            nvState.purposes = purposes;
            nvState.byKey = byKey;
            nvState.byPrefix = byPrefix;
            nvState.moneyDoors = moneyDoors;
            nvState.rules = (data && data.rules) || null;
            nvState.loaded = true;
            nvState.error = '';
            nvState.fetchedAt = Date.now();
        } catch (err) {
            nvState.loaded = false;
            nvState.error = 'could not read the note vocabulary (' + (err && err.message ? err.message : 'unknown') + ')';
        } finally {
            nvState.loading = false;
            nvLoadPromise = null;
        }
        return nvState;
    })();

    return nvLoadPromise;
}

// ---------------------------------------------------------------------------
// Prefix resolution — every path below FAILS CLOSED (returns '' and records why)
// ---------------------------------------------------------------------------

// prefix(key) returns the served prefix for a purpose key, or ''.
export function prefix(key) {
    if (!nvState.loaded) {
        nvState.error = 'note vocabulary not loaded: call await NoteVocab.load() first';
        return '';
    }
    const entry = nvState.byKey[key];
    if (!entry) {
        nvState.error = 'unknown note purpose "' + key + '" (the server does not declare it)';
        return '';
    }
    return entry.prefix;
}

// isDeclared(p) — EXACT match only: "VBT_ONBOARD:" must NOT answer true for
// "VBT_ONBOARD_SNAPSHOT:".
export function isDeclared(p) {
    return !!(p && nvState.byPrefix[p]);
}

// buildNote(key, ...parts) mirrors the server's notePrefixBuild:
//   prefix + part + ':' for each part.
// A bound part that is empty returns '' — an unbound purpose could be replayed
// against a different target, and the server refuses an unbound prefix anyway.
export function buildNote(key, ...parts) {
    const base = prefix(key);
    if (!base) return '';

    const entry = nvState.byKey[key];
    if (entry && entry.scope !== 'money_door') {
        nvState.error = 'purpose "' + key + '" is not a client-namable money door';
        return '';
    }

    let out = base;
    for (const part of parts) {
        const s = (part === undefined || part === null) ? '' : String(part);
        if (s === '') {
            nvState.error = 'refused to build "' + key + '": a bound part is empty (an unbound purpose can be replayed)';
            return '';
        }
        out += s + ':';
    }
    return out;
}

// prepareNote is the ergonomic form for a payment site: it ensures the
// vocabulary is loaded, then builds the note. Returns '' with a stated reason on
// any refusal, so the caller can refuse to sign.
export async function prepareNote(key, ...parts) {
    if (!nvState.loaded) await load();
    if (!nvState.loaded) return '';
    return buildNote(key, ...parts);
}

// ---------------------------------------------------------------------------
// Introspection for surfaces
// ---------------------------------------------------------------------------
export function ready() { return nvState.loaded; }
export function moneyDoors() { return nvState.moneyDoors.slice(); }
export function purposes() { return nvState.purposes.slice(); }
export function describe(key) {
    const e = nvState.byKey[key];
    if (!e) return null;
    return { key: e.key, prefix: e.prefix, scope: e.scope, direction: e.direction, description: e.description };
}

export const NoteVocab = {
    load,
    ready,
    prefix,
    isDeclared,
    buildNote,
    prepareNote,
    moneyDoors,
    purposes,
    describe,
    state: nvState,
    ENDPOINT: NV_ENDPOINT,
};

window.NoteVocab = NoteVocab;

export default NoteVocab;
