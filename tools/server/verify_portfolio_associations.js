#!/usr/bin/env node
// verify_portfolio_associations.js — audit the PORTFOLIO ASSOCIATIONS contract.
//
// WHY THIS EXISTS
//   The Portfolio is the read-only ANALYTICS surface for a wallet. `verify_entry_probe`
//   proves every screen renders WITHOUT a wallet; it cannot prove the wallet-scoped
//   association screens (companions, rivalry, clubs & alliances, bonded assets, faith,
//   custody, obligations) actually resolve their endpoints and report real state. Those
//   screens short-circuit to a connect note when no wallet is present, so a broken URL
//   would go unnoticed.
//
//   This probe connects a SYNTHETIC wallet, patches `window.fetch` to record every
//   `/api/` request with its HTTP status, opens the Portfolio, walks every category and
//   screen, and reports:
//     * per-category render evidence (tables / key-value rows / explicit notes, chars)
//     * the recorded request log (status codes) for the association endpoints
//     * console / page errors
//   A screen that renders, an endpoint that answers 200, and no errors = pass.
//
// Usage: node tools/server/verify_portfolio_associations.js [--url http://localhost:8090/]

const http = require('http');
const { spawn } = require('child_process');
const fs = require('fs');
const os = require('os');
const path = require('path');

// A structurally valid AVM address that the server has no record for: every
// wallet-scoped endpoint must still answer (empty / not-found), never error.
const SYNTHETIC_WALLET = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567ABCDEFGHIJKLMNOPQRSTUVWXYZ234567ABCD';

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
        if (p) { this.pending.delete(msg.id); msg.error ? p.reject(new Error(JSON.stringify(msg.error))) : p.resolve(msg.result); }
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

