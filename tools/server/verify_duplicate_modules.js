#!/usr/bin/env node
/**
 * verify_duplicate_modules.js — THE ONE-IMPLEMENTATION GATE.
 *
 * WHY THIS EXISTS. Two files with the SAME BASENAME and DIFFERENT CONTENT are one owner too
 * many, and the failure is silent: a relative specifier decides which one a given importer gets,
 * so ONE of them is what actually runs and the other is a decoy that still looks authoritative.
 *
 * The measured case: `Public/collective-intelligence.js` (15,751 B, the real personality
 * registry + `generatePlaystyleTaunt`) and `Public/js/collective-intelligence.js` (729 B, an
 * EMPTY registry with `personalities: {}` and no taunt method). Nothing imported the small one —
 * because `app.js` resolved `./collective-intelligence.js` from `Public/` and
 * `Public/js/economy.js` resolved `../collective-intelligence.js` UP a level. So the empty copy
 * was a trap waiting for the first in-directory import, which would have yielded an NPC layer
 * that still "worked" while knowing nothing.
 *
 * THE RULE (and nothing wider)
 *   For every basename with more than one file under Public/:
 *     · exactly ONE member may be an IMPLEMENTATION (it defines something of its own);
 *     · every other member must be an ALIAS — a re-export that defines no state
 *       (`export { x } from '../x.js'`), which is a supported, documented forwarding pattern.
 *   FAIL when a group has TWO OR MORE implementations (a decoy can win resolution), or ZERO
 *   (a chain of aliases with nothing behind it).
 *
 * Scope: `Public/**.js` — the served tree, where a duplicate changes which file a URL or
 * specifier resolves to. Vendor, node_modules and generated output are skipped.
 *
 * Usage: node tools/server/verify_duplicate_modules.js [--json|--selftest]
 */

'use strict';

const fs = require('fs');
const path = require('path');

const ROOT = path.resolve(__dirname, '..', '..');
const PUBLIC_DIR = path.join(ROOT, 'Public');
const SKIP_DIR = /([\\/])(vendor|node_modules|prototypes|Assets|Generated|devdata|\.git)([\\/])/i;
const SKIP_FILES = new Set(['wasm_exec.js']);

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

/** Comments removed; literals kept (a specifier inside one is not an own definition). */
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

/** Does this file define anything of its own (vs. only forwarding)? */
function ownDefinitions(code) {
    const signals = [
        /\b(?:function|class)\s+[A-Za-z_$][\w$]*/g,
        /\b(?:const|let|var)\s+[A-Za-z_$][\w$]*/g,
        /\b(?:window|globalThis)\s*\.\s*[A-Za-z_$][\w$]*\s*=[^=]/g,
        /\b(?:window|globalThis)\s*\[\s*['"][A-Za-z_$][\w$]*['"]\s*\]\s*=[^=]/g,
    ];
    let hits = 0;
    for (const re of signals) {
        re.lastIndex = 0;
        const m = code.match(re);
        if (m) hits += m.length;
    }
    return hits;
}

/** Re-export edges: `export … from '…'`. */
const REEXPORT_RE = /\bexport\s+(?:\*|\{[^}]*\})\s*from\s*['"]([^'"]+)['"]/g;

function classify(file) {
    let src;
    try { src = fs.readFileSync(file, 'utf8'); } catch (_) { return null; }
    const code = strip(src);
    REEXPORT_RE.lastIndex = 0;
    const forwards = [...code.matchAll(REEXPORT_RE)].map((m) => m[1]);
    const own = ownDefinitions(code);
    return {
        file,
        rel: path.relative(ROOT, file).split(path.sep).join('/'),
        bytes: Buffer.byteLength(src, 'utf8'),
        own,
        forwards,
        kind: (forwards.length > 0 && own === 0) ? 'alias' : 'implementation',
    };
}

function groups() {
    const byName = new Map();
    for (const file of walk(PUBLIC_DIR)) {
        const base = path.basename(file).toLowerCase();
        if (!byName.has(base)) byName.set(base, []);
        byName.get(base).push(file);
    }
    const out = [];
    for (const [base, files] of byName) {
        if (files.length < 2) continue;
        out.push({ base, members: files.map(classify).filter(Boolean).sort((a, b) => b.bytes - a.bytes) });
    }
    return out.sort((a, b) => a.base.localeCompare(b.base));
}

/** Negative control: an alias must classify as a forwarder, and a STUB must classify as an implementation. */
function selftest() {
    const os = require('os');
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'dup-modules-'));
    try {
        const impl = path.join(dir, 'thing.js');
        const alias = path.join(dir, 'thing-alias.js');
        const stub = path.join(dir, 'thing-stub.js');
        fs.writeFileSync(impl, 'export const thing = { data: 1 };\nexport function use() { return 1; }\n');
        fs.writeFileSync(alias, "export { thing } from './thing.js';\n");
        fs.writeFileSync(stub, 'export const thing = { data: {} };\n');
        const a = classify(alias), i = classify(impl), s = classify(stub);
        const ok = a.kind === 'alias' && a.own === 0 && i.kind === 'implementation' && s.kind === 'implementation';
        console.log(`[DUPLICATES SELFTEST] alias=${a.kind}(own=${a.own}) impl=${i.kind} stub=${s.kind} ok=${ok}`);
        process.exitCode = ok ? 0 : 1;
    } finally {
        fs.rmSync(dir, { recursive: true, force: true });
    }
}

