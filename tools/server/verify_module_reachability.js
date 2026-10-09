#!/usr/bin/env node
/**
 * verify_module_reachability.js — THE COMPOSITION-GRAPH GATE.
 *
 * WHY THIS EXISTS. The Single-Entry Mandate makes `Public/index.html` a pure shell and
 * `Public/app.js` the ONE composition root: every first-party module must be REACHABLE from it.
 * A module nothing imports cannot render, cannot publish a global and cannot be reached by any
 * route — it is dead code with a file size. Nothing measured that, and the metric that looked
 * like it did (archscan's `isReferenced`) answers a different question: "is this path mentioned
 * anywhere?", which a mention INSIDE another unloaded module satisfies. Referenced is not
 * reachable, and the gap between them is where built-and-forgotten modules live.
 *
 * WHAT IT MEASURES
 *   roots  = `Public/app.js` (the single composition root) plus every module a PAGE loads
 *            directly (`<script type="module" src>` / `<script src>` in any .html at the repo
 *            root or directly under Public/), so a legacy page's own entry points count.
 *   edges  = ESM `import … from '…'`, `export … from '…'`, dynamic `import('…')` with a LITERAL
 *            specifier, and classic script injection (`<script src="…">` / `el.src = '…js'`).
 *   result = the transitive closure; a first-party file outside it is unreachable.
 *
 * FAILURE CRITERIA (exit 1)
 *   1. a specifier that resolves to NO file — the module silently never evaluates;
 *   2. a module that is unreachable and NOT in the baseline below;
 *   3. a baselined module that has BECOME reachable — a stale entry means the baseline is
 *      legitimising rot instead of recording it.
 *
 * THE BASELINE IS A HOLDING PEN, NOT AN APPROVAL. Each entry names the decision that owns it.
 * Wiring or retiring one is an edit to THIS list plus the source — and retiring a module needs
 * the operator's consent, so the agent never deletes one on its own authority.
 *
 * Usage:
 *   node tools/server/verify_module_reachability.js
 *   node tools/server/verify_module_reachability.js --json
 */

'use strict';

const fs = require('fs');
const path = require('path');

const ROOT = path.resolve(__dirname, '..', '..');
const PUBLIC_DIR = path.join(ROOT, 'Public');
const SKIP_DIR = /([\\/])(vendor|node_modules|prototypes|Assets|Generated|devdata|\.git)([\\/])/i;
const SKIP_FILES = new Set(['wasm_exec.js']);

/** The single composition root. */
const ROOTS = [path.join(PUBLIC_DIR, 'app.js')];

/**
 * Known-unreachable modules: repo-relative path → the decision that owns it. Every one is a
 * built module no entry point loads. They are REPORTED, never hidden: the entry exists so a NEW
 * one fails the gate while the operator decides wire-or-retire.
 */
const BASELINE = {
    // ── THE HOLDING PEN IS EMPTY (2026-09-15 g) ─────────────────────────────────────────────
    // Five modules were RETIRED out of this pen, on MEASURED grounds rather than on a reading:
    //
    //   Public/js/misc_panel.js            Public/js/systems_panel.js
    //   Public/js/market_creator_panel.js  Public/js/faith_extended.js
    //   Public/js/governance_extended.js
    //
    // WHY THEY COULD GO (each fact is a measurement, not a claim):
    //   1. NOTHING COMPOSES THEM — they sit outside the closure from app.js and the pages, which is
    //      what put them in this pen. A search of every OTHER first-party file for their filenames
    //      and for every global they publish returns ZERO hits, so no live surface called them.
    //   2. NO CAPABILITY IS EXCLUSIVE TO THEM — every /api/ literal they reference is referenced by
    //      a COMPOSED module too. The twelve endpoints whose only owner had been a shell were placed
    //      in 2026-09-15 (f); a re-measure against this gate's own --json closure reports 8
    //      literal-only misses, and each is in fact owned through the api('/path') convention
    //      (world_events.js, persistent_identity.js, ai_citizens.js, remaining_tabs.js). One owner
    //      per capability, and the owner is reachable.
    //   3. THEY WERE ALREADY BROKEN — the only five modules still handing an already-prefixed path
    //      to their own helper (api('/api/…')), so their requests could never have reached the
    //      server even if something had loaded them (see the verify:api gate).
    //
    // WHAT AN EMPTY PEN MEANS: a NEW unreachable module now fails this gate with nothing to hide
    // behind. Retirement is still the OPERATOR's call, so a future entry needs a NAMED decision
    // recorded here rather than an assumption.
    //
    // ── PRIOR RECORDS, CARRIED FORWARD ──────────────────────────────────────────────────────
    // MEASUREMENT CORRECTION (2026-09-15 f): the first pass was wrong in BOTH directions — it was
    //   INFLATED to 40+ "unowned" routes by a 152-endpoint blind GET sweep in misc_panel (whose
    //   results were summed into a single integer, naming routes other modules own), and it was
    //   BLIND to the /api prefix convention, because a composed module writes api('/pets/spawn')
    //   while these panels wrote api('/api/pets/spawn') — so matching on the literal path made a
    //   live owner read as absent. Re-measured with the prefix normalised, only TWELVE endpoints
    //   across all five modules had no live owner, and every one was placed.
    // THEME ENGINE = WIRE, NOT RETIRE (2026-09-15 e): Public/js/theme_engine.js was composed by
    //   app.js instead of being deleted, and its entry is GONE — this gate failing on a stale entry
    //   is the mechanical proof the verdict was executed. It is the only writer of
    //   --theme-accent/-rgb/-dim/-glow; the SCSS contract already existed and was inert without it,
    //   and its default element 'fire' IS the token the compiled :root ships, so the wire activates
    //   the contract without changing the shipped look. (f) It also reads the SERVED
    //   /api/theme/vector and a refused read keeps the last theme, setting data-theme-source.
};

