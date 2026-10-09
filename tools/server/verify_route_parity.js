#!/usr/bin/env node
/**
 * verify_route_parity.js — ONE gate for the routed-capability asymmetry.
 *
 * WHY THIS EXISTS
 * ---------------
 * `AI-Brain/RAG/17_aspects_flow.md` derived the route table as its aspect spine and measured
 * something no single aspect row could hide: of the routes this application serves, a large number
 * are registered in `server_main.go` and NOT in `console_server.go`. Nothing in the tree measured
 * it, so it could drift silently in either direction — a route added to one server and forgotten in
 * the other, or a route that quietly APPEARS in both and leaves a stale exemption behind.
 *
 * THE SUBJECT IS DERIVED, NEVER LISTED
 * ------------------------------------
 * Both route sets are parsed from source on every run. The baseline is the ledger of EXEMPTIONS
 * (routes knowingly served by one server only, each with a reason), and `--write` GENERATES it from
 * the tree — so it can never be a hand-typed guess, and it must SHRINK as parity lands.
 *
 * IT FAILS CLOSED
 * ---------------
 *  1. a route in `server_main.go` and not in `console_server.go`, with no baseline entry  -> FAIL
 *  2. a STALE baseline entry (the route is now in BOTH servers)                            -> FAIL
 *  3. a baseline entry naming a route the parser cannot see at all                          -> FAIL
 *  4. ZERO routes parsed from either file (a parser that finds nothing must never read as
 *     "clean" — that is how a blind gate reports a healthy repository for ever)              -> FAIL
 *
 * USAGE
 *   node tools/server/verify_route_parity.js            # verify (exit 1 on any failure)
 *   node tools/server/verify_route_parity.js --json     # machine-readable report
 *   node tools/server/verify_route_parity.js --write    # regenerate the baseline from the tree
 *   node tools/server/verify_route_parity.js --selftest # negative control: prove it can FAIL
 */
'use strict';
const fs = require('fs');
const path = require('path');

const REPO = path.resolve(__dirname, '..', '..');
const MAIN = path.join(REPO, 'server_main.go');
const CONSOLE = path.join(REPO, 'console_server.go');
const BASELINE = path.join(REPO, 'tools', 'server', 'route_parity_baseline.json');

/** routesFrom parses every `HandleFunc("<path>"` registration in one file. It reads the PATH only,
 *  never the handler: the question here is which SERVER answers a path, not who handles it. */
function routesFrom(file) {
  const text = fs.readFileSync(file, 'utf8');
  const out = new Set();
  const re = /HandleFunc\(\s*"([^"]+)"/g; // the handler may be wrapped — withRateLimit(l.handleX, …) — so stop at the path
  let m;
  while ((m = re.exec(text)) !== null) out.add(m[1]);
  return out;
}

function loadBaseline() {
  if (!fs.existsSync(BASELINE)) return null;
  try {
    const raw = JSON.parse(fs.readFileSync(BASELINE, 'utf8'));
    if (!raw || typeof raw !== 'object' || !raw.exempt || typeof raw.exempt !== 'object') return { broken: 'no `exempt` object' };
    return raw;
  } catch (e) {
    return { broken: String(e) };
  }
}

/** runSelftest is the NEGATIVE CONTROL. A gate whose control cannot fail is a gate that passes
 *  silently, and this repository has recorded that lesson more than once. */
function runSelftest() {
  const os = require('os');
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'parity-'));
  const a = path.join(tmp, 'a.go');
  const b = path.join(tmp, 'b.go');
  const write = (p, body) => fs.writeFileSync(p, body, 'utf8');
  let pass = 0, fail = 0;
  const check = (label, got, want) => {
    if (JSON.stringify(got) === JSON.stringify(want)) { pass++; console.log(`  ok   ${label}`); }
    else { fail++; console.log(`  FAIL ${label}: got ${JSON.stringify(got)} want ${JSON.stringify(want)}`); }
  };

  // MUST report: a route present in one file and not the other.
  write(a, 'mux.HandleFunc("/api/x", h)\n');
  write(b, '');
  check('parser sees a route', routesFrom(a).has('/api/x'), true);
  check('parser sees nothing in an empty file', routesFrom(b).size, 0);

  // MUST report: this is the form the tree actually uses (a wrapped handler), and a parser that
  // only accepted a bare identifier would silently find NOTHING.
  write(a, 'mux.HandleFunc("/api/wrapped", withRateLimit(l.handleX, "economy-tight"))\n');
  check('wrapped handler form parses', routesFrom(a).has('/api/wrapped'), true);

  // MUST NOT report: identical sets.
  write(a, 'mux.HandleFunc("/api/same", h)\n');
  write(b, 'mux.HandleFunc("/api/same", h)\n');
  check('identical sets have no gap', [...routesFrom(a)].filter(r => !routesFrom(b).has(r)).length, 0);

  // The parser's OWN stated limit, pinned so it cannot be mistaken for a strength: a path inside a
  // COMMENT is matched by this raw scan. Recorded as a known false-positive class, not hidden.
  write(a, '// mux.HandleFunc("/api/commented", h)\n');
  check('commented path IS matched (known limit)', routesFrom(a).has('/api/commented'), true);

  fs.rmSync(tmp, { recursive: true, force: true });
  console.log(`\nSELFTEST: ${pass} passed, ${fail} failed`);
  process.exit(fail === 0 ? 0 : 1);
}