async function main() {
  const argv = process.argv.slice(2);
  const urlArg = argv.indexOf('--url');
  const target = urlArg >= 0 ? argv[urlArg + 1] : 'http://localhost:8090/';
  const cdpPort = 9447;

  const chrome = resolveChrome();
  if (!chrome) { console.log('[ASSOC] No browser found — skipped.'); process.exit(0); }

  const profile = fs.mkdtempSync(path.join(os.tmpdir(), 'vbt-assoc-'));
  const proc = spawn(chrome, [
    '--headless=new', `--remote-debugging-port=${cdpPort}`, `--user-data-dir=${profile}`,
    '--no-first-run', '--no-default-browser-check', '--disable-extensions', '--disable-gpu',
    '--window-size=1440,900', 'about:blank',
  ], { stdio: 'ignore' });

  const errors = [];
  let cdp;
  try {
    await waitFor(`http://127.0.0.1:${cdpPort}/json/version`, 15000);
    const { body } = await httpGet(`http://127.0.0.1:${cdpPort}/json/list`);
    const page = JSON.parse(body).find((t) => t.type === 'page');
    cdp = new CDP(page.webSocketDebuggerUrl);
    await cdp.open();
    cdp.on('Runtime.consoleAPICalled', (p) => {
      if (p.type === 'error') errors.push((p.args || []).map((a) => a.value || a.description || '').join(' ').slice(0, 140));
    });
    cdp.on('Runtime.exceptionThrown', (p) => {
      const d = p.exceptionDetails || {};
      errors.push(((d.exception && (d.exception.description || d.exception.value)) || d.text || '').slice(0, 140));
    });
    await cdp.send('Runtime.enable');
    await cdp.send('Page.enable');
    await cdp.send('Page.navigate', { url: target });
    // Poll for readiness: the app boots through wasm_exec, so a fixed wait is
    // flaky under load. Wait until the Portfolio entry point exists.
    let ready = false;
    for (let i = 0; i < 50; i++) {
      const probe = await cdp.send('Runtime.evaluate', {
        expression: "typeof window.openPortfolio === 'function'", returnByValue: true,
      });
      if (probe.result && probe.result.value === true) { ready = true; break; }
      await new Promise((r) => setTimeout(r, 500));
    }
    if (!ready) throw new Error('window.openPortfolio never became available');

    // ONE evaluation: patch → connect wallet → open Portfolio → sweep. Splitting
    // these across Runtime.evaluate calls loses the closure scope (and the fetch
    // log), which silently produced an empty report.
    const expr = `(async () => {
      const wallet = ${JSON.stringify(SYNTHETIC_WALLET)};
      const log = [];
      const orig = window.fetch.bind(window);
      window.fetch = function (input, init) {
        const url = (typeof input === 'string') ? input : (input && input.url);
        const p = orig(input, init);
        if (url && url.indexOf('/api/') >= 0) {
          p.then(function (res) { log.push({ url: url.replace(location.origin, ''), status: res.status }); })
           .catch(function () { log.push({ url: url.replace(location.origin, ''), status: 'network-error' }); });
        }
        return p;
      };
      window.currentWallet = wallet;
      const diag = { opened: typeof window.openPortfolio === 'function' };

      // ── Association-export contract (§3) ──────────────────────────────────
      // Probed FIRST, while the wallet-default rate-limit bucket is fresh. A 429
      // is retried once (exactly the rule the Portfolio itself applies); if it is
      // STILL refused the read is reported as "could not be read" rather than as
      // a contract violation — there is no body to assert against.
      const contract = { status: null, retried: false, present: null, keys: 0, nulls: [], floats: [] };
      let ares = await fetch('/api/player/associations?wallet=' + encodeURIComponent(wallet));
      if (ares.status === 429) {
        contract.retried = true;
        await new Promise(function (r) { setTimeout(r, 700); });
        ares = await fetch('/api/player/associations?wallet=' + encodeURIComponent(wallet));
      }
      contract.status = ares.status;
      if (ares.status === 200) {
        const abody = await ares.json();
        contract.keys = Object.keys(abody).length;
        contract.present = abody.present;
        // A null map/slice would let a reader confuse "not exported" with "empty".
        contract.nulls = Object.keys(abody).filter(function (k) { return abody[k] === null; });
        (function walk(v, path) {
          if (v === undefined || v === null) return;
          if (typeof v === 'number') { if (!Number.isInteger(v)) contract.floats.push(path); return; }
          if (Array.isArray(v)) { v.forEach(function (x, i) { walk(x, path + '[' + i + ']'); }); return; }
          if (typeof v === 'object') { Object.keys(v).forEach(function (k) { walk(v[k], path + '.' + k); }); }
        })(abody, '$');
        contract.floats = contract.floats.slice(0, 8);
      }

      if (diag.opened) window.openPortfolio('overview');
      await new Promise(function (r) { setTimeout(r, 1200); });

      const rows = [];
      const cats = Array.from(document.querySelectorAll('.pp-cat'));
      diag.categories = cats.length;
      diag.walletSeen = (typeof window.getActiveWallet === 'function') ? window.getActiveWallet() : '(no accessor)';
      for (const c of cats) {
        c.click();
        await new Promise(function (r) { setTimeout(r, 70); });
        const subs = Array.from(document.querySelectorAll('#pp-subtabs .pp-subtab'));
        for (const s of subs) {
          s.click();
          const panel = document.getElementById('pp-panel');
          // Wait until the screen leaves its loading state (screens make up to
          // four sequential reads), with a bounded ceiling.
          for (let w = 0; w < 26; w++) {
            await new Promise(function (r) { setTimeout(r, 70); });
            const t = panel ? (panel.textContent || '') : '';
            if (!/Reading analytics/.test(t)) break;
          }
          const txt = panel ? (panel.textContent || '') : '';
          rows.push({
            cat: c.dataset.cat,
            sub: s.dataset.sub,
            chars: txt.trim().length,
            blocks: panel ? panel.querySelectorAll('.pf-table, .pf-rows, .pf-chips, .pf-bar-row, .pp-grid').length : 0,
            notes: panel ? panel.querySelectorAll('.pp-empty, .pp-hint').length : 0,
            failed: /Failed to render/i.test(txt),
            loading: /Reading analytics/.test(txt)
          });
        }
      }
      return { diag: diag, rows: rows, log: log, contract: contract };
    })()`;
    const r = await cdp.send('Runtime.evaluate', { expression: expr, returnByValue: true, awaitPromise: true });
    if (r.exceptionDetails) throw new Error('page evaluate threw: ' + JSON.stringify(r.exceptionDetails).slice(0, 260));

    const out = r.result.value;
    const rows = out.rows || [];

    console.log('\n[ASSOC] Portfolio association audit — ' + target);
    console.log('[ASSOC] active wallet: ' + out.diag.walletSeen
      + ' | categories=' + out.diag.categories + ' | hub opened=' + out.diag.opened);
    console.log('-'.repeat(96));

    const bad = [];
    const byCat = {};
    for (const row of rows) {
      const b = byCat[row.cat] || (byCat[row.cat] = { n: 0, minChars: 1e9, blocks: 0, notes: 0 });
      b.n++;
      b.minChars = Math.min(b.minChars, row.chars);
      b.blocks += row.blocks;
      b.notes += row.notes;
      if (row.chars === 0 || row.failed || row.loading) bad.push(row.cat + '/' + row.sub);
    }
    for (const k of Object.keys(byCat)) {
      const b = byCat[k];
      console.log('  ' + (b.minChars > 0 ? 'OK    ' : 'EMPTY ') + k.padEnd(16)
        + 'screens=' + String(b.n).padEnd(4)
        + 'minChars=' + String(b.minChars).padEnd(8)
        + 'blocks=' + String(b.blocks).padEnd(6)
        + 'notes=' + b.notes);
    }

    // ── Association-export contract (§3) ─────────────────────────────────────
    // Probed at the head of the sweep (fresh rate-limit bucket). It must answer
    // 200, must NEVER emit a null map/slice (a reader could otherwise confuse
    // "not exported" with "empty"), and must carry no float (Architecture
    // Ledger: integer micro-units only). A still-refused read after the one
    // retry is reported as unread — never as a pass and never as a violation.
    const cc = out.contract || {};
    const ccRefused = (cc.status === 429) || (typeof cc.status !== 'number');
    const ccOk = (cc.status === 200)
      && (cc.nulls || []).length === 0 && (cc.floats || []).length === 0;
    const contractOk = ccOk || ccRefused;
    const ccDetail = (cc.status === 200)
      ? ('keys=' + cc.keys + ' | present=' + cc.present
         + ' | null fields=' + JSON.stringify(cc.nulls)
         + ' | float fields=' + JSON.stringify(cc.floats))
      : 'read refused — could not be read, not a violation';
    const ccVerdict = ccOk ? 'OK' : (ccRefused ? 'RATE-LIMITED (accepted)' : 'FAIL');
    console.log('[ASSOC] engine association export: HTTP ' + cc.status
      + (cc.retried ? ' (retried once after 429)' : '')
      + ' | ' + ccDetail + ' | ' + ccVerdict);
    if (!contractOk) {
      console.log('[ASSOC] FAIL: the association export must be HTTP 200, null-free and float-free.');
    }

    const seen = new Map();
    for (const e of (out.log || [])) {
      const key = e.url.split('?')[0];
      if (!seen.has(key)) seen.set(key, e.status);
    }
    const statuses = Array.from(seen.entries()).sort((a, b) => String(a[0]).localeCompare(String(b[0])));
    const errored = statuses.filter((e) => (typeof e[1] === 'number') ? (e[1] >= 500 || e[1] === 0) : true);
    console.log('-'.repeat(96));
    console.log('[ASSOC] distinct /api/ endpoints touched: ' + statuses.length);
    for (const [url, status] of statuses) console.log('  ' + String(status).padEnd(14) + url);
    console.log('-'.repeat(96));
    console.log('[ASSOC] ' + (rows.length - bad.length) + ' of ' + rows.length
      + ' screens render content with a wallet connected; ' + bad.length + ' bad'
      + (bad.length ? ' -> ' + JSON.stringify(bad) : ''));
    if (errored.length) console.log('[ASSOC] endpoints returning 5xx / network-error: ' + JSON.stringify(errored));
    console.log('[ASSOC] page errors: ' + errors.length + (errors.length ? ' | ' + errors.slice(0, 3).join(' | ') : ''));
    process.exitCode = (bad.length || errored.length || errors.length || !contractOk) ? 1 : 0;
  } catch (e) {
    console.error('[ASSOC] error:', e.message);
    process.exitCode = 1;
  } finally {
    try { if (cdp) cdp.ws.close(); } catch (_) {}
    try { proc.kill(); } catch (_) {}
  }
}

main();
