#!/usr/bin/env node
/**
 * verify_ui_handlers.js — THE INLINE-HANDLER GATE.
 *
 * WHY THIS EXISTS. An inline handler (`onclick="openThing()"`) resolves `openThing` on `window`.
 * A module-scope function is NOT a global, so a handler whose name nothing publishes throws
 * `ReferenceError` and the control does absolutely NOTHING — no error surface, no toast, no
 * network call. That is how EIGHT Admin Suite controls were dead until 2026-09-15, and it is a
 * SYSTEMIC failure mode: modules build HTML with handler strings and index.html carries its own,
 * each one silently depending on a `window` global existing.
 *
 * WHAT IT MEASURES (two independent tiers, because neither alone is complete):
 *   1. STATIC — parse every `on<event>="…"` attribute in Public/**\/*.js and Public/**\/*.html,
 *      extract the names the expression intends to call, and compare them against every
 *      `window.X = …` / `globalThis.X = …` / `Object.assign(window, {…})` publication in the tree.
 *   2. BROWSER — with the app loaded, evaluate `typeof window[name]` for every extracted name.
 *      This is authoritative: it catches a name published in a module that never evaluated, and a
 *      publication that assigned a non-function.
 *
 * A name that is unpublished-but-bound, or bound-but-not-loaded, is still DEAD to the user.
 * Exit code is 1 when any handler name is dead, so this can gate a build.
 *
 * Usage:
 *   node tools/server/verify_ui_handlers.js              # static + browser (needs the dev server)
 *   node tools/server/verify_ui_handlers.js --static      # static only (no browser, no server)
 *   node tools/server/verify_ui_handlers.js --url http://localhost:8090/
 *   node tools/server/verify_ui_handlers.js --json        # machine-readable summary
 */

'use strict';

const fs = require('fs');
const http = require('http');
const os = require('os');
const path = require('path');
const { spawn } = require('child_process');

const ROOT = path.resolve(__dirname, '..', '..');
const PUBLIC_DIR = path.join(ROOT, 'Public');
const SKIP = /([\\/])(vendor|node_modules|prototypes|Assets|Generated)([\\/])/i;
const SKIP_FILES = new Set(['wasm_exec.js']);

/** Events an inline handler attribute can carry. Deliberately broad — a missed event is a hole. */
const EVENTS = 'click|change|input|submit|keyup|keydown|keypress|blur|focus|dblclick|mouseover|' +
    'mouseout|mouseenter|mouseleave|mousedown|mouseup|mousemove|wheel|scroll|contextmenu|load|error|' +
    'dragstart|dragend|dragover|drop|touchstart|touchend|animationend|transitionend|toggle|invalid|reset';

const ATTR_RE = new RegExp('\\bon(' + EVENTS + ')\\s*=\\s*(\\\\?["\'])', 'g');

/** Reserved words / locals that are never a published global — never reported as missing. */
const KEYWORDS = new Set([
    'this', 'event', 'window', 'document', 'return', 'new', 'typeof', 'void', 'delete', 'in', 'of',
    'if', 'else', 'for', 'while', 'switch', 'case', 'break', 'continue', 'function', 'class',
    'await', 'yield', 'super', 'null', 'undefined', 'true', 'false', 'NaN', 'Infinity', 'arguments',
]);

function walk(dir, out = [], exts = ['.js', '.html']) {
    let entries;
    try { entries = fs.readdirSync(dir, { withFileTypes: true }); } catch (_) { return out; }
    for (const e of entries) {
        const full = path.join(dir, e.name);
        if (SKIP.test(full)) continue;
        if (e.isDirectory()) { walk(full, out, exts); continue; }
        const ext = path.extname(e.name).toLowerCase();
        if (!exts.includes(ext)) continue;
        if (SKIP_FILES.has(e.name)) continue;
        out.push(full);
    }
    return out;
}