function main() {
  const argv = process.argv.slice(2);
  const asJson = argv.includes('--json');

  if (argv.includes('--selftest')) runSelftest();

  const missing = [];
  if (!fs.existsSync(MAIN)) missing.push(MAIN);
  if (!fs.existsSync(CONSOLE)) missing.push(CONSOLE);
  if (missing.length) {
    console.error('FATAL: missing ' + missing.join(', '));
    process.exit(1);
  }

  const mainRoutes = routesFrom(MAIN);
  const consoleRoutes = routesFrom(CONSOLE);

  // FAIL CLOSED (4): a parser that finds nothing must never read as clean.
  if (mainRoutes.size === 0 || consoleRoutes.size === 0) {
    console.error(`FATAL: parsed ${mainRoutes.size} routes from server_main.go and ${consoleRoutes.size} from console_server.go.`);
    console.error('A gate that parses nothing reports a healthy repository for ever. Refusing to pass.');
    process.exit(1);
  }

  const onlyMain = [...mainRoutes].filter(r => !consoleRoutes.has(r)).sort();
  const onlyConsole = [...consoleRoutes].filter(r => !mainRoutes.has(r)).sort();

  if (argv.includes('--write')) {
    const exempt = {};
    for (const r of onlyMain) {
      exempt[r] = 'served by server_main.go only — the console target does not register this path';
    }
    const doc = {
      generated_by: 'tools/server/verify_route_parity.js --write',
      rule: 'Each key is a path registered in server_main.go and NOT in console_server.go; the value is the reason it is exempt. This file must SHRINK as parity lands — a key that becomes registered in BOTH servers is STALE and FAILS the gate.',
      counts: { server_main: mainRoutes.size, console_server: consoleRoutes.size, exempt: onlyMain.length, console_only: onlyConsole.length },
      exempt,
    };
    fs.writeFileSync(BASELINE, JSON.stringify(doc, null, 2) + '\n', 'utf8');
    console.log(`WROTE ${BASELINE}`);
    console.log(`  server_main.go    : ${mainRoutes.size} routes`);
    console.log(`  console_server.go : ${consoleRoutes.size} routes`);
    console.log(`  exempt (one server only): ${onlyMain.length}`);
    process.exit(0);
  }

  const base = loadBaseline();
  if (base && base.broken) { console.error(`FATAL: baseline is not usable: ${base.broken}`); process.exit(1); }
  if (!base) {
    console.error(`FATAL: no baseline at ${BASELINE}`);
    console.error('Run `node tools/server/verify_route_parity.js --write` to DERIVE it from the tree.');
    process.exit(1);
  }
  const exempt = base.exempt;

  const unbaselined = onlyMain.filter(r => !(r in exempt));                    // FAIL (1) a new gap
  const stale = Object.keys(exempt).filter(r => !onlyMain.includes(r));        // FAIL (2) a closed gap
  const invisible = Object.keys(exempt).filter(r => !mainRoutes.has(r) && !consoleRoutes.has(r)); // FAIL (3) a ghost

  const report = {
    server_main: mainRoutes.size,
    console_server: consoleRoutes.size,
    only_in_server_main: onlyMain.length,
    only_in_console: onlyConsole.length,
    baselined: Object.keys(exempt).length,
    unbaselined, stale, invisible,
    console_only_sample: onlyConsole.slice(0, 10),
  };

  if (asJson) {
    console.log(JSON.stringify(report, null, 2));
  } else {
    console.log('ROUTE PARITY — server_main.go vs console_server.go\n');
    console.log(`  server_main.go    : ${report.server_main} routes`);
    console.log(`  console_server.go : ${report.console_server} routes`);
    console.log(`  one server only   : server_main ${report.only_in_server_main} · console ${report.only_in_console}`);
    console.log(`  baselined exempt  : ${report.baselined}\n`);
    if (unbaselined.length) {
      console.log(`FAIL — ${unbaselined.length} route(s) in server_main.go only, with NO exemption:`);
      for (const r of unbaselined.slice(0, 40)) console.log(`    ${r}`);
      if (unbaselined.length > 40) console.log(`    … and ${unbaselined.length - 40} more`);
      console.log('  Either register it in console_server.go, or add it to the baseline WITH a reason.\n');
    }
    if (stale.length) {
      console.log(`FAIL — ${stale.length} STALE baseline entr(y/ies) (now registered in BOTH servers):`);
      for (const r of stale.slice(0, 40)) console.log(`    ${r}`);
      console.log('  Delete each one: the baseline must SHRINK as parity lands.\n');
    }
    if (invisible.length) {
      console.log(`FAIL — ${invisible.length} baseline entr(y/ies) name a route the parser cannot see at all:`);
      for (const r of invisible.slice(0, 40)) console.log(`    ${r}`);
      console.log('  The route was renamed or removed; correct the baseline.\n');
    }
  }

  if (unbaselined.length || stale.length || invisible.length) process.exit(1);
  if (!asJson) console.log('PASS — every route served by server_main.go alone is baselined with a reason, and no baseline entry is stale.');
  process.exit(0);
}

main();
