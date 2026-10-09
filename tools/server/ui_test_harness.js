#!/usr/bin/env node
// ui_test_harness.js — Dependency-free CDP UI/function smoke-test for NFT-Seduction dev server.
// Uses Node 21+ global WebSocket. Spawns a headless Chrome, attaches via the DevTools Protocol,
// loads the app, captures console/page errors, and checks key UI elements + API endpoints.
//
// Usage:  node tools/server/ui_test_harness.js [--url http://localhost:8090/] [--cdp-port 9333]
// Exit:   0 = all checks passed (or skipped because no browser), 1 = failures.

const http = require('http');
const { spawn } = require('child_process');
const fs = require('fs');
const os = require('os');
const path = require('path');

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

function waitFor(url, timeoutMs) {
  const start = Date.now();
  return new Promise((resolve, reject) => {
    const tick = async () => {
      try {
        const r = await httpGet(url);
        if (r.status === 200) return resolve(JSON.parse(r.body));
      } catch (_) {}
      if (Date.now() - start > timeoutMs) return reject(new Error('timeout waiting for ' + url));
      setTimeout(tick, 300);
    };
    tick();
  });
}

// Minimal CDP client over Node's global WebSocket.
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
      } else if (msg.method && this.handlers[msg.method]) {
        this.handlers[msg.method](msg.params);
      }
    };
  }
  open() { return new Promise((res, rej) => { this.ws.onopen = res; this.ws.onerror = (e) => rej(e); }); }
  on(method, cb) { this.handlers[method] = cb; }
  send(method, params = {}) {
    const id = ++this.id;
    return new Promise((res, rej) => {
      this.pending.set(id, { resolve: res, reject: rej });
      this.ws.send(JSON.stringify({ id, method, params }));
    });
  }
}