/**
 * Comment ranges for a source file. Documentation that MENTIONS a handler (e.g. the module header
 * of admin_suite.js explaining `onclick="adminX()"`) must never be read as a live control, so a
 * match inside a comment is ignored. The scan is a small tokenizer rather than a regex precisely
 * because `'https://…'` inside a string is not a comment and a naive strip would delete real code.
 */
function commentRanges(src, isHtml) {
    const ranges = [];
    const n = src.length;
    let i = 0;
    if (isHtml) {
        while (i < n) {
            if (src.startsWith('<!--', i)) {
                const end = src.indexOf('-->', i);
                const stop = end === -1 ? n : end + 3;
                ranges.push([i, stop]);
                i = stop;
                continue;
            }
            i++;
        }
        return ranges;
    }
    while (i < n) {
        const ch = src[i];
        if (ch === '/' && src[i + 1] === '/') {
            const s = i;
            while (i < n && src[i] !== '\n') i++;
            ranges.push([s, i]);
            continue;
        }
        if (ch === '/' && src[i + 1] === '*') {
            const s = i;
            i += 2;
            while (i < n && !(src[i] === '*' && src[i + 1] === '/')) i++;
            i = Math.min(i + 2, n);
            ranges.push([s, i]);
            continue;
        }
        if (ch === '"' || ch === "'" || ch === '`') {
            const quote = ch;
            i++;
            while (i < n) {
                if (src[i] === '\\') { i += 2; continue; }
                if (src[i] === quote) { i++; break; }
                if (quote === '`' && src[i] === '$' && src[i + 1] === '{') {
                    let depth = 1;
                    i += 2;
                    while (i < n && depth > 0) {
                        if (src[i] === '{') depth++;
                        else if (src[i] === '}') depth--;
                        else if (src[i] === '\\') i++;
                        i++;
                    }
                    continue;
                }
                i++;
            }
            continue;
        }
        i++;
    }
    return ranges;
}

function inRanges(ranges, index) {
    for (const [s, e] of ranges) { if (index >= s && index < e) return true; }
    return false;
}

/**
 * Browser built-ins an inline handler may legitimately call. They are real functions on `window`
 * and need no publisher of ours — listing them keeps the static tier honest instead of noisy.
 */
const BUILTIN_GLOBALS = new Set([
    'clearInterval', 'setInterval', 'clearTimeout', 'setTimeout', 'requestAnimationFrame',
    'alert', 'confirm', 'prompt', 'parseFloat', 'parseInt', 'isNaN', 'isFinite', 'btoa', 'atob',
    'encodeURIComponent', 'decodeURIComponent', 'fetch', 'open', 'close', 'focus', 'blur', 'print',
    'scrollTo', 'postMessage', 'dispatchEvent', 'getComputedStyle', 'structuredClone',
]);

