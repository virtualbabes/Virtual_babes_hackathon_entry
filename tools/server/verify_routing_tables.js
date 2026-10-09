#!/usr/bin/env node
/**
 * verify_routing_tables.js — THE ROUTING-TABLE GATE.
 *
 * WHY THIS EXISTS. A navigation table maps a UI id to the NAME of the function that opens it
 * (`WD_ROUTES`, the constellation hub's `panelMap`, …). The openers are invoked through a
 * `typeof window[name] === 'function'` guard, so a name nothing publishes does not throw: the
 * guard is false, the dispatcher returns, and the button is DEAD with no error surface — the
 * exact way `panelMap.governance: 'openGovernance'` (the real publisher is
 * `openGovernancePanel`) silently did nothing.
 *
 * `verify_ui_handlers.js` cannot see this class: it resolves names appearing in MARKUP
 * (`onclick="x()"`). These names appear in a JS TABLE, which no onclick scan reads. Two input
 * classes, two gates, no duplicated logic.
 *
 * WHAT IT MEASURES (static — no browser, no server)
 *   A. every `key: 'openX'` opener REFERENCE in first-party JS must have a publisher
 *      (`window.X =`, `window['X'] =`, `globalThis.X =`, `Object.assign(window, {X…})`, or the
 *      Go/WASM bridge `js.Global().Set("X"`). A `null` value is a deliberate "no route" and is
 *      skipped, never reported.
 *   B. every route in `WD_ROUTES` must name a tab DECLARED in `WD_CATEGORIES` — a route no
 *      feature button can reach is an orphan entry.
 *   C. self-check: the tables must PARSE (a regex that silently matches nothing would make this
 *      gate pass by measuring nothing). `--selftest` proves the detector can fail.
 *
 * Usage: node tools/server/verify_routing_tables.js [--json|--selftest]
 */

'use strict';

const fs = require('fs');
const path = require('path');

const ROOT = path.resolve(__dirname, '..', '..');
const PUBLIC_DIR = path.join(ROOT, 'Public');
const SKIP_DIR = /([\\/])(vendor|node_modules|prototypes|Assets|Generated|devdata|\.git)([\\/])/i;
const SKIP_FILES = new Set(['wasm_exec.js']);

/** The one owner of the World Dashboard's routing table. */
const WD_FILE = path.join(PUBLIC_DIR, 'js', 'world_dashboard.js');

/** `key: 'openX'` — the routing-table shape, wherever it appears. */
const OPENER_REF_RE = /([A-Za-z_$][\w$]*)\s*:\s*'(open[A-Z][A-Za-z0-9_]*)'/g;

function strip(src) {
    let out = '';
    let i = 0;
    const n = src.length;
    while (i < n) {
        const two = src.slice(i, i + 2);
        if (two === '//') {
            let end = src.indexOf('\n', i);
            if (end === -1) end = n;
            out += ' '.repeat(end - i); i = end; continue;
        }
        if (two === '/*') {
            const end = src.indexOf('*/', i + 2);
            const stop = end === -1 ? n : end + 2;
            out += ' '.repeat(stop - i); i = stop; continue;
        }
        const c = src[i];
        if (c === "'" || c === '"' || c === '`') {
            let j = i + 1;
            while (j < n) {
                if (src[j] === '\\') { j += 2; continue; }
                if (src[j] === c) break;
                if (c !== '`' && src[j] === '\n') break;
                j++;
            }
            const stop = Math.min(j + 1, n);
            out += src.slice(i, stop); i = stop; continue;
        }
        out += c; i++;
    }
    return out;
}

function walk(dir, out = []) {
    let entries;
    try { entries = fs.readdirSync(dir, { withFileTypes: true }); } catch (_) { return out; }
    for (const e of entries) {
        const full = path.join(dir, e.name);
        if (SKIP_DIR.test(full)) continue;
        if (e.isDirectory()) { walk(full, out); continue; }
        if (path.extname(e.name).toLowerCase() !== '.js') continue;
        if (SKIP_FILES.has(e.name)) continue;
        out.push(full);
    }
    return out;
}

/** Every `key: 'openX'` reference in the tree, with file + line for the report. */
function openerReferences() {
    const refs = [];
    for (const file of walk(PUBLIC_DIR)) {
        let src;
        try { src = fs.readFileSync(file, 'utf8'); } catch (_) { continue; }
        const text = strip(src);
        OPENER_REF_RE.lastIndex = 0;
        let m;
        while ((m = OPENER_REF_RE.exec(text)) !== null) {
            const line = text.slice(0, m.index).split('\n').length;
            refs.push({
                key: m[1], name: m[2],
                file: path.relative(ROOT, file).split(path.sep).join('/'), line,
            });
        }
    }
    return refs;
}