function main() {
    const asJson = process.argv.includes('--json');
    const scanned = walk(PUBLIC_DIR).length;
    const gs = groups();

    // Canary: a walk that found almost nothing would make every group look clean.
    const findings = [];
    if (scanned < 100) {
        findings.push({ base: '(canary)', reason: `the file walk found only ${scanned} module(s) under Public/ — check the SKIP rules; the gate is measuring nothing` });
    }

    const report = [];
    for (const g of gs) {
        const impls = g.members.filter((m) => m.kind === 'implementation');
        const aliases = g.members.filter((m) => m.kind === 'alias');
        let verdict = 'PASS';
        let reason = `one implementation + ${aliases.length} forwarding alias(es)`;
        if (impls.length === 0) {
            verdict = 'FAIL';
            reason = 'NO member defines anything — a chain of aliases with nothing behind it';
        } else if (impls.length > 1) {
            verdict = 'FAIL';
            reason = `${impls.length} COMPETING IMPLEMENTATIONS — which one runs depends on the specifier`;
        }
        if (verdict === 'FAIL') findings.push({ base: g.base, reason });
        report.push({
            base: g.base, verdict, reason,
            members: g.members.map((m) => ({ file: m.rel, bytes: m.bytes, kind: m.kind, forwards: m.forwards })),
        });
    }

    if (asJson) {
        console.log(JSON.stringify({
            filesScanned: scanned,
            duplicateBasenames: gs.length,
            groups: report,
            findings,
            failures: findings.length,
        }, null, 2));
    } else {
        const aliasCount = report.reduce((n, r) => n + r.members.filter((m) => m.kind === 'alias').length, 0);
        const implCount = report.reduce((n, r) => n + r.members.filter((m) => m.kind === 'implementation').length, 0);
        console.log(`[DUPLICATES] modules scanned           : ${scanned}`);
        console.log(`  same-basename groups              : ${gs.length}`);
        console.log(`  members: implementations / aliases: ${implCount} / ${aliasCount}`);
        for (const r of report) {
            console.log(`\n  ${r.verdict}  ${r.base}  — ${r.reason}`);
            for (const m of r.members) {
                console.log(`     · ${m.file}  (${m.bytes} B, ${m.kind}${m.forwards.length ? ` → ${m.forwards.join(', ')}` : ''})`);
            }
        }
        console.log(findings.length
            ? `\n  RESULT: FAIL (${findings.length} finding(s))`
            : '\n  RESULT: PASS — every duplicated basename has exactly one implementation.');
    }
    process.exitCode = findings.length ? 1 : 0;
}

if (process.argv.includes('--selftest')) selftest();
else main();