// In-page checks: UI elements + live API calls (run in page origin so cookies/headers match).
function pageChecks() {
  return (async () => {
    const results = [];
    const check = (name, pass, detail) => results.push({ name, pass: !!pass, detail: String(detail == null ? '' : detail).slice(0, 220) });
    try {
      const actionBtns = document.querySelectorAll('.action-dropdown-list button, .action-bar button, [data-action]').length;
      check('ui.action_buttons_present', actionBtns > 0, actionBtns);
      const appShell = !!document.querySelector('.app-shell, #app, [class*="app-shell"]');
      check('ui.app_shell_present', appShell, appShell);
      const overlays = document.querySelectorAll('.overlay, .vbt-overlay').length;
      check('ui.overlay_containers_present', overlays >= 0, overlays);
      check('ui.GetGameState_defined', typeof window.GetGameState !== 'undefined', typeof window.GetGameState);
      check('ui.openSeasonalEvents_defined', typeof window.openSeasonalEvents === 'function', typeof window.openSeasonalEvents);
      check('ui.openCriminality_defined', typeof window.openCriminality === 'function', typeof window.openCriminality);
      check('ui.openMatchArena_defined', typeof window.openMatchArena === 'function', typeof window.openMatchArena);
    } catch (e) { check('ui.dom_checks', false, e.message); }
    const api = ['/api/faucet/status', '/api/regions', '/api/match/active', '/api/entity/market/list', '/api/leaderboard'];
    for (const p of api) {
      try {
        const r = await fetch(p);
        let keys = 0;
        try { const j = await r.json(); keys = j && typeof j === 'object' ? Object.keys(j).length : 0; } catch (_) {}
        check('api' + p.replace(/\//g, '_'), r.ok, 'status=' + r.status + ' keys=' + keys);
      } catch (e) { check('api' + p.replace(/\//g, '_'), false, e.message); }
    }
    return JSON.stringify(results);
  })();
}

async function main() {
  const args = process.argv.slice(2);
  const getArg = (n, d) => { const i = args.indexOf(n); return i >= 0 ? args[i + 1] : d; };
  const APP_URL = getArg('--url', 'http://localhost:8090/');
  const CDP_PORT = parseInt(getArg('--cdp-port', '9333'), 10);

  const chrome = resolveChrome();
  if (!chrome) {
    console.log('[HARNESS] SKIP: no Chrome/Edge found. Set CHROME_BIN or install a browser.');
    process.exit(0);
  }

  const userData = fs.mkdtempSync(path.join(os.tmpdir(), 'vbt-cdp-'));
  const proc = spawn(chrome, [
    '--headless=new', '--disable-gpu', '--no-first-run', '--no-default-browser-check',
    '--remote-debugging-port=' + CDP_PORT, '--remote-allow-origins=*',
    '--user-data-dir=' + userData, 'about:blank',
  ], { stdio: 'ignore', detached: false });

  const cleanup = () => { try { proc.kill('SIGKILL'); } catch (_) {} try { fs.rmSync(userData, { recursive: true, force: true }); } catch (_) {} };
  process.on('exit', cleanup);

  let cdp;
  try {
    const ver = await waitFor('http://127.0.0.1:' + CDP_PORT + '/json/version', 15000);
    let targetUrl = ver.webSocketDebuggerUrl;
    // Attach to a page target instead of the browser-level socket so Runtime/Page/Log work.
    try {
      const list = await waitFor('http://127.0.0.1:' + CDP_PORT + '/json/list', 5000);
      const pageTarget = Array.isArray(list) ? list.find((t) => t.type === 'page') : null;
      if (pageTarget && pageTarget.webSocketDebuggerUrl) targetUrl = pageTarget.webSocketDebuggerUrl;
    } catch (_) {}
    cdp = new CDP(targetUrl);
    await cdp.open();

    const consoleErrors = [];
    cdp.on('Runtime.consoleAPICalled', (p) => { if (p.type === 'error') consoleErrors.push('console.error: ' + (p.args || []).map((a) => a.value).join(' ')); });
    cdp.on('Runtime.exceptionThrown', (p) => { const d = p.exceptionDetails || {}; consoleErrors.push('exception: ' + (d.text || '?') + ' @ ' + (d.url || '?') + ':' + (d.lineNumber || '?') + (d.exception ? ' :: ' + d.exception.description : '')); });
    cdp.on('Log.entryAdded', (p) => { if (p.entry && p.entry.level === 'error') consoleErrors.push('log.error: ' + p.entry.text); });

    await cdp.send('Runtime.enable');
    await cdp.send('Page.enable');
    await cdp.send('Log.enable');
    await cdp.send('Network.enable');

    const loaded = new Promise((resolve) => { cdp.on('Page.loadEventFired', () => resolve()); setTimeout(resolve, 12000); });
    await cdp.send('Page.navigate', { url: APP_URL });
    await loaded;
    await new Promise((r) => setTimeout(r, 3500)); // let WASM + dynamic modules initialize

    const evalRes = await cdp.send('Runtime.evaluate', { expression: '(' + pageChecks.toString() + ')()', returnByValue: true, awaitPromise: true });
    let results = [];
    if (evalRes && evalRes.result && evalRes.result.value) {
      try { results = JSON.parse(evalRes.result.value); } catch (_) { consoleErrors.push('eval parse error: ' + evalRes.result.value); }
    } else if (evalRes && evalRes.exceptionDetails) {
      consoleErrors.push('eval exception: ' + (evalRes.exceptionDetails.text || ''));
    }

    // Merge console/page errors into the report as hard failures.
    for (const e of consoleErrors) results.push({ name: 'console.' + results.length, pass: false, detail: e.slice(0, 220) });

    let pass = 0, fail = 0;
    console.log('\n[HARNESS] UI/function smoke-test — ' + APP_URL);
    console.log('------------------------------------------------------------');
    for (const r of results) {
      if (r.pass) { pass++; console.log('  PASS  ' + r.name + '  (' + r.detail + ')'); }
      else { fail++; console.log('  FAIL  ' + r.name + '  (' + r.detail + ')'); }
    }
    console.log('------------------------------------------------------------');
    console.log('[HARNESS] ' + pass + ' passed, ' + fail + ' failed');
    cleanup();
    process.exit(fail === 0 ? 0 : 1);
  } catch (e) {
    console.log('[HARNESS] ERROR: ' + e.message);
    cleanup();
    process.exit(1);
  }
}

main();