// ---------------------------------------------------------------------------
// The publisher index: who PROMISES `window.X` (JS) or `js.Global().Set("X")` (Go/WASM bridge).
// ---------------------------------------------------------------------------
const JS_PUBLISH_RE = /\b(?:window|globalThis)\s*(?:\.\s*([A-Za-z_$][\w$]*)|\[\s*'([A-Za-z_$][\w$]*)'\s*\])\s*=[^=]/g;
const ASSIGN_WINDOW_RE = /\bObject\.assign\s*\(\s*(?:window|globalThis)\s*,/g;
const GO_PUBLISH_RE = /js\.Global\(\)\.Set\(\s*"([A-Za-z_$][\w$]*)"/g;
const OBJECT_KEY_RE = /(?:^|[{,])\s*([A-Za-z_$][\w$]*)\s*(?::|,|\})/g;

/** Body of the `{…}` starting at/after `from` in an already-stripped text (null when unbalanced). */
function braceBlock(text, from) {
    const open = text.indexOf('{', from);
    if (open === -1) return null;
    let depth = 0;
    for (let i = open; i < text.length; i++) {
        if (text[i] === '{') depth++;
        else if (text[i] === '}') {
            depth--;
            if (depth === 0) return { body: text.slice(open + 1, i), start: open, end: i };
        }
    }
    return null;
}

/**
 * Body of the next OBJECT or ARRAY literal at/after `from`, whichever opens first.
 * `WD_ROUTES` is an object but `WD_CATEGORIES` is an ARRAY — matching only braces against the
 * array returned the FIRST CATEGORY's body (9 features instead of 67) and every other feature
 * then looked like an orphan route. The delimiter is taken from the source, never assumed.
 */
function literalBlock(text, from) {
    const objAt = text.indexOf('{', from);
    const arrAt = text.indexOf('[', from);
    let open;
    if (objAt === -1) open = arrAt;
    else if (arrAt === -1) open = objAt;
    else open = Math.min(objAt, arrAt);
    if (open === -1) return null;
    const oc = text[open];
    const cc = oc === '{' ? '}' : ']';
    let depth = 0;
    for (let i = open; i < text.length; i++) {
        if (text[i] === oc) depth++;
        else if (text[i] === cc) {
            depth--;
            if (depth === 0) return { body: text.slice(open + 1, i), start: open, end: i };
        }
    }
    return null;
}

function collectPublishers() {
    const js = new Map();
    for (const file of walk(PUBLIC_DIR)) {
        let src;
        try { src = fs.readFileSync(file, 'utf8'); } catch (_) { continue; }
        const text = strip(src);
        const where = path.relative(ROOT, file).split(path.sep).join('/');
        JS_PUBLISH_RE.lastIndex = 0;
        let m;
        while ((m = JS_PUBLISH_RE.exec(text)) !== null) js.set(m[1] || m[2], where);
        // `Object.assign(window, { a, b })` — the publish-a-table idiom.
        ASSIGN_WINDOW_RE.lastIndex = 0;
        while ((m = ASSIGN_WINDOW_RE.exec(text)) !== null) {
            const block = braceBlock(text, ASSIGN_WINDOW_RE.lastIndex - 1);
            if (!block) continue;
            OBJECT_KEY_RE.lastIndex = 0;
            let k;
            while ((k = OBJECT_KEY_RE.exec(block.body)) !== null) if (!js.has(k[1])) js.set(k[1], where);
        }
    }
    const go = new Set();
    for (const name of fs.readdirSync(ROOT).filter((f) => f.endsWith('.go'))) {
        let src;
        try { src = fs.readFileSync(path.join(ROOT, name), 'utf8'); } catch (_) { continue; }
        GO_PUBLISH_RE.lastIndex = 0;
        let m;
        while ((m = GO_PUBLISH_RE.exec(src)) !== null) go.add(m[1]);
    }
    return { js, go };
}

/** The body of `const NAME = {…}` / `const NAME = […]` in an already-stripped text. */
function tableBody(text, name) {
    const re = new RegExp('\\b(?:const|let|var)\\s+' + name + '\\s*=\\s*');
    const m = re.exec(text);
    if (!m) return null;
    return literalBlock(text, m.index);
}

/** `key: 'value'` pairs from a table body (literal values only). */
function literalPairs(body) {
    const re = /([A-Za-z_$][\w$]*)\s*:\s*'([^']*)'/g;
    const out = new Map();
    let m;
    while ((m = re.exec(body)) !== null) out.set(m[1], m[2]);
    return out;
}

