#!/usr/bin/env node
// probe_firstrun.cjs — temporary headless CDP probe: first-run quick-start flow.
const http = require('http'), { spawn } = require('child_process'), fs = require('fs'), os = require('os'), path = require('path');
const CDP_PORT = 9443, APP_URL = 'http://localhost:8090/', sleep = (ms) => new Promise((r) => setTimeout(r, ms));
function resolveChrome() {
  const cands = [process.env.CHROME_BIN, 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe', 'C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe', process.env.LOCALAPPDATA + '\\Google\\Chrome\\Application\\chrome.exe', 'C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe', 'C:\\Program Files\\Microsoft\\Edge\\Application\\msedge.exe'].filter(Boolean);
  for (const c of cands) { if (fs.existsSync(c)) return c; } return null;
}
function httpGet(url) { return new Promise((resv, rej) => { const q = http.get(url, (r) => { let b = ''; r.on('data', (d) => (b += d)); r.on('end', () => resv({ status: r.statusCode, body: b })); }); q.on('error', rej); q.setTimeout(2000, () => q.destroy(new Error('timeout'))); }); }
function waitFor(url, t) { const s = Date.now(); return new Promise((resv, rej) => { const tick = async () => { try { const r = await httpGet(url); if (r.status === 200) return resv(JSON.parse(r.body)); } catch (_) {} if (Date.now() - s > t) return rej(new Error('timeout ' + url)); setTimeout(tick, 300); }; tick(); }); }
class CDP {
  constructor(url) {
    this.ws = new WebSocket(url); this.id = 0; this.pending = new Map(); this.handlers = {};
    this.ws.onmessage = (ev) => { const m = JSON.parse(ev.data);
      if (m.id != null) { const p = this.pending.get(m.id); if (p) { this.pending.delete(m.id); m.error ? p.reject(new Error(JSON.stringify(m.error))) : p.resolve(m.result); } }
      else if (m.method && this.handlers[m.method]) this.handlers[m.method](m.params); };
  }
  open() { return new Promise((r, j) => { this.ws.onopen = r; this.ws.onerror = (e) => j(e); }); }
  on(m, cb) { this.handlers[m] = cb; }
  send(m, p = {}) { const id = ++this.id; return new Promise((r, j) => { this.pending.set(id, { resolve: r, reject: j }); this.ws.send(JSON.stringify({ id, method: m, params: p })); }); }
}
async function main() {
  const chrome = resolveChrome();
  if (!chrome) { console.log('PROBE_RESULT NO_CHROME'); process.exit(2); }
  const ud = fs.mkdtempSync(path.join(os.tmpdir(), 'vbt-fr-'));
  const proc = spawn(chrome, ['--headless=new', '--disable-gpu', '--no-first-run', '--no-default-browser-check', '--remote-debugging-port=' + CDP_PORT, '--remote-allow-origins=*', '--user-data-dir=' + ud, 'about:blank'], { stdio: 'ignore', detached: false });
  const cleanup = () => { try { proc.kill('SIGKILL'); } catch (_) {} try { fs.rmSync(ud, { recursive: true, force: true }); } catch (_) {} };
  process.on('exit', cleanup);
  try {
    const ver = await waitFor('http://127.0.0.1:' + CDP_PORT + '/json/version', 15000);
    let targetUrl = ver.webSocketDebuggerUrl;
    try { const list = await waitFor('http://127.0.0.1:' + CDP_PORT + '/json/list', 5000); const pt = Array.isArray(list) ? list.find((t) => t.type === 'page') : null; if (pt && pt.webSocketDebuggerUrl) targetUrl = pt.webSocketDebuggerUrl; } catch (_) {}
    const cdp = new CDP(targetUrl); await cdp.open();
    const consoleErrors = [];
    cdp.on('Runtime.consoleAPICalled', (p) => { if (p.type === 'error') consoleErrors.push('console.error: ' + (p.args || []).map((a) => a.value).join(' ')); });
    cdp.on('Runtime.exceptionThrown', (p) => { const d = p.exceptionDetails || {}; consoleErrors.push('exception: ' + (d.text || '?') + '@' + (d.url || '?') + ':' + (d.lineNumber || '?') + ' :: ' + ((d.exception || {}).description || '')); });
    await cdp.send('Runtime.enable'); await cdp.send('Page.enable');
    const loaded = new Promise((res) => { cdp.on('Page.loadEventFired', () => res()); setTimeout(res, 12000); });
    await cdp.send('Page.navigate', { url: APP_URL }); await loaded; await sleep(4200);
    const getV = async (expr) => { const r = await cdp.send('Runtime.evaluate', { expression: expr, returnByValue: true }); return (r.result && r.result.value); };
    const v1 = await getV(`(() => { const el = document.getElementById('first-run-overlay'); return { overlay_exists: !!el, display: el ? getComputedStyle(el).display : null, position: el ? getComputedStyle(el).position : null, step0_active: !!el && !!el.querySelector('.fr-step.fr-active'), step_count: el ? el.querySelectorAll('.fr-step').length : 0 }; })()`);
    await getV(`(() => { const b = document.querySelector('#first-run-overlay .fr-cta[data-step="0"]'); if (b) b.click(); return !!b; })()`); await sleep(700);
    const v2 = await getV(`(() => { const sel = document.getElementById('wallet-selector-overlay'); const fr = document.getElementById('first-run-overlay'); return { selector_shown: !!sel && !sel.classList.contains('hidden'), quickstart_hidden: !fr || getComputedStyle(fr).display === 'none' }; })()`);
    await getV(`window.getActiveWallet = () => '0xTESTWALLET'; true;`); await sleep(2600);
    const v3 = await getV(`(() => { const fr = document.getElementById('first-run-overlay'); return { stepIdx: window.__firstRunState().stepIdx, step0_done: !!document.querySelector('#first-run-overlay .fr-step.fr-done'), claim_cta: !!document.querySelector('#first-run-overlay .fr-cta[data-step="1"]'), visible: fr ? getComputedStyle(fr).display : 'absent' }; })()`);
    await getV(`(() => { const b = document.querySelector('#first-run-overlay .fr-cta[data-step="1"]'); if (b) b.click(); return !!b; })()`); await sleep(700);
    const v4 = await getV(`(() => { const fa = document.getElementById('faucet-dashboard'); const fr = document.getElementById('first-run-overlay'); return { faucet_open: !!fa, quickstart_hidden: !fr || getComputedStyle(fr).display === 'none' }; })()`);
    await sleep(2000);
    const v5 = await getV(`(() => ({ stepIdx: window.__firstRunState().stepIdx }))()`);
    await getV(`(() => { const fa = document.getElementById('faucet-dashboard'); if (fa) fa.remove(); return !!fa; })()`); await sleep(2600);
    const v6 = await getV(`(() => { const fr = document.getElementById('first-run-overlay'); return { visible: fr ? getComputedStyle(fr).display : 'absent', hub_cta: !!document.querySelector('#first-run-overlay .fr-cta[data-step="2"]'), stepIdx: window.__firstRunState().stepIdx }; })()`);
    await getV(`(() => { const b = document.querySelector('#first-run-overlay .fr-cta[data-step="2"]'); if (b) b.click(); return !!b; })()`); await sleep(900);
    const v7 = await getV(`(() => { const el = document.getElementById('first-run-overlay'); return { overlay_hidden: !el || getComputedStyle(el).display === 'none', flag: (() => { try { return localStorage.getItem('vbt_first_run_seen'); } catch (_) { return 'err'; } })() }; })()`);
    console.log('AUTO_SHOW ' + JSON.stringify(v1));
    console.log('AFTER_CLICK_CONNECT ' + JSON.stringify(v2));
    console.log('AFTER_WALLET_DONE ' + JSON.stringify(v3));
    console.log('AFTER_CLICK_CLAIM ' + JSON.stringify(v4));
    console.log('FAUCET_SYNCED ' + JSON.stringify(v5));
    console.log('AFTER_FAUCET_CLOSED ' + JSON.stringify(v6));
    console.log('AFTER_HUB ' + JSON.stringify(v7));
    if (consoleErrors.length) { console.log('CONSOLE_ERRORS ' + JSON.stringify(consoleErrors)); cleanup(); process.exit(1); }
    cleanup(); process.exit(0);
  } catch (e) { console.log('PROBE_ERROR ' + e.message); cleanup(); process.exit(1); }
}
main();