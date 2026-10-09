#!/usr/bin/env node
// cd_ws_probe.cjs — opens a REAL headless-Chrome WebSocket to ws://localhost:8090/ws
// and reports the messages received (the user's actual client). Decisive WS test.
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
    const req = http.get(url, (res) => { let b = ''; res.on('data', d => b += d); res.on('end', () => resolve({ status: res.statusCode, body: b })); });
    req.on('error', reject); req.setTimeout(2000, () => req.destroy(new Error('timeout')));
  });
}
function waitFor(url, t) {
  const s = Date.now();
  return new Promise((resolve, reject) => { const tick = async () => { try { const r = await httpGet(url); if (r.status === 200) return resolve(JSON.parse(r.body)); } catch (_) {} if (Date.now() - s > t) return reject(new Error('timeout ' + url)); setTimeout(tick, 300); }; tick(); });
}
class CDP {
  constructor(url) { this.ws = new WebSocket(url); this.id = 0; this.pending = new Map(); this.handlers = {}; this.ws.onmessage = (ev) => { const m = JSON.parse(ev.data); if (m.id != null) { const p = this.pending.get(m.id); if (p) { this.pending.delete(m.id); m.error ? p.reject(new Error(JSON.stringify(m.error))) : p.resolve(m.result); } } else if (m.method && this.handlers[m.method]) this.handlers[m.method](m.params); }; }
  open() { return new Promise((res, rej) => { this.ws.onopen = res; this.ws.onerror = (e) => rej(e); }); }
  on(method, cb) { this.handlers[method] = cb; }
  send(method, params = {}) { const id = ++this.id; return new Promise((res, rej) => { this.pending.set(id, { resolve: res, reject: rej }); this.ws.send(JSON.stringify({ id, method, params })); }); }
}

async function main() {
  const CDP_PORT = 9334;
  const chrome = resolveChrome();
  if (!chrome) { console.log('NO CHROME'); process.exit(0); }
  const userData = fs.mkdtempSync(path.join(os.tmpdir(), 'vbt-ws-'));
  const proc = spawn(chrome, ['--headless=new', '--disable-gpu', '--no-first-run', '--no-default-browser-check', '--remote-debugging-port=' + CDP_PORT, '--remote-allow-origins=*', '--user-data-dir=' + userData, 'about:blank'], { stdio: 'ignore' });
  const cleanup = () => { try { proc.kill('SIGKILL'); } catch (_) {} try { fs.rmSync(userData, { recursive: true, force: true }); } catch (_) {} };
  process.on('exit', cleanup);
  try {
    const ver = await waitFor('http://127.0.0.1:' + CDP_PORT + '/json/version', 15000);
    let targetUrl = ver.webSocketDebuggerUrl;
    try { const list = await waitFor('http://127.0.0.1:' + CDP_PORT + '/json/list', 5000); const pt = Array.isArray(list) ? list.find(t => t.type === 'page') : null; if (pt && pt.webSocketDebuggerUrl) targetUrl = pt.webSocketDebuggerUrl; } catch (_) {}
    const cdp = new CDP(targetUrl);
    await cdp.open();
    await cdp.send('Runtime.enable');
    const expr = `(async () => {
      const msgs = [];
      try {
        const ws = new WebSocket('ws://localhost:8090/ws');
        ws.onopen = () => msgs.push('OPEN');
        ws.onmessage = (e) => { try { const m = JSON.parse(e.data); msgs.push('MSG type=' + m.type + ' to_id=' + (m.to_id || '') + ' from_id=' + (m.from_id || '')); } catch (err) { msgs.push('RAW ' + String(e.data).slice(0,80)); } };
        ws.onerror = () => msgs.push('WSERR');
        ws.onclose = () => msgs.push('CLOSE');
      } catch (e) { msgs.push('THROW ' + e.message); }
      await new Promise(r => setTimeout(r, 5000));
      return msgs;
    })()`;
    const evalRes = await cdp.send('Runtime.evaluate', { expression: expr, returnByValue: true, awaitPromise: true });
    console.log('WS_MSGS=' + JSON.stringify(evalRes.result ? evalRes.result.value : evalRes));
    cleanup();
    process.exit(0);
  } catch (e) { console.log('ERR ' + e.message); cleanup(); process.exit(1); }
}
main();