/** Tab ids declared by `WD_CATEGORIES` via `{ tab: 'x', … }`. */
function declaredFeatureTabs(body) {
    const re = /\btab\s*:\s*'([^']+)'/g;
    const out = new Set();
    let m;
    while ((m = re.exec(body)) !== null) out.add(m[1]);
    return out;
}

/** Negative control: prove the detector reports a name nothing publishes. */
function selftest() {
    const sample = strip("const panelMap = {\n  a: 'openReal',\n  b: 'openMissing',\n  c: null,\n};\n");
    const found = [...sample.matchAll(OPENER_REF_RE)].map((m) => m[2]);
    const published = new Set(['openReal']);
    const dead = found.filter((n) => !published.has(n));
    const ok = found.length === 2 && dead.length === 1 && dead[0] === 'openMissing';
    console.log(`[ROUTING SELFTEST] extracted=${JSON.stringify(found)} dead=${JSON.stringify(dead)} ok=${ok}`);
    process.exitCode = ok ? 0 : 1;
}

function main() {
    const asJson = process.argv.includes('--json');
    const { js, go } = collectPublishers();
    const refs = openerReferences();
    const dead = refs.filter((r) => !js.has(r.name) && !go.has(r.name));

    const wd = strip(fs.readFileSync(WD_FILE, 'utf8'));
    const routesBlock = tableBody(wd, 'WD_ROUTES');
    const catsBlock = tableBody(wd, 'WD_CATEGORIES');
    const routes = routesBlock ? literalPairs(routesBlock.body) : new Map();
    const tabs = catsBlock ? declaredFeatureTabs(catsBlock.body) : new Set();

    const findings = [];
    if (routes.size === 0) findings.push("WD_ROUTES did not parse into any entry — the table or its name moved; without this the gate would pass by measuring nothing.");
    if (tabs.size === 0) findings.push('WD_CATEGORIES did not parse into any feature tab — same self-check.');
    if (refs.length === 0) findings.push("no `key: 'openX'` opener reference was found anywhere — same self-check.");

    const orphanRoutes = [...routes.keys()].filter((k) => !tabs.has(k)).sort();
    const openable = (name) => js.has(name) || go.has(name);
    const routed = [...tabs].filter((t) => routes.has(t) && openable(routes.get(t))).sort();
    const embedding = [...tabs].filter((t) => !routed.includes(t)).sort();
    const failures = dead.length + orphanRoutes.length + findings.length;

    if (asJson) {
        console.log(JSON.stringify({
            openerReferences: refs.length,
            jsPublishers: js.size,
            goBridgePublishers: go.size,
            wdRoutes: routes.size,
            wdFeatureTabs: tabs.size,
            routed,
            embedding,
            dead: dead.map((d) => ({ key: d.key, name: d.name, file: d.file, line: d.line })),
            orphanRoutes,
            parseFindings: findings,
            failures,
        }, null, 2));
    } else {
        console.log(`[ROUTING] opener references          : ${refs.length}`);
        console.log(`  publishers: JS window.*           : ${js.size}`);
        console.log(`  publishers: Go/WASM bridge        : ${go.size}`);
        console.log(`  WD_ROUTES entries                 : ${routes.size}`);
        console.log(`  WD_CATEGORIES feature tabs        : ${tabs.size}`);
        console.log(`  features routing out / embedding  : ${routed.length} / ${embedding.length}`);
        if (findings.length) {
            console.log(`\n  PARSE FINDINGS: ${findings.length}`);
            for (const f of findings) console.log(`   - ${f}`);
        }
        console.log(dead.length
            ? `\n  DEAD ROUTER NAMES: ${dead.length} — the guard is false, so the control does NOTHING.`
            : '\n  DEAD ROUTER NAMES: 0 — every opener reference resolves to a publisher.');
        for (const d of dead) console.log(`   - ${d.key} → '${d.name}'  (${d.file}:${d.line}) has NO publisher`);
        if (orphanRoutes.length) {
            console.log(`\n  ORPHAN ROUTES: ${orphanRoutes.length} — routed but not declared as a feature, so nothing can reach them.`);
            for (const o of orphanRoutes) console.log(`   - ${o} → '${routes.get(o)}'`);
        }
        console.log(failures
            ? `\n  RESULT: FAIL (${failures} finding(s))`
            : '\n  RESULT: PASS — every routing-table name resolves and every route is reachable.');
    }
    process.exitCode = failures ? 1 : 0;
}

if (process.argv.includes('--selftest')) selftest();
else main();
