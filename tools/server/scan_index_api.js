// ============================================================================
// scan_index_api.js — index.html API/handler inventory (dev utility)
// ----------------------------------------------------------------------------
// RETIRED INTO tools/ (2026-09-15 e). This is the tool that lived at
// `Public/_load_spectate_c.js` — INSIDE the directory the server serves with
// `http.FileServer(http.Dir("./Public"))` (server_main.go), so a developer
// scratch script was published over HTTP while being loadable by nothing.
// It is NOT app code and NOT part of the composition graph.
//
// What it does: reads the shell, lists every `window.*` name the markup calls,
// every element id, every `src` and the inline-script count — the inventory
// used when reconciling "which handler does this markup expect?".
//
// Usage:
//   node tools/server/scan_index_api.js               # print to stdout
//   node tools/server/scan_index_api.js --out api.txt # also write a file
//
// It resolves `Public/index.html` from the REPOSITORY ROOT, so it runs from
// any working directory (the retired copy only worked when cwd was `Public/`).
// ============================================================================

'use strict';

const fs = require('fs');
const path = require('path');

const ROOT = path.resolve(__dirname, '..', '..');
const INDEX = path.join(ROOT, 'Public', 'index.html');

function main() {
    if (!fs.existsSync(INDEX)) {
        console.error(`[scan_index_api] shell not found: ${INDEX}`);
        process.exit(1);
    }
    const html = fs.readFileSync(INDEX, 'utf8');
    const wins = [...html.matchAll(/window\.([A-Za-z0-9_]+)\s*(?==|\()/g)].map((m) => m[1]);
    const ids = [...html.matchAll(/id="([a-zA-Z0-9_-]+)"/g)].map((m) => m[1]);
    const scriptSrcs = [...html.matchAll(/<script[^>]+src="([^"]+)"/g)].map((m) => m[1]);
    const inlineScripts = [...html.matchAll(/<script>([^<]{5,})<\/script>/g)].map((m) => m[1]);

    const lines = [
        '=== window.* HANDLERS ===',
        ...wins.map((w) => '  window.' + w),
        '=== TOTAL window handlers: ' + wins.length,
        '=== notable ids ===',
        ...ids.map((i) => '  ' + i).slice(0, 200),
        '=== script srcs ===',
        ...scriptSrcs.map((s) => '  ' + s),
        '=== inline scripts ===',
        String(inlineScripts.length),
        '',
    ];
    const report = lines.join('\n');
    process.stdout.write(report + '\n');

    const outFlag = process.argv.indexOf('--out');
    if (outFlag >= 0 && process.argv[outFlag + 1]) {
        const out = path.resolve(ROOT, process.argv[outFlag + 1]);
        fs.writeFileSync(out, report);
        process.stderr.write(`[scan_index_api] wrote ${out}\n`);
    }
}

main();
