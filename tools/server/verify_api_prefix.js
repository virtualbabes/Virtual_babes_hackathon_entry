#!/usr/bin/env node
'use strict';
/**
 * verify_api_prefix.js — THE URL-CONSTRUCTION GATE.
 *
 * WHY THIS EXISTS. Every client module builds its requests the same way:
 *
 *     var API_BASE = '/api';
 *     async function api(path, opts) { const resp = await fetch(API_BASE + path, opts); … }
 *
 * so the CONVENTION is that a caller passes a path WITHOUT the prefix: `api('/pets')`.
 * A caller that passes an already-prefixed path — `api('/api/pets')` — produces
 * `/api/api/pets`, which the server answers **404**. The module then renders
 * "data unavailable", and nothing upstream can tell that apart from a genuinely empty
 * backend: a wrong URL raises no error, no console message, and no gate catches it.
 * (The handler gate resolves NAMES in markup; the entry probe checks overlay visibility
 * and console/page errors. Neither reads a URL.)
 *
 * Measured when this gate was written: **26 files carried it, 22 of them COMPOSED** — the
 * Underworld, Governance, the Orphan Cleaner, Religion Governance, the Theme Dashboard,
 * Advertising, Entity Shares, Launchpad, the Stat Overlay, Rivalry, Children Bots and the
 * Creator Economy were all reaching `/api/api/…` and answering nothing. Proven against the
 * live server: `/api/underworld/heists` → 200, `/api/api/underworld/heists` → 404.
 *
 * WHAT IT CHECKS (fail closed)
 *   A module that PREPENDS API_BASE to its `path` parameter must either
 *     (a) not be called with an `/api/…` path, or
 *     (b) neutralise the duplicate with the `path.startsWith('/api/')` guard
 *         (the pattern `world_dashboard.js` already used).
 *
 * Usage:
 *   node tools/server/verify_api_prefix.js              detect (exit 1 on a finding)
 *   node tools/server/verify_api_prefix.js --fix        repair every finding
 *   node tools/server/verify_api_prefix.js --selftest   prove it can report a failure
 */
const fs = require('fs');
const path = require('path');

const JS = path.resolve(__dirname, '..', '..', 'Public', 'js');
const APPLY = process.argv.includes('--fix');

function scan(dir) {
    const out = [];
    for (const f of fs.readdirSync(dir).filter((n) => n.endsWith('.js')).sort()) {
        const src = fs.readFileSync(path.join(dir, f), 'utf8');
        const helper = src.match(/^([ \t]*)async function api\(path, opts\) \{$/m);
        if (!helper) continue;
        if (!src.includes('API_BASE + path')) continue;          // does not prepend
        if (src.includes('path.startsWith(')) continue;          // already guarded
        const bad = (src.match(/\bapi\(\s*['"`]\/api\//g) || []).length;
        const good = (src.match(/\bapi\(\s*['"`]\/(?!api\/)/g) || []).length;
        if (bad === 0) continue;                                 // callers are correct
        out.push({ f, bad, good, src, helper });
    }
    return out;
}

function repair(finding) {
    const indent = finding.helper[1] + '    ';
    const decl = `${indent}const url = path.startsWith('/api/') ? path : API_BASE + path;`;
    let src = finding.src.replace(/API_BASE \+ path/g, 'url');
    src = src.replace(finding.helper[0], `${finding.helper[0]}\n${decl}`);
    fs.writeFileSync(path.join(JS, finding.f), src);
}

function main() {
    const findings = scan(JS);
    if (!findings.length) {
        console.log('[API-PREFIX] every module that prepends API_BASE is guarded against a duplicate prefix.');
        console.log('  RESULT: PASS');
        process.exitCode = 0;
        return;
    }
    console.log(`[API-PREFIX] ${findings.length} module(s) build a request the server cannot route:`);
    for (const d of findings) {
        console.log(`   · Public/js/${d.f}  — ${d.bad} call(s) pass '/api/…' into a helper that already prepends it → /api/api/… (404)`);
        if (APPLY) { repair(d); console.log('       repaired (path.startsWith guard)'); }
    }
    console.log(APPLY ? '\n  RESULT: FIXED' : '\n  RESULT: FAIL — run with --fix');
    process.exitCode = APPLY ? 0 : 1;
}

function selftest() {
    const os = require('os');
    const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'apifix-'));
    fs.writeFileSync(path.join(tmp, 'broken.js'),
        "var API_BASE = '/api';\n" +
        "async function api(path, opts) {\n" +
        "    const resp = await fetch(API_BASE + path, opts);\n" +
        "    return resp.json();\n" +
        "}\n" +
        "async function go() { return api('/api/pets'); }\n");
    fs.writeFileSync(path.join(tmp, 'fine.js'),
        "var API_BASE = '/api';\n" +
        "async function api(path, opts) {\n" +
        "    const url = path.startsWith('/api/') ? path : API_BASE + path;\n" +
        "    const resp = await fetch(url, opts);\n" +
        "    return resp.json();\n" +
        "}\n" +
        "async function go() { return api('/api/pets'); }\n");
    const found = scan(tmp);
    const ok = found.length === 1 && found[0].f === 'broken.js' && found[0].bad === 1;
    console.log(ok
        ? '[API-PREFIX] selftest PASS — the detector reports exactly the broken fixture and ignores the guarded one.'
        : `[API-PREFIX] selftest FAIL — found ${JSON.stringify(found.map((x) => x.f))}`);
    process.exitCode = ok ? 0 : 1;
}

if (process.argv.includes('--selftest')) selftest();
else main();
