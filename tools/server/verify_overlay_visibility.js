#!/usr/bin/env node
// verify_overlay_visibility.js — audit the CONTAINED-OVERLAY VISIBILITY contract.
//
// WHY THIS EXISTS
//   `hideAllOverlays()` (ui.js) adds the `.hidden` class to EVERY element with the
//   `.overlay` class. `.hidden` is defined as `display: none !important`
//   (_spacing.scss / _faucet_dashboard.scss), which BEATS any plain inline
//   `style.display = 'flex'`. So an overlay root that carries both `.overlay` and
//   `.vbt-overlay` and is revealed only by setting inline display will compute as
//   `display:none` — silently invisible, while its DOM still renders.
//
//   This probe opens each standalone overlay and reports the COMPUTED display /
//   visibility / opacity of the element that opener just revealed, so "the opener
//   ran" and "the user can see it" are separately measurable.
//
// Usage: node tools/server/verify_overlay_visibility.js [--url http://localhost:8090/]

const http = require('http');
const { spawn } = require('child_process');
const fs = require('fs');
const os = require('os');
const path = require('path');

const OPENERS = [
  'openPortfolio', 'openPersistentIdentity', 'openCareers', 'openAchievements',
  'openStatOverlay', 'openLeaderboardRegion', 'openRivalryViewer', 'openMatchArena',
  'openCreatorStore', 'openWorldEvents', 'openFaithSystem', 'openFaithChurch',
  'openGovernancePanel', 'openOrphanCleaner', 'openPetBattleArena', 'openLifeAssets',
  'openUnderworld', 'openCriminality', 'openIndustrialLoop', 'openGamingOS',
  'openLaunchpad', 'openAdvertising', 'openBondedBranding', 'openAdminConsole',
];

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
  const cdpPort = 9446;

  const chrome = resolveChrome();
  if (!chrome) { console.log('[OVERLAY] No browser found — skipped.'); process.exit(0); }

  const profile = fs.mkdtempSync(path.join(os.tmpdir(), 'vbt-overlay-'));
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
    await new Promise((r) => setTimeout(r, 9000));

    const expr = `(async () => {
      const openers = ${JSON.stringify(OPENERS)};
      const rows = [];
      const revealed = () => Array.from(document.querySelectorAll('[class*="overlay"], .pp-hub'))
        .filter(el => el.style && el.style.display && el.style.display !== 'none');
      for (const name of openers) {
        const fn = window[name];
        if (typeof fn !== 'function') { rows.push({ name, status: 'not-defined' }); continue; }
        const before = new Set(revealed());
        try { fn(); } catch (e) { rows.push({ name, status: 'threw: ' + String(e.message || e).slice(0, 70) }); continue; }
        await new Promise(r => setTimeout(r, 500));
        const fresh = revealed().filter(el => !before.has(el));
        if (!fresh.length) { rows.push({ name, status: 'revealed-nothing' }); continue; }
        const el = fresh[0];
        const cs = getComputedStyle(el);
        let chain = '';
        let par = el.parentElement;
        let guard = 0;
        while (par && guard < 5) {
          const pcs = getComputedStyle(par);
          chain += (par.id || par.tagName.toLowerCase()) + '[' + pcs.display + ',' + pcs.visibility + '] ';
          par = par.parentElement;
          guard++;
        }
        rows.push({
          name, status: 'revealed',
          id: el.id || String(el.className).split(' ')[0],
          inline: el.style.display,
          computed: cs.display,
          vis: cs.visibility,
          op: cs.opacity,
          hasHiddenClass: el.classList.contains('hidden'),
          chain: chain
        });
      }
      return { rows };
    })()`;

    const r = await cdp.send('Runtime.evaluate', { expression: expr, returnByValue: true, awaitPromise: true });
    const rows = r.result.value.rows;
    let bad = 0;
    console.log('\n[OVERLAY] Contained-overlay visibility audit — ' + target);
    console.log('-'.repeat(80));
    for (const row of rows) {
      const visible = row.computed && row.computed !== 'none' && row.vis !== 'hidden' && String(row.op) !== '0';
      if (row.status === 'revealed' && visible) {
        console.log('  OK     ' + row.name.padEnd(26) + String(row.id || '').padEnd(24) + 'computed=' + row.computed);
      } else if (row.status === 'revealed') {
        bad++;
        console.log('  BROKEN ' + row.name.padEnd(26) + String(row.id || '').padEnd(24)
          + 'inline=' + row.inline + ' computed=' + row.computed + ' vis=' + row.vis + ' hiddenClass=' + row.hasHiddenClass);
        if (row.chain) console.log('         ancestors: ' + row.chain);
      } else if (row.status === 'revealed-nothing') {
        console.log('  NOTE   ' + row.name.padEnd(26) + 'opener ran but revealed nothing (may need state)');
      } else {
        console.log('  NOTE   ' + row.name.padEnd(26) + row.status);
      }
    }
    console.log('-'.repeat(80));
    console.log('[OVERLAY] ' + (rows.length - bad) + ' of ' + rows.length
      + ' openers produce a VISIBLE overlay; ' + bad + ' invisible');
    if (errors.length) console.log('[OVERLAY] page errors: ' + errors.slice(0, 3).join(' | '));
    process.exitCode = bad ? 1 : 0;
  } catch (e) {
    console.error('[OVERLAY] error:', e.message);
    process.exitCode = 1;
  } finally {
    try { if (cdp) cdp.ws.close(); } catch (_) {}
    try { proc.kill(); } catch (_) {}
  }
}

main();