// ---------------------------------------------------------------------------
// Comment / literal stripping. A commented-out import is not an edge, and a URL inside a string
// is not code. Removed text becomes spaces so offsets stay meaningful.
// ---------------------------------------------------------------------------
function strip(src, isHtml) {
    let out = '';
    let i = 0;
    const n = src.length;
    while (i < n) {
        if (isHtml && src.startsWith('<!--', i)) {
            const end = src.indexOf('-->', i);
            const stop = end === -1 ? n : end + 3;
            out += ' '.repeat(stop - i); i = stop; continue;
        }
        const two = src.slice(i, i + 2);
        if (!isHtml && two === '//') {
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
            let j = i + 1;                       // keep the literal TEXT (specifiers live inside)
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

function walk(dir, out = [], exts = ['.js']) {
    let entries;
    try { entries = fs.readdirSync(dir, { withFileTypes: true }); } catch (_) { return out; }
    for (const e of entries) {
        const full = path.join(dir, e.name);
        if (SKIP_DIR.test(full)) continue;
        if (e.isDirectory()) { walk(full, out, exts); continue; }
        if (!exts.includes(path.extname(e.name).toLowerCase())) continue;
        if (SKIP_FILES.has(e.name)) continue;
        out.push(full);
    }
    return out;
}

/** First-party JS sitting directly at the repo root (a page may load one of these). */
function rootJsFiles() {
    try {
        return fs.readdirSync(ROOT, { withFileTypes: true })
            .filter((e) => e.isFile() && path.extname(e.name).toLowerCase() === '.js')
            .map((e) => path.join(ROOT, e.name));
    } catch (_) { return []; }
}

/** Pages that can serve as entry points (directly at the repo root or under Public/). */
function htmlPages() {
    const pages = [];
    for (const dir of [ROOT, PUBLIC_DIR]) {
        let entries;
        try { entries = fs.readdirSync(dir, { withFileTypes: true }); } catch (_) { continue; }
        for (const e of entries) {
            if (!e.isFile() || path.extname(e.name).toLowerCase() !== '.html') continue;
            if (/architecture\.html$/i.test(e.name)) continue;   // generated report, not a page
            const full = path.join(dir, e.name);
            if (SKIP_DIR.test(full)) continue;
            pages.push(full);
        }
    }
    return pages;
}

// Specifier extraction. Every STATIC import/export carries `from '<spec>'`, so ONE anchor covers
// `import x from '…'`, `import { a, b } from '…'`, `export … from '…'` and `export * from '…'`.
// Dynamic `import('…')` and side-effect `import '…'` carry no `from`.
const FROM_RE = /\bfrom\b\s*['"]([^'"]+)['"]/g;
const DYNAMIC_IMPORT_RE = /\bimport\s*\(\s*['"]([^'"]+)['"]/g;
const SIDE_EFFECT_IMPORT_RE = /\bimport\s+['"]([^'"]+)['"]/g;
const SRC_RE = /\b(?:src|href)\s*=\s*['"]([^'"]+\.js)['"]/g;

function specifiersOf(file) {
    const isHtml = path.extname(file).toLowerCase() === '.html';
    let src;
    try { src = fs.readFileSync(file, 'utf8'); } catch (_) { return []; }
    const text = strip(src, isHtml);
    const found = new Map();
    const patterns = [
        [FROM_RE, 'module'], [DYNAMIC_IMPORT_RE, 'module'],
        [SIDE_EFFECT_IMPORT_RE, 'module'], [SRC_RE, 'asset'],
    ];
    for (const [re, kind] of patterns) {
        re.lastIndex = 0;
        let m;
        while ((m = re.exec(text)) !== null) if (!found.has(m[1])) found.set(m[1], kind);
    }
    return [...found].map(([spec, kind]) => ({ spec, kind }));
}

/**
 * Resolve a specifier to a real file, or say why it cannot be.
 *
 * A MODULE specifier is resolved by the ESM loader relative to the importing MODULE (or from the
 * web root when it starts with `/`); a bare specifier is an importmap/vendor name, never a path.
 * An ASSET (`<script src>` / `el.src = '…'`) is resolved by the BROWSER relative to the DOCUMENT,
 * so `js/devsim.js` inside `Public/js/app_bridge.js` means `Public/js/devsim.js`, not
 * `Public/js/js/devsim.js` — getting that wrong reports a live module as unreachable.
 */
function resolveSpec(fromFile, spec, kind) {
    if (/^[a-z][a-z0-9+.-]*:/i.test(spec) && !spec.startsWith('/')) return { external: true }; // http:, data:, blob:
    const pageRelative = kind === 'asset';
    if (!pageRelative && !spec.startsWith('.') && !spec.startsWith('/')) return { external: true }; // bare → importmap/vendor
    const bases = [];
    if (spec.startsWith('/')) bases.push(path.join(PUBLIC_DIR, spec.replace(/^\/+/, '')));
    else if (pageRelative) {
        bases.push(path.join(PUBLIC_DIR, spec), path.resolve(path.dirname(fromFile), spec));
    } else bases.push(path.resolve(path.dirname(fromFile), spec));

    for (const base of bases) {
        const candidates = [base];
        if (!path.extname(base)) candidates.push(base + '.js', path.join(base, 'index.js'));
        for (const c of candidates) {
            if (fs.existsSync(c) && fs.statSync(c).isFile()) {
                if (SKIP_DIR.test(c)) return { external: true };                              // vendor etc.
                return { file: c };
            }
        }
    }
    // An asset URL that matches no file in this repo is not a first-party module; only a MODULE
    // specifier that resolves nowhere is a genuine broken import.
    if (pageRelative) return { external: true };
    return { missing: bases[0] };
}

const rel = (f) => path.relative(ROOT, f).split(path.sep).join('/');

/**
 * Negative control. These are the exact parser facts that were WRONG on the first run of this
 * gate, each of which silently reported live modules as unreachable: a static `import { X } from
 * '…'` must be seen (only side-effect/dynamic imports carry no `from`), a commented-out import
 * must NOT be seen, and a classic `el.src = 'js/x.js'` must resolve against the DOCUMENT.
 */
function selftest() {
    const os = require('os');
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'reach-'));
    const checks = [];
    try {
        const staticFile = path.join(dir, 'static.js');
        fs.writeFileSync(staticFile, "import { A } from './missing.js';\n");
        const staticSpecs = specifiersOf(staticFile).map((s) => s.spec);
        checks.push(['static `from` specifier seen', staticSpecs.includes('./missing.js')]);
        checks.push(['its resolution is MISSING', !!resolveSpec(staticFile, './missing.js', 'module').missing]);

        const commented = path.join(dir, 'commented.js');
        fs.writeFileSync(commented, "// import './gone.js';\n/* import './gone2.js'; */\nconst x = 1;\n");
        checks.push(['commented-out import ignored', specifiersOf(commented).length === 0]);

        const sideEffect = path.join(dir, 'side.js');
        fs.writeFileSync(sideEffect, "import './a.js';\nconst go = () => import('./b.js');\n");
        const sideSpecs = specifiersOf(sideEffect).map((s) => s.spec);
        checks.push(['side-effect + dynamic imports seen', sideSpecs.includes('./a.js') && sideSpecs.includes('./b.js')]);

        const asset = path.join(PUBLIC_DIR, 'js', 'app_bridge.js');
        const r = resolveSpec(asset, 'js/devsim.js', 'asset');
        checks.push(['asset src resolves against the DOCUMENT (Public/js/devsim.js)',
            !!r.file && rel(r.file) === 'Public/js/devsim.js']);
    } finally {
        fs.rmSync(dir, { recursive: true, force: true });
    }
    const ok = checks.every(([, pass]) => pass);
    for (const [label, pass] of checks) console.log(`   ${pass ? 'ok  ' : 'FAIL'}  ${label}`);
    console.log(`[REACHABILITY SELFTEST] ok=${ok}`);
    process.exitCode = ok ? 0 : 1;
}

function main() {
    const asJson = process.argv.includes('--json');
    const inventory = new Set(walk(PUBLIC_DIR).concat(rootJsFiles()));
    const pages = htmlPages();
    // Canary: a walk that found almost nothing would make every module look reachable.
    const canary = inventory.size < 100
        ? [`the module walk found only ${inventory.size} first-party file(s) — check the SKIP rules; the gate is measuring nothing`]
        : [];

    // Page-declared entry points: a page may load a first-party module directly.
    const roots = [...ROOTS];
    for (const page of pages) {
        for (const { spec, kind } of specifiersOf(page)) {
            const r = resolveSpec(page, spec, kind);
            if (r.file && inventory.has(r.file) && !roots.includes(r.file)) roots.push(r.file);
        }
    }

    const seen = new Set();
    const broken = [];
    const queue = [...roots];
    while (queue.length) {
        const file = queue.pop();
        if (seen.has(file)) continue;
        seen.add(file);
        for (const { spec, kind } of specifiersOf(file)) {
            const r = resolveSpec(file, spec, kind);
            if (r.external) continue;
            if (r.missing) {
                broken.push({ from: rel(file), spec, tried: rel(r.missing) });
                continue;
            }
            if (inventory.has(r.file)) queue.push(r.file);
        }
    }

    const unreachable = [...inventory].filter((f) => !seen.has(f)).sort();
    const unexpected = unreachable.filter((f) => !(rel(f) in BASELINE));
    const stale = Object.keys(BASELINE).filter((key) => {
        const full = path.join(ROOT, key.split('/').join(path.sep));
        return !inventory.has(full) || seen.has(full);
    });
    const failures = broken.length + unexpected.length + stale.length + canary.length;
    const reached = [...seen].filter((f) => inventory.has(f)).length;

    if (asJson) {
        console.log(JSON.stringify({
            roots: roots.map(rel),
            pages: pages.map(rel),
            firstPartyModules: inventory.size,
            reachable: reached,
            // The closure itself, so a downstream measurement can ask "is this module
            // COMPOSED?" without re-deriving the graph (one owner per question).
            reachableFiles: [...seen].filter((f) => inventory.has(f)).map(rel).sort(),
            broken,
            unreachable: unreachable.map(rel),
            baselined: unreachable.filter((f) => rel(f) in BASELINE)
                .map((f) => ({ file: rel(f), owner: BASELINE[rel(f)] })),
            unexpected: unexpected.map(rel),
            staleBaseline: stale,
            canary,
            failures,
        }, null, 2));
    } else {
        console.log(`[REACHABILITY] first-party modules          : ${inventory.size}`);
        console.log(`  composition roots        : ${ROOTS.length} declared + ${roots.length - ROOTS.length} page-declared`);
        console.log(`  pages scanned            : ${pages.length}`);
        console.log(`  REACHABLE (closure)      : ${reached}`);
        console.log(`  unreachable              : ${unreachable.length} (${unexpected.length} NOT baselined)`);
        if (broken.length) {
            console.log(`\n  BROKEN IMPORTS: ${broken.length} — the target file does not exist, so nothing loads.`);
            for (const b of broken) console.log(`   - ${b.from} → '${b.spec}'  (tried ${b.tried})`);
        }
        if (unreachable.length) {
            console.log('\n  UNREACHABLE MODULES:');
            for (const f of unreachable) {
                const key = rel(f);
                console.log(`   · ${key}${key in BASELINE ? `\n       [baselined] ${BASELINE[key]}` : '  ← NOT BASELINED'}`);
            }
        }
        if (stale.length) {
            console.log(`\n  STALE BASELINE ENTRIES: ${stale.length} — now reachable or gone; remove them.`);
            for (const s of stale) console.log(`   - ${s}`);
        }
        for (const c of canary) console.log(`\n  CANARY: ${c}`);
        console.log(failures
            ? `\n  RESULT: FAIL (${failures} finding(s))`
            : '\n  RESULT: PASS — every first-party module is reachable from a composition root.');
    }
    process.exitCode = failures ? 1 : 0;
}

if (process.argv.includes('--selftest')) selftest();
else main();