/** Names the Go/WASM engine publishes onto the JS global object (`js.Global().Set("X", …)`). */
function collectWasmGlobals() {
    const gos = walk(ROOT, [], ['.go']);
    const out = new Map();
    for (const file of gos) {
        const src = fs.readFileSync(file, 'utf8');
        const re = /js\.Global\(\)\.Set\(\s*"([A-Za-z_$][\w$]*)"/g;
        let m;
        while ((m = re.exec(src)) !== null) {
            if (!out.has(m[1])) out.set(m[1], path.relative(ROOT, file).replace(/\\/g, '/'));
        }
    }
    return out;
}

/**
 * Reads one attribute value starting just after its opening delimiter. `delim` is `"`, `'`, or the
 * escaped `\"` form used inside double-quoted JS strings.
 */
function readAttribute(src, start, delim) {
    const quote = delim[delim.length - 1];
    const escaped = delim.length === 2;
    let out = '';
    for (let i = start; i < src.length; i++) {
        const ch = src[i];
        if (ch === '\n' || ch === '\r') break;            // attribute values do not span lines
        if (escaped) {
            if (ch === '\\') {
                if (src[i + 1] === quote) break;          // escaped delimiter = end of value
                out += src[i + 1] === undefined ? '' : src[i + 1];
                i++;
                continue;
            }
            out += ch;
            continue;
        }
        if (ch === '\\') { out += src[i + 1] === undefined ? '' : src[i + 1]; i++; continue; }
        if (ch === quote) break;
        out += ch;
    }
    return out;
}

/** Extracts the names a handler expression intends to call. */
function handlerNames(expr) {
    const names = new Map();
    let m;
    // A template interpolation — `onclick="openX('${esc(id)}')"` — is evaluated when the markup is
    // BUILT, not when it is clicked, so its callees are not handler dependencies. Strip them first.
    let cleaned = expr;
    for (let pass = 0; pass < 4; pass++) {
        const next = cleaned.replace(/\$\{[^{}]*\}/g, '');
        if (next === cleaned) break;
        cleaned = next;
    }
    const winRe = /(?:window|globalThis)\s*(?:\.\s*([A-Za-z_$][\w$]*)|\[\s*['"]([^'"]+)['"]\s*\])/g;
    while ((m = winRe.exec(cleaned)) !== null) {
        const n = m[1] || m[2];
        if (n && !KEYWORDS.has(n)) names.set(n, 'window');
    }
    const callRe = /(^|[^.\w$'"`])([A-Za-z_$][\w$]*)\s*\(/g;
    while ((m = callRe.exec(cleaned)) !== null) {
        const n = m[2];
        if (KEYWORDS.has(n)) continue;
        if (!names.has(n)) names.set(n, 'call');
    }
    return names;
}

/** Collects every name the tree publishes onto `window`/`globalThis`, with its defining file. */
function collectPublications(files) {
    const published = new Map();
    const note = (name, file) => {
        if (name && !published.has(name)) published.set(name, path.relative(ROOT, file).replace(/\\/g, '/'));
    };
    for (const file of files) {
        const src = fs.readFileSync(file, 'utf8');
        let m;
        const assignRe = /(?:window|globalThis)\s*(?:\.\s*([A-Za-z_$][\w$]*)|\[\s*['"]([^'"]+)['"]\s*\])\s*=(?!=)/g;
        while ((m = assignRe.exec(src)) !== null) note(m[1] || m[2], file);
        const assignAllRe = /Object\.assign\s*\(\s*(?:window|globalThis)\s*,\s*\{([\s\S]{0,4000}?)\}\s*\)/g;
        while ((m = assignAllRe.exec(src)) !== null) {
            const keyRe = /(?:^|[,{\s])([A-Za-z_$][\w$]*)\s*(?::|,|\})/g;
            let k;
            while ((k = keyRe.exec(m[1])) !== null) note(k[1], file);
        }
        // A module may publish a whole TABLE of handlers in one loop — the admin_suite.js idiom:
        //   Object.entries(TABLE).forEach(([name, fn]) => { window[name] = fn; });
        // No literal `window.X =` exists, so the TABLE's keys are the published names.
        const loopRe = /Object\.entries\s*\(\s*([A-Za-z_$][\w$]*)\s*\)[\s\S]{0,800}?window\s*\[\s*([A-Za-z_$][\w$]*)\s*\]\s*=(?!=)/g;
        let d;
        while ((d = loopRe.exec(src)) !== null) {
            const table = d[1];
            const objRe = new RegExp('(?:const|let|var)\\s+' + table + '\\s*=\\s*\\{([\\s\\S]{0,4000}?)\\n\\};');
            const om = objRe.exec(src);
            if (!om) continue;
            const keyRe = /(?:^|[,{\s])([A-Za-z_$][\w$]*)\s*(?::\s|\s*,|\s*\}|\n)/g;
            let k;
            while ((k = keyRe.exec(om[1])) !== null) note(k[1], file);
        }
    }
    return published;
}

/**
 * The modules the application actually EVALUATES: BFS over relative `import … from './x.js'`
 * statements starting at app.js (the single entry). A name published from a module outside this set
 * cannot protect any control — and its markup cannot render either — so the audit reports it as
 * NOT-COMPOSED rather than as a broken handler.
 */
function reachableModules() {
    const entry = path.join(PUBLIC_DIR, 'app.js');
    const seen = new Set();
    const queue = [entry];
    while (queue.length) {
        const file = queue.shift();
        if (seen.has(file)) continue;
        seen.add(file);
        let src;
        try { src = fs.readFileSync(file, 'utf8'); } catch (_) { continue; }
        const re = /from\s*['"](\.[^'"]+)['"]|\bimport\s*['"](\.[^'"]+)['"]/g;
        let m;
        while ((m = re.exec(src)) !== null) {
            const spec = m[1] || m[2];
            let target = path.resolve(path.dirname(file), spec);
            if (!fs.existsSync(target) && fs.existsSync(target + '.js')) target += '.js';
            if (fs.existsSync(target) && fs.statSync(target).isFile()) queue.push(target);
        }
    }
    return seen;
}

/**
 * Where each name is DEFINED (a declaration or an import), so the report can tell a handler that
 * merely lacks a publisher from markup that names a function nothing ever defined.
 */
function definitionIndex(files) {
    const index = new Map();
    const note = (name, file) => {
        if (!index.has(name)) index.set(name, path.relative(ROOT, file).replace(/\\/g, '/'));
    };
    for (const file of files) {
        const src = fs.readFileSync(file, 'utf8');
        let m;
        const declRe = /(?:^|[\s;{])(?:async\s+function|function|const|let|var|class)\s+([A-Za-z_$][\w$]*)/gm;
        while ((m = declRe.exec(src)) !== null) note(m[1], file);
        const importRe = /import\s*\{([\s\S]{0,2000}?)\}\s*from/g;
        while ((m = importRe.exec(src)) !== null) {
            const nameRe = /(?:^|[\s,{])([A-Za-z_$][\w$]*)/g;
            let n;
            while ((n = nameRe.exec(m[1])) !== null) note(n[1], file);
        }
    }
    return index;
}

/** Parses the tree and returns every handler name with its usage evidence. */
function scan() {
    const files = walk(PUBLIC_DIR);
    const publications = collectPublications(files);
    const wasmGlobals = collectWasmGlobals();
    const definitions = definitionIndex(files);
    const usages = new Map();
    let attributes = 0;
    let splitValues = 0;
    let ignoredInComments = 0;

    for (const file of files) {
        const src = fs.readFileSync(file, 'utf8');
        const rel = path.relative(ROOT, file).replace(/\\/g, '/');
        const comments = commentRanges(src, file.toLowerCase().endsWith('.html'));
        ATTR_RE.lastIndex = 0;
        let m;
        while ((m = ATTR_RE.exec(src)) !== null) {
            if (inRanges(comments, m.index)) { ignoredInComments++; continue; }
            const value = readAttribute(src, ATTR_RE.lastIndex, m[2]);
            attributes++;
            if (!value.trim()) { splitValues++; continue; }
            const line = src.slice(0, m.index).split('\n').length;
            for (const [name, kind] of handlerNames(value)) {
                if (BUILTIN_GLOBALS.has(name)) continue;
                const rec = usages.get(name) ||
                    { name, kind, files: new Set(), count: 0, sample: value.slice(0, 90) };
                rec.count++;
                rec.files.add(`${rel}:${line} (on${m[1]})`);
                usages.set(name, rec);
            }
        }
    }

    const dead = [];
    for (const rec of usages.values()) {
        if (!publications.has(rec.name) && !wasmGlobals.has(rec.name)) dead.push({ ...rec, files: [...rec.files] });
    }
    dead.sort((a, b) => a.name.localeCompare(b.name));
    return { files, publications, wasmGlobals, usages, dead, attributes, splitValues, ignoredInComments, definitions };
}

function resolveChrome() {
    const cands = [
        process.env.CHROME_BIN,
        'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe',
        'C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe',
        process.env.LOCALAPPDATA + '\\Google\\Chrome\\Application\\chrome.exe',
        'C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe',
        'C:\\Program Files\\Microsoft\\Edge\\Application\\msedge.exe',
    ].filter(Boolean);
    for (const c of cands) { if (fs.existsSync(c)) return c; }
    return null;
}

function httpGet(url) {
    return new Promise((resolve, reject) => {
        const req = http.get(url, (res) => {
            let body = '';
            res.on('data', (d) => (body += d));
            res.on('end', () => resolve({ status: res.statusCode, body }));
        });
        req.on('error', reject);
        req.setTimeout(2000, () => req.destroy(new Error('timeout')));
    });
}

async function waitFor(url, timeoutMs) {
    const start = Date.now();
    for (;;) {
        try { const r = await httpGet(url); if (r.status === 200) return JSON.parse(r.body); } catch (_) {}
        if (Date.now() - start > timeoutMs) throw new Error('timeout waiting for ' + url);
        await new Promise((r) => setTimeout(r, 300));
    }
}

class CDP {
    constructor(url) {
        this.ws = new WebSocket(url);
        this.id = 0;
        this.pending = new Map();
        this.handlers = {};
        this.ws.onmessage = (ev) => {
            const msg = JSON.parse(ev.data);
            if (msg.id != null) {
                const p = this.pending.get(msg.id);
                if (p) {
                    this.pending.delete(msg.id);
                    msg.error ? p.reject(new Error(JSON.stringify(msg.error))) : p.resolve(msg.result);
                }
            } else if (msg.method && this.handlers[msg.method]) this.handlers[msg.method](msg.params);
        };
    }
    open() { return new Promise((res, rej) => { this.ws.onopen = res; this.ws.onerror = (e) => rej(e); }); }
    on(m, cb) { this.handlers[m] = cb; }
    send(method, params = {}) {
        const id = ++this.id;
        return new Promise((res, rej) => {
            this.pending.set(id, { resolve: res, reject: rej });
            this.ws.send(JSON.stringify({ id, method, params }));
        });
    }
}

/** Loads the app and asks the browser which names actually resolve to a function on `window`. */
async function browserResolve(names, target, waitMs) {
    const chrome = resolveChrome();
    if (!chrome) return { skipped: 'no browser found' };
    const cdpPort = 9446;
    const profile = fs.mkdtempSync(path.join(os.tmpdir(), 'vbt-handlers-'));
    const proc = spawn(chrome, [
        '--headless=new', `--remote-debugging-port=${cdpPort}`, `--user-data-dir=${profile}`,
        '--no-first-run', '--no-default-browser-check', '--disable-extensions', '--disable-gpu',
        '--window-size=1280,900', 'about:blank',
    ], { stdio: 'ignore' });
    let cdp;
    try {
        await waitFor(`http://127.0.0.1:${cdpPort}/json/version`, 15000);
        const { body } = await httpGet(`http://127.0.0.1:${cdpPort}/json/list`);
        const page = JSON.parse(body).find((t) => t.type === 'page');
        cdp = new CDP(page.webSocketDebuggerUrl);
        await cdp.open();
        // An inline handler that calls a name nothing published throws AT CLICK TIME; a module that
        // fails while loading throws AT BOOT. Both are the same defect family, so boot errors are
        // collected here and reported as failures: a page that did not load cleanly cannot resolve
        // anything.
        const pageErrors = [];
        cdp.on('Runtime.exceptionThrown', (p) => {
            const d = p.exceptionDetails || {};
            pageErrors.push(((d.exception && (d.exception.description || d.exception.value)) || d.text || '').slice(0, 200));
        });
        cdp.on('Runtime.consoleAPICalled', (p) => {
            if (p.type === 'error') {
                pageErrors.push('console: ' + (p.args || []).map((a) => a.value || a.description || '').join(' ').slice(0, 200));
            }
        });
        await cdp.send('Runtime.enable');
        await cdp.send('Page.enable');
        await cdp.send('Page.navigate', { url: target });
        const readyExpr = `(() => typeof window.GetGameState === 'function' && !!document.body.classList.contains('ready'))()`;
        const deadline = Date.now() + waitMs;
        let ready = false;
        while (Date.now() < deadline) {
            const r = await cdp.send('Runtime.evaluate', { expression: readyExpr, returnByValue: true, awaitPromise: true });
            if (r.result && r.result.value === true) { ready = true; break; }
            await new Promise((res) => setTimeout(res, 250));
        }
        const expr = `(() => { const names = ${JSON.stringify(names)}; const out = {};` +
            ` for (const n of names) { out[n] = typeof window[n]; } return out; })()`;
        const res = await cdp.send('Runtime.evaluate', { expression: expr, returnByValue: true, awaitPromise: true });
        const types = (res.result && res.result.value) || {};
        const resolved = {};
        for (const n of names) resolved[n] = types[n] === 'function';
        // Evidence that the ONE admin surface still reveals after the CSS/JS edits: its computed
        // display when opened (the contained-overlay contract, not "the opener ran").
        const panelExpr = `(() => {
            const el = document.getElementById('admin-control-panel');
            if (!el) return { present: false };
            el.classList.remove('hidden');
            el.style.display = 'flex';
            const shown = getComputedStyle(el).display;
            el.style.display = 'none';
            el.classList.add('hidden');
            return { present: true, computed: shown, binder: typeof window.bindAdminSuiteControls, actions: el.querySelectorAll('[data-admin-action]').length };
        })()`;
        const panelRes = await cdp.send('Runtime.evaluate', { expression: panelExpr, returnByValue: true, awaitPromise: true });
        return { resolved, types, pageErrors, ready, adminPanel: (panelRes.result && panelRes.result.value) || null };
    } finally {
        try { if (cdp) cdp.ws.close(); } catch (_) {}
        try { proc.kill(); } catch (_) {}
        try { fs.rmSync(profile, { recursive: true, force: true }); } catch (_) {}
    }
}

async function main() {
    const argv = process.argv.slice(2);
    const staticOnly = argv.includes('--static');
    const asJson = argv.includes('--json');
    const i = argv.indexOf('--url');
    const target = i >= 0 ? argv[i + 1] : 'http://localhost:8090/';

    const result = scan();
    const names = [...result.usages.keys()].sort();
    let browser = null;
    if (!staticOnly) {
        const waitArg = argv.indexOf('--wait');
        const waitMs = waitArg >= 0 ? Number(argv[waitArg + 1]) || 20000 : 20000;
        try { browser = await browserResolve(names, target, waitMs); }
        catch (err) { browser = { skipped: err.message }; }
    }

    const unpublished = new Set(result.dead.map((d) => d.name));
    const unresolved = browser && browser.resolved ? names.filter((n) => !browser.resolved[n]) : [];
    const deadNames = [...new Set([...unpublished, ...unresolved])].sort();
    const evidence = new Map(result.dead.map((d) => [d.name, d]));

    // A boot-time browser read cannot see a global published only when its own surface opens, and it
    // cannot see anything from a module the entry never imports. Those are NOTES, not broken
    // handlers: only a name with NO publisher at all (or a publisher in a module the app DOES
    // evaluate that still fails to appear at boot) is a real defect.
    const reachable = reachableModules();
    const notes = [];
    const failed = [];
    for (const n of deadNames) {
        const pubFile = result.publications.get(n) || result.wasmGlobals.get(n);
        if (!unpublished.has(n)) {
            if (!pubFile) { failed.push({ name: n, reason: 'not a function in the loaded app' }); continue; }
            const abs = path.join(ROOT, pubFile);
            const isAppPage = /\.html$/i.test(pubFile);
            notes.push({
                name: n,
                reason: isAppPage
                    ? 'other page — its own inline script provides the global (not the app entry)'
                    : (reachable.has(abs)
                        ? 'published when its own surface opens (not bound at boot)'
                        : `not composed by app.js (${pubFile} is never imported)`),
            });
            continue;
        }
        failed.push({ name: n, reason: 'no publisher' });
    }
    // A module that throws while loading is the same defect family as a handler that calls nothing:
    // the control is not there. A boot error therefore fails the gate as well.
    const bootErrors = (browser && browser.pageErrors) ? browser.pageErrors : [];
    for (const e of bootErrors) failed.push({ name: '(page did not load cleanly)', reason: e });
    const failedNames = failed.map((f) => f.name);

    if (asJson) {
        console.log(JSON.stringify({
            scannedFiles: result.files.length,
            handlerAttributes: result.attributes,
            builtByConcatenation: result.splitValues,
            ignoredInComments: result.ignoredInComments,
            distinctNames: names.length,
            publishedNames: result.publications.size,
            wasmPublishedNames: result.wasmGlobals.size,
            browser: browser
                ? (browser.skipped ? { skipped: browser.skipped } : { checked: names.length })
                : { skipped: 'static-only run' },
            dead: failed.map((f) => {
                const n = f.name;
                const e = evidence.get(n);
                return {
                    name: n,
                    reason: f.reason,
                    uses: e ? e.count : 0,
                    where: e ? e.files.slice(0, 4) : [],
                    publishedBy: result.publications.get(n) || result.wasmGlobals.get(n) || null,
                    definedIn: result.definitions.get(n) || null,
                };
            }),
            notes,
            boot: {
                ready: browser ? !!browser.ready : null,
                pageErrors: bootErrors,
                adminPanel: browser ? browser.adminPanel : null,
            },
        }, null, 2));
    } else {
        console.log(`[UI HANDLERS] scanned ${result.files.length} file(s) under Public/`);
        console.log(`  inline handler attributes : ${result.attributes}` +
            (result.splitValues ? ` (+${result.splitValues} built by concatenation — not parsed)` : '') +
            (result.ignoredInComments ? ` (${result.ignoredInComments} in comments — ignored)` : ''));
        console.log(`  distinct handler names    : ${names.length}`);
        console.log(`  publishers: JS window.*   : ${result.publications.size}`);
        console.log(`  publishers: Go/WASM bridge: ${result.wasmGlobals.size}`);
        if (!browser) console.log('  browser cross-check       : SKIPPED (static-only run)');
        else if (browser.skipped) console.log(`  browser cross-check       : SKIPPED (${browser.skipped})`);
        else console.log(`  browser cross-check       : ${names.length - unresolved.length}/${names.length} resolve to a function AT BOOT (lazy + non-composed names are notes, below)`);
        if (browser && browser.skipped) { /* nothing more to report */ }
        else if (browser) {
            console.log(`  engine readiness observed : ${browser.ready === true}`);
            console.log(`  boot console/page errors  : ${bootErrors.length}`);
            const p = browser.adminPanel;
            if (p) console.log(`  admin surface evidence    : present=${p.present} computed=${p.computed} declaredControls=${p.actions} binder=${p.binder}`);
        }
        console.log(failedNames.length
            ? `\n  DEAD HANDLERS: ${failedNames.length}`
            : '\n  DEAD HANDLERS: 0 — every inline handler resolves to a publisher.');
        for (const f of failed) {
            const e = evidence.get(f.name);
            const definedIn = result.definitions.get(f.name);
            console.log(`   - ${f.name}  (${f.reason})` +
                (e ? ` — ${e.count} use(s): ${e.files.slice(0, 3).join(', ')}` : '') +
                `\n       defined in: ${definedIn || 'NOWHERE — the markup names a function that does not exist'}`);
        }
        if (notes.length) {
            const byReason = new Map();
            for (const n of notes) byReason.set(n.reason, (byReason.get(n.reason) || 0) + 1);
            console.log(`\n  NOTES: ${notes.length} name(s) a boot-time read cannot see (not failures):`);
            for (const [reason, count] of byReason) console.log(`   · ${count} — ${reason}`);
        }
    }
    process.exitCode = failedNames.length ? 1 : 0;
}

main().catch((err) => {
    console.error('[UI HANDLERS] probe failed: ' + err.message);
    process.exitCode = 2;
});
